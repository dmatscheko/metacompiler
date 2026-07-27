package abnf

// The PYTHON half of the shared regular-expression subsystem: the `re` module as
// the python-to-llvm-ir grammar's emitted IR sees it.
//
// The engine itself is language-neutral and lives in jsrtregex.go (the Go twin of
// languages/lib/regex.js). What a compiler grammar still needs on top is a
// receiver-and-name dispatch table, because its output is an LLVM IR module whose
// only way out is a js_* extern - it cannot run the interpreter grammar's tag
// script. Ruby's table is rxRegexMethod; this file is Python's.
//
// Nothing here is a fork of the engine. Every match, group, replacement and split
// goes through the same rxRe as Ruby's, so the matrix comparing
// python-interpreter.abnf against python-to-llvm-ir.abnf on the same input is
// comparing two front ends over one matcher. The bodies below are line-by-line
// twins of the pyRe* helpers in languages/python-interpreter.abnf - keep the two
// in step.
//
// Registration is additive: an init() appends to rxExtraExterns, which
// addRegexExterns runs. Only NEW names are added, and js_pyrxmcall delegates to the
// js_pymcall it wraps for every receiver that is not the re module, a compiled
// pattern or a match - so a Python program that uses no regexps emits and runs
// exactly what it did before.

import (
	"strconv"
	"strings"
)

// ----- CPython's re constants (sre_constants) -----
const (
	pyReI = 2
	pyReL = 4
	pyReM = 8
	pyReS = 16
	pyReU = 32
	pyReX = 64
	pyReA = 256
)

// pyReFlagStr maps the bitmask onto the engine's four neutral letters. re.A is the
// engine's only behaviour and re.U is the py3 default, so both are accepted and
// ignored; re.L has no locale to read.
func pyReFlagStr(n int) string {
	s := ""
	if n&pyReI != 0 {
		s += "i"
	}
	if n&pyReS != 0 {
		s += "s"
	}
	if n&pyReM != 0 {
		s += "m"
	}
	if n&pyReX != 0 {
		s += "x"
	}
	return s
}

// pyRePat translates Python's P-spellings into the neutral engine's:
// (?P<name>...) is (?<name>...) and (?P=name) is the named backreference \k<name>.
// Escapes and character classes are stepped over, so `[(?P<]` is left alone. An
// unknown name is the ENGINE's error to report, which is why nothing is counted here.
func pyRePat(src string) (string, string) {
	var out strings.Builder
	inCls := false
	for i := 0; i < len(src); {
		ch := src[i]
		if ch == '\\' {
			end := i + 2
			if end > len(src) {
				end = len(src)
			}
			out.WriteString(src[i:end])
			i = end
			continue
		}
		if inCls {
			if ch == ']' {
				inCls = false
			}
			out.WriteByte(ch)
			i++
			continue
		}
		if ch == '[' {
			inCls = true
			out.WriteByte(ch)
			i++
			continue
		}
		if ch != '(' {
			out.WriteByte(ch)
			i++
			continue
		}
		if strings.HasPrefix(src[i:], "(?P<") {
			j := i + 4
			for j < len(src) && src[j] != '>' {
				j++
			}
			if j >= len(src) {
				return "", "unterminated group name"
			}
			out.WriteString("(?<" + src[i+4:j] + ">")
			i = j + 1
			continue
		}
		if strings.HasPrefix(src[i:], "(?P=") {
			k := i + 4
			for k < len(src) && src[k] != ')' {
				k++
			}
			if k >= len(src) {
				return "", "unterminated group name"
			}
			out.WriteString("\\k<" + src[i+4:k] + ">")
			i = k + 1
			continue
		}
		out.WriteByte(ch)
		i++
	}
	return out.String(), ""
}

// ----- The three value shapes -----
//
// All three are plain *jsObject, so js_pygetattr already answers re.I, p.pattern,
// p.flags, m.re and m.string without a wrapper of its own. A match additionally
// carries the __rx* fields rxMatchOf reads, which is what lets it share Ruby's
// group machinery unchanged.

func rxPyIsMod(v interface{}) bool {
	o, ok := v.(*jsObject)
	if !ok {
		return false
	}
	s, _ := o.props["__pymod"].(string)
	return s == "re"
}

func rxPyIsRe(v interface{}) bool {
	o, ok := v.(*jsObject)
	if !ok {
		return false
	}
	b, _ := o.props["__pyre"].(bool)
	return b
}

func rxPyIsMatch(v interface{}) bool {
	o, ok := v.(*jsObject)
	if !ok {
		return false
	}
	b, _ := o.props["__pymatch"].(bool)
	return b
}

