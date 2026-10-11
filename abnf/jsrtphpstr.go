package abnf

import (
	"math"
	"strconv"
	"strings"
	"unicode/utf8"
)

// PHP's byte-string primitives and the natives lib/php-stdlib.php is written on -
// the GO TWIN of languages/lib/php-strings.js (shared by the interpreter half and
// layer 2) plus the js_ph* externs that wrap it. A PHP string's text is held in
// WTF-8 like every string here, with a byte that is not part of a character held
// as the lone surrogate U+DC00 + byte (php-strings.js); phpRaw turns that text into
// the bytes PHP's offsets count, and phpBytesText turns bytes back into it.

// phpRaw is phsBytes: the bytes of s, a held byte (WTF-8 ED B2 80 .. ED B3 BF)
// being that one byte.
func phpRaw(s string) string {
	if strings.IndexByte(s, 0xED) < 0 {
		return s
	}
	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); {
		if s[i] == 0xED && i+2 < len(s) && (s[i+1] == 0xB2 || s[i+1] == 0xB3) {
			b = append(b, byte(s[i+1]&1)<<6|s[i+2]&0x3F|0x80)
			i += 3
			continue
		}
		b = append(b, s[i])
		i++
	}
	return string(b)
}

// phpBytesText is phsText: a well-formed UTF-8 sequence is its character and any
// other byte is held as U+DC00 + byte.
func phpBytesText(b string) string {
	if utf8.ValidString(b) {
		return b
	}
	out := make([]byte, 0, len(b)+8)
	for i := 0; i < len(b); {
		r, size := utf8.DecodeRuneInString(b[i:])
		if r == utf8.RuneError && size <= 1 {
			out = appendWTF8(out, 0xDC00+rune(b[i]))
			i++
			continue
		}
		out = append(out, b[i:i+size]...)
		i += size
	}
	return string(out)
}

// phpJoin is phsJoin: a . b, re-read when a ends in a held byte and b starts with
// a held continuation byte (WTF-8 ED B2 ..), since the two may spell a character.
func phpJoin(a, b string) string {
	n := len(a)
	if n < 3 || len(b) < 3 || b[0] != 0xED || b[1] != 0xB2 || a[n-3] != 0xED || (a[n-2] != 0xB2 && a[n-2] != 0xB3) {
		return a + b
	}
	return phpBytesText(phpRaw(a) + phpRaw(b))
}

// phpOffset is phsOffset: $s[$i], the one-byte string at byte offset i (from the
// end when negative), "" outside the string.
func phpOffset(s string, i int) string {
	r := phpRaw(s)
	if i < 0 {
		i += len(r)
	}
	if i < 0 || i >= len(r) {
		return ""
	}
	return phpBytesText(r[i : i+1])
}

// phpSetOffset is phsSetOffset: the byte at offset i replaced by c's first byte,
// a write past the end padding with spaces; ok is false before the start.
func phpSetOffset(s string, i int, c string) (string, bool) {
	r := []byte(phpRaw(s))
	if i < 0 {
		i += len(r)
	}
	if i < 0 {
		return "", false
	}
	for len(r) < i {
		r = append(r, ' ')
	}
	cb := phpRaw(c)[0]
	if i == len(r) {
		r = append(r, cb)
	} else {
		r[i] = cb
	}
	return phpBytesText(string(r)), true
}

// phpSubstr is phsSubstr: PHP 8's substr rules over bytes.
func phpSubstr(s string, start int, length int, hasLen bool) string {
	s = phpRaw(s)
	n := len(s)
	f := start
	if f > n {
		return ""
	}
	if f < 0 {
		if -f > n {
			f = 0
		} else {
			f = n + f
		}
	}
	l := n - f
	if hasLen {
		if length < 0 {
			if -length > n-f {
				l = 0
			} else {
				l = n - f + length
			}
		} else if length < n-f {
			l = length
		}
	}
	if l <= 0 {
		return ""
	}
	return phpBytesText(s[f : f+l])
}

// phpStrpos is phsStrpos: the first match at or after off, -1 for none, -2 for
// an offset outside the haystack.
func phpStrpos(h, needle string, off int) int {
	h, needle = phpRaw(h), phpRaw(needle)
	n := len(h)
	if off < 0 {
		off += n
	}
	if off < 0 || off > n {
		return -2
	}
	at := strings.Index(h[off:], needle)
	if at < 0 {
		return -1
	}
	return at + off
}

