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

// jsvString is String(v). An object with a toString of its own answers with it (the
// method convention prepends the receiver, exactly as rt.memberCall does).
func (rt *jsrt) jsvString(v interface{}) string {
	switch t := v.(type) {
	case *jsObject:
		if r, ok := rt.jsvCallMethod(t, "toString"); ok {
			return rt.toString(r)
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
			rt.fail("toString() radix must be between 2 and 36")
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
		if root, ok := rt.root.get("Object"); ok {
			if ro, isObj := root.(*jsObject); isObj {
				if p, has := ro.props["prototype"]; has {
					o.set("prototype", p)
				}
			}
		}
		return w(o)
	}

	// js_jsmget hides the String/Array method names from js_get, so that the CALL
	// site falls into js_jsmcall below instead of the shared boundMethod path, whose
	// semantics differ for several of them.
	baseGet := m["js_get"]
	m["js_jsmget"] = func(a []uint64) uint64 {
		if jsvHidesMember(u(a[0]), rt.toString(u(a[1]))) {
			return jsHUndefined
		}
		return baseGet(a)
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
}
