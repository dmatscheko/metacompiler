// Package cmplx: Go's math/cmplx package WRITTEN IN GO, compiled with a program
// that imports "math/cmplx" (core.importFile in both go grammars).
//
// The digits a program prints are the answer, so each function performs the
// sequence of float64 operations of Go's own math/cmplx (Copyright 2010 The Go
// Authors, BSD-style license). The compiler fuses a written `x*y + z` as gc does
// on arm64 (languages/lib/go-constfold.js); math.FMA stands where gc also fuses
// a product held in a variable, which the grammars do not model.
package cmplx

import "math"

func Abs(x complex128) float64 { return math.Hypot(real(x), imag(x)) }

func Phase(x complex128) float64 { return math.Atan2(imag(x), real(x)) }

func Polar(x complex128) (r, theta float64) { return Abs(x), Phase(x) }

func Rect(r, theta float64) complex128 {
	s, c := math.Sincos(theta)
	return complex(r*c, r*s)
}

func Conj(x complex128) complex128 { return complex(real(x), -imag(x)) }

func Inf() complex128 {
	inf := math.Inf(1)
	return complex(inf, inf)
}

func NaN() complex128 {
	nan := math.NaN()
	return complex(nan, nan)
}

// An infinite part makes the value infinite, not NaN, whatever the other part is.
func IsInf(x complex128) bool { return math.IsInf(real(x), 0) || math.IsInf(imag(x), 0) }

func IsNaN(x complex128) bool {
	if IsInf(x) {
		return false
	}
	return math.IsNaN(real(x)) || math.IsNaN(imag(x))
}

// The principal square root, its imaginary part of the sign of x's (a signed zero
// included). The operands are rescaled first so that Hypot neither overflows nor
// underflows.
func Sqrt(x complex128) complex128 {
	if imag(x) == 0 {
		if real(x) == 0 {
			return complex(0, imag(x))
		}
		if real(x) < 0 {
			return complex(0, math.Copysign(math.Sqrt(-real(x)), imag(x)))
		}
		return complex(math.Sqrt(real(x)), imag(x))
	} else if math.IsInf(imag(x), 0) {
		return complex(math.Inf(1), imag(x))
	}
	if real(x) == 0 {
		if imag(x) < 0 {
			r := math.Sqrt(-0.5 * imag(x))
			return complex(r, -r)
		}
		r := math.Sqrt(0.5 * imag(x))
		return complex(r, r)
	}
	a := real(x)
	b := imag(x)
	var scale float64
	if math.Abs(a) > 4 || math.Abs(b) > 4 {
		a *= 0.25
		b *= 0.25
		scale = 2
	} else {
		a *= 1.8014398509481984e16 // 2**54
		b *= 1.8014398509481984e16
		scale = 7.450580596923828125e-9 // 2**-27
	}
	r := math.Hypot(a, b)
	var t float64
	if a > 0 {
		t = math.Sqrt(0.5*r + 0.5*a)
		r = scale * math.Abs((0.5*b)/t)
		t *= scale
	} else {
		r = math.Sqrt(0.5*r - 0.5*a)
		t = scale * math.Abs((0.5*b)/r)
		r *= scale
	}
	if b < 0 {
		return complex(t, -r)
	}
	return complex(t, r)
}

func Exp(x complex128) complex128 {
	re, im := real(x), imag(x)
	switch {
	case math.IsInf(re, 0):
		switch {
		case re > 0 && im == 0:
			return x
		case math.IsInf(im, 0) || math.IsNaN(im):
			if re < 0 {
				return complex(0, math.Copysign(0, im))
			}
			return complex(math.Inf(1), math.NaN())
		}
	case math.IsNaN(re):
		if im == 0 {
			return complex(math.NaN(), im)
		}
	}
	r := math.Exp(re)
	s, c := math.Sincos(im)
	return complex(r*c, r*s)
}

// The principal natural logarithm.
func Log(x complex128) complex128 { return complex(math.Log(Abs(x)), Phase(x)) }

func Log10(x complex128) complex128 {
	z := Log(x)
	return complex(math.Log10E*real(z), math.Log10E*imag(z))
}

