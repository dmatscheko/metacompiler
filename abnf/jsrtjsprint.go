package abnf

// jsrtjsprint.go -- the JavaScript/TypeScript half of the shared runtime.
//
// The compiler grammars (languages/js-to-llvm-ir.abnf, languages/typescript-to-llvm-ir.abnf)
// used to reach the LANGUAGE-NEUTRAL externs of abnf/jsrt.go for printing, for the
// equality and relational operators, for `instanceof`, for `delete` and for the
// String/Array/Object method surface. Every one of those is a compromise between a
// dozen languages, and JavaScript is the one language whose answer they are NOT:
//
//   println(null)        printed Go's %v ("<nil>") instead of "null"
//   println([1, 2, 3])   printed "[1 2 3]" instead of "1,2,3"
//   println({a: 1})      printed a Go map dump instead of "[object Object]"
//   [1] == 1             was false (no ToPrimitive on the object operand)
//   [2] > 1              was false (same)
//   new F() instanceof F was false for a plain constructor FUNCTION
//   delete a[1]          was a no-op
//   Object.keys(o)       aborted with "unknown method 'keys'"
//   a.map(f) / "s".padStart(...) and ~30 more were missing outright
//
// Everything here is ADDITIVE, exactly like abnf/jsrtregexkt.go and
// abnf/jsrtregexjs.go: the init() below appends to rxExtraExterns and the registrar
// only ADDS js_js* names. Each wrapper delegates to the extern it extends for every
// receiver it does not claim, so no other language sees any of it.
//
// The interpreter halves (languages/js-interpreter.abnf, languages/typescript-interpreter.abnf)
// carry the SAME answers written in their own tag script - `jsStr`, `jsLooseEq`,
// `jsRelate`, `jsBuiltin`, `jsObjectGlobal`. The two files must be read together: the
// matrix compares each engine against itself and can never see them drift apart (see
// docs/abnf-dialect-gotchas.md, "The matrix compares each engine against ITSELF").
// Every answer below was settled against node v24.

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

func init() {
	rxExtraExterns = append(rxExtraExterns, func(rt *jsrt, m map[string]func(args []uint64) uint64) {
		rt.addJSValueExterns(m)
	})
	// js_jsmcall reads its argument array in position 2, exactly like the js_mcall
	// it ultimately wraps, so the handle recycler may hand that array through.
	jsThroughArgs["js_jsmcall"] = 1 << 2
}

// ============================================================================
// ToString
//
// rt.toString already spells JavaScript's ToString for every primitive, for an
// array (comma-joined, a hole/null/undefined element joining as "") and for a
// plain object ("[object Object]"). What it cannot do is run a USER toString:
// an instance keeps its methods on the __class descriptor rather than on a
// prototype chain, so the lookup has to walk __class/__super by hand.

// jsvCallMethod runs o's own `name` method, if it has one, and answers
// (result, true). The two places a method can live take DIFFERENT calling
// conventions: a class method sits on the __class descriptor and gets the
// receiver PREPENDED to its arguments (what rt.memberCall does), while an
// object-literal method is an ordinary closure that reads `this` from the call's
// receiver slot. Getting that backwards silently passes the wrong `this`.
func (rt *jsrt) jsvCallMethod(o *jsObject, name string) (interface{}, bool) {
	for cls := o.props["__class"]; cls != nil; {
		clsObj, ok := cls.(*jsObject)
		if !ok {
			break
		}
		if m, ok := clsObj.props[name]; ok && isCallable(m) {
			return rt.call(m, jsUndef, []interface{}{o}), true
		}
		cls = clsObj.props["__super"]
	}
	if m, ok := o.props[name]; ok && isCallable(m) {
		return rt.call(m, o, nil), true
	}
	return nil, false
}

// jsvFindClassMethod walks the __class/__super chain for a callable member. It is
// the read-side half of jsvCallMethod: an instance keeps its methods on the class
// descriptor, so a plain property read never finds one. The `guard` cap is the same
// 64 the rest of this project puts on a descriptor walk - a cyclic __super hangs the
// oracle too, and no program here can express a 65-deep hierarchy.
func jsvFindClassMethod(o *jsObject, name string) (interface{}, bool) {
	guard := 0
	for cls := o.props["__class"]; cls != nil && guard < 64; guard++ {
		clsObj, ok := cls.(*jsObject)
		if !ok {
			break
		}
		if m, has := clsObj.props[name]; has && isCallable(m) {
			return m, true
		}
		cls = clsObj.props["__super"]
	}
	return nil, false
}

// jsvIsStaticSide reports that a member read on this receiver must be answered from
// the STATIC namespace and from nowhere else. A class descriptor's own property map
// holds its INSTANCE methods, so falling back to it after a static miss is exactly
// the bug: `class C { m(){} }` then `C.m` is undefined in node, not the method.
// The engine's own slots (__ctor, __name, __super, ...) are excluded - they are read
// off the descriptor itself by makeNewFrom, instanceof and the error machinery.
// A mangled PRIVATE name is a __ name too, and it is NOT one of those: `static #s = 5`
// files '__#s' in the static table, so skipping it made `C.#s` answer undefined in
// both compiled halves. jsvPrivateKey is the exception. docs/todo.md 2.4.
func jsvIsStaticSide(v interface{}, name string) (*jsObject, bool) {
	o, ok := v.(*jsObject)
	if !ok || !isClassDesc(o) || (jsvInternalKey(name) && !jsvPrivateKey(name)) {
		return nil, false
	}
	return o, true
}

// jsvPrivateKey recognizes the mangled spelling of a private name, '__#x' - the key
// both js/typescript emitters build with privName and both interpreter halves build
// with their own privName. Twin: jvIsPrivateKey in languages/lib/js-rt.metajs.
func jsvPrivateKey(k string) bool {
	return len(k) >= 3 && k[0] == '_' && k[1] == '_' && k[2] == '#'
}

// The value of a member read on a class descriptor: the static chain, an accessor
// unwrapped against the receiver the read started from (node runs `static get g()`
// with `this` bound to the SUBCLASS), and `name` last - every class object has one,
// and it is the descriptor's __name unless a `static name` member shadows it
// (docs/todo.md 2.13 item 3). Anything else is undefined, as node says.
func (rt *jsrt) jsvStaticSideRead(o *jsObject, name string) uint64 {
	if v, _, found := jsStaticsFind(o, name); found {
		if acc, isAcc := v.(*jsAccessor); isAcc {
			if acc.get == nil {
				return jsHUndefined
			}
			return rt.wrap(rt.call(acc.get, o, nil))
		}
		return rt.wrap(v)
	}
	if name == "name" {
		if n, has := o.props["__name"]; has {
			return rt.wrap(n)
		}
	}
	return jsHUndefined
}

// jsvProtoTag is Object.prototype.toString's answer for a value.
func jsvProtoTag(v interface{}) string {
	switch v.(type) {
	case nil, jsNullT:
		return "[object Null]"
	case jsUndefT:
		return "[object Undefined]"
	case *jsArray:
		return "[object Array]"
	case float64:
		return "[object Number]"
	case string:
		return "[object String]"
	case bool:
		return "[object Boolean]"
	}
	if isCallable(v) {
		return "[object Function]"
	}
	return "[object Object]"
}

// jsvString is String(v). An object with a toString of its own answers with it (the
// method convention prepends the receiver, exactly as rt.memberCall does).
func (rt *jsrt) jsvString(v interface{}) string {
	switch t := v.(type) {
	case *jsObject:
		if r, ok := rt.jsvCallMethod(t, "toString"); ok {
			return rt.toString(r)
		}
		// A promise carries hidden slots only, so the generic object arm below
		// would spell it "[object Object]" where node says "[object Promise]" -
		// Promise.prototype has a @@toStringTag. The same arm exists in
		// languages/lib/js-rt.metajs's jvStr and in the interpreter's jsStr.
		if promIs(t) {
			return "[object Promise]"
		}
		return "[object Object]"
	case *jsArray:
		parts := make([]string, len(t.elems))
		for i, e := range t.elems {
			if isUndefOrNull(e) {
				continue // undefined and null join as the empty string.
			}
			parts[i] = rt.jsvString(e)
		}
		return strJoin(parts, ",")
	}
	return rt.toString(v)
}

// ============================================================================
// ToPrimitive, abstract equality and the relational operators

func jsvIsObjLike(v interface{}) bool {
	switch v.(type) {
	case *jsObject, *jsArray:
		return true
	}
	return false
}

// jsvToPrimitive is ToPrimitive(v, default): valueOf first, then toString. It is the
// step the shared js_eq / js_lt were missing, which is why `[1] == 1` and `[2] > 1`
// were false in the compiler and true in node.
func (rt *jsrt) jsvToPrimitive(v interface{}) interface{} {
	if o, ok := v.(*jsObject); ok {
		if r, ok := rt.jsvCallMethod(o, "valueOf"); ok && !jsvIsObjLike(r) {
			return r
		}
		return rt.jsvString(o)
	}
	if _, ok := v.(*jsArray); ok {
		return rt.jsvString(v)
	}
	return v
}

// jsvLooseEq is the abstract equality comparison. null == undefined is true, both are
// equal to nothing else, two objects compare by identity and an object against a
// primitive compares its primitive form.
func (rt *jsrt) jsvLooseEq(a, b interface{}) bool {
	if isUndefOrNull(a) || isUndefOrNull(b) {
		return isUndefOrNull(a) && isUndefOrNull(b)
	}
	ao, bo := jsvIsObjLike(a), jsvIsObjLike(b)
	if ao && bo {
		return rt.strictEq(a, b)
	}
	if ao {
		return rt.jsvLooseEq(rt.jsvToPrimitive(a), b)
	}
	if bo {
		return rt.jsvLooseEq(a, rt.jsvToPrimitive(b))
	}
	return rt.looseEq(a, b)
}

// jsvCompare is the abstract relational comparison: ToPrimitive on both sides, then
// the shared numeric/string ordering (2 means "NaN was involved", every relation false).
func (rt *jsrt) jsvCompare(a, b interface{}) int {
	return rt.jsCompare(rt.jsvToPrimitive(a), rt.jsvToPrimitive(b))
}

// ============================================================================
// The Object global
//
// The root scope's `Object` is a bare host object carrying only `prototype`, so
// Object.keys / values / entries / assign aborted with "unknown method 'keys'". The
// grammar declares the marked object below in the PROGRAM's own scope (the way it
// already declares RegExp), so nothing outside the JS/TS pair sees it.

func jsvIsObjectGlobal(v interface{}) bool {
	o, ok := v.(*jsObject)
	return ok && o.props["__jsobject"] != nil
}

// jsvIsArrayGlobal recognizes the root scope's `Array` binding by its one member.
func jsvIsArrayGlobal(v interface{}) bool {
	o, ok := v.(*jsObject)
	if !ok {
		return false
	}
	_, has := o.props["isArray"]
	return has && len(o.props) == 1
}

// jsvOwnKeys are an object's own enumerable keys in insertion order, WITHOUT the
// __-prefixed internal slots (__class, __keys, __super, ...). The interpreter half
// leaked exactly those through Object.keys / Object.entries.
func jsvOwnKeys(v interface{}) []string {
	switch o := v.(type) {
	case *jsObject:
		// A CLASS DESCRIPTOR answers with the enumerable half of its STATIC
		// namespace, and with nothing off its own property map: that map holds the
		// class's INSTANCE methods, which node reports on the prototype and never on
		// the constructor, so `Object.keys(class C { static sf = 1; m(){} })`
		// answered ["m"] where node says ["sf"]. docs/todo.md 2.10.
		if isClassDesc(o) {
			return jsvStaticEnumKeys(o)
		}
		out := []string{}
		for _, k := range o.keys {
			if !jsvInternalKey(k) {
				out = append(out, k)
			}
		}
		return jsvErrHide(o, out)
	case *jsArray:
		out := make([]string, len(o.elems))
		for i := range o.elems {
			out[i] = strconv.Itoa(i)
		}
		return out
	case string:
		n := len([]rune(o))
		out := make([]string, n)
		for i := 0; i < n; i++ {
			out[i] = strconv.Itoa(i)
		}
		return out
	}
	// A FUNCTION enumerates the own properties written onto it, which node reports
	// and this engine keeps in a side table. docs/todo.md 2.10.
	if p := jsvFnPropsOf(v); p != nil {
		return jsvOwnKeys(p)
	}
	return []string{}
}

