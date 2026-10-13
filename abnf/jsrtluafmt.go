package abnf

// jsrtluafmt.go -- string.format for the compiler grammar
// (languages/lua-to-llvm-ir.abnf): a line-by-line port of lsFormat in
// languages/lib/lua-strlib.js, which the interpreter half includes and layer 2
// imports. It formats as lua 5.5 on this platform does - lstrlib.c over macOS
// libc's printf - so every rule here is the shared file's; keep the two in step.

import (
	"math"
	"math/big"
	"strconv"
	"strings"
)

const (
	lsFlagsF = "-+ #0" // a A e E f g G
	lsFlagsX = "-#0"   // o x X
	lsFlagsI = "-+ 0"  // d i
	lsFlagsU = "-0"    // u
	lsFlagsC = "-"     // c s
	lsHex    = "0123456789abcdef"
)

type lsSpecT struct {
	left, plus, space, alt, zero bool
	width, prec                  int
}

func lsFormCh(form string, i int) byte {
	if i < len(form) {
		return form[i]
	}
	return 0
}

func lsGet2Digits(form string, i int) int {
	if i < len(form) && lsIsDigit(int(form[i])) {
		i++
		if i < len(form) && lsIsDigit(int(form[i])) {
			i++
		}
	}
	return i
}

func (rt *jsrt) lsCheckFormat(form, flags string, precision bool) {
	i := 1
	for i < len(form) && strings.IndexByte(flags, form[i]) >= 0 {
		i++
	}
	if lsFormCh(form, i) != '0' {
		i = lsGet2Digits(form, i)
		if lsFormCh(form, i) == '.' && precision {
			i = lsGet2Digits(form, i+1)
		}
	}
	if !lsIsAlpha(int(lsFormCh(form, i))) {
		rt.luFail("%s", "invalid conversion specification: '"+form+"'")
	}
}

func lsSpec(form string) lsSpecT {
	sp := lsSpecT{prec: -1}
	i := 1
loop:
	for ; i < len(form); i++ {
		switch form[i] {
		case '-':
			sp.left = true
		case '+':
			sp.plus = true
		case ' ':
			sp.space = true
		case '#':
			sp.alt = true
		case '0':
			sp.zero = true
		default:
			break loop
		}
	}
	for i < len(form) && lsIsDigit(int(form[i])) {
		sp.width = sp.width*10 + int(form[i]-'0')
		i++
	}
	if i < len(form) && form[i] == '.' {
		i++
		sp.prec = 0
		for i < len(form) && lsIsDigit(int(form[i])) {
			sp.prec = sp.prec*10 + int(form[i]-'0')
			i++
		}
	}
	return sp
}

func lsPad(sp lsSpecT, sign, body string, zeroOk bool) string {
	n := len(sign) + len(body)
	if n >= sp.width {
		return sign + body
	}
	ch := " "
	if sp.zero && zeroOk && !sp.left {
		ch = "0"
	}
	fill := strings.Repeat(ch, sp.width-n)
	if sp.left {
		return sign + body + fill
	}
	if sp.zero && zeroOk {
		return sign + fill + body
	}
	return fill + sign + body
}

func lsZeros(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat("0", n)
}

type lsIntDigitsT struct {
	neg        bool
	d, u, x, o string
}

func lsIntDigits(v int64) lsIntDigitsT {
	u := uint64(v)
	mag := u
	if v < 0 {
		mag = -u
	}
	return lsIntDigitsT{neg: v < 0, d: strconv.FormatUint(mag, 10), u: strconv.FormatUint(u, 10),
		x: strconv.FormatUint(u, 16), o: strconv.FormatUint(u, 8)}
}

