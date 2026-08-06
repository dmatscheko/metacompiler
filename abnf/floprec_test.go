package abnf

import (
	"math"
	"strconv"
	"strings"
	"testing"
)

// floPrec's CONTRACT is "exactly n significant digits, trailing zeros stripped",
// and it has three implementations that have to agree (docs/todo.md 2.8):
//
//	floPrecStr here            - the binding both script hosts get (goja through
//	                             commonscript.go, the frozen VM through frozen.go),
//	                             and therefore what llvm.Run sees
//	languages/lib/runtime.c    - host id 70, over shortest_digits with g_fixdig
//	languages/metajs-interpreter.abnf - its hostGlobals entry, which IS this function
//
// The floor's used to answer max(n, shortest) instead, because g_mindig only
// started the round-trip SEARCH at n: floPrec(1/3, 1) was "3e-1" here and
// "3333333333333333e-1" natively. Nothing caught it because the one live caller
// (lua's %g) always asks for an n at least as large as the shortest form, where
// the two answers coincide - so the tests below deliberately live where they do
// NOT coincide, at n strictly below the shortest round-tripping length.
//
// This pins the Go side only. The floor's side is a native binary and is pinned
// by a differential probe (tests/probe.sh) rather than by a Go test.
func TestFloPrecExactlyNDigits(t *testing.T) {
	cases := []struct {
		v    float64
		n    int
		want string
	}{
		// n BELOW the shortest form - the whole point. 1/3 needs 16 digits.
		{1.0 / 3.0, 1, "3e-1"},
		{1.0 / 3.0, 2, "33e-1"},
		{1.0 / 3.0, 6, "333333e-1"},
		{2.0 / 3.0, 1, "7e-1"},   // rounds up
		{2.0 / 3.0, 3, "667e-1"}, // rounds up
		// The %e case the C floor got wrong: seven digits of 1234.5678 must be
		// the correctly rounded 1234568, not the first seven of the shortest
		// eight digit form (which would truncate to 1234567).
		{1234.5678, 7, "1234568e3"},
		{1234.5678, 4, "1235e3"},
		// Trailing zeros are stripped, so asking for more digits than the value
		// has is idempotent.
		{100.0, 1, "1e2"},
		{100.0, 17, "1e2"},
		{1.5, 2, "15e0"},
		{1.5, 17, "15e0"},
		// Half to EVEN, not half away from zero: 0.5 and 1.5 at one digit.
		{0.5, 1, "5e-1"},
		{2.5, 1, "2e0"},
		{3.5, 1, "4e0"},
		// The non-finite and zero arms are a fixed answer in every engine.
		{0, 5, "0e0"},
		{math.NaN(), 5, "0e0"},
		{math.Inf(1), 5, "0e0"},
		// n is clamped to [1, 17] rather than rejected.
		{1.0 / 3.0, 0, "3e-1"},
		{1.0 / 3.0, 99, "33333333333333331e-1"},
	}
	for _, c := range cases {
		if got := floPrecStr(c.v, c.n); got != c.want {
			t.Errorf("floPrec(%v, %d) = %q, want %q", c.v, c.n, got, c.want)
		}
	}
}

// The property behind the table: for every n, floPrec answers the digit string of
// C's "%.*e" - at most n digits, and exactly n before the trailing zeros come off.
// A sweep is what would have caught the floor's disagreement without knowing which
// value to look at.
func TestFloPrecMatchesPrintfE(t *testing.T) {
	vals := []float64{
		1.0 / 3.0, 2.0 / 3.0, 1234.5678, 0.1, 1e15 + 1, 9.87654321e-5,
		math.Pi, 5e-324, 1.7976931348623157e308, 2.2250738585072014e-308,
		0.0013060363926342689, 1e100, 1e-100, 0.30000000000000004, 123456789.0,
	}
	for _, v := range vals {
		for n := 1; n <= 17; n++ {
			got := floPrecStr(v, n)
			// strconv's 'e' at precision n-1 IS "%.{n-1}e", i.e. n significant
			// digits, correctly rounded half to even.
			ref := strconv.FormatFloat(math.Abs(v), 'e', n-1, 64)
			i := strings.IndexByte(ref, 'e')
			mant := strings.Replace(ref[:i], ".", "", 1)
			exp := ref[i+1:]
			for len(mant) > 1 && mant[len(mant)-1] == '0' {
				mant = mant[:len(mant)-1]
			}
			e, _ := strconv.Atoi(exp)
			want := mant + "e" + strconv.Itoa(e)
			if got != want {
				t.Errorf("floPrec(%v, %d) = %q, want %q", v, n, got, want)
			}
		}
	}
}
