package abnf

// jsrtluastr.go -- Lua's string library for the compiler grammar
// (languages/lua-to-llvm-ir.abnf): string.find, match, gmatch, gsub, byte, char,
// reverse, and select, plus the lauxlib argument checks they share with the math
// externs in jsrtlua.go.
//
// The pattern matcher is a line-by-line port of languages/lib/lua-strlib.js,
// which the interpreter half includes and layer 2 imports; keep the three in
// step. A Lua string is a BYTE string held one character per byte, so the port
// works on the UTF-16 units of the host string (every unit is 0..255).

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

const (
	lsCapUnfinished = -1
	lsCapPosition   = -2
	lsMaxCaptures   = 32
	lsMaxCCalls     = 200
)

// ----- the lauxlib argument checks -----

// luArgErr raises lua's "bad argument #n to 'fn' (msg)", fn the function's global
// name ('math.floor'). The marker also carries n and msg on their own: the
// emitted lua_argmsg rewords the message with the name the CALL SITE spelled
// ('floor'), which only the module knows. Twin of luArgErr in
// languages/lib/lua-rt.metajs.
func (rt *jsrt) luArgErr(fn string, n int, msg string) {
	o := newJSObject()
	o.set("__luerr", fmt.Sprintf("bad argument #%d to '%s' (%s)", n, fn, msg))
	o.set("__luarg", float64(n))
	o.set("__lumsg", msg)
	panic(&jsThrown{value: o})
}

// luCheckAny is luaL_checkany: argument n must be present, nil or not.
func (rt *jsrt) luCheckAny(fn string, n int, args []interface{}) {
	if n > len(args) {
		rt.luArgErr(fn, n, "value expected")
	}
}

// luArgTypeErr names what argument n was, "no value" when the call passed fewer
// than n arguments.
func (rt *jsrt) luArgTypeErr(fn string, n int, want string, args []interface{}) {
	got := "no value"
	if n <= len(args) {
		got = rt.luTypeName(args[n-1])
	}
	rt.luArgErr(fn, n, want+" expected, got "+got)
}

func luArgAt(args []interface{}, n int) interface{} {
	if n <= len(args) {
		return args[n-1]
	}
	return jsUndef
}

func luIsNil(v interface{}) bool { return v == nil || v == jsUndef }

func (rt *jsrt) luCheckStr(fn string, n int, args []interface{}) string {
	v := luArgAt(args, n)
	if s, ok := v.(string); ok {
		return s
	}
	if luIsNum(v) {
		return luNumStr(v)
	}
	rt.luArgTypeErr(fn, n, "string", args)
	return ""
}

// luCheckNum is a number argument, keeping its subtype; a numeric string
// converts.
func (rt *jsrt) luCheckNum(fn string, n int, args []interface{}) interface{} {
	v, ok := luToNum(luArgAt(args, n))
	if !ok {
		rt.luArgTypeErr(fn, n, "number", args)
	}
	return v
}

// luCheckInt is an integer argument as a double: a float with an integral value
// and a numeric string convert; anything else is an error.
func (rt *jsrt) luCheckInt(fn string, n int, args []interface{}) float64 {
	v := rt.luCheckNum(fn, n, args)
	d := luNumOf(v)
	if !luIsInt(v) && (d-d != 0 || d != math.Trunc(d) || d >= luMinIntF || d < -luMinIntF) {
		rt.luArgErr(fn, n, "number has no integer representation")
	}
	return d
}

func (rt *jsrt) luOptInt(fn string, n int, args []interface{}, dflt float64) float64 {
	if luIsNil(luArgAt(args, n)) {
		return dflt
	}
	return rt.luCheckInt(fn, n, args)
}

func (rt *jsrt) luOptStr(fn string, n int, args []interface{}, dflt string) string {
	if luIsNil(luArgAt(args, n)) {
		return dflt
	}
	return rt.luCheckStr(fn, n, args)
}

// luIntOf is an integer-valued number as the exact integer (3.0 is 3).
func luIntOf(v interface{}) int64 {
	if luIsInt(v) {
		return luInt64(v)
	}
	return int64(luNumOf(v))
}

// luMk is several results as ONE Lua value: exactly one stays unwrapped, any
// other count becomes the {__mv} box lua_mk builds in the emitted module.
func luMk(vals []interface{}) interface{} {
	if len(vals) == 1 {
		return vals[0]
	}
	o := newJSObject()
	o.set("__mv", &jsArray{elems: vals})
	return o
}

