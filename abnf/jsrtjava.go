package abnf

// jsrtjava.go -- Java's VALUE RENDERING for the compiler grammar
// (languages/java-to-llvm-ir.abnf).
//
// The shared `println` host function hands the raw runtime value to Go's `%v`,
// which is wrong for every Java reference: real java 24 prints
//
//	null / [I@1dbd16a6 / Main$Box@7344699f / RED / P[x=1, y=2]
//
// (and Go's `%v` of an enum constant would recurse forever, because the
// constant's `__class` points at a descriptor that points back at it). This file
// is the Go twin of `jstr` / `jIdHash` / `jArrayTag` in java-interpreter.abnf and
// the two MUST stay in step, because ./test.sh --cross diffs them.
//
// Registration is additive: an init() appends a registrar to rxExtraExterns, so
// nothing here runs unless a program calls one of the js_jv* externs, and only the
// Java compiler grammar emits them.
//
// Known simplifications, all DELIBERATE and all shared by the two halves:
//
//   - The identity HASH after the `@` is a per-run counter, not an address: the
//     k-th distinct object rendered in a run gets 0x1a2b3c00+k, printed with %x.
//     Real java prints System.identityHashCode, which differs between two runs of
//     the SAME program, so no assertion can ever depend on the digits; what the two
//     halves DO have to agree on is that the same object renders the same way twice
//     and that a program's output is reproducible, which a counter gives.
//   - An array created with a declared type carries that type (js_jvarrtag).
//     Any other array's ELEMENT TYPE is inferred from the elements: plain number
//     -> [I, boxed double -> [D, String -> [Ljava.lang.String;, boolean -> [Z,
//     char -> [C, and empty or mixed -> [Ljava.lang.Object;.
//   - `Object#toString` IS a callable member: `b.toString()` on a class that
//     declares none routes to js_jvstr, the same rendering println / `+` /
//     String.valueOf use, and `b.equals(o)` routes to js_jvaleq. Both dispatch to a
//     declared or inherited method first, so an override is unaffected.

import (
	"fmt"
	"math"
	"math/bits"
	"math/rand/v2"
	"strconv"
	"strings"
	"sync"
	"unicode/utf16"
)

// ----------------------------------------------------------------------------
// SIZED INTEGERS
//
// Java's integral types have fixed widths and wrap silently, and the width has
// to survive into the next operation - `byte b = 127; b++` is -128, and nothing
// about the value 127 says so. The box that carries it is jsGInt from
// jsrtint.go, the same one Go and Swift use; only the RULES are Java's, and they
// live here so no shared file has to learn a second dialect.
//
// The invariant is Java's and it is deliberately NOT Go's. There a plain number
// is a 64 bit int; here
//
//	a plain number  ==  an `int`, i.e. a SIGNED 32 BIT value
//	a jsGInt        ==  a long (w 64), a short (w 16) or a byte (w 8)
//	a jsJFlo        ==  a double / float                        (jsrtjvm.go)
//	a jsChar        ==  a char
//
// int is the type of every unsuffixed integer literal and of every arithmetic
// result with no long operand, so it stays unboxed. The cost is that a 64 bit
// value stays boxed even when it would fit in a double, because a plain number
// means `int` here - giNorm unboxes such a result, so jvBox boxes it straight back.
//
// languages/java-interpreter.abnf implements exactly these rules on top of the
// sz* layer in lib/interp-core.js, and ./test.sh --cross diffs the two halves,
// so the two MUST stay in step.

// jvBox is "read this as a long": always a box, never a plain number.
func jvBox(v int64) jsGInt { return jsGInt{v: v, w: 64, u: false} }

func jvIsLong(v interface{}) bool {
	b, ok := v.(jsGInt)
	return ok && b.w == 64
}

// jvIsIntegral says whether a value is an operand of INTEGER arithmetic - which
// a char is, Java's char being an integral type, and a double is not.
func jvIsIntegral(v interface{}) bool {
	switch v.(type) {
	case jsGInt, float64, jsChar:
		return true
	}
	return false
}

// jvW is JLS 5.6.2 binary numeric promotion for two integral operands: long when
// either side is long, int otherwise - so `byte + byte` is an int and
// `short * short` is an int, which is the rule people are surprised by.
func jvW(l, r interface{}) uint8 {
	if jvIsLong(l) || jvIsLong(r) {
		return 64
	}
	return 32
}

// jvI is the int reading of an integral operand: the low 32 bits of its value.
func jvI(rt *jsrt, v interface{}) int32 { return int32(giVal(rt, v)) }

// The six runtime-raised exception class descriptors, each registered by NAME
// by js_jvexc at module init, because the externs that raise them have no access
// to the program's scope; without one the raise aborts the run instead of
// throwing. One compile runs one program, so a package-level holder is
// per-program and stays out of the shared jsrt struct.
//
// ArithmeticException is "/ by zero" (JLS 15.17.2). The other five: an
// out-of-range array access, an out-of-range string index or range, a negative
// array length, a null array reference and a malformed parseInt / parseLong
// string. The layer-2 twin is the block of jv*Cls slots in
// languages/lib/java-rt.metajs.
var jvArithExcCls *jsObject
var jvAioobeCls *jsObject
var jvSioobeCls *jsObject
var jvNegArrCls *jsObject
var jvNpeCls *jsObject
var jvNfeCls *jsObject
var jvCceCls *jsObject

// Every runtime-raised class, by name (jvThrowNamed).
var jvExcNamed = map[string]*jsObject{}

func (rt *jsrt) jvThrowNamed(cls, msg string) { rt.jvThrowCls(jvExcNamed[cls], msg) }

func (rt *jsrt) jvThrowCls(cls *jsObject, msg interface{}) {
	if cls == nil {
		rt.fail("java: %v", msg)
		return
	}
	o := newJSObject()
	o.set("__class", cls)
	// __msg is the field the emitted builtin exception constructor writes and
	// getMessage() reads (see emitBuiltinExc in java-to-llvm-ir.abnf).
	o.set("__msg", msg)
	// jsThrown carries the VALUE, not its handle - js_throw stores u(a[0]).
	panic(&jsThrown{value: o})
}

func (rt *jsrt) jvThrowArith(msg string) { rt.jvThrowCls(jvArithExcCls, msg) }

// jvParseNull is Integer.parseInt(null) and Long.parseLong(null). Twin of
// jParseInt in java-interpreter.abnf and jvParseInt in
// languages/lib/java-rt.metajs.
func (rt *jsrt) jvParseNull(v interface{}) {
	if _, isNull := v.(jsNullT); isNull {
		rt.jvThrowCls(jvNfeCls, "Cannot parse null string")
	}
}

// jvOutText is a String as System.out writes it: UTF-8 cannot carry a lone
// surrogate, which the encoder replaces with '?' (wtf8Clean writes U+FFFD).
// Twin of jOutText in java-interpreter.abnf.
func jvOutText(s string) string {
	i := strings.IndexByte(s, 0xED)
	if i < 0 {
		return s
	}
	b := make([]byte, 0, len(s))
	b = append(b, s[:i]...)
	for i < len(s) {
		if wtf8Surrogate(s[i:]) != 0 {
			b = append(b, '?')
			i += 3
			continue
		}
		b = append(b, s[i])
		i++
	}
	return string(b)
}

// jvNpeIfNull throws the NullPointerException a null raises where Java unboxes
// it (JLS 5.1.8) or a library method dereferences it. Twin of jvNpeIfNull in
// languages/lib/java-rt.metajs.
func (rt *jsrt) jvNpeIfNull(v interface{}) {
	if _, isNull := v.(jsNullT); isNull {
		rt.jvThrowCls(jvNpeCls, jsNull)
	}
}

// Is the value part of the java.lang Throwable hierarchy at all? The 64-deep cap
// is the one every __class/__super walk here carries (manual chapter 7.9).
func jvIsThrowable(v interface{}) bool {
	o, ok := v.(*jsObject)
	if !ok {
		return false
	}
	cls := o.props["__class"]
	for guard := 0; guard < 64; guard++ {
		co, isObj := cls.(*jsObject)
		if !isObj {
			return false
		}
		if nm, _ := co.props["__name"].(string); nm == "Throwable" {
			return true
		}
		cls = co.props["__super"]
	}
	return false
}

// The three bounds messages are java 24.0.2's, verified against the real
// toolchain. jArrIdx/jStrIdx/jSubstr in languages/java-interpreter.abnf and
// jvThrowAioobe/jvThrowSioobe/jvThrowRange in languages/lib/java-rt.metajs spell
// the same three, and tests/java-test-full.java SECTION 34 pins them.
func (rt *jsrt) jvThrowAioobe(i, n int) {
	rt.jvThrowCls(jvAioobeCls, fmt.Sprintf("Index %d out of bounds for length %d", i, n))
}
func (rt *jsrt) jvThrowSioobe(i, n int) {
	rt.jvThrowCls(jvSioobeCls, fmt.Sprintf("Index %d out of bounds for length %d", i, n))
}
func (rt *jsrt) jvThrowRange(b, e, n int) {
	rt.jvThrowCls(jvSioobeCls, fmt.Sprintf("Range [%d, %d) out of bounds for length %d", b, e, n))
}

// jvArith is one binary arithmetic or bitwise operator on two INTEGRAL operands.
// The int case is native int32 arithmetic; only a long operand pays for the
// 64 bit box.
func (rt *jsrt) jvArith(op string, l, r interface{}) interface{} {
	if jvW(l, r) == 64 {
		a, b := jvBox(giVal(rt, l)), jvBox(giVal(rt, r))
		if (op == "/" || op == "%") && b.v == 0 {
			rt.jvThrowArith("/ by zero")
		}
		return jvBox(giVal(rt, rt.giArith(op, a, b)))
	}
	a, b := jvI(rt, l), jvI(rt, r)
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
			rt.jvThrowArith("/ by zero")
		}
		// Integer.MIN_VALUE / -1 overflows back to itself (JLS 15.17.2); Go's own
		// int32 division panics on exactly that pair, so it is special-cased.
		if a == math.MinInt32 && b == -1 {
			x = a
		} else {
			x = a / b
		}
	case "%":
		if b == 0 {
			rt.jvThrowArith("/ by zero")
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
		rt.fail("js_jvarith: unknown operator %q", op)
	}
	return float64(x)
}

// jvShift is << >> >>>. The result type is the LEFT operand's alone (JLS 15.19 -
// each operand is promoted separately), and the COUNT is MASKED: & 31 for an int,
// & 63 for a long. That is why `1 << 32` is 1 rather than 0, and it is the one
// place Java and Go disagree outright, Go shifting everything out instead.
func (rt *jsrt) jvShift(op string, l, r interface{}) interface{} {
	n := giVal(rt, r)
	if jvIsLong(l) {
		s := n & 63
		a := giVal(rt, l)
		switch op {
		case "<<":
			return jvBox(a << uint(s))
		case ">>":
			return jvBox(a >> uint(s))
		}
		// >>> is the LOGICAL shift: read the operand as unsigned, shift, read the
		// result back as signed. C# spells this ">>" on an unsigned type.
		return jvBox(int64(uint64(a) >> uint(s)))
	}
	a := jvI(rt, l)
	s := uint(n & 31)
	switch op {
	case "<<":
		return float64(a << s)
	case ">>":
		return float64(a >> s)
	}
	return float64(int32(uint32(a) >> s))
}

