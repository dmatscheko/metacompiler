package abnf

// PYTHON INTEGER PROMOTION PAST 2^53 for the Go twin (docs/todo.md 1.1).
//
// A plain Python int is a double in this value model, so it is EXACT up to 2^53
// and silently rounds above it: `9007199254740992 + 1` answered
// 9007199254740992 in all three engines, and 54 of the 55 differing rows of a
// 626-literal probe against CPython were the `*` column. The arbitrary precision
// box already existed - the literal half landed in 506000a - so what is added
// here is the TRIGGER, in the last of the three engines.
//
// The other two are binOp's + / - / * arms in languages/python-interpreter.abnf
// (pyOver53) and the same arms of js_pybin in languages/lib/python-rt.metajs
// (pyAOver53 / pyAPromote). Keep the three in step.
//
// THE GUARD IS SOUND BOTH WAYS, which is what keeps the fast path to two
// comparisons in the two engines where the cost is measurable. By the time an
// arithmetic arm runs, both operands are plain (anything larger is already a box
// and was taken by the pyBigPair arm), so each is an integer of magnitude
// <= 2^53. If the TRUE result is >= 2^53 then so is the rounded double -
// rounding to nearest of a value at or above 2^53 cannot land below it, because
// 2^53 is representable - and if the true result is < 2^53 the double is exact.
// So no out-of-range answer escapes, and no in-range one pays for the box.
//
// Registration is additive, like abnf/jsrtregexpy.go's: an init() appends to
// rxExtraExterns and WRAPS js_pybin rather than editing it. A pair that is not
// two plain exact ints, or an operator that is not + - or *, is handed straight
// on to the wrapped dispatcher, so a user class still gets its __add__ chance
// and a set still gets the set algebra - those operands never reach the guard.

import (
	"math"
	"math/big"
)

const pyMax53 = 9007199254740992.0 // 2^53

// pyPlainInt is "a plain Python int": a float64 (or a bool, since a Python bool
// IS an int) that is integral and inside the exactly-representable range. A
// *jsPyFlo, a *jsBigInt, a string, an array and an instance all answer false, so
// none of them can be diverted into the big path by this guard.
func pyPlainInt(v interface{}) (float64, bool) {
	switch t := v.(type) {
	case float64:
		if t == math.Trunc(t) && math.Abs(t) <= pyMax53 {
			return t, true
		}
	case bool:
		if t {
			return 1, true
		}
		return 0, true
	}
	return 0, false
}

// pyPromoteArith is the exact answer for `l op r` when both are plain exact ints
// and the double result has left the exact range, or ok=false to keep the double.
func (rt *jsrt) pyPromoteArith(op string, l, r interface{}) (interface{}, bool) {
	x, okl := pyPlainInt(l)
	if !okl {
		return nil, false
	}
	y, okr := pyPlainInt(r)
	if !okr {
		return nil, false
	}
	var d float64
	switch op {
	case "+":
		d = x + y
	case "-":
		d = x - y
	case "*":
		d = x * y
	default:
		return nil, false
	}
	if math.Abs(d) < pyMax53 {
		return nil, false // Exact already: the ordinary double arm answers it.
	}
	bx, by := big.NewFloat(x), big.NewFloat(y)
	ix, _ := bx.Int(nil)
	iy, _ := by.Int(nil)
	out := new(big.Int)
	switch op {
	case "+":
		out.Add(ix, iy)
	case "-":
		out.Sub(ix, iy)
	case "*":
		out.Mul(ix, iy)
	}
	return rt.pyBigNarrow(out), true
}

func init() {
	rxExtraExterns = append(rxExtraExterns, func(rt *jsrt, m map[string]func(args []uint64) uint64) {
		u := rt.unwrap
		w := rt.wrap
		base := m["js_pybin"]
		if base == nil {
			return
		}
		m["js_pybin"] = func(a []uint64) uint64 {
			switch rt.toString(u(a[0])) {
			case "+", "-", "*":
				if v, ok := rt.pyPromoteArith(rt.toString(u(a[0])), u(a[1]), u(a[2])); ok {
					return w(v)
				}
			}
			return base(a)
		}
	})
}
