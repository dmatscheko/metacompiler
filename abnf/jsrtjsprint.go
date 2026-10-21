package abnf

// jsrtjsprint.go -- the JavaScript/TypeScript half of the shared runtime.
//
// The compiler grammars (languages/js-to-llvm-ir.abnf, languages/typescript-to-llvm-ir.abnf)
// reach JavaScript's own answers here instead of the LANGUAGE-NEUTRAL externs of
// abnf/jsrt.go for printing (null, arrays and objects spelled the way String()
// spells them), for the equality and relational operators (ToPrimitive on an object
// operand), for `instanceof` on a plain constructor FUNCTION, for `delete` on an
// array and for the String/Array/Object method surface. Every neutral extern is a
// compromise between a dozen languages, and JavaScript is the one language whose
// answer they are NOT.
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
	"unicode/utf16"
	"unicode/utf8"
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
		// A function on a prototype view reads `this`, as an object literal's does.
		if v, hasView := clsObj.props["__pview"].(*jsObject); hasView {
			if m, ok := v.props[name]; ok && isCallable(m) {
				return rt.call(m, o, nil), true
			}
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
	return jsvFindMethodFrom(o.props["__class"], name)
}

// jsvFindMethodFrom walks the __super chain from the descriptor cls.
func jsvFindMethodFrom(cls interface{}, name string) (interface{}, bool) {
	guard := 0
	for ; cls != nil && guard < 64; guard++ {
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

// jsvChainRead is a read that missed an object's own slots, along the prototype
// chain from descriptor cls up: at each level the class body's method, as its value,
// or its accessor, else an own property of that level's prototype view - which is how
// `B.prototype.x = 1` and Object.create's prototype object reach the objects below
// them. An accessor is answered as itself, for jsvChainGet. Twins: protoChainRead in both *-interpreter.abnf, jvChainRead in
// languages/lib/js-rt.metajs.
func jsvChainRead(cls interface{}, name string) (interface{}, bool) {
	for guard := 0; cls != nil && guard < 64; guard++ {
		d, ok := cls.(*jsObject)
		if !ok {
			break
		}
		if m, has := d.props[name]; has && isCallable(m) {
			return jsvMethodValue(name, m), true
		} else if _, isAcc := m.(*jsAccessor); isAcc {
			return m, true
		}
		if v, hasView := d.props["__pview"].(*jsObject); hasView {
			if pv, has := v.props[name]; has {
				return pv, true
			}
		}
		cls = d.props["__super"]
	}
	return nil, false
}

// jsvChainGet reads a value jsvChainRead found for receiver recv: an accessor of a
// prototype object runs its getter with recv as `this`. Twins: findAccessor in both
// *-interpreter.abnf, jvFindClassAcc in languages/lib/js-rt.metajs.
func (rt *jsrt) jsvChainGet(v, recv interface{}) interface{} {
	if acc, isAcc := v.(*jsAccessor); isAcc {
		if acc.get == nil {
			return jsUndef
		}
		return rt.call(acc.get, recv, nil)
	}
	return v
}

// jsvViewFn is the function a method CALL finds on a prototype view before any class
// method of that name - a function on Object.create's prototype object - which is
// called with `this` rather than with the receiver prepended. Twins: mcall in both
// *-interpreter.abnf, jvFallbackCall in languages/lib/js-rt.metajs.
func jsvViewFn(o *jsObject, name string) (interface{}, bool) {
	for cls, guard := o.props["__class"], 0; cls != nil && guard < 64; guard++ {
		d, ok := cls.(*jsObject)
		if !ok {
			break
		}
		if _, has := d.props[name]; has {
			return nil, false
		}
		if v, hasView := d.props["__pview"].(*jsObject); hasView {
			if pv, has := v.props[name]; has {
				return pv, isCallable(pv)
			}
		}
		cls = d.props["__super"]
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
// files '__#s' in the static table, so skipping it would make `C.#s` answer undefined.
// jsvPrivateKey is the exception.
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
// with `this` bound to the SUBCLASS), and `name` and `length` last - every class
// object has both, the descriptor's __name and its constructor's arity, unless a
// static member shadows them.
// Object.prototype's methods are inherited by a class object too, so they read
// as the same shim js_jsmcall's class arm applies. Anything else is undefined, as
// node says. Twins: jvStaticSideRead in languages/lib/js-rt.metajs, getMember's
// static arm in both *-interpreter.abnf.
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
	if name == "__#@species" && jsvSpeciesRoot(o) {
		return rt.wrap(o)
	}
	if (name == "name" || name == "length") && o.props["__del"+name] == true {
		if sup, isObj := o.props["__super"].(*jsObject); isObj && isClassDesc(sup) {
			return rt.jsvStaticSideRead(sup, name)
		}
		if name == "name" {
			return rt.wrap("")
		}
		return rt.wrap(float64(0))
	}
	if name == "name" {
		if n, has := o.props["__name"]; has {
			return rt.wrap(n)
		}
	}
	if name == "length" {
		return rt.wrap(float64(jsvFnMeta(o.props["__ctor"]).length))
	}
	// A class is a function, so its inherited `constructor` is Function.
	if name == "constructor" {
		return rt.wrap(rt.jsGlobals["Function"])
	}
	if jsvIsProtoName(name) {
		return rt.wrap(jsvProtoShim(name))
	}
	return jsHUndefined
}

// jsvHasSpecies: is g a builtin whose constructor has a [Symbol.species] getter
// answering `this` (ECMA-262 23.1.2.5, 27.2.4.8, 22.2.5.2)? Map's and Set's are
// lib/js-builtins.js's. jsvSpeciesRoot: does class cls's chain end at one, so that
// it inherits the getter and answers itself? Twins: jsHasSpecies / jsSpeciesRoot in
// both *-interpreter.abnf, jvHasSpecies / jvSpeciesRoot in languages/lib/js-rt.metajs.
func jsvHasSpecies(g interface{}) bool {
	if hf, ok := g.(*hostFunc); ok && (hf.name == "Promise" || hf.name == "RegExp") {
		return true
	}
	return jsvIsArrayGlobal(g)
}

func jsvSpeciesRoot(cls *jsObject) bool {
	for guard := 0; guard < 64; guard++ {
		sup, ok := cls.props["__super"].(*jsObject)
		if !ok || !isClassDesc(sup) {
			break
		}
		cls = sup
	}
	return jsvHasSpecies(cls.props["__ctorfn"])
}

// jsvProtoTag is Object.prototype.toString's answer for a value.
func (rt *jsrt) jsvProtoTag(v interface{}) string {
	switch v.(type) {
	case nil, jsNullT:
		return "[object Null]"
	case jsUndefT:
		return "[object Undefined]"
	case *jsArray:
		if jsvIsArgs(v) {
			return "[object Arguments]"
		}
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
	// The objects node gives a @@toStringTag: an Error instance (its __errh mark), a
	// promise, and Math / JSON, which carry their tag in a hidden __strtag slot.
	if o, ok := v.(*jsObject); ok {
		// A string [Symbol.toStringTag], own or inherited, names the object.
		if t, isStr := rt.jsvSymMember(o, "__#@toStringTag").(string); isStr {
			return "[object " + t + "]"
		}
		if _, has := o.props["__errh"]; has {
			return "[object Error]"
		}
		if t, has := o.props["__strtag"].(string); has {
			return "[object " + t + "]"
		}
		if p, isWrap := jsvWrapped(o); isWrap {
			return rt.jsvProtoTag(p)
		}
		if promIs(o) {
			return "[object Promise]"
		}
	}
	return "[object Object]"
}

// jsvIterList is the values an iterable yields, for the statics that take one
// (Object.fromEntries, Object.groupBy): an array as it is, a string's code points, a
// generator drained, the iterator a [Symbol.iterator] method answers driven to the
// end. Anything else is node's TypeError. Twins: jsIterList in both
// *-interpreter.abnf, jvIterList in languages/lib/js-rt.metajs.
func (rt *jsrt) jsvIterList(v interface{}) []interface{} {
	it := v
	iterated := false
	if o, isObj := v.(*jsObject); isObj {
		if r, called := rt.jsvCallMethod(o, "__#@iterator"); called {
			it, iterated = r, true
		}
	}
	switch x := it.(type) {
	case *jsArray:
		return x.elems
	case string:
		return jsvCodePoints(x)
	case *jsGenerator:
		return x.drain(rt).elems
	case *jsObject:
		var out []interface{}
		for iterated {
			step, called := rt.jsvCallMethod(x, "next")
			if !called {
				break
			}
			if rt.truthy(rt.jsvGetKey(step, "done")) {
				return out
			}
			out = append(out, rt.jsvGetKey(step, "value"))
		}
	}
	if isUndefOrNull(v) {
		rt.bigRaise("TypeError: undefined is not iterable")
	}
	rt.jsvNotIterable(v)
	return nil
}

// jsvNotIterable raises GetIterator's TypeError for a value with no callable
// [Symbol.iterator], in the wording node gives where it cannot name the source
// expression. Twins: jsNotIterable in both *-interpreter.abnf, jvNotIterable in
// languages/lib/js-rt.metajs.
func (rt *jsrt) jsvNotIterable(v interface{}) {
	what := "object"
	switch t := v.(type) {
	case float64:
		what = "number " + rt.jsvString(t)
	case bool:
		what = "boolean " + rt.jsvString(t)
	default:
		if _, isNull := v.(jsNullT); isNull {
			what = "object null"
		} else if _, isUndef := v.(jsUndefT); isUndef {
			what = "undefined"
		} else if _, ok := bigOf(v); ok {
			what = "bigint"
		} else if jsvIsSym(v) {
			what = "symbol"
		} else if isCallable(v) {
			what = "function"
		}
	}
	rt.bigRaise("TypeError: %s is not iterable (cannot read property Symbol(Symbol.iterator))", what)
}

// jsvPropKey is ToPropertyKey as a key of the value model: a symbol's __symkey, any
// other value's string, escaped.
func (rt *jsrt) jsvPropKey(k interface{}) string {
	if o, ok := k.(*jsObject); ok {
		if sk, isSym := o.props["__symkey"].(string); isSym {
			return sk
		}
	}
	return jsvKeyEsc(rt.jsvString(k))
}

// jsvNotFnText is V8's rendering of a non-callable callback in "<x> is not a
// function": a primitive's string, "#<Object>" for an object. Twins: jsNotFnText in
// both *-interpreter.abnf, jvNotFnText in languages/lib/js-rt.metajs.
func (rt *jsrt) jsvNotFnText(f interface{}) string {
	switch f.(type) {
	case *jsArray:
		return "[object Array]"
	case *jsObject:
		if !jsvIsSym(f) {
			if _, isBig := bigOf(f); !isBig {
				return "#<Object>"
			}
		}
	}
	return rt.jsvString(f)
}

// jsvStringRaw is String.raw(template, ...subs): template.raw's strings, the i-th
// substitution between the i-th and the next one, over ToLength(raw.length)
// strings. Twins: jsStringRaw in both *-interpreter.abnf, jvStringRaw in
// languages/lib/js-rt.metajs.
func (rt *jsrt) jsvStringRaw(args []interface{}) string {
	var raw interface{} = jsUndef
	if tp := argAt(args, 0); !isUndefOrNull(tp) {
		raw = rt.jsvGetKey(tp, "raw")
	}
	if isUndefOrNull(raw) {
		rt.bigRaise("TypeError: Cannot convert undefined or null to object")
	}
	n := rt.toNumber(rt.jsvGetKey(raw, "length"))
	if !(n > 0) {
		n = 0
	}
	var b strings.Builder
	for i := 0; float64(i) < math.Floor(n); i++ {
		b.WriteString(rt.jsvString(rt.jsvGetKey(raw, strconv.Itoa(i))))
		if float64(i+1) < math.Floor(n) && i+1 < len(args) {
			b.WriteString(rt.jsvString(args[i+1]))
		}
	}
	return b.String()
}

// jsvIsProtoName names Object.prototype's methods. No value here has the prototype
// chain they are inherited through, so a call or a value read that finds no member of
// one of these names is answered by jsvProtoApply. Twins: jsProtoApply in both
// *-interpreter.abnf, jvProtoApply in languages/lib/js-rt.metajs.
func jsvIsProtoName(name string) bool {
	_, ok := jsvProtoFns[name]
	return ok
}

var jsvProtoNames = []string{"toString", "valueOf", "hasOwnProperty", "propertyIsEnumerable", "isPrototypeOf", "toLocaleString"}

func (rt *jsrt) jsvProtoApply(recv interface{}, name string, args []interface{}) interface{} {
	switch name {
	case "hasOwnProperty":
		return rt.jsvHasOwn(recv, rt.jsvPropKey(argAt(args, 0)))
	case "propertyIsEnumerable":
		return jsvPropEnum(recv, jsvKeyEsc(rt.toString(argAt(args, 0))))
	case "toString":
		return rt.jsvProtoTag(recv)
	case "isPrototypeOf":
		return jsvIsProtoOf(recv, argAt(args, 0))
	case "toLocaleString":
		// Invoke(this, "toString") (ECMA-262 20.1.3.5), which ToString performs.
		return rt.jsvString(recv)
	}
	// valueOf is ToObject(this): a primitive receiver comes back wrapped.
	if isUndefOrNull(recv) {
		rt.bigRaise("TypeError: Cannot convert undefined or null to object")
	}
	return jsvToObject(recv)
}

// jsvWrap, jsvWrapped and jsvToObject are the primitive wrapper objects: an object
// whose hidden __bvof slot holds a number, a string, a boolean, a BigInt or a
// symbol. Twin of jsWrap /
// jsIsWrap / jsToObject in js-interpreter.abnf, which states the rule; jvWrap /
// jvToObject in languages/lib/js-rt.metajs.
func jsvWrap(p interface{}) *jsObject {
	o := newJSObject()
	o.props["__bvof"] = p
	return o
}

// jsvWrapped is the primitive a wrapper object holds.
func jsvWrapped(v interface{}) (interface{}, bool) {
	o, isObj := v.(*jsObject)
	if !isObj {
		return nil, false
	}
	p, has := o.props["__bvof"]
	return p, has
}

func jsvToObject(v interface{}) interface{} {
	switch v.(type) {
	case nil, jsUndefT, jsNullT:
		return newJSObject()
	case float64, string, bool, *jsBigInt:
		return jsvWrap(v)
	}
	if jsvIsSym(v) {
		return jsvWrap(v)
	}
	return v
}

// jsvWrapCtor is the name of the global whose prototype a wrapper of p inherits from.
func jsvWrapCtor(p interface{}) string {
	switch p.(type) {
	case float64:
		return "Number"
	case string:
		return "String"
	case *jsBigInt:
		return "BigInt"
	}
	if jsvIsSym(p) {
		return "Symbol"
	}
	return "Boolean"
}

// jsvProtoLocaleOwn: a number's, a BigInt's and an array's toLocaleString are their
// own prototypes' (jsToLocaleJS, jsvArrayLocale), so a call of one is not answered
// by jsvProtoApply. Twins: jsProtoCall in both *-interpreter.abnf,
// jvFallbackCall in languages/lib/js-rt.metajs.
func jsvProtoLocaleOwn(recv interface{}, name string) bool {
	if name != "toLocaleString" {
		return false
	}
	switch recv.(type) {
	case float64, *jsArray, *jsBigInt:
		return true
	}
	return false
}

// jsvIsProtoOf is Object.prototype.isPrototypeOf: is p on v's prototype chain?
// Object.prototype (marked __objproto) is on every object's; any other prototype is a
// view, or an Object.create prototype object, held as __pview by a descriptor on the
// chain. Twins: jsIsProtoOf in both *-interpreter.abnf, jvIsProtoOf in
// languages/lib/js-rt.metajs.
func jsvIsProtoOf(p, v interface{}) bool {
	if !jsvIsObjLike(v) && !isCallable(v) {
		return false
	}
	po, ok := p.(*jsObject)
	if !ok {
		return false
	}
	if _, isRoot := po.props["__objproto"]; isRoot {
		return true
	}
	vo, ok := v.(*jsObject)
	if !ok {
		return false
	}
	cls := vo.props["__class"]
	if d, isView := vo.props["__protoof"].(*jsObject); isView {
		cls = d.props["__super"]
	}
	for guard := 0; cls != nil && guard < 64; guard++ {
		d, ok := cls.(*jsObject)
		if !ok {
			break
		}
		if d.props["__pview"] == interface{}(po) {
			return true
		}
		cls = d.props["__super"]
	}
	return false
}

// jsvObjCreate is Object.create(p): an object whose prototype is p. A class's
// prototype view makes it an instance of that class whose constructor never ran. Any
// other object becomes the view of a descriptor of its own, whose parent is p's own
// class, so the read walk (jsvChainRead) reaches p's properties and then p's chain.
// null gives an object marked as inheriting nothing (jsvNoObjProto); a function and
// an array give a plain object, as no view reaches their own properties. Twins:
// jsObjCreate in both *-interpreter.abnf, jvObjCreate in languages/lib/js-rt.metajs.
func (rt *jsrt) jsvObjCreate(p interface{}) *jsObject {
	o := newJSObject()
	po, isObj := p.(*jsObject)
	switch {
	case p == interface{}(jsNull):
		o.props["__nullproto"] = true
		jsvNullProtoN++
		return o
	case p == nil || isCallable(p) || isClassDesc(p) || jsvIsObjectGlobal(p):
		return o
	case !isObj:
		if _, isArr := p.(*jsArray); isArr {
			return o
		}
		fallthrough
	case jsvIsSym(p):
		rt.bigRaise("TypeError: Object prototype may only be an Object or null: %s", rt.jsvString(p))
	}
	o.set("__class", jsvProtoDesc(po))
	return o
}

// jsvProtoDesc is the descriptor whose view is the object p, for an object whose
// prototype is p: a class's prototype view is its class's, any other object becomes
// the view of a descriptor of its own, whose parent is p's own class. Twins:
// protoDescFor in both *-interpreter.abnf, jvProtoDesc in languages/lib/js-rt.metajs.
func jsvProtoDesc(p *jsObject) *jsObject {
	if d, isView := p.props["__protoof"].(*jsObject); isView {
		return d
	}
	d := newJSObject()
	d.set("__pview", p)
	if c, has := p.props["__class"]; has {
		d.set("__super", c)
	}
	if p.props["__nullproto"] == true {
		d.props["__nullproto"] = true
	}
	return d
}

// jsvNullProtoN counts the null-prototype marks made (Object.create(null),
// setPrototypeOf(o, null)); while it is 0 jsvNoObjProto answers without a walk.
var jsvNullProtoN = 0

// jsvNoObjProto: does v inherit nothing from Object.prototype - no toString, no
// hasOwnProperty, `"toString" in v` false - because its chain ends at a null
// prototype? The mark is __nullproto, on the object or on a descriptor of its chain.
// Twins: jsNoObjProto in both *-interpreter.abnf, jvNoObjProto in
// languages/lib/js-rt.metajs.
func jsvNoObjProto(v interface{}) bool {
	o, ok := v.(*jsObject)
	if jsvNullProtoN == 0 || !ok {
		return false
	}
	if o.props["__nullproto"] == true {
		return true
	}
	cls := o.props["__class"]
	for guard := 0; cls != nil && guard < 64; guard++ {
		d, ok := cls.(*jsObject)
		if !ok {
			break
		}
		if d.props["__nullproto"] == true {
			return true
		}
		cls = d.props["__super"]
	}
	return false
}

// jsvSetProto is Object.setPrototypeOf(o, p) on an ordinary object: its descriptor
// becomes the one whose view is p (jsvProtoDesc), none for Object.prototype, and the
// null-prototype mark for null. A primitive comes back unchanged, and so do an array,
// a function and a prototype view, whose prototypes this value model keeps elsewhere.
// Twins: jsSetProto in both *-interpreter.abnf, jvSetProto in
// languages/lib/js-rt.metajs.
func (rt *jsrt) jsvSetProto(v, p interface{}) interface{} {
	if isUndefOrNull(v) {
		rt.bigRaise("TypeError: Object.setPrototypeOf called on null or undefined")
	}
	po, pObj := p.(*jsObject)
	if p != interface{}(jsNull) && !pObj && !isCallable(p) {
		if _, isArr := p.(*jsArray); !isArr {
			rt.bigRaise("TypeError: Object prototype may only be an Object or null: %s", rt.jsvString(p))
		}
	}
	o, ok := v.(*jsObject)
	if !ok || isClassDesc(o) {
		return v
	}
	if _, isView := o.props["__protoof"]; isView {
		return v
	}
	delete(o.props, "__class")
	delete(o.props, "__nullproto")
	switch {
	case p == interface{}(jsNull):
		o.props["__nullproto"] = true
		jsvNullProtoN++
	case pObj && po.props["__objproto"] == nil && !isClassDesc(po) && !jsvIsObjectGlobal(po) && !jsvIsSym(po):
		o.set("__class", jsvProtoDesc(po))
	}
	return v
}

// jsvDefineProp is Object.defineProperty(o, key, desc) for a key already in the
// object's own spelling: a data property from desc.value, or an accessor from
// desc.get / desc.set. An attribute desc does not name keeps its value on an existing
// property and is false on a new one; a false one is the key's entry in the object's
// __hidden, __ro or __nc set (jsvAttrHas). Twin of jsDefineKey in
// js-interpreter.abnf, which states the rule; jvDefineProp in
// languages/lib/js-rt.metajs.
func (rt *jsrt) jsvDefineProp(v interface{}, key string, desc interface{}) {
	if _, isObj := desc.(*jsObject); !isObj && !isCallable(desc) {
		if _, isArr := desc.(*jsArray); !isArr {
			rt.bigRaise("TypeError: Property description must be an object: %s", rt.jsvString(desc))
		}
	}
	g, s := rt.jsvGetKey(desc, "get"), rt.jsvGetKey(desc, "set")
	// An array's named data property keeps its attributes on the object that holds it
	// (jsvArrNamed), so the rules below apply there unchanged.
	if a, isArr := v.(*jsArray); isArr && key != "length" && rt.arrIndexOf(key) < 0 {
		p := jsvArrNamed(a)
		if p == nil {
			p = newJSObject()
			a.props = p
		}
		if _, has := p.props[key]; !has && jsvArrLevel(a) > 0 {
			rt.bigRaise("TypeError: Cannot define property %s, object is not extensible", jsvKeyUnesc(key))
		}
		rt.jsvDefineProp(p, key, desc)
		return
	}
	o, ok := v.(*jsObject)
	if !ok || isClassDesc(o) {
		if !isUndefOrNull(g) || !isUndefOrNull(s) {
			return
		}
		// Unlike `=`, a definition replaces a function's or a class's own `name` or
		// `length`.
		dv := rt.jsvGetKey(desc, "value")
		if cls, isStatic := jsvIsStaticSide(v, key); isStatic {
			rt.jsvStaticWrite(cls, key, dv)
		} else if p := jsvFnPropsMake(v); p != nil {
			if key == "name" || key == "length" {
				rt.jsvFnIntrDefine(v, key, desc)
			} else {
				p.set(key, dv)
			}
		} else {
			rt.setMember(v, key, dv)
		}
		return
	}
	eo, _ := desc.(*jsObject)
	has := func(k string) bool {
		if eo == nil {
			return false
		}
		_, found := eo.props[k]
		return found
	}
	flag := func(k string) bool { return rt.truthy(rt.jsvGetKey(desc, k)) }
	isAcc := !isUndefOrNull(g) || !isUndefOrNull(s)
	cur, exists := o.props[key]
	curAcc, _ := cur.(*jsAccessor)
	if !exists && o.props["__noext"] == true {
		rt.bigRaise("TypeError: Cannot define property %s, object is not extensible", jsvKeyUnesc(key))
	}
	if exists && jsvAttrHas(o, "__nc", key) {
		bad := has("configurable") && flag("configurable")
		if has("enumerable") && flag("enumerable") == jsvAttrHas(o, "__hidden", key) {
			bad = true
		}
		switch {
		case curAcc != nil:
			bad = bad || has("value") || has("writable") ||
				(isCallable(g) && g != curAcc.get) || (isCallable(s) && s != curAcc.set)
		case isAcc:
			bad = true
		case jsvAttrHas(o, "__ro", key):
			bad = bad || (has("writable") && flag("writable")) ||
				(has("value") && !rt.jsvSameValue(rt.jsvGetKey(desc, "value"), cur))
		}
		if bad {
			rt.bigRaise("TypeError: Cannot redefine property: %s", jsvKeyUnesc(key))
		}
	}
	switch {
	case isAcc:
		acc := curAcc
		if acc == nil {
			acc = &jsAccessor{}
		}
		if isCallable(g) {
			acc.get = g
		}
		if isCallable(s) {
			acc.set = s
		}
		o.set(key, acc)
		rt.accessorCount++
	case has("value") || !exists:
		o.set(key, rt.jsvGetKey(desc, "value"))
	}
	attr := func(set, name string, keep bool) {
		on := keep
		if has(name) {
			on = !flag(name)
		} else if !exists {
			on = true
		}
		jsvAttrSet(o, set, key, on)
	}
	attr("__hidden", "enumerable", jsvAttrHas(o, "__hidden", key))
	if isAcc || curAcc != nil {
		jsvAttrSet(o, "__ro", key, false)
	} else {
		attr("__ro", "writable", jsvAttrHas(o, "__ro", key))
	}
	attr("__nc", "configurable", jsvAttrHas(o, "__nc", key))
}

// jsvAttrN counts the objects holding an __ro or an __nc set or an integrity mark
// (jsvSetIntegrity), so a program that made none pays one compare per write and
// delete.
var jsvAttrN = 0

// jsvAttrHas: is key in o's attribute set (__hidden, __ro or __nc)?
func jsvAttrHas(o *jsObject, set, key string) bool {
	t, ok := o.props[set].(*jsObject)
	return ok && t.props[key] == true
}

func jsvAttrSet(o *jsObject, set, key string, on bool) {
	t, ok := o.props[set].(*jsObject)
	if !ok {
		if !on {
			return
		}
		t = newJSObject()
		o.props[set] = t
		if set != "__hidden" {
			jsvAttrN++
		}
	}
	if on {
		t.props[key] = true
	} else {
		delete(t.props, key)
	}
}

// Integrity levels, Object.preventExtensions / seal / freeze: __noext on an object,
// with __nc and __ro on every own property, and the kind word of an array (20
// frozen, 21 sealed, 22 not extensible). Every mark counts in jsvAttrN. Twin of the
// integrity block of js-interpreter.abnf (jsSetIntegrity), which states the rule;
// jvSetIntegrity in languages/lib/js-rt.metajs.
func (rt *jsrt) jsvSetIntegrity(v interface{}, lv int) interface{} {
	if jsvIsFnLike(v) {
		rt.jsvSetIntegrity(jsvFnPropsMake(v), lv)
		return v
	}
	switch o := v.(type) {
	case *jsArray:
		if jsvIsArgs(o) {
			return v
		}
		jsvAttrN++
		if lv > jsvArrLevel(o) {
			o.kind = o.kind&^31 + 23 - lv
		}
		// Its named properties are ordinary ones (jsvArrNamed).
		if p := jsvArrNamed(o); p != nil {
			rt.jsvSetIntegrity(p, lv)
		}
	case *jsObject:
		if isClassDesc(o) || jsvIsSym(o) {
			return v
		}
		jsvAttrN++
		o.props["__noext"] = true
		if lv == 1 {
			return v
		}
		for _, k := range jsvErrGopn(o, jsvOwnSlotKeys(o)) {
			jsvAttrSet(o, "__nc", k, true)
			if _, isAcc := o.props[k].(*jsAccessor); lv == 3 && !isAcc {
				jsvAttrSet(o, "__ro", k, true)
			}
		}
	}
	return v
}

// jsvOwnSlotKeys are o's own property keys, the non-enumerable ones included.
func jsvOwnSlotKeys(o *jsObject) []string {
	out := []string{}
	for _, k := range o.keys {
		if !jsvInternalKey(k) {
			out = append(out, k)
		}
	}
	return out
}

// jsvArrLevel is an array's integrity level: 3 frozen, 2 sealed, 1 not extensible.
func jsvArrLevel(a *jsArray) int {
	if k := a.kind & 31; k >= 20 && k <= 22 {
		return 23 - k
	}
	return 0
}

// jsvIntegrity is TestIntegrityLevel as a number: 3 frozen, 2 sealed, 1 not
// extensible, 0 none.
func jsvIntegrity(v interface{}) int {
	if jsvIsFnLike(v) {
		if p := jsvFnPropsOf(v); p != nil {
			return jsvIntegrity(p)
		}
		return 0
	}
	switch o := v.(type) {
	case *jsArray:
		if lv := jsvArrLevel(o); lv != 1 || len(o.elems) > 0 {
			return lv
		}
		return 2
	case *jsObject:
		if isClassDesc(o) {
			return 0
		}
		if jsvIsSym(o) {
			return 3
		}
		if o.props["__noext"] != true {
			return 0
		}
		lv := 3
		for _, k := range jsvErrGopn(o, jsvOwnSlotKeys(o)) {
			if !jsvAttrHas(o, "__nc", k) {
				return 1
			}
			if _, isAcc := o.props[k].(*jsAccessor); !isAcc && !jsvAttrHas(o, "__ro", k) {
				lv = 2
			}
		}
		return lv
	}
	return 3
}

// jsvIsExtensible is Object.isExtensible.
func jsvIsExtensible(v interface{}) bool {
	switch o := v.(type) {
	case *jsArray, *jsObject:
		if s, isObj := o.(*jsObject); isObj && jsvIsSym(s) {
			return false
		}
		return jsvIntegrity(v) == 0
	}
	return jsvIsFnLike(v) && jsvIntegrity(v) == 0
}

// jsvObjText is V8's rendering of the object in a refused write's message.
func (rt *jsrt) jsvObjText(v interface{}) string {
	o, isObj := v.(*jsObject)
	switch {
	case !isObj:
		return "[object Array]"
	case o.props["__errh"] != nil:
		return rt.jsvString(o)
	case jsvNoObjProto(o):
		return "[object Object]"
	}
	name := ""
	switch c := rt.jsvBuiltinCtor(o).(type) {
	case *jsObject:
		name, _ = c.props["__name"].(string)
	default:
		name = jsvFnMeta(c).name
	}
	return jsvCtorText(name)
}

// jsvCtorText is V8's "#<Name>" for an object whose constructor's name is name.
func jsvCtorText(name string) string {
	if name == "" {
		name = "Object"
	}
	return "#<" + name + ">"
}

func (rt *jsrt) jsvRoText(o interface{}, key string) string {
	return "Cannot assign to read only property '" + jsvKeyUnesc(key) + "' of object '" + rt.jsvObjText(o) + "'"
}

// jsvFnRoText is V8's text for a refused write to a function's `name` or `length`,
// which names the function by its String(f). Twin of jsFnRoText in both
// *-interpreter.abnf.
func (rt *jsrt) jsvFnRoText(f interface{}, key string) string {
	return "Cannot assign to read only property '" + key + "' of function '" + jsvElide(rt.jsvString(f)) + "'"
}

// jsvElide is a function's text inside a V8 error message: past 128 UTF-16 units, its
// first 111 and its last 2 around "...<omitted>...". Twin of fsElide in
// languages/lib/js-fnsrc.js.
func jsvElide(t string) string {
	u := strUnits(t)
	if len(u) <= 128 {
		return t
	}
	return strFromUnits(u[:111]) + "...<omitted>..." + strFromUnits(u[len(u)-2:])
}

// jsvOwnsStatic answers whether class cls itself (not a superclass) declares a static
// member name.
func jsvOwnsStatic(cls *jsObject, name string) bool {
	t := jsStaticsOf(cls, false)
	if t == nil {
		return false
	}
	_, found := t.props[name]
	return found
}

// jsvStaticDel is `delete C.k` on a class object: the static member leaves the table,
// its key order and the enumerable list, so a read falls through to the parent class.
// A deleted own `name` or `length` reads the parent's, or Function.prototype's
// (jsvStaticSideRead). Twins: staticDelete in both *-interpreter.abnf, jvStaticDel in
// languages/lib/js-rt.metajs.
func jsvStaticDel(cls *jsObject, name string) {
	if name == "name" || name == "length" {
		cls.props["__del"+name] = true
	}
	t := jsStaticsOf(cls, false)
	if t == nil {
		return
	}
	jsvObjDel(t, name)
	if lst, _ := t.props[jsvSFieldKeys].(*jsArray); lst != nil {
		for i, e := range lst.elems {
			if s, ok := e.(string); ok && s == name {
				lst.elems = append(lst.elems[:i], lst.elems[i+1:]...)
				break
			}
		}
	}
}

// jsvObjDel drops an own property and its key-order entry.
// jsvArrNamed is the object holding an array's NAMED properties (jsArray.props,
// which the shared setMember fills), or nil while it has none. The indices come
// first in every key list and the names after them, in insertion order, which is
// node's OrdinaryOwnPropertyKeys. Twin: jvArrNamed in languages/lib/js-rt.metajs.
func jsvArrNamed(a *jsArray) *jsObject {
	p, _ := a.props.(*jsObject)
	return p
}

// jsvArrAccRead answers (value, true) when v is an array whose named property k is an
// accessor: its getter's answer with the array as `this`, or undefined when it has
// none. The shared getMember hands the accessor box back as it is.
func (rt *jsrt) jsvArrAccRead(v interface{}, k string) (interface{}, bool) {
	acc := jsvArrAccessor(v, k)
	if acc == nil {
		return nil, false
	}
	if acc.get == nil {
		return jsUndef, true
	}
	return rt.call(acc.get, v, nil), true
}

// jsvArrAccessor is the accessor v's named property k holds when v is an array, or nil.
func jsvArrAccessor(v interface{}, k string) *jsAccessor {
	if a, isArr := v.(*jsArray); isArr {
		if p := jsvArrNamed(a); p != nil {
			acc, _ := p.props[k].(*jsAccessor)
			return acc
		}
	}
	return nil
}

// jsvArrNamedHas: does array a hold the named property k?
func jsvArrNamedHas(a *jsArray, k string) bool {
	if p := jsvArrNamed(a); p != nil {
		_, has := p.props[k]
		return has
	}
	return false
}

func jsvObjDel(o *jsObject, name string) {
	if _, found := o.props[name]; !found {
		return
	}
	delete(o.props, name)
	for i, k := range o.keys {
		if k == name {
			o.keys = append(o.keys[:i], o.keys[i+1:]...)
			break
		}
	}
}

// jsvStaticWrite is a write onto class cls's static namespace: a setter anywhere on
// the static chain wins over a fresh own slot, the same rule js_set follows for an
// ordinary object. It answers the class whose static accessor has no setter, which
// refuses the write, else nil. Twin of jvStaticWrite in languages/lib/js-rt.metajs.
func (rt *jsrt) jsvStaticWrite(cls *jsObject, name string, v interface{}) *jsObject {
	if sv, holder, found := jsStaticsFind(cls, name); found {
		if acc, isAcc := sv.(*jsAccessor); isAcc {
			if acc.set == nil {
				return holder
			}
			rt.call(acc.set, cls, []interface{}{v})
			return nil
		}
	}
	t := jsStaticsOf(cls, true)
	_, had := t.props[name]
	t.set(name, v)
	// A runtime `C.zz = 1` creates an ENUMERABLE own property of the class object,
	// exactly as it does on any other object; a write over an existing one (a static
	// method, say) keeps its enumerability.
	if !had {
		jsvSfTrack(t, name)
	}
	return nil
}

// jsvAccHolder answers the object holding name's accessor for a write to o and that
// accessor: o itself, else the nearest class descriptor or prototype object on its
// __class chain, or nil when the write meets a data property or nothing. Twins: accHolder in both
// *-interpreter.abnf, jvClassAccHolder in languages/lib/js-rt.metajs.
func jsvAccHolder(o *jsObject, name string) (*jsObject, *jsAccessor) {
	if v, own := o.props[name]; own {
		if acc, isAcc := v.(*jsAccessor); isAcc {
			return o, acc
		}
		return nil, nil
	}
	// A prototype view starts at its own descriptor, anything else at its class.
	start := o.props["__class"]
	if d, isView := o.props["__protoof"]; isView {
		start = d
	}
	for cls, guard := start, 0; cls != nil && guard < 64; guard++ {
		c, ok := cls.(*jsObject)
		if !ok {
			break
		}
		if v, found := c.props[name]; found {
			if acc, isAcc := v.(*jsAccessor); isAcc {
				return c, acc
			}
			return nil, nil
		}
		if pv, hasView := c.props["__pview"].(*jsObject); hasView {
			if acc, isAcc := pv.props[name].(*jsAccessor); isAcc {
				return pv, acc
			}
		}
		cls = c.props["__super"]
	}
	return nil, nil
}

// jsvGetterOnlyText is V8's TypeError text for a write to an accessor with no setter,
// naming the object holding it. Twins: jsGetterOnlyText in both *-interpreter.abnf,
// jvGetterOnlyText in languages/lib/js-rt.metajs.
func jsvGetterOnlyText(key, holder string) string {
	return "Cannot set property " + jsvKeyUnesc(key) + " of " + holder + " which has only a getter"
}

// jsvAccHolderText names holder h of o's accessor: o as a refused write names it, a
// class by its own name, an anonymous one as "[object Object]". Twins: jsAccHolderText
// in both *-interpreter.abnf, jvAccHolderText in languages/lib/js-rt.metajs.
func (rt *jsrt) jsvAccHolderText(o, h *jsObject) string {
	if h == o {
		return rt.jsvObjText(o)
	}
	// A prototype is named by its class or constructor function, any other object by
	// its constructor.
	if d, isView := h.props["__protoof"].(*jsObject); isView {
		h = d
	}
	if fn, isFn := h.props["__ctorfn"]; isFn {
		return jsvCtorText(jsvFnMeta(fn).name)
	}
	if !isClassDesc(h) {
		return rt.jsvObjText(h)
	}
	if n, _ := h.props["__name"].(string); n != "" {
		return "#<" + n + ">"
	}
	return "[object Object]"
}

func jsvAddText(key string) string {
	return "Cannot add property " + jsvKeyUnesc(key) + ", object is not extensible"
}

func (rt *jsrt) jsvDelText(o interface{}, key string) string {
	return "Cannot delete property '" + jsvKeyUnesc(key) + "' of " + rt.jsvObjText(o)
}

// jsvWriteRefusal is the TypeError text a write of v to o[key] meets under o's
// marks, or "".
func (rt *jsrt) jsvWriteRefusal(o interface{}, key string, v interface{}) string {
	switch t := o.(type) {
	case *jsArray:
		if p := jsvArrNamed(t); p != nil && jsvAttrHas(p, "__ro", key) {
			return rt.jsvRoText(o, key)
		}
		lv := jsvArrLevel(t)
		n := len(t.elems)
		switch {
		case lv == 0:
			return ""
		case key == "length" && lv == 3:
			return rt.jsvRoText(o, key)
		case key == "length" && lv == 2 && rt.toNumber(v) < float64(n):
			return rt.jsvDelText(o, strconv.Itoa(n-1))
		case key == "length":
			return ""
		}
		if ix, isIdx := jsvArrayIndex(key); (!isIdx && !jsvArrNamedHas(t, key)) || (isIdx && ix >= uint64(n)) {
			return jsvAddText(key)
		}
		if lv == 3 {
			return rt.jsvRoText(o, key)
		}
	case *jsObject:
		if jsvAttrHas(t, "__ro", key) {
			return rt.jsvRoText(o, key)
		}
		if _, own := t.props[key]; !own && t.props["__noext"] == true && rt.findClassAccessor(t, key) == nil {
			return jsvAddText(key)
		}
	}
	return ""
}

// jsvDeleteRefusal is the TypeError text a delete of o[key] meets, or "".
func (rt *jsrt) jsvDeleteRefusal(o interface{}, key string) string {
	switch t := o.(type) {
	case *jsArray:
		if p := jsvArrNamed(t); p != nil && jsvAttrHas(p, "__nc", key) {
			return rt.jsvDelText(o, key)
		}
		if ix, isIdx := jsvArrayIndex(key); isIdx && ix < uint64(len(t.elems)) && jsvArrLevel(t) >= 2 {
			return rt.jsvDelText(o, key)
		}
	case *jsObject:
		if jsvAttrHas(t, "__nc", key) {
			return rt.jsvDelText(o, key)
		}
	}
	return ""
}

// jsvRefused is the text of the last write or delete the marks refused, which
// js_jsrefuse raises in strict code.
var jsvRefused = ""

// jsvArrMethodRefusal is the TypeError an Array method meets on an array of
// integrity level lv, or "" when it runs: the spec's first refused Set or
// DeletePropertyOrThrow, after making the writes the level allows before it. Twin of
// jsArrMethodRefusal in both *-interpreter.abnf.
func (rt *jsrt) jsvArrMethodRefusal(a *jsArray, name string, args []interface{}, lv int) string {
	n := len(a.elems)
	switch name {
	case "push", "unshift":
		if len(args) == 0 {
			if lv == 3 {
				return rt.jsvRoText(a, "length")
			}
			return ""
		}
		if name == "unshift" && n > 0 {
			return jsvAddText(strconv.Itoa(n + len(args) - 1))
		}
		return jsvAddText(strconv.Itoa(n))
	case "splice":
		st := n
		if len(args) > 0 {
			st = jsvRelIndex(rt.toNumber(args[0]), n)
		}
		dc := n - st
		if len(args) > 1 {
			dc = jsToInt(rt.toNumber(args[1]))
		}
		if dc < 0 {
			dc = 0
		}
		if dc > n-st {
			dc = n - st
		}
		ins := 0
		if len(args) > 2 {
			ins = len(args) - 2
		}
		tail := st+dc < n
		switch {
		case ins > dc && tail:
			return jsvAddText(strconv.Itoa(n - dc + ins - 1))
		case ins > dc && lv == 3 && dc > 0:
			return rt.jsvRoText(a, strconv.Itoa(st))
		case ins > dc:
			a.dropIdx()
			copy(a.elems[st:], args[2:2+dc])
			return jsvAddText(strconv.Itoa(n))
		case lv == 1:
			return ""
		case ins < dc && lv == 3 && tail:
			return rt.jsvRoText(a, strconv.Itoa(st+ins))
		case ins < dc:
			if lv < 3 {
				a.dropIdx()
				copy(a.elems[st+ins:], a.elems[st+dc:])
			}
			return rt.jsvDelText(a, strconv.Itoa(n-1))
		case lv < 3:
			return ""
		case ins > 0:
			return rt.jsvRoText(a, strconv.Itoa(st))
		}
		return rt.jsvRoText(a, "length")
	case "pop", "shift", "sort", "reverse", "fill", "copyWithin":
	default:
		return ""
	}
	if lv == 1 {
		return ""
	}
	switch name {
	case "pop", "shift":
		if n == 0 {
			if lv == 3 {
				return rt.jsvRoText(a, "length")
			}
			return ""
		}
		if lv == 3 && name == "shift" && n > 1 {
			return rt.jsvRoText(a, "0")
		}
		if name == "shift" {
			rt.jsvArrMoveDown(a, name, args)
		}
		return rt.jsvDelText(a, strconv.Itoa(n-1))
	}
	if lv < 3 {
		return ""
	}
	first := -1
	switch name {
	case "fill":
		begin := jsvRelIndex(rt.jsvNumArg(args, 1, 0), n)
		end := n
		if len(args) > 2 && args[2] != jsUndef {
			end = jsvRelIndex(rt.jsvToNum(args[2]), n)
		}
		if begin < end {
			first = begin
		}
	case "copyWithin":
		to := jsvRelIndex(rt.jsvNumArg(args, 0, 0), n)
		from := jsvRelIndex(rt.jsvNumArg(args, 1, 0), n)
		end := n
		if len(args) > 2 && args[2] != jsUndef {
			end = jsvRelIndex(rt.jsvToNum(args[2]), n)
		}
		count := end - from
		if n-to < count {
			count = n - to
		}
		// An overlapping copy to a higher index runs backwards from its last slot.
		if count > 0 && from < to && to < from+count {
			first = to + count - 1
		} else if count > 0 {
			first = to
		}
	case "reverse", "sort":
		// V8's sort, unlike the spec's, writes nothing back for fewer than two
		// elements.
		if n > 1 {
			first = 0
		}
	}
	if first < 0 {
		return ""
	}
	return rt.jsvRoText(a, strconv.Itoa(first))
}

// jsvArrMoveDown writes the elements a shrinking method leaves over a, keeping its
// length.
func (rt *jsrt) jsvArrMoveDown(a *jsArray, name string, args []interface{}) {
	cp := &jsArray{elems: append([]interface{}{}, a.elems...)}
	rt.jsvArrayMethod(cp, name, args)
	copy(a.elems, cp.elems)
}

// jsvSameValue is SameValue: === except that NaN is itself and +0 is not -0. Twins:
// jvSameValue in languages/lib/js-rt.metajs, sameValue in both *-interpreter.abnf.
func (rt *jsrt) jsvSameValue(a, b interface{}) bool {
	x, xNum := a.(float64)
	y, yNum := b.(float64)
	if xNum && yNum {
		if x != x {
			return y != y
		}
		if x == 0 && y == 0 {
			return math.Signbit(x) == math.Signbit(y)
		}
	}
	return rt.strictEq(a, b)
}

// jsvDefineProps is Object.defineProperties(o, props), and Object.create's second
// argument: one jsvDefineProp per own enumerable key of props.
func (rt *jsrt) jsvDefineProps(o, props interface{}) interface{} {
	if isUndefOrNull(props) {
		rt.bigRaise("TypeError: Cannot convert undefined or null to object")
	}
	for _, k := range jsvOwnKeys(props) {
		rt.jsvDefineProp(o, k, rt.jsvGetKey(props, k))
	}
	return o
}

// jsvOwnProto is the prototype a program wrote onto the function fn
// (`F.prototype = p`), if it wrote one.
func jsvOwnProto(fn interface{}) (interface{}, bool) {
	fp := jsvFnPropsOf(fn)
	if fp == nil {
		return nil, false
	}
	p, has := fp.props["prototype"]
	return p, has
}

// jsvProtoFns are Object.prototype's methods READ AS A VALUE: one function per name,
// the one Object.prototype holds, so `o.valueOf === o.valueOf` and
// `o.valueOf === Object.prototype.valueOf` as in node. Each takes its receiver from
// the call site.
var jsvProtoFns = func() map[string]*hostFunc {
	out := map[string]*hostFunc{}
	for _, n := range jsvProtoNames {
		name := n
		out[name] = jsvNativeFn(jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return rt.jsvProtoApply(rt.unwrap(this), name, args)
		}), name, 0)
	}
	return out
}()

func jsvProtoShim(name string) *hostFunc { return jsvProtoFns[name] }

// jsvPropEnum is propertyIsEnumerable: an own ENUMERABLE key, which is exactly
// Object.keys - an array's `length`, an Error's `message` and a function's `name` are
// own and not enumerable.
func jsvPropEnum(v interface{}, k string) bool {
	for _, x := range jsvOwnKeys(v) {
		if x == k {
			return true
		}
	}
	return false
}

// The own property names of the Object and Array globals, in node's order. None of
// them is enumerable. Twins: jsObjectOwnNames / jsArrayOwnNames in both
// *-interpreter.abnf and in languages/lib/js-rt.metajs.
var jsvObjectOwnNames = []string{"length", "name", "prototype", "assign", "getOwnPropertyDescriptor",
	"getOwnPropertyDescriptors", "getOwnPropertyNames", "getOwnPropertySymbols", "hasOwn", "is",
	"preventExtensions", "seal", "create", "defineProperties", "defineProperty", "freeze",
	"getPrototypeOf", "setPrototypeOf", "isExtensible", "isFrozen", "isSealed", "keys", "entries",
	"fromEntries", "values", "groupBy"}
var jsvArrayOwnNames = []string{"length", "name", "prototype", "isArray", "from", "fromAsync", "of"}

// The own property names of the Math and JSON globals, in node 24's order. None of
// them is enumerable. Twins: jsMathOwnNames / jsJsonOwnNames in both
// *-interpreter.abnf, jvMathOwnNames / jvJsonOwnNames in languages/lib/js-rt.metajs.
var jsvMathOwnNames = []string{"abs", "acos", "acosh", "asin", "asinh", "atan", "atanh", "atan2", "ceil",
	"cbrt", "expm1", "clz32", "cos", "cosh", "exp", "floor", "fround", "hypot", "imul", "log", "log1p",
	"log2", "log10", "max", "min", "pow", "random", "round", "sign", "sin", "sinh", "sqrt", "tan", "tanh",
	"trunc", "E", "LN10", "LN2", "LOG10E", "LOG2E", "PI", "SQRT1_2", "SQRT2", "f16round"}
var jsvJSONOwnNames = []string{"parse", "stringify", "rawJSON", "isRawJSON"}

// jsvObjProtoName: Object.prototype's own names, which `k in o` finds on every
// object. Twins: objProtoHas in both *-interpreter.abnf, jvObjProtoHas in
// languages/lib/js-rt.metajs.
func jsvObjProtoName(k string) bool {
	switch k {
	case "constructor", "__defineGetter__", "__defineSetter__", "hasOwnProperty", "__lookupGetter__",
		"__lookupSetter__", "isPrototypeOf", "propertyIsEnumerable", "toString", "valueOf", "__proto__",
		"toLocaleString":
		return true
	}
	return false
}

// jsvStaticNames: the static methods of each builtin global that js_jsmcall answers.
// Twins: stNames in both *-interpreter.abnf, jvStaticHas in
// languages/lib/js-rt.metajs.
var jsvStaticNames = map[string][]string{
	"Object": {"assign", "create", "entries", "freeze", "fromEntries", "getOwnPropertyNames", "getOwnPropertySymbols", "groupBy",
		"hasOwn", "is", "keys", "seal", "values", "defineProperties", "defineProperty", "getPrototypeOf", "setPrototypeOf",
		"preventExtensions", "isExtensible", "isFrozen", "isSealed", "getOwnPropertyDescriptor", "getOwnPropertyDescriptors"},
	"String":  {"fromCharCode", "fromCodePoint", "raw"},
	"Promise": {"resolve", "reject", "all", "allSettled", "race", "any"},
	"Array":   {"isArray"},
	"Symbol":  {"for", "keyFor"},
	"Number":  {"isFinite", "isInteger", "isNaN", "isSafeInteger", "parseFloat", "parseInt"},
}

// jsvStaticLen2: the statics whose node length is 2; every other one's is 1 but
// Object.defineProperty's, which is 3.
var jsvStaticLen2 = []string{"assign", "create", "is", "hasOwn", "groupBy", "parseInt", "defineProperties", "setPrototypeOf",
	"getOwnPropertyDescriptor"}

// jsvNumberConst: Number's constant members. Twins: jsNumberConst in both
// *-interpreter.abnf, jvNumberConst in languages/lib/js-rt.metajs.
func jsvNumberConst(k string) (float64, bool) {
	switch k {
	case "MAX_SAFE_INTEGER":
		return 9007199254740991, true
	case "MIN_SAFE_INTEGER":
		return -9007199254740991, true
	case "EPSILON":
		return math.Pow(2, -52), true
	case "MAX_VALUE":
		return math.MaxFloat64, true
	case "MIN_VALUE":
		return math.SmallestNonzeroFloat64, true
	case "POSITIVE_INFINITY":
		return math.Inf(1), true
	case "NEGATIVE_INFINITY":
		return math.Inf(-1), true
	case "NaN":
		return math.NaN(), true
	}
	return 0, false
}

func jsvInNames(names []string, k string) bool {
	for _, x := range names {
		if x == k {
			return true
		}
	}
	return false
}

// jsvString is String(v). An object with a toString of its own answers with it (the
// method convention prepends the receiver, exactly as rt.memberCall does).
func (rt *jsrt) jsvString(v interface{}) string {
	switch t := v.(type) {
	case *jsObject:
		if rt.jsSrc != nil && isClassDesc(t) {
			return rt.jsvClassText(t)
		}
		if d, isSym := t.props["__symdesc"]; isSym {
			if d == jsUndef {
				return "Symbol()"
			}
			return "Symbol(" + rt.toString(d) + ")"
		}
		if r, ok := rt.jsvExoticPrim(t, "string"); ok {
			return rt.jsvString(r)
		}
		if r, ok := rt.jsvCallMethod(t, "toString"); ok {
			return rt.toString(r)
		}
		if p, isWrap := jsvWrapped(t); isWrap {
			return rt.jsvString(p)
		}
		// OrdinaryToPrimitive finds no toString and no valueOf to call.
		if jsvNoObjProto(t) {
			rt.bigRaise("TypeError: Cannot convert object to primitive value")
		}
		return rt.jsvProtoTag(t)
	case *jsArray:
		if jsvIsArgs(t) {
			return "[object Arguments]"
		}
		if cls := jsvArrClass(t); cls != nil {
			if r, ok := rt.jsvClassCall(t, cls, "toString", nil); ok {
				return rt.jsvString(r)
			}
		}
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

// jsvSymWKNames are the well-known symbols, the static members of Symbol. Twins:
// jsSymWKNames in js-interpreter.abnf, jvSymIsWK in languages/lib/js-rt.metajs.
var jsvSymWKNames = []string{"asyncIterator", "hasInstance", "isConcatSpreadable", "iterator", "match",
	"matchAll", "replace", "search", "species", "split", "toPrimitive", "toStringTag", "unscopables"}

// jsvIsSym reports a symbol value (see the SYMBOLS note in addJSValueExterns).
func jsvIsSym(v interface{}) bool {
	o, ok := v.(*jsObject)
	if !ok {
		return false
	}
	_, isSym := o.props["__symkey"]
	return isSym
}

// jsvSymConvErr is the TypeError of ToString / ToNumber on a symbol: `"" + s`, `${s}`
// and `s * 1` throw where String(s) and s.toString() answer "Symbol(desc)". Twin of
// jsSymConvErr in js-interpreter.abnf and jvSymConvErr in languages/lib/js-rt.metajs.
func (rt *jsrt) jsvSymConvErr(toStr bool) {
	what := "number"
	if toStr {
		what = "string"
	}
	panic(&jsThrown{value: rt.jsErrFromText("TypeError: Cannot convert a Symbol value to a " + what)})
}

// jsvSymOperand raises the ToNumber TypeError when either operand is a symbol.
func (rt *jsrt) jsvSymOperand(l, r interface{}) {
	if jsvIsSym(l) || jsvIsSym(r) {
		rt.jsvSymConvErr(false)
	}
}

// jsvSymKeys are an object's own symbol keys in insertion order: the ones no key
// listing shows and object spread and Object.assign still copy.
func jsvSymKeys(v interface{}) []string {
	o, ok := v.(*jsObject)
	if !ok || isClassDesc(o) {
		return nil
	}
	var out []string
	for _, k := range o.keys {
		if _, has := o.props[k]; has && strings.HasPrefix(k, "__#@") {
			out = append(out, k)
		}
	}
	return out
}

// jsvCodePoints is a string's CODE POINTS: a surrogate pair is one, a lone surrogate
// one of its own.
func jsvCodePoints(s string) []interface{} {
	var cps []interface{}
	u := strUnits(s)
	for i := 0; i < len(u); {
		n := 1
		if u[i] >= 0xD800 && u[i] <= 0xDBFF && i+1 < len(u) && u[i+1] >= 0xDC00 && u[i+1] <= 0xDFFF {
			n = 2
		}
		cps = append(cps, strFromUnits(u[i:i+n]))
		i += n
	}
	return cps
}

// jsvSeqIter is the iterator [Symbol.iterator](), values(), keys() and entries()
// answer over an array or a string: kind 1 yields the elements - a string's CODE
// POINTS - kind 0 the indices and kind 2 [index, element] pairs. An array is read
// live. Its own [Symbol.iterator] answers itself. Twins: jsSeqIter in
// js-interpreter.abnf, jvSeqIter in languages/lib/js-rt.metajs.
func jsvSeqIter(src interface{}, kind int) *jsObject {
	var cps []interface{}
	if s, isStr := src.(string); isStr {
		cps = jsvCodePoints(s)
	}
	it := newJSObject()
	i := 0
	it.props["next"] = jsHostFunc("next", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		elems := cps
		if a, isArr := src.(*jsArray); isArr {
			elems = a.elems
			if jsvIsArgs(a) {
				elems = jsvArgsView(a).elems
			}
		}
		r := newJSObject()
		if i < len(elems) {
			switch kind {
			case 0:
				r.set("value", float64(i))
			case 2:
				r.set("value", &jsArray{elems: []interface{}{float64(i), elems[i]}})
			default:
				r.set("value", elems[i])
			}
			r.set("done", false)
			i++
		} else {
			r.set("value", jsUndef)
			r.set("done", true)
		}
		return r
	})
	it.props["__#@iterator"] = jsHostFunc("[Symbol.iterator]", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return it
	})
	return it
}

// jsvToPrimitive is ToPrimitive(v, hint) for the hints "default" and "number":
// [Symbol.toPrimitive] first, then valueOf, then toString. It is the step the shared
// js_eq / js_lt lack: `[1] == 1` and `[2] > 1` are true in node.
func (rt *jsrt) jsvToPrimitive(v interface{}, hint string) interface{} {
	if o, ok := v.(*jsObject); ok {
		if r, ok := rt.jsvExoticPrim(o, hint); ok {
			return r
		}
		if r, ok := rt.jsvCallMethod(o, "valueOf"); ok && !jsvIsObjLike(r) {
			return r
		}
		if p, isWrap := jsvWrapped(o); isWrap {
			return p
		}
		return rt.jsvString(o)
	}
	if _, ok := v.(*jsArray); ok {
		return rt.jsvString(v)
	}
	if rt.jsSrc != nil && jsvIsFnLike(v) {
		return rt.jsvString(v)
	}
	return v
}

// jsvSymMember reads o's member under a symbol key: an own or accessor property, else
// a method of its class, bound to o. nil when there is none.
func (rt *jsrt) jsvSymMember(o *jsObject, key string) interface{} {
	if _, isSym := o.props["__symkey"]; isSym {
		return nil
	}
	if v := rt.getMember(o, key); !isUndefOrNull(v) {
		return v
	}
	if fn, found := jsvFindClassMethod(o, key); found {
		return jsHostFunc(key, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return rt.call(fn, jsUndef, append([]interface{}{o}, args...))
		})
	}
	return nil
}

// jsvExoticPrim is ToPrimitive's first step: o's [Symbol.toPrimitive], called with
// the hint; an object result is node's TypeError. Twins: jsExoticPrim in
// js-interpreter.abnf, jvExoticPrim in languages/lib/js-rt.metajs.
func (rt *jsrt) jsvExoticPrim(o *jsObject, hint string) (interface{}, bool) {
	f := rt.jsvSymMember(o, "__#@toPrimitive")
	if f == nil {
		return nil, false
	}
	rt.jsvNeedFn(f)
	r := rt.call(f, o, []interface{}{hint})
	if jsvIsObjLike(r) {
		panic(&jsThrown{value: rt.jsErrFromText("TypeError: Cannot convert object to primitive value")})
	}
	return r, true
}

// jsvNeedFn is GetMethod's TypeError (ES2023 7.3.11) for a well-known-symbol member
// that is present but not callable, in V8's words: "number 1 is not a function",
// 'string "x" ...', "object ...". Twins: jsNeedFn in both *-interpreter.abnf,
// jvNeedFn in languages/lib/js-rt.metajs.
func (rt *jsrt) jsvNeedFn(f interface{}) {
	if isCallable(f) {
		return
	}
	what := "object"
	switch t := f.(type) {
	case float64:
		what = "number " + rt.jsvString(t)
	case string:
		what = "string \"" + t + "\""
	case bool:
		what = "boolean " + rt.jsvString(t)
	default:
		if _, ok := bigOf(f); ok {
			what = "bigint"
		} else if jsvIsSym(f) {
			what = "symbol"
		}
	}
	panic(&jsThrown{value: rt.jsErrFromText("TypeError: " + what + " is not a function")})
}

// jsvLooseEq is the abstract equality comparison. null == undefined is true, both are
// equal to nothing else, two objects compare by identity and an object against a
// primitive compares its primitive form.
func (rt *jsrt) jsvLooseEq(a, b interface{}) bool {
	if isUndefOrNull(a) || isUndefOrNull(b) {
		return isUndefOrNull(a) && isUndefOrNull(b)
	}
	// A function of a program that has its source text is an object here too: equal
	// to itself alone, and its text against a primitive.
	ao, bo := jsvIsObjLike(a), jsvIsObjLike(b)
	if rt.jsSrc != nil {
		ao, bo = ao || jsvIsFnLike(a), bo || jsvIsFnLike(b)
	}
	if ao && bo {
		return rt.strictEq(a, b)
	}
	// A symbol is a primitive equal to itself alone.
	if jsvIsSym(a) || jsvIsSym(b) {
		return false
	}
	if ao {
		return rt.jsvLooseEq(rt.jsvToPrimitive(a, "default"), b)
	}
	if bo {
		return rt.jsvLooseEq(a, rt.jsvToPrimitive(b, "default"))
	}
	// A string against a number or a boolean compares as numbers, by StringToNumber.
	_, as := a.(string)
	_, bs := b.(string)
	if as != bs {
		if _, ok := bigOf(a); !ok {
			if _, ok := bigOf(b); !ok {
				x, y := rt.jsvToNum(a), rt.jsvToNum(b)
				return x == y
			}
		}
	}
	return rt.looseEq(a, b)
}

// jsvCompare is the abstract relational comparison: ToPrimitive on both sides, then
// the shared numeric/string ordering (2 means "NaN was involved", every relation false).
// jsvStrCmp orders two strings by their UTF-16 code units, which is JavaScript's
// string order (ECMA-262 IsLessThan) and not the code point order Go's `<` gives
// a WTF-8 string: an astral character sorts below U+E000..U+FFFF, and a lone
// surrogate is a unit like any other. Only the character at the first differing
// byte decides, through its first unit and, for two astral characters behind one
// high surrogate, the second. Twin of jsnStrCmp in languages/lib/js-num.js.
func jsvStrCmp(a, b string) int {
	i := 0
	for i < len(a) && i < len(b) && a[i] == b[i] {
		i++
	}
	if i == len(a) || i == len(b) {
		switch {
		case len(a) < len(b):
			return -1
		case len(a) > len(b):
			return 1
		}
		return 0
	}
	for i > 0 && a[i]&0xC0 == 0x80 {
		i--
	}
	ca, cb := wtf8Rune(a[i:]), wtf8Rune(b[i:])
	ka, kb := ca, cb
	if ka > 0xFFFF {
		ka = 0xD800 + (ka-0x10000)>>10
	}
	if kb > 0xFFFF {
		kb = 0xD800 + (kb-0x10000)>>10
	}
	if ka == kb {
		ka, kb = ca, cb
	}
	if ka < kb {
		return -1
	}
	return 1
}

// wtf8Rune decodes the character at the start of s, a lone surrogate as itself.
func wtf8Rune(s string) rune {
	if c := wtf8Surrogate(s); c != 0 {
		return rune(c)
	}
	r, _ := utf8.DecodeRuneInString(s)
	return r
}

func (rt *jsrt) jsvCompare(a, b interface{}) int {
	rt.jsvSymOperand(a, b)
	pa, pb := rt.jsvToPrimitive(a, "number"), rt.jsvToPrimitive(b, "number")
	// Unless both sides are strings, a string side is read by StringToNumber.
	sa, as := pa.(string)
	sb, bs := pb.(string)
	if as != bs {
		if as {
			pa = jsStrToNum(sa)
		} else {
			pb = jsStrToNum(sb)
		}
	} else if as {
		return jsvStrCmp(sa, sb)
	}
	return rt.jsCompare(pa, pb)
}

// ============================================================================
// The Object global
//
// The root scope's `Object` is a bare host object carrying only `prototype`, so the
// grammar declares the host function js_jsobject makes in the PROGRAM's own scope (the
// way it already declares RegExp), and nothing outside the JS/TS pair sees it.
func jsvIsObjectGlobal(v interface{}) bool {
	hf, ok := v.(*hostFunc)
	return ok && hf.name == "Object"
}

// jsvIsArrayGlobal recognizes the `Array` global: a js program's own (js_jsarrayfn)
// or the root scope's binding, by its one member.
func jsvIsArrayGlobal(v interface{}) bool {
	if hf, ok := v.(*hostFunc); ok {
		return hf.name == "Array"
	}
	o, ok := v.(*jsObject)
	if !ok {
		return false
	}
	_, has := o.props["isArray"]
	return has && len(o.props) == 1
}

// jsvOwnKeys are an object's own enumerable keys in insertion order, WITHOUT the
// __-prefixed internal slots (__class, __keys, __super, ...).
func jsvOwnKeys(v interface{}) []string {
	switch o := v.(type) {
	case *jsObject:
		// A CLASS DESCRIPTOR answers with the enumerable half of its STATIC
		// namespace, and with nothing off its own property map: that map holds the
		// class's INSTANCE methods, which node reports on the prototype and never on
		// the constructor (node says ["sf"] for
		// `Object.keys(class C { static sf = 1; m(){} })`).
		if isClassDesc(o) {
			return jsvStaticEnumKeys(o)
		}
		d, isView := o.props["__protoof"].(*jsObject)
		// A property defineProperty made non-enumerable (jsvDefineProp).
		hid, _ := o.props["__hidden"].(*jsObject)
		out := []string{}
		// A String wrapper's own keys begin with its string's indices.
		if p, isStr := o.props["__bvof"].(string); isStr {
			out = jsvOwnKeys(p)
		}
		for _, k := range o.keys {
			if !jsvInternalKey(k) && !(isView && jsvProtoDeclared(d, k)) && (hid == nil || hid.props[k] != true) {
				out = append(out, k)
			}
		}
		return jsvErrHide(o, jsvIndexOrder(out))
	case *jsArray:
		if jsvIsArgs(o) {
			return jsvArgsKeys(o)
		}
		out := make([]string, 0, len(o.elems))
		for i, e := range o.elems {
			if e != jsHole {
				out = append(out, strconv.Itoa(i))
			}
		}
		if p := jsvArrNamed(o); p != nil {
			out = append(out, jsvOwnKeys(p)...)
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
	// and this engine keeps in a side table.
	if p := jsvFnPropsOf(v); p != nil {
		ks := jsvOwnKeys(p)
		// The Array global's `from` and `of` are lib/js-builtins.js's, written onto
		// it as ordinary properties; node's are not enumerable. Twins: jvOwnKeys in
		// languages/lib/js-rt.metajs, ownKeys in both *-interpreter.abnf.
		if hf, isHost := v.(*hostFunc); isHost && hf.name == "Array" {
			out := []string{}
			for _, k := range ks {
				if k != "from" && k != "of" {
					out = append(out, k)
				}
			}
			return out
		}
		return ks
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
func jsvGopnKeys(v interface{}) []string {
	if jsvIsObjectGlobal(v) {
		return append([]string{}, jsvObjectOwnNames...)
	}
	if o, isObj := v.(*jsObject); isObj {
		switch o.props["__strtag"] {
		case "Math":
			return append(append([]string{}, jsvMathOwnNames...), jsvOwnKeys(v)...)
		case "JSON":
			return append(append([]string{}, jsvJSONOwnNames...), jsvOwnKeys(v)...)
		}
	}
	if !jsvIsFnLike(v) {
		if o, isObj := v.(*jsObject); isObj {
			if d, isView := o.props["__protoof"].(*jsObject); isView {
				return jsvProtoGopn(o, d)
			}
			if isClassDesc(o) {
				return jsvClassOwnNames(o)
			}
			// A String wrapper's indices and `length`, then its own properties.
			if p, isStr := o.props["__bvof"].(string); isStr {
				ks := append(jsvOwnKeys(p), "length")
				own := []string{}
				for _, k := range o.keys {
					if !jsvInternalKey(k) {
						own = append(own, k)
					}
				}
				return append(ks, jsvIndexOrder(own)...)
			}
			// A non-enumerable property is in the key order too, where it was defined.
			if _, hid := o.props["__hidden"]; hid {
				ks := []string{}
				for _, k := range o.keys {
					if !jsvInternalKey(k) {
						ks = append(ks, k)
					}
				}
				return jsvErrGopn(o, jsvIndexOrder(ks))
			}
			return jsvErrGopn(o, jsvOwnKeys(v))
		}
		// An array's or a string's `length` after its indices, before an array's
		// named properties, an arguments object's `callee` after that.
		switch t := v.(type) {
		case string:
			return append(jsvOwnKeys(v), "length")
		case *jsArray:
			ks := jsvOwnKeys(v)
			n := len(ks)
			if !jsvIsArgs(t) {
				n = 0
				for _, e := range t.elems {
					if e != jsHole {
						n++
					}
				}
			}
			out := append(append([]string{}, ks[:n]...), "length")
			if jsvIsArgs(t) {
				out = append(out, "callee")
			}
			// The named properties, the non-enumerable ones too.
			if p := jsvArrNamed(t); p != nil {
				return append(out, jsvGopnKeys(p)...)
			}
			return append(out, ks[n:]...)
		}
		return jsvOwnKeys(v)
	}
	if jsvIsArrayGlobal(v) {
		return append(append([]string{}, jsvArrayOwnNames...), jsvOwnKeys(v)...)
	}
	out := []string{}
	for _, k := range []string{"length", "name"} {
		if jsvFnOwn(v, k) {
			out = append(out, k)
		}
	}
	return append(out, jsvOwnKeys(v)...)
}

// jsvProtoGopn is getOwnPropertyNames of the prototype view of descriptor d:
// `constructor`, then the methods and accessors the class body declared, which are
// d's own keys in source order, then what the program wrote onto the view. A
// FUNCTION written through the view is installed on d as well; protoWrite lists the
// names that installed in __pw, which this skips among d's keys, so a declared name
// written again keeps its place, as a redefined property does. Twins: jsProtoGopn in
// both *-interpreter.abnf, jvProtoGopn in languages/lib/js-rt.metajs.
func jsvProtoGopn(view, d *jsObject) []string {
	out := []string{"constructor"}
	for _, k := range d.keys {
		if jsvProtoDeclared(d, k) {
			out = append(out, k)
		}
	}
	for _, k := range jsvOwnKeys(view) {
		if !jsvInNames(out, k) {
			out = append(out, k)
		}
	}
	return out
}

// jsvFnProps -- `f.p = 1` on a FUNCTION. A function value here is a *jsClosure or
// a *hostFunc and neither carries a property map, while node gives every function
// an ordinary own property map, and `f.cache = {}` is a common idiom. The map lives
// BESIDE the function in this side table, keyed by the pointer, which is the
// identity JavaScript uses. Only the two POINTER kinds are admitted, so nothing
// uncomparable can reach the map key. The table pins its functions - a function
// whose properties are reachable is live anyway - and it is package level rather
// than per-jsrt so that jsvOwnKeys, which has no runtime, can read it; the keys are
// pointers, so two runtimes in one process cannot collide.
// Twin: jvFnProps* in languages/lib/js-rt.metajs. The interpreter halves keep the
// same table under the same name.
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
	// rec is an object js_jsfnmeta was handed in place of a name, as the C floor's
	// name slot holds any handle (php's closure record, js_phclo).
	rec *jsObject
	// src is where the function's source text is, one above its id so that 0 is
	// "none": a js interpreter's span (lib/js-fnsrc.js), or for a compiled js program
	// the function index whose text a wrapper value shows (js_jsfnsrc).
	src int
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

// jsvFnRecSet puts rec in the meta jsvFnMetaSet just recorded on v.
func jsvFnRecSet(v interface{}, rec *jsObject) {
	switch f := v.(type) {
	case *jsClosure:
		f.meta.rec = rec
	case *hostFunc:
		f.meta.rec = rec
	}
}

// jsvFnMetaSet records the pair on a function value; any other value is ignored. The
// source a value already carries is kept.
func jsvFnMetaSet(v interface{}, name string, length int) {
	src := jsvFnMeta(v).src
	switch f := v.(type) {
	case *jsClosure:
		f.meta = &jsvFnInfo{name: name, length: length, src: src}
	case *hostFunc:
		f.meta = &jsvFnInfo{name: name, length: length, src: src}
	}
}

// jsvSrcTable is a compiled js program's function source text: the text of its files
// and, per function index, the span of that text its definition covers, in UTF-16
// units ("idx,from,to;" entries behind the highest index and a ';', see emitSrcTable
// in languages/js-to-llvm-ir.abnf). Both are read on the first String(f).
type jsvSrcTable struct {
	text  string
	tab   string
	units []uint16
	spans map[int][2]int
}

// jsvFnText is String(f) of a function: the source text of its definition, or V8's
// native code for one that has none (a builtin, a bound function). Twins: fn_text in
// languages/lib/runtime.c, jsFnText in js-interpreter.abnf.
func (rt *jsrt) jsvFnText(v interface{}) string {
	if src, ok := rt.jsvFnSource(v); ok {
		return src
	}
	return jsvNativeText(jsvFnMeta(v).name)
}

// jsvClassText is String(C) of a class: its source text, kept on its constructor; a
// builtin class is native code under its name. Twins: jvClassText in
// languages/lib/js-rt.metajs, jsClassText in js-interpreter.abnf.
func (rt *jsrt) jsvClassText(c *jsObject) string {
	if src, ok := rt.jsvFnSource(c.props["__ctor"]); ok {
		return src
	}
	return jsvNativeText(rt.toString(c.props["__name"]))
}

func jsvNativeText(name string) string {
	if strings.HasPrefix(name, "bound ") {
		name = ""
	}
	return "function " + name + "() { [native code] }"
}

// jsvFnSource is the source text of f's definition, if the program has one for it.
func (rt *jsrt) jsvFnSource(v interface{}) (string, bool) {
	t := rt.jsSrc
	if t.spans == nil {
		t.units = strUnits(t.text)
		t.spans = map[int][2]int{}
		ents := strings.Split(t.tab, ";")
		for _, e := range ents[1:] {
			f := strings.Split(e, ",")
			if len(f) == 3 {
				idx, _ := strconv.Atoi(f[0])
				from, _ := strconv.Atoi(f[1])
				to, _ := strconv.Atoi(f[2])
				t.spans[idx] = [2]int{from, to}
			}
		}
	}
	if se, ok := t.spans[jsvFnSrcIdx(v)]; ok {
		return strFromUnits(t.units[se[0]:se[1]]), true
	}
	return "", false
}

// jsvFnSrcIdx is the function index whose source text f shows: the one js_jsfnsrc
// linked it to, else a closure's own; -1 for anything else.
func jsvFnSrcIdx(v interface{}) int {
	if s := jsvFnMeta(v).src; s > 0 {
		return s - 1
	}
	if c, ok := v.(*jsClosure); ok {
		if n := c.fn.Name(); strings.HasPrefix(n, "jsf_") {
			if idx, err := strconv.Atoi(n[4:]); err == nil {
				return idx
			}
		}
	}
	return -1
}

// jsvFnSrcSet records where a function value's source text is (jsvFnInfo.src).
func jsvFnSrcSet(v interface{}, src int) {
	mi := jsvFnMeta(v)
	mi.src = src
	switch f := v.(type) {
	case *jsClosure:
		f.meta = &mi
	case *hostFunc:
		f.meta = &mi
	}
}

// jsvNamedHost is a builtin global's host function carrying node's own `name` and
// `length` (Number/1, RegExp/2), recorded as the program's global of that name.
func (rt *jsrt) jsvNamedHost(name string, length int, fn func(rt *jsrt, this uint64, args []interface{}) interface{}) *hostFunc {
	f := jsHostFunc(name, fn)
	jsvFnMetaSet(f, name, length)
	rt.jsvGlobalSet(name, f)
	return f
}

// jsvGlobalSet records f as the program's builtin global of that name (jsGlobals).
func (rt *jsrt) jsvGlobalSet(name string, f interface{}) {
	if rt.jsGlobals == nil {
		rt.jsGlobals = map[string]interface{}{}
	}
	rt.jsGlobals[name] = f
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
	jsvFnSrcSet(shim, jsvFnSrcIdx(fn)+1)
	jsvMethodVals[fn] = shim
	return shim
}

// An instance of a class extending Array is an ARRAY whose kind word carries 1 + the
// index of its class in jsvArrClasses above the low five bits. Twins: jsArrClasses
// and its neighbours in both *-interpreter.abnf (which state the rule), jvArrClasses
// in languages/lib/js-rt.metajs.
var jsvArrClasses []*jsObject

// The builtins that make their array through ArraySpeciesCreate (jsvArrBuiltin).
var jsvArrSpecies = map[string]bool{"concat": true, "filter": true, "flat": true, "flatMap": true, "map": true, "slice": true, "splice": true}

func jsvArrTag(a *jsArray, cls *jsObject) {
	at := -1
	for i, c := range jsvArrClasses {
		if c == cls {
			at = i
			break
		}
	}
	if at < 0 {
		at = len(jsvArrClasses)
		jsvArrClasses = append(jsvArrClasses, cls)
	}
	a.kind = a.kind&31 + 32*(at+1)
}

func jsvArrClass(v interface{}) *jsObject {
	if len(jsvArrClasses) == 0 {
		return nil
	}
	a, ok := v.(*jsArray)
	if !ok || a.kind < 32 || a.kind&31 == jsArgsKind {
		return nil
	}
	return jsvArrClasses[a.kind>>5-1]
}

// A method call on an instance of a class extending a builtin: the class chain's own
// method, or false for the builtin's methods to answer.
func (rt *jsrt) jsvClassCall(a interface{}, cls *jsObject, name string, args []interface{}) (interface{}, bool) {
	for c, guard := cls, 0; c != nil && guard < 64; guard++ {
		if m, has := c.props[name]; has && isCallable(m) && !jsvInternalKey(name) {
			return rt.call(m, jsUndef, append([]interface{}{a}, args...)), true
		}
		if v, hasView := c.props["__pview"].(*jsObject); hasView {
			if pv, has := v.props[name]; has && isCallable(pv) {
				return rt.call(pv, a, args), true
			}
		}
		c, _ = c.props["__super"].(*jsObject)
	}
	return nil, false
}

// An Array builtin on an instance, which is also what its super.m() reaches. One
// that makes its array through ArraySpeciesCreate (ECMA-262 10.4.2.3) makes it with
// the instance's species constructor (jsSpeciesOf): constructed with the length the builtin
// asks for - map the source's, slice and splice the count, the others 0 - before
// the builtin runs, or after it for slice and splice, whose count is the length of
// the builtin's own answer; then filled with that answer's elements, and concat,
// slice and splice set its length. Twins: jvArrBuiltin in
// languages/lib/js-rt.metajs, jsArrBuiltin in both *-interpreter.abnf.
func (rt *jsrt) jsvArrBuiltin(a *jsArray, name string, args []interface{}) (interface{}, bool) {
	if lv := jsvArrLevel(a); jsvAttrN > 0 && lv > 0 {
		if msg := rt.jsvArrMethodRefusal(a, name, args, lv); msg != "" {
			rt.bigRaise("TypeError: %s", msg)
		}
	}
	var sc *jsObject
	if cls := jsvArrClass(a); cls != nil && jsvArrSpecies[name] {
		sc = rt.jsSpeciesOf(cls)
	}
	if sc == nil {
		return rt.jsvMethod(a, name, args)
	}
	pre := name != "slice" && name != "splice"
	var out interface{}
	if pre {
		n := 0
		if name == "map" {
			n = len(a.elems)
		}
		out = rt.jsvConstruct(sc, []interface{}{float64(n)})
	}
	v, handled := rt.jsvMethod(a, name, args)
	r, isArr := v.(*jsArray)
	if !handled || !isArr {
		return v, handled
	}
	if !pre {
		out = rt.jsvConstruct(sc, []interface{}{float64(len(r.elems))})
	}
	for i, e := range r.elems {
		rt.setMember(out, float64(i), e)
	}
	if !pre || name == "concat" {
		rt.setMember(out, "length", float64(len(r.elems)))
	}
	return out, true
}

// `new C(args...)` for a class descriptor C: the instance carries C, the
// constructor runs with new.target C, and an object it returns wins. Twin of
// makeNewFrom's class arm in js-to-llvm-ir.abnf.
func (rt *jsrt) jsvConstruct(cls *jsObject, args []interface{}) interface{} {
	o := newJSObject()
	o.set("__class", cls)
	rt.pendingNewTarget = cls
	r := rt.call(cls.props["__ctor"], jsUndef, append([]interface{}{o}, args...))
	rt.pendingNewTarget = nil
	if jsvIsObjLike(r) || isCallable(r) {
		return r
	}
	return o
}

// A read off an instance: `constructor`, a getter or a method value of its class chain.
func (rt *jsrt) jsvArrSubRead(a *jsArray, cls *jsObject, name string) (interface{}, bool) {
	if jsvInternalKey(name) {
		return nil, false
	}
	if name == "constructor" {
		return cls, true
	}
	for c, guard := cls, 0; c != nil && guard < 64; guard++ {
		if m, has := c.props[name]; has {
			if acc, isAcc := m.(*jsAccessor); isAcc {
				if acc.get == nil {
					return jsUndef, true
				}
				return rt.call(acc.get, a, nil), true
			}
			if isCallable(m) {
				return jsvMethodValue(name, m), true
			}
		}
		if v, hasView := c.props["__pview"].(*jsObject); hasView {
			if pv, has := v.props[name]; has {
				return pv, true
			}
		}
		c, _ = c.props["__super"].(*jsObject)
	}
	return nil, false
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
	if r := jsvFnIntrRec(v, name); r != nil {
		if _, gone := r.props["del"]; gone {
			if name == "name" {
				return ""
			}
			return float64(0)
		}
		return r.props["v"]
	}
	mi := jsvFnMeta(v)
	if name == "name" {
		return mi.name
	}
	return float64(mi.length)
}

// jsvFnIntrRec is the change a definition or a delete made to a function's `name` or
// `length`, or nil while it is the intrinsic one. Both are configurable (ES2023
// 10.2.9, 10.2.10): Object.defineProperty replaces either - non-writable unless the
// descriptor says writable - and `delete` removes it, after which a read finds
// Function.prototype's ("" and 0). The record lives in the side table under "__fn" +
// key: {v, w} for a defined property, {del} for a deleted one. Twins: fnIntrRec in
// both *-interpreter.abnf, jvFnIntrRec in languages/lib/js-rt.metajs.
func jsvFnIntrRec(f interface{}, k string) *jsObject {
	p := jsvFnPropsOf(f)
	if p == nil {
		return nil
	}
	r, _ := p.props["__fn"+k].(*jsObject)
	return r
}

// jsvFnIntrDefine is Object.defineProperty of a function's `name` or `length` with a
// data descriptor.
func (rt *jsrt) jsvFnIntrDefine(f interface{}, k string, desc interface{}) {
	cur := jsvFnIntrRec(f, k)
	gone := false
	if cur != nil {
		_, gone = cur.props["del"]
	}
	r := newJSObject()
	var v interface{} = jsUndef
	if !gone {
		v = jsvFnIntrinsic(f, k)
	}
	wr := cur != nil && !gone && cur.props["w"] == true
	if d, ok := desc.(*jsObject); ok {
		if _, has := d.props["value"]; has {
			v = rt.jsvGetKey(desc, "value")
		}
		if _, has := d.props["writable"]; has {
			wr = rt.truthy(rt.jsvGetKey(desc, "writable"))
		}
	}
	r.set("v", v)
	r.set("w", wr)
	jsvFnPropsMake(f).set("__fn"+k, r)
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

// jsvFnMethod -- the Object.prototype SURFACE of a FUNCTION, as methods:
// `f.toString()`, `f.valueOf()`, `f.hasOwnProperty(k)` and
// `f.propertyIsEnumerable(k)`. Twins: jvFnMethod in languages/lib/js-rt.metajs and
// fnHasOwnShim / fnToStringShim / fnValueOfShim in both *-interpreter.abnf. Answers
// nil for any receiver or name it does not claim, so the caller tests one value
// instead of four names.
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
		return jsvNativeFn(jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			k := rt.toString(argAt(args, 0))
			if enumerable {
				return jsvPropEnum(recv, k)
			}
			return jsvFnOwn(recv, k)
		}), name, 0)
	case "toString":
		return jsvNativeFn(jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return rt.jsvString(recv)
		}), name, 0)
	case "valueOf":
		return jsvNativeFn(jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return recv
		}), name, 0)
	}
	return nil
}