// jvCmp is the ordered comparison at the PROMOTED width. giCmp on its own is not
// usable: it compares at the widest operand's width, so `byte b = 5; b < 1000`
// would truncate 1000 to a byte and answer false where Java promotes both to int.
func (rt *jsrt) jvCmp(op string, l, r interface{}) bool {
	var c int
	if jvW(l, r) == 64 {
		a, b := giVal(rt, l), giVal(rt, r)
		if a < b {
			c = -1
		} else if a > b {
			c = 1
		}
	} else {
		a, b := jvI(rt, l), jvI(rt, r)
		if a < b {
			c = -1
		} else if a > b {
			c = 1
		}
	}
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

// jvToLong is the exact 64 bit reading of an INTEGRAL value.
func jvToLong(rt *jsrt, v interface{}) int64 {
	switch t := v.(type) {
	case jsGInt:
		return t.v
	case jsChar:
		return int64(t.code)
	case float64:
		return int64(t) // an int is exact
	}
	rt.jvNpeIfNull(v)
	return jvNarrowFloat(rt.toNumber(v), 64)
}

// jvNarrowFloat is the JLS 5.1.3 narrowing of a FLOATING POINT value. It
// SATURATES at the target's range rather than wrapping the way Go does - and for
// a target narrower than int it saturates at the INT range first and only then
// truncates the bits, so (int) 1e20 is Integer.MAX_VALUE and (byte) 1e20 is
// (byte) Integer.MAX_VALUE == -1.
func jvNarrowFloat(f float64, w uint8) int64 {
	if math.IsNaN(f) {
		return 0
	}
	if w == 64 {
		if f >= 9223372036854775808 {
			return math.MaxInt64
		}
		if f < -9223372036854775808 {
			return math.MinInt64
		}
		return int64(math.Trunc(f))
	}
	if f >= 2147483648 {
		return math.MaxInt32
	}
	if f < -2147483648 {
		return math.MinInt32
	}
	return int64(math.Trunc(f))
}

// jvNarrow is (byte) / (short) / (int) / (long) / a declared integral type. An
// integral source is a pure bit truncation; a floating point one saturates first
// (see jvNarrowFloat).
func jvNarrow(rt *jsrt, v interface{}, w uint8) interface{} {
	var p int64
	if f, isFlo := v.(jsJFlo); isFlo {
		p = jvNarrowFloat(f.f, w)
	} else {
		p = jvToLong(rt, v)
	}
	switch w {
	case 64:
		return jvBox(p)
	case 32:
		return float64(int32(p))
	case 16:
		return jsGInt{v: int64(int16(p)), w: 16, u: false}
	}
	return jsGInt{v: int64(int8(p)), w: 8, u: false}
}

// jvParseInt is Integer.parseInt / Long.parseLong and their radix overloads: an
// optional sign and a digit run in the radix; anything else - whitespace and a
// magnitude past the width's range included - throws NumberFormatException.
// Twin of jParseInt in languages/java-interpreter.abnf and jvParseInt in
// languages/lib/java-rt.metajs.
func (rt *jsrt) jvParseInt(s string, width uint8, radix int) interface{} {
	bad := "For input string: \"" + s + "\""
	if radix != 10 {
		bad += " under radix " + strconv.Itoa(radix)
	}
	if radix < 2 {
		rt.jvThrowCls(jvNfeCls, "radix "+strconv.Itoa(radix)+" less than Character.MIN_RADIX")
	}
	if radix > 36 {
		rt.jvThrowCls(jvNfeCls, "radix "+strconv.Itoa(radix)+" greater than Character.MAX_RADIX")
	}
	t := s
	neg := false
	if strings.HasPrefix(t, "+") || strings.HasPrefix(t, "-") {
		neg = t[0] == '-'
		t = t[1:]
	}
	if t == "" {
		rt.jvThrowCls(jvNfeCls, bad)
	}
	limit := uint64(1)<<(width-1) - 1
	if neg {
		limit++
	}
	var acc uint64
	for i := 0; i < len(t); i++ {
		d := jvDigitOf(t[i])
		if d >= radix || acc > (limit-uint64(d))/uint64(radix) {
			rt.jvThrowCls(jvNfeCls, bad)
		}
		acc = acc*uint64(radix) + uint64(d)
	}
	v := int64(acc)
	if neg {
		v = -v
	}
	return jvNarrow(rt, jvBox(v), width)
}

// jvDigitOf is Character.digit's ASCII reading: 0-9, then a-z / A-Z as 10-35,
// and 99 - which no radix accepts - for anything else.
func jvDigitOf(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'z':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'Z':
		return int(c-'A') + 10
	}
	return 99
}

// jvRadixArg is the optional radix argument of parseInt / valueOf: 10 if absent.
func jvRadixArg(rt *jsrt, args []interface{}) int {
	if len(args) < 2 {
		return 10
	}
	return int(giVal(rt, args[1]))
}

// jvMathMembers adds java.lang.Math beyond abs / max / min to jvmMathObject.
// The double methods widen the argument (JLS 5.3) and answer a double whatever
// it was, so Math.sqrt(4) is 2.0. Twin of hostGlobals["Math"] in
// java-interpreter.abnf and of jvBoxType("Math") in languages/lib/java-rt.metajs.
func jvMathMembers(o *jsObject) {
	dbl := func(f float64) interface{} { return jsJFlo{f: f, sty: floJava} }
	o.set("PI", dbl(math.Pi))
	o.set("E", dbl(math.E))
	// The transcendentals are StrictMath's, fdlibm (jsrtfdlibm.go), which
	// java.lang.Math answers too - but for sin and cos on a JIT intrinsic.
	one := map[string]func(float64) float64{
		"sqrt": math.Sqrt, "cbrt": fdCbrt, "exp": fdExp, "expm1": fdExpm1,
		"log": fdLog, "log10": fdLog10, "log1p": fdLog1p,
		"sin": fdSin, "cos": fdCos, "tan": fdTan,
		"asin": fdAsin, "acos": fdAcos, "atan": fdAtan,
		"sinh": fdSinh, "cosh": fdCosh, "tanh": fdTanh,
		"floor": math.Floor, "ceil": math.Ceil, "rint": math.RoundToEven,
		// One multiplication by java.lang.Math's own constant, which is why
		// toRadians(180) is exactly Math.PI.
		"toRadians": func(x float64) float64 { return x * 0.017453292519943295 },
		"toDegrees": func(x float64) float64 { return x * 57.29577951308232 },
	}
	for name, fn := range one {
		fn := fn
		o.set(name, jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return dbl(fn(rt.toNumber(argAt(args, 0))))
		}))
	}
	two := map[string]func(float64, float64) float64{
		"atan2": fdAtan2, "hypot": fdHypot, "pow": fdPow,
	}
	for name, fn := range two {
		fn := fn
		o.set(name, jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return dbl(fn(rt.toNumber(argAt(args, 0)), rt.toNumber(argAt(args, 1))))
		}))
	}
	o.set("random", jsHostFunc("random", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return dbl(rand.Float64())
	}))
	// round(double) is a long and round(float) an int; an integral argument
	// takes the float overload (int -> float is the more specific widening).
	// The answer is the closest integer with ties toward +Infinity, saturated
	// at the width's extremes (NaN -> 0).
	o.set("round", jsHostFunc("round", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		v := argAt(args, 0)
		f := rt.toNumber(v)
		if jvIsDouble(v) {
			return jvNarrow(rt, dbl(jvRoundHalfUp(f)), 64)
		}
		return jvNarrow(rt, dbl(jvRoundHalfUp(jvmFround(f))), 32)
	}))
	// signum keeps a double a double and a float a float; an integral argument
	// takes the float overload. NaN and both zeros answer themselves.
	o.set("signum", jsHostFunc("signum", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		v := argAt(args, 0)
		f := rt.toNumber(v)
		if f > 0 {
			f = 1
		} else if f < 0 {
			f = -1
		}
		if jvIsDouble(v) {
			return dbl(f)
		}
		return jsJFlo{f: f, sty: floJavaF}
	}))
	// ulp's float overload takes an integral argument too, as signum's; the
	// remainder is IEEE 754's, which math.Remainder is. Twin of jxUlp /
	// jxIeeeRem in languages/lib/java-strings.js.
	o.set("ulp", jsHostFunc("ulp", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		v := argAt(args, 0)
		f := math.Abs(rt.toNumber(v))
		minExp, mant := -126, 23
		if jvIsDouble(v) {
			minExp, mant = -1022, 52
		}
		u := f
		if !math.IsNaN(f) && !math.IsInf(f, 0) {
			if f < math.Ldexp(1, minExp) {
				u = math.Ldexp(1, minExp-mant)
			} else {
				_, e := math.Frexp(f)
				u = math.Ldexp(1, e-1-mant)
			}
		}
		if jvIsDouble(v) {
			return dbl(u)
		}
		return jsJFlo{f: u, sty: floJavaF}
	}))
	o.set("IEEEremainder", jsHostFunc("IEEEremainder", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return dbl(math.Remainder(rt.toNumber(argAt(args, 0)), rt.toNumber(argAt(args, 1))))
	}))
	// The high 64 bits of the signed 128-bit product: the unsigned one, less
	// the other operand for each negative one. Twin of jxMulHigh.
	o.set("multiplyHigh", jsHostFunc("multiplyHigh", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		a, b := jvToLong(rt, argAt(args, 0)), jvToLong(rt, argAt(args, 1))
		hi, _ := bits.Mul64(uint64(a), uint64(b))
		if a < 0 {
			hi -= uint64(b)
		}
		if b < 0 {
			hi -= uint64(a)
		}
		return jvBox(int64(hi))
	}))
	// floorDiv / floorMod round the quotient toward -Infinity, through the
	// language's own / and % so the width and the "/ by zero" throw follow.
	o.set("floorDiv", jsHostFunc("floorDiv", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		a, b := argAt(args, 0), argAt(args, 1)
		q := rt.jvArith("/", a, b)
		r := rt.jvArith("%", a, b)
		if rt.jvCmp("!=", r, float64(0)) && rt.jvCmp("<", r, float64(0)) != rt.jvCmp("<", b, float64(0)) {
			return rt.jvArith("-", q, float64(1))
		}
		return q
	}))
	o.set("floorMod", jsHostFunc("floorMod", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		a, b := argAt(args, 0), argAt(args, 1)
		r := rt.jvArith("%", a, b)
		if rt.jvCmp("!=", r, float64(0)) && rt.jvCmp("<", r, float64(0)) != rt.jvCmp("<", b, float64(0)) {
			return rt.jvArith("+", r, b)
		}
		return r
	}))
	for name, op := range map[string]string{"addExact": "+", "subtractExact": "-", "multiplyExact": "*"} {
		op := op
		o.set(name, jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return rt.jvExact(op, argAt(args, 0), argAt(args, 1))
		}))
	}
	unit := func(v interface{}, n int64) interface{} {
		if jvIsLong(v) {
			return jvBox(n)
		}
		return float64(n)
	}
	o.set("incrementExact", jsHostFunc("incrementExact", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return rt.jvExact("+", argAt(args, 0), unit(argAt(args, 0), 1))
	}))
	o.set("decrementExact", jsHostFunc("decrementExact", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return rt.jvExact("-", argAt(args, 0), unit(argAt(args, 0), 1))
	}))
	o.set("negateExact", jsHostFunc("negateExact", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return rt.jvExact("-", unit(argAt(args, 0), 0), argAt(args, 0))
	}))
	o.set("toIntExact", jsHostFunc("toIntExact", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		n := jvNarrow(rt, argAt(args, 0), 32)
		if rt.jvCmp("!=", n, argAt(args, 0)) {
			rt.jvThrowArith("integer overflow")
		}
		return n
	}))
}

// jvExact is Math.addExact / subtractExact / multiplyExact at the promoted
// width: the wrapped result, or ArithmeticException "integer overflow" / "long
// overflow" when it differs from the true one. Twin of jExact in
// java-interpreter.abnf, whose comment has the two tests.
func (rt *jsrt) jvExact(op string, a, b interface{}) interface{} {
	r := rt.jvArith(op, a, b)
	if !jvIsLong(r) {
		x, y := rt.toNumber(a), rt.toNumber(b)
		t := x * y
		switch op {
		case "+":
			t = x + y
		case "-":
			t = x - y
		}
		if t != rt.toNumber(r) {
			rt.jvThrowArith("integer overflow")
		}
		return r
	}
	zero := float64(0)
	na, nb, nr := rt.jvCmp("<", a, zero), rt.jvCmp("<", b, zero), rt.jvCmp("<", r, zero)
	bad := false
	switch op {
	case "+":
		bad = na == nb && nr != na
	case "-":
		bad = na != nb && nr != na
	default:
		bad = (rt.jvCmp("!=", b, zero) && rt.jvCmp("!=", rt.jvArith("/", r, b), a)) ||
			(rt.jvCmp("==", a, jvBox(math.MinInt64)) && rt.jvCmp("==", b, float64(-1)))
	}
	if bad {
		rt.jvThrowArith("long overflow")
	}
	return r
}

// jvCharMembers is java.lang.Character's classification and case mapping.
// Twin of jCharMembers in java-interpreter.abnf, whose comment has the rules,
// and of languages/lib/java-rt.metajs.
func jvCharMembers(o *jsObject) {
	code := func(rt *jsrt, v interface{}) int {
		if c, ok := v.(jsChar); ok {
			return int(c.code)
		}
		return int(rt.toNumber(v))
	}
	// Character.toChars: a code point's one or two chars. Twin of
	// jCharMembers in java-interpreter.abnf.
	o.set("toChars", jsHostFunc("toChars", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		cp := int(jvToLong(rt, argAt(args, 0)))
		if cp < 0 || cp > 0x10FFFF {
			rt.jvThrowCls(jvIaeCls, "Not a valid Unicode code point: 0x"+strings.ToUpper(strconv.FormatUint(uint64(uint32(int32(cp))), 16)))
		}
		var elems []interface{}
		for _, u := range utf16.Encode([]rune{rune(cp)}) {
			elems = append(elems, jsChar{code: int32(u)})
		}
		return &jsArray{elems: elems, kind: jvArrTagKind("[C")}
	}))
	like := func(v interface{}, c int) interface{} {
		if _, ok := v.(jsChar); ok {
			return jsChar{code: int32(c)}
		}
		return float64(c)
	}
	pred := func(name string, p func(c int) bool) {
		o.set(name, jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return p(code(rt, argAt(args, 0)))
		}))
	}
	pred("isDigit", jcIsDigit)
	pred("isLetter", jcIsLetter)
	pred("isAlphabetic", jcIsAlphabetic)
	pred("isLetterOrDigit", func(c int) bool { return jcIsLetter(c) || jcIsDigit(c) })
	pred("isUpperCase", jcIsUpper)
	pred("isLowerCase", jcIsLower)
	pred("isWhitespace", func(c int) bool {
		return (c >= 9 && c <= 13) || (c >= 28 && c <= 32) || c == 5760 || (c >= 8192 && c <= 8198) ||
			(c >= 8200 && c <= 8202) || c == 8232 || c == 8233 || c == 8287 || c == 12288
	})
	o.set("toUpperCase", jsHostFunc("toUpperCase", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return like(argAt(args, 0), jvChUp(code(rt, argAt(args, 0))))
	}))
	o.set("toLowerCase", jsHostFunc("toLowerCase", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return like(argAt(args, 0), jvChLo(code(rt, argAt(args, 0))))
	}))
	o.set("getNumericValue", jsHostFunc("getNumericValue", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return float64(jcNumericValue(code(rt, argAt(args, 0))))
	}))
	o.set("digit", jsHostFunc("digit", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		if d := jcDigitVal(code(rt, argAt(args, 0))); float64(d) < rt.toNumber(argAt(args, 1)) {
			return float64(d)
		}
		return float64(-1)
	}))
	o.set("compare", jsHostFunc("compare", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return float64(code(rt, argAt(args, 0)) - code(rt, argAt(args, 1)))
	}))
	o.set("valueOf", jsHostFunc("valueOf", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return argAt(args, 0)
	}))
}

