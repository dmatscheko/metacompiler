package abnf

// The primitives lib/go-regexp.go (Go's regexp package) is written over, as the
// go-to-llvm-ir grammar's module calls them: goRxError, goRxNames and goRxExec
// over jsrtregex.go's R dialect. Twins of lib/go-regexp.js.

import "unicode/utf8"

// goRxLast is the Go string the last search read and goRxText's reading of it.
var goRxLast struct {
	have bool
	s    string
	t    []rune
	off  []int
}

// goRxText reads a Go string the way go-interpreter.abnf holds one: each
// character as its UTF-16 units, a byte that is not part of one as U+DC00 +
// byte. off[i] is unit i's byte offset and off[len(t)] the end; nil when every
// unit is one byte. Twin of goRxText in go-interpreter.abnf and lib/go-rt.metajs.
func goRxText(s string) ([]rune, []int) {
	t := make([]rune, 0, len(s))
	ascii := true
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			ascii = false
		}
		t = append(t, rune(s[i]))
	}
	if ascii {
		return t, nil
	}
	t = t[:0]
	off := make([]int, 0, len(s)+1)
	for b := 0; b < len(s); {
		r, w := utf8.DecodeRuneInString(s[b:])
		switch {
		case r == utf8.RuneError && w == 1:
			t = append(t, 0xDC00+rune(s[b]))
			off = append(off, b)
		case r > 0xFFFF:
			r -= 0x10000
			t = append(t, 0xD800+r>>10, 0xDC00+r&0x3FF)
			off = append(off, b, b)
		default:
			t = append(t, r)
			off = append(off, b)
		}
		b += w
	}
	return t, append(off, len(s))
}

func init() {
	rxExtraExterns = append(rxExtraExterns, func(rt *jsrt, m map[string]func(args []uint64) uint64) {
		u := rt.unwrap
		w := rt.wrap
		arg := func(a []uint64, i int) interface{} {
			args, _ := u(a[0]).(*jsArray)
			if args == nil || i >= len(args.elems) {
				return jsUndef
			}
			return args.elems[i]
		}
		// goRxError(expr): Go's error text, "" when it compiles.
		m["js_gorxerror"] = func(a []uint64) uint64 {
			re := rxGet(rt.toString(arg(a, 0)), "R")
			return w(re.err)
		}
		// goRxNames(expr): SubexpNames.
		m["js_gorxnames"] = func(a []uint64) uint64 {
			re := rxGet(rt.toString(arg(a, 0)), "R")
			out := &jsArray{elems: []interface{}{""}}
			for g := 1; g <= re.ngroups; g++ {
				name := ""
				for k, ng := range re.nameGroups {
					if ng == g {
						name = re.names[k]
						break
					}
				}
				out.elems = append(out.elems, name)
			}
			return w(out)
		}
		// goRxExec(expr, s, pos): the leftmost match at or after byte pos as
		// submatch index pairs, empty for none.
		m["js_gorxexec"] = func(a []uint64) uint64 {
			re := rxGet(rt.toString(arg(a, 0)), "R")
			s := rt.toString(arg(a, 1))
			pos := int(rt.toNumber(arg(a, 2)))
			if !goRxLast.have || s != goRxLast.s {
				goRxLast.t, goRxLast.off = goRxText(s)
				goRxLast.s, goRxLast.have = s, true
			}
			off := goRxLast.off
			at := pos
			if off != nil {
				lo, hi := 0, len(off)-1
				for lo < hi {
					mid := (lo + hi) / 2
					if off[mid] < pos {
						lo = mid + 1
					} else {
						hi = mid
					}
				}
				at = lo
			}
			out := &jsArray{}
			mt, ok := re.search(goRxLast.t, at)
			if !ok {
				rt.fail("regexp: step limit exceeded")
			}
			if mt == nil {
				return w(out)
			}
			for _, c := range mt.caps {
				switch {
				case c < 0:
					out.elems = append(out.elems, float64(-1))
				case off != nil:
					out.elems = append(out.elems, float64(off[c]))
				default:
					out.elems = append(out.elems, float64(c))
				}
			}
			return w(out)
		}
	})
}
