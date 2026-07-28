package abnf

// The JavaScript / TypeScript half of the shared regular-expression subsystem, for the
// COMPILER grammars (languages/js-to-llvm-ir.abnf and languages/typescript-to-llvm-ir.abnf).
//
// abnf/jsrtregex.go holds the ENGINE - the Go twin of languages/lib/regex.js - and Ruby's
// method dispatcher. The engine is language-neutral, but every language still needs its
// own receiver-and-name table, so this file adds JavaScript's and registers it through the
// rxExtraExterns extension point. Nothing here runs unless a program calls one of the
// js_jsx* externs, which only the two JS-family compiler grammars emit, and every extern
// added below is a NEW name that captures the one it extends and delegates to it for
// every receiver that is not a regular expression.
//
// The interpreter twin is the regular-expression block of languages/js-interpreter.abnf.
// Keep the two in step: ./test.sh runs the interpreter and the compiler on the same file
// and demands byte-identical stdout, which is exactly what catches a drift here.
//
// Positions are RUNE indices on this side and UTF-16 units on the interpreter side. They
// agree over the whole Basic Multilingual Plane, which is every test in this tree.

import "strings"

// ----- Flags -----
//
// Three of JavaScript's letters are match modes and map straight onto the engine's neutral
// set: i (fold case), s (dot matches newline) and m (^ and $ are line anchors). Unlike
// Ruby, 'm' must NOT be forced on - without it JavaScript's ^ and $ are string anchors.
// g (lastIndex iteration), y (sticky), d (hasIndices), u and v (the strict parse-time
// grammar, checked by the ReStrictU guard in the grammar) are not match modes at all.

func jsxHasFlag(flags string, letter byte) bool {
	return strings.IndexByte(flags, letter) >= 0
}

// jsxCanonFlags is RegExp#flags: the canonical order, not the written one.
func jsxCanonFlags(flags string) string {
	out := ""
	for _, c := range "dgimsuvy" {
		if strings.ContainsRune(flags, c) {
			out += string(c)
		}
	}
	return out
}

func jsxNeutral(flags string) string {
	// "o" is unconditional: in a non-unicode pattern a backslash-digit escape with no
	// such group is a LEGACY OCTAL character escape (Annex B), not a backreference -
	// /\1/.test("") is false in node because \1 is U+0001. Under u/v that spelling is
	// a SyntaxError, which the ReStrictU parse guard already refuses before anything
	// reaches the engine.
	// "j" is unconditional too: JavaScript has the POSIX-style loop reading. An
	// iteration of an unbounded repeat that consumed nothing is DISCARDED rather than
	// kept, so /^(a*)*$/.exec("aaa") reports "aaa" for group 1 (node), not "" (Perl,
	// Python, Ruby, Java). It also clears the captures inside a repeated atom at the
	// start of every iteration: /(?:(a)|b)+/.exec("ab") leaves group 1 undefined.
	out := "oj"
	if jsxHasFlag(flags, 'i') {
		out += "i"
	}
	if jsxHasFlag(flags, 's') {
		out += "s"
	}
	if jsxHasFlag(flags, 'm') {
		out += "m"
	}
	return out
}

// jsxNewRegex builds the RegExp value: the __rx slots the shared engine reads plus the
// properties JavaScript exposes. lastIndex is per-OBJECT state, so a literal inside a loop
// must build a fresh one on every evaluation - which is what the emitted js_jsxnew does.
func jsxNewRegex(src, jsFlags string) *jsObject {
	fl := jsxCanonFlags(jsFlags)
	o := rxNewRegexObj(src, jsxNeutral(fl))
	o.set("source", src)
	o.set("flags", fl)
	o.set("global", jsxHasFlag(fl, 'g'))
	o.set("sticky", jsxHasFlag(fl, 'y'))
	o.set("ignoreCase", jsxHasFlag(fl, 'i'))
	o.set("multiline", jsxHasFlag(fl, 'm'))
	o.set("dotAll", jsxHasFlag(fl, 's'))
	o.set("unicode", jsxHasFlag(fl, 'u'))
	o.set("unicodeSets", jsxHasFlag(fl, 'v'))
	o.set("hasIndices", jsxHasFlag(fl, 'd'))
	o.set("lastIndex", float64(0))
	return o
}

