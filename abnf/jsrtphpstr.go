package abnf

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

// PHP's byte-string primitives and the natives lib/php-stdlib.php is written on -
// the GO TWIN of languages/lib/php-strings.js (shared by the interpreter half and
// layer 2) plus the js_ph* externs that wrap it. A Go string already holds the
// UTF-8 bytes PHP's offsets count, so a slice is a byte slice; it is turned back
// into text the way php-strings.js's phsText does: a well-formed UTF-8 sequence is
// its character and any other byte is the character of the same value.

func phpBytesText(b string) string {
	if utf8.ValidString(b) {
		return b
	}
	var sb strings.Builder
	for i := 0; i < len(b); {
		r, size := utf8.DecodeRuneInString(b[i:])
		if r == utf8.RuneError && size <= 1 {
			sb.WriteRune(rune(b[i]))
			i++
			continue
		}
		sb.WriteString(b[i : i+size])
		i += size
	}
	return sb.String()
}

// phpSubstr is phsSubstr: PHP 8's substr rules over bytes.
func phpSubstr(s string, start int, length int, hasLen bool) string {
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
			return w(phpMkInt(int64(strings.Compare(rt.phpStr(u(a[0])), rt.phpStr(u(a[1]))))))
		}
		m["js_phord"] = func(a []uint64) uint64 {
			s := rt.phpStr(u(a[0]))
			if s == "" {
				return w(phpMkInt(0))
			}
			return w(phpMkInt(int64(s[0])))
		}
		m["js_phchr"] = func(a []uint64) uint64 {
			b := rt.phpToLong(u(a[0])) % 256
			if b < 0 {
				b += 256
			}
			return w(string(rune(b)))
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
