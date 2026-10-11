package abnf

import (
	"math"
	"math/big"
	"math/bits"
	"strconv"
	"strings"
)

// int's and float's own methods for the compiler half: conjugate, is_integer,
// as_integer_ratio, bit_length, bit_count, to_bytes and float.hex, plus the type
// methods int.from_bytes and float.fromhex. A bool is an int. Twin of
// pyIsNumMethod / pyNumMethod in languages/python-interpreter.abnf and
// languages/lib/python-rt.metajs, whose algorithms are lib/python-num.js.

// pyIsNumMethod is the NAME SET, so hasattr()/getattr() ask what the dispatcher
// answers.
func pyIsNumMethod(v interface{}, name string) bool {
	isInt, isFlo := false, false
	switch v.(type) {
	case float64, bool, *jsBigInt:
		isInt = true
	case *jsPyFlo:
		isFlo = true
	}
	switch name {
	case "conjugate", "is_integer", "as_integer_ratio":
		return isInt || isFlo
	case "hex":
		return isFlo
	case "bit_length", "bit_count", "to_bytes":
		return isInt
	}
	return false
}

// pyRaise raises a catchable Python exception of the builtin class cls.
func (rt *jsrt) pyRaise(cls, msg string) {
	panic(&jsThrown{value: rt.pyExcInstance(cls, msg)})
}

func (rt *jsrt) pyNumMethod(v interface{}, name string, args []interface{}, kw interface{}) interface{} {
	if f, ok := v.(*jsPyFlo); ok {
		switch name {
		case "conjugate":
			return f
		case "is_integer":
			return !math.IsInf(f.f, 0) && f.f == math.Trunc(f.f)
		case "hex":
			return pyFloatHex(f.f)
		}
		if math.IsNaN(f.f) {
			rt.pyRaise("ValueError", "cannot convert NaN to integer ratio")
		}
		if math.IsInf(f.f, 0) {
			rt.pyRaise("OverflowError", "cannot convert Infinity to integer ratio")
		}
		r := new(big.Rat).SetFloat64(f.f)
		return pyTup(&jsArray{elems: []interface{}{rt.pyBigNarrow(new(big.Int).Set(r.Num())),
			rt.pyBigNarrow(new(big.Int).Set(r.Denom()))}})
	}
	x, _ := pyBigOperand(v)
	switch name {
	case "conjugate":
		re, _ := pyNumAttr(v, "real")
		return re
	case "is_integer":
		return true
	case "as_integer_ratio":
		return pyTup(&jsArray{elems: []interface{}{rt.pyBigNarrow(x), 1.0}})
	case "bit_length":
		return float64(x.BitLen())
	case "bit_count":
		n := 0
		for _, w := range x.Bits() {
			n += bits.OnesCount(uint(w))
		}
		return float64(n)
	}
	// to_bytes(length=1, byteorder="big", *, signed=False)
	n := 1
	if l := pyArgOrKw(rt, args, 0, kw, "length"); l != jsUndef {
		n = int(rt.toNumber(l))
	}
	if n < 0 {
		rt.pyRaise("ValueError", "length argument must be non-negative")
	}
	little := rt.pyByteOrder(pyArgOrKw(rt, args, 1, kw, "byteorder"))
	signed := rt.pyTruthyOf(rt.pyKwArg(kw, "signed"))
	if x.Sign() < 0 && !signed {
		rt.pyRaise("OverflowError", "can't convert negative int to unsigned")
	}
	nb := uint(8 * n)
	t := new(big.Int).Set(x)
	if x.Sign() < 0 {
		t.Add(t, new(big.Int).Lsh(big.NewInt(1), nb))
	}
	// Signed, the top bit of the n bytes must agree with the sign.
	fits := t.Sign() >= 0 && uint(t.BitLen()) <= nb
	if signed && fits && nb > 0 && (t.Bit(int(nb-1)) == 1) != (x.Sign() < 0) {
		fits = false
	}
	if signed && nb == 0 && x.Sign() != 0 {
		fits = false
	}
	if !fits {
		rt.pyRaise("OverflowError", "int too big to convert")
	}
	buf := t.FillBytes(make([]byte, n))
	out := &jsArray{}
	for i := range buf {
		b := buf[i]
		if little {
			b = buf[n-1-i]
		}
		out.elems = append(out.elems, float64(b))
	}
	return out
}

