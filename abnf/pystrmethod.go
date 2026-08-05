package abnf

// Python's str METHOD LIBRARY for the Go twin - the third of the three engines
// that carry it (docs/todo.md 1.5).
//
// The other two are the "str methods" chapter at the end of
// languages/lib/python-rt.metajs (layer 2, what a NATIVE binary runs) and the
// identical chapter in languages/python-interpreter.abnf (the tree walker). This
// file is a line-by-line port of that MetaJS text: same function order, same
// names with a `pys` prefix, same helper split. Keep the three in step - a
// method that exists here and not there is exactly the halves divergence
// ./test.sh --cross is for.
//
// Registration is additive, like abnf/jsrtregexpy.go's: an init() appends to
// rxExtraExterns and wraps js_pymcall AND js_pyrxmcall, so the order the two
// registrars run in cannot matter. A receiver that is not a string, or a name
// that is not in pysIsStrMethod, is handed straight on to the wrapped function -
// including the Java-style .length / .charAt / .substring / .indexOf / .equals /
// .isEmpty of rt.memberCall, which are left reachable rather than removed.
//
// INDEXING is by UTF-16 CODE UNIT, through rt.strLen / rt.strAt / rt.strCodeAt /
// rt.strRange / rt.strIndexOf, because that is what the floor's own .length /
// charAt / charCodeAt / substring / indexOf are and what the MetaJS chapter is
// therefore written against. The length-shaped methods (center / ljust / rjust /
// zfill and the per-character walks) count CODE POINTS, as CPython does.
//
// The documented divergences from CPython are listed in the MetaJS chapter's
// header and are NOT repeated here; the short form is: no bytes type (encode() is
// the identity), ASCII-exact classification with a cased-derived fallback outside
// it, and a case mapping restricted to the mappings that do not change the code
// unit count (which is what makes goja, this engine and the C floor agree).

import (
	"strconv"
	"strings"
)

// pyStrEnv carries the two externs the port needs to reach: js_pyget (a format
// field's [key] accessor) and js_pyformat (the format mini-language itself).
// Both are taken from the extern map so that `"{0[1]:>4}".format(xs)` runs the
// same code an f-string does.
type pyStrEnv struct {
	rt    *jsrt
	getIt func(v, k interface{}) interface{}
	fmt1  func(v interface{}, spec, conv string) string
}

// ---- the string primitives, JS-shaped ----

func (e *pyStrEnv) sLen(s string) int { return e.rt.strLen(s) }

// JS substring: both ends clamp into [0, len] and begin > end swaps.
func (e *pyStrEnv) sSub(s string, a, b int) string {
	n := e.rt.strLen(s)
	if a < 0 {
		a = 0
	}
	if b < 0 {
		b = 0
	}
	if a > n {
		a = n
	}
	if b > n {
		b = n
	}
	if a > b {
		a, b = b, a
	}
	return e.rt.strRange(s, a, b)
}

func (e *pyStrEnv) sAt(s string, i int) string   { return e.rt.strAt(s, i) }
func (e *pyStrEnv) sCode(s string, i int) int    { return e.rt.strCodeAt(s, i) }
func (e *pyStrEnv) sIndex(s, sub string) int     { return e.rt.strIndexOf(s, sub) }
func (e *pyStrEnv) fail(msg string)              { e.rt.fail("%s", msg) }
func pysFromCode(c int) string                   { return strFromUnits([]uint16{uint16(c)}) }

// ---- the character primitives (pySCPW / pySChars / pySUpCh / ...) ----

func (e *pyStrEnv) cpw(s string, i int) int {
	c := e.sCode(s, i)
	if c >= 55296 && c <= 56319 && i+1 < e.sLen(s) {
		t := e.sCode(s, i+1)
		if t >= 56320 && t <= 57343 {
			return 2
		}
	}
	return 1
}

func (e *pyStrEnv) chars(s string) []string {
	out := []string{}
	n := e.sLen(s)
	for i := 0; i < n; {
		w := e.cpw(s, i)
		out = append(out, e.sSub(s, i, i+w))
		i += w
	}
	return out
}

