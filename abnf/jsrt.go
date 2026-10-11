package abnf

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"math/bits"
	"math/rand/v2"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/llir/llvm/ir"
)

// ----------------------------------------------------------------------------
// MetaJS handle runtime (jsrt)
//
// The IR that metajs-to-llvm-ir.abnf emits works exclusively with i64 handles:
// every dynamic value of the compiled MetaJS program is an index into the
// value table of this runtime, and every operation with real JS semantics is
// a call to one of the js_* external functions below. The IR itself only
// moves handles around and branches on js_truthy results, so the existing
// integer IR interpreter (llvmmap.go) can execute it unchanged.
//
// The runtime also contains the bridge that goja provides in the interpreted
// world: arbitrary Go values can live behind a handle. Property access and
// method calls on them are resolved with reflection (struct fields, methods,
// maps, slices, variadic functions), so the compiled scripts can drive e.g.
// the llir/llvm builder objects or the abnf.* rule builders directly.
//
// Handle 0..3 are the singletons undefined, null, false and true. A handle with
// the top bit set carries an integer VALUE in its payload and needs no table
// entry (see numHandle); every other handle indexes the table. Strings are
// interned by value, our own pointer kinds by identity, so the table does not
// grow with every operation.

const (
	jsHUndefined = uint64(0)
	jsHNull      = uint64(1)
	jsHFalse     = uint64(2)
	jsHTrue      = uint64(3)
)

// Table chunking: 64 Ki entries (1 MB) per chunk.
const (
	tblChunkBits = 16
	tblChunkSize = 1 << tblChunkBits
	tblChunkMask = tblChunkSize - 1
)

// Immediate integers. The runtime is a value-per-table-entry design with no
// reclamation, so integers that fit the payload are encoded in the handle
// itself: nothing is allocated, nothing is interned, and the value is recovered
// by shifting. Handle identity for numbers is not observable (=== compares the
// unwrapped values, and the dict index and the trace hooks do the same), so
// this is invisible to a program.
const (
	numTag  = uint64(1) << 63 // Set = the handle IS the number.
	numBits = 62              // Payload width, signed: [-2^61, 2^61).
	numMask = uint64(1)<<numBits - 1
	numSign = uint64(1) << (numBits - 1)
	numLim  = float64(int64(1) << (numBits - 1))
)

// numHandle encodes f as an immediate handle, or reports that it cannot.
// Non-integers, values outside the payload range, NaN and negative zero (whose
// handle must stay distinct from +0's, so 1/-0 keeps its sign) fall back to the
// table.
func numHandle(f float64) (uint64, bool) {
	if !(f >= -numLim && f < numLim) { // Also false for NaN.
		return 0, false
	}
	i := int64(f)
	if float64(i) != f || (i == 0 && math.Signbit(f)) {
		return 0, false
	}
	return numTag | uint64(i)&numMask, true
}

// numValue decodes an immediate handle (sign-extending the payload).
func numValue(h uint64) float64 {
	x := h & numMask
	if x&numSign != 0 {
		x |= ^numMask
	}
	return float64(int64(x))
}

// jsUndef and jsNull are the singleton marker values inside the table.
type jsUndefT struct{}
type jsNullT struct{}

var jsUndef = jsUndefT{}
var jsNull = jsNullT{}

// jsAnyT is the value of the host global 'anytype'. Declaring a variable with
// it (var v = anytype) starts the variable as undefined WITHOUT pinning: it may
// hold values of every type class for its whole (re)declared life. Only valid
// as a declaration initializer; assigning it to an existing variable fails.
// (Goja binds 'anytype' to plain undefined instead - it cannot pin types
// anyway, and this way the variable starts as undefined under both engines.)
type jsAnyT struct{}

var jsAnytype = jsAnyT{}

func isAnytype(v interface{}) bool {
	_, ok := v.(jsAnyT)
	return ok
}

// jsHandleID lets a value the runtime itself creates carry its own handle
// instead of being looked up in the identity intern map. Object identity (the
// same pointer must always wrap to the same handle) is what the map provided;
// a field on the object provides it too, in O(1) and without growing a map that
// is never emptied - and a compile creates hundreds of thousands of these.
// hrt records WHICH runtime the handle belongs to, because a process runs
// several (one per stage, plus the compiled program's own): for any other
// runtime the value falls back to that runtime's intern map.
type jsHandleID struct {
	h   uint64
	hrt *jsrt
}

// jsObject is a plain MetaJS object. The key order is kept for deterministic behavior.
type jsObject struct {
	id    jsHandleID
	props map[string]interface{}
	keys  []string
}

func newJSObject() *jsObject {
	return &jsObject{props: map[string]interface{}{}}
}

func (o *jsObject) set(key string, v interface{}) {
	if _, ok := o.props[key]; !ok {
		o.keys = append(o.keys, key)
	}
	o.props[key] = v
}

// jsArray is a MetaJS array.
type jsArray struct {
	id    jsHandleID
	elems []interface{}

	// When this array is the key array of a dict, dictIdx maps a key to its
	// position in elems so a lookup does not scan (see dictFind). dictIdxLen is
	// the element count it was built for and dictIdxAll says whether every key
	// made it in; a nil map means "not built / no longer trustworthy".
	dictIdx    map[interface{}]int
	dictIdxLen int
	dictIdxAll bool

	// pinned marks an argument array a callee can keep past the call, so the
	// call site must not reclaim it (see machine.recycle and jsrt.pinArray).
	pinned bool
}

// jsClosure is a compiled MetaJS function: an IR function plus the captured
// scope. It also remembers the machine of its module, so closures from
// different modules (e.g. a helper library and a tag script) can call each
// other through one shared runtime.
type jsClosure struct {
	id   jsHandleID
	fn   *ir.Func
	env  uint64 // Scope handle of the creation site.
	ma   *machine
	meta *jsvFnInfo // js/ts `name` and `length`, set by js_jsfnmeta; nil = ""/0
}

// jsThrown carries a user-thrown value (any handle) as a Go panic. js_throw raises
// it; js_try recovers it (a Go panic unwinds the native machine.call frames until
// the recover installed by the nearest enclosing js_try). Runtime errors (rt.fail)
// panic a plain string instead, so js_try re-panics anything that is not a jsThrown.
type jsThrown struct {
	value interface{}
}

// jsCtl is a control-flow signal for a return/break/continue that LEAVES a try body.
// A try/catch/finally body compiles to its own IR closure, so a non-local jump inside
// it cannot branch to the enclosing function/loop directly; instead the closure
// returns a jsCtl, which js_try passes through and the try's compiled dispatch
// (compile-core excDispatch) re-issues as a real ret/br in the enclosing frame.
type jsCtl struct {
	kind  byte        // 1 = return, 2 = break, 3 = continue
	value interface{} // the returned value (kind 1); ignored otherwise
}

// jsScope is one link of a scope chain. Variables can hold undefined, so
// existence is a name being PRESENT, not the value it holds.
//
// The names live in parallel slices, not in a map: a scope holds a handful of
// names (a call frame, a block), and for those a linear scan over one
// contiguous slice beats hashing every read - a map also had to be ALLOCATED
// per scope and per declaration, which was 23 % of everything the frozen engine
// allocated. The slices stay the storage in every case; a scope that does grow
// (the root scope with its host bindings, a script's top level) additionally
// builds an INDEX over them past jsScopeLinear entries, so a big scope is still
// O(1). Sorting the names for a binary search would be the one option that does
// not compose with this: it would have to reorder all three slices on every
// declaration, and it measures slower than the scan below the threshold and
// slower than the index above it.
type jsScope struct {
	id     jsHandleID
	names  []string
	vals   []interface{}
	tcs    []string // Pinned type class per name, "" = unpinned. See typedDecl.
	index  map[string]int
	parent *jsScope
	// Python scoping (js_pyfnscope / js_pyset_var; nothing else sets these).
	// pyFn marks a scope that is a BINDING BOUNDARY - a def, a lambda, a class
	// body or the module top level - so an assignment inside it binds locally
	// instead of reaching an enclosing function's binding of the same name.
	// pyDecl records the names a `global` ('g') or `nonlocal` ('n') statement
	// declared, which is what lets an assignment cross the boundary on purpose.
	pyFn   bool
	pyDecl map[string]byte
}

// jsScopeLinear is where a scope switches from scanning to an index.
const jsScopeLinear = 8

func (s *jsScope) find(name string) int {
	if s.index != nil {
		if i, ok := s.index[name]; ok {
			return i
		}
		return -1
	}
	for i, n := range s.names {
		if n == name {
			return i
		}
	}
	return -1
}

func (s *jsScope) get(name string) (interface{}, bool) {
	if i := s.find(name); i >= 0 {
		return s.vals[i], true
	}
	return nil, false
}

func (s *jsScope) has(name string) bool { return s.find(name) >= 0 }

// put declares a name or overwrites it.
func (s *jsScope) put(name string, v interface{}) {
	if i := s.find(name); i >= 0 {
		s.vals[i] = v
		return
	}
	s.names = append(s.names, name)
	s.vals = append(s.vals, v)
	s.tcs = append(s.tcs, "")
	if s.index != nil {
		s.index[name] = len(s.names) - 1
	} else if len(s.names) > jsScopeLinear {
		s.index = make(map[string]int, 2*len(s.names))
		for i, n := range s.names {
			s.index[n] = i
		}
	}
}

// typeClassOf reports the pinned type class of a name ("" when unpinned).
func (s *jsScope) typeClassOf(name string) string {
	if i := s.find(name); i >= 0 {
		return s.tcs[i]
	}
	return ""
}

// pinType pins (or with "" unpins) the type class of a name.
func (s *jsScope) pinType(name, tc string) {
	if i := s.find(name); i >= 0 {
		s.tcs[i] = tc
	}
}

// hostFunc is a builtin implemented directly on handles (no reflection).
type hostFunc struct {
	name string
	fn   func(rt *jsrt, this uint64, args []interface{}) interface{}
	// pybm is set only on the closures python's pyBoundBuiltin / pyBindMethod
	// mint, and is what makes them introspectable - see pyBoundInfo. nil for
	// every other host function in every other language, so this costs one
	// pointer per host function and nothing else.
	pybm *pyBoundInfo
	meta *jsvFnInfo // as on jsClosure
}

// pyBoundInfo is what a python BOUND METHOD knows about itself. Neither of the
// two name tables (rt.pyFuncNames for a `def`, rt.pyBuiltinRender for a builtin)
// can hold it: both are keyed by the closure VALUE and a bound method is a fresh
// closure minted at the attribute read. Keep in step with the pyBM* table in
// languages/lib/python-rt.metajs and pyBMSet/pyBMFind in
// languages/python-interpreter.abnf.
//
//	name  the method name           -> __name__
//	self  the receiver              -> __self__
//	fn    the underlying function   -> __func__, nil for a builtin method
//	typ   the receiver's type name  -> the __qualname__ prefix and the repr
type pyBoundInfo struct {
	name string
	self interface{}
	fn   interface{}
	typ  string
	// kind distinguishes the three callables this record describes -
	// "" a bound method, "d" an unbound method descriptor (list.count),
	// "m" a module-level builtin function (math.sqrt). They print, name and
	// introspect differently; see abnf/jsrtpydesc.go, which is where the three
	// are minted, and the matching fifth pyBMTab slot in
	// languages/lib/python-rt.metajs and languages/python-interpreter.abnf.
	kind string
}

// pyBoundOf answers the bound-method record of v, or nil.
func pyBoundOf(v interface{}) *pyBoundInfo {
	if h, ok := v.(*hostFunc); ok {
		return h.pybm
	}
	return nil
}

// boundMethod is a builtin method of a string or array value, e.g. "abc".charCodeAt.
type boundMethod struct {
	recv interface{}
	name string
}

// jsrt is one MetaJS runtime: a shared value table, a root scope with the host
// bindings, and any number of attached IR modules (each on its own machine).
type jsrt struct {
	// The value table is chunked, not one growing slice: it reaches millions of
	// entries on a loop-heavy program, and doubling a contiguous slice of that
	// size copies and re-maps tens of megabytes per growth step (it showed up as
	// runtime.madvise/mmap). Chunks are allocated once and never moved.
	chunks [][]interface{}
	count  uint64 // Handles handed out so far = the next never-used slot.

	// Released slots, oldest first from freeHead on (see releaseHandle).
	freed    []uint64
	freeHead int

	strIntern map[string]uint64
	numIntern map[uint64]uint64      // Keyed by the float bits, so -0 and +0 stay distinct handles.
	objIntern map[interface{}]uint64 // Identity interning for pointer-like values.

	root *jsScope

	// phpLsb is PHP's late-static-binding class: the class a C::m() call named,
	// so `static::` inside an inherited static method resolves to it. Only
	// js_phscall writes it (saved and restored around the call); see jsrtphp.go.
	phpLsb *jsObject

	// retSlot holds the completion value of the running program: js_setret is
	// emitted for every expression statement, so after a run the slot holds the
	// value of the last executed expression statement - the same thing that a
	// goja Run() returns. The frozen engine saves and restores it around
	// nested runs.
	retSlot interface{}

	// No trail of recent member lookups is kept for the call-of-a-non-function
	// diagnostic: the message must match the floor's die2 (languages/lib/runtime.c)
	// byte for byte, and the floor has no allocation-free place to keep one. The
	// site is an abort, so only tests/probe.sh compares the two engines' stderr.

	// The three RECOVERABLE-ERROR HOOKS, the twins of RT_MISS_VAR / RT_MISS_MEM /
	// RT_MISS_SET in languages/lib/runtime.c. scopeGet, getMember and setMember
	// are shared by all languages; a language whose semantics say "this is a
	// catchable exception" (js, typescript, python) registers a closure here in its
	// boot block and one that means "the program is broken" (kotlin, swift, java,
	// metajs - and the three SHOULD-ABORT matrix rows) registers nothing and
	// aborts. The closure ANSWERS the value to throw; the slot is cleared while it
	// runs, so a handler that itself reads a name nowhere bound falls back to the
	// original abort instead of recursing.
	missVar interface{} // f(name)      -> the value to throw, or nil = abort
	missMem interface{} // f(key, obj)  -> the value to throw, or nil = abort
	missSet interface{} // f(key, obj)  -> the value to throw, or nil = abort
	missFn  interface{} // f(callee)    -> the value to throw, or nil = abort (lua)

	// The FOURTH of the family, and the odd one out: it is a plain VALUE, not a
	// closure. memberCall cannot build node's message for a call of a non-function
	// - `xs[1] is not a function` names the SOURCE EXPRESSION, and `xs[1]` and
	// `xs[i]` reach it as the same key - so the runtime's half of the job is only
	// to say "nothing claimed this call" and let the CALL SITE, which knows the
	// text at compile time, raise. missCall holds the sentinel to answer with; a
	// language that never registers one keeps the abort.
	missCall interface{}

	callBuf [2]uint64 // The (env, args) pair handed to a compiled callee; see callInner.

	// strCache remembers what indexing and interning had to derive from the
	// last few long strings (see strEntry); strScratch serves the short ones.
	strCache   [strCacheSlots]strInfo
	strScratch strInfo

	// The -trace hook (see trace.go). Only the program runtime of runJSModule
	// is traced; the frozen engine's tag-script runtime stays silent.
	traced     bool
	traceDepth int
	traceNames map[*jsClosure]string // Under which name a closure was stored.
	curPos     int                   // Source offset of the executing statement (js_srcpos), -1 = unknown.

	// thisStack is the dynamic `this` of the compiled closures currently on the
	// call stack: callInner pushes the receiver a call was made with and pops it
	// again, and js_this reads the top. A compiled function that mentions `this`
	// binds it from there at entry; one that does not pays nothing. js_try
	// truncates the stack back to its own depth when a throw unwinds past a call.
	thisStack []interface{}

	// newTargetStack is the `new.target` of the compiled closures on the call
	// stack, filled the same way as thisStack: js_call_new arms pendingNewTarget
	// for the ONE call that follows, callInner consumes it, and js_newtarget reads
	// the top. An ordinary call pushes nil, so new.target is undefined in it.
	newTargetStack   []interface{}
	pendingNewTarget interface{}

	// rubyClasses caches the BUILTIN class objects of ruby-to-llvm-ir.abnf
	// (Integer, String, Symbol, Array, ...), so that `x.class == Integer` and
	// `case v when Integer` compare the same object every time. Only js_rclass
	// fills it; every other grammar leaves it nil.
	rubyClasses map[string]*jsObject

	// rubyNmArr / rubyNmRecv: the Array a Range, Hash or Enumerator is delegating
	// to right now, and that delegator (rubyDelegArr), so a NoMethodError names it.
	rubyNmArr  *jsArray
	rubyNmRecv interface{}

	// rubyGlobals holds Ruby's $globals, which live outside every scope: a write
	// inside a method is visible everywhere, and a read of a never-assigned one
	// answers nil. Only js_rgset / js_rgget (ruby-to-llvm-ir.abnf) touch it.
	rubyGlobals map[string]interface{}

	// ktRndDef is kotlin.random.Random.Default - the one generator every unseeded
	// `random()` draws from. Built lazily by ktRndDefault (abnf/jsrtkotlin.go);
	// deterministic on purpose, see the kotlin.random block there.
	ktRndDef *jsObject

	// rubyLambdas remembers which closures came from a lambda literal (->) or
	// Kernel#lambda, which is all that Proc#lambda? needs. Filled by js_rlambda.
	rubyLambdas map[interface{}]bool

	// rubyOids is Object#object_id's table: the values numbered so far, in order
	// (see rubyObjectId in jsrtruby.go).
	rubyOids []interface{}

	// rubyMethodVals remembers which callables `method(:x)` handed out, and the
	// owner and name each one carries. MRI's Method is a class of its own -
	// #class is Method, #to_s is `#<Method: Object#x>` and #arity counts optionals
	// like a lambda's.
	rubyMethodVals map[interface{}]rubyMethodVal

	// rubyArities remembers a closure's PARAMETER SHAPE, which is the only thing
	// Proc#arity can be computed from and which no function value carries. Filled
	// by js_rarity where the closure is built; see rubyArityOf for why the three
	// counts are stored rather than the answer.
	rubyArities map[interface{}]rubyArity

	// rubyCurExc is the exception the innermost js_try handed to a catch clause,
	// which is what a BARE `raise' inside a rescue re-raises (Ruby's $!). It is
	// only ever read by js_rraise.
	rubyCurExc interface{}

	// goPanics is Go's panic state, one entry per compiled Go frame currently
	// unwinding-aware (js_gotry pushes, js_gorepanic pops). An entry is nil while
	// the frame runs normally and holds the panic value once one was caught, so a
	// recover() in one of the frame's deferred functions can take it (js_gorecover
	// clears the top entry). Only the go-to-llvm-ir grammar emits these.
	goPanics []interface{}
	// goDeferAt is the goPanics index of each frame whose deferred functions are
	// running right now (js_rundefers pushes and pops it). recover() is only
	// meaningful when it is called BY such a deferred function, so js_gorecover
	// reads the frame named here rather than the innermost one - the deferred
	// closure is a compiled Go function and has pushed a goPanics entry of its own.
	goDeferAt []int

	// goQueue is the compiled Go program's goroutine run queue. Concurrency is
	// COOPERATIVE and deterministic (see the channel externs and the matching
	// goSpawn/goDrain in go-interpreter.abnf): `go f()` queues f, and a receive
	// that finds its channel empty runs the queue until one fills it.
	goQueue []interface{}

	// trackThis is set by attach when the module actually declares js_this or
	// js_newtarget, i.e. when the grammar that produced it has a dynamic `this`.
	// Only then does callInner maintain the two per-call stacks below; for every
	// other grammar (MetaJS, Java, Kotlin, Go, Python, ...) a call costs exactly
	// what it cost before: one boolean test.
	trackThis bool

	// pyLang says the module being run came out of a PYTHON grammar, decided the
	// same way trackThis is: a module declares only the externs it uses, and
	// js_pytruthy is emitted by python-to-llvm-ir.abnf and by no other grammar
	// (it is Python's core.truthyExt).
	//
	// It exists for ONE rule, and it needs the guard because the externs that
	// carry the rule are SHARED: js_pyget and js_pyset are the dict subscript of
	// dart, kotlin, swift, csharp, go, js, typescript, php, lua and ruby as well,
	// and in every one of those `true` and `1` are two different map keys. In
	// Python they are one key, because bool subclasses int - see pyDictFind.
	pyLang bool

	// goLang says the module being run came out of the GO grammar, decided the
	// same way pyLang is: js_gotop is emitted by go-to-llvm-ir.abnf, by no other
	// grammar, and unconditionally (it is the entry's panic-frame pop), so a go
	// module always declares it.
	//
	// It exists for ONE rule, and it needs the guard for the same reason pyLang
	// does: a STRUCT or ARRAY is a Go VALUE type and `==` on it is
	// element-by-element, so two different objects holding the same fields are
	// ONE map key - while in the ten other grammars that share js_pyget/js_pyset
	// an object key is identity. See the goLang arm of dictKeyEq.
	goLang bool

	// accessorCount counts the getter/setter properties js_defprop has defined in
	// this runtime. While it is zero - which it stays for every grammar that never
	// emits js_defprop, and for every JS program without an accessor - getMember
	// and setMember take their original path and never look for an accessor.
	accessorCount int

	// jsErrClasses is the Error hierarchy the js/ts emitter built and handed over
	// with js_jserrreg, keyed by name. The runtime's own raises (bigRaise) build
	// their instance from THESE class objects rather than from a second set of
	// their own, which is what makes `e instanceof TypeError` hold for an error
	// the engine raised. Stays nil for every grammar that is not js or ts.
	jsErrClasses map[string]*jsObject

	// hasBigInt says whether a BigInt value was ever created in this runtime.
	// BigInt is the only value type the arithmetic externals cannot handle as a
	// double, so while this is false they take exactly their original path and the
	// arbitrary-precision check costs one boolean test - nothing for the grammars
	// that have no BigInt literal at all.
	hasBigInt bool

	// pyTypes memoizes the synthetic class object type() hands out for a BUILTIN
	// Python type, so `type(1) is type(2)` holds; pyFuncNames remembers the name a
	// `def` bound a closure under, which is what f.__name__ answers. Both stay nil
	// for every grammar that never compiles Python.
	pyTypes     map[string]*jsObject
	pyFuncNames map[interface{}]string
	// pyFuncAttrTab holds a user function's own attributes (f.k = 4) as the
	// Python dict f.__dict__ answers, keyed by the closure value.
	pyFuncAttrTab map[interface{}]*jsObject
	// pyBuiltinsMod is the ONE `builtins` module object of a run (len.__self__
	// is it, so it cannot be minted afresh). abnf/jsrtpydesc.go.
	pyBuiltinsMod *jsObject
	// pyBuiltinRender is how a BUILTIN prints, keyed by the closure the emitter
	// bound the name to and filled by js_pybfn. Separate from pyFuncNames on
	// purpose: that one answers f.__name__ and holds only what a def recorded.
	pyBuiltinRender map[interface{}]string
	pySigs          map[interface{}]*pySig
	pyEllipsis      *jsObject
	// pyAliases memoizes the list[int]-style generic aliases, so the same written
	// alias is the same value every time (see pyGenericAlias).
	pyAliases map[string]*jsObject

	// curGen is the generator whose body is currently running, so js_yield knows
	// which one to suspend. Only one goroutine ever runs (the handshake below is
	// strictly alternating), so a single field is enough and step() saves and
	// restores it around a resume, which makes generators nest.
	curGen *jsGenerator

	// microQ is the job (microtask) queue promises and async functions schedule
	// onto; js_jsdrain runs it to exhaustion once the script has finished, which
	// is the only point at which this runtime has an event loop turn.
	microQ []func()
	// promFn is the one `Promise` global value, interned so that `Promise ===
	// Promise` holds and the statics can be recognized by identity.
	promFn *hostFunc
}

// ----------------------------------------------------------------------------
// Generators
//
// A generator body is compiled as an ordinary IR function; what makes it a
// generator is that calling it does not RUN it (js_genfn wraps the closure) but
// creates a jsGenerator, whose body then runs on its own goroutine. next() and
// yield are a strictly alternating handshake over two unbuffered channels, so at
// any moment exactly one of the two goroutines runs and the shared runtime state
// needs no locking - only a context switch of the per-call stacks (thisStack,
// newTargetStack), which step() does around every resume.
//
// A generator that is never exhausted leaves its goroutine parked on <-resume
// for the rest of the process; nothing else keeps it alive, so it costs one
// blocked goroutine and is collected when the program ends.
type jsGenerator struct {
	fn      interface{}
	args    []interface{}
	started bool
	done    bool

	resume chan interface{} // next(v) -> the body
	yields chan *genStep    // yield v / return -> next()

	// The call stacks of the SUSPENDED body, swapped in while it runs.
	savedThis []interface{}
	savedNT   []interface{}

	// The last handshake result, kept so the iterator can be INSPECTED without
	// advancing it - current()/key()/valid() answer from here, and getReturn()
	// from retValue once the body has returned. autoKey is the running index a
	// yield without a key of its own gets.
	lastValue interface{}
	lastKey   interface{}
	retValue  interface{}
	autoKey   float64
}

// genExit is the sentinel finish() sends INTO a suspended body so that the
// `finally` clauses wrapping its yield run before the generator is abandoned -
// JavaScript's return completion and CPython's GeneratorExit. js_yield turns it
// into a panic on the BODY's goroutine, so the unwinding is the ordinary one and
// every js_try between the yield and the body's entry runs its finally clause.
//
// It is deliberately not a *jsThrown: js_try's catch arm tests for one, so a
// `catch` clause cannot swallow a close. The C floor gets the same answer with
// an explicit `pending != GEN_EXIT` guard (runtime.c's js_try).
type genExitSignal struct{ _ byte }

var genExit interface{} = &genExitSignal{}

// genThrowSignal is what throwInto puts ON THE RESUME VALUE so that g.throw(v)
// raises v at the yield the body is parked at. It rides the resume channel rather
// than a field of the generator because a per-PROGRAM record cannot survive
// coroutines, while a value handed to one suspension is per-coroutine by
// construction - only the yield it was aimed at ever reads it.
//
// Unlike genExit it becomes an ORDINARY *jsThrown, so the body's own catch arms
// may take it; that is the whole point of throw() against return().
type genThrowSignal struct{ v interface{} }

// genStep is one handshake result: a yielded value, the return value (done), or a
// panic (a throw or a runtime error) that has to be re-raised in the resumer.
type genStep struct {
	value    interface{}
	done     bool
	panicVal interface{}
}

// step runs the body until its next yield or its return and answers the
// {value, done} object of the iterator protocol.
func (g *jsGenerator) step(rt *jsrt, sent interface{}) interface{} {
	res := newJSObject()
	if g.done {
		res.set("value", jsUndef)
		res.set("done", true)
		return res
	}
	if !g.started {
		g.started = true
		g.resume = make(chan interface{})
		g.yields = make(chan *genStep)
		go func() {
			defer func() {
				if p := recover(); p != nil {
					g.yields <- &genStep{done: true, panicVal: p}
				}
			}()
			<-g.resume // The body starts only on the first next().
			ret := rt.call(g.fn, jsUndef, g.args)
			g.yields <- &genStep{value: ret, done: true}
		}()
	}
	savedThis, savedNT := rt.thisStack, rt.newTargetStack
	rt.thisStack, rt.newTargetStack = g.savedThis, g.savedNT
	prevGen := rt.curGen
	rt.curGen = g
	g.resume <- sent
	st := <-g.yields
	rt.curGen = prevGen
	g.savedThis, g.savedNT = rt.thisStack, rt.newTargetStack
	rt.thisStack, rt.newTargetStack = savedThis, savedNT
	if st.panicVal != nil {
		g.done = true
		panic(st.panicVal)
	}
	if st.done {
		g.done = true
		g.retValue = st.value
		g.lastValue = jsUndef
		g.lastKey = jsUndef
	} else {
		g.lastValue, g.lastKey = genSplitKV(st.value, g.autoKey)
		g.autoKey++
	}
	res.set("value", st.value)
	res.set("done", st.done)
	return res
}

// genSplitKV unpacks a yielded value into (value, key). A language whose yield
// carries a key of its own hands over a small record marked __genkv (the PHP
// grammars do, for `yield $k => $v`); anything else is the value itself, with the
// running index as its key - which is also what a keyless yield means.
func genSplitKV(v interface{}, auto float64) (interface{}, interface{}) {
	if o, ok := v.(*jsObject); ok {
		if tag, has := o.props["__genkv"]; has && tag == true {
			return o.props["v"], o.props["k"]
		}
	}
	return v, auto
}

// prime runs the body up to its first yield when it has not started, so that
// current() and key() answer without the caller having to step it first - PHP's
// Generator does exactly this.
func (g *jsGenerator) prime(rt *jsrt) {
	if !g.started && !g.done {
		g.step(rt, jsUndef)
	}
}

// closeBody abandons the body, RUNNING the finally clauses that wrap its
// suspended yield (see genExit). A body that never started has none to run, so
// it is only marked done - which is what node and CPython do too.
func (g *jsGenerator) closeBody(rt *jsrt) {
	if g.done {
		return
	}
	if !g.started {
		g.done = true
		g.lastValue = jsUndef
		g.lastKey = jsUndef
		return
	}
	savedThis, savedNT := rt.thisStack, rt.newTargetStack
	rt.thisStack, rt.newTargetStack = g.savedThis, g.savedNT
	prevGen := rt.curGen
	rt.curGen = g
	g.resume <- genExit
	st := <-g.yields
	rt.curGen = prevGen
	g.savedThis, g.savedNT = rt.thisStack, rt.newTargetStack
	rt.thisStack, rt.newTargetStack = savedThis, savedNT
	// A body that swallowed the sentinel and yielded again is abandoned where it
	// stands (its goroutine stays parked), the same answer the C floor gives:
	// node's TypeError and CPython's RuntimeError for that shape are not
	// expressible in either engine.
	g.done = true
	g.lastValue = jsUndef
	g.lastKey = jsUndef
	if st.panicVal != nil && st.panicVal != genExit {
		panic(st.panicVal)
	}
}

// finish is generator.return(v): close the body and answer the record the
// iterator protocol wants.
func (g *jsGenerator) finish(rt *jsrt, v interface{}) interface{} {
	g.closeBody(rt)
	g.retValue = v
	res := newJSObject()
	res.set("value", v)
	res.set("done", true)
	return res
}

// throwInto is generator.throw(v): raise v AT the yield the body is parked at,
// so the body's own try/catch/finally see it. The twin of runtime.c's gen_throw,
// and the three shapes are node's and CPython 3.14's, probed rather than reasoned
// about: suspended raises inside the body (a catch that yields again answers
// {value, false}, one that returns answers {value: ret, true}, and an uncaught one
// propagates out of throw() as step()'s re-panic already does); NOT STARTED runs
// nothing at all - there is no suspension point and so no `finally` - and marks
// the generator done; ALREADY DONE just propagates.
func (g *jsGenerator) throwInto(rt *jsrt, v interface{}) interface{} {
	if g.done {
		panic(&jsThrown{value: v})
	}
	if !g.started {
		g.done = true
		g.lastValue = jsUndef
		g.lastKey = jsUndef
		panic(&jsThrown{value: v})
	}
	return g.step(rt, &genThrowSignal{v: v})
}

// drain materializes the remaining values of a generator, which is how a for-of
// loop over one gets its sequence (see js_iterable).
func (g *jsGenerator) drain(rt *jsrt) *jsArray {
	out := &jsArray{}
	for {
		st, ok := g.step(rt, jsUndef).(*jsObject)
		if !ok {
			break
		}
		if d, _ := st.props["done"].(bool); d {
			break
		}
		out.elems = append(out.elems, st.props["value"])
	}
	return out
}

// ----------------------------------------------------------------------------
// Promises and the microtask queue (js / typescript)
//
// A promise is a PLAIN OBJECT carrying hidden slots only, so the three engines
// (this twin, languages/lib/js-rt.metajs and the two *-interpreter.abnf start
// scripts) can run the SAME algorithm over the same value model instead of each
// inventing a private representation - keysOf hides a '__' slot, so a promise
// still has no own keys, and typeof answers "object" with no arm of its own.
//
//	__prom   true            marks the object
//	__ps     0 | 1 | 2       pending / fulfilled / rejected
//	__pv     any             the settled value or the rejection reason
//	__pcb    array           reactions registered while still pending
//
// A reaction is {f, r, p}: the fulfil handler, the reject handler and the
// derived promise then() answered (undefined for an await, which has none).
//
// 'async' is 'awaits as yields': the emitter compiles an async body as an
// ordinary generator body, `await e` as js_yield, and js_jsasyncfn drives the
// generator from the microtask queue - so every statement form works inside an
// async function for exactly the reason it works inside a generator.
func promNew() *jsObject {
	p := newJSObject()
	p.set("__prom", true)
	p.set("__ps", float64(0))
	p.set("__pv", jsUndef)
	p.set("__pcb", &jsArray{})
	return p
}

func promIs(v interface{}) bool {
	o, ok := v.(*jsObject)
	if !ok {
		return false
	}
	t, has := o.props["__prom"]
	return has && t == true
}

func promState(p *jsObject) int { s, _ := p.props["__ps"].(float64); return int(s) }

// enqueue appends one job to the microtask queue. The queue is drained by
// js_jsdrain, which the emitter calls once the whole script has run.
func (rt *jsrt) enqueue(job func()) { rt.microQ = append(rt.microQ, job) }

// promSettle moves a pending promise to its final state and hands every
// registered reaction to the queue, in registration order.
func (rt *jsrt) promSettle(p *jsObject, state int, v interface{}) {
	if promState(p) != 0 {
		return
	}
	p.set("__ps", float64(state))
	p.set("__pv", v)
	cbs, _ := p.props["__pcb"].(*jsArray)
	p.set("__pcb", &jsArray{})
	if cbs != nil {
		for _, r := range cbs.elems {
			reaction, _ := r.(*jsObject)
			rt.promEnqueueReaction(p, reaction)
		}
	}
}

// promResolve is the spec's resolve procedure: resolving with a THENABLE adopts
// its state through a job of its own, which is where the two extra ticks that
// every ordering test measures come from.
func (rt *jsrt) promResolve(p *jsObject, x interface{}) {
	if promState(p) != 0 {
		return
	}
	if xo, ok := x.(*jsObject); ok && xo == p {
		rt.promSettle(p, 2, "TypeError: Chaining cycle detected for promise")
		return
	}
	if xo, ok := x.(*jsObject); ok {
		then := rt.getMember(xo, "then")
		if promIs(xo) || isCallable(then) {
			rt.enqueue(func() {
				defer func() {
					if e := recover(); e != nil {
						if t, ok := e.(*jsThrown); ok {
							rt.promSettle(p, 2, t.value)
							return
						}
						panic(e)
					}
				}()
				if promIs(xo) {
					rt.promThen(xo,
						jsHostFunc("resolve", func(rt *jsrt, this uint64, args []interface{}) interface{} {
							rt.promResolve(p, argAt(args, 0))
							return jsUndef
						}),
						jsHostFunc("reject", func(rt *jsrt, this uint64, args []interface{}) interface{} {
							rt.promSettle(p, 2, argAt(args, 0))
							return jsUndef
						}), nil)
					return
				}
				rt.call(then, xo, []interface{}{
					jsHostFunc("resolve", func(rt *jsrt, this uint64, args []interface{}) interface{} {
						rt.promResolve(p, argAt(args, 0))
						return jsUndef
					}),
					jsHostFunc("reject", func(rt *jsrt, this uint64, args []interface{}) interface{} {
						rt.promSettle(p, 2, argAt(args, 0))
						return jsUndef
					})})
			})
			return
		}
	}
	rt.promSettle(p, 1, x)
}

// promEnqueueReaction is the spec's NewPromiseReactionJob: run the handler for
// the state the promise settled in, and settle the derived promise with what it
// answered - or with what it threw.
func (rt *jsrt) promEnqueueReaction(p *jsObject, reaction *jsObject) {
	state := promState(p)
	arg := p.props["__pv"]
	rt.enqueue(func() {
		var handler interface{}
		if state == 1 {
			handler = reaction.props["f"]
		} else {
			handler = reaction.props["r"]
		}
		derived, _ := reaction.props["p"].(*jsObject)
		if !isCallable(handler) {
			if derived != nil {
				if state == 1 {
					rt.promResolve(derived, arg)
				} else {
					rt.promSettle(derived, 2, arg)
				}
			}
			return
		}
		var out interface{}
		threw := false
		var thrown interface{}
		func() {
			defer func() {
				if e := recover(); e != nil {
					if t, ok := e.(*jsThrown); ok {
						threw, thrown = true, t.value
						return
					}
					panic(e)
				}
			}()
			out = rt.call(handler, jsUndef, []interface{}{arg})
		}()
		if derived == nil {
			return
		}
		if threw {
			rt.promSettle(derived, 2, thrown)
		} else {
			rt.promResolve(derived, out)
		}
	})
}

// promThen registers one reaction. derived is the promise then() answers, or nil
// for an await, which consumes the result itself.
func (rt *jsrt) promThen(p *jsObject, onF, onR interface{}, derived *jsObject) {
	reaction := newJSObject()
	reaction.set("f", onF)
	reaction.set("r", onR)
	if derived != nil {
		reaction.set("p", derived)
	} else {
		reaction.set("p", jsUndef)
	}
	if promState(p) == 0 {
		cbs, _ := p.props["__pcb"].(*jsArray)
		cbs.elems = append(cbs.elems, reaction)
		return
	}
	rt.promEnqueueReaction(p, reaction)
}

// promResolveValue is the spec's PromiseResolve: a promise is its own, anything
// else gets a fresh already-resolving promise. Returning x unchanged is what
// keeps `await aPromise` one tick rather than three.
func (rt *jsrt) promResolveValue(x interface{}) *jsObject {
	if promIs(x) {
		return x.(*jsObject)
	}
	p := promNew()
	rt.promResolve(p, x)
	return p
}

// promMethod answers p.then / p.catch / p.finally. The names are dispatched in
// memberCall rather than stored on the object so that a promise keeps no own
// keys at all.
func (rt *jsrt) promMethod(p *jsObject, name string, args []interface{}) (interface{}, bool) {
	switch name {
	case "then":
		derived := promNew()
		rt.promThen(p, argAt(args, 0), argAt(args, 1), derived)
		return derived, true
	case "catch":
		derived := promNew()
		rt.promThen(p, jsUndef, argAt(args, 0), derived)
		return derived, true
	case "finally":
		fn := argAt(args, 0)
		derived := promNew()
		pass := jsHostFunc("finally", func(rt *jsrt, this uint64, a []interface{}) interface{} {
			if isCallable(fn) {
				rt.call(fn, jsUndef, nil)
			}
			return argAt(a, 0)
		})
		rethrow := jsHostFunc("finally", func(rt *jsrt, this uint64, a []interface{}) interface{} {
			if isCallable(fn) {
				rt.call(fn, jsUndef, nil)
			}
			panic(&jsThrown{value: argAt(a, 0)})
		})
		rt.promThen(p, pass, rethrow, derived)
		return derived, true
	}
	return nil, false
}

// asyncStep resumes an async body once and re-arms it. sent is the value the
// pending await answers with; when isThrow is set the await raises it instead,
// which is how a rejected promise reaches a try/catch inside the body (the
// marker record is unpacked by js_jsawaitv at the await site, so neither
// generator engine needs a throw-in channel of its own).
func (rt *jsrt) asyncStep(g *jsGenerator, p *jsObject, sent interface{}, isThrow bool) {
	send := sent
	if isThrow {
		m := newJSObject()
		m.set("__athrow", true)
		m.set("v", sent)
		send = m
	}
	var res interface{}
	threw := false
	var thrown interface{}
	func() {
		defer func() {
			if e := recover(); e != nil {
				if t, ok := e.(*jsThrown); ok {
					threw, thrown = true, t.value
					return
				}
				panic(e)
			}
		}()
		res = g.step(rt, send)
	}()
	if threw {
		rt.promSettle(p, 2, thrown)
		return
	}
	st, _ := res.(*jsObject)
	if d, _ := st.props["done"].(bool); d {
		rt.promResolve(p, st.props["value"])
		return
	}
	awaited := rt.promResolveValue(st.props["value"])
	rt.promThen(awaited,
		jsHostFunc("await", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			rt.asyncStep(g, p, argAt(args, 0), false)
			return jsUndef
		}),
		jsHostFunc("await", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			rt.asyncStep(g, p, argAt(args, 0), true)
			return jsUndef
		}), nil)
}

// promiseGlobal answers the `Promise` binding: a function value, so that
// `new Promise(executor)` constructs one and `typeof Promise` is "function".
// The executor runs SYNCHRONOUSLY, as in the specification.
func (rt *jsrt) promiseGlobal() *hostFunc {
	if rt.promFn != nil {
		return rt.promFn
	}
	rt.promFn = jsHostFunc("Promise", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		p := promNew()
		exec := argAt(args, 0)
		if !isCallable(exec) {
			// node throws this and a try/catch recovers it, where rt.fail is an
			// engine abort js_try deliberately does not see. bigRaise is the same
			// *jsThrown panic js_throw builds, and jsErrFromText turns the text
			// into the very TypeError the program's own `new TypeError` builds.
			// The resolver's VALUE is part of node's message; node spells an
			// object `#<Object>` where String(v) says "[object Object]", which is
			// a node-internal rendering with no other user here.
			rt.bigRaise("TypeError: Promise resolver %s is not a function", rt.jsvString(exec))
		}
		res := jsHostFunc("resolve", func(rt *jsrt, this uint64, a []interface{}) interface{} {
			rt.promResolve(p, argAt(a, 0))
			return jsUndef
		})
		rej := jsHostFunc("reject", func(rt *jsrt, this uint64, a []interface{}) interface{} {
			rt.promSettle(p, 2, argAt(a, 0))
			return jsUndef
		})
		func() {
			defer func() {
				if e := recover(); e != nil {
					if t, ok := e.(*jsThrown); ok {
						rt.promSettle(p, 2, t.value)
						return
					}
					panic(e)
				}
			}()
			rt.call(exec, jsUndef, []interface{}{res, rej})
		}()
		return p
	})
	return rt.promFn
}

// promStatic answers Promise.resolve / reject / all / allSettled / race / any.
func (rt *jsrt) promStatic(name string, args []interface{}) (interface{}, bool) {
	switch name {
	case "resolve":
		return rt.promResolveValue(argAt(args, 0)), true
	case "reject":
		p := promNew()
		rt.promSettle(p, 2, argAt(args, 0))
		return p, true
	case "all", "allSettled", "race", "any":
		src, ok := argAt(args, 0).(*jsArray)
		if !ok {
			rt.fail("Promise.%s needs an array", name)
		}
		out := promNew()
		n := len(src.elems)
		results := &jsArray{elems: make([]interface{}, n)}
		for i := range results.elems {
			results.elems[i] = jsUndef
		}
		left := n
		if n == 0 {
			switch name {
			case "all", "allSettled":
				rt.promResolve(out, results)
			case "any":
				rt.promSettle(out, 2, "AggregateError: All promises were rejected")
			}
			return out, true
		}
		for i, el := range src.elems {
			idx := i
			ep := rt.promResolveValue(el)
			rt.promThen(ep,
				jsHostFunc("all", func(rt *jsrt, this uint64, a []interface{}) interface{} {
					v := argAt(a, 0)
					switch name {
					case "race", "any":
						rt.promResolve(out, v)
						return jsUndef
					case "allSettled":
						r := newJSObject()
						r.set("status", "fulfilled")
						r.set("value", v)
						results.elems[idx] = r
					default:
						results.elems[idx] = v
					}
					left--
					if left == 0 {
						rt.promResolve(out, results)
					}
					return jsUndef
				}),
				jsHostFunc("all", func(rt *jsrt, this uint64, a []interface{}) interface{} {
					v := argAt(a, 0)
					switch name {
					case "race":
						rt.promSettle(out, 2, v)
						return jsUndef
					case "all":
						rt.promSettle(out, 2, v)
						return jsUndef
					case "allSettled":
						r := newJSObject()
						r.set("status", "rejected")
						r.set("reason", v)
						results.elems[idx] = r
					default: // any
						results.elems[idx] = v
					}
					left--
					if left == 0 {
						if name == "any" {
							rt.promSettle(out, 2, "AggregateError: All promises were rejected")
						} else {
							rt.promResolve(out, results)
						}
					}
					return jsUndef
				}), nil)
		}
		return out, true
	}
	return nil, false
}

// drainMicrotasks runs the job queue to exhaustion. This is the whole event
// loop: there are no timers or I/O here, so one drain after the script has run
// is every turn the program will ever get.
//
// An UNHANDLED REJECTION is deliberately not reported, in any of the three
// engines. node terminates the process on one; here the interpreter half drives
// an async body by REPLAY (see the generator section of js-interpreter.abnf's
// start script), so a rejected promise the body created is created again on
// every resume and the copies nothing awaited would be reported as unhandled -
// spurious aborts on correct programs. Reporting it in the compiler halves only
// would make the halves disagree on an error path, so no engine reports it and
// the divergence from node is documented instead.
func (rt *jsrt) drainMicrotasks() {
	for len(rt.microQ) > 0 {
		job := rt.microQ[0]
		rt.microQ = rt.microQ[1:]
		job()
	}
}

// jsAccessor is a getter/setter property: the value stored under a key when the
// property was defined with js_defprop (an object literal's `get x() {}` / `set
// x(v) {}`, or the accessor member of a class). getMember calls the getter with
// the object as the receiver instead of returning the record, and setMember calls
// the setter instead of overwriting it - the record itself never escapes to
// compiled code.
type jsAccessor struct {
	get interface{}
	set interface{}
}

// newJSRT creates a runtime. The bindings become the variables of the root
// scope (the host globals that the compiled programs can see).
func newJSRT(bindings map[string]interface{}) *jsrt {
	rt := &jsrt{
		strIntern: map[string]uint64{},
		numIntern: map[uint64]uint64{},
		objIntern: map[interface{}]uint64{},
		retSlot:   jsUndef,
		curPos:    -1,
	}
	rt.alloc(jsUndef) // Handles 0..3, the singletons.
	rt.alloc(jsNull)
	rt.alloc(false)
	rt.alloc(true)
	rt.root = &jsScope{}
	for k, v := range bindings {
		rt.root.put(k, v)
	}
	return rt
}

// attach loads a module into the runtime: the module gets its own machine
// (memory, globals, functions) whose js_* externals all work on the shared
// value table and scope world of this runtime.
func (rt *jsrt) attach(m *ir.Module) *machine {
	ma := newMachine(m, "")
	ma.externs = rt.externs(ma)
	ma.bindExterns() // Resolve every declared function to its handler now, not per call.
	// A module declares only the externs it uses, so its function list says whether
	// this program can ever ask for a dynamic `this` (see trackThis) and which
	// grammar emitted it (see pyLang). Both latch on: a run is one language, and a
	// multi-module program need not repeat the extern in every module.
	for _, f := range m.Funcs {
		switch f.GlobalIdent.Name() {
		case "js_this", "js_newtarget":
			rt.trackThis = true
		case "js_pytruthy":
			rt.pyLang = true
		case "js_gotop":
			rt.goLang = true
		}
	}
	ma.relNew, ma.relThru = jsReclaimable, jsThroughArgs
	ma.release, ma.pin = rt.releaseHandle, rt.pinArray
	return ma
}

// jsReclaimable lists the externs whose result the IR machine may reclaim: a
// per-block scope and a per-call argument array, the two values a compiled
// program creates once per executed block and once per call. See machine.recycle.
var jsReclaimable = map[string]bool{"js_scope_new": true, "js_arr_new": true}

// jsThroughArgs lists, per extern, the argument positions whose handle the
// extern only READS: it resolves the handle, works on the value for the duration
// of the call and keeps neither the handle nor a reference that would need the
// handle again.
//
// js_scope_new is here because a child scope stores its parent as a Go POINTER -
// the parent's handle is never asked for again (nothing in the runtime turns a
// *jsScope back into a handle except the creation site itself). js_arr_push
// appends to the array in position 0 without keeping it; position 1 is the
// pushed VALUE and is deliberately not listed. The call externs read the
// argument array in their last position: js_mcall/js_rmcall/js_supercall reach
// their callee through rt.call, which boxes a fresh array around the elements,
// and js_call hands the array itself to a compiled callee - which is why the
// callee is asked separately whether it keeps it (machine.pin, see recycle).
//
// js_closure is deliberately absent: it stores the scope HANDLE in the closure,
// which is exactly how a scope outlives the frame that made it. Any extern not
// listed is treated as retaining every argument, so forgetting one costs memory,
// never correctness.
var jsThroughArgs = map[string]uint32{
	"js_scope_new": 1 << 0, "js_scope_decl": 1 << 0, "js_scope_get": 1 << 0,
	"js_scope_set": 1 << 0, "js_scope_set_or_create": 1 << 0, "js_scope_own": 1 << 0,
	"js_kget": 1 << 0, "js_kset": 1 << 0, "js_tdecl": 1 << 0, "js_tset": 1 << 0,
	"js_pyset_var": 1 << 0, "js_pyglobal": 1 << 0, "js_pynonlocal": 1 << 0,

	"js_arr_push": 1 << 0, "js_arg": 1 << 0, "js_pyrest": 1 << 0,
	"js_pyprint": 1 << 0, "js_pyexc": 1 << 1,
	"js_call": 1 << 2, "js_mcall": 1 << 2, "js_rmcall": 1 << 2, "js_supercall": 1 << 3,
}

// releaseHandle drops the value behind a handle the IR machine proved dead. Only
// a scope or an argument array is ever released. The slot is reused, but only
// after relQuarantine later releases: a handle that (against the analysis) is
// used again soon after finds nil and fails loudly instead of silently reading
// whatever moved in, while a long run's table stays the size of its live set.
// The value forgets its handle, so wrapping it again hands out a fresh one.
func (rt *jsrt) releaseHandle(h uint64) {
	if h&numTag != 0 || h >= rt.count {
		return
	}
	chunk, off := rt.chunks[h>>tblChunkBits], h&tblChunkMask
	switch v := chunk[off].(type) {
	case *jsScope:
		v.id = jsHandleID{}
	case *jsArray:
		if v.pinned {
			return
		}
		v.id = jsHandleID{}
	default:
		return
	}
	chunk[off] = nil
	rt.freed = append(rt.freed, h)
}

const relQuarantine = 1 << 16

// pinArray marks an argument array that the callee about to run can keep past
// its own return, so the call site must not reclaim it. Handles that are not an
// array (a helper function of the same shape) are simply ignored.
func (rt *jsrt) pinArray(h uint64) {
	if h&numTag != 0 || h >= rt.count {
		return
	}
	if arr, ok := rt.chunks[h>>tblChunkBits][h&tblChunkMask].(*jsArray); ok {
		arr.pinned = true
	}
}

// setRootVar binds or rebinds a host global. The frozen engine uses this to
// point 'up' and the stack functions at the environment of the current tag.
func (rt *jsrt) setRootVar(name string, v interface{}) {
	rt.root.put(name, v)
}

// newScopeHandle creates a scope (a child of the root scope by default) and
// returns its handle. The frozen engine passes it as the shared environment
// of all scripts of one compile run.
func (rt *jsrt) newScopeHandle(parent *jsScope) uint64 {
	if parent == nil {
		parent = rt.root
	}
	return rt.wrap(&jsScope{parent: parent})
}

func (rt *jsrt) fail(format string, args ...interface{}) {
	panic("js runtime error: " + fmt.Sprintf(format, args...))
}

// ----------------------------------------------------------------------------
// Handles

// wrap turns a Go/JS value into a handle. Numeric Go kinds all become JS
// numbers (float64); the original type is restored by convertToType when the
// value is passed back into a typed Go function.
func (rt *jsrt) wrap(v interface{}) uint64 {
	switch t := v.(type) {
	case nil:
		return jsHUndefined
	case jsUndefT:
		return jsHUndefined
	case jsNullT:
		return jsHNull
	case bool:
		if t {
			return jsHTrue
		}
		return jsHFalse
	case float64:
		return rt.wrapNum(t)
	case string:
		return rt.wrapStr(t)
	case int:
		return rt.wrapNum(float64(t))
	case int32:
		return rt.wrapNum(float64(t))
	case int64:
		return rt.wrapNum(float64(t))
	case uint32:
		return rt.wrapNum(float64(t))
	case uint64:
		return rt.wrapNum(float64(t))

	// The values the runtime creates itself carry their handle (see
	// jsHandleID): no reflection, no intern map, no map growth.
	case *jsObject:
		return rt.wrapID(&t.id, v)
	case *jsArray:
		return rt.wrapID(&t.id, v)
	case *jsScope:
		return rt.wrapID(&t.id, v)
	case *jsClosure:
		return rt.wrapID(&t.id, v)
	case *boundMethod:
		// Every member read builds a fresh one, so interning it by identity
		// would only grow the intern map.
		return rt.alloc(v)
	}

	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rt.wrapNum(float64(rv.Int())) // Named integer types (enum.IPred, r.OperatorID, ...).
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return rt.wrapNum(float64(rv.Uint()))
	case reflect.Float32, reflect.Float64:
		return rt.wrapNum(rv.Float())
	case reflect.String:
		return rt.wrapStr(rv.String())
	case reflect.Ptr:
		if rv.IsNil() {
			return jsHNull
		}
		return rt.wrapIdentity(v)
	case reflect.Map:
		// Maps are uncomparable as interface keys; their data pointer
		// identifies them well enough for interning.
		type mapKey struct {
			p uintptr
			t reflect.Type
		}
		return rt.wrapIdentityKey(mapKey{rv.Pointer(), rv.Type()}, v)
	default:
		// Funcs, slices, structs by value...: a fresh handle every time.
		// (Funcs must NOT be interned by their code pointer: all reflect
		// method values share one adapter, so bound methods of different
		// receivers would collide.)
		return rt.alloc(v)
	}
}

// alloc puts v in a free table slot and returns its handle: the oldest
// released slot once more than relQuarantine wait, else a never-used one.
func (rt *jsrt) alloc(v interface{}) uint64 {
	if len(rt.freed)-rt.freeHead > relQuarantine {
		h := rt.freed[rt.freeHead]
		rt.freeHead++
		if rt.freeHead >= relQuarantine && rt.freeHead*2 >= len(rt.freed) {
			rt.freed = rt.freed[:copy(rt.freed, rt.freed[rt.freeHead:])]
			rt.freeHead = 0
		}
		rt.chunks[h>>tblChunkBits][h&tblChunkMask] = v
		return h
	}
	h := rt.count
	if int(h>>tblChunkBits) == len(rt.chunks) {
		rt.chunks = append(rt.chunks, make([]interface{}, tblChunkSize))
	}
	rt.chunks[h>>tblChunkBits][h&tblChunkMask] = v
	rt.count = h + 1
	return h
}

// wrapID hands out the handle a runtime-owned value carries, creating it on
// first use. A value first wrapped by ANOTHER runtime keeps that runtime's
// handle in the field and goes through this runtime's intern map instead, so
// identity stays right in both.
func (rt *jsrt) wrapID(id *jsHandleID, v interface{}) uint64 {
	if id.hrt == rt {
		return id.h
	}
	if id.hrt != nil {
		return rt.wrapIdentity(v)
	}
	id.h, id.hrt = rt.alloc(v), rt
	return id.h
}

func (rt *jsrt) wrapIdentity(v interface{}) uint64 {
	return rt.wrapIdentityKey(v, v)
}

func (rt *jsrt) wrapIdentityKey(key, v interface{}) uint64 {
	if h, ok := rt.objIntern[key]; ok {
		return h
	}
	h := rt.alloc(v)
	rt.objIntern[key] = h
	return h
}

func (rt *jsrt) wrapNum(f float64) uint64 {
	if h, ok := numHandle(f); ok {
		return h
	}
	// The intern key is the bit pattern, not the value: -0.0 == 0.0 as a float
	// key, so a value-keyed map handed out one shared handle for both zeros
	// (whichever was wrapped first supplied the other). NaN stays uninterned:
	// its handles need not be stable.
	bits := math.Float64bits(f)
	if f == f {
		if h, ok := rt.numIntern[bits]; ok {
			return h
		}
	}
	h := rt.alloc(f)
	if f == f {
		rt.numIntern[bits] = h
	}
	return h
}

// wrapStr interns s. The map lookup hashes the whole string, so for long
// strings the handle is remembered in the string cache (see strEntry): a loop
// that keeps reading one big string would otherwise re-hash it per iteration.
func (rt *jsrt) wrapStr(s string) uint64 {
	if len(s) < strMemoMin {
		return rt.internStr(s)
	}
	e := rt.strEntry(s)
	if e.h == 0 {
		e.h = rt.internStr(s)
	}
	return e.h
}

func (rt *jsrt) internStr(s string) uint64 {
	if h, ok := rt.strIntern[s]; ok {
		return h
	}
	h := rt.alloc(s)
	rt.strIntern[s] = h
	return h
}

func (rt *jsrt) unwrap(h uint64) interface{} {
	if h&numTag != 0 {
		return numValue(h)
	}
	if h >= rt.count {
		rt.fail("invalid handle %d", h)
	}
	return rt.chunks[h>>tblChunkBits][h&tblChunkMask]
}

// ----------------------------------------------------------------------------
// JS value semantics

func (rt *jsrt) truthy(v interface{}) bool {
	switch t := v.(type) {
	case jsUndefT, jsNullT:
		return false
	case bool:
		return t
	case float64:
		return t != 0 && t == t
	case jsPhpFlo: // PHP's boxed float (see jsrtphp.go); 0.0 is falsy.
		return t.f != 0
	case jsJFlo: // Java's boxed double (see jsrtjvm.go).
		return t.f != 0 && t.f == t.f
	case jsGInt: // A sized integer (see jsrtint.go); 0 is falsy like any other.
		return t.v != 0
	case jsDartFlo: // Dart's boxed double (see jsrtdart.go).
		return t.f != 0 && t.f == t.f
	case *jsPyFlo: // Python's boxed float: 0.0 is falsy, NaN is TRUTHY (unlike JS).
		return t.f != 0
	case string:
		return len(t) > 0
	case *jsBigInt:
		return t.v.Sign() != 0
	default:
		return true
	}
}

func (rt *jsrt) toNumber(v interface{}) float64 {
	switch t := v.(type) {
	case jsChar: // Kotlin's Char does arithmetic and compares as its code.
		return float64(t.code)
	case jsFlo: // Ruby's boxed Float / Rational / Complex are numbers.
		return t.f
	case jsPhpFlo: // PHP's boxed float.
		return t.f
	case jsJFlo: // Java's boxed double.
		return t.f
	case jsGInt: // A sized integer (see jsrtint.go); a uint64 reads unsigned.
		if t.u && t.w == 64 {
			return float64(uint64(t.v))
		}
		return float64(t.v)
	case jsDartFlo: // Dart's boxed double.
		return t.f
	case *jsPyFlo: // Python's boxed float.
		return t.f
	case jsRat:
		return rubyRatToF(t)
	case jsCpx:
		return rubyToF(t.re)
	case jsUndefT:
		return math.NaN()
	case jsNullT:
		return 0
	case bool:
		if t {
			return 1
		}
		return 0
	case float64:
		return t
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return 0
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return math.NaN()
		}
		return f
	default:
		return math.NaN()
	}
}

// jsNumString formats a number like JS does for the common cases. Extreme
// exponents may differ slightly from V8 ("1e-07" instead of "1e-7").
func jsNumString(f float64) string {
	if f != f {
		return "NaN"
	}
	if math.IsInf(f, 1) {
		return "Infinity"
	}
	if math.IsInf(f, -1) {
		return "-Infinity"
	}
	if f == 0 {
		return "0"
	}
	if f == math.Trunc(f) && math.Abs(f) < 1e21 {
		return strconv.FormatFloat(f, 'f', -1, 64)
	}
	if math.Abs(f) >= 1e-6 && math.Abs(f) < 1e21 {
		return strconv.FormatFloat(f, 'f', -1, 64)
	}
	// Go pads single-digit exponents to two digits ("1e-07"); JS does not
	// ("1e-7"). Strip the leading exponent zeros (the sign stays).
	s := strconv.FormatFloat(f, 'g', -1, 64)
	if e := strings.IndexByte(s, 'e'); e >= 0 && e+2 < len(s) {
		digits := e + 2 // Behind "e+"/"e-".
		trimmed := digits
		for trimmed < len(s)-1 && s[trimmed] == '0' {
			trimmed++
		}
		s = s[:digits] + s[trimmed:]
	}
	return s
}

func (rt *jsrt) toString(v interface{}) string {
	switch t := v.(type) {
	case jsChar: // Kotlin's Char renders as its glyph, not as its code.
		return string(rune(t.code))
	case jsSym: // Ruby's Symbol renders as its bare name.
		return t.s
	case jsFlo: // Ruby's Float keeps its decimal point (1000.0), unlike an Integer.
		return rubyFloStr(t.f)
	case jsPhpFlo: // PHP prints a float without a fraction as an integer ((string)3.0 == "3").
		return phpFloStr(t.f)
	case jsJFlo: // Each language's own float rendering (see jsrtjvm.go).
		return jvmFloText(t)
	case jsGInt: // A sized integer prints its digits, unsigned where it is (jsrtint.go).
		return giStr(t)
	case jsDartFlo: // Dart's double keeps its point: 1.0, not 1 (jsrtdart.go).
		return dartFloStr(t.f)
	case *jsPyFlo: // Python's float keeps its point too, in CPython's own window.
		return pyFloStr(t.f)
	case jsRat:
		return rubyRatText(t)
	case jsCpx:
		return rt.rubyCpxText(t, false)
	case jsUndefT:
		return "undefined"
	case jsNullT:
		return "null"
	case bool:
		if t {
			return "true"
		}
		return "false"
	case float64:
		return jsNumString(t)
	case string:
		return t
	case *jsArray:
		parts := make([]string, len(t.elems))
		for i, e := range t.elems {
			if _, u := e.(jsUndefT); u {
				continue // undefined and null join as empty strings.
			}
			if _, n := e.(jsNullT); n {
				continue
			}
			parts[i] = rt.toString(e)
		}
		return strJoin(parts, ",")
	case *jsBigInt:
		return t.v.String()
	case *jsObject:
		return "[object Object]"
	case *jsClosure, *hostFunc, *boundMethod:
		return "[function]"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// toGoNatural converts a JS value for an interface{} slot (print arguments,
// stack pushes into typed-less Go APIs). Integral numbers become int64 like in
// goja, so fmt verbs like %d and %c work.
func (rt *jsrt) toGoNatural(v interface{}) interface{} {
	switch t := v.(type) {
	case jsChar: // print/println show the glyph.
		return string(rune(t.code))
	case jsSym: // print/println show the symbol's name.
		return t.s
	case jsFlo, jsRat, jsCpx: // print/println show Ruby's rendering (1000.0, 1/2, 0+2i).
		return rt.rubyStr(v)
	case jsPhpFlo: // print/println show PHP's rendering (3.0 prints as 3).
		return phpFloStr(t.f)
	case jsJFlo: // print/println show the language's own float rendering.
		return jvmFloText(t)
	case jsGInt: // print/println show a sized integer's digits (jsrtint.go).
		if t.u {
			return giU(t.v, t.w)
		}
		return t.v
	case jsDartFlo: // print/println show Dart's double rendering (jsrtdart.go).
		return dartFloStr(t.f)
	case *jsPyFlo: // print/println show Python's float rendering.
		return pyFloStr(t.f)
	case jsUndefT, jsNullT:
		return nil
	case float64:
		// goja normalizes every integral number that fits an int64 to its int
		// representation (floatToValue), so its Export delivers int64 for them
		// and fmt prints digits. Mirror that exactly (no 1e15 cutoff), so
		// -frozen prints the same. -0 stays a float like in goja (fmt prints it
		// as -0).
		if t == math.Trunc(t) && !math.IsInf(t, 0) && t >= math.MinInt64 && t < math.MaxInt64 &&
			!(t == 0 && math.Signbit(t)) {
			return int64(t)
		}
		return t
	case *jsArray:
		out := make([]interface{}, len(t.elems))
		for i, e := range t.elems {
			out[i] = rt.toGoNatural(e)
		}
		return out
	case *jsObject:
		out := map[string]interface{}{}
		for k, p := range t.props {
			out[k] = rt.toGoNatural(p)
		}
		return out
	default:
		return v // Strings, bools, closures, natives.
	}
}

func (rt *jsrt) jsAdd(a, b interface{}) interface{} {
	as, aIsStr := a.(string)
	bs, bIsStr := b.(string)
	if aIsStr || bIsStr {
		if !aIsStr {
			as = rt.toString(a)
		}
		if !bIsStr {
			bs = rt.toString(b)
		}
		return strConcat(as, bs)
	}
	if _, ok := a.(*jsArray); ok {
		return strConcat(rt.toString(a), rt.toString(b))
	}
	if _, ok := b.(*jsArray); ok {
		return strConcat(rt.toString(a), rt.toString(b))
	}
	if _, ok := a.(*jsObject); ok {
		return strConcat(rt.toString(a), rt.toString(b))
	}
	if _, ok := b.(*jsObject); ok {
		return strConcat(rt.toString(a), rt.toString(b))
	}
	if rt.hasBigInt {
		if r, ok := bigArith('+', a, b); ok {
			return r
		}
	}
	return rt.toNumber(a) + rt.toNumber(b)
}

func isUndefOrNull(v interface{}) bool {
	switch v.(type) {
	case jsUndefT, jsNullT:
		return true
	}
	return false
}

func (rt *jsrt) strictEq(a, b interface{}) bool {
	// Kotlin's Char: unbox both sides first, so a Char equals a Char of the same code
	// and equals that code as an Int (the interpreter's kEq does exactly this).
	if av, isChar := charCode(a); isChar {
		bv, _ := charCode(b)
		return rt.strictEq(av, bv)
	}
	if bv, isChar := charCode(b); isChar {
		return rt.strictEq(a, bv)
	}
	// Java's boxed double (jsrtjvm.go): 1.0 == 1 holds, so the box is compared by
	// VALUE against the other side's number - never by identity.
	if af, isFlo := a.(jsJFlo); isFlo {
		return jvmNumEq(af.f, b)
	}
	if bf, isFlo := b.(jsJFlo); isFlo {
		return jvmNumEq(bf.f, a)
	}
	// A sized integer (jsrtint.go) compares by VALUE too, never by identity: a
	// box and a plain number are the same integer, and int8(1) == 1 holds.
	if _, isInt := a.(jsGInt); isInt {
		return giIsNumeric(b) && rt.giEq(a, b)
	}
	if _, isInt := b.(jsGInt); isInt {
		return giIsNumeric(a) && rt.giEq(a, b)
	}
	// Dart's boxed double (jsrtdart.go) compares by VALUE too: 1 == 1.0 holds there
	// and two boxes of the same number must not differ just because they are objects.
	if af, isFlo := a.(jsDartFlo); isFlo {
		return dartIsNum(b) && af.f == rt.toNumber(b)
	}
	if bf, isFlo := b.(jsDartFlo); isFlo {
		return dartIsNum(a) && bf.f == rt.toNumber(a)
	}
	// Python's boxed float compares by VALUE against ANOTHER FLOAT only. This is
	// what `is` reads (js_pybin), and in Python a float is never `is` an int even
	// when the two are ==, so the cross-type answer here is false rather than a
	// numeric comparison. pyEqual has the == rule.
	if af, isFlo := a.(*jsPyFlo); isFlo {
		bf, ok := b.(*jsPyFlo)
		// The same box is the same value even when it holds NaN (`x is x`), and
		// two different boxes are the same value when their doubles are ===
		// (CPython folds 1.0 is 1.0 to true). A float is never `is` an INT even
		// when the two are ==, so the cross-type answer is false rather than a
		// numeric comparison; pyEqual has the == rule.
		return ok && (af == bf || af.f == bf.f)
	}
	if _, isFlo := b.(*jsPyFlo); isFlo {
		return false
	}
	switch at := a.(type) {
	case jsUndefT:
		_, ok := b.(jsUndefT)
		return ok
	case jsNullT:
		_, ok := b.(jsNullT)
		return ok
	case bool:
		bt, ok := b.(bool)
		return ok && at == bt
	case float64:
		bt, ok := b.(float64)
		return ok && at == bt
	case string:
		bt, ok := b.(string)
		return ok && at == bt
	case *jsBigInt:
		bt, ok := b.(*jsBigInt)
		return ok && at.v.Cmp(bt.v) == 0
	default:
		// Objects, arrays, closures and Go natives compare by identity.
		return identityEq(a, b)
	}
}

// giIsNilMap answers "this is the zero value of a map type" - an empty dict
// carrying the __nil mark. The twins are isNilMap in languages/go-interpreter.abnf
// and goIsNilMap in languages/lib/go-rt.metajs.
func giIsNilMap(v interface{}) bool {
	o, ok := v.(*jsObject)
	if !ok {
		return false
	}
	if d, has := o.props["__dict"].(bool); !has || !d {
		return false
	}
	n, has := o.props["__nil"].(bool)
	return has && n
}

// giIsNilZero answers "this is the zero value of a map OR a slice type", i.e. what
// Go's `== nil` has to answer true for. A slice zeroes to a marked empty HEADER for
// the same reason a map zeroes to a marked empty dict - it has to READ like an empty
// one (len, range, append, print all work on a nil slice in Go) - so nil-ness is the
// __nil mark rather than the null in both cases. Set by emitNilSl in
// languages/go-to-llvm-ir.abnf; the twins are isNilZero in go-interpreter.abnf and
// goIsNilZero in languages/lib/go-rt.metajs. Go-specific: reached only from
// goValueEq, which only the go grammar calls.
func giIsNilZero(v interface{}) bool {
	if giIsNilMap(v) {
		return true
	}
	o, ok := v.(*jsObject)
	if !ok {
		return false
	}
	if n, has := o.props["__nil"].(bool); !has || !n {
		return false
	}
	cls, ok := o.props["__class"].(*jsObject)
	if !ok {
		return false
	}
	name, _ := cls.props["__name"].(string)
	return name == "$sl"
}

// goValueEq is Go's ==: structs and arrays are VALUE types and compare element by
// element, nil equals an unset slice/map/pointer, everything else is strictEq.
// Used only by the go-to-llvm-ir grammar (js_goeq / js_gone).
func (rt *jsrt) goValueEq(a, b interface{}) bool {
	// `m == nil` on the zero value of a map type, `s == nil` on the zero value of a
	// slice type. Both zero to a MARKED EMPTY container and not to nil (emitZeroTy /
	// emitNilSl in languages/go-to-llvm-ir.abnf have the argument: a nil map and a nil
	// slice both have to READ like empty ones), so nil-ness is the __nil mark rather
	// than the null.
	if isUndefOrNull(a) {
		return isUndefOrNull(b) || giIsNilZero(b)
	}
	if isUndefOrNull(b) {
		return giIsNilZero(a)
	}
	if aa, ok := a.(*jsArray); ok {
		if ba, ok2 := b.(*jsArray); ok2 {
			if len(aa.elems) != len(ba.elems) {
				return false
			}
			for i := range aa.elems {
				if !rt.goValueEq(aa.elems[i], ba.elems[i]) {
					return false
				}
			}
			return true
		}
		return false
	}
	ao, aok := a.(*jsObject)
	bo, bok := b.(*jsObject)
	if aok && bok {
		ac, ahas := ao.props["__class"].(*jsObject)
		bc, bhas := bo.props["__class"].(*jsObject)
		if ahas && bhas {
			an, _ := ac.props["__name"].(string)
			bn, _ := bc.props["__name"].(string)
			if an != bn {
				return false
			}
			af, aok2 := ac.props["__fields"].(*jsArray)
			bf, bok2 := bc.props["__fields"].(*jsArray)
			if !aok2 || !bok2 || len(af.elems) != len(bf.elems) {
				return identityEq(a, b)
			}
			for i := range af.elems {
				k := rt.toString(af.elems[i])
				if k != rt.toString(bf.elems[i]) {
					return false
				}
				av, ahad := ao.props[k]
				bv, bhad := bo.props[k]
				if !ahad {
					av = jsUndef
				}
				if !bhad {
					bv = jsUndef
				}
				if !rt.goValueEq(av, bv) {
					return false
				}
			}
			return true
		}
	}
	return rt.strictEq(a, b)
}

// ----- Go channels and goroutines (go-to-llvm-ir.abnf) -----
// The model is the interpreter's, so both engines agree: a channel is a queue with
// no blocking send, `go f()` only QUEUES f, and a receive that finds nothing runs
// the queued goroutines until one of them fills the channel. That makes the whole
// handshake deterministic on a single thread; a receive that still finds nothing
// with nothing left to run is a deadlock, exactly as in Go.
func goChanParts(v interface{}) (*jsObject, *jsArray, bool) {
	o, ok := v.(*jsObject)
	if !ok {
		return nil, nil, false
	}
	if tag, has := o.props["__chan"]; !has || tag != true {
		return nil, nil, false
	}
	buf, _ := o.props["buf"].(*jsArray)
	if buf == nil {
		return nil, nil, false
	}
	return o, buf, true
}

// goZeroOf is the zero value of a type spelled as text - the runtime twin of the
// interpreter's zeroOfType2.
func goZeroOf(ty string) interface{} {
	if strings.HasPrefix(ty, "int") || strings.HasPrefix(ty, "uint") ||
		strings.HasPrefix(ty, "byte") || strings.HasPrefix(ty, "rune") ||
		strings.HasPrefix(ty, "float") {
		return float64(0)
	}
	if strings.HasPrefix(ty, "string") {
		return ""
	}
	if strings.HasPrefix(ty, "bool") {
		return false
	}
	return jsNull
}

// goDrain runs every queued goroutine to completion, in spawn order.
func (rt *jsrt) goDrain() {
	guard := 0
	for len(rt.goQueue) > 0 && guard < 1000000 {
		guard++
		t := rt.goQueue[0]
		rt.goQueue = rt.goQueue[1:]
		rt.call(t, jsUndef, nil)
	}
}

// goChanRecv takes one value; ok is false exactly when the channel is closed and
// drained, and the value is then the element type's zero value.
func (rt *jsrt) goChanRecv(v interface{}) (interface{}, bool) {
	o, buf, ok := goChanParts(v)
	if !ok {
		rt.fail("receive from a non-channel")
	}
	if len(buf.elems) == 0 {
		rt.goDrain()
	}
	if len(buf.elems) == 0 {
		if o.props["closed"] == true {
			ty, _ := o.props["czero"].(string)
			return goZeroOf(ty), false
		}
		rt.fail("all goroutines are asleep - deadlock")
	}
	first := buf.elems[0]
	buf.elems = buf.elems[1:]
	return first, true
}

// goClassOf is the struct descriptor of a Go struct VALUE, or nil.
func goClassOf(v interface{}) *jsObject {
	if o, ok := v.(*jsObject); ok {
		if cls, ok2 := o.props["__class"].(*jsObject); ok2 {
			return cls
		}
	}
	return nil
}

// goCopyVal is Go's copy of a value type: a struct is copied field by field (a
// nested struct with it), an array element by element. Everything else - a slice, a
// map, a function, a scalar - is a reference or immutable and passes through.
func (rt *jsrt) goCopyVal(v interface{}) interface{} {
	if arr, ok := v.(*jsArray); ok {
		out := &jsArray{elems: make([]interface{}, len(arr.elems))}
		copy(out.elems, arr.elems)
		return out
	}
	o, ok := v.(*jsObject)
	if !ok || goClassOf(v) == nil {
		return v
	}
	out := newJSObject()
	for _, k := range o.keys {
		out.set(k, rt.goCopyVal(o.props[k]))
	}
	return out
}

// goPromField finds the field `key` PROMOTED from an embedded struct of o, BREADTH
// FIRST, which is Go's own rule: the shallowest depth wins and only a tie at that
// depth is ambiguous. The depth cap is the one the __class walks use (manual
// 7.9); a pointer cycle in this value model would otherwise not terminate. The twins
// are goPromField in languages/lib/go-rt.metajs and goPromField in
// languages/go-interpreter.abnf.
func (rt *jsrt) goPromField(o *jsObject, key string, maxd int) (interface{}, bool) {
	// LEVEL ONE WITHOUT ALLOCATING, which is what the one-level loop this replaced
	// did: js_gofield reaches here on every own-property miss, so the common answer
	// must cost no slice. The breadth-first walk only starts when there IS an
	// embedded struct and the first level did not have the name.
	cls1, isCls1 := o.props["__class"].(*jsObject)
	if !isCls1 {
		return nil, false
	}
	fs1, okf1 := cls1.props["__fields"].(*jsArray)
	if !okf1 {
		return nil, false
	}
	hasEmb := false
	for _, f := range fs1.elems {
		inner, isObj := o.props[rt.toString(f)].(*jsObject)
		if !isObj {
			continue
		}
		if _, embedded := inner.props["__class"]; !embedded {
			continue
		}
		if v, has := inner.props[key]; has {
			return v, true
		}
		hasEmb = true
	}
	if !hasEmb {
		return nil, false
	}
	return rt.goPromDeep(o, key, maxd)
}

// goPromDeep is the breadth-first part of goPromField, from level two down. Split
// out so the common case above allocates nothing.
func (rt *jsrt) goPromDeep(o *jsObject, key string, maxd int) (interface{}, bool) {
	level := []*jsObject{o}
	for d := 0; d < maxd && len(level) > 0; d++ {
		next := make([]*jsObject, 0, len(level))
		for _, cur := range level {
			cls, isCls := cur.props["__class"].(*jsObject)
			if !isCls {
				continue
			}
			fs, okf := cls.props["__fields"].(*jsArray)
			if !okf {
				continue
			}
			for _, f := range fs.elems {
				inner, isObj := cur.props[rt.toString(f)].(*jsObject)
				if !isObj {
					continue
				}
				if _, embedded := inner.props["__class"]; !embedded {
					continue
				}
				if v, has := inner.props[key]; has {
					return v, true
				}
				next = append(next, inner)
			}
		}
		level = next
	}
	return nil, false
}

// goMethod finds `name` on a Go struct value: on its own descriptor, or promoted
// from an embedded struct field. It answers the method, the receiver to call it
// with (a COPY for a value receiver, the struct itself for a pointer receiver) and
// whether anything was found at all.
func (rt *jsrt) goMethod(v interface{}, name string) (interface{}, interface{}, bool) {
	cls := goClassOf(v)
	if cls == nil {
		return nil, nil, false
	}
	if fn, has := cls.props[name]; has && isCallable(fn) {
		recv := v
		if cls.props["__ptr$"+name] != true {
			recv = rt.goCopyVal(v)
		}
		return fn, recv, true
	}
	// Promotion: a method of an embedded struct is a method of the outer one.
	o := v.(*jsObject)
	if fs, okf := cls.props["__fields"].(*jsArray); okf {
		for _, f := range fs.elems {
			inner := o.props[rt.toString(f)]
			if goClassOf(inner) == nil {
				continue
			}
			if fn, recv, found := rt.goMethod(inner, name); found {
				return fn, recv, true
			}
		}
	}
	return nil, nil, false
}

// identityEq compares two reference values without panicking on uncomparable types.
func identityEq(a, b interface{}) bool {
	ra := reflect.ValueOf(a)
	rb := reflect.ValueOf(b)
	if ra.Kind() != rb.Kind() {
		return false
	}
	switch ra.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Func, reflect.Chan, reflect.UnsafePointer:
		return ra.Pointer() == rb.Pointer() && ra.Type() == rb.Type()
	case reflect.Slice:
		return ra.Pointer() == rb.Pointer() && ra.Len() == rb.Len() && ra.Type() == rb.Type()
	default:
		if ra.Type() != rb.Type() {
			return false
		}
		if !ra.Type().Comparable() {
			return false
		}
		return a == b
	}
}

func (rt *jsrt) looseEq(a, b interface{}) bool {
	if av, isChar := charCode(a); isChar {
		bv, _ := charCode(b)
		return rt.looseEq(av, bv)
	}
	if bv, isChar := charCode(b); isChar {
		return rt.looseEq(a, bv)
	}
	if isUndefOrNull(a) && isUndefOrNull(b) {
		return true
	}
	if isUndefOrNull(a) || isUndefOrNull(b) {
		return false
	}
	if ab, ok := a.(bool); ok {
		return rt.looseEq(boolToNum(ab), b)
	}
	if bb, ok := b.(bool); ok {
		return rt.looseEq(a, boolToNum(bb))
	}
	an, aIsNum := a.(float64)
	bn, bIsNum := b.(float64)
	as, aIsStr := a.(string)
	bs, bIsStr := b.(string)
	switch {
	case aIsNum && bIsNum:
		return an == bn
	case aIsStr && bIsStr:
		return as == bs
	case aIsNum && bIsStr:
		return an == rt.toNumber(bs)
	case aIsStr && bIsNum:
		return rt.toNumber(as) == bn
	case (aIsNum || aIsStr) != (bIsNum || bIsStr):
		return false // Primitive against object: not needed by the subset.
	default:
		return rt.strictEq(a, b)
	}
}

func boolToNum(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

// jsCompare returns -1, 0, 1, or NaN-marker 2 for the relational operators.
func (rt *jsrt) jsCompare(a, b interface{}) int {
	as, aIsStr := a.(string)
	bs, bIsStr := b.(string)
	if aIsStr && bIsStr {
		switch {
		case as < bs:
			return -1
		case as > bs:
			return 1
		default:
			return 0
		}
	}
	if rt.hasBigInt {
		if x, y, both := bigPair(a, b); both {
			return x.Cmp(y)
		}
	}
	an := rt.toNumber(a)
	bn := rt.toNumber(b)
	if an != an || bn != bn {
		return 2 // NaN: every relation is false.
	}
	switch {
	case an < bn:
		return -1
	case an > bn:
		return 1
	default:
		return 0
	}
}

func (rt *jsrt) typeOf(v interface{}) string {
	switch v.(type) {
	case jsChar:
		return "char"
	case jsJFlo: // Java's boxed double is a number like any other.
		return "number"
	case jsGInt: // A sized integer is a number like any other (jsrtint.go).
		return "number"
	case jsDartFlo: // Dart's boxed double is a number like any other (jsrtdart.go).
		return "number"
	case *jsPyFlo: // Python's boxed float is a number like any other.
		return "number"
	case jsSym:
		return "symbol"
	case jsUndefT:
		return "undefined"
	case jsNullT:
		return "object"
	case bool:
		return "boolean"
	case float64:
		return "number"
	case string:
		return "string"
	case *jsClosure, *hostFunc, *boundMethod:
		return "function"
	case *jsBigInt:
		return "bigint"
	case *jsObject, *jsArray, *jsGenerator:
		return "object"
	default:
		if reflect.ValueOf(v).Kind() == reflect.Func {
			return "function"
		}
		return "object"
	}
}

// ----------------------------------------------------------------------------
// Scopes

func (rt *jsrt) scopeOf(h uint64) *jsScope {
	if h == 0 {
		return rt.root
	}
	sc, ok := rt.unwrap(h).(*jsScope)
	if !ok {
		rt.fail("handle %d is not a scope", h)
	}
	return sc
}

func (rt *jsrt) scopeGet(sc *jsScope, name string) interface{} {
	for s := sc; s != nil; s = s.parent {
		if v, ok := s.get(name); ok {
			// Python's `del x` leaves the slot behind holding this sentinel, so a
			// later read raises UnboundLocalError - a catchable exception - instead
			// of resolving to an enclosing binding of the same name.
			if v == pyDeleted {
				// WHICH exception depends on where the deleted name lived, and
				// CPython 3.14 draws the line at the module: a deleted GLOBAL is a
				// NameError ("name 'x' is not defined", the same text an unbound
				// name gets) and a deleted LOCAL is an UnboundLocalError. The
				// messages are 3.14's; "referenced before assignment" is the
				// pre-3.11 wording.
				if s == pyModuleScope(sc) {
					panic(&jsThrown{value: rt.pyExcInstance("NameError",
						"name '"+name+"' is not defined")})
				}
				panic(&jsThrown{value: rt.pyExcInstance("UnboundLocalError",
					"cannot access local variable '"+name+"' where it is not associated with a value")})
			}
			if rt.traced {
				rt.trVar("read", name, v)
			}
			return v
		}
	}
	if exc, ok := rt.runMissHook(&rt.missVar, name); ok {
		panic(&jsThrown{value: exc})
	}
	rt.fail("variable not defined: %s", name)
	return nil
}

// runMissHook calls one of the three recoverable-error hooks, with the slot
// cleared for the duration so that a handler which itself trips the same site
// falls back to the original abort instead of recursing. The slot is restored
// BEFORE the caller throws, because a panic would skip a deferred restore's
// effect on a later run of the same program. See the fields' declaration.
func (rt *jsrt) runMissHook(slot *interface{}, args ...interface{}) (interface{}, bool) {
	h := *slot
	if h == nil {
		return nil, false
	}
	*slot = nil
	v := rt.call(h, jsUndef, args)
	*slot = h
	if v == nil || v == jsUndef {
		return nil, false
	}
	return v, true
}

// mcallMiss answers the registered MISS SENTINEL at every memberCall site that was
// about to abort with "no such method". Nothing is cleared and nothing recurses -
// unlike the three closure hooks this one does not run program code - so it is a
// field read on a path that had already decided to end the program. See missCall.
func (rt *jsrt) mcallMiss() (interface{}, bool) {
	if rt.missCall == nil {
		return nil, false
	}
	return rt.missCall, true
}

func (rt *jsrt) scopeSet(sc *jsScope, name string, v interface{}) {
	for s := sc; s != nil; s = s.parent {
		if s.has(name) {
			s.put(name, v)
			if rt.traced {
				rt.trVar("write", name, v)
			}
			return
		}
	}
	rt.fail("assignment to undeclared variable: %s", name)
}

// scopeSetOrCreate is scopeSet without the undeclared check: a name that is
// nowhere in the chain is created in the root (global) scope. This models the
// implicit global of non-strict JavaScript (assigning to an undeclared name),
// matching the setVar of js-interpreter.abnf. Only plain `=` assignment uses
// it; compound assignment (+=, ++, ...) reads the old value first and so still
// requires a prior declaration.
func (rt *jsrt) scopeSetOrCreate(sc *jsScope, name string, v interface{}) {
	for s := sc; s != nil; s = s.parent {
		if s.has(name) {
			s.put(name, v)
			if rt.traced {
				rt.trVar("write", name, v)
			}
			return
		}
	}
	rt.root.put(name, v)
	if rt.traced {
		rt.trVar("decl", name, v)
	}
}

// typeClass returns the fixed type class of a value for MetaJS type pinning.
// undefined and null return "" (they never pin a type and are always
// assignable): null is the deliberate absence of a value, not an object.
func (rt *jsrt) typeClass(v interface{}) string {
	if _, u := v.(jsUndefT); u {
		return ""
	}
	if _, n := v.(jsNullT); n {
		return ""
	}
	return rt.typeOf(v)
}

// typedDecl declares a variable and pins its type if the value already has one.
// This is the declaration of MetaJS itself: the compiled MetaJS programs and
// (through the frozen bootstrap) every tag script declare through it.
// Declaring with the anytype marker starts the variable as undefined and
// exempts it from pinning ("*") until a redeclaration says otherwise.
func (rt *jsrt) typedDecl(sc *jsScope, name string, v interface{}) {
	if isAnytype(v) {
		sc.put(name, jsUndef)
		sc.pinType(name, "*")
		if rt.traced {
			rt.trVar("decl", name, jsUndef)
		}
		return
	}
	sc.put(name, v)
	sc.pinType(name, rt.typeClass(v)) // "" means a redeclaration starts untyped again.
	if rt.traced {
		rt.trVar("decl", name, v)
	}
}

// typedSet assigns like scopeSet but refuses to change a pinned type.
// Assigning undefined or null is allowed and keeps the pinned type; a
// variable declared as anytype ("*") accepts every class and never pins.
func (rt *jsrt) typedSet(sc *jsScope, name string, v interface{}) {
	if isAnytype(v) {
		rt.fail("MetaJS: anytype can only initialize a declaration")
	}
	for s := sc; s != nil; s = s.parent {
		if s.has(name) {
			if tc := rt.typeClass(v); tc != "" {
				old := s.typeClassOf(name)
				if old == "*" {
					// anytype variable: no check, and nothing re-pins it.
				} else if old != "" && old != tc {
					rt.fail("MetaJS: variable '%s' has type %s and cannot hold a %s", name, old, tc)
				} else {
					s.pinType(name, tc)
				}
			}
			s.put(name, v)
			if rt.traced {
				rt.trVar("write", name, v)
			}
			return
		}
	}
	rt.fail("assignment to undeclared variable: %s", name)
}

// ----------------------------------------------------------------------------
// The STATIC NAMESPACE of a js/ts class
//
// `class C { static m(){} m(){} }` is two different members in JavaScript - one on
// the constructor, one on the prototype - so they cannot share the descriptor's
// own property map. The js/ts emitters and interpreters put every static
// member - methods, fields and accessor boxes - in a plain object stored on the
// descriptor under `__statics`, and leave instance members where they are.
// Instance dispatch is unaffected, which is why js_get, js_set and js_mcall (shared
// by all languages, and the hottest externs here) do not look at it: only the js/ts
// externs js_jsvget / js_jsmget / js_jsmset / js_jsmcall and the three super
// primitives below read the table, and for a language that emits no __statics
// every one of them falls through to the plain code.
//
// Twins: jsStaticFind / jvStaticFind in languages/lib/js-rt.metajs and
// classStaticFind / staticsOf in the two -interpreter grammars.

// isClassDesc reports whether v is a class DESCRIPTOR ({__isclass: true, ...}) as
// opposed to an instance of one. It is what tells the static side of a `super`
// lookup from the instance side: only a static method's receiver is a class.
func isClassDesc(v interface{}) bool {
	o, ok := v.(*jsObject)
	if !ok {
		return false
	}
	isc, has := o.props["__isclass"]
	return has && isc == true
}

// jsStaticsOf answers a descriptor's own static table, creating it when create is
// set (a runtime `C.zz = 1` is a static write like any other).
func jsStaticsOf(o *jsObject, create bool) *jsObject {
	if t, ok := o.props["__statics"].(*jsObject); ok {
		return t
	}
	if !create {
		return nil
	}
	t := newJSObject()
	o.set("__statics", t)
	return t
}

// jsStaticsFind walks the descriptor chain from start along __super, looking in each
// link's static table. It answers the RAW slot, so an accessor box reaches its caller
// and can be invoked with the right receiver, plus the descriptor that owned it.
// Statics ARE inherited in JavaScript - `class D extends C {}` then `D.s()` walks the
// class object's own prototype chain - which is what the walk is for.
func jsStaticsFind(start interface{}, name string) (interface{}, *jsObject, bool) {
	guard := 0
	for cls := start; cls != nil && guard < 64; guard++ {
		clsObj, ok := cls.(*jsObject)
		if !ok {
			break
		}
		if t := jsStaticsOf(clsObj, false); t != nil {
			if v, found := t.props[name]; found {
				return v, clsObj, true
			}
		}
		cls = clsObj.props["__super"]
	}
	return nil, nil, false
}

// ----------------------------------------------------------------------------
// Property access (including the Go bridge)

// isCallable reports whether a value can be invoked.
func isCallable(v interface{}) bool {
	switch v.(type) {
	case *jsClosure, *hostFunc, *boundMethod:
		return true
	}
	return reflect.ValueOf(v).Kind() == reflect.Func
}

// maybeNumeric reports whether key could convert to a number at all. Array and
// string members that are not a known method go through toNumber to see whether
// they are an index, and a failing ParseFloat costs a formatted error - so the
// ordinary property name is rejected by its first byte instead. Only a digit,
// a sign, a dot, leading space or the start of Infinity/NaN can begin a number.
func maybeNumeric(key interface{}) bool {
	s, isStr := key.(string)
	if !isStr {
		return true
	}
	if s == "" {
		return true // "" converts to 0, like it always did.
	}
	c := s[0]
	return c <= ' ' || (c >= '0' && c <= '9') || c == '+' || c == '-' || c == '.' ||
		c == 'I' || c == 'i' || c == 'N' || c == 'n'
}

// atoiName is strconv.Atoi for a property name, without the error allocation
// for the names that obviously are not a number.
func atoiName(name string) (int, error) {
	for i := 0; i < len(name); i++ {
		c := name[i]
		if (c < '0' || c > '9') && !(i == 0 && (c == '+' || c == '-') && len(name) > 1) {
			return 0, errNotAnIndex
		}
	}
	if name == "" {
		return 0, errNotAnIndex
	}
	return strconv.Atoi(name)
}

var errNotAnIndex = errors.New("not an index")

// dictMemberHit / dictMemberIdx answer whether a dict handle holds `key`, using
// Kotlin's value equality so a boxed Long key and a plain Int key are the same key.
// ADDITIVE: they are consulted only when the receiver is a {__dict, keys, vals}
// handle and the name is not already one of its own properties.
func (rt *jsrt) dictMemberHit(keys *jsArray, key interface{}) bool {
	return rt.dictMemberIdx(keys, key) >= 0
}

func (rt *jsrt) dictMemberIdx(keys *jsArray, key interface{}) int {
	if s, isStr := key.(string); isStr && (s == "keys" || s == "vals" || s == "__dict") {
		return -1
	}
	return rt.ktMapFind(keys, key)
}

func (rt *jsrt) getMember(obj interface{}, key interface{}) interface{} {
	if isUndefOrNull(obj) {
		if exc, ok := rt.runMissHook(&rt.missMem, key, obj); ok {
			panic(&jsThrown{value: exc})
		}
		rt.fail("member '%s' of %s", rt.toString(key), rt.toString(obj))
	}
	// Function values understand apply and call like in JS.
	if ks, isStr := key.(string); isStr && (ks == "apply" || ks == "call") && isCallable(obj) {
		return &boundMethod{recv: obj, name: ks}
	}
	// Kotlin only: Char.code is a PROPERTY in Kotlin, not a method, so it
	// arrives here rather than at js_ktsmcall. kotlin-interpreter.abnf answers
	// the same name in kGetField.
	if c, isChar := obj.(jsChar); isChar {
		if ks, isStr := key.(string); isStr && ks == "code" {
			return float64(c.code)
		}
	}
	// ADDITIVE, JavaScript/TypeScript only: then/catch/finally READ AS A VALUE.
	// Promise method dispatch is table-based (promMethod, reached from js_jsmcall),
	// so a promise carries no own slot of those names; `const t = p.then` must
	// still answer a function. promIs is true only for an
	// object the js promise machinery built (a '__prom' slot), so no other language
	// can reach this. The native build's twin arm is in abnf/jsrtjsprint.go's
	// js_jsmget, layer 2's in languages/lib/js-rt.metajs's js_jsmget, and the
	// interpreters' in their own getMember.
	if ks, isStr := key.(string); isStr && (ks == "then" || ks == "catch" || ks == "finally") && promIs(obj) {
		po := obj.(*jsObject)
		return jsHostFunc(ks, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			v, _ := rt.promMethod(po, ks, args)
			return v
		})
	}
	switch o := obj.(type) {
	case *jsObject:
		name := rt.toString(key)
		// ADDITIVE, Kotlin only: a MAP is the shared {__dict, keys, vals} handle and
		// kotlin.Map declares size / entries as properties. Neither name is in the
		// props map, so no other dict user (Python, Go) can observe this.
		if keys, vals, isDict := dictParts(o); isDict && (name == "size" || name == "length" ||
			name == "entries" || name == "values" || rt.dictMemberHit(keys, key)) {
			if name == "size" || name == "length" {
				return float64(len(keys.elems))
			}
			if name == "values" {
				return &jsArray{elems: append([]interface{}{}, vals.elems...)}
			}
			// m[k] / m.k for a key the map actually holds. A MISS is left to the
			// existing path (undefined), so no non-Kotlin dict user changes.
			if i := rt.dictMemberIdx(keys, key); i >= 0 {
				return vals.elems[i]
			}
			out := &jsArray{}
			for i := range keys.elems {
				e := newJSObject()
				e.set("key", keys.elems[i])
				e.set("value", vals.elems[i])
				e.set("first", keys.elems[i])
				e.set("second", vals.elems[i])
				out.elems = append(out.elems, e)
			}
			return out
		}
		v, ok := o.props[name]
		if rt.accessorCount == 0 { // No accessor exists at all: the original path.
			if ok {
				return v
			}
			return jsUndef
		}
		if ok {
			if acc, isAcc := v.(*jsAccessor); isAcc {
				if acc.get == nil {
					return jsUndef
				}
				return rt.call(acc.get, o, nil)
			}
			return v
		}
		// An instance does not carry its class's accessors; they live on the
		// __class descriptor (and its __super chain), like the methods do.
		if acc := rt.findClassAccessor(o, name); acc != nil {
			if acc.get == nil {
				return jsUndef
			}
			return rt.call(acc.get, o, nil)
		}
		return jsUndef
	case *jsArray:
		if ks, isStr := key.(string); isStr {
			switch ks {
			case "length":
				return float64(len(o.elems))
			case "push", "pop", "shift", "unshift", "reverse", "slice", "indexOf", "join", "concat":
				return &boundMethod{recv: o, name: ks}
			}
		}
		if !maybeNumeric(key) { // A property name like 'mname' is not an index.
			return jsUndef
		}
		idx := rt.toNumber(key)
		if idx == math.Trunc(idx) && idx >= 0 && int(idx) < len(o.elems) {
			return o.elems[int(idx)]
		}
		return jsUndef
	case *jsGenerator:
		switch rt.toString(key) {
		case "next":
			return jsHostFunc("next", func(rt *jsrt, this uint64, args []interface{}) interface{} {
				return o.step(rt, argAt(args, 0))
			})
		case "return":
			return jsHostFunc("return", func(rt *jsrt, this uint64, args []interface{}) interface{} {
				return o.finish(rt, argAt(args, 0))
			})
		case "throw":
			return jsHostFunc("throw", func(rt *jsrt, this uint64, args []interface{}) interface{} {
				return o.throwInto(rt, argAt(args, 0))
			})
		// The INSPECTION half of the iterator protocol, which JavaScript's
		// {value, done} record folds into next() but most other languages expose as
		// members of their own: PHP's Generator (current/key/valid/send/rewind/
		// getReturn), Python's generator.send, C#'s enumerator. They answer from the
		// last handshake instead of performing one, so reading a generator twice does
		// not advance it - and priming on first read is what makes current() work on a
		// generator nobody has stepped yet.
		case "current":
			return jsHostFunc("current", func(rt *jsrt, this uint64, args []interface{}) interface{} {
				o.prime(rt)
				if o.done {
					return jsNull
				}
				return o.lastValue
			})
		case "key":
			return jsHostFunc("key", func(rt *jsrt, this uint64, args []interface{}) interface{} {
				o.prime(rt)
				if o.done {
					return jsNull
				}
				return o.lastKey
			})
		case "valid":
			return jsHostFunc("valid", func(rt *jsrt, this uint64, args []interface{}) interface{} {
				o.prime(rt)
				return !o.done
			})
		case "rewind":
			return jsHostFunc("rewind", func(rt *jsrt, this uint64, args []interface{}) interface{} {
				o.prime(rt)
				return jsNull
			})
		// send(v) resumes the body with v and answers the value it yields NEXT, which
		// is the shape PHP and Python share; the {value, done} record is next()'s.
		case "send":
			return jsHostFunc("send", func(rt *jsrt, this uint64, args []interface{}) interface{} {
				if !o.started {
					o.step(rt, jsUndef) // a send() to a fresh generator primes it first
				}
				o.step(rt, argAt(args, 0))
				if o.done {
					return jsNull
				}
				return o.lastValue
			})
		case "getReturn":
			return jsHostFunc("getReturn", func(rt *jsrt, this uint64, args []interface{}) interface{} {
				if o.retValue == nil {
					return jsNull
				}
				return o.retValue
			})
		}
		return jsUndef
	case string:
		if ks, isStr := key.(string); isStr {
			switch ks {
			case "length":
				return float64(rt.strLen(o))
			case "charCodeAt", "charAt", "indexOf", "replace", "slice", "substring", "split",
				"toUpperCase", "toLowerCase", "trim":
				return &boundMethod{recv: o, name: ks}
			}
		}
		if !maybeNumeric(key) {
			return jsUndef
		}
		idx := rt.toNumber(key)
		if idx == math.Trunc(idx) && idx >= 0 {
			if ch := rt.strAt(o, int(idx)); ch != "" {
				return ch
			}
		}
		return jsUndef
	default:
		return rt.getGoMember(obj, rt.toString(key))
	}
}

// getGoMember resolves a property on an arbitrary Go value: methods (value or
// pointer receiver), exported struct fields, map entries, and slice indexing /
// length. This is the read side of the goja-like bridge.
func (rt *jsrt) getGoMember(obj interface{}, name string) interface{} {
	rv := reflect.ValueOf(obj)

	if m := rv.MethodByName(name); m.IsValid() {
		return m.Interface() // A bound method value; called via reflectCall.
	}

	deref := rv
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			rt.fail("member '%s' of nil %s", name, rv.Type())
		}
		deref = rv.Elem()
		if m := deref.MethodByName(name); m.IsValid() {
			return m.Interface()
		}
	}

	switch deref.Kind() {
	case reflect.Struct:
		if f := deref.FieldByName(name); f.IsValid() && f.CanInterface() {
			return rt.importGoValue(f.Interface())
		}
	case reflect.Map:
		mv := deref.MapIndex(reflect.ValueOf(name))
		if mv.IsValid() {
			return rt.importGoValue(mv.Interface())
		}
		return jsUndef
	case reflect.Slice, reflect.Array:
		if name == "length" {
			return float64(deref.Len())
		}
		if idx, err := atoiName(name); err == nil {
			if idx >= 0 && idx < deref.Len() {
				return rt.importGoValue(deref.Index(idx).Interface())
			}
			return jsUndef
		}
	}
	return jsUndef
}

// importGoValue normalizes a Go value that enters the JS world: numeric kinds
// become JS numbers, typed nil pointers become null, everything else stays native.
func (rt *jsrt) importGoValue(v interface{}) interface{} {
	if v == nil {
		return jsUndef
	}
	switch t := v.(type) {
	case bool, string, float64, *jsObject, *jsArray, *jsClosure, *hostFunc, *boundMethod, jsUndefT, jsNullT:
		return v
	case *ltrText:
		return t.String() // The lazy ltr.in accumulator materializes on read.
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(rv.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(rv.Uint())
	case reflect.Float32:
		return rv.Float()
	case reflect.String:
		return rv.String()
	case reflect.Ptr, reflect.Map, reflect.Slice, reflect.Func, reflect.Interface:
		if rv.IsNil() {
			return jsNull
		}
		return v
	default:
		return v
	}
}

func (rt *jsrt) setMember(obj interface{}, key interface{}, val interface{}) {
	if isUndefOrNull(obj) {
		if exc, ok := rt.runMissHook(&rt.missSet, key, obj); ok {
			panic(&jsThrown{value: exc})
		}
		rt.fail("member assignment '%s' on %s", rt.toString(key), rt.toString(obj))
	}
	switch o := obj.(type) {
	case *jsObject:
		name := rt.toString(key)
		if rt.accessorCount > 0 { // Skipped entirely while no accessor exists.
			if v, ok := o.props[name]; ok {
				if acc, isAcc := v.(*jsAccessor); isAcc {
					if acc.set != nil {
						rt.call(acc.set, o, []interface{}{val})
					}
					return
				}
			} else if acc := rt.findClassAccessor(o, name); acc != nil {
				if acc.set != nil {
					rt.call(acc.set, o, []interface{}{val})
				}
				return
			}
		}
		o.set(name, val)
	case *jsArray:
		// `arr.length = n` TRUNCATES (or pads with undefined) in JavaScript.
		if s, isStr := key.(string); isStr && s == "length" {
			n := rt.toNumber(val)
			if n != math.Trunc(n) || n < 0 {
				rt.fail("invalid array length %s", rt.toString(val))
			}
			ln := int(n)
			o.dropIdx()
			for len(o.elems) > ln {
				o.elems = o.elems[:len(o.elems)-1]
			}
			for len(o.elems) < ln {
				o.elems = append(o.elems, jsUndef)
			}
			return
		}
		if !maybeNumeric(key) {
			rt.fail("invalid array index %s", rt.toString(key))
		}
		idx := rt.toNumber(key)
		if idx != math.Trunc(idx) || idx < 0 {
			rt.fail("invalid array index %s", rt.toString(key))
		}
		i := int(idx)
		o.dropIdx()
		for len(o.elems) <= i {
			o.elems = append(o.elems, jsUndef)
		}
		o.elems[i] = val
	default:
		rt.setGoMember(obj, rt.toString(key), val)
	}
}

// setGoMember is the write side of the bridge: struct fields (through a
// pointer) and map entries. The value is converted to the field/element type,
// so e.g. 'tag.Childs = someJSArray' fills a *r.Rules field.
func (rt *jsrt) setGoMember(obj interface{}, name string, val interface{}) {
	rv := reflect.ValueOf(obj)
	if rv.Kind() == reflect.Ptr && rv.Elem().Kind() == reflect.Struct {
		f := rv.Elem().FieldByName(name)
		if f.IsValid() && f.CanSet() {
			f.Set(rt.convertToType(val, f.Type()))
			return
		}
		rt.fail("cannot set field '%s' on %s", name, rv.Type())
	}
	if rv.Kind() == reflect.Map {
		rv.SetMapIndex(reflect.ValueOf(name), rt.convertToType(val, rv.Type().Elem()))
		return
	}
	// THE LAST ARM IS NOT THE GO BRIDGE'S - it is setMember's `default:`, reached by
	// any JS PRIMITIVE (`var x = 5; x.foo = 1`). The wording is the C floor's
	// (die3 in languages/lib/runtime.c), the same VALUE-not-TYPE shape as the
	// undefined/null arm above and the sibling get diagnostic. The two genuinely
	// Go-bridge failures above keep rv.Type(), because there the Go type IS the
	// thing the user has to fix.
	rt.fail("member assignment '%s' on %s", name, rt.toString(obj))
}

// ----------------------------------------------------------------------------
// Calls

func (rt *jsrt) call(callee interface{}, this interface{}, args []interface{}) interface{} {
	return rt.callH(callee, this, args, 0)
}

// callH is call with the handle of the array the arguments came in, if the
// caller had one (js_call and friends build one per call site). A compiled
// callee receives its arguments AS an array handle, so that array can be
// handed straight through instead of boxing a second one - the call sites
// build a fresh array per execution and drop it right after the call, so
// nothing can observe the sharing. 0 means "no array handle", box one.
func (rt *jsrt) callH(callee interface{}, this interface{}, args []interface{}, argsH uint64) interface{} {
	if !rt.traced {
		return rt.callInner(callee, this, args, argsH)
	}
	traceEmit(&TraceEvent{Ev: "call", Depth: rt.traceDepth, Line: lineOfPos(rt.curPos), Name: rt.calleeName(callee)})
	rt.traceDepth++
	savedPos := rt.curPos
	completed := false
	// A js_throw panic unwinding through a traced call must still restore the
	// depth and curPos and balance the call event, otherwise every event after
	// a caught exception carries an inflated depth and the throw-site line.
	defer func() {
		if completed {
			return
		}
		rt.curPos = savedPos
		rt.traceDepth--
		traceEmit(&TraceEvent{Ev: "ret", Depth: rt.traceDepth, Line: lineOfPos(rt.curPos), Val: "throw!"})
	}()
	ret := rt.callInner(callee, this, args, argsH)
	completed = true
	rt.curPos = savedPos // The caller's statement continues after the call.
	rt.traceDepth--
	traceEmit(&TraceEvent{Ev: "ret", Depth: rt.traceDepth, Line: lineOfPos(rt.curPos), Val: rt.traceVal(ret)})
	return ret
}

func (rt *jsrt) callInner(callee interface{}, this interface{}, args []interface{}, argsH uint64) interface{} {
	switch c := callee.(type) {
	case *jsClosure:
		if argsH == 0 {
			argsH = rt.wrap(&jsArray{elems: args})
		}
		// The two-element argument slice is reused: machine.call copies the
		// parameters into the callee's frame before anything can run, so a
		// nested call may overwrite the buffer afterwards.
		rt.callBuf[0], rt.callBuf[1] = c.env, argsH
		if !rt.trackThis {
			return rt.unwrap(c.ma.call(c.fn, rt.callBuf[:]))
		}
		rt.thisStack = append(rt.thisStack, this)
		nt := rt.pendingNewTarget
		rt.pendingNewTarget = nil
		rt.newTargetStack = append(rt.newTargetStack, nt)
		ret := c.ma.call(c.fn, rt.callBuf[:])
		rt.thisStack = rt.thisStack[:len(rt.thisStack)-1]
		rt.newTargetStack = rt.newTargetStack[:len(rt.newTargetStack)-1]
		return rt.unwrap(ret)
	case *hostFunc:
		return c.fn(rt, rt.wrap(this), args)
	case *boundMethod:
		return rt.builtinMethod(c, args)
	default:
		rv := reflect.ValueOf(callee)
		if rv.Kind() == reflect.Func {
			return rt.reflectCall(rv, args)
		}
		if exc, ok := rt.runMissHook(&rt.missFn, callee); ok {
			panic(&jsThrown{value: exc})
		}
		// The wording is the floor's die2 in languages/lib/runtime.c, byte for byte:
		// this message is reachable from a metajs program and the two engines have
		// to agree on it.
		rt.fail("call of a non function value: %s", rt.toString(callee))
		return nil
	}
}

// reflectCall invokes an arbitrary Go function with converted arguments. This
// covers the whole llvm.*/abnf.*/c.* API including variadic functions.
func (rt *jsrt) reflectCall(fn reflect.Value, args []interface{}) interface{} {
	t := fn.Type()
	numIn := t.NumIn()
	var in []reflect.Value
	if t.IsVariadic() {
		fixed := numIn - 1
		for i := 0; i < fixed; i++ {
			in = append(in, rt.convertArgToType(args, i, t.In(i)))
		}
		elemT := t.In(numIn - 1).Elem()
		// Like goja: a single array passed at the variadic position spreads
		// into the variadic parameter (m.NewFunc(name, i64, params) style),
		// unless the array itself is a valid element.
		if len(args) == fixed+1 {
			if arr, ok := args[fixed].(*jsArray); ok && elemT.Kind() != reflect.Slice {
				for _, e := range arr.elems {
					in = append(in, rt.convertToType(e, elemT))
				}
				out := fn.Call(in)
				return rt.finishReflectCall(out)
			}
		}
		for i := fixed; i < len(args); i++ {
			in = append(in, rt.convertToType(args[i], elemT))
		}
	} else {
		for i := 0; i < numIn; i++ {
			in = append(in, rt.convertArgToType(args, i, t.In(i)))
		}
	}
	out := fn.Call(in)
	return rt.finishReflectCall(out)
}

func (rt *jsrt) finishReflectCall(out []reflect.Value) interface{} {
	// A trailing error return behaves like in goja: nil is dropped, non-nil escalates.
	if len(out) > 0 {
		last := out[len(out)-1]
		if last.Type() == reflect.TypeOf((*error)(nil)).Elem() {
			if !last.IsNil() {
				rt.fail("%v", last.Interface())
			}
			out = out[:len(out)-1]
		}
	}
	switch len(out) {
	case 0:
		return jsUndef
	case 1:
		return rt.importGoValue(out[0].Interface())
	default:
		arr := &jsArray{}
		for _, o := range out {
			arr.elems = append(arr.elems, rt.importGoValue(o.Interface()))
		}
		return arr
	}
}

func (rt *jsrt) convertArgToType(args []interface{}, i int, t reflect.Type) reflect.Value {
	var v interface{} = jsUndef
	if i < len(args) {
		v = args[i]
	}
	return rt.convertToType(v, t)
}

var interfaceType = reflect.TypeOf((*interface{})(nil)).Elem()

// convertToType converts a JS value to a concrete Go type for the bridge.
func (rt *jsrt) convertToType(v interface{}, t reflect.Type) reflect.Value {
	// interface{}: natural conversion.
	if t == interfaceType {
		n := rt.toGoNatural(v)
		if n == nil {
			return reflect.Zero(t)
		}
		return reflect.ValueOf(n).Convert(t)
	}

	if isUndefOrNull(v) {
		return reflect.Zero(t)
	}

	// A native value that already fits (or converts, e.g. float64 -> enum type).
	rv := reflect.ValueOf(v)
	if rv.Type().AssignableTo(t) {
		return rv
	}

	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return reflect.ValueOf(rt.toNumber(v)).Convert(t)
	case reflect.String:
		return reflect.ValueOf(rt.toString(v))
	case reflect.Bool:
		return reflect.ValueOf(rt.truthy(v))
	case reflect.Slice:
		if arr, ok := v.(*jsArray); ok {
			out := reflect.MakeSlice(t, len(arr.elems), len(arr.elems))
			for i, e := range arr.elems {
				out.Index(i).Set(rt.convertToType(e, t.Elem()))
			}
			return out
		}
		// A native slice (also behind a pointer, like ltr.stack) converts elementwise.
		if src := derefSliceValue(rv); src.IsValid() {
			out := reflect.MakeSlice(t, src.Len(), src.Len())
			for i := 0; i < src.Len(); i++ {
				out.Index(i).Set(rt.convertToType(src.Index(i).Interface(), t.Elem()))
			}
			return out
		}
	case reflect.Ptr:
		// A pointer to a slice (like *r.Rules) is filled from a JS array or another slice.
		if t.Elem().Kind() == reflect.Slice {
			if _, ok := v.(*jsArray); ok {
				out := reflect.New(t.Elem())
				out.Elem().Set(rt.convertToType(v, t.Elem()))
				return out
			}
			if src := derefSliceValue(rv); src.IsValid() && rv.Type() != t {
				out := reflect.New(t.Elem())
				out.Elem().Set(rt.convertToType(src.Interface(), t.Elem()))
				return out
			}
		}
	case reflect.Map:
		if o, ok := v.(*jsObject); ok {
			out := reflect.MakeMap(t)
			for _, k := range o.keys {
				out.SetMapIndex(reflect.ValueOf(k), rt.convertToType(o.props[k], t.Elem()))
			}
			return out
		}
	case reflect.Interface:
		if rv.Type().Implements(t) {
			return rv
		}
	}

	if rv.Type().ConvertibleTo(t) {
		return rv.Convert(t)
	}
	rt.fail("cannot convert %s (%T) to %s", rt.toString(v), v, t)
	return reflect.Value{}
}

// dictParts returns the keys and vals arrays of a Python dict handle: a jsObject
// tagged with __dict whose two parallel arrays keep the entries in insertion
// order (the object's Go property map cannot).
func dictParts(v interface{}) (*jsArray, *jsArray, bool) {
	o, ok := v.(*jsObject)
	if !ok {
		return nil, nil, false
	}
	if tag, ok := o.props["__dict"]; !ok || tag != true {
		return nil, nil, false
	}
	keys, _ := o.props["keys"].(*jsArray)
	vals, _ := o.props["vals"].(*jsArray)
	if keys == nil || vals == nil {
		return nil, nil, false
	}
	return keys, vals, true
}

// dictFind returns the position of key k in the keys array, or -1.
//
// The array keeps the insertion order, which is part of the value model, so the
// lookup goes through an index built alongside it (dictIdx). The index is a
// pure accelerator: it is dropped by every array mutation that is not a dict
// insertion (see dropIdx), rebuilt when the length no longer matches the one it
// was built for, and a hit is confirmed against the array before it is used -
// so a script that writes d.keys behind our back cannot produce a wrong answer,
// only a rebuild. Keys that cannot be a Go map key (objects, closures) are left
// out of the index, and a miss then falls back to the scan.
func (rt *jsrt) dictFind(keys *jsArray, k interface{}) int {
	if !dictKeyable(k) { // Also keeps an unhashable Go value out of the map.
		return rt.dictScan(keys, k)
	}
	if keys.dictIdx == nil || keys.dictIdxLen != len(keys.elems) {
		rt.reindex(keys)
	}
	if i, ok := keys.dictIdx[k]; ok {
		if i < len(keys.elems) && rt.strictEq(keys.elems[i], k) {
			return i
		}
		rt.reindex(keys) // An element changed in place: rebuild and answer again.
		if i, ok := keys.dictIdx[k]; ok && rt.strictEq(keys.elems[i], k) {
			return i
		}
		return -1
	}
	if keys.dictIdxAll {
		return -1
	}
	return rt.dictScan(keys, k)
}

// pyDictFind is dictFind under PYTHON's key rule, which is not JavaScript's and
// not any other target language's: `bool` subclasses `int`, so `True` and `1` and
// `1.0` are ONE key and `False`, `0` and `-0.0` are one key.
//
//	d = {}; d[True] = "t"; d[1] = "one"     ->  CPython 3.14: ONE entry, d[True]
//	                                            is "one", and list(d) is [True] -
//	                                            the FIRST key object is the one
//	                                            that stays, only the value is
//	                                            overwritten.
//
// That last part is why this is a lookup alias and not a normalization at
// insertion: normalizing would store 1 and print `1` where CPython prints `True`.
// dictFind's index accelerator is keyed by the Go value, so `true` and
// `float64(1)` sit in it separately and both have to be asked for; the alias is
// tried only after an exact miss, so the fast path is unchanged.
//
// The pyLang guard is load bearing. js_pyget and js_pyset are the dict subscript
// of ten other grammars, and `mapOf(true to "a", 1 to "b")` is TWO entries in all
// of them. Scoping it to the dict object instead would need a python-only dict
// constructor in languages/python-to-llvm-ir.abnf.
func (rt *jsrt) pyDictFind(keys *jsArray, k interface{}) int {
	// Every dict lookup, insertion, membership test and removal in this engine
	// goes through here, so the unhashable-key refusal needs no other dict site.
	// The type assertion is inline: almost every key is a string or a number and
	// this is the hottest function on the dict path.
	if rt.pyLang {
		if _, isObj := k.(*jsObject); isObj {
			rt.pyHashCheck(k, false)
		}
	}
	if i := rt.dictFind(keys, k); i >= 0 {
		return i
	}
	if !rt.pyLang {
		return -1
	}
	// Tried TWICE: a float box's alias is the plain number and the plain
	// number's is the bool, so {True: "t"}[1.0] takes two steps.
	alias, ok := pyKeyAlias(k)
	if !ok {
		return -1
	}
	if i := rt.dictFind(keys, alias); i >= 0 {
		return i
	}
	if alias2, ok2 := pyKeyAlias(alias); ok2 {
		return rt.dictFind(keys, alias2)
	}
	return -1
}

// pyKeyAlias answers the OTHER spelling of a key that Python considers equal to
// this one: a bool is the int it IS, an int that is 0 or 1 is also the bool, and
// a float box (jsPyFlo) is the plain number it holds. -0.0 == 0 in Go, so the
// negative zero falls into the False arm exactly as CPython puts it there.
func pyKeyAlias(k interface{}) (interface{}, bool) {
	switch v := k.(type) {
	case *jsPyFlo:
		return v.f, true
	case bool:
		if v {
			return float64(1), true
		}
		return float64(0), true
	case float64:
		if v == 1 {
			return true, true
		}
		if v == 0 {
			return false, true
		}
	}
	return nil, false
}

func (rt *jsrt) dictScan(keys *jsArray, k interface{}) int {
	for i, e := range keys.elems {
		if rt.dictKeyEq(e, k) {
			return i
		}
	}
	return -1
}

// dictKeyEq is the scan's confirmation step: === everywhere, except that under
// PYTHON a float box keys with the int of the same value (1.0 and 1 are one
// key). The alias chain in pyDictFind cannot cover that direction, because a
// float box is not a Go map key and so is never IN the index - a dict that holds
// one has dictIdxAll false and every lookup in it already comes through here.
func (rt *jsrt) dictKeyEq(a, b interface{}) bool {
	if rt.pyLang {
		if pyIsFlo(a) || pyIsFlo(b) {
			return rt.pyNumEq(a, b)
		}
		// PYTHON: a TUPLE key hashes by its CONTENTS, so `{(1,2): "x"}[(1,2)]` is
		// a hit off two different objects - and a tuple IS an array in this value
		// model. So the comparison is Python's == (as on the set side: pySetHas ->
		// pyEqual). Neither operand is dictKeyable, so every such lookup already
		// arrives here through the scan.
		if aa, isA := a.(*jsArray); isA {
			if bb, isB := b.(*jsArray); isB {
				return rt.pyEqual(aa, bb)
			}
		}
	}
	// GO: a struct or array key compares BY VALUE (goValueEq), because both are
	// value types and Go's == on them is element-by-element. Neither is
	// dictKeyable, so every lookup with one already arrives here; the identity
	// answer made m[Pair{1,2}] miss an entry stored under an equal Pair. A slice
	// is not a legal Go map key and is left on identity.
	if rt.goLang && (goIsValueKey(a) || goIsValueKey(b)) {
		return rt.goValueEq(a, b)
	}
	return rt.strictEq(a, b)
}

// goIsValueKey: a Go map key whose == is VALUE equality - an array, or a struct
// instance (anything with a class descriptor that is not the slice header's).
// The twins are goKeyIsVal in languages/lib/go-rt.metajs and in
// languages/go-interpreter.abnf.
func goIsValueKey(v interface{}) bool {
	if _, ok := v.(*jsArray); ok {
		return true
	}
	cls := goClassOf(v)
	if cls == nil {
		return false
	}
	name, _ := cls.props["__name"].(string)
	return name != "$sl"
}

// reindex rebuilds the key index. Only the first position of a key is indexed,
// so a (malformed) array with duplicates still answers like the scan did.
func (rt *jsrt) reindex(keys *jsArray) {
	idx := make(map[interface{}]int, len(keys.elems))
	all := true
	for i, e := range keys.elems {
		if !dictKeyable(e) {
			all = false
			continue
		}
		if _, dup := idx[e]; !dup {
			idx[e] = i
		}
	}
	keys.dictIdx, keys.dictIdxLen, keys.dictIdxAll = idx, len(keys.elems), all
}

// dictKeyable reports whether v can be a Go map key with exactly the ===
// semantics of strictEq. Everything else compares by identity and stays in the
// scan (a Go map would either panic on it or use the wrong equality).
func dictKeyable(v interface{}) bool {
	switch v.(type) {
	case string, float64, bool, jsUndefT, jsNullT, jsSym:
		return true
	}
	return false
}

// dictAppend adds one entry to a dict's parallel arrays, keeping the index.
func dictAppend(keys, vals *jsArray, k, v interface{}) {
	keys.elems = append(keys.elems, k)
	vals.elems = append(vals.elems, v)
	if keys.dictIdx != nil {
		if dictKeyable(k) {
			if _, dup := keys.dictIdx[k]; !dup {
				keys.dictIdx[k] = len(keys.elems) - 1
			}
		} else {
			keys.dictIdxAll = false
		}
		keys.dictIdxLen = len(keys.elems)
	}
}

// dropIdx invalidates the key index of an array that is about to be mutated by
// anything other than a dict insertion.
func (a *jsArray) dropIdx() {
	a.dictIdx = nil
}

// pySliceRange resolves Python slice bounds against a length: undefined ends
// mean the whole side, negatives wrap, everything clamps into [0, n].
func (rt *jsrt) pySliceRange(lo, hi interface{}, n int) (int, int) {
	clamp := func(v interface{}, dflt int) int {
		if _, undef := v.(jsUndefT); undef {
			return dflt
		}
		i := int(rt.toNumber(v))
		if i < 0 {
			i += n
		}
		if i < 0 {
			i = 0
		}
		if i > n {
			i = n
		}
		return i
	}
	from := clamp(lo, 0)
	to := clamp(hi, n)
	if to < from {
		to = from
	}
	return from, to
}

// pyReprEsc answers whether a code point must be ESCAPED by repr(). "Printable"
// is Unicode's: NOT in the categories Cc Cf Cs Co Cn Zl Zp Zs, with U+0020
// itself excepted. The Cn (UNASSIGNED) half of that needs the Unicode database,
// which nothing in this project carries, so this knows the CLOSED sets instead -
// C0/C1, the two Latin-1 strays U+00A0 and U+00AD, the separators, the format
// characters, the surrogates and the private-use planes - and leaves an
// unassigned code point literal. Every ASSIGNED character is exact, and the
// whole of Latin-1 (the only range \xNN can ever come from) is exact by
// construction. Same table as pyReprEsc in languages/python-interpreter.abnf and
// languages/lib/python-rt.metajs.
func pyReprEsc(cp rune) bool {
	switch {
	case cp < 0x20, cp == 0x7f:
		return true
	case cp >= 0x80 && cp <= 0xa0, cp == 0xad: // C1, NBSP, SOFT HYPHEN
		return true
	case cp < 0x100:
		return false
	case cp >= 0x600 && cp <= 0x605, cp == 0x61c, cp == 0x6dd, cp == 0x70f:
		return true
	case cp == 0x890, cp == 0x891, cp == 0x8e2:
		return true
	case cp == 0x1680, cp == 0x180e:
		return true
	case cp >= 0x2000 && cp <= 0x200f, cp >= 0x2028 && cp <= 0x202f:
		return true
	case cp >= 0x205f && cp <= 0x206f, cp == 0x3000:
		return true
	case cp >= 0xd800 && cp <= 0xdfff, cp >= 0xe000 && cp <= 0xf8ff:
		return true
	case cp == 0xfeff, cp >= 0xfff9 && cp <= 0xfffb:
		return true
	case cp == 0x110bd, cp == 0x110cd, cp >= 0x13430 && cp <= 0x1343f:
		return true
	case cp >= 0x1bca0 && cp <= 0x1bca3, cp >= 0x1d173 && cp <= 0x1d17a:
		return true
	case cp == 0xe0001, cp >= 0xe0020 && cp <= 0xe007f:
		return true
	case cp >= 0xf0000 && cp <= 0xffffd, cp >= 0x100000 && cp <= 0x10fffd:
		return true
	}
	return false
}

// pyStrRepr is CPython's unicode_repr. The quote is ' unless the string holds a
// ' and no " (then it is "), so repr("it's") reads "it's" while
// repr(`both ' and "`) reads 'both \' and "'. Inside, the quote in force and a
// backslash DOUBLE UP, \t \n \r have their short forms, and every other
// non-printable code point becomes \xNN (below U+0100), \uNNNN (below U+10000)
// or \UNNNNNNNN. Everything printable - "café" and an emoji included - is
// emitted literally. It is pyStrRepr in languages/python-interpreter.abnf and
// languages/lib/python-rt.metajs, one for one.
func pyStrRepr(s string) string {
	quote := "'"
	if strings.ContainsRune(s, '\'') && !strings.ContainsRune(s, '"') {
		quote = "\""
	}
	out := quote
	for _, r := range s {
		switch {
		case string(r) == quote || r == '\\':
			out += "\\" + string(r)
		case r == '\t':
			out += "\\t"
		case r == '\n':
			out += "\\n"
		case r == '\r':
			out += "\\r"
		case !pyReprEsc(r):
			out += string(r)
		case r < 0x100:
			out += fmt.Sprintf("\\x%02x", r)
		case r < 0x10000:
			out += fmt.Sprintf("\\u%04x", r)
		default:
			out += fmt.Sprintf("\\U%08x", r)
		}
	}
	return out + quote
}

// pyRepr renders a container element like Python's repr: strings get quotes.
func (rt *jsrt) pyRepr(v interface{}) string {
	if s, isStr := v.(string); isStr {
		return pyStrRepr(s)
	}
	// A CLASS object is not an instance to render: pyString's own <class 'name'>
	// arm has to win, exactly as it does for str().
	if _, isCls := rt.pyClassName(v); !isCls {
		if o, cls, isInst := pyInstance(v); isInst {
			return rt.pyUserRender(o, cls, true)
		}
	}
	return rt.pyString(v)
}

// pyUserRender renders a USER INSTANCE. This half had no such arm at all, so a
// class with a __repr__ printed [object Object] under llvm.Run and the native
// binary where the interpreter and CPython print V(3) - and so did an exception
// and a plain instance. It is pyUserRender in languages/python-interpreter.abnf,
// one for one, including the split CPython makes: str() and print() prefer
// __str__ and fall back to __repr__, while repr() and every CONTAINER element use
// __repr__ only. The <Name object> default deliberately omits CPython's
// `at 0x...` address, which no engine here can reproduce.
func (rt *jsrt) pyUserRender(o *jsObject, cls *jsObject, wantRepr bool) string {
	if !wantRepr {
		if m, found := pyLookup(cls, "__str__"); found && isCallable(m) {
			return rt.toString(rt.call(m, jsUndef, []interface{}{o}))
		}
		// BaseException.__str__, which an exception INHERITS and which therefore
		// beats a __repr__ the class also defines: no args is "", one arg is
		// str(that arg), more is the args TUPLE's repr.
		if args, ok := o.props["args"].(*jsArray); ok {
			if len(args.elems) == 0 {
				return ""
			}
			// KeyError OVERRIDES BaseException.__str__ with repr(args[0]), which
			// is why str(KeyError('x')) is "'x'" and not "x". The argument is the
			// key itself and the repr happens here.
			if len(args.elems) == 1 && pyIsKeyErrCls(cls) {
				return rt.pyRepr(args.elems[0])
			}
			if len(args.elems) == 1 {
				return rt.pyString(args.elems[0])
			}
			out := "("
			for i, e := range args.elems {
				if i > 0 {
					out += ", "
				}
				out += rt.pyRepr(e)
			}
			return out + ")"
		}
	}
	if m, found := pyLookup(cls, "__repr__"); found && isCallable(m) {
		return rt.toString(rt.call(m, jsUndef, []interface{}{o}))
	}
	// An exception renders as Name(args...), like Python's default repr.
	if args, ok := o.props["args"].(*jsArray); ok {
		out := rt.pyClsName(cls) + "("
		for i, e := range args.elems {
			if i > 0 {
				out += ", "
			}
			out += rt.pyRepr(e)
		}
		return out + ")"
	}
	return "<" + rt.pyClsName(cls) + " object>"
}

// pyClsName is a class object's __name, or "object" when it has none.
func (rt *jsrt) pyClsName(cls *jsObject) string {
	if n, ok := cls.props["__name"].(string); ok {
		return n
	}
	return "object"
}

// pyString renders a value like Python's str(): True/False/None capitalized,
// lists in bracket and dicts in brace notation.
func (rt *jsrt) pyString(v interface{}) string {
	switch t := v.(type) {
	case jsUndefT, jsNullT:
		return "None"
	case bool:
		if t {
			return "True"
		}
		return "False"
	case *jsArray:
		out := "["
		for i, e := range t.elems {
			if i > 0 {
				out += ", "
			}
			out += rt.pyRepr(e)
		}
		return out + "]"
	case *jsObject:
		// A TYPE object renders as CPython's <class 'name'>, so print(type(1.0))
		// reads <class 'float'> rather than the default object rendering.
		if re, im, ok := pyComplexParts(t); ok {
			return pyCplxRepr(re, im)
		}
		if name, ok := rt.pyClassName(t); ok {
			// A class a `class` statement made carries its defining module in
			// an own __module__ (<class '__main__.B'>); a builtin has none.
			if mod, ok := t.props["__module__"].(string); ok {
				name = mod + "." + name
			}
			return "<class '" + name + "'>"
		}
		if els, ok := pySetElems(t); ok {
			if len(els.elems) == 0 {
				return "set()" // {} is the empty DICT
			}
			out := "{"
			for i, e := range els.elems {
				if i > 0 {
					out += ", "
				}
				out += rt.pyRepr(e)
			}
			return out + "}"
		}
		if keys, vals, ok := dictParts(t); ok {
			out := "{"
			for i := range keys.elems {
				if i > 0 {
					out += ", "
				}
				out += rt.pyRepr(keys.elems[i]) + ": " + rt.pyRepr(vals.elems[i])
			}
			return out + "}"
		}
		if o, cls, isInst := pyInstance(t); isInst {
			return rt.pyUserRender(o, cls, false)
		}
		// An iterator object. CPython prints <map object at 0x...>; the address is
		// its own and cannot be matched, so the address is left off rather than
		// invented - and the same short form is printed by all three engines.
		if n := pyIterName(rt, t); n != "" {
			return "<" + n + " object>"
		}
		// A compiled PATTERN and a MATCH render the way CPython's repr does.
		if s, ok := rxPyRepr(t); ok {
			return s
		}
		// A MODULE object. CPython prints <module 'builtins' (built-in)> for a
		// built-in module and, for an extension module like math, the .so PATH
		// it was loaded from - machine-specific and unmatchable, so every module
		// here takes the built-in spelling.
		if mod, isMod := pyIsModule(t); isMod {
			return "<module '" + rt.toString(mod.props["__pymod"]) + "' (built-in)>"
		}
		return rt.toString(v)
	default:
		if isCallable(v) {
			return rt.pyFuncRender(v)
		}
		// A GENERATOR. CPython says <generator object gen at 0x...>; neither the
		// address nor the generating function's name is recoverable from the
		// value, so the short form is printed - the same rule <map object> above
		// follows.
		if _, isGen := v.(*jsGenerator); isGen {
			return "<generator object>"
		}
		return rt.toString(v)
	}
}

// pyFuncRender is how a FUNCTION prints. CPython says <function fn at 0x...>; the
// address is its own and cannot be matched, so it is left off rather than
// invented - the same rule pyIterName's <map object> line above follows. The
// name comes from rt.pyFuncNames, the side table js_pyfnname fills; CPython
// calls an unnamed one <lambda>, and so does every engine here.
func (rt *jsrt) pyFuncRender(v interface{}) string {
	// A BOUND METHOD prints as one. CPython appends the
	// receiver's ADDRESS in both forms; the address is its own and cannot be
	// matched, so it is left off here exactly as it is for <function f> above.
	if bi := pyBoundOf(v); bi != nil {
		// An UNBOUND descriptor names the type it belongs to, in single quotes,
		// and says "objects" plural - CPython's own spelling for list.count. A
		// module-level builtin prints exactly like any other builtin function.
		if bi.kind == "d" {
			return "<method '" + bi.name + "' of '" + bi.typ + "' objects>"
		}
		if bi.kind == "m" {
			return "<built-in function " + bi.name + ">"
		}
		if bi.fn == nil {
			return "<built-in method " + bi.name + " of " + bi.typ + " object>"
		}
		return "<bound method " + bi.typ + "." + bi.name + " of " + rt.pyRepr(bi.self) + ">"
	}
	// A BUILTIN prints its own way, and its table is asked FIRST: a user
	// `def len(...)` records its own name against its own closure object, so the
	// two tables cannot collide by identity even when they collide by name.
	if c, ok := rt.pyBuiltinRender[v]; ok {
		if strings.HasPrefix(c, "*") {
			return "<class '" + c[1:] + "'>"
		}
		return "<built-in function " + c + ">"
	}
	if n, ok := rt.pyFuncNames[v]; ok {
		return "<function " + n + ">"
	}
	return "<function <lambda>>"
}

// csString renders a value for C# style string concatenation: a null operand
// contributes the EMPTY string (String.Concat skips it), where Java writes "null".
func (rt *jsrt) csString(v interface{}) string {
	if isUndefOrNull(v) {
		return ""
	}
	return rt.toString(v)
}

// javaString renders a value for Java style string concatenation:
// null/undefined print as "null", everything else like toString.
func (rt *jsrt) javaString(v interface{}) string {
	if isUndefOrNull(v) {
		return "null"
	}
	return rt.toString(v)
}

// findClassAccessor looks a getter/setter up on the __class descriptor chain of an
// instance. Accessors are stored on the class, not on the instance, so an instance
// read/write has to follow the same single-inheritance chain that memberCall walks
// for methods. Returns nil when the name is not an accessor anywhere on the chain.
func (rt *jsrt) findClassAccessor(o *jsObject, name string) *jsAccessor {
	cls, ok := o.props["__class"]
	if !ok {
		// A CLASS DESCRIPTOR has no __class of its own: its STATIC accessors are
		// inherited along its own __super chain, which is this very walk one link
		// further in. That is how `class D extends C {}` reaches C's `static get g()`
		// and `static set g(v)`.
		if isc, isClass := o.props["__isclass"]; !isClass || isc != true {
			return nil
		}
		cls, ok = o.props["__super"]
		if !ok {
			return nil
		}
	}
	for cls != nil {
		clsObj, isObj := cls.(*jsObject)
		if !isObj {
			return nil
		}
		if v, found := clsObj.props[name]; found {
			if acc, isAcc := v.(*jsAccessor); isAcc {
				return acc
			}
			return nil
		}
		cls = clsObj.props["__super"]
	}
	return nil
}

// jsBigInt is the ECMAScript BigInt type: an arbitrary precision integer, a value
// type of its own (typeof is "bigint") that never silently mixes with the double
// numbers. The runtime models the type, its literals, its printing, its equality
// and comparison, and the arithmetic BETWEEN two BigInts; a BigInt that meets a
// double in an arithmetic operator is converted to a double instead of raising the
// TypeError the spec asks for, which is the one simplification here.
type jsBigInt struct {
	v *big.Int
}

// jsChar is Kotlin's Char: neither an Int nor a one-character String. It RENDERS as
// its glyph (in a string template, in println, in string concatenation) but COMPARES
// and does ARITHMETIC on its code, so `'A' + 1 == 'B'`, `'z' > 'a'` and `"" + 'b' ==
// "b"` all hold at once - which no single unboxed representation can give. The
// kotlin-interpreter grammar models the same value as a boxed {__char: code}; the
// semantics here are matched to it so the two engines agree.
//
// Only the Kotlin compiler grammar ever creates one (the js_char / js_kindex externs
// below), so every branch added for this type is unreachable from MetaJS, JS, Java,
// Go, Python and the rest: their behaviour is untouched.
type jsChar struct {
	code int32
}

// jsSym is Ruby's Symbol: it RENDERS as its name (in an interpolation, in puts, in
// string concatenation) but is NOT a String, so `:hello == "hello"` is false and
// `{size: 4}["size"]` is nil - :size and "size" are different Hash keys. That pair of
// requirements is exactly why it cannot simply be the name string, and it mirrors the
// boxed {__sym: true, s: name} of ruby-interpreter.abnf.
//
// It is a comparable value struct, so two symbols of the same name are equal through
// identityEq (and hash to the same Go map key in a dict index) without any interning.
// Only the Ruby compiler grammar ever creates one, via the js_sym extern below.
type jsSym struct {
	s string
}

// ----- Ruby's numeric tower: Integer / Float / Rational / Complex -----
//
// Ruby needs 7 / 2 == 3 and 7.0 / 2 == 3.5 to hold at the SAME time, so Integer and
// Float cannot both be the runtime's one number type. An Integer stays the plain
// float64 (exact to 2^53, and never truncated to 32 bit the way the Java style
// operators are); Float, Rational and Complex are BOXED comparable value structs,
// mirroring the {__flo}/{__rat}/{__cpx} boxes of ruby-interpreter.abnf value for
// value so the two engines agree.
//
// Only ruby-to-llvm-ir.abnf ever creates one (the js_rflo / js_rrat / js_rcpx
// externs below), so every branch added for these three types is unreachable from
// MetaJS, JS, Java, Kotlin, Go, Python and the rest.
type jsFlo struct{ f float64 }

// n / d, or past 2^53 the exact bn / bd with n / d their nearest doubles: see
// rubyMkRatB in jsrtruby.go.
type jsRat struct {
	n, d   float64
	bn, bd *big.Int
}

// A Complex's parts are TYPED Ruby reals (float64 Integer, *jsBigInt, jsFlo,
// jsRat): see rubyCpxBin in jsrtruby.go.
type jsCpx struct{ re, im interface{} }

// rubyNumRank orders the tower: int < rational < float < complex. Every arithmetic
// step promotes both operands to the higher of the two ranks. -1 is "not a number".
// A *jsBigInt is rank 0: it IS an Integer, not a fifth kind of number. Ruby's
// Integer has no width, so a value the double cannot count exactly carries the
// arbitrary precision shape and everything else stays the plain number it was -
// which is what keeps the fast path two comparisons wide (see rubyOver53). A big
// meeting a Rational, a Float or a Complex promotes to that rank THROUGH THE
// DOUBLE, i.e. it stops being exact; that boundary is stated at rubyBigArith.
func rubyNumRank(v interface{}) int {
	switch v.(type) {
	case float64:
		return 0
	case jsRat:
		return 1
	case jsFlo:
		return 2
	case jsCpx:
		return 3
	case *jsBigInt:
		return 0
	}
	return -1
}

func rubyIsNum(v interface{}) bool { return rubyNumRank(v) >= 0 }

func rubyToF(v interface{}) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case jsRat:
		return rubyRatToF(t)
	case jsFlo:
		return t.f
	case jsCpx:
		return rubyToF(t.re)
	case *jsBigInt:
		f, _ := new(big.Float).SetInt(t.v).Float64()
		return f
	}
	return 0
}

// rubyBigMax is 2^53, the largest integer a double still counts exactly, as a
// big.Int - the boundary between the plain and the arbitrary precision shape.
var rubyBigMax = big.NewInt(9007199254740992)

// rubyBigNarrow is the only exit from the boxed world, and the twin of bigOut in
// ruby-interpreter.abnf and rtABigNarrow in languages/lib/runtime.metajs: a value
// that still fits the exactly representable range comes back as a plain number,
// so only what GENUINELY needs arbitrary precision carries the shape.
func (rt *jsrt) rubyBigNarrow(x *big.Int) interface{} {
	if x.IsInt64() {
		if n := x.Int64(); n <= 9007199254740992 && n >= -9007199254740992 {
			return float64(n)
		}
	}
	rt.hasBigInt = true
	return &jsBigInt{v: x}
}

// rubyBigOf answers an EXACT big.Int for an Integer operand: a big as itself, and
// an integral double inside the exactly representable range as itself. A Float
// box, a Rational, a non-integral double and anything past 2^53 that is NOT
// already a big answer false, so the caller falls back to the double path rather
// than inventing precision the operand never had.
func rubyBigOf(v interface{}) (*big.Int, bool) {
	switch t := v.(type) {
	case *jsBigInt:
		return t.v, true
	case bool:
		if t {
			return big.NewInt(1), true
		}
		return big.NewInt(0), true
	case float64:
		// A plain double past 2^53 lifts EXACTLY, which is where this differs
		// from pyBigOperand below. A Python int that large is already boxed and a
		// bare double IS a float; in Ruby a plain number is always an Integer and
		// a large one can only have come from Float#to_i or float arithmetic,
		// where MRI's answer is the exact integer value OF THAT DOUBLE. So
		// `(1e30).to_i + 1` is 1000000000000000019884624838657, as in MRI. The
		// finiteness test is not optional: big.Float of an infinity has no Int.
		if t == math.Trunc(t) && !math.IsInf(t, 0) {
			bf, _ := big.NewFloat(t).Int(nil)
			return bf, true
		}
	}
	return nil, false
}

// rubyBigPair answers both Integer operands exactly when at least one of them is
// a big. Ruby has ONE Integer type and it is arbitrary precision, so a plain int
// operand meeting a big PROMOTES - which is why this is not bigPair (the
// ECMAScript rule shared with js_add and the rest, where a BigInt refuses to mix).
func rubyBigPair(a, b interface{}) (*big.Int, *big.Int, bool) {
	_, aBig := a.(*jsBigInt)
	_, bBig := b.(*jsBigInt)
	if !aBig && !bBig {
		return nil, nil, false
	}
	x, xok := rubyBigOf(a)
	if !xok {
		return nil, nil, false
	}
	y, yok := rubyBigOf(b)
	if !yok {
		return nil, nil, false
	}
	return x, y, true
}

// rubyOver53 is the PROMOTION TRIGGER for two plain Integers, and it is sound
// both ways. Every plain operand is exact and of magnitude <= 2^53 (anything
// larger is already a big and was taken by rubyBigArith before this), so if the
// TRUE result is >= 2^53 the rounded double is too - rounding to nearest of a
// value at or above 2^53 cannot land below it - and if the true result is < 2^53
// the double is exact and the guard does not fire.
func rubyOver53(v float64) bool { return v >= 9007199254740992 || v <= -9007199254740992 }

// rubyBigArith is one arithmetic step at arbitrary precision, and the twin of
// bigOp in ruby-interpreter.abnf and rbABigArith in ruby-rt.metajs. It answers
// ok=false for a pair it cannot take exactly, and the caller then keeps the
// double path it always had. Ruby FLOORS: -7 / 2 is -4 and -7 % 3 is 2, which is
// math/big's DivMod (Euclidean) corrected for a negative divisor, not Quo/Rem.
func (rt *jsrt) rubyBigArith(op string, l, r interface{}) (interface{}, bool) {
	x, y, ok := rubyBigPair(l, r)
	if !ok {
		return nil, false
	}
	switch op {
	case "+":
		return rt.rubyBigNarrow(new(big.Int).Add(x, y)), true
	case "-":
		return rt.rubyBigNarrow(new(big.Int).Sub(x, y)), true
	case "*":
		return rt.rubyBigNarrow(new(big.Int).Mul(x, y)), true
	case "/", "%":
		if y.Sign() == 0 {
			rt.rubyRaiseErr("ZeroDivisionError", "divided by 0")
		}
		q, m := new(big.Int), new(big.Int)
		q.QuoRem(x, y, m)
		if m.Sign() != 0 && (m.Sign() < 0) != (y.Sign() < 0) {
			q.Sub(q, big.NewInt(1))
			m.Add(m, y)
		}
		if op == "/" {
			return rt.rubyBigNarrow(q), true
		}
		return rt.rubyBigNarrow(m), true
	case "**":
		if y.Sign() < 0 {
			if y.IsInt64() && y.Int64() >= -4096 {
				return rt.rubyRatPow(rt.rubyAsRat(l), y), true
			}
			return jsFlo{f: math.Pow(rubyToF(l), rubyToF(r))}, true
		}
		if !y.IsInt64() || y.Int64() > 4096 {
			return math.Pow(rubyToF(l), rubyToF(r)), true
		}
		return rt.rubyBigNarrow(new(big.Int).Exp(x, y, nil)), true
	case "&", "|", "^", "<<", ">>":
		out, oky := rubyBigBits(op, x, y)
		if !oky {
			return nil, false
		}
		return rt.rubyBigNarrow(out), true
	}
	return nil, false
}

// rubyBigPromote redoes one Integer step at arbitrary precision after the double
// answer was found to have left the exact range. Both operands are plain numbers
// here, so rubyBigPair cannot fire on them - they are lifted explicitly. A pair
// that has no exact integer form (a non-integral double, an absurd shift count)
// keeps the double answer it always had.
func (rt *jsrt) rubyBigPromote(op string, x, y float64) interface{} {
	bx, okx := rubyBigOf(x)
	by, oky := rubyBigOf(y)
	if okx && oky {
		if out, ok := rt.rubyBigArith(op, &jsBigInt{v: bx}, &jsBigInt{v: by}); ok {
			return out
		}
	}
	switch op {
	case "+":
		return x + y
	case "-":
		return x - y
	case "*":
		return x * y
	case "**":
		return math.Pow(x, y)
	case "&":
		return float64(int64(x) & int64(y))
	case "|":
		return float64(int64(x) | int64(y))
	case "^":
		return float64(int64(x) ^ int64(y))
	case "<<":
		return x * math.Pow(2, y)
	}
	return math.Floor(x / math.Pow(2, y))
}

// rubyBigBits is |, &, ^, << and >> at arbitrary precision. math/big's And/Or/Xor
// ARE Ruby's infinite two's complement (they are defined on the signed value, not
// on a fixed width), and a NEGATIVE shift count is the opposite shift in Ruby -
// `1 << -2` is 0 and `4 >> -2` is 16 - rather than an error.
func rubyBigBits(op string, x, y *big.Int) (*big.Int, bool) {
	switch op {
	case "&":
		return new(big.Int).And(x, y), true
	case "|":
		return new(big.Int).Or(x, y), true
	case "^":
		return new(big.Int).Xor(x, y), true
	}
	if !y.IsInt64() {
		return nil, false
	}
	k := y.Int64()
	left := op == "<<"
	if k < 0 {
		k, left = -k, !left
	}
	if k > 1048576 {
		return nil, false
	}
	if left {
		return new(big.Int).Lsh(x, uint(k)), true
	}
	return new(big.Int).Rsh(x, uint(k)), true
}

// rubyBigCmp compares two Integers exactly when either is a big. Through
// rubyToF they would both round to the same double and answer 0.
func rubyBigCmp(l, r interface{}) (int, bool) {
	if _, lb := l.(*jsBigInt); !lb {
		if _, rb := r.(*jsBigInt); !rb {
			return 0, false
		}
	}
	x, xok := rubyBigCmpOf(l)
	if !xok {
		return 0, false
	}
	y, yok := rubyBigCmpOf(r)
	if !yok {
		return 0, false
	}
	return x.Cmp(y), true
}

// rubyIntStr renders an Integer. Ruby NEVER writes one in exponent form, which is
// what strconv's shortest form does from 1e21 up, so a plain double that large
// gets its own exact expansion - `(2.0 ** 70).to_i` prints all 22 digits.
// The twins are intStr in ruby-interpreter.abnf
// and rbIntStr in languages/lib/ruby-rt.metajs.
func rubyIntStr(x float64) string {
	if x != x || math.IsInf(x, 0) || x != math.Trunc(x) {
		return jsNumString(x)
	}
	if x <= 9007199254740992 && x >= -9007199254740992 {
		return jsNumString(x)
	}
	bf, _ := big.NewFloat(x).Int(nil)
	return bf.String()
}

// ----- Float#round / #floor / #ceil AND Integer#round / #floor / #ceil with a
// DIGIT COUNT -----------------------------------------------------------------
//
// MRI's algorithm (numeric.c), not an approximation of it: #round corrects
// toward the DECIMAL the double prints as, a negative ndigits answers an
// INTEGER, and Infinity/NaN answer the receiver. Against ruby 2.6.10
// (Float#round is unchanged in 3.x):
//
//	2.345.round(2)   2.35     (2.345 is really 2.3449999999999997513)
//	2.675.round(2)   2.68     (2.675 is really 2.6749999999999998224)
//	3.7.floor(1)     3.7      2.345.floor(2)  2.34    2.345.ceil(5)  2.34501
//	1234.567.round(-2) -> the Integer 1200, and (…).class is Integer
//	1.7976931348623157e308.round(1) -> itself, where x*10 is Infinity
//	5e-324.round(374) -> NaN, because MRI's own pow(10, 374) is Infinity
//
// The twins are floBinExp/floDigitRound/intDigitOp in languages/ruby-interpreter.abnf
// and rbFloBinExp/rbFloDigitRound/rbIntDigitOp in languages/lib/ruby-rt.metajs.

// 10**ndigits * 2**binexp is past the mantissa: every digit asked for is already
// there, so the receiver IS the answer. And the other end: the value is entirely
// below the last digit asked for. Both are MRI's, in C's TRUNCATING integer
// division, which Go's `/` on ints already is.
func rubyFloRoundOver(nd, be int) bool {
	if be > 0 {
		return nd >= 17-be/4
	}
	return nd >= 17-(be/3-1)
}

func rubyFloRoundUnder(nd, be int) bool {
	if be > 0 {
		return nd < -(be/3 + 1)
	}
	return nd < -(be / 4)
}

// 10**nd as a double, by PARSING the decimal rather than by math.Pow. The tag
// engine of ruby-interpreter.abnf keeps an integral value as a sized integer and
// its integer power WRAPS (Math.pow(10, 20) is 10**20 mod 2**64 there), and layer
// 2's Math.pow is the C floor's d_pow - a third implementation to agree with.
// Parsing is exact in all three and answers +Inf past 1e308, which is what MRI's
// own pow() does; strconv reports that as an ErrRange the caller does not want.
func rubyPow10(nd int) float64 {
	v, _ := strconv.ParseFloat("1e"+strconv.Itoa(nd), 64)
	return v
}

// The three ndigits > 0 bodies. round_half_up is C's round() (math.Round) plus
// the correction that makes 2.675.round(2) == 2.68: if the double `(f + 0.5) / s`
// is <= x then x is at or past the midpoint whatever the scaled product said.
// That correction is why the exact decimal expansion is NOT used here - it would
// answer 2.67, which is the true value and not MRI's. A zero result carries the
// receiver's sign (-0.05.ceil(1) is -0.0 in MRI).
func rubyFloDigitRound(x float64, nd int) float64 {
	_, be := math.Frexp(x)
	if rubyFloRoundOver(nd, be) {
		return x
	}
	if rubyFloRoundUnder(nd, be) {
		return 0
	}
	s := rubyPow10(nd)
	f := math.Round(x * s)
	if x > 0 {
		if (f+0.5)/s <= x {
			f++
		}
	} else {
		if (f-0.5)/s >= x {
			f--
		}
	}
	r := f / s
	if r == 0 {
		return math.Copysign(0, x)
	}
	return r
}

func rubyFloDigitFloor(x float64, nd int) float64 {
	_, be := math.Frexp(x)
	if rubyFloRoundOver(nd, be) {
		return x
	}
	if x > 0 && rubyFloRoundUnder(nd, be) {
		return 0
	}
	s := rubyPow10(nd)
	r := math.Floor(x*s) / s
	if r == 0 {
		return math.Copysign(0, x)
	}
	return r
}

func rubyFloDigitCeil(x float64, nd int) float64 {
	_, be := math.Frexp(x)
	if rubyFloRoundOver(nd, be) {
		return x
	}
	if x < 0 && rubyFloRoundUnder(nd, be) {
		return 0
	}
	s := rubyPow10(nd)
	r := math.Ceil(x*s) / s
	if r == 0 {
		return math.Copysign(0, x)
	}
	return r
}

// MRI's int_round_zero_p, and it is NOT an optimisation that happens to agree
// with the arithmetic - it OVERRIDES it. numeric.c: "If 10**N / 2 > num, then
// return 0", decided on the receiver's BYTE SIZE with log_256(10) > 0.415241 and
// log_256(1/2) == -0.125, where a Fixnum's size is sizeof(long) == 8 whatever the
// value is. So `(-1234).floor(-20)` is 0 in MRI where the arithmetic says
// -10**20. First-zero-at: -20 for every Fixnum and for 2**63, -22 for
// 2**64 (9 bytes), -32 for 2**100 (13), -63 for 2**200 (26), -304 for 2**1000
// (126), -309 for Float::MAX floored (128).
func rubyIntByteSize(v interface{}) int {
	bits := 0
	if b, isBig := v.(*jsBigInt); isBig {
		bits = b.v.BitLen()
	} else {
		m := math.Abs(rubyToF(v))
		for m >= 1 {
			m = math.Floor(m / 2)
			bits++
		}
	}
	bytes := (bits + 7) / 8
	if bytes < 8 {
		bytes = 8
	}
	return bytes
}

func rubyIntRoundZero(v interface{}, k int) bool {
	return 0.415241*float64(k)-0.125 > float64(rubyIntByteSize(v))
}

// rubyDecIncr is "199" -> "200": a decimal string incremented by one.
func rubyDecIncr(s string) string {
	b := []byte(s)
	for i := len(b) - 1; i >= 0; i-- {
		if b[i] != '9' {
			b[i]++
			return string(b)
		}
		b[i] = '0'
	}
	return "1" + string(b)
}

// Integer#round / #floor / #ceil with k = -ndigits > 0, and the ndigits < 0 half
// of the Float methods once the receiver is an Integer. Exact, and on the DECIMAL
// STRING rather than on 10**k, because the other two engines cannot multiply by a
// 10**k past 2**53 without a bignum either. mode is 'r', 'f' or 'c'; #round ties
// AWAY FROM ZERO (25.round(-1) is 30, -25.round(-1) is -30).
//
// One deliberate NON-convergence: MRI 2.6.10's BIGNUM #ceil adds the step even
// when the receiver is already a multiple of it - `10000000000000000000.ceil(-1)`
// is 10000000000000000010 there, while the Fixnum `1250.ceil(-1)` is 1250. MRI
// disagrees with itself, so that is a bug in the oracle and not a rule.
func (rt *jsrt) rubyIntDigitOp(v interface{}, k int, mode byte) interface{} {
	if rubyIntRoundZero(v, k) {
		return float64(0)
	}
	var s string
	if b, isBig := v.(*jsBigInt); isBig {
		s = b.v.String()
	} else {
		s = rubyIntStr(rubyToF(v))
	}
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	head, tail := "", s
	if k < len(s) {
		head, tail = s[:len(s)-k], s[len(s)-k:]
	} else {
		for len(tail) < k {
			tail = "0" + tail
		}
	}
	zeroTail := true
	for i := 0; i < len(tail); i++ {
		if tail[i] != '0' {
			zeroTail = false
			break
		}
	}
	away := false
	// Mode 't' is TRUNCATION and never carries: 1234.truncate(-2) is 1200.
	if !zeroTail && mode != 't' {
		switch mode {
		case 'r':
			away = tail[0] >= '5'
		case 'f':
			away = neg
		default:
			away = !neg
		}
	}
	mag := head
	if mag == "" {
		mag = "0"
	}
	if away {
		mag = rubyDecIncr(mag)
	}
	if mag == "0" {
		return float64(0)
	}
	out := mag + strings.Repeat("0", k)
	if neg {
		out = "-" + out
	}
	bi, _ := new(big.Int).SetString(out, 10)
	return rt.rubyBigNarrow(bi)
}

// rubyNumDigitOp is the one entry point both numeric method sets call. `t` is the
// receiver's double, `isInt` says whether the receiver is an Integer (an Integer
// is already rounded at every ndigits >= 0), `big` is the receiver itself when it
// is a big and nil otherwise, and mode is 'r' / 'f' / 'c'.
func (rt *jsrt) rubyNumDigitOp(t float64, big *jsBigInt, isInt bool, mode byte, nd int) interface{} {
	if isInt {
		if nd >= 0 {
			if big != nil {
				return big
			}
			return t
		}
		if big != nil {
			return rt.rubyIntDigitOp(big, -nd, mode)
		}
		return rt.rubyIntDigitOp(t, -nd, mode)
	}
	if nd > 0 {
		if t != t || math.IsInf(t, 0) || t == 0 {
			return jsFlo{f: t}
		}
		switch mode {
		case 'r':
			return jsFlo{f: rubyFloDigitRound(t, nd)}
		case 'f':
			return jsFlo{f: rubyFloDigitFloor(t, nd)}
		case 't':
			// #truncate at a POSITIVE ndigits is toward zero: floor above, ceil
			// below.
			if t < 0 {
				return jsFlo{f: rubyFloDigitCeil(t, nd)}
			}
			return jsFlo{f: rubyFloDigitFloor(t, nd)}
		}
		return jsFlo{f: rubyFloDigitCeil(t, nd)}
	}
	rubyFloDomain(rt, t)
	// ndigits == 0 rounds the DOUBLE; a NEGATIVE one truncates first and then
	// applies the Integer rule (MRI: rb_int_round(flo_truncate(num), ndigits)).
	// 1234.567.round(-1) is 1230 and not the 1240 that rounding to 1235 first
	// would give.
	if nd == 0 {
		switch mode {
		case 'r':
			return rt.rubyIntFromF(math.Round(t))
		case 'f':
			return rt.rubyIntFromF(math.Floor(t))
		case 't':
			return rt.rubyIntFromF(math.Trunc(t))
		}
		return rt.rubyIntFromF(math.Ceil(t))
	}
	w := math.Trunc(t)
	if mode == 'f' {
		w = math.Floor(t)
	} else if mode == 'c' {
		w = math.Ceil(t)
	}
	return rt.rubyIntDigitOp(rt.rubyIntFromF(w), -nd, mode)
}

func rubyGcd(a, b float64) float64 {
	x, y := math.Abs(a), math.Abs(b)
	for y != 0 {
		x, y = y, math.Mod(x, y)
	}
	return x
}

// rubyMkRat normalizes a Rational: the sign lives in the numerator and the pair is
// reduced by its gcd, so two equal Rationals are equal as Go values too.
func rubyMkRat(n, d float64) jsRat {
	if d < 0 {
		n, d = -n, -d
	}
	if g := rubyGcd(n, d); g > 1 {
		n, d = n/g, d/g
	}
	return jsRat{n: n, d: d}
}

func rubyReOf(v interface{}) interface{} {
	if c, ok := v.(jsCpx); ok {
		return c.re
	}
	return v
}

func rubyImOf(v interface{}) interface{} {
	if c, ok := v.(jsCpx); ok {
		return c.im
	}
	return 0.0
}

// ----- Float#to_r, EXACTLY -------------------------------------------------
//
// A double IS m * 2^e with m an integer below 2^53, so its Rational is exact and needs NO
// bit access - the same argument manual 7.7 makes for ruby's %a, and the same safety
// property: scaling a double by a power of two is exact provided you never overflow and
// never enter the subnormals, and the loops below are driven so they do not.
//
// The arithmetic is spelled out rather than handed to math.Frexp, so this twin, layer 2
// and the interpreter run THE SAME STEPS and agree by construction; math.Frexp takes an
// out-parameter and is not reachable from MetaJS at all (manual 7.7).
//
// A float64 denominator holds 2^-e only while -e <= 1023; below about 2^-971 the
// pair is built as bigs (rubyMkRatB).
//
// An infinity or a NaN has no Rational: MRI raises FloatDomainError, and without the guard
// rubyGcd would loop forever on math.Mod(x, NaN). The twins are ratOfFlo in
// ruby-interpreter.abnf and rbRatOfFlo in languages/lib/ruby-rt.metajs.
func (rt *jsrt) rubyRatOfFlo(x float64) jsRat {
	rubyFloDomain(rt, x)
	if x == 0 {
		return rubyMkRat(0, 1)
	}
	neg := x < 0
	a := x
	if neg {
		a = -x
	}
	e := 0
	// Normalize the significand into [2^52, 2^53), where it is an exact integer.
	// Doubling a subnormal and halving a large value are both exact.
	for a < 4503599627370496 {
		a = a * 2
		e = e - 1
	}
	for a >= 9007199254740992 {
		a = a / 2
		e = e + 1
	}
	// Reduce the power of two out of the numerator, so the gcd has nothing to find.
	for e < 0 && a/2 == math.Floor(a/2) {
		a = a / 2
		e = e + 1
	}
	// Every pair built here is ALREADY reduced - the loop above took the twos out of a
	// and d is a power of two, so their gcd is 1 - so the value is built directly rather
	// than through rubyMkRat. That is not only a saving: rubyMkRat's Euclid runs
	// math.Mod on a pair spanning 1074 binades, and layer 2's twin lowers that to the
	// floor's d_mod_go, a shift-and-subtract loop with one iteration per binade, which
	// takes the native binary tens of seconds on the fallback pair (5e-324, 1).
	//
	// e >= 0 means x is itself an integer, and then to_r is x/1 - including every
	// integral Float past 2^53, whose numerator rubyIntStr expands exactly.
	if e >= 0 {
		return jsRat{n: x, d: 1}
	}
	if e < -1023 {
		n := rubyExactInt(a)
		if neg {
			n.Neg(n)
		}
		return rubyMkRatB(n, new(big.Int).Lsh(big.NewInt(1), uint(-e)))
	}
	d := 1.0
	for k := 0; k < -e; k++ {
		d = d * 2
	}
	if neg {
		return jsRat{n: -a, d: d}
	}
	return jsRat{n: a, d: d}
}

func (rt *jsrt) rubyAsRat(v interface{}) jsRat {
	if r, ok := v.(jsRat); ok {
		return r
	}
	if f, ok := v.(jsFlo); ok {
		return rt.rubyRatOfFlo(f.f)
	}
	if n, ok := v.(float64); ok {
		return rubyMkRat(n, 1)
	}
	if b, ok := v.(*jsBigInt); ok {
		return rubyMkRatB(b.v, big.NewInt(1))
	}
	return rubyMkRat(rubyToF(v), 1)
}

// ----- Complex#abs, which is hypot and not sqrt(re*re + im*im) --------------
//
// The plain sum SQUARES both parts, so it overflows to Infinity above ~1.3e154 and
// underflows to 0 below ~1.5e-154 where MRI answers the magnitude perfectly well.
//
// So the operands are SCALED BY A POWER OF TWO and the plain sum is kept, not
// replaced by the textbook `hi * sqrt(1 + (lo/hi)^2)`: that ratio form disagrees
// with MRI far more often, also in the ORDINARY range. Scaling by 2^k is EXACT
// (manual 7.7's %a argument), so the scaled operands carry the same significands
// and the same roundings as the unscaled ones - which is why the two paths agree
// bit-for-bit wherever both are defined, and the safe-band branch below is a SPEED
// guard and not a second answer. A ~1 ulp residue against MRI remains because
// Apple's libm hypot is not correctly rounded.
//
// math.Hypot is DELIBERATELY not used: it IS the ratio form, so it would put this twin one
// ulp away from layer 2 and the interpreter over the whole ordinary range. Spelling the
// same steps in all three engines is what makes them agree by construction. math.Ldexp
// is avoided for the same reason: it is not reachable from MetaJS, so the scale is built
// by the same loop there and here. The twins are hypotOf in ruby-interpreter.abnf and
// rbHypot in languages/lib/ruby-rt.metajs.
//
// 2^511 is the widest safe threshold: hi <= 2^511 keeps hi*hi + hi*hi under 2^1023, and
// hi >= 2^-511 keeps hi*hi at or above 2^-1022, so it never goes subnormal.
var rubyHypBig = func() float64 {
	r := 1.0
	for i := 0; i < 511; i++ {
		r = r * 2
	}
	return r
}()

// rubyHypScale is 2^k for the k that puts hi in [1, 2). Every value on the way is a power
// of two and therefore exact, at both ends of the range: k runs from 1023 down to -1074.
func rubyHypScale(hi float64) float64 {
	h, s := hi, 1.0
	for h >= 2 {
		h = h / 2
		s = s * 2
	}
	for h < 1 {
		h = h * 2
		s = s / 2
	}
	return s
}

func rubyHypot(a, b float64) float64 {
	x, y := math.Abs(a), math.Abs(b)
	// An infinity beats a NaN, which is IEEE-754's rule for hypot and MRI's:
	// Complex(Float::INFINITY, Float::NAN).abs is Infinity, not NaN.
	if math.IsInf(x, 0) {
		return x
	}
	if math.IsInf(y, 0) {
		return y
	}
	if x != x || y != y {
		return x + y
	}
	hi := x
	if y > x {
		hi = y
	}
	if hi == 0 {
		return 0
	}
	if hi <= rubyHypBig && hi >= 1/rubyHypBig {
		return math.Sqrt(x*x + y*y)
	}
	sc := rubyHypScale(hi)
	xs, ys := x/sc, y/sc
	return math.Sqrt(xs*xs+ys*ys) * sc
}

// rubyRatText is Rational#to_s (Complex#to_s is rubyCpxText in jsrtruby.go).
// Both parts go through rubyIntStr and NOT through jsNumString, because a Rational's
// parts are INTEGERS and jsNumString answers the SHORTEST ROUND-TRIPPING form, which past
// 2^53 is a different integer: 2^55 prints as "36028797018963970", while MRI's
// (0.001).to_r is 1152921504606847/1152921504606846976. rubyIntStr is the exact
// expansion.
func rubyRatText(t jsRat) string {
	if t.bn != nil {
		return t.bn.String() + "/" + t.bd.String()
	}
	return rubyIntStr(t.n) + "/" + rubyIntStr(t.d)
}

// rubyRatDigitOp is Rational#floor / #ceil / #round / #truncate, EXACTLY. MRI
// keeps the Rational at a positive ndigits - `(2r/3r).round(1)` is (7/10), not
// 0.7. The full rule
// are at ratDigitOp in languages/ruby-interpreter.abnf; rbRatDigitOp in
// languages/lib/ruby-rt.metajs is the other twin. `round` is half AWAY FROM ZERO
// and is written on integers, (2n +/- d) over 2d, so it does not depend on a
// double holding n/d exactly.
func rubyRatIntOp(n, d float64, mode byte) float64 {
	switch mode {
	case 'f':
		return math.Floor(n / d)
	case 'c':
		return math.Ceil(n / d)
	case 't':
		return math.Trunc(n / d)
	}
	if n >= 0 {
		return math.Floor((2*n + d) / (2 * d))
	}
	return math.Ceil((2*n - d) / (2 * d))
}

func (rt *jsrt) rubyRatDigitOp(v jsRat, mode byte, nd int) interface{} {
	if v.bn != nil || (nd > 0 && math.Abs(v.n*math.Pow(10, float64(nd))) > 9007199254740992) {
		n, d := rubyRatParts(v)
		if nd > 0 {
			p := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(nd)), nil)
			return rubyMkRatB(rubyRatIntOpB(new(big.Int).Mul(n, p), d, mode), p)
		}
		k := rt.rubyBigNarrow(rubyRatIntOpB(n, d, mode))
		if nd == 0 {
			return k
		}
		return rt.rubyIntDigitOp(k, -nd, mode)
	}
	if nd > 0 {
		p := math.Pow(10, float64(nd))
		return rubyMkRat(rubyRatIntOp(v.n*p, v.d, mode), p)
	}
	k := rt.rubyIntFromF(rubyRatIntOp(v.n, v.d, mode))
	if nd == 0 {
		return k
	}
	return rt.rubyIntDigitOp(k, -nd, mode)
}

// rubyFloStr renders a Float the way Ruby does: with a decimal point even when the
// value is whole (1000.0), which is what tells it apart from the Integer 1000, and
// in exponent form OUTSIDE ITS OWN WINDOW, which is not JavaScript's window.
// Ruby's rule is numeric.c flo_to_s: take the shortest
// round-tripping digit run and its decimal point position `decpt`, and go
// exponential when `decpt < -3 || (decpt > DBL_DIG (15) && decpt >= len(digits))`.
//
// The second half of the high test matters: MRI only goes exponential when the
// fixed form would have to PAD WITH ZEROS - i.e. when there is no fraction left
// to write. So 1234567890123456.0 (decpt 16, 16 digits) IS exponential and
// 3002399751580330.5 (decpt 16, 17 digits) is NOT. Float#to_s is unchanged
// between ruby 2.6 and 3.x:
//
//	1e15                1.0e+15
//	999999999999999.0   999999999999999.0
//	1234567890123456.0  1.234567890123456e+15
//	1e20                1.0e+20
//	1e-5                1.0e-05
//	0.0001              0.0001
//	-0.0                -0.0
//
// Note the two shapes JavaScript never writes: the mantissa always carries at least
// one fraction digit ("1.0e+15", not "1e+15") and the exponent is at least two
// digits and always signed ("e-05"). The JS twin is floStr in ruby-interpreter.abnf
// and rbFloStr in languages/lib/ruby-rt.metajs; all three MUST stay in step,
// because ./test.sh --cross diffs them.
func rubyFloStr(x float64) string {
	if x != x {
		return "NaN"
	}
	if math.IsInf(x, 1) {
		return "Infinity"
	}
	if math.IsInf(x, -1) {
		return "-Infinity"
	}
	if math.Signbit(x) {
		return "-" + rubyFloDigits(math.Abs(x))
	}
	return rubyFloDigits(x)
}

// rubyFloDigits renders a >= 0. strconv's 'e' form gives the shortest round-tripping
// digit run and the scientific exponent directly; `decpt` below is Ruby's own name
// for the decimal point position, which is that exponent plus one.
func rubyFloDigits(a float64) string {
	if a == 0 {
		return "0.0"
	}
	s := strconv.FormatFloat(a, 'e', -1, 64)
	i := strings.IndexByte(s, 'e')
	mant, exp := s[:i], s[i+1:]
	e10, _ := strconv.Atoi(exp)
	digits := strings.Replace(mant, ".", "", 1)
	decpt := e10 + 1
	if decpt < -3 || (decpt > 15 && decpt >= len(digits)) {
		out := digits[:1]
		if len(digits) > 1 {
			out += "." + digits[1:]
		} else {
			out += ".0"
		}
		sign := "+"
		if e10 < 0 {
			sign = "-"
			e10 = -e10
		}
		if e10 < 10 {
			return out + "e" + sign + "0" + strconv.Itoa(e10)
		}
		return out + "e" + sign + strconv.Itoa(e10)
	}
	if decpt <= 0 {
		return "0." + strings.Repeat("0", -decpt) + digits
	}
	if decpt >= len(digits) {
		return digits + strings.Repeat("0", decpt-len(digits)) + ".0"
	}
	return digits[:decpt] + "." + digits[decpt:]
}

// charCode unboxes a Char to its code and leaves every other value alone, so the
// arithmetic and comparison paths can treat a Char as its code without special cases.
func charCode(v interface{}) (interface{}, bool) {
	if c, ok := v.(jsChar); ok {
		return float64(c.code), true
	}
	return v, false
}

// bigPair answers the two operands as big.Ints when BOTH are BigInts, which is when
// an arithmetic operator has to stay in arbitrary precision.
func bigPair(a, b interface{}) (*big.Int, *big.Int, bool) {
	ab, aok := a.(*jsBigInt)
	if !aok {
		return nil, nil, false
	}
	bb, bok := b.(*jsBigInt)
	if !bok {
		return nil, nil, false
	}
	return ab.v, bb.v, true
}

// pyBigOperand answers an EXACT big.Int for a Python int operand: a BigInt as
// itself, and an integral double inside the exactly representable range as
// itself. A float box, a non-integral double and anything past 2^53 (where the
// double is already an approximation) answer false, so `**` falls back to the
// double path rather than inventing precision the operand never had.
// pyABigOperand in languages/lib/python-rt.metajs is the same test.
func pyBigOperand(v interface{}) (*big.Int, bool) {
	switch t := v.(type) {
	case *jsBigInt:
		return t.v, true
	case bool:
		if t {
			return big.NewInt(1), true
		}
		return big.NewInt(0), true
	case float64:
		if t == math.Trunc(t) && math.Abs(t) <= 9007199254740992 {
			return big.NewInt(int64(t)), true
		}
	}
	return nil, false
}

// rubyBigCmpOf is rubyBigOf for a COMPARISON, which may also take a Float:
// `2**64 == 1.8446744073709552e19` is true in MRI and `(2**53+1) ==
// (2**53+1).to_f` is false, and neither can be settled by rounding the Integer
// to a double. An arithmetic operator must NOT use this - there the Float wins
// the rank and the Integer converts, which is the tower's rule.
func rubyBigCmpOf(v interface{}) (*big.Int, bool) {
	if f, ok := v.(jsFlo); ok {
		if math.IsNaN(f.f) || math.IsInf(f.f, 0) || f.f != math.Trunc(f.f) {
			return nil, false
		}
		bf, _ := big.NewFloat(f.f).Int(nil)
		return bf, true
	}
	return rubyBigOf(v)
}

// rubyIntFromF builds an Integer from a double: exact past 2^53, so that the ONE
// shape invariant holds - an Integer of magnitude above 2^53 is always a big and
// never a plain double. Float#to_i and the rest of the truncating family go
// through this, which is also what keeps a big and a plain number from ever
// being two spellings of one Hash key.
func (rt *jsrt) rubyIntFromF(x float64) interface{} {
	if math.IsNaN(x) || math.IsInf(x, 0) || math.Abs(x) <= 9007199254740992 {
		return x
	}
	bf, _ := big.NewFloat(x).Int(nil)
	return rt.rubyBigNarrow(bf)
}

// pyBigNarrow is the Go twin of the interpreter's bigOut: a big that still fits
// in the exactly representable range comes back as a plain number, so only the
// values that GENUINELY need arbitrary precision carry the BigInt shape. Creating
// one sets hasBigInt, which is what lets the following operators reach bigArith.
func (rt *jsrt) pyBigNarrow(x *big.Int) interface{} {
	if x.IsInt64() {
		if n := x.Int64(); n <= 9007199254740992 && n >= -9007199254740992 {
			return float64(n)
		}
	}
	rt.hasBigInt = true
	return &jsBigInt{v: x}
}

// pyBigPair answers both operands as exact big.Ints when AT LEAST ONE of them is
// a BigInt and the other is an exact Python int. Python has ONE int type and it is
// arbitrary precision, so an int operand meeting a big PROMOTES; ECMAScript's
// BigInt refuses that mix, which is why bigPair above - shared with js_add,
// js_sub and the rest of the generic externals - demands two BigInts and this
// Python-only pair does not.
// pyBigFloPair recognises an arbitrary precision int and a float box as the two
// operands: the int, the double, and whether the int is the LEFT one.
func pyBigFloPair(a, b interface{}) (*big.Int, float64, bool, bool) {
	if x, ok := a.(*jsBigInt); ok {
		if y, ok2 := b.(*jsPyFlo); ok2 {
			return x.v, y.f, true, true
		}
	}
	if y, ok := b.(*jsBigInt); ok {
		if x, ok2 := a.(*jsPyFlo); ok2 {
			return y.v, x.f, false, true
		}
	}
	return nil, 0, false, false
}

func pyBigPair(a, b interface{}) (*big.Int, *big.Int, bool) {
	_, aBig := a.(*jsBigInt)
	_, bBig := b.(*jsBigInt)
	if !aBig && !bBig {
		return nil, nil, false
	}
	x, xok := pyBigOperand(a)
	if !xok {
		return nil, nil, false
	}
	y, yok := pyBigOperand(b)
	if !yok {
		return nil, nil, false
	}
	return x, y, true
}

// pyBitShiftMax is the largest shift count the bitwise arm will honour. CPython
// answers a MemoryError somewhere above this; a cap keeps `1 << 10**9` from
// asking for a gigabyte instead of aborting.
const pyBitShiftMax = 1 << 20

// pyBitBin is Python's BITWISE arithmetic, which is INFINITE TWO'S COMPLEMENT over
// arbitrary precision ints - exactly what math/big's And/Or/Xor/Lsh/Rsh implement
// (not ToInt32: `-1 & 0xffffffff` is 4294967295 and `1 << 40` is 1099511627776).
//
// ok is false for a shift count that is negative (CPython raises ValueError) or
// past the cap, so the caller can fail rather than answer a wrong number.
func pyBitBin(op string, x, y *big.Int) (*big.Int, bool) {
	out := new(big.Int)
	switch op {
	case "&":
		out.And(x, y)
	case "|":
		out.Or(x, y)
	case "^":
		out.Xor(x, y)
	case "<<", ">>":
		if y.Sign() < 0 || !y.IsInt64() || y.Int64() > pyBitShiftMax {
			return nil, false
		}
		n := uint(y.Int64())
		if op == "<<" {
			out.Lsh(x, n)
		} else {
			// big.Int.Rsh is an ARITHMETIC shift on the two's complement value,
			// which is Python's floor: -5 >> 1 is -3.
			out.Rsh(x, n)
		}
	default:
		return nil, false
	}
	return out, true
}

// pyBigBin is Python's integer arithmetic in arbitrary precision. `//` and `%`
// FLOOR, as Python's do: Go's Quo/Rem truncate and Go's Div/Mod are Euclidean, and
// neither is Python's answer for a negative divisor (1e30 % -7 is -6, not +1).
// ok is false for an operator with no exact integer form, so the caller falls
// through to the double path it always had.
// pyZeroDiv raises CPython's ZeroDivisionError for `/`, `//`, `%` and divmod by a
// numeric zero. Twin of pyZeroDiv in languages/python-interpreter.abnf and of the
// same arms of js_pybin in languages/lib/python-rt.metajs.
func (rt *jsrt) pyZeroDiv() {
	panic(&jsThrown{value: rt.pyExcInstance("ZeroDivisionError", "division by zero")})
}

func pyBigBin(op string, x, y *big.Int) (*big.Int, bool) {
	out := new(big.Int)
	switch op {
	case "+":
		out.Add(x, y)
	case "-":
		out.Sub(x, y)
	case "*":
		out.Mul(x, y)
	case "//", "%":
		if y.Sign() == 0 {
			return nil, false // Let the double path raise the usual error.
		}
		q, r := new(big.Int), new(big.Int)
		q.QuoRem(x, y, r)
		if r.Sign() != 0 && r.Sign() != y.Sign() {
			q.Sub(q, big.NewInt(1))
			r.Add(r, y)
		}
		if op == "//" {
			out = q
		} else {
			out = r
		}
	case "**":
		if y.Sign() < 0 || !y.IsInt64() || y.Int64() > 4096 {
			return nil, false
		}
		out.Exp(x, y, nil)
	default:
		return nil, false
	}
	return out, true
}

// bigArith is the BigInt path of the binary arithmetic externals. ok is false when
// the operands are not both BigInts, and the caller falls back to double arithmetic.
func bigArith(op byte, a, b interface{}) (res interface{}, ok bool) {
	x, y, both := bigPair(a, b)
	if !both {
		return nil, false
	}
	out := new(big.Int)
	switch op {
	case '+':
		out.Add(x, y)
	case '-':
		out.Sub(x, y)
	case '*':
		out.Mul(x, y)
	case '/':
		if y.Sign() == 0 {
			return nil, false // Let the double path produce the usual error/Inf.
		}
		out.Quo(x, y) // BigInt division truncates towards zero.
	case '%':
		if y.Sign() == 0 {
			return nil, false
		}
		out.Rem(x, y)
	default:
		return nil, false
	}
	return &jsBigInt{v: out}, true
}

// memberCall implements the method call convention that the class based
// language subsets (Java, Kotlin, Go) share: instances are objects with a
// "__class" property pointing to their class descriptor object, whose
// properties hold the method closures (called with the instance prepended to
// the arguments); the lookup follows the descriptor's "__super" chain, which
// gives Java its single inheritance. Objects without __class (class
// descriptors with static methods, plain objects with function properties)
// call the property directly. Strings get the Java style builtins.
func (rt *jsrt) memberCall(target interface{}, name string, args []interface{}) interface{} {
	switch o := target.(type) {
	case string:
		switch name {
		case "length":
			return float64(rt.strLen(o))
		case "charAt":
			i := jsToInt(rt.toNumber(argAt(args, 0)))
			ch := rt.strAt(o, i)
			if ch == "" {
				rt.fail("charAt(%d) out of range for %q", i, o)
			}
			return gojaCharAt(ch)
		case "equals":
			return rt.strictEq(o, argAt(args, 0))
		case "substring":
			begin, end := substringRange(rt.strLen(o), args, rt)
			return rt.strRange(o, begin, end)
		case "indexOf":
			return float64(rt.strIndexOf(o, rt.toString(argAt(args, 0))))
		case "isEmpty":
			return len(o) == 0
		}
		if v, ok := rt.mcallMiss(); ok {
			return v
		}
		rt.fail("unknown String method: %s", name)
	case *jsObject:
		if keys, vals, isDict := dictParts(o); isDict {
			// The Python dict methods.
			switch name {
			case "keys":
				return &jsArray{elems: append([]interface{}{}, keys.elems...)}
			case "values":
				return &jsArray{elems: append([]interface{}{}, vals.elems...)}
			case "get":
				if i := rt.pyDictFind(keys, argAt(args, 0)); i >= 0 {
					return vals.elems[i]
				}
				if len(args) > 1 {
					return args[1]
				}
				return jsUndef
			}
			if v, ok := rt.mcallMiss(); ok {
				return v
			}
			rt.fail("unknown dict method '%s'", name)
		}
		if cls, ok := o.props["__class"]; ok {
			// The lookup follows the __super chain (single inheritance).
			for cls != nil {
				clsObj, ok := cls.(*jsObject)
				if !ok {
					break
				}
				if m, ok := clsObj.props[name]; ok && isCallable(m) {
					return rt.call(m, jsUndef, append([]interface{}{target}, args...))
				}
				cls = clsObj.props["__super"]
			}
			if v, ok := rt.mcallMiss(); ok {
				return v
			}
			rt.fail("unknown method '%s' on an instance", name)
		}
		if m, ok := o.props[name]; ok && isCallable(m) {
			return rt.call(m, jsUndef, args)
		}
		if v, ok := rt.mcallMiss(); ok {
			return v
		}
		rt.fail("unknown method '%s'", name)
	case *jsGenerator:
		// A generator's protocol (next, return, ...) lives in its MEMBER TABLE - the
		// *jsGenerator case of getMember - and not in a property map, so a method call
		// resolves the member and calls it. Delegating rather than repeating the
		// member names here keeps the two in step and stays language neutral: every
		// member getMember exposes is callable as a method, in every grammar that
		// emits js_mcall. Without it `g.next()` failed with "method call on a object"
		// and each language had to spell the call out as js_get + js_call.
		if m := rt.getMember(o, name); isCallable(m) {
			return rt.call(m, jsUndef, args)
		}
		if v, ok := rt.mcallMiss(); ok {
			return v
		}
		rt.fail("unknown method '%s' on a generator", name)
	case *jsArray:
		// Kotlin and Python style list methods.
		switch name {
		case "add":
			o.dropIdx()
			o.elems = append(o.elems, argAt(args, 0))
			return true
		case "append": // Python: returns None.
			o.dropIdx()
			o.elems = append(o.elems, argAt(args, 0))
			return jsUndef
		case "pop", "removeLast": // Python / Dart: both remove and return the last element.
			if len(o.elems) == 0 {
				rt.fail("pop from empty list")
			}
			v := o.elems[len(o.elems)-1]
			o.dropIdx()
			o.elems = o.elems[:len(o.elems)-1]
			return v
		case "size":
			return float64(len(o.elems))
		case "get":
			i := int(rt.toNumber(argAt(args, 0)))
			if i < 0 || i >= len(o.elems) {
				rt.fail("list index %d out of range", i)
			}
			return o.elems[i]
		case "contains":
			for _, e := range o.elems {
				if rt.strictEq(e, argAt(args, 0)) {
					return true
				}
			}
			return false
		// The Kotlin higher order methods; the argument is a lambda (closure handle).
		case "map":
			out := &jsArray{}
			for _, e := range o.elems {
				out.elems = append(out.elems, rt.call(argAt(args, 0), jsUndef, []interface{}{e}))
			}
			return out
		case "filter":
			out := &jsArray{}
			for _, e := range o.elems {
				if rt.truthy(rt.call(argAt(args, 0), jsUndef, []interface{}{e})) {
					out.elems = append(out.elems, e)
				}
			}
			return out
		case "sumOf":
			var sum int32
			for _, e := range o.elems {
				sum += int32(int64(rt.toNumber(rt.call(argAt(args, 0), jsUndef, []interface{}{e}))))
			}
			return float64(sum)
		case "forEach":
			for _, e := range o.elems {
				rt.call(argAt(args, 0), jsUndef, []interface{}{e})
			}
			return jsUndef
		case "count":
			if len(args) == 0 {
				return float64(len(o.elems))
			}
			n := 0
			for _, e := range o.elems {
				if rt.truthy(rt.call(argAt(args, 0), jsUndef, []interface{}{e})) {
					n++
				}
			}
			return float64(n)
		case "any":
			for _, e := range o.elems {
				if rt.truthy(rt.call(argAt(args, 0), jsUndef, []interface{}{e})) {
					return true
				}
			}
			return false
		}
		if v, ok := rt.mcallMiss(); ok {
			return v
		}
		rt.fail("unknown list method '%s'", name)
	}
	if v, ok := rt.mcallMiss(); ok {
		return v
	}
	rt.fail("method call '%s' on a %s", name, rt.typeOf(target))
	return nil
}

// rubyTruthy is Ruby truthiness: only nil and false are falsy (0, "" and empty
// collections are all truthy). It backs the select/reject predicates below so
// js_rmcall agrees with the rtest of ruby-interpreter.abnf - unlike the JS
// truthiness that the shared memberCall/filter uses, where 0 would be falsy.
func rubyTruthy(v interface{}) bool {
	if isUndefOrNull(v) {
		return false
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return true
}

// rubyFloMod is Ruby's Float#%, and it is numeric.c flodivmod's body verbatim:
//
//	mod = fmod(x, y); if (y*mod < 0) mod += y;
//
// That one line gets all three behaviours at once. The DIVISOR's sign
// (-5.0 % 2.0 is 1.0, 5.0 % -2.0 is -1.0) comes from the correction; the
// DIVIDEND's sign on a ZERO (-4.0 % 2.0 is -0.0, -0.0 % 1.0 is -0.0) comes from
// fmod, because the correction cannot fire when mod is zero; and the huge
// dividends come from fmod being exact where `x - floor(x/y)*y` is not -
// floor(1e308/3)*3 rounds, so that form answers 0.0 for 1e308 % 3.0 where ruby
// answers 2.0.
//
// The infinite divisor falls out of the same line and needs no special case:
// fmod(-5, +Inf) is -5, +Inf * -5 is -Inf, so -5.0 % Infinity is Infinity, as
// real ruby says.
//
// The other two halves - numBin's `%` arm in languages/ruby-interpreter.abnf
// and rbNumBin's in languages/lib/ruby-rt.metajs - are the same body, plus a
// re-signing of the zero (`zeroSigned` / `rbZeroSign`, spelled `z / (0 - 1)`)
// that this half does not need: the tag engine keeps an integral value as an
// INTEGER and so drops the sign of a zero, where Go is IEEE all the way down
// and has math.Copysign. That is also why `x * 0` is not the spelling there.
//
// Reference rows (ruby 2.6.10, Float#% is unchanged in 3.x):
// 1e308 % 3.0 = 2.0, -1e308 % 3.0 = 1.0, 1e308 % 2.5 = 1.0,
// -1e308 % 2.5 = 1.5, 123456789012345678.0 % 7.0 = 3.0, 1e16 % 3.0 = 1.0 and
// -1e300 % 7.3 = 6.570033512034619.
func rubyFloMod(x, y float64) float64 {
	m := math.Mod(x, y)
	if m == 0 {
		return math.Copysign(0, x)
	}
	if y*m < 0 {
		return m + y
	}
	return m
}

// rubyNumBin is one arithmetic step of the numeric tower, mirroring numBin of
// ruby-interpreter.abnf: both operands promote to the higher rank, Integer / and %
// FLOOR (-7 / 2 == -4, -7 % 3 == 2) and Integer ** stays exact unless the exponent
// is negative. Reached only from rubyBin, i.e. only from the Ruby compiler.
func (rt *jsrt) rubyNumBin(op string, l, r interface{}) interface{} {
	rk, rr := rubyNumRank(l), rubyNumRank(r)
	if rk < 0 || rr < 0 {
		rt.fail("not a number in '%s'", op)
	}
	if rr > rk {
		rk = rr
	}
	if rk == 3 {
		return rt.rubyCpxBin(op, l, r)
	}
	if rk == 1 {
		p, q := rt.rubyAsRat(l), rt.rubyAsRat(r)
		switch op {
		case "+", "-", "*", "/":
			return rt.rubyRatArith(op, p, q)
		case "%":
			fq := rt.rubyRatArith("/", p, q)
			fn, fd := rubyRatParts(fq)
			return rt.rubyNumBin("-", p, rt.rubyNumBin("*", q, rt.rubyBigNarrow(rubyRatIntOpB(fn, fd, 'f'))))
		case "**":
			if k, ok := rt.rubyIntExponent(r); ok {
				return rt.rubyRatPow(p, k)
			}
		}
	}
	if rk >= 1 { // Float (and a Rational meeting an operator it has no exact form for).
		x, y := rubyToF(l), rubyToF(r)
		switch op {
		case "+":
			return jsFlo{f: x + y}
		case "-":
			return jsFlo{f: x - y}
		case "*":
			return jsFlo{f: x * y}
		case "/":
			return jsFlo{f: x / y}
		case "%":
			return jsFlo{f: rubyFloMod(x, y)}
		case "**":
			return jsFlo{f: math.Pow(x, y)}
		}
		rt.fail("Float does not support '%s'", op)
	}
	// Two Integers. One of them past 2^53 goes straight to the exact path;
	// otherwise the double answer is computed first and only PROMOTED when it
	// leaves the exact range, so a loop of small arithmetic pays one comparison.
	if out, ok := rt.rubyBigArith(op, l, r); ok {
		return out
	}
	x, y := rubyToF(l), rubyToF(r)
	switch op {
	case "+":
		if s := x + y; !rubyOver53(s) {
			return s
		}
		return rt.rubyBigPromote(op, x, y)
	case "-":
		if s := x - y; !rubyOver53(s) {
			return s
		}
		return rt.rubyBigPromote(op, x, y)
	case "*":
		if s := x * y; !rubyOver53(s) {
			return s
		}
		return rt.rubyBigPromote(op, x, y)
	case "/":
		if y == 0 {
			rt.rubyRaiseErr("ZeroDivisionError", "divided by 0")
		}
		return math.Floor(x / y)
	case "%":
		if y == 0 {
			rt.rubyRaiseErr("ZeroDivisionError", "divided by 0")
		}
		// math.FMA AND NOT `x - math.Floor(x/y)*y`: the product and the subtract must
		// round ONCE, so 9007199254740991 % -3 is -2 as in real ruby (rounding twice
		// gives -1). Layer 2 reproduces the fused result with rbFmaSub (Dekker
		// two-product / two-sum). Go may or may not contract the plain expression
		// into an FMA depending on the target; math.FMA IS the fused operation by
		// definition rather than by codegen.
		return math.FMA(-math.Floor(x/y), y, x)
	case "**":
		// A negative Integer power is an exact Rational (2 ** -1 is (1/2)).
		// ... except a base of 1 or -1, which MRI answers with an Integer.
		if y < 0 {
			if x == 1 || x == -1 {
				if math.Mod(y, 2) == 0 {
					return 1.0
				}
				return x
			}
			if y >= -4096 {
				return rt.rubyRatPow(rubyMkRat(x, 1), big.NewInt(int64(y)))
			}
			return jsFlo{f: math.Pow(x, y)}
		}
		if p := math.Pow(x, y); !rubyOver53(p) {
			return p
		}
		return rt.rubyBigPromote(op, x, y)
	// & | ^ << >> are INFINITE two's complement at ARBITRARY precision in Ruby:
	// `1 << 100` is a 31-digit Integer and `-1 & 0xffffffffffffffff` is
	// 18446744073709551615. int64 IS that answer for the range it holds, which is
	// what a bit-twiddling loop pays; outside it the operands promote.
	case "&", "|", "^":
		if x == math.Trunc(x) && y == math.Trunc(y) && math.Abs(x) <= 9007199254740992 && math.Abs(y) <= 9007199254740992 {
			var n int64
			switch op {
			case "&":
				n = int64(x) & int64(y)
			case "|":
				n = int64(x) | int64(y)
			default:
				n = int64(x) ^ int64(y)
			}
			// The RESULT has to be checked too, not just the operands: `1 |
			// 9007199254740992` is 2^53+1, which float64(n) rounds back down.
			// Layer 2 cannot make this test at all - its int64 comes back through
			// a double - so it takes the int32 fast path and the bignum, and the
			// two engines meet on the ANSWER rather than on the path.
			if n <= 9007199254740992 && n >= -9007199254740992 {
				return float64(n)
			}
		}
		return rt.rubyBigPromote(op, x, y)
	case "<<":
		if s := x * math.Pow(2, y); !rubyOver53(s) && y >= 0 && x == math.Trunc(x) {
			return s
		}
		return rt.rubyBigPromote(op, x, y)
	case ">>":
		if y >= 0 && y <= 53 && x == math.Trunc(x) && math.Abs(x) <= 9007199254740992 {
			return math.Floor(x / math.Pow(2, y))
		}
		return rt.rubyBigPromote(op, x, y)
	}
	rt.fail("Integer does not support '%s'", op)
	return nil
}

// rubyUserObj answers whether v is an instance of a user class (it carries a
// __class descriptor), which is what makes it dispatch its own operator methods.
func rubyUserObj(v interface{}) bool {
	o, ok := v.(*jsObject)
	if !ok {
		return false
	}
	_, has := o.props["__class"]
	return has
}

// ----- The PER-OBJECT singleton method table -----
//
// Ruby gives every object a singleton ("eigen") class in front of its own class.
// `def obj.m', `class << obj' and #define_singleton_method write the __sing slot
// (a plain object, name -> closure, with __singn holding the names in definition
// order); `obj.extend(M)' pushes the module onto __sext instead. The two tiers are
// separate because MRI keeps them apart - `def a.hi' then `a.extend(M)' still
// answers the def. Both slots are __-prefixed, so keysOf hides them (manual 7.9).
// Twins: singFind/singPut/singExtend of languages/ruby-interpreter.abnf and
// rbSingFind/rbSingPut/rbSingExtend of languages/lib/ruby-rt.metajs.
func rubySingFind(o *jsObject, name string) (interface{}, bool) {
	if t, isObj := o.props["__sing"].(*jsObject); isObj {
		if m, has := t.props[name]; has && isCallable(m) {
			return m, true
		}
	}
	if xs, isArr := o.props["__sext"].(*jsArray); isArr {
		for i := len(xs.elems) - 1; i >= 0; i-- {
			mod, isObj := xs.elems[i].(*jsObject)
			if !isObj {
				continue
			}
			if m, has := mod.props[name]; has && isCallable(m) {
				return m, true
			}
		}
	}
	return nil, false
}

func rubySingPut(o *jsObject, name string, fn interface{}) {
	t, isObj := o.props["__sing"].(*jsObject)
	if !isObj {
		t = newJSObject()
		o.set("__sing", t)
		o.set("__singn", &jsArray{})
	}
	if _, has := t.props[name]; !has {
		if ns, isArr := o.props["__singn"].(*jsArray); isArr {
			ns.elems = append(ns.elems, name)
		}
	}
	t.set(name, fn)
}

func rubySingExtend(o *jsObject, m interface{}) {
	mod, isObj := m.(*jsObject)
	if !isObj {
		return
	}
	xs, isArr := o.props["__sext"].(*jsArray)
	if !isArr {
		xs = &jsArray{}
		o.set("__sext", xs)
	}
	for _, e := range xs.elems {
		if e == interface{}(mod) {
			return
		}
	}
	xs.elems = append(xs.elems, mod)
}

// Object#singleton_methods: the direct defs, then each extended module's instance
// methods (the names the emitter records on the module object as __mnames), once each.
func (rt *jsrt) rubySingNames(o *jsObject) *jsArray {
	out := &jsArray{}
	seen := map[string]bool{}
	add := func(n string) {
		if !seen[n] {
			seen[n] = true
			out.elems = append(out.elems, jsSym{s: n})
		}
	}
	if ns, isArr := o.props["__singn"].(*jsArray); isArr {
		for _, e := range ns.elems {
			if s, isStr := e.(string); isStr {
				add(s)
			}
		}
	}
	// A class object's own `def self.` / `class << self` methods are singleton
	// methods of that object and MRI lists them; the names come from the emitter's
	// build-time record because a "$s$" key cannot be found by enumeration.
	if ns, isArr := o.props["__snames"].(*jsArray); isArr {
		for _, e := range ns.elems {
			if s, isStr := e.(string); isStr {
				add(s)
			}
		}
	}
	if xs, isArr := o.props["__sext"].(*jsArray); isArr {
		// NEAREST FIRST, the order rubySingFind searches. MRI's own order inside one
		// module is its method-table hash order, so only the tier order is settleable.
		for i := len(xs.elems) - 1; i >= 0; i-- {
			mod, isObj := xs.elems[i].(*jsObject)
			if !isObj {
				continue
			}
			if mn, isArr2 := mod.props["__mnames"].(*jsArray); isArr2 {
				for _, n := range mn.elems {
					if s, isStr := n.(string); isStr {
						add(s)
					}
				}
			}
		}
	}
	return out
}

// Object#dup / Object#clone. Both are SHALLOW: a fresh instance of the same class
// with the same instance variables. THE HIDDEN SLOTS ARE HANDLED DELIBERATELY here
// rather than by a generic key walk - manual 7.9 warns that this twin's `clone`
// walks o.keys RAW and copies __class by accident, which would carry __sing and
// __sext into a #dup and make it a #clone. The one difference between the two IS
// those slots: #clone keeps the singleton class, #dup drops it, and the tables are
// COPIED rather than shared so a def on the clone does not reach the original.
// `clone(freeze: false)` is not modelled and does
// not need to be: nothing here has a frozen bit to clear (Object#frozen? answers
// true only for the immediate values), so the keyword would be a no-op.
// Twins: singCopy of ruby-interpreter.abnf, rbSingCopy of lib/ruby-rt.metajs.
func rubySingCopy(o *jsObject, withSing bool) *jsObject {
	out := newJSObject()
	out.set("__class", o.props["__class"])
	for _, k := range o.keys {
		if len(k) >= 2 && k[0] == '_' && k[1] == '_' {
			continue
		}
		out.set(k, o.props[k])
	}
	// An exception's message travels with the copy.
	if a, has := o.props["__args"]; has {
		out.set("__args", a)
	}
	if !withSing {
		return out
	}
	if t, isObj := o.props["__sing"].(*jsObject); isObj {
		nt := newJSObject()
		for _, k := range t.keys {
			nt.set(k, t.props[k])
		}
		out.set("__sing", nt)
		if ns, isArr := o.props["__singn"].(*jsArray); isArr {
			out.set("__singn", &jsArray{elems: append([]interface{}{}, ns.elems...)})
		}
	}
	if xs, isArr := o.props["__sext"].(*jsArray); isArr {
		out.set("__sext", &jsArray{elems: append([]interface{}{}, xs.elems...)})
	}
	return out
}

// Object#respond_to?: the singleton table, then the class chain (an instance) or
// the $s$ class-method chain (a class object). It was absent from all four engines.
func rubySingRespond(o *jsObject, name string) bool {
	if _, ok := rubySingFind(o, name); ok {
		return true
	}
	if isCls, _ := o.props["__isclass"].(bool); isCls {
		for cls := interface{}(o); cls != nil; {
			clsObj, isObj := cls.(*jsObject)
			if !isObj {
				break
			}
			if m, has := clsObj.props["$s$"+name]; has && isCallable(m) {
				return true
			}
			cls = clsObj.props["__super"]
		}
		return false
	}
	_, ok := rubyFindMethod(o, name)
	return ok
}

// rubyFindMethod walks the singleton table and then the __class / __super chain of
// an instance for a method.
func rubyFindMethod(v interface{}, name string) (interface{}, bool) {
	o, ok := v.(*jsObject)
	if !ok {
		return nil, false
	}
	if m, has := rubySingFind(o, name); has {
		return m, true
	}
	for cls := o.props["__class"]; cls != nil; {
		clsObj, ok := cls.(*jsObject)
		if !ok {
			break
		}
		if m, ok := clsObj.props[name]; ok && isCallable(m) {
			return m, true
		}
		cls = clsObj.props["__super"]
	}
	return nil, false
}

// rubyLogicBin is rubyBin for the operators js_rband / js_rbor / js_rbxor
// lower: on true, false or nil, & | ^ are the NON-short-circuit logical operators
// over the right side's truthiness (nil acts as false). Twin of js_rband in
// lib/ruby-rt.metajs and rbin in ruby-interpreter.abnf.
func (rt *jsrt) rubyLogicBin(op string, l, r interface{}) interface{} {
	if op == "&" || op == "|" || op == "^" {
		if _, ok := l.(bool); ok || isUndefOrNull(l) {
			lb, rb := rubyTruthy(l), rubyTruthy(r)
			switch op {
			case "&":
				return lb && rb
			case "|":
				return lb || rb
			}
			return lb != rb
		}
	}
	return rt.rubyBin(op, l, r)
}

// rubyBin is the one binary operator step of ruby-to-llvm-ir.abnf (js_radd and
// friends below), mirroring radd/rbin of ruby-interpreter.abnf: a user object
// dispatches to its own operator method, strings and arrays get their Ruby
// operators, everything else is the numeric tower above.
func (rt *jsrt) rubyBin(op string, l, r interface{}) interface{} {
	if rubyUserObj(l) {
		// An exception instance that defines no + of its own concatenates as its
		// message, the same rule radd of ruby-interpreter.abnf applies - so
		// `raise e + "er"' inside a rescue keeps working.
		if op == "+" {
			if o, isObj := l.(*jsObject); isObj {
				if _, isExc := o.props["__args"]; isExc {
					if _, hasOp := rubyFindMethod(l, "+"); !hasOp {
						return strConcat(rt.rubyStr(l), rt.rubyStr(r))
					}
				}
			}
		}
		return rt.rubyMethod(l, op, []interface{}{r})
	}
	switch op {
	case "+":
		if _, ok := l.(string); ok {
			return strConcat(rt.rubyStr(l), rt.rubyStr(r))
		}
		if _, ok := r.(string); ok {
			return strConcat(rt.rubyStr(l), rt.rubyStr(r))
		}
		if la, ok := l.(*jsArray); ok {
			if ra, ok2 := r.(*jsArray); ok2 {
				out := &jsArray{}
				out.elems = append(append(out.elems, la.elems...), ra.elems...)
				return out
			}
		}
	case "*":
		if ls, ok := l.(string); ok {
			n := int(rubyToF(r))
			if n < 0 {
				n = 0
			}
			return strings.Repeat(ls, n)
		}
		if la, ok := l.(*jsArray); ok {
			// Array#* is repetition with an Integer and JOIN with a String.
			if sep, isStr := r.(string); isStr {
				return rt.rubyArrJoin(la, sep)
			}
			out := &jsArray{}
			for i := 0; i < int(rubyToF(r)); i++ {
				out.elems = append(out.elems, la.elems...)
			}
			return out
		}
	// Array#& (intersection) and Array#| (union). Both keep the LEFT-hand order,
	// both dedupe, and both compare with eql? like Array#- below. Without these
	// arms they reached rubyNumBin and ABORTED with "not a number in '&'".
	case "&":
		if la, ok := l.(*jsArray); ok {
			if ra, ok2 := r.(*jsArray); ok2 {
				return rt.rubyArrAnd(la, ra)
			}
		}
	case "|":
		if la, ok := l.(*jsArray); ok {
			if ra, ok2 := r.(*jsArray); ok2 {
				return rt.rubyArrOr(la, ra)
			}
		}
	case "-":
		if la, ok := l.(*jsArray); ok {
			if ra, ok2 := r.(*jsArray); ok2 {
				// Array#- is eql?/hash, like uniq and unlike include?: MRI answers
				// `[1, 2.0] - [2]` with [1, 2.0]. This was pyEqual, which converts
				// across the numeric classes and dropped the 2.0.
				out := &jsArray{}
				for _, e := range la.elems {
					drop := false
					for _, x := range ra.elems {
						if rt.rubyEql(e, x) {
							drop = true
							break
						}
					}
					if !drop {
						out.elems = append(out.elems, e)
					}
				}
				return out
			}
		}
	case "%":
		if ls, ok := l.(string); ok {
			return rt.rubyFormat(ls, r)
		}
	case "<<":
		if la, ok := l.(*jsArray); ok {
			la.dropIdx()
			la.elems = append(la.elems, r)
			return la
		}
		if ls, ok := l.(string); ok {
			return strConcat(ls, rt.rubyStr(r))
		}
	}
	return rt.rubyNumBin(op, l, r)
}

// rubyNumMethod is the Integer / Float / Rational / Complex method set of
// js_rmcall. It answers ok=false for a name it does not know, so the caller can
// carry on with the generic dispatch (and its error message).
func (rt *jsrt) rubyNumMethod(t interface{}, name string, args []interface{}) (interface{}, bool) {
	if bi, isBig := t.(*jsBigInt); isBig {
		if out, ok := rt.rubyBigMethod(bi, name, args); ok {
			return out, true
		}
	}
	if c, isCpx := t.(jsCpx); isCpx {
		if out, ok := rt.rubyCpxMethod(c, name); ok {
			return out, true
		}
	}
	x := rubyToF(t)
	_, isInt := t.(float64)
	_, isBig := t.(*jsBigInt)
	// Integer's own methods, which Float, Rational and Complex do not have.
	if !isInt && !isBig {
		switch name {
		case "even?", "odd?", "succ", "next", "pred", "chr", "times", "upto", "downto", "gcd", "pow":
			return nil, false
		}
	}
	switch name {
	case "to_s", "inspect":
		if isInt && len(args) > 0 {
			// Past 2^53 the int64 is not the value, so the exact expansion does
			// the base conversion instead - see rubyIntStr for the same reason.
			if math.Abs(x) > 9007199254740992 && x == math.Trunc(x) {
				bf, _ := big.NewFloat(x).Int(nil)
				return bf.Text(int(rubyToF(args[0]))), true
			}
			return strconv.FormatInt(int64(x), int(rubyToF(args[0]))), true
		}
		// #inspect is NOT #to_s for a Rational or a Complex - it parenthesizes.
		if name == "inspect" {
			return rt.rubyInspect(t), true
		}
		return rt.rubyStr(t), true
	// #truncate is not here: it takes an ndigits like #floor and #round
	// (`1.567.truncate(1)` is the Float 1.5).
	case "to_i", "to_int":
		if r, ok := t.(jsRat); ok {
			return rt.rubyRatDigitOp(r, 't', 0), true
		}
		return rt.rubyIntFromF(math.Trunc(x)), true
	case "to_f":
		if f, ok := t.(jsFlo); ok {
			return f, true
		}
		return jsFlo{f: x}, true
	case "to_r":
		return rt.rubyAsRat(t), true
	case "to_c":
		if c, ok := t.(jsCpx); ok {
			return c, true
		}
		return jsCpx{re: t, im: 0.0}, true
	// Rational#abs stays a RATIONAL (Complex#abs is rubyCpxAbs).
	case "abs", "magnitude":
		if r, ok := t.(jsRat); ok {
			if r.n < 0 {
				return rubyRatNeg(r), true
			}
			return r, true
		}
		if _, ok := t.(jsFlo); ok {
			return jsFlo{f: math.Abs(x)}, true
		}
		return math.Abs(x), true
	// #floor / #ceil / #round all take an optional ndigits, and only a POSITIVE
	// one answers a Float (and does not raise on an infinity); 0 and negatives
	// answer an Integer. rubyNumDigitOp is the whole rule.
	case "floor", "ceil", "round", "truncate":
		nd := 0
		if len(args) > 0 {
			nd = int(math.Trunc(rubyToF(args[0])))
		}
		mode := byte('r')
		switch name {
		case "floor":
			mode = 'f'
		case "ceil":
			mode = 'c'
		case "truncate":
			mode = 't'
		}
		// A Rational stays EXACT at every ndigits - see rubyRatDigitOp.
		if r, ok := t.(jsRat); ok {
			return rt.rubyRatDigitOp(r, mode, nd), true
		}
		return rt.rubyNumDigitOp(x, nil, isInt, mode, nd), true
	case "zero?":
		return x == 0, true
	case "positive?":
		return x > 0, true
	case "negative?":
		return x < 0, true
	case "even?":
		return math.Mod(x, 2) == 0, true
	case "odd?":
		return math.Mod(x, 2) != 0, true
	case "succ", "next":
		return rt.rubyNumBin("+", t, float64(1)), true
	case "pred":
		return rt.rubyNumBin("-", t, float64(1)), true
	// Integer#chr: one byte, so 0..255; anything else is a RangeError.
	case "chr":
		if x < 0 || x > 255 {
			rt.rubyRaiseErr("RangeError", rubyIntStr(x)+" out of char range")
		}
		return string(rune(int32(x))), true
	case "integer?":
		return isInt, true
	case "numerator":
		return rt.rubyRatPart(rt.rubyAsRat(t), true), true
	case "denominator":
		return rt.rubyRatPart(rt.rubyAsRat(t), false), true
	case "times": // n.times { |i| ... }
		_, tblk := rubyBlockArgs(args)
		for i := 0.0; i < x; i++ {
			rt.call(tblk, jsUndef, []interface{}{i})
		}
		return t, true
	case "upto":
		upos, ublk := rubyBlockArgs(args)
		for i := x; i <= rubyToF(argAt(upos, 0)); i++ {
			rt.call(ublk, jsUndef, []interface{}{i})
		}
		return t, true
	case "downto":
		dpos, dblk := rubyBlockArgs(args)
		for i := x; i >= rubyToF(argAt(dpos, 0)); i-- {
			rt.call(dblk, jsUndef, []interface{}{i})
		}
		return t, true
	case "div", "divmod", "modulo", "rationalize":
		return rt.rubyNumDivOp(t, name, args), true
	case "fdiv":
		if _, isCpx := t.(jsCpx); !isCpx {
			return jsFlo{f: rubyIntFdiv(t, argAt(args, 0))}, true
		}
	case "**":
		return rt.rubyBin("**", t, argAt(args, 0)), true
	// between? / clamp are not here: they are Comparable's, shared with String,
	// and must be exact on a big Integer - rubyMethod answers both through
	// rubySpaceship before this table is reached.
	case "real":
		return rubyReOf(t), true
	case "imaginary", "imag":
		return rubyImOf(t), true
	case "+", "-", "*", "/", "%", "&", "|", "^", "<<", ">>":
		return rt.rubyBin(name, t, argAt(args, 0)), true
	case "==":
		return rt.rubyEq(t, argAt(args, 0)), true
	case "<=>":
		c, ok := rt.rubySpaceship(t, argAt(args, 0))
		if !ok {
			return jsNull, true
		}
		return float64(c), true
	}
	return rt.rubyNumMore(t, name, args)
}

// rubyBigMethod is the Integer method set for a value past 2^53, and the twin of
// bigMethod in ruby-interpreter.abnf and rbABigMethod in ruby-rt.metajs. Every
// step is routed through the exact arithmetic instead of the double
// (`(2**64).abs` is 18446744073709551616). ok=false hands
// the name back to rubyNumMethod's own list, which answers it in double
// precision (times / upto / to_r and the rest) or reports it unknown.
func (rt *jsrt) rubyBigMethod(t *jsBigInt, name string, args []interface{}) (interface{}, bool) {
	switch name {
	case "to_s", "inspect":
		if len(args) > 0 {
			return t.v.Text(int(rubyToF(args[0]))), true
		}
		return t.v.String(), true
	// An Integer is already floored, ceilinged, truncated and rounded, at any
	// digit argument a Ruby program can write here.
	case "to_i", "to_int":
		return t, true
	// ...at every ndigits >= 0; a NEGATIVE one rounds to a multiple of
	// 10**-ndigits, which past 2**53 has to be exact.
	case "floor", "ceil", "round", "truncate":
		nd := 0
		if len(args) > 0 {
			nd = int(math.Trunc(rubyToF(args[0])))
		}
		mode := byte('r')
		switch name {
		case "floor":
			mode = 'f'
		case "ceil":
			mode = 'c'
		case "truncate":
			mode = 't'
		}
		return rt.rubyNumDigitOp(0, t, true, mode, nd), true
	case "abs", "magnitude":
		return rt.rubyBigNarrow(new(big.Int).Abs(t.v)), true
	case "zero?":
		return t.v.Sign() == 0, true
	case "positive?":
		return t.v.Sign() > 0, true
	case "negative?":
		return t.v.Sign() < 0, true
	case "even?":
		return t.v.Bit(0) == 0, true
	case "odd?":
		return t.v.Bit(0) == 1, true
	case "succ", "next":
		return rt.rubyBigNarrow(new(big.Int).Add(t.v, big.NewInt(1))), true
	case "pred":
		return rt.rubyBigNarrow(new(big.Int).Sub(t.v, big.NewInt(1))), true
	case "integer?":
		return true, true
	case "divmod":
		if q, ok := rt.rubyBigArith("/", t, argAt(args, 0)); ok {
			m, _ := rt.rubyBigArith("%", t, argAt(args, 0))
			return &jsArray{elems: []interface{}{q, m}}, true
		}
	}
	return nil, false
}

// rubyIdenticalP is BasicObject#equal?, OBJECT IDENTITY.
//
// The rule is NOT a bare `==` and the measured row table is at identicalP in
// languages/ruby-interpreter.abnf; rbIdenticalP in languages/lib/ruby-rt.metajs is
// the other twin. Three kinds of value need saying:
//
//	a jsFlo compares by its DOUBLE, because MRI's Float is a flonum and
//	1.0.equal?(1.0) is true there; NaN's `false` falls out of that.
//
//	a string, a jsRat and a jsCpx answer FALSE always. MRI allocates a fresh
//	object per value ("a".equal?("a") and (1r/3r).equal?(1r/3r) are both false)
//	and none of the three has identity in this value model. THIS FILE IS WHY IT
//	CANNOT BE `==`: jsRat and jsCpx are comparable Go STRUCTS, so two equal
//	Rationals are `==` here and two distinct box objects in the other two
//	engines - a designed-in halves divergence if it were left to `==`.
//
//	everything else is a reference or an interned value and `==` IS identity: a
//	float64 Integer, a jsSym, nil, a bool, a *jsArray / *jsObject / *jsBigInt and
//	every callable (all three callable representations are pointer types, which
//	is what rt.rubyLambdas already relies on to key a map by them).
func rubyIdenticalP(a, b interface{}) bool {
	if isUndefOrNull(a) || isUndefOrNull(b) {
		return isUndefOrNull(a) && isUndefOrNull(b)
	}
	af, aIsFlo := a.(jsFlo)
	bf, bIsFlo := b.(jsFlo)
	if aIsFlo || bIsFlo {
		return aIsFlo && bIsFlo && af.f == bf.f
	}
	switch a.(type) {
	case string, jsRat, jsCpx:
		return false
	}
	switch b.(type) {
	case string, jsRat, jsCpx:
		return false
	}
	return a == b
}

// rubyClassName is Ruby's #class for the values that are not user instances: the
// name of the builtin class a runtime value belongs to ("" when there is none).
func rubyClassName(v interface{}) string {
	if b, ok := v.(bool); ok {
		if b {
			return "TrueClass"
		}
		return "FalseClass"
	}
	switch t := v.(type) {
	case float64:
		return "Integer"
	case *jsBigInt:
		return "Integer"
	case jsFlo:
		return "Float"
	case jsRat:
		return "Rational"
	case jsCpx:
		return "Complex"
	case string:
		return "String"
	case jsSym:
		return "Symbol"
	case jsNullT, jsUndefT:
		return "NilClass"
	case *jsArray:
		return "Array"
	case *jsObject:
		if _, _, isDict := dictParts(t); isDict {
			return "Hash"
		}
		if rubyIsEnum(t) {
			return "Enumerator"
		}
		// A Range is a VALUE, so it has a class like everything else. This one arm
		// gates `.class', `is_a?(Range)', `case r when Range', eql? and therefore
		// the hash-key comparison, and the emitter's own `*r' splat test.
		if _, isRng := t.props["__rrange"]; isRng {
			return "Range"
		}
		if rxIsRegexObj(t) {
			return "Regexp"
		}
		if rxIsMatchObj(t) {
			return "MatchData"
		}
	}
	// A Proc / lambda / block closure: `blk.class`, `blk.is_a?(Proc)`.
	if isCallable(v) {
		return "Proc"
	}
	return ""
}

// rubyClassNameRT is rubyClassName with the ONE answer that needs runtime state:
// a Method is a class of its own in MRI and is NOT a Proc, and which callables are
// Methods lives in rt.rubyMethodVals. Every #class / is_a? / case-when path goes
// through this; rubyClassName stays free of the receiver for the callers that have
// no jsrt to hand.
func (rt *jsrt) rubyClassNameRT(v interface{}) string {
	// THE TYPE SWITCH IS NOT AN OPTIMISATION. This is reached with every receiver
	// a Ruby `.class` / `is_a?` / `case when` names, and a Go map lookup PANICS on
	// an interface holding a non-comparable dynamic type - isCallable's last line
	// accepts a bare reflect.Func, so "it is callable" is not enough. The three
	// pointer types below are the ones every callable this runtime mints has, and
	// the ones rubyMarkMethodVal will store.
	if _, isMeth := rt.rubyMethodValOf(v); isMeth {
		return "Method"
	}
	return rubyClassName(v)
}

// rubyBuiltinClass hands out the ONE class object of a builtin class name, so that
// `1.class == Integer` and `case v when Integer` compare the same object. The
// descriptor carries __isclass (like a compiled class) plus __rbuiltin, which is
// what rubyIsA keys the type test on.
func (rt *jsrt) rubyBuiltinClass(name string) *jsObject {
	if rt.rubyClasses == nil {
		rt.rubyClasses = map[string]*jsObject{}
	}
	if c, ok := rt.rubyClasses[name]; ok {
		return c
	}
	c := newJSObject()
	c.set("__isclass", true)
	c.set("__rbuiltin", name)
	c.set("__name", name)
	// The builtin names that are modules, not classes: their #class is Module.
	// Twins: rbBuiltinClass in languages/lib/ruby-rt.metajs, isBuiltinModule in
	// ruby-interpreter.abnf.
	switch name {
	case "Kernel", "Comparable", "Enumerable", "Math":
		c.set("__ismodule", true)
	}
	// Math::PI and Math::E. A scoped constant is a "$c$"-prefixed slot on the
	// namespace object (makeScopedConst in ruby-to-llvm-ir.abnf, lookupConst in
	// ruby-interpreter.abnf).
	if name == "Math" {
		c.set("$c$PI", jsFlo{f: math.Pi})
		c.set("$c$E", jsFlo{f: math.E})
		c.set("$c$DomainError", rt.rubyBuiltinClass("Math::DomainError"))
	}
	// Regexp's option bits, the second argument of Regexp.new.
	if name == "Regexp" {
		c.set("$c$IGNORECASE", 1.0)
		c.set("$c$EXTENDED", 2.0)
		c.set("$c$MULTILINE", 4.0)
	}
	rt.rubyClasses[name] = c
	return c
}

// rubyMathCall is Ruby's Math module: every method answers a FLOAT, which is the
// whole reason it cannot simply be JavaScript's Math (Ruby's Math.sqrt(4) is 2.0,
// JavaScript's is 2). Math.log takes an optional base, which JavaScript's does
// not. The other two engines are rbMathCall in languages/lib/ruby-rt.metajs and
// the "$s$" methods on hostGlobals["Math"] in languages/ruby-interpreter.abnf.
//
// All three compute with Go's math algorithms (runtime.c ports them line for
// line), so they agree to the bit. MRI calls the PLATFORM libm instead, and on
// macOS its last bit matches neither Go's algorithm, nor fdlibm, nor the correctly
// rounded value, so no port can reproduce it: Math.atan2 / Math.atan one ulp from
// MRI on some arguments is expected.
func rubyMathCall(name string, rt *jsrt, args []interface{}) (interface{}, bool) {
	a := func(i int) float64 { return rt.rubyMathArg(name, rt.toNumber(argAt(args, i))) }
	one := map[string]func(float64) float64{
		"sqrt": math.Sqrt, "cbrt": math.Cbrt, "sin": math.Sin, "cos": math.Cos,
		"tan": math.Tan, "asin": math.Asin, "acos": math.Acos, "atan": math.Atan,
		"sinh": math.Sinh, "cosh": math.Cosh, "tanh": math.Tanh, "asinh": math.Asinh,
		"acosh": math.Acosh, "atanh": math.Atanh, "exp": math.Exp,
		"log2": math.Log2, "log10": math.Log10,
	}
	if fn, ok := one[name]; ok {
		return jsFlo{f: fn(a(0))}, true
	}
	switch name {
	case "atan2":
		return jsFlo{f: math.Atan2(a(0), a(1))}, true
	case "hypot":
		return jsFlo{f: math.Hypot(a(0), a(1))}, true
	case "log":
		if len(args) > 1 {
			return jsFlo{f: math.Log(a(0)) / math.Log(a(1))}, true
		}
		return jsFlo{f: math.Log(a(0))}, true
	}
	return nil, false
}

// rubyMathArg raises MRI's Math::DomainError for an argument outside the
// function's real domain (a NaN is in it). Twins: mathArg in
// ruby-interpreter.abnf, rbMathArg in languages/lib/ruby-rt.metajs.
func (rt *jsrt) rubyMathArg(name string, x float64) float64 {
	bad := false
	switch name {
	case "sqrt", "log", "log2", "log10":
		bad = x < 0
	case "asin", "acos", "atanh":
		bad = x < -1 || x > 1
	case "acosh":
		bad = x < 1
	}
	if bad {
		rt.rubyRaiseErr("Math::DomainError", "Numerical argument is out of domain - \""+name+"\"")
	}
	return x
}

// ----- Ruby ranges -----
// EVERY range is a {__rrange, begin, end, excl} VALUE, bounded or open.
// rubyRangeArr is the materializer - for numbers and for the String ranges
// "a".."c" alike - and runs ON DEMAND, out of rubyRangeToArr; the Range method
// table delegates the whole Enumerable surface to the Array's. Twins:
// rbRangeToArr / rbRangeMethod in languages/lib/ruby-rt.metajs, rangeToA /
// rangeMethod in languages/ruby-interpreter.abnf.

// rubyStrSucc is String#succ for the simple alphanumeric case: the last character
// is incremented, carrying into the one before it when it wraps ("az" -> "ba").
func rubyStrSucc(s string) string {
	if s == "" {
		return ""
	}
	b := []byte(s)
	for i := len(b) - 1; i >= 0; i-- {
		switch {
		case b[i] >= 'a' && b[i] < 'z', b[i] >= 'A' && b[i] < 'Z', b[i] >= '0' && b[i] < '9':
			b[i]++
			return string(b)
		case b[i] == 'z':
			b[i] = 'a'
		case b[i] == 'Z':
			b[i] = 'A'
		case b[i] == '9':
			b[i] = '0'
		default:
			b[i]++
			return string(b)
		}
	}
	return string(b[0]) + string(b)
}

// rubyRangeArr materializes lo..hi (excl: the three-dot form) as an array. The
// step cap keeps a nonsensical range from hanging the program.
func (rt *jsrt) rubyRangeArr(lo, hi interface{}, excl bool) *jsArray {
	out := &jsArray{}
	if ls, isStr := lo.(string); isStr {
		hs := rt.toString(hi)
		for s, n := ls, 0; n < 1000000; n++ {
			if excl && s == hs {
				break
			}
			if len(s) > len(hs) {
				break
			}
			out.elems = append(out.elems, s)
			if s == hs {
				break
			}
			s = rubyStrSucc(s)
		}
		return out
	}
	l, h := rubyToF(lo), rubyToF(hi)
	if excl {
		h = h - 1
	}
	for i, n := l, 0; i <= h && n < 10000000; i, n = i+1, n+1 {
		out.elems = append(out.elems, i)
	}
	return out
}

// rubyRangeValue is what a range LITERAL becomes: the object, always. The object
// carries its own raw endpoints, which `s[1..-1]' needs.
func (rt *jsrt) rubyRangeValue(lo, hi interface{}, excl bool) interface{} {
	return rubyOpenRange(lo, hi, excl)
}

// rubyOpenRange is THE Range constructor: 1..3, 1...3, (5..), (..9).
func rubyOpenRange(lo, hi interface{}, excl bool) *jsObject {
	o := newJSObject()
	o.set("__rrange", true)
	o.set("begin", lo)
	o.set("end", hi)
	o.set("excl", excl)
	return o
}

func rubyIsRange(v interface{}) (*jsObject, bool) {
	o, isObj := v.(*jsObject)
	if !isObj {
		return nil, false
	}
	_, isRng := o.props["__rrange"]
	return o, isRng
}

// rubyRangeText: #to_s renders its endpoints, #inspect inspects them. That is the
// only place the two differ - `("a".."e").inspect' is `"a".."e"' and its #to_s is
// `a..e'.
func (rt *jsrt) rubyRangeText(r *jsObject, insp bool) string {
	dots := ".."
	if excl, _ := r.props["excl"].(bool); excl {
		dots = "..."
	}
	one := func(v interface{}) string {
		if isUndefOrNull(v) {
			return ""
		}
		if insp {
			return rt.rubyInspect(v)
		}
		return rt.rubyStr(v)
	}
	return one(r.props["begin"]) + dots + one(r.props["end"])
}

// The element sequence. An ENDLESS range has none, and asking for one is an error
// in MRI too; the message is ruby-interpreter.abnf's, so all three engines agree.
func (rt *jsrt) rubyRangeToArr(r *jsObject) *jsArray {
	lo, hi := r.props["begin"], r.props["end"]
	if isUndefOrNull(lo) || isUndefOrNull(hi) {
		rt.fail("cannot convert an endless range to an array")
	}
	excl, _ := r.props["excl"].(bool)
	return rt.rubyRangeArr(lo, hi, excl)
}

func rubyRangeNumeric(r *jsObject) bool {
	_, isStr := r.props["begin"].(string)
	return !isStr
}

// rubyRangeLen / rubyRangeAt are what js_pylen / js_pyget answer for a range, and
// they are ARITHMETIC for the numeric case on purpose: for-in, splat and
// destructuring ask for the length once and then index element by element, so
// materializing inside rubyRangeAt would make every one of them quadratic.
// THE ARITHMETIC HERE MUST BE rubyRangeArr'S, EXACTLY. rubyRangeArr walks
// `i = lo; i <= hi; i++', so element i is lo + i and the count is
// floor(hi - lo) + 1 - NOT trunc(hi) - trunc(lo) + 1, which is the same for
// integers and differs for a FLOAT range: `(1.5..3.5).to_a' is [1.5, 2.5, 3.5]
// here. (MRI raises TypeError for both - a known divergence of rubyRangeArr -
// but the two paths of ONE value model have to agree with each other.)
func rubyRangeSpan(r *jsObject) float64 {
	h := rubyToF(r.props["end"])
	if excl, _ := r.props["excl"].(bool); excl {
		h = h - 1
	}
	return h - rubyToF(r.props["begin"])
}

func (rt *jsrt) rubyRangeLen(r *jsObject) float64 {
	lo, hi := r.props["begin"], r.props["end"]
	if isUndefOrNull(lo) || isUndefOrNull(hi) {
		rt.fail("cannot take the length of an endless range")
	}
	if !rubyRangeNumeric(r) {
		return float64(len(rt.rubyRangeToArr(r).elems))
	}
	if n := math.Floor(rubyRangeSpan(r)) + 1; n > 0 {
		return n
	}
	return 0
}

func (rt *jsrt) rubyRangeAt(r *jsObject, i int) interface{} {
	if !rubyRangeNumeric(r) {
		a := rt.rubyRangeToArr(r)
		if i < 0 || i >= len(a.elems) {
			rt.fail("range index out of range: %d", i)
		}
		return a.elems[i]
	}
	if i < 0 || float64(i) >= rt.rubyRangeLen(r) {
		rt.fail("range index out of range: %d", i)
	}
	return rubyToF(r.props["begin"]) + float64(i)
}

// Two ranges are == when their endpoints and their exclusivity are; MRI compares
// `1..3' and `1...3' as different, and a Range is never == to its own to_a.
func (rt *jsrt) rubyRangeEq(l, r *jsObject) bool {
	le, _ := l.props["excl"].(bool)
	re, _ := r.props["excl"].(bool)
	if le != re {
		return false
	}
	return rt.rubyEq(l.props["begin"], r.props["begin"]) && rt.rubyEq(l.props["end"], r.props["end"])
}

// The Range method table. Everything a Range can answer that an Array cannot is
// here; EVERYTHING ELSE MATERIALIZES AND DELEGATES to rubyArrayMethod, which is
// what keeps the whole Enumerable surface (each / map / select / sum / reduce /
// zip / group_by / each_slice / find / reverse_each / …) working without a second
// copy of it. Twin: rbRangeMethod in languages/lib/ruby-rt.metajs.
func (rt *jsrt) rubyRangeMethod(t *jsObject, name string, args []interface{}) interface{} {
	pos, blk := rubyBlockArgs(args)
	endless := isUndefOrNull(t.props["begin"]) || isUndefOrNull(t.props["end"])
	switch name {
	// Membership stays LAZY - it is the one part an endless range can answer, and
	// materializing (1..1_000_000) to ask `include?(3)' would be absurd.
	case "cover?", "include?", "member?", "===":
		return rt.rubyOpenRangeCover(t, argAt(args, 0))
	case "exclude_end?":
		b, _ := t.props["excl"].(bool)
		return b
	case "begin":
		return t.props["begin"]
	// #end and #last with NO argument are the range's END, not its last element:
	// `(1...3).last' is 3 where `(1...3).to_a.last' is 2. With an argument they are
	// the Array's, so those fall through to the delegation below.
	case "end":
		return t.props["end"]
	case "to_s":
		return rt.rubyRangeText(t, false)
	case "inspect":
		return rt.rubyRangeText(t, true)
	case "class":
		return rt.rubyBuiltinClass("Range")
	case "is_a?", "kind_of?":
		return rt.rubyIsA(t, argAt(args, 0))
	case "instance_of?":
		a0, isObj := argAt(args, 0).(*jsObject)
		if !isObj {
			return false
		}
		bn, _ := a0.props["__rbuiltin"].(string)
		return bn == "Range"
	case "nil?", "frozen?":
		return false
	case "hash":
		return rt.rubyHashOf(t)
	case "object_id":
		return rt.rubyObjectId(t)
	case "dup", "clone", "freeze", "itself":
		return t
	case "==", "eql?", "equal?":
		o, isRng := rubyIsRange(argAt(args, 0))
		if !isRng {
			return false
		}
		return rt.rubyRangeEq(t, o)
	case "to_a", "entries", "to_ary":
		return rt.rubyRangeToArr(t)
	case "first":
		if len(pos) == 0 {
			return t.props["begin"]
		}
	case "last":
		if len(pos) == 0 {
			return t.props["end"]
		}
	// Range#step(n): every n-th element.
	// Without a block MRI answers an ArithmeticSequence; the nearest thing in this
	// value model is an Enumerator, which answers #to_a and #each the same way.
	case "step":
		by := int(math.Trunc(rubyToF(argAt(pos, 0))))
		if by <= 0 {
			rt.rubyRaiseErr("ArgumentError", "step can't be 0 or negative")
		}
		src := rt.rubyRangeToArr(t)
		picked := &jsArray{}
		for i := 0; i < len(src.elems); i += by {
			picked.elems = append(picked.elems, src.elems[i])
		}
		if blk != nil {
			for _, v := range picked.elems {
				rt.call(blk, jsUndef, []interface{}{v})
			}
			return t
		}
		return rubyMkEnum(picked)
	}
	// An endless range has no sequence, so only the two questions that do not need
	// one can be answered; MRI agrees that (1..).min is 1 and (1..).max raises.
	if endless {
		switch name {
		case "min":
			return t.props["begin"]
		case "size", "length", "count":
			rt.fail("cannot take the length of an endless range")
		}
	}
	if len(pos) == 0 && blk == nil {
		switch name {
		// #size, #length and #count all answer the ELEMENT COUNT, and that is a
		// DELIBERATE divergence - see the twin in languages/lib/ruby-rt.metajs
		// for why splitting them is not worth it.
		case "size", "length", "count":
			return rt.rubyRangeLen(t)
		}
	}
	av := rt.rubyDelegArr(rt.rubyRangeToArr(t), t, name, args)
	// #each answers the RECEIVER in MRI, and the receiver is the range.
	if name == "each" && blk != nil {
		return t
	}
	return av
}

// rubyOpenRangeCover is `r.cover?(v)' / `r === v' for ANY range, bounded or
// open, exclusive end included.
func (rt *jsrt) rubyOpenRangeCover(o *jsObject, v interface{}) bool {
	if lo := o.props["begin"]; !isUndefOrNull(lo) {
		if c, ok := rt.rubySpaceship(v, lo); !ok || c < 0 {
			return false
		}
	}
	if hi := o.props["end"]; !isUndefOrNull(hi) {
		c, ok := rt.rubySpaceship(v, hi)
		if !ok {
			return false
		}
		if excl, _ := o.props["excl"].(bool); excl {
			return c < 0
		}
		return c <= 0
	}
	return true
}

// rubyExcObj answers whether v is an exception instance: a user object carrying
// the `args' slot every raise path fills in (rubyExc / the default initialize).
// rubyExcInst is "v is an EXCEPTION instance", which rubyExcObj alone is not:
// makeClassStmt of ruby-to-llvm-ir.abnf gives every body-less class a default
// initialize that stores its arguments under `args', so an ordinary instance has
// the same shape. The class-chain test keeps a plain object rendering as
// "#<Name>" rather than as its class NAME (Exception#to_s).
func rubyExcInst(v interface{}) bool {
	return rubyExcObj(v) && rubyExcIsA(v, "Exception")
}

func rubyExcObj(v interface{}) bool {
	o, ok := v.(*jsObject)
	if !ok {
		return false
	}
	if _, isCls := o.props["__isclass"]; isCls {
		return false
	}
	_, hasArgs := o.props["__args"]
	_, hasClass := o.props["__class"]
	return hasArgs && hasClass
}

// rubyExcParent is the ancestry of the builtin exception classes: the part of
// Ruby's class tree a `rescue' clause has to know about. Exception is the root.
var rubyExcParent = map[string]string{
	"StandardError":     "Exception",
	"RuntimeError":      "StandardError",
	"ArgumentError":     "StandardError",
	"TypeError":         "StandardError",
	"NameError":         "StandardError",
	"NoMethodError":     "NameError",
	"IndexError":        "StandardError",
	"KeyError":          "IndexError",
	"RangeError":        "StandardError",
	"ZeroDivisionError": "StandardError",
	"StopIteration":     "IndexError",
	"FrozenError":       "RuntimeError",
	"IOError":           "StandardError",
	"EOFError":          "IOError",
	"LocalJumpError":    "StandardError",
	// ScriptError sits beside StandardError, so a bare `rescue' does not catch
	// NotImplementedError.
	"ScriptError":         "Exception",
	"NotImplementedError": "ScriptError",
}

// rubyExcIsA answers `v is an exception of builtin class named bn': it walks the
// value's own __class/__super chain (so a user subclass counts) and, once that
// reaches a builtin class, continues up rubyExcParent by name.
func rubyExcIsA(v interface{}, bn string) bool {
	if _, known := rubyExcParent[bn]; !known && bn != "Exception" {
		return false
	}
	o, ok := v.(*jsObject)
	if !ok {
		return false
	}
	for c := o.props["__class"]; c != nil; {
		co, isObj := c.(*jsObject)
		if !isObj {
			return false
		}
		name, _ := co.props["__name"].(string)
		if _, isBuiltin := co.props["__rbuiltin"].(string); isBuiltin {
			for n := name; n != ""; n = rubyExcParent[n] {
				if n == bn {
					return true
				}
			}
			return false
		}
		if name == bn {
			return true
		}
		c = co.props["__super"]
	}
	return false
}

// rubyIsA is `v.is_a?(cls)` / `cls === v`: a builtin class matches by the value's
// runtime kind (with Integer/Float also counting as Numeric and Comparable), a
// compiled class by walking the instance's __class / __super chain.
// rubyMixIsA is the mixin half of is_a?. ruby-to-llvm-ir.abnf FLATTENS `include M'
// at build time - M's method bodies are re-emitted into the including class and the
// module is never named again - so the emitter records the ancestry:
// `$mix$<Name>' -> the module object on every class in the chain (include and
// prepend), `$xmix$<Name>' on the class OBJECT for `extend M'. Identity, not the
// name, decides. Twin: rbMixIsA in languages/lib/ruby-rt.metajs; emitMixRecord in
// the emitter.
func rubyMixIsA(v interface{}, clsObj *jsObject) bool {
	o, isObj := v.(*jsObject)
	if !isObj {
		return false
	}
	nm, hasName := clsObj.props["__name"].(string)
	if !hasName {
		return false
	}
	// An INSTANCE that was `extend'ed is a member of that module too, and of every
	// module THAT one includes - `o.extend(B)' where `module B; include A' makes o
	// an A as well ($mix$<Name> is the module object's own ancestry record). The
	// per-object twin of the class case below.
	if xs, isArr := o.props["__sext"].(*jsArray); isArr {
		for _, e := range xs.elems {
			if e == interface{}(clsObj) {
				return true
			}
			if mo, isObj := e.(*jsObject); isObj && mo.props["$mix$"+nm] == interface{}(clsObj) {
				return true
			}
		}
	}
	if isCls, _ := o.props["__isclass"].(bool); isCls {
		return o.props["$xmix$"+nm] == interface{}(clsObj)
	}
	key := "$mix$" + nm
	c := o.props["__class"]
	for guard := 0; guard < 64; guard++ {
		co, isO := c.(*jsObject)
		if !isO {
			return false
		}
		if co.props[key] == interface{}(clsObj) {
			return true
		}
		c = co.props["__super"]
	}
	return false
}

func (rt *jsrt) rubyIsA(v interface{}, cls interface{}) bool {
	clsObj, ok := cls.(*jsObject)
	if !ok {
		return false
	}
	if rubyMixIsA(v, clsObj) {
		return true
	}
	if bn, isBuiltin := clsObj.props["__rbuiltin"].(string); isBuiltin {
		kind := rt.rubyClassNameRT(v)
		if kind == bn {
			return true
		}
		// `rescue StandardError' has to catch a RuntimeError, and a user class
		// `class MyErr < StandardError' has to be caught by both - so an exception
		// name walks the value's class chain and then the builtin parent table.
		if rubyExcIsA(v, bn) {
			return true
		}
		switch bn {
		case "Numeric":
			return rubyIsNum(v)
		case "Comparable":
			return rubyIsNum(v) || kind == "String"
		case "Class", "Module":
			// `A.instance_of?(Class)' / `M.is_a?(Module)'. A Class IS a Module in
			// MRI, a Module is not a Class, and neither name means anything for a
			// value that is not a class object. rubyClassName answers "" for one, so
			// the kind == bn test above never fires here.
			o, isObj := v.(*jsObject)
			if !isObj {
				return false
			}
			if isCls, _ := o.props["__isclass"].(bool); !isCls {
				return false
			}
			if bn == "Module" {
				return true
			}
			isMod, _ := o.props["__ismodule"].(bool)
			return !isMod
		case "Object", "BasicObject", "Kernel":
			return true
		case "Enumerable":
			return kind == "Array" || kind == "Hash" || kind == "Range"
		}
		return false
	}
	for c := interface{}(nil); ; {
		o, isObj := v.(*jsObject)
		if !isObj {
			return false
		}
		c = o.props["__class"]
		for c != nil {
			if c == cls {
				return true
			}
			co, isO := c.(*jsObject)
			if !isO {
				return false
			}
			c = co.props["__super"]
		}
		return false
	}
}

// ----- Module#ancestors and the module relational operators -----
// They all read the same emitted record rubyMixIsA reads. Twins: rbAncestors /
// rbModRel in languages/lib/ruby-rt.metajs, ancestorsOf / modRel in
// ruby-interpreter.abnf.
func rubyIsClassObj(v interface{}) (*jsObject, bool) {
	o, isObj := v.(*jsObject)
	if !isObj {
		return nil, false
	}
	isCls, _ := o.props["__isclass"].(bool)
	return o, isCls
}

func rubyRootAncestor(n string) bool {
	return n == "Object" || n == "Kernel" || n == "BasicObject"
}

func rubyPushAll(out *jsArray, v interface{}) {
	if a, isArr := v.(*jsArray); isArr {
		out.elems = append(out.elems, a.elems...)
	}
}

// The ancestor list in MRI's order: at each level the PREPENDED modules, then the
// class itself, then the INCLUDED modules, then the superclass. A synthetic level
// (the `X$core' a prepend introduces, the `X$builtin' shim an exception subclass
// gets) is skipped - it is an artifact of the lowering and MRI has no name for it.
// A builtin class ends the __super chain, so the walk continues by NAME through
// rubyExcParent, which is where the builtin hierarchy lives.
func (rt *jsrt) rubyAncestors(cls interface{}) *jsArray {
	out := &jsArray{}
	clsObj, isCls := rubyIsClassObj(cls)
	if !isCls {
		return out
	}
	c := interface{}(clsObj)
	for guard := 0; guard < 64; guard++ {
		co, isO := c.(*jsObject)
		if !isO {
			break
		}
		if synth, _ := co.props["__synth"].(bool); synth {
			c = co.props["__super"]
			continue
		}
		if _, isBuiltin := co.props["__rbuiltin"].(string); isBuiltin {
			// Object / Kernel / BasicObject close EVERY class's list below, so the
			// name chase stops before them rather than duplicating them.
			for bn, _ := co.props["__name"].(string); bn != "" && !rubyRootAncestor(bn); bn = rubyExcParent[bn] {
				out.elems = append(out.elems, rt.rubyBuiltinClass(bn))
			}
			break
		}
		rubyPushAll(out, co.props["__preorder"])
		out.elems = append(out.elems, co)
		rubyPushAll(out, co.props["__mixorder"])
		c = co.props["__super"]
	}
	if isMod, _ := clsObj.props["__ismodule"].(bool); !isMod {
		out.elems = append(out.elems, rt.rubyBuiltinClass("Object"),
			rt.rubyBuiltinClass("Kernel"), rt.rubyBuiltinClass("BasicObject"))
	}
	return out
}

func (rt *jsrt) rubyAncestorHas(cls interface{}, m interface{}) bool {
	if m == cls {
		return false
	}
	for _, a := range rt.rubyAncestors(cls).elems {
		if a == m {
			return true
		}
	}
	return false
}

// Module#< / #<= / #> / #>=: the ancestry relation, and NIL for two unrelated
// modules - which is why these do not go through rubyCmp, whose contract is to raise.
// rubyRel is what js_rlt / js_rgt / js_rle / js_rge do: the module relation when
// both sides are class objects, the ordinary comparison otherwise.
func (rt *jsrt) rubyRel(op string, l, r interface{}) uint64 {
	_, lIsCls := rubyIsClassObj(l)
	_, rIsCls := rubyIsClassObj(r)
	if lIsCls && rIsCls {
		return rt.wrap(rt.rubyModRel(l, r, op))
	}
	c := rt.rubyCmp(l, r)
	switch op {
	case "<":
		return rt.wrap(c < 0)
	case ">":
		return rt.wrap(c > 0)
	case "<=":
		return rt.wrap(c <= 0)
	}
	return rt.wrap(c >= 0)
}

func (rt *jsrt) rubyModRel(l, r interface{}, op string) interface{} {
	same := l == r
	lIsR := rt.rubyAncestorHas(l, r)
	rIsL := rt.rubyAncestorHas(r, l)
	if !same && !lIsR && !rIsL {
		return jsNull
	}
	switch op {
	case "<":
		return lIsR
	case "<=":
		return same || lIsR
	case ">":
		return rIsL
	}
	return same || rIsL
}

// rubyExc builds an exception instance: {__class: <class>, args: [message]}, the
// shape the message reader and the rescue type test below both understand.
// rubyRaiseErr raises a builtin Ruby exception FROM THE RUNTIME, so `1/0 rescue
// "d"' and `begin; 1/0; rescue ZeroDivisionError; end' work. A plain rt.fail is an
// ENGINE abort that no rescue clause can see.
func (rt *jsrt) rubyRaiseErr(cls, msg string) {
	panic(&jsThrown{value: rt.rubyExc(rt.rubyBuiltinClass(cls), []interface{}{msg})})
}

// rubyNoMethod raises NoMethodError for a name no method table answers, worded as
// MRI 2.6 does: the receiver's #inspect (its "#<Class>" form past 65 characters),
// then ":" and its class name unless that text already starts with "#". Twins:
// noMethod in ruby-interpreter.abnf, rbNoMethod in languages/lib/ruby-rt.metajs.
func (rt *jsrt) rubyNoMethod(t interface{}, name string) {
	if a, isArr := t.(*jsArray); isArr && rt.rubyNmArr != nil && a == rt.rubyNmArr {
		t = rt.rubyNmRecv
	}
	rt.rubyNameErr("NoMethodError", "undefined method `"+name+"' for "+rt.rubyNoMethodDesc(t), name, t)
}

// rubyNameErr raises a NameError / NoMethodError carrying MRI's #name and
// #receiver in the hidden __nmname / __nmrecv slots. Twins: nameErr in
// ruby-interpreter.abnf, rbNameErr in lib/ruby-rt.metajs.
func (rt *jsrt) rubyNameErr(cls, msg, name string, recv interface{}) {
	e := rt.rubyExc(rt.rubyBuiltinClass(cls), []interface{}{msg})
	e.set("__nmname", jsSym{s: name})
	e.set("__nmrecv", recv)
	panic(&jsThrown{value: e})
}

// rubyNameErrSlot answers NameError#name / #receiver (name) from slot, for a NameError
// instance whose class defines no method of that name; nil when the slot is unset.
func (rt *jsrt) rubyNameErrSlot(target interface{}, name, slot string) (interface{}, bool) {
	o, isObj := target.(*jsObject)
	if !isObj || !rubyExcInst(o) || rt.rubyIsA(o, rt.rubyBuiltinClass("NameError")) != true {
		return nil, false
	}
	if _, hasOwn := rubyFindMethod(target, name); hasOwn {
		return nil, false
	}
	if v, ok := o.props[slot]; ok {
		return v, true
	}
	return jsNull, true
}

// rubyDelegArr runs an Array method for a Range, Hash or Enumerator answering
// through the Array of its elements, naming the delegator for a NoMethodError as
// MRI does (`1..3:Range'). Twins: delegArr in ruby-interpreter.abnf, rbDelegArr
// in lib/ruby-rt.metajs.
func (rt *jsrt) rubyDelegArr(arr *jsArray, recv interface{}, name string, args []interface{}) interface{} {
	sa, sr := rt.rubyNmArr, rt.rubyNmRecv
	rt.rubyNmArr, rt.rubyNmRecv = arr, recv
	defer func() { rt.rubyNmArr, rt.rubyNmRecv = sa, sr }()
	return rt.rubyArrayMethod(arr, name, args)
}

// rubyNoMethodDesc is the receiver as a NoMethodError / NameError message names it.
// Twins: noMethodDesc in ruby-interpreter.abnf, rbNoMethodDesc in
// languages/lib/ruby-rt.metajs.
func (rt *jsrt) rubyNoMethodDesc(t interface{}) string {
	cn := rt.rubyClassNameRT(t)
	if o, isObj := t.(*jsObject); isObj && cn == "" {
		cn = "Object"
		if _, isCls := o.props["__isclass"]; isCls {
			cn = "Class"
			if isMod, _ := o.props["__ismodule"].(bool); isMod {
				cn = "Module"
			}
		} else if c, hasCls := o.props["__class"].(*jsObject); hasCls {
			cn, _ = c.props["__name"].(string)
		}
	}
	d := rt.rubyInspect(t)
	if rubyCharCount(d) > 65 {
		d = "#<" + cn + ">"
	}
	if !strings.HasPrefix(d, "#") {
		d += ":" + cn
	}
	return d
}

// rubyMissMark is the sentinel rubyMemberCall registers as rt.missCall, so the
// shared memberCall hands a miss back instead of aborting.
var rubyMissMark = &jsObject{}

// rubyMemberCall is the shared memberCall with a miss turned into NoMethodError.
func (rt *jsrt) rubyMemberCall(target interface{}, name string, args []interface{}) interface{} {
	saved := rt.missCall
	rt.missCall = rubyMissMark
	v := rt.memberCall(target, name, args)
	rt.missCall = saved
	if v == interface{}(rubyMissMark) {
		rt.rubyNoMethod(target, name)
	}
	return v
}

// ---------------------------------------------------------------------------
// Kernel#Integer / #Float / #String / #Array. Integer and Float are STRICT -
// MRI raises ArgumentError on anything the whole string is not - which is what
// makes `Integer(s) rescue default' the idiom it is. Mirrored line for line by
// rbKernelConv of lib/ruby-rt.metajs and kernelConv of ruby-interpreter.abnf.

// rubyConvDigit is the value of one base-36 digit, -1 when it is not one. (Not
// rubyDigitVal: that one answers 0 for a non-digit and is uppercase-blind.)
func rubyConvDigit(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'z':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'Z':
		return int(c-'A') + 10
	}
	return -1
}

func rubyIsSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' || c == '\v'
}

func rubyTrimSpace(s string) string {
	i, j := 0, len(s)
	for i < j && rubyIsSpace(s[i]) {
		i++
	}
	for j > i && rubyIsSpace(s[j-1]) {
		j--
	}
	return s[i:j]
}

// rubyStrToInum is MRI's rb_int_parse_cstr, the one parser behind String#to_i,
// #hex, #oct and Kernel#Integer, answering the EXACT value. Leading whitespace,
// one sign, then a radix prefix: base <= 0 detects 0x/0b/0o/0d or a bare
// leading 0 (octal; otherwise 10, or -base when base < -1, which is how #oct
// asks for 8), and an explicit base skips only its OWN prefix. Leading zeros
// may carry single underscores; between digits one underscore is allowed. With
// badcheck (Kernel#Integer) a doubled or trailing underscore or anything but
// trailing whitespace is ok=false; without it parsing stops there, and a string
// with no digits is 0. The caller has rejected an invalid radix. Twins: strToInum
// of ruby-interpreter.abnf and rbStrToInum of lib/ruby-rt.metajs.
func rubyStrToInum(s string, base int, badcheck bool) (*big.Int, bool) {
	i, n := 0, len(s)
	at := func(k int) byte {
		if k < n {
			return s[k]
		}
		return 0
	}
	for i < n && rubyIsSpace(s[i]) {
		i++
	}
	neg := false
	if at(i) == '+' || at(i) == '-' {
		neg = s[i] == '-'
		i++
	}
	p := at(i+1) | 0x20
	if base <= 0 {
		if at(i) == '0' && i+1 < n {
			switch p {
			case 'x':
				base, i = 16, i+2
			case 'b':
				base, i = 2, i+2
			case 'o':
				base, i = 8, i+2
			case 'd':
				base, i = 10, i+2
			default:
				base = 8
			}
		} else if base < -1 {
			base = -base
		} else {
			base = 10
		}
	} else if at(i) == '0' && ((base == 2 && p == 'b') || (base == 8 && p == 'o') || (base == 10 && p == 'd') || (base == 16 && p == 'x')) {
		i += 2
	}
	if i >= n {
		return big.NewInt(0), false
	}
	// Squeeze the leading zeros, stepping back onto the last one when nothing
	// but the end or whitespace follows them.
	if s[i] == '0' && i+1 < n {
		for us := 0; ; {
			i++
			c := at(i)
			if c == '_' {
				if us++; us >= 2 {
					break
				}
			} else if c == '0' {
				us = 0
			} else {
				if c == 0 || rubyIsSpace(c) {
					i--
				}
				break
			}
		}
	}
	if d := rubyConvDigit(at(i)); d < 0 || d >= base {
		return big.NewInt(0), false
	}
	digits := make([]byte, 0, n-i)
	nondigit := false
	for ; i < n; i++ {
		if s[i] == '_' {
			if nondigit {
				if badcheck {
					return big.NewInt(0), false
				}
				break
			}
			nondigit = true
			continue
		}
		if d := rubyConvDigit(s[i]); d < 0 || d >= base {
			break
		}
		digits = append(digits, s[i])
		nondigit = false
	}
	if badcheck {
		for i < n && rubyIsSpace(s[i]) {
			i++
		}
		if nondigit || i < n {
			return big.NewInt(0), false
		}
	}
	v, _ := new(big.Int).SetString(string(digits), base)
	if neg {
		v.Neg(v)
	}
	return v, true
}

// rubyRadixArg is the base argument of String#to_i and Kernel#Integer: MRI
// raises "invalid radix" outside 2..36, where 0 means "detect the prefix" and
// Kernel#Integer also takes a negative one (see rubyStrToInum).
func (rt *jsrt) rubyRadixArg(v interface{}, negOK bool) int {
	base := int(rt.toNumber(v))
	if base == 1 || base > 36 || base < -36 || (base < 0 && !negOK) {
		rt.rubyRaiseErr("ArgumentError", "invalid radix "+strconv.Itoa(base))
	}
	return base
}

// rubyStrToFloat is MRI's Float(String): the whole string must be a decimal
// literal, with underscores allowed between digits.
func rubyStrToFloat(s string) (float64, bool) {
	s = rubyTrimSpace(s)
	clean := ""
	for i := 0; i < len(s); i++ {
		if s[i] == '_' {
			if i == 0 || i+1 >= len(s) || rubyConvDigit(s[i-1]) < 0 || rubyConvDigit(s[i+1]) < 0 {
				return 0, false
			}
			continue
		}
		clean += string(s[i])
	}
	i := 0
	if i < len(clean) && (clean[i] == '+' || clean[i] == '-') {
		i++
	}
	intDigits := 0
	for i < len(clean) && clean[i] >= '0' && clean[i] <= '9' {
		i++
		intDigits++
	}
	if intDigits == 0 {
		return 0, false
	}
	if i < len(clean) && clean[i] == '.' {
		i++
		fd := 0
		for i < len(clean) && clean[i] >= '0' && clean[i] <= '9' {
			i++
			fd++
		}
		if fd == 0 {
			return 0, false
		}
	}
	if i < len(clean) && (clean[i] == 'e' || clean[i] == 'E') {
		i++
		if i < len(clean) && (clean[i] == '+' || clean[i] == '-') {
			i++
		}
		ed := 0
		for i < len(clean) && clean[i] >= '0' && clean[i] <= '9' {
			i++
			ed++
		}
		if ed == 0 {
			return 0, false
		}
	}
	if i != len(clean) {
		return 0, false
	}
	f, err := strconv.ParseFloat(clean, 64)
	if err != nil {
		return 0, false
	}
	return f, true
}

func (rt *jsrt) rubyKernelConv(name string, args []interface{}) interface{} {
	var v interface{} = jsNull
	if len(args) > 0 {
		v = args[0]
	}
	switch name {
	case "String":
		return rt.rubyStr(v)
	// Kernel#Rational and Kernel#Complex, the ONLY spelling that builds either from
	// two parts. Rational(a) IS a.to_r and Rational(a, b) is a.to_r / b.to_r, so
	// both go through rubyAsRat and inherit exactly what #to_r answers.
	case "Rational":
		p := rt.rubyAsRat(v)
		if len(args) < 2 {
			return rubyMkRat(p.n, p.d)
		}
		q := rt.rubyAsRat(args[1])
		if q.n == 0 {
			rt.rubyRaiseErr("ZeroDivisionError", "divided by 0")
		}
		return rt.rubyRatArith("/", p, q)
	case "Complex":
		if len(args) < 2 {
			if c, isCpx := v.(jsCpx); isCpx {
				return c
			}
		}
		var im interface{} = 0.0
		if len(args) > 1 {
			im = args[1]
		}
		return jsCpx{re: v, im: im}
	case "Array":
		if isUndefOrNull(v) {
			return &jsArray{}
		}
		if arr, isArr := v.(*jsArray); isArr {
			return arr
		}
		if o, isObj := v.(*jsObject); isObj {
			if keys, vals, isDict := dictParts(o); isDict {
				out := &jsArray{}
				for i := range keys.elems {
					out.elems = append(out.elems, &jsArray{elems: []interface{}{keys.elems[i], vals.elems[i]}})
				}
				return out
			}
		}
		return &jsArray{elems: []interface{}{v}}
	case "Integer":
		base := 0
		if len(args) > 1 {
			if _, isStr := v.(string); !isStr {
				rt.rubyRaiseErr("ArgumentError", "base specified for non string value")
			}
			base = rt.rubyRadixArg(args[1], true)
		}
		if isUndefOrNull(v) {
			rt.rubyRaiseErr("TypeError", "can't convert nil into Integer")
		}
		if s, isStr := v.(string); isStr {
			if n, ok := rubyStrToInum(s, base, true); ok {
				return rt.rubyBigNarrow(n)
			}
			rt.rubyRaiseErr("ArgumentError", "invalid value for Integer(): "+rubyStrInspect(s))
		}
		// A Float converts as Float#to_i does.
		if f, isFlo := v.(jsFlo); isFlo {
			rubyFloDomain(rt, f.f)
			return rt.rubyIntFromF(math.Trunc(f.f))
		}
		return v
	case "Float":
		if isUndefOrNull(v) {
			rt.rubyRaiseErr("TypeError", "can't convert nil into Float")
		}
		if s, isStr := v.(string); isStr {
			if f, ok := rubyStrToFloat(s); ok {
				return jsFlo{f: f}
			}
			rt.rubyRaiseErr("ArgumentError", "invalid value for Float(): "+rubyStrInspect(s))
		}
		return jsFlo{f: rubyToF(v)}
	}
	return v
}

// rubyIndexSet is js_rset's body, factored out so rubySetSlice can reuse the
// one-element rule (a user class dispatches its own []=, a missing Hash key is
// APPENDED and an Array GROWS, filling the gap with nil).
func (rt *jsrt) rubyIndexSet(t, k, v interface{}) {
	if rubyUserObj(t) {
		rt.rubyMethod(t, "[]=", []interface{}{k, v})
		return
	}
	if keys, vals, ok := dictParts(t); ok {
		if i := rt.rubyDictFind(keys, k); i >= 0 {
			vals.elems[i] = v
		} else {
			dictAppend(keys, vals, k, v)
		}
		return
	}
	arr, ok := t.(*jsArray)
	if !ok {
		rt.fail("item assignment on a %s", rt.typeOf(t))
	}
	idx := int(rt.toNumber(k))
	if idx < 0 {
		idx += len(arr.elems)
	}
	if idx < 0 {
		rt.fail("index %d too small for array", idx)
	}
	arr.dropIdx()
	for len(arr.elems) <= idx {
		arr.elems = append(arr.elems, jsNull)
	}
	arr.elems[idx] = v
}

// rubySliceBounds resolves a Ruby subscript pair to (start, length) against a
// receiver of size n. k2 absent means a ONE-element write, unless k1 is a Range,
// which is read from its OWN endpoints.
func rubySliceBounds(n int, k1, k2 interface{}) (int, int) {
	if !isUndefOrNull(k2) {
		s := int(math.Trunc(rubyToF(k1)))
		if s < 0 {
			s += n
		}
		l := int(math.Trunc(rubyToF(k2)))
		if l < 0 {
			l = 0
		}
		return s, l
	}
	if r, isRng := rubyIsRange(k1); isRng {
		s := 0
		if lo := r.props["begin"]; !isUndefOrNull(lo) {
			s = int(math.Trunc(rubyToF(lo)))
			if s < 0 {
				s += n
			}
		}
		e := n - 1
		if hi := r.props["end"]; !isUndefOrNull(hi) {
			e = int(math.Trunc(rubyToF(hi)))
			if e < 0 {
				e += n
			}
			if excl, _ := r.props["excl"].(bool); excl {
				e--
			}
		}
		l := e - s + 1
		if l < 0 {
			l = 0
		}
		return s, l
	}
	if arr, isArr := k1.(*jsArray); isArr {
		if len(arr.elems) == 0 {
			return n, 0
		}
		s := int(math.Trunc(rubyToF(arr.elems[0])))
		if s < 0 {
			s += n
		}
		return s, len(arr.elems)
	}
	s := int(math.Trunc(rubyToF(k1)))
	if s < 0 {
		s += n
	}
	return s, 1
}

// rubySetSlice is js_rsetsl: see the extern's comment. It answers the container.
func (rt *jsrt) rubySetSlice(o, k1, k2, v interface{}) interface{} {
	if rubyUserObj(o) {
		args := []interface{}{k1, v}
		if !isUndefOrNull(k2) {
			args = []interface{}{k1, k2, v}
		}
		rt.rubyMethod(o, "[]=", args)
		return o
	}
	if _, _, isDict := dictParts(o); isDict {
		rt.rubyIndexSet(o, k1, v)
		return o
	}
	if str, isStr := o.(string); isStr {
		n := rt.strLen(str)
		s, l := rubySliceBounds(n, k1, k2)
		if s < 0 || s > n {
			rt.rubyRaiseErr("IndexError", "index "+jsNumString(float64(s))+" out of string")
		}
		if s+l > n {
			l = n - s
		}
		return rt.strRange(str, 0, s) + rt.rubyStr(v) + rt.strRange(str, s+l, n)
	}
	arr, isArr := o.(*jsArray)
	if !isArr {
		rt.fail("item assignment on a %s", rt.typeOf(o))
	}
	// A plain one-element write keeps js_rset's own growth rule. A RANGE key is a
	// slice write, not an element write.
	if isUndefOrNull(k2) {
		_, isArrKey := k1.(*jsArray)
		_, isRngKey := rubyIsRange(k1)
		if !isArrKey && !isRngKey {
			rt.rubyIndexSet(arr, k1, v)
			return arr
		}
	}
	s, l := rubySliceBounds(len(arr.elems), k1, k2)
	if s < 0 {
		rt.fail("index %d too small for array", s)
	}
	arr.dropIdx()
	for len(arr.elems) < s {
		arr.elems = append(arr.elems, jsNull)
	}
	if s+l > len(arr.elems) {
		l = len(arr.elems) - s
	}
	ins := []interface{}{v}
	if va, isArr2 := v.(*jsArray); isArr2 {
		ins = append([]interface{}{}, va.elems...)
	}
	tail := append([]interface{}{}, arr.elems[s+l:]...)
	arr.elems = append(append(arr.elems[:s:s], ins...), tail...)
	return arr
}

// ===== the String surface shared with lib/ruby-rt.metajs and ==================
// ruby-interpreter.abnf (smMethod / smUtf8Bytes / smSetExpand / ...).

// The receiver as a list of UTF-16-unit characters, which is what rt.strAt hands
// out and therefore what the other two engines see.
func (rt *jsrt) rubySmChars(o string) []string {
	n := rt.strLen(o)
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, rt.strAt(o, i))
	}
	return out
}

// MRI's character-set notation, shared by tr / tr_s / count / delete / squeeze:
// a leading ^ negates, `a-z` is a range, a backslash escapes the next character.
func rubySmSetExpand(spec string) (bool, []string) {
	cs := []rune(spec)
	neg := false
	i := 0
	if len(cs) > 1 && cs[0] == '^' {
		neg = true
		i = 1
	}
	out := []string{}
	for i < len(cs) {
		if cs[i] == '\\' && i+1 < len(cs) {
			out = append(out, string(cs[i+1]))
			i += 2
			continue
		}
		if i+2 < len(cs) && cs[i+1] == '-' {
			for k := cs[i]; k <= cs[i+2]; k++ {
				out = append(out, string(k))
			}
			i += 3
			continue
		}
		out = append(out, string(cs[i]))
		i++
	}
	return neg, out
}

func rubySmHas(list []string, ch string) int {
	for i, c := range list {
		if c == ch {
			return i
		}
	}
	return -1
}

// Is ch in EVERY one of the given set specs? MRI intersects them.
func rubySmInSets(specs []string, ch string) bool {
	for _, sp := range specs {
		neg, chars := rubySmSetExpand(sp)
		found := rubySmHas(chars, ch) >= 0
		if neg {
			found = !found
		}
		if !found {
			return false
		}
	}
	return len(specs) > 0
}

// String#tr / #tr_s.
func (rt *jsrt) rubySmTr(o, from, to string, sq bool) string {
	fneg, fchars := rubySmSetExpand(from)
	_, tchars := rubySmSetExpand(to)
	last := ""
	if len(tchars) > 0 {
		last = tchars[len(tchars)-1]
	}
	out := ""
	prev := ""
	for _, ch := range rt.rubySmChars(o) {
		k := rubySmHas(fchars, ch)
		rep := ch
		did := false
		if fneg {
			if k < 0 {
				rep = last
				did = true
			}
		} else if k >= 0 {
			if len(tchars) == 0 {
				rep = ""
			} else if k >= len(tchars) {
				rep = last
			} else {
				rep = tchars[k]
			}
			did = true
		}
		if rep == "" {
			prev = ""
			continue
		}
		if sq && did && rep == prev {
			continue
		}
		out += rep
		if did {
			prev = rep
		} else {
			prev = ""
		}
	}
	return out
}

func (rt *jsrt) rubySmSqueeze(o string, specs []string) string {
	out := ""
	prev := ""
	for _, ch := range rt.rubySmChars(o) {
		if ch == prev && (len(specs) == 0 || rubySmInSets(specs, ch)) {
			continue
		}
		out += ch
		prev = ch
	}
	return out
}

// String#center / #ljust / #rjust: mode 0 centres, 1 pads right, 2 pads left.
// MRI counts CHARACTERS here, unlike printf's %10s, which counts bytes.
func (rt *jsrt) rubySmPad(o string, width int, pad string, mode int) string {
	n := rt.strLen(o)
	if rt.strLen(pad) == 0 || width <= n {
		return o
	}
	total := width - n
	left := 0
	if mode == 0 {
		left = total / 2
	} else if mode == 2 {
		left = total
	}
	return rt.rubySmRepeatTo(pad, left) + o + rt.rubySmRepeatTo(pad, total-left)
}

func (rt *jsrt) rubySmRepeatTo(pad string, n int) string {
	units := rt.rubySmChars(pad)
	out := ""
	for i := 0; i < n; i++ {
		out += units[i%len(units)]
	}
	return out
}

func (rt *jsrt) rubySmSwapcase(o string) string {
	out := ""
	for _, ch := range rt.rubySmChars(o) {
		up := caseMapMode(ch, caseUpper|caseFull)
		if ch == up {
			out += caseMapMode(ch, caseFull)
		} else {
			out += up
		}
	}
	return out
}

// String#lines: split after each newline, keeping it.
func (rt *jsrt) rubySmLines(o string) []string {
	out := []string{}
	cur := ""
	for _, ch := range rt.rubySmChars(o) {
		cur += ch
		if ch == "\n" {
			out = append(out, cur)
			cur = ""
		}
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

func rubySmIsStrip(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' || c == '\v' || c == 0
}

func rubySmLstrip(o string) string {
	i := 0
	for i < len(o) && rubySmIsStrip(o[i]) {
		i++
	}
	return o[i:]
}

func rubySmRstrip(o string) string {
	j := len(o)
	for j > 0 && rubySmIsStrip(o[j-1]) {
		j--
	}
	return o[:j]
}

func rubySmNumArr(bs []byte) *jsArray {
	out := &jsArray{}
	for _, b := range bs {
		out.elems = append(out.elems, float64(b))
	}
	return out
}

func rubySmStrArr(ss []string) *jsArray {
	out := &jsArray{}
	for _, e := range ss {
		out.elems = append(out.elems, e)
	}
	return out
}

// The BANG forms of the case / trim / translate methods. A Ruby String is a VALUE
// in this model, so the in-place effect is unrepresentable (the same limitation
// sub! / gsub! already have) - but the RETURN VALUE is not: MRI answers the new
// string, or nil when nothing changed. Mirrors smBangBase of the other two
// engines.
func rubySmBangBase(name string) string {
	if len(name) < 2 || name[len(name)-1] != '!' {
		return ""
	}
	b := name[:len(name)-1]
	switch b {
	case "upcase", "downcase", "capitalize", "swapcase", "reverse", "strip",
		"lstrip", "rstrip", "chomp", "chop", "squeeze", "tr", "tr_s", "delete",
		"succ", "next":
		return b
	}
	return ""
}

// String#upto: every string from the receiver to limit by #succ.
func rubySmUptoList(s, limit string, excl bool) []string {
	out := []string{}
	cur := s
	for guard := 0; guard < 100000; guard++ {
		if len(cur) > len(limit) {
			return out
		}
		if len(cur) == len(limit) && cur > limit {
			return out
		}
		if excl && cur == limit {
			return out
		}
		out = append(out, cur)
		if cur == limit {
			return out
		}
		cur = rubyStrSucc(cur)
	}
	return out
}

// rubySmMethod is smMethod of lib/ruby-rt.metajs and ruby-interpreter.abnf. It
// answers ok=false for a name it does not know, so the caller's own
// "unknown String method" abort still happens.
func (rt *jsrt) rubySmMethod(o, name string, sargs []string, nargs []float64) (interface{}, bool) {
	num := func(i int) int {
		if i < len(nargs) {
			return int(math.Trunc(nargs[i]))
		}
		return 0
	}
	sarg := func(i int) string {
		if i < len(sargs) {
			return sargs[i]
		}
		return ""
	}
	switch name {
	case "bytesize":
		return float64(len(o)), true
	case "bytes":
		return rubySmNumArr([]byte(o)), true
	case "getbyte":
		bs := []byte(o)
		gi := num(0)
		if gi < 0 {
			gi += len(bs)
		}
		if gi < 0 || gi >= len(bs) {
			return jsNull, true
		}
		return float64(bs[gi]), true
	case "sum":
		acc := 0.0
		for _, b := range []byte(o) {
			acc += float64(b)
		}
		return acc, true
	case "codepoints":
		out := &jsArray{}
		for _, r := range o {
			out.elems = append(out.elems, float64(r))
		}
		return out, true
	case "tr", "tr!":
		return rt.rubySmTr(o, sarg(0), sarg(1), false), true
	case "tr_s", "tr_s!":
		return rt.rubySmTr(o, sarg(0), sarg(1), true), true
	case "squeeze", "squeeze!":
		return rt.rubySmSqueeze(o, sargs), true
	case "count":
		n := 0.0
		for _, ch := range rt.rubySmChars(o) {
			if rubySmInSets(sargs, ch) {
				n++
			}
		}
		return n, true
	case "delete", "delete!":
		out := ""
		for _, ch := range rt.rubySmChars(o) {
			if !rubySmInSets(sargs, ch) {
				out += ch
			}
		}
		return out, true
	case "center", "ljust", "rjust":
		pad := " "
		if len(sargs) > 1 {
			pad = sargs[1]
		}
		mode := 0
		if name == "ljust" {
			mode = 1
		} else if name == "rjust" {
			mode = 2
		}
		return rt.rubySmPad(o, num(0), pad, mode), true
	case "chop", "chop!":
		if strings.HasSuffix(o, "\r\n") {
			return o[:len(o)-2], true
		}
		if n := rt.strLen(o); n > 0 {
			return rt.strRange(o, 0, n-1), true
		}
		return o, true
	case "chomp", "chomp!":
		if len(sargs) > 0 {
			if sargs[0] != "" && strings.HasSuffix(o, sargs[0]) {
				return o[:len(o)-len(sargs[0])], true
			}
			return o, true
		}
		if strings.HasSuffix(o, "\r\n") {
			return o[:len(o)-2], true
		}
		if strings.HasSuffix(o, "\n") || strings.HasSuffix(o, "\r") {
			return o[:len(o)-1], true
		}
		return o, true
	case "chr":
		if rt.strLen(o) == 0 {
			return "", true
		}
		return rt.strAt(o, 0), true
	case "lstrip", "lstrip!":
		return rubySmLstrip(o), true
	case "rstrip", "rstrip!":
		return rubySmRstrip(o), true
	case "strip!":
		return rubySmLstrip(rubySmRstrip(o)), true
	case "swapcase", "swapcase!":
		return rt.rubySmSwapcase(o), true
	case "lines":
		return rubySmStrArr(rt.rubySmLines(o)), true
	case "partition", "rpartition":
		sep := sarg(0)
		at := -1
		if name == "partition" {
			at = rt.strIndexOf(o, sep)
		} else {
			at = rubySmRindexUnits(rt, o, sep)
		}
		if at < 0 {
			if name == "partition" {
				return rubySmStrArr([]string{o, "", ""}), true
			}
			return rubySmStrArr([]string{"", "", o}), true
		}
		n := rt.strLen(o)
		return rubySmStrArr([]string{rt.strRange(o, 0, at), sep,
			rt.strRange(o, at+rt.strLen(sep), n)}), true
	case "delete_prefix", "delete_prefix!":
		if sarg(0) != "" && strings.HasPrefix(o, sarg(0)) {
			return o[len(sarg(0)):], true
		}
		return o, true
	case "delete_suffix", "delete_suffix!":
		if sarg(0) != "" && strings.HasSuffix(o, sarg(0)) {
			return o[:len(o)-len(sarg(0))], true
		}
		return o, true
	case "casecmp":
		la, lb := strings.ToLower(o), strings.ToLower(sarg(0))
		if la < lb {
			return float64(-1), true
		}
		if la > lb {
			return float64(1), true
		}
		return float64(0), true
	case "casecmp?":
		return caseMapMode(o, caseFold|caseFull) == caseMapMode(sarg(0), caseFold|caseFull), true
	case "rindex":
		if i := rubySmRindexUnits(rt, o, sarg(0)); i >= 0 {
			return float64(i), true
		}
		return jsNull, true
	case "insert":
		n := rt.strLen(o)
		at := num(0)
		if at < 0 {
			at += n + 1
		}
		if at < 0 || at > n {
			return nil, false
		}
		return rt.strRange(o, 0, at) + sarg(1) + rt.strRange(o, at, n), true
	case "prepend":
		return sarg(0) + o, true
	case "concat", "<<":
		acc := o
		for _, a := range sargs {
			acc += a
		}
		return acc, true
	// String#hex / #oct never raise: they are #to_i(16) and a prefix-detecting
	// #to_i whose default radix is 8.
	case "hex":
		n, _ := rubyStrToInum(o, 16, false)
		return rt.rubyBigNarrow(n), true
	case "oct":
		n, _ := rubyStrToInum(o, -8, false)
		return rt.rubyBigNarrow(n), true
	case "clear":
		return "", true
	case "b", "scrub", "unicode_normalize", "force_encoding", "encode":
		return o, true
	case "unicode_normalized?", "valid_encoding?":
		return true, true
	case "ascii_only?":
		for i := 0; i < len(o); i++ {
			if o[i] > 127 {
				return false, true
			}
		}
		return true, true
	}
	return nil, false
}

// The LAST index of sub in UTF-16 units, or -1.
func rubySmRindexUnits(rt *jsrt, o, sub string) int {
	n := rt.strLen(o)
	m := rt.strLen(sub)
	if m == 0 {
		return n
	}
	for i := n - m; i >= 0; i-- {
		if rt.strRange(o, i, i+m) == sub {
			return i
		}
	}
	return -1
}

// rubyExc is an exception instance: its constructor arguments sit in the hidden
// __args slot, which no @ivar reflection sees. Twin of rbExc in lib/ruby-rt.metajs.
func (rt *jsrt) rubyExc(cls *jsObject, args []interface{}) *jsObject {
	inst := newJSObject()
	inst.set("__class", cls)
	inst.set("__args", &jsArray{elems: append([]interface{}{}, args...)})
	return inst
}

// rubyMakeExc turns the operand of Ruby's `raise' into the exception object to
// throw, so the compiler grammar (ruby-to-llvm-ir.abnf) needs no runtime type
// dispatch of its own in IR:
//   - a class object    -> a fresh instance, the message as its single argument
//     (its own initialize still runs when the class defines one, so a subclass
//     that supplies a default message through `super' gets it)
//   - a string          -> a RuntimeError carrying that message, as in MRI
//   - nil / undefined   -> a BARE `raise': re-raise what the running rescue caught
//   - anything else     -> raised unchanged (`raise ArgumentError.new("arg")')
//
// The instance always carries `args', which is where rubyExcMessage reads the
// message from, so a user exception class needs no code of its own.
func (rt *jsrt) rubyMakeExc(v interface{}, msg interface{}) interface{} {
	if isUndefOrNull(v) {
		if rt.rubyCurExc != nil {
			return rt.rubyCurExc
		}
		return rt.rubyExc(rt.rubyBuiltinClass("RuntimeError"), nil)
	}
	if s, isStr := v.(string); isStr {
		return rt.rubyExc(rt.rubyBuiltinClass("RuntimeError"), []interface{}{s})
	}
	if o, isObj := v.(*jsObject); isObj {
		if _, isCls := o.props["__isclass"]; isCls {
			var argv []interface{}
			if !isUndefOrNull(msg) {
				argv = []interface{}{msg}
			}
			inst := rt.rubyExc(o, argv)
			// A user class may define initialize (possibly only through super);
			// a builtin one, and a class chain without any, needs nothing more.
			if _, has := rubyFindMethod(inst, "initialize"); has {
				rt.rubyMethod(inst, "initialize", argv)
			}
			return inst
		}
	}
	return v
}

// rubySubrange is Ruby's subscript rule for a range whose RAW ENDPOINTS are still
// known: a nil endpoint is the beginning / the end (`s[1..]`), a negative one counts
// back from the end, and an exclusive end with no endpoint is still the end. The
// second result is false when the receiver is neither a String nor an Array, so the
// caller can fall back to ordinary indexing. Mirrors rsubrange of
// ruby-interpreter.abnf and js_rsubrange of lib/ruby-rt.metajs.
func (rt *jsrt) rubySubrange(o, lo, hi interface{}, excl bool) (interface{}, bool) {
	n := -1
	switch t := o.(type) {
	case *jsArray:
		n = len(t.elems)
	case string:
		n = rt.strLen(t)
	}
	if n < 0 {
		return nil, false
	}
	b0 := 0
	if !isUndefOrNull(lo) {
		b0 = int(math.Trunc(rubyToF(lo)))
	}
	if b0 < 0 {
		b0 += n
	}
	if b0 < 0 || b0 > n {
		return jsNull, true
	}
	e0 := n - 1
	if !isUndefOrNull(hi) {
		e0 = int(math.Trunc(rubyToF(hi)))
		if e0 < 0 {
			e0 += n
		}
		if excl {
			e0--
		}
	}
	length := e0 - b0 + 1
	if length < 0 {
		length = 0
	}
	if str, isStr := o.(string); isStr {
		return rt.rubyStrSubseq(str, float64(b0), float64(length)), true
	}
	return rubyArrSubseq(o.(*jsArray), float64(b0), float64(length)), true
}

// rubyIndexGet is Ruby indexing: a[i], h[k], s[i]. Identical to js_pyget except
// that a missing Hash key and an out-of-range index answer NIL instead of raising
// - which is Ruby's Hash#[] / Array#[] / String#[], and what ruby-interpreter.abnf
// does. js_rget and js_rsubrange's non-sequence fallback share it.
func (rt *jsrt) rubyIndexGet(o, k interface{}) interface{} {
	// A Proc answers p[x] by calling itself.
	if isCallable(o) {
		return rt.call(o, jsUndef, []interface{}{k})
	}
	// A user class defines its own indexing with `def [](i)`.
	if rubyUserObj(o) {
		return rt.rubyMethod(o, "[]", []interface{}{k})
	}
	switch o.(type) {
	case float64, *jsBigInt:
		return rt.rubyIntBit(o, k)
	}
	if keys, vals, ok := dictParts(o); ok {
		i := rt.rubyDictFind(keys, k)
		if i < 0 {
			return rt.rubyHashDefault(o, k)
		}
		return vals.elems[i]
	}
	// a[0..2] / s[6..] / a[1..-1]: a Range as the subscript of an Array or a String
	// is a SUBSEQUENCE, exact for every range because the range carries its own
	// raw endpoints.
	if ko, isRng := rubyIsRange(k); isRng {
		if v, ok := rt.rubySubrange(o, ko.props["begin"], ko.props["end"], rt.truthy(ko.props["excl"])); ok {
			return v
		}
	}
	idx := int(rt.toNumber(k))
	switch o := o.(type) {
	case *jsArray:
		if idx < 0 {
			idx += len(o.elems)
		}
		if idx < 0 || idx >= len(o.elems) {
			return jsNull
		}
		return o.elems[idx]
	case string:
		n := rt.strLen(o)
		if idx < 0 {
			idx += n
		}
		if idx < 0 || idx >= n {
			return jsNull
		}
		return rt.strAt(o, idx)
	}
	if isUndefOrNull(o) {
		rt.fail("indexing nil")
	}
	return rt.getMember(o, k)
}

// rubyExcTopLine is MRI's uncaught-exception line:
// "prog.rb:2:in `<main>': boom (RuntimeError)". The site is what js_rraiseat
// recorded; an exception raised by the RUNTIME rather than by a `raise' has none,
// and then only "msg (Class)" is printed. Mirrors excTopLine of
// ruby-interpreter.abnf and rbExcTopLine of lib/ruby-rt.metajs.
func (rt *jsrt) rubyExcTopLine(exc interface{}) string {
	o, isObj := exc.(*jsObject)
	if !isObj {
		return "uncaught throw " + rt.rubyInspect(exc)
	}
	msg := ""
	if m, ok := rt.rubyExcMessage(o); ok {
		msg = rt.rubyStr(m)
	}
	cname := "RuntimeError"
	if c, has := o.props["__class"]; has {
		if co, isCo := c.(*jsObject); isCo {
			if n, isStr := co.props["__name"].(string); isStr {
				cname = n
			}
		}
	}
	head := ""
	if where, isStr := o.props["__rwhere"].(string); isStr {
		head = where + ": "
	}
	return head + msg + " (" + cname + ")"
}

// rubyExcMessage is Exception#message / #to_s: the first constructor argument,
// or the class name when the exception was raised without one (MRI's default).
func (rt *jsrt) rubyExcMessage(o *jsObject) (interface{}, bool) {
	if arr, ok := o.props["__args"].(*jsArray); ok && len(arr.elems) > 0 {
		return rt.rubyStr(arr.elems[0]), true
	}
	if cls, ok := o.props["__class"].(*jsObject); ok {
		if n, isStr := cls.props["__name"].(string); isStr {
			return n, true
		}
	}
	return nil, false
}

// rubyClassOfSelf answers the class a @@class variable belongs to: the class of an
// instance, or `self` itself when the code runs in a class method / a class body.
func rubyClassOfSelf(self interface{}) interface{} {
	o, ok := self.(*jsObject)
	if !ok {
		return nil
	}
	if _, isCls := o.props["__isclass"]; isCls {
		return o
	}
	return o.props["__class"]
}

// rubyNeg is unary minus over the tower; a user object dispatches its own -@.
func (rt *jsrt) rubyNeg(v interface{}) interface{} {
	switch t := v.(type) {
	case float64:
		return -t
	case jsFlo:
		return jsFlo{f: -t.f}
	case jsRat:
		return rubyRatNeg(t)
	case jsCpx:
		return jsCpx{re: rt.rubyNeg(t.re), im: rt.rubyNeg(t.im)}
	case *jsBigInt:
		return rt.rubyBigNarrow(new(big.Int).Neg(t.v))
	}
	if rubyUserObj(v) {
		return rt.rubyMethod(v, "-@", nil)
	}
	rt.fail("cannot negate a %s", rt.typeOf(v))
	return nil
}

// rubyStr is Ruby's to_s: a Float keeps its decimal point, a Symbol is its bare
// name, and an Array / Hash renders like #inspect (which is what interpolation of a
// collection gives in Ruby). It mirrors rstr of ruby-interpreter.abnf.
// rubyPuts is Kernel#puts for ONE argument, mirroring putsVal of
// ruby-interpreter.abnf: an array is walked recursively (one element per line),
// an EMPTY array prints nothing, everything else prints its to_s.
func (rt *jsrt) rubyPuts(v interface{}) {
	if arr, ok := v.(*jsArray); ok {
		for _, e := range arr.elems {
			rt.rubyPuts(e)
		}
		return
	}
	fmt.Fprintln(outWriter, wtf8Clean(rt.rubyStr(v)))
}

func (rt *jsrt) rubyStr(v interface{}) string {
	switch t := v.(type) {
	case jsNullT, jsUndefT:
		return ""
	case float64:
		return rubyIntStr(t)
	case *jsBigInt:
		return t.v.String()
	case jsFlo:
		return rubyFloStr(t.f)
	case jsRat:
		return rubyRatText(t)
	case jsCpx:
		return rt.rubyCpxText(t, false)
	case *jsArray:
		return rt.rubyInspect(v)
	case *jsObject:
		if _, _, isDict := dictParts(t); isDict {
			return rt.rubyInspect(v)
		}
		// A CLASS object stringifies as its name (`Foo.to_s` / "#{Foo}" == "Foo"),
		// not through the generic object path, which printed "[object Object]",
		// unless a class method #to_s says otherwise.
		if _, isCls := t.props["__isclass"]; isCls {
			if _, ok := t.props["__name"].(string); ok {
				return rt.rubyClassText(t, "to_s")
			}
		}
		if m, ok := rubyFindMethod(t, "to_s"); ok {
			return rt.rubyStr(rt.call(m, jsUndef, []interface{}{t}))
		}
		// An exception instance renders as its message, like Exception#to_s in MRI
		// (so `"log " + e' and "#{e}" read the way a Ruby program expects).
		if rubyExcInst(t) {
			if msg, ok := rt.rubyExcMessage(t); ok {
				return rt.toString(msg)
			}
		}
		if _, isRng := t.props["__rrange"]; isRng {
			return rt.rubyRangeText(t, false)
		}
		// A plain instance with no #to_s of its own. MRI writes
		// `#<A:0x00007f... @x=1>'; the address is its own and cannot be matched,
		// so the address and the instance variables are left off rather than
		// invented - the same rule the Proc renderer above follows.
		if n := rubyInstClassName(t); n != "" {
			return "#<" + n + ">"
		}
		if rxIsRegexObj(t) || rxIsMatchObj(t) {
			if s, ok := rt.rxRegexMethod(t, "to_s", nil); ok {
				return rt.toString(s)
			}
		}
	}
	if isCallable(v) {
		return rt.rubyFuncRender(v)
	}
	return rt.toString(v)
}

// rubyFuncRender is how a PROC prints. MRI says `#<Proc:0x0000...@f.rb:4
// (lambda)>`; the address is its own and cannot be matched, and neither can the
// source location, so both are left off rather than invented - the same rule
// python's <function fn> follows. Mirrors rstr of
// ruby-interpreter.abnf and rbFuncRender of lib/ruby-rt.metajs.
func (rt *jsrt) rubyFuncRender(v interface{}) string {
	// A METHOD value renders as MRI's `#<Method: Object#foo>`, and that one IS
	// byte-exact against the oracle: MRI 2.6 writes exactly it and only 3.0
	// appends a source location.
	if mv, ok := rt.rubyMethodValOf(v); ok {
		return "#<Method: " + mv.owner + "#" + mv.name + ">"
	}
	if rt.rubyLambdas[v] {
		return "#<Proc (lambda)>"
	}
	return "#<Proc>"
}

// rubyMethodVal is what a Method value knows about itself: the class its receiver
// belongs to and the method name. Mirrors rbMethodRec of lib/ruby-rt.metajs and
// methodOwners / methodNames of ruby-interpreter.abnf.
type rubyMethodVal struct{ owner, name string }

// rubyMethodValOf is the only read of rt.rubyMethodVals, and the type switch in it
// is not an optimisation - see rubyClassNameRT.
func (rt *jsrt) rubyMethodValOf(v interface{}) (rubyMethodVal, bool) {
	if len(rt.rubyMethodVals) == 0 {
		return rubyMethodVal{}, false
	}
	switch v.(type) {
	case *jsClosure, *hostFunc, *boundMethod:
		mv, ok := rt.rubyMethodVals[v]
		return mv, ok
	}
	return rubyMethodVal{}, false
}

// rubyMarkMethodVal records f as a Method rather than a Proc. The owner is the
// receiver's class, which is what MRI prints for a method defined there; a method
// INHERITED from a superclass is written `D(C)#m` by MRI and plainly `D#m` here -
// the one shape this leaves divergent, and it can only shorten the answer.
func (rt *jsrt) rubyMarkMethodVal(f interface{}, owner string, name string) interface{} {
	// Only a pointer-typed callable is recorded, for rubyClassNameRT's reason: a
	// map insert panics on a non-comparable key just as a lookup does.
	switch f.(type) {
	case *jsClosure, *hostFunc, *boundMethod:
	default:
		return f
	}
	if rt.rubyMethodVals == nil {
		rt.rubyMethodVals = map[interface{}]rubyMethodVal{}
	}
	rt.rubyMethodVals[f] = rubyMethodVal{owner: owner, name: name}
	return f
}

// rubyKernelMethodNames is the Kernel functions `method(:x)` hands out as ordinary
// Methods, and rubyKernelMethodArity is MRI's Method#arity for each (-1 is "any
// number"; only String and Array take exactly one).
// THE LIST AND ITS ORDER ARE THE SAME in languages/ruby-interpreter.abnf
// (kernelMethodNames) and languages/lib/ruby-rt.metajs (rbKernelMethodNames) - a
// name in one and not the others would be a halves divergence by construction.
// What is deliberately left out, and why, is written out once at kernelMethodNames
// in the interpreter half.
var rubyKernelMethodNames = []string{"puts", "print", "p", "format", "sprintf", "printf", "rand",
	"Integer", "Float", "String", "Array", "Rational", "Complex"}

var rubyKernelMethodArity = []int{-1, -1, -1, -1, -1, -1, -1, -1, -1, 1, 1, -1, -1}

func rubyKernelMethodIndex(n string) int {
	for i, k := range rubyKernelMethodNames {
		if k == n {
			return i
		}
	}
	return -1
}

// rubyKernelInvoke calls one from an argument SLICE - through the same bodies the
// emitter reaches for the bare name, so a Method value and a plain call cannot
// answer differently.
func (rt *jsrt) rubyKernelInvoke(n string, args []interface{}) interface{} {
	switch n {
	case "puts":
		if len(args) == 0 {
			fmt.Fprintln(outWriter)
			return jsNull
		}
		for _, e := range args {
			rt.rubyPuts(e)
		}
		return jsNull
	case "print":
		for _, e := range args {
			fmt.Fprint(outWriter, wtf8Clean(rt.rubyStr(e)))
		}
		return jsNull
	case "p":
		if len(args) == 0 {
			return jsNull
		}
		for _, e := range args {
			fmt.Fprintln(outWriter, wtf8Clean(rt.rubyInspect(e)))
		}
		if len(args) == 1 {
			return args[0]
		}
		return &jsArray{elems: append([]interface{}{}, args...)}
	case "format", "sprintf":
		return rt.rubyFormatArgs(args)
	case "printf":
		fmt.Fprint(outWriter, wtf8Clean(rt.rubyFormatArgs(args)))
		return jsNull
	case "rand":
		if len(args) == 0 {
			return rubyRand(nil)
		}
		return rubyRand(args[0])
	}
	return rt.rubyKernelConv(n, args)
}

// rubyKernelMethodVal is the Method value for one of them, with MRI's arity
// recorded so Method#arity answers off the ordinary registry.
func (rt *jsrt) rubyKernelMethodVal(name string, ki int) interface{} {
	f := interface{}(&hostFunc{name: "method:" + name, fn: func(rt *jsrt, this uint64, a []interface{}) interface{} {
		return rt.rubyKernelInvoke(name, a)
	}})
	if rt.rubyArities == nil {
		rt.rubyArities = map[interface{}]rubyArity{}
	}
	if rubyKernelMethodArity[ki] < 0 {
		rt.rubyArities[f] = rubyArity{req: -rubyKernelMethodArity[ki] - 1, star: true}
	} else {
		rt.rubyArities[f] = rubyArity{req: rubyKernelMethodArity[ki]}
	}
	return rt.rubyMarkMethodVal(f, "Object(Kernel)", name)
}

// rubyOwnerName is the class name a bound Method prints as its owner.
func rubyOwnerName(self interface{}) string {
	if o, isObj := self.(*jsObject); isObj && rubyUserObj(o) {
		if c, isCls := o.props["__class"].(*jsObject); isCls {
			if n, isStr := c.props["__name"].(string); isStr {
				return n
			}
		}
	}
	if kind := rubyClassName(self); kind != "" {
		return kind
	}
	return "Object"
}

// rubyStrInspect is String#inspect: the DOUBLE-QUOTED SOURCE form, and it is the
// escaping that makes it one. MRI escapes the quote, the backslash, the eight
// named control characters, a `#` that would start an interpolation (`#{`, `#$`,
// `#@` - a bare `#` is left alone), and every other control character as \uXXXX
// with UPPERCASE hex. Printable non-ASCII is left alone under a UTF-8 locale, so
// this walks BYTES and copies anything >= 0x80 through untouched. Mirrors
// strInspect of ruby-interpreter.abnf and rbStrInspect of lib/ruby-rt.metajs.
func rubyStrInspect(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '"':
			b.WriteString("\\\"")
		case '\\':
			b.WriteString("\\\\")
		case '\n':
			b.WriteString("\\n")
		case '\t':
			b.WriteString("\\t")
		case '\r':
			b.WriteString("\\r")
		case '\f':
			b.WriteString("\\f")
		case '\v':
			b.WriteString("\\v")
		case '\b':
			b.WriteString("\\b")
		case 7:
			b.WriteString("\\a")
		case 27:
			b.WriteString("\\e")
		case '#':
			if i+1 < len(s) && (s[i+1] == '{' || s[i+1] == '$' || s[i+1] == '@') {
				b.WriteString("\\#")
			} else {
				b.WriteByte('#')
			}
		default:
			if c < 0x20 || c == 0x7f {
				fmt.Fprintf(&b, "\\u%04X", c)
			} else {
				b.WriteByte(c)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}

// rubyInspect is Ruby's #inspect: nil, quoted strings, :symbols, and the bracketed
// forms of Array and Hash.
func (rt *jsrt) rubyInspect(v interface{}) string {
	switch t := v.(type) {
	case jsNullT, jsUndefT:
		return "nil"
	case string:
		return rubyStrInspect(t)
	case jsSym:
		return ":" + t.s
	// Rational#inspect and Complex#inspect are PARENTHESIZED where #to_s is not -
	// `(2/3)` against `2/3`, `(3+4i)` against `3+4i` - and it is the one place the
	// two differ for a numeric.
	case jsRat:
		return "(" + rubyRatText(t) + ")"
	case jsCpx:
		return "(" + rt.rubyCpxText(t, true) + ")"
	case *jsArray:
		parts := make([]string, len(t.elems))
		for i, e := range t.elems {
			parts[i] = rt.rubyInspect(e)
		}
		return "[" + strJoin(parts, ", ") + "]"
	case *jsObject:
		// Range#inspect INSPECTS its endpoints where #to_s renders them, which is
		// the only place the two differ: `("a".."e").inspect' is `"a".."e"' and its
		// #to_s is `a..e'.
		if _, isRng := t.props["__rrange"]; isRng {
			return rt.rubyRangeText(t, true)
		}
		// Module#inspect is the NAME even when the class defines #to_s, since MRI
		// binds the alias to the original Module#to_s.
		if _, isCls := t.props["__isclass"]; isCls {
			if _, ok := t.props["__name"].(string); ok {
				return rt.rubyClassText(t, "inspect")
			}
		}
		if keys, vals, isDict := dictParts(t); isDict {
			parts := make([]string, len(keys.elems))
			for i := range keys.elems {
				// NO SPACES around the arrow: MRI's Hash#inspect writes
				// `{1=>2, "a"=>[1, 2]}`.
				parts[i] = rt.rubyInspect(keys.elems[i]) + "=>" + rt.rubyInspect(vals.elems[i])
			}
			return "{" + strJoin(parts, ", ") + "}"
		}
		// Exception#inspect / a plain instance's #inspect. MRI answers
		// `#<RuntimeError: boom>' - the CLASS NAME and #to_s - and the bare class
		// name when #to_s is empty. A class of its own that defines #inspect stays
		// in charge.
		if m, ok := rubyFindMethod(t, "inspect"); ok {
			return rt.rubyStr(rt.call(m, jsUndef, []interface{}{t}))
		}
		if rubyExcInst(t) {
			return rubyExcInspect(rubyInstClassName(t), rt.rubyStr(t))
		}
		// A Regexp / MatchData answers from the regexp half's own table.
		if rxIsRegexObj(t) || rxIsMatchObj(t) {
			if s, ok := rt.rxRegexMethod(t, "inspect", nil); ok {
				return rt.toString(s)
			}
		}
		// Object#inspect never calls a user #to_s: MRI's default form, with the
		// address and instance variables left off as in rubyStr.
		if _, isCls := t.props["__isclass"]; !isCls {
			if n := rubyInstClassName(t); n != "" {
				return "#<" + n + ">"
			}
		}
	}
	return rt.rubyStr(v)
}

// rubyClassText is a class object's rendering by name ("to_s" or "inspect"): a
// singleton or class method of that name (its own or inherited) answers, else the
// class name. Twin of rbClassText in lib/ruby-rt.metajs and classText of
// ruby-interpreter.abnf.
func (rt *jsrt) rubyClassText(o *jsObject, name string) string {
	m, ok := rubySingFind(o, name)
	for cls := interface{}(o); !ok && cls != nil; {
		clsObj, isObj := cls.(*jsObject)
		if !isObj {
			break
		}
		if sm, has := clsObj.props["$s$"+name]; has && isCallable(sm) {
			m, ok = sm, true
		}
		cls = clsObj.props["__super"]
	}
	if ok {
		return rt.rubyStr(rt.call(m, jsUndef, []interface{}{o}))
	}
	n, _ := o.props["__name"].(string)
	return n
}

// rubyInstClassName is the __name of an instance's class, "" when it has none.
func rubyInstClassName(o *jsObject) string {
	if cls, ok := o.props["__class"].(*jsObject); ok {
		if n, isStr := cls.props["__name"].(string); isStr {
			return n
		}
	}
	return ""
}

// rubyExcInspect is MRI's Exception#inspect: "#<Class: to_s>", or the bare class
// name when to_s is empty (`class E < StandardError; def to_s; ""; end; end'
// inspects as "E"). Mirrored by rbExcInspect of lib/ruby-rt.metajs and
// excInspect of ruby-interpreter.abnf.
func rubyExcInspect(name, msg string) string {
	if name == "" {
		return msg
	}
	if msg == "" {
		return name
	}
	return "#<" + name + ": " + msg + ">"
}

// rubyEq is Ruby's ==: numeric across the whole tower (2.0 == 2, 1r/2r == 0.5), a
// user class's own == when it defines one, and the structural comparison of
// Array/Hash (pyEqual) otherwise. A Symbol only equals another Symbol.
func (rt *jsrt) rubyEq(l, r interface{}) bool {
	if rubyIsNum(l) && rubyIsNum(r) {
		_, lc := l.(jsCpx)
		if _, rc := r.(jsCpx); lc || rc {
			return rt.rubyEq(rubyReOf(l), rubyReOf(r)) && rt.rubyEq(rubyImOf(l), rubyImOf(r))
		}
		if c, ok := rt.rubyRatExact(l, r); ok {
			return c == 0
		}
		if c, ok := rubyBigCmp(l, r); ok {
			return c == 0
		}
		return rubyToF(l) == rubyToF(r)
	}
	// Two Ranges compare by their endpoints, so `(1..3) == (1..3)' is true and a
	// Range can be a Hash KEY - rubyEql, which is what rubyDictFind scans with,
	// lands here.
	if lo, isL := rubyIsRange(l); isL {
		if ro, isR := rubyIsRange(r); isR {
			return rt.rubyRangeEq(lo, ro)
		}
	}
	if rubyUserObj(l) {
		if m, ok := rubyFindMethod(l, "=="); ok {
			return rubyTruthy(rt.call(m, jsUndef, []interface{}{l, r}))
		}
	}
	// An ABSENT argument reaches this half as UNDEFINED (js_arg / js_rblockarg
	// answer undefined for one that was not passed) and ruby-interpreter.abnf as
	// nil. Both spell Ruby's nil, so `b == nil` and `[a, b] == [1, nil]` have to
	// hold either way.
	if isUndefOrNull(l) || isUndefOrNull(r) {
		return isUndefOrNull(l) && isUndefOrNull(r)
	}
	// An exception compares equal to its own message, the same deliberate
	// divergence rubyEq of ruby-interpreter.abnf makes, so a program that raises a
	// plain string still reads its rescue value as that string.
	if s, isStr := r.(string); isStr && rubyExcObj(l) {
		return rt.rubyStr(l) == s
	}
	if s, isStr := l.(string); isStr && rubyExcObj(r) {
		return rt.rubyStr(r) == s
	}
	// true / false are their own classes in Ruby, so `false == 0` is FALSE - where
	// the shared pyEqual rubyStructEq ends in applies Python's rule and says true
	// (`[true,false,nil].include?(0)` is false).
	if lb, isB := l.(bool); isB {
		rb, ok := r.(bool)
		return ok && rb == lb
	}
	if _, isB := r.(bool); isB {
		return false
	}
	// Two Regexps are == when their source and options are.
	if rxIsRegexObj(l) && rxIsRegexObj(r) {
		lo, ro := l.(*jsObject), r.(*jsObject)
		lf, _ := lo.props["__rxflags"].(string)
		rf, _ := ro.props["__rxflags"].(string)
		return lo.props["__rxsrc"] == ro.props["__rxsrc"] && rubyReOpts(lf, false) == rubyReOpts(rf, false)
	}
	return rt.rubyStructEq(l, r)
}

// rubyStructEq is Array#== and Hash#==: element by element and key by key, with
// each element compared by RUBY'S ==. It cannot be pyEqual: pyEqual recurses
// into an array with ITSELF, so an element pair would never reach rubyEq's
// numeric tower and `[1] == [1.0]` would be false where MRI says true. The Hash
// arm looks its keys up with rubyDictFind, so an Array key is found the same way
// it is everywhere else.
func (rt *jsrt) rubyStructEq(l, r interface{}) bool {
	if la, isArr := l.(*jsArray); isArr {
		ra, ok := r.(*jsArray)
		if !ok || len(la.elems) != len(ra.elems) {
			return false
		}
		for i := range la.elems {
			if !rt.rubyEq(la.elems[i], ra.elems[i]) {
				return false
			}
		}
		return true
	}
	if lk, lv, isDict := dictParts(l); isDict {
		rk, rv, ok := dictParts(r)
		if !ok || len(lk.elems) != len(rk.elems) {
			return false
		}
		for i := range lk.elems {
			at := rt.rubyDictFind(rk, lk.elems[i])
			if at < 0 || !rt.rubyEq(lv.elems[i], rv.elems[at]) {
				return false
			}
		}
		return true
	}
	return rt.pyEqual(l, r)
}

// rubyDictFind is dictFind under RUBY's key rule, which is not JavaScript's: an
// ARRAY is a VALUE key (Array#hash and Array#eql? are structural), so
// `{[1] => 5}[[1]]` is 5. A Hash keys the same way.
//
// It is a WRAPPER and not a change to dictFind, exactly as the big-Integer arm of
// layer 2's rbDictFind is: an array key is not dictKeyable, so it never enters the
// index and the shared find already ends in a scan - this arm runs only when that
// scan MISSED, so a hit costs what it did and an unchanged program pays one type
// test. dictFind and strictEq are shared by nine languages in which two arrays are
// two objects (python's `{(1,): 5}` is a tuple, a JS `Map` keys by identity), and
// that is why the rule lives here. The twins are core.keyEq in
// languages/ruby-interpreter.abnf and rbDictFind in languages/lib/ruby-rt.metajs.
//
// One MRI corner is deliberately not modelled, in all three engines alike: MRI
// hashes a key ONCE at insertion, so mutating a key array afterwards loses the
// entry until Hash#rehash. This compares at lookup time and still finds it.
func (rt *jsrt) rubyDictFind(keys *jsArray, k interface{}) int {
	if i := rt.dictFind(keys, k); i >= 0 {
		return i
	}
	if !rubyValueKey(k) {
		return -1
	}
	for i, e := range keys.elems {
		if rubyValueKey(e) && rt.rubyEq(e, k) {
			return i
		}
	}
	return -1
}

// rubyEql is Ruby's eql?: == that does NOT convert across classes, so
// `1.eql?(1.0)` is false where `1 == 1.0` is true. It is what Array#uniq
// compares with (and what Hash keying is defined on).
func (rt *jsrt) rubyEql(a, b interface{}) bool {
	return rubyClassName(a) == rubyClassName(b) && rt.rubyEq(a, b)
}

// rubyArrIndex is the first position at which an element is Ruby-== to v, or -1.
// Array#include?, #index and #find_index are all defined on ==.
func (rt *jsrt) rubyArrIndex(t *jsArray, v interface{}) int {
	for i, e := range t.elems {
		if rt.rubyEq(e, v) {
			return i
		}
	}
	return -1
}

// rubyIdxOrNil turns a find result into Array#index's answer: the position, or
// nil (not -1) when there was no match.
func rubyIdxOrNil(i int) interface{} {
	if i < 0 {
		return jsNull
	}
	return float64(i)
}

// rubyValueKey answers whether a key is one of the two STRUCTURALLY compared
// shapes: an Array, or a Hash (a dict object).
func rubyValueKey(v interface{}) bool {
	if _, isArr := v.(*jsArray); isArr {
		return true
	}
	if o, isObj := v.(*jsObject); isObj {
		if _, _, isDict := dictParts(o); isDict {
			return true
		}
		// A Range is a value key too - `h[1..3]' has to find the entry `h[1..3] = v'
		// made, and two range literals are two distinct objects that strictEq (which
		// rt.dictFind scans with) can never match.
		if _, isRng := o.props["__rrange"]; isRng {
			return true
		}
	}
	return false
}

// rubySpaceship is <=>: nil (a Go nil result reported as jsNull by the caller) for
// values that cannot be compared, which is exactly Ruby's answer.
func (rt *jsrt) rubySpaceship(l, r interface{}) (int, bool) {
	if rubyUserObj(l) {
		if m, ok := rubyFindMethod(l, "<=>"); ok {
			return int(rubyToF(rt.call(m, jsUndef, []interface{}{l, r}))), true
		}
		return 0, false
	}
	if rubyIsNum(l) && rubyIsNum(r) {
		if c, ok := rt.rubyRatExact(l, r); ok {
			return c, true
		}
		if c, ok := rubyBigCmp(l, r); ok {
			return c, true
		}
		x, y := rubyToF(l), rubyToF(r)
		switch {
		case x < y:
			return -1, true
		case x > y:
			return 1, true
		}
		return 0, true
	}
	if ls, ok := l.(string); ok {
		if rs, ok2 := r.(string); ok2 {
			switch {
			case ls < rs:
				return -1, true
			case ls > rs:
				return 1, true
			}
			return 0, true
		}
	}
	// A Symbol IS Comparable in Ruby - Symbol#<=> compares the NAMES - so
	// `[:b, :a].sort`, #min, #max, #between? and `:a < :b` all work. It is the
	// name comparison and nothing else: `:a <=> 1` is nil, i.e. the ok=false at the
	// bottom.
	if lsym, ok := l.(jsSym); ok {
		if rsym, ok2 := r.(jsSym); ok2 {
			switch {
			case lsym.s < rsym.s:
				return -1, true
			case lsym.s > rsym.s:
				return 1, true
			}
			return 0, true
		}
	}
	if la, ok := l.(*jsArray); ok {
		if ra, ok2 := r.(*jsArray); ok2 {
			// ONE incomparable element pair makes the WHOLE comparison nil -
			// `[1, "a"] <=> [1, 2]` is nil, not 0 - and the answer is NORMALISED to
			// -1/0/1, so `[1,2,3] <=> [1]` is 1, not 2.
			for i := 0; i < len(la.elems) && i < len(ra.elems); i++ {
				c, ok3 := rt.rubySpaceship(la.elems[i], ra.elems[i])
				if !ok3 {
					return 0, false
				}
				if c != 0 {
					return rubyCmpSign(c), true
				}
			}
			return rubyCmpSign(len(la.elems) - len(ra.elems)), true
		}
	}
	return 0, false
}

func rubyCmpSign(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	}
	return 0
}

// rubyCmpName is MRI's rb_cmperr rendering: the FIRST operand is named by its
// class, the second by its class UNLESS it is a special constant or a Float, which
// are inspected - hence "comparison of NilClass with 1 failed" for `[1, nil].max`.
func (rt *jsrt) rubyCmpName(v interface{}) string {
	if isUndefOrNull(v) {
		return "nil"
	}
	switch v.(type) {
	case bool, float64, jsFlo, jsSym:
		return rt.rubyInspect(v)
	}
	if n := rubyClassName(v); n != "" {
		return n
	}
	return "Object"
}

// rubyCmp backs < > <= >=, which raise in Ruby when the values cannot be compared.
func (rt *jsrt) rubyCmp(l, r interface{}) int {
	c, ok := rt.rubySpaceship(l, r)
	if !ok {
		// A CATCHABLE ArgumentError, not a host abort: MRI's sort raises one and
		// a program may rescue it. The message is rb_cmperr's.
		ln := rubyClassName(l)
		if ln == "" {
			ln = "Object"
		}
		panic(&jsThrown{value: rt.rubyExc(rt.rubyBuiltinClass("ArgumentError"),
			[]interface{}{"comparison of " + ln + " with " + rt.rubyCmpName(r) + " failed"})})
	}
	return c
}

// rubyFloatWord is Ruby's rendering of a NON-FINITE double under %f/%e/%E/%g/%G:
// MRI prints "Inf", "-Inf" and "NaN" for every one of them, in that case whatever
// the directive's own case is ("%G" % Float::INFINITY is "Inf", not "INF"), and it
// pads them with SPACES even when the 0 flag is set ("%010f" % inf is "       Inf").
func rubyFloatWord(v float64) (string, bool) {
	if math.IsNaN(v) {
		return "NaN", true
	}
	if math.IsInf(v, 1) {
		return "Inf", true
	}
	if math.IsInf(v, -1) {
		return "-Inf", true
	}
	return "", false
}

// rubyAltPoint forces a decimal point into a rendered float that has none - what
// the # flag asks for ("%#.0f" % 1.0 is "1.", "%#.1g" % 1234.0 is "1.e+03"). The
// point goes at the END of the mantissa, in front of any exponent suffix.
func rubyAltPoint(s string) string {
	at := strings.IndexAny(s, "eE")
	mant, tail := s, ""
	if at >= 0 {
		mant, tail = s[:at], s[at:]
	}
	if strings.Contains(mant, ".") {
		return s
	}
	return mant + "." + tail
}

// rubyGenAlt is %g's shape WITHOUT the trailing-zero stripping - the # flag's
// "alternate form" ("%#g" % 100.0 is "100.000" where "%g" is "100"). The shape
// decision is the same one strconv's 'g' makes and has to be read AFTER the
// rounding, so the %e rendering is what answers it.
func rubyGenAlt(v float64, prec int) string {
	e := strconv.FormatFloat(v, 'e', prec-1, 64)
	at := strings.IndexByte(e, 'e')
	ex, _ := strconv.Atoi(e[at+1:])
	if ex < -4 || ex >= prec {
		return rubyAltPoint(e)
	}
	return rubyAltPoint(strconv.FormatFloat(v, 'f', prec-1-ex, 64))
}

// rubyBasePrefix is the # flag on %x/%X/%o/%b: "0x", "0X", "0" and "0b" in front
// of the digits, and NOTHING in front of a zero ("%#x" % 0 is "0"). It returns
// the prefix only; the caller keeps its length so the 0 flag can pad BEHIND it
// ("%#010x" % 255 is "0x000000ff", not "00000000xff").
//
// The rule is NOT "the digits are not zero": a precision pads them, so
// "%#.5x" % 0 is "00000" in MRI, with no prefix, and the digit string is
// neither "0" nor "". The test is on the VALUE.
// Octal is a different rule again and is answered by rubyIntBody: "#" on %o means
// "make sure a leading zero is there", so "%#.5o" % 255 is "00377" and not
// "000377", and MRI omits it entirely from the ".." two's-complement form.
func rubyBasePrefix(conv byte, v *big.Int) string {
	if v.Sign() == 0 {
		return ""
	}
	switch conv {
	case 'x':
		return "0x"
	case 'X':
		return "0X"
	case 'b':
		return "0b"
	case 'B':
		return "0B"
	}
	return ""
}

// rubyCharCode is %c on an Integer: the CODE POINT, not the byte. MRI writes
// "%c" % 128512 as the astral character itself, so this is Go's rune conversion
// and layer 2's is a surrogate pair.
//
// Out of range is the one place this does not follow MRI: 0x110000 raises
// ArgumentError there and a negative renders a raw invalid byte, neither of
// which any engine here can express, so both answer the empty string.
func rubyCharCode(cp int) string {
	if cp < 0 || cp > 0x10FFFF {
		return ""
	}
	return string(rune(cp))
}

// rubyFirstChar is %c on a String. MRI demands EXACTLY one character and raises
// ArgumentError otherwise; this takes the first one and is documented as such.
// MRI's sprintf counts CHARACTERS in a width and in a precision, not bytes:
// "%10s" % "café" is six spaces and four characters, and its result has
// String#length 10 and #bytesize 11 (the byte rule in
// docs/working-on-this-project.md chapter 7.7 does not hold for Ruby).
//
// rbCharCount in languages/lib/ruby-rt.metajs and charCount in
// languages/ruby-interpreter.abnf are the twins, and they count CODE POINTS -
// a surrogate pair is one character to MRI ("%5c" % 128512 is four spaces).
func rubyCharCount(s string) int { return utf8.RuneCountInString(s) }

// The first `n` CHARACTERS of a string, which is what a %s or %p precision cuts
// to: "%.1s" % "日本語" is "日" in MRI.
func rubyCharCut(s string, n int) string {
	i := 0
	for at := range s {
		if i == n {
			return s[:at]
		}
		i++
	}
	return s
}

// ----- the three integer directives ----------------------------------------
//
// Past 2^53 neither `strconv.FormatInt(int64(f), base)` nor jsNumString will
// do: int64() SATURATES and jsNumString prints the double's SHORTEST form
// ("1e+30"). MRI converts the Float with to_i - truncation toward zero -
// and then prints the resulting Integer EXACTLY, all 31 digits of
// 1e30.to_i == 1000000000000000019884624838656. Integer formatting is
// unchanged between ruby 2.6 and 3.x.
//
// A double is an integer times a power of two, so `big.Float.Int` is exact and
// `Int.Text(base)` is the answer for every base here. Layer 2 has no bignum
// multiply, so rbMagDigits there takes the binary expansion and regroups it in
// threes and fours; the two agree by construction.
const rubyDigitAlphabet = "0123456789abcdefghijklmnopqrstuvwxyz"

func rubyDigitVal(c byte) int {
	if c >= '0' && c <= '9' {
		return int(c - '0')
	}
	if c >= 'a' && c <= 'z' {
		return int(c-'a') + 10
	}
	return 0
}

// The exact base-`base` digits of a non-negative magnitude.
func rubyMagDigits(mag *big.Int, base int) string {
	return mag.Text(base)
}

// `s` minus one, in base `base`. The operand is a magnitude of at least 1, so the
// borrow always finds a digit to take from.
func rubyDecOne(s string, base int) string {
	b := []byte(s)
	for i := len(b) - 1; i >= 0; i-- {
		if d := rubyDigitVal(b[i]); d > 0 {
			b[i] = rubyDigitAlphabet[d-1]
			break
		}
		b[i] = rubyDigitAlphabet[base-1]
	}
	k := 0
	for k < len(b)-1 && b[k] == '0' {
		k++
	}
	return string(b[k:])
}

// MRI's INFINITE two's-complement digits for a negative value, without the ".."
// that announces them: "%x" % -5 is "..fb", "%o" is "..73", "%b" is "..1011".
//
// ~n is n's magnitude minus one with every digit complemented, which is exact on
// the digit string and needs no bignum subtract; the leading digit of the
// infinite run of (base-1)s is then written once. MRI collapses that run to
// exactly ONE digit, so -1 is "..f" and not "..ff".
func rubyTwosDigits(mag *big.Int, base int) string {
	src := rubyDecOne(rubyMagDigits(mag, base), base)
	b := make([]byte, len(src))
	for i := 0; i < len(src); i++ {
		b[i] = rubyDigitAlphabet[base-1-rubyDigitVal(src[i])]
	}
	mx := rubyDigitAlphabet[base-1]
	if len(b) == 0 || b[0] != mx {
		return string(mx) + string(b)
	}
	return string(b)
}

// rubyIntBody assembles ONE integer directive: the sign or the ".." marker, the #
// base prefix, and the digits with any precision applied. It answers the fill
// CHARACTER for the 0 flag as well, because the ".." form pads with the base's
// largest digit and not with zeros ("%010x" % -5 is "..fffffb"), and `keep`, the
// number of leading characters those fill characters must go behind.
//
// `sign` is the + or SPACE flag, which switches MRI back to sign-and-magnitude:
// "%+x" % -5 is "-5", not "..fb".
func rubyIntBody(f *big.Int, base int, upper, alt, sign bool, prec int) (body, prefix string, fill byte, keep int) {
	neg := f.Sign() < 0
	mag := new(big.Int).Abs(f)
	fill, keep = '0', 0
	twos := false
	var digits string
	if base == 10 || !neg || sign {
		digits = rubyMagDigits(mag, base)
		if prec >= 0 {
			if digits == "0" && prec == 0 {
				digits = ""
			}
			for len(digits) < prec {
				digits = "0" + digits
			}
		}
	} else {
		twos = true
		mx := rubyDigitAlphabet[base-1]
		digits = rubyTwosDigits(mag, base)
		// A precision counts the ".." as two of its characters: "%.5x" % -1 is
		// "..fff" and "%.3x" % -1 is "..f".
		for len(digits) < prec-2 {
			digits = string(mx) + digits
		}
		fill = mx
	}
	if alt {
		if base == 8 {
			if !twos && (len(digits) == 0 || digits[0] != '0') {
				prefix = "0"
			}
		} else if base == 16 {
			if upper {
				prefix = rubyBasePrefix('X', f)
			} else {
				prefix = rubyBasePrefix('x', f)
			}
		} else if base == 2 {
			// %B is %b with an UPPERCASE prefix: "%#B" % 5 is "0B101". The digits
			// are 0 and 1, so `upper` changes nothing else in base 2 and the
			// caller passes it for 'B' exactly as it does for 'X'.
			if upper {
				prefix = rubyBasePrefix('B', f)
			} else {
				prefix = rubyBasePrefix('b', f)
			}
		}
	}
	if twos {
		digits = ".." + digits
		keep = 2
	}
	if upper {
		digits = strings.ToUpper(digits)
		fill = byte(strings.ToUpper(string(fill))[0])
	}
	body = prefix + digits
	keep += len(prefix)
	if neg && !twos {
		body = "-" + body
		keep++
	}
	return
}

// ----- what a numeric directive may be GIVEN --------------------------------
//
// MRI's numeric directives do not read a number out of the operand, they CONVERT it with
// Kernel#Integer / Kernel#Float rules - so they raise on a value that has no number in it,
// and they PARSE one that has a number in its text:
//
//	"%d" % [nil]    TypeError     can't convert nil into Integer
//	"%d" % [true]   TypeError     can't convert true into Integer
//	"%d" % [:s]     TypeError     can't convert Symbol into Integer
//	"%d" % ["x"]    ArgumentError invalid value for Integer(): "x"
//	"%d" % ["12"]   "12"
//	"%f" % [nil]    TypeError can't convert nil into Float
//	"%c" % [nil]    TypeError     no implicit conversion from nil to integer
//
// nil / true / false name THEMSELVES in the message and everything else names its CLASS,
// which is MRI's wording and not a simplification. %c has a message of its own. The twins
// are fmtIntOperand / fmtFloOperand / fmtCharOperand in ruby-interpreter.abnf and
// rbFmtIntOperand / rbFmtFloOperand / rbFmtCharOperand in languages/lib/ruby-rt.metajs.
// rubyOwnerName and NOT rubyClassName: rubyClassName answers "" for a USER OBJECT,
// where the message must name the class ("Plain"). rubyOwnerName reads
// __class.__name first.
func rubyFmtConvName(v interface{}) string {
	if isUndefOrNull(v) {
		return "nil"
	}
	if b, isB := v.(bool); isB {
		if b {
			return "true"
		}
		return "false"
	}
	return rubyOwnerName(v)
}

// A user object is CONVERTED first, with the same two names Kernel#Integer tries
// (#to_int, then #to_i) and #to_f for a float directive - MRI's `"%d" % obj` is 7 when
// obj answers to_int. Same find-then-call shape rubyStr already uses for #to_s.
func (rt *jsrt) rubyFmtUserConv(v interface{}, n1, n2 string) (interface{}, bool) {
	if !rubyUserObj(v) {
		return nil, false
	}
	if m, has := rubyFindMethod(v, n1); has {
		return rt.call(m, v, nil), true
	}
	if n2 != "" {
		if m, has := rubyFindMethod(v, n2); has {
			return rt.call(m, v, nil), true
		}
	}
	return nil, false
}

func rubyFmtIsNumV(v interface{}) bool {
	switch v.(type) {
	case float64, *jsBigInt, jsFlo, jsRat, jsCpx:
		return true
	}
	return false
}

func (rt *jsrt) rubyFmtIntOperand(v interface{}) interface{} {
	if rubyFmtIsNumV(v) {
		return v
	}
	if s, isStr := v.(string); isStr {
		if n, ok := rubyStrToInum(s, 0, true); ok {
			return rt.rubyBigNarrow(n)
		}
		rt.rubyRaiseErr("ArgumentError", "invalid value for Integer(): "+rubyStrInspect(s))
	}
	if uc, has := rt.rubyFmtUserConv(v, "to_int", "to_i"); has && rubyFmtIsNumV(uc) {
		return uc
	}
	rt.rubyRaiseErr("TypeError", "can't convert "+rubyFmtConvName(v)+" into Integer")
	return float64(0)
}

func (rt *jsrt) rubyFmtFloOperand(v interface{}) float64 {
	if rubyFmtIsNumV(v) {
		return rubyToF(v)
	}
	if s, isStr := v.(string); isStr {
		if f, ok := rubyStrToFloat(s); ok {
			return f
		}
		rt.rubyRaiseErr("ArgumentError", "invalid value for Float(): "+rubyStrInspect(s))
	}
	if uf, has := rt.rubyFmtUserConv(v, "to_f", ""); has && rubyFmtIsNumV(uf) {
		return rubyToF(uf)
	}
	rt.rubyRaiseErr("TypeError", "can't convert "+rubyFmtConvName(v)+" into Float")
	return 0
}

// %c's message is NOT the Integer one, and MRI writes it TWO ways: nil is
// "no implicit conversion from nil to integer" and everything else is
// "no implicit conversion of Symbol into Integer" - a different preposition and a
// different case on "integer". Both are /usr/bin/ruby 2.6.10's own text.
func (rt *jsrt) rubyFmtCharOperand(v interface{}) float64 {
	if rubyFmtIsNumV(v) {
		return rubyToF(v)
	}
	if uc, has := rt.rubyFmtUserConv(v, "to_int", "to_i"); has && rubyFmtIsNumV(uc) {
		return rubyToF(uc)
	}
	if isUndefOrNull(v) {
		rt.rubyRaiseErr("TypeError", "no implicit conversion from nil to integer")
	}
	rt.rubyRaiseErr("TypeError", "no implicit conversion of "+rubyFmtConvName(v)+" into Integer")
	return 0
}

// rubyFormat is Kernel#format / String#% - the printf directives Ruby shares with C:
// %[-][0][+][ ][#][width][.prec](d|i|u|f|s|p|c|x|X|o|b|B|e|E|g|G|%). The right
// operand is the argument, or an array of them.
//
// MRI: %u is %d (its 1.8 unsigned meaning is long gone), %B is %b with an
// uppercase # prefix, %c is a code point and %p is #inspect.
//
// Not implemented: %a/%A (the hex-float form), and the argument
// selectors %N$, %<name> and %{name} and the * width/precision.
//
// %g/%G is C's "shortest of the two shapes": the value rounded to P significant
// digits (P defaults to 6, and P == 0 means 1) is written in %e form when its
// decimal exponent is below -4 or at least P, and in %f form otherwise, with the
// trailing zeros of the fraction - and a bare trailing point - removed. That is
// exactly strconv.FormatFloat(v, 'g', P, 64).
func (rt *jsrt) rubyFormat(spec string, arg interface{}) string {
	args := []interface{}{arg}
	if a, ok := arg.(*jsArray); ok {
		args = a.elems
	}
	out := make([]byte, 0, len(spec))
	ai := 0
	next := func() interface{} {
		if ai < len(args) {
			v := args[ai]
			ai++
			return v
		}
		ai++
		return jsNull
	}
	for i := 0; i < len(spec); i++ {
		if spec[i] != '%' {
			out = append(out, spec[i])
			continue
		}
		i++
		if i >= len(spec) {
			out = append(out, '%')
			break
		}
		if spec[i] == '%' {
			out = append(out, '%')
			continue
		}
		left, zero, plus, space, alt := false, false, false, false, false
		for i < len(spec) && (spec[i] == '-' || spec[i] == '0' || spec[i] == '+' || spec[i] == ' ' || spec[i] == '#') {
			switch spec[i] {
			case '-':
				left = true
			case '0':
				zero = true
			case '+':
				plus = true
			case ' ':
				space = true
			case '#':
				alt = true
			}
			i++
		}
		width := 0
		for i < len(spec) && spec[i] >= '0' && spec[i] <= '9' {
			width = width*10 + int(spec[i]-'0')
			i++
		}
		prec := -1
		if i < len(spec) && spec[i] == '.' {
			i++
			prec = 0
			for i < len(spec) && spec[i] >= '0' && spec[i] <= '9' {
				prec = prec*10 + int(spec[i]-'0')
				i++
			}
		}
		if i >= len(spec) {
			break
		}
		var body string
		nonFinite := false
		prefix := ""
		// The 0 flag's fill character and the number of leading characters it must
		// go behind. Only the integer directives move them off their defaults; keep
		// < 0 means "work the sign and the prefix out below, as before".
		fill, keep := byte('0'), -1
		intPrec := false
		switch spec[i] {
		case 'd', 'i', 'u', 'x', 'X', 'o', 'b', 'B':
			base := 10
			switch spec[i] {
			case 'x', 'X':
				base = 16
			case 'o':
				base = 8
			case 'b', 'B':
				base = 2
			}
			// The argument's EXACT integer value: an Integer past 2^53 is a big
			// and carries all of it, and a Float is truncated toward zero (MRI's
			// %d converts with to_i) and then expanded exactly.
			av := rt.rubyFmtIntOperand(next())
			var iv *big.Int
			if bi, isBig := av.(*jsBigInt); isBig {
				iv = bi.v
			}
			f := math.Trunc(rubyToF(av))
			if iv == nil && !math.IsNaN(f) && !math.IsInf(f, 0) {
				iv, _ = new(big.Float).SetFloat64(f).Int(nil)
			}
			if iv == nil {
				// Unreachable from real Ruby - Float::INFINITY.to_i raises
				// FloatDomainError - and left as is rather than
				// invented: jsNumString's word, or int64's arm64 saturation.
				if base == 10 {
					body = jsNumString(f)
				} else {
					body = strconv.FormatInt(int64(f), base)
					if spec[i] == 'X' {
						body = strings.ToUpper(body)
					}
				}
				break
			}
			body, prefix, fill, keep = rubyIntBody(iv, base, spec[i] == 'X' || spec[i] == 'B', alt, plus || space, prec)
			// A precision turns the 0 flag OFF for an integer directive, as it does
			// in C: "%08.3x" % 1 is "     001".
			intPrec = prec >= 0
		case 'f':
			if prec < 0 {
				prec = 6
			}
			f := rt.rubyFmtFloOperand(next())
			if w, bad := rubyFloatWord(f); bad {
				body, nonFinite = w, true
			} else {
				body = strconv.FormatFloat(f, 'f', prec, 64)
				if alt {
					body = rubyAltPoint(body)
				}
			}
		case 'e', 'E':
			if prec < 0 {
				prec = 6
			}
			f := rt.rubyFmtFloOperand(next())
			if w, bad := rubyFloatWord(f); bad {
				body, nonFinite = w, true
			} else {
				body = strconv.FormatFloat(f, 'e', prec, 64)
				if alt {
					body = rubyAltPoint(body)
				}
				if spec[i] == 'E' {
					body = strings.ToUpper(body)
				}
			}
		case 'g', 'G':
			if prec < 0 {
				prec = 6
			}
			if prec == 0 {
				prec = 1
			}
			f := rt.rubyFmtFloOperand(next())
			if w, bad := rubyFloatWord(f); bad {
				body, nonFinite = w, true
			} else if alt {
				body = rubyGenAlt(f, prec)
			} else {
				body = strconv.FormatFloat(f, 'g', prec, 64)
			}
			if !nonFinite && spec[i] == 'G' {
				body = strings.ToUpper(body)
			}
		case 's':
			body = rt.rubyStr(next())
			if prec >= 0 && prec < rubyCharCount(body) {
				body = rubyCharCut(body, prec)
			}
		case 'c':
			// %c is ONE character: a code point from an Integer (a Float is
			// truncated, as it is under %d) or the first character of a String.
			// It is not a numeric directive - "%05c" % 65 is "    A" - and MRI
			// ignores a precision on it outright ("%.0c" % 65 is "A").
			av := next()
			if s, isStr := av.(string); isStr {
				// A String operand must be EXACTLY one character. This took the
				// first and said so at the site; MRI raises, and the message is
				// its own.
				if rubyCharCount(s) != 1 {
					rt.rubyRaiseErr("ArgumentError", "%c requires a character")
				}
				body = s
			} else {
				body = rubyCharCode(int(math.Trunc(rt.rubyFmtCharOperand(av))))
			}
		case 'p':
			// %p is #inspect where %s is #to_s, and it truncates by BYTES the same
			// way: "%.2p" % "abcdef" is `"a'.
			body = rt.rubyInspect(next())
			if prec >= 0 && prec < rubyCharCount(body) {
				body = rubyCharCut(body, prec)
			}
		default:
			body = string(spec[i])
		}
		// The + and SPACE flags: "%+d" % 7 is "+7". They apply to the NUMERIC
		// directives only (a string never grows a sign), and they apply to Inf and
		// NaN too ("%+f" % Float::INFINITY is "+Inf").
		numeric := false
		switch spec[i] {
		case 'd', 'i', 'u', 'f', 'e', 'E', 'g', 'G', 'x', 'X', 'o', 'b', 'B':
			numeric = true
		}
		if numeric && (len(body) == 0 || body[0] != '-') {
			if plus {
				body = "+" + body
			} else if space {
				body = " " + body
			}
			if (plus || space) && keep >= 0 {
				keep++
			}
		}
		// The 0 flag is for NUMBERS: MRI's "%05s" % "ab" is "   ab". A non-finite
		// float is never zero-padded either, and neither is an integer directive
		// that carried a precision.
		pad := byte(' ')
		if zero && !left && !nonFinite && !intPrec && numeric {
			pad = fill
		}
		// The zeros of the 0 flag go BEHIND the sign and behind a # base prefix:
		// "%05d" % -42 is "-0042" and "%#010x" % 255 is "0x000000ff". For the ".."
		// two's-complement form they go behind that too, and they are f/7/1 rather
		// than 0: "%#010x" % -5 is "0x..fffffb".
		skip := keep
		if skip < 0 {
			skip = 0
			if len(body) > 0 && (body[0] == '-' || body[0] == '+' || body[0] == ' ') {
				skip = 1
			}
			skip += len(prefix)
		}
		for rubyCharCount(body) < width {
			if left {
				body += " "
			} else if pad != ' ' {
				body = body[:skip] + string(pad) + body[skip:]
			} else {
				body = string(pad) + body
			}
		}
		out = append(out, body...)
	}
	return string(out)
}

// rubyMethod is js_rmcall: the direct method dispatch of the Ruby compiler
// (ruby-to-llvm-ir.abnf). It mirrors the mcall of ruby-interpreter.abnf exactly
// for strings, arrays and hashes (Ruby nil-on-empty, Ruby truthiness, .each over
// key/value pairs) and delegates class instances - and everything else - to the
// shared memberCall. It is deliberately separate from memberCall so that Ruby's
// semantics never perturb the Kotlin/Java/Go/Python languages that also use
// js_mcall.
func (rt *jsrt) rubyMethod(target interface{}, name string, args []interface{}) interface{} {
	// Object#send / #__send__ / #public_send: the method NAME is the first
	// argument and the rest of the list (block wrapper included) is the call.
	// Re-entering the dispatcher rather than reaching into any one type's
	// table is what makes it work for a user object, an Array and a Proc alike.
	if name == "send" || name == "__send__" || name == "public_send" {
		if len(args) == 0 {
			rt.fail("no method name given")
		}
		mn := rt.rubyStr(args[0])
		// An OPERATOR reached by name (`3.send(:+, 4)`, "a".send(:+, "b"),
		// [1].send(:<<, 2)) is the one shape the type tables cannot answer: they
		// hold named methods, and + / << / <=> are evaluated by rubyBin.
		if len(args) == 2 && rubyIsBinOpName(mn) {
			// The comparisons are NOT rubyBin's (it has no <=> arm at all): they
			// are the js_rlt/js_rcmp family, exactly as the emitter spells them.
			switch mn {
			case "<", ">", "<=", ">=":
				c := rt.rubyCmp(target, args[1])
				return (mn == "<" && c < 0) || (mn == ">" && c > 0) ||
					(mn == "<=" && c <= 0) || (mn == ">=" && c >= 0)
			case "<=>":
				if c, ok := rt.rubySpaceship(target, args[1]); ok {
					return float64(c)
				}
				return jsNull
			}
			return rt.rubyLogicBin(mn, target, args[1])
		}
		if mn == "respond_to?" {
			return rt.rubyRespond(target, args[1:])
		}
		// A Regexp / MatchData method, or a String one taking a pattern, is the
		// regexp half's (js_rxmcall's dispatch).
		_, isStr := target.(string)
		if rxIsRegexObj(target) || rxIsMatchObj(target) || (isStr && rxWantsPattern(mn)) {
			if v, handled := rt.rxRegexMethod(target, mn, args[1:]); handled {
				return v
			}
		}
		return rt.rubyMethod(target, mn, args[1:])
	}
	// Ruby's universal reflective methods, answered for every kind of value.
	switch name {
	case "class":
		if kind := rt.rubyClassNameRT(target); kind != "" {
			return rt.rubyBuiltinClass(kind)
		}
	case "is_a?", "kind_of?", "instance_of?":
		if rt.rubyClassNameRT(target) != "" && !rubyUserObj(target) {
			return rt.rubyIsA(target, argAt(args, 0))
		}
	// Object#method on a builtin value: a Method that re-dispatches by name, or
	// MRI's NameError when the value does not respond to it. Twins: the "method"
	// arms of mcall in ruby-interpreter.abnf and rbMethod in lib/ruby-rt.metajs.
	case "method":
		o, isObj := target.(*jsObject)
		isCls := isObj && o.props["__isclass"] == true
		if kind := rt.rubyClassNameRT(target); kind != "" && !rubyUserObj(target) && !isCls && !isCallable(target) {
			nm := rt.rubyStr(argAt(args, 0))
			if rt.rubyRespond(target, []interface{}{jsSym{s: nm}}) != true {
				rt.rubyRaiseErr("NameError", "undefined method `"+nm+"' for class `"+kind+"'")
			}
			recv := target
			return rt.rubyMarkMethodVal(&hostFunc{name: "method:" + nm,
				fn: func(rt *jsrt, this uint64, a []interface{}) interface{} {
					return rt.rubyMethod(recv, "send", append([]interface{}{jsSym{s: nm}}, a...))
				}}, kind, nm)
		}
	case "nil?":
		return isUndefOrNull(target)
	// Kernel#itself / #tap, unless a user class defines its own.
	case "itself", "tap":
		if _, hasOwn := rubyFindMethod(target, name); !hasOwn {
			if name == "tap" {
				_, blk := rubyBlockArgs(args)
				rt.call(blk, jsUndef, []interface{}{target})
			}
			return target
		}
	// Object#hash / #object_id, unless a user class defines its own.
	case "hash", "object_id":
		if _, hasOwn := rubyFindMethod(target, name); !hasOwn {
			if name == "hash" {
				return rt.rubyHashOf(target)
			}
			return rt.rubyObjectId(target)
		}
	case "to_s", "inspect":
		// TrueClass / FalseClass, ahead of the shared memberCall.
		if bv, isBool := target.(bool); isBool {
			if bv {
				return "true"
			}
			return "false"
		}
		// A Proc / Method. Both are the same string in MRI.
		if isCallable(target) {
			return rt.rubyFuncRender(target)
		}
		// A user INSTANCE with no to_s / inspect of its own. rubyUserObj excludes a
		// Hash / Enumerator (neither carries __class), so their own arms below still
		// answer.
		if o, isObj := target.(*jsObject); isObj && rubyUserObj(o) {
			if _, hasOwn := rubyFindMethod(target, name); !hasOwn {
				if name == "inspect" {
					return rt.rubyInspect(o)
				}
				return rt.rubyStr(o)
			}
		}
	// Exception#backtrace: MRI's frame list. Only the RAISE SITE is modelled here
	// (js_rraiseat records it), so the answer is that one frame, in MRI's own
	// spelling - "prog.rb:2:in `<main>'". An exception raised by the runtime rather
	// than by a `raise', and one that was never raised at all, has no site and
	// answers nil, which is what MRI does for an un-raised exception object.
	case "backtrace":
		if o, isObj := target.(*jsObject); isObj && rubyExcInst(o) {
			if _, hasOwn := rubyFindMethod(target, name); !hasOwn {
				if where, isStr := o.props["__rwhere"].(string); isStr {
					return &jsArray{elems: []interface{}{where}}
				}
				return jsNull
			}
		}
	// NameError#receiver (and #name, in the Method#name arm below): the hidden
	// slots rubyNameErr fills, nil for a NameError the program raised itself.
	case "receiver":
		if v, ok := rt.rubyNameErrSlot(target, name, "__nmrecv"); ok {
			return v
		}
	case "message", "full_message":
		// Exception#message: the raise argument, or the class name. Only for an
		// INSTANCE that does not define a message method of its own, so a user
		// class named message stays in charge.
		if o, isObj := target.(*jsObject); isObj {
			if _, isCls := o.props["__isclass"]; !isCls {
				if _, hasOwn := rubyFindMethod(target, name); !hasOwn {
					// MRI's Exception#message IS #to_s, so a subclass that
					// overrides to_s changes its message too. rubyStr prefers a
					// user to_s and falls back to the raise argument.
					if rubyExcInst(o) {
						return rt.rubyStr(o)
					}
					if msg, ok := rt.rubyExcMessage(o); ok {
						return msg
					}
				}
			}
		}
	// Object#dup / #clone on a user INSTANCE, only when the class defines neither -
	// so a user #dup still wins, exactly as the to_s / inspect arm above works. A
	// String / Array / Hash keeps its own arm further down.
	case "dup", "clone":
		if o, isObj := target.(*jsObject); isObj && rubyUserObj(o) {
			if _, hasOwn := rubyFindMethod(target, name); !hasOwn {
				return rubySingCopy(o, name == "clone")
			}
		}
	case "instance_variable_get", "instance_variable_set":
		if o, isObj := target.(*jsObject); isObj && (rubyUserObj(o) || rubyClassObj(o)) {
			if _, hasOwn := rubyFindMethod(target, name); !hasOwn {
				return rt.rubyIvarAccess(o, name, args)
			}
		}
	case "instance_variables":
		if o, isObj := target.(*jsObject); isObj && (rubyUserObj(o) || rubyClassObj(o)) {
			if _, hasOwn := rubyFindMethod(target, name); !hasOwn {
				return rubyIvarNames(o)
			}
		} else if rt.rubyRespondClass(target) != "" {
			// A builtin value carries no instance variables.
			return &jsArray{}
		}
	case "frozen?":
		_, isStr := target.(string)
		return isStr || rubyIsNum(target)
	case "equal?":
		return rubyIdenticalP(target, argAt(args, 0))
	case "eql?":
		if _, hasOwn := rubyFindMethod(target, name); !hasOwn {
			if rt.rubyRespondClass(target) == "" || isCallable(target) {
				return rubyIdenticalP(target, argAt(args, 0))
			}
			return rt.rubyEqlDeep(target, argAt(args, 0))
		}
	case "call", "()", "yield", "[]", "===":
		// A Proc / lambda / block: p.call(x), p.(x), p[x] and p.yield(x).
		if isCallable(target) {
			return rt.call(target, jsUndef, args)
		}
	case "lambda?":
		if isCallable(target) {
			return rt.rubyLambdas[target]
		}
	case "arity":
		if isCallable(target) {
			return rt.rubyArityOf(target)
		}
	case "to_proc":
		// Method#to_proc answers a PROC, and a LAMBDA one: MRI's
		// `method(:foo).to_proc` is `#<Proc (lambda)>` with the Method's own
		// arity. A fresh wrapper rather than the Method itself, because the Method
		// is what carries the Method class.
		if _, isMeth := rt.rubyMethodValOf(target); isMeth {
			inner := target
			pw := interface{}(&hostFunc{name: "to_proc", fn: func(rt *jsrt, this uint64, a []interface{}) interface{} {
				return rt.call(inner, jsUndef, a)
			}})
			if ar, ok := rt.rubyArities[target]; ok {
				rt.rubyArities[pw] = ar
			}
			if rt.rubyLambdas == nil {
				rt.rubyLambdas = map[interface{}]bool{}
			}
			rt.rubyLambdas[pw] = true
			return pw
		}
		if isCallable(target) {
			return target
		}
	case "name":
		// Method#name: the Symbol `#<Method: Object#x>` ends in. #owner is NOT
		// implemented in any engine and that is deliberate - it has to answer the
		// class OBJECT, which the two compiled halves reach only through the
		// emitted module, so one of the three would have to answer a synthetic
		// stand-in. An omission is not a divergence.
		if mv, isMeth := rt.rubyMethodValOf(target); isMeth {
			return jsSym{s: mv.name}
		}
		if v, ok := rt.rubyNameErrSlot(target, name, "__nmname"); ok {
			return v
		}
	case "between?", "clamp":
		// Comparable#between? and #clamp are defined by <=> ALONE, so Integer,
		// Float, Rational, Symbol and String all answer them off ONE pair of arms.
		// rubySpaceship is the whole rule and it is big-aware. A user class that
		// defines either method itself still wins: its own dispatch is below.
		if !rubyUserObj(target) {
			lo, hi := argAt(args, 0), argAt(args, 1)
			cl, okLo := rt.rubySpaceship(target, lo)
			ch, okHi := rt.rubySpaceship(target, hi)
			if okLo && okHi {
				if name == "between?" {
					return cl >= 0 && ch <= 0
				}
				if cl < 0 {
					return lo
				}
				if ch > 0 {
					return hi
				}
				return target
			}
		}
	}
	// NilClass's conversions. MRI answers "" / "nil" / [] / 0 / 0.0 for these, so
	// they must not reach the "method .x on nil" abort at the bottom - and to_s in
	// particular must not fall through to the JS stringification, which said
	// "null" while ruby-interpreter.abnf said "".
	if isUndefOrNull(target) {
		switch name {
		case "to_s":
			return ""
		case "inspect":
			return "nil"
		case "to_a":
			return &jsArray{}
		case "to_i":
			return float64(0)
		case "to_f":
			return jsFlo{f: 0}
		}
	}
	if rubyIsNum(target) {
		if v, ok := rt.rubyNumMethod(target, name, args); ok {
			return v
		}
	}
	switch o := target.(type) {
	case jsSym:
		switch name {
		case "to_s", "id2name", "name":
			return o.s
		case "to_sym":
			return o
		case "length", "size":
			return float64(rubyCharCount(o.s))
		case "inspect":
			return ":" + o.s
		// Symbol's case and successor methods are the String ones on its name.
		case "upcase", "downcase", "swapcase", "capitalize", "succ", "next":
			return jsSym{s: rt.rubyStr(rt.rubyMethod(o.s, name, args))}
		case "empty?":
			return o.s == ""
		case "to_proc":
			return rt.rubySymProc(o.s)
		}
		rt.rubyNoMethod(target, name)
	case string:
		switch name {
		case "length", "size":
			// CHARACTERS, as js_pylen counts them for the lowered `.length`.
			return float64(rubyCharCount(o))
		case "to_s":
			return o
		case "inspect":
			// The QUOTED form.
			return rt.rubyInspect(o)
		case "to_sym", "intern":
			return jsSym{s: o}
		// String#ord: the code point of the FIRST character (`"\\e".ord`).
		case "ord":
			if r := []rune(o); len(r) > 0 {
				return float64(r[0])
			}
			return float64(0)
		// String#to_i / #to_f parse a LEADING number and answer 0 when there is
		// none.
		case "to_i":
			base := 10
			if len(args) > 0 {
				base = rt.rubyRadixArg(args[0], false)
			}
			n, _ := rubyStrToInum(o, base, false)
			return rt.rubyBigNarrow(n)
		case "to_f":
			if v := jsParseFloat(o); v == v {
				return jsFlo{f: v}
			}
			return jsFlo{f: 0}
		case "upcase":
			return caseMapMode(o, caseUpper|caseFull)
		case "downcase":
			return caseMapMode(o, caseFull)
		case "include?":
			return strings.Contains(o, rt.toString(argAt(args, 0)))
		// String#[] / #slice: one index or a Range (js_rget's own rule), or
		// start+len.
		case "[]", "slice":
			spos, _ := rubyBlockArgs(args)
			if len(spos) > 1 {
				return rt.rubyStrSubseq(o, spos[0], spos[1])
			}
			if r, isRng := rubyIsRange(argAt(spos, 0)); isRng {
				v, _ := rt.rubySubrange(o, r.props["begin"], r.props["end"], rt.truthy(r.props["excl"]))
				return v
			}
			n := rt.strLen(o)
			ix := int(rt.toNumber(argAt(spos, 0)))
			if ix < 0 {
				ix += n
			}
			if ix < 0 || ix >= n {
				return jsNull
			}
			return rt.strAt(o, ix)
		// String#start_with? / #end_with? take ANY NUMBER of prefixes/suffixes and
		// answer true when one of them matches.
		case "start_with?":
			pfx, _ := rubyBlockArgs(args)
			for _, a := range pfx {
				if strings.HasPrefix(o, rt.toString(a)) {
					return true
				}
			}
			return false
		case "end_with?":
			sfx, _ := rubyBlockArgs(args)
			for _, a := range sfx {
				if strings.HasSuffix(o, rt.toString(a)) {
					return true
				}
			}
			return false
		// Plain String methods, with ruby-interpreter.abnf's bodies. `split` keeps
		// that half's LITERAL separator (MRI's single space is the awk separator - a
		// run of whitespace with the leading run dropped); the two halves must agree.
		case "capitalize":
			if o == "" {
				return o
			}
			return caseMapMode(rt.strAt(o, 0), caseTitle|caseFull) + caseMapMode(rt.strRange(o, 1, rt.strLen(o)), caseFull)
		case "reverse":
			n := rt.strLen(o)
			out := ""
			for i := n - 1; i >= 0; i-- {
				out += rt.strAt(o, i)
			}
			return out
		case "strip":
			return strings.Trim(o, " \t\n\r\f\v\x00")
		case "chars":
			// One element per CHARACTER, so a surrogate pair stays whole.
			out := &jsArray{}
			for _, r := range o {
				out.elems = append(out.elems, string(r))
			}
			return out
		case "split":
			sep := " "
			if pos, _ := rubyBlockArgs(args); len(pos) > 0 {
				sep = rt.toString(pos[0])
			}
			out := &jsArray{}
			if sep == "" {
				for i := 0; i < rt.strLen(o); i++ {
					out.elems = append(out.elems, rt.strAt(o, i))
				}
				return out
			}
			for _, part := range strings.Split(o, sep) {
				out.elems = append(out.elems, part)
			}
			return out
		case "index":
			if i := rt.strIndexOf(o, rt.toString(argAt(args, 0))); i >= 0 {
				return float64(i)
			}
			return jsNull
		case "empty?":
			return o == ""
		case "succ", "next":
			return rubyStrSucc(o)
		case "replace":
			return rt.toString(argAt(args, 0))
		case "each_char":
			if _, blk := rubyBlockArgs(args); blk != nil {
				for i := 0; i < rt.strLen(o); i++ {
					rt.call(blk, jsUndef, []interface{}{rt.strAt(o, i)})
				}
			}
			return o
		case "dup", "clone", "freeze", "to_str", "+@", "-@", "itself":
			// A Ruby String is mutable, so `s = "a".dup` hands back a copy; the
			// compiler models a String as a value, which makes the copy the value.
			return o
		}
		// The rest of the String surface (bytes / tr / center / ljust / rjust /
		// count / delete and their families).
		spos, sblk := rubyBlockArgs(args)
		switch name {
		case "upto":
			lim := o
			if len(spos) > 0 {
				lim = rt.rubyStr(spos[0])
			}
			ul := rubySmUptoList(o, lim, len(spos) > 1 && rubyTruthy(spos[1]))
			if sblk == nil {
				return rubySmStrArr(ul)
			}
			for _, e := range ul {
				rt.call(sblk, jsUndef, []interface{}{e})
			}
			return o
		case "dump":
			return rubyStrInspect(o)
		// String#slice! answers the REMOVED part; the removal itself is
		// unrepresentable (a Ruby String is a value here), so the value is
		// String#slice's.
		case "slice!":
			return rt.rubyMethod(o, "slice", args)
		case "each_byte", "each_codepoint", "each_line":
			if sblk != nil {
				var list []interface{}
				if name == "each_byte" {
					list = rubySmNumArr([]byte(o)).elems
				} else if name == "each_codepoint" {
					for _, r := range o {
						list = append(list, float64(r))
					}
				} else {
					list = rubySmStrArr(rt.rubySmLines(o)).elems
				}
				for _, e := range list {
					rt.call(sblk, jsUndef, []interface{}{e})
				}
			}
			return o
		}
		sargs := make([]string, 0, len(spos))
		nargs := make([]float64, 0, len(spos))
		for _, a := range spos {
			sargs = append(sargs, rt.rubyStr(a))
			nargs = append(nargs, rubyToF(a))
		}
		if bb := rubySmBangBase(name); bb != "" {
			bv := rt.rubyMethod(o, bb, args)
			if bs, isStr := bv.(string); isStr && bs == o {
				return jsNull
			}
			return bv
		}
		if v, ok := rt.rubySmMethod(o, name, sargs, nargs); ok {
			return v
		}
		rt.rubyNoMethod(target, name)
	case *jsArray:
		return rt.rubyArrayMethod(o, name, args)
	case *jsObject:
		if rubyIsEnum(o) {
			return rt.rubyEnumMethod(o, name, args)
		}
		if _, isRng := o.props["__rrange"]; isRng {
			return rt.rubyRangeMethod(o, name, args)
		}
		if keys, vals, isDict := dictParts(o); isDict {
			switch name {
			case "inspect":
				return rt.rubyInspect(o)
			case "size", "length":
				return float64(len(keys.elems))
			case "keys":
				return &jsArray{elems: append([]interface{}{}, keys.elems...)}
			case "values":
				return &jsArray{elems: append([]interface{}{}, vals.elems...)}
			case "to_a", "entries":
				// Hash#to_a is the list of [key, value] PAIRS, as in MRI.
				out := &jsArray{}
				for i := range keys.elems {
					out.elems = append(out.elems, &jsArray{elems: []interface{}{keys.elems[i], vals.elems[i]}})
				}
				return out
			// Hash#[] as a METHOD (h.send(:[], k)); the subscript itself goes
			// through js_rget. String and Array answer it too.
			case "[]":
				if i := rt.rubyDictFind(keys, argAt(args, 0)); i >= 0 {
					return vals.elems[i]
				}
				return rt.rubyHashDefault(o, argAt(args, 0))
			case "include?", "has_key?", "key?":
				return rt.rubyDictFind(keys, argAt(args, 0)) >= 0
			case "each":
				_, eblk := rubyBlockArgs(args)
				for i := range keys.elems {
					rt.call(eblk, jsUndef, []interface{}{keys.elems[i], vals.elems[i]})
				}
				return o
			case "default":
				// Hash#default is the value Hash.new(v) recorded; a hash built
				// with a BLOCK answers nil here, exactly as MRI does (the block
				// is #default_proc).
				if d, has := o.props["__hdef"]; has {
					return d
				}
				return jsNull
			case "fetch":
				// Hash#fetch DELIBERATELY ignores the default: MRI raises
				// KeyError for a missing key unless a second argument or a
				// block supplies the answer.
				pos, blk := rubyBlockArgs(args)
				if i := rt.rubyDictFind(keys, argAt(pos, 0)); i >= 0 {
					return vals.elems[i]
				}
				if len(pos) > 1 {
					return pos[1]
				}
				if blk != nil {
					return rt.call(blk, jsUndef, []interface{}{argAt(pos, 0)})
				}
				rt.rubyRaiseErr("KeyError", "key not found: "+rt.rubyInspect(argAt(pos, 0)))
			}
			if v, handled := rt.rubyHashMethod(o, keys, vals, name, args); handled {
				return v
			}
			rt.rubyNoMethod(target, name)
		}
		// The SINGLETON table comes first: `def obj.m', `class << obj' and an
		// extended module all sit in front of the object's own class, for an
		// instance and for a class object alike.
		_, objIsCls := o.props["__isclass"]
		_, objHasCls := o.props["__class"]
		if objIsCls || objHasCls {
			if m, has := rubySingFind(o, name); has {
				return rt.call(m, jsUndef, append([]interface{}{o}, args...))
			}
			// NOTE: these four are answered AHEAD of a user class's own method
			// of the same name. MRI would let the user method win (Object#extend
			// and friends are ordinary, overridable methods); `method' below is
			// guarded because a user `method' is plausible. All three engines do
			// the same thing - a known divergence from MRI.
			switch name {
			case "extend":
				pos, _ := rubyBlockArgs(args)
				for _, m := range pos {
					rubySingExtend(o, m)
				}
				return o
			case "define_singleton_method":
				pos, blk := rubyBlockArgs(args)
				if len(pos) == 0 {
					rt.fail("no method name given")
				}
				body := blk
				if body == nil && len(pos) > 1 {
					body = pos[1]
				}
				nm := rt.rubyStr(pos[0])
				fn := body
				rubySingPut(o, nm, &hostFunc{name: "singleton:" + nm,
					fn: func(rt *jsrt, this uint64, a []interface{}) interface{} {
						// The receiver is prepended by the dispatch above, exactly
						// as it is for a compiled method; the block body does not
						// take it (it closes over its defining scope instead).
						if len(a) > 0 {
							a = a[1:]
						}
						return rt.call(fn, jsUndef, a)
					}})
				return jsSym{s: nm}
			case "singleton_methods":
				return rt.rubySingNames(o)
			case "respond_to?":
				return rt.rubyRespond(o, args)
			case "method":
				// Object#method with an explicit RECEIVER (Kernel#method with none
				// is js_rmethodval above). It is the one way a singleton method is
				// reachable as a value. A user class that defines `method' itself still
				// wins - the singleton table was searched first and the class chain is
				// asked here.
				if _, userHas := rubyFindMethod(o, "method"); !userHas {
					nm := rt.rubyStr(argAt(args, 0))
					if _, found := rubyFindMethod(o, nm); !found {
						rt.rubyRaiseErr("NameError", "undefined method `"+nm+"' for class `"+rubyOwnerName(o)+"'")
					}
					recv := interface{}(o)
					return rt.rubyMarkMethodVal(&hostFunc{name: "method:" + nm,
						fn: func(rt *jsrt, this uint64, a []interface{}) interface{} {
							return rt.rubyMethod(recv, nm, a)
						}}, rubyOwnerName(recv), nm)
				}
			}
		}
		// Ruby's type predicates. is_a?/kind_of? walk the __super chain of the
		// instance's class; instance_of? compares the exact class.
		switch name {
		case "class":
			if c, ok := o.props["__class"]; ok {
				return c
			}
		case "instance_of?":
			// A class OBJECT has no __class slot; the __isclass block below owns
			// `A.instance_of?(Class)'.
			if isCls, _ := o.props["__isclass"].(bool); !isCls {
				return o.props["__class"] == argAt(args, 0)
			}
		case "is_a?", "kind_of?":
			// rubyIsA, not a bare identity walk of __super. A builtin class object
			// carries no __super (the hierarchy is rubyExcParent, by NAME), so the
			// walk alone would answer false for `raise "x" rescue e; e.is_a?(Exception)'
			// and for every user exception subclass tested against Exception.
			return rt.rubyIsA(o, argAt(args, 0))
		}
		// A singleton (class) method: `def self.m` / `class << self` stores the
		// method on the class descriptor under a "$s$" prefixed key, so it cannot
		// collide with the INSTANCE method of the same name (Ruby keeps the two
		// apart, and only ruby-to-llvm-ir.abnf emits that key). The lookup walks
		// __super, like the instance one, and passes the class itself as self.
		if _, isCls := o.props["__isclass"]; isCls {
			for cls := interface{}(o); cls != nil; {
				clsObj, ok := cls.(*jsObject)
				if !ok {
					break
				}
				if m, ok := clsObj.props["$s$"+name]; ok && isCallable(m) {
					return rt.call(m, jsUndef, append([]interface{}{o}, args...))
				}
				cls = clsObj.props["__super"]
			}
			// Class#name, #to_s and #inspect are the class NAME.
			switch name {
			case "name", "to_s", "inspect":
				return rt.rubyClassText(o, name)
			}
			// The reflection surface. A user `def self.ancestors' still wins - the
			// $s$ walk above runs first.
			isMod, _ := o.props["__ismodule"].(bool)
			switch name {
			case "ancestors":
				return rt.rubyAncestors(o)
			case "include?":
				return rt.rubyAncestorHas(o, argAt(args, 0))
			case "class":
				if isMod {
					return rt.rubyBuiltinClass("Module")
				}
				return rt.rubyBuiltinClass("Class")
			case "instance_of?":
				a0, isObj0 := argAt(args, 0).(*jsObject)
				if !isObj0 {
					return false
				}
				bn, _ := a0.props["__rbuiltin"].(string)
				if isMod {
					return bn == "Module"
				}
				return bn == "Class"
			case "superclass":
				// Class#superclass skips the synthetic levels the lowering inserts.
				// A builtin exception class has no __super: its parent is named in
				// rubyExcParent. Any other root class answers Object.
				if isMod {
					return jsNull
				}
				sup := o.props["__super"]
				for guard := 0; guard < 64; guard++ {
					so, isO := sup.(*jsObject)
					if !isO {
						break
					}
					if synth, _ := so.props["__synth"].(bool); !synth {
						return so
					}
					sup = so.props["__super"]
				}
				nm, _ := o.props["__name"].(string)
				if _, isBuiltin := o.props["__rbuiltin"].(string); isBuiltin {
					if p := rubyExcParent[nm]; p != "" {
						return rt.rubyBuiltinClass(p)
					}
				}
				switch nm {
				case "BasicObject":
					return jsNull
				case "Object":
					return rt.rubyBuiltinClass("BasicObject")
				}
				return rt.rubyBuiltinClass("Object")
			}
		}
		// Kernel.format / Kernel.sprintf / Kernel.rand: the Kernel module's methods
		// are ordinarily called without a receiver, but naming Kernel explicitly is
		// legal Ruby.
		if bn, isBuiltin := o.props["__rbuiltin"].(string); isBuiltin && bn == "Kernel" {
			switch name {
			case "format", "sprintf":
				return rt.rubyFormatArgs(args)
			case "rand":
				return rubyRand(argAt(args, 0))
			}
		}
		// Math.sqrt / Math.sin / ... - see rubyMathCall. Ruby's Math is not the
		// ROOT scope's JavaScript Math: `Math.sqrt(4)' is 2.0 and `Math::PI' is
		// defined.
		if bn, isBuiltin := o.props["__rbuiltin"].(string); isBuiltin && bn == "Math" {
			if v, ok := rubyMathCall(name, rt, args); ok {
				return v
			}
		}
		// A class instance or class object: the generic dispatch handles it.
		return rt.rubyMemberCall(target, name, args)
	}
	if isUndefOrNull(target) {
		rt.rubyNoMethod(target, name)
	}
	return rt.rubyMemberCall(target, name, args)
}

// rubyRand is Kernel#rand. There is no random source in this project and there
// cannot be one the two engines agree on byte-for-byte, so rand is
// DETERMINISTICALLY ZERO - but zero of the right class, which is the part a
// program can test: `rand` and `rand(0)` are Floats in [0,1), `rand(n)` an Integer
// in [0,n), and `rand(a..b)` the range's own low end.
func rubyRand(v interface{}) interface{} {
	if isUndefOrNull(v) {
		return jsFlo{f: 0}
	}
	// A closed range reaches the compiler halves already materialized as an array.
	if a, isArr := v.(*jsArray); isArr {
		if len(a.elems) == 0 {
			return jsNull
		}
		return a.elems[0]
	}
	if o, isObj := v.(*jsObject); isObj {
		if _, isRng := o.props["__rrange"]; isRng {
			return o.props["begin"]
		}
	}
	if _, isFlo := v.(jsFlo); isFlo || rubyToF(v) == 0 {
		return jsFlo{f: 0}
	}
	return float64(0)
}

// rubySymProc is Symbol#to_proc (and `&:name`): the proc that calls the method
// `name` on its first argument with the rest. Twin of js_rsymproc in
// lib/ruby-rt.metajs.
func (rt *jsrt) rubySymProc(name string) *hostFunc {
	return &hostFunc{name: "&:" + name, fn: func(rt *jsrt, this uint64, args []interface{}) interface{} {
		rest := []interface{}{}
		if len(args) > 1 {
			rest = args[1:]
		}
		return rt.rubyMethod(argAt(args, 0), name, rest)
	}}
}

// rubyBlockArgs splits a Ruby argument list into its positional part and the
// trailing block. The two compiler halves pass a block as the LAST positional
// argument, WRAPPED in the block marker (see emitArgs in ruby-to-llvm-ir.abnf),
// where ruby-interpreter.abnf also keeps it in a parameter of its own - which is
// why arrayMethod there takes a fourth argument and this does not. The body is
// below, beside rubyBlkWrap.
//
// rubyArity is the parameter shape Proc#arity is computed from: how many
// parameters are required, how many have defaults, and whether there is a *splat.
// Keyword and block parameters are not counted at all.
type rubyArity struct {
	req  int
	opt  int
	star bool
}

// rubyArgCountErr is the ArgumentError MRI raises when a method or a lambda is
// called with too few positional arguments, with MRI's own wording and its three
// shapes for the expectation: `expected 1` for a fixed list, `expected 1..2` when
// optionals follow, `expected 1+` when a *splat does. `kinds` is js_rposbind's
// per-parameter code string ("0" required, "1" optional, "2" splat). Mirrors
// argCountErr of ruby-interpreter.abnf and rbArgCountErr of lib/ruby-rt.metajs.
// rubyLamArity is Kernel#lambda's count check: a trailing block marker is not a
// positional argument, a *splat lifts the ceiling, and the message is MRI's.
// Mirrors lamArity of ruby-interpreter.abnf and rbLamArity of lib/ruby-rt.metajs.
func (rt *jsrt) rubyLamArity(args []interface{}, kinds string) {
	npos := len(args)
	if npos > 0 && rubyIsBlkArg(args[npos-1]) {
		npos--
	}
	req := 0
	star := false
	for i := 0; i < len(kinds); i++ {
		switch kinds[i] {
		case '2':
			star = true
		case '1':
		default:
			req++
		}
	}
	if npos < req {
		rt.rubyArgCountErr(npos, kinds)
	}
	if !star && npos > len(kinds) {
		rt.rubyArgCountErr(npos, kinds)
	}
}

func (rt *jsrt) rubyArgCountErr(given int, kinds string) {
	req, opt, star := 0, 0, false
	for i := 0; i < len(kinds); i++ {
		switch kinds[i] {
		case '2':
			star = true
		case '1':
			opt++
		default:
			req++
		}
	}
	want := strconv.Itoa(req)
	if star {
		want += "+"
	} else if opt > 0 {
		want += ".." + strconv.Itoa(req+opt)
	}
	panic(&jsThrown{value: rt.rubyExc(rt.rubyBuiltinClass("ArgumentError"),
		[]interface{}{"wrong number of arguments (given " + strconv.Itoa(given) +
			", expected " + want + ")"})})
}

// rubyArityOf is Proc#arity. The SHAPE is stored rather than the answer because
// Kernel#lambda turns an already-built proc into a lambda, and the two count
// optional parameters differently: a proc ignores them (`proc { |x = 0| }.arity`
// is 0), a lambda does not (`lambda { |x = 0| }.arity` is -1). A *splat makes it
// -(required + 1) either way. Mirrors procArity of ruby-interpreter.abnf and
// rbProcArity of lib/ruby-rt.metajs.
func (rt *jsrt) rubyArityOf(f interface{}) float64 {
	a, ok := rt.rubyArities[f]
	if !ok {
		return 0
	}
	// A METHOD counts optionals the way a lambda does, not the way a proc does:
	// MRI's `method(:m).arity` for `def m(a, b = 1)` is -2, not 1.
	_, isMeth := rt.rubyMethodValOf(f)
	if a.star || ((rt.rubyLambdas[f] || isMeth) && a.opt > 0) {
		return float64(-(a.req + 1))
	}
	return float64(a.req)
}

// THE BLOCK MARKER. A Ruby block travels as the last element of the argument
// array, wrapped in a one-slot object that nothing else can produce, so "the
// trailing argument is a block" is a fact about the CALL SITE rather than a
// guess about the value: a Proc handed over as an ordinary positional argument
// (`def m(f = 3); end; m(some_proc)`) is not a block. The wrapper is built by
// js_rblkwrap (emitArgs of ruby-to-llvm-ir.abnf), and rbBlkOf of
// lib/ruby-rt.metajs is the same pair.
func rubyBlkWrap(v interface{}) interface{} {
	o := newJSObject()
	o.set("__rblk", v)
	return o
}

// rubyBlkOf answers the block a wrapper carries, and whether v is one at all.
func rubyBlkOf(v interface{}) (interface{}, bool) {
	if o, isObj := v.(*jsObject); isObj {
		if f, has := o.props["__rblk"]; has {
			return f, true
		}
	}
	return nil, false
}

// rubyIsBlkArg reports whether the last element of a positional list is the
// call's block rather than a value.
func rubyIsBlkArg(v interface{}) bool {
	_, is := rubyBlkOf(v)
	return is
}

// rubyIsBinOpName is the set of operator METHOD names Object#send may be handed.
// == and != are left out on purpose: every dispatcher answers those for every
// value already, and routing them through rubyBin would send a user object's
// `==` back through the method table it just came out of.
func rubyIsBinOpName(n string) bool {
	switch n {
	case "+", "-", "*", "/", "%", "**", "<<", ">>", "&", "|", "^", "<", ">", "<=", ">=", "<=>":
		return true
	}
	return false
}

func rubyBlockArgs(args []interface{}) ([]interface{}, interface{}) {
	if n := len(args); n > 0 {
		if f, is := rubyBlkOf(args[n-1]); is {
			return args[:n-1], f
		}
	}
	return args, nil
}

// rubyArrSort is a STABLE merge sort. MRI's Array#sort is not stable and its
// #sort_by is, so one stable sort satisfies both contracts. Mirrors arrSortWith of
// ruby-interpreter.abnf and rbArrSortWith of lib/ruby-rt.metajs.
func rubyArrSort(a []interface{}, cmp func(x, y interface{}) int) []interface{} {
	if len(a) < 2 {
		return append([]interface{}{}, a...)
	}
	mid := len(a) / 2
	lhs := rubyArrSort(a[:mid], cmp)
	rhs := rubyArrSort(a[mid:], cmp)
	out := make([]interface{}, 0, len(a))
	li, ri := 0, 0
	for li < len(lhs) && ri < len(rhs) {
		// Left-first on a tie keeps equal elements in their original order. The
		// operands are passed in THIS order (left, right) so that the
		// ArgumentError an incomparable pair raises names them the way MRI's own
		// sort does.
		if cmp(lhs[li], rhs[ri]) <= 0 {
			out = append(out, lhs[li])
			li++
		} else {
			out = append(out, rhs[ri])
			ri++
		}
	}
	out = append(out, lhs[li:]...)
	return append(out, rhs[ri:]...)
}

// rubyArrCmpFn is the comparator a sort / min / max uses: the block when one was
// given (whose value is an Integer, like <=>), rcmp otherwise - which raises on
// values that cannot be compared, as MRI does.
func (rt *jsrt) rubyArrCmpFn(blk interface{}) func(x, y interface{}) int {
	if blk == nil {
		return func(x, y interface{}) int { return rt.rubyCmp(x, y) }
	}
	return func(x, y interface{}) int {
		return int(math.Trunc(rubyToF(rt.call(blk, jsUndef, []interface{}{x, y}))))
	}
}

// rubyArrKeyed is the decorate-sort step of sort_by / min_by / max_by: each
// element paired with its block value, stably sorted on that key.
func (rt *jsrt) rubyArrKeyed(t *jsArray, blk interface{}) []interface{} {
	pairs := make([]interface{}, 0, len(t.elems))
	for _, e := range t.elems {
		pairs = append(pairs, &jsArray{elems: []interface{}{rt.call(blk, jsUndef, []interface{}{e}), e}})
	}
	return rubyArrSort(pairs, func(x, y interface{}) int {
		return rt.rubyCmp(x.(*jsArray).elems[0], y.(*jsArray).elems[0])
	})
}

func rubyUndecorate(pairs []interface{}) *jsArray {
	out := &jsArray{}
	for _, p := range pairs {
		out.elems = append(out.elems, p.(*jsArray).elems[1])
	}
	return out
}

// rubyArrJoin RECURSES into a nested array with the same separator, and sends
// every other element through to_s (so nil contributes "").
func (rt *jsrt) rubyArrJoin(t *jsArray, sep string) string {
	out := ""
	for i, e := range t.elems {
		if i > 0 {
			out += sep
		}
		if sub, isArr := e.(*jsArray); isArr {
			out += rt.rubyArrJoin(sub, sep)
		} else {
			out += rt.rubyStr(e)
		}
	}
	return out
}

// rubyArrFlat splices nested arrays to a DEPTH: flatten(1) one level, -1 all the
// way down (which is what a missing argument means).
func rubyArrFlat(t *jsArray, depth int, out *jsArray) *jsArray {
	for _, e := range t.elems {
		if sub, isArr := e.(*jsArray); isArr && depth != 0 {
			rubyArrFlat(sub, depth-1, out)
		} else {
			out.elems = append(out.elems, e)
		}
	}
	return out
}

// rubyArrReplace is how the bang methods mutate in place. It copies first, because
// the source is usually the receiver itself.
func rubyArrReplace(t *jsArray, src []interface{}) *jsArray {
	keep := append([]interface{}{}, src...)
	t.dropIdx()
	t.elems = keep
	return t
}

// rubyArrNorm resolves a possibly negative index, exactly as Array#[] does.
func rubyArrNorm(t *jsArray, v interface{}) int {
	n := int(math.Trunc(rubyToF(v)))
	if n < 0 {
		return len(t.elems) + n
	}
	return n
}

func rubyArrSub(t *jsArray, from, length int) *jsArray {
	out := &jsArray{}
	for i := from; i < len(t.elems) && len(out.elems) < length; i++ {
		if i >= 0 {
			out.elems = append(out.elems, t.elems[i])
		}
	}
	return out
}

// Ruby's SUBSEQUENCE rule for Array#[start, len] and String#[start, len]: nil for
// a negative length or a start past the end, an EMPTY result for a start exactly at
// the end, and the length clamped to what is left. A negative start counts back
// from the end. Mirrors rsubseq of ruby-interpreter.abnf and
// rbArrSubseq/rbStrSubseq of lib/ruby-rt.metajs.
func rubyArrSubseq(t *jsArray, start, length interface{}) interface{} {
	s := int(math.Trunc(rubyToF(start)))
	k := int(math.Trunc(rubyToF(length)))
	if k < 0 {
		return jsNull
	}
	if s < 0 {
		s += len(t.elems)
	}
	if s < 0 || s > len(t.elems) {
		return jsNull
	}
	return rubyArrSub(t, s, k)
}

func (rt *jsrt) rubyStrSubseq(t string, start, length interface{}) interface{} {
	n := rt.strLen(t)
	s := int(math.Trunc(rubyToF(start)))
	k := int(math.Trunc(rubyToF(length)))
	if k < 0 {
		return jsNull
	}
	if s < 0 {
		s += n
	}
	if s < 0 || s > n {
		return jsNull
	}
	if s+k > n {
		k = n - s
	}
	return rt.strRange(t, s, s+k)
}

// rubyArrHas / rubyArrAnd / rubyArrOr back Array#& and Array#|. Both keep the
// LEFT-hand order, both dedupe, and both compare with eql? (the hash-key
// comparison), not == - like Array#- and Array#uniq and unlike #include?.
func (rt *jsrt) rubyArrHas(a []interface{}, v interface{}) bool {
	for _, e := range a {
		if rt.rubyEql(e, v) {
			return true
		}
	}
	return false
}

func (rt *jsrt) rubyArrAnd(l, r *jsArray) *jsArray {
	out := &jsArray{}
	for _, e := range l.elems {
		if rt.rubyArrHas(r.elems, e) && !rt.rubyArrHas(out.elems, e) {
			out.elems = append(out.elems, e)
		}
	}
	return out
}

func (rt *jsrt) rubyArrOr(l, r *jsArray) *jsArray {
	out := &jsArray{}
	for _, e := range append(append([]interface{}{}, l.elems...), r.elems...) {
		if !rt.rubyArrHas(out.elems, e) {
			out.elems = append(out.elems, e)
		}
	}
	return out
}

// rubyFormatArgs is Kernel#format / #sprintf on a whole argument list
// [format, args...] - shared by the js_rformat extern and the Kernel.format form.
func (rt *jsrt) rubyFormatArgs(args []interface{}) string {
	if len(args) == 0 {
		return ""
	}
	return rt.rubyFormat(rt.rubyStr(args[0]), &jsArray{elems: args[1:]})
}

// rubyPatternHit is the `pattern === value` test that all?/any?/none?/one? apply
// to a non-block argument: a class object matches by is_a?, everything else by ==.
// It is the rule js_rwhen applies to a `when V` label.
func (rt *jsrt) rubyPatternHit(pat, v interface{}) bool {
	if o, isObj := pat.(*jsObject); isObj {
		if _, isCls := o.props["__isclass"]; isCls {
			return rt.rubyIsA(v, pat)
		}
	}
	return rt.rubyEq(pat, v)
}

// rubyNewDict builds an empty Hash in the shared {__dict, keys, vals} shape that
// group_by and tally answer with.
func rubyNewDict() *jsObject {
	d := newJSObject()
	d.set("__dict", true)
	d.set("keys", &jsArray{})
	d.set("vals", &jsArray{})
	return d
}

// rubyNewHash is Hash.new / Hash.new(v) / Hash.new { |h, k| }. The default value
// or default block lives in a hidden __-prefixed slot, which keysOf skips, so it
// reaches neither keys nor inspect nor ==.
func rubyNewHash(args []interface{}) *jsObject {
	d := rubyNewDict()
	pos, blk := rubyBlockArgs(args)
	if blk != nil {
		d.set("__hdefp", blk)
		return d
	}
	if len(pos) > 0 {
		d.set("__hdef", pos[0])
	}
	return d
}

// rubyHashDefault is the value a Hash answers for a key it does not hold: the
// default block called with (hash, key) - which may WRITE the hash, the
// memoisation idiom - else the default value, else nil.
func (rt *jsrt) rubyHashDefault(o, k interface{}) interface{} {
	obj, ok := o.(*jsObject)
	if !ok {
		return jsNull
	}
	if p, has := obj.props["__hdefp"]; has && isCallable(p) {
		return rt.call(p, jsUndef, []interface{}{o, k})
	}
	if d, has := obj.props["__hdef"]; has {
		return d
	}
	return jsNull
}

// rubyArrNew is Array.new(n) / Array.new(n, v) / Array.new(n) { |i| }.
func (rt *jsrt) rubyArrNew(args []interface{}) *jsArray {
	pos, blk := rubyBlockArgs(args)
	out := &jsArray{}
	if len(pos) > 0 {
		if src, isArr := pos[0].(*jsArray); isArr {
			return &jsArray{elems: append([]interface{}{}, src.elems...)}
		}
	}
	n := 0
	if len(pos) > 0 {
		n = int(math.Trunc(rubyToF(pos[0])))
	}
	for i := 0; i < n; i++ {
		switch {
		case blk != nil:
			out.elems = append(out.elems, rt.call(blk, jsUndef, []interface{}{float64(i)}))
		case len(pos) > 1:
			out.elems = append(out.elems, pos[1])
		default:
			out.elems = append(out.elems, jsNull)
		}
	}
	return out
}

// rubyHashPairs is the [key, value] pair list a Hash yields to every Enumerable
// method. Mirrors hashPairs of ruby-interpreter.abnf and rbHashPairs of
// lib/ruby-rt.metajs.
func rubyHashPairs(keys, vals *jsArray) *jsArray {
	out := &jsArray{}
	for i := range keys.elems {
		out.elems = append(out.elems, &jsArray{elems: []interface{}{keys.elems[i], vals.elems[i]}})
	}
	return out
}

// rubyHashCopy is a shallow copy that KEEPS THE DEFAULT: MRI's Hash#dup, #clone
// and #merge all carry the receiver's default value / default_proc into the new
// Hash, so `Hash.new(0).merge(other)["missing"]` still answers 0.
func rubyHashCopy(src *jsObject) *jsObject {
	keys, vals, _ := dictParts(src)
	out := &jsObject{props: map[string]interface{}{
		"__dict": true,
		"keys":   &jsArray{elems: append([]interface{}{}, keys.elems...)},
		"vals":   &jsArray{elems: append([]interface{}{}, vals.elems...)},
	}}
	if d, has := src.props["__hdef"]; has {
		out.set("__hdef", d)
	}
	if d, has := src.props["__hdefp"]; has {
		out.set("__hdefp", d)
	}
	return out
}

// rubyPairsToHash rebuilds a Hash out of a pair list, carrying src's default.
func (rt *jsrt) rubyPairsToHash(pairs *jsArray, src *jsObject) *jsObject {
	out := rubyHashCopy(src)
	okeys, ovals, _ := dictParts(out)
	okeys.elems = nil
	ovals.elems = nil
	for _, p := range pairs.elems {
		row, isArr := p.(*jsArray)
		if !isArr || len(row.elems) < 2 {
			continue
		}
		if i := rt.rubyDictFind(okeys, row.elems[0]); i >= 0 {
			ovals.elems[i] = row.elems[1]
			continue
		}
		okeys.elems = append(okeys.elems, row.elems[0])
		ovals.elems = append(ovals.elems, row.elems[1])
	}
	return out
}

// rubyHashEnumName lists the Enumerable names a Hash answers over its PAIRS. It is
// explicit rather than a fallthrough so that a name Hash does not have (push,
// rotate, ...) is still reported instead of silently answered over the pairs.
// Mirrors hashEnumName of ruby-interpreter.abnf and rbHashEnumName of
// lib/ruby-rt.metajs.
func rubyHashEnumName(n string) bool {
	switch n {
	case "map", "collect", "flat_map", "collect_concat", "select", "filter",
		"reject", "select!", "filter!", "reject!", "keep_if", "delete_if",
		"find", "detect", "find_index", "sort", "sort_by", "min", "max",
		"min_by", "max_by", "minmax", "sum", "count", "group_by", "partition",
		"each_with_index", "each_with_object", "inject", "reduce", "all?",
		"any?", "none?", "one?", "first", "take", "drop", "take_while",
		"drop_while", "each_slice", "each_cons", "empty?", "tally", "zip",
		"assoc", "uniq", "reverse", "freeze", "itself":
		return true
	}
	return false
}

// rubyHashMethod is the Hash-specific and Enumerable half of Hash's surface, in
// one place for both compiled halves. It answers handled=false for a name Hash
// does not have, so the caller still reports it. Mirrors the Hash arm of mcall in
// ruby-interpreter.abnf and rbHashMethod of lib/ruby-rt.metajs.
func (rt *jsrt) rubyHashMethod(o *jsObject, keys, vals *jsArray, name string, args []interface{}) (interface{}, bool) {
	pos, blk := rubyBlockArgs(args)
	switch name {
	case "merge", "merge!", "update":
		mg := o
		if name == "merge" {
			mg = rubyHashCopy(o)
		}
		mkeys, mvals, _ := dictParts(mg)
		for _, src := range pos {
			skeys, svals, isDict := dictParts(src)
			if !isDict {
				continue
			}
			for i := range skeys.elems {
				if j := rt.rubyDictFind(mkeys, skeys.elems[i]); j >= 0 {
					if blk != nil {
						mvals.elems[j] = rt.call(blk, jsUndef,
							[]interface{}{skeys.elems[i], mvals.elems[j], svals.elems[i]})
					} else {
						mvals.elems[j] = svals.elems[i]
					}
					continue
				}
				mkeys.elems = append(mkeys.elems, skeys.elems[i])
				mvals.elems = append(mvals.elems, svals.elems[i])
			}
		}
		mkeys.dropIdx()
		return mg, true
	case "dup", "clone":
		return rubyHashCopy(o), true
	case "to_h":
		if blk == nil {
			return o, true
		}
	case "delete":
		i := rt.rubyDictFind(keys, argAt(pos, 0))
		if i < 0 {
			if blk != nil {
				return rt.call(blk, jsUndef, []interface{}{argAt(pos, 0)}), true
			}
			return jsNull, true
		}
		v := vals.elems[i]
		keys.elems = append(keys.elems[:i], keys.elems[i+1:]...)
		vals.elems = append(vals.elems[:i], vals.elems[i+1:]...)
		keys.dropIdx()
		return v, true
	case "clear":
		keys.elems = nil
		vals.elems = nil
		keys.dropIdx()
		return o, true
	case "each_pair":
		if blk == nil {
			return rubyMkEnum(rubyHashPairs(keys, vals)), true
		}
		for i := range keys.elems {
			rt.call(blk, jsUndef, []interface{}{keys.elems[i], vals.elems[i]})
		}
		return o, true
	case "each_key":
		for _, k := range append([]interface{}{}, keys.elems...) {
			rt.call(blk, jsUndef, []interface{}{k})
		}
		return o, true
	case "each_value":
		for _, v := range append([]interface{}{}, vals.elems...) {
			rt.call(blk, jsUndef, []interface{}{v})
		}
		return o, true
	case "values_at":
		// Hash#values_at takes KEYS, where Array#values_at takes indices, so it
		// must not reach the pair delegation below.
		out := &jsArray{}
		for _, k := range pos {
			if i := rt.rubyDictFind(keys, k); i >= 0 {
				out.elems = append(out.elems, vals.elems[i])
			} else {
				out.elems = append(out.elems, rt.rubyHashDefault(o, k))
			}
		}
		return out, true
	case "invert":
		out := rt.rubyPairsToHash(&jsArray{}, o)
		okeys, ovals, _ := dictParts(out)
		delete(out.props, "__hdef")
		delete(out.props, "__hdefp")
		for i := range keys.elems {
			if j := rt.rubyDictFind(okeys, vals.elems[i]); j >= 0 {
				ovals.elems[j] = keys.elems[i]
				continue
			}
			okeys.elems = append(okeys.elems, vals.elems[i])
			ovals.elems = append(ovals.elems, keys.elems[i])
		}
		return out, true
	case "key":
		for i := range vals.elems {
			if rt.rubyEq(vals.elems[i], argAt(pos, 0)) {
				return keys.elems[i], true
			}
		}
		return jsNull, true
	case "transform_values":
		out := rubyHashCopy(o)
		_, ovals, _ := dictParts(out)
		for i := range ovals.elems {
			ovals.elems[i] = rt.call(blk, jsUndef, []interface{}{ovals.elems[i]})
		}
		return out, true
	case "transform_keys":
		out := rt.rubyPairsToHash(&jsArray{}, o)
		delete(out.props, "__hdef")
		delete(out.props, "__hdefp")
		okeys, ovals, _ := dictParts(out)
		for i := range keys.elems {
			nk := rt.call(blk, jsUndef, []interface{}{keys.elems[i]})
			if j := rt.rubyDictFind(okeys, nk); j >= 0 {
				ovals.elems[j] = vals.elems[i]
				continue
			}
			okeys.elems = append(okeys.elems, nk)
			ovals.elems = append(ovals.elems, vals.elems[i])
		}
		return out, true
	}
	if !rubyHashEnumName(name) {
		return nil, false
	}
	// Everything else Enumerable is answered over the [key, value] PAIRS, which is
	// what MRI's Hash does (Hash includes Enumerable and yields a pair). select /
	// filter / reject answer a HASH again, everything else the Array that
	// Enumerable builds.
	hp := rubyHashPairs(keys, vals)
	switch name {
	case "select", "filter", "reject", "select!", "filter!", "reject!", "keep_if", "delete_if":
		if blk == nil {
			return rubyMkEnumOp(hp, name), true
		}
		inner := "select"
		if name == "reject" || name == "reject!" || name == "delete_if" {
			inner = "reject"
		}
		kept, _ := rt.rubyArrayMethod(hp, inner, args).(*jsArray)
		if kept == nil {
			kept = &jsArray{}
		}
		hs := rt.rubyPairsToHash(kept, o)
		if name == "select" || name == "filter" || name == "reject" {
			return hs, true
		}
		hkeys, hvals, _ := dictParts(hs)
		keys.elems = append([]interface{}{}, hkeys.elems...)
		vals.elems = append([]interface{}{}, hvals.elems...)
		keys.dropIdx()
		return o, true
	}
	return rt.rubyDelegArr(hp, o, name, args), true
}

// rubyArrayMethod mirrors the arrayMethod of ruby-interpreter.abnf. select and
// reject use rubyTruthy (Ruby semantics), and pop/first/last return nil (not an
// error) on an empty array.
// rubyCombIdx / rubyPermIdx are combination / permutation as INDEX lists, so one
// pair of helpers serves both the Array methods and (through them) an Enumerator.
// MRI's orders: combination is index-ascending, permutation is the depth-first walk
// that takes each unused index in turn. Mirrors combIdx / permIdx of
// ruby-interpreter.abnf and rbCombIdx / rbPermIdx of lib/ruby-rt.metajs.
func rubyCombIdx(n, k int) [][]int {
	out := [][]int{}
	if k < 0 || k > n {
		return out
	}
	var rec func(start int, cur []int)
	rec = func(start int, cur []int) {
		if len(cur) == k {
			out = append(out, append([]int{}, cur...))
			return
		}
		for i := start; i < n; i++ {
			rec(i+1, append(cur, i))
		}
	}
	rec(0, []int{})
	return out
}

func rubyPermIdx(n, k int) [][]int {
	out := [][]int{}
	if k < 0 || k > n {
		return out
	}
	used := make([]bool, n)
	var rec func(cur []int)
	rec = func(cur []int) {
		if len(cur) == k {
			out = append(out, append([]int{}, cur...))
			return
		}
		for i := 0; i < n; i++ {
			if used[i] {
				continue
			}
			used[i] = true
			rec(append(cur, i))
			used[i] = false
		}
	}
	rec([]int{})
	return out
}

// rubyPick turns an index list into the element rows it selects.
func rubyPick(t *jsArray, picks [][]int) *jsArray {
	out := &jsArray{}
	for _, p := range picks {
		row := &jsArray{}
		for _, i := range p {
			row.elems = append(row.elems, t.elems[i])
		}
		out.elems = append(out.elems, row)
	}
	return out
}

// rubyMkEnum builds an EAGER Enumerator: the materialized element list, a cursor
// for #next, and the whole Enumerable surface by delegation to Array. A lazy one
// needs a suspendable body and ruby-to-llvm-ir.abnf lowers every block by hand
// (grep js_genfn: zero hits), so there is no coroutine to park in. Mirrors mkEnum /
// enumMethod of ruby-interpreter.abnf and rbMkEnum / rbEnumMethod of
// lib/ruby-rt.metajs.
func rubyMkEnum(items *jsArray) *jsObject {
	return rubyMkEnumOp(items, "each")
}

// An Enumerator remembers the OPERATION that made it: `h.map` is an Enumerator
// whose pending op is map, so `h.map.with_index { }` has to answer the MAPPED
// list, not the receiver.
func rubyMkEnumOp(items *jsArray, op string) *jsObject {
	e := newJSObject()
	e.set("__renum", true)
	e.set("items", items)
	e.set("pos", float64(0))
	e.set("op", op)
	return e
}

// rubyEnumFinish is what one pass of the pending operation answers, given the
// block results. Mirrors enumFinish of ruby-interpreter.abnf and rbEnumFinish of
// lib/ruby-rt.metajs.
func rubyEnumFinish(t *jsObject, items *jsArray, results []interface{}) interface{} {
	op, _ := t.props["op"].(string)
	switch op {
	case "map", "collect":
		return &jsArray{elems: results}
	case "flat_map", "collect_concat":
		out := &jsArray{}
		for _, r := range results {
			if row, isArr := r.(*jsArray); isArr {
				out.elems = append(out.elems, row.elems...)
				continue
			}
			out.elems = append(out.elems, r)
		}
		return out
	case "select", "filter", "reject":
		out := &jsArray{}
		for i := range items.elems {
			hit := rubyTruthy(results[i])
			if op == "reject" {
				hit = !hit
			}
			if hit {
				out.elems = append(out.elems, items.elems[i])
			}
		}
		return out
	}
	return t
}

func rubyIsEnum(v interface{}) bool {
	o, ok := v.(*jsObject)
	if !ok {
		return false
	}
	b, has := o.props["__renum"]
	return has && b == true
}

func (rt *jsrt) rubyEnumMethod(t *jsObject, name string, args []interface{}) interface{} {
	items, _ := t.props["items"].(*jsArray)
	if items == nil {
		items = &jsArray{}
	}
	pos, blk := rubyBlockArgs(args)
	switch name {
	case "to_a", "entries", "force":
		return &jsArray{elems: append([]interface{}{}, items.elems...)}
	case "size", "length", "count":
		if len(pos) == 0 && blk == nil {
			return float64(len(items.elems))
		}
	case "next", "peek":
		cur := int(rubyToF(t.props["pos"]))
		if cur >= len(items.elems) {
			rt.fail("iteration reached an end")
		}
		if name == "next" {
			t.set("pos", float64(cur+1))
		}
		return items.elems[cur]
	case "rewind":
		t.set("pos", float64(0))
		return t
	case "class":
		return rt.rubyBuiltinClass("Enumerator")
	case "inspect", "to_s":
		return "#<Enumerator: " + rt.rubyInspect(items) + ">"
	case "each":
		if blk == nil {
			return t
		}
		res := []interface{}{}
		for _, e := range items.elems {
			res = append(res, rt.call(blk, jsUndef, []interface{}{e}))
		}
		return rubyEnumFinish(t, items, res)
	case "with_index", "each_with_index":
		off := 0
		if name == "with_index" && len(pos) > 0 {
			off = int(math.Trunc(rubyToF(pos[0])))
		}
		if blk == nil {
			pairs := &jsArray{}
			for i, e := range items.elems {
				pairs.elems = append(pairs.elems, &jsArray{elems: []interface{}{e, float64(i + off)}})
			}
			return rubyMkEnum(pairs)
		}
		wres := []interface{}{}
		for i, e := range items.elems {
			wres = append(wres, rt.call(blk, jsUndef, []interface{}{e, float64(i + off)}))
		}
		return rubyEnumFinish(t, items, wres)
	case "with_object", "each_with_object":
		memo := argAt(pos, 0)
		for _, e := range items.elems {
			rt.call(blk, jsUndef, []interface{}{e, memo})
		}
		return memo
	}
	// Everything else is Enumerable, and an eager Enumerator IS its element list.
	return rt.rubyDelegArr(items, t, name, args)
}

// rubyArrWantsBlock lists the Array/Enumerable names that REQUIRE a block; called
// without one MRI hands back an Enumerator over the receiver rather than raising.
// Names that are legal blockless (sort, min, max, count, sum, all?, any?, none?,
// one?, find_index, include?) are not here, and each / each_with_index answer
// their own Enumerator below. Mirrors arrWantsBlock of ruby-interpreter.abnf and
// rbArrWantsBlock of lib/ruby-rt.metajs.
func rubyArrWantsBlock(n string) bool {
	switch n {
	case "map", "collect", "map!", "collect!", "select", "filter", "select!",
		"filter!", "reject", "reject!", "keep_if", "delete_if", "sort_by",
		"sort_by!", "group_by", "partition", "flat_map", "collect_concat",
		"find", "detect", "min_by", "max_by", "take_while", "drop_while":
		return true
	}
	return false
}

func (rt *jsrt) rubyArrayMethod(t *jsArray, name string, args []interface{}) interface{} {
	pos, blk := rubyBlockArgs(args)
	if blk == nil && rubyArrWantsBlock(name) {
		return rubyMkEnumOp(&jsArray{elems: append([]interface{}{}, t.elems...)}, name)
	}
	switch name {
	case "size", "length":
		return float64(len(t.elems))
	// Array#[]: one index or a Range (js_rget's own rule), or start+len.
	case "[]":
		if len(pos) > 1 {
			return rubyArrSubseq(t, pos[0], pos[1])
		}
		if r, isRng := rubyIsRange(argAt(pos, 0)); isRng {
			v, _ := rt.rubySubrange(t, r.props["begin"], r.props["end"], rt.truthy(r.props["excl"]))
			return v
		}
		ix := int(rt.toNumber(argAt(pos, 0)))
		if ix < 0 {
			ix += len(t.elems)
		}
		if ix < 0 || ix >= len(t.elems) {
			return jsNull
		}
		return t.elems[ix]
	case "push", "append", "add":
		// Ruby's Array#push is VARIADIC (`a.push(1, 2, 3)`); taking only args[0]
		// dropped every further element silently. `a.push` with no argument is a
		// no-op in Ruby, so an empty args list must append nothing.
		t.dropIdx()
		t.elems = append(t.elems, args...)
		return t
	// first / last / pop / shift all take an optional COUNT, and then answer an
	// ARRAY: `[1,2,3].first(2)` is [1, 2].
	case "pop":
		if len(pos) > 0 {
			n := int(math.Trunc(rubyToF(pos[0])))
			if n > len(t.elems) {
				n = len(t.elems)
			}
			tail := rubyArrSub(t, len(t.elems)-n, n)
			rubyArrReplace(t, rubyArrSub(t, 0, len(t.elems)-n).elems)
			return tail
		}
		if len(t.elems) == 0 {
			return jsNull
		}
		v := t.elems[len(t.elems)-1]
		t.dropIdx()
		t.elems = t.elems[:len(t.elems)-1]
		return v
	case "first":
		if len(pos) > 0 {
			return rubyArrSub(t, 0, int(math.Trunc(rubyToF(pos[0]))))
		}
		if len(t.elems) == 0 {
			return jsNull
		}
		return t.elems[0]
	case "last":
		if len(pos) > 0 {
			n := int(math.Trunc(rubyToF(pos[0])))
			if n > len(t.elems) {
				n = len(t.elems)
			}
			return rubyArrSub(t, len(t.elems)-n, n)
		}
		if len(t.elems) == 0 {
			return jsNull
		}
		return t.elems[len(t.elems)-1]
	// shift removes from the FRONT; unshift/prepend adds there, variadically.
	case "shift":
		if len(pos) > 0 {
			n := int(math.Trunc(rubyToF(pos[0])))
			if n > len(t.elems) {
				n = len(t.elems)
			}
			head := rubyArrSub(t, 0, n)
			rubyArrReplace(t, rubyArrSub(t, n, len(t.elems)-n).elems)
			return head
		}
		if len(t.elems) == 0 {
			return jsNull
		}
		v := t.elems[0]
		rubyArrReplace(t, rubyArrSub(t, 1, len(t.elems)-1).elems)
		return v
	case "unshift", "prepend":
		return rubyArrReplace(t, append(append([]interface{}{}, pos...), t.elems...))
	case "include?", "contains":
		// Array#include? is ==, not === and not eql?: `[[1]].include?([1])` and
		// `[1].include?(1.0)` are both true in MRI. A plain scan and not the dict
		// index, because an ordinary data array must not grow a key index just to
		// be searched.
		return rt.rubyArrIndex(t, argAt(pos, 0)) >= 0
	case "inspect":
		return rt.rubyInspect(t)
	case "index", "find_index":
		// == as well, and the same scan: `[[1]].index([1])` is 0 in MRI.
		if blk != nil && len(pos) == 0 {
			for i, e := range t.elems {
				if rubyTruthy(rt.call(blk, jsUndef, []interface{}{e})) {
					return float64(i)
				}
			}
			return jsNull
		}
		return rubyIdxOrNil(rt.rubyArrIndex(t, argAt(pos, 0)))
	case "rindex":
		for i := len(t.elems) - 1; i >= 0; i-- {
			if rt.rubyEq(t.elems[i], argAt(pos, 0)) {
				return float64(i)
			}
		}
		return jsNull
	case "uniq":
		// uniq is eql?/hash, NOT ==: `[1, 1.0].uniq` keeps BOTH in MRI, because
		// 1.eql?(1.0) is false across classes. index and include? above are ==,
		// which is why they do not share this test.
		out := &jsArray{}
		for _, e := range t.elems {
			dup := false
			for _, k := range out.elems {
				if rt.rubyEql(k, e) {
					dup = true
					break
				}
			}
			if !dup {
				out.elems = append(out.elems, e)
			}
		}
		return out
	case "to_a":
		return &jsArray{elems: append([]interface{}{}, t.elems...)}
	case "each":
		// A blockless each is an Enumerator in MRI, not the array.
		if blk == nil {
			return rubyMkEnum(&jsArray{elems: append([]interface{}{}, t.elems...)})
		}
		for _, e := range t.elems {
			rt.call(blk, jsUndef, []interface{}{e})
		}
		return t
	case "each_with_index":
		if blk == nil {
			pairs := &jsArray{}
			for i, e := range t.elems {
				pairs.elems = append(pairs.elems, &jsArray{elems: []interface{}{e, float64(i)}})
			}
			return rubyMkEnum(pairs)
		}
		for i, e := range t.elems {
			rt.call(blk, jsUndef, []interface{}{e, float64(i)})
		}
		return t
	case "each_index":
		for i := range t.elems {
			rt.call(blk, jsUndef, []interface{}{float64(i)})
		}
		return t
	case "map", "collect":
		out := &jsArray{}
		for _, e := range t.elems {
			out.elems = append(out.elems, rt.call(blk, jsUndef, []interface{}{e}))
		}
		return out
	case "select", "filter":
		out := &jsArray{}
		for _, e := range t.elems {
			if rubyTruthy(rt.call(blk, jsUndef, []interface{}{e})) {
				out.elems = append(out.elems, e)
			}
		}
		return out
	case "reject":
		out := &jsArray{}
		for _, e := range t.elems {
			if !rubyTruthy(rt.call(blk, jsUndef, []interface{}{e})) {
				out.elems = append(out.elems, e)
			}
		}
		return out
	// Ruby's sum is ordinary + (not an int32 accumulator: `[1.5, 2.5].sum` is
	// 4.0) from an Integer 0 (or from the given initial value), the block
	// applied first. MRI REFUSES a non-numeric element:
	// `["b","a"].sum` is a TypeError, not "0ba". The refusal is only against a
	// NUMERIC accumulator - `["a","b"].sum("")` is "ab" and `[[1],[2]].sum([])` is
	// [1,2]. A catchable TypeError, since MRI's is rescuable.
	case "sum":
		var s interface{} = float64(0)
		if len(pos) > 0 {
			s = pos[0]
		}
		for _, e := range t.elems {
			v := e
			if blk != nil {
				v = rt.call(blk, jsUndef, []interface{}{e})
			}
			if rubyIsNum(s) && !rubyIsNum(v) {
				panic(&jsThrown{value: rt.rubyExc(rt.rubyBuiltinClass("TypeError"),
					[]interface{}{rt.rubyCmpName(v) + " can't be coerced into " + rubyClassName(s)})})
			}
			s = rt.rubyBin("+", s, v)
		}
		return s
	// ----- the rest of the Array / Enumerable surface -----
	case "empty?":
		return len(t.elems) == 0
	case "clear":
		return rubyArrReplace(t, nil)
	case "sort":
		return &jsArray{elems: rubyArrSort(t.elems, rt.rubyArrCmpFn(blk))}
	case "sort!":
		return rubyArrReplace(t, rubyArrSort(t.elems, rt.rubyArrCmpFn(blk)))
	case "sort_by":
		return rubyUndecorate(rt.rubyArrKeyed(t, blk))
	case "sort_by!":
		return rubyArrReplace(t, rubyUndecorate(rt.rubyArrKeyed(t, blk)).elems)
	// #reverse_each yields (or enumerates) from the back. Range#reverse_each
	// delegates here.
	case "reverse_each":
		rev := &jsArray{}
		for i := len(t.elems) - 1; i >= 0; i-- {
			rev.elems = append(rev.elems, t.elems[i])
		}
		if blk == nil {
			return rubyMkEnum(rev)
		}
		for _, v := range rev.elems {
			rt.call(blk, jsUndef, []interface{}{v})
		}
		return t
	case "reverse", "reverse!":
		rev := make([]interface{}, 0, len(t.elems))
		for i := len(t.elems) - 1; i >= 0; i-- {
			rev = append(rev, t.elems[i])
		}
		if name == "reverse" {
			return &jsArray{elems: rev}
		}
		return rubyArrReplace(t, rev)
	case "join":
		sep := ""
		if len(pos) > 0 {
			sep = rt.rubyStr(pos[0])
		}
		return rt.rubyArrJoin(t, sep)
	// min / max take an optional COUNT and an optional block, and with the count
	// they answer an array - ascending for min, DESCENDING for max.
	case "min", "max":
		cmp := rt.rubyArrCmpFn(blk)
		wantMax := name == "max"
		if len(pos) > 0 {
			n := int(math.Trunc(rubyToF(pos[0])))
			asc := &jsArray{elems: rubyArrSort(t.elems, cmp)}
			if !wantMax {
				return rubyArrSub(asc, 0, n)
			}
			desc := &jsArray{}
			for i := len(asc.elems) - 1; i >= 0; i-- {
				desc.elems = append(desc.elems, asc.elems[i])
			}
			return rubyArrSub(desc, 0, n)
		}
		if len(t.elems) == 0 {
			return jsNull
		}
		best := t.elems[0]
		for _, e := range t.elems[1:] {
			c := cmp(e, best)
			if (wantMax && c > 0) || (!wantMax && c < 0) {
				best = e
			}
		}
		return best
	case "min_by", "max_by":
		if len(t.elems) == 0 {
			return jsNull
		}
		pairs := rt.rubyArrKeyed(t, blk)
		if name == "min_by" {
			return pairs[0].(*jsArray).elems[1]
		}
		return pairs[len(pairs)-1].(*jsArray).elems[1]
	case "minmax":
		if len(t.elems) == 0 {
			return &jsArray{elems: []interface{}{jsNull, jsNull}}
		}
		s := rubyArrSort(t.elems, rt.rubyArrCmpFn(blk))
		return &jsArray{elems: []interface{}{s[0], s[len(s)-1]}}
	// count: no argument is the length, an argument counts == matches, a block
	// counts the elements it accepts.
	case "count":
		if blk != nil && len(pos) == 0 {
			n := 0
			for _, e := range t.elems {
				if rubyTruthy(rt.call(blk, jsUndef, []interface{}{e})) {
					n++
				}
			}
			return float64(n)
		}
		if len(pos) == 0 {
			return float64(len(t.elems))
		}
		n := 0
		for _, e := range t.elems {
			if rt.rubyEq(e, pos[0]) {
				n++
			}
		}
		return float64(n)
	case "tally":
		d := rubyNewDict()
		keys := d.props["keys"].(*jsArray)
		vals := d.props["vals"].(*jsArray)
		for _, e := range t.elems {
			if ix := rt.rubyDictFind(keys, e); ix >= 0 {
				vals.elems[ix] = rubyToF(vals.elems[ix]) + 1
			} else {
				keys.elems = append(keys.elems, e)
				vals.elems = append(vals.elems, float64(1))
			}
		}
		return d
	case "flatten", "flatten!":
		depth := -1
		if len(pos) > 0 {
			depth = int(math.Trunc(rubyToF(pos[0])))
		}
		flat := rubyArrFlat(t, depth, &jsArray{})
		if name == "flatten" {
			return flat
		}
		return rubyArrReplace(t, flat.elems)
	case "compact", "compact!":
		out := &jsArray{}
		for _, e := range t.elems {
			if !isUndefOrNull(e) {
				out.elems = append(out.elems, e)
			}
		}
		if name == "compact" {
			return out
		}
		// The bang form answers nil when nothing was removed, which is how MRI
		// reports "no change" for every ! method that can make none.
		if len(out.elems) == len(t.elems) {
			return jsNull
		}
		return rubyArrReplace(t, out.elems)
	case "uniq!":
		uq := rt.rubyArrayMethod(t, "uniq", nil).(*jsArray)
		if len(uq.elems) == len(t.elems) {
			return jsNull
		}
		return rubyArrReplace(t, uq.elems)
	// zip is element-wise across the receiver AND every argument array, padding a
	// short one with nil; the result is always as long as the RECEIVER.
	case "zip":
		out := &jsArray{}
		for i, e := range t.elems {
			row := &jsArray{elems: []interface{}{e}}
			for _, a := range pos {
				if za, isArr := a.(*jsArray); isArr && i < len(za.elems) {
					row.elems = append(row.elems, za.elems[i])
				} else {
					row.elems = append(row.elems, jsNull)
				}
			}
			out.elems = append(out.elems, row)
		}
		return out
	// each_slice / each_cons: with a block they yield each group and answer SELF
	// (Ruby 3.1 changed this from nil, and this project's corpus is Ruby 3); with
	// no block MRI answers an Enumerator, which has no value here, so they answer
	// the array of groups - what .to_a on that Enumerator would give.
	case "each_slice", "each_cons":
		n := int(math.Trunc(rubyToF(argAt(pos, 0))))
		if n < 1 {
			rt.fail("invalid size")
		}
		groups := &jsArray{}
		if name == "each_slice" {
			for i := 0; i < len(t.elems); i += n {
				groups.elems = append(groups.elems, rubyArrSub(t, i, n))
			}
		} else {
			for i := 0; i+n <= len(t.elems); i++ {
				groups.elems = append(groups.elems, rubyArrSub(t, i, n))
			}
		}
		if blk == nil {
			return rubyMkEnum(groups)
		}
		for _, g := range groups.elems {
			rt.call(blk, jsUndef, []interface{}{g})
		}
		return t
	// sample / shuffle. There is no random source in this project and there cannot
	// be one the engines agree on byte-for-byte (see rubyRand, which is
	// deterministically zero), so these are the DETERMINISTIC draw: sample takes
	// from the front and shuffle is the identity permutation. Every invariant a
	// program can test - the length, membership, and that a shuffle is a
	// permutation of its receiver - holds; the particular ordering does not, and
	// MRI's would not be reproducible either.
	case "sample":
		if len(pos) == 0 {
			if len(t.elems) == 0 {
				return jsNull
			}
			return t.elems[0]
		}
		n := int(math.Trunc(rubyToF(pos[0])))
		if n > len(t.elems) {
			n = len(t.elems)
		}
		return rubyArrSub(t, 0, n)
	case "shuffle":
		return &jsArray{elems: append([]interface{}{}, t.elems...)}
	case "shuffle!":
		return t
	// cycle(n) { } runs the block over the whole array n times; cycle { } with no
	// count repeats FOREVER, and the way out is the block's own `break'.
	case "cycle":
		n := 0
		if len(pos) > 0 {
			n = int(math.Trunc(rubyToF(pos[0])))
		}
		if blk == nil {
			if len(pos) == 0 {
				rt.fail("cycle without a count needs a block")
			}
			cyc := &jsArray{}
			for i := 0; i < n; i++ {
				cyc.elems = append(cyc.elems, t.elems...)
			}
			return rubyMkEnum(cyc)
		}
		if len(pos) == 0 {
			if len(t.elems) == 0 {
				return jsNull
			}
			for {
				for _, e := range t.elems {
					rt.call(blk, jsUndef, []interface{}{e})
				}
			}
		}
		for i := 0; i < n; i++ {
			for _, e := range t.elems {
				rt.call(blk, jsUndef, []interface{}{e})
			}
		}
		return jsNull
	// combination(n) / permutation(n): the subsets and the ordered arrangements, in
	// MRI's own order. With a block they yield each one and answer SELF.
	case "combination", "permutation":
		k := len(t.elems)
		if len(pos) > 0 {
			k = int(math.Trunc(rubyToF(pos[0])))
		}
		var picks [][]int
		if name == "combination" {
			picks = rubyCombIdx(len(t.elems), k)
		} else {
			picks = rubyPermIdx(len(t.elems), k)
		}
		res := rubyPick(t, picks)
		if blk == nil {
			return rubyMkEnum(res)
		}
		for _, r := range res.elems {
			rt.call(blk, jsUndef, []interface{}{r})
		}
		return t
	case "take":
		return rubyArrSub(t, 0, int(math.Trunc(rubyToF(argAt(pos, 0)))))
	case "drop":
		n := int(math.Trunc(rubyToF(argAt(pos, 0))))
		return rubyArrSub(t, n, len(t.elems)-n)
	case "take_while", "drop_while":
		cut := len(t.elems)
		for i, e := range t.elems {
			if !rubyTruthy(rt.call(blk, jsUndef, []interface{}{e})) {
				cut = i
				break
			}
		}
		if name == "take_while" {
			return rubyArrSub(t, 0, cut)
		}
		return rubyArrSub(t, cut, len(t.elems)-cut)
	case "find", "detect":
		for _, e := range t.elems {
			if rubyTruthy(rt.call(blk, jsUndef, []interface{}{e})) {
				return e
			}
		}
		return jsNull
	// inject / reduce: four forms - a block, a block with an initial value, a
	// Symbol naming a binary operator, and an initial value with that Symbol.
	case "inject", "reduce":
		var sym string
		var acc interface{} = jsNull
		haveInit := false
		for _, a := range pos {
			if s, isSym := a.(jsSym); isSym {
				sym = s.s
			} else {
				acc = a
				haveInit = true
			}
		}
		from := 0
		if !haveInit {
			if len(t.elems) == 0 {
				return jsNull
			}
			acc = t.elems[0]
			from = 1
		}
		for _, e := range t.elems[from:] {
			if sym != "" {
				acc = rt.rubyBin(sym, acc, e)
			} else {
				acc = rt.call(blk, jsUndef, []interface{}{acc, e})
			}
		}
		return acc
	case "each_with_object":
		memo := argAt(pos, 0)
		for _, e := range t.elems {
			rt.call(blk, jsUndef, []interface{}{e, memo})
		}
		return memo
	case "all?", "any?", "none?", "one?":
		hits := 0
		for _, e := range t.elems {
			hit := false
			switch {
			case blk != nil:
				hit = rubyTruthy(rt.call(blk, jsUndef, []interface{}{e}))
			case len(pos) > 0:
				// With an argument the test is `arg === element`, which for a
				// class object is is_a? and == for everything else - the rule
				// js_rwhen applies.
				hit = rt.rubyPatternHit(pos[0], e)
			default:
				hit = rubyTruthy(e)
			}
			if hit {
				hits++
			}
		}
		switch name {
		case "all?":
			return hits == len(t.elems)
		case "any?":
			return hits > 0
		case "none?":
			return hits == 0
		}
		return hits == 1
	case "flat_map", "collect_concat":
		out := &jsArray{}
		for _, e := range t.elems {
			v := rt.call(blk, jsUndef, []interface{}{e})
			if sub, isArr := v.(*jsArray); isArr {
				out.elems = append(out.elems, sub.elems...)
			} else {
				out.elems = append(out.elems, v)
			}
		}
		return out
	case "group_by":
		d := rubyNewDict()
		keys := d.props["keys"].(*jsArray)
		vals := d.props["vals"].(*jsArray)
		for _, e := range t.elems {
			k := rt.call(blk, jsUndef, []interface{}{e})
			if ix := rt.rubyDictFind(keys, k); ix >= 0 {
				bucket := vals.elems[ix].(*jsArray)
				bucket.elems = append(bucket.elems, e)
			} else {
				keys.elems = append(keys.elems, k)
				vals.elems = append(vals.elems, &jsArray{elems: []interface{}{e}})
			}
		}
		return d
	case "partition":
		yes, no := &jsArray{}, &jsArray{}
		for _, e := range t.elems {
			if rubyTruthy(rt.call(blk, jsUndef, []interface{}{e})) {
				yes.elems = append(yes.elems, e)
			} else {
				no.elems = append(no.elems, e)
			}
		}
		return &jsArray{elems: []interface{}{yes, no}}
	case "rotate":
		if len(t.elems) == 0 {
			return &jsArray{}
		}
		n := 1
		if len(pos) > 0 {
			n = int(math.Trunc(rubyToF(pos[0])))
		}
		n = ((n % len(t.elems)) + len(t.elems)) % len(t.elems)
		out := &jsArray{}
		for i := range t.elems {
			out.elems = append(out.elems, t.elems[(i+n)%len(t.elems)])
		}
		return out
	case "values_at":
		out := &jsArray{}
		for _, a := range pos {
			ix := rubyArrNorm(t, a)
			if ix >= 0 && ix < len(t.elems) {
				out.elems = append(out.elems, t.elems[ix])
			} else {
				out.elems = append(out.elems, jsNull)
			}
		}
		return out
	case "slice":
		// A Range argument is a SUBSEQUENCE, read from its own endpoints.
		if r, isRng := rubyIsRange(argAt(pos, 0)); isRng {
			v, _ := rt.rubySubrange(t, r.props["begin"], r.props["end"], rt.truthy(r.props["excl"]))
			return v
		}
		ix := rubyArrNorm(t, argAt(pos, 0))
		if len(pos) > 1 {
			return rubyArrSub(t, ix, int(math.Trunc(rubyToF(pos[1]))))
		}
		if ix >= 0 && ix < len(t.elems) {
			return t.elems[ix]
		}
		return jsNull
	case "insert":
		ix := rubyArrNorm(t, argAt(pos, 0))
		out := append([]interface{}{}, t.elems[:ix]...)
		out = append(out, pos[1:]...)
		return rubyArrReplace(t, append(out, t.elems[ix:]...))
	// delete answers the value it removed (nil when it removed nothing), while
	// delete_at answers the element that was at that index.
	// delete answers the LAST element it removed - which is not always the
	// argument, since `[1, 1.0].delete(1)` removes both and MRI answers 1.0.
	case "delete":
		kept := []interface{}{}
		var hit interface{} = jsNull
		for _, e := range t.elems {
			if rt.rubyEq(e, argAt(pos, 0)) {
				hit = e
			} else {
				kept = append(kept, e)
			}
		}
		rubyArrReplace(t, kept)
		return hit
	case "delete_at":
		ix := rubyArrNorm(t, argAt(pos, 0))
		if ix < 0 || ix >= len(t.elems) {
			return jsNull
		}
		v := t.elems[ix]
		rubyArrReplace(t, append(append([]interface{}{}, t.elems[:ix]...), t.elems[ix+1:]...))
		return v
	// delete_if / keep_if always answer self; select! / reject! answer NIL when
	// they removed nothing, which is MRI's "no change" report for a ! method.
	case "delete_if", "reject!":
		kept := []interface{}{}
		for _, e := range t.elems {
			if !rubyTruthy(rt.call(blk, jsUndef, []interface{}{e})) {
				kept = append(kept, e)
			}
		}
		same := len(kept) == len(t.elems)
		rubyArrReplace(t, kept)
		if name == "reject!" && same {
			return jsNull
		}
		return t
	case "keep_if", "select!", "filter!":
		kept := []interface{}{}
		for _, e := range t.elems {
			if rubyTruthy(rt.call(blk, jsUndef, []interface{}{e})) {
				kept = append(kept, e)
			}
		}
		same := len(kept) == len(t.elems)
		rubyArrReplace(t, kept)
		if name != "keep_if" && same {
			return jsNull
		}
		return t
	case "map!", "collect!":
		out := []interface{}{}
		for _, e := range t.elems {
			out = append(out, rt.call(blk, jsUndef, []interface{}{e}))
		}
		return rubyArrReplace(t, out)
	case "concat":
		for _, a := range pos {
			if src, isArr := a.(*jsArray); isArr {
				t.dropIdx()
				t.elems = append(t.elems, src.elems...)
			}
		}
		return t
	case "fill":
		out := []interface{}{}
		for i := range t.elems {
			if blk != nil {
				out = append(out, rt.call(blk, jsUndef, []interface{}{float64(i)}))
			} else {
				out = append(out, argAt(pos, 0))
			}
		}
		return rubyArrReplace(t, out)
	case "assoc":
		for _, e := range t.elems {
			if row, isArr := e.(*jsArray); isArr && len(row.elems) > 0 && rt.rubyEq(row.elems[0], argAt(pos, 0)) {
				return row
			}
		}
		return jsNull
	case "product":
		prod := &jsArray{}
		for _, e := range t.elems {
			prod.elems = append(prod.elems, &jsArray{elems: []interface{}{e}})
		}
		for _, a := range pos {
			src, isArr := a.(*jsArray)
			if !isArr {
				continue
			}
			next := &jsArray{}
			for _, row := range prod.elems {
				for _, v := range src.elems {
					r := append([]interface{}{}, row.(*jsArray).elems...)
					next.elems = append(next.elems, &jsArray{elems: append(r, v)})
				}
			}
			prod = next
		}
		return prod
	case "dup", "clone", "entries":
		return &jsArray{elems: append([]interface{}{}, t.elems...)}
	case "freeze", "itself":
		return t
	// Array#fetch. Twin of arrayMethod in ruby-interpreter.abnf.
	case "fetch":
		n := len(t.elems)
		fx := int(rt.toNumber(argAt(pos, 0)))
		fj := fx
		if fx < 0 {
			fj += n
		}
		if fj >= 0 && fj < n {
			return t.elems[fj]
		}
		if len(pos) > 1 {
			return pos[1]
		}
		if blk != nil {
			return rt.call(blk, jsUndef, []interface{}{argAt(pos, 0)})
		}
		rt.rubyRaiseErr("IndexError", fmt.Sprintf("index %d outside of array bounds: %d...%d", fx, -n, n))
	}
	rt.rubyNoMethod(t, name)
	return nil
}

// builtinMethod implements the string and array methods of the subset, plus
// apply/call on function values.
func (rt *jsrt) builtinMethod(m *boundMethod, args []interface{}) interface{} {
	if m.name == "apply" && isCallable(m.recv) {
		var callArgs []interface{}
		if arr, ok := argAt(args, 1).(*jsArray); ok {
			callArgs = arr.elems
		}
		return rt.call(m.recv, argAt(args, 0), callArgs)
	}
	if m.name == "call" && isCallable(m.recv) {
		var callArgs []interface{}
		if len(args) > 1 {
			callArgs = args[1:]
		}
		return rt.call(m.recv, argAt(args, 0), callArgs)
	}
	argN := func(i int) float64 {
		if i < len(args) {
			return rt.toNumber(args[i])
		}
		return math.NaN()
	}
	argS := func(i int) string {
		if i < len(args) {
			return rt.toString(args[i])
		}
		return ""
	}

	switch recv := m.recv.(type) {
	case *jsArray:
		switch m.name {
		case "push":
			recv.dropIdx()
			recv.elems = append(recv.elems, args...)
			return float64(len(recv.elems))
		case "pop":
			if len(recv.elems) == 0 {
				return jsUndef
			}
			v := recv.elems[len(recv.elems)-1]
			recv.dropIdx()
			recv.elems = recv.elems[:len(recv.elems)-1]
			return v
		case "shift":
			if len(recv.elems) == 0 {
				return jsUndef
			}
			v := recv.elems[0]
			recv.dropIdx()
			recv.elems = append([]interface{}{}, recv.elems[1:]...)
			return v
		case "unshift":
			recv.dropIdx()
			recv.elems = append(append([]interface{}{}, args...), recv.elems...)
			return float64(len(recv.elems))
		case "reverse":
			// In place, and the array itself is the result, like in JS.
			recv.dropIdx()
			for i, j := 0, len(recv.elems)-1; i < j; i, j = i+1, j-1 {
				recv.elems[i], recv.elems[j] = recv.elems[j], recv.elems[i]
			}
			return recv
		case "slice":
			begin, end := sliceRange(len(recv.elems), args, rt)
			out := &jsArray{}
			for i := begin; i < end; i++ {
				out.elems = append(out.elems, recv.elems[i])
			}
			return out
		case "indexOf":
			// The from-index is relative like slice's start. Twin of mid 7 in
			// languages/lib/runtime.c's builtin_method.
			from := 0
			if len(args) > 1 {
				from = clampIndex(jsToInt(argN(1)), len(recv.elems))
			}
			for i := from; i < len(recv.elems); i++ {
				if rt.strictEq(recv.elems[i], argAt(args, 0)) {
					return float64(i)
				}
			}
			return float64(-1)
		case "join":
			sep := ","
			if len(args) > 0 {
				sep = argS(0)
			}
			parts := make([]string, len(recv.elems))
			for i, e := range recv.elems {
				if !isUndefOrNull(e) {
					parts[i] = rt.toString(e)
				}
			}
			return strJoin(parts, sep)
		case "concat":
			out := &jsArray{elems: append([]interface{}{}, recv.elems...)}
			for _, a := range args {
				if aa, ok := a.(*jsArray); ok {
					out.elems = append(out.elems, aa.elems...)
				} else {
					out.elems = append(out.elems, a)
				}
			}
			return out
		}
	case string:
		switch m.name {
		case "charCodeAt":
			i := jsToInt(argN(0)) // A missing or NaN index reads unit 0 like in JS.
			if code := rt.strCodeAt(recv, i); code >= 0 {
				return float64(code)
			}
			return math.NaN()
		case "charAt":
			return gojaCharAt(rt.strAt(recv, jsToInt(argN(0))))
		case "indexOf":
			// The position clamps to [0, length] in code units. Twin of mid 22 in
			// languages/lib/runtime.c's builtin_method.
			if len(args) > 1 {
				n := rt.strLen(recv)
				from := jsToInt(argN(1))
				if from > n {
					from = n
				}
				if from > 0 {
					p := rt.strIndexOf(rt.strRange(recv, from, n), argS(0))
					if p < 0 {
						return float64(-1)
					}
					return float64(from + p)
				}
			}
			return float64(rt.strIndexOf(recv, argS(0)))
		case "replace":
			return strings.Replace(recv, argS(0), argS(1), 1)
		case "slice":
			begin, end := sliceRange(rt.strLen(recv), args, rt)
			return rt.strRange(recv, begin, end)
		case "substring":
			begin, end := substringRange(rt.strLen(recv), args, rt)
			return rt.strRange(recv, begin, end)
		case "split":
			parts := strings.Split(recv, argS(0))
			out := &jsArray{}
			for _, p := range parts {
				out.elems = append(out.elems, p)
			}
			return out
		case "toUpperCase":
			return caseMapMode(recv, caseUpper|caseFull)
		case "toLowerCase":
			return caseMapMode(recv, caseFull|caseSigma)
		case "trim":
			return strings.TrimSpace(recv)
		}
	}
	rt.fail("unknown method %s on %s", m.name, rt.typeOf(m.recv))
	return nil
}

func argAt(args []interface{}, i int) interface{} {
	if i < len(args) {
		return args[i]
	}
	return jsUndef
}

// derefSliceValue returns the slice behind v (directly or behind one pointer),
// or an invalid Value.
func derefSliceValue(v reflect.Value) reflect.Value {
	if !v.IsValid() {
		return v
	}
	if v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}
	if v.Kind() == reflect.Slice {
		return v
	}
	return reflect.Value{}
}

// ----------------------------------------------------------------------------
// UTF-16 string semantics
//
// goja strings are UTF-16 code unit sequences, jsrt strings are Go (UTF-8)
// strings. Every operation that measures or indexes a string must count UTF-16
// code units, not bytes, or the engines diverge on any non-ASCII string.
// ASCII strings take the byte fast path.
//
// A code unit of an astral character is half a surrogate pair, and a lone
// surrogate is not a Unicode scalar value, so Go's UTF-8 encoder would replace it
// with U+FFFD - and the unit-at-a-time idiom that every string literal goes
// through (`out += String.fromCharCode(s.charCodeAt(i))` in unescapeJs) would
// take an emoji apart into two halves and destroy both before the concatenation
// could put them back. So a jsrt string is WTF-8, not UTF-8: a lone surrogate
// keeps its own three byte encoding (the encoding UTF-8 would give the code
// point if surrogates were scalars), which strUnits decodes back to the unit it
// came from. Concatenation rejoins a pair split across the seam (strConcat), so
// the halves only ever exist while the round trip is in flight, and printing
// substitutes U+FFFD for whatever lone surrogates are left (wtf8Clean), which is
// what goja writes for them too. Everything below therefore matches goja exactly
// for BMP and astral text alike (see the utf16-astral checks in
// tests/metajs-test-features.js). goja replaces a lone half by U+FFFD as it
// crosses into a Go string, so byteLen and the char array of an emitted string
// constant take it through gojaWTF8 (abnf/commonscript.go) instead.

func strASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

const (
	surrHigh0 = 0xD800 // First high surrogate.
	surrLow0  = 0xDC00 // First low surrogate.
	surrEnd   = 0xE000 // First code point after the surrogates.
)

func isSurrogate(c rune) bool { return c >= surrHigh0 && c < surrEnd }
func isHighSurr(c rune) bool  { return c >= surrHigh0 && c < surrLow0 }
func isLowSurr(c rune) bool   { return c >= surrLow0 && c < surrEnd }
func surrPair(h, l rune) rune { return 0x10000 + (h-surrHigh0)<<10 + (l - surrLow0) }

// appendWTF8 appends the three byte encoding of a lone surrogate.
func appendWTF8(b []byte, c rune) []byte {
	return append(b, 0xE0|byte(c>>12), 0x80|byte(c>>6)&0x3F, 0x80|byte(c)&0x3F)
}

// wtf8Surrogate decodes the WTF-8 encoded lone surrogate at the head of s, or
// returns 0 when s does not start with one. The three byte form of a surrogate
// always begins 0xED, which no valid UTF-8 sequence uses for another code point.
func wtf8Surrogate(s string) rune {
	if len(s) < 3 || s[0] != 0xED || s[1] < 0xA0 || s[1] > 0xBF || s[2] < 0x80 || s[2] > 0xBF {
		return 0
	}
	return rune(s[0]&0x0F)<<12 | rune(s[1]&0x3F)<<6 | rune(s[2]&0x3F)
}

// strUnits returns s as UTF-16 code units, decoding WTF-8 lone surrogates back
// to the single unit each stands for.
func strUnits(s string) []uint16 {
	u := make([]uint16, 0, len(s))
	for i := 0; i < len(s); {
		if c := s[i]; c < utf8.RuneSelf {
			u = append(u, uint16(c))
			i++
			continue
		}
		if c := wtf8Surrogate(s[i:]); c != 0 {
			u = append(u, uint16(c))
			i += 3
			continue
		}
		r, n := utf8.DecodeRuneInString(s[i:])
		if r > 0xFFFF {
			h, l := utf16.EncodeRune(r)
			u = append(u, uint16(h), uint16(l))
		} else {
			u = append(u, uint16(r))
		}
		i += n
	}
	return u
}

// strFromUnits builds a string back from UTF-16 code units: a surrogate pair
// becomes the astral character it spells, a lone surrogate its WTF-8 form.
func strFromUnits(u []uint16) string {
	b := make([]byte, 0, len(u)+len(u)/2+4)
	for i := 0; i < len(u); i++ {
		c := rune(u[i])
		if isHighSurr(c) && i+1 < len(u) && isLowSurr(rune(u[i+1])) {
			b = utf8.AppendRune(b, surrPair(c, rune(u[i+1])))
			i++
			continue
		}
		if isSurrogate(c) {
			b = appendWTF8(b, c)
			continue
		}
		b = utf8.AppendRune(b, c)
	}
	return string(b)
}

// seamPair reports the astral character spelled by a WTF-8 high surrogate at the
// end of a and a WTF-8 low surrogate at the start of b, or 0 when the seam
// between them does not split a pair.
func seamPair(a, b string) rune {
	if len(a) < 3 || len(b) < 3 || a[len(a)-3] != 0xED || b[0] != 0xED {
		return 0
	}
	h, l := wtf8Surrogate(a[len(a)-3:]), wtf8Surrogate(b)
	if !isHighSurr(h) || !isLowSurr(l) {
		return 0
	}
	return surrPair(h, l)
}

// strConcat is `a + b` on two strings: plain concatenation, except that a
// surrogate pair split across the seam is rejoined into its astral character.
// This is what makes the `out += String.fromCharCode(s.charCodeAt(i))` walk of
// a string literal give the string back unchanged.
func strConcat(a, b string) string {
	if c := seamPair(a, b); c != 0 {
		return a[:len(a)-3] + string(c) + b[3:]
	}
	return a + b
}

// strJoin is strings.Join with the same seam rejoining, for the operations that
// build one string out of many ([].join("") is a string walk put back together
// just as `out +=` is).
func strJoin(parts []string, sep string) string {
	if len(parts) < 2 {
		return strings.Join(parts, sep)
	}
	b := make([]byte, 0, len(parts)*(len(sep)+8))
	add := func(s string) {
		// The guard keeps the tail out of seamPair (and off the heap) unless the
		// seam really could split a pair.
		if len(b) >= 3 && len(s) >= 3 && b[len(b)-3] == 0xED && s[0] == 0xED {
			if c := seamPair(string(b[len(b)-3:]), s); c != 0 {
				b = utf8.AppendRune(b[:len(b)-3], c)
				s = s[3:]
			}
		}
		b = append(b, s...)
	}
	add(parts[0])
	for _, p := range parts[1:] {
		add(sep)
		add(p)
	}
	return string(b)
}

// wtf8Clean substitutes U+FFFD for the WTF-8 lone surrogates in s, which is what
// a Go string can hold and what goja writes for an unpaired half. Output goes
// through it so that the engines agree on every byte they print.
func wtf8Clean(s string) string {
	i := strings.IndexByte(s, 0xED)
	if i < 0 {
		return s
	}
	b := make([]byte, 0, len(s))
	b = append(b, s[:i]...)
	for i < len(s) {
		if c := wtf8Surrogate(s[i:]); c != 0 {
			b = utf8.AppendRune(b, utf8.RuneError)
			i += 3
			continue
		}
		b = append(b, s[i])
		i++
	}
	return string(b)
}

// gojaCharAt is charAt's one code unit result with a lone surrogate replaced by
// U+FFFD, because that is what the pinned goja does: its stringproto_charAt
// (builtin_string.go) ends in string(s.charAt(pos)), a rune-to-string conversion,
// and Go writes U+FFFD for a rune that is not a scalar value. Later goja
// versions slice the string instead and keep the half. Being right where goja is
// wrong would be a divergence, and the engines agreeing is the whole point of
// -frozen, so charAt - and only charAt - loses the half here too. Index access
// (s[i]), charCodeAt, substring and split are exact in both engines. Delete this
// once goja is updated (and the utf16-astral-charat check in
// tests/metajs-test-features.js will say so).
func gojaCharAt(ch string) string {
	if len(ch) == 3 && ch[0] == 0xED {
		return wtf8Clean(ch)
	}
	return ch
}

// unitLen is the JS length of s: its number of UTF-16 code units. It walks s,
// so the indexing accessors go through the memo below instead.
func unitLen(s string) int {
	if strASCII(s) {
		return len(s)
	}
	return len(strUnits(s))
}

// strInfo is what an indexing accessor needs to know about one string: whether
// it takes the byte fast path and, if not, its code units. Deriving that costs a
// walk of the string and interning its handle costs a hash of it, so the runtime
// remembers both per long string - without the memo the ordinary
// `for (i...) s.charCodeAt(i)` loop is quadratic in the length of s.
type strInfo struct {
	s     string
	shape int8     // 0 = not derived yet, 1 = ASCII (byte fast path), 2 = units below.
	units []uint16 // The UTF-16 code units of a non-ASCII string.
	h     uint64   // Interned handle, 0 (never a string handle) = not looked up yet.
}

// strCacheSlots is a power of two: the slot is a hash of the string's shape.
// Only strings of at least strMemoMin bytes take a slot - below that a walk or
// a map hash is cheaper than the bookkeeping, and keeping the short strings out
// stops the names and operators of a loop from evicting the string it indexes.
const (
	strCacheSlots = 16
	strMemoMin    = 64
)

func strSlot(s string) int {
	h := len(s)*131 + int(s[0])*7 + int(s[len(s)-1])
	return h & (strCacheSlots - 1)
}

// strEntry returns the record for s, reset if its slot held another string.
// The comparison e.s == s compares the string headers first, so the string a
// loop keeps re-reading hits in O(1), and a record for another value can never
// be returned in its place.
func (rt *jsrt) strEntry(s string) *strInfo {
	e := &rt.strScratch // Short strings answer without taking a slot.
	if len(s) >= strMemoMin {
		e = &rt.strCache[strSlot(s)]
	}
	if e.s != s {
		e.s, e.shape, e.units, e.h = s, 0, nil, 0
	}
	return e
}

// strMeta is strEntry with the UTF-16 shape derived.
func (rt *jsrt) strMeta(s string) *strInfo {
	e := rt.strEntry(s)
	if e.shape == 0 {
		if strASCII(s) {
			e.shape = 1
		} else {
			e.shape, e.units = 2, strUnits(s)
		}
	}
	return e
}

// strLen is the JS length of s: its number of UTF-16 code units.
func (rt *jsrt) strLen(s string) int {
	e := rt.strMeta(s)
	if e.shape == 1 {
		return len(s)
	}
	return len(e.units)
}

// strIndexOf is the JS indexOf: the byte match position converted to a
// code unit index (a UTF-8 substring match always lies on a rune boundary).
func (rt *jsrt) strIndexOf(s, sub string) int {
	p := strings.Index(s, sub)
	if p <= 0 {
		return p // -1 and 0 are the same in both worlds.
	}
	if rt.strMeta(s).shape == 1 {
		return p
	}
	return unitLen(s[:p])
}

// strAt returns the one-code-unit string at unit index i, or "" outside.
func (rt *jsrt) strAt(s string, i int) string {
	e := rt.strMeta(s)
	if e.shape == 1 {
		if i < 0 || i >= len(s) {
			return ""
		}
		return string(s[i])
	}
	if i < 0 || i >= len(e.units) {
		return ""
	}
	return strFromUnits(e.units[i : i+1])
}

// strCodeAt returns the code unit at unit index i, or -1 outside.
func (rt *jsrt) strCodeAt(s string, i int) int {
	e := rt.strMeta(s)
	if e.shape == 1 {
		if i < 0 || i >= len(s) {
			return -1
		}
		return int(s[i])
	}
	if i < 0 || i >= len(e.units) {
		return -1
	}
	return int(e.units[i])
}

// strRange slices s by code unit indexes (begin <= end, both in range).
func (rt *jsrt) strRange(s string, begin, end int) string {
	e := rt.strMeta(s)
	if e.shape == 1 {
		return s[begin:end]
	}
	return strFromUnits(e.units[begin:end])
}

// substringRange resolves JS substring(begin, end) arguments: NaN and negative
// values clamp to 0, values beyond the length clamp to it, and begin > end swap
// (unlike slice, which wraps negative indexes from the end).
func substringRange(length int, args []interface{}, rt *jsrt) (int, int) {
	begin := clampSubstringIndex(rt.toNumber(argAt(args, 0)), length)
	end := length
	if len(args) > 1 {
		if _, u := args[1].(jsUndefT); !u {
			end = clampSubstringIndex(rt.toNumber(args[1]), length)
		}
	}
	if begin > end {
		begin, end = end, begin
	}
	return begin, end
}

func clampSubstringIndex(f float64, length int) int {
	if f != f || f < 0 { // NaN and negatives clamp to 0.
		return 0
	}
	if f > float64(length) {
		return length
	}
	return int(f)
}

// sliceRange resolves JS slice(begin, end) arguments including negative indexes.
func sliceRange(length int, args []interface{}, rt *jsrt) (int, int) {
	begin := 0
	end := length
	if len(args) > 0 {
		begin = clampIndex(int(rt.toNumber(args[0])), length)
	}
	if len(args) > 1 {
		if _, u := args[1].(jsUndefT); !u {
			end = clampIndex(int(rt.toNumber(args[1])), length)
		}
	}
	if end < begin {
		end = begin
	}
	return begin, end
}

func clampIndex(i, length int) int {
	if i < 0 {
		i += length
	}
	if i < 0 {
		return 0
	}
	if i > length {
		return length
	}
	return i
}

// ----------------------------------------------------------------------------
// The js_* externals

// externs builds the machine hook table for one attached module. All functions
// take and return i64 (handles, unless stated otherwise).
func (rt *jsrt) externs(ma *machine) map[string]func(args []uint64) uint64 {
	u := rt.unwrap
	w := rt.wrap
	// The handle of a module string constant is invariant: the emitter puts every
	// literal into a module global and js_str_mem(ptr,len) is emitted at each USE
	// site, so the same few hundred constants are re-derived hundreds of
	// thousands of times per run. Remembering the handle per (ptr,len) of this
	// module turns that into one small map probe - no byte copy out of the arena
	// and no hash of the string body.
	strMemCache := map[[2]uint64]uint64{}
	boolH := func(b bool) uint64 {
		if b {
			return jsHTrue
		}
		return jsHFalse
	}

	m := map[string]func(args []uint64) uint64{
		// Scopes.
		"js_scope_new": func(a []uint64) uint64 {
			return w(&jsScope{parent: rt.scopeOf(a[0])})
		},
		"js_scope_decl": func(a []uint64) uint64 {
			name := rt.toString(u(a[1]))
			rt.scopeOf(a[0]).put(name, u(a[2]))
			if rt.traced {
				rt.trVar("decl", name, u(a[2]))
			}
			return 0
		},
		"js_scope_get": func(a []uint64) uint64 {
			return w(rt.scopeGet(rt.scopeOf(a[0]), rt.toString(u(a[1]))))
		},
		"js_scope_set": func(a []uint64) uint64 {
			rt.scopeSet(rt.scopeOf(a[0]), rt.toString(u(a[1])), u(a[2]))
			return 0
		},
		// Like js_scope_set, but an undeclared name becomes an implicit global
		// (created in the root scope) instead of an error. Used by normal JS for
		// plain `=` assignment; see scopeSetOrCreate.
		"js_scope_set_or_create": func(a []uint64) uint64 {
			rt.scopeSetOrCreate(rt.scopeOf(a[0]), rt.toString(u(a[1])), u(a[2]))
			return 0
		},

		// Scope access with 'this' fallback (Kotlin style implicit property access):
		// a name that is no local resolves against the properties of 'this'.
		"js_kget": func(a []uint64) uint64 {
			sc := rt.scopeOf(a[0])
			name := rt.toString(u(a[1]))
			for s := sc; s != nil; s = s.parent {
				if v, ok := s.get(name); ok {
					if rt.traced {
						rt.trVar("read", name, v)
					}
					return w(v)
				}
			}
			for s := sc; s != nil; s = s.parent {
				if t, ok := s.get("this"); ok {
					if obj, isObj := t.(*jsObject); isObj {
						if v, ok := obj.props[name]; ok {
							if rt.traced {
								rt.trVar("read", name, v)
							}
							return w(v)
						}
					} else if !isUndefOrNull(t) {
						// A receiver that is NOT an object still has members: a String
						// has length, an Array has length and its indexes. Inside an
						// extension function or property (fun String.f() = length) the
						// unqualified name has to reach them, exactly as it reaches an
						// object's properties above. getMember answers undefined for a
						// name it does not know, which falls through to the failure
						// below - so this only ever turns a hard error into a read.
						if v := rt.getMember(t, name); !isUndefOrNull(v) {
							if rt.traced {
								rt.trVar("read", name, v)
							}
							return w(v)
						}
					}
					break
				}
			}
			rt.fail("unknown name: %s", name)
			return 0
		},
		"js_kset": func(a []uint64) uint64 {
			sc := rt.scopeOf(a[0])
			name := rt.toString(u(a[1]))
			v := u(a[2])
			for s := sc; s != nil; s = s.parent {
				if s.has(name) {
					s.put(name, v)
					if rt.traced {
						rt.trVar("write", name, v)
					}
					return 0
				}
			}
			for s := sc; s != nil; s = s.parent {
				if t, ok := s.get("this"); ok {
					if obj, isObj := t.(*jsObject); isObj {
						if _, ok := obj.props[name]; ok {
							obj.set(name, v)
							if rt.traced {
								rt.trVar("write", name, v)
							}
							return 0
						}
					}
					break
				}
			}
			rt.fail("assignment to unknown name: %s", name)
			return 0
		},

		// MetaJS declarations and assignments: scope ops with type pinning.
		"js_tdecl": func(a []uint64) uint64 {
			rt.typedDecl(rt.scopeOf(a[0]), rt.toString(u(a[1])), u(a[2]))
			return 0
		},
		"js_tset": func(a []uint64) uint64 {
			rt.typedSet(rt.scopeOf(a[0]), rt.toString(u(a[1])), u(a[2]))
			return 0
		},

		// The source position marker: compiled in per statement when the host
		// collects positions (c.tracing), so traces and steppers know which
		// statement executes. Costs one int store at run time.
		"js_srcpos": func(a []uint64) uint64 {
			rt.curPos = int(int64(a[0]))
			if rt.traced {
				traceEmit(&TraceEvent{Ev: "stmt", Depth: rt.traceDepth, Line: lineOfPos(rt.curPos)})
			}
			return 0
		},

		// Constants.
		"js_str_mem": func(a []uint64) uint64 { // (ptr, len) -> string handle
			key := [2]uint64{a[0], a[1]}
			if h, ok := strMemCache[key]; ok {
				return h
			}
			ptr, n := a[0], a[1]
			if ptr+n > uint64(len(ma.mem)) {
				rt.fail("js_str_mem out of range")
			}
			h := rt.wrapStr(string(ma.mem[ptr : ptr+n]))
			strMemCache[key] = h
			return h
		},
		// js_gc_pin is the IDENTITY here. Natively (languages/lib/runtime.c) it also
		// registers the handle as a permanent GC root, because the emitted module has
		// globals - jsnumg.N, jsrtlib_env, jsrtlib_f_* - that hold a handle for the
		// life of the program and that the C floor's root set cannot otherwise see.
		// This half is garbage collected by Go, so there is nothing to do.
		"js_gc_pin": func(a []uint64) uint64 { return a[0] },

		// The two recoverable-error hooks. An emitter registers a closure here in
		// its boot block to make "no such variable" / "member of null" a CATCHABLE
		// exception in its language; one that registers nothing keeps the abort.
		// The twins are js_rt_missvar / js_rt_missmem in languages/lib/runtime.c.
		"js_rt_missvar": func(a []uint64) uint64 { rt.missVar = u(a[0]); return 0 },
		"js_rt_missmem": func(a []uint64) uint64 { rt.missMem = u(a[0]); return 0 },
		// The WRITE sibling of missMem: a hook of its own because node's two
		// messages differ ("Cannot read properties of" / "Cannot set properties of").
		"js_rt_missset": func(a []uint64) uint64 { rt.missSet = u(a[0]); return 0 },
		// The CALL sibling, and a value rather than a closure: see the missCall
		// field.
		"js_rt_misscall": func(a []uint64) uint64 { rt.missCall = u(a[0]); return 0 },
		// The closure hook for a call of a non-function value, which lua's pcall
		// catches as "attempt to call a nil value". Twin of js_rt_missfn in
		// languages/lib/runtime.c.
		"js_rt_missfn": func(a []uint64) uint64 { rt.missFn = u(a[0]); return 0 },

		"js_num_i": func(a []uint64) uint64 { // (i64 value) -> number handle
			return rt.wrapNum(float64(int64(a[0])))
		},
		"js_num_str": func(a []uint64) uint64 { // (string handle) -> number handle
			f, err := strconv.ParseFloat(rt.toString(u(a[0])), 64)
			if err != nil {
				rt.fail("invalid number literal %s", rt.toString(u(a[0])))
			}
			return rt.wrapNum(f)
		},

		// Objects, arrays.
		"js_obj_new": func(a []uint64) uint64 { return w(newJSObject()) },
		"js_arr_new": func(a []uint64) uint64 { return w(&jsArray{}) },
		"js_arr_new_n": func(a []uint64) uint64 { // (length handle, fill value) -> array handle
			n := int(rt.toNumber(u(a[0])))
			arr := &jsArray{elems: make([]interface{}, n)}
			fill := u(a[1])
			for i := 0; i < n; i++ {
				arr.elems[i] = fill
			}
			return w(arr)
		},
		"js_arr_push": func(a []uint64) uint64 {
			arr := u(a[0]).(*jsArray)
			arr.dropIdx()
			arr.elems = append(arr.elems, u(a[1]))
			return 0
		},
		// js_keys returns an object's own keys in insertion order (deterministic, so
		// for-in / Lua pairs / dict key enumeration stay byte-identical across engines).
		// A dict ({__dict,keys,vals}) yields its key array; a plain object yields its
		// string keys, skipping the internal __-prefixed slots (__class, __super, ...).
		"js_keys": func(a []uint64) uint64 {
			o, ok := u(a[0]).(*jsObject)
			if !ok {
				rt.fail("js_keys: not an object (got %s)", rt.typeOf(u(a[0])))
			}
			if keys, _, isDict := dictParts(o); isDict {
				return w(&jsArray{elems: append([]interface{}{}, keys.elems...)})
			}
			out := &jsArray{}
			for _, k := range o.keys {
				if len(k) >= 2 && k[0] == '_' && k[1] == '_' {
					continue
				}
				out.elems = append(out.elems, k)
			}
			return w(out)
		},
		"js_get": func(a []uint64) uint64 {
			v := rt.getMember(u(a[0]), u(a[1]))
			if rt.traced {
				rt.trMember("mread", a[0], u(a[1]), v)
			}
			return w(v)
		},
		"js_set": func(a []uint64) uint64 {
			rt.setMember(u(a[0]), u(a[1]), u(a[2]))
			if rt.traced {
				rt.trMember("mwrite", a[0], u(a[1]), u(a[2]))
			}
			return 0
		},

		// Calls.
		"js_closure": func(a []uint64) uint64 { // (func index, scope handle)
			name := "jsf_" + strconv.FormatUint(a[0], 10)
			f, ok := ma.funcs[name]
			if !ok {
				rt.fail("closure function %s not found", name)
			}
			return w(&jsClosure{fn: f, env: a[1], ma: ma})
		},
		"js_call": func(a []uint64) uint64 { // (callee, this, args array)
			args, ok := u(a[2]).(*jsArray)
			if !ok {
				rt.fail("js_call args must be an array")
			}
			return w(rt.callH(u(a[0]), u(a[1]), args.elems, a[2]))
		},
		"js_mcall": func(a []uint64) uint64 { // (target, method name, args array)
			args, ok := u(a[2]).(*jsArray)
			if !ok {
				rt.fail("js_mcall args must be an array")
			}
			return w(rt.memberCall(u(a[0]), rt.toString(u(a[1])), args.elems))
		},
		"js_rmcall": func(a []uint64) uint64 { // Ruby method dispatch: (target, method name, args array)
			args, ok := u(a[2]).(*jsArray)
			if !ok {
				rt.fail("js_rmcall args must be an array")
			}
			return w(rt.rubyMethod(u(a[0]), rt.toString(u(a[1])), args.elems))
		},
		// `def obj.m' and a def inside `class << obj': install ONE closure in the
		// receiver's own singleton table. The class machinery of ruby-to-llvm-ir.abnf
		// is build time and the receiver of these two forms is only known at run
		// time.
		"js_rsingdef": func(a []uint64) uint64 {
			o, ok := u(a[0]).(*jsObject)
			if !ok {
				rt.fail("can't define singleton method on %s", rt.typeOf(u(a[0])))
			}
			rubySingPut(o, rt.toString(u(a[1])), u(a[2]))
			return 0
		},
		// Ruby indexing: a[i], h[k], s[i]. Identical to js_pyget except that a missing
		// Hash key and an out-of-range index answer NIL instead of raising - which is
		// Ruby's Hash#[] / Array#[] / String#[], and what ruby-interpreter.abnf does.
		"js_rget": func(a []uint64) uint64 {
			return w(rt.rubyIndexGet(u(a[0]), u(a[1])))
		},
		// a[i, len] = v / a[range] = v / s[i] = c: SLICE assignment. It answers the
		// (possibly new) CONTAINER, because a Ruby String is a VALUE in this model
		// and the emitter has to write the result back - exactly what `s << "x"'
		// already does.
		"js_rsetsl": func(a []uint64) uint64 {
			return w(rt.rubySetSlice(u(a[0]), u(a[1]), u(a[2]), u(a[3])))
		},
		// Ruby index assignment: a[i] = v, h[k] = v, obj[i] = v. Like js_pyset except
		// that a user class dispatches its own `def []=`, a missing Hash key is
		// APPENDED and an Array GROWS (filling the gap with nil) - all three Ruby.
		"js_rset": func(a []uint64) uint64 {
			rt.rubyIndexSet(u(a[0]), u(a[1]), u(a[2]))
			return 0
		},
		"js_supercall": func(a []uint64) uint64 { // (super class, this, method name, args array)
			args, ok := u(a[3]).(*jsArray)
			if !ok {
				rt.fail("js_supercall args must be an array")
			}
			name := rt.toString(u(a[2]))
			// `super.s(x)` inside a STATIC method resolves on the STATIC SIDE, and a
			// static is a plain closure whose `this` is the receiver rather than an
			// instance method with the receiver PREPENDED - handing it a[1] as
			// arguments[0] shifts every argument by one. A class may hold a static and
			// an instance member of the SAME name, so "which side" is answered by WHICH
			// TABLE the member was found in. Which side to look at first is decided by
			// the receiver - a class descriptor means the caller was a static method -
			// and the fallthrough to the plain props below is what keeps this inert for
			// the other languages, none of which emits __statics.
			if isClassDesc(u(a[1])) {
				if m, owner, found := jsStaticsFind(u(a[0]), name); found && isCallable(m) {
					_ = owner
					return w(rt.call(m, u(a[1]), args.elems))
				}
			}
			// Walk the __super chain starting AT the given class (the caller already
			// resolved the superclass of the defining class).
			for cls := u(a[0]); cls != nil; {
				clsObj, ok := cls.(*jsObject)
				if !ok {
					break
				}
				if m, ok := clsObj.props[name]; ok && isCallable(m) {
					return w(rt.call(m, jsUndef, append([]interface{}{u(a[1])}, args.elems...)))
				}
				cls = clsObj.props["__super"]
			}
			// The MISS SENTINEL, for a language that asked for one: node makes
			// `super.zz()` an ordinary "is not a function" TypeError and the call
			// site raises it.
			if v, ok := rt.mcallMiss(); ok {
				return w(v)
			}
			rt.fail("unknown super method '%s'", name)
			return jsHUndefined
		},
		// 'super.x' as a VALUE and as an assignment target. Both start the lookup AT
		// the given class (the caller resolved the superclass of the DEFINING class,
		// exactly as js_supercall does) and walk __super from there, so a subclass
		// override never shadows what super.x means. A getter/setter found on the way
		// is invoked with the receiver, which is why 'this' travels separately.
		"js_supget": func(a []uint64) uint64 { // (super class, this, key) -> value
			name := rt.toString(u(a[2]))
			// The static side first when the receiver is a class descriptor - see
			// js_supercall for why the receiver is what decides, and why the walk
			// below still runs for every language that emits no __statics.
			if isClassDesc(u(a[1])) {
				if v, owner, found := jsStaticsFind(u(a[0]), name); found {
					if acc, isAcc := v.(*jsAccessor); isAcc {
						if acc.get == nil {
							return jsHUndefined
						}
						return w(rt.call(acc.get, u(a[1]), nil))
					}
					_ = owner
					return w(v)
				}
			}
			for cls := u(a[0]); cls != nil; {
				clsObj, ok := cls.(*jsObject)
				if !ok {
					break
				}
				if v, found := clsObj.props[name]; found {
					if acc, isAcc := v.(*jsAccessor); isAcc {
						if acc.get == nil {
							return jsHUndefined
						}
						return w(rt.call(acc.get, u(a[1]), nil))
					}
					// An INSTANCE method found on the super chain is stored with
					// this engine's __class calling convention - the receiver is
					// PREPENDED to the arguments - so the raw closure would run with
					// `this` undefined and every argument shifted by one, where
					// node runs the parent's method on the current receiver. The
					// answer is a bound shim, the same shape js_jsvget builds for
					// `const f = p.m`. `super.who` STORED and called later is
					// node's own undefined-receiver case and stays that way: the
					// shim carries the receiver the READ was made with, which is
					// exactly what a Reference does.
					if isCallable(v) {
						self := u(a[1])
						bound := v
						return w(jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
							return rt.call(bound, jsUndef, append([]interface{}{self}, args...))
						}))
					}
					return w(v)
				}
				cls = clsObj.props["__super"]
			}
			// Nothing on the class chain is UNDEFINED, and deliberately not a read of
			// the receiver: node prints NaN for 'super.zz += 4' after 'super.zz = 3'
			// (the store lands on the receiver, the read does not see it), so falling
			// back to the own property would be a divergence from the real language.
			return jsHUndefined
		},
		"js_supset": func(a []uint64) uint64 { // (super class, this, key, value)
			name := rt.toString(u(a[2]))
			// The static side first when the receiver is a class descriptor; see
			// js_supercall.
			if isClassDesc(u(a[1])) {
				if v, owner, found := jsStaticsFind(u(a[0]), name); found {
					if acc, isAcc := v.(*jsAccessor); isAcc {
						if acc.set != nil {
							rt.call(acc.set, u(a[1]), []interface{}{u(a[3])})
						}
						return 0
					}
					_ = owner
					if o, ok := u(a[1]).(*jsObject); ok {
						jsStaticsOf(o, true).set(name, u(a[3]))
						return 0
					}
				}
			}
			for cls := u(a[0]); cls != nil; {
				clsObj, ok := cls.(*jsObject)
				if !ok {
					break
				}
				if v, found := clsObj.props[name]; found {
					if acc, isAcc := v.(*jsAccessor); isAcc {
						if acc.set != nil {
							rt.call(acc.set, u(a[1]), []interface{}{u(a[3])})
						}
						return 0
					}
					break
				}
				cls = clsObj.props["__super"]
			}
			// No setter on the super chain: 'super.x = v' performs an ordinary
			// [[Set]] whose RECEIVER is 'this', so the own property lands there.
			if o, ok := u(a[1]).(*jsObject); ok {
				o.set(name, u(a[3]))
				return 0
			}
			rt.setMember(u(a[1]), u(a[2]), u(a[3]))
			return 0
		},
		"js_arg": func(a []uint64) uint64 { // (args array, index) -> value
			arr := u(a[0]).(*jsArray)
			if int(a[1]) < len(arr.elems) {
				return w(arr.elems[a[1]])
			}
			return jsHUndefined
		},

		// Operators.
		"js_truthy": func(a []uint64) uint64 {
			if rt.truthy(u(a[0])) {
				return 1
			}
			return 0
		},
		"js_add": func(a []uint64) uint64 { return w(rt.jsAdd(u(a[0]), u(a[1]))) },
		"js_jadd": func(a []uint64) uint64 { // Java/C# +: string concat, 32 bit add for two int values, float add otherwise.
			l, r := u(a[0]), u(a[1])
			_, ls := l.(string)
			_, rs := r.(string)
			if ls || rs {
				return rt.wrapStr(strConcat(rt.javaString(l), rt.javaString(r)))
			}
			if jvmIsFlo(l) || jvmIsFlo(r) {
				// A double operand pulls the whole addition into floating point.
				return w(jsJFlo{f: rt.toNumber(l) + rt.toNumber(r), sty: jvmStyleOf(l, r)})
			}
			ln, rn := rt.toNumber(l), rt.toNumber(r)
			if isInt32Value(ln) && isInt32Value(rn) {
				return rt.wrapNum(float64(int32(int64(ln) + int64(rn))))
			}
			return rt.wrapNum(ln + rn)
		},
		// Java & | ^: the non-short-circuit boolean operator when both sides are
		// booleans, the int32 bit operation otherwise.
		"js_jband": func(a []uint64) uint64 {
			if lb, lok := u(a[0]).(bool); lok {
				if rb, rok := u(a[1]).(bool); rok {
					return boolH(lb && rb)
				}
			}
			return rt.wrapNum(float64(rt.toInt32(u(a[0])) & rt.toInt32(u(a[1]))))
		},
		"js_jbor": func(a []uint64) uint64 {
			if lb, lok := u(a[0]).(bool); lok {
				if rb, rok := u(a[1]).(bool); rok {
					return boolH(lb || rb)
				}
			}
			return rt.wrapNum(float64(rt.toInt32(u(a[0])) | rt.toInt32(u(a[1]))))
		},
		"js_jbxor": func(a []uint64) uint64 {
			if lb, lok := u(a[0]).(bool); lok {
				if rb, rok := u(a[1]).(bool); rok {
					return boolH(lb != rb)
				}
			}
			return rt.wrapNum(float64(rt.toInt32(u(a[0])) ^ rt.toInt32(u(a[1]))))
		},
		"js_sub": func(a []uint64) uint64 {
			if rt.hasBigInt {
				if r, ok := bigArith('-', u(a[0]), u(a[1])); ok {
					return w(r)
				}
			}
			return rt.wrapNum(rt.toNumber(u(a[0])) - rt.toNumber(u(a[1])))
		},
		"js_mul": func(a []uint64) uint64 {
			if rt.hasBigInt {
				if r, ok := bigArith('*', u(a[0]), u(a[1])); ok {
					return w(r)
				}
			}
			return rt.wrapNum(rt.toNumber(u(a[0])) * rt.toNumber(u(a[1])))
		},
		"js_div": func(a []uint64) uint64 {
			if rt.hasBigInt {
				if r, ok := bigArith('/', u(a[0]), u(a[1])); ok {
					return w(r)
				}
			}
			return rt.wrapNum(rt.toNumber(u(a[0])) / rt.toNumber(u(a[1])))
		},
		"js_mod": func(a []uint64) uint64 {
			if rt.hasBigInt {
				if r, ok := bigArith('%', u(a[0]), u(a[1])); ok {
					return w(r)
				}
			}
			return rt.wrapNum(math.Mod(rt.toNumber(u(a[0])), rt.toNumber(u(a[1]))))
		},
		"js_eq":  func(a []uint64) uint64 { return boolH(rt.looseEq(u(a[0]), u(a[1]))) },
		"js_ne":  func(a []uint64) uint64 { return boolH(!rt.looseEq(u(a[0]), u(a[1]))) },
		"js_seq": func(a []uint64) uint64 { return boolH(rt.strictEq(u(a[0]), u(a[1]))) },
		"js_sne": func(a []uint64) uint64 { return boolH(!rt.strictEq(u(a[0]), u(a[1]))) },
		"js_lt":  func(a []uint64) uint64 { return boolH(rt.jsCompare(u(a[0]), u(a[1])) == -1) },
		"js_gt":  func(a []uint64) uint64 { return boolH(rt.jsCompare(u(a[0]), u(a[1])) == 1) },
		"js_le": func(a []uint64) uint64 {
			c := rt.jsCompare(u(a[0]), u(a[1]))
			return boolH(c == -1 || c == 0)
		},
		"js_ge": func(a []uint64) uint64 {
			c := rt.jsCompare(u(a[0]), u(a[1]))
			return boolH(c == 1 || c == 0)
		},
		// Bitwise operators work on ToInt32/ToUint32 like in JS.
		"js_bor":  func(a []uint64) uint64 { return rt.wrapNum(float64(rt.toInt32(u(a[0])) | rt.toInt32(u(a[1])))) },
		"js_bxor": func(a []uint64) uint64 { return rt.wrapNum(float64(rt.toInt32(u(a[0])) ^ rt.toInt32(u(a[1])))) },
		"js_band": func(a []uint64) uint64 { return rt.wrapNum(float64(rt.toInt32(u(a[0])) & rt.toInt32(u(a[1])))) },
		"js_shl": func(a []uint64) uint64 {
			return rt.wrapNum(float64(rt.toInt32(u(a[0])) << (uint32(rt.toInt32(u(a[1]))) & 31)))
		},
		"js_shr": func(a []uint64) uint64 {
			return rt.wrapNum(float64(rt.toInt32(u(a[0])) >> (uint32(rt.toInt32(u(a[1]))) & 31)))
		},
		"js_ushr": func(a []uint64) uint64 {
			return rt.wrapNum(float64(uint32(rt.toInt32(u(a[0]))) >> (uint32(rt.toInt32(u(a[1]))) & 31)))
		},
		"js_neg": func(a []uint64) uint64 {
			if rt.hasBigInt {
				if bi, ok := u(a[0]).(*jsBigInt); ok {
					return w(&jsBigInt{v: new(big.Int).Neg(bi.v)})
				}
			}
			return rt.wrapNum(-rt.toNumber(u(a[0])))
		},
		"js_not":    func(a []uint64) uint64 { return boolH(!rt.truthy(u(a[0]))) },
		"js_typeof": func(a []uint64) uint64 { return rt.wrapStr(rt.typeOf(u(a[0]))) },
		// The shared dynamic type test behind `is`/`instanceof`-style checks of the
		// typed languages. The interpreter grammars carry a JS twin (rtIsType in
		// lib/interp-core.js) that must match this logic exactly.
		"js_is_type": func(a []uint64) uint64 {
			t, _ := u(a[1]).(string)
			return boolH(rt.isTypeName(u(a[0]), t))
		},
		// A builtin exception instance for the Python pair: ExcName(args...) yields
		// {__class: {__name: ExcName}, args: [...]}, the shape js_is_type/rtIsType
		// discriminate and e.args[N] reads.
		"js_pyexc": func(a []uint64) uint64 {
			name, _ := u(a[0]).(string)
			args, _ := u(a[1]).(*jsArray)
			cls := newJSObject()
			cls.set("__name", name)
			inst := newJSObject()
			inst.set("__class", cls)
			arr := &jsArray{}
			if args != nil {
				arr.elems = append(arr.elems, args.elems...)
			}
			inst.set("args", arr)
			return w(inst)
		},
		"js_tonum": func(a []uint64) uint64 { return rt.wrapNum(rt.toNumber(u(a[0]))) },
		// Ruby's `raise' in one step: (operand, message|undef). rubyMakeExc turns a
		// class / a string / a bare re-raise into the exception object, which is then
		// thrown exactly like js_throw does.
		"js_rraise": func(a []uint64) uint64 {
			panic(&jsThrown{value: rt.rubyMakeExc(u(a[0]), u(a[1]))})
		},
		// The same raise, plus the SOURCE SITE of the `raise` that wrote it -
		// "prog.rb:2:in `<main>'", exactly MRI's first backtrace frame. Only
		// makeRaise emits this form, so a re-raise (js_rraise, which hands the same
		// object back) keeps the ORIGINAL site, as MRI does. js_rtop prints it when
		// the exception reaches the top of the program.
		"js_rraiseat": func(a []uint64) uint64 {
			e := rt.rubyMakeExc(u(a[0]), u(a[1]))
			if o, isObj := e.(*jsObject); isObj {
				if _, had := o.props["__rwhere"]; !had {
					if where, isStr := u(a[2]).(string); isStr {
						rt.setMember(o, "__rwhere", where)
					}
				}
			}
			panic(&jsThrown{value: e})
		},
		// THE TOP OF THE PROGRAM: an exception nobody rescued, printed with its
		// message and class rather than by the generic uncaught-throw handler.
		"js_rtop": func(a []uint64) uint64 {
			fmt.Fprintln(outWriter, wtf8Clean(rt.rubyExcTopLine(u(a[0]))))
			os.Exit(1)
			return 0
		},
		"js_throw": func(a []uint64) uint64 {
			// Raise the thrown value as a Go panic; the nearest js_try recovers it.
			// An uncaught one is turned into a clean runtime error at the program
			// boundary (runJSModule) or surfaces as a tag-script bug under -frozen.
			panic(&jsThrown{value: u(a[0])})
		},
		"js_try": func(a []uint64) uint64 { // (try closure, catch closure|undef, finally closure|undef)
			tryC, catchC, finallyC := u(a[0]), u(a[1]), u(a[2])
			hasCatch := isCallable(catchC)
			hasFinally := isCallable(finallyC)
			// run executes one clause closure and catches ANY panic (a user
			// throw or a runtime error), so the finally clause still runs and
			// the panic can be re-raised afterwards.
			run := func(c interface{}, args []interface{}) (res interface{}, caught interface{}) {
				// A panic unwinds the callInner frames without their pops, so the
				// `this` stack has to be cut back to the depth of this try.
				depth := len(rt.thisStack)
				ntDepth := len(rt.newTargetStack)
				defer func() {
					if caught = recover(); caught != nil && rt.trackThis {
						if len(rt.thisStack) > depth {
							rt.thisStack = rt.thisStack[:depth]
						}
						if len(rt.newTargetStack) > ntDepth {
							rt.newTargetStack = rt.newTargetStack[:ntDepth]
						}
					}
				}()
				res = rt.call(c, jsUndef, args)
				return
			}
			result, pending := run(tryC, nil)
			if pending != nil {
				if exc, isThrow := pending.(*jsThrown); isThrow && hasCatch {
					// Ruby's $!: a bare `raise' inside the catch clause re-raises
					// this value (js_rraise). Restored afterwards so a nested try
					// hands the outer clause its own exception back.
					savedCur := rt.rubyCurExc
					rt.rubyCurExc = exc.value
					result, pending = run(catchC, []interface{}{exc.value})
					rt.rubyCurExc = savedCur
				}
			}
			if hasFinally {
				finRes, finPanic := run(finallyC, nil)
				if finPanic != nil {
					// A throw (or error) from finally itself replaces everything.
					pending = finPanic
				} else if ctl, isCtl := finRes.(*jsCtl); isCtl {
					// The finally body is a ctl closure: a return/break/continue
					// in it overrides the try/catch completion AND swallows a
					// pending throw, like in JS.
					pending = nil
					result = ctl
				}
			}
			if pending != nil {
				panic(pending)
			}
			return w(result)
		},
		// The dynamic `this` of the innermost compiled call (see thisStack). Outside
		// every call - at the top level - it is undefined, like a sloppy-mode script's.
		"js_this": func(a []uint64) uint64 {
			if len(rt.thisStack) == 0 {
				return jsHUndefined
			}
			return w(rt.thisStack[len(rt.thisStack)-1])
		},
		// A BigInt literal: the digits of '123n' as an arbitrary precision integer.
		"js_bigint": func(a []uint64) uint64 {
			v, ok := new(big.Int).SetString(rt.toString(u(a[0])), 0)
			if !ok {
				rt.fail("invalid BigInt literal %q", rt.toString(u(a[0])))
			}
			rt.hasBigInt = true
			return w(&jsBigInt{v: v})
		},
		// Wrap a compiled closure into a generator FUNCTION: calling the result does not
		// run the body but hands back a generator object over it.
		"js_genfn": func(a []uint64) uint64 {
			body := u(a[0])
			return w(jsHostFunc("generator", func(rt *jsrt, this uint64, args []interface{}) interface{} {
				return &jsGenerator{fn: body, args: append([]interface{}{}, args...)}
			}))
		},
		// 'yield v' from inside a generator body: suspend, hand v to the pending next()
		// and answer the value that the following next(x) sends in.
		"js_yield": func(a []uint64) uint64 {
			g := rt.curGen
			if g == nil {
				rt.fail("yield outside of a generator")
			}
			g.yields <- &genStep{value: u(a[0]), done: false}
			sent := <-g.resume
			// finish() resumes with genExit to CLOSE the body: this yield does
			// not answer, it panics, so the body unwinds through its own
			// finally clauses.
			if sent == genExit {
				panic(genExit)
			}
			// throwInto() resumes with a throw-in record: this yield does not
			// answer either, it raises - as an ORDINARY throw, so the body's own
			// catch arms may take it. The record travels on the resume value and
			// nothing is kept across the suspension, which is what makes it
			// per-coroutine by construction.
			if t, ok := sent.(*genThrowSignal); ok {
				panic(&jsThrown{value: t.v})
			}
			return w(sent)
		},
		// The sequence a for-of loop iterates: arrays, strings and objects are their own,
		// a generator is materialized (the subset iterates by index, so a lazy source has
		// to be drained first).
		"js_iterable": func(a []uint64) uint64 {
			if g, ok := u(a[0]).(*jsGenerator); ok {
				return w(g.drain(rt))
			}
			return a[0]
		},
		// Like js_call, but the callee is invoked AS A CONSTRUCTOR, so new.target inside
		// it is the callee instead of undefined.
		"js_call_new": func(a []uint64) uint64 {
			args, ok := u(a[2]).(*jsArray)
			if !ok {
				rt.fail("js_call_new args must be an array")
			}
			rt.pendingNewTarget = u(a[0])
			return w(rt.callH(u(a[0]), u(a[1]), args.elems, a[2]))
		},
		// 'new.target': the constructor of the innermost call, or undefined.
		"js_newtarget": func(a []uint64) uint64 {
			if len(rt.newTargetStack) == 0 || rt.newTargetStack[len(rt.newTargetStack)-1] == nil {
				return jsHUndefined
			}
			return w(rt.newTargetStack[len(rt.newTargetStack)-1])
		},
		// 'key in obj' / Object.prototype.hasOwnProperty: own properties of an object,
		// index and 'length' of an array or string.
		"js_has": func(a []uint64) uint64 {
			key := rt.toString(u(a[1]))
			switch o := u(a[0]).(type) {
			case *jsObject:
				_, ok := o.props[key]
				if !ok {
					ok = rt.findClassAccessor(o, key) != nil
				}
				return boolH(ok)
			case *jsArray:
				if key == "length" {
					return boolH(true)
				}
				if !maybeNumeric(u(a[1])) {
					return boolH(false)
				}
				idx := rt.toNumber(u(a[1]))
				return boolH(idx == math.Trunc(idx) && idx >= 0 && int(idx) < len(o.elems))
			case string:
				if key == "length" {
					return boolH(true)
				}
				if !maybeNumeric(u(a[1])) {
					return boolH(false)
				}
				idx := rt.toNumber(u(a[1]))
				return boolH(idx == math.Trunc(idx) && idx >= 0 && int(idx) < rt.strLen(o))
			}
			rt.fail("'in' needs an object on the right side")
			return boolH(false)
		},
		// 'delete obj.key': drop an own property (and its key-order entry). Like in JS
		// it answers true even when there was nothing to delete.
		"js_del": func(a []uint64) uint64 {
			key := rt.toString(u(a[1]))
			if o, ok := u(a[0]).(*jsObject); ok {
				if _, had := o.props[key]; had {
					delete(o.props, key)
					for i, k := range o.keys {
						if k == key {
							o.keys = append(o.keys[:i], o.keys[i+1:]...)
							break
						}
					}
				}
			}
			return boolH(true)
		},
		// 'typeof name' for a name that need not be declared: an unbound name is
		// "undefined" rather than the reference error a plain js_scope_get raises.
		"js_scope_typeof": func(a []uint64) uint64 {
			name := rt.toString(u(a[1]))
			for sc := rt.scopeOf(a[0]); sc != nil; sc = sc.parent {
				if v, ok := sc.get(name); ok {
					return rt.wrapStr(rt.typeOf(v))
				}
			}
			return rt.wrapStr("undefined")
		},
		// Is `name` bound in THIS scope - no chain walk. It is the EXTERN twin of
		// the scopeHas builtin (host 67 of languages/lib/runtime.c, jsrtint.go's
		// scBindings): a builtin is callable from MetaJS, an extern from emitted IR.
		//
		// js_scope_typeof is the chain walk and cannot express it - it answers
		// "undefined" for an absent name and for a slot holding undefined alike,
		// and it finds a name an ENCLOSING scope binds. python-to-llvm-ir.abnf's
		// class fill needs the own-scope test, or a name the class body did
		// not reach is copied onto the class FROM THE MODULE:
		//     w = "module"
		//     class K:
		//         if False:
		//             w = "inner"
		//     K.w                     -> CPython raises AttributeError
		"js_scope_has": func(a []uint64) uint64 {
			name := rt.toString(u(a[1]))
			sc := rt.scopeOf(a[0])
			if sc == nil {
				return boolH(false)
			}
			_, ok := sc.get(name)
			return boolH(ok)
		},
		// The value `name` holds in THIS scope, or undefined when it is not bound
		// here - no chain walk, no abort. Twin of js_scope_own in
		// languages/lib/runtime.c.
		"js_scope_own": func(a []uint64) uint64 {
			name := rt.toString(u(a[1]))
			sc := rt.scopeOf(a[0])
			if sc == nil {
				return w(jsUndef)
			}
			v, ok := sc.get(name)
			if !ok {
				return w(jsUndef)
			}
			if rt.traced {
				rt.trVar("read", name, v)
			}
			return w(v)
		},
		// 'v instanceof C': walk the __class/__super chain of the instance and compare
		// with the class descriptor. Anything that is not an instance answers false.
		"js_instanceof": func(a []uint64) uint64 {
			o, ok := u(a[0]).(*jsObject)
			if !ok {
				return boolH(false)
			}
			target := u(a[1])
			for cls := o.props["__class"]; cls != nil; {
				if cls == target {
					return boolH(true)
				}
				clsObj, isObj := cls.(*jsObject)
				if !isObj {
					break
				}
				cls = clsObj.props["__super"]
			}
			return boolH(false)
		},
		// ----- Ruby's Symbol (see the jsSym type) -----
		// :name from its name. Only ruby-to-llvm-ir.abnf emits this.
		"js_sym": func(a []uint64) uint64 { return w(jsSym{s: rt.toString(u(a[0]))}) },
		// ----- Ruby's numeric tower and operators (see the jsFlo/jsRat/jsCpx types) -----
		// Only ruby-to-llvm-ir.abnf emits these; every other grammar keeps the shared
		// js_jadd/js_sub/... operators, whose semantics are untouched by them.
		"js_rflo":  func(a []uint64) uint64 { return w(jsFlo{f: rt.toNumber(u(a[0]))}) },
		"js_rrat":  func(a []uint64) uint64 { return w(rubyMkRat(rt.toNumber(u(a[0])), rt.toNumber(u(a[1])))) },
		"js_rcpx":  func(a []uint64) uint64 { return w(jsCpx{re: u(a[0]), im: u(a[1])}) },
		"js_radd":  func(a []uint64) uint64 { return w(rt.rubyBin("+", u(a[0]), u(a[1]))) },
		"js_rsub":  func(a []uint64) uint64 { return w(rt.rubyBin("-", u(a[0]), u(a[1]))) },
		"js_rmul":  func(a []uint64) uint64 { return w(rt.rubyBin("*", u(a[0]), u(a[1]))) },
		"js_rdiv":  func(a []uint64) uint64 { return w(rt.rubyBin("/", u(a[0]), u(a[1]))) },
		"js_rmod":  func(a []uint64) uint64 { return w(rt.rubyBin("%", u(a[0]), u(a[1]))) },
		"js_rpow":  func(a []uint64) uint64 { return w(rt.rubyBin("**", u(a[0]), u(a[1]))) },
		"js_rband": func(a []uint64) uint64 { return w(rt.rubyLogicBin("&", u(a[0]), u(a[1]))) },
		"js_rbor":  func(a []uint64) uint64 { return w(rt.rubyLogicBin("|", u(a[0]), u(a[1]))) },
		"js_rbxor": func(a []uint64) uint64 { return w(rt.rubyLogicBin("^", u(a[0]), u(a[1]))) },
		// << appends to an Array (in place, so `a << b << c` chains), concatenates a
		// String and shifts an Integer - one operator, three receivers, like Ruby.
		"js_rshl": func(a []uint64) uint64 { return w(rt.rubyBin("<<", u(a[0]), u(a[1]))) },
		"js_rshr": func(a []uint64) uint64 { return w(rt.rubyBin(">>", u(a[0]), u(a[1]))) },
		"js_rneg": func(a []uint64) uint64 { return w(rt.rubyNeg(u(a[0]))) },
		// ~x is -(x+1) at any size, which math/big's Not is exactly.
		"js_rbnot": func(a []uint64) uint64 {
			// Past 2^53 the double cannot hold the +1, so ~9007199254740992 must
			// go through the big path to answer -9007199254740993.
			if b, ok := rubyBigOf(u(a[0])); ok && b.CmpAbs(rubyBigMax) >= 0 {
				return w(rt.rubyBigNarrow(new(big.Int).Not(b)))
			}
			return w(-rubyToF(u(a[0])) - 1)
		},
		"js_req": func(a []uint64) uint64 { return boolH(rt.rubyEq(u(a[0]), u(a[1]))) },
		"js_rne": func(a []uint64) uint64 { return boolH(!rt.rubyEq(u(a[0]), u(a[1]))) },
		// < > <= >= between two MODULES is the ancestry relation, not a value
		// comparison: `A < M' after `include M' is true and `A < String' is NIL,
		// where rubyCmp's contract is to raise.
		"js_rlt": func(a []uint64) uint64 { return rt.rubyRel("<", u(a[0]), u(a[1])) },
		"js_rgt": func(a []uint64) uint64 { return rt.rubyRel(">", u(a[0]), u(a[1])) },
		"js_rle": func(a []uint64) uint64 { return rt.rubyRel("<=", u(a[0]), u(a[1])) },
		"js_rge": func(a []uint64) uint64 { return rt.rubyRel(">=", u(a[0]), u(a[1])) },
		"js_rcmp": func(a []uint64) uint64 { // <=>: nil when the two cannot be compared
			c, ok := rt.rubySpaceship(u(a[0]), u(a[1]))
			if !ok {
				return w(jsNull)
			}
			return rt.wrapNum(float64(c))
		},
		// lo..hi / lo...hi with BOTH bounds: the range object, which carries its
		// raw endpoints. Kept separate from js_rrangeopen only because the emitter
		// distinguishes the two syntactic forms; the two build the same thing.
		"js_rrangearr": func(a []uint64) uint64 {
			return w(rt.rubyRangeValue(u(a[0]), u(a[1]), rt.truthy(u(a[2]))))
		},
		// s[lo..hi] / a[lo...hi] written as a RANGE LITERAL at the subscript: the
		// endpoints arrive RAW - `(1..-1)` is the empty array while `s[1..-1]` means
		// "from index 1 to the last character". A nil endpoint is the beginning / the
		// end (`s[1..]`), and an exclusive end with no endpoint is still the end.
		// Anything that is not a String or an Array (a Hash whose key is a range, a
		// user class' own `def []`) is handed the RANGE OBJECT.
		"js_rsubrange": func(a []uint64) uint64 {
			if v, ok := rt.rubySubrange(u(a[0]), u(a[1]), u(a[2]), rt.truthy(u(a[3]))); ok {
				return w(v)
			}
			return w(rt.rubyIndexGet(u(a[0]), rubyOpenRange(u(a[1]), u(a[2]), rt.truthy(u(a[3])))))
		},
		// THE Range constructor: 1..3, 1...3, (5..), (..9) - all of them.
		"js_rrangeopen": func(a []uint64) uint64 {
			return w(rubyOpenRange(u(a[0]), u(a[1]), rt.truthy(u(a[2]))))
		},
		"js_rcase": func(a []uint64) uint64 {
			l, r := u(a[0]), u(a[1])
			if lo, ok := l.(*jsObject); ok {
				if _, isRng := lo.props["__rrange"]; isRng {
					return boolH(rt.rubyOpenRangeCover(lo, r))
				}
			}
			if la, ok := l.(*jsArray); ok {
				for _, e := range la.elems {
					if rt.rubyEq(e, r) {
						return boolH(true)
					}
				}
				return boolH(false)
			}
			if lo, ok := l.(*jsObject); ok {
				if _, isCls := lo.props["__isclass"]; isCls {
					return boolH(rt.rubyIsA(r, l))
				}
			}
			return boolH(rt.rubyEq(l, r))
		},
		// A plain `when V` value: a class matches by is_a?, everything else by ==
		// (an Array compares structurally here - only a RANGE means membership,
		// which is what js_rcase above is for).
		"js_rwhen": func(a []uint64) uint64 {
			l, r := u(a[0]), u(a[1])
			if lo, ok := l.(*jsObject); ok {
				if _, isCls := lo.props["__isclass"]; isCls {
					return boolH(rt.rubyIsA(r, l))
				}
			}
			// `when' is Ruby's ===, and a Range's === is MEMBERSHIP. The emitter
			// routes a range LITERAL to js_rcase by syntax, so this only bites when
			// the when value is a variable (`when r').
			if lo, isRng := rubyIsRange(l); isRng {
				return boolH(rt.rubyOpenRangeCover(lo, r))
			}
			return boolH(rt.rubyEq(l, r))
		},
		// ----- Ruby's $globals and @@class variables -----
		// A $global lives outside every scope (a write in a method is visible
		// everywhere) and reads as nil until it is first assigned.
		// ----- Ruby parameter forms (see emitFunc in ruby-to-llvm-ir.abnf) -----
		// Whether an argument was absent (js_arg answers undefined for one that was
		// not passed). Ruby's nil is a VALUE and must not trigger a default.
		"js_rundef": func(a []uint64) uint64 {
			_, isUndef := u(a[0]).(jsUndefT)
			return boolH(isUndef)
		},
		// *rest: the arguments from index i on, as an array (a trailing block closure
		// is not one of them).
		"js_rargrest": func(a []uint64) uint64 {
			arr, ok := u(a[0]).(*jsArray)
			out := &jsArray{}
			if !ok {
				return w(out)
			}
			for i := int(a[1]); i < len(arr.elems); i++ {
				if i == len(arr.elems)-1 && rubyIsBlkArg(arr.elems[i]) {
					break
				}
				out.elems = append(out.elems, arr.elems[i])
			}
			return w(out)
		},
		// ALL of one method's POSITIONAL parameters, bound in one step. Ruby does not
		// bind them left to right at fixed indices: everything AFTER a *splat is
		// counted from the END (`def d(a, b=2, *r, c); d(1,2,3,1)` binds c to 1, not
		// to args[3]), and an OPTIONAL parameter only takes an argument when there
		// are SPARE ones beyond what the required parameters need. Both facts need
		// the whole shape at once, which is why this is one call rather than a
		// per-parameter accessor.
		//
		//   off   - the index positional arguments start at (1 for a method, whose
		//           args[0] is self).
		//   kinds - one character per positional parameter: "0" required, "1"
		//           optional, "2" splat.
		//   wantsKw - the method declares keyword parameters, so a trailing Hash is
		//           the keyword hash and NOT a positional value.
		//
		// The answer is one value per character: undefined for an optional that was
		// not given (so the emitter's js_rundef default branch still fires), an
		// Array for the splat, nil for a required one that was not passed. Mirrors
		// bindParams of ruby-interpreter.abnf.
		"js_rposbind": func(a []uint64) uint64 {
			out := &jsArray{}
			var pos []interface{}
			if arr, ok := u(a[0]).(*jsArray); ok {
				if off := int(a[1]); off < len(arr.elems) {
					pos = append(pos, arr.elems[off:]...)
				}
			}
			kinds := rt.toString(u(a[2]))
			// The keyword Hash is the last argument BEFORE the trailing block slot,
			// not the last argument outright.
			if rubyTruthy(u(a[3])) {
				kwAt := len(pos) - 1
				if kwAt >= 0 && rubyIsBlkArg(pos[kwAt]) {
					kwAt--
				}
				if kwAt >= 0 {
					if _, _, isDict := dictParts(pos[kwAt]); isDict {
						pos = append(pos[:kwAt], pos[kwAt+1:]...)
					}
				}
			}
			// A block travels as the last positional argument, wrapped by
			// js_rblkwrap, and is not a value any parameter may take - a Proc
			// handed over as an ORDINARY argument is not wrapped and binds
			// normally, which is what `def m(f = 3); end; m(some_proc)` needs.
			npos := len(pos)
			if npos > 0 && rubyIsBlkArg(pos[npos-1]) {
				npos--
			}
			required := 0
			for i := 0; i < len(kinds); i++ {
				if kinds[i] == '0' {
					required++
				}
			}
			// Too few positional arguments is an ArgumentError in MRI, and this
			// is the general shape of the check; the all-required fast path skips
			// js_rposbind entirely and asks it separately, through js_rargchk. A
			// BLOCK never reaches here - the emitter binds one with js_rblockarg,
			// and Ruby arity-checks methods and lambdas only.
			if npos < required {
				rt.rubyArgCountErr(npos, kinds)
			}
			// TOO MANY IS AN ERROR TOO, and only a *splat lifts the ceiling. MRI
			// raises for `->(x, y = 2) { }.call(1, 2, 3)` exactly as it does for
			// too few. bindParams of ruby-interpreter.abnf and js_rargchk's fast
			// path ask the same question.
			if !strings.ContainsRune(kinds, '2') && npos > len(kinds) {
				rt.rubyArgCountErr(npos, kinds)
			}
			spare := npos - required
			ai := 0
			for k := 0; k < len(kinds); k++ {
				switch kinds[k] {
				case '2':
					after := 0
					for q := k + 1; q < len(kinds); q++ {
						if kinds[q] != '2' {
							after++
						}
					}
					take := npos - ai - after
					if take < 0 {
						take = 0
					}
					rest := &jsArray{}
					for r := 0; r < take; r++ {
						rest.elems = append(rest.elems, pos[ai])
						ai++
					}
					out.elems = append(out.elems, rest)
				case '1':
					if spare > 0 && ai < npos {
						out.elems = append(out.elems, pos[ai])
						ai++
						spare--
					} else {
						out.elems = append(out.elems, jsUndef)
					}
				default:
					if ai < npos {
						out.elems = append(out.elems, pos[ai])
					} else {
						out.elems = append(out.elems, jsNull)
					}
					ai++
				}
			}
			return w(out)
		},
		// The keyword arguments a call passed: the trailing Hash of the argument
		// list (a block closure may sit behind it), or an empty Hash.
		"js_rkwargs": func(a []uint64) uint64 {
			arr, ok := u(a[0]).(*jsArray)
			if ok {
				for i := len(arr.elems) - 1; i >= 0 && i >= len(arr.elems)-2; i-- {
					if _, _, isDict := dictParts(arr.elems[i]); isDict {
						return w(arr.elems[i])
					}
				}
			}
			return w(&jsObject{props: map[string]interface{}{
				"__dict": true, "keys": &jsArray{}, "vals": &jsArray{},
			}})
		},
		// **rest: the keyword hash minus the keys the named keyword parameters took.
		"js_rkwrest": func(a []uint64) uint64 {
			keys, vals, isDict := dictParts(u(a[0]))
			out := &jsObject{props: map[string]interface{}{
				"__dict": true, "keys": &jsArray{}, "vals": &jsArray{},
			}}
			if !isDict {
				return w(out)
			}
			taken, _ := u(a[1]).(*jsArray)
			okeys, ovals, _ := dictParts(out)
			for i := range keys.elems {
				drop := false
				if taken != nil {
					for _, t := range taken.elems {
						if rt.rubyEq(t, keys.elems[i]) {
							drop = true
							break
						}
					}
				}
				if !drop {
					dictAppend(okeys, ovals, keys.elems[i], vals.elems[i])
				}
			}
			return w(out)
		},
		// &blk: the block a call was given - the trailing callable of the argument
		// list - or nil when it was called without one.
		"js_rblock": func(a []uint64) uint64 {
			arr, ok := u(a[0]).(*jsArray)
			if ok && len(arr.elems) > 0 {
				if f, is := rubyBlkOf(arr.elems[len(arr.elems)-1]); is {
					return w(f)
				}
			}
			return w(jsNull)
		},
		// js_rblkwrap builds the block marker - see rubyBlkWrap. emitArgs of
		// ruby-to-llvm-ir.abnf calls it once per call site that was written with
		// a block, and nothing else produces the shape.
		"js_rblkwrap": func(a []uint64) uint64 {
			return w(rubyBlkWrap(u(a[0])))
		},
		// A lambda literal (->) / Kernel#lambda: an ordinary closure, remembered
		// here so `lambda?` can tell it from a proc (their argument arity and their
		// `return` differ in Ruby).
		// C.new(args): build an instance of a compiled class and run its initialize.
		// Two builtin classes are constructed directly: Proc.new { } is its block, and
		// an exception class makes the {__class, args} shape the rescue clauses read.
		"js_rnew": func(a []uint64) uint64 {
			cls, _ := u(a[0]).(*jsObject)
			args, _ := u(a[1]).(*jsArray)
			var argv []interface{}
			if args != nil {
				argv = args.elems
			}
			if cls != nil {
				if bn, isBuiltin := cls.props["__rbuiltin"].(string); isBuiltin {
					if bn == "Proc" {
						if _, blk := rubyBlockArgs(argv); blk != nil {
							return w(blk)
						}
						return w(jsNull)
					}
					// Array.new(n) / Array.new(n, v) / Array.new(n) { |i| }, ahead
					// of the exception construction below.
					if bn == "Array" {
						return w(rt.rubyArrNew(argv))
					}
					// Hash.new / Hash.new(v) / Hash.new { |h, k| }. The default
					// lives in the hidden __hdef / __hdefp slot and
					// rubyHashDefault honours it at every read.
					if bn == "Hash" {
						return w(rubyNewHash(argv))
					}
					if bn == "Regexp" {
						return w(rt.rubyRegexNew(argv))
					}
					return w(rt.rubyExc(cls, argv))
				}
			}
			inst := newJSObject()
			inst.set("__class", u(a[0]))
			rt.rubyMethod(inst, "initialize", argv)
			return w(inst)
		},
		// Kernel#format / sprintf: the argument array is [format, args...].
		"js_rformat": func(a []uint64) uint64 {
			arr, ok := u(a[0]).(*jsArray)
			if !ok {
				return rt.wrapStr("")
			}
			return rt.wrapStr(rt.rubyFormatArgs(arr.elems))
		},
		// Kernel#rand, as a call the emitter makes for the bare name.
		"js_rrand": func(a []uint64) uint64 {
			arr, ok := u(a[0]).(*jsArray)
			if !ok || len(arr.elems) == 0 {
				return w(rubyRand(nil))
			}
			return w(rubyRand(arr.elems[0]))
		},
		// &:sym as a block argument: the "call this method on each element" proc
		// (`[1, 2].map(&:to_s)`). A Proc is passed straight through.
		"js_rsymproc": func(a []uint64) uint64 {
			sym, isSym := u(a[0]).(jsSym)
			if !isSym {
				return a[0]
			}
			return w(rt.rubySymProc(sym.s))
		},
		// A BLOCK parameter read: Ruby auto-splats a single Array argument over a
		// block that declares more than one parameter, so `[[1, [2, 3]]].each { |a,
		// (b, c)| }` sees a=1 rather than the whole pair. A lambda is strict and uses
		// js_arg instead.
		"js_rblockarg": func(a []uint64) uint64 {
			arr, ok := u(a[0]).(*jsArray)
			if !ok {
				return jsHUndefined
			}
			i, n := int(a[1]), int(a[2])
			if n > 1 && len(arr.elems) == 1 {
				if inner, isArr := arr.elems[0].(*jsArray); isArr {
					if i < len(inner.elems) {
						return w(inner.elems[i])
					}
					return jsHUndefined
				}
			}
			if i < len(arr.elems) {
				return w(arr.elems[i])
			}
			return jsHUndefined
		},
		"js_rlambda": func(a []uint64) uint64 {
			if rt.rubyLambdas == nil {
				rt.rubyLambdas = map[interface{}]bool{}
			}
			rt.rubyLambdas[u(a[0])] = true
			return a[0]
		},
		// KERNEL#LAMBDA'S ARITY CHECK. A block binds loosely (a missing parameter
		// is nil, an extra argument is dropped) and a lambda does not, so
		// `lambda { |x| }` has to WRAP its block in a frame that raises MRI's
		// ArgumentError; `->(x) { }` gets the same check from emitParams instead,
		// because its body IS the checked function. `kinds` is js_rposbind's
		// encoding, recorded at the block LITERAL by the emitter - the arity
		// registry cannot supply it, being armed only when the program names
		// `arity`. The registry entry, if there is one, is re-registered onto the
		// wrapper so Proc#arity still answers.
		"js_rlamchk": func(a []uint64) uint64 {
			inner := u(a[0])
			if !isCallable(inner) {
				return a[0]
			}
			kinds, _ := u(a[1]).(string)
			wrap := jsHostFunc("lambda", func(rt *jsrt, this uint64, args []interface{}) interface{} {
				rt.rubyLamArity(args, kinds)
				return rt.call(inner, jsUndef, args)
			})
			if rt.rubyArities != nil {
				if rec, has := rt.rubyArities[inner]; has {
					rt.rubyArities[wrap] = rec
				}
			}
			return w(wrap)
		},
		// The parameter shape of the closure just built, for Proc#arity - the
		// closure itself is handed straight back, exactly as js_rlambda does.
		"js_rarity": func(a []uint64) uint64 {
			if rt.rubyArities == nil {
				rt.rubyArities = map[interface{}]rubyArity{}
			}
			rt.rubyArities[u(a[0])] = rubyArity{
				req:  int(rt.toNumber(u(a[1]))),
				opt:  int(rt.toNumber(u(a[2]))),
				star: rubyTruthy(u(a[3])),
			}
			return a[0]
		},
		// The count check of the FAST PATH. emitParams binds an all-required
		// parameter list with js_arg rather than js_rposbind, so that the common
		// case allocates no binding array - and that path therefore has to ask the
		// question separately. Answers nothing; it either raises or returns.
		"js_rargchk": func(a []uint64) uint64 {
			arr, ok := u(a[0]).(*jsArray)
			if !ok {
				return jsHUndefined
			}
			off, req := int(a[1]), int(a[2])
			npos := len(arr.elems) - off
			if npos < 0 {
				npos = 0
			}
			if npos > 0 && rubyIsBlkArg(arr.elems[len(arr.elems)-1]) {
				npos--
			}
			// TOO MANY is an error as well: this arm's parameter list is
			// all-required, so the count must be EXACTLY req.
			if npos != req {
				rt.rubyArgCountErr(npos, strings.Repeat("0", req))
			}
			return jsHUndefined
		},
		// Kernel#method(:name): the scope chain first - every def of the file is
		// declared there - and then an INSTANCE METHOD of the running `self`, bound
		// to it. The whole of it is one extern because js_scope_get ABORTS on a
		// miss and so cannot be used to ask a question; the scope is right here in
		// the twin, and layer 2 has scopeHas/scopeParent/scopeGet. Mirrors
		// hostGlobals["method"] of ruby-interpreter.abnf and js_rmethodval of
		// lib/ruby-rt.metajs.
		"js_rmethodval": func(a []uint64) uint64 {
			name := rt.toString(u(a[1]))
			var self interface{} = jsNull
			for sc := rt.scopeOf(a[0]); sc != nil; sc = sc.parent {
				// A def's own slot, which a local of the same name cannot overwrite
				// (curTopLocals in ruby-to-llvm-ir.abnf).
				if v, ok := sc.get("$d$" + name); ok && isCallable(v) {
					return w(rt.rubyMarkMethodVal(v, "Object", name))
				}
				if v, ok := sc.get(name); ok && isCallable(v) {
					// A top-level `def` is a scope variable holding its own
					// closure, so THAT is the Method value; MRI's owner for one
					// is Object.
					//
					// ...UNLESS THE HIT IS IN THE ROOT SCOPE AND THE NAME IS A
					// KERNEL ONE: the outermost scope is the HOST's and carries
					// the shared `print` / `printf` / `sprintf`, which are Go's
					// fmt and not Ruby's. Same reasoning, and the same three
					// names, as js_rmethodval in languages/lib/ruby-rt.metajs.
					if sc.parent != nil || rubyKernelMethodIndex(name) < 0 {
						return w(rt.rubyMarkMethodVal(v, "Object", name))
					}
				}
				if v, ok := sc.get("self"); ok && isUndefOrNull(self) {
					self = v
				}
			}
			if !isUndefOrNull(self) {
				if m, found := rubyFindMethod(self, name); found {
					recv, mname := self, name
					// A variadic wrapper that re-dispatches on the receiver, so
					// that #call, #arity and #to_proc all work on the answer.
					// rubyMethod rather than the stored closure, so a singleton
					// method and a prepended module resolve as an ordinary call.
					f := &hostFunc{name: "method:" + name, fn: func(rt *jsrt, this uint64, args []interface{}) interface{} {
						return rt.rubyMethod(recv, mname, args)
					}}
					if ar, ok := rt.rubyArities[m]; ok {
						if rt.rubyArities == nil {
							rt.rubyArities = map[interface{}]rubyArity{}
						}
						rt.rubyArities[f] = ar
					}
					return w(rt.rubyMarkMethodVal(f, rubyOwnerName(recv), mname))
				}
			}
			// A KERNEL method, which MRI hands out as an ordinary Method bound to
			// the main object - `#<Method: Object(Kernel)#puts>`. After the scope
			// walk and after rubyFindMethod, so a `def puts` of the program's own
			// still wins.
			if ki := rubyKernelMethodIndex(name); ki >= 0 {
				return w(rt.rubyKernelMethodVal(name, ki))
			}
			// A CATCHABLE NameError, which is what MRI raises (and what the
			// interpreter half prints).
			panic(&jsThrown{value: rt.rubyExc(rt.rubyBuiltinClass("NameError"),
				[]interface{}{"undefined method `" + name + "' for class `Object'"})})
		},
		"js_rgget": func(a []uint64) uint64 {
			if v, ok := rt.rubyGlobals[rt.toString(u(a[0]))]; ok {
				return w(v)
			}
			return w(jsNull)
		},
		"js_rgset": func(a []uint64) uint64 {
			if rt.rubyGlobals == nil {
				rt.rubyGlobals = map[string]interface{}{}
			}
			rt.rubyGlobals[rt.toString(u(a[0]))] = u(a[1])
			return 0
		},
		// defined?(X): the KIND of a name, or nil when it is nothing. The kind is
		// decided at compile time from the shape of the name; these two only test
		// whether it exists (as a $global, or anywhere on the scope chain).
		"js_rgdefined": func(a []uint64) uint64 {
			if _, ok := rt.rubyGlobals[rt.toString(u(a[0]))]; ok {
				return w(u(a[1]))
			}
			return w(jsNull)
		},
		// (scope, name, kind) -> kind | "method" | nil. A Ruby local is visible
		// up to the nearest scope declaring "$blk" - the method's own frame, or
		// the program's main scope - and never beyond it: not the top-level
		// locals from inside a def, nor the host's root scope (MetaJS's own
		// print/printf/sprintf). A name bound nowhere is "method" when self has
		// one. Twin of js_rdefined in languages/lib/ruby-rt.metajs.
		"js_rdefined": func(a []uint64) uint64 {
			name := rt.toString(u(a[1]))
			var self interface{} = jsNull
			for s := rt.scopeOf(a[0]); s != nil; s = s.parent {
				if s.has(name) {
					return w(u(a[2]))
				}
				if v, ok := s.get("self"); ok && isUndefOrNull(self) {
					self = v
				}
				if s.has("$blk") {
					break
				}
			}
			if _, found := rubyFindMethod(self, name); found && !isUndefOrNull(self) {
				return w("method")
			}
			return w(jsNull)
		},
		// (scope, "@x" | "@@x") -> "instance-variable" / "class variable" | nil:
		// defined only once ASSIGNED, on self or up its class chain. Twin of
		// js_rvardefined in languages/lib/ruby-rt.metajs.
		"js_rvardefined": func(a []uint64) uint64 {
			name := rt.toString(u(a[1]))
			var self interface{} = jsNull
			for s := rt.scopeOf(a[0]); s != nil; s = s.parent {
				if v, ok := s.get("self"); ok {
					self = v
					break
				}
			}
			if strings.HasPrefix(name, "@@") {
				for cls := rubyClassOfSelf(self); cls != nil; {
					o, ok := cls.(*jsObject)
					if !ok {
						break
					}
					if _, has := o.props[name]; has {
						return w("class variable")
					}
					cls = o.props["__super"]
				}
				return w(jsNull)
			}
			if o, ok := self.(*jsObject); ok {
				if _, has := o.props[name[1:]]; has {
					return w("instance-variable")
				}
			}
			return w(jsNull)
		},
		// A @@class variable belongs to the CLASS, shared by every instance and by
		// the subclasses: the lookup starts at the class of `self` (or at self when
		// it already IS a class) and walks __super.
		"js_rcvget": func(a []uint64) uint64 {
			for cls := rubyClassOfSelf(u(a[0])); cls != nil; {
				o, ok := cls.(*jsObject)
				if !ok {
					break
				}
				if v, has := o.props["@@"+rt.toString(u(a[1]))]; has {
					return w(v)
				}
				cls = o.props["__super"]
			}
			return w(jsNull)
		},
		"js_rcvset": func(a []uint64) uint64 {
			start := rubyClassOfSelf(u(a[0]))
			key := "@@" + rt.toString(u(a[1]))
			for cls := start; cls != nil; {
				o, ok := cls.(*jsObject)
				if !ok {
					break
				}
				if _, has := o.props[key]; has {
					o.set(key, u(a[2]))
					return 0
				}
				cls = o.props["__super"]
			}
			if o, ok := start.(*jsObject); ok {
				o.set(key, u(a[2]))
			}
			return 0
		},
		// The builtin class objects (Integer, String, Symbol, Array, ...): one object
		// per name, so `1.class == Integer` and `when Integer` compare identities.
		"js_rclass": func(a []uint64) uint64 { return w(rt.rubyBuiltinClass(rt.toString(u(a[0])))) },
		"js_rmain":  func(a []uint64) uint64 { return w(rt.rubyMainObj()) },
		// Ruby's to_s, used for string interpolation and puts.
		"js_rstr": func(a []uint64) uint64 { return rt.wrapStr(rt.rubyStr(u(a[0]))) },
		// Kernel#puts and Kernel#print with RUBY rendering (not Go's %v: `puts nil`
		// prints "", `puts 1.class` prints "Integer"). puts also FLATTENS an array
		// (one element per line) and prints nothing at all for an empty one, which
		// is MRI.
		"js_rputs": func(a []uint64) uint64 { rt.rubyPuts(u(a[0])); return 0 },
		"js_rprint": func(a []uint64) uint64 {
			fmt.Fprint(outWriter, wtf8Clean(rt.rubyStr(u(a[0]))))
			return 0
		},
		// Kernel#p: INSPECT, one argument per line, answering the argument (the
		// whole list when there is more than one, nil when there is none). It is
		// one extern rather than an emitted loop because that is what makes the
		// return value the same in both engines.
		"js_rp": func(a []uint64) uint64 {
			arr, ok := u(a[0]).(*jsArray)
			if !ok || len(arr.elems) == 0 {
				return w(jsNull)
			}
			for _, e := range arr.elems {
				fmt.Fprintln(outWriter, wtf8Clean(rt.rubyInspect(e)))
			}
			if len(arr.elems) == 1 {
				return w(arr.elems[0])
			}
			return w(&jsArray{elems: append([]interface{}{}, arr.elems...)})
		},
		// Kernel#Integer / #Float / #String / #Array, the conversion functions.
		"js_rkconv": func(a []uint64) uint64 {
			arr, ok := u(a[1]).(*jsArray)
			if !ok {
				arr = &jsArray{}
			}
			return w(rt.rubyKernelConv(rt.toString(u(a[0])), arr.elems))
		},
		// %W[..] / %I[..]: split an interpolated body on whitespace into an array of
		// strings (or of symbols when the flag is set).
		"js_rsplitw": func(a []uint64) uint64 {
			out := &jsArray{}
			asSym := rubyTruthy(u(a[1]))
			for _, word := range strings.Fields(rt.rubyStr(u(a[0]))) {
				if asSym {
					out.elems = append(out.elems, jsSym{s: word})
				} else {
					out.elems = append(out.elems, word)
				}
			}
			return w(out)
		},
		// ----- PHP's value model (see jsrtphp.go) -----
		// Only php-to-llvm-ir.abnf emits these. They give the compiler the value
		// model php-interpreter.abnf already has: a BOXED float (so 3.0 !== 3),
		// PHP truthiness ("0" is falsy), PHP's == / === / <=> and the one ordered
		// array type whose === compares the keys and their order.
		"js_phflo": func(a []uint64) uint64 { return w(phpMkFlo(rt.toNumber(u(a[0])))) },
		"js_phtruthy": func(a []uint64) uint64 {
			if rt.phpTest(u(a[0])) {
				return 1
			}
			return 0
		},
		"js_phbool": func(a []uint64) uint64 { return boolH(rt.phpTest(u(a[0]))) },
		"js_phnot":  func(a []uint64) uint64 { return boolH(!rt.phpTest(u(a[0]))) },
		"js_phstr":  func(a []uint64) uint64 { return rt.wrapStr(rt.phpStr(u(a[0]))) },
		"js_phcat":  func(a []uint64) uint64 { return rt.wrapStr(rt.phpStr(u(a[0])) + rt.phpStr(u(a[1]))) },
		"js_pheq":   func(a []uint64) uint64 { return boolH(rt.phpLoose(u(a[0]), u(a[1]))) },
		"js_phne":   func(a []uint64) uint64 { return boolH(!rt.phpLoose(u(a[0]), u(a[1]))) },
		"js_phseq":  func(a []uint64) uint64 { return boolH(rt.phpIdentical(u(a[0]), u(a[1]))) },
		"js_phsne":  func(a []uint64) uint64 { return boolH(!rt.phpIdentical(u(a[0]), u(a[1]))) },
		"js_phcmp":  func(a []uint64) uint64 { return rt.wrapNum(rt.phpCmp(u(a[0]), u(a[1]))) },
		"js_phlt":   func(a []uint64) uint64 { return boolH(rt.phpCmp(u(a[0]), u(a[1])) < 0) },
		"js_phgt":   func(a []uint64) uint64 { return boolH(rt.phpCmp(u(a[0]), u(a[1])) > 0) },
		"js_phle":   func(a []uint64) uint64 { return boolH(rt.phpCmp(u(a[0]), u(a[1])) <= 0) },
		"js_phge":   func(a []uint64) uint64 { return boolH(rt.phpCmp(u(a[0]), u(a[1])) >= 0) },
		"js_phadd":  func(a []uint64) uint64 { return w(rt.phpArith("+", u(a[0]), u(a[1]))) },
		"js_phsub":  func(a []uint64) uint64 { return w(rt.phpArith("-", u(a[0]), u(a[1]))) },
		"js_phmul":  func(a []uint64) uint64 { return w(rt.phpArith("*", u(a[0]), u(a[1]))) },
		"js_phdiv":  func(a []uint64) uint64 { return w(rt.phpArith("/", u(a[0]), u(a[1]))) },
		"js_phmod":  func(a []uint64) uint64 { return w(rt.phpArith("%", u(a[0]), u(a[1]))) },
		"js_phpow":  func(a []uint64) uint64 { return w(rt.phpArith("**", u(a[0]), u(a[1]))) },
		"js_phneg":  func(a []uint64) uint64 { return w(rt.phpArith("-", float64(0), u(a[0]))) },
		"js_phband": func(a []uint64) uint64 { return w(rt.phpBit("&", u(a[0]), u(a[1]))) },
		"js_phbor":  func(a []uint64) uint64 { return w(rt.phpBit("|", u(a[0]), u(a[1]))) },
		"js_phbxor": func(a []uint64) uint64 { return w(rt.phpBit("^", u(a[0]), u(a[1]))) },
		"js_phshl":  func(a []uint64) uint64 { return w(rt.phpBit("<<", u(a[0]), u(a[1]))) },
		"js_phshr":  func(a []uint64) uint64 { return w(rt.phpBit(">>", u(a[0]), u(a[1]))) },
		"js_phbnot": func(a []uint64) uint64 {
			return rt.wrapNum(float64(^int32(int64(phpTrunc(phpNumOf(rt.phpToNum(u(a[0]))))))))
		},
		"js_phnum":  func(a []uint64) uint64 { return w(rt.phpToNum(u(a[0]))) },
		"js_phcast": func(a []uint64) uint64 { return w(rt.phpCast(rt.toString(u(a[0])), u(a[1]))) },
		// The one array type: an ordered key->value map with __next, the next
		// automatic integer key. The shape is the runtime's __dict, so js_range_len /
		// js_range_key / js_range_val iterate it unchanged.
		"js_pharr": func(a []uint64) uint64 { return w(phpNewArr()) },
		"js_phset": func(a []uint64) uint64 {
			rt.phpArrSet(u(a[0]), u(a[1]), u(a[2]))
			return 0
		},
		"js_phpush": func(a []uint64) uint64 {
			rt.phpArrPush(u(a[0]), u(a[1]))
			return 0
		},
		"js_phget":   func(a []uint64) uint64 { return w(rt.phpArrGet(u(a[0]), u(a[1]))) },
		"js_phisset": func(a []uint64) uint64 { return boolH(rt.phpArrHas(u(a[0]), u(a[1]))) },
		// The storage cell of an array element, created on demand and REUSED: two
		// references to the same element must share one cell (php-interpreter.abnf's
		// arrCell). js_phget dereferences, so the compiler cannot read the raw slot
		// back and find the cell itself.
		"js_phcell": func(a []uint64) uint64 { return w(rt.phpElemCell(u(a[0]), u(a[1]))) },
		// Read THROUGH a PHP reference cell: the value the cell holds, or the value
		// itself when it is not one (see the reference model in jsrtphp.go). Every
		// variable read in a program that writes a '&' anywhere passes through here,
		// which is why it is one call rather than a branch at each read site.
		"js_phderef": func(a []uint64) uint64 { return w(phpDeref(u(a[0]))) },
		"js_phunset": func(a []uint64) uint64 {
			rt.phpArrUnset(u(a[0]), u(a[1]))
			return 0
		},
		"js_phmerge": func(a []uint64) uint64 { // [...$a]: integer keys renumber, string keys stay
			if keys, vals, ok := phpArrParts(u(a[1])); ok {
				for i := range keys.elems {
					if _, isNum := keys.elems[i].(float64); isNum {
						rt.phpArrPush(u(a[0]), vals.elems[i])
					} else {
						rt.phpArrSet(u(a[0]), keys.elems[i], vals.elems[i])
					}
				}
			}
			return 0
		},
		"js_phcopy": func(a []uint64) uint64 { return w(rt.phpArrCopy(u(a[0]))) },
		"js_phlen":  func(a []uint64) uint64 { return rt.wrapNum(rt.phpLen(u(a[0]))) },
		"js_phkeys": func(a []uint64) uint64 { return w(rt.phpKeysOf(u(a[0]))) },
		"js_phvals": func(a []uint64) uint64 { return w(rt.phpValsOf(u(a[0]))) },
		"js_phin":   func(a []uint64) uint64 { return boolH(rt.phpInArray(u(a[0]), u(a[1]))) },
		// ----- classes: '::' constants, static properties and static dispatch -----
		"js_phconst": func(a []uint64) uint64 { return w(rt.phpConst(u(a[0]), rt.toString(u(a[1])))) },
		"js_phsget":  func(a []uint64) uint64 { return w(rt.phpStaticGet(u(a[0]), rt.toString(u(a[1])))) },
		"js_phsset": func(a []uint64) uint64 {
			rt.phpStaticSet(u(a[0]), rt.toString(u(a[1])), u(a[2]))
			return 0
		},
		"js_phcname": func(a []uint64) uint64 { // C::class / $o::class
			if s, isStr := u(a[0]).(string); isStr {
				return rt.wrapStr(s)
			}
			return w(rt.phpClassOf(u(a[0])).props["__name"])
		},
		// instanceof. The right side is a NAME, but it may also arrive as a class
		// descriptor or an instance ($x instanceof $cn), so resolve it first.
		"js_phclone": func(a []uint64) uint64 { return w(rt.phpClone(u(a[0]))) },
		// Does the caught value match one of a catch clause's types? \Throwable
		// catches everything, exactly as php-interpreter.abnf's excMatches has it.
		"js_phcatch": func(a []uint64) uint64 {
			if _, vals, ok := phpArrParts(u(a[1])); ok {
				for _, t := range vals.elems {
					n := rt.phpStr(t)
					if n == "Throwable" || rt.phpIsA(u(a[0]), n) {
						return boolH(true)
					}
				}
			}
			return boolH(false)
		},
		// A class descriptor by name, or null when nothing of that name is declared
		// yet - an interface list must not abort on a forward reference.
		"js_phclslookup": func(a []uint64) uint64 {
			name := rt.toString(u(a[1]))
			for sc := rt.scopeOf(a[0]); sc != nil; sc = sc.parent {
				if v, ok := sc.get(name); ok {
					return w(v)
				}
			}
			return jsHNull
		},
		"js_phisa": func(a []uint64) uint64 {
			name := ""
			if o, ok := u(a[1]).(*jsObject); ok {
				name = rt.phpStr(rt.phpClassOf(o).props["__name"])
			} else {
				name = rt.phpStr(u(a[1]))
			}
			return boolH(rt.phpIsA(u(a[0]), name))
		},
		// $this inside a method, or undefined in a static one. A plain js_scope_get
		// would abort on the name; this probe answers undefined instead.
		"js_phthis": func(a []uint64) uint64 {
			for sc := rt.scopeOf(a[0]); sc != nil; sc = sc.parent {
				if v, ok := sc.get("$this"); ok {
					return w(v)
				}
			}
			return jsHUndefined
		},
		// C::m($this, args): the method is resolved through the __super chain, and
		// the class the CALL named becomes the late-static-binding class for the
		// duration - which is what makes `static::who()` inside an inherited static
		// method dispatch to the subclass.
		"js_phscall": func(a []uint64) uint64 {
			cls := rt.phpClassOf(u(a[0]))
			name := rt.toString(u(a[1]))
			owner := phpOwner(cls, name)
			if owner == nil {
				// __callStatic($name, $args) is PHP's fallback for a missing
				// static method, exactly as __call is for an instance method.
				if cs := phpOwner(cls, "__callStatic"); cs != nil {
					args := &jsArray{}
					if _, vals, ok := phpArrParts(u(a[3])); ok {
						args.elems = append(args.elems, vals.elems...)
					}
					saved := rt.phpLsb
					rt.phpLsb = cls
					defer func() { rt.phpLsb = saved }()
					return w(rt.call(cs.props["__callStatic"], jsUndef,
						[]interface{}{u(a[2]), name, args}))
				}
				rt.fail("undefined method %s::%s", rt.phpStr(cls.props["__name"]), name)
			}
			args := []interface{}{u(a[2])}
			if _, vals, ok := phpArrParts(u(a[3])); ok {
				args = append(args, vals.elems...)
			}
			saved := rt.phpLsb
			rt.phpLsb = cls
			defer func() { rt.phpLsb = saved }()
			return w(rt.call(owner.props[name], jsUndef, args))
		},
		// The 'static::' class inside a method body: the runtime class of $this when
		// there is one, else the class the static call named, else 'self'.
		"js_phlsb": func(a []uint64) uint64 {
			if o, ok := u(a[0]).(*jsObject); ok {
				if cls, has := o.props["__class"]; has {
					return w(cls)
				}
			}
			if rt.phpLsb != nil {
				return w(rt.phpLsb)
			}
			return a[1]
		},
		// The few builtins whose result must keep PHP's int/float distinction:
		// abs(-3.5) is a float, intdiv() truncates, max/min pick with PHP's own
		// comparison and hand the ORIGINAL value back.
		"js_phintdiv": func(a []uint64) uint64 {
			y := phpTrunc(phpNumOf(rt.phpToNum(u(a[1]))))
			if y == 0 {
				rt.fail("division by zero")
			}
			return rt.wrapNum(phpTrunc(phpTrunc(phpNumOf(rt.phpToNum(u(a[0])))) / y))
		},
		"js_phabs": func(a []uint64) uint64 {
			n := rt.phpToNum(u(a[0]))
			if phpNumOf(n) < 0 {
				return w(rt.phpArith("-", float64(0), n))
			}
			return w(n)
		},
		"js_phmax": func(a []uint64) uint64 {
			if rt.phpCmp(u(a[0]), u(a[1])) > 0 {
				return a[0]
			}
			return a[1]
		},
		"js_phmin": func(a []uint64) uint64 {
			if rt.phpCmp(u(a[0]), u(a[1])) < 0 {
				return a[0]
			}
			return a[1]
		},
		// ----- Kotlin's Char (see the jsChar type) -----
		// A Char from its code. The Kotlin grammars' char literals compile to this.
		"js_char": func(a []uint64) uint64 { return w(jsChar{code: int32(int64(a[0]))}) },
		// The code of a Char, and the identity for everything else - so a caller can
		// drive a counter with it without knowing whether it holds a Char.
		"js_char_code": func(a []uint64) uint64 { return rt.wrapNum(rt.toNumber(u(a[0]))) },
		// Box `v` back into a Char when `model` is one, else hand `v` back unchanged.
		// This is how a range keeps its element type: 'a'..'c' counts numerically and
		// re-boxes each element, so `for (ch in 'a'..'c')` yields Chars while
		// `for (i in 1..3)` yields Ints, from one emitted loop.
		"js_char_like": func(a []uint64) uint64 {
			if _, isChar := u(a[0]).(jsChar); isChar {
				return w(jsChar{code: int32(int64(rt.toNumber(u(a[1]))))})
			}
			return a[1]
		},
		// Kotlin's indexed read s[i]: a String yields the Char at that index (Kotlin's
		// CharSequence.get), everything else reads the member like js_get. A separate
		// external because plain js_get must keep yielding a one-character STRING for
		// JS, Python and the other languages that index strings.
		"js_kindex": func(a []uint64) uint64 {
			if s, isStr := u(a[0]).(string); isStr {
				i := jsToInt(rt.toNumber(u(a[1])))
				ch := rt.strAt(s, i)
				if ch == "" {
					rt.fail("string index %d out of range for %q", i, s)
				}
				return w(jsChar{code: int32([]rune(ch)[0])})
			}
			return w(rt.getMember(u(a[0]), u(a[1])))
		},
		// Java's == when a char may be involved. Java's char reuses the jsChar type
		// (so it does arithmetic and compares as its code), but String.charAt keeps
		// answering a one-character String - the Java subset has no char-typed
		// expression results - so a Char must compare equal to that String too.
		// Without a jsChar operand this is exactly js_seq, and only the Java compiler
		// grammar emits it, so no other language can reach the extra branch.
		"js_jchareq": func(a []uint64) uint64 {
			x, y := u(a[0]), u(a[1])
			if c, ok := x.(jsChar); ok {
				if s, isS := y.(string); isS {
					r := []rune(s)
					return boolH(len(r) == 1 && int32(r[0]) == c.code)
				}
			}
			if c, ok := y.(jsChar); ok {
				if s, isS := x.(string); isS {
					r := []rune(s)
					return boolH(len(r) == 1 && int32(r[0]) == c.code)
				}
			}
			return boolH(rt.strictEq(x, y))
		},
		// ----- Java's / Kotlin's boxed double (see the jsJFlo type in jsrtjvm.go) -----
		// A double from any numeric value: the (double)/(float) cast, a floating
		// point literal and a `double`-declared variable's initializer.
		"js_jflo": func(a []uint64) uint64 { return w(jvmMkFlo(rt.toNumber(u(a[0])))) },
		// ----- C#: the two places its value semantics differ from Java's -----
		// C#'s '+': a null operand renders as the EMPTY string, not as "null"
		// (String.Concat(object) calls ToString() only on a non-null argument,
		// ECMA-334 12.4.7 / .NET String.Concat), where Java spells it "null".
		// Otherwise it is js_jadd: string concat, 32 bit add for two ints, float add
		// when a double is involved.
		"js_csadd": func(a []uint64) uint64 {
			l, r := u(a[0]), u(a[1])
			_, ls := l.(string)
			_, rs := r.(string)
			if ls || rs {
				return rt.wrapStr(strConcat(rt.csString(l), rt.csString(r)))
			}
			if jvmIsFlo(l) || jvmIsFlo(r) {
				return w(jsJFlo{f: rt.toNumber(l) + rt.toNumber(r), sty: jvmStyleOf(l, r)})
			}
			ln, rn := rt.toNumber(l), rt.toNumber(r)
			if isInt32Value(ln) && isInt32Value(rn) {
				return rt.wrapNum(float64(int32(int64(ln) + int64(rn))))
			}
			return rt.wrapNum(ln + rn)
		},
		// C#'s .Length / .Count. A string's Length counts UTF-16 CODE UNITS
		// (System.String is a UTF-16 sequence, so "😀".Length is 2), where
		// js_pylen counts code points because Python's len() does. Everything else
		// is js_pylen.
		"js_cslen": func(a []uint64) uint64 {
			switch o := u(a[0]).(type) {
			case string:
				return rt.wrapNum(float64(len(utf16.Encode([]rune(o)))))
			case *jsArray:
				return rt.wrapNum(float64(len(o.elems)))
			}
			if keys, _, ok := dictParts(u(a[0])); ok {
				return rt.wrapNum(float64(len(keys.elems)))
			}
			rt.fail("Length/Count of a %s", rt.typeOf(u(a[0])))
			return 0
		},
		// The same box in Go's / C#'s print style (see jsrtjvm.go).
		"js_gflo": func(a []uint64) uint64 { return w(jsJFlo{f: rt.toNumber(u(a[0])), sty: floGo}) },
		// Go's float32: the same box at the 32 BIT WIDTH, so `float32(x)`, a
		// `var f float32` slot and a float32 struct field all round to 24
		// significant bits here and stay rounded through every operator
		// (jvmIs32 in jsrtjvm.go). Its layer-2 twin is js_gflo32 in
		// languages/lib/go-rt.metajs and its interpreter twin gFlo32 in
		// languages/go-interpreter.abnf.
		"js_gflo32": func(a []uint64) uint64 {
			return w(jsJFlo{f: jvmFround(rt.toNumber(u(a[0]))), sty: floGoF})
		},
		// The WIDTH applied to a binary operator's RESULT: (op, result, l, r).
		// See goF32Fix in abnf/jsrtgolang.go for why it is a post-fix and not a
		// replacement, and languages/lib/go-rt.metajs for its native twin.
		"js_gf32fix": func(a []uint64) uint64 { return rt.goF32Fix(u(a[0]), u(a[1]), u(a[2]), u(a[3])) },
		// Go's own complex division - Smith's algorithm plus the C99 G.5.1 fixups,
		// which is what the gc runtime does and not the textbook formula. See
		// goCxDiv in abnf/jsrtgolang.go. It answers a plain {re, im} object because
		// the $cx class descriptor lives in the emitted module's scope.
		"js_gocxdiv": func(a []uint64) uint64 {
			re, im := goCxDiv(rt.toNumber(u(a[0])), rt.toNumber(u(a[1])),
				rt.toNumber(u(a[2])), rt.toNumber(u(a[3])))
			o := newJSObject()
			o.set("re", re)
			o.set("im", im)
			return w(o)
		},
		"js_csflo": func(a []uint64) uint64 { return w(jsJFlo{f: rt.toNumber(u(a[0])), sty: floCS}) },
		// The integral casts ((int), (long), (short), (byte)): truncate towards
		// zero and wrap to 32 bits, which is what the compiler emitted before.
		"js_jfint": func(a []uint64) uint64 { return rt.wrapNum(float64(rt.toInt32(u(a[0])))) },
		"js_jfsub": func(a []uint64) uint64 { return w(rt.jvmArith('-', u(a[0]), u(a[1]))) },
		"js_jfmul": func(a []uint64) uint64 { return w(rt.jvmArith('*', u(a[0]), u(a[1]))) },
		"js_jfdiv": func(a []uint64) uint64 { return w(rt.jvmArith('/', u(a[0]), u(a[1]))) },
		"js_jfmod": func(a []uint64) uint64 { return w(rt.jvmArith('%', u(a[0]), u(a[1]))) },
		// -x keeps the operand's type: a double negates as a double (so -0.0 and
		// -1.0/0.0 are real), everything else negates as an int32.
		"js_jfneg": func(a []uint64) uint64 {
			if f, ok := u(a[0]).(jsJFlo); ok {
				return w(jsJFlo{f: -f.f, sty: f.sty})
			}
			return rt.wrapNum(float64(rt.toInt32(-rt.toNumber(u(a[0])))))
		},
		// ++ / -- keep the type of what they step, like the interpreter's jStep: a
		// double stays a double, a char stays a char, everything else is an int32.
		"js_jfstep": func(a []uint64) uint64 {
			if c, ok := u(a[0]).(jsChar); ok {
				return w(jsChar{code: int32((int64(c.code) + int64(rt.toNumber(u(a[1])))) & 65535)})
			}
			return w(rt.jvmArith('+', u(a[0]), u(a[1])))
		},
		// Java's String.charAt: a CHAR, not a one-character String - so
		// s.charAt(1) + 0 is 101 and not "e0" (JLS 4.2.1, char is an integral
		// type). Only the Java compiler grammar emits it, and only for the name
		// charAt; a user class that declares its own charAt still dispatches
		// through memberCall.
		"js_jcharat": func(a []uint64) uint64 { // (target, args array)
			args, ok := u(a[1]).(*jsArray)
			if !ok {
				rt.fail("js_jcharat args must be an array")
			}
			if str, isStr := u(a[0]).(string); isStr {
				i := jsToInt(rt.toNumber(argAt(args.elems, 0)))
				units := utf16.Encode([]rune(str))
				if i < 0 || i >= len(units) {
					rt.fail("string index %d out of range for %q", i, str)
				}
				return w(jsChar{code: int32(units[i])})
			}
			return w(rt.memberCall(u(a[0]), "charAt", args.elems))
		},
		// java.lang.String.valueOf for the printable subset.
		"js_jfstr": func(a []uint64) uint64 { return rt.wrapStr(rt.javaString(u(a[0]))) },
		// Define a getter/setter property: (obj, key, getter|undef, setter|undef). Two
		// calls for the same key merge, so 'get x' and 'set x' of one accessor pair meet
		// on one record.
		"js_defprop": func(a []uint64) uint64 {
			o, ok := u(a[0]).(*jsObject)
			if !ok {
				rt.fail("js_defprop needs an object")
			}
			key := rt.toString(u(a[1]))
			acc, _ := o.props[key].(*jsAccessor)
			if acc == nil {
				acc = &jsAccessor{}
			}
			if g := u(a[2]); isCallable(g) {
				acc.get = g
			}
			if st := u(a[3]); isCallable(st) {
				acc.set = st
			}
			o.set(key, acc)
			rt.accessorCount++
			return 0
		},

		// Control-flow signals for a return/break/continue that leaves a try body.
		"js_ctl_return":   func(a []uint64) uint64 { return w(&jsCtl{kind: 1, value: u(a[0])}) },
		"js_ctl_break":    func(a []uint64) uint64 { return w(&jsCtl{kind: 2}) },
		"js_ctl_continue": func(a []uint64) uint64 { return w(&jsCtl{kind: 3}) },
		"js_ctl_kind": func(a []uint64) uint64 { // 1/2/3 for a signal, 0 for an ordinary value
			if c, ok := u(a[0]).(*jsCtl); ok {
				return uint64(c.kind)
			}
			return 0
		},
		"js_ctl_value": func(a []uint64) uint64 {
			if c, ok := u(a[0]).(*jsCtl); ok {
				return w(c.value)
			}
			return jsHUndefined
		},

		// The Python dialect externals.
		"js_fdiv": func(a []uint64) uint64 { // Floor division (Python //).
			return rt.wrapNum(math.Floor(rt.toNumber(u(a[0])) / rt.toNumber(u(a[1]))))
		},
		"js_fmod": func(a []uint64) uint64 { // Floor modulo (Python %): -7 % 2 == 1.
			x, y := rt.toNumber(u(a[0])), rt.toNumber(u(a[1]))
			r := math.Mod(x, y)
			if r != 0 && (r < 0) != (y < 0) {
				r += y
			}
			return rt.wrapNum(r)
		},
		"js_pytruthy": func(a []uint64) uint64 { // Python truthiness: empty lists/dicts/sets are falsy.
			if els, ok := pySetElems(u(a[0])); ok {
				if len(els.elems) > 0 {
					return 1
				}
				return 0
			}
			if arr, ok := u(a[0]).(*jsArray); ok {
				if len(arr.elems) > 0 {
					return 1
				}
				return 0
			}
			if keys, _, ok := dictParts(u(a[0])); ok {
				if len(keys.elems) > 0 {
					return 1
				}
				return 0
			}
			if _, cls, ok := pyInstance(u(a[0])); ok {
				if rt.pyUserTruthy(cls, u(a[0])) {
					return 1
				}
				return 0
			}
			if rt.truthy(u(a[0])) {
				return 1
			}
			return 0
		},
		"js_pyget": func(a []uint64) uint64 { // Sequence indexing (negative wraps) and dict lookup.
			if _, cls, isInst := pyInstance(u(a[0])); isInst { // A user class may define __getitem__.
				if m, found := pyLookup(cls, "__getitem__"); found && isCallable(m) {
					return w(rt.call(m, jsUndef, []interface{}{u(a[0]), u(a[1])}))
				}
			}
			// A subscript MISS is a CATCHABLE exception, not an abort, so
			// `try: d[k] / except KeyError` works. CPython's message for the
			// sequence cases carries no index, so neither does this one.
			if keys, vals, ok := dictParts(u(a[0])); ok {
				i := rt.pyDictFind(keys, u(a[1]))
				if i < 0 {
					panic(&jsThrown{value: rt.pyExcInstanceV("KeyError", u(a[1]))})
				}
				return w(vals.elems[i])
			}
			idx := 0
			switch o := u(a[0]).(type) {
			case *jsArray:
				idx = rt.pySeqIdx(o, u(a[1]))
				if idx < 0 {
					idx += len(o.elems)
				}
				if idx < 0 || idx >= len(o.elems) {
					panic(&jsThrown{value: rt.pyExcInstance("IndexError", "list index out of range")})
				}
				return w(o.elems[idx])
			case string:
				idx = rt.pySeqIdx(o, u(a[1]))
				n := rt.strLen(o)
				if idx < 0 {
					idx += n
				}
				if idx < 0 || idx >= n {
					panic(&jsThrown{value: rt.pyExcInstance("IndexError", "string index out of range")})
				}
				return rt.wrapStr(rt.strAt(o, idx))
			}
			// A ruby Range. THIS ARM AND js_pylen'S CARRY EVERY SEQUENCE USE OF A
			// RANGE THERE IS: `for x in a..b', `[*r]', `a, b, c = *r' and every
			// destructure lower to js_pylen once and js_pyget per element - the
			// emitter has no separate range path, and a miss here is silent (the
			// loop simply runs zero times). Only ruby builds a __rrange object.
			if r, isRng := rubyIsRange(u(a[0])); isRng {
				return w(rt.rubyRangeAt(r, int(rt.toNumber(u(a[1])))))
			}
			// list[int] / dict[str, int]: subscripting a TYPE (a class object or a
			// builtin like list) is a generic alias, not an indexing. One object per
			// (origin, parameter) pair, so 'list[int] == list[int]' holds.
			if v, ok := rt.pyGenericAlias(u(a[0]), u(a[1])); ok {
				return w(v)
			}
			// CPython's TypeError, and CATCHABLE: `None[0]` runs the except clause.
			panic(&jsThrown{value: rt.pyExcInstance("TypeError",
				"'"+pyTypeName(rt, u(a[0]))+"' object is not subscriptable")})
		},
		"js_pyset": func(a []uint64) uint64 { // List element (negative wraps) or dict entry assignment.
			if _, cls, isInst := pyInstance(u(a[0])); isInst { // A user class may define __setitem__.
				if m, found := pyLookup(cls, "__setitem__"); found && isCallable(m) {
					rt.call(m, jsUndef, []interface{}{u(a[0]), u(a[1]), u(a[2])})
					return 0
				}
			}
			if keys, vals, ok := dictParts(u(a[0])); ok {
				if i := rt.pyDictFind(keys, u(a[1])); i >= 0 {
					vals.elems[i] = u(a[2])
				} else {
					dictAppend(keys, vals, u(a[1]), u(a[2]))
				}
				return 0
			}
			arr, ok := u(a[0]).(*jsArray)
			if !ok {
				rt.pyNoItemOp(u(a[0]), false)
			}
			idx := rt.pySeqIdx(arr, u(a[1]))
			if idx < 0 {
				idx += len(arr.elems)
			}
			if idx < 0 || idx >= len(arr.elems) {
				panic(&jsThrown{value: rt.pyExcInstance("IndexError", "list assignment index out of range")})
			}
			arr.dropIdx()
			arr.elems[idx] = u(a[2])
			return 0
		},
		"js_dict_new": func(a []uint64) uint64 { // An empty Python dict / Go map handle.
			return w(&jsObject{props: map[string]interface{}{
				"__dict": true, "keys": &jsArray{}, "vals": &jsArray{},
			}})
		},
		"js_map_get": func(a []uint64) uint64 { // Go indexing: maps read their zero value for
			if keys, vals, ok := dictParts(u(a[0])); ok { // missing keys, slices their element.
				if i := rt.dictFind(keys, u(a[1])); i >= 0 {
					return w(vals.elems[i])
				}
				if z, has := u(a[0]).(*jsObject).props["zero"]; has {
					return w(z)
				}
				return w(jsUndef)
			}
			if arr, ok := u(a[0]).(*jsArray); ok {
				i := int(rt.toNumber(u(a[1])))
				if i < 0 || i >= len(arr.elems) {
					goIdxFail(i, len(arr.elems))
				}
				return w(arr.elems[i])
			}
			if str, ok := u(a[0]).(string); ok { // s[i] is the i-th BYTE of a Go string
				i := int(rt.toNumber(u(a[1])))
				if i < 0 || i >= len(str) {
					goIdxFail(i, len(str))
				}
				return rt.wrapNum(float64(str[i]))
			}
			// f[int] / Stack[K, V]: a generic instantiation. Generics are ERASED in the
			// Go grammars, so instantiating is the identity on the function it indexes.
			if isCallable(u(a[0])) {
				return a[0]
			}
			rt.fail("indexing a %s", rt.typeOf(u(a[0])))
			return 0
		},
		"js_map_del": func(a []uint64) uint64 { // delete(m, k); missing keys are a no-op.
			keys, vals, ok := dictParts(u(a[0]))
			if !ok {
				rt.fail("delete() needs a map")
			}
			if i := rt.dictFind(keys, u(a[1])); i >= 0 {
				keys.dropIdx()
				keys.elems = append(keys.elems[:i], keys.elems[i+1:]...)
				vals.elems = append(vals.elems[:i], vals.elems[i+1:]...)
			}
			return 0
		},
		"js_range_len": func(a []uint64) uint64 { // The Go range bound: an int is its own bound,
			switch o := u(a[0]).(type) { // otherwise the element/entry count.
			case float64:
				return rt.wrapNum(o)
			case string:
				return rt.wrapNum(float64(rt.strLen(o)))
			case *jsArray:
				return rt.wrapNum(float64(len(o.elems)))
			}
			if keys, _, ok := dictParts(u(a[0])); ok {
				return rt.wrapNum(float64(len(keys.elems)))
			}
			rt.fail("range over a %s", rt.typeOf(u(a[0])))
			return 0
		},
		"js_range_key": func(a []uint64) uint64 { // The i-th range key: maps yield their key,
			if keys, _, ok := dictParts(u(a[0])); ok { // everything else the index itself.
				return w(keys.elems[int(rt.toNumber(u(a[1])))])
			}
			return a[1]
		},
		"js_range_val": func(a []uint64) uint64 { // The i-th range value (maps by entry, lists by index).
			if _, vals, ok := dictParts(u(a[0])); ok {
				return w(vals.elems[int(rt.toNumber(u(a[1])))])
			}
			if arr, ok := u(a[0]).(*jsArray); ok {
				i := int(rt.toNumber(u(a[1])))
				if i >= 0 && i < len(arr.elems) {
					return w(arr.elems[i])
				}
			}
			return w(jsUndef)
		},
		// ----- Go builtins with no Python/JS equivalent (go-to-llvm-ir.abnf) -----
		// len() on a Go string counts BYTES - len("héllo") is 6, not 5 and not
		// the UTF-16 count; on a slice or map it is the element / entry count.
		"js_golen": func(a []uint64) uint64 {
			switch o := u(a[0]).(type) {
			case string:
				return rt.wrapNum(float64(len(o)))
			case *jsArray:
				return rt.wrapNum(float64(len(o.elems)))
			case jsUndefT, jsNullT: // a nil slice / map has length 0
				return rt.wrapNum(0)
			}
			if keys, _, ok := dictParts(u(a[0])); ok {
				return rt.wrapNum(float64(len(keys.elems)))
			}
			rt.fail("len() of a %s", rt.typeOf(u(a[0])))
			return 0
		},
		// `for i, ch := range s` walks a Go string by RUNE: i is the rune's BYTE
		// offset and ch its code point (an int32), NOT a one-character substring.
		// Normalizing the subject into a keys/vals dict lets the single emitted
		// range loop keep reading it through js_range_len/key/val. Anything else
		// (int bound, slice, map) passes through unchanged.
		"js_gorange": func(a []uint64) uint64 {
			// Ranging a CHANNEL yields one received value per iteration and ends when
			// the channel is closed and drained. Under the cooperative model every
			// producer runs to completion inside the first receive, so the sequence is
			// fully known here: drain it into the same keys/vals shape a map uses, with
			// the received value as BOTH (the single range name is the value in Go).
			if _, _, isChan := goChanParts(u(a[0])); isChan {
				keys := &jsArray{}
				vals := &jsArray{}
				for {
					v, ok := rt.goChanRecv(u(a[0]))
					if !ok {
						break
					}
					keys.elems = append(keys.elems, v)
					vals.elems = append(vals.elems, v)
				}
				return w(&jsObject{props: map[string]interface{}{
					"__dict": true, "keys": keys, "vals": vals,
				}})
			}
			s, isStr := u(a[0]).(string)
			if !isStr {
				return a[0]
			}
			keys := &jsArray{}
			vals := &jsArray{}
			for i, r := range s {
				keys.elems = append(keys.elems, float64(i))
				vals.elems = append(vals.elems, float64(r))
			}
			return w(&jsObject{props: map[string]interface{}{
				"__dict": true, "keys": keys, "vals": vals,
			}})
		},
		// The variadic Go builtins, taking the callee's whole argument array: append
		// starts a new slice from a nil one (append(m[k], v) where m[k] is missing),
		// and min/max/clear are the Go 1.21 builtins.
		"js_goappend": func(a []uint64) uint64 {
			args, ok := u(a[0]).(*jsArray)
			if !ok || len(args.elems) == 0 {
				rt.fail("append() needs a slice")
			}
			arr, isArr := args.elems[0].(*jsArray)
			if !isArr {
				arr = &jsArray{} // a nil slice: append starts a new one
			}
			for _, v := range args.elems[1:] {
				arr.dropIdx()
				arr.elems = append(arr.elems, v)
			}
			return w(arr)
		},
		"js_gominmax": func(a []uint64) uint64 {
			args, ok := u(a[0]).(*jsArray)
			if !ok || len(args.elems) == 0 {
				rt.fail("min()/max() needs an argument")
			}
			wantMax := rt.truthy(u(a[1]))
			best := args.elems[0]
			for _, v := range args.elems[1:] {
				less := false
				if bs, ok1 := best.(string); ok1 {
					if vs, ok2 := v.(string); ok2 {
						less = vs < bs
					}
				} else {
					less = rt.toNumber(v) < rt.toNumber(best)
				}
				if less != wantMax {
					best = v
				}
			}
			return w(best)
		},
		// clear(m) empties a map; clear(s) zeroes a slice's elements in place.
		"js_goclear": func(a []uint64) uint64 {
			args, isArgs := u(a[0]).(*jsArray)
			if !isArgs || len(args.elems) == 0 {
				rt.fail("clear() needs an argument")
			}
			x := args.elems[0]
			if keys, vals, ok := dictParts(x); ok {
				keys.dropIdx()
				keys.elems = nil
				vals.elems = nil
				return 0
			}
			if arr, ok := x.(*jsArray); ok {
				arr.dropIdx()
				for i := range arr.elems {
					arr.elems[i] = float64(0)
				}
			}
			return 0
		},
		// A variadic parameter: the arguments from index i on, as a slice.
		"js_gorest": func(a []uint64) uint64 {
			args, ok := u(a[0]).(*jsArray)
			if !ok {
				rt.fail("js_gorest needs the argument array")
			}
			from := int(a[1])
			out := &jsArray{}
			if from < len(args.elems) {
				out.elems = append(out.elems, args.elems[from:]...)
			}
			return w(out)
		},
		// f(xs...): flatten the LAST element of the argument array in place, so the
		// callee sees the slice's elements as separate arguments.
		"js_gospread": func(a []uint64) uint64 {
			args, ok := u(a[0]).(*jsArray)
			if !ok || len(args.elems) == 0 {
				return 0
			}
			last := args.elems[len(args.elems)-1]
			args.dropIdx()
			args.elems = args.elems[:len(args.elems)-1]
			if arr, isArr := last.(*jsArray); isArr {
				args.elems = append(args.elems, arr.elems...)
			} else if _, isNil := last.(jsNullT); !isNil {
				if _, isU := last.(jsUndefT); !isU {
					args.elems = append(args.elems, last)
				}
			}
			return 0
		},
		// A Go named-type conversion T(v). One numeric type carries every integer
		// width, so the sized forms truncate and mask; string(r) is the rune
		// spelling and string([]byte{...}) joins the code points.
		// s[lo:hi] / s[lo:hi:max]. On a Go STRING the bounds are BYTE offsets and the
		// result is the decoded byte range; on a slice the elements are COPIED (the
		// backing array is not shared yet - see tests/go-test-full.go section 07).
		// An absent bound arrives as undefined.
		"js_goslice": func(a []uint64) uint64 {
			bound := func(h uint64, dflt int) int {
				v := u(h)
				if _, isU := v.(jsUndefT); isU {
					return dflt
				}
				return int(rt.toNumber(v))
			}
			if str, isStr := u(a[0]).(string); isStr {
				lo := bound(a[1], 0)
				hi := bound(a[2], len(str))
				goSliceFail(lo, hi, 0, len(str), false, "length")
				return w(str[lo:hi])
			}
			arr, isArr := u(a[0]).(*jsArray)
			if !isArr {
				rt.fail("slicing a %s", rt.typeOf(u(a[0])))
			}
			lo := bound(a[1], 0)
			hi := bound(a[2], len(arr.elems))
			goSliceFail(lo, hi, 0, len(arr.elems), false, "length")
			out := &jsArray{}
			out.elems = append(out.elems, arr.elems[lo:hi]...)
			return w(out)
		},
		"js_goconv": func(a []uint64) uint64 {
			to := rt.toString(u(a[0]))
			v := u(a[1])
			switch to {
			case "string":
				switch t := v.(type) {
				case float64:
					return w(string(rune(int64(t))))
				case *jsArray:
					out := ""
					for _, e := range t.elems {
						out += string(rune(int64(rt.toNumber(e))))
					}
					return w(out)
				}
				return w(rt.toString(v))
			case "[]byte", "[]rune":
				// Both spellings decode to CODE POINTS, exactly like the interpreter's
				// goConvert: the value model has one array type, so a later string(x)
				// cannot tell bytes from runes apart. Identical for ASCII, which is
				// what the byte form is used on.
				out := &jsArray{}
				for _, r := range rt.toString(v) {
					out.elems = append(out.elems, float64(r))
				}
				return w(out)
			case "bool":
				return boolH(rt.truthy(v))
			case "float64":
				// float64(x) BOXES: the result is a float64 whatever x was, which
				// is what makes float64(7)/2 == 3.5 (see jsrtjvm.go).
				return w(jsJFlo{f: rt.toNumber(v), sty: floGo})
			case "float32":
				// float32(x) boxes AT THE 32 BIT WIDTH and therefore ROUNDS:
				// float32(0.1) is 0.1 read at 24 significant bits, which is why
				// float32(0.1)+float32(0.2) is 0.3 and not 0.30000000000000004.
				return w(jsJFlo{f: jvmFround(rt.toNumber(v)), sty: floGoF})
			}
			n := math.Trunc(rt.toNumber(v))
			switch to {
			case "byte", "uint8":
				return rt.wrapNum(float64(int64(n) & 255))
			case "uint16":
				return rt.wrapNum(float64(int64(n) & 65535))
			case "uint32":
				return rt.wrapNum(float64(int64(n) & 4294967295))
			case "int8":
				return rt.wrapNum(float64(int8(int64(n))))
			case "int16":
				return rt.wrapNum(float64(int16(int64(n))))
			case "int32", "rune":
				return rt.wrapNum(float64(int32(int64(n))))
			}
			if strings.HasPrefix(to, "int") || strings.HasPrefix(to, "uint") {
				return rt.wrapNum(n)
			}
			return a[1] // a named struct / interface type: the value is unchanged
		},
		// Go's == is VALUE equality for structs and arrays (both are value types) and
		// identity for everything else; nil compares equal to an unset slice/map/pointer.
		// Two struct values are equal when their descriptors agree on name and field
		// list (so two anonymous struct literals of the same shape compare equal, the
		// way Go's structural identity for unnamed types does) and every field is equal.
		// ----- Go's panic / defer / recover -----
		// A compiled Go function runs its BODY in a closure that js_gotry calls, so a
		// panic unwinding through the frame is caught HERE: the frame's deferred
		// functions still run afterwards, and a recover() in one of them takes the
		// panic value and stops the unwinding. js_gorepanic (emitted after the defers)
		// re-raises whatever was not recovered.
		"js_gotry": func(a []uint64) uint64 {
			rt.goPanics = append(rt.goPanics, nil)
			res := func() (res interface{}) {
				depth := len(rt.thisStack)
				ntDepth := len(rt.newTargetStack)
				defer func() {
					if caught := recover(); caught != nil {
						if rt.trackThis {
							if len(rt.thisStack) > depth {
								rt.thisStack = rt.thisStack[:depth]
							}
							if len(rt.newTargetStack) > ntDepth {
								rt.newTargetStack = rt.newTargetStack[:ntDepth]
							}
						}
						rt.goPanics[len(rt.goPanics)-1] = caught
						res = jsUndef
					}
				}()
				return rt.call(u(a[0]), jsUndef, nil)
			}()
			return w(res)
		},
		"js_gorepanic": func(a []uint64) uint64 {
			p := rt.goPanics[len(rt.goPanics)-1]
			rt.goPanics = rt.goPanics[:len(rt.goPanics)-1]
			if p != nil {
				panic(p)
			}
			return 0
		},
		"js_gopanic": func(a []uint64) uint64 { // panic(v): the argument array is the callee's
			args, _ := u(a[0]).(*jsArray)
			var v interface{} = jsUndef
			if args != nil && len(args.elems) > 0 {
				v = args.elems[0]
			}
			if isUndefOrNull(v) {
				v = goPanicNil()
			}
			panic(&jsThrown{value: v})
		},
		// recover(): the pending panic of the frame whose defers are running, or nil.
		// A runtime error (rt.fail panics a plain string) recovers as that message,
		// which is what Go's recover of a runtime error yields in spirit.
		"js_gorecover": func(a []uint64) uint64 {
			if len(rt.goDeferAt) == 0 {
				return w(jsNull)
			}
			at := rt.goDeferAt[len(rt.goDeferAt)-1]
			// Only a function DEFERRED by that frame may recover its panic: it is one
			// frame deeper (or, for a native callee, none).
			if at < 0 || at >= len(rt.goPanics) || len(rt.goPanics) > at+2 {
				return w(jsNull)
			}
			if rt.goPanics[at] == nil {
				return w(jsNull)
			}
			p := rt.goPanics[at]
			rt.goPanics[at] = nil
			if exc, ok := p.(*jsThrown); ok {
				return w(exc.value)
			}
			return w(fmt.Sprintf("%v", p))
		},
		// The completion value of a body closure: a return signal carries the returned
		// value, a body that just ran off its end returns undefined.
		"js_gounctl": func(a []uint64) uint64 {
			if ctl, ok := u(a[0]).(*jsCtl); ok {
				if ctl.kind == 1 {
					return w(ctl.value)
				}
				return w(jsUndef)
			}
			return w(jsUndef)
		},
		// Go ints are 64 bit, so << and >> are NOT the 32-bit js_shl/js_shr: they are
		// multiplication and division by a power of two, exact over the 53 bits a
		// float64 carries - which covers 1 << 40 and every other form the subset uses.
		// The interpreter's gshl/gshr do exactly this.
		"js_goshl": func(a []uint64) uint64 {
			n := rt.toNumber(u(a[1]))
			if n >= 53 {
				return rt.wrapNum(0)
			}
			return rt.wrapNum(rt.toNumber(u(a[0])) * math.Pow(2, n))
		},
		"js_goshr": func(a []uint64) uint64 {
			l := rt.toNumber(u(a[0]))
			n := rt.toNumber(u(a[1]))
			if n >= 53 {
				if l < 0 {
					return rt.wrapNum(-1)
				}
				return rt.wrapNum(0)
			}
			return rt.wrapNum(math.Trunc(l / math.Pow(2, n)))
		},
		// make(chan T[, n]): the buffer size is not modeled (a send never blocks), the
		// ELEMENT TYPE is, so a receive from a closed channel yields its zero value.
		"js_gochan_new": func(a []uint64) uint64 {
			o := newJSObject()
			o.set("__chan", true)
			o.set("buf", &jsArray{})
			o.set("closed", false)
			o.set("czero", rt.toString(u(a[0])))
			return w(o)
		},
		"js_gochan_send": func(a []uint64) uint64 {
			o, buf, ok := goChanParts(u(a[0]))
			if !ok {
				rt.fail("send on a non-channel")
			}
			if o.props["closed"] == true {
				rt.fail("send on closed channel")
			}
			buf.elems = append(buf.elems, u(a[1]))
			return 0
		},
		"js_gochan_recv": func(a []uint64) uint64 {
			v, _ := rt.goChanRecv(u(a[0]))
			return w(v)
		},
		// v, ok := <-ch: the pair, as a two-element array the emitted code indexes.
		"js_gochan_recvok": func(a []uint64) uint64 {
			v, ok := rt.goChanRecv(u(a[0]))
			return w(&jsArray{elems: []interface{}{v, ok}})
		},
		// Can a receive proceed WITHOUT blocking? What a select's arms ask before
		// falling back to the default arm; it drains the goroutine queue first.
		"js_gochan_ready": func(a []uint64) uint64 {
			o, buf, ok := goChanParts(u(a[0]))
			if !ok {
				rt.fail("receive from a non-channel")
			}
			if len(buf.elems) == 0 && o.props["closed"] != true {
				rt.goDrain()
			}
			return boolH(len(buf.elems) > 0 || o.props["closed"] == true)
		},
		"js_gochan_close": func(a []uint64) uint64 { // close(ch): the callee's argument array
			args, _ := u(a[0]).(*jsArray)
			if args == nil || len(args.elems) == 0 {
				rt.fail("close() needs a channel")
			}
			o, _, ok := goChanParts(args.elems[0])
			if !ok {
				rt.fail("close() needs a channel")
			}
			o.set("closed", true)
			return w(jsUndef)
		},
		"js_gospawn": func(a []uint64) uint64 { // go f(...): queue the already-bound call
			rt.goQueue = append(rt.goQueue, u(a[0]))
			return 0
		},
		"js_godrain": func(a []uint64) uint64 { rt.goDrain(); return 0 },
		"js_goeq":    func(a []uint64) uint64 { return boolH(rt.goValueEq(u(a[0]), u(a[1]))) },
		"js_gone":    func(a []uint64) uint64 { return boolH(!rt.goValueEq(u(a[0]), u(a[1]))) },
		// A Go field read: the struct's own fields, then the PROMOTED fields of an
		// embedded struct, then the ordinary member lookup (methods, len, builtins).
		"js_gofield": func(a []uint64) uint64 {
			key := rt.toString(u(a[1]))
			if o, ok := u(a[0]).(*jsObject); ok {
				if v, has := o.props[key]; has {
					return w(v)
				}
				if v, has := rt.goPromField(o, key, 16); has {
					return w(v)
				}
				// A METHOD VALUE: the receiver is bound now (and a value receiver is
				// COPIED now), so later writes to the variable do not reach it.
				if fn, recv, found := rt.goMethod(u(a[0]), key); found {
					return w(jsHostFunc(key, func(rt *jsrt, this uint64, args []interface{}) interface{} {
						return rt.call(fn, jsUndef, append([]interface{}{recv}, args...))
					}))
				}
				// A METHOD EXPRESSION: T.m / (*T).m off the descriptor itself. The
				// compiled method already takes its receiver as the first argument,
				// so the raw function IS the method expression.
				if o.props["__isclass"] == true {
					if fn, has := o.props[key]; has {
						return w(fn)
					}
				}
			}
			return w(rt.getMember(u(a[0]), u(a[1])))
		},
		// A Go method call: the struct's own methods and the promoted methods of an
		// embedded struct, with Go's receiver semantics (a value receiver gets a copy).
		// Anything else - fmt.Println, a string or slice builtin - falls through to the
		// ordinary member call.
		"js_gomcall": func(a []uint64) uint64 {
			args, ok := u(a[2]).(*jsArray)
			if !ok {
				rt.fail("js_gomcall args must be an array")
			}
			name := rt.toString(u(a[1]))
			if fn, recv, found := rt.goMethod(u(a[0]), name); found {
				return w(rt.call(fn, jsUndef, append([]interface{}{recv}, args.elems...)))
			}
			return w(rt.memberCall(u(a[0]), name, args.elems))
		},
		"js_rundefers": func(a []uint64) uint64 { // Runs collected [f, args] pairs LIFO (Go defer).
			arr, ok := u(a[0]).(*jsArray)
			if !ok {
				rt.fail("js_rundefers needs the defer list")
			}
			// Which frame these defers belong to, for a recover() inside one of them
			// (Go only; goPanics is empty for every other grammar, so at is -1 and
			// js_gorecover answers nil). Popped even when a defer panics itself.
			rt.goDeferAt = append(rt.goDeferAt, len(rt.goPanics)-1)
			defer func() { rt.goDeferAt = rt.goDeferAt[:len(rt.goDeferAt)-1] }()
			for i := len(arr.elems) - 1; i >= 0; i-- {
				pair, ok := arr.elems[i].(*jsArray)
				if !ok || len(pair.elems) != 2 {
					rt.fail("broken defer entry")
				}
				args, _ := pair.elems[1].(*jsArray)
				rt.call(pair.elems[0], jsUndef, args.elems)
			}
			return 0
		},
		"js_pylen": func(a []uint64) uint64 { // len() for strings, lists and dicts.
			if _, cls, isInst := pyInstance(u(a[0])); isInst { // A user class may define __len__.
				if m, found := pyLookup(cls, "__len__"); found && isCallable(m) {
					return w(rt.call(m, jsUndef, []interface{}{u(a[0])}))
				}
			}
			if els, ok := pySetElems(u(a[0])); ok {
				return rt.wrapNum(float64(len(els.elems)))
			}
			switch o := u(a[0]).(type) {
			case string:
				// Python counts CODE POINTS, not UTF-16 code units, so len of a
				// single astral character (an emoji) is 1 and not 2.
				return rt.wrapNum(float64(pyStrLen(rt, o)))
			case *jsArray:
				return rt.wrapNum(float64(len(o.elems)))
			}
			if keys, _, ok := dictParts(u(a[0])); ok {
				return rt.wrapNum(float64(len(keys.elems)))
			}
			// A ruby Enumerator's length is its element count: ruby-to-llvm-ir.abnf
			// lowers a bare `.size' / `.length' to js_pylen rather than to a method
			// call, so it never reaches rubyEnumMethod. Only ruby builds a __renum
			// object, so no other language can reach this arm.
			if rubyIsEnum(u(a[0])) {
				if items, isArr := u(a[0]).(*jsObject).props["items"].(*jsArray); isArr {
					return rt.wrapNum(float64(len(items.elems)))
				}
			}
			// A ruby Range's length is its ELEMENT COUNT, computed arithmetically
			// rather than by materializing, because for-in asks for it once and
			// then indexes. Only ruby builds a __rrange object. See js_pyget.
			if r, isRng := rubyIsRange(u(a[0])); isRng {
				return rt.wrapNum(rt.rubyRangeLen(r))
			}
			rt.fail("len() of a %s", rt.typeOf(u(a[0])))
			return 0
		},
		"js_pylist": func(a []uint64) uint64 { // list(x): always a fresh list.
			if g, ok := u(a[0]).(*jsGenerator); ok {
				return w(g.drain(rt))
			}
			// An ITERATOR object gives what is LEFT of it, and is exhausted by the
			// read - CPython's rule, and what makes next(it) then list(it) work.
			if pyIterName(rt, u(a[0])) != "" {
				return w(&jsArray{elems: rt.pyElemsOf(u(a[0]))})
			}
			if els, ok := pySetElems(u(a[0])); ok {
				return w(&jsArray{elems: append([]interface{}{}, els.elems...)})
			}
			switch o := u(a[0]).(type) {
			case *jsArray:
				return w(&jsArray{elems: append([]interface{}{}, o.elems...)})
			case string:
				return w(&jsArray{elems: pyStrChars(rt, o)})
			}
			if keys, _, ok := dictParts(u(a[0])); ok {
				return w(&jsArray{elems: append([]interface{}{}, keys.elems...)})
			}
			return w(&jsArray{elems: rt.pyElemsOf(u(a[0]))}) // The TypeError for the rest.
		},
		"js_pyiter": func(a []uint64) uint64 { // The list a for loop runs over: dicts iterate
			if pyIterName(rt, u(a[0])) != "" { // An iterator object, exhausted by the read.
				return w(&jsArray{elems: rt.pyElemsOf(u(a[0]))})
			}
			if g, ok := u(a[0]).(*jsGenerator); ok { // A generator is drained first.
				return w(g.drain(rt))
			}
			if els, ok := pySetElems(u(a[0])); ok { // A set iterates its elements.
				return w(&jsArray{elems: append([]interface{}{}, els.elems...)})
			}
			switch o := u(a[0]).(type) { // their keys, strings their characters.
			case *jsArray:
				return a[0]
			case string:
				return w(&jsArray{elems: pyStrChars(rt, o)})
			}
			if keys, _, ok := dictParts(u(a[0])); ok {
				return w(&jsArray{elems: append([]interface{}{}, keys.elems...)})
			}
			if it, ok := rt.pyUserIter(u(a[0])); ok {
				return w(&jsArray{elems: rt.pyDrainIter(it)})
			}
			rt.fail("iteration over a %s", rt.typeOf(u(a[0])))
			return 0
		},
		"js_pyslice": func(a []uint64) uint64 { // s[lo:hi] on lists and strings; open ends are undefined.
			switch o := u(a[0]).(type) {
			case *jsArray:
				from, to := rt.pySliceRange(u(a[1]), u(a[2]), len(o.elems))
				return w(&jsArray{elems: append([]interface{}{}, o.elems[from:to]...)})
			case string:
				from, to := rt.pySliceRange(u(a[1]), u(a[2]), rt.strLen(o))
				return rt.wrapStr(rt.strRange(o, from, to))
			}
			rt.fail("slicing a %s", rt.typeOf(u(a[0])))
			return 0
		},
		// Python's assignment rule: the name binds in the innermost BINDING
		// BOUNDARY scope (the def, lambda, class body or module the assignment
		// stands in - see jsScope.pyFn), unless a `global` / `nonlocal`
		// declaration sent it elsewhere. The walk still passes through the
		// scopes BELOW that boundary, which is what makes an assignment inside
		// a try/except/with body - each of which is an own IR closure with a
		// scope of its own - land in the enclosing function, as Python has it.
		"js_pyset_var": func(a []uint64) uint64 {
			sc := rt.scopeOf(a[0])
			name := rt.toString(u(a[1]))
			for s := sc; s != nil; s = s.parent {
				if k, ok := s.pyDecl[name]; ok {
					if k == 'g' {
						pyModuleScope(s).put(name, u(a[2]))
						return 0
					}
					for t := s.parent; t != nil; t = t.parent {
						if t.has(name) {
							t.put(name, u(a[2]))
							return 0
						}
					}
				}
				if s.has(name) {
					s.put(name, u(a[2]))
					if rt.traced {
						rt.trVar("write", name, u(a[2]))
					}
					return 0
				}
				if s.pyFn {
					s.put(name, u(a[2]))
					if rt.traced {
						rt.trVar("decl", name, u(a[2]))
					}
					return 0
				}
			}
			sc.put(name, u(a[2]))
			if rt.traced {
				rt.trVar("decl", name, u(a[2]))
			}
			return 0
		},
		// Mark a scope as a Python binding boundary (a def, lambda, class body or
		// the module top level). Only the Python compiler emits this.
		"js_pyfnscope": func(a []uint64) uint64 {
			rt.scopeOf(a[0]).pyFn = true
			return 0
		},
		"js_pyrest": func(a []uint64) uint64 { // *args: the call arguments from index N on, as a list.
			arr := u(a[0]).(*jsArray)
			from := int(a[1])
			rest := &jsArray{}
			for i := from; i < len(arr.elems); i++ {
				rest.elems = append(rest.elems, arr.elems[i])
			}
			return w(rest)
		},
		"js_pyglobal": func(a []uint64) uint64 { // global NAME: ensure NAME is bound at module level.
			sc := rt.scopeOf(a[0])
			name := rt.toString(u(a[1]))
			root := pyModuleScope(sc)
			if !root.has(name) {
				root.put(name, jsUndef)
			}
			if sc.pyDecl == nil {
				sc.pyDecl = map[string]byte{}
			}
			sc.pyDecl[name] = 'g' // A later assignment reaches the root, not a local.
			return 0
		},
		"js_pynonlocal": func(a []uint64) uint64 { // nonlocal NAME: an enclosing non-root scope must bind NAME.
			sc := rt.scopeOf(a[0])
			name := rt.toString(u(a[1]))
			for s := sc.parent; s != nil && s.parent != nil; s = s.parent {
				if s.has(name) {
					if sc.pyDecl == nil {
						sc.pyDecl = map[string]byte{}
					}
					sc.pyDecl[name] = 'n' // A later assignment reaches that binding.
					return 0
				}
			}
			rt.fail("no binding for nonlocal %s", name)
			return 0
		},
		"js_pyrange": func(a []uint64) uint64 { // range(stop) / range(start, stop[, step]), materialized.
			from, step := 0.0, 1.0
			to := rt.pyRangeArg(u(a[0]))
			if _, undef := u(a[1]).(jsUndefT); !undef {
				from, to = to, rt.pyRangeArg(u(a[1]))
			}
			if _, undef := u(a[2]).(jsUndefT); !undef {
				step = rt.pyRangeArg(u(a[2]))
			}
			if step == 0 {
				panic(&jsThrown{value: rt.pyExcInstance("ValueError", "range() arg 3 must not be zero")})
			}
			arr := &jsArray{}
			for i := from; (step > 0 && i < to) || (step < 0 && i > to); i += step {
				arr.elems = append(arr.elems, i)
			}
			return w(arr)
		},
		"js_pyin": func(a []uint64) uint64 { // Python 'x in y' for lists, strings and dict keys.
			switch c := u(a[1]).(type) {
			case *jsArray:
				for _, e := range c.elems {
					if rt.strictEq(e, u(a[0])) {
						return boolH(true)
					}
				}
				return boolH(false)
			case string:
				return boolH(strings.Contains(c, rt.toString(u(a[0]))))
			}
			if keys, _, ok := dictParts(u(a[1])); ok {
				return boolH(rt.pyDictFind(keys, u(a[0])) >= 0)
			}
			rt.fail("'in' needs a list, a string or a dict on the right side")
			return boolH(false)
		},
		// Python's == compares CONTAINERS BY VALUE, unlike JavaScript's, where two
		// distinct arrays are never equal. js_pyeq/js_pyne are the Python spelling:
		// lists compare element by element and dicts key by key (order-insensitively,
		// as Python does), recursively; everything else falls back to strict equality.
		// 'is' keeps using js_seq, because that one really is identity.
		"js_pyeq": func(a []uint64) uint64 { return boolH(rt.pyEqual(u(a[0]), u(a[1]))) },
		"js_pyne": func(a []uint64) uint64 { return boolH(!rt.pyEqual(u(a[0]), u(a[1]))) },

		// ----- Python classes (see languages/python-to-llvm-ir.abnf) -----
		//
		// A Python class object is a plain jsObject carrying "__name", the C3
		// linearized "__mro" (a jsArray of class objects, the class itself first)
		// and "__super" (the FIRST base, so every existing __super walker -
		// memberCall, js_instanceof, rtIsType - keeps working on single
		// inheritance). An instance is a jsObject with "__class" pointing at its
		// class; that is the same shape js_pyexc already produces, and it is
		// exactly the receiver-prepending convention rt.memberCall implements.
		// Multiple inheritance is resolved ONCE, here, so every later lookup is a
		// flat walk of "__mro".
		"js_pyclass_new": func(a []uint64) uint64 { // (name, bases array) -> class
			name := rt.toString(u(a[0]))
			bases, _ := u(a[1]).(*jsArray)
			cls := newJSObject()
			cls.set("__name", name)
			var baseObjs []*jsObject
			if bases != nil {
				for _, b := range bases.elems {
					bo, ok := b.(*jsObject)
					if !ok {
						rt.fail("class %s: base is not a class", name)
					}
					baseObjs = append(baseObjs, bo)
				}
			}
			if len(baseObjs) > 0 {
				cls.set("__super", baseObjs[0])
			}
			// The declared bases, for __bases__. Kept beside __mro because the
			// linearization cannot be un-merged back into the base list.
			bl := &jsArray{}
			for _, bo := range baseObjs {
				bl.elems = append(bl.elems, bo)
			}
			cls.set("__basesl", bl)
			mro := &jsArray{elems: []interface{}{cls}}
			for _, c := range pyLinearize(baseObjs) {
				mro.elems = append(mro.elems, c)
			}
			cls.set("__mro", mro)
			return w(cls)
		},
		// super(): what a READ of the name `super` lowers to - see makeVarRef in
		// languages/python-to-llvm-ir.abnf for why the magic is in the emitter and
		// what the two operands are. The result is both the callable (super() with
		// no arguments answers itself, super(C, o) builds a new one) and the proxy
		// an attribute read or a method call goes through.
		"js_pysuper": func(a []uint64) uint64 { // (defining class|undef, instance|undef)
			return w(pySuperNew(u(a[0]), u(a[1])))
		},
		// The class body is a suite: it runs in a scope of its own and every name
		// it bound (methods, class attributes, __slots__) becomes a class member.
		"js_pyclass_fill": func(a []uint64) uint64 { // (class, class-body scope)
			cls, ok := u(a[0]).(*jsObject)
			if !ok {
				rt.fail("js_pyclass_fill: not a class object")
			}
			sc := rt.scopeOf(a[1])
			for i, n := range sc.names {
				cls.set(n, sc.vals[i])
			}
			return 0
		},
		// obj.name: an instance checks its own attributes, then the MRO of its
		// class (running a __get__ descriptor found there); a class checks its own
		// MRO. Anything else takes the ordinary member read.
		"js_pygetattr": func(a []uint64) uint64 { // (object, name) -> value
			return w(rt.pyGetAttr(u(a[0]), rt.toString(u(a[1]))))
		},
		// obj.name = v: a __set__ descriptor on the class wins; __slots__ (when the
		// class declares it) rejects an unknown name by raising, like Python.
		"js_pysetattr": func(a []uint64) uint64 { // (object, name, value)
			rt.pySetAttr(u(a[0]), rt.toString(u(a[1])), u(a[2]))
			return 0
		},
		// obj.name(args): an instance dispatches along the MRO with itself
		// prepended (Python's `self`); a CLASS dispatches unbound, which is what
		// makes the explicit `Base.m(self)` super-call spelling work.
		"js_pymcall": func(a []uint64) uint64 { // (target, method name, args array, keyword dict|undef)
			args, ok := u(a[2]).(*jsArray)
			if !ok {
				rt.fail("js_pymcall args must be an array")
			}
			return w(rt.pyMethodCall(u(a[0]), rt.toString(u(a[1])), args.elems, u(a[3])))
		},
		// f(args, kwargs): calling a CLASS instantiates it (allocate, then __init__
		// with self prepended); calling an instance runs its __call__. The keyword
		// dict is mapped onto the callee's parameters by pyBindCall.
		"js_pycall": func(a []uint64) uint64 { // (callee, positional array, keyword dict|undef)
			args, ok := u(a[1]).(*jsArray)
			if !ok {
				rt.fail("js_pycall args must be an array")
			}
			callee, kw := u(a[0]), u(a[2])
			if sup, isSup := pySuperObj(callee); isSup {
				return w(rt.pySuperCall(sup, args.elems))
			}
			// A BUILTIN TYPE OBJECT is callable: `int` is the class, and calling it
			// converts. The emitter binds int/str/float/bool/list/dict/set/tuple to
			// js_pybuiltincls's stable class object with the conversion closure
			// hung on it as __conv, so `type(1) == int` is True (it is the same
			// object) and int("42") still runs the same IR function it always did.
			if o, isObj := callee.(*jsObject); isObj {
				if conv, has := o.props["__conv"]; has && isCallable(conv) {
					return w(rt.call(conv, jsUndef, rt.pyConvArgs(o, conv, args.elems, kw)))
				}
			}
			if cls, ok := pyClassObj(callee); ok {
				return w(rt.pyNewInstance(cls, args.elems, kw))
			}
			if _, cls, ok := pyInstance(callee); ok {
				if m, found := pyLookup(cls, "__call__"); found && isCallable(m) {
					self := append([]interface{}{callee}, args.elems...)
					return w(rt.call(m, jsUndef, rt.pyBindCall(m, self, kw)))
				}
			}
			rt.pyCheckCallable(callee)
			bound, rebound := rt.pyBindCallR(callee, args.elems, kw)
			if !rebound {
				return w(rt.callH(callee, jsUndef, args.elems, a[1]))
			}
			return w(rt.call(callee, jsUndef, bound))
		},
		// Register a def's parameter shape, so a keyword call can be bound to it.
		"js_pysig": func(a []uint64) uint64 { // (closure, name array, kw-only count, extended?)
			if rt.pySigs == nil {
				rt.pySigs = map[interface{}]*pySig{}
			}
			names, _ := u(a[1]).(*jsArray)
			sig := &pySig{nkwonly: int(rt.toNumber(u(a[2]))), ext: rt.truthy(u(a[3]))}
			if names != nil {
				for _, n := range names.elems {
					sig.names = append(sig.names, rt.toString(n))
				}
			}
			rt.pySigs[u(a[0])] = sig
			return a[0]
		},
		// *seq / **map unpacking at a call site: append / merge into the array or
		// dict the call is building.
		"js_pyspread": func(a []uint64) uint64 { // (positional array, iterable)
			arr, ok := u(a[0]).(*jsArray)
			if !ok {
				rt.fail("js_pyspread needs an array")
			}
			switch o := u(a[1]).(type) {
			case *jsArray:
				arr.dropIdx()
				arr.elems = append(arr.elems, o.elems...)
				return 0
			case string:
				arr.dropIdx()
				arr.elems = append(arr.elems, pyStrChars(rt, o)...)
				return 0
			}
			if keys, _, ok := dictParts(u(a[1])); ok {
				arr.dropIdx()
				arr.elems = append(arr.elems, keys.elems...)
				return 0
			}
			rt.fail("cannot unpack a %s with *", rt.typeOf(u(a[1])))
			return 0
		},
		"js_pykwspread": func(a []uint64) uint64 { // (keyword dict, mapping)
			keys, vals, ok := dictParts(u(a[0]))
			if !ok {
				rt.fail("js_pykwspread needs a dict")
			}
			sk, sv, ok := dictParts(u(a[1]))
			if !ok {
				rt.fail("cannot unpack a %s with **", rt.typeOf(u(a[1])))
			}
			for i, k := range sk.elems {
				if j := rt.pyDictFind(keys, k); j >= 0 {
					vals.elems[j] = sv.elems[i]
				} else {
					dictAppend(keys, vals, k, sv.elems[i])
				}
			}
			return 0
		},
		// type(v): the class object of an instance, else a synthetic class object
		// naming the builtin type (so type(x).__name__ answers for everything).
		"js_pytype": func(a []uint64) uint64 {
			v := u(a[0])
			if _, cls, ok := pyInstance(v); ok {
				return w(cls)
			}
			return w(rt.pyBuiltinClass(pyTypeName(rt, v)))
		},
		// isinstance(v, C) / isinstance(v, (A, B)): a walk of the instance's MRO.
		"js_pyisinst": func(a []uint64) uint64 {
			return boolH(rt.pyIsInstance(u(a[0]), u(a[1])))
		},
		// issubclass(C, B).
		"js_pyissubclass": func(a []uint64) uint64 {
			return boolH(rt.pyIsSubclass(u(a[0]), u(a[1])))
		},
		// A complex number: (real, imaginary) -> the {__cplx, real, imag} value the
		// complex arithmetic in js_pybin / js_pyunary and the .real / .imag attribute
		// reads understand. Only the Python compiler grammar builds this shape.
		"js_pycomplex": func(a []uint64) uint64 {
			return w(newPyComplex(rt.toNumber(u(a[0])), rt.toNumber(u(a[1]))))
		},
		// complex(real=0, imag=0), bound as [positionals, keywords].
		"js_pycomplexnew": func(a []uint64) uint64 {
			el := u(a[0]).(*jsArray).elems
			pos := el[0].(*jsArray).elems
			if keys, _, ok := dictParts(el[1]); ok && len(keys.elems) > 0 {
				pos = rt.pyKwPos("complex", pos, el[1], []string{"real", "imag"}, 0)
				if pos[0] == jsUndef {
					pos[0] = 0.0
				}
			}
			return w(rt.pyComplexNew(pos))
		},
		// x: T in a class or module body records the annotation. __annotations__ is
		// created in THIS scope on first use (js_scope_decl writes the own scope, so a
		// class body gets its own dict rather than adding to the module's) and the
		// annotation is kept as its source text - enough for 'name in __annotations__'
		// without an evaluated type object.
		"js_pyannot": func(a []uint64) uint64 { // (scope, name, annotation text)
			sc := rt.scopeOf(a[0])
			var keys, vals *jsArray
			if i := sc.find("__annotations__"); i >= 0 {
				if k, v, ok := dictParts(sc.vals[i]); ok {
					keys, vals = k, v
				}
			}
			if keys == nil {
				keys, vals = &jsArray{}, &jsArray{}
				sc.put("__annotations__", &jsObject{props: map[string]interface{}{
					"__dict": true, "keys": keys, "vals": vals,
				}})
			}
			dictAppend(keys, vals, rt.toString(u(a[1])), u(a[2]))
			return 0
		},
		// {a, b, c} / {e for x in it}: a real set built from the candidate list the
		// literal or the comprehension collected, deduplicated with Python's ==.
		"js_pyset_new": func(a []uint64) uint64 {
			out := &jsArray{}
			// ANY iterable, not just a list: set("aab") is {'a','b'}. set() with no
			// argument is the empty set.
			if !isUndefOrNull(u(a[0])) {
				for _, e := range rt.pyElemsOf(u(a[0])) {
					rt.pySetAdd(out, e)
				}
			}
			return w(newPySet(out))
		},
		// Add one element (or, for a '*s' spread, every element of an iterable) to the
		// candidate list a set literal is being built from.
		"js_pyset_spread": func(a []uint64) uint64 { // (candidate list, iterable)
			dst, isArr := u(a[0]).(*jsArray)
			if !isArr {
				return 0
			}
			if src, ok := pySetElems(u(a[1])); ok {
				dst.elems = append(dst.elems, src.elems...)
				return 0
			}
			if src, isSrc := u(a[1]).(*jsArray); isSrc {
				dst.elems = append(dst.elems, src.elems...)
				return 0
			}
			if keys, _, isDict := dictParts(u(a[1])); isDict {
				dst.elems = append(dst.elems, keys.elems...)
				return 0
			}
			rt.fail("cannot unpack a %s into a set", rt.typeOf(u(a[1])))
			return 0
		},
		// s[lo:hi:step] with a step, on lists and strings. The open ends arrive as
		// undefined and mean what Python's defaults mean for the DIRECTION of the step,
		// so xs[::-1] is the reversal.
		"js_pyslice3": func(a []uint64) uint64 { // (sequence, lo, hi, step)
			switch o := pyUnwrapBuiltin(u(a[0])).(type) { // A builtin subclass slices its value.
			case *jsArray:
				idx := rt.pySliceIndices(len(o.elems), u(a[1]), u(a[2]), u(a[3]))
				out := &jsArray{}
				for _, i := range idx {
					out.elems = append(out.elems, o.elems[i])
				}
				return w(out)
			case string:
				cs := pyStrChars(rt, o)
				var out strings.Builder
				for _, i := range rt.pySliceIndices(len(cs), u(a[1]), u(a[2]), u(a[3])) {
					out.WriteString(cs[i].(string))
				}
				return rt.wrapStr(out.String())
			}
			rt.fail("slicing a %s", rt.typeOf(u(a[0])))
			return 0
		},
		// xs[lo:hi:step] = values. A step of 1 SPLICES (the replacement may have a
		// different length, so xs[1:3] = [9] shortens the list); an extended step
		// assigns element by element and needs exactly as many values as it selects.
		"js_pysetslice": func(a []uint64) uint64 { // (list, lo, hi, step, values)
			arr, isArr := pyUnwrapBuiltin(u(a[0])).(*jsArray)
			if !isArr {
				rt.fail("slice assignment to a %s", rt.typeOf(u(a[0])))
				return 0
			}
			vals := rt.pySeqElems(u(a[4]))
			if isUndefOrNull(u(a[3])) || rt.toNumber(u(a[3])) == 1 {
				idx := rt.pySliceIndices(len(arr.elems), u(a[1]), u(a[2]), u(a[3]))
				lo := len(arr.elems)
				hi := lo
				if len(idx) > 0 {
					lo, hi = idx[0], idx[len(idx)-1]+1
				} else {
					lo = rt.pySliceStart(len(arr.elems), u(a[1]))
					hi = lo
				}
				out := append([]interface{}{}, arr.elems[:lo]...)
				out = append(out, vals...)
				out = append(out, arr.elems[hi:]...)
				arr.elems = out
				return 0
			}
			idx := rt.pySliceIndices(len(arr.elems), u(a[1]), u(a[2]), u(a[3]))
			if len(idx) != len(vals) {
				rt.fail("attempt to assign sequence of size %d to extended slice of size %d",
					len(vals), len(idx))
			}
			for k, i := range idx {
				arr.elems[i] = vals[k]
			}
			return 0
		},
		// del xs[lo:hi:step]: drop every selected element.
		"js_pydelslice": func(a []uint64) uint64 { // (list, lo, hi, step)
			arr, isArr := pyUnwrapBuiltin(u(a[0])).(*jsArray)
			if !isArr {
				rt.fail("deleting a slice of a %s", rt.typeOf(u(a[0])))
				return 0
			}
			idx := rt.pySliceIndices(len(arr.elems), u(a[1]), u(a[2]), u(a[3]))
			drop := map[int]bool{}
			for _, i := range idx {
				drop[i] = true
			}
			out := []interface{}{}
			for i, e := range arr.elems {
				if !drop[i] {
					out = append(out, e)
				}
			}
			arr.elems = out
			return 0
		},
		// The value a `raise` actually raises. `raise SomeClass` names a CLASS, and
		// CPython instantiates it with no arguments; anything else is raised as it
		// stands. A class OBJECT would match no except clause. The interpreter half
		// has the same line inside makeRaise. g.throw(SomeClass) instantiates the
		// same way (pyMethodCall's throw arm).
		"js_pyraiseval": func(a []uint64) uint64 {
			if _, ok := pyClassObj(u(a[0])); ok {
				return w(rt.callPyValue(u(a[0]), nil, jsUndef))
			}
			return a[0]
		},
		// The type test of an except clause. It is isinstance, PLUS the one thing an
		// except clause has to answer that isinstance must not: this subset lets a
		// program raise a value that is no class instance at all (raise "boom"), and
		// only the ROOT type - Exception / BaseException - catches such a value.
		"js_pyexcmatch": func(a []uint64) uint64 {
			v, typ := u(a[0]), u(a[1])
			if rt.pyIsInstance(v, typ) {
				return boolH(true)
			}
			if _, _, isInst := pyInstance(v); isInst {
				return boolH(false)
			}
			return boolH(pyRootExcType(typ))
		},
		// PEP 654 'except* T' dispatch: (ExceptionGroup instance, type, ExceptionGroup
		// class) -> a FRESH group over the leaves of the first that the type accepts,
		// or undefined when the value is not a group or nothing in it matches. Only the
		// Python grammars ever build the {__class, args, exceptions} shape it reads.
		"js_pyexcsplit": func(a []uint64) uint64 {
			grp, typ, cls := u(a[0]), u(a[1]), u(a[2])
			leaves, msg, isGroup := rt.pyExcGroupLeaves(grp)
			if !isGroup {
				return jsHUndefined
			}
			matched := []interface{}{}
			for _, e := range leaves {
				if rt.pyIsInstance(e, typ) {
					matched = append(matched, e)
				}
			}
			if len(matched) == 0 {
				return jsHUndefined
			}
			sub := newJSObject()
			sub.set("__class", cls)
			sub.set("args", &jsArray{elems: []interface{}{msg, &jsArray{elems: matched}}})
			sub.set("exceptions", &jsArray{elems: matched})
			return w(sub)
		},
		// Every user visible binary operator of the Python compiler goes through
		// here so an instance of a user class can answer it with a dunder
		// (__add__, __lt__, __eq__, __contains__, ...). Everything that is not
		// such an instance takes exactly the path its plain js_* extern takes.
		"js_pybin": func(a []uint64) uint64 { // (operator, left, right)
			op := rt.toString(u(a[0]))
			l, r := u(a[1]), u(a[2])
			// `xs += it` is list.extend IN PLACE, so every alias of xs sees it and
			// any iterable is accepted; on anything else it is the binary `+`.
			if op == "+=" {
				if la, ok := l.(*jsArray); ok {
					la.elems = append(la.elems, rt.pyElemsOf(r)...)
					return a[1]
				}
				op = "+"
			}
			switch op {
			case "is":
				return boolH(rt.strictEq(l, r))
			case "is not":
				return boolH(!rt.strictEq(l, r))
			case "in", "not in":
				want := op == "in"
				if _, cls, ok := pyInstance(r); ok {
					if m, found := pyLookup(cls, "__contains__"); found && isCallable(m) {
						hit := rt.truthy(rt.call(m, jsUndef, []interface{}{r, l}))
						return boolH(hit == want)
					}
				}
				return boolH(rt.pyContains(l, r) == want)
			}
			if v, ok := rt.pySetBin(op, l, r); ok {
				return w(v)
			}
			switch op {
			case "|", "&", "^", "<<", ">>":
				// bool & bool is a BOOL in Python (True & True is True, and
				// repr says so), where a bool meeting an int is an int. Only
				// the three logical operators; True << 1 is the int 2.
				if lb, ok := l.(bool); ok {
					if rb, ok2 := r.(bool); ok2 && (op == "|" || op == "&" || op == "^") {
						switch op {
						case "|":
							return boolH(lb || rb)
						case "&":
							return boolH(lb && rb)
						}
						return boolH(lb != rb)
					}
				}
				// Two exact Python ints answer in ARBITRARY PRECISION two's
				// complement (pyBitBin). Anything else - a float, a string, an
				// instance - keeps ToInt32 behaviour, after the user class
				// has had its __and__ / __lshift__ chance (pyDunderBin).
				if x, xok := pyBigOperand(l); xok {
					if y, yok := pyBigOperand(r); yok {
						v, ok := pyBitBin(op, x, y)
						if !ok {
							rt.fail("negative shift count")
						}
						return w(rt.pyBigNarrow(v))
					}
				}
				if v, ok := rt.pyDunderBin(op, l, r); ok {
					return w(v)
				}
				switch op {
				case "|":
					return rt.wrapNum(float64(rt.toInt32(l) | rt.toInt32(r)))
				case "&":
					return rt.wrapNum(float64(rt.toInt32(l) & rt.toInt32(r)))
				case "^":
					return rt.wrapNum(float64(rt.toInt32(l) ^ rt.toInt32(r)))
				case "<<":
					return rt.wrapNum(float64(rt.toInt32(l) << (uint32(rt.toInt32(r)) & 31)))
				}
				return rt.wrapNum(float64(rt.toInt32(l) >> (uint32(rt.toInt32(r)) & 31)))
			}
			if isPyComplex(l) || isPyComplex(r) {
				// The complex arithmetic reads any operand as a number, so a mixed
				// `+` is refused here unless a user class may still answer it.
				_, _, li := pyInstance(l)
				_, _, ri := pyInstance(r)
				if op == "+" && !li && !ri {
					rt.pyAddCheck(l, r)
				}
				if v, ok := rt.pyComplexBin(op, l, r); ok {
					return w(v)
				}
			}
			if v, ok := rt.pyDunderBin(op, l, r); ok {
				return w(v)
			}
			// An int operand meeting an arbitrary precision int PROMOTES, because
			// Python has one int type - for + - * and for //, % and ** alike.
			if x, y, both := pyBigPair(l, r); both {
				if v, ok := pyBigBin(op, x, y); ok {
					return w(rt.pyBigNarrow(v))
				}
				// The ORDER comparisons need the same promotion: rt.jsCompare's
				// big arm is bigPair's, so a big meeting a plain int would reach
				// toNumber and answer NaN.
				switch op {
				case "<", ">", "<=", ">=", "==", "!=":
					c := x.Cmp(y)
					switch op {
					case "<":
						return boolH(c < 0)
					case ">":
						return boolH(c > 0)
					case "<=":
						return boolH(c <= 0)
					case ">=":
						return boolH(c >= 0)
					case "==":
						return boolH(c == 0)
					}
					return boolH(c != 0)
				}
			}
			// An arbitrary precision int meeting a float: the comparisons are EXACT
			// and the arithmetic takes the int as its nearest double (OverflowError
			// past the double range), after which the float arms below apply.
			if bi, f, bigLeft, ok := pyBigFloPair(l, r); ok {
				switch op {
				case "<", ">", "<=", ">=", "==", "!=":
					c := pyCmpBigFloat(bi, f)
					if !bigLeft && c != 2 {
						c = -c
					}
					switch op {
					case "<":
						return boolH(c == -1)
					case ">":
						return boolH(c == 1)
					case "<=":
						return boolH(c == -1 || c == 0)
					case ">=":
						return boolH(c == 1 || c == 0)
					case "==":
						return boolH(c == 0)
					}
					return boolH(c != 0)
				}
				if bigLeft {
					l = &jsPyFlo{f: rt.pyBigToF(bi)}
				} else {
					r = &jsPyFlo{f: rt.pyBigToF(bi)}
				}
			}
			switch op {
			case "+":
				// Two sequences CONCATENATE in Python ([1] + [2] is [1, 2], and
				// bytes are a list of byte numbers here), which is not what the
				// JavaScript + of the shared runtime does with two arrays.
				if la, ok := l.(*jsArray); ok {
					if ra, ok2 := r.(*jsArray); ok2 {
						out := &jsArray{}
						out.elems = append(out.elems, la.elems...)
						out.elems = append(out.elems, ra.elems...)
						return w(out)
					}
				}
				rt.pyAddCheck(l, r)
				if pyIsFlo(l) || pyIsFlo(r) {
					return w(&jsPyFlo{f: rt.toNumber(l) + rt.toNumber(r)})
				}
				return w(rt.jsAdd(l, r))
			case "*":
				// Python's SEQUENCE REPETITION: "ab" * 3 and [0] * 3 (either way
				// round) - the ordinary way to build a fixed-size list.
				if v, ok := pySeqRepeat(l, r); ok {
					return w(v)
				}
				if v, ok := pySeqRepeat(r, l); ok {
					return w(v)
				}
				if pyIsFlo(l) || pyIsFlo(r) {
					return w(&jsPyFlo{f: rt.toNumber(l) * rt.toNumber(r)})
				}
				return w(rt.pyArith(op[0], l, r))
			case "-":
				return w(rt.pyArith(op[0], l, r))
			case "/":
				// Python's `/` is TRUE division: it answers a float even from two
				// ints, so 4 / 2 is 2.0 and 10**30 / 1 is 1e+30.
				if rt.pyToF(r) == 0 {
					rt.pyZeroDiv()
				}
				// int / int with an arbitrary precision operand is ONE rounding of
				// the exact quotient, as CPython's long_true_divide.
				_, lBig := l.(*jsBigInt)
				_, rBig := r.(*jsBigInt)
				if lBig || rBig {
					if a, ok := pyBigOperand(l); ok {
						if b, ok := pyBigOperand(r); ok {
							return w(&jsPyFlo{f: rt.pyIntTrueDiv(a, b)})
						}
					}
				}
				return w(&jsPyFlo{f: rt.pyToF(l) / rt.pyToF(r)})
			case "//":
				if rt.toNumber(r) == 0 {
					rt.pyZeroDiv()
				}
				return w(pyIntOrFlo(math.Floor(rt.toNumber(l)/rt.toNumber(r)), l, r))
			case "%":
				// "fmt" % args is printf-style FORMATTING, not a remainder.
				if s, isStr := l.(string); isStr {
					return w(rt.pyPct(s, r))
				}
				if rt.toNumber(r) == 0 {
					rt.pyZeroDiv()
				}
				return w(pyIntOrFlo(pyFloorMod(rt.toNumber(l), rt.toNumber(r)), l, r))
			case "**":
				// Python's exponentiation. Only js_pybin (the Python compilers and
				// nothing else) ever passes this operator, and a user class already
				// had its __pow__ / __rpow__ chance above.
				// int ** NEGATIVE int is a float (2 ** -1 is 0.5); every other pair
				// of ints stays an int - and a Python int is ARBITRARY PRECISION,
				// so 10**30 is 1000000000000000000000000000000, not the double
				// 1e+30. Past 2^53 the double has lost digits, so the exact answer
				// comes from math/big; pyBigNarrow hands back a plain number whenever
				// it still fits, so nothing below 2^53 changes shape.
				e := rt.toNumber(r)
				if e < 0 {
					return w(&jsPyFlo{f: math.Pow(rt.toNumber(l), e)})
				}
				if !pyIsFlo(l) && !pyIsFlo(r) && e == math.Trunc(e) && e <= 4096 {
					if bl, ok := pyBigOperand(l); ok {
						return w(rt.pyBigNarrow(new(big.Int).Exp(bl, big.NewInt(int64(e)), nil)))
					}
				}
				return w(pyIntOrFlo(math.Pow(rt.toNumber(l), e), l, r))
			case "==":
				return boolH(rt.pyEqual(l, r))
			case "!=":
				return boolH(!rt.pyEqual(l, r))
			case "<", ">", "<=", ">=":
				if la, ok := l.(*jsArray); ok {
					if ra, ok := r.(*jsArray); ok {
						return w(rt.pySeqOrder(op, la.elems, ra.elems))
					}
				}
			}
			switch op {
			case "<":
				return boolH(rt.jsCompare(l, r) == -1)
			case ">":
				return boolH(rt.jsCompare(l, r) == 1)
			case "<=":
				c := rt.jsCompare(l, r)
				return boolH(c == -1 || c == 0)
			case ">=":
				c := rt.jsCompare(l, r)
				return boolH(c == 1 || c == 0)
			}
			rt.fail("unsupported operand type for %s: %s and %s", op, rt.typeOf(l), rt.typeOf(r))
			return 0
		},
		// dict(...): a fresh dict from a mapping (or from the keyword arguments,
		// which reach a signature-less builtin as one trailing dict).
		// dict(mapping_or_pairs=(), **kw), bound by its signature as [positionals,
		// keywords]; the source is read by update()'s own rules.
		"js_pydict": func(a []uint64) uint64 {
			el := u(a[0]).(*jsArray).elems
			pos := el[0].(*jsArray).elems
			if len(pos) > 1 {
				panic(&jsThrown{value: rt.pyExcInstance("TypeError",
					fmt.Sprintf("dict expected at most 1 argument, got %d", len(pos)))})
			}
			keys, vals := &jsArray{}, &jsArray{}
			if len(pos) == 1 {
				rt.pyDictUpdateFrom(keys, vals, pos[0])
			}
			if kk, kv, ok := dictParts(el[1]); ok {
				for i, k := range kk.elems {
					rt.pyDictStore(keys, vals, k, kv.elems[i])
				}
			}
			return w(&jsObject{props: map[string]interface{}{
				"__dict": true, "keys": keys, "vals": vals,
			}})
		},
		// max(...) / min(...), bound by their signature as [positionals, keywords].
		"js_pymax": func(a []uint64) uint64 { return rt.pyMinMax(u(a[0]), 1) },
		"js_pymin": func(a []uint64) uint64 { return rt.pyMinMax(u(a[0]), -1) },
		// The context-manager protocol. A context expression that is NOT a context
		// manager (no __enter__) binds its own value and runs the body, so a plain
		// value still works instead of aborting.
		"js_pyenter": func(a []uint64) uint64 {
			if _, cls, ok := pyInstance(u(a[0])); ok {
				if m, found := pyLookup(cls, "__enter__"); found && isCallable(m) {
					return w(rt.call(m, jsUndef, []interface{}{u(a[0])}))
				}
			}
			return a[0]
		},
		// __exit__(type, value, traceback) -> truthy swallows the exception.
		"js_pyexit": func(a []uint64) uint64 {
			if _, cls, ok := pyInstance(u(a[0])); ok {
				if m, found := pyLookup(cls, "__exit__"); found && isCallable(m) {
					e := u(a[1])
					return w(rt.call(m, jsUndef, []interface{}{u(a[0]), e, e, jsUndef}))
				}
			}
			return jsHFalse
		},
		// The Ellipsis singleton: `...` is a value of its own, and `x is ...` has to
		// hold, so there is exactly one of it per runtime.
		"js_pyellipsis": func(a []uint64) uint64 {
			if rt.pyEllipsis == nil {
				o := newJSObject()
				o.set("__name", "ellipsis")
				rt.pyEllipsis = o
			}
			return w(rt.pyEllipsis)
		},
		// del NAME: the binding stays, holding the deleted sentinel, so a later
		// read raises instead of falling through to an enclosing binding.
		"js_pydel_var": func(a []uint64) uint64 {
			sc := rt.scopeOf(a[0])
			name := rt.toString(u(a[1]))
			for s := sc; s != nil; s = s.parent {
				if s.has(name) {
					s.put(name, pyDeleted)
					return 0
				}
			}
			panic(&jsThrown{value: rt.pyExcInstance("NameError", "name '"+name+"' is not defined")})
		},
		// del obj[k] / del obj.name.
		"js_pydel_item": func(a []uint64) uint64 {
			if _, cls, isInst := pyInstance(u(a[0])); isInst { // A user class may define __delitem__.
				if m, found := pyLookup(cls, "__delitem__"); found && isCallable(m) {
					rt.call(m, jsUndef, []interface{}{u(a[0]), u(a[1])})
					return 0
				}
			}
			if keys, vals, ok := dictParts(u(a[0])); ok {
				i := rt.pyDictFind(keys, u(a[1]))
				if i < 0 {
					panic(&jsThrown{value: rt.pyExcInstanceV("KeyError", u(a[1]))})
				}
				keys.dropIdx()
				keys.elems = append(keys.elems[:i], keys.elems[i+1:]...)
				vals.elems = append(vals.elems[:i], vals.elems[i+1:]...)
				return 0
			}
			arr, ok := u(a[0]).(*jsArray)
			if !ok {
				rt.pyNoItemOp(u(a[0]), true)
			}
			i := rt.pySeqIdx(arr, u(a[1]))
			if i < 0 {
				i += len(arr.elems)
			}
			if i < 0 || i >= len(arr.elems) {
				panic(&jsThrown{value: rt.pyExcInstance("IndexError", "list assignment index out of range")})
			}
			arr.dropIdx()
			arr.elems = append(arr.elems[:i], arr.elems[i+1:]...)
			return 0
		},
		"js_pydel_attr": func(a []uint64) uint64 {
			if isCallable(u(a[0])) {
				name := rt.toString(u(a[1]))
				d, i := rt.pyFuncAttrIdx(u(a[0]), name)
				if i < 0 {
					panic(&jsThrown{value: rt.pyExcInstance("AttributeError",
						"'"+pyTypeName(rt, u(a[0]))+"' object has no attribute '"+name+"'")})
				}
				keys, vals, _ := dictParts(d)
				keys.dropIdx()
				keys.elems = append(keys.elems[:i], keys.elems[i+1:]...)
				vals.elems = append(vals.elems[:i], vals.elems[i+1:]...)
				return 0
			}
			if _, _, isInst := pyInstance(u(a[0])); !isInst {
				rt.pyNoDictAttr(u(a[0]), rt.toString(u(a[1])))
			}
			o, ok := u(a[0]).(*jsObject)
			if !ok {
				rt.fail("attribute deletion on a %s", rt.typeOf(u(a[0])))
			}
			name := rt.toString(u(a[1]))
			delete(o.props, name)
			for i, k := range o.keys {
				if k == name {
					o.keys = append(o.keys[:i], o.keys[i+1:]...)
					break
				}
			}
			return 0
		},
		// int(x=0, /, base=10), bound by its signature as [positionals, keywords].
		"js_pyint": func(a []uint64) uint64 {
			el := u(a[0]).(*jsArray).elems
			pos := el[0].(*jsArray).elems
			if keys, _, ok := dictParts(el[1]); ok && len(keys.elems) > 0 {
				pos = rt.pyKwPos("int", pos, el[1], []string{"x", "base"}, 1)
			}
			return w(rt.pyIntNew(pos))
		},
		// str(object='', encoding=..., errors=...), bound as [positionals, keywords].
		"js_pystrnew": func(a []uint64) uint64 {
			el := u(a[0]).(*jsArray).elems
			pos := el[0].(*jsArray).elems
			if keys, _, ok := dictParts(el[1]); ok && len(keys.elems) > 0 {
				// str(encoding=...) with no object is "" - nothing is decoded.
				if len(pos) == 0 && !rt.pyKwHas(el[1], "object") {
					rt.pyKwPos("str", []interface{}{""}, el[1], []string{"object", "encoding", "errors"}, 0)
					return rt.wrapStr("")
				}
				pos = rt.pyKwPos("str", pos, el[1], []string{"object", "encoding", "errors"}, 0)
			}
			if len(pos) == 0 {
				return rt.wrapStr("")
			}
			if len(pos) > 1 && (pos[1] != jsUndef || (len(pos) > 2 && pos[2] != jsUndef)) {
				return rt.wrapStr(rt.pyStrDecode(pos[0], pos[1]))
			}
			return pyExternMaps[rt]["js_pyrxstr"]([]uint64{w(pos[0])})
		},
		// Remember the name a `def` bound a closure under and hand the closure back,
		// so f.__name__ can answer it (a closure carries no name of its own).
		"js_pyfnname": func(a []uint64) uint64 { // (closure, name) -> closure
			if rt.pyFuncNames == nil {
				rt.pyFuncNames = map[interface{}]string{}
			}
			rt.pyFuncNames[u(a[0])] = rt.toString(u(a[1]))
			return a[0]
		},
		// Unary - / ~ with the __neg__ / __invert__ fallback.
		"js_pyunary": func(a []uint64) uint64 { // (operator, value)
			op := rt.toString(u(a[0]))
			v := u(a[1])
			name := "__neg__"
			if op == "~" {
				name = "__invert__"
			}
			if _, cls, ok := pyInstance(v); ok {
				if m, found := pyLookup(cls, name); found && isCallable(m) {
					return w(rt.call(m, jsUndef, []interface{}{v}))
				}
			}
			if op == "~" {
				// ~x is -x-1 in Python's infinite two's complement, at arbitrary
				// precision; the ToInt32 fallback is only for a non-int operand.
				if x, ok := pyBigOperand(v); ok {
					out := new(big.Int).Not(x)
					return w(rt.pyBigNarrow(out))
				}
				return rt.wrapNum(float64(^rt.toInt32(v)))
			}
			if re, im, ok := pyComplexParts(v); ok {
				return w(newPyComplex(-re, -im))
			}
			if rt.hasBigInt {
				if bi, ok := v.(*jsBigInt); ok {
					return w(&jsBigInt{v: new(big.Int).Neg(bi.v)})
				}
			}
			if f, ok := v.(*jsPyFlo); ok {
				return w(&jsPyFlo{f: -f.f}) // -0.0 keeps its sign, for repr(-0.0).
			}
			return rt.wrapNum(-rt.toNumber(v))
		},
		"js_pystr": func(a []uint64) uint64 { // str(v) with Python style rendering.
			return rt.wrapStr(rt.pyString(u(a[0])))
		},
		// ----- match statement support (the Python compilers only) -----
		// The class object of a BUILTIN type name, so `case str():` can ask
		// isinstance about a value that is not an instance of a user class.
		"js_pybuiltincls": func(a []uint64) uint64 {
			return w(rt.pyBuiltinClass(rt.toString(u(a[0]))))
		},
		// A sequence pattern matches lists and tuples (one type here), never a
		// string or a dict: (value, count before the *rest, count after it,
		// is there a *rest) -> does the shape fit.
		"js_pymatchseq": func(a []uint64) uint64 {
			arr, ok := u(a[0]).(*jsArray)
			if !ok {
				return boolH(false)
			}
			before, after := int(rt.toNumber(u(a[1]))), int(rt.toNumber(u(a[2])))
			if rt.truthy(u(a[3])) {
				return boolH(len(arr.elems) >= before+after)
			}
			return boolH(len(arr.elems) == before)
		},
		// A mapping pattern matches a dict (not a set) that holds every listed key.
		"js_pymatchmap": func(a []uint64) uint64 {
			if _, isSet := pySetElems(u(a[0])); isSet {
				return boolH(false)
			}
			keys, _, ok := dictParts(u(a[0]))
			if !ok {
				return boolH(false)
			}
			want, isArr := u(a[1]).(*jsArray)
			if !isArr {
				return boolH(false)
			}
			for _, k := range want.elems {
				if rt.pyDictFind(keys, k) < 0 {
					return boolH(false)
				}
			}
			return boolH(true)
		},
		// The **rest of a mapping pattern: a fresh dict of every entry whose key the
		// pattern did not name, in the subject's insertion order.
		"js_pymaprest": func(a []uint64) uint64 {
			keys, vals, ok := dictParts(u(a[0]))
			if !ok {
				rt.fail("js_pymaprest needs a dict")
			}
			want, _ := u(a[1]).(*jsArray)
			outK, outV := &jsArray{}, &jsArray{}
			for i, k := range keys.elems {
				skip := false
				if want != nil {
					for _, wk := range want.elems {
						if rt.pyEqual(wk, k) {
							skip = true
						}
					}
				}
				if !skip {
					dictAppend(outK, outV, k, vals.elems[i])
				}
			}
			return w(&jsObject{props: map[string]interface{}{
				"__dict": true, "keys": outK, "vals": outV,
			}})
		},
		// The POSITIONAL sub-values of a class pattern: (value, class, count) -> the
		// n attributes __match_args__ names, or undefined when the class cannot
		// provide them. A class WITHOUT __match_args__ accepts exactly one
		// positional pattern and hands the value itself over, which is what makes
		// `case str(sv):` capture the string.
		"js_pymatchargs": func(a []uint64) uint64 {
			n := int(rt.toNumber(u(a[2])))
			out := &jsArray{}
			names, hasNames := u(a[1]).(*jsObject)
			var ma interface{}
			if hasNames {
				if m, found := pyLookup(names, "__match_args__"); found {
					ma = m
				}
			}
			if ma == nil {
				if n == 1 {
					out.elems = append(out.elems, u(a[0]))
					return w(out)
				}
				return w(jsUndef)
			}
			elems := rt.pySeqElems(ma)
			if n > len(elems) {
				return w(jsUndef)
			}
			for i := 0; i < n; i++ {
				out.elems = append(out.elems, rt.pyGetAttr(u(a[0]), rt.toString(elems[i])))
			}
			return w(out)
		},
		// repr(v) / the !r conversion of an f-string field.
		// Lua's print separates its arguments with a TAB (js_pyprint uses a space,
		// which is Python's rule), and a Lua FLOAT renders as inf / -inf / nan and
		// always carries a fraction, as lua-interpreter.abnf does. Only the Lua
		// compiler emits these two.
		"js_luaprint": func(a []uint64) uint64 {
			args, ok := u(a[0]).(*jsArray)
			if !ok {
				rt.fail("js_luaprint needs an argument array")
			}
			out := ""
			for i, e := range args.elems {
				if i > 0 {
					out += "\t"
				}
				out += rt.toString(e)
			}
			fmt.Fprintln(outWriter, wtf8Clean(out))
			return 0
		},
		// A Lua string is a BYTE string, so lua-to-llvm-ir.abnf (like
		// lua-interpreter.abnf) holds one host character per byte: every code unit
		// of a Lua string is 0..255 and # / string.len / string.sub therefore count
		// and cut in bytes, which is what real Lua does. js_luaout is the one place
		// that model is undone again - the bytes are decoded as UTF-8 on the way to
		// the terminal, so a program prints the byte sequence it holds. A byte that
		// starts no valid sequence is passed through as its own code point, the
		// closest a UTF-8 terminal can come to the raw byte real Lua would write.
		// Only the Lua compiler emits this, and only around print.
		"js_luaout": func(a []uint64) uint64 {
			units := strUnits(rt.toString(u(a[0])))
			b := make([]byte, 0, len(units))
			for _, c := range units {
				b = append(b, byte(c))
			}
			out := make([]rune, 0, len(b))
			for i := 0; i < len(b); {
				if b[i] < utf8.RuneSelf {
					out = append(out, rune(b[i]))
					i++
					continue
				}
				r, n := utf8.DecodeRune(b[i:])
				if r == utf8.RuneError && n <= 1 {
					out = append(out, rune(b[i]))
					i++
					continue
				}
				out = append(out, r)
				i += n
			}
			return rt.wrapStr(string(out))
		},
		// Lua's 64-bit logical shift: value << k / value >> k over the two's
		// complement 64-bit integer, k >= 64 clears, a negative k reverses the
		// direction. (a[2] != 0 means "shift left".)
		"js_luashift": func(a []uint64) uint64 {
			v := uint64(int64(math.Trunc(rt.toNumber(u(a[0])))))
			k := int64(math.Trunc(rt.toNumber(u(a[1]))))
			left := a[2] != 0 // a RAW handle flag, not a wrapped value
			if k < 0 {
				k = -k
				left = !left
			}
			if k >= 64 {
				return rt.wrapNum(0)
			}
			if left {
				v = v << uint(k)
			} else {
				v = v >> uint(k)
			}
			return rt.wrapNum(float64(int64(v)))
		},
		"js_luaflt": func(a []uint64) uint64 {
			f := rt.toNumber(u(a[0]))
			if math.IsNaN(f) {
				return rt.wrapStr("nan")
			}
			if math.IsInf(f, 1) {
				return rt.wrapStr("inf")
			}
			if math.IsInf(f, -1) {
				return rt.wrapStr("-inf")
			}
			s := jsNumString(f)
			if f == math.Trunc(f) && !strings.ContainsAny(s, ".eE") {
				s += ".0"
			}
			return rt.wrapStr(s)
		},
		"js_pyrepr": func(a []uint64) uint64 {
			return rt.wrapStr(rt.pyRepr(u(a[0])))
		},
		// bool(x) / float(x) / sum(it): the conversions pyConvert of
		// python-interpreter.abnf has.
		"js_pybool": func(a []uint64) uint64 {
			v := u(a[0])
			if els, ok := pySetElems(v); ok {
				return boolH(len(els.elems) > 0)
			}
			if arr, ok := v.(*jsArray); ok {
				return boolH(len(arr.elems) > 0)
			}
			return boolH(rt.pyTruthyOf(v))
		},
		"js_pyfloat": func(a []uint64) uint64 {
			a = []uint64{w(pyUnwrapBuiltin(u(a[0])))}
			if s, isStr := u(a[0]).(string); isStr {
				return w(&jsPyFlo{f: pyStrToF(s)})
			}
			if bi, ok := u(a[0]).(*jsBigInt); ok {
				return w(&jsPyFlo{f: rt.pyBigToF(bi.v)})
			}
			return w(&jsPyFlo{f: rt.toNumber(u(a[0]))})
		},
		// The float LITERAL builder: constant folding cannot carry a box, so
		// languages/python-to-llvm-ir.abnf emits this call for 1.0 and 1e3.
		"js_pyflo": func(a []uint64) uint64 {
			return w(&jsPyFlo{f: rt.toNumber(u(a[0]))})
		},
		// abs() keeps the type: abs(-2) is the int 2, abs(-1.5) the float 1.5,
		// and abs() of an arbitrary precision int stays exact.
		"js_pyabs": func(a []uint64) uint64 {
			v := u(a[0])
			if re, im, ok := pyComplexParts(v); ok {
				return w(&jsPyFlo{f: rt.pyCxAbs(re, im)})
			}
			if f, ok := v.(*jsPyFlo); ok {
				return w(&jsPyFlo{f: math.Abs(f.f)})
			}
			if bi, ok := v.(*jsBigInt); ok {
				return w(&jsBigInt{v: new(big.Int).Abs(bi.v)})
			}
			return rt.wrapNum(math.Abs(rt.toNumber(v)))
		},
		// ord(c) / chr(n). Twins: hostGlobals["ord"]/["chr"] in
		// languages/python-interpreter.abnf, js_pyord/js_pychr in
		// languages/lib/python-rt.metajs and the declBuiltin pair in
		// languages/python-to-llvm-ir.abnf.
		//
		// A Go string is UTF-8 here while the other two engines hold UTF-16, so
		// the CODE POINT is what the three have in common: ord answers the single
		// rune's value (ord("\U0001F600") is 128512, not a surrogate) and chr
		// builds the rune. The domain errors use rt.fail with CPython's wording,
		// as every other builtin here reports one.
		"js_pyord": func(a []uint64) uint64 {
			v := u(a[0])
			s, isStr := v.(string)
			if !isStr {
				rt.fail("TypeError: ord() expected string of length 1, but %s found", pyTypeName(rt, v))
			}
			if n := utf8.RuneCountInString(s); n != 1 {
				rt.fail("TypeError: ord() expected a character, but string of length %d found", n)
			}
			r, _ := utf8.DecodeRuneInString(s)
			return rt.wrapNum(float64(r))
		},
		"js_pychr": func(a []uint64) uint64 {
			v := u(a[0])
			// A Python bool IS an int, so chr(True) is "\x01".
			if b, isB := v.(bool); isB {
				if b {
					return rt.wrapStr("\x01")
				}
				return rt.wrapStr("\x00")
			}
			if _, isNum := v.(float64); !isNum {
				rt.fail("TypeError: '%s' object cannot be interpreted as an integer", pyTypeName(rt, v))
			}
			n := math.Trunc(rt.toNumber(v))
			if n < 0 || n > 1114111 {
				rt.fail("ValueError: chr() arg not in range(0x110000)")
			}
			return rt.wrapStr(string(rune(int64(n))))
		},
		// ---- More python builtins. ----
		//
		// Each is bound by declBuiltin in languages/python-to-llvm-ir.abnf, has a
		// twin of the same js_py* name in languages/lib/python-rt.metajs, and a
		// hostGlobals entry in languages/python-interpreter.abnf. The variadic
		// ones (next, round, pow, zip, map, enumerate) are declared with argc 0,
		// which hands the extern the whole ARGUMENT ARRAY.
		//
		// iter(x): a generator is ITSELF an iterator in CPython and is answered
		// unchanged - which is what keeps `for x in iter(endless())` lazy;
		// everything else is materialized into a cursor. See pyMkIter.
		"js_pyiterfn": func(a []uint64) uint64 {
			v := u(a[0])
			if _, ok := v.(*jsGenerator); ok {
				return a[0]
			}
			if it, ok := rt.pyUserIter(v); ok {
				return w(it)
			}
			if pyIterName(rt, v) != "" {
				return a[0]
			}
			return w(rt.pyMkIter(pyIterKind(rt, v), rt.pyElemsOf(v)))
		},
		// The `yield from` RESUME GUARD, emitted once per delegated turn by
		// python-to-llvm-ir.abnf's makeYield just before it steps the delegate.
		// PEP 380 spells the resume out: `if _s is None: _y = _i.__next__()` else
		// `_y = _i.send(_s)`. A CURSOR has no send, so sending a non-None value
		// into a `yield from` over a list, a string or a dict raises
		// AttributeError INSIDE the generator, which ends it.
		// Twin: js_pyyfsend in languages/lib/python-rt.metajs.
		"js_pyyfsend": func(a []uint64) uint64 {
			sent := u(a[1])
			if isUndefOrNull(sent) {
				return jsHUndefined
			}
			if n := pyIterName(rt, u(a[0])); n != "" {
				panic(&jsThrown{value: rt.pyExcInstance("AttributeError",
					"'"+n+"' object has no attribute 'send'")})
			}
			return jsHUndefined
		},
		// The BUILTIN render table: print(len) is "<built-in function len>" and
		// print(map) is "<class 'map'>". A builtin has no def to record a name,
		// so the emitter registers one entry per builtin at boot.
		//
		// What is stored is a CODE - the builtin's name, with a leading "*" when
		// CPython prints it as a class - so the two-form split lives in the
		// emitter (pyClassBuiltin), in one place per half, instead of three
		// copies of the same list. It is a code rather than the finished text
		// because layer 2's copy of this table is in the GC heap (see the table in
		// languages/lib/python-rt.metajs). The def-name table rt.pyFuncNames is
		// left alone, so f.__name__ still reads exactly what a def recorded.
		"js_pybfn": func(a []uint64) uint64 { // (closure, code) -> closure
			if rt.pyBuiltinRender == nil {
				rt.pyBuiltinRender = map[interface{}]string{}
			}
			rt.pyBuiltinRender[u(a[0])] = rt.toString(u(a[1]))
			return a[0]
		},
		// next(it[, default]): the generator protocol, or one step of a cursor.
		"js_pynext": func(a []uint64) uint64 {
			args := rt.argArray(u(a[0]))
			it := argAt(args, 0)
			if g, ok := it.(*jsGenerator); ok {
				st, _ := g.step(rt, jsUndef).(*jsObject)
				if st == nil {
					rt.fail("generator step failed")
				}
				if done, _ := st.props["done"].(bool); done {
					return rt.pyStopIter(args, st.props["value"])
				}
				return w(st.props["value"])
			}
			if o, ok := it.(*jsObject); ok && pyIterName(rt, o) != "" {
				st := rt.pyItStep(o)
				if done, _ := st.props["done"].(bool); done {
					return rt.pyStopIter(args, jsUndef)
				}
				return w(st.props["value"])
			}
			panic(&jsThrown{value: rt.pyExcInstance("TypeError",
				"'"+pyTypeName(rt, it)+"' object is not an iterator")})
		},
		"js_pyenumerate": func(a []uint64) uint64 {
			args := rt.argArray(u(a[0]))
			start := 0.0
			if len(args) > 1 {
				start = rt.toNumber(args[1])
			}
			var out []interface{}
			for i, e := range rt.pyElemsOf(argAt(args, 0)) {
				out = append(out, &jsArray{elems: []interface{}{start + float64(i), e}})
			}
			return w(rt.pyMkIter("enumerate", out))
		},
		"js_pyzip": func(a []uint64) uint64 {
			args := rt.argArray(u(a[0]))
			var srcs [][]interface{}
			n := -1
			for _, s := range args {
				es := rt.pyElemsOf(s)
				srcs = append(srcs, es)
				if n < 0 || len(es) < n {
					n = len(es)
				}
			}
			if n < 0 {
				n = 0
			}
			var out []interface{}
			for i := 0; i < n; i++ {
				row := &jsArray{}
				for _, s := range srcs {
					row.elems = append(row.elems, s[i])
				}
				out = append(out, row)
			}
			return w(rt.pyMkIter("zip", out))
		},
		"js_pymapfn": func(a []uint64) uint64 {
			args := rt.argArray(u(a[0]))
			f := argAt(args, 0)
			var srcs [][]interface{}
			n := -1
			for _, s := range pyTailArgs(args) {
				es := rt.pyElemsOf(s)
				srcs = append(srcs, es)
				if n < 0 || len(es) < n {
					n = len(es)
				}
			}
			if n < 0 {
				n = 0
			}
			var out []interface{}
			for i := 0; i < n; i++ {
				var row []interface{}
				for _, s := range srcs {
					row = append(row, s[i])
				}
				out = append(out, rt.callPyValue(f, row, jsUndef))
			}
			return w(rt.pyMkIter("map", out))
		},
		// filter(f, xs): a None (or missing) function keeps the TRUTHY elements,
		// which is CPython's documented special case.
		"js_pyfilterfn": func(a []uint64) uint64 {
			f, src := u(a[0]), u(a[1])
			var out []interface{}
			for _, e := range rt.pyElemsOf(src) {
				keep := false
				if isUndefOrNull(f) {
					keep = rt.pyTruthyOf(e)
				} else {
					keep = rt.pyTruthyOf(rt.callPyValue(f, []interface{}{e}, jsUndef))
				}
				if keep {
					out = append(out, e)
				}
			}
			return w(rt.pyMkIter("filter", out))
		},
		"js_pyreversed": func(a []uint64) uint64 {
			src := rt.pyElemsOf(u(a[0]))
			out := make([]interface{}, 0, len(src))
			for i := len(src) - 1; i >= 0; i-- {
				out = append(out, src[i])
			}
			return w(rt.pyMkIter("reversed", out))
		},
		// sorted(iterable, /, *, key=None, reverse=False): list.sort on a new
		// list, bound by its signature as [positionals, keywords].
		"js_pysorted": func(a []uint64) uint64 {
			el := u(a[0]).(*jsArray).elems
			pos := el[0].(*jsArray).elems
			if len(pos) != 1 {
				panic(&jsThrown{value: rt.pyExcInstance("TypeError",
					fmt.Sprintf("sorted expected 1 argument, got %d", len(pos)))})
			}
			out := &jsArray{elems: append([]interface{}{}, rt.pyElemsOf(pos[0])...)}
			rt.pySortList(out, nil, el[1])
			return w(out)
		},
		"js_pyall": func(a []uint64) uint64 {
			for _, e := range rt.pyElemsOf(u(a[0])) {
				if !rt.pyTruthyOf(e) {
					return boolH(false)
				}
			}
			return boolH(true)
		},
		"js_pyany2": func(a []uint64) uint64 {
			for _, e := range rt.pyElemsOf(u(a[0])) {
				if rt.pyTruthyOf(e) {
					return boolH(true)
				}
			}
			return boolH(false)
		},
		// bin/hex/oct: CPython's 0b/0x/0o prefix, the sign OUTSIDE the prefix
		// (bin(-5) is '-0b101'), and arbitrary precision through math/big.
		"js_pybinstr": func(a []uint64) uint64 { return rt.wrapStr(rt.pyRadix(u(a[0]), 2, "0b")) },
		"js_pyhexstr": func(a []uint64) uint64 { return rt.wrapStr(rt.pyRadix(u(a[0]), 16, "0x")) },
		"js_pyoctstr": func(a []uint64) uint64 { return rt.wrapStr(rt.pyRadix(u(a[0]), 8, "0o")) },
		// divmod(a, b) is the PAIR (a // b, a % b) - floor division and the
		// floored remainder, so divmod(-7, 2) is (-4, 1). A tuple is a list here
		// (docs/working-on-this-project.md 7.9).
		// An exception that escaped the program: abort with CPython's last
		// traceback line, `Name: str(e)`, or the bare name when str(e) is empty.
		// Twin of pyUncaughtText in languages/python-interpreter.abnf and
		// js_pyuncaught in languages/lib/python-rt.metajs.
		"js_pyuncaught": func(a []uint64) uint64 {
			v := u(a[0])
			text := rt.pyString(v)
			if _, cls, ok := pyInstance(v); ok {
				if text == "" {
					text = rt.pyClsName(cls)
				} else {
					text = rt.pyClsName(cls) + ": " + text
				}
			}
			rt.fail("uncaught exception: %s", text)
			return 0
		},
		"js_pydivmod": func(a []uint64) uint64 {
			l, r := u(a[0]), u(a[1])
			if rt.toNumber(r) == 0 {
				rt.pyZeroDiv()
			}
			q := pyIntOrFlo(math.Floor(rt.toNumber(l)/rt.toNumber(r)), l, r)
			m := pyIntOrFlo(pyFloorMod(rt.toNumber(l), rt.toNumber(r)), l, r)
			return w(&jsArray{elems: []interface{}{q, m}})
		},
		// round(x[, n]): CPython rounds HALF TO EVEN, so round(2.5) is 2 and
		// round(0.5) is 0 - the one thing a Math.floor(x + 0.5) spelling gets
		// wrong, and it is wrong on every other integer. With no n the answer is
		// an INT; with an n it keeps the argument's type.
		"js_pyround": func(a []uint64) uint64 {
			args := rt.argArray(u(a[0]))
			v := argAt(args, 0)
			if len(args) < 2 || isUndefOrNull(args[1]) {
				return w(pyRoundHalfEven(rt.pyToF(v), 0))
			}
			nd := int(rt.toNumber(args[1]))
			r := pyRoundHalfEven(rt.pyToF(v), nd)
			if nd <= 0 && !pyIsFlo(v) {
				return w(r)
			}
			return w(&jsPyFlo{f: r})
		},
		// pow(a, b[, m]): the ** operator, plus CPython's three-argument MODULAR
		// form, which is exact through math/big.
		"js_pypow": func(a []uint64) uint64 {
			args := rt.argArray(u(a[0]))
			l, r := argAt(args, 0), argAt(args, 1)
			if len(args) > 2 && !isUndefOrNull(args[2]) {
				bl, ok1 := pyBigOperand(l)
				br, ok2 := pyBigOperand(r)
				bm, ok3 := pyBigOperand(args[2])
				if !ok1 || !ok2 || !ok3 {
					panic(&jsThrown{value: rt.pyExcInstance("TypeError",
						"pow() 3rd argument not allowed unless all arguments are integers")})
				}
				return w(rt.pyBigNarrow(new(big.Int).Exp(bl, br, bm)))
			}
			e := rt.toNumber(r)
			if e < 0 {
				return w(&jsPyFlo{f: math.Pow(rt.toNumber(l), e)})
			}
			if !pyIsFlo(l) && !pyIsFlo(r) && e == math.Trunc(e) && e <= 4096 {
				if bl, ok := pyBigOperand(l); ok {
					return w(rt.pyBigNarrow(new(big.Int).Exp(bl, big.NewInt(int64(e)), nil)))
				}
			}
			return w(pyIntOrFlo(math.Pow(rt.toNumber(l), e), l, r))
		},
		// callable(x): a function, a class object, or an instance whose class
		// defines __call__.
		"js_pycallable": func(a []uint64) uint64 {
			v := u(a[0])
			if isCallable(v) {
				return boolH(true)
			}
			if _, ok := pyClassObj(v); ok {
				return boolH(true)
			}
			if tc, ok := v.(*jsObject); ok && hasKey(tc, "__name") && !hasKey(tc, "__mro") {
				return boolH(true) // A builtin class object: int, str, ...
			}
			if _, cls, ok := pyInstance(v); ok {
				if m, found := pyLookup(cls, "__call__"); found && isCallable(m) {
					return boolH(true)
				}
			}
			return boolH(false)
		},
		// ascii(x) is repr(x) with every non-ASCII code point escaped as well.
		"js_pyascii": func(a []uint64) uint64 { return rt.wrapStr(pyAsciiOf(rt.pyRepr(u(a[0])))) },
		// getattr(o, n[, default]) / hasattr(o, n) / setattr(o, n, v), over the
		// same pyGetAttr / pySetAttr the `.` operator uses. A missing name
		// without a default raises AttributeError, which is what pyGetAttr
		// already does - so the default arm is the only new behaviour.
		"js_pygetattr3": func(a []uint64) uint64 {
			args := rt.argArray(u(a[0]))
			if len(args) > 2 {
				if !rt.pyHasAttr(argAt(args, 0), rt.toString(argAt(args, 1))) {
					return w(args[2])
				}
			}
			return w(rt.pyGetAttr(argAt(args, 0), rt.toString(argAt(args, 1))))
		},
		"js_pyhasattr": func(a []uint64) uint64 {
			return boolH(rt.pyHasAttr(u(a[0]), rt.toString(u(a[1]))))
		},
		// vars(o) IS o.__dict__ - CPython's own definition.
		"js_pyvars": func(a []uint64) uint64 {
			return w(rt.pyGetAttr(u(a[0]), "__dict__"))
		},
		"js_pysetattr3": func(a []uint64) uint64 {
			args := rt.argArray(u(a[0]))
			rt.pySetAttr(argAt(args, 0), rt.toString(argAt(args, 1)), argAt(args, 2))
			return jsHUndefined
		},
		// sum(iterable, /, start=0), bound by its signature as [positionals,
		// keywords]. A str start is refused outright.
		"js_pysum": func(a []uint64) uint64 {
			el := u(a[0]).(*jsArray).elems
			pos := el[0].(*jsArray).elems
			rt.pyKwOnly(el[1], "sum", "start", "start")
			start := rt.pyKwArg(el[1], "start")
			given := len(pos)
			if rt.pyKwHas(el[1], "start") {
				given++
			}
			if len(pos) == 0 {
				panic(&jsThrown{value: rt.pyExcInstance("TypeError",
					"sum() takes at least 1 positional argument (0 given)")})
			}
			if given > 2 {
				panic(&jsThrown{value: rt.pyExcInstance("TypeError",
					fmt.Sprintf("sum() takes at most 2 arguments (%d given)", given))})
			}
			if len(pos) == 2 {
				start = pos[1]
			}
			if _, isStr := start.(string); isStr {
				panic(&jsThrown{value: rt.pyExcInstance("TypeError",
					"sum() can't sum strings [use ''.join(seq) instead]")})
			}
			return w(rt.pySum(pos[0], start))
		},
		// format(v, spec, conv): the slice of Python's format mini-language an
		// f-string replacement field writes - [[fill]align][sign][0][width]
		// [.precision][type]. conv is "", "r", "s" or "a" (the !conversion).
		// Only the Python grammars ever emit this extern.
		"js_pyformat": func(a []uint64) uint64 {
			return rt.wrapStr(rt.pyFormat(u(a[0]), rt.toString(u(a[1])), rt.toString(u(a[2]))))
		},
		// The format(value[, spec]) builtin. Twin of js_pyformatfn in
		// languages/lib/python-rt.metajs and hostGlobals["format"] in
		// languages/python-interpreter.abnf.
		"js_pyformatfn": func(a []uint64) uint64 {
			args := rt.argArray(u(a[0]))
			spec := ""
			if len(args) > 1 {
				s, ok := args[1].(string)
				if !ok {
					panic(&jsThrown{value: rt.pyExcInstance("TypeError",
						"format() argument 2 must be str, not "+pyTypeName(rt, args[1]))})
				}
				spec = s
			}
			return rt.wrapStr(rt.pyFormat(argAt(args, 0), spec, ""))
		},
		// print(*objects, sep=' ', end='\n', file=None, flush=False). The closure
		// carries that signature (js_pysig), so the argument array arrives bound as
		// [sep, end, file, flush, objects, unknown keywords].
		"js_pyprint": func(a []uint64) uint64 {
			args, ok := u(a[0]).(*jsArray)
			if !ok || len(args.elems) != 6 {
				rt.fail("js_pyprint needs a bound argument array")
			}
			el := args.elems
			if keys, _, isDict := dictParts(el[5]); isDict && len(keys.elems) > 0 {
				panic(&jsThrown{value: rt.pyExcInstance("TypeError",
					"print() got an unexpected keyword argument '"+rt.toString(keys.elems[0])+"'")})
			}
			sep := rt.pyPrintText(el[0], "sep", " ")
			end := rt.pyPrintText(el[1], "end", "\n")
			objs, _ := el[4].(*jsArray)
			file := el[2]
			if file == jsUndef || file == jsNull {
				var sb strings.Builder
				for i, e := range objs.elems {
					if i > 0 {
						sb.WriteString(sep)
					}
					sb.WriteString(rt.pyString(e))
				}
				sb.WriteString(end)
				fmt.Fprint(outWriter, wtf8Clean(sb.String()))
				return 0
			}
			// Any other file is an object with a write method, called once per
			// piece as CPython does, then flush() when flush is true.
			for i, e := range objs.elems {
				if i > 0 {
					rt.pyMethodCall(file, "write", []interface{}{sep}, jsUndef)
				}
				rt.pyMethodCall(file, "write", []interface{}{rt.pyString(e)}, jsUndef)
			}
			rt.pyMethodCall(file, "write", []interface{}{end}, jsUndef)
			if rt.truthy(el[3]) {
				rt.pyMethodCall(file, "flush", []interface{}{}, jsUndef)
			}
			return 0
		},

		// Completion value tracking (see jsrt.retSlot).
		"js_setret": func(a []uint64) uint64 {
			rt.retSlot = u(a[0])
			return 0
		},
		"js_getret": func(a []uint64) uint64 {
			return w(rt.retSlot)
		},
		// js_bytelen: the UTF-8 BYTE length of a value's string form. PHP strings
		// are byte strings, so strlen("h<e-acute>llo") is 6 and not 5, while
		// js_pylen (code points) and .length (UTF-16 code units) both answer 5.
		// The host-side byteLen in this file computes the same number for the
		// emitters; this is its runtime twin.
		"js_bytelen": func(a []uint64) uint64 {
			return rt.wrapNum(float64(len(rt.toString(u(a[0])))))
		},
	}
	// The shared regular-expression engine (abnf/jsrtregex.go) adds its js_rx*
	// externs here. Strictly additive: it registers new names and rebinds none.
	rt.addRegexExterns(m)
	return m
}

// ----------------------------------------------------------------------------
// Standard host bindings and the module runner

func jsHostFunc(name string, fn func(rt *jsrt, this uint64, args []interface{}) interface{}) *hostFunc {
	return &hostFunc{name: name, fn: fn}
}

func (rt *jsrt) printArgs(args []interface{}) []interface{} {
	out := make([]interface{}, len(args))
	for i, a := range args {
		out[i] = rt.toGoNatural(a)
		if s, ok := out[i].(string); ok {
			out[i] = wtf8Clean(s) // A lone surrogate prints as U+FFFD, as in goja.
		}
	}
	return out
}

// jsMathRound is Math.round: the nearest integer, a tie going toward +Infinity, and
// a result of zero keeping the operand's sign. floor(x+0.5) is not it: the addition
// rounds 0.49999999999999994 up to 1, and floor(-0.4+0.5) is +0 where node says -0.
// x - floor(x) is exact, so comparing it with 0.5 is. Host id 17 of
// languages/lib/runtime.c is the same steps.
func jsMathRound(x float64) float64 {
	if math.IsNaN(x) || math.IsInf(x, 0) || x == 0 {
		return x
	}
	r := math.Floor(x)
	if x-r >= 0.5 {
		r++
	}
	if r == 0 && x < 0 {
		return math.Copysign(0, -1)
	}
	return r
}

// standardJSBindings are the host globals of a GRAMMAR TAG SCRIPT under the
// frozen engine (frozenBaseBindings, frozen.go), mirrored for goja in
// commonscript.go. It is ALSO the base programJSBindings (jsrtint.go) starts
// from - but not the set a MetaJS program ends up with: that one is the C
// floor's seed_root list, and programJSBindings removes the grammar-only names
// from this map and says which and why. abnf/hostglobals_test.go checks it.
func standardJSBindings() map[string]interface{} {
	mathObj := newJSObject()
	// ToInt32 is a MODULO, not a range test, and `int32(int64(f))` outside the
	// int64 range is implementation defined in Go - so Math.imul(1, 1e300) must
	// go through rt.toInt32 to answer node's 0 (the C floor's to_int32 does the
	// reduction with bit arithmetic).
	mathObj.set("imul", jsHostFunc("imul", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		a := rt.toInt32(argAt(args, 0))
		b := rt.toInt32(argAt(args, 1))
		return float64(a * b)
	}))
	mathObj.set("floor", jsHostFunc("floor", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return math.Floor(rt.toNumber(argAt(args, 0)))
	}))
	mathObj.set("abs", jsHostFunc("abs", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return math.Abs(rt.toNumber(argAt(args, 0)))
	}))
	// max/min are variadic like in JS: 0 arguments give -/+Infinity, and a NaN
	// argument wins (math.Max/Min propagate NaN).
	mathObj.set("max", jsHostFunc("max", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		acc := math.Inf(-1)
		for _, a := range args {
			acc = math.Max(acc, rt.toNumber(a))
		}
		return acc
	}))
	mathObj.set("min", jsHostFunc("min", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		acc := math.Inf(1)
		for _, a := range args {
			acc = math.Min(acc, rt.toNumber(a))
		}
		return acc
	}))
	// Single-argument functions map straight to Go's math package. goja implements
	// the same JS Math methods on top of the very same package, so the frozen VM
	// and goja agree bit for bit (and therefore format identically for -q output).
	addMath1 := func(name string, fn func(float64) float64) {
		mathObj.set(name, jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return fn(rt.toNumber(argAt(args, 0)))
		}))
	}
	addMath1("sqrt", math.Sqrt)
	addMath1("cbrt", math.Cbrt)
	addMath1("sin", math.Sin)
	addMath1("cos", math.Cos)
	addMath1("tan", math.Tan)
	addMath1("asin", math.Asin)
	addMath1("acos", math.Acos)
	addMath1("atan", math.Atan)
	addMath1("sinh", math.Sinh)
	addMath1("cosh", math.Cosh)
	addMath1("tanh", math.Tanh)
	addMath1("asinh", math.Asinh)
	addMath1("acosh", math.Acosh)
	addMath1("atanh", math.Atanh)
	addMath1("exp", math.Exp)
	addMath1("expm1", math.Expm1)
	addMath1("log", math.Log)
	addMath1("log2", math.Log2)
	addMath1("log10", math.Log10)
	addMath1("log1p", math.Log1p)
	addMath1("ceil", math.Ceil)
	addMath1("trunc", math.Trunc)
	addMath1("round", jsMathRound)
	addMath1("sign", func(x float64) float64 {
		if math.IsNaN(x) || x == 0 {
			return x // NaN stays NaN; +0 and -0 keep their sign.
		}
		if x < 0 {
			return -1
		}
		return 1
	})
	mathObj.set("pow", jsHostFunc("pow", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return math.Pow(rt.toNumber(argAt(args, 0)), rt.toNumber(argAt(args, 1)))
	}))
	mathObj.set("atan2", jsHostFunc("atan2", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return math.Atan2(rt.toNumber(argAt(args, 0)), rt.toNumber(argAt(args, 1)))
	}))
	// Math.hypot is variadic, so it cannot simply be math.Hypot - but it must not
	// be the naive sum of squares either: Math.hypot(1e300, 1e300) is
	// 1.4142135623730952e+300 in goja and in node, and Math.hypot(1e-300,
	// 1e-300) is 1.41...e-300. ECMA-262 asks implementations to avoid exactly
	// this overflow.
	//
	// Scaling by the largest magnitude first is what goja and V8 do, and for TWO
	// arguments it is bit for bit Go's own math.Hypot, because (max/max)*(max/max)
	// is exactly 1. The identical loop is host id 91 of languages/lib/runtime.c.
	// The special-case ORDER is ECMA-262's: an infinity wins over a NaN.
	mathObj.set("hypot", jsHostFunc("hypot", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		mx, sawInf, sawNaN := 0.0, false, false
		vals := make([]float64, len(args))
		for i, a := range args {
			v := rt.toNumber(a)
			vals[i] = v
			if math.IsInf(v, 0) {
				sawInf = true
			}
			if math.IsNaN(v) {
				sawNaN = true
			}
			if av := math.Abs(v); av > mx {
				mx = av
			}
		}
		switch {
		case sawInf:
			return math.Inf(1)
		case sawNaN:
			return math.NaN()
		case mx == 0:
			return 0.0
		}
		sum := 0.0
		for _, v := range vals {
			t := v / mx
			// `sum += t * t` would be FUSED into an FMA by the Go compiler on arm64 (and
			// not on amd64), one ulp away from the C floor's identical loop. Assigning
			// the product to a local does NOT stop it; only the explicit float64()
			// conversion the Go spec names as "explicitly rounded" does.
			sum += float64(t * t)
		}
		return mx * math.Sqrt(sum)
	}))
	// clz32 and fround are goja's Math, so -frozen must have them too. Host ids
	// 92 and 93 of languages/lib/runtime.c are the third engine.
	mathObj.set("clz32", jsHostFunc("clz32", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return float64(bits.LeadingZeros32(uint32(rt.toInt32(argAt(args, 0)))))
	}))
	mathObj.set("fround", jsHostFunc("fround", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return float64(float32(rt.toNumber(argAt(args, 0))))
	}))
	// Math.random: a uniform double in [0, 1), from a generator seeded per process,
	// as in node. Host id 96 of languages/lib/runtime.c is the native engine; under
	// goja installGojaMathSurface hands out this same function.
	mathObj.set("random", jsHostFunc("random", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return rand.Float64()
	}))
	mathObj.set("PI", math.Pi)
	mathObj.set("E", math.E)
	mathObj.set("LN2", math.Ln2)
	mathObj.set("LN10", math.Ln10)
	mathObj.set("LOG2E", math.Log2E)
	mathObj.set("LOG10E", math.Log10E)
	mathObj.set("SQRT2", math.Sqrt2)
	mathObj.set("SQRT1_2", 0.7071067811865476)

	stringObj := newJSObject()
	stringObj.set("fromCharCode", jsHostFunc("fromCharCode", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		units := make([]uint16, len(args))
		for i, a := range args {
			// ECMA ToUint16: NaN becomes 0 (via jsToInt) and larger values
			// wrap modulo 2^16 before the code unit becomes a rune.
			units[i] = uint16(jsToInt(rt.toNumber(a)))
		}
		return strFromUnits(units)
	}))

	arrayObj := newJSObject()
	arrayObj.set("isArray", jsHostFunc("isArray", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		switch argAt(args, 0).(type) {
		case *jsArray:
			return true
		case jsUndefT, jsNullT, bool, float64, string:
			return false
		}
		// Native Go slices (e.g. merged stacks) count as arrays too.
		rv := reflect.ValueOf(argAt(args, 0))
		if rv.Kind() == reflect.Ptr && !rv.IsNil() {
			rv = rv.Elem()
		}
		return rv.Kind() == reflect.Slice
	}))

	// Object.prototype.hasOwnProperty, for the hasOwn(o, n) idiom of the scripts.
	hasOwnFn := jsHostFunc("hasOwnProperty", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		key := rt.toString(argAt(args, 0))
		switch o := rt.unwrap(this).(type) {
		case *jsObject:
			_, ok := o.props[key]
			return ok
		case *jsArray:
			if key == "length" {
				return true
			}
			idx, err := strconv.Atoi(key)
			return err == nil && idx >= 0 && idx < len(o.elems)
		case string:
			if key == "length" {
				return true
			}
			idx, err := strconv.Atoi(key)
			return err == nil && idx >= 0 && idx < rt.strLen(o)
		default:
			rv := reflect.ValueOf(rt.unwrap(this))
			if rv.Kind() == reflect.Map {
				return rv.MapIndex(reflect.ValueOf(key)).IsValid()
			}
			return false
		}
	})
	objectProto := newJSObject()
	objectProto.set("hasOwnProperty", hasOwnFn)
	objectObj := newJSObject()
	objectObj.set("prototype", objectProto)

	return map[string]interface{}{
		"println": jsHostFunc("println", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			fmt.Fprintln(outWriter, rt.printArgs(args)...)
			return jsUndef
		}),
		"print": jsHostFunc("print", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			fmt.Fprint(outWriter, rt.printArgs(args)...)
			return jsUndef
		}),
		// eprintln is the DIAGNOSTIC channel (warnings): standard error, never
		// outWriter (which -pipe swaps to capture a stage's text) and never the
		// emitted module. Not silenced by the quiet flags - frozenBaseBindings
		// leaves it alone where it noops print/println/printf, exactly like the
		// goja binding in commonscript.go. The two hosts must stay identical.
		"eprintln": jsHostFunc("eprintln", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			fmt.Fprintln(warnWriter, rt.printArgs(args)...)
			return jsUndef
		}),
		"printf": jsHostFunc("printf", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			if len(args) == 0 {
				return jsUndef
			}
			fmt.Fprintf(outWriter, rt.toString(args[0]), rt.printArgs(args[1:])...)
			return jsUndef
		}),
		// fmt.Sprint's text: the same rendering println gives, returned instead of
		// printed. The Go compiler grammar binds fmt.Sprint to it.
		"sprint": jsHostFunc("sprint", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return fmt.Sprint(rt.printArgs(args)...)
		}),
		"sprintf": jsHostFunc("sprintf", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			if len(args) == 0 {
				return ""
			}
			return fmt.Sprintf(rt.toString(args[0]), rt.printArgs(args[1:])...)
		}),
		// The UTF-8 byte length of a string: .length counts UTF-16 code units,
		// but the emitters need the byte count of the char arrays they emit
		// (lib/compile-core.js emitStr).
		"byteLen": jsHostFunc("byteLen", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return float64(len(rt.toString(argAt(args, 0))))
		}),
		// unicodeCase(s, mode): upper or lower case, simple or full (abnf/casemap.go).
		// Host id 97 in languages/lib/runtime.c; installGojaCaseMapping for goja.
		"unicodeCase": jsHostFunc("unicodeCase", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return caseMapMode(rt.toString(argAt(args, 0)), int(rt.toNumber(argAt(args, 1))))
		}),
		// rawSet writes an own property; the goja side bypasses the
		// Object.prototype "__proto__" accessor with it, here a plain
		// property write is already exactly that.
		"rawSet": jsHostFunc("rawSet", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			obj, ok := argAt(args, 0).(*jsObject)
			if !ok {
				rt.fail("rawSet needs an object")
			}
			obj.props[rt.toString(argAt(args, 1))] = argAt(args, 2)
			return jsUndef
		}),
		"parseInt": jsHostFunc("parseInt", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			// A missing radix is NaN here; jsToInt turns it into 0, which
			// jsParseInt treats as "auto" (10, or 16 for an 0x prefix). A raw
			// int(NaN) is minInt64 on amd64.
			return jsParseInt(rt.toString(argAt(args, 0)), jsToInt(rt.toNumber(argAt(args, 1))))
		}),
		"parseFloat": jsHostFunc("parseFloat", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return jsParseFloat(rt.toString(argAt(args, 0)))
		}),
		"exit": jsHostFunc("exit", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			os.Exit(int(int32(jsToInt(rt.toNumber(argAt(args, 0))))))
			return jsUndef
		}),
		// Standard numeric globals (goja has them natively; the frozen VM did not).
		"Infinity": math.Inf(1),
		"NaN":      math.NaN(),
		"Math":     mathObj,
		// The Java subset's Math (abs/max/min only, and double-aware); the Java and
		// Kotlin compiler grammars bind their `Math` to it. See jsrtjvm.go.
		"__jmath": jvmMathObject(),
		"String":  stringObj,
		"Object":  objectObj,
		"Array":   arrayObj,
		// The declaration marker: var v = anytype declares v as never pinning.
		// frozenBaseBindings inherits it, so tag scripts get it under -frozen too.
		"anytype": jsAnytype,
	}
}

// jsParseInt implements the JS parseInt() prefix parsing.
func jsParseInt(s string, radix int) float64 {
	s = strings.TrimSpace(s)
	sign := 1.0
	if strings.HasPrefix(s, "-") {
		sign = -1
		s = s[1:]
	} else if strings.HasPrefix(s, "+") {
		s = s[1:]
	}
	if radix == 0 {
		radix = 10
		if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
			radix = 16
			s = s[2:]
		}
	} else if radix == 16 && (strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X")) {
		s = s[2:]
	}
	val := 0.0
	digits := 0
	for i := 0; i < len(s); i++ {
		d := digitValue(s[i])
		if d < 0 || d >= radix {
			break
		}
		val = val*float64(radix) + float64(d)
		digits++
	}
	if digits == 0 {
		return math.NaN()
	}
	return sign * val
}

func digitValue(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'z':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'Z':
		return int(c-'A') + 10
	default:
		return -1
	}
}

// pyStrToF is float(str) - CPython's float_from_string, in the one respect where
// it is not JavaScript's parseFloat: it accepts the three non-finite SPELLINGS
// "inf", "infinity" and "nan", case-insensitively, with an optional sign, after
// stripping whitespace (goja's parseFloat would take "Infinity" but not "inf";
// jsParseFloat takes neither).
//
// Everything else still goes to jsParseFloat, so float("1.5abc") is still the
// prefix parse 1.5 where CPython raises ValueError - a known looseness shared by
// all three engines. float("-nan") is nan in CPython, so the sign is dropped there.
//
// pyStrToF in languages/python-interpreter.abnf and languages/lib/python-rt.metajs
// are the same body, with the infinity BUILT as Math.pow(2, 1024) because the
// script dialect has no exponent form in a numeric literal.
func pyStrToF(s string) float64 {
	s = pyDecAscii(s)
	t := strings.TrimSpace(s)
	neg := false
	if len(t) > 0 && (t[0] == '+' || t[0] == '-') {
		neg = t[0] == '-'
		t = t[1:]
	}
	switch strings.ToLower(t) {
	case "inf", "infinity":
		if neg {
			return math.Inf(-1)
		}
		return math.Inf(1)
	case "nan":
		return math.NaN()
	}
	return jsParseFloat(s)
}

// jsParseFloat implements the JS parseFloat() prefix parsing.
func jsParseFloat(s string) float64 {
	s = strings.TrimSpace(s)
	end := 0
	seenDigit := false
	seenDot := false
	seenExp := false
	for end < len(s) {
		c := s[end]
		switch {
		case c >= '0' && c <= '9':
			seenDigit = true
		case c == '.' && !seenDot && !seenExp:
			seenDot = true
		case (c == 'e' || c == 'E') && seenDigit && !seenExp:
			seenExp = true
			if end+1 < len(s) && (s[end+1] == '+' || s[end+1] == '-') {
				end++
			}
		case (c == '+' || c == '-') && end == 0:
			// Leading sign.
		default:
			goto done
		}
		end++
	}
done:
	if !seenDigit {
		return math.NaN()
	}
	f, err := strconv.ParseFloat(strings.TrimRight(s[:end], "eE+-."), 64)
	if err != nil {
		// An OUT OF RANGE literal is not a parse failure. Go answers ±Inf on
		// overflow and ±0 on underflow and reports ErrRange alongside; JavaScript
		// answers exactly those values and reports nothing. Returning NaN here was
		// a live FROZEN-DIFF in the dialect itself - parseFloat("1e400") was NaN
		// under -frozen (this function) and Infinity under goja (its own
		// parseFloat) - which any test writing an overflow literal would hit.
		if ne, isNum := err.(*strconv.NumError); isNum && ne.Err == strconv.ErrRange {
			return f
		}
		return math.NaN()
	}
	return f
}

// callEntry runs a (env, args) MetaJS entry function and returns the value handle.
func (rt *jsrt) callEntry(ma *machine, name string, env uint64) uint64 {
	if !rt.traced {
		return ma.callByName(name, []uint64{env, rt.wrap(&jsArray{})})
	}
	traceEmit(&TraceEvent{Ev: "call", Depth: rt.traceDepth, Name: name})
	rt.traceDepth++
	h := ma.callByName(name, []uint64{env, rt.wrap(&jsArray{})})
	rt.traceDepth--
	traceEmit(&TraceEvent{Ev: "ret", Depth: rt.traceDepth, Line: lineOfPos(rt.curPos), Val: rt.traceVal(rt.unwrap(h))})
	return h
}

// isInt32Value reports whether f is an integral value inside the int32 range -
// the same test the interpreter grammars express as (v | 0) == v.
func isInt32Value(f float64) bool {
	return f == math.Trunc(f) && f >= math.MinInt32 && f <= math.MaxInt32
}

// isTypeName is the dynamic type test behind the typed languages' `is` /
// `instanceof` checks (extern js_is_type): a value-model test by type NAME.
// Generic arguments are ignored (List<Int> tests as List), a trailing ? lets
// null match, and user classes match on the __class descriptor's __name (the
// __super chain and each __ifaces list are walked; a List/Collection name also
// matches a class implementing it). Integral numbers count as Int AND as
// Double - the value model has one number type, so `3.0 is Int` is true here
// although real Kotlin would say false for a Double-typed 3.0. The interpreter
// grammars carry a JS twin (rtIsType in lib/interp-core.js) - keep in sync.
func (rt *jsrt) isTypeName(v interface{}, t string) bool {
	if i := strings.IndexByte(t, '<'); i >= 0 {
		t = t[:i]
	}
	opt := false
	if strings.HasSuffix(t, "?") {
		t = t[:len(t)-1]
		opt = true
	}
	switch v.(type) {
	case nil, jsUndefT, jsNullT:
		return opt
	}
	// Kotlin's Char is its own type: `c is Char` holds, `c is Int` does not. Tested
	// before the name switch below, whose "Char" case answers for the plain numbers
	// the other languages use (Java/C# Character, Go rune).
	if _, isChar := v.(jsChar); isChar {
		return t == "Char" || t == "Character" || t == "Any" || t == "Comparable"
	}
	switch t {
	case "Any", "Object":
		return true
	case "Int", "Integer", "Long", "Short", "Byte", "Char", "Character":
		f, ok := v.(float64)
		return ok && f == math.Trunc(f)
	case "Double", "Float", "Number":
		_, ok := v.(float64)
		return ok
	case "String", "CharSequence":
		_, ok := v.(string)
		return ok
	case "Boolean":
		_, ok := v.(bool)
		return ok
	case "List", "MutableList", "Collection", "Array":
		if _, ok := v.(*jsArray); ok {
			return true
		}
	}
	if o, ok := v.(*jsObject); ok {
		cls, _ := o.props["__class"].(*jsObject)
		return classIsNamed(cls, t)
	}
	return false
}

// classIsNamed walks a class descriptor's __super chain and, at each step, every
// descriptor in its __ifaces list (the interfaces it implements or extends).
func classIsNamed(cls *jsObject, t string) bool {
	for cls != nil {
		if n, _ := cls.props["__name"].(string); n == t {
			return true
		}
		if ifs, ok := cls.props["__ifaces"].(*jsArray); ok {
			for _, e := range ifs.elems {
				if ic, ok := e.(*jsObject); ok && classIsNamed(ic, t) {
					return true
				}
			}
		}
		sup, _ := cls.props["__super"].(*jsObject)
		cls = sup
	}
	return false
}

// jsToInt converts a JS number to a Go int for host-side argument positions:
// NaN becomes 0 (a plain int(f) of NaN is architecture specific in Go - 0 on
// arm64 but minInt64 on amd64), and out-of-range values saturate.
func jsToInt(f float64) int {
	if f != f {
		return 0
	}
	if f >= math.MaxInt64 {
		return math.MaxInt64
	}
	if f <= math.MinInt64 {
		return math.MinInt64
	}
	return int(f)
}

// toInt32 converts a JS value to an int32 like the JS ToInt32 operation.
func (rt *jsrt) toInt32(v interface{}) int32 {
	f := rt.toNumber(v)
	if f != f || math.IsInf(f, 0) {
		return 0
	}
	f = math.Trunc(f)
	if f >= -9223372036854775808 && f < 9223372036854775808 {
		return int32(int64(f))
	}
	// Outside the int64 range `int64(f)` is IMPLEMENTATION DEFINED in Go (it
	// answers -1 on arm64 and math.MinInt64 on amd64), and ToInt32 is a
	// MODULO rather than a range test: `1e20 | 0` is 1661992960 and `1e19 | 0`
	// is -1981284352. math.Mod is fmod, which is exact for an integral operand,
	// so the reduction loses nothing. languages/lib/runtime.c's to_int32 does
	// the same thing with bit arithmetic.
	m := math.Mod(f, 4294967296)
	if m < 0 {
		m += 4294967296
	}
	if m >= 2147483648 {
		m -= 4294967296
	}
	return int32(m)
}

// jsProgramPanic marks a failure of the RUNNING PROGRAM (llvm.RunJS) as opposed
// to a failure of the tag script that started it. The distinction matters only
// for the diagnostic: the frozen engine wraps a panic escaping a tag with that
// tag's whole source text, while goja wraps only errors goja itself returns and
// lets a Go panic through untouched. The marker lets the frozen engine pass a
// program failure through unwrapped, so both engines report it identically -
// which the matrix checks on stderr. Error() returns the message verbatim.
type jsProgramPanic struct{ msg string }

func (e jsProgramPanic) Error() string  { return e.msg }
func (e jsProgramPanic) String() string { return e.msg }

// excText is the text of an exception nobody caught, and the twin of exc_text in
// languages/lib/runtime.c. String(o) of an object is "[object Object]" - correct
// for String() and useless here - so the two shapes the exception hierarchies in
// this project use are recognised first:
//
//	{name, message}   the js/ts Error instance      -> "TypeError: boom"
//	{__class, args}   the python / ruby style one   -> str(e)
func (rt *jsrt) excText(v interface{}) string {
	if o, ok := v.(*jsObject); ok {
		if mv, has := o.props["message"]; has {
			if msg, isStr := mv.(string); isStr {
				name := ""
				if nv, hasN := o.props["name"]; hasN {
					if n, isStr := nv.(string); isStr {
						name = n
					}
				}
				return jsErrText(name, msg)
			}
		}
		if av, has := o.props["args"]; has {
			if arr, isArr := av.(*jsArray); isArr && len(arr.elems) > 0 {
				return rt.toString(arr.elems[0])
			}
		}
	}
	return rt.jsvString(v)
}

// runJSModule is llvm.RunJS(): it executes the entry function of a MetaJS
// module with the standard host bindings and returns its int32 result.
// module with the standard host bindings and returns its int32 result.
// This is the program runtime, so the -cfgraph and -trace hooks live here.
func runJSModule(m *ir.Module, entry string) *RunResult {
	maybeDumpCFG(m)
	maybeDumpCallgraph(m)
	rt := newJSRT(programJSBindings())
	rt.enableTrace()
	ma := rt.attach(m)
	// An exception that escapes the program's entry point is an uncaught throw;
	// report it like any other runtime error (the same wording rt.fail gives, so
	// the caller's recover turns it into a clean message and a non-zero exit).
	// Everything that escapes here is re-panicked as a jsProgramPanic, so the
	// tag-script engines can tell "the program failed" from "the tag script
	// failed" and diagnose them identically (see the type).
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		if _, ok := r.(jsProgramPanic); ok {
			panic(r)
		}
		if exc, ok := r.(*jsThrown); ok {
			// excText, not jsvString: String(o) of an object is "[object Object]",
			// which says nothing about a python-style {__class, args} instance.
			// See excText.
			panic(jsProgramPanic{"js runtime error: uncaught exception: " + rt.excText(exc.value)})
		}
		panic(jsProgramPanic{fmt.Sprint(r)})
	}()
	h := rt.callEntry(ma, entry, 0)
	return &RunResult{Ret: uint32(rt.toInt32(rt.unwrap(h))), Out: ""}
}

// ----------------------------------------------------------------------------
// Python's float: a boxed double, so that int and float stay two types
//
// Python's int and float print differently (1 vs 1.0), answer different type()s,
// and `/` always makes a float even out of two ints - and one double cannot
// carry that, because 2.0 and 2 ARE the same double. So the float is the boxed
// one (like dart's jsDartFlo and ruby's jsFlo) and the int stays the plain
// float64, which leaves the common integer path untouched.
//
// Only languages/python-to-llvm-ir.abnf ever creates one (js_pyflo), so every
// branch added for this type is unreachable from the other fifteen languages.
// The twin implementations are pyFlo/floStr in languages/python-interpreter.abnf
// and pyBFlo/pyBFloStr in languages/lib/python-rt.metajs; ./test.sh --cross
// diffs the three, so they change together or not at all.
// A POINTER, so that the box has IDENTITY: `x = nan; x is x` is true in CPython
// (it is one object) while `float('nan') is float('nan')` is false, and a value
// struct could express neither, since NaN != NaN. python-interpreter.abnf and
// python-rt.metajs get the same rule for free, their box being an object.
type jsPyFlo struct{ f float64 }

func pyIsFlo(v interface{}) bool { _, ok := v.(*jsPyFlo); return ok }

// pyToF is toNumber for the operands of `/`. A Python bigint is an arbitrary
// precision INT, so 10**28 / 1 has to be the float 1e+28 - where rt.toNumber
// answers NaN for one, matching JavaScript, in which a BigInt is not a number.
func (rt *jsrt) pyToF(v interface{}) float64 {
	if bi, ok := v.(*jsBigInt); ok {
		f, _ := new(big.Float).SetInt(bi.v).Float64()
		return f
	}
	return rt.toNumber(v)
}

// pyIntTrueDiv is int / int, the exact quotient rounded once to the nearest
// double (big.Rat rounds half to even, subnormals included); a quotient past the
// double range is CPython's OverflowError. b is not zero. Twin of bigTrueDiv in
// languages/python-interpreter.abnf and pyIntTrueDiv in
// languages/lib/python-rt.metajs.
func (rt *jsrt) pyIntTrueDiv(a, b *big.Int) float64 {
	f, _ := new(big.Rat).SetFrac(a, b).Float64()
	if math.IsInf(f, 0) {
		panic(&jsThrown{value: rt.pyExcInstance("OverflowError", "integer division result too large for a float")})
	}
	if f == 0 && (a.Sign() < 0) != (b.Sign() < 0) {
		return math.Copysign(0, -1)
	}
	return f
}

// pyIntOrFlo is the result type of // and %: an int from two ints (7 // 2 is 3)
// and a float as soon as either operand is one (7.5 // 2 is 3.0).
func pyIntOrFlo(f float64, l, r interface{}) interface{} {
	if pyIsFlo(l) || pyIsFlo(r) {
		return &jsPyFlo{f: f}
	}
	return f
}

// pyIsNumeric is "this value has a double in it": a float box, a plain int, a
// bool (True is 1) or an arbitrary precision integer.
func pyIsNumeric(v interface{}) bool {
	switch v.(type) {
	case *jsPyFlo, float64, bool, *jsBigInt:
		return true
	}
	return false
}

// pyNumEq compares two numeric values BY VALUE across the int/float/bool split,
// which is what Python's == and its dict keys need (1 == 1.0 == True).
func (rt *jsrt) pyNumEq(a, b interface{}) bool {
	if !pyIsNumeric(a) || !pyIsNumeric(b) {
		return false
	}
	return rt.toNumber(a) == rt.toNumber(b)
}

// pyClassName answers the name of a Python TYPE object - a user class (it
// carries an __mro) or one of the builtin type objects pyBuiltinClass hands out.
// The Ellipsis singleton also carries a __name and is deliberately not one.
func (rt *jsrt) pyClassName(v interface{}) (string, bool) {
	o, ok := v.(*jsObject)
	if !ok {
		return "", false
	}
	name, ok := o.props["__name"].(string)
	if !ok {
		return "", false
	}
	if _, has := o.props["__mro"]; has {
		return name, true
	}
	if rt.pyTypes != nil && rt.pyTypes[name] == o {
		return name, true
	}
	return "", false
}

// ----- repr(float), which in Python 3 is also str(float) -----
//
// Take the SHORTEST round-tripping digit string and the decimal point position
// `decpt`, then use exponential form iff `decpt <= -4 || decpt > 16`. In
// exponential form there is NO forced ".0" (2e16 is "2e+16", unlike ruby's
// rubyFloStr and java's jvmFloStr), the exponent is signed and zero padded to
// two digits ("1e-05"), and ONE significant digit is allowed ("5e-324", where
// java forces two). Non-finite is lowercase, and repr(-float('nan')) is "nan".
// Against CPython 3.14:
//
//	1e15 -> 1000000000000000.0    1e16 -> 1e+16      2e16   -> 2e+16
//	1e-4 -> 0.0001                1e-5 -> 1e-05      5e-324 -> 5e-324
//	1.0  -> 1.0                   -0.0 -> -0.0       1e400  -> inf
//
// strconv.FormatFloat(f, 'e', -1, 64) is exactly the shortest-round-trip pair
// (digits, exponent) the two script engines get out of their number-to-string.
func pyFloStr(f float64) string {
	if math.IsNaN(f) {
		return "nan"
	}
	if math.IsInf(f, 1) {
		return "inf"
	}
	if math.IsInf(f, -1) {
		return "-inf"
	}
	if f == 0 {
		if math.Signbit(f) {
			return "-0.0"
		}
		return "0.0"
	}
	if f < 0 {
		return "-" + pyFloDigits(-f)
	}
	return pyFloDigits(f)
}

// pyFloDigits renders f > 0.
func pyFloDigits(f float64) string {
	s := strconv.FormatFloat(f, 'e', -1, 64)
	ei := strings.IndexByte(s, 'e')
	mant, exp := s[:ei], s[ei+1:]
	e10, _ := strconv.Atoi(exp)
	digits := strings.Replace(mant, ".", "", 1)
	decpt := e10 + 1
	if decpt <= -4 || decpt > 16 {
		out := digits[0:1]
		if len(digits) > 1 {
			out += "." + digits[1:]
		}
		sign, ex := "+", e10
		if ex < 0 {
			sign, ex = "-", -ex
		}
		if ex < 10 {
			return out + "e" + sign + "0" + strconv.Itoa(ex)
		}
		return out + "e" + sign + strconv.Itoa(ex)
	}
	if decpt <= 0 {
		return "0." + strings.Repeat("0", -decpt) + digits
	}
	if decpt >= len(digits) {
		return digits + strings.Repeat("0", decpt-len(digits)) + ".0"
	}
	return digits[:decpt] + "." + digits[decpt:]
}

// pyEqual is Python's ==: containers compare by VALUE, recursively. A list equals a
// list of the same length whose elements are all equal; a dict equals a dict with the
// same key set and equal values for every key (insertion order does not matter, as in
// Python). Everything else - numbers, strings, booleans, None, functions, instances -
// falls back to the strict equality the rest of the runtime uses, so identity still
// decides where Python says it should. Cycles are not expected in this value model and
// are not guarded against.
func (rt *jsrt) pyEqual(x, y interface{}) bool {
	if xs, ok := pySetElems(x); ok {
		ys, ok2 := pySetElems(y)
		if !ok2 || len(xs.elems) != len(ys.elems) {
			return false
		}
		for _, e := range xs.elems {
			if !rt.pySetHas(ys, e) {
				return false
			}
		}
		return true
	}
	if _, ok := pySetElems(y); ok {
		return false
	}
	if xa, ok := x.(*jsArray); ok {
		ya, ok2 := y.(*jsArray)
		if !ok2 || len(xa.elems) != len(ya.elems) {
			return false
		}
		for i := range xa.elems {
			if !rt.pyEqual(xa.elems[i], ya.elems[i]) {
				return false
			}
		}
		return true
	}
	if xk, xv, ok := dictParts(x); ok {
		yk, yv, ok2 := dictParts(y)
		if !ok2 || len(xk.elems) != len(yk.elems) {
			return false
		}
		for i, k := range xk.elems {
			j := rt.pyDictFind(yk, k)
			if j < 0 || !rt.pyEqual(xv.elems[i], yv.elems[j]) {
				return false
			}
		}
		return true
	}
	// A Python bool IS an int - `class bool(int)` - so True == 1, True == 1.0,
	// False == 0 and False == -0.0 all hold, and so does every container equality
	// built on them ([True] == [1], {True: 1} == {1: 1}). Reading the bool as its
	// int value HERE, below the container arms and above the numeric ones, covers
	// the bigint, the complex and the plain-number arm in one place; True == 2
	// stays false because 1.0 != 2.0, and True == "x" stays false because
	// strictEq's last arm sees a number against a string.
	if b, ok := x.(bool); ok {
		x = pyBoolInt(b)
	}
	if b, ok := y.(bool); ok {
		y = pyBoolInt(b)
	}
	// A float equals the int of the same value (1.0 == 1) and two boxes holding
	// the same double are equal though they are not the same box. Above the
	// bigint and complex arms, which read the box through rt.toNumber anyway.
	if pyIsFlo(x) || pyIsFlo(y) {
		if bi, f, _, ok := pyBigFloPair(x, y); ok {
			return pyCmpBigFloat(bi, f) == 0
		}
		return rt.pyNumEq(x, y)
	}
	// Python compares an int to a float BY VALUE, and an arbitrary precision int
	// (a literal too large for a double, see js_bigint) is still an int.
	if bx, ok := x.(*jsBigInt); ok {
		if by, ok2 := y.(*jsBigInt); ok2 {
			return bx.v.Cmp(by.v) == 0
		}
		if fy, ok2 := y.(float64); ok2 {
			return bigEqFloat(bx.v, fy)
		}
	}
	if by, ok := y.(*jsBigInt); ok {
		if fx, ok2 := x.(float64); ok2 {
			return bigEqFloat(by.v, fx)
		}
	}
	if re, im, ok := pyComplexParts(x); ok {
		if re2, im2, ok2 := pyComplexParts(y); ok2 {
			return re == re2 && im == im2
		}
		return im == 0 && re == rt.toNumber(y)
	}
	if re, im, ok := pyComplexParts(y); ok {
		return im == 0 && re == rt.toNumber(x)
	}
	return rt.strictEq(x, y)
}

// pyBoolInt is the int a Python bool IS: True is 1, False is 0.
func pyBoolInt(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

// bigEqFloat is an arbitrary precision int against a double, by value: only an
// integral, finite double can equal one.
func bigEqFloat(b *big.Int, f float64) bool {
	if f != math.Trunc(f) || math.IsInf(f, 0) || f != f {
		return false
	}
	fi, _ := big.NewFloat(f).Int(nil)
	return b.Cmp(fi) == 0
}

// pyBigToF is an arbitrary precision int as the nearest double (half to even),
// which float(n) and every int-meets-float operator use; past the double range
// CPython raises OverflowError rather than answering an infinity. Twin of
// pyBigToF in languages/lib/python-rt.metajs and bigToFlo in
// languages/python-interpreter.abnf.
func (rt *jsrt) pyBigToF(b *big.Int) float64 {
	f, _ := new(big.Float).SetInt(b).Float64()
	if math.IsInf(f, 0) {
		panic(&jsThrown{value: rt.pyExcInstance("OverflowError", "int too large to convert to float")})
	}
	return f
}

// pyCmpBigFloat compares an int with a double EXACTLY, as CPython does, so
// 10**23 != float(10**23): -1, 0 or 1, and 2 when the double is a NaN (every
// relation false).
func pyCmpBigFloat(b *big.Int, f float64) int {
	switch {
	case f != f:
		return 2
	case math.IsInf(f, 1):
		return -1
	case math.IsInf(f, -1):
		return 1
	}
	return new(big.Float).SetInt(b).Cmp(big.NewFloat(f))
}

// pyFloatToInt is int(f) for a double: truncated toward zero, and EXACT past
// 2^53, where the double is already an integer the plain number cannot carry
// (int(1e23) is 99999999999999991611392). Twin of pyFToInt in
// languages/lib/python-rt.metajs and floToInt in languages/python-interpreter.abnf.
func (rt *jsrt) pyFloatToInt(f float64) interface{} {
	if f != f {
		panic(&jsThrown{value: rt.pyExcInstance("ValueError", "cannot convert float NaN to integer")})
	}
	if math.IsInf(f, 0) {
		panic(&jsThrown{value: rt.pyExcInstance("OverflowError", "cannot convert float infinity to integer")})
	}
	t := math.Trunc(f)
	if math.Abs(t) <= pyMax53 {
		return t
	}
	bi, _ := big.NewFloat(t).Int(nil)
	return rt.pyBigNarrow(bi)
}

// pyGenericAlias answers the memoized list[int]-style alias of a type object (a
// class, or a builtin such as list bound to a closure). It is keyed by the origin
// and the rendered parameter, so the same written alias is the same value twice -
// which is what lets a PEP 695 'type X = list[int]' be compared for equality.
func (rt *jsrt) pyGenericAlias(origin, param interface{}) (interface{}, bool) {
	switch origin.(type) {
	case *jsObject, *jsClosure, *hostFunc, *boundMethod:
	default:
		return nil, false
	}
	if pyTypeName(rt, origin) != "type" { // Only a type takes parameters.
		return nil, false
	}
	if _, _, isDict := dictParts(origin); isDict {
		return nil, false
	}
	if rt.pyAliases == nil {
		rt.pyAliases = map[string]*jsObject{}
	}
	name := rt.pyString(origin)
	if o, isObj := origin.(*jsObject); isObj {
		if n, has := o.props["__name"]; has {
			name = rt.toString(n)
		}
	} else if n, has := rt.pyFuncNames[origin]; has {
		name = n
	}
	key := name + "[" + rt.pyString(param) + "]"
	if a, has := rt.pyAliases[key]; has {
		return a, true
	}
	a := newJSObject()
	a.set("__name", key)
	a.set("__origin__", origin)
	a.set("__args__", &jsArray{elems: []interface{}{param}})
	rt.pyAliases[key] = a
	return a, true
}

// pySliceIndices is Python's slice index computation: it answers, in order, the
// positions a[lo:hi:step] selects of a sequence of n elements. An undefined end
// means the default for the DIRECTION of the step (the whole sequence forwards,
// the whole sequence backwards for a negative one), and a negative index counts
// from the end before the range is clamped.
func (rt *jsrt) pySliceIndices(n int, loV, hiV, stepV interface{}) []int {
	step := 1
	if !isUndefOrNull(stepV) {
		step = int(rt.toNumber(stepV))
		if step == 0 {
			rt.fail("slice step cannot be zero")
		}
	}
	lo, hi := 0, n
	if step < 0 {
		lo, hi = n-1, -n-1
	}
	if !isUndefOrNull(loV) {
		lo = rt.pyClampIndex(int(rt.toNumber(loV)), n, step)
	}
	if !isUndefOrNull(hiV) {
		hi = rt.pyClampIndex(int(rt.toNumber(hiV)), n, step)
	}
	var out []int
	for i := lo; (step > 0 && i < hi) || (step < 0 && i > hi); i += step {
		if i >= 0 && i < n {
			out = append(out, i)
		}
	}
	return out
}

// pyClampIndex turns one written slice bound into a position, the way Python does:
// negative counts from the end, and out of range clamps to the edge the step runs
// towards.
func (rt *jsrt) pyClampIndex(i, n, step int) int {
	if i < 0 {
		i += n
		if i < 0 {
			if step < 0 {
				return -1
			}
			return 0
		}
		return i
	}
	if i >= n {
		if step < 0 {
			return n - 1
		}
		return n
	}
	return i
}

// pySliceStart is where an EMPTY step-1 slice splices at: the clamped low bound.
func (rt *jsrt) pySliceStart(n int, loV interface{}) int {
	if isUndefOrNull(loV) {
		return 0
	}
	return rt.pyClampIndex(int(rt.toNumber(loV)), n, 1)
}

// pySeqElems is the element list of anything a slice assignment may take on its
// right hand side.
func (rt *jsrt) pySeqElems(v interface{}) []interface{} {
	if els, ok := pySetElems(v); ok {
		return append([]interface{}{}, els.elems...)
	}
	if arr, ok := v.(*jsArray); ok {
		return append([]interface{}{}, arr.elems...)
	}
	if g, ok := v.(*jsGenerator); ok {
		return append([]interface{}{}, g.drain(rt).elems...)
	}
	// A CURSOR gives what is LEFT of it, and the read exhausts it - the rule
	// pyElemsOf follows: `xs[1:3] = iter([5, 6])` answers [9, 5, 6, 9].
	if pyIterName(rt, v) != "" {
		return rt.pyElemsOf(v)
	}
	if str, ok := v.(string); ok {
		out := []interface{}{}
		for i, n := 0, rt.strLen(str); i < n; i++ {
			out = append(out, rt.strAt(str, i))
		}
		return out
	}
	rt.fail("cannot assign a %s to a slice", rt.typeOf(v))
	return nil
}

// ----- Python sets -----
// A set is an object carrying the marker "__set" and its elements as a list kept
// free of duplicates (by Python's ==, so 1 and 1.0 are the same member). No other
// grammar builds this shape, so every branch reading it is unreachable from the
// other languages. The list keeps insertion order, which real Python does not
// promise - but it makes a set render and iterate deterministically, which the
// byte-identical goja/frozen invariant needs.
func newPySet(elems *jsArray) *jsObject {
	o := newJSObject()
	o.set("__set", true)
	o.set("elems", elems)
	return o
}

func pySetElems(v interface{}) (*jsArray, bool) {
	o, isObj := v.(*jsObject)
	if !isObj {
		return nil, false
	}
	if _, has := o.props["__set"]; !has {
		return nil, false
	}
	els, isArr := o.props["elems"].(*jsArray)
	return els, isArr
}

func (rt *jsrt) pySetHas(els *jsArray, v interface{}) bool {
	for _, e := range els.elems {
		if rt.pyEqual(e, v) {
			return true
		}
	}
	return false
}

func (rt *jsrt) pySetAdd(els *jsArray, v interface{}) {
	// The INSERT form: no frozenset conversion, so `s.add({1})` raises where
	// `{1} in s` does not.
	rt.pyHashCheck(v, true)
	if !rt.pySetHas(els, v) {
		els.elems = append(els.elems, v)
	}
}

// ----- Hashability -----------------------------------------------------------
//
// The twin of pyHashName / pyHashCheck in languages/lib/python-rt.metajs and of
// pyHashName / pyHashCheck in languages/python-interpreter.abnf. CPython:
//
//	{[1]}      TypeError: cannot use 'list' as a set element (unhashable type: 'list')
//	{{}: 1}    TypeError: cannot use 'dict' as a dict key (unhashable type: 'dict')
//
// The wording differs between the two positions, and the OUTER name is the type
// handed over while the INNER one is the type that is unhashable - they differ
// only for a tuple holding a list, which this value model cannot build, so one
// name serves both slots.
//
// dict and set are refused; the ARRAY is not. A tuple IS an array here
// and the two directions of that mistake are not symmetric:
// treating a tuple as hashable only accepts a program CPython would reject,
// while treating a list as unhashable would abort `{(1, 2), (3, 4)}` and
// `d[i, j]`, which CPython accepts. bytearray and frozenset are not in this
// subset at all.
func pyUnhashName(v interface{}) string {
	o, isObj := v.(*jsObject)
	if !isObj {
		return ""
	}
	if tag, has := o.props["__set"]; has && tag == true {
		return "set"
	}
	if tag, has := o.props["__dict"]; has && tag == true {
		return "dict"
	}
	return ""
}

// elem: true for a set MEMBER, false for a dict KEY.
func (rt *jsrt) pyHashCheck(v interface{}, elem bool) {
	n := pyUnhashName(v)
	if n == "" {
		return
	}
	where := "a dict key"
	if elem {
		where = "a set element"
	}
	panic(&jsThrown{value: rt.pyExcInstance("TypeError",
		fmt.Sprintf("cannot use '%s' as %s (unhashable type: '%s')", n, where, n))})
}

// pyHashCheckMember is the set LOOKUP form. CPython's set.__contains__ /
// .discard / .remove convert a set argument to a frozenset instead of refusing
// it, so `{1} in {1}` is False where `{} in {1}` raises. Only that one pair is
// exempt; a set used as a DICT key still raises.
func (rt *jsrt) pyHashCheckMember(k interface{}) {
	if o, isObj := k.(*jsObject); isObj {
		if tag, has := o.props["__set"]; has && tag == true {
			return
		}
	}
	rt.pyHashCheck(k, true)
}

// pySetBin is the set algebra: union, intersection, difference and symmetric
// difference, plus the subset/superset orderings Python gives < <= > >=. ok is
// false when the operator is not one of them, so js_pybin falls through.
func (rt *jsrt) pySetBin(op string, l, r interface{}) (interface{}, bool) {
	le, lok := pySetElems(l)
	re, rok := pySetElems(r)
	if !lok || !rok {
		return nil, false
	}
	out := &jsArray{}
	switch op {
	case "|":
		out.elems = append(out.elems, le.elems...)
		for _, e := range re.elems {
			rt.pySetAdd(out, e)
		}
		return newPySet(out), true
	case "&":
		for _, e := range le.elems {
			if rt.pySetHas(re, e) {
				rt.pySetAdd(out, e)
			}
		}
		return newPySet(out), true
	case "-":
		for _, e := range le.elems {
			if !rt.pySetHas(re, e) {
				rt.pySetAdd(out, e)
			}
		}
		return newPySet(out), true
	case "^":
		for _, e := range le.elems {
			if !rt.pySetHas(re, e) {
				rt.pySetAdd(out, e)
			}
		}
		for _, e := range re.elems {
			if !rt.pySetHas(le, e) {
				rt.pySetAdd(out, e)
			}
		}
		return newPySet(out), true
	case "<=", "<", ">=", ">":
		sub, sup := le, re
		if op == ">=" || op == ">" {
			sub, sup = re, le
		}
		for _, e := range sub.elems {
			if !rt.pySetHas(sup, e) {
				return false, true
			}
		}
		if op == "<" || op == ">" {
			return len(sub.elems) < len(sup.elems), true
		}
		return true, true
	}
	return nil, false
}

// ----- Python complex numbers -----
// A complex is an ordinary object carrying the marker "__cplx" and the two
// components under the names Python reads them by, so obj.real / obj.imag need no
// special case in the attribute path. No other grammar builds this shape, so every
// branch below is unreachable from the other languages.
func newPyComplex(re, im float64) *jsObject {
	o := newJSObject()
	o.set("__cplx", true)
	o.set("real", re)
	o.set("imag", im)
	return o
}

// pyComplexParts answers the components of a complex value, and treats a plain
// number as the complex (n, 0) so mixed arithmetic needs no second code path.
func pyComplexParts(v interface{}) (re float64, im float64, ok bool) {
	o, isObj := v.(*jsObject)
	if !isObj {
		return 0, 0, false
	}
	if _, has := o.props["__cplx"]; !has {
		return 0, 0, false
	}
	re, _ = o.props["real"].(float64)
	im, _ = o.props["imag"].(float64)
	return re, im, true
}

func isPyComplex(v interface{}) bool {
	_, _, ok := pyComplexParts(v)
	return ok
}

// pyRealOf is a real operand of complex arithmetic as a double; an arbitrary
// precision int is its nearest one.
func (rt *jsrt) pyRealOf(v interface{}) float64 {
	if bi, ok := v.(*jsBigInt); ok {
		return rt.pyBigToF(bi.v)
	}
	return rt.toNumber(v)
}

// pyComplexBin is + - * / ** == != with a complex operand. ok is false for an
// operator complex numbers do not have (Python has no ordering on them either),
// which lets js_pybin fall through to its usual error. A REAL operand on either
// side takes CPython 3.14's mixed-mode rule rather than being widened, which
// keeps a -0.0 imaginary part through `z + 1.0` and an infinity through
// `z * 2.0`. Twin of cplxOp in languages/python-interpreter.abnf and
// pyAComplexBin in languages/lib/python-rt.metajs.
func (rt *jsrt) pyComplexBin(op string, l, r interface{}) (interface{}, bool) {
	lr, li, lc := pyComplexParts(l)
	if !lc {
		lr, li = rt.pyRealOf(l), 0
	}
	rr, ri, rc := pyComplexParts(r)
	if !rc {
		rr, ri = rt.pyRealOf(r), 0
	}
	switch op {
	case "+":
		if !lc {
			return newPyComplex(lr+rr, ri), true
		}
		if !rc {
			return newPyComplex(lr+rr, li), true
		}
		return newPyComplex(lr+rr, li+ri), true
	case "-":
		if !lc {
			return newPyComplex(lr-rr, -ri), true
		}
		if !rc {
			return newPyComplex(lr-rr, li), true
		}
		return newPyComplex(lr-rr, li-ri), true
	case "*":
		if !lc {
			return newPyComplex(float64(rr*lr), float64(ri*lr)), true
		}
		if !rc {
			return newPyComplex(float64(lr*rr), float64(li*rr)), true
		}
		return newPyComplex(pyCxProd(lr, li, rr, ri)), true
	case "/":
		qr, qi, ok := pyCxDiv(lc, rc, lr, li, rr, ri)
		if !ok {
			panic(&jsThrown{value: rt.pyExcInstance("ZeroDivisionError", "division by zero")})
		}
		return newPyComplex(qr, qi), true
	case "==":
		return lr == rr && li == ri, true
	case "!=":
		return lr != rr || li != ri, true
	case "**":
		pr, pi, ok := pyCxPow(lr, li, rr, ri)
		if !ok {
			panic(&jsThrown{value: rt.pyExcInstance("ZeroDivisionError", "zero to a negative or complex power")})
		}
		if math.IsInf(pr, 0) || math.IsInf(pi, 0) {
			panic(&jsThrown{value: rt.pyExcInstance("OverflowError", "complex exponentiation")})
		}
		return newPyComplex(pr, pi), true
	}
	return nil, false
}

// ---- complex arithmetic, as CPython 3.14's Objects/complexobject.c ----
//
// CPython's binary is compiled with floating-point contraction, so every
// `x + y * z` in its C source is a fused multiply-add; pyCxFma is that. Every
// other product and sum is written float64(...) so Go does not fuse it (arm64
// does). Twin of the pyCx* functions in lib/python-rt.metajs and
// python-interpreter.abnf, whose software FMA falls back to two roundings for
// an operand beyond 2^995 or a product below 2^-960; this one does the same so
// the engines agree there too.
func pyCxFma(a, b, c float64) float64 {
	p := float64(a * b)
	if p == 0 || math.IsNaN(p) || math.IsInf(p, 0) || math.IsNaN(c) || math.IsInf(c, 0) ||
		math.Abs(a) > 0x1p995 || math.Abs(b) > 0x1p995 || math.Abs(p) < 0x1p-960 {
		return p + c
	}
	return math.FMA(a, b, c)
}

// pyCxSign is copysign(m, s); a NaN counts as positive, as in the MetaJS twins
// that cannot read a NaN's sign bit.
func pyCxSign(m, s float64) float64 {
	if s < 0 || (s == 0 && math.Signbit(s)) {
		return -m
	}
	return m
}

func pyCxBox(x float64) float64 {
	if math.IsInf(x, 0) {
		return pyCxSign(1, x)
	}
	return pyCxSign(0, x)
}

func pyCxDeNaN(x float64) float64 {
	if math.IsNaN(x) {
		return pyCxSign(0, x)
	}
	return x
}

// pyCxProd is _Py_c_prod, with C11 Annex G's recovery of an infinity that the
// four products turned into nan+nanj.
func pyCxProd(a, b, c, d float64) (float64, float64) {
	ac, bd, ad, bc := float64(a*c), float64(b*d), float64(a*d), float64(b*c)
	re, im := ac-bd, ad+bc
	if math.IsNaN(re) && math.IsNaN(im) {
		recalc := false
		if math.IsInf(a, 0) || math.IsInf(b, 0) {
			a, b, c, d = pyCxBox(a), pyCxBox(b), pyCxDeNaN(c), pyCxDeNaN(d)
			recalc = true
		}
		if math.IsInf(c, 0) || math.IsInf(d, 0) {
			c, d, a, b = pyCxBox(c), pyCxBox(d), pyCxDeNaN(a), pyCxDeNaN(b)
			recalc = true
		}
		if !recalc && (math.IsInf(ac, 0) || math.IsInf(bd, 0) || math.IsInf(ad, 0) || math.IsInf(bc, 0)) {
			a, b, c, d = pyCxDeNaN(a), pyCxDeNaN(b), pyCxDeNaN(c), pyCxDeNaN(d)
			recalc = true
		}
		if recalc {
			inf := math.Inf(1)
			return inf * (float64(a*c) - float64(b*d)), inf * (float64(a*d) + float64(b*c))
		}
	}
	return re, im
}

// pyCxQuot is _Py_c_quot: Smith's algorithm, then Annex G's recovery. ok is
// false for a zero divisor.
func pyCxQuot(a, b, c, d float64) (float64, float64, bool) {
	ac, ad := math.Abs(c), math.Abs(d)
	var re, im float64
	switch {
	case ac >= ad:
		if ac == 0 {
			return 0, 0, false
		}
		ratio := d / c
		denom := pyCxFma(d, ratio, c)
		re, im = pyCxFma(b, ratio, a)/denom, pyCxFma(-a, ratio, b)/denom
	case ad >= ac:
		ratio := c / d
		denom := pyCxFma(c, ratio, d)
		re, im = pyCxFma(a, ratio, b)/denom, pyCxFma(b, ratio, -a)/denom
	default:
		re, im = math.NaN(), math.NaN()
	}
	if math.IsNaN(re) && math.IsNaN(im) {
		finite := func(x float64) bool { return !math.IsNaN(x) && !math.IsInf(x, 0) }
		if (math.IsInf(a, 0) || math.IsInf(b, 0)) && finite(c) && finite(d) {
			x, y := pyCxBox(a), pyCxBox(b)
			inf := math.Inf(1)
			return inf * (float64(x*c) + float64(y*d)), inf * (float64(y*c) - float64(x*d)), true
		}
		if (math.IsInf(ac, 0) || math.IsInf(ad, 0)) && finite(a) && finite(b) {
			x, y := pyCxBox(c), pyCxBox(d)
			return 0 * (float64(a*x) + float64(b*y)), 0 * (float64(b*x) - float64(a*y)), true
		}
	}
	return re, im, true
}

// pyCxRQuot is _Py_rc_quot: a REAL dividend, so no imaginary part takes part.
func pyCxRQuot(a, c, d float64) (float64, float64, bool) {
	ac, ad := math.Abs(c), math.Abs(d)
	var re, im float64
	switch {
	case ac >= ad:
		if ac == 0 {
			return 0, 0, false
		}
		ratio := d / c
		denom := pyCxFma(d, ratio, c)
		re, im = a/denom, float64(-a*ratio)/denom
	case ad >= ac:
		ratio := c / d
		denom := pyCxFma(c, ratio, d)
		re, im = float64(a*ratio)/denom, -a/denom
	default:
		re, im = math.NaN(), math.NaN()
	}
	if math.IsNaN(re) && math.IsNaN(im) && !math.IsNaN(a) && !math.IsInf(a, 0) &&
		(math.IsInf(ac, 0) || math.IsInf(ad, 0)) {
		x, y := pyCxBox(c), pyCxBox(d)
		return 0 * float64(a*x), 0 * float64(-a*y), true
	}
	return re, im, true
}

// pyCxDiv is a / b with either side possibly real (lc / rc say which are
// complex); ok is false for a zero divisor.
func pyCxDiv(lc, rc bool, a, b, c, d float64) (float64, float64, bool) {
	if !rc {
		if c == 0 {
			return 0, 0, false
		}
		return a / c, b / c, true
	}
	if !lc {
		return pyCxRQuot(a, c, d)
	}
	return pyCxQuot(a, b, c, d)
}

// pyCxHypot is hypot(x, y) as the platform libm computes it:
// sqrt(fma(small, small, big*big)), the operands scaled by 2^600 when the
// squares could leave the normal range.
func pyCxHypot(x, y float64) float64 {
	a, b := math.Abs(x), math.Abs(y)
	if math.IsInf(a, 0) || math.IsInf(b, 0) {
		return math.Inf(1)
	}
	if math.IsNaN(a) || math.IsNaN(b) {
		return math.NaN()
	}
	if a < b {
		a, b = b, a
	}
	if a == 0 {
		return 0
	}
	if a > 0x1p400 {
		a, b = a/0x1p600, b/0x1p600
		return math.Sqrt(pyCxFma(b, b, float64(a*a))) * 0x1p600
	}
	if a < 0x1p-400 {
		a, b = a*0x1p600, b*0x1p600
		return math.Sqrt(pyCxFma(b, b, float64(a*a))) / 0x1p600
	}
	return math.Sqrt(pyCxFma(b, b, float64(a*a)))
}

// pyCxPowU is c_powu: square and multiply, the order of the products CPython's.
func pyCxPowU(a, b float64, n int) (float64, float64) {
	rr, ri, pr, pi := 1.0, 0.0, a, b
	for ; n > 0; n >>= 1 {
		if n&1 == 1 {
			rr, ri = pyCxProd(rr, ri, pr, pi)
		}
		pr, pi = pyCxProd(pr, pi, pr, pi)
	}
	return rr, ri
}

// pyCxPow is complex ** complex: c_powi for an integral real exponent up to 100
// in size, else _Py_c_pow. ok is false for a zero base to a negative or
// complex power.
func pyCxPow(a, b, c, d float64) (float64, float64, bool) {
	if d == 0 && c == math.Floor(c) && math.Abs(c) <= 100 {
		if c > 0 {
			re, im := pyCxPowU(a, b, int(c))
			return re, im, true
		}
		ur, ui := pyCxPowU(a, b, int(-c))
		return pyCxQuot(1, 0, ur, ui)
	}
	if c == 0 && d == 0 {
		return 1, 0, true
	}
	if a == 0 && b == 0 {
		if d != 0 || c < 0 {
			return 0, 0, false
		}
		return 0, 0, true
	}
	vabs := pyCxHypot(a, b)
	ln := math.Pow(vabs, c)
	at := math.Atan2(b, a)
	phase := float64(at * c)
	if d != 0 {
		ln /= math.Exp(float64(at * d))
		phase = pyCxFma(d, math.Log(vabs), phase)
	}
	return float64(ln * math.Cos(phase)), float64(ln * math.Sin(phase)), true
}

// pyCxAbs is abs(complex): the libm hypot, and CPython's OverflowError when
// finite parts give an infinite modulus.
func (rt *jsrt) pyCxAbs(re, im float64) float64 {
	h := pyCxHypot(re, im)
	if math.IsInf(h, 0) && !math.IsInf(re, 0) && !math.IsInf(im, 0) {
		panic(&jsThrown{value: rt.pyExcInstance("OverflowError", "absolute value too large")})
	}
	return h
}

// pyCplxPart is one part of repr(complex): repr(float) without its forced ".0".
func pyCplxPart(f float64) string {
	return strings.TrimSuffix(pyFloStr(f), ".0")
}

// pyCplxRepr is repr(complex), which is also str(): the real part is left out
// when it is +0.0, otherwise the pair is parenthesized. Twin of cplxStr in
// languages/python-interpreter.abnf and pyCplxRepr in
// languages/lib/python-rt.metajs.
func pyCplxRepr(re, im float64) string {
	ims := pyCplxPart(im)
	if re == 0 && !math.Signbit(re) {
		return ims + "j"
	}
	if ims[0] != '-' {
		ims = "+" + ims
	}
	return "(" + pyCplxPart(re) + ims + "j)"
}

// pyComplexNew is complex(real=0, imag=0): each argument a real number or a
// complex, the result real.real - imag.imag + (real.imag + imag.real)j as CPython
// builds it; a lone string is parsed. Twin of pyComplexNew in
// languages/python-interpreter.abnf and js_pycomplexnew in
// languages/lib/python-rt.metajs.
func (rt *jsrt) pyComplexNew(args []interface{}) interface{} {
	var r, i interface{} = 0.0, 0.0
	if len(args) > 0 {
		r = args[0]
	}
	if len(args) > 1 {
		i = args[1]
	}
	if s, ok := r.(string); ok && len(args) < 2 {
		return rt.pyCplxFromStr(s)
	}
	part := func(v interface{}, what string) (float64, float64, bool) {
		if re, im, ok := pyComplexParts(v); ok {
			return re, im, true
		}
		switch t := v.(type) {
		case float64, bool, *jsPyFlo, *jsBigInt:
			return rt.pyRealOf(t), 0, false
		}
		panic(&jsThrown{value: rt.pyExcInstance("TypeError", what+pyTypeName(rt, v))})
	}
	what := "complex() argument must be a string or a number, not "
	if len(args) > 1 {
		what = "complex() argument 'real' must be a real number, not "
	}
	re, im, rc := part(r, what)
	ire, iim, ic := part(i, "complex() argument 'imag' must be a real number, not ")
	if len(args) > 1 {
		im = ire
		if rc {
			_, rim, _ := pyComplexParts(r)
			im += rim
		}
		if ic {
			re -= iim
		}
	}
	return newPyComplex(re, im)
}

// pyCplxFromStr is complex("1+2j"), CPython's complex_from_string: surrounding
// spaces and one pair of parentheses, then a real, an imaginary (2j), or a real
// and a signed imaginary (1-2j); an imaginary with no digits (j, -j, 1+j) is 1.
func (rt *jsrt) pyCplxFromStr(s0 string) interface{} {
	s := strings.TrimSpace(s0)
	if len(s) > 1 && s[0] == '(' && s[len(s)-1] == ')' {
		s = strings.TrimSpace(s[1 : len(s)-1])
	}
	num := func(t string) float64 { return pyStrToF(strings.ReplaceAll(t, "_", "")) }
	isJ := func(k int) bool { return k < len(s) && (s[k] == 'j' || s[k] == 'J') }
	e1 := pyFloPrefix(s, 0)
	a := 0.0
	if e1 > 0 {
		a = num(s[:e1])
		if e1 == len(s) {
			return newPyComplex(a, 0)
		}
		if isJ(e1) && e1+1 == len(s) {
			return newPyComplex(0, a)
		}
	}
	var sg byte
	if e1 < len(s) {
		sg = s[e1]
	}
	if sg == '+' || sg == '-' || e1 == 0 {
		e2 := pyFloPrefix(s, e1)
		b := 0.0
		if e2 > e1 {
			b = num(s[e1:e2])
		} else {
			b = 1
			if sg == '-' {
				b = -1
			}
			if sg == '+' || sg == '-' {
				e2 = e1 + 1
			}
		}
		if isJ(e2) && e2+1 == len(s) {
			return newPyComplex(a, b)
		}
	}
	panic(&jsThrown{value: rt.pyExcInstance("ValueError", "complex() arg is a malformed string")})
}

// pyFloPrefix answers the end of the float literal at i (a sign, digits with an
// optional point, an exponent; or inf / infinity / nan), or i when there is none.
func pyFloPrefix(s string, i int) int {
	at := func(k int) byte {
		if k < len(s) {
			return s[k]
		}
		return 0
	}
	digit := func(k int) bool { c := at(k); return (c >= '0' && c <= '9') || c == '_' }
	j := i
	if at(j) == '+' || at(j) == '-' {
		j++
	}
	end := j + 8
	if end > len(s) {
		end = len(s)
	}
	low := strings.ToLower(s[j:end])
	if low == "infinity" {
		return j + 8
	}
	if strings.HasPrefix(low, "inf") || strings.HasPrefix(low, "nan") {
		return j + 3
	}
	d := 0
	for ; digit(j); j++ {
		d++
	}
	if at(j) == '.' {
		for j++; digit(j); j++ {
			d++
		}
	}
	if d == 0 {
		return i
	}
	if at(j) == 'e' || at(j) == 'E' {
		k := j + 1
		if at(k) == '+' || at(k) == '-' {
			k++
		}
		ed := 0
		for ; digit(k); k++ {
			ed++
		}
		if ed > 0 {
			j = k
		}
	}
	return j
}

// ----------------------------------------------------------------------------
// Python classes
//
// The shape is deliberately the one the class based languages already use (see
// memberCall): an instance is {__class: <class object>}, a class object carries
// its methods and class attributes as plain properties. What Python adds is the
// MRO: "__mro" is the C3 linearization computed once, when the class object is
// built, so every attribute read and every method call stays a flat walk even
// with multiple inheritance. "__super" is kept in step with the first base so
// the single-inheritance walkers elsewhere in this file still see a chain.

// pyClassObj reports whether v is a Python class object (it carries an MRO).
func pyClassObj(v interface{}) (*jsObject, bool) {
	o, ok := v.(*jsObject)
	if !ok {
		return nil, false
	}
	if _, has := o.props["__mro"]; !has {
		return nil, false
	}
	return o, true
}

// pyInstance reports whether v is an instance of a Python class, and of which.
func pyInstance(v interface{}) (*jsObject, *jsObject, bool) {
	o, ok := v.(*jsObject)
	if !ok {
		return nil, nil, false
	}
	cls, ok := o.props["__class"].(*jsObject)
	if !ok {
		return nil, nil, false
	}
	return o, cls, true
}

// pyMRO is the resolution order of a class, itself first. A class object that
// predates the MRO (js_pyexc's builtin exception descriptors) answers with its
// __super chain, so the two shapes stay interchangeable.
func pyMRO(c *jsObject) []*jsObject {
	if m, ok := c.props["__mro"].(*jsArray); ok {
		out := make([]*jsObject, 0, len(m.elems))
		for _, e := range m.elems {
			if o, ok := e.(*jsObject); ok {
				out = append(out, o)
			}
		}
		return out
	}
	var out []*jsObject
	for k := c; k != nil; {
		out = append(out, k)
		sup, _ := k.props["__super"].(*jsObject)
		k = sup
	}
	return out
}

// pyLinearize is the C3 merge of the bases' MROs with the base list itself -
// the order Python computes, minus the class itself (the caller prepends it).
// A hierarchy C3 cannot resolve consistently falls back to first-seen order
// rather than aborting the program.
func pyLinearize(bases []*jsObject) []*jsObject {
	if len(bases) == 0 {
		return nil
	}
	var seqs [][]*jsObject
	for _, b := range bases {
		seqs = append(seqs, pyMRO(b))
	}
	seqs = append(seqs, append([]*jsObject{}, bases...))
	var out []*jsObject
	for {
		empty := true
		for _, s := range seqs {
			if len(s) > 0 {
				empty = false
			}
		}
		if empty {
			return out
		}
		var pick *jsObject
		for _, s := range seqs {
			if len(s) == 0 {
				continue
			}
			head := s[0]
			inTail := false
			for _, t := range seqs {
				for i := 1; i < len(t); i++ {
					if t[i] == head {
						inTail = true
					}
				}
			}
			if !inTail {
				pick = head
				break
			}
		}
		if pick == nil { // Inconsistent hierarchy: take the first head there is.
			for _, s := range seqs {
				if len(s) > 0 {
					pick = s[0]
					break
				}
			}
		}
		out = append(out, pick)
		for i, s := range seqs {
			if len(s) > 0 && s[0] == pick {
				seqs[i] = s[1:]
			}
		}
	}
}

// pyLookup finds a name along the MRO of a class.
// pyIsKeyErrCls reports whether a class IS KeyError or derives from it. KeyError
// is the one builtin exception whose __str__ differs from BaseException's, and a
// subclass INHERITS that, so the test is the class chain rather than the name.
// isKeyErrCls in languages/python-interpreter.abnf and pyIsKeyErrCls in
// languages/lib/python-rt.metajs are this body.
func pyIsKeyErrCls(c *jsObject) bool {
	for _, k := range pyMRO(c) {
		if n, ok := k.props["__name"].(string); ok && n == "KeyError" {
			return true
		}
	}
	return false
}

func pyLookup(c *jsObject, name string) (interface{}, bool) {
	if name == "" {
		return nil, false
	}
	for _, k := range pyMRO(c) {
		if v, ok := pyClassMember(k, name); ok {
			return v, true
		}
	}
	return nil, false
}

// ----------------------------------------------------------------- super ---
//
// A super() proxy carries the class the call was written in ("__scls") and the
// instance ("__sobj"); a member lookup through it scans the INSTANCE's MRO
// starting one past that class, which is what makes a diamond resolve as CPython
// resolves it. The one-argument (unbound) form is declined - it is a descriptor
// and this value model has no protocol for one.
//
// The twins are pySuper* in languages/lib/python-rt.metajs and pySuper* in
// languages/python-interpreter.abnf.
func pySuperNew(cls, self interface{}) *jsObject {
	s := newJSObject()
	s.set("__pysup", true)
	if cls != nil && !isUndefOrNull(cls) {
		s.set("__scls", cls)
	}
	if self != nil && self != jsUndef {
		s.set("__sobj", self)
	}
	return s
}

func pySuperObj(v interface{}) (*jsObject, bool) {
	o, ok := v.(*jsObject)
	if !ok {
		return nil, false
	}
	if b, has := o.props["__pysup"].(bool); !has || !b {
		return nil, false
	}
	return o, true
}

// CPython's own two messages for a zero-argument super() with nothing to bind.
func (rt *jsrt) pySuperCheck(s *jsObject) {
	if _, has := s.props["__scls"]; !has {
		panic(&jsThrown{value: rt.pyExcInstance("RuntimeError", "super(): __class__ cell not found")})
	}
	if _, has := s.props["__sobj"]; !has {
		panic(&jsThrown{value: rt.pyExcInstance("RuntimeError", "super(): no arguments")})
	}
}

func (rt *jsrt) pySuperCall(s *jsObject, args []interface{}) interface{} {
	switch len(args) {
	case 0:
		rt.pySuperCheck(s)
		return s
	case 2:
		return pySuperNew(args[0], args[1])
	}
	panic(&jsThrown{value: rt.pyExcInstance("TypeError",
		"super(): one-argument (unbound) form is not supported")})
}

func (rt *jsrt) pySuperFind(s *jsObject, name string) (interface{}, bool) {
	var mro []*jsObject
	obj := s.props["__sobj"]
	if c, isCls := pyClassObj(obj); isCls {
		mro = pyMRO(c)
	} else if _, cls, isInst := pyInstance(obj); isInst {
		mro = pyMRO(cls)
	}
	start, _ := s.props["__scls"].(*jsObject)
	at := -1
	for i, k := range mro {
		if k == start && at < 0 {
			at = i
		}
	}
	if at < 0 {
		panic(&jsThrown{value: rt.pyExcInstance("TypeError",
			"super(type, obj): obj must be an instance or subtype of type")})
	}
	for _, k := range mro[at+1:] {
		if v, ok := pyClassMember(k, name); ok {
			return v, true
		}
	}
	return nil, false
}

func (rt *jsrt) pySuperAttr(s *jsObject, name string) interface{} {
	rt.pySuperCheck(s)
	m, found := rt.pySuperFind(s, name)
	if !found {
		panic(&jsThrown{value: rt.pyExcInstance("AttributeError",
			"'super' object has no attribute '"+name+"'")})
	}
	// A staticmethod / classmethod found through super(). The owner a
	// classmethod binds to is the ORIGINAL cls - `super().who()` inside C.who()
	// must still pass C - and __sobj is exactly that when the frame's first
	// argument was a class.
	if pyDescKind(m) != "" {
		return rt.pyDescGet(m, rt.pySuperOwner(s), name)
	}
	if isCallable(m) {
		return rt.pyBindMethod(s.props["__sobj"], m, name)
	}
	return m
}

// pySuperOwner is the class a classmethod reached through super() binds to.
func (rt *jsrt) pySuperOwner(s *jsObject) interface{} {
	o := s.props["__sobj"]
	if _, isCls := pyClassObj(o); isCls {
		return o
	}
	return rt.pyBuiltinClass(pyTypeName(rt, o))
}

func (rt *jsrt) pySuperMCall(s *jsObject, name string, args []interface{}, kw interface{}) interface{} {
	rt.pySuperCheck(s)
	m, found := rt.pySuperFind(s, name)
	if !found {
		panic(&jsThrown{value: rt.pyExcInstance("AttributeError",
			"'super' object has no attribute '"+name+"'")})
	}
	if pyDescKind(m) != "" {
		d := rt.pyDescGet(m, rt.pySuperOwner(s), name)
		return rt.call(d, jsUndef, rt.pyBindCall(d, args, kw))
	}
	if !isCallable(m) {
		return rt.callPyValue(m, args, kw)
	}
	self := append([]interface{}{s.props["__sobj"]}, args...)
	return rt.call(m, jsUndef, rt.pyBindCall(m, self, kw))
}

// pyBindMethod is the bound method a METHOD read off an instance answers, so
// `f = a.m; f()` passes the instance as self. The interpreter half's twin is
// pyattr's bindMethod; layer 2: pyDBindMethod.
func (rt *jsrt) pyBindMethod(self, m interface{}, name string) interface{} {
	h := jsHostFunc("pymethod", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		full := append([]interface{}{self}, args...)
		return rt.call(m, jsUndef, rt.pyBindCall(m, full, jsUndef))
	})
	h.pybm = &pyBoundInfo{name: name, self: self, fn: m, typ: rt.pyOwnerName(self)}
	return h
}

// pyOwnerName is the type name a bound method's __qualname__ and repr are
// prefixed with: an INSTANCE names its class, everything else names its builtin
// type. pyTypeName answers "object" for an instance (it has no class arm and
// several callers depend on that), so the two cannot be merged.
func (rt *jsrt) pyOwnerName(self interface{}) string {
	if _, cls, ok := pyInstance(self); ok {
		return rt.toString(cls.props["__name"])
	}
	return pyTypeName(rt, self)
}

// pyIsDescriptor reports whether v is an instance whose class defines the given
// descriptor hook (__get__ / __set__).
func pyIsDescriptor(v interface{}, hook string) (interface{}, bool) {
	_, cls, ok := pyInstance(v)
	if !ok {
		return nil, false
	}
	m, found := pyLookup(cls, hook)
	if !found || !isCallable(m) {
		return nil, false
	}
	return m, true
}

// pyMroAttr answers C.__mro__ and C.__bases__. The class chains here leave out
// `object`, which owns nothing a lookup could find, while CPython ends every
// resolution order with it and gives a root class the base (object,); a builtin
// type is its own chain, bool's going through int. Twin of pyMroAttr in
// languages/lib/python-rt.metajs and languages/python-interpreter.abnf.
func (rt *jsrt) pyMroAttr(cls *jsObject, name string) *jsArray {
	obj := rt.pyBuiltinClass("object")
	if cls == obj {
		if name == "__bases__" {
			return &jsArray{}
		}
		return &jsArray{elems: []interface{}{obj}}
	}
	var out []interface{}
	if name == "__bases__" {
		if bl, ok := cls.props["__basesl"].(*jsArray); ok {
			out = append(out, bl.elems...)
		}
	} else if mro, ok := cls.props["__mro"].(*jsArray); ok {
		out = append(out, mro.elems...)
	} else {
		out = append(out, cls)
	}
	if pyBuiltinClsName(cls) == "bool" {
		out = append(out, rt.pyBuiltinClass("int"))
	}
	if len(out) == 0 || (name == "__mro__" && out[len(out)-1] != obj) {
		out = append(out, obj)
	}
	return &jsArray{elems: out}
}

// pyGetAttr reads obj.name with Python's lookup order.
func (rt *jsrt) pyGetAttr(obj interface{}, name string) interface{} {
	// None has no attributes and CPython says so with a catchable AttributeError
	// rather than a getMember abort. __class__ is the one name None does answer.
	if isUndefOrNull(obj) && name != "__class__" {
		panic(&jsThrown{value: rt.pyExcInstance("AttributeError",
			"'NoneType' object has no attribute '"+name+"'")})
	}
	if sup, isSup := pySuperObj(obj); isSup {
		return rt.pySuperAttr(sup, name)
	}
	if cls, ok := pyClassObj(obj); ok {
		switch name {
		// __qualname__ of a class is its __name__ for every class this subset can
		// write: CPython qualifies only a NESTED class, and a nested `class` here
		// is bound by the same name as a top-level one.
		case "__name__", "__qualname__":
			if n, ok := cls.props["__name"].(string); ok {
				return pyShortName(n)
			}
			return cls.props["__name"]
		case "__mro__", "__bases__":
			return rt.pyMroAttr(cls, name)
		}
		if v, found := pyLookup(cls, name); found {
			// A staticmethod / classmethod read off the CLASS: the static one
			// loses nothing, the class one binds to this very class.
			// See abnf/jsrtpydesc.go.
			if pyDescKind(v) != "" {
				return rt.pyDescGet(v, cls, name)
			}
			return v
		}
		// C.mro(), type's own method: the resolution order as a fresh list.
		if name == "mro" {
			return jsHostFunc("mro", func(rt *jsrt, this uint64, args []interface{}) interface{} {
				return rt.pyMroAttr(cls, "__mro__")
			})
		}
		// A CATCHABLE AttributeError, as CPython raises, so `try: getattr(C,
		// "nope") except AttributeError:` works.
		panic(&jsThrown{value: rt.pyExcInstance("AttributeError",
			"type object '"+rt.toString(cls.props["__name"])+"' has no attribute '"+name+"'")})
	}
	if inst, cls, ok := pyInstance(obj); ok {
		switch name {
		case "__class__":
			return cls
		case "__dict__":
			return rt.pyInstanceDict(inst)
		}
		if v, found := inst.props[name]; found {
			return v
		}
		if v, found := pyLookup(cls, name); found {
			// staticmethod / classmethod, before every other arm: a static one
			// must NOT pick up the instance as self, and a class one binds to
			// the receiver's OWN class, so D().c() passes D.
			if pyDescKind(v) != "" {
				return rt.pyDescGet(v, cls, name)
			}
			if get, isDesc := pyIsDescriptor(v, "__get__"); isDesc {
				return rt.call(get, jsUndef, []interface{}{v, obj, cls})
			}
			// A METHOD read off an instance comes back BOUND - see pyBindMethod.
			if isCallable(v) {
				return rt.pyBindMethod(obj, v, name)
			}
			return v
		}
		panic(&jsThrown{value: rt.pyExcInstance("AttributeError",
			"'"+rt.toString(cls.props["__name"])+"' object has no attribute '"+name+"'")})
	}
	// A MODULE object. `builtins.len` resolves through the module's own slot,
	// which the emitter filled with the very binding the global name resolves
	// to, so `builtins.len is len` holds rather than answering a copy; a math
	// function becomes a first-class value the same way `re`'s constants are
	// readable. Ahead of the builtin arms because a module's own data slots
	// (math.pi) must win. See abnf/jsrtpydesc.go.
	if mod, isMod := pyIsModule(obj); isMod {
		if v, has := rt.pyModuleAttr(mod, name); has {
			return v
		}
		panic(&jsThrown{value: rt.pyExcInstance("AttributeError",
			"module '"+rt.toString(mod.props["__pymod"])+"' has no attribute '"+name+"'")})
	}
	// A method the builtin TYPE OBJECT itself carries - dict.fromkeys.
	if m, ok := rt.pyBuiltinClsMethod(obj, name); ok {
		return m
	}
	if name == "__mro__" || name == "__bases__" {
		if _, isType := rt.pyClassName(obj); isType {
			return rt.pyMroAttr(obj.(*jsObject), name)
		}
	}
	// list.count / str.upper / dict.get read off the BUILTIN type object: an
	// unbound method descriptor. It sits HERE rather than in
	// the pyClassObj branch above because a builtin type object carries only a
	// __name and no __mro, so pyClassObj declines it - which is also why
	// dict.fromkeys is answered on this same line.
	if tn, isType := rt.pyClassName(obj); isType {
		if samp := pyDescTypeSample(tn); samp != nil && rt.pyBuiltinAttr(samp, name) {
			return rt.pyUnboundDesc(obj.(*jsObject), name)
		}
		// A dunder read off the type - str.__repr__(self) in a subclass.
		if f, ok := pyBuiltinDelegate(tn, name); ok {
			return f
		}
	}
	// A BOUND METHOD knows its own name, receiver and underlying function.
	// Asked before the two name tables because a bound method
	// is in neither of them. __qualname__ is answered HERE and nowhere else on
	// purpose: 'list.count' and 'C.m' are exactly CPython's answers and both
	// fall out of the record, while a plain function's qualname would need the
	// DEFINING scope recorded at every def site in four places - see the note on
	// pyBoundInfo and the declined rows in tests/python-test-full.py.
	if bi := pyBoundOf(obj); bi != nil {
		switch name {
		case "__name__":
			return bi.name
		case "__qualname__":
			// A module-level builtin's qualname is its bare name; the other two
			// kinds are TYPE.NAME, which bi.typ already holds.
			if bi.kind == "m" {
				return bi.name
			}
			return bi.typ + "." + bi.name
		case "__objclass__":
			// CPython gives a method DESCRIPTOR no __self__ at all and offers
			// __objclass__ instead.
			if bi.kind == "d" {
				return bi.self
			}
		case "__self__":
			if bi.kind == "d" {
				panic(&jsThrown{value: rt.pyExcInstance("AttributeError",
					"'method_descriptor' object has no attribute '__self__'")})
			}
			return bi.self
		case "__func__":
			if bi.fn == nil {
				// CPython: a built-in method has no __func__.
				panic(&jsThrown{value: rt.pyExcInstance("AttributeError",
					"'builtin_function_or_method' object has no attribute '__func__'")})
			}
			return bi.fn
		}
	}
	// DECLINED, and it is worth saying where: __doc__ and a function's
	// __module__ stay unanswered (an AttributeError, like any other absent
	// attribute). __doc__ needs a def's leading string statement recorded in
	// three engines AND CPython's own text for every builtin, which this project
	// will never carry; a function's __module__ needs the defining module
	// recorded at every def site.
	// __qualname__ rides with __name__ everywhere below. It is EXACT for a
	// module-level def, a lambda, a builtin function and a builtin type, and it
	// is wrong for exactly one shape: a function read off a class body, where
	// CPython says 'C.m' and this says 'm'. Closing that one needs the DEFINING
	// scope recorded at every def site in the emitter, the interpreter and layer
	// 2; the bound-method arm above already answers 'C.m' because a bound method
	// knows its receiver.
	// len.__self__ is the builtins MODULE in CPython, not a receiver - the
	// builtin functions are that module's members.
	if name == "__self__" {
		if _, isBuiltin := rt.pyBuiltinRender[obj]; isBuiltin {
			return rt.pyBuiltinsModule()
		}
	}
	if name == "__name__" || name == "__qualname__" {
		// A synthetic class object for a BUILTIN type (what type(3) hands out)
		// carries only its name, and a closure carries none of its own - the name
		// its def bound it under was recorded by js_pyfnname.
		if o, ok := obj.(*jsObject); ok {
			if n, has := o.props["__name"]; has {
				if ns, isStr := n.(string); isStr {
					return pyShortName(ns)
				}
				return n
			}
		}
		// The BUILTIN table is asked too, and FIRST, exactly as pyFuncRender asks
		// it - len.__name__ is 'len'. The two tables are keyed by the closure
		// VALUE, so a user `def len(...)` still answers its own name.
		if c, ok := rt.pyBuiltinRender[obj]; ok {
			return strings.TrimPrefix(c, "*")
		}
		if fn, ok := rt.pyFuncNames[obj]; ok {
			return fn
		}
	}
	// A function's own attributes, and the dict holding them.
	if isCallable(obj) {
		if d, i := rt.pyFuncAttrIdx(obj, name); i >= 0 {
			_, vals, _ := dictParts(d)
			return vals.elems[i]
		}
		if name == "__dict__" && pyTypeName(rt, obj) == "function" {
			return rt.pyFuncAttrs(obj, true)
		}
	}
	// A BOUND built-in method, so getattr(xs, "count")(1) is
	// the 1 CPython answers rather than whatever getMember made of the name.
	if rt.pyBuiltinAttr(obj, name) {
		return rt.pyBoundBuiltin(obj, name)
	}
	// x.__class__ for a value that is not a user instance - the same synthetic
	// class object type(x) hands out, so `[].__class__ is list` holds.
	if name == "__class__" {
		return rt.pyBuiltinClass(pyTypeName(rt, obj))
	}
	// CPython names a TYPE OBJECT differently from a value: "type object 'list'
	// has no attribute 'nope'", not "'type' object has no attribute 'nope'".
	if tn2, isType2 := rt.pyClassName(obj); isType2 {
		panic(&jsThrown{value: rt.pyExcInstance("AttributeError",
			"type object '"+tn2+"' has no attribute '"+name+"'")})
	}
	// An attribute a builtin value does NOT have raises AttributeError, as
	// CPython does (`[1,2].nope`, getattr([1,2], "nope")). An OWN property is
	// still answered first, because a value that really holds None must not be
	// mistaken for a missing one (a compiled regex and a match object are read
	// this way).
	// A complex's parts are floats.
	if re, im, ok := pyComplexParts(obj); ok && (name == "real" || name == "imag") {
		if name == "real" {
			return &jsPyFlo{f: re}
		}
		return &jsPyFlo{f: im}
	}
	if o, ok := obj.(*jsObject); ok {
		if v, has := o.props[name]; has {
			return v
		}
	}
	v := rt.getMember(obj, name)
	if isUndefOrNull(v) {
		panic(&jsThrown{value: rt.pyExcInstance("AttributeError",
			"'"+pyTypeName(rt, obj)+"' object has no attribute '"+name+"'")})
	}
	return v
}

// pyInstanceDict renders an instance's own attributes as a Python dict.
func (rt *jsrt) pyInstanceDict(inst *jsObject) interface{} {
	keys, vals := &jsArray{}, &jsArray{}
	for _, k := range inst.keys {
		if len(k) >= 2 && k[0] == '_' && k[1] == '_' {
			continue
		}
		dictAppend(keys, vals, k, inst.props[k])
	}
	d := newJSObject()
	d.set("__dict", true)
	d.set("keys", keys)
	d.set("vals", vals)
	return d
}

// pySetAttr writes obj.name, honouring a __set__ descriptor and __slots__.
func (rt *jsrt) pySetAttr(obj interface{}, name string, v interface{}) {
	if inst, cls, ok := pyInstance(obj); ok {
		if cur, found := pyLookup(cls, name); found {
			if set, isDesc := pyIsDescriptor(cur, "__set__"); isDesc {
				rt.call(set, jsUndef, []interface{}{cur, obj, v})
				return
			}
		}
		if !rt.pySlotsAllow(cls, name) {
			panic(&jsThrown{value: rt.pyAttrError(cls, name)})
		}
		inst.set(name, v)
		return
	}
	if cls, ok := pyClassObj(obj); ok {
		cls.set(name, v)
		return
	}
	if isCallable(obj) {
		rt.pyFuncSetAttr(obj, name, v)
		return
	}
	rt.pyNoDictAttr(obj, name)
	rt.setMember(obj, name, v)
}

// pyFuncAttrs answers a user function's attribute dict, made on demand, or nil.
// Twin of pyFnAttrs in languages/python-interpreter.abnf and
// languages/lib/python-rt.metajs.
func (rt *jsrt) pyFuncAttrs(f interface{}, make bool) *jsObject {
	if d, ok := rt.pyFuncAttrTab[f]; ok || !make {
		return d
	}
	d := newJSObject()
	d.set("__dict", true)
	d.set("keys", &jsArray{})
	d.set("vals", &jsArray{})
	if rt.pyFuncAttrTab == nil {
		rt.pyFuncAttrTab = map[interface{}]*jsObject{}
	}
	rt.pyFuncAttrTab[f] = d
	return d
}

// pyFuncOwner: a bound method reads its function's attributes (C().m.k is C.m.k).
func pyFuncOwner(f interface{}) interface{} {
	if bi := pyBoundOf(f); bi != nil && bi.fn != nil {
		return bi.fn
	}
	return f
}

// pyFuncAttrIdx answers the slot of a function attribute, or -1.
func (rt *jsrt) pyFuncAttrIdx(f interface{}, name string) (*jsObject, int) {
	d := rt.pyFuncAttrs(pyFuncOwner(f), false)
	if d == nil {
		return nil, -1
	}
	keys, _, _ := dictParts(d)
	return d, rt.dictFind(keys, name)
}

// pyFuncSetAttr: only a plain function has a __dict__; a builtin or a bound
// method refuses a new attribute with CPython's text.
func (rt *jsrt) pyFuncSetAttr(f interface{}, name string, v interface{}) {
	if tn := pyTypeName(rt, f); tn != "function" {
		panic(&jsThrown{value: rt.pyExcInstance("AttributeError",
			"'"+tn+"' object has no attribute '"+name+"' and no __dict__ for setting new attributes")})
	}
	keys, vals, _ := dictParts(rt.pyFuncAttrs(f, true))
	if i := rt.dictFind(keys, name); i >= 0 {
		vals.elems[i] = v
		return
	}
	dictAppend(keys, vals, name, v)
}

// pyNoDictAttr: a builtin value with no __dict (None, a number, a str, a list, a
// dict, a set) refuses a new attribute, and its deletion, with a catchable
// AttributeError. Twin of pyNoDictAttr in languages/python-interpreter.abnf and
// languages/lib/python-rt.metajs.
func (rt *jsrt) pyNoDictAttr(obj interface{}, name string) {
	tn := pyTypeName(rt, obj)
	if isPyComplex(obj) {
		tn = "complex"
	}
	switch tn {
	case "NoneType", "int", "float", "bool", "str", "list", "dict", "set", "complex":
		panic(&jsThrown{value: rt.pyExcInstance("AttributeError",
			"'"+tn+"' object has no attribute '"+name+"' and no __dict__ for setting new attributes")})
	}
}

// pySlotsAllow applies __slots__ over the WHOLE resolution order, which is
// CPython's rule, not the nearest declaration's: a name is writable if ANY
// class on the MRO declares it in its own __slots__, and any name at all is
// writable as soon as one class on the MRO declares no __slots__ (that class
// gives its instances a __dict__). With `class A: __slots__ = ["s"]` and
// `class B(A): __slots__ = ["t"]`, writing B().s is allowed.
func (rt *jsrt) pySlotsAllow(cls *jsObject, name string) bool {
	all := true
	for _, k := range pyMRO(cls) {
		slots, has := k.props["__slots__"]
		if !has {
			all = false
			continue
		}
		if pySlotAllows(rt, slots, name) {
			return true
		}
	}
	return !all
}

// pySlotAllows answers whether ONE __slots__ declaration (a list, a
// tuple-as-list or a single string) names an attribute.
func pySlotAllows(rt *jsrt, slots interface{}, name string) bool {
	switch s := slots.(type) {
	case *jsArray:
		for _, e := range s.elems {
			if rt.toString(e) == name {
				return true
			}
		}
		return false
	case string:
		return s == name
	}
	return false
}

// pyAttrError builds the AttributeError instance a rejected __slots__ write
// raises, in the shape the except clauses match ({__class:{__name}, args}).
func (rt *jsrt) pyAttrError(cls *jsObject, name string) interface{} {
	return rt.pyExcInstance("AttributeError",
		"'"+rt.toString(cls.props["__name"])+"' object has no attribute '"+name+"'")
}

// pyGenValue is one {value, done} record from a generator step, in PYTHON's
// protocol: the yielded value, or a StopIteration carrying the body's return
// value. Shared by send()/next()/__next__() and by throw(), which answers the
// same way. The twin of python-rt.metajs's pyGenValue.
func (rt *jsrt) pyGenValue(res interface{}) interface{} {
	st, _ := res.(*jsObject)
	if st == nil {
		rt.fail("generator step failed")
	}
	if done, _ := st.props["done"].(bool); done {
		exc := rt.pyExcInstance("StopIteration", "").(*jsObject)
		// CPython's StopIteration carries NO argument when the body returned None
		// and exactly one when it returned a value: repr() says StopIteration()
		// against StopIteration(7). pyExcInstance's one-string args is every other
		// runtime exception's shape, so this one is rewritten here.
		v := st.props["value"]
		if v == nil || v == jsUndef {
			v = jsNull
		}
		exc.set("value", v)
		args := &jsArray{}
		if v != jsNull {
			args.elems = append(args.elems, v)
		}
		exc.set("args", args)
		panic(&jsThrown{value: exc})
	}
	return st.props["value"]
}

// pyMethodCall is obj.name(args) with Python's binding rule: an instance
// prepends itself, a class object does not (so Base.m(self) works).
func (rt *jsrt) pyMethodCall(target interface{}, name string, args []interface{}, kw interface{}) interface{} {
	// super().m(...): the lookup starts one past the defining class in the
	// instance's MRO, and the instance is prepended as self.
	if sup, isSup := pySuperObj(target); isSup {
		return rt.pySuperMCall(sup, name, args, kw)
	}
	// A METHOD CALL THROUGH A MODULE OBJECT - `builtins.len(xs)`, `math.sqrt(2)`
	// once math's own dispatcher has declined. The member is read and called, so
	// `builtins.len(xs)` and `builtins.len` answer the same function. It sits
	// FIRST because a module holds arbitrary member names, one of which is
	// literally `next`: without this the cursor and generator arms below would
	// claim it.
	if mod, isMod := pyIsModule(target); isMod {
		if v, has := rt.pyModuleAttr(mod, name); has {
			return rt.callPyValue(v, args, kw)
		}
		panic(&jsThrown{value: rt.pyExcInstance("AttributeError",
			"module '"+rt.toString(mod.props["__pymod"])+"' has no attribute '"+name+"'")})
	}
	// A CURSOR (iter(), map(), zip(), ...) is NOT a generator. The arm below is
	// a real type test, but layer 2's pyIsGen is STRUCTURAL and would answer
	// true for one, so this arm keeps the three engines saying the same thing.
	// CPython's list_iterator carries __next__ and __iter__ and nothing else;
	// every other name is a catchable AttributeError.
	if pyIterName(rt, target) != "" {
		it, _ := target.(*jsObject)
		switch name {
		case "__next__":
			return rt.pyGenValue(rt.pyItStep(it))
		case "__iter__":
			// iter(it) is it, and so is it.__iter__() - CPython's rule for
			// every iterator, and what makes `for x in it` after a partial
			// drain walk the REST of it.
			return target
		}
		panic(&jsThrown{value: rt.pyExcInstance("AttributeError",
			"'"+pyIterName(rt, target)+"' object has no attribute '"+name+"'")})
	}
	// A generator's own protocol. Python's send()/next() answer the YIELDED value
	// and raise StopIteration - carrying the body's return value - at the end,
	// where JavaScript's next() answers a {value, done} record.
	if g, ok := target.(*jsGenerator); ok {
		switch name {
		case "send", "next", "__next__":
			var sent interface{} = jsUndef
			if len(args) > 0 {
				sent = args[0]
			}
			return rt.pyGenValue(g.step(rt, sent))
		case "throw":
			// g.throw(exc): raise exc AT the yield the body is parked at
			// (throwInto). The record it answers is JavaScript's, so it is
			// unwrapped into Python's protocol exactly as send()'s is - a body
			// that catches it and returns ends with a StopIteration carrying
			// that return value.
			var tv interface{} = jsUndef
			if len(args) > 0 {
				tv = args[0]
			}
			// g.throw(SomeClass) instantiates it, exactly as `raise SomeClass`
			// does (js_pyraiseval).
			if _, isCls := pyClassObj(tv); isCls {
				tv = rt.callPyValue(tv, nil, jsUndef)
			}
			return rt.pyGenValue(g.throwInto(rt, tv))
		case "close":
			// CPython's generator.close() answers None, not a {value, done}
			// record - the record is JavaScript's g.return(v).
			g.closeBody(rt)
			return jsUndef
		}
	}
	// A method the builtin TYPE OBJECT itself carries - dict.fromkeys.
	if m, ok := rt.pyBuiltinClsMethod(target, name); ok {
		return rt.call(m, jsUndef, args)
	}
	// list.count(xs, 2) - the UNBOUND descriptor CALLED straight off the type
	// object rather than read as a value. The emitter routes a method call
	// through js_pyrxmcall and never through pyGetAttr, so the arm has to exist
	// on both paths or the read works and the call says "unknown method".
	if tn, isType := rt.pyClassName(target); isType {
		if samp := pyDescTypeSample(tn); samp != nil && rt.pyBuiltinAttr(samp, name) {
			if len(args) == 0 {
				panic(&jsThrown{value: rt.pyExcInstance("TypeError",
					"unbound method "+tn+"."+name+"() needs an argument")})
			}
			return rt.unwrap(rt.pyBoundBuiltinCall(args[0], name, args[1:]))
		}
		// str.__repr__(self): a dunder called off the type.
		if f, ok := pyBuiltinDelegate(tn, name); ok {
			return rt.call(f, jsUndef, args)
		}
	}
	if cls, ok := pyClassObj(target); ok {
		if m, found := pyLookup(cls, name); found {
			// C.s(...) / C.c(...) on a staticmethod or classmethod: pyDescGet
			// answers the very callable the attribute read hands back, so the
			// call path and the read path cannot drift.
			if pyDescKind(m) != "" {
				d := rt.pyDescGet(m, cls, name)
				return rt.call(d, jsUndef, rt.pyBindCall(d, args, kw))
			}
			if isCallable(m) {
				return rt.call(m, jsUndef, rt.pyBindCall(m, args, kw))
			}
			return rt.callPyValue(m, args, kw)
		}
		if name == "mro" {
			return rt.pyMroAttr(cls, "__mro__")
		}
		rt.fail("type object '%s' has no method '%s'", rt.toString(cls.props["__name"]), name)
	}
	if inst, cls, ok := pyInstance(target); ok {
		if v, found := inst.props[name]; found && isCallable(v) {
			return rt.call(v, jsUndef, rt.pyBindCall(v, args, kw)) // A plain function in an attribute.
		}
		if m, found := pyLookup(cls, name); found {
			// inst.s(...) on a staticmethod gets NO self, and inst.c(...) on a
			// classmethod gets the receiver's OWN class - checked before the
			// callable arm, which would otherwise prepend the instance.
			if pyDescKind(m) != "" {
				d := rt.pyDescGet(m, cls, name)
				return rt.call(d, jsUndef, rt.pyBindCall(d, args, kw))
			}
			// A builtin type's method reached by a subclass instance takes its
			// keywords straight to the builtin dispatcher.
			if pyIsBDelMethod(m) {
				return rt.pyMCall(pyBDelVal(target), name, args, kw)
			}
			if isCallable(m) {
				self := append([]interface{}{target}, args...)
				return rt.call(m, jsUndef, rt.pyBindCall(m, self, kw))
			}
			return rt.callPyValue(m, args, kw)
		}
		if _, isDict := inst.props["__dict"]; !isDict {
			rt.fail("'%s' object has no method '%s'", rt.toString(cls.props["__name"]), name)
		}
	}
	// list.count(x). rt.memberCall's `count` arm is KOTLIN's
	// `xs.count { pred }`. Python's counts EQUAL elements
	// (by ==, not by identity: [1.0].count(1) is 1), so it cannot share that arm
	// and is answered here, before the shared table is reached. The same arm is
	// pyDListCount in languages/lib/python-rt.metajs and the `count` arm of
	// mcall in languages/python-interpreter.abnf. str.count is CPython's
	// non-overlapping substring count (pystrmethod.go).
	// THE WHOLE DICT AND SET METHOD SURFACE. The twins are pySetMethod/
	// pyDictMethod in languages/lib/python-rt.metajs and in
	// languages/python-interpreter.abnf.
	//
	// A name the OTHER type owns raises AttributeError, which is what CPython
	// does - and the two "only" lists are narrower than the two method sets,
	// because pop/clear/copy/update are on BOTH types.
	if els, isSet := pySetElems(target); isSet {
		if pyIsSetMethod(name) {
			return rt.pySetMethod(els, name, args)
		}
		if pyIsDictOnly(name) {
			panic(&jsThrown{value: rt.pyExcInstance("AttributeError",
				"'"+pyTypeName(rt, target)+"' object has no attribute '"+name+"'")})
		}
	}
	if keys, vals, isDict := dictParts(target); isDict {
		if pyIsDictMethod(name) {
			return rt.pyDictMethod(keys, vals, name, args, kw)
		}
		if pyIsSetOnly(name) {
			panic(&jsThrown{value: rt.pyExcInstance("AttributeError",
				"'"+pyTypeName(rt, target)+"' object has no attribute '"+name+"'")})
		}
	}
	// THE WHOLE LIST METHOD SURFACE. Answered here, before the shared table,
	// because rt.memberCall is shared by many languages and its
	// `count`/`add`/`get`/`pop` arms are not Python's (pop(i) must honour the
	// index: [3,1,2].pop(0) is 3). The twins are pyListMethod
	// in languages/lib/python-rt.metajs and pyListMethod in
	// languages/python-interpreter.abnf.
	if arr, isArr := target.(*jsArray); isArr && pyIsListMethod(name) {
		return rt.pyListMethod(arr, name, args, kw)
	}
	// rt.memberCall is the SHARED member-call table that many languages land
	// in, and some of its names are ones Python cannot utter: `xs.sumOf(f)`
	// and `xs.size()` must raise AttributeError. They are denied HERE,
	// in python's own dispatcher and after the user-class arms above (so a Python
	// class that DEFINES a method called forEach is unaffected), rather than in
	// the shared table - isEmpty and removeLast are real Swift and Dart methods
	// and stay there and in languages/lib/{swift,dart}-rt.metajs.
	//
	// The denial is keyed by RECEIVER TYPE and not by name alone: `get` is a
	// real dict method (d.get(k, default)) and `add` is a real set method, so a
	// flat name switch would break both. The same table is pyDForeign in
	// languages/lib/python-rt.metajs, which is what keeps llvm.Run and the
	// native binary agreeing.
	if pyForeignMethod(target, name) {
		// A REAL, catchable AttributeError and not rt.fail: CPython's is catchable,
		// the interpreter half raises one (pyAttrErr in
		// languages/python-interpreter.abnf) and layer 2 throws one, so `try:
		// xs.add(3) / except AttributeError` has to behave the same in all three.
		panic(&jsThrown{value: rt.pyExcInstance("AttributeError",
			"'"+pyTypeName(rt, target)+"' object has no attribute '"+name+"'")})
	}
	return rt.memberCall(target, name, args)
}

// pyIsListMethod is the NAME SET of Python's list methods.
// It is a name test only, so hasattr()/getattr() can ask the same question the
// dispatcher answers. Keep in step with pyIsListMethod in
// languages/lib/python-rt.metajs and in languages/python-interpreter.abnf.
func pyIsListMethod(name string) bool {
	switch name {
	case "append", "pop", "clear", "copy", "count",
		"sort", "insert", "remove", "extend", "index", "reverse":
		return true
	case "decode": // A bytes value is a list of ints in this value model.
		return true
	}
	return false
}

// pyListMethod runs one of them. Every error it raises is a CATCHABLE Python
// exception with CPython 3.14's exact text, not rt.fail: `try: xs.remove(9) /
// except ValueError` has to behave the same in all three engines.
func (rt *jsrt) pyListMethod(arr *jsArray, name string, args []interface{}, kw interface{}) interface{} {
	n := len(arr.elems)
	switch name {
	case "decode":
		if keys, _, ok := dictParts(kw); ok && len(keys.elems) > 0 {
			args = rt.pyKwPos("decode", args, kw, []string{"encoding", "errors"}, 0)
		}
		return rt.pyBytesDecode(arr.elems, argAt(args, 0))
	case "append":
		arr.dropIdx()
		arr.elems = append(arr.elems, argAt(args, 0))
		return jsUndef
	case "clear":
		arr.dropIdx()
		arr.elems = arr.elems[:0]
		return jsUndef
	case "copy":
		return &jsArray{elems: append([]interface{}{}, arr.elems...)}
	case "reverse":
		arr.dropIdx()
		for i, j := 0, n-1; i < j; i, j = i+1, j-1 {
			arr.elems[i], arr.elems[j] = arr.elems[j], arr.elems[i]
		}
		return jsUndef
	case "count":
		// By ==, not by identity: [1.0, 2].count(1) is 1.
		if len(args) != 1 {
			panic(&jsThrown{value: rt.pyExcInstance("TypeError",
				"list.count() takes exactly one argument")})
		}
		c := 0
		for _, e := range arr.elems {
			if rt.pyEqual(e, args[0]) {
				c++
			}
		}
		return float64(c)
	case "pop":
		if n == 0 {
			panic(&jsThrown{value: rt.pyExcInstance("IndexError", "pop from empty list")})
		}
		i := n - 1
		if len(args) > 0 && !isUndefOrNull(args[0]) {
			i = int(rt.toNumber(args[0]))
			if i < 0 {
				i += n
			}
		}
		if i < 0 || i >= n {
			panic(&jsThrown{value: rt.pyExcInstance("IndexError", "pop index out of range")})
		}
		v := arr.elems[i]
		arr.dropIdx()
		arr.elems = append(arr.elems[:i], arr.elems[i+1:]...)
		return v
	case "insert":
		// CPython CLAMPS rather than raising: insert(-5, x) on a 2-list prepends
		// and insert(99, x) appends.
		i := int(rt.toNumber(argAt(args, 0)))
		if i < 0 {
			i += n
			if i < 0 {
				i = 0
			}
		}
		if i > n {
			i = n
		}
		arr.dropIdx()
		arr.elems = append(arr.elems, nil)
		copy(arr.elems[i+1:], arr.elems[i:])
		arr.elems[i] = argAt(args, 1)
		return jsUndef
	case "remove":
		for i, e := range arr.elems {
			if rt.pyEqual(e, argAt(args, 0)) {
				arr.dropIdx()
				arr.elems = append(arr.elems[:i], arr.elems[i+1:]...)
				return jsUndef
			}
		}
		panic(&jsThrown{value: rt.pyExcInstance("ValueError", "list.remove(x): x not in list")})
	case "extend":
		// pyElemsOf raises CPython's own "'int' object is not iterable" for a
		// non-iterable, and `xs.extend(xs)` doubles the list, as it does there.
		src := rt.pyElemsOf(argAt(args, 0))
		arr.dropIdx()
		arr.elems = append(arr.elems, src...)
		return jsUndef
	case "index":
		// start/stop clamp exactly like a slice, so index(x, -1) searches the
		// last element only; the ValueError carries no repr in 3.14.
		from, to := rt.pySliceRange(argAt(args, 1), argAt(args, 2), n)
		for i := from; i < to; i++ {
			if rt.pyEqual(arr.elems[i], argAt(args, 0)) {
				return float64(i)
			}
		}
		panic(&jsThrown{value: rt.pyExcInstance("ValueError", "list.index(x): x not in list")})
	case "sort":
		rt.pySortList(arr, args, kw)
		return jsUndef
	}
	rt.fail("unknown list method '%s'", name)
	return nil
}

// pySortList is list.sort(*, key=None, reverse=False), in place. CPython's is
// STABLE and its two parameters are KEYWORD-ONLY - a positional argument is a
// TypeError there, so it is one here too rather than being read as the key.
// reverse= is done by reversing, sorting ascending and reversing again, which is
// how CPython keeps equal elements in their original order under it; sorting
// with an inverted comparison would not. sorted() is this same function run on
// a copy, as in CPython, which is why its unknown-keyword error names sort().
func (rt *jsrt) pySortList(arr *jsArray, args []interface{}, kw interface{}) {
	if len(args) > 0 {
		panic(&jsThrown{value: rt.pyExcInstance("TypeError",
			"sort() takes no positional arguments")})
	}
	rt.pyKwOnly(kw, "sort", "key", "reverse")
	key := rt.pyKwArg(kw, "key")
	rev := rt.pyTruthyOf(rt.pyKwArg(kw, "reverse"))
	type pySortPair struct{ k, v interface{} }
	pairs := make([]pySortPair, len(arr.elems))
	for i, e := range arr.elems {
		j := i
		if rev {
			j = len(arr.elems) - 1 - i
		}
		e = arr.elems[j]
		k := e
		if !isUndefOrNull(key) {
			k = rt.callPyValue(key, []interface{}{e}, jsUndef)
		}
		pairs[i] = pySortPair{k: k, v: e}
	}
	// The same comparison js_pysorted uses, so sort() and sorted() can never
	// disagree inside one engine.
	sort.SliceStable(pairs, func(i, j int) bool { return rt.pyOrder("<", pairs[i].k, pairs[j].k) })
	out := make([]interface{}, len(pairs))
	for i, p := range pairs {
		j := i
		if rev {
			j = len(pairs) - 1 - i
		}
		out[j] = p.v
	}
	arr.dropIdx()
	arr.elems = out
}

// pyKwOnly raises CPython's TypeError for a keyword argument other than a or b.
func (rt *jsrt) pyKwOnly(kw interface{}, fname, a, b string) {
	keys, _, ok := dictParts(kw)
	if !ok {
		return
	}
	for _, k := range keys.elems {
		if n := rt.toString(k); n != a && n != b {
			panic(&jsThrown{value: rt.pyExcInstance("TypeError",
				fname+"() got an unexpected keyword argument '"+n+"'")})
		}
	}
}

// pyKwHas says whether a keyword argument was passed at all - None included,
// which the compiled halves represent as undefined.
func (rt *jsrt) pyKwHas(kw interface{}, name string) bool {
	keys, _, ok := dictParts(kw)
	if !ok {
		return false
	}
	for _, k := range keys.elems {
		if rt.toString(k) == name {
			return true
		}
	}
	return false
}

// pyKwArg reads one keyword argument out of the keyword dict a method call
// carries, or undefined when it was not passed.
func (rt *jsrt) pyKwArg(kw interface{}, name string) interface{} {
	keys, vals, ok := dictParts(kw)
	if !ok {
		return jsUndef
	}
	for i, k := range keys.elems {
		if rt.toString(k) == name && i < len(vals.elems) {
			return vals.elems[i]
		}
	}
	return jsUndef
}

// pyBuiltinClsName answers the name of a BUILTIN type object (what
// js_pybuiltincls hands out: a __name and nothing else) and "" for anything
// else, a user class object included - those carry a __mro and go through
// pyLookup.
func pyBuiltinClsName(v interface{}) string {
	o, ok := v.(*jsObject)
	if !ok {
		return ""
	}
	if _, isCls := o.props["__mro"]; isCls {
		return ""
	}
	n, _ := o.props["__name"].(string)
	return n
}

// pyBuiltinClsMethod is the method surface a builtin TYPE OBJECT itself carries -
// only dict.fromkeys so far, and it is the reason this exists at all. Its
// RECEIVER is the type rather than an instance, so it reaches neither the dict
// dispatcher nor a class MRO, and each of the three engines resolves such a
// receiver in an arm of its own. Both the mcall path (pyMethodCall) and the value
// path (pyGetAttr) need it, or it is reachable one way and not the other. Keep in
// step with pyBuiltinClsMethod in languages/lib/python-rt.metajs and in
// languages/python-interpreter.abnf.
func (rt *jsrt) pyBuiltinClsMethod(target interface{}, name string) (interface{}, bool) {
	if pyBuiltinClsName(target) != "dict" || name != "fromkeys" {
		return nil, false
	}
	return jsHostFunc("dict.fromkeys", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return rt.pyDictFromKeys(argAt(args, 0), argAt(args, 1))
	}), true
}

// dict.fromkeys(iterable[, value]): a fresh dict mapping every element of the
// iterable to the same value, None by default.
func (rt *jsrt) pyDictFromKeys(ks, dv interface{}) interface{} {
	keys, vals := &jsArray{}, &jsArray{}
	for _, k := range rt.pyElemsOf(ks) {
		if i := rt.pyDictFind(keys, k); i >= 0 {
			vals.elems[i] = dv
			continue
		}
		dictAppend(keys, vals, k, dv)
	}
	return &jsObject{props: map[string]interface{}{
		"__dict": true, "keys": keys, "vals": vals,
	}}
}

// ----- Python dict and set methods -----
//
// The NAME SETS first, so hasattr()/getattr() can ask exactly what the
// dispatcher answers. Keep in step with pyIsDictMethod/pyIsSetMethod in
// languages/lib/python-rt.metajs and in languages/python-interpreter.abnf.
// dict.fromkeys lives on the TYPE object (pyBuiltinClsMethod), not here.
func pyIsDictMethod(name string) bool {
	if name == "popitem" {
		return true
	}
	switch name {
	case "keys", "values", "items", "get", "pop", "clear", "copy", "setdefault", "update":
		return true
	}
	return false
}

func pyIsSetMethod(name string) bool {
	switch name {
	case "add", "pop", "clear", "copy", "remove", "discard", "union", "update", "issubset":
		return true
	}
	return pyIsSetAlgebra(name)
}

// pyIsSetAlgebra: the set algebra methods, which a dict does not have either.
func pyIsSetAlgebra(name string) bool {
	switch name {
	case "issuperset", "isdisjoint", "intersection", "difference", "symmetric_difference",
		"intersection_update", "difference_update", "symmetric_difference_update":
		return true
	}
	return false
}

// pyIsDictOnly and pyIsSetOnly are the names one of the two types owns and the
// other does NOT, which is exactly what CPython raises AttributeError for. They
// are narrower than the two sets above: pop, clear, copy and update are on BOTH.
func pyIsDictOnly(name string) bool {
	if name == "popitem" {
		return true
	}
	switch name {
	case "keys", "values", "items", "get", "setdefault":
		return true
	}
	return false
}

func pyIsSetOnly(name string) bool {
	switch name {
	case "add", "remove", "discard", "union", "issubset":
		return true
	}
	return pyIsSetAlgebra(name)
}

// pyDictMethod runs one of the dict methods. Every error is a CATCHABLE Python
// exception with CPython 3.14's exact text, not rt.fail, for the same reason the
// list surface is: `try: d.pop(k) / except KeyError` has to behave the same in
// all three engines.
func (rt *jsrt) pyDictMethod(keys, vals *jsArray, name string, args []interface{}, kw interface{}) interface{} {
	switch name {
	case "keys":
		return &jsArray{elems: append([]interface{}{}, keys.elems...)}
	case "values":
		return &jsArray{elems: append([]interface{}{}, vals.elems...)}
	case "items":
		// A list of [key, value] pairs in insertion order (each pair a 2-element
		// list, since the subset has no tuples; CPython answers a dict_items VIEW
		// of TUPLES). Every form - a written d.items() and getattr(d, "items")() -
		// arrives here.
		pairs := &jsArray{}
		for i, k := range keys.elems {
			pairs.elems = append(pairs.elems, &jsArray{elems: []interface{}{k, vals.elems[i]}})
		}
		return pairs
	case "get":
		if i := rt.pyDictFind(keys, argAt(args, 0)); i >= 0 {
			return vals.elems[i]
		}
		if len(args) > 1 {
			return args[1]
		}
		return jsUndef
	case "pop":
		if i := rt.pyDictFind(keys, argAt(args, 0)); i >= 0 {
			v := vals.elems[i]
			keys.elems = append(keys.elems[:i], keys.elems[i+1:]...)
			vals.elems = append(vals.elems[:i], vals.elems[i+1:]...)
			keys.dropIdx()
			return v
		}
		if len(args) > 1 {
			return args[1]
		}
		panic(&jsThrown{value: rt.pyExcInstanceV("KeyError", argAt(args, 0))})
	case "popitem":
		// The LAST inserted pair, removed - CPython's popitem is LIFO since 3.7.
		// The pair is a 2-element list, like items()' pairs, because this subset
		// has no tuple. Empty raises the KeyError whose argument is a sentence
		// rather than a key, which the arg-carrying KeyError renders as CPython
		// renders it.
		if len(keys.elems) == 0 {
			panic(&jsThrown{value: rt.pyExcInstanceV("KeyError", "popitem(): dictionary is empty")})
		}
		i := len(keys.elems) - 1
		k, v := keys.elems[i], vals.elems[i]
		keys.elems = keys.elems[:i]
		vals.elems = vals.elems[:i]
		keys.dropIdx()
		return &jsArray{elems: []interface{}{k, v}}
	case "clear":
		keys.elems = keys.elems[:0]
		vals.elems = vals.elems[:0]
		keys.dropIdx()
		return jsUndef
	case "copy":
		// SHALLOW, like CPython's: the values are shared, only the two spine
		// arrays are new. d.copy() is not d, and d.copy() == d.
		return &jsObject{props: map[string]interface{}{
			"__dict": true,
			"keys":   &jsArray{elems: append([]interface{}{}, keys.elems...)},
			"vals":   &jsArray{elems: append([]interface{}{}, vals.elems...)},
		}}
	case "setdefault":
		// INSERTS and returns; the default defaults to None, and it is inserted
		// too - d.setdefault("b") leaves {"b": None}.
		if len(args) < 1 {
			panic(&jsThrown{value: rt.pyExcInstance("TypeError",
				"setdefault expected at least 1 argument, got 0")})
		}
		if i := rt.pyDictFind(keys, args[0]); i >= 0 {
			return vals.elems[i]
		}
		var v interface{} = jsUndef
		if len(args) > 1 {
			v = args[1]
		}
		dictAppend(keys, vals, args[0], v)
		return v
	case "update":
		// update([mapping | iterable of pairs], **kwargs). All three forms, and
		// the KEYWORD one works only on a WRITTEN call: a bound builtin from
		// getattr() cannot carry keyword arguments (a signature-less builtin
		// receives its kw dict as a trailing positional and the wrapper cannot
		// tell that from a real argument), so getattr(d, "update")(a=1) is not
		// d.update(a=1); the written call is.
		if len(args) > 1 {
			panic(&jsThrown{value: rt.pyExcInstance("TypeError",
				fmt.Sprintf("update expected at most 1 argument, got %d", len(args)))})
		}
		if len(args) == 1 {
			rt.pyDictUpdateFrom(keys, vals, args[0])
		}
		if kkeys, kvals, ok := dictParts(kw); ok {
			for i, k := range kkeys.elems {
				rt.pyDictStore(keys, vals, k, kvals.elems[i])
			}
		}
		return jsUndef
	}
	rt.fail("unknown dict method '%s'", name)
	return nil
}

// pyDictStore is d[k] = v on the two spine arrays.
func (rt *jsrt) pyDictStore(keys, vals *jsArray, k, v interface{}) {
	if i := rt.pyDictFind(keys, k); i >= 0 {
		vals.elems[i] = v
		return
	}
	dictAppend(keys, vals, k, v)
}

// pyDictUpdateFrom is update()'s positional argument: a MAPPING copies key by
// key, anything else is read as an iterable of 2-element pairs. CPython's two
// error texts differ and are reproduced - the argument itself names its type,
// an ELEMENT does not.
func (rt *jsrt) pyDictUpdateFrom(keys, vals *jsArray, src interface{}) {
	if _, isSet := pySetElems(src); !isSet {
		if skeys, svals, ok := dictParts(src); ok {
			for i, k := range skeys.elems {
				rt.pyDictStore(keys, vals, k, svals.elems[i])
			}
			return
		}
	}
	for i, el := range rt.pyElemsOf(src) {
		k, v := rt.pyPairOf(el, i)
		rt.pyDictStore(keys, vals, k, v)
	}
}

func (rt *jsrt) pyPairOf(el interface{}, i int) (interface{}, interface{}) {
	if !rt.pyIsIterable(el) {
		panic(&jsThrown{value: rt.pyExcInstance("TypeError", "object is not iterable")})
	}
	pair := rt.pyElemsOf(el)
	if len(pair) != 2 {
		panic(&jsThrown{value: rt.pyExcInstance("ValueError",
			fmt.Sprintf("dictionary update sequence element #%d has length %d; 2 is required", i, len(pair)))})
	}
	return pair[0], pair[1]
}

// pyIsIterable is pyElemsOf's own accept test, asked without raising: an
// update() ELEMENT that is not iterable gets CPython's typeless "object is not
// iterable" rather than pyElemsOf's "'int' object is not iterable".
func (rt *jsrt) pyIsIterable(v interface{}) bool {
	switch v.(type) {
	case *jsGenerator, *jsArray, string:
		return true
	}
	if _, ok := pySetElems(v); ok {
		return true
	}
	if _, _, ok := dictParts(v); ok {
		return true
	}
	if o, ok := v.(*jsObject); ok && pyIterName(rt, o) != "" {
		return true
	}
	if _, cls, ok := pyInstance(v); ok {
		m, found := pyLookup(cls, "__iter__")
		return found && isCallable(m)
	}
	return false
}

// pySetMethod runs one of the set methods.
func (rt *jsrt) pySetMethod(els *jsArray, name string, args []interface{}) interface{} {
	switch name {
	case "add":
		rt.pySetAdd(els, argAt(args, 0)) // Already-present member: no-op, returns None.
		return jsUndef
	case "clear":
		els.elems = els.elems[:0]
		els.dropIdx()
		return jsUndef
	case "copy":
		return newPySet(&jsArray{elems: append([]interface{}{}, els.elems...)}) // SHALLOW.
	case "pop":
		// pop() removes and returns an ARBITRARY element - CPython promises no
		// order at all. This project's sets keep insertion order (it is what
		// makes a set render deterministically, which the byte-identical
		// goja/frozen invariant needs), so the first element is the one taken.
		// Assert on membership and size, never on which element.
		if len(els.elems) == 0 {
			panic(&jsThrown{value: rt.pyExcInstance("KeyError", "pop from an empty set")})
		}
		v := els.elems[0]
		els.elems = append(els.elems[:0], els.elems[1:]...)
		els.dropIdx()
		return v
	case "remove", "discard":
		// remove() raises KeyError when the member is absent and discard() does
		// not. That difference is the whole reason both exist.
		rt.pyHashCheckMember(argAt(args, 0))
		for i, e := range els.elems {
			if rt.pyEqual(e, argAt(args, 0)) {
				els.elems = append(els.elems[:i], els.elems[i+1:]...)
				els.dropIdx()
				return jsUndef
			}
		}
		if name == "remove" {
			panic(&jsThrown{value: rt.pyExcInstanceV("KeyError", argAt(args, 0))})
		}
		return jsUndef
	case "union", "update":
		// union(*others) takes ANY iterable, any number of them, and does not
		// mutate the receiver: s | t insists both sides be sets, s.union(t) does
		// not, so {1}.union([2], "a") is a set of three. update() is the same
		// walk IN PLACE, returning None.
		out := els
		if name == "union" {
			out = &jsArray{elems: append([]interface{}{}, els.elems...)}
		}
		for _, a := range args {
			for _, e := range rt.pyElemsOf(a) {
				rt.pySetAdd(out, e)
			}
		}
		if name == "union" {
			return newPySet(out)
		}
		return jsUndef
	case "issubset":
		// issubset(other) takes ANY iterable, as union does.
		other := &jsArray{elems: rt.pyElemsOf(argAt(args, 0))}
		for _, e := range els.elems {
			if !rt.pySetHas(other, e) {
				return false
			}
		}
		return true
	case "issuperset", "isdisjoint":
		// issuperset(other) and isdisjoint(other) take ANY iterable too.
		want := name == "isdisjoint"
		for _, e := range rt.pyElemsOf(argAt(args, 0)) {
			if rt.pySetHas(els, e) == want {
				return false
			}
		}
		return true
	case "intersection", "difference", "symmetric_difference":
		// intersection / difference (any number of iterables) and
		// symmetric_difference (exactly one) answer a NEW set; each _update form
		// writes the same answer back into the receiver and returns None.
		return newPySet(rt.pySetAlgebra(els, name, args))
	case "intersection_update", "difference_update", "symmetric_difference_update":
		upd := rt.pySetAlgebra(els, strings.TrimSuffix(name, "_update"), args)
		els.elems = els.elems[:0]
		els.dropIdx()
		for _, e := range upd.elems {
			rt.pySetAdd(els, e)
		}
		return jsUndef
	}
	rt.fail("unknown set method '%s'", name)
	return nil
}

// pySetAlgebra: the members of the answer, receiver order first, then
// (symmetric_difference) the argument's. Twin of pySetAlgebra in
// languages/lib/python-rt.metajs and languages/python-interpreter.abnf.
func (rt *jsrt) pySetAlgebra(els *jsArray, name string, args []interface{}) *jsArray {
	if name == "symmetric_difference" {
		o := &jsArray{}
		for _, e := range rt.pyElemsOf(argAt(args, 0)) {
			rt.pySetAdd(o, e)
		}
		sym := &jsArray{}
		for _, e := range els.elems {
			if !rt.pySetHas(o, e) {
				rt.pySetAdd(sym, e)
			}
		}
		for _, e := range o.elems {
			if !rt.pySetHas(els, e) {
				rt.pySetAdd(sym, e)
			}
		}
		return sym
	}
	out := &jsArray{elems: append([]interface{}{}, els.elems...)}
	keep := name == "intersection"
	for _, a := range args {
		other := &jsArray{elems: rt.pyElemsOf(a)}
		kept := &jsArray{}
		for _, e := range out.elems {
			if rt.pySetHas(other, e) == keep {
				rt.pySetAdd(kept, e)
			}
		}
		out = kept
	}
	return out
}

// pyBuiltinAttr is the hasattr()/getattr() question for BUILTIN methods
// (hasattr([3,1,2], "count") and hasattr("s", "upper") are True). This is
// the same question the DISPATCHER answers, asked by name: the
// str table (pysIsStrMethod, abnf/pystrmethod.go), the list table
// (pyIsListMethod above), and the two dict/set arms of pyMethodCall. Keep in
// step with pyDBuiltinAttr in languages/lib/python-rt.metajs and
// pyBuiltinAttr in languages/python-interpreter.abnf.
//
// A builtin that this project does not HAVE stays False on purpose - hasattr(3,
// "bit_length") is False here and True in CPython - because the alternative is
// hasattr answering True for a call that then aborts.
func (rt *jsrt) pyBuiltinAttr(target interface{}, name string) bool {
	if _, isStr := target.(string); isStr {
		return pysIsStrMethod(name)
	}
	if _, isArr := target.(*jsArray); isArr {
		return pyIsListMethod(name)
	}
	// A set is a dict whose keys are its members in the interpreter half, so it
	// is asked FIRST here too and the two name sets stay disjoint.
	if _, isSet := pySetElems(target); isSet {
		return pyIsSetMethod(name)
	}
	if _, _, isDict := dictParts(target); isDict {
		return pyIsDictMethod(name)
	}
	return false
}

// pyBoundBuiltin is what getattr() hands back for one of those names: CPython's
// `<built-in method count of list object ...>`, spelled here as a host function
// that re-enters the dispatcher with the receiver captured. Keyword arguments
// are NOT carried through it (a signature-less builtin receives its keyword dict
// as a trailing positional, which this cannot tell from a real argument), so
// getattr(xs, "sort")(reverse=True) is not the same as xs.sort(reverse=True);
// the direct call is.
func (rt *jsrt) pyBoundBuiltin(target interface{}, name string) interface{} {
	h := jsHostFunc("pybound."+name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
		// A STRING's methods are NOT in pyMethodCall: abnf/pystrmethod.go's
		// registrar installs them by WRAPPING the js_pymcall extern, so
		// re-entering pyMethodCall directly reaches the shared member table and
		// dies with "unknown String method: upper". Re-enter through the extern
		// instead, which is the same path a written `s.upper()` takes.
		if _, isStr := target.(string); isStr {
			if m := pyExternMaps[rt]; m != nil {
				if f := m["js_pymcall"]; f != nil {
					return rt.unwrap(f([]uint64{rt.wrap(target), rt.wrap(name),
						rt.wrap(&jsArray{elems: args}), rt.wrap(jsUndef)}))
				}
			}
		}
		return rt.pyMethodCall(target, name, args, jsUndef)
	})
	// A builtin method has no underlying python function, and that nil is what
	// tells __func__ to raise and the repr to say <built-in method ...>.
	h.pybm = &pyBoundInfo{name: name, self: target, fn: nil, typ: pyTypeName(rt, target)}
	return h
}

// pyExternMaps remembers the extern table each run installed. The table is
// filled by rxInstallExterns and then WRAPPED by the per-language registrars
// (rxExtraExterns), so a lookup made at CALL time - which is the only time
// pyBoundBuiltin looks - sees the fully wrapped entry no matter what order the
// registrars ran in. Only python's getattr()-produced bound methods read it.
var pyExternMaps = map[*jsrt]map[string]func(args []uint64) uint64{}

func init() {
	rxExtraExterns = append(rxExtraExterns, func(rt *jsrt, m map[string]func(args []uint64) uint64) {
		pyExternMaps[rt] = m
	})
}

// pyForeignMethod answers whether NAME is a member-table method that Python's
// receiver type does not have. Keep in step with pyDForeign in
// languages/lib/python-rt.metajs.
func pyForeignMethod(target interface{}, name string) bool {
	switch target.(type) {
	case *jsArray:
		// Kotlin's, Java's, Dart's and Swift's list methods. `count` is NOT here:
		// Python has list.count(x) and it is answered above.
		switch name {
		case "isEmpty", "removeLast", "sumOf", "forEach",
			"add", "size", "get", "contains", "map", "filter", "any":
			return true
		}
	case string:
		// Java's and Kotlin's string methods. Python spells them len(s), s[i],
		// s == t, s[a:b] and s.index(t).
		switch name {
		case "isEmpty", "length", "charAt", "equals", "substring", "indexOf":
			return true
		}
	default:
		switch name {
		case "isEmpty", "removeLast", "sumOf", "forEach":
			return true
		}
	}
	return false
}

// callPyValue calls a value that may itself be a class or a __call__ instance.
// pyConvArgs is the argument array for a builtin type's conversion closure: one
// marked __convkw (dict) carries a signature and takes its arguments bound, the
// others take the positionals as they are.
// A conversion without one takes no keyword arguments at all, as in CPython
// (type, whose error differs, is left alone).
func (rt *jsrt) pyConvArgs(cls *jsObject, conv interface{}, args []interface{}, kw interface{}) []interface{} {
	if _, bound := cls.props["__convkw"]; bound {
		return rt.pyBindCall(conv, args, kw)
	}
	if keys, _, ok := dictParts(kw); ok && len(keys.elems) > 0 {
		if name := rt.toString(cls.props["__name"]); name != "type" {
			panic(&jsThrown{value: rt.pyExcInstance("TypeError", name+"() takes no keyword arguments")})
		}
	}
	return args
}

func (rt *jsrt) callPyValue(v interface{}, args []interface{}, kw interface{}) interface{} {
	if sup, isSup := pySuperObj(v); isSup {
		return rt.pySuperCall(sup, args)
	}
	// A BUILTIN TYPE OBJECT is callable and CONVERTS: the emitter binds int/str/
	// float/... to js_pybuiltincls's stable class object with the conversion
	// closure hung on it as __conv (declBind in python-to-llvm-ir.abnf). js_pycall
	// had this arm and this function did not, so `map(str, xs)` - a builtin class
	// passed as a function - died with "call of a non function value".
	if o, isObj := v.(*jsObject); isObj {
		if conv, has := o.props["__conv"]; has && isCallable(conv) {
			return rt.call(conv, jsUndef, rt.pyConvArgs(o, conv, args, kw))
		}
	}
	if cls, ok := pyClassObj(v); ok {
		return rt.pyNewInstance(cls, args, kw)
	}
	if _, cls, ok := pyInstance(v); ok {
		if m, found := pyLookup(cls, "__call__"); found && isCallable(m) {
			self := append([]interface{}{v}, args...)
			return rt.call(m, jsUndef, rt.pyBindCall(m, self, kw))
		}
	}
	rt.pyCheckCallable(v)
	return rt.call(v, jsUndef, rt.pyBindCall(v, args, kw))
}

// pyCheckCallable raises CPython's TypeError for a call of a value that is not a
// function ("'C' object is not callable"). Twin of pyNotCallable in
// languages/lib/python-rt.metajs; the interpreter's is in pyCall.
func (rt *jsrt) pyCheckCallable(v interface{}) {
	if !isCallable(v) {
		panic(&jsThrown{value: rt.pyExcInstance("TypeError",
			"'"+pyTypeName(rt, v)+"' object is not callable")})
	}
}

// Python's own names for three shared externs: js_pylenof is len() and
// js_pyiterof / js_pyunpack are js_pyiter, each raising CPython's TypeError ("object
// of type 'C' has no len()", "'C' object is not iterable", "cannot unpack
// non-iterable C object") where the shared extern's failure is not Python's.
// Twins of the same names in languages/lib/python-rt.metajs.
func init() {
	rxExtraExterns = append(rxExtraExterns, func(rt *jsrt, m map[string]func(args []uint64) uint64) {
		// js_pyiterof / js_pyunpack are js_pyiter for a for loop or a
		// comprehension and for an unpacking target, raising CPython's TypeError
		// for a value that is not iterable.
		m["js_pyiterof"] = func(a []uint64) uint64 {
			if v := rt.unwrap(a[0]); !rt.pyIsIterable(v) {
				panic(&jsThrown{value: rt.pyExcInstance("TypeError",
					"'"+pyTypeName(rt, v)+"' object is not iterable")})
			}
			return m["js_pyiter"](a)
		}
		m["js_pyunpack"] = func(a []uint64) uint64 { // (value, target count, has a *rest)
			v := rt.unwrap(a[0])
			if !rt.pyIsIterable(v) {
				panic(&jsThrown{value: rt.pyExcInstance("TypeError",
					"cannot unpack non-iterable "+pyTypeName(rt, v)+" object")})
			}
			h := m["js_pyiter"](a[:1])
			if arr, ok := rt.unwrap(h).(*jsArray); ok {
				rt.pyUnpackCheck(v, len(arr.elems), int(rt.toNumber(rt.unwrap(a[1]))), rt.truthy(rt.unwrap(a[2])))
			}
			return h
		}
		m["js_pylenof"] = func(a []uint64) uint64 {
			v := rt.unwrap(a[0])
			sized := false
			switch v.(type) {
			case string, *jsArray:
				sized = true
			}
			if _, cls, isInst := pyInstance(v); isInst {
				lm, found := pyLookup(cls, "__len__")
				sized = found && isCallable(lm)
			} else if _, isSet := pySetElems(v); isSet {
				sized = true
			} else if _, _, isDict := dictParts(v); isDict {
				sized = true
			}
			if !sized {
				panic(&jsThrown{value: rt.pyExcInstance("TypeError",
					"object of type '"+pyTypeName(rt, v)+"' has no len()")})
			}
			return m["js_pylen"](a)
		}
	})
}

// pyIsInstance is isinstance(v, C), with a list/tuple of classes accepted.
func (rt *jsrt) pyIsInstance(v interface{}, target interface{}) bool {
	if arr, ok := target.(*jsArray); ok {
		for _, e := range arr.elems {
			if rt.pyIsInstance(v, e) {
				return true
			}
		}
		return false
	}
	_, cls, ok := pyInstance(v)
	if !ok {
		// A builtin value against a builtin class object (int, str, ...).
		if tc, isCls := target.(*jsObject); isCls {
			if n, has := tc.props["__name"]; has && !hasKey(tc, "__mro") {
				return pyBuiltinSub(pyTypeName(rt, v), rt.toString(n))
			}
		}
		return false
	}
	// Every class derives from object, so isinstance(C(), object) is True.
	if tc, isCls := target.(*jsObject); isCls && !hasKey(tc, "__mro") {
		if n, has := tc.props["__name"]; has && rt.toString(n) == "object" {
			return true
		}
	}
	for _, k := range pyMRO(cls) {
		if k == target {
			return true
		}
	}
	// Builtin exception classes are matched by name, the way js_is_type does.
	if tc, isCls := target.(*jsObject); isCls {
		want, has := tc.props["__name"]
		if has {
			for _, k := range pyMRO(cls) {
				if n, ok := k.props["__name"]; ok && rt.toString(n) == rt.toString(want) {
					return true
				}
			}
		}
	}
	return false
}

func hasKey(o *jsObject, k string) bool {
	_, ok := o.props[k]
	return ok
}

// ----- the Python iterator object ------------------------------------------
//
// iter() / enumerate() / zip() / map() / filter() / reversed() all answer an
// ITERATOR in CPython, and next() drives it. There is no such value in this
// project: js_pyiter MATERIALIZES, and the only steppable thing is a generator.
//
// So an iterator is built to look EXACTLY like a generator to every engine: a
// plain object carrying a callable "next" that answers {value, done}. That is
// the structural test the compiled for-loop already emits (python-to-llvm-ir.abnf
// makeForStmt reads js_get(v, "next") and compares its typeof to "function"),
// the test layer 2's pyIsGen already makes, and the protocol pyMethodCall's
// generator arm already speaks - so a for loop over one is LAZY and `it.__next__()`
// raises StopIteration, with no change to any of that machinery.
//
// The SOURCE is materialized, though, and that is the line drawn here: CPython's
// map/filter/zip/enumerate are lazy in their input as well, so
// `map(f, endless())` works there and hangs here. iter(gen) answers the
// GENERATOR ITSELF - CPython's rule too - so `for x in endless(): break`
// stays lazy. The same three functions are
// pyMkIter / pyItStep in languages/lib/python-rt.metajs and in
// languages/python-interpreter.abnf.
//
// __pyit is the type NAME, which is what type(it).__name__ and the printed form
// read; __a is the materialized element array and __i the cursor.
func (rt *jsrt) pyMkIter(name string, elems []interface{}) *jsObject {
	it := newJSObject()
	it.set("__pyit", name)
	it.set("__a", &jsArray{elems: elems})
	it.set("__i", float64(0))
	it.set("next", jsHostFunc("pyiter.next", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return rt.pyItStep(it)
	}))
	return it
}

// One step of an iterator object: the {value, done} record JavaScript's iterator
// protocol - and the floor's generator cell - answer.
func (rt *jsrt) pyItStep(it *jsObject) *jsObject {
	out := newJSObject()
	a, _ := it.props["__a"].(*jsArray)
	i := int(rt.toNumber(it.props["__i"]))
	if a == nil || i >= len(a.elems) {
		out.set("value", jsUndef)
		out.set("done", true)
		return out
	}
	it.set("__i", float64(i+1))
	out.set("value", a.elems[i])
	out.set("done", false)
	return out
}

// pyIterName answers an iterator object's type name, or "" for anything else.
func pyIterName(rt *jsrt, v interface{}) string {
	o, ok := v.(*jsObject)
	if !ok {
		return ""
	}
	n, has := o.props["__pyit"]
	if !has {
		return ""
	}
	return rt.toString(n)
}

// pyTailArgs is args[1:] for a variadic builtin, empty when there is no tail.
func pyTailArgs(args []interface{}) []interface{} {
	if len(args) < 2 {
		return nil
	}
	return args[1:]
}

// pyTruthyOf is Python truthiness for the container types (an empty list, dict
// or set is FALSE, where JavaScript calls every object truthy) - the js_pybool
// arms, reused by all(), any() and filter().
func (rt *jsrt) pyTruthyOf(v interface{}) bool {
	if els, ok := pySetElems(v); ok {
		return len(els.elems) > 0
	}
	if arr, ok := v.(*jsArray); ok {
		return len(arr.elems) > 0
	}
	if keys, _, ok := dictParts(v); ok {
		return len(keys.elems) > 0
	}
	if _, cls, ok := pyInstance(v); ok {
		return rt.pyUserTruthy(cls, v)
	}
	return rt.truthy(v)
}

// argArray unwraps the argument array a signature-less builtin (declBuiltin with
// argc 0) receives.
func (rt *jsrt) argArray(v interface{}) []interface{} {
	if arr, ok := v.(*jsArray); ok {
		return arr.elems
	}
	return nil
}

// pyElemsOf is js_pyiter's element list, as a Go slice: a generator drains, a
// cursor gives WHAT IS LEFT of it (so next(it) then list(it) does not repeat an
// element), a set copies, a dict gives its keys and a string its characters.
func (rt *jsrt) pyElemsOf(v interface{}) []interface{} {
	if g, ok := v.(*jsGenerator); ok {
		return g.drain(rt).elems
	}
	if o, ok := v.(*jsObject); ok && pyIterName(rt, o) != "" {
		a, _ := o.props["__a"].(*jsArray)
		i := int(rt.toNumber(o.props["__i"]))
		if a == nil || i >= len(a.elems) {
			o.set("__i", float64(0))
			return nil
		}
		out := append([]interface{}{}, a.elems[i:]...)
		o.set("__i", float64(len(a.elems))) // Reading an iterator EXHAUSTS it.
		return out
	}
	if els, ok := pySetElems(v); ok {
		return append([]interface{}{}, els.elems...)
	}
	switch o := v.(type) {
	case *jsArray:
		return o.elems
	case string:
		return pyStrChars(rt, o)
	}
	if keys, _, ok := dictParts(v); ok {
		return append([]interface{}{}, keys.elems...)
	}
	if it, ok := rt.pyUserIter(v); ok {
		return rt.pyDrainIter(it)
	}
	panic(&jsThrown{value: rt.pyExcInstance("TypeError",
		"'"+pyTypeName(rt, v)+"' object is not iterable")})
}

// pyIsGenerator is the generator test as a predicate, for the callers that only
// need the yes/no.
func pyIsGenerator(v interface{}) bool {
	_, ok := v.(*jsGenerator)
	return ok
}

// pyIterKind is the CPython type NAME of the cursor iter() answers for a value.
// It is what type(it).__name__ reads and what the AttributeError a cursor raises
// interpolates, so the names are CPython 3.14's and not invented: a str whose
// code points are all ASCII gets str_ascii_iterator and any other str
// str_iterator (CPython splits the two implementations), a dict gets
// dict_keyiterator and a set set_iterator. The twins are pyIterKind in
// languages/lib/python-rt.metajs and pyItKind in languages/python-interpreter.abnf.
func pyIterKind(rt *jsrt, v interface{}) string {
	switch t := v.(type) {
	case string:
		for _, r := range t {
			if r > 127 {
				return "str_iterator"
			}
		}
		return "str_ascii_iterator"
	case *jsArray:
		return "list_iterator"
	}
	if _, ok := pySetElems(v); ok {
		return "set_iterator"
	}
	if _, _, ok := dictParts(v); ok {
		return "dict_keyiterator"
	}
	return "iterator"
}

// pyStopIter is next()'s end: the default argument when there is one, else the
// StopIteration CPython raises.
func (rt *jsrt) pyStopIter(args []interface{}, val interface{}) uint64 {
	if len(args) > 1 {
		return rt.wrap(args[1])
	}
	exc := rt.pyExcInstance("StopIteration", "")
	if o, ok := exc.(*jsObject); ok {
		// CPython's StopIteration carries no argument at all unless the generator
		// returned a value; see pyGenValue.
		args := &jsArray{}
		if isUndefOrNull(val) {
			o.set("value", jsNull)
		} else {
			o.set("value", val)
			args.elems = append(args.elems, val)
		}
		o.set("args", args)
	}
	panic(&jsThrown{value: exc})
}

// pyHasAttr is hasattr(), and it MIRRORS pyGetAttr's lookup rather than calling
// it and catching: pyGetAttr reports a miss with rt.fail, a Go panic carrying a
// string, and recovering from that would swallow every unrelated runtime error
// too. Layer 2's fail() is not catchable at all, so the structural form is also
// the only one the two engines can both have. Keep in step with pyDHasAttr in
// languages/lib/python-rt.metajs.
func (rt *jsrt) pyHasAttr(obj interface{}, name string) bool {
	if sup, isSup := pySuperObj(obj); isSup {
		rt.pySuperCheck(sup)
		_, found := rt.pySuperFind(sup, name)
		return found
	}
	if cls, ok := pyClassObj(obj); ok {
		if name == "__name__" || name == "__qualname__" || name == "__mro__" || name == "__bases__" || name == "mro" {
			return true
		}
		_, found := pyLookup(cls, name)
		return found
	}
	if inst, cls, ok := pyInstance(obj); ok {
		if name == "__class__" || name == "__dict__" {
			return true
		}
		if _, found := inst.props[name]; found {
			return true
		}
		_, found := pyLookup(cls, name)
		return found
	}
	// A MODULE object, mirroring pyGetAttr's module arm - hasattr(math, "sqrt")
	// has to agree with what `math.sqrt` answers.
	if mod, isMod := pyIsModule(obj); isMod {
		_, has := rt.pyModuleAttr(mod, name)
		return has
	}
	// What pyGetAttr answers on a builtin TYPE OBJECT: dict.fromkeys, and the
	// unbound methods (list.count).
	if _, ok := rt.pyBuiltinClsMethod(obj, name); ok {
		return true
	}
	if tn, isType := rt.pyClassName(obj); isType {
		if samp := pyDescTypeSample(tn); samp != nil && rt.pyBuiltinAttr(samp, name) {
			return true
		}
	}
	// The BUILT-IN methods of str / list / dict / set, and
	// __class__, which EVERY value has (None.__class__ is NoneType).
	if name == "__class__" || rt.pyBuiltinAttr(obj, name) {
		return true
	}
	if bi := pyBoundOf(obj); bi != nil {
		switch name {
		case "__name__", "__qualname__", "__self__":
			return true
		case "__func__":
			return bi.fn != nil
		}
	}
	if name == "__name__" || name == "__qualname__" {
		if o, ok := obj.(*jsObject); ok {
			if _, has := o.props["__name"]; has {
				return true
			}
		}
		// Both tables, matching pyGetAttr above - hasattr must
		// answer exactly what getattr can answer, or the three-argument getattr
		// (which asks this first) hands back its default for a readable name.
		if _, ok := rt.pyBuiltinRender[obj]; ok {
			return true
		}
		if _, ok := rt.pyFuncNames[obj]; ok {
			return true
		}
	}
	if isCallable(obj) {
		if _, i := rt.pyFuncAttrIdx(obj, name); i >= 0 {
			return true
		}
		return name == "__dict__" && pyTypeName(rt, obj) == "function"
	}
	if o, ok := obj.(*jsObject); ok {
		_, has := o.props[name]
		return has
	}
	return false
}

// pySeqIdx is the position a list or str subscript names: an int (a bool is
// one), or CPython's TypeError naming the key's type. Twin of pySeqIdx in
// python-interpreter.abnf and lib/python-rt.metajs. js_pyset is shared, so a
// key of a type python never makes (another language's sized int) still reads
// as its number.
func (rt *jsrt) pySeqIdx(c, k interface{}) int {
	switch x := k.(type) {
	case float64:
		return int(x)
	case bool:
		if x {
			return 1
		}
		return 0
	case *jsBigInt:
		panic(&jsThrown{value: rt.pyExcInstance("IndexError", "cannot fit 'int' into an index-sized integer")})
	case string, jsUndefT, jsNullT, *jsPyFlo, *jsObject, *jsArray:
	default:
		if !isCallable(k) {
			return int(rt.toNumber(k))
		}
	}
	tn := pyTypeName(rt, k)
	if _, isStr := c.(string); isStr {
		panic(&jsThrown{value: rt.pyExcInstance("TypeError", "string indices must be integers, not '"+tn+"'")})
	}
	panic(&jsThrown{value: rt.pyExcInstance("TypeError", "list indices must be integers or slices, not "+tn)})
}

// pyNoItemOp raises CPython's TypeError for an item write or delete on a value
// with no such slot. Twin of pyNoItemOp in python-interpreter.abnf, which
// explains the two spellings of the deletion message.
func (rt *jsrt) pyNoItemOp(c interface{}, del bool) {
	tn := pyTypeName(rt, c)
	if !del {
		panic(&jsThrown{value: rt.pyExcInstance("TypeError", "'"+tn+"' object does not support item assignment")})
	}
	how := "does not"
	_, isStr := c.(string)
	_, isSet := pySetElems(c)
	_, _, isInst := pyInstance(c)
	if isStr || isSet || isInst {
		how = "doesn't"
	}
	panic(&jsThrown{value: rt.pyExcInstance("TypeError", "'"+tn+"' object "+how+" support item deletion")})
}

// pyRangeArg is one range() argument: an int (a bool is one), anything else the
// TypeError CPython raises. Twin of pyListIdx in lib/python-rt.metajs.
func (rt *jsrt) pyRangeArg(v interface{}) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case bool:
		if x {
			return 1
		}
		return 0
	}
	panic(&jsThrown{value: rt.pyExcInstance("TypeError",
		"'"+pyTypeName(rt, v)+"' object cannot be interpreted as an integer")})
}

// pyRadix is bin/hex/oct: CPython puts the sign OUTSIDE the prefix (bin(-5) is
// '-0b101') and its ints are arbitrary precision, so the digits come from
// math/big rather than from a double.
func (rt *jsrt) pyRadix(v interface{}, base int, prefix string) string {
	b, ok := pyBigOperand(v)
	if !ok {
		panic(&jsThrown{value: rt.pyExcInstance("TypeError",
			"'"+pyTypeName(rt, v)+"' object cannot be interpreted as an integer")})
	}
	s := b.Text(base)
	if len(s) > 0 && s[0] == '-' {
		return "-" + prefix + s[1:]
	}
	return prefix + s
}

// pyRoundHalfEven is CPython's round(): HALF TO EVEN, so round(0.5) is 0,
// round(1.5) is 2 and round(2.5) is 2. Written on the exact decimal expansion
// through strconv so it is right past 2^53 as well, which the arithmetic
// spelling x*10^n rounded and divided back is not.
func pyRoundHalfEven(f float64, nd int) float64 {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return f
	}
	s := strconv.FormatFloat(f, 'f', nd, 64) // FormatFloat rounds half to EVEN.
	out, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return f
	}
	return out
}

// pyAsciiOf turns repr()'s answer into ascii()'s: every code point past ASCII
// becomes \xNN, \uNNNN or \UNNNNNNNN.
func pyAsciiOf(s string) string {
	out := ""
	for _, r := range s {
		switch {
		case r < 0x80:
			out += string(r)
		case r < 0x100:
			out += fmt.Sprintf("\\x%02x", r)
		case r < 0x10000:
			out += fmt.Sprintf("\\u%04x", r)
		default:
			out += fmt.Sprintf("\\U%08x", r)
		}
	}
	return out
}

// pyBuiltinSub is the subclass relation over the BUILTIN type names, which is
// the whole of Python's builtin hierarchy that this subset models:
//
//	object   is the base of everything          isinstance(1, object)  -> True
//	bool     derives from int                   isinstance(True, int)  -> True
//
// bool subclassing int is not a curiosity - it is why True + 1 is 2 and why
// {True: "t"}[1] hits.
//
// Keep in step with pyBuiltinSub in languages/lib/python-rt.metajs and in
// languages/python-interpreter.abnf.
func pyBuiltinSub(child, base string) bool {
	if child == base || base == "object" {
		return true
	}
	return child == "bool" && base == "int"
}

// pyIsSubclass is issubclass(C, B): a class object (or a tuple of them) against
// a class object. Keep in step with pyDIsSubclass in
// languages/lib/python-rt.metajs and clsIsSub in
// languages/python-interpreter.abnf.
func (rt *jsrt) pyIsSubclass(c, target interface{}) bool {
	if arr, ok := target.(*jsArray); ok {
		for _, e := range arr.elems {
			if rt.pyIsSubclass(c, e) {
				return true
			}
		}
		return false
	}
	// A BUILTIN class object (int, str, object) carries a __name and NO __mro, so
	// pyClassObj rejects it: both kinds have to be accepted here.
	cls, ok := c.(*jsObject)
	if !ok || !hasKey(cls, "__name") {
		panic(&jsThrown{value: rt.pyExcInstance("TypeError", "issubclass() arg 1 must be a class")})
	}
	tc, isCls := target.(*jsObject)
	if !isCls {
		panic(&jsThrown{value: rt.pyExcInstance("TypeError",
			"issubclass() arg 2 must be a class, a tuple of classes, or a union")})
	}
	tn := ""
	if n, has := tc.props["__name"]; has {
		tn = rt.toString(n)
	}
	// A BUILTIN class object on either side has no __mro: the relation between
	// two of them is pyBuiltinSub, and a user class against `object` is True.
	if !hasKey(cls, "__mro") && !hasKey(tc, "__mro") {
		cn := ""
		if n, has := cls.props["__name"]; has {
			cn = rt.toString(n)
		}
		return pyBuiltinSub(cn, tn)
	}
	if !hasKey(tc, "__mro") && tn == "object" {
		return true
	}
	for _, k := range pyMRO(cls) {
		if k == target {
			return true
		}
		if n, has := k.props["__name"]; has && tn != "" && rt.toString(n) == tn {
			return true
		}
	}
	return false
}

// pyPrintText is print()'s sep or end: None takes the default, a str is used
// as it is, and anything else is CPython's TypeError. Twin of pyPrintText in
// languages/lib/python-rt.metajs and languages/python-interpreter.abnf.
func (rt *jsrt) pyPrintText(v interface{}, name, dflt string) string {
	if v == jsUndef || v == jsNull {
		return dflt
	}
	if s, ok := v.(string); ok {
		return s
	}
	panic(&jsThrown{value: rt.pyExcInstance("TypeError",
		name+" must be None or a string, not "+pyTypeName(rt, v))})
}

// pyAddKind classes an operand of Python's `+`, which is defined only within one
// kind: 'n' a number (bool, int, float, complex), 's' a str, 'l' a list, and 0
// anything else.
func pyAddKind(v interface{}) byte {
	switch t := v.(type) {
	case float64, bool, *jsPyFlo, *jsBigInt:
		return 'n'
	case string:
		return 's'
	case *jsArray:
		return 'l'
	case *jsObject:
		if isPyComplex(t) {
			return 'n'
		}
	}
	return 0
}

// pyAddCheck raises CPython's TypeError for a `+` across kinds ("a" + 1, 1 + None).
// Twin of pyAddCheck in languages/python-interpreter.abnf and pyAAddCheck in
// languages/lib/python-rt.metajs.
func (rt *jsrt) pyAddCheck(l, r interface{}) {
	kl := pyAddKind(l)
	if kl != 0 && kl == pyAddKind(r) {
		return
	}
	rn := pyTypeName(rt, r)
	msg := "unsupported operand type(s) for +: '" + pyTypeName(rt, l) + "' and '" + rn + "'"
	switch kl {
	case 's':
		msg = "can only concatenate str (not \"" + rn + "\") to str"
	case 'l':
		msg = "can only concatenate list (not \"" + rn + "\") to list"
	}
	panic(&jsThrown{value: rt.pyExcInstance("TypeError", msg)})
}

// pyTypeName is the Python name of a builtin value's type.
func pyTypeName(rt *jsrt, v interface{}) string {
	switch t := v.(type) {
	case jsUndefT, jsNullT:
		return "NoneType"
	case bool:
		return "bool"
	case string:
		return "str"
	case *jsBigInt:
		return "int"
	case *jsPyFlo:
		return "float"
	case float64:
		// A plain number is an INT: 2.0 and 2 are the same double, so nothing
		// about the value can tell the two Python types apart - which is why the
		// float is boxed (jsPyFlo) in the first place.
		return "int"
	case *jsArray:
		return "list"
	case *jsObject:
		// type(C) is 'type' for a class object - a BUILTIN one (what type(3)
		// hands out, which carries only __name and no __mro) as well as a user
		// class.
		if _, isCls := pyClassObj(t); isCls {
			return "type"
		}
		if _, hasName := t.props["__name"]; hasName {
			if _, isInst := t.props["__class"]; !isInst {
				return "type"
			}
		}
		if _, isSet := pySetElems(t); isSet {
			return "set"
		}
		if _, _, isDict := dictParts(t); isDict {
			return "dict"
		}
		if isPyComplex(t) {
			return "complex"
		}
		// A CURSOR names its own type - CPython's list_iterator /
		// str_ascii_iterator / dict_keyiterator / map / filter / zip /
		// enumerate / reversed. type(it).__name__ reads it (js_pytype hands the
		// name to pyBuiltinClass) and so does the AttributeError pyMethodCall's
		// cursor arm raises: CPython says "'list_iterator' object has no
		// attribute 'send'", not "'object' ...".
		if n := pyIterName(rt, t); n != "" {
			return n
		}
		// A compiled regexp and a match object name their own CPython types, and
		// the name is MODULE-QUALIFIED on purpose: this is the string an
		// AttributeError interpolates, and CPython says "'re.Pattern' object has
		// no attribute 'nope'". type(p).__name__ is the SHORT 'Pattern', which is
		// why the __name__ arms below cut at the last dot (pyShortName). The
		// markers are the ones rxPyCompile and the match builder set in
		// abnf/jsrtregexpy.go.
		if _, isRe := t.props["__pyre"]; isRe {
			return "re.Pattern"
		}
		if _, isM := t.props["__pymatch"]; isM {
			return "re.Match"
		}
		// The two decorator wrappers and a module object are
		// their own CPython types. abnf/jsrtpydesc.go mints the first two.
		if k := pyDescKind(t); k == "s" {
			return "staticmethod"
		} else if k == "c" {
			return "classmethod"
		}
		if _, isMod := t.props["__pymod"]; isMod {
			return "module"
		}
		// An instance of a user class is a "'C' object" in every message.
		if _, cls, isInst := pyInstance(t); isInst {
			if n, has := cls.props["__name"]; has {
				return rt.toString(n)
			}
		}
		return "object"
	case *jsGenerator:
		// CPython's type(gen).__name__, and the name its AttributeError uses.
		return "generator"
	case *jsClosure, *hostFunc, *boundMethod:
		// A BOUND method and a BUILTIN are their own CPython types, and a class
		// object is 'type'.
		if bi := pyBoundOf(v); bi != nil {
			// An UNBOUND descriptor is its own CPython type; a module-level
			// builtin and a bound builtin method share one.
			if bi.kind == "d" {
				return "method_descriptor"
			}
			if bi.fn == nil {
				return "builtin_function_or_method"
			}
			return "method"
		}
		if c, ok := rt.pyBuiltinRender[v]; ok {
			if strings.HasPrefix(c, "*") {
				return "type" // range / enumerate / zip / map / filter / reversed.
			}
			return "builtin_function_or_method"
		}
		return "function"
	}
	if _, _, ok := dictParts(v); ok {
		return "dict"
	}
	return "object"
}

// pyShortName is the last component of a possibly module-qualified type name.
// __name__ and __qualname__ of a builtin type object are the SHORT name in
// CPython (type(re.compile("a")).__name__ is "Pattern") while the type object
// itself renders and reports as "re.Pattern". Keep in step with pyShortName in
// languages/lib/python-rt.metajs and in languages/python-interpreter.abnf.
func pyShortName(n string) string {
	for i := len(n) - 1; i >= 0; i-- {
		if n[i] == '.' {
			return n[i+1:]
		}
	}
	return n
}

// pyBuiltinClass hands out one stable class object per builtin type name, so
// type(x) is type(y) holds for two values of the same builtin type.
func (rt *jsrt) pyBuiltinClass(name string) *jsObject {
	if rt.pyTypes == nil {
		rt.pyTypes = map[string]*jsObject{}
	}
	if c, ok := rt.pyTypes[name]; ok {
		return c
	}
	c := newJSObject()
	c.set("__name", name)
	rt.pyTypes[name] = c
	return c
}

// pyContains is 'x in y' for the builtin containers (the js_pyin logic, reached
// from js_pybin once no __contains__ answered).
func (rt *jsrt) pyContains(x, y interface{}) bool {
	if els, ok := pySetElems(y); ok {
		rt.pyHashCheckMember(x)
		for _, e := range els.elems {
			if rt.pyEqual(e, x) {
				return true
			}
		}
		return false
	}
	switch c := y.(type) {
	case *jsArray:
		for _, e := range c.elems {
			if rt.pyEqual(e, x) {
				return true
			}
		}
		return false
	case string:
		return strings.Contains(c, rt.toString(x))
	}
	if keys, _, ok := dictParts(y); ok {
		return rt.pyDictFind(keys, x) >= 0
	}
	// A CURSOR or a GENERATOR is CONSUMED by `in`, up to the match - CPython's
	// rule. The scan STOPS at the match rather than draining, so
	// `it := iter(xs); 1 in it; list(it)` leaves the rest, as CPython does.
	if g, isG := y.(*jsGenerator); isG {
		for {
			st, _ := g.step(rt, jsUndef).(*jsObject)
			if st == nil {
				return false
			}
			if done, _ := st.props["done"].(bool); done {
				return false
			}
			if rt.pyEqual(st.props["value"], x) {
				return true
			}
		}
	}
	if it, isIt := y.(*jsObject); isIt && pyIterName(rt, y) != "" {
		for {
			st := rt.pyItStep(it)
			if done, _ := st.props["done"].(bool); done {
				return false
			}
			if rt.pyEqual(st.props["value"], x) {
				return true
			}
		}
	}
	rt.fail("'in' needs a list, a string, a dict or an object with __contains__")
	return false
}

// pyFloorMod is Python's %: the result takes the sign of the divisor.
func pyFloorMod(x, y float64) float64 {
	r := math.Mod(x, y)
	if r != 0 && (r < 0) != (y < 0) {
		r += y
	}
	return r
}

// pyArith is -, * and / with the bigint path js_sub/js_mul/js_div take.
func (rt *jsrt) pyArith(op byte, l, r interface{}) interface{} {
	if rt.hasBigInt {
		if v, ok := bigArith(op, l, r); ok {
			return v
		}
	}
	if pyIsFlo(l) || pyIsFlo(r) {
		switch op {
		case '-':
			return &jsPyFlo{f: rt.toNumber(l) - rt.toNumber(r)}
		case '*':
			return &jsPyFlo{f: rt.toNumber(l) * rt.toNumber(r)}
		}
		return &jsPyFlo{f: rt.toNumber(l) / rt.toNumber(r)}
	}
	switch op {
	case '-':
		return rt.toNumber(l) - rt.toNumber(r)
	case '*':
		return rt.toNumber(l) * rt.toNumber(r)
	}
	return rt.toNumber(l) / rt.toNumber(r)
}

// pyBinDunder / pyBinReflected map an operator to the method an instance on the
// left (resp. on the right) may answer it with.
var pyBinDunder = map[string]string{
	"+": "__add__", "-": "__sub__", "*": "__mul__", "/": "__truediv__",
	"//": "__floordiv__", "%": "__mod__", "@": "__matmul__", "**": "__pow__",
	"==": "__eq__", "!=": "__ne__", "<": "__lt__", ">": "__gt__",
	"<=": "__le__", ">=": "__ge__",
}
var pyBinReflected = map[string]string{
	"+": "__radd__", "-": "__rsub__", "*": "__rmul__", "/": "__rtruediv__",
	"//": "__rfloordiv__", "%": "__rmod__", "@": "__rmatmul__", "**": "__rpow__",
	"==": "__eq__", "!=": "__ne__", "<": "__gt__", ">": "__lt__",
	"<=": "__ge__", ">=": "__le__",
}

// pyDunderBin answers a binary operator through a user class's dunder, trying
// the left operand first and then the right one's reflected form. ok is false
// when neither side defines one - the caller then takes the builtin path.
func (rt *jsrt) pyDunderBin(op string, l, r interface{}) (interface{}, bool) {
	if _, cls, ok := pyInstance(l); ok {
		if m, found := pyLookup(cls, pyBinDunder[op]); found && isCallable(m) {
			if v := rt.call(m, jsUndef, []interface{}{l, r}); v != pyNotImplemented {
				return v, true
			}
		}
	}
	if _, cls, ok := pyInstance(r); ok {
		if m, found := pyLookup(cls, pyBinReflected[op]); found && isCallable(m) {
			if v := rt.call(m, jsUndef, []interface{}{r, l}); v != pyNotImplemented {
				return v, true
			}
		}
	}
	if op == "!=" { // Python's default __ne__ is the negation of __eq__.
		if v, ok := rt.pyDunderBin("==", l, r); ok {
			return !rt.truthy(v), true
		}
	}
	return nil, false
}

// ----------------------------------------------------------------------------
// Python call binding
//
// A Python call carries positional AND keyword arguments, while a compiled
// function reads a flat array by index. The def registers its signature
// (js_pysig) and pyBindCall maps one onto the other at the call site.
//
// Two prologue layouts exist, and which one a function uses is fixed at compile
// time by its own parameter list:
//
//	A  [p0 .. pn-1, extra positionals ...]   - the plain layout; a *args rest
//	   parameter reads the extras with js_pyrest(args, n).
//	B  [p0 .. pn-1, *args list, **kwargs dict]  - used as soon as the def has
//	   keyword-only parameters or a **kwargs one, because those cannot be
//	   recovered from a flat positional array.
type pySig struct {
	names   []string
	nkwonly int // The LAST nkwonly names are keyword-only.
	ext     bool
}

// pyBindCall turns (positional, keyword dict) into the callee's argument array.
func (rt *jsrt) pyBindCall(callee interface{}, pos []interface{}, kw interface{}) []interface{} {
	out, _ := rt.pyBindCallR(callee, pos, kw)
	return out
}

// pyBindCallR is pyBindCall plus the answer to "did binding actually build a
// NEW array". A caller that wants to pass its own argument handle straight
// through (js_pycall) must ask THAT question and not "did the length change":
// the extended layout is [p0..pn-1, *args, **kwargs], so a call whose
// positional count is exactly len(names)+2 produces a rebound array of the very
// same length - `def f(a, *rest, **kw)` called `f(1, 2, 3)`.
func (rt *jsrt) pyBindCallR(callee interface{}, pos []interface{}, kw interface{}) ([]interface{}, bool) {
	kwKeys, kwVals, hasKw := dictParts(kw)
	if hasKw && len(kwKeys.elems) == 0 {
		hasKw = false
	}
	var sig *pySig
	if rt.pySigs != nil {
		sig = rt.pySigs[callee]
	}
	if sig == nil {
		// A builtin (no declared signature) takes the keyword dict as one extra
		// trailing argument - which is exactly what dict(a=1) / dict(**m) want.
		if hasKw {
			return append(append([]interface{}{}, pos...), kw), true
		}
		return pos, false
	}
	if !sig.ext && !hasKw {
		return pos, false
	}
	n := len(sig.names)
	npos := n - sig.nkwonly
	out := make([]interface{}, n)
	for i := range out {
		out[i] = jsUndef
	}
	for i := 0; i < npos && i < len(pos); i++ {
		out[i] = pos[i]
	}
	used := map[string]bool{}
	fromKw := func(i int) {
		if !hasKw {
			return
		}
		for j, k := range kwKeys.elems {
			if rt.toString(k) == sig.names[i] {
				out[i] = kwVals.elems[j]
				used[sig.names[i]] = true
			}
		}
	}
	for i := 0; i < npos; i++ {
		if i >= len(pos) {
			fromKw(i)
		}
	}
	for i := npos; i < n; i++ {
		fromKw(i)
	}
	var extra []interface{}
	if len(pos) > npos {
		extra = pos[npos:]
	}
	if !sig.ext {
		return append(out, extra...), true
	}
	rest := &jsArray{elems: append([]interface{}{}, extra...)}
	lk, lv := &jsArray{}, &jsArray{}
	if hasKw {
		for j, k := range kwKeys.elems {
			if !used[rt.toString(k)] {
				dictAppend(lk, lv, k, kwVals.elems[j])
			}
		}
	}
	left := &jsObject{props: map[string]interface{}{"__dict": true, "keys": lk, "vals": lv}}
	return append(out, rest, left), true
}

// pyMinMax is max/min(iterable, *, key=None, default=...) and max/min(a, b,
// *args, key=None), bound as [positionals, keywords]. want is +1 for max, -1
// for min; the first of equal extremes wins.
func (rt *jsrt) pyMinMax(argsV interface{}, want int) uint64 {
	el := argsV.(*jsArray).elems
	elems := el[0].(*jsArray).elems
	kw := el[1]
	fname, op := "max", ">"
	if want < 0 {
		fname, op = "min", "<"
	}
	rt.pyKwOnly(kw, fname, "key", "default")
	key := rt.pyKwArg(kw, "key")
	dflt, hasDflt := rt.pyKwArg(kw, "default"), rt.pyKwHas(kw, "default")
	switch {
	case len(elems) == 0:
		panic(&jsThrown{value: rt.pyExcInstance("TypeError", fname+" expected at least 1 argument, got 0")})
	case len(elems) == 1: // Any iterable: max("abc"), max({1, 2}), max(map(f, xs)).
		elems = rt.pyElemsOf(elems[0])
	case hasDflt:
		panic(&jsThrown{value: rt.pyExcInstance("TypeError",
			"Cannot specify a default for "+fname+"() with multiple positional arguments")})
	}
	if len(elems) == 0 {
		if hasDflt {
			return rt.wrap(dflt)
		}
		panic(&jsThrown{value: rt.pyExcInstance("ValueError", fname+"() iterable argument is empty")})
	}
	keyOf := func(e interface{}) interface{} {
		if isUndefOrNull(key) {
			return e
		}
		return rt.callPyValue(key, []interface{}{e}, jsUndef)
	}
	best := elems[0]
	bestK := keyOf(best)
	for _, e := range elems[1:] {
		if k := keyOf(e); rt.pyOrder(op, k, bestK) {
			best, bestK = e, k
		}
	}
	return rt.wrap(best)
}

// pyKwPos is a builtin's arguments as positionals: the keyword arguments of
// names are moved into their slots (the first nposonly are positional-only),
// an absent slot stays undefined. CPython's TypeError for an unknown keyword or
// one given twice. Twin of pyKwPos in python-interpreter.abnf and
// lib/python-rt.metajs.
func (rt *jsrt) pyKwPos(fname string, pos []interface{}, kw interface{}, names []string, nposonly int) []interface{} {
	keys, vals, _ := dictParts(kw)
	if given := len(pos) + len(keys.elems); given > len(names) {
		panic(&jsThrown{value: rt.pyExcInstance("TypeError",
			fmt.Sprintf("%s() takes at most %d arguments (%d given)", fname, len(names), given))})
	}
	out := append([]interface{}{}, pos...)
	for k, kv := range keys.elems {
		kn := rt.toString(kv)
		at := -1
		for j := nposonly; j < len(names); j++ {
			if names[j] == kn {
				at = j
			}
		}
		if at < 0 {
			panic(&jsThrown{value: rt.pyExcInstance("TypeError",
				fname+"() got an unexpected keyword argument '"+kn+"'")})
		}
		if at < len(pos) {
			panic(&jsThrown{value: rt.pyExcInstance("TypeError",
				fmt.Sprintf("argument for %s() given by name ('%s') and position (%d)", fname, kn, at+1))})
		}
		for len(out) <= at {
			out = append(out, jsUndef)
		}
		out[at] = vals.elems[k]
	}
	return out
}

// pyIntNew is int(x=0, /, base=10). With a base, x must be a str and is parsed
// in that base; without one, a str is parsed in base 10 and a number TRUNCATES
// (a bigint stays exact).
func (rt *jsrt) pyIntNew(args []interface{}) interface{} {
	if len(args) > 2 {
		panic(&jsThrown{value: rt.pyExcInstance("TypeError",
			fmt.Sprintf("int expected at most 2 arguments, got %d", len(args)))})
	}
	var v interface{} = jsUndef
	if len(args) > 0 {
		v = pyUnwrapBuiltin(args[0])
	}
	if len(args) < 2 || args[1] == jsUndef {
		switch x := v.(type) {
		case jsUndefT:
			return 0.0
		case string:
			return rt.pyParseInt(x, 10)
		case *jsBigInt:
			return x
		case *jsPyFlo:
			return rt.pyFloatToInt(x.f)
		}
		return math.Trunc(rt.toNumber(v))
	}
	if v == jsUndef {
		panic(&jsThrown{value: rt.pyExcInstance("TypeError", "int() missing string argument")})
	}
	var base float64
	switch b := args[1].(type) {
	case float64:
		base = b
	case bool:
		if b {
			base = 1
		}
	case *jsBigInt:
		base = 37
	default:
		panic(&jsThrown{value: rt.pyExcInstance("TypeError",
			"'"+pyTypeName(rt, b)+"' object cannot be interpreted as an integer")})
	}
	if (base != 0 && base < 2) || base > 36 {
		panic(&jsThrown{value: rt.pyExcInstance("ValueError", "int() base must be >= 2 and <= 36, or 0")})
	}
	s, ok := v.(string)
	if !ok {
		panic(&jsThrown{value: rt.pyExcInstance("TypeError", "int() can't convert non-string with explicit base")})
	}
	return rt.pyParseInt(s, int(base))
}

// pyParseInt is CPython's int-from-str: surrounding whitespace, a sign, a
// 0x/0o/0b prefix (required to match the base, or choosing it under base 0),
// single underscores between digits or right after the prefix, and under base
// 0 no leading zero on a non-zero decimal.
func (rt *jsrt) pyParseInt(s string, base int) interface{} {
	t := strings.TrimSpace(pyDecAscii(s))
	neg := false
	if t != "" && (t[0] == '+' || t[0] == '-') {
		neg, t = t[0] == '-', t[1:]
	}
	b, prefixed := base, false
	if len(t) >= 2 && t[0] == '0' {
		pb := map[byte]int{'x': 16, 'X': 16, 'o': 8, 'O': 8, 'b': 2, 'B': 2}[t[1]]
		if pb != 0 && (b == 0 || b == pb) {
			b, prefixed, t = pb, true, t[2:]
		}
	}
	ok := t != ""
	if b == 0 {
		b = 10
		if ok && t[0] == '0' && strings.Trim(t, "0_") != "" {
			ok = false
		}
	}
	acc := new(big.Int)
	prevDigit := prefixed
	for i := 0; i < len(t) && ok; i++ {
		ch := t[i]
		if ch == '_' {
			ok = prevDigit
			prevDigit = false
			continue
		}
		if ch >= 'A' && ch <= 'Z' {
			ch += 'a' - 'A'
		}
		d := strings.IndexByte("0123456789abcdefghijklmnopqrstuvwxyz", ch)
		if d < 0 || d >= b {
			ok = false
			break
		}
		acc.Mul(acc, big.NewInt(int64(b)))
		acc.Add(acc, big.NewInt(int64(d)))
		prevDigit = true
	}
	if !ok || !prevDigit {
		panic(&jsThrown{value: rt.pyExcInstance("ValueError",
			fmt.Sprintf("invalid literal for int() with base %d: %s", base, pyStrRepr(s)))})
	}
	if neg {
		acc.Neg(acc)
	}
	return rt.pyBigNarrow(acc)
}

// pyUnpackCheck raises CPython's ValueError for an unpacking of got values into
// n targets (n plain ones beside a *rest when star). A too-many message names
// the count only for a list, a tuple or a dict, as CPython does. Twin of
// pyUnpackCheck in python-interpreter.abnf and lib/python-rt.metajs.
func (rt *jsrt) pyUnpackCheck(v interface{}, got, n int, star bool) {
	msg := ""
	switch {
	case star && got < n:
		msg = fmt.Sprintf("not enough values to unpack (expected at least %d, got %d)", n, got)
	case star:
	case got < n:
		msg = fmt.Sprintf("not enough values to unpack (expected %d, got %d)", n, got)
	case got > n:
		_, isArr := v.(*jsArray)
		_, isSet := pySetElems(v)
		_, _, isDict := dictParts(v)
		msg = fmt.Sprintf("too many values to unpack (expected %d)", n)
		if isArr || (isDict && !isSet) {
			msg = fmt.Sprintf("too many values to unpack (expected %d, got %d)", n, got)
		}
	}
	if msg != "" {
		panic(&jsThrown{value: rt.pyExcInstance("ValueError", msg)})
	}
}

// pySum folds with Python's own `+` (js_pybin), so a float, an arbitrary
// precision int, a complex or a list start all add as they do in CPython, and
// "a" + 1 raises - except over floats: CPython 3.12+ keeps a run of floats (and
// ints) in a C double with Neumaier's compensated summation, entered from an
// int start at the first float or from a float start, and left for good at the
// first item of any other type. sum([0.1] * 10) is 1.0. mode is 0 for the int
// run, 1 for the float run, 2 for plain `+`.
func (rt *jsrt) pySum(iter, start interface{}) interface{} {
	var t interface{} = 0.0
	if start != jsUndef {
		t = start
	}
	mode := 2
	var f, c float64
	switch tv := t.(type) {
	case float64:
		mode = 0
	case *jsPyFlo:
		mode, f = 1, tv.f
	}
	for _, e := range rt.pyElemsOf(iter) {
		_, isNum := e.(float64)
		_, isBool := e.(bool)
		isInt := isNum || isBool
		if mode == 1 {
			if x, ok := e.(*jsPyFlo); ok {
				s := f + x.f
				if math.Abs(f) >= math.Abs(x.f) {
					c += (f - s) + x.f
				} else {
					c += (x.f - s) + f
				}
				f = s
				continue
			}
			if isInt {
				f += rt.toNumber(e)
				continue
			}
			t, mode = &jsPyFlo{f: pySumEnd(f, c)}, 2
		}
		t = rt.pyExt("js_pybin", "+", t, e)
		if _, stillInt := t.(float64); mode == 0 && (!isInt || !stillInt) {
			mode = 2
			if fl, ok := t.(*jsPyFlo); ok && !isInt {
				mode, f, c = 1, fl.f, 0
			}
		}
	}
	if mode == 1 {
		return &jsPyFlo{f: pySumEnd(f, c)}
	}
	return t
}

// pySumEnd ends sum()'s float run: the compensation is added back unless it is
// zero (which keeps a -0.0 total) or not finite (which would turn an infinite
// total into a NaN).
func pySumEnd(f, c float64) float64 {
	if c != 0 && !math.IsInf(c, 0) && !math.IsNaN(c) {
		return f + c
	}
	return f
}

// pyOrder answers Python's `l op r` for an order operator: js_pybin's rules, so
// a list compares element-wise and a user class answers with its dunder. Two
// plain numbers or two strings take the comparison js_pybin would reach anyway.
func (rt *jsrt) pyOrder(op string, l, r interface{}) bool {
	_, ln := l.(float64)
	_, rn := r.(float64)
	_, ls := l.(string)
	_, rs := r.(string)
	if (ln && rn) || (ls && rs) {
		c := rt.jsCompare(l, r)
		switch op {
		case "<":
			return c == -1
		case ">":
			return c == 1
		}
	}
	return rt.pyTruthyOf(rt.pyExt("js_pybin", op, l, r))
}

// pySeqOrder is an order comparison of two lists, CPython's: the first pair of
// elements that are not equal decides, with op itself; otherwise the lengths do.
func (rt *jsrt) pySeqOrder(op string, l, r []interface{}) interface{} {
	for i := 0; i < len(l) && i < len(r); i++ {
		if !rt.pyEqual(l[i], r[i]) {
			return rt.pyExt("js_pybin", op, l[i], r[i])
		}
	}
	switch op {
	case "<":
		return len(l) < len(r)
	case ">":
		return len(l) > len(r)
	case "<=":
		return len(l) <= len(r)
	}
	return len(l) >= len(r)
}

// pyDeletedT is the value `del x` leaves in the scope slot; scopeGet turns a
// read of it into UnboundLocalError. It is a type of its own so nothing a
// program can produce ever compares equal to it.
type pyDeletedT struct{}

var pyDeleted interface{} = pyDeletedT{}

// pyExcInstance builds a builtin exception instance - the {__class:{__name},
// args} shape the except clauses match - for a runtime error the program is
// meant to be able to catch.
// pyRootExcType reports whether the except clause's type is the root of the
// exception hierarchy - the only one that catches a raised non-instance value.
// A tuple of types (written as a list here) counts when any element is a root.
func pyRootExcType(target interface{}) bool {
	if arr, ok := target.(*jsArray); ok {
		for _, e := range arr.elems {
			if pyRootExcType(e) {
				return true
			}
		}
		return false
	}
	o, ok := target.(*jsObject)
	if !ok {
		return false
	}
	n, has := o.props["__name"]
	if !has {
		return false
	}
	name, _ := n.(string)
	return name == "Exception" || name == "BaseException"
}

// pyExcGroupLeaves flattens an ExceptionGroup instance into its leaf exceptions and
// answers its message alongside. ok is false for anything that is not an instance of
// a class called ExceptionGroup, which is what makes 'except*' fall through to a
// re-raise on an ordinary exception. The nested exceptions live in "exceptions" when
// the group was split, and otherwise in args[1] - where ExceptionGroup(msg, [errs])
// puts them through the inherited Exception.__init__.
func (rt *jsrt) pyExcGroupLeaves(v interface{}) (leaves []interface{}, msg interface{}, ok bool) {
	o, cls, isInst := pyInstance(v)
	if !isInst {
		return nil, nil, false
	}
	isGroup := false
	for _, k := range pyMRO(cls) {
		if n, has := k.props["__name"]; has && rt.toString(n) == "ExceptionGroup" {
			isGroup = true
			break
		}
	}
	if !isGroup {
		return nil, nil, false
	}
	if ar, isArr := o.props["args"].(*jsArray); isArr && len(ar.elems) > 0 {
		msg = ar.elems[0]
	}
	src, has := o.props["exceptions"]
	if !has {
		if ar, isArr := o.props["args"].(*jsArray); isArr && len(ar.elems) > 1 {
			src = ar.elems[1]
		}
	}
	arr, isArr := src.(*jsArray)
	if !isArr {
		return nil, msg, true
	}
	for _, e := range arr.elems {
		if sub, _, isSub := rt.pyExcGroupLeaves(e); isSub {
			leaves = append(leaves, sub...)
		} else {
			leaves = append(leaves, e)
		}
	}
	return leaves, msg, true
}

// pyExcInstanceV is pyExcInstance with the argument kept as a VALUE rather than
// as text. It exists for KeyError, whose argument is the KEY: CPython's
// KeyError.__str__ is repr(args[0]), so repr(e) is KeyError(3) and
// e.args[0] is the key itself. The repr happens in pyUserRender.
func (rt *jsrt) pyExcInstanceV(name string, arg interface{}) interface{} {
	inst := newJSObject()
	inst.set("__class", rt.pyExcClass(name))
	inst.set("args", &jsArray{elems: []interface{}{arg}})
	return inst
}

// pyExcParentName is CPython's hierarchy above a RUNTIME-raised exception:
// rt.pyIsInstance walks the __super chain by name, so a KeyError must derive
// from LookupError for `except LookupError` to catch it. pyExcParent in
// languages/python-interpreter.abnf and the excParent block of
// languages/python-to-llvm-ir.abnf are the same table.
func pyExcParentName(name string) string {
	switch name {
	case "ZeroDivisionError":
		return "ArithmeticError"
	case "IndexError", "KeyError":
		return "LookupError"
	case "UnboundLocalError":
		return "NameError"
	case "NotImplementedError":
		return "RuntimeError"
	case "OverflowError":
		return "ArithmeticError"
	case "UnicodeError":
		return "ValueError"
	case "UnicodeDecodeError", "UnicodeEncodeError":
		return "UnicodeError"
	case "Exception":
		return "BaseException"
	}
	return "Exception"
}

func (rt *jsrt) pyExcClass(name string) *jsObject {
	cls := rt.pyBuiltinClass(name)
	if name != "BaseException" {
		if _, has := cls.props["__super"]; !has {
			cls.set("__super", rt.pyExcClass(pyExcParentName(name)))
		}
	}
	return cls
}

func (rt *jsrt) pyExcInstance(name string, msg string) interface{} {
	// The class the runtime raises with is a stable per-name object that derives
	// from a class called Exception, so pyIsInstance's name walk answers
	// `except Exception` (and `except <ThatName>`) for a StopIteration /
	// UnboundLocalError / AttributeError the RUNTIME raised, exactly like for one
	// the program raised through the module's own class objects.
	inst := newJSObject()
	inst.set("__class", rt.pyExcClass(name))
	inst.set("args", &jsArray{elems: []interface{}{msg}})
	return inst
}

// ----- Python's format mini-language (js_pyformat): see pyFormat -----
func pyIsAlign(ch byte) bool { return ch == '<' || ch == '>' || ch == '^' || ch == '=' }

// ----- "fmt" % args, CPython's printf-style formatting -----
//
// Supported: the conversions s r a d i u x X o c e E g G f F and %%; the flags
// - + 0 and space, and # for x X o; a width; and a .precision, which TRUNCATES
// for s/r/a and gives the minimum digit count for the int conversions. A
// %(name)s mapping key reads a dict argument. NOT supported, and a LOUD failure
// rather than a silent wrong answer: the # alternate form of the float
// conversions. Every operand and arity error is CPython's catchable exception
// with CPython's text.
//
// THE ARGUMENT TUPLE. CPython takes one value per conversion from a TUPLE and
// treats every other object as a single value - but a tuple IS a list in this
// value model, so an ARRAY is read as the argument tuple exactly when its length
// equals the conversion count, and as one value otherwise. That is CPython's
// answer in every case except a ONE-element list meeting a single conversion,
// where the two readings are genuinely indistinguishable here.
//
// pyPct in languages/python-interpreter.abnf and languages/lib/python-rt.metajs
// are the same body; those two spell the float verbs out over floPrec because
// their dialect has no strconv.
func pyPctCount(f string) int {
	n, i := 0, 0
	for i < len(f) {
		if f[i] == '%' {
			if i+1 < len(f) && f[i+1] == '%' {
				i += 2
				continue
			}
			n++
		}
		i++
	}
	return n
}

func (rt *jsrt) pyPctTypeErr(msg string) {
	panic(&jsThrown{value: rt.pyExcInstance("TypeError", msg)})
}

// pyPctInt is the operand of an int conversion (d i u x X o) as an exact int: any
// real number for d i u, truncated (exactly past 2^53, raising for an infinity or
// a NaN), and an int only for x X o. Twin of pyPctInt in
// languages/python-interpreter.abnf and languages/lib/python-rt.metajs.
func (rt *jsrt) pyPctInt(v interface{}, typ byte) *big.Int {
	if fl, ok := v.(*jsPyFlo); ok && (typ == 'd' || typ == 'i' || typ == 'u') {
		v = rt.pyFloatToInt(fl.f)
	}
	if _, _, isInst := pyInstance(v); !isInst {
		if b, ok := pyBigOperand(v); ok {
			return b
		}
	}
	tn := pyTypeName(rt, v)
	if isPyComplex(v) {
		tn = "complex"
	}
	if typ == 'd' || typ == 'i' || typ == 'u' {
		rt.pyPctTypeErr("%" + string(typ) + " format: a real number is required, not " + tn)
	}
	rt.pyPctTypeErr("%" + string(typ) + " format: an integer is required, not " + tn)
	return nil
}

// pyPctReal is the operand of a float conversion (e f g) as a double: a float,
// or an int converted to its nearest double. Twin of pyPctReal in
// languages/python-interpreter.abnf and languages/lib/python-rt.metajs.
func (rt *jsrt) pyPctReal(v interface{}) float64 {
	switch t := v.(type) {
	case *jsPyFlo:
		return t.f
	case *jsBigInt:
		return rt.pyBigToF(t.v)
	case float64:
		return t
	case bool:
		return pyBoolInt(t)
	}
	tn := pyTypeName(rt, v)
	if isPyComplex(v) {
		tn = "complex"
	}
	rt.pyPctTypeErr("must be real number, not " + tn)
	return 0
}

// pyPctChar is %c: a one-character str, or an int naming a code point.
func (rt *jsrt) pyPctChar(v interface{}) string {
	if s, isStr := v.(string); isStr {
		if n := utf8.RuneCountInString(s); n != 1 {
			rt.pyPctTypeErr("%c requires an int or a unicode character, not a string of length " + strconv.Itoa(n))
		}
		return s
	}
	if _, _, isInst := pyInstance(v); !isInst {
		if b, ok := pyBigOperand(v); ok {
			if b.Sign() < 0 || b.Cmp(big.NewInt(0x110000)) >= 0 {
				panic(&jsThrown{value: rt.pyExcInstance("OverflowError", "%c arg not in range(0x110000)")})
			}
			return string(rune(b.Int64()))
		}
	}
	tn := pyTypeName(rt, v)
	if isPyComplex(v) {
		tn = "complex"
	}
	rt.pyPctTypeErr("%c requires an int or a unicode character, not " + tn)
	return ""
}

// pyPctIntText is an int conversion's text: the sign, then the # prefix, then the
// digits zero-padded to the precision ("%#.4x" % -255 is -0x00ff).
func pyPctIntText(b *big.Int, base int, prefix string, prec int, upper bool) string {
	digits := new(big.Int).Abs(b).Text(base)
	for len(digits) < prec {
		digits = "0" + digits
	}
	out := prefix + digits
	if upper {
		out = strings.ToUpper(out)
	}
	if b.Sign() < 0 {
		return "-" + out
	}
	return out
}

func (rt *jsrt) pyPctOne(v interface{}, typ byte, left, zero, plus, space, alt bool, width, prec int) string {
	var body string
	numeric := false
	switch typ {
	case 's', 'r', 'a':
		if typ == 's' {
			body = rt.pyString(v)
		} else {
			body = rt.pyRepr(v)
		}
		if prec >= 0 {
			rs := []rune(body)
			if len(rs) > prec {
				body = string(rs[:prec])
			}
		}
	case 'c':
		body = rt.pyPctChar(v)
	case 'd', 'i', 'u':
		numeric = true
		body = pyPctIntText(rt.pyPctInt(v, typ), 10, "", prec, false)
	case 'x', 'X':
		prefix := ""
		if alt {
			prefix = "0x"
		}
		numeric, body = true, pyPctIntText(rt.pyPctInt(v, typ), 16, prefix, prec, typ == 'X')
	case 'o':
		prefix := ""
		if alt {
			prefix = "0o"
		}
		numeric, body = true, pyPctIntText(rt.pyPctInt(v, typ), 8, prefix, prec, false)
	case 'f', 'F':
		if alt {
			rt.fail("unsupported format flag '#'")
		}
		p := prec
		if p < 0 {
			p = 6
		}
		n := rt.pyPctReal(v)
		if math.IsNaN(n) || math.IsInf(n, 0) {
			numeric, body = true, pyPctExp(n, typ, prec) // inf / nan, the same text.
		} else {
			numeric, body = true, strconv.FormatFloat(n, 'f', p, 64)
		}
	case 'e', 'E', 'g', 'G':
		if alt {
			rt.fail("unsupported format flag '#'")
		}
		numeric, body = true, pyPctExp(rt.pyPctReal(v), typ, prec)
	}
	if numeric && !strings.HasPrefix(body, "-") {
		if plus {
			body = "+" + body
		} else if space {
			body = " " + body
		}
	}
	pad := width - len([]rune(body))
	if pad <= 0 {
		return body
	}
	if left {
		return body + strings.Repeat(" ", pad)
	}
	if zero && numeric {
		if body[0] == '-' || body[0] == '+' || body[0] == ' ' {
			return body[:1] + strings.Repeat("0", pad) + body[1:]
		}
		return strings.Repeat("0", pad) + body
	}
	return strings.Repeat(" ", pad) + body
}

// pyPctExp is %e / %g, and the inf / nan text of %f. strconv's 'e' and 'g' ARE
// C's shapes - an exponent that is signed and at least two digits, and 'g'
// choosing between the two on the exponent that was used - so this half needs no
// digit machinery of its own; the two script halves build the same text out of
// floPrec.
func pyPctExp(n float64, typ byte, prec int) string {
	upper := typ == 'E' || typ == 'G' || typ == 'F'
	p := prec
	if p < 0 {
		p = 6
	}
	if math.IsNaN(n) {
		if upper {
			return "NAN"
		}
		return "nan"
	}
	if math.IsInf(n, 0) {
		s := "inf"
		if upper {
			s = "INF"
		}
		if n < 0 {
			return "-" + s
		}
		return s
	}
	if typ == 'g' || typ == 'G' {
		if p == 0 {
			p = 1
		}
		s := strconv.FormatFloat(n, 'g', p, 64)
		if upper {
			s = strings.ToUpper(s)
		}
		return s
	}
	s := strconv.FormatFloat(n, 'e', p, 64)
	if upper {
		s = strings.ToUpper(s)
	}
	return s
}

func (rt *jsrt) pyPct(f string, arg interface{}) string {
	nconv := pyPctCount(f)
	var args []interface{}
	var dictK, dictV *jsArray
	if arr, ok := arg.(*jsArray); ok && len(arr.elems) == nconv {
		args = arr.elems
	} else if k, v, ok := dictParts(arg); ok {
		if _, isSet := pySetElems(arg); !isSet {
			dictK, dictV = k, v
		}
		args = []interface{}{arg}
	} else {
		args = []interface{}{arg}
	}
	out := ""
	ai := 0
	mapped := false
	i := 0
	for i < len(f) {
		if f[i] != '%' {
			j := i
			for j < len(f) && f[j] != '%' {
				j++
			}
			out += f[i:j]
			i = j
			continue
		}
		i++
		if i >= len(f) {
			rt.pyPctValueErr("incomplete format")
		}
		if f[i] == '%' {
			out += "%"
			i++
			continue
		}
		key := ""
		hasKey := false
		if f[i] == '(' {
			i++
			st := i
			for i < len(f) && f[i] != ')' {
				i++
			}
			key, hasKey, mapped = f[st:i], true, true
			i++
		}
		left, zero, plus, space, alt := false, false, false, false, false
		scanning := true
		for scanning && i < len(f) {
			switch f[i] {
			case '-':
				left = true
				i++
			case '0':
				zero = true
				i++
			case '+':
				plus = true
				i++
			case ' ':
				space = true
				i++
			case '#':
				alt = true
				i++
			default:
				scanning = false
			}
		}
		width := 0
		for i < len(f) && f[i] >= '0' && f[i] <= '9' {
			width = width*10 + int(f[i]-'0')
			i++
		}
		prec := -1
		if i < len(f) && f[i] == '.' {
			i++
			prec = 0
			for i < len(f) && f[i] >= '0' && f[i] <= '9' {
				prec = prec*10 + int(f[i]-'0')
				i++
			}
		}
		for i < len(f) && (f[i] == 'l' || f[i] == 'h' || f[i] == 'L') {
			i++
		}
		if i >= len(f) {
			rt.pyPctValueErr("incomplete format")
		}
		typ := f[i]
		if !strings.ContainsRune("sraiduxXofFeEgGc", rune(typ)) {
			rt.pyPctValueErr(fmt.Sprintf("unsupported format character '%c' (0x%x) at index %d", typ, typ, i))
		}
		i++
		var v interface{}
		if hasKey {
			if dictK == nil {
				rt.pyPctTypeErr("format requires a mapping")
			}
			at := rt.pyDictFind(dictK, key)
			if at < 0 {
				panic(&jsThrown{value: rt.pyExcInstanceV("KeyError", key)})
			}
			v = dictV.elems[at]
		} else {
			if ai >= len(args) {
				rt.pyPctTypeErr("not enough arguments for format string")
			}
			v = args[ai]
			ai++
		}
		out += rt.pyPctOne(v, typ, left, zero, plus, space, alt, width, prec)
	}
	if !mapped && ai < len(args) {
		rt.pyPctTypeErr("not all arguments converted during string formatting")
	}
	return out
}

func (rt *jsrt) pyPctValueErr(msg string) {
	panic(&jsThrown{value: rt.pyExcInstance("ValueError", msg)})
}

// pyFmtCplx is complex.__format__: each part goes through the FLOAT formatter.
// No type is repr's shape (parenthesized, or the imaginary part alone when the
// real one is +0.0) - with a precision it is 'g' - and 'n' is 'g'. The
// imaginary part always shows its sign once a real part precedes it. Twin of
// pyFmtCplx in languages/python-interpreter.abnf and
// languages/lib/python-rt.metajs.
func (rt *jsrt) pyFmtCplx(re, im float64, typ string, prec int, sign rune, zflag, alt bool, group string) string {
	bare := typ == "" && prec < 0
	skipRe := typ == "" && re == 0 && !math.Signbit(re)
	ptyp := typ
	if typ == "" || typ == "n" {
		ptyp = "g"
	}
	tail := ""
	if zflag {
		tail += "z"
	}
	if alt && !bare {
		tail += "#"
	}
	tail += group
	if prec >= 0 {
		tail += "." + strconv.Itoa(prec)
	}
	if !bare {
		tail += ptyp
	}
	sg := ""
	if sign != 0 {
		sg = string(sign)
	}
	// One part. With no type and no precision it is repr's text: the float
	// form minus its forced ".0", and '#' puts the point back.
	part := func(f float64, spec string) string {
		t := rt.pyFormat(&jsPyFlo{f: f}, spec, "")
		if !bare {
			return t
		}
		t = strings.TrimSuffix(t, ".0")
		if alt && !strings.ContainsAny(t, ".n") {
			if ep := strings.IndexByte(t, 'e'); ep >= 0 {
				t = t[:ep] + "." + t[ep:]
			} else {
				t += "."
			}
		}
		return t
	}
	imSign := "+"
	if skipRe {
		imSign = sg
	}
	ims := part(im, imSign+tail) + "j"
	if skipRe {
		return ims
	}
	res := part(re, sg+tail)
	if typ == "" {
		return "(" + res + ims + ")"
	}
	return res + ims
}

// pyFormat is the format mini-language behind format(), str.format and the
// f-string: [[fill]align][sign][z][#][0][width][grouping][.precision][type], by
// CPython's rules per type. A str takes s; an int (a bool too) takes b c d o x X
// n, or a float type after becoming its nearest double; a float takes e E f F g
// G n % or none - repr() without a precision, and with one a 'g' that keeps a
// fractional digit and turns exponential one digit sooner. Any other value takes
// only the empty spec, and a complex e E f F g G n or none (pyFmtCplx). A number
// right-aligns and a str left-aligns by default; the 0 flag is a '0' fill with
// '=' alignment for a number, and with a grouping it groups the padding too.
// Widths count code points. The errors are CPython's, with its text. Twin of
// pyFormat in languages/python-interpreter.abnf and js_pyformat in
// languages/lib/python-rt.metajs.
func (rt *jsrt) pyFormat(v interface{}, spec, conv string) string {
	x := v
	switch conv {
	case "r":
		x = rt.pyRepr(v)
	case "s":
		x = rt.pyString(v)
	case "a":
		x = pyAsciiOf(rt.pyRepr(v))
	}
	if spec == "" {
		if xs, ok := x.(string); ok {
			return xs
		}
		return rt.pyString(x)
	}
	tn := pyTypeName(rt, x)
	if isPyComplex(x) {
		tn = "complex"
	}
	fail := func(kind, msg string) {
		panic(&jsThrown{value: rt.pyExcInstance(kind, msg)})
	}
	sp := []rune(spec)
	isAlign := func(r rune) bool { return r < 128 && pyIsAlign(byte(r)) }
	isDigit := func(i int) bool { return i < len(sp) && sp[i] >= '0' && sp[i] <= '9' }
	i := 0
	fill, fillGiven, align := " ", false, rune(0)
	if len(sp) > 1 && isAlign(sp[1]) {
		fill, fillGiven, align, i = string(sp[0]), true, sp[1], 2
	} else if isAlign(sp[0]) {
		align, i = sp[0], 1
	}
	sign := rune(0)
	if i < len(sp) && (sp[i] == '+' || sp[i] == '-' || sp[i] == ' ') {
		sign, i = sp[i], i+1
	}
	zflag, alt, zero := false, false, false
	if i < len(sp) && sp[i] == 'z' {
		zflag, i = true, i+1
	}
	if i < len(sp) && sp[i] == '#' {
		alt, i = true, i+1
	}
	if i < len(sp) && sp[i] == '0' {
		zero, i = true, i+1
	}
	width := 0
	for isDigit(i) {
		width, i = width*10+int(sp[i]-'0'), i+1
	}
	group := ""
	if i < len(sp) && (sp[i] == ',' || sp[i] == '_') {
		group, i = string(sp[i]), i+1
	}
	prec := -1
	if i < len(sp) && sp[i] == '.' {
		i++
		if !isDigit(i) {
			fail("ValueError", "Format specifier missing precision")
		}
		prec = 0
		for isDigit(i) {
			prec, i = prec*10+int(sp[i]-'0'), i+1
		}
	}
	typ := ""
	if i < len(sp) {
		typ, i = string(sp[i]), i+1
	}
	if i < len(sp) {
		fail("ValueError", "Invalid format specifier '"+spec+"' for object of type '"+tn+"'")
	}
	if zero && !fillGiven {
		fill = "0"
	}
	unknown := "Unknown format code '" + typ + "' for object of type '" + tn + "'"
	if xs, ok := x.(string); ok {
		switch {
		case typ != "" && typ != "s":
			fail("ValueError", unknown)
		case sign != 0:
			fail("ValueError", "Sign not allowed in string format specifier")
		case alt:
			fail("ValueError", "Alternate form (#) not allowed in string format specifier")
		case align == '=':
			fail("ValueError", "'=' alignment not allowed in string format specifier")
		case group != "":
			fail("ValueError", "Cannot specify '"+group+"' with 's'.")
		}
		if rs := []rune(xs); prec >= 0 && len(rs) > prec {
			xs = string(rs[:prec])
		}
		if align == 0 {
			align = '<'
		}
		return rt.pyFmtPad(fill, align, width, "", xs)
	}
	if re, im, ok := pyComplexParts(x); ok {
		switch {
		case typ != "" && !strings.Contains("eEfFgGn", typ):
			fail("ValueError", unknown)
		case fill == "0":
			fail("ValueError", "Zero padding is not allowed in complex format specifier")
		case align == '=':
			fail("ValueError", "'=' alignment flag is not allowed in complex format specifier")
		}
		if align == 0 {
			align = '>'
		}
		return rt.pyFmtPad(fill, align, width, "", rt.pyFmtCplx(re, im, typ, prec, sign, zflag, alt, group))
	}
	bi, isInt := pyBigOperand(x)
	if _, _, isInst := pyInstance(x); isInst {
		isInt = false
	}
	fl, isFlo := x.(*jsPyFlo)
	if !isInt && !isFlo {
		fail("TypeError", "unsupported format string passed to "+tn+".__format__")
	}
	if zero && align == 0 {
		align = '='
	}
	if align == 0 {
		align = '>'
	}
	var f float64
	if isInt {
		if typ == "" || strings.Contains("bcdoxXn", typ) {
			if prec >= 0 {
				fail("ValueError", "Precision not allowed in integer format specifier")
			}
			if typ == "c" {
				switch {
				case sign != 0:
					fail("ValueError", "Sign not allowed with integer format specifier 'c'")
				case alt:
					fail("ValueError", "Alternate form (#) not allowed with integer format specifier 'c'")
				case group != "":
					fail("ValueError", "Cannot specify '"+group+"' with 'c'.")
				case !bi.IsInt64():
					fail("OverflowError", "Python int too large to convert to C long")
				case bi.Sign() < 0 || bi.Int64() >= 0x110000:
					fail("OverflowError", "%c arg not in range(0x110000)")
				}
				return rt.pyFmtPad(fill, align, width, "", string(rune(bi.Int64())))
			}
			if group != "" && (typ == "n" || (group == "," && typ != "" && typ != "d")) {
				fail("ValueError", "Cannot specify '"+group+"' with '"+typ+"'.")
			}
			base := 10
			switch typ {
			case "b":
				base = 2
			case "o":
				base = 8
			case "x", "X":
				base = 16
			}
			digits := new(big.Int).Abs(bi).Text(base)
			if typ == "X" {
				digits = strings.ToUpper(digits)
			}
			prefix := ""
			if alt && base != 10 {
				prefix = "0" + typ
			}
			n := 3
			if base != 10 {
				n = 4
			}
			return rt.pyFmtNum(bi.Sign() < 0, sign, prefix, digits, "", group, n, fill, align, width, zero)
		}
		if !strings.Contains("eEfFgG%", typ) {
			fail("ValueError", unknown)
		}
		f = rt.pyBigToF(bi)
	} else {
		if typ != "" && !strings.Contains("eEfFgGn%", typ) {
			fail("ValueError", unknown)
		}
		f = fl.f
	}
	neg := math.Signbit(f) && !math.IsNaN(f)
	a := math.Abs(f)
	upper := typ == "E" || typ == "F" || typ == "G"
	body := ""
	switch {
	case math.IsNaN(a):
		body = "nan"
	case math.IsInf(a, 0):
		body = "inf"
	case typ == "f" || typ == "F" || typ == "%":
		p := prec
		if p < 0 {
			p = 6
		}
		if typ == "%" {
			a *= 100
		}
		body = strconv.FormatFloat(a, 'f', p, 64)
		if alt && p == 0 {
			body += "."
		}
	case typ == "e" || typ == "E":
		p := prec
		if p < 0 {
			p = 6
		}
		body = strconv.FormatFloat(a, 'e', p, 64)
		if alt && p == 0 {
			body = body[:1] + "." + body[1:]
		}
	case typ == "g" || typ == "G" || typ == "n":
		p := prec
		if p < 0 {
			p = 6
		}
		body = pyFmtGeneral(a, p, alt, false)
	case prec < 0:
		body = pyFloStr(a)
	default:
		body = pyFmtGeneral(a, prec, alt, true)
	}
	if upper {
		body = strings.ToUpper(body)
	}
	if typ == "%" {
		body += "%"
	}
	if zflag && neg && strings.Trim(body, "0.%eE+-") == "" {
		neg = false
	}
	ip := 0
	for ip < len(body) && body[ip] >= '0' && body[ip] <= '9' {
		ip++
	}
	return rt.pyFmtNum(neg, sign, "", body[:ip], body[ip:], group, 3, fill, align, width, zero)
}

// pyFmtGeneral is C's %g choice between the fixed and the exponent shape, on the
// exponent of the value rounded to p digits. keepOne is the none type's: at
// least one fractional digit, and the exponent shape from p - 1 on. alt keeps
// the trailing zeros and the point.
func pyFmtGeneral(a float64, prec int, alt, keepOne bool) string {
	p := prec
	if p == 0 {
		p = 1
	}
	e := strconv.FormatFloat(a, 'e', p-1, 64)
	ge, _ := strconv.Atoi(e[strings.IndexByte(e, 'e')+1:])
	limit := p
	if keepOne {
		limit = p - 1
	}
	if ge < -4 || ge >= limit {
		at := strings.IndexByte(e, 'e')
		mant, tail := e[:at], e[at:]
		if !alt {
			if strings.Contains(mant, ".") {
				mant = strings.TrimRight(strings.TrimRight(mant, "0"), ".")
			}
		} else if !strings.Contains(mant, ".") {
			mant += "."
		}
		return mant + tail
	}
	out := strconv.FormatFloat(a, 'f', p-1-ge, 64)
	if !alt && strings.Contains(out, ".") {
		out = strings.TrimRight(strings.TrimRight(out, "0"), ".")
	}
	if !strings.Contains(out, ".") {
		if keepOne {
			out += ".0"
		} else if alt {
			out += "."
		}
	}
	return out
}

// pyFmtGroup groups digits from the right in runs of n by sep.
func pyFmtGroup(digits, sep string, n int) string {
	out := ""
	c := 0
	for i := len(digits) - 1; i >= 0; i-- {
		if c == n {
			out, c = sep+out, 0
		}
		out = digits[i:i+1] + out
		c++
	}
	return out
}

// pyFmtNum is a number's text: sign, prefix, the grouped integer digits and the
// rest, padded. With the 0 flag and a grouping the padding zeros are grouped
// too (format(1234, "010,") is 00,001,234).
func (rt *jsrt) pyFmtNum(neg bool, sign rune, prefix, digits, rest, group string, n int, fill string, align rune, width int, zero bool) string {
	sg := ""
	switch {
	case neg:
		sg = "-"
	case sign == '+':
		sg = "+"
	case sign == ' ':
		sg = " "
	}
	g := digits
	if group != "" {
		g = pyFmtGroup(digits, group, n)
		if zero && fill == "0" && align == '=' && digits != "" {
			for len(sg)+len(prefix)+len(g)+pyStrLen(rt, rest) < width {
				digits = "0" + digits
				g = pyFmtGroup(digits, group, n)
			}
		}
	}
	return rt.pyFmtPad(fill, align, width, sg+prefix, g+rest)
}

// pyFmtPad pads head + body to width code points; '=' pads between the two.
func (rt *jsrt) pyFmtPad(fill string, align rune, width int, head, body string) string {
	pad := width - pyStrLen(rt, head+body)
	if pad <= 0 {
		return head + body
	}
	switch align {
	case '<':
		return head + body + strings.Repeat(fill, pad)
	case '=':
		return head + strings.Repeat(fill, pad) + body
	case '^':
		left := pad / 2
		return strings.Repeat(fill, left) + head + body + strings.Repeat(fill, pad-left)
	}
	return strings.Repeat(fill, pad) + head + body
}

// pyStrLen counts Unicode CODE POINTS: Python's len() of a string, where a
// surrogate pair is one character. Only the Python grammars ask for this.
// pyStrChars is a str as Python iterates it: one string per CODE POINT, so an
// astral character is one element where the UTF-16 units make it two. Twin of
// pySChars in lib/python-rt.metajs and python-interpreter.abnf.
func pyStrChars(rt *jsrt, s string) []interface{} {
	e := rt.strMeta(s)
	if e.shape == 1 {
		out := make([]interface{}, len(s))
		for i := 0; i < len(s); i++ {
			out[i] = s[i : i+1]
		}
		return out
	}
	out := make([]interface{}, 0, len(e.units))
	for i, n := 0, len(e.units); i < n; {
		k := 1
		if c := e.units[i]; c >= 0xD800 && c <= 0xDBFF && i+1 < n && e.units[i+1] >= 0xDC00 && e.units[i+1] <= 0xDFFF {
			k = 2
		}
		out = append(out, strFromUnits(e.units[i:i+k]))
		i += k
	}
	return out
}

func pyStrLen(rt *jsrt, s string) int {
	n, unitN := 0, rt.strLen(s)
	for i := 0; i < unitN; i++ {
		c := rt.strCodeAt(s, i)
		if c >= 0xD800 && c <= 0xDBFF && i+1 < unitN {
			i++
		}
		n++
	}
	return n
}

// pyModuleScope is the scope Python's `global` means: the OUTERMOST binding
// boundary in the chain, which is the module top level (the host runtime's own
// scope sits above it and is not a Python namespace). Falls back to the true
// root for a chain that carries no boundary at all.
func pyModuleScope(s *jsScope) *jsScope {
	root, mod := s, (*jsScope)(nil)
	for ; root.parent != nil; root = root.parent {
		if root.pyFn {
			mod = root
		}
	}
	if root.pyFn {
		mod = root
	}
	if mod != nil {
		return mod
	}
	return root
}

// pySeqRepeat is Python's sequence repetition: seq * n for a string or a list
// (a tuple is a list here). n is truncated to an integer and a non-positive n
// yields an empty sequence, exactly as MRI... as CPython does. ok is false when
// seq is not a sequence or n is not a number, so the caller falls back to
// numeric multiplication.
func pySeqRepeat(seq, n interface{}) (interface{}, bool) {
	cnt, isNum := n.(float64)
	if !isNum {
		if b, isBool := n.(bool); isBool { // True * "ab" is "ab" in Python.
			cnt = 0
			if b {
				cnt = 1
			}
		} else {
			return nil, false
		}
	}
	k := int(math.Trunc(cnt))
	if k < 0 {
		k = 0
	}
	switch s := seq.(type) {
	case string:
		return strings.Repeat(s, k), true
	case *jsArray:
		out := &jsArray{}
		for i := 0; i < k; i++ {
			out.elems = append(out.elems, s.elems...)
		}
		return out, true
	}
	return nil, false
}
