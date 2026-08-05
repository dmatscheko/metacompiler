package abnf

// Floating point for the STATICALLY TYPED languages' compiler grammars
// (languages/java-to-llvm-ir.abnf, kotlin-to-llvm-ir.abnf, go-to-llvm-ir.abnf,
// csharp-to-llvm-ir.abnf).
//
// The handle runtime models a number as one JS double, which is enough for a
// dynamically typed language but not for a statically typed one: `1.0 / 3.0` and
// `1 / 3` are the SAME two operands there, and Java answers 0.3333333333333333 for
// the first and 0 for the second. The compiler used to lower every arithmetic
// operator onto js_sub/js_mul/js_div followed by a 32 bit wrap, so the float half
// of the language was silently integer arithmetic: 2.5 * 1.5 was 3, 1.0 / 3.0 was
// 0, println(1.0) printed "1" and there was no Infinity, no NaN and no -0.0.
//
// jsJFlo is the fix and it mirrors what the interpreter grammar does with its
// {__flo} box (and what jsPhpFlo does for PHP, and jsFlo for Ruby): a
// floating-point value is BOXED, so the runtime can still tell a double from an
// int. The invariant is
//
//	plain number  ==  an integral type (int / long / short / byte)
//	jsJFlo        ==  a double / float
//
// and every operator that meets a jsJFlo evaluates in floating point and answers a
// jsJFlo (JLS 5.6.2, binary numeric promotion).
//
// This file is append-only with respect to the rest of the runtime: nothing here
// runs unless a program calls one of the js_jf* externs, which only the Java and
// Kotlin compiler grammars emit. The exception is the jsJFlo box itself, which
// rt.truthy / rt.toNumber / rt.toString / rt.toGoNatural / rt.typeOf / rt.strictEq
// learn about in jsrt.go - each as one extra case for a type nothing else creates.

import (
	"math"
	"strconv"
	"strings"
)

// jsJFlo is a double (or float), boxed so that it stays distinguishable from an
// int. Ruby's jsFlo and PHP's jsPhpFlo are the same idea; those two languages
// have enough further numeric behaviour of their own to justify separate types,
// while the statically typed languages differ only in how a float PRINTS - so the
// box carries a style tag and the arithmetic is shared.
//
//	floJava  1.0 -> "1.0"   1e20 -> "1.0E20"    inf -> "Infinity"  (Java, Kotlin)
//	floGo    1.0 -> "1"     1e20 -> "1e+20"     inf -> "+Inf"      (Go)
//	floCS    1.0 -> "1"     1e20 -> "1E+20"     inf -> "Infinity"  (C#)
//	floJavaF the JAVA `float` - a 32 BIT WIDTH, not a fourth print style
//
// THE STYLE BYTE CARRIES THE WIDTH, and that is a deliberate choice with one
// consequence worth stating. `float` is not a print style: a Java float is a
// binary32, so `1.0f/3.0f` is 0.33333334 where the double is 0.3333333333333333,
// and every arithmetic result has to be ROUNDED to 24 significant bits. The value
// itself still travels as a float64 (every float is exactly representable as a
// double), so the only thing that has to go ON the box is the width - and the box
// already has a byte for a per-language tag. Reusing it means ZERO change to
// languages/lib/runtime.c, whose tag 14 cell stores cell.b as an opaque long and
// hands it back through jf_style_of.
//
// The consequence: THE FLOOR CANNOT RENDER A style-3 BOX. runtime.c's jf_text
// switches on the style and has no arm for 3, so it would print a float with the
// DOUBLE renderer. languages/lib/java-rt.metajs therefore intercepts every
// rendering and every arithmetic result of a style-3 box before the floor sees
// it (jvFloText / jvFloFix), and llvm.Run's side of that contract is jvmFloText /
// jvmArith here. If a fourth engine ever needs a float box, the floor is where
// the width should move to - see docs/todo.md 1.2.
const (
	floJava  = 0
	floGo    = 1
	floCS    = 2
	floJavaF = 3
)

