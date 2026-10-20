// Package strconv: Go's strconv package WRITTEN IN GO, compiled with a program
// that imports "strconv" (core.importFile in both go grammars), so one
// implementation serves the interpreter, llvm.Run and a native build alike.
//
// Parsing is Go's own (internal/strconv's atoi.go, atob.go and the exact and
// decimal paths of atof.go/decimal.go, Copyright The Go Authors, BSD-style
// license), restructured only where this project's compiler half lacks a
// construct (a forward goto). Formatting a float goes through fmt, whose verbs
// are strconv's own formats; the shortest form of 'e' and 'f' is rebuilt from the
// shortest digits fmt's %v prints. Quoting is fmt's %q.
package strconv

import (
	"errors"
	"fmt"
	"math"
	"unicode/utf8"
)

const IntSize = 64

var ErrRange = errors.New("value out of range")
var ErrSyntax = errors.New("invalid syntax")

type NumError struct {
	Func string
	Num  string
	Err  error
}

func (e *NumError) Error() string {
	return "strconv." + e.Func + ": " + "parsing " + Quote(e.Num) + ": " + e.Err.Error()
}

func (e *NumError) Unwrap() error { return e.Err }

func numError(fn, s string, err error) error {
	if err == nil {
		return nil
	}
	return &NumError{fn, s, err}
}

// ----- bool -----

func ParseBool(str string) (bool, error) {
	switch str {
	case "1", "t", "T", "true", "TRUE", "True":
		return true, nil
	case "0", "f", "F", "false", "FALSE", "False":
		return false, nil
	}
	return false, &NumError{"ParseBool", str, ErrSyntax}
}

