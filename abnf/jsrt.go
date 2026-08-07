package abnf

import (
	"errors"
	"fmt"
	"math"
	"math/big"
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
// reclamation, so every number a program ever produces used to cost a table
// slot, a boxed float64 and an intern-map entry FOREVER - a loop over a million
// integers left a million of each behind. Integers that fit the payload are
// therefore encoded in the handle itself: nothing is allocated, nothing is
// interned, and the value is recovered by shifting. Handle identity for numbers
// is not observable (=== compares the unwrapped values, and the dict index and
// the trace hooks do the same), so this is invisible to a program.
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
	id  jsHandleID
	fn  *ir.Func
	env uint64 // Scope handle of the creation site.
	ma  *machine
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

// put declares a name or overwrites it, like an assignment into the old map.
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
	count  uint64 // Handles handed out so far = the next free slot.

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

	lastGets [][2]uint64 // The most recent member lookups (obj, key handles), for error messages.

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

	// rubyGlobals holds Ruby's $globals, which live outside every scope: a write
	// inside a method is visible everywhere, and a read of a never-assigned one
	// answers nil. Only js_rgset / js_rgget (ruby-to-llvm-ir.abnf) touch it.
	rubyGlobals map[string]interface{}

	// rubyLambdas remembers which closures came from a lambda literal (->) or
	// Kernel#lambda, which is all that Proc#lambda? needs. Filled by js_rlambda.
	rubyLambdas map[interface{}]bool

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

	// accessorCount counts the getter/setter properties js_defprop has defined in
	// this runtime. While it is zero - which it stays for every grammar that never
	// emits js_defprop, and for every JS program without an accessor - getMember
	// and setMember take their original path and never look for an accessor.
	accessorCount int

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
	pySigs      map[interface{}]*pySig
	pyEllipsis  *jsObject
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
// than a field of the generator for the reason docs/todo.md 2.1 records: a
// per-PROGRAM record cannot survive coroutines, while a value handed to one
// suspension is per-coroutine by construction - only the yield it was aimed at
// ever reads it.
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
			rt.fail("TypeError: Promise resolver is not a function")
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

// noteGet records a member lookup cheaply; failure messages format them lazily.
func (rt *jsrt) noteGet(obj, key uint64) {
	if len(rt.lastGets) > 8 {
		rt.lastGets = rt.lastGets[1:]
	}
	rt.lastGets = append(rt.lastGets, [2]uint64{obj, key})
}

func (rt *jsrt) formatLastGets() string {
	out := ""
	for _, g := range rt.lastGets {
		out += fmt.Sprintf("%s (%T) ", rt.toString(rt.unwrap(g[1])), rt.unwrap(g[0]))
	}
	return out
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
	"js_scope_set": 1 << 0, "js_scope_set_or_create": 1 << 0,
	"js_kget": 1 << 0, "js_kset": 1 << 0, "js_tdecl": 1 << 0, "js_tset": 1 << 0,
	"js_pyset_var": 1 << 0, "js_pyglobal": 1 << 0, "js_pynonlocal": 1 << 0,

	"js_arr_push": 1 << 0, "js_arg": 1 << 0, "js_pyrest": 1 << 0,
	"js_pyprint": 1 << 0, "js_pyexc": 1 << 1,
	"js_call": 1 << 2, "js_mcall": 1 << 2, "js_rmcall": 1 << 2, "js_supercall": 1 << 3,
}

// releaseHandle drops the value behind a handle the IR machine proved dead. Only
// a scope or an argument array is ever released, and only its VALUE: the slot
// stays taken, so a handle that (against the analysis) is used again finds nil
// and fails loudly instead of silently reading whatever moved in.
func (rt *jsrt) releaseHandle(h uint64) {
	if h&numTag != 0 || h >= rt.count {
		return
	}
	chunk, off := rt.chunks[h>>tblChunkBits], h&tblChunkMask
	switch v := chunk[off].(type) {
	case *jsScope:
		chunk[off] = nil
	case *jsArray:
		if !v.pinned {
			chunk[off] = nil
		}
	}
}

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

// alloc puts v in the next free table slot and returns its handle.
func (rt *jsrt) alloc(v interface{}) uint64 {
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
		return t.n / t.d
	case jsCpx:
		return t.re
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
		return jsNumString(t.n) + "/" + jsNumString(t.d)
	case jsCpx:
		return jsNumString(t.re) + "+" + jsNumString(t.im) + "i"
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
		// and fmt prints digits. Mirror that exactly: the old 1e15 cutoff made
		// println(1000000000000000) print 1e+15 under -frozen only. -0 stays a
		// float like in goja (fmt prints it as -0).
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
	// than the null. docs/todo.md 1.9, 1.7.
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
				panic(&jsThrown{value: rt.pyExcInstance("UnboundLocalError",
					"local variable '"+name+"' referenced before assignment")})
			}
			if rt.traced {
				rt.trVar("read", name, v)
			}
			return v
		}
	}
	rt.fail("variable not defined: %s", name)
	return nil
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
		rt.fail("member '%s' of %s", rt.toString(key), rt.toString(obj))
	}
	// Function values understand apply and call like in JS.
	if ks, isStr := key.(string); isStr && (ks == "apply" || ks == "call") && isCallable(obj) {
		return &boundMethod{recv: obj, name: ks}
	}
	// ADDITIVE, Kotlin only: Char.code is a PROPERTY in Kotlin, not a method, so it
	// arrives here rather than at js_ktsmcall. jsChar had no case at all before, so
	// nothing that used to work changes; kotlin-interpreter.abnf answers the same
	// name in kGetField.
	if c, isChar := obj.(jsChar); isChar {
		if ks, isStr := key.(string); isStr && ks == "code" {
			return float64(c.code)
		}
	}
	// ADDITIVE, JavaScript/TypeScript only: then/catch/finally READ AS A VALUE.
	// Promise method dispatch is table-based (promMethod, reached from js_jsmcall),
	// so a promise carries no own slot of those names and `const t = p.then`
	// answered undefined where node answers a function. promIs is true only for an
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
		// kotlin.Map declares size / entries as properties. Both names were absent
		// from the props map and therefore answered undefined, so no existing dict
		// user (Python, Go) can observe a change.
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
		// `arr.length = n` TRUNCATES (or pads with undefined) in JavaScript. It
		// used to abort here with "invalid array index length", while the
		// interpreter grammars, which run on a real JS array, truncated - so a
		// program that clears an array this way diverged.
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
	// any JS PRIMITIVE (`var x = 5; x.foo = 1`), and it used to say
	// "cannot set member 'foo' on float64": the Go reflect TYPE, in a diagnostic for a
	// language that has no float64. The C floor says "member assignment 'foo' on 5"
	// (languages/lib/runtime.c:2971, die3), and so does setMember's own
	// undefined/null arm eleven lines above (jsrt.go:2029) - which is the invariant
	// die3's comment states and this was the one site breaking it. Aligned on the
	// floor's wording, which is also the VALUE-not-TYPE shape the sibling get
	// diagnostic already uses ("member 'foo' of undefined", jsrt.go:1771). The two
	// genuinely Go-bridge failures above keep rv.Type(), because there the Go type IS
	// the thing the user has to fix.
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
		rt.fail("call of a non function value: %s (last member lookups: %s)", rt.toString(callee), rt.formatLastGets())
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
	}
	return rt.strictEq(a, b)
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
			// is why str(KeyError('x')) is "'x'" and not "x". Every engine here
			// used to fake that by pre-repr'ing the ARGUMENT, which made str()
			// right and repr() doubly quoted (docs/todo.md 1.7). The argument is
			// now the key itself and the repr happens here.
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
		if name, ok := rt.pyClassName(t); ok {
			return "<class '" + name + "'>"
		}
		if els, ok := pySetElems(t); ok {
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
		return rt.toString(v)
	default:
		return rt.toString(v)
	}
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
		return nil
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
type jsRat struct{ n, d float64 }
type jsCpx struct{ re, im float64 }

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
		return t.n / t.d
	case jsFlo:
		return t.f
	case jsCpx:
		return t.re
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
// ruby-interpreter.abnf and rbABigNarrow in languages/lib/ruby-rt.metajs: a value
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
			rt.fail("divided by 0")
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
// gets its own exact expansion - `(2.0 ** 70).to_i` printed 1.1805916207174113e+21
// where MRI prints all 22 digits. The twins are intStr in ruby-interpreter.abnf
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

func rubyReOf(v interface{}) float64 {
	if c, ok := v.(jsCpx); ok {
		return c.re
	}
	return rubyToF(v)
}

func rubyImOf(v interface{}) float64 {
	if c, ok := v.(jsCpx); ok {
		return c.im
	}
	return 0
}

func rubyAsRat(v interface{}) jsRat {
	if r, ok := v.(jsRat); ok {
		return r
	}
	return rubyMkRat(rubyToF(v), 1)
}

