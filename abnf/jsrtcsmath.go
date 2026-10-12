package abnf

import (
	"math"
	"math/bits"
)

// System.Math's and System.MathF's static members: the Go twin's port of
// languages/lib/csharp-math.js, whose header has the rules.

func csmIsFlo(k string) bool { return k == "d" || k == "f" }

func csmFloOf(hasD bool, x float64) interface{} {
	if hasD {
		return jsJFlo{f: x, sty: floCS}
	}
	return jsJFlo{f: float64(float32(x)), sty: floCSF}
}

func csmRoundEven(d float64) float64 {
	if d != d || d == 0 || math.Abs(d) >= 4503599627370496 {
		return d
	}
	r := math.Floor(d)
	f := d - r
	if f > 0.5 || (f == 0.5 && r-2*math.Floor(r/2) != 0) {
		r++
	}
	if r == 0 && d < 0 {
		return math.Copysign(0, -1)
	}
	return r
}

func csmF32(x float64) float64 { return float64(float32(x)) }

func csmRoundMode(d float64, mode int) float64 {
	if mode == 0 {
		return csmRoundEven(d)
	}
	if d != d || d == 0 || math.Abs(d) >= 4503599627370496 {
		return d
	}
	r := math.Trunc(d)
	switch mode {
	case 1:
		if f := d - r; f >= 0.5 {
			r++
		} else if f <= -0.5 {
			r--
		}
	case 3:
		r = math.Floor(d)
	case 4:
		r = math.Ceil(d)
	}
	if r == 0 && d < 0 {
		return math.Copysign(0, -1)
	}
	return r
}

func (rt *jsrt) csmRoundDigits(d, digits float64, mode int, f32 bool) float64 {
	most, limit := 15.0, 10000000000000000.0
	if f32 {
		most, limit = 6, 100000000
	}
	if digits < 0 || digits > most {
		rt.csThrow("ArgumentOutOfRangeException", "Rounding digits must be between 0 and "+jsNumString(most)+", inclusive. (Parameter 'digits')")
	}
	if !(math.Abs(d) < limit) {
		return d
	}
	p := 1.0
	for i := 0.0; i < digits; i++ {
		p *= 10
	}
	if f32 {
		return csmF32(csmRoundMode(csmF32(d*p), mode) / p)
	}
	return csmRoundMode(d*p, mode) / p
}

func csmCopySign(x, y float64) float64 {
	if y != y {
		return math.Abs(x)
	}
	return math.Copysign(x, y)
}

func (rt *jsrt) csmBigMul(args []interface{}) interface{} {
	ts := []int{csmIntType(argAt(args, 0)), csmIntType(argAt(args, 1))}
	if len(args) == 2 {
		t := csmIntOverload(ts, []int{32, 33})
		if t == 0 {
			rt.fail("Math.BigMul of two 64-bit integers (an Int128) is not modelled")
		}
		return rt.csArith("*", csConvTo(rt, args[0], 64, t == 33), csConvTo(rt, args[1], 64, t == 33))
	}
	w := csmIntOverload(ts, []int{64, 65})
	a, b := uint64(giVal(rt, args[0])), uint64(giVal(rt, args[1]))
	hi, lo := bits.Mul64(a, b)
	if w == 64 {
		if int64(a) < 0 {
			hi -= b
		}
		if int64(b) < 0 {
			hi -= a
		}
	}
	rt.csxRefSet(argAt(args, 2), csNorm(int64(lo), 64, w == 65))
	return csNorm(int64(hi), 64, w == 65)
}

func (rt *jsrt) csmDivRem(args []interface{}) interface{} {
	ts := []int{csmIntType(argAt(args, 0)), csmIntType(argAt(args, 1))}
	cands := csmAllInts
	if len(args) == 3 {
		cands = []int{32, 64}
	}
	t := csmIntOverload(ts, cands)
	a, b := rt.csmAtType(args[0], t), rt.csmAtType(args[1], t)
	q := rt.csmAtType(rt.csArith("/", a, b), t)
	r := rt.csmAtType(rt.csArith("-", a, rt.csArith("*", q, b)), t)
	if len(args) == 3 {
		rt.csxRefSet(args[2], r)
		return q
	}
	tup := newJSObject()
	tup.set("__tuple", float64(2))
	tup.set("Item1", q)
	tup.set("Item2", r)
	tup.set("Quotient", q)
	tup.set("Remainder", r)
	return tup
}

