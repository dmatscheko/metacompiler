package abnf

import (
	"fmt"
	"math"
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
// Handle 0..3 are the singletons undefined, null, false and true. All other
// handles index the table. Numbers and strings are interned by value, our own
// pointer kinds by identity, so the table does not grow with every operation.

const (
	jsHUndefined = uint64(0)
	jsHNull      = uint64(1)
	jsHFalse     = uint64(2)
	jsHTrue      = uint64(3)
)

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

// jsObject is a plain MetaJS object. The key order is kept for deterministic behavior.
type jsObject struct {
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
	elems []interface{}
}

// jsClosure is a compiled MetaJS function: an IR function plus the captured
// scope. It also remembers the machine of its module, so closures from
// different modules (e.g. a helper library and a tag script) can call each
// other through one shared runtime.
type jsClosure struct {
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
// existence is the map key, not the value.
type jsScope struct {
	vars   map[string]interface{}
	parent *jsScope

	// types backs the MetaJS type pinning (js_tdecl/js_tset): it pins the type
	// class of a variable at its first non-undefined, non-null value. Languages
	// that declare through js_scope_decl/js_scope_set stay unpinned.
	types map[string]string
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
	table []interface{}

	strIntern map[string]uint64
	numIntern map[uint64]uint64 // Keyed by the float bits, so -0 and +0 stay distinct handles.
	objIntern map[interface{}]uint64 // Identity interning for pointer-like values.

	root *jsScope

	// retSlot holds the completion value of the running program: js_setret is
	// emitted for every expression statement, so after a run the slot holds the
	// value of the last executed expression statement - the same thing that a
	// goja Run() returns. The frozen engine saves and restores it around
	// nested runs.
	retSlot interface{}

	lastGets [][2]uint64 // The most recent member lookups (obj, key handles), for error messages.

	// The -trace hook (see trace.go). Only the program runtime of runJSModule
	// is traced; the frozen engine's tag-script runtime stays silent.
	traced     bool
	traceDepth int
	traceNames map[*jsClosure]string // Under which name a closure was stored.
	curPos     int                   // Source offset of the executing statement (js_srcpos), -1 = unknown.
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
		table:     []interface{}{jsUndef, jsNull, false, true},
		strIntern: map[string]uint64{},
		numIntern: map[uint64]uint64{},
		objIntern: map[interface{}]uint64{},
		retSlot:   jsUndef,
		curPos:    -1,
	}
	rootVars := map[string]interface{}{}
	for k, v := range bindings {
		rootVars[k] = v
	}
	rt.root = &jsScope{vars: rootVars}
	return rt
}

// attach loads a module into the runtime: the module gets its own machine
// (memory, globals, functions) whose js_* externals all work on the shared
// value table and scope world of this runtime.
func (rt *jsrt) attach(m *ir.Module) *machine {
	ma := newMachine(m, "")
	ma.externs = rt.externs(ma)
	return ma
}

// setRootVar binds or rebinds a host global. The frozen engine uses this to
// point 'up' and the stack functions at the environment of the current tag.
func (rt *jsrt) setRootVar(name string, v interface{}) {
	rt.root.vars[name] = v
}

// newScopeHandle creates a scope (a child of the root scope by default) and
// returns its handle. The frozen engine passes it as the shared environment
// of all scripts of one compile run.
func (rt *jsrt) newScopeHandle(parent *jsScope) uint64 {
	if parent == nil {
		parent = rt.root
	}
	return rt.wrap(&jsScope{vars: map[string]interface{}{}, parent: parent})
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
		rt.table = append(rt.table, v)
		return uint64(len(rt.table) - 1)
	}
}

func (rt *jsrt) wrapIdentity(v interface{}) uint64 {
	return rt.wrapIdentityKey(v, v)
}

func (rt *jsrt) wrapIdentityKey(key, v interface{}) uint64 {
	if h, ok := rt.objIntern[key]; ok {
		return h
	}
	rt.table = append(rt.table, v)
	h := uint64(len(rt.table) - 1)
	rt.objIntern[key] = h
	return h
}

func (rt *jsrt) wrapNum(f float64) uint64 {
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
	rt.table = append(rt.table, f)
	h := uint64(len(rt.table) - 1)
	if f == f {
		rt.numIntern[bits] = h
	}
	return h
}

func (rt *jsrt) wrapStr(s string) uint64 {
	if h, ok := rt.strIntern[s]; ok {
		return h
	}
	rt.table = append(rt.table, s)
	h := uint64(len(rt.table) - 1)
	rt.strIntern[s] = h
	return h
}