// jsvNativeFn gives f a builtin method's own `name` and `length`. Twin of jsNativeFn
// in js-interpreter.abnf, which states the rule; jvNativeFn in
// languages/lib/js-rt.metajs.
func jsvNativeFn(f *hostFunc, k string, fam byte) *hostFunc {
	n := 1
	switch k {
	case "slice", "substring", "substr", "split", "replace", "replaceAll", "splice", "toSpliced", "with",
		"copyWithin", "apply":
		n = 2
	case "toUpperCase", "toLowerCase", "trim", "trimStart", "trimEnd", "normalize", "pop", "shift", "reverse",
		"flat", "keys", "values", "entries", "toReversed", "toLocaleString", "valueOf":
		n = 0
	case "toString":
		if fam != 'n' {
			n = 0
		}
	}
	jsvFnMetaSet(f, k, n)
	return f
}

func jsvInternalKey(k string) bool {
	return len(k) >= 2 && k[0] == '_' && k[1] == '_'
}

// ----- which static members of a class are ENUMERABLE --------------------------
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

// jsvFnOwn answers whether k is an own property of a function: `name` and `length`
// are own properties of every function, the rest live in the side table.
func jsvFnOwn(f interface{}, k string) bool {
	if k == "name" || k == "length" {
		if r := jsvFnIntrRec(f, k); r != nil {
			_, gone := r.props["del"]
			return !gone
		}
		return true
	}
	if jsvIsArrayGlobal(f) && jsvInNames(jsvArrayOwnNames, k) {
		return true
	}
	if jsvIsObjectGlobal(f) && jsvInNames(jsvObjectOwnNames, k) {
		return true
	}
	p := jsvFnPropsOf(f)
	if p == nil {
		return false
	}
	_, found := p.props[k]
	return found
}