func rxPyModule() *jsObject {
	o := newJSObject()
	o.set("__pymod", "re")
	o.set("NOFLAG", float64(0))
	pairs := []struct {
		n string
		v int
	}{
		{"A", pyReA}, {"ASCII", pyReA}, {"I", pyReI}, {"IGNORECASE", pyReI},
		{"L", pyReL}, {"LOCALE", pyReL}, {"M", pyReM}, {"MULTILINE", pyReM},
		{"S", pyReS}, {"DOTALL", pyReS}, {"U", pyReU}, {"UNICODE", pyReU},
		{"X", pyReX}, {"VERBOSE", pyReX},
	}
	for _, p := range pairs {
		o.set(p.n, float64(p.v))
	}
	return o
}

func (rt *jsrt) rxPyCompile(pat string, flags int) *jsObject {
	src, errMsg := pyRePat(pat)
	if errMsg != "" {
		rt.fail("re.error: %s", errMsg)
	}
	neutral := pyReFlagStr(flags)
	re := rxGet(src, neutral)
	if !re.ok {
		rt.fail("re.error: %s", re.err)
	}
	o := newJSObject()
	o.set("__pyre", true)
	o.set("pattern", pat)
	// Python's own .flags always reports re.UNICODE for a str pattern.
	o.set("flags", float64(flags|pyReU))
	o.set("__src", src)
	o.set("__nf", neutral)
	return o
}

func rxPyReOf(v interface{}) *rxRe {
	o := v.(*jsObject)
	src, _ := o.props["__src"].(string)
	nf, _ := o.props["__nf"].(string)
	return rxGet(src, nf)
}

func rxPyMkMatch(rv *jsObject, re *rxRe, anchoredSrc, s string, m *rxMatch) *jsObject {
	o := rxNewMatchObj(anchoredSrc, re.flags, s, m)
	o.set("__pymatch", true)
	o.set("re", rv)
	o.set("string", s)
	return o
}

func (rt *jsrt) rxPySearch(rv *jsObject, s string, from int) interface{} {
	re := rxPyReOf(rv)
	src, _ := rv.props["__src"].(string)
	m, ok := re.search([]rune(s), from)
	if !ok {
		rt.fail("regexp: step limit exceeded")
	}
	if m == nil {
		return jsUndef
	}
	return rxPyMkMatch(rv, re, src, s, m)
}

// rxPyAnchoredMatch is re.match / re.fullmatch. These are NOT a search plus a
// position check: a fullmatch has to BACKTRACK into an alternation, so
// re.fullmatch("a|ab", "ab") is "ab" and not None. The engine's own anchored entry
// points express both, including Python's `pos` argument - and unlike the
// \A(?:...)\z wrapper this used to build, they have no quarrel with the x flag's
// trailing comments.
func (rt *jsrt) rxPyAnchoredMatch(rv *jsObject, s string, from int, whole bool) interface{} {
	re := rxPyReOf(rv)
	src, _ := rv.props["__src"].(string)
	var m *rxMatch
	var ok bool
	if whole {
		m, ok = re.matchWhole([]rune(s), from)
	} else {
		m, ok = re.matchAt([]rune(s), from)
	}
	if !ok {
		rt.fail("regexp: step limit exceeded")
	}
	if m == nil {
		return jsUndef
	}
	return rxPyMkMatch(rv, re, src, s, m)
}

// ----- Match accessors -----

func (rt *jsrt) rxPyGroupIdx(md interface{}, key interface{}) int {
	re, _, _ := rxMatchOf(md)
	if f, isNum := key.(float64); isNum {
		i := int(f)
		if i < 0 || i > re.ngroups {
			rt.fail("IndexError: no such group")
		}
		return i
	}
	name := rt.toString(key)
	gi := re.indexOfName(name)
	if gi < 0 {
		rt.fail("IndexError: no such group")
	}
	return gi
}

func (rt *jsrt) rxPyGroup(md interface{}, key interface{}) interface{} {
	re, text, m := rxMatchOf(md)
	if g, ok := re.group(text, m, rt.rxPyGroupIdx(md, key)); ok {
		return g
	}
	return jsUndef
}

func (rt *jsrt) rxPySpan(md interface{}, key interface{}, which int) float64 {
	_, _, m := rxMatchOf(md)
	slot := 2*rt.rxPyGroupIdx(md, key) + which
	if slot >= len(m.caps) {
		return -1
	}
	return float64(m.caps[slot])
}

