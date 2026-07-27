package abnf

// jsrtregexkt.go -- the Kotlin half of the shared regular-expression subsystem.
//
// abnf/jsrtregex.go holds the language-neutral engine (and Ruby's dispatcher);
// languages/lib/regex.js is its interpreter-side twin. This file is the Go twin of
// the "Regular expressions: Regex / MatchResult" section of
// languages/kotlin-interpreter.abnf: the same value model, the same flag mapping,
// the same method surface. The matrix runs the interpreter and the compiler on the
// same input and demands byte-identical stdout, so the two MUST stay in step.
//
// Nothing here rebinds an existing extern. The registrar below only ADDS js_rxkt*
// names, and each wrapper delegates to the extern it extends for every receiver
// that is not a Regex / MatchResult, so a Kotlin program that uses no regexps
// emits and runs exactly as before.
//
// Kotlin has no regexp LITERAL: everything arrives through Regex("...") or
// "...".toRegex(), so there is no lexer ambiguity and no tag-time literal to
// validate - a bad pattern is a clean runtime error from js_rxktnew.

import "strings"

func init() {
	rxExtraExterns = append(rxExtraExterns, func(rt *jsrt, m map[string]func(args []uint64) uint64) {
		rt.addKotlinRegexExterns(m)
	})
	// js_rxktmcall reads its argument array in position 2, exactly like the
	// js_mcall it wraps, so the handle recycler may hand that array through.
	jsThroughArgs["js_rxktmcall"] = 1 << 2
}

// ===== The value model =====
//
// A Regex is the neutral regexp object of jsrtregex.go (__rxsrc / __rxflags), so
// rxObjRe and friends apply unchanged. A MatchResult is the neutral match object
// plus __ktsrc, the source of the regexp the match CAME FROM - which differs from
// __rxsrc for matchEntire(), whose groups are read out of the anchored program.

func ktRxIsRegex(v interface{}) bool { return rxIsRegexObj(v) }

func ktRxIsMatch(v interface{}) bool { return rxIsMatchObj(v) }

func ktRxIsGroups(v interface{}) bool {
	o, ok := v.(*jsObject)
	if !ok {
		return false
	}
	return o.props["__ktgroups"] != nil
}

// ktRxIsCompanion answers the `Regex` name used as a value (Regex.escape(...)).
func ktRxIsCompanion(v interface{}) bool {
	o, ok := v.(*jsObject)
	if !ok {
		return false
	}
	return o.props["__ktrxco"] != nil
}

// ktRxNewMatch builds a MatchResult over `prog` (the plain or the anchored program)
// while remembering `origSrc`, the pattern of the Regex the match came from.
func ktRxNewMatch(prog *rxRe, origSrc, text string, m *rxMatch) *jsObject {
	o := rxNewMatchObj(prog.src, prog.flags, text, m)
	o.set("__ktsrc", origSrc)
	return o
}

// ktRxOrig is the plain (unanchored) program of the regexp a MatchResult came from.
func ktRxOrig(md interface{}) *rxRe {
	o := md.(*jsObject)
	src, _ := o.props["__ktsrc"].(string)
	flags, _ := o.props["__rxflags"].(string)
	return rxGet(src, flags)
}

// ktRxWhole is the both-ends-anchored twin of a program. Kotlin's matches() and
// matchEntire() require the WHOLE input; the engine spells that \A ... \z, and the
// wrapper group is non-capturing so the group numbering is unchanged.
func ktRxWhole(re *rxRe) *rxRe {
	return rxGet(`\A(?:`+re.src+ktRxWrapSep(re.flags)+`)\z`, re.flags)
}

// Under COMMENTS (the engine's x) a trailing '# ...' comment runs to the end of the
// LINE, so it would swallow the closing ')' of the anchoring wrapper. A newline
// closes the comment and is itself ignored in extended mode.
func ktRxWrapSep(flags string) string {
	if strings.ContainsRune(flags, 'x') {
		return "\n"
	}
	return ""
}

func (rt *jsrt) ktRxCheck(re *rxRe) *rxRe {
	if !re.ok {
		rt.fail("invalid regex '%s': %s", re.src, re.err)
	}
	return re
}

func (rt *jsrt) ktRxSearch(re *rxRe, text []rune, at int) *rxMatch {
	rt.ktRxCheck(re)
	m, ok := re.search(text, at)
	if !ok {
		rt.fail("regexp: step limit exceeded")
	}
	return m
}