// Object.getOwnPropertyNames: jsvOwnKeys plus the non-enumerable own names. A FUNCTION
// has `length` and `name`, in that order, before anything the program wrote onto it -
// node answers ["length","name"] for an arrow and
// ["length","name","arguments","caller","prototype"] for a function declaration, and the
// last three cannot exist in a value model with no prototype chain and no call-site
// frame. Object.keys, values, entries and assign keep reading jsvOwnKeys, and so does
// for-in, which is what preserves node's empty `for (k in f)`. Twins: jvGopnKeys in
// languages/lib/js-rt.metajs and jsGopnKeys in both *-interpreter.abnf.
// docs/todo.md 1.4.
func jsvGopnKeys(v interface{}) []string {
	if !jsvIsFnLike(v) {
		return jsvOwnKeys(v)
	}
	return append([]string{"length", "name"}, jsvOwnKeys(v)...)
}

// jsvFnProps -- `f.p = 1` on a FUNCTION. A function value here is a *jsClosure or
// a *hostFunc and neither carries a property map, so a member write on one aborted
// with "cannot set field 'p' on *abnf.jsClosure" - while node gives every function
// an ordinary own property map, and `f.cache = {}` is a common idiom. The map lives
// BESIDE the function in this side table, keyed by the pointer, which is the
// identity JavaScript uses. Only the two POINTER kinds are admitted, so nothing
// uncomparable can reach the map key. The table pins its functions - a function
// whose properties are reachable is live anyway - and it is package level rather
// than per-jsrt so that jsvOwnKeys, which has no runtime, can read it; the keys are
// pointers, so two runtimes in one process cannot collide.
// Twin: jvFnProps* in languages/lib/js-rt.metajs. The interpreter halves keep the
// same table under the same name. docs/todo.md 2.10.
var jsvFnProps = map[interface{}]*jsObject{}

// jsvFnMeta -- the declared NAME and ARITY of a function, set by js_jsfnmeta at the
// closure's creation site. The pair is stored ON the function value (the meta field of
// *jsClosure and *hostFunc), so setting it is O(1) and it does not keep the function
// alive - the C floor keeps it in the function cell the same way. Twins: js_jsfnmeta
// in languages/lib/runtime.c, reached from layer 2 as fnMeta/fnMetaSet, and fnMeta*
// in both *-interpreter.abnf.
type jsvFnInfo struct {
	name   string
	length int
}

// jsvFnMeta answers ""/0 for a function that was never named and for a non-function.
func jsvFnMeta(v interface{}) jsvFnInfo {
	switch f := v.(type) {
	case *jsClosure:
		if f.meta != nil {
			return *f.meta
		}
	case *hostFunc:
		if f.meta != nil {
			return *f.meta
		}
	}
	return jsvFnInfo{}
}

// jsvFnMetaSet records the pair on a function value; any other value is ignored.
func jsvFnMetaSet(v interface{}, name string, length int) {
	switch f := v.(type) {
	case *jsClosure:
		f.meta = &jsvFnInfo{name: name, length: length}
	case *hostFunc:
		f.meta = &jsvFnInfo{name: name, length: length}
	}
}

// jsvMethodVals -- a __class method READ AS A VALUE, one per method closure, so that
// `p.m === p.m` and `p.m === q.m` hold as they do in node. The closure takes its
// receiver PREPENDED to the arguments (see jsvCallMethod); the value is the plain
// function that takes `this` from its call site instead. Nothing is bound: a bare
// `g()` runs the body with `this` undefined, which is the TypeError a class body's
// strict mode gives in node. The table pins the method closures it has seen, which
// live as long as their class descriptor anyway. Twins:
// jvMethodValue in languages/lib/js-rt.metajs and methodValueShim in both
// *-interpreter.abnf.
var jsvMethodVals = map[interface{}]*hostFunc{}

func jsvMethodValue(name string, fn interface{}) *hostFunc {
	if shim, found := jsvMethodVals[fn]; found {
		return shim
	}
	shim := jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return rt.call(fn, jsUndef, append([]interface{}{rt.unwrap(this)}, args...))
	})
	// The value answers the method's own name and arity: node says `meth`/1 for
	// `new C().meth` of `meth(x){}`. The closure's registered length already excludes
	// the prepended receiver.
	jsvFnMetaSet(shim, name, jsvFnMeta(fn).length)
	jsvMethodVals[fn] = shim
	return shim
}

// The value of `f.name` / `f.length`, or nil for any other name and any receiver that is
// not a function. The DEFAULTS are ""/0 and not "absent": every function has both
// properties, and a function this table never saw - a host builtin like a String method,
// or one of the emitter's own internal thunks - answers what the interpreter halves
// answer for it, which is what keeps the two halves equal. It is asked on the MISS path
// of js_jsvget and js_jsmget, before jsvFnMethod, and BEFORE the own-property side table
// would be consulted for a write that node drops (both are non-writable there).
func jsvFnIntrinsic(v interface{}, name string) interface{} {
	if name != "name" && name != "length" {
		return nil
	}
	if !jsvIsFnLike(v) {
		return nil
	}
	mi := jsvFnMeta(v)
	if name == "name" {
		return mi.name
	}
	return float64(mi.length)
}

func jsvIsFnLike(v interface{}) bool {
	switch v.(type) {
	case *jsClosure, *hostFunc:
		return true
	}
	return false
}

func jsvFnPropsOf(v interface{}) *jsObject {
	if len(jsvFnProps) == 0 || !jsvIsFnLike(v) {
		return nil
	}
	return jsvFnProps[v]
}

func jsvFnPropsMake(v interface{}) *jsObject {
	if !jsvIsFnLike(v) {
		return nil
	}
	if p, found := jsvFnProps[v]; found {
		return p
	}
	p := newJSObject()
	jsvFnProps[v] = p
	return p
}

// jsvFnMethod -- the Object.prototype SURFACE of a FUNCTION, as methods.
// `f.toString()`, `f.valueOf()`, `f.hasOwnProperty(k)` and
// `f.propertyIsEnumerable(k)` all died with "f.toString is not a function" in both
// compiled halves and in the -frozen interpreter, while the goja interpreter answered
// them off the host's own prototype chain - a live goja-vs--frozen divergence that
// predates the function side table and that nothing reached, because a function was
// not a property carrier and nobody asked. Twins: jvFnMethod in
// languages/lib/js-rt.metajs and fnHasOwnShim / fnToStringShim / fnValueOfShim in both
// *-interpreter.abnf. Answers nil for any receiver or name it does not claim, so the
// caller tests one value instead of four names. docs/todo.md 2.10.
//
// Every own property a program can write onto a function here is an ordinary
// enumerable data property, so hasOwnProperty and propertyIsEnumerable share a body,
// and the KEY MAP is asked rather than the value, so a property holding undefined
// still answers true. toString agrees with rt.jsvString rather than inventing a second
// spelling: no engine here keeps a function's source text.
func (rt *jsrt) jsvFnMethod(recv interface{}, name string) interface{} {
	// The NAME switches before the type test, not after: this body sits on the MISS
	// path of js_jsmget and js_jsvget, which an ordinary absent-property read reaches.
	switch name {
	case "hasOwnProperty", "propertyIsEnumerable", "toString", "valueOf":
	default:
		return nil
	}
	if !jsvIsFnLike(recv) {
		return nil
	}
	switch name {
	case "hasOwnProperty", "propertyIsEnumerable":
		enumerable := name == "propertyIsEnumerable"
		return jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			// `name` and `length` are own properties of every function, so
			// hasOwnProperty says true - and they are the one pair here for which
			// propertyIsEnumerable says FALSE, which is why the two questions no
			// longer share one answer. docs/todo.md 1.4.
			if k := rt.toString(argAt(args, 0)); k == "name" || k == "length" {
				return !enumerable
			}
			p := jsvFnPropsOf(recv)
			if p == nil {
				return false
			}
			_, found := p.props[rt.toString(argAt(args, 0))]
			return found
		})
	case "toString":
		return jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return rt.jsvString(recv)
		})
	case "valueOf":
		return jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return recv
		})
	}
	return nil
}

func jsvInternalKey(k string) bool {
	return len(k) >= 2 && k[0] == '_' && k[1] == '_'
}

// ----- which static members of a class are ENUMERABLE (docs/todo.md 2.10) -----
//
// A class object's own enumerable properties are its static FIELDS plus anything a
// runtime `C.zz = 1` put there. A static METHOD and a static ACCESSOR are not
// enumerable, and neither is `name`, `length` or `prototype`. Every one of those
// lives in the SAME table (__statics, so that jsStaticsFind can walk one chain), so
// the table's own key order cannot separate them: the enumerable names are recorded
// beside them in __sfk, by js_jssfield at class-definition time and by js_jsmset for
// a runtime write. Twins: jvSfTrack / jvStaticEnumKeys in languages/lib/js-rt.metajs,
// and the statics table's __keys list in the two *-interpreter.abnf start scripts
// (an interpreter object already tracks insertion order by hand, so it needs no
// second list - only a decision about which writes track).
const jsvSFieldKeys = "__sfk"

// jsvSfTrack records `name` as an enumerable static of the class owning this table.
func jsvSfTrack(statics *jsObject, name string) {
	if jsvInternalKey(name) {
		return
	}
	lst, _ := statics.props[jsvSFieldKeys].(*jsArray)
	if lst == nil {
		lst = &jsArray{}
		statics.set(jsvSFieldKeys, lst)
	}
	for _, e := range lst.elems {
		if s, ok := e.(string); ok && s == name {
			return
		}
	}
	lst.elems = append(lst.elems, name)
}

// jsvIndexKeys is the own enumerable key list of an indexed value - an array or a
// string - which node answers as the index STRINGS "0".."n-1". Twin: jvIndexKeys in
// languages/lib/js-rt.metajs. docs/todo.md 2.4.
func jsvIndexKeys(n int) *jsArray {
	out := &jsArray{}
	for i := 0; i < n; i++ {
		out.elems = append(out.elems, strconv.Itoa(i))
	}
	return out
}