// jvUnbox is Number's intValue() ... doubleValue(), Character's charValue() and
// Boolean's booleanValue() on a boxed primitive: the primitive conversion to the
// named type (JLS 5.1.2, 5.1.3). ok is false for any other receiver or name.
// Twin of jUnbox in java-interpreter.abnf and languages/lib/java-rt.metajs.
func (rt *jsrt) jvUnbox(v interface{}, name string) (r interface{}, ok bool) {
	if bn := jvBoxName(v); bn == "" || bn == "String" {
		return nil, false
	}
	switch name {
	case "intValue":
		return jvNarrow(rt, v, 32), true
	case "longValue":
		return jvNarrow(rt, v, 64), true
	case "doubleValue":
		return jvmMkFlo(rt.toNumber(v)), true
	case "floatValue":
		return jsJFlo{f: jvmFround(rt.toNumber(v)), sty: floJavaF}, true
	case "shortValue":
		return jvNarrow(rt, v, 16), true
	case "byteValue":
		return jvNarrow(rt, v, 8), true
	case "charValue", "booleanValue":
		return v, true
	}
	return nil, false
}

// Character.toUpperCase / toLowerCase: the simple mapping of one unit. Twin of
// jxChUp / jxChLo in languages/lib/java-strings.js.
func jvChUp(c int) int {
	if s := strUnits(caseMapMode(strFromUnits([]uint16{uint16(c)}), caseUpper)); len(s) == 1 {
		return int(s[0])
	}
	return c
}

func jvChLo(c int) int {
	if s := strUnits(caseMapMode(strFromUnits([]uint16{uint16(c)}), 0)); len(s) == 1 {
		return int(s[0])
	}
	return c
}

// jvMaxD / jvMinD are Math.max / Math.min on doubles: NaN wins, and -0.0 is
// below 0.0. Twin of jMaxD / jMinD in java-interpreter.abnf.
func jvMaxD(a, b float64) float64 {
	if a != a {
		return a
	}
	if a == 0 && b == 0 && math.Signbit(a) {
		return b
	}
	if a >= b {
		return a
	}
	return b
}

func jvMinD(a, b float64) float64 {
	if a != a {
		return a
	}
	if a == 0 && b == 0 && math.Signbit(b) {
		return b
	}
	if a <= b {
		return a
	}
	return b
}

// jvIsDouble: is v a Java double (and not a float or an integral value)?
func jvIsDouble(v interface{}) bool {
	f, ok := v.(jsJFlo)
	return ok && f.sty != floJavaF
}

// jvRoundHalfUp is the closest integer with ties toward +Infinity. x -
// floor(x) is exact, so 0.49999999999999994 rounds to 0.
func jvRoundHalfUp(x float64) float64 {
	if math.IsNaN(x) || math.Abs(x) >= 4503599627370496 {
		return x
	}
	f := math.Floor(x)
	if x-f >= 0.5 {
		return f + 1
	}
	return f
}

// jvDoubleCompare is Double.compare / Float.compare: the numeric order, except
// that -0.0 is below 0.0 and NaN is above everything and equal to itself.
func jvDoubleCompare(a, b float64) float64 {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	an, bn := math.IsNaN(a), math.IsNaN(b)
	if an || bn {
		if an && bn {
			return 0
		}
		if an {
			return 1
		}
		return -1
	}
	if a == 0 && math.Signbit(a) != math.Signbit(b) {
		if math.Signbit(a) {
			return -1
		}
		return 1
	}
	return 0
}

// jvWrapCompare is compareTo on a boxed primitive (java.lang.Comparable):
// Integer and Long answer the sign, Short, Byte and Character the difference,
// Double and Float Double.compare, Boolean false < true. ok is false for any
// other receiver. Twin of the compareTo arm of mcall in java-interpreter.abnf
// and jvWrapCompare in languages/lib/java-rt.metajs.
func (rt *jsrt) jvWrapCompare(x, y interface{}) (r float64, ok bool) {
	switch v := x.(type) {
	case jsJFlo:
		return jvDoubleCompare(v.f, rt.toNumber(y)), true
	case bool:
		if yb, _ := y.(bool); v == yb {
			return 0, true
		}
		if v {
			return 1, true
		}
		return -1, true
	case jsChar:
		return float64(int64(v.code) - giVal(rt, y)), true
	case jsGInt:
		if v.w < 32 {
			return float64(v.v - giVal(rt, y)), true
		}
	case float64:
	default:
		return 0, false
	}
	if rt.jvCmp("<", x, y) {
		return -1, true
	}
	if rt.jvCmp(">", x, y) {
		return 1, true
	}
	return 0, true
}

// jvParseDbl is Double.parseDouble's text: String.trim, an optional sign, then
// NaN, Infinity, or decimal digits with an optional point, exponent and
// f/F/d/D suffix. Anything else throws NumberFormatException with java
// 24.0.2's message: "empty String", "multiple points", or `For input string:
// "x"`. A hexadecimal significand is jvParseHex's, rounded once to the target
// width (f32 for Float). Twin of jParseDbl in
// java-interpreter.abnf and jvParseDbl in languages/lib/java-rt.metajs.
func (rt *jsrt) jvParseDbl(s string, f32 bool) float64 {
	t := strings.TrimFunc(s, func(r rune) bool { return r <= ' ' })
	if t == "" {
		rt.jvThrowCls(jvNfeCls, "empty String")
	}
	bad := "For input string: \"" + s + "\""
	i := 0
	neg := false
	if t[0] == '+' || t[0] == '-' {
		neg = t[0] == '-'
		i = 1
	}
	switch t[i:] {
	case "NaN":
		return math.NaN()
	case "Infinity":
		if neg {
			return math.Inf(-1)
		}
		return math.Inf(1)
	}
	if rest := t[i:]; len(rest) > 1 && rest[0] == '0' && (rest[1] == 'x' || rest[1] == 'X') {
		hx, ok := jvParseHex(rest[2:], f32)
		if !ok {
			rt.jvThrowCls(jvNfeCls, bad)
		}
		if neg {
			return -hx
		}
		return hx
	}
	isDigit := func(j int) bool { return j < len(t) && t[j] >= '0' && t[j] <= '9' }
	digits := 0
	for ; isDigit(i); i++ {
		digits++
	}
	if i < len(t) && t[i] == '.' {
		for i++; isDigit(i); i++ {
			digits++
		}
		if i < len(t) && t[i] == '.' {
			rt.jvThrowCls(jvNfeCls, "multiple points")
		}
	}
	if digits == 0 {
		rt.jvThrowCls(jvNfeCls, bad)
	}
	if i < len(t) && (t[i] == 'e' || t[i] == 'E') {
		i++
		if i < len(t) && (t[i] == '+' || t[i] == '-') {
			i++
		}
		ed := 0
		for ; isDigit(i); i++ {
			ed++
		}
		if ed == 0 {
			rt.jvThrowCls(jvNfeCls, bad)
		}
	}
	end := i
	if i < len(t) && strings.IndexByte("fFdD", t[i]) >= 0 {
		i++
	}
	if i != len(t) {
		rt.jvThrowCls(jvNfeCls, bad)
	}
	f, _ := strconv.ParseFloat(t[:end], 64)
	return f
}

// jvParseHex is a hexadecimal significand's magnitude, rounded once to a double
// or (f32) a float. Twin of jParseHex in java-interpreter.abnf, whose comment
// has the method.
func jvParseHex(t string, f32 bool) (float64, bool) {
	hexVal := func(c byte) int {
		switch {
		case c >= '0' && c <= '9':
			return int(c - '0')
		case c >= 'a' && c <= 'f':
			return int(c-'a') + 10
		case c >= 'A' && c <= 'F':
			return int(c-'A') + 10
		}
		return -1
	}
	var ds []byte
	intDigits := -1
	i := 0
	for ; i < len(t); i++ {
		if t[i] == '.' && intDigits < 0 {
			intDigits = len(ds)
		} else if hexVal(t[i]) >= 0 {
			ds = append(ds, t[i])
		} else {
			break
		}
	}
	if intDigits < 0 {
		intDigits = len(ds)
	}
	if len(ds) == 0 || i >= len(t) || (t[i] != 'p' && t[i] != 'P') {
		return 0, false
	}
	i++
	eneg := false
	if i < len(t) && (t[i] == '+' || t[i] == '-') {
		eneg = t[i] == '-'
		i++
	}
	pexp, ed := 0, 0
	for ; i < len(t) && t[i] >= '0' && t[i] <= '9'; i++ {
		if pexp < 100000000 {
			pexp = pexp*10 + int(t[i]-'0')
		}
		ed++
	}
	if i < len(t) && strings.IndexByte("fFdD", t[i]) >= 0 {
		i++
	}
	if ed == 0 || i != len(t) {
		return 0, false
	}
	if eneg {
		pexp = -pexp
	}
	j := 0
	for j < len(ds) && hexVal(ds[j]) == 0 {
		j++
	}
	if j == len(ds) {
		return 0, true
	}
	lz := bits.LeadingZeros8(uint8(hexVal(ds[j]))) - 4
	e := (intDigits-j)*4 - 1 - lz + pexp
	big, emin, emax := 53, -1022, 1023
	if f32 {
		big, emin, emax = 24, -126, 127
	}
	if e > emax {
		return math.Inf(1), true
	}
	p := big
	if e < emin {
		p = big - (emin - e)
	}
	if p < 0 {
		return 0, true
	}
	var m uint64
	guard, sticky := 0, false
	for b, n := j*4+lz, 0; b < len(ds)*4 || n < p; b, n = b+1, n+1 {
		bit := 0
		if b < len(ds)*4 {
			bit = (hexVal(ds[b/4]) >> (3 - b%4)) & 1
		}
		switch {
		case n < p:
			m = m*2 + uint64(bit)
		case n == p:
			guard = bit
		case bit == 1:
			sticky = true
		}
	}
	if guard == 1 && (sticky || m%2 == 1) {
		m++
	}
	return math.Ldexp(float64(m), e-p+1), true
}

