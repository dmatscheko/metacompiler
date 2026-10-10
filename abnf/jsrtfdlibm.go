package abnf

// java.lang.StrictMath is fdlibm 5.3 (Java's own port of it, FdLibm.java), and
// java.lang.Math answers exactly what StrictMath does for every function here
// except sin and cos on a JIT intrinsic (see jvMathMembers). This is the Go
// twin's copy of the port; languages/lib/fdlibm.js is the one the interpreter
// and layer 2 share, written without bit access. Each function follows the C
// line for line. Every product that feeds an addition is wrapped in float64()
// so that arm64 does not fuse the pair into an FMA, which fdlibm never does.

import "math"

func fdHi(x float64) int32  { return int32(math.Float64bits(x) >> 32) }
func fdLo(x float64) uint32 { return uint32(math.Float64bits(x)) }
func fdWords(hi int32, lo uint32) float64 {
	return math.Float64frombits(uint64(uint32(hi))<<32 | uint64(lo))
}
func fdSetHi(x float64, hi int32) float64  { return fdWords(hi, fdLo(x)) }
func fdSetLo(x float64, lo uint32) float64 { return fdWords(fdHi(x), lo) }

// fdPoly is c[0] + z*(c[1] + z*(... + z*c[n-1])), each product rounded on
// its own.
func fdPoly(z float64, c ...float64) float64 {
	q := c[len(c)-1]
	for i := len(c) - 2; i >= 0; i-- {
		q = c[i] + float64(z*q)
	}
	return q
}

const (
	fdLn2Hi = 6.93147180369123816490e-01
	fdLn2Lo = 1.90821492927058770002e-10
	fdTwo54 = 1.80143985094819840000e+16
	fdHuge  = 1.0e+300
	fdTiny  = 1.0e-300
)

// fdExp is e_exp.c.
func fdExp(x float64) float64 {
	const (
		invln2 = 1.44269504088896338700e+00
		p1     = 1.66666666666666019037e-01
		p2     = -2.77777777770155933842e-03
		p3     = 6.61375632143793436117e-05
		p4     = -1.65339022054652515390e-06
		p5     = 4.13813679705723846039e-08
	)
	if x != x {
		return x + x
	}
	neg := x < 0
	ax := math.Abs(x)
	if ax >= 709.7822265625 {
		switch {
		case math.IsInf(x, 1):
			return x
		case math.IsInf(x, -1):
			return 0
		case x > 7.09782712893383973096e+02:
			return math.Inf(1)
		case x < -7.45133219101941108420e+02:
			return 0
		}
	}
	var hi, lo float64
	k := 0
	if ax >= 0.3465735912322998 {
		if ax < 1.0397205352783203 {
			if neg {
				hi, lo, k = x+fdLn2Hi, -fdLn2Lo, -1
			} else {
				hi, lo, k = x-fdLn2Hi, fdLn2Lo, 1
			}
		} else {
			h := 0.5
			if neg {
				h = -0.5
			}
			k = int(float64(invln2*x) + h)
			t := float64(k)
			hi = x - float64(t*fdLn2Hi)
			lo = float64(t * fdLn2Lo)
		}
		x = hi - lo
	} else if ax < 3.725290298461914e-09 {
		return 1 + x
	}
	t := float64(x * x)
	q := float64(t * p5)
	q = float64(t * (p4 + q))
	q = float64(t * (p3 + q))
	q = float64(t * (p2 + q))
	q = float64(t * (p1 + q))
	c := x - q
	if k == 0 {
		return 1 - (float64(x*c)/(c-2) - x)
	}
	y := 1 - ((lo - float64(x*c)/(2-c)) - hi)
	if k >= -1021 {
		return math.Ldexp(y, k)
	}
	return math.Ldexp(y, k+1000) * 0x1p-1000
}

// fdLog is e_log.c.
func fdLog(x float64) float64 {
	const (
		lg1 = 6.666666666666735130e-01
		lg2 = 3.999999999940941908e-01
		lg3 = 2.857142874366239149e-01
		lg4 = 2.222219843214978396e-01
		lg5 = 1.818357216161805012e-01
		lg6 = 1.531383769920937332e-01
		lg7 = 1.479819860511658591e-01
	)
	hx := fdHi(x)
	lx := fdLo(x)
	k := int32(0)
	if hx < 0x00100000 {
		if (hx&0x7fffffff)|int32(lx) == 0 {
			return math.Inf(-1)
		}
		if hx < 0 {
			return math.NaN()
		}
		k -= 54
		x *= fdTwo54
		hx = fdHi(x)
	}
	if hx >= 0x7ff00000 {
		return x + x
	}
	k += (hx >> 20) - 1023
	hx &= 0x000fffff
	i := (hx + 0x95f64) & 0x100000
	x = fdSetHi(x, hx|(i^0x3ff00000))
	k += i >> 20
	f := x - 1.0
	if (0x000fffff & (2 + hx)) < 3 {
		if f == 0 {
			if k == 0 {
				return 0
			}
			dk := float64(k)
			return float64(dk*fdLn2Hi) + float64(dk*fdLn2Lo)
		}
		r := float64(float64(f*f) * (0.5 - float64(0.33333333333333333*f)))
		if k == 0 {
			return f - r
		}
		dk := float64(k)
		return float64(dk*fdLn2Hi) - ((r - float64(dk*fdLn2Lo)) - f)
	}
	s := f / (2.0 + f)
	dk := float64(k)
	z := float64(s * s)
	i = hx - 0x6147a
	w := float64(z * z)
	j := 0x6b851 - hx
	t1 := float64(w * fdPoly(w, lg2, lg4, lg6))
	t2 := float64(z * fdPoly(w, lg1, lg3, lg5, lg7))
	i |= j
	r := t2 + t1
	if i > 0 {
		hfsq := float64(float64(0.5*f) * f)
		if k == 0 {
			return f - (hfsq - float64(s*(hfsq+r)))
		}
		return float64(dk*fdLn2Hi) - ((hfsq - (float64(s*(hfsq+r)) + float64(dk*fdLn2Lo))) - f)
	}
	if k == 0 {
		return f - float64(s*(f-r))
	}
	return float64(dk*fdLn2Hi) - ((float64(s*(f-r)) - float64(dk*fdLn2Lo)) - f)
}

