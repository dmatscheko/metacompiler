package abnf

import (
	"math"
	"testing"
)

// TestRubyModIsFused pins the ONE property that `x - math.Floor(x/y)*y` had only
// by accident of code generation.
//
// Ruby's Integer#% is a floored remainder, and both of this project's halves have
// to answer the same thing. The Go twin's source spelling was
// `x - math.Floor(x/y)*y`, which the Go compiler contracts into a single arm64
// FMSUBD: the product and the subtraction round ONCE. clang does not contract the
// equivalent source in the C floor, so layer 2 rounds twice - and the two answers
// part company the moment q*y leaves the range a double holds exactly.
// ruby 2.6.10 says `9007199254740991 % -3` is -2, which is the FUSED answer, so
// layer 2 grew rbFmaSub (Dekker two-product / two-sum) to reproduce it.
//
// That is why the twin now says math.FMA out loud. The test below cannot fail on
// an arm64 machine that fuses anyway - fused and math.FMA are the same number
// there, which is exactly the point - so it asserts the two things that CAN be
// observed here:
//
//   - the operand really is in the range where fusion matters: the unfused
//     computation (an explicit float64() conversion of the product is the Go
//     spec's own way to forbid contraction) gives a DIFFERENT answer, so a
//     non-fusing target would have re-opened the divergence;
//   - math.FMA agrees with what real ruby prints.
func TestRubyModIsFused(t *testing.T) {
	cases := []struct {
		x, y, want float64
	}{
		// The case from the 36,000-line differential probe, and more from the same
		// range. Every `want` is what /usr/bin/ruby prints for the Integer
		// expression, where an Integer is exact - and every operand here is
		// exactly representable as a double, which is what Ruby's Integer is in
		// this project (see the manual, 7.5).
		{9007199254740991, -3, -2},
		{9007199254740991, -7, -4},
		{9007199254740991, 7, 3},
		{9007199254740992, -3, -1},
		{9007199254740992, -7, -3},
		{18014398509481982, -7, -1},
		// One where fusion cannot matter, so a regression that broke the ordinary
		// case would still be caught.
		{7, -3, -2},
		// A CEILING that fusion does not lift, and that no row here may cross:
		// once |x/y| passes 2^53 the QUOTIENT is rounded too, and no spelling of
		// the subtraction recovers the remainder. 123456789012345680 % -3 is 8 in
		// both halves and -1 in real ruby. Both halves agree, so it is a
		// divergence from MRI and not a divergence between engines, and it is
		// outside what this test is about.
	}
	split := 0
	for _, c := range cases {
		q := math.Floor(c.x / c.y)
		fused := math.FMA(-q, c.y, c.x)
		if fused != c.want {
			t.Errorf("math.FMA(-floor(%v/%v), %v, %v) = %v, want %v (real ruby)",
				c.x, c.y, c.y, c.x, fused, c.want)
		}
		// float64(...) is an EXPLICIT rounding of the intermediate, which the Go
		// spec forbids an implementation to contract away. This is the answer a
		// target with no fused multiply-add would produce.
		if unfused := c.x - float64(q*c.y); unfused != fused {
			split++
		}
	}
	if split == 0 {
		t.Error("no case in this table distinguishes fused from unfused - the test " +
			"proves nothing; pick operands where q*y leaves the exactly-representable range")
	}
}
