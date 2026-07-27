package abnf

// jsrtint.go -- SIZED INTEGERS for the statically typed languages' compiler
// grammars (languages/go-to-llvm-ir.abnf first; java/kotlin/csharp are the same
// shape when they follow).
//
// The handle runtime models every number as one JS double. That is enough for a
// dynamically typed language and wrong for a statically typed one in two ways
// that a dynamic rule cannot repair, because in an untyped value model the
// operands carry no type:
//
//	var i8 int8 = 127; i8++      real go: -128     here: 128
//	var u8 uint8 = 255; u8++     real go: 0        here: 256
//	1 << 62                      real go: 4611686018427387904   here: 0
//	int64max                     real go: 9223372036854775807   here: 9.223372036854776e+18
//	int64max + 1                 real go: -9223372036854775808  here: 9.223372036854776e+18
//
// The first two need the DECLARED WIDTH of the variable to survive into the
// increment; the last three need 64 bits of precision, which a double has not
// got. So the type goes ON the value, exactly as commit 636e326 put floatness on
// the value with the jsJFlo box.
//
// jsGInt is that box. The invariant, in both halves of every language that opts
// in:
//
//	plain number   ==  a signed 64 bit integer whose value is exact in a double
//	                   (|v| <= 2^53), i.e. Go's int / int64 / rune in the range
//	                   every ordinary program stays inside
//	jsGInt         ==  any other integer: a sized type (int8..uint32), any
//	                   unsigned type, or a 64 bit value that has left the
//	                   exactly representable range
//	jsJFlo         ==  a float64 / float32                    (jsrtjvm.go)
//
// Keeping the common case a PLAIN NUMBER is deliberate and is what makes the
// change affordable: a program that never writes a sized type and never leaves
// 2^53 allocates no box at all and runs exactly the code it ran before. The box
// is rare, so the handful of places that consume a number raw (an array index, a
// slice bound, a map key) meet one only in the programs that asked for it.
//
// The interpreter halves implement the identical rule on top of the i64* pair
// arithmetic in languages/lib/interp-core.js; ./test.sh --cross diffs the two, so
// the two files must stay in step.
//
// Registration is additive: an init() appends to rxExtraExterns, so nothing here
// runs unless a program calls one of the js_gi* externs.

import (
	"math"
	"strconv"
	"strings"
)

// giExact is 2^53: the largest magnitude an integer keeps exactly in a float64.
const giExact = 9007199254740992

// jsGInt is one sized integer. `v` holds the value already truncated to `w`
// bits: sign extended for a signed width, so a int8 -1 is int64(-1); zero
// extended into the low `w` bits for an unsigned one, except that a uint64 keeps
// the raw bit pattern and is READ as uint64(v).
type jsGInt struct {
	v int64
	w uint8 // 8, 16, 32 or 64
	u bool  // unsigned
}

func giIsInt(v interface{}) bool { _, ok := v.(jsGInt); return ok }

// giTrunc wraps a value into `w` bits with `u`'s signedness, which is what every
// arithmetic operator does on overflow in Go, Java, Kotlin and C#.
func giTrunc(v int64, w uint8, u bool) int64 {
	switch w {
	case 8:
		if u {
			return int64(uint8(v))
		}
		return int64(int8(v))
	case 16:
		if u {
			return int64(uint16(v))
		}
		return int64(int16(v))
	case 32:
		if u {
			return int64(uint32(v))
		}
		return int64(int32(v))
	}
	return v
}

// giNorm is the ONE place the invariant above is applied: it answers a plain
// number when the value is a signed 64 bit one that a double holds exactly, and
// a box otherwise. Everything that produces an integer goes through it.
func giNorm(v int64, w uint8, u bool) interface{} {
	v = giTrunc(v, w, u)
	if w == 64 && !u && v <= giExact && v >= -giExact {
		return float64(v)
	}
	return jsGInt{v: v, w: w, u: u}
}

// giVal is the 64 bit value of any integral operand: a box's payload, a plain
// number truncated toward zero, a char's code point. It is the operand reader
// for every operator below.
func giVal(rt *jsrt, x interface{}) int64 {
	switch t := x.(type) {
	case jsGInt:
		return t.v
	case float64:
		return giFromFloat(t)
	case jsChar:
		return int64(t.code)
	case jsJFlo:
		return giFromFloat(t.f)
	}
	return giFromFloat(rt.toNumber(x))
}