// Python's None is `undefined` in this value model (see the None production in
// python-to-llvm-ir.abnf), and it is NOT the same value as null - so every "no
// match" and every non-participating group answers undefined. That is why this
// cannot simply be rxCaptureArray, which fills Ruby's nil.
func rxPyGroups(md interface{}) *jsArray {
	re, text, m := rxMatchOf(md)
	out := &jsArray{}
	rxPyAppendGroups(out, re, text, m)
	return out
}

func rxPyGroupDict(md interface{}) *jsObject {
	re, text, m := rxMatchOf(md)
	keys, vals := &jsArray{}, &jsArray{}
	for i, n := range re.names {
		g, ok := re.group(text, m, re.nameGroups[i])
		if ok {
			dictAppend(keys, vals, n, g)
		} else {
			dictAppend(keys, vals, n, jsUndef)
		}
	}
	d := newJSObject()
	d.set("__dict", true)
	d.set("keys", keys)
	d.set("vals", vals)
	return d
}

// rxPyTemplate rewrites Python's \g<1> / \g<name> onto the \1 / \k<name> the
// engine's expander understands. The plain \1 form is already the engine's own.
func rxPyTemplate(repl string) string {
	var out strings.Builder
	for i := 0; i < len(repl); {
		if strings.HasPrefix(repl[i:], "\\g<") {
			j := i + 3
			for j < len(repl) && repl[j] != '>' {
				j++
			}
			nm := repl[i+3 : j]
			allDigit := len(nm) > 0
			for q := 0; q < len(nm); q++ {
				if nm[q] < '0' || nm[q] > '9' {
					allDigit = false
				}
			}
			if allDigit {
				out.WriteString("\\" + nm)
			} else {
				out.WriteString("\\k<" + nm + ">")
			}
			i = j + 1
			continue
		}
		if repl[i] == '\\' {
			end := i + 2
			if end > len(repl) {
				end = len(repl)
			}
			out.WriteString(repl[i:end])
			i = end
			continue
		}
		out.WriteByte(repl[i])
		i++
	}
	return out.String()
}

// ----- The pattern-taking operations -----

// rxPySub is re.sub / re.subn. `repl` may be a CALLABLE, which is why this cannot
// simply be rt.rxReplace: the closure has to be run per match.
func (rt *jsrt) rxPySub(rv *jsObject, repl interface{}, s string, count int) (string, int) {
	re := rxPyReOf(rv)
	src, _ := rv.props["__src"].(string)
	text := []rune(s)
	var out strings.Builder
	_, replIsStr := repl.(string)
	isFn := !isUndefOrNull(repl) && !replIsStr
	tpl := ""
	if !isFn {
		tpl = rxPyTemplate(rt.toString(repl))
	}
	at, last, n := 0, 0, 0
	for at <= len(text) {
		if count > 0 && n >= count {
			break
		}
		m, ok := re.search(text, at)
		if !ok {
			rt.fail("regexp: step limit exceeded")
		}
		if m == nil {
			break
		}
		out.WriteString(string(text[last:m.begin]))
		if isFn {
			md := rxPyMkMatch(rv, re, src, s, m)
			out.WriteString(rt.toString(rt.call(repl, jsUndef, []interface{}{md})))
		} else {
			out.WriteString(re.expand(text, m, tpl))
		}
		n++
		last, at = m.end, m.end
		if m.end == m.begin {
			if m.end < len(text) {
				out.WriteString(string(text[m.end]))
			}
			last, at = m.end+1, m.end+1
		}
	}
	if last <= len(text) {
		out.WriteString(string(text[last:]))
	}
	return out.String(), n
}

// rxPySplit is re.split. Unlike the engine's neutral rxSplit, Python SPLICES the
// capture groups of each separator into the result, so this walks the matches
// itself.
func (rt *jsrt) rxPySplit(rv *jsObject, s string, maxsplit int) *jsArray {
	re := rxPyReOf(rv)
	text := []rune(s)
	parts := &jsArray{}
	at, last, n := 0, 0, 0
	for at <= len(text) {
		if maxsplit > 0 && n >= maxsplit {
			break
		}
		m, ok := re.search(text, at)
		if !ok {
			rt.fail("regexp: step limit exceeded")
		}
		if m == nil {
			break
		}
		if m.end == m.begin {
			if m.begin >= len(text) {
				break
			}
			parts.elems = append(parts.elems, string(text[last:m.begin]))
			rxPyAppendGroups(parts, re, text, m)
			last = m.begin
			at = m.begin + 1
			n++
			continue
		}
		parts.elems = append(parts.elems, string(text[last:m.begin]))
		rxPyAppendGroups(parts, re, text, m)
		last, at = m.end, m.end
		n++
	}
	parts.elems = append(parts.elems, string(text[last:]))
	return parts
}

