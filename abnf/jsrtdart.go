package abnf

// jsrtdart.go -- the Dart half of the value model: Object.toString() rendering and
// const canonicalization.
//
// This is the Go twin of the `dstr` / `dcanon` pair in
// languages/dart-interpreter.abnf. Only languages/dart-to-llvm-ir.abnf emits these
// externs, and nothing here rebinds an existing one: the registrar below only ADDS
// js_dart* names, so every other grammar emits and runs exactly as before.
//
// Why it has to exist at all. The compiler half used to hand a RAW handle to the
// shared println host function, so Go's %v reached stdout - `print(null)` said
// "<nil>", `print([1, 2, 3])` said "[1 2 3]", a Map said
// "map[__dict:true keys:[a] vals:[1]]" and an instance dumped its whole class
// descriptor INCLUDING A GO POINTER ADDRESS, which is not even reproducible between
// two runs of the same program. (`print(SomeClass)` did not print at all: the
// descriptor's __class back-reference sent toGoNatural into a stack overflow.)
// Each language wants a different answer here, so the fix is a per-language print
// extern - the shape js_rputs / js_rprint (Ruby) and js_luaprint already have - and
// not a change to the shared printArgs.
//
// The rendering implemented here is Dart's Object.toString() and its overrides:
//
//	null            null                     Instance of 'E'   an object with no
//	true / false    true / false                               user toString()
//	[1, 2, 3]       a List                   E                 a Type (a class)
//	{a: 1}          a Map                    Col.red           an enum value
//	{1, 2}          a Set                    (1, a)            a Record
//
// Strings are NOT quoted inside a collection (`print(['a'])` is `[a]`), which is
// what makes this different from Python's repr-based rendering in pyString.
// Corpus citations, from tests/reference/dart/dart-language-tests:
//   language/regress/regress45642_test.dart:14   "should print: Instance of 'Foo'"
//   language/records/simple/literals_and_field_access_test.dart:121
//                                                x.toString() == "[1, 2, 3]"
//   language/enum/enum_test.dart:62              'Enum2.A' == Enum2.A.toString()
//   language/enum/enhanced_enums_basic_test.dart:143  a user toString() wins
//
// Const canonicalization (js_dartcanon) is keyed on the class and the FIELD VALUES,
// not on the constructor, because language/const/ct_const_test.dart:62 demands
//
//	Expect.equals(true, identical(const Point.X(5), const Point(5, 0)));
//
// - two different constructors, one canonical instance. The same rule covers const
// collections: language/canonicalize/const_test.dart asserts
// identical(const <int>[1, 2], const <int>[1, 2]) and
// identical(const {"a": 1, "b": 2}, const {"a": 1, "b": 2}).

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

func init() {
	rxExtraExterns = append(rxExtraExterns, func(rt *jsrt, m map[string]func(args []uint64) uint64) {
		rt.addDartExterns(m)
	})
}

// ===== Object.toString() =====

// dartCycleDepth caps the recursion of both the renderer and the key builder. Dart
// itself renders a self-referential collection as "[...]"; the depth cap is the
// belt to that braces, so a value model surprise degrades into a printable string
// rather than a stack overflow.
const dartCycleDepth = 64

func (rt *jsrt) dartStr(v interface{}) string { return rt.dartStrIn(v, nil, 0) }