// csmIntType codes an integral value's type as its width plus 1 when unsigned; a
// char converts as ushort does.
func csmIntType(v interface{}) int {
	switch t := v.(type) {
	case float64:
		return 32
	case jsGInt:
		if t.u {
			return int(t.w) + 1
		}
		return int(t.w)
	case jsChar:
		return 17
	}
	return 0
}

func csmIntConv(s, t int) bool {
	if s == t {
		return true
	}
	if t-t%2 <= s-s%2 {
		return false
	}
	return t%2 == 0 || s%2 == 1
}

var csmAllInts = []int{8, 9, 16, 17, 32, 33, 64, 65}

func csmIntOverload(types []int, cands []int) int {
	for _, c := range cands {
		ok := true
		for _, t := range types {
			if !csmIntConv(t, c) {
				ok = false
			}
		}
		if ok {
			return c
		}
	}
	return 0
}

func (rt *jsrt) csmAtType(v interface{}, t int) interface{} {
	if t == 0 {
		return v
	}
	return csConvTo(rt, v, uint8(t-t%2), t%2 == 1)
}

func (rt *jsrt) csmAbs(x interface{}) interface{} {
	k := csqNumKind(x)
	if csmIsFlo(k) {
		return csmFloOf(k == "d", math.Abs(rt.toNumber(x)))
	}
	t := csmIntType(x)
	if t != 32 && t != 64 {
		t = csmIntOverload([]int{t}, []int{8, 16, 32, 64})
	}
	if !rt.csxLt(x, float64(0)) {
		return rt.csmAtType(x, t)
	}
	r := rt.csmAtType(rt.csArith("-", float64(0), x), t)
	if rt.csxLt(r, float64(0)) {
		rt.csThrow("OverflowException", "Negating the minimum value of a twos complement number is invalid.")
	}
	return r
}

func (rt *jsrt) csmSign(x interface{}) interface{} {
	d := rt.toNumber(x)
	if d != d {
		rt.csThrow("ArithmeticException", "Function does not accept floating point Not-a-Number values.")
	}
	if d > 0 {
		return float64(1)
	}
	if d < 0 {
		return float64(-1)
	}
	return float64(0)
}

func (rt *jsrt) csmMinMax(a, b interface{}, max bool) interface{} {
	ka, kb := csqNumKind(a), csqNumKind(b)
	if csmIsFlo(ka) || csmIsFlo(kb) {
		x, y := rt.toNumber(a), rt.toNumber(b)
		r := x
		if y != y {
			r = y
		} else if x == y {
			if x == 0 && (1/x > 0) != max {
				r = y
			}
		} else if x == x && (y > x) == max {
			r = y
		}
		return csmFloOf(ka == "d" || kb == "d", r)
	}
	pick := a
	if rt.csxLt(a, b) == max {
		pick = b
	}
	ta, tb := csmIntType(a), csmIntType(b)
	if ta == tb && ta != 17 {
		return pick
	}
	return rt.csmAtType(pick, csmIntOverload([]int{ta, tb}, csmAllInts))
}

func (rt *jsrt) csmClamp(v, lo, hi interface{}) interface{} {
	if rt.csxLt(hi, lo) {
		rt.csThrow("ArgumentException", "'"+rt.cspStr(lo)+"' cannot be greater than "+rt.cspStr(hi)+".")
	}
	r := v
	if rt.csxLt(v, lo) {
		r = lo
	} else if rt.csxLt(hi, v) {
		r = hi
	}
	kv, kl, kh := csqNumKind(v), csqNumKind(lo), csqNumKind(hi)
	if csmIsFlo(kv) || csmIsFlo(kl) || csmIsFlo(kh) {
		return csmFloOf(kv == "d" || kl == "d" || kh == "d", rt.toNumber(r))
	}
	tv, tl, th := csmIntType(v), csmIntType(lo), csmIntType(hi)
	if tv == tl && tv == th && tv != 17 {
		return r
	}
	return rt.csmAtType(r, csmIntOverload([]int{tv, tl, th}, csmAllInts))
}

func csmPow(x, y float64) float64 {
	if x == 1 {
		return 1
	}
	if x == -1 && math.IsInf(y, 0) {
		return 1
	}
	return crPow(x, y)
}

