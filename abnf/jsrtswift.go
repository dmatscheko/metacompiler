package abnf

// jsrtswift.go -- Swift's VALUE RENDERING and Swift's `==` for the compiler
// grammar (languages/swift-to-llvm-ir.abnf).
//
// swift-to-llvm-ir.abnf used to bind Swift's `print` straight to the shared
// `println` host function, which hands the raw runtime value to Go's `%v`. That
// printed `<nil>` for nil, `[1 2 3]` for an array, `map[__dict:true keys:[x]
// vals:[1]]` for a dictionary and - worst - a map containing a raw Go POINTER
// address for a struct or class instance, so the compiler's stdout was not even
// reproducible between two runs of the same program. Real Swift prints
//
//	nil / [1, 2, 3] / ["x": 1] / P(x: 1, y: 2) / a(1, "x") / (x: 1, y: "s")
//
// (verified against swift 6.1.2 - see the probe output in the commit that added
// this file). The interpreter half was wrong in its own way: its `swstr` fell
// through to JavaScript's ToString, so an array printed "1,2,3" and everything
// else "[object Object]". Both halves now render through the same rules; this
// file is the Go twin of `swDesc` / `swDescIn` in swift-interpreter.abnf and the
// two MUST stay in step, because ./test.sh --cross diffs them.
//
// Registration is additive: an init() appends to rxExtraExterns, so nothing here
// runs unless a program calls one of the js_sw* externs, and only the Swift
// compiler grammar emits them. No existing extern is renamed, removed or
// rebound.
//
// Known simplifications, both DELIBERATE and both shared by the two halves:
//
//   - A class instance prints as its bare type name. Real Swift prints the
//     module-qualified name (`main.C` / `sp2.C`), and this runtime has no module
//     concept at all; the bare name is the reproducible part of that answer.
//   - There is no Optional BOX in this value model - `Int?` is just the value or
//     nil - so a non-nil optional prints as the wrapped value where real Swift
//     prints `Optional(3)`. Recovering that needs the declared type at the print
//     site, which is a type-checker job this front end does not do. `nil` itself
//     prints as `nil`, which is right.
//   - A Dictionary prints in INSERTION order. Swift's order is unspecified (it is
//     the hash order and varies per run), so any fixed order is as correct as any
//     other and insertion order is the one both halves can reproduce.

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

