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

func ktExcClass() *jsObject {
	if ktArithExcCls != nil {
		return ktArithExcCls
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
	thr := mk("Throwable", nil)
	exc := mk("Exception", thr)
	run := mk("RuntimeException", exc)
	ktArithExcCls = mk("ArithmeticException", run)
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
		if name == "Float" {
			mx, mn, bits = math.MaxFloat32, 1.4e-45, 32
		}
		o.set("MAX_VALUE", jsJFlo{f: mx})
		o.set("MIN_VALUE", jsJFlo{f: mn})
		o.set("NaN", jsJFlo{f: math.NaN()})
		o.set("POSITIVE_INFINITY", jsJFlo{f: math.Inf(1)})
		o.set("NEGATIVE_INFINITY", jsJFlo{f: math.Inf(-1)})
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
	case "toDouble", "toFloat":
		return jsJFlo{f: giFloat(rt, target)}, true
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
	case "isNaN":
		f, ok := target.(jsJFlo)
		return ok && math.IsNaN(f.f), true
	case "hashCode":
		if b, ok := target.(jsGInt); ok {
			return float64(int32(b.v ^ (b.v >> 32))), true
		}
		return float64(int32(giVal(rt, target))), true
	}
	return nil, false
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
				return w(jsJFlo{f: giFloat(rt, l) + giFloat(rt, r), sty: jvmStyleOf(l, r)})
			}
			if ktIsIntegral(l) && ktIsIntegral(r) {
				return w(rt.ktArith("+", l, r))
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
			return baseEq(a)
		}
		m["js_ktne"] = func(a []uint64) uint64 {
			l, r := u(a[0]), u(a[1])
			if giIsInt(l) || giIsInt(r) {
				return boolH(!rt.giEq(l, r))
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
		m["js_ktis"] = func(a []uint64) uint64 {
			v := u(a[0])
			switch v.(type) {
			case jsGInt, jsJFlo, float64:
			default:
				return baseIs(a)
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
				return boolH(t == "Number" || t == "Double" || t == "Float")
			}
			return boolH(t == "Number" || t == "Int")
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
			if v, handled := rt.ktNumMethod(u(a[0]), rt.toString(u(a[1])), arr.elems); handled {
				return rt.wrap(v)
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
			if next := m["js_rxktmcall"]; next != nil {
				return next(a)
			}
			return baseMcall(a)
		}
	})
	// js_ktsmcall reads its argument array in position 2, exactly like js_mcall and
	// js_rxktmcall: the handle must arrive as the array itself, not as a value.
	jsThroughArgs["js_ktsmcall"] = 1 << 2
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
	return nil, false
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
	// A Regex renders as its pattern and a MatchResult as the matched text, which is
	// what their toString answers (see abnf/jsrtregexkt.go); neither carries a
	// __class, so without this they fell through to "[object Object]".
	if ktRxIsRegex(o) || ktRxIsMatch(o) {
		if v, handled := rt.ktRxMethod(o, "toString", nil); handled {
			return rt.ktpRender(v, depth+1), true
		}
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