type jsJFlo struct {
	f   float64
	sty uint8
}

func jvmMkFlo(f float64) jsJFlo { return jsJFlo{f: f} }

func jvmIsFlo(v interface{}) bool { _, ok := v.(jsJFlo); return ok }

// jvmFloText renders a boxed float in ITS OWN language's style.
func jvmFloText(v jsJFlo) string {
	switch v.sty {
	case floGo:
		return goFloStr(v.f)
	case floCS:
		return csFloStr(v.f)
	case floJavaF:
		return jvmFloStr32(v.f)
	}
	return jvmFloStr(v.f)
}

// jvmFround is the binary32 rounding every `float`-typed value carries: a Java
// float IS a double whose significand has been rounded to 24 bits, so the box
// only has to apply this after each operation and every other reader (compare,
// ==, min/max, the double widening) is correct without a second thought.
func jvmFround(x float64) float64 { return float64(float32(x)) }

// jvmStyleOf is the style an operation's result inherits: the one of whichever
// operand is a float (the left one when both are).
func jvmStyleOf(l, r interface{}) uint8 {
	if f, ok := l.(jsJFlo); ok {
		return f.sty
	}
	if f, ok := r.(jsJFlo); ok {
		return f.sty
	}
	return floJava
}

// jvmFloStr is java.lang.Double.toString: always a decimal point, the shortest
// digit run that round-trips, plain decimal for 1e-3 <= |d| < 1e7 and computerized
// scientific notation ("1.0E20") outside that range.
func jvmFloStr(d float64) string { return jvmFloStrN(d, 64) }

// jvmFloStr32 is java.lang.Float.toString, which is the SAME rule read at 24
// significant bits instead of 53: the shortest decimal that round-trips as a
// binary32. 1.0f/3.0f is "0.33333334" where the double is "0.3333333333333333",
// and Float.MIN_VALUE is "1.4E-45" - the two-significant-digit minimum again,
// against the ACTUAL value (1.401298464324817E-45), not a padded "1.0E-45".
// Measured against java 24.0.2.
func jvmFloStr32(d float64) string { return jvmFloStrN(d, 32) }

// bits is 64 for a double and 32 for a float; it is the ONLY difference between
// Double.toString and Float.toString, because strconv.FormatFloat's bitSize is
// what decides "shortest run of digits that round-trips".
func jvmFloStrN(d float64, bits int) string {
	if math.IsNaN(d) {
		return "NaN"
	}
	if math.IsInf(d, 1) {
		return "Infinity"
	}
	if math.IsInf(d, -1) {
		return "-Infinity"
	}
	if d == 0 {
		if math.Signbit(d) {
			return "-0.0"
		}
		return "0.0"
	}
	neg := d < 0
	a := math.Abs(d)
	var s string
	if a >= 1e-3 && a < 1e7 {
		s = strconv.FormatFloat(a, 'f', -1, bits)
		if !strings.ContainsRune(s, '.') {
			s += ".0"
		}
	} else {
		s = strconv.FormatFloat(a, 'E', -1, bits)
		// Java's Double.toString takes the SHORTEST decimal that round-trips,
		// except that it never uses fewer than TWO significant digits - and
		// when the shortest has one, the second digit is not a zero but the
		// closest two digit decimal to the actual value. For every normal
		// double those are the same thing (the value is within ~1e-16 of the
		// one digit form, a thousandth of a two digit step), so it shows only
		// among the SUBNORMALS, where the gap between neighbours is comparable
		// to the value: Double.MIN_VALUE is 4.9406564584124654E-324, so Java
		// renders it "4.9E-324" where appending ".0" to the shortest gives
		// "5.0E-324". Measured against real java 24.0.2; 254 lines of the
		// 17,674 line java probe (docs/runtime-next-plan.md part 3).
		// FormatFloat with precision 1 IS "the closest two digit decimal",
		// carry included (9.99e-321 -> "1.0E-320"), so the one digit case is
		// simply re-formatted rather than padded.
		if i := strings.IndexByte(s, 'E'); i >= 0 && !strings.ContainsRune(s[:i], '.') {
			s = strconv.FormatFloat(a, 'E', 1, bits)
		}
		// Go writes "1E+20" / "1.5E-08"; Java writes "1.0E20" / "1.5E-8".
		mant, exp := s, ""
		if i := strings.IndexByte(s, 'E'); i >= 0 {
			mant, exp = s[:i], s[i+1:]
		}
		if !strings.ContainsRune(mant, '.') {
			mant += ".0"
		}
		sign := ""
		if strings.HasPrefix(exp, "-") {
			sign, exp = "-", exp[1:]
		} else {
			exp = strings.TrimPrefix(exp, "+")
		}
		exp = strings.TrimLeft(exp, "0")
		if exp == "" {
			exp = "0"
		}
		s = mant + "E" + sign + exp
	}
	if neg {
		return "-" + s
	}
	return s
}