// x**y. Pow(0, y) is 1 for a y of zero real part, 0 for a positive one and an
// infinity (Inf(), or +Inf+0i for a real y) for a negative one.
func Pow(x, y complex128) complex128 {
	if x == 0 {
		if IsNaN(y) {
			return NaN()
		}
		r, i := real(y), imag(y)
		switch {
		case r == 0:
			return 1
		case r < 0:
			if i == 0 {
				return complex(math.Inf(1), 0)
			}
			return Inf()
		}
		return 0
	}
	modulus := Abs(x)
	if modulus == 0 {
		return complex(0, 0)
	}
	r := math.Pow(modulus, real(y))
	arg := Phase(x)
	theta := real(y) * arg
	if imag(y) != 0 {
		r *= math.Exp(-imag(y) * arg)
		theta = math.FMA(imag(y), math.Log(modulus), theta)
	}
	s, c := math.Sincos(theta)
	return complex(r*c, r*s)
}

// ----- the trigonometric and hyperbolic functions -----

func Sin(x complex128) complex128 {
	switch re, im := real(x), imag(x); {
	case im == 0 && (math.IsInf(re, 0) || math.IsNaN(re)):
		return complex(math.NaN(), im)
	case math.IsInf(im, 0):
		switch {
		case re == 0:
			return x
		case math.IsInf(re, 0) || math.IsNaN(re):
			return complex(math.NaN(), im)
		}
	case re == 0 && math.IsNaN(im):
		return x
	}
	s, c := math.Sincos(real(x))
	sh, ch := sinhcosh(imag(x))
	return complex(s*ch, c*sh)
}

func Sinh(x complex128) complex128 {
	switch re, im := real(x), imag(x); {
	case re == 0 && (math.IsInf(im, 0) || math.IsNaN(im)):
		return complex(re, math.NaN())
	case math.IsInf(re, 0):
		switch {
		case im == 0:
			return complex(re, im)
		case math.IsInf(im, 0) || math.IsNaN(im):
			return complex(re, math.NaN())
		}
	case im == 0 && math.IsNaN(re):
		return complex(math.NaN(), im)
	}
	s, c := math.Sincos(imag(x))
	sh, ch := sinhcosh(real(x))
	return complex(c*sh, s*ch)
}

func Cos(x complex128) complex128 {
	switch re, im := real(x), imag(x); {
	case im == 0 && (math.IsInf(re, 0) || math.IsNaN(re)):
		return complex(math.NaN(), -im*math.Copysign(0, re))
	case math.IsInf(im, 0):
		switch {
		case re == 0:
			return complex(math.Inf(1), -re*math.Copysign(0, im))
		case math.IsInf(re, 0) || math.IsNaN(re):
			return complex(math.Inf(1), math.NaN())
		}
	case re == 0 && math.IsNaN(im):
		return complex(math.NaN(), 0)
	}
	s, c := math.Sincos(real(x))
	sh, ch := sinhcosh(imag(x))
	return complex(c*ch, -s*sh)
}

func Cosh(x complex128) complex128 {
	switch re, im := real(x), imag(x); {
	case re == 0 && (math.IsInf(im, 0) || math.IsNaN(im)):
		return complex(math.NaN(), re*math.Copysign(0, im))
	case math.IsInf(re, 0):
		switch {
		case im == 0:
			return complex(math.Inf(1), im*math.Copysign(0, re))
		case math.IsInf(im, 0) || math.IsNaN(im):
			return complex(math.Inf(1), math.NaN())
		}
	case im == 0 && math.IsNaN(re):
		return complex(math.NaN(), im)
	}
	s, c := math.Sincos(imag(x))
	sh, ch := sinhcosh(real(x))
	return complex(c*ch, s*sh)
}

func sinhcosh(x float64) (sh, ch float64) {
	if math.Abs(x) <= 0.5 {
		return math.Sinh(x), math.Cosh(x)
	}
	e := math.Exp(x)
	ei := 0.5 / e
	e *= 0.5
	return e - ei, e + ei
}

func Tan(x complex128) complex128 {
	switch re, im := real(x), imag(x); {
	case math.IsInf(im, 0):
		switch {
		case math.IsInf(re, 0) || math.IsNaN(re):
			return complex(math.Copysign(0, re), math.Copysign(1, im))
		}
		return complex(math.Copysign(0, math.Sin(2*re)), math.Copysign(1, im))
	case re == 0 && math.IsNaN(im):
		return x
	}
	d := math.Cos(2*real(x)) + math.Cosh(2*imag(x))
	if math.Abs(d) < 0.25 {
		d = tanSeries(x)
	}
	if d == 0 {
		return Inf()
	}
	return complex(math.Sin(2*real(x))/d, math.Sinh(2*imag(x))/d)
}