// The LENGTH GUARD: only a case mapping that keeps the code unit count is taken,
// which is what makes goja's full mapping and this engine's / the floor's simple
// mapping answer the same string.
func (e *pyStrEnv) upCh(ch string) string {
	c := e.sCode(ch, 0)
	if c < 128 {
		if c >= 97 && c <= 122 {
			return pysFromCode(c - 32)
		}
		return ch
	}
	u := strings.ToUpper(ch)
	if e.sLen(u) != e.sLen(ch) {
		return ch
	}
	return u
}

func (e *pyStrEnv) loCh(ch string) string {
	c := e.sCode(ch, 0)
	if c < 128 {
		if c >= 65 && c <= 90 {
			return pysFromCode(c + 32)
		}
		return ch
	}
	u := strings.ToLower(ch)
	if e.sLen(u) != e.sLen(ch) {
		return ch
	}
	return u
}

func (e *pyStrEnv) isCased(ch string) bool { return e.upCh(ch) != e.loCh(ch) }
func (e *pyStrEnv) isUpCh(ch string) bool  { return e.isCased(ch) && ch == e.upCh(ch) }
func (e *pyStrEnv) isLoCh(ch string) bool  { return e.isCased(ch) && ch == e.loCh(ch) }

func (e *pyStrEnv) isAlphaCh(ch string) bool {
	c := e.sCode(ch, 0)
	if c < 128 {
		return (c >= 65 && c <= 90) || (c >= 97 && c <= 122)
	}
	return e.isCased(ch)
}

func (e *pyStrEnv) isDigitCh(ch string) bool {
	c := e.sCode(ch, 0)
	return c >= 48 && c <= 57
}

// Py_UNICODE_ISSPACE.
func pysIsSpaceCode(c int) bool {
	if c == 32 || (c >= 9 && c <= 13) || (c >= 28 && c <= 31) {
		return true
	}
	if c == 133 || c == 160 || c == 5760 {
		return true
	}
	if c >= 8192 && c <= 8202 {
		return true
	}
	return c == 8232 || c == 8233 || c == 8239 || c == 8287 || c == 12288
}

func (e *pyStrEnv) isSpaceCh(ch string) bool { return pysIsSpaceCode(e.sCode(ch, 0)) }

// str.splitlines' universal newlines.
func pysIsBreakCode(c int) bool {
	if c == 10 || c == 11 || c == 12 || c == 13 {
		return true
	}
	if c == 28 || c == 29 || c == 30 || c == 133 {
		return true
	}
	return c == 8232 || c == 8233
}

func pysRepeat(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}

func (e *pyStrEnv) cpLen(s string) int { return len(e.chars(s)) }

// ---- searching (CPython's ADJUST_INDICES) ----

func pysAdjStart(v, n int) int {
	if v < 0 {
		v += n
		if v < 0 {
			v = 0
		}
	}
	return v
}

func pysAdjEnd(v, n int) int {
	if v < 0 {
		v += n
		if v < 0 {
			v = 0
		}
	}
	if v > n {
		v = n
	}
	return v
}

func (e *pyStrEnv) find(s, sub string, start, end int) int {
	if start > end {
		return -1
	}
	at := e.sIndex(e.sSub(s, start, end), sub)
	if at < 0 {
		return -1
	}
	return at + start
}

func (e *pyStrEnv) rfind(s, sub string, start, end int) int {
	if start > end {
		return -1
	}
	m := e.sLen(sub)
	for i := end - m; i >= start; i-- {
		if e.sSub(s, i, i+m) == sub {
			return i
		}
	}
	return -1
}

func (e *pyStrEnv) count(s, sub string, start, end int) int {
	if start > end {
		return 0
	}
	m := e.sLen(sub)
	if m == 0 {
		return e.cpLen(e.sSub(s, start, end)) + 1
	}
	n := 0
	for i := start; i+m <= end; {
		if e.sSub(s, i, i+m) == sub {
			n++
			i += m
		} else {
			i++
		}
	}
	return n
}

