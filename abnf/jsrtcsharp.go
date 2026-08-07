package abnf

// jsrtcsharp.go -- C#'s VALUE RENDERING for the compiler grammar
// (languages/csharp-to-llvm-ir.abnf).
//
// csharp-to-llvm-ir.abnf used to bind Console.WriteLine / Console.Write straight
// to the shared `println` / `print` host functions, which hand the raw runtime
// value to Go's `%v`. That printed `<nil>` for a null reference, `[1 2 3]` for an
// array and a full `map[__class:map[__isclass:true __name:Box ...] x:0]` dump -
// including a raw Go POINTER address for the constructor - for a class instance,
// so the compiler's stdout was not even reproducible between two runs of the same
// program. The interpreter half was wrong in its own way: its `csStr` fell through
// to JavaScript's ToString, so an array printed `1,2,3` and an instance
// `[object Object]`.
//
// On top of that both halves agreed on ONE wrong answer, which is why
// `./test.sh --cross` could never flag it: a bool converted to a string printed
// JavaScript's `true` / `false`. `System.Boolean.ToString()` returns
// `Boolean.TrueString` / `Boolean.FalseString`, which are the strings "True" and
// "False" (the LITERALS `true` / `false` are unchanged - only the string
// CONVERSION is capitalised).
//
// This file is the Go twin of `csStr` in csharp-interpreter.abnf and the two MUST
// stay in step, because ./test.sh --cross diffs them.
//
// Registration is additive: an init() appends a registrar to rxExtraExterns, so
// nothing here runs unless a program calls one of the js_cs* externs registered
// below, and only the C# compiler grammar emits them. No existing extern is
// renamed, removed or rebound, and abnf/jsrt.go is untouched.
//
// Spec basis. There is no dotnet on this machine, so every answer below is CITED
// rather than executed:
//
//   - bool     "True" / "False"        System.Boolean.ToString(), which is
//                                      documented to return Boolean.TrueString /
//                                      Boolean.FalseString ("True" / "False").
//                                      A .NET BCL library contract, not an
//                                      ECMA-334 language rule.
//   - null     ""                      ECMA-334 12.4.7 (string concatenation:
//                                      a null operand contributes the empty
//                                      string) and System.Console.WriteLine(object),
//                                      which writes only a line terminator when the
//                                      argument is null. Also String.Concat, which
//                                      calls ToString() on non-null arguments only.
//   - array    "System.Int32[]"        System.Object.ToString() returns the fully
//                                      qualified type name (ECMA-334 8.2.3 lists
//                                      ToString among the members every type
//                                      inherits from object); an array type's name
//                                      is the element type's name plus "[]".
//   - instance "Box"                   System.Object.ToString() returns
//                                      GetType().ToString(), the fully qualified
//                                      name. See the simplification below.
//   - double   csFloStr                unchanged from the commit that boxed the
//                                      float (see jvmFloText / floCS in
//                                      abnf/jsrtjvm.go); ECMA-335 "G" round-trip.
//   - char     the glyph               System.Char.ToString().
//
// A user-defined ToString() override wins over ALL of the above, in both halves -
// System.Object.ToString() is virtual and every conversion site (Console.WriteLine,
// String.Concat, an interpolation hole) calls the virtual method.
//
// Known simplifications, all DELIBERATE and all shared by the two halves:
//
//   - An ARRAY's element type is INFERRED from its elements, because the declared
//     element type is not carried by this value model. The rule, in both halves:
//     plain number -> System.Int32[], float box -> System.Double[], string ->
//     System.String[], bool -> System.Boolean[], char -> System.Char[], and an
//     empty or mixed array -> System.Object[]. Real C# would answer from the
//     DECLARED type, so `object[] o = new object[]{1}` prints System.Int32[] here
//     and System.Object[] there.
//   - A List<T> is the same plain array in this runtime, so it renders like an
//     array. Real C# prints "System.Collections.Generic.List`1[System.Int32]".
//   - An INSTANCE prints its BARE declared name. Real C# prints the fully
//     qualified name, which for a nested class in a namespace would be
//     `Demo.Program+Box`; this runtime has no namespace and no nesting concept at
//     all, so the bare name is the reproducible part of that answer. (The same
//     simplification abnf/jsrtswift.go records for Swift's module qualification.)
//   - A RECORD prints its type name like any other instance. Real C# synthesizes
//     a `Box { X = 1 }` ToString for a record.
//   - A Dictionary and a tuple are left exactly as they rendered before (the
//     generic "[object Object]"), because both halves already agreed there and
//     neither shape carries the type information the real answer needs.

import (
	"fmt"
	"math"
	"strings"
)

// ----------------------------------------------------------------------------
// SIZED INTEGERS
//
// C#'s integral types have fixed widths AND a signedness, and both have to
// survive into the next operation: `byte b = 255; b++` is 0, uint.MaxValue is
// 4294967295 rather than -1, and ulong.MaxValue / 3 is 6148914691236517205 - an
// answer no double holds. The box that carries it is jsGInt from jsrtint.go, the
// same one Go, Swift and Java use; only the RULES are C#'s and they live here, so
// no shared file has to learn a second dialect.
//
// The invariant is C#'s and it is deliberately NOT Go's. There a plain number is
// a 64 bit int; here
//
//	a plain number  ==  an `int`, i.e. a SIGNED 32 BIT value
//	a jsGInt        ==  every other integral type: long, ulong, uint, short,
//	                    ushort, byte (UNSIGNED in C#) and sbyte
//	a jsJFlo        ==  a double / float / decimal              (jsrtjvm.go)
//	a jsChar        ==  a char (16 bit unsigned, rendered as a glyph)
//
// ARITHMETIC IS UNCHECKED, C#'s default (ECMA-334 12.8.20): every operator wraps.
// A `checked` block is parsed but NOT modelled - it does not throw - and that is
// a documented gap. Division by zero DOES throw (12.9.3), because it throws in
// both contexts.
//
// There is no dotnet on this machine, so every rule below is CITED rather than
// executed; the section header of each function names the clause.
//
// languages/csharp-interpreter.abnf implements exactly these rules on top of the
// sz* layer in lib/interp-core.js, and ./test.sh --cross diffs the two halves.

func csSzW(v interface{}) uint8 {
	if b, ok := v.(jsGInt); ok {
		return b.w
	}
	return 32
}

func csSzU(v interface{}) bool {
	if b, ok := v.(jsGInt); ok {
		return b.u
	}
	return false
}

// csIsIntegral says whether a value is an operand of INTEGER arithmetic - which a
// char is, C#'s char being an integral type, and a double is not.
func csIsIntegral(v interface{}) bool {
	switch v.(type) {
	case jsGInt, float64, jsChar:
		return true
	}
	return false
}

