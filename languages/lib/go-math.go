// Package math: Go's math package WRITTEN IN GO, compiled with a program that imports
// "math" (core.importFile in both go grammars), so one implementation serves the
// interpreter, llvm.Run and a native build alike.
//
// Every function here is EXACT - bit for bit Go's answer - because it is built from
// IEEE additions, multiplications and divisions by powers of two that do not round,
// plus at most one final rounding where Go rounds once too. Sqrt is the host's
// correctly rounded square root (goMathSqrt, bound by both grammars). Pow follows
// Go's own algorithm, which for an integral exponent is the same sequence of
// roundings; a non-integral exponent other than ±0.5 goes through Exp and Log, and
// those, like every transcendental function, are left out: Go's arm64 build fuses
// a*b+c in them, so a port would differ from it in the last bit.
//
// A NaN's sign and payload are not modelled: Signbit(NaN) is false.
package math

const (
	E                      = 2.718281828459045
	Pi                     = 3.141592653589793
	Phi                    = 1.618033988749895
	Sqrt2                  = 1.4142135623730951
	SqrtE                  = 1.6487212707001282
	SqrtPi                 = 1.772453850905516
	SqrtPhi                = 1.272019649514069
	Ln2                    = 0.6931471805599453
	Log2E                  = 1.4426950408889634
	Ln10                   = 2.302585092994046
	Log10E                 = 0.4342944819032518
	MaxFloat64             = 1.7976931348623157e+308
	SmallestNonzeroFloat64 = 5e-324
	MaxFloat32             = 3.4028234663852886e+38
	SmallestNonzeroFloat32 = 1.401298464324817e-45

	MaxInt    = 1<<63 - 1
	MinInt    = -1 << 63
	MaxInt8   = 127
	MinInt8   = -128
	MaxInt16  = 32767
	MinInt16  = -32768
	MaxInt32  = 2147483647
	MinInt32  = -2147483648
	MaxInt64  = 1<<63 - 1
	MinInt64  = -1 << 63
	MaxUint8  = 255
	MaxUint16 = 65535
	MaxUint32 = 4294967295
)

// 2^52: from here on every float64 is an integer.
const two52 = 4503599627370496.0

func Inf(sign int) float64 {
	big := MaxFloat64
	if sign >= 0 {
		return big * 2
	}
	return -big * 2
}

func NaN() float64 {
	inf := Inf(1)
	return inf - inf
}

func IsNaN(f float64) bool { return f != f }

func IsInf(f float64, sign int) bool {
	return sign >= 0 && f > MaxFloat64 || sign <= 0 && f < -MaxFloat64
}

// A zero's sign shows in the infinity it divides one into.
func Signbit(x float64) bool { return x < 0 || x == 0 && 1/x < 0 }

func Abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	if x == 0 {
		return 0
	}
	return x
}

func Copysign(f, sign float64) float64 {
	if Signbit(f) != Signbit(sign) {
		return -f
	}
	return f
}

func Sqrt(x float64) float64 { return goMathSqrt(x) }

// Floor below 2^52: adding and taking away 2^52 rounds to an integer, which is then
// stepped down if it rounded up. A zero, an infinity, a NaN and an integer are
// their own floor.
func Floor(x float64) float64 {
	if x == 0 || IsNaN(x) || x >= two52 || x <= -two52 {
		return x
	}
	var r float64
	if x > 0 {
		r = (x + two52) - two52
	} else {
		r = (x - two52) + two52
	}
	if r > x {
		r--
	}
	return r
}

func Ceil(x float64) float64 { return -Floor(-x) }

func Trunc(x float64) float64 {
	if x < 0 {
		return Ceil(x)
	}
	return Floor(x)
}

// Round rounds half away from zero; x - Trunc(x) is exact.
func Round(x float64) float64 {
	t := Trunc(x)
	if Abs(x-t) >= 0.5 {
		t += Copysign(1, x)
	}
	return t
}

func RoundToEven(x float64) float64 {
	t := Trunc(x)
	d := Abs(x - t)
	if d > 0.5 || d == 0.5 && t-2*Trunc(t/2) != 0 {
		t += Copysign(1, x)
	}
	return t
}

func Modf(f float64) (float64, float64) {
	if f < 1 {
		switch {
		case f < 0:
			ip, frac := Modf(-f)
			return -ip, -frac
		case f == 0:
			return f, f
		}
		return 0.0, f
	}
	ip := Trunc(f)
	return ip, f - ip
}

// pow2 is 2^n for -1074 <= n <= 1023, built by exact doublings and halvings.
func pow2(n int) float64 {
	p := 1.0
	for n >= 64 {
		p *= 18446744073709551616.0
		n -= 64
	}
	for n > 0 {
		p *= 2
		n--
	}
	for n <= -64 && p >= 1 {
		p *= 1 / 18446744073709551616.0
		n += 64
	}
	for n < 0 {
		p *= 0.5
		n++
	}
	return p
}