// fdLog10 is e_log10.c.
func fdLog10(x float64) float64 {
	const (
		ivln10    = 4.34294481903251816668e-01
		log10_2hi = 3.01029995663611771306e-01
		log10_2lo = 3.69423907715893078616e-13
	)
	hx := fdHi(x)
	lx := fdLo(x)
	k := int32(0)
	if hx < 0x00100000 {
		if (hx&0x7fffffff)|int32(lx) == 0 {
			return math.Inf(-1)
		}
		if hx < 0 {
			return math.NaN()
		}
		k -= 54
		x *= fdTwo54
		hx = fdHi(x)
	}
	if hx >= 0x7ff00000 {
		return x + x
	}
	k += (hx >> 20) - 1023
	i := int32(uint32(k) >> 31)
	hx = (hx & 0x000fffff) | ((0x3ff - i) << 20)
	y := float64(k + i)
	x = fdSetHi(x, hx)
	z := float64(y*log10_2lo) + float64(ivln10*fdLog(x))
	return z + float64(y*log10_2hi)
}

// fdLog1p is s_log1p.c.
func fdLog1p(x float64) float64 {
	const (
		lp1 = 6.666666666666735130e-01
		lp2 = 3.999999999940941908e-01
		lp3 = 2.857142874366239149e-01
		lp4 = 2.222219843214978396e-01
		lp5 = 1.818357216161805012e-01
		lp6 = 1.531383769920937332e-01
		lp7 = 1.479819860511658591e-01
	)
	hx := fdHi(x)
	ax := hx & 0x7fffffff
	k := int32(1)
	var f, c float64
	hu := int32(0)
	if hx < 0x3FDA827A {
		if ax >= 0x3ff00000 {
			if x == -1.0 {
				return math.Inf(-1)
			}
			return math.NaN()
		}
		if ax < 0x3e200000 {
			if fdTwo54+x > 0 && ax < 0x3c900000 {
				return x
			}
			return x - float64(float64(x*x)*0.5)
		}
		if hx > 0 || hx <= int32(-1076707645) { // (int)0xbfd2bec3
			k = 0
			f = x
			hu = 1
		}
	}
	if hx >= 0x7ff00000 {
		return x + x
	}
	if k != 0 {
		var u float64
		if hx < 0x43400000 {
			u = 1.0 + x
			hu = fdHi(u)
			k = (hu >> 20) - 1023
			if k > 0 {
				c = 1.0 - (u - x)
			} else {
				c = x - (u - 1.0)
			}
			c /= u
		} else {
			u = x
			hu = fdHi(u)
			k = (hu >> 20) - 1023
			c = 0
		}
		hu &= 0x000fffff
		if hu < 0x6a09e {
			u = fdSetHi(u, hu|0x3ff00000)
		} else {
			k += 1
			u = fdSetHi(u, hu|0x3fe00000)
			hu = (0x00100000 - hu) >> 2
		}
		f = u - 1.0
	}
	hfsq := float64(float64(0.5*f) * f)
	dk := float64(k)
	if hu == 0 {
		if f == 0 {
			if k == 0 {
				return 0
			}
			c += float64(dk * fdLn2Lo)
			return float64(dk*fdLn2Hi) + c
		}
		r := float64(hfsq * (1.0 - float64(0.66666666666666666*f)))
		if k == 0 {
			return f - r
		}
		return float64(dk*fdLn2Hi) - ((r - (float64(dk*fdLn2Lo) + c)) - f)
	}
	s := f / (2.0 + f)
	z := float64(s * s)
	r := float64(z * fdPoly(z, lp1, lp2, lp3, lp4, lp5, lp6, lp7))
	if k == 0 {
		return f - (hfsq - float64(s*(hfsq+r)))
	}
	return float64(dk*fdLn2Hi) - ((hfsq - (float64(s*(hfsq+r)) + (float64(dk*fdLn2Lo) + c))) - f)
}

// fdExpm1 is s_expm1.c.
func fdExpm1(x float64) float64 {
	const (
		oThreshold = 7.09782712893383973096e+02
		invln2     = 1.44269504088896338700e+00
		q1         = -3.33333333333331316428e-02
		q2         = 1.58730158725481460165e-03
		q3         = -7.93650757867487942473e-05
		q4         = 4.00821782732936239552e-06
		q5         = -2.01099218183624371326e-07
	)
	hx := uint32(fdHi(x))
	xsb := hx & 0x80000000
	var y float64
	if xsb == 0 {
		y = x
	} else {
		y = -x
	}
	_ = y
	hx &= 0x7fffffff
	if hx >= 0x4043687A {
		if hx >= 0x40862E42 {
			if hx >= 0x7ff00000 {
				if (hx&0xfffff)|fdLo(x) != 0 {
					return x + x
				}
				if xsb == 0 {
					return x
				}
				return -1.0
			}
			if x > oThreshold {
				return math.Inf(1)
			}
		}
		if xsb != 0 {
			if x+fdTiny < 0.0 {
				return fdTiny - 1
			}
		}
	}
	var hi, lo, c float64
	var k int32
	if hx > 0x3fd62e42 {
		if hx < 0x3FF0A2B2 {
			if xsb == 0 {
				hi, lo, k = x-fdLn2Hi, fdLn2Lo, 1
			} else {
				hi, lo, k = x+fdLn2Hi, -fdLn2Lo, -1
			}
		} else {
			h := 0.5
			if xsb != 0 {
				h = -0.5
			}
			k = int32(float64(invln2*x) + h)
			t := float64(k)
			hi = x - float64(t*fdLn2Hi)
			lo = float64(t * fdLn2Lo)
		}
		x = hi - lo
		c = (hi - x) - lo
	} else if hx < 0x3c900000 {
		t := fdHuge + x
		return x - (t - (fdHuge + x))
	} else {
		k = 0
	}
	hfx := float64(0.5 * x)
	hxs := float64(x * hfx)
	r1 := 1 + float64(hxs*fdPoly(hxs, q1, q2, q3, q4, q5))
	t := 3.0 - float64(r1*hfx)
	e := float64(hxs * ((r1 - t) / (6.0 - float64(x*t))))
	if k == 0 {
		return x - (float64(x*e) - hxs)
	}
	e = float64(x*(e-c)) - c
	e -= hxs
	if k == -1 {
		return float64(0.5*(x-e)) - 0.5
	}
	if k == 1 {
		if x < -0.25 {
			return float64(-2.0 * (e - (x + 0.5)))
		}
		return 1 + float64(2.0*(x-e))
	}
	if k <= -2 || k > 56 {
		y = 1 - (e - x)
		y = fdSetHi(y, fdHi(y)+(k<<20))
		return y - 1
	}
	t = 1
	if k < 20 {
		t = fdSetHi(t, 0x3ff00000-(0x200000>>uint(k)))
		y = t - (e - x)
		y = fdSetHi(y, fdHi(y)+(k<<20))
	} else {
		t = fdSetHi(t, (0x3ff-k)<<20)
		y = x - (e + t)
		y += 1
		y = fdSetHi(y, fdHi(y)+(k<<20))
	}
	return y
}