// csIsTyWidth resolves an INTEGRAL type name - C#'s keyword or its BCL alias
// (ECMA-334 8.3.5: the keywords are aliases for the System types) - to the width
// and signedness the value model stores. The third result is false for every name
// that is not an integral type.
func csIsTyWidth(t string) (uint8, bool, bool) {
	alias := map[string]string{"SByte": "sbyte", "Byte": "byte", "Int16": "short",
		"UInt16": "ushort", "Int32": "int", "UInt32": "uint", "Int64": "long", "UInt64": "ulong"}
	if a, ok := alias[t]; ok {
		t = a
	}
	bits, ok := csTypeW[t]
	if !ok {
		return 0, false, false
	}
	return bits, csTypeU[t], true
}

// csIsNumericV says whether a value can be CONVERTED to a double/float by an
// implicit numeric conversion: a plain number (an `int` here), a sized integer,
// or a float box. A char is deliberately NOT one - `double d = 'a'` is legal C#
// but the interpreter's csImplicitConv declines it too, and the two halves have
// to agree. Everything else (null, a string, a lambda, an instance) passes
// through the adoption sites untouched.
// "double[]" -> "double" when the elements adopt, "" otherwise. One level only, the
// same rule csElemAdoptTy applies in both grammars.
func csElemAdoptName(ty string) string {
	if len(ty) < 3 || !strings.HasSuffix(ty, "[]") {
		return ""
	}
	e := ty[:len(ty)-2]
	if e == "double" || e == "float" || e == "decimal" {
		return e
	}
	if _, ok := csTypeW[e]; ok {
		return e
	}
	return ""
}

func csIsNumericV(v interface{}) bool {
	switch v.(type) {
	case jsGInt, float64, jsJFlo:
		return true
	}
	return false
}

// csNorm applies the invariant: a signed 32 bit result is a PLAIN NUMBER,
// everything else is a box. giNorm cannot be used - it unboxes at 64 bits, where
// a plain number means `int` here.
func csNorm(v int64, w uint8, u bool) interface{} {
	v = giTrunc(v, w, u)
	if w == 32 && !u {
		return float64(int32(v))
	}
	return jsGInt{v: v, w: w, u: u}
}

// csBox reads a value at a given width and signedness.
func csBox(rt *jsrt, v interface{}, w uint8, u bool) jsGInt {
	return jsGInt{v: giTrunc(giVal(rt, v), w, u), w: w, u: u}
}

// csPromote is the BINARY NUMERIC PROMOTION of ECMA-334 12.4.7.3 for two integral
// operands. The surprising rule is the uint one: `uint + int` is a LONG, because
// int does not fit in uint and uint does not fit in int. That is the only place
// C# differs from Java's much simpler "long if either is long, else int".
//
//	ulong, if either operand is ulong
//	long,  if either operand is long
//	long,  if one is uint and the other is sbyte / short / int
//	uint,  if either operand is uint
//	int,   otherwise (so byte + byte and short * short are int)
func csPromote(l, r interface{}) (uint8, bool) {
	lw, lu := csSzW(l), csSzU(l)
	rw, ru := csSzW(r), csSzU(r)
	if (lw == 64 && lu) || (rw == 64 && ru) {
		return 64, true
	}
	if lw == 64 || rw == 64 {
		return 64, false
	}
	if (lw == 32 && lu) || (rw == 32 && ru) {
		other := l
		if lw == 32 && lu {
			other = r
		}
		// A char, a byte and a ushort convert to uint and stay there; a plain
		// number is an `int` and forces the pair up to long.
		if _, isChar := other.(jsChar); isChar {
			return 32, true
		}
		if b, isBox := other.(jsGInt); isBox {
			if b.u || b.w > 32 {
				return 32, true
			}
			return 64, false
		}
		return 64, false
	}
	return 32, false
}

// csThrow raises one of the BCL exception types, which in both halves are plain
// objects carrying __excname and Message - the shape `new
// DivideByZeroException(msg)` builds and a catch clause reads.
func (rt *jsrt) csThrow(name, msg string) {
	o := newJSObject()
	o.set("__excname", name)
	o.set("Message", msg)
	panic(&jsThrown{value: o})
}

// csArith is one binary arithmetic or bitwise operator on two INTEGRAL operands,
// at the promoted type and wrapping to it (unchecked).
func (rt *jsrt) csArith(op string, l, r interface{}) interface{} {
	w, un := csPromote(l, r)
	if w != 32 || un {
		a, b := csBox(rt, l, w, un), csBox(rt, r, w, un)
		if (op == "/" || op == "%") && b.v == 0 {
			rt.csThrow("DivideByZeroException", "Attempted to divide by zero.")
		}
		return csNorm(giVal(rt, rt.giArith(op, a, b)), w, un)
	}
	a, b := int32(giVal(rt, l)), int32(giVal(rt, r))
	var x int32
	switch op {
	case "+":
		x = a + b
	case "-":
		x = a - b
	case "*":
		x = a * b
	case "/":
		if b == 0 {
			rt.csThrow("DivideByZeroException", "Attempted to divide by zero.")
		}
		// int.MinValue / -1 overflows. ECMA-334 12.9.3 leaves the UNCHECKED
		// outcome implementation-defined between a System.OverflowException and
		// "the resulting value being that of the left operand"; the second is what
		// both halves answer, and it is what Java specifies outright.
		if a == math.MinInt32 && b == -1 {
			x = a
		} else {
			x = a / b
		}
	case "%":
		if b == 0 {
			rt.csThrow("DivideByZeroException", "Attempted to divide by zero.")
		}
		if a == math.MinInt32 && b == -1 {
			x = 0
		} else {
			x = a % b
		}
	case "&":
		x = a & b
	case "|":
		x = a | b
	case "^":
		x = a ^ b
	default:
		rt.fail("js_csarith: unknown operator %q", op)
	}
	return float64(x)
}

// csShift is << >> >>>. The result type is the LEFT operand's alone after unary
// promotion (ECMA-334 12.11: sbyte/byte/short/ushort/char all become int), and
// the COUNT is MASKED - & 31 for an int/uint, & 63 for a long/ulong, so
// `1 << 32` is 1. `>>` on an UNSIGNED type is the LOGICAL shift, which is how C#
// spells what Java writes as `>>>`; C# 11's own `>>>` is that same operation on a
// signed type.
func (rt *jsrt) csShift(op string, l, r interface{}) interface{} {
	n := giVal(rt, r)
	w := uint8(32)
	if csSzW(l) == 64 {
		w = 64
	}
	un := false
	if csSzW(l) >= 32 {
		un = csSzU(l)
	}
	s := uint(n & 31)
	if w == 64 {
		s = uint(n & 63)
	}
	a := giTrunc(giVal(rt, l), w, un)
	var x int64
	switch op {
	case "<<":
		x = a << s
	case ">>>":
		x = int64(giU(a, w) >> s)
	default:
		if un {
			x = int64(giU(a, w) >> s)
		} else {
			x = a >> s
		}
	}
	return csNorm(x, w, un)
}

