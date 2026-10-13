package abnf

import (
	"math"
	"strconv"
	"strings"
)

// java.lang.String's methods and String.format: the Go twin's port of
// languages/lib/java-strings.js, whose header has the rules. The strings are
// worked on as UTF-16 units, as the MetaJS hosts see them; the value-model hooks
// (jxk* there) are the jx*Arg methods at the end of this file.

func jxIsWs(c int) bool {
	return (c >= 9 && c <= 13) || (c >= 28 && c <= 32) || c == 5760 || (c >= 8192 && c <= 8198) ||
		(c >= 8200 && c <= 8202) || c == 8232 || c == 8233 || c == 8287 || c == 12288
}

// The simple mapping (lib/utf16-strings.js); java's String.toUpperCase /
// toLowerCase map fully, at their own arms.
func jxUpper(s string) string { return caseMapMode(s, caseUpper) }
func jxLower(s string) string { return caseMapMode(s, 0) }

func jxTrim(s string) string {
	u := strUnits(s)
	b, e := 0, len(u)
	for b < e && u[b] <= 32 {
		b++
	}
	for e > b && u[e-1] <= 32 {
		e--
	}
	return strFromUnits(u[b:e])
}

func jxStrip(s string, lead, trail bool) string {
	u := strUnits(s)
	b, e := 0, len(u)
	if lead {
		for b < e && jxIsWs(int(u[b])) {
			b++
		}
	}
	if trail {
		for e > b && jxIsWs(int(u[e-1])) {
			e--
		}
	}
	return strFromUnits(u[b:e])
}

func jxAt(s, t []uint16, i int) bool {
	if i < 0 || i+len(t) > len(s) {
		return false
	}
	for k := range t {
		if s[i+k] != t[k] {
			return false
		}
	}
	return true
}

func jxIndexOf(s, t []uint16, from int) int {
	i := from
	if i < 0 {
		i = 0
	}
	if i > len(s) {
		i = len(s)
	}
	for ; i+len(t) <= len(s); i++ {
		if jxAt(s, t, i) {
			return i
		}
	}
	return -1
}

func jxLastIndexOf(s, t []uint16, from int) int {
	i := len(s) - len(t)
	if from < i {
		i = from
	}
	for ; i >= 0; i-- {
		if jxAt(s, t, i) {
			return i
		}
	}
	return -1
}

func jxReplace(s, t, r string) string {
	su, tu, ru := strUnits(s), strUnits(t), strUnits(r)
	var out []uint16
	if len(tu) == 0 {
		for _, c := range su {
			out = append(out, ru...)
			out = append(out, c)
		}
		return strFromUnits(append(out, ru...))
	}
	last := 0
	for i := jxIndexOf(su, tu, 0); i >= 0; i = jxIndexOf(su, tu, last) {
		out = append(out, su[last:i]...)
		out = append(out, ru...)
		last = i + len(tu)
	}
	return strFromUnits(append(out, su[last:]...))
}

func jxCompare(a, b []uint16, fold bool) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		x, y := int(a[i]), int(b[i])
		if x != y && fold {
			x, y = jvChLo(jvChUp(x)), jvChLo(jvChUp(y))
			// Inside a surrogate pair the whole code points compare (JDK 16
			// and later); a supplementary one folds to itself. Twin of
			// jxCompareIC in languages/lib/java-strings.js.
			if cx, cy := jxCpAt(a, i), jxCpAt(b, i); x != y && (cx > 0xFFFF || cy > 0xFFFF) {
				if cx <= 0xFFFF {
					cx = jvChLo(jvChUp(cx))
				}
				if cy <= 0xFFFF {
					cy = jvChLo(jvChUp(cy))
				}
				if cx == cy {
					continue
				}
				return cx - cy
			}
		}
		if x != y {
			return x - y
		}
	}
	return len(a) - len(b)
}

// jxRegionMatches is java-strings.js's: the len units of s from to and of o
// from oo, folded when ic.
func jxRegionMatches(s []uint16, ic bool, to int, o []uint16, oo, n int) bool {
	if oo < 0 || to < 0 || to > len(s)-n || oo > len(o)-n {
		return false
	}
	for i := 0; i < n; i++ {
		a, b := int(s[to+i]), int(o[oo+i])
		if a != b && (!ic || jvChLo(jvChUp(a)) != jvChLo(jvChUp(b))) {
			return false
		}
	}
	return true
}

// jxCpAt is the code point the unit at i belongs to: a surrogate pair's whole
// value whichever half i is, the unit itself otherwise.
func jxCpAt(s []uint16, i int) int {
	c := int(s[i])
	if c >= 0xD800 && c < 0xDC00 && i+1 < len(s) && s[i+1] >= 0xDC00 && s[i+1] < 0xE000 {
		return 0x10000 + (c-0xD800)*1024 + int(s[i+1]) - 0xDC00
	}
	if c >= 0xDC00 && c < 0xE000 && i > 0 && s[i-1] >= 0xD800 && s[i-1] < 0xDC00 {
		return 0x10000 + (int(s[i-1])-0xD800)*1024 + c - 0xDC00
	}
	return c
}

// ----- the regex methods -----

func (rt *jsrt) jxRe(pattern string) *rxRe {
	re := rxGet(pattern, "q")
	if !re.ok {
		rt.jvThrowNamed("PatternSyntaxException", jxReMsg(pattern, re.err))
	}
	return re
}

// jxReMsg is Java's PatternSyntaxException message for a regex error kind, the
// index read off the shortest failing prefix of the \Q-expanded pattern. Twin of
// jxReMsg in languages/lib/java-strings.js.
func jxReMsg(pattern, err string) string {
	p := rxExpandQuotes([]rune(pattern))
	n := len(p)
	switch err {
	case "unterminated group":
		return jxReFmt(pattern, "Unclosed group", n)
	case "unterminated character class":
		return jxReFmt(pattern, "Unclosed character class", n-1)
	case "trailing backslash in pattern":
		return jxReFmt(pattern, "Unescaped trailing backslash", n)
	case "unterminated group name":
		return jxReFmt(pattern, "named capturing group is missing trailing '>'", n)
	}
	k := 1
	for k < n && rxCompile(string(p[:k]), "").err != err {
		k++
	}
	switch {
	case err == "unmatched ) in pattern":
		return jxReFmt(pattern, "Unmatched closing ')'", k-2)
	case err == "character class range is out of order":
		return jxReFmt(pattern, "Illegal character range", k-1)
	case err == "quantifier with nothing to repeat" || err == "multiple repeat":
		return jxReFmt(pattern, "Dangling meta character '"+string(p[k-1])+"'", k-1)
	case err == "quantifier bounds are out of order":
		return jxReFmt(pattern, "Illegal repetition range", k-1)
	case err == "invalid hex escape":
		return jxReFmt(pattern, "Illegal hexadecimal escape sequence", k)
	case strings.HasPrefix(err, "unsupported group form"):
		return jxReFmt(pattern, "Unknown inline modifier", k-1)
	case strings.HasPrefix(err, "unknown group name '"):
		name := err[len("unknown group name '") : len(err)-1]
		return jxReFmt(pattern, "named capturing group <"+name+"> does not exist", k-1)
	}
	return err
}