func (e *pyStrEnv) starts(s, pre string, start, end int) bool {
	if start > end || start+e.sLen(pre) > end {
		return false
	}
	return e.sSub(s, start, start+e.sLen(pre)) == pre
}

func (e *pyStrEnv) ends(s, suf string, start, end int) bool {
	if start > end || end-e.sLen(suf) < start {
		return false
	}
	return e.sSub(s, end-e.sLen(suf), end) == suf
}

// ---- stripping: `chars` is a SET, not a prefix ----

func (e *pyStrEnv) stripHit(ch, chars string, useWS bool) bool {
	if useWS {
		return e.isSpaceCh(ch)
	}
	return e.sIndex(chars, ch) >= 0
}

func (e *pyStrEnv) strip(s, chars string, useWS, doL, doR bool) string {
	a, b := 0, e.sLen(s)
	if doL {
		for a < b && e.stripHit(e.sAt(s, a), chars, useWS) {
			a++
		}
	}
	if doR {
		for b > a && e.stripHit(e.sAt(s, b-1), chars, useWS) {
			b--
		}
	}
	return e.sSub(s, a, b)
}

// ---- splitting ----

func (e *pyStrEnv) splitWS(s string, maxsplit int) []interface{} {
	out := []interface{}{}
	n := e.sLen(s)
	cnt := 0
	i := 0
	for i < n {
		for i < n && e.isSpaceCh(e.sAt(s, i)) {
			i++
		}
		if i >= n {
			return out
		}
		if maxsplit >= 0 && cnt >= maxsplit {
			return append(out, e.sSub(s, i, n))
		}
		j := i
		for j < n && !e.isSpaceCh(e.sAt(s, j)) {
			j++
		}
		out = append(out, e.sSub(s, i, j))
		cnt++
		i = j
	}
	return out
}

func (e *pyStrEnv) splitSep(s, sep string, maxsplit int) []interface{} {
	out := []interface{}{}
	n := e.sLen(s)
	start, cnt := 0, 0
	for maxsplit < 0 || cnt < maxsplit {
		at := e.find(s, sep, start, n)
		if at < 0 {
			break
		}
		out = append(out, e.sSub(s, start, at))
		start = at + e.sLen(sep)
		cnt++
	}
	return append(out, e.sSub(s, start, n))
}

func pysReverse(a []interface{}) []interface{} {
	out := make([]interface{}, 0, len(a))
	for i := len(a) - 1; i >= 0; i-- {
		out = append(out, a[i])
	}
	return out
}

func (e *pyStrEnv) rsplitWS(s string, maxsplit int) []interface{} {
	out := []interface{}{}
	end := e.sLen(s)
	cnt := 0
	for end > 0 {
		for end > 0 && e.isSpaceCh(e.sAt(s, end-1)) {
			end--
		}
		if end <= 0 {
			return pysReverse(out)
		}
		if maxsplit >= 0 && cnt >= maxsplit {
			return pysReverse(append(out, e.sSub(s, 0, end)))
		}
		j := end
		for j > 0 && !e.isSpaceCh(e.sAt(s, j-1)) {
			j--
		}
		out = append(out, e.sSub(s, j, end))
		cnt++
		end = j
	}
	return pysReverse(out)
}

func (e *pyStrEnv) rsplitSep(s, sep string, maxsplit int) []interface{} {
	out := []interface{}{}
	end := e.sLen(s)
	cnt := 0
	for maxsplit < 0 || cnt < maxsplit {
		at := e.rfind(s, sep, 0, end)
		if at < 0 {
			break
		}
		out = append(out, e.sSub(s, at+e.sLen(sep), end))
		end = at
		cnt++
	}
	return pysReverse(append(out, e.sSub(s, 0, end)))
}

