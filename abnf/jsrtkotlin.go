package abnf

// jsrtkotlin.go -- Kotlin's VALUE RENDERING for the compiler grammar
// (languages/kotlin-to-llvm-ir.abnf).
//
// kotlin-to-llvm-ir.abnf used to bind Kotlin's println/print straight to the
// shared `println` host function, which hands the RAW runtime value to Go's %v.
// So the compiler printed
//
//	<nil>                      for a null reference
//	[1 2]                      for a list or an array
//	map[__class:map[__ctor:0x… __isclass:true __name:D …] a:1 b:q]
//	                           for a class instance - including a raw Go POINTER
//	                           address, so its stdout was not even reproducible
//	                           between two runs of the same program
//
// and an enum entry (whose descriptor points back at the entry) sent Go's %v into
// an infinite recursion: `println(E.A)` died with a stack overflow.
//
// Kotlin/JVM's Int/Long/Double/String ARE java.lang.*, and println IS
// System.out.println, so the answers below were settled with java 24 (there is no
// kotlinc on this machine - see the commit message for the probe and its output):
//
//	null / [1, 2] / D(a=1, b=q) / C@<hash> / A (an enum entry) / 1.5
//
// ktpRender is the Go twin of `kstr` in kotlin-interpreter.abnf and the two MUST
// stay in step, because ./test.sh --cross diffs the two halves.
//
// Registration is additive: an init() appends a registrar to rxExtraExterns, so
// nothing here runs unless a program calls one of the js_kt* externs, and only the
// Kotlin compiler grammar emits them. abnf/jsrt.go, its extern table and printArgs
// are untouched, so every other language keeps the rendering it has.
//
// Known simplifications, all DELIBERATE and all shared by the two halves:
//
//   - An ARRAY renders exactly like a List, `[1, 2]`. Real Kotlin prints
//     `[Ljava.lang.Integer;@1b6d3586` for an Array, because an Array is a JVM array
//     and inherits Object.toString. In this value model arrayOf and listOf both
//     build a plain host array and are indistinguishable at runtime, so no renderer
//     can tell them apart; the List answer is the one that is right more often.
//   - A class without a toString renders as `Name@1`: a FIXED synthetic hash. Real
//     Kotlin prints the identity hash, which differs between two runs of the real
//     toolchain and therefore cannot be asserted anyway; a constant is the only
//     value the two halves can agree on without an object-identity counter.
//     kotlin-interpreter.abnf's defaultToString has always answered `Name@1`.
//   - A RANGE prints as the list of its elements (`[1, 2, 3]`), where real Kotlin
//     prints `1..3`. The compiler has no first-class range value at all - `1..3`
//     is materialised into a list at the point of use - so `1..3` is not
//     reachable there, and the interpreter was made to agree with the compiler
//     rather than the other way round.
//   - There is no Map in either Kotlin half (mapOf is not a builtin in either
//     grammar: both answer "unknown name: mapOf"), so java.util.AbstractMap's
//     `{a=1, b=2}` rendering has nothing to render and is not implemented.

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// ----------------------------------------------------------------------------
// SIZED INTEGERS
//
// Kotlin's integral types have fixed widths and WRAP silently on overflow, and
// the width has to survive into the next operation - `var b: Byte = 127; b++` is
// -128, and nothing about the value 127 says so. The box that carries it is
// jsGInt from jsrtint.go, the same one Go, Swift, Java and C# use; only the RULES
// are Kotlin's and they live here, so no shared file has to learn a dialect.
//
// The invariant is JAVA's, because Kotlin/JVM's Int and Long ARE Java's:
//
//	a plain number  ==  an `Int`, i.e. a SIGNED 32 BIT value
//	a jsGInt        ==  Long / Short / Byte, or any UNSIGNED type
//	a jsJFlo        ==  a Double / Float                       (jsrtjvm.go)
//	a jsChar        ==  a Char
//
// Int is the type of every unsuffixed literal that fits in it and of every
// arithmetic result with no Long operand, so keeping IT unboxed is what makes the
// change affordable: an ordinary program allocates no box at all.
//
// The four ways Kotlin is not Java, all of them making this SMALLER rather than
// larger:
//
//  1. UByte/UShort/UInt/ULong exist, so the box's signedness bit is used. But
//     Kotlin has NO implicit numeric conversions, so both operands of an operator
//     already have the same type and there is no promotion LADDER to model (C#'s
//     `uint + int -> long` has no Kotlin counterpart: it is a compile error).
//  2. The conversions are METHODS - toInt(), toLong(), toUInt(), toDouble() - not
//     cast syntax.
//  3. shl / shr / ushr / and / or / xor / inv are infix FUNCTIONS, so they parse
//     at InfixExpr rather than at an operator level.
//  4. Int.MAX_VALUE and friends are companion members, and the literal suffixes
//     are L, u, U, uL, UL.
//
// languages/kotlin-interpreter.abnf implements exactly these rules on top of the
// sz* layer in lib/interp-core.js, and ./test.sh --cross diffs the two halves, so
// the two MUST stay in step.

// ktNorm applies the invariant: a signed 32 bit result is a PLAIN NUMBER, every
// other width or signedness is a box. It is deliberately not giNorm, which unboxes
// at 64 bits because a plain number means int64 for Go.
func ktNorm(x int64, w uint8, u bool) interface{} {
	x = giTrunc(x, w, u)
	if w == 32 && !u {
		return float64(int32(x))
	}
	return jsGInt{v: x, w: w, u: u}
}

// ktWU is the result type of a binary operator. Kotlin requires both operands to
// have the same type, so in a program kotlinc accepts at most one side carries a
// surprise; the widening rules that DO exist are Byte+Byte -> Int, Short+Short ->
// Int, Int+Long -> Long and UInt+ULong -> ULong, which is exactly "64 bits if
// either side is 64, else 32" plus "unsigned if either side is unsigned".
func ktWU(l, r interface{}) (uint8, bool) {
	w, u := uint8(32), false
	if b, ok := l.(jsGInt); ok {
		if b.w == 64 {
			w = 64
		}
		u = u || b.u
	}
	if b, ok := r.(jsGInt); ok {
		if b.w == 64 {
			w = 64
		}
		u = u || b.u
	}
	return w, u
}

// ktIsIntegral says whether a value is an operand of INTEGER arithmetic - which a
// Char is, Kotlin's Char having code arithmetic, and a Double is not.
func ktIsIntegral(v interface{}) bool {
	switch v.(type) {
	case jsGInt, float64, jsChar:
		return true
	}
	return false
}

// ktArithExcCls is the ArithmeticException descriptor a divide by zero throws.
// The Kotlin compiler grammar has no builtin exception classes to hand over (its
// `catch` takes the first clause and reads `message`), so the descriptor is built
// here, once, and its __super chain lets js_ktis answer `e is ArithmeticException`.
// One compile runs one program, so a package-level holder is per-program.
var ktArithExcCls *jsObject

// ktExcHier is the builtin throwable hierarchy, built once per program. It is what
// `Exception("m")` / `IllegalStateException("m")` construct: the Kotlin compiler
// grammar has no user class for them, and before this the call aborted with
// "unknown name: Exception" while kotlin-interpreter.abnf ran the same program.
// The chain is Kotlin's own (IllegalStateException -> RuntimeException -> Exception
// -> Throwable), so js_ktis answers `e is Exception` for one - see the report note
// about installThrowables in the other half, which parents all of them directly to
// Throwable and therefore answers false.
var ktExcHier map[string]*jsObject

func ktExcHierarchy() map[string]*jsObject {
	if ktExcHier != nil {
		return ktExcHier
	}
	mk := func(name string, super *jsObject) *jsObject {
		o := newJSObject()
		o.set("__isclass", true)
		o.set("__name", name)
		if super != nil {
			o.set("__super", super)
		}
		return o
	}
	h := map[string]*jsObject{}
	h["Throwable"] = mk("Throwable", nil)
	h["Exception"] = mk("Exception", h["Throwable"])
	h["Error"] = mk("Error", h["Throwable"])
	h["RuntimeException"] = mk("RuntimeException", h["Exception"])
	for _, n := range []string{"IllegalStateException", "IllegalArgumentException",
		"IndexOutOfBoundsException", "NullPointerException",
		"UnsupportedOperationException", "ArithmeticException", "NoSuchElementException",
		"ConcurrentModificationException", "ClassCastException"} {
		h[n] = mk(n, h["RuntimeException"])
	}
	// NumberFormatException is under IllegalArgumentException, not directly under
	// RuntimeException - checked with `instanceof` under java (JDK 24).
	h["NumberFormatException"] = mk("NumberFormatException", h["IllegalArgumentException"])
	h["IllegalAccessException"] = mk("IllegalAccessException", h["Exception"])
	h["UninitializedPropertyAccessException"] = mk("UninitializedPropertyAccessException", h["RuntimeException"])
	h["StackOverflowError"] = mk("StackOverflowError", h["Error"])
	h["OutOfMemoryError"] = mk("OutOfMemoryError", h["Error"])
	ktExcHier = h
	return h
}

// ktLateinitCheck is the read of a `lateinit var`. A lateinit property cannot hold
// null in Kotlin, so "the field is still null" IS "not initialized yet": reading one
// before it is assigned throws UninitializedPropertyAccessException where both halves
// used to answer null. __lateinit is the name table the class emitter installs on the
// descriptor (emitLateinit in kotlin-to-llvm-ir.abnf); a class that declares no
// lateinit property has none, so the walk stops immediately. The twin is
// kLateinitCheck in kotlin-interpreter.abnf.
func (rt *jsrt) ktLateinitCheck(o *jsObject, name string, v interface{}) interface{} {
	if !isUndefOrNull(v) {
		return v
	}
	cls, _ := o.props["__class"].(*jsObject)
	for guard := 0; cls != nil && guard < 64; guard++ {
		if li, ok := cls.props["__lateinit"].(*jsObject); ok {
			if t, has := li.props[name]; has && t == true {
				rt.ktRaise("UninitializedPropertyAccessException",
					"lateinit property "+name+" has not been initialized")
			}
		}
		cls, _ = cls.props["__super"].(*jsObject)
	}
	return v
}

// ktIsBuiltinExc reports a name the hierarchy above provides.
func ktIsBuiltinExc(name string) bool {
	_, ok := ktExcHierarchy()[name]
	return ok
}

func ktExcClass() *jsObject {
	if ktArithExcCls != nil {
		return ktArithExcCls
	}
	ktArithExcCls = ktExcHierarchy()["ArithmeticException"]
	return ktArithExcCls
}

func (rt *jsrt) ktThrowArith(msg string) {
	o := newJSObject()
	o.set("__class", ktExcClass())
	// `message` is the property Kotlin's Throwable exposes and the field both
	// halves read; the interpreter's installThrowables stores it under the same
	// name.
	o.set("message", msg)
	panic(&jsThrown{value: o})
}

// ktArith is one binary arithmetic or bitwise operator on two INTEGRAL operands,
// evaluated at the result type's width and wrapped to it (Kotlin overflow is
// silent - unlike Swift, which traps).
func (rt *jsrt) ktArith(op string, l, r interface{}) interface{} {
	w, u := ktWU(l, r)
	a := jsGInt{v: giTrunc(giVal(rt, l), w, u), w: w, u: u}
	b := jsGInt{v: giTrunc(giVal(rt, r), w, u), w: w, u: u}
	if op == "/" || op == "%" {
		if b.v == 0 {
			rt.ktThrowArith("/ by zero")
		}
	}
	// giArith already divides truncating toward zero, reads an unsigned width
	// unsigned, and answers Int.MIN_VALUE for MIN / -1 (the hardware wrap, which
	// is what the JVM's idiv does and therefore what Kotlin/JVM does).
	return ktNorm(giVal(rt, rt.giArith(op, a, b)), w, u)
}

// ktShift is shl / shr / ushr. The result type is the LEFT operand's alone and the
// COUNT is MASKED - & 31 for a 32 bit left operand, & 63 for a 64 bit one - which
// is why `1 shl 32` is 1 rather than 0. (Kotlin's shift functions are declared on
// Int, Long, UInt and ULong only.)
func (rt *jsrt) ktShift(op string, l, r interface{}) interface{} {
	w, u := uint8(32), false
	if b, ok := l.(jsGInt); ok {
		w, u = b.w, b.u
	}
	mask := int64(31)
	if w == 64 {
		mask = 63
	}
	s := uint(giVal(rt, r) & mask)
	a := giTrunc(giVal(rt, l), w, u)
	switch op {
	case "shl":
		return ktNorm(a<<s, w, u)
	case "shr":
		if u {
			return ktNorm(int64(giU(a, w)>>s), w, u)
		}
		return ktNorm(a>>s, w, u)
	}
	// ushr is the LOGICAL shift: read the operand as unsigned at its width, shift,
	// read the result back at the operand's own signedness.
	return ktNorm(int64(giU(a, w)>>s), w, u)
}