// fdKSin is k_sin.c.
func fdKSin(x, y float64, iy int) float64 {
	const (
		s1 = -1.66666666666666324348e-01
		s2 = 8.33333333332248946124e-03
		s3 = -1.98412698298579493134e-04
		s4 = 2.75573137070700676789e-06
		s5 = -2.50507602534068634195e-08
		s6 = 1.58969099521155010221e-10
	)
	ix := fdHi(x) & 0x7fffffff
	if ix < 0x3e400000 {
		if int(x) == 0 {
			return x
		}
	}
	z := float64(x * x)
	v := float64(z * x)
	r := fdPoly(z, s2, s3, s4, s5, s6)
	if iy == 0 {
		return x + float64(v*(s1+float64(z*r)))
	}
	return x - ((float64(z*(float64(0.5*y)-float64(v*r))) - y) - float64(v*s1))
}

// fdKCos is k_cos.c.
func fdKCos(x, y float64) float64 {
	const (
		c1 = 4.16666666666666019037e-02
		c2 = -1.38888888888741095749e-03
		c3 = 2.48015872894767294178e-05
		c4 = -2.75573143513906633035e-07
		c5 = 2.08757232129817482790e-09
		c6 = -1.13596475577881948265e-11
	)
	ix := fdHi(x) & 0x7fffffff
	if ix < 0x3e400000 {
		if int(x) == 0 {
			return 1
		}
	}
	z := float64(x * x)
	r := float64(z * fdPoly(z, c1, c2, c3, c4, c5, c6))
	if ix < 0x3FD33333 {
		return 1 - (float64(0.5*z) - (float64(z*r) - float64(x*y)))
	}
	var qx float64
	if ix > 0x3fe90000 {
		qx = 0.28125
	} else {
		qx = fdWords(ix-0x00200000, 0)
	}
	hz := float64(0.5*z) - qx
	a := 1 - qx
	return a - (hz - (float64(z*r) - float64(x*y)))
}

var fdTanT = [...]float64{
	3.33333333333334091986e-01,
	1.33333333333201242699e-01,
	5.39682539762260521377e-02,
	2.18694882948595424599e-02,
	8.86323982359930005737e-03,
	3.59207910759131235356e-03,
	1.45620945432529025516e-03,
	5.88041240820264096874e-04,
	2.46463134818469906812e-04,
	7.81794442939557092300e-05,
	7.14072491382608190305e-05,
	-1.85586374855275456654e-05,
	2.59073051863633712884e-05,
}

// fdKTan is k_tan.c.
func fdKTan(x, y float64, iy int) float64 {
	const (
		pio4   = 7.85398163397448278999e-01
		pio4lo = 3.06161699786838301793e-17
	)
	T := &fdTanT
	hx := fdHi(x)
	ix := hx & 0x7fffffff
	if ix < 0x3e300000 {
		if int(x) == 0 {
			if (ix|int32(fdLo(x)))|int32(iy+1) == 0 {
				return 1 / math.Abs(x)
			}
			if iy == 1 {
				return x
			}
			w := x + y
			z := fdSetLo(w, 0)
			v := y - (z - x)
			a := -1 / w
			t := fdSetLo(a, 0)
			s := 1 + float64(t*z)
			return t + float64(a*(s+float64(t*v)))
		}
	}
	if ix >= 0x3FE59428 {
		if hx < 0 {
			x = -x
			y = -y
		}
		z := pio4 - x
		w := pio4lo - y
		x = z + w
		y = 0.0
	}
	z := float64(x * x)
	w := float64(z * z)
	r := fdPoly(w, T[1], T[3], T[5], T[7], T[9], T[11])
	v := float64(z * fdPoly(w, T[2], T[4], T[6], T[8], T[10], T[12]))
	s := float64(z * x)
	r = y + float64(z*(float64(s*(r+v))+y))
	r += float64(T[0] * s)
	w = x + r
	if ix >= 0x3FE59428 {
		v = float64(iy)
		return float64(float64(1-((hx>>30)&2)) * (v - float64(2.0*(x-(float64(w*w)/(w+v)-r)))))
	}
	if iy == 1 {
		return w
	}
	z = fdSetLo(w, 0)
	v = r - (z - x)
	a := -1.0 / w
	t := fdSetLo(a, 0)
	s = 1.0 + float64(t*z)
	return t + float64(a*(s+float64(t*v)))
}

var fdTwoOverPi = [...]int32{
	0xA2F983, 0x6E4E44, 0x1529FC, 0x2757D1, 0xF534DD, 0xC0DB62,
	0x95993C, 0x439041, 0xFE5163, 0xABDEBB, 0xC561B7, 0x246E3A,
	0x424DD2, 0xE00649, 0x2EEA09, 0xD1921C, 0xFE1DEB, 0x1CB129,
	0xA73EE8, 0x8235F5, 0x2EBB44, 0x84E99C, 0x7026B4, 0x5F7E41,
	0x3991D6, 0x398353, 0x39F49C, 0x845F8B, 0xBDF928, 0x3B1FF8,
	0x97FFDE, 0x05980F, 0xEF2F11, 0x8B5A0A, 0x6D1F6D, 0x367ECF,
	0x27CB09, 0xB74F46, 0x3F669E, 0x5FEA2D, 0x7527BA, 0xC7EBE5,
	0xF17B3D, 0x0739F7, 0x8A5292, 0xEA6BFB, 0x5FB11F, 0x8D5D08,
	0x560330, 0x46FC7B, 0x6BABF0, 0xCFBC20, 0x9AF436, 0x1DA9E3,
	0x91615E, 0xE61B08, 0x659985, 0x5F14A0, 0x68408D, 0xFFD880,
	0x4D7327, 0x310606, 0x1556CA, 0x73A8C9, 0x60E27B, 0xC08C6B,
}