// jsvHasOwn is Object.prototype.hasOwnProperty(k) for any receiver: a function's
// own properties, an array's or a string's `length` and indices (canonical index
// strings only, so "01" is not one), an Error's own `stack` / `message`, and an
// object's own slots - the KEY MAP, so a slot holding undefined is still one. A class
// instance's methods live on its class and are not own. Twins: jvHasOwn in
// languages/lib/js-rt.metajs, jsHasOwn in both *-interpreter.abnf.
func (rt *jsrt) jsvHasOwn(v interface{}, k string) bool {
	if jsvIsFnLike(v) {
		return jsvFnOwn(v, k)
	}
	n := -1
	var arr *jsArray
	switch o := v.(type) {
	case *jsObject:
		if own, ok := jsvErrOwn(o, k); ok {
			return own
		}
		if isClassDesc(o) {
			return jsvInNames(jsvClassOwnNames(o), k)
		}
		// A String wrapper owns its string's indices and length.
		if p, isStr := o.props["__bvof"].(string); isStr && rt.jsvHasOwn(p, k) {
			return true
		}
		if d, isView := o.props["__protoof"].(*jsObject); isView && (k == "constructor" || jsvProtoDeclared(d, k)) {
			return true
		}
		_, has := o.props[k]
		return has
	case *jsArray:
		if jsvIsArgs(o) && k != "length" {
			return jsvArgsHas(o, k)
		}
		if p := jsvArrNamed(o); p != nil {
			if _, has := p.props[k]; has {
				return true
			}
		}
		n = len(o.elems)
		arr = o
	case string:
		n = rt.strLen(o)
	default:
		return false
	}
	if k == "length" {
		return true
	}
	i, err := strconv.Atoi(k)
	return err == nil && strconv.Itoa(i) == k && i >= 0 && i < n && (arr == nil || arr.elems[i] != jsHole)
}