// jvmNumEq is Java's == with a double on one side: the other side has to be a
// number too - a plain one, another double, a char, or a SIZED INTEGER - and then
// the comparison is numeric, so 1.0 == 1 is true and NaN == NaN is false.
//
// THE jsGInt ARM IS A FIX. It was missing, and its absence was called
// deliberate at three sites ("a box against a jsGInt is false"); it is a DEFECT.
// `long x = 0; double y = 0.0; x == y` is TRUE in both languages - JLS 15.21.1
// and ECMA-334 12.12.9 promote the integral operand to double first - and java
// 24.0.2 confirms it. Both COMPILER halves said false (this, and the floor's
// jf_num_eq through lib/runtime-jvm.metajs's rtjStrictEq, which takes the same
// fix) while both INTERPRETER halves said true; --cross never saw it because
// nothing in the matrix compared a long to a double. giEq (jsrtint.go:283)
// already read the pair this way, and rt.strictEq short-circuited past it.
func jvmNumEq(f float64, other interface{}) bool {
	switch t := other.(type) {
	case jsJFlo:
		return f == t.f
	case float64:
		return f == t
	case jsChar:
		return f == float64(t.code)
	case jsGInt:
		// The unsigned 64 bit reading needs its own path, exactly as giFloat says:
		// int64(-1) read as a uint64 is 1.8446744073709552e19.
		if t.u && t.w == 64 {
			return f == float64(uint64(t.v))
		}
		return f == float64(t.v)
	}
	return false
}

// jvmArith is one binary arithmetic operator of the Java subset. A double on
// either side makes the whole operation floating point and its result a double;
// two integral operands keep the 32 bit wrap the compiler emitted before (the
// int-width work is a separate concern, so this reproduces it exactly).
func (rt *jsrt) jvmArith(op byte, l, r interface{}) interface{} {
	a, b := rt.toNumber(l), rt.toNumber(r)
	sty := jvmArithStyle(l, r)
	// BOTH OPERANDS CONVERT FIRST, and rounding only the result is not the same
	// answer: `float f = 0.1f; f + 16777217` is 1.6777216E7 in java, because the
	// int becomes a float (JLS 5.6.2 binary numeric promotion) BEFORE the add.
	// Rounding once at the end gives 1.6777218E7 - measured against java 24.0.2,
	// 156 rows of a 1,231-value probe.
	if sty == floJavaF {
		a, b = jvmFround(a), jvmFround(b)
	}
	var x float64
	switch op {
	case '+':
		x = a + b
	case '-':
		x = a - b
	case '*':
		x = a * b
	case '/':
		x = a / b
	case '%':
		x = math.Mod(a, b)
	}
	if jvmIsFlo(l) || jvmIsFlo(r) {
		if sty == floJavaF {
			x = jvmFround(x)
		}
		return jsJFlo{f: x, sty: sty}
	}
	return float64(rt.toInt32(x))
}