// ktCmp is an ordered comparison at the promoted width. giCmp on its own is not
// usable: it compares at the WIDEST operand's width, so `val b: Byte = 5; b < 1000`
// would truncate 1000 into a byte.
func (rt *jsrt) ktCmp(op string, l, r interface{}) bool {
	w, u := ktWU(l, r)
	a := giTrunc(giVal(rt, l), w, u)
	b := giTrunc(giVal(rt, r), w, u)
	c := 0
	if u {
		ua, ub := giU(a, w), giU(b, w)
		if ua < ub {
			c = -1
		} else if ua > ub {
			c = 1
		}
	} else if a < b {
		c = -1
	} else if a > b {
		c = 1
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

// ktFloNarrow is a FLOATING POINT to integral conversion. Kotlin/JVM compiles
// Double.toInt() to the JVM's d2i and Double.toLong() to d2l, both of which
// SATURATE at the target's range and answer 0 for NaN; a target narrower than 32
// bits goes through the 32 bit one first and then truncates. The unsigned
// conversions saturate at [0, 2^w-1] instead (kotlin.Double.toUInt / toULong).
func ktFloNarrow(f float64, w uint8, u bool) int64 {
	if math.IsNaN(f) {
		return 0
	}
	if u {
		if w == 64 {
			if f <= 0 {
				return 0
			}
			if f >= 18446744073709551616 {
				return -1 // the bit pattern of ULong.MAX_VALUE
			}
			return int64(uint64(math.Trunc(f)))
		}
		if f <= 0 {
			return 0
		}
		if f >= 4294967295 {
			return 4294967295
		}
		return int64(math.Trunc(f))
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

// ktConv is toByte() / toShort() / toInt() / toLong() / toUByte() / ... An integral
// source is a pure bit reinterpretation, so (-1).toUInt() is 4294967295u; a
// floating point one saturates first (see ktFloNarrow).
func (rt *jsrt) ktConv(v interface{}, w uint8, u bool) interface{} {
	if f, isFlo := v.(jsJFlo); isFlo {
		return ktNorm(ktFloNarrow(f.f, w, u), w, u)
	}
	return ktNorm(giVal(rt, v), w, u)
}

// ktTypeObj builds one of Kotlin's number types as a value, for its companion
// constants. Int.MAX_VALUE was `unknown name: Int` in both halves before, and
// Long.MAX_VALUE cannot be written as a plain number at all.
func ktTypeObj(name string) *jsObject {
	o := newJSObject()
	set := func(mx, mn interface{}, bits float64) {
		o.set("MAX_VALUE", mx)
		o.set("MIN_VALUE", mn)
		o.set("SIZE_BITS", bits)
		o.set("SIZE_BYTES", bits/8)
	}
	switch name {
	case "Int":
		set(float64(math.MaxInt32), float64(math.MinInt32), 32)
	case "Long":
		set(jsGInt{v: math.MaxInt64, w: 64}, jsGInt{v: math.MinInt64, w: 64}, 64)
	case "Byte":
		set(jsGInt{v: 127, w: 8}, jsGInt{v: -128, w: 8}, 8)
	case "Short":
		set(jsGInt{v: 32767, w: 16}, jsGInt{v: -32768, w: 16}, 16)
	case "UInt":
		set(jsGInt{v: 4294967295, w: 32, u: true}, jsGInt{v: 0, w: 32, u: true}, 32)
	case "ULong":
		set(jsGInt{v: -1, w: 64, u: true}, jsGInt{v: 0, w: 64, u: true}, 64)
	case "UByte":
		set(jsGInt{v: 255, w: 8, u: true}, jsGInt{v: 0, w: 8, u: true}, 8)
	case "UShort":
		set(jsGInt{v: 65535, w: 16, u: true}, jsGInt{v: 0, w: 16, u: true}, 16)
	case "Char":
		o.set("MAX_VALUE", jsChar{code: 65535})
		o.set("MIN_VALUE", jsChar{code: 0})
		o.set("SIZE_BITS", float64(16))
		o.set("SIZE_BYTES", float64(2))
	case "Double", "Float":
		// The float companions, boxed so that Double.NaN is a Double rather than an
		// Int: kotlin.Double declares MIN_VALUE as the smallest POSITIVE value, the
		// way java.lang.Double does.
		mx, mn := math.MaxFloat64, 4.9e-324
		bits := float64(64)
		// Float's five constants are FLOAT boxes (style floJavaF), so
		// Float.MAX_VALUE prints "3.4028235E38" and not the double's 17 digits.
		sty := uint8(floJava)
		if name == "Float" {
			mx, mn, bits, sty = math.MaxFloat32, math.SmallestNonzeroFloat32, 32, floJavaF
		}
		o.set("MAX_VALUE", jsJFlo{f: mx, sty: sty})
		o.set("MIN_VALUE", jsJFlo{f: mn, sty: sty})
		o.set("NaN", jsJFlo{f: math.NaN(), sty: sty})
		o.set("POSITIVE_INFINITY", jsJFlo{f: math.Inf(1), sty: sty})
		o.set("NEGATIVE_INFINITY", jsJFlo{f: math.Inf(-1), sty: sty})
		o.set("SIZE_BITS", bits)
		o.set("SIZE_BYTES", bits/8)
	}
	return o
}

// ktLitSfx is the suffix code the two grammars pass to js_ktlit: 0 none, 1 L,
// 2 u/U, 3 uL/UL. An unsuffixed literal is an Int when it fits in one and a Long
// otherwise (so 0xFFFFFFFF is a Long 4294967295 here, where Java makes it int -1);
// a `u` literal is a UInt when it fits and a ULong otherwise.
func ktLitValue(acc uint64, sfx int) interface{} {
	switch sfx {
	case 1:
		return jsGInt{v: int64(acc), w: 64}
	case 2:
		if acc <= 4294967295 {
			return jsGInt{v: int64(acc), w: 32, u: true}
		}
		return jsGInt{v: int64(acc), w: 64, u: true}
	case 3:
		return jsGInt{v: int64(acc), w: 64, u: true}
	}
	if acc <= 2147483647 {
		return float64(acc)
	}
	return jsGInt{v: int64(acc), w: 64}
}

// ktDigits accumulates a literal's DIGITS exactly. The text travels rather than a
// number because a double rounds 9223372036854775807 on the way in - and rounds it
// DIFFERENTLY under goja and the frozen engine, which the matrix's byte-identity
// requirement would then fail on.
func ktDigits(s string, radix uint64) uint64 {
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
	return acc
}

// ktNumMethod is the member-function surface of Kotlin's number types: the
// conversions, inv(), and the handful of names a program calls on an Int or a
// Double. It is the Go twin of the numeric branches of `mcall` in
// kotlin-interpreter.abnf. It answers (value, true) when it handled the name.
func (rt *jsrt) ktNumMethod(target interface{}, name string, args []interface{}) (interface{}, bool) {
	if !giIsNumeric(target) {
		return nil, false
	}
	switch name {
	// rangeTo / until / downTo in their FUNCTION spelling. `..`, `until` and
	// `downTo` are the infix operators the emitter lowers to js_ktrange, but
	// kotlin.ranges declares each as an ordinary member/extension too, and
	// `1.rangeTo(5)` aborted with "unknown Int method: rangeTo" in all three
	// engines. ktRangeMake is shared with the Char receiver.
	case "rangeTo", "until", "downTo":
		if len(args) == 1 {
			return ktRangeMake(target, args[0], name), true
		}
	case "toByte":
		return rt.ktConv(target, 8, false), true
	case "toShort":
		return rt.ktConv(target, 16, false), true
	case "toInt":
		return rt.ktConv(target, 32, false), true
	case "toLong":
		return rt.ktConv(target, 64, false), true
	case "toUByte":
		return rt.ktConv(target, 8, true), true
	case "toUShort":
		return rt.ktConv(target, 16, true), true
	case "toUInt":
		return rt.ktConv(target, 32, true), true
	case "toULong":
		return rt.ktConv(target, 64, true), true
	case "toDouble":
		return jsJFlo{f: giFloat(rt, target)}, true
	case "toFloat":
		// toFloat() NARROWS to a binary32: 0.1.toFloat() is 0.1f, which prints
		// "0.1" and is a different value from the double 0.1.
		return jsJFlo{f: jvmFround(giFloat(rt, target)), sty: floJavaF}, true
	case "toChar":
		return jsChar{code: int32(giVal(rt, target) & 65535)}, true
	case "inv":
		if b, ok := target.(jsGInt); ok {
			return ktNorm(^b.v, b.w, b.u), true
		}
		return float64(^int32(giVal(rt, target))), true
	case "compareTo":
		o := argAt(args, 0)
		if ktIsIntegral(target) && ktIsIntegral(o) {
			if rt.ktCmp("<", target, o) {
				return float64(-1), true
			}
			if rt.ktCmp(">", target, o) {
				return float64(1), true
			}
			return float64(0), true
		}
		a, b := giFloat(rt, target), giFloat(rt, o)
		if a < b {
			return float64(-1), true
		}
		if a > b {
			return float64(1), true
		}
		return float64(0), true
	case "equals":
		return rt.giEq(target, argAt(args, 0)), true
	case "coerceAtLeast":
		o := argAt(args, 0)
		if rt.ktNumLess(target, o) {
			return o, true
		}
		return target, true
	case "coerceAtMost":
		o := argAt(args, 0)
		if rt.ktNumLess(o, target) {
			return o, true
		}
		return target, true
	// coerceIn(min, max): the third member of the coerce family, missing in both
	// halves while coerceAtLeast and coerceAtMost were present.
	case "coerceIn":
		lo, hi := argAt(args, 0), argAt(args, 1)
		if !isUndefOrNull(lo) && rt.ktNumLess(target, lo) {
			return lo, true
		}
		if !isUndefOrNull(hi) && rt.ktNumLess(hi, target) {
			return hi, true
		}
		return target, true
	case "isNaN":
		f, ok := target.(jsJFlo)
		return ok && math.IsNaN(f.f), true
	case "hashCode":
		if f, ok := target.(jsJFlo); ok {
			return float64(ktFloHash(f)), true
		}
		if b, ok := target.(jsGInt); ok {
			return float64(int32(b.v ^ (b.v >> 32))), true
		}
		return float64(int32(giVal(rt, target))), true
	}
	return nil, false
}

// ktDblHash is java.lang.Double.hashCode, which is what Kotlin/JVM's
// Double.hashCode() IS: doubleToLongBits(d) xor (bits ushr 32), read as an Int.
// Before this a Double hashed as its TRUNCATED value in BOTH halves, so
// 1.5.hashCode() answered 1 where java answers 1073217536. The twin is kDblHash in
// kotlin-interpreter.abnf, which has to derive the bits arithmetically because
// neither engine has typed arrays; here math.Float64bits is the whole job.
// A FLOAT shares the jsJFlo box with a Double and is told apart by the style
// byte, so it takes the OTHER hash - see ktFltHash and ktFloHash below.
func ktDblHash(d float64) int32 {
	// doubleToLongBits COLLAPSES every NaN to 0x7ff8000000000000, which
	// math.Float64bits does not (Go's own math.NaN() is ...0001).
	if math.IsNaN(d) {
		return 2146959360
	}
	bits := math.Float64bits(d)
	return int32(bits ^ (bits >> 32))
}

// ktFltHash is java.lang.Float.hashCode, which is what Kotlin/JVM's
// Float.hashCode() IS: floatToIntBits(f), read as a SIGNED Int - so it is the raw
// binary32 bit pattern and not a fold of two halves, and the hash of a negative
// float is a negative number (-1.5f is -1077936128). It is a different function
// from ktDblHash at every value but a handful of coincidences: 1.5f hashes to
// 1069547520 where the double 1.5 hashes to 1073217536, and -0.0f hashes to
// -2147483648 where 0.0f hashes to 0, so the two zeros do NOT hash alike.
// Measured against java 24.0.2 over 28 values, docs/todo.md 2.4.
func ktFltHash(d float64) int32 {
	// floatToIntBits COLLAPSES every NaN to 0x7fc00000, exactly as
	// doubleToLongBits collapses a double's.
	if math.IsNaN(d) {
		return 2143289344
	}
	return int32(math.Float32bits(float32(d)))
}

// ktFloHash is the hash of a tag-14 box, which of the two depending on the style
// byte. Every reader of a float's hash goes through it, so the width question is
// asked in ONE place: the receiver method, ktElemHash, and through the latter
// every collection fold and generated data-class hash.
func ktFloHash(f jsJFlo) int32 {
	if f.sty == floJavaF {
		return ktFltHash(f.f)
	}
	return ktDblHash(f.f)
}

// ktBoxTypeName is the Kotlin type a box stands for, which is what `is` and the
// array-element rendering ask it. The twin is kBoxTypeName in
// kotlin-interpreter.abnf.
func ktBoxTypeName(b jsGInt) string {
	if b.u {
		switch b.w {
		case 8:
			return "UByte"
		case 16:
			return "UShort"
		case 32:
			return "UInt"
		}
		return "ULong"
	}
	switch b.w {
	case 8:
		return "Byte"
	case 16:
		return "Short"
	case 32:
		return "Int"
	}
	return "Long"
}

// ktNumLess is the "<" the coerce pair needs: integral operands compare at the
// promoted width, anything with a Double in it as floating point.
func (rt *jsrt) ktNumLess(l, r interface{}) bool {
	if ktIsIntegral(l) && ktIsIntegral(r) {
		return rt.ktCmp("<", l, r)
	}
	return giFloat(rt, l) < giFloat(rt, r)
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
		bitsOf := func(h uint64) uint8 { return uint8(jsToInt(rt.toNumber(u(h)))) }

		// ----- the sized-integer externs (see the SIZED INTEGERS block above) -----

		// One binary integral operator: (op, left, right). A Double operand keeps the
		// FLOAT path of jsrtjvm.go, which is where the two boxes meet - so mixed
		// Int/Double arithmetic still answers a Double.
		m["js_ktarith"] = func(a []uint64) uint64 {
			op, l, r := opOf(a[0]), u(a[1]), u(a[2])
			// `-` on a COLLECTION is kotlin.collections.minus, the twin of the `+`
			// branch in js_ktadd: a list, set or map loses the given element or every
			// element of the given collection. Before this it fell through to the
			// numeric tail, so `listOf(1, 2, 3) - 1` answered 0 in BOTH halves.
			if op == "-" {
				if rg, isRg := ktRangeOf(l); isRg {
					if v, ok := rt.ktSeqMethod(&jsArray{elems: ktRangeList(rg)}, "minus", []interface{}{r}); ok {
						return w(v)
					}
				}
				if la, isArr := l.(*jsArray); isArr {
					if v, ok := rt.ktSeqMethod(la, "minus", []interface{}{r}); ok {
						return w(v)
					}
				}
				if lo, isObj := l.(*jsObject); isObj {
					if ktIsSet(lo) {
						if v, ok := rt.ktSetMethod(lo, "minus", []interface{}{r}); ok {
							return w(v)
						}
					} else if _, _, isDict := dictParts(lo); isDict {
						if v, ok := rt.ktMapMethod(lo, "minus", []interface{}{r}); ok {
							return w(v)
						}
					}
				}
			}
			if jvmIsFlo(l) || jvmIsFlo(r) {
				return w(rt.jvmArith(op[0], l, r))
			}
			if !ktIsIntegral(l) || !ktIsIntegral(r) {
				// `and`/`or`/`xor` on two Booleans are Kotlin's non-short-circuit
				// boolean operators; anything else is not a program kotlinc accepts.
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
			return w(rt.ktArith(op, l, r))
		}
		m["js_ktshift"] = func(a []uint64) uint64 {
			return w(rt.ktShift(opOf(a[0]), u(a[1]), u(a[2])))
		}
		// An ordered comparison: two integers compare at the promoted width, so a
		// Long above 2^53 compares exactly rather than through a rounded double.
		// Everything else (a Double, a String pair) keeps the shared comparison.
		m["js_ktcmp"] = func(a []uint64) uint64 {
			op, l, r := opOf(a[0]), u(a[1]), u(a[2])
			if ktIsIntegral(l) && ktIsIntegral(r) && !jvmIsFlo(l) && !jvmIsFlo(r) {
				return boolH(rt.ktCmp(op, l, r))
			}
			// Kotlin's COMPARISON convention: `a < b` on a class that declares
			// `operator fun compareTo` is `a.compareTo(b) < 0`. Without this the two
			// halves disagreed on a user Comparable - `V(1) < V(2)` was true in
			// kotlin-interpreter.abnf (whose kNumOp2 routes all four operators through
			// compareTo) and FALSE here, because jsCompare fell back to comparing the
			// two objects as values. The probe is on the RECEIVER's descriptor chain,
			// so a program that declares no compareTo emits and runs exactly as before.
			if lo, isObj := l.(*jsObject); isObj && ktMemberCall != nil && ktClassChainHas(lo, "compareTo") {
				c := int(rt.toNumber(ktMemberCall(l, "compareTo", []interface{}{r})))
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
			// A Float operand promotes the other side to Float too, so
			// `16777217 < 16777216f` is FALSE - the Int converts to 1.6777216E7.
			// (Kotlin's Int.compareTo(Float) compiles to a JVM float comparison,
			// i.e. exactly JLS 5.6.2.)
			l, r = jvmFloCmpPair(rt, l, r)
			c := rt.jsCompare(l, r)
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
		// Kotlin's `+`: string concatenation is decided at the emit site, so this is
		// the numeric case plus the fallback the runtime had.
		m["js_ktadd"] = func(a []uint64) uint64 {
			l, r := u(a[0]), u(a[1])
			if jvmIsFlo(l) || jvmIsFlo(r) {
				// jvmArith, not a hand-rolled add: the float WIDTH needs both
				// operands converted first and the sum rounded after, and
				// jvmStyleOf gets `Float + Double` backwards (it is a Double).
				return w(rt.jvmArith('+', l, r))
			}
			if ktIsIntegral(l) && ktIsIntegral(r) {
				return w(rt.ktArith("+", l, r))
			}
			// `+` on a COLLECTION is kotlin.collections.plus: a list gains the elements
			// of another collection, or the single element. Before this it fell through
			// to the runtime's JavaScript `+` and `listOf(1, 2) + listOf(3)` answered
			// the string "1,23" here and 0 in the interpreter half.
			if rg, isRg := ktRangeOf(l); isRg {
				if v, ok := rt.ktSeqMethod(&jsArray{elems: ktRangeList(rg)}, "plus", []interface{}{r}); ok {
					return w(v)
				}
			}
			if la, isArr := l.(*jsArray); isArr {
				if v, ok := rt.ktSeqMethod(la, "plus", []interface{}{r}); ok {
					return w(v)
				}
			}
			if lo, isObj := l.(*jsObject); isObj {
				// A SET gains elements as a Set, not as a List
				// (kotlin.collections.plus is declared on Set separately from
				// Iterable and keeps the receiver's shape).
				if ktIsSet(lo) {
					if v, ok := rt.ktSetMethod(lo, "plus", []interface{}{r}); ok {
						return w(v)
					}
				}
				if _, _, isDict := dictParts(lo); isDict {
					if v, ok := rt.ktMapMethod(lo, "plus", []interface{}{r}); ok {
						return w(v)
					}
				}
			}
			return w(rt.jsAdd(l, r))
		}
		// == / != where a box may be involved. Two numbers compare by VALUE across
		// the box (giEq); every other pair keeps the equality the runtime had, so a
		// data class still compares through its own equals.
		baseEq := m["js_seq"]
		baseNe := m["js_sne"]
		m["js_kteq"] = func(a []uint64) uint64 {
			l, r := u(a[0]), u(a[1])
			if giIsInt(l) || giIsInt(r) {
				return boolH(rt.giEq(l, r))
			}
			// Two boxes: a Float is not equal to a Double even at the same value
			// (java.lang.Float#equals demands a Float). See ktFloClsEq.
			if v, decided := ktFloClsEq(l, r); decided {
				return boolH(v)
			}
			// Two callable references / class literals. See ktRefEq; without it the
			// comparison fell through to identity and `b::v == b::v` was false.
			if v, decided := ktRefEq(l, r); decided {
				return boolH(v)
			}
			// A list, a map or a Pair/Triple compares STRUCTURALLY (ktStructEq): none
			// of them carries a class descriptor, so the fallback below is identity
			// and `listOf(1, 2) == listOf(1, 2)` answered false.
			if v, decided := rt.ktStructEq(l, r); decided {
				return boolH(v)
			}
			// An instance whose class declares equals (a data or value class does)
			// answers for itself. The emit site already dispatches the convention when
			// the operand's class is KNOWN to declare it; this is the same rule reached
			// from a NESTED position - a data-class property holding another data
			// instance, which emitEqualsFunc now compares with js_kteq.
			if lo, isObj := l.(*jsObject); isObj {
				if mth, found := ktpFindMember(lo, "equals"); found {
					return boolH(rt.truthy(rt.call(mth, jsUndef, []interface{}{lo, r})))
				}
			}
			return baseEq(a)
		}
		m["js_ktne"] = func(a []uint64) uint64 {
			l, r := u(a[0]), u(a[1])
			if giIsInt(l) || giIsInt(r) {
				return boolH(!rt.giEq(l, r))
			}
			// Two boxes: a Float is not equal to a Double even at the same value.
			if v, decided := ktFloClsEq(l, r); decided {
				return boolH(!v)
			}
			// Two callable references / class literals. See ktRefEq; without it the
			// comparison fell through to identity and `b::v == b::v` was false.
			if v, decided := ktRefEq(l, r); decided {
				return boolH(!v)
			}
			if v, decided := rt.ktStructEq(l, r); decided {
				return boolH(!v)
			}
			if lo, isObj := l.(*jsObject); isObj {
				if mth, found := ktpFindMember(lo, "equals"); found {
					return boolH(!rt.truthy(rt.call(mth, jsUndef, []interface{}{lo, r})))
				}
			}
			return baseNe(a)
		}
		// A conversion method - toInt(), toLong(), toUInt(): (value, bits, unsigned).
		m["js_ktconv"] = func(a []uint64) uint64 {
			return w(rt.ktConv(u(a[0]), bitsOf(a[1]), rt.truthy(u(a[2]))))
		}
		// A DECLARED integral type applied to its initializer. Kotlin has no implicit
		// conversions, so this only ever sees the one case the language does allow -
		// an integer LITERAL typed by its expectation, `val x: Long = 1` - and a
		// non-integral value passes through untouched.
		m["js_ktadopt"] = func(a []uint64) uint64 {
			v := u(a[0])
			if !ktIsIntegral(v) {
				return a[0]
			}
			return w(rt.ktConv(v, bitsOf(a[1]), rt.truthy(u(a[2]))))
		}
		// A declared DOUBLE / FLOAT type applied to its initializer. `val d: Double = 1`
		// is the float half of the same Kotlin rule js_ktadopt implements: an integer
		// literal takes the expected type, and without this the value stayed an Int and
		// printed "1" where Kotlin prints "1.0".
		m["js_ktadoptf"] = func(a []uint64) uint64 {
			v := u(a[0])
			if !ktIsIntegral(v) {
				return a[0]
			}
			return w(jsJFlo{f: giFloat(rt, v)})
		}
		// The FLOAT half of js_ktadoptf: `val f: Float = 1` and every other
		// declared-Float binding site. The twin is js_ktadoptf32 in
		// languages/lib/kotlin-rt.metajs.
		m["js_ktadoptf32"] = func(a []uint64) uint64 {
			v := u(a[0])
			if !ktIsIntegral(v) {
				return a[0]
			}
			return w(jsJFlo{f: jvmFround(giFloat(rt, v)), sty: floJavaF})
		}
		// js_ktflo32 is the ONE new extern the width needs: a `1.0f` literal and
		// a `.toFloat()` both lower to it. Its layer-2 twin is js_ktflo32 in
		// languages/lib/kotlin-rt.metajs.
		m["js_ktflo32"] = func(a []uint64) uint64 {
			return w(jsJFlo{f: jvmFround(rt.toNumber(u(a[0]))), sty: floJavaF})
		}
		// An integer literal, passed as DIGITS: (text, radix, suffix code).
		m["js_ktlit"] = func(a []uint64) uint64 {
			s := rt.toString(u(a[0]))
			radix := uint64(jsToInt(rt.toNumber(u(a[1]))))
			return w(ktLitValue(ktDigits(s, radix), jsToInt(rt.toNumber(u(a[2])))))
		}
		// -x. A Double negates as a Double; a Long stays a Long; a Byte, a Short and
		// a Char promote to Int, because that is what Kotlin's unaryMinus returns.
		m["js_ktneg"] = func(a []uint64) uint64 {
			v := u(a[0])
			if f, ok := v.(jsJFlo); ok {
				return w(jsJFlo{f: -f.f, sty: f.sty})
			}
			if b, ok := v.(jsGInt); ok {
				if b.w == 64 || b.u {
					return w(ktNorm(-b.v, b.w, b.u))
				}
				return rt.wrapNum(float64(-int32(b.v)))
			}
			return rt.wrapNum(float64(-int32(giVal(rt, v))))
		}
		// inv(), at the operand's OWN width - Kotlin declares it on Int, Long and the
		// four unsigned types, each answering its own type.
		m["js_ktinv"] = func(a []uint64) uint64 {
			v := u(a[0])
			if b, ok := v.(jsGInt); ok {
				return w(ktNorm(^b.v, b.w, b.u))
			}
			return rt.wrapNum(float64(^int32(giVal(rt, v))))
		}
		// ++ / -- keep the type of what they step, which is what makes
		// `var b: Byte = 127; b++` answer -128.
		m["js_ktstep"] = func(a []uint64) uint64 {
			v, d := u(a[0]), giVal(rt, u(a[1]))
			switch t := v.(type) {
			case jsChar:
				return w(jsChar{code: int32((int64(t.code) + d) & 65535)})
			case jsJFlo:
				// ++/-- keeps the operand's own type, WIDTH included: a Float
				// steps as a Float, so `var f = 16777216f; f++` stays 1.6777216E7.
				if t.sty == floJavaF {
					return w(jsJFlo{f: jvmFround(t.f + float64(d)), sty: floJavaF})
				}
				return w(jsJFlo{f: t.f + float64(d), sty: t.sty})
			case jsGInt:
				return w(ktNorm(t.v+d, t.w, t.u))
			}
			return rt.wrapNum(float64(int32(giVal(rt, v)) + int32(d)))
		}
		// The plain-number reading a raw consumer needs (an index, a length, a
		// range bound): a box is not a float64 and every such consumer wants one.
		m["js_ktnum"] = func(a []uint64) uint64 {
			if _, ok := u(a[0]).(jsGInt); ok {
				return rt.wrapNum(giFloat(rt, u(a[0])))
			}
			return a[0]
		}
		// Int / Long / UInt / ... as VALUES, for their companion constants.
		m["js_kttype"] = func(a []uint64) uint64 { return w(ktTypeObj(rt.toString(u(a[0])))) }
		// `x is T`. The shared js_is_type answers by the value model's own rules,
		// where every integral number is an Int AND a Long; a box carries its real
		// type, so it is answered here and everything else falls through unchanged.
		baseIs := m["js_is_type"]
		// js_ktnewexc(name, message) constructs one of the builtin throwables. The
		// Kotlin grammar emits it for a call of a throwable name the program does not
		// itself declare; `message` is the field both halves read.
		m["js_ktnewexc"] = func(a []uint64) uint64 {
			name := rt.toString(u(a[0]))
			cls, ok := ktExcHierarchy()[name]
			if !ok {
				rt.fail("unknown builtin exception %s", name)
			}
			o := newJSObject()
			o.set("__class", cls)
			o.set("message", u(a[1]))
			return w(o)
		}
		// js_ktthisat(this, "Outer") is Kotlin's labelled `this@Outer`: walk the
		// __outer chain for an instance whose class is named `Outer`, and fall back to
		// the receiver itself when there is none - which is what makeThisAt does in
		// kotlin-interpreter.abnf (thisAtIn, then findThis). Reading `__outer` blindly
		// aborted with "member 'tag' of undefined" whenever an inner class was
		// constructed from inside its outer's own method, where nothing sets it.
		m["js_ktthisat"] = func(a []uint64) uint64 {
			want := rt.toString(u(a[1]))
			cur := u(a[0])
			for i := 0; i < 64; i++ {
				o, isObj := cur.(*jsObject)
				if !isObj {
					break
				}
				if cls, ok := o.props["__class"]; ok {
					if co, isCls := cls.(*jsObject); isCls {
						if n, ok := co.props["__name"]; ok && rt.toString(n) == want {
							return w(cur)
						}
					}
				}
				nxt, ok := o.props["__outer"]
				if !ok || isUndefOrNull(nxt) {
					break
				}
				cur = nxt
			}
			return a[0]
		}
		// ktIsType is js_ktis as a plain Go call, installed here so the collection
		// builtins (filterIsInstance) can ask the same question the `is` operator asks.
		ktIsType = func(v interface{}, tname string) bool {
			return u(m["js_ktis"]([]uint64{w(v), rt.wrapStr(tname)})) == interface{}(true)
		}
		m["js_ktis"] = func(a []uint64) uint64 {
			v := u(a[0])
			switch v.(type) {
			case jsGInt, jsJFlo, float64:
			default:
				// A PROGRESSION is an IntRange / CharRange / IntProgression and
				// NOT a List - the shared probe has no arm for it and answered
				// false for all three, while the materialized list it replaced
				// answered `is List` true.
				if rg, isRg := ktRangeOf(v); isRg {
					t := rt.toString(u(a[1]))
					if i := strings.IndexByte(t, '<'); i >= 0 {
						t = t[:i]
					}
					return boolH(ktRangeIsType(rg, strings.TrimSuffix(t, "?")))
				}
				if r := baseIs(a); u(r) == interface{}(true) {
					return r
				}
				// A QUALIFIED type name - `is Shape.Circle`, which is how a NESTED
				// class is spelled from outside its owner, and the shape a sealed
				// hierarchy is normally written in. A descriptor is registered under
				// its SIMPLE name, so the qualified spelling matched nothing and the
				// `is` branch of an ordinary `when (this)` over a sealed class fell
				// through to the else. Dropping the qualifier is the same convention
				// this subset already applies to packages and imports; the twin is
				// kIsType in kotlin-interpreter.abnf.
				qt := rt.toString(u(a[1]))
				if i := strings.IndexByte(qt, '<'); i >= 0 {
					qt = qt[:i]
				}
				if i := strings.LastIndexByte(strings.TrimSuffix(qt, "?"), '.'); i >= 0 {
					short := strings.TrimSuffix(qt, "?")[i+1:]
					if strings.HasSuffix(qt, "?") {
						short += "?"
					}
					return baseIs([]uint64{a[0], rt.wrapStr(short)})
				}
				return boolH(false)
			}
			t := rt.toString(u(a[1]))
			if i := strings.IndexByte(t, '<'); i >= 0 {
				t = t[:i]
			}
			t = strings.TrimSuffix(t, "?")
			if t == "Any" || t == "Comparable" {
				return boolH(true)
			}
			switch b := v.(type) {
			case jsGInt:
				if t == "Number" {
					return boolH(!b.u) // the four unsigned types are not kotlin.Number
				}
				return boolH(t == ktBoxTypeName(b))
			case jsJFlo:
				// A Double and a plain number carry a type just as definitely: the
				// value model makes a plain number an Int, so `10 is Long` is FALSE
				// and `1.5 is Double` is TRUE - neither of which the shared probe can
				// answer, since it has one number type and no float box.
				// The style byte says WHICH of the two the box holds, so Double and
				// Float are EXCLUSIVE: `1.5 is Float` and `1.5f is Double` are both
				// false, where both were true while the width was invisible.
				// `is Number` - and `is Any` / `is Comparable` above - stay true for
				// either width, exactly as java.lang.Float and java.lang.Double both
				// extend Number and implement Comparable.
				if t == "Number" {
					return boolH(true)
				}
				if b.sty == floJavaF {
					return boolH(t == "Float")
				}
				return boolH(t == "Double")
			}
			return boolH(t == "Number" || t == "Int")
		}

		// js_ktouter(scope, "Outer") answers the enclosing INSTANCE a bare `Inner()`
		// captures: the first implicit receiver in the scope chain whose class (or a
		// class on its __super chain) is named "Outer", and undefined when there is
		// none - which is what a construction from outside the owner does. The twin is
		// the kNamedType walk in kNew (kotlin-interpreter.abnf).
		m["js_ktouter"] = func(a []uint64) uint64 {
			want := rt.toString(u(a[1]))
			for s := rt.scopeOf(a[0]); s != nil; s = s.parent {
				t, ok := s.get("this")
				if !ok {
					continue
				}
				o, isObj := t.(*jsObject)
				if !isObj {
					continue
				}
				for cls := o.props["__class"]; cls != nil; {
					co, isCls := cls.(*jsObject)
					if !isCls {
						break
					}
					if n, ok := co.props["__name"]; ok && rt.toString(n) == want {
						return w(t)
					}
					cls = co.props["__super"]
				}
			}
			return w(jsUndef)
		}

		// js_ktexcmatch(thrown, "TypeName") is Kotlin's CATCH-CLAUSE selection: does
		// this thrown value match the declared type of this clause? Before it, the
		// compiler half took the FIRST catch clause whatever its type, so
		// `try { throw MyExc() } catch (e: IllegalStateException) {...} catch (e:
		// Exception) {...}` ran the wrong one - while kotlin-interpreter.abnf, which
		// has had kExcMatches all along, ran the right one. The rule is kExcMatches':
		// Throwable/Any accept everything (this model lets any value be thrown), the
		// three intermediate roots accept a NON-throwable too, and everything else is
		// the ordinary dynamic type test.
		m["js_ktexcmatch"] = func(a []uint64) uint64 {
			t := rt.toString(u(a[1]))
			if i := strings.IndexByte(t, '<'); i >= 0 {
				t = t[:i]
			}
			t = strings.TrimSuffix(t, "?")
			if t == "Throwable" || t == "Any" {
				return jsHTrue
			}
			isThr := func() bool {
				o, ok := u(a[0]).(*jsObject)
				if !ok {
					return false
				}
				cls := o.props["__class"]
				for cls != nil {
					co, isCls := cls.(*jsObject)
					if !isCls {
						return false
					}
					if n, ok := co.props["__name"]; ok && rt.toString(n) == "Throwable" {
						return true
					}
					cls = co.props["__super"]
				}
				return false
			}
			if t == "Exception" || t == "RuntimeException" || t == "Error" {
				if !isThr() {
					return jsHTrue
				}
			}
			return m["js_ktis"]([]uint64{a[0], rt.wrapStr(t)})
		}

		// Kotlin's println / print, as a VALUE: the extern takes no arguments and
		// returns the host function, which buildMain declares into the program
		// scope under the names `println` and `print`. Binding a value rather than
		// intercepting the call site keeps `::println` and `list.forEach(::println)`
		// working, and keeps the shared println host function untouched for the
		// twelve other languages that use it.
		m["js_ktprint"] = func(a []uint64) uint64 {
			return rt.wrap(jsHostFunc("println", func(rt *jsrt, this uint64, args []interface{}) interface{} {
				fmt.Fprint(outWriter, wtf8Clean(ktpLine(rt, args)+"\n"))
				return jsUndef
			}))
		}
		m["js_ktwrite"] = func(a []uint64) uint64 {
			return rt.wrap(jsHostFunc("print", func(rt *jsrt, this uint64, args []interface{}) interface{} {
				fmt.Fprint(outWriter, wtf8Clean(ktpLine(rt, args)))
				return jsUndef
			}))
		}

		// The text of a value: what println writes, what "$v" interpolates and what
		// v.toString() answers.
		m["js_ktstr"] = func(a []uint64) uint64 { return rt.wrapStr(rt.ktpRender(u(a[0]), 0)) }

		// js_ktsmcall is the Kotlin method-call entry point: kotlin's STRING member
		// surface first, then the Regex dispatcher (js_rxktmcall), then the shared
		// js_mcall. It exists because the shared js_mcall carries the JAVA String
		// builtins - length/charAt/equals/substring/indexOf/isEmpty and nothing else -
		// while kotlin-interpreter.abnf's own mcall answers a further nine names. Every
		// one of those was a cross-half divergence: `"aBc".uppercase()` printed ABC in
		// the interpreter and aborted the compiler with `unknown String method:
		// uppercase`. The bodies below are the Go twins of the `typeof target ==
		// "string"` branch of that mcall and MUST stay in step with it.
		//
		// The delegate is looked up LAZILY: rxExtraExterns runs its registrars in file
		// order, so js_rxktmcall (abnf/jsrtregexkt.go, a later file name) does not exist
		// yet at this point - capturing it here would bind nil and lose the whole Regex
		// method surface.
		baseMcall := m["js_mcall"]
		m["js_ktsmcall"] = func(a []uint64) uint64 {
			arr, ok := u(a[2]).(*jsArray)
			if !ok {
				rt.fail("js_ktsmcall args must be an array")
			}
			// A NUMERIC receiver: the conversions (toInt/toLong/toUInt/toDouble),
			// inv() and the few other members Kotlin declares on its number types.
			// The shared js_mcall has no method table for numbers at all, so
			// `7.toDouble()` used to abort the compiled half with "method call
			// 'toDouble' on a number" while the interpreter answered 7.0 - a live
			// cross-half divergence that no test happened to exercise.
			recv, mname := u(a[0]), rt.toString(u(a[1]))
			// A callable reference receiver: get / set / invoke, with the receiver as
			// the FIRST argument when the reference is unbound (Kotlin's
			// KProperty1.get(receiver)). The twin is the __propref arm of mcall in
			// kotlin-interpreter.abnf.
			if info, isRef := ktRefOf(recv); isRef {
				switch mname {
				case "invoke":
					return w(rt.ktRefApply(info, arr.elems))
				case "get":
					return w(ktRefRead(ktRefRecv(info, arr.elems), info.name))
				case "set":
					rest := ktRefRest(info, arr.elems)
					ktRefWrite(ktRefRecv(info, arr.elems), info.name, argAt(rest, 0))
					return w(jsUndef)
				case "equals":
					eq, _ := ktRefEq(recv, argAt(arr.elems, 0))
					return boolH(eq)
				case "toString":
					return rt.wrapStr(rt.ktpRender(recv, 0))
				}
				if !info.bound {
					rt.fail("method .%s() of an unbound callable reference ::%s", mname, info.name)
				}
				recv = ktRefRead(info.recv, info.name)
			}
			// A KClass receiver.
			if co, isCls := ktIsClassLit(recv); isCls {
				switch mname {
				case "isInstance":
					return m["js_ktis"]([]uint64{w(argAt(arr.elems, 0)), w(co.props["kname"])})
				case "toString":
					return rt.wrapStr(rt.ktpRender(recv, 0))
				case "equals":
					eq, _ := ktRefEq(recv, argAt(arr.elems, 0))
					return boolH(eq)
				}
				rt.fail("method .%s() of a class literal", mname)
			}
			// A QUALIFIED stdlib call - kotlin.math.abs(-3). The receiver is a
			// package handle (see ktPkg) and the member is the same global function
			// the bare spelling reaches; a call is a METHOD call in the tree, so it
			// never passes through js_ktfget.
			if p, isPkg := ktIsPkg(recv); isPkg {
				f := rt.ktPkgMember(p, mname)
				if !isCallable(f) {
					rt.fail("not a function: %s.%s", p, mname)
				}
				return w(rt.call(f, jsUndef, arr.elems))
			}
			// The SCOPE FUNCTIONS first: let/run/apply/also/takeIf/takeUnless are
			// extensions on Any, so no receiver branch below owns them. A class
			// member of the same name still wins (ktScopeMethod declines then).
			if v, handled := rt.ktScopeMethod(recv, mname, arr.elems); handled {
				return rt.wrap(v)
			}
			// A PROGRESSION. The members whose answer IS the range are answered
			// from the bounds; everything else is a Collection/Iterable extension
			// and runs on the materialized elements. `contains` is answered
			// arithmetically, so `x in 1..1000000` costs nothing - the old
			// representation materialized a million cells at the construction
			// site alone.
			if rg, isRg := ktRangeOf(recv); isRg {
				switch mname {
				case "toString":
					return rt.wrapStr(rt.ktRangeStr(rg))
				case "contains":
					return boolH(ktRangeHas(rg, argAt(arr.elems, 0)))
				case "isEmpty":
					return boolH(ktRangeEmpty(rg))
				case "hashCode":
					return rt.wrap(float64(ktRangeHash(rg)))
				case "equals":
					if org, ok := ktRangeOf(argAt(arr.elems, 0)); ok {
						return boolH(ktRangeEq(rg, org))
					}
					return boolH(false)
				}
				if !rg.fl {
					switch mname {
					// IntProgression.reversed() is an IntProgression, not a List -
					// every engine answered the materialized "[5, 4, 3, 2, 1]", and
					// the `.first`/`.last`/`.step` of that answer were kotlin.Unit
					// here. todo.md 1.5.
					case "reversed":
						return w(ktMakeRange(ktRangeRev(rg)))
					// IntProgression.step(k) in its function spelling; the infix
					// `step` is lowered by the emitter and never arrives here.
					case "step":
						if len(arr.elems) == 1 {
							sn, sok := ktRangeNum(arr.elems[0])
							if !sok || sn <= 0 {
								rt.fail("range step must be positive")
							}
							rg.st = sn
							rg.pg = true
							return w(ktMakeRange(rg))
						}
					// kotlin.random: random() throws NoSuchElementException on an
					// empty progression, randomOrNull() answers null.
					case "random", "randomOrNull":
						cnt := ktRangeCount(rg)
						if cnt == 0 {
							if mname == "randomOrNull" {
								return w(jsNull)
							}
							rt.ktRaise("NoSuchElementException",
								"Cannot get random in empty range: "+rt.ktRangeStr(rg))
						}
						return w(ktRangeAt(rg, ktRndBelow(rt.ktRndOf(arr.elems), cnt)))
					}
				}
				recv = &jsArray{elems: ktRangeList(rg)}
			}
			// kotlin.hashCode() and kotlin.toString() are extensions on Any?, so a
			// NULL receiver answers them rather than throwing (java's
			// Objects.hashCode(null) is 0, String.valueOf(null) is "null"), and a
			// BOOLEAN receiver owns hashCode/toString/not - none of which any branch
			// below claims, so `true.hashCode()` aborted the compiled half with
			// "method call 'hashCode' on a boolean" while the interpreter answered
			// 1231. The twin is the null and boolean tail of mcall in
			// kotlin-interpreter.abnf.
			if isUndefOrNull(recv) {
				switch mname {
				case "hashCode":
					return rt.wrap(float64(0))
				case "toString":
					return rt.wrap("null")
				}
			}
			// An ENUM descriptor: values() / valueOf(name) / entries. The compiler
			// emits __entries (the entry list in declaration order) on the
			// descriptor; the twin is the __isenum branch of mcall / kGetField in
			// kotlin-interpreter.abnf.
			if ev, handled := rt.ktEnumMethod(recv, mname, arr.elems); handled {
				return rt.wrap(ev)
			}
			if bv, isBool := recv.(bool); isBool {
				switch mname {
				case "hashCode":
					if bv {
						return rt.wrap(float64(1231))
					}
					return rt.wrap(float64(1237))
				case "toString":
					if bv {
						return rt.wrap("true")
					}
					return rt.wrap("false")
				case "not":
					return rt.wrap(!bv)
				case "equals":
					ov, isB := argAt(arr.elems, 0).(bool)
					return rt.wrap(isB && ov == bv)
				case "compareTo":
					ov, _ := argAt(arr.elems, 0).(bool)
					switch {
					case bv == ov:
						return rt.wrap(float64(0))
					case bv:
						return rt.wrap(float64(1))
					}
					return rt.wrap(float64(-1))
				}
			}
			if v, handled := rt.ktNumMethod(recv, mname, arr.elems); handled {
				return rt.wrap(v)
			}
			if c, isChar := recv.(jsChar); isChar {
				if v, handled := rt.ktCharMethod(c, mname, arr.elems); handled {
					return rt.wrap(v)
				}
			}
			if lst, isArr := recv.(*jsArray); isArr {
				if v, handled := rt.ktSeqMethod(lst, mname, arr.elems); handled {
					return rt.wrap(v)
				}
			}
			// kotlin.random.Random's COMPANION OBJECT: `Random.nextInt(10)` and
			// `Random.Default`. `Random` itself is the seeded-constructor host
			// function (js_ktglobal), so the companion spellings arrive as a member
			// call on that function value - which is why the receiver is matched by
			// its host-function NAME rather than by a marker on an object.
			if hf, isHF := recv.(*hostFunc); isHF && hf.name == "Random" {
				if v, handled := rt.ktRndMethod(rt.ktRndDefault(), mname, arr.elems); handled {
					return rt.wrap(v)
				}
			}
			if mo, isObj := recv.(*jsObject); isObj {
				// A kotlin.random.Random receiver.
				if ktIsRnd(mo) {
					if v, handled := rt.ktRndMethod(mo, mname, arr.elems); handled {
						return rt.wrap(v)
					}
				}
				if ktIsSb(mo) {
					if v, handled := rt.ktSbMethod(mo, mname, arr.elems); handled {
						return rt.wrap(v)
					}
				}
				if ktIsSet(mo) {
					if v, handled := rt.ktSetMethod(mo, mname, arr.elems); handled {
						return rt.wrap(v)
					}
				}
				if _, _, isDict := dictParts(mo); isDict {
					if v, handled := rt.ktMapMethod(mo, mname, arr.elems); handled {
						return rt.wrap(v)
					}
				} else if ro, isRes := ktIsResult(mo); isRes {
					if v, handled := rt.ktResultMethod(ro, mname, arr.elems); handled {
						return rt.wrap(v)
					}
				} else if lz, _ := mo.props["__lazy"].(bool); lz {
					switch mname {
					case "getValue", "value":
						return rt.wrap(rt.ktLazyValue(mo))
					case "isInitialized":
						return rt.wrap(mo.props["done"])
					}
				} else if j, isJob := ktIsJob(mo); isJob {
					// The Job / Deferred an `async` or `launch` answers.
					if v, handled := rt.ktJobMethod(j, mname); handled {
						return rt.wrap(v)
					}
				} else if po, isPair := ktIsPair(mo); isPair {
					if v, handled := rt.ktPairMethod2(po, mname, arr.elems); handled {
						return rt.wrap(v)
					}
				}
			}
			if s, isStr := u(a[0]).(string); isStr {
				name := rt.toString(u(a[1]))
				// A pattern argument belongs to the Regex dispatcher: `s.contains(re)`
				// is a regex search, `s.contains("B")` is a substring test.
				pat := len(arr.elems) > 0 && ktRxIsRegex(arr.elems[0])
				if v, handled := rt.ktStrMethod(s, name, arr.elems); handled && !pat {
					return rt.wrap(v)
				}
			}
			// SAM CONVERSION: `Op43 { it + 100 }` on a `fun interface` builds an
			// instance whose single abstract member is carried as a FIELD holding the
			// lambda (the descriptor has no such method - every conversion has its own
			// body). The shared js_mcall stops at the __class chain and fails with
			// "unknown method '...' on an instance", so the field is answered here, and
			// only when the chain does NOT declare the name: a real method always wins,
			// which is the order kotlin-interpreter.abnf's mcall uses. The field is
			// applied WITHOUT the receiver, exactly as that half does.
			if mo, isObj := recv.(*jsObject); isObj {
				if _, hasCls := mo.props["__class"]; hasCls {
					if fv, ok := mo.props[mname]; ok && isCallable(fv) && !ktClassChainHas(mo, mname) {
						return w(rt.call(fv, jsUndef, arr.elems))
					}
					// Every Kotlin class inherits equals() from Any. A class that
					// overrides it has it on the chain and is dispatched below; one
					// that does not used to ABORT here with "unknown method 'equals'
					// on an instance" while kotlin-interpreter.abnf answered the
					// identity comparison.
					if mname == "equals" && !ktClassChainHas(mo, "equals") {
						return boolH(rt.strictEq(recv, argAt(arr.elems, 0)))
					}
				}
			}
			// toString() / hashCode() are members of kotlin.Any, so EVERY receiver
			// has them - and after every branch above has declined, nothing else
			// can. An unqualified `with(p) { toString() }` reached "unknown list
			// method 'toString'" / "unknown name: toString" in all three engines
			// (todo.md 1.10), and so did the same call on a Pair, a Map.Entry, a
			// number and a range. A class that OVERRIDES either has it on its
			// descriptor chain and is dispatched by baseMcall below, which is why
			// the chain is asked first - the same guard the `equals` arm uses.
			// A Regex / MatchResult receiver reaches its own toString through
			// ktpObj, so routing it here answers exactly what the rx chain would.
			if len(arr.elems) == 0 && (mname == "toString" || mname == "hashCode") &&
				!ktChainDeclares(recv, mname) {
				if mname == "toString" {
					return rt.wrapStr(rt.ktpRender(recv, 0))
				}
				return rt.wrap(float64(int32(ktElemHash(rt, recv))))
			}
			// equals(other) is the third Any member, and it is the SAME comparison
			// `==` makes - js_kteq. `listOf(1, 2).equals(listOf(1, 2))` aborted
			// with "unknown list method 'equals'" in both compiled halves while
			// the interpreter answered true: a live divergence, found sweeping
			// todo.md 1.10.
			if len(arr.elems) == 1 && mname == "equals" && !ktChainDeclares(recv, mname) {
				return m["js_kteq"]([]uint64{w(recv), w(argAt(arr.elems, 0))})
			}
			if next := m["js_rxktmcall"]; next != nil {
				return next(a)
			}
			return baseMcall(a)
		}

		// ----- the global builders and the implicit receiver -----

		// ktMemberCall lets the bound-builtin closures reach the whole method surface.
		ktMemberCall = func(recv interface{}, name string, args []interface{}) interface{} {
			if args == nil {
				args = []interface{}{}
			}
			return u(m["js_ktsmcall"]([]uint64{w(recv), rt.wrapStr(name), w(&jsArray{elems: args})}))
		}
		ktRecvStack = nil
		// The reference machinery reaches fields through exactly the externs the
		// emitted code uses, and its two side tables are per-runtime.
		ktRefRead = func(recv interface{}, name string) interface{} {
			return u(m["js_ktfget"]([]uint64{w(recv), rt.wrapStr(name)}))
		}
		ktRefWrite = func(recv interface{}, name string, v interface{}) {
			rt.setMember(recv, name, v)
		}
		ktRefs = map[*hostFunc]*ktRefInfo{}
		ktFuncNames = map[interface{}]string{}

		// js_ktrefbase(scope, "Box") resolves the BASE of a `::`. A base that names a
		// value answers that value; one that names a TYPE this value model has no
		// binding for (String::length) answers a ktTypeName marker instead of failing,
		// which is what makes the reference unbound. The twin is the hasVar test in
		// makePropRef (kotlin-interpreter.abnf).
		m["js_ktrefbase"] = func(a []uint64) uint64 {
			sc := rt.scopeOf(a[0])
			name := rt.toString(u(a[1]))
			// A builtin TYPE name is a type even though the scope carries a same-named
			// builder for it: without this, `String::length` bound itself to the String
			// conversion function and .get("abcd") answered kotlin.Unit here while the
			// interpreter half answered 4.
			if ktTypeNames[name] {
				return w(ktTypeName{name: name})
			}
			for s := sc; s != nil; s = s.parent {
				if v, ok := s.get(name); ok {
					return w(v)
				}
			}
			for s := sc; s != nil; s = s.parent {
				if t, ok := s.get("this"); ok {
					if obj, isObj := t.(*jsObject); isObj {
						if _, ok := obj.props[name]; ok {
							return w(t)
						}
					}
				}
			}
			return w(ktTypeName{name: name})
		}

		// js_ktref(base, "v") is `base::v`. The base decides which of Kotlin's two
		// reference kinds it is: a class descriptor or a type name gives an UNBOUND
		// reference (KProperty1 / KFunction1, whose get/set/invoke take the receiver
		// first), anything else a BOUND one.
		m["js_ktref"] = func(a []uint64) uint64 {
			base, name := u(a[0]), rt.toString(u(a[1]))
			var ext interface{}
			if len(a) > 2 && isCallable(u(a[2])) {
				ext = u(a[2])
			}
			if tn, isType := base.(ktTypeName); isType {
				return w(rt.ktMakeRef(&ktRefInfo{name: name, tname: tn.name, ext: ext}))
			}
			if o, isObj := base.(*jsObject); isObj {
				if b, isCls := o.props["__isclass"].(bool); isCls && b {
					nm, _ := o.props["__name"].(string)
					return w(rt.ktMakeRef(&ktRefInfo{name: name, tname: nm, ext: ext}))
				}
			}
			return w(rt.ktMakeRef(&ktRefInfo{recv: base, name: name, bound: true, ext: ext}))
		}

		// js_ktbareref(scope, "name") is a bare `::name`. A scope binding is the value
		// itself (::fn has always been the closure), except that a CLASS descriptor
		// becomes a constructor reference; a name the scope does not carry but the
		// implicit receiver DOES becomes a bound property reference, which is what makes
		// `::prop` inside a member a KProperty0. Anything else falls through to the
		// ordinary name resolution. The twin is makeBareRef in kotlin-interpreter.abnf.
		m["js_ktbareref"] = func(a []uint64) uint64 {
			sc := rt.scopeOf(a[0])
			name := rt.toString(u(a[1]))
			for s := sc; s != nil; s = s.parent {
				if v, ok := s.get(name); ok {
					if isCallable(v) {
						ktFuncNames[v] = name
					}
					return m["js_ktctorref"]([]uint64{w(v)})
				}
			}
			for s := sc; s != nil; s = s.parent {
				if t, ok := s.get("this"); ok {
					if obj, isObj := t.(*jsObject); isObj {
						if _, has := obj.props[name]; has {
							return w(rt.ktMakeRef(&ktRefInfo{recv: obj, name: name, bound: true}))
						}
					}
				}
			}
			r := m["js_ktget"](a)
			if isCallable(u(r)) {
				ktFuncNames[u(r)] = name
			}
			return m["js_ktctorref"]([]uint64{r})
		}

		// js_ktctorref(v) turns `::Box` into a CONSTRUCTOR reference (Kotlin names it
		// "<init>"): callable, and passable where a lambda is expected. A bare `::fn`
		// is handed back untouched - a function is already a first-class value.
		m["js_ktctorref"] = func(a []uint64) uint64 {
			v := u(a[0])
			if o, isObj := v.(*jsObject); isObj {
				if b, isCls := o.props["__isclass"].(bool); isCls && b {
					return w(rt.ktMakeRef(&ktRefInfo{recv: o, name: "<init>", bound: true, ctor: true}))
				}
			}
			return a[0]
		}

		// js_ktclasslit(base) is `base::class`.
		m["js_ktclasslit"] = func(a []uint64) uint64 {
			base := u(a[0])
			if tn, isType := base.(ktTypeName); isType {
				return w(ktClassLit(tn.name))
			}
			if o, isObj := base.(*jsObject); isObj {
				if b, isCls := o.props["__isclass"].(bool); isCls && b {
					return w(ktClassLit(o.props["__name"]))
				}
			}
			return w(ktClassLit(ktSimpleName(base)))
		}

		// js_ktnameval(v, "topFun") remembers the name a `::` spelled against the
		// closure it resolved to, so KFunction.name can answer it, and hands the value
		// straight back. The twin is fnRefName in kotlin-interpreter.abnf.
		m["js_ktnameval"] = func(a []uint64) uint64 {
			if isCallable(u(a[0])) {
				ktFuncNames[u(a[0])] = rt.toString(u(a[1]))
			}
			return a[0]
		}

		// js_ktglobal(name) is the VALUE of one global builder; buildMain declares
		// one per name. See the block above ktRecvStack.
		m["js_ktglobal"] = func(a []uint64) uint64 { return w(ktGlobalFn(rt, rt.toString(u(a[0])))) }

		// js_ktpropset(this, "x", v) stores ONE property initializer. It is not js_set,
		// because a property a SUBCLASS overrides with accessors has a js_defprop pair
		// on the descriptor chain, and the shared setMember then routes the base
		// class's own initializer into the subclass's setter - or, for a getter-only
		// override, DROPS it. Kotlin initializes the declaring class's backing field
		// directly, so that is what happens here: the value lands in the backing slot
		// the accessor model already uses (`x$f`), which is where js_ktsupget then
		// finds it for `super.x`. Without this, `open class B { open val v = 1 }` /
		// `class D : B() { override val v get() = super.v + 1 }` answered 1 (the value
		// was never stored at all) where Kotlin and the interpreter half answer 2.
		m["js_ktpropset"] = func(a []uint64) uint64 {
			o, isObj := u(a[0]).(*jsObject)
			if !isObj {
				return m["js_set"](a)
			}
			name := rt.toString(u(a[1]))
			if _, own := o.props[name]; !own && rt.accessorCount > 0 {
				if acc := rt.findClassAccessor(o, name); acc != nil {
					o.set(name+"$f", u(a[2]))
					return 0
				}
			}
			o.set(name, u(a[2]))
			return 0
		}

		// js_ktsupget(super class, this, "x") is `super.x` as a VALUE. The walk starts
		// AT the given descriptor (the emitter already resolved the superclass of the
		// DEFINING class, exactly as js_supercall does), so an override never shadows
		// what super.x means. An accessor found on the way runs with the receiver.
		// Kotlin differs from the shared js_supget in the FALLBACK: a supertype that
		// declares `open val x = 1` keeps no accessor at all - the value lives in the
		// instance's own slot, which is that property's backing field in this flat
		// object model - so a chain miss reads the receiver rather than answering
		// undefined.
		m["js_ktsupget"] = func(a []uint64) uint64 {
			name := rt.toString(u(a[2]))
			for cls := u(a[0]); cls != nil; {
				clsObj, isObj := cls.(*jsObject)
				if !isObj {
					break
				}
				if v, found := clsObj.props[name]; found {
					if acc, isAcc := v.(*jsAccessor); isAcc {
						if acc.get == nil {
							return w(jsUndef)
						}
						return w(rt.call(acc.get, u(a[1]), nil))
					}
					return w(v)
				}
				cls = clsObj.props["__super"]
			}
			if o, isObj := u(a[1]).(*jsObject); isObj {
				if v, found := o.props[name]; found {
					return w(v)
				}
				// A property the supertype declared with accessors keeps its value in
				// the emitted backing slot.
				if v, found := o.props[name+"$f"]; found {
					return w(v)
				}
			}
			return w(jsNull)
		}

		// js_ktsupset(super class, this, "x", v) is `super.x = v`. A setter found on
		// the chain runs with the receiver; otherwise the value lands in the slot
		// js_ktpropset chose for the declaring class's backing field, so a `super.x`
		// read sees it again.
		m["js_ktsupset"] = func(a []uint64) uint64 {
			name := rt.toString(u(a[2]))
			for cls := u(a[0]); cls != nil; {
				clsObj, isObj := cls.(*jsObject)
				if !isObj {
					break
				}
				if v, found := clsObj.props[name]; found {
					if acc, isAcc := v.(*jsAccessor); isAcc {
						if acc.set != nil {
							rt.call(acc.set, u(a[1]), []interface{}{u(a[3])})
						}
						return 0
					}
					break
				}
				cls = clsObj.props["__super"]
			}
			if o, isObj := u(a[1]).(*jsObject); isObj {
				if _, own := o.props[name]; own {
					o.set(name, u(a[3]))
					return 0
				}
				o.set(name+"$f", u(a[3]))
				return 0
			}
			return 0
		}

		// js_ktmextcall(scope, receiver, "name", args) dispatches a MEMBER extension
		// function: one declared inside a class, whose `this` is the extension
		// receiver while the enclosing instance rides along as the dispatch receiver.
		// Which class instance is enclosing is a RUN-TIME question (the emitter only
		// knows that some class declares the name), so the lookup walks the frame's
		// `this` and `__dispatch` bindings and their __class / __outer graphs for an
		// `ext$name` entry, exactly as memberExtLookup does in the interpreter half.
		// Nothing found means the name was an ordinary method after all.
		m["js_ktmextcall"] = func(a []uint64) uint64 {
			name := rt.toString(u(a[2]))
			args, ok := u(a[3]).(*jsArray)
			if !ok {
				rt.fail("js_ktmextcall args must be an array")
			}
			if fn, disp, found := rt.ktMemberExtFind(rt.scopeOf(a[0]), "ext$"+name); found {
				return rt.wrap(rt.call(fn, jsUndef, append([]interface{}{disp, u(a[1])}, args.elems...)))
			}
			return m["js_ktsmcall"]([]uint64{a[1], a[2], a[3]})
		}

		// js_ktmextget(scope, recv, "name") / js_ktmextset(scope, recv, "name", v) are
		// the same lookup for a member extension PROPERTY - `val Int.scaled get() = ...`
		// declared inside a class. The accessor pair is installed as extget$name /
		// extset$name and takes (dispatchReceiver, extensionReceiver[, value]), exactly
		// like the ext$name functions above and exactly like makeMemberExtGetter /
		// makeMemberExtSetter in kotlin-interpreter.abnf. Nothing found means the name
		// was an ordinary field after all.
		m["js_ktmextget"] = func(a []uint64) uint64 {
			name := rt.toString(u(a[2]))
			if fn, disp, found := rt.ktMemberExtFind(rt.scopeOf(a[0]), "extget$"+name); found {
				return rt.wrap(rt.call(fn, jsUndef, []interface{}{disp, u(a[1])}))
			}
			return m["js_ktfget"]([]uint64{a[1], a[2]})
		}
		m["js_ktmextset"] = func(a []uint64) uint64 {
			name := rt.toString(u(a[2]))
			if fn, disp, found := rt.ktMemberExtFind(rt.scopeOf(a[0]), "extset$"+name); found {
				rt.call(fn, jsUndef, []interface{}{disp, u(a[1]), u(a[3])})
				return a[3]
			}
			rt.setMember(u(a[1]), name, u(a[3]))
			return a[3]
		}

		// js_ktrecvlam(f) wraps the value bound to a parameter whose declared type is a
		// receiver function type (`body: Node.() -> Unit`). The wrapper takes the
		// receiver in its FIRST slot and runs f with that receiver as `this`, which is
		// exactly what ktWithRecv does for with / run / apply - so `t.body()` and
		// `body(t)` are the same call, as in Kotlin, and the body's unqualified member
		// reads resolve against the receiver.
		m["js_ktrecvlam"] = func(a []uint64) uint64 {
			f := u(a[0])
			if !isCallable(f) {
				return a[0]
			}
			return rt.wrap(jsHostFunc("<recvlam>", func(rt *jsrt, this uint64, args []interface{}) interface{} {
				if len(args) == 0 {
					return rt.ktWithRecv(jsNull, f)
				}
				// The receiver is passed on as the first ARGUMENT as well as bound as
				// `this`: `fun scoped(block: FeatScope.() -> Int) = block(Scope(9))`
				// with `scoped { s -> s.v }` reads it through the value parameter,
				// which is what both halves did before the receiver was bound at all.
				return rt.ktWithRecv(args[0], f, args...)
			}))
		}

		// ----- the delegated-property protocol (val/var x by d) -----
		// The twin of delegCheck / pdelegGet / pdelegSet in kotlin-interpreter.abnf.
		// A delegate is any value whose class chain declares getValue; the check runs
		// when the DECLARATION runs, exactly as it does in the other half, so the
		// message and the -warn-unsupported placeholder are the same in both.
		// The property reference handed to getValue/setValue. Only `.name` is
		// meaningful for a delegate, which is what Kotlin's own Map delegate uses.
		ktdRef := func(owner interface{}, name string) interface{} {
			if o, isObj := owner.(*jsObject); isObj {
				return rt.ktMakeRef(&ktRefInfo{recv: o, name: name, bound: true})
			}
			return rt.ktMakeRef(&ktRefInfo{name: name, bound: true})
		}
		// js_ktdcheck(d, "file:line", warn, thisRef, "name") answers the delegate to
		// store - d itself, or what its provideDelegate hands back - and null after
		// reporting when d is no delegate at all. The four delegate kinds Kotlin's
		// stdlib and user code produce: a user object with getValue, a MAP (delegated
		// by the property's NAME), the kotlin.properties.Delegates boxes, and a lazy
		// box (`by lazy` has its own eager lowering, but a lazy value reaching here
		// through a variable still works).
		m["js_ktdcheck"] = func(a []uint64) uint64 {
			d := u(a[0])
			name := ""
			var thisRef interface{} = jsNull
			if len(a) > 4 {
				thisRef, name = u(a[3]), rt.toString(u(a[4]))
			}
			if ktHasMethod(d, "provideDelegate") {
				d = u(m["js_ktsmcall"]([]uint64{w(d), rt.wrapStr("provideDelegate"),
					w(&jsArray{elems: []interface{}{thisRef, ktdRef(thisRef, name)}})}))
			}
			if ktDelegKind(d) != "" {
				return w(d)
			}
			where := rt.toString(u(a[1]))
			if u(a[2]) == float64(1) || u(a[2]) == true {
				fmt.Fprint(warnWriter, "warning: "+where+": delegated property not implemented (ignored)\n")
				return w(jsNull)
			}
			rt.fail("delegated property not implemented (%s); use -warn-unsupported to ignore", where)
			return jsHUndefined
		}
		// js_ktdthis(scope) is the `this` a delegate declaration should pass as
		// thisRef: the enclosing instance for a member, null for a local or a
		// top-level property. It cannot be a plain scope read - there is no `this`
		// binding at top level, and js_ktget aborts on a name it cannot resolve.
		m["js_ktdthis"] = func(a []uint64) uint64 {
			for s := rt.scopeOf(a[0]); s != nil; s = s.parent {
				if t, ok := s.get("this"); ok {
					return w(t)
				}
			}
			return w(jsNull)
		}
		// js_ktdget(d, thisRef, "name") - one read of a delegated property.
		m["js_ktdget"] = func(a []uint64) uint64 {
			d := u(a[0])
			name := rt.toString(u(a[2]))
			switch ktDelegKind(d) {
			case "":
				return w(jsNull)
			case "map":
				// Kotlin's Map delegate reads the entry under the property's NAME,
				// and getValue on a missing key is an error (Map.get would answer
				// null; the delegate uses getValue).
				return m["js_ktsmcall"]([]uint64{a[0], rt.wrapStr("getValue"),
					w(&jsArray{elems: []interface{}{name}})})
			case "lazy":
				return w(rt.ktLazyValue(d.(*jsObject)))
			case "box":
				bo := d.(*jsObject)
				if k, _ := bo.props["__ktdeleg"].(string); k == "notNull" {
					if set, _ := bo.props["set"].(bool); !set {
						rt.fail("Property %s should be initialized before get.", name)
					}
				}
				return w(bo.props["v"])
			}
			return m["js_ktsmcall"]([]uint64{a[0], rt.wrapStr("getValue"),
				w(&jsArray{elems: []interface{}{u(a[1]), ktdRef(u(a[1]), name)}})})
		}
		// js_ktdset(d, thisRef, "name", v) - one write.
		m["js_ktdset"] = func(a []uint64) uint64 {
			d := u(a[0])
			name := rt.toString(u(a[2]))
			switch ktDelegKind(d) {
			case "", "lazy":
				return 0
			case "map":
				rt.ktMapPut(d.(*jsObject), name, u(a[3]))
				return 0
			case "box":
				// kotlin.properties.Delegates: observable notifies AFTER the change,
				// vetoable asks BEFORE it and keeps the old value on a false answer,
				// notNull simply refuses to be read before it is written.
				box := d.(*jsObject)
				old, nv := box.props["v"], u(a[3])
				kind, _ := box.props["__ktdeleg"].(string)
				cb := box.props["cb"]
				if kind == "vetoable" && cb != nil && !isUndefOrNull(cb) {
					if !rt.truthy(rt.call(cb, jsUndef, []interface{}{ktdRef(u(a[1]), name), old, nv})) {
						return 0
					}
				}
				box.set("v", nv)
				box.set("set", true)
				if kind == "observable" && cb != nil && !isUndefOrNull(cb) {
					rt.call(cb, jsUndef, []interface{}{ktdRef(u(a[1]), name), old, nv})
				}
				return 0
			}
			m["js_ktsmcall"]([]uint64{a[0], rt.wrapStr("setValue"),
				w(&jsArray{elems: []interface{}{u(a[1]), ktdRef(u(a[1]), name), u(a[3])}})})
			return 0
		}
		// A LOCAL or top-level delegated property binds a BOX; js_ktget unwraps it on
		// every read and js_ktvarset routes every write through setValue. thisRef is
		// null, as it is in Kotlin for a property with no owner.
		m["js_ktdbox"] = func(a []uint64) uint64 {
			if isUndefOrNull(u(a[0])) {
				return w(jsNull)
			}
			box := newJSObject()
			box.set("__pdeleg", true)
			box.set("d", u(a[0]))
			box.set("owner", jsNull)
			box.set("pname", rt.toString(u(a[1])))
			return w(box)
		}
		// js_ktdmemo(box) answers the delegate of a MEMBER-level extension delegated
		// property (`val Ctx.mem: Int by D()` inside a class), resolving it on FIRST
		// ACCESS and remembering it in the box. The delegate still belongs to the
		// declaration site - one object shared by every receiver - but it cannot be
		// evaluated when the descriptor is built: the compiler half emits class items
		// in topological order, so a `by` expression naming a top-level val is not
		// bound yet at that point, and an eager resolution aborted with "unknown
		// name: <val>" in a file where the same two declarations worked in isolation.
		// The box is {init: closure, d, done}; kotlin-interpreter.abnf's
		// installMemberExtDeleg memoizes identically, so both halves run the `by`
		// expression at the same moment.
		m["js_ktdmemo"] = func(a []uint64) uint64 {
			box, ok := u(a[0]).(*jsObject)
			if !ok {
				return w(jsNull)
			}
			if done, _ := box.props["done"].(bool); done {
				return w(box.props["d"])
			}
			box.set("done", true)
			d := rt.call(box.props["init"], jsUndef, nil)
			box.set("d", d)
			return w(d)
		}
		// js_ktvarset(scope, name, v) is js_kset made delegate-aware: a binding
		// holding a delegate box is WRITTEN through its setValue.
		m["js_ktvarset"] = func(a []uint64) uint64 {
			sc := rt.scopeOf(a[0])
			name := rt.toString(u(a[1]))
			for s := sc; s != nil; s = s.parent {
				if cur, ok := s.get(name); ok {
					if box, isBox := ktPDeleg(cur); isBox {
						m["js_ktdset"]([]uint64{w(box.props["d"]), w(box.props["owner"]),
							w(box.props["pname"]), a[2]})
						return 0
					}
					break
				}
			}
			// The dispatch receiver of a MEMBER extension is a write target too:
			// `fun String.emit() { out += this }` assigns the declaring class's own
			// property while `this` is the String. js_kset stops at the first scope
			// carrying a `this`, so the fallback has to be tried here.
			for s := sc; s != nil; s = s.parent {
				if _, ok := s.get(name); ok {
					return m["js_kset"](a)
				}
				if t, ok := s.get("this"); ok {
					if obj, isObj := t.(*jsObject); isObj {
						if _, has := obj.props[name]; has {
							return m["js_kset"](a)
						}
						// The write twin of the accessor branch in js_ktget: a
						// property with a SETTER (and every delegated property) is a
						// js_defprop pair on the DESCRIPTOR, so an unqualified
						// `m += 1` inside the class's own member aborted with
						// "assignment to unknown name: m" while the interpreter half
						// wrote through the setter.
						if acc := rt.findClassAccessor(obj, name); acc != nil {
							if acc.set != nil {
								rt.call(acc.set, obj, []interface{}{u(a[2])})
							}
							if rt.traced {
								rt.trVar("write", name, u(a[2]))
							}
							return 0
						}
					}
				}
				if d, ok := s.get("__dispatch"); ok {
					if obj, isObj := d.(*jsObject); isObj {
						if _, has := obj.props[name]; has {
							obj.set(name, u(a[2]))
							if rt.traced {
								rt.trVar("write", name, u(a[2]))
							}
							return 0
						}
					}
				}
			}
			return m["js_kset"](a)
		}

		// js_ktiter is the SEQUENCE a `for` runs over. A list, a string and anything
		// else are used as they are; a MAP iterates its entries (Kotlin's rule, and
		// what kIterable does in the other half) and a SET its elements, neither of
		// which has a `length` for the index loop the grammar emits. It replaced an
		// inline __dict probe at the emit site, which ran a set's for-loop zero times.
		// js_ktrange builds a progression: `down` is 1 for downTo and 0 otherwise,
		// and the `..<` / until subtraction on the bound is the emitter's, exactly
		// as it was when the emitter materialized the list itself.
		m["js_ktrange"] = func(a []uint64) uint64 {
			from, to, st := u(a[0]), u(a[1]), u(a[2])
			fn, fok := ktRangeNum(from)
			tn, tok := ktRangeNum(to)
			sn, sok := ktRangeNum(st)
			if !fok || !tok || !sok {
				rt.fail("range bound is not a number")
			}
			if sn <= 0 {
				rt.fail("range step must be positive")
			}
			_, isCh := from.(jsChar)
			_, isFl := from.(jsJFlo)
			return w(ktMakeRange(ktRange{from: fn, to: tn, st: sn,
				down: rt.toNumber(u(a[3])) != 0, ch: isCh, fl: isFl}))
		}
		// `r step k` re-steps an existing progression, keeping bounds and direction.
		m["js_ktrestep"] = func(a []uint64) uint64 {
			rg, isRg := ktRangeOf(u(a[0]))
			if !isRg {
				rt.fail("step applied to a non-range")
			}
			sn, sok := ktRangeNum(u(a[1]))
			if !sok || sn <= 0 {
				rt.fail("range step must be positive")
			}
			rg.st = sn
			// `step` answers an IntProgression, never an IntRange, whatever the
			// step is: `(1..5) step 1` prints "1..5 step 1" and `is IntRange` is
			// false for it.
			rg.pg = true
			return w(ktMakeRange(rg))
		}
		m["js_ktiter"] = func(a []uint64) uint64 {
			v := u(a[0])
			if o, isObj := v.(*jsObject); isObj {
				// A PROGRESSION is a value, not a list: the loop runs over its
				// elements.
				if rg, isRg := ktRangeOf(o); isRg {
					return w(&jsArray{elems: ktRangeList(rg)})
				}
				if _, _, isDict := dictParts(o); isDict {
					return w(&jsArray{elems: ktElems(rt, o)})
				}
				if keys, isSet := ktSetParts(o); isSet {
					return w(keys)
				}
			}
			return a[0]
		}

		// js_ktget is js_kget plus the implicit receiver of with / run / apply /
		// buildString / buildList / buildMap: a name that is neither a local nor a
		// member of the enclosing `this` is resolved against the receiver on
		// ktRecvStack, so `with(sb) { append("a") }` and `with(list) { size }` mean
		// what they mean in Kotlin. The two earlier steps are js_kget's, byte for
		// byte, so a name that already resolved resolves the same way.
		m["js_ktget"] = func(a []uint64) uint64 {
			sc := rt.scopeOf(a[0])
			name := rt.toString(u(a[1]))
			// a[2] is the SYNTACTIC POSITION the emitter recorded - true for
			// `name(...)`, false for a bare `name`. See ktRecvMember. Older callers
			// (and the tracer) pass two arguments and mean a value read.
			iscall := len(a) > 2 && rt.truthy(u(a[2]))
			for s := sc; s != nil; s = s.parent {
				if v, ok := s.get(name); ok {
					// A LOCAL or top-level delegated property binds a box, not a
					// value: every read goes through the delegate's getValue. The
					// twin is the makeVarRef override in kotlin-interpreter.abnf.
					if box, isBox := ktPDeleg(v); isBox {
						v = u(m["js_ktdget"]([]uint64{w(box.props["d"]), w(box.props["owner"]),
							w(box.props["pname"])}))
					}
					if rt.traced {
						rt.trVar("read", name, v)
					}
					return w(v)
				}
			}
			// Kotlin's implicit receivers form a STACK, and an unqualified name the
			// innermost one does not answer falls through to the next one OUT. This
			// loop used to `break` at the first scope carrying a `this`, so
			// `with(P(1)) { tag }` written inside a member of Q - where ktWithRecv
			// inserts a scope binding `this` to the P - failed with "unknown name:
			// tag" instead of reading this@Q.tag. The twin is findThisChain /
			// core.varMiss in kotlin-interpreter.abnf.
			for s := sc; s != nil; s = s.parent {
				if t, ok := s.get("this"); ok {
					if obj, isObj := t.(*jsObject); isObj {
						if v, ok := obj.props[name]; ok {
							// An unqualified read of a `lateinit var` throws
							// too - see ktLateinitCheck.
							rt.ktLateinitCheck(obj, name, v)
							if rt.traced {
								rt.trVar("read", name, v)
							}
							return w(v)
						}
						// A property with an ACCESSOR - `val c: Int get() = b * 2`,
						// and every DELEGATED property - is a js_defprop pair on the
						// DESCRIPTOR, not an own property of the instance, so the
						// lookup above missed it and an unqualified read of the
						// class's own computed property aborted with "unknown name:
						// c" while the interpreter half (whose recvLookup consults
						// get$name) answered. Only the accessor case is added here;
						// a method name still falls through, as it did before.
						if acc := rt.findClassAccessor(obj, name); acc != nil {
							var v interface{} = jsUndef
							if acc.get != nil {
								v = rt.call(acc.get, obj, nil)
							}
							if rt.traced {
								rt.trVar("read", name, v)
							}
							return w(v)
						}
					} else if !isUndefOrNull(t) {
						if v := rt.getMember(t, name); !isUndefOrNull(v) {
							if rt.traced {
								rt.trVar("read", name, v)
							}
							return w(v)
						}
					}
				}
			}
			// Inside a MEMBER extension function, `this` is the EXTENSION receiver, so
			// a name that belongs to the DECLARING class is resolved against the
			// dispatch receiver the frame carries - Kotlin's second implicit receiver.
			// The twin is core.varMiss consulting findDispatch in the other half.
			for s := sc; s != nil; s = s.parent {
				if d, ok := s.get("__dispatch"); ok {
					if v, hit := rt.ktRecvMember(d, name, iscall); hit {
						if rt.traced {
							rt.trVar("read", name, v)
						}
						return w(v)
					}
					break
				}
			}
			for i := len(ktRecvStack) - 1; i >= 0; i-- {
				if v, ok := rt.ktRecvMember(ktRecvStack[i], name, iscall); ok {
					if rt.traced {
						rt.trVar("read", name, v)
					}
					return w(v)
				}
			}
			// `kotlin` is the head of a QUALIFIED stdlib path - kotlin.math.abs(-3),
			// the same function the bare `abs(-3)` reaches. It resolved in NEITHER
			// half before ("unknown name: kotlin"). See ktPkgMember.
			if name == "kotlin" {
				return w(ktPkg("kotlin"))
			}
			rt.fail("unknown name: %s", name)
			return 0
		}

		// js_ktfget is the field-read extern: the StringBuilder box's properties, a
		// Map MISS as null (kotlin.Map.get answers null, where the shared js_get
		// answers undefined - the one place the two halves still disagreed on a map),
		// then js_rxktget. The delegate is looked up LAZILY, for the reason
		// js_ktsmcall's is.
		m["js_ktfget"] = func(a []uint64) uint64 {
			o, name := u(a[0]), rt.toString(u(a[1]))
			if p, isPkg := ktIsPkg(o); isPkg {
				return w(rt.ktPkgMember(p, name))
			}
			// kotlin.random.Random.Default - the companion object, which IS the
			// default generator here. `Random` is the seeded-constructor host
			// function, so this is a member read on that function value.
			if hf, isHF := o.(*hostFunc); isHF && hf.name == "Random" && name == "Default" {
				return w(rt.ktRndDefault())
			}
			// A callable reference: KCallable.name. The twin is the __propref arm of
			// kGetField in kotlin-interpreter.abnf.
			if info, isRef := ktRefOf(o); isRef {
				if name == "name" {
					return rt.wrapStr(info.name)
				}
				if !info.bound {
					rt.fail("member .%s of an unbound callable reference ::%s", name, info.name)
				}
				return w(ktRefRead(info.recv, info.name))
			}
			// ::topLevelFun is the closure itself, and KFunction.name is the name the
			// reference site spelled - recorded by js_ktnameval, since a closure
			// carries none of its own.
			if isCallable(o) && name == "name" {
				if n, has := ktFuncNames[o]; has {
					return rt.wrapStr(n)
				}
			}
			// A KClass. simpleName and qualifiedName agree for a top-level class in
			// the default package, and .java is the same handle again.
			if co, isCls := ktIsClassLit(o); isCls {
				switch name {
				case "simpleName", "qualifiedName", "name":
					return w(co.props["kname"])
				case "java", "javaClass", "javaObjectType":
					return w(co)
				}
				rt.fail("member .%s of a class literal", name)
			}
			// `Col.entries` is a PROPERTY (an EnumEntries, which is a List) where
			// values() is a function, so it is read here too.
			if name == "entries" {
				if v, handled := rt.ktEnumMethod(o, name, nil); handled {
					return w(v)
				}
			}
			if mo, isObj := o.(*jsObject); isObj {
				if ktIsSb(mo) {
					if v, handled := rt.ktSbMethod(mo, name, nil); handled {
						return w(v)
					}
				}
				// kotlin.Result's isSuccess / isFailure and lazy's value are
				// PROPERTIES, so they are read here rather than through js_ktsmcall.
				if ro, isRes := ktIsResult(mo); isRes {
					if v, handled := rt.ktResultMethod(ro, name, nil); handled {
						return w(v)
					}
				}
				if lz, _ := mo.props["__lazy"].(bool); lz {
					switch name {
					case "value":
						return w(rt.ktLazyValue(mo))
					case "isInitialized":
						return w(mo.props["done"])
					}
				}
				if j, isJob := ktIsJob(mo); isJob {
					if v, handled := rt.ktJobMethod(j, name); handled {
						return w(v)
					}
				}
				// A SET has no `length`, so `s.size` - which the emitter spells
				// as `.length` - read undefined and printed "kotlin.Unit".
				if ktIsSet(mo) {
					switch name {
					case "length", "size":
						keys, _ := ktSetParts(mo)
						return w(float64(len(keys.elems)))
					}
				}
				// Map.keys is a SET (java.util.Map.keySet) and Map.entries a set of
				// entries. Read as PROPERTIES they used to fall through to js_get,
				// which handed back the raw `keys` array behind the handle - so
				// `m.keys == setOf("a")` was false in the compiled half only.
				if _, _, isDict := dictParts(mo); isDict {
					switch name {
					case "keys", "values", "entries":
						if v, handled := rt.ktMapMethod(mo, name, nil); handled {
							return w(v)
						}
					case "length", "size":
						if v, handled := rt.ktMapMethod(mo, "size", nil); handled {
							return w(v)
						}
					}
				}
				// A dotted name on a MAP that is neither one of Kotlin's map
				// properties nor a present key. Kotlin refuses `m.bogus` outright, so
				// no correct program reaches this; what matters is that the two halves
				// answer the SAME thing, and kotlin-interpreter.abnf's kGetField misses
				// to undefined (which renders as kotlin.Unit). jsNull here made the
				// halves print `null` and `kotlin.Unit` for the same program.
				if keys, _, isDict := dictParts(mo); isDict && !ktRecvProp(name) &&
					rt.ktMapFind(keys, u(a[1])) < 0 {
					return w(jsUndef)
				}
			}
			var r uint64
			// js_rxktget is js_get plus Regex.pattern and the MatchResult properties,
			// so it is the tail whenever the regex layer is registered.
			if next := m["js_rxktget"]; next != nil {
				r = next(a)
			} else {
				r = m["js_get"](a)
			}
			// `.size` is Kotlin's collection length, and the emitter used to rewrite it
			// to `.length` for EVERY receiver - so a user class declaring its own `size`
			// property read undefined and printed "kotlin.Unit", where the interpreter
			// half answered the property. The rewrite now happens HERE, only when the
			// receiver has no `size` of its own, which is the same fast-path-versus-
			// dispatcher trap recorded in docs/abnf-dialect-gotchas.md.
			if name == "size" && isUndefOrNull(u(r)) {
				lr := m["js_get"]([]uint64{a[0], rt.wrapStr("length")})
				if !isUndefOrNull(u(lr)) {
					return lr
				}
			}
			// KOTLIN'S INDEX PROPERTIES, on the receivers the stdlib declares them for,
			// and read ONLY WHEN THE ORDINARY MEMBER READ MISSED - the same
			// fast-path-versus-dispatcher shape as the `.size` rewrite above it, and for
			// two reasons: a receiver that declares its own `first` keeps it, and every
			// field read that HITS pays nothing for these arms at all (js_ktfget is the
			// hot path of the compiler half's member reads).
			//
			//   val Collection<*>.indices: IntRange      (kotlin.collections)
			//   val CharSequence.indices: IntRange       (kotlin.text)
			//   val <T> List<T>.lastIndex: Int           = size - 1
			//   val CharSequence.lastIndex: Int          = length - 1
			//
			// Both read UNDEFINED here, so `for (i in vs.indices)` handed
			// js_ktiter an undefined and the loop died with "member 'length' of
			// undefined" while the interpreter half ran it - todo.md 1.8. `.size` was
			// never affected because of the `.length` rewrite below it.
			//
			// `indices` answers a real IntRange, which is what makes the for-loop,
			// `i in vs.indices` and `vs.indices.reversed()` all work with the
			// machinery that is already here.
			//
			// first / last / start / endInclusive / step are the PROGRESSION's own
			// properties, read off its bounds - which is how an EMPTY progression
			// answers all five (todo.md 1.8). Kotlin declares them on
			// IntProgression / ClosedRange only; on a List `first`/`last` are
			// member FUNCTIONS and need parentheses, so a valid program reaches
			// this arm with a range and nothing else.
			if rg, isRg := ktRangeOf(o); isRg && isUndefOrNull(u(r)) {
				switch name {
				case "first":
					return w(ktRangeEl(rg, rg.from))
				// IntProgression.last is the last ELEMENT, not the bound it was
				// built from: (1..10 step 4).last is 9.
				case "last":
					return w(ktRangeEl(rg, ktRangeLastEl(rg)))
				// ClosedRange.start / ClosedRange.endInclusive are the BOUNDS -
				// they coincide with first/last for every plain a..b, and a
				// stepped progression declares neither in Kotlin.
				case "start":
					return w(ktRangeEl(rg, rg.from))
				case "endInclusive":
					return w(ktRangeEl(rg, rg.to))
				case "step":
					return w(ktRangeSigned(rg))
				// `.size` on a progression is not Kotlin (IntRange is an Iterable,
				// not a Collection), but the interpreter half answers the element
				// count, so this one does too rather than diverge.
				case "size":
					return w(float64(len(ktRangeList(rg))))
				case "indices":
					return w(ktMakeRange(ktRange{from: 0, to: float64(len(ktRangeList(rg)) - 1), st: 1}))
				case "lastIndex":
					return w(float64(len(ktRangeList(rg)) - 1))
				}
			}
			if arr, isArr := o.(*jsArray); isArr && isUndefOrNull(u(r)) {
				n := len(arr.elems)
				switch name {
				case "indices":
					return w(ktMakeRange(ktRange{from: 0, to: float64(n - 1), st: 1}))
				case "lastIndex":
					return w(float64(n - 1))
				}
			}
			// A StringBuilder IS a CharSequence, so the same two properties apply to
			// it. Only `length` was answered, in both halves alike.
			if sb, isObj := o.(*jsObject); isObj && ktIsSb(sb) && isUndefOrNull(u(r)) {
				if txt, isStr := sb.props["s"].(string); isStr {
					switch name {
					case "indices":
						return w(ktMakeRange(ktRange{from: 0, to: float64(rt.strLen(txt) - 1), st: 1}))
					case "lastIndex":
						return w(float64(rt.strLen(txt) - 1))
					}
				}
			}
			if s, isStr := o.(string); isStr && isUndefOrNull(u(r)) {
				switch name {
				case "indices":
					return w(ktMakeRange(ktRange{from: 0, to: float64(rt.strLen(s) - 1), st: 1}))
				case "lastIndex":
					return w(float64(rt.strLen(s) - 1))
				}
			}
			// A `lateinit var` that is still null has NOT been assigned, and Kotlin
			// throws rather than answering null - see ktLateinitCheck.
			if mo, isObj := o.(*jsObject); isObj {
				rt.ktLateinitCheck(mo, name, u(r))
			}
			return r
		}

		// js_ktindex is the same rule for the INDEXED read: `m[9]` on a map that has no
		// key 9 is null in Kotlin (Map.get answers null), where the shared js_kindex
		// answers undefined - which printed "kotlin.Unit" and made `m[9] == null` false
		// in the compiler while the interpreter said true. Everything else falls through
		// to js_rxktindex (js_kindex plus the MatchResult group readers).
		m["js_ktindex"] = func(a []uint64) uint64 {
			// A progression has no `get`, but elementAt and every list builtin
			// reaches it through the materialized elements, so the indexed read
			// answers from them too.
			if rg, isRg := ktRangeOf(u(a[0])); isRg {
				es := ktRangeList(rg)
				i := int(rt.toNumber(u(a[1])))
				if i < 0 || i >= len(es) {
					return w(jsUndef)
				}
				return w(es[i])
			}
			if mo, isObj := u(a[0]).(*jsObject); isObj {
				if keys, _, isDict := dictParts(mo); isDict && rt.ktMapFind(keys, u(a[1])) < 0 {
					return w(jsNull)
				}
			}
			if next := m["js_rxktindex"]; next != nil {
				return next(a)
			}
			return m["js_kindex"](a)
		}

		// js_ktset is js_set plus the map handle: `mm[2] = "b"` is Map.put, which the
		// shared setMember cannot answer (it stored a plain PROPERTY named "2", so the
		// map kept its old size and the write was invisible to every map operation).
		baseSet := m["js_set"]
		m["js_ktset"] = func(a []uint64) uint64 {
			if mo, isObj := u(a[0]).(*jsObject); isObj {
				if _, _, isDict := dictParts(mo); isDict {
					rt.ktMapPut(mo, u(a[1]), u(a[2]))
					return 0
				}
				// A field WRITE adopts the property's DECLARED type: `var y: Long = 1`
				// followed by `w.y = 1` stores a Long. __ptypes is the width table the
				// class emitter installs; kFieldTy in kotlin-interpreter.abnf is the twin.
				if ty := ktFieldTy(mo, rt.toString(u(a[1]))); ty != "" {
					if v := rt.ktAdoptTy(u(a[2]), ty); v != nil {
						return baseSet([]uint64{a[0], a[1], w(v)})
					}
				}
			}
			return baseSet(a)
		}
	})
	// js_ktsmcall reads its argument array in position 2, exactly like js_mcall and
	// js_rxktmcall: the handle must arrive as the array itself, not as a value.
	jsThroughArgs["js_ktsmcall"] = 1 << 2
	// js_ktget reads a SCOPE in position 0, exactly like js_kget.
	jsThroughArgs["js_ktget"] = 1 << 0
}

// ktStrMethod is the Go twin of the String branch of `mcall` in
// kotlin-interpreter.abnf. It answers (value, true) for the names the shared
// js_mcall does NOT carry and (nil, false) for everything else, so a name js_mcall
// already implements keeps exactly the behaviour it had.
func (rt *jsrt) ktStrMethod(s, name string, args []interface{}) (interface{}, bool) {
	switch name {
	case "isNotEmpty":
		return len(s) > 0, true
	case "get":
		return jsChar{code: int32(rt.strCodeAt(s, jsToInt(rt.toNumber(argAt(args, 0)))))}, true
	case "plus":
		return s + rt.ktStr2(argAt(args, 0)), true
	case "compareTo":
		o := rt.ktStr2(argAt(args, 0))
		if s < o {
			return float64(-1), true
		}
		if s > o {
			return float64(1), true
		}
		return float64(0), true
	case "repeat":
		n := jsToInt(rt.toNumber(argAt(args, 0)))
		out := ""
		for i := 0; i < n; i++ {
			out += s
		}
		return out, true
	// Kotlin 1.5 renamed toUpperCase/toLowerCase to uppercase/lowercase; the old
	// spellings are deprecated but still resolve, so both halves accept all four.
	case "uppercase", "toUpperCase":
		return strings.ToUpper(s), true
	case "lowercase", "toLowerCase":
		return strings.ToLower(s), true
	case "contains":
		return rt.strIndexOf(s, rt.ktStr2(argAt(args, 0))) >= 0, true
	case "hashCode":
		return float64(ktStrHash(rt, s)), true
	}
	return rt.ktStrMethod2(s, name, args)
}

// ktStrMethod2 is the rest of the kotlin.text String surface: the CHAR-valued
// members (Kotlin's String is a sequence of Char, so first/last/get/single answer
// a Char and never a one-character String), the trimming and padding family, and
// the char-sequence operations that make a String behave like a collection.
//
// charAt is Java's spelling and does not exist in Kotlin at all - it is get / the
// indexing operator - but both halves have carried it as a builtin for a long
// time, so it stays, now answering a Char like its two Kotlin spellings instead of
// a String. That was the last place where `s.charAt(0) == 'a'` could be false.
func (rt *jsrt) ktStrMethod2(s, name string, args []interface{}) (interface{}, bool) {
	n := rt.strLen(s)
	chars := func() []interface{} { return rt.ktChars(s) }
	switch name {
	case "charAt":
		i := jsToInt(rt.toNumber(argAt(args, 0)))
		if i < 0 || i >= n {
			rt.fail("charAt(%d) out of range for %s", i, s)
		}
		return jsChar{code: int32(rt.strCodeAt(s, i))}, true
	case "first":
		if n == 0 {
			rt.fail("first() on an empty string")
		}
		return jsChar{code: int32(rt.strCodeAt(s, 0))}, true
	case "last":
		if n == 0 {
			rt.fail("last() on an empty string")
		}
		return jsChar{code: int32(rt.strCodeAt(s, n-1))}, true
	case "single":
		// single(predicate) filters first - the predicate used to be IGNORED, so
		// `"hello".single { it == 'h' }` aborted with "single() on a string of length
		// 5" in both halves instead of answering 'h'.
		if len(args) > 0 && isCallable(args[0]) {
			return rt.ktSeqMethod(&jsArray{elems: chars()}, name, args)
		}
		if n != 1 {
			rt.fail("single() on a string of length %d", n)
		}
		return jsChar{code: int32(rt.strCodeAt(s, 0))}, true
	case "firstOrNull":
		if n == 0 {
			return jsNull, true
		}
		return jsChar{code: int32(rt.strCodeAt(s, 0))}, true
	case "lastOrNull":
		if n == 0 {
			return jsNull, true
		}
		return jsChar{code: int32(rt.strCodeAt(s, n-1))}, true
	case "toCharArray", "toList":
		return &jsArray{elems: chars()}, true
	case "trim":
		return strings.TrimSpace(s), true
	case "trimStart":
		return strings.TrimLeft(s, " \t\r\n\f"), true
	case "trimEnd":
		return strings.TrimRight(s, " \t\r\n\f"), true
	case "startsWith":
		return strings.HasPrefix(s, rt.ktStrArg(argAt(args, 0))), true
	case "endsWith":
		return strings.HasSuffix(s, rt.ktStrArg(argAt(args, 0))), true
	case "padStart", "padEnd":
		want := jsToInt(rt.toNumber(argAt(args, 0)))
		pad := " "
		if len(args) > 1 {
			pad = rt.ktStrArg(args[1])
		}
		out := s
		for rt.strLen(out) < want {
			if name == "padStart" {
				out = pad + out
			} else {
				out = out + pad
			}
		}
		return out, true
	case "reversed":
		cs := chars()
		out := ""
		for i := len(cs) - 1; i >= 0; i-- {
			out += string(rune(cs[i].(jsChar).code))
		}
		return out, true
	case "lines":
		parts := strings.Split(s, "\n")
		out := &jsArray{}
		for _, p := range parts {
			out.elems = append(out.elems, strings.TrimSuffix(p, "\r"))
		}
		return out, true
	case "split":
		if len(args) == 0 {
			return nil, false
		}
		sep := rt.ktStrArg(args[0])
		if sep == "" {
			return nil, false
		}
		out := &jsArray{}
		for _, p := range strings.Split(s, sep) {
			out.elems = append(out.elems, p)
		}
		return out, true
	case "drop", "take", "dropLast", "takeLast":
		k := jsToInt(rt.toNumber(argAt(args, 0)))
		if k < 0 {
			k = 0
		}
		if k > n {
			k = n
		}
		switch name {
		case "drop":
			return rt.strRange(s, k, n), true
		case "take":
			return rt.strRange(s, 0, k), true
		case "dropLast":
			return rt.strRange(s, 0, n-k), true
		}
		return rt.strRange(s, n-k, n), true
	case "indexOf":
		return float64(rt.strIndexOf(s, rt.ktStrArg(argAt(args, 0)))), true
	case "lastIndexOf":
		return float64(strings.LastIndex(s, rt.ktStrArg(argAt(args, 0)))), true
	case "contains":
		return rt.strIndexOf(s, rt.ktStrArg(argAt(args, 0))) >= 0, true
	case "toInt", "toLong", "toDouble", "toIntOrNull":
		return ktParseNum(rt, s, name)
	case "format":
		// "%.2f".format(x): the receiver is the format string, as kotlin.text.format
		// declares it (String.Companion.format has the arguments the other way round).
		return ktFormat(rt, s, args), true

	// ----- More of kotlin.text, added in one batch: every one of these aborted the
	// run in BOTH halves with "unknown String method". The twin is the `typeof target
	// == "string"` branch of mcall in kotlin-interpreter.abnf.
	// `replace` with a Regex receiver never reaches here - js_ktsmcall routes a
	// pattern argument to the Regex dispatcher first.
	case "replace", "replaceFirst":
		old, nw := rt.ktStrArg(argAt(args, 0)), rt.ktStrArg(argAt(args, 1))
		if old == "" {
			return s, true
		}
		if name == "replaceFirst" {
			return strings.Replace(s, old, nw, 1), true
		}
		return strings.ReplaceAll(s, old, nw), true
	case "isBlank", "isNotBlank":
		return (strings.TrimSpace(s) == "") == (name == "isBlank"), true
	case "removePrefix":
		return strings.TrimPrefix(s, rt.ktStrArg(argAt(args, 0))), true
	case "removeSuffix":
		return strings.TrimSuffix(s, rt.ktStrArg(argAt(args, 0))), true
	case "removeSurrounding":
		pre := rt.ktStrArg(argAt(args, 0))
		post := pre
		if len(args) > 1 {
			post = rt.ktStrArg(argAt(args, 1))
		}
		if len(s) >= len(pre)+len(post) && strings.HasPrefix(s, pre) && strings.HasSuffix(s, post) {
			return s[len(pre) : len(s)-len(post)], true
		}
		return s, true
	case "substringBefore", "substringAfter", "substringBeforeLast", "substringAfterLast":
		delim := rt.ktStrArg(argAt(args, 0))
		miss := s
		if len(args) > 1 {
			miss = rt.ktStrArg(argAt(args, 1))
		}
		i := strings.Index(s, delim)
		if name == "substringBeforeLast" || name == "substringAfterLast" {
			i = strings.LastIndex(s, delim)
		}
		if i < 0 || delim == "" {
			return miss, true
		}
		if name == "substringBefore" || name == "substringBeforeLast" {
			return s[:i], true
		}
		return s[i+len(delim):], true
	case "ifEmpty":
		if s == "" {
			return rt.ktCall(argAt(args, 0)), true
		}
		return s, true
	case "ifBlank":
		if strings.TrimSpace(s) == "" {
			return rt.ktCall(argAt(args, 0)), true
		}
		return s, true
	case "capitalize", "replaceFirstChar":
		if s == "" {
			return s, true
		}
		rs := []rune(s)
		if name == "replaceFirstChar" && isCallable(argAt(args, 0)) {
			head := rt.ktCall(argAt(args, 0), jsChar{code: int32(rs[0])})
			return rt.ktStr2(head) + string(rs[1:]), true
		}
		return string(ktUpper(rs[0])) + string(rs[1:]), true
	case "decapitalize":
		if s == "" {
			return s, true
		}
		rs := []rune(s)
		return string(ktLower(rs[0])) + string(rs[1:]), true
	case "subSequence":
		return rt.strRange(s, jsToInt(rt.toNumber(argAt(args, 0))), jsToInt(rt.toNumber(argAt(args, 1)))), true
	case "trimIndent":
		return ktTrimIndent(s), true
	}
	// ----- the STRING family of _Strings.kt -----
	// kotlin.text generates the collection operators for CharSequence from the same
	// templates as kotlin.collections, but the ones that SELECT characters answer a
	// String (`public inline fun String.filter(predicate: (Char) -> Boolean):
	// String`), not a List<Char>; chunked and windowed answer List<String>;
	// partition answers Pair<String, String>. Everything that TRANSFORMS - map,
	// flatMap, groupBy, zip, sorted, distinct, withIndex - answers a List in Kotlin
	// too and keeps falling through below. All of these printed "[a, b, d]" in every
	// engine. todo.md 1.5.
	switch name {
	case "substring", "slice":
		// String.substring(IntRange) is substring(range.start, range.endInclusive+1)
		// and String.slice(IntRange) is substring(range); the range argument used to
		// reach the shared js_mcall's toNumber, which reads an object as NaN, so
		// `"abcdef".substring(1..3)` answered the whole string. slice over an
		// Iterable<Int> picks the chars at those indices and answers a String too.
		if len(args) == 1 {
			if rg, isRg := ktRangeOf(args[0]); isRg {
				if ktRangeEmpty(rg) {
					return "", true
				}
				return rt.strRange(s, int(rg.from), int(rg.to)+1), true
			}
			if name == "slice" {
				v, _ := rt.ktSeqMethod(&jsArray{elems: chars()}, name, args)
				return ktCharsStr(v), true
			}
		}
	case "filter", "filterNot", "filterIndexed", "takeWhile", "dropWhile", "onEach":
		v, handled := rt.ktSeqMethod(&jsArray{elems: chars()}, name, args)
		if handled {
			return ktCharsStr(v), true
		}
	case "partition":
		v, handled := rt.ktSeqMethod(&jsArray{elems: chars()}, name, args)
		if handled {
			po := v.(*jsObject)
			p := newJSObject()
			p.set("first", ktCharsStr(po.props["first"]))
			p.set("second", ktCharsStr(po.props["second"]))
			return p, true
		}
	case "chunked", "windowed":
		// The transform of the String overloads receives a CharSequence, so it runs
		// on the joined piece rather than on the char list.
		var xf interface{}
		rest := []interface{}{}
		for _, a := range args {
			if isCallable(a) {
				xf = a
			} else {
				rest = append(rest, a)
			}
		}
		v, handled := rt.ktSeqMethod(&jsArray{elems: chars()}, name, rest)
		if handled {
			out := []interface{}{}
			for _, piece := range v.(*jsArray).elems {
				txt := ktCharsStr(piece)
				if xf != nil {
					out = append(out, rt.ktCall(xf, txt))
				} else {
					out = append(out, txt)
				}
			}
			return &jsArray{elems: out}, true
		}
	}
	// Everything a String shares with a List of Char - map/filter/forEach/any/all/
	// count/fold/joinToString/... - goes through the collection surface.
	if v, handled := rt.ktSeqMethod(&jsArray{elems: chars()}, name, args); handled {
		return v, true
	}
	return nil, false
}

// A list of Char boxes back to the text it came from.
func ktCharsStr(v interface{}) string {
	a, isArr := v.(*jsArray)
	if !isArr {
		return ""
	}
	out := ""
	for _, e := range a.elems {
		if c, isCh := e.(jsChar); isCh {
			out += string(rune(c.code))
		}
	}
	return out
}

// ktTrimIndent is kotlin.text.trimIndent: drop a leading and a trailing BLANK
// line, then remove the common indent of every non-blank line. The twin is
// kTrimIndent in kotlin-interpreter.abnf.
func ktTrimIndent(s string) string {
	lines := strings.Split(s, "\n")
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	ind := -1
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		k := 0
		for k < len(l) && (l[k] == ' ' || l[k] == '\t') {
			k++
		}
		if ind < 0 || k < ind {
			ind = k
		}
	}
	if ind < 0 {
		ind = 0
	}
	out := make([]string, len(lines))
	for i, l := range lines {
		if len(l) >= ind {
			out[i] = l[ind:]
		} else {
			out[i] = strings.TrimLeft(l, " \t")
		}
	}
	return strings.Join(out, "\n")
}

// ktParseNum is toInt() / toLong() / toDouble() on a String. Kotlin throws a
// NumberFormatException for text that is not a number, and java.lang's message is
// the one both halves report.
func ktParseNum(rt *jsrt, s, name string) (interface{}, bool) {
	t := strings.TrimSpace(s)
	if name == "toDouble" {
		var f float64
		if _, err := fmt.Sscanf(t, "%g", &f); err != nil || t == "" {
			rt.ktRaise("NumberFormatException", fmt.Sprintf("For input string: \"%s\"", s))
		}
		return jsJFlo{f: f}, true
	}
	neg := false
	body := t
	if strings.HasPrefix(body, "-") {
		neg, body = true, body[1:]
	} else if strings.HasPrefix(body, "+") {
		body = body[1:]
	}
	if body == "" {
		if name == "toIntOrNull" {
			return jsNull, true
		}
		rt.ktRaise("NumberFormatException", fmt.Sprintf("For input string: \"%s\"", s))
	}
	var acc int64
	for _, r := range body {
		if r < '0' || r > '9' {
			if name == "toIntOrNull" {
				return jsNull, true
			}
			rt.ktRaise("NumberFormatException", fmt.Sprintf("For input string: \"%s\"", s))
		}
		acc = acc*10 + int64(r-'0')
	}
	if neg {
		acc = -acc
	}
	if name == "toLong" {
		return ktNorm(acc, 64, false), true
	}
	return ktNorm(acc, 32, false), true
}

// ktFormat implements the subset of java.util.Formatter that a test program
// actually writes: %s, %d, %f with an optional precision, %x, %%, and a width.
func ktFormat(rt *jsrt, f string, args []interface{}) string {
	out := ""
	ai := 0
	rs := []rune(f)
	for i := 0; i < len(rs); i++ {
		if rs[i] != '%' {
			out += string(rs[i])
			continue
		}
		if i+1 < len(rs) && rs[i+1] == '%' {
			out += "%"
			i++
			continue
		}
		spec := "%"
		i++
		for i < len(rs) && (rs[i] == '-' || rs[i] == '+' || rs[i] == '0' || rs[i] == ' ' ||
			rs[i] == '.' || rs[i] == ',' || (rs[i] >= '0' && rs[i] <= '9')) {
			if rs[i] != ',' {
				spec += string(rs[i])
			}
			i++
		}
		if i >= len(rs) {
			break
		}
		verb := rs[i]
		var a interface{} = jsUndef
		if ai < len(args) {
			a = args[ai]
		}
		ai++
		switch verb {
		case 's':
			out += fmt.Sprintf(spec+"s", rt.ktpRender(a, 0))
		case 'd':
			out += fmt.Sprintf(spec+"d", giVal(rt, a))
		case 'f':
			// Java's %f rounds HALF UP, Go's Sprintf rounds half to even, so
			// String.format("%5.1f", 1.25) is "  1.3" on java (JDK 24) and "  1.2"
			// through Sprintf. ktFixed is the Go twin of kFixed in
			// kotlin-interpreter.abnf and gives both halves Java's answer.
			prec := 6
			if d := strings.Index(spec, "."); d >= 0 {
				prec = 0
				for _, r := range spec[d+1:] {
					if r < '0' || r > '9' {
						break
					}
					prec = prec*10 + int(r-'0')
				}
				spec = spec[:d]
			}
			out += fmt.Sprintf(spec+"s", ktFixed(giFloat(rt, a), prec))
		case 'e', 'g':
			out += fmt.Sprintf(spec+string(verb), giFloat(rt, a))
		case 'x', 'X', 'o':
			out += fmt.Sprintf(spec+string(verb), giVal(rt, a))
		case 'c':
			if c, ok := a.(jsChar); ok {
				out += string(rune(c.code))
			} else {
				out += rt.ktpRender(a, 0)
			}
		case 'b':
			out += fmt.Sprintf(spec+"t", rt.truthy(a))
		default:
			out += string(verb)
			ai--
		}
	}
	return out
}

// ktStr2 is kstr2: a string is itself, everything else renders through ktpRender.
func (rt *jsrt) ktStr2(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return rt.ktpRender(v, 0)
}

// ktStrHash is java.lang.String.hashCode over UTF-16 CODE UNITS, which is what
// strHash in kotlin-interpreter.abnf computes (the script string type is UTF-16).
func ktStrHash(rt *jsrt, s string) int32 {
	var h int32
	n := rt.strLen(s)
	for i := 0; i < n; i++ {
		h = 31*h + int32(rt.strCodeAt(s, i))
	}
	return h
}

// ktpLine is the text one println/print call writes. Kotlin's println has only
// single-argument overloads plus the no-argument one (which writes just the line
// terminator), so this is either "" or the one rendered argument.
func ktpLine(rt *jsrt, args []interface{}) string {
	out := ""
	for _, a := range args {
		out += rt.ktpRender(a, 0)
	}
	return out
}

// ktpRender is Kotlin's toString for a runtime value. `depth` only guards a cyclic
// object graph - an enum entry's class descriptor holds the entry, which holds the
// descriptor - so the renderer cannot recurse forever the way Go's %v did.
func (rt *jsrt) ktpRender(v interface{}, depth int) string {
	if depth > 32 {
		return "..."
	}
	switch t := v.(type) {
	case jsUndefT:
		return "kotlin.Unit" // Unit is a VALUE in Kotlin, and it prints as its name.
	case jsNullT:
		return "null"
	case jsChar:
		return string(rune(t.code))
	case jsGInt:
		// A sized integer prints its exact digits, and an UNSIGNED one prints its
		// unsigned reading: ULong.MAX_VALUE is 18446744073709551615, not -1.
		return giStr(t)
	case jsJFlo:
		return jvmFloText(t) // 1.5 / 1.0E20 / Infinity - see jsrtjvm.go.
	case string:
		// Neither the top level nor a collection element quotes a String:
		// java.util.AbstractCollection.toString calls String.valueOf, so
		// `println(listOf("x", "y"))` is `[x, y]`.
		return t
	case *jsArray:
		parts := make([]string, len(t.elems))
		for i, e := range t.elems {
			parts[i] = rt.ktpRender(e, depth+1)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case *hostFunc:
		// A callable reference. Real kotlin-reflect renders "val Box.v: kotlin.Int";
		// without the reflection library on the class path it prints "property v
		// (Kotlin reflection is not available)". Neither is reproducible here, so
		// both halves agree on the short form - a deliberate, recorded divergence.
		if info, isRef := ktRefOf(t); isRef {
			return "reference " + info.name
		}
	case *jsObject:
		if s, ok := rt.ktpObj(t, depth); ok {
			return s
		}
	}
	return rt.toString(v)
}

// ktpObj renders the object shapes the Kotlin compiler grammar builds. It reports
// false for a shape it does not know, which falls back to the generic ToString.
func (rt *jsrt) ktpObj(o *jsObject, depth int) (string, bool) {
	// A PROGRESSION prints its BOUNDS - "1..5", "10 downTo 1 step 1",
	// "1..9 step 4" - which is IntRange.toString / IntProgression.toString. It
	// printed the materialized element list before, in every engine (todo.md
	// 1.10's `mr.range.toString()`, which read "[object Object]" here).
	if rg, isRg := ktRangeOf(o); isRg {
		return rt.ktRangeStr(rg), true
	}
	// A KClass. java.lang.Class.toString() is "class Box" too, which is why .java
	// answers the same handle. The twin is the __kclass arm of kstr.
	if co, isCls := ktIsClassLit(o); isCls {
		return "class " + rt.ktpRender(co.props["kname"], depth+1), true
	}
	// A Regex renders as its pattern and a MatchResult as the matched text, which is
	// what their toString answers (see abnf/jsrtregexkt.go); neither carries a
	// __class, so without this they fell through to "[object Object]".
	if ktRxIsRegex(o) || ktRxIsMatch(o) {
		if v, handled := rt.ktRxMethod(o, "toString", nil); handled {
			return rt.ktpRender(v, depth+1), true
		}
	}
	// A StringBuilder renders as its accumulated text (java.lang.StringBuilder's
	// toString), the same answer kstr gives it in kotlin-interpreter.abnf.
	if ktIsSb(o) {
		return ktSbText(o), true
	}
	// A MAP renders as java.util.AbstractMap.toString does: {a=1, b=2}. This is the
	// rendering the header note said had nothing to render; it does now, because
	// groupBy/associate/toMap build the shared {__dict, keys, vals} handle.
	if keys, vals, isDict := dictParts(o); isDict {
		parts := make([]string, len(keys.elems))
		for i := range keys.elems {
			parts[i] = rt.ktpRender(keys.elems[i], depth+1) + "=" + rt.ktpRender(vals.elems[i], depth+1)
		}
		return "{" + strings.Join(parts, ", ") + "}", true
	}
	// A SET renders like a List - java.util.AbstractCollection.toString again - in
	// its iteration order: [1, 2].
	if keys, isSet := ktSetParts(o); isSet {
		return rt.ktpRender(keys, depth), true
	}
	// A Map.Entry renders as a=1 (java.util.AbstractMap's entry toString), NOT as the
	// Pair it also destructures as - so `println(m.entries)` is [a=1, b=2].
	if ktIsEntry(o) {
		return rt.ktpRender(o.props["key"], depth+1) + "=" + rt.ktpRender(o.props["value"], depth+1), true
	}
	// An IndexedValue is a data class, so its toString is the generated one.
	if iv, _ := o.props["__ktiv"].(bool); iv {
		return "IndexedValue(index=" + rt.ktpRender(o.props["index"], depth+1) +
			", value=" + rt.ktpRender(o.props["value"], depth+1) + ")", true
	}
	// A Pair / Triple renders as (a, b) / (a, b, c) - kotlin.Pair.toString.
	if po, isPair := ktIsPair(o); isPair {
		out := "(" + rt.ktpRender(po.props["first"], depth+1) + ", " + rt.ktpRender(po.props["second"], depth+1)
		if third, has := po.props["third"]; has {
			out += ", " + rt.ktpRender(third, depth+1)
		}
		return out + ")", true
	}
	cls, _ := o.props["__class"].(*jsObject)
	if cls == nil {
		// A class descriptor printed on its own (`println(E)`), which Go's %v
		// dumped as a map of pointers.
		if n, ok := ktpName(o); ok {
			return n, true
		}
		return "", false
	}
	// A user-written or synthesized toString() wins over every built-in rendering,
	// exactly as it does in Kotlin: a data class renders through the toString the
	// compiler generated for it (D(a=1, b=q)), an `override fun toString()` through
	// its own body. The walk is the __super chain memberCall follows.
	if mth, ok := ktpFindMember(o, "toString"); ok {
		return rt.ktpRender(rt.call(mth, jsUndef, []interface{}{o}), depth+1), true
	}
	// An enum entry prints its NAME (java.lang.Enum.toString); both halves store it
	// on the entry as `name`.
	if ktpIsEnum(cls) {
		if n, ok := o.props["name"].(string); ok {
			return n, true
		}
	}
	name, _ := cls.props["__name"].(string)
	if name == "" {
		return "", false
	}
	return name + "@1", true
}

// ktpName is the __name of a class descriptor.
func ktpName(o *jsObject) (string, bool) {
	if _, ok := o.props["__isclass"]; !ok {
		return "", false
	}
	n, ok := o.props["__name"].(string)
	return n, ok && n != ""
}

// ktpIsEnum reports whether a descriptor (or one of its supers - an enum entry
// with a body is an anonymous subclass) belongs to an enum class.
func ktpIsEnum(cls *jsObject) bool {
	for i := 0; cls != nil && i < 32; i++ {
		if e, ok := cls.props["__isenum"]; ok && e == true {
			return true
		}
		cls, _ = cls.props["__super"].(*jsObject)
	}
	return false
}

// ktpFindMember walks the __class / __super chain for a callable member, the same
// walk memberCall does for a js_mcall and clsFind does in kotlin-interpreter.abnf.
func ktpFindMember(o *jsObject, name string) (interface{}, bool) {
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

// ============================================================================
// THE MEMBER SURFACE: Char, String, collections and the scope functions
//
// Everything below is reached from js_ktsmcall, which is Kotlin's method-call
// entry point in the compiler grammar, so a name added here works in BOTH halves
// at once - kotlin-interpreter.abnf's own `mcall` carries the twin bodies and
// ./test.sh --cross diffs the two.
//
// What CANNOT be added here is a global NAME: `mapOf`, `Pair`, `with`,
// `StringBuilder` and friends are declared by each grammar's own builtin block
// (js_scope_decl in kotlin-to-llvm-ir.abnf), not by the runtime, so a builder has
// to be ported into the compiler grammar by hand. The map VALUE and the Pair
// VALUE are shared shapes and their methods live here, so once the builder is
// registered on the compiler side the whole surface comes with it.
//
// The map shape is the runtime's own dict handle {__dict, keys, vals} (see
// dictParts in abnf/jsrt.go), the same one Go maps and Python dicts use, so
// js_pyset / js_pyget / iteration already understand it.
//
// No kotlinc on this machine: every semantic below is kotlin.text / kotlin.collections
// as documented, and the ones Kotlin shares with java.lang were checked against the
// equivalent Java program under `java` (JDK 24).

// ktMakeMap builds an empty map handle.
func ktMakeMap() *jsObject {
	o := newJSObject()
	o.set("__dict", true)
	o.set("keys", &jsArray{})
	o.set("vals", &jsArray{})
	return o
}

// ----------------------------------------------------------------------------
// SETS
//
// A SET is its own shape, a jsObject tagged {__set: true, keys: [...]} - the map
// handle without the parallel `vals` array, which is exactly what
// java.util.LinkedHashSet is. Before this every set constructor was an alias of
// listOf in BOTH halves, so setOf(1, 1, 2) did not deduplicate, setOf(2, 1) ==
// setOf(1, 2) was FALSE (java: true) and listOf(1) == setOf(1) was TRUE (java:
// false). Three properties come with the shape at once: uniqueness on insertion,
// order-INDEPENDENT equality, and java.util.AbstractSet's hashCode, the plain SUM
// of the element hashes (java: Set.of(1, 2).hashCode() is 3, where the list fold
// answers 994).
//
// The tag is deliberately NOT __dict: dictParts is shared with Go maps and Python
// dicts (abnf/jsrt.go) and everything map-shaped would otherwise claim a set.
//
// The twin is kMakeSet / kSetMethod in languages/kotlin-interpreter.abnf.
// ktHasMethod reports whether v is a class instance whose descriptor chain declares
// a callable member `name`. The twin of lookupMethod's delegate use in
// kotlin-interpreter.abnf, and the test js_ktdcheck applies to a `by` expression.
// ktChainDeclares reports whether a class descriptor chain declares `name` - the
// guard the Any-member fallbacks owe a user class that overrides toString() or
// hashCode().
func ktChainDeclares(v interface{}, name string) bool {
	o, isObj := v.(*jsObject)
	if !isObj {
		return false
	}
	if _, hasCls := o.props["__class"]; !hasCls {
		return false
	}
	return ktClassChainHas(o, name)
}

func ktHasMethod(v interface{}, name string) bool {
	o, isObj := v.(*jsObject)
	if !isObj {
		return false
	}
	for cls := o.props["__class"]; cls != nil; {
		co, isCls := cls.(*jsObject)
		if !isCls {
			return false
		}
		if mv, ok := co.props[name]; ok && isCallable(mv) {
			return true
		}
		cls = co.props["__super"]
	}
	return false
}

// ktClsFind walks a descriptor's __super chain for a callable member.
func ktClsFind(cls interface{}, name string) interface{} {
	for c := cls; c != nil; {
		co, isObj := c.(*jsObject)
		if !isObj {
			return nil
		}
		if v, ok := co.props[name]; ok && isCallable(v) {
			return v
		}
		c = co.props["__super"]
	}
	return nil
}

// ktMemberExtFind looks a mangled member-extension name up on the enclosing receiver
// chain of a frame: the innermost `this`, then the dispatch receiver of a member
// extension we are already inside, each followed out through its __outer instances.
// It answers the function and the DISPATCH receiver to pass as its first argument.
func (rt *jsrt) ktMemberExtFind(sc *jsScope, mangled string) (interface{}, interface{}, bool) {
	var starts []interface{}
	for _, key := range []string{"this", "__dispatch"} {
		for s := sc; s != nil; s = s.parent {
			if v, ok := s.get(key); ok {
				starts = append(starts, v)
				break
			}
		}
	}
	for _, cur := range starts {
		for i := 0; i < 64; i++ {
			o, isObj := cur.(*jsObject)
			if !isObj {
				break
			}
			// The receiver's OWN properties first. A COMPANION object's members hang
			// on the class descriptor itself (its methods live one level down, on the
			// holder the descriptor's __class points at), so a member extension
			// declared in a companion was installed on the descriptor and the
			// __class walk below never saw it: `fun Int.tripled() = ...` inside a
			// companion aborted the compiled half with "method call 'tripled' on a
			// number" while the interpreter half, whose clsFind starts at the
			// descriptor, answered. No Kotlin identifier can contain `$`, so a
			// mangled name is never an ordinary field.
			if fn, ok := o.props[mangled]; ok && isCallable(fn) {
				return fn, o, true
			}
			if fn := ktClsFind(o.props["__class"], mangled); fn != nil {
				return fn, o, true
			}
			nxt, ok := o.props["__outer"]
			if !ok || isUndefOrNull(nxt) {
				break
			}
			cur = nxt
		}
	}
	return nil, nil, false
}

// ktDelegKind names which delegate protocol a `by` expression implements, or "" when
// it implements none. "obj" is a user object with getValue (and optionally setValue),
// "map" a Map delegated by the property's NAME, "box" one of the
// kotlin.properties.Delegates boxes and "lazy" a lazy handle.
func ktDelegKind(v interface{}) string {
	o, isObj := v.(*jsObject)
	if !isObj {
		return ""
	}
	if k, ok := o.props["__ktdeleg"].(string); ok && k != "" {
		return "box"
	}
	if lz, _ := o.props["__lazy"].(bool); lz {
		return "lazy"
	}
	if _, _, isDict := dictParts(o); isDict {
		return "map"
	}
	if ktHasMethod(v, "getValue") {
		return "obj"
	}
	return ""
}

// ktDelegBox builds one kotlin.properties.Delegates delegate.
func ktDelegBox(kind string, initial, cb interface{}) *jsObject {
	o := newJSObject()
	o.set("__ktdeleg", kind)
	o.set("v", initial)
	o.set("cb", cb)
	o.set("set", kind != "notNull")
	return o
}

// ktPDeleg recognizes the box a LOCAL or top-level delegated property binds.
func ktPDeleg(v interface{}) (*jsObject, bool) {
	o, isObj := v.(*jsObject)
	if !isObj {
		return nil, false
	}
	if b, ok := o.props["__pdeleg"].(bool); ok && b {
		return o, true
	}
	return nil, false
}

func ktMakeSet() *jsObject {
	o := newJSObject()
	o.set("__set", true)
	o.set("keys", &jsArray{})
	return o
}

// ktSetParts returns the element array of a set handle.
func ktSetParts(v interface{}) (*jsArray, bool) {
	o, ok := v.(*jsObject)
	if !ok {
		return nil, false
	}
	if tag, has := o.props["__set"]; !has || tag != true {
		return nil, false
	}
	keys, _ := o.props["keys"].(*jsArray)
	if keys == nil {
		return nil, false
	}
	return keys, true
}

// ktEnumMethod is values() / valueOf(name) / entries on an enum class descriptor.
// `entries` is Kotlin 1.9's EnumEntries, a List; `values()` is an Array. Both are the
// declaration-ordered entry list in this value model.
func (rt *jsrt) ktEnumMethod(recv interface{}, name string, args []interface{}) (interface{}, bool) {
	o, ok := recv.(*jsObject)
	if !ok {
		return nil, false
	}
	if t, has := o.props["__isenum"]; !has || t != true {
		return nil, false
	}
	es, _ := o.props["__entries"].(*jsArray)
	if es == nil {
		return nil, false
	}
	switch name {
	case "values", "entries":
		return &jsArray{elems: append([]interface{}{}, es.elems...)}, true
	case "valueOf":
		want := rt.toString(argAt(args, 0))
		for _, e := range es.elems {
			if eo, isObj := e.(*jsObject); isObj {
				if n, _ := eo.props["name"].(string); n == want {
					return e, true
				}
			}
		}
		cn, _ := o.props["__name"].(string)
		// Kotlin THROWS IllegalArgumentException here, which is why
		// `runCatching { Level.valueOf(s) }.getOrNull()` is the ordinary way to parse
		// an enum from text. This used to abort the whole run with a host error in
		// both halves, so that idiom could not be written. Twin: the valueOf arm of
		// mcall in kotlin-interpreter.abnf.
		rt.ktRaise("IllegalArgumentException", "No enum constant "+cn+"."+want)
	}
	return nil, false
}

func ktIsSet(v interface{}) bool {
	_, ok := ktSetParts(v)
	return ok
}

// ktSetAdd inserts unless an equal element is already present; it reports whether
// the set GREW, which is what kotlin.collections.MutableSet.add answers.
func (rt *jsrt) ktSetAdd(s *jsObject, e interface{}) bool {
	keys, _ := ktSetParts(s)
	if rt.ktMapFind(keys, e) >= 0 {
		return false
	}
	keys.dropIdx()
	keys.elems = append(keys.elems, e)
	return true
}

// ktSetFrom builds a set from a slice, deduplicating in first-seen order.
func (rt *jsrt) ktSetFrom(es []interface{}) *jsObject {
	s := ktMakeSet()
	for _, e := range es {
		rt.ktSetAdd(s, e)
	}
	return s
}

// ktMapPut inserts or replaces, keeping insertion order.
func (rt *jsrt) ktMapPut(m *jsObject, k, v interface{}) {
	keys, vals, _ := dictParts(m)
	if i := rt.ktMapFind(keys, k); i >= 0 {
		vals.elems[i] = v
		return
	}
	keys.dropIdx()
	keys.elems = append(keys.elems, k)
	vals.elems = append(vals.elems, v)
}

// ktMapFind compares keys the way Kotlin's Map does - by equals, which for the
// value model here is ktEqVals (a boxed Long 1L and a plain Int 1 are the same
// key, exactly as java.lang.Long(1).equals is NOT... see the note on ktEqVals).
func (rt *jsrt) ktMapFind(keys *jsArray, k interface{}) int {
	for i, e := range keys.elems {
		if rt.ktEqVals(e, k) {
			return i
		}
	}
	return -1
}

// ktFloClsEq is the COLLECTION-KEY contract between two jsJFlo boxes, and it is a
// WIDTH question rather than a value one. Kotlin's `==` on Any is
// java.lang.Float.equals / java.lang.Double.equals, and Float.equals(Double) is
// FALSE by specification - java.lang.Float#equals answers true only "if the
// argument is not null and is a Float object" - so `setOf(1.5f, 1.5)` has TWO
// elements on the JVM. Confirmed against java 24.0.2, which is a real oracle here
// because Kotlin/JVM's Float IS java.lang.Float (the argument e5e68b5 and 30cd9f6
// settled the binary32 rounding and the hash with). 30cd9f6 already made the two
// widths HASH apart; this is the half of the same contract equality owes the hash.
//
// It is deliberately NOT in rt.strictEq. That primitive is shared with java,
// csharp, go and dart and its tag 13/14 arms are the numeric promotion `==` needs
// (docs/working-on-this-project.md 7.9); `<`, `>` and a well-typed program's
// primitive `==` reach jvmArith / ktCmp, which never come through here. The twin
// is ktFloEq in languages/lib/kotlin-rt.metajs and kEq in kotlin-interpreter.abnf.
//
// decided=false when either side is not a box, so a box against a plain number or
// a sized integer keeps the caller's own fallback. That pairing is a compile error
// in Kotlin (`1.5f == 1.5` and `1.0 == 1` are both rejected by kotlinc), so
// narrowing it would move only unreachable answers while risking reachable ones.
func ktFloClsEq(l, r interface{}) (bool, bool) {
	lf, lok := l.(jsJFlo)
	if !lok {
		return false, false
	}
	rf, rok := r.(jsJFlo)
	if !rok {
		return false, false
	}
	if jvmIs32(lf.sty) != jvmIs32(rf.sty) {
		return false, true
	}
	return lf.f == rf.f, true
}

// ktEqVals is `==` for the collection operations: numeric across the box, and the
// runtime's own strict equality for everything else (a data class still compares
// through its generated equals, which memberCall reaches).
func (rt *jsrt) ktEqVals(a, b interface{}) bool {
	if giIsInt(a) || giIsInt(b) {
		return rt.giEq(a, b)
	}
	if ca, ok := a.(jsChar); ok {
		if cb, ok2 := b.(jsChar); ok2 {
			return ca.code == cb.code
		}
	}
	// Two boxes: width-aware, so a Float is not a Double key. See ktFloClsEq.
	if v, decided := ktFloClsEq(a, b); decided {
		return v
	}
	if oa, ok := a.(*jsObject); ok {
		if _, ok2 := b.(*jsObject); ok2 {
			if mth, found := ktpFindMember(oa, "equals"); found {
				return rt.truthy(rt.call(mth, jsUndef, []interface{}{oa, b}))
			}
		}
	}
	if v, decided := rt.ktStructEq(a, b); decided {
		return v
	}
	return rt.strictEq(a, b)
}

// ktStructEq is the structural half of Kotlin's `==`, over the shapes this runtime
// models WITHOUT a class descriptor: a list/array/set (*jsArray), a map (the shared
// {__dict, keys, vals} handle) and a Pair/Triple. None of them can ever reach an
// `equals` member, so `listOf(1, 2) == listOf(1, 2)` used to fall through to
// identity and answer FALSE in both halves - while `listOf(1, 2).hashCode()` was
// already the element-wise java.util.AbstractList hash, so equality and hash
// disagreed. Verified against java (JDK 24): List/Map/Set equals are all
// element-wise. The twin is kStructEq in kotlin-interpreter.abnf.
//
// The second result says whether the question was DECIDED here; false means
// neither side is one of these shapes and the caller keeps its own fallback.
//
// Not a full model of Kotlin's rule: a Set is a list in this value model, so
// `setOf(2, 1) == setOf(1, 2)` is false and `listOf(1) == setOf(1)` is true. That
// is the setOf-is-listOf collapse, not this function.
func (rt *jsrt) ktStructEq(a, b interface{}) (bool, bool) {
	// A PROGRESSION, whose equals is kotlin.ranges.IntProgression's: two EMPTY
	// progressions are equal whatever their bounds, and a progression is never
	// equal to a List.
	arg, aIsRg := ktRangeOf(a)
	brg, bIsRg := ktRangeOf(b)
	if aIsRg || bIsRg {
		if !aIsRg || !bIsRg {
			return false, true
		}
		return ktRangeEq(arg, brg), true
	}
	// A SET first, so a Set never reaches the element-WISE array branch: two sets
	// are equal when they hold the same elements in ANY order, and a Set is never
	// equal to a List (java: Set.of(1, 2).equals(Set.of(2, 1)) is true and
	// List.of(1).equals(Set.of(1)) is false).
	as, aIsSet := ktSetParts(a)
	bs, bIsSet := ktSetParts(b)
	if aIsSet || bIsSet {
		if !aIsSet || !bIsSet || len(as.elems) != len(bs.elems) {
			return false, true
		}
		for _, e := range as.elems {
			if rt.ktMapFind(bs, e) < 0 {
				return false, true
			}
		}
		return true, true
	}
	aa, aIsArr := a.(*jsArray)
	ba, bIsArr := b.(*jsArray)
	if aIsArr || bIsArr {
		if !aIsArr || !bIsArr || len(aa.elems) != len(ba.elems) {
			return false, true
		}
		for i := range aa.elems {
			if !rt.ktEqVals(aa.elems[i], ba.elems[i]) {
				return false, true
			}
		}
		return true, true
	}
	ak, av, aIsMap := dictParts(a)
	bk, bv, bIsMap := dictParts(b)
	if aIsMap || bIsMap {
		if !aIsMap || !bIsMap || len(ak.elems) != len(bk.elems) {
			return false, true
		}
		for i, k := range ak.elems {
			j := rt.ktMapFind(bk, k)
			if j < 0 || !rt.ktEqVals(av.elems[i], bv.elems[j]) {
				return false, true
			}
		}
		return true, true
	}
	ap, aIsPair := ktIsPair(a)
	bp, bIsPair := ktIsPair(b)
	if aIsPair || bIsPair {
		if !aIsPair || !bIsPair {
			return false, true
		}
		if !rt.ktEqVals(ap.props["first"], bp.props["first"]) ||
			!rt.ktEqVals(ap.props["second"], bp.props["second"]) {
			return false, true
		}
		at, aHas := ap.props["third"]
		bt, bHas := bp.props["third"]
		if aHas != bHas {
			return false, true
		}
		return !aHas || rt.ktEqVals(at, bt), true
	}
	return false, false
}

// ktIsPair reports the {first, second[, third]} shape `to`, Pair() and Triple()
// build. It is a plain object rather than a class instance in both halves, which
// is why its rendering and its componentN live here.
func ktIsPair(v interface{}) (*jsObject, bool) {
	o, ok := v.(*jsObject)
	if !ok {
		return nil, false
	}
	if _, has := o.props["first"]; !has {
		return nil, false
	}
	if _, has := o.props["second"]; !has {
		return nil, false
	}
	// The compiler half's `to` builds a real Pair CLASS instance while the
	// interpreter's builds a plain object, so both shapes have to be recognized -
	// otherwise `associate` refused the compiler's pairs and `println(1 to 2)`
	// printed `Pair@1` in one half and `(1, 2)` in the other. A user class that
	// happens to have `first` and `second` is NOT a Pair: only the two names Kotlin
	// gives its own tuples are accepted.
	if cls, isCls := o.props["__class"]; isCls {
		co, _ := cls.(*jsObject)
		if co == nil {
			return nil, false
		}
		n, _ := co.props["__name"].(string)
		if n != "Pair" && n != "Triple" {
			return nil, false
		}
	}
	return o, true
}

// ktCall invokes a lambda argument.
func (rt *jsrt) ktCall(f interface{}, args ...interface{}) interface{} {
	return rt.call(f, jsUndef, args)
}

// ktScopeMethod is kotlin.let / run / apply / also / takeIf / takeUnless: the
// scope functions, which are extensions on ANY receiver. A class MEMBER of the
// same name wins, exactly as it does in Kotlin (a member always beats an
// extension), which is what the ktpFindMember probe below is for.
func (rt *jsrt) ktScopeMethod(target interface{}, name string, args []interface{}) (interface{}, bool) {
	switch name {
	case "let", "run", "apply", "also", "takeIf", "takeUnless":
	default:
		return nil, false
	}
	if len(args) != 1 || !isCallable(args[0]) {
		return nil, false
	}
	if o, ok := target.(*jsObject); ok {
		if _, found := ktpFindMember(o, name); found {
			return nil, false
		}
	}
	f := args[0]
	switch name {
	case "let":
		return rt.ktCall(f, target), true
	case "run":
		// `x.run { … }` binds the receiver as `this`. A bare closure handle has no
		// receiver slot, so the receiver goes onto ktRecvStack for the duration of
		// the call (which is what makes an unqualified `size` inside resolve) and is
		// ALSO passed as `it` - which is what the interpreter half does too.
		return rt.ktWithRecv(target, f, target), true
	case "apply":
		rt.ktWithRecv(target, f, target)
		return target, true
	case "also":
		rt.ktCall(f, target)
		return target, true
	case "takeIf":
		if rt.truthy(rt.ktCall(f, target)) {
			return target, true
		}
		return jsNull, true
	case "takeUnless":
		if rt.truthy(rt.ktCall(f, target)) {
			return jsNull, true
		}
		return target, true
	}
	return nil, false
}

// ktCharMethod is the kotlin.Char member surface. Char is a real type here (the
// jsChar box), not a one-character String, so every one of these has to answer
// from the CODE rather than from a string.
func (rt *jsrt) ktCharMethod(c jsChar, name string, args []interface{}) (interface{}, bool) {
	r := rune(c.code)
	switch name {
	case "rangeTo", "until", "downTo":
		if len(args) == 1 {
			return ktRangeMake(c, args[0], name), true
		}
	case "code":
		return float64(c.code), true
	case "toString":
		return string(r), true
	case "isLetter":
		return ktIsLetter(r), true
	case "isDigit":
		return r >= '0' && r <= '9', true
	case "isLetterOrDigit":
		return ktIsLetter(r) || (r >= '0' && r <= '9'), true
	case "isWhitespace":
		return r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\f' || r == 0x0b, true
	case "isUpperCase":
		return r >= 'A' && r <= 'Z', true
	case "isLowerCase":
		return r >= 'a' && r <= 'z', true
	case "uppercaseChar":
		return jsChar{code: int32(ktUpper(r))}, true
	case "lowercaseChar":
		return jsChar{code: int32(ktLower(r))}, true
	case "uppercase":
		return string(ktUpper(r)), true
	case "lowercase":
		return string(ktLower(r)), true
	case "digitToInt":
		if r >= '0' && r <= '9' {
			return float64(r - '0'), true
		}
		rt.fail("Char %q is not a digit", string(r))
	case "compareTo":
		o := int32(0)
		if oc, ok := argAt(args, 0).(jsChar); ok {
			o = oc.code
		} else {
			o = int32(giVal(rt, argAt(args, 0)))
		}
		switch {
		case c.code < o:
			return float64(-1), true
		case c.code > o:
			return float64(1), true
		}
		return float64(0), true
	case "equals":
		if oc, ok := argAt(args, 0).(jsChar); ok {
			return c.code == oc.code, true
		}
		return false, true
	case "hashCode":
		return float64(c.code), true
	}
	// The numeric surface (toInt/toLong/toDouble) reads the code.
	if v, handled := rt.ktNumMethod(float64(c.code), name, args); handled {
		return v, true
	}
	return nil, false
}

// ktIsLetter is Character.isLetter restricted to what this subset can promise: the
// ASCII letters plus every rune above 0x7f that Go classifies as a letter.
func ktIsLetter(r rune) bool {
	if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
		return true
	}
	return r > 0x7f && strings.ToUpper(string(r)) != strings.ToLower(string(r))
}

func ktUpper(r rune) rune {
	s := []rune(strings.ToUpper(string(r)))
	if len(s) == 1 {
		return s[0]
	}
	return r
}

func ktLower(r rune) rune {
	s := []rune(strings.ToLower(string(r)))
	if len(s) == 1 {
		return s[0]
	}
	return r
}

// ktChars explodes a string into its Char elements.
func (rt *jsrt) ktChars(s string) []interface{} {
	n := rt.strLen(s)
	out := make([]interface{}, n)
	for i := 0; i < n; i++ {
		out[i] = jsChar{code: int32(rt.strCodeAt(s, i))}
	}
	return out
}

// ktStrArg renders an argument the way a String member expects it: a Char argument
// is its glyph, so `s.indexOf('B')` and `s.indexOf("B")` agree.
func (rt *jsrt) ktStrArg(v interface{}) string {
	if c, ok := v.(jsChar); ok {
		return string(rune(c.code))
	}
	return rt.ktStr2(v)
}

// ktSeqMethod is the kotlin.collections surface over a list. The shared js_mcall
// already carries add/size/get/contains/map/filter/sumOf/forEach/count/any; this
// answers the rest, and answers (nil, false) for a name it does not know so the
// shared table keeps whatever behaviour it had.
//
// `sumOf` is re-answered here because the shared one accumulates in int32 and
// Kotlin's sumOf takes the type of the selector's result - a Long selector sums to
// a Long, which is the difference between 3000000000 and -1294967296.
func (rt *jsrt) ktSeqMethod(o *jsArray, name string, args []interface{}) (interface{}, bool) {
	es := o.elems
	f := argAt(args, 0)
	arr := func(v []interface{}) *jsArray { return &jsArray{elems: v} }
	switch name {
	case "isEmpty":
		return len(es) == 0, true
	case "isNotEmpty":
		return len(es) > 0, true
	// The names the shared js_mcall already answers for an ARRAY are repeated here
	// because a STRING reaches this function too (a String is a sequence of Char in
	// Kotlin, so "abc".map { it.code } is ordinary). The bodies are the same ones, so
	// an array's answer is unchanged whichever branch reaches it.
	case "map":
		out := make([]interface{}, len(es))
		for i, e := range es {
			out[i] = rt.ktCall(f, e)
		}
		return arr(out), true
	case "filter":
		out := []interface{}{}
		for _, e := range es {
			if rt.truthy(rt.ktCall(f, e)) {
				out = append(out, e)
			}
		}
		return arr(out), true
	case "forEach":
		for _, e := range es {
			rt.ktCall(f, e)
		}
		return jsUndef, true
	case "count":
		if len(args) == 0 {
			return float64(len(es)), true
		}
		n := 0
		for _, e := range es {
			if rt.truthy(rt.ktCall(f, e)) {
				n++
			}
		}
		return float64(n), true
	case "any":
		for _, e := range es {
			if rt.truthy(rt.ktCall(f, e)) {
				return true, true
			}
		}
		return false, true
	case "contains":
		for _, e := range es {
			if rt.ktEqVals(e, f) {
				return true, true
			}
		}
		return false, true
	case "indexOf":
		for i, e := range es {
			if rt.ktEqVals(e, f) {
				return float64(i), true
			}
		}
		return float64(-1), true
	case "first":
		if len(args) == 1 && isCallable(f) {
			for _, e := range es {
				if rt.truthy(rt.ktCall(f, e)) {
					return e, true
				}
			}
			rt.fail("no element matching the predicate")
		}
		if len(es) == 0 {
			rt.fail("first() on an empty list")
		}
		return es[0], true
	case "last":
		if len(es) == 0 {
			rt.fail("last() on an empty list")
		}
		return es[len(es)-1], true
	case "single":
		if len(args) == 1 && isCallable(f) {
			var hit interface{} = jsUndef
			n := 0
			for _, e := range es {
				if rt.truthy(rt.ktCall(f, e)) {
					hit, n = e, n+1
				}
			}
			if n != 1 {
				rt.fail("single() matched %d elements", n)
			}
			return hit, true
		}
		if len(es) != 1 {
			rt.fail("single() on a list of size %d", len(es))
		}
		return es[0], true
	case "firstOrNull":
		if len(args) == 1 && isCallable(f) {
			for _, e := range es {
				if rt.truthy(rt.ktCall(f, e)) {
					return e, true
				}
			}
			return jsNull, true
		}
		if len(es) == 0 {
			return jsNull, true
		}
		return es[0], true
	case "lastOrNull":
		if len(es) == 0 {
			return jsNull, true
		}
		return es[len(es)-1], true
	case "find":
		for _, e := range es {
			if rt.truthy(rt.ktCall(f, e)) {
				return e, true
			}
		}
		return jsNull, true
	case "all":
		for _, e := range es {
			if !rt.truthy(rt.ktCall(f, e)) {
				return false, true
			}
		}
		return true, true
	case "none":
		if len(args) == 0 {
			return len(es) == 0, true
		}
		for _, e := range es {
			if rt.truthy(rt.ktCall(f, e)) {
				return false, true
			}
		}
		return true, true
	case "indexOfFirst":
		for i, e := range es {
			if rt.truthy(rt.ktCall(f, e)) {
				return float64(i), true
			}
		}
		return float64(-1), true
	case "fold":
		acc := argAt(args, 0)
		for _, e := range es {
			acc = rt.ktCall(argAt(args, 1), acc, e)
		}
		return acc, true
	case "foldRight":
		acc := argAt(args, 0)
		for i := len(es) - 1; i >= 0; i-- {
			acc = rt.ktCall(argAt(args, 1), es[i], acc)
		}
		return acc, true
	case "reduce":
		if len(es) == 0 {
			rt.fail("reduce() on an empty list")
		}
		acc := es[0]
		for _, e := range es[1:] {
			acc = rt.ktCall(f, acc, e)
		}
		return acc, true
	case "sumOf":
		return ktSumOf(rt, es, f), true
	case "sum":
		return ktSumOf(rt, es, nil), true
	case "average":
		if len(es) == 0 {
			return jsJFlo{f: math.NaN()}, true
		}
		t := 0.0
		for _, e := range es {
			t += giFloat(rt, e)
		}
		return jsJFlo{f: t / float64(len(es))}, true
	case "flatMap":
		out := []interface{}{}
		for _, e := range es {
			out = append(out, ktElems(rt, rt.ktCall(f, e))...)
		}
		return arr(out), true
	case "flatten":
		out := []interface{}{}
		for _, e := range es {
			out = append(out, ktElems(rt, e)...)
		}
		return arr(out), true
	case "mapIndexed":
		out := make([]interface{}, len(es))
		for i, e := range es {
			out[i] = rt.ktCall(f, float64(i), e)
		}
		return arr(out), true
	case "mapNotNull":
		out := []interface{}{}
		for _, e := range es {
			v := rt.ktCall(f, e)
			if _, isNull := v.(jsNullT); !isNull {
				out = append(out, v)
			}
		}
		return arr(out), true
	case "forEachIndexed":
		for i, e := range es {
			rt.ktCall(f, float64(i), e)
		}
		return jsUndef, true
	case "filterNot":
		out := []interface{}{}
		for _, e := range es {
			if !rt.truthy(rt.ktCall(f, e)) {
				out = append(out, e)
			}
		}
		return arr(out), true
	case "partition":
		yes, no := []interface{}{}, []interface{}{}
		for _, e := range es {
			if rt.truthy(rt.ktCall(f, e)) {
				yes = append(yes, e)
			} else {
				no = append(no, e)
			}
		}
		p := newJSObject()
		p.set("first", arr(yes))
		p.set("second", arr(no))
		return p, true
	case "zip":
		other := ktElems(rt, argAt(args, 0))
		out := []interface{}{}
		for i := 0; i < len(es) && i < len(other); i++ {
			if len(args) > 1 && isCallable(args[1]) {
				out = append(out, rt.ktCall(args[1], es[i], other[i]))
				continue
			}
			p := newJSObject()
			p.set("first", es[i])
			p.set("second", other[i])
			out = append(out, p)
		}
		return arr(out), true
	// chunked(size[, transform]) and windowed(size, step = 1, partialWindows =
	// false[, transform]). The TRANSFORM used to be dropped on the floor in all
	// three engines - `listOf(1,2,3,4).chunked(2) { it.sum() }` answered the chunks
	// themselves - and windowed's partialWindows was ignored, so the tail windows it
	// asks for were never produced. todo.md 1.5.
	case "chunked", "windowed":
		k := jsToInt(rt.toNumber(argAt(args, 0)))
		step, partial := k, true
		var xf interface{}
		if name == "windowed" {
			step, partial = 1, false
			if len(args) > 1 && !isCallable(args[1]) {
				step = jsToInt(rt.toNumber(args[1]))
			}
			if len(args) > 2 && !isCallable(args[2]) {
				partial = rt.truthy(args[2])
			}
		}
		for _, a := range args[1:] {
			if isCallable(a) {
				xf = a
			}
		}
		if k <= 0 || step <= 0 {
			rt.fail("%s() size and step must be positive", name)
		}
		out := []interface{}{}
		for i := 0; ; i += step {
			if partial {
				if i >= len(es) {
					break
				}
			} else if i+k > len(es) {
				break
			}
			j := i + k
			if j > len(es) {
				j = len(es)
			}
			piece := arr(append([]interface{}{}, es[i:j]...))
			if xf != nil {
				out = append(out, rt.ktCall(xf, piece))
			} else {
				out = append(out, piece)
			}
		}
		return arr(out), true
	// kotlin.collections.random / randomOrNull. See the kotlin.random block: the
	// generator is deterministic, which is the only thing three engines can agree on.
	case "random", "randomOrNull":
		if len(es) == 0 {
			if name == "randomOrNull" {
				return jsNull, true
			}
			rt.ktRaise("NoSuchElementException", "Collection is empty.")
		}
		return es[ktRndBelow(rt.ktRndOf(args), len(es))], true
	case "distinct":
		// distinct answers a LIST and toSet/toMutableSet/toHashSet a SET - the
		// deduplication is the same, the SHAPE is not (kotlin.collections declares
		// distinct(): List and toSet(): Set). While every set was a list these were
		// one case.
		out := []interface{}{}
		for _, e := range es {
			seen := false
			for _, k := range out {
				if rt.ktEqVals(k, e) {
					seen = true
					break
				}
			}
			if !seen {
				out = append(out, e)
			}
		}
		return arr(out), true
	case "toSet", "toMutableSet", "toHashSet":
		return rt.ktSetFrom(es), true
	case "toSortedSet":
		return rt.ktSetFrom(ktSortBy(rt, es, nil, false).elems), true
	case "reversed", "asReversed":
		out := make([]interface{}, len(es))
		for i, e := range es {
			out[len(es)-1-i] = e
		}
		return arr(out), true
	case "sorted", "sortedDescending":
		return ktSortBy(rt, es, nil, name == "sortedDescending"), true
	// The ARRAY spellings of sorted/sortedDescending/reversed. Kotlin gives Array its
	// own names because they answer an Array rather than a List; the two are one shape
	// here, so they are the list operators under a second name.
	case "sortedArray", "sortedArrayDescending":
		return ktSortBy(rt, es, nil, name == "sortedArrayDescending"), true
	case "reversedArray":
		out := make([]interface{}, len(es))
		for i, e := range es {
			out[len(es)-1-i] = e
		}
		return arr(out), true
	// contentToString / contentEquals: an Array gets NO structural toString or equals in
	// Kotlin (it prints as its identity and compares by reference), so these two are how
	// an array is printed and compared element by element. ktpRender already produces
	// Kotlin's `[1, 2]` for the list shape an array is here.
	case "contentToString", "contentDeepToString":
		return rt.ktpRender(arr(es), 0), true
	case "contentEquals", "contentDeepEquals":
		other, ok := argAt(args, 0).(*jsArray)
		if !ok || len(other.elems) != len(es) {
			return false, true
		}
		for i, e := range es {
			if !rt.ktEqVals(e, other.elems[i]) {
				return false, true
			}
		}
		return true, true
	case "sortedBy", "sortedByDescending":
		return ktSortBy(rt, es, f, name == "sortedByDescending"), true
	case "maxOrNull", "minOrNull":
		if len(es) == 0 {
			return jsNull, true
		}
		s := ktSortBy(rt, es, nil, name == "maxOrNull")
		return s.elems[0], true
	case "maxByOrNull", "minByOrNull":
		if len(es) == 0 {
			return jsNull, true
		}
		s := ktSortBy(rt, es, f, name == "maxByOrNull")
		return s.elems[0], true
	case "take", "drop", "takeLast", "dropLast":
		k := jsToInt(rt.toNumber(argAt(args, 0)))
		if k < 0 {
			k = 0
		}
		if k > len(es) {
			k = len(es)
		}
		switch name {
		case "take":
			return arr(append([]interface{}{}, es[:k]...)), true
		case "drop":
			return arr(append([]interface{}{}, es[k:]...)), true
		case "dropLast":
			return arr(append([]interface{}{}, es[:len(es)-k]...)), true
		}
		return arr(append([]interface{}{}, es[len(es)-k:]...)), true
	// The ARRAY-to-collection conversions. An Array and a List are the same host array
	// in this value model, so every one of them is a COPY of the elements - which is
	// what Kotlin's toList/toMutableList/toTypedArray/to*Array do. `asList`,
	// `asSequence` and `asIterable` are Kotlin VIEWS (a write through the array shows
	// in them); the copy differs observably only for a program that mutates the array
	// and then reads the view, which this subset does not model. Twin of the same
	// grouped arm in kotlin-interpreter.abnf.
	case "toList", "toMutableList", "asSequence", "asIterable", "asList",
		"toIntArray", "toLongArray", "toShortArray", "toByteArray",
		"toDoubleArray", "toFloatArray", "toBooleanArray", "toCharArray",
		"toTypedArray":
		return arr(append([]interface{}{}, es...)), true
	case "withIndex":
		out := make([]interface{}, len(es))
		for i, e := range es {
			p := newJSObject()
			// __ktiv marks the value as kotlin.collections.IndexedValue, a DATA
			// class whose toString is "IndexedValue(index=0, value=10)" - not the
			// Pair "(0, 10)" every engine printed, agreeing and all wrong. The
			// first/second slots stay so that `for ((i, v) in xs.withIndex())` and
			// `associate` keep destructuring it, exactly as before; only the render
			// changes, and the `__` prefix keeps the marker out of keysOf.
			p.set("__ktiv", true)
			p.set("index", float64(i))
			p.set("value", e)
			p.set("first", float64(i))
			p.set("second", e)
			out[i] = p
		}
		return arr(out), true
	case "groupBy":
		m := ktMakeMap()
		for _, e := range es {
			k := rt.ktCall(f, e)
			keys, vals, _ := dictParts(m)
			i := rt.ktMapFind(keys, k)
			if i < 0 {
				rt.ktMapPut(m, k, arr([]interface{}{e}))
				continue
			}
			b := vals.elems[i].(*jsArray)
			b.elems = append(b.elems, e)
		}
		return m, true
	case "associate":
		m := ktMakeMap()
		for _, e := range es {
			p := rt.ktCall(f, e)
			po, ok := ktIsPair(p)
			if !ok {
				rt.fail("associate() needs a Pair")
			}
			rt.ktMapPut(m, po.props["first"], po.props["second"])
		}
		return m, true
	case "associateWith":
		m := ktMakeMap()
		for _, e := range es {
			rt.ktMapPut(m, e, rt.ktCall(f, e))
		}
		return m, true
	case "associateBy":
		m := ktMakeMap()
		for _, e := range es {
			rt.ktMapPut(m, rt.ktCall(f, e), e)
		}
		return m, true
	case "toMap":
		m := ktMakeMap()
		for _, e := range es {
			if po, ok := ktIsPair(e); ok {
				rt.ktMapPut(m, po.props["first"], po.props["second"])
			}
		}
		return m, true
	case "joinToString":
		// The TRANSFORM is joinToString's last parameter, so `joinToString { ... }`
		// passes a function - which used to be read as the SEPARATOR and printed
		// "[function]" between the elements (a wrong answer, and a divergence: the
		// interpreter half rendered the whole lambda's script text there).
		var transform interface{}
		text := []interface{}{}
		for _, a := range args {
			if isCallable(a) {
				transform = a
			} else {
				text = append(text, a)
			}
		}
		sep := ", "
		if len(text) > 0 {
			sep = rt.ktStrArg(text[0])
		}
		pre, post := "", ""
		if len(text) > 1 {
			pre = rt.ktStrArg(text[1])
		}
		if len(text) > 2 {
			post = rt.ktStrArg(text[2])
		}
		parts := make([]string, len(es))
		for i, e := range es {
			if transform != nil {
				parts[i] = rt.ktStr2(rt.ktCall(transform, e))
				continue
			}
			parts[i] = rt.ktpRender(e, 0)
		}
		return pre + strings.Join(parts, sep) + post, true
	case "hashCode":
		// java.util.AbstractList.hashCode: 31 * acc + element.
		var h int32 = 1
		for _, e := range es {
			h = 31*h + int32(ktElemHash(rt, e))
		}
		return float64(h), true

	// ----- More of kotlin.collections, added in one batch because a realistic program
	// reaches for them constantly and every one aborted the run in BOTH halves with
	// "unknown list method". The twin is kSeqMethod in kotlin-interpreter.abnf; the two
	// tables are kept in step name for name.
	// filterIsInstance<T>(): the elements that ARE a T. The type argument is the whole
	// point of the call, and both grammars used to SKIP a call's explicit type arguments
	// (`"." KId [ SkipAngle ] MArgs`), so the name reached the runtime with nothing to
	// filter on and both halves aborted with "unknown list method". MTypeArgs now
	// captures the list and the mcall emitters pass the type NAME as a trailing string
	// argument for the handful of methods that need it. A null is never an instance of a
	// non-nullable T, so it is dropped, exactly as Kotlin's own filterIsInstance does.
	case "filterIsInstance":
		want := "Any"
		if sv, isStr := f.(string); isStr && sv != "" {
			want = sv
		}
		out := []interface{}{}
		for _, e := range es {
			if ktIsType != nil && ktIsType(e, want) {
				out = append(out, e)
			}
		}
		return arr(out), true
	case "filterNotNull", "requireNoNulls":
		out := []interface{}{}
		for _, e := range es {
			if isUndefOrNull(e) {
				if name == "requireNoNulls" {
					rt.fail("null element in requireNoNulls()")
				}
				continue
			}
			out = append(out, e)
		}
		return arr(out), true
	case "filterIndexed", "filterNotIndexed":
		out := []interface{}{}
		for i, e := range es {
			if rt.truthy(rt.ktCall(f, float64(i), e)) != (name == "filterNotIndexed") {
				out = append(out, e)
			}
		}
		return arr(out), true
	case "findLast":
		for i := len(es) - 1; i >= 0; i-- {
			if rt.truthy(rt.ktCall(f, es[i])) {
				return es[i], true
			}
		}
		return jsNull, true
	case "lastIndexOf":
		for i := len(es) - 1; i >= 0; i-- {
			if rt.ktEqVals(es[i], f) {
				return float64(i), true
			}
		}
		return float64(-1), true
	case "indexOfLast":
		for i := len(es) - 1; i >= 0; i-- {
			if rt.truthy(rt.ktCall(f, es[i])) {
				return float64(i), true
			}
		}
		return float64(-1), true
	case "takeWhile", "dropWhile":
		cut := len(es)
		for i, e := range es {
			if !rt.truthy(rt.ktCall(f, e)) {
				cut = i
				break
			}
		}
		if name == "takeWhile" {
			return arr(append([]interface{}{}, es[:cut]...)), true
		}
		return arr(append([]interface{}{}, es[cut:]...)), true
	case "distinctBy":
		out := []interface{}{}
		keys := []interface{}{}
		for _, e := range es {
			k := rt.ktCall(f, e)
			dup := false
			for _, sk := range keys {
				if rt.ktEqVals(sk, k) {
					dup = true
					break
				}
			}
			if !dup {
				keys = append(keys, k)
				out = append(out, e)
			}
		}
		return arr(out), true
	case "containsAll":
		for _, w := range ktElems(rt, f) {
			got := false
			for _, e := range es {
				if rt.ktEqVals(e, w) {
					got = true
					break
				}
			}
			if !got {
				return false, true
			}
		}
		return true, true
	case "elementAt", "elementAtOrNull", "elementAtOrElse", "getOrNull", "getOrElse":
		ix := jsToInt(rt.toNumber(f))
		if ix >= 0 && ix < len(es) {
			return es[ix], true
		}
		switch name {
		case "elementAt":
			rt.fail("index %d is out of bounds", ix)
		case "elementAtOrElse", "getOrElse":
			return rt.ktCall(argAt(args, 1), float64(ix)), true
		}
		return jsNull, true
	case "maxOf", "minOf", "maxOfOrNull", "minOfOrNull":
		if len(es) == 0 {
			if name == "maxOf" || name == "minOf" {
				rt.fail("%s() on an empty list", name)
			}
			return jsNull, true
		}
		wantMax := name == "maxOf" || name == "maxOfOrNull"
		best := rt.ktCall(f, es[0])
		for _, e := range es[1:] {
			cand := rt.ktCall(f, e)
			if ktKeyLess(rt, best, cand) == wantMax {
				best = cand
			}
		}
		return best, true
	// max()/min() were removed in Kotlin 1.7 and reintroduced as the throwing
	// counterparts of maxOrNull/minOrNull; both spellings answer the same thing.
	case "max", "min":
		if len(es) == 0 {
			rt.fail("%s() on an empty list", name)
		}
		return ktSortBy(rt, es, nil, name == "max").elems[0], true
	case "maxBy", "minBy":
		if len(es) == 0 {
			rt.fail("%s() on an empty list", name)
		}
		return ktSortBy(rt, es, f, name == "maxBy").elems[0], true
	case "onEach":
		for _, e := range es {
			rt.ktCall(f, e)
		}
		return o, true
	case "onEachIndexed":
		for i, e := range es {
			rt.ktCall(f, float64(i), e)
		}
		return o, true
	case "plus":
		out := append([]interface{}{}, es...)
		if ktIsCollection(f) {
			out = append(out, ktElems(rt, f)...)
		} else {
			out = append(out, f)
		}
		return arr(out), true
	case "minus":
		rem := []interface{}{f}
		if ktIsCollection(f) {
			rem = ktElems(rt, f)
		}
		out := []interface{}{}
		for _, e := range es {
			drop := false
			for _, r := range rem {
				if rt.ktEqVals(r, e) {
					drop = true
					break
				}
			}
			if !drop {
				out = append(out, e)
			}
		}
		return arr(out), true
	case "intersect", "subtract":
		// union / intersect / subtract are declared on Iterable and answer a SET,
		// not a List - they used to answer a deduplicated list, which was only right
		// while a set WAS a list.
		oth := ktElems(rt, f)
		out := ktMakeSet()
		for _, e := range es {
			inOth := false
			for _, x := range oth {
				if rt.ktEqVals(x, e) {
					inOth = true
					break
				}
			}
			if inOth != (name == "subtract") {
				rt.ktSetAdd(out, e)
			}
		}
		return out, true
	case "union":
		out := rt.ktSetFrom(es)
		for _, e := range ktElems(rt, f) {
			rt.ktSetAdd(out, e)
		}
		return out, true
	// sliceArray is slice under the Array name (Kotlin gives it its own name
	// because it answers an Array); here the two shapes are one, so it is the same
	// operator. It aborted with "unknown list method" in all three engines.
	case "slice", "sliceArray":
		out := []interface{}{}
		for _, ix := range ktElems(rt, f) {
			out = append(out, es[jsToInt(rt.toNumber(ix))])
		}
		return arr(out), true
	case "subList":
		lo, hi := jsToInt(rt.toNumber(f)), jsToInt(rt.toNumber(argAt(args, 1)))
		return arr(append([]interface{}{}, es[lo:hi]...)), true
	case "zipWithNext":
		out := []interface{}{}
		for i := 0; i+1 < len(es); i++ {
			if isCallable(f) {
				out = append(out, rt.ktCall(f, es[i], es[i+1]))
			} else {
				p := newJSObject()
				p.set("first", es[i])
				p.set("second", es[i+1])
				out = append(out, p)
			}
		}
		return arr(out), true
	case "runningFold", "scan", "runningReduce":
		out := []interface{}{}
		if name == "runningReduce" {
			if len(es) == 0 {
				return arr(out), true
			}
			acc := es[0]
			out = append(out, acc)
			for _, e := range es[1:] {
				acc = rt.ktCall(f, acc, e)
				out = append(out, acc)
			}
			return arr(out), true
		}
		acc := f
		out = append(out, acc)
		for _, e := range es {
			acc = rt.ktCall(argAt(args, 1), acc, e)
			out = append(out, acc)
		}
		return arr(out), true
	case "ifEmpty":
		if len(es) == 0 {
			return rt.ktCall(f), true
		}
		return o, true
	// A COMPARATOR (compareBy / compareByDescending / a two-argument lambda) is a
	// function of two elements answering an Int, so this is an insertion sort driven
	// by that function rather than by ktKeyLess.
	case "sortedWith":
		out := append([]interface{}{}, es...)
		for i := 1; i < len(out); i++ {
			v := out[i]
			j := i - 1
			for j >= 0 && rt.toNumber(rt.ktCall(f, out[j], v)) > 0 {
				out[j+1] = out[j]
				j--
			}
			out[j+1] = v
		}
		return arr(out), true
	case "removeAt":
		ix := jsToInt(rt.toNumber(f))
		if ix < 0 || ix >= len(o.elems) {
			rt.fail("index %d is out of bounds", ix)
		}
		gone := o.elems[ix]
		o.dropIdx()
		o.elems = append(o.elems[:ix], o.elems[ix+1:]...)
		return gone, true
	// The rest of the MutableList surface, all of it IN PLACE. Every one of these
	// aborted the run with "unknown list method" in BOTH halves, while their copying
	// twins (sorted, sortedBy, reversed) were present - so `l.sort()` and
	// `l.set(0, x)`, which any realistic program reaches for, were unreachable.
	case "add":
		// add(element) -> Boolean; add(index, element) -> Unit, INSERTING. The shared
		// js_mcall knows only the appending one-argument form, so `l.add(1, 7)`
		// appended the INDEX there. Answering both arities here keeps abnf/jsrt.go -
		// which backs all fifteen languages - untouched.
		if len(args) < 2 {
			o.dropIdx()
			o.elems = append(o.elems, f)
			return true, true
		}
		i := jsToInt(rt.toNumber(f))
		if i < 0 || i > len(es) {
			rt.fail("index %d is out of bounds", i)
		}
		o.dropIdx()
		o.elems = append(o.elems, nil)
		copy(o.elems[i+1:], o.elems[i:])
		o.elems[i] = argAt(args, 1)
		return jsUndef, true
	case "set":
		i := jsToInt(rt.toNumber(f))
		if i < 0 || i >= len(es) {
			rt.fail("index %d is out of bounds", i)
		}
		old := es[i]
		es[i] = argAt(args, 1)
		return old, true
	case "sort", "sortDescending", "sortBy", "sortByDescending", "sortWith", "reverse":
		var ord []interface{}
		switch name {
		case "reverse":
			ord = make([]interface{}, len(es))
			for i := range es {
				ord[i] = es[len(es)-1-i]
			}
		case "sortWith":
			v, _ := rt.ktSeqMethod(o, "sortedWith", args)
			ord = v.(*jsArray).elems
		case "sortBy":
			ord = ktSortBy(rt, es, f, false).elems
		case "sortByDescending":
			ord = ktSortBy(rt, es, f, true).elems
		default:
			ord = ktSortBy(rt, es, nil, name == "sortDescending").elems
		}
		copy(o.elems, ord)
		o.dropIdx()
		return jsUndef, true
	case "clear":
		o.dropIdx()
		o.elems = nil
		return jsUndef, true
	case "removeAll", "retainAll":
		// Kotlin has TWO overloads of each: one takes a collection of elements, the
		// other a PREDICATE. `l.removeAll { it % 2 == 0 }` is the commoner of the two
		// in real code and aborted BOTH halves with "not a sequence: [function]",
		// because only the collection overload existed. The twin is the same branch of
		// kMutListMethod in languages/kotlin-interpreter.abnf.
		isPred := isCallable(f)
		var drop []interface{}
		if !isPred {
			drop = ktElems(rt, f)
		}
		keep := []interface{}{}
		for _, e := range es {
			hit := false
			if isPred {
				hit = rt.truthy(rt.call(f, jsUndef, []interface{}{e}))
			} else {
				for _, d := range drop {
					if rt.ktEqVals(d, e) {
						hit = true
						break
					}
				}
			}
			if hit == (name == "retainAll") {
				keep = append(keep, e)
			}
		}
		changed := len(keep) != len(es)
		o.dropIdx()
		o.elems = keep
		return changed, true
	case "removeFirst", "removeLast":
		if len(o.elems) == 0 {
			rt.fail("%s() on an empty list", name)
		}
		ix := 0
		if name == "removeLast" {
			ix = len(o.elems) - 1
		}
		return rt.ktSeqMethod(o, "removeAt", []interface{}{float64(ix)})
	case "remove":
		for i, e := range o.elems {
			if rt.ktEqVals(e, f) {
				o.dropIdx()
				o.elems = append(o.elems[:i], o.elems[i+1:]...)
				return true, true
			}
		}
		return false, true
	case "addAll":
		more := ktElems(rt, f)
		o.dropIdx()
		o.elems = append(o.elems, more...)
		return len(more) > 0, true
	}
	return nil, false
}

// ktElemHash is the hash of a list element, matching kHash in the interpreter.
func ktElemHash(rt *jsrt, v interface{}) int64 {
	switch t := v.(type) {
	case string:
		return int64(ktStrHash(rt, t))
	case jsChar:
		return int64(t.code)
	case bool:
		if t {
			return 1231
		}
		return 1237
	case jsNullT:
		return 0
	case jsJFlo:
		// java.lang.Double.hashCode of a Double and java.lang.Float.hashCode of a
		// Float, not the truncated value - see ktFloHash.
		return int64(ktFloHash(t))
	}
	if giIsNumeric(v) {
		x := giVal(rt, v)
		if b, ok := v.(jsGInt); ok && b.w == 64 {
			return x ^ (x >> 32)
		}
		return int64(int32(x))
	}
	// A NESTED collection hashes element-wise, exactly as kHash does in the other
	// half: before this every object answered 1 here, so listOf(listOf(1, 2)) hashed
	// to 32 in the compiler and to 1025 - java's answer - in the interpreter.
	// A PROGRESSION hashes by its bounds, not its elements - and an EMPTY one
	// hashes to -1, which is what makes (5..1) and (7..3) agree with their equals.
	if rg, ok := ktRangeOf(v); ok {
		return int64(ktRangeHash(rg))
	}
	switch t := v.(type) {
	case *jsArray:
		var h int32 = 1
		for _, e := range t.elems {
			h = 31*h + int32(ktElemHash(rt, e))
		}
		return int64(h)
	case *jsObject:
		// java.util.AbstractSet.hashCode is the plain SUM of the element hashes,
		// which is what makes it order-independent. Verified against java (JDK 24):
		// Set.of(1, 2).hashCode() is 3, where the LIST fold answers 994.
		if keys, isSet := ktSetParts(t); isSet {
			var h int32
			for _, e := range keys.elems {
				h += int32(ktElemHash(rt, e))
			}
			return int64(h)
		}
		// java.util.AbstractMap.hashCode is the SUM of key XOR value over the entries.
		if keys, vals, isDict := dictParts(t); isDict {
			var h int32
			for i, k := range keys.elems {
				h += int32(ktElemHash(rt, k)) ^ int32(ktElemHash(rt, vals.elems[i]))
			}
			return int64(h)
		}
		// java.util.Map.Entry.hashCode is key.hashCode() XOR value.hashCode(), which
		// is what makes the entry SET hash to the same value as the map itself.
		if ktIsEntry(t) {
			return int64(int32(ktElemHash(rt, t.props["key"])) ^ int32(ktElemHash(rt, t.props["value"])))
		}
		// Pair/Triple are data classes in Kotlin: the same 31-fold their components
		// would get from a generated hashCode.
		if p, isPair := ktIsPair(t); isPair {
			h := int32(ktElemHash(rt, p.props["first"]))
			h = 31*h + int32(ktElemHash(rt, p.props["second"]))
			if third, has := p.props["third"]; has {
				h = 31*h + int32(ktElemHash(rt, third))
			}
			return int64(h)
		}
		// A user class that declares hashCode (a data class does) answers for itself.
		if mth, found := ktpFindMember(t, "hashCode"); found {
			return int64(int32(rt.toNumber(rt.call(mth, jsUndef, []interface{}{t}))))
		}
	}
	return 1
}

// ktSetMethod is the Set member surface. Only the members whose ANSWER is a Set,
// or that mutate the set, live here; everything else falls through to ktSeqMethod
// over the elements, which is right because kotlin.collections declares map /
// filter / sorted / reversed / joinToString / take / ... on Iterable and they all
// answer a LIST even for a Set receiver. The twin is kSetMethod in
// languages/kotlin-interpreter.abnf.
func (rt *jsrt) ktSetMethod(s *jsObject, name string, args []interface{}) (interface{}, bool) {
	keys, ok := ktSetParts(s)
	if !ok {
		return nil, false
	}
	f := argAt(args, 0)
	switch name {
	case "size":
		return float64(len(keys.elems)), true
	case "isEmpty":
		return len(keys.elems) == 0, true
	case "isNotEmpty":
		return len(keys.elems) > 0, true
	case "contains":
		return rt.ktMapFind(keys, f) >= 0, true
	case "containsAll":
		for _, e := range ktElems(rt, f) {
			if rt.ktMapFind(keys, e) < 0 {
				return false, true
			}
		}
		return true, true
	case "add":
		return rt.ktSetAdd(s, f), true
	case "addAll":
		grew := false
		for _, e := range ktElems(rt, f) {
			if rt.ktSetAdd(s, e) {
				grew = true
			}
		}
		return grew, true
	case "remove", "removeAll", "retainAll":
		gone := []interface{}{f}
		if name != "remove" {
			gone = ktElems(rt, f)
		}
		kept := []interface{}{}
		changed := false
		for _, e := range keys.elems {
			hit := false
			for _, g := range gone {
				if rt.ktEqVals(g, e) {
					hit = true
					break
				}
			}
			if hit == (name == "retainAll") {
				kept = append(kept, e)
			} else {
				changed = true
			}
		}
		keys.dropIdx()
		keys.elems = kept
		return changed, true
	case "clear":
		keys.dropIdx()
		keys.elems = nil
		return jsUndef, true
	case "hashCode":
		return float64(int32(ktElemHash(rt, s))), true
	case "equals":
		return rt.ktEqVals(s, f), true
	// The Set-valued operators: plus/union add, minus/subtract remove and intersect
	// keeps the common elements - all of them answering a Set, where the list
	// versions answer a List.
	case "plus", "union":
		out := rt.ktSetFrom(keys.elems)
		add := []interface{}{f}
		if ktIsCollection(f) {
			add = ktElems(rt, f)
		}
		for _, e := range add {
			rt.ktSetAdd(out, e)
		}
		return out, true
	case "minus", "subtract", "intersect":
		oth := []interface{}{f}
		if name != "minus" || ktIsCollection(f) {
			oth = ktElems(rt, f)
		}
		out := ktMakeSet()
		for _, e := range keys.elems {
			inOth := false
			for _, x := range oth {
				if rt.ktEqVals(x, e) {
					inOth = true
					break
				}
			}
			if inOth == (name == "intersect") {
				rt.ktSetAdd(out, e)
			}
		}
		return out, true
	case "toSet", "toMutableSet", "toHashSet":
		return rt.ktSetFrom(keys.elems), true
	case "toSortedSet":
		return rt.ktSetFrom(ktSortBy(rt, keys.elems, nil, false).elems), true
	}
	return rt.ktSeqMethod(keys, name, args)
}

// ktIsCollection reports the shapes kotlin.collections.plus/minus treat as a
// SEQUENCE of elements rather than as one element.
func ktIsCollection(v interface{}) bool {
	if _, isArr := v.(*jsArray); isArr {
		return true
	}
	if ktIsRange(v) {
		return true
	}
	if ktIsSet(v) {
		return true
	}
	_, _, isDict := dictParts(v)
	return isDict
}

// ktSumOf is Kotlin's sumOf/sum, at the WIDTH of what it is summing: a Long
// selector sums to a Long and a Double selector to a Double, where the shared
// js_mcall accumulated in int32 for everything.
func ktSumOf(rt *jsrt, es []interface{}, f interface{}) interface{} {
	isFlo, isLong := false, false
	vals := make([]interface{}, len(es))
	for i, e := range es {
		v := e
		if f != nil {
			v = rt.call(f, jsUndef, []interface{}{e})
		}
		vals[i] = v
		if _, ok := v.(jsJFlo); ok {
			isFlo = true
		}
		if b, ok := v.(jsGInt); ok && b.w == 64 {
			isLong = true
		}
	}
	if isFlo {
		t := 0.0
		for _, v := range vals {
			t += giFloat(rt, v)
		}
		return jsJFlo{f: t}
	}
	var acc int64
	for _, v := range vals {
		acc += giVal(rt, v)
	}
	if isLong {
		return ktNorm(acc, 64, false)
	}
	return ktNorm(acc, 32, false)
}

// ktElems reads the element list out of a list, a string (its Chars) or a map
// (its entries), so flatMap/flatten/zip accept any of them.
func ktElems(rt *jsrt, v interface{}) []interface{} {
	if a, ok := v.(*jsArray); ok {
		return a.elems
	}
	if rg, ok := ktRangeOf(v); ok {
		return ktRangeList(rg)
	}
	if s, ok := v.(string); ok {
		return rt.ktChars(s)
	}
	// A SET iterates its elements; the handle itself is not an array (see ktMakeSet).
	if keys, ok := ktSetParts(v); ok {
		return keys.elems
	}
	if keys, vals, ok := dictParts(v); ok {
		out := make([]interface{}, len(keys.elems))
		for i := range keys.elems {
			out[i] = ktMapEntry(keys.elems[i], vals.elems[i])
		}
		return out
	}
	rt.fail("not a sequence: %s", rt.ktpRender(v, 0))
	return nil
}

// ktMapEntry is one Map.Entry as this value model spells it: key/value for the
// Kotlin member names and first/second so it also destructures as a Pair - but it
// is NOT a Pair, and the __entry marker is what says so.
// java.util.AbstractMap.SimpleEntry.toString is `a=1` where kotlin.Pair.toString is
// `(a, 1)`, and Map.Entry.hashCode is key.hashCode() XOR value.hashCode() where a
// Pair uses the data-class 31-fold. Verified against java (JDK 24): for {a=1, b=2},
// entrySet() prints [a=1, b=2], one entry prints a=1 and hashes to 96, and the
// entry SET hashes to 192, which is the map's own hash. Before the marker,
// `m.entries` printed [(a, 1), (b, 2)] in both halves.
// The twin is kMapEntry in kotlin-interpreter.abnf.
func ktMapEntry(k, v interface{}) *jsObject {
	e := newJSObject()
	e.set("__entry", true)
	e.set("key", k)
	e.set("value", v)
	e.set("first", k)
	e.set("second", v)
	return e
}

func ktIsEntry(v interface{}) bool {
	o, ok := v.(*jsObject)
	if !ok {
		return false
	}
	t, has := o.props["__entry"]
	return has && t == true
}

// ktSetOfDistinct builds a set from elements that are ALREADY distinct (a map's
// keys, its entries), skipping the quadratic uniqueness scan.
func ktSetOfDistinct(es []interface{}) *jsObject {
	s := ktMakeSet()
	keys, _ := ktSetParts(s)
	keys.elems = append(keys.elems, es...)
	return s
}

// ktSortBy is an insertion sort (stable, and small enough to keep the two halves
// byte identical) by an optional selector, ascending or descending.
func ktSortBy(rt *jsrt, es []interface{}, sel interface{}, desc bool) *jsArray {
	out := append([]interface{}{}, es...)
	keys := make([]interface{}, len(out))
	for i, e := range out {
		if sel != nil {
			keys[i] = rt.call(sel, jsUndef, []interface{}{e})
		} else {
			keys[i] = e
		}
	}
	for i := 1; i < len(out); i++ {
		v, k := out[i], keys[i]
		j := i - 1
		for j >= 0 && ktKeyLess(rt, k, keys[j]) != desc && !ktKeyEqual(rt, k, keys[j]) {
			out[j+1], keys[j+1] = out[j], keys[j]
			j--
		}
		out[j+1], keys[j+1] = v, k
	}
	return &jsArray{elems: out}
}

func ktKeyLess(rt *jsrt, a, b interface{}) bool {
	as, aok := a.(string)
	bs, bok := b.(string)
	if aok && bok {
		return as < bs
	}
	if ca, ok := a.(jsChar); ok {
		if cb, ok2 := b.(jsChar); ok2 {
			return ca.code < cb.code
		}
	}
	if ktIsIntegral(a) && ktIsIntegral(b) {
		return rt.ktCmp("<", a, b)
	}
	// A user `Comparable`: `listOf(V(3), V(1)).sorted()` sorts by the class's own
	// compareTo, exactly as `V(1) < V(2)` already did through js_ktcmp. Without this
	// every element compared as giFloat(...) = NaN, no pair was ever "less", and the
	// list came back in its original order - silently, in BOTH halves.
	if ao, isObj := a.(*jsObject); isObj && ktMemberCall != nil && ktClassChainHas(ao, "compareTo") {
		return rt.toNumber(ktMemberCall(a, "compareTo", []interface{}{b})) < 0
	}
	return giFloat(rt, a) < giFloat(rt, b)
}

func ktKeyEqual(rt *jsrt, a, b interface{}) bool {
	return !ktKeyLess(rt, a, b) && !ktKeyLess(rt, b, a)
}

// ktMapMethod is the kotlin.collections Map surface over the shared dict handle.
func (rt *jsrt) ktMapMethod(m *jsObject, name string, args []interface{}) (interface{}, bool) {
	keys, vals, _ := dictParts(m)
	f := argAt(args, 0)
	switch name {
	case "get":
		if i := rt.ktMapFind(keys, f); i >= 0 {
			return vals.elems[i], true
		}
		return jsNull, true
	case "getValue":
		i := rt.ktMapFind(keys, f)
		if i < 0 {
			rt.fail("key %s is missing in the map", rt.ktpRender(f, 0))
		}
		return vals.elems[i], true
	case "getOrDefault":
		if i := rt.ktMapFind(keys, f); i >= 0 {
			return vals.elems[i], true
		}
		return argAt(args, 1), true
	// getOrPut(key) { default }: the value under key, or - when there is none - the
	// lambda's value, STORED under the key and then answered. It is the standard way
	// to build a map of collections (`m.getOrPut(k) { mutableListOf() }.add(v)`) and
	// existed in neither half ("unknown dict method 'getOrPut'"). The twin is the same
	// branch of kMapMethod in languages/kotlin-interpreter.abnf.
	case "getOrPut":
		if i := rt.ktMapFind(keys, f); i >= 0 {
			return vals.elems[i], true
		}
		nv := rt.call(argAt(args, 1), jsUndef, nil)
		rt.ktMapPut(m, f, nv)
		return nv, true
	case "getOrElse":
		if i := rt.ktMapFind(keys, f); i >= 0 {
			return vals.elems[i], true
		}
		return rt.ktCall(argAt(args, 1)), true
	case "put", "set":
		old := interface{}(jsNull)
		if i := rt.ktMapFind(keys, f); i >= 0 {
			old = vals.elems[i]
		}
		rt.ktMapPut(m, f, argAt(args, 1))
		return old, true
	case "remove":
		i := rt.ktMapFind(keys, f)
		if i < 0 {
			return jsNull, true
		}
		old := vals.elems[i]
		keys.dropIdx()
		keys.elems = append(keys.elems[:i], keys.elems[i+1:]...)
		vals.elems = append(vals.elems[:i], vals.elems[i+1:]...)
		return old, true
	case "containsKey":
		return rt.ktMapFind(keys, f) >= 0, true
	case "containsValue":
		for _, v := range vals.elems {
			if rt.ktEqVals(v, f) {
				return true, true
			}
		}
		return false, true
	case "size":
		return float64(len(keys.elems)), true
	case "isEmpty":
		return len(keys.elems) == 0, true
	case "isNotEmpty":
		return len(keys.elems) > 0, true
	case "keys", "toList", "entries", "values":
		switch name {
		case "keys":
			// Map.keys is a SET in Kotlin (java.util.Map.keySet), Map.values a
			// Collection.
			return ktSetOfDistinct(keys.elems), true
		case "values":
			return &jsArray{elems: append([]interface{}{}, vals.elems...)}, true
		case "entries":
			// Map.entries is a SET of entries; Map.toList is a LIST of PAIRS, which
			// is why they no longer share one branch - `m.toList()` prints [(a, 1)]
			// and `m.entries` prints [a=1].
			return ktSetOfDistinct(ktElems(rt, m)), true
		}
		out := make([]interface{}, len(keys.elems))
		for i := range keys.elems {
			p := newJSObject()
			p.set("first", keys.elems[i])
			p.set("second", vals.elems[i])
			out[i] = p
		}
		return &jsArray{elems: out}, true
	case "toMap", "toMutableMap":
		out := ktMakeMap()
		for i := range keys.elems {
			rt.ktMapPut(out, keys.elems[i], vals.elems[i])
		}
		return out, true
	case "clear":
		keys.dropIdx()
		keys.elems = nil
		vals.elems = nil
		return jsUndef, true
	case "hashCode":
		return float64(int32(ktElemHash(rt, m))), true
	case "equals":
		return rt.ktEqVals(m, f), true
	case "filterKeys", "filterValues", "filter", "filterNot", "mapValues", "mapKeys":
		// Every kotlin.collections filter/mapper on a Map answers a MAP. Before this
		// they fell through to the entry-LIST tail below, so `filter` gave a list of
		// Pairs where Kotlin gives a Map, and the four named ones aborted outright.
		out := ktMakeMap()
		for i, k := range keys.elems {
			v := vals.elems[i]
			ent := ktMapEntry(k, v)
			switch name {
			case "filterKeys":
				if rt.truthy(rt.ktCall(f, k)) {
					rt.ktMapPut(out, k, v)
				}
			case "filterValues":
				if rt.truthy(rt.ktCall(f, v)) {
					rt.ktMapPut(out, k, v)
				}
			case "filter":
				if rt.truthy(rt.ktCall(f, ent)) {
					rt.ktMapPut(out, k, v)
				}
			case "filterNot":
				if !rt.truthy(rt.ktCall(f, ent)) {
					rt.ktMapPut(out, k, v)
				}
			case "mapValues":
				rt.ktMapPut(out, k, rt.ktCall(f, ent))
			default:
				rt.ktMapPut(out, rt.ktCall(f, ent), v)
			}
		}
		return out, true
	case "plus", "putAll":
		out := m
		if name == "plus" {
			out = ktMakeMap()
			for i, k := range keys.elems {
				rt.ktMapPut(out, k, vals.elems[i])
			}
		}
		switch add := f.(type) {
		case *jsArray:
			for _, e := range add.elems {
				if p, isPair := ktIsPair(e); isPair {
					rt.ktMapPut(out, p.props["first"], p.props["second"])
				}
			}
		case *jsObject:
			if ak, av, isDict := dictParts(add); isDict {
				for i, k := range ak.elems {
					rt.ktMapPut(out, k, av.elems[i])
				}
			} else if p, isPair := ktIsPair(add); isPair {
				rt.ktMapPut(out, p.props["first"], p.props["second"])
			}
		}
		return out, true
	case "minus":
		drop := []interface{}{f}
		if a, isArr := f.(*jsArray); isArr {
			drop = a.elems
		}
		out := ktMakeMap()
		for i, k := range keys.elems {
			hit := false
			for _, d := range drop {
				if rt.ktEqVals(d, k) {
					hit = true
				}
			}
			if !hit {
				rt.ktMapPut(out, k, vals.elems[i])
			}
		}
		return out, true
	}
	// map/filter/forEach/any/... over the entry sequence.
	return rt.ktSeqMethod(&jsArray{elems: ktElems(rt, m)}, name, args)
}

// ============================================================================
// THE GLOBAL BUILDERS, and the implicit receiver of the this-bound scope functions
//
// A global NAME is declared by each grammar's own builtin block - hostGlobals in
// kotlin-interpreter.abnf, js_scope_decl in kotlin-to-llvm-ir.abnf - so the twelve
// builders below (mapOf/Pair/Triple/with/repeat/StringBuilder/buildString/...) had
// to be given a VALUE the compiler half can bind. js_ktglobal(name) answers that
// value; buildMain declares one per name. Everything they BUILD is a shape whose
// member surface already lives above (ktMapMethod, ktPairMethod, ktSbMethod), so
// the declaration is the whole port.
//
// ktRecvStack is the receiver channel. Kotlin's with / run / apply bind their
// receiver as `this`, so an unqualified `size` or `append("a")` inside the lambda
// is a member call on it. The compiler's lambda is a bare IR closure with no
// receiver slot, so the builder pushes the receiver here for the duration of the
// call and js_ktget consults it AFTER a local and after the enclosing `this` -
// which means it can only ever turn "unknown name" into a member read, never
// change a name that already resolved. The twin is kSetRecv / recvLookup in
// kotlin-interpreter.abnf.
var ktRecvStack []interface{}

// ktMemberCall is js_ktsmcall as a plain Go call, installed by the registrar so the
// bound-builtin closures below reach the full method surface (including the Regex
// and shared js_mcall tails) without duplicating its dispatch.
var ktMemberCall func(recv interface{}, name string, args []interface{}) interface{}

// ktIsType is js_ktis as a plain Go call - the `x is T` test - installed by the
// registrar so filterIsInstance answers exactly what an `is` branch would.
var ktIsType func(v interface{}, tname string) bool

// ktRecvProps are the names Kotlin declares as PROPERTIES on a builtin receiver;
// everything else resolves to a bound method, so a lookup never turns a property
// into a callable. The twin is kRecvProps in kotlin-interpreter.abnf.
// The last four are the properties of the receivers that are neither collections
// nor class instances - kotlin.Lazy's value / isInitialized and kotlin.Result's
// isSuccess / isFailure, plus MatchResult.value, and Char.code. All five are `val`s in the
// stdlib, and no builtin receiver here has a zero-argument METHOD of any of those
// names, which is what makes adding them to this table safe: the table's whole job
// is to say which unqualified names are a field read rather than a bound call.
// Keep it in step with k4RecvProp (languages/lib/kotlin-rt.metajs), kRecvProps
// (languages/kotlin-interpreter.abnf) and kt_recvprop (kotlin-to-llvm-ir.abnf).
func ktRecvProp(name string) bool {
	switch name {
	case "size", "length", "keys", "values", "entries", "indices", "lastIndex",
		"value", "isInitialized", "isSuccess", "isFailure", "code":
		return true
	}
	return false
}

// ----------------------------------------------------------------------------
// PROGRESSIONS - a range is a VALUE, not a materialized list
//
// `a..b`, `a..<b`, `a until b`, `a downTo b` and `... step k` used to be
// MATERIALIZED into a *jsArray by the emitter's own loop, and first / last /
// step / start / endInclusive were recovered from the elements. That recovery
// cannot answer an EMPTY progression at all - `(5..1).first` is 5 in Kotlin (an
// IntRange keeps the bounds it was DECLARED with; only isEmpty() reports the
// emptiness, and `first()`, the FUNCTION, is the one that throws) and it read
// kotlin.Unit in both compiled halves - and it cannot answer toString either,
// which is "5..1" in Kotlin and was "[]". todo.md 1.8 / 1.10.
//
// A progression is now the same first-class value kotlin-interpreter.abnf has
// always had, field for field: {__range, from, to, st, down, ch, fl}. `from` and
// `to` are the INCLUSIVE numeric bounds (..< / until subtract one when the range
// is BUILT), `st` is the POSITIVE step, `down` the direction and `ch` remembers
// that the bounds were Chars - and `fl` that they were Doubles - so an element
// boxes back into whatever the bounds were.
//
// It is a plain *jsObject on purpose: layer 2 has to be able to build the same
// value out of the C floor's primitives, and an object with properties is the
// one shape both engines can hold. Every Kotlin entry point that can receive one
// materializes it with ktRangeList - js_ktiter, js_ktsmcall (after the scope
// functions, so `(1..5).let` still sees the range), js_ktindex, ktElems - while
// the ones where the range-ness IS the answer (js_ktfget, ktpRender, ktIsType,
// ktStructEq, ktElemHash) answer from the bounds. Nothing outside those sites
// sees the object, which is why the shared js_* of abnf/jsrt.go need no arm.
// ----------------------------------------------------------------------------

type ktRange struct {
	from, to, st float64
	down, ch     bool
	// fl marks a ClosedFloatingPointRange - `1.0..2.0`. Kotlin declares no step
	// and no iteration for one, and its contains is the BOUNDS test alone; the
	// old representation materialized it by stepping 1 and answered
	// `1.5 in 1.0..2.0` FALSE in every engine, agreeing and all wrong.
	fl bool
	// pg marks a value that is an IntProgression and NOT an IntRange. `..`, `until`
	// and `..<` build an IntRange; `step` and `reversed()` build an IntProgression,
	// and kotlin.ranges.IntProgression.toString ALWAYS prints " step N" where
	// IntRange.toString never does - so `(1..5) step 1` is "1..5 step 1" and
	// `(5 downTo 1).reversed()` is "1..5 step 1", neither of which the direction
	// and the step alone can tell apart from a plain "1..5". It is also what
	// `x is IntRange` has to answer false to.
	pg bool
}

// IntProgression.reversed(): "a progression that goes over the same range in the
// opposite direction with the same step" - IntProgression.fromClosedRange(last,
// first, -step), so the new FIRST is the old LAST ELEMENT (1..10 step 4 reverses
// to 9 downTo 1 step 4, not 10 downTo 1). kotlin.ranges.reversed.
// The function spelling of rangeTo / until / downTo, shared by the number and Char
// receivers. The `..<` / until subtraction is on the BOUND, exactly as the
// emitter's own lowering does it.
func ktRangeMake(from, to interface{}, name string) *jsObject {
	fn, _ := ktRangeNum(from)
	tn, _ := ktRangeNum(to)
	_, isCh := from.(jsChar)
	_, isFl := from.(jsJFlo)
	if name == "until" {
		tn = tn - 1
	}
	return ktMakeRange(ktRange{from: fn, to: tn, st: 1, down: name == "downTo",
		ch: isCh, fl: isFl})
}

func ktRangeRev(rg ktRange) ktRange {
	return ktRange{from: ktRangeLastEl(rg), to: rg.from, st: rg.st,
		down: !rg.down, ch: rg.ch, pg: true}
}

// The number of elements in a progression, arithmetically: `(1..1000000).random()`
// must not materialize a million cells to pick one.
func ktRangeCount(rg ktRange) int {
	if ktRangeEmpty(rg) {
		return 0
	}
	span := ktRangeLastEl(rg) - rg.from
	if rg.down {
		span = rg.from - ktRangeLastEl(rg)
	}
	return int(math.Floor(span/rg.st)) + 1
}

func ktRangeAt(rg ktRange, i int) interface{} {
	off := rg.st * float64(i)
	if rg.down {
		return ktRangeEl(rg, rg.from-off)
	}
	return ktRangeEl(rg, rg.from+off)
}

// ----------------------------------------------------------------------------
// kotlin.random
//
// THERE IS NO RANDOM SOURCE IN THIS PROJECT and there cannot be one the three
// engines agree on byte-for-byte - the same argument Kernel#rand is settled by in
// ruby-interpreter.abnf. So `random()` draws from a DETERMINISTIC generator with a
// fixed default seed: a program sees a varying sequence (two draws from 1..1000000
// differ, which is what distinguishes this from ruby's "always the front
// element"), every run of every engine sees the SAME sequence, and the matrix's
// byte-identity requirement holds by construction. The particular values are ours,
// not the JVM's - Kotlin's Random(seed) is XorWowRandom - so assert membership and
// bounds, never the value.
//
// xorshift32 in 32-bit arithmetic. The twins are kRndBox in
// kotlin-interpreter.abnf and k4RndBox in languages/lib/kotlin-rt.metajs; all
// three must produce the identical stream, which is why the shifts are spelled
// with an explicit 32-bit reduction rather than the host's own word size.
func ktRndMake(seed float64) *jsObject {
	x := int32(jsToInt(seed)) ^ 1640531527
	if x == 0 {
		x = 1
	}
	o := newJSObject()
	o.set("__ktrnd", true)
	o.set("s", float64(x))
	ktRndNext(o)
	ktRndNext(o)
	return o
}

func ktIsRnd(v interface{}) bool {
	o, isObj := v.(*jsObject)
	if !isObj {
		return false
	}
	b, ok := o.props["__ktrnd"].(bool)
	return ok && b
}

func ktRndNext(o *jsObject) int32 {
	f, _ := o.props["s"].(float64)
	x := int32(f)
	x = x ^ int32(uint32(x)<<13)
	x = x ^ int32(uint32(x)>>17)
	x = x ^ int32(uint32(x)<<5)
	if x == 0 {
		x = 1
	}
	o.set("s", float64(x))
	return x
}

// A non-negative draw below n. The 30-bit mask is what keeps this off an unsigned
// shift, whose answer for a negative operand is the one spelling the three engines
// are least likely to agree on.
func ktRndBelow(o *jsObject, n int) int {
	if n <= 1 {
		return 0
	}
	return int(ktRndNext(o)&1073741823) % n
}

func (rt *jsrt) ktRndOf(args []interface{}) *jsObject {
	if len(args) > 0 && ktIsRnd(args[0]) {
		return args[0].(*jsObject)
	}
	return rt.ktRndDefault()
}

// The default generator, kotlin.random.Random.Default. One per jsrt, so the stream
// a program sees does not depend on how many jsrt values the engine happened to
// build.
func (rt *jsrt) ktRndDefault() *jsObject {
	if rt.ktRndDef == nil {
		rt.ktRndDef = ktRndMake(20260808)
	}
	return rt.ktRndDef
}

// ktRndMethod is the kotlin.random.Random member surface. The twins are the
// kIsRnd branch of mcall in kotlin-interpreter.abnf and k4RndMethod in
// languages/lib/kotlin-rt.metajs.
func (rt *jsrt) ktRndMethod(o *jsObject, name string, args []interface{}) (interface{}, bool) {
	switch name {
	case "nextInt", "nextLong":
		if len(args) == 0 {
			return float64(ktRndNext(o)), true
		}
		lo, hi := float64(0), rt.toNumber(argAt(args, 0))
		if len(args) > 1 {
			lo, hi = hi, rt.toNumber(argAt(args, 1))
		}
		if hi <= lo {
			rt.ktRaise("IllegalArgumentException",
				"Random range is empty: ["+rt.ktpRender(lo, 1)+", "+rt.ktpRender(hi, 1)+").")
		}
		return lo + float64(ktRndBelow(o, int(hi-lo))), true
	case "nextBoolean":
		return ktRndNext(o)&1 == 1, true
	case "nextDouble":
		d := float64(ktRndBelow(o, 1073741824)) / 1073741824
		if len(args) == 1 {
			return jvmMkFlo(d * rt.toNumber(argAt(args, 0))), true
		}
		if len(args) > 1 {
			a, b := rt.toNumber(argAt(args, 0)), rt.toNumber(argAt(args, 1))
			return jvmMkFlo(a + d*(b-a)), true
		}
		return jvmMkFlo(d), true
	case "toString":
		return "Random", true
	}
	return nil, false
}

func ktRangeOf(v interface{}) (ktRange, bool) {
	o, isObj := v.(*jsObject)
	if !isObj {
		return ktRange{}, false
	}
	if b, ok := o.props["__range"].(bool); !ok || !b {
		return ktRange{}, false
	}
	rg := ktRange{}
	rg.from, _ = o.props["from"].(float64)
	rg.to, _ = o.props["to"].(float64)
	rg.st, _ = o.props["st"].(float64)
	rg.down, _ = o.props["down"].(bool)
	rg.ch, _ = o.props["ch"].(bool)
	rg.fl, _ = o.props["fl"].(bool)
	rg.pg, _ = o.props["pg"].(bool)
	return rg, true
}

func ktIsRange(v interface{}) bool {
	_, ok := ktRangeOf(v)
	return ok
}

func ktMakeRange(rg ktRange) *jsObject {
	o := newJSObject()
	o.set("__range", true)
	o.set("from", rg.from)
	o.set("to", rg.to)
	o.set("st", rg.st)
	o.set("down", rg.down)
	o.set("ch", rg.ch)
	o.set("fl", rg.fl)
	o.set("pg", rg.pg)
	return o
}

// ktRangeNum is the numeric reading of a bound: a Char bound progresses in code
// points and a sized integer in its own value. The twin is k4RangeNum.
func ktRangeNum(v interface{}) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case jsChar:
		return float64(t.code), true
	case jsGInt:
		if t.u && t.w == 64 {
			return float64(uint64(t.v)), true
		}
		return float64(t.v), true
	case jsJFlo:
		return t.f, true
	}
	return 0, false
}

func ktRangeEl(rg ktRange, n float64) interface{} {
	if rg.ch {
		return jsChar{code: int32(n)}
	}
	if rg.fl {
		return jvmMkFlo(n)
	}
	return n
}

// The LAST ELEMENT of a progression, which is what IntProgression.last answers -
// getProgressionLastElement in kotlin.internal.ProgressionUtil. For a step of 1
// it is the bound; for `1..10 step 4` the elements are 1, 5, 9 and the answer is
// 9. An EMPTY progression answers the bound, exactly as
// getProgressionLastElement does when first and last are already crossed.
func ktRangeLastEl(rg ktRange) float64 {
	// A floating range has no elements, so its endInclusive IS the bound.
	if rg.fl {
		return rg.to
	}
	if rg.down {
		if rg.from <= rg.to {
			return rg.to
		}
		return rg.to + math.Mod(rg.from-rg.to, rg.st)
	}
	if rg.from >= rg.to {
		return rg.to
	}
	return rg.to - math.Mod(rg.to-rg.from, rg.st)
}

func ktRangeEmpty(rg ktRange) bool {
	if rg.down {
		return rg.from < rg.to
	}
	return rg.from > rg.to
}

func ktRangeList(rg ktRange) []interface{} {
	out := []interface{}{}
	step := rg.st
	if rg.down {
		step = -step
	}
	for i := rg.from; ktRangeGoes(rg, i); i += step {
		out = append(out, ktRangeEl(rg, i))
	}
	return out
}

func ktRangeGoes(rg ktRange, i float64) bool {
	if rg.down {
		return i >= rg.to
	}
	return i <= rg.to
}

// `x in r`. A plain IntRange declares contains(value) and consults the bounds
// only; a STEPPED progression is an Iterable and `in` is its linear membership,
// so 3 is NOT in (1..10 step 4). Both are the one test below.
func ktRangeHas(rg ktRange, x interface{}) bool {
	// A ClosedFloatingPointRange consults the bounds and nothing else.
	if rg.fl {
		if _, isCh := x.(jsChar); isCh {
			return false
		}
		n, ok := ktRangeNum(x)
		return ok && n >= rg.from && n <= rg.to
	}
	if _, isCh := x.(jsChar); rg.ch != isCh {
		return false
	}
	n, ok := ktRangeNum(x)
	if !ok {
		return false
	}
	if rg.down {
		if n > rg.from || n < rg.to {
			return false
		}
		return math.Mod(rg.from-n, rg.st) == 0
	}
	if n < rg.from || n > rg.to {
		return false
	}
	return math.Mod(n-rg.from, rg.st) == 0
}

func ktRangeSigned(rg ktRange) float64 {
	if rg.down {
		return -rg.st
	}
	return rg.st
}

// IntRange.toString is "$first..$last"; IntProgression.toString is
// "$first..$last step $step" when the step is positive and
// "$first downTo $last step ${-step}" when it is negative. CharRange overrides
// with the IntRange form. A plain a..b IS an IntRange, so the bare form is the
// one a step-less, non-downTo progression prints.
func (rt *jsrt) ktRangeStr(rg ktRange) string {
	// ClosedFloatingPointRange.toString is "$start..$endInclusive" - the declared
	// BOUND, not a last element, since it has no step and no elements.
	if rg.fl {
		return rt.ktpRender(jvmMkFlo(rg.from), 1) + ".." + rt.ktpRender(jvmMkFlo(rg.to), 1)
	}
	a := rt.ktpRender(ktRangeEl(rg, rg.from), 1)
	b := rt.ktpRender(ktRangeEl(rg, ktRangeLastEl(rg)), 1)
	if rg.down {
		return a + " downTo " + b + " step " + rt.ktpRender(rg.st, 1)
	}
	if rg.st != 1 || rg.pg {
		return a + ".." + b + " step " + rt.ktpRender(rg.st, 1)
	}
	return a + ".." + b
}

// IntRange.equals: two EMPTY progressions are equal whatever their bounds, and
// two non-empty ones agree when first, last and step agree.
func ktRangeEq(a, b ktRange) bool {
	if ktRangeEmpty(a) {
		return ktRangeEmpty(b)
	}
	if ktRangeEmpty(b) {
		return false
	}
	return a.from == b.from && ktRangeLastEl(a) == ktRangeLastEl(b) &&
		ktRangeSigned(a) == ktRangeSigned(b)
}

// hashCode is -1 for an empty progression - kotlin.ranges.IntProgression's own
// contract, and what makes it agree with the equals above.
func ktRangeHash(rg ktRange) int32 {
	if ktRangeEmpty(rg) {
		return -1
	}
	h := int32(31)*int32(rg.from) + int32(ktRangeLastEl(rg))
	return int32(31)*h + int32(ktRangeSigned(rg))
}

// `r is T`. A plain a..b is an IntRange (a CharRange when the bounds were Chars)
// and therefore an IntProgression too; a STEPPED or downTo progression is an
// IntProgression and NOT an IntRange, which is what `10 downTo 1 is IntRange`
// answers false to on the JVM. A range is an Iterable and is NOT a List.
// A LongRange is not distinguishable here: ktRangeNum reduces a Long bound to
// its numeric value, so `1L..5L` and `1..5` are the same value.
func ktRangeIsType(rg ktRange, t string) bool {
	if t == "Any" {
		return true
	}
	if rg.fl {
		// A floating range is a ClosedRange and NOT an Iterable.
		return t == "ClosedFloatingPointRange" || t == "ClosedRange"
	}
	if t == "Iterable" {
		return true
	}
	plain := rg.st == 1 && !rg.down && !rg.pg
	if rg.ch {
		if t == "CharProgression" {
			return true
		}
		return t == "CharRange" && plain
	}
	if t == "IntProgression" {
		return true
	}
	return t == "IntRange" && plain
}

// ktIsBuiltinRecv reports a receiver whose members come from the runtime rather
// than from a class descriptor: a list, a map, a SET, a StringBuilder box or a
// String. The set was missing in every engine, so `with(setOf(1, 2)) { size }` and
// `{ contains(1) }` aborted with "unknown name" in both halves alike - agreement
// is not correctness, and only an oracle can see this class. Kotlin's `with` binds
// any receiver and Set.size is Collection.size.
//
// A REGEX, a MATCHRESULT, a RESULT and a CHAR were missing, in all three engines,
// so `with(Regex("a")) { pattern }`, an unqualified `find()`, `with(runCatching
// { 7 }) { getOrNull() }` and `with('q') { isLetter() }` reached nothing anywhere -
// todo.md 1.6's second bullet and the sweep around it. Each of the four has a real
// method surface in js_ktsmcall and a real property surface in js_ktfget, which is
// exactly what this predicate is asking about; the plain-object arm below reads own
// properties only and binds no method at all, which is why they could not stay
// there. The twins are k4IsBuiltinRecv (languages/lib/kotlin-rt.metajs) and
// recvLookup's shape list (languages/kotlin-interpreter.abnf).
func ktIsBuiltinRecv(v interface{}) bool {
	switch t := v.(type) {
	case *jsArray, string, jsChar:
		return true
	case *jsObject:
		if _, isCls := t.props["__class"]; isCls {
			return false
		}
		if _, isClsDesc := t.props["__isclass"]; isClsDesc {
			return false
		}
		// A PROGRESSION: first / last / step / start / endInclusive are members.
		if ktIsRange(t) {
			return true
		}
		if _, _, isDict := dictParts(t); isDict {
			return true
		}
		if ktIsSet(t) {
			return true
		}
		if ktRxIsRegex(t) || ktRxIsMatch(t) {
			return true
		}
		if _, isRes := ktIsResult(t); isRes {
			return true
		}
		return ktIsSb(t)
	}
	return false
}

// ktRecvMember answers an unqualified name against an implicit receiver.
//
// `iscall` is the SYNTACTIC POSITION of the name: true for `first()`, false for a
// bare `first`. todo.md 1.6 concluded that `first`/`last` - a PROPERTY of
// IntProgression and a member FUNCTION of List, over a range this project
// materializes as a plain list - needed the receiver's ORIGIN carried on the value.
// It needs the SITE instead: Kotlin has no unqualified name in value position that
// means a bound method (that spelling is `::name`), so a builtin receiver's field
// read is tried first in value position and not at all in call position. That
// answers `with(1..7 step 2) { first }` as 1 and leaves `with(listOf(4, 5))
// { first() }` binding the method, which assertion kr9 pins. Carrying the origin
// was also not available: a jsArray holds no properties at all (setMember rejects
// every non-numeric key but `length`) and the C floor's array is the same shape.
// The twins are ktRecvMember in languages/lib/kotlin-rt.metajs and recvLookup in
// languages/kotlin-interpreter.abnf.
func (rt *jsrt) ktRecvMember(recv interface{}, name string, iscall bool) (interface{}, bool) {
	// A USER-CLASS instance reached through with / run / apply. Its FIELDS are its own
	// properties; its METHODS live on the descriptor chain and come back as a bound
	// callable, so an unqualified `method()` inside the lambda dispatches on the
	// receiver. recvLookup in kotlin-interpreter.abnf answers exactly this set.
	if o, isObj := recv.(*jsObject); isObj && ktMemberCall != nil {
		if _, hasCls := o.props["__class"]; hasCls {
			if v, ok := o.props[name]; ok {
				return v, true
			}
			// A property with a GETTER: the accessor pair lives on the descriptor
			// chain (js_defprop), not on the instance, so findClassAccessor is what
			// sees it - reading it through the method branch below would hand back a
			// callable instead of the value.
			if acc := rt.findClassAccessor(o, name); acc != nil {
				if acc.get == nil {
					return jsUndef, true
				}
				return rt.call(acc.get, o, nil), true
			}
			if ktClassChainHas(o, name) {
				self := recv
				return jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
					return ktMemberCall(self, name, args)
				}), true
			}
			return nil, false
		}
	}
	if !ktIsBuiltinRecv(recv) || ktMemberCall == nil {
		// A PLAIN OBJECT receiver that is neither a class instance nor one of the
		// collection kinds: a Pair, a Triple, a Map.Entry, a lazy, a Result, a
		// MatchResult. Kotlin's `with` binds any receiver at all, and every one of
		// those carries real PROPERTIES - first / second / third, key / value,
		// value / isInitialized, isSuccess / isFailure. Its own properties answer
		// the first group and js_ktfget the second, which is exactly what the
		// emitter's `this` probe does for the same receiver in
		// kotlin-to-llvm-ir.abnf's kt_ktget (own read, then the recv-property gate),
		// so the three engines agree by construction rather than by coincidence.
		// Nothing is bound as a METHOD here: a plain object has no runtime method
		// table to dispatch into, and binding one would swallow every global name.
		if o, isObj := recv.(*jsObject); isObj && ktMemberCall != nil {
			if _, isClsDesc := o.props["__isclass"]; !isClsDesc {
				if v, ok := o.props[name]; ok {
					return v, true
				}
				// The two receiver shapes js_ktfget ABORTS on rather than missing
				// (kt_fprobe in kotlin-to-llvm-ir.abnf carries the same guard).
				_, isClsLit := ktIsClassLit(o)
				_, isPkg := ktIsPkg(o)
				if (ktRecvProp(name) || !iscall) && ktRefRead != nil && !isClsLit && !isPkg {
					if v := ktRefRead(recv, name); !isUndefOrNull(v) {
						return v, true
					}
				}
			}
		}
		// toString(), hashCode() and equals() are members of kotlin.Any, so an
		// unqualified CALL of one resolves against the implicit receiver whatever
		// its shape - a Pair, a Map.Entry, a number. `with(p) { toString() }`
		// aborted with
		// "unknown name: toString" here (todo.md 1.10); js_ktsmcall's own Any arm
		// answers it once the name resolves at all.
		if (name == "toString" || name == "hashCode" || name == "equals") && iscall && ktMemberCall != nil {
			self := recv
			return jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
				return ktMemberCall(self, name, args)
			}), true
		}
		return nil, false
	}
	// A PROPERTY of the receiver is a FIELD READ, not a zero-argument method call.
	// This used to route through ktMemberCall, and the two are not the same set:
	// `with(list) { size }` worked because the list method table happens to carry a
	// `size` method, while `with(list) { lastIndex }` died with "unknown list method
	// 'lastIndex'" - and `indices` would have, once it existed. The twin has always
	// read kGetField here (kRecvProps in kotlin-interpreter.abnf), so this is the Go
	// side catching up rather than a new rule.
	if ktRecvProp(name) && ktRefRead != nil {
		return ktRefRead(recv, name), true
	}
	// VALUE POSITION on a builtin receiver: the field read is TRIED, and only a miss
	// falls through to the bound method below - which is where every name that is not
	// a property of this receiver still lands, so nothing that resolved stops
	// resolving. This is the arm that answers `with(1..7 step 2) { first }`.
	if !iscall && ktRefRead != nil {
		if v := ktRefRead(recv, name); !isUndefOrNull(v) {
			return v, true
		}
	}
	self := recv
	return jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return ktMemberCall(self, name, args)
	}), true
}