func lsFmtIntConv(sp lsSpecT, conv byte, iv lsIntDigitsT) string {
	digits, sign := iv.d, ""
	switch {
	case conv == 'u':
		digits = iv.u
	case conv == 'o':
		digits = iv.o
	case conv == 'x':
		digits = iv.x
	case conv == 'X':
		digits = strings.ToUpper(iv.x)
	case iv.neg:
		sign = "-"
	case sp.plus:
		sign = "+"
	case sp.space:
		sign = " "
	}
	zero := digits == "0"
	if sp.prec >= 0 {
		if sp.prec == 0 && zero {
			digits = ""
		}
		digits = lsZeros(sp.prec-len(digits)) + digits
	}
	if sp.alt && conv == 'o' && (len(digits) == 0 || digits[0] != '0') {
		digits = "0" + digits
	}
	if sp.alt && !zero && conv == 'x' {
		sign = "0x"
	}
	if sp.alt && !zero && conv == 'X' {
		sign = "0X"
	}
	return lsPad(sp, sign, digits, sp.prec < 0)
}

// lsExactDec is the exact decimal of a finite a >= 0 as its integer and
// fraction digits: a double is m * 2^e, so for e < 0 its decimal is m * 5^-e
// with the point -e digits from the right.
func lsExactDec(a float64) (string, string) {
	if a == 0 {
		return "0", ""
	}
	frac, exp := math.Frexp(a)
	m := int64(frac * (1 << 53))
	e := exp - 53
	for m != 0 && m%2 == 0 {
		m /= 2
		e++
	}
	n := new(big.Int).SetInt64(m)
	if e >= 0 {
		return n.Lsh(n, uint(e)).String(), ""
	}
	n.Mul(n, new(big.Int).Exp(big.NewInt(5), big.NewInt(int64(-e)), nil))
	all := n.String()
	k := -e
	if len(all) <= k {
		all = lsZeros(k-len(all)+1) + all
	}
	return all[:len(all)-k], all[len(all)-k:]
}

// lsRoundHalfEven is the digit string rounded to keep digits, half to even -
// one digit longer when the carry ran out (999 -> 1000). Twin of
// rtRoundHalfEven in languages/lib/runtime.metajs.
func lsRoundHalfEven(digits string, keep int) string {
	if keep >= len(digits) {
		return digits + lsZeros(keep-len(digits))
	}
	head := []byte(digits[:keep])
	d := digits[keep] - '0'
	up := d > 5
	if d == 5 {
		if strings.TrimRight(digits[keep+1:], "0") != "" {
			up = true
		} else if keep > 0 {
			up = (head[keep-1]-'0')%2 == 1
		}
	}
	if !up {
		if len(head) == 0 {
			return "0"
		}
		return string(head)
	}
	i := len(head) - 1
	for ; i >= 0; i-- {
		if head[i] == '9' {
			head[i] = '0'
			continue
		}
		head[i]++
		break
	}
	if i < 0 {
		return "1" + string(head)
	}
	return string(head)
}

func lsSigDigits(a float64, n int) (string, int) {
	ip, fp := lsExactDec(a)
	all := ip + fp
	e := len(ip) - 1
	lead := 0
	for lead < len(all) && all[lead] == '0' {
		lead++
	}
	e -= lead
	ds := lsRoundHalfEven(all[lead:], n)
	if len(ds) > n {
		ds = ds[:n]
		e++
	}
	return ds, e
}

func lsExpText(e int) string {
	sign := "+"
	if e < 0 {
		sign, e = "-", -e
	}
	s := strconv.Itoa(e)
	if len(s) < 2 {
		s = "0" + s
	}
	return sign + s
}

func lsFmtE(a float64, prec int, alt bool) string {
	ds, e := lsZeros(prec+1), 0
	if a != 0 {
		ds, e = lsSigDigits(a, prec+1)
	}
	body := ds[:1]
	if prec > 0 || alt {
		body += "." + ds[1:]
	}
	return body + "e" + lsExpText(e)
}

func lsFmtF(a float64, prec int, alt bool) string {
	ip, fp := lsExactDec(a)
	kept := lsRoundHalfEven(ip+fp, len(ip)+prec)
	ilen := len(ip)
	if len(kept) > len(ip)+prec {
		ilen++
	}
	ints := kept[:ilen]
	if ints == "" {
		ints = "0"
	}
	if prec > 0 {
		return ints + "." + kept[ilen:]
	}
	if alt {
		return ints + "."
	}
	return ints
}

