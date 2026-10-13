package abnf

// Python's str METHOD LIBRARY for the Go twin - the third of the three engines
// that carry it.
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
// header and are NOT repeated here; the short form is: ASCII-exact classification with a cased-derived fallback outside
// it, and the case mapping of abnf/casemap.go.

import (
	"fmt"
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

func (e *pyStrEnv) sAt(s string, i int) string { return e.rt.strAt(s, i) }
func (e *pyStrEnv) sCode(s string, i int) int  { return e.rt.strCodeAt(s, i) }
func (e *pyStrEnv) sIndex(s, sub string) int   { return e.rt.strIndexOf(s, sub) }
func (e *pyStrEnv) fail(msg string)            { e.rt.fail("%s", msg) }

// raise is a catchable exception of class cls; raiseKey is the KeyError whose
// argument is the missing key itself. Twins: pySRaise and pySRaiseKey in
// languages/python-interpreter.abnf and languages/lib/python-rt.metajs.
func (e *pyStrEnv) raise(cls, msg string) {
	panic(&jsThrown{value: e.rt.pyExcInstance(cls, msg)})
}
func (e *pyStrEnv) raiseKey(nm string) {
	panic(&jsThrown{value: e.rt.pyExcInstanceV("KeyError", nm)})
}
func pysFromCode(c int) string { return strFromUnits([]uint16{uint16(c)}) }

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

// Python indexes a str by CODE POINT and the strings here are UTF-16: the unit
// index of code point cp, and back. A cp past the end stays as far past it, so a
// search window that starts beyond the string is still empty.
func (e *pyStrEnv) cpUnit(s string, cp int) int {
	if e.rt.strMeta(s).shape == 1 { // ASCII: a unit is a code point.
		return cp
	}
	i, n, k := 0, e.sLen(s), 0
	for ; k < cp && i < n; k++ {
		i += e.cpw(s, i)
	}
	return i + cp - k
}

func (e *pyStrEnv) unitCp(s string, u int) int {
	if e.rt.strMeta(s).shape == 1 {
		return u
	}
	k := 0
	for i := 0; i < u; k++ {
		i += e.cpw(s, i)
	}
	return k
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

// The SIMPLE per-character mapping (see the CASE MAPPING note in
// python-interpreter.abnf).
func (e *pyStrEnv) upCh(ch string) string {
	c := e.sCode(ch, 0)
	if c < 128 {
		if c >= 97 && c <= 122 {
			return pysFromCode(c - 32)
		}
		return ch
	}
	return caseMapMode(ch, caseUpper)
}

func (e *pyStrEnv) loCh(ch string) string {
	c := e.sCode(ch, 0)
	if c < 128 {
		if c >= 65 && c <= 90 {
			return pysFromCode(c + 32)
		}
		return ch
	}
	return caseMapMode(ch, 0)
}

// CASED CHARACTERS THE SIMPLE MAPPING LEAVES ALONE. CPython's cased / lower /
// upper tests read the Unicode Lowercase and Uppercase properties, and some
// characters carry one with no single-character mapping to tell: sharp s (whose
// upper case is "SS"), kra, the IPA and modifier letters, the mathematical
// alphanumerics. Without these, "straße".title() would start a word after the ß.
// [lo, hi] code point pairs, generated with CPython 3.14 (Unicode 16) over every
// code point Go's simple mapping fixes.
// Twin of pySLowOnly / pySUpOnly in languages/lib/python-rt.metajs and
// languages/python-interpreter.abnf.
var pysLowOnly = []int{
	170, 170, 186, 186, 223, 223, 312, 312, 329, 329, 397, 397, 426, 427, 442, 442, 446,
	446, 496, 496, 545, 545, 564, 569, 597, 597, 600, 600, 602, 602, 605, 607, 610, 610,
	615, 615, 621, 622, 624, 624, 627, 628, 630, 636, 638, 639, 641, 641, 644, 646, 653,
	657, 659, 659, 661, 668, 671, 696, 704, 705, 736, 740, 890, 890, 912, 912, 944, 944,
	1020, 1020, 1376, 1376, 1415, 1416, 4348, 4348, 7424, 7544, 7546, 7548, 7550, 7565,
	7567, 7615, 7830, 7834, 7836, 7837, 7839, 7839, 8016, 8016, 8018, 8018, 8020, 8020,
	8022, 8022, 8114, 8114, 8116, 8116, 8118, 8119, 8130, 8130, 8132, 8132, 8134, 8135,
	8146, 8147, 8150, 8151, 8162, 8164, 8166, 8167, 8178, 8178, 8180, 8180, 8182, 8183,
	8305, 8305, 8319, 8319, 8336, 8348, 8458, 8458, 8462, 8463, 8467, 8467, 8495, 8495,
	8500, 8500, 8505, 8505, 8508, 8509, 8518, 8521, 11377, 11377, 11380, 11380, 11383,
	11389, 11492, 11492, 42652, 42653, 42800, 42801, 42864, 42872, 42894, 42894, 42901,
	42901, 42927, 42927, 42994, 42996, 43000, 43002, 43824, 43858, 43860, 43866, 43868,
	43881, 64256, 64262, 64275, 64279, 67456, 67456, 67459, 67461, 67463, 67504, 67506,
	67514, 119834, 119859, 119886, 119892, 119894, 119911, 119938, 119963, 119990, 119993,
	119995, 119995, 119997, 120003, 120005, 120015, 120042, 120067, 120094, 120119,
	120146, 120171, 120198, 120223, 120250, 120275, 120302, 120327, 120354, 120379,
	120406, 120431, 120458, 120485, 120514, 120538, 120540, 120545, 120572, 120596,
	120598, 120603, 120630, 120654, 120656, 120661, 120688, 120712, 120714, 120719,
	120746, 120770, 120772, 120777, 120779, 120779, 122624, 122633, 122635, 122654,
	122661, 122666, 122928, 122989}
var pysUpOnly = []int{
	978, 980, 8450, 8450, 8455, 8455, 8459, 8461, 8464, 8466, 8469, 8469, 8473, 8477,
	8484, 8484, 8488, 8488, 8492, 8493, 8496, 8497, 8499, 8499, 8510, 8511, 8517, 8517,
	119808, 119833, 119860, 119885, 119912, 119937, 119964, 119964, 119966, 119967,
	119970, 119970, 119973, 119974, 119977, 119980, 119982, 119989, 120016, 120041,
	120068, 120069, 120071, 120074, 120077, 120084, 120086, 120092, 120120, 120121,
	120123, 120126, 120128, 120132, 120134, 120134, 120138, 120144, 120172, 120197,
	120224, 120249, 120276, 120301, 120328, 120353, 120380, 120405, 120432, 120457,
	120488, 120512, 120546, 120570, 120604, 120628, 120662, 120686, 120720, 120744,
	120778, 120778, 127280, 127305, 127312, 127337, 127344, 127369}

func pysInRanges(t []int, c int) bool {
	lo, hi := 0, len(t)/2-1
	for lo <= hi {
		mid := (lo + hi) / 2
		if c < t[2*mid] {
			hi = mid - 1
		} else if c > t[2*mid+1] {
			lo = mid + 1
		} else {
			return true
		}
	}
	return false
}

// The TITLECASE letters (category Lt): neither upper nor lower case to CPython,
// but they open a title-cased word the way an uppercase letter does.
var pysTitleOnly = []int{453, 453, 456, 456, 459, 459, 498, 498, 8072, 8079, 8088, 8095, 8104, 8111, 8124, 8124, 8140, 8140, 8188, 8188}

// 1 for a lowercase and 2 for an uppercase character of those tables, 3 for a
// titlecase one, else 0.
func (e *pyStrEnv) caseOnly(ch string) int {
	c := e.sCode(ch, 0)
	if c < 170 {
		return 0
	}
	if e.sLen(ch) == 2 {
		c = (c-55296)*1024 + (e.sCode(ch, 1) - 56320) + 65536
	}
	if pysInRanges(pysTitleOnly, c) {
		return 3
	}
	if pysInRanges(pysLowOnly, c) {
		return 1
	}
	if pysInRanges(pysUpOnly, c) {
		return 2
	}
	return 0
}

func (e *pyStrEnv) isCased(ch string) bool { return e.upCh(ch) != e.loCh(ch) || e.caseOnly(ch) != 0 }

// lower_ucs4: ch's full lowercase in its string, where low is
// caseMapMode(s, caseFull|caseSigma) and at the byte offset where ch's
// lowercase starts in it. Twin of pySLowAt in python-interpreter.abnf.
func pyLowAt(low string, at int, ch string) string {
	return low[at : at+len(caseMapMode(ch, caseFull))]
}

func (e *pyStrEnv) isUpCh(ch string) bool {
	if k := e.caseOnly(ch); k != 0 {
		return k == 2
	}
	u := e.upCh(ch)
	return u != e.loCh(ch) && ch == u
}

func (e *pyStrEnv) isLoCh(ch string) bool {
	if k := e.caseOnly(ch); k != 0 {
		return k == 1
	}
	l := e.loCh(ch)
	return e.upCh(ch) != l && ch == l
}

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

// Each end is tested a whole code point at a time, so an astral character in
// `chars` strips as one character and never splits a pair.
func (e *pyStrEnv) strip(s, chars string, useWS, doL, doR bool) string {
	a, b := 0, e.sLen(s)
	if doL {
		for a < b && e.stripHit(e.sSub(s, a, a+e.cpw(s, a)), chars, useWS) {
			a += e.cpw(s, a)
		}
	}
	if doR {
		for b > a && e.stripHit(e.sSub(s, b-e.cpwBack(s, a, b), b), chars, useWS) {
			b -= e.cpwBack(s, a, b)
		}
	}
	return e.sSub(s, a, b)
}

// cpwBack is the unit width of the code point ENDING at unit b, not reaching below a.
func (e *pyStrEnv) cpwBack(s string, a, b int) int {
	if b-2 >= a && e.cpw(s, b-2) == 2 {
		return 2
	}
	return 1
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
	low := caseMapMode(s, caseFull|caseSigma)
	at := 0
	out := ""
	prev := false
	for _, ch := range e.chars(s) {
		if prev {
			out += pyLowAt(low, at, ch)
		} else {
			out += caseMapMode(ch, caseTitle|caseFull)
		}
		at += len(caseMapMode(ch, caseFull))
		prev = e.isCased(ch)
	}
	return out
}

func (e *pyStrEnv) isTitle(s string) bool {
	prev, any := false, false
	for _, ch := range e.chars(s) {
		if e.isUpCh(ch) || e.caseOnly(ch) == 3 {
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
	cs := e.chars(s)
	if len(cs) == 0 {
		return s
	}
	return caseMapMode(cs[0], caseTitle|caseFull) + caseMapMode(s, caseFull|caseSigma)[len(caseMapMode(cs[0], caseFull)):]
}

func (e *pyStrEnv) swapCase(s string) string {
	low := caseMapMode(s, caseFull|caseSigma)
	at := 0
	out := ""
	for _, ch := range e.chars(s) {
		if e.isUpCh(ch) {
			out += pyLowAt(low, at, ch)
		} else if e.isLoCh(ch) {
			out += caseMapMode(ch, caseUpper|caseFull)
		} else {
			out += ch
		}
		at += len(caseMapMode(ch, caseFull))
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
	e.raiseKey(nm)
	return jsUndef
}

// badConv is do_conversion's refusal of a conversion other than r, s and a: a
// printable ASCII character is named as itself, anything else as \x and its hex
// code. Twin of pySBadConv.
func (e *pyStrEnv) badConv(conv string) {
	cp := e.sCode(conv, 0)
	if e.sLen(conv) == 2 {
		cp = (cp-55296)*1024 + e.sCode(conv, 1) - 56320 + 65536
	}
	name := conv
	if cp <= 32 || cp >= 127 {
		name = "\\x" + strconv.FormatInt(int64(cp), 16)
	}
	e.raise("ValueError", "Unknown conversion specifier "+name)
}

// fmtOne renders one parsed replacement field: its name, conversion ("" for
// none) and spec. `auto` is the automatic field counter, threaded through so a
// nested width field takes the next argument, and -1 once a field was numbered
// by hand. Twin of pySFmtOne.
func (e *pyStrEnv) fmtOne(nm, conv, spec string, args []interface{}, kw interface{}, auto, depth int) (string, int) {
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
		if at < 0 {
			e.raise("ValueError", "cannot switch from manual field specification to automatic field numbering")
		}
		if at >= len(args) {
			e.raise("IndexError", "Replacement index "+strconv.Itoa(at)+" out of range for positional args tuple")
		}
		v = args[at]
		at++
	} else if e.allDigits(head) {
		if at > 0 {
			e.raise("ValueError", "cannot switch from automatic field numbering to manual field specification")
		}
		at = -1
		ix, _ := strconv.Atoi(head)
		if ix >= len(args) {
			e.raise("IndexError", "Replacement index "+strconv.Itoa(ix)+" out of range for positional args tuple")
		}
		v = args[ix]
	} else {
		v = e.kwGet(kw, head)
	}
	for p < nl {
		c := e.sAt(nm, p)
		key := ""
		p++
		if c == "." {
			for p < nl && e.sAt(nm, p) != "." && e.sAt(nm, p) != "[" {
				key += e.sAt(nm, p)
				p++
			}
		} else if c == "[" {
			for p < nl && e.sAt(nm, p) != "]" {
				key += e.sAt(nm, p)
				p++
			}
			p++
		} else {
			e.raise("ValueError", "Only '.' or '[' may follow ']' in format field specifier")
		}
		if key == "" {
			e.raise("ValueError", "Empty attribute in format string")
		}
		if c == "." {
			v = e.rt.pyGetAttr(v, key)
		} else if e.allDigits(key) {
			ix, _ := strconv.Atoi(key)
			v = e.getIt(v, float64(ix))
		} else {
			v = e.getIt(v, key)
		}
	}
	if conv != "" && conv != "r" && conv != "s" && conv != "a" {
		e.badConv(conv)
	}
	if e.sIndex(spec, "{") >= 0 {
		spec, at = e.formatGo(spec, args, kw, at, depth-1)
	}
	return e.fmt1(v, spec, conv), at
}

// formatGo is CPython's MarkupIterator_next and parse_field; `depth` is
// build_string's recursion budget. Twin of pySFormatGo.
func (e *pyStrEnv) formatGo(s string, args []interface{}, kw interface{}, auto, depth int) (string, int) {
	if depth <= 0 {
		e.raise("ValueError", "Max string recursion exceeded")
	}
	out := ""
	at := auto
	n := e.sLen(s)
	for i := 0; i < n; {
		ch := e.sAt(s, i)
		if ch == "}" {
			if i+1 < n && e.sAt(s, i+1) == "}" {
				out += "}"
				i += 2
				continue
			}
			e.raise("ValueError", "Single '}' encountered in format string")
		}
		if ch != "{" {
			out += ch
			i++
			continue
		}
		if i+1 >= n {
			e.raise("ValueError", "Single '{' encountered in format string")
		}
		if e.sAt(s, i+1) == "{" {
			out += "{"
			i += 2
			continue
		}
		j, c := i+1, ""
		for j < n {
			c = e.sAt(s, j)
			j++
			if c == "{" {
				e.raise("ValueError", "unexpected '{' in field name")
			}
			if c == "[" {
				for j < n && e.sAt(s, j) != "]" {
					j++
				}
				continue
			}
			if c == "}" || c == ":" || c == "!" {
				break
			}
		}
		nm := e.sSub(s, i+1, j-1)
		conv, spec := "", ""
		if c == "!" || c == ":" {
			closed := false
			if c == "!" {
				if j >= n {
					e.raise("ValueError", "end of string while looking for conversion specifier")
				}
				w := e.cpw(s, j)
				conv = e.sSub(s, j, j+w)
				j += w
				if j < n {
					c3 := e.sAt(s, j)
					j++
					if c3 == "}" {
						closed = true
					} else if c3 != ":" {
						e.raise("ValueError", "expected ':' after conversion specifier")
					}
				}
			}
			if !closed {
				from, d := j, 1
				for j < n && d > 0 {
					c4 := e.sAt(s, j)
					j++
					if c4 == "{" {
						d++
					} else if c4 == "}" {
						d--
					}
				}
				if d > 0 {
					e.raise("ValueError", "unmatched '{' in format spec")
				}
				spec = e.sSub(s, from, j-1)
			}
		} else if c != "}" {
			e.raise("ValueError", "expected '}' before end of string")
		}
		var text string
		text, at = e.fmtOne(nm, conv, spec, args, kw, at, depth)
		out += text
		i = j
	}
	return out, at
}

// ---- the dispatch table ----

// intArg is an integer argument (an int, a bool, or an int subclass's value),
// dflt when it is absent. A slice window (find's start and end) also takes None
// for its default; anywhere else None is refused like any other non-integer,
// each with CPython's own TypeError. Twin of pySIntArg.
func (e *pyStrEnv) intArg(args []interface{}, i, dflt int, slice bool) int {
	if i >= len(args) {
		return dflt
	}
	v := pyUnwrapBuiltin(args[i])
	if v == nil || v == jsUndef || v == jsNull {
		if slice {
			return dflt
		}
	} else if tn := pyTypeName(e.rt, v); tn == "int" || tn == "bool" {
		return int(e.rt.toInt32(e.rt.toNumber(v)))
	}
	if slice {
		e.raise("TypeError", "slice indices must be integers or None or have an __index__ method")
	}
	e.raise("TypeError", "'"+pyTypeName(e.rt, args[i])+"' object cannot be interpreted as an integer")
	return dflt
}

// strArg is a str argument (a str or a str subclass's value); anything else
// raises TypeError(what + ", not " + its type). An Argument Clinic parameter
// (clinic) spells None's type "None" where the others say "NoneType". Twin of
// pySStrArg.
func (e *pyStrEnv) strArg(v interface{}, what string, clinic bool) string {
	if s, ok := pyUnwrapBuiltin(v).(string); ok {
		return s
	}
	tn := pyTypeName(e.rt, v)
	if clinic && tn == "NoneType" {
		tn = "None"
	}
	e.raise("TypeError", what+", not "+tn)
	return ""
}

func (e *pyStrEnv) fillArg(args []interface{}, i int) string {
	if i >= len(args) {
		return " "
	}
	f := e.strArg(args[i], "The fill character must be a unicode character", false)
	if e.cpLen(f) != 1 {
		e.raise("TypeError", "The fill character must be exactly one character long")
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
		u, ok := pyUnwrapBuiltin(v).(string)
		if !ok {
			e.raise("TypeError", fmt.Sprintf("sequence item %d: expected str instance, %s found", i, pyTypeName(e.rt, v)))
		}
		out += u
	}
	return out
}

// arity raises CPython's arity errors, in the wording of each calling
// convention: a method with no parameters, a single-argument one, and Argument
// Clinic's positional-only ("expected at least") and keyword-capable ("takes at
// most") forms. format takes any number. Twin of pySArity.
func (e *pyStrEnv) arity(name string, n int) {
	lo, hi, style := 0, 0, 0
	switch name {
	case "format":
		return
	case "join", "zfill", "partition", "rpartition", "removeprefix", "removesuffix", "format_map", "translate":
		lo, hi, style = 1, 1, 1
	case "strip", "lstrip", "rstrip":
		hi, style = 1, 2
	case "find", "rfind", "index", "rindex", "count", "startswith", "endswith":
		lo, hi, style = 1, 3, 2
	case "center", "ljust", "rjust":
		lo, hi, style = 1, 2, 2
	case "split", "rsplit", "encode":
		hi, style = 2, 3
	case "splitlines", "expandtabs":
		hi, style = 1, 3
	case "replace":
		lo, hi, style = 2, 3, 3
	}
	if n >= lo && n <= hi {
		return
	}
	plural := func(k int) string {
		if k == 1 {
			return ""
		}
		return "s"
	}
	msg := ""
	switch {
	case style == 0:
		msg = fmt.Sprintf("str.%s() takes no arguments (%d given)", name, n)
	case style == 1:
		msg = fmt.Sprintf("str.%s() takes exactly one argument (%d given)", name, n)
	case style == 2 && n < lo:
		msg = fmt.Sprintf("%s expected at least %d argument%s, got %d", name, lo, plural(lo), n)
	case style == 2:
		msg = fmt.Sprintf("%s expected at most %d argument%s, got %d", name, hi, plural(hi), n)
	case n < lo:
		msg = fmt.Sprintf("%s() takes at least %d positional arguments (%d given)", name, lo, n)
	default:
		msg = fmt.Sprintf("%s() takes at most %d argument%s (%d given)", name, hi, plural(hi), n)
	}
	e.raise("TypeError", msg)
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
		"islower", "isupper", "istitle", "isascii", "isprintable", "isidentifier", "translate", "maketrans",
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

// pysKwArgs moves the keyword arguments of the str methods that take any into
// their slots, under CPython 3.14's parameter names; every other one is
// positional-only. Twin of pyStrKwArgs in languages/python-interpreter.abnf and
// in lib/python-rt.metajs.
func (rt *jsrt) pysKwArgs(name string, args []interface{}, kw interface{}) []interface{} {
	switch name {
	case "format":
		return args
	case "split", "rsplit":
		return rt.pyKwPos(name, args, kw, []string{"sep", "maxsplit"}, 0)
	case "splitlines":
		return rt.pyKwPos(name, args, kw, []string{"keepends"}, 0)
	case "expandtabs":
		return rt.pyKwPos(name, args, kw, []string{"tabsize"}, 0)
	case "encode":
		return rt.pyKwPos(name, args, kw, []string{"encoding", "errors"}, 0)
	case "replace":
		if len(args) < 2 {
			panic(&jsThrown{value: rt.pyExcInstance("TypeError",
				fmt.Sprintf("replace() takes at least 2 positional arguments (%d given)", len(args)))})
		}
		return rt.pyKwPos(name, args, kw, []string{"old", "new", "count"}, 2)
	}
	panic(&jsThrown{value: rt.pyExcInstance("TypeError", "str."+name+"() takes no keyword arguments")})
}

// pysEncode is str.encode(encoding="utf-8", errors="strict"): the bytes - a list
// of byte values in this value model - in UTF-8, ASCII or Latin-1. A run of
// characters the codec cannot encode raises CPython's UnicodeEncodeError under
// "strict"; "ignore", "replace", "backslashreplace" and "xmlcharrefreplace"
// substitute. Twin of pyStrEncode in languages/python-interpreter.abnf and in
// lib/python-rt.metajs.
func (rt *jsrt) pysEncode(s string, encV, errV interface{}) interface{} {
	enc, errs := "utf-8", "strict"
	if encV != nil && encV != jsUndef && encV != jsNull {
		enc = rt.toString(encV)
	}
	if errV != nil && errV != jsUndef && errV != jsNull {
		errs = rt.toString(errV)
	}
	limit, codec := 0, "utf-8"
	switch strings.ReplaceAll(strings.ToLower(enc), "_", "-") {
	case "latin-1", "latin1", "iso-8859-1", "iso8859-1":
		limit, codec = 256, "latin-1"
	case "ascii", "us-ascii":
		limit, codec = 128, "ascii"
	case "utf-8", "utf8":
	default:
		panic(&jsThrown{value: rt.pyExcInstance("LookupError", "unknown encoding: "+enc)})
	}
	units := strUnits(s)
	var cps []int
	for i := 0; i < len(units); i++ {
		c := int(units[i])
		if c >= 0xd800 && c <= 0xdbff && i+1 < len(units) && units[i+1] >= 0xdc00 && units[i+1] <= 0xdfff {
			c = (c-0xd800)*1024 + int(units[i+1]) - 0xdc00 + 0x10000
			i++
		}
		cps = append(cps, c)
	}
	bad := func(cp int) bool {
		if limit == 0 {
			return cp >= 0xd800 && cp <= 0xdfff
		}
		return cp >= limit
	}
	out := &jsArray{kind: pyBytesKind}
	push := func(b int) { out.elems = append(out.elems, float64(b)) }
	ascii := func(t string) {
		for i := 0; i < len(t); i++ {
			push(int(t[i]))
		}
	}
	for k := 0; k < len(cps); {
		cp := cps[k]
		if !bad(cp) {
			switch {
			case limit != 0 || cp < 0x80:
				push(cp)
			case cp < 0x800:
				push(0xc0 | cp>>6)
				push(0x80 | cp&63)
			case cp < 0x10000:
				push(0xe0 | cp>>12)
				push(0x80 | cp>>6&63)
				push(0x80 | cp&63)
			default:
				push(0xf0 | cp>>18)
				push(0x80 | cp>>12&63)
				push(0x80 | cp>>6&63)
				push(0x80 | cp&63)
			}
			k++
			continue
		}
		j := k
		for j < len(cps) && bad(cps[j]) {
			j++
		}
		if errs == "strict" {
			where := fmt.Sprintf("character '%s' in position %d", pysEncEsc(cp), k)
			if j-k > 1 {
				where = fmt.Sprintf("characters in position %d-%d", k, j-1)
			}
			why := fmt.Sprintf("ordinal not in range(%d)", limit)
			if limit == 0 {
				why = "surrogates not allowed"
			}
			panic(&jsThrown{value: rt.pyExcInstance("UnicodeEncodeError", "'"+codec+"' codec can't encode "+where+": "+why)})
		}
		for q := k; q < j; q++ {
			switch errs {
			case "replace":
				push('?')
			case "backslashreplace":
				ascii(pysEncEsc(cps[q]))
			case "xmlcharrefreplace":
				ascii(fmt.Sprintf("&#%d;", cps[q]))
			case "ignore":
			default:
				panic(&jsThrown{value: rt.pyExcInstance("LookupError", "unknown error handler name '"+errs+"'")})
			}
		}
		k = j
	}
	return out
}

// pysEncEsc is a code point as repr spells it inside a string.
func pysEncEsc(cp int) string {
	switch {
	case cp < 0x100:
		return fmt.Sprintf("\\x%02x", cp)
	case cp < 0x10000:
		return fmt.Sprintf("\\u%04x", cp)
	}
	return fmt.Sprintf("\\U%08x", cp)
}

// isIdent is str.isidentifier: an XID_Start code point or "_" first, then
// XID_Continue ones (pyIsXidStart / pyIsXidCont). Twin of pySIsIdent in
// languages/python-interpreter.abnf and lib/python-rt.metajs.
func (e *pyStrEnv) isIdent(s string) bool {
	cs := e.chars(s)
	if len(cs) == 0 {
		return false
	}
	for i, ch := range cs {
		c := wtf8Surrogate(ch)
		if c == 0 {
			c = []rune(ch)[0]
		}
		ok := (i == 0 && pyIsXidStart(c)) || (i > 0 && pyIsXidCont(c))
		if !ok {
			return false
		}
	}
	return true
}

// translate is str.translate(table): table[ord(c)] for each character, so any
// mapping or sequence serves - a dict is read directly, anything else is
// subscripted, and a LookupError (a missing key, an index past a list) keeps
// the character. A str value replaces it, an int is a code point, None deletes
// it. Twin of pySTranslate in languages/python-interpreter.abnf and
// lib/python-rt.metajs.
func (e *pyStrEnv) translate(s string, table interface{}) string {
	keys, vals, plain := dictParts(table)
	if _, isSet := pySetElems(table); isSet {
		plain = false
	}
	out := ""
	for _, ch := range e.chars(s) {
		cp := e.rt.pyExt("js_pyord", ch)
		var v interface{}
		if plain {
			at := e.rt.pyDictFind(keys, cp)
			if at < 0 {
				out += ch
				continue
			}
			v = vals.elems[at]
		} else if got, ok := e.transGet(table, cp); ok {
			v = got
		} else {
			out += ch
			continue
		}
		switch v := v.(type) {
		case jsUndefT, jsNullT:
		case string:
			out += v
		case float64, bool, *jsBigInt:
			cp := e.rt.toNumber(v)
			if cp < 0 || cp > 0x10ffff {
				e.raise("ValueError", "character mapping must be in range(0x110000)")
			}
			out += string(rune(int(cp)))
		default:
			e.raise("TypeError", "character mapping must return integer, None or str")
		}
	}
	return out
}

// transGet is table[cp], ok false when it raises LookupError.
func (e *pyStrEnv) transGet(table interface{}, cp interface{}) (v interface{}, ok bool) {
	defer func() {
		if p := recover(); p != nil {
			if th, isT := p.(*jsThrown); isT && e.rt.pyIsInstance(th.value, e.rt.pyExcClass("LookupError")) {
				v, ok = nil, false
				return
			}
			panic(p)
		}
	}()
	return e.rt.pyExt("js_pyidx", table, cp), true
}

// makeTrans is str.maketrans(x[, y[, z]]): a dict of code points. One argument
// is a dict whose str keys of length 1 become their code points; two
// equal-length strs map each character of x to the one at the same place in y,
// and z's characters map to None. Argument Clinic checks y and z before x.
// Twin of pySMakeTrans in languages/python-interpreter.abnf and
// lib/python-rt.metajs.
func (e *pyStrEnv) makeTrans(args []interface{}) interface{} {
	rt := e.rt
	n := len(args)
	if n == 0 {
		e.raise("TypeError", "maketrans expected at least 1 argument, got 0")
	}
	if n > 3 {
		e.raise("TypeError", fmt.Sprintf("maketrans expected at most 3 arguments, got %d", n))
	}
	keys, vals := &jsArray{}, &jsArray{}
	put := func(k, v interface{}) {
		if i := rt.pyDictFind(keys, k); i >= 0 {
			vals.elems[i] = v
			return
		}
		dictAppend(keys, vals, k, v)
	}
	if n == 1 {
		xk, xv, ok := dictParts(args[0])
		if _, isSet := pySetElems(args[0]); !ok || isSet {
			e.raise("TypeError", "if you give only one argument to maketrans it must be a dict")
		}
		for i, k := range xk.elems {
			if ks, isStr := k.(string); isStr {
				if len(e.chars(ks)) != 1 {
					e.raise("ValueError", "string keys in translatetable must be of length 1")
				}
				put(rt.pyExt("js_pyord", ks), xv.elems[i])
			} else if pyNumTypeName(k) == "int" {
				put(k, xv.elems[i])
			} else {
				e.raise("TypeError", "keys in translate table mustbe strings or integers")
			}
		}
	} else {
		for i := 1; i < n; i++ {
			if _, isStr := args[i].(string); !isStr {
				e.raise("TypeError", fmt.Sprintf("maketrans() argument %d must be str, not %s", i+1, pyTypeName(rt, args[i])))
			}
		}
		x, isStr := args[0].(string)
		if !isStr {
			e.raise("TypeError", "first maketrans argument must be a string if there is a second argument")
		}
		xs, ys := e.chars(x), e.chars(args[1].(string))
		if len(xs) != len(ys) {
			e.raise("ValueError", "the first two maketrans arguments must have equal length")
		}
		for i := range xs {
			put(rt.pyExt("js_pyord", xs[i]), rt.pyExt("js_pyord", ys[i]))
		}
		if n == 3 {
			for _, z := range e.chars(args[2].(string)) {
				put(rt.pyExt("js_pyord", z), jsUndef)
			}
		}
	}
	return &jsObject{props: map[string]interface{}{"__dict": true, "keys": keys, "vals": vals}}
}

func (e *pyStrEnv) call(s, name string, args []interface{}, kw interface{}) interface{} {
	n := e.sLen(s)
	rt := e.rt
	switch name {
	case "upper":
		return caseMapMode(s, caseUpper|caseFull)
	case "lower":
		return caseMapMode(s, caseFull|caseSigma)
	case "casefold":
		return caseMapMode(s, caseFold|caseFull)
	case "title":
		return e.title(s)
	case "capitalize":
		return e.capitalize(s)
	case "swapcase":
		return e.swapCase(s)

	case "strip", "lstrip", "rstrip":
		chars, useWS := "", true
		if len(args) > 0 && args[0] != jsUndef && args[0] != jsNull && args[0] != nil {
			cu, ok := pyUnwrapBuiltin(args[0]).(string)
			if !ok {
				e.raise("TypeError", name+" arg must be None or str")
			}
			chars, useWS = cu, false
		}
		return e.strip(s, chars, useWS, name != "rstrip", name != "lstrip")

	case "split", "rsplit":
		ms := e.intArg(args, 1, -1, false)
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
		sep := e.strArg(args[0], "must be str or None", false)
		if e.sLen(sep) == 0 {
			e.raise("ValueError", "empty separator")
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
		// A GENERATOR or an ITERATOR object (iter/map/filter/zip/enumerate) is
		// materialized rather than refused - `", ".join(map(str,
		// xs))` is ordinary Python. Keep in step with the join arm of pyStrMethod
		// in languages/lib/python-rt.metajs and languages/python-interpreter.abnf.
		if a := pysArgAt(args, 0); pyIterName(e.rt, a) != "" || pyIsGenerator(a) || pyIsDView(a) {
			return e.join(s, e.rt.pyElemsOf(a))
		}
		e.raise("TypeError", "can only join an iterable")
		return jsUndef

	case "replace":
		return e.replace(s, e.strArg(pysArgAt(args, 0), "replace() argument 1 must be str", true),
			e.strArg(pysArgAt(args, 1), "replace() argument 2 must be str", true), e.intArg(args, 2, -1, false))

	// The window arguments and the answer are CODE POINT positions; the search
	// itself runs on units.
	case "find", "index", "rfind", "rindex":
		cn := e.cpLen(s)
		sub := e.strArg(pysArgAt(args, 0), name+"() argument 1 must be str", true)
		st := e.cpUnit(s, pysAdjStart(e.intArg(args, 1, 0, true), cn))
		en := e.cpUnit(s, pysAdjEnd(e.intArg(args, 2, cn, true), cn))
		at := -1
		if name == "find" || name == "index" {
			at = e.find(s, sub, st, en)
		} else {
			at = e.rfind(s, sub, st, en)
		}
		if at < 0 {
			if name == "index" || name == "rindex" {
				e.raise("ValueError", "substring not found")
			}
			return float64(at)
		}
		return float64(e.unitCp(s, at))

	case "count":
		cn := e.cpLen(s)
		return float64(e.count(s, e.strArg(pysArgAt(args, 0), "count() argument 1 must be str", true),
			e.cpUnit(s, pysAdjStart(e.intArg(args, 1, 0, true), cn)),
			e.cpUnit(s, pysAdjEnd(e.intArg(args, 2, cn, true), cn))))

	case "startswith", "endswith":
		cn := e.cpLen(s)
		st := e.cpUnit(s, pysAdjStart(e.intArg(args, 1, 0, true), cn))
		en := e.cpUnit(s, pysAdjEnd(e.intArg(args, 2, cn, true), cn))
		hit := func(one string) bool {
			if name == "startswith" {
				return e.starts(s, one, st, en)
			}
			return e.ends(s, one, st, en)
		}
		if arr, ok := pysArgAt(args, 0).(*jsArray); ok {
			for _, p := range arr.elems {
				if hit(e.strArg(p, "tuple for "+name+" must only contain str", false)) {
					return true
				}
			}
			return false
		}
		return hit(e.strArg(pysArgAt(args, 0), name+" first arg must be str or a tuple of str", false))

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
	case "isprintable":
		return pyStrIsPrintable(s)
	case "isidentifier":
		return e.isIdent(s)
	case "translate":
		return e.translate(s, pysArgAt(args, 0))

	case "center":
		return e.center(s, e.intArg(args, 0, 0, false), e.fillArg(args, 1))
	case "ljust":
		return e.just(s, e.intArg(args, 0, 0, false), e.fillArg(args, 1), false)
	case "rjust":
		return e.just(s, e.intArg(args, 0, 0, false), e.fillArg(args, 1), true)
	case "zfill":
		return e.zfill(s, e.intArg(args, 0, 0, false))
	case "expandtabs":
		return e.expandTabs(s, e.intArg(args, 0, 8, false))
	case "encode":
		return rt.pysEncode(s, pysArgAt(args, 0), pysArgAt(args, 1))

	case "partition", "rpartition":
		sep := e.strArg(pysArgAt(args, 0), "must be str", false)
		if e.sLen(sep) == 0 {
			e.raise("ValueError", "empty separator")
		}
		at := -1
		if name == "partition" {
			at = e.find(s, sep, 0, n)
		} else {
			at = e.rfind(s, sep, 0, n)
		}
		if at < 0 {
			if name == "partition" {
				return pyTup(&jsArray{elems: []interface{}{s, "", ""}})
			}
			return pyTup(&jsArray{elems: []interface{}{"", "", s}})
		}
		return pyTup(&jsArray{elems: []interface{}{e.sSub(s, 0, at), sep, e.sSub(s, at+e.sLen(sep), n)}})

	case "removeprefix":
		pre := e.strArg(pysArgAt(args, 0), "removeprefix() argument must be str", true)
		if e.sLen(pre) > 0 && e.starts(s, pre, 0, n) {
			return e.sSub(s, e.sLen(pre), n)
		}
		return s
	case "removesuffix":
		suf := e.strArg(pysArgAt(args, 0), "removesuffix() argument must be str", true)
		if e.sLen(suf) > 0 && e.ends(s, suf, 0, n) {
			return e.sSub(s, 0, n-e.sLen(suf))
		}
		return s

	case "format":
		text, _ := e.formatGo(s, args, kw, 0, 2)
		return text
	case "format_map":
		text, _ := e.formatGo(s, []interface{}{}, pysArgAt(args, 0), 0, 2)
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
						args, kw := arr.elems, u(a[3])
						keys, _, isDict := dictParts(kw)
						if name == "maketrans" { // A static method: the receiver plays no part.
							if isDict && len(keys.elems) > 0 {
								env.raise("TypeError", "str.maketrans() takes no keyword arguments")
							}
							return w(env.makeTrans(args))
						}
						if isDict && len(keys.elems) > 0 {
							args = rt.pysKwArgs(name, args, kw)
						} else {
							env.arity(name, len(args))
						}
						return w(env.call(s, name, args, kw))
					}
				}
				return base(a)
			}
		}
		// js_pyidx is python's subscript: js_pyget, except that a str is indexed
		// by CODE POINT. js_pyget stays UTF-16 for dart and csharp. Twin of
		// js_pyidx in lib/python-rt.metajs.
		get := m["js_pyget"]
		m["js_pyidx"] = func(a []uint64) uint64 {
			s, isStr := u(a[0]).(string)
			if !isStr {
				return get(a)
			}
			i := rt.pySeqIdx(s, u(a[1]))
			if i < 0 {
				i += pyStrLen(rt, s)
			}
			at := env.cpUnit(s, i)
			if i < 0 || at >= env.sLen(s) {
				panic(&jsThrown{value: rt.pyExcInstance("IndexError", "string index out of range")})
			}
			return rt.wrapStr(env.sSub(s, at, at+env.cpw(s, at)))
		}
		if f := wrap(m["js_pymcall"]); f != nil {
			m["js_pymcall"] = f
		}
		if f := wrap(m["js_pyrxmcall"]); f != nil {
			m["js_pyrxmcall"] = f
		}
	})
}