func jsxBool(o *jsObject, key string) bool {
	b, _ := o.props[key].(bool)
	return b
}

func jsxLastIndex(o *jsObject) int {
	f, ok := o.props["lastIndex"].(float64)
	if !ok {
		return 0
	}
	return int(f)
}

// jsxProg is the compiled program behind a RegExp value. A pattern the shared engine
// refuses is NOT reported here at construction: the engine is deliberately one permissive
// dialect for four languages, so it rejects a few patterns JavaScript accepts (lookbehind,
// [[:alpha:]]). The failure is raised when the pattern is actually MATCHED, exactly as
// js-interpreter.abnf's reProgOf does.
func (rt *jsrt) jsxProg(o *jsObject) *rxRe {
	re := rxObjRe(o)
	if !re.ok {
		src, _ := o.props["source"].(string)
		rt.fail("invalid regular expression: /%s/: %s", src, re.err)
	}
	return re
}

// jsxMatchAt is the sticky-flag counterpart of jsxSearch: the match must BEGIN at
// `at`, with no scanning forward.
func (rt *jsrt) jsxMatchAt(re *rxRe, text []rune, at int) *rxMatch {
	m, ok := re.matchAt(text, at)
	if !ok {
		rt.fail("regexp: step limit exceeded")
	}
	return m
}

func (rt *jsrt) jsxSearch(re *rxRe, text []rune, at int) *rxMatch {
	m, ok := re.search(text, at)
	if !ok {
		rt.fail("regexp: step limit exceeded")
	}
	return m
}

// jsxNameAt answers the name of capture group i, or "" when it has none.
func jsxNameAt(re *rxRe, i int) string {
	for k, g := range re.nameGroups {
		if g == i {
			return re.names[k]
		}
	}
	return ""
}

// jsxResult is the match record. JavaScript models it as an Array carrying three extra
// properties; neither runtime here allows a non-index property on an array (both report
// "invalid array index"), so it is an array-LIKE object instead - m[0], m.length, m.index,
// m.input and m.groups all read back, and only Array.isArray(m) tells the difference. The
// interpreter side builds exactly the same shape.
func (rt *jsrt) jsxResult(re *rxRe, text []rune, m *rxMatch, input string) *jsObject {
	o := newJSObject()
	for i := 0; i <= re.ngroups; i++ {
		if g, ok := re.group(text, m, i); ok {
			o.set(itoaSmall(i), g)
		} else {
			o.set(itoaSmall(i), jsUndef)
		}
	}
	o.set("length", float64(re.ngroups+1))
	o.set("index", float64(m.begin))
	o.set("input", input)
	var groups interface{} = jsUndef
	if len(re.names) > 0 {
		g := newJSObject()
		for i := 1; i <= re.ngroups; i++ {
			name := jsxNameAt(re, i)
			if name == "" {
				continue
			}
			if v, ok := re.group(text, m, i); ok {
				g.set(name, v)
			} else {
				g.set(name, jsUndef)
			}
		}
		groups = g
	}
	o.set("groups", groups)
	return o
}

func itoaSmall(i int) string {
	if i < 10 {
		return string(rune('0' + i))
	}
	return itoaSmall(i/10) + string(rune('0'+i%10))
}

// jsxExec is RegExp#exec: g and y both read the search start out of lastIndex and reset it
// to 0 on failure, and y additionally refuses a match that does not BEGIN there.
func (rt *jsrt) jsxExec(o *jsObject, s string) interface{} {
	re := rt.jsxProg(o)
	text := []rune(s)
	useLast := jsxBool(o, "global") || jsxBool(o, "sticky")
	at := 0
	if useLast {
		at = jsxLastIndex(o)
	}
	if at < 0 || at > len(text) {
		if useLast {
			o.set("lastIndex", float64(0))
		}
		return jsNull
	}
	// Sticky is the engine's own anchored entry point: matchAt tries ONE position
	// instead of searching forward and then throwing the answer away.
	var m *rxMatch
	if jsxBool(o, "sticky") {
		m = rt.jsxMatchAt(re, text, at)
	} else {
		m = rt.jsxSearch(re, text, at)
	}
	if m == nil {
		if useLast {
			o.set("lastIndex", float64(0))
		}
		return jsNull
	}
	if useLast {
		o.set("lastIndex", float64(m.end))
	}
	return rt.jsxResult(re, text, m, s)
}