// jxReFmt is PatternSyntaxException.getMessage: the description, " near index N"
// when N >= 0, the pattern, and a caret under N when N is inside it.
func jxReFmt(pattern, desc string, at int) string {
	if at < 0 {
		return desc + "\n" + pattern
	}
	out := desc + " near index " + strconv.Itoa(at) + "\n" + pattern
	if at >= len([]rune(pattern)) {
		return out
	}
	return out + "\n" + strings.Repeat(" ", at) + "^"
}

func (rt *jsrt) jxSearch(re *rxRe, text []rune, at int) *rxMatch {
	m, ok := re.search(text, at)
	if !ok {
		rt.fail("regexp: step limit exceeded")
	}
	return m
}

func jxNext(m *rxMatch) int {
	if m.end == m.begin {
		return m.end + 1
	}
	return m.end
}

func (rt *jsrt) jxSplit(s, pattern string, limit int) []string {
	re := rt.jxRe(pattern)
	text := []rune(s)
	var out []string
	index := 0
	for at := 0; at <= len(text); {
		m := rt.jxSearch(re, text, at)
		if m == nil {
			break
		}
		if limit <= 0 || len(out) < limit-1 {
			if !(index == 0 && m.begin == 0 && m.end == 0) {
				out = append(out, string(text[index:m.begin]))
				index = m.end
			}
		} else {
			out = append(out, string(text[index:]))
			index = m.end
			break
		}
		at = jxNext(m)
	}
	if index == 0 {
		return []string{s}
	}
	if limit <= 0 || len(out) < limit {
		out = append(out, string(text[index:]))
	}
	if limit == 0 {
		for len(out) > 0 && out[len(out)-1] == "" {
			out = out[:len(out)-1]
		}
	}
	return out
}

func (rt *jsrt) jxMatches(s, pattern string) bool {
	re := rt.jxRe(pattern)
	m, ok := re.matchWhole([]rune(s), 0)
	if !ok {
		rt.fail("regexp: step limit exceeded")
	}
	return m != nil
}

func (rt *jsrt) jxRxReplace(s, pattern, repl string, all bool) string {
	re := rt.jxRe(pattern)
	text := []rune(s)
	var out strings.Builder
	tail := 0
	for at := 0; at <= len(text); {
		m := rt.jxSearch(re, text, at)
		if m == nil {
			break
		}
		out.WriteString(string(text[tail:m.begin]))
		out.WriteString(rt.jxExpand(re, text, m, []rune(repl)))
		tail = m.end
		if !all {
			break
		}
		at = jxNext(m)
	}
	out.WriteString(string(text[tail:]))
	return out.String()
}

func (rt *jsrt) jxExpand(re *rxRe, text []rune, m *rxMatch, repl []rune) string {
	var out strings.Builder
	for i := 0; i < len(repl); {
		c := repl[i]
		if c == '\\' {
			if i+1 >= len(repl) {
				rt.jvThrowNamed("IllegalArgumentException", "character to be escaped is missing")
			}
			out.WriteRune(repl[i+1])
			i += 2
			continue
		}
		if c != '$' {
			out.WriteRune(c)
			i++
			continue
		}
		i++
		if i >= len(repl) {
			rt.jvThrowNamed("IllegalArgumentException", "Illegal group reference: group index is missing")
		}
		g := -1
		if repl[i] == '{' {
			e := i + 1
			for e < len(repl) && repl[e] != '}' {
				e++
			}
			nm := string(repl[i+1 : e])
			if g = re.indexOfName(nm); g < 0 {
				rt.jvThrowNamed("IllegalArgumentException", "No group with name {"+nm+"}")
			}
			i = e + 1
		} else {
			d := int(repl[i]) - '0'
			if d < 0 || d > 9 {
				rt.jvThrowNamed("IllegalArgumentException", "Illegal group reference")
			}
			g = d
			i++
			if g > re.ngroups {
				rt.jvThrowNamed("IndexOutOfBoundsException", "No group "+strconv.Itoa(g))
			}
			for i < len(repl) {
				d2 := int(repl[i]) - '0'
				if d2 < 0 || d2 > 9 || g*10+d2 > re.ngroups {
					break
				}
				g = g*10 + d2
				i++
			}
		}
		if gs, ok := re.group(text, m, g); ok {
			out.WriteString(gs)
		}
	}
	return out.String()
}

// ----- String.format -----

// jxFormat is lib/java-strings.js's jxFormat: a malformed specifier, a
// missing argument and a mismatched one throw java.util.Formatter's
// exceptions (jvThrowNamed).
func (rt *jsrt) jxFormat(fmtS string, args []interface{}) string {
	f := []rune(fmtS)
	var out strings.Builder
	next, last := 0, -1
	digit := func(c rune) bool { return c >= '0' && c <= '9' }
	for i := 0; i < len(f); {
		if f[i] != '%' {
			out.WriteRune(f[i])
			i++
			continue
		}
		start := i
		j := i + 1
		k := j
		for k < len(f) && digit(f[k]) {
			k++
		}
		idx := -1
		if k > j && k < len(f) && f[k] == '$' {
			n, _ := strconv.Atoi(string(f[j:k]))
			idx = n - 1
			if idx < 0 {
				rt.jvThrowNamed("IllegalFormatArgumentIndexException", "Illegal format argument index = "+strconv.Itoa(n))
			}
			j = k + 1
		}
		flags := ""
		for j < len(f) && strings.ContainsRune("-#+ 0,(<", f[j]) {
			if strings.ContainsRune(flags, f[j]) {
				rt.jvThrowNamed("DuplicateFormatFlagsException", "Flags = '"+string(f[j])+"'")
			}
			flags += string(f[j])
			j++
		}
		width := -1
		for k = j; k < len(f) && digit(f[k]); k++ {
		}
		if k > j {
			width, _ = strconv.Atoi(string(f[j:k]))
		}
		j = k
		prec := -1
		bad := false
		if j < len(f) && f[j] == '.' {
			for k = j + 1; k < len(f) && digit(f[k]); k++ {
			}
			if k == j+1 {
				bad = true
			}
			prec, _ = strconv.Atoi(string(f[j+1 : k]))
			j = k
		}
		// A specifier that does not parse names the character after the '%'.
		if bad || j >= len(f) {
			c := "%"
			if start+1 < len(f) {
				c = string(f[start+1])
			}
			rt.jvThrowNamed("UnknownFormatConversionException", "Conversion = '"+c+"'")
		}
		conv := f[j]
		i = j + 1
		rt.jxCheckSpec(conv, flags, width, prec, idx)
		if conv == 'n' {
			out.WriteString("\n")
			continue
		}
		if conv == '%' {
			out.WriteString(jxPad("%", flags, width))
			continue
		}
		ai := next
		if strings.Contains(flags, "<") {
			ai = last
		} else if idx >= 0 {
			ai = idx
		} else {
			next++
		}
		if ai < 0 || ai >= len(args) {
			rt.jvThrowNamed("MissingFormatArgumentException", "Format specifier '"+string(f[start:i])+"'")
		}
		last = ai
		out.WriteString(rt.jxConv(conv, flags, width, prec, args[ai]))
	}
	return out.String()
}