var fdNpio2Hw = [...]int32{
	0x3FF921FB, 0x400921FB, 0x4012D97C, 0x401921FB, 0x401F6A7A, 0x4022D97C,
	0x4025FDBB, 0x402921FB, 0x402C463A, 0x402F6A7A, 0x4031475C, 0x4032D97C,
	0x40346B9C, 0x4035FDBB, 0x40378FDB, 0x403921FB, 0x403AB41B, 0x403C463A,
	0x403DD85A, 0x403F6A7A, 0x40407E4C, 0x4041475C, 0x4042106C, 0x4042D97C,
	0x4043A28C, 0x40446B9C, 0x404534AC, 0x4045FDBB, 0x4046C6CB, 0x40478FDB,
	0x404858EB, 0x404921FB,
}

// fdRemPio2 is e_rem_pio2.c: x - n*pi/2 as y0 + y1, and n.
func fdRemPio2(x float64) (int, float64, float64) {
	const (
		two24   = 1.67772160000000000000e+07
		invpio2 = 6.36619772367581382433e-01
		pio2_1  = 1.57079632673412561417e+00
		pio2_1t = 6.07710050650619224932e-11
		pio2_2  = 6.07710050630396597660e-11
		pio2_2t = 2.02226624879595063154e-21
		pio2_3  = 2.02226624871116645580e-21
		pio2_3t = 8.47842766036889956997e-32
	)
	hx := fdHi(x)
	ix := hx & 0x7fffffff
	if ix <= 0x3fe921fb {
		return 0, x, 0
	}
	if ix < 0x4002d97c {
		if hx > 0 {
			z := x - pio2_1
			if ix != 0x3ff921fb {
				y0 := z - pio2_1t
				return 1, y0, (z - y0) - pio2_1t
			}
			z -= pio2_2
			y0 := z - pio2_2t
			return 1, y0, (z - y0) - pio2_2t
		}
		z := x + pio2_1
		if ix != 0x3ff921fb {
			y0 := z + pio2_1t
			return -1, y0, (z - y0) + pio2_1t
		}
		z += pio2_2
		y0 := z + pio2_2t
		return -1, y0, (z - y0) + pio2_2t
	}
	if ix <= 0x413921fb {
		t := math.Abs(x)
		n := int(float64(t*invpio2) + 0.5)
		fn := float64(n)
		r := t - float64(fn*pio2_1)
		w := float64(fn * pio2_1t)
		var y0 float64
		if n < 32 && ix != fdNpio2Hw[n-1] {
			y0 = r - w
		} else {
			j := ix >> 20
			y0 = r - w
			i := j - ((fdHi(y0) >> 20) & 0x7ff)
			if i > 16 {
				t = r
				w = float64(fn * pio2_2)
				r = t - w
				w = float64(fn*pio2_2t) - ((t - r) - w)
				y0 = r - w
				i = j - ((fdHi(y0) >> 20) & 0x7ff)
				if i > 49 {
					t = r
					w = float64(fn * pio2_3)
					r = t - w
					w = float64(fn*pio2_3t) - ((t - r) - w)
					y0 = r - w
				}
			}
		}
		y1 := (r - y0) - w
		if hx < 0 {
			return -n, -y0, -y1
		}
		return n, y0, y1
	}
	if ix >= 0x7ff00000 {
		return 0, x - x, x - x
	}
	e0 := (ix >> 20) - 1046
	z := fdWords(ix-(e0<<20), fdLo(x))
	var tx [3]float64
	for i := 0; i < 2; i++ {
		tx[i] = float64(int32(z))
		z = float64((z - tx[i]) * two24)
	}
	tx[2] = z
	nx := 3
	for tx[nx-1] == 0 {
		nx--
	}
	n, y0, y1 := fdKRemPio2(tx[:nx], int(e0))
	if hx < 0 {
		return -n, -y0, -y1
	}
	return n, y0, y1
}

var fdPIo2 = [...]float64{
	1.57079625129699707031e+00,
	7.54978941586159635335e-08,
	5.39030252995776476554e-15,
	3.28200341580791294123e-22,
	1.27065575308067607349e-29,
	1.22933308981111328932e-36,
	2.73370053816464559624e-44,
	2.16741683877804819444e-51,
}

// fdKRemPio2 is k_rem_pio2.c at prec 2 (jk 4): the large-argument reduction
// over 24-bit chunks of 2/pi.
func fdKRemPio2(x []float64, e0 int) (int, float64, float64) {
	const (
		two24  = 1.67772160000000000000e+07
		twon24 = 5.96046447753906250000e-08
	)
	var iq [20]int32
	var f, fq, q [20]float64
	jk := 4
	jp := jk
	jx := len(x) - 1
	jv := (e0 - 3) / 24
	if jv < 0 {
		jv = 0
	}
	q0 := e0 - 24*(jv+1)
	j := jv - jx
	m := jx + jk
	for i := 0; i <= m; i, j = i+1, j+1 {
		if j < 0 {
			f[i] = 0
		} else {
			f[i] = float64(fdTwoOverPi[j])
		}
	}
	for i := 0; i <= jk; i++ {
		fw := 0.0
		for j := 0; j <= jx; j++ {
			fw += float64(x[j] * f[jx+i-j])
		}
		q[i] = fw
	}
	jz := jk
	var z float64
	var n, ih int
	for {
		i := 0
		z = q[jz]
		for j := jz; j > 0; j-- {
			fw := float64(int32(float64(twon24 * z)))
			iq[i] = int32(z - float64(two24*fw))
			z = q[j-1] + fw
			i++
		}
		z = math.Ldexp(z, q0)
		z -= float64(8.0 * math.Floor(float64(z*0.125)))
		n = int(z)
		z -= float64(n)
		ih = 0
		if q0 > 0 {
			i := int(iq[jz-1] >> uint(24-q0))
			n += i
			iq[jz-1] -= int32(i << uint(24-q0))
			ih = int(iq[jz-1] >> uint(23-q0))
		} else if q0 == 0 {
			ih = int(iq[jz-1] >> 23)
		} else if z >= 0.5 {
			ih = 2
		}
		if ih > 0 {
			n += 1
			carry := 0
			for i := 0; i < jz; i++ {
				j := iq[i]
				if carry == 0 {
					if j != 0 {
						carry = 1
						iq[i] = 0x1000000 - j
					}
				} else {
					iq[i] = 0xffffff - j
				}
			}
			if q0 > 0 {
				switch q0 {
				case 1:
					iq[jz-1] &= 0x7fffff
				case 2:
					iq[jz-1] &= 0x3fffff
				}
			}
			if ih == 2 {
				z = 1 - z
				if carry != 0 {
					z -= math.Ldexp(1, q0)
				}
			}
		}
		if z == 0 {
			j := int32(0)
			for i := jz - 1; i >= jk; i-- {
				j |= iq[i]
			}
			if j == 0 {
				k := 1
				for iq[jk-k] == 0 {
					k++
				}
				for i := jz + 1; i <= jz+k; i++ {
					f[jx+i] = float64(fdTwoOverPi[jv+i])
					fw := 0.0
					for j := 0; j <= jx; j++ {
						fw += float64(x[j] * f[jx+i-j])
					}
					q[i] = fw
				}
				jz += k
				continue
			}
		}
		break
	}
	if z == 0 {
		jz -= 1
		q0 -= 24
		for iq[jz] == 0 {
			jz--
			q0 -= 24
		}
	} else {
		z = math.Ldexp(z, -q0)
		if z >= two24 {
			fw := float64(int32(float64(twon24 * z)))
			iq[jz] = int32(z - float64(two24*fw))
			jz += 1
			q0 += 24
			iq[jz] = int32(fw)
		} else {
			iq[jz] = int32(z)
		}
	}
	fw := math.Ldexp(1, q0)
	for i := jz; i >= 0; i-- {
		q[i] = float64(fw * float64(iq[i]))
		fw *= twon24
	}
	for i := jz; i >= 0; i-- {
		fw = 0.0
		for k := 0; k <= jp && k <= jz-i; k++ {
			fw += float64(fdPIo2[k] * q[i+k])
		}
		fq[jz-i] = fw
	}
	fw = 0.0
	for i := jz; i >= 0; i-- {
		fw += fq[i]
	}
	var y0, y1 float64
	if ih == 0 {
		y0 = fw
	} else {
		y0 = -fw
	}
	fw = fq[0] - fw
	for i := 1; i <= jz; i++ {
		fw += fq[i]
	}
	if ih == 0 {
		y1 = fw
	} else {
		y1 = -fw
	}
	return n & 7, y0, y1
}

