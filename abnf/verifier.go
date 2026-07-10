package abnf

// Grammar verification: static name-consistency checks on an a-grammar, without
// running it. The modern successor of the 2021 _unused/ebnf/verifyer.go (whose
// code was tied to the old parser); the two checks it performed carry over:
//
//   - undefined: an Identifier names a production that no production defines
//     (a typo, or a fragment that was never :include()d). This is an error -
//     the reference resolves to the -1 marker and fails at parse time.
//   - unreachable: a production is defined but can never be reached from the
//     start rule (following identifiers, including the ones in command
//     parameters like :whitespace(Whitespace)). A warning - dead grammar.
//
// The check walks the a-grammar purely by NAME, so it needs no reference
// resolution pass and never mutates a rule. Run it on a FULLY ASSEMBLED grammar
// (after a parse merged any :include() fragments in place); on the raw output of
// CompileASG the fragment productions are not present yet.

import (
	"sort"
	"strings"
	"unicode/utf8"

	"14.gy/mec/abnf/r"
)

// VerifyIssue is one problem found by Verify.
type VerifyIssue struct {
	Kind   string // "undefined" / "badrange" (errors) or "unreachable" (warning).
	Name   string // The offending identifier / production name, or bad range bound.
	Line   int    // 1-based line in the grammar source (0 if unknown).
	Detail string // Extra context (the range unit "rune"/"byte" for badrange).
}

// IsError reports whether the issue breaks the grammar (vs. a mere warning).
func (vi VerifyIssue) IsError() bool { return vi.Kind == "undefined" || vi.Kind == "badrange" }

// Message renders the issue as a human sentence (without the location).
func (vi VerifyIssue) Message() string {
	switch vi.Kind {
	case "undefined":
		return "undefined name '" + vi.Name + "' is used but no production defines it (typo, or a missing :include())"
	case "unreachable":
		return "production '" + vi.Name + "' is defined but never reached from the start rule (dead)"
	case "badrange":
		return "malformed range: the bound " + quote(vi.Name) + " must be exactly one " + vi.Detail
	}
	return vi.Kind + " " + vi.Name
}

// quote renders a range bound for a message, escaping the unprintable bytes a
// range endpoint may hold (e.g. a lone 0x00 or 0xff).
func quote(s string) string {
	var b strings.Builder
	b.WriteByte('\'')
	for _, c := range []byte(s) {
		if c >= 0x20 && c < 0x7f {
			b.WriteByte(c)
		} else {
			b.WriteString("\\x")
			const hex = "0123456789abcdef"
			b.WriteByte(hex[c>>4])
			b.WriteByte(hex[c&0xf])
		}
	}
	b.WriteByte('\'')
	return b.String()
}

// HasInclude reports whether a grammar has any :include() command - i.e. whether
// it must be assembled (parsed) before Verify sees its complete production set.
func HasInclude(aGrammar *r.Rules) bool {
	if aGrammar == nil {
		return false
	}
	for _, rule := range *aGrammar {
		if rule.Operator == r.Command && rule.String == "include" {
			return true
		}
	}
	return false
}

// ProductionNames returns the set of production names defined at the top level
// of an a-grammar. Captured BEFORE a grammar is assembled, it is the set of a
// grammar's OWN productions (the ones written in its file), as opposed to those
// merged in later from :include() fragments - which Verify uses to keep
// "unreachable" reports to the grammar's own code.
func ProductionNames(aGrammar *r.Rules) map[string]bool {
	names := map[string]bool{}
	if aGrammar == nil {
		return names
	}
	for _, rule := range *aGrammar {
		if rule.Operator == r.Production {
			names[rule.String] = true
		}
	}
	return names
}