// giFromFloat truncates toward zero and wraps modulo 2^64, which is what a
// conversion from a floating point value to a 64 bit integer type does.
func giFromFloat(f float64) int64 {
	if math.IsNaN(f) {
		return 0
	}
	if f >= 9223372036854775808 || f < -9223372036854775808 {
		// Out of int64 range: reduce modulo 2^64 the way the hardware wrap does.
		m := math.Mod(math.Trunc(f), 18446744073709551616)
		if m >= 9223372036854775808 {
			m -= 18446744073709551616
		} else if m < -9223372036854775808 {
			m += 18446744073709551616
		}
		return int64(m)
	}
	return int64(math.Trunc(f))
}

// giWidthOf is the result type of a binary operation. Go (and Java, Kotlin, C#)
// require both operands of an arithmetic operator to have the same type, so at
// most one of them is a box and it decides; two boxes agree in practice, and
// where a program manages to disagree the LEFT one wins, which is the same rule
// jvmStyleOf uses for the float box.
func giWidthOf(l, r interface{}) (uint8, bool) {
	if b, ok := l.(jsGInt); ok {
		return b.w, b.u
	}
	if b, ok := r.(jsGInt); ok {
		return b.w, b.u
	}
	return 64, false
}

// giArith is one binary arithmetic or bitwise operator, evaluated at the result
// type's full width and wrapped to it.
func (rt *jsrt) giArith(op string, l, r interface{}) interface{} {
	w, u := giWidthOf(l, r)
	a, b := giVal(rt, l), giVal(rt, r)
	var x int64
	switch op {
	case "+":
		x = a + b
	case "-":
		x = a - b
	case "*":
		x = a * b
	case "/":
		if b == 0 {
			rt.fail("integer divide by zero")
		}
		if u {
			x = int64(giU(a, w) / giU(b, w))
		} else if a == math.MinInt64 && b == -1 {
			x = a // the hardware wrap; Go panics, but only for the 64 bit case
		} else {
			x = a / b
		}
	case "%":
		if b == 0 {
			rt.fail("integer divide by zero")
		}
		if u {
			x = int64(giU(a, w) % giU(b, w))
		} else if a == math.MinInt64 && b == -1 {
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
	case "&^":
		x = a &^ b
	case "<<":
		// The shift COUNT is a separate operand and is never negative in a
		// well-typed program; a count at or above the width shifts everything
		// out, which is Go's rule (and unlike C, not undefined).
		s := giVal(rt, r)
		if s < 0 || s >= int64(w) {
			x = 0
		} else {
			x = a << uint(s)
		}
	case ">>":
		s := giVal(rt, r)
		if s < 0 {
			x = 0
		} else if u {
			if s >= int64(w) {
				x = 0
			} else {
				x = int64(giU(a, w) >> uint(s))
			}
		} else {
			if s >= int64(w) {
				s = int64(w) - 1
			}
			x = a >> uint(s)
		}
	default:
		rt.fail("js_giarith: unknown operator %q", op)
	}
	return giNorm(x, w, u)
}

// giU reads a value as unsigned at width w.
func giU(v int64, w uint8) uint64 {
	switch w {
	case 8:
		return uint64(uint8(v))
	case 16:
		return uint64(uint16(v))
	case 32:
		return uint64(uint32(v))
	}
	return uint64(v)
}

// giCmp is <, <=, > and >= for two integral operands: unsigned when the result
// type is unsigned, so uint64max > 0 rather than -1 < 0.
func (rt *jsrt) giCmp(l, r interface{}) int {
	w, u := giWidthOf(l, r)
	a, b := giVal(rt, l), giVal(rt, r)
	if u {
		ua, ub := giU(a, w), giU(b, w)
		if ua < ub {
			return -1
		}
		if ua > ub {
			return 1
		}
		return 0
	}
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

// giEq is == between an integer box and any other value. A box never equals a
// non-numeric value, and two integers compare by VALUE whatever their widths -
// which is what a well-typed program can observe, since it could not have
// written the comparison at all if the types disagreed.
func (rt *jsrt) giEq(l, r interface{}) bool {
	if giIsNumeric(l) && giIsNumeric(r) {
		if f, ok := l.(jsJFlo); ok {
			return f.f == giFloat(rt, r)
		}
		if f, ok := r.(jsJFlo); ok {
			return f.f == giFloat(rt, l)
		}
		return giVal(rt, l) == giVal(rt, r)
	}
	return rt.strictEq(l, r)
}

// giIsIntegral is the narrower test the ordered comparisons need: a float
// operand must NOT be truncated into an integer comparison.
func giIsIntegral(v interface{}) bool {
	switch v.(type) {
	case jsGInt, float64, jsChar:
		return true
	}
	return false
}

func giIsNumeric(v interface{}) bool {
	switch v.(type) {
	case jsGInt, float64, jsJFlo, jsChar:
		return true
	}
	return false
}

// giFloat is the floating point reading of an integral operand, which an
// unsigned 64 bit box needs its own path for (int64(-1) as a uint64 is 1.8e19).
func giFloat(rt *jsrt, v interface{}) float64 {
	if b, ok := v.(jsGInt); ok {
		if b.u && b.w == 64 {
			return float64(uint64(b.v))
		}
		return float64(b.v)
	}
	if f, ok := v.(jsJFlo); ok {
		return f.f
	}
	return rt.toNumber(v)
}

// giStr is the decimal text of a box - the one place the unsigned reading
// matters for output.
func giStr(b jsGInt) string {
	if b.u {
		return strconv.FormatUint(giU(b.v, b.w), 10)
	}
	return strconv.FormatInt(b.v, 10)
}

// giConv is an explicit conversion to a sized integer type: int8(x), uint32(x),
// int64(x), uint(x). A floating point source truncates toward zero first.
func (rt *jsrt) giConv(x interface{}, w uint8, u bool) interface{} {
	return giNorm(giVal(rt, x), w, u)
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

		// One binary operator: (op name, left, right). A single extern keeps the
		// emitted IR one call per operator, the way js_jf* is one per operator -
		// the operator itself is a compile time constant string, so the switch
		// above costs one comparison chain and no dispatch table.
		m["js_giarith"] = func(a []uint64) uint64 {
			op, l, r := rt.toString(u(a[0])), u(a[1]), u(a[2])
			// A float64 operand keeps the FLOAT path (the jsJFlo box of
			// jsrtjvm.go): the emitters route every arithmetic operator here, so
			// this is where the two boxes meet. Only '- * / %' exist for floats;
			// a bitwise operator on one is not a program the front end accepts.
			if jvmIsFlo(l) || jvmIsFlo(r) {
				switch op {
				case "-", "*", "/", "%":
					return w(rt.jvmArith(op[0], l, r))
				}
			}
			return w(rt.giArith(op, l, r))
		}
		// Ordered comparison: answers -1, 0 or 1 so one extern serves < <= > >=.
		m["js_gicmp"] = func(a []uint64) uint64 {
			return rt.wrapNum(float64(rt.giCmp(u(a[0]), u(a[1]))))
		}
		m["js_gieq"] = func(a []uint64) uint64 { return boolH(rt.giEq(u(a[0]), u(a[1]))) }
		// A conversion / a declared width: (value, bits, unsigned).
		m["js_giconv"] = func(a []uint64) uint64 {
			return w(rt.giConv(u(a[0]), uint8(jsToInt(rt.toNumber(u(a[1])))), rt.truthy(u(a[2]))))
		}
		// Unary minus and ^x (bitwise complement), both at the operand's width.
		m["js_gineg"] = func(a []uint64) uint64 {
			v := u(a[0])
			// -x keeps the operand's type: a float64 negates as a float64, so
			// -0.0 and -1.0/0.0 stay real; an integer negates at its own width,
			// so -int8(-128) wraps back to -128.
			if f, ok := v.(jsJFlo); ok {
				return w(jsJFlo{f: -f.f, sty: f.sty})
			}
			wd, un := giWidthOf(v, v)
			return w(giNorm(-giVal(rt, v), wd, un))
		}
		m["js_ginot"] = func(a []uint64) uint64 {
			v := u(a[0])
			wd, un := giWidthOf(v, v)
			return w(giNorm(^giVal(rt, v), wd, un))
		}
		// The plain-number reading an index, a slice bound or a repeat count
		// needs: a box is not a float64 and every such consumer wants one.
		m["js_ginum"] = func(a []uint64) uint64 { return rt.wrapNum(giFloat(rt, u(a[0]))) }
		// The decimal text, for the print and string-conversion paths.
		m["js_gistr"] = func(a []uint64) uint64 {
			if b, ok := u(a[0]).(jsGInt); ok {
				return rt.wrapStr(giStr(b))
			}
			return rt.wrapStr(rt.toString(u(a[0])))
		}
		m["js_giis"] = func(a []uint64) uint64 { return boolH(giIsInt(u(a[0]))) }
		// The UNBOXER: a sized integer becomes its plain number, everything else
		// passes through untouched. It is what an existing consumer that only
		// knows plain numbers needs in front of it - js_goconv's string(rune),
		// an index, a slice bound - without that consumer having to learn the box.
		m["js_gival"] = func(a []uint64) uint64 {
			if b, ok := u(a[0]).(jsGInt); ok {
				if b.u && b.w == 64 {
					return rt.wrapNum(float64(uint64(b.v)))
				}
				return rt.wrapNum(float64(b.v))
			}
			return a[0]
		}
		// The four ordered comparisons. A box has to be compared through giCmp
		// rather than through the shared jsCompare, because a uint64 above 2^63
		// reads as a NEGATIVE int64 there, so uint64max < 0 would hold. Anything
		// that is not two integers (a string pair, a float operand) keeps the
		// runtime's own comparison, which already reads a box through toNumber.
		rel := func(name string, want func(int) bool) {
			m[name] = func(a []uint64) uint64 {
				l, r := u(a[0]), u(a[1])
				if giIsInt(l) || giIsInt(r) {
					if giIsIntegral(l) && giIsIntegral(r) {
						return boolH(want(rt.giCmp(l, r)))
					}
				}
				return boolH(want(rt.jsCompare(l, r)))
			}
		}
		rel("js_gilt", func(c int) bool { return c < 0 })
		rel("js_gile", func(c int) bool { return c <= 0 })
		rel("js_gigt", func(c int) bool { return c > 0 })
		rel("js_gige", func(c int) bool { return c >= 0 })
		// '+' is the one arithmetic operator that is not only arithmetic: in Go
		// it also concatenates strings, and a float operand keeps its box. Two
		// integers add at 64 bits and wrap to the result type's width.
		m["js_giadd"] = func(a []uint64) uint64 {
			l, r := u(a[0]), u(a[1])
			if ls, ok := l.(string); ok {
				if rs, ok2 := r.(string); ok2 {
					return rt.wrapStr(strConcat(ls, rs))
				}
			}
			if jvmIsFlo(l) || jvmIsFlo(r) {
				return w(rt.jvmArith('+', l, r))
			}
			if !giIsNumeric(l) || !giIsNumeric(r) {
				return w(rt.jsAdd(l, r))
			}
			return w(rt.giArith("+", l, r))
		}
		// A literal that a double cannot hold exactly: the emitter passes its
		// DIGITS, because emitNum would already have rounded 9223372036854775807
		// to 9223372036854776000 on the way into the module. (text, radix, bits,
		// unsigned); an out-of-range value wraps, which is what a Go constant
		// conversion does.
		m["js_gilit"] = func(a []uint64) uint64 {
			s := rt.toString(u(a[0]))
			radix := uint64(jsToInt(rt.toNumber(u(a[1]))))
			bits := uint8(jsToInt(rt.toNumber(u(a[2]))))
			uns := rt.truthy(u(a[3]))
			neg := false
			if strings.HasPrefix(s, "-") {
				neg, s = true, s[1:]
			}
			// Accumulated by hand rather than with ParseUint: the full 64 bit
			// range has to round-trip (ParseInt rejects 18446744073709551615)
			// and an overflowing literal has to WRAP rather than saturate.
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
			v := int64(acc)
			if neg {
				v = -v
			}
			return w(giNorm(v, bits, uns))
		}
	})
}