// fdSin, fdCos and fdTan are s_sin.c, s_cos.c and s_tan.c.
func fdSin(x float64) float64 {
	ix := fdHi(x) & 0x7fffffff
	if ix <= 0x3fe921fb {
		return fdKSin(x, 0, 0)
	}
	if ix >= 0x7ff00000 {
		return x - x
	}
	n, y0, y1 := fdRemPio2(x)
	switch n & 3 {
	case 0:
		return fdKSin(y0, y1, 1)
	case 1:
		return fdKCos(y0, y1)
	case 2:
		return -fdKSin(y0, y1, 1)
	}
	return -fdKCos(y0, y1)
}

func fdCos(x float64) float64 {
	ix := fdHi(x) & 0x7fffffff
	if ix <= 0x3fe921fb {
		return fdKCos(x, 0)
	}
	if ix >= 0x7ff00000 {
		return x - x
	}
	n, y0, y1 := fdRemPio2(x)
	switch n & 3 {
	case 0:
		return fdKCos(y0, y1)
	case 1:
		return -fdKSin(y0, y1, 1)
	case 2:
		return -fdKCos(y0, y1)
	}
	return fdKSin(y0, y1, 1)
}

func fdTan(x float64) float64 {
	ix := fdHi(x) & 0x7fffffff
	if ix <= 0x3fe921fb {
		return fdKTan(x, 0, 1)
	}
	if ix >= 0x7ff00000 {
		return x - x
	}
	n, y0, y1 := fdRemPio2(x)
	return fdKTan(y0, y1, 1-((n&1)<<1))
}

const (
	fdPio2Hi = 1.57079632679489655800e+00
	fdPio2Lo = 6.12323399573676603587e-17
	fdPS0    = 1.66666666666666657415e-01
	fdPS1    = -3.25565818622400915405e-01
	fdPS2    = 2.01212532134862925881e-01
	fdPS3    = -4.00555345006794114027e-02
	fdPS4    = 7.91534994289814532176e-04
	fdPS5    = 3.47933107596021167570e-05
	fdQS1    = -2.40339491173441421878e+00
	fdQS2    = 2.02094576023350569471e+00
	fdQS3    = -6.88283971605453293030e-01
	fdQS4    = 7.70381505559019352791e-02
)

// fdAsinPQ is the rational approximation e_asin.c and e_acos.c share.
func fdAsinPQ(t float64) (float64, float64) {
	p := float64(t * fdPoly(t, fdPS0, fdPS1, fdPS2, fdPS3, fdPS4, fdPS5))
	q := 1 + float64(t*fdPoly(t, fdQS1, fdQS2, fdQS3, fdQS4))
	return p, q
}

// fdAsin is e_asin.c.
func fdAsin(x float64) float64 {
	const pio4Hi = 7.85398163397448278999e-01
	hx := fdHi(x)
	ix := hx & 0x7fffffff
	if ix >= 0x3ff00000 {
		if (ix-0x3ff00000)|int32(fdLo(x)) == 0 {
			return float64(x*fdPio2Hi) + float64(x*fdPio2Lo)
		}
		return math.NaN()
	} else if ix < 0x3fe00000 {
		if ix < 0x3e400000 {
			if fdHuge+x > 1 {
				return x
			}
		}
		t := float64(x * x)
		p, q := fdAsinPQ(t)
		w := p / q
		return x + float64(x*w)
	}
	w := 1 - math.Abs(x)
	t := float64(w * 0.5)
	p, q := fdAsinPQ(t)
	s := math.Sqrt(t)
	if ix >= 0x3FEF3333 {
		w = p / q
		t = fdPio2Hi - (float64(2.0*(s+float64(s*w))) - fdPio2Lo)
	} else {
		w = fdSetLo(s, 0)
		c := (t - float64(w*w)) / (s + w)
		r := p / q
		p = float64(float64(2.0*s)*r) - (fdPio2Lo - float64(2.0*c))
		q = pio4Hi - float64(2.0*w)
		t = pio4Hi - (p - q)
	}
	if hx > 0 {
		return t
	}
	return -t
}