// jxCheckSpec is java-strings.js's jxCheckSpec: the checks Formatter makes of
// a specifier before it reads the argument.
func (rt *jsrt) jxCheckSpec(conv rune, flags string, width, prec, idx int) {
	if !strings.ContainsRune("bBhHsScCdoxXeEfgGaAtT%n", conv) {
		rt.jvThrowNamed("UnknownFormatConversionException", "Conversion = '"+string(conv)+"'")
	}
	lc := []rune(strings.ToLower(string(conv)))[0]
	has := func(c string) bool { return strings.Contains(flags, c) }
	sp := jxSpecStr(conv, flags, width, prec, idx)
	missing := func() { rt.jvThrowNamed("MissingFormatWidthException", sp) }
	badPrec := func() { rt.jvThrowNamed("IllegalFormatPrecisionException", strconv.Itoa(prec)) }
	illegal := func() { rt.jvThrowNamed("IllegalFormatFlagsException", "Flags = '"+jxFlagsCanon(flags)+"'") }
	switch lc {
	case '%', 'n':
		if prec >= 0 {
			badPrec()
		}
		if lc == 'n' && width >= 0 {
			rt.jvThrowNamed("IllegalFormatWidthException", strconv.Itoa(width))
		}
		if flags != "" && (lc == 'n' || flags != "-") {
			illegal()
		}
		if width < 0 && has("-") {
			missing()
		}
		return
	case 'b', 'h', 's':
		if lc != 's' && has("#") {
			rt.jxMismatch(lc, '#')
		}
		if width < 0 && has("-") {
			missing()
		}
		rt.jxBadFlags(lc, flags, "+ 0,(")
		return
	case 'c':
		if prec >= 0 {
			badPrec()
		}
		rt.jxBadFlags(lc, flags, "#+ 0,(")
		if width < 0 && has("-") {
			missing()
		}
		return
	case 't':
		return
	}
	if width < 0 && (has("-") || has("0")) {
		missing()
	}
	if (has("+") && has(" ")) || (has("-") && has("0")) {
		illegal()
	}
	switch lc {
	case 'd', 'o', 'x':
		if prec >= 0 {
			badPrec()
		}
		if lc == 'd' {
			rt.jxBadFlags(lc, flags, "#")
		} else {
			rt.jxBadFlags(lc, flags, ",")
		}
	case 'a':
		rt.jxBadFlags(lc, flags, "(,")
	case 'e':
		rt.jxBadFlags(lc, flags, ",")
	case 'g':
		rt.jxBadFlags(lc, flags, "#")
	}
}

func (rt *jsrt) jxBadFlags(lc rune, flags, bad string) {
	for _, c := range bad {
		if strings.ContainsRune(flags, c) {
			rt.jxMismatch(lc, c)
		}
	}
}

func (rt *jsrt) jxMismatch(lc, flag rune) {
	rt.jvThrowNamed("FormatFlagsConversionMismatchException", "Conversion = "+string(lc)+", Flags = "+string(flag))
}

// jxFlagsCanon is the flags in Formatter's own order, as its messages spell them.
func jxFlagsCanon(flags string) string {
	out := ""
	for _, c := range "-#+ 0,(<" {
		if strings.ContainsRune(flags, c) {
			out += string(c)
		}
	}
	return out
}

// jxSpecStr is the specifier as Formatter's toString writes it back.
func jxSpecStr(conv rune, flags string, width, prec, idx int) string {
	out := "%" + jxFlagsCanon(flags)
	if idx >= 0 {
		out += strconv.Itoa(idx+1) + "$"
	}
	if width >= 0 {
		out += strconv.Itoa(width)
	}
	if prec >= 0 {
		out += "." + strconv.Itoa(prec)
	}
	return out + string(conv)
}

func jxPad(s, flags string, width int) string {
	n := len(strUnits(s))
	if n >= width {
		return s
	}
	pad := strings.Repeat(" ", width-n)
	if strings.Contains(flags, "-") {
		return s + pad
	}
	return pad + s
}