func init() {
	rxExtraExterns = append(rxExtraExterns, func(rt *jsrt, m map[string]func(args []uint64) uint64) {
		u := rt.unwrap
		boolH := func(b bool) uint64 {
			if b {
				return jsHTrue
			}
			return jsHFalse
		}

		// String(describing:) / "\(v)" / the thing print writes.
		m["js_swdesc"] = func(a []uint64) uint64 { return rt.wrapStr(rt.swDesc(u(a[0]))) }

		// Swift's print(_:separator:terminator:). a[0] is the argument ARRAY,
		// a[1] the separator and a[2] the terminator; undefined means the
		// default (" " and "\n"). The labelled arguments used to be passed
		// positionally to the shared println, so `print(1, 2, separator: "-")`
		// printed `1 2 -`.
		m["js_swprint"] = func(a []uint64) uint64 {
			args, ok := u(a[0]).(*jsArray)
			if !ok {
				rt.fail("js_swprint needs an argument array")
			}
			sep := " "
			if s := u(a[1]); !isNullish(s) {
				sep = rt.swDesc(s)
			}
			term := "\n"
			if t := u(a[2]); !isNullish(t) {
				term = rt.swDesc(t)
			}
			out := ""
			for i, e := range args.elems {
				if i > 0 {
					out += sep
				}
				out += rt.swDesc(e)
			}
			fmt.Fprint(outWriter, wtf8Clean(out+term))
			return 0
		}

		// Swift's ==. `js_seq` compares an Array, a Dictionary and a tuple by
		// IDENTITY, so `[1, 2] == [1, 2]` was always false in the compiler; in
		// Swift all three are value types and compare element by element.
		m["js_sweq"] = func(a []uint64) uint64 { return boolH(rt.swEqual(u(a[0]), u(a[1]))) }

		// ----- Swift's arithmetic (see the "Integers" note below) -----
		w := rt.wrap
		// One binary arithmetic or bitwise operator: (op, left, right). Two
		// INTEGRAL operands go to the sized-integer layer of jsrtint.go, which
		// evaluates at the result type's width and wraps to it; an operand that
		// is a FLOAT BOX pulls the whole operation into floating point and the
		// result is a box again (see the "Doubles" note below).
		// swift-interpreter.abnf implements the identical rule with szArith and
		// its {__flo} box, and ./test.sh --cross diffs the two.
		m["js_swarith"] = func(a []uint64) uint64 {
			op, l, r := rt.toString(u(a[0])), u(a[1]), u(a[2])
			if swIsWhole(l) && swIsWhole(r) {
				return w(rt.giArith(op, l, r))
			}
			if op == "+" {
				_, ls := l.(string)
				_, rs := r.(string)
				if ls || rs {
					return rt.wrapStr(strConcat(rt.swDesc(l), rt.swDesc(r)))
				}
				// Array + Array is CONCATENATION in Swift, where the shared
				// jsAdd coerced both to text and answered "1,23".
				if la, ok := l.(*jsArray); ok {
					if ra, ok2 := r.(*jsArray); ok2 {
						out := make([]interface{}, 0, len(la.elems)+len(ra.elems))
						out = append(out, la.elems...)
						out = append(out, ra.elems...)
						return w(&jsArray{elems: out})
					}
				}
			}
			x, y := swNum(rt, l), swNum(rt, r)
			switch op {
			case "+":
				return w(swMkFlo(x + y))
			case "-":
				return w(swMkFlo(x - y))
			case "*":
				return w(swMkFlo(x * y))
			case "/":
				return w(swMkFlo(x / y))
			case "%":
				return w(swMkFlo(math.Mod(x, y)))
			}
			return w(rt.giArith(op, l, r))
		}
		// ----- Swift's Double: the boxed float (see the "Doubles" note below) -----
		// A float literal, Double(x) and Float(x) all arrive here. A String
		// source is FAILABLE, like Int("x").
		m["js_swflo"] = func(a []uint64) uint64 {
			v := u(a[0])
			if s, ok := v.(string); ok {
				if !swFloText(s) {
					return jsHNull
				}
				f, err := strconv.ParseFloat(s, 64)
				if err != nil {
					return jsHNull
				}
				return w(swMkFlo(f))
			}
			if isNullish(v) {
				rt.fail("Double conversion of nil")
			}
			if b, ok := v.(bool); ok {
				if b {
					return w(swMkFlo(1))
				}
				return w(swMkFlo(0))
			}
			return w(swMkFlo(swNum(rt, v)))
		}
		// String(x) / String(describing: x): the text print would have written.
		m["js_swstr"] = func(a []uint64) uint64 { return rt.wrapStr(rt.swDesc(u(a[0]))) }
		// abs/max/min, Double-aware: a boxed operand keeps its box (abs(-1.5) is
		// 1.5, not the 1 an integer truncation gives) and compares by VALUE.
		m["js_swabs"] = func(a []uint64) uint64 {
			v := u(a[0])
			if f, ok := v.(jsJFlo); ok {
				return w(swMkFlo(math.Abs(f.f)))
			}
			if giIsInt(v) {
				n := giVal(rt, v)
				if n < 0 {
					n = -n
				}
				wd, un := giWidthOf(v, v)
				return w(giNorm(n, wd, un))
			}
			return rt.wrapNum(math.Abs(math.Trunc(rt.toNumber(v))))
		}
		swPick := func(want func(int) bool) func(a []uint64) uint64 {
			return func(a []uint64) uint64 {
				l, r := u(a[0]), u(a[1])
				if swIsFlo(l) || swIsFlo(r) {
					c := 0
					if swNum(rt, l) < swNum(rt, r) {
						c = -1
					} else if swNum(rt, l) > swNum(rt, r) {
						c = 1
					}
					if want(c) {
						return a[0]
					}
					return a[1]
				}
				if giIsIntegral(l) && giIsIntegral(r) {
					if want(rt.giCmp(l, r)) {
						return a[0]
					}
					return a[1]
				}
				if want(rt.jsCompare(l, r)) {
					return a[0]
				}
				return a[1]
			}
		}
		m["js_swmax"] = swPick(func(c int) bool { return c > 0 })
		m["js_swmin"] = swPick(func(c int) bool { return c < 0 })
		// The Double methods this subset carries, in front of the runtime's own
		// js_mcall (a float box is not a shape memberCall knows). Verified
		// against swift 6.1.2: 2.0.rounded() is 2.0, 2.5.squareRoot() is
		// 1.5811388300841898 and 7.0.truncatingRemainder(dividingBy: 2.0) is 1.0.
		baseMcall := m["js_mcall"]
		m["js_swmcall"] = func(a []uint64) uint64 {
			f, isFlo := u(a[0]).(jsJFlo)
			if !isFlo {
				return baseMcall(a)
			}
			name := rt.toString(u(a[1]))
			args, _ := u(a[2]).(*jsArray)
			arg0 := func() float64 {
				if args == nil || len(args.elems) == 0 {
					return 0
				}
				return swNum(rt, args.elems[0])
			}
			switch name {
			case "rounded":
				return w(swMkFlo(swRoundHalfAway(f.f)))
			case "squareRoot":
				return w(swMkFlo(math.Sqrt(f.f)))
			case "truncatingRemainder":
				return w(swMkFlo(math.Mod(f.f, arg0())))
			case "isMultiple":
				return boolH(math.Mod(f.f, arg0()) == 0)
			case "description":
				return rt.wrapStr(swFloStr(f.f))
			}
			rt.fail("unknown Double method: %s", name)
			return 0
		}
		// Swift's SMART SHIFT: an over-shift answers 0 (or the sign, for a
		// signed >>) and a NEGATIVE count shifts the other way - `256 >> -2` is
		// 1024, verified against swift 6. giArith already gets the over-shift
		// right; only the negative count is Swift specific.
		m["js_swshift"] = func(a []uint64) uint64 {
			op, l, r := rt.toString(u(a[0])), u(a[1]), u(a[2])
			n := giVal(rt, r)
			if n < 0 {
				if op == "<<" {
					op = ">>"
				} else {
					op = "<<"
				}
				n = -n
			}
			return w(rt.giArith(op, l, giNorm(n, 64, false)))
		}
		// -x and ~x, both at the operand's own width. A fractional Double keeps
		// the floating point negation.
		m["js_swneg"] = func(a []uint64) uint64 {
			v := u(a[0])
			if !swIsWhole(v) {
				// A float negates as a float, so -0.0 keeps its sign and
				// -1.0/0.0 is -inf.
				return w(swMkFlo(-swNum(rt, v)))
			}
			wd, un := giWidthOf(v, v)
			return w(giNorm(-giVal(rt, v), wd, un))
		}
		m["js_swnot"] = func(a []uint64) uint64 {
			v := u(a[0])
			wd, un := giWidthOf(v, v)
			return w(giNorm(^giVal(rt, v), wd, un))
		}
		// Int(x) / Int8(x) / UInt64(x): (value, bits, unsigned). A String source
		// is FAILABLE - Swift answers nil for anything that is not a complete,
		// in-range integer - and is the one conversion that does not wrap.
		m["js_swintconv"] = func(a []uint64) uint64 {
			v := u(a[0])
			bits := uint8(jsToInt(rt.toNumber(u(a[1]))))
			uns := rt.truthy(u(a[2]))
			if s, ok := v.(string); ok {
				r, ok2 := swIntFromStr(s, bits, uns)
				if !ok2 {
					return jsHNull
				}
				return w(r)
			}
			if isNullish(v) {
				rt.fail("integer conversion of nil")
			}
			return w(rt.giConv(v, bits, uns))
		}
		// The four ordered comparisons. A box has to go through giCmp rather than
		// the shared jsCompare, because a UInt64 above 2^63 reads as a NEGATIVE
		// int64 there, so UInt64.max < 0 would hold.
		rel := func(name string, want func(int) bool) {
			m[name] = func(a []uint64) uint64 {
				l, r := u(a[0]), u(a[1])
				// A float box is an object to the shared comparison, so it has
				// to be unboxed first: `1.5 < 2.0` between two boxes is not a
				// numeric comparison at all.
				if swIsFlo(l) || swIsFlo(r) {
					x, y := swNum(rt, l), swNum(rt, r)
					c := 0
					if x < y {
						c = -1
					} else if x > y {
						c = 1
					} else if x != y {
						return jsHFalse // a NaN operand: every ordering is false
					}
					return boolH(want(c))
				}
				if giIsInt(l) || giIsInt(r) {
					if giIsIntegral(l) && giIsIntegral(r) {
						return boolH(want(rt.giCmp(l, r)))
					}
				}
				return boolH(want(rt.jsCompare(l, r)))
			}
		}
		// The members a sized integer answers for itself. Everything else falls
		// straight through to the runtime's own js_get, so this is only ever a
		// type switch in front of the read the grammar already emitted.
		baseGet := m["js_get"]
		m["js_swget"] = func(a []uint64) uint64 {
			// The Double members. .isNaN / .isInfinite are the only way to spot
			// the values a comparison cannot: NaN != NaN.
			if f, ok := u(a[0]).(jsJFlo); ok {
				switch rt.toString(u(a[1])) {
				case "isNaN":
					return boolH(math.IsNaN(f.f))
				case "isInfinite":
					return boolH(math.IsInf(f.f, 0))
				case "isFinite":
					return boolH(!math.IsNaN(f.f) && !math.IsInf(f.f, 0))
				case "isZero":
					return boolH(f.f == 0)
				case "description":
					return rt.wrapStr(swFloStr(f.f))
				case "magnitude":
					return w(swMkFlo(math.Abs(f.f)))
				}
			}
			if b, ok := u(a[0]).(jsGInt); ok {
				switch rt.toString(u(a[1])) {
				case "description":
					return rt.wrapStr(giStr(b))
				case "bitWidth":
					return rt.wrapNum(float64(b.w))
				case "magnitude":
					if !b.u && b.v < 0 {
						return w(giNorm(-b.v, b.w, b.u))
					}
					return a[0]
				}
			}
			// The same three on a value that stayed a plain number, which is
			// where every integer inside +/- 2^53 lives.
			if f, ok := u(a[0]).(float64); ok {
				switch rt.toString(u(a[1])) {
				case "description":
					return rt.wrapStr(rt.swDesc(f))
				case "bitWidth":
					return rt.wrapNum(64)
				case "magnitude":
					return rt.wrapNum(math.Abs(f))
				}
			}
			return baseGet(a)
		}
		rel("js_swlt", func(c int) bool { return c < 0 })
		rel("js_swle", func(c int) bool { return c <= 0 })
		rel("js_swgt", func(c int) bool { return c > 0 })
		rel("js_swge", func(c int) bool { return c >= 0 })
	})
}