func (e *pyStrEnv) splitLines(s string, keepends bool) []interface{} {
	out := []interface{}{}
	n := e.sLen(s)
	start, i := 0, 0
	for i < n {
		c := e.sCode(s, i)
		if pysIsBreakCode(c) {
			w := 1
			if c == 13 && i+1 < n && e.sCode(s, i+1) == 10 {
				w = 2
			}
			if keepends {
				out = append(out, e.sSub(s, start, i+w))
			} else {
				out = append(out, e.sSub(s, start, i))
			}
			i += w
			start = i
		} else {
			i++
		}
	}
	if start < n {
		out = append(out, e.sSub(s, start, n))
	}
	return out
}

// ---- rewriting ----

func (e *pyStrEnv) replace(s, old, nw string, cnt int) string {
	out := ""
	reps := 0
	n := e.sLen(s)
	if e.sLen(old) == 0 {
		if cnt != 0 {
			out = nw
			reps = 1
		}
		for i := 0; i < n; {
			w := e.cpw(s, i)
			out += e.sSub(s, i, i+w)
			i += w
			if cnt < 0 || reps < cnt {
				out += nw
				reps++
			}
		}
		return out
	}
	start := 0
	for cnt < 0 || reps < cnt {
		at := e.find(s, old, start, n)
		if at < 0 {
			break
		}
		out += e.sSub(s, start, at) + nw
		start = at + e.sLen(old)
		reps++
	}
	return out + e.sSub(s, start, n)
}

// do_title: a character opens a word unless the PREVIOUS one was cased.
func (e *pyStrEnv) title(s string) string {
	out := ""
	prev := false
	for _, ch := range e.chars(s) {
		if prev {
			out += e.loCh(ch)
		} else {
			out += e.upCh(ch)
		}
		prev = e.isCased(ch)
	}
	return out
}

func (e *pyStrEnv) isTitle(s string) bool {
	prev, any := false, false
	for _, ch := range e.chars(s) {
		if e.isUpCh(ch) {
			if prev {
				return false
			}
			prev, any = true, true
		} else if e.isLoCh(ch) {
			if !prev {
				return false
			}
			prev, any = true, true
		} else {
			prev = false
		}
	}
	return any
}

func (e *pyStrEnv) capitalize(s string) string {
	out := ""
	for i, ch := range e.chars(s) {
		if i == 0 {
			out += e.upCh(ch)
		} else {
			out += e.loCh(ch)
		}
	}
	return out
}

func (e *pyStrEnv) swapCase(s string) string {
	out := ""
	for _, ch := range e.chars(s) {
		if e.isUpCh(ch) {
			out += e.loCh(ch)
		} else if e.isLoCh(ch) {
			out += e.upCh(ch)
		} else {
			out += ch
		}
	}
	return out
}

func (e *pyStrEnv) mapCase(s string, up bool) string {
	out := ""
	for _, ch := range e.chars(s) {
		if up {
			out += e.upCh(ch)
		} else {
			out += e.loCh(ch)
		}
	}
	return out
}

func (e *pyStrEnv) expandTabs(s string, tab int) string {
	out := ""
	col := 0
	n := e.sLen(s)
	for i := 0; i < n; {
		c := e.sCode(s, i)
		if c == 9 {
			if tab > 0 {
				pad := tab - (col - (col/tab)*tab)
				out += pysRepeat(" ", pad)
				col += pad
			}
			i++
		} else if c == 10 || c == 13 {
			out += e.sAt(s, i)
			col = 0
			i++
		} else {
			w := e.cpw(s, i)
			out += e.sSub(s, i, i+w)
			col++
			i += w
		}
	}
	return out
}

// ---- padding ----

func (e *pyStrEnv) center(s string, width int, fill string) string {
	n := e.cpLen(s)
	marg := width - n
	if marg <= 0 {
		return s
	}
	left := marg / 2
	if marg%2 == 1 && width%2 == 1 {
		left++
	}
	return pysRepeat(fill, left) + s + pysRepeat(fill, marg-left)
}

func (e *pyStrEnv) just(s string, width int, fill string, right bool) string {
	n := e.cpLen(s)
	if width <= n {
		return s
	}
	if right {
		return pysRepeat(fill, width-n) + s
	}
	return s + pysRepeat(fill, width-n)
}