func FormatBool(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func AppendBool(dst []byte, b bool) []byte { return append(dst, FormatBool(b)...) }

// ----- integers -----

func lower(c byte) byte {
	return c | ('x' - 'X')
}

func ParseUint(s string, base int, bitSize int) (uint64, error) {
	n, err := parseUint(s, base, bitSize)
	return n, numError("ParseUint", s, err)
}

func parseUint(s string, base int, bitSize int) (uint64, error) {
	if s == "" {
		return 0, ErrSyntax
	}
	base0 := base == 0
	s0 := s
	switch {
	case 2 <= base && base <= 36:
	case base == 0:
		base = 10
		if s[0] == '0' {
			switch {
			case len(s) >= 3 && lower(s[1]) == 'b':
				base = 2
				s = s[2:]
			case len(s) >= 3 && lower(s[1]) == 'o':
				base = 8
				s = s[2:]
			case len(s) >= 3 && lower(s[1]) == 'x':
				base = 16
				s = s[2:]
			default:
				base = 8
				s = s[1:]
			}
		}
	default:
		return 0, errors.New("invalid base " + Itoa(base))
	}
	if bitSize == 0 {
		bitSize = IntSize
	} else if bitSize < 0 || bitSize > 64 {
		return 0, errors.New("invalid bit size " + Itoa(bitSize))
	}
	const maxUint64 = 1<<64 - 1
	cutoff := uint64(maxUint64)/uint64(base) + 1
	maxVal := uint64(1)<<uint(bitSize) - 1
	underscores := false
	var n uint64
	for _, c := range []byte(s) {
		var d byte
		switch {
		case c == '_' && base0:
			underscores = true
			continue
		case '0' <= c && c <= '9':
			d = c - '0'
		case 'a' <= lower(c) && lower(c) <= 'z':
			d = lower(c) - 'a' + 10
		default:
			return 0, ErrSyntax
		}
		if d >= byte(base) {
			return 0, ErrSyntax
		}
		if n >= cutoff {
			return maxVal, ErrRange
		}
		n *= uint64(base)
		n1 := n + uint64(d)
		if n1 < n || n1 > maxVal {
			return maxVal, ErrRange
		}
		n = n1
	}
	if underscores && !underscoreOK(s0) {
		return 0, ErrSyntax
	}
	return n, nil
}

func ParseInt(s string, base int, bitSize int) (int64, error) {
	n, err := parseInt(s, base, bitSize)
	return n, numError("ParseInt", s, err)
}

func parseInt(s string, base int, bitSize int) (int64, error) {
	if s == "" {
		return 0, ErrSyntax
	}
	neg := false
	switch s[0] {
	case '+':
		s = s[1:]
	case '-':
		s = s[1:]
		neg = true
	}
	un, err := parseUint(s, base, bitSize)
	if err != nil && err != ErrRange {
		return 0, err
	}
	if bitSize == 0 {
		bitSize = IntSize
	}
	cutoff := uint64(1) << uint(bitSize-1)
	if !neg && un >= cutoff {
		return int64(cutoff - 1), ErrRange
	}
	if neg && un > cutoff {
		return -int64(cutoff-1) - 1, ErrRange
	}
	n := int64(un)
	if neg {
		n = -n
	}
	return n, nil
}

func Atoi(s string) (int, error) {
	sLen := len(s)
	if 0 < sLen && sLen < 19 {
		s0 := s
		if s[0] == '-' || s[0] == '+' {
			s = s[1:]
			if len(s) < 1 {
				return 0, &NumError{"Atoi", s0, ErrSyntax}
			}
		}
		n := 0
		for _, ch := range []byte(s) {
			ch -= '0'
			if ch > 9 {
				return 0, &NumError{"Atoi", s0, ErrSyntax}
			}
			n = n*10 + int(ch)
		}
		if s0[0] == '-' {
			n = -n
		}
		return n, nil
	}
	i64, err := parseInt(s, 10, 0)
	return int(i64), numError("Atoi", s, err)
}

func underscoreOK(s string) bool {
	saw := '^'
	i := 0
	if len(s) >= 1 && (s[0] == '-' || s[0] == '+') {
		s = s[1:]
	}
	hex := false
	if len(s) >= 2 && s[0] == '0' && (lower(s[1]) == 'b' || lower(s[1]) == 'o' || lower(s[1]) == 'x') {
		i = 2
		saw = '0'
		hex = lower(s[1]) == 'x'
	}
	for ; i < len(s); i++ {
		if '0' <= s[i] && s[i] <= '9' || hex && 'a' <= lower(s[i]) && lower(s[i]) <= 'f' {
			saw = '0'
			continue
		}
		if s[i] == '_' {
			if saw != '0' {
				return false
			}
			saw = '_'
			continue
		}
		if saw == '_' {
			return false
		}
		saw = '!'
	}
	return saw != '_'
}

const digits = "0123456789abcdefghijklmnopqrstuvwxyz"

func FormatUint(i uint64, base int) string {
	if base < 2 || base > 36 {
		panic("strconv: illegal AppendInt/FormatInt base")
	}
	if i == 0 {
		return "0"
	}
	var buf []byte
	b := uint64(base)
	for i > 0 {
		q := i / b
		buf = append(buf, digits[i-q*b])
		i = q
	}
	for l, r := 0, len(buf)-1; l < r; l, r = l+1, r-1 {
		buf[l], buf[r] = buf[r], buf[l]
	}
	return string(buf)
}

func FormatInt(i int64, base int) string {
	if i < 0 {
		return "-" + FormatUint(uint64(-(i+1))+1, base)
	}
	return FormatUint(uint64(i), base)
}

func Itoa(i int) string { return FormatInt(int64(i), 10) }

func AppendInt(dst []byte, i int64, base int) []byte   { return append(dst, FormatInt(i, base)...) }
func AppendUint(dst []byte, i uint64, base int) []byte { return append(dst, FormatUint(i, base)...) }

// ----- floats: parsing -----

func ParseFloat(s string, bitSize int) (float64, error) {
	f, n, err := atof(s, bitSize == 32)
	if n != len(s) {
		return 0, &NumError{"ParseFloat", s, ErrSyntax}
	}
	return f, numError("ParseFloat", s, err)
}

type floatInfo struct {
	mantbits uint
	expbits  uint
	bias     int
}

var float32info = floatInfo{23, 8, -127}
var float64info = floatInfo{52, 11, -1023}

func commonPrefixLenIgnoreCase(s, prefix string) int {
	n := min(len(prefix), len(s))
	for i := 0; i < n; i++ {
		c := s[i]
		if 'A' <= c && c <= 'Z' {
			c += 'a' - 'A'
		}
		if c != prefix[i] {
			return i
		}
	}
	return n
}

func special(s string) (f float64, n int, ok bool) {
	if len(s) == 0 {
		return 0, 0, false
	}
	sign := 1
	nsign := 0
	c := s[0]
	if c == '+' || c == '-' {
		if c == '-' {
			sign = -1
		}
		nsign = 1
		s = s[1:]
	}
	if c == '+' || c == '-' || c == 'i' || c == 'I' {
		n := commonPrefixLenIgnoreCase(s, "infinity")
		if 3 < n && n < 8 {
			n = 3
		}
		if n == 3 || n == 8 {
			return math.Inf(sign), nsign + n, true
		}
	} else if c == 'n' || c == 'N' {
		if commonPrefixLenIgnoreCase(s, "nan") == 3 {
			return math.NaN(), 3, true
		}
	}
	return 0, 0, false
}

func readFloat(s string) (mantissa uint64, exp int, neg, trunc, hex bool, i int, ok bool) {
	underscores := false
	if i >= len(s) {
		return
	}
	switch s[i] {
	case '+':
		i++
	case '-':
		i++
		neg = true
	}
	base := uint64(10)
	maxMantDigits := 19
	expChar := byte('e')
	if i+2 < len(s) && s[i] == '0' && lower(s[i+1]) == 'x' {
		base = 16
		maxMantDigits = 16
		i += 2
		expChar = 'p'
		hex = true
	}
	sawdot := false
	sawdigits := false
	nd := 0
	ndMant := 0
	dp := 0
	for ; i < len(s); i++ {
		c := s[i]
		if c == '_' {
			underscores = true
			continue
		}
		if c == '.' {
			if sawdot {
				break
			}
			sawdot = true
			dp = nd
			continue
		}
		if '0' <= c && c <= '9' {
			sawdigits = true
			if c == '0' && nd == 0 {
				dp--
				continue
			}
			nd++
			if ndMant < maxMantDigits {
				mantissa *= base
				mantissa += uint64(c - '0')
				ndMant++
			} else if c != '0' {
				trunc = true
			}
			continue
		}
		if base == 16 && 'a' <= lower(c) && lower(c) <= 'f' {
			sawdigits = true
			nd++
			if ndMant < maxMantDigits {
				mantissa *= 16
				mantissa += uint64(lower(c) - 'a' + 10)
				ndMant++
			} else {
				trunc = true
			}
			continue
		}
		break
	}
	if !sawdigits {
		return
	}
	if !sawdot {
		dp = nd
	}
	if base == 16 {
		dp *= 4
		ndMant *= 4
	}
	if i < len(s) && lower(s[i]) == expChar {
		i++
		if i >= len(s) {
			return
		}
		esign := 1
		switch s[i] {
		case '+':
			i++
		case '-':
			i++
			esign = -1
		}
		if i >= len(s) || s[i] < '0' || s[i] > '9' {
			return
		}
		e := 0
		for ; i < len(s) && ('0' <= s[i] && s[i] <= '9' || s[i] == '_'); i++ {
			if s[i] == '_' {
				underscores = true
				continue
			}
			if e < 10000 {
				e = e*10 + int(s[i]) - '0'
			}
		}
		dp += e * esign
	} else if base == 16 {
		return
	}
	if mantissa != 0 {
		exp = dp - ndMant
	}
	if underscores && !underscoreOK(s[:i]) {
		return
	}
	ok = true
	return
}

var float64pow10 = []float64{
	1e0, 1e1, 1e2, 1e3, 1e4, 1e5, 1e6, 1e7, 1e8, 1e9,
	1e10, 1e11, 1e12, 1e13, 1e14, 1e15, 1e16, 1e17, 1e18, 1e19,
	1e20, 1e21, 1e22,
}

// mantissa * 10^exp where float64 arithmetic does it exactly, so the one
// rounding is the correct one.
func atof64exact(mantissa uint64, exp int, neg bool) (f float64, ok bool) {
	if mantissa>>52 != 0 {
		return
	}
	f = float64(mantissa)
	if neg {
		f = -f
	}
	switch {
	case exp == 0:
		return f, true
	case exp > 0 && exp <= 15+22:
		if exp > 22 {
			f *= float64pow10[exp-22]
			exp = 22
		}
		if f > 1e15 || f < -1e15 {
			return
		}
		return f * float64pow10[exp], true
	case exp < 0 && exp >= -22:
		return f / float64pow10[-exp], true
	}
	return
}

func atof(s string, f32 bool) (float64, int, error) {
	if val, n, ok := special(s); ok {
		return val, n, nil
	}
	d, p, neg, trunc, hex, n, ok := readFloat(s)
	if !ok {
		return 0, n, ErrSyntax
	}
	flt := &float64info
	if f32 {
		flt = &float32info
	}
	if hex {
		f, err := atofHex(flt, d, p, neg, trunc)
		return f, n, err
	}
	if !f32 && !trunc {
		if f, ok := atof64exact(d, p, neg); ok {
			return f, n, nil
		}
	}
	var dec decimal
	if !dec.set(s[:n]) {
		return 0, n, ErrSyntax
	}
	b, ovf := dec.floatBits(flt)
	f := fromBits(b, flt)
	if ovf {
		return f, n, ErrRange
	}
	return f, n, nil
}

// The value of IEEE bits b in format flt, as a float64 (exact for float32 too).
func fromBits(b uint64, flt *floatInfo) float64 {
	if flt == &float64info {
		return math.Float64frombits(b)
	}
	neg := b>>31 != 0
	e := int(b>>23) & 0xff
	m := b & (1<<23 - 1)
	var f float64
	switch {
	case e == 0xff && m == 0:
		f = math.Inf(1)
	case e == 0xff:
		f = math.NaN()
	case e == 0:
		f = math.Ldexp(float64(m), -149)
	default:
		f = math.Ldexp(float64(m|1<<23), e-150)
	}
	if neg {
		f = -f
	}
	return f
}

func atofHex(flt *floatInfo, mantissa uint64, exp int, neg, trunc bool) (float64, error) {
	maxExp := 1<<flt.expbits + flt.bias - 2
	minExp := flt.bias + 1
	exp += int(flt.mantbits)
	for mantissa != 0 && mantissa>>(flt.mantbits+2) == 0 {
		mantissa <<= 1
		exp--
	}
	if trunc {
		mantissa |= 1
	}
	for mantissa>>(1+flt.mantbits+2) != 0 {
		mantissa = mantissa>>1 | mantissa&1
		exp++
	}
	for mantissa > 1 && exp < minExp-2 {
		mantissa = mantissa>>1 | mantissa&1
		exp++
	}
	round := mantissa & 3
	mantissa >>= 2
	round |= mantissa & 1
	exp += 2
	if round == 3 {
		mantissa++
		if mantissa == 1<<(1+flt.mantbits) {
			mantissa >>= 1
			exp++
		}
	}
	if mantissa>>flt.mantbits == 0 {
		exp = flt.bias
	}
	var err error
	if exp > maxExp {
		mantissa = 1 << flt.mantbits
		exp = maxExp + 1
		err = ErrRange
	}
	bits := mantissa & (1<<flt.mantbits - 1)
	bits |= uint64((exp-flt.bias)&(1<<flt.expbits-1)) << flt.mantbits
	if neg {
		bits |= 1 << flt.mantbits << flt.expbits
	}
	return fromBits(bits, flt), err
}

// ----- the decimal slow path (internal/strconv/decimal.go) -----

type decimal struct {
	d     [800]byte
	nd    int
	dp    int
	neg   bool
	trunc bool
}

func (b *decimal) set(s string) (ok bool) {
	i := 0
	b.neg = false
	b.trunc = false
	if i >= len(s) {
		return
	}
	switch s[i] {
	case '+':
		i++
	case '-':
		i++
		b.neg = true
	}
	sawdot := false
	sawdigits := false
	for ; i < len(s); i++ {
		if s[i] == '_' {
			continue
		}
		if s[i] == '.' {
			if sawdot {
				return
			}
			sawdot = true
			b.dp = b.nd
			continue
		}
		if '0' <= s[i] && s[i] <= '9' {
			sawdigits = true
			if s[i] == '0' && b.nd == 0 {
				b.dp--
				continue
			}
			if b.nd < len(b.d) {
				b.d[b.nd] = s[i]
				b.nd++
			} else if s[i] != '0' {
				b.trunc = true
			}
			continue
		}
		break
	}
	if !sawdigits {
		return
	}
	if !sawdot {
		b.dp = b.nd
	}
	if i < len(s) && lower(s[i]) == 'e' {
		i++
		if i >= len(s) {
			return
		}
		esign := 1
		switch s[i] {
		case '+':
			i++
		case '-':
			i++
			esign = -1
		}
		if i >= len(s) || s[i] < '0' || s[i] > '9' {
			return
		}
		e := 0
		for ; i < len(s) && ('0' <= s[i] && s[i] <= '9' || s[i] == '_'); i++ {
			if s[i] == '_' {
				continue
			}
			if e < 10000 {
				e = e*10 + int(s[i]) - '0'
			}
		}
		b.dp += e * esign
	}
	if i != len(s) {
		return
	}
	ok = true
	return
}

var powtab = []int{1, 3, 6, 9, 13, 16, 19, 23, 26}

// Go's floatBits, its two `goto`s folded into the flags they jump on.
func (d *decimal) floatBits(flt *floatInfo) (b uint64, overflow bool) {
	var exp int
	var mant uint64
	zero := false
	if d.nd == 0 || d.dp < -330 {
		zero = true
	} else if d.dp > 310 {
		overflow = true
	}
	if !zero && !overflow {
		exp = 0
		for d.dp > 0 {
			var n int
			if d.dp >= len(powtab) {
				n = 27
			} else {
				n = powtab[d.dp]
			}
			d.Shift(-n)
			exp += n
		}
		for d.dp < 0 || d.dp == 0 && d.d[0] < '5' {
			var n int
			if -d.dp >= len(powtab) {
				n = 27
			} else {
				n = powtab[-d.dp]
			}
			d.Shift(n)
			exp -= n
		}
		exp--
		if exp < flt.bias+1 {
			n := flt.bias + 1 - exp
			d.Shift(-n)
			exp += n
		}
		if exp-flt.bias >= 1<<flt.expbits-1 {
			overflow = true
		} else {
			d.Shift(int(1 + flt.mantbits))
			mant = d.RoundedInteger()
			if mant == 2<<flt.mantbits {
				mant >>= 1
				exp++
				if exp-flt.bias >= 1<<flt.expbits-1 {
					overflow = true
				}
			}
			if !overflow && mant&(1<<flt.mantbits) == 0 {
				exp = flt.bias
			}
		}
	}
	if zero {
		mant = 0
		exp = flt.bias
	}
	if overflow {
		mant = 0
		exp = 1<<flt.expbits - 1 + flt.bias
	}
	bits := mant & (uint64(1)<<flt.mantbits - 1)
	bits |= uint64((exp-flt.bias)&(1<<flt.expbits-1)) << flt.mantbits
	if d.neg {
		bits |= 1 << flt.mantbits << flt.expbits
	}
	return bits, overflow
}

func trim(a *decimal) {
	for a.nd > 0 && a.d[a.nd-1] == '0' {
		a.nd--
	}
	if a.nd == 0 {
		a.dp = 0
	}
}

const maxShift = 60

func rightShift(a *decimal, k uint) {
	r := 0
	w := 0
	var n uint
	for ; n>>k == 0; r++ {
		if r >= a.nd {
			if n == 0 {
				a.nd = 0
				return
			}
			for n>>k == 0 {
				n = n * 10
				r++
			}
			break
		}
		c := uint(a.d[r])
		n = n*10 + c - '0'
	}
	a.dp -= r - 1
	var mask uint = (1 << k) - 1
	for ; r < a.nd; r++ {
		c := uint(a.d[r])
		dig := n >> k
		n &= mask
		a.d[w] = byte(dig + '0')
		w++
		n = n*10 + c - '0'
	}
	for n > 0 {
		dig := n >> k
		n &= mask
		if w < len(a.d) {
			a.d[w] = byte(dig + '0')
			w++
		} else if dig > 0 {
			a.trunc = true
		}
		n = n * 10
	}
	a.nd = w
	trim(a)
}

// leftcheats[k]: how many digits a left shift by k adds, one fewer when the
// number is below the cutoff (5^k).
type leftCheat struct {
	delta  int
	cutoff string
}

var leftcheats = []leftCheat{
	{0, ""},
	{1, "5"},
	{1, "25"},
	{1, "125"},
	{2, "625"},
	{2, "3125"},
	{2, "15625"},
	{3, "78125"},
	{3, "390625"},
	{3, "1953125"},
	{4, "9765625"},
	{4, "48828125"},
	{4, "244140625"},
	{4, "1220703125"},
	{5, "6103515625"},
	{5, "30517578125"},
	{5, "152587890625"},
	{6, "762939453125"},
	{6, "3814697265625"},
	{6, "19073486328125"},
	{7, "95367431640625"},
	{7, "476837158203125"},
	{7, "2384185791015625"},
	{7, "11920928955078125"},
	{8, "59604644775390625"},
	{8, "298023223876953125"},
	{8, "1490116119384765625"},
	{9, "7450580596923828125"},
	{9, "37252902984619140625"},
	{9, "186264514923095703125"},
	{10, "931322574615478515625"},
	{10, "4656612873077392578125"},
	{10, "23283064365386962890625"},
	{10, "116415321826934814453125"},
	{11, "582076609134674072265625"},
	{11, "2910383045673370361328125"},
	{11, "14551915228366851806640625"},
	{12, "72759576141834259033203125"},
	{12, "363797880709171295166015625"},
	{12, "1818989403545856475830078125"},
	{13, "9094947017729282379150390625"},
	{13, "45474735088646411895751953125"},
	{13, "227373675443232059478759765625"},
	{13, "1136868377216160297393798828125"},
	{14, "5684341886080801486968994140625"},
	{14, "28421709430404007434844970703125"},
	{14, "142108547152020037174224853515625"},
	{15, "710542735760100185871124267578125"},
	{15, "3552713678800500929355621337890625"},
	{15, "17763568394002504646778106689453125"},
	{16, "88817841970012523233890533447265625"},
	{16, "444089209850062616169452667236328125"},
	{16, "2220446049250313080847263336181640625"},
	{16, "11102230246251565404236316680908203125"},
	{17, "55511151231257827021181583404541015625"},
	{17, "277555756156289135105907917022705078125"},
	{17, "1387778780781445675529539585113525390625"},
	{18, "6938893903907228377647697925567626953125"},
	{18, "34694469519536141888238489627838134765625"},
	{18, "173472347597680709441192448139190673828125"},
	{19, "867361737988403547205962240695953369140625"},
}

func prefixIsLessThan(b []byte, s string) bool {
	for i := 0; i < len(s); i++ {
		if i >= len(b) {
			return true
		}
		if b[i] != s[i] {
			return b[i] < s[i]
		}
	}
	return false
}

func leftShift(a *decimal, k uint) {
	delta := leftcheats[k].delta
	if prefixIsLessThan(a.d[0:a.nd], leftcheats[k].cutoff) {
		delta--
	}
	r := a.nd
	w := a.nd + delta
	var n uint
	for r--; r >= 0; r-- {
		n += (uint(a.d[r]) - '0') << k
		quo := n / 10
		rem := n - 10*quo
		w--
		if w < len(a.d) {
			a.d[w] = byte(rem + '0')
		} else if rem != 0 {
			a.trunc = true
		}
		n = quo
	}
	for n > 0 {
		quo := n / 10
		rem := n - 10*quo
		w--
		if w < len(a.d) {
			a.d[w] = byte(rem + '0')
		} else if rem != 0 {
			a.trunc = true
		}
		n = quo
	}
	a.nd += delta
	if a.nd >= len(a.d) {
		a.nd = len(a.d)
	}
	a.dp += delta
	trim(a)
}

func (a *decimal) Shift(k int) {
	switch {
	case a.nd == 0:
	case k > 0:
		for k > maxShift {
			leftShift(a, maxShift)
			k -= maxShift
		}
		leftShift(a, uint(k))
	case k < 0:
		for k < -maxShift {
			rightShift(a, maxShift)
			k += maxShift
		}
		rightShift(a, uint(-k))
	}
}

func shouldRoundUp(a *decimal, nd int) bool {
	if nd < 0 || nd >= a.nd {
		return false
	}
	if a.d[nd] == '5' && nd+1 == a.nd {
		if a.trunc {
			return true
		}
		return nd > 0 && (a.d[nd-1]-'0')%2 != 0
	}
	return a.d[nd] >= '5'
}

func (a *decimal) RoundedInteger() uint64 {
	if a.dp > 20 {
		return 0xFFFFFFFFFFFFFFFF
	}
	var i int
	n := uint64(0)
	for i = 0; i < a.dp && i < a.nd; i++ {
		n = n*10 + uint64(a.d[i]-'0')
	}
	for ; i < a.dp; i++ {
		n *= 10
	}
	if shouldRoundUp(a, a.dp) {
		n++
	}
	return n
}

// ----- floats: formatting -----

func FormatFloat(f float64, format byte, prec, bitSize int) string {
	var v any = f
	if bitSize == 32 {
		v = float32(f)
	}
	if f != f || f > math.MaxFloat64 || f < -math.MaxFloat64 {
		return fmt.Sprint(v)
	}
	switch format {
	case 'e', 'E', 'f', 'g', 'G', 'x', 'X':
	case 'b':
		return fmtB(f)
	default:
		return "%" + string(format)
	}
	if prec >= 0 {
		return fmt.Sprintf("%."+Itoa(prec)+string(format), v)
	}
	if format == 'x' || format == 'X' {
		return fmt.Sprintf("%"+string(format), v)
	}
	s := fmt.Sprint(v)
	if format == 'g' {
		return s
	}
	if format == 'G' {
		return upperE(s)
	}
	sign := ""
	if s[0] == '-' {
		sign = "-"
		s = s[1:]
	}
	d, dp := shortestDigits(s)
	if format == 'f' {
		return sign + fixed(d, dp)
	}
	out := sign + d[:1]
	if len(d) > 1 {
		out += "." + d[1:]
	}
	exp := dp - 1
	if d == "0" {
		exp = 0
	}
	es := "+"
	if exp < 0 {
		es = "-"
		exp = -exp
	}
	if exp < 10 {
		es += "0"
	}
	out += "e" + es + Itoa(exp)
	if format == 'E' {
		return upperE(out)
	}
	return out
}

func AppendFloat(dst []byte, f float64, format byte, prec, bitSize int) []byte {
	return append(dst, FormatFloat(f, format, prec, bitSize)...)
}

func upperE(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] == 'e' {
			b[i] = 'E'
		}
	}
	return string(b)
}