func Tanh(x complex128) complex128 {
	switch re, im := real(x), imag(x); {
	case math.IsInf(re, 0):
		switch {
		case math.IsInf(im, 0) || math.IsNaN(im):
			return complex(math.Copysign(1, re), math.Copysign(0, im))
		}
		return complex(math.Copysign(1, re), math.Copysign(0, math.Sin(2*im)))
	case im == 0 && math.IsNaN(re):
		return x
	}
	d := math.Cosh(2*real(x)) + math.Cos(2*imag(x))
	if d == 0 {
		return Inf()
	}
	return complex(math.Sinh(2*real(x))/d, math.Sin(2*imag(x))/d)
}

func Cot(x complex128) complex128 {
	d := math.Cosh(2*imag(x)) - math.Cos(2*real(x))
	if math.Abs(d) < 0.25 {
		d = tanSeries(x)
	}
	if d == 0 {
		return Inf()
	}
	return complex(math.Sin(2*real(x))/d, -math.Sinh(2*imag(x))/d)
}

// x reduced into (-Pi/2, Pi/2] modulo Pi: Cody-Waite below 2^30, and above it
// Payne-Hanek with 1/Pi's binary digits.
func reducePi(x float64) float64 {
	const reduceThreshold float64 = 1 << 30
	if math.Abs(x) < reduceThreshold {
		const (
			PI1 = 3.141592502593994
			PI2 = 1.5099578831723193e-07
			PI3 = 1.0780605716316238e-14
		)
		t := x / math.Pi
		t += 0.5
		t = float64(int64(t))
		return ((x - t*PI1) - t*PI2) - t*PI3
	}
	const (
		mask     = 0x7FF
		shift    = 64 - 11 - 1
		bias     = 1023
		fracMask = 1<<shift - 1
	)
	ix := math.Float64bits(x)
	exp := int(ix>>shift&mask) - bias - shift
	ix &= fracMask
	ix |= 1 << shift
	mPi := []uint64{
		0x0000000000000000,
		0x517cc1b727220a94,
		0xfe13abe8fa9a6ee0,
		0x6db14acc9e21c820,
		0xff28b1d5ef5de2b0,
		0xdb92371d2126e970,
		0x0324977504e8c90e,
		0x7f0ef58e5894d39f,
		0x74411afa975da242,
		0x74ce38135a2fbf20,
		0x9cc8eb1cc1a99cfa,
		0x4e422fc5defc941d,
		0x8ffc4bffef02cc07,
		0xf79788c5ad05368f,
		0xb69b3f6793e584db,
		0xa7a31fb34f2ff516,
		0xba93dd63f5f2f8bd,
		0x9e839cfbc5294975,
		0x35fdafd88fc6ae84,
		0x2b0198237e3db5d5,
	}
	digit, bitshift := uint(exp+64)/64, uint(exp+64)%64
	z0 := (mPi[digit] << bitshift) | (mPi[digit+1] >> (64 - bitshift))
	z1 := (mPi[digit+1] << bitshift) | (mPi[digit+2] >> (64 - bitshift))
	z2 := (mPi[digit+2] << bitshift) | (mPi[digit+3] >> (64 - bitshift))
	z2hi, _ := mul64(z2, ix)
	z1hi, z1lo := mul64(z1, ix)
	z0lo := z0 * ix
	lo, c := add64(z1lo, z2hi, 0)
	hi, _ := add64(z0lo, z1hi, c)
	lz := uint(lz64(hi))
	e := uint64(bias - (lz + 1))
	hi = (hi << (lz + 1)) | (lo >> (64 - (lz + 1)))
	hi >>= 64 - shift
	hi |= e << shift
	x = math.Float64frombits(hi)
	if x > 0.5 {
		x--
	}
	return math.Pi * x
}

// The Taylor series of cosh(2y) - cos(2x), for Tan and Cot where the closed form
// cancels. x2 *= x and y2 *= y are products, so `y2 + x2` adds two: the function's
// only such add fuses the right one, and `y2 - x2` subtracts the right one fused.
func tanSeries(z complex128) float64 {
	const MACHEP = 1.0 / (1 << 53)
	x := math.Abs(2 * real(z))
	y := math.Abs(2 * imag(z))
	x = reducePi(x)
	x = x * x
	y = y * y
	x2 := 1.0
	y2 := 1.0
	f := 1.0
	rn := 0.0
	d := 0.0
	for {
		rn++
		f *= rn
		rn++
		f *= rn
		xo := x2
		x2 *= x
		y2 *= y
		t := math.FMA(xo, x, y2)
		t /= f
		d += t
		rn++
		f *= rn
		rn++
		f *= rn
		xo = x2
		x2 *= x
		y2 *= y
		t = math.FMA(-xo, x, y2)
		t /= f
		d += t
		if !(math.Abs(t/d) > MACHEP) {
			break
		}
	}
	return d
}