// jsvStaticEnumKeys is Object.keys of a class object.
func jsvStaticEnumKeys(cls *jsObject) []string {
	out := []string{}
	t := jsStaticsOf(cls, false)
	if t == nil {
		return out
	}
	lst, _ := t.props[jsvSFieldKeys].(*jsArray)
	if lst == nil {
		return out
	}
	for _, e := range lst.elems {
		if s, ok := e.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// jsvConstructorOf is what `x.constructor` answers for an object this engine built:
// the class descriptor the instance carries, or - for an instance of a constructor
// FUNCTION - the function itself, which js_jsctordesc remembers under __ctorfn. nil
// for a value with no __class at all (a plain object literal; node answers Object
// there, which needs a prototype chain this value model does not have). Twins:
// jvConstructorOf in languages/lib/js-rt.metajs and classCtorOf in the two
// *-interpreter.abnf start scripts. docs/todo.md 2.10.
func jsvConstructorOf(o *jsObject) (interface{}, bool) {
	cls, ok := o.props["__class"].(*jsObject)
	if !ok {
		return nil, false
	}
	if fn, has := cls.props["__ctorfn"]; has {
		return fn, true
	}
	return cls, true
}

func (rt *jsrt) jsvGetKey(v interface{}, k string) interface{} {
	// A CLASS DESCRIPTOR reads from its static namespace, exactly as js_jsvget does:
	// rt.getMember would answer the descriptor's own slot, which is where the
	// INSTANCE methods live, so Object.values(C) / Object.entries(C) / Object.assign
	// came back undefined for every static field once jsvOwnKeys started naming them.
	// The native leg goes through js_jsmget and never had this. docs/todo.md 2.10.
	if cls, isStatic := jsvIsStaticSide(v, k); isStatic {
		return rt.unwrap(rt.jsvStaticSideRead(cls, k))
	}
	// A FUNCTION's own properties live in the side table, and rt.getMember knows
	// nothing about it - the SAME shape as the class-descriptor arm above, one layer
	// out: once jsvOwnKeys started naming a function's properties, Object.values(f)
	// and Object.entries(f) answered the right KEYS with undefined VALUES, and
	// Object.assign({}, f) copied undefined. On the llvm.Run leg only, because the
	// native leg goes through layer 2's js_jsmget. docs/todo.md 2.10.
	if p := jsvFnPropsOf(v); p != nil {
		if val, found := p.props[k]; found {
			return val
		}
	}
	return rt.getMember(v, k)
}

// ============================================================================
// The String / Array / Number method surface
//
// jsvMethod answers (value, true) when the call is one of ours and (nil, false) when
// it is not, so js_jsmcall falls through to js_jsxmcall (the regular-expression
// wrapper) and from there to the shared js_mcall for everything else.

// jsvStringMethods / jsvArrayMethods are the names js_jsmget hides from js_get, so
// that a call on a string or an array reaches jsvMethod instead of the shared
// boundMethod path - which implements a DIFFERENT (Java/Kotlin flavoured) semantics
// for several of them (indexOf ignores its from-index, split takes no limit,
// replace has no pattern surface, map's callback gets no index).
var jsvStringMethods = map[string]bool{
	"charAt": true, "charCodeAt": true, "codePointAt": true, "at": true,
	"indexOf": true, "lastIndexOf": true, "includes": true,
	"startsWith": true, "endsWith": true, "slice": true, "substring": true,
	"substr": true, "split": true, "toUpperCase": true, "toLowerCase": true,
	"trim": true, "trimStart": true, "trimEnd": true, "padStart": true,
	"padEnd": true, "repeat": true, "concat": true, "replace": true,
	"replaceAll": true, "toString": true, "valueOf": true, "localeCompare": true,
}

var jsvArrayMethods = map[string]bool{
	"push": true, "pop": true, "shift": true, "unshift": true, "slice": true,
	"splice": true, "indexOf": true, "lastIndexOf": true, "includes": true,
	"join": true, "concat": true, "reverse": true, "sort": true, "map": true,
	"filter": true, "forEach": true, "reduce": true, "reduceRight": true,
	"some": true, "every": true, "find": true, "findIndex": true,
	"findLast": true, "findLastIndex": true, "flat": true, "flatMap": true,
	"fill": true, "at": true, "toString": true,
}

func jsvHidesMember(target interface{}, name string) bool {
	switch target.(type) {
	case string:
		return jsvStringMethods[name]
	case *jsArray:
		return jsvArrayMethods[name]
	}
	return false
}

// jsvRelIndex resolves a possibly negative / out-of-range index against a length,
// the way slice/splice/fill do (ECMA RelativeIndex).
func jsvRelIndex(v float64, length int) int {
	i := 0
	if math.IsNaN(v) {
		v = 0
	}
	if v < 0 {
		i = length + int(math.Trunc(v))
		if i < 0 {
			i = 0
		}
	} else {
		i = int(math.Trunc(v))
		if i > length {
			i = length
		}
	}
	return i
}

func (rt *jsrt) jsvNumArg(args []interface{}, i int, dflt float64) float64 {
	if i >= len(args) || isUndefOrNull(args[i]) {
		return dflt
	}
	return rt.toNumber(args[i])
}

// jsvToFixed is Number#toFixed. The rounding is spelled the same way the
// interpreter half spells it (scale by 10^digits, add a half, floor), because the
// two halves must agree with each other before either agrees with node - and on
// every value tested this IS node's answer: the spec rounds the EXACT binary value
// half up, and 1.005 * 100 is 100.4999... in binary, so both say "1.00".
func jsvToFixed(x float64, digits int) string {
	if math.IsNaN(x) {
		return "NaN"
	}
	if math.IsInf(x, 0) || math.Abs(x) >= 1e21 {
		return jsNumString(x)
	}
	sign := ""
	if x < 0 {
		sign, x = "-", -x
	}
	scale := math.Pow(10, float64(digits))
	n := math.Floor(x*scale + 0.5)
	s := strconv.FormatFloat(n, 'f', 0, 64)
	if digits > 0 {
		for len(s) <= digits {
			s = "0" + s
		}
		s = s[:len(s)-digits] + "." + s[len(s)-digits:]
	}
	if sign == "-" && n == 0 {
		return s
	}
	return sign + s
}

// jsvRadixString is Number#toString(radix) for a radix other than 10. The integer
// part is exact; the fraction is emitted to 52 digits and trailing zeros dropped,
// which is what V8 does for every value a program is likely to print.
func jsvRadixString(x float64, radix int) string {
	if math.IsNaN(x) {
		return "NaN"
	}
	if math.IsInf(x, 1) {
		return "Infinity"
	}
	if math.IsInf(x, -1) {
		return "-Infinity"
	}
	sign := ""
	if x < 0 {
		sign, x = "-", -x
	}
	ip := math.Floor(x)
	frac := x - ip
	digits := "0123456789abcdefghijklmnopqrstuvwxyz"
	intPart := ""
	if ip == 0 {
		intPart = "0"
	}
	for ip >= 1 {
		d := int(math.Mod(ip, float64(radix)))
		intPart = string(digits[d]) + intPart
		ip = math.Floor(ip / float64(radix))
	}
	if frac == 0 {
		return sign + intPart
	}
	out := intPart + "."
	for i := 0; i < 52 && frac > 0; i++ {
		frac *= float64(radix)
		d := int(math.Floor(frac))
		out += string(digits[d])
		frac -= float64(d)
	}
	out = strings.TrimRight(out, "0")
	out = strings.TrimSuffix(out, ".")
	return sign + out
}

// jsvSortDefault is Array#sort with no comparator: undefined sorts last and every
// other element compares as its STRING form. V8's sort is stable, so a stable sort
// is what both halves implement.
func (rt *jsrt) jsvSort(elems []interface{}, cmp interface{}) {
	undef := 0
	for i := 0; i < len(elems); i++ {
		if _, isU := elems[i].(jsUndefT); isU {
			undef++
		}
	}
	vals := make([]interface{}, 0, len(elems))
	for _, e := range elems {
		if _, isU := e.(jsUndefT); !isU {
			vals = append(vals, e)
		}
	}
	less := func(i, j int) bool {
		if isCallable(cmp) {
			return rt.toNumber(rt.call(cmp, jsUndef, []interface{}{vals[i], vals[j]})) < 0
		}
		return rt.jsvString(vals[i]) < rt.jsvString(vals[j])
	}
	sort.SliceStable(vals, less)
	copy(elems, vals)
	for i := len(vals); i < len(elems); i++ {
		elems[i] = jsUndef
	}
	_ = undef
}

func (rt *jsrt) jsvFlatten(out *[]interface{}, elems []interface{}, depth int) {
	for _, e := range elems {
		if arr, ok := e.(*jsArray); ok && depth > 0 {
			rt.jsvFlatten(out, arr.elems, depth-1)
			continue
		}
		*out = append(*out, e)
	}
}

// jsvMethod is the Go twin of `jsBuiltin` in languages/js-interpreter.abnf.
func (rt *jsrt) jsvMethod(target interface{}, name string, args []interface{}) (interface{}, bool) {
	// Function.prototype.bind. jsrt.go's getMember/builtinMethod already answer
	// `call` and `apply` for every language, but `bind` is not in that pair and
	// `f.bind(o)` aborted both compiled halves with "unknown method 'bind' on
	// function" while a decorator written with call() worked. It is answered HERE,
	// js/ts-locally, rather than beside call/apply in the shared runtime: the other
	// fifteen languages have no Function.prototype and must not grow one. The bound
	// arguments are PREPENDED to the eventual call's, which is the one way bind
	// differs from call beyond remembering the receiver. The twins are fnBindShim in
	// both *-interpreter.abnf start scripts and jvBindShim in lib/js-rt.metajs.
	if name == "bind" && isCallable(target) {
		self := argAt(args, 0)
		var pre []interface{}
		if len(args) > 1 {
			pre = append([]interface{}{}, args[1:]...)
		}
		fn := target
		bound := jsHostFunc("bound", func(rt *jsrt, this uint64, later []interface{}) interface{} {
			return rt.call(fn, self, append(append([]interface{}{}, pre...), later...))
		})
		// A bound function's own name is "bound " + the target's, and its length is the
		// target's less the arguments already supplied, floored at 0 - node says
		// "bound f"/1 for `f.bind(null)` of `function f(a){}`. docs/todo.md 1.4.
		bl := jsvFnMeta(fn).length - len(pre)
		if bl < 0 {
			bl = 0
		}
		jsvFnMetaSet(bound, "bound "+jsvFnMeta(fn).name, bl)
		return bound, true
	}
	// BigInt.prototype (abnf/jsrtjsbig.go): (255n).toString(16), (1n).valueOf().
	if bi, ok := target.(*jsBigInt); ok {
		if v, handled := rt.jsvBigIntMethod(bi.v, name, args); handled {
			return v, true
		}
	}
	// Promise.prototype.then / catch / finally. The names are dispatched here rather
	// than stored on the promise object so that a promise keeps no own keys at all.
	if promIs(target) {
		if v, handled := rt.promMethod(target.(*jsObject), name, args); handled {
			return v, true
		}
	}
	// Promise.resolve / reject / all / allSettled / race / any on the `Promise`
	// global, which is a function value, so js_get finds no member of that name.
	if hf, ok := target.(*hostFunc); ok && rt.promFn != nil && hf == rt.promFn {
		if v, handled := rt.promStatic(name, args); handled {
			return v, true
		}
	}
	// Object.keys / values / entries / assign / freeze / fromEntries.
	if jsvIsObjectGlobal(target) {
		switch name {
		case "keys":
			out := &jsArray{}
			for _, k := range jsvOwnKeys(argAt(args, 0)) {
				out.elems = append(out.elems, k)
			}
			return out, true
		case "values":
			out := &jsArray{}
			src := argAt(args, 0)
			for _, k := range jsvOwnKeys(src) {
				out.elems = append(out.elems, rt.jsvGetKey(src, k))
			}
			return out, true
		case "entries":
			out := &jsArray{}
			src := argAt(args, 0)
			for _, k := range jsvOwnKeys(src) {
				out.elems = append(out.elems, &jsArray{elems: []interface{}{k, rt.jsvGetKey(src, k)}})
			}
			return out, true
		case "getOwnPropertyNames":
			out := &jsArray{}
			for _, k := range jsvGopnKeys(argAt(args, 0)) {
				out.elems = append(out.elems, k)
			}
			return out, true
		case "assign":
			dst, ok := argAt(args, 0).(*jsObject)
			if !ok {
				rt.fail("Object.assign needs an object target")
			}
			for _, src := range args[1:] {
				if isUndefOrNull(src) {
					continue
				}
				for _, k := range jsvOwnKeys(src) {
					dst.set(k, rt.jsvGetKey(src, k))
				}
			}
			return dst, true
		case "fromEntries":
			out := newJSObject()
			if arr, ok := argAt(args, 0).(*jsArray); ok {
				for _, e := range arr.elems {
					pair, isArr := e.(*jsArray)
					if !isArr || len(pair.elems) == 0 {
						continue
					}
					out.set(rt.jsvString(pair.elems[0]), argAt(pair.elems, 1))
				}
			}
			return out, true
		case "freeze", "seal":
			return argAt(args, 0), true
		case "is":
			return rt.strictEq(argAt(args, 0), argAt(args, 1)), true
		}
		return nil, false
	}

	// String.fromCharCode / String.raw on the `String` global (a host function, so
	// js_get finds no member of that name and the call lands here).
	if hf, ok := target.(*hostFunc); ok && hf.name == "String" {
		switch name {
		case "fromCharCode":
			units := make([]uint16, len(args))
			for i, x := range args {
				units[i] = uint16(jsToInt(rt.toNumber(x)))
			}
			return strFromUnits(units), true
		}
		return nil, false
	}

	switch recv := target.(type) {
	case string:
		// A pattern argument belongs to the regular-expression surface
		// (abnf/jsrtregexjs.go), which js_jsmcall falls through to.
		if len(args) > 0 && rxIsRegexObj(args[0]) {
			return nil, false
		}
		return rt.jsvStringMethod(recv, name, args)
	case *jsArray:
		return rt.jsvArrayMethod(recv, name, args)
	case float64:
		return rt.jsvNumberMethod(recv, name, args)
	}
	return nil, false
}

func (rt *jsrt) jsvStringMethod(s string, name string, args []interface{}) (interface{}, bool) {
	units := rt.strLen(s)
	argS := func(i int) string {
		if i < len(args) {
			return rt.jsvString(args[i])
		}
		return "undefined"
	}
	switch name {
	case "toString", "valueOf":
		return s, true
	case "charAt":
		return rt.strAt(s, jsToInt(rt.jsvNumArg(args, 0, 0))), true
	case "charCodeAt":
		if c := rt.strCodeAt(s, jsToInt(rt.jsvNumArg(args, 0, 0))); c >= 0 {
			return float64(c), true
		}
		return math.NaN(), true
	case "codePointAt":
		i := jsToInt(rt.jsvNumArg(args, 0, 0))
		if i < 0 || i >= units {
			return jsUndef, true
		}
		hi := rt.strCodeAt(s, i)
		// A surrogate PAIR is one code point; the string is measured in UTF-16 code
		// units, so the low half has to be folded in by hand.
		if hi >= 0xd800 && hi <= 0xdbff && i+1 < units {
			if lo := rt.strCodeAt(s, i+1); lo >= 0xdc00 && lo <= 0xdfff {
				return float64((hi-0xd800)*0x400 + (lo - 0xdc00) + 0x10000), true
			}
		}
		return float64(hi), true
	case "at":
		i := jsToInt(rt.jsvNumArg(args, 0, 0))
		if i < 0 {
			i += units
		}
		if i < 0 || i >= units {
			return jsUndef, true
		}
		return rt.strRange(s, i, i+1), true
	case "indexOf":
		from := jsToInt(rt.jsvNumArg(args, 1, 0))
		if from < 0 {
			from = 0
		}
		if from > units {
			from = units
		}
		sub := argS(0)
		at := rt.strIndexOf(rt.strRange(s, from, units), sub)
		if at < 0 {
			return float64(-1), true
		}
		return float64(at + from), true
	case "lastIndexOf":
		sub := argS(0)
		best := -1
		subLen := rt.strLen(sub)
		for i := 0; i+subLen <= units; i++ {
			if rt.strRange(s, i, i+subLen) == sub {
				best = i
			}
		}
		return float64(best), true
	case "includes":
		return rt.strIndexOf(s, argS(0)) >= 0, true
	case "startsWith":
		sub := argS(0)
		at := jsToInt(rt.jsvNumArg(args, 1, 0))
		if at < 0 {
			at = 0
		}
		return at+rt.strLen(sub) <= units && rt.strRange(s, at, at+rt.strLen(sub)) == sub, true
	case "endsWith":
		sub := argS(0)
		end := units
		if len(args) > 1 && !isUndefOrNull(args[1]) {
			end = jsToInt(rt.toNumber(args[1]))
		}
		start := end - rt.strLen(sub)
		return start >= 0 && end <= units && rt.strRange(s, start, end) == sub, true
	case "slice":
		begin, end := sliceRange(units, args, rt)
		return rt.strRange(s, begin, end), true
	case "substring":
		begin, end := substringRange(units, args, rt)
		return rt.strRange(s, begin, end), true
	case "substr":
		start := jsToInt(rt.jsvNumArg(args, 0, 0))
		if start < 0 {
			start += units
			if start < 0 {
				start = 0
			}
		}
		if start > units {
			start = units
		}
		n := units - start
		if len(args) > 1 && !isUndefOrNull(args[1]) {
			n = jsToInt(rt.toNumber(args[1]))
		}
		if n < 0 {
			n = 0
		}
		if start+n > units {
			n = units - start
		}
		return rt.strRange(s, start, start+n), true
	case "split":
		out := &jsArray{}
		limit := -1
		if len(args) > 1 && !isUndefOrNull(args[1]) {
			limit = jsToInt(rt.toNumber(args[1]))
		}
		if len(args) == 0 || isUndefOrNull(args[0]) {
			if limit != 0 {
				out.elems = append(out.elems, s)
			}
			return out, true
		}
		sep := rt.jsvString(args[0])
		var parts []string
		if sep == "" {
			for i := 0; i < units; i++ {
				parts = append(parts, rt.strRange(s, i, i+1))
			}
		} else {
			parts = strings.Split(s, sep)
		}
		for _, p := range parts {
			if limit >= 0 && len(out.elems) >= limit {
				break
			}
			out.elems = append(out.elems, p)
		}
		return out, true
	case "toUpperCase":
		return strings.ToUpper(s), true
	case "toLowerCase":
		return strings.ToLower(s), true
	case "trim":
		return strings.Trim(s, jsvSpace), true
	case "trimStart":
		return strings.TrimLeft(s, jsvSpace), true
	case "trimEnd":
		return strings.TrimRight(s, jsvSpace), true
	case "padStart", "padEnd":
		want := jsToInt(rt.jsvNumArg(args, 0, 0))
		fill := " "
		if len(args) > 1 && !isUndefOrNull(args[1]) {
			fill = rt.jsvString(args[1])
		}
		if want <= units || fill == "" {
			return s, true
		}
		pad := ""
		for rt.strLen(pad) < want-units {
			pad += fill
		}
		pad = rt.strRange(pad, 0, want-units)
		if name == "padStart" {
			return pad + s, true
		}
		return s + pad, true
	case "repeat":
		n := jsToInt(rt.jsvNumArg(args, 0, 0))
		if n < 0 {
			rt.fail("Invalid count value: %d", n)
		}
		return strings.Repeat(s, n), true
	case "concat":
		out := s
		for i := range args {
			out += argS(i)
		}
		return out, true
	case "replace", "replaceAll":
		pat := argS(0)
		n := 1
		if name == "replaceAll" {
			n = -1
		}
		if isCallable(argAt(args, 1)) {
			return rt.jsvReplaceFn(s, pat, args[1], n), true
		}
		return jsvReplaceTemplate(rt, s, pat, argS(1), n), true
	case "localeCompare":
		other := argS(0)
		switch {
		case s < other:
			return float64(-1), true
		case s > other:
			return float64(1), true
		}
		return float64(0), true
	}
	return nil, false
}

// jsvSpace is the character set String#trim removes (WhiteSpace + LineTerminator).
const jsvSpace = "\t\n\v\f\r \u00a0\u1680\u2000\u2001\u2002\u2003\u2004\u2005" +
	"\u2006\u2007\u2008\u2009\u200a\u2028\u2029\u202f\u205f\u3000\ufeff"

// jsvReplaceFn is the replacer-function form: the callback gets (match, offset, whole).
func (rt *jsrt) jsvReplaceFn(s, pat string, fn interface{}, limit int) string {
	if pat == "" {
		return rt.toString(rt.call(fn, jsUndef, []interface{}{"", float64(0), s})) + s
	}
	out, at, done := "", 0, 0
	for {
		i := strings.Index(s[at:], pat)
		if i < 0 || (limit >= 0 && done >= limit) {
			break
		}
		i += at
		out += s[at:i]
		out += rt.toString(rt.call(fn, jsUndef, []interface{}{pat, float64(rt.strLen(s[:i])), s}))
		at = i + len(pat)
		done++
	}
	return out + s[at:]
}

// jsvReplaceTemplate expands the $ patterns of a replacement string ($& the match,
// $` the prefix, $' the suffix, $$ a literal dollar).
func jsvReplaceTemplate(rt *jsrt, s, pat, repl string, limit int) string {
	if pat == "" {
		return jsvExpandDollar(repl, "", "", s) + s
	}
	out, at, done := "", 0, 0
	for {
		i := strings.Index(s[at:], pat)
		if i < 0 || (limit >= 0 && done >= limit) {
			break
		}
		i += at
		out += s[at:i]
		out += jsvExpandDollar(repl, pat, s[:i], s[i+len(pat):])
		at = i + len(pat)
		done++
	}
	return out + s[at:]
}

func jsvExpandDollar(repl, match, before, after string) string {
	if !strings.ContainsRune(repl, '$') {
		return repl
	}
	var out strings.Builder
	for i := 0; i < len(repl); i++ {
		if repl[i] != '$' || i+1 >= len(repl) {
			out.WriteByte(repl[i])
			continue
		}
		switch repl[i+1] {
		case '$':
			out.WriteByte('$')
		case '&':
			out.WriteString(match)
		case '`':
			out.WriteString(before)
		case '\'':
			out.WriteString(after)
		default:
			out.WriteByte(repl[i])
			continue
		}
		i++
	}
	return out.String()
}

func (rt *jsrt) jsvArrayMethod(a *jsArray, name string, args []interface{}) (interface{}, bool) {
	n := len(a.elems)
	cb := argAt(args, 0)
	call3 := func(i int) interface{} {
		return rt.call(cb, argAt(args, 1), []interface{}{a.elems[i], float64(i), a})
	}
	switch name {
	case "toString":
		return rt.jsvString(a), true
	case "at":
		i := jsToInt(rt.jsvNumArg(args, 0, 0))
		if i < 0 {
			i += n
		}
		if i < 0 || i >= n {
			return jsUndef, true
		}
		return a.elems[i], true
	case "push":
		a.dropIdx()
		a.elems = append(a.elems, args...)
		return float64(len(a.elems)), true
	case "pop":
		if n == 0 {
			return jsUndef, true
		}
		v := a.elems[n-1]
		a.dropIdx()
		a.elems = a.elems[:n-1]
		return v, true
	case "shift":
		if n == 0 {
			return jsUndef, true
		}
		v := a.elems[0]
		a.dropIdx()
		a.elems = append([]interface{}{}, a.elems[1:]...)
		return v, true
	case "unshift":
		a.dropIdx()
		a.elems = append(append([]interface{}{}, args...), a.elems...)
		return float64(len(a.elems)), true
	case "reverse":
		a.dropIdx()
		for i, j := 0, n-1; i < j; i, j = i+1, j-1 {
			a.elems[i], a.elems[j] = a.elems[j], a.elems[i]
		}
		return a, true
	case "slice":
		begin, end := sliceRange(n, args, rt)
		out := &jsArray{}
		for i := begin; i < end; i++ {
			out.elems = append(out.elems, a.elems[i])
		}
		return out, true
	case "splice":
		start := n
		if len(args) > 0 {
			start = jsvRelIndex(rt.toNumber(args[0]), n)
		}
		del := n - start
		if len(args) > 1 {
			del = jsToInt(rt.toNumber(args[1]))
		}
		if del < 0 {
			del = 0
		}
		if start+del > n {
			del = n - start
		}
		removed := &jsArray{elems: append([]interface{}{}, a.elems[start:start+del]...)}
		var ins []interface{}
		if len(args) > 2 {
			ins = args[2:]
		}
		next := append([]interface{}{}, a.elems[:start]...)
		next = append(next, ins...)
		next = append(next, a.elems[start+del:]...)
		a.dropIdx()
		a.elems = next
		return removed, true
	case "indexOf":
		from := 0
		if len(args) > 1 {
			from = jsvRelIndex(rt.toNumber(args[1]), n)
		}
		for i := from; i < n; i++ {
			if rt.strictEq(a.elems[i], cb) {
				return float64(i), true
			}
		}
		return float64(-1), true
	case "lastIndexOf":
		for i := n - 1; i >= 0; i-- {
			if rt.strictEq(a.elems[i], cb) {
				return float64(i), true
			}
		}
		return float64(-1), true
	case "includes":
		for _, e := range a.elems {
			if rt.strictEq(e, cb) {
				return true, true
			}
			// includes, unlike indexOf, finds NaN.
			x, isX := e.(float64)
			y, isY := cb.(float64)
			if isX && isY && math.IsNaN(x) && math.IsNaN(y) {
				return true, true
			}
		}
		return false, true
	case "join":
		sep := ","
		if len(args) > 0 && !isUndefOrNull(args[0]) {
			sep = rt.jsvString(args[0])
		}
		parts := make([]string, n)
		for i, e := range a.elems {
			if !isUndefOrNull(e) {
				parts[i] = rt.jsvString(e)
			}
		}
		return strJoin(parts, sep), true
	case "concat":
		out := &jsArray{elems: append([]interface{}{}, a.elems...)}
		for _, x := range args {
			if xa, ok := x.(*jsArray); ok {
				out.elems = append(out.elems, xa.elems...)
			} else {
				out.elems = append(out.elems, x)
			}
		}
		return out, true
	case "sort":
		a.dropIdx()
		rt.jsvSort(a.elems, cb)
		return a, true
	case "map":
		out := &jsArray{}
		for i := 0; i < n && i < len(a.elems); i++ {
			out.elems = append(out.elems, call3(i))
		}
		return out, true
	case "filter":
		out := &jsArray{}
		for i := 0; i < n && i < len(a.elems); i++ {
			if rt.truthy(call3(i)) {
				out.elems = append(out.elems, a.elems[i])
			}
		}
		return out, true
	case "forEach":
		for i := 0; i < n && i < len(a.elems); i++ {
			call3(i)
		}
		return jsUndef, true
	case "some":
		for i := 0; i < n && i < len(a.elems); i++ {
			if rt.truthy(call3(i)) {
				return true, true
			}
		}
		return false, true
	case "every":
		for i := 0; i < n && i < len(a.elems); i++ {
			if !rt.truthy(call3(i)) {
				return false, true
			}
		}
		return true, true
	case "find":
		for i := 0; i < n && i < len(a.elems); i++ {
			if rt.truthy(call3(i)) {
				return a.elems[i], true
			}
		}
		return jsUndef, true
	case "findIndex":
		for i := 0; i < n && i < len(a.elems); i++ {
			if rt.truthy(call3(i)) {
				return float64(i), true
			}
		}
		return float64(-1), true
	case "findLast":
		for i := n - 1; i >= 0; i-- {
			if rt.truthy(call3(i)) {
				return a.elems[i], true
			}
		}
		return jsUndef, true
	case "findLastIndex":
		for i := n - 1; i >= 0; i-- {
			if rt.truthy(call3(i)) {
				return float64(i), true
			}
		}
		return float64(-1), true
	case "reduce", "reduceRight":
		idx := make([]int, 0, n)
		if name == "reduce" {
			for i := 0; i < n; i++ {
				idx = append(idx, i)
			}
		} else {
			for i := n - 1; i >= 0; i-- {
				idx = append(idx, i)
			}
		}
		at := 0
		var acc interface{}
		if len(args) > 1 {
			acc = args[1]
		} else {
			if n == 0 {
				rt.fail("Reduce of empty array with no initial value")
			}
			acc = a.elems[idx[0]]
			at = 1
		}
		for ; at < len(idx); at++ {
			i := idx[at]
			acc = rt.call(cb, jsUndef, []interface{}{acc, a.elems[i], float64(i), a})
		}
		return acc, true
	case "flat":
		depth := 1
		if len(args) > 0 && !isUndefOrNull(args[0]) {
			d := rt.toNumber(args[0])
			if math.IsInf(d, 1) {
				depth = 1 << 30
			} else {
				depth = jsToInt(d)
			}
		}
		out := []interface{}{}
		rt.jsvFlatten(&out, a.elems, depth)
		return &jsArray{elems: out}, true
	case "flatMap":
		out := []interface{}{}
		for i := 0; i < n && i < len(a.elems); i++ {
			v := call3(i)
			if va, ok := v.(*jsArray); ok {
				out = append(out, va.elems...)
			} else {
				out = append(out, v)
			}
		}
		return &jsArray{elems: out}, true
	case "fill":
		v := argAt(args, 0)
		begin, end := 0, n
		if len(args) > 1 && !isUndefOrNull(args[1]) {
			begin = jsvRelIndex(rt.toNumber(args[1]), n)
		}
		if len(args) > 2 && !isUndefOrNull(args[2]) {
			end = jsvRelIndex(rt.toNumber(args[2]), n)
		}
		for i := begin; i < end; i++ {
			a.elems[i] = v
		}
		return a, true
	}
	return nil, false
}

func (rt *jsrt) jsvNumberMethod(x float64, name string, args []interface{}) (interface{}, bool) {
	switch name {
	case "toFixed":
		return jsvToFixed(x, jsToInt(rt.jsvNumArg(args, 0, 0))), true
	case "toString":
		radix := jsToInt(rt.jsvNumArg(args, 0, 10))
		if radix == 10 {
			return jsNumString(x), true
		}
		if radix < 2 || radix > 36 {
			// node makes an out-of-range radix a CATCHABLE RangeError, and this is
			// the number path - the BigInt path beside it (jsrtjsbig.go) has raised
			// one all along, so the same engine answered the same mistake two
			// different ways. Reachable now that the hierarchy exists.
			rt.bigRaise("RangeError: toString() radix must be between 2 and 36")
		}
		return jsvRadixString(x, radix), true
	case "valueOf":
		return x, true
	}
	return nil, false
}

// ============================================================================
// The externs

func (rt *jsrt) addJSValueExterns(m map[string]func(args []uint64) uint64) {
	u := rt.unwrap
	w := rt.wrap
	boolH := func(b bool) uint64 {
		if b {
			return jsHTrue
		}
		return jsHFalse
	}

	// println with JAVASCRIPT's ToString. The shared println host function hands the
	// raw value to Go's %v, which is where "<nil>", "[1 2 3]" and "map[a:1]" came
	// from. Multiple arguments stay space-joined, each stringified on its own.
	m["js_jsputs"] = func(a []uint64) uint64 {
		args, ok := u(a[0]).(*jsArray)
		if !ok {
			rt.fail("js_jsputs needs an argument array")
		}
		out := ""
		for i, e := range args.elems {
			if i > 0 {
				out += " "
			}
			out += rt.jsvString(e)
		}
		fmt.Fprintln(outWriter, wtf8Clean(out))
		return 0
	}
	// print: the same rendering without the newline and without the separator, which
	// is what Go's Fprint does for a run of strings.
	m["js_jsprint"] = func(a []uint64) uint64 {
		args, ok := u(a[0]).(*jsArray)
		if !ok {
			rt.fail("js_jsprint needs an argument array")
		}
		out := ""
		for _, e := range args.elems {
			out += rt.jsvString(e)
		}
		fmt.Fprint(outWriter, wtf8Clean(out))
		return 0
	}
	// String(v) / the ToString an interpolation needs.
	m["js_jsstr"] = func(a []uint64) uint64 { return rt.wrapStr(rt.jsvString(u(a[0]))) }

	// js_jstypeof is `typeof` for js/ts, and the twin of the layer-2 function of
	// the same name. It used to be reached under -exe only, because the one thing
	// it added there - "bigint" for layer 2's object box - the Go twin already
	// answered from the Go type. It is now the extern in BOTH compiled halves,
	// because a CLASS DESCRIPTOR IS A FUNCTION to `typeof`: every class here is
	// an ordinary object carrying __isclass, so `typeof K` answered "object"
	// where node says "function" for every class there has ever been. The test
	// stays out of typeOf itself, which is shared by all sixteen languages, and
	// out of the engine's own `is this callable` guards, which decide whether a
	// value can be CALLED - a class descriptor still cannot. docs/todo.md 1.8.
	m["js_jstypeof"] = func(a []uint64) uint64 {
		v := u(a[0])
		if o, ok := v.(*jsObject); ok {
			if c, has := o.props["__isclass"]; has && c == interface{}(true) {
				return rt.wrapStr("function")
			}
		}
		return rt.wrapStr(rt.typeOf(v))
	}

	// js_jsprintfn("println"|"print") answers the host FUNCTION VALUE the grammar
	// declares over the root scope's binding of the same name. Going through a
	// binding rather than through a call-site rewrite is what keeps shadowing
	// honest: a program that declares a println of its own simply overwrites it.
	m["js_jsprintfn"] = func(a []uint64) uint64 {
		nl := rt.toString(u(a[0])) == "println"
		return w(jsHostFunc(rt.toString(u(a[0])), func(rt *jsrt, this uint64, args []interface{}) interface{} {
			parts := make([]string, len(args))
			for i, e := range args {
				parts[i] = rt.jsvString(e)
			}
			sep := ""
			if nl {
				sep = " "
			}
			out := wtf8Clean(strJoin(parts, sep))
			if nl {
				fmt.Fprintln(outWriter, out)
			} else {
				fmt.Fprint(outWriter, out)
			}
			return jsUndef
		}))
	}

	// '+' WITH ToPrimitive. The shared js_add renders an object with the neutral
	// rendering, so `"" + d` was "[object Object]" for a class that has a toString of
	// its own while the interpreter half answered the method. Two primitive operands
	// go straight to the shared extern, which is every addition a program that adds
	// no objects performs.
	baseAdd := m["js_add"]
	m["js_jsadd"] = func(a []uint64) uint64 {
		l, r := u(a[0]), u(a[1])
		if !jsvIsObjLike(l) && !jsvIsObjLike(r) {
			return baseAdd(a)
		}
		lp, rp := rt.jsvToPrimitive(l), rt.jsvToPrimitive(r)
		ls, lIsStr := lp.(string)
		rs, rIsStr := rp.(string)
		if lIsStr || rIsStr {
			if !lIsStr {
				ls = rt.jsvString(lp)
			}
			if !rIsStr {
				rs = rt.jsvString(rp)
			}
			return rt.wrapStr(ls + rs)
		}
		return rt.wrapNum(rt.toNumber(lp) + rt.toNumber(rp))
	}

	// The `String` global. The runtime root binds an OBJECT (it carries
	// fromCharCode), so a bare String(v) call aborted with "call of a non function
	// value" in the compiler and worked only under goja in the interpreter. This is a
	// real host function, and String.fromCharCode is answered by js_jsmcall below -
	// which is reached because js_get has no member of that name on a function value.
	m["js_jsstringfn"] = func(a []uint64) uint64 {
		return w(jsHostFunc("String", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			if len(args) == 0 {
				return ""
			}
			return rt.jsvString(args[0])
		}))
	}

	// The equality and relational operators WITH ToPrimitive.
	m["js_jseq"] = func(a []uint64) uint64 { return boolH(rt.jsvLooseEq(u(a[0]), u(a[1]))) }
	m["js_jsne"] = func(a []uint64) uint64 { return boolH(!rt.jsvLooseEq(u(a[0]), u(a[1]))) }
	m["js_jslt"] = func(a []uint64) uint64 { return boolH(rt.jsvCompare(u(a[0]), u(a[1])) == -1) }
	m["js_jsgt"] = func(a []uint64) uint64 { return boolH(rt.jsvCompare(u(a[0]), u(a[1])) == 1) }
	m["js_jsle"] = func(a []uint64) uint64 {
		c := rt.jsvCompare(u(a[0]), u(a[1]))
		return boolH(c == -1 || c == 0)
	}
	m["js_jsge"] = func(a []uint64) uint64 {
		c := rt.jsvCompare(u(a[0]), u(a[1]))
		return boolH(c == 1 || c == 0)
	}

	// 'new F()' for a plain constructor FUNCTION: its instances need a class
	// descriptor to answer instanceof with, exactly as the interpreter half's
	// ctorDescriptor gives them one. Memoized on the function's identity, so two
	// instances of the same constructor share a descriptor.
	ctorDescs := map[interface{}]*jsObject{}
	descOf := func(fn interface{}) *jsObject {
		if d, ok := ctorDescs[fn]; ok {
			return d
		}
		d := newJSObject()
		d.set("__isclass", true)
		d.set("__ctorfn", fn)
		ctorDescs[fn] = d
		return d
	}
	m["js_jsctordesc"] = func(a []uint64) uint64 { return w(descOf(u(a[0]))) }
	// [[Construct]]: an object the constructor RETURNS wins over the fresh receiver.
	m["js_jsnewres"] = func(a []uint64) uint64 {
		r := u(a[0])
		if jsvIsObjLike(r) || isCallable(r) {
			return a[0]
		}
		return a[1]
	}

	baseInstanceof := m["js_instanceof"]
	m["js_jsinstanceof"] = func(a []uint64) uint64 {
		v, target := u(a[0]), u(a[1])
		if jsvIsObjectGlobal(target) {
			return boolH(jsvIsObjLike(v) || isCallable(v))
		}
		if jsvIsArrayGlobal(target) {
			_, isArr := v.(*jsArray)
			return boolH(isArr)
		}
		if isCallable(target) {
			d, ok := ctorDescs[target]
			if !ok {
				return boolH(false)
			}
			o, isObj := v.(*jsObject)
			if !isObj {
				return boolH(false)
			}
			for cls := o.props["__class"]; cls != nil; {
				if cls == interface{}(d) {
					return boolH(true)
				}
				clsObj, isO := cls.(*jsObject)
				if !isO {
					break
				}
				cls = clsObj.props["__super"]
			}
			return boolH(false)
		}
		return baseInstanceof(a)
	}

	// 'delete a[i]' on an ARRAY: the length is unchanged and the slot reads as
	// undefined. (A real JS hole cannot be modelled here: the interpreter half's
	// arrays are host arrays that carry no side data, so both halves agree on
	// blanking the slot - which is what the interpreter has always done.)
	baseDel := m["js_del"]
	m["js_jsdel"] = func(a []uint64) uint64 {
		if arr, ok := u(a[0]).(*jsArray); ok {
			if maybeNumeric(u(a[1])) {
				idx := rt.toNumber(u(a[1]))
				if idx == math.Trunc(idx) && idx >= 0 && int(idx) < len(arr.elems) {
					arr.elems[int(idx)] = jsUndef
				}
			}
			return boolH(true)
		}
		// A function's own properties live in the side table. docs/todo.md 2.10.
		if p := jsvFnPropsOf(u(a[0])); p != nil {
			name := rt.toString(u(a[1]))
			if _, found := p.props[name]; found {
				delete(p.props, name)
				for i, k := range p.keys {
					if k == name {
						p.keys = append(p.keys[:i], p.keys[i+1:]...)
						break
					}
				}
			}
			return boolH(true)
		}
		return baseDel(a)
	}

	// The `Object` global, declared by the grammar in the PROGRAM's own scope so no
	// other language sees it. It keeps the root binding's prototype, so the
	// Object.prototype.hasOwnProperty idiom still resolves.
	m["js_jsobject"] = func(a []uint64) uint64 {
		o := newJSObject()
		o.set("__jsobject", true)
		// Object.prototype.toString.call(v) is the classic type probe, and it used to
		// abort BOTH compiled halves: the root binding's `prototype` carries no
		// methods, so llvm.Run said "member 'call' of undefined" and the native build,
		// whose layer-2 js_jsobject had no prototype slot at all, said "member
		// 'toString' of undefined". The three members are the ones a program in this
		// subset can reach; each reads its receiver from the call's `this`, which is
		// what Function.prototype.call passes. Twins: js_jsobject in
		// languages/lib/js-rt.metajs and jsObjectProto in both *-interpreter.abnf
		// start scripts. docs/todo.md 2.5.
		proto := newJSObject()
		proto.set("toString", jsHostFunc("toString", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return jsvProtoTag(u(this))
		}))
		proto.set("valueOf", jsHostFunc("valueOf", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return u(this)
		}))
		proto.set("hasOwnProperty", jsHostFunc("hasOwnProperty", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			// A FUNCTION's own properties live in the side table. docs/todo.md 2.10.
			if p := jsvFnPropsOf(u(this)); p != nil {
				_, has := p.props[rt.toString(argAt(args, 0))]
				return has
			}
			self, isObj := u(this).(*jsObject)
			if !isObj {
				return false
			}
			_, has := self.props[rt.toString(argAt(args, 0))]
			return has
		}))
		o.set("prototype", proto)
		return w(o)
	}

	// js_jsmget hides the String/Array method names from js_get, so that the CALL
	// site falls into js_jsmcall below instead of the shared boundMethod path, whose
	// semantics differ for several of them.
	// ----- CLOSING AN ITERATOR ON AN EARLY EXIT (docs/todo.md 1.8) -----
	//
	// node closes a for-of's iterator on every abrupt exit from the loop body, not
	// just on `break`: a `return` out of the body and a labeled break to an OUTER
	// statement run the iterator's return() too, and neither reaches a block
	// makeForOf owns - one rets the frame, the other branches to the outer label's
	// exit. The emitter closes them BY VALUE instead: a for-of's entry block
	// dominates its body, so the loop's iterable handle is a legal operand at every
	// jump inside it, and one of these is emitted per loop being left.
	//
	// There is deliberately NO runtime open-iterator stack here, and that is a
	// finding rather than an omission - see the forOfIters note in
	// languages/js-to-llvm-ir.abnf. A stack unwound by DEPTH is what a `throw`
	// would need, and `for await` suspends INSIDE its own for-of, so a suspended
	// body leaves an entry on such a stack while unrelated frames run; measured, it
	// closed the wrong loop.
	//
	// The twins are js_jsiterclose in languages/lib/js-rt.metajs and closeIterator
	// in the two *-interpreter.abnf start scripts.
	m["js_jsiterclose"] = func(a []uint64) uint64 {
		it := u(a[0])
		if !isUndefOrNull(it) {
			if r := rt.getMember(it, "return"); isCallable(r) {
				rt.call(r, it, nil)
			}
		}
		return w(jsUndef)
	}

	// ----- Async generators (docs/todo.md 1.7) -----
	// js_jsawaitmark marks an awaited operand so that the one suspension channel an
	// async generator body has can carry both its yields and its awaits; js_jsasyncgen
	// wraps the generator FUNCTION the body compiled to into one that answers an
	// async-generator object. See the agRequest block at the end of this file.
	m["js_jsawaitmark"] = func(a []uint64) uint64 {
		mk := newJSObject()
		mk.set("__awt", true)
		mk.set("v", u(a[0]))
		return w(mk)
	}
	m["js_jsasyncgen"] = func(a []uint64) uint64 {
		genFn := u(a[0])
		return w(jsHostFunc("asyncgen", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			g, ok := rt.call(genFn, jsUndef, args).(*jsGenerator)
			if !ok {
				rt.fail("js_jsasyncgen needs a generator function")
			}
			return rt.agObject(g)
		}))
	}
	// The two RAW PREDICATES the call site uses to keep node's TypeErrors off the
	// hot path (docs/todo.md 1.4). They answer 0 or 1 as a plain integer, never a
	// handle, so the emitter can branch on the result with an icmp instead of
	// paying js_typeof + js_seq - the two-extern shape the method-call arm above
	// still uses. The layer-2 twins are js_jsfnp__raw / js_jsctorp__raw in
	// languages/lib/js-rt.metajs, and the `__raw` suffix there is what makes them
	// return the same plain integer instead of a boolean HANDLE (manual 7.1).
	m["js_jsfnp"] = func(a []uint64) uint64 {
		if isCallable(u(a[0])) {
			return 1
		}
		return 0
	}
	// `new X` needs a class DESCRIPTOR - the {__isclass, __ctor, ...} object every
	// emitted class is - and a non-constructor must not reach the __ctor read.
	m["js_jsctorp"] = func(a []uint64) uint64 {
		if o, ok := u(a[0]).(*jsObject); ok {
			if _, has := o.props["__isclass"]; has {
				return 1
			}
		}
		return 0
	}
	baseGet := m["js_get"]
	// js_jsvget is the VALUE read of a member - `const f = p.m`, `typeof p.m` - as
	// opposed to js_jsmget, which the emitter uses at a member-followed-by-a-call site
	// and which deliberately HIDES the String/Array method names so the call falls into
	// js_jsmcall. The difference this extern exists for: an instance's methods live on
	// its __class descriptor, not as own properties, so `typeof p.m` answered
	// "undefined" in all four engines where node says "function". The answer is
	// jsvMethodValue: one unbound function per method, which takes `this` from its
	// call site. The layer-2 twin is js_jsvget in languages/lib/js-rt.metajs and
	// the interpreters' is getMember's class-method arm. docs/todo.md 2.5.
	m["js_jsvget"] = func(a []uint64) uint64 {
		// A CLASS DESCRIPTOR reads from its static namespace and from nowhere else
		// (docs/todo.md 2.13). Ordered in front of baseGet because the descriptor's
		// own property map is where its INSTANCE methods live, and answering one of
		// those for `C.m` is the defect.
		if cls, isStatic := jsvIsStaticSide(u(a[0]), rt.toString(u(a[1]))); isStatic {
			return rt.jsvStaticSideRead(cls, rt.toString(u(a[1])))
		}
		// The FAST PATH must not re-wrap: the handle baseGet answered is returned
		// as it stands, and the fallback below runs only when the read found
		// nothing. The layer-2 twin pays the same attention and for a measured
		// reason - see js_jsvget in languages/lib/js-rt.metajs.
		r := baseGet(a)
		if !isUndefOrNull(u(r)) {
			return r
		}
		recv, isObj := u(a[0]).(*jsObject)
		if !isObj {
			// `f.name` and `f.length`, before the own-property side table because both
			// are non-writable in node: an `f.name = "q"` that reached the table must
			// not win, and js_jsmset drops it for the same reason. docs/todo.md 1.4.
			if v := jsvFnIntrinsic(u(a[0]), rt.toString(u(a[1]))); v != nil {
				return w(v)
			}
			// An own property written onto a FUNCTION lives in the side table, not
			// on the closure. docs/todo.md 2.10.
			if p := jsvFnPropsOf(u(a[0])); p != nil {
				if v, found := p.props[rt.toString(u(a[1]))]; found {
					return w(v)
				}
			}
			// ...and a function's Object.prototype surface, AFTER the side table so
			// an own `f.toString = ...` still wins. docs/todo.md 2.10.
			if m := rt.jsvFnMethod(u(a[0]), rt.toString(u(a[1]))); m != nil {
				return w(m)
			}
			return r
		}
		name := rt.toString(u(a[1]))
		if jsvInternalKey(name) {
			return r
		}
		// `e.constructor` is the CLASS an instance was built from. It answered
		// undefined in all four engines where node hands back the class object (and,
		// for `new F()`, the function F itself). docs/todo.md 2.10.
		if name == "constructor" {
			if c, found := jsvConstructorOf(recv); found {
				return w(c)
			}
		}
		if fn, found := jsvFindClassMethod(recv, name); found {
			return w(jsvMethodValue(name, fn))
		}
		return r
	}
	// js_has plus the __class methods: an instance keeps its methods on its class
	// descriptor rather than as own properties, so `"m" in instance` answered false
	// where node answers true. The layer-2 twin is js_jshas in
	// languages/lib/js-rt.metajs; the interpreters' is propIn. docs/todo.md 2.5.
	baseHas := m["js_has"]
	m["js_jshas"] = func(a []uint64) uint64 {
		recv, isObj := u(a[0]).(*jsObject)
		name := rt.toString(u(a[1]))
		// The __-prefixed engine slots are not properties of the program's object:
		// `"__class" in p` answered TRUE in both compiled halves and false in the
		// interpreters and in node. Measured on the way past, not part of the item.
		// ...but a mangled PRIVATE name IS one of the program's own, and `#x in o` -
		// the ergonomic brand check - asks for exactly that slot, so it falls through
		// to the own-slot and class-method arms below. docs/todo.md 2.4.
		if isObj && jsvInternalKey(name) && !jsvPrivateKey(name) {
			return w(false)
		}
		// `"m" in C` asks the STATIC namespace and nothing else, and it is asked
		// BEFORE js_has: the descriptor's own property map holds C's INSTANCE
		// methods, which node does not report as properties of the constructor.
		// docs/todo.md 2.13.
		if isObj && isClassDesc(recv) {
			if _, _, found := jsStaticsFind(recv, name); found {
				return w(true)
			}
			return w(name == "name")
		}
		// `"constructor" in e` is true in node for every instance, and the read
		// answers it now. docs/todo.md 2.10.
		if isObj && name == "constructor" {
			if _, found := jsvConstructorOf(recv); found {
				return w(true)
			}
		}
		// The side table holds a function's own properties; the key map is asked, so
		// a property holding undefined is still reported. A function that is not IN
		// the table answers false rather than falling through, because the shared
		// js_has aborts with "'in' needs an object on the right side" for any
		// receiver that is not an object. docs/todo.md 2.10.
		if jsvIsFnLike(u(a[0])) {
			if p := jsvFnPropsOf(u(a[0])); p != nil {
				if _, found := p.props[name]; found {
					return w(true)
				}
			}
			return w(false)
		}
		if r := baseHas(a); u(r) == true {
			return r
		}
		if isObj {
			if _, found := jsvFindClassMethod(recv, name); found {
				return w(true)
			}
		}
		return w(false)
	}
	m["js_jsmget"] = func(a []uint64) uint64 {
		if jsvHidesMember(u(a[0]), rt.toString(u(a[1]))) {
			return jsHUndefined
		}
		// The static namespace, exactly as in js_jsvget above (docs/todo.md 2.13).
		if cls, isStatic := jsvIsStaticSide(u(a[0]), rt.toString(u(a[1]))); isStatic {
			return rt.jsvStaticSideRead(cls, rt.toString(u(a[1])))
		}
		// then/catch/finally READ AS A VALUE. Promise method dispatch is
		// table-based (jsvMethod -> promMethod), so a promise has no own slot of
		// that name and `const t = p.then` answered undefined where node answers a
		// function. Answer a receiver-BOUND host function, which is what makes
		// `t.call(p, f)` and `p.then.bind(p)` work; the same arm exists in
		// languages/lib/js-rt.metajs's js_jsmget and in the interpreters' getMember.
		if recv := u(a[0]); promIs(recv) {
			if name := rt.toString(u(a[1])); name == "then" || name == "catch" || name == "finally" {
				po := recv.(*jsObject)
				return w(jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
					v, _ := rt.promMethod(po, name, args)
					return v
				}))
			}
		}
		// `f.name` and `f.length` - see js_jsvget above. docs/todo.md 1.4.
		if v := jsvFnIntrinsic(u(a[0]), rt.toString(u(a[1]))); v != nil {
			return w(v)
		}
		// An own property written onto a FUNCTION lives in the side table, at the end
		// so a program that never writes one pays one map-length test.
		// docs/todo.md 2.10.
		if p := jsvFnPropsOf(u(a[0])); p != nil {
			if v, found := p.props[rt.toString(u(a[1]))]; found {
				return w(v)
			}
		}
		// ...and a function's Object.prototype surface. This is the arm the CALL site
		// reaches: the emitter reads the member with js_jsmget and calls the result
		// when it is a function, so one body serves `f.hasOwnProperty("p")` and
		// `typeof f.toString`. docs/todo.md 2.10.
		if m := rt.jsvFnMethod(u(a[0]), rt.toString(u(a[1]))); m != nil {
			return w(m)
		}
		return baseGet(a)
	}

	// The static table of one class, built at its definition (docs/todo.md 2.13).
	// The layer-2 twin also counts the classes, so that a program with none pays a
	// single integer compare per member read instead of a hash lookup; the Go twin
	// has no such loop to protect and only has to answer the same object.
	// js_jssupget is `super.x` READ AS A VALUE for js/typescript, and it exists only
	// because of the shim in its callable arm. The shared js_supget (abnf/jsrt.go)
	// hands back a host function that IGNORES the receiver of the eventual call and
	// always re-enters with the receiver the READ was made with, so
	// `super.m.call(x)` ran the parent's method on `this` instead of on x. Fixing
	// that in js_supget would change it for the eight other languages that emit it,
	// and the extern map is ONE map for every language (manual chapter 4:
	// overriding a shared name in a language's own twin file leaks across
	// languages), so js/ts get a name of their own - the same pattern as js_jvsub /
	// js_csmcall / js_ktsmcall. Everything else here is js_supget's own behaviour,
	// deliberately: the static side first when the receiver is a class descriptor,
	// then the __super walk, and a miss is undefined rather than a read of the
	// receiver. The layer-2 twin is js_jssupget in languages/lib/js-rt.metajs.
	// docs/todo.md 2.10.
	m["js_jssupget"] = func(a []uint64) uint64 { // (super class, this, key) -> value
		name := rt.toString(u(a[2]))
		self := u(a[1])
		if isClassDesc(self) {
			if v, _, found := jsStaticsFind(u(a[0]), name); found {
				if acc, isAcc := v.(*jsAccessor); isAcc {
					if acc.get == nil {
						return jsHUndefined
					}
					return w(rt.call(acc.get, self, nil))
				}
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
					return w(rt.call(acc.get, self, nil))
				}
				if isCallable(v) {
					bound := v
					return w(jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
						// An EXPLICIT receiver wins; an absent one keeps the
						// read-time receiver, which is what a Reference does and
						// what makes `(super.m)()` behave as it did.
						recv := self
						if !isUndefOrNull(u(this)) {
							recv = u(this)
						}
						return rt.call(bound, jsUndef, append([]interface{}{recv}, args...))
					}))
				}
				return w(v)
			}
			cls = clsObj.props["__super"]
		}
		return jsHUndefined
	}

	// js_jskeys is js_keys plus the class-descriptor arm. The floor's js_keys is
	// shared by sixteen languages and answers an object's own key map, which for a
	// class descriptor is its INSTANCE methods - so `{...C}` copied the methods and
	// `for (k in C)` enumerated them, where node gives the static FIELDS and this
	// engine's own Object.keys now does too. A name of its own rather than an
	// override, for manual chapter 4's reason. The layer-2 twin is js_jskeys in
	// languages/lib/js-rt.metajs. docs/todo.md 2.10.
	// It also has to be TOTAL, which the floor's js_keys is not. `for (i in [7,8])`
	// and `{...[7,8]}` both reach here with an ARRAY, and js_keys aborts the whole
	// program with "js_keys: not an object (got object)" - a plain abort on ordinary
	// code, where node answers the INDICES as strings ("0","1"). A string enumerates
	// the same way, and null/undefined/a number/a boolean enumerate NOTHING rather
	// than raising. docs/todo.md 2.4.
	baseKeys := m["js_keys"]
	m["js_jskeys"] = func(a []uint64) uint64 {
		switch v := u(a[0]).(type) {
		case *jsObject:
			if isClassDesc(v) {
				out := &jsArray{}
				for _, k := range jsvStaticEnumKeys(v) {
					out.elems = append(out.elems, k)
				}
				return w(out)
			}
			if _, marked := v.props["__errh"]; marked {
				out := &jsArray{}
				for _, k := range jsvOwnKeys(v) {
					out.elems = append(out.elems, k)
				}
				return w(out)
			}
		case *jsArray:
			return w(jsvIndexKeys(len(v.elems)))
		case string:
			// strLen, not len(v): a string is indexed in UTF-16 CODE UNITS here.
			return w(jsvIndexKeys(rt.strLen(v)))
		default:
			// A FUNCTION enumerates the own properties written onto it.
			// docs/todo.md 2.10.
			if p := jsvFnPropsOf(v); p != nil {
				out := &jsArray{}
				for _, k := range jsvOwnKeys(p) {
					out.elems = append(out.elems, k)
				}
				return w(out)
			}
			// Not an object at all: node's ToObject of a primitive has no own
			// enumerable keys, and for-in over null/undefined is a no-op.
			return w(&jsArray{})
		}
		return baseKeys(a)
	}

	m["js_jsstatics"] = func(a []uint64) uint64 {
		o, ok := u(a[0]).(*jsObject)
		if !ok {
			rt.fail("js_jsstatics needs a class descriptor")
		}
		return w(jsStaticsOf(o, true))
	}

	// js_jssfield(statics, key, value) files one static FIELD, which is the only
	// kind of static member that is ENUMERABLE. A static method and a static
	// accessor go into the same table with a plain js_set / js_defprop and are
	// therefore invisible to Object.keys, which is what node says. The emitter calls
	// this instead of js_set in emitClass's static-field loop; the layer-2 twin is
	// js_jssfield in languages/lib/js-rt.metajs. docs/todo.md 2.10.
	m["js_jssfield"] = func(a []uint64) uint64 {
		t, ok := u(a[0]).(*jsObject)
		if !ok {
			rt.fail("js_jssfield needs a static table")
		}
		name := rt.toString(u(a[1]))
		t.set(name, u(a[2]))
		jsvSfTrack(t, name)
		return 0
	}

	// js_jsmset used to be emitted only under -exe, where layer 2 has to unwrap the
	// accessor box the C floor knows nothing about, and so it needed no Go twin. It
	// is emitted in BOTH halves since docs/todo.md 2.13: a member WRITE on a class
	// descriptor is a write to the STATIC namespace (`C.zz = 1` then `C.zz`), and
	// putting it in the descriptor's own property map would put it where the instance
	// methods live. Everything that is not a class descriptor - which is every write
	// in a program with no classes in it - goes straight to js_set.
	baseSet := m["js_set"]
	m["js_jsmset"] = func(a []uint64) uint64 {
		if cls, isStatic := jsvIsStaticSide(u(a[0]), rt.toString(u(a[1]))); isStatic {
			name := rt.toString(u(a[1]))
			// A setter anywhere on the static chain wins over a fresh own slot,
			// the same rule js_set follows for an ordinary object.
			if v, _, found := jsStaticsFind(cls, name); found {
				if acc, isAcc := v.(*jsAccessor); isAcc {
					if acc.set != nil {
						rt.call(acc.set, cls, []interface{}{u(a[2])})
					}
					return 0
				}
			}
			t := jsStaticsOf(cls, true)
			t.set(name, u(a[2]))
			// A runtime `C.zz = 1` creates an ENUMERABLE own property of the class
			// object, exactly as it does on any other object. docs/todo.md 2.10.
			jsvSfTrack(t, name)
			return 0
		}
		// `name` and `length` are NON-WRITABLE: node drops `f.name = "q"` silently in
		// sloppy code (and raises a TypeError in strict code, which no engine here
		// does). Dropping it keeps one answer for the read, for hasOwnProperty and for
		// getOwnPropertyNames. docs/todo.md 1.4.
		if jsvIsFnLike(u(a[0])) {
			if n := rt.toString(u(a[1])); n == "name" || n == "length" {
				return 0
			}
		}
		// A write onto a FUNCTION goes to the side table: neither *jsClosure nor
		// *hostFunc has a property map, and rt.setMember's reflect fallback aborted
		// with "cannot set field 'p' on *abnf.jsClosure". docs/todo.md 2.10.
		if p := jsvFnPropsMake(u(a[0])); p != nil {
			p.set(rt.toString(u(a[1])), u(a[2]))
			return 0
		}
		if o, isObj := u(a[0]).(*jsObject); isObj {
			if rt.toString(u(a[1])) == "name" {
				jsvErrNameWrite(o)
			}
		}
		return baseSet(a)
	}

	// (closure, name, length) -> closure. The emitter calls it at the creation site of
	// every function that HAS a name or a parameter, which is python's js_pyfnname
	// pattern exactly. A repeat for the same value overwrites.
	m["js_jsfnmeta"] = func(a []uint64) uint64 {
		if v := u(a[0]); jsvIsFnLike(v) {
			jsvFnMetaSet(v, rt.toString(u(a[1])), int(rt.toNumber(u(a[2]))))
		}
		return a[0]
	}

	// Wrap a compiled closure into an ASYNC function: calling the result runs the
	// body as a generator whose yields are its awaits, drives it from the microtask
	// queue and answers a promise for its completion. See asyncStep.
	m["js_jsasyncfn"] = func(a []uint64) uint64 {
		// The argument is the GENERATOR FUNCTION the async body compiled to (js_genfn
		// of its closure), not the closure itself: layer 2 cannot reach the floor's
		// coroutines from MetaJS, so the emitter builds the generator function and
		// both engines only ever CALL it. See languages/lib/js-rt.metajs.
		genFn := u(a[0])
		return w(jsHostFunc("async", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			g, ok := rt.call(genFn, jsUndef, args).(*jsGenerator)
			if !ok {
				rt.fail("js_jsasyncfn needs a generator function")
			}
			p := promNew()
			rt.asyncStep(g, p, jsUndef, false)
			return p
		}))
	}
	// The value an `await` answers, given what the driver sent back in. A rejection
	// arrives as a marker record and is re-raised HERE, at the await site, so a
	// try/catch around the await catches it with no generator support at all.
	m["js_jsawaitv"] = func(a []uint64) uint64 {
		if o, ok := u(a[0]).(*jsObject); ok {
			if t, has := o.props["__athrow"]; has && t == true {
				panic(&jsThrown{value: o.props["v"]})
			}
		}
		return a[0]
	}
	// The `Promise` global: a function value, so `new Promise(exec)` constructs and
	// `typeof Promise` is "function"; Promise.resolve / reject / all / race are
	// answered by js_jsmcall (memberCall's hostFunc arm).
	m["js_jspromfn"] = func(a []uint64) uint64 { return w(rt.promiseGlobal()) }
	// Run the microtask queue to exhaustion. The emitter calls this once, after the
	// whole script has run, which is this runtime's single event-loop turn.
	m["js_jsdrain"] = func(a []uint64) uint64 {
		rt.drainMicrotasks()
		return w(jsUndef)
	}

	// The method-call wrapper. js_jsxmcall is the regular-expression one; it in turn
	// delegates to the shared js_mcall, so a receiver none of the three claims
	// behaves exactly as it did before.
	// The delegate is looked up LAZILY: rxExtraExterns runs its registrars in file
	// order, so js_jsxmcall (abnf/jsrtregexjs.go) does not exist yet at this point.
	// Capturing it here bound nil and lost the whole RegExp method surface -
	// `re.test(s)` failed with "unknown method 'test'".
	baseMcall := m["js_mcall"]
	m["js_jsmcall"] = func(a []uint64) uint64 {
		arr, ok := u(a[2]).(*jsArray)
		if !ok {
			rt.fail("js_jsmcall args must be an array")
		}
		if v, handled := rt.jsvMethod(u(a[0]), rt.toString(u(a[1])), arr.elems); handled {
			return w(v)
		}
		// A CLASS DESCRIPTOR stops here (docs/todo.md 2.13). js_jsmget already
		// answered every static, so reaching this line means the name is not one -
		// and the shared js_mcall below would find the descriptor's own property of
		// that name, which is its INSTANCE method. Answer the miss sentinel so the
		// call site raises node's "C.m is not a function" instead.
		if _, isStatic := jsvIsStaticSide(u(a[0]), rt.toString(u(a[1]))); isStatic {
			if v, ok := rt.mcallMiss(); ok {
				return w(v)
			}
		}
		if next := m["js_jsxmcall"]; next != nil {
			return next(a)
		}
		return baseMcall(a)
	}

	// ----- the Error hierarchy (docs/todo.md 1.1) --------------------------
	//
	// The classes are ordinary {__isclass, __name, __errname, __super, __ctor}
	// descriptors - the same shape emitClass builds - so js_jsinstanceof's chain
	// walk and makeNewFrom's class arm answer them with no arm of their own. The
	// emitter owns the two closures and the js_scope_decl bindings (a floor
	// primitive layer 2 cannot call); everything else is these externs:
	//
	//   js_jserrcls   builds (and memoizes) one class from its name and the two
	//                 emitted closures, so that the errors the RUNTIME raises
	//                 (bigRaise) carry the VERY SAME class object the program's
	//                 own `new TypeError(...)` constructs from - two class
	//                 objects of one name would make instanceof false. Building
	//                 them here rather than with six memSets each in the emitter
	//                 is what keeps the fixed cost of the hierarchy at +3,075
	//                 bytes of IR per module instead of +10,201;
	//   js_jserrinit  the shared constructor body: it walks the instance's own
	//                 __class chain for __errname, which is what makes a user
	//                 subclass answer its nearest builtin ancestor's name;
	//   js_jserrstr   Error.prototype.toString.
	//
	// The twins are in languages/lib/js-rt.metajs and the interpreter halves'
	// jsErr* helpers. name / message / stack are set with set() rather than
	// through the key-order machinery, because all three are non-enumerable in a
	// real engine: Object.keys(new Error("x")) is [].
	m["js_jserrcls"] = func(a []uint64) uint64 { // (name, ctor, toString) -> the class
		if rt.jsErrClasses == nil {
			rt.jsErrClasses = map[string]*jsObject{}
		}
		name := rt.toString(u(a[0]))
		if o, has := rt.jsErrClasses[name]; has {
			return w(o)
		}
		cls := newJSObject()
		cls.set("__isclass", true)
		cls.set("__name", name)
		cls.set("__errname", name)
		cls.set("__ctor", u(a[1]))
		// Every class descriptor gets a static table, the builtin Error hierarchy
		// included - the layer-2 twin needs it to keep `TypeError.name` independent
		// of whether the program declares a class of its own. docs/todo.md 2.13.
		jsStaticsOf(cls, true)
		if name == jsErrNames[0] {
			// toString lives on the ROOT only, exactly as Error.prototype does:
			// the __class/__super walk finds it from any of the seven and from
			// any user subclass.
			cls.set("toString", u(a[2]))
		} else if root, has := rt.jsErrClasses[jsErrNames[0]]; has {
			cls.set("__super", root)
		}
		rt.jsErrClasses[name] = cls
		return w(cls)
	}
	m["js_jserrinit"] = func(a []uint64) uint64 {
		self, ok := u(a[0]).(*jsObject)
		if !ok {
			return jsHUndefined
		}
		msg := ""
		// `new Error()` and `new Error(undefined)` both leave message "": the
		// spec only assigns it when the argument is not undefined. null is NOT
		// undefined, so `new Error(null).message` is "null".
		if v := u(a[1]); v != interface{}(jsUndef) {
			msg = rt.jsvString(v)
		}
		rt.jsErrInit(self, msg)
		return jsHUndefined
	}
	m["js_jserrstr"] = func(a []uint64) uint64 {
		return w(rt.jsErrToString(u(a[0])))
	}
	// "TypeError: Cannot mix BigInt..." -> a real TypeError carrying just the
	// message, or the text UNCHANGED when its prefix names no registered Error
	// class. This is what routes the emitters' own thrown strings (the
	// iterator-throw TypeError) through the hierarchy.
	m["js_jserrmk"] = func(a []uint64) uint64 { return w(rt.jsErrFromText(rt.toString(u(a[0])))) }

	// BigInt last: its wrappers sit in FRONT of the js_js* names registered above
	// (js_jsadd, js_jseq, js_jslt, ...), so they have to exist first.
	rt.addJSBigIntExterns(m)
}

