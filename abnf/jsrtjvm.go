package abnf

// Floating point for the STATICALLY TYPED languages' compiler grammars
// (languages/java-to-llvm-ir.abnf, kotlin-to-llvm-ir.abnf, go-to-llvm-ir.abnf,
// csharp-to-llvm-ir.abnf).
//
// The handle runtime models a number as one JS double, which is enough for a
// dynamically typed language but not for a statically typed one: `1.0 / 3.0` and
// `1 / 3` are the SAME two operands there, and Java answers 0.3333333333333333 for
// the first and 0 for the second.
//
// jsJFlo mirrors what the interpreter grammar does with its {__flo} box (and what
// jsPhpFlo does for PHP, and jsFlo for Ruby): a floating-point value is BOXED, so
// the runtime can still tell a double from an int. The invariant is
//
//	plain number  ==  an integral type (int / long / short / byte)
//	jsJFlo        ==  a double / float
//
// and every operator that meets a jsJFlo evaluates in floating point and answers a
// jsJFlo (JLS 5.6.2, binary numeric promotion).
//
// Nothing here runs unless a program calls one of the js_jf* externs, which only
// the statically typed compiler grammars emit. The exception is the jsJFlo box
// itself, which rt.truthy / rt.toNumber / rt.toString / rt.toGoNatural / rt.typeOf /
// rt.strictEq know about in jsrt.go.