// ----------------------------------------------------------------------------
// Integers
//
// Swift's Int is 64 bit and Int8 ... UInt64 are exactly what they say, so both
// Swift halves ride the sized-integer box of jsrtint.go / the sz* layer of
// languages/lib/interp-core.js. What is Swift specific is the SELECTION rule:
// this value model has one number type, and an operation is integer arithmetic
// exactly when both operands are integral - a Double that happens to hold a
// whole number is therefore treated as an Int, the one place the model shows.
//
// DELIBERATE DIVERGENCE, both halves: real Swift TRAPS on signed overflow
// (`Int.max + 1` is a runtime crash) and reserves &+ / &- / &* for wrapping.
// Here + - * WRAP at the operand's width, exactly like &+ / &- / &*. The
// divergence is confined to programs Swift would have killed outright, and is
// documented rather than hidden.

// ----------------------------------------------------------------------------
// Doubles
//
// Swift's Double is a DIFFERENT type from Int, and the handle runtime models
// every number as one JS double, so `7.0 / 2.0` arrived here with exactly the
// operands `7 / 2` has and divided as integers (3, not 3.5). No dynamic rule can
// repair that - 7.0 has no fraction to detect - so the type goes ON the value,
// exactly as commit 636e326 did for Java/Kotlin/Go/C#. Swift reuses THAT box,
// jsJFlo of jsrtjvm.go, which rt.truthy / rt.toNumber / rt.typeOf / rt.strictEq
// already understand; only the RENDERING is Swift's own (swFloStr below), and
// every Swift print path goes through swRender, so the box's style tag is never
// consulted for a Swift value.
//
// Invariant, in both halves:
//
//	a plain number / a jsGInt  ==  an Int (or a sized integer type)
//	a jsJFlo                   ==  a Double (or a Float)
//
// DELIBERATE SIMPLIFICATION, both halves: Float is an ALIAS of Double, i.e. it is
// modelled at 64 bit precision. Every value both types hold exactly prints and
// computes the same; a computation that LOSES precision does not - real swift
// prints `Float(1) / Float(3)` as 0.33333334 and this prints 0.3333333333333333.
// Modelling the 32 bit width honestly needs float32 rounding after every operator
// AND a float32 shortest-round-trip renderer in the frozen JS subset, which has
// neither toPrecision nor Math.fround. Aliasing was chosen over refusing `Float`
// outright because a program that writes Float is asking for floating point.
//
// DELIBERATE SIMPLIFICATION, both halves: real Swift does not implicitly convert
// between Int and Double (`x + 2.0` with `let x = 5` is a compile ERROR, and only
// an untyped literal may take the other side's type). This front end has no type
// checker, so a mixed operation evaluates in floating point - which is the right
// answer for the literal case, `1 + 2.0 == 3.0` - instead of being rejected.

