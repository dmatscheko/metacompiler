// Package math: Go's math package WRITTEN IN GO, compiled with a program that imports
// "math" (core.importFile in both go grammars), so one implementation serves the
// interpreter, llvm.Run and a native build alike.
//
// Every function here is EXACT - bit for bit Go's answer on arm64 - because it
// performs Go's own sequence of IEEE operations, each rounded once: Sqrt is the
// host's correctly rounded square root (goMathSqrt, bound by both grammars), FMA
// rounds x*y + z once, and the transcendental functions call FMA exactly where
// Go's arm64 build fuses a multiply and an add.
//
// A NaN's sign and payload are not modelled: Signbit(NaN) is false and
// Float64bits of any NaN is NaN()'s pattern.
package math

// Go's own constant texts: a constant is exact until it lands in a typed place,
// so math.Pi == 3.141592653589793 is false.
const (
	E   = 2.71828182845904523536028747135266249775724709369995957496696763
	Pi  = 3.14159265358979323846264338327950288419716939937510582097494459
	Phi = 1.61803398874989484820458683436563811772030917980576286213544862

	Sqrt2   = 1.41421356237309504880168872420969807856967187537694807317667974
	SqrtE   = 1.64872127070012814684865078781416357165377610071014801157507931
	SqrtPi  = 1.77245385090551602729816748334114518279754945612238712821380779
	SqrtPhi = 1.27201964951406896425242246173749149171560804184009624861664038

	Ln2    = 0.693147180559945309417232121458176568075500134360255254120680009
	Log2E  = 1 / Ln2
	Ln10   = 2.30258509299404568401799145468436420760110148862877297603332790
	Log10E = 1 / Ln10

	MaxFloat32             = 0x1p127 * (1 + (1 - 0x1p-23))
	SmallestNonzeroFloat32 = 0x1p-126 * 0x1p-23
	MaxFloat64             = 0x1p1023 * (1 + (1 - 0x1p-52))
	SmallestNonzeroFloat64 = 0x1p-1022 * 0x1p-52

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
	MaxUint   = 1<<64 - 1
	MaxUint64 = 1<<64 - 1
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

// pow2 is 2^n for -1074 <= n <= 1023, a product of the powers 2^(±2^i) the bits of
// |n| select. Each partial product is a power of two no smaller than 2^n, so
// none of them rounds.
func pow2(n int) float64 {
	p, b := 1.0, 2.0
	if n < 0 {
		n, b = -n, 0.5
	}
	for n > 0 {
		if n&1 == 1 {
			p *= b
		}
		n >>= 1
		if n > 0 {
			b *= b
		}
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
	for a >= 0x1p64 {
		a *= 0x1p-64
		exp += 64
	}
	for a < 0x1p-63 {
		a *= 0x1p64
		exp -= 64
	}
	// From [2^-63, 2^64) into [1, 2) by halving the distance: a step of 2^k down
	// leaves a below 2^k, a step up leaves it at 2^(1-k) or above.
	if a >= 0x1p32 {
		a *= 0x1p-32
		exp += 32
	}
	if a >= 0x1p16 {
		a *= 0x1p-16
		exp += 16
	}
	if a >= 0x1p8 {
		a *= 0x1p-8
		exp += 8
	}
	if a >= 0x1p4 {
		a *= 0x1p-4
		exp += 4
	}
	if a >= 0x1p2 {
		a *= 0x1p-2
		exp += 2
	}
	if a >= 2 {
		a *= 0.5
		exp++
	}
	if a < 0x1p-31 {
		a *= 0x1p32
		exp -= 32
	}
	if a < 0x1p-15 {
		a *= 0x1p16
		exp -= 16
	}
	if a < 0x1p-7 {
		a *= 0x1p8
		exp -= 8
	}
	if a < 0x1p-3 {
		a *= 0x1p4
		exp -= 4
	}
	if a < 0.5 {
		a *= 4
		exp -= 2
	}
	if a < 1 {
		a *= 2
		exp--
	}
	return Copysign(a*0.5, f), exp + 1
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

// Pow: Go's special cases, then x^yf through Exp and Log for the fraction yf of y
// and x^yi by successive squaring of Frexp(x)'s fraction with the powers of two
// kept apart, and one Ldexp at the end.
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
	a1 := 1.0
	ae := 0
	if yf != 0 {
		if yf > 0.5 {
			yf--
			yi++
		}
		a1 = Exp(yf * Log(x))
	}
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

// ----- bits -----

// Float64bits builds the IEEE 754 pattern from Frexp's exact split; scaling by a
// power of two whose product is representable does not round. A NaN's sign and
// payload are not modelled: every NaN answers NaN()'s own pattern.
func Float64bits(f float64) uint64 {
	if f != f {
		return 0x7FF8000000000001
	}
	sign := uint64(0)
	if Signbit(f) {
		sign = uint64(1) << 63
	}
	a := Abs(f)
	switch {
	case a == 0:
		return sign
	case a > MaxFloat64:
		return sign | 0x7FF0000000000000
	case a < 2.2250738585072014e-308:
		return sign | uint64(a*0x1p1000*0x1p74)
	}
	frac, exp := Frexp(a)
	return sign | uint64(exp+1022)<<52 | uint64(frac*0x1p53)&(1<<52-1)
}

func Float64frombits(b uint64) float64 {
	e := int(b >> 52 & 0x7FF)
	m := b & (1<<52 - 1)
	var f float64
	switch {
	case e == 0x7FF && m != 0:
		return NaN()
	case e == 0x7FF:
		f = Inf(1)
	case e == 0:
		f = float64(m) * 0x1p-1074
	default:
		f = float64(m|1<<52) * pow2(e-1075)
	}
	if b>>63 == 1 {
		return -f
	}
	return f
}

// ----- FMA -----

// FMA is x*y + z rounded once. The product's rounding error is exact by Dekker's
// split, so x*y + z is the exact sum th + tl + e of three doubles; rounding
// tl + e first can only change the result where th + u lands exactly halfway
// between two doubles, and there the sign of what that rounding dropped decides.
// The split needs its operands and the product away from overflow and the
// product's error clear of the subnormals; outside that, fmaBits does the
// arithmetic on the bits, as Go's own FMA does.
func FMA(x, y, z float64) float64 {
	// v-v is 0 exactly when v is finite.
	if x == 0 || y == 0 || x-x != 0 || y-y != 0 {
		return x*y + z
	}
	if z == 0 {
		return x * y
	}
	if z-z != 0 {
		return z
	}
	p := x * y
	ap, ax, ay, az := p, x, y, z
	if ap < 0 {
		ap = -ap
	}
	if ax < 0 {
		ax = -ax
	}
	if ay < 0 {
		ay = -ay
	}
	if az < 0 {
		az = -az
	}
	if ap >= 0x1p1020 || ap < 0x1p-960 || ax >= 0x1p995 || ay >= 0x1p995 || az >= 0x1p1020 {
		return fmaBits(x, y, z)
	}
	// e is x*y - p exactly, from Veltkamp's split of x and y into 26-bit halves.
	c := 134217729 * x
	xh := c - (c - x)
	xl := x - xh
	c = 134217729 * y
	yh := c - (c - y)
	yl := y - yh
	e := ((xh*yh - p) + xh*yl + xl*yh) + xl*yl
	// th + tl == z + p, u + ue == tl + e and r + re == th + u, each exactly.
	th := z + p
	bb := th - z
	tl := (z - (th - bb)) + (p - bb)
	u := tl + e
	bb = u - tl
	ue := (tl - (u - bb)) + (e - bb)
	r := th + u
	bb = r - th
	re := (th - (r - bb)) + (u - bb)
	if ue != 0 && re != 0 && (ue < 0) == (re < 0) {
		if n := r + 2*re; n-r == 2*re {
			return n
		}
	}
	return r
}

// fmaBits is Go's FMA on the IEEE patterns: a 106-bit product, the aligned
// addend with every bit shifted out kept as a sticky bit, and one rounding.
func fmaBits(x, y, z float64) float64 {
	xs, xe, xm := fmaSplit(Float64bits(x))
	ys, ye, ym := fmaSplit(Float64bits(y))
	zs, ze, zm := fmaSplit(Float64bits(z))
	pe := xe + ye - 1023 + 1
	pm1, pm2 := mul64(xm<<10, ym<<11)
	zm1, zm2 := zm<<10, uint64(0)
	ps := xs ^ ys
	is62zero := uint((^pm1 >> 62) & 1)
	pm1, pm2 = shl128(pm1, pm2, is62zero)
	pe -= int(is62zero)
	if pe < ze || pe == ze && pm1 < zm1 {
		ps, pe, pm1, pm2, zs, ze, zm1, zm2 = zs, ze, zm1, zm2, ps, pe, pm1, pm2
	}
	if ps != zs && pe == ze && pm1 == zm1 && pm2 == zm2 {
		return 0
	}
	zm1, zm2 = shrSticky(zm1, zm2, uint(pe-ze))
	var m, c uint64
	if ps == zs {
		pm2, c = add64(pm2, zm2, 0)
		pm1, _ = add64(pm1, zm1, c)
		pe -= int(^pm1 >> 63)
		pm1, m = shrSticky(pm1, pm2, uint(64+pm1>>63))
	} else {
		pm2, c = sub64(pm2, zm2, 0)
		pm1, _ = sub64(pm1, zm1, c)
		nz := lz64(pm1)
		if nz == 64 {
			nz += lz64(pm2)
		}
		pe -= nz
		m, pm2 = shl128(pm1, pm2, uint(nz-1))
		m |= nonzero64(pm2)
	}
	if pe > 1022+1023 || pe == 1022+1023 && (m+1<<9)>>63 == 1 {
		return Float64frombits(ps<<63 | 0x7FF0000000000000)
	}
	if pe < 0 {
		n := uint(-pe)
		if n >= 64 {
			m = nonzero64(m)
		} else {
			m = m>>n | nonzero64(m&(1<<n-1))
		}
		pe = 0
	}
	m = ((m + 1<<9) >> 10) & ^zero64((m&(1<<10-1))^1<<9)
	if m == 0 {
		pe = 0
	}
	return Float64frombits(ps<<63 + uint64(pe)<<52 + m)
}

// fmaSplit is a pattern's sign, biased exponent and mantissa, the implicit bit
// added and a subnormal normalized.
func fmaSplit(b uint64) (uint64, int, uint64) {
	e := int(b>>52) & 0x7FF
	m := b & (1<<52 - 1)
	if e == 0 {
		sh := uint(lz64(m) - 11)
		return b >> 63, 1 - int(sh), m << sh
	}
	return b >> 63, e, m | 1<<52
}

func zero64(x uint64) uint64 {
	if x == 0 {
		return 1
	}
	return 0
}

func nonzero64(x uint64) uint64 {
	if x != 0 {
		return 1
	}
	return 0
}

// shl128 and shr128 shift the 128-bit value u1:u2 by n < 128.
func shl128(u1, u2 uint64, n uint) (uint64, uint64) {
	switch {
	case n == 0:
		return u1, u2
	case n >= 64:
		return u2 << (n - 64), 0
	}
	return u1<<n | u2>>(64-n), u2 << n
}

func shr128(u1, u2 uint64, n uint) (uint64, uint64) {
	switch {
	case n == 0:
		return u1, u2
	case n >= 64:
		return 0, u1 >> (n - 64)
	}
	return u1 >> n, u2>>n | u1<<(64-n)
}

// shrSticky is u1:u2 >> n with the bits shifted out ORed into the lowest bit.
func shrSticky(u1, u2 uint64, n uint) (uint64, uint64) {
	switch {
	case n == 0:
		return u1, u2
	case n == 64:
		return 0, u1 | nonzero64(u2)
	case n >= 128:
		return 0, nonzero64(u1 | u2)
	case n < 64:
		r1, r2 := shr128(u1, u2, n)
		return r1, r2 | nonzero64(u2&(1<<n-1))
	}
	r1, r2 := shr128(u1, u2, n)
	return r1, r2 | nonzero64(u1&(1<<(n-64)-1)|u2)
}

// mul64, add64, sub64 and lz64 are math/bits' Mul64, Add64, Sub64 and
// LeadingZeros64.
func mul64(x, y uint64) (uint64, uint64) {
	x0, x1 := x&(1<<32-1), x>>32
	y0, y1 := y&(1<<32-1), y>>32
	w0 := x0 * y0
	t := x1*y0 + w0>>32
	w1, w2 := t&(1<<32-1), t>>32
	w1 += x0 * y1
	return x1*y1 + w2 + w1>>32, x * y
}

func add64(x, y, carry uint64) (uint64, uint64) {
	s := x + y + carry
	return s, ((x & y) | ((x | y) &^ s)) >> 63
}

func sub64(x, y, borrow uint64) (uint64, uint64) {
	d := x - y - borrow
	return d, ((^x & y) | (^(x ^ y) & d)) >> 63
}

func lz64(x uint64) int {
	if x == 0 {
		return 64
	}
	n := 0
	for x>>63 == 0 {
		x <<= 1
		n++
	}
	return n
}

// ----- the transcendental functions -----
//
// Each is Go's algorithm with Go's constants, and with an FMA wherever Go's arm64
// build computes one: there the compiler contracts a*b + c into a single
// rounding (Exp is the arm64 assembly, which spells its FMAs out). Every
// operation is otherwise rounded on its own, so all three engines compute the
// same double as `go run` on arm64.

func Exp(x float64) float64 {
	const (
		Ln2Hi     = 6.93147180369123816490e-01
		Ln2Lo     = 1.90821492927058770002e-10
		Log2e     = 1.44269504088896338700e+00
		Overflow  = 7.09782712893383973096e+02
		Underflow = -7.45133219101941108420e+02
		NearZero  = 3.725290298461914e-09
		P1        = 1.66666666666666657415e-01
		P2        = -2.77777777770155933842e-03
		P3        = 6.61375632143793436117e-05
		P4        = -1.65339022054652515390e-06
		P5        = 4.13813679705723846039e-08
	)
	switch {
	case x != x:
		return x
	case x > Overflow:
		return Inf(1)
	case x < Underflow:
		return 0
	case Abs(x) < NearZero:
		return 1 + x
	}
	kx := FMA(Log2e, x, 0.5)
	if x < 0 {
		kx = FMA(Log2e, x, -0.5)
	}
	k := int(kx)
	kf := float64(k)
	hi := FMA(-kf, Ln2Hi, x)
	lo := kf * Ln2Lo
	r := hi - lo
	t := r * r
	c := FMA(-t, FMA(t, FMA(t, FMA(t, FMA(t, P5, P4), P3), P2), P1), r)
	y := 1 - ((lo - (r*c)/(2-c)) - hi)
	// Ldexp(y, k) on the bits, as the assembly does it.
	b := Float64bits(y)
	e := int(b>>52) + k
	scale := 1.0
	if e < 1 {
		e += 52
		scale = 2.220446049250313e-16
	}
	return Float64frombits(uint64(e)<<52|b&(1<<52-1)) * scale
}

func Log(x float64) float64 {
	const (
		Ln2Hi = 6.93147180369123816490e-01
		Ln2Lo = 1.90821492927058770002e-10
		L1    = 6.666666666666735130e-01
		L2    = 3.999999999940941908e-01
		L3    = 2.857142874366239149e-01
		L4    = 2.222219843214978396e-01
		L5    = 1.818357216161805012e-01
		L6    = 1.531383769920937332e-01
		L7    = 1.479819860511658591e-01
	)
	switch {
	case x != x || IsInf(x, 1):
		return x
	case x < 0:
		return NaN()
	case x == 0:
		return Inf(-1)
	}
	f1, ki := Frexp(x)
	if f1 < 0.7071067811865476 {
		f1 *= 2
		ki--
	}
	f := f1 - 1
	k := float64(ki)
	s := f / (2 + f)
	s2 := s * s
	s4 := s2 * s2
	t2 := s4 * FMA(s4, FMA(s4, L6, L4), L2)
	R := FMA(s2, FMA(s4, FMA(s4, FMA(s4, L7, L5), L3), L1), t2)
	hf := 0.5 * f
	inner := FMA(s, FMA(hf, f, R), k*Ln2Lo)
	return FMA(k, Ln2Hi, -(FMA(hf, f, -inner) - f))
}

// The sin and cos coefficients, highest power first.
var sinC = []float64{1.58962301576546568060e-10, -2.50507477628578072866e-8, 2.75573136213857245213e-6,
	-1.98412698295895385996e-4, 8.33333333332211858878e-3, -1.66666666666666307295e-1}
var cosC = []float64{-1.13585365213876817300e-11, 2.08757008419747316778e-9, -2.75573141792967388112e-7,
	2.48015872888517045348e-5, -1.38888888888730564116e-3, 4.16666666666665929218e-2}

func trigPoly(c []float64, zz float64) float64 {
	p := FMA(zz, c[0], c[1])
	for i := 2; i < 6; i++ {
		p = FMA(zz, p, c[i])
	}
	return p
}

// sinPoly and cosPoly are sin and cos of z in [-Pi/4, Pi/4], zz = z*z.
func sinPoly(z, zz float64) float64 { return FMA(z*zz, trigPoly(sinC, zz), z) }
func cosPoly(zz float64) float64    { return FMA(zz*zz, trigPoly(cosC, zz), FMA(-zz, 0.5, 1)) }

// trigOctant reduces x >= 0 to its octant j and the remainder z of x / (Pi/4).
func trigOctant(x float64, mask bool) (uint64, float64) {
	const (
		PI4A = 7.85398125648498535156e-1
		PI4B = 3.77489470793079817668e-8
		PI4C = 2.69515142907905952645e-15
	)
	if x >= 536870912 {
		return trigReduce(x)
	}
	j := uint64(x * 1.2732395447351628)
	y := float64(j)
	if j&1 == 1 {
		j++
		y++
	}
	if mask {
		j &= 7
	}
	return j, FMA(-y, PI4C, FMA(-y, PI4B, FMA(-y, PI4A, x)))
}

func Sin(x float64) float64 {
	switch {
	case x == 0 || x != x:
		return x
	case IsInf(x, 0):
		return NaN()
	}
	sign := false
	if x < 0 {
		x = -x
		sign = true
	}
	j, z := trigOctant(x, true)
	if j > 3 {
		sign = !sign
		j -= 4
	}
	zz := z * z
	var y float64
	if j == 1 || j == 2 {
		y = cosPoly(zz)
	} else {
		y = sinPoly(z, zz)
	}
	if sign {
		y = -y
	}
	return y
}

func Cos(x float64) float64 {
	if x != x || IsInf(x, 0) {
		return NaN()
	}
	sign := false
	j, z := trigOctant(Abs(x), true)
	if j > 3 {
		j -= 4
		sign = !sign
	}
	if j > 1 {
		sign = !sign
	}
	zz := z * z
	var y float64
	if j == 1 || j == 2 {
		y = sinPoly(z, zz)
	} else {
		y = cosPoly(zz)
	}
	if sign {
		y = -y
	}
	return y
}

func Tan(x float64) float64 {
	const (
		P0 = -1.30936939181383777646e4
		P1 = 1.15351664838587416140e6
		P2 = -1.79565251976484877988e7
		Q1 = 1.36812963470692954678e4
		Q2 = -1.32089234440210967447e6
		Q3 = 2.50083801823357915839e7
		Q4 = -5.38695755929454629881e7
	)
	switch {
	case x == 0 || x != x:
		return x
	case IsInf(x, 0):
		return NaN()
	}
	sign := false
	if x < 0 {
		x = -x
		sign = true
	}
	j, z := trigOctant(x, false)
	zz := z * z
	y := z
	if zz > 1e-14 {
		num := zz * FMA(zz, FMA(zz, P0, P1), P2)
		den := FMA(zz, FMA(zz, FMA(zz, FMA(z, z, Q1), Q2), Q3), Q4)
		y = FMA(num/den, z, z)
	}
	if j&2 == 2 {
		y = -1 / y
	}
	if sign {
		y = -y
	}
	return y
}

// mPi4 is the binary digits of 4/Pi, 64 at a time, for trigReduce.
var mPi4 = []uint64{
	0x0000000000000001, 0x45f306dc9c882a53, 0xf84eafa3ea69bb81, 0xb6c52b3278872083,
	0xfca2c757bd778ac3, 0x6e48dc74849ba5c0, 0x0c925dd413a32439, 0xfc3bd63962534e7d,
	0xd1046bea5d768909, 0xd338e04d68befc82, 0x7323ac7306a673e9, 0x3908bf177bf25076,
	0x3ff12fffbc0b301f, 0xde5e2316b414da3e, 0xda6cfd9e4f96136e, 0x9e8c7ecd3cbfd45a,
	0xea4f758fd7cbe2f6, 0x7a0e73ef14a525d4, 0xd7f6bf623f1aba10, 0xac06608df8f6d757,
}

// trigReduce is Go's Payne-Hanek reduction of x >= 2^29 by Pi/4: the octant j
// and the remainder z, from the product of x's mantissa with 192 bits of 4/Pi.
func trigReduce(x float64) (uint64, float64) {
	ix := Float64bits(x)
	exp := int(ix>>52&0x7FF) - 1023 - 52
	ix = ix&(1<<52-1) | 1<<52
	digit, bitshift := uint(exp+61)/64, uint(exp+61)%64
	z0 := mPi4[digit]<<bitshift | shrOr(mPi4[digit+1], bitshift)
	z1 := mPi4[digit+1]<<bitshift | shrOr(mPi4[digit+2], bitshift)
	z2 := mPi4[digit+2]<<bitshift | shrOr(mPi4[digit+3], bitshift)
	z2hi, _ := mul64(z2, ix)
	z1hi, z1lo := mul64(z1, ix)
	z0lo := z0 * ix
	lo, c := add64(z1lo, z2hi, 0)
	hi, _ := add64(z0lo, z1hi, c)
	j := hi >> 61
	hi = hi<<3 | lo>>61
	lz := uint(lz64(hi))
	e := uint64(1023 - (lz + 1))
	hi = (hi<<(lz+1) | shrOr(lo, lz+1)) >> 12
	hi |= e << 52
	z := Float64frombits(hi)
	if j&1 == 1 {
		j++
		j &= 7
		z--
	}
	return j, z * 0.7853981633974483
}

// shrOr is v >> (64 - n), which Go defines as 0 for n == 0.
func shrOr(v uint64, n uint) uint64 {
	if n == 0 {
		return 0
	}
	return v >> (64 - n)
}

func xatan(x float64) float64 {
	const (
		P0 = -8.750608600031904122785e-01
		P1 = -1.615753718733365076637e+01
		P2 = -7.500855792314704667340e+01
		P3 = -1.228866684490136173410e+02
		P4 = -6.485021904942025371773e+01
		Q0 = +2.485846490142306297962e+01
		Q1 = +1.650270098316988542046e+02
		Q2 = +4.328810604912902668951e+02
		Q3 = +4.853903996359136964868e+02
		Q4 = +1.945506571482613964425e+02
	)
	z := x * x
	num := z * FMA(z, FMA(z, FMA(z, FMA(z, P0, P1), P2), P3), P4)
	den := FMA(z, FMA(z, FMA(z, FMA(z, FMA(x, x, Q0), Q1), Q2), Q3), Q4)
	return FMA(x, num/den, x)
}

func satan(x float64) float64 {
	const Morebits = 6.123233995736765886130e-17
	if x <= 0.66 {
		return xatan(x)
	}
	if x > 2.41421356237309504880 {
		return 1.5707963267948966 - xatan(1/x) + Morebits
	}
	return 0.7853981633974483 + xatan((x-1)/(x+1)) + 3.061616997868383e-17
}

func Atan(x float64) float64 {
	if x == 0 {
		return x
	}
	if x > 0 {
		return satan(x)
	}
	return -satan(-x)
}

func Hypot(p, q float64) float64 {
	p, q = Abs(p), Abs(q)
	switch {
	case IsInf(p, 1) || IsInf(q, 1):
		return Inf(1)
	case p != p || q != q:
		return NaN()
	}
	if p < q {
		p, q = q, p
	}
	if p == 0 {
		return 0
	}
	q = q / p
	return p * Sqrt(FMA(q, q, 1))
}

func Cbrt(x float64) float64 {
	const (
		B1 = 715094163
		B2 = 696219795
		C  = 5.42857142857142815906e-01
		D  = -7.05306122448979611050e-01
		E  = 1.41428571428571436819e+00
		F  = 1.60714285714285720630e+00
		G  = 3.57142857142857150787e-01
	)
	if x == 0 || x != x || IsInf(x, 0) {
		return x
	}
	sign := false
	if x < 0 {
		x = -x
		sign = true
	}
	t := Float64frombits(Float64bits(x)/3 + B1<<32)
	if x < 2.2250738585072014e-308 {
		t = 0x1p54 * x
		t = Float64frombits(Float64bits(t)/3 + B2<<32)
	}
	r := t * t / x
	s := FMA(r, t, C)
	t *= G + F/(s+E+D/s)
	t = Float64frombits(Float64bits(t)&(0xFFFFFFFFC<<28) + 1<<30)
	s = t * t
	r = x / s
	w := t + t
	r = (r - t) / (w + r)
	t = FMA(t, r, t)
	if sign {
		t = -t
	}
	return t
}
