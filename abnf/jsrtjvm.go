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
const (
	floJava = 0
	floGo   = 1
	floCS   = 2
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
	}
	return jvmFloStr(v.f)
}

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
func jvmFloStr(d float64) string {
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
		s = strconv.FormatFloat(a, 'f', -1, 64)
		if !strings.ContainsRune(s, '.') {
			s += ".0"
		}
	} else {
		s = strconv.FormatFloat(a, 'E', -1, 64)
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
			s = strconv.FormatFloat(a, 'E', 1, 64)
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
// number too (a plain one, another double or a char), and then the comparison is
// numeric - so 1.0 == 1 is true and NaN == NaN is false.
func jvmNumEq(f float64, other interface{}) bool {
	switch t := other.(type) {
	case jsJFlo:
		return f == t.f
	case float64:
		return f == t
	case jsChar:
		return f == float64(t.code)
	}
	return false
}

// jvmArith is one binary arithmetic operator of the Java subset. A double on
// either side makes the whole operation floating point and its result a double;
// two integral operands keep the 32 bit wrap the compiler emitted before (the
// int-width work is a separate concern, so this reproduces it exactly).
func (rt *jsrt) jvmArith(op byte, l, r interface{}) interface{} {
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

// jvmMinMax picks one operand, but the DOUBLE overload of Math.max/min is
// selected as soon as one side is a double, so the answer is a double then
// (Math.max(1.5, 2) is 2.0).
func (rt *jsrt) jvmMinMax(l, r interface{}, takeL bool) interface{} {
	v := r
	if takeL {
		v = l
	}
	if jvmIsFlo(l) || jvmIsFlo(r) {
		if f, ok := v.(jsJFlo); ok {
			return f
		}
		return jsJFlo{f: rt.toNumber(v), sty: jvmStyleOf(l, r)}
	}
	return v
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
		return rt.jvmMinMax(l, r, rt.toNumber(l) > rt.toNumber(r))
	}))
	o.set("min", jsHostFunc("min", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		l, r := argAt(args, 0), argAt(args, 1)
		return rt.jvmMinMax(l, r, rt.toNumber(l) < rt.toNumber(r))
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
		return rt.jvmArith(op, argAt(args, 1), argAt(args, 2))
	})
	// strictEq's two jsJFlo arms, which is what a program can otherwise only
	// reach through ===: 1.0 === 1 holds, and a box against a SIZED INTEGER is
	// false because jvmNumEq has no jsGInt case.
	b["floEq"] = jsHostFunc("floEq", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		l, r := argAt(args, 0), argAt(args, 1)
		if f, ok := l.(jsJFlo); ok {
			return jvmNumEq(f.f, r)
		}
		if f, ok := r.(jsJFlo); ok {
			return jvmNumEq(f.f, l)
		}
		return rt.strictEq(l, r)
	})
	// Math.max / Math.min / Math.abs of jvmMathObject, whose double overload is
	// selected as soon as one side is a double (Math.max(1.5, 2) is 2.0).
	b["floMax"] = jsHostFunc("floMax", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		l, r := argAt(args, 0), argAt(args, 1)
		return rt.jvmMinMax(l, r, rt.toNumber(l) > rt.toNumber(r))
	})
	b["floMin"] = jsHostFunc("floMin", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		l, r := argAt(args, 0), argAt(args, 1)
		return rt.jvmMinMax(l, r, rt.toNumber(l) < rt.toNumber(r))
	})
	b["floAbs"] = jsHostFunc("floAbs", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		v := argAt(args, 0)
		if f, ok := v.(jsJFlo); ok {
			return jsJFlo{f: math.Abs(f.f), sty: f.sty}
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
