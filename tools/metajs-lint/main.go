// metajs-lint makes the MetaJS dialect's gaps LOUD (docs/todo.md 6.4).
//
// docs/abnf-dialect-gotchas.md lists a dozen ways MetaJS is not JavaScript, and
// until now NOTHING ENFORCED ANY OF THEM. Every one is discovered the same way:
// a confusing failure, usually under `-frozen` only, usually behind a green goja
// run, and in one case (`\f`) in the NATIVE BINARY ONLY, where no gate in this
// project can see it at all. Each has cost at least one agent-round.
//
// WHY A STANDALONE TOOL AND NOT A `-verify` PASS. `mec <grammar> -verify` takes a
// GRAMMAR and checks it after loading; abnf/verifier.go's whole surface is rules,
// tags and script bindings for one .abnf file. The corpus that matters most here
// is languages/lib/*.metajs, which is not a grammar and is never loaded by
// -verify at all - it is compiled by metajs-to-llvm-ir.abnf as a source program.
// Wiring layer-2 files through -verify would mean inventing a second meaning for
// the flag. A standalone tool also runs over the WHOLE corpus in one command,
// the way `go run ./tools/shape-scan` does, which is what makes it usable as a
// gate; and it touches no shared Go code, which matters when other work is in
// flight in abnf/.
//
// WHAT IT CHECKS. Every rule below is a line in docs/abnf-dialect-gotchas.md or
// chapter 4 of docs/working-on-this-project.md, and every one has a recorded
// cost:
//
//	exponent-literal   `1e308` LEXES AS `1` FOLLOWED BY THE NAME `e308`. The
//	                   grammar becomes unparseable under -frozen with "cannot
//	                   parse script", pointing nowhere near the line. Two agents
//	                   hit it independently on one day.
//	string-escape      MetaJS's string literal has no \f and no \v AND DROPS THE
//	                   BACKSLASH, so `ch == "\f"` compiles to `ch == "f"`. This
//	                   ate the f's of Integer("0xff") in the native binary only -
//	                   goja and the Go twin were green, so the matrix, --full and
//	                   --cross were ALL blind to it.
//	asi-return         a value-less `return` followed by an expression on the
//	                   next line parses as `return (expr)`. It silently lost
//	                   EVERY STORE in one half for an entire agent-round.
//	to-precision       no toPrecision.
//	for-in             no for...in under -frozen.
//	frozen-builtin     no splice and no lastIndexOf; 2-argument indexOf IGNORES
//	                   the from-index - it does not raise, it returns the SAME
//	                   match every time, so a scan loop spins to the IR step cap.
//	has-own            `hasOwn` exists in interp-core.js and NOT in
//	                   compile-core.js: an emitter that uses it works in one half.
//	keys-of            `keysOf` is not bound for grammar scripts at all - only
//	                   for layer-2 MetaJS. goja answers ReferenceError.
//	extern-from-lay2   an extern is callable FROM AN EMITTED MODULE, never from
//	                   layer 2: `js_keys(o)` in a *-rt.metajs links, passes clang
//	                   and dies natively with "js_keys is not defined".
//	typed-local        MetaJS IS TYPED: a reassigned local keeps its first type,
//	                   and the failure is -frozen-only, so --full reads
//	                   FROZEN-DIFF while goja is green.
//	arguments-in-try   `arguments` is not resolved from inside a try block.
//	dup-function       two top-level functions of one name: in a grammar script
//	                   they silently rebind tags; in a layer-2 file they are
//	                   `redefinition of global`.
//	warn-on-stdout     a warning on println lands INSIDE the emitted module.
//
// THREE CLAIMS IN docs/abnf-dialect-gotchas.md ARE STALE and are deliberately NOT
// rules here, because a rule that fires on working code is how a linter gets
// switched off. MEASURED 2026-08-10 with a two-line probe grammar in both script
// hosts and with a MetaJS program compiled to a native binary:
//
//	string.split   works   (grammar script goja AND -frozen, and layer 2)
//	array.join     works   (same three)
//	array.length = works   (same three; ruby-rt.metajs:6858 does it and the
//	                        native ruby slice-assign rows are correct)
//
// The doc lists all three under "The frozen MetaJS engine" as absent. They are
// not, today. What IS absent there, measured the same way: splice, lastIndexOf,
// toPrecision, for...in - and indexOf's from-index is accepted and ignored,
// which is the dangerous one because it is silent.
//
// Each finding prints file:line and the rule name. Nothing here is a heuristic
// about intent: every rule fires on a construct that is wrong in this dialect.
//
// Usage:
//
//	go run ./tools/metajs-lint                    # languages/lib/*.metajs and lib/*.js
//	go run ./tools/metajs-lint -grammars          # ALSO every languages/*.abnf script block
//	go run ./tools/metajs-lint -only asi-return   # one rule
//	go run ./tools/metajs-lint -max 0             # exit 1 above N findings: the gate
//	go run ./tools/metajs-lint -list              # the rules and what each costs
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type finding struct {
	rule string
	file string
	line int
	text string
	note string
}