func lsFmtG(a float64, prec int, alt bool) string {
	P := prec
	if prec < 0 {
		P = 6
	} else if prec == 0 {
		P = 1
	}
	x := 0
	if a != 0 {
		_, x = lsSigDigits(a, P)
	}
	var s string
	if x < -4 || x >= P {
		s = lsFmtE(a, P-1, alt)
	} else {
		s = lsFmtF(a, P-1-x, alt)
	}
	if alt {
		return s
	}
	mant, tail := s, ""
	if ei := strings.IndexByte(s, 'e'); ei >= 0 {
		mant, tail = s[:ei], s[ei:]
	}
	if strings.IndexByte(mant, '.') >= 0 {
		mant = strings.TrimRight(mant, "0")
		mant = strings.TrimSuffix(mant, ".")
	}
	return mant + tail
}

// lsFmtA is %a of a finite a >= 0: normalized, and with a precision the hex
// digits are cut and rounded as lsFmtA in languages/lib/lua-strlib.js says
// (this platform's libc), a carry making the lead digit 2.
func lsFmtA(a float64, prec int, alt bool) string {
	lead, ds, e := byte('1'), "", 0
	if a == 0 {
		lead = '0'
	} else {
		m := a
		for m >= 2 {
			m /= 2
			e++
		}
		for m < 1 {
			m *= 2
			e--
		}
		f := m - 1
		for f > 0 {
			f *= 16
			d := math.Floor(f)
			ds += string(lsHex[int(d)])
			f -= d
		}
	}
	if prec >= 0 {
		if len(ds) > prec {
			next := strings.IndexByte(lsHex, ds[prec]) * 16
			if prec+1 < len(ds) {
				next += strings.IndexByte(lsHex, ds[prec+1])
			}
			up := next >= 144 || (next >= 128 && next%2 == 1)
			b := []byte(ds[:prec])
			if up {
				i := len(b) - 1
				for ; i >= 0; i-- {
					if b[i] == 'f' {
						b[i] = '0'
						continue
					}
					b[i] = lsHex[strings.IndexByte(lsHex, b[i])+1]
					break
				}
				if i < 0 {
					if lead == '0' {
						lead = '1'
					} else {
						lead = '2'
					}
				}
			}
			ds = string(b)
		}
		ds += lsZeros(prec - len(ds))
	}
	body := "0x" + string(lead)
	if len(ds) > 0 || alt {
		body += "." + ds
	}
	if e < 0 {
		return body + "p-" + strconv.Itoa(-e)
	}
	return body + "p+" + strconv.Itoa(e)
}

func lsFmtFloat(sp lsSpecT, conv byte, v float64) string {
	lowc := conv | 0x20
	neg := v < 0 || (v == 0 && math.Signbit(v))
	a := math.Abs(v)
	sign := ""
	switch {
	case v != v:
	case neg:
		sign = "-"
	case sp.plus:
		sign = "+"
	case sp.space:
		sign = " "
	}
	finite := !math.IsNaN(v) && !math.IsInf(v, 0)
	prec6 := sp.prec
	if prec6 < 0 {
		prec6 = 6
	}
	var body string
	switch {
	case !finite && v != v:
		body = "nan"
	case !finite:
		body = "inf"
	case lowc == 'e':
		body = lsFmtE(a, prec6, sp.alt)
	case lowc == 'f':
		body = lsFmtF(a, prec6, sp.alt)
	case lowc == 'g':
		body = lsFmtG(a, sp.prec, sp.alt)
	default:
		body = lsFmtA(a, sp.prec, sp.alt)
	}
	if conv < 'a' {
		body = strings.ToUpper(body)
	}
	if finite && lowc == 'a' && sp.zero && !sp.left {
		return sign + body[:2] + lsZeros(sp.width-len(sign)-len(body)) + body[2:]
	}
	return lsPad(sp, sign, body, finite)
}