func csmLogBase(a, b float64) float64 {
	if a != a {
		return a
	}
	if b != b {
		return b
	}
	if b == 1 || (a != 1 && (b == 0 || math.IsInf(b, 1))) {
		return math.NaN()
	}
	return math.Log(a) / math.Log(b)
}

var csmUnary = map[string]func(float64) float64{
	"Sqrt": math.Sqrt, "Floor": math.Floor, "Ceiling": math.Ceil, "Truncate": math.Trunc,
	"Round": csmRoundEven, "Exp": math.Exp, "Log": math.Log, "Log10": math.Log10,
	"Log2": math.Log2, "Sin": math.Sin, "Cos": math.Cos, "Tan": math.Tan, "Asin": math.Asin,
	"Acos": math.Acos, "Atan": math.Atan, "Sinh": math.Sinh, "Cosh": math.Cosh,
	"Tanh": math.Tanh, "Cbrt": math.Cbrt,
}

var csmDoubleConstNames = []string{"NaN", "PositiveInfinity", "NegativeInfinity", "Epsilon", "MaxValue", "MinValue"}

func csmDoubleConst(name string, f32 bool) float64 {
	tiny, huge := math.SmallestNonzeroFloat64, math.MaxFloat64
	if f32 {
		tiny, huge = math.SmallestNonzeroFloat32, math.MaxFloat32
	}
	switch name {
	case "NaN":
		return math.NaN()
	case "PositiveInfinity":
		return math.Inf(1)
	case "NegativeInfinity":
		return math.Inf(-1)
	case "Epsilon":
		return tiny
	case "MaxValue":
		return huge
	}
	return -huge
}

var csmDoubleIsNames = []string{"IsNaN", "IsInfinity", "IsPositiveInfinity", "IsNegativeInfinity", "IsFinite"}

func csmDoubleIs(name string, d float64) bool {
	switch name {
	case "IsNaN":
		return d != d
	case "IsInfinity":
		return math.IsInf(d, 0)
	case "IsPositiveInfinity":
		return math.IsInf(d, 1)
	case "IsNegativeInfinity":
		return math.IsInf(d, -1)
	}
	return !math.IsNaN(d) && !math.IsInf(d, 0)
}

func (rt *jsrt) csmStatic(name string, args0 []interface{}, f32 bool) interface{} {
	args := args0
	if f32 {
		args = make([]interface{}, len(args0))
		for i, a := range args0 {
			if name == "Round" && i > 0 {
				args[i] = a
			} else {
				args[i] = jsJFlo{f: csmF32(rt.toNumber(a)), sty: floCSF}
			}
		}
	}
	n := len(args)
	if name == "Round" && n > 1 {
		if e, ok := csxIsEnumBox(args[n-1]); ok {
			digits := 0.0
			if n == 3 {
				digits = rt.toNumber(args[1])
			}
			return csmFloOf(!f32, rt.csmRoundDigits(rt.toNumber(args[0]), digits, csxEnumV(e), f32))
		}
	}
	if !f32 && name == "BigMul" {
		return rt.csmBigMul(args)
	}
	if !f32 && name == "DivRem" {
		return rt.csmDivRem(args)
	}
	switch n {
	case 1:
		switch name {
		case "Abs":
			return rt.csmAbs(args[0])
		case "Sign":
			return rt.csmSign(args[0])
		}
		if f, ok := csmUnary[name]; ok {
			return csmFloOf(!f32, f(rt.toNumber(args[0])))
		}
	case 2:
		switch name {
		case "Max":
			return rt.csmMinMax(args[0], args[1], true)
		case "Min":
			return rt.csmMinMax(args[0], args[1], false)
		}
		a, b := rt.toNumber(args[0]), rt.toNumber(args[1])
		switch name {
		case "Pow":
			return csmFloOf(!f32, csmPow(a, b))
		case "Atan2":
			return csmFloOf(!f32, math.Atan2(a, b))
		case "Log":
			return csmFloOf(!f32, csmLogBase(a, b))
		case "Round":
			return csmFloOf(!f32, rt.csmRoundDigits(a, b, 0, f32))
		case "CopySign":
			return csmFloOf(!f32, csmCopySign(a, b))
		}
	case 3:
		if name == "Clamp" {
			return rt.csmClamp(args[0], args[1], args[2])
		}
	}
	prefix := "Math."
	if f32 {
		prefix = "MathF."
	}
	rt.fail("%s%s with %d arguments is not modelled", prefix, name, n)
	return nil
}