// pyArgOrKw is positional argument i, else the keyword of that name, else jsUndef.
func pyArgOrKw(rt *jsrt, args []interface{}, i int, kw interface{}, name string) interface{} {
	if i < len(args) {
		return args[i]
	}
	return rt.pyKwArg(kw, name)
}

// pyByteOrder reads a byteorder argument: true for "little", false for "big" or
// absent.
func (rt *jsrt) pyByteOrder(v interface{}) bool {
	if v == jsUndef {
		return false
	}
	if s, ok := v.(string); ok && (s == "big" || s == "little") {
		return s == "little"
	}
	rt.pyRaise("ValueError", "byteorder must be either 'little' or 'big'")
	return false
}

// pyIntFromBytes is int.from_bytes(bytes, byteorder="big", *, signed=False). Read
// as a value and called, a builtin receives its keyword dict as a trailing
// positional (pyBindCallR); byteorder is a str, so a trailing dict is the dict.
func (rt *jsrt) pyIntFromBytes(args []interface{}, kw interface{}) interface{} {
	if len(args) >= 2 {
		if _, _, isDict := dictParts(args[len(args)-1]); isDict {
			kw = args[len(args)-1]
			args = args[:len(args)-1]
		}
	}
	bs := pyArgOrKw(rt, args, 0, kw, "bytes")
	little := rt.pyByteOrder(pyArgOrKw(rt, args, 1, kw, "byteorder"))
	if _, isStr := bs.(string); isStr {
		rt.pyRaise("TypeError", "cannot convert 'str' object to bytes")
	}
	elems := rt.pyElemsOf(bs)
	n := len(elems)
	buf := make([]byte, n)
	for i, e := range elems {
		b := rt.toNumber(e)
		if b < 0 || b > 255 {
			rt.pyRaise("ValueError", "bytes must be in range(0, 256)")
		}
		if little {
			buf[n-1-i] = byte(b)
		} else {
			buf[i] = byte(b)
		}
	}
	t := new(big.Int).SetBytes(buf)
	if rt.pyTruthyOf(rt.pyKwArg(kw, "signed")) && n > 0 && buf[0]&0x80 != 0 {
		t.Sub(t, new(big.Int).Lsh(big.NewInt(1), uint(8*n)))
	}
	return rt.pyBigNarrow(t)
}

// pyFloatHex is float.hex, CPython's float_hex: frexp's mantissa shifted so a
// normal value leads with 1 and a subnormal with 0 at exponent -1022, then 13 hex
// digits.
func pyFloatHex(x float64) string {
	switch {
	case math.IsNaN(x):
		return "nan"
	case math.IsInf(x, 1):
		return "inf"
	case math.IsInf(x, -1):
		return "-inf"
	case x == 0:
		if math.Signbit(x) {
			return "-0x0.0p+0"
		}
		return "0x0.0p+0"
	}
	m, e := math.Frexp(math.Abs(x))
	shift := 1
	if -1021-e > 0 {
		shift = 1 - (-1021 - e)
	}
	m = math.Ldexp(m, shift)
	e -= shift
	const digits = "0123456789abcdef"
	d := int(m)
	var sb strings.Builder
	sb.WriteByte(digits[d])
	sb.WriteByte('.')
	m -= float64(d)
	for i := 0; i < 13; i++ {
		m *= 16
		d = int(m)
		sb.WriteByte(digits[d])
		m -= float64(d)
	}
	sign := ""
	if x < 0 {
		sign = "-"
	}
	esign := "+"
	if e < 0 {
		esign = "-"
		e = -e
	}
	return sign + "0x" + sb.String() + "p" + esign + strconv.Itoa(e)
}