// ktMakeSb builds the StringBuilder box, {__sb: true, s: "..."}, the same shape
// kotlin-interpreter.abnf builds.
func ktMakeSb(s string) *jsObject {
	o := newJSObject()
	o.set("__sb", true)
	o.set("s", s)
	return o
}

func ktIsSb(v interface{}) bool {
	o, ok := v.(*jsObject)
	if !ok {
		return false
	}
	b, has := o.props["__sb"]
	return has && b == true
}

func ktSbText(o *jsObject) string {
	s, _ := o.props["s"].(string)
	return s
}

// ktSbMethod is the StringBuilder member surface, the Go twin of the `target.__sb`
// branch of `mcall` in kotlin-interpreter.abnf. A name it does not know falls
// through to the String methods on the accumulated text, exactly as there.
func (rt *jsrt) ktSbMethod(o *jsObject, name string, args []interface{}) (interface{}, bool) {
	switch name {
	case "append":
		s := ktSbText(o)
		for _, a := range args {
			s += rt.ktStr2(a)
		}
		o.set("s", s)
		return o, true
	case "appendLine":
		s := ktSbText(o)
		for _, a := range args {
			s += rt.ktStr2(a)
		}
		o.set("s", s+"\n")
		return o, true
	case "toString":
		return ktSbText(o), true
	case "length":
		return float64(len([]rune(ktSbText(o)))), true
	case "isEmpty":
		return len(ktSbText(o)) == 0, true
	case "isNotEmpty":
		return len(ktSbText(o)) > 0, true
	case "clear":
		o.set("s", "")
		return o, true
	case "reverse":
		r := []rune(ktSbText(o))
		for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
			r[i], r[j] = r[j], r[i]
		}
		o.set("s", string(r))
		return o, true
	}
	return rt.ktStrMethod(ktSbText(o), name, args)
}

