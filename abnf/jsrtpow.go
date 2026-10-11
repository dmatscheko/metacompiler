package abnf

// crPow is a correctly rounded pow: the double nearest to x^y, ties to even.
// math.Pow is up to hundreds of ulps off for a large y*log(x); the C library pow
// that CPython, node, Ruby, Lua and PHP call rounds correctly. Twin of d_pow in
// languages/lib/runtime.c, which does the same arithmetic.
//
// log(x) and then exp(y*log(x)) are evaluated in double-double arithmetic to a
// relative error below 2^-94. If the result rounds the same way at both ends of
// a 2^-88 error interval (Ziv's test) that rounding is the answer. Otherwise x^y
// is close to a midpoint between two doubles; the only inputs that sit ON one
// are those where x^y is a dyadic rational, and powExact computes those exactly.
//
// Every product is wrapped in float64() so that arm64 cannot fuse a*b+c into an
// FMA: the error-free transformations depend on each product being rounded.

import "math"

func crPow(x, y float64) float64 {
	if y == 0 || x == 1 || y == 1 || x == 0 || math.IsNaN(x) || math.IsNaN(y) ||
		math.IsInf(x, 0) || math.IsInf(y, 0) {
		return math.Pow(x, y) // the C99 special cases
	}
	neg := false
	if x < 0 {
		if y != math.Trunc(y) {
			return math.NaN()
		}
		x = -x
		neg = math.Abs(y) < 1<<53 && math.Mod(y, 2) != 0
	}
	r := powPos(x, y)
	if neg {
		return -r
	}
	return r
}

// powPos is x^y for a positive finite x and a finite non-zero y.
func powPos(x, y float64) float64 {
	switch y {
	case 2:
		return float64(x * x)
	case -1:
		return 1 / x
	case 0.5:
		return math.Sqrt(x)
	}
	lh, ll := powLogDD(x)
	ph, pl := ddTwoProd(y, lh)
	if ph > 710 {
		return math.Inf(1)
	}
	if ph < -746 {
		return 0
	}
	pl += float64(y * ll)
	ph, pl = ddFastTwoSum(ph, pl)
	k, vh, vl := powExpDD(ph, pl)
	err := float64(vh * 0x1p-88)
	c := math.Ldexp(1, -1022-k)
	if vh >= c {
		// A normal result: vh+vl rounds once, and scaling by 2^k is exact.
		hi := vh + (vl + err)
		if hi == vh+(vl-err) {
			return math.Ldexp(hi, k)
		}
		if r, ok := powExact(x, y); ok {
			return r
		}
		return math.Ldexp(vh+vl, k)
	}
	// A subnormal result rounds to a multiple of q = 2^(-1074-k): adding c puts
	// the rounding position of vh at q, and rem+vl is what that rounding dropped.
	q := math.Ldexp(1, -1074-k)
	s := c + vh
	j := s - c
	rem := vh - j
	up, dn := rem+(vl+err), rem+(vl-err)
	switch {
	case dn > q/2:
		j += q
	case up < -q/2:
		j -= q
	case up >= q/2 || dn <= -q/2:
		if r, ok := powExact(x, y); ok {
			return r
		}
		if rem+vl > q/2 {
			j += q
		} else if rem+vl < -q/2 {
			j -= q
		}
	}
	return math.Ldexp(math.Ldexp(j, k+600), -600)
}

// powLogDD is log(x) for a positive finite x as a double-double: one Newton step
// on exp from the double log, log(x) = l0 + log1p(x/exp(l0) - 1).
func powLogDD(x float64) (float64, float64) {
	l0 := math.Log(x)
	k, vh, vl := powExpDD(l0, 0)
	d := (math.Ldexp(x, -k) - vh) - vl
	c := d / vh
	c -= float64(c*c) * 0.5
	return ddFastTwoSum(l0, c)
}

