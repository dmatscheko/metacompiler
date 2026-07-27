package abnf

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"os"
	"reflect"
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

// finish abandons the body without resuming it (generator.return(v)).
func (g *jsGenerator) finish(v interface{}) interface{} {
	g.done = true
	g.retValue = v
	g.lastValue = jsUndef
	g.lastKey = jsUndef
	res := newJSObject()
	res.set("value", v)
	res.set("done", true)
	return res
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
	// this program can ever ask for a dynamic `this` (see trackThis).
	for _, f := range m.Funcs {
		if f.GlobalIdent.Name() == "js_this" || f.GlobalIdent.Name() == "js_newtarget" {
			rt.trackThis = true
			break
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

// goValueEq is Go's ==: structs and arrays are VALUE types and compare element by
// element, nil equals an unset slice/map/pointer, everything else is strictEq.
// Used only by the go-to-llvm-ir grammar (js_goeq / js_gone).
func (rt *jsrt) goValueEq(a, b interface{}) bool {
	if isUndefOrNull(a) {
		return isUndefOrNull(b)
	}
	if isUndefOrNull(b) {
		return false
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

func (rt *jsrt) getMember(obj interface{}, key interface{}) interface{} {
	if isUndefOrNull(obj) {
		rt.fail("member '%s' of %s", rt.toString(key), rt.toString(obj))
	}
	// Function values understand apply and call like in JS.
	if ks, isStr := key.(string); isStr && (ks == "apply" || ks == "call") && isCallable(obj) {
		return &boundMethod{recv: obj, name: ks}
	}
	switch o := obj.(type) {
	case *jsObject:
		name := rt.toString(key)
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
				return o.finish(argAt(args, 0))
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
	rt.fail("cannot set member '%s' on %s", name, rv.Type())
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

func (rt *jsrt) dictScan(keys *jsArray, k interface{}) int {
	for i, e := range keys.elems {
		if rt.strictEq(e, k) {
			return i
		}
	}
	return -1
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

// pyRepr renders a container element like Python's repr: strings get quotes.
func (rt *jsrt) pyRepr(v interface{}) string {
	if s, isStr := v.(string); isStr {
		return "'" + s + "'"
	}
	return rt.pyString(v)
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
		return rt.toString(v)
	default:
		return rt.toString(v)
	}
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
	}
	return 0
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
// value is whole (1000.0), which is what tells it apart from the Integer 1000.
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
	if x == math.Trunc(x) && math.Abs(x) < 1e16 {
		return jsNumString(x) + ".0"
	}
	return jsNumString(x)
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
				if i := rt.dictFind(keys, argAt(args, 0)); i >= 0 {
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
			return jsFlo{f: x - math.Floor(x/y)*y}
		case "**":
			return jsFlo{f: math.Pow(x, y)}
		}
		rt.fail("Float does not support '%s'", op)
	}
	x, y := rubyToF(l), rubyToF(r)
	switch op {
	case "+":
		return x + y
	case "-":
		return x - y
	case "*":
		return x * y
	case "/":
		if y == 0 {
			rt.fail("divided by 0")
		}
		return math.Floor(x / y)
	case "%":
		if y == 0 {
			rt.fail("divided by 0")
		}
		return x - math.Floor(x/y)*y
	case "**":
		if y < 0 {
			return jsFlo{f: math.Pow(x, y)}
		}
		return math.Pow(x, y)
	case "&":
		return float64(int64(x) & int64(y))
	case "|":
		return float64(int64(x) | int64(y))
	case "^":
		return float64(int64(x) ^ int64(y))
	case "<<":
		return x * math.Pow(2, y)
	case ">>":
		return math.Floor(x / math.Pow(2, y))
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
			out := &jsArray{}
			for i := 0; i < int(rubyToF(r)); i++ {
				out.elems = append(out.elems, la.elems...)
			}
			return out
		}
	case "-":
		if la, ok := l.(*jsArray); ok {
			if ra, ok2 := r.(*jsArray); ok2 {
				out := &jsArray{}
				for _, e := range la.elems {
					drop := false
					for _, x := range ra.elems {
						if rt.pyEqual(e, x) {
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
	x := rubyToF(t)
	_, isInt := t.(float64)
	switch name {
	case "to_s", "inspect":
		if isInt && len(args) > 0 {
			return strconv.FormatInt(int64(x), int(rubyToF(args[0]))), true
		}
		return rt.rubyStr(t), true
	case "to_i", "to_int", "truncate":
		return math.Trunc(x), true
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
		return math.Floor(x), true
	case "ceil":
		return math.Ceil(x), true
	case "round":
		if len(args) > 0 {
			p := math.Pow(10, rubyToF(args[0]))
			return jsFlo{f: math.Floor(x*p+0.5) / p}, true
		}
		if x < 0 {
			return -math.Floor(-x + 0.5), true
		}
		return math.Floor(x + 0.5), true
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
		return x + 1, true
	case "pred":
		return x - 1, true
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
		y := rubyToF(argAt(args, 0))
		return &jsArray{elems: []interface{}{math.Floor(x / y), x - math.Floor(x/y)*y}}, true
	case "fdiv":
		return jsFlo{f: x / rubyToF(argAt(args, 0))}, true
	case "pow", "**":
		return rt.rubyBin("**", t, argAt(args, 0)), true
	case "gcd":
		return rubyGcd(x, rubyToF(argAt(args, 0))), true
	case "between?":
		return x >= rubyToF(argAt(args, 0)) && x <= rubyToF(argAt(args, 1)), true
	case "clamp":
		lo, hi := rubyToF(argAt(args, 0)), rubyToF(argAt(args, 1))
		if x < lo {
			return args[0], true
		}
		if x > hi {
			return args[1], true
		}
		return t, true
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
func (rt *jsrt) rubyStr(v interface{}) string {
	switch t := v.(type) {
	case jsNullT, jsUndefT:
		return ""
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

// rubyInspect is Ruby's #inspect: nil, quoted strings, :symbols, and the bracketed
// forms of Array and Hash.
func (rt *jsrt) rubyInspect(v interface{}) string {
	switch t := v.(type) {
	case jsNullT, jsUndefT:
		return "nil"
	case string:
		return "\"" + t + "\""
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
				parts[i] = rt.rubyInspect(keys.elems[i]) + " => " + rt.rubyInspect(vals.elems[i])
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
		return rubyToF(l) == rubyToF(r)
	}
	if rubyUserObj(l) {
		if m, ok := rubyFindMethod(l, "=="); ok {
			return rubyTruthy(rt.call(m, jsUndef, []interface{}{l, r}))
		}
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
	return rt.pyEqual(l, r)
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
			for i := 0; i < len(la.elems) && i < len(ra.elems); i++ {
				if c, ok3 := rt.rubySpaceship(la.elems[i], ra.elems[i]); ok3 && c != 0 {
					return c, true
				}
			}
			return len(la.elems) - len(ra.elems), true
		}
	}
	return 0, false
}

// rubyCmp backs < > <= >=, which raise in Ruby when the values cannot be compared.
func (rt *jsrt) rubyCmp(l, r interface{}) int {
	c, ok := rt.rubySpaceship(l, r)
	if !ok {
		rt.fail("comparison of incompatible values")
	}
	return c
}

// rubyFormat is Kernel#format / String#% - the printf directives Ruby shares with C:
// %[-][0][width][.prec](d|i|f|s|x|X|o|b|e|%). The right operand is the argument, or
// an array of them.
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
		left, zero := false, false
		for i < len(spec) && (spec[i] == '-' || spec[i] == '0' || spec[i] == '+' || spec[i] == ' ') {
			if spec[i] == '-' {
				left = true
			}
			if spec[i] == '0' {
				zero = true
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
		switch spec[i] {
		case 'd', 'i':
			body = jsNumString(math.Trunc(rubyToF(next())))
		case 'f':
			if prec < 0 {
				prec = 6
			}
			body = strconv.FormatFloat(rubyToF(next()), 'f', prec, 64)
		case 'e':
			if prec < 0 {
				prec = 6
			}
			body = strconv.FormatFloat(rubyToF(next()), 'e', prec, 64)
		case 's':
			body = rt.rubyStr(next())
			if prec >= 0 && prec < len(body) {
				body = body[:prec]
			}
		case 'x':
			body = strconv.FormatInt(int64(rubyToF(next())), 16)
		case 'X':
			body = strings.ToUpper(strconv.FormatInt(int64(rubyToF(next())), 16))
		case 'o':
			body = strconv.FormatInt(int64(rubyToF(next())), 8)
		case 'b':
			body = strconv.FormatInt(int64(rubyToF(next())), 2)
		default:
			body = string(spec[i])
		}
		pad := " "
		if zero && !left {
			pad = "0"
		}
		for len(body) < width {
			if left {
				body += " "
			} else if pad == "0" && len(body) > 0 && (body[0] == '-') {
				body = "-" + pad + body[1:]
			} else {
				body = pad + body
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
	case "to_proc":
		if isCallable(target) {
			return target
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
		case "to_sym", "intern":
			return jsSym{s: o}
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
			case "size", "length":
				return float64(len(keys.elems))
			case "keys":
				return &jsArray{elems: append([]interface{}{}, keys.elems...)}
			case "values":
				return &jsArray{elems: append([]interface{}{}, vals.elems...)}
			case "include?", "has_key?", "key?":
				return rt.dictFind(keys, argAt(args, 0)) >= 0
			case "each":
				for i := range keys.elems {
					rt.call(argAt(args, 0), jsUndef, []interface{}{keys.elems[i], vals.elems[i]})
				}
				return o
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
		}
		// A class instance or class object: the generic dispatch handles it.
		return rt.memberCall(target, name, args)
	}
	if isUndefOrNull(target) {
		rt.fail("method .%s on nil", name)
	}
	return rt.memberCall(target, name, args)
}

// rubyArrayMethod mirrors the arrayMethod of ruby-interpreter.abnf. select and
// reject use rubyTruthy (Ruby semantics), and pop/first/last return nil (not an
// error) on an empty array.
func (rt *jsrt) rubyArrayMethod(t *jsArray, name string, args []interface{}) interface{} {
	switch name {
	case "size", "length":
		return float64(len(t.elems))
	case "push", "append", "add":
		t.dropIdx()
		t.elems = append(t.elems, argAt(args, 0))
		return t
	case "pop":
		if len(t.elems) == 0 {
			return jsNull
		}
		v := t.elems[len(t.elems)-1]
		t.dropIdx()
		t.elems = t.elems[:len(t.elems)-1]
		return v
	case "first":
		if len(t.elems) == 0 {
			return jsNull
		}
		return t.elems[0]
	case "last":
		if len(t.elems) == 0 {
			return jsNull
		}
		return t.elems[len(t.elems)-1]
	case "include?", "contains":
		return rt.dictScan(t, argAt(args, 0)) >= 0 // A list, so a scan and no index.
	case "to_a":
		return &jsArray{elems: append([]interface{}{}, t.elems...)}
	case "each":
		for _, e := range t.elems {
			rt.call(argAt(args, 0), jsUndef, []interface{}{e})
		}
		return t
	case "each_with_index":
		for i, e := range t.elems {
			rt.call(argAt(args, 0), jsUndef, []interface{}{e, float64(i)})
		}
		return t
	case "map", "collect":
		out := &jsArray{}
		for _, e := range t.elems {
			out.elems = append(out.elems, rt.call(argAt(args, 0), jsUndef, []interface{}{e}))
		}
		return out
	case "select", "filter":
		out := &jsArray{}
		for _, e := range t.elems {
			if rubyTruthy(rt.call(argAt(args, 0), jsUndef, []interface{}{e})) {
				out.elems = append(out.elems, e)
			}
		}
		return out
	case "reject":
		out := &jsArray{}
		for _, e := range t.elems {
			if !rubyTruthy(rt.call(argAt(args, 0), jsUndef, []interface{}{e})) {
				out.elems = append(out.elems, e)
			}
		}
		return out
	case "sum":
		var s int32 = 0
		for _, e := range t.elems {
			s = rt.toInt32(float64(s) + rt.toNumber(e))
		}
		return float64(s)
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
				i := rt.dictFind(keys, u(a[1]))
				if i < 0 {
					return w(jsNull)
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
				if i := rt.dictFind(keys, u(a[1])); i >= 0 {
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
			return w(<-g.resume)
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
		"js_rbnot": func(a []uint64) uint64 { return w(-rubyToF(u(a[0])) - 1) },
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
			if !ok || len(arr.elems) == 0 {
				return rt.wrapStr("")
			}
			return rt.wrapStr(rt.rubyFormat(rt.rubyStr(arr.elems[0]), &jsArray{elems: arr.elems[1:]}))
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
			if keys, vals, ok := dictParts(u(a[0])); ok {
				i := rt.dictFind(keys, u(a[1]))
				if i < 0 {
					rt.fail("KeyError: %s", rt.pyString(u(a[1])))
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
					rt.fail("list index out of range: %d", int(rt.toNumber(u(a[1]))))
				}
				return w(o.elems[idx])
			case string:
				n := rt.strLen(o)
				if idx < 0 {
					idx += n
				}
				if idx < 0 || idx >= n {
					rt.fail("string index out of range: %d", int(rt.toNumber(u(a[1]))))
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
				if i := rt.dictFind(keys, u(a[1])); i >= 0 {
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
			case "float32", "float64":
				return rt.wrapNum(rt.toNumber(v))
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
			rt.fail("len() of a %s", rt.typeOf(u(a[0])))
			return 0
		},
		"js_pylist": func(a []uint64) uint64 { // list(x): always a fresh list.
			if g, ok := u(a[0]).(*jsGenerator); ok {
				return w(g.drain(rt))
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
				return boolH(rt.dictFind(keys, u(a[0])) >= 0)
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
			mro := &jsArray{elems: []interface{}{cls}}
			for _, c := range pyLinearize(baseObjs) {
				mro.elems = append(mro.elems, c)
			}
			cls.set("__mro", mro)
			return w(cls)
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
			bound := rt.pyBindCall(callee, args.elems, kw)
			if len(bound) == len(args.elems) {
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
				if j := rt.dictFind(keys, k); j >= 0 {
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
			if src, isArr := u(a[0]).(*jsArray); isArr {
				for _, e := range src.elems {
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
			case "|":
				return rt.wrapNum(float64(rt.toInt32(l) | rt.toInt32(r)))
			case "&":
				return rt.wrapNum(float64(rt.toInt32(l) & rt.toInt32(r)))
			case "^":
				return rt.wrapNum(float64(rt.toInt32(l) ^ rt.toInt32(r)))
			case "<<":
				return rt.wrapNum(float64(rt.toInt32(l) << (uint32(rt.toInt32(r)) & 31)))
			case ">>":
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
				return w(rt.jsAdd(l, r))
			case "-", "*", "/":
				return w(rt.pyArith(op[0], l, r))
			case "//":
				return rt.wrapNum(math.Floor(rt.toNumber(l) / rt.toNumber(r)))
			case "%":
				return rt.wrapNum(pyFloorMod(rt.toNumber(l), rt.toNumber(r)))
			case "**":
				// Python's exponentiation. Only js_pybin (the Python compilers and
				// nothing else) ever passes this operator, and a user class already
				// had its __pow__ / __rpow__ chance above.
				return rt.wrapNum(math.Pow(rt.toNumber(l), rt.toNumber(r)))
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
				i := rt.dictFind(keys, u(a[1]))
				if i < 0 {
					panic(&jsThrown{value: rt.pyExcInstance("KeyError", rt.pyString(u(a[1])))})
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
				if rt.dictFind(keys, k) < 0 {
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
		"js_pyrepr": func(a []uint64) uint64 {
			return rt.wrapStr(rt.pyRepr(u(a[0])))
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
		"printf": jsHostFunc("printf", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			if len(args) == 0 {
				return jsUndef
			}
			fmt.Fprintf(outWriter, rt.toString(args[0]), rt.printArgs(args[1:])...)
			return jsUndef
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
	return int32(int64(f))
}

// runJSModule is llvm.RunJS(): it executes the entry function of a MetaJS
// module with the standard host bindings and returns its int32 result.
// This is the program runtime, so the -cfgraph and -trace hooks live here.
func runJSModule(m *ir.Module, entry string) *RunResult {
	maybeDumpCFG(m)
	maybeDumpCallgraph(m)
	rt := newJSRT(standardJSBindings())
	rt.enableTrace()
	ma := rt.attach(m)
	// An exception that escapes the program's entry point is an uncaught throw;
	// report it like any other runtime error (rt.fail panics a string that the
	// caller's recover turns into a clean message and a non-zero exit).
	defer func() {
		if r := recover(); r != nil {
			if exc, ok := r.(*jsThrown); ok {
				rt.fail("uncaught exception: %s", rt.toString(exc.value))
			}
			panic(r)
		}
	}()
	h := rt.callEntry(ma, entry, 0)
	return &RunResult{Ret: uint32(rt.toInt32(rt.unwrap(h))), Out: ""}
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
			j := rt.dictFind(yk, k)
			if j < 0 || !rt.pyEqual(xv.elems[i], yv.elems[j]) {
				return false
			}
		}
		return true
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
	if cls, ok := pyClassObj(obj); ok {
		switch name {
		case "__name__":
			return cls.props["__name"]
		case "__mro__":
			return cls.props["__mro"]
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
		if slots, found := pyLookup(cls, "__slots__"); found {
			if !pySlotAllows(rt, slots, name) {
				panic(&jsThrown{value: rt.pyAttrError(cls, name)})
			}
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

// pySlotAllows answers whether __slots__ (a list, a tuple-as-list or a single
// string) permits an attribute name.
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
	return true
}

// pyAttrError builds the AttributeError instance a rejected __slots__ write
// raises, in the shape the except clauses match ({__class:{__name}, args}).
func (rt *jsrt) pyAttrError(cls *jsObject, name string) interface{} {
	return rt.pyExcInstance("AttributeError",
		"'"+rt.toString(cls.props["__name"])+"' object has no attribute '"+name+"'")
}

// pyMethodCall is obj.name(args) with Python's binding rule: an instance
// prepends itself, a class object does not (so Base.m(self) works).
func (rt *jsrt) pyMethodCall(target interface{}, name string, args []interface{}, kw interface{}) interface{} {
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
			st, _ := g.step(rt, sent).(*jsObject)
			if st == nil {
				rt.fail("generator step failed")
			}
			if done, _ := st.props["done"].(bool); done {
				exc := rt.pyExcInstance("StopIteration", "")
				exc.(*jsObject).set("value", st.props["value"])
				panic(&jsThrown{value: exc})
			}
			return st.props["value"]
		case "close":
			return g.finish(jsUndef)
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
	return rt.memberCall(target, name, args)
}

// callPyValue calls a value that may itself be a class or a __call__ instance.
func (rt *jsrt) callPyValue(v interface{}, args []interface{}, kw interface{}) interface{} {
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
				return pyTypeName(rt, v) == rt.toString(n)
			}
		}
		return false
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
	case float64:
		if t == math.Trunc(t) && !math.IsInf(t, 0) {
			return "int"
		}
		return "float"
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
		return rt.dictFind(keys, x) >= 0
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
			return append(append([]interface{}{}, pos...), kw)
		}
		return pos
	}
	if !sig.ext && !hasKw {
		return pos
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
		return append(out, extra...)
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
	return append(out, rest, left)
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

func (rt *jsrt) pyExcInstance(name string, msg string) interface{} {
	// The class the runtime raises with is a stable per-name object that derives
	// from a class called Exception, so pyIsInstance's name walk answers
	// `except Exception` (and `except <ThatName>`) for a StopIteration /
	// UnboundLocalError / AttributeError the RUNTIME raised, exactly like for one
	// the program raised through the module's own class objects.
	cls := rt.pyBuiltinClass(name)
	if name != "Exception" {
		if _, has := cls.props["__super"]; !has {
			cls.set("__super", rt.pyBuiltinClass("Exception"))
		}
	}
	inst := newJSObject()
	inst.set("__class", cls)
	inst.set("args", &jsArray{elems: []interface{}{msg}})
	return inst
}


// ----- Python's format mini-language (js_pyformat) -----
// [[fill]align][sign][0][width][.precision][type], which is the slice of it an
// f-string replacement field in practice writes.
func pyIsAlign(ch byte) bool { return ch == '<' || ch == '>' || ch == '^' || ch == '=' }

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
		if n, isNum := v.(float64); isNum && n >= 0 {
			body = "+" + body
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
