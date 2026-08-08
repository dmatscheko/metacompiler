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
		out := []string{}
		for _, k := range o.keys {
			if !jsvInternalKey(k) {
				out = append(out, k)
			}
		}
		return out
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
	return []string{}
}

func jsvInternalKey(k string) bool {
	return len(k) >= 2 && k[0] == '_' && k[1] == '_'
}

func (rt *jsrt) jsvGetKey(v interface{}, k string) interface{} {
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
		return jsHostFunc("bound", func(rt *jsrt, this uint64, later []interface{}) interface{} {
			return rt.call(fn, self, append(append([]interface{}{}, pre...), later...))
		}), true
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
			for _, k := range jsvOwnKeys(argAt(args, 0)) {
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
	baseGet := m["js_get"]
	// js_jsvget is the VALUE read of a member - `const f = p.m`, `typeof p.m` - as
	// opposed to js_jsmget, which the emitter uses at a member-followed-by-a-call site
	// and which deliberately HIDES the String/Array method names so the call falls into
	// js_jsmcall. The difference this extern exists for: an instance's methods live on
	// its __class descriptor, not as own properties, so `typeof p.m` answered
	// "undefined" in all four engines where node says "function". The answer is a
	// receiver-BOUND host function that re-enters the method with the receiver
	// PREPENDED, which is the calling convention a __class method has here (see
	// jsvCallMethod). The layer-2 twin is js_jsvget in languages/lib/js-rt.metajs and
	// the interpreters' is getMember's class-method arm. docs/todo.md 2.5.
	m["js_jsvget"] = func(a []uint64) uint64 {
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
			return r
		}
		name := rt.toString(u(a[1]))
		if jsvInternalKey(name) {
			return r
		}
		if fn, found := jsvFindClassMethod(recv, name); found {
			bound := fn
			return w(jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
				return rt.call(bound, jsUndef, append([]interface{}{recv}, args...))
			}))
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
		if isObj && jsvInternalKey(name) {
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
		return baseGet(a)
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
	// NAME / MESSAGE / STACK ARE ENUMERABLE HERE AND NON-ENUMERABLE IN A REAL
	// ENGINE, and that is a value-model ceiling rather than a slip:
	// Object.keys(new Error("x")) is [] in node and ["name","message","stack"] in
	// all four engines here. The floor's js_keys IS for-in and object spread, a
	// layer-2 object has exactly one key list - the floor's - and no way to mark
	// an entry hidden; the only alternative was three more key comparisons on
	// js_jsmget's hot path, which that function's own measurements (+21% to +43%
	// for one added test) rule out. So set(), not props[]: all four engines were
	// made to AGREE and to diverge from node together, which is the one thing
	// this project's byte-identity gates can still see.
	name := jsErrNameOf(self)
	self.set("name", name)
	self.set("message", msg)
	self.set("stack", jsErrText(name, msg))
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