// jvmFloCmpPair converts a comparison's operands the same way: `16777217 ==
// 16777216f` is TRUE in java, because the int is converted to a float first.
// It answers the pair unchanged unless the promoted type is float.
func jvmFloCmpPair(rt *jsrt, l, r interface{}) (interface{}, interface{}) {
	if !jvmFloPromotes(l, r) {
		return l, r
	}
	return jsJFlo{f: jvmFround(rt.toNumber(l)), sty: floJavaF},
		jsJFlo{f: jvmFround(rt.toNumber(r)), sty: floJavaF}
}

// jvmFloPromotes is "these two operands promote to FLOAT" - one is a Java float
// box, the other is numeric, and neither is a double. Both operands are numeric
// by construction in a program javac accepts, but `==` also reaches this with a
// reference on one side, and that pair must keep the identity comparison.
func jvmFloPromotes(l, r interface{}) bool {
	if !jvmIsFlo(l) && !jvmIsFlo(r) {
		return false
	}
	if jvmArithStyle(l, r) != floJavaF {
		return false
	}
	return jvmFloNumeric(l) && jvmFloNumeric(r)
}

func jvmFloNumeric(v interface{}) bool {
	switch v.(type) {
	case jsJFlo, float64, jsGInt, jsChar:
		return true
	}
	return false
}

// jvmArithStyle is JLS 5.6.2 read for the WIDTH as well as the style: `float op
// float` is a float, and `float op double` is a DOUBLE - the wider type wins,
// which jvmStyleOf's left-operand rule gets backwards for `1.0f + 1.0`. Every
// other pairing is unchanged, so kotlin, C# and Go (which never build a
// floJavaF box) see exactly the old answer.
func jvmArithStyle(l, r interface{}) uint8 {
	lf, lok := l.(jsJFlo)
	rf, rok := r.(jsJFlo)
	if lok && rok {
		if lf.sty == floJavaF {
			return rf.sty // float+float -> float, float+double -> double
		}
		return lf.sty
	}
	if lok {
		return lf.sty
	}
	if rok {
		return rf.sty
	}
	return floJava
}

// jvmArithRaw is the floor's own tag-14 arithmetic - jf_arith in
// languages/lib/runtime.c - which knows nothing about a float width: the style
// is jf_style_of's left-preferring one and there is no rounding step. It is what
// the `floOp` HOST GLOBAL has to answer, because that global's contract is "the
// floor's own arm of this", and lib/java-rt.metajs applies the Java width rule
// on top of it exactly as jvmArith does above.
func (rt *jsrt) jvmArithRaw(op byte, l, r interface{}) interface{} {
	a, b := rt.toNumber(l), rt.toNumber(r)
	var x float64
	switch op {
	case '+':
		x = a + b
	case '-':
		x = a - b
	case '*':
		x = a * b
	case '/':
		x = a / b
	case '%':
		x = math.Mod(a, b)
	}
	if jvmIsFlo(l) || jvmIsFlo(r) {
		return jsJFlo{f: x, sty: jvmStyleOf(l, r)}
	}
	return float64(rt.toInt32(x))
}

// jvmTakeL is the OPERAND CHOICE of Math.max / Math.min, and the two cases a
// plain `>` / `<` cannot see. java.lang.Math documents both for the double
// overloads, and java 24.0.2 confirms each:
//
//	Math.min(-0.0, 0.0) is -0.0 and Math.max(-0.0, 0.0) is 0.0 - "if one
//	argument is positive zero and the other negative zero, the result of min
//	is negative zero" (and of max, positive zero). -0.0 < 0.0 is FALSE, so a
//	bare comparison took the right operand and answered 0.0 for min.
//
//	Math.min(NaN, 1.0) and Math.max(NaN, 1.0) are both NaN - "if either value
//	is NaN, then the result is NaN". Every comparison against NaN is false, so
//	a bare `<` took the right operand and answered 1.0.
//
// Everything else is the ordinary comparison, tie going to the right operand
// exactly as before.
func jvmTakeL(a, b float64, wantMax bool) bool {
	if a != a {
		return true // the left one is the NaN
	}
	if b != b {
		return false // the right one is
	}
	if a == 0 && b == 0 {
		// Both zero and only the sign bit tells them apart: min wants the
		// negative one, max the positive one.
		if wantMax {
			return math.Signbit(b)
		}
		return math.Signbit(a)
	}
	if wantMax {
		return a > b
	}
	return a < b
}