func (rt *jsrt) jsxTest(o *jsObject, s string) bool {
	if jsxBool(o, "global") || jsxBool(o, "sticky") {
		return rt.jsxExec(o, s) != jsNull
	}
	return rt.jsxSearch(rt.jsxProg(o), []rune(s), 0) != nil
}

// jsxMatchAll walks every match. whole=true collects just the matched text (String#match
// with g), whole=false the full match records (String#matchAll). JavaScript's matchAll
// hands back an iterator; this hands back the array a spread would build, which is what a
// for-of over it needs anyway - the interpreter does the same.
func (rt *jsrt) jsxMatchAll(o *jsObject, s string, whole bool) *jsArray {
	re := rt.jsxProg(o)
	text := []rune(s)
	out := &jsArray{}
	for at := 0; at <= len(text); {
		m := rt.jsxSearch(re, text, at)
		if m == nil {
			break
		}
		if whole {
			g, _ := re.group(text, m, 0)
			out.elems = append(out.elems, g)
		} else {
			out.elems = append(out.elems, rt.jsxResult(re, text, m, s))
		}
		if m.end == m.begin {
			at = m.end + 1
		} else {
			at = m.end
		}
	}
	o.set("lastIndex", float64(0))
	return out
}

// jsxSplit follows SplitMatcher: the match is ANCHORED at the scan position, an empty
// match at the current segment start does not split, and the separator's capture groups
// are part of the output. Ruby's rxSplit answers a different shape, so this is its own.
func (rt *jsrt) jsxSplit(o *jsObject, s string, limit int) *jsArray {
	re := rt.jsxProg(o)
	text := []rune(s)
	out := &jsArray{}
	if len(text) == 0 {
		if rt.jsxSearch(re, text, 0) == nil {
			out.elems = append(out.elems, "")
		}
		return out
	}
	if limit == 0 {
		return out
	}
	p, q := 0, 0
	for q < len(text) {
		m := rt.jsxSearch(re, text, q)
		if m == nil || m.begin != q || m.end == p {
			q++
			continue
		}
		out.elems = append(out.elems, string(text[p:q]))
		if limit >= 0 && len(out.elems) >= limit {
			return out
		}
		for i := 1; i <= re.ngroups; i++ {
			if g, ok := re.group(text, m, i); ok {
				out.elems = append(out.elems, g)
			} else {
				out.elems = append(out.elems, jsUndef)
			}
			if limit >= 0 && len(out.elems) >= limit {
				return out
			}
		}
		p = m.end
		q = p
	}
	out.elems = append(out.elems, string(text[p:]))
	return out
}

// jsxTemplate rewrites JavaScript's replacement template into the one rxRe.expand reads.
// The engine expands \1 / \k<name>, which is what Ruby and Python write; JavaScript writes
// $1 / $<name> / $& / $$. The rewrite belongs to the language, not to the engine - the
// interpreter's jsReTemplate is the same function.
// Two forms are NOT covered, because the engine has nothing to rewrite them to: the prefix
// $` and the suffix $' stay the two characters they are written with, and $10 and up read
// as $1 followed by a '0', since expand takes one digit. The interpreter side has exactly
// the same two gaps.
func jsxTemplate(t string) string {
	var out strings.Builder
	r := []rune(t)
	for i := 0; i < len(r); {
		if r[i] == '\\' { // a literal backslash has to survive rxRe.expand
			out.WriteString(`\\`)
			i++
			continue
		}
		if r[i] != '$' || i+1 >= len(r) {
			out.WriteRune(r[i])
			i++
			continue
		}
		d := r[i+1]
		if d == '$' {
			out.WriteRune('$')
			i += 2
			continue
		}
		if d == '&' {
			out.WriteString(`\0`)
			i += 2
			continue
		}
		if d >= '1' && d <= '9' {
			out.WriteRune('\\')
			out.WriteRune(d)
			i += 2
			continue
		}
		if d == '<' {
			j := i + 2
			for j < len(r) && r[j] != '>' {
				j++
			}
			out.WriteString(`\k<`)
			out.WriteString(string(r[i+2 : j]))
			out.WriteString(">")
			i = j + 1
			continue
		}
		out.WriteRune('$')
		i++
	}
	return out.String()
}