// mul64, add64 and lz64 are math/bits' Mul64, Add64 and LeadingZeros64.
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

// ----- the inverse functions -----

func Asin(x complex128) complex128 {
	switch re, im := real(x), imag(x); {
	case im == 0 && math.Abs(re) <= 1:
		return complex(math.Asin(re), im)
	case re == 0 && math.Abs(im) <= 1:
		return complex(re, math.Asinh(im))
	case math.IsNaN(im):
		switch {
		case re == 0:
			return complex(re, math.NaN())
		case math.IsInf(re, 0):
			return complex(math.NaN(), re)
		default:
			return NaN()
		}
	case math.IsInf(im, 0):
		switch {
		case math.IsNaN(re):
			return x
		case math.IsInf(re, 0):
			return complex(math.Copysign(math.Pi/4, re), im)
		default:
			return complex(math.Copysign(0, re), im)
		}
	case math.IsInf(re, 0):
		return complex(math.Copysign(math.Pi/2, re), math.Copysign(re, im))
	}
	ct := complex(-imag(x), real(x))
	xx := x * x
	x1 := complex(1-real(xx), -imag(xx))
	x2 := Sqrt(x1)
	w := Log(ct + x2)
	return complex(imag(w), -real(w))
}

func Asinh(x complex128) complex128 {
	switch re, im := real(x), imag(x); {
	case im == 0 && math.Abs(re) <= 1:
		return complex(math.Asinh(re), im)
	case re == 0 && math.Abs(im) <= 1:
		return complex(re, math.Asin(im))
	case math.IsInf(re, 0):
		switch {
		case math.IsInf(im, 0):
			return complex(re, math.Copysign(math.Pi/4, im))
		case math.IsNaN(im):
			return x
		default:
			return complex(re, math.Copysign(0.0, im))
		}
	case math.IsNaN(re):
		switch {
		case im == 0:
			return x
		case math.IsInf(im, 0):
			return complex(im, re)
		default:
			return NaN()
		}
	case math.IsInf(im, 0):
		return complex(math.Copysign(im, re), math.Copysign(math.Pi/2, im))
	}
	xx := x * x
	x1 := complex(1+real(xx), imag(xx))
	return Log(x + Sqrt(x1))
}

func Acos(x complex128) complex128 {
	w := Asin(x)
	return complex(math.Pi/2-real(w), -imag(w))
}

func Acosh(x complex128) complex128 {
	if x == 0 {
		return complex(0, math.Copysign(math.Pi/2, imag(x)))
	}
	w := Acos(x)
	if imag(w) <= 0 {
		return complex(-imag(w), real(w))
	}
	return complex(imag(w), -real(w))
}

// x2 and t*t are products: gc fuses `1 - x2` as FMA(-re, re, 1), and of the two
// adds `x2 + t*t` - two products, twice in the function - the multiply computed
// first, x2.
func Atan(x complex128) complex128 {
	switch re, im := real(x), imag(x); {
	case im == 0:
		return complex(math.Atan(re), im)
	case re == 0 && math.Abs(im) <= 1:
		return complex(re, math.Atanh(im))
	case math.IsInf(im, 0) || math.IsInf(re, 0):
		if math.IsNaN(re) {
			return complex(math.NaN(), math.Copysign(0, im))
		}
		return complex(math.Copysign(math.Pi/2, re), math.Copysign(0, im))
	case math.IsNaN(re) || math.IsNaN(im):
		return NaN()
	}
	a := math.FMA(-real(x), real(x), 1) - imag(x)*imag(x)
	if a == 0 {
		return NaN()
	}
	t := 0.5 * math.Atan2(2*real(x), a)
	w := reducePi(t)
	t = imag(x) - 1
	b := math.FMA(real(x), real(x), t*t)
	if b == 0 {
		return NaN()
	}
	t = imag(x) + 1
	c := math.FMA(real(x), real(x), t*t) / b
	return complex(w, 0.25*math.Log(c))
}

func Atanh(x complex128) complex128 {
	z := complex(-imag(x), real(x))
	z = Atan(z)
	return complex(imag(z), -real(z))
}