// ===== Flags =====
//
// The engine's neutral letters are i (fold case), s (dot matches newline), m (^/$
// are LINE anchors) and x (extended). Kotlin's RegexOption maps onto them one for
// one; note that Kotlin, like Java and unlike Ruby, anchors ^/$ to the whole input
// UNLESS MULTILINE is given, so "m" is NOT forced on. LITERAL is not a match mode -
// it quotes the pattern. CANON_EQ and UNIX_LINES have no equivalent and are
// reported rather than silently ignored.
func (rt *jsrt) ktRxFlags(opts interface{}) (string, bool) {
	var names []string
	if !isUndefOrNull(opts) {
		if arr, isArr := opts.(*jsArray); isArr {
			for _, e := range arr.elems {
				names = append(names, rt.toString(e))
			}
		} else {
			names = append(names, rt.toString(opts))
		}
	}
	flags, literal := "", false
	for _, o := range names {
		switch o {
		case "IGNORE_CASE":
			flags += "i"
		case "DOT_MATCHES_ALL":
			flags += "s"
		case "MULTILINE":
			flags += "m"
		case "COMMENTS":
			flags += "x"
		case "LITERAL":
			literal = true
		case "CANON_EQ", "UNIX_LINES":
			rt.fail("RegexOption.%s is not supported", o)
		default:
			rt.fail("unknown RegexOption: %s", o)
		}
	}
	return flags, literal
}

// ktRxQuote is Regex.escape / RegexOption.LITERAL. Kotlin answers Pattern.quote's
// \Q...\E here; the shared engine has no \Q, so every metacharacter is backslashed
// instead. The MATCH is the same, the returned TEXT is not.
func ktRxQuote(s string) string {
	var out strings.Builder
	for _, c := range s {
		switch c {
		case '\n':
			out.WriteString(`\n`)
			continue
		case '\r':
			out.WriteString(`\r`)
			continue
		case '\t':
			out.WriteString(`\t`)
			continue
		}
		// A bare space and a # are ignored under the extended flag, so they are
		// quoted too - LITERAL must beat COMMENTS.
		if strings.ContainsRune(" #\\.^$|?*+()[]{}-", c) {
			out.WriteRune('\\')
		}
		out.WriteRune(c)
	}
	return out.String()
}

func (rt *jsrt) ktRxNew(pat, opts interface{}) *jsObject {
	flags, literal := rt.ktRxFlags(opts)
	src := rt.toString(pat)
	if literal {
		src = ktRxQuote(src)
	}
	rt.ktRxCheck(rxGet(src, flags))
	rt.ktRxCheck(rxGet(`\A(?:`+src+ktRxWrapSep(flags)+`)\z`, flags))
	return rxNewRegexObj(src, flags)
}