// ktFieldTy answers the declared type of a property, walking the __class / __super
// chain for the __ptypes table the class emitter installs. "" when nothing declares
// one, which makes the adoption a no-op. The twin is kFieldTy in
// kotlin-interpreter.abnf.
func ktFieldTy(o *jsObject, name string) string {
	cls, _ := o.props["__class"].(*jsObject)
	for i := 0; cls != nil && i < 64; i++ {
		if pt, ok := cls.props["__ptypes"].(*jsObject); ok {
			if ty, has := pt.props[name].(string); has && ty != "" {
				return ty
			}
		}
		cls, _ = cls.props["__super"].(*jsObject)
	}
	return ""
}

// ktAdoptTy retypes a value by a declared type name, or answers nil when the type
// carries no width. The width table is the one ktDeclWidth spells in both grammars.
func (rt *jsrt) ktAdoptTy(v interface{}, ty string) interface{} {
	t := strings.TrimSuffix(ty, "?")
	if !ktIsIntegral(v) {
		return nil
	}
	switch t {
	case "Double":
		return jsJFlo{f: giFloat(rt, v)}
	case "Float":
		// A `Float` declared type is a BINARY32, not a print style - see the
		// float WIDTH block in languages/lib/kotlin-rt.metajs. Kotlin/JVM's
		// Float IS Java's, so the machinery is jsrtjvm.go's floJavaF.
		return jsJFlo{f: jvmFround(giFloat(rt, v)), sty: floJavaF}
	case "Int":
		return rt.ktConv(v, 32, false)
	case "Long":
		return rt.ktConv(v, 64, false)
	case "Byte":
		return rt.ktConv(v, 8, false)
	case "Short":
		return rt.ktConv(v, 16, false)
	case "UInt":
		return rt.ktConv(v, 32, true)
	case "ULong":
		return rt.ktConv(v, 64, true)
	case "UByte":
		return rt.ktConv(v, 8, true)
	case "UShort":
		return rt.ktConv(v, 16, true)
	}
	return nil
}