// jsErrNames is the Error hierarchy, root first. jsErrFromText scans it in this
// order, and "Error: " cannot shadow "TypeError: " either way round because a
// prefix test is anchored at the start.
var jsErrNames = []string{"Error", "TypeError", "RangeError", "ReferenceError",
	"SyntaxError"}

// jsErrNameOf is the nearest BUILTIN error name up the instance's class chain,
// which is what makes `class E extends TypeError {}` answer "TypeError" and
// `class E extends Error {}` answer "Error" - name lives on the prototype in a
// real engine and E.prototype does not shadow it. The guard is the same 64 the
// rest of this project puts on a descriptor walk.
func jsErrNameOf(self *jsObject) string {
	guard := 0
	for cls := self.props["__class"]; cls != nil && guard < 64; guard++ {
		clsObj, ok := cls.(*jsObject)
		if !ok {
			break
		}
		if n, has := clsObj.props["__errname"]; has {
			if s, isStr := n.(string); isStr {
				return s
			}
		}
		cls = clsObj.props["__super"]
	}
	return "Error"
}

// jsErrText is Error.prototype.toString's rule: "TypeError: x", the name alone
// when there is no message, the message alone when the name has been emptied.
func jsErrText(name, msg string) string {
	if msg == "" {
		return name
	}
	if name == "" {
		return msg
	}
	return name + ": " + msg
}