// jvmMinMax picks one operand, but the DOUBLE overload of Math.max/min is
// selected as soon as one side is a double, so the answer is a double then
// (Math.max(1.5, 2) is 2.0).
func (rt *jsrt) jvmMinMax(l, r interface{}, takeL bool) interface{} {
	v := r
	if takeL {
		v = l
	}
	if jvmIsFlo(l) || jvmIsFlo(r) {
		// The overload is picked at the WIDER type, so Math.max(2.0f, 1.0) is the
		// double 2.0 and only Math.max(2.0f, 1.0f) is a float. For every language
		// but Java that is jvmStyleOf unchanged, because nothing else builds a
		// floJavaF box and the two agree on all three remaining styles.
		sty := jvmArithStyle(l, r)
		if f, ok := v.(jsJFlo); ok && f.sty == sty {
			return f
		}
		x := rt.toNumber(v)
		if sty == floJavaF {
			x = jvmFround(x)
		}
		return jsJFlo{f: x, sty: sty}
	}
	return v
}

// js_jflo32 is the ONE new extern the float width needs: `(float) e`, a
// `1.0f` literal and a `float x = e` declaration all lower to it, and its layer-2
// twin is js_jflo32 in languages/lib/java-rt.metajs. It is registered here rather
// than in abnf/jsrt.go's table because the width belongs to this file; the
// emitter wraps its operand in js_jvnum first, exactly as it does for js_jflo,
// so a Java `long` (a tag-13 cell marked UNSIGNED) reads signed.
func init() {
	rxExtraExterns = append(rxExtraExterns, func(rt *jsrt, m map[string]func(args []uint64) uint64) {
		// js_jvchareq is js_jchareq (abnf/jsrt.go) with the float promotion of
		// JLS 5.6.2 in front of it. It is a separate name because js_jchareq is
		// shared with kotlin, which has no float width.
		m["js_jvchareq"] = func(a []uint64) uint64 {
			x, y := rt.unwrap(a[0]), rt.unwrap(a[1])
			if jvmFloPromotes(x, y) {
				if jvmFround(rt.toNumber(x)) == jvmFround(rt.toNumber(y)) {
					return jsHTrue
				}
				return jsHFalse
			}
			return m["js_jchareq"](a)
		}
		m["js_jflo32"] = func(a []uint64) uint64 {
			return rt.wrap(jsJFlo{f: jvmFround(rt.toNumber(rt.unwrap(a[0]))), sty: floJavaF})
		}
	})
}

