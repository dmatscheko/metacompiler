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
		}
		if x != y {
			return x - y
		}
	}
	return len(a) - len(b)
}

// ----- the regex methods -----

func (rt *jsrt) jxRe(pattern string) *rxRe {
	re := rxGet(pattern, "q")
	if !re.ok {
		rt.fail("PatternSyntaxException: %s", re.err)
	}
	return re
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
				rt.fail("IllegalArgumentException: character to be escaped is missing")
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
			rt.fail("IllegalArgumentException: Illegal group reference: group index is missing")
		}
		g := -1
		if repl[i] == '{' {
			e := i + 1
			for e < len(repl) && repl[e] != '}' {
				e++
			}
			nm := string(repl[i+1 : e])
			if g = re.indexOfName(nm); g < 0 {
				rt.fail("IllegalArgumentException: No group with name {%s}", nm)
			}
			i = e + 1
		} else {
			d := int(repl[i]) - '0'
			if d < 0 || d > 9 {
				rt.fail("IllegalArgumentException: Illegal group reference")
			}
			g = d
			i++
			if g > re.ngroups {
				rt.fail("IndexOutOfBoundsException: No group %d", g)
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
			j = k + 1
		}
		flags := ""
		for j < len(f) && strings.ContainsRune("-#+ 0,(<", f[j]) {
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
		if j < len(f) && f[j] == '.' {
			for k = j + 1; k < len(f) && digit(f[k]); k++ {
			}
			if k == j+1 {
				rt.fail("IllegalFormatPrecisionException: %s", string(f[start:k]))
			}
			prec, _ = strconv.Atoi(string(f[j+1 : k]))
			j = k
		}
		if j >= len(f) {
			rt.fail("UnknownFormatConversionException: Conversion = '%%'")
		}
		conv := f[j]
		i = j + 1
		spec := string(f[start:i])
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
			rt.fail("MissingFormatArgumentException: Format specifier '%s'", spec)
		}
		last = ai
		out.WriteString(rt.jxConv(conv, flags, width, prec, args[ai]))
	}
	return out.String()
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
	switch lc {
	case 's', 'b':
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
		if u := strUnits(body); prec >= 0 && prec < len(u) {
			body = strFromUnits(u[:prec])
		}
	case 'c':
		switch k {
		case "null":
			body = "null"
		case "char", "int":
			if c := jxCode(rt, v); c < 0x10000 {
				body = strFromUnits([]uint16{uint16(c)})
			} else {
				body = string(rune(c))
			}
		default:
			rt.jxBadConv(conv, k)
		}
	case 'd':
		if k == "null" {
			body = "null"
			break
		}
		if k != "int" {
			rt.jxBadConv(conv, k)
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
			rt.jxBadConv(conv, k)
		}
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
	case 'f', 'e', 'g':
		if k == "null" {
			body = "null"
			break
		}
		if k != "float" {
			rt.jxBadConv(conv, k)
		}
		return jxFloat(lc, v.(jsJFlo).f, flags, width, prec, up)
	default:
		rt.fail("UnknownFormatConversionException: Conversion = '%c'", conv)
	}
	if up {
		body = caseMapMode(body, caseUpper|caseFull)
	}
	return jxPad(body, flags, width)
}

func (rt *jsrt) jxBadConv(conv rune, k string) {
	rt.fail("IllegalFormatConversionException: %c != %s", conv, k)
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
// wrapper (JLS 15.20.2); every other value goes to isTypeName. Twin of jIsType
// in java-interpreter.abnf and js_jvis in languages/lib/java-rt.metajs.
func (rt *jsrt) jvIsType(v interface{}, t string) bool {
	switch v.(type) {
	case nil, jsUndefT, jsNullT:
		return false
	}
	if i := strings.IndexByte(t, '<'); i >= 0 {
		t = t[:i]
	}
	t = strings.TrimPrefix(t, "java.lang.")
	bn := jvBoxName(v)
	if bn == "" {
		return rt.isTypeName(v, t)
	}
	switch t {
	case bn, "Object", "Comparable", "Serializable", "java.io.Serializable":
		return true
	case "Number":
		return bn != "String" && bn != "Boolean" && bn != "Character"
	}
	return t == "CharSequence" && bn == "String"
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
		slot("toString", func(self *jsObject) string { return "class " + rt.toString(self.props["__cname"]) })
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
		m["js_jvis"] = func(a []uint64) uint64 {
			return w(rt.jvIsType(u(a[0]), rt.toString(u(a[1]))))
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
					if arr, ok := rest[0].(*jsArray); ok && strings.HasPrefix(jvpArrayTag(arr), "[L") {
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
// int[], [Ljava.lang.String; is String[].
func jxArraySimple(tag string) string {
	if tag[1] == 'L' {
		return tag[strings.LastIndexByte(tag, '.')+1:len(tag)-1] + "[]"
	}
	names := map[byte]string{'I': "int", 'J': "long", 'D': "double", 'F': "float", 'Z': "boolean",
		'C': "char", 'S': "short", 'B': "byte"}
	return names[tag[1]] + "[]"
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