func swMkFlo(f float64) jsJFlo { return jsJFlo{f: f} }

func swIsFlo(v interface{}) bool { _, ok := v.(jsJFlo); return ok }

// swIsNumeric reports whether a value takes part in numeric equality at all, so
// that 1.0 == "x" is false rather than 1.0 == 0.
func swIsNumeric(v interface{}) bool {
	switch v.(type) {
	case jsJFlo, jsGInt, float64:
		return true
	}
	return false
}

// swRoundHalfAway is Double.rounded(), i.e. .toNearestOrAwayFromZero - which is
// what Go's math.Round does, and NOT what a half-to-even or half-up rule gives:
// (-2.5).rounded() is -3.
func swRoundHalfAway(f float64) float64 { return math.Round(f) }

// swFloStr is Swift's Double.description (SwiftDtoa): the shortest decimal that
// round-trips, written as a plain decimal when the scientific exponent is in
// -4 ... 15 and in computerized scientific notation - two digit minimum exponent,
// always signed - outside it, with the integral case always carrying a ".0". So
// 1e15 prints 1000000000000000.0 and 1e16 prints 1e+16. The infinities are
// "inf"/"-inf" and the NaN is "nan". Every boundary here was read off swift 6.1.2
// rather than remembered. The JS twin is swFloStr in swift-interpreter.abnf and
// the two MUST stay in step, because ./test.sh --cross diffs them.
func swFloStr(d float64) string {
	if math.IsNaN(d) {
		return "nan"
	}
	if math.IsInf(d, 1) {
		return "inf"
	}
	if math.IsInf(d, -1) {
		return "-inf"
	}
	neg := math.Signbit(d)
	a := math.Abs(d)
	s := swFloDigits(a)
	if neg {
		return "-" + s
	}
	return s
}