// luFirstOf is the first value of a call result.
func luFirstOf(v interface{}) interface{} {
	if o, ok := v.(*jsObject); ok {
		if mv, ok := o.props["__mv"].(*jsArray); ok {
			if len(mv.elems) == 0 {
				return jsUndef
			}
			return mv.elems[0]
		}
	}
	return v
}

// ----- positions -----

func lsPosI(pos, l float64) float64 {
	if pos > 0 {
		return pos
	}
	if pos == 0 {
		return 1
	}
	if pos < -l {
		return 1
	}
	return l + pos + 1
}

func lsEndPos(pos, l float64) float64 {
	if pos > l {
		return l
	}
	if pos >= 0 {
		return pos
	}
	if pos < -l {
		return 0
	}
	return l + pos + 1
}

// ----- character classes (C locale) -----

func lsIsAlpha(c int) bool  { return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') }
func lsIsDigit(c int) bool  { return c >= '0' && c <= '9' }
func lsIsLower(c int) bool  { return c >= 'a' && c <= 'z' }
func lsIsUpper(c int) bool  { return c >= 'A' && c <= 'Z' }
func lsIsSpace(c int) bool  { return c == ' ' || (c >= 9 && c <= 13) }
func lsIsCntrl(c int) bool  { return c < 32 || c == 127 }
func lsIsPunct(c int) bool  { return c > 32 && c < 127 && !lsIsAlpha(c) && !lsIsDigit(c) }
func lsIsXDigit(c int) bool { return lsIsDigit(c) || (c >= 'A' && c <= 'F') || (c >= 'a' && c <= 'f') }

func lsMatchClass(c, cl int) bool {
	lower := cl
	if lsIsUpper(cl) {
		lower = cl + 32
	}
	var res bool
	switch lower {
	case 'a':
		res = lsIsAlpha(c)
	case 'c':
		res = lsIsCntrl(c)
	case 'd':
		res = lsIsDigit(c)
	case 'g':
		res = c > 32 && c < 127
	case 'l':
		res = lsIsLower(c)
	case 'p':
		res = lsIsPunct(c)
	case 's':
		res = lsIsSpace(c)
	case 'u':
		res = lsIsUpper(c)
	case 'w':
		res = lsIsAlpha(c) || lsIsDigit(c)
	case 'x':
		res = lsIsXDigit(c)
	case 'z': // kept by lua 5.4 for the NUL byte
		res = c == 0
	default:
		return cl == c
	}
	if lsIsUpper(cl) {
		return !res
	}
	return res
}

// ----- the match state -----

type lsState struct {
	rt       *jsrt
	src, pat []uint16
	level    int
	cinit    [lsMaxCaptures]int
	clen     [lsMaxCaptures]int
	depth    int
}

func (ms *lsState) p(i int) int {
	if i < len(ms.pat) {
		return int(ms.pat[i])
	}
	return 0
}

func (ms *lsState) s(i int) int {
	if i < len(ms.src) {
		return int(ms.src[i])
	}
	return 0
}

func (ms *lsState) fail(msg string) { ms.rt.luFail("%s", msg) }

func (ms *lsState) reprep() {
	ms.level = 0
	ms.depth = lsMaxCCalls
}

func (ms *lsState) classEnd(p int) int {
	q := p
	c := ms.p(q)
	q++
	if c == '%' {
		if q >= len(ms.pat) {
			ms.fail("malformed pattern (ends with '%')")
		}
		return q + 1
	}
	if c == '[' {
		if ms.p(q) == '^' {
			q++
		}
		for {
			if q >= len(ms.pat) {
				ms.fail("malformed pattern (missing ']')")
			}
			cc := ms.p(q)
			q++
			if cc == '%' && q < len(ms.pat) {
				q++
			}
			if ms.p(q) == ']' {
				break
			}
		}
		return q + 1
	}
	return q
}

func (ms *lsState) matchBracket(c, p, ec int) bool {
	q := p + 1
	sig := true
	if ms.p(q) == '^' {
		sig = false
		q++
	}
	for q < ec {
		pc := ms.p(q)
		if pc == '%' {
			q++
			if lsMatchClass(c, ms.p(q)) {
				return sig
			}
			q++
		} else if ms.p(q+1) == '-' && q+2 < ec {
			if pc <= c && c <= ms.p(q+2) {
				return sig
			}
			q += 3
		} else {
			if pc == c {
				return sig
			}
			q++
		}
	}
	return !sig
}

func (ms *lsState) singleMatch(s, p, ep int) bool {
	if s >= len(ms.src) {
		return false
	}
	c := int(ms.src[s])
	switch ms.p(p) {
	case '.':
		return true
	case '%':
		return lsMatchClass(c, ms.p(p+1))
	case '[':
		return ms.matchBracket(c, p, ep-1)
	}
	return ms.p(p) == c
}

func (ms *lsState) matchBalance(s, p int) int {
	if p+1 >= len(ms.pat) {
		ms.fail("malformed pattern (missing arguments to '%b')")
	}
	if s >= len(ms.src) || int(ms.src[s]) != ms.p(p) {
		return -1
	}
	b, e := ms.p(p), ms.p(p+1)
	cont := 1
	for q := s + 1; q < len(ms.src); q++ {
		c := int(ms.src[q])
		if c == e {
			cont--
			if cont == 0 {
				return q + 1
			}
		} else if c == b {
			cont++
		}
	}
	return -1
}

func (ms *lsState) maxExpand(s, p, ep int) int {
	i := 0
	for ms.singleMatch(s+i, p, ep) {
		i++
	}
	for ; i >= 0; i-- {
		if res := ms.doMatch(s+i, ep+1); res != -1 {
			return res
		}
	}
	return -1
}

func (ms *lsState) minExpand(s, p, ep int) int {
	for q := s; ; q++ {
		if res := ms.doMatch(q, ep+1); res != -1 {
			return res
		}
		if !ms.singleMatch(q, p, ep) {
			return -1
		}
	}
}

func (ms *lsState) startCapture(s, p, what int) int {
	if ms.level >= lsMaxCaptures {
		ms.fail("too many captures")
	}
	ms.cinit[ms.level] = s
	ms.clen[ms.level] = what
	ms.level++
	res := ms.doMatch(s, p)
	if res == -1 {
		ms.level--
	}
	return res
}

func (ms *lsState) captureToClose() int {
	for level := ms.level - 1; level >= 0; level-- {
		if ms.clen[level] == lsCapUnfinished {
			return level
		}
	}
	ms.fail("invalid pattern capture")
	return 0
}

func (ms *lsState) endCapture(s, p int) int {
	l := ms.captureToClose()
	ms.clen[l] = s - ms.cinit[l]
	res := ms.doMatch(s, p)
	if res == -1 {
		ms.clen[l] = lsCapUnfinished
	}
	return res
}

func (ms *lsState) checkCapture(ch int) int {
	l := ch - '1'
	if l < 0 || l >= ms.level || ms.clen[l] == lsCapUnfinished {
		ms.fail("invalid capture index %" + strconv.Itoa(l+1))
	}
	return l
}

func (ms *lsState) matchCapture(s, ch int) int {
	l := ms.checkCapture(ch)
	n, init := ms.clen[l], ms.cinit[l]
	if len(ms.src)-s >= n {
		for k := 0; k < n; k++ {
			if ms.src[init+k] != ms.src[s+k] {
				return -1
			}
		}
		return s + n
	}
	return -1
}

// doMatch is do_match: the end of the match of pattern[p..] at subject offset s,
// or -1.
func (ms *lsState) doMatch(s, p int) int {
	ms.depth--
	if ms.depth < 0 {
		ms.fail("pattern too complex")
	}
	for p < len(ms.pat) {
		pc := ms.p(p)
		if pc == '(' {
			if ms.p(p+1) == ')' {
				s = ms.startCapture(s, p+2, lsCapPosition)
			} else {
				s = ms.startCapture(s, p+1, lsCapUnfinished)
			}
			break
		}
		if pc == ')' {
			s = ms.endCapture(s, p+1)
			break
		}
		if pc == '$' && p+1 == len(ms.pat) {
			if s != len(ms.src) {
				s = -1
			}
			break
		}
		if pc == '%' {
			nx := ms.p(p + 1)
			if nx == 'b' {
				s = ms.matchBalance(s, p+2)
				if s != -1 {
					p += 4
					continue
				}
				break
			}
			if nx == 'f' {
				p += 2
				if ms.p(p) != '[' {
					ms.fail("missing '[' after '%f' in pattern")
				}
				fep := ms.classEnd(p)
				prev := 0
				if s != 0 {
					prev = int(ms.src[s-1])
				}
				if !ms.matchBracket(prev, p, fep-1) && ms.matchBracket(ms.s(s), p, fep-1) {
					p = fep
					continue
				}
				s = -1
				break
			}
			if lsIsDigit(nx) {
				s = ms.matchCapture(s, nx)
				if s != -1 {
					p += 2
					continue
				}
				break
			}
		}
		ep := ms.classEnd(p)
		epc := ms.p(ep)
		if !ms.singleMatch(s, p, ep) {
			if epc == '*' || epc == '?' || epc == '-' {
				p = ep + 1
				continue
			}
			s = -1
			break
		}
		if epc == '?' {
			if r := ms.doMatch(s+1, ep+1); r != -1 {
				s = r
				break
			}
			p = ep + 1
			continue
		}
		if epc == '+' {
			s = ms.maxExpand(s+1, p, ep)
			break
		}
		if epc == '*' {
			s = ms.maxExpand(s, p, ep)
			break
		}
		if epc == '-' {
			s = ms.minExpand(s, p, ep)
			break
		}
		s++
		p = ep
	}
	ms.depth++
	return s
}

// ----- captures -----

func (ms *lsState) oneCapture(i, s, e int) interface{} {
	if i >= ms.level {
		if i != 0 {
			ms.fail("invalid capture index %" + strconv.Itoa(i+1))
		}
		return strFromUnits(ms.src[s:e])
	}
	l, init := ms.clen[i], ms.cinit[i]
	if l == lsCapUnfinished {
		ms.fail("unfinished capture")
	}
	if l == lsCapPosition {
		return float64(init + 1)
	}
	return strFromUnits(ms.src[init : init+l])
}

func (ms *lsState) captures(s, e int, whole bool) []interface{} {
	n := ms.level
	if n == 0 && whole {
		n = 1
	}
	out := make([]interface{}, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, ms.oneCapture(i, s, e))
	}
	return out
}

