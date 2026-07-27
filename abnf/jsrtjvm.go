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