// jsErrInit is the shared Error constructor. STACK cannot be honest here - there
// is no frame walker in any of the four engines - so it holds exactly the first
// line node's stack starts with and no frames, which is what an
// `e.stack.split("\n")[0]` idiom reads.
func (rt *jsrt) jsErrInit(self *jsObject, msg string) {
	// name / message / stack are NOT ENUMERABLE in node: Object.keys(new
	// Error("x")) is []. The __errh mark makes jsvErrHide drop them from the key
	// lists, the same shape as layer 2's, where a floor object cannot hold a hidden
	// key at all. js_jsmset clears the mark's name half when the program writes
	// `name` itself, which node makes an ordinary enumerable own property because
	// the original lives on the prototype. Twins: js_jserrinit in
	// languages/lib/js-rt.metajs and jsErrInit in both *-interpreter.abnf.
	name := jsErrNameOf(self)
	self.set("name", name)
	self.set("message", msg)
	self.set("stack", jsErrText(name, msg))
	self.props["__errh"] = true
}

// jsvErrHide is an Error instance's own key list without the non-enumerable three
// (see jsErrInit); any other object's list comes back as it is.
func jsvErrHide(o *jsObject, ks []string) []string {
	h, marked := o.props["__errh"]
	if !marked {
		return ks
	}
	out := make([]string, 0, len(ks))
	for _, k := range ks {
		if k == "message" || k == "stack" || (k == "name" && h == true) {
			continue
		}
		out = append(out, k)
	}
	return out
}

