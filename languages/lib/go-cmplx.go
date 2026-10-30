// Package cmplx: Go's math/cmplx package WRITTEN IN GO, compiled with a program
// that imports "math/cmplx" (core.importFile in both go grammars).
//
// The digits a program prints are the answer, so each function performs the
// sequence of float64 operations of Go's own math/cmplx (Copyright 2010 The Go
// Authors, BSD-style license), with math.FMA where Go's arm64 build fuses a
// multiply and an add. Not provided: the complex trigonometric and hyperbolic
// functions (Sin, Cos, Tan, Asin, Sinh, ...), whose real counterparts the math
// package here does not have.
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