// powExpDD is exp(ah+al) as 2^k * (vh+vl) with vh in [0.7, 1.42]: k*ln2 is
// removed with ln2 split in three (k*powLn2A is exact), the rest is divided by
// 16, expm1 of it is a Taylor polynomial, and four squarings undo the division.
func powExpDD(ah, al float64) (int, float64, float64) {
	const (
		powLn2A = 6.93147180369123816490e-01
		powLn2B = 1.9082149292705877e-10
		powLn2C = 1.1612227229362532e-26
	)
	k := math.Floor(float64(ah*1.4426950408889634) + 0.5)
	t, te := ddTwoProd(k, powLn2B)
	sh, sl := ddAdd(ah-float64(k*powLn2A), 0, -t, -te)
	sl += al - float64(k*powLn2C)
	sh, sl = ddFastTwoSum(sh, sl)
	sh, sl = float64(sh*0.0625), float64(sl*0.0625)
	// expm1(s) = s*(1 + s*(1/2! + s*(1/3! + ...))): the terms past 1/8! in
	// double, the rest in double-double.
	q := 1.6059043836821613e-10
	q = 2.08767569878681e-09 + float64(sh*q)
	q = 2.505210838544172e-08 + float64(sh*q)
	q = 2.755731922398589e-07 + float64(sh*q)
	q = 2.7557319223985893e-06 + float64(sh*q)
	th, tl := q, 0.0
	for _, c := range [...][2]float64{
		{2.48015873015873e-05, 2.1511947866775882e-23},
		{0.0001984126984126984, 1.7209558293420705e-22},
		{0.001388888888888889, -5.300543954373577e-20},
		{0.008333333333333333, 1.1564823173178714e-19},
		{0.041666666666666664, 2.3129646346357427e-18},
		{0.16666666666666666, 9.25185853854297e-18},
		{0.5, 0},
		{1, 0},
	} {
		th, tl = ddMul(sh, sl, th, tl)
		th, tl = ddAdd(c[0], c[1], th, tl)
	}
	uh, ul := ddMul(sh, sl, th, tl)
	for i := 0; i < 4; i++ { // expm1(2s) = expm1(s) * (2 + expm1(s))
		ph, pl := ddAdd(2, 0, uh, ul)
		uh, ul = ddMul(uh, ul, ph, pl)
	}
	vh, vl := ddFastTwoSum(1, uh)
	vh, vl = ddFastTwoSum(vh, vl+ul)
	return int(k), vh, vl
}

// powExact answers x^y, rounded once, when it is m*2^e with m an odd integer
// below 2^54 - a double, or a midpoint between two. With x = mx*2^ex and
// y = p*2^ey (mx, p odd) that needs mx to be a perfect 2^-ey-th power when y is
// not an integer, and the power of its root to stay below 2^54.
func powExact(x, y float64) (float64, bool) {
	fr, e := math.Frexp(x)
	mx := int64(math.Ldexp(fr, 53))
	ex := int64(e - 53)
	for mx&1 == 0 {
		mx >>= 1
		ex++
	}
	fr, e = math.Frexp(y)
	p := int64(math.Ldexp(fr, 53))
	ey := e - 53
	for p&1 == 0 {
		p >>= 1
		ey++
	}
	for i := ey; i < 0 && mx != 1; i++ {
		r := int64(math.Sqrt(float64(mx)))
		for r*r > mx {
			r--
		}
		for (r+1)*(r+1) <= mx {
			r++
		}
		if r*r != mx {
			return 0, false
		}
		mx = r
	}
	if ey >= 0 {
		if ey > 8 || p > 1<<40 || p < -(1<<40) {
			return 0, false
		}
		p <<= uint(ey)
		ex *= p
	} else {
		if ex%(int64(1)<<uint(-ey)) != 0 {
			return 0, false
		}
		ex = (ex >> uint(-ey)) * p
	}
	m := int64(1)
	if mx != 1 {
		if p < 0 {
			return 0, false
		}
		for i := int64(0); i < p; i++ {
			if m > (int64(1)<<54)/mx {
				return 0, false
			}
			m *= mx
		}
		if m >= int64(1)<<54 {
			return 0, false
		}
	}
	top := ex
	for t := m; t > 1; t >>= 1 {
		top++
	}
	if top >= -1022 {
		return math.Ldexp(float64(m), int(ex)), true
	}
	s := -1074 - ex
	if s > 60 {
		return 0, true
	}
	n := m >> uint(s)
	rest, half := m-n<<uint(s), int64(1)<<uint(s-1)
	if rest > half || (rest == half && n&1 == 1) {
		n++
	}
	return math.Ldexp(float64(n), -1074), true
}

// Double-double arithmetic: a value is an unevaluated sum hi+lo, |lo| at most
// half an ulp of hi.

func ddTwoSum(a, b float64) (float64, float64) {
	s := a + b
	bb := s - a
	return s, (a - (s - bb)) + (b - bb)
}

func ddFastTwoSum(a, b float64) (float64, float64) { // |a| >= |b|
	s := a + b
	return s, b - (s - a)
}

// ddTwoProd is a*b exactly, by Dekker's split into 26-bit halves.
func ddTwoProd(a, b float64) (float64, float64) {
	p := float64(a * b)
	t := float64(134217729 * a)
	ah := t - (t - a)
	al := a - ah
	t = float64(134217729 * b)
	bh := t - (t - b)
	bl := b - bh
	return p, ((float64(ah*bh) - p) + float64(ah*bl) + float64(al*bh)) + float64(al*bl)
}

func ddMul(ah, al, bh, bl float64) (float64, float64) {
	p, e := ddTwoProd(ah, bh)
	return ddFastTwoSum(p, e+(float64(ah*bl)+float64(al*bh)))
}

func ddAdd(ah, al, bh, bl float64) (float64, float64) {
	s, e := ddTwoSum(ah, bh)
	return ddFastTwoSum(s, e+(al+bl))
}