func (e *pyStrEnv) zfill(s string, width int) string {
	n := e.cpLen(s)
	if width <= n {
		return s
	}
	sign, rest := "", s
	if e.sLen(s) > 0 {
		c := e.sCode(s, 0)
		if c == 43 || c == 45 {
			sign = e.sSub(s, 0, 1)
			rest = e.sSub(s, 1, e.sLen(s))
		}
	}
	return sign + pysRepeat("0", width-n) + rest
}

// ---- str.format ----

func (e *pyStrEnv) allDigits(t string) bool {
	n := e.sLen(t)
	if n == 0 {
		return false
	}
	for i := 0; i < n; i++ {
		c := e.sCode(t, i)
		if c < 48 || c > 57 {
			return false
		}
	}
	return true
}

func (e *pyStrEnv) kwGet(kw interface{}, nm string) interface{} {
	if keys, vals, ok := dictParts(kw); ok {
		if i := e.rt.pyDictFind(keys, nm); i >= 0 {
			return vals.elems[i]
		}
	}
	if o, ok := kw.(*jsObject); ok {
		if v, has := o.props[nm]; has {
			return v
		}
	}
	e.fail("KeyError: '" + nm + "'")
	return jsUndef
}

// One replacement field's body, without the braces. `auto` is the automatic field
// counter, threaded through so a nested width field takes the next argument.
func (e *pyStrEnv) fmtOne(body string, args []interface{}, kw interface{}, auto int) (string, int) {
	nm, conv, spec := "", "", ""
	part, depth := 0, 0
	n := e.sLen(body)
	for k := 0; k < n; k++ {
		c := e.sAt(body, k)
		if part < 2 {
			if c == "[" {
				depth++
			} else if c == "]" {
				depth--
			} else if depth == 0 && c == "!" && part == 0 && k+1 < n && e.sAt(body, k+1) != "=" {
				part = 1
				continue
			} else if depth == 0 && c == ":" {
				part = 2
				continue
			}
		}
		switch part {
		case 0:
			nm += c
		case 1:
			conv += c
		default:
			spec += c
		}
	}
	at := auto
	head := ""
	nl := e.sLen(nm)
	p := 0
	for p < nl && e.sAt(nm, p) != "." && e.sAt(nm, p) != "[" {
		head += e.sAt(nm, p)
		p++
	}
	var v interface{} = jsUndef
	if head == "" {
		if at >= len(args) {
			e.fail("IndexError: Replacement index " + strconv.Itoa(at) + " out of range")
		}
		v = args[at]
		at++
	} else if e.allDigits(head) {
		ix, _ := strconv.Atoi(head)
		if ix >= len(args) {
			e.fail("IndexError: Replacement index " + strconv.Itoa(ix) + " out of range")
		}
		v = args[ix]
	} else {
		v = e.kwGet(kw, head)
	}
	for p < nl {
		c := e.sAt(nm, p)
		if c == "." {
			an := ""
			p++
			for p < nl && e.sAt(nm, p) != "." && e.sAt(nm, p) != "[" {
				an += e.sAt(nm, p)
				p++
			}
			v = e.rt.pyGetAttr(v, an)
		} else if c == "[" {
			key := ""
			p++
			for p < nl && e.sAt(nm, p) != "]" {
				key += e.sAt(nm, p)
				p++
			}
			p++
			if e.allDigits(key) {
				ix, _ := strconv.Atoi(key)
				v = e.getIt(v, float64(ix))
			} else {
				v = e.getIt(v, key)
			}
		} else {
			e.fail("ValueError: Invalid format field: " + nm)
			p++
		}
	}
	if e.sIndex(spec, "{") >= 0 {
		spec, at = e.formatGo(spec, args, kw, at)
	}
	return e.fmt1(v, spec, conv), at
}