func (rt *jsrt) jxConv(conv rune, flags string, width, prec int, v interface{}) string {
	lc := []rune(strings.ToLower(string(conv)))[0]
	up := lc != conv
	k := jxKind(v)
	body := ""
	cut := func() {
		if u := strUnits(body); prec >= 0 && prec < len(u) {
			body = strFromUnits(u[:prec])
		}
	}
	switch lc {
	case 's', 'b':
		// '#' is for a Formattable, which no value here is.
		if lc == 's' && strings.Contains(flags, "#") {
			rt.jxMismatch(lc, '#')
		}
		switch {
		case k == "null" && lc == 's':
			body = "null"
		case k == "null":
			body = "false"
		case lc == 's' || k == "bool":
			body = rt.jvpStr(v)
		default:
			body = "true"
		}
		cut()
	case 'h':
		if k == "null" {
			body = "null"
		} else {
			body = strconv.FormatUint(uint64(uint32(int32(rt.jvHash(v)))), 16)
		}
		cut()
	case 'c':
		switch {
		case k == "null":
			body = "null"
		case k == "char" || (k == "int" && jvClassNameOf(v) != "java.lang.Long"):
			c := jxCode(rt, v)
			if k == "int" && (c < 0 || c > 0x10FFFF) {
				rt.jvThrowNamed("IllegalFormatCodePointException", "Code point = 0x"+strconv.FormatUint(uint64(uint32(int32(c))), 16))
			}
			if c < 0x10000 {
				body = strFromUnits([]uint16{uint16(c)})
			} else {
				body = string(rune(c))
			}
		default:
			rt.jxBadConv(lc, v)
		}
	case 'd':
		if k == "null" {
			body = "null"
			break
		}
		if k != "int" {
			rt.jxBadConv(lc, v)
		}
		dd := strconv.FormatInt(jvToLong(rt, v), 10)
		neg := strings.HasPrefix(dd, "-")
		dd = strings.TrimPrefix(dd, "-")
		if strings.Contains(flags, ",") {
			dd = jxGroup(dd)
		}
		return jxNum(neg, dd, flags, width, up)
	case 'x', 'o':
		if k == "null" {
			body = "null"
			break
		}
		if k != "int" {
			rt.jxBadConv(lc, v)
		}
		// A sign or a bracket is for a BigInteger only.
		rt.jxBadFlags(lc, flags, "( +")
		base := 8
		if lc == 'x' {
			base = 16
		}
		rd := jxRadix(rt, v, base)
		if strings.Contains(flags, "#") {
			if lc == 'x' {
				rd = "0x" + rd
			} else {
				rd = "0" + rd
			}
		}
		if strings.Contains(flags, "0") {
			for len(rd) < width {
				if lc == 'x' && strings.Contains(flags, "#") {
					rd = "0x0" + rd[2:]
				} else {
					rd = "0" + rd
				}
			}
		}
		body = rd
	case 'f', 'e', 'g', 'a':
		if k == "null" {
			body = "null"
			break
		}
		if k != "float" {
			rt.jxBadConv(lc, v)
		}
		if lc == 'a' {
			return jxHexFloat(v.(jsJFlo).f, flags, width, prec, up)
		}
		return jxFloat(lc, v.(jsJFlo).f, flags, width, prec, up)
	default:
		rt.fail("unsupported conversion %%%c", conv)
	}
	// A null argument prints "null", cut to the precision like a string.
	if k == "null" {
		cut()
	}
	if up {
		body = caseMapMode(body, caseUpper|caseFull)
	}
	return jxPad(body, flags, width)
}

func (rt *jsrt) jxBadConv(lc rune, v interface{}) {
	rt.jvThrowNamed("IllegalFormatConversionException", string(lc)+" != "+jvClassNameOf(v))
}

// jxHexFloat is %a: java-strings.js's jxHexFloat.
func jxHexFloat(x float64, flags string, width, prec int, up bool) string {
	if x != x || math.IsInf(x, 0) {
		return jxFloat('e', x, flags, width, prec, up)
	}
	neg := x < 0 || (x == 0 && math.Signbit(x))
	a := math.Abs(x)
	lead, e, m := 1, 0, a
	cut := prec >= 0 && prec < 13
	switch {
	case a == 0:
		lead = 0
	case a < 0x1p-1022:
		m = a * 0x1p1022
		e = -1022
		if cut {
			for m < 1 {
				m *= 2
				e--
			}
		} else {
			lead = 0
		}
	default:
		for m >= 2 {
			m /= 2
			e++
		}
		for m < 1 {
			m *= 2
			e--
		}
	}
	frac := m - float64(lead)
	var ds [13]int
	for i := range ds {
		frac *= 16
		d := math.Floor(frac)
		ds[i] = int(d)
		frac -= d
	}
	n := 13
	if cut {
		n = prec
		if n == 0 {
			n = 1
		}
		more := false
		for r := n + 1; r < 13; r++ {
			if ds[r] != 0 {
				more = true
			}
		}
		if first := ds[n]; first > 8 || (first == 8 && (more || ds[n-1]%2 == 1)) {
			q := n - 1
			for q >= 0 && ds[q] == 15 {
				ds[q] = 0
				q--
			}
			if q >= 0 {
				ds[q]++
			} else {
				lead++
			}
			if lead == 2 {
				lead = 1
				e++
			}
		}
	} else if prec < 0 {
		for n > 1 && ds[n-1] == 0 {
			n--
		}
	}
	hex := ""
	for _, d := range ds[:n] {
		hex += string("0123456789abcdef"[d])
	}
	for len(hex) < prec {
		hex += "0"
	}
	body := strconv.Itoa(lead) + "." + hex + "p" + strconv.Itoa(e)
	sign := ""
	switch {
	case neg:
		sign = "-"
	case strings.Contains(flags, "+"):
		sign = "+"
	case strings.Contains(flags, " "):
		sign = " "
	}
	if strings.Contains(flags, "0") {
		for len(sign)+2+len(body) < width {
			body = "0" + body
		}
	}
	out := sign + "0x" + body
	if up {
		out = strings.ToUpper(out)
	}
	return jxPad(out, flags, width)
}

func jxGroup(ds string) string {
	var out strings.Builder
	n := len(ds)
	for i := 0; i < n; i++ {
		if i > 0 && (n-i)%3 == 0 {
			out.WriteByte(',')
		}
		out.WriteByte(ds[i])
	}
	return out.String()
}

func jxNum(neg bool, digits, flags string, width int, up bool) string {
	pre, post := "", ""
	switch {
	case neg && strings.Contains(flags, "("):
		pre, post = "(", ")"
	case neg:
		pre = "-"
	case strings.Contains(flags, "+"):
		pre = "+"
	case strings.Contains(flags, " "):
		pre = " "
	}
	body := digits
	if up {
		body = caseMapMode(body, caseUpper|caseFull)
	}
	if strings.Contains(flags, "0") {
		for len(pre)+len(body)+len(post) < width {
			body = "0" + body
		}
	}
	return jxPad(pre+body+post, flags, width)
}

func jxRoundUp(ds string, keep int) string {
	if keep < 0 {
		return ""
	}
	if keep >= len(ds) {
		return ds
	}
	head := []byte(ds[:keep])
	if ds[keep] < '5' {
		return string(head)
	}
	for i := keep - 1; i >= 0; i-- {
		if head[i] == '9' {
			head[i] = '0'
			continue
		}
		head[i]++
		return string(head)
	}
	return "1" + string(head)
}