// jvFloType fills the Double (sty floJava) or Float (sty floJavaF) descriptor:
// the IEEE-754 constants of the width and the static methods over it. Twin
// of jFloType in java-interpreter.abnf and jvFloType in java-rt.metajs.
func jvFloType(o *jsObject, sty uint8) {
	f32 := sty == floJavaF
	mk := func(f float64) interface{} {
		if f32 {
			f = jvmFround(f)
		}
		return jsJFlo{f: f, sty: sty}
	}
	o.set("NaN", mk(math.NaN()))
	o.set("POSITIVE_INFINITY", mk(math.Inf(1)))
	o.set("NEGATIVE_INFINITY", mk(math.Inf(-1)))
	if f32 {
		o.set("MAX_VALUE", mk(math.MaxFloat32))
		o.set("MIN_VALUE", mk(math.SmallestNonzeroFloat32))
		o.set("MIN_NORMAL", mk(0x1p-126))
		o.set("SIZE", float64(32))
		o.set("BYTES", float64(4))
	} else {
		o.set("MAX_VALUE", mk(math.MaxFloat64))
		o.set("MIN_VALUE", mk(math.SmallestNonzeroFloat64))
		o.set("MIN_NORMAL", mk(0x1p-1022))
		o.set("SIZE", float64(64))
		o.set("BYTES", float64(8))
	}
	pred := func(name string, p func(float64) bool) {
		o.set(name, jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return p(rt.toNumber(argAt(args, 0)))
		}))
	}
	pred("isNaN", math.IsNaN)
	pred("isInfinite", func(f float64) bool { return math.IsInf(f, 0) })
	pred("isFinite", func(f float64) bool { return !math.IsNaN(f) && !math.IsInf(f, 0) })
	o.set("compare", jsHostFunc("compare", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return jvDoubleCompare(rt.toNumber(argAt(args, 0)), rt.toNumber(argAt(args, 1)))
	}))
	// The bit patterns, NaN canonical (JLS 4.2.3); hashCode is the boxed
	// value's. Twin of jFloType in java-interpreter.abnf.
	if f32 {
		bits := jsHostFunc("floatToIntBits", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return jvHashFloat(rt.toNumber(argAt(args, 0)))
		})
		o.set("floatToIntBits", bits)
		o.set("hashCode", bits)
	} else {
		o.set("doubleToLongBits", jsHostFunc("doubleToLongBits", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			f := rt.toNumber(argAt(args, 0))
			if f != f {
				return jvBox(0x7ff8000000000000)
			}
			return jvBox(int64(math.Float64bits(f)))
		}))
		o.set("hashCode", jsHostFunc("hashCode", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return jvHashDouble(rt.toNumber(argAt(args, 0)))
		}))
	}
	for name, fn := range map[string]func(float64, float64) float64{
		"sum": func(a, b float64) float64 { return a + b }, "max": jvMaxD, "min": jvMinD,
	} {
		fn := fn
		o.set(name, jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return mk(fn(rt.toNumber(argAt(args, 0)), rt.toNumber(argAt(args, 1))))
		}))
	}
	parse := jsHostFunc("parse", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		rt.jvNpeIfNull(argAt(args, 0))
		return mk(rt.jvParseDbl(rt.toString(argAt(args, 0)), f32))
	})
	if f32 {
		o.set("parseFloat", parse)
	} else {
		o.set("parseDouble", parse)
	}
	o.set("valueOf", jsHostFunc("valueOf", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		if s, ok := argAt(args, 0).(string); ok {
			return mk(rt.jvParseDbl(s, f32))
		}
		return mk(rt.toNumber(argAt(args, 0)))
	}))
	o.set("toString", jsHostFunc("toString", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return rt.jvpStr(mk(rt.toNumber(argAt(args, 0))))
	}))
}

// jvValueOfHost is Integer.valueOf / Long.valueOf: a String parses exactly as
// parseInt / parseLong does, anything else narrows.
func jvValueOfHost(width uint8) *hostFunc {
	return jsHostFunc("valueOf", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		if s, ok := argAt(args, 0).(string); ok {
			return rt.jvParseInt(s, width, jvRadixArg(rt, args))
		}
		return jvNarrow(rt, argAt(args, 0), width)
	})
}

// jvRadixOf is Integer / Long .toString(v, radix), toHexString, toOctalString
// and toBinaryString. toString is the SIGNED reading ("-ff"; a radix outside
// 2..36 means 10); the other three are the UNSIGNED reading AT THE GIVEN WIDTH,
// so Integer.toHexString(-1) is "ffffffff". Lowercase, no leading zeros. Twin of
// jRadixStr in java-interpreter.abnf and jvRadixStr in java-rt.metajs.
func jvRadixOf(rt *jsrt, v interface{}, w uint8, radix int, uns bool) string {
	if radix < 2 || radix > 36 {
		radix = 10
	}
	var p int64
	if f, isFlo := v.(jsJFlo); isFlo {
		p = jvNarrowFloat(f.f, w)
	} else {
		p = jvToLong(rt, v)
	}
	if w == 32 {
		if uns {
			return strconv.FormatUint(uint64(uint32(p)), radix)
		}
		return strconv.FormatInt(int64(int32(p)), radix)
	}
	if uns {
		return strconv.FormatUint(uint64(p), radix)
	}
	return strconv.FormatInt(p, radix)
}

// jvBitOps adds Integer / Long .sum, signum and the bit queries - bitCount,
// numberOfLeadingZeros / numberOfTrailingZeros, highestOneBit /
// lowestOneBit - at the width w. Twin of jAddBitOps in java-interpreter.abnf
// and jvBitOps in languages/lib/java-rt.metajs.
// jvUnsignedOps adds Integer's unsigned readings and bit permutations. Twin of
// jAddUnsigned in java-interpreter.abnf.
func jvUnsignedOps(o *jsObject) {
	u32 := func(rt *jsrt, v interface{}) uint32 { return uint32(jvI(rt, v)) }
	fn := func(name string, f func(rt *jsrt, args []interface{}) interface{}) {
		o.set(name, jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} { return f(rt, args) }))
	}
	fn("reverse", func(rt *jsrt, a []interface{}) interface{} { return float64(int32(bits.Reverse32(u32(rt, argAt(a, 0))))) })
	fn("reverseBytes", func(rt *jsrt, a []interface{}) interface{} {
		return float64(int32(bits.ReverseBytes32(u32(rt, argAt(a, 0)))))
	})
	fn("rotateLeft", func(rt *jsrt, a []interface{}) interface{} {
		return float64(int32(bits.RotateLeft32(u32(rt, argAt(a, 0)), int(jvI(rt, argAt(a, 1))&31))))
	})
	fn("rotateRight", func(rt *jsrt, a []interface{}) interface{} {
		return float64(int32(bits.RotateLeft32(u32(rt, argAt(a, 0)), -int(jvI(rt, argAt(a, 1))&31))))
	})
	fn("toUnsignedLong", func(rt *jsrt, a []interface{}) interface{} { return jvBox(int64(u32(rt, argAt(a, 0)))) })
	fn("compareUnsigned", func(rt *jsrt, a []interface{}) interface{} {
		x, y := u32(rt, argAt(a, 0)), u32(rt, argAt(a, 1))
		switch {
		case x < y:
			return float64(-1)
		case x > y:
			return float64(1)
		}
		return float64(0)
	})
	divide := func(rem bool) func(rt *jsrt, a []interface{}) interface{} {
		return func(rt *jsrt, a []interface{}) interface{} {
			x, y := u32(rt, argAt(a, 0)), u32(rt, argAt(a, 1))
			if y == 0 {
				rt.jvThrowArith("/ by zero")
			}
			if rem {
				return float64(int32(x % y))
			}
			return float64(int32(x / y))
		}
	}
	fn("divideUnsigned", divide(false))
	fn("remainderUnsigned", divide(true))
	fn("toUnsignedString", func(rt *jsrt, a []interface{}) interface{} {
		r := jvRadixArg(rt, a)
		if r < 2 || r > 36 {
			r = 10
		}
		return strconv.FormatUint(uint64(u32(rt, argAt(a, 0))), r)
	})
	fn("parseUnsignedInt", func(rt *jsrt, a []interface{}) interface{} {
		rt.jvParseNull(argAt(a, 0))
		s := rt.toString(argAt(a, 0))
		if strings.HasPrefix(s, "-") {
			rt.jvThrowCls(jvNfeCls, "Illegal leading minus sign on unsigned string "+s+".")
		}
		v := jvToLong(rt, rt.jvParseInt(s, 64, jvRadixArg(rt, a)))
		if uint64(v)>>32 != 0 {
			rt.jvThrowCls(jvNfeCls, "String value "+s+" exceeds range of unsigned int.")
		}
		return float64(int32(uint32(v)))
	})
}

func jvBitOps(o *jsObject, w uint8) {
	bitsOf := func(rt *jsrt, v interface{}) uint64 {
		p := uint64(jvToLong(rt, jvNarrow(rt, v, w)))
		if w == 32 {
			return uint64(uint32(p))
		}
		return p
	}
	ofInt := func(n int) interface{} { return float64(n) }
	o.set("sum", jsHostFunc("sum", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return jvNarrow(rt, rt.jvArith("+", argAt(args, 0), argAt(args, 1)), w)
	}))
	o.set("signum", jsHostFunc("signum", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		v := argAt(args, 0)
		if rt.jvCmp("<", v, float64(0)) {
			return float64(-1)
		}
		if rt.jvCmp(">", v, float64(0)) {
			return float64(1)
		}
		return float64(0)
	}))
	o.set("bitCount", jsHostFunc("bitCount", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return ofInt(bits.OnesCount64(bitsOf(rt, argAt(args, 0))))
	}))
	lead := func(x uint64) int {
		if w == 32 {
			return bits.LeadingZeros32(uint32(x))
		}
		return bits.LeadingZeros64(x)
	}
	trail := func(x uint64) int {
		if w == 32 {
			return bits.TrailingZeros32(uint32(x))
		}
		return bits.TrailingZeros64(x)
	}
	o.set("numberOfLeadingZeros", jsHostFunc("numberOfLeadingZeros", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return ofInt(lead(bitsOf(rt, argAt(args, 0))))
	}))
	o.set("numberOfTrailingZeros", jsHostFunc("numberOfTrailingZeros", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return ofInt(trail(bitsOf(rt, argAt(args, 0))))
	}))
	oneBit := func(rt *jsrt, x uint64, k int) interface{} {
		if x == 0 {
			return jvNarrow(rt, float64(0), w)
		}
		return jvNarrow(rt, jvBox(int64(uint64(1)<<uint(k))), w)
	}
	o.set("highestOneBit", jsHostFunc("highestOneBit", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		x := bitsOf(rt, argAt(args, 0))
		return oneBit(rt, x, int(w)-1-lead(x))
	}))
	o.set("lowestOneBit", jsHostFunc("lowestOneBit", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		x := bitsOf(rt, argAt(args, 0))
		return oneBit(rt, x, trail(x))
	}))
}

// jvRadixMembers adds toString(v [, radix]) and the three unsigned renderings
// to the Integer (w 32) or Long (w 64) descriptor.
func jvRadixMembers(o *jsObject, w uint8) {
	o.set("toString", jsHostFunc("toString", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return jvRadixOf(rt, argAt(args, 0), w, jvRadixArg(rt, args), false)
	}))
	for name, radix := range map[string]int{"toHexString": 16, "toOctalString": 8, "toBinaryString": 2} {
		radix := radix
		o.set(name, jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return jvRadixOf(rt, argAt(args, 0), w, radix, true)
		}))
	}
}

// jvBoolOf is Boolean.valueOf / Boolean.parseBoolean: a STRING argument is true
// iff it equalsIgnoreCase("true"), anything else is its own truth. The twin is
// jvBoolOf in languages/lib/java-rt.metajs.
func jvBoolOf(v interface{}) bool {
	if s, ok := v.(string); ok {
		return strings.EqualFold(s, "true")
	}
	b, _ := v.(bool)
	return b
}

// jvBoxType builds one of java.lang's boxed-primitive types.
func (rt *jsrt) jvBoxType(name string) *jsObject {
	o := newJSObject()
	cmp := func(a, b interface{}) interface{} {
		if rt.jvCmp("<", a, b) {
			return float64(-1)
		}
		if rt.jvCmp(">", a, b) {
			return float64(1)
		}
		return float64(0)
	}
	addCommon := func(width uint8, size, bytes float64) {
		o.set("SIZE", size)
		o.set("BYTES", bytes)
		o.set("valueOf", jsHostFunc("valueOf", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return jvNarrow(rt, argAt(args, 0), width)
		}))
		o.set("toString", jsHostFunc("toString", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return rt.jvpStr(argAt(args, 0))
		}))
		// Short.compare and Byte.compare answer x - y, Integer and Long the sign.
		o.set("compare", jsHostFunc("compare", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			if width < 32 {
				return float64(jvToLong(rt, argAt(args, 0)) - jvToLong(rt, argAt(args, 1)))
			}
			return cmp(argAt(args, 0), argAt(args, 1))
		}))
		o.set("max", jsHostFunc("max", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			l, r := argAt(args, 0), argAt(args, 1)
			if rt.jvCmp(">", l, r) {
				return jvNarrow(rt, l, width)
			}
			return jvNarrow(rt, r, width)
		}))
		o.set("min", jsHostFunc("min", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			l, r := argAt(args, 0), argAt(args, 1)
			if rt.jvCmp("<", l, r) {
				return jvNarrow(rt, l, width)
			}
			return jvNarrow(rt, r, width)
		}))
	}
	switch name {
	case "Integer":
		o.set("MAX_VALUE", float64(math.MaxInt32))
		o.set("MIN_VALUE", float64(math.MinInt32))
		addCommon(32, 32, 4)
		o.set("parseInt", jsHostFunc("parseInt", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			rt.jvParseNull(argAt(args, 0))
			return rt.jvParseInt(rt.toString(argAt(args, 0)), 32, jvRadixArg(rt, args))
		}))
		o.set("valueOf", jvValueOfHost(32))
		jvRadixMembers(o, 32)
		jvBitOps(o, 32)
		jvUnsignedOps(o)
	case "Long":
		o.set("MAX_VALUE", jvBox(math.MaxInt64))
		o.set("MIN_VALUE", jvBox(math.MinInt64))
		addCommon(64, 64, 8)
		o.set("parseLong", jsHostFunc("parseLong", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			rt.jvParseNull(argAt(args, 0))
			return rt.jvParseInt(rt.toString(argAt(args, 0)), 64, jvRadixArg(rt, args))
		}))
		o.set("valueOf", jvValueOfHost(64))
		jvRadixMembers(o, 64)
		jvBitOps(o, 64)
	case "Byte":
		o.set("MAX_VALUE", jsGInt{v: 127, w: 8})
		o.set("MIN_VALUE", jsGInt{v: -128, w: 8})
		addCommon(8, 8, 1)
	case "Short":
		o.set("MAX_VALUE", jsGInt{v: 32767, w: 16})
		o.set("MIN_VALUE", jsGInt{v: -32768, w: 16})
		addCommon(16, 16, 2)
	case "Character":
		o.set("MAX_VALUE", jsChar{code: 65535})
		o.set("MIN_VALUE", jsChar{code: 0})
		o.set("SIZE", float64(16))
		o.set("BYTES", float64(2))
		jvCharMembers(o)
	case "Double":
		jvFloType(o, floJava)
	case "Float":
		jvFloType(o, floJavaF)
	// Boolean carries only a subset; see the note on the layer-2 twin in
	// languages/lib/java-rt.metajs.
	case "Boolean":
		o.set("TRUE", true)
		o.set("FALSE", false)
		bv := jsHostFunc("valueOf", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return jvBoolOf(argAt(args, 0))
		})
		o.set("valueOf", bv)
		o.set("parseBoolean", bv)
		bin := func(name string, f func(a, b bool) interface{}) {
			o.set(name, jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
				a, _ := argAt(args, 0).(bool)
				b, _ := argAt(args, 1).(bool)
				return f(a, b)
			}))
		}
		bin("compare", func(a, b bool) interface{} {
			if a == b {
				return float64(0)
			}
			if a {
				return float64(1)
			}
			return float64(-1)
		})
		bin("logicalAnd", func(a, b bool) interface{} { return a && b })
		bin("logicalOr", func(a, b bool) interface{} { return a || b })
		bin("logicalXor", func(a, b bool) interface{} { return a != b })
		o.set("toString", jsHostFunc("toString", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			if b, _ := argAt(args, 0).(bool); b {
				return "true"
			}
			return "false"
		}))
	}
	return o
}