// jsvErrNameWrite: the program writes `name` on an Error instance, so from now on it
// is an ordinary own property, created at THIS point of the key order.
func jsvErrNameWrite(o *jsObject) {
	if o.props["__errh"] != true {
		return
	}
	o.props["__errh"] = false
	for i, k := range o.keys {
		if k == "name" {
			o.keys = append(o.keys[:i:i], o.keys[i+1:]...)
			break
		}
	}
	delete(o.props, "name")
}

func (rt *jsrt) jsErrToString(v interface{}) string {
	o, ok := v.(*jsObject)
	if !ok {
		return "Error"
	}
	name := "Error"
	if n, has := o.props["name"]; has && n != interface{}(jsUndef) {
		name = rt.jsvString(n)
	}
	msg := ""
	if mv, has := o.props["message"]; has && mv != interface{}(jsUndef) {
		msg = rt.jsvString(mv)
	}
	return jsErrText(name, msg)
}

// jsMakeErr builds an instance of a REGISTERED class - the same object the
// program's own `new TypeError(...)` constructs from, so instanceof holds.
// Answers nil when the emitter never registered that name (a grammar with no
// Error block, or a language that is not js/ts).
func (rt *jsrt) jsMakeErr(name, msg string) *jsObject {
	cls, ok := rt.jsErrClasses[name]
	if !ok {
		return nil
	}
	o := newJSObject()
	o.set("__class", cls)
	rt.jsErrInit(o, msg)
	return o
}