func lsNoSpecials(p []uint16) bool {
	for _, c := range p {
		switch c {
		case '^', '$', '*', '+', '?', '.', '(', '[', '%', '-':
			return false
		}
	}
	return true
}

func lsIndex(s, p []uint16, from int) int {
	for i := from; i+len(p) <= len(s); i++ {
		k := 0
		for k < len(p) && s[i+k] == p[k] {
			k++
		}
		if k == len(p) {
			return i
		}
	}
	return -1
}

// lsFindAux is str_find_aux: nil, or find's start and end and the captures, or
// match's captures (the whole match when there are none).
func (rt *jsrt) lsFindAux(s, p []uint16, initArg float64, plain, find bool) []interface{} {
	ls := len(s)
	init := int(lsPosI(initArg, float64(ls))) - 1
	if init > ls {
		return nil
	}
	if find && (plain || lsNoSpecials(p)) {
		at := lsIndex(s, p, init)
		if at < 0 {
			return nil
		}
		return []interface{}{float64(at + 1), float64(at + len(p))}
	}
	anchor := len(p) > 0 && p[0] == '^'
	if anchor {
		p = p[1:]
	}
	ms := &lsState{rt: rt, src: s, pat: p}
	for s1 := init; ; {
		ms.reprep()
		if e := ms.doMatch(s1, 0); e != -1 {
			if find {
				return append([]interface{}{float64(s1 + 1), float64(e)}, ms.captures(-1, 0, false)...)
			}
			return ms.captures(s1, e, true)
		}
		s1++
		if anchor || s1 > ls {
			break
		}
	}
	return nil
}