// swFloDigits renders a >= 0. strconv's 'e' form gives the shortest round-tripping
// digit run and the scientific exponent directly, which is exactly the pair
// SwiftDtoa formats from.
func swFloDigits(a float64) string {
	if a == 0 {
		return "0.0"
	}
	s := strconv.FormatFloat(a, 'e', -1, 64)
	i := strings.IndexByte(s, 'e')
	mant, exp := s[:i], s[i+1:]
	e10, _ := strconv.Atoi(exp)
	digits := strings.Replace(mant, ".", "", 1)
	if e10 < -4 || e10 >= 16 {
		out := digits[:1]
		if len(digits) > 1 {
			out += "." + digits[1:]
		}
		sign := "+"
		if e10 < 0 {
			sign = "-"
			e10 = -e10
		}
		if e10 < 10 {
			return out + "e" + sign + "0" + strconv.Itoa(e10)
		}
		return out + "e" + sign + strconv.Itoa(e10)
	}
	pt := e10 + 1 // digits before the decimal point
	if pt <= 0 {
		return "0." + strings.Repeat("0", -pt) + digits
	}
	if pt >= len(digits) {
		return digits + strings.Repeat("0", pt-len(digits)) + ".0"
	}
	return digits[:pt] + "." + digits[pt:]
}

// swFloText reports whether the WHOLE string is a decimal float, for the failable
// Double("3.5"): swift answers nil for "3.5x", where a lenient parse stops at the
// first bad character and answers 3.5.
func swFloText(s string) bool {
	i := 0
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		i++
	}
	d0 := i
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	seen := i > d0
	if i < len(s) && s[i] == '.' {
		i++
		d1 := i
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
		}
		if i > d1 {
			seen = true
		}
	}
	if !seen {
		return false
	}
	if i < len(s) && (s[i] == 'e' || s[i] == 'E') {
		i++
		if i < len(s) && (s[i] == '+' || s[i] == '-') {
			i++
		}
		d2 := i
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
		}
		if i == d2 {
			return false
		}
	}
	return i == len(s)
}