// jsErrFromText turns "TypeError: ..." into a real TypeError carrying just the
// message. Every error the js runtime raises already spells its own class in its
// message text, so the whole routing is this one split rather than a rewrite of
// thirty call sites - and a text naming no class of ours is returned exactly as
// it was, a thrown string.
func (rt *jsrt) jsErrFromText(t string) interface{} {
	for _, n := range jsErrNames {
		pre := n + ": "
		if len(t) > len(pre) && t[:len(pre)] == pre {
			if e := rt.jsMakeErr(n, t[len(pre):]); e != nil {
				return e
			}
			return t
		}
	}
	return t
}

// ============================================================================
// Async generators (docs/todo.md 1.7)
//
// An async generator body yields TWO kinds of thing on ONE suspension channel:
// its awaits and its yields. They are told apart by a MARKER RECORD on the
// yielded value - {__awt, v}, built by js_jsawaitmark, which the emitter wraps
// every awaited operand of an async generator body in - and that marker is the
// whole mechanism. The generator model needed nothing added to it: the body is
// an ordinary generator body, exactly as an async function's is, and only the
// driver is new.
//
// The driver is asyncStep with a REQUEST QUEUE in front of it. next(v),
// return(v) and throw(e) each answer a promise and append a request; one
// request is served at a time, an await resumes the body WITHOUT answering the
// pending next(), and a yield answers it and parks.
//
// The twins are jpAg* in languages/lib/js-rt.metajs and in the two
// *-interpreter.abnf start scripts. All three run the same algorithm over the
// same value model, which is why languages/lib/runtime.c is not touched.
//
// ag.throw(e) RAISES AT THE SUSPENDED YIELD, and it needs no throw-in channel in
// the generator engine: the resume value is the channel. The driver resumes the
// body with a {__athrow, v} record and the emitter unpacks every yield's result
// through js_jsawaitv, which raises when it sees one - the same record and the
// same unpacker an await's rejected operand already used. Nothing is kept across
// a suspension: the record travels ON the resume, so it is per-coroutine by
// construction (the constraint docs/todo.md 2.1 was reverted for).
// A throw before the body has ever run does NOT start it (st.started), which is
// node's suspendedStart rule, and one that reaches a `yield*` raises at the
// yield* rather than being forwarded to the delegate's throw().