// The digits of fmt's shortest %v text s (no sign) and the position of the
// decimal point: s = 0.digits * 10^dp.
func shortestDigits(s string) (string, int) {
	mant := s
	exp := 0
	for i := 0; i < len(s); i++ {
		if s[i] == 'e' {
			mant = s[:i]
			e, _ := parseInt(s[i+1:], 10, 0)
			exp = int(e)
			break
		}
	}
	ds := ""
	dp := -1
	for i := 0; i < len(mant); i++ {
		if mant[i] == '.' {
			dp = len(ds)
		} else {
			ds += mant[i : i+1]
		}
	}
	if dp < 0 {
		dp = len(ds)
	}
	lead := 0
	for lead < len(ds)-1 && ds[lead] == '0' {
		lead++
	}
	ds = ds[lead:]
	dp = dp - lead + exp
	end := len(ds)
	for end > 1 && ds[end-1] == '0' {
		end--
	}
	return ds[:end], dp
}

func fixed(d string, dp int) string {
	if d == "0" {
		return "0"
	}
	if dp <= 0 {
		z := ""
		for i := 0; i < -dp; i++ {
			z += "0"
		}
		return "0." + z + d
	}
	if dp >= len(d) {
		for len(d) < dp {
			d += "0"
		}
		return d
	}
	return d[:dp] + "." + d[dp:]
}