// Verify checks an (assembled) a-grammar and returns the issues found, sorted by
// source line. source is the grammar text the a-grammar was compiled from,
// scanned to locate each name's line; pass "" to omit line numbers.
//
// ownNames (from ProductionNames before assembly) limits the "unreachable"
// report to the grammar's own productions: a shared :include() fragment
// intentionally defines more than any one language uses, so its extra
// productions are not dead code of the includer. Pass nil to report every
// unreachable production.
func Verify(aGrammar *r.Rules, source string, ownNames map[string]bool) []VerifyIssue {
	if aGrammar == nil {
		return nil
	}

	// The productions defined at the top level, by name.
	defined := map[string]*r.Rule{}
	for _, rule := range *aGrammar {
		if rule.Operator == r.Production {
			defined[rule.String] = rule
		}
	}

	// Rule positions are byte offsets into the grammar source (since abnf-of-abnf
	// stamps up.pos onto the identifiers and productions it builds). A rule that
	// came from an :include()d fragment carries no meaningful offset here, so the
	// line falls back to a by-name scan of the source.
	lineFor := func(pos int, fallback func() int) int {
		if l := posLine(source, pos); l != 0 {
			return l
		}
		return fallback()
	}
	var issues []VerifyIssue

	// Checks 1 & 2 walk every rule once:
	//   - undefined: an Identifier whose name has no production. One issue per
	//     distinct name (a name typoed ten times is one problem); the line
	//     points at the first use.
	//   - badrange: a rune/byte range ("a"..."z" / "\x00"..b"\xff") whose bound
	//     is not exactly one rune/byte - the parser would silently use only its
	//     first character, so "ab"..."z" reads as "a"..."z".
	seenUndef := map[string]bool{}
	var scan func(rules *r.Rules)
	scan = func(rules *r.Rules) {
		if rules == nil {
			return
		}
		for _, rule := range *rules {
			switch rule.Operator {
			case r.Identifier:
				if _, ok := defined[rule.String]; !ok && !seenUndef[rule.String] {
					seenUndef[rule.String] = true
					name := rule.String
					line := lineFor(rule.Pos, func() int { return firstUseLine(source, name) })
					issues = append(issues, VerifyIssue{Kind: "undefined", Name: name, Line: line})
				}
			case r.Range:
				unit := "rune"
				if rule.Int == r.RangeTypeByte {
					unit = "byte"
				}
				if rule.CodeChilds != nil {
					for _, bound := range *rule.CodeChilds {
						if !validRangeBound(bound.String, rule.Int) {
							issues = append(issues, VerifyIssue{Kind: "badrange", Name: bound.String, Line: posLine(source, rule.Pos), Detail: unit})
						}
					}
				}
			}
			scan(rule.Childs)
			scan(rule.CodeChilds)
		}
	}
	scan(aGrammar)

	// Check 2 - unreachable: reachability from the start rule plus every name a
	// top-level command references (e.g. :whitespace(Whitespace) roots the whole
	// whitespace/comment sub-grammar even if no production names it inline).
	// Only meaningful when the grammar parses: a startScript-driven grammar (no
	// :startRule(), it builds everything in JS) has no parse-time reachability,
	// so this check is skipped for it.
	start := r.GetStartRule(aGrammar)
	if start != nil {
		roots := []string{start.String}
		for _, rule := range *aGrammar {
			if rule.Operator == r.Command {
				collectIdentNames(rule.CodeChilds, func(n string) { roots = append(roots, n) })
			}
		}
		reached := map[string]bool{}
		var visit func(name string)
		visit = func(name string) {
			if reached[name] {
				return
			}
			reached[name] = true
			if prod, ok := defined[name]; ok {
				collectIdentNames(prod.Childs, visit)
				collectIdentNames(prod.CodeChilds, visit)
			}
		}
		for _, root := range roots {
			visit(root)
		}
		for name, rule := range defined {
			if reached[name] {
				continue
			}
			if ownNames != nil && !ownNames[name] {
				continue // Defined by an :include() fragment, not the grammar's own code.
			}
			nm := name
			line := lineFor(rule.Pos, func() int { return definitionLine(source, nm) })
			issues = append(issues, VerifyIssue{Kind: "unreachable", Name: name, Line: line})
		}
	}

	sort.SliceStable(issues, func(i, j int) bool {
		if issues[i].Line != issues[j].Line {
			return issues[i].Line < issues[j].Line
		}
		return issues[i].Name < issues[j].Name
	})
	return issues
}

// collectIdentNames calls f for every Identifier name anywhere in the tree.
func collectIdentNames(rules *r.Rules, f func(string)) {
	if rules == nil {
		return
	}
	for _, rule := range *rules {
		if rule.Operator == r.Identifier {
			f(rule.String)
		}
		collectIdentNames(rule.Childs, f)
		collectIdentNames(rule.CodeChilds, f)
	}
}

// posLine maps a byte offset in source to a 1-based line, or 0 when the offset
// is unusable (<=0, or past the end - e.g. a rule that came from another file).
func posLine(source string, pos int) int {
	if pos <= 0 || pos > len(source) {
		return 0
	}
	line := 1
	for i := 0; i < pos; i++ {
		if source[i] == '\n' {
			line++
		}
	}
	return line
}

// validRangeBound reports whether s is a well-formed bound of a range of the
// given RangeType: exactly one byte for a byte range, exactly one rune for a
// rune range (an empty or multi-character bound is malformed).
func validRangeBound(s string, rangeType int) bool {
	if rangeType == r.RangeTypeByte {
		return len(s) == 1
	}
	return utf8.RuneCountInString(s) == 1
}

// isIdentByte reports whether b can be part of a grammar name.
func isIdentByte(b byte) bool {
	return b == '_' || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

// definitionLine finds the 1-based line where a production is defined: the first
// line whose leading token (after indentation) is exactly name. Productions in
// this dialect start with their name (Name = ... or Name <~~ tag ~~> = ...).
// Returns 0 if not found (e.g. the production came from an :include()d file).
func definitionLine(source, name string) int {
	for i, line := range strings.Split(source, "\n") {
		s := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(s, name) {
			rest := s[len(name):]
			if rest == "" || !isIdentByte(rest[0]) {
				return i + 1
			}
		}
	}
	return 0
}

// firstUseLine finds the 1-based line of the first whole-word occurrence of
// name (used to point an undefined-name error at its use). 0 if not found.
func firstUseLine(source, name string) int {
	for i, line := range strings.Split(source, "\n") {
		for c := 0; c+len(name) <= len(line); c++ {
			if line[c:c+len(name)] != name {
				continue
			}
			before := c == 0 || !isIdentByte(line[c-1])
			after := c+len(name) == len(line) || !isIdentByte(line[c+len(name)])
			if before && after {
				return i + 1
			}
		}
	}
	return 0
}