// fdAcos is e_acos.c.
func fdAcos(x float64) float64 {
	const pi = 3.14159265358979311600e+00
	hx := fdHi(x)
	ix := hx & 0x7fffffff
	if ix >= 0x3ff00000 {
		if (ix-0x3ff00000)|int32(fdLo(x)) == 0 {
			if hx > 0 {
				return 0.0
			}
			return pi + float64(2.0*fdPio2Lo)
		}
		return math.NaN()
	}
	if ix < 0x3fe00000 {
		if ix <= 0x3c600000 {
			return fdPio2Hi + fdPio2Lo
		}
		z := float64(x * x)
		p, q := fdAsinPQ(z)
		r := p / q
		return fdPio2Hi - (x - (fdPio2Lo - float64(x*r)))
	} else if hx < 0 {
		z := float64((1 + x) * 0.5)
		p, q := fdAsinPQ(z)
		s := math.Sqrt(z)
		r := p / q
		w := float64(r*s) - fdPio2Lo
		return pi - float64(2.0*(s+w))
	}
	z := float64((1 - x) * 0.5)
	s := math.Sqrt(z)
	df := fdSetLo(s, 0)
	c := (z - float64(df*df)) / (s + df)
	p, q := fdAsinPQ(z)
	r := p / q
	w := float64(r*s) + c
	return float64(2.0 * (df + w))
}

var fdAtanHi = [...]float64{
	4.63647609000806093515e-01,
	7.85398163397448278999e-01,
	9.82793723247329054082e-01,
	1.57079632679489655800e+00,
}
var fdAtanLo = [...]float64{
	2.26987774529616870924e-17,
	3.06161699786838301793e-17,
	1.39033110312309984516e-17,
	6.12323399573676603587e-17,
}
var fdAT = [...]float64{
	3.33333333333329318027e-01,
	-1.99999999998764832476e-01,
	1.42857142725034663711e-01,
	-1.11111104054623557880e-01,
	9.09088713343650656196e-02,
	-7.69187620504482999495e-02,
	6.66107313738753120669e-02,
	-5.83357013379057348645e-02,
	4.97687799461593236017e-02,
	-3.65315727442169155270e-02,
	1.62858201153657823623e-02,
}

// fdAtan is s_atan.c.
func fdAtan(x float64) float64 {
	hx := fdHi(x)
	ix := hx & 0x7fffffff
	var id int
	if ix >= 0x44100000 {
		if ix > 0x7ff00000 || (ix == 0x7ff00000 && fdLo(x) != 0) {
			return x + x
		}
		if hx > 0 {
			return fdAtanHi[3] + fdAtanLo[3]
		}
		return -fdAtanHi[3] - fdAtanLo[3]
	}
	if ix < 0x3fdc0000 {
		if ix < 0x3e200000 {
			if fdHuge+x > 1 {
				return x
			}
		}
		id = -1
	} else {
		x = math.Abs(x)
		if ix < 0x3ff30000 {
			if ix < 0x3fe60000 {
				id = 0
				x = (float64(2.0*x) - 1) / (2.0 + x)
			} else {
				id = 1
				x = (x - 1) / (x + 1)
			}
		} else {
			if ix < 0x40038000 {
				id = 2
				x = (x - 1.5) / (1 + float64(1.5*x))
			} else {
				id = 3
				x = -1.0 / x
			}
		}
	}
	a := &fdAT
	z := float64(x * x)
	w := float64(z * z)
	s1 := float64(z * fdPoly(w, a[0], a[2], a[4], a[6], a[8], a[10]))
	s2 := float64(w * fdPoly(w, a[1], a[3], a[5], a[7], a[9]))
	if id < 0 {
		return x - float64(x*(s1+s2))
	}
	z = fdAtanHi[id] - ((float64(x*(s1+s2)) - fdAtanLo[id]) - x)
	if hx < 0 {
		return -z
	}
	return z
}

// fdAtan2 is e_atan2.c.
func fdAtan2(y, x float64) float64 {
	const (
		piO4 = 7.8539816339744827900e-01
		piO2 = 1.5707963267948965580e+00
		pi   = 3.1415926535897931160e+00
		piLo = 1.2246467991473531772e-16
	)
	if x != x || y != y {
		return x + y
	}
	hx := fdHi(x)
	ix := hx & 0x7fffffff
	lx := fdLo(x)
	hy := fdHi(y)
	iy := hy & 0x7fffffff
	ly := fdLo(y)
	if (hx-0x3ff00000)|int32(lx) == 0 {
		return fdAtan(y)
	}
	m := ((hy >> 31) & 1) | ((hx >> 30) & 2)
	if iy|int32(ly) == 0 {
		switch m {
		case 0, 1:
			return y
		case 2:
			return pi + fdTiny
		}
		return -pi - fdTiny
	}
	if ix|int32(lx) == 0 {
		if hy < 0 {
			return -piO2 - fdTiny
		}
		return piO2 + fdTiny
	}
	if ix == 0x7ff00000 {
		if iy == 0x7ff00000 {
			switch m {
			case 0:
				return piO4 + fdTiny
			case 1:
				return -piO4 - fdTiny
			case 2:
				return float64(3.0*piO4) + fdTiny
			}
			return float64(-3.0*piO4) - fdTiny
		}
		switch m {
		case 0:
			return 0
		case 1:
			return math.Copysign(0, -1)
		case 2:
			return pi + fdTiny
		}
		return -pi - fdTiny
	}
	if iy == 0x7ff00000 {
		if hy < 0 {
			return -piO2 - fdTiny
		}
		return piO2 + fdTiny
	}
	k := (iy - ix) >> 20
	var z float64
	if k > 60 {
		z = piO2 + float64(0.5*piLo)
	} else if hx < 0 && k < -60 {
		z = 0.0
	} else {
		z = fdAtan(math.Abs(y / x))
	}
	switch m {
	case 0:
		return z
	case 1:
		return -z
	case 2:
		return pi - (z - piLo)
	}
	return (z - piLo) - pi
}

// fdSinh, fdCosh and fdTanh are e_sinh.c, e_cosh.c and s_tanh.c.
func fdSinh(x float64) float64 {
	const shuge = 1.0e307
	jx := fdHi(x)
	ix := jx & 0x7fffffff
	if ix >= 0x7ff00000 {
		return x + x
	}
	h := 0.5
	if jx < 0 {
		h = -h
	}
	if ix < 0x40360000 {
		if ix < 0x3e300000 {
			if shuge+x > 1 {
				return x
			}
		}
		t := fdExpm1(math.Abs(x))
		if ix < 0x3ff00000 {
			return float64(h * (float64(2.0*t) - float64(t*t)/(t+1)))
		}
		return float64(h * (t + t/(t+1)))
	}
	if ix < 0x40862E42 {
		return float64(h * fdExp(math.Abs(x)))
	}
	lx := fdLo(x)
	if ix < 0x408633CE || (ix == 0x408633ce && lx <= 0x8fb9f87d) {
		w := fdExp(float64(0.5 * math.Abs(x)))
		t := float64(h * w)
		return float64(t * w)
	}
	return x * shuge
}