// jsvOwnDesc is Object.getOwnPropertyDescriptor(v, k) for a key in the object's own
// spelling, nil for one v does not own; get is the VALUE read of a member (js_jsvget).
// Twin of jsOwnDesc in js-interpreter.abnf, which states the rule; jvOwnDesc in
// languages/lib/js-rt.metajs.
func (rt *jsrt) jsvOwnDesc(v interface{}, k string, get func(interface{}, string) interface{}) *jsObject {
	if isUndefOrNull(v) {
		rt.bigRaise("TypeError: Cannot convert undefined or null to object")
	}
	if !rt.jsvHasOwn(v, k) {
		return nil
	}
	en := jsvPropEnum(v, k)
	wr, cf := true, true
	var at *jsObject    // the object whose attribute sets hold k
	var slots *jsObject // the map that may hold k's accessor
	switch o := v.(type) {
	case string:
		wr, cf = false, false
	case *jsArray:
		if !jsvIsArgs(o) {
			lv := jsvArrLevel(o)
			wr, cf = lv < 3, k != "length" && lv < 2
		}
		// A named property's own attributes narrow what the array's level allows.
		if p := jsvArrNamed(o); p != nil {
			if _, named := p.props[k]; named {
				wr = wr && !jsvAttrHas(p, "__ro", k)
				cf = cf && !jsvAttrHas(p, "__nc", k)
				slots = p
			}
		}
	case *jsObject:
		if p, isStr := o.props["__bvof"].(string); isStr && rt.jsvHasOwn(p, k) {
			wr, cf = false, false
			break
		}
		if isClassDesc(o) {
			switch {
			case k == "prototype":
				wr, cf = false, false
			case (k == "name" || k == "length") && !jsvOwnsStatic(o, k):
				wr = false
			default:
				slots = jsStaticsOf(o, false)
			}
			break
		}
		at, slots = o, o
		if d, isView := o.props["__protoof"].(*jsObject); isView {
			if _, own := o.props[k]; !own {
				slots = d
			}
		}
	default:
		if !jsvIsFnLike(v) {
			break
		}
		switch k {
		case "name", "length":
			r := jsvFnIntrRec(v, k)
			wr, cf = r != nil && r.props["w"] == true, jsvIntegrity(v) < 2
		case "prototype":
			wr, cf = false, false
		default:
			at = jsvFnPropsOf(v)
		}
	}
	if strings.HasPrefix(k, "__#@") {
		en = at == nil || !jsvAttrHas(at, "__hidden", k)
	}
	if at != nil {
		wr, cf = !jsvAttrHas(at, "__ro", k), !jsvAttrHas(at, "__nc", k)
	}
	d := newJSObject()
	var acc *jsAccessor
	if slots != nil {
		acc, _ = slots.props[k].(*jsAccessor)
	}
	if acc != nil {
		d.set("get", jsvOrUndef(acc.get))
		d.set("set", jsvOrUndef(acc.set))
	} else {
		d.set("value", get(v, k))
		d.set("writable", wr)
	}
	d.set("enumerable", en)
	d.set("configurable", cf)
	return d
}

// jsvOrUndef is v, or undefined for an absent (nil) half of an accessor.
func jsvOrUndef(v interface{}) interface{} {
	if v == nil {
		return jsUndef
	}
	return v
}

// jsvOwnDescs is Object.getOwnPropertyDescriptor and Object.getOwnPropertyDescriptors.
func (rt *jsrt) jsvOwnDescs(name string, args []interface{}, get func(interface{}, string) interface{}) interface{} {
	src := argAt(args, 0)
	if name == "getOwnPropertyDescriptor" {
		if d := rt.jsvOwnDesc(src, rt.jsvPropKey(argAt(args, 1)), get); d != nil {
			return d
		}
		return jsUndef
	}
	if isUndefOrNull(src) {
		rt.bigRaise("TypeError: Cannot convert undefined or null to object")
	}
	out := newJSObject()
	for _, k := range append(jsvGopnKeys(src), jsvSymKeys(src)...) {
		out.set(k, rt.jsvOwnDesc(src, k, get))
	}
	return out
}

// jsvClassOwnNames are the own property names of a class object: `length`, `name`
// and `prototype`, then its static members and what the program wrote onto it, in
// the order they were defined, which is the static table's own key order. Twins:
// jsClassOwnNames in both *-interpreter.abnf, jvClassOwnNames in
// languages/lib/js-rt.metajs.
func jsvClassOwnNames(c *jsObject) []string {
	out := []string{}
	for _, k := range []string{"length", "name"} {
		if c.props["__del"+k] != true {
			out = append(out, k)
		}
	}
	out = append(out, "prototype")
	if t := jsStaticsOf(c, false); t != nil {
		for _, k := range t.keys {
			if !jsvInternalKey(k) && !jsvInNames(out, k) {
				out = append(out, k)
			}
		}
	}
	return jsvIndexOrder(out)
}

// jsvProtoDeclared reports whether k is a method or accessor the class body of
// descriptor d declared: an own key of d that protoWrite did not install (__pw).
// Those are own properties of the prototype that are not enumerable, and a write
// through the prototype keeps them so. Twins: jsProtoDeclared in both
// *-interpreter.abnf, jvProtoDeclared in languages/lib/js-rt.metajs.
func jsvProtoDeclared(d *jsObject, k string) bool {
	if _, has := d.props[k]; !has || jsvInternalKey(k) {
		return false
	}
	if pw, ok := d.props["__pw"].(*jsArray); ok {
		for _, e := range pw.elems {
			if e == k {
				return false
			}
		}
	}
	return true
}

// jsvInheritedKeys appends to the own key list ks the enumerable keys of the
// prototype chain from descriptor cls up: each level's prototype view's keys
// (jsvOwnKeys). A name the object or a lower level already has - its declared methods
// and accessors included, which are not enumerable - shadows it, so it is visited once
// at most. Twins: jvInheritedKeys in languages/lib/js-rt.metajs, jsInheritedKeys in
// both *-interpreter.abnf.
func jsvInheritedKeys(cls interface{}, ks []interface{}) []interface{} {
	seen := map[string]bool{}
	for _, k := range ks {
		if s, isStr := k.(string); isStr {
			seen[s] = true
		}
	}
	for guard := 0; cls != nil && guard < 64; guard++ {
		d, ok := cls.(*jsObject)
		if !ok {
			break
		}
		var vk []string
		if v, hasView := d.props["__pview"].(*jsObject); hasView {
			vk = jsvOwnKeys(v)
			for _, k := range vk {
				if !seen[k] {
					ks = append(ks, k)
				}
			}
		}
		for _, k := range append(vk, d.keys...) {
			seen[k] = true
		}
		cls = d.props["__super"]
	}
	return ks
}

// jsvStaticForInKeys is for-in over a CLASS OBJECT: its own enumerable statics ks,
// then each superclass's, nearest first. A name a lower class owns - enumerable or
// not, its length, name and prototype included - shadows the inherited one. Twins:
// jsStaticForInKeys in both *-interpreter.abnf, jvStaticForInKeys in
// languages/lib/js-rt.metajs.
func jsvStaticForInKeys(c *jsObject, ks []interface{}) []interface{} {
	seen := map[string]bool{}
	for _, k := range jsvClassOwnNames(c) {
		seen[k] = true
	}
	cl, _ := c.props["__super"].(*jsObject)
	for guard := 0; cl != nil && isClassDesc(cl) && guard < 64; guard++ {
		lvl := jsvStaticEnumKeys(cl)
		if fn, has := cl.props["__ctorfn"]; has {
			lvl = jsvOwnKeys(fn)
		}
		for _, k := range lvl {
			if !seen[k] {
				ks = append(ks, k)
			}
		}
		for _, k := range jsvClassOwnNames(cl) {
			seen[k] = true
		}
		cl, _ = cl.props["__super"].(*jsObject)
	}
	return ks
}

// jsvProtoChainHas reports whether k is a property of the prototype chain from
// descriptor cls up: a method or accessor a class body declared, or a key written
// onto a prototype view. Twins: jvProtoChainHas in languages/lib/js-rt.metajs,
// jsProtoChainHas in both *-interpreter.abnf.
func jsvProtoChainHas(cls interface{}, k string) bool {
	if jsvInternalKey(k) {
		return false
	}
	for guard := 0; cls != nil && guard < 64; guard++ {
		d, ok := cls.(*jsObject)
		if !ok {
			break
		}
		if _, has := d.props[k]; has {
			return true
		}
		if v, hasView := d.props["__pview"].(*jsObject); hasView {
			if _, has := v.props[k]; has {
				return true
			}
		}
		cls = d.props["__super"]
	}
	return false
}

// jsvIndexKeys is the own enumerable key list of an indexed value - an array or a
// string - which node answers as the index STRINGS "0".."n-1". Twin: jvIndexKeys in
// languages/lib/js-rt.metajs.
// jsvIndexOrder is OrdinaryOwnPropertyKeys' order (ECMA-262 10.1.11.1): the
// array-index keys ascending, then the other string keys in insertion order. ks is
// returned as it is when it holds no array index. Twins: jvIndexOrder in
// languages/lib/js-rt.metajs, jsIndexOrder in both *-interpreter.abnf.
func jsvIndexOrder(ks []string) []string {
	var idx []uint64
	for _, k := range ks {
		if n, ok := jsvArrayIndex(k); ok {
			idx = append(idx, n)
		}
	}
	if idx == nil {
		return ks
	}
	sort.Slice(idx, func(i, j int) bool { return idx[i] < idx[j] })
	out := make([]string, 0, len(ks))
	for _, n := range idx {
		out = append(out, strconv.FormatUint(n, 10))
	}
	for _, k := range ks {
		if _, ok := jsvArrayIndex(k); !ok {
			out = append(out, k)
		}
	}
	return out
}

// jsvArrayIndex answers whether k is an array index: the canonical decimal spelling
// of an integer in 0 .. 2^32-2.
func jsvArrayIndex(k string) (uint64, bool) {
	if len(k) == 0 || len(k) > 10 || (k[0] == '0' && len(k) > 1) {
		return 0, false
	}
	var n uint64
	for i := 0; i < len(k); i++ {
		if k[i] < '0' || k[i] > '9' {
			return 0, false
		}
		n = n*10 + uint64(k[i]-'0')
	}
	return n, n <= 4294967294
}

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
	return jsvIndexOrder(out)
}

// jsvConstructorOf is what `x.constructor` answers for an object this engine built:
// the class descriptor the instance carries, or - for an instance of a constructor
// FUNCTION - the function itself, which js_jsnewclass's descriptor keeps under
// __ctorfn. nil for a value with no __class at all (a plain object literal, which
// jsvBuiltinCtor answers with Object). Twins: jvConstructorOf in languages/lib/js-rt.metajs and classCtorOf in the
// two *-interpreter.abnf start scripts.
func jsvConstructorOf(o *jsObject) (interface{}, bool) {
	cls, ok := o.props["__class"].(*jsObject)
	// The descriptor Object.create files for a plain prototype object is no class, so
	// the nearest class above it answers.
	for ok && !isClassDesc(cls) {
		if v, has := cls.props["__pview"].(*jsObject); has {
			if c, own := v.props["constructor"]; own {
				return c, true
			}
		}
		cls, ok = cls.props["__super"].(*jsObject)
	}
	if !ok {
		return nil, false
	}
	if fn, has := cls.props["__ctorfn"]; has {
		return fn, true
	}
	return cls, true
}

// jsvBuiltinCtor is `x.constructor` where no class answers it: the program's builtin
// global whose prototype x inherits - Number for 7, Array for [], Object for a plain
// object - and nil where the ordinary read decides. Twin of jsBuiltinCtor in
// js-interpreter.abnf, which states the rule; jvBuiltinCtor in
// languages/lib/js-rt.metajs.
func (rt *jsrt) jsvBuiltinCtor(v interface{}) interface{} {
	name := "Object"
	switch x := v.(type) {
	case float64:
		name = "Number"
	case string:
		name = "String"
	case bool:
		name = "Boolean"
	case *jsBigInt:
		name = "BigInt"
	case *jsArray:
		if !jsvIsArgs(x) {
			name = "Array"
		}
	case *jsObject:
		switch {
		case isClassDesc(x) || jsvNoObjProto(x):
			return nil
		case jsvIsSym(x):
			name = "Symbol"
		case x.props["__class"] != nil:
			if c, found := jsvConstructorOf(x); found {
				return c
			}
		case promIs(x):
			name = "Promise"
		case rxIsRegexObj(x):
			name = "RegExp"
		}
	default:
		if !isCallable(v) {
			return nil
		}
		name = "Function"
	}
	return rt.jsGlobals[name]
}

func (rt *jsrt) jsvGetKey(v interface{}, k string) interface{} {
	// A CLASS DESCRIPTOR reads from its static namespace, exactly as js_jsvget does:
	// rt.getMember would answer the descriptor's own slot, which is where the
	// INSTANCE methods live, so Object.values(C) / Object.entries(C) / Object.assign
	// would see undefined for every static field. (The native leg goes through
	// js_jsmget.)
	if cls, isStatic := jsvIsStaticSide(v, k); isStatic {
		return rt.unwrap(rt.jsvStaticSideRead(cls, k))
	}
	if val, isAcc := rt.jsvArrAccRead(v, k); isAcc {
		return val
	}
	// A FUNCTION's own properties live in the side table, and rt.getMember knows
	// nothing about it - the SAME shape as the class-descriptor arm above, one layer
	// out: without it Object.values(f), Object.entries(f) and Object.assign({}, f)
	// would read undefined values on the llvm.Run leg (the native leg goes through
	// layer 2's js_jsmget).
	if p := jsvFnPropsOf(v); p != nil {
		if val, found := p.props[k]; found {
			return val
		}
	}
	// A String wrapper's indices are its string's.
	if p, isWrap := jsvWrapped(v); isWrap {
		if ps, isStr := p.(string); isStr {
			if _, isIdx := jsvArrayIndex(k); isIdx {
				return rt.getMember(ps, k)
			}
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
	"normalize": true,
}

var jsvArrayMethods = map[string]bool{
	"push": true, "pop": true, "shift": true, "unshift": true, "slice": true,
	"splice": true, "indexOf": true, "lastIndexOf": true, "includes": true,
	"join": true, "concat": true, "reverse": true, "sort": true, "map": true,
	"filter": true, "forEach": true, "reduce": true, "reduceRight": true,
	"some": true, "every": true, "find": true, "findIndex": true,
	"findLast": true, "findLastIndex": true, "flat": true, "flatMap": true,
	"fill": true, "at": true, "toString": true, "keys": true, "values": true, "entries": true,
	"toReversed": true, "toSorted": true, "toSpliced": true, "with": true, "copyWithin": true,
}

// jsvNumberMethods are a Number's methods read as a value (jsvBuiltinFam).
var jsvNumberMethods = map[string]bool{"toFixed": true, "toExponential": true, "toPrecision": true, "toString": true, "valueOf": true, "toLocaleString": true}

// jsvBuiltinFam is 's' / 'a' / 'n' / 'b' when name is a builtin method of a String,
// an Array, a Number or a Boolean receiver READ AS A VALUE (js_jsvget), else 0: the
// names js_jsmcall answers, plus an Array's valueOf and toLocaleString. Twins: builtinFam in both
// *-interpreter.abnf, jvBuiltinFam in languages/lib/js-rt.metajs.
func jsvBuiltinFam(target interface{}, name string) byte {
	switch target.(type) {
	case string:
		if jsvStringMethods[name] {
			return 's'
		}
	case float64:
		if jsvNumberMethods[name] {
			return 'n'
		}
	case bool:
		if name == "toString" || name == "valueOf" {
			return 'b'
		}
	case *jsArray:
		if (jsvArrayMethods[name] || name == "valueOf" || name == "toLocaleString") && !jsvIsArgs(target) {
			return 'a'
		}
	}
	return 0
}

// jsvBuiltinRecv is the receiver a builtin method value runs on, with node's
// TypeErrors: a Number or a Boolean method and String's toString/valueOf demand their
// own kind, a primitive or its wrapper, every other method refuses only null and
// undefined, and a String method coerces what it is given. Twins: builtinRecv in both
// *-interpreter.abnf, jvBuiltinRecv in languages/lib/js-rt.metajs.
func (rt *jsrt) jsvBuiltinRecv(fam byte, name string, recv interface{}) interface{} {
	raise := func(msg string) { panic(&jsThrown{value: rt.jsErrFromText("TypeError: " + msg)}) }
	if p, isWrap := jsvWrapped(recv); isWrap && (fam == 'n' || fam == 'b' || (fam == 's' && (name == "toString" || name == "valueOf"))) {
		recv = p
	}
	if fam == 'n' {
		if _, isNum := recv.(float64); !isNum {
			raise("Number.prototype." + name + " requires that 'this' be a Number")
		}
		return recv
	}
	if fam == 'b' {
		if _, isBool := recv.(bool); !isBool {
			raise("Boolean.prototype." + name + " requires that 'this' be a Boolean")
		}
		return recv
	}
	if isUndefOrNull(recv) {
		if fam == 's' {
			switch name {
			case "trimStart":
				name = "trimLeft"
			case "trimEnd":
				name = "trimRight"
			}
			raise("String.prototype." + name + " called on null or undefined")
		}
		switch name {
		case "indexOf", "concat", "map", "filter", "forEach", "reduce", "reduceRight", "some", "every",
			"find", "findIndex", "findLast", "findLastIndex":
			raise("Array.prototype." + name + " called on null or undefined")
		}
		raise("Cannot convert undefined or null to object")
	}
	if _, isStr := recv.(string); fam == 's' && !isStr {
		if name == "toString" || name == "valueOf" {
			raise("String.prototype." + name + " requires that 'this' be a String")
		}
		return rt.jsvString(recv)
	}
	// An Array method called on an arguments object reads it up to its `length`; the
	// ones that write keep the object itself.
	if a, isArr := recv.(*jsArray); isArr && fam == 'a' && jsvIsArgs(a) {
		switch name {
		case "push", "pop", "shift", "unshift", "splice", "reverse", "sort", "fill", "copyWithin":
		default:
			return jsvArgsView(a)
		}
	}
	return recv
}

func jsvHidesMember(target interface{}, name string) bool {
	switch target.(type) {
	case string:
		return jsvStringMethods[name]
	case *jsArray:
		return jsvArrayMethods[name] && !jsvIsArgs(target)
	}
	return false
}

// A program's own property key that starts with "__" would be one of the value
// model's hidden slots, so it is stored behind a leading U+0001, as is one that
// itself starts with U+0001. That includes a string of a symbol key's "__#@" shape:
// a symbol reaches the slot as its key through jsvPropKey / js_jspropkey, never
// through here. A private name's "__#" key is escaped like any other here and
// reaches js_jsprivin and the member externs RAW.
// The emitters escape a member name at compile time and pass a
// computed key through js_jskeyesc (or js_jspropkey), so the member externs see
// stored keys; js_jshas and hasOwnProperty escape the key a program hands them,
// and js_jskeys, Object.keys / entries / getOwnPropertyNames and JSON unescape what
// they hand back. Twins: jsKeyEsc in both *-interpreter.abnf, js_jskeyesc in
// languages/lib/js-rt.metajs.
const jsvKeyMark = "\x01"

func jsvKeyNeedsEsc(k string) bool {
	if len(k) == 0 {
		return false
	}
	return k[0] == 1 || (len(k) >= 2 && k[0] == '_' && k[1] == '_')
}
func jsvKeyEsc(k string) string {
	if jsvKeyNeedsEsc(k) {
		return jsvKeyMark + k
	}
	return k
}
func jsvKeyUnesc(k string) string {
	if len(k) > 0 && k[0] == 1 {
		return k[1:]
	}
	return k
}

// jsvEscArg is the argument slice with a[i], a program's property key, escaped; the
// slice itself when the key needs nothing.
func (rt *jsrt) jsvEscArg(a []uint64, i int) []uint64 {
	if k, ok := rt.unwrap(a[i]).(string); ok && jsvKeyNeedsEsc(k) {
		b := append([]uint64(nil), a...)
		b[i] = rt.wrapStr(jsvKeyMark + k)
		return b
	}
	return a
}

// jsvIsArgs: the arguments object, an array whose kind word carries jsArgsKind in
// its low five bits (js_jsargs below). Above them the word keeps the value of the
// object's own `length`: the call's argument count, which an element write does not
// change (ECMA-262 10.4.4.6, CreateUnmappedArgumentsObject step 4) and only a write
// to `length` itself does. Twins: jsIsArgs / jsArgsLen in both *-interpreter.abnf,
// jvIsArgs / jvArgsLen in languages/lib/js-rt.metajs.
const jsArgsKind = 19

func jsvIsArgs(v interface{}) bool {
	a, ok := v.(*jsArray)
	return ok && a.kind&31 == jsArgsKind
}

func jsvArgsKind(length int) int { return jsArgsKind + 32*length }

// jsvArgsHas / jsvArgsView / jsvArgsKeys: an index of an arguments object is an own
// property when it is below `length` or holds a value, and what an iteration, a
// spread, apply and an Array method read is the list up to `length`. Twins:
// jsArgsHas / jsArgsView in both *-interpreter.abnf, jvArgsHas / jvArgsView in
// languages/lib/js-rt.metajs.
func jsvArgsHas(a *jsArray, k string) bool {
	i, isIdx := jsvArrayIndex(k)
	if !isIdx || i >= uint64(len(a.elems)) {
		return false
	}
	_, undef := a.elems[i].(jsUndefT)
	return a.elems[i] != jsHole && (int(i) < a.kind>>5 || !(undef || a.elems[i] == nil))
}

func jsvArgsView(a *jsArray) *jsArray {
	out := &jsArray{elems: make([]interface{}, a.kind>>5)}
	for i := range out.elems {
		out.elems[i] = jsUndef
		if i < len(a.elems) {
			out.elems[i] = a.elems[i]
		}
	}
	return out
}

// jsvArgsIterFn is an arguments object's [Symbol.iterator] (Array.prototype.values),
// which the iteration sites probe under the key it stands for; nil for anything else.
func jsvArgsIterFn(v interface{}, name string) *hostFunc {
	arr, isArr := v.(*jsArray)
	if !isArr || !jsvIsArgs(arr) || name != "__#@iterator" {
		return nil
	}
	return jsHostFunc("values", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return jsvSeqIter(arr, 1)
	})
}

func jsvArgsKeys(a *jsArray) []string {
	out := []string{}
	for i := range a.elems {
		if k := strconv.Itoa(i); jsvArgsHas(a, k) {
			out = append(out, k)
		}
	}
	return out
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
	return rt.jsvToNum(args[i])
}

// Number#toFixed / toExponential / toPrecision (ES2023 21.1.3). Each rounds the
// EXACT binary value half UP (of two equally close candidates the spec takes the
// larger). The digit count has been through ToIntegerOrInfinity; a RangeError is
// answered as its message. Twins: jsnToFixed / jsnToExponential / jsnToPrecision in
// languages/lib/js-num.js.
func jsToFixedJS(x, f float64) (string, string) {
	if math.IsInf(f, 0) || f < 0 || f > 100 {
		return "", "RangeError: toFixed() digits argument must be between 0 and 100"
	}
	if math.IsNaN(x) || math.Abs(x) >= 1e21 {
		return jsNumString(x), ""
	}
	sign, a := "", x
	if x < 0 {
		sign, a = "-", -x
	}
	n := int(f)
	ip, fp := "0", strings.Repeat("0", n)
	if a != 0 {
		i, fr := jsExactDec(a)
		r := jsRoundUpDigits(i+fr, len(i)+n)
		ip, fp = r[:len(r)-n], r[len(r)-n:]
	}
	if n == 0 {
		return sign + ip, ""
	}
	return sign + ip + "." + fp, ""
}

func jsToExponentialJS(x, f float64, undef bool) (string, string) {
	if math.IsNaN(x) || math.IsInf(x, 0) {
		return jsNumString(x), ""
	}
	if math.IsInf(f, 0) || f < 0 || f > 100 {
		return "", "RangeError: toExponential() argument must be between 0 and 100"
	}
	sign, a := "", x
	if x < 0 {
		sign, a = "-", -x
	}
	if a == 0 {
		if undef {
			return sign + "0e+0", ""
		}
		return sign + jsExpForm(strings.Repeat("0", int(f)+1), 0), ""
	}
	var d string
	var e int
	if undef {
		d, e = jsShortestDigits(a)
	} else {
		d, e = jsSigDigits(a, int(f)+1)
	}
	return sign + jsExpForm(d, e), ""
}

func jsToPrecisionJS(x, p float64) (string, string) {
	if math.IsNaN(x) || math.IsInf(x, 0) {
		return jsNumString(x), ""
	}
	if math.IsInf(p, 0) || p < 1 || p > 100 {
		return "", "RangeError: toPrecision() argument must be between 1 and 100"
	}
	sign, a := "", x
	if x < 0 {
		sign, a = "-", -x
	}
	n := int(p)
	d, e := strings.Repeat("0", n), 0
	if a != 0 {
		d, e = jsSigDigits(a, n)
	}
	switch {
	case e < -6 || e >= n:
		return sign + jsExpForm(d, e), ""
	case e == n-1:
		return sign + d, ""
	case e >= 0:
		return sign + d[:e+1] + "." + d[e+1:], ""
	}
	return sign + "0." + strings.Repeat("0", -(e+1)) + d, ""
}

// jsExactDec is the exact decimal of a positive finite double: integer digits ("0"
// when there are none) and fraction digits. 1074 fraction digits hold every double
// exactly, so FormatFloat rounds nothing away; the trailing zeros are dropped.
func jsExactDec(a float64) (string, string) {
	s := strings.TrimRight(strconv.FormatFloat(a, 'f', 1074, 64), "0")
	i, f, _ := strings.Cut(s, ".")
	return i, f
}

// jsRoundUpDigits is the first keep digits of d rounded half up, one digit longer
// when the rounding carried out of the first.
func jsRoundUpDigits(d string, keep int) string {
	head := d
	if keep < len(d) {
		head = d[:keep]
	}
	for len(head) < keep {
		head += "0"
	}
	if keep >= len(d) || d[keep] < '5' {
		return head
	}
	b := []byte(head)
	i := len(b) - 1
	for i >= 0 && b[i] == '9' {
		b[i] = '0'
		i--
	}
	if i < 0 {
		return "1" + string(b)
	}
	b[i]++
	return string(b)
}

// jsSigDigits is k significant digits of a, rounded half up, and the decimal
// exponent of the first.
func jsSigDigits(a float64, k int) (string, int) {
	i, f := jsExactDec(a)
	all := i + f
	z := 0
	for all[z] == '0' {
		z++
	}
	e := len(i) - z - 1
	d := jsRoundUpDigits(all[z:], k)
	if len(d) > k {
		d, e = d[:k], e+1
	}
	return d, e
}

// jsShortestDigits is the shortest digit string that reads back as a, and the
// exponent of its first digit.
func jsShortestDigits(a float64) (string, int) {
	s := strconv.FormatFloat(a, 'e', -1, 64)
	m, ex, _ := strings.Cut(s, "e")
	e, _ := strconv.Atoi(ex)
	return strings.Replace(m, ".", "", 1), e
}

// jsToLocaleJS is Number#toLocaleString() in node's default locale (en-US): the
// shortest digits rounded half up to at most three fraction digits, trailing zeros
// dropped, the integer part grouped; a negative value keeps its sign through a
// rounding to zero. Twin: jsnToLocale in languages/lib/js-num.js.
func jsToLocaleJS(x float64) string {
	if math.IsNaN(x) {
		return "NaN"
	}
	sign := ""
	if math.Signbit(x) {
		sign, x = "-", -x
	}
	if math.IsInf(x, 0) {
		return sign + "∞"
	}
	if x == 0 {
		return sign + "0"
	}
	d, e := jsShortestDigits(x)
	il := e + 1
	if e < 0 {
		d, il = strings.Repeat("0", -e)+d, 1
	}
	r := jsRoundUpDigits(d, il+3)
	if len(r) > il+3 {
		il++
	}
	out := sign + jsGroupDigits(r[:il])
	if fp := strings.TrimRight(r[il:], "0"); fp != "" {
		out += "." + fp
	}
	return out
}

// jsGroupDigits is an integer's digit run, signed or not, grouped by threes with
// ",", as en-US writes it. Twin: jsnGroup in languages/lib/js-num.js.
func jsGroupDigits(ip string) string {
	sign := ""
	if strings.HasPrefix(ip, "-") {
		sign, ip = "-", ip[1:]
	}
	out := ""
	i := len(ip)
	for i > 3 {
		out = "," + ip[i-3:i] + out
		i -= 3
	}
	return sign + ip[:i] + out
}

// jsvArrayLocale is Array#toLocaleString: each element's own toLocaleString, ""
// for undefined and null, joined by ",". Twins: jsArrLocale in both
// *-interpreter.abnf, jvArrLocale in languages/lib/js-rt.metajs.
func (rt *jsrt) jsvArrayLocale(a *jsArray) string {
	var b strings.Builder
	for i, e := range a.elems {
		if i > 0 {
			b.WriteByte(',')
		}
		switch v := e.(type) {
		case float64:
			b.WriteString(jsToLocaleJS(v))
		case *jsBigInt:
			b.WriteString(jsGroupDigits(rt.jsvString(v)))
		case *jsArray:
			b.WriteString(rt.jsvArrayLocale(v))
		case *jsObject:
			// An own or class toLocaleString, else Object.prototype's: ToString.
			if r, ok := rt.jsvCallMethod(v, "toLocaleString"); ok {
				b.WriteString(rt.jsvString(r))
			} else {
				b.WriteString(rt.jsvString(v))
			}
		default:
			if !isUndefOrNull(e) {
				b.WriteString(rt.jsvString(e))
			}
		}
	}
	return b.String()
}

func jsExpForm(d string, e int) string {
	m := d[:1]
	if len(d) > 1 {
		m += "." + d[1:]
	}
	if e >= 0 {
		return m + "e+" + strconv.Itoa(e)
	}
	return m + "e-" + strconv.Itoa(-e)
}

// jsToIntOrInf is ToIntegerOrInfinity over a number. Twin of jsnToInt.
func jsToIntOrInf(n float64) float64 {
	if math.IsNaN(n) {
		return 0
	}
	return math.Trunc(n) + 0
}

// jsvRadixString is Number#toString(radix) for a radix other than 10: V8's
// DoubleToRadixCString. Twin of jsnRadixStr in languages/lib/js-num.js, where the
// rules are.
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
	if x == 0 {
		return "0"
	}
	const digits = "0123456789abcdefghijklmnopqrstuvwxyz"
	r := float64(radix)
	v := math.Abs(x)
	integer := math.Floor(v)
	fraction := v - integer
	delta := math.Max(0.5*(math.Nextafter(v, math.Inf(1))-v), math.SmallestNonzeroFloat64)
	var fd []int
	if fraction >= delta {
		for {
			fraction = float64(fraction * r)
			delta = float64(delta * r)
			digit := int(fraction)
			fd = append(fd, digit)
			fraction -= float64(digit)
			if (fraction > 0.5 || (fraction == 0.5 && digit%2 == 1)) && fraction+delta > 1 {
				for {
					if len(fd) == 0 {
						integer++
						break
					}
					last := fd[len(fd)-1]
					fd = fd[:len(fd)-1]
					if last+1 < radix {
						fd = append(fd, last+1)
						break
					}
				}
				break
			}
			if fraction < delta {
				break
			}
		}
	}
	out := ""
	for integer/r >= 1<<53 {
		integer /= r
		out += "0"
	}
	for {
		rem := math.Mod(integer, r)
		out = string(digits[int(rem)]) + out
		integer = (integer - rem) / r
		if integer <= 0 {
			break
		}
	}
	if len(fd) > 0 {
		out += "."
		for _, d := range fd {
			out += string(digits[d])
		}
	}
	if x < 0 {
		return "-" + out
	}
	return out
}