func jxFloat(lc rune, x float64, flags string, width, prec int, up bool) string {
	if x != x {
		if up {
			return jxPad("NAN", flags, width)
		}
		return jxPad("NaN", flags, width)
	}
	neg := x < 0 || (x == 0 && math.Signbit(x))
	a := math.Abs(x)
	if math.IsInf(a, 0) {
		inf := "Infinity"
		if up {
			inf = "INFINITY"
		}
		return jxNum(neg, inf, strings.ReplaceAll(flags, "0", ""), width, false)
	}
	ds, e := "0", 0
	if a != 0 {
		ds, e = jxDigitsOf(jvmFloStr(a))
	}
	group := strings.Contains(flags, ",")
	body := ""
	switch lc {
	case 'g':
		if prec < 0 {
			prec = 6
		}
		if prec == 0 {
			prec = 1
		}
		re := e
		if a != 0 && len(jxRoundUp(ds, prec)) > prec {
			re = e + 1
		}
		if a == 0 {
			body = jxFixed(ds, e, prec-1, group)
		} else if re >= -4 && re < prec {
			body = jxFixed(ds, e, prec-1-re, group)
		} else {
			body = jxSci(ds, e, prec-1, false)
		}
	case 'f':
		if prec < 0 {
			prec = 6
		}
		body = jxFixed(ds, e, prec, group)
		if prec == 0 && strings.Contains(flags, "#") {
			body += "."
		}
	default:
		if prec < 0 {
			prec = 6
		}
		body = jxSci(ds, e, prec, strings.Contains(flags, "#"))
	}
	return jxNum(neg, body, flags, width, up)
}

// jxDigitsOf reads Double.toString's text as its significant digits and the
// decimal exponent of the first one.
func jxDigitsOf(t string) (string, int) {
	mant, exp := t, 0
	if ei := strings.IndexByte(t, 'E'); ei >= 0 {
		mant = t[:ei]
		exp, _ = strconv.Atoi(t[ei+1:])
	}
	pt := strings.IndexByte(mant, '.')
	ds := strings.Replace(mant, ".", "", 1)
	if pt < 0 {
		pt = len(ds)
	}
	lead := 0
	for lead < len(ds)-1 && ds[lead] == '0' {
		lead++
	}
	ds = strings.TrimRight(ds[lead:], "0")
	if ds == "" {
		ds = "0"
	}
	return ds, pt - lead + exp - 1
}

func jxFixed(ds string, e, prec int, group bool) string {
	pt := e + 1
	r := ""
	if ds != "0" {
		r = jxRoundUp(ds, pt+prec)
	}
	if len(r) > pt+prec && pt+prec >= 0 {
		pt++
	}
	ints, frac := "0", ""
	if pt <= 0 {
		frac = strings.Repeat("0", -pt) + r
	} else {
		for len(r) < pt {
			r += "0"
		}
		ints, frac = r[:pt], r[pt:]
	}
	for len(frac) < prec {
		frac += "0"
	}
	frac = frac[:prec]
	if group {
		ints = jxGroup(ints)
	}
	if prec == 0 {
		return ints
	}
	return ints + "." + frac
}

func jxSci(ds string, e, prec int, point bool) string {
	ex, r := e, ds
	if ds != "0" {
		r = jxRoundUp(ds, prec+1)
		if len(r) > prec+1 {
			ex++
			r = r[:prec+1]
		}
	} else {
		ex = 0
	}
	for len(r) < prec+1 {
		r += "0"
	}
	m := r[:1]
	if prec > 0 || point {
		m += "." + r[1:]
	}
	sign := "+"
	if ex < 0 {
		sign, ex = "-", -ex
	}
	es := strconv.Itoa(ex)
	if len(es) < 2 {
		es = "0" + es
	}
	return m + "e" + sign + es
}

// ----- the value-model hooks -----

func jxKind(v interface{}) string {
	switch v.(type) {
	case jsNullT, jsUndefT, nil:
		return "null"
	case bool:
		return "bool"
	case jsChar:
		return "char"
	case jsGInt, float64:
		return "int"
	case jsJFlo:
		return "float"
	case string:
		return "str"
	}
	return "obj"
}

func jxCode(rt *jsrt, v interface{}) int {
	if c, ok := v.(jsChar); ok {
		return int(c.code)
	}
	return int(jvToLong(rt, v))
}

// jxRadix is an integral value's two's complement digits at its own width.
func jxRadix(rt *jsrt, v interface{}, base int) string {
	w := uint8(32)
	if g, ok := v.(jsGInt); ok {
		w = g.w
	}
	p := uint64(jvToLong(rt, v))
	if w < 64 {
		p &= uint64(1)<<w - 1
	}
	return strconv.FormatUint(p, base)
}

// jxArgStr is a String method's CharSequence or char argument as a string.
func (rt *jsrt) jxArgStr(v interface{}) string {
	rt.jvNpeIfNull(v)
	if c, ok := v.(jsChar); ok {
		return strFromUnits([]uint16{uint16(c.code)})
	}
	return rt.jvpStr(v)
}

// jxArgPat is indexOf's first argument: a string, or a char / int code point.
func (rt *jsrt) jxArgPat(v interface{}) []uint16 {
	switch t := v.(type) {
	case string:
		return strUnits(t)
	case jsChar:
		return []uint16{uint16(t.code)}
	}
	rt.jvNpeIfNull(v)
	return strUnits(string(rune(jxCode(rt, v))))
}

func jxStrArr(parts []string) *jsArray {
	out := &jsArray{elems: make([]interface{}, len(parts))}
	for i, p := range parts {
		out.elems[i] = p
	}
	return out
}

// jvIaeCls is IllegalArgumentException, raised by String.repeat; registered by
// js_jvexc like the other runtime-raised classes.
var jvIaeCls *jsObject

// jvBoxName is the java.lang wrapper class a boxed primitive (or a String)
// belongs to, or "" for any other value. Twin of jBoxName in
// java-interpreter.abnf and jvBoxName in languages/lib/java-rt.metajs.
func jvBoxName(v interface{}) string {
	switch t := v.(type) {
	case string:
		return "String"
	case bool:
		return "Boolean"
	case float64:
		return "Integer"
	case jsChar:
		return "Character"
	case jsJFlo:
		if jvmIs32(t.sty) {
			return "Float"
		}
		return "Double"
	case jsGInt:
		switch t.w {
		case 64:
			return "Long"
		case 16:
			return "Short"
		}
		return "Byte"
	}
	return ""
}

// jvIsType is `v instanceof T` (extern js_jvis): a boxed primitive is an
// instance of its own wrapper class and its supertypes only, never of another
// wrapper (JLS 15.20.2); every other value goes to isTypeName. A qualified name
// is tested by its simple name. Twin of jIsType in java-interpreter.abnf and
// js_jvis in languages/lib/java-rt.metajs.
func (rt *jsrt) jvIsType(v interface{}, t string) bool {
	switch v.(type) {
	case nil, jsUndefT, jsNullT:
		return false
	}
	if i := strings.IndexByte(t, '<'); i >= 0 {
		t = t[:i]
	}
	t = t[strings.LastIndexByte(t, '.')+1:]
	bn := jvBoxName(v)
	if bn == "" {
		return rt.isTypeName(v, t)
	}
	switch t {
	case bn, "Object", "Comparable", "Serializable":
		return true
	case "Number":
		return bn != "String" && bn != "Boolean" && bn != "Character"
	}
	return t == "CharSequence" && bn == "String"
}