// swIsWhole is the interpreter's swWhole: a sized-integer box, or a Double whose
// value is a whole number.
func swIsWhole(v interface{}) bool {
	switch t := v.(type) {
	case jsGInt:
		return true
	case float64:
		return !math.IsNaN(t) && !math.IsInf(t, 0) && t == math.Trunc(t)
	}
	return false
}

// swNum is the plain-number reading of an operand, for the Double side of a
// mixed operation.
func swNum(rt *jsrt, v interface{}) float64 { return giFloat(rt, v) }

// swIntFromStr is Int("17") / UInt8("300"): the failable string initializer. It
// reports false for anything that is not a complete, in-range decimal integer.
func swIntFromStr(s string, w uint8, u bool) (interface{}, bool) {
	i, neg := 0, false
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		neg = s[i] == '-'
		i++
	}
	if i >= len(s) {
		return nil, false
	}
	var acc uint64
	for k := i; k < len(s); k++ {
		c := s[k]
		if c < '0' || c > '9' {
			return nil, false
		}
		d := uint64(c - '0')
		if acc > (1<<64-1-d)/10 {
			return nil, false // beyond 64 bits: out of range for every width
		}
		acc = acc*10 + d
	}
	v := int64(acc)
	if neg {
		v = -v
	}
	// The range check: the truncated value has to read back as what came in.
	out := giNorm(v, w, u)
	var back string
	if b, ok := out.(jsGInt); ok {
		back = giStr(b)
	} else {
		back = strconv.FormatInt(int64(out.(float64)), 10)
	}
	if back != swIntCanon(s, neg) {
		return nil, false
	}
	return out, true
}

// swIntCanon is the canonical decimal text of a digit run - the sign and the
// leading zeros removed - for the range check above.
func swIntCanon(s string, neg bool) string {
	d := strings.TrimLeft(strings.TrimLeft(s, "+-"), "0")
	if d == "" {
		return "0"
	}
	if neg {
		return "-" + d
	}
	return d
}

// ----------------------------------------------------------------------------
// Rendering

// swDesc is Swift's `String(describing:)`: the TOP-LEVEL form, in which a String
// renders as itself. swDescIn is the form used for an element inside a
// collection, a tuple or a struct, where a String is quoted and escaped.
func (rt *jsrt) swDesc(v interface{}) string { return rt.swRender(v, false, 0) }

func (rt *jsrt) swDescIn(v interface{}) string { return rt.swRender(v, true, 0) }