// csCmp is the ordered comparison at the PROMOTED type. giCmp on its own is not
// usable: it compares at the widest operand's width, so `byte b = 5; b < 1000`
// would truncate 1000 to a byte and answer false.
func (rt *jsrt) csCmp(op string, l, r interface{}) bool {
	w, un := csPromote(l, r)
	c := rt.giCmp(csBox(rt, l, w, un), csBox(rt, r, w, un))
	switch op {
	case "<":
		return c < 0
	case ">":
		return c > 0
	case "<=":
		return c <= 0
	case ">=":
		return c >= 0
	case "==":
		return c == 0
	}
	return c != 0
}

// csConvTo is an explicit conversion to an integral type. An INTEGRAL source is a
// pure bit truncation (unchecked); a FLOATING POINT one whose value does not fit
// is "undefined behaviour" in ECMA-334 12.3.2, so this SATURATES - deterministic,
// and identical in both halves.
func csConvTo(rt *jsrt, v interface{}, w uint8, u bool) interface{} {
	if f, isFlo := v.(jsJFlo); isFlo {
		n := f.f
		if math.IsNaN(n) {
			n = 0
		}
		var p int64
		switch {
		case n >= 18446744073709551616:
			p = -1 // ulong.MaxValue's bit pattern
		case n >= 9223372036854775808:
			if u {
				p = int64(uint64(n))
			} else {
				p = math.MaxInt64
			}
		case n < -9223372036854775808:
			p = math.MinInt64
		default:
			p = int64(math.Trunc(n))
		}
		return csNorm(p, w, u)
	}
	return csNorm(giVal(rt, v), w, u)
}

// The widths and signedness of C#'s integral type names. C#'s `byte` is UNSIGNED
// (0..255) and `sbyte` is the signed one - the opposite of Java, and the single
// most common way to get this wrong.
var csTypeW = map[string]uint8{"sbyte": 8, "byte": 8, "short": 16, "ushort": 16,
	"int": 32, "uint": 32, "long": 64, "ulong": 64, "nint": 64, "nuint": 64}
var csTypeU = map[string]bool{"sbyte": false, "byte": true, "short": false, "ushort": true,
	"int": false, "uint": true, "long": false, "ulong": true, "nint": false, "nuint": true}

// csParseInt is int.Parse / long.Parse / uint.Parse: a decimal digit run with an
// optional sign, and a System.FormatException on anything else.
func (rt *jsrt) csParseInt(s string, w uint8, u bool) interface{} {
	t := strings.TrimSpace(s)
	neg := false
	if strings.HasPrefix(t, "+") || strings.HasPrefix(t, "-") {
		neg = t[0] == '-'
		t = t[1:]
	}
	if t == "" {
		rt.csThrow("FormatException", "The input string was not in a correct format.")
	}
	var acc uint64
	for i := 0; i < len(t); i++ {
		if t[i] < '0' || t[i] > '9' {
			rt.csThrow("FormatException", "The input string was not in a correct format.")
		}
		acc = acc*10 + uint64(t[i]-'0')
	}
	v := int64(acc)
	if neg {
		v = -v
	}
	return csNorm(v, w, u)
}

// csTypeObject is one of the integral type names as an expression primary, so
// that `int.MaxValue` and `ulong.MaxValue` resolve.
func (rt *jsrt) csTypeObject(name string) *jsObject {
	o := newJSObject()
	if name == "char" || name == "Char" {
		o.set("MinValue", jsChar{code: 0})
		o.set("MaxValue", jsChar{code: 65535})
		return o
	}
	alias := map[string]string{"SByte": "sbyte", "Byte": "byte", "Int16": "short",
		"UInt16": "ushort", "Int32": "int", "UInt32": "uint", "Int64": "long", "UInt64": "ulong"}
	if a, ok := alias[name]; ok {
		name = a
	}
	w, ok := csTypeW[name]
	if !ok {
		return o
	}
	un := csTypeU[name]
	var lo, hi int64
	if un {
		lo, hi = 0, int64(-1)
		if w < 64 {
			hi = int64(1)<<uint(w) - 1
		}
	} else {
		hi = int64(1)<<uint(w-1) - 1
		lo = -hi - 1
	}
	o.set("MinValue", csNorm(lo, w, un))
	o.set("MaxValue", csNorm(hi, w, un))
	o.set("Parse", jsHostFunc("Parse", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return rt.csParseInt(rt.toString(argAt(args, 0)), w, un)
	}))
	return o
}