func (rt *jsrt) unwrap(h uint64) interface{} {
	if h >= uint64(len(rt.table)) {
		rt.fail("invalid handle %d", h)
	}
	return rt.table[h]
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
	case string:
		return len(t) > 0
	default:
		return true
	}
}

func (rt *jsrt) toNumber(v interface{}) float64 {
	switch t := v.(type) {
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
		return strings.Join(parts, ",")
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
		return as + bs
	}
	if _, ok := a.(*jsArray); ok {
		return rt.toString(a) + rt.toString(b)
	}
	if _, ok := b.(*jsArray); ok {
		return rt.toString(a) + rt.toString(b)
	}
	if _, ok := a.(*jsObject); ok {
		return rt.toString(a) + rt.toString(b)
	}
	if _, ok := b.(*jsObject); ok {
		return rt.toString(a) + rt.toString(b)
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
	default:
		// Objects, arrays, closures and Go natives compare by identity.
		return identityEq(a, b)
	}
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
	case *jsObject, *jsArray:
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
		if v, ok := s.vars[name]; ok {
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
		if _, ok := s.vars[name]; ok {
			s.vars[name] = v
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
		if _, ok := s.vars[name]; ok {
			s.vars[name] = v
			if rt.traced {
				rt.trVar("write", name, v)
			}
			return
		}
	}
	rt.root.vars[name] = v
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
	if sc.types == nil {
		sc.types = map[string]string{}
	}
	if isAnytype(v) {
		sc.vars[name] = jsUndef
		sc.types[name] = "*"
		if rt.traced {
			rt.trVar("decl", name, jsUndef)
		}
		return
	}
	sc.vars[name] = v
	if tc := rt.typeClass(v); tc != "" {
		sc.types[name] = tc
	} else {
		delete(sc.types, name) // A redeclaration starts untyped again.
	}
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
		if _, ok := s.vars[name]; ok {
			if tc := rt.typeClass(v); tc != "" {
				old, pinned := s.types[name]
				if pinned && old == "*" {
					// anytype variable: no check, and nothing re-pins it.
				} else if pinned && old != tc {
					rt.fail("MetaJS: variable '%s' has type %s and cannot hold a %s", name, old, tc)
				} else {
					if s.types == nil {
						s.types = map[string]string{}
					}
					s.types[name] = tc
				}
			}
			s.vars[name] = v
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
		if v, ok := o.props[rt.toString(key)]; ok {
			return v
		}
		return jsUndef
	case *jsArray:
		if ks, isStr := key.(string); isStr {
			switch ks {
			case "length":
				return float64(len(o.elems))
			case "push", "pop", "shift", "unshift", "slice", "indexOf", "join", "concat":
				return &boundMethod{recv: o, name: ks}
			}
		}
		idx := rt.toNumber(key)
		if idx == math.Trunc(idx) && idx >= 0 && int(idx) < len(o.elems) {
			return o.elems[int(idx)]
		}
		return jsUndef
	case string:
		if ks, isStr := key.(string); isStr {
			switch ks {
			case "length":
				return float64(jsStrLen(o))
			case "charCodeAt", "charAt", "indexOf", "replace", "slice", "substring", "split",
				"toUpperCase", "toLowerCase", "trim":
				return &boundMethod{recv: o, name: ks}
			}
		}
		idx := rt.toNumber(key)
		if idx == math.Trunc(idx) && idx >= 0 {
			if ch := jsStrAt(o, int(idx)); ch != "" {
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
		if idx, err := strconv.Atoi(name); err == nil {
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
	switch v.(type) {
	case bool, string, float64, *jsObject, *jsArray, *jsClosure, *hostFunc, *boundMethod, jsUndefT, jsNullT:
		return v
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
		o.set(rt.toString(key), val)
	case *jsArray:
		idx := rt.toNumber(key)
		if idx != math.Trunc(idx) || idx < 0 {
			rt.fail("invalid array index %s", rt.toString(key))
		}
		i := int(idx)
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
	if !rt.traced {
		return rt.callInner(callee, this, args)
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
	ret := rt.callInner(callee, this, args)
	completed = true
	rt.curPos = savedPos // The caller's statement continues after the call.
	rt.traceDepth--
	traceEmit(&TraceEvent{Ev: "ret", Depth: rt.traceDepth, Line: lineOfPos(rt.curPos), Val: rt.traceVal(ret)})
	return ret
}

func (rt *jsrt) callInner(callee interface{}, this interface{}, args []interface{}) interface{} {
	switch c := callee.(type) {
	case *jsClosure:
		arr := &jsArray{elems: args}
		ret := c.ma.call(c.fn, []uint64{c.env, rt.wrap(arr)})
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
func (rt *jsrt) dictFind(keys *jsArray, k interface{}) int {
	for i, e := range keys.elems {
		if rt.strictEq(e, k) {
			return i
		}
	}
	return -1
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
			return float64(jsStrLen(o))
		case "charAt":
			i := jsToInt(rt.toNumber(argAt(args, 0)))
			ch := jsStrAt(o, i)
			if ch == "" {
				rt.fail("charAt(%d) out of range for %q", i, o)
			}
			return ch
		case "equals":
			return rt.strictEq(o, argAt(args, 0))
		case "substring":
			begin, end := substringRange(jsStrLen(o), args, rt)
			return jsStrRange(o, begin, end)
		case "indexOf":
			return float64(jsStrIndexOf(o, rt.toString(argAt(args, 0))))
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
	case *jsArray:
		// Kotlin and Python style list methods.
		switch name {
		case "add":
			o.elems = append(o.elems, argAt(args, 0))
			return true
		case "append": // Python: returns None.
			o.elems = append(o.elems, argAt(args, 0))
			return jsUndef
		case "pop", "removeLast": // Python / Dart: both remove and return the last element.
			if len(o.elems) == 0 {
				rt.fail("pop from empty list")
			}
			v := o.elems[len(o.elems)-1]
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

// rubyMethod is js_rmcall: the direct method dispatch of the Ruby compiler
// (ruby-to-llvm-ir.abnf). It mirrors the mcall of ruby-interpreter.abnf exactly
// for strings, arrays and hashes (Ruby nil-on-empty, Ruby truthiness, .each over
// key/value pairs) and delegates class instances - and everything else - to the
// shared memberCall. It is deliberately separate from memberCall so that Ruby's
// semantics never perturb the Kotlin/Java/Go/Python languages that also use
// js_mcall.
func (rt *jsrt) rubyMethod(target interface{}, name string, args []interface{}) interface{} {
	switch o := target.(type) {
	case string:
		switch name {
		case "length", "size":
			return float64(jsStrLen(o))
		case "to_s":
			return o
		case "upcase":
			return strings.ToUpper(o)
		case "downcase":
			return strings.ToLower(o)
		case "include?":
			return strings.Contains(o, rt.toString(argAt(args, 0)))
		}
		rt.fail("unknown String method: %s", name)
	case *jsArray:
		return rt.rubyArrayMethod(o, name, args)
	case *jsObject:
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
		t.elems = append(t.elems, argAt(args, 0))
		return t
	case "pop":
		if len(t.elems) == 0 {
			return jsNull
		}
		v := t.elems[len(t.elems)-1]
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
		return rt.dictFind(t, argAt(args, 0)) >= 0
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
			recv.elems = append(recv.elems, args...)
			return float64(len(recv.elems))
		case "pop":
			if len(recv.elems) == 0 {
				return jsUndef
			}
			v := recv.elems[len(recv.elems)-1]
			recv.elems = recv.elems[:len(recv.elems)-1]
			return v
		case "shift":
			if len(recv.elems) == 0 {
				return jsUndef
			}
			v := recv.elems[0]
			recv.elems = append([]interface{}{}, recv.elems[1:]...)
			return v
		case "unshift":
			recv.elems = append(append([]interface{}{}, args...), recv.elems...)
			return float64(len(recv.elems))
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
			return strings.Join(parts, sep)
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
			if code := jsStrCodeAt(recv, i); code >= 0 {
				return float64(code)
			}
			return math.NaN()
		case "charAt":
			return jsStrAt(recv, jsToInt(argN(0)))
		case "indexOf":
			return float64(jsStrIndexOf(recv, argS(0)))
		case "replace":
			return strings.Replace(recv, argS(0), argS(1), 1)
		case "slice":
			begin, end := sliceRange(jsStrLen(recv), args, rt)
			return jsStrRange(recv, begin, end)
		case "substring":
			begin, end := substringRange(jsStrLen(recv), args, rt)
			return jsStrRange(recv, begin, end)
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
// Lone surrogate halves cannot round-trip through a Go string (they decode to
// U+FFFD); BMP behavior is exact.

func strASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

// strUnits returns s as UTF-16 code units.
func strUnits(s string) []uint16 { return utf16.Encode([]rune(s)) }

// strFromUnits builds a string back from UTF-16 code units.
func strFromUnits(u []uint16) string { return string(utf16.Decode(u)) }

// jsStrLen is the JS length of s: its number of UTF-16 code units.
func jsStrLen(s string) int {
	if strASCII(s) {
		return len(s)
	}
	return len(strUnits(s))
}

// jsStrIndexOf is the JS indexOf: the byte match position converted to a
// code unit index (a UTF-8 substring match always lies on a rune boundary).
func jsStrIndexOf(s, sub string) int {
	p := strings.Index(s, sub)
	if p <= 0 {
		return p // -1 and 0 are the same in both worlds.
	}
	return jsStrLen(s[:p])
}

// jsStrAt returns the one-code-unit string at unit index i, or "" outside.
func jsStrAt(s string, i int) string {
	if strASCII(s) {
		if i < 0 || i >= len(s) {
			return ""
		}
		return string(s[i])
	}
	units := strUnits(s)
	if i < 0 || i >= len(units) {
		return ""
	}
	return strFromUnits(units[i : i+1])
}

// jsStrCodeAt returns the code unit at unit index i, or -1 outside.
func jsStrCodeAt(s string, i int) int {
	if strASCII(s) {
		if i < 0 || i >= len(s) {
			return -1
		}
		return int(s[i])
	}
	units := strUnits(s)
	if i < 0 || i >= len(units) {
		return -1
	}
	return int(units[i])
}

// jsStrRange slices s by code unit indexes (begin <= end, both in range).
func jsStrRange(s string, begin, end int) string {
	if strASCII(s) {
		return s[begin:end]
	}
	return strFromUnits(strUnits(s)[begin:end])
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
	boolH := func(b bool) uint64 {
		if b {
			return jsHTrue
		}
		return jsHFalse
	}

	return map[string]func(args []uint64) uint64{
		// Scopes.
		"js_scope_new": func(a []uint64) uint64 {
			return w(&jsScope{vars: map[string]interface{}{}, parent: rt.scopeOf(a[0])})
		},
		"js_scope_decl": func(a []uint64) uint64 {
			name := rt.toString(u(a[1]))
			rt.scopeOf(a[0]).vars[name] = u(a[2])
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
				if v, ok := s.vars[name]; ok {
					if rt.traced {
						rt.trVar("read", name, v)
					}
					return w(v)
				}
			}
			for s := sc; s != nil; s = s.parent {
				if t, ok := s.vars["this"]; ok {
					if obj, isObj := t.(*jsObject); isObj {
						if v, ok := obj.props[name]; ok {
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
				if _, ok := s.vars[name]; ok {
					s.vars[name] = v
					if rt.traced {
						rt.trVar("write", name, v)
					}
					return 0
				}
			}
			for s := sc; s != nil; s = s.parent {
				if t, ok := s.vars["this"]; ok {
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
			ptr, n := a[0], a[1]
			if ptr+n > uint64(len(ma.mem)) {
				rt.fail("js_str_mem out of range")
			}
			return rt.wrapStr(string(ma.mem[ptr : ptr+n]))
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
			return w(rt.call(u(a[0]), u(a[1]), args.elems))
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
				return rt.wrapStr(rt.javaString(l) + rt.javaString(r))
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
		"js_sub": func(a []uint64) uint64 { return rt.wrapNum(rt.toNumber(u(a[0])) - rt.toNumber(u(a[1]))) },
		"js_mul": func(a []uint64) uint64 { return rt.wrapNum(rt.toNumber(u(a[0])) * rt.toNumber(u(a[1]))) },
		"js_div": func(a []uint64) uint64 { return rt.wrapNum(rt.toNumber(u(a[0])) / rt.toNumber(u(a[1]))) },
		"js_mod": func(a []uint64) uint64 { return rt.wrapNum(math.Mod(rt.toNumber(u(a[0])), rt.toNumber(u(a[1])))) },
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
		"js_neg":    func(a []uint64) uint64 { return rt.wrapNum(-rt.toNumber(u(a[0]))) },
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
		"js_tonum":  func(a []uint64) uint64 { return rt.wrapNum(rt.toNumber(u(a[0]))) },
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
				defer func() { caught = recover() }()
				res = rt.call(c, jsUndef, args)
				return
			}
			result, pending := run(tryC, nil)
			if pending != nil {
				if exc, isThrow := pending.(*jsThrown); isThrow && hasCatch {
					result, pending = run(catchC, []interface{}{exc.value})
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
		"js_pytruthy": func(a []uint64) uint64 { // Python truthiness: empty lists/dicts are falsy.
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
				n := jsStrLen(o)
				if idx < 0 {
					idx += n
				}
				if idx < 0 || idx >= n {
					rt.fail("string index out of range: %d", int(rt.toNumber(u(a[1]))))
				}
				return rt.wrapStr(jsStrAt(o, idx))
			}
			rt.fail("indexing a %s", rt.typeOf(u(a[0])))
			return 0
		},
		"js_pyset": func(a []uint64) uint64 { // List element (negative wraps) or dict entry assignment.
			if keys, vals, ok := dictParts(u(a[0])); ok {
				if i := rt.dictFind(keys, u(a[1])); i >= 0 {
					vals.elems[i] = u(a[2])
				} else {
					keys.elems = append(keys.elems, u(a[1]))
					vals.elems = append(vals.elems, u(a[2]))
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
			rt.fail("indexing a %s", rt.typeOf(u(a[0])))
			return 0
		},
		"js_map_del": func(a []uint64) uint64 { // delete(m, k); missing keys are a no-op.
			keys, vals, ok := dictParts(u(a[0]))
			if !ok {
				rt.fail("delete() needs a map")
			}
			if i := rt.dictFind(keys, u(a[1])); i >= 0 {
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
				return rt.wrapNum(float64(jsStrLen(o)))
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
		"js_rundefers": func(a []uint64) uint64 { // Runs collected [f, args] pairs LIFO (Go defer).
			arr, ok := u(a[0]).(*jsArray)
			if !ok {
				rt.fail("js_rundefers needs the defer list")
			}
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
			switch o := u(a[0]).(type) {
			case string:
				return rt.wrapNum(float64(jsStrLen(o)))
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
			switch o := u(a[0]).(type) {
			case *jsArray:
				return w(&jsArray{elems: append([]interface{}{}, o.elems...)})
			case string:
				out := &jsArray{}
				for i, n := 0, jsStrLen(o); i < n; i++ {
					out.elems = append(out.elems, jsStrAt(o, i))
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
			switch o := u(a[0]).(type) { // their keys, strings their characters.
			case *jsArray:
				return a[0]
			case string:
				out := &jsArray{}
				for i, n := 0, jsStrLen(o); i < n; i++ {
					out.elems = append(out.elems, jsStrAt(o, i))
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
				from, to := rt.pySliceRange(u(a[1]), u(a[2]), jsStrLen(o))
				return rt.wrapStr(jsStrRange(o, from, to))
			}
			rt.fail("slicing a %s", rt.typeOf(u(a[0])))
			return 0
		},
		"js_pyset_var": func(a []uint64) uint64 { // Assign in the chain, else declare locally.
			sc := rt.scopeOf(a[0])
			name := rt.toString(u(a[1]))
			for s := sc; s != nil; s = s.parent {
				if _, ok := s.vars[name]; ok {
					s.vars[name] = u(a[2])
					if rt.traced {
						rt.trVar("write", name, u(a[2]))
					}
					return 0
				}
			}
			sc.vars[name] = u(a[2])
			if rt.traced {
				rt.trVar("decl", name, u(a[2]))
			}
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
		"js_pyglobal": func(a []uint64) uint64 { // global NAME: ensure NAME is bound in the root scope.
			sc := rt.scopeOf(a[0])
			name := rt.toString(u(a[1]))
			root := sc
			for root.parent != nil {
				root = root.parent
			}
			if _, ok := root.vars[name]; !ok {
				root.vars[name] = jsUndef
			}
			return 0
		},
		"js_pynonlocal": func(a []uint64) uint64 { // nonlocal NAME: an enclosing non-root scope must bind NAME.
			sc := rt.scopeOf(a[0])
			name := rt.toString(u(a[1]))
			for s := sc.parent; s != nil && s.parent != nil; s = s.parent {
				if _, ok := s.vars[name]; ok {
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
		"js_pystr": func(a []uint64) uint64 { // str(v) with Python style rendering.
			return rt.wrapStr(rt.pyString(u(a[0])))
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
			fmt.Fprintln(outWriter, out)
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
	}
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
		var sb strings.Builder
		for _, a := range args {
			// ECMA ToUint16: NaN becomes 0 (via jsToInt) and larger values
			// wrap modulo 2^16 before the code unit becomes a rune.
			sb.WriteRune(rune(uint16(jsToInt(rt.toNumber(a)))))
		}
		return sb.String()
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
			return err == nil && idx >= 0 && idx < jsStrLen(o)
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
		"Math":   mathObj,
		"String": stringObj,
		"Object": objectObj,
		"Array":  arrayObj,
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
// This is the program runtime, so the -cfg and -trace hooks live here.
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