func fdCosh(x float64) float64 {
	ix := fdHi(x) & 0x7fffffff
	if ix >= 0x7ff00000 {
		return x * x
	}
	if ix < 0x3fd62e43 {
		t := fdExpm1(math.Abs(x))
		w := 1 + t
		if ix < 0x3c800000 {
			return w
		}
		return 1 + float64(t*t)/(w+w)
	}
	if ix < 0x40360000 {
		t := fdExp(math.Abs(x))
		return float64(0.5*t) + 0.5/t
	}
	if ix < 0x40862E42 {
		return float64(0.5 * fdExp(math.Abs(x)))
	}
	lx := fdLo(x)
	if ix < 0x408633CE || (ix == 0x408633ce && lx <= 0x8fb9f87d) {
		w := fdExp(float64(0.5 * math.Abs(x)))
		t := float64(0.5 * w)
		return float64(t * w)
	}
	return math.Inf(1)
}

func fdTanh(x float64) float64 {
	jx := fdHi(x)
	ix := jx & 0x7fffffff
	if ix >= 0x7ff00000 {
		if jx >= 0 {
			return 1/x + 1
		}
		return 1/x - 1
	}
	var z float64
	if ix < 0x40360000 {
		if ix < 0x3c800000 {
			return float64(x * (1 + x))
		}
		if ix >= 0x3ff00000 {
			t := fdExpm1(float64(2.0 * math.Abs(x)))
			z = 1 - 2.0/(t+2.0)
		} else {
			t := fdExpm1(float64(-2.0 * math.Abs(x)))
			z = -t / (t + 2.0)
		}
	} else {
		z = 1 - fdTiny
	}
	if jx >= 0 {
		return z
	}
	return -z
}

// fdCbrt is s_cbrt.c.
func fdCbrt(x float64) float64 {
	const (
		b1 = 715094163
		b2 = 696219795
		C  = 5.42857142857142815906e-01
		D  = -7.05306122448979611050e-01
		E  = 1.41428571428571436819e+00
		F  = 1.60714285714285720630e+00
		G  = 3.57142857142857150787e-01
	)
	if x == 0 || math.IsNaN(x) || math.IsInf(x, 0) {
		return x + x
	}
	sign := 1.0
	if x < 0 {
		sign = -1.0
	}
	x = math.Abs(x)
	t := 0.0
	if x < 0x1p-1022 {
		t = 0x1p54
		t *= x
		t = fdSetHi(t, fdHi(t)/3+b2)
	} else {
		t = fdSetHi(t, fdHi(x)/3+b1)
	}
	r := float64(t*t) / x
	s := C + float64(r*t)
	t *= G + F/(s+E+D/s)
	t = fdSetLo(t, 0)
	t = fdSetHi(t, fdHi(t)+1)
	s = float64(t * t)
	r = x / s
	w := t + t
	r = (r - t) / (w + r)
	t = t + float64(t*r)
	return sign * t
}

// fdHypot is e_hypot.c as FdLibm.Hypot spells it.
func fdHypot(x, y float64) float64 {
	a := math.Abs(x)
	b := math.Abs(y)
	if math.IsInf(a, 0) || math.IsInf(b, 0) {
		return math.Inf(1)
	}
	if a != a || b != b {
		return a + b
	}
	if b > a {
		a, b = b, a
	}
	ha := fdHi(a)
	hb := fdHi(b)
	if ha-hb > 0x3c00000 {
		return a + b
	}
	k := 0
	if a > 0x1.00000ffffffffp500 {
		ha -= 0x25800000
		hb -= 0x25800000
		a = a * 0x1p-600
		b = b * 0x1p-600
		k += 600
	}
	var t1, t2 float64
	if b < 0x1p-500 {
		if b < 0x1p-1022 {
			if b == 0 {
				return a
			}
			t1 = 0x1p1022
			b *= t1
			a *= t1
			k -= 1022
		} else {
			ha += 0x25800000
			hb += 0x25800000
			a = a * 0x1p600
			b = b * 0x1p600
			k -= 600
		}
	}
	w := a - b
	if w > b {
		t1 = fdWords(ha, 0)
		t2 = a - t1
		w = math.Sqrt(float64(t1*t1) - (float64(b*(-b)) - float64(t2*(a+t1))))
	} else {
		a = a + a
		y1 := fdWords(hb, 0)
		y2 := b - y1
		t1 = fdWords(ha+0x00100000, 0)
		t2 = a - t1
		w = math.Sqrt(float64(t1*y1) - (float64(w*(-w)) - (float64(t1*y2) + float64(t2*b))))
	}
	if k != 0 {
		return float64(math.Ldexp(1, k) * w)
	}
	return w
}