func lsQuoted(s []uint16) string {
	var b strings.Builder
	b.WriteByte('"')
	var out []uint16
	for i, c := range s {
		switch {
		case c == '"' || c == '\\' || c == '\n':
			out = append(out, '\\', c)
		case lsIsCntrl(int(c)):
			ds := strconv.Itoa(int(c))
			if i+1 < len(s) && lsIsDigit(int(s[i+1])) {
				ds = lsZeros(3-len(ds)) + ds
			}
			out = append(out, strUnits("\\"+ds)...)
		default:
			out = append(out, c)
		}
	}
	b.WriteString(strFromUnits(out))
	b.WriteByte('"')
	return b.String()
}

// luFmtInt is luaL_checkinteger of argument n as an int64.
func (rt *jsrt) luFmtInt(args []interface{}, n int) int64 {
	rt.luCheckInt("string.format", n, args)
	v, _ := luToNum(args[n-1])
	return luIntOf(v)
}

func (rt *jsrt) luFmtToStr(v interface{}) string {
	switch x := v.(type) {
	case string:
		return x
	case bool:
		if x {
			return "true"
		}
		return "false"
	}
	if luIsNil(v) {
		return "nil"
	}
	if luIsNum(v) {
		return luNumStr(v)
	}
	return rt.luObjStr(v)
}

// luIdents numbers the tables and functions in first-use order, and luIdObjs[n-1]
// is object n. The number is both how luIdent renders the object and its identity
// as a table key (luKey). One compile runs one program, so package-level state is
// per program.
var luIdents = map[interface{}]int{}
var luIdObjs []interface{}

// luObjId is the identity number of a table or a function. Twin of luObjId in
// languages/lib/lua-rt.metajs and luaObjId in lua-interpreter.abnf.
func luObjId(v interface{}) int {
	n, ok := luIdents[v]
	if !ok {
		luIdObjs = append(luIdObjs, v)
		n = len(luIdObjs)
		luIdents[v] = n
	}
	return n
}

// luIdent is how a table or a function renders: its type and an identity,
// `table: 0x...`. Real lua prints the address, which differs between runs, so
// what has to hold is that one object renders the same way every time and two
// objects differently; a per-run counter gives that. Twin of js_luident in
// languages/lib/lua-rt.metajs.
func (rt *jsrt) luIdent(v interface{}) string {
	n := luObjId(v)
	hex := strconv.FormatInt(int64(0x55550000+16*n), 16)
	if luIsFile(v) {
		return "file (0x" + hex + ")"
	}
	return rt.luTypeName(v) + ": 0x" + hex
}

func (rt *jsrt) lsLiteral(args []interface{}, n int) string {
	v := args[n-1]
	if s, ok := v.(string); ok {
		return lsQuoted(strUnits(s))
	}
	if luIsNum(v) {
		if luIsInt(v) {
			iv := lsIntDigits(rt.luFmtInt(args, n))
			if iv.neg && iv.x == "8000000000000000" {
				return "0x8000000000000000"
			}
			if iv.neg {
				return "-" + iv.d
			}
			return iv.d
		}
		f := luNumOf(v)
		switch {
		case math.IsNaN(f):
			return "(0/0)"
		case math.IsInf(f, 1):
			return "1e9999"
		case math.IsInf(f, -1):
			return "-1e9999"
		}
		if f < 0 || (f == 0 && math.Signbit(f)) {
			return "-" + lsFmtA(-f, -1, false)
		}
		return lsFmtA(f, -1, false)
	}
	if luIsNil(v) {
		return "nil"
	}
	if _, ok := v.(bool); ok {
		return rt.luFmtToStr(v)
	}
	rt.luArgErr("string.format", n, "value has no literal form")
	return ""
}