// jsvSortDefault is Array#sort with no comparator: undefined sorts last and every
// other element compares as its STRING form. V8's sort is stable, so a stable sort
// is what both halves implement.
func (rt *jsrt) jsvSort(elems []interface{}, cmp interface{}) {
	holes := 0
	for i := 0; i < len(elems); i++ {
		if elems[i] == jsHole {
			holes++
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
		return jsvStrCmp(rt.jsvString(vals[i]), rt.jsvString(vals[j])) < 0
	}
	sort.SliceStable(vals, less)
	// The undefined elements follow the sorted ones, and the absent ones come last.
	copy(elems, vals)
	for i := len(vals); i < len(elems); i++ {
		elems[i] = jsUndef
		if i >= len(elems)-holes {
			elems[i] = jsHole
		}
	}
}

func (rt *jsrt) jsvFlatten(out *[]interface{}, elems []interface{}, depth int) {
	for _, e := range elems {
		if e == jsHole {
			continue
		}
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
	// `call` and `apply` for every language, but `bind` is not in that pair. It is
	// answered HERE, js/ts-locally, rather than beside call/apply in the shared
	// runtime: the other fifteen languages have no Function.prototype and must not
	// grow one. The bound arguments are PREPENDED to the eventual call's, which is
	// the one way bind differs from call beyond remembering the receiver. The twins
	// are fnBindShim in both *-interpreter.abnf start scripts and jvBindShim in
	// lib/js-rt.metajs.
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
		// "bound f"/1 for `f.bind(null)` of `function f(a){}`.
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
		if v, handled := rt.promStatic(nil, name, args); handled {
			return v, true
		}
	}
	// Object.keys / values / entries / assign / freeze / fromEntries.
	if jsvIsObjectGlobal(target) {
		switch name {
		case "keys":
			out := &jsArray{}
			for _, k := range jsvOwnKeys(argAt(args, 0)) {
				out.elems = append(out.elems, jsvKeyUnesc(k))
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
				out.elems = append(out.elems, &jsArray{elems: []interface{}{jsvKeyUnesc(k), rt.jsvGetKey(src, k)}})
			}
			return out, true
		case "getOwnPropertyNames":
			out := &jsArray{}
			for _, k := range jsvGopnKeys(argAt(args, 0)) {
				out.elems = append(out.elems, jsvKeyUnesc(k))
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
					v := rt.jsvGetKey(src, k)
					// Object.assign is a builtin, and so strict code: a refused write raises.
					if jsvAttrN > 0 {
						if msg := rt.jsvWriteRefusal(dst, k, v); msg != "" {
							rt.bigRaise("TypeError: %s", msg)
						}
					}
					if rt.accessorCount > 0 {
						if h, acc := jsvAccHolder(dst, k); acc != nil {
							if acc.set == nil {
								rt.bigRaise("TypeError: %s", jsvGetterOnlyText(k, rt.jsvAccHolderText(dst, h)))
							}
							rt.call(acc.set, dst, []interface{}{v})
							continue
						}
					}
					rt.setMember(dst, k, v)
				}
				for _, k := range jsvSymKeys(src) {
					rt.setMember(dst, k, rt.jsvGetKey(src, k))
				}
			}
			return dst, true
		case "fromEntries":
			out := newJSObject()
			for _, e := range rt.jsvIterList(argAt(args, 0)) {
				switch e.(type) {
				case *jsObject, *jsArray, *hostFunc:
				default:
					if !isCallable(e) {
						rt.bigRaise("TypeError: Iterator value %s is not an entry object", rt.jsvString(e))
					}
				}
				out.set(rt.jsvPropKey(rt.jsvGetKey(e, "0")), rt.jsvGetKey(e, "1"))
			}
			return out, true
		case "create":
			o := rt.jsvObjCreate(argAt(args, 0))
			if len(args) > 1 && args[1] != interface{}(jsUndef) {
				rt.jsvDefineProps(o, args[1])
			}
			return o, true
		case "setPrototypeOf":
			return rt.jsvSetProto(argAt(args, 0), argAt(args, 1)), true
		case "defineProperty", "defineProperties":
			o := argAt(args, 0)
			if _, isObj := o.(*jsObject); !isObj && !isCallable(o) {
				if _, isArr := o.(*jsArray); !isArr {
					rt.bigRaise("TypeError: Object.%s called on non-object", name)
				}
			}
			if name == "defineProperties" {
				return rt.jsvDefineProps(o, argAt(args, 1)), true
			}
			rt.jsvDefineProp(o, rt.jsvPropKey(argAt(args, 1)), argAt(args, 2))
			return o, true
		case "hasOwn":
			if isUndefOrNull(argAt(args, 0)) {
				rt.bigRaise("TypeError: Cannot convert undefined or null to object")
			}
			return rt.jsvHasOwn(argAt(args, 0), rt.jsvPropKey(argAt(args, 1))), true
		case "groupBy":
			if isUndefOrNull(argAt(args, 0)) {
				rt.bigRaise("TypeError: Object.groupBy called on null or undefined")
			}
			f := argAt(args, 1)
			if !isCallable(f) {
				rt.bigRaise("TypeError: %s is not a function", rt.jsvNotFnText(f))
			}
			out := newJSObject()
			for i, v := range rt.jsvIterList(argAt(args, 0)) {
				k := rt.jsvPropKey(rt.call(f, jsUndef, []interface{}{v, float64(i)}))
				grp, found := out.props[k].(*jsArray)
				if !found {
					grp = &jsArray{}
					out.set(k, grp)
				}
				grp.elems = append(grp.elems, v)
			}
			return out, true
		case "freeze":
			return rt.jsvSetIntegrity(argAt(args, 0), 3), true
		case "seal":
			return rt.jsvSetIntegrity(argAt(args, 0), 2), true
		case "preventExtensions":
			return rt.jsvSetIntegrity(argAt(args, 0), 1), true
		case "isFrozen":
			return jsvIntegrity(argAt(args, 0)) == 3, true
		case "isSealed":
			return jsvIntegrity(argAt(args, 0)) >= 2, true
		case "isExtensible":
			return jsvIsExtensible(argAt(args, 0)), true
		case "is":
			return rt.jsvSameValue(argAt(args, 0), argAt(args, 1)), true
		}
		return nil, false
	}

	// Array.isArray on a js program's `Array` global (js_jsarrayfn). Array.from and
	// Array.of are lib/js-builtins.js's own properties of it, found by the fallback.
	if hf, ok := target.(*hostFunc); ok && hf.name == "Array" {
		if name == "isArray" {
			_, isArr := argAt(args, 0).(*jsArray)
			return isArr && !jsvIsArgs(argAt(args, 0)), true
		}
		return nil, false
	}

	// Number's statics on the `Number` global: the predicates do not convert their
	// argument, and the two parsers are the global functions. Twins: jsNumberStatic in
	// both *-interpreter.abnf, jvNumberStatic in languages/lib/js-rt.metajs.
	if hf, ok := target.(*hostFunc); ok && hf.name == "Number" {
		x, isNum := argAt(args, 0).(float64)
		fin := isNum && !math.IsNaN(x) && !math.IsInf(x, 0)
		switch name {
		case "isNaN":
			return isNum && x != x, true
		case "isFinite":
			return fin, true
		case "isInteger":
			return fin && math.Trunc(x) == x, true
		case "isSafeInteger":
			return fin && math.Trunc(x) == x && math.Abs(x) <= 9007199254740991, true
		case "parseFloat":
			return jsParseFloatJS(rt.jsvString(argAt(args, 0))), true
		case "parseInt":
			return jsParseIntJS(rt.jsvString(argAt(args, 0)), rt.toInt32(rt.jsvToNum(argAt(args, 1)))), true
		}
		return nil, false
	}

	// String.fromCharCode / String.raw on the `String` global (a host function, so
	// js_get finds no member of that name and the call lands here).
	if hf, ok := target.(*hostFunc); ok && hf.name == "String" {
		switch name {
		case "raw":
			return rt.jsvStringRaw(args), true
		case "fromCharCode":
			units := make([]uint16, len(args))
			for i, x := range args {
				units[i] = uint16(jsToInt(rt.toNumber(x)))
			}
			return strFromUnits(units), true
		case "fromCodePoint":
			// Twin of jsnCodePoint in languages/lib/js-num.js.
			var units []uint16
			for _, x := range args {
				n := rt.toNumber(x)
				if n != math.Trunc(n) || n < 0 || n > 0x10FFFF {
					rt.bigRaise("RangeError: Invalid code point %s", rt.jsvString(n))
				}
				units = append(units, utf16.Encode([]rune{rune(n)})...)
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
		if jsvIsArgs(recv) {
			return nil, false
		}
		return rt.jsvArrayMethod(recv, name, args)
	case float64:
		return rt.jsvNumberMethod(recv, name, args)
	case bool:
		// Boolean.prototype's own two (ECMA-262 20.3.3).
		switch name {
		case "toString":
			return strconv.FormatBool(recv), true
		case "valueOf":
			return recv, true
		}
	case *jsObject:
		// A wrapper's methods are its primitive's.
		if p, isWrap := recv.props["__bvof"]; isWrap {
			return rt.jsvMethod(p, name, args)
		}
		// Symbol.prototype's toString and valueOf, which a wrapped symbol reaches.
		if jsvIsSym(recv) {
			switch name {
			case "toString":
				return rt.jsvString(recv), true
			case "valueOf":
				return recv, true
			}
		}
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
		// The last match at or before the position; a NaN (an absent) one is the
		// whole string. Twin: jsLastIndex in languages/js-interpreter.abnf.
		sub := argS(0)
		subLen := rt.strLen(sub)
		start := units
		if p := rt.jsvToNum(argAt(args, 1)); p == p {
			start = jsToInt(p)
		}
		if start < 0 {
			start = 0
		}
		if start > units-subLen {
			start = units - subLen
		}
		for i := start; i >= 0; i-- {
			if rt.strRange(s, i, i+subLen) == sub {
				return float64(i), true
			}
		}
		return float64(-1), true
	case "includes":
		from := jsToInt(rt.jsvNumArg(args, 1, 0))
		if from < 0 {
			from = 0
		}
		if from > units {
			from = units
		}
		return rt.strIndexOf(rt.strRange(s, from, units), argS(0)) >= 0, true
	case "startsWith":
		sub := argS(0)
		at := jsToInt(rt.jsvNumArg(args, 1, 0))
		if at < 0 {
			at = 0
		}
		if at > units {
			at = units
		}
		return at+rt.strLen(sub) <= units && rt.strRange(s, at, at+rt.strLen(sub)) == sub, true
	case "endsWith":
		sub := argS(0)
		end := units
		if len(args) > 1 && args[1] != jsUndef {
			end = jsToInt(rt.jsvToNum(args[1]))
		}
		if end < 0 {
			end = 0
		}
		if end > units {
			end = units
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
		if len(args) > 1 && args[1] != jsUndef {
			n = jsToInt(rt.jsvToNum(args[1]))
		}
		if n < 0 {
			n = 0
		}
		if n > units-start {
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
		return caseMapMode(s, caseUpper|caseFull), true
	case "toLowerCase":
		return caseMapMode(s, caseFull|caseSigma), true
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
		// V8's String::kMaxLength, the limit "repeat" checks too.
		if want > 536870888 {
			rt.bigRaise("RangeError: Invalid string length")
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
		c := rt.jsvNumArg(args, 0, 0)
		n := jsToInt(c)
		if n < 0 || math.IsInf(c, 1) {
			rt.bigRaise("RangeError: Invalid count value: %s", rt.jsvString(c))
		}
		if len(s) > 0 && n > 536870888/units {
			rt.bigRaise("RangeError: Invalid string length")
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
	case "normalize":
		// Twin of jsNormalize in languages/js-interpreter.abnf.
		f := "NFC"
		if len(args) > 0 && args[0] != jsUndef {
			f = argS(0)
		}
		if f != "NFC" && f != "NFD" && f != "NFKC" && f != "NFKD" {
			rt.bigRaise("RangeError: The normalization form should be one of NFC, NFD, NFKC, NFKD.")
		}
		return rt.jsvNormalize(s, f), true
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
	case "toLocaleString":
		return rt.jsvArrayLocale(a), true
	// ES2023's copying methods: reverse, sort and splice on a copy, and with.
	// copyWithin moves a run in place. Twin: jsArrayMethod in js-interpreter.abnf.
	case "toReversed", "toSorted", "toSpliced":
		cp := &jsArray{elems: append([]interface{}{}, a.elems...)}
		rt.jsvArrayMethod(cp, map[string]string{"toReversed": "reverse", "toSorted": "sort", "toSpliced": "splice"}[name], args)
		return cp, true
	case "with":
		i := jsToInt(rt.jsvNumArg(args, 0, 0))
		r := i
		if r < 0 {
			r += n
		}
		if r < 0 || r >= n {
			rt.bigRaise("RangeError: Invalid index : %s", rt.jsvString(float64(i)))
		}
		cp := &jsArray{elems: append([]interface{}{}, a.elems...)}
		cp.elems[r] = argAt(args, 1)
		return cp, true
	case "copyWithin":
		to := jsvRelIndex(rt.jsvNumArg(args, 0, 0), n)
		from := jsvRelIndex(rt.jsvNumArg(args, 1, 0), n)
		end := n
		if len(args) > 2 && args[2] != jsUndef {
			end = jsvRelIndex(rt.jsvToNum(args[2]), n)
		}
		if from < end {
			a.dropIdx()
			copy(a.elems[to:], append([]interface{}{}, a.elems[from:end]...))
		}
		return a, true
	case "values":
		return jsvSeqIter(a, 1), true
	case "keys":
		return jsvSeqIter(a, 0), true
	case "entries":
		return jsvSeqIter(a, 2), true
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
		if del > n-start {
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
			if a.elems[i] != jsHole && rt.strictEq(a.elems[i], cb) {
				return float64(i), true
			}
		}
		return float64(-1), true
	case "lastIndexOf":
		from := n - 1
		if len(args) > 1 {
			from = jsToInt(rt.jsvToNum(args[1]))
			if from < 0 {
				from += n
			}
			if from > n-1 {
				from = n - 1
			}
		}
		for i := from; i >= 0; i-- {
			if a.elems[i] != jsHole && rt.strictEq(a.elems[i], cb) {
				return float64(i), true
			}
		}
		return float64(-1), true
	case "includes":
		from := 0
		if len(args) > 1 {
			from = jsvRelIndex(rt.jsvToNum(args[1]), n)
		}
		for _, e := range a.elems[from:] {
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
		if len(args) > 0 && args[0] != jsUndef {
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
	// The iteration methods skip an absent element (jsHole); map's result keeps it
	// absent.
	case "map":
		out := &jsArray{}
		for i := 0; i < n && i < len(a.elems); i++ {
			if a.elems[i] == jsHole {
				out.elems = append(out.elems, jsHole)
				continue
			}
			out.elems = append(out.elems, call3(i))
		}
		return out, true
	case "filter":
		out := &jsArray{}
		for i := 0; i < n && i < len(a.elems); i++ {
			if a.elems[i] != jsHole && rt.truthy(call3(i)) {
				out.elems = append(out.elems, a.elems[i])
			}
		}
		return out, true
	case "forEach":
		for i := 0; i < n && i < len(a.elems); i++ {
			if a.elems[i] != jsHole {
				call3(i)
			}
		}
		return jsUndef, true
	case "some":
		for i := 0; i < n && i < len(a.elems); i++ {
			if a.elems[i] != jsHole && rt.truthy(call3(i)) {
				return true, true
			}
		}
		return false, true
	case "every":
		for i := 0; i < n && i < len(a.elems); i++ {
			if a.elems[i] != jsHole && !rt.truthy(call3(i)) {
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
				if a.elems[i] != jsHole {
					idx = append(idx, i)
				}
			}
		} else {
			for i := n - 1; i >= 0; i-- {
				if a.elems[i] != jsHole {
					idx = append(idx, i)
				}
			}
		}
		at := 0
		var acc interface{}
		if len(args) > 1 {
			acc = args[1]
		} else {
			if len(idx) == 0 {
				rt.bigRaise("TypeError: Reduce of empty array with no initial value")
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
			if a.elems[i] == jsHole {
				continue
			}
			v := call3(i)
			if va, ok := v.(*jsArray); ok {
				for _, e := range va.elems {
					if e != jsHole {
						out = append(out, e)
					}
				}
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
		if len(args) > 2 && args[2] != jsUndef {
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
	case "toFixed", "toExponential", "toPrecision":
		// A missing digit count is 0 for toFixed, the shortest digits for
		// toExponential and ToString for toPrecision.
		undef := len(args) == 0 || args[0] == interface{}(jsUndef)
		n := 0.0
		if len(args) > 0 {
			n = jsToIntOrInf(rt.jsvToNum(args[0]))
		}
		var out, bad string
		switch {
		case name == "toFixed":
			out, bad = jsToFixedJS(x, n)
		case name == "toExponential":
			out, bad = jsToExponentialJS(x, n, undef)
		case undef:
			out = jsNumString(x)
		default:
			out, bad = jsToPrecisionJS(x, n)
		}
		if bad != "" {
			rt.bigRaise("%s", bad)
		}
		return out, true
	case "toString":
		radix := jsToInt(rt.jsvNumArg(args, 0, 10))
		if radix == 10 {
			return jsNumString(x), true
		}
		if radix < 2 || radix > 36 {
			// node makes an out-of-range radix a CATCHABLE RangeError, as on the
			// BigInt path beside it (jsrtjsbig.go).
			rt.bigRaise("RangeError: toString() radix argument must be between 2 and 36")
		}
		return jsvRadixString(x, radix), true
	case "valueOf":
		return x, true
	case "toLocaleString":
		return jsToLocaleJS(x), true
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
	// Object.prototype, made with the Object global (js_jsobject), and the builtin
	// prototypes by their global's name (defined beside builtinProto below).
	var objProto *jsObject
	var builtinProtoNamed func(name string) interface{}

	// println with JAVASCRIPT's ToString. The shared println host function hands the
	// raw value to Go's %v. Multiple arguments stay space-joined, each stringified on
	// its own.
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
	// A template substitution: js_jsstr, except that it is ToString proper, which a
	// symbol refuses. Twin of js_jstplstr in languages/lib/js-rt.metajs.
	m["js_jstplstr"] = func(a []uint64) uint64 {
		if jsvIsSym(u(a[0])) {
			rt.jsvSymConvErr(true)
		}
		return rt.wrapStr(rt.jsvString(u(a[0])))
	}
	// The keys object spread and an object rest pattern copy: the own enumerable
	// string keys, then the own symbol keys. Twin of js_jsspreadkeys in
	// languages/lib/js-rt.metajs.
	m["js_jsspreadkeys"] = func(a []uint64) uint64 {
		out := u(m["js_jskeys"]([]uint64{a[0]})).(*jsArray)
		// The copy reads and writes each key, so it takes them as they are stored.
		for i, k := range out.elems {
			if ks, isStr := k.(string); isStr {
				out.elems[i] = jsvKeyEsc(ks)
			}
		}
		for _, k := range jsvSymKeys(u(a[0])) {
			out.elems = append(out.elems, k)
		}
		return w(out)
	}

	// js_jstypeof is `typeof` for js/ts in BOTH compiled halves, and the twin of the
	// layer-2 function of the same name. A CLASS DESCRIPTOR IS A FUNCTION to
	// `typeof`: every class here is an ordinary object carrying __isclass, while node
	// says "function" for every class. The test stays out of typeOf itself, which is
	// shared by all sixteen languages, and out of the engine's own `is this callable`
	// guards, which decide whether a value can be CALLED - a class descriptor still
	// cannot.
	m["js_jstypeof"] = func(a []uint64) uint64 {
		v := u(a[0])
		if o, ok := v.(*jsObject); ok {
			if c, has := o.props["__isclass"]; has && c == interface{}(true) {
				return rt.wrapStr("function")
			}
			if _, isSym := o.props["__symkey"]; isSym {
				return rt.wrapStr("symbol")
			}
		}
		return rt.wrapStr(rt.typeOf(v))
	}

	// js_jsprintfn("println"|"print") answers the host FUNCTION VALUE the grammar
	// declares over the root scope's binding of the same name. Going through a
	// binding rather than through a call-site rewrite is what keeps shadowing
	// honest: a program that declares a println of its own simply overwrites it.
	// js_jssrc(text, table) registers the program's function source text, which
	// String(f) answers from then on (jsvFnText); js_jsfnsrc(f, idx) links a value
	// that is no plain closure to the function index whose text it shows. Twins of
	// both in languages/lib/runtime.c.
	m["js_jssrc"] = func(a []uint64) uint64 {
		rt.jsSrc = &jsvSrcTable{text: rt.toString(u(a[0])), tab: rt.toString(u(a[1]))}
		return 0
	}
	m["js_jsfnsrc"] = func(a []uint64) uint64 {
		jsvFnSrcSet(u(a[0]), int(rt.toNumber(u(a[1])))+1)
		return a[0]
	}
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
	// rendering, which ignores a class's own toString. Two primitive operands go
	// straight to the shared extern, which is every addition a program that adds no
	// objects performs.
	baseAdd := m["js_add"]
	m["js_jsadd"] = func(a []uint64) uint64 {
		l, r := u(a[0]), u(a[1])
		if !jsvIsObjLike(l) && !jsvIsObjLike(r) {
			return baseAdd(a)
		}
		// A symbol converts to a string when the other side is one, to a number
		// otherwise, and both throw.
		if jsvIsSym(l) || jsvIsSym(r) {
			lp, rp := l, r
			if !jsvIsSym(l) {
				lp = rt.jsvToPrimitive(l, "default")
			}
			if !jsvIsSym(r) {
				rp = rt.jsvToPrimitive(r, "default")
			}
			_, lIsStr := lp.(string)
			_, rIsStr := rp.(string)
			rt.jsvSymConvErr(lIsStr || rIsStr)
		}
		lp, rp := rt.jsvToPrimitive(l, "default"), rt.jsvToPrimitive(r, "default")
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
	// fromCharCode); this is a real host function, and String.fromCharCode is answered
	// by js_jsmcall below - which is reached because js_get has no member of that name
	// on a function value.
	m["js_jsstringfn"] = func(a []uint64) uint64 {
		return w(rt.jsvNamedHost("String", 1, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			if len(args) == 0 {
				return ""
			}
			return rt.jsvString(args[0])
		}))
	}

	// A js program's `Array` global, declared in its own scope like String: Array(n)
	// and new Array(n) are n empty slots (read as undefined), any other argument list
	// is the array of the arguments. Twins: js_jsarrayfn in languages/lib/js-rt.metajs
	// and jsArrayFn in js-interpreter.abnf.
	m["js_jsarrayfn"] = func(a []uint64) uint64 {
		return w(rt.jsvNamedHost("Array", 1, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			if len(args) == 1 {
				if n, isNum := args[0].(float64); isNum {
					if !(n >= 0 && n <= 4294967295 && n == math.Trunc(n)) {
						rt.bigRaise("RangeError: Invalid array length")
					}
					out := &jsArray{elems: make([]interface{}, int(n))}
					for i := range out.elems {
						out.elems[i] = jsHole
					}
					return out
				}
			}
			return &jsArray{elems: append([]interface{}{}, args...)}
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
	// js_jsnewclass(o, F) files on `new F()`'s fresh object o the descriptor its chain
	// starts at. A prototype the program wrote onto F starts it at that object, as
	// Object.create's does, and every instance made while it is F's prototype shares
	// one descriptor, kept on F's own. A prototype that is no object leaves o with no
	// class: Object.prototype (ECMA-262 GetPrototypeFromConstructor). Twins: newClassOf
	// in both *-interpreter.abnf, js_jsnewclass in languages/lib/js-rt.metajs.
	m["js_jsnewclass"] = func(a []uint64) uint64 {
		o, fn := u(a[0]).(*jsObject), u(a[1])
		// `new Number(v)`, `new String(v)`, `new Boolean(v)`: a wrapper, which
		// js_jsnewres fills with the primitive the call answers.
		if hf, isHost := fn.(*hostFunc); isHost && (hf.name == "Number" || hf.name == "String" || hf.name == "Boolean") {
			o.props["__bvof"] = jsUndef
			return a[0]
		}
		cd := descOf(fn)
		p, has := jsvOwnProto(fn)
		if !has {
			o.set("__class", cd)
			return a[0]
		}
		po, isObj := p.(*jsObject)
		if !isObj || isCallable(p) || isClassDesc(po) {
			return a[0]
		}
		if cd.props["__rpview"] != interface{}(po) {
			cd.set("__rpview", po)
			cd.set("__rpdesc", jsvProtoDesc(po))
		}
		o.set("__class", cd.props["__rpdesc"])
		return a[0]
	}
	// THE PROTOTYPE of a class or a constructor function: one VIEW object per class
	// descriptor, {__protoof: descriptor}, kept on the descriptor as __pview. The
	// descriptor already holds the instance methods, so the view only answers for it:
	// a read that misses its own slots is the descriptor chain's method (or, on the
	// Error classes, the name), and a FUNCTION written onto it is installed on the
	// descriptor too, wrapped to take the receiver prepended as a class method does -
	// which is what makes `F.prototype.m = function () {...}` a method of every
	// instance, and an override of a class method of that name. Every callable that
	// is not a builtin global answers one, arrows and methods included (node: no
	// prototype), as every callable is also accepted by `new` here. Twins: protoView
	// in both *-interpreter.abnf, jvProtoView in languages/lib/js-rt.metajs.
	// SYMBOLS. A symbol is an object {__symdesc, __symkey}: a new one per Symbol()
	// call, one per name for the well-known ones and per key for Symbol.for. As a
	// property KEY it is its __symkey - "__#@" and a name no identifier can spell - so
	// every key listing already skips it, as it skips the engine's own "__" slots, and
	// the private-name exemption files a class method under it. Every member extern
	// that takes a key turns a symbol into its key first (see the end of this
	// function). Twins: jsSym* in both *-interpreter.abnf, jvSym* in
	// languages/lib/js-rt.metajs.
	symN := 0
	symWK := map[string]*jsObject{}
	symReg := map[string]*jsObject{}
	// Every symbol by its key, for Object.getOwnPropertySymbols.
	symByKey := map[string]*jsObject{}
	symMake := func(desc interface{}, key string) *jsObject {
		s := newJSObject()
		s.set("__symdesc", desc)
		s.set("__symkey", key)
		symByKey[key] = s
		return s
	}
	symWellKnown := func(n string) *jsObject {
		if s, found := symWK[n]; found {
			return s
		}
		s := symMake("Symbol."+n, "__#@"+n)
		symWK[n] = s
		return s
	}
	symbolFn := jsHostFunc("Symbol", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		var desc interface{} = jsUndef
		if len(args) > 0 && args[0] != jsUndef {
			desc = rt.jsvString(args[0])
		}
		symN++
		return symMake(desc, "__#@"+strconv.Itoa(symN)+"@")
	})
	jsvFnMetaSet(symbolFn, "Symbol", 0)
	m["js_jssymbolfn"] = func(a []uint64) uint64 {
		rt.jsvGlobalSet("Symbol", symbolFn)
		return w(symbolFn)
	}
	// A STATIC METHOD of a builtin global read as a value - `typeof Object.keys`,
	// `const k = Object.keys` - one interned function per global and name, which
	// re-enters js_jsmcall on the global. The names are the ones js_jsmcall answers;
	// the length is node's. Twins: staticValue in both *-interpreter.abnf,
	// jvStaticValue in languages/lib/js-rt.metajs.
	staticVals := map[string]*hostFunc{}
	staticValue := func(g interface{}, name string) interface{} {
		if name == "__#@species" && jsvHasSpecies(g) {
			return g
		}
		gname := ""
		if jsvIsObjectGlobal(g) {
			gname = "Object"
		} else if hf, ok := g.(*hostFunc); ok {
			gname = hf.name
		}
		if g == interface{}(symbolFn) && jsvInNames(jsvSymWKNames, name) {
			return symWellKnown(name)
		}
		if gname == "Number" {
			if c, isConst := jsvNumberConst(name); isConst {
				return c
			}
			// Number.parseInt and Number.parseFloat ARE the two globals (ECMA-262
			// 21.1.2.12/13).
			if name == "parseInt" {
				return u(m["js_jsparseintfn"](nil))
			}
			if name == "parseFloat" {
				return u(m["js_jsparsefloatfn"](nil))
			}
		}
		if !jsvInNames(jsvStaticNames[gname], name) {
			return nil
		}
		key := gname + "." + name
		if f, found := staticVals[key]; found {
			return f
		}
		f := jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return u(m["js_jsmcall"]([]uint64{w(g), w(name), w(&jsArray{elems: append([]interface{}{}, args...)})}))
		})
		length := 1
		if jsvInNames(jsvStaticLen2, name) {
			length = 2
		}
		if name == "defineProperty" {
			length = 3
		}
		jsvFnMetaSet(f, name, length)
		staticVals[key] = f
		return f
	}
	protoView := func(d *jsObject) *jsObject {
		if v, ok := d.props["__pview"].(*jsObject); ok {
			return v
		}
		v := newJSObject()
		v.set("__protoof", d)
		d.set("__pview", v)
		return v
	}
	protoRead := func(d *jsObject, name string) interface{} {
		if name == "constructor" {
			if fn, has := d.props["__ctorfn"]; has {
				return fn
			}
			return d
		}
		if v, found := jsvChainRead(d, name); found {
			return v
		}
		if name == "name" || name == "message" {
			guard := 0
			for c, _ := interface{}(d).(*jsObject); c != nil && guard < 64; guard++ {
				if en, has := c.props["__errname"]; has {
					if name == "message" {
						return ""
					}
					return en
				}
				c, _ = c.props["__super"].(*jsObject)
			}
		}
		return nil
	}
	protoWrite := func(d *jsObject, name string, v interface{}) {
		if !isCallable(v) || (jsvInternalKey(name) && !strings.HasPrefix(name, "__#@")) {
			return
		}
		if _, had := d.props[name]; !had && !jsvInternalKey(name) {
			pw, _ := d.props["__pw"].(*jsArray)
			if pw == nil {
				pw = &jsArray{}
				d.set("__pw", pw)
			}
			pw.elems = append(pw.elems, name)
		}
		d.set(name, jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			if len(args) == 0 {
				return rt.call(v, jsUndef, nil)
			}
			return rt.call(v, args[0], args[1:])
		}))
	}
	// js_jsnewcall(nt, ctor, args): a class constructor's call, made with new.target
	// nt - the class `new` names, or the running new.target for super(). The arming
	// is cleared afterwards, since a builtin parent's host constructor never takes
	// it. Twin of js_jsnewcall in languages/lib/js-rt.metajs.
	m["js_jsnewcall"] = func(a []uint64) uint64 {
		args, ok := u(a[2]).(*jsArray)
		if !ok {
			rt.fail("js_jsnewcall args must be an array")
		}
		rt.pendingNewTarget = u(a[0])
		r := rt.callH(u(a[1]), jsUndef, args.elems, a[2])
		rt.pendingNewTarget = nil
		return w(r)
	}
	// [[Construct]]: an object the constructor RETURNS wins over the fresh receiver,
	// and a wrapper js_jsnewclass made holds the primitive.
	m["js_jsnewres"] = func(a []uint64) uint64 {
		r := u(a[0])
		if r == jsUndef {
			return a[1]
		}
		if jsvIsObjLike(r) || isCallable(r) {
			return a[0]
		}
		if o, isObj := u(a[1]).(*jsObject); isObj {
			if _, isWrap := o.props["__bvof"]; isWrap {
				o.props["__bvof"] = r
			}
		}
		return a[1]
	}

	baseInstanceof := m["js_instanceof"]
	// js_jsargs(args, off): the arguments object of a call, a copy of its argument
	// array from index off (an instance method's receiver comes first) carrying the
	// arguments kind (jsvIsArgs). Twin: js_jsargs in languages/lib/js-rt.metajs.
	m["js_jsargs"] = func(a []uint64) uint64 {
		src, _ := u(a[0]).(*jsArray)
		off := int(rt.toNumber(u(a[1])))
		c := &jsArray{kind: jsvArgsKind(0)}
		if src != nil && off < len(src.elems) {
			c.elems = append([]interface{}(nil), src.elems[off:]...)
			c.kind = jsvArgsKind(len(c.elems))
		}
		return w(c)
	}
	// The MAPPED arguments object of a sloppy function with plain parameters (ECMA-262
	// 10.4.4.7). js_jsargmap(names, arguments, scope) is the names mapped to
	// arguments[0..n), the parameters below the argument count, which the call's scope
	// keeps under "0argm"; it files the scope on the arguments object (the hidden slot
	// __argsc of its named properties). js_jsargsync(scope, name, v), after a store to
	// a name, copies v into arguments when the binding the store reached is a mapped
	// parameter; js_jsargput(obj, k, v, unmap), after a store to obj[k] through any
	// reference, copies v into the parameter k maps to when obj is a mapped arguments
	// object (after a delete, unmap true, k stops being mapped). Twins: the same
	// externs in languages/lib/js-rt.metajs, jsArgSync / jsArgPut in both
	// *-interpreter.abnf.
	m["js_jsargmap"] = func(a []uint64) uint64 {
		names, _ := u(a[0]).(*jsArray)
		args, _ := u(a[1]).(*jsArray)
		out := &jsArray{}
		if args != nil {
			p := jsvArrNamed(args)
			if p == nil {
				p = newJSObject()
				args.props = p
			}
			p.set("__argsc", rt.scopeOf(a[2]))
		}
		if names != nil && args != nil {
			n := len(names.elems)
			if len(args.elems) < n {
				n = len(args.elems)
			}
			out.elems = append(out.elems, names.elems[:n]...)
		}
		return w(out)
	}
	m["js_jsargsync"] = func(a []uint64) uint64 {
		name := rt.toString(u(a[1]))
		for s := rt.scopeOf(a[0]); s != nil; s = s.parent {
			if !s.has(name) {
				continue
			}
			mv, mapped := s.get("0argm")
			names, _ := mv.(*jsArray)
			av, _ := s.get("arguments")
			args, isArgs := av.(*jsArray)
			if mapped && names != nil && isArgs && jsvIsArgs(args) {
				for i := len(names.elems) - 1; i >= 0; i-- {
					if rt.toString(names.elems[i]) == name {
						if i < len(args.elems) {
							args.elems[i] = u(a[2])
						}
						break
					}
				}
			}
			break
		}
		return 0
	}
	m["js_jsargput"] = func(a []uint64) uint64 {
		obj := u(a[0])
		if !jsvIsArgs(obj) {
			return 0
		}
		p := jsvArrNamed(obj.(*jsArray))
		if p == nil {
			return 0
		}
		s, filed := p.props["__argsc"].(*jsScope)
		if !filed {
			return 0
		}
		mv, _ := s.get("0argm")
		names, _ := mv.(*jsArray)
		if names == nil {
			return 0
		}
		if i := rt.arrIndexOf(u(a[1])); i >= 0 && i < len(names.elems) {
			if n := rt.toString(names.elems[i]); n != "" {
				if rt.truthy(u(a[3])) {
					names.elems[i] = ""
				} else {
					s.put(n, u(a[2]))
				}
			}
		}
		return 0
	}
	m["js_jsinstanceof"] = func(a []uint64) uint64 {
		v, target := u(a[0]), u(a[1])
		if !jsvIsObjLike(target) && !isCallable(target) {
			rt.bigRaise("TypeError: Right-hand side of 'instanceof' is not an object")
		}
		// A [Symbol.hasInstance] method on the right decides. Twins: instanceOf in
		// js-interpreter.abnf, js_jsinstanceof in languages/lib/js-rt.metajs.
		if _, isObj := target.(*jsObject); isObj || isCallable(target) {
			if f := u(m["js_jsvget"]([]uint64{a[1], rt.wrapStr("__#@hasInstance")})); !isUndefOrNull(f) {
				rt.jsvNeedFn(f)
				return boolH(rt.truthy(rt.call(f, target, []interface{}{v})))
			}
		}
		if jsvIsObjectGlobal(target) {
			return boolH(jsvIsObjLike(v) || isCallable(v))
		}
		// The prototype of a class extending a builtin is an instance of the builtin.
		if o, isObj := v.(*jsObject); isObj {
			c, _ := o.props["__protoof"].(*jsObject)
			for guard := 0; c != nil && guard < 64; guard++ {
				if b, isHost := c.props["__ctorfn"].(*hostFunc); isHost && interface{}(b) == target {
					return boolH(true)
				}
				c, _ = c.props["__super"].(*jsObject)
			}
		}
		if hf, isHost := target.(*hostFunc); isHost {
			switch hf.name {
			case "Function":
				return boolH(isCallable(v) || isClassDesc(v))
			case "Promise":
				return boolH(promIs(v))
			case "RegExp":
				return boolH(rxIsRegexObj(v))
			case "Number", "String", "Boolean", "BigInt", "Symbol":
				// A wrapper is an instance of its primitive's global; the prototype
				// object it may itself be is not.
				p, isWrap := jsvWrapped(v)
				return boolH(isWrap && jsvWrapCtor(p) == hf.name && v != builtinProtoNamed(hf.name))
			}
		}
		// An object that is no constructor (and has no Symbol.hasInstance, above).
		if o, isObj := target.(*jsObject); (isObj && !isClassDesc(o) && !jsvIsArrayGlobal(target)) || !(isObj || isCallable(target)) {
			rt.bigRaise("TypeError: Right-hand side of 'instanceof' is not callable")
		}
		if jsvIsArrayGlobal(target) {
			_, isArr := v.(*jsArray)
			return boolH(isArr && !jsvIsArgs(v))
		}
		if isCallable(target) {
			o, isObj := v.(*jsObject)
			if !isObj {
				return boolH(false)
			}
			// Once the program wrote F.prototype = p: is p on v's chain, as a level's
			// own view or as the class view it is?
			var d interface{}
			var view interface{}
			if p, has := jsvOwnProto(target); has {
				po, isObj := p.(*jsObject)
				if !isObj && !isCallable(p) {
					rt.bigRaise("TypeError: Function has non-object prototype '%s' in instanceof check", rt.jsvString(p))
				}
				view = p
				if isObj {
					if pd, isView := po.props["__protoof"].(*jsObject); isView {
						d = pd
					}
				}
			} else {
				cd, ok := ctorDescs[target]
				if !ok {
					return boolH(false)
				}
				d = cd
			}
			for cls := o.props["__class"]; cls != nil; {
				if cls == d && d != nil {
					return boolH(true)
				}
				if c, isC := cls.(*jsObject); isC && view != nil && c.props["__pview"] == view {
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
		if cls := jsvArrClass(v); cls != nil {
			for c, guard := cls, 0; c != nil && guard < 64; guard++ {
				if c == target {
					return boolH(true)
				}
				c, _ = c.props["__super"].(*jsObject)
			}
			return boolH(false)
		}
		return baseInstanceof(a)
	}

	// 'delete a[i]' on an ARRAY: the length is unchanged and the slot reads as
	// undefined. (A real JS hole cannot be modelled here: the interpreter half's
	// arrays are host arrays that carry no side data, so both halves blank the slot.)
	baseDel := m["js_del"]
	m["js_jsdel"] = func(a []uint64) uint64 {
		// A delete the marks refuse (jsvDeleteRefusal) answers false; js_jsrefuse
		// raises it in strict code.
		if jsvAttrN > 0 {
			if msg := rt.jsvDeleteRefusal(u(a[0]), rt.toString(u(a[1]))); msg != "" {
				jsvRefused = msg
				return boolH(false)
			}
		}
		if arr, ok := u(a[0]).(*jsArray); ok {
			if i := rt.arrIndexOf(u(a[1])); i >= 0 {
				if i < len(arr.elems) {
					arr.elems[i] = jsHole
				}
			} else if p := jsvArrNamed(arr); p != nil {
				jsvObjDel(p, rt.toString(u(a[1])))
			}
			return boolH(true)
		}
		// A class object's own properties are its statics; the descriptor's own slots
		// are the instance methods, which a delete on the class never reaches.
		if cls, isStatic := jsvIsStaticSide(u(a[0]), rt.toString(u(a[1]))); isStatic {
			jsvStaticDel(cls, rt.toString(u(a[1])))
			return boolH(true)
		}
		if jsvIsFnLike(u(a[0])) {
			if n := rt.toString(u(a[1])); n == "name" || n == "length" {
				r := newJSObject()
				r.set("del", true)
				jsvFnPropsMake(u(a[0])).set("__fn"+n, r)
				return boolH(true)
			}
		}
		// A function's own properties live in the side table.
		if p := jsvFnPropsOf(u(a[0])); p != nil {
			jsvObjDel(p, rt.toString(u(a[1])))
			return boolH(true)
		}
		return baseDel(a)
	}

	// The `Object` global, declared by the grammar in the PROGRAM's own scope so no
	// other language sees it. It keeps the root binding's prototype, so the
	// Object.prototype.hasOwnProperty idiom still resolves.
	// Object(v) and new Object(v) are ToObject, with a fresh object for undefined and
	// null; its `prototype` is objProto (builtinProtoNamed).
	m["js_jsobject"] = func(a []uint64) uint64 {
		f := jsHostFunc("Object", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return jsvToObject(argAt(args, 0))
		})
		jsvFnMetaSet(f, "Object", 1)
		rt.jsvGlobalSet("Object", f)
		// Object.prototype.toString.call(v) is the classic type probe. The root
		// binding's `prototype` carries no methods, so this one does: the jsvProtoFns,
		// which read their receiver from the call's `this`. __objproto marks it for
		// isPrototypeOf. Twins: js_jsobject in languages/lib/js-rt.metajs and
		// jsObjectProto in both *-interpreter.abnf start scripts.
		objProto = newJSObject()
		objProto.props["__objproto"] = true
		for _, n := range jsvProtoNames {
			objProto.set(n, jsvProtoFns[n])
		}
		return w(f)
	}
	// Boolean(v) is ToBoolean; `new Boolean(v)` is a wrapper (js_jsnewclass).
	m["js_jsbooleanfn"] = func(a []uint64) uint64 {
		f := jsHostFunc("Boolean", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return rt.truthy(argAt(args, 0))
		})
		jsvFnMetaSet(f, "Boolean", 1)
		rt.jsvGlobalSet("Boolean", f)
		return w(f)
	}
	// The Function global, for `f instanceof Function` and Function.prototype. No
	// engine here compiles source text at run time, so a call is the EvalError node
	// raises where code generation from strings is disallowed.
	m["js_jsfunctionfn"] = func(a []uint64) uint64 {
		f := jsHostFunc("Function", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			rt.bigRaise("EvalError: Code generation from strings disallowed for this context")
			return jsUndef
		})
		jsvFnMetaSet(f, "Function", 1)
		rt.jsvGlobalSet("Function", f)
		return w(f)
	}

	// `class A extends Object`: the Object global is not a class descriptor here, so
	// the heritage stands for this one - no members, and a constructor that does
	// nothing, which is all Object's own construction gives a subclass instance. A
	// plain constructor FUNCTION F stands as the descriptor `new F()` files on its
	// instances (descOf), or, once the program wrote F.prototype = p, as a fresh one
	// whose view is p; its __ctor runs F on the new `this`, and jsStaticsFind reads
	// F's own properties through __ctorfn. Any other heritage value comes back
	// unchanged. Twins: js_jsheritage in languages/lib/js-rt.metajs and jsHeritage in
	// both *-interpreter.abnf.
	// `class C extends Array`: the Array global stands as arrayRoot, whose
	// constructor makes what Array(...args) makes and files new.target's class on it.
	// Twins: jsArrayRootClass in both *-interpreter.abnf, jvArrayRoot in
	// languages/lib/js-rt.metajs.
	var objectRoot, arrayRoot *jsObject
	// `class C extends Promise` (RegExp, Number, String, Boolean): the builtin stands
	// as a descriptor of its own, whose constructor makes the object `new B()` makes
	// and files new.target's class on it. Twins: jsHostRoot in both
	// *-interpreter.abnf, jvHostRoot in languages/lib/js-rt.metajs.
	hostRoots := map[*hostFunc]*jsObject{}
	hostRoot := func(b *hostFunc) *jsObject {
		if d, ok := hostRoots[b]; ok {
			return d
		}
		d := newJSObject()
		d.set("__isclass", true)
		d.set("__name", b.name)
		d.set("__ctorfn", b)
		d.set("__ctor", jsHostFunc("", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			nt, _ := rt.pendingNewTarget.(*jsObject)
			rt.pendingNewTarget = nil
			o := newJSObject()
			m["js_jsnewclass"]([]uint64{w(o), w(b)})
			r := m["js_call_new"]([]uint64{w(b), w(o), w(&jsArray{elems: append([]interface{}{}, args[1:]...)})})
			rt.pendingNewTarget = nil
			res := u(m["js_jsnewres"]([]uint64{r, w(o)}))
			if ro, isObj := res.(*jsObject); isObj && nt != nil && isClassDesc(nt) {
				ro.set("__class", nt)
			}
			return res
		}))
		hostRoots[b] = d
		return d
	}
	m["js_jsheritage"] = func(a []uint64) uint64 {
		if hf, isHost := u(a[0]).(*hostFunc); isHost {
			switch hf.name {
			case "Promise", "RegExp", "Number", "String", "Boolean":
				return w(hostRoot(hf))
			}
		}
		if jsvIsArrayGlobal(u(a[0])) {
			if arrayRoot == nil {
				arrayFn := u(a[0])
				arrayRoot = newJSObject()
				arrayRoot.set("__isclass", true)
				arrayRoot.set("__name", "Array")
				arrayRoot.set("__ctorfn", arrayFn)
				arrayRoot.set("__ctor", jsHostFunc("", func(rt *jsrt, this uint64, args []interface{}) interface{} {
					nt, _ := rt.pendingNewTarget.(*jsObject)
					rt.pendingNewTarget = nil
					r := rt.call(arrayFn, jsUndef, args[1:])
					if arr, isArr := r.(*jsArray); isArr && nt != nil && isClassDesc(nt) {
						jsvArrTag(arr, nt)
					}
					return r
				}))
			}
			return w(arrayRoot)
		}
		if fn, isFn := u(a[0]).(*jsClosure); isFn {
			d := descOf(fn)
			if p, has := jsvOwnProto(fn); has {
				if po, isObj := p.(*jsObject); isObj && !isClassDesc(po) {
					d = newJSObject()
					d.set("__isclass", true)
					d.set("__ctorfn", fn)
					d.set("__pview", po)
					if pc, hasC := po.props["__class"]; hasC {
						d.set("__super", pc)
					}
				}
			}
			if _, has := d.props["__ctor"]; !has {
				d.set("__ctor", jsHostFunc("", func(rt *jsrt, this uint64, args []interface{}) interface{} {
					return rt.call(fn, args[0], args[1:])
				}))
			}
			return w(d)
		}
		if !jsvIsObjectGlobal(u(a[0])) {
			return a[0]
		}
		if objectRoot == nil {
			objectRoot = newJSObject()
			objectRoot.set("__isclass", true)
			objectRoot.set("__name", "Object")
			objectRoot.set("__ctor", jsHostFunc("", func(rt *jsrt, this uint64, args []interface{}) interface{} {
				return jsUndef
			}))
		}
		return w(objectRoot)
	}

	// js_jsmget hides the String/Array method names from js_get, so that the CALL
	// site falls into js_jsmcall below instead of the shared boundMethod path, whose
	// semantics differ for several of them.
	// ----- CLOSING AN ITERATOR ON AN EARLY EXIT -----
	//
	// node closes a for-of's iterator on every abrupt exit from the loop body, not
	// just on `break`: a `return` out of the body and a labeled break to an OUTER
	// statement run the iterator's return() too, and neither reaches a block
	// makeForOf owns - one rets the frame, the other branches to the outer label's
	// exit. The emitter closes them BY VALUE instead: a for-of's entry block
	// dominates its body, so the loop's iterable handle is a legal operand at every
	// jump inside it, and one of these is emitted per loop being left.
	//
	// There is deliberately NO runtime open-iterator stack here - see the forOfIters
	// note in languages/js-to-llvm-ir.abnf: `for await` suspends INSIDE its own
	// for-of, so a stack unwound by DEPTH (what a `throw` would need) would close the
	// wrong loop.
	//
	// The twins are js_jsiterclose in languages/lib/js-rt.metajs and closeIterator
	// in the two *-interpreter.abnf start scripts.
	// js_jsiterchk is the iteration sites' answer for a value with no callable
	// "__#@iterator" member: an array or a string passes through to be read by
	// index, and a generator, whose [Symbol.iterator] is answered only for a symbol
	// key (it returns the generator), to be driven by next(). Anything else is
	// GetIterator's TypeError. Twin of js_jsiterchk in languages/lib/js-rt.metajs;
	// the interpreters' jsIterOf raises in place.
	m["js_jsiterchk"] = func(a []uint64) uint64 {
		switch u(a[0]).(type) {
		case *jsArray, string, *jsGenerator:
			return a[0]
		}
		rt.jsvNotIterable(u(a[0]))
		return a[0]
	}

	m["js_jsiterclose"] = func(a []uint64) uint64 {
		it := u(a[0])
		if !isUndefOrNull(it) {
			// js_jsvget, the member VALUE read: a class instance's return() lives on
			// its __class descriptor, where rt.getMember does not look.
			if r := u(m["js_jsretvget"]([]uint64{a[0], w("return")})); isCallable(r) {
				rt.call(r, it, nil)
			}
		}
		return w(jsUndef)
	}

	// ----- Async generators -----
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
	// hot path. They answer 0 or 1 as a plain integer, never a
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
	// ----- g.return(v) -----
	//
	// node RESUMES the body with a return completion at the yield it is parked at, so
	// the finally clauses around that yield run as the body's own code: one that
	// yields makes return() answer {value, done: false} and leaves the generator
	// suspended there, one that returns overrides v, and at a yield* the completion
	// goes to the delegate's return(). Here the completion is a MARKER raised at the
	// yield as an ordinary throw (throwInto); the emitter keeps it away from every
	// catch clause of a plain generator body and turns it back into `return v` at
	// the body's top, and a yield* hands it to its delegate (js_jsgenretp). A
	// generator that has not started, or is done, runs nothing and raises the marker
	// straight back, which is {v, done: true}. Twins: jvGenReturn in
	// languages/lib/js-rt.metajs, genClose in both *-interpreter.abnf.
	genReturn := func(g *jsGenerator) *hostFunc {
		return jsHostFunc("return", func(rt *jsrt, this uint64, args []interface{}) (res interface{}) {
			v := argAt(args, 0)
			mk := newJSObject()
			mk.props["__jsgret"] = true
			mk.props["v"] = v
			defer func() {
				if p := recover(); p != nil {
					if t, isThrow := p.(*jsThrown); !isThrow || t.value != mk {
						panic(p)
					}
					g.retValue = v
					r := newJSObject()
					r.set("value", v)
					r.set("done", true)
					res = r
				}
			}()
			return g.throwInto(rt, mk)
		})
	}
	// Is this the return-completion marker genReturn raises?
	m["js_jsgenretp"] = func(a []uint64) uint64 {
		if o, isObj := u(a[0]).(*jsObject); isObj && o.props["__jsgret"] == true {
			return w(true)
		}
		return w(false)
	}
	// The emitters read a member spelled `.return` through these two instead of
	// js_jsmget / js_jsvget, so that no other member read pays for the test.
	m["js_jsretmget"] = func(a []uint64) uint64 {
		if g, isGen := u(a[0]).(*jsGenerator); isGen {
			return w(genReturn(g))
		}
		return m["js_jsmget"](a)
	}
	m["js_jsretvget"] = func(a []uint64) uint64 {
		if g, isGen := u(a[0]).(*jsGenerator); isGen {
			return w(genReturn(g))
		}
		return m["js_jsvget"](a)
	}
	// IteratorClose on the delegate of a yield* an async generator's close unwinds:
	// its return() runs. Twin of js_jsdelegclose in languages/lib/js-rt.metajs.
	m["js_jsdelegclose"] = func(a []uint64) uint64 {
		if rf := u(m["js_jsretvget"]([]uint64{a[0], w("return")})); isCallable(rf) {
			rt.call(rf, u(a[0]), nil)
		}
		return jsHUndefined
	}
	// A String / Array / Number builtin method READ AS A VALUE: node's prototype
	// function, one per kind and name whatever the receiver, taking `this` from its
	// call site (`f.call([9], 3)` pushes onto [9]). Interned per runtime; a call
	// re-enters js_jsmcall on the call-site receiver. Twins: builtinValue in both
	// *-interpreter.abnf, jvBuiltinValue in languages/lib/js-rt.metajs.
	builtinVals := map[string]*hostFunc{}
	builtinValue := func(fam byte, name string) *hostFunc {
		key := string(fam) + name
		if f, found := builtinVals[key]; found {
			return f
		}
		f := jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			recv := rt.jsvBuiltinRecv(fam, name, rt.unwrap(this))
			// An Array's valueOf (Object.prototype's): the receiver itself.
			if fam == 'a' && name == "valueOf" {
				return recv
			}
			// Array.prototype's methods run on any array, the arguments object too,
			// which js_jsmcall's own dispatch refuses them.
			if arr, isArr := recv.(*jsArray); isArr && fam == 'a' {
				if v, ok := rt.jsvArrayMethod(arr, name, args); ok {
					return v
				}
			}
			return u(m["js_jsmcall"]([]uint64{w(recv), w(name), w(&jsArray{elems: append([]interface{}{}, args...)})}))
		})
		jsvNativeFn(f, name, fam)
		builtinVals[key] = f
		return f
	}
	// Array.prototype is an empty array (Array.isArray answers true, as in node), and
	// String.prototype, Number.prototype and Boolean.prototype the wrapper objects of
	// "", 0 and false (jsvWrap), whose __bvof jsvBuiltinFam asks; either way its members
	// are the builtin method values. Function.prototype is a function that answers
	// undefined; RegExp, Promise and BigInt answer a bare object. Twins: builtinProto in
	// both *-interpreter.abnf, jvBuiltinProto in languages/lib/js-rt.metajs.
	builtinProtos := map[string]interface{}{}
	builtinProto := func(fn interface{}) interface{} {
		hf, isHost := fn.(*hostFunc)
		if !isHost {
			return nil
		}
		return builtinProtoNamed(hf.name)
	}
	builtinProtoNamed = func(name string) interface{} {
		if p, found := builtinProtos[name]; found {
			return p
		}
		var p interface{}
		switch name {
		case "Object":
			if objProto == nil {
				return nil
			}
			return objProto
		case "Array":
			p = &jsArray{}
		case "String":
			p = jsvWrap("")
		case "Number":
			p = jsvWrap(float64(0))
		case "Boolean":
			p = jsvWrap(false)
		case "Function":
			p = jsHostFunc("", func(rt *jsrt, this uint64, args []interface{}) interface{} { return jsUndef })
		case "RegExp", "Promise", "BigInt":
			p = newJSObject()
		default:
			return nil
		}
		builtinProtos[name] = p
		return p
	}
	// js_jsvget is the VALUE read of a member - `const f = p.m`, `typeof p.m` - as
	// opposed to js_jsmget, which the emitter uses at a member-followed-by-a-call site
	// and which deliberately HIDES the String/Array method names so the call falls into
	// js_jsmcall. The difference this extern exists for: an instance's methods live on
	// its __class descriptor, not as own properties, yet node says `typeof p.m` is
	// "function". The answer is jsvMethodValue: one unbound function per method, which
	// takes `this` from its call site. The layer-2 twin is js_jsvget in
	// languages/lib/js-rt.metajs and the interpreters' is getMember's class-method arm.
	var vget func(a []uint64) uint64
	vget = func(a []uint64) uint64 {
		if v, isAcc := rt.jsvArrAccRead(u(a[0]), rt.toString(u(a[1]))); isAcc {
			return w(v)
		}
		if cls := jsvArrClass(u(a[0])); cls != nil {
			if v, found := rt.jsvArrSubRead(u(a[0]).(*jsArray), cls, rt.toString(u(a[1]))); found {
				return w(v)
			}
		}
		if fam := jsvBuiltinFam(u(a[0]), rt.toString(u(a[1]))); fam != 0 {
			return w(builtinValue(fam, rt.toString(u(a[1]))))
		}
		// Function.prototype's call / apply / bind read as a value: a function over the
		// receiver that makes the same call `f.call(...)` makes. Twins: fnCallShim and
		// its neighbours in both *-interpreter.abnf, jvCallShim and its neighbours in
		// languages/lib/js-rt.metajs.
		if k := rt.toString(u(a[1])); (k == "call" || k == "apply" || k == "bind") && jsvIsFnLike(u(a[0])) {
			fn := u(a[0])
			return w(jsvNativeFn(jsHostFunc(k, func(rt *jsrt, this uint64, args []interface{}) interface{} {
				if k == "bind" {
					v, _ := rt.jsvMethod(fn, k, args)
					return v
				}
				return rt.builtinMethod(&boundMethod{recv: fn, name: k}, args)
			}), k, 0))
		}
		// A CLASS DESCRIPTOR reads from its static namespace and from nowhere else,
		// its prototype aside. Ordered in front of baseGet because the descriptor's own
		// property map is where its INSTANCE methods live, and answering one of those
		// for `C.m` would be wrong.
		if cls, isStatic := jsvIsStaticSide(u(a[0]), rt.toString(u(a[1]))); isStatic {
			if rt.toString(u(a[1])) == "prototype" {
				return w(protoView(cls))
			}
			return rt.jsvStaticSideRead(cls, rt.toString(u(a[1])))
		}
		// The FAST PATH must not re-wrap: the handle baseGet answered is returned
		// as it stands, and the fallback below runs only when the read found
		// nothing. The layer-2 twin pays the same attention - see js_jsvget in
		// languages/lib/js-rt.metajs.
		r := baseGet(a)
		if !isUndefOrNull(u(r)) {
			return r
		}
		recv, isObj := u(a[0]).(*jsObject)
		if !isObj {
			// `f.name` and `f.length`, before the own-property side table because both
			// are non-writable in node: an `f.name = "q"` that reached the table must
			// not win, and js_jsmset drops it for the same reason.
			if v := jsvFnIntrinsic(u(a[0]), rt.toString(u(a[1]))); v != nil {
				return w(v)
			}
			// An own property written onto a FUNCTION lives in the side table, not
			// on the closure.
			if p := jsvFnPropsOf(u(a[0])); p != nil {
				if v, found := p.props[rt.toString(u(a[1]))]; found {
					return w(v)
				}
			}
			// ...and a function's Object.prototype surface, AFTER the side table so
			// an own `f.toString = ...` still wins.
			if m := rt.jsvFnMethod(u(a[0]), rt.toString(u(a[1]))); m != nil {
				return w(m)
			}
			if rt.toString(u(a[1])) == "constructor" {
				if c := rt.jsvBuiltinCtor(u(a[0])); c != nil {
					return w(c)
				}
			}
			if f := staticValue(u(a[0]), rt.toString(u(a[1]))); f != nil {
				return w(f)
			}
			if name := rt.toString(u(a[1])); isCallable(u(a[0])) && jsvIsProtoName(name) {
				return w(jsvProtoShim(name))
			}
			if rt.toString(u(a[1])) == "prototype" {
				if p := builtinProto(u(a[0])); p != nil {
					return w(p)
				}
				if isCallable(u(a[0])) {
					return w(protoView(descOf(u(a[0]))))
				}
			}
			return r
		}
		name := rt.toString(u(a[1]))
		if jsvInternalKey(name) && !jsvPrivateKey(name) {
			return r
		}
		if d, isSym := recv.props["__symdesc"]; isSym && name == "description" {
			return w(d)
		}
		if p, isWrap := recv.props["__bvof"]; isWrap {
			// An instance of a class extending Number / String / Boolean (jsHostRoot):
			// its class first.
			if cls := recv.props["__class"]; cls != nil {
				if name == "constructor" {
					if c, found := jsvConstructorOf(recv); found {
						return w(c)
					}
				}
				if v, found := jsvChainRead(cls, name); found {
					return w(rt.jsvChainGet(v, recv))
				}
			}
			return vget([]uint64{w(p), a[1]})
		}
		if d, isView := recv.props["__protoof"].(*jsObject); isView {
			if v := protoRead(d, name); v != nil {
				return w(rt.jsvChainGet(v, recv))
			}
			return r
		}
		if f := staticValue(recv, name); f != nil {
			return w(f)
		}
		// `e.constructor` is the CLASS an instance was built from (and, for
		// `new F()`, the function F itself).
		if name == "constructor" {
			if c := rt.jsvBuiltinCtor(recv); c != nil {
				return w(c)
			}
		}
		if v, found := jsvChainRead(recv.props["__class"], name); found {
			return w(rt.jsvChainGet(v, recv))
		}
		return r
	}
	m["js_jsvget"] = func(a []uint64) uint64 {
		// The arguments object's `length` is its own slot (jsvIsArgs), not the storage's.
		if arr, isArr := u(a[0]).(*jsArray); isArr && arr.kind&31 == jsArgsKind && rt.toString(u(a[1])) == "length" {
			return w(float64(arr.kind >> 5))
		}
		r := vget(a)
		if !isUndefOrNull(u(r)) {
			return r
		}
		// A string's [Symbol.iterator], which the iteration sites probe under the key it
		// stands for (emitIterOf): code points, not UTF-16 units. An arguments object's
		// walks it up to its `length` (jsvSeqIter).
		if s, isStr := u(a[0]).(string); isStr && rt.toString(u(a[1])) == "__#@iterator" {
			return w(jsHostFunc("[Symbol.iterator]", func(rt *jsrt, this uint64, args []interface{}) interface{} {
				return jsvSeqIter(s, 1)
			}))
		}
		if f := jsvArgsIterFn(u(a[0]), rt.toString(u(a[1]))); f != nil {
			return w(f)
		}
		// Object.prototype's methods, off a plain object, an instance, an array, or a
		// string, number or boolean primitive (through its wrapper's prototype).
		switch recv := u(a[0]).(type) {
		case *jsObject:
			if isClassDesc(recv) || rxIsRegexObj(recv) || jsvNoObjProto(recv) {
				return r
			}
		case *jsArray, string, float64, bool:
		default:
			return r
		}
		if name := rt.toString(u(a[1])); jsvIsProtoName(name) {
			return w(jsvProtoShim(name))
		}
		return r
	}
	// js_has plus the __class methods: an instance keeps its methods on its class
	// descriptor rather than as own properties, but `"m" in instance` is true in node.
	// The layer-2 twin is js_jshas in languages/lib/js-rt.metajs; the interpreters' is
	// propIn.
	baseHas := m["js_has"]
	// The `in` operator: js_jshas, after node's TypeError for a primitive on the right
	// (ECMA-262 13.10.1). Twin: js_jsin in languages/lib/js-rt.metajs.
	m["js_jsin"] = func(a []uint64) uint64 {
		if o := u(a[0]); !jsvIsObjLike(o) && !isCallable(o) {
			rt.bigRaise("TypeError: Cannot use 'in' operator to search for '%s' in %s", rt.jsvString(u(a[1])), rt.jsvString(o))
		}
		return m["js_jshas"](a)
	}
	// `#x in o`, the brand check: the private name's key is the engine's own, handed
	// over RAW. Twin: js_jsprivin in languages/lib/js-rt.metajs.
	var hasRaw func(a []uint64) uint64
	m["js_jsprivin"] = func(a []uint64) uint64 {
		if o := u(a[0]); !jsvIsObjLike(o) && !isCallable(o) {
			k := rt.toString(u(a[1]))
			rt.bigRaise("TypeError: Cannot use 'in' operator to search for '#%s' in %s", k[strings.LastIndex(k, "#")+1:], rt.jsvString(o))
		}
		return hasRaw(a)
	}
	m["js_jshas"] = func(a []uint64) uint64 { return hasRaw(rt.jsvEscArg(a, 1)) }
	hasRaw = func(a []uint64) uint64 {
		recv, isObj := u(a[0]).(*jsObject)
		name := rt.toString(u(a[1]))
		// Every object inherits Object.prototype's names (`"toString" in {}`), an
		// Array its methods and length, a function or a class call / apply / bind /
		// name / length.
		if isObj || jsvIsFnLike(u(a[0])) {
			if jsvObjProtoName(jsvKeyUnesc(name)) && !jsvNoObjProto(u(a[0])) {
				return w(true)
			}
		}
		if arr, isArr := u(a[0]).(*jsArray); isArr && jsvIsArgs(arr) {
			if _, isIdx := jsvArrayIndex(name); isIdx {
				return w(jsvArgsHas(arr, name))
			}
		}
		if arr, isArr := u(a[0]).(*jsArray); isArr {
			if name == "length" || jsvArrayMethods[name] || jsvObjProtoName(jsvKeyUnesc(name)) {
				return w(true)
			}
			if p := jsvArrNamed(arr); p != nil {
				if _, has := p.props[name]; has {
					return w(true)
				}
			}
		}
		if cls := jsvArrClass(u(a[0])); cls != nil && jsvProtoChainHas(cls, name) {
			return w(true)
		}
		if p, isWrap := jsvWrapped(u(a[0])); isWrap {
			if ps, isStr := p.(string); isStr && rt.jsvHasOwn(ps, name) {
				return w(true)
			}
		}
		if (jsvIsFnLike(u(a[0])) || (isObj && isClassDesc(recv))) &&
			(name == "call" || name == "apply" || name == "bind" || name == "name" || name == "length") {
			return w(true)
		}
		// The __-prefixed engine slots are not properties of the program's object.
		// ...but a mangled PRIVATE name IS one of the program's own, and `#x in o` -
		// the ergonomic brand check - asks for exactly that slot, so it falls through
		// to the own-slot and class-method arms below.
		if isObj && jsvInternalKey(name) && !jsvPrivateKey(name) {
			return w(false)
		}
		// `"m" in C` asks the STATIC namespace and nothing else, and it is asked
		// BEFORE js_has: the descriptor's own property map holds C's INSTANCE
		// methods, which node does not report as properties of the constructor.
		// `name` and `prototype` are own properties of every class object.
		if isObj && isClassDesc(recv) {
			if _, _, found := jsStaticsFind(recv, name); found {
				return w(true)
			}
			return w(name == "name" || name == "prototype")
		}
		// `"constructor" in e` is true in node for every instance.
		if isObj && name == "constructor" {
			if _, found := jsvConstructorOf(recv); found {
				return w(true)
			}
		}
		// A function's own properties (jsvFnOwn), and nothing else: the shared
		// js_has aborts with "'in' needs an object on the right side" for any
		// receiver that is not an object.
		if jsvIsFnLike(u(a[0])) {
			return w(jsvFnOwn(u(a[0]), name))
		}
		if r := baseHas(a); u(r) == true {
			return r
		}
		if isObj {
			if _, found := jsvFindClassMethod(recv, name); found {
				return w(true)
			}
			// An inherited property: an instance's chain starts at its class, a
			// prototype view's at the descriptor whose declared members are its own.
			from := recv.props["__class"]
			if d, isView := recv.props["__protoof"].(*jsObject); isView {
				from = d
			}
			return w(jsvProtoChainHas(from, name))
		}
		return w(false)
	}
	m["js_jsmget"] = func(a []uint64) uint64 {
		if jsvHidesMember(u(a[0]), rt.toString(u(a[1]))) {
			return jsHUndefined
		}
		if v, isAcc := rt.jsvArrAccRead(u(a[0]), rt.toString(u(a[1]))); isAcc {
			return w(v)
		}
		if f := jsvArgsIterFn(u(a[0]), rt.toString(u(a[1]))); f != nil {
			return w(f)
		}
		// The static namespace, exactly as in js_jsvget above.
		if cls, isStatic := jsvIsStaticSide(u(a[0]), rt.toString(u(a[1]))); isStatic {
			return rt.jsvStaticSideRead(cls, rt.toString(u(a[1])))
		}
		// then/catch/finally READ AS A VALUE. Promise method dispatch is
		// table-based (jsvMethod -> promMethod), so a promise has no own slot of
		// that name. Answer a receiver-BOUND host function, which is what makes
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
		// `f.name` and `f.length` - see js_jsvget above.
		if v := jsvFnIntrinsic(u(a[0]), rt.toString(u(a[1]))); v != nil {
			return w(v)
		}
		// An own property written onto a FUNCTION lives in the side table, at the end
		// so a program that never writes one pays one map-length test.
		if p := jsvFnPropsOf(u(a[0])); p != nil {
			if v, found := p.props[rt.toString(u(a[1]))]; found {
				return w(v)
			}
		}
		// ...and a function's Object.prototype surface. This is the arm the CALL site
		// reaches: the emitter reads the member with js_jsmget and calls the result
		// when it is a function, so one body serves `f.hasOwnProperty("p")` and
		// `typeof f.toString`.
		if m := rt.jsvFnMethod(u(a[0]), rt.toString(u(a[1]))); m != nil {
			return w(m)
		}
		return baseGet(a)
	}

	// The static table of one class, built at its definition.
	// The layer-2 twin also counts the classes, so that a program with none pays a
	// single integer compare per member read instead of a hash lookup; the Go twin
	// has no such loop to protect and only has to answer the same object.
	// js_jssupget is `super.x` READ AS A VALUE for js/typescript, and it exists only
	// because of the shim in its callable arm. The shared js_supget (abnf/jsrt.go)
	// hands back a host function that IGNORES the receiver of the eventual call and
	// always re-enters with the receiver the READ was made with, so
	// `super.m.call(x)` would run the parent's method on `this` instead of on x.
	// that in js_supget would change it for the eight other languages that emit it,
	// and the extern map is ONE map for every language (manual chapter 4:
	// overriding a shared name in a language's own twin file leaks across
	// languages), so js/ts get a name of their own - the same pattern as js_jvsub /
	// js_csmcall / js_ktsmcall. Everything else here is js_supget's own behaviour,
	// deliberately: the static side first when the receiver is a class descriptor,
	// then the __super walk, and a miss is undefined rather than a read of the
	// receiver. The layer-2 twin is js_jssupget in languages/lib/js-rt.metajs.
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
		// The chain ends at Object.prototype, so its methods answer last:
		// `typeof super.toString` is "function" in a class with no toString above
		// it. Twins: js_jssupget in languages/lib/js-rt.metajs, superGet in both
		// *-interpreter.abnf.
		if o, isObj := self.(*jsObject); isObj && !isClassDesc(o) && jsvIsProtoName(name) {
			return w(jsvProtoShim(name))
		}
		return jsHUndefined
	}
	// super.m(args) for one of Object.prototype's methods that nothing on the
	// __super chain defines (js_supercall answered its miss sentinel): the method
	// itself, on the receiver - `super.toString()` is "[object Object]". The emitter
	// calls it only for those names. Twins: js_jssupproto in
	// languages/lib/js-rt.metajs, makeSuperCall in both *-interpreter.abnf.
	m["js_jssupproto"] = func(a []uint64) uint64 { // (this, method name, args array)
		args, _ := u(a[2]).(*jsArray)
		if args == nil {
			args = &jsArray{}
		}
		name := rt.toString(u(a[1]))
		// An Array subclass's super.m() reaches Array.prototype, a Promise / RegExp /
		// wrapper subclass's the builtin's (hostRoot); any other name that is not
		// Object.prototype's answers the miss sentinel for the call site.
		if arr, isArr := u(a[0]).(*jsArray); isArr {
			if v, handled := rt.jsvArrBuiltin(arr, name, args.elems); handled {
				return w(v)
			}
		}
		if o, isObj := u(a[0]).(*jsObject); isObj && o.props["__class"] != nil {
			if p, isWrap := o.props["__bvof"]; isWrap {
				return m["js_jsmcall"]([]uint64{w(p), a[1], a[2]})
			}
			if rxIsRegexObj(o) {
				if v, handled := rt.jsxMethod(o, name, args.elems); handled {
					return w(v)
				}
			}
			if promIs(o) {
				if v, handled := rt.promMethod(o, name, args.elems); handled {
					return w(v)
				}
			}
		}
		if !jsvIsProtoName(name) {
			if v, ok := rt.mcallMiss(); ok {
				return w(v)
			}
		}
		return w(rt.jsvProtoApply(u(a[0]), name, args.elems))
	}

	// js_jskeys is js_keys plus the class-descriptor arm. The floor's js_keys is
	// shared by sixteen languages and answers an object's own key map, which for a
	// class descriptor is its INSTANCE methods, where node gives the static FIELDS
	// (`{...C}`, `for (k in C)`). A name of its own rather than an override, for
	// manual chapter 4's reason. The layer-2 twin is js_jskeys in
	// languages/lib/js-rt.metajs.
	// It also has to be TOTAL, which the floor's js_keys is not: `for (i in [7,8])`
	// and `{...[7,8]}` both reach here with an ARRAY, where node answers the INDICES
	// as strings ("0","1"). A string enumerates the same way, and
	// null/undefined/a number/a boolean enumerate NOTHING rather than raising.
	baseKeys := m["js_keys"]
	var jsKeysAll func(a []uint64) uint64
	unescKeys := func(r uint64) uint64 {
		if arr, ok := u(r).(*jsArray); ok {
			for i, k := range arr.elems {
				if ks, isStr := k.(string); isStr && len(ks) > 0 && ks[0] == 1 {
					arr.elems[i] = ks[1:]
				}
			}
		}
		return r
	}
	m["js_jskeys"] = func(a []uint64) uint64 { return unescKeys(jsKeysAll(a)) }
	// for-in's key list: js_jskeys, then the enumerable keys inherited through the
	// prototype chain (EnumerateObjectProperties, ECMA-262 14.7.5.9). The walk starts at
	// an instance's class, or at a prototype view's own descriptor, whose declared
	// methods shadow what its parent's view holds.
	// Twins: js_jsforinkeys in languages/lib/js-rt.metajs, jsForInKeys in both
	// *-interpreter.abnf.
	m["js_jsforinkeys"] = func(a []uint64) uint64 {
		r := jsKeysAll(a)
		if o, isObj := u(a[0]).(*jsObject); isObj {
			if isClassDesc(o) {
				if arr, ok := u(r).(*jsArray); ok {
					arr.elems = jsvStaticForInKeys(o, arr.elems)
				}
				return unescKeys(r)
			}
			from := o.props["__class"]
			if d, isView := o.props["__protoof"].(*jsObject); isView {
				from = d
			}
			if arr, ok := u(r).(*jsArray); ok && from != nil {
				arr.elems = jsvInheritedKeys(from, arr.elems)
			}
		}
		return unescKeys(r)
	}
	jsKeysAll = func(a []uint64) uint64 {
		switch v := u(a[0]).(type) {
		case *jsObject:
			if isClassDesc(v) {
				out := &jsArray{}
				for _, k := range jsvStaticEnumKeys(v) {
					out.elems = append(out.elems, k)
				}
				return w(out)
			}
			_, marked := v.props["__errh"]
			_, hid := v.props["__hidden"]
			_, strWrap := v.props["__bvof"].(string)
			if _, isView := v.props["__protoof"]; marked || isView || hid || strWrap {
				out := &jsArray{}
				for _, k := range jsvOwnKeys(v) {
					out.elems = append(out.elems, k)
				}
				return w(out)
			}
			r := baseKeys(a)
			if arr, ok := u(r).(*jsArray); ok {
				ks := make([]string, len(arr.elems))
				for i, k := range arr.elems {
					ks[i], _ = k.(string)
				}
				for i, k := range jsvIndexOrder(ks) {
					arr.elems[i] = k
				}
			}
			return r
		case *jsArray:
			if jsvIsArgs(v) {
				out := &jsArray{}
				for _, k := range jsvArgsKeys(v) {
					out.elems = append(out.elems, k)
				}
				return w(out)
			}
			out := &jsArray{}
			for i, e := range v.elems {
				if e != jsHole {
					out.elems = append(out.elems, strconv.Itoa(i))
				}
			}
			if p := jsvArrNamed(v); p != nil {
				for _, k := range jsvOwnKeys(p) {
					out.elems = append(out.elems, k)
				}
			}
			return w(out)
		case string:
			// strLen, not len(v): a string is indexed in UTF-16 CODE UNITS here.
			return w(jsvIndexKeys(rt.strLen(v)))
		default:
			// A FUNCTION enumerates the own properties written onto it.
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
	// js_jssfield in languages/lib/js-rt.metajs.
	// js_jstagstrs(cooked, raw) makes a tagged template's strings array out of its
	// cooked and raw texts: cooked, frozen, with raw, frozen too, as its own
	// NON-ENUMERABLE `raw` property (ECMA-262 13.2.8.4 GetTemplateObject). Twins:
	// js_jstagstrs in languages/lib/js-rt.metajs, tagArgs in both *-interpreter.abnf.
	m["js_jstagstrs"] = func(a []uint64) uint64 {
		cooked, ok := u(a[0]).(*jsArray)
		if !ok {
			rt.fail("js_jstagstrs needs an array")
		}
		rt.setMember(cooked, "raw", u(a[1]))
		jsvAttrSet(jsvArrNamed(cooked), "__hidden", "raw", true)
		rt.jsvSetIntegrity(u(a[1]), 3)
		rt.jsvSetIntegrity(cooked, 3)
		return a[0]
	}
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

	// js_jsmset is emitted in BOTH halves. Under -exe layer 2 unwraps the accessor box
	// the C floor knows nothing about; here a member WRITE on a class descriptor is a
	// write to the STATIC namespace (`C.zz = 1` then `C.zz`) - putting it in the
	// descriptor's own property map would put it where the instance methods live.
	// Everything that is not a class descriptor - which is every write in a program
	// with no classes in it - goes straight to js_set.
	baseSet := m["js_set"]
	m["js_jsmset"] = func(a []uint64) uint64 {
		if cls, isStatic := jsvIsStaticSide(u(a[0]), rt.toString(u(a[1]))); isStatic {
			name := rt.toString(u(a[1]))
			// A class's own `name` and `length` are non-writable, unless the class
			// body declares a static member of that name. Twin of setMember's static
			// arm in both *-interpreter.abnf.
			if name == "name" || name == "length" {
				if !jsvOwnsStatic(cls, name) {
					jsvRefused = rt.jsvFnRoText(cls, name)
					return boolH(false)
				}
			}
			if h := rt.jsvStaticWrite(cls, name, u(a[2])); h != nil {
				jsvRefused = jsvGetterOnlyText(name, jsvElide(rt.jsvString(h)))
				return boolH(false)
			}
			return 0
		}
		// `name` and `length` are NON-WRITABLE: node drops `f.name = "q"` in sloppy
		// code and raises a TypeError in strict code - unless a definition made one
		// writable (jsvFnIntrRec). A deleted one is refused too: the write then meets
		// Function.prototype's, which is read-only as well.
		if jsvIsFnLike(u(a[0])) {
			if n := rt.toString(u(a[1])); n == "name" || n == "length" {
				if r := jsvFnIntrRec(u(a[0]), n); r != nil && r.props["w"] == true {
					r.set("v", u(a[2]))
					return 0
				}
				jsvRefused = rt.jsvFnRoText(u(a[0]), n)
				return boolH(false)
			}
		}
		// A write onto a FUNCTION goes to the side table: neither *jsClosure nor
		// *hostFunc has a property map.
		// A write the attributes or the integrity level refuse answers false, and
		// js_jsrefuse raises it in strict code (jsvWriteRefusal).
		if jsvAttrN > 0 {
			t := u(a[0])
			if p := jsvFnPropsOf(t); p != nil {
				t = p
			}
			if msg := rt.jsvWriteRefusal(t, rt.toString(u(a[1])), u(a[2])); msg != "" {
				jsvRefused = msg
				return boolH(false)
			}
		}
		if p := jsvFnPropsMake(u(a[0])); p != nil {
			p.set(rt.toString(u(a[1])), u(a[2]))
			return 0
		}
		if o, isObj := u(a[0]).(*jsObject); isObj {
			// A setter found on o or its chain runs with o as `this`; an accessor with no
			// setter refuses the write, and js_jsrefuse raises it in strict code.
			if rt.accessorCount > 0 {
				name := rt.toString(u(a[1]))
				if h, acc := jsvAccHolder(o, name); acc != nil {
					if acc.set == nil {
						jsvRefused = jsvGetterOnlyText(name, rt.jsvAccHolderText(o, h))
						return boolH(false)
					}
					rt.call(acc.set, o, []interface{}{u(a[2])})
					return 0
				}
			}
			if rt.toString(u(a[1])) == "name" {
				jsvErrNameWrite(o)
			}
			if d, isView := o.props["__protoof"].(*jsObject); isView {
				protoWrite(d, rt.toString(u(a[1])), u(a[2]))
			}
		}
		// An arguments object's `length` is an ordinary property, kept in its kind word
		// (jsvIsArgs): the elements stay as they are. A value that is no array length
		// resizes the storage as an array write does, and the word follows.
		arr, isArr := u(a[0]).(*jsArray)
		argsLen := isArr && arr.kind&31 == jsArgsKind && rt.toString(u(a[1])) == "length"
		if argsLen {
			if n, isNum := u(a[2]).(float64); isNum && n >= 0 && n == math.Floor(n) && n < 67108864 {
				arr.kind = jsvArgsKind(int(n))
				return 0
			}
		}
		// A named accessor of an array runs its setter, with the array as `this`.
		if acc := jsvArrAccessor(u(a[0]), rt.toString(u(a[1]))); acc != nil {
			if acc.set == nil {
				jsvRefused = jsvGetterOnlyText(rt.toString(u(a[1])), "[object Array]")
				return boolH(false)
			}
			rt.call(acc.set, u(a[0]), []interface{}{u(a[2])})
			return 0
		}
		r := baseSet(a)
		if argsLen {
			arr.kind = jsvArgsKind(len(arr.elems))
		}
		return r
	}

	// (closure, name, length) -> closure. The emitter calls it at the creation site of
	// every function that HAS a name or a parameter, which is python's js_pyfnname
	// pattern exactly. A repeat for the same value overwrites.
	m["js_jsfnmeta"] = func(a []uint64) uint64 {
		if v := u(a[0]); jsvIsFnLike(v) {
			if rec, ok := u(a[1]).(*jsObject); ok {
				jsvFnMetaSet(v, "", int(rt.toNumber(u(a[2]))))
				jsvFnRecSet(v, rec)
			} else {
				jsvFnMetaSet(v, rt.toString(u(a[1])), int(rt.toNumber(u(a[2]))))
			}
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
	// The TypeError of the write or delete the integrity marks last refused, which
	// the emitter raises after a refused js_jsmset / js_jsdel in strict code. Twin:
	// js_jsrefuse in languages/lib/js-rt.metajs.
	m["js_jsrefuse"] = func(a []uint64) uint64 {
		if msg := jsvRefused; msg != "" {
			jsvRefused = ""
			rt.bigRaise("TypeError: %s", msg)
		}
		return 0
	}
	m["js_jspromfn"] = func(a []uint64) uint64 {
		p := rt.promiseGlobal()
		jsvFnMetaSet(p, "Promise", 1)
		rt.jsvGlobalSet("Promise", p)
		return w(p)
	}
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
	// order, so js_jsxmcall (abnf/jsrtregexjs.go) does not exist yet at this point
	// and capturing it here would bind nil.
	baseMcall := m["js_mcall"]
	// Object.getPrototypeOf(v). An object's prototype is the view of its __class
	// descriptor, and a prototype view's is its parent class's view; without either
	// it is Object.prototype, and null for a null-prototype object and for
	// Object.prototype itself. A primitive answers its builtin's prototype, a wrapper
	// its primitive's, a function and a base class Function.prototype, and a derived
	// class the class it extends. Twins: jsGetProto in both *-interpreter.abnf,
	// jvGetProto in languages/lib/js-rt.metajs.
	getProto := func(objGlobal interface{}, v interface{}) interface{} {
		switch x := v.(type) {
		case string:
			return builtinProtoNamed("String")
		case float64:
			return builtinProtoNamed("Number")
		case bool:
			return builtinProtoNamed("Boolean")
		case *jsBigInt:
			return builtinProtoNamed("BigInt")
		case *jsArray:
			if cls := jsvArrClass(x); cls != nil {
				return protoView(cls)
			}
			if jsvIsArgs(x) || interface{}(x) == builtinProtoNamed("Array") {
				return objProto
			}
			return builtinProtoNamed("Array")
		case *jsObject:
			if isClassDesc(x) {
				sup := x.props["__super"]
				if objectRoot != nil && sup == interface{}(objectRoot) {
					return objGlobal
				}
				if so, ok := sup.(*jsObject); ok {
					if fn, has := so.props["__ctorfn"]; has {
						return fn
					}
					return sup
				}
				return builtinProtoNamed("Function")
			}
			if x.props["__objproto"] != nil || x.props["__nullproto"] == true {
				return jsNull
			}
			if p, isWrap := x.props["__bvof"]; isWrap {
				if d, isInst := x.props["__class"].(*jsObject); isInst {
					return protoView(d)
				}
				if wp := builtinProtoNamed(jsvWrapCtor(p)); wp != interface{}(x) {
					return wp
				}
				return objProto
			}
			if d, isView := x.props["__protoof"].(*jsObject); isView {
				sd, ok := d.props["__super"].(*jsObject)
				if !ok || sd == objectRoot {
					return objProto
				}
				// A class extending a builtin: the builtin's prototype is the next one up.
				if b, isHost := sd.props["__ctorfn"].(*hostFunc); isHost {
					if bp := builtinProto(b); bp != nil {
						return bp
					}
				}
				return protoView(sd)
			}
			if d, ok := x.props["__class"].(*jsObject); ok {
				return protoView(d)
			}
			return objProto
		}
		if isUndefOrNull(v) {
			rt.bigRaise("TypeError: Cannot convert undefined or null to object")
		}
		if isCallable(v) {
			if fp := builtinProtoNamed("Function"); fp != v {
				return fp
			}
			return objProto
		}
		return jsUndef
	}
	m["js_jsmcall"] = func(a []uint64) uint64 {
		arr, ok := u(a[2]).(*jsArray)
		if !ok {
			rt.fail("js_jsmcall args must be an array")
		}
		// An Array subclass instance's own methods win over the Array builtins, and
		// over an integrity level's refusal of a builtin of the same name.
		if cls := jsvArrClass(u(a[0])); cls != nil {
			if v, handled := rt.jsvClassCall(u(a[0]), cls, rt.toString(u(a[1])), arr.elems); handled {
				return w(v)
			}
		} else if o, isObj := u(a[0]).(*jsObject); isObj && len(hostRoots) > 0 && (promIs(o) || rxIsRegexObj(o) || o.props["__bvof"] != nil) {
			// A Promise / RegExp / wrapper subclass instance (hostRoot): the same order.
			if cls, isInst := o.props["__class"].(*jsObject); isInst {
				if v, handled := rt.jsvClassCall(o, cls, rt.toString(u(a[1])), arr.elems); handled {
					return w(v)
				}
			}
		}
		if jsvAttrN > 0 {
			name := rt.toString(u(a[1]))
			if t, isArr := u(a[0]).(*jsArray); isArr {
				if lv := jsvArrLevel(t); lv > 0 {
					if msg := rt.jsvArrMethodRefusal(t, name, arr.elems, lv); msg != "" {
						rt.bigRaise("TypeError: %s", msg)
					}
				}
			}
			if o, isObj := argAt(arr.elems, 0).(*jsObject); isObj && name == "setPrototypeOf" && jsvIsObjectGlobal(u(a[0])) &&
				o.props["__noext"] == true && getProto(u(a[0]), o) != argAt(arr.elems, 1) {
				rt.bigRaise("TypeError: %s is not extensible", rt.jsvObjText(o))
			}
		}
		if jsvIsObjectGlobal(u(a[0])) {
			switch name := rt.toString(u(a[1])); name {
			case "getPrototypeOf":
				return w(getProto(u(a[0]), argAt(arr.elems, 0)))
			case "getOwnPropertyDescriptor", "getOwnPropertyDescriptors":
				return w(rt.jsvOwnDescs(name, arr.elems, func(v interface{}, k string) interface{} {
					return u(m["js_jsvget"]([]uint64{w(v), w(k)}))
				}))
			}
		}
		if t, isArr := u(a[0]).(*jsArray); isArr && jsvArrClass(t) != nil {
			if v, handled := rt.jsvArrBuiltin(t, rt.toString(u(a[1])), arr.elems); handled {
				return w(v)
			}
		}
		if v, handled := rt.jsvMethod(u(a[0]), rt.toString(u(a[1])), arr.elems); handled {
			return w(v)
		}
		// A CLASS DESCRIPTOR stops here. js_jsmget already
		// answered every static, so reaching this line means the name is not one -
		// and the shared js_mcall below would find the descriptor's own property of
		// that name, which is its INSTANCE method. Answer the miss sentinel so the
		// call site raises node's "C.m is not a function" instead. Object.prototype's
		// methods are the exception: a class object inherits them too.
		if _, isStatic := jsvIsStaticSide(u(a[0]), rt.toString(u(a[1]))); isStatic {
			if name := rt.toString(u(a[1])); jsvIsProtoName(name) {
				return w(rt.jsvProtoApply(u(a[0]), name, arr.elems))
			}
			// A class extending a builtin inherits the builtin's statics (`V.isArray(x)`):
			// the root descriptor arrayRoot / hostRoot files the builtin as __ctorfn.
			// Twins: jsBuiltinBase in both *-interpreter.abnf, jvBuiltinBase in
			// languages/lib/js-rt.metajs.
			c, _ := u(a[0]).(*jsObject)
			for guard := 0; c != nil && guard < 64; guard++ {
				if b, isHost := c.props["__ctorfn"].(*hostFunc); isHost {
					// Promise's statics make the subclass's instance (promStatic).
					if rt.promFn != nil && b == rt.promFn {
						if v, handled := rt.promStatic(u(a[0]).(*jsObject), rt.toString(u(a[1])), arr.elems); handled {
							return w(v)
						}
					}
					return m["js_jsmcall"]([]uint64{w(b), a[1], a[2]})
				}
				c, _ = c.props["__super"].(*jsObject)
			}
			if v, ok := rt.mcallMiss(); ok {
				return w(v)
			}
		}
		if o, isObj := u(a[0]).(*jsObject); isObj {
			if fn, found := jsvViewFn(o, rt.toString(u(a[1]))); found {
				return w(rt.call(fn, o, arr.elems))
			}
		}
		// Object.prototype's methods on a receiver with no member of that name
		// (jsvProtoApply). A RegExp is left to js_jsxmcall below, which answers its
		// toString. Twins: jvFallbackCall in languages/lib/js-rt.metajs, mcall in both
		// *-interpreter.abnf.
		// (An Array's valueOf reads as a builtin value - jsvBuiltinFam - and is still
		// Object.prototype's here.)
		if name := rt.toString(u(a[1])); jsvIsProtoName(name) && !isUndefOrNull(u(a[0])) && !rxIsRegexObj(u(a[0])) &&
			!jsvProtoLocaleOwn(u(a[0]), name) && !jsvNoObjProto(u(a[0])) &&
			(jsvBuiltinFam(u(a[0]), name) != 0 || isUndefOrNull(u(vget([]uint64{a[0], a[1]})))) {
			return w(rt.jsvProtoApply(u(a[0]), name, arr.elems))
		}
		// The arguments object has no Array method: the miss, so the call site raises
		// node's TypeError.
		if jsvIsArgs(u(a[0])) && jsvArrayMethods[rt.toString(u(a[1]))] {
			if v, ok := rt.mcallMiss(); ok {
				return w(v)
			}
		}
		if next := m["js_jsxmcall"]; next != nil {
			return next(a)
		}
		return baseMcall(a)
	}

	// ----- the Error hierarchy ---------------------------------------------
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
	//                 them here rather than with memSets in the emitter keeps
	//                 the hierarchy's fixed IR cost per module small;
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
		// of whether the program declares a class of its own.
		jsStaticsOf(cls, true)
		if name == jsErrNames[0] {
			// toString lives on the ROOT only, exactly as Error.prototype does:
			// the __class/__super walk finds it from any of the builtin classes and from
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
		// `new Error()` and `new Error(undefined)` both leave message "": the
		// spec only assigns it when the argument is not undefined. null is NOT
		// undefined, so `new Error(null).message` is "null". Without the
		// assignment message is not an OWN property either (__errnm).
		if v := u(a[1]); v != interface{}(jsUndef) {
			rt.jsErrInit(self, rt.jsvString(v))
		} else {
			rt.jsErrInit(self, "")
			self.props["__errnm"] = true
		}
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

	// A symbol KEY becomes its __symkey before any member extern sees it.
	symKey := func(v interface{}) (string, bool) {
		if o, ok := v.(*jsObject); ok {
			if k, isSym := o.props["__symkey"].(string); isSym {
				return k, true
			}
		}
		return "", false
	}
	keyed := func(name string) {
		inner := m[name]
		m[name] = func(a []uint64) uint64 {
			if k, isSym := symKey(u(a[1])); isSym {
				// [Symbol.iterator] of an array or a string, read as a value (bound to
				// the receiver) or called; a generator's answers the generator.
				if recv := u(a[0]); k == "__#@iterator" {
					_, isArr := recv.(*jsArray)
					_, isStr := recv.(string)
					if isArr || isStr {
						switch name {
						case "js_jsvget", "js_jsmget":
							return w(jsHostFunc("[Symbol.iterator]", func(rt *jsrt, this uint64, args []interface{}) interface{} {
								return jsvSeqIter(recv, 1)
							}))
						case "js_jsmcall":
							return w(jsvSeqIter(recv, 1))
						}
					}
					if _, isGen := recv.(*jsGenerator); isGen {
						switch name {
						case "js_jsvget", "js_jsmget":
							return w(jsHostFunc("[Symbol.iterator]", func(rt *jsrt, this uint64, args []interface{}) interface{} {
								return recv
							}))
						case "js_jsmcall":
							return w(recv)
						}
					}
				}
				b := append([]uint64{}, a...)
				b[1] = rt.wrapStr(k)
				// js_jshas escapes a program's key; a symbol's is a stored key already.
				if name == "js_jshas" {
					return hasRaw(b)
				}
				return inner(b)
			}
			return inner(a)
		}
	}
	keyed("js_jsvget")
	keyed("js_jsmget")
	keyed("js_jsmset")
	keyed("js_jshas")
	keyed("js_jsdel")
	keyed("js_jsmcall")
	// A computed member key, escaped (jsvKeyEsc) before it reaches a member extern.
	m["js_jskeyesc"] = func(a []uint64) uint64 { return rt.jsvEscArg(a, 0)[0] }
	// A class member's or an accessor's computed key, which reaches no member
	// extern: a symbol's key, or the escaped string (jsvKeyEsc).
	m["js_jspropkey"] = func(a []uint64) uint64 {
		if k, isSym := symKey(u(a[0])); isSym {
			return rt.wrapStr(k)
		}
		return rt.jsvEscArg(a, 0)[0]
	}
	// Symbol.for / Symbol.keyFor, and a symbol's toString / valueOf.
	symCall := m["js_jsmcall"]
	m["js_jsmcall"] = func(a []uint64) uint64 {
		name := rt.toString(u(a[1]))
		args, _ := u(a[2]).(*jsArray)
		arg0 := interface{}(jsUndef)
		if args != nil && len(args.elems) > 0 {
			arg0 = args.elems[0]
		}
		if u(a[0]) == interface{}(symbolFn) {
			switch name {
			case "for":
				k := rt.jsvString(arg0)
				s, found := symReg[k]
				if !found {
					s = symMake(k, "__#@for@"+k)
					symReg[k] = s
				}
				return w(s)
			case "keyFor":
				s, isObj := arg0.(*jsObject)
				if _, isSym := symKey(arg0); !isSym {
					panic(&jsThrown{value: rt.jsErrFromText("TypeError: " + rt.jsvString(arg0) + " is not a symbol")})
				}
				if k, _ := symKey(s); strings.HasPrefix(k, "__#@for@") && isObj {
					return w(s.props["__symdesc"])
				}
				return jsHUndefined
			}
		}
		// Object.getOwnPropertySymbols: the symbols of the own symbol keys, in the order
		// they were first written.
		if name == "getOwnPropertySymbols" && jsvIsObjectGlobal(u(a[0])) {
			if isUndefOrNull(arg0) {
				rt.bigRaise("TypeError: Cannot convert undefined or null to object")
			}
			out := &jsArray{}
			for _, k := range jsvSymKeys(arg0) {
				out.elems = append(out.elems, symByKey[k])
			}
			return w(out)
		}
		if _, isSym := symKey(u(a[0])); isSym {
			switch name {
			case "toString":
				return rt.wrapStr(rt.jsvString(u(a[0])))
			case "valueOf":
				return a[0]
			}
		}
		return symCall(a)
	}

	// `with`: a name inside a with body resolves innermost first - a scope's own
	// binding, else the object a scope holds under "0with" when it has the property,
	// else outward. withObj answers that object, nil when an ordinary binding (or
	// nothing) resolves the name, which then goes the ordinary way; withRecv is the
	// object the last js_jswithget read from, which a call of that name takes as
	// `this`. Twins: withGet / withSet / withRecv in js-interpreter.abnf, js_jswith*
	// in languages/lib/js-rt.metajs.
	var withRecv interface{} = jsUndef
	withObj := func(sc *jsScope, name string) interface{} {
		for s := sc; s != nil; s = s.parent {
			if s.has(name) {
				return nil
			}
			if o, ok := s.get("0with"); ok {
				if jsvIsObjLike(o) || isCallable(o) {
					if u(m["js_jshas"]([]uint64{w(o), rt.wrapStr(name)})) == true {
						return o
					}
				} else if !isUndefOrNull(u(m["js_jsvget"]([]uint64{w(o), rt.wrapStr(name)}))) {
					return o
				}
			}
		}
		return nil
	}
	m["js_jswithget"] = func(a []uint64) uint64 {
		withRecv = jsUndef
		if o := withObj(rt.scopeOf(a[0]), rt.toString(u(a[1]))); o != nil {
			withRecv = o
			return m["js_jsvget"]([]uint64{w(o), a[1]})
		}
		return m["js_scope_get"](a)
	}
	m["js_jswithset"] = func(a []uint64) uint64 {
		if o := withObj(rt.scopeOf(a[0]), rt.toString(u(a[1]))); o != nil {
			return m["js_jsmset"]([]uint64{w(o), a[1], a[2]})
		}
		return m["js_scope_set_or_create"](a)
	}
	m["js_jswithrecv"] = func(a []uint64) uint64 { return w(withRecv) }
	// An assignment to a const binding: the scope the write reaches carries the
	// "0const:" marker beside the name. Twins: js_jsconstchk in
	// languages/lib/js-rt.metajs, jsConstChk in js-interpreter.abnf.
	m["js_jsconstchk"] = func(a []uint64) uint64 {
		name := rt.toString(u(a[1]))
		for sc := rt.scopeOf(a[0]); sc != nil; sc = sc.parent {
			if sc.has(name) {
				if sc.has("0const:" + name) {
					rt.bigRaise("TypeError: Assignment to constant variable.")
				}
				break
			}
		}
		return w(jsUndef)
	}
	m["js_jswithtypeof"] = func(a []uint64) uint64 {
		if o := withObj(rt.scopeOf(a[0]), rt.toString(u(a[1]))); o != nil {
			return m["js_jstypeof"]([]uint64{m["js_jsvget"]([]uint64{w(o), a[1]})})
		}
		if tv := m["js_scope_typeof"](a); rt.toString(u(tv)) == "undefined" {
			return tv
		}
		return m["js_jstypeof"]([]uint64{m["js_scope_get"](a)})
	}

	// BigInt last: its wrappers sit in FRONT of the js_js* names registered above
	// (js_jsadd, js_jseq, js_jslt, ...), so they have to exist first.
	rt.addJSONExterns(m)
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

// jsvErrOwn answers whether k is an OWN property of an Error instance, for the keys
// jsErrInit hides: node keeps `stack` and (when the constructor was given one)
// `message` on the instance, and `name` on the prototype until the program writes
// it. ok is false for any other key or object. Twins: jvErrOwn in
// languages/lib/js-rt.metajs, jsErrOwn in both *-interpreter.abnf.
func jsvErrOwn(o *jsObject, k string) (own bool, ok bool) {
	h, marked := o.props["__errh"]
	if !marked {
		return false, false
	}
	switch k {
	case "stack":
		return true, true
	case "message":
		return o.props["__errnm"] != true, true
	case "name":
		return h != true, true
	}
	return false, false
}

// jsvErrGopn is Object.getOwnPropertyNames of an Error instance: node's own
// non-enumerable `stack` and `message` first, then the enumerable own keys.
func jsvErrGopn(o *jsObject, ks []string) []string {
	if _, marked := o.props["__errh"]; !marked {
		return ks
	}
	out := []string{"stack"}
	if o.props["__errnm"] != true {
		out = append(out, "message")
	}
	return append(out, ks...)
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
// message text, so the whole routing is this one split - and a text naming no
// class of ours is returned exactly as it was, a thrown string.
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
// Async generators
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
// construction. A throw before the body has ever run does NOT start it
// (st.started), which is node's suspendedStart rule; one that reaches a `yield*`
// is forwarded to the delegate's throw() by the emitted yield* loop.

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
	// An async generator is an async iterable whose [Symbol.asyncIterator]() is itself.
	ag.set("__#@asyncIterator", jsHostFunc("[Symbol.asyncIterator]", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return ag
	}))
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