func init() {
	rxExtraExterns = append(rxExtraExterns, func(rt *jsrt, m map[string]func(args []uint64) uint64) {
		u := rt.unwrap
		w := rt.wrap
		boolH := func(b bool) uint64 {
			if b {
				return jsHTrue
			}
			return jsHFalse
		}
		opOf := func(h uint64) string { return rt.toString(u(h)) }

		// One binary integral operator: (op, left, right). A double operand keeps
		// the FLOAT path of jsrtjvm.go, which is where the two boxes meet; two
		// bools take C#'s non-short-circuit & | ^.
		m["js_csarith"] = func(a []uint64) uint64 {
			op, l, r := opOf(a[0]), u(a[1]), u(a[2])
			if jvmIsFlo(l) || jvmIsFlo(r) {
				return w(rt.jvmArith(op[0], l, r))
			}
			lb, lok := l.(bool)
			rb, rok := r.(bool)
			if lok && rok {
				switch op {
				case "&":
					return boolH(lb && rb)
				case "|":
					return boolH(lb || rb)
				case "^":
					return boolH(lb != rb)
				}
			}
			if !csIsIntegral(l) || !csIsIntegral(r) {
				return w(rt.jvmArith(op[0], l, r))
			}
			return w(rt.csArith(op, l, r))
		}
		// js_csflo32 is the ONE new extern the float width needs, and it is the
		// twin of js_csflo (a FLOOR primitive, languages/lib/runtime.c) at 24
		// significant bits: a `1.0f` literal, a `(float)e` cast and a `float x =
		// e` declaration all lower to it. It is registered here rather than in
		// abnf/jsrt.go's table because the floor cannot hold a 32-bit box - see
		// the header of abnf/jsrtjvm.go. Its layer-2 twin is js_csflo32 in
		// languages/lib/csharp-rt.metajs.
		m["js_csflo32"] = func(a []uint64) uint64 {
			return w(jsJFlo{f: jvmFround(rt.toNumber(u(a[0]))), sty: floCSF})
		}
		m["js_csshift"] = func(a []uint64) uint64 {
			return w(rt.csShift(opOf(a[0]), u(a[1]), u(a[2])))
		}
		m["js_cscmp"] = func(a []uint64) uint64 {
			op, l, r := opOf(a[0]), u(a[1]), u(a[2])
			if csIsIntegral(l) && csIsIntegral(r) && !jvmIsFlo(l) && !jvmIsFlo(r) {
				return boolH(rt.csCmp(op, l, r))
			}
			// A `float` operand converts the OTHER side to a float before the
			// comparison (ECMA-334 12.4.7.3 binary numeric promotion, the same
			// rule as JLS 5.6.2), so `16777216f < 16777217` is FALSE - the int
			// becomes 1.6777216E7 on the way in. The test sits BEHIND the
			// integral fast path above, which is where 8f43e84 measured java at
			// +5.32% for putting it in front.
			l, r = jvmFloCmpPair(rt, l, r)
			c := rt.jsCompare(l, r)
			// jsCompare answers the SENTINEL 2 for a NaN operand, commented
			// there as "every relation is false" - and that is exactly C#'s
			// rule (ECMA-334 12.12.1: for the floating point relational
			// operators, "if either operand is NaN, the result is false for
			// all operators except !="). Read as an ordering the sentinel made
			// `>` and `>=` TRUE, so `double.NaN > 0.0` answered true here and
			// in layer 2. It is not 0 either: 0 would make `<=` and `>=` true,
			// which is wrong in the other direction.
			if c == 2 {
				return boolH(false)
			}
			switch op {
			case "<":
				return boolH(c < 0)
			case ">":
				return boolH(c > 0)
			case "<=":
				return boolH(c <= 0)
			}
			return boolH(c >= 0)
		}
		// A cast or a declared integral type: (value, bits, unsigned).
		m["js_csconv"] = func(a []uint64) uint64 {
			return w(csConvTo(rt, u(a[0]), uint8(jsToInt(rt.toNumber(u(a[1])))), rt.truthy(u(a[2]))))
		}
		// A declared type applied to an initializer: only an INTEGRAL value
		// converts, so `object o = "s"` and a lambda pass through untouched.
		m["js_csadopt"] = func(a []uint64) uint64 {
			v := u(a[0])
			if !csIsIntegral(v) {
				return a[0]
			}
			bits := uint8(jsToInt(rt.toNumber(u(a[1]))))
			uns := rt.truthy(u(a[2]))
			// Adopting a value that ALREADY carries the width and signedness is the
			// identity (see js_csadoptty for the measurement that made this matter).
			if g, isG := v.(jsGInt); isG {
				if g.w == bits && g.u == uns {
					return a[0]
				}
			} else if f, isF := v.(float64); isF && bits == 32 && !uns && f == math.Trunc(f) {
				return a[0]
			}
			return w(csConvTo(rt, v, bits, uns))
		}
		// js_csadoptty(v, ty) is ECMA-334 10.2.3's implicit numeric conversion at
		// a DECLARED TYPE NAME - a parameter, a method's return type, a field or
		// an auto-property - and it is the twin of csImplicitConv's numeric arms
		// in languages/csharp-interpreter.abnf. It is what makes
		//
		//     static double G(double x) { return x / 2; }   G(3)
		//
		// answer 1.5 rather than the 1 an int division gave: THE WIDTH IS ON THE
		// VALUE, NOT THE ANNOTATION, so nothing but a conversion at the binding
		// site can put it there (docs/todo.md 1.1). Anything that is not a
		// number, a sized integer or a float box is handed back UNCHANGED, so a
		// null, a string or a lambda at such a site is untouched.
		m["js_csadoptty"] = func(a []uint64) uint64 {
			v := u(a[0])
			ty := rt.toString(u(a[1]))
			// ADOPTING A VALUE THAT ALREADY CARRIES THE TYPE IS THE IDENTITY, and
			// saying so here is not a micro-optimisation: without it every write to
			// a declared-type slot re-boxed, and `long i; while (...) { i = i + 1; }`
			// allocated a fresh 64-bit cell per iteration - measured at +15.8% on
			// tests/bench/mod.cs against a 1.94% spread when the local-write site
			// started adopting (docs/todo.md 1.3). A sized integer and a float box
			// are compared BY VALUE by strict_eq, so handing back the same value
			// rather than a copy is not observable. The twin is js_csadoptty in
			// languages/lib/csharp-rt.metajs.
			if ty == "float" {
				if f, isFlo := v.(jsJFlo); isFlo && f.sty == floCSF {
					return a[0]
				}
				if !csIsNumericV(v) {
					return a[0]
				}
				return w(jsJFlo{f: jvmFround(rt.toNumber(v)), sty: floCSF})
			}
			if ty == "double" || ty == "decimal" {
				if f, isFlo := v.(jsJFlo); isFlo && f.sty == floCS {
					return a[0]
				}
				if !csIsNumericV(v) {
					return a[0]
				}
				return w(jsJFlo{f: rt.toNumber(v), sty: floCS})
			}
			bits, ok := csTypeW[ty]
			if !ok || !csIsIntegral(v) {
				return a[0]
			}
			uns := csTypeU[ty]
			if g, isG := v.(jsGInt); isG {
				if g.w == bits && g.u == uns {
					return a[0]
				}
			} else if f, isF := v.(float64); isF && bits == 32 && !uns && f == math.Trunc(f) {
				return a[0]
			}
			return w(csConvTo(rt, v, bits, uns))
		}
		// js_csadoptmem(obj, name, v): the declared type a WRITE to a member adopts
		// (ECMA-334 10.2.3 at 12.21.2's simple assignment), read off the '__mty'
		// table the class descriptor carries. A qualified write - `c.D = 3` on a
		// `public double D;` - has no static receiver type here, so the type has to
		// come off the object at run time; this is the only site that needs it, and
		// the emitter calls it only for member NAMES some class declares with an
		// adopting type (docs/todo.md 1.3). The twin is csAdoptMemW in
		// languages/csharp-interpreter.abnf and js_csadoptmem in
		// languages/lib/csharp-rt.metajs.
		m["js_csadoptmem"] = func(a []uint64) uint64 {
			o, ok := u(a[0]).(*jsObject)
			if !ok {
				return a[2]
			}
			name := rt.toString(u(a[1]))
			// A qualified STATIC write ('C.S = 3') has the class DESCRIPTOR as its
			// receiver, and the descriptor is where __mty lives - so the walk starts
			// at the object itself there, and at its class for an instance.
			cls, _ := o.props["__class"]
			if isCls, ok2 := o.props["__isclass"]; ok2 && isCls == true {
				cls = o
			}
			ad := m["js_csadoptty"]
			for n := 0; n < 64 && cls != nil; n++ {
				clsObj, isObj := cls.(*jsObject)
				if !isObj {
					return a[2]
				}
				if mty, found := clsObj.props["__mty"]; found {
					if mtyObj, isO := mty.(*jsObject); isO {
						if ty, has := mtyObj.props[name]; has {
							if el := csElemAdoptName(rt.toString(ty)); el != "" {
								return m["js_csadoptarr"]([]uint64{a[2], w(el)})
							}
							return ad([]uint64{a[2], w(rt.toString(ty))})
						}
					}
				}
				cls = clsObj.props["__super"]
			}
			return a[2]
		}
		// js_csadoptarr(v, ty): `double[] a = {3, 4}` types the ELEMENTS, so
		// a[0] / 2 is 1.5. IN PLACE, not into a copy - `int[] x = {1}; int[] y =
		// x;` must still alias. One level only; `double[][]` adopts nothing.
		m["js_csadoptarr"] = func(a []uint64) uint64 {
			arr, ok := u(a[0]).(*jsArray)
			if !ok {
				return a[0]
			}
			ad := m["js_csadoptty"]
			for i := range arr.elems {
				arr.elems[i] = u(ad([]uint64{w(arr.elems[i]), a[1]}))
			}
			return a[0]
		}
		// js_csis(v, t) is `v is T` / `v as T` / a type pattern. ECMA-334 12.12.12
		// asks for the RUN-TIME type of v - there is no implicit conversion in a
		// type test, which is why `object o = 5; o is long` is false in C# - and
		// the shared js_is_type cannot answer that for a number: it says "is it an
		// integral number" for all eight integral names at once and "is it a
		// number" for all three floating ones, so `1.5f is float`, `1.5 is
		// double`, `5L is long` and `(byte)3 is byte` were all FALSE while
		// `5 is long` was true (docs/todo.md 1.9).
		//
		// This value model does carry the answer: a plain number is an `int`, a
		// jsGInt is every other integral type and carries its width and
		// signedness, and a jsJFlo's style byte says double (2) or float (5).
		// `is decimal` is answered by the `double` arm, because a decimal IS a
		// style-2 double box here - the one residue this cannot fix without a
		// decimal representation of its own. Every non-numeric name falls through
		// to the shared probe unchanged, so a user class, a string, a char, an
		// array and `object` behave exactly as before.
		baseIs := m["js_is_type"]
		m["js_csis"] = func(a []uint64) uint64 {
			t := rt.toString(u(a[1]))
			if i := strings.IndexByte(t, '<'); i >= 0 {
				t = t[:i]
			}
			opt := strings.HasSuffix(t, "?")
			t = strings.TrimSuffix(t, "?")
			bits, uns, isInt := csIsTyWidth(t)
			isFlo := t == "double" || t == "Double" || t == "decimal" || t == "Decimal"
			isF32 := t == "float" || t == "Single"
			// `char` is answered here too: the shared probe's "Char" arm answers
			// for a PLAIN NUMBER (the Java/Go reading, where a char is one), so
			// `1 is char` came back TRUE in this half and FALSE in the
			// interpreter, which has a char box. A pre-existing halves
			// divergence --cross never reached because no test program asks.
			isChr := t == "char" || t == "Char"
			if !isInt && !isFlo && !isF32 && !isChr {
				return baseIs(a)
			}
			if isChr {
				switch u(a[0]).(type) {
				case nil, jsUndefT, jsNullT:
					return boolH(opt)
				case jsChar:
					return boolH(true)
				}
				return boolH(false)
			}
			switch v := u(a[0]).(type) {
			case nil, jsUndefT, jsNullT:
				return boolH(opt)
			case float64:
				return boolH(isInt && bits == 32 && !uns && v == math.Trunc(v))
			case jsGInt:
				return boolH(isInt && v.w == bits && v.u == uns)
			case jsJFlo:
				if v.sty == floCSF {
					return boolH(isF32)
				}
				return boolH(isFlo)
			}
			return boolH(false)
		}
		// An integer literal, passed as DIGITS because emitNum would already have
		// rounded 9223372036854775807 to 9223372036854776000 on the way into the
		// module. Its TYPE follows its suffix and its value (ECMA-334 6.4.5.3):
		// with no suffix it is the first of int, uint, long, ulong that holds it;
		// U makes it uint or ulong; L long or ulong; UL ulong.
		m["js_cslit"] = func(a []uint64) uint64 {
			s := rt.toString(u(a[0]))
			radix := uint64(jsToInt(rt.toNumber(u(a[1]))))
			hasU, hasL := rt.truthy(u(a[2])), rt.truthy(u(a[3]))
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
					continue
				}
				if d >= radix {
					continue
				}
				acc = acc*radix + d
			}
			p := int64(acc)
			fitsUInt := acc <= 0xffffffff
			fitsInt := acc <= 0x7fffffff
			fitsLong := acc <= 0x7fffffffffffffff
			switch {
			case hasU && hasL:
				return w(jsGInt{v: p, w: 64, u: true})
			case hasU:
				if fitsUInt {
					return w(jsGInt{v: p, w: 32, u: true})
				}
				return w(jsGInt{v: p, w: 64, u: true})
			case hasL:
				if fitsLong {
					return w(jsGInt{v: p, w: 64})
				}
				return w(jsGInt{v: p, w: 64, u: true})
			case fitsInt:
				return rt.wrapNum(float64(int32(p)))
			case fitsUInt:
				return w(jsGInt{v: p, w: 32, u: true})
			case fitsLong:
				return w(jsGInt{v: p, w: 64})
			}
			return w(jsGInt{v: p, w: 64, u: true})
		}
		// -x and ~x, at the UNARY promoted type (ECMA-334 12.4.7.2): a long keeps
		// its type, a uint becomes a LONG under '-' (because -uint.MaxValue does
		// not fit in a uint) and keeps uint under '~', and everything narrower
		// becomes an int.
		m["js_csneg"] = func(a []uint64) uint64 {
			v := u(a[0])
			if f, ok := v.(jsJFlo); ok {
				return w(jsJFlo{f: -f.f, sty: f.sty})
			}
			if csSzW(v) == 64 {
				return w(csNorm(-giVal(rt, v), 64, csSzU(v)))
			}
			if csSzW(v) == 32 && csSzU(v) {
				return w(csNorm(-giVal(rt, v), 64, false))
			}
			return rt.wrapNum(float64(-int32(giVal(rt, v))))
		}
		m["js_csnot"] = func(a []uint64) uint64 {
			v := u(a[0])
			if csSzW(v) == 64 || (csSzW(v) == 32 && csSzU(v)) {
				return w(csNorm(^giVal(rt, v), csSzW(v), csSzU(v)))
			}
			return rt.wrapNum(float64(^int32(giVal(rt, v))))
		}
		// ++ / -- step at the operand's OWN type and wrap there (ECMA-334 12.8.15:
		// `x++` is `x = (T)(x + 1)`), so `byte b = 255; b++` is 0.
		m["js_csstep"] = func(a []uint64) uint64 {
			v, d := u(a[0]), giVal(rt, u(a[1]))
			switch t := v.(type) {
			case jsChar:
				return w(jsChar{code: int32((int64(t.code) + d) & 65535)})
			case jsJFlo:
				// ++/-- keeps the operand's own type, WIDTH included: a float
				// steps as a float, so `float f = 16777216f; f++` stays
				// 1.6777216E7.
				if jvmIs32(t.sty) {
					return w(jsJFlo{f: jvmFround(t.f + float64(d)), sty: t.sty})
				}
				return w(jsJFlo{f: t.f + float64(d), sty: t.sty})
			case jsGInt:
				return w(csConvTo(rt, rt.csArith("+", t, float64(d)), t.w, t.u))
			}
			return rt.wrapNum(float64(int32(giVal(rt, v)) + int32(d)))
		}
		// The result of a COMPOUND assignment: the operation, then the implicit
		// cast back to the LEFT operand's type (ECMA-334 12.21.4: `x op= y` is
		// `x = (T)(x op y)`), so `byte b = 250; b += 10` is (byte)260 == 4.
		m["js_csnarrowlike"] = func(a []uint64) uint64 {
			l, v := u(a[0]), u(a[1])
			switch t := l.(type) {
			case jsJFlo:
				// A compound assignment carries an implicit cast back to the
				// LEFT operand's type (ECMA-334 12.21.4), so `float f = 1.1f;
				// f += 0.1` is (float)(f + 0.1) and NOT a double.
				if jvmIs32(t.sty) {
					return w(jsJFlo{f: jvmFround(rt.toNumber(v)), sty: t.sty})
				}
				if f, ok := v.(jsJFlo); ok {
					return w(f)
				}
				return w(jsJFlo{f: rt.toNumber(v), sty: t.sty})
			case jsChar:
				return w(jsChar{code: int32(giVal(rt, v) & 65535)})
			case jsGInt:
				return w(csConvTo(rt, v, t.w, t.u))
			case float64:
				if csIsIntegral(v) || jvmIsFlo(v) {
					return w(csConvTo(rt, v, 32, false))
				}
			}
			return a[1]
		}
		// A (char) cast: narrow to 16 UNSIGNED bits and re-box.
		m["js_cschar"] = func(a []uint64) uint64 {
			v := u(a[0])
			if f, isFlo := v.(jsJFlo); isFlo {
				return w(jsChar{code: int32(giVal(rt, csConvTo(rt, f, 16, true)) & 65535)})
			}
			return w(jsChar{code: int32(giVal(rt, v) & 65535)})
		}
		// `a[lo..hi]`, C#'s range indexer, on a string or an array.
		//
		// This used to be emitted as js_goslice, the shared extern, and that is
		// GO's slice and not C#'s: a Go string is a byte sequence, so js_goslice
		// takes a BYTE window, while System.Range on a System.String is
		// Substring - UTF-16 code units (ECMA-334 12.8.12 / String.Substring).
		// `"日本語"[0..1]` therefore answered a lone replacement character under
		// llvm.Run and "日" in the interpreter and in the clang-built executable,
		// which is a disagreement ./test.sh cannot see (each engine matches
		// itself) and --cross did not reach. One name was carrying two
		// specifications; C# gets its own, and js_goslice stays Go's.
		//
		// The array arm COPIES, and an absent bound arrives as undefined and
		// reads as 0 / the length. The twin is js_csslice in
		// languages/lib/csharp-rt.metajs.
		m["js_csslice"] = func(a []uint64) uint64 {
			bound := func(h uint64, dflt int) int {
				v := u(h)
				if _, isU := v.(jsUndefT); isU {
					return dflt
				}
				return int(rt.toNumber(v))
			}
			if str, isStr := u(a[0]).(string); isStr {
				n := rt.strLen(str)
				lo, hi := bound(a[1], 0), bound(a[2], n)
				if lo < 0 || hi > n || lo > hi {
					rt.fail("slice bounds [%d:%d] out of range", lo, hi)
				}
				return w(rt.strRange(str, lo, hi))
			}
			arr, isArr := u(a[0]).(*jsArray)
			if !isArr {
				rt.fail("slicing a %s", rt.typeOf(u(a[0])))
			}
			lo, hi := bound(a[1], 0), bound(a[2], len(arr.elems))
			if lo < 0 || hi > len(arr.elems) || lo > hi {
				rt.fail("slice bounds [%d:%d] out of range", lo, hi)
			}
			out := &jsArray{}
			out.elems = append(out.elems, arr.elems[lo:hi]...)
			return w(out)
		}
		// C#'s '+'. Identical to the shared js_csadd - string concat with C#'s
		// null rule, float add when a double is involved - except that two
		// INTEGRAL operands add at their promoted type and wrap to it, instead of
		// through a double that rounds above 2^53. js_csadd itself is a shared
		// extern in jsrt.go and is left exactly as it was.
		m["js_csadd2"] = func(a []uint64) uint64 {
			l, r := u(a[0]), u(a[1])
			_, ls := l.(string)
			_, rs := r.(string)
			if ls || rs {
				return rt.wrapStr(strConcat(rt.csString(l), rt.csString(r)))
			}
			if jvmIsFlo(l) || jvmIsFlo(r) {
				// jvmArithStyle, not jvmStyleOf: the two differ only when a
				// 32-bit box is involved, and there the WIDER type wins
				// (ECMA-334 12.4.7.3), which jvmStyleOf's left-operand rule gets
				// backwards for `1.0f + 1.0`. Both operands convert to float
				// FIRST when the promoted type is float - rounding only the sum
				// is a different answer for `0.1f + 16777217`.
				sty := jvmArithStyle(l, r)
				if jvmIs32(sty) {
					return w(jsJFlo{f: jvmFround(jvmFround(rt.toNumber(l)) + jvmFround(rt.toNumber(r))), sty: sty})
				}
				return w(jsJFlo{f: rt.toNumber(l) + rt.toNumber(r), sty: sty})
			}
			if csIsIntegral(l) && csIsIntegral(r) {
				return w(rt.csArith("+", l, r))
			}
			return rt.wrapNum(rt.toNumber(l) + rt.toNumber(r))
		}
		// == / != with a sized integer on either side: two integers compare BY
		// VALUE at their promoted type, so a uint and a long holding the same
		// number are equal and two boxes holding 5 are not two distinct objects.
		m["js_cseq"] = func(a []uint64) uint64 {
			l, r := u(a[0]), u(a[1])
			// THE FLOAT WIDTH RIDES HERE rather than on a new js_csvchareq
			// extern the way java's does, because C#'s emitter already wraps
			// every `==` / `!=` in js_cseq and hands it the js_jchareq answer to
			// override. ECMA-334 12.4.7.3 converts the INT operand to a float
			// first, so `16777216f == 16777217` is true where a bare value
			// comparison says false. A float against a DOUBLE is not this case -
			// the pair promotes to double, jvmFloPromotes answers false, and the
			// caller's answer stands.
			// The INTEGRAL fast path stays FIRST and pays nothing for the
			// width, which is the lesson 8f43e84 bought at +5.32% in java.
			if csIsIntegral(l) && csIsIntegral(r) && !jvmIsFlo(l) && !jvmIsFlo(r) {
				if _, lb := l.(jsGInt); lb {
					return boolH(rt.csCmp("==", l, r))
				}
				if _, rb := r.(jsGInt); rb {
					return boolH(rt.csCmp("==", l, r))
				}
				return a[2]
			}
			if jvmFloPromotes(l, r) {
				return boolH(jvmFround(rt.toNumber(l)) == jvmFround(rt.toNumber(r)))
			}
			return a[2] // the js_jchareq / $valeq result the caller computed
		}
		// ----- method-group conversion (docs/todo.md 1.1) -----
		// The Go twin of js_csmg in languages/lib/csharp-rt.metajs and of
		// csMethodGroup in languages/csharp-interpreter.abnf; the comment in the
		// interpreter is the specification half (ECMA-334 10.8, 20.1, 20.5).
		// `bound` is the forwarder the EMITTER built - layer 2 cannot build one,
		// having no rest parameter - and this decides only whether the group
		// exists, whether it needs binding at all, and which object it denotes.
		m["js_csmg"] = func(a []uint64) uint64 {
			recv := u(a[0])
			name := rt.toString(u(a[1]))
			o, ok := recv.(*jsObject)
			if !ok {
				return w(jsUndef)
			}
			if b, isCls := o.props["__isclass"].(bool); isCls && b {
				// A class descriptor: its OWN statics were answered by the js_get
				// that got here, so this walk is what reaches an inherited one.
				for cls := interface{}(o); cls != nil; {
					clsObj, ok := cls.(*jsObject)
					if !ok {
						break
					}
					if mth, ok := clsObj.props[name]; ok && isCallable(mth) {
						return w(mth)
					}
					cls = clsObj.props["__super"]
				}
				return w(jsUndef)
			}
			found := false
			for cls := o.props["__class"]; cls != nil && !found; {
				clsObj, ok := cls.(*jsObject)
				if !ok {
					break
				}
				if mth, ok := clsObj.props[name]; ok && isCallable(mth) {
					found = true
					break
				}
				cls = clsObj.props["__super"]
			}
			if !found {
				return w(jsUndef)
			}
			// MEMOISED so that `E -= s.Got` removes what `E += s.Got` added:
			// ECMA-334 20.5 makes two delegates equal when target and method are
			// equal, and a delegate is compared by object identity here.
			key := "__mg_" + name
			if had, ok := o.props[key]; ok && had != jsUndef {
				return w(had)
			}
			o.set(key, u(a[2]))
			return a[2]
		}
		// ----- `ref` / `out` arguments: the REFERENCE BOX (docs/todo.md 1.1) -----
		//
		// ECMA-334 12.6.2.3.3 makes a `ref` argument's parameter an ALIAS for the
		// argument's variable. Nothing in this value model can alias a scope slot,
		// so csharp-to-llvm-ir.abnf builds `{__csref, rd, wb}` - two emitted
		// closures over the place - at the call site, and emits these two probes
		// around every read and write of a `ref` PARAMETER's name.
		//
		// Both are total: a value that is not a box passes straight through, which
		// is what lets the emitter gate on the NAME alone and emit no basic blocks.
		// The twins are js_csrefrd / js_csrefwr in languages/lib/csharp-rt.metajs,
		// and csGetVar / csSetVar in languages/csharp-interpreter.abnf.
		m["js_csrefrd"] = func(a []uint64) uint64 {
			v := u(a[0])
			if o, ok := v.(*jsObject); ok {
				if b, isRef := o.props["__csref"].(bool); isRef && b {
					return w(rt.call(o.props["rd"], jsUndef, []interface{}{}))
				}
			}
			return a[0]
		}
		// js_csrefwr(cur, v): writes v THROUGH the box `cur` and answers the box, so
		// the scope store the emitter puts after it re-stores the same handle; for a
		// plain slot it answers v and that store is the ordinary one.
		m["js_csrefwr"] = func(a []uint64) uint64 {
			cur := u(a[0])
			if o, ok := cur.(*jsObject); ok {
				if b, isRef := o.props["__csref"].(bool); isRef && b {
					rt.call(o.props["wb"], jsUndef, []interface{}{u(a[1])})
					return a[0]
				}
			}
			return a[1]
		}
		// js_csdfld(recv, name): the DELEGATE stored in a member of that name, or
		// undefined. `k.E(5)` on a delegate field parses as a method call and C#
		// resolves it as the invocation of the field's value (ECMA-334 12.8.10.2).
		// Answers undefined for every receiver that is not a class instance, so the
		// call site falls back to js_mcall exactly as it did before.
		m["js_csdfld"] = func(a []uint64) uint64 {
			o, ok := u(a[0]).(*jsObject)
			if !ok {
				return w(jsUndef)
			}
			cls, hasCls := o.props["__class"]
			if !hasCls {
				return w(jsUndef)
			}
			name := rt.toString(u(a[1]))
			v, has := o.props[name]
			if !has || v == jsUndef {
				// A property with a get accessor body may hold one too.
				for c := cls; c != nil; {
					clsObj, ok := c.(*jsObject)
					if !ok {
						break
					}
					if g, ok := clsObj.props["__get_"+name]; ok && isCallable(g) {
						v = rt.call(g, jsUndef, []interface{}{o})
						break
					}
					c = clsObj.props["__super"]
				}
			}
			if cspIsDeleg(v) {
				return w(v)
			}
			return w(jsUndef)
		}
		// One of the integral type NAMES as an expression primary.
		m["js_cstype"] = func(a []uint64) uint64 { return w(rt.csTypeObject(rt.toString(u(a[0])))) }
		// The whole Console object, built here rather than in the grammar so that
		// WriteLine / Write render through cspStr instead of through the shared
		// println host function (which is what leaked Go's %v).
		m["js_csconsole"] = func(a []uint64) uint64 {
			o := newJSObject()
			o.set("WriteLine", jsHostFunc("js_csprint", func(rt *jsrt, this uint64, args []interface{}) interface{} {
				fmt.Fprint(outWriter, wtf8Clean(rt.cspStr(argAt(args, 0)))+"\n")
				return jsUndef
			}))
			o.set("Write", jsHostFunc("js_cswrite", func(rt *jsrt, this uint64, args []interface{}) interface{} {
				fmt.Fprint(outWriter, wtf8Clean(rt.cspStr(argAt(args, 0))))
				return jsUndef
			}))
			return rt.wrap(o)
		}

		// The rendering step of a string CONVERSION, used by '+', by an
		// interpolation hole and by .ToString(). It converts only the values whose
		// C# string form differs from the shared runtime's - a bool, an array and a
		// class instance - and passes everything else through UNCHANGED, so that
		// `1 + 2` stays an integer addition and a boxed double stays a double. The
		// caller wraps the result in js_csadd, which supplies C#'s null rule and
		// the numeric cases.
		m["js_csstr"] = func(a []uint64) uint64 {
			v := u(a[0])
			switch t := v.(type) {
			case bool:
				return rt.wrapStr(cspBoolStr(t))
			case *jsArray:
				return rt.wrapStr(cspArrayName(t))
			case *jsObject:
				if s, ok := rt.cspObjStr(t, 0); ok {
					return rt.wrapStr(s)
				}
			}
			return a[0]
		}
	})
}

