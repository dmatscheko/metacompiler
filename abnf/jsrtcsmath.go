package abnf

import "math"

// System.Math's static members: the Go twin's port of
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

func (rt *jsrt) csmRoundDigits(d, digits float64) float64 {
	if digits < 0 || digits > 15 {
		rt.csThrow("ArgumentOutOfRangeException", "Rounding digits must be between 0 and 15, inclusive. (Parameter 'digits')")
	}
	if !(math.Abs(d) < 10000000000000000) {
		return d
	}
	p := 1.0
	for i := 0.0; i < digits; i++ {
		p *= 10
	}
	return csmRoundEven(d*p) / p
}

func (rt *jsrt) csmAbs(x interface{}) interface{} {
	k := csqNumKind(x)
	if csmIsFlo(k) {
		return csmFloOf(k == "d", math.Abs(rt.toNumber(x)))
	}
	if !rt.csxLt(x, float64(0)) {
		return x
	}
	r := rt.csArith("-", float64(0), x)
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
	lt := rt.csxLt(a, b)
	pick, other := a, b
	if lt == max {
		pick, other = b, a
	}
	if ka == kb {
		return pick
	}
	return rt.csArith("+", pick, rt.csArith("-", other, other))
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
	return r
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

func (rt *jsrt) csmStatic(name string, args []interface{}) interface{} {
	switch len(args) {
	case 1:
		switch name {
		case "Abs":
			return rt.csmAbs(args[0])
		case "Sign":
			return rt.csmSign(args[0])
		}
		if f, ok := csmUnary[name]; ok {
			return jsJFlo{f: f(rt.toNumber(args[0])), sty: floCS}
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
			return jsJFlo{f: csmPow(a, b), sty: floCS}
		case "Atan2":
			return jsJFlo{f: math.Atan2(a, b), sty: floCS}
		case "Log":
			return jsJFlo{f: csmLogBase(a, b), sty: floCS}
		case "Round":
			return jsJFlo{f: rt.csmRoundDigits(a, b), sty: floCS}
		}
	case 3:
		if name == "Clamp" {
			return rt.csmClamp(args[0], args[1], args[2])
		}
	}
	rt.fail("Math.%s with %d arguments is not modelled", name, len(args))
	return nil
}