func rxPyAppendGroups(parts *jsArray, re *rxRe, text []rune, m *rxMatch) {
	for g := 1; g <= re.ngroups; g++ {
		if v, ok := re.group(text, m, g); ok {
			parts.elems = append(parts.elems, v)
		} else {
			parts.elems = append(parts.elems, jsUndef)
		}
	}
}

// rxPyFindAll is re.findall / re.finditer: no group -> the whole matches, one
// group -> that group, more -> the capture tuple of each match (a tuple is a list
// in this value model).
func (rt *jsrt) rxPyFindAll(rv *jsObject, s string, asMatch bool) *jsArray {
	re := rxPyReOf(rv)
	src, _ := rv.props["__src"].(string)
	text := []rune(s)
	out := &jsArray{}
	for at := 0; at <= len(text); {
		m, ok := re.search(text, at)
		if !ok {
			rt.fail("regexp: step limit exceeded")
		}
		if m == nil {
			break
		}
		switch {
		case asMatch:
			out.elems = append(out.elems, rxPyMkMatch(rv, re, src, s, m))
		case re.ngroups == 0:
			g, _ := re.group(text, m, 0)
			out.elems = append(out.elems, g)
		case re.ngroups == 1:
			if g, gok := re.group(text, m, 1); gok {
				out.elems = append(out.elems, g)
			} else {
				out.elems = append(out.elems, jsUndef)
			}
		default:
			t := &jsArray{}
			rxPyAppendGroups(t, re, text, m)
			out.elems = append(out.elems, t)
		}
		if m.end == m.begin {
			at = m.end + 1
		} else {
			at = m.end
		}
	}
	return out
}

// rxPyEscape is re.escape, CPython 3.7+: exactly the characters in
// _special_chars_map.
func rxPyEscape(s string) string {
	const special = "()[]{}?*+-|^$\\.&~# \t\n\r\v\f"
	var out strings.Builder
	for _, r := range s {
		if r < 128 && strings.ContainsRune(special, r) {
			out.WriteByte('\\')
		}
		out.WriteRune(r)
	}
	return out.String()
}

// ----- Dispatch -----

func (rt *jsrt) rxPyArgRe(v interface{}, flags int) *jsObject {
	if rxPyIsRe(v) {
		return v.(*jsObject)
	}
	return rt.rxPyCompile(rt.toString(v), flags)
}

func rxPyIntArg(args []interface{}, i int) int {
	if i < len(args) {
		if f, ok := args[i].(float64); ok {
			return int(f)
		}
	}
	return 0
}

// rxPyModCall is re.f(pattern, ...). Every one of those is the compiled pattern's
// own method with the pattern dropped, so the two entry points share one body; the
// only difference is where the optional flags argument sits.
func (rt *jsrt) rxPyModCall(name string, args []interface{}) interface{} {
	switch name {
	case "escape":
		return rxPyEscape(rt.toString(argAt(args, 0)))
	case "purge":
		return jsUndef
	case "compile":
		return rt.rxPyArgRe(argAt(args, 0), rxPyIntArg(args, 1))
	}
	flpos := -1
	switch name {
	case "search", "match", "fullmatch", "findall", "finditer":
		flpos = 2
	case "sub", "subn":
		flpos = 4
	case "split":
		flpos = 3
	default:
		rt.fail("unknown re function: %s", name)
	}
	rv := rt.rxPyArgRe(argAt(args, 0), rxPyIntArg(args, flpos))
	var rest []interface{}
	for i := 1; i < len(args) && i < flpos; i++ {
		rest = append(rest, args[i])
	}
	return rt.rxPyReMethod(rv, name, rest)
}

func (rt *jsrt) rxPyReMethod(rv *jsObject, name string, a []interface{}) interface{} {
	switch name {
	case "search":
		return rt.rxPySearch(rv, rt.toString(argAt(a, 0)), rxPyIntArg(a, 1))
	case "match":
		return rt.rxPyAnchoredMatch(rv, rt.toString(argAt(a, 0)), rxPyIntArg(a, 1), false)
	case "fullmatch":
		return rt.rxPyAnchoredMatch(rv, rt.toString(argAt(a, 0)), rxPyIntArg(a, 1), true)
	case "findall":
		return rt.rxPyFindAll(rv, rt.toString(argAt(a, 0)), false)
	case "finditer":
		return rt.rxPyFindAll(rv, rt.toString(argAt(a, 0)), true)
	case "sub", "subn":
		text, n := rt.rxPySub(rv, argAt(a, 0), rt.toString(argAt(a, 1)), rxPyIntArg(a, 2))
		if name == "sub" {
			return text
		}
		return &jsArray{elems: []interface{}{text, float64(n)}}
	case "split":
		return rt.rxPySplit(rv, rt.toString(argAt(a, 0)), rxPyIntArg(a, 1))
	}
	rt.fail("unknown Pattern method: %s", name)
	return jsUndef
}