// Double.compare(x, y) == 0, which is Double.equals and therefore what a record
// component of type double or float is compared with. It is doubleToLongBits
// equality: every NaN collapses to one bit pattern, so NaN equals NaN, and the
// sign of a zero is part of the value, so +0.0 does not equal -0.0. ONE body
// serves both widths - a float box already holds the binary32-rounded value
// (jvmFround), and distinct binary32 values are distinct float64 values.
func jvDoubleCompareEq(x, y float64) bool {
	if math.IsNaN(x) || math.IsNaN(y) {
		return math.IsNaN(x) && math.IsNaN(y)
	}
	if x == 0 && y == 0 {
		return math.Signbit(x) == math.Signbit(y)
	}
	return x == y
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

		// One runtime-raised exception class descriptor, handed over BY NAME at
		// module init so that the externs below can THROW rather than abort.
		m["js_jvexc"] = func(a []uint64) uint64 {
			o, ok := u(a[1]).(*jsObject)
			if !ok {
				return 0
			}
			name := rt.toString(u(a[0]))
			jvExcNamed[name] = o
			switch name {
			case "ArithmeticException":
				jvArithExcCls = o
			case "ArrayIndexOutOfBoundsException":
				jvAioobeCls = o
			case "StringIndexOutOfBoundsException":
				jvSioobeCls = o
			case "NegativeArraySizeException":
				jvNegArrCls = o
			case "NullPointerException":
				jvNpeCls = o
			case "NumberFormatException":
				jvNfeCls = o
			case "IllegalArgumentException":
				jvIaeCls = o
			case "ClassCastException":
				jvCceCls = o
			}
			return 0
		}
		// `a[i]`, read or written. EVERY array access is bounds-checked in Java
		// (JLS 10.4) and a violation throws ArrayIndexOutOfBoundsException - it
		// does not answer a default and it does not extend the array on a write.
		// The emitter emits this for both halves; the write site discards the
		// value and keeps only the check (emitRefPath in java-to-llvm-ir.abnf).
		m["js_jvidx"] = func(a []uint64) uint64 {
			o := u(a[0])
			if arr, isArr := o.(*jsArray); isArr {
				i := jsToInt(rt.toNumber(u(a[1])))
				if i < 0 || i >= len(arr.elems) {
					rt.jvThrowAioobe(i, len(arr.elems))
				}
				return w(arr.elems[i])
			}
			if isUndefOrNull(o) {
				// `int[] z = null; z[0]` - a catchable NullPointerException with a
				// null message, exactly as the interpreter half throws it.
				rt.jvThrowCls(jvNpeCls, jsNull)
			}
			return w(rt.getMember(o, u(a[1])))
		}
		// Does a thrown value match ONE catch-clause type name? The emitter emits
		// one call per name, so a multi-catch `catch (A | B e)` is two calls. The
		// twin of core.excMatches in java-interpreter.abnf and of js_jvexcmatch in
		// languages/lib/java-rt.metajs. A value outside the Throwable hierarchy -
		// this subset lets a program throw any object - is caught by the four
		// general types, which is how `catch (Exception e)` is written in practice.
		m["js_jvexcmatch"] = func(a []uint64) uint64 {
			v := u(a[0])
			n := rt.toString(u(a[1]))
			if rt.isTypeName(v, n) {
				return jsHTrue
			}
			switch n {
			case "Exception", "Throwable", "RuntimeException", "Error":
				return boolH(!jvIsThrowable(v))
			}
			return jsHFalse
		}
		// One dimension of `new T[n]`: answers it, and raises
		// NegativeArraySizeException on a negative one (JLS 15.10.1). The message
		// is the offending size and nothing else.
		m["js_jvnewlen"] = func(a []uint64) uint64 {
			n := jsToInt(rt.toNumber(u(a[0])))
			if n < 0 {
				rt.jvThrowCls(jvNegArrCls, strconv.Itoa(n))
			}
			return a[0]
		}
		// java.lang.String.charAt THROWS StringIndexOutOfBoundsException where the
		// shared arm in abnf/jsrt.go fails the run uncatchably. Only the java
		// grammar emits js_jcharat, so overriding it here is a java-only change -
		// rxExtraExterns runs after the base table is filled.
		baseCharAt := m["js_jcharat"]
		m["js_jcharat"] = func(a []uint64) uint64 {
			if str, isStr := u(a[0]).(string); isStr {
				args, ok := u(a[1]).(*jsArray)
				if ok {
					i := jsToInt(rt.toNumber(argAt(args.elems, 0)))
					units := strUnits(str)
					if i < 0 || i >= len(units) {
						rt.jvThrowSioobe(i, len(units))
					}
				}
			}
			return baseCharAt(a)
		}
		// java.lang.String.substring checks the RANGE - begin > end is out of
		// bounds too - where the shared memberCall CLAMPS through substringRange.
		// A SEPARATE EXTERN, not a wrapper on js_mcall: rxExtraExterns fills ONE
		// map for every language, so overriding js_mcall here would make a C#
		// Substring raise java's message. js_ktsmcall and js_swmcall are the
		// same pattern for the same reason. Only the java emitter emits
		// js_jvsub, and only for the name `substring`.
		// An overload dispatcher: (class, name, spec, args, off). jvOvPick chooses
		// a signature from the arguments past `off` (1 when args[0] is the
		// receiver), and the implementation under "name#sig" is found from the
		// dispatcher's own class upwards and handed the same args array.
		m["js_jvovcall"] = func(a []uint64) uint64 {
			args, ok := u(a[3]).(*jsArray)
			if !ok {
				rt.fail("js_jvovcall args must be an array")
			}
			slot := rt.toString(u(a[1])) + "#" + jvOvPick(rt.toString(u(a[2])), args.elems, jsToInt(rt.toNumber(u(a[4]))))
			owner := jvStaticOwner(u(a[0]), slot)
			if owner == nil {
				rt.fail("unknown method '%s'", slot)
			}
			return w(rt.callH(owner.props[slot], jsUndef, args.elems, a[3]))
		}
		baseMcall := m["js_mcall"]
		// A call javac resolved from static types: (receiver, name, sig, args).
		// The implementation under "name#sig" is found from the receiver's class
		// upwards - or from a class descriptor itself, for a static call. A
		// receiver with no such slot (a lambda, a class that declares the method
		// under another signature) takes the ordinary js_mcall. Twin of jSigCall
		// in java-interpreter.abnf and of js_jvsigcall in
		// languages/lib/java-rt.metajs.
		m["js_jvsigcall"] = func(a []uint64) uint64 {
			args, ok := u(a[3]).(*jsArray)
			if !ok {
				rt.fail("js_jvsigcall args must be an array")
			}
			slot := rt.toString(u(a[1])) + "#" + rt.toString(u(a[2]))
			if o, isObj := u(a[0]).(*jsObject); isObj {
				if isClassDesc(o) {
					if owner := jvStaticOwner(o, slot); owner != nil {
						return w(rt.callH(owner.props[slot], jsUndef, args.elems, a[3]))
					}
				} else if owner := jvStaticOwner(o.props["__class"], slot); owner != nil {
					return w(rt.call(owner.props[slot], jsUndef, append([]interface{}{o}, args.elems...)))
				}
			}
			return baseMcall([]uint64{a[0], a[1], a[3]})
		}
		// String.toCharArray: one char per UTF-16 unit. Anything else is an
		// ordinary call of a method by that name.
		// An array's clone() is a new array of the same type and length holding
		// the same elements, a shallow copy (JLS 10.7); any other receiver
		// dispatches.
		// Twin of the clone arm of mcall in java-interpreter.abnf and js_jvclone
		// in languages/lib/java-rt.metajs.
		m["js_jvclone"] = func(a []uint64) uint64 {
			if arr, ok := u(a[0]).(*jsArray); ok {
				return w(&jsArray{elems: append([]interface{}(nil), arr.elems...), kind: arr.kind})
			}
			return baseMcall([]uint64{a[0], w("clone"), w(&jsArray{})})
		}
		// Tags an array with the binary name of its declared type (jvArrTagKind).
		m["js_jvarrtag"] = func(a []uint64) uint64 {
			if arr, ok := u(a[0]).(*jsArray); ok {
				arr.kind = jvArrTagKind(rt.toString(u(a[1])))
			}
			return a[0]
		}
		m["js_jvchars"] = func(a []uint64) uint64 {
			if s, ok := u(a[0]).(string); ok {
				units := strUnits(s)
				out := make([]interface{}, len(units))
				for i, c := range units {
					out[i] = jsChar{code: int32(c)}
				}
				return w(&jsArray{elems: out})
			}
			return baseMcall([]uint64{a[0], w("toCharArray"), w(&jsArray{})})
		}
		// new String(...): (), (String), (char[]) and (char[] or int[] of code
		// points, offset, count) - and a CharSequence's text. A String has no
		// identity in this value model, so the copy is the same value. Twin of
		// jNewString in java-interpreter.abnf and js_jvnewstr in
		// languages/lib/java-rt.metajs.
		m["js_jvnewstr"] = func(a []uint64) uint64 {
			args, _ := u(a[0]).(*jsArray)
			if args == nil || len(args.elems) == 0 {
				return rt.wrapStr("")
			}
			arr, isArr := args.elems[0].(*jsArray)
			if !isArr {
				return rt.wrapStr(rt.jvpStr(args.elems[0]))
			}
			n := len(arr.elems)
			off, cnt := 0, n
			if len(args.elems) >= 3 {
				off = jsToInt(rt.toNumber(args.elems[1]))
				cnt = jsToInt(rt.toNumber(args.elems[2]))
				if off < 0 || cnt < 0 || off > n-cnt {
					rt.jvThrowCls(jvSioobeCls, fmt.Sprintf("Range [%d, %d + %d) out of bounds for length %d", off, off, cnt, n))
				}
			}
			units := make([]uint16, 0, cnt)
			for _, e := range arr.elems[off : off+cnt] {
				if c, ok := e.(jsChar); ok {
					units = append(units, uint16(c.code))
				} else {
					units = utf16.AppendRune(units, rune(jsToInt(rt.toNumber(e))))
				}
			}
			return rt.wrapStr(strFromUnits(units))
		}
		m["js_jvsub"] = func(a []uint64) uint64 {
			if str, isStr := u(a[0]).(string); isStr {
				if args, ok := u(a[1]).(*jsArray); ok {
					units := strUnits(str)
					bi := jsToInt(rt.toNumber(argAt(args.elems, 0)))
					ei := len(units)
					if len(args.elems) > 1 {
						ei = jsToInt(rt.toNumber(argAt(args.elems, 1)))
					}
					if bi < 0 || ei > len(units) || bi > ei {
						rt.jvThrowRange(bi, ei, len(units))
					}
				}
			}
			return baseMcall([]uint64{a[0], w("substring"), a[1]})
		}
		// One binary integral operator: (op, left, right). A double operand keeps
		// the FLOAT path of jsrtjvm.go, which is where the two boxes meet.
		m["js_jvarith"] = func(a []uint64) uint64 {
			op, l, r := opOf(a[0]), u(a[1]), u(a[2])
			if jvmIsFlo(l) || jvmIsFlo(r) {
				return w(rt.jvmArith(op[0], l, r))
			}
			if !jvIsIntegral(l) || !jvIsIntegral(r) {
				rt.jvNpeIfNull(l)
				rt.jvNpeIfNull(r)
				// & | ^ on two booleans are Java's non-short-circuit boolean
				// operators; anything else is not a program javac accepts.
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
				return w(rt.jvmArith(op[0], l, r))
			}
			return w(rt.jvArith(op, l, r))
		}
		m["js_jvshift"] = func(a []uint64) uint64 {
			rt.jvNpeIfNull(u(a[1]))
			rt.jvNpeIfNull(u(a[2]))
			return w(rt.jvShift(opOf(a[0]), u(a[1]), u(a[2])))
		}
		// An ordered comparison. Two integers compare at the promoted width;
		// anything else (a double, a string pair) keeps the shared comparison.
		m["js_jvcmp"] = func(a []uint64) uint64 {
			op, l, r := opOf(a[0]), u(a[1]), u(a[2])
			if jvIsIntegral(l) && jvIsIntegral(r) && !jvmIsFlo(l) && !jvmIsFlo(r) {
				return boolH(rt.jvCmp(op, l, r))
			}
			// A float operand promotes the other side to float too (JLS 5.6.2),
			// so 16777217 < 16777216f is FALSE - the int converts to 1.6777216E7.
			rt.jvNpeIfNull(l)
			rt.jvNpeIfNull(r)
			l, r = jvmFloCmpPair(rt, l, r)
			c := rt.jsCompare(l, r)
			// jsCompare answers the SENTINEL 2 for a NaN operand ("every
			// relation is false"), which is exactly Java's rule (JLS 15.20.1:
			// if either operand is NaN then <, <=, > and >= all produce false);
			// it must not be read as an ordering. Twins: jsrtcsharp.go and
			// js_jvcmp in languages/lib/java-rt.metajs.
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
		// A cast or a declared integral type: (value, bits).
		m["js_jvconv"] = func(a []uint64) uint64 {
			return w(jvNarrow(rt, u(a[0]), uint8(jsToInt(rt.toNumber(u(a[1]))))))
		}
		// A declared type applied to an initializer: only an INTEGRAL value
		// narrows, so `Object o = "s"` and a lambda pass through untouched.
		m["js_jvadopt"] = func(a []uint64) uint64 {
			v := u(a[0])
			if !jvIsIntegral(v) {
				rt.jvNpeIfNull(v)
				return a[0]
			}
			return w(jvNarrow(rt, v, uint8(jsToInt(rt.toNumber(u(a[1]))))))
		}
		// An integral literal, passed as DIGITS: emitNum would already have
		// rounded 9223372036854775807 to 9223372036854776000 on the way into the
		// module. (text, radix, isLong).
		m["js_jvlit"] = func(a []uint64) uint64 {
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
					continue // a '_' digit separator
				}
				if d >= radix {
					continue
				}
				acc = acc*radix + d
			}
			if rt.truthy(u(a[2])) {
				return w(jvBox(int64(acc)))
			}
			return rt.wrapNum(float64(int32(acc)))
		}
		// -x and ~x, both at the PROMOTED width (JLS 5.6.1): a long stays a long,
		// everything integral else becomes an int.
		m["js_jvneg"] = func(a []uint64) uint64 {
			v := u(a[0])
			if f, ok := v.(jsJFlo); ok {
				return w(jsJFlo{f: -f.f, sty: f.sty})
			}
			if jvIsLong(v) {
				return w(jvBox(-giVal(rt, v)))
			}
			rt.jvNpeIfNull(v)
			return rt.wrapNum(float64(-jvI(rt, v)))
		}
		m["js_jvnot"] = func(a []uint64) uint64 {
			v := u(a[0])
			if jvIsLong(v) {
				return w(jvBox(^giVal(rt, v)))
			}
			rt.jvNpeIfNull(v)
			return rt.wrapNum(float64(^jvI(rt, v)))
		}
		// ++ / -- keep the type of what they step, which is the same implicit
		// narrowing a compound assignment carries (JLS 15.14.2).
		m["js_jvstep"] = func(a []uint64) uint64 {
			v, d := u(a[0]), giVal(rt, u(a[1]))
			switch t := v.(type) {
			case jsChar:
				return w(jsChar{code: int32((int64(t.code) + d) & 65535)})
			case jsJFlo:
				// ++/-- keeps the operand's own type, WIDTH included: a float
				// steps as a float (rtF32Round in lib/runtime.metajs).
				if t.sty == floJavaF {
					return w(jsJFlo{f: jvmFround(t.f + float64(d)), sty: floJavaF})
				}
				return w(jsJFlo{f: t.f + float64(d), sty: t.sty})
			case jsGInt:
				return w(jvNarrow(rt, rt.jvArith("+", t, float64(d)), t.w))
			}
			rt.jvNpeIfNull(v)
			return rt.wrapNum(float64(jvI(rt, v) + int32(d)))
		}
		// The result of a COMPOUND assignment: the operation, then the implicit
		// narrowing cast back to the LEFT operand's type (JLS 15.26.2). So
		// `byte b = 10; b += 300` is (byte)310 == 54 and `int i = 5; i += 1.5` is
		// (int)6.5 == 6. A String left operand keeps its concatenation.
		m["js_jvnarrowlike"] = func(a []uint64) uint64 {
			l, v := u(a[0]), u(a[1])
			switch t := l.(type) {
			case jsJFlo:
				// A float left operand narrows the result back to a float:
				// `float f = 1.1f; f += 0.1` is (float)(f + 0.1) and not a double.
				if t.sty == floJavaF {
					return w(jsJFlo{f: jvmFround(rt.toNumber(v)), sty: floJavaF})
				}
				if f, ok := v.(jsJFlo); ok {
					return w(f)
				}
				return w(jsJFlo{f: rt.toNumber(v), sty: t.sty})
			case jsChar:
				return w(jsChar{code: int32(giVal(rt, v) & 65535)})
			case jsGInt:
				return w(jvNarrow(rt, v, t.w))
			case float64:
				if jvIsIntegral(v) || jvmIsFlo(v) {
					return w(jvNarrow(rt, v, 32))
				}
			}
			return a[1]
		}
		// The boxed-primitive types Integer / Long / Byte / Short / Character, one
		// object per name. MIN_VALUE / MAX_VALUE are the whole reason a sized
		// integer had to exist: Long.MAX_VALUE cannot be written as a plain number
		// at all, and Integer.MAX_VALUE + 1 has to wrap rather than grow. The twin
		// is hostGlobals["Integer"] etc in java-interpreter.abnf.
		m["js_jvboxtype"] = func(a []uint64) uint64 { return w(rt.jvBoxType(rt.toString(u(a[0])))) }
		// A (char) cast: narrow to 16 bits and re-box. char is the one integral
		// type that is UNSIGNED, so the mask rather than a sign extension is the
		// whole answer - (char)-1 is 65535 and (char)70000 is 4464. A floating
		// operand saturates at the int range first (JLS 5.1.3): (char)1e10 is
		// 65535.
		m["js_jvchar"] = func(a []uint64) uint64 {
			return w(jsChar{code: int32(int64(jvNarrow(rt, u(a[0]), 32).(float64)) & 65535)})
		}
		// The plain-number reading a raw consumer needs (an array index, a length,
		// a bound): a box is not a float64 and every such consumer wants one.
		m["js_jvnum"] = func(a []uint64) uint64 {
			if b, ok := u(a[0]).(jsGInt); ok {
				return rt.wrapNum(float64(b.v))
			}
			rt.jvNpeIfNull(u(a[0]))
			return a[0]
		}
		// A null reference used as an object or unboxed, at a site the emitter
		// checks itself (a Boolean condition, `throw null`, a switch).
		m["js_jvnpe"] = func(a []uint64) uint64 {
			rt.jvThrowCls(jvNpeCls, jsNull)
			return 0
		}

		// System.out.println(x) / System.out.print(x). The argument ARRAY of the
		// emitted closure arrives whole, so the no-argument println(), which
		// writes just the line terminator, is distinguishable from println(null).
		m["js_jvprint"] = func(a []uint64) uint64 {
			fmt.Fprintln(outWriter, jvOutText(jvpFirst(rt, u(a[0]))))
			return 0
		}
		m["js_jvwrite"] = func(a []uint64) uint64 {
			fmt.Fprint(outWriter, jvOutText(jvpFirst(rt, u(a[0]))))
			return 0
		}
		// System.arraycopy(src, srcPos, dest, destPos, length), its argument
		// array whole. Twin of js_jvarraycopy in languages/lib/java-rt.metajs,
		// which documents the checks.
		m["js_jvarraycopy"] = func(a []uint64) uint64 {
			args := u(a[0]).(*jsArray).elems
			src, sok := args[0].(*jsArray)
			dst, dok := args[2].(*jsArray)
			if isUndefOrNull(args[0]) || isUndefOrNull(args[2]) {
				rt.jvThrowCls(jvNpeCls, jsNull)
			}
			if !sok || !dok {
				rt.fail("java: System.arraycopy of a non-array is not supported")
			}
			sp, dp, n := int(jvToLong(rt, args[1])), int(jvToLong(rt, args[3])), int(jvToLong(rt, args[4]))
			switch {
			case sp < 0:
				rt.jvThrowCls(jvAioobeCls, fmt.Sprintf("arraycopy: source index %d out of bounds for %s", sp, jvAcType(src)))
			case dp < 0:
				rt.jvThrowCls(jvAioobeCls, fmt.Sprintf("arraycopy: destination index %d out of bounds for %s", dp, jvAcType(dst)))
			case n < 0:
				rt.jvThrowCls(jvAioobeCls, fmt.Sprintf("arraycopy: length %d is negative", n))
			case sp+n > len(src.elems):
				rt.jvThrowCls(jvAioobeCls, fmt.Sprintf("arraycopy: last source index %d out of bounds for %s", sp+n, jvAcType(src)))
			case dp+n > len(dst.elems):
				rt.jvThrowCls(jvAioobeCls, fmt.Sprintf("arraycopy: last destination index %d out of bounds for %s", dp+n, jvAcType(dst)))
			}
			copy(dst.elems[dp:dp+n], src.elems[sp:sp+n])
			return 0
		}
		// ONE record component of two record instances, compared the way JLS
		// 8.10.3 says a record's canonical equals compares it - which is NOT ===
		// in either direction:
		//
		//   a float / double component  Float.compare / Double.compare == 0, so
		//                               NaN EQUALS NaN and +0.0 does NOT equal
		//                               -0.0. Both are the opposite of ===.
		//   any other primitive         ==  (js_jchareq, chars included)
		//   a reference                 Objects.equals(a, b), i.e. a == b || (a
		//                               != null && a.equals(b)) - so a nested
		//                               record compares component-wise and a
		//                               class with a user-declared equals uses
		//                               it, where identity said unequal.
		//
		// A class WITHOUT an equals (and an array, which does not override it)
		// keeps identity, because Object.equals is identity. The layer-2 twin is
		// js_jvaleq in languages/lib/java-rt.metajs and it matches arm for arm.
		m["js_jvaleq"] = func(a []uint64) uint64 {
			x, y := u(a[0]), u(a[1])
			// TWO PRIMITIVES OF DIFFERENT KINDS ARE NEVER EQUAL. Integer#equals
			// asks `instanceof Integer` before it compares, so
			// `((Object) 1).equals(1L)` is FALSE in java - while the floor's
			// strict_eq deliberately compares a sized integer and a plain number
			// BY VALUE (chapter 7.9 of the manual), and js_jchareq below falls
			// through to it. The same holds for a record component: an
			// Object-typed component holding 1 does not equal one holding 1L.
			// Only PRIMITIVES are screened - kind 0 (an instance, an array,
			// null) must still reach the dispatch below, because a declared
			// equals(Object) legitimately compares against a String.
			kx, ky := jvEqKind(x), jvEqKind(y)
			if kx != 0 && ky != 0 && kx != ky {
				return jsHFalse
			}
			if fx, ok := x.(jsJFlo); ok {
				if fy, ok2 := y.(jsJFlo); ok2 {
					return boolH(jvDoubleCompareEq(fx.f, fy.f))
				}
			}
			if m["js_jchareq"](a) == jsHTrue {
				return jsHTrue
			}
			o, ok := x.(*jsObject)
			if !ok {
				return jsHFalse
			}
			// The __class / __super walk rt.memberCall uses, with the guard < 64
			// cap chapter 7.9 of the manual keeps everywhere else (a cyclic
			// __super hangs the oracle, not us).
			cls := o.props["__class"]
			for guard := 0; guard < 64; guard++ {
				clsObj, isObj := cls.(*jsObject)
				if !isObj {
					return jsHFalse
				}
				if eq, has := clsObj.props["equals"]; has && isCallable(eq) {
					// `=== true`, not truthiness: equals is declared boolean and
					// layer 2's twin cannot spell truthiness on an anytype
					// without disagreeing with this line for a program the
					// grammar accepts and javac would not.
					res, isBool := rt.call(eq, jsUndef, []interface{}{x, y}).(bool)
					return boolH(isBool && res)
				}
				cls = clsObj.props["__super"]
			}
			return jsHFalse
		}
		// Object#hashCode and every override of it the JLS pins exactly, plus
		// the record combination this project pins (see jvHash for what is
		// specified and what is a decision). The layer-2 twin is js_jhash in
		// languages/lib/java-rt.metajs.
		m["js_jhash"] = func(a []uint64) uint64 { return rt.wrapNum(rt.jvHash(u(a[0]))) }
		// Is this value a CLASS DESCRIPTOR? The one thing an emitted probe needs to
		// know to apply JLS 6.4.2 - a variable name obscures a type name - because
		// a class descriptor is declared in the same scope chain a bare name reads
		// (see makeVarRef in java-to-llvm-ir.abnf). The layer-2 twin is js_jisclass
		// in languages/lib/java-rt.metajs.
		m["js_jisclass"] = func(a []uint64) uint64 {
			o, ok := u(a[0]).(*jsObject)
			if !ok {
				return jsHFalse
			}
			return boolH(o.props["__isclass"] == true)
		}
		// THE OUTER HOP. An inner or anonymous class body names a
		// field of its ENCLOSING instance unqualified, and javac resolves it there
		// (JLS 6.5.6.1: the innermost enclosing class that declares the name wins).
		// The instance carries that enclosing instance as __outer - every creation
		// site of a nested or anonymous class records one - so the search is `this`,
		// the statics of this' class chain, then __outer, and out.
		//
		// Answers the OWNER object, which the caller then reads or writes the name
		// on, so one extern serves both directions. The twins are jOuterOwner in
		// languages/java-interpreter.abnf (which needs no extern: it holds the
		// objects) and js_jouter in languages/lib/java-rt.metajs.
		m["js_jouter"] = func(a []uint64) uint64 {
			name := rt.toString(u(a[1]))
			v := u(a[0])
			for guard := 0; guard < 64; guard++ {
				o, ok := v.(*jsObject)
				if !ok {
					break
				}
				// "not undefined" rather than an own-key test, because the
				// layer-2 twin has no own-key primitive and the two must agree.
				// A declared field is never undefined (the emitted constructor
				// gives every one of them 0, false or null before the body runs).
				if pv, has := o.props[name]; has && pv != interface{}(jsUndef) {
					return w(o)
				}
				if own := jvStaticOwner(o.props["__class"], name); own != nil {
					return w(own)
				}
				v = o.props["__outer"]
			}
			rt.fail("unknown name: " + name)
			return jsHUndefined
		}
		// Outer.this: out along __outer until an instance whose class chain carries
		// that name. One unconditional __outer hop is wrong at two levels of
		// nesting; the twin is makeOuterThis in java-interpreter.abnf.
		m["js_jouterthis"] = func(a []uint64) uint64 {
			name := rt.toString(u(a[1]))
			v := u(a[0])
			for guard := 0; guard < 64; guard++ {
				o, ok := v.(*jsObject)
				if !ok {
					break
				}
				if jvClassIs(o.props["__class"], name) {
					return w(o)
				}
				v = o.props["__outer"]
			}
			rt.fail("no enclosing instance of " + name)
			return jsHUndefined
		}
		// String.valueOf(x) and the string form used by `+`.
		m["js_jvstr"] = func(a []uint64) uint64 { return rt.wrapStr(rt.jvpStr(u(a[0]))) }
		// super.m(args) - (super class, this, name, args). A call no user class
		// answers reaches java.lang.Object: toString (or the one Throwable, Enum
		// or Record put in its place), equals (identity) and hashCode (the
		// identity hash). Twin of jSuperObject in languages/java-interpreter.abnf
		// and js_jvsuper in languages/lib/java-rt.metajs.
		baseSuper := m["js_supercall"]
		m["js_jvsuper"] = func(a []uint64) uint64 {
			self, isObj := u(a[1]).(*jsObject)
			args, isArr := u(a[3]).(*jsArray)
			if !isObj || !isArr || isClassDesc(self) {
				return baseSuper(a)
			}
			name := rt.toString(u(a[2]))
			for cls, g := u(a[0]), 0; g < 64; g++ {
				clsObj, ok := cls.(*jsObject)
				if !ok {
					break
				}
				if f, ok := clsObj.props[name]; ok && isCallable(f) {
					return baseSuper(a)
				}
				cls = clsObj.props["__super"]
			}
			switch n := len(args.elems); {
			case name == "toString" && n == 0:
				return rt.wrapStr(rt.jvpInherited(self, 0))
			case name == "equals" && n == 1:
				other, same := args.elems[0].(*jsObject)
				return w(same && other == self)
			case name == "hashCode" && n == 0:
				return w(jvpIdentNum(self))
			}
			return baseSuper(a)
		}
		// Java's `+`. Identical to js_jadd (string concat / 32 bit int add / float
		// add) except that the string case renders through jvpStr, so `"" + arr`
		// and `"" + obj` agree with println instead of falling through to
		// JavaScript's ToString ("1,2,3", "[object Object]").
		m["js_jvadd"] = func(a []uint64) uint64 {
			l, r := u(a[0]), u(a[1])
			_, ls := l.(string)
			_, rs := r.(string)
			if ls || rs {
				return rt.wrapStr(strConcat(rt.jvpStr(l), rt.jvpStr(r)))
			}
			if jvmIsFlo(l) || jvmIsFlo(r) {
				// jvmArith, not a hand-rolled add: the float WIDTH needs both
				// operands converted first and the sum rounded after, and
				// jvmStyleOf gets `float + double` backwards.
				return w(rt.jvmArith('+', l, r))
			}
			// Two integral operands: int + int wraps at 32 bits, anything with a
			// long in it at 64 (JLS 15.18.2 on top of the 5.6.2 promotion).
			if jvIsIntegral(l) && jvIsIntegral(r) {
				return w(rt.jvArith("+", l, r))
			}
			rt.jvNpeIfNull(l)
			rt.jvNpeIfNull(r)
			return w(rt.jsAdd(l, r))
		}
	})
	// js_jvsub reads its argument array in position 1, so the handle must arrive
	// as the array itself rather than as a value.
	jsThroughArgs["js_jvsub"] = 1 << 1
}

