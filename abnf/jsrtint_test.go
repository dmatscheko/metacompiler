package abnf

import (
	"math/rand"
	"strconv"
	"testing"
)

// TestGiArithMatchesGo is the differential test for the sized-integer box: the
// SAME operation is computed by jsrtint.go and by Go's own native operators at
// the same width and signedness, over every edge case plus a deterministic
// random sample. Go is the oracle here, which is exactly what makes the test
// worth having: the box exists to reproduce Go's (and Java's, and C#'s) integer
// semantics, and nothing else in this tree can check that claim.
//
// languages/lib/interp-core.js implements the identical rules for the
// interpreter halves; the two were validated against the same vectors.
func TestGiArithMatchesGo(t *testing.T) {
	type spec struct {
		w uint8
		u bool
	}
	specs := []spec{{8, false}, {8, true}, {16, false}, {16, true},
		{32, false}, {32, true}, {64, false}, {64, true}}

	trunc := func(v int64, s spec) int64 { return giTrunc(v, s.w, s.u) }
	uval := func(v int64, s spec) uint64 { return giU(v, s.w) }
	text := func(v int64, s spec) string {
		if s.u {
			return strconv.FormatUint(uval(v, s), 10)
		}
		return strconv.FormatInt(v, 10)
	}
	minOf := func(s spec) int64 { return trunc(int64(1)<<(s.w-1), s) }

	seeds := []int64{0, 1, -1, 2, 3, -3, 6, 127, 128, 255, 256, -128, -129,
		32767, 65535, 2147483647, -2147483648, 4294967295,
		9007199254740992, -9007199254740992, 9007199254740993, -9007199254740993,
		9223372036854775807, -9223372036854775808, 1000000007, 1000000000000}
	r := rand.New(rand.NewSource(20260727))
	for i := 0; i < 300; i++ {
		seeds = append(seeds, int64(r.Uint64()))
		seeds = append(seeds, int64(r.Uint64()>>uint(r.Intn(63))))
	}

	rt := &jsrt{}
	got := func(v interface{}) string {
		if b, ok := v.(jsGInt); ok {
			return giStr(b)
		}
		return strconv.FormatInt(int64(v.(float64)), 10)
	}

	ops := []string{"+", "-", "*", "/", "%", "&", "|", "^", "&^", "<<", ">>"}
	n := 0
	for _, s := range specs {
		for _, a0 := range seeds {
			a := trunc(a0, s)
			av := giNorm(a, s.w, s.u)

			// Unary and conversion.
			if g, wnt := got(giNorm(-a, s.w, s.u)), text(trunc(-a, s), s); g != wnt {
				t.Fatalf("neg %d w%d u%v: got %s want %s", a, s.w, s.u, g, wnt)
			}
			if g, wnt := got(giNorm(^a, s.w, s.u)), text(trunc(^a, s), s); g != wnt {
				t.Fatalf("not %d w%d u%v: got %s want %s", a, s.w, s.u, g, wnt)
			}
			if g, wnt := got(rt.giConv(giNorm(a0, 64, false), s.w, s.u)), text(trunc(a0, s), s); g != wnt {
				t.Fatalf("conv %d w%d u%v: got %s want %s", a0, s.w, s.u, g, wnt)
			}

			for _, b0 := range seeds[:26] {
				b := trunc(b0, s)
				bv := giNorm(b, s.w, s.u)

				// Ordered comparison.
				want := 0
				if s.u {
					if uval(a, s) < uval(b, s) {
						want = -1
					} else if uval(a, s) > uval(b, s) {
						want = 1
					}
				} else if a < b {
					want = -1
				} else if a > b {
					want = 1
				}
				if c := rt.giCmp(av, bv); c != want {
					t.Fatalf("cmp %d %d w%d u%v: got %d want %d", a, b, s.w, s.u, c, want)
				}

				for _, op := range ops {
					// Go itself panics on these, so there is nothing to compare.
					if (op == "/" || op == "%") && (b == 0 || (!s.u && a == minOf(s) && b == -1)) {
						continue
					}
					if (op == "<<" || op == ">>") && !s.u && b < 0 {
						continue
					}
					n++
					if g, wnt := got(rt.giArith(op, av, bv)), text(goOp(op, a, b, s.w, s.u), s); g != wnt {
						t.Fatalf("%d %s %d w%d u%v: got %s want %s", a, op, b, s.w, s.u, g, wnt)
					}
				}
			}
		}
	}
	t.Logf("%d operator vectors, all matching Go", n)
}

// goOp is the same operator computed with Go's own native operators at width w.
func goOp(op string, a, b int64, w uint8, u bool) int64 {
	if u {
		ua, ub := giU(a, w), giU(b, w)
		var x uint64
		switch op {
		case "+":
			x = ua + ub
		case "-":
			x = ua - ub
		case "*":
			x = ua * ub
		case "/":
			x = ua / ub
		case "%":
			x = ua % ub
		case "&":
			x = ua & ub
		case "|":
			x = ua | ub
		case "^":
			x = ua ^ ub
		case "&^":
			x = ua &^ ub
		case "<<":
			if uint64(b) >= uint64(w) {
				x = 0
			} else {
				x = ua << uint(b)
			}
		case ">>":
			if uint64(b) >= uint64(w) {
				x = 0
			} else {
				x = ua >> uint(b)
			}
		}
		return giTrunc(int64(x), w, u)
	}
	var x int64
	switch op {
	case "+":
		x = a + b
	case "-":
		x = a - b
	case "*":
		x = a * b
	case "/":
		x = a / b
	case "%":
		x = a % b
	case "&":
		x = a & b
	case "|":
		x = a | b
	case "^":
		x = a ^ b
	case "&^":
		x = a &^ b
	case "<<":
		if b >= int64(w) {
			x = 0
		} else {
			x = a << uint(b)
		}
	case ">>":
		s := b
		if s >= int64(w) {
			s = int64(w) - 1
		}
		x = a >> uint(s)
	}
	return giTrunc(x, w, u)
}
