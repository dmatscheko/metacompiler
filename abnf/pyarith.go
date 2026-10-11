package abnf

// PYTHON INTEGER PROMOTION PAST 2^53 for the Go twin.
//
// A plain Python int is a double in this value model, so it is EXACT up to 2^53
// and would silently round above it (`9007199254740992 + 1`). This is the
// TRIGGER that moves such a result into the arbitrary precision box.
//
// Twins: the other two are binOp's + / - / * arms in languages/python-interpreter.abnf
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
	"strings"
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

// pyNdZeros is the zero of every non-ASCII Unicode decimal-digit run (category
// Nd, Unicode 16.0; each is ten consecutive code points). int() and float() read
// any such digit as its ASCII one, as CPython's
// _PyUnicode_TransformDecimalAndSpaceToASCII does. Twin of pyDecAscii in
// languages/python-interpreter.abnf and lib/python-rt.metajs.
var pyNdZeros = []rune{
	1632, 1776, 1984, 2406, 2534, 2662, 2790, 2918, 3046, 3174, 3302, 3430, 3558, 3664,
	3792, 3872, 4160, 4240, 6112, 6160, 6470, 6608, 6784, 6800, 6992, 7088, 7232, 7248,
	42528, 43216, 43264, 43472, 43504, 43600, 44016, 65296, 66720, 68912, 68928, 69734,
	69872, 69942, 70096, 70384, 70736, 70864, 71248, 71360, 71376, 71386, 71472, 71904,
	72016, 72688, 72784, 73040, 73120, 73552, 90416, 92768, 92864, 93008, 93552, 118000,
	120782, 120792, 120802, 120812, 120822, 123200, 123632, 124144, 124401, 125264, 130032}

func pyDecAscii(s string) string {
	i := 0
	for i < len(s) && s[i] < 0x80 {
		i++
	}
	if i == len(s) {
		return s
	}
	var b strings.Builder
	b.WriteString(s[:i])
	for _, c := range s[i:] {
		d := rune(-1)
		for _, z := range pyNdZeros {
			if c >= z && c < z+10 {
				d = c - z
				break
			}
		}
		if d >= 0 {
			b.WriteRune('0' + d)
		} else {
			b.WriteRune(c)
		}
	}
	return b.String()
}

// pyOrdCheck raises CPython's TypeError for an ORDERING of two values of
// different kinds: only numbers (bool, int, float - not complex), str, two lists,
// two tuples and two sets have an order. Twin of pyOrdCheck in
// languages/python-interpreter.abnf and pyAOrdCheck in lib/python-rt.metajs.
func pyOrdKind(v interface{}) byte {
	switch t := v.(type) {
	case float64, bool, *jsPyFlo, *jsBigInt:
		return 'n'
	case string:
		return 's'
	case *jsArray:
		if t.kind == pyTupKind {
			return 'u'
		}
		return 'l'
	}
	if _, ok := pySetElems(v); ok {
		return 't'
	}
	return 0
}

func (rt *jsrt) pyOrdCheck(op string, l, r interface{}) {
	if k := pyOrdKind(l); k != 0 && k == pyOrdKind(r) {
		return
	}
	msg := "'" + op + "' not supported between instances of '" + pyTypeName(rt, l) + "' and '" + pyTypeName(rt, r) + "'"
	panic(&jsThrown{value: rt.pyExcInstance("TypeError", msg)})
}

// pyBinTypeErr is CPython's TypeError text for binary operator op on operands
// whose builtin values are a and b (a subclass instance unwrapped), named by l
// and r's own types; "" when op serves that pair. Twin of pyBinTypeErr in
// languages/python-interpreter.abnf and pyABinTypeErr in lib/python-rt.metajs.
func (rt *jsrt) pyBinTypeErr(op string, l, r, a, b interface{}) string {
	ln, rn := pyTypeName(rt, l), pyTypeName(rt, r)
	switch op {
	case "<", ">", "<=", ">=":
		if k := pyOrdKind(a); k != 0 && k == pyOrdKind(b) {
			return ""
		}
		return "'" + op + "' not supported between instances of '" + ln + "' and '" + rn + "'"
	case "+":
		kl := pyAddKind(a)
		if kl != 0 && kl == pyAddKind(b) {
			return ""
		}
		switch kl {
		case 's':
			return "can only concatenate str (not \"" + rn + "\") to str"
		case 'l':
			return "can only concatenate list (not \"" + rn + "\") to list"
		case 'u':
			return "can only concatenate tuple (not \"" + rn + "\") to tuple"
		}
	}
	na, nb := pyNumTypeName(a), pyNumTypeName(b)
	nums := na != "" && nb != ""
	ints := na == "int" && nb == "int"
	_, sa := pySetElems(a)
	_, sb := pySetElems(b)
	sets := sa && sb
	_, isStr := a.(string)
	switch op {
	case "-":
		if nums || sets {
			return ""
		}
	case "/", "//", "**":
		if nums {
			return ""
		}
	case "%":
		if nums || isStr {
			return ""
		}
	case "<<", ">>":
		if ints {
			return ""
		}
	case "&", "|", "^":
		if ints || sets {
			return ""
		}
		if _, _, da := dictParts(a); op == "|" && da && !sa {
			if _, _, db := dictParts(b); db && !sb {
				return ""
			}
		}
	case "*":
		if nums {
			return ""
		}
		qa, qb := pyAddKind(a), pyAddKind(b)
		seqA, seqB := qa == 's' || qa == 'l' || qa == 'u', qb == 's' || qb == 'l' || qb == 'u'
		if (seqA && nb == "int") || (seqB && na == "int") {
			return ""
		}
		if seqA {
			return "can't multiply sequence by non-int of type '" + rn + "'"
		}
		if seqB {
			return "can't multiply sequence by non-int of type '" + ln + "'"
		}
	}
	shown := op
	if op == "**" {
		shown = "** or pow()"
	}
	return "unsupported operand type(s) for " + shown + ": '" + ln + "' and '" + rn + "'"
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
			switch op := rt.toString(u(a[0])); op {
			case "<", ">", "<=", ">=":
				l, r := u(a[1]), u(a[2])
				_, lnum := l.(float64)
				_, rnum := r.(float64)
				if !lnum || !rnum {
					if v, ok := rt.pyDunderBin(op, l, r); ok {
						return w(v)
					}
					rt.pyOrdCheck(op, l, r)
				}
			case "+", "-", "*", "+=":
				if op == "+=" {
					op = "+"
				}
				if v, ok := rt.pyPromoteArith(op, u(a[1]), u(a[2])); ok {
					return w(v)
				}
			}
			return base(a)
		}
	})
}