// lsAddS is add_s: a replacement string with %0..%9 and %%.
func (ms *lsState) addS(out []uint16, repl []uint16, s, e int) []uint16 {
	for i := 0; i < len(repl); i++ {
		c := repl[i]
		if c != '%' {
			out = append(out, c)
			continue
		}
		i++
		d := 0
		if i < len(repl) {
			d = int(repl[i])
		}
		if d == '%' {
			out = append(out, '%')
		} else if lsIsDigit(d) {
			if d == '0' {
				out = append(out, ms.src[s:e]...)
			} else {
				switch cv := ms.oneCapture(d-'1', s, e).(type) {
				case string:
					out = append(out, strUnits(cv)...)
				case float64:
					out = append(out, strUnits(luNumStr(cv))...)
				}
			}
		} else {
			ms.fail("invalid use of '%' in replacement string")
		}
	}
	return out
}

// luReplVal is a gsub replacement value: false or nil keeps the match (ok
// false), a string or a number replaces it, anything else is an error.
func (rt *jsrt) luReplVal(v interface{}) (string, bool) {
	if luIsNil(v) || v == false {
		return "", false
	}
	if s, ok := v.(string); ok {
		return s, true
	}
	if luIsNum(v) {
		return luNumStr(v), true
	}
	rt.luFail("invalid replacement value (a %s)", rt.luTypeName(v))
	return "", false
}