// The 'b' format: mantissa "p" binary exponent.
func fmtB(f float64) string {
	bits := math.Float64bits(f)
	sign := ""
	if bits>>63 != 0 {
		sign = "-"
	}
	exp := int(bits>>52) & 0x7ff
	mant := bits & (1<<52 - 1)
	if exp == 0 {
		exp++
	} else {
		mant |= 1 << 52
	}
	exp -= 1075
	es := "+"
	if exp < 0 {
		es = "-"
		exp = -exp
	}
	return sign + FormatUint(mant, 10) + "p" + es + Itoa(exp)
}

// ----- quoting -----

func Quote(s string) string                   { return fmt.Sprintf("%q", s) }
func QuoteToASCII(s string) string            { return fmt.Sprintf("%+q", s) }
func QuoteRune(r rune) string                 { return fmt.Sprintf("%q", r) }
func QuoteRuneToASCII(r rune) string          { return fmt.Sprintf("%+q", r) }
func AppendQuote(dst []byte, s string) []byte { return append(dst, Quote(s)...) }

// CanBackquote: s is valid UTF-8 and has no control character but a tab, no `
// and no byte order mark, so it can be a raw string literal.
func CanBackquote(s string) bool {
	for len(s) > 0 {
		r, wid := utf8.DecodeRuneInString(s)
		s = s[wid:]
		if wid > 1 {
			if r == '\ufeff' {
				return false
			}
			continue
		}
		if r == utf8.RuneError {
			return false
		}
		if (r < ' ' && r != '\t') || r == '`' || r == '\u007F' {
			return false
		}
	}
	return true
}