func (rt *jsrt) dartStrIn(v interface{}, seen []interface{}, depth int) string {
	if depth > dartCycleDepth {
		return "..."
	}
	switch t := v.(type) {
	case jsUndefT, jsNullT:
		return "null"
	case bool:
		if t {
			return "true"
		}
		return "false"
	case string:
		return t
	case *jsArray:
		if dartSeen(seen, v) {
			return "[...]"
		}
		seen = append(seen, v)
		parts := make([]string, 0, len(t.elems))
		for _, e := range t.elems {
			parts = append(parts, rt.dartStrIn(e, seen, depth+1))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case *jsObject:
		if dartSeen(seen, v) {
			return "{...}"
		}
		return rt.dartObjStr(t, append(seen, v), depth)
	}
	if isCallable(v) {
		return "Closure"
	}
	return rt.toString(v)
}

func dartSeen(seen []interface{}, v interface{}) bool {
	for _, s := range seen {
		if s == v {
			return true
		}
	}
	return false
}

func (rt *jsrt) dartObjStr(o *jsObject, seen []interface{}, depth int) string {
	// A class descriptor used as a value is a Type, and Dart prints a Type as its
	// name. This case has to come first: the descriptor's own members would
	// otherwise be walked, and it points back at itself through every instance.
	if t, ok := o.props["__isclass"]; ok && t == true {
		if n, isStr := o.props["__name"].(string); isStr {
			return n
		}
	}
	if els, ok := pySetElems(o); ok {
		parts := make([]string, 0, len(els.elems))
		for _, e := range els.elems {
			parts = append(parts, rt.dartStrIn(e, seen, depth+1))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	}
	if keys, vals, ok := dictParts(o); ok {
		parts := make([]string, 0, len(keys.elems))
		for i := range keys.elems {
			parts = append(parts, rt.dartStrIn(keys.elems[i], seen, depth+1)+": "+
				rt.dartStrIn(vals.elems[i], seen, depth+1))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	}
	cls, hasCls := o.props["__class"]
	if !hasCls {
		return rt.toString(o)
	}
	// A user toString() anywhere on the __super chain wins, for a plain class and
	// for an enum alike (enhanced_enums_basic_test.dart:143).
	for c := cls; c != nil; {
		co, isObj := c.(*jsObject)
		if !isObj {
			break
		}
		if mth, ok := co.props["toString"]; ok && isCallable(mth) {
			return rt.toString(rt.call(mth, jsUndef, []interface{}{o}))
		}
		c = co.props["__super"]
	}
	name := "Object"
	if co, isObj := cls.(*jsObject); isObj {
		if n, isStr := co.props["__name"].(string); isStr {
			name = n
		}
	}
	// An enum value prints as EnumName.valueName. __enumName is written by the enum
	// emitter of dart-to-llvm-ir.abnf beside the public `name`, so an ordinary class
	// that happens to declare a field called `name` is not mistaken for one.
	if en, ok := o.props["__enumName"].(string); ok {
		return name + "." + en
	}
	// A record renders as its field list: positional fields (keyed "$1", "$2", ...)
	// as bare values, named ones as "name: value".
	if name == "Record" {
		if keys, vals, ok := dictParts(o.props["__f"]); ok {
			parts := make([]string, 0, len(keys.elems))
			for i := range keys.elems {
				k := rt.toString(keys.elems[i])
				val := rt.dartStrIn(vals.elems[i], seen, depth+1)
				if strings.HasPrefix(k, "$") {
					parts = append(parts, val)
				} else {
					parts = append(parts, k+": "+val)
				}
			}
			return "(" + strings.Join(parts, ", ") + ")"
		}
	}
	return "Instance of '" + name + "'"
}

// ===== const canonicalization =====

// dartCanon returns the canonical instance for a const value: the first value
// registered under its structural key. Children are canonicalized FIRST, so a
// nested const creation is canonical too and the key of the parent is built out of
// values that are already shared.
//
// A value the language cannot make const (a closure, a plain host object) is
// returned unchanged and never enters the cache.
func (rt *jsrt) dartCanon(v interface{}, consts map[string]interface{}, depth int) interface{} {
	if depth > dartCycleDepth {
		return v
	}
	switch t := v.(type) {
	case *jsArray:
		for i, e := range t.elems {
			t.elems[i] = rt.dartCanon(e, consts, depth+1)
		}
	case *jsObject:
		if isc, ok := t.props["__isclass"]; ok && isc == true {
			return v // a Type is already canonical
		}
		if els, ok := pySetElems(t); ok {
			for i, e := range els.elems {
				els.elems[i] = rt.dartCanon(e, consts, depth+1)
			}
		} else if keys, vals, ok := dictParts(t); ok {
			for i := range keys.elems {
				keys.elems[i] = rt.dartCanon(keys.elems[i], consts, depth+1)
				vals.elems[i] = rt.dartCanon(vals.elems[i], consts, depth+1)
			}
		} else if _, hasCls := t.props["__class"]; hasCls {
			for _, k := range t.keys {
				if strings.HasPrefix(k, "__") {
					continue
				}
				t.props[k] = rt.dartCanon(t.props[k], consts, depth+1)
			}
			if f, ok := t.props["__f"]; ok { // a record's field dict
				rt.dartCanon(f, consts, depth+1)
			}
		} else {
			return v
		}
	default:
		return v
	}
	k := rt.dartConstKey(v, 0)
	if c, ok := consts[k]; ok {
		return c
	}
	consts[k] = v
	return v
}

// dartConstKey is the structural identity of a const value. It is never printed;
// it only has to separate values that are not identical and join values that are.
func (rt *jsrt) dartConstKey(v interface{}, depth int) string {
	if depth > dartCycleDepth {
		return "!"
	}
	switch t := v.(type) {
	case jsUndefT, jsNullT:
		return "n"
	case bool:
		if t {
			return "b1"
		}
		return "b0"
	// int and double are different types, so const [1] and const [1.0] are NOT the
	// same const - the two key prefixes have to differ.
	case float64:
		return "i" + strconv.FormatFloat(t, 'f', -1, 64)
	case jsGInt:
		return "i" + giStr(t)
	case jsDartFlo:
		return "d" + dartFloStr(t.f)
	case string:
		return "s" + strconv.Itoa(len(t)) + ":" + t
	case *jsArray:
		parts := make([]string, 0, len(t.elems))
		for _, e := range t.elems {
			parts = append(parts, rt.dartConstKey(e, depth+1))
		}
		return "L(" + strings.Join(parts, ",") + ")"
	case *jsObject:
		if isc, ok := t.props["__isclass"]; ok && isc == true {
			return "T:" + rt.toString(t.props["__name"])
		}
		if els, ok := pySetElems(t); ok {
			parts := make([]string, 0, len(els.elems))
			for _, e := range els.elems {
				parts = append(parts, rt.dartConstKey(e, depth+1))
			}
			return "S(" + strings.Join(parts, ",") + ")"
		}
		if keys, vals, ok := dictParts(t); ok {
			parts := make([]string, 0, len(keys.elems))
			for i := range keys.elems {
				parts = append(parts, rt.dartConstKey(keys.elems[i], depth+1)+"=>"+
					rt.dartConstKey(vals.elems[i], depth+1))
			}
			return "M(" + strings.Join(parts, ",") + ")"
		}
		if cls, ok := t.props["__class"]; ok {
			name := ""
			if co, isObj := cls.(*jsObject); isObj {
				name = rt.toString(co.props["__name"])
			}
			// The field ORDER of an instance depends on which constructor ran, and
			// two different constructors must produce the same key
			// (ct_const_test.dart:62), so the names are sorted.
			names := make([]string, 0, len(t.keys))
			for _, k := range t.keys {
				if !strings.HasPrefix(k, "__") {
					names = append(names, k)
				}
			}
			sort.Strings(names)
			parts := make([]string, 0, len(names))
			for _, k := range names {
				parts = append(parts, k+"="+rt.dartConstKey(t.props[k], depth+1))
			}
			return "C:" + name + "(" + strings.Join(parts, ",") + ")"
		}
		return fmt.Sprintf("O:%p", t)
	}
	return fmt.Sprintf("X:%p", v)
}

// ===== The externs =====

func (rt *jsrt) addDartExterns(m map[string]func(args []uint64) uint64) {
	u := rt.unwrap
	w := rt.wrap
	// One canonical-const table per runtime, so a value canonicalized in one
	// program cannot leak into the next.
	consts := map[string]interface{}{}

	// print(x): Dart's print writes x.toString() followed by a newline.
	m["js_dartprint"] = func(a []uint64) uint64 {
		fmt.Fprintln(outWriter, wtf8Clean(rt.dartStr(u(a[0]))))
		return 0
	}
	// The same rendering as a value: string interpolation and an explicit
	// .toString() both go through it.
	m["js_dartstr"] = func(a []uint64) uint64 { return rt.wrapStr(rt.dartStr(u(a[0]))) }
	// const e: the canonical value for e.
	m["js_dartcanon"] = func(a []uint64) uint64 { return w(rt.dartCanon(u(a[0]), consts, 0)) }
	// `v is T` for Dart's OWN type names. The shared js_is_type knows the JVM
	// spelling (Int, Double, Boolean, List) and answers false for everything it
	// does not recognize - so `1 is int`, `1 is num`, `true is bool`, `{1: 2} is
	// Map` and `{1, 2} is Set` were all FALSE here while dart-interpreter.abnf's
	// dartIsType answered true for each. This is that function, in Go; user class
	// names still fall through to the shared walk of the __class chain.
	m["js_dartis"] = func(a []uint64) uint64 {
		t, _ := u(a[1]).(string)
		if rt.dartIsType(u(a[0]), t) {
			return jsHTrue
		}
		return jsHFalse
	}

	// ----- the number layer (see the invariant at the end of this file) -----
	boolH := func(v bool) uint64 {
		if v {
			return jsHTrue
		}
		return jsHFalse
	}
	// A double literal, and the value of a `.toDouble()`-shaped conversion.
	m["js_dartflo"] = func(a []uint64) uint64 { return w(jsDartFlo{f: rt.toNumber(u(a[0]))}) }
	// An int literal a double cannot hold exactly: the emitter passes its DIGITS,
	// because emitNum would already have rounded 9223372036854775807 to
	// 9223372036854776000 on the way into the module. (text, radix); a literal past
	// the 64-bit range wraps, which is how the VM reads 0x8000000000000000.
	m["js_dartlit"] = func(a []uint64) uint64 {
		s := rt.toString(u(a[0]))
		radix := uint64(jsToInt(rt.toNumber(u(a[1]))))
		var acc uint64
		for i := 0; i < len(s); i++ {
			var d uint64
			switch c := s[i]; {
			case c >= '0' && c <= '9':
				d = uint64(c - '0')
			case c >= 'a' && c <= 'f':
				d = uint64(c-'a') + 10
			case c >= 'A' && c <= 'F':
				d = uint64(c-'A') + 10
			default:
				continue // a '_' digit separator, or the 0x prefix's 'x'
			}
			if d >= radix {
				continue
			}
			acc = acc*radix + d
		}
		return w(giNorm(int64(acc), 64, false))
	}
	// One binary operator: (op name, left, right). The operator is a compile time
	// constant string, so this is one call per operator in the emitted IR - the same
	// shape js_giarith has. '+' also concatenates strings, and '+' on two Lists
	// concatenates them, so a non-numeric pair keeps the runtime's own rule.
	m["js_dartarith"] = func(a []uint64) uint64 {
		op, l, r := rt.toString(u(a[0])), u(a[1]), u(a[2])
		if op == "+" {
			// '+' is the one operator that is not only arithmetic: a string operand
			// concatenates (through the Dart rendering, so "x" + [1, 2] is "x[1, 2]"
			// and not "x1,2"), and two Lists concatenate.
			// This mirrors core.add in dart-interpreter.abnf.
			_, ls := l.(string)
			_, rs := r.(string)
			if ls || rs {
				return rt.wrapStr(strConcat(rt.dartStr(l), rt.dartStr(r)))
			}
			// dart:core, List.operator +: "Returns the concatenation of this list
			// and other" - a NEW list holding this list's elements followed by
			// other's, neither operand touched. The runtime's own rt.jsAdd is
			// JavaScript's + and renders both sides instead, so `[1, 2] + [3]`
			// used to be the STRING "1,23" with length 4 rather than the list
			// [1, 2, 3] with length 3. No dart toolchain is installed here, so
			// the dart:core signature is the oracle.
			if la, ok := l.(*jsArray); ok {
				if ra, ok2 := r.(*jsArray); ok2 {
					out := make([]interface{}, 0, len(la.elems)+len(ra.elems))
					out = append(out, la.elems...)
					out = append(out, ra.elems...)
					return w(&jsArray{elems: out})
				}
			}
			if !dartIsNum(l) || !dartIsNum(r) {
				return w(rt.jsAdd(l, r))
			}
		}
		return w(rt.dartArith(op, l, r))
	}
	rel := func(name string, want func(int) bool) {
		m[name] = func(a []uint64) uint64 {
			l, r := u(a[0]), u(a[1])
			if dartIsNum(l) && dartIsNum(r) {
				c := rt.dartCmp(l, r)
				return boolH(c != 2 && want(c))
			}
			return boolH(want(rt.jsCompare(l, r)))
		}
	}
	rel("js_dartlt", func(c int) bool { return c < 0 })
	rel("js_dartle", func(c int) bool { return c <= 0 })
	rel("js_dartgt", func(c int) bool { return c > 0 })
	rel("js_dartge", func(c int) bool { return c >= 0 })
	// == : value equality across int and double, then the runtime's own ===.
	m["js_darteq"] = func(a []uint64) uint64 {
		l, r := u(a[0]), u(a[1])
		if dartIsNum(l) && dartIsNum(r) {
			return boolH(rt.dartNumEq(l, r))
		}
		return boolH(rt.strictEq(l, r))
	}
	m["js_dartne"] = func(a []uint64) uint64 {
		l, r := u(a[0]), u(a[1])
		if dartIsNum(l) && dartIsNum(r) {
			return boolH(!rt.dartNumEq(l, r))
		}
		return boolH(!rt.strictEq(l, r))
	}
	// identical(a, b).
	m["js_dartident"] = func(a []uint64) uint64 { return boolH(rt.dartIdentical(u(a[0]), u(a[1]))) }
	// -x keeps the operand's TYPE: a double negates as a double (so -0.0 and
	// -1.0/0.0 stay real), an int at 64 bits (so -int64min wraps back to itself).
	m["js_dartneg"] = func(a []uint64) uint64 {
		v := u(a[0])
		if f, ok := v.(jsDartFlo); ok {
			return w(jsDartFlo{f: -f.f})
		}
		if !dartIsNum(v) {
			return rt.wrapNum(-rt.toNumber(v))
		}
		return w(giNorm(-giVal(rt, v), 64, false))
	}
	// ~x flips all 64 bits (Dart's int is 64-bit two's complement).
	m["js_dartnot"] = func(a []uint64) uint64 {
		return w(giNorm(^giVal(rt, rt.dartToInt(u(a[0]))), 64, false))
	}
	// The plain-number reading an index, a length or a repeat count needs.
	m["js_dartnum"] = func(a []uint64) uint64 {
		v := u(a[0])
		if dartIsNum(v) {
			return rt.wrapNum(rt.dartNumOf(v))
		}
		return a[0]
	}
	// A METHOD CALL, in the shape js_mcall has: (receiver, name, args array). It
	// answers num's own members (toInt, round, isEven, ...) when the receiver is a
	// number and the name is one of them, and otherwise hands the call straight to
	// the runtime's ordinary member dispatch - so the Dart compiler can emit this
	// one extern for every method call and a String's compareTo still reaches the
	// String. The interpreter half does exactly this in mcall / readField.
	m["js_dartnm"] = func(a []uint64) uint64 {
		recv := u(a[0])
		name := rt.toString(u(a[1]))
		args, ok := u(a[2]).(*jsArray)
		if !ok {
			rt.fail("js_dartnm args must be an array")
		}
		if dartIsNum(recv) {
			if v, hit := rt.dartNumMember(recv, name, args.elems); hit {
				return w(v)
			}
		}
		return w(rt.memberCall(recv, name, args.elems))
	}
}

// dartIsType is the Go twin of dartIsType in languages/dart-interpreter.abnf. The
// type ARGUMENTS are handled by the caller (the reified List<int> check lives in the
// grammar), so only the base name reaches here.
func (rt *jsrt) dartIsType(v interface{}, t string) bool {
	if i := strings.IndexByte(t, '<'); i >= 0 {
		t = t[:i]
	}
	opt := false
	if strings.HasSuffix(t, "?") {
		t = t[:len(t)-1]
		opt = true
	}
	if isUndefOrNull(v) {
		return opt || t == "Null"
	}
	switch t {
	case "dynamic", "Object", "var":
		return true
	// int and double are DISJOINT types in Dart: `1 is double` is false and
	// `1.0 is int` is false, even though 1 == 1.0. The answer comes from the BOX
	// (see the number layer at the end of this file), not from the value - an
	// unboxed number is an int, which is why this used to answer true for both.
	case "int":
		return dartIsInt(v)
	case "double":
		return dartIsFlo(v)
	case "num":
		return dartIsNum(v)
	case "String":
		_, ok := v.(string)
		return ok
	case "bool":
		_, ok := v.(bool)
		return ok
	case "List", "Iterable":
		_, ok := v.(*jsArray)
		return ok
	case "Map":
		// A Map is the shared dict object; a Set is a dict-shaped object carrying
		// __set, and it is NOT a Map.
		if o, ok := v.(*jsObject); ok {
			if _, isSet := o.props["__set"]; isSet {
				return false
			}
		}
		_, _, ok := dictParts(v)
		return ok
	case "Set":
		o, ok := v.(*jsObject)
		if !ok {
			return false
		}
		_, isSet := o.props["__set"]
		return isSet
	case "Function":
		return isCallable(v)
	}
	return rt.isTypeName(v, t)
}

// ===== Dart's numbers: a 64-bit int and a separate double =====
//
// Dart has TWO numeric types and no implicit conversion between them, and the
// handle runtime models every number as one JS double - so 1 and 1.0 arrived here
// as the same value. print(1.0) said "1", `1 / 1` said "1" although Dart's `/`
// ALWAYS yields a double, `1 is double` was true and `1.0 is int` was true. Dart's
// int is also exactly 64-bit two's complement on the VM and WRAPS, which a double
// cannot represent: 9223372036854775807 + 1 printed 9.2e+18 here and -1 in the
// interpreter half - a live cross-half divergence.
//
// The fix is the one commit 636e326 used for the JVM languages and aaa195e for
// Go's widths: the type goes ON the value. The invariant, in BOTH halves of this
// language (languages/dart-interpreter.abnf implements exactly these rules with
// the {__flo} box and the shared {__sz} layer of lib/interp-core.js):
//
//	plain number   ==  an int whose value is exact in a double (|v| <= 2^53)
//	jsGInt         ==  an int outside that range (w = 64, u = false, always)
//	jsDartFlo      ==  a double
//
// The integer side is the sized-integer layer of jsrtint.go unchanged - Dart only
// ever asks it for the signed 64-bit width, so giNorm's fast case keeps every
// ordinary program's numbers unboxed and this file adds no arithmetic of its own
// for them.

// jsDartFlo is a double, boxed so that it stays distinguishable from an int.
// jsJFlo would not do: it carries a per-language rendering style whose switch
// lives in jsrtjvm.go, and Dart's `/` rule (always a double, even for two ints)
// is not one the JVM operators can express.
type jsDartFlo struct{ f float64 }

func dartIsFlo(v interface{}) bool { _, ok := v.(jsDartFlo); return ok }

func dartIsInt(v interface{}) bool {
	switch v.(type) {
	case float64, jsGInt:
		return true
	}
	return false
}

func dartIsNum(v interface{}) bool { return dartIsFlo(v) || dartIsInt(v) }

// dartFloStr is Dart's double.toString: ECMAScript's number-to-string with ".0"
// appended when the result carries neither a point nor an exponent, so 1.0 is
// "1.0", 1e21 is "1e+21" and 1e-7 is "1e-7". -0.0 keeps its sign although
// ECMAScript spells it "0".
// Cited: tests/reference/dart/dart-language-tests/language/double/to_string_test.dart
// lines 9-11 (NaN / Infinity / -Infinity), 26 ("1e-7"), 33 ("-0.0"), 56 ("1e+21").
func dartFloStr(f float64) string {
	if math.IsNaN(f) {
		return "NaN"
	}
	if math.IsInf(f, 1) {
		return "Infinity"
	}
	if math.IsInf(f, -1) {
		return "-Infinity"
	}
	s := jsNumString(f)
	if f == 0 && math.Signbit(f) {
		s = "-0"
	}
	if !strings.ContainsAny(s, ".e") {
		s += ".0"
	}
	return s
}

// dartNumOf is the double reading of any Dart number.
func (rt *jsrt) dartNumOf(v interface{}) float64 {
	if f, ok := v.(jsDartFlo); ok {
		return f.f
	}
	return giFloat(rt, v)
}

// dartToInt truncates a double toward zero into the int representation; an int
// passes through. It is toInt(), truncate(), and the operand coercion every
// bitwise operator performs.
func (rt *jsrt) dartToInt(v interface{}) interface{} {
	if f, ok := v.(jsDartFlo); ok {
		return giNorm(giFromFloat(f.f), 64, false)
	}
	return v
}

// dartArith is one binary operator on two Dart numbers.
//
//	/    ALWAYS a double, for two ints as well (7 / 2 is 3.5, 1 / 1 is 1.0)
//	~/   truncating division, and its result is an int even for two doubles
//	%    EUCLIDEAN on both types: 0 <= r < b.abs(), so -7 % 3 is 2
//	     (remainder() is the truncating one and answers -1)
//	the rest  floating point as soon as either operand is a double, exact
//	     64-bit integer arithmetic otherwise (giArith, jsrtint.go)
func (rt *jsrt) dartArith(op string, l, r interface{}) interface{} {
	switch op {
	case "/":
		return jsDartFlo{f: rt.dartNumOf(l) / rt.dartNumOf(r)}
	case "~/":
		if dartIsFlo(l) || dartIsFlo(r) {
			q := rt.dartNumOf(l) / rt.dartNumOf(r)
			if math.IsNaN(q) || math.IsInf(q, 0) {
				panic(&jsThrown{value: "Unsupported operation: Result of truncating division is " + dartFloStr(q)})
			}
			return giNorm(giFromFloat(q), 64, false)
		}
		return rt.giArith("/", rt.dartToInt(l), rt.dartToInt(r))
	case ">>>":
		n := rt.dartNumOf(r)
		if n < 0 {
			panic(&jsThrown{value: "Unsupported operation: negative shift amount"})
		}
		if n >= 64 {
			return float64(0)
		}
		return giNorm(int64(uint64(giVal(rt, rt.dartToInt(l)))>>uint(n)), 64, false)
	}
	if dartIsFlo(l) || dartIsFlo(r) {
		a, b := rt.dartNumOf(l), rt.dartNumOf(r)
		switch op {
		case "+":
			return jsDartFlo{f: a + b}
		case "-":
			return jsDartFlo{f: a - b}
		case "*":
			return jsDartFlo{f: a * b}
		case "%":
			return jsDartFlo{f: dartModF(a, b)}
		}
	}
	li, ri := rt.dartToInt(l), rt.dartToInt(r)
	if op == "%" {
		m := rt.giArith("%", li, ri)
		if rt.giCmp(m, float64(0)) < 0 {
			absr := ri
			if rt.giCmp(ri, float64(0)) < 0 {
				absr = giNorm(-giVal(rt, ri), 64, false)
			}
			m = rt.giArith("+", m, absr)
		}
		return m
	}
	return rt.giArith(op, li, ri)
}

func dartModF(a, b float64) float64 {
	m := math.Mod(a, b)
	if m < 0 {
		m += math.Abs(b)
	}
	return m
}

// dartCmp is the ordered comparison: -1, 0, 1, or 2 for the unordered NaN case.
func (rt *jsrt) dartCmp(l, r interface{}) int {
	if dartIsFlo(l) || dartIsFlo(r) {
		a, b := rt.dartNumOf(l), rt.dartNumOf(r)
		switch {
		case a < b:
			return -1
		case a > b:
			return 1
		case a == b:
			return 0
		}
		return 2
	}
	return rt.giCmp(l, r)
}

// dartNumEq is Dart's == between numbers: value equality ACROSS the two types, so
// 1 == 1.0 holds even though `1 is double` is false
// (language/number/numbers_test.dart:16-29 pins the types; the equality across them
// is spec-derived from dart:core num.operator==).
func (rt *jsrt) dartNumEq(l, r interface{}) bool {
	if dartIsFlo(l) || dartIsFlo(r) {
		return rt.dartNumOf(l) == rt.dartNumOf(r)
	}
	return rt.giEq(l, r)
}

// dartIdentical is identical(): numbers have no identity apart from their value and
// their type. language/number/identity_test.dart:20 and :31 assert identical(8, 8+0)
// and identical(8.0, 8.0+0.0); :27 asserts an int is never identical to a double;
// language/number/identity2_test.dart:24 asserts identical(nan, nan+0.0).
func (rt *jsrt) dartIdentical(a, b interface{}) bool {
	if dartIsNum(a) || dartIsNum(b) {
		if !dartIsNum(a) || !dartIsNum(b) || dartIsFlo(a) != dartIsFlo(b) {
			return false
		}
		if dartIsFlo(a) {
			x, y := a.(jsDartFlo).f, b.(jsDartFlo).f
			if math.IsNaN(x) && math.IsNaN(y) {
				return true
			}
			if x == 0 && y == 0 {
				return math.Signbit(x) == math.Signbit(y) // 0.0 is not -0.0
			}
			return x == y
		}
		return rt.giEq(a, b)
	}
	return rt.strictEq(a, b)
}

// dartNumMember answers num's own members - the getters and methods dart:core
// declares on num / int / double. They matter far more now that int and double are
// distinct, because toInt / toDouble are how a program crosses between them.
// floor, ceil, round and truncate answer an INT even on a double; abs keeps the
// operand's type.
// Cited: language/operator/arithmetic_test.dart:96-116 (floor/ceil/round/truncate on
// an int answer the int itself, both signs) and :57/:73 (remainder is the TRUNCATING
// one, unlike %); language/operator/round_test.dart:22-26 (round is not floor(x+0.5)).
// The rest - toDouble, abs, sign, isEven, toRadixString, clamp - is spec-derived from
// the dart:core documentation of num / int / double; the corpus is silent on them.
func (rt *jsrt) dartNumMember(v interface{}, name string, args []interface{}) (interface{}, bool) {
	f := rt.dartNumOf(v)
	isF := dartIsFlo(v)
	arg := func(i int) interface{} {
		if i < len(args) {
			return args[i]
		}
		return nil
	}
	switch name {
	case "toDouble":
		return jsDartFlo{f: f}, true
	case "toInt", "truncate":
		return rt.dartToInt(v), true
	case "floor":
		if isF {
			return giNorm(giFromFloat(math.Floor(f)), 64, false), true
		}
		return v, true
	case "ceil":
		if isF {
			return giNorm(giFromFloat(math.Ceil(f)), 64, false), true
		}
		return v, true
	case "round":
		// Dart rounds a tie AWAY FROM ZERO: 2.5 is 3 and -2.5 is -3.
		if isF {
			return giNorm(giFromFloat(dartRound(f)), 64, false), true
		}
		return v, true
	case "floorToDouble":
		return jsDartFlo{f: math.Floor(f)}, true
	case "ceilToDouble":
		return jsDartFlo{f: math.Ceil(f)}, true
	case "truncateToDouble":
		return jsDartFlo{f: math.Trunc(f)}, true
	case "roundToDouble":
		return jsDartFlo{f: dartRound(f)}, true
	case "abs":
		if isF {
			return jsDartFlo{f: math.Abs(f)}, true
		}
		if rt.giCmp(v, float64(0)) < 0 {
			return giNorm(-giVal(rt, v), 64, false), true
		}
		return v, true
	case "isNaN":
		return math.IsNaN(f), true
	case "isFinite":
		return !math.IsNaN(f) && !math.IsInf(f, 0), true
	case "isInfinite":
		return math.IsInf(f, 0), true
	case "isNegative":
		return f < 0 || (f == 0 && math.Signbit(f)), true
	case "isEven":
		return dartIsInt(v) && giVal(rt, v)%2 == 0, true
	case "isOdd":
		return dartIsInt(v) && giVal(rt, v)%2 != 0, true
	case "sign":
		if isF {
			if math.IsNaN(f) {
				return jsDartFlo{f: f}, true
			}
			if f > 0 {
				return jsDartFlo{f: 1}, true
			}
			if f < 0 {
				return jsDartFlo{f: -1}, true
			}
			return jsDartFlo{f: f}, true
		}
		return float64(rt.giCmp(v, float64(0))), true
	case "remainder":
		if isF || dartIsFlo(arg(0)) {
			return jsDartFlo{f: math.Mod(f, rt.dartNumOf(arg(0)))}, true
		}
		return rt.giArith("%", v, arg(0)), true
	case "compareTo":
		c := rt.dartCmp(v, arg(0))
		if c == 2 {
			c = 0
		}
		return float64(c), true
	case "clamp":
		if rt.dartCmp(v, arg(0)) < 0 {
			return arg(0), true
		}
		if rt.dartCmp(v, arg(1)) == 1 {
			return arg(1), true
		}
		return v, true
	case "toRadixString":
		return dartRadixStr(rt, rt.dartToInt(v), int(rt.dartNumOf(arg(0)))), true
	}
	return nil, false
}

// dartRound rounds to the nearest integer with TIES AWAY FROM ZERO. It is
// deliberately not Math.floor(x + 0.5): language/operator/round_test.dart:22 and
// :25 exist precisely to reject that formula, because 0.49999999999999994 + 0.5
// is 1.0 and 9007199254740991 + 0.5 is 9007199254740992.
func dartRound(f float64) float64 {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return f
	}
	// A negative operand rounds by SYMMETRY (ties away from zero is symmetric)
	// rather than through Floor(f) directly: Floor(-0.49999999999999994) is -1 and
	// the difference from it rounds to exactly 0.5, which would answer -1.
	if f < 0 {
		return -dartRoundPos(-f)
	}
	return dartRoundPos(f)
}

func dartRoundPos(a float64) float64 {
	fl := math.Floor(a)
	if a-fl >= 0.5 {
		return fl + 1
	}
	return fl
}

func dartRadixStr(rt *jsrt, v interface{}, radix int) string {
	n := giVal(rt, v)
	if radix < 2 || radix > 36 {
		radix = 10
	}
	if n < 0 {
		// strconv.FormatInt handles the sign, and int64min negates to itself, so
		// the unsigned path below cannot be used for it.
		return strconv.FormatInt(n, radix)
	}
	return strconv.FormatUint(uint64(n), radix)
}