// jvPrimZero is the zero of the primitive type named n, nil for a reference type.
func jvPrimZero(n string) interface{} {
	switch n {
	case "int":
		return float64(0)
	case "long":
		return jvBox(0)
	case "short":
		return jsGInt{w: 16}
	case "byte":
		return jsGInt{w: 8}
	case "char":
		return jsChar{}
	case "boolean":
		return false
	case "double":
		return jsJFlo{}
	case "float":
		return jsJFlo{sty: floJavaF}
	}
	return nil
}

// jvArrOfDesc is the binary name of an array of the class named comp.
func jvArrOfDesc(comp string) string {
	for c, p := range map[byte]string{'I': "int", 'J': "long", 'S': "short", 'B': "byte", 'F': "float", 'D': "double", 'C': "char", 'Z': "boolean"} {
		if p == comp {
			return "[" + string(c)
		}
	}
	if strings.HasPrefix(comp, "[") {
		return "[" + comp
	}
	return "[L" + comp + ";"
}

// jvClassNameOf is v.getClass().getName() of a non-null value. Twin of
// jxkClass in java-interpreter.abnf and languages/lib/java-rt.metajs.
func jvClassNameOf(v interface{}) string {
	if bn := jvBoxName(v); bn != "" {
		return "java.lang." + bn
	}
	switch t := v.(type) {
	case *jsArray:
		return jvpArrayTag(t)
	case *jsObject:
		if cls, ok := t.props["__class"].(*jsObject); ok {
			return jvpClassName(cls)
		}
	}
	return "java.lang.Object"
}

// jvCceMsg is java 24's ClassCastException message: the two classes and where
// each was loaded from (java.base for java.*, an array of primitives and an
// array of java.* classes; the unnamed module of the app loader otherwise).
// Twin of jCceMsg in java-interpreter.abnf and jvCceMsg in
// languages/lib/java-rt.metajs.
func jvCceMsg(src, dst string) string {
	where := func(n string) bool {
		e := strings.TrimLeft(n, "[")
		if len(e) < len(n) && !strings.HasPrefix(e, "L") {
			return true
		}
		if len(e) < len(n) {
			e = e[1:]
		}
		return strings.HasPrefix(e, "java.")
	}
	phrase := func(base bool) string {
		if base {
			return " is in module java.base of loader 'bootstrap'"
		}
		return " is in unnamed module of loader 'app'"
	}
	head := "class " + src + " cannot be cast to class " + dst + " ("
	bs, bd := where(src), where(dst)
	switch {
	case bs && bd:
		return head + src + " and " + dst + " are in module java.base of loader 'bootstrap')"
	case !bs && !bd:
		return head + src + " and " + dst + " are in unnamed module of loader 'app')"
	}
	return head + src + phrase(bs) + "; " + dst + phrase(bd) + ")"
}