// jsxReplace is String#replace / #replaceAll with a pattern. A STRING replacement goes
// through the engine's own expander after the template rewrite; a FUNCTION replacement is
// called once per match, with the whole match, the groups, the offset and the input.
func (rt *jsrt) jsxReplace(o *jsObject, s string, repl interface{}, forceAll bool) string {
	re := rt.jsxProg(o)
	all := forceAll || jsxBool(o, "global")
	if rt.typeOf(repl) != "function" {
		res := rt.rxReplace(o, s, jsxTemplate(rt.toString(repl)), all)
		if jsxBool(o, "global") {
			o.set("lastIndex", float64(0))
		}
		return res
	}
	text := []rune(s)
	var out strings.Builder
	at, last := 0, 0
	for at <= len(text) {
		m := rt.jsxSearch(re, text, at)
		if m == nil {
			break
		}
		out.WriteString(string(text[last:m.begin]))
		args := []interface{}{}
		for i := 0; i <= re.ngroups; i++ {
			if g, ok := re.group(text, m, i); ok {
				args = append(args, g)
			} else {
				args = append(args, jsUndef)
			}
		}
		args = append(args, float64(m.begin), s)
		out.WriteString(rt.toString(rt.call(repl, jsUndef, args)))
		last, at = m.end, m.end
		if m.end == m.begin {
			if m.end < len(text) {
				out.WriteRune(text[m.end])
			}
			last, at = m.end+1, m.end+1
		}
		if !all {
			break
		}
	}
	if last <= len(text) {
		out.WriteString(string(text[last:]))
	}
	if jsxBool(o, "global") {
		o.set("lastIndex", float64(0))
	}
	return out.String()
}

// jsxRegexArg turns the first argument of String#match / #matchAll / #search into a
// pattern: a RegExp is used as it is, anything else is its SOURCE, exactly as in
// JavaScript ("a1".match("\\d") matches a digit).
func (rt *jsrt) jsxRegexArg(v interface{}) *jsObject {
	if rxIsRegexObj(v) {
		return v.(*jsObject)
	}
	if isUndefOrNull(v) {
		return jsxNewRegex("", "")
	}
	return jsxNewRegex(rt.toString(v), "")
}

// jsxMethod is the dispatcher: a RegExp receiver, or a string receiver whose argument is a
// pattern. It answers handled=false for everything else, which is every call in a program
// that uses no regular expression at all.
func (rt *jsrt) jsxMethod(target interface{}, name string, args []interface{}) (interface{}, bool) {
	if rxIsRegexObj(target) {
		o := target.(*jsObject)
		switch name {
		case "test":
			return rt.jsxTest(o, rt.toString(argAt(args, 0))), true
		case "exec":
			return rt.jsxExec(o, rt.toString(argAt(args, 0))), true
		case "toString":
			src, _ := o.props["source"].(string)
			fl, _ := o.props["flags"].(string)
			return "/" + src + "/" + fl, true
		}
		rt.fail("unknown RegExp method: %s", name)
	}
	s, isStr := target.(string)
	if !isStr {
		return nil, false
	}
	switch name {
	case "match", "matchAll", "search":
		re := rt.jsxRegexArg(argAt(args, 0))
		if name == "search" {
			m := rt.jsxSearch(rt.jsxProg(re), []rune(s), 0)
			if m == nil {
				return float64(-1), true
			}
			return float64(m.begin), true
		}
		if name == "matchAll" {
			return rt.jsxMatchAll(re, s, false), true
		}
		if jsxBool(re, "global") {
			all := rt.jsxMatchAll(re, s, true)
			if len(all.elems) == 0 {
				return jsNull, true
			}
			return all, true
		}
		return rt.jsxExec(re, s), true
	case "replace", "replaceAll", "split":
		if !rxIsRegexObj(argAt(args, 0)) {
			return nil, false // a plain-string argument keeps the base behaviour
		}
		re := argAt(args, 0).(*jsObject)
		switch name {
		case "replace":
			return rt.jsxReplace(re, s, argAt(args, 1), false), true
		case "replaceAll":
			return rt.jsxReplace(re, s, argAt(args, 1), true), true
		}
		limit := -1
		if v := argAt(args, 1); !isUndefOrNull(v) {
			limit = int(rt.toNumber(v))
		}
		return rt.jsxSplit(re, s, limit), true
	}
	return nil, false
}