// ktMapOf builds a map from mapOf's arguments: each is a Pair, or (after a spread)
// an array of Pairs.
func (rt *jsrt) ktMapOf(args []interface{}) *jsObject {
	mp := ktMakeMap()
	for _, p := range args {
		if po, ok := ktIsPair(p); ok {
			rt.ktMapPut(mp, po.props["first"], po.props["second"])
			continue
		}
		if arr, ok := p.(*jsArray); ok {
			for _, e := range arr.elems {
				if po, ok2 := ktIsPair(e); ok2 {
					rt.ktMapPut(mp, po.props["first"], po.props["second"])
				}
			}
		}
	}
	return mp
}

// ktWithRecv runs f with `recv` as the lambda's implicit receiver.
//
// It does that TWO ways, and both are needed. The receiver goes on ktRecvStack, which
// is what answers a BUILTIN receiver's members (`with(sb) { append("a") }`). And, when
// the lambda is a compiled closure, the call runs in a fresh scope layered on the
// closure's own env with `this` bound to the receiver - which is exactly the model
// kotlin-interpreter.abnf uses (its `with` binds `this` and nothing else), and which is
// what makes Kotlin's INNERMOST-WINS rule fall out of the ordinary scope walk in
// js_ktget: a `with(i) { tag }` written inside a member of another class used to answer
// the ENCLOSING this's `tag`, because the stack was consulted only after it. Measured
// before the change: `with(i) { tag }` in Outer.show printed OUTER here and INNER in the
// interpreter half - the halves disagreed and this half was the wrong one.
func (rt *jsrt) ktWithRecv(recv interface{}, f interface{}, args ...interface{}) interface{} {
	if !isCallable(f) {
		return jsUndef
	}
	ktRecvStack = append(ktRecvStack, recv)
	defer func() { ktRecvStack = ktRecvStack[:len(ktRecvStack)-1] }()
	if cl, ok := f.(*jsClosure); ok {
		inner := &jsScope{parent: rt.scopeOf(cl.env)}
		inner.put("this", recv)
		bound := &jsClosure{fn: cl.fn, env: rt.wrap(inner), ma: cl.ma}
		return rt.call(bound, jsUndef, args)
	}
	return rt.call(f, jsUndef, args)
}