func init() {
	rxExtraExterns = append(rxExtraExterns, func(rt *jsrt, m map[string]func(args []uint64) uint64) {
		u := rt.unwrap
		w := rt.wrap
		baseMcall := m["js_mcall"]
		// Object#getClass: one java.lang.Class object per class, so == compares
		// classes - an instance of the Class descriptor carrying the binary and the
		// simple name. Twin of jGetClass in java-interpreter.abnf and js_jvgetclass
		// in languages/lib/java-rt.metajs.
		classCls := newJSObject()
		classCls.set("__isclass", true)
		classCls.set("__name", "Class")
		classCls.set("__bname", "java.lang.Class")
		slot := func(name string, f func(self *jsObject) string) {
			classCls.set(name, jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
				self, _ := argAt(args, 0).(*jsObject)
				return f(self)
			}))
		}
		slot("getName", func(self *jsObject) string { return rt.toString(self.props["__cname"]) })
		slot("getSimpleName", func(self *jsObject) string { return rt.toString(self.props["__csimple"]) })
		slot("toString", func(self *jsObject) string {
			n := rt.toString(self.props["__cname"])
			if jvPrimZero(n) != nil || n == "void" {
				return n
			}
			if self.props["__ciface"] == true {
				return "interface " + n
			}
			return "class " + n
		})
		classObjs := map[string]*jsObject{}
		classOf := func(name, simple string) *jsObject {
			if c, ok := classObjs[name]; ok {
				return c
			}
			c := newJSObject()
			c.set("__class", classCls)
			c.set("__cname", name)
			c.set("__csimple", simple)
			classObjs[name] = c
			return c
		}
		lang := func(simple string) *jsObject { return classOf("java.lang."+simple, simple) }
		// Class#getComponentType and java.lang.reflect.Array.newInstance. Twin
		// of getComponentType / hostGlobals["Array"] in java-interpreter.abnf
		// and jvClassOf / js_jvreflarray in languages/lib/java-rt.metajs.
		classCls.set("getComponentType", jsHostFunc("getComponentType", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			self, _ := argAt(args, 0).(*jsObject)
			n := rt.toString(self.props["__cname"])
			if !strings.HasPrefix(n, "[") {
				return jsNull
			}
			simple := strings.TrimSuffix(jxArraySimple(n), "[]")
			switch n[1] {
			case 'L':
				return classOf(n[2:len(n)-1], simple)
			case '[':
				return classOf(n[1:], simple)
			}
			return classOf(simple, simple)
		}))
		reflArray := newJSObject()
		reflArray.set("getLength", jsHostFunc("getLength", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			v := argAt(args, 0)
			rt.jvNpeIfNull(v)
			arr, ok := v.(*jsArray)
			if !ok {
				rt.jvThrowCls(jvIaeCls, "Argument is not an array")
			}
			return float64(len(arr.elems))
		}))
		reflArray.set("newInstance", jsHostFunc("newInstance", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			cls, _ := argAt(args, 0).(*jsObject)
			n := int(jvToLong(rt, argAt(args, 1)))
			if n < 0 {
				rt.jvThrowCls(jvNegArrCls, strconv.Itoa(n))
			}
			comp := rt.toString(cls.props["__cname"])
			z := jvPrimZero(comp)
			if z == nil {
				z = jsNull
			}
			elems := make([]interface{}, n)
			for i := range elems {
				elems[i] = z
			}
			return &jsArray{elems: elems, kind: jvArrTagKind(jvArrOfDesc(comp))}
		}))
		m["js_jvreflarray"] = func(a []uint64) uint64 { return w(reflArray) }
		// Is v an array of references (an Object[], a String[], an int[][])?
		m["js_jvrefarr"] = func(a []uint64) uint64 {
			if arr, ok := u(a[0]).(*jsArray); ok {
				if t := jvpArrayTag(arr); t[1] == 'L' || t[1] == '[' {
					return jsHTrue
				}
			}
			return jsHFalse
		}
		// A class literal: (descriptor or undefined, name, simple name, dims,
		// is an interface).
		// Twin of js_jvclasslit in languages/lib/java-rt.metajs and
		// makeClassLit in java-interpreter.abnf.
		m["js_jvclasslit"] = func(a []uint64) uint64 {
			name, simple := rt.toString(u(a[1])), rt.toString(u(a[2]))
			if cls, ok := u(a[0]).(*jsObject); ok {
				name, simple = jvpClassName(cls), rt.toString(cls.props["__name"])
			}
			dims := int(jvToLong(rt, u(a[3])))
			if dims == 0 {
				c := classOf(name, simple)
				if u(a[4]) == true {
					c.set("__ciface", true)
				}
				return w(c)
			}
			tag := strings.Repeat("[", dims-1) + jvArrOfDesc(name)
			return w(classOf(tag, jxArraySimple(tag)))
		}
		m["js_jvgetclass"] = func(a []uint64) uint64 {
			if bn := jvBoxName(u(a[0])); bn != "" {
				return w(lang(bn))
			}
			switch t := u(a[0]).(type) {
			case *jsArray:
				tag := jvpArrayTag(t)
				return w(classOf(tag, jxArraySimple(tag)))
			case *jsObject:
				if cls, ok := t.props["__class"].(*jsObject); ok {
					return w(classOf(jvpClassName(cls), rt.toString(cls.props["__name"])))
				}
			case jsNullT, jsUndefT:
				rt.fail("method call .getClass() on null")
			}
			rt.fail("getClass() of a value with no class")
			return 0
		}
		// String's chars / codePoints / lines: (receiver, name, args,
		// IntStream, Stream) - the stream's of() over the receiver's array, or
		// an ordinary call when the receiver is no String. Twin of jStrMethod
		// in java-interpreter.abnf and js_jvstrstream in
		// languages/lib/java-rt.metajs.
		m["js_jvstrstream"] = func(a []uint64) uint64 {
			s, isStr := u(a[0]).(string)
			if !isStr {
				return baseMcall(a[:3])
			}
			su := strUnits(s)
			arr := &jsArray{}
			cls := u(a[3])
			switch rt.toString(u(a[1])) {
			case "lines":
				cls = u(a[4])
				arr.kind = jvArrTagKind("[Ljava.lang.String;")
				start := 0
				for i := 0; i < len(su); i++ {
					if su[i] == '\n' || su[i] == '\r' {
						arr.elems = append(arr.elems, strFromUnits(su[start:i]))
						if su[i] == '\r' && i+1 < len(su) && su[i+1] == '\n' {
							i++
						}
						start = i + 1
					}
				}
				if start < len(su) {
					arr.elems = append(arr.elems, strFromUnits(su[start:]))
				}
			case "chars":
				arr.kind = jvArrTagKind("[I")
				for _, c := range su {
					arr.elems = append(arr.elems, float64(c))
				}
			default:
				arr.kind = jvArrTagKind("[I")
				for i := 0; i < len(su); i++ {
					cp := jxCpAt(su, i)
					arr.elems = append(arr.elems, float64(cp))
					if cp > 0xFFFF {
						i++
					}
				}
			}
			return w(rt.call(rt.getMember(cls, "of"), jsUndef, []interface{}{arr}))
		}
		m["js_jvis"] = func(a []uint64) uint64 {
			return w(rt.jvIsType(u(a[0]), rt.toString(u(a[1]))))
		}
		// (T) v: (v, T's simple name, T's binary name). A ClassCastException
		// when v is not a T; Comparable and CharSequence are judged only for a
		// box, a String or an array. Twin of makeRefCast in
		// java-interpreter.abnf and js_jvcast in languages/lib/java-rt.metajs.
		m["js_jvcast"] = func(a []uint64) uint64 {
			v := u(a[0])
			t := rt.toString(u(a[1]))
			if isUndefOrNull(v) || isCallable(v) || rt.jvIsType(v, t) {
				return a[0]
			}
			if _, isArr := v.(*jsArray); (t == "Comparable" || t == "CharSequence") && jvBoxName(v) == "" && !isArr {
				return a[0]
			}
			rt.jvThrowCls(jvCceCls, jvCceMsg(jvClassNameOf(v), rt.toString(u(a[2]))))
			return 0
		}
		// A java.lang.String method on a string receiver, compareTo and the
		// unboxing members on a boxed primitive too; any other receiver is an ordinary call of a method by
		// that name. Twin of jStrMethod in java-interpreter.abnf and js_jvstrm in
		// languages/lib/java-rt.metajs.
		m["js_jvstrm"] = func(a []uint64) uint64 {
			s, isStr := u(a[0]).(string)
			if !isStr {
				args, _ := u(a[2]).(*jsArray)
				if args != nil && len(args.elems) == 1 && rt.toString(u(a[1])) == "compareTo" {
					if r, ok := rt.jvWrapCompare(u(a[0]), args.elems[0]); ok {
						return w(r)
					}
				}
				if args == nil || len(args.elems) == 0 {
					if r, ok := rt.jvUnbox(u(a[0]), rt.toString(u(a[1]))); ok {
						return w(r)
					}
				}
				return baseMcall(a)
			}
			name := rt.toString(u(a[1]))
			args, _ := u(a[2]).(*jsArray)
			var av []interface{}
			if args != nil {
				av = args.elems
			}
			arg := func(i int) interface{} { return argAt(av, i) }
			num := func(i int) int { return int(jvToLong(rt, arg(i))) }
			su := strUnits(s)
			var r interface{}
			switch name {
			case "isEmpty":
				r = len(su) == 0
			case "formatted":
				return m["js_jvstrstatic"]([]uint64{w("format"), w(&jsArray{elems: append([]interface{}{s}, av...)})})
			case "regionMatches":
				if ic, ok := arg(0).(bool); ok {
					r = jxRegionMatches(su, ic, num(1), strUnits(rt.jxArgStr(arg(2))), num(3), num(4))
				} else {
					r = jxRegionMatches(su, false, num(0), strUnits(rt.jxArgStr(arg(1))), num(2), num(3))
				}
			case "codePointCount":
				b, e := num(0), num(1)
				if b < 0 || e > len(su) || b > e {
					rt.jvThrowNamed("IndexOutOfBoundsException", "Range ["+strconv.Itoa(b)+", "+strconv.Itoa(e)+") out of bounds for length "+strconv.Itoa(len(su)))
				}
				n := 0
				for i := b; i < e; i++ {
					if su[i] >= 0xD800 && su[i] < 0xDC00 && i+1 < e && su[i+1] >= 0xDC00 && su[i+1] < 0xE000 {
						i++
					}
					n++
				}
				r = float64(n)
			case "isBlank":
				r = jxStrip(s, true, false) == ""
			case "indexOf":
				from := 0
				if len(av) > 1 {
					from = num(1)
				}
				r = float64(jxIndexOf(su, rt.jxArgPat(arg(0)), from))
			case "lastIndexOf":
				from := len(su)
				if len(av) > 1 {
					from = num(1)
				}
				r = float64(jxLastIndexOf(su, rt.jxArgPat(arg(0)), from))
			case "contains":
				r = jxIndexOf(su, strUnits(rt.jxArgStr(arg(0))), 0) >= 0
			case "startsWith":
				off := 0
				if len(av) > 1 {
					off = num(1)
				}
				r = jxAt(su, strUnits(rt.jxArgStr(arg(0))), off)
			case "endsWith":
				t := strUnits(rt.jxArgStr(arg(0)))
				r = jxAt(su, t, len(su)-len(t))
			case "equalsIgnoreCase":
				o, ok := arg(0).(string)
				r = ok && len(strUnits(o)) == len(su) && jxCompare(su, strUnits(o), true) == 0
			case "compareTo", "compareToIgnoreCase":
				r = float64(jxCompare(su, strUnits(rt.jxArgStr(arg(0))), name == "compareToIgnoreCase"))
			case "concat":
				r = s + rt.jxArgStr(arg(0))
			case "replace":
				r = jxReplace(s, rt.jxArgStr(arg(0)), rt.jxArgStr(arg(1)))
			case "replaceAll", "replaceFirst":
				r = rt.jxRxReplace(s, rt.jxArgStr(arg(0)), rt.jxArgStr(arg(1)), name == "replaceAll")
			case "matches":
				r = rt.jxMatches(s, rt.jxArgStr(arg(0)))
			case "split":
				limit := 0
				if len(av) > 1 {
					limit = num(1)
				}
				r = jxStrArr(rt.jxSplit(s, rt.jxArgStr(arg(0)), limit))
			case "trim":
				r = jxTrim(s)
			case "strip":
				r = jxStrip(s, true, true)
			case "stripLeading":
				r = jxStrip(s, true, false)
			case "stripTrailing":
				r = jxStrip(s, false, true)
			case "toUpperCase":
				r = caseMapMode(s, caseUpper|caseFull)
			case "toLowerCase":
				r = caseMapMode(s, caseFull|caseSigma)
			case "repeat":
				n := num(0)
				if n < 0 {
					rt.jvThrowCls(jvIaeCls, "count is negative: "+strconv.Itoa(n))
				}
				r = strings.Repeat(s, n)
			case "codePointAt":
				i := num(0)
				if i < 0 || i >= len(su) {
					rt.jvThrowSioobe(i, len(su))
				}
				c := int(su[i])
				if c >= 0xD800 && c < 0xDC00 && i+1 < len(su) && su[i+1] >= 0xDC00 && su[i+1] < 0xE000 {
					c = 0x10000 + (c-0xD800)*1024 + int(su[i+1]) - 0xDC00
				}
				r = float64(c)
			case "subSequence":
				return m["js_jvsub"]([]uint64{a[0], a[2]})
			case "intern":
				r = s
			default:
				return baseMcall(a)
			}
			return w(r)
		}
		// String.valueOf / copyValueOf / join / format. Twin of jStrStatic in
		// java-interpreter.abnf and js_jvstrstatic in languages/lib/java-rt.metajs.
		m["js_jvstrstatic"] = func(a []uint64) uint64 {
			name := rt.toString(u(a[0]))
			args, _ := u(a[1]).(*jsArray)
			var av []interface{}
			if args != nil {
				av = args.elems
			}
			if len(av) == 0 {
				rt.fail("unknown method 'String.%s'", name)
			}
			switch name {
			case "valueOf", "copyValueOf":
				if _, isArr := av[0].(*jsArray); isArr {
					return m["js_jvnewstr"]([]uint64{a[1]})
				}
				return w(rt.jvpStr(av[0]))
			case "join":
				parts := []string{}
				for _, e := range rt.jxElems(av[1:]) {
					parts = append(parts, rt.jvpStr(e))
				}
				return w(strings.Join(parts, rt.jxArgStr(av[0])))
			case "format":
				rest := av[1:]
				if len(rest) == 1 {
					// An array of references (an Object[], also a String[][]) is
					// the varargs array itself.
					if arr, ok := rest[0].(*jsArray); ok && strings.ContainsRune("L[", rune(jvpArrayTag(arr)[1])) {
						rest = arr.elems
					}
				}
				return w(rt.jxFormat(rt.jxArgStr(av[0]), rest))
			}
			rt.fail("unknown method 'String.%s'", name)
			return 0
		}
	})
}