// jvmMathObject is the `Math` a compiled Java program sees. The handle runtime's
// own Math is JavaScript's and answers a plain number for everything, which would
// make Math.abs(-2.0) print "2" where Java prints "2.0". Only the three methods
// the Java subset documents are provided, which is exactly what
// java-interpreter.abnf's hostGlobals["Math"] has - the two halves must agree.
func jvmMathObject() *jsObject {
	o := newJSObject()
	o.set("abs", jsHostFunc("abs", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		v := argAt(args, 0)
		if f, ok := v.(jsJFlo); ok {
			return jsJFlo{f: math.Abs(f.f), sty: f.sty}
		}
		// A long is a 64-bit jsGInt box, and toInt32 truncates it: Math.abs of
		// 3000000000L answered -1294967296 and of Long.MAX_VALUE answered 0.
		// Negate in the box instead. Math.abs(Long.MIN_VALUE) is Long.MIN_VALUE
		// (JLS 15.15.4) and falls out of two's-complement negation unaided.
		if b, ok := v.(jsGInt); ok && b.w == 64 {
			if b.v < 0 {
				return jvBox(-b.v)
			}
			return b
		}
		return float64(rt.toInt32(math.Abs(rt.toNumber(v))))
	}))
	o.set("max", jsHostFunc("max", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		l, r := argAt(args, 0), argAt(args, 1)
		return rt.jvmMinMax(l, r, jvmTakeL(rt.toNumber(l), rt.toNumber(r), true))
	}))
	o.set("min", jsHostFunc("min", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		l, r := argAt(args, 0), argAt(args, 1)
		return rt.jvmMinMax(l, r, jvmTakeL(rt.toNumber(l), rt.toNumber(r), false))
	}))
	return o
}

// The js_jf* externals live in the big extern table in jsrt.go, next to
// js_jadd and js_jchareq; only the Java and Kotlin compiler grammars emit them.

// jfBindings adds the TEN flo* host globals that languages/lib/runtime.c seeds
// (host ids 51..60, `seed_root("flo", mk_host(51))` and the nine after it) to a
// program's global set. It is the jsJFlo twin of jsrtint.go's giBindings and it
// exists for the same reason, which f19a8ad's sint* work found the hard way: a
// layer-2 file written against the C floor could otherwise be linked natively
// but not run by llvm.Run, because `flo` would be "variable not defined" in the
// Go half. programJSBindings (jsrtint.go) calls both.
//
// Every function below is the jvmXxx above, so the three engines - the
// interpreter grammar's {__fl} box, llvm.Run through here, and tag 14 of the C
// floor - implement one specification.
func jfBindings(b map[string]interface{}) {
	// flo(v, style): a double from any numeric value, in one of the three print
	// styles. ONE argument, not the sint* pair - a MetaJS number already IS a
	// double, so nothing has to be carried in halves.
	b["flo"] = jsHostFunc("flo", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		sty := uint8(floJava)
		if len(args) > 1 {
			sty = uint8(jsToInt(rt.toNumber(args[1])))
		}
		return jsJFlo{f: rt.toNumber(argAt(args, 0)), sty: sty}
	})
	b["floIs"] = jsHostFunc("floIs", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return jvmIsFlo(argAt(args, 0))
	})
	b["floNum"] = jsHostFunc("floNum", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return rt.toNumber(argAt(args, 0))
	})
	b["floStyle"] = jsHostFunc("floStyle", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		v := argAt(args, 0)
		return float64(jvmStyleOf(v, v))
	})
	b["floStr"] = jsHostFunc("floStr", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		if f, ok := argAt(args, 0).(jsJFlo); ok {
			return jvmFloText(f)
		}
		return rt.toString(argAt(args, 0))
	})
	// One binary operator by INDEX - 0 + 1 - 2 * 3 / 4 % - which is the same
	// numbering si_apply / sintOp use for its first five. An index the table
	// does not have answers jvmArith's `var x float64` untouched, i.e. 0,
	// boxed when an operand is; stated rather than approximated so that the
	// halves agree even on a caller's mistake.
	b["floOp"] = jsHostFunc("floOp", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		ops := []byte{'+', '-', '*', '/', '%'}
		c := jsToInt(rt.toNumber(argAt(args, 0)))
		op := byte(0)
		if c >= 0 && c < len(ops) {
			op = ops[c]
		}
		return rt.jvmArithRaw(op, argAt(args, 1), argAt(args, 2))
	})
	// floEq IS THE FLOOR'S jf_num_eq (languages/lib/runtime.c), and the two have
	// to answer the same thing on every input or layer 2 and the twin diverge.
	// jf_num_eq has NO tag 13 arm, so a box against a SIZED INTEGER is false
	// here as well - even though jvmNumEq gained one, because that arm belongs to
	// the LANGUAGE's `==` (which promotes the integral operand to double first,
	// JLS 15.21.1 / ECMA-334 12.12.9) and not to the primitive. The promotion is
	// the CALLER's: lib/runtime-jvm.metajs's rtjStrictEq does it before it gets
	// here. tests/metajs-test-full.js's flo38 pins this contract.
	b["floEq"] = jsHostFunc("floEq", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		l, r := argAt(args, 0), argAt(args, 1)
		if f, ok := l.(jsJFlo); ok {
			if giIsInt(r) {
				return false
			}
			return jvmNumEq(f.f, r)
		}
		if f, ok := r.(jsJFlo); ok {
			if giIsInt(l) {
				return false
			}
			return jvmNumEq(f.f, l)
		}
		return rt.strictEq(l, r)
	})
	// Math.max / Math.min / Math.abs of jvmMathObject, whose double overload is
	// selected as soon as one side is a double (Math.max(1.5, 2) is 2.0).
	b["floMax"] = jsHostFunc("floMax", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		l, r := argAt(args, 0), argAt(args, 1)
		return rt.jvmMinMax(l, r, jvmTakeL(rt.toNumber(l), rt.toNumber(r), true))
	})
	b["floMin"] = jsHostFunc("floMin", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		l, r := argAt(args, 0), argAt(args, 1)
		return rt.jvmMinMax(l, r, jvmTakeL(rt.toNumber(l), rt.toNumber(r), false))
	})
	b["floAbs"] = jsHostFunc("floAbs", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		v := argAt(args, 0)
		if f, ok := v.(jsJFlo); ok {
			return jsJFlo{f: math.Abs(f.f), sty: f.sty}
		}
		// The same 64-bit arm jvmMathObject's own abs carries, for the same
		// reason: toInt32 TRUNCATES a long, so the plain wrap answered
		// -1294967296 for 3000000000L and 0 for Long.MAX_VALUE. Negate in the
		// box instead; two's-complement negation maps Long.MIN_VALUE to itself,
		// which is what JLS 15.15.4 asks for. Everything narrower than a long
		// widens to int before the call and keeps the 32-bit wrap.
		if bx, ok := v.(jsGInt); ok && bx.w == 64 {
			if !bx.u && bx.v < 0 {
				return jvBox(-bx.v)
			}
			return bx
		}
		return float64(rt.toInt32(math.Abs(rt.toNumber(v))))
	})
}

