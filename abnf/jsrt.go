package abnf

import (
	"fmt"
	"math"
	"os"
	"reflect"
	"strconv"
	"strings"

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

// jsScope is one link of a scope chain. Variables can hold undefined, so
// existence is the map key, not the value.
type jsScope struct {
	vars   map[string]interface{}
	parent *jsScope

	// types is only used by the typed JS dialect (js_tdecl/js_tset): it pins
	// the type class of a variable at its first non-undefined value.
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
	numIntern map[float64]uint64
	objIntern map[interface{}]uint64 // Identity interning for pointer-like values.

	root *jsScope

	// retSlot holds the completion value of the running program: js_setret is
	// emitted for every expression statement, so after a run the slot holds the
	// value of the last executed expression statement - the same thing that a
	// goja Run() returns. The frozen engine saves and restores it around
	// nested runs.
	retSlot interface{}

	lastGets [][2]uint64 // The most recent member lookups (obj, key handles), for error messages.
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
		numIntern: map[float64]uint64{},
		objIntern: map[interface{}]uint64{},
		retSlot:   jsUndef,
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
	if f == f { // Not NaN: NaN must not be a map key.
		if h, ok := rt.numIntern[f]; ok {
			return h
		}
	}
	rt.table = append(rt.table, f)
	h := uint64(len(rt.table) - 1)
	if f == f {
		rt.numIntern[f] = h
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
	return strconv.FormatFloat(f, 'g', -1, 64)
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
		if t == math.Trunc(t) && !math.IsInf(t, 0) && math.Abs(t) < 1e15 {
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
			return
		}
	}
	rt.fail("assignment to undeclared variable: %s", name)
}

// typeClass returns the fixed type class of a value for the typed JS dialect.
// undefined returns "" (it never pins a type); null counts as object.
func (rt *jsrt) typeClass(v interface{}) string {
	if _, u := v.(jsUndefT); u {
		return ""
	}
	if _, n := v.(jsNullT); n {
		return "object"
	}
	return rt.typeOf(v)
}

// typedDecl declares a variable and pins its type if the value already has one.
func (rt *jsrt) typedDecl(sc *jsScope, name string, v interface{}) {
	sc.vars[name] = v
	if sc.types == nil {
		sc.types = map[string]string{}
	}
	if tc := rt.typeClass(v); tc != "" {
		sc.types[name] = tc
	} else {
		delete(sc.types, name) // A redeclaration starts untyped again.
	}
}

// typedSet assigns like scopeSet but refuses to change a pinned type.
// Assigning undefined is allowed and keeps the pinned type.
func (rt *jsrt) typedSet(sc *jsScope, name string, v interface{}) {
	for s := sc; s != nil; s = s.parent {
		if _, ok := s.vars[name]; ok {
			if tc := rt.typeClass(v); tc != "" {
				if old, pinned := s.types[name]; pinned && old != tc {
					rt.fail("typed JS: variable '%s' has type %s and cannot hold a %s", name, old, tc)
				}
				if s.types == nil {
					s.types = map[string]string{}
				}
				s.types[name] = tc
			}
			s.vars[name] = v
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
				return float64(len(o))
			case "charCodeAt", "charAt", "indexOf", "replace", "slice", "substring", "split":
				return &boundMethod{recv: o, name: ks}
			}
		}
		idx := rt.toNumber(key)
		if idx == math.Trunc(idx) && idx >= 0 && int(idx) < len(o) {
			return string(o[int(idx)])
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

// pyString renders a value like Python's str(): True/False/None capitalized,
// lists in bracket notation.
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
			if s, isStr := e.(string); isStr {
				out += "'" + s + "'"
			} else {
				out += rt.pyString(e)
			}
		}
		return out + "]"
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
			return float64(len(o))
		case "charAt":
			i := int(rt.toNumber(argAt(args, 0)))
			if i < 0 || i >= len(o) {
				rt.fail("charAt(%d) out of range for %q", i, o)
			}
			return string(o[i])
		case "equals":
			return rt.strictEq(o, argAt(args, 0))
		case "substring":
			begin, end := sliceRange(len(o), args, rt)
			return o[begin:end]
		case "indexOf":
			return float64(strings.Index(o, rt.toString(argAt(args, 0))))
		case "isEmpty":
			return len(o) == 0
		}
		rt.fail("unknown String method: %s", name)
	case *jsObject:
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
		case "pop": // Python: removes and returns the last element.
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
		}
		rt.fail("unknown list method '%s'", name)
	}
	rt.fail("method call '%s' on a %s", name, rt.typeOf(target))
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
			i := int(argN(0))
			if len(args) == 0 {
				i = 0
			}
			if i < 0 || i >= len(recv) {
				return math.NaN()
			}
			return float64(recv[i])
		case "charAt":
			i := int(argN(0))
			if len(args) == 0 {
				i = 0
			}
			if i < 0 || i >= len(recv) {
				return ""
			}
			return string(recv[i])
		case "indexOf":
			return float64(strings.Index(recv, argS(0)))
		case "replace":
			return strings.Replace(recv, argS(0), argS(1), 1)
		case "slice", "substring":
			begin, end := sliceRange(len(recv), args, rt)
			return recv[begin:end]
		case "split":
			parts := strings.Split(recv, argS(0))
			out := &jsArray{}
			for _, p := range parts {
				out.elems = append(out.elems, p)
			}
			return out
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
			rt.scopeOf(a[0]).vars[rt.toString(u(a[1]))] = u(a[2])
			return 0
		},
		"js_scope_get": func(a []uint64) uint64 {
			return w(rt.scopeGet(rt.scopeOf(a[0]), rt.toString(u(a[1]))))
		},
		"js_scope_set": func(a []uint64) uint64 {
			rt.scopeSet(rt.scopeOf(a[0]), rt.toString(u(a[1])), u(a[2]))
			return 0
		},

		// Scope access with 'this' fallback (Kotlin style implicit property access):
		// a name that is no local resolves against the properties of 'this'.
		"js_kget": func(a []uint64) uint64 {
			sc := rt.scopeOf(a[0])
			name := rt.toString(u(a[1]))
			for s := sc; s != nil; s = s.parent {
				if v, ok := s.vars[name]; ok {
					return w(v)
				}
			}
			for s := sc; s != nil; s = s.parent {
				if t, ok := s.vars["this"]; ok {
					if obj, isObj := t.(*jsObject); isObj {
						if v, ok := obj.props[name]; ok {
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
					return 0
				}
			}
			for s := sc; s != nil; s = s.parent {
				if t, ok := s.vars["this"]; ok {
					if obj, isObj := t.(*jsObject); isObj {
						if _, ok := obj.props[name]; ok {
							obj.set(name, v)
							return 0
						}
					}
					break
				}
			}
			rt.fail("assignment to unknown name: %s", name)
			return 0
		},

		// The typed JS dialect: declarations and assignments with type pinning.
		"js_tdecl": func(a []uint64) uint64 {
			rt.typedDecl(rt.scopeOf(a[0]), rt.toString(u(a[1])), u(a[2]))
			return 0
		},
		"js_tset": func(a []uint64) uint64 {
			rt.typedSet(rt.scopeOf(a[0]), rt.toString(u(a[1])), u(a[2]))
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
		"js_arr_new_n": func(a []uint64) uint64 { // (length, fill value) -> array handle
			n := int(a[0])
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
		"js_get": func(a []uint64) uint64 {
			rt.noteGet(a[0], a[1])
			return w(rt.getMember(u(a[0]), u(a[1])))
		},
		"js_set": func(a []uint64) uint64 {
			rt.setMember(u(a[0]), u(a[1]), u(a[2]))
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
		"js_jadd": func(a []uint64) uint64 { // Java +: string concatenation or 32 bit int addition.
			l, r := u(a[0]), u(a[1])
			_, ls := l.(string)
			_, rs := r.(string)
			if ls || rs {
				return rt.wrapStr(rt.javaString(l) + rt.javaString(r))
			}
			return rt.wrapNum(float64(int32(int64(rt.toNumber(l)) + int64(rt.toNumber(r)))))
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
		"js_tonum":  func(a []uint64) uint64 { return rt.wrapNum(rt.toNumber(u(a[0]))) },
		"js_throw": func(a []uint64) uint64 {
			rt.fail("thrown: %s", rt.toString(u(a[0])))
			return 0
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
		"js_pytruthy": func(a []uint64) uint64 { // Python truthiness: empty lists are falsy.
			if arr, ok := u(a[0]).(*jsArray); ok {
				if len(arr.elems) > 0 {
					return 1
				}
				return 0
			}
			if rt.truthy(u(a[0])) {
				return 1
			}
			return 0
		},
		"js_pyget": func(a []uint64) uint64 { // Sequence indexing with negative wrap around.
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
				if idx < 0 {
					idx += len(o)
				}
				if idx < 0 || idx >= len(o) {
					rt.fail("string index out of range: %d", int(rt.toNumber(u(a[1]))))
				}
				return rt.wrapStr(string(o[idx]))
			}
			rt.fail("indexing a %s", rt.typeOf(u(a[0])))
			return 0
		},
		"js_pyset": func(a []uint64) uint64 { // List element assignment with negative wrap around.
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
		"js_pyset_var": func(a []uint64) uint64 { // Assign in the chain, else declare locally.
			sc := rt.scopeOf(a[0])
			name := rt.toString(u(a[1]))
			for s := sc; s != nil; s = s.parent {
				if _, ok := s.vars[name]; ok {
					s.vars[name] = u(a[2])
					return 0
				}
			}
			sc.vars[name] = u(a[2])
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
			fmt.Println(out)
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
	mathObj.set("max", jsHostFunc("max", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return math.Max(rt.toNumber(argAt(args, 0)), rt.toNumber(argAt(args, 1)))
	}))
	mathObj.set("min", jsHostFunc("min", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return math.Min(rt.toNumber(argAt(args, 0)), rt.toNumber(argAt(args, 1)))
	}))

	stringObj := newJSObject()
	stringObj.set("fromCharCode", jsHostFunc("fromCharCode", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		var sb strings.Builder
		for _, a := range args {
			sb.WriteRune(rune(int64(rt.toNumber(a))))
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
			return err == nil && idx >= 0 && idx < len(o)
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
			fmt.Println(rt.printArgs(args)...)
			return jsUndef
		}),
		"print": jsHostFunc("print", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			fmt.Print(rt.printArgs(args)...)
			return jsUndef
		}),
		"printf": jsHostFunc("printf", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			if len(args) == 0 {
				return jsUndef
			}
			fmt.Printf(rt.toString(args[0]), rt.printArgs(args[1:])...)
			return jsUndef
		}),
		"sprintf": jsHostFunc("sprintf", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			if len(args) == 0 {
				return ""
			}
			return fmt.Sprintf(rt.toString(args[0]), rt.printArgs(args[1:])...)
		}),
		"parseInt": jsHostFunc("parseInt", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return jsParseInt(rt.toString(argAt(args, 0)), int(rt.toNumber(argAt(args, 1))))
		}),
		"parseFloat": jsHostFunc("parseFloat", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return jsParseFloat(rt.toString(argAt(args, 0)))
		}),
		"exit": jsHostFunc("exit", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			os.Exit(int(int32(int64(rt.toNumber(argAt(args, 0))))))
			return jsUndef
		}),
		"Math":   mathObj,
		"String": stringObj,
		"Object": objectObj,
		"Array":  arrayObj,
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
	return ma.callByName(name, []uint64{env, rt.wrap(&jsArray{})})
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
func runJSModule(m *ir.Module, entry string) *RunResult {
	rt := newJSRT(standardJSBindings())
	ma := rt.attach(m)
	h := rt.callEntry(ma, entry, 0)
	return &RunResult{Ret: uint32(rt.toInt32(rt.unwrap(h))), Out: ""}
}
