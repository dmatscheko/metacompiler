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
			recv, mname := u(a[0]), rt.toString(u(a[1]))
			// The SCOPE FUNCTIONS first: let/run/apply/also/takeIf/takeUnless are
			// extensions on Any, so no receiver branch below owns them. A class
			// member of the same name still wins (ktScopeMethod declines then).
			if v, handled := rt.ktScopeMethod(recv, mname, arr.elems); handled {
				return rt.wrap(v)
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
			if mo, isObj := recv.(*jsObject); isObj {
				if ktIsSb(mo) {
					if v, handled := rt.ktSbMethod(mo, mname, arr.elems); handled {
						return rt.wrap(v)
					}
				}
				if _, _, isDict := dictParts(mo); isDict {
					if v, handled := rt.ktMapMethod(mo, mname, arr.elems); handled {
						return rt.wrap(v)
					}
				} else if po, isPair := ktIsPair(mo); isPair {
					if v, handled := ktPairMethod(po, mname); handled {
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

		// js_ktglobal(name) is the VALUE of one global builder; buildMain declares
		// one per name. See the block above ktRecvStack.
		m["js_ktglobal"] = func(a []uint64) uint64 { return w(ktGlobalFn(rt, rt.toString(u(a[0])))) }

		// js_ktget is js_kget plus the implicit receiver of with / run / apply /
		// buildString / buildList / buildMap: a name that is neither a local nor a
		// member of the enclosing `this` is resolved against the receiver on
		// ktRecvStack, so `with(sb) { append("a") }` and `with(list) { size }` mean
		// what they mean in Kotlin. The two earlier steps are js_kget's, byte for
		// byte, so a name that already resolved resolves the same way.
		m["js_ktget"] = func(a []uint64) uint64 {
			sc := rt.scopeOf(a[0])
			name := rt.toString(u(a[1]))
			for s := sc; s != nil; s = s.parent {
				if v, ok := s.get(name); ok {
					if rt.traced {
						rt.trVar("read", name, v)
					}
					return w(v)
				}
			}
			for s := sc; s != nil; s = s.parent {
				if t, ok := s.get("this"); ok {
					if obj, isObj := t.(*jsObject); isObj {
						if v, ok := obj.props[name]; ok {
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
					break
				}
			}
			for i := len(ktRecvStack) - 1; i >= 0; i-- {
				if v, ok := rt.ktRecvMember(ktRecvStack[i], name); ok {
					if rt.traced {
						rt.trVar("read", name, v)
					}
					return w(v)
				}
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
			if mo, isObj := o.(*jsObject); isObj {
				if ktIsSb(mo) {
					if v, handled := rt.ktSbMethod(mo, name, nil); handled {
						return w(v)
					}
				}
				if keys, _, isDict := dictParts(mo); isDict && !ktRecvProp(name) &&
					rt.ktMapFind(keys, u(a[1])) < 0 {
					return w(jsNull)
				}
			}
			if next := m["js_rxktget"]; next != nil {
				return next(a)
			}
			return m["js_get"](a)
		}

		// js_ktindex is the same rule for the INDEXED read: `m[9]` on a map that has no
		// key 9 is null in Kotlin (Map.get answers null), where the shared js_kindex
		// answers undefined - which printed "kotlin.Unit" and made `m[9] == null` false
		// in the compiler while the interpreter said true. Everything else falls through
		// to js_rxktindex (js_kindex plus the MatchResult group readers).
		m["js_ktindex"] = func(a []uint64) uint64 {
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
	}
	// Everything a String shares with a List of Char - map/filter/forEach/any/all/
	// count/fold/joinToString/... - goes through the collection surface.
	if v, handled := rt.ktSeqMethod(&jsArray{elems: chars()}, name, args); handled {
		return v, true
	}
	return nil, false
}

// ktParseNum is toInt() / toLong() / toDouble() on a String. Kotlin throws a
// NumberFormatException for text that is not a number, and java.lang's message is
// the one both halves report.
func ktParseNum(rt *jsrt, s, name string) (interface{}, bool) {
	t := strings.TrimSpace(s)
	if name == "toDouble" {
		var f float64
		if _, err := fmt.Sscanf(t, "%g", &f); err != nil || t == "" {
			rt.fail("For input string: \"%s\"", s)
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
		rt.fail("For input string: \"%s\"", s)
	}
	var acc int64
	for _, r := range body {
		if r < '0' || r > '9' {
			if name == "toIntOrNull" {
				return jsNull, true
			}
			rt.fail("For input string: \"%s\"", s)
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
	if fa, ok := a.(jsJFlo); ok {
		if fb, ok2 := b.(jsJFlo); ok2 {
			return fa.f == fb.f
		}
	}
	if oa, ok := a.(*jsObject); ok {
		if _, ok2 := b.(*jsObject); ok2 {
			if mth, found := ktpFindMember(oa, "equals"); found {
				return rt.truthy(rt.call(mth, jsUndef, []interface{}{oa, b}))
			}
		}
	}
	return rt.strictEq(a, b)
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
	case "chunked":
		k := jsToInt(rt.toNumber(argAt(args, 0)))
		if k <= 0 {
			rt.fail("chunked() size must be positive")
		}
		out := []interface{}{}
		for i := 0; i < len(es); i += k {
			j := i + k
			if j > len(es) {
				j = len(es)
			}
			out = append(out, arr(append([]interface{}{}, es[i:j]...)))
		}
		return arr(out), true
	case "windowed":
		k := jsToInt(rt.toNumber(argAt(args, 0)))
		step := 1
		if len(args) > 1 {
			step = jsToInt(rt.toNumber(args[1]))
		}
		if k <= 0 || step <= 0 {
			rt.fail("windowed() size and step must be positive")
		}
		out := []interface{}{}
		for i := 0; i+k <= len(es); i += step {
			out = append(out, arr(append([]interface{}{}, es[i:i+k]...)))
		}
		return arr(out), true
	case "distinct", "toSet", "toMutableSet":
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
	case "reversed", "asReversed":
		out := make([]interface{}, len(es))
		for i, e := range es {
			out[len(es)-1-i] = e
		}
		return arr(out), true
	case "sorted", "sortedDescending":
		return ktSortBy(rt, es, nil, name == "sortedDescending"), true
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
	case "toList", "toMutableList", "asSequence", "asIterable", "toTypedArray":
		return arr(append([]interface{}{}, es...)), true
	case "withIndex":
		out := make([]interface{}, len(es))
		for i, e := range es {
			p := newJSObject()
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
		sep := ", "
		if len(args) > 0 {
			sep = rt.ktStrArg(args[0])
		}
		pre, post := "", ""
		if len(args) > 1 {
			pre = rt.ktStrArg(args[1])
		}
		if len(args) > 2 {
			post = rt.ktStrArg(args[2])
		}
		parts := make([]string, len(es))
		for i, e := range es {
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
	}
	if giIsNumeric(v) {
		x := giVal(rt, v)
		if b, ok := v.(jsGInt); ok && b.w == 64 {
			return x ^ (x >> 32)
		}
		return int64(int32(x))
	}
	return 1
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
	if s, ok := v.(string); ok {
		return rt.ktChars(s)
	}
	if keys, vals, ok := dictParts(v); ok {
		out := make([]interface{}, len(keys.elems))
		for i := range keys.elems {
			e := newJSObject()
			e.set("key", keys.elems[i])
			e.set("value", vals.elems[i])
			e.set("first", keys.elems[i])
			e.set("second", vals.elems[i])
			out[i] = e
		}
		return out
	}
	rt.fail("not a sequence: %s", rt.ktpRender(v, 0))
	return nil
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
			return &jsArray{elems: append([]interface{}{}, keys.elems...)}, true
		case "values":
			return &jsArray{elems: append([]interface{}{}, vals.elems...)}, true
		}
		return &jsArray{elems: ktElems(rt, m)}, true
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

// ktRecvProps are the names Kotlin declares as PROPERTIES on a builtin receiver;
// everything else resolves to a bound method, so a lookup never turns a property
// into a callable. The twin is kRecvProps in kotlin-interpreter.abnf.
func ktRecvProp(name string) bool {
	switch name {
	case "size", "length", "keys", "values", "entries", "indices", "lastIndex":
		return true
	}
	return false
}

// ktIsBuiltinRecv reports a receiver whose members come from the runtime rather
// than from a class descriptor: a list, a map, a StringBuilder box or a String.
func ktIsBuiltinRecv(v interface{}) bool {
	switch t := v.(type) {
	case *jsArray, string:
		return true
	case *jsObject:
		if _, isCls := t.props["__class"]; isCls {
			return false
		}
		if _, isClsDesc := t.props["__isclass"]; isClsDesc {
			return false
		}
		if _, _, isDict := dictParts(t); isDict {
			return true
		}
		return ktIsSb(t)
	}
	return false
}

// ktRecvMember answers an unqualified name against an implicit receiver.
func (rt *jsrt) ktRecvMember(recv interface{}, name string) (interface{}, bool) {
	if !ktIsBuiltinRecv(recv) || ktMemberCall == nil {
		return nil, false
	}
	if ktRecvProp(name) {
		return ktMemberCall(recv, name, nil), true
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
	case "Double", "Float":
		return jsJFlo{f: giFloat(rt, v)}
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

// ktWithRecv runs f with `recv` on the implicit-receiver stack.
func (rt *jsrt) ktWithRecv(recv interface{}, f interface{}, args ...interface{}) interface{} {
	if !isCallable(f) {
		return jsUndef
	}
	ktRecvStack = append(ktRecvStack, recv)
	defer func() { ktRecvStack = ktRecvStack[:len(ktRecvStack)-1] }()
	return rt.call(f, jsUndef, args)
}

// ktGlobalFn is the VALUE of one global builder name.
func ktGlobalFn(rt *jsrt, name string) interface{} {
	switch name {
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
			for guard := 0; guard < 100000 && !isUndefOrNull(v); guard++ {
				out.elems = append(out.elems, v)
				if !isCallable(next) {
					break
				}
				v = rt.ktCall(next, v)
			}
			return out
		})
	}
	return jsUndef
}

// ktPairMethod is componentN / toString on the {first, second[, third]} shape.
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
	}
	return nil, false
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