func unhex(b byte) (v rune, ok bool) {
	c := rune(b)
	switch {
	case '0' <= c && c <= '9':
		return c - '0', true
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10, true
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10, true
	}
	return
}

// UnquoteChar decodes the first character or escape of s, quoted by quote.
func UnquoteChar(s string, quote byte) (value rune, multibyte bool, tail string, err error) {
	if len(s) == 0 {
		err = ErrSyntax
		return
	}
	c := s[0]
	if c == quote && (quote == '\'' || quote == '"') {
		err = ErrSyntax
		return
	}
	if c >= 0x80 {
		for _, r := range s {
			return r, true, s[len(string(r)):], nil
		}
	}
	if c != '\\' {
		return rune(s[0]), false, s[1:], nil
	}
	if len(s) <= 1 {
		err = ErrSyntax
		return
	}
	c = s[1]
	s = s[2:]
	switch c {
	case 'a':
		value = '\a'
	case 'b':
		value = '\b'
	case 'f':
		value = '\f'
	case 'n':
		value = '\n'
	case 'r':
		value = '\r'
	case 't':
		value = '\t'
	case 'v':
		value = '\v'
	case 'x', 'u', 'U':
		n := 2
		if c == 'u' {
			n = 4
		} else if c == 'U' {
			n = 8
		}
		var v rune
		if len(s) < n {
			err = ErrSyntax
			return
		}
		for j := 0; j < n; j++ {
			x, ok := unhex(s[j])
			if !ok {
				err = ErrSyntax
				return
			}
			v = v<<4 | x
		}
		s = s[n:]
		if c != 'x' {
			if v > 0x10FFFF || (v >= 0xD800 && v <= 0xDFFF) {
				err = ErrSyntax
				return
			}
			multibyte = true
		}
		value = v
	case '0', '1', '2', '3', '4', '5', '6', '7':
		v := rune(c) - '0'
		if len(s) < 2 {
			err = ErrSyntax
			return
		}
		for j := 0; j < 2; j++ {
			x := rune(s[j]) - '0'
			if x < 0 || x > 7 {
				err = ErrSyntax
				return
			}
			v = (v << 3) | x
		}
		s = s[2:]
		if v > 255 {
			err = ErrSyntax
			return
		}
		value = v
	case '\\':
		value = '\\'
	case '\'', '"':
		if c != quote {
			err = ErrSyntax
			return
		}
		value = rune(c)
	default:
		err = ErrSyntax
		return
	}
	tail = s
	return
}

func Unquote(s string) (string, error) {
	n := len(s)
	if n < 2 {
		return "", ErrSyntax
	}
	quote := s[0]
	if quote != s[n-1] {
		return "", ErrSyntax
	}
	s = s[1 : n-1]
	if quote == '`' {
		for i := 0; i < len(s); i++ {
			if s[i] == '`' {
				return "", ErrSyntax
			}
		}
		out := []byte{}
		for i := 0; i < len(s); i++ {
			if s[i] != '\r' {
				out = append(out, s[i])
			}
		}
		return string(out), nil
	}
	if quote != '"' && quote != '\'' {
		return "", ErrSyntax
	}
	var buf []byte
	runes := 0
	for len(s) > 0 {
		if s[0] == '\n' {
			return "", ErrSyntax
		}
		c, multibyte, ss, err := UnquoteChar(s, quote)
		if err != nil {
			return "", err
		}
		s = ss
		if c < 0x80 || !multibyte {
			buf = append(buf, byte(c))
		} else {
			buf = append(buf, string(c)...)
		}
		runes++
		if quote == '\'' && len(s) != 0 {
			return "", ErrSyntax
		}
	}
	// Go accepts an empty '' (its fast path finds nothing to reject).
	return string(buf), nil
}
