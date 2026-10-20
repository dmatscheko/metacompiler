package abnf

import "strings"

// The natives PHP's preg_* functions (lib/php-stdlib.php) are written over, as
// the php-to-llvm-ir grammar's module calls them: js_phpregc compiles and
// js_phpregx runs one match, on jsrtregex.go's C dialect. Twins of
// lib/php-preg.js, which states the rules.

// phpRxLast is the subject the last run read and the engine's reading of it.
var phpRxLast struct {
	have, utf, bad bool
	s              string
	t              []rune
	off            []int
}

// phpRxUnits is a PHP string's bytes, one unit each.
func phpRxUnits(s string) []rune {
	b := phpRaw(s)
	t := make([]rune, len(b))
	for i := 0; i < len(b); i++ {
		t[i] = rune(b[i])
	}
	return t
}

// phpRxHasByte: does s hold a byte that is not part of a character?
func phpRxHasByte(s string) bool { return phpRaw(s) != s }

// phpRxRe is phpRxRe in lib/php-preg.js: nil under u when the pattern is not UTF-8.
func phpRxRe(key string) *rxRe {
	k := strings.IndexByte(key, '/')
	mods, pat := key[:k], key[k+1:]
	if !strings.ContainsRune(mods, 'u') {
		return rxGet(string(phpRxUnits(pat)), "C"+mods)
	}
	if phpRxHasByte(pat) {
		return nil
	}
	return rxGet(pat, "C"+mods)
}

func phpRxText(s string, utf bool) {
	l := &phpRxLast
	if l.have && l.utf == utf && l.s == s {
		return
	}
	l.have, l.utf, l.s, l.off, l.bad = true, utf, s, nil, false
	if !utf {
		l.t = phpRxUnits(s)
		return
	}
	l.bad = phpRxHasByte(s)
	l.t = wtf8Runes(s) // code points, as this engine reads a J text
	off := make([]int, 0, len(l.t)+1)
	b, ascii := 0, true
	for i := 0; i < len(l.t); i++ {
		c := l.t[i]
		off = append(off, b)
		switch {
		case c < 0x80:
			b++
		case c < 0x800:
			b += 2
			ascii = false
		case c < 0x10000:
			b += 3
			ascii = false
		default:
			b += 4
			ascii = false
		}
	}
	if !ascii {
		l.off = append(off, b)
	}
}

func phpRxUnit(b int) int {
	off := phpRxLast.off
	if off == nil {
		return b
	}
	lo, hi := 0, len(off)-1
	for lo < hi {
		mid := (lo + hi) / 2
		if off[mid] < b {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if off[lo] != b {
		return -1
	}
	return lo
}

func init() {
	rxExtraExterns = append(rxExtraExterns, func(rt *jsrt, m map[string]func(args []uint64) uint64) {
		u := rt.unwrap
		w := rt.wrap
		list := func(vs []interface{}) uint64 {
			a := phpNewArr()
			for _, v := range vs {
				rt.phpArrPush(a, v)
			}
			return w(a)
		}
		// (key): the name of each group by number, or PCRE2's error text.
		m["js_phpregc"] = func(a []uint64) uint64 {
			re := phpRxRe(rt.phpStr(u(a[0])))
			if re == nil {
				return w("UTF-8 error")
			}
			if !re.ok {
				return w(re.err)
			}
			names := []interface{}{""}
			for g := 1; g <= re.ngroups; g++ {
				nm := ""
				for k, ng := range re.nameGroups {
					if ng == g {
						nm = re.names[k]
						break
					}
				}
				names = append(names, nm)
			}
			return list(names)
		}
		// (key, s, pos): the match as byte offsets, [] for none, or the PREG
		// error code; pos -o - 1 is the retry after an empty match at o.
		m["js_phpregx"] = func(a []uint64) uint64 {
			key := rt.phpStr(u(a[0]))
			re := phpRxRe(key)
			slash := strings.IndexByte(key, '/')
			phpRxText(rt.phpStr(u(a[1])), strings.ContainsRune(key[:slash], 'u'))
			if phpRxLast.bad {
				return w(phpMkInt(4))
			}
			pos := int(rt.phpToLong(u(a[2])))
			retry := pos < 0
			if retry {
				pos = -pos - 1
			}
			at := phpRxUnit(pos)
			if at < 0 {
				return w(phpMkInt(5))
			}
			var mt *rxMatch
			var ok bool
			if retry {
				mt, ok = re.searchAdv(phpRxLast.t, at, true)
			} else {
				mt, ok = re.search(phpRxLast.t, at)
			}
			if !ok {
				rt.fail("regexp: step limit exceeded")
			}
			out := []interface{}{}
			if mt != nil {
				for _, c := range mt.caps {
					switch {
					case c < 0:
						out = append(out, phpMkInt(-1))
					case phpRxLast.off != nil:
						out = append(out, phpMkInt(int64(phpRxLast.off[c])))
					default:
						out = append(out, phpMkInt(int64(c)))
					}
				}
			}
			return list(out)
		}
	})
}