// ----------------------------------------------------------------------------
// THE QUALIFIED STDLIB PATHS
//
// `kotlin.math.abs(-3)` is the same function as the unqualified `abs(-3)`, and it
// resolved in NEITHER half before ("unknown name: kotlin") - the builtins are
// global names, so only the bare spelling reached them. `kotlin` is a value now: a
// package handle whose field read either descends into a known sub-package or
// answers the GLOBAL of that name. Membership is deliberately not modelled per
// package - the last segment is looked up wherever it lives - because a wrong
// membership table would REJECT correct code, and Kotlin's own resolution has
// already decided the path is valid by the time a program runs.
// `import kotlin.math.abs` needed nothing: an import of a builtin is already a
// no-op in both halves.
// The twin is kPkg / kPkgMember in languages/kotlin-interpreter.abnf.
var ktPkgPaths = map[string]bool{
	"kotlin": true, "kotlin.math": true, "kotlin.collections": true,
	"kotlin.text": true, "kotlin.sequences": true, "kotlin.comparisons": true,
	"kotlin.ranges": true, "kotlin.system": true, "kotlin.io": true,
	"kotlin.random": true, "kotlin.jvm": true, "kotlin.reflect": true,
	"kotlin.time": true, "kotlin.concurrent": true,
}

func ktPkg(path string) *jsObject {
	o := newJSObject()
	o.set("__pkg", path)
	return o
}