// goFloStr is Go's fmt %v for a float64: strconv.FormatFloat(f, 'g', -1, 64),
// which uses scientific notation when the decimal exponent is below -4 or at
// least 6, and spells the infinities "+Inf" / "-Inf".
func goFloStr(f float64) string {
	if math.IsNaN(f) {
		return "NaN"
	}
	if math.IsInf(f, 1) {
		return "+Inf"
	}
	if math.IsInf(f, -1) {
		return "-Inf"
	}
	return strconv.FormatFloat(f, 'g', -1, 64)
}

// csFloStr is C#'s double.ToString() under the invariant culture: the shortest
// round-tripping digits, scientific notation with a capital E outside
// [1e-5, 1e15), and no trailing ".0" for an integral value.
func csFloStr(f float64) string {
	if math.IsNaN(f) {
		return "NaN"
	}
	if math.IsInf(f, 1) {
		return "Infinity"
	}
	if math.IsInf(f, -1) {
		return "-Infinity"
	}
	if f == 0 {
		if math.Signbit(f) {
			return "-0"
		}
		return "0"
	}
	a := math.Abs(f)
	if a >= 1e-5 && a < 1e15 {
		return strconv.FormatFloat(f, 'f', -1, 64)
	}
	s := strconv.FormatFloat(f, 'E', -1, 64)
	// Go writes "1E+20"/"1.5E-08"; .NET writes "1E+20"/"1.5E-08" too (two-digit
	// minimum exponent), so the Go spelling already matches.
	return s
}