// cspBoolStr is System.Boolean.ToString(): Boolean.TrueString / Boolean.FalseString.
func cspBoolStr(b bool) string {
	if b {
		return "True"
	}
	return "False"
}

// cspStr is the full C# string conversion - the twin of csStr in
// csharp-interpreter.abnf.
func (rt *jsrt) cspStr(v interface{}) string { return rt.cspRender(v, 0) }

func (rt *jsrt) cspRender(v interface{}, depth int) string {
	if depth > 32 {
		return "..."
	}
	switch t := v.(type) {
	case jsNullT, jsUndefT:
		return "" // Console.WriteLine(null) writes an empty line.
	case bool:
		return cspBoolStr(t)
	case string:
		return t
	case *jsArray:
		return cspArrayName(t)
	case *jsObject:
		if s, ok := rt.cspObjStr(t, depth); ok {
			return s
		}
	}
	return rt.toString(v) // numbers, char, the boxed double (csFloStr), everything else.
}

// cspObjStr renders the object shapes the two C# grammars build. It reports false
// for a shape it does not know, which keeps the generic rendering both halves
// already agreed on.
func (rt *jsrt) cspObjStr(o *jsObject, depth int) (string, bool) {
	// A user ToString() override wins over every built-in rendering, exactly as
	// the virtual System.Object.ToString() does in C#.
	if mth, ok := cspFindToString(o); ok {
		return rt.cspRender(rt.call(mth, jsUndef, []interface{}{o}), depth+1), true
	}
	// The type descriptor itself.
	if _, ok := o.props["__isclass"]; ok {
		if n, ok := o.props["__name"].(string); ok {
			return n, true
		}
	}
	// An instance: its type's name (see the simplification note in the header).
	if cls, ok := o.props["__class"].(*jsObject); ok {
		if n, ok := cls.props["__name"].(string); ok && n != "" {
			return n, true
		}
	}
	return "", false
}