type agRequest struct {
	kind int // 0 next, 1 return, 2 throw
	arg  interface{}
	p    *jsObject
}

type jsAsyncGen struct {
	g    *jsGenerator
	q    []*agRequest
	run  bool
	done bool
	// The body has been entered at least once. A throw() before that does not
	// start it - node's suspendedStart rule.
	started bool
}

func agIsAwaitMark(v interface{}) bool {
	o, ok := v.(*jsObject)
	if !ok {
		return false
	}
	t, has := o.props["__awt"]
	return has && t == true
}

func agUnmark(v interface{}) interface{} {
	if agIsAwaitMark(v) {
		return v.(*jsObject).props["v"]
	}
	return v
}

// agObject is the value an async generator function answers: a plain object
// with hidden marker '__agen' and three bound methods, and NO own keys (the
// keys slice is cleared, exactly as a promise carries none).
func (rt *jsrt) agObject(g *jsGenerator) *jsObject {
	st := &jsAsyncGen{g: g}
	ag := newJSObject()
	ag.set("__agen", true)
	mk := func(name string, kind int) {
		ag.set(name, jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return rt.agReq(st, kind, argAt(args, 0))
		}))
	}
	mk("next", 0)
	mk("return", 1)
	mk("throw", 2)
	ag.keys = nil
	return ag
}

func (rt *jsrt) agReq(st *jsAsyncGen, kind int, arg interface{}) *jsObject {
	p := promNew()
	st.q = append(st.q, &agRequest{kind: kind, arg: arg, p: p})
	rt.agPump(st)
	return p
}

func (rt *jsrt) agPump(st *jsAsyncGen) {
	if st.run || len(st.q) == 0 {
		return
	}
	st.run = true
	req := st.q[0]
	if st.done {
		if req.kind == 2 {
			rt.agFail(st, req.arg)
			return
		}
		arg := interface{}(jsUndef)
		if req.kind == 1 {
			arg = req.arg
		}
		rt.agComplete(st, arg, true)
		return
	}
	// throw(e) on a body that is parked at a yield RESUMES it with a throw
	// completion, so a try around that yield catches.
	if req.kind == 2 && st.started {
		rt.agStep(st, req.arg, true)
		return
	}
	if req.kind == 1 || req.kind == 2 {
		st.done = true
		st.g.closeBody(rt)
		if req.kind == 2 {
			rt.agFail(st, req.arg)
			return
		}
		rt.agComplete(st, req.arg, true)
		return
	}
	rt.agStep(st, req.arg, false)
}

func (rt *jsrt) agStep(st *jsAsyncGen, sent interface{}, isThrow bool) {
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
	st.started = true
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
		res = st.g.step(rt, send)
	}()
	if threw {
		st.done = true
		rt.agFail(st, thrown)
		return
	}
	rec, _ := res.(*jsObject)
	if d, _ := rec.props["done"].(bool); d {
		st.done = true
		rt.agComplete(st, rec.props["value"], true)
		return
	}
	y := rec.props["value"]
	if agIsAwaitMark(y) {
		// An AWAIT: resume the body once the operand settles, and leave the
		// pending next() unanswered.
		rt.promThen(rt.promResolveValue(agUnmark(y)),
			jsHostFunc("await", func(rt *jsrt, this uint64, args []interface{}) interface{} {
				rt.agStep(st, argAt(args, 0), false)
				return jsUndef
			}),
			jsHostFunc("await", func(rt *jsrt, this uint64, args []interface{}) interface{} {
				rt.agStep(st, argAt(args, 0), true)
				return jsUndef
			}), nil)
		return
	}
	// A YIELD. The specification awaits the operand before handing it over.
	rt.promThen(rt.promResolveValue(y),
		jsHostFunc("yield", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			rt.agComplete(st, argAt(args, 0), false)
			return jsUndef
		}),
		jsHostFunc("yield", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			st.done = true
			rt.agFail(st, argAt(args, 0))
			return jsUndef
		}), nil)
}

func (rt *jsrt) agShift(st *jsAsyncGen) *agRequest {
	req := st.q[0]
	st.q = st.q[1:]
	st.run = false
	return req
}

func (rt *jsrt) agComplete(st *jsAsyncGen, v interface{}, done bool) {
	req := rt.agShift(st)
	r := newJSObject()
	r.set("value", v)
	r.set("done", done)
	rt.promSettle(req.p, 1, r)
	rt.agPump(st)
}

func (rt *jsrt) agFail(st *jsAsyncGen, e interface{}) {
	req := rt.agShift(st)
	rt.promSettle(req.p, 2, e)
	rt.agPump(st)
}