// jvEqKind is the java TYPE of a primitive value, for Object#equals: 0 means "not
// a primitive" (an instance, an array, null, undefined) and every other number is
// one boxed type. The split between 1 and 2 is Float vs Double, which
// Float.equals(Double) rejects.
func jvEqKind(v interface{}) int {
	switch t := v.(type) {
	case jsJFlo:
		if t.sty == floJavaF {
			return 1
		}
		return 2
	case jsGInt:
		return 3
	case jsChar:
		return 4
	case bool:
		return 5
	case string:
		return 6
	case float64:
		return 7
	}
	return 0
}

// jvOvPick is overload resolution at a dispatcher (js_jvovcall): the twin of
// jOvPick in languages/lib/java-decltypes.js, whose header describes the spec
// string. It answers the chosen candidate's signature key.
func jvOvPick(spec string, args []interface{}, off int) string {
	ents := strings.Split(spec, "|")
	n := len(args) - off
	toksOf := func(e string) ([]string, string) {
		at := strings.IndexByte(e, '@')
		if at == 0 {
			return nil, e[1:]
		}
		return strings.Split(e[:at], ","), e[at+1:]
	}
	for phase := 1; phase <= 3; phase++ {
		for _, e := range ents {
			toks, raw := toksOf(e)
			p := len(toks)
			isVar := p > 0 && strings.HasSuffix(toks[p-1], "...")
			ok := true
			if phase < 3 {
				if p != n {
					continue
				}
				for i := 0; i < p && ok; i++ {
					t := toks[i]
					if isVar && i == p-1 {
						t = strings.TrimSuffix(t, "...") + "[]"
					}
					ok = jvOvAccepts(t, args[off+i], phase)
				}
			} else {
				if !isVar || n < p-1 {
					continue
				}
				for j := 0; j < n && ok; j++ {
					t := strings.TrimSuffix(toks[p-1], "...")
					if j < p-1 {
						t = toks[j]
					}
					ok = jvOvAccepts(t, args[off+j], 2)
				}
			}
			if ok {
				return raw
			}
		}
	}
	for _, e := range ents {
		if toks, raw := toksOf(e); len(toks) == n {
			return raw
		}
	}
	_, raw := toksOf(ents[0])
	return raw
}