// swQuote is Swift's debugDescription of a String: double quotes and the four
// escapes Swift itself emits.
func swQuote(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`, "\t", `\t`, "\r", `\r`, "\x00", `\0`)
	return `"` + r.Replace(s) + `"`
}

// swRender does the work. `nested` selects the quoted form for strings; `depth`
// only guards against a cyclic object graph (a class instance can point back at
// its owner) so the renderer cannot recurse forever.
func (rt *jsrt) swRender(v interface{}, nested bool, depth int) string {
	if depth > 32 {
		return "..."
	}
	switch t := v.(type) {
	case jsNullT, jsUndefT:
		return "nil"
	case jsGInt:
		// A sized integer prints as its DECIMAL text, not through a double:
		// Int.max would otherwise render as 9.223372036854776e+18.
		return giStr(t)
	case jsJFlo:
		// A Double prints through SWIFT's float rendering: 1.0 is "1.0" and the
		// infinity is "inf", where the shared toString would answer the style of
		// whichever language owns the box's tag.
		return swFloStr(t.f)
	case string:
		if nested {
			return swQuote(t)
		}
		return t
	case *jsArray:
		parts := make([]string, len(t.elems))
		for i, e := range t.elems {
			parts[i] = rt.swRender(e, true, depth+1)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case *jsObject:
		if s, ok := rt.swObjDesc(t, depth); ok {
			return s
		}
	}
	return rt.toString(v)
}

// swObjDesc renders the object shapes the two Swift grammars build. It reports
// false for a shape it does not know, which falls back to the generic ToString.
func (rt *jsrt) swObjDesc(o *jsObject, depth int) (string, bool) {
	// A user `description` (CustomStringConvertible) wins over every built-in
	// rendering, exactly as it does in Swift. Both halves store a computed
	// property as the method __get_<name> on the type descriptor.
	if g, ok := swFindMember(o, "__get_description"); ok {
		return rt.swDesc(rt.call(g, jsUndef, []interface{}{o})), true
	}
	if d, ok := o.props["description"]; ok && !isCallable(d) {
		return rt.swDesc(d), true
	}

	// Dictionary: ["k": v, ...], and the empty one is [:].
	if keys, vals, ok := dictParts(o); ok {
		if len(keys.elems) == 0 {
			return "[:]", true
		}
		parts := make([]string, 0, len(keys.elems))
		for i, k := range keys.elems {
			var val interface{} = jsUndef
			if i < len(vals.elems) {
				val = vals.elems[i]
			}
			parts = append(parts, rt.swRender(k, true, depth+1)+": "+rt.swRender(val, true, depth+1))
		}
		return "[" + strings.Join(parts, ", ") + "]", true
	}

	// Tuple: (1, "two") / (x: 1, y: "s"). The element labels ride along as
	// __tlabels when the tuple has any (an unlabelled slot is "").
	if n, ok := swNumProp(o, "__tuple"); ok {
		labels := swStrArray(o.props["__tlabels"])
		parts := make([]string, 0, n)
		for i := 0; i < n; i++ {
			s := rt.swRender(o.props[itoa(i)], true, depth+1)
			if i < len(labels) && labels[i] != "" {
				s = labels[i] + ": " + s
			}
			parts = append(parts, s)
		}
		return "(" + strings.Join(parts, ", ") + ")", true
	}

	// An enum case: `c`, `a(1, "x")`, `b(n: 2)`. A raw-value enum still prints
	// the CASE NAME, not the raw value.
	if tag, ok := o.props["__ecase"]; ok && tag == true {
		name, _ := o.props["__ename"].(string)
		vals, _ := o.props["__vals"].(*jsArray)
		if vals == nil || len(vals.elems) == 0 {
			return name, true
		}
		labels := swStrArray(o.props["__labels"])
		parts := make([]string, 0, len(vals.elems))
		for i, e := range vals.elems {
			s := rt.swRender(e, true, depth+1)
			if i < len(labels) && labels[i] != "" {
				s = labels[i] + ": " + s
			}
			parts = append(parts, s)
		}
		return name + "(" + strings.Join(parts, ", ") + ")", true
	}

	// A Set literal, where the grammars build one.
	if tag, ok := o.props["__set"]; ok && tag == true {
		if el, ok := o.props["elems"].(*jsArray); ok {
			parts := make([]string, len(el.elems))
			for i, e := range el.elems {
				parts[i] = rt.swRender(e, true, depth+1)
			}
			return "[" + strings.Join(parts, ", ") + "]", true
		}
	}

	// The type descriptor itself (`print(P.self)`) is its name.
	if _, ok := o.props["__isclass"]; ok {
		if n, ok := o.props["__name"].(string); ok {
			return n, true
		}
	}
	if _, ok := o.props["__isenum"]; ok {
		if n, ok := o.props["__name"].(string); ok {
			return n, true
		}
	}

	// An instance. A STRUCT prints memberwise - P(x: 1, y: 2) - because Swift
	// synthesizes that reflection-based description for every struct; a CLASS
	// prints its type name (real Swift qualifies it with the module, which this
	// runtime does not have).
	cls, _ := o.props["__class"].(*jsObject)
	if cls == nil {
		return "", false
	}
	name, _ := cls.props["__name"].(string)
	if name == "" {
		return "", false
	}
	if isStruct, ok := cls.props["__struct"]; !ok || isStruct != true {
		return name, true
	}
	flds := swStrArray(cls.props["__fields"])
	parts := make([]string, 0, len(flds))
	for _, f := range flds {
		val, ok := o.props[f]
		if !ok {
			val = jsUndef
		}
		parts = append(parts, f+": "+rt.swRender(val, true, depth+1))
	}
	return name + "(" + strings.Join(parts, ", ") + ")", true
}

// swFindMember walks the __class / __super chain for a callable member, the same
// walk classFind does in swift-interpreter.abnf.
func swFindMember(v interface{}, name string) (interface{}, bool) {
	o, ok := v.(*jsObject)
	if !ok {
		return nil, false
	}
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

func swNumProp(o *jsObject, key string) (int, bool) {
	f, ok := o.props[key].(float64)
	if !ok {
		return 0, false
	}
	return int(f), true
}

func swStrArray(v interface{}) []string {
	arr, ok := v.(*jsArray)
	if !ok {
		return nil
	}
	out := make([]string, len(arr.elems))
	for i, e := range arr.elems {
		out[i], _ = e.(string)
	}
	return out
}

// ----------------------------------------------------------------------------
// Equality

// swEqual is Swift's ==. Array, Dictionary and tuple are VALUE types in Swift, so
// they compare element by element; js_seq compared them by identity and answered
// false for every one of them. An enum case compares by name plus associated
// values (which the compiler also encodes in __eqkey; this path agrees with it
// and additionally handles the shapes __eqkey cannot round-trip).
func (rt *jsrt) swEqual(l, r interface{}) bool {
	return rt.swEqualDepth(l, r, 0)
}

func (rt *jsrt) swEqualDepth(l, r interface{}, depth int) bool {
	if depth > 32 {
		return false
	}
	if isNullish(l) || isNullish(r) {
		return isNullish(l) && isNullish(r)
	}
	// A float box compares BY VALUE and ACROSS the box boundary: `1.0 == 1` holds
	// in Swift (the untyped literal takes the Double type). It has to be spelled
	// before the integer path, which would compare 1.5 against Int8(1) as the
	// truncated 1.
	if swIsFlo(l) || swIsFlo(r) {
		if !swIsNumeric(l) || !swIsNumeric(r) {
			return false
		}
		return swNum(rt, l) == swNum(rt, r)
	}
	// A sized-integer box is a value of its own, so two integers have to be
	// compared BY VALUE here rather than falling through to strictEq, which
	// would call a box and a plain number different.
	if giIsInt(l) || giIsInt(r) {
		return rt.giEq(l, r)
	}
	if la, ok := l.(*jsArray); ok {
		ra, ok := r.(*jsArray)
		if !ok || len(la.elems) != len(ra.elems) {
			return false
		}
		for i := range la.elems {
			if !rt.swEqualDepth(la.elems[i], ra.elems[i], depth+1) {
				return false
			}
		}
		return true
	}
	lo, lok := l.(*jsObject)
	ro, rok := r.(*jsObject)
	if lok && rok {
		if lo == ro {
			return true
		}
		if lk, lv, ok := dictParts(lo); ok {
			rk, rv, ok := dictParts(ro)
			if !ok || len(lk.elems) != len(rk.elems) {
				return false
			}
			// A Dictionary is unordered, so every entry of the left must have a
			// matching entry somewhere in the right.
			for i, k := range lk.elems {
				found := false
				for j, k2 := range rk.elems {
					if rt.swEqualDepth(k, k2, depth+1) && rt.swEqualDepth(lv.elems[i], rv.elems[j], depth+1) {
						found = true
						break
					}
				}
				if !found {
					return false
				}
			}
			return true
		}
		if ln, ok := swNumProp(lo, "__tuple"); ok {
			rn, ok := swNumProp(ro, "__tuple")
			if !ok || ln != rn {
				return false
			}
			for i := 0; i < ln; i++ {
				if !rt.swEqualDepth(lo.props[itoa(i)], ro.props[itoa(i)], depth+1) {
					return false
				}
			}
			return true
		}
		if tag, ok := lo.props["__ecase"]; ok && tag == true {
			if tag2, ok := ro.props["__ecase"]; !ok || tag2 != true {
				return false
			}
			if lo.props["__ename"] != ro.props["__ename"] {
				return false
			}
			lv, _ := lo.props["__vals"].(*jsArray)
			rv, _ := ro.props["__vals"].(*jsArray)
			if lv == nil || rv == nil {
				return lv == nil && rv == nil
			}
			if len(lv.elems) != len(rv.elems) {
				return false
			}
			for i := range lv.elems {
				if !rt.swEqualDepth(lv.elems[i], rv.elems[i], depth+1) {
					return false
				}
			}
			return true
		}
		return false
	}
	return rt.strictEq(l, r)
}