// fdPow is e_pow.c as FdLibm.Pow spells it.
func fdPow(x, y float64) float64 {
	if y == 0 {
		return 1
	}
	if x != x || y != y {
		return x + y
	}
	yAbs := math.Abs(y)
	xAbs := math.Abs(x)
	if y == 2.0 {
		return float64(x * x)
	} else if y == 0.5 {
		if x >= -math.MaxFloat64 {
			return math.Sqrt(x + 0.0)
		}
	} else if yAbs == 1.0 {
		if y == 1.0 {
			return x
		}
		return 1.0 / x
	} else if math.IsInf(yAbs, 0) {
		if xAbs == 1.0 {
			return y - y
		} else if xAbs > 1.0 {
			if y >= 0 {
				return y
			}
			return 0.0
		}
		if y < 0 {
			return -y
		}
		return 0.0
	}
	hx := fdHi(x)
	ix := hx & 0x7fffffff
	yIsInt := 0
	if hx < 0 {
		if yAbs >= 0x1p53 {
			yIsInt = 2
		} else if yAbs >= 1.0 {
			yl := int64(yAbs)
			if float64(yl) == yAbs {
				yIsInt = 2 - int(yl&1)
			}
		}
	}
	if xAbs == 0.0 || math.IsInf(xAbs, 0) || xAbs == 1.0 {
		z := xAbs
		if y < 0.0 {
			z = 1.0 / z
		}
		if hx < 0 {
			if (int(ix-0x3ff00000) | yIsInt) == 0 {
				z = math.NaN()
			} else if yIsInt == 1 {
				z = float64(-1.0 * z)
			}
		}
		return z
	}
	n := int((hx >> 31) + 1)
	if (n | yIsInt) == 0 {
		return math.NaN()
	}
	s := 1.0
	if (n | (yIsInt - 1)) == 0 {
		s = -1.0
	}
	var pH, pL, t1, t2 float64
	if yAbs > 0x1.00000ffffffffp31 {
		const (
			invLn2  = 0x1.71547652b82fep0
			invLn2H = 0x1.715476p0
			invLn2L = 0x1.4ae0bf85ddf44p-26
		)
		if xAbs < 0x1.fffff00000000p-1 {
			if y < 0.0 {
				return s * math.Inf(1)
			}
			return float64(s * 0.0)
		}
		if xAbs > 0x1.00000ffffffffp0 {
			if y > 0.0 {
				return s * math.Inf(1)
			}
			return float64(s * 0.0)
		}
		t := xAbs - 1.0
		w := float64(float64(t*t) * (0.5 - float64(t*(0.3333333333333333333333-float64(t*0.25)))))
		u := float64(invLn2H * t)
		v := float64(t*invLn2L) - float64(w*invLn2)
		t1 = u + v
		t1 = fdSetLo(t1, 0)
		t2 = v - (t1 - u)
	} else {
		const (
			cp  = 0x1.ec709dc3a03fdp-1
			cpH = 0x1.ec709ep-1
			cpL = -0x1.e2fe0145b01f5p-28
			l1  = 0x1.3333333333303p-1
			l2  = 0x1.b6db6db6fabffp-2
			l3  = 0x1.55555518f264dp-2
			l4  = 0x1.17460a91d4101p-2
			l5  = 0x1.d864a93c9db65p-3
			l6  = 0x1.a7e284a454eefp-3
		)
		bp := [2]float64{1.0, 1.5}
		dpH := [2]float64{0.0, 0x1.2b8034p-1}
		dpL := [2]float64{0.0, 0x1.cfdeb43cfd006p-27}
		n = 0
		if ix < 0x00100000 {
			xAbs *= 0x1p53
			n -= 53
			ix = fdHi(xAbs)
		}
		n += int((ix >> 20) - 0x3ff)
		j := ix & 0x000fffff
		ix = j | 0x3ff00000
		var k int
		if j <= 0x3988E {
			k = 0
		} else if j < 0xBB67A {
			k = 1
		} else {
			k = 0
			n += 1
			ix -= 0x00100000
		}
		xAbs = fdSetHi(xAbs, ix)
		u := xAbs - bp[k]
		v := 1.0 / (xAbs + bp[k])
		ss := float64(u * v)
		sH := fdSetLo(ss, 0)
		tH := fdWords(((ix>>1)|0x20000000)+0x00080000+int32(k<<18), 0)
		tL := xAbs - (tH - bp[k])
		sL := float64(v * ((u - float64(sH*tH)) - float64(sH*tL)))
		s2 := float64(ss * ss)
		r := float64(float64(s2*s2) * fdPoly(s2, l1, l2, l3, l4, l5, l6))
		r += float64(sL * (sH + ss))
		s2 = float64(sH * sH)
		tH = 3.0 + s2 + r
		tH = fdSetLo(tH, 0)
		tL = r - ((tH - 3.0) - s2)
		u = float64(sH * tH)
		v = float64(sL*tH) + float64(tL*ss)
		pH = u + v
		pH = fdSetLo(pH, 0)
		pL = v - (pH - u)
		zH := float64(cpH * pH)
		zL := float64(cpL*pH) + float64(pL*cp) + dpL[k]
		t := float64(n)
		t1 = ((zH + zL) + dpH[k]) + t
		t1 = fdSetLo(t1, 0)
		t2 = zL - (((t1 - t) - dpH[k]) - zH)
	}
	y1 := fdSetLo(y, 0)
	pL = float64((y-y1)*t1) + float64(y*t2)
	pH = float64(y1 * t1)
	z := pL + pH
	j := fdHi(z)
	i := int32(fdLo(z))
	if j >= 0x40900000 {
		if ((j - 0x40900000) | i) != 0 {
			return s * math.Inf(1)
		}
		const ovt = 8.0085662595372944372e-0017
		if pL+ovt > z-pH {
			return s * math.Inf(1)
		}
	} else if (j & 0x7fffffff) >= 0x4090cc00 {
		if ((j - int32(-1064252416)) | i) != 0 { // 0xc090cc00
			return float64(s * 0.0)
		}
		if pL <= z-pH {
			return float64(s * 0.0)
		}
	}
	const (
		p1   = 0x1.555555555553ep-3
		p2   = -0x1.6c16c16bebd93p-9
		p3   = 0x1.1566aaf25de2cp-14
		p4   = -0x1.bbd41c5d26bf1p-20
		p5   = 0x1.6376972bea4d0p-25
		lg2  = 0x1.62e42fefa39efp-1
		lg2H = 0x1.62e43p-1
		lg2L = -0x1.05c610ca86c39p-29
	)
	i = j & 0x7fffffff
	k := (i >> 20) - 0x3ff
	n = 0
	if i > 0x3fe00000 {
		nn := j + (0x00100000 >> uint(k+1))
		k = ((nn & 0x7fffffff) >> 20) - 0x3ff
		t := fdWords(nn&^(0x000fffff>>uint(k)), 0)
		nn = ((nn & 0x000fffff) | 0x00100000) >> uint(20-k)
		if j < 0 {
			nn = -nn
		}
		n = int(nn)
		pH -= t
	}
	t := pL + pH
	t = fdSetLo(t, 0)
	u := float64(t * lg2H)
	v := float64((pL-(t-pH))*lg2) + float64(t*lg2L)
	z = u + v
	w := v - (z - u)
	t = float64(z * z)
	t1 = z - float64(t*fdPoly(t, p1, p2, p3, p4, p5))
	r := float64(z*t1)/(t1-2.0) - (w + float64(z*w))
	z = 1.0 - (r - z)
	j = fdHi(z)
	j += int32(n << 20)
	if (j >> 20) <= 0 {
		z = math.Ldexp(z, n)
	} else {
		z = fdSetHi(z, fdHi(z)+int32(n<<20))
	}
	return float64(s * z)
}