// jvOvKind names an argument's runtime type as jOvKind does.
func jvOvKind(v interface{}) string {
	switch t := v.(type) {
	case nil, jsUndefT, jsNullT:
		return "null"
	case bool:
		return "boolean"
	case string:
		return "String"
	case float64:
		return "int"
	case jsChar:
		return "char"
	case jsJFlo:
		if t.sty == floJavaF {
			return "float"
		}
		return "double"
	case jsGInt:
		switch t.w {
		case 64:
			return "long"
		case 16:
			return "short"
		case 8:
			return "byte"
		}
		return "int"
	case *jsArray:
		return "[]"
	case *jsObject:
		return "obj"
	}
	if isCallable(v) {
		return "fn"
	}
	return "obj"
}

var jvOvPrims = ",byte,short,char,int,long,float,double,boolean,"

func jvOvIsPrim(t string) bool { return strings.Contains(jvOvPrims, ","+t+",") }

// jvOvWiden is JLS 5.1.2's widening primitive conversion, identity included.
func jvOvWiden(k, t string) bool {
	if k == t {
		return true
	}
	if k == "boolean" || t == "boolean" || t == "char" || t == "byte" {
		return false
	}
	if k == "char" {
		return t != "short"
	}
	const order = ",byte,short,int,long,float,double,"
	return strings.Index(order, ","+t+",") > strings.Index(order, ","+k+",")
}

// jvOvAccepts is jOvAccepts of languages/lib/java-decltypes.js.
func jvOvAccepts(t string, v interface{}, phase int) bool {
	k := jvOvKind(v)
	if strings.HasSuffix(t, "[]") {
		if k == "null" {
			return true
		}
		arr, ok := v.(*jsArray)
		if !ok {
			return false
		}
		if len(arr.elems) == 0 {
			return true
		}
		et := t[:len(t)-2]
		if jvOvIsPrim(et) {
			return jvOvKind(arr.elems[0]) == et
		}
		return jvOvAccepts(et, arr.elems[0], 1)
	}
	if jvOvIsPrim(t) {
		return jvOvIsPrim(k) && jvOvWiden(k, t)
	}
	if k == "null" {
		return true
	}
	if jvOvIsPrim(k) {
		if phase < 2 {
			return false
		}
		switch t {
		case "*", "Object":
			return true
		case "Number":
			return k != "char" && k != "boolean"
		}
		box := map[string]string{"int": "Integer", "char": "Character", "long": "Long", "double": "Double",
			"float": "Float", "short": "Short", "byte": "Byte", "boolean": "Boolean"}
		return t == box[k]
	}
	if t == "*" {
		return true
	}
	if k == "fn" {
		return strings.HasPrefix(t, "~")
	}
	if t == "Object" {
		return true
	}
	if k == "[]" {
		return false
	}
	if k == "String" {
		return t == "String"
	}
	// A class token, or "#" / "~" + a declared interface's name (jOvTok in
	// languages/lib/java-decltypes.js); classIsNamed walks the interfaces too.
	o, ok := v.(*jsObject)
	if !ok {
		return false
	}
	cls, _ := o.props["__class"].(*jsObject)
	return classIsNamed(cls, strings.TrimLeft(t, "#~"))
}

// jvStaticOwner finds the class descriptor in cls' __super chain that carries
// `name` as an own property. In the COMPILED world a static field is an ordinary
// property of the descriptor (the emitter js_set's it there), so the walk is the
// same __super walk method dispatch does, with the guard < 64 cap chapter 7.9 of
// the manual keeps on every one of them.
func jvStaticOwner(cls interface{}, name string) *jsObject {
	for guard := 0; guard < 64; guard++ {
		clsObj, ok := cls.(*jsObject)
		if !ok {
			return nil
		}
		if pv, has := clsObj.props[name]; has && pv != interface{}(jsUndef) {
			return clsObj
		}
		cls = clsObj.props["__super"]
	}
	return nil
}

// jvClassIs answers whether cls, or anything it extends, is named nm - what
// Outer.this walks out until it finds.
func jvClassIs(cls interface{}, nm string) bool {
	for guard := 0; guard < 64; guard++ {
		clsObj, ok := cls.(*jsObject)
		if !ok {
			return false
		}
		if n, _ := clsObj.props["__name"].(string); n == nm {
			return true
		}
		cls = clsObj.props["__super"]
	}
	return false
}