func (e *pyStrEnv) formatGo(s string, args []interface{}, kw interface{}, auto int) (string, int) {
	out := ""
	at := auto
	n := e.sLen(s)
	for i := 0; i < n; {
		ch := e.sAt(s, i)
		if ch == "{" {
			if i+1 < n && e.sAt(s, i+1) == "{" {
				out += "{"
				i += 2
				continue
			}
			depth, j := 1, i+1
			for ; j < n; j++ {
				c2 := e.sAt(s, j)
				if c2 == "{" {
					depth++
				} else if c2 == "}" {
					depth--
					if depth == 0 {
						break
					}
				}
			}
			if depth != 0 {
				e.fail("ValueError: Single '{' encountered in format string")
			}
			var text string
			text, at = e.fmtOne(e.sSub(s, i+1, j), args, kw, at)
			out += text
			i = j + 1
			continue
		}
		if ch == "}" {
			if i+1 < n && e.sAt(s, i+1) == "}" {
				out += "}"
				i += 2
				continue
			}
			e.fail("ValueError: Single '}' encountered in format string")
		}
		out += ch
		i++
	}
	return out, at
}

// ---- the dispatch table ----

func pysOptInt(rt *jsrt, args []interface{}, i, dflt int) int {
	if i >= len(args) {
		return dflt
	}
	v := args[i]
	if v == nil || v == jsUndef || v == jsNull {
		return dflt
	}
	return int(rt.toInt32(rt.toNumber(v)))
}

func (e *pyStrEnv) strArg(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	e.fail("TypeError: a str argument is required")
	return ""
}

func (e *pyStrEnv) fillArg(args []interface{}, i int) string {
	if i >= len(args) {
		return " "
	}
	v := args[i]
	if v == nil || v == jsUndef || v == jsNull {
		return " "
	}
	f := e.strArg(v)
	if e.cpLen(f) != 1 {
		e.fail("TypeError: The fill character must be exactly one character long")
	}
	return f
}

// isdigit / isalpha / isalnum / isspace: empty is False and every character must
// qualify.
func (e *pyStrEnv) allCh(s string, kind int) bool {
	cs := e.chars(s)
	if len(cs) == 0 {
		return false
	}
	for _, ch := range cs {
		ok := false
		switch kind {
		case 1:
			ok = e.isDigitCh(ch)
		case 2:
			ok = e.isAlphaCh(ch)
		case 3:
			ok = e.isAlphaCh(ch) || e.isDigitCh(ch)
		default:
			ok = e.isSpaceCh(ch)
		}
		if !ok {
			return false
		}
	}
	return true
}

// islower / isupper: at least one cased character and none of the other case.
func (e *pyStrEnv) caseAll(s string, up bool) bool {
	any := false
	for _, ch := range e.chars(s) {
		if e.isCased(ch) {
			if up {
				if !e.isUpCh(ch) {
					return false
				}
			} else if !e.isLoCh(ch) {
				return false
			}
			any = true
		}
	}
	return any
}

func (e *pyStrEnv) isAscii(s string) bool {
	n := e.sLen(s)
	for i := 0; i < n; i++ {
		if e.sCode(s, i) > 127 {
			return false
		}
	}
	return true
}

func (e *pyStrEnv) join(s string, seq []interface{}) string {
	out := ""
	for i, v := range seq {
		if i > 0 {
			out += s
		}
		out += e.strArg(v)
	}
	return out
}

// The names this table answers; the caller checks it FIRST so an unknown name
// keeps whatever behaviour it had.
func pysIsStrMethod(name string) bool {
	switch name {
	case "upper", "lower", "casefold", "title", "capitalize", "swapcase",
		"strip", "lstrip", "rstrip",
		"split", "rsplit", "splitlines", "join", "replace",
		"find", "rfind", "index", "rindex", "count", "startswith", "endswith",
		"isdigit", "isdecimal", "isnumeric", "isalpha", "isalnum", "isspace",
		"islower", "isupper", "istitle", "isascii",
		"center", "ljust", "rjust", "zfill", "expandtabs", "encode",
		"partition", "rpartition", "removeprefix", "removesuffix",
		"format", "format_map":
		return true
	}
	return false
}