// cspIsDeleg: a delegate value is a closure, or a MULTICAST invocation list whose
// first entry is one. The twins are csIsDelegVal in languages/csharp-interpreter.abnf
// and in languages/lib/csharp-rt.metajs.
func cspIsDeleg(v interface{}) bool {
	if v == nil {
		return false
	}
	if isCallable(v) {
		return true
	}
	if arr, ok := v.(*jsArray); ok && len(arr.elems) > 0 {
		return isCallable(arr.elems[0])
	}
	return false
}

// cspFindToString walks the __class / __super chain for a user-declared ToString,
// the same walk the method dispatch does in both C# grammars.
func cspFindToString(o *jsObject) (interface{}, bool) {
	for cls := o.props["__class"]; cls != nil; {
		clsObj, ok := cls.(*jsObject)
		if !ok {
			return nil, false
		}
		if mth, ok := clsObj.props["ToString"]; ok && isCallable(mth) {
			return mth, true
		}
		cls = clsObj.props["__super"]
	}
	return nil, false
}

// cspArrayName is Object.ToString() of an array: the element type's full name plus
// "[]". The DECLARED element type is not available in this value model, so it is
// inferred from the elements - see the header for the exact rule.
func cspArrayName(a *jsArray) string {
	if len(a.elems) == 0 {
		return "System.Object[]"
	}
	name := ""
	for _, e := range a.elems {
		n := cspElemName(e)
		if n == "" || (name != "" && n != name) {
			return "System.Object[]"
		}
		name = n
	}
	return name + "[]"
}

func cspElemName(e interface{}) string {
	switch t := e.(type) {
	case jsGInt:
		// A sized integer carries its own type, so long[] / uint[] / byte[] are
		// the one array kind whose element type is not a guess.
		switch {
		case t.w == 64 && t.u:
			return "System.UInt64"
		case t.w == 64:
			return "System.Int64"
		case t.w == 32 && t.u:
			return "System.UInt32"
		case t.w == 16 && t.u:
			return "System.UInt16"
		case t.w == 16:
			return "System.Int16"
		case t.u:
			return "System.Byte"
		}
		return "System.SByte"
	case jsJFlo:
		// A `float` is System.Single and a `double` System.Double, which the
		// style byte now tells apart (ECMA-334 8.3.7). `decimal` is still a
		// double box here, so a decimal[] answers System.Double[] - a
		// pre-existing approximation, unchanged.
		if jvmIs32(t.sty) {
			return "System.Single"
		}
		return "System.Double"
	case string:
		return "System.String"
	case bool:
		return "System.Boolean"
	case jsChar:
		return "System.Char"
	case float64:
		return "System.Int32"
	}
	return ""
}