// jvpFirst renders the first element of a print call's argument array. An empty
// array is the no-argument System.out.println(), which prints nothing but the
// terminator.
func jvpFirst(rt *jsrt, args interface{}) string {
	arr, ok := args.(*jsArray)
	if !ok {
		return rt.jvpStr(args)
	}
	if len(arr.elems) == 0 {
		return ""
	}
	return rt.jvpStr(arr.elems[0])
}

// ----------------------------------------------------------------------------
// Rendering

// jvpStr is String.valueOf: what println writes and what `+` concatenates.
func (rt *jsrt) jvpStr(v interface{}) string { return rt.jvpRender(v, 0) }

func (rt *jsrt) jvpRender(v interface{}, depth int) string {
	if depth > 32 {
		return "..."
	}
	switch t := v.(type) {
	case jsNullT, jsUndefT:
		return "null"
	case string:
		return t
	case *jsArray:
		return jvpArrayTag(t) + "@" + jvpHash(t)
	case *jsObject:
		return rt.jvpObj(t, depth)
	}
	return rt.toString(v)
}

// jvpObj renders the object shapes the two Java grammars build.
func (rt *jsrt) jvpObj(o *jsObject, depth int) string {
	// A user toString() wins over every built-in rendering, as it does in Java.
	if f, ok := jvpFindMember(o, "toString"); ok {
		return rt.jvpRender(rt.call(f, o, []interface{}{o}), depth+1)
	}
	return rt.jvpInherited(o, depth)
}

// jvpInherited is the toString an object inherits from the library when no
// class on its chain declares one: Throwable's, Enum's, Record's or Object's.
// It is also what `super.toString()` reaches past the last user class.
func (rt *jsrt) jvpInherited(o *jsObject, depth int) string {
	// java.lang.Throwable#toString: the BINARY class name and, only when
	// getMessage() is non-null, ": " and the message - println(new
	// RuntimeException("x")) is "java.lang.RuntimeException: x" (java 24.0.2).
	// It carries NO stack trace (printStackTrace() is the method that does).
	// The twins are jThrowableStr in languages/java-interpreter.abnf and
	// jvThrowableStr in languages/lib/java-rt.metajs, and jvmExcTopLine in
	// abnf/jsrtjvm.go prints exactly this text after the launcher's fixed prefix.
	if jvIsThrowable(o) {
		return rt.jvmThrowableStr(o)
	}
	// The class descriptor itself (println(Box.class) has no spelling here, but
	// the interpreter half renders one this way, so the two agree).
	if tag, ok := o.props["__isclass"]; ok && tag == true {
		if n, ok := o.props["__name"].(string); ok {
			return "class " + n
		}
	}
	cls, _ := o.props["__class"].(*jsObject)
	if cls == nil {
		return rt.toString(o)
	}
	// An enum constant without an override prints its constant name (the
	// java.lang.Enum#toString contract).
	if n, ok := o.props["__ename"].(string); ok {
		return n
	}
	// A record prints its canonical form, P[x=1, y=2] (the java.lang.Record
	// contract). __record is the component name list the class carries. The name
	// here is the SIMPLE one - the generated toString uses getSimpleName, so a
	// nested record is `P[...]` and not `Main$P[...]`.
	if comps, ok := cls.props["__record"].(*jsArray); ok {
		parts := make([]string, 0, len(comps.elems))
		for _, c := range comps.elems {
			name, _ := c.(string)
			parts = append(parts, name+"="+rt.jvpRender(o.props[name], depth+1))
		}
		simple, _ := cls.props["__name"].(string)
		return simple + "[" + strings.Join(parts, ", ") + "]"
	}
	return jvpClassName(cls) + "@" + jvpHash(o)
}

// jvpClassName is the class' BINARY name: a nested type is spelled Outer$Inner,
// the way java.lang.Class#getName does, and the grammars record that as __bname
// while they walk the declaration. A class that carries none (an anonymous or
// locally built descriptor) falls back to its simple name.
func jvpClassName(cls *jsObject) string {
	if b, ok := cls.props["__bname"].(string); ok && b != "" {
		return b
	}
	if n, ok := cls.props["__name"].(string); ok {
		return n
	}
	return "Object"
}

// jvpFindMember walks the __class / __super chain for a callable member, the same
// walk the two grammars' method dispatch does.
func jvpFindMember(o *jsObject, name string) (interface{}, bool) {
	for cls := o.props["__class"]; cls != nil; {
		clsObj, ok := cls.(*jsObject)
		if !ok {
			break
		}
		if mth, ok := clsObj.props[name]; ok && isCallable(mth) {
			return mth, true
		}
		cls = clsObj.props["__super"]
	}
	return nil, false
}

// An array created with a declared type carries that type's binary name as its
// kind number, jvArrKindBase and up so it meets no other language's kinds;
// jvArrTags maps the number back. The numbering is this process's own and only
// ever grows. Twin of jArrTag in java-interpreter.abnf and js_jvarrtag in
// languages/lib/java-rt.metajs.
const jvArrKindBase = 1000

var jvArrTags struct {
	sync.Mutex
	names []string
	kinds map[string]int
}

func jvArrTagKind(desc string) int {
	jvArrTags.Lock()
	defer jvArrTags.Unlock()
	k, ok := jvArrTags.kinds[desc]
	if !ok {
		if jvArrTags.kinds == nil {
			jvArrTags.kinds = map[string]int{}
		}
		k = jvArrKindBase + len(jvArrTags.names)
		jvArrTags.names = append(jvArrTags.names, desc)
		jvArrTags.kinds[desc] = k
	}
	return k
}

// jvpArrayTag is the array's class name as getName spells it: the type it was
// created with, else one INFERRED from its elements (see the header).
func jvpArrayTag(a *jsArray) string {
	if a.kind >= jvArrKindBase {
		jvArrTags.Lock()
		defer jvArrTags.Unlock()
		return jvArrTags.names[a.kind-jvArrKindBase]
	}
	tag := ""
	for _, e := range a.elems {
		t := jvpElemTag(e)
		if t == "" || (tag != "" && t != tag) {
			return "[Ljava.lang.Object;"
		}
		tag = t
	}
	if tag == "" {
		return "[Ljava.lang.Object;"
	}
	return tag
}

// jvAcType names an array the way System.arraycopy's messages do: "int[5]",
// "object array[5]". Twin of jvAcType in languages/lib/java-rt.metajs.
func jvAcType(a *jsArray) string {
	names := map[byte]string{'I': "int", 'J': "long", 'S': "short", 'B': "byte", 'F': "float", 'D': "double", 'C': "char", 'Z': "boolean"}
	t, ok := names[jvpArrayTag(a)[1]]
	if !ok {
		t = "object array"
	}
	return fmt.Sprintf("%s[%d]", t, len(a.elems))
}

func jvpElemTag(v interface{}) string {
	switch t := v.(type) {
	case jsGInt:
		// A sized integer carries its own width, so long[] / short[] / byte[] are
		// the one array kind whose element type is not a guess.
		switch t.w {
		case 64:
			return "[J"
		case 16:
			return "[S"
		}
		return "[B"
	case jsJFlo:
		if t.sty == floJavaF {
			return "[F"
		}
		return "[D"
	case jsChar:
		return "[C"
	case bool:
		return "[Z"
	case string:
		return "[Ljava.lang.String;"
	case float64:
		return "[I"
	}
	return ""
}

// ----------------------------------------------------------------------------
// Identity hash

// jvpIdent gives every object rendered with an `@` a stable per-run number. Real
// java prints System.identityHashCode, which is an address-derived value that
// changes between two runs of the same program - so it is not reproducible and no
// test can assert it. What the two halves must agree on is that the same object
// renders identically twice within a run and that the run as a whole is
// deterministic, which the counter provides. One compile runs one program, so a
// package-level counter is per-program.
var jvpIdent = map[interface{}]int{}
var jvpIdentN int

// The NUMBER behind that rendering, which is exactly what Object.hashCode
// answers in real java: `o.toString()` is getName() + "@" +
// Integer.toHexString(hashCode()), so the two must agree, and here they do by
// construction. jvpHash below is this value printed with %x.
func jvpIdentNum(v interface{}) float64 {
	n, ok := jvpIdent[v]
	if !ok {
		jvpIdentN++
		n = jvpIdentN
		jvpIdent[v] = n
	}
	return float64(0x1a2b3c00 + n)
}

func jvpHash(v interface{}) string {
	return fmt.Sprintf("%x", int(jvpIdentNum(v)))
}

// ----------------------------------------------------------------------------
// hashCode
//
// java.lang.Object#hashCode and the overrides of it that the JLS pins to an
// exact value. THE COMBINATION A RECORD USES IS NOT ONE OF THEM: JLS 8.10.3
// requires only that a record's hashCode be derived from the components'
// hashCodes and that equal records hash equally - the exact function "is
// unspecified". What IS specified exactly, and reproduced here bit for bit:
//
//	Boolean    true -> 1231, false -> 1237                       (JDK javadoc)
//	Integer    the value itself; Byte/Short/Character widen to it
//	Long       (int)(value ^ (value >>> 32))
//	Double     bits = doubleToLongBits(v); (int)(bits ^ (bits >>> 32))
//	Float      floatToIntBits(v)
//	String     s[0]*31^(n-1) + s[1]*31^(n-2) + ... , over UTF-16 units
//	null       0, via Objects.hashCode
//
// For the record COMBINATION we pin OpenJDK's, which is what
// java.lang.runtime.ObjectMethods generates: `h = 0; for each component h =
// h*31 + hash(component)`, at int width - `record R(int a, String b, double c)`
// with (1, "x", 2.5) answers 1074008649 in java 24.0.2 and here. It is pinned
// because the invariant that MUST hold (equal records hash equally) does not
// choose a value, and two engines have to agree on one.
//
// The twin is js_jhash in languages/lib/java-rt.metajs, arm for arm; that one
// cannot call math.Float64bits and takes the bit pattern apart with exact
// arithmetic instead.
// doubleToLongBits and floatToIntBits collapse every NaN to one canonical
// pattern; math.Float64bits keeps the payload (math.NaN() is ...0001).
func jvHashDouble(f float64) float64 {
	bits := math.Float64bits(f)
	if f != f {
		bits = 0x7ff8000000000000
	}
	return float64(int32(uint32(bits>>32) ^ uint32(bits)))
}

func jvHashFloat(f float64) float64 {
	bits := math.Float32bits(float32(f))
	if f != f {
		bits = 0x7fc00000
	}
	return float64(int32(bits))
}

func (rt *jsrt) jvHashString(s string) float64 {
	h := int32(0)
	n := rt.strLen(s)
	for i := 0; i < n; i++ {
		h = h*31 + int32(rt.strCodeAt(s, i))
	}
	return float64(h)
}

func (rt *jsrt) jvHash(v interface{}) float64 {
	switch x := v.(type) {
	case nil, jsUndefT, jsNullT:
		return 0
	case bool:
		if x {
			return 1231
		}
		return 1237
	case string:
		return rt.jvHashString(x)
	case float64:
		// A plain number is an INT in this value model (java never unboxes a
		// long - see chapter 7.5 of the manual), and Integer.hashCode is the
		// value itself.
		return float64(int32(jsToInt(x)))
	case jsChar:
		return float64(x.code)
	case jsGInt:
		// Long.hashCode. A byte/short/int box narrows to the same answer,
		// because its high half is the sign extension the value already has.
		u := uint64(x.v)
		return float64(int32(uint32(u>>32) ^ uint32(u)))
	case jsJFlo:
		if x.sty == floJavaF {
			return jvHashFloat(x.f)
		}
		return jvHashDouble(x.f)
	}
	if o, ok := v.(*jsObject); ok {
		if cls, has := o.props["__class"]; has {
			// A user-declared (or inherited) hashCode wins, exactly as an
			// override does. The __class/__super walk is js_jvaleq's, cap and
			// all.
			for guard := 0; guard < 64; guard++ {
				clsObj, isObj := cls.(*jsObject)
				if !isObj {
					break
				}
				if h, hit := clsObj.props["hashCode"]; hit && isCallable(h) {
					return rt.toNumber(rt.call(h, jsUndef, []interface{}{v}))
				}
				if comps, isRec := clsObj.props["__record"]; isRec {
					return rt.jvRecordHash(v, comps)
				}
				cls = clsObj.props["__super"]
			}
		}
	}
	return jvpIdentNum(v)
}

// h = 0; h = h*31 + Objects.hashCode(component), at int width - OpenJDK's.
func (rt *jsrt) jvRecordHash(v interface{}, comps interface{}) float64 {
	arr, ok := comps.(*jsArray)
	if !ok {
		return jvpIdentNum(v)
	}
	o, ok := v.(*jsObject)
	if !ok {
		return jvpIdentNum(v)
	}
	h := int32(0)
	for _, c := range arr.elems {
		name, _ := c.(string)
		h = h*31 + int32(rt.jvHash(o.props[name]))
	}
	return float64(h)
}