func pysArgAt(args []interface{}, i int) interface{} {
	if i < len(args) {
		return args[i]
	}
	return jsUndef
}

func (e *pyStrEnv) call(s, name string, args []interface{}, kw interface{}) interface{} {
	n := e.sLen(s)
	rt := e.rt
	switch name {
	case "upper":
		return e.mapCase(s, true)
	case "lower", "casefold":
		return e.mapCase(s, false)
	case "title":
		return e.title(s)
	case "capitalize":
		return e.capitalize(s)
	case "swapcase":
		return e.swapCase(s)

	case "strip", "lstrip", "rstrip":
		chars, useWS := "", true
		if len(args) > 0 && args[0] != jsUndef && args[0] != jsNull && args[0] != nil {
			chars, useWS = e.strArg(args[0]), false
		}
		return e.strip(s, chars, useWS, name != "rstrip", name != "lstrip")

	case "split", "rsplit":
		ms := pysOptInt(rt, args, 1, -1)
		noSep := len(args) == 0
		if !noSep {
			noSep = args[0] == jsUndef || args[0] == jsNull || args[0] == nil
		}
		if noSep {
			if name == "split" {
				return &jsArray{elems: e.splitWS(s, ms)}
			}
			return &jsArray{elems: e.rsplitWS(s, ms)}
		}
		sep := e.strArg(args[0])
		if e.sLen(sep) == 0 {
			e.fail("ValueError: empty separator")
		}
		if name == "split" {
			return &jsArray{elems: e.splitSep(s, sep, ms)}
		}
		return &jsArray{elems: e.rsplitSep(s, sep, ms)}

	case "splitlines":
		ke := false
		if len(args) > 0 {
			ke = rt.truthy(args[0])
		}
		return &jsArray{elems: e.splitLines(s, ke)}

	case "join":
		switch sq := pysArgAt(args, 0).(type) {
		case string:
			cs := e.chars(sq)
			vs := make([]interface{}, 0, len(cs))
			for _, c := range cs {
				vs = append(vs, c)
			}
			return e.join(s, vs)
		case *jsArray:
			return e.join(s, sq.elems)
		}
		e.fail("TypeError: can only join an iterable")
		return jsUndef

	case "replace":
		return e.replace(s, e.strArg(pysArgAt(args, 0)), e.strArg(pysArgAt(args, 1)),
			pysOptInt(rt, args, 2, -1))

	case "find", "index", "rfind", "rindex":
		sub := e.strArg(pysArgAt(args, 0))
		st := pysAdjStart(pysOptInt(rt, args, 1, 0), n)
		en := pysAdjEnd(pysOptInt(rt, args, 2, n), n)
		at := -1
		if name == "find" || name == "index" {
			at = e.find(s, sub, st, en)
		} else {
			at = e.rfind(s, sub, st, en)
		}
		if at < 0 && (name == "index" || name == "rindex") {
			e.fail("ValueError: substring not found")
		}
		return float64(at)

	case "count":
		return float64(e.count(s, e.strArg(pysArgAt(args, 0)),
			pysAdjStart(pysOptInt(rt, args, 1, 0), n),
			pysAdjEnd(pysOptInt(rt, args, 2, n), n)))

	case "startswith", "endswith":
		st := pysAdjStart(pysOptInt(rt, args, 1, 0), n)
		en := pysAdjEnd(pysOptInt(rt, args, 2, n), n)
		hit := func(one string) bool {
			if name == "startswith" {
				return e.starts(s, one, st, en)
			}
			return e.ends(s, one, st, en)
		}
		if arr, ok := pysArgAt(args, 0).(*jsArray); ok {
			for _, p := range arr.elems {
				if hit(e.strArg(p)) {
					return true
				}
			}
			return false
		}
		return hit(e.strArg(pysArgAt(args, 0)))

	case "isdigit", "isdecimal", "isnumeric":
		return e.allCh(s, 1)
	case "isalpha":
		return e.allCh(s, 2)
	case "isalnum":
		return e.allCh(s, 3)
	case "isspace":
		return e.allCh(s, 4)
	case "islower":
		return e.caseAll(s, false)
	case "isupper":
		return e.caseAll(s, true)
	case "istitle":
		return e.isTitle(s)
	case "isascii":
		return e.isAscii(s)

	case "center":
		return e.center(s, pysOptInt(rt, args, 0, 0), e.fillArg(args, 1))
	case "ljust":
		return e.just(s, pysOptInt(rt, args, 0, 0), e.fillArg(args, 1), false)
	case "rjust":
		return e.just(s, pysOptInt(rt, args, 0, 0), e.fillArg(args, 1), true)
	case "zfill":
		return e.zfill(s, pysOptInt(rt, args, 0, 0))
	case "expandtabs":
		return e.expandTabs(s, pysOptInt(rt, args, 0, 8))
	case "encode":
		// No bytes type in this value model; CPython answers b'...'.
		return s

	case "partition", "rpartition":
		sep := e.strArg(pysArgAt(args, 0))
		if e.sLen(sep) == 0 {
			e.fail("ValueError: empty separator")
		}
		at := -1
		if name == "partition" {
			at = e.find(s, sep, 0, n)
		} else {
			at = e.rfind(s, sep, 0, n)
		}
		if at < 0 {
			if name == "partition" {
				return &jsArray{elems: []interface{}{s, "", ""}}
			}
			return &jsArray{elems: []interface{}{"", "", s}}
		}
		return &jsArray{elems: []interface{}{e.sSub(s, 0, at), sep, e.sSub(s, at+e.sLen(sep), n)}}

	case "removeprefix":
		pre := e.strArg(pysArgAt(args, 0))
		if e.sLen(pre) > 0 && e.starts(s, pre, 0, n) {
			return e.sSub(s, e.sLen(pre), n)
		}
		return s
	case "removesuffix":
		suf := e.strArg(pysArgAt(args, 0))
		if e.sLen(suf) > 0 && e.ends(s, suf, 0, n) {
			return e.sSub(s, 0, n-e.sLen(suf))
		}
		return s

	case "format":
		text, _ := e.formatGo(s, args, kw, 0)
		return text
	case "format_map":
		text, _ := e.formatGo(s, []interface{}{}, pysArgAt(args, 0), 0)
		return text
	}
	e.fail("unknown String method: " + name)
	return jsUndef
}