// ---------------------------------------------------------------------------
// A tokenizer, because every rule here is about what is CODE and what is not.
// Half of the false positives a grep-based version would produce come from the
// word `return` in a comment and from `\f` inside a regex character class in a
// comment block.

type tokKind int

const (
	tIdent tokKind = iota
	tNum
	tStr
	tPunct
	tComment
)

type token struct {
	kind tokKind
	text string
	line int
	// for strings: the raw text including quotes
}

func lex(src string) []token {
	var out []token
	line := 1
	i, n := 0, len(src)
	for i < n {
		c := src[i]
		if c == '\n' {
			line++
			i++
			continue
		}
		if c == ' ' || c == '\t' || c == '\r' {
			i++
			continue
		}
		if c == '/' && i+1 < n && src[i+1] == '*' {
			st := line
			j := i + 2
			for j+1 < n && !(src[j] == '*' && src[j+1] == '/') {
				if src[j] == '\n' {
					line++
				}
				j++
			}
			out = append(out, token{tComment, src[i:min(j+2, n)], st})
			i = j + 2
			continue
		}
		if c == '/' && i+1 < n && src[i+1] == '/' {
			j := i
			for j < n && src[j] != '\n' {
				j++
			}
			out = append(out, token{tComment, src[i:j], line})
			i = j
			continue
		}
		if c == '"' || c == '\'' {
			st := line
			j := i + 1
			for j < n && src[j] != c {
				if src[j] == '\\' && j+1 < n {
					j++
				}
				if src[j] == '\n' {
					line++
				}
				j++
			}
			out = append(out, token{tStr, src[i:min(j+1, n)], st})
			i = j + 1
			continue
		}
		if c >= '0' && c <= '9' {
			// MetaJS's NUMBER RULE, not JavaScript's, and the difference IS the
			// exponent bug: lib/numbers.abnf's DecLit is digits with an optional
			// fraction and NO exponent form, plus a 0x hex form. So `1e308`
			// lexes here exactly as the frozen engine lexes it - the number `1`
			// followed by the NAME `e308` - and checkExponent is then a plain
			// token-adjacency test. Eating the exponent the way a JS lexer would
			// is precisely how this rule silently stops firing.
			j := i
			if c == '0' && j+1 < n && (src[j+1] == 'x' || src[j+1] == 'X') {
				j += 2
				for j < n && (src[j] >= '0' && src[j] <= '9' ||
					src[j] >= 'a' && src[j] <= 'f' || src[j] >= 'A' && src[j] <= 'F') {
					j++
				}
			} else {
				for j < n && src[j] >= '0' && src[j] <= '9' {
					j++
				}
				if j < n && src[j] == '.' && j+1 < n && src[j+1] >= '0' && src[j+1] <= '9' {
					j++
					for j < n && src[j] >= '0' && src[j] <= '9' {
						j++
					}
				}
			}
			out = append(out, token{tNum, src[i:j], line})
			i = j
			continue
		}
		if c == '_' || c == '$' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' {
			j := i
			for j < n && (src[j] == '_' || src[j] == '$' || isAlnum(src[j])) {
				j++
			}
			out = append(out, token{tIdent, src[i:j], line})
			i = j
			continue
		}
		out = append(out, token{tPunct, string(c), line})
		i++
	}
	return out
}