// Frexp splits f into frac in [0.5, 1) and exp with f == frac * 2^exp. Scaling by a
// power of two between the normal numbers is exact.
func Frexp(f float64) (frac float64, exp int) {
	if f == 0 || IsNaN(f) || IsInf(f, 0) {
		return f, 0
	}
	a := Abs(f)
	for a >= 18446744073709551616.0 {
		a *= 1 / 18446744073709551616.0
		exp += 64
	}
	for a < 1/18446744073709551616.0 {
		a *= 18446744073709551616.0
		exp -= 64
	}
	for a >= 1 {
		a *= 0.5
		exp++
	}
	for a < 0.5 {
		a *= 2
		exp--
	}
	return Copysign(a, f), exp
}

// Ldexp is frac * 2^exp rounded ONCE: the scaling stays among the normal numbers
// until the last multiplication, which is the only one that can round.
func Ldexp(frac float64, exp int) float64 {
	if frac == 0 || IsNaN(frac) || IsInf(frac, 0) {
		return frac
	}
	m, e := Frexp(frac)
	exp += e
	if exp > 1024 {
		return Copysign(Inf(1), frac)
	}
	if exp < -1100 {
		return Copysign(0, frac)
	}
	if exp >= -1021 {
		if exp == 1024 {
			return m * 2 * pow2(1023)
		}
		return m * pow2(exp)
	}
	return m * pow2(-1021) * pow2(exp+1021)
}

func Max(x, y float64) float64 {
	switch {
	case IsInf(x, 1) || IsInf(y, 1):
		return Inf(1)
	case IsNaN(x) || IsNaN(y):
		return NaN()
	case x == 0 && x == y:
		if Signbit(x) {
			return y
		}
		return x
	}
	if x > y {
		return x
	}
	return y
}

func Min(x, y float64) float64 {
	switch {
	case IsInf(x, -1) || IsInf(y, -1):
		return Inf(-1)
	case IsNaN(x) || IsNaN(y):
		return NaN()
	case x == 0 && x == y:
		if Signbit(x) {
			return x
		}
		return y
	}
	if x < y {
		return x
	}
	return y
}

func Dim(x, y float64) float64 {
	v := x - y
	if v <= 0 {
		return 0
	}
	return v
}

// Mod is x - n*y for the n that truncates x/y, which every step below subtracts
// exactly; the result has the sign of x.
func Mod(x, y float64) float64 {
	if y == 0 || IsInf(x, 0) || IsNaN(x) || IsNaN(y) {
		return NaN()
	}
	y = Abs(y)
	yfr, yexp := Frexp(y)
	r := x
	if x < 0 {
		r = -x
	}
	for r >= y {
		rfr, rexp := Frexp(r)
		if rfr < yfr {
			rexp = rexp - 1
		}
		r = r - Ldexp(y, rexp-yexp)
	}
	if x < 0 {
		r = -r
	}
	return r
}

func isOddInt(x float64) bool {
	if Abs(x) >= 9007199254740992.0 {
		return false
	}
	xi, xf := Modf(x)
	return xf == 0 && int64(xi)&1 == 1
}

// Pow: Go's special cases, then x^yi by successive squaring of Frexp(x)'s fraction
// with the powers of two kept apart, and one Ldexp at the end.
func Pow(x, y float64) float64 {
	switch {
	case y == 0 || x == 1:
		return 1
	case y == 1:
		return x
	case IsNaN(x) || IsNaN(y):
		return NaN()
	case x == 0:
		switch {
		case y < 0:
			if Signbit(x) && isOddInt(y) {
				return Inf(-1)
			}
			return Inf(1)
		case y > 0:
			if Signbit(x) && isOddInt(y) {
				return x
			}
			return 0
		}
	case IsInf(y, 0):
		switch {
		case x == -1:
			return 1
		case (Abs(x) < 1) == IsInf(y, 1):
			return 0
		default:
			return Inf(1)
		}
	case IsInf(x, 0):
		if IsInf(x, -1) {
			return Pow(1/x, -y)
		}
		switch {
		case y < 0:
			return 0
		case y > 0:
			return Inf(1)
		}
	case y == 0.5:
		return Sqrt(x)
	case y == -0.5:
		return 1 / Sqrt(x)
	}
	yi, yf := Modf(Abs(y))
	if yf != 0 && x < 0 {
		return NaN()
	}
	if yi >= 9223372036854775808.0 {
		switch {
		case x == -1:
			return 1
		case (Abs(x) < 1) == (y > 0):
			return 0
		default:
			return Inf(1)
		}
	}
	if yf != 0 {
		panic("math.Pow: a non-integral exponent other than ±0.5 is not supported")
	}
	a1 := 1.0
	ae := 0
	x1, xe := Frexp(x)
	for i := int64(yi); i != 0; i = i >> 1 {
		if xe < -4096 || 4096 < xe {
			ae += xe
			break
		}
		if i&1 == 1 {
			a1 *= x1
			ae += xe
		}
		x1 *= x1
		xe <<= 1
		if x1 < 0.5 {
			x1 += x1
			xe--
		}
	}
	if y < 0 {
		a1 = 1 / a1
		ae = -ae
	}
	return Ldexp(a1, ae)
}