// pyFloatFromHex is float.fromhex: CPython's syntax (optional sign, inf/infinity/
// nan, optional 0x, hex digits with an optional point, optional p exponent,
// surrounding whitespace) read here, and the rounding left to strconv's
// hexadecimal parser, which rounds to nearest, ties to even.
func (rt *jsrt) pyFloatFromHex(src string) interface{} {
	invalid := func() { rt.pyRaise("ValueError", "invalid hexadecimal floating-point string") }
	s := strings.Trim(src, " \t\n\r\f\v")
	neg := false
	if s != "" && (s[0] == '-' || s[0] == '+') {
		neg = s[0] == '-'
		s = s[1:]
	}
	switch strings.ToLower(s) {
	case "inf", "infinity":
		if neg {
			return &jsPyFlo{f: math.Inf(-1)}
		}
		return &jsPyFlo{f: math.Inf(1)}
	case "nan":
		return &jsPyFlo{f: math.NaN()}
	}
	if len(s) >= 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') {
		s = s[2:]
	}
	p := 0
	var mant strings.Builder
	ndig, frac, seenPoint := 0, 0, false
	for ; p < len(s); p++ {
		c := s[p]
		if c == '.' && !seenPoint {
			seenPoint = true
			continue
		}
		if !strings.ContainsRune("0123456789abcdefABCDEF", rune(c)) {
			break
		}
		mant.WriteByte(c)
		ndig++
		if seenPoint {
			frac++
		}
	}
	if ndig == 0 {
		invalid()
	}
	pexp := 0
	if p < len(s) && (s[p] == 'p' || s[p] == 'P') {
		p++
		eneg := false
		if p < len(s) && (s[p] == '-' || s[p] == '+') {
			eneg = s[p] == '-'
			p++
		}
		nd := 0
		for ; p < len(s) && s[p] >= '0' && s[p] <= '9'; p++ {
			// Past any double's range the exact value no longer matters.
			if pexp < 100000000 {
				pexp = pexp*10 + int(s[p]-'0')
			}
			nd++
		}
		if nd == 0 {
			invalid()
		}
		if eneg {
			pexp = -pexp
		}
	}
	if p != len(s) {
		invalid()
	}
	f, err := strconv.ParseFloat("0x"+mant.String()+"p"+strconv.Itoa(pexp-4*frac), 64)
	if err != nil && math.IsInf(f, 0) {
		rt.pyRaise("OverflowError", "hexadecimal value too large to represent as a float")
	}
	if neg {
		f = -f
	}
	return &jsPyFlo{f: f}
}

// pyRound is round(x[, n]). An int with no n, or with n >= 0, is already round
// and comes back as the int it is; a float with no n is the EXACT int of the
// double rounded half to even (round(1e300) has 301 digits). A negative n
// rounds to a multiple of 10**-n (pyRoundTens). Past 400 digits a float is its
// own rounding. Twin of hostGlobals["round"] in languages/python-interpreter.abnf
// and js_pyround in lib/python-rt.metajs.
func (rt *jsrt) pyRound(args []interface{}) interface{} {
	v := argAt(args, 0)
	hasN := len(args) >= 2 && !isUndefOrNull(args[1])
	n := 0
	if hasN {
		n = int(math.Max(-1e6, math.Min(400, math.Trunc(rt.toNumber(args[1])))))
	}
	if pyNumTypeName(v) == "int" {
		i := rt.pyIntNew([]interface{}{v})
		if n >= 0 {
			return i
		}
		x, _ := pyBigOperand(i)
		return rt.pyBigNarrow(pyRoundTens(x, -n, false))
	}
	f := rt.pyToF(v)
	finite := !math.IsNaN(f) && !math.IsInf(f, 0)
	if !hasN {
		if !finite { // int() raises CPython's OverflowError / ValueError.
			return rt.pyIntNew([]interface{}{v})
		}
		return rt.pyIntNew([]interface{}{&jsPyFlo{f: pyRoundHalfEven(f, 0)}})
	}
	if n >= 0 || !finite {
		return &jsPyFlo{f: pyRoundHalfEven(f, n)}
	}
	t := math.Trunc(f)
	x, _ := new(big.Float).SetFloat64(t).Int(nil)
	r, _ := new(big.Float).SetInt(pyRoundTens(x, -n, f != t)).Float64()
	if r == 0 && math.Signbit(f) {
		r = math.Copysign(0, -1)
	}
	return &jsPyFlo{f: r}
}

// pyRoundTens is round(i, -k) for k > 0: the nearest multiple of 10**k, a tie
// going to the even multiple. sticky says the true value lies just above |i|
// (a float's dropped fraction), which breaks a tie upward. Twin of pynRoundTens
// in languages/lib/python-num.js.
func pyRoundTens(i *big.Int, k int, sticky bool) *big.Int {
	m := new(big.Int).Abs(i)
	if len(m.String()) < k {
		return new(big.Int)
	}
	p := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(k)), nil)
	q, rem := new(big.Int).QuoRem(m, p, new(big.Int))
	if c := new(big.Int).Lsh(rem, 1).Cmp(p); c > 0 || (c == 0 && (sticky || q.Bit(0) == 1)) {
		q.Add(q, big.NewInt(1))
	}
	q.Mul(q, p)
	if i.Sign() < 0 {
		q.Neg(q)
	}
	return q
}