func init() {
	rxExtraExterns = append(rxExtraExterns, func(rt *jsrt, m map[string]func(args []uint64) uint64) {
		u := rt.unwrap
		w := rt.wrap
		argsOf := func(h uint64) []interface{} {
			if arr, ok := u(h).(*jsArray); ok {
				return arr.elems
			}
			return nil
		}
		// string.find (a[1] true) and string.match: the argument array in, the
		// result values out.
		m["js_lufind"] = func(a []uint64) uint64 {
			args := argsOf(a[0])
			find := rt.truthy(u(a[1]))
			fn := "string.match"
			if find {
				fn = "string.find"
			}
			s := rt.luCheckStr(fn, 1, args)
			p := rt.luCheckStr(fn, 2, args)
			init := rt.luOptInt(fn, 3, args, 1)
			plain := find && !luIsNil(luArgAt(args, 4)) && luArgAt(args, 4) != false
			r := rt.lsFindAux(strUnits(s), strUnits(p), init, plain, find)
			if r == nil {
				return w(jsUndef)
			}
			return w(luMk(r))
		}
		// gmatch's iteration state, and one step of it: the captures of the next
		// match, or nil.
		m["js_lugmatch"] = func(a []uint64) uint64 {
			args := argsOf(a[0])
			s := rt.luCheckStr("string.gmatch", 1, args)
			p := rt.luCheckStr("string.gmatch", 2, args)
			ls := float64(len(strUnits(s)))
			init := lsPosI(rt.luOptInt("string.gmatch", 3, args, 1), ls) - 1
			if init > ls {
				init = ls + 1
			}
			st := newJSObject()
			st.set("s", s)
			st.set("p", p)
			st.set("pos", init)
			st.set("last", float64(-1))
			return w(st)
		}
		m["js_lugmatchnext"] = func(a []uint64) uint64 {
			st := u(a[0]).(*jsObject)
			ms := &lsState{rt: rt, src: strUnits(st.props["s"].(string)), pat: strUnits(st.props["p"].(string))}
			last := int(st.props["last"].(float64))
			for src := int(st.props["pos"].(float64)); src <= len(ms.src); src++ {
				ms.reprep()
				if e := ms.doMatch(src, 0); e != -1 && e != last {
					st.set("pos", float64(e))
					st.set("last", float64(e))
					return w(luMk(ms.captures(src, e, true)))
				}
			}
			return w(jsUndef)
		}
		m["js_lugsub"] = func(a []uint64) uint64 {
			args := argsOf(a[0])
			src := strUnits(rt.luCheckStr("string.gsub", 1, args))
			p := strUnits(rt.luCheckStr("string.gsub", 2, args))
			replV := luArgAt(args, 3)
			var repl []uint16
			isStr := false
			switch rv := replV.(type) {
			case string:
				repl, isStr = strUnits(rv), true
			case *jsClosure:
			default:
				if luIsNum(rv) {
					repl, isStr = strUnits(luNumStr(rv)), true
				} else if _, ok := rv.(*jsObject); !ok {
					rt.luArgTypeErr("string.gsub", 3, "string/function/table", args)
				}
			}
			maxN := -1
			if !luIsNil(luArgAt(args, 4)) {
				maxN = int(rt.luCheckInt("string.gsub", 4, args))
				if maxN < 0 {
					maxN = 0
				}
			}
			anchor := len(p) > 0 && p[0] == '^'
			if anchor {
				p = p[1:]
			}
			ms := &lsState{rt: rt, src: src, pat: p}
			pos, last, n := 0, -1, 0
			var out []uint16
			for maxN < 0 || n < maxN {
				ms.reprep()
				e := ms.doMatch(pos, 0)
				if e != -1 && e != last {
					n++
					if isStr {
						out = ms.addS(out, repl, pos, e)
					} else {
						caps := ms.captures(pos, e, true)
						var v interface{}
						if f, ok := replV.(*jsClosure); ok {
							v = luFirstOf(rt.call(f, jsUndef, caps))
						} else {
							v = rt.getMember(replV, luKey(caps[0]))
						}
						if text, ok := rt.luReplVal(v); ok {
							out = append(out, strUnits(text)...)
						} else {
							out = append(out, src[pos:e]...)
						}
					}
					pos, last = e, e
				} else if pos < len(src) {
					out = append(out, src[pos])
					pos++
				} else {
					break
				}
				if anchor {
					break
				}
			}
			out = append(out, src[pos:]...)
			return w(luMk([]interface{}{strFromUnits(out), float64(n)}))
		}
		m["js_lubyte"] = func(a []uint64) uint64 {
			args := argsOf(a[0])
			s := strUnits(rt.luCheckStr("string.byte", 1, args))
			pi := rt.luOptInt("string.byte", 2, args, 1)
			pose := lsEndPos(rt.luOptInt("string.byte", 3, args, pi), float64(len(s)))
			out := []interface{}{}
			for k := lsPosI(pi, float64(len(s))); k <= pose; k++ {
				out = append(out, float64(s[int(k)-1]))
			}
			return w(luMk(out))
		}
		m["js_luchar"] = func(a []uint64) uint64 {
			args := argsOf(a[0])
			out := make([]uint16, 0, len(args))
			for k := range args {
				c := rt.luCheckInt("string.char", k+1, args)
				if c < 0 || c > 255 {
					rt.luArgErr("string.char", k+1, "value out of range")
				}
				out = append(out, uint16(c))
			}
			return rt.wrapStr(strFromUnits(out))
		}
		m["js_lulen"] = func(a []uint64) uint64 {
			return w(float64(len(strUnits(rt.luCheckStr("string.len", 1, argsOf(a[0]))))))
		}
		lsCase := func(qual string, up bool) func(a []uint64) uint64 {
			return func(a []uint64) uint64 {
				s := strUnits(rt.luCheckStr(qual, 1, argsOf(a[0])))
				for i, c := range s {
					if up && c >= 'a' && c <= 'z' {
						s[i] = c - 32
					} else if !up && c >= 'A' && c <= 'Z' {
						s[i] = c + 32
					}
				}
				return rt.wrapStr(strFromUnits(s))
			}
		}
		m["js_luupper"] = lsCase("string.upper", true)
		m["js_lulower"] = lsCase("string.lower", false)
		m["js_lusub"] = func(a []uint64) uint64 {
			args := argsOf(a[0])
			s := strUnits(rt.luCheckStr("string.sub", 1, args))
			l := float64(len(s))
			i := lsPosI(rt.luCheckInt("string.sub", 2, args), l)
			e := lsEndPos(rt.luOptInt("string.sub", 3, args, -1), l)
			if i > e {
				return rt.wrapStr("")
			}
			return rt.wrapStr(strFromUnits(s[int(i)-1 : int(e)]))
		}
		m["js_lurep"] = func(a []uint64) uint64 {
			args := argsOf(a[0])
			s := rt.luCheckStr("string.rep", 1, args)
			n := rt.luCheckInt("string.rep", 2, args)
			sep := rt.luOptStr("string.rep", 3, args, "")
			if n < 1 || s+sep == "" {
				return rt.wrapStr("")
			}
			var b strings.Builder
			b.WriteString(s)
			for k := 1.0; k < n; k++ {
				b.WriteString(sep)
				b.WriteString(s)
			}
			return rt.wrapStr(b.String())
		}
		m["js_lureverse"] = func(a []uint64) uint64 {
			s := strUnits(rt.luCheckStr("string.reverse", 1, argsOf(a[0])))
			out := make([]uint16, len(s))
			for i, c := range s {
				out[len(s)-1-i] = c
			}
			return rt.wrapStr(strFromUnits(out))
		}
		// select(n, ...): the arguments from the n-th on, a negative n counting
		// from the end, or their count for select('#', ...).
		m["js_luselect"] = func(a []uint64) uint64 {
			args := argsOf(a[0])
			n := len(args)
			if s, ok := luArgAt(args, 1).(string); ok && len(s) > 0 && s[0] == '#' {
				return w(float64(n - 1))
			}
			i := int(rt.luCheckInt("select", 1, args))
			if i < 0 {
				i = n + i
			} else if i > n {
				i = n
			}
			if i < 1 {
				rt.luArgErr("select", 1, "index out of range")
			}
			return w(luMk(append([]interface{}{}, args[i:]...)))
		}
	})
}