func isAlnum(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// code returns the tokens with comments dropped.
func code(ts []token) []token {
	var out []token
	for _, t := range ts {
		if t.kind != tComment {
			out = append(out, t)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// The rules.

type unit struct {
	file    string // path as reported
	src     string
	toks    []token
	isLay2  bool // languages/lib/*.metajs - layer 2, compiled by metajs-to-llvm-ir
	isCore  bool // languages/lib/*.js - a shared script core run by the grammar hosts
	isCompi bool // a compiler-half grammar or compile-core.js
	isGram  bool // extracted from a .abnf script block
	lineOff int  // added to token lines (for .abnf extraction)
}

func (u *unit) add(fs *[]finding, rule string, line int, text, note string) {
	*fs = append(*fs, finding{rule, u.file, line + u.lineOff, strings.TrimSpace(text), note})
}

// rExpLit: an exponent literal written in source. The lexer above deliberately
// does NOT eat the exponent (MetaJS's DecLit does not either), so `1e308`
// arrives as the number `1` followed by the identifier `e308` - which is exactly
// what the frozen engine sees, and is why this rule is a token adjacency test
// rather than a regex over the text.
var rExpDigits = regexp.MustCompile(`^[eE][+-]?[0-9]+$`)

func checkExponent(u *unit, fs *[]finding) {
	ts := code(u.toks)
	for i := 0; i+1 < len(ts); i++ {
		if ts[i].kind != tNum || ts[i+1].kind != tIdent || ts[i+1].line != ts[i].line {
			continue
		}
		if !rExpDigits.MatchString(ts[i+1].text) {
			continue
		}
		// `1 e5` with a space is the same token sequence but is not a literal
		// anybody wrote on purpose; both are broken, so both are reported.
		u.add(fs, "exponent-literal", ts[i].line, ts[i].text+ts[i+1].text,
			"MetaJS has no exponent literal: this lexes as the number "+ts[i].text+
				" followed by the NAME "+ts[i+1].text+". Under -frozen the whole script becomes "+
				"\"cannot parse script\". Write parseFloat(\"...\") or 1e307 * 10 as a product.")
	}
}

func checkStringEscapes(u *unit, fs *[]finding) {
	for _, t := range u.toks {
		if t.kind != tStr {
			continue
		}
		s := t.text
		for i := 0; i+1 < len(s); i++ {
			if s[i] != '\\' {
				continue
			}
			e := s[i+1]
			if e == 'f' || e == 'v' {
				// WHERE it is wrong depends on the surface, and saying the wrong
				// one sends the reader to the wrong gate. A layer-2 file is
				// compiled by metajs-to-llvm-ir, so the escape is wrong in the
				// NATIVE BINARY and nowhere else - the matrix, --full and --cross
				// all run llvm.Run against the Go twin and cannot see it. A
				// grammar script or shared core is parsed by real JavaScript under
				// goja and by the frozen MetaJS grammar under -frozen, so it is a
				// live goja-vs--frozen DIVERGENCE, which ./test.sh would catch if
				// any entry exercised the line.
				where := "under -frozen ONLY - goja parses this file as real JavaScript, so the two " +
					"engines genuinely disagree and ./test.sh sees it only if an entry reaches the line"
				if u.isLay2 {
					where = "in the NATIVE BINARY ONLY - the matrix, --full and --cross all run llvm.Run " +
						"against the Go twin and are structurally blind to it"
				}
				u.add(fs, "string-escape", t.line, fmt.Sprintf("\\%c in %s", e, s),
					fmt.Sprintf("MetaJS's string literal has no \\%c and DROPS THE BACKSLASH, so this "+
						"compiles to a literal '%c'. Wrong %s. Use the character code.", e, e, where))
			}
			i++ // skip the escaped char, so \\\\f is not read as \\f
		}
	}
}

// checkASIReturn: a value-less `return` at end of line, followed by something
// that is not `}` and not another statement keyword that cannot be an operand.
// `return` followed by `;`, `}`, or an expression ON THE SAME LINE is fine.
func checkASIReturn(u *unit, fs *[]finding) {
	ts := code(u.toks)
	for i := 0; i < len(ts); i++ {
		if ts[i].kind != tIdent || ts[i].text != "return" {
			continue
		}
		if i+1 >= len(ts) {
			continue
		}
		nx := ts[i+1]
		if nx.line == ts[i].line {
			continue // has an expression (or a `;`) on the same line
		}
		if nx.kind == tPunct && (nx.text == "}" || nx.text == ";") {
			continue // `return` then a closing brace: nothing to swallow
		}
		u.add(fs, "asi-return", ts[i].line, "return <newline> "+nx.text,
			"a value-less `return` is NOT terminated by the newline in this dialect: "+
				"the next line is parsed as its operand and is EVALUATED. Brace it: `{ return }`.")
	}
}

// checkNamedCalls. localFns is every `function NAME` declared anywhere in the
// same FILE (all of a grammar's script blocks together), because a grammar that
// defines its own helper is not using the missing one. Four grammars define
// their own `hasOwn` in exactly this way - c, bash, batch and lisp - and
// without this the rule reported 47 findings, every one of them wrong. A linter
// that cries wolf 47 times is a linter nobody runs.
func checkNamedCalls(u *unit, fs *[]finding, lay2Defined, localFns map[string]bool) {
	ts := code(u.toks)
	for i := 0; i < len(ts); i++ {
		t := ts[i]
		if t.kind != tIdent {
			continue
		}
		call := i+1 < len(ts) && ts[i+1].kind == tPunct && ts[i+1].text == "("
		member := i > 0 && ts[i-1].kind == tPunct && ts[i-1].text == "."

		switch {
		case member && t.text == "toPrecision":
			u.add(fs, "to-precision", t.line, ".toPrecision(", "MetaJS has no toPrecision. Choose the digit count by round-trip with floPrec/parseFloat instead.")
		case member && t.text == "splice":
			u.add(fs, "frozen-builtin", t.line, ".splice(", "no array.splice under -frozen. Shift indices and pop().")
		case member && t.text == "lastIndexOf":
			u.add(fs, "frozen-builtin", t.line, ".lastIndexOf(", "no string.lastIndexOf under -frozen.")
		case member && t.text == "indexOf" && call && argCount(ts, i+1) >= 2:
			u.add(fs, "frozen-builtin", t.line, ".indexOf(x, from)",
				"indexOf IGNORES its from-index under -frozen - it does not raise, it returns the "+
					"SAME match every time, so a scan loop spins to the IR step cap. Write a for scan.")
		case call && t.text == "hasOwn" && u.isCompi && !localFns["hasOwn"]:
			u.add(fs, "has-own", t.line, "hasOwn(",
				"hasOwn is defined in lib/interp-core.js and NOT in lib/compile-core.js, so this "+
					"works in the interpreter half and is undefined in the compiler half. "+
					"Test map[\"$k\"] == undefined instead.")
		case call && t.text == "keysOf" && (u.isGram || u.isCore) && !localFns["keysOf"]:
			u.add(fs, "keys-of", t.line, "keysOf(",
				"keysOf is bound for LAYER-2 MetaJS only, not for grammar scripts - goja answers "+
					"ReferenceError. Carry an explicit key list.")
		case call && u.isLay2 && strings.HasPrefix(t.text, "js_") && !lay2Defined[t.text] && !localFns[t.text]:
			u.add(fs, "extern-from-lay2", t.line, t.text+"(",
				"an extern is callable from an EMITTED MODULE, never from layer 2: no layer-2 file "+
					"and no floor body defines "+t.text+", so this links, passes clang and dies "+
					"natively with \"ReferenceError: "+t.text+" is not defined\". Use the "+
					"host-builtin name.")
		case call && t.text == "println" && (u.isGram || u.isCore):
			if i+2 < len(ts) && ts[i+2].kind == tStr && strings.Contains(strings.ToLower(ts[i+2].text), "warning") {
				u.add(fs, "warn-on-stdout", t.line, "println(\"warning...",
					"println is stdout - the channel a compiler grammar emits its MODULE on. A warning "+
						"printed here ends up INSIDE the module. Use eprintln.")
			}
		}
	}
}

// argCount counts the top-level commas after the '(' at index open, +1.
func argCount(ts []token, open int) int {
	depth := 0
	args := 1
	for i := open; i < len(ts); i++ {
		if ts[i].kind != tPunct {
			continue
		}
		switch ts[i].text {
		case "(", "[", "{":
			depth++
		case ")", "]", "}":
			depth--
			if depth == 0 {
				return args
			}
		case ",":
			if depth == 1 {
				args++
			}
		}
	}
	return args
}

func checkForIn(u *unit, fs *[]finding) {
	ts := code(u.toks)
	for i := 0; i+3 < len(ts); i++ {
		if !(ts[i].kind == tIdent && ts[i].text == "for" && ts[i+1].kind == tPunct && ts[i+1].text == "(") {
			continue
		}
		depth := 0
		for j := i + 1; j < len(ts); j++ {
			if ts[j].kind == tPunct && ts[j].text == "(" {
				depth++
			} else if ts[j].kind == tPunct && ts[j].text == ")" {
				depth--
				if depth == 0 {
					break
				}
			} else if depth == 1 && ts[j].kind == tIdent && ts[j].text == "in" {
				u.add(fs, "for-in", ts[i].line, "for (... in ...)",
					"no for...in under -frozen. Carry an explicit key list alongside the object.")
				break
			}
		}
	}
}

func checkDupFunctions(u *unit, fs *[]finding) {
	ts := code(u.toks)
	seen := map[string]int{}
	depth := 0
	for i := 0; i < len(ts); i++ {
		if ts[i].kind == tPunct {
			if ts[i].text == "{" {
				depth++
			} else if ts[i].text == "}" {
				depth--
			}
			continue
		}
		if depth != 0 || ts[i].text != "function" || i+1 >= len(ts) || ts[i+1].kind != tIdent {
			continue
		}
		name := ts[i+1].text
		if prev, ok := seen[name]; ok {
			u.add(fs, "dup-function", ts[i+1].line, "function "+name,
				fmt.Sprintf("a second top-level `function %s` - the first is at line %d. In a grammar "+
					"script this SILENTLY REBINDS TAGS (no JS error, correct parse tree, wrong tag "+
					"running); in a layer-2 file it is `redefinition of global`.", name, prev+u.lineOff))
		} else {
			seen[name] = ts[i+1].line
		}
	}
}


// litType classifies a literal token so a re-typed slot can be spotted without a
// type checker. Only the three PRIMITIVE kinds are classified: those are the ones
// that produce "variable 'x' has type number and cannot hold a string", and they
// are the ones a literal-to-literal comparison can be sure about. Anything whose
// initialiser or assignment is a call, a name or an array/object literal returns
// "", and the rule stays quiet - a checker that guessed would fire on most of
// layer 2 and be switched off within the day.
func litType(ts []token, i int) string {
	if i >= len(ts) {
		return ""
	}
	// A literal only counts when it is the WHOLE right-hand side, i.e. the next
	// token ends the statement or starts a new one on a later line.
	end := i + 1
	if end < len(ts) && ts[end].line == ts[i].line &&
		!(ts[end].kind == tPunct && (ts[end].text == ";" || ts[end].text == "}" || ts[end].text == ")")) {
		return ""
	}
	switch ts[i].kind {
	case tNum:
		return "number"
	case tStr:
		return "string"
	case tIdent:
		if ts[i].text == "true" || ts[i].text == "false" {
			return "boolean"
		}
	}
	return ""
}

// checkTypedLocal: MetaJS IS TYPED, and a reassigned local keeps the type it was
// first given. A Go-style `x = normalize(x)` dies with "variable 'x' has type
// boolean and cannot hold a number" - and in the interpreter under -frozen ONLY,
// so --full reads FROZEN-DIFF while goja is green.
//
// Scope is deliberately narrow: a `var NAME = <primitive literal>` and a later
// `NAME = <primitive literal of another type>` inside the same top-level
// function. That is provable from the token stream and nothing else here is.
func checkTypedLocal(u *unit, fs *[]finding) {
	ts := code(u.toks)
	depth := 0
	declAt := map[string]int{} // name -> line, reset at each top-level function
	kind := map[string]string{}
	for i := 0; i < len(ts); i++ {
		if ts[i].kind == tPunct {
			if ts[i].text == "{" {
				depth++
			} else if ts[i].text == "}" {
				depth--
				if depth == 0 {
					declAt = map[string]int{}
					kind = map[string]string{}
				}
			}
			continue
		}
		if ts[i].text == "var" && i+2 < len(ts) && ts[i+1].kind == tIdent &&
			ts[i+2].kind == tPunct && ts[i+2].text == "=" {
			if t := litType(ts, i+3); t != "" {
				declAt[ts[i+1].text] = ts[i+1].line
				kind[ts[i+1].text] = t
			}
			continue
		}
		if ts[i].kind == tIdent && i+1 < len(ts) && ts[i+1].kind == tPunct && ts[i+1].text == "=" &&
			!(i+2 < len(ts) && ts[i+2].kind == tPunct && ts[i+2].text == "=") &&
			!(i > 0 && ts[i-1].kind == tPunct && (ts[i-1].text == "." || ts[i-1].text == "[")) {
			name := ts[i].text
			was, ok := kind[name]
			if !ok {
				continue
			}
			if now := litType(ts, i+2); now != "" && now != was {
				u.add(fs, "typed-local", ts[i].line, name+" = <"+now+">",
					fmt.Sprintf("MetaJS is TYPED: `%s` was pinned to %s at line %d and cannot hold a %s. "+
						"The error is \"variable '%s' has type %s and cannot hold a %s\" and it appears "+
						"under -frozen ONLY, so --full reads FROZEN-DIFF while goja is green. Use a "+
						"second name, or start the slot as anytype.",
						name, was, declAt[name]+u.lineOff, now, name, was, now))
				kind[name] = now
			}
		}
	}
}

// checkArgumentsInTry: `arguments` is not resolved from inside a try block.
// Capture it into a local before the try.
func checkArgumentsInTry(u *unit, fs *[]finding) {
	ts := code(u.toks)
	for i := 0; i < len(ts); i++ {
		if !(ts[i].kind == tIdent && ts[i].text == "try" &&
			i+1 < len(ts) && ts[i+1].kind == tPunct && ts[i+1].text == "{") {
			continue
		}
		depth := 0
		for j := i + 1; j < len(ts); j++ {
			if ts[j].kind == tPunct && ts[j].text == "{" {
				depth++
			} else if ts[j].kind == tPunct && ts[j].text == "}" {
				depth--
				if depth == 0 {
					break
				}
			} else if ts[j].kind == tIdent && ts[j].text == "arguments" {
				u.add(fs, "arguments-in-try", ts[j].line, "arguments",
					"`arguments` is not resolved from inside a try block. Capture it into a local "+
						"BEFORE the try and use that.")
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Reading the corpus.

// abnfScripts pulls the script text out of a grammar. Script bodies in this
// dialect live in `:script(...)`, `:startScript(...)` and `:endScript(...)`
// commands; taking the whole file would lint the ABNF rules as if they were
// JavaScript. The extraction is by paren matching from the command name, and the
// line offset is carried so every finding still reports the GRAMMAR's line.
func abnfScripts(path string) ([]unit, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	s := string(b)
	compiler := strings.Contains(filepath.Base(path), "-to-llvm-ir")
	var out []unit
	for _, kw := range []string{":startScript(", ":endScript(", ":script("} {
		at := 0
		for {
			k := strings.Index(s[at:], kw)
			if k < 0 {
				break
			}
			k += at
			open := k + len(kw) - 1
			depth := 0
			j := open
			for ; j < len(s); j++ {
				if s[j] == '(' {
					depth++
				} else if s[j] == ')' {
					depth--
					if depth == 0 {
						break
					}
				}
			}
			if j >= len(s) {
				break
			}
			body := s[open+1 : j]
			out = append(out, unit{
				file:    path,
				src:     body,
				toks:    lex(body),
				isGram:  true,
				isCompi: compiler,
				lineOff: strings.Count(s[:open+1], "\n"),
			})
			at = j
		}
	}
	return out, nil
}

func main() {
	root := flag.String("root", ".", "repository root")
	grammars := flag.Bool("grammars", false, "also lint the script blocks of every languages/*.abnf")
	only := flag.String("only", "", "comma-separated rule names to report")
	max := flag.Int("max", -1, "exit 1 above N findings (a ratchet); -1 disables")
	list := flag.Bool("list", false, "print the rules and what each one has cost, then stop")
	quiet := flag.Bool("q", false, "one line per finding, no explanation")
	// -selftest is the answer to "how do you know a rule still fires?". A linter
	// that has stopped detecting anything reports zero findings and looks exactly
	// like a clean tree, which is the same false-green shape tests/gates.sh's
	// recorded numbers exist to prevent. Every rule below has a POSITIVE case it
	// must report and a NEGATIVE case it must not.
	selftest := flag.Bool("selftest", false, "run every rule against a built-in positive and negative case; exit 1 if any rule stopped firing")
	flag.Parse()

	if *selftest {
		os.Exit(runSelfTest())
	}

	if *list {
		fmt.Print(rulesDoc)
		return
	}

	abs, err := filepath.Abs(*root)
	if err != nil {
		fatal("%v", err)
	}

	var units []unit
	lay2 := map[string]bool{}

	// Which js_* names EXIST: every `function js_*` across layer 2, plus every
	// js_* body in the C floor (a floor body is callable from layer 2 - that is
	// what an upcall is). Anything else called from layer 2 is an extern that
	// only an emitted module can reach.
	metajs, _ := filepath.Glob(filepath.Join(abs, "languages", "lib", "*.metajs"))
	for _, f := range metajs {
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		for _, m := range regexp.MustCompile(`function\s+(js_\w+)`).FindAllStringSubmatch(string(b), -1) {
			lay2[m[1]] = true
		}
	}
	if b, err := os.ReadFile(filepath.Join(abs, "languages", "lib", "runtime.c")); err == nil {
		for _, m := range regexp.MustCompile(`(?m)^[A-Za-z_][A-Za-z0-9_ *]*\b(js_\w+)\s*\(`).FindAllStringSubmatch(string(b), -1) {
			lay2[m[1]] = true
		}
	}

	for _, f := range metajs {
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		units = append(units, unit{file: rel(abs, f), src: string(b), toks: lex(string(b)), isLay2: true})
	}
	cores, _ := filepath.Glob(filepath.Join(abs, "languages", "lib", "*.js"))
	for _, f := range cores {
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		units = append(units, unit{
			file: rel(abs, f), src: string(b), toks: lex(string(b)),
			isCore: true, isCompi: strings.Contains(filepath.Base(f), "compile-core"),
		})
	}
	if *grammars {
		gs, _ := filepath.Glob(filepath.Join(abs, "languages", "*.abnf"))
		for _, g := range gs {
			us, err := abnfScripts(g)
			if err != nil {
				continue
			}
			for i := range us {
				us[i].file = rel(abs, us[i].file)
			}
			units = append(units, us...)
		}
	}

	// Every `function NAME` declared in each FILE, so a unit that defines its own
	// helper is not reported for using it. Collected across all of a grammar's
	// script blocks, since :startScript declares what a later :script uses.
	localByFile := map[string]map[string]bool{}
	for _, u := range units {
		m := localByFile[u.file]
		if m == nil {
			m = map[string]bool{}
			localByFile[u.file] = m
		}
		ts := code(u.toks)
		for i := 0; i+1 < len(ts); i++ {
			if ts[i].kind == tIdent && ts[i].text == "function" && ts[i+1].kind == tIdent {
				m[ts[i+1].text] = true
			}
			// `makeSeq = function (...)` - the assignment form the dialect notes
			// require for overriding a shared helper - declares a name too.
			if ts[i].kind == tIdent && ts[i+1].kind == tPunct && ts[i+1].text == "=" &&
				i+2 < len(ts) && ts[i+2].kind == tIdent && ts[i+2].text == "function" {
				m[ts[i].text] = true
			}
		}
	}
	// A shared core's declarations are visible to every grammar that includes it.
	// interp-core.js is where hasOwn lives, so an INTERPRETER grammar calling it
	// is correct - which is why the has-own rule is scoped to compiler halves in
	// the first place - but a compiler grammar that ALSO includes interp-core.js
	// (java does) is equally fine.
	for i := range units {
		u := &units[i]
		if !u.isGram {
			continue
		}
		for _, core := range []string{"interp-core.js", "compile-core.js", "regex.js"} {
			if !strings.Contains(u.src, core) {
				continue
			}
			for n := range localByFile[filepath.Join("languages", "lib", core)] {
				localByFile[u.file][n] = true
			}
		}
	}

	var fs []finding
	for i := range units {
		u := &units[i]
		checkExponent(u, &fs)
		checkStringEscapes(u, &fs)
		checkASIReturn(u, &fs)
		checkNamedCalls(u, &fs, lay2, localByFile[u.file])
		checkForIn(u, &fs)
		checkTypedLocal(u, &fs)
		checkArgumentsInTry(u, &fs)
		if !u.isGram {
			// A grammar's script blocks are separate units here, so a name
			// declared in :startScript and again in :endScript would not be seen
			// by a per-unit check anyway - and `mec <grammar> -verify` already
			// reports duplicate top-level functions for grammars. This rule is
			// for the files -verify never sees.
			checkDupFunctions(u, &fs)
		}
	}

	if *only != "" {
		want := map[string]bool{}
		for _, r := range strings.Split(*only, ",") {
			want[strings.TrimSpace(r)] = true
		}
		var keep []finding
		for _, f := range fs {
			if want[f.rule] {
				keep = append(keep, f)
			}
		}
		fs = keep
	}

	sort.Slice(fs, func(a, b int) bool {
		if fs[a].rule != fs[b].rule {
			return fs[a].rule < fs[b].rule
		}
		if fs[a].file != fs[b].file {
			return fs[a].file < fs[b].file
		}
		return fs[a].line < fs[b].line
	})

	nf := 0
	for _, u := range units {
		_ = u
		nf++
	}
	fmt.Printf("metajs-lint: %d unit(s) scanned (%d layer-2 files, %d shared cores%s)\n",
		nf, countIf(units, func(u unit) bool { return u.isLay2 }),
		countIf(units, func(u unit) bool { return u.isCore }),
		map[bool]string{true: fmt.Sprintf(", %d grammar script blocks", countIf(units, func(u unit) bool { return u.isGram })), false: ""}[*grammars])

	byRule := map[string]int{}
	last := ""
	for _, f := range fs {
		byRule[f.rule]++
		if f.rule != last {
			fmt.Println()
			last = f.rule
		}
		fmt.Printf("  %-18s %s:%d  %s\n", f.rule, f.file, f.line, f.text)
		if !*quiet {
			fmt.Printf("  %-18s   %s\n", "", wrap(f.note, 76, "                       "))
		}
	}

	fmt.Println()
	if len(fs) == 0 {
		fmt.Println("metajs-lint: 0 findings - the corpus is clean against every rule above.")
		fmt.Println("             That is a RESULT, not an absence of one: these constructs are")
		fmt.Println("             invisible to every other gate here (the \\f one is invisible to")
		fmt.Println("             all of them - it is wrong in the native binary only), so the")
		fmt.Println("             value of this tool is the NEXT one somebody writes.")
	} else {
		var rs []string
		for r := range byRule {
			rs = append(rs, fmt.Sprintf("%s %d", r, byRule[r]))
		}
		sort.Strings(rs)
		fmt.Printf("metajs-lint: %d finding(s): %s\n", len(fs), strings.Join(rs, ", "))
	}
	if *max >= 0 && len(fs) > *max {
		os.Exit(1)
	}
}

func countIf(us []unit, f func(unit) bool) int {
	n := 0
	for _, u := range us {
		if f(u) {
			n++
		}
	}
	return n
}

func rel(abs, p string) string {
	if r, err := filepath.Rel(abs, p); err == nil {
		return r
	}
	return p
}

func wrap(s string, w int, indent string) string {
	var out []string
	line := ""
	for _, word := range strings.Fields(s) {
		if len(line)+len(word)+1 > w && line != "" {
			out = append(out, line)
			line = word
		} else if line == "" {
			line = word
		} else {
			line += " " + word
		}
	}
	if line != "" {
		out = append(out, line)
	}
	return strings.Join(out, "\n"+indent)
}

func fatal(f string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, "metajs-lint: "+f+"\n", a...)
	os.Exit(2)
}

const rulesDoc = `metajs-lint rules - each is a documented dialect gap with a recorded cost.

  exponent-literal   1e308 lexes as the number 1 and the NAME e308. Under -frozen
                     the whole script is "cannot parse script", pointing nowhere
                     near the line. Two agents hit this independently in one day.
  string-escape      no \f and no \v; the BACKSLASH IS DROPPED, so ch == "\f"
                     compiles to ch == "f". Wrong in the NATIVE BINARY ONLY - the
                     matrix, --full and --cross are all structurally blind to it.
  asi-return         a value-less return is not terminated by the newline: the
                     next line becomes its operand and is evaluated. Silently
                     lost every store in one half for an entire agent-round.
  to-precision       no toPrecision.
  for-in             no for...in under -frozen.
  frozen-builtin     no splice, no lastIndexOf, and 2-argument indexOf IGNORES
                     the from-index - it does not raise, it returns the same
                     match every time, so a scan loop spins to the IR step cap.
                     (split, join and 'length =' all WORK - measured; the doc
                     that says otherwise is stale and they are not rules here.)
  has-own            hasOwn is in interp-core.js and NOT compile-core.js: an
                     emitter using it works in one half only.
  keys-of            keysOf is bound for layer-2 MetaJS only, never for grammar
                     scripts; goja answers ReferenceError.
  extern-from-lay2   a js_* extern is callable from an emitted module, never from
                     layer 2 - it links, passes clang, and dies natively.
  typed-local        MetaJS is typed: a local pinned to one primitive type by its
                     initialiser cannot later hold another. -frozen only.
  arguments-in-try   arguments is not resolved from inside a try block; capture
                     it into a local first.
  dup-function       a second top-level function of the same name silently
                     rebinds tags (grammar) or is a redefinition (layer 2).
  warn-on-stdout     println is the channel the MODULE goes out on; a warning
                     printed there ends up inside the module. Use eprintln.
`


// ---------------------------------------------------------------------------
// The self-test. Each case is (rule, source, mustFire).
//
// The NEGATIVE cases are not decoration: three of them are constructs this tool
// used to report wrongly and no longer does - a grammar that defines its own
// hasOwn, an escaped backslash before an f, and a `return` whose value is on its
// own line but whose next token is a closing brace.
type selfCase struct {
	rule string
	kind string // "lay2", "core", "compi-gram", "gram"
	src  string
	fire bool
}

var selfCases = []selfCase{
	{"exponent-literal", "lay2", `function f() { var x = 1e308 }`, true},
	{"exponent-literal", "lay2", `function f() { var x = 1000 }`, false},
	{"exponent-literal", "lay2", `function f() { var x = e308 }`, false},

	{"string-escape", "lay2", `function f() { return c == "\f" }`, true},
	{"string-escape", "lay2", `function f() { return c == "\v" }`, true},
	{"string-escape", "lay2", `function f() { return c == "\\f" }`, false},
	{"string-escape", "lay2", `function f() { return c == "\u000c" }`, false},

	{"asi-return", "lay2", "function f() {\n if (bad) return\n log = 1\n}", true},
	{"asi-return", "lay2", "function f() {\n if (bad) { return }\n log = 1\n}", false},
	{"asi-return", "lay2", "function f() {\n return\n}", false},
	{"asi-return", "lay2", "function f() {\n return v\n}", false},

	{"to-precision", "lay2", `function f() { return x.toPrecision(3) }`, true},
	{"to-precision", "lay2", `function f() { return x.toFixed(3) }`, false},

	{"for-in", "lay2", `function f() { for (var k in o) { n = n + 1 } }`, true},
	{"for-in", "lay2", `function f() { for (var i = 0; i < n; i = i + 1) { } }`, false},

	{"frozen-builtin", "lay2", `function f() { a.splice(1, 1) }`, true},
	{"frozen-builtin", "lay2", `function f() { return s.lastIndexOf("x") }`, true},
	{"frozen-builtin", "lay2", `function f() { return s.indexOf("x", i) }`, true},
	{"frozen-builtin", "lay2", `function f() { return s.indexOf("x") }`, false},
	{"frozen-builtin", "lay2", `function f() { a.length = 0 }`, false},
	{"frozen-builtin", "lay2", `function f() { return s.split(",").join("-") }`, false},

	{"has-own", "compi-gram", `function f() { return hasOwn(m, k) }`, true},
	{"has-own", "gram", `function f() { return hasOwn(m, k) }`, false},
	{"has-own", "compi-gram", `function hasOwn(o, n) { return true }
function f() { return hasOwn(m, k) }`, false},

	{"keys-of", "gram", `function f() { return keysOf(o) }`, true},
	{"keys-of", "lay2", `function f() { return keysOf(o) }`, false},

	{"extern-from-lay2", "lay2", `function f() { return js_no_such_extern(o) }`, true},
	{"extern-from-lay2", "lay2", `function js_mine(o) { return 1 }
function f() { return js_mine(o) }`, false},

	{"dup-function", "lay2", `function a() { }
function a() { }`, true},
	{"dup-function", "lay2", `function a() { }
function b() { var a = function () { } }`, false},

	{"typed-local", "lay2", `function f() {
var x = 0
x = "s"
}`, true},
	{"typed-local", "lay2", `function f() {
var x = 0
x = 1
}`, false},
	{"typed-local", "lay2", `function f() {
var x = 0
}
function g() {
var x = "s"
}`, false},

	{"arguments-in-try", "lay2", `function f() { try { var n = arguments.length } catch (e) { } }`, true},
	{"arguments-in-try", "lay2", `function f() { var a = arguments
try { var n = a.length } catch (e) { } }`, false},

	{"warn-on-stdout", "gram", `function f() { println("warning: x") }`, true},
	{"warn-on-stdout", "gram", `function f() { eprintln("warning: x") }`, false},
}

func runSelfTest() int {
	bad := 0
	for i, c := range selfCases {
		src := strings.ReplaceAll(c.src, "\\n", "\n")
		u := unit{file: fmt.Sprintf("selftest[%d]", i), src: src, toks: lex(src)}
		switch c.kind {
		case "lay2":
			u.isLay2 = true
		case "core":
			u.isCore = true
		case "compi-gram":
			u.isGram, u.isCompi = true, true
		case "gram":
			u.isGram = true
		}
		local := map[string]bool{}
		ts := code(u.toks)
		for j := 0; j+1 < len(ts); j++ {
			if ts[j].kind == tIdent && ts[j].text == "function" && ts[j+1].kind == tIdent {
				local[ts[j+1].text] = true
			}
		}
		lay2 := map[string]bool{}
		for n := range local {
			lay2[n] = true
		}
		var fs []finding
		checkExponent(&u, &fs)
		checkStringEscapes(&u, &fs)
		checkASIReturn(&u, &fs)
		checkNamedCalls(&u, &fs, lay2, local)
		checkForIn(&u, &fs)
		checkTypedLocal(&u, &fs)
		checkArgumentsInTry(&u, &fs)
		if !u.isGram {
			checkDupFunctions(&u, &fs)
		}
		got := false
		for _, f := range fs {
			if f.rule == c.rule {
				got = true
			}
		}
		if got != c.fire {
			bad++
			verb := "DID NOT FIRE but must"
			if !c.fire {
				verb = "FIRED but must not"
			}
			fmt.Printf("  FAIL  %-18s %s\n        source: %s\n", c.rule, verb, strings.ReplaceAll(src, "\n", " / "))
		}
	}
	rules := map[string]bool{}
	for _, c := range selfCases {
		rules[c.rule] = true
	}
	if bad == 0 {
		fmt.Printf("metajs-lint -selftest: %d cases over %d rules, all as expected\n", len(selfCases), len(rules))
		return 0
	}
	fmt.Printf("metajs-lint -selftest: %d of %d cases WRONG - a rule has stopped detecting its own construct\n", bad, len(selfCases))
	return 1
}