func ktIsPkg(v interface{}) (string, bool) {
	o, ok := v.(*jsObject)
	if !ok {
		return "", false
	}
	p, isStr := o.props["__pkg"].(string)
	return p, isStr
}

// ktPkgMember descends into a sub-package or answers the global of that name.
func (rt *jsrt) ktPkgMember(pkg, name string) interface{} {
	sub := pkg + "." + name
	if ktPkgPaths[sub] {
		return ktPkg(sub)
	}
	if v := ktGlobalFn(rt, name); !isUndefOrNull(v) {
		return v
	}
	rt.fail("unknown name: %s", sub)
	return nil
}

// ktGlobalFn is the VALUE of one global builder name.
func ktGlobalFn(rt *jsrt, name string) interface{} {
	switch name {
	case "Delegates":
		// kotlin.properties.Delegates: the three standard property delegates. Each
		// answers a box js_ktdget / js_ktdset drive (see ktDelegKind).
		d := newJSObject()
		d.set("observable", jsHostFunc("observable", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return ktDelegBox("observable", argAt(args, 0), argAt(args, 1))
		}))
		d.set("vetoable", jsHostFunc("vetoable", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return ktDelegBox("vetoable", argAt(args, 0), argAt(args, 1))
		}))
		d.set("notNull", jsHostFunc("notNull", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return ktDelegBox("notNull", jsNull, nil)
		}))
		return d
	case "mapOf", "mutableMapOf", "hashMapOf", "linkedMapOf", "sortedMapOf", "emptyMap":
		return jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return rt.ktMapOf(args)
		})
	case "buildMap":
		return jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			mp := ktMakeMap()
			rt.ktWithRecv(mp, argAt(args, 0), mp)
			return mp
		})
	case "buildList":
		return jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			l := &jsArray{}
			rt.ktWithRecv(l, argAt(args, 0), l)
			return l
		})
	case "buildString":
		return jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			sb := ktMakeSb("")
			rt.ktWithRecv(sb, argAt(args, 0), sb)
			return ktSbText(sb)
		})
	case "Pair":
		return jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			o := newJSObject()
			o.set("first", argAt(args, 0))
			o.set("second", argAt(args, 1))
			return o
		})
	case "Triple":
		return jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			o := newJSObject()
			o.set("first", argAt(args, 0))
			o.set("second", argAt(args, 1))
			o.set("third", argAt(args, 2))
			return o
		})
	case "with":
		// with(receiver) { ... }: the scope function that is a top-level FUNCTION
		// rather than an extension. The receiver is `this` AND the lambda's single
		// argument, which is what the interpreter half does too.
		return jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			recv := argAt(args, 0)
			return rt.ktWithRecv(recv, argAt(args, 1), recv)
		})
	case "run":
		// The NULLARY global run { ... }; the member form x.run { } is ktScopeMethod's.
		return jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			f := argAt(args, 0)
			if !isCallable(f) {
				return jsUndef
			}
			return rt.ktCall(f)
		})
	case "repeat":
		return jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			n := int(rt.toNumber(argAt(args, 0)))
			f := argAt(args, 1)
			for i := 0; i < n; i++ {
				rt.ktCall(f, float64(i))
			}
			return jsUndef
		})
	case "StringBuilder":
		return jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			init := argAt(args, 0)
			if isUndefOrNull(init) {
				return ktMakeSb("")
			}
			return ktMakeSb(rt.ktStr2(init))
		})
	// kotlin.random.Random(seed) - the SEEDED generator. `Random` with no argument
	// is not a Kotlin spelling (Random.Default is), so a missing seed answers the
	// default stream. See the kotlin.random block for why the stream is
	// deterministic in all three engines.
	case "Random":
		return jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			seed := argAt(args, 0)
			if isUndefOrNull(seed) {
				return rt.ktRndDefault()
			}
			return ktRndMake(rt.toNumber(seed))
		})
	// ----- The kotlin stdlib top-level functions the subset was missing. Each aborted
	// the run with "unknown name" in BOTH halves; the twins are the hostGlobals block
	// in kotlin-interpreter.abnf.
	// A COMPARATOR is a plain function of two elements answering an Int, which is what
	// sortedWith drives.
	case "compareBy", "compareByDescending":
		desc := name == "compareByDescending"
		return jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			sel := argAt(args, 0)
			return jsHostFunc("comparator", func(rt *jsrt, this uint64, ab []interface{}) interface{} {
				ka, kb := rt.ktCall(sel, argAt(ab, 0)), rt.ktCall(sel, argAt(ab, 1))
				return float64(ktCmpKeys(rt, ka, kb, desc))
			})
		})
	case "Comparator":
		return jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return argAt(args, 0)
		})
	case "compareValues":
		return jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			a, b := argAt(args, 0), argAt(args, 1)
			if isUndefOrNull(a) {
				if isUndefOrNull(b) {
					return float64(0)
				}
				return float64(-1)
			}
			if isUndefOrNull(b) {
				return float64(1)
			}
			return float64(ktCmpKeys(rt, a, b, false))
		})
	// kotlin.math.abs: the operand's own TYPE, so abs(-1.5) is the Double 1.5 and
	// abs(-3L) a Long. JavaScript's Math.abs, which this used to be bound to, answers
	// a plain number - so `max(1.5, 2.0)` printed `2` where Kotlin prints `2.0`.
	case "abs":
		return jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			v := argAt(args, 0)
			if f, isFlo := v.(jsJFlo); isFlo {
				return jsJFlo{f: math.Abs(f.f)}
			}
			if rt.ktNumLess(v, float64(0)) {
				return rt.ktArith("-", ktNorm(0, 32, false), v)
			}
			return v
		})
	case "max", "min", "maxOf", "minOf":
		wantMax := name == "maxOf" || name == "max"
		return jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			best := argAt(args, 0)
			for _, v := range args[1:] {
				if ktKeyLess(rt, best, v) == wantMax {
					best = v
				}
			}
			return best
		})
	// The contract functions. Kotlin throws IllegalArgumentException from require and
	// IllegalStateException from check.
	case "require", "check", "requireNotNull", "checkNotNull", "error", "TODO":
		return jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			switch name {
			case "error":
				rt.ktRaise("IllegalStateException", rt.ktStr2(argAt(args, 0)))
			case "TODO":
				msg := "An operation is not implemented."
				if len(args) > 0 && !isUndefOrNull(args[0]) {
					msg = "An operation is not implemented: " + rt.ktStr2(args[0])
				}
				rt.ktRaise("NotImplementedError", msg)
			}
			cls, dflt := "IllegalArgumentException", "Failed requirement."
			if name == "check" || name == "checkNotNull" {
				cls, dflt = "IllegalStateException", "Check failed."
			}
			ok := rt.truthy(argAt(args, 0))
			if name == "requireNotNull" || name == "checkNotNull" {
				ok = !isUndefOrNull(argAt(args, 0))
				dflt = "Required value was null."
			}
			if !ok {
				text := dflt
				if len(args) > 1 && !isUndefOrNull(args[1]) {
					if isCallable(args[1]) {
						text = rt.ktStr2(rt.ktCall(args[1]))
					} else {
						text = rt.ktStr2(args[1])
					}
				}
				rt.ktRaise(cls, text)
			}
			if name == "requireNotNull" || name == "checkNotNull" {
				return argAt(args, 0)
			}
			return jsUndef
		})
	// runCatching { ... }: Kotlin's Result, the {__result, value, exc} box whose
	// members ktResultMethod answers.
	case "runCatching":
		return jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return rt.ktRunCatching(argAt(args, 0))
		})
	case "List", "MutableList", "Array", "IntArray", "LongArray", "ShortArray",
		"ByteArray", "DoubleArray", "FloatArray", "BooleanArray", "CharArray",
		"UByteArray", "UShortArray", "UIntArray", "ULongArray", "arrayOfNulls":
		return jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			n := jsToInt(rt.toNumber(argAt(args, 0)))
			init := argAt(args, 1)
			out := &jsArray{}
			for i := 0; i < n; i++ {
				switch {
				case isCallable(init):
					out.elems = append(out.elems, rt.ktCall(init, float64(i)))
				case name == "BooleanArray":
					out.elems = append(out.elems, false)
				case name == "CharArray":
					// A Char is a real type here, so CharArray(n) fills with
					// Kotlin's NUL char, not with the number 0.
					out.elems = append(out.elems, jsChar{code: 0})
				case name == "List" || name == "MutableList" || name == "Array" || name == "arrayOfNulls":
					out.elems = append(out.elems, jsNull)
				default:
					out.elems = append(out.elems, float64(0))
				}
			}
			return out
		})
	case "listOfNotNull":
		// listOf's "return the argument array" function is WRONG here: it kept the
		// nulls, so the compiler answered [1, null, 2] where Kotlin (and now the
		// interpreter half) answers [1, 2].
		return jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			out := &jsArray{}
			for _, v := range args {
				if !isUndefOrNull(v) {
					out.elems = append(out.elems, v)
				}
			}
			return out
		})
	// The SET constructors. They used to be aliases of listOf in the grammar's
	// builtin block, which is what made a set a list; a set is its own shape now
	// (see ktMakeSet), so they build one and deduplicate.
	// The LIST constructors. The grammar declares them as an IR closure that simply
	// returns its argument array; they are here too so a QUALIFIED spelling
	// (kotlin.collections.listOf) resolves through ktPkgMember, which has no reach
	// into the emitted scope.
	case "listOf", "mutableListOf", "arrayListOf", "arrayOf", "emptyList", "emptyArray",
		"intArrayOf", "longArrayOf", "doubleArrayOf", "floatArrayOf",
		"booleanArrayOf", "charArrayOf", "byteArrayOf", "shortArrayOf",
		"ubyteArrayOf", "ushortArrayOf", "uintArrayOf", "ulongArrayOf",
		"sequenceOf", "emptySequence":
		return jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return &jsArray{elems: append([]interface{}{}, args...)}
		})
	case "setOf", "mutableSetOf", "hashSetOf", "linkedSetOf":
		return jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return rt.ktSetFrom(args)
		})
	case "sortedSetOf":
		// A java.util.TreeSet: ascending order, not insertion order.
		return jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return rt.ktSetFrom(ktSortBy(rt, args, nil, false).elems)
		})
	case "emptySet":
		return jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return ktMakeSet()
		})
	case "setOfNotNull":
		return jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			out := ktMakeSet()
			for _, v := range args {
				if !isUndefOrNull(v) {
					rt.ktSetAdd(out, v)
				}
			}
			return out
		})

	// ----- kotlinx.coroutines, as a SYNCHRONOUS subset -----
	//
	// `launch`, `async`, `runBlocking`, `delay` and the scope builders were UNDEFINED
	// in both halves, so any program with a coroutine in it stopped at the import.
	// They run their block IMMEDIATELY and to completion here, on the calling
	// goroutine: runBlocking { } is its body, async { } computes its value at the
	// point it is written and await() hands it back, launch { } is the block itself,
	// delay() is a no-op, and withContext / coroutineScope / supervisorScope are
	// their bodies.
	//
	// THE LIMITATION, stated the way the JS async subset states its own: the RESULTS
	// are right and the ORDERING is not. Nothing suspends, so no two coroutines
	// interleave - a `launch` block runs before the statement after it rather than
	// concurrently, `delay` does not yield, and a program whose output DEPENDS on
	// interleaving prints a different (but deterministic) order than the real
	// kotlinx.coroutines runtime. Programs that use coroutines to compute a value -
	// the common case - get the right answer.
	//
	// Not modelled at all, and still reported as unknown: Flow, Channel, select,
	// cancellation semantics, and any dispatcher behaviour beyond the name.
	// The twin is the coroutine block of hostGlobals in kotlin-interpreter.abnf.
	case "runBlocking", "coroutineScope", "supervisorScope", "withContext":
		return jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return rt.ktCoroRun(args)
		})
	case "async", "launch":
		return jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return ktMakeJob(rt.ktCoroRun(args))
		})
	case "delay", "yield":
		return jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return jsUndef
		})
	case "Job":
		return jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return ktMakeJob(jsUndef)
		})
	case "CoroutineScope":
		return jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			o := newJSObject()
			o.set("__coroscope", true)
			return o
		})
	case "awaitAll":
		return jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			out := &jsArray{}
			for _, a := range args {
				if j, isJob := ktIsJob(a); isJob {
					out.elems = append(out.elems, j.props["v"])
				} else {
					out.elems = append(out.elems, a)
				}
			}
			return out
		})
	case "Dispatchers":
		// Dispatchers.IO / Main / Default / Unconfined are NAMES here: nothing
		// dispatches, so they carry no behaviour, but a program that mentions one
		// still runs.
		d := newJSObject()
		for _, n := range []string{"IO", "Main", "Default", "Unconfined"} {
			d.set(n, "Dispatchers."+n)
		}
		return d
	case "lazy":
		return jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			o := newJSObject()
			o.set("__lazy", true)
			o.set("f", argAt(args, 0))
			o.set("done", false)
			o.set("v", jsNull)
			return o
		})
	case "generateSequence":
		// A Sequence is EAGER here (see the note on sequenceOf in the grammar), so an
		// infinite generator is capped rather than lazy.
		return jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			seed, next := argAt(args, 0), argAt(args, 1)
			var v interface{}
			if isCallable(seed) {
				v = rt.ktCall(seed)
			} else {
				v = seed
			}
			out := &jsArray{}
			guard := 0
			for ; guard < ktSeqCap && !isUndefOrNull(v); guard++ {
				out.elems = append(out.elems, v)
				if !isCallable(next) {
					break
				}
				v = rt.ktCall(next, v)
			}
			// The cap is a REAL limit, not a formality: an eager Sequence cannot
			// represent an infinite generator, so say so instead of handing back a
			// silently truncated list. take(n) on an infinite generator still gets
			// the right answer - it just paid for 100000 elements to get there.
			if guard >= ktSeqCap {
				fmt.Fprint(warnWriter, "warning: generateSequence is EAGER here and stopped at "+
					strconv.Itoa(ktSeqCap)+" elements; a Sequence is not lazy in this subset\n")
			}
			return out
		})
	}
	return jsUndef
}

