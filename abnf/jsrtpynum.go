package abnf

import (
	"math"
	"math/big"
	"math/bits"
	"reflect"
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

// ---- hash() ----
//
// CPython hashes in Py_uhash_t: 64-bit unsigned arithmetic that wraps. Twin of
// the pynHash* functions in languages/lib/python-num.js, which carry the word
// as four 16-bit limbs, and of hostGlobals["hash"] in
// languages/python-interpreter.abnf, which has the rules.

const pyHashP = uint64(1)<<61 - 1

// pyHashFix: -1 is the error return of CPython's hash slots, so it is -2.
func pyHashFix(x uint64) uint64 {
	if x == ^uint64(0) {
		return ^uint64(1)
	}
	return x
}

// pyHashBig is long_hash: the magnitude modulo 2^61 - 1, signed.
func pyHashBig(n *big.Int) uint64 {
	m := new(big.Int).Mod(new(big.Int).Abs(n), new(big.Int).SetUint64(pyHashP)).Uint64()
	if n.Sign() < 0 {
		m = -m
	}
	return pyHashFix(m)
}

// pyHashSmall is the hash of an int below 2^53 in size: itself, -1 excepted.
func pyHashSmall(n float64) uint64 { return pyHashFix(uint64(int64(n))) }

// pyHashFloat is _Py_HashDouble for a finite v that is not integral below 2^53:
// the mantissa enters 28 bits at a time and the binary exponent as a rotation,
// since 2^61 = 1 modulo 2^61 - 1.
func pyHashFloat(v float64) uint64 {
	m, e := math.Frexp(math.Abs(v))
	if m == 0 {
		return 0
	}
	x := uint64(0)
	for m != 0 {
		x = ((x << 28) & pyHashP) | x>>(61-28)
		m *= 268435456.0
		e -= 28
		y := uint64(m)
		m -= float64(y)
		x += y
		if x >= pyHashP {
			x -= pyHashP
		}
	}
	if e >= 0 {
		e = e % 61
	} else {
		e = 60 - (-1-e)%61
	}
	x = ((x << uint(e)) & pyHashP) | x>>uint(61-e)
	if v < 0 {
		x = -x
	}
	return pyHashFix(x)
}

// pyHashTuple is tuplehash: xxHash's round over the items' hashes.
func pyHashTuple(lanes []uint64) uint64 {
	const p1, p2, p5 = 11400714785074694791, 14029467366897019727, 2870177450012600261
	acc := uint64(p5)
	for _, l := range lanes {
		acc += l * p2
		acc = acc<<31 | acc>>33
		acc *= p1
	}
	acc += uint64(len(lanes)) ^ (p5 ^ 3527539)
	if acc == ^uint64(0) {
		return 1546275796
	}
	return acc
}

// pyHashFrozen is frozenset_hash: the members' shuffled hashes XORed, then the
// size and a final dispersal.
func pyHashFrozen(lanes []uint64) uint64 {
	h := uint64(0)
	for _, l := range lanes {
		h ^= ((l ^ 89869747) ^ (l << 16)) * 3644798167
	}
	h ^= (uint64(len(lanes)) + 1) * 1927868237
	h ^= (h >> 11) ^ (h >> 25)
	h = h*69069 + 907133923
	if h == ^uint64(0) {
		return 590923713
	}
	return h
}

// pyHashUnits is the fixed polynomial over UTF-16 code units that stands in for
// CPython's randomized str hash.
func pyHashUnits(units []uint16) uint64 {
	h := uint64(0)
	for _, u := range units {
		h = (h*31 + uint64(u)) % 2147483647
	}
	return h
}

// pyHashW is hash(v) as a word; js_pyhash reads it signed.
func (rt *jsrt) pyHashW(v interface{}) uint64 {
	switch t := v.(type) {
	case jsUndefT, jsNullT:
		return 4238894112
	case bool:
		if t {
			return 1
		}
		return 0
	case float64:
		return pyHashSmall(t)
	case *jsBigInt:
		return pyHashBig(t.v)
	case *jsPyFlo:
		return rt.pyHashF(t.f, v)
	case string:
		return pyHashUnits(strUnits(t))
	}
	if re, im, ok := pyComplexParts(v); ok {
		return pyHashFix(rt.pyHashF(re, v) + rt.pyHashF(im, v)*1000003)
	}
	if arr, ok := v.(*jsArray); ok && arr.kind == pyTupKind {
		lanes := make([]uint64, len(arr.elems))
		for i, e := range arr.elems {
			lanes[i] = rt.pyHashW(e)
		}
		return pyHashTuple(lanes)
	}
	if els, ok := pySetElems(v); ok && pySetFrozen(v) {
		lanes := make([]uint64, len(els.elems))
		for i, e := range els.elems {
			lanes[i] = rt.pyHashW(e)
		}
		return pyHashFrozen(lanes)
	}
	// A union hashes as the frozenset of its types.
	if u, ok := pyUnionOf(v); ok {
		lanes := make([]uint64, len(u.elems))
		for i, e := range u.elems {
			lanes[i] = rt.pyHashW(e)
		}
		return pyHashFrozen(lanes)
	}
	_, isArr := v.(*jsArray)
	_, isSet := pySetElems(v)
	_, _, isDict := dictParts(v)
	if isArr || isSet || isDict || pyIsDView(v) {
		panic(&jsThrown{value: rt.pyExcInstance("TypeError", "unhashable type: '"+pyTypeName(rt, v)+"'")})
	}
	if _, cls, ok := pyInstance(v); ok {
		switch m, how := pyHashOwner(cls); how {
		case pyHashByMethod:
			return rt.pyHashCall(m, v)
		case pyHashByNone:
			panic(&jsThrown{value: rt.pyExcInstance("TypeError", "unhashable type: '"+pyTypeName(rt, v)+"'")})
		case pyHashByBuiltin:
			return rt.pyHashW(pyUnwrapBuiltin(v))
		}
	}
	return rt.pyHashId(v)
}

// How an instance hashes, as the first class on its resolution order that
// settles it decides: object's identity hash, its own __hash__, none at all
// (`__hash__ = None`, or an __eq__ without a __hash__), or a builtin base's.
const (
	pyHashById = iota
	pyHashByMethod
	pyHashByNone
	pyHashByBuiltin
)

// pyHashOwner answers how instances of cls hash, and the __hash__ for
// pyHashByMethod. Twin of pyHashOwner in languages/python-interpreter.abnf and
// lib/python-rt.metajs.
func pyHashOwner(cls *jsObject) (interface{}, int) {
	for _, c := range pyMRO(cls) {
		m, hasH := c.props["__hash__"]
		if hasH && !isUndefOrNull(m) {
			return m, pyHashByMethod
		}
		if _, hasEq := c.props["__eq__"]; hasEq || hasH {
			return nil, pyHashByNone
		}
		if n := pyBuiltinClsName(c); n == "object" {
			break
		} else if n != "" {
			return nil, pyHashByBuiltin
		}
	}
	return nil, pyHashById
}

// pyHashCall is a class's own __hash__ m on v: the answer must be an int, kept
// when it fits 64 bits and hashed as an int otherwise.
func (rt *jsrt) pyHashCall(m, v interface{}) uint64 {
	switch h := rt.call(m, jsUndef, []interface{}{v}).(type) {
	case bool:
		if h {
			return 1
		}
		return 0
	case float64:
		return pyHashSmall(h)
	case *jsBigInt:
		if h.v.IsInt64() {
			return pyHashFix(uint64(h.v.Int64()))
		}
		return pyHashBig(h.v)
	}
	panic(&jsThrown{value: rt.pyExcInstance("TypeError", "__hash__ method should return an integer")})
}

// pyHashF is a double's hash: an integral one as the int it equals, an infinity
// as +-314159, a NaN by the identity of the value holding it.
func (rt *jsrt) pyHashF(f float64, holder interface{}) uint64 {
	switch {
	case math.IsNaN(f):
		return rt.pyHashId(holder)
	case math.IsInf(f, 1):
		return 314159
	case math.IsInf(f, -1):
		return uint64(math.MaxUint64 - 314159 + 1)
	case f == math.Floor(f) && math.Abs(f) < 1<<53:
		return pyHashSmall(f)
	}
	return pyHashFloat(f)
}

// pyHashInt is a 64-bit hash as the Python int hash() answers.
func (rt *jsrt) pyHashInt(x uint64) interface{} {
	if int64(x) >= -(1<<53) && int64(x) < 1<<53 {
		return float64(int64(x))
	}
	return rt.pyBigNarrow(big.NewInt(int64(x)))
}

// pyHashId is the identity hash: the order in which hash() first met the value.
func (rt *jsrt) pyHashId(v interface{}) uint64 {
	for i, o := range rt.pyHashIds {
		if pySameValue(o, v) {
			return uint64(i + 1)
		}
	}
	rt.pyHashIds = append(rt.pyHashIds, v)
	return uint64(len(rt.pyHashIds))
}

// pySameValue is identity for any two values, functions included, which Go's ==
// on interfaces would panic on.
func pySameValue(a, b interface{}) bool {
	ta, tb := reflect.TypeOf(a), reflect.TypeOf(b)
	if ta != tb {
		return false
	}
	if ta != nil && ta.Kind() == reflect.Func {
		return reflect.ValueOf(a).Pointer() == reflect.ValueOf(b).Pointer()
	}
	if ta != nil && !ta.Comparable() {
		return false
	}
	return a == b
}

// pyBigDec is an int's decimal text as Python shows it. CPython refuses one
// longer than sys.get_int_max_str_digits(). Twin of pyBigDec in
// languages/python-interpreter.abnf and languages/lib/python-rt.metajs.
func (rt *jsrt) pyBigDec(v *big.Int) string {
	s := v.String()
	lim := rt.pyIntMaxDigits()
	if n := len(s); lim > 0 && n > lim && (v.Sign() >= 0 || n-1 > lim) {
		rt.pyRaise("ValueError", "Exceeds the limit ("+strconv.Itoa(lim)+" digits) for integer string conversion; "+
			"use sys.set_int_max_str_digits() to increase the limit")
	}
	return s
}

// pyIntMaxDigits is sys.get_int_max_str_digits(): 4300 until the program sets
// it, 0 for no limit.
func (rt *jsrt) pyIntMaxDigits() int {
	if !rt.pyMaxDigitsSet {
		return 4300
	}
	return rt.pyMaxDigits
}

// pySysCall is a call of one of sys's functions; the module offers
// get_int_max_str_digits, set_int_max_str_digits and exit. Twin of pySysCall in
// languages/python-interpreter.abnf and languages/lib/python-rt.metajs.
func (rt *jsrt) pySysCall(name string, args []interface{}, kw interface{}) interface{} {
	switch name {
	case "get_int_max_str_digits":
		return float64(rt.pyIntMaxDigits())
	case "set_int_max_str_digits":
		v := rt.pyKwArg(kw, "maxdigits")
		if len(args) > 0 {
			v = args[0]
		}
		n := int(rt.pyRangeArg(v))
		if n != 0 && n < 640 {
			rt.pyRaise("ValueError", "maxdigits must be >= 640 or 0 for unlimited")
		}
		rt.pyMaxDigits, rt.pyMaxDigitsSet = n, true
		return jsUndef
	case "exit":
		if len(args) > 1 {
			rt.pyRaise("TypeError", "exit expected at most 1 argument, got "+strconv.Itoa(len(args)))
		}
		inst := newJSObject()
		inst.set("__class", rt.pyExcClass("SystemExit"))
		inst.set("args", pyTup(&jsArray{elems: append([]interface{}{}, args...)}))
		panic(&jsThrown{value: inst})
	}
	panic(&jsThrown{value: rt.pyExcInstance("AttributeError", "module 'sys' has no attribute '"+name+"'")})
}

// pyIntDigitsOver is the digit count of t in base b when CPython's int()
// refuses it for its length under the limit lim (0: none), else 0. Twin of
// pyIntDigitsOver in
// languages/python-interpreter.abnf and languages/lib/python-rt.metajs.
func pyIntDigitsOver(t string, b, lim int) int {
	if lim == 0 || len(t) <= lim || b&(b-1) == 0 {
		return 0
	}
	if n := len(t) - strings.Count(t, "_"); n > lim {
		return n
	}
	return 0
}