// ===== Replacement templates =====
//
// Kotlin's template is $1 / ${name}, with a backslash escaping the next character;
// the shared engine expands \1 / \k<name>. The rewrite happens here, mirroring
// krxTemplate in kotlin-interpreter.abnf.
func ktRxTemplate(t string) string {
	r := []rune(t)
	var out strings.Builder
	for i := 0; i < len(r); {
		c := r[i]
		if c == '\\' {
			if i+1 >= len(r) {
				out.WriteString(`\\`)
				i++
				continue
			}
			if r[i+1] == '\\' {
				out.WriteString(`\\`)
			} else {
				out.WriteRune(r[i+1])
			}
			i += 2
			continue
		}
		if c == '$' && i+1 < len(r) {
			d := r[i+1]
			if d >= '0' && d <= '9' {
				out.WriteRune('\\')
				out.WriteRune(d)
				i += 2
				continue
			}
			if d == '{' {
				j := i + 2
				for j < len(r) && r[j] != '}' {
					j++
				}
				nm := string(r[i+2 : j])
				if len(nm) > 0 && nm[0] >= '0' && nm[0] <= '9' {
					out.WriteString(`\` + nm[:1])
				} else {
					out.WriteString(`\k<` + nm + `>`)
				}
				i = j + 1
				continue
			}
		}
		out.WriteRune(c)
		i++
	}
	return out.String()
}

func ktRxEscapeRepl(s string) string {
	var out strings.Builder
	for _, c := range s {
		if c == '\\' || c == '$' {
			out.WriteRune('\\')
		}
		out.WriteRune(c)
	}
	return out.String()
}

// ===== The operations =====

func (rt *jsrt) ktRxReplace(re *rxRe, s, repl string, all bool) string {
	rt.ktRxCheck(re)
	text := []rune(s)
	var out strings.Builder
	at, last := 0, 0
	for at <= len(text) {
		m := rt.ktRxSearch(re, text, at)
		if m == nil {
			break
		}
		out.WriteString(string(text[last:m.begin]))
		out.WriteString(re.expand(text, m, repl))
		if m.end == m.begin {
			if m.end < len(text) {
				out.WriteRune(text[m.end])
			}
			last, at = m.end+1, m.end+1
		} else {
			last, at = m.end, m.end
		}
		if !all {
			break
		}
	}
	if last <= len(text) {
		out.WriteString(string(text[last:]))
	}
	return out.String()
}

// Regex#replace(input) { m -> ... }: the lambda produces each replacement.
func (rt *jsrt) ktRxReplaceFn(re *rxRe, s string, fn interface{}) string {
	text := []rune(s)
	var out strings.Builder
	at, last := 0, 0
	for at <= len(text) {
		m := rt.ktRxSearch(re, text, at)
		if m == nil {
			break
		}
		out.WriteString(string(text[last:m.begin]))
		md := ktRxNewMatch(re, re.src, s, m)
		out.WriteString(rt.toString(rt.call(fn, jsUndef, []interface{}{md})))
		if m.end == m.begin {
			if m.end < len(text) {
				out.WriteRune(text[m.end])
			}
			last, at = m.end+1, m.end+1
		} else {
			last, at = m.end, m.end
		}
	}
	if last <= len(text) {
		out.WriteString(string(text[last:]))
	}
	return out.String()
}

// Regex#split. limit 0 (Kotlin's default) means no limit; a positive limit stops
// after limit-1 cuts and keeps the rest of the input as the last part.
func (rt *jsrt) ktRxSplit(re *rxRe, s string, limit int) *jsArray {
	text := []rune(s)
	parts := &jsArray{}
	at, last := 0, 0
	for at <= len(text) {
		if limit > 0 && len(parts.elems) >= limit-1 {
			break
		}
		m := rt.ktRxSearch(re, text, at)
		if m == nil {
			break
		}
		if m.end == m.begin {
			if m.begin >= len(text) {
				break
			}
			parts.elems = append(parts.elems, string(text[last:m.begin+1]))
			last, at = m.begin+1, m.begin+1
		} else {
			parts.elems = append(parts.elems, string(text[last:m.begin]))
			last, at = m.end, m.end
		}
	}
	parts.elems = append(parts.elems, string(text[last:]))
	return parts
}

func (rt *jsrt) ktRxFindAll(re *rxRe, s string) *jsArray {
	text := []rune(s)
	out := &jsArray{}
	for at := 0; at <= len(text); {
		m := rt.ktRxSearch(re, text, at)
		if m == nil {
			break
		}
		out.elems = append(out.elems, ktRxNewMatch(re, re.src, s, m))
		if m.end == m.begin {
			at = m.end + 1
		} else {
			at = m.end
		}
	}
	return out
}

// groupValues: index 0 is the whole match and a group that did not take part reads
// as "" (Kotlin), unlike groups[i], which is null there.
func ktRxGroupValues(re *rxRe, text []rune, m *rxMatch) *jsArray {
	out := &jsArray{}
	for i := 0; i <= re.ngroups; i++ {
		if g, ok := re.group(text, m, i); ok {
			out.elems = append(out.elems, g)
		} else {
			out.elems = append(out.elems, "")
		}
	}
	return out
}

func ktRxRange(first, last int) *jsObject {
	o := newJSObject()
	o.set("first", float64(first))
	o.set("last", float64(last))
	return o
}

// m.groups[i] / m.groups["name"]: a MatchGroup (value + range) or null.
func (rt *jsrt) ktRxGroupAt(md interface{}, key interface{}) interface{} {
	re, text, m := rxMatchOf(md)
	// A group that took no part reads as null; an index or a name that does not
	// exist at all is an error, as it is on the JVM.
	i := -1
	if s, isStr := key.(string); isStr {
		i = re.indexOfName(s)
		if i < 0 {
			rt.fail("no group with name '%s'", s)
		}
	} else {
		i = jsToInt(rt.toNumber(key))
		if i < 0 || i > re.ngroups {
			rt.fail("group index %d out of range", i)
		}
	}
	g, ok := re.group(text, m, i)
	if !ok {
		return jsNull
	}
	o := newJSObject()
	o.set("value", g)
	o.set("range", ktRxRange(m.caps[2*i], m.caps[2*i+1]-1))
	return o
}

func ktRxArgInt(rt *jsrt, args []interface{}, i, dflt int) int {
	if i >= len(args) || isUndefOrNull(args[i]) {
		return dflt
	}
	return jsToInt(rt.toNumber(args[i]))
}

// ktRxMethod is the Go twin of krxMethod in kotlin-interpreter.abnf: it answers
// (value, true) when the call is one of ours and (nil, false) when it is not, so
// js_rxktmcall falls through to js_mcall for everything else.
func (rt *jsrt) ktRxMethod(target interface{}, name string, args []interface{}) (interface{}, bool) {
	if ktRxIsCompanion(target) {
		switch name {
		case "escape":
			return ktRxQuote(rt.toString(argAt(args, 0))), true
		case "escapeReplacement":
			return ktRxEscapeRepl(rt.toString(argAt(args, 0))), true
		case "fromLiteral":
			return rt.ktRxNew(argAt(args, 0), "LITERAL"), true
		}
		return nil, false
	}
	if s, isStr := target.(string); isStr {
		if name == "toRegex" {
			return rt.ktRxNew(s, argAt(args, 0)), true
		}
		// The String members that take a pattern. A non-Regex argument is left to
		// the ordinary String behaviour, so plain replace/split/contains are
		// untouched.
		if len(args) > 0 && ktRxIsRegex(args[0]) {
			re := rxObjRe(args[0])
			switch name {
			case "matches":
				return rt.ktRxSearch(ktRxWhole(re), []rune(s), 0) != nil, true
			case "contains":
				return rt.ktRxSearch(re, []rune(s), 0) != nil, true
			case "split":
				return rt.ktRxSplit(re, s, ktRxArgInt(rt, args, 1, 0)), true
			case "replace":
				if isCallable(argAt(args, 1)) {
					return rt.ktRxReplaceFn(re, s, args[1]), true
				}
				return rt.ktRxReplace(re, s, ktRxTemplate(rt.toString(argAt(args, 1))), true), true
			case "replaceFirst":
				return rt.ktRxReplace(re, s, ktRxTemplate(rt.toString(argAt(args, 1))), false), true
			}
		}
		return nil, false
	}
	if ktRxIsRegex(target) {
		re := rxObjRe(target)
		switch name {
		case "matches":
			return rt.ktRxSearch(ktRxWhole(re), []rune(rt.toString(argAt(args, 0))), 0) != nil, true
		case "containsMatchIn":
			return rt.ktRxSearch(re, []rune(rt.toString(argAt(args, 0))), 0) != nil, true
		case "find":
			s := rt.toString(argAt(args, 0))
			m := rt.ktRxSearch(re, []rune(s), ktRxArgInt(rt, args, 1, 0))
			if m == nil {
				return jsNull, true
			}
			return ktRxNewMatch(re, re.src, s, m), true
		case "findAll":
			return rt.ktRxFindAll(re, rt.toString(argAt(args, 0))), true
		case "matchEntire":
			s := rt.toString(argAt(args, 0))
			w := ktRxWhole(re)
			m := rt.ktRxSearch(w, []rune(s), 0)
			if m == nil {
				return jsNull, true
			}
			return ktRxNewMatch(w, re.src, s, m), true
		case "replace":
			if isCallable(argAt(args, 1)) {
				return rt.ktRxReplaceFn(re, rt.toString(argAt(args, 0)), args[1]), true
			}
			return rt.ktRxReplace(re, rt.toString(argAt(args, 0)),
				ktRxTemplate(rt.toString(argAt(args, 1))), true), true
		case "replaceFirst":
			return rt.ktRxReplace(re, rt.toString(argAt(args, 0)),
				ktRxTemplate(rt.toString(argAt(args, 1))), false), true
		case "split":
			return rt.ktRxSplit(re, rt.toString(argAt(args, 0)), ktRxArgInt(rt, args, 1, 0)), true
		case "toString":
			return re.src, true
		}
		rt.fail("unknown Regex method: %s", name)
	}
	if ktRxIsMatch(target) {
		re, text, m := rxMatchOf(target)
		switch name {
		case "next":
			at := m.end
			if m.end == m.begin {
				at = m.end + 1
			}
			orig := ktRxOrig(target)
			if at > len(text) {
				return jsNull, true
			}
			nm := rt.ktRxSearch(orig, text, at)
			if nm == nil {
				return jsNull, true
			}
			return ktRxNewMatch(orig, orig.src, string(text), nm), true
		case "toString":
			g, _ := re.group(text, m, 0)
			return g, true
		}
		rt.fail("unknown MatchResult method: %s", name)
	}
	return nil, false
}

// ktRxField is the Go twin of krxField: Regex.pattern and the MatchResult
// properties.
func (rt *jsrt) ktRxField(o interface{}, name string) (interface{}, bool) {
	if ktRxIsRegex(o) {
		switch name {
		case "pattern":
			return rxObjRe(o).src, true
		case "options":
			return &jsArray{}, true
		}
		return nil, false
	}
	if ktRxIsMatch(o) {
		re, text, m := rxMatchOf(o)
		switch name {
		case "value":
			g, _ := re.group(text, m, 0)
			return g, true
		case "range":
			return ktRxRange(m.begin, m.end-1), true
		case "groupValues":
			return ktRxGroupValues(re, text, m), true
		case "groups":
			g := newJSObject()
			g.set("__ktgroups", o)
			return g, true
		case "destructured":
			// `val (a, b) = m.destructured` reads componentN, and this subset takes
			// componentN of a LIST positionally - so the holder IS the group list
			// without the whole match, which both halves of the pair agree on.
			all := ktRxGroupValues(re, text, m)
			return &jsArray{elems: append([]interface{}{}, all.elems[1:]...)}, true
		}
		return nil, false
	}
	if ktRxIsGroups(o) {
		// The compiler grammar rewrites a `.size` read to `.length` (its lists are
		// arrays), so both spellings arrive here.
		if name == "size" || name == "length" {
			md := o.(*jsObject).props["__ktgroups"]
			re, _, _ := rxMatchOf(md)
			return float64(re.ngroups + 1), true
		}
		return nil, false
	}
	return nil, false
}

// ===== The externs =====

func (rt *jsrt) addKotlinRegexExterns(m map[string]func(args []uint64) uint64) {
	u := rt.unwrap
	w := rt.wrap

	// Regex(pattern) / Regex(pattern, options) / "pat".toRegex(): the pattern is
	// validated here, so a bad one is a clean runtime error rather than a wrong
	// match.
	m["js_rxktnew"] = func(a []uint64) uint64 {
		return w(rt.ktRxNew(u(a[0]), u(a[1])))
	}

	// The `RegexOption` and `Regex` names used as VALUES. A RegexOption is modelled
	// as its own name, so it prints the way Kotlin prints an enum entry.
	m["js_rxktglobal"] = func(a []uint64) uint64 {
		o := newJSObject()
		if rt.toString(u(a[0])) == "Regex" {
			o.set("__ktrxco", true)
			return w(o)
		}
		for _, n := range []string{"IGNORE_CASE", "MULTILINE", "DOT_MATCHES_ALL",
			"COMMENTS", "LITERAL", "CANON_EQ", "UNIX_LINES"} {
			o.set(n, n)
		}
		return w(o)
	}

	baseMcall := m["js_mcall"]
	m["js_rxktmcall"] = func(a []uint64) uint64 {
		arr, ok := u(a[2]).(*jsArray)
		if !ok {
			rt.fail("js_rxktmcall args must be an array")
		}
		target := u(a[0])
		if ktRxIsRegex(target) || ktRxIsMatch(target) || ktRxIsCompanion(target) ||
			(len(arr.elems) > 0 && ktRxIsRegex(arr.elems[0])) || isKtToRegex(target, rt.toString(u(a[1]))) {
			if v, handled := rt.ktRxMethod(target, rt.toString(u(a[1])), arr.elems); handled {
				return w(v)
			}
		}
		return baseMcall(a)
	}

	baseGet := m["js_get"]
	m["js_rxktget"] = func(a []uint64) uint64 {
		target := u(a[0])
		if ktRxIsRegex(target) || ktRxIsMatch(target) || ktRxIsGroups(target) {
			if v, handled := rt.ktRxField(target, rt.toString(u(a[1]))); handled {
				return w(v)
			}
		}
		return baseGet(a)
	}

	baseIndex := m["js_kindex"]
	m["js_rxktindex"] = func(a []uint64) uint64 {
		if ktRxIsGroups(u(a[0])) {
			md := u(a[0]).(*jsObject).props["__ktgroups"]
			return w(rt.ktRxGroupAt(md, u(a[1])))
		}
		return baseIndex(a)
	}
}

// isKtToRegex keeps "pat".toRegex() out of the base String dispatch, which knows
// nothing of the name.
func isKtToRegex(target interface{}, name string) bool {
	_, isStr := target.(string)
	return isStr && name == "toRegex"
}