// jxArraySimple is an array class's simple name from its binary one: [I is
// int[], [[LMain$1P; is P[][] (a nested or local class goes by its own name).
// Twin of jArraySimple in java-interpreter.abnf and jvArraySimple in
// languages/lib/java-rt.metajs.
func jxArraySimple(tag string) string {
	d := 0
	for tag[d] == '[' {
		d++
	}
	var name string
	if tag[d] == 'L' {
		from := d + 1
		for i := from; i < len(tag); i++ {
			if tag[i] == '.' || tag[i] == '$' {
				from = i + 1
			}
		}
		for tag[from] >= '0' && tag[from] <= '9' {
			from++
		}
		name = tag[from : len(tag)-1]
	} else {
		names := map[byte]string{'I': "int", 'J': "long", 'D': "double", 'F': "float", 'Z': "boolean",
			'C': "char", 'S': "short", 'B': "byte"}
		name = names[tag[d]]
	}
	return name + strings.Repeat("[]", d)
}

// jxElems is String.join's elements: the varargs themselves, the one array
// they were passed as, or what one Iterable's iterator() yields.
func (rt *jsrt) jxElems(rest []interface{}) []interface{} {
	if len(rest) != 1 {
		return rest
	}
	switch t := rest[0].(type) {
	case *jsArray:
		return t.elems
	case *jsObject:
		var out []interface{}
		it := rt.memberCall(t, "iterator", nil)
		for rt.truthy(rt.memberCall(it, "hasNext", nil)) {
			out = append(out, rt.memberCall(it, "next", nil))
		}
		return out
	}
	return rest
}