// luFormat is string.format over the call's arguments.
func (rt *jsrt) luFormat(args []interface{}) string {
	fmtS := strUnits(rt.luCheckStr("string.format", 1, args))
	var out []uint16
	arg := 1
	for i := 0; i < len(fmtS); {
		c := fmtS[i]
		if c != '%' {
			out = append(out, c)
			i++
			continue
		}
		i++
		if i < len(fmtS) && fmtS[i] == '%' {
			out = append(out, '%')
			i++
			continue
		}
		arg++
		if arg > len(args) {
			rt.luArgErr("string.format", arg, "no value")
		}
		j := i
		for j < len(fmtS) && fmtS[j] < 128 && strings.IndexByte("-+ #0123456789.", byte(fmtS[j])) >= 0 {
			j++
		}
		if j-i+1 >= 22 {
			rt.luFail("invalid format string to 'format'")
		}
		end := j + 1
		if end > len(fmtS) {
			end = len(fmtS)
		}
		form := "%" + strFromUnits(fmtS[i:end])
		var conv byte
		if j < len(fmtS) && fmtS[j] < 128 {
			conv = byte(fmtS[j])
		}
		i = j + 1
		switch conv {
		case 'c':
			v := rt.luFmtInt(args, arg)
			rt.lsCheckFormat(form, lsFlagsC, false)
			sp := lsSpec(form)
			ch := []uint16{uint16(uint8(v))}
			if sp.width > 1 {
				pad := strUnits(strings.Repeat(" ", sp.width-1))
				if sp.left {
					ch = append(ch, pad...)
				} else {
					ch = append(pad, ch...)
				}
			}
			out = append(out, ch...)
		case 'd', 'i', 'u', 'o', 'x', 'X':
			v := rt.luFmtInt(args, arg)
			fl := lsFlagsX
			if conv == 'd' || conv == 'i' {
				fl = lsFlagsI
			} else if conv == 'u' {
				fl = lsFlagsU
			}
			rt.lsCheckFormat(form, fl, true)
			out = append(out, strUnits(lsFmtIntConv(lsSpec(form), conv, lsIntDigits(v)))...)
		case 'a', 'A', 'e', 'E', 'f', 'g', 'G':
			v := luNumOf(rt.luCheckNum("string.format", arg, args))
			rt.lsCheckFormat(form, lsFlagsF, true)
			out = append(out, strUnits(lsFmtFloat(lsSpec(form), conv, v))...)
		case 'q':
			if len(form) > 2 {
				rt.luFail("specifier '%%q' cannot have modifiers")
			}
			out = append(out, strUnits(rt.lsLiteral(args, arg))...)
		case 's':
			s := strUnits(rt.luFmtToStr(args[arg-1]))
			if len(form) == 2 {
				out = append(out, s...)
				break
			}
			// A modifier goes through C's %s, which would stop at a NUL.
			for _, cu := range s {
				if cu == 0 {
					rt.luArgErr("string.format", arg, "string contains zeros")
				}
			}
			rt.lsCheckFormat(form, lsFlagsC, true)
			sp := lsSpec(form)
			if sp.prec < 0 && len(s) >= 100 {
				out = append(out, s...)
				break
			}
			if sp.prec >= 0 && len(s) > sp.prec {
				s = s[:sp.prec]
			}
			n := len(s)
			if n < sp.width {
				fill := strUnits(strings.Repeat(" ", sp.width-n))
				if sp.left {
					s = append(append([]uint16{}, s...), fill...)
				} else {
					s = append(fill, s...)
				}
			}
			out = append(out, s...)
		default:
			rt.luFail("%s", "invalid conversion '"+form+"' to 'format'")
		}
	}
	return strFromUnits(out)
}

func init() {
	rxExtraExterns = append(rxExtraExterns, func(rt *jsrt, m map[string]func(args []uint64) uint64) {
		m["js_luformat"] = func(a []uint64) uint64 {
			var args []interface{}
			if arr, ok := rt.unwrap(a[0]).(*jsArray); ok {
				args = arr.elems
			}
			return rt.wrapStr(rt.luFormat(args))
		}
	})
}