// phpStrrpos is phsStrrpos: the last match, searched from a non-negative off, or
// ending by n+off+len(needle) for a negative one.
func phpStrrpos(h, needle string, off int) int {
	h, needle = phpRaw(h), phpRaw(needle)
	n := len(h)
	lo, hi := 0, n
	if off >= 0 {
		if off > n {
			return -2
		}
		lo = off
	} else {
		if -off > n {
			return -2
		}
		if -off >= len(needle) {
			hi = n + off + len(needle)
		}
	}
	for p := hi - len(needle); p >= lo; p-- {
		if strings.HasPrefix(h[p:], needle) {
			return p
		}
	}
	return -1
}

// phpCase is phsCase: ASCII-only case mapping.
func phpCase(s string, up bool) string {
	b := []byte(s)
	for i, c := range b {
		if up && c >= 'a' && c <= 'z' {
			b[i] = c - 32
		} else if !up && c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}

// phpFmtFloat is phsFmtFloat: sprintf's %f / %e of a non-negative finite double,
// exact and rounded half to even, with php_conv_fp's unpadded exponent.
func phpFmtFloat(a float64, conv string, prec int) string {
	if conv == "f" {
		return strconv.FormatFloat(a, 'f', prec, 64)
	}
	s := strconv.FormatFloat(a, 'e', prec, 64)
	at := strings.IndexByte(s, 'e')
	exp, _ := strconv.Atoi(s[at+1:])
	if exp < 0 {
		return s[:at] + "e-" + strconv.Itoa(-exp)
	}
	return s[:at] + "e+" + strconv.Itoa(exp)
}

// phpMath is phsMath: the C library function of one or two doubles op names.
func phpMath(op string, a, b float64) float64 {
	switch op {
	case "floor":
		return math.Floor(a)
	case "ceil":
		return math.Ceil(a)
	case "trunc":
		return math.Trunc(a)
	case "sqrt":
		return math.Sqrt(a)
	case "exp":
		return math.Exp(a)
	case "expm1":
		return math.Expm1(a)
	case "log":
		return math.Log(a)
	case "log1p":
		return math.Log1p(a)
	case "log2":
		return math.Log2(a)
	case "log10":
		return math.Log10(a)
	case "sin":
		return math.Sin(a)
	case "cos":
		return math.Cos(a)
	case "tan":
		return math.Tan(a)
	case "asin":
		return math.Asin(a)
	case "acos":
		return math.Acos(a)
	case "atan":
		return math.Atan(a)
	case "sinh":
		return math.Sinh(a)
	case "cosh":
		return math.Cosh(a)
	case "tanh":
		return math.Tanh(a)
	case "asinh":
		return math.Asinh(a)
	case "acosh":
		return math.Acosh(a)
	case "atanh":
		return math.Atanh(a)
	case "atan2":
		return math.Atan2(a, b)
	case "hypot":
		return math.Hypot(a, b)
	case "pow":
		return math.Pow(a, b)
	case "div":
		return a / b
	case "fmod":
		return math.Mod(a, b)
	}
	return math.Copysign(a, b)
}

func init() {
	rxExtraExterns = append(rxExtraExterns, func(rt *jsrt, m map[string]func(args []uint64) uint64) {
		u := rt.unwrap
		w := rt.wrap
		num := func(v interface{}) int { return int(rt.phpToLong(v)) }
		m["js_phsubstr"] = func(a []uint64) uint64 {
			l := u(a[2])
			return w(phpSubstr(rt.phpStr(u(a[0])), num(u(a[1])), num(l), !isNullish(l)))
		}
		pos := func(name string, at int) uint64 {
			if at == -2 {
				rt.phpRaise("ValueError", name+"(): Argument #3 ($offset) must be contained in argument #1 ($haystack)")
			}
			if at < 0 {
				return w(false)
			}
			return w(phpMkInt(int64(at)))
		}
		m["js_phstrpos"] = func(a []uint64) uint64 {
			return pos("strpos", phpStrpos(rt.phpStr(u(a[0])), rt.phpStr(u(a[1])), num(u(a[2]))))
		}
		m["js_phstrrpos"] = func(a []uint64) uint64 {
			return pos("strrpos", phpStrrpos(rt.phpStr(u(a[0])), rt.phpStr(u(a[1])), num(u(a[2]))))
		}
		m["js_phstrcmp"] = func(a []uint64) uint64 {
			return w(phpMkInt(int64(strings.Compare(phpRaw(rt.phpStr(u(a[0]))), phpRaw(rt.phpStr(u(a[1])))))))
		}
		m["js_phord"] = func(a []uint64) uint64 {
			s := phpRaw(rt.phpStr(u(a[0])))
			if s == "" {
				return w(phpMkInt(0))
			}
			return w(phpMkInt(int64(s[0])))
		}
		m["js_phstrlen"] = func(a []uint64) uint64 {
			return w(phpMkInt(int64(len(phpRaw(rt.phpStr(u(a[0])))))))
		}
		// A string literal assembled from bytes, one per character (dqPiece in
		// php-to-llvm-ir.abnf).
		m["js_phfrombytes"] = func(a []uint64) uint64 {
			s := rt.phpStr(u(a[0]))
			b := make([]byte, 0, len(s))
			for _, r := range s {
				b = append(b, byte(r))
			}
			return w(phpBytesText(string(b)))
		}
		// $s[$k] = $v on a string: the string with the one byte replaced.
		m["js_phsetoffset"] = func(a []uint64) uint64 {
			c := rt.phpStr(u(a[2]))
			if c == "" {
				rt.fail("cannot assign an empty string to a string offset")
			}
			s, ok := phpSetOffset(rt.phpStr(u(a[0])), int(rt.phpToLong(u(a[1]))), c)
			if !ok {
				rt.fail("cannot write to a negative string offset")
			}
			return w(s)
		}
		m["js_phchr"] = func(a []uint64) uint64 {
			b := rt.phpToLong(u(a[0])) % 256
			if b < 0 {
				b += 256
			}
			return w(phpBytesText(string([]byte{byte(b)})))
		}
		m["js_phcase"] = func(a []uint64) uint64 {
			return w(phpCase(rt.phpStr(u(a[0])), rt.truthy(u(a[1]))))
		}
		m["js_phfmtfloat"] = func(a []uint64) uint64 {
			x := phpNumOf(rt.phpToNum(u(a[0])))
			if x < 0 {
				x = -x
			}
			conv := rt.phpStr(u(a[1]))
			prec := num(u(a[2]))
			if conv == "g" {
				return w(strings.Replace(phpFloFormat(x, prec), "E", "e", 1))
			}
			return w(phpFmtFloat(x, conv, prec))
		}
		m["js_phmath"] = func(a []uint64) uint64 {
			x := phpNumOf(rt.phpToNum(u(a[1])))
			y := phpNumOf(rt.phpToNum(u(a[2])))
			return w(phpMkFlo(phpMath(rt.phpStr(u(a[0])), x, y)))
		}
		m["js_phkeyexists"] = func(a []uint64) uint64 {
			keys, _, ok := phpArrParts(u(a[1]))
			if !ok {
				rt.phpRaise("TypeError", "array_key_exists(): Argument #2 ($array) must be of type array, "+
					rt.phpDebugType(u(a[1]))+" given")
			}
			return w(phpKeyIndex(keys, u(a[0])) >= 0)
		}
		m["js_phkeylast"] = func(a []uint64) uint64 {
			keys, _, ok := phpArrParts(u(a[0]))
			if !ok || len(keys.elems) == 0 {
				return w(jsNull)
			}
			return w(keys.elems[len(keys.elems)-1])
		}
		// array_pop: the last value, and PHP's next free key steps back when the
		// popped key was the last one handed out.
		m["js_phpop"] = func(a []uint64) uint64 {
			o, ok := phpAsArr(u(a[0]))
			if !ok {
				return w(jsNull)
			}
			keys, _ := o.props["keys"].(*jsArray)
			vals, _ := o.props["vals"].(*jsArray)
			n := len(keys.elems)
			if n == 0 {
				return w(jsNull)
			}
			k, v := keys.elems[n-1], phpDeref(vals.elems[n-1])
			keys.dropIdx()
			vals.dropIdx()
			keys.elems = keys.elems[:n-1]
			vals.elems = vals.elems[:n-1]
			if f, isN := k.(float64); isN && f == phpArrNext(o)-1 {
				o.set("__next", f)
			}
			return w(v)
		}
	})
}