import (
	"fmt"
	"math"
	"os"
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
//	floJavaF the JAVA `float`    - java's print style at a 32 BIT WIDTH
//	floGoF   the GO `float32`    - go's print style at a 32 BIT WIDTH
//	floCSF   the C# `float`      - C#'s print style at a 32 BIT WIDTH
//
// THE STYLE BYTE CARRIES THE WIDTH. A Java float is a binary32, so `1.0f/3.0f` is
// 0.33333334 where the double is 0.3333333333333333, and every arithmetic result
// has to be ROUNDED to 24 significant bits. The value itself still travels as a
// float64 (every float is exactly representable as a double), so the only thing
// that has to go ON the box is the width - and the box already has a byte for a
// per-language tag, which languages/lib/runtime.c's tag 14 cell stores as an
// opaque long and hands back through jf_style_of.
//
// THE WIDTH IS GENERAL AND THE PRINT STYLE IS NOT: a Go float32 prints with Go's
// rules and a Java float with Java's, so each 32-bit language has its own byte;
// but they round, promote and compare identically, so everything about the WIDTH
// is asked through jvmIs32 rather than an equality against one constant.
//
// WHAT A MIXED 32/64 PAIR PROMOTES TO is per-language: Java widens (JLS 5.6.2:
// `1.0f + 1.0` is a double), and so does C# (ECMA-334 12.4.7.3). Go has no
// implicit conversion between float32 and float64 - a mixed pair is a compile
// error, so the only pair that reaches the runtime is a float32 against an
// UNTYPED CONSTANT, which Go converts TO float32. jvmWidens32 says which.
//
// THE FLOOR CANNOT RENDER A 32-BIT BOX: runtime.c's jf_text has arms for styles
// 0, 1 and 2 only. languages/lib/java-rt.metajs, go-rt.metajs and csharp-rt.metajs
// therefore intercept every rendering and every arithmetic result of a 32-bit box
// before the floor sees it (jvFloText / jvFloFix, goFloText / js_gf32fix,
// csFloText / csFloFix), and llvm.Run's side of that contract is jvmFloText /
// jvmArith here. The floor itself cannot do it: runtime.c is compiled by our own
// c-to-llvm-ir.abnf, which computes every float at double precision and so cannot
// spell `(float)x`.
const (
	floJava  = 0
	floGo    = 1
	floCS    = 2
	floJavaF = 3
	floGoF   = 4
	floCSF   = 5
)

// jvmIs32 is the WIDTH question, asked of a style byte: does a box carrying this
// style hold a binary32? Every width-sensitive site in this file goes through it,
// so a new 32-bit language adds one arm here and nowhere else.
func jvmIs32(sty uint8) bool { return sty == floJavaF || sty == floGoF || sty == floCSF }

// jvmWidens32 is "this language promotes a mixed 32/64 pair to the WIDER type",
// which is JLS 5.6.2 and holds for Java and Kotlin, and is ECMA-334 12.4.7.3
// word for word for C#. It is FALSE for Go, whose spec forbids the mixed pair
// outright: the only pair that reaches the runtime is a float32 against an
// untyped constant, and Go converts the constant to float32, so the 32-bit side
// wins. Swift has Go's rule if it ever gets a box here.
func jvmWidens32(sty uint8) bool { return sty == floJavaF || sty == floCSF }

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
	case floGoF:
		return goFloStr32(v.f)
	case floCSF:
		return csFloStr32(v.f)
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
// Checked against java 24.0.2.
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
		// closest two digit decimal to the actual value. That only differs
		// from padding among the SUBNORMALS: Double.MIN_VALUE renders
		// "4.9E-324" where appending ".0" to the shortest gives "5.0E-324".
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
// The jsGInt arm matters: `long x = 0; double y = 0.0; x == y` is TRUE - JLS
// 15.21.1 and ECMA-334 12.12.9 promote the integral operand to double first. The
// floor's twin is rtjStrictEq in lib/runtime.metajs; giEq (jsrtint.go) reads
// the pair the same way.
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
// two integral operands keep the 32 bit wrap.
func (rt *jsrt) jvmArith(op byte, l, r interface{}) interface{} {
	a, b := rt.toNumber(l), rt.toNumber(r)
	sty := jvmArithStyle(l, r)
	// BOTH OPERANDS CONVERT FIRST, and rounding only the result is not the same
	// answer: `float f = 0.1f; f + 16777217` is 1.6777216E7 in java, because the
	// int becomes a float (JLS 5.6.2 binary numeric promotion) BEFORE the add.
	// Rounding once at the end gives 1.6777218E7.
	if jvmIs32(sty) {
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
		if jvmIs32(sty) {
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
	sty := jvmArithStyle(l, r)
	return jsJFlo{f: jvmFround(rt.toNumber(l)), sty: sty},
		jsJFlo{f: jvmFround(rt.toNumber(r)), sty: sty}
}

// jvmFloPromotes is "these two operands promote to FLOAT" - one is a Java float
// box, the other is numeric, and neither is a double. Both operands are numeric
// by construction in a program javac accepts, but `==` also reaches this with a
// reference on one side, and that pair must keep the identity comparison.
func jvmFloPromotes(l, r interface{}) bool {
	if !jvmIsFlo(l) && !jvmIsFlo(r) {
		return false
	}
	if !jvmIs32(jvmArithStyle(l, r)) {
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

// jvmArithStyle is the promotion rule read for the WIDTH as well as the style.
// For a WIDENING language (java, kotlin) it is JLS 5.6.2: `float op float` is a
// float and `float op double` is a DOUBLE - the wider type wins, which
// jvmStyleOf's left-operand rule gets backwards for `1.0f + 1.0`. For a
// NON-widening one (go) the 32-bit side wins instead, because the only mixed pair
// Go accepts is a float32 against an untyped constant and Go converts the
// constant - see jvmWidens32. A pairing with no 32-bit operand at all is
// jvmStyleOf's answer.
func jvmArithStyle(l, r interface{}) uint8 {
	lf, lok := l.(jsJFlo)
	rf, rok := r.(jsJFlo)
	if lok && rok {
		if jvmIs32(lf.sty) && jvmWidens32(lf.sty) {
			return rf.sty // float+float -> float, float+double -> double
		}
		if jvmIs32(rf.sty) && !jvmWidens32(rf.sty) {
			return rf.sty // a float64-styled constant + float32 -> float32
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
// overloads:
//
//	Math.min(-0.0, 0.0) is -0.0 and Math.max(-0.0, 0.0) is 0.0 - "if one
//	argument is positive zero and the other negative zero, the result of min
//	is negative zero" (and of max, positive zero). -0.0 < 0.0 is FALSE, so a
//	bare comparison would take the right operand.
//
//	Math.min(NaN, 1.0) and Math.max(NaN, 1.0) are both NaN - "if either value
//	is NaN, then the result is NaN". Every comparison against NaN is false.
//
// Everything else is the ordinary comparison, tie going to the right operand.
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
		if jvmIs32(sty) {
			x = jvmFround(x)
		}
		return jsJFlo{f: x, sty: sty}
	}
	return v
}

// js_jflo32 is the float-width extern: `(float) e`, a `1.0f` literal and a
// `float x = e` declaration all lower to it, and its layer-2 twin is js_jflo32 in
// languages/lib/java-rt.metajs. The emitter wraps its operand in js_jvnum first,
// exactly as it does for js_jflo, so a Java `long` (a tag-13 cell marked
// UNSIGNED) reads signed.
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
		// THE JAVA LAUNCHER'S UNCAUGHT-EXCEPTION LINE, called from the catch
		// clause of the one js_try the emitter wraps the entry call in
		// (buildMain in languages/java-to-llvm-ir.abnf). Without it
		// runJSModule's handler would render the value with excText, which
		// knows neither the {name, message} nor the {__class, args} shape.
		//
		// java 24.0.2, on stderr with exit status 1:
		//     Exception in thread "main" java.lang.RuntimeException: abort probe
		//     Exception in thread "main" P2$NoMsg      <- getMessage() == null
		// The "\tat Main.main(Main.java:4)" frames are deliberately not
		// reproduced; no engine here has a call-site line table.
		//
		// The twins are jExcTopLine in languages/java-interpreter.abnf and
		// jvExcTopLine in languages/lib/java-rt.metajs - same two message slots
		// (`message` from the interpreter's jExcCtor, `__msg` from the emitted
		// builtin constructor), same binary class name, same stream, same code.
		m["js_jvtop"] = func(a []uint64) uint64 { return jvmUncaught(rt.jvmExcTopLine(rt.unwrap(a[0]))) }
	})
}

// jvmMainHead is the JVM launcher's prefix for an exception that escapes main,
// java's and kotlin's alike (kotlin's throwables are java.lang's).
const jvmMainHead = "Exception in thread \"main\" "

// jvmUncaught writes an uncaught-exception line to stderr and exits 1, as the
// java launcher and the .NET host both do. The js_jvtop / js_kttop / js_cstop
// closures end the program through it.
func jvmUncaught(line string) uint64 {
	fmt.Fprint(warnWriter, wtf8Clean(line+"\n"))
	os.Exit(1)
	return 0
}

// jvmExcTopLine is the text of js_jvtop above. Kept out of the closure so that
// the three engines' bodies read alike side by side.
func (rt *jsrt) jvmExcTopLine(v interface{}) string {
	o, ok := v.(*jsObject)
	if !ok {
		return jvmMainHead + rt.jvpStr(v)
	}
	return jvmMainHead + rt.jvmThrowableStr(o)
}

// jvmThrowableStr is java.lang.Throwable#toString - the whole of the abort line
// after the fixed prefix, and also what println(e) prints (the Throwable arm of
// jvpObj in abnf/jsrtjava.go). The twins are jThrowableStr in
// languages/java-interpreter.abnf and jvThrowableStr in
// languages/lib/java-rt.metajs. The message slot differs between the halves - the
// interpreter's jExcCtor writes `message`, the emitted builtin constructor writes
// `__msg` (emitBuiltinExc) - so both are read, in that order, in all three.
func (rt *jsrt) jvmThrowableStr(o *jsObject) string {
	name := "java.lang.Throwable"
	if cls, isObj := o.props["__class"].(*jsObject); isObj {
		name = jvpClassName(cls)
	}
	msg := o.props["message"]
	if jvmNullish(msg) {
		msg = o.props["__msg"]
	}
	if jvmNullish(msg) {
		return name
	}
	return name + ": " + rt.jvpStr(msg)
}

// jvmNullish is "absent, undefined or null" - the three ways a message slot can
// say that getMessage() would answer null. An EMPTY message is not one of them:
// real java prints the colon and the space for it.
func jvmNullish(v interface{}) bool {
	if v == nil {
		return true
	}
	switch v.(type) {
	case jsUndefT, jsNullT:
		return true
	}
	return false
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
		// A long is a 64-bit jsGInt box, and toInt32 would truncate it, so
		// negate in the box. Math.abs(Long.MIN_VALUE) is Long.MIN_VALUE
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
// program's global set. It is the jsJFlo twin of jsrtint.go's giBindings: without
// it a layer-2 file written against the C floor could be linked natively but not
// run by llvm.Run. programJSBindings (jsrtint.go) calls both.
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
	// here - unlike jvmNumEq, because that arm belongs to the LANGUAGE's `==`
	// (JLS 15.21.1 / ECMA-334 12.12.9) and not to the primitive. The promotion
	// is the CALLER's: lib/runtime.metajs's rtjStrictEq does it first.
	// tests/metajs-test-full.js's flo38 pins this contract.
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
		// The same 64-bit arm jvmMathObject's own abs carries: toInt32 would
		// TRUNCATE a long, so negate in the box; two's-complement negation maps
		// Long.MIN_VALUE to itself (JLS 15.15.4). Everything narrower than a
		// long widens to int before the call and keeps the 32-bit wrap.
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

// goFloStr32 is Go's fmt %v for a float32: the SAME 'g' rule read at 24
// significant bits instead of 53, which is all bitSize does - the shortest run of
// digits that reads back as the same binary32. float32(1)/float32(3) prints
// "0.33333334" where the float64 prints "0.3333333333333333", and the
// scientific-notation window (a decimal exponent below -4 or at least 6) is the
// same one, because for the shortest form Go fixes that threshold at 6 whatever
// the digit count. Checked against `go run`.
func goFloStr32(f float64) string {
	if math.IsNaN(f) {
		return "NaN"
	}
	if math.IsInf(f, 1) {
		return "+Inf"
	}
	if math.IsInf(f, -1) {
		return "-Inf"
	}
	return strconv.FormatFloat(f, 'g', -1, 32)
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

// csFloStr32 is System.Single.ToString() - the C# `float`, ECMA-334 8.3.7's
// binary32. ECMA-334 does not specify ToString()'s TEXT (it refers the string
// conversion to the BCL), and no C# toolchain is available here, so the DIGIT
// PLACEMENT is derived rather than measured; the arithmetic is the same binary32
// operation as Java's and is settled.
//
//   - The digits are the SHORTEST round-tripping decimal read at 24 significant
//     bits, the .NET Core 3.0+ default for Single.ToString(), so 1.0f/3.0f is
//     "0.33333334".
//   - NO forced ".0" and no two-significant-digit minimum (those are Java's).
//   - The plain/scientific window: .NET's general format goes scientific when the
//     decimal point sits past the type's G-precision, 7 for Single and 15 for
//     Double - hence 1e7f prints "1E+07" where the double 1e7 prints "10000000".
//     Only the UPPER bound differs from csFloStr; the lower bound is csFloStr's,
//     itself unverified (1e-5 renders "0.00001"; .NET is reported to answer
//     "1E-05"), and nothing in tests/csharp-test-full.cs asserts that boundary.
//   - "NaN" / "Infinity" / "-Infinity" are NumberFormatInfo.InvariantInfo's
//     symbols, and "-0" for a negative zero, both exactly as csFloStr has them.
func csFloStr32(f float64) string {
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
	if a >= 1e-5 && a < 1e7 {
		return strconv.FormatFloat(f, 'f', -1, 32)
	}
	return strconv.FormatFloat(f, 'E', -1, 32)
}