// jsxCtorArgs is new RegExp(src, flags) / RegExp(src, flags): the same construction as a
// literal, with the flags arriving at run time.
func (rt *jsrt) jsxCtorArgs(args []interface{}) *jsObject {
	a0 := argAt(args, 0)
	src, fl := "", ""
	if rxIsRegexObj(a0) {
		o := a0.(*jsObject)
		src, _ = o.props["source"].(string)
		fl, _ = o.props["flags"].(string)
	} else if !isUndefOrNull(a0) {
		src = rt.toString(a0)
	}
	if v := argAt(args, 1); !isUndefOrNull(v) {
		fl = rt.toString(v)
	}
	return jsxNewRegex(src, fl)
}

func init() {
	rxExtraExterns = append(rxExtraExterns, func(rt *jsrt, m map[string]func(args []uint64) uint64) {
		u := rt.unwrap
		w := rt.wrap

		// A regexp literal: /src/flags, with the JavaScript flag letters.
		m["js_jsxnew"] = func(a []uint64) uint64 {
			return w(jsxNewRegex(rt.toString(u(a[0])), rt.toString(u(a[1]))))
		}
		// new RegExp(...): the argument array of the emitted call site.
		m["js_jsxregexp"] = func(a []uint64) uint64 {
			arr, ok := u(a[0]).(*jsArray)
			if !ok {
				rt.fail("js_jsxregexp needs an argument array")
			}
			return w(rt.jsxCtorArgs(arr.elems))
		}
		// The global binding, so a bare RegExp(...) call and `typeof RegExp` work. The
		// grammar declares it in the program's own scope, so no other language sees it.
		m["js_jsxctor"] = func(a []uint64) uint64 {
			return w(jsHostFunc("RegExp", func(rt *jsrt, this uint64, args []interface{}) interface{} {
				return rt.jsxCtorArgs(args)
			}))
		}

		// The method-call wrapper. js_get answers undefined for every RegExp method and
		// for the string methods this runtime has no native for (match, matchAll, search,
		// replaceAll), so the emitted call lands in the js_mcall arm and reaches here.
		baseMcall := m["js_mcall"]
		m["js_jsxmcall"] = func(a []uint64) uint64 {
			arr, ok := u(a[2]).(*jsArray)
			if !ok {
				rt.fail("js_jsxmcall args must be an array")
			}
			target := u(a[0])
			_, isStr := target.(string)
			if rxIsRegexObj(target) || isStr {
				if v, handled := rt.jsxMethod(target, rt.toString(u(a[1])), arr.elems); handled {
					return w(v)
				}
			}
			return baseMcall(a)
		}

		// The call wrapper. replace and split DO have a native for a string receiver, so
		// they come through js_call as a bound method instead; only those two, and only
		// with a pattern argument, are taken over here.
		baseCall := m["js_call"]
		m["js_jsxcall"] = func(a []uint64) uint64 {
			if bm, ok := u(a[0]).(*boundMethod); ok {
				if s, isStr := bm.recv.(string); isStr && (bm.name == "replace" || bm.name == "split") {
					if arr, isArr := u(a[2]).(*jsArray); isArr && rxIsRegexObj(argAt(arr.elems, 0)) {
						if v, handled := rt.jsxMethod(s, bm.name, arr.elems); handled {
							return w(v)
						}
					}
				}
			}
			return baseCall(a)
		}
	})
}

func init() {
	// js_jsxmcall and js_jsxcall read their argument array in position 2, exactly like
	// the js_mcall / js_call they wrap, so the handle recycler may hand that array on.
	jsThroughArgs["js_jsxmcall"] = 1 << 2
	jsThroughArgs["js_jsxcall"] = 1 << 2
}