// ktSeqCap is how far an EAGER generateSequence walks an infinite generator before
// it gives up. See the :description note in kotlin-to-llvm-ir.abnf; the twin is
// kSeqCap in kotlin-interpreter.abnf.
const ktSeqCap = 100000

// ktCoroRun runs the LAST callable argument - `launch(Dispatchers.IO) { ... }` puts
// the context first and the block last - and answers its value.
func (rt *jsrt) ktCoroRun(args []interface{}) interface{} {
	for i := len(args) - 1; i >= 0; i-- {
		if isCallable(args[i]) {
			return rt.call(args[i], jsUndef, nil)
		}
	}
	return jsUndef
}

// ktMakeJob is the Job / Deferred an `async` or `launch` answers. The block has
// already run, so the completed value is carried right here.
func ktMakeJob(v interface{}) *jsObject {
	o := newJSObject()
	o.set("__job", true)
	o.set("v", v)
	return o
}

func ktIsJob(v interface{}) (*jsObject, bool) {
	o, ok := v.(*jsObject)
	if !ok {
		return nil, false
	}
	t, has := o.props["__job"]
	if !has || t != true {
		return nil, false
	}
	return o, true
}

// ktJobMethod is await / join / cancel and the state flags. Every one of them is
// immediate, because the block ran at the point it was written.
func (rt *jsrt) ktJobMethod(j *jsObject, name string) (interface{}, bool) {
	switch name {
	case "await", "getCompleted":
		return j.props["v"], true
	case "join", "cancel", "cancelAndJoin", "start":
		return jsUndef, true
	case "isActive", "isCancelled":
		return false, true
	case "isCompleted":
		return true, true
	}
	return nil, false
}

// ktPairMethod is componentN / toString on the {first, second[, third]} shape.
func (rt *jsrt) ktPairMethod2(o *jsObject, name string, args []interface{}) (interface{}, bool) {
	// Pair/Triple are data classes in Kotlin, so equals/hashCode are structural.
	switch name {
	case "hashCode":
		return float64(int32(ktElemHash(rt, o))), true
	case "equals":
		return rt.ktEqVals(o, argAt(args, 0)), true
	}
	return ktPairMethod(o, name)
}

func ktPairMethod(o *jsObject, name string) (interface{}, bool) {
	switch name {
	case "component1":
		return o.props["first"], true
	case "component2":
		return o.props["second"], true
	case "component3":
		if v, ok := o.props["third"]; ok {
			return v, true
		}
	case "toList":
		// Pair.toList() / Triple.toList(): [a, b] / [a, b, c].
		out := &jsArray{elems: []interface{}{o.props["first"], o.props["second"]}}
		if v, ok := o.props["third"]; ok {
			out.elems = append(out.elems, v)
		}
		return out, true
	}
	return nil, false
}


// ktCmpKeys is one comparator step over two selector keys: -1, 0 or 1, reversed for
// compareByDescending.
func ktCmpKeys(rt *jsrt, a, b interface{}, desc bool) int {
	c := 0
	if ktKeyLess(rt, a, b) {
		c = -1
	} else if ktKeyLess(rt, b, a) {
		c = 1
	}
	if desc {
		return -c
	}
	return c
}

// ktRaise throws one of the builtin throwables from a host builtin, in the shape
// js_try catches. The twin is kRaise in kotlin-interpreter.abnf.
func (rt *jsrt) ktRaise(cls, msg string) {
	o := newJSObject()
	o.set("__class", ktExcHierarchy()[cls])
	o.set("message", msg)
	panic(&jsThrown{value: o})
}

// ktRunCatching runs a lambda and boxes its completion as a kotlin.Result.
func (rt *jsrt) ktRunCatching(f interface{}) *jsObject {
	box := newJSObject()
	box.set("__result", true)
	box.set("value", jsNull)
	box.set("exc", jsNull)
	caught := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				th, ok := r.(*jsThrown)
				if !ok {
					panic(r)
				}
				box.set("exc", th.value)
				caught = true
			}
		}()
		box.set("value", rt.ktCall(f))
	}()
	_ = caught
	return box
}

// ktIsResult / ktResultMethod are the kotlin.Result member surface over the
// {__result, value, exc} box. isSuccess and isFailure are PROPERTIES in Kotlin, so
// js_ktfget reaches this too. The twin is kResultMethod in kotlin-interpreter.abnf.
func ktIsResult(v interface{}) (*jsObject, bool) {
	o, ok := v.(*jsObject)
	if !ok {
		return nil, false
	}
	if tag, has := o.props["__result"]; !has || tag != true {
		return nil, false
	}
	return o, true
}

func (rt *jsrt) ktResultMethod(r *jsObject, name string, args []interface{}) (interface{}, bool) {
	ok := isUndefOrNull(r.props["exc"])
	switch name {
	case "isSuccess":
		return ok, true
	case "isFailure":
		return !ok, true
	case "getOrNull":
		if ok {
			return r.props["value"], true
		}
		return jsNull, true
	case "exceptionOrNull":
		if ok {
			return jsNull, true
		}
		return r.props["exc"], true
	case "getOrDefault":
		if ok {
			return r.props["value"], true
		}
		return argAt(args, 0), true
	case "getOrElse":
		if ok {
			return r.props["value"], true
		}
		return rt.ktCall(argAt(args, 0), r.props["exc"]), true
	case "getOrThrow":
		if ok {
			return r.props["value"], true
		}
		panic(&jsThrown{value: r.props["exc"]})
	case "onSuccess":
		if ok {
			rt.ktCall(argAt(args, 0), r.props["value"])
		}
		return r, true
	case "onFailure":
		if !ok {
			rt.ktCall(argAt(args, 0), r.props["exc"])
		}
		return r, true
	case "map":
		if !ok {
			return r, true
		}
		out := newJSObject()
		out.set("__result", true)
		out.set("value", rt.ktCall(argAt(args, 0), r.props["value"]))
		out.set("exc", jsNull)
		return out, true
	case "fold":
		if ok {
			return rt.ktCall(argAt(args, 0), r.props["value"]), true
		}
		return rt.ktCall(argAt(args, 1), r.props["exc"]), true
	}
	return nil, false
}

// ktLazyValue is the lazy {} delegate's cached value.
func (rt *jsrt) ktLazyValue(o *jsObject) interface{} {
	if done, _ := o.props["done"].(bool); !done {
		o.set("v", rt.ktCall(o.props["f"]))
		o.set("done", true)
	}
	return o.props["v"]
}

// ktFixed renders a float with `prec` decimals, rounding HALF UP the way java's
// %f does. The twin is kFixed in kotlin-interpreter.abnf.
func ktFixed(f float64, prec int) string {
	neg := f < 0
	x := f
	if neg {
		x = -x
	}
	scale := 1.0
	for i := 0; i < prec; i++ {
		scale *= 10
	}
	scaled := math.Floor(x*scale + 0.5)
	whole := math.Floor(scaled / scale)
	frac := int64(scaled - whole*scale)
	fs := fmt.Sprintf("%d", frac)
	for len(fs) < prec {
		fs = "0" + fs
	}
	out := fmt.Sprintf("%d", int64(whole))
	if prec > 0 {
		out = out + "." + fs
	}
	if neg {
		out = "-" + out
	}
	return out
}

// ktClassChainHas answers whether the instance's descriptor - or anything on its
// __super chain - declares a callable member of this name. It is the guard on the
// own-field dispatch in js_ktsmcall: a declared method must still win over a field
// that happens to hold a function.
func ktClassChainHas(o *jsObject, name string) bool {
	cls := o.props["__class"]
	for cls != nil {
		clsObj, ok := cls.(*jsObject)
		if !ok {
			return false
		}
		if m, ok := clsObj.props[name]; ok && isCallable(m) {
			return true
		}
		cls = clsObj.props["__super"]
	}
	return false
}

// ----- Callable references and class literals -----
//
// Kotlin's `b::v`, `Box::v`, `b::twice`, `::topFun`, `::Box`, `b::class` and
// `Box::class` (the Kotlin documentation, "Callable references" and "Class
// references"). The twins are kRefMake / kRefApply / kClassOf / kSimpleName in
// languages/kotlin-interpreter.abnf and the two MUST stay in step - ./test.sh
// --cross diffs the two halves.
//
// A reference is a *hostFunc, not a record, and that is the whole design: it makes
// the value CALLABLE by every path the runtime already has (js_call, and the
// rt.call inside ktSeqMethod that `list.map(Box::v)` goes through), with no change
// to abnf/jsrt.go at all. The metadata hangs off a side table keyed by the
// hostFunc POINTER, which is safe because the value is built at run time and never
// crosses the tag stack - the identity trap in docs/abnf-dialect-gotchas.md is a
// BUILD-time one.
type ktRefInfo struct {
	recv  interface{} // the bound receiver; nil when unbound
	name  string
	bound bool
	ctor  bool        // ::Box - a constructor reference
	tname string      // the TYPE the base named, for an unbound reference
	ext   interface{} // the EXTENSION function of that name, resolved at compile time
}

// ktTypeName is what js_ktrefbase answers when the base of a `::` named a type
// rather than a value (`String::length`). The twin is the !hasVar arm of
// makePropRef in the interpreter half.
type ktTypeName struct{ name string }

var ktRefs map[*hostFunc]*ktRefInfo
var ktFuncNames map[interface{}]string

func ktRefOf(v interface{}) (*ktRefInfo, bool) {
	hf, ok := v.(*hostFunc)
	if !ok || ktRefs == nil {
		return nil, false
	}
	info, has := ktRefs[hf]
	return info, has
}

// ktMakeRef builds the callable. Calling it IS KFunction.invoke / KProperty.invoke:
// a name the receiver's class chain declares as a METHOD is called, anything else
// is read as a property - which is what lets one shape serve both kinds.
func (rt *jsrt) ktMakeRef(info *ktRefInfo) interface{} {
	hf := &hostFunc{name: "::" + info.name}
	hf.fn = func(r *jsrt, this uint64, args []interface{}) interface{} {
		return r.ktRefApply(info, args)
	}
	if ktRefs == nil {
		ktRefs = map[*hostFunc]*ktRefInfo{}
	}
	ktRefs[hf] = info
	return hf
}

func ktRefRecv(info *ktRefInfo, args []interface{}) interface{} {
	if info.bound {
		return info.recv
	}
	if len(args) > 0 {
		return args[0]
	}
	return jsNull
}

func ktRefRest(info *ktRefInfo, args []interface{}) []interface{} {
	if info.bound || len(args) == 0 {
		return args
	}
	return args[1:]
}

func (rt *jsrt) ktRefApply(info *ktRefInfo, args []interface{}) interface{} {
	if info.ctor {
		return rt.ktConstruct(info.recv, args)
	}
	recv := ktRefRecv(info, args)
	if o, isObj := recv.(*jsObject); isObj {
		if ktClassChainHas(o, info.name) {
			return ktMemberCall(recv, info.name, ktRefRest(info, args))
		}
		// A member the receiver really HAS wins over a same-named extension, which is
		// Kotlin's rule for the call too.
		if _, own := o.props[info.name]; own {
			return ktRefRead(recv, info.name)
		}
	}
	if info.ext != nil {
		return rt.call(info.ext, jsUndef, append([]interface{}{recv}, ktRefRest(info, args)...))
	}
	return ktRefRead(recv, info.name)
}

// ktRefRead / ktRefWrite are KProperty.get and KProperty.set: the property READ and
// WRITE, never the method. They are installed by the registrar so the reference code
// reaches exactly the field paths the emitted code uses (js_ktfget / js_set).
var ktRefRead func(recv interface{}, name string) interface{}
var ktRefWrite func(recv interface{}, name string, v interface{})

// ktConstruct runs a class descriptor's constructor - what `::Box.invoke(7)` and
// `list.map(::Box)` do. It is the runtime twin of the emitted `new` sequence.
func (rt *jsrt) ktConstruct(cls interface{}, args []interface{}) interface{} {
	clsObj, ok := cls.(*jsObject)
	if !ok {
		rt.fail("constructor reference on a non-class")
	}
	obj := newJSObject()
	obj.set("__class", clsObj)
	if ctor, has := clsObj.props["__ctor"]; has && isCallable(ctor) {
		rt.call(ctor, jsUndef, append([]interface{}{obj}, args...))
	}
	return obj
}

// ktSimpleName is the simple name of a value's class - KClass.simpleName. The twin
// is kSimpleName in kotlin-interpreter.abnf.
func ktSimpleName(v interface{}) interface{} {
	switch t := v.(type) {
	case jsUndefT, jsNullT:
		return jsNull
	case jsChar:
		return "Char"
	case jsJFlo:
		// The style byte says which of the two the box holds, so `1.5f::class`
		// is Float where it used to answer Double alongside every real double.
		if t.sty == floJavaF {
			return "Float"
		}
		return "Double"
	case jsGInt:
		return ktBoxTypeName(t)
	case float64:
		return "Int"
	case string:
		return "String"
	case bool:
		return "Boolean"
	case *jsArray:
		return "ArrayList"
	case *jsObject:
		if b, isCls := t.props["__isclass"].(bool); isCls && b {
			return t.props["__name"]
		}
		if cls, has := t.props["__class"].(*jsObject); has {
			return cls.props["__name"]
		}
	}
	return jsNull
}

// ktClassLit is `x::class` / `Type::class`. A KClass renders "class Box", and so
// does java.lang.Class - which is why .java answers the same handle here.
func ktClassLit(name interface{}) *jsObject {
	o := newJSObject()
	o.set("__kclass", true)
	o.set("kname", name)
	return o
}

func ktIsClassLit(v interface{}) (*jsObject, bool) {
	o, ok := v.(*jsObject)
	if !ok {
		return nil, false
	}
	b, isB := o.props["__kclass"].(bool)
	return o, isB && b
}

// ktRefEq is Kotlin's equality on the two reflection shapes: two BOUND references
// are equal when they name the same member of the same receiver (`b::v == b::v` is
// true), and two class literals when they name the same class. Answers handled=false
// for anything that is not one of the two.
func ktRefEq(l, r interface{}) (bool, bool) {
	li, lIsRef := ktRefOf(l)
	ri, rIsRef := ktRefOf(r)
	if lIsRef || rIsRef {
		if !lIsRef || !rIsRef {
			return false, true
		}
		return li.name == ri.name && li.bound == ri.bound && li.ctor == ri.ctor &&
			li.tname == ri.tname && li.recv == ri.recv, true
	}
	lc, lIsCls := ktIsClassLit(l)
	rc, rIsCls := ktIsClassLit(r)
	if lIsCls || rIsCls {
		if !lIsCls || !rIsCls {
			return false, true
		}
		return lc.props["kname"] == rc.props["kname"], true
	}
	return false, false
}

// ktTypeNames are the names Kotlin declares as TYPES rather than values, so that a
// `::` in front of one of them is an UNBOUND reference. The twin is kTypeNames in
// languages/kotlin-interpreter.abnf and the two lists MUST stay in step.
var ktTypeNames = map[string]bool{
	"String": true, "CharSequence": true, "Int": true, "Long": true, "Short": true,
	"Byte": true, "UInt": true, "ULong": true, "UShort": true, "UByte": true,
	"Double": true, "Float": true, "Boolean": true, "Char": true, "Any": true,
	"Unit": true, "Number": true, "Nothing": true, "List": true, "MutableList": true,
	"Set": true, "MutableSet": true, "Map": true, "MutableMap": true, "Array": true,
	"Collection": true, "Iterable": true, "Comparable": true, "IntRange": true,
	"CharRange": true, "LongRange": true, "Regex": true, "StringBuilder": true,
}