// rubyFloStr renders a Float the way Ruby does: with a decimal point even when the
// value is whole (1000.0), which is what tells it apart from the Integer 1000, and
// in exponent form OUTSIDE ITS OWN WINDOW - which is not JavaScript's window and was
// the defect this replaced. Ruby's rule is numeric.c flo_to_s: take the shortest
// round-tripping digit run and its decimal point position `decpt`, and go
// exponential when `decpt < -3 || (decpt > DBL_DIG (15) && decpt >= len(digits))`.
//
// THE SECOND HALF OF THE HIGH TEST IS NOT DECORATION, and leaving it out was the
// defect docs/todo.md 1.6 recorded: `decpt > 15` alone sends every 16-digit-point
// value to exponent form, but MRI only does so when the fixed form would have to
// PAD WITH ZEROS - i.e. when there is no fraction left to write. So
// 1234567890123456.0 (decpt 16, 16 digits) IS exponential and
// 3002399751580330.5 (decpt 16, 17 digits) is NOT. Settled against
// /usr/bin/ruby over 3,000 random doubles in [1e15, 1e19): the two-part rule
// reproduces all 3,000, `decpt > 15` alone misses 297 and `decpt > 16` 433.
// Read off ruby 2.6.10p210;
// Float#to_s has not changed between 2.6 and 3.x, and needs no Ruby-3 syntax to
// probe, so the local 2.6 interpreter settles this row:
//
//	                    ruby                    was (JavaScript's window)
//	1e15                1.0e+15                 1000000000000000.0
//	999999999999999.0   999999999999999.0       999999999999999.0
//	1234567890123456.0  1.234567890123456e+15   1234567890123456.0
//	1e20                1.0e+20                 100000000000000000000
//	1e-5                1.0e-05                 0.00001
//	0.0001              0.0001                  0.0001
//	-0.0                -0.0                    0.0     <- the sign was dropped
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
// pyABigFromNum in languages/lib/python-rt.metajs is the same test.
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
// arbitrary precision ints - exactly what math/big's And/Or/Xor/Lsh/Rsh implement.
//
// It used to be rt.toInt32 on both sides: `-1 & 0xffffffff` was -1 where CPython
// says 4294967295, `1 << 40` was 256 because ECMAScript masks the count to 5 bits,
// and every operand past 2^31 was truncated. BOTH HALVES agreed on all of it, so
// --cross and the matrix were structurally blind; python3 3.14.6 is the oracle
// that settles it. Measured on a 944-row differential probe: 439 rows wrong.
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
			rt.fail("unknown method '%s' on an instance", name)
		}
		if m, ok := o.props[name]; ok && isCallable(m) {
			return rt.call(m, jsUndef, args)
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
		rt.fail("unknown list method '%s'", name)
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
// dividends come from fmod being exact where the older `x - floor(x/y)*y` was
// not - floor(1e308/3)*3 rounds, so that form answered 0.0 for 1e308 % 3.0
// where ruby answers 2.0. Measured over an 18-row probe: 9 of 18 were wrong,
// and not only astronomically - 123456789012345678.0 % 7.0 is 3.0 in ruby and
// was 0.0 here, and 1e16 % 3.0 is 1.0 and was 0.0.
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
// Settled against ruby 2.6.10p210 - Float#% is unchanged in 3.x - over a
// 27-row probe: the four signed-zero rows, the four sign-of-divisor rows,
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
		ar, ai, br, bi := rubyReOf(l), rubyImOf(l), rubyReOf(r), rubyImOf(r)
		switch op {
		case "+":
			return jsCpx{re: ar + br, im: ai + bi}
		case "-":
			return jsCpx{re: ar - br, im: ai - bi}
		case "*":
			return jsCpx{re: ar*br - ai*bi, im: ar*bi + ai*br}
		case "/":
			den := br*br + bi*bi
			return jsCpx{re: (ar*br + ai*bi) / den, im: (ai*br - ar*bi) / den}
		}
		rt.fail("Complex does not support '%s'", op)
	}
	if rk == 1 {
		p, q := rubyAsRat(l), rubyAsRat(r)
		switch op {
		case "+":
			return rubyMkRat(p.n*q.d+q.n*p.d, p.d*q.d)
		case "-":
			return rubyMkRat(p.n*q.d-q.n*p.d, p.d*q.d)
		case "*":
			return rubyMkRat(p.n*q.n, p.d*q.d)
		case "/":
			return rubyMkRat(p.n*q.d, p.d*q.n)
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
			rt.fail("divided by 0")
		}
		return math.Floor(x / y)
	case "%":
		if y == 0 {
			rt.fail("divided by 0")
		}
		// math.FMA AND NOT `x - math.Floor(x/y)*y`, which is the same value only by
		// ACCIDENT on this machine. Go fuses that expression into a single arm64
		// FMSUBD - verified in `go tool objdump` - so the product and the subtract
		// round ONCE, and 9007199254740991 % -3 is -2. clang does not fuse the same
		// source in the C floor, rounds twice, and answers -1. Real ruby 2.6.10
		// says -2, so the FUSED answer is the correct one and layer 2 grew
		// rbFmaSub (Dekker two-product / two-sum) to reproduce it exactly.
		//
		// That left the Go twin agreeing for a reason no source line states. An
		// amd64 build, a GOARM64 without FMA, or a compiler that simply stopped
		// fusing would silently move this half off rbFmaSub and re-open the
		// divergence. math.FMA IS the fused operation, by definition rather than by
		// codegen, and it lowers to the same single FMADDD here.
		return math.FMA(-math.Floor(x/y), y, x)
	case "**":
		if y < 0 {
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

// rubyFindMethod walks the __class / __super chain of an instance for a method.
func rubyFindMethod(v interface{}, name string) (interface{}, bool) {
	o, ok := v.(*jsObject)
	if !ok {
		return nil, false
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
				if _, isExc := o.props["args"]; isExc {
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
	x := rubyToF(t)
	_, isInt := t.(float64)
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
		return rt.rubyStr(t), true
	case "to_i", "to_int", "truncate":
		return rt.rubyIntFromF(math.Trunc(x)), true
	case "to_f":
		if f, ok := t.(jsFlo); ok {
			return f, true
		}
		return jsFlo{f: x}, true
	case "to_r":
		return rubyAsRat(t), true
	case "abs", "magnitude":
		if _, ok := t.(jsFlo); ok {
			return jsFlo{f: math.Abs(x)}, true
		}
		return math.Abs(x), true
	case "floor":
		return rt.rubyIntFromF(math.Floor(x)), true
	case "ceil":
		return rt.rubyIntFromF(math.Ceil(x)), true
	case "round":
		if len(args) > 0 {
			p := math.Pow(10, rubyToF(args[0]))
			return jsFlo{f: math.Floor(x*p+0.5) / p}, true
		}
		if x < 0 {
			return rt.rubyIntFromF(-math.Floor(-x + 0.5)), true
		}
		return rt.rubyIntFromF(math.Floor(x + 0.5)), true
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
	case "chr":
		return string(rune(int32(x))), true
	case "integer?":
		return isInt, true
	case "numerator":
		return rubyAsRat(t).n, true
	case "denominator":
		return rubyAsRat(t).d, true
	case "times": // n.times { |i| ... }
		for i := 0.0; i < x; i++ {
			rt.call(argAt(args, 0), jsUndef, []interface{}{i})
		}
		return t, true
	case "upto":
		for i := x; i <= rubyToF(argAt(args, 0)); i++ {
			rt.call(argAt(args, 1), jsUndef, []interface{}{i})
		}
		return t, true
	case "downto":
		for i := x; i >= rubyToF(argAt(args, 0)); i-- {
			rt.call(argAt(args, 1), jsUndef, []interface{}{i})
		}
		return t, true
	case "divmod":
		// The remainder is the FUSED subtraction, spelled so it does not depend on
		// the compiler choosing to fuse it - see the "%" arm of rubyNumBin, and
		// rbFmaSub in languages/lib/ruby-rt.metajs, which is layer 2's emulation of
		// exactly this operation.
		y := rubyToF(argAt(args, 0))
		q := math.Floor(x / y)
		return &jsArray{elems: []interface{}{q, math.FMA(-q, y, x)}}, true
	case "fdiv":
		return jsFlo{f: x / rubyToF(argAt(args, 0))}, true
	case "pow", "**":
		return rt.rubyBin("**", t, argAt(args, 0)), true
	case "gcd":
		return rubyGcd(x, rubyToF(argAt(args, 0))), true
	// between? / clamp used to be here, on rubyToF. They are Comparable's, they
	// are shared with String, and on a big Integer the double was not exact -
	// rubyMethod answers both through rubySpaceship before this table is reached.
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
	return nil, false
}

// rubyBigMethod is the Integer method set for a value past 2^53, and the twin of
// bigMethod in ruby-interpreter.abnf and rbABigMethod in ruby-rt.metajs. Every
// step is routed through the exact arithmetic instead of the double -
// `(2**64).abs` answered 1.8446744073709552e+19 through rubyToF. ok=false hands
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
	case "to_i", "to_int", "truncate", "floor", "ceil", "round":
		return t, true
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
	}
	// A Proc / lambda / block closure. Without this arm `blk.class` and
	// `blk.is_a?(Proc)` aborted both compiled halves with "method call 'class' on a
	// function", where ruby-interpreter.abnf answered off classOf.
	if isCallable(v) {
		return "Proc"
	}
	return ""
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
	rt.rubyClasses[name] = c
	return c
}

// ----- Ruby ranges -----
// A range with both bounds is MATERIALIZED as an array (that is what the whole
// lowering of ruby-to-llvm-ir.abnf expects: `for i in 1..3', `a[1..2]',
// `(1..n).each'), which rubyRangeArr builds - for numbers and for the String
// ranges "a".."c" alike. A range with an OPEN end ((5..), (..9)) has no array to
// build, so it stays the rubyOpenRange object below, which answers the membership
// questions that are the only thing such a range is used for.

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

// rubyOpenRange builds the object standing for a range with a missing bound.
func rubyOpenRange(lo, hi interface{}, excl bool) *jsObject {
	o := newJSObject()
	o.set("__rrange", true)
	o.set("begin", lo)
	o.set("end", hi)
	o.set("excl", excl)
	return o
}

// rubyOpenRangeCover is `r.cover?(v)' / `r === v' for such a range: the missing
// side never constrains.
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
func rubyExcObj(v interface{}) bool {
	o, ok := v.(*jsObject)
	if !ok {
		return false
	}
	if _, isCls := o.props["__isclass"]; isCls {
		return false
	}
	_, hasArgs := o.props["args"]
	_, hasClass := o.props["__class"]
	return hasArgs && hasClass
}

// rubyExcParent is the ancestry of the builtin exception classes: the part of
// Ruby's class tree a `rescue' clause has to know about. Exception is the root.
var rubyExcParent = map[string]string{
	"StandardError":       "Exception",
	"RuntimeError":        "StandardError",
	"ArgumentError":       "StandardError",
	"TypeError":           "StandardError",
	"NameError":           "StandardError",
	"NoMethodError":       "NameError",
	"IndexError":          "StandardError",
	"KeyError":            "IndexError",
	"RangeError":          "StandardError",
	"ZeroDivisionError":   "StandardError",
	"StopIteration":       "IndexError",
	"FrozenError":         "RuntimeError",
	"NotImplementedError": "StandardError",
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
func (rt *jsrt) rubyIsA(v interface{}, cls interface{}) bool {
	clsObj, ok := cls.(*jsObject)
	if !ok {
		return false
	}
	if bn, isBuiltin := clsObj.props["__rbuiltin"].(string); isBuiltin {
		kind := rubyClassName(v)
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
		case "Object", "BasicObject", "Kernel":
			return true
		case "Enumerable":
			return kind == "Array" || kind == "Hash"
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

// rubyExc builds an exception instance: {__class: <class>, args: [message]}, the
// shape the message reader and the rescue type test below both understand.
func (rt *jsrt) rubyExc(cls *jsObject, args []interface{}) *jsObject {
	inst := newJSObject()
	inst.set("__class", cls)
	inst.set("args", &jsArray{elems: append([]interface{}{}, args...)})
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

// rubyExcMessage is Exception#message / #to_s: the first constructor argument,
// or the class name when the exception was raised without one (MRI's default).
func (rt *jsrt) rubyExcMessage(o *jsObject) (interface{}, bool) {
	if arr, ok := o.props["args"].(*jsArray); ok && len(arr.elems) > 0 {
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
		return jsRat{n: -t.n, d: t.d}
	case jsCpx:
		return jsCpx{re: -t.re, im: -t.im}
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
		return jsNumString(t.n) + "/" + jsNumString(t.d)
	case jsCpx:
		return jsNumString(t.re) + "+" + jsNumString(t.im) + "i"
	case *jsArray:
		return rt.rubyInspect(v)
	case *jsObject:
		if _, _, isDict := dictParts(t); isDict {
			return rt.rubyInspect(v)
		}
		// A CLASS object stringifies as its name (`Foo.to_s` / "#{Foo}" == "Foo"),
		// not through the generic object path, which printed "[object Object]".
		if _, isCls := t.props["__isclass"]; isCls {
			if n, ok := t.props["__name"].(string); ok {
				return n
			}
		}
		if m, ok := rubyFindMethod(t, "to_s"); ok {
			return rt.rubyStr(rt.call(m, jsUndef, []interface{}{t}))
		}
		// An exception instance renders as its message, like Exception#to_s in MRI
		// (so `"log " + e' and "#{e}" read the way a Ruby program expects).
		if _, isExc := t.props["args"]; isExc {
			if msg, ok := rt.rubyExcMessage(t); ok {
				return rt.toString(msg)
			}
		}
	}
	return rt.toString(v)
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
	case *jsArray:
		parts := make([]string, len(t.elems))
		for i, e := range t.elems {
			parts[i] = rt.rubyInspect(e)
		}
		return "[" + strJoin(parts, ", ") + "]"
	case *jsObject:
		if keys, vals, isDict := dictParts(t); isDict {
			parts := make([]string, len(keys.elems))
			for i := range keys.elems {
				// NO SPACES around the arrow: MRI's Hash#inspect writes
				// `{1=>2, "a"=>[1, 2]}`, and the spaced form this used to write was
				// 48 of the 58 residual rows in the bignum probe (docs/todo.md 2.11).
				parts[i] = rt.rubyInspect(keys.elems[i]) + "=>" + rt.rubyInspect(vals.elems[i])
			}
			return "{" + strJoin(parts, ", ") + "}"
		}
	}
	return rt.rubyStr(v)
}

// rubyEq is Ruby's ==: numeric across the whole tower (2.0 == 2, 1r/2r == 0.5), a
// user class's own == when it defines one, and the structural comparison of
// Array/Hash (pyEqual) otherwise. A Symbol only equals another Symbol.
func (rt *jsrt) rubyEq(l, r interface{}) bool {
	if rubyIsNum(l) && rubyIsNum(r) {
		if _, lc := l.(jsCpx); lc {
			return rubyReOf(l) == rubyReOf(r) && rubyImOf(l) == rubyImOf(r)
		}
		if _, rc := r.(jsCpx); rc {
			return rubyReOf(l) == rubyReOf(r) && rubyImOf(l) == rubyImOf(r)
		}
		if c, ok := rubyBigCmp(l, r); ok {
			return c == 0
		}
		return rubyToF(l) == rubyToF(r)
	}
	if rubyUserObj(l) {
		if m, ok := rubyFindMethod(l, "=="); ok {
			return rubyTruthy(rt.call(m, jsUndef, []interface{}{l, r}))
		}
	}
	// An ABSENT argument reaches this half as UNDEFINED (js_arg / js_rblockarg
	// answer undefined for one that was not passed) and ruby-interpreter.abnf as
	// nil. Both spell Ruby's nil, so `b == nil` and `[a, b] == [1, nil]` have to
	// hold either way - they answered false here and true there.
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
	// the shared pyEqual rubyStructEq ends in applies Python's rule and says true.
	// Found by the Array probe: `[true,false,nil].include?(0)` answered true under
	// llvm.Run and false in the interpreter AND in the native binary, whose layer-2
	// rbPyEqual never had the coercion.
	if lb, isB := l.(bool); isB {
		rb, ok := r.(bool)
		return ok && rb == lb
	}
	if _, isB := r.(bool); isB {
		return false
	}
	return rt.rubyStructEq(l, r)
}

// rubyStructEq is Array#== and Hash#==: element by element and key by key, with
// each element compared by RUBY'S ==. It cannot be pyEqual, which is where this
// used to end: pyEqual recurses into an array with ITSELF, so an element pair
// never reached rubyEq's numeric tower and `[1] == [1.0]` was false where MRI
// says true. The Hash arm looks its keys up with rubyDictFind, so an Array key
// is found the same way it is everywhere else.
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
// `{[1] => 5}[[1]]` is 5 where the `===` scan answered nil - docs/todo.md 2.11.
// A Hash keys the same way.
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
	if la, ok := l.(*jsArray); ok {
		if ra, ok2 := r.(*jsArray); ok2 {
			// ONE incomparable element pair makes the WHOLE comparison nil -
			// skipping it and carrying on made `[1, "a"] <=> [1, 2]` answer 0
			// where MRI answers nil - and the answer is NORMALISED to -1/0/1, so
			// `[1,2,3] <=> [1]` is 1, not 2. Both measured against ruby 2.6.10p210.
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
// strconv's own "+Inf" reached the output before this. Measured against ruby 2.6.10,
// whose float formatting is unchanged in 3.x.
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
// The rule is NOT "the digits are not zero" - that was the first spelling and it
// breaks the moment a precision pads them: "%#.5x" % 0 is "00000" in MRI, with no
// prefix, and the digit string is neither "0" nor "". The test is on the VALUE.
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
	}
	return ""
}

// ----- the three integer directives ----------------------------------------
//
// %d/%i/%x/%X/%o/%b took `strconv.FormatInt(int64(f), base)` and jsNumString,
// which are wrong in two independent ways past 2^53: int64() SATURATES (1e30
// came out "7fffffffffffffff") and jsNumString prints the double's SHORTEST
// form ("1e+30"). MRI converts the Float with to_i - truncation toward zero -
// and then prints the resulting Integer EXACTLY, all 31 digits of
// 1e30.to_i == 1000000000000000019884624838656. Settled against ruby 2.6.10 over
// a 29-format x 22-value table; Integer formatting is unchanged in 3.x.
//
// A double is an integer times a power of two, so `big.Float.Int` is exact and
// `Int.Text(base)` is the answer for every base here. Layer 2 has no bignum
// multiply, so rbMagDigits there takes the binary expansion and regroups it in
// threes and fours; the two agree by construction and the probe checks it.
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
			prefix = rubyBasePrefix('b', f)
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

// rubyFormat is Kernel#format / String#% - the printf directives Ruby shares with C:
// %[-][0][+][ ][#][width][.prec](d|i|f|s|x|X|o|b|e|E|g|G|%). The right operand is
// the argument, or an array of them.
//
// %g/%G is C's "shortest of the two shapes": the value rounded to P significant
// digits (P defaults to 6, and P == 0 means 1) is written in %e form when its
// decimal exponent is below -4 or at least P, and in %f form otherwise, with the
// trailing zeros of the fraction - and a bare trailing point - removed. That is
// exactly strconv.FormatFloat(v, 'g', P, 64), verified digit for digit against
// ruby 2.6.10 over 14 values x 4 precisions.
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
		case 'd', 'i', 'x', 'X', 'o', 'b':
			base := 10
			switch spec[i] {
			case 'x', 'X':
				base = 16
			case 'o':
				base = 8
			case 'b':
				base = 2
			}
			// The argument's EXACT integer value: an Integer past 2^53 is a big
			// and carries all of it, and a Float is truncated toward zero (MRI's
			// %d converts with to_i) and then expanded exactly.
			av := next()
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
				// FloatDomainError - and left exactly as it was rather than
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
			body, prefix, fill, keep = rubyIntBody(iv, base, spec[i] == 'X', alt, plus || space, prec)
			// A precision turns the 0 flag OFF for an integer directive, as it does
			// in C: "%08.3x" % 1 is "     001".
			intPrec = prec >= 0
		case 'f':
			if prec < 0 {
				prec = 6
			}
			f := rubyToF(next())
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
			f := rubyToF(next())
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
			f := rubyToF(next())
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
			if prec >= 0 && prec < len(body) {
				body = body[:prec]
			}
		default:
			body = string(spec[i])
		}
		// The + and SPACE flags were parsed and thrown away, in every half, so
		// "%+d" % 7 was "7" against MRI's "+7" - both halves agreeing and both
		// wrong, the one class byte-identity cannot see. They apply to the NUMERIC
		// directives only (a string never grows a sign), and they apply to Inf and
		// NaN too ("%+f" % Float::INFINITY is "+Inf").
		numeric := false
		switch spec[i] {
		case 'd', 'i', 'f', 'e', 'E', 'g', 'G', 'x', 'X', 'o', 'b':
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
		// The 0 flag is for NUMBERS: MRI's "%05s" % "ab" is "   ab", where this
		// used to pad with zeros. A non-finite float is never zero-padded either,
		// and neither is an integer directive that carried a precision.
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
		for len(body) < width {
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
	// Ruby's universal reflective methods, answered for every kind of value.
	switch name {
	case "class":
		if kind := rubyClassName(target); kind != "" {
			return rt.rubyBuiltinClass(kind)
		}
	case "is_a?", "kind_of?", "instance_of?":
		if rubyClassName(target) != "" && !rubyUserObj(target) {
			return rt.rubyIsA(target, argAt(args, 0))
		}
	case "nil?":
		return isUndefOrNull(target)
	case "to_s", "inspect":
		// TrueClass / FalseClass. Without this arm both compiled halves fell
		// through to the shared memberCall and aborted with "method call
		// 'inspect' on a boolean", where ruby-interpreter.abnf answered - so a `p`
		// or an `.inspect` of a predicate result was a halves divergence.
		if bv, isBool := target.(bool); isBool {
			if bv {
				return "true"
			}
			return "false"
		}
	case "message", "full_message":
		// Exception#message: the raise argument, or the class name. Only for an
		// INSTANCE that does not define a message method of its own, so a user
		// class named message stays in charge.
		if o, isObj := target.(*jsObject); isObj {
			if _, isCls := o.props["__isclass"]; !isCls {
				if _, hasOwn := rubyFindMethod(target, name); !hasOwn {
					if msg, ok := rt.rubyExcMessage(o); ok {
						return msg
					}
				}
			}
		}
	case "frozen?":
		_, isStr := target.(string)
		return isStr || rubyIsNum(target)
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
		if isCallable(target) {
			return target
		}
	case "between?", "clamp":
		// Comparable#between? and #clamp are defined by <=> ALONE, so Integer,
		// Float, Rational, Symbol and String all answer them off ONE pair of arms.
		// They used to sit in the numeric table only, which left `"m".between?
		// ("a", "z")` aborting here and everywhere (docs/todo.md 2.9) - and that
		// table compared through rubyToF, so a big Integer answered on the double
		// rather than exactly. rubySpaceship is the whole rule and it is
		// big-aware. A user class that defines either method itself still wins:
		// its own dispatch is below.
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
			return float64(rt.strLen(o.s))
		case "inspect":
			return ":" + o.s
		case "upcase":
			return jsSym{s: strings.ToUpper(o.s)}
		case "downcase":
			return jsSym{s: strings.ToLower(o.s)}
		}
		rt.fail("unknown Symbol method: %s", name)
	case string:
		switch name {
		case "length", "size":
			return float64(rt.strLen(o))
		case "to_s":
			return o
		case "inspect":
			// The QUOTED form. Missing from both compiled halves, where
			// ruby-interpreter.abnf answered - so `p` of a string, or any
			// .inspect reaching a String, was a halves divergence.
			return rt.rubyInspect(o)
		case "to_sym", "intern":
			return jsSym{s: o}
		// String#ord: the code point of the FIRST character. It worked in
		// ruby-interpreter.abnf and aborted here, so `"\\e".ord` - the ordinary way
		// to name a control character - was a halves divergence.
		case "ord":
			if r := []rune(o); len(r) > 0 {
				return float64(r[0])
			}
			return float64(0)
		// String#to_i / #to_f parse a LEADING number and answer 0 when there is
		// none. Both worked in ruby-interpreter.abnf and aborted here, so any
		// program reading a number out of text was a halves divergence.
		case "to_i":
			if v := jsParseInt(o, 10); v == v {
				return v
			}
			return float64(0)
		case "to_f":
			if v := jsParseFloat(o); v == v {
				return jsFlo{f: v}
			}
			return jsFlo{f: 0}
		case "upcase":
			return strings.ToUpper(o)
		case "downcase":
			return strings.ToLower(o)
		case "include?":
			return strings.Contains(o, rt.toString(argAt(args, 0)))
		case "dup", "clone", "freeze", "to_str", "+@", "-@", "itself":
			// A Ruby String is mutable, so `s = "a".dup` hands back a copy; the
			// compiler models a String as a value, which makes the copy the value.
			return o
		}
		rt.fail("unknown String method: %s", name)
	case *jsArray:
		return rt.rubyArrayMethod(o, name, args)
	case *jsObject:
		if rubyIsEnum(o) {
			return rt.rubyEnumMethod(o, name, args)
		}
		if _, isRng := o.props["__rrange"]; isRng {
			// A range with an open end: only the membership questions make sense.
			switch name {
			case "cover?", "include?", "member?", "===":
				return rt.rubyOpenRangeCover(o, argAt(args, 0))
			case "begin", "first":
				return o.props["begin"]
			case "end", "last":
				return o.props["end"]
			case "exclude_end?":
				return o.props["excl"]
			}
			rt.fail("unknown method on an endless Range: %s", name)
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
			case "include?", "has_key?", "key?":
				return rt.rubyDictFind(keys, argAt(args, 0)) >= 0
			case "each":
				for i := range keys.elems {
					rt.call(argAt(args, 0), jsUndef, []interface{}{keys.elems[i], vals.elems[i]})
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
				rt.fail("key not found: %s", rt.rubyInspect(argAt(pos, 0)))
			}
			if v, handled := rt.rubyHashMethod(o, keys, vals, name, args); handled {
				return v
			}
			rt.fail("unknown Hash method: %s", name)
		}
		// Ruby's type predicates. is_a?/kind_of? walk the __super chain of the
		// instance's class; instance_of? compares the exact class.
		switch name {
		case "class":
			if c, ok := o.props["__class"]; ok {
				return c
			}
		case "instance_of?":
			return o.props["__class"] == argAt(args, 0)
		case "is_a?", "kind_of?":
			want := argAt(args, 0)
			for cls := o.props["__class"]; cls != nil; {
				if cls == want {
					return true
				}
				clsObj, ok := cls.(*jsObject)
				if !ok {
					break
				}
				cls = clsObj.props["__super"]
			}
			return false
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
			// Class#name and Class#inspect are the class NAME, which is what
			// Class#to_s already answered here: `Foo.inspect' and `Foo.name'
			// aborted this half while ruby-interpreter.abnf answered.
			switch name {
			case "name", "inspect":
				return rt.rubyStr(o)
			}
		}
		// Kernel.format / Kernel.sprintf / Kernel.rand: the Kernel module's methods
		// are ordinarily called without a receiver, but naming Kernel explicitly is
		// legal Ruby and aborted in every engine.
		if bn, isBuiltin := o.props["__rbuiltin"].(string); isBuiltin && bn == "Kernel" {
			switch name {
			case "format", "sprintf":
				return rt.rubyFormatArgs(args)
			case "rand":
				return rubyRand(argAt(args, 0))
			}
		}
		// A class instance or class object: the generic dispatch handles it.
		return rt.memberCall(target, name, args)
	}
	if isUndefOrNull(target) {
		rt.fail("method .%s on nil", name)
	}
	return rt.memberCall(target, name, args)
}

// rubyRand is Kernel#rand. There is no random source in this project and there
// cannot be one the two engines agree on byte-for-byte, so rand is
// DETERMINISTICALLY ZERO - but zero of the right class, which is the part a
// program can test: `rand` and `rand(0)` are Floats in [0,1), `rand(n)` an Integer
// in [0,n), and `rand(a..b)` the range's own low end. It aborted with `variable
// not defined: rand` in both compiler halves.
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

// rubyBlockArgs splits a Ruby argument list into its positional part and the
// trailing block. The two compiler halves pass a block as the LAST positional
// argument (see unwrapArgs in ruby-to-llvm-ir.abnf), where ruby-interpreter.abnf
// keeps it in a parameter of its own - which is why arrayMethod there takes a
// fourth argument and this does not.
// rubyArity is the parameter shape Proc#arity is computed from: how many
// parameters are required, how many have defaults, and whether there is a *splat.
// Keyword and block parameters are not counted at all.
type rubyArity struct {
	req  int
	opt  int
	star bool
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
	if a.star || (rt.rubyLambdas[f] && a.opt > 0) {
		return float64(-(a.req + 1))
	}
	return float64(a.req)
}

func rubyBlockArgs(args []interface{}) ([]interface{}, interface{}) {
	if n := len(args); n > 0 && isCallable(args[n-1]) {
		return args[:n-1], args[n-1]
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
	return rt.rubyArrayMethod(hp, name, args), true
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
// list, not the receiver. An eager Enumerator that forgot the op answered the
// Enumerator itself and every `map.with_index` was silently wrong.
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
	return rt.rubyArrayMethod(items, name, args)
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
	case "push", "append", "add":
		// Ruby's Array#push is VARIADIC (`a.push(1, 2, 3)`); taking only args[0]
		// dropped every further element silently. `a.push` with no argument is a
		// no-op in Ruby, so an empty args list must append nothing.
		t.dropIdx()
		t.elems = append(t.elems, args...)
		return t
	// first / last / pop / shift all take an optional COUNT, and then answer an
	// ARRAY. Without it they answered the single element whatever was passed, so
	// `[1,2,3].first(2)` was a silent 1 in every engine.
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
		// `[1].include?(1.0)` are both true in MRI. It was a `===` scan, which
		// answered false to both. A plain scan and not the dict index, because an
		// ordinary data array must not grow a key index just to be searched.
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
	// sum was an int32 accumulator, which truncated every Float and every value
	// past 2^31: `[1.5, 2.5].sum` answered 3. Ruby's sum is ordinary + from an
	// Integer 0 (or from the given initial value), the block applied first.
	// MRI's sum starts from an Integer 0 and REFUSES a non-numeric element:
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
		// A closed range reaches the compiler halves ALREADY MATERIALIZED as an
		// array (that is what ruby-to-llvm-ir.abnf emits for 1..2), so an array
		// first argument here IS the range: its first element is the low bound and
		// its length is the count.
		if r, isArr := argAt(pos, 0).(*jsArray); isArr {
			if len(r.elems) == 0 {
				return &jsArray{}
			}
			return rubyArrSub(t, rubyArrNorm(t, r.elems[0]), len(r.elems))
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
	}
	rt.fail("unknown Array method: %s", name)
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
			for i, e := range recv.elems {
				if rt.strictEq(e, argAt(args, 0)) {
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
			return strings.ToUpper(recv)
		case "toLowerCase":
			return strings.ToLower(recv)
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
// code units, not bytes, or the engines diverge on any non-ASCII string
// ("é".length was 1 under goja but the byte count under -frozen, and the
// byte-based charCodeAt/fromCharCode round trip in unescapeJs double-encoded
// every non-ASCII string literal). ASCII strings take the byte fast path.
//
// A code unit of an astral character is half a surrogate pair, and a lone
// surrogate is not a Unicode scalar value, so Go's UTF-8 encoder replaces it
// with U+FFFD. The unit-at-a-time idiom that every string literal goes through
// (`out += String.fromCharCode(s.charCodeAt(i))` in unescapeJs) therefore took
// an emoji apart into two halves and destroyed both before the concatenation
// could put them back. So a jsrt string is WTF-8, not UTF-8: a lone surrogate
// keeps its own three byte encoding (the encoding UTF-8 would give the code
// point if surrogates were scalars), which strUnits decodes back to the unit it
// came from. Concatenation rejoins a pair split across the seam (strConcat), so
// the halves only ever exist while the round trip is in flight, and printing
// substitutes U+FFFD for whatever lone surrogates are left (wtf8Clean), which is
// what goja writes for them too. Everything below therefore matches goja exactly
// for BMP and astral text alike (see the utf16-astral checks in
// tests/metajs-test-features.js). What is left is a string that deliberately
// keeps a lone half around: the two engines print it the same, but goja will
// have replaced it by U+FFFD as it crossed into a Go string, so byteLen and any
// bytes emitted from it (a string global of a compiled module) still differ.

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
		// (Historically introduced for the typed JS dialect, hence the names.)
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
			rt.noteGet(a[0], a[1])
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
		// Ruby indexing: a[i], h[k], s[i]. Identical to js_pyget except that a missing
		// Hash key and an out-of-range index answer NIL instead of raising - which is
		// Ruby's Hash#[] / Array#[] / String#[], and what ruby-interpreter.abnf does.
		"js_rget": func(a []uint64) uint64 {
			// A Proc answers p[x] by calling itself.
			if isCallable(u(a[0])) {
				return w(rt.call(u(a[0]), jsUndef, []interface{}{u(a[1])}))
			}
			// A user class defines its own indexing with `def [](i)`.
			if rubyUserObj(u(a[0])) {
				return w(rt.rubyMethod(u(a[0]), "[]", []interface{}{u(a[1])}))
			}
			if keys, vals, ok := dictParts(u(a[0])); ok {
				i := rt.rubyDictFind(keys, u(a[1]))
				if i < 0 {
					return w(rt.rubyHashDefault(u(a[0]), u(a[1])))
				}
				return w(vals.elems[i])
			}
			idx := int(rt.toNumber(u(a[1])))
			switch o := u(a[0]).(type) {
			case *jsArray:
				if idx < 0 {
					idx += len(o.elems)
				}
				if idx < 0 || idx >= len(o.elems) {
					return w(jsNull)
				}
				return w(o.elems[idx])
			case string:
				n := rt.strLen(o)
				if idx < 0 {
					idx += n
				}
				if idx < 0 || idx >= n {
					return w(jsNull)
				}
				return rt.wrapStr(rt.strAt(o, idx))
			}
			if isUndefOrNull(u(a[0])) {
				rt.fail("indexing nil")
			}
			return w(rt.getMember(u(a[0]), u(a[1])))
		},
		// Ruby index assignment: a[i] = v, h[k] = v, obj[i] = v. Like js_pyset except
		// that a user class dispatches its own `def []=`, a missing Hash key is
		// APPENDED and an Array GROWS (filling the gap with nil) - all three Ruby.
		"js_rset": func(a []uint64) uint64 {
			t := u(a[0])
			if rubyUserObj(t) {
				rt.rubyMethod(t, "[]=", []interface{}{u(a[1]), u(a[2])})
				return 0
			}
			if keys, vals, ok := dictParts(t); ok {
				if i := rt.rubyDictFind(keys, u(a[1])); i >= 0 {
					vals.elems[i] = u(a[2])
				} else {
					dictAppend(keys, vals, u(a[1]), u(a[2]))
				}
				return 0
			}
			arr, ok := t.(*jsArray)
			if !ok {
				rt.fail("item assignment on a %s", rt.typeOf(t))
			}
			idx := int(rt.toNumber(u(a[1])))
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
			arr.elems[idx] = u(a[2])
			return 0
		},
		"js_supercall": func(a []uint64) uint64 { // (super class, this, method name, args array)
			args, ok := u(a[3]).(*jsArray)
			if !ok {
				rt.fail("js_supercall args must be an array")
			}
			name := rt.toString(u(a[2]))
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
					// pending throw, like in JS. (It used to be discarded via a
					// bare defer, so 'try { return 1 } finally { return 2 }'
					// compiled to 1 while the interpreters returned 2.)
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
			// per-coroutine by construction (docs/todo.md 2.1).
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
		// scBindings), which layer 2 has had since d9014d7 and an EMITTER could
		// not reach: a builtin is callable from MetaJS, an extern from emitted IR,
		// and the own-scope question was only ever answerable on one side.
		//
		// js_scope_typeof is the chain walk and cannot express it - it answers
		// "undefined" for an absent name and for a slot holding undefined alike,
		// and it finds a name an ENCLOSING scope binds. python-to-llvm-ir.abnf's
		// class fill used it as an own-scope test, so a name the class body did
		// not reach was copied onto the class FROM THE MODULE:
		//     w = "module"
		//     class K:
		//         if False:
		//             w = "inner"
		//     K.w                     -> "module", where CPython raises AttributeError
		"js_scope_has": func(a []uint64) uint64 {
			name := rt.toString(u(a[1]))
			sc := rt.scopeOf(a[0])
			if sc == nil {
				return boolH(false)
			}
			_, ok := sc.get(name)
			return boolH(ok)
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
		"js_rflo": func(a []uint64) uint64 { return w(jsFlo{f: rt.toNumber(u(a[0]))}) },
		"js_rrat": func(a []uint64) uint64 { return w(rubyMkRat(rt.toNumber(u(a[0])), rt.toNumber(u(a[1])))) },
		"js_rcpx": func(a []uint64) uint64 { return w(jsCpx{re: rt.toNumber(u(a[0])), im: rt.toNumber(u(a[1]))}) },
		"js_radd": func(a []uint64) uint64 { return w(rt.rubyBin("+", u(a[0]), u(a[1]))) },
		"js_rsub": func(a []uint64) uint64 { return w(rt.rubyBin("-", u(a[0]), u(a[1]))) },
		"js_rmul": func(a []uint64) uint64 { return w(rt.rubyBin("*", u(a[0]), u(a[1]))) },
		"js_rdiv": func(a []uint64) uint64 { return w(rt.rubyBin("/", u(a[0]), u(a[1]))) },
		"js_rmod": func(a []uint64) uint64 { return w(rt.rubyBin("%", u(a[0]), u(a[1]))) },
		"js_rpow": func(a []uint64) uint64 { return w(rt.rubyBin("**", u(a[0]), u(a[1]))) },
		"js_rband": func(a []uint64) uint64 {
			if lb, ok := u(a[0]).(bool); ok {
				return boolH(lb && rubyTruthy(u(a[1])))
			}
			return w(rt.rubyBin("&", u(a[0]), u(a[1])))
		},
		"js_rbor": func(a []uint64) uint64 {
			if lb, ok := u(a[0]).(bool); ok {
				return boolH(lb || rubyTruthy(u(a[1])))
			}
			return w(rt.rubyBin("|", u(a[0]), u(a[1])))
		},
		"js_rbxor": func(a []uint64) uint64 {
			if lb, ok := u(a[0]).(bool); ok {
				return boolH(lb != rubyTruthy(u(a[1])))
			}
			return w(rt.rubyBin("^", u(a[0]), u(a[1])))
		},
		// << appends to an Array (in place, so `a << b << c` chains), concatenates a
		// String and shifts an Integer - one operator, three receivers, like Ruby.
		"js_rshl":  func(a []uint64) uint64 { return w(rt.rubyBin("<<", u(a[0]), u(a[1]))) },
		"js_rshr":  func(a []uint64) uint64 { return w(rt.rubyBin(">>", u(a[0]), u(a[1]))) },
		"js_rneg":  func(a []uint64) uint64 { return w(rt.rubyNeg(u(a[0]))) },
		// ~x is -(x+1) at any size, which math/big's Not is exactly.
		"js_rbnot": func(a []uint64) uint64 {
			// Past 2^53 the double cannot hold the +1, so ~9007199254740992 came
			// back as -9007199254740992 instead of ...993.
			if b, ok := rubyBigOf(u(a[0])); ok && b.CmpAbs(rubyBigMax) >= 0 {
				return w(rt.rubyBigNarrow(new(big.Int).Not(b)))
			}
			return w(-rubyToF(u(a[0])) - 1)
		},
		"js_req":   func(a []uint64) uint64 { return boolH(rt.rubyEq(u(a[0]), u(a[1]))) },
		"js_rne":   func(a []uint64) uint64 { return boolH(!rt.rubyEq(u(a[0]), u(a[1]))) },
		"js_rlt":   func(a []uint64) uint64 { return boolH(rt.rubyCmp(u(a[0]), u(a[1])) < 0) },
		"js_rgt":   func(a []uint64) uint64 { return boolH(rt.rubyCmp(u(a[0]), u(a[1])) > 0) },
		"js_rle":   func(a []uint64) uint64 { return boolH(rt.rubyCmp(u(a[0]), u(a[1])) <= 0) },
		"js_rge":   func(a []uint64) uint64 { return boolH(rt.rubyCmp(u(a[0]), u(a[1])) >= 0) },
		"js_rcmp": func(a []uint64) uint64 { // <=>: nil when the two cannot be compared
			c, ok := rt.rubySpaceship(u(a[0]), u(a[1]))
			if !ok {
				return w(jsNull)
			}
			return rt.wrapNum(float64(c))
		},
		// === (case equality): a Range (an eager array here) matches by membership, a
		// class object by is_a?, everything else by ==.
		// lo..hi / lo...hi with BOTH bounds: the array the lowering iterates and
		// indexes with. Numbers and the String ranges "a".."c" alike.
		"js_rrangearr": func(a []uint64) uint64 {
			return w(rt.rubyRangeArr(u(a[0]), u(a[1]), rt.truthy(u(a[2]))))
		},
		// (5..) / (..9): no array to build, so the range stays an object that answers
		// cover? / include? / === (see rubyOpenRangeCover).
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
				if i == len(arr.elems)-1 && isCallable(arr.elems[i]) {
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
				if kwAt >= 0 && isCallable(pos[kwAt]) {
					kwAt--
				}
				if kwAt >= 0 {
					if _, _, isDict := dictParts(pos[kwAt]); isDict {
						pos = append(pos[:kwAt], pos[kwAt+1:]...)
					}
				}
			}
			// A literal block travels as the last positional argument and is not a
			// value an optional or *splat parameter may take. A REQUIRED parameter
			// still reads it, so `def m(f); end; m(some_proc)` keeps binding f.
			npos := len(pos)
			if npos > 0 && isCallable(pos[npos-1]) {
				npos--
			}
			required := 0
			for i := 0; i < len(kinds); i++ {
				if kinds[i] == '0' {
					required++
				}
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
					if ai < len(pos) {
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
			if ok && len(arr.elems) > 0 && isCallable(arr.elems[len(arr.elems)-1]) {
				return w(arr.elems[len(arr.elems)-1])
			}
			return w(jsNull)
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
						if len(argv) > 0 && isCallable(argv[len(argv)-1]) {
							return w(argv[len(argv)-1])
						}
						return w(jsNull)
					}
					// Array.new(n) / Array.new(n, v) / Array.new(n) { |i| }.
					// Without this arm it fell into the exception construction
					// below and answered the first argument - a silent wrong
					// answer, and a different one in each engine.
					if bn == "Array" {
						return w(rt.rubyArrNew(argv))
					}
					// Hash.new / Hash.new(v) / Hash.new { |h, k| }. The default
					// lives in the hidden __hdef / __hdefp slot and
					// rubyHashDefault honours it at every read; before that the
					// two argument forms fell through to rubyExc below and
					// answered an exception object.
					if bn == "Hash" {
						return w(rubyNewHash(argv))
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
			name := sym.s
			return w(&hostFunc{name: "&:" + name, fn: func(rt *jsrt, this uint64, args []interface{}) interface{} {
				rest := []interface{}{}
				if len(args) > 1 {
					rest = args[1:]
				}
				return rt.rubyMethod(argAt(args, 0), name, rest)
			}})
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
		"js_rdefined": func(a []uint64) uint64 { // (scope, name, kind) -> kind | nil
			name := rt.toString(u(a[1]))
			for s := rt.scopeOf(a[0]); s != nil; s = s.parent {
				if s.has(name) {
					return w(u(a[2]))
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
		// Ruby's to_s, used for string interpolation and puts.
		"js_rstr": func(a []uint64) uint64 { return rt.wrapStr(rt.rubyStr(u(a[0]))) },
		// Kernel#puts and Kernel#print with RUBY rendering. The compiler used to
		// hand the raw value to the shared println host function, which prints it
		// with Go's %v - so `puts nil` said "<nil>", `puts [1, 2]` said "[1 2]" and
		// `puts 1.class` printed a Go map, while ruby-interpreter.abnf printed
		// "", "1\n2" and "Integer". puts also FLATTENS an array (one element per
		// line) and prints nothing at all for an empty one, which is MRI.
		"js_rputs": func(a []uint64) uint64 { rt.rubyPuts(u(a[0])); return 0 },
		"js_rprint": func(a []uint64) uint64 {
			fmt.Fprint(outWriter, wtf8Clean(rt.rubyStr(u(a[0]))))
			return 0
		},
		// Kernel#p: INSPECT, one argument per line, answering the argument (the
		// whole list when there is more than one, nil when there is none). It is
		// one extern rather than an emitted loop because that is what makes the
		// return value the same in both engines. `p` used to resolve as a bare
		// NAME and abort with "variable not defined: p".
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
		"js_phflo":   func(a []uint64) uint64 { return w(phpMkFlo(rt.toNumber(u(a[0])))) },
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
		// which is why it is one call: the compiler used to spell it as a three-block
		// branch around js_phisset + js_get at each read site.
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
			rt.noteGet(a[0], a[1])
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
			// A subscript MISS is a CATCHABLE exception, not an abort. All three
			// engines used to rt.fail() here, so `try: d[k] / except KeyError` -
			// which CPython supports and which js_pydel_item next door already
			// did - killed the process instead (docs/todo.md 1.7). CPython's
			// message for the sequence cases carries no index, so neither does
			// this one any more.
			if keys, vals, ok := dictParts(u(a[0])); ok {
				i := rt.pyDictFind(keys, u(a[1]))
				if i < 0 {
					panic(&jsThrown{value: rt.pyExcInstanceV("KeyError", u(a[1]))})
				}
				return w(vals.elems[i])
			}
			idx := int(rt.toNumber(u(a[1])))
			switch o := u(a[0]).(type) {
			case *jsArray:
				if idx < 0 {
					idx += len(o.elems)
				}
				if idx < 0 || idx >= len(o.elems) {
					panic(&jsThrown{value: rt.pyExcInstance("IndexError", "list index out of range")})
				}
				return w(o.elems[idx])
			case string:
				n := rt.strLen(o)
				if idx < 0 {
					idx += n
				}
				if idx < 0 || idx >= n {
					panic(&jsThrown{value: rt.pyExcInstance("IndexError", "string index out of range")})
				}
				return rt.wrapStr(rt.strAt(o, idx))
			}
			// list[int] / dict[str, int]: subscripting a TYPE (a class object or a
			// builtin like list) is a generic alias, not an indexing. One object per
			// (origin, parameter) pair, so 'list[int] == list[int]' holds.
			if v, ok := rt.pyGenericAlias(u(a[0]), u(a[1])); ok {
				return w(v)
			}
			rt.fail("indexing a %s", rt.typeOf(u(a[0])))
			return 0
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
				rt.fail("item assignment on a %s", rt.typeOf(u(a[0])))
			}
			idx := int(rt.toNumber(u(a[1])))
			if idx < 0 {
				idx += len(arr.elems)
			}
			if idx < 0 || idx >= len(arr.elems) {
				rt.fail("list assignment index out of range")
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
					rt.fail("index %d out of range", i)
				}
				return w(arr.elems[i])
			}
			if str, ok := u(a[0]).(string); ok { // s[i] is the i-th BYTE of a Go string
				i := int(rt.toNumber(u(a[1])))
				if i < 0 || i >= len(str) {
					rt.fail("index %d out of range", i)
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
		// One arm of a Go type switch: does v have dynamic type `want`? The value
		// model has a single numeric type, so an integral number is an int and a
		// fractional one a float64; a map, a slice and a struct answer their own
		// spelling, and any / interface{} matches every non-nil value.
		"js_gotypeis": func(a []uint64) uint64 {
			want := rt.toString(u(a[1]))
			v := u(a[0])
			got := "any"
			switch t := v.(type) {
			case jsUndefT, jsNullT:
				got = "nil"
			case bool:
				got = "bool"
			case string:
				got = "string"
			case float64:
				if t == math.Trunc(t) {
					got = "int"
				} else {
					got = "float64"
				}
			case *jsArray:
				got = "slice"
			case *jsObject:
				if _, _, isDict := dictParts(v); isDict {
					got = "map"
				} else if cls, has := t.props["__class"]; has {
					if co, ok := cls.(*jsObject); ok {
						if nm, ok2 := co.props["__name"].(string); ok2 {
							got = nm
						}
					}
				}
			}
			if want == "any" || want == "interface{}" {
				return boolH(got != "nil")
			}
			return boolH(got == want)
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
				if lo < 0 || hi > len(str) || lo > hi {
					rt.fail("slice bounds [%d:%d] out of range", lo, hi)
				}
				return w(str[lo:hi])
			}
			arr, isArr := u(a[0]).(*jsArray)
			if !isArr {
				rt.fail("slicing a %s", rt.typeOf(u(a[0])))
			}
			lo := bound(a[1], 0)
			hi := bound(a[2], len(arr.elems))
			if lo < 0 || hi > len(arr.elems) || lo > hi {
				rt.fail("slice bounds [%d:%d] out of range", lo, hi)
			}
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
		"js_goeq": func(a []uint64) uint64 { return boolH(rt.goValueEq(u(a[0]), u(a[1]))) },
		"js_gone": func(a []uint64) uint64 { return boolH(!rt.goValueEq(u(a[0]), u(a[1]))) },
		// A Go field read: the struct's own fields, then the PROMOTED fields of an
		// embedded struct, then the ordinary member lookup (methods, len, builtins).
		"js_gofield": func(a []uint64) uint64 {
			key := rt.toString(u(a[1]))
			if o, ok := u(a[0]).(*jsObject); ok {
				if v, has := o.props[key]; has {
					return w(v)
				}
				if cls, isCls := o.props["__class"].(*jsObject); isCls {
					if fs, okf := cls.props["__fields"].(*jsArray); okf {
						for _, f := range fs.elems {
							inner, isObj := o.props[rt.toString(f)].(*jsObject)
							if !isObj {
								continue
							}
							if _, embedded := inner.props["__class"]; !embedded {
								continue
							}
							if v, has := inner.props[key]; has {
								return w(v)
							}
						}
					}
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
				out := &jsArray{}
				for i, n := 0, rt.strLen(o); i < n; i++ {
					out.elems = append(out.elems, rt.strAt(o, i))
				}
				return w(out)
			}
			if keys, _, ok := dictParts(u(a[0])); ok {
				return w(&jsArray{elems: append([]interface{}{}, keys.elems...)})
			}
			rt.fail("list() of a %s", rt.typeOf(u(a[0])))
			return 0
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
				out := &jsArray{}
				for i, n := 0, rt.strLen(o); i < n; i++ {
					out.elems = append(out.elems, rt.strAt(o, i))
				}
				return w(out)
			}
			if keys, _, ok := dictParts(u(a[0])); ok {
				return w(&jsArray{elems: append([]interface{}{}, keys.elems...)})
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
		"js_pyrange": func(a []uint64) uint64 { // range(a) or range(a, b) as a materialized list.
			from := int(rt.toNumber(u(a[0])))
			to := from
			if _, undef := u(a[1]).(jsUndefT); undef {
				from = 0
			} else {
				to = int(rt.toNumber(u(a[1])))
			}
			arr := &jsArray{}
			for i := from; i < to; i++ {
				arr.elems = append(arr.elems, float64(i))
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
					return w(rt.call(conv, jsUndef, args.elems))
				}
			}
			if cls, ok := pyClassObj(callee); ok {
				inst := newJSObject()
				inst.set("__class", cls)
				if init, found := pyLookup(cls, "__init__"); found && isCallable(init) {
					self := append([]interface{}{inst}, args.elems...)
					rt.call(init, jsUndef, rt.pyBindCall(init, self, kw))
				}
				return w(inst)
			}
			if _, cls, ok := pyInstance(callee); ok {
				if m, found := pyLookup(cls, "__call__"); found && isCallable(m) {
					self := append([]interface{}{callee}, args.elems...)
					return w(rt.call(m, jsUndef, rt.pyBindCall(m, self, kw)))
				}
			}
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
				for i, n := 0, rt.strLen(o); i < n; i++ {
					arr.elems = append(arr.elems, rt.strAt(o, i))
				}
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
		// issubclass(C, B) - docs/todo.md 1.4. It was one of the item's two named
		// compiler-half gaps: this engine answered `variable not defined:
		// issubclass` where the interpreter had hostGlobals["issubclass"].
		"js_pyissubclass": func(a []uint64) uint64 {
			return boolH(rt.pyIsSubclass(u(a[0]), u(a[1])))
		},
		// A complex number: (real, imaginary) -> the {__cplx, real, imag} value the
		// complex arithmetic in js_pybin / js_pyunary and the .real / .imag attribute
		// reads understand. Only the Python compiler grammar builds this shape.
		"js_pycomplex": func(a []uint64) uint64 {
			return w(newPyComplex(rt.toNumber(u(a[0])), rt.toNumber(u(a[1]))))
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
			// ANY iterable, not just a list: set("aab") was {'a','b'} in the
			// interpreter and the EMPTY set in both compiled halves - a live
			// halves divergence, found by the docs/todo.md 1.4 sweep, that the
			// item did not list. set() with no argument is still the empty set.
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
			switch o := u(a[0]).(type) {
			case *jsArray:
				idx := rt.pySliceIndices(len(o.elems), u(a[1]), u(a[2]), u(a[3]))
				out := &jsArray{}
				for _, i := range idx {
					out.elems = append(out.elems, o.elems[i])
				}
				return w(out)
			case string:
				idx := rt.pySliceIndices(rt.strLen(o), u(a[1]), u(a[2]), u(a[3]))
				out := ""
				for _, i := range idx {
					out += rt.toString(rt.strAt(o, i))
				}
				return rt.wrapStr(out)
			}
			rt.fail("slicing a %s", rt.typeOf(u(a[0])))
			return 0
		},
		// xs[lo:hi:step] = values. A step of 1 SPLICES (the replacement may have a
		// different length, so xs[1:3] = [9] shortens the list); an extended step
		// assigns element by element and needs exactly as many values as it selects.
		"js_pysetslice": func(a []uint64) uint64 { // (list, lo, hi, step, values)
			arr, isArr := u(a[0]).(*jsArray)
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
			arr, isArr := u(a[0]).(*jsArray)
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
				// instance - keeps the old ToInt32 behaviour, after the user class
				// has had its __and__ / __lshift__ chance, which the int32 arm used
				// to swallow because it sat in front of pyDunderBin.
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
				if v, ok := rt.pyComplexBin(op, l, r); ok {
					return w(v)
				}
			}
			if v, ok := rt.pyDunderBin(op, l, r); ok {
				return w(v)
			}
			// An int operand meeting an arbitrary precision int PROMOTES, because
			// Python has one int type. Without this, 10**30 + 1 was NaN here and
			// exact in the interpreter, and so were //, % and ** on any big at all.
			if x, y, both := pyBigPair(l, r); both {
				if v, ok := pyBigBin(op, x, y); ok {
					return w(rt.pyBigNarrow(v))
				}
				// The ORDER comparisons need the same promotion: rt.jsCompare's
				// big arm is bigPair's, so a big meeting a plain int reached
				// toNumber, answered NaN and made every relation false.
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
				if pyIsFlo(l) || pyIsFlo(r) {
					return w(&jsPyFlo{f: rt.toNumber(l) + rt.toNumber(r)})
				}
				return w(rt.jsAdd(l, r))
			case "*":
				// Python's SEQUENCE REPETITION: "ab" * 3 and [0] * 3 (either way
				// round). The compiler used to fall straight through to numeric
				// arithmetic and answer NaN, while python-interpreter.abnf has
				// always repeated - a live divergence on `[0] * n`, the ordinary
				// way to build a fixed-size list.
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
				return w(&jsPyFlo{f: rt.pyToF(l) / rt.pyToF(r)})
			case "//":
				return w(pyIntOrFlo(math.Floor(rt.toNumber(l)/rt.toNumber(r)), l, r))
			case "%":
				// "fmt" % args is printf-style FORMATTING, not a remainder.
				if s, isStr := l.(string); isStr {
					return w(rt.pyPct(s, r))
				}
				return w(pyIntOrFlo(pyFloorMod(rt.toNumber(l), rt.toNumber(r)), l, r))
			case "**":
				// Python's exponentiation. Only js_pybin (the Python compilers and
				// nothing else) ever passes this operator, and a user class already
				// had its __pow__ / __rpow__ chance above.
				// int ** NEGATIVE int is a float (2 ** -1 is 0.5); every other pair
				// of ints stays an int - and a Python int is ARBITRARY PRECISION,
				// so 10**30 is 1000000000000000000000000000000, not the double
				// 1e+30 this used to fold. (The interpreter answered
				// 5076944270305264000, the int64 its tag engine wrapped the same
				// double into, so the two halves did not even agree.) Past 2^53 the
				// double has lost digits, so the exact answer comes from math/big;
				// pyBigNarrow hands back a plain number whenever it still fits, so
				// nothing below 2^53 changes shape.
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
		"js_pydict": func(a []uint64) uint64 {
			keys, vals := &jsArray{}, &jsArray{}
			if sk, sv, ok := dictParts(u(a[0])); ok {
				for i, k := range sk.elems {
					dictAppend(keys, vals, k, sv.elems[i])
				}
			}
			return w(&jsObject{props: map[string]interface{}{
				"__dict": true, "keys": keys, "vals": vals,
			}})
		},
		// max(...) / min(...): over the arguments, or over a single iterable one.
		"js_pymax": func(a []uint64) uint64 { return rt.pyMinMax(u(a[0]), 1) },
		"js_pymin": func(a []uint64) uint64 { return rt.pyMinMax(u(a[0]), -1) },
		// The context-manager protocol. A context expression that is NOT a context
		// manager (no __enter__) binds its own value and runs the body: that is the
		// approximation `with` had before the protocol existed, kept as the
		// fallback so a plain value still works instead of aborting.
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
				rt.fail("del on a %s", rt.typeOf(u(a[0])))
			}
			i := int(rt.toNumber(u(a[1])))
			if i < 0 {
				i += len(arr.elems)
			}
			if i < 0 || i >= len(arr.elems) {
				panic(&jsThrown{value: rt.pyExcInstance("IndexError", "list index out of range")})
			}
			arr.dropIdx()
			arr.elems = append(arr.elems[:i], arr.elems[i+1:]...)
			return 0
		},
		"js_pydel_attr": func(a []uint64) uint64 {
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
		// int(v): Python's truncation toward zero (a bigint stays exact).
		"js_pyint": func(a []uint64) uint64 {
			if bi, ok := u(a[0]).(*jsBigInt); ok {
				return w(bi)
			}
			return rt.wrapNum(math.Trunc(rt.toNumber(u(a[0]))))
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
		// always carries a fraction. Both were JavaScript's spelling in the
		// compiler ("1 2 3", "Infinity", "NaN") while lua-interpreter.abnf already
		// answered Lua's. Only the Lua compiler emits these two.
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
		// python-interpreter.abnf has always had and the compiler had no extern
		// for, so `bool(0)`, `float("1.5")` and `sum([1, 2])` aborted with
		// "variable not defined" on the compiler side only.
		"js_pybool": func(a []uint64) uint64 {
			v := u(a[0])
			if els, ok := pySetElems(v); ok {
				return boolH(len(els.elems) > 0)
			}
			if arr, ok := v.(*jsArray); ok {
				return boolH(len(arr.elems) > 0)
			}
			if keys, _, ok := dictParts(v); ok {
				return boolH(len(keys.elems) > 0)
			}
			return boolH(rt.truthy(v))
		},
		"js_pyfloat": func(a []uint64) uint64 {
			if s, isStr := u(a[0]).(string); isStr {
				return w(&jsPyFlo{f: pyStrToF(s)})
			}
			return w(&jsPyFlo{f: rt.toNumber(u(a[0]))})
		},
		// The float LITERAL builder: constant folding cannot carry a box, so
		// languages/python-to-llvm-ir.abnf emits this call for 1.0 and 1e3.
		"js_pyflo": func(a []uint64) uint64 {
			return w(&jsPyFlo{f: rt.toNumber(u(a[0]))})
		},
		// abs(), which used to be bound straight from JS Math and would answer
		// NaN on a box. It keeps the type: abs(-2) is the int 2, abs(-1.5) the
		// float 1.5, and abs() of an arbitrary precision int stays exact.
		"js_pyabs": func(a []uint64) uint64 {
			v := u(a[0])
			if f, ok := v.(*jsPyFlo); ok {
				return w(&jsPyFlo{f: math.Abs(f.f)})
			}
			if bi, ok := v.(*jsBigInt); ok {
				return w(&jsBigInt{v: new(big.Int).Abs(bi.v)})
			}
			return rt.wrapNum(math.Abs(rt.toNumber(v)))
		},
		// ord(c) / chr(n). They were missing from ALL THREE engines (docs/todo.md
		// 2.9 said the compiled halves had them; a probe at 114fbd5 answered
		// `variable not defined: ord` under llvm.Run too), so these land with
		// hostGlobals["ord"]/["chr"] in languages/python-interpreter.abnf,
		// js_pyord/js_pychr in languages/lib/python-rt.metajs and the
		// declBuiltin pair in languages/python-to-llvm-ir.abnf.
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
		// ---- docs/todo.md 1.4: the builtins missing from ALL THREE engines. ----
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
			if pyIterName(rt, v) != "" {
				return a[0]
			}
			return w(rt.pyMkIter(pyIterKind(v), rt.pyElemsOf(v)))
		},
		// next(it[, default]): the generator protocol, or one step of a cursor.
		// Missing HERE only - the interpreter had hostGlobals["next"] - and it is
		// the ordinary way to drive a generator, so it is docs/todo.md 1.4's
		// sharpest halves gap.
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
		// sorted(x): a NEW list, ascending, by the same comparison `<` uses.
		// CPython's key= and reverse= are KEYWORD-ONLY and a signature-less
		// builtin here receives positional arguments only, so they are refused
		// loudly rather than silently ignored.
		"js_pysorted": func(a []uint64) uint64 {
			args := rt.argArray(u(a[0]))
			if len(args) > 1 {
				rt.fail("sorted(): key= and reverse= are not supported")
			}
			out := append([]interface{}{}, rt.pyElemsOf(argAt(args, 0))...)
			sort.SliceStable(out, func(i, j int) bool { return rt.jsCompare(out[i], out[j]) == -1 })
			return w(&jsArray{elems: out})
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
		"js_pydivmod": func(a []uint64) uint64 {
			l, r := u(a[0]), u(a[1])
			if rt.toNumber(r) == 0 {
				panic(&jsThrown{value: rt.pyExcInstance("ZeroDivisionError", "integer division or modulo by zero")})
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
		"js_pysetattr3": func(a []uint64) uint64 {
			args := rt.argArray(u(a[0]))
			rt.pySetAttr(argAt(args, 0), rt.toString(argAt(args, 1)), argAt(args, 2))
			return jsHUndefined
		},
		"js_pysum": func(a []uint64) uint64 {
			v := u(a[0])
			if g, ok := v.(*jsGenerator); ok {
				v = g.drain(rt)
			}
			// An ITERATOR object (docs/todo.md 1.4) is materialized too, so
			// sum(map(f, xs)) works.
			if pyIterName(rt, v) != "" {
				v = &jsArray{elems: rt.pyElemsOf(v)}
			}
			if els, ok := pySetElems(v); ok {
				v = els
			}
			total := float64(0)
			anyFlo := false
			if arr, ok := v.(*jsArray); ok {
				for _, e := range arr.elems {
					anyFlo = anyFlo || pyIsFlo(e)
					total += rt.toNumber(e)
				}
			} else {
				rt.fail("sum() of a %s", rt.typeOf(u(a[0])))
			}
			// One float in the sequence makes the total a float: sum([1.0, 2])
			// is 3.0.
			if anyFlo {
				return w(&jsPyFlo{f: total})
			}
			return rt.wrapNum(total)
		},
		// format(v, spec, conv): the slice of Python's format mini-language an
		// f-string replacement field writes - [[fill]align][sign][0][width]
		// [.precision][type]. conv is "", "r", "s" or "a" (the !conversion).
		// Only the Python grammars ever emit this extern.
		"js_pyformat": func(a []uint64) uint64 {
			return rt.wrapStr(rt.pyFormat(u(a[0]), rt.toString(u(a[1])), rt.toString(u(a[2]))))
		},
		"js_pyprint": func(a []uint64) uint64 { // print(...) with Python style rendering.
			args, ok := u(a[0]).(*jsArray)
			if !ok {
				rt.fail("js_pyprint needs an argument array")
			}
			out := ""
			for i, e := range args.elems {
				if i > 0 {
					out += " "
				}
				out += rt.pyString(e)
			}
			fmt.Fprintln(outWriter, wtf8Clean(out))
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

// standardJSBindings are the host globals of a standalone MetaJS program (the
// same set that metajs-interpreter.abnf exposes).
func standardJSBindings() map[string]interface{} {
	mathObj := newJSObject()
	mathObj.set("imul", jsHostFunc("imul", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		a := int32(int64(rt.toNumber(argAt(args, 0))))
		b := int32(int64(rt.toNumber(argAt(args, 1))))
		return float64(a * b)
	}))
	mathObj.set("floor", jsHostFunc("floor", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return math.Floor(rt.toNumber(argAt(args, 0)))
	}))
	mathObj.set("abs", jsHostFunc("abs", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return math.Abs(rt.toNumber(argAt(args, 0)))
	}))
	// max/min are variadic like in JS: 0 arguments give -/+Infinity, and a NaN
	// argument wins (math.Max/Min propagate NaN). The old two-argument versions
	// silently ignored everything behind the second argument.
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
	// JS rounds half toward +Infinity (floor(x+0.5)), not Go's round-half-away.
	addMath1("round", func(x float64) float64 { return math.Floor(x + 0.5) })
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
	mathObj.set("hypot", jsHostFunc("hypot", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		sum := 0.0
		for _, a := range args {
			v := rt.toNumber(a)
			sum += v * v
		}
		return math.Sqrt(sum)
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
			// int(NaN) is minInt64 on amd64, which made every one-argument
			// parseInt return NaN there.
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
		"String":   stringObj,
		"Object":   objectObj,
		"Array":    arrayObj,
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
// stripping whitespace. float("inf") was nan in all three engines, and
// float("Infinity") was inf in the interpreter (whose parseFloat is goja's, which
// takes it) and nan here (jsParseFloat below does not) - a live halves divergence
// on top of the defect.
//
// Everything else still goes to jsParseFloat, so float("1.5abc") is still the
// prefix parse 1.5 where CPython raises ValueError. That looseness is
// pre-existing, is shared by all three engines, and is recorded rather than
// half-fixed here. float("-nan") is nan in CPython, so the sign is dropped there.
//
// pyStrToF in languages/python-interpreter.abnf and languages/lib/python-rt.metajs
// are the same body, with the infinity BUILT as Math.pow(2, 1024) because the
// script dialect has no exponent form in a numeric literal.
func pyStrToF(s string) float64 {
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
// __super chain is walked when present). Integral numbers count as Int AND as
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
		_, ok := v.(*jsArray)
		return ok
	}
	if o, ok := v.(*jsObject); ok {
		cls, _ := o.props["__class"].(*jsObject)
		for cls != nil {
			if n, _ := cls.props["__name"].(string); n == t {
				return true
			}
			sup, _ := cls.props["__super"].(*jsObject)
			cls = sup
		}
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
	// answered -1 here on arm64 and math.MinInt64 on amd64), and ToInt32 is a
	// MODULO rather than a range test, so neither that nor the C floor's old
	// `return 0` is the JS answer: `1e20 | 0` is 1661992960 and `1e19 | 0` is
	// -1981284352. math.Mod is fmod, which is exact for an integral operand, so
	// the reduction loses nothing. languages/lib/runtime.c's to_int32 does the
	// same thing with bit arithmetic. See docs/runtime-next-plan.md part 2.
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
// lets a Go panic through untouched. So the same program error read as two lines
// under goja and as ~4800 lines under -frozen (the entire startScript). The
// marker lets the frozen engine pass a program failure through unwrapped, so
// both engines report it identically - which is what the matrix now checks on
// stderr. Error() returns the message verbatim, so the text at the top level
// (CompileASG's fmt.Errorf("%s", err)) is exactly what it was before.
type jsProgramPanic struct{ msg string }

func (e jsProgramPanic) Error() string  { return e.msg }
func (e jsProgramPanic) String() string { return e.msg }

// runJSModule is llvm.RunJS(): it executes the entry function of a MetaJS
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
	// Everything that escapes here is re-panicked as a jsProgramPanic: the text
	// is unchanged, but the tag-script engines can now tell "the program failed"
	// from "the tag script failed" and diagnose them identically (see the type).
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		if _, ok := r.(jsProgramPanic); ok {
			panic(r)
		}
		if exc, ok := r.(*jsThrown); ok {
			panic(jsProgramPanic{"js runtime error: uncaught exception: " + rt.toString(exc.value)})
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
// Probed against the installed CPython 3.14.6:
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
	if !rt.pySetHas(els, v) {
		els.elems = append(els.elems, v)
	}
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

// pyComplexBin is + - * / == != on complex numbers, with a real operand promoted.
// ok is false for an operator complex numbers do not have (Python has no ordering
// on them either), which lets js_pybin fall through to its usual error.
func (rt *jsrt) pyComplexBin(op string, l, r interface{}) (interface{}, bool) {
	lr, li, lok := pyComplexParts(l)
	if !lok {
		lr, li = rt.toNumber(l), 0
	}
	rr, ri, rok := pyComplexParts(r)
	if !rok {
		rr, ri = rt.toNumber(r), 0
	}
	switch op {
	case "+":
		return newPyComplex(lr+rr, li+ri), true
	case "-":
		return newPyComplex(lr-rr, li-ri), true
	case "*":
		return newPyComplex(lr*rr-li*ri, lr*ri+li*rr), true
	case "/":
		d := rr*rr + ri*ri
		if d == 0 {
			return nil, false
		}
		return newPyComplex((lr*rr+li*ri)/d, (li*rr-lr*ri)/d), true
	case "==":
		return lr == rr && li == ri, true
	case "!=":
		return lr != rr || li != ri, true
	}
	return nil, false
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
		if v, ok := k.props[name]; ok {
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
		if v, ok := k.props[name]; ok {
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
	if isCallable(m) {
		return rt.pyBindMethod(s.props["__sobj"], m)
	}
	return m
}

func (rt *jsrt) pySuperMCall(s *jsObject, name string, args []interface{}, kw interface{}) interface{} {
	rt.pySuperCheck(s)
	m, found := rt.pySuperFind(s, name)
	if !found {
		panic(&jsThrown{value: rt.pyExcInstance("AttributeError",
			"'super' object has no attribute '"+name+"'")})
	}
	if !isCallable(m) {
		return rt.callPyValue(m, args, kw)
	}
	self := append([]interface{}{s.props["__sobj"]}, args...)
	return rt.call(m, jsUndef, rt.pyBindCall(m, self, kw))
}

// pyBindMethod is the bound method a METHOD read off an instance answers, so
// `f = a.m; f()` passes the instance as self. Both compiled halves used to hand
// back the raw function and died with "member 'v' of undefined" where the
// interpreter half (pyattr's bindMethod) answered - a live halves divergence,
// found by the docs/todo.md 1.9 sweep. Layer 2: pyDBindMethod.
func (rt *jsrt) pyBindMethod(self, m interface{}) interface{} {
	return jsHostFunc("pymethod", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		full := append([]interface{}{self}, args...)
		return rt.call(m, jsUndef, rt.pyBindCall(m, full, jsUndef))
	})
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

// pyGetAttr reads obj.name with Python's lookup order.
func (rt *jsrt) pyGetAttr(obj interface{}, name string) interface{} {
	if sup, isSup := pySuperObj(obj); isSup {
		return rt.pySuperAttr(sup, name)
	}
	if cls, ok := pyClassObj(obj); ok {
		switch name {
		case "__name__":
			return cls.props["__name"]
		case "__mro__":
			return cls.props["__mro"]
		case "__bases__":
			if bl, has := cls.props["__basesl"]; has {
				return bl
			}
			return &jsArray{}
		}
		if v, found := pyLookup(cls, name); found {
			return v
		}
		rt.fail("type object '%s' has no attribute '%s'", rt.toString(cls.props["__name"]), name)
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
			if get, isDesc := pyIsDescriptor(v, "__get__"); isDesc {
				return rt.call(get, jsUndef, []interface{}{v, obj, cls})
			}
			// A METHOD read off an instance comes back BOUND - see pyBindMethod.
			if isCallable(v) {
				return rt.pyBindMethod(obj, v)
			}
			return v
		}
		rt.fail("'%s' object has no attribute '%s'", rt.toString(cls.props["__name"]), name)
	}
	if name == "__name__" {
		// A synthetic class object for a BUILTIN type (what type(3) hands out)
		// carries only its name, and a closure carries none of its own - the name
		// its def bound it under was recorded by js_pyfnname.
		if o, ok := obj.(*jsObject); ok {
			if n, has := o.props["__name"]; has {
				return n
			}
		}
		if fn, ok := rt.pyFuncNames[obj]; ok {
			return fn
		}
	}
	// docs/todo.md 1.2: a BOUND built-in method, so getattr(xs, "count")(1) is
	// the 1 CPython answers rather than whatever getMember made of the name.
	if rt.pyBuiltinAttr(obj, name) {
		return rt.pyBoundBuiltin(obj, name)
	}
	// x.__class__ for a value that is not a user instance - the same synthetic
	// class object type(x) hands out, so `[].__class__ is list` holds.
	if name == "__class__" {
		return rt.pyBuiltinClass(pyTypeName(rt, obj))
	}
	return rt.getMember(obj, name)
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
	rt.setMember(obj, name, v)
}

// pySlotsAllow applies __slots__ over the WHOLE resolution order, which is
// CPython's rule and not the nearest-declaration one all three engines used to
// apply: a name is writable if ANY class on the MRO declares it in its own
// __slots__, and any name at all is writable as soon as one class on the MRO
// declares no __slots__ (that class gives its instances a __dict__). With
// `class A: __slots__ = ["s"]` and `class B(A): __slots__ = ["t"]`, writing
// B().s raised AttributeError in every engine where CPython allows it.
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
		// runtime exception's shape, so this one is rewritten here. Both engines
		// used to disagree with CPython AND with each other - this one said
		// StopIteration('') and the interpreter half StopIteration(None).
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
			return rt.pyGenValue(g.throwInto(rt, tv))
		case "close":
			// CPython's generator.close() answers None, not a {value, done}
			// record - the record is JavaScript's g.return(v).
			g.closeBody(rt)
			return jsUndef
		}
	}
	if cls, ok := pyClassObj(target); ok {
		if m, found := pyLookup(cls, name); found {
			if isCallable(m) {
				return rt.call(m, jsUndef, rt.pyBindCall(m, args, kw))
			}
			return rt.callPyValue(m, args, kw)
		}
		rt.fail("type object '%s' has no method '%s'", rt.toString(cls.props["__name"]), name)
	}
	if inst, cls, ok := pyInstance(target); ok {
		if v, found := inst.props[name]; found && isCallable(v) {
			return rt.call(v, jsUndef, rt.pyBindCall(v, args, kw)) // A plain function in an attribute.
		}
		if m, found := pyLookup(cls, name); found {
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
	// list.count(x) - docs/todo.md 1.5. rt.memberCall's `count` arm is KOTLIN's
	// `xs.count { pred }`, so `[1,2,2].count(2)` died with "call of a non
	// function value: 2" in both compiled halves. Python's counts EQUAL elements
	// (by ==, not by identity: [1.0].count(1) is 1), so it cannot share that arm
	// and is answered here, before the shared table is reached. The same arm is
	// pyDListCount in languages/lib/python-rt.metajs and the `count` arm of
	// mcall in languages/python-interpreter.abnf. str.count already existed and
	// was already CPython's non-overlapping substring count (pystrmethod.go).
	// THE WHOLE DICT AND SET METHOD SURFACE - docs/todo.md 1.3, the residue of
	// 51436f9 (which did the list surface). Only set.add and dict.pop were here,
	// and keys/values/get reached rt.memberCall's shared dict arm;
	// clear/copy/setdefault/update and pop/clear/copy/remove/discard/union all
	// aborted in every engine. The twins are pySetMethod/pyDictMethod in
	// languages/lib/python-rt.metajs and in languages/python-interpreter.abnf.
	//
	// A name the OTHER type owns raises AttributeError, which is what CPython
	// does - and the two "only" lists are narrower than the two method sets,
	// because pop/clear/copy/update are on BOTH types. `set().keys()` used to
	// answer the member LIST in the interpreter half and abort with "unknown
	// method 'keys'" here - a live halves divergence the 1.3 sweep found.
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
	// THE WHOLE LIST METHOD SURFACE - docs/todo.md 1.3. `count` was the only one
	// 6eec533 added; sort/insert/remove/extend/index/reverse/clear/copy all
	// aborted with "unknown list method" in every engine, and `pop(i)` reached
	// rt.memberCall's shared arm, which IGNORES the index and always removed the
	// LAST element ([3,1,2].pop(0) answered 2 where CPython answers 3). Answered
	// here, before the shared table, because rt.memberCall is nine languages'
	// and its `count`/`add`/`get` arms are Kotlin's. The twins are pyListMethod
	// in languages/lib/python-rt.metajs and pyListMethod in
	// languages/python-interpreter.abnf.
	if arr, isArr := target.(*jsArray); isArr && pyIsListMethod(name) {
		return rt.pyListMethod(arr, name, args, kw)
	}
	// docs/todo.md 5.2 and 2.4. rt.memberCall is the SHARED member-call table
	// that nine languages land in, and sixteen of its names are ones Python
	// cannot utter: `xs.sumOf(f)` answered 6 and `xs.size()` answered 2 in both
	// compiled halves where CPython raises AttributeError. They are denied HERE,
	// in python's own dispatcher and after the user-class arms above (so a Python
	// class that DEFINES a method called forEach is unaffected), rather than in
	// the shared table - isEmpty and removeLast are real Swift and Dart methods
	// and stay there and in languages/lib/{swift,dart}-rt.metajs.
	//
	// The denial is keyed by RECEIVER TYPE and not by name alone, which the
	// four-name version did not have to be: `get` is a real dict method
	// (d.get(k, default)) and `add` is a real set method, so a flat name switch
	// would have broken both. The same table is pyDForeign in
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

// pyIsListMethod is the NAME SET of Python's list methods - docs/todo.md 1.3.
// It is a name test only, so hasattr()/getattr() can ask the same question the
// dispatcher answers. Keep in step with pyIsListMethod in
// languages/lib/python-rt.metajs and in languages/python-interpreter.abnf.
func pyIsListMethod(name string) bool {
	switch name {
	case "append", "pop", "clear", "copy", "count",
		"sort", "insert", "remove", "extend", "index", "reverse":
		return true
	}
	return false
}

// pyListMethod runs one of them. Every error it raises is a CATCHABLE Python
// exception with CPython 3.14's exact text, not rt.fail: `try: xs.remove(9) /
// except ValueError` has to behave the same in all three engines, which is the
// only way any of this is assertable in one file (the reason 6eec533 gives for
// AttributeError).
func (rt *jsrt) pyListMethod(arr *jsArray, name string, args []interface{}, kw interface{}) interface{} {
	n := len(arr.elems)
	switch name {
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
// with an inverted comparison would not.
func (rt *jsrt) pySortList(arr *jsArray, args []interface{}, kw interface{}) {
	if len(args) > 0 {
		panic(&jsThrown{value: rt.pyExcInstance("TypeError",
			"sort() takes no positional arguments")})
	}
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
		if isCallable(key) {
			k = rt.call(key, jsUndef, []interface{}{e})
		}
		pairs[i] = pySortPair{k: k, v: e}
	}
	// The same comparison js_pysorted uses, so sort() and sorted() can never
	// disagree inside one engine.
	sort.SliceStable(pairs, func(i, j int) bool { return rt.jsCompare(pairs[i].k, pairs[j].k) == -1 })
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

// ----- Python dict and set methods - docs/todo.md 1.3 -----
//
// The NAME SETS first, so hasattr()/getattr() can ask exactly what the
// dispatcher answers. Keep in step with pyIsDictMethod/pyIsSetMethod in
// languages/lib/python-rt.metajs and in languages/python-interpreter.abnf.
//
// NOT the whole of CPython's: dict.popitem/fromkeys and set.intersection/
// difference/symmetric_difference/isdisjoint/issubset/issuperset and their
// *_update forms are absent, hasattr() therefore says False for them, and a
// written call still ABORTS rather than raising - the 51436f9 rule. The set
// ALGEBRA is spelled with the operators & - ^ <= >= instead, and those are
// complete.
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
	case "add", "pop", "clear", "copy", "remove", "discard", "union", "update":
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
	case "add", "remove", "discard", "union":
		return true
	}
	return false
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
		// of TUPLES, which is docs/todo.md 3.1 and not this item). The compiler
		// grammar USED to lower a written d.items() into a loop of its own and
		// never reach here, so getattr(d, "items")() aborted; that lowering is
		// gone and every form arrives here.
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
		// has no tuple (docs/todo.md 3.1). Empty raises the KeyError whose
		// argument is a sentence rather than a key, which c2b4071's arg-carrying
		// KeyError renders as CPython renders it.
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
		// tell that from a real argument - docs/todo.md 2.13), so
		// getattr(d, "update")(a=1) is not d.update(a=1); the written call is.
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
	}
	rt.fail("unknown set method '%s'", name)
	return nil
}

// pyBuiltinAttr - docs/todo.md 1.2. hasattr()/getattr() walked only the
// user-class MRO and an object's own property map, so hasattr([3,1,2], "count")
// was False and hasattr("s", "upper") was False where CPython answers True in
// both. This is the same question the DISPATCHER answers, asked by name: the
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
	return jsHostFunc("pybound."+name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
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
			return rt.call(conv, jsUndef, args)
		}
	}
	if cls, ok := pyClassObj(v); ok {
		inst := newJSObject()
		inst.set("__class", cls)
		if init, found := pyLookup(cls, "__init__"); found && isCallable(init) {
			self := append([]interface{}{inst}, args...)
			rt.call(init, jsUndef, rt.pyBindCall(init, self, kw))
		}
		return inst
	}
	if _, cls, ok := pyInstance(v); ok {
		if m, found := pyLookup(cls, "__call__"); found && isCallable(m) {
			self := append([]interface{}{v}, args...)
			return rt.call(m, jsUndef, rt.pyBindCall(m, self, kw))
		}
	}
	return rt.call(v, jsUndef, rt.pyBindCall(v, args, kw))
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

// ----- the Python iterator object (docs/todo.md 1.4) ------------------------
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
// GENERATOR ITSELF - CPython's rule too - so the lazy case that docs/todo.md 1.7
// fixed (`for x in endless(): break`) is untouched. The same three functions are
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
		var out []interface{}
		for i, n := 0, rt.strLen(o); i < n; i++ {
			out = append(out, rt.strAt(o, i))
		}
		return out
	}
	if keys, _, ok := dictParts(v); ok {
		return append([]interface{}{}, keys.elems...)
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

// pyIterKind is the CPython name of the iterator iter() answers for a value.
func pyIterKind(v interface{}) string {
	switch v.(type) {
	case string:
		return "str_iterator"
	case *jsArray:
		return "list_iterator"
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
		if name == "__name__" || name == "__mro__" || name == "__bases__" {
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
	// docs/todo.md 1.2: the BUILT-IN methods of str / list / dict / set, and
	// __class__, which EVERY value has (None.__class__ is NoneType).
	if name == "__class__" || rt.pyBuiltinAttr(obj, name) {
		return true
	}
	if name == "__name__" {
		if o, ok := obj.(*jsObject); ok {
			if _, has := o.props["__name"]; has {
				return true
			}
		}
		if _, ok := rt.pyFuncNames[obj]; ok {
			return true
		}
	}
	if o, ok := obj.(*jsObject); ok {
		_, has := o.props[name]
		return has
	}
	return false
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
// Both were wrong here, and in two different ways: `isinstance(1, object)` was
// True in the interpreter and FALSE under llvm.Run (a live halves divergence,
// found by the docs/todo.md 1.4 sweep and not listed there), while
// `isinstance(True, int)` and `issubclass(bool, int)` were False in EVERY
// engine, which is the defect class byte-identity cannot see. bool subclassing
// int is not a curiosity - it is why True + 1 is 2 and why {True: "t"}[1] hits.
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
// a class object. It was missing from BOTH compiled engines - `variable not
// defined: issubclass` - while the interpreter had it, so it was one of
// docs/todo.md 1.4's two named halves gaps. Keep in step with pyDIsSubclass in
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
		if _, isSet := pySetElems(t); isSet {
			return "set"
		}
		if _, _, isDict := dictParts(t); isDict {
			return "dict"
		}
		return "object"
	case *jsClosure, *hostFunc, *boundMethod:
		return "function"
	}
	if _, _, ok := dictParts(v); ok {
		return "dict"
	}
	return "object"
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
			return rt.call(m, jsUndef, []interface{}{l, r}), true
		}
	}
	if _, cls, ok := pyInstance(r); ok {
		if m, found := pyLookup(cls, pyBinReflected[op]); found && isCallable(m) {
			return rt.call(m, jsUndef, []interface{}{r, l}), true
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
//   A  [p0 .. pn-1, extra positionals ...]   - the layout every def had before
//      keyword parameters existed; a *args rest parameter reads the extras with
//      js_pyrest(args, n).
//   B  [p0 .. pn-1, *args list, **kwargs dict]  - used as soon as the def has
//      keyword-only parameters or a **kwargs one, because those cannot be
//      recovered from a flat positional array.
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
// same length - `def f(a, *rest, **kw)` called `f(1, 2, 3)`. Passing the
// original array there bound `rest` to a number and `kw` to another, and the
// first `len(rest)` aborted.
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

// pyMinMax is the max()/min() builtin: over the call arguments, or over the
// elements of a single iterable argument. want is +1 for max, -1 for min.
func (rt *jsrt) pyMinMax(argsV interface{}, want int) uint64 {
	args, ok := argsV.(*jsArray)
	if !ok {
		rt.fail("max/min needs an argument array")
	}
	elems := args.elems
	if len(elems) == 1 {
		if inner, ok := elems[0].(*jsArray); ok {
			elems = inner.elems
		} else if pyIterName(rt, elems[0]) != "" || pyIsGenerator(elems[0]) {
			elems = rt.pyElemsOf(elems[0]) // max(map(f, xs)), max(iter(xs)).
		}
	}
	if len(elems) == 0 {
		rt.fail("max/min of an empty sequence")
	}
	best := elems[0]
	for _, e := range elems[1:] {
		if rt.jsCompare(e, best) == want {
			best = e
		}
	}
	return rt.wrap(best)
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
// KeyError.__str__ is repr(args[0]), so every engine here used to pre-repr the
// argument, which made str(e) right by accident and left repr(e) doubly quoted -
// KeyError('3') where CPython says KeyError(3) - and e.args[0] a string where
// CPython has the key (docs/todo.md 1.7). The repr now happens in pyUserRender.
func (rt *jsrt) pyExcInstanceV(name string, arg interface{}) interface{} {
	inst := newJSObject()
	inst.set("__class", rt.pyExcClass(name))
	inst.set("args", &jsArray{elems: []interface{}{arg}})
	return inst
}

// pyExcParentName is CPython's hierarchy above a RUNTIME-raised exception, which
// used to be a flat derivation from Exception: rt.pyIsInstance walks the __super
// chain by name, so a KeyError whose __super was Exception was not caught by
// `except LookupError` even though the interpreter half caught it
// (docs/todo.md 1.7). pyExcParent in languages/python-interpreter.abnf and the
// excParent block of languages/python-to-llvm-ir.abnf are the same table.
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


// ----- Python's format mini-language (js_pyformat) -----
// [[fill]align][sign][0][width][.precision][type], which is the slice of it an
// f-string replacement field in practice writes.
func pyIsAlign(ch byte) bool { return ch == '<' || ch == '>' || ch == '^' || ch == '=' }

// ----- "fmt" % args, CPython's printf-style formatting -----
//
// UNIMPLEMENTED in both halves until now: js_pybin's "%" on a string fell into
// pyFloorMod and answered NaN. The two halves agreed, which is why --cross was
// blind to it and only a CPython diff saw it.
//
// Supported: the conversions s r a d i u x X o b c e E g G f F and %%; the flags
// - + 0 and space; a width; and a .precision, which TRUNCATES for s/r/a. A
// %(name)s mapping key reads a dict argument. NOT supported, and a LOUD failure
// rather than a silent wrong answer: the # alternate form.
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

// pyPctNum is a conversion operand as a double, unwrapping the float box and an
// arbitrary precision int.
func (rt *jsrt) pyPctNum(v interface{}) float64 {
	if bi, ok := v.(*jsBigInt); ok {
		f, _ := new(big.Float).SetInt(bi.v).Float64()
		return f
	}
	return rt.toNumber(v)
}

// pyPctDigits is %d: the integer part, and an arbitrary precision int stays exact.
func (rt *jsrt) pyPctDigits(v interface{}) string {
	if bi, ok := v.(*jsBigInt); ok {
		return bi.v.String()
	}
	return strconv.FormatInt(int64(math.Trunc(rt.pyPctNum(v))), 10)
}

// pyPctRadix is the base-N digits of an integral value: exact for every magnitude
// a double holds exactly, and the SAME loop in all three halves (pyBaseStr in
// languages/python-interpreter.abnf, pyBPctRadix in languages/lib/python-rt.metajs
// - neither of which has an int64 to convert through, and int64() SATURATES on
// arm64, so a shared loop over the double is the only spelling that agrees).
func (rt *jsrt) pyPctRadix(v interface{}, base int, upper bool) string {
	n := math.Trunc(rt.pyPctNum(v))
	if math.IsNaN(n) {
		n = 0
	}
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	out := ""
	fb := float64(base)
	for n > 0 {
		q := math.Floor(n / fb)
		out = string("0123456789abcdefghijklmnopqrstuvwxyz"[int(n-q*fb)]) + out
		n = q
	}
	if upper {
		out = strings.ToUpper(out)
	}
	if neg {
		return "-" + out
	}
	return out
}

func (rt *jsrt) pyPctOne(v interface{}, typ byte, left, zero, plus, space bool, width, prec int) string {
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
		if s, isStr := v.(string); isStr {
			body = s
		} else {
			body = string(rune(int(math.Trunc(rt.pyPctNum(v)))))
		}
	case 'd', 'i', 'u':
		numeric = true
		body = rt.pyPctDigits(v)
	case 'x':
		numeric, body = true, rt.pyPctRadix(v, 16, false)
	case 'X':
		numeric, body = true, rt.pyPctRadix(v, 16, true)
	case 'o':
		numeric, body = true, rt.pyPctRadix(v, 8, false)
	case 'b':
		numeric, body = true, rt.pyPctRadix(v, 2, false)
	case 'f', 'F':
		p := prec
		if p < 0 {
			p = 6
		}
		numeric, body = true, strconv.FormatFloat(rt.pyPctNum(v), 'f', p, 64)
	case 'e', 'E', 'g', 'G':
		numeric, body = true, pyPctExp(rt.pyPctNum(v), typ, prec)
	default:
		rt.fail("unsupported format character '%c'", typ)
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

// pyPctExp is %e / %g. strconv's 'e' and 'g' ARE C's shapes - an exponent that is
// signed and at least two digits, and 'g' choosing between the two on the exponent
// that was used - so this half needs no digit machinery of its own; the two script
// halves build the same text out of floPrec.
func pyPctExp(n float64, typ byte, prec int) string {
	upper := typ == 'E' || typ == 'G'
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
			rt.fail("incomplete format")
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
		left, zero, plus, space := false, false, false, false
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
				rt.fail("unsupported format flag '#'")
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
			rt.fail("incomplete format")
		}
		typ := f[i]
		i++
		var v interface{}
		if hasKey {
			if dictK == nil {
				rt.fail("format requires a mapping")
			}
			at := rt.pyDictFind(dictK, key)
			if at < 0 {
				panic(&jsThrown{value: rt.pyExcInstanceV("KeyError", key)})
			}
			v = dictV.elems[at]
		} else {
			if ai >= len(args) {
				rt.fail("not enough arguments for format string")
			}
			v = args[ai]
			ai++
		}
		out += rt.pyPctOne(v, typ, left, zero, plus, space, width, prec)
	}
	if !mapped && ai < len(args) {
		rt.fail("not all arguments converted during string formatting")
	}
	return out
}

func (rt *jsrt) pyFormat(v interface{}, spec, conv string) string {
	if spec == "" {
		if conv == "r" {
			return rt.pyRepr(v)
		}
		return rt.pyString(v)
	}
	i := 0
	fill := " "
	align := byte(0)
	if len(spec) > 1 && pyIsAlign(spec[1]) {
		fill, align, i = spec[0:1], spec[1], 2
	} else if len(spec) > 0 && pyIsAlign(spec[0]) {
		align, i = spec[0], 1
	}
	sign := byte(0)
	if i < len(spec) && (spec[i] == '+' || spec[i] == '-' || spec[i] == ' ') {
		sign, i = spec[i], i+1
	}
	if i < len(spec) && spec[i] == '0' && align == 0 {
		fill, align, i = "0", '=', i+1
	}
	width := 0
	for i < len(spec) && spec[i] >= '0' && spec[i] <= '9' {
		width, i = width*10+int(spec[i]-'0'), i+1
	}
	prec := -1
	if i < len(spec) && spec[i] == '.' {
		i++
		prec = 0
		for i < len(spec) && spec[i] >= '0' && spec[i] <= '9' {
			prec, i = prec*10+int(spec[i]-'0'), i+1
		}
	}
	typ := byte(0)
	if i < len(spec) {
		typ = spec[i]
	}
	body := rt.pyFormatBody(v, typ, prec, sign, conv)
	pad := width - rt.strLen(body)
	if pad <= 0 {
		return body
	}
	_, isNum := v.(float64)
	isNum = isNum || pyIsFlo(v)
	// A number right-aligns by default, everything else left-aligns.
	if align == '>' || align == '=' || (align == 0 && isNum) {
		return strings.Repeat(fill, pad) + body
	}
	if align == '^' {
		left := pad / 2
		return strings.Repeat(fill, left) + body + strings.Repeat(fill, pad-left)
	}
	return body + strings.Repeat(fill, pad)
}

func (rt *jsrt) pyFormatBody(v interface{}, typ byte, prec int, sign byte, conv string) string {
	var body string
	switch typ {
	case 'f', 'F':
		p := prec
		if p < 0 {
			p = 6
		}
		body = strconv.FormatFloat(rt.toNumber(v), 'f', p, 64)
	case 'd':
		body = strconv.FormatInt(int64(rt.toInt32(v)), 10)
	case 'x':
		body = strconv.FormatInt(int64(rt.toInt32(v)), 16)
	case 'X':
		body = strings.ToUpper(strconv.FormatInt(int64(rt.toInt32(v)), 16))
	case 'b':
		body = strconv.FormatInt(int64(rt.toInt32(v)), 2)
	case 'o':
		body = strconv.FormatInt(int64(rt.toInt32(v)), 8)
	default:
		if conv == "r" {
			body = rt.pyRepr(v)
		} else {
			body = rt.pyString(v)
		}
	}
	if sign == '+' {
		if _, isNum := v.(float64); isNum || pyIsFlo(v) {
			if rt.toNumber(v) >= 0 {
				body = "+" + body
			}
		}
	}
	return body
}

// pyStrLen counts Unicode CODE POINTS: Python's len() of a string, where a
// surrogate pair is one character. Only the Python grammars ask for this.
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