func (rt *jsrt) rxPyMatchMethod(md interface{}, name string, a []interface{}) interface{} {
	switch name {
	case "group":
		if len(a) == 0 {
			return rt.rxPyGroup(md, float64(0))
		}
		if len(a) == 1 {
			return rt.rxPyGroup(md, a[0])
		}
		out := &jsArray{}
		for _, k := range a {
			out.elems = append(out.elems, rt.rxPyGroup(md, k))
		}
		return out
	case "groups":
		return rxPyGroups(md)
	case "groupdict":
		return rxPyGroupDict(md)
	case "start":
		return rt.rxPySpan(md, rxPyKeyArg(a), 0)
	case "end":
		return rt.rxPySpan(md, rxPyKeyArg(a), 1)
	case "span":
		k := rxPyKeyArg(a)
		return &jsArray{elems: []interface{}{rt.rxPySpan(md, k, 0), rt.rxPySpan(md, k, 1)}}
	case "expand":
		re, text, m := rxMatchOf(md)
		return re.expand(text, m, rxPyTemplate(rt.toString(argAt(a, 0))))
	}
	rt.fail("unknown Match method: %s", name)
	return jsUndef
}

func rxPyKeyArg(a []interface{}) interface{} {
	if len(a) > 0 {
		return a[0]
	}
	return float64(0)
}

// rxPyRepr renders a pattern and a match the way CPython's repr does, so
// str(m)/print(m) agree with the interpreter grammar.
func rxPyRepr(v interface{}) (string, bool) {
	if rxPyIsRe(v) {
		o := v.(*jsObject)
		pat, _ := o.props["pattern"].(string)
		return "re.compile('" + pat + "')", true
	}
	if rxPyIsMatch(v) {
		re, text, m := rxMatchOf(v)
		g, _ := re.group(text, m, 0)
		return "<re.Match object; span=(" + strconv.Itoa(m.begin) + ", " + strconv.Itoa(m.end) +
			"), match='" + g + "'>", true
	}
	return "", false
}

func init() {
	rxExtraExterns = append(rxExtraExterns, func(rt *jsrt, m map[string]func(args []uint64) uint64) {
		u := rt.unwrap
		w := rt.wrap

		// The `re` module object. `import re` binds no module object in this flat
		// import model, so the grammar declares this one into the global scope; it
		// is a plain object, which js_pygetattr already reads re.I / re.DOTALL out
		// of without a wrapper of its own.
		m["js_pyre_module"] = func(a []uint64) uint64 { return w(rxPyModule()) }

		// The method-call wrapper: the re module, a compiled pattern and a match
		// dispatch here, everything else goes straight on to js_pymcall.
		basePyMcall := m["js_pymcall"]
		m["js_pyrxmcall"] = func(a []uint64) uint64 {
			target := u(a[0])
			if rxPyIsMod(target) || rxPyIsRe(target) || rxPyIsMatch(target) {
				arr, ok := u(a[2]).(*jsArray)
				if !ok {
					rt.fail("js_pyrxmcall args must be an array")
				}
				name := rt.toString(u(a[1]))
				if rxPyIsMod(target) {
					return w(rt.rxPyModCall(name, arr.elems))
				}
				if rxPyIsRe(target) {
					return w(rt.rxPyReMethod(target.(*jsObject), name, arr.elems))
				}
				return w(rt.rxPyMatchMethod(target, name, arr.elems))
			}
			return basePyMcall(a)
		}

		// str()/print() of a pattern or a match.
		basePyStr := m["js_pystr"]
		m["js_pyrxstr"] = func(a []uint64) uint64 {
			if s, ok := rxPyRepr(u(a[0])); ok {
				return rt.wrapStr(s)
			}
			return basePyStr(a)
		}
	})
}

func init() {
	// js_pyrxmcall reads its argument array in position 2, exactly like the
	// js_pymcall it wraps, so the handle recycler may hand that array through.
	jsThroughArgs["js_pyrxmcall"] = 1 << 2
}