func init() {
	rxExtraExterns = append(rxExtraExterns, func(rt *jsrt, m map[string]func(args []uint64) uint64) {
		u := rt.unwrap
		w := rt.wrap
		getIt := m["js_pyget"]
		fmtIt := m["js_pyformat"]
		env := &pyStrEnv{
			rt: rt,
			getIt: func(v, k interface{}) interface{} {
				return u(getIt([]uint64{w(v), w(k)}))
			},
			fmt1: func(v interface{}, spec, conv string) string {
				return rt.toString(u(fmtIt([]uint64{w(v), w(spec), w(conv)})))
			},
		}
		// Both entry points are wrapped, so the order this registrar runs in
		// relative to jsrtregexpy.go's cannot matter: whichever wraps second sees
		// the other's function as its base, and a string is never an `re` object.
		wrap := func(base func([]uint64) uint64) func([]uint64) uint64 {
			if base == nil {
				return nil
			}
			return func(a []uint64) uint64 {
				if s, isStr := u(a[0]).(string); isStr {
					name := rt.toString(u(a[1]))
					if pysIsStrMethod(name) {
						arr, ok := u(a[2]).(*jsArray)
						if !ok {
							rt.fail("js_pymcall args must be an array")
						}
						return w(env.call(s, name, arr.elems, u(a[3])))
					}
				}
				return base(a)
			}
		}
		if f := wrap(m["js_pymcall"]); f != nil {
			m["js_pymcall"] = f
		}
		if f := wrap(m["js_pyrxmcall"]); f != nil {
			m["js_pyrxmcall"] = f
		}
	})
}
