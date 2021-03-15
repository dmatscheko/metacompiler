package ebnf

import (
	"fmt"
	"strings"

	"./r"
)

type Grammar = struct {
	Productions r.Rules
	Extras      map[string]r.Rule
}

// ----------------------------------------------------------------------------
// EBNF parser

type ebnfParser struct {
	src          []rune
	ch           rune
	sdx          int
	token        *r.Rule
	isSeq        bool // Only true if the String in ep.token is valid.
	traceEnabled bool
	ididx        []int
	idents       []string
	aGrammar     r.Rules
	productions  r.Rules
}

// TODO: DEDUPLICATE!!
func (ep *ebnfParser) skipSpaces() {
	for {
		if ep.sdx >= len(ep.src) {
			break
		}
		ep.ch = ep.src[ep.sdx]
		if strings.IndexRune(" \t\r\n", ep.ch) == -1 {
			break
		}
		ep.sdx++
	}
}

func (ep *ebnfParser) error(msg string, pos int) {
	if ep.token.Operator == r.Error { // Keep the first error
		return
	}
	ep.sdx = len(ep.src)
	ep.token = &r.Rule{Operator: r.Error, String: fmt.Sprintf("Error at %s: %s\n", LinePosFromStrPos(string(ep.src), pos), msg), Pos: pos}
}

func (ep *ebnfParser) getToken() {
	// Yields a single character token, one of {}()[]<>|=.;+-,
	// or {"TERMINAL",string} or {"IDENT", string} or -1.
	ep.skipSpaces()
	if ep.sdx >= len(ep.src) {
		ep.error("Error while parsing EBNF (beyond EOF)", ep.sdx)
		ep.isSeq = false
		return
	}
	tokstart := ep.sdx
	if strings.IndexRune("{}()[]<>|=.;+-,", ep.ch) >= 0 {
		ep.token = &r.Rule{Operator: r.Factor, Rune: ep.ch, Pos: tokstart}
		ep.isSeq = false
		ep.sdx++
	} else if ep.ch == '"' || ep.ch == '\'' {
		closech := ep.ch
		atEscapeCh := 0
		unescape := false
		for tokend := ep.sdx + 1; tokend < len(ep.src); tokend++ {
			if ep.src[tokend] == '\\' {
				atEscapeCh = (atEscapeCh + 1) % 2
				unescape = true
			}
			if ep.src[tokend] == closech && atEscapeCh == 0 {
				ep.sdx = tokend + 1

				tokenrunes := ep.src[tokstart+1 : tokend]
				if unescape {
					for pos := 0; pos+1 < len(tokenrunes); pos++ {
						if tokenrunes[pos] == '\\' {
							tokenrunes = append(tokenrunes[:pos], tokenrunes[pos+1:]...)
							switch tokenrunes[pos] {
							case 'r':
								tokenrunes[pos] = '\r'
							case 'n':
								tokenrunes[pos] = '\n'
							case 't':
								tokenrunes[pos] = '\t'
							}
						}
					}
				}

				ep.token = &r.Rule{Operator: r.Token, String: string(tokenrunes), Pos: tokstart}
				ep.isSeq = true
				return
			}
			if ep.src[tokend] != '\\' {
				atEscapeCh = 0
			}

		}
		ep.error("No closing quote", tokstart)
		ep.isSeq = false

	} else if ep.ch == '~' && len(ep.src) > ep.sdx+1 && ep.src[ep.sdx+1] == '~' {
		atEscapeCh := 0
		// unescape := false
		for tokend := ep.sdx + 2; tokend < len(ep.src); tokend++ {
			if ep.src[tokend] == '\\' {
				atEscapeCh = (atEscapeCh + 1) % 2
				// unescape = true
			}
			if ep.src[tokend] == '~' && atEscapeCh == 0 && len(ep.src) > tokend+1 && ep.src[tokend+1] == '~' {
				ep.sdx = tokend + 2

				tokenrunes := ep.src[tokstart+2 : tokend]
				// if unescape {
				// 	for pos := 0; pos+1 < len(tokenrunes); pos++ {
				// 		switch tokenrunes[pos+1] {
				// 		case '"', '\'', '\\', '~':
				// 			tokenrunes = append(tokenrunes[:pos], tokenrunes[pos+1:]...)
				// 			// case 'r':
				// 			// 	tokenrunes = append(tokenrunes[:pos], tokenrunes[pos+1:]...)
				// 			// 	tokenrunes[pos] = '\r'
				// 			// case 'n':
				// 			// 	tokenrunes = append(tokenrunes[:pos], tokenrunes[pos+1:]...)
				// 			// 	tokenrunes[pos] = '\n'
				// 			// case 't':
				// 			// 	tokenrunes = append(tokenrunes[:pos], tokenrunes[pos+1:]...)
				// 			// 	tokenrunes[pos] = '\t'
				// 		}
				// 	}
				// }

				ep.token = &r.Rule{Operator: r.Token, String: string(tokenrunes), Pos: tokstart}
				ep.isSeq = true
				return
			}
			if ep.src[tokend] != '\\' {
				atEscapeCh = 0
			}

		}
		ep.error("No closing quote", tokstart)
		ep.isSeq = false
	} else if (ep.ch >= 'a' && ep.ch <= 'z') || (ep.ch >= 'A' && ep.ch <= 'Z') {
		// To simplify things for the purposes of this task,
		// identifiers are strictly a-zA-Z only, not 1-9.
		for {
			ep.sdx++
			if ep.sdx >= len(ep.src) {
				break
			}
			ep.ch = ep.src[ep.sdx]
			if !((ep.ch >= 'a' && ep.ch <= 'z') || (ep.ch >= 'A' && ep.ch <= 'Z')) {
				break
			}
		}
		ep.token = &r.Rule{Operator: r.Ident, String: string(ep.src[tokstart:ep.sdx]), Pos: tokstart}
		ep.isSeq = true
	} else {
		ep.error(fmt.Sprintf("Invalid char '%c'", ep.ch), ep.sdx)
		ep.isSeq = false
	}
}

// also works like getToken()
func (ep *ebnfParser) matchToken(ch rune) {
	if ep.token.Operator == r.Factor && ep.token.Rune == ch {
		ep.getToken()
	} else {
		ep.error(fmt.Sprintf("Invalid char ('%c' expected, '%c' found)", ch, ep.src[ep.sdx]), ep.sdx)
		ep.isSeq = false
	}
}

func (ep *ebnfParser) addIdent(ident string) int {
	k := -1
	for i, id := range ep.idents {
		if id == ident {
			k = i
			break
		}
	}
	if k == -1 {
		ep.idents = append(ep.idents, ident)
		k = len(ep.idents) - 1
		ep.ididx = append(ep.ididx, -1)
	}
	return k
}

// also works like getToken(), but advances before that as much as it itself knows
func (ep *ebnfParser) factor() *r.Rule {
	if ep.traceEnabled {
		fmt.Printf("Fact(%d)  ", ep.sdx)
	}
	pos := ep.sdx
	var res *r.Rule

	valid := true
	if ep.token.Operator == r.Token {
		res = ep.token
		ep.getToken()
	} else if ep.token.Operator == r.Ident {
		ep.token.Int = ep.addIdent(ep.token.String) // Complete the IDENT command with the index of the target.
		res = ep.token
		ep.getToken()
	} else if ep.token.Operator == r.Factor {

		switch ep.token.Rune {
		case '[':
			ep.getToken()
			res = &r.Rule{Operator: r.Optional, Childs: &r.Rules{ep.expression()}, Pos: pos}
			ep.matchToken(']')
		case '(':
			ep.getToken()
			res = ep.expression()
			ep.matchToken(')')
		case '{':
			ep.getToken()
			res = &r.Rule{Operator: r.Repeat, Childs: &r.Rules{ep.expression()}, Pos: pos}
			ep.matchToken('}')
		case '+':
			res = &r.Rule{Operator: r.SkipSpace, Bool: true, Pos: pos}
			ep.getToken()
		case '-':
			res = &r.Rule{Operator: r.SkipSpace, Bool: false, Pos: pos}
			ep.getToken()
		case '<':
			res = ep.tag()
		case ',':
			ep.getToken()
			res = ep.token
			if ep.ch != '~' {
				valid = false
			}
			ep.getToken()
		default:
			valid = false
		}
	} else if ep.token.Operator == r.Tag {
		if ep.token.Rune == '~' {
			res = ep.token
			ep.getToken()
			ep.matchToken('~')
		} else if ep.token.Rune == ',' {
			ep.getToken()
		} else {
			valid = false
		}
	} else {
		valid = false
	}

	if !valid {
		ep.error(fmt.Sprintf("Invalid token in factor() function (%s)", PprintRuleOnly(ep.token)), ep.sdx)
	}

	return res
}

// also works like getToken(), but advances before that as much as it itself knows
// TODO: allow multiple strings/text, separated by ";"!
func (ep *ebnfParser) tag() *r.Rule {
	if ep.traceEnabled {
		fmt.Printf("Tag(%d)  ", ep.sdx)
	}
	pos := ep.sdx

	res := r.Rule{Operator: r.Tag, Pos: pos}
	for {
		ep.getToken()
		res.TagChilds = r.AppendPossibleSequence(res.TagChilds, ep.token)
		ep.getToken()
		if !(ep.token.Operator == r.Factor && ep.ch == ',') {
			break
		}
	}

	ep.matchToken('>')
	return &res
}

// also works like getToken(), but advances before that as much as it itself knows (= implements sequence)
func (ep *ebnfParser) term() *r.Rule {
	if ep.traceEnabled {
		fmt.Printf("Term(%d)  ", ep.sdx)
	}
	pos := ep.sdx

	firstFactor := ep.factor()
	if firstFactor.Operator == r.Tag {
		ep.error("TAG is invalid at this position", pos)
		return ep.token
	}

	res := r.Rules{firstFactor}

	for {
		if (ep.token.Operator == r.Factor && strings.IndexRune("})]>|.;", ep.token.Rune) >= 0) || ep.token.Operator == r.Error {
			break
		}

		newFactor := ep.factor()

		// Move everything that the TAG desribes (which is to the left) into into the TAG.
		// If newFactor is a TAG, merge this TAG with the last command in firstFactor.
		if newFactor.Operator == r.Tag {
			lastFactor := res[len(res)-1]
			res = res[:len(res)-1]                                                    // Remove the last factor from the result, because it will be appended again, but as a child of TAG.
			newFactor.Childs = r.AppendPossibleSequence(newFactor.Childs, lastFactor) // Fill the TAG.
			res = append(res, newFactor)
		} else {
			// If newFactor is not a TAG, only append newFactor to the firstFactor.
			res = *r.AppendPossibleSequence(&res, newFactor)
		}
	}

	if len(res) == 1 {
		return res[0]
	}
	return &r.Rule{Operator: r.Sequence, Childs: &res, Pos: pos}
}

// also works like getToken(), but advances before that as much as it itself knows
func (ep *ebnfParser) expression() *r.Rule {
	if ep.traceEnabled {
		fmt.Printf("Expr(%d)  ", ep.sdx)
	}
	pos := ep.sdx
	res := ep.term()

	if ep.token.Operator == r.Factor && ep.token.Rune == '|' {
		res = &r.Rule{Operator: r.Or, Childs: &r.Rules{res}, Pos: pos} // Override the result (a factor) with the command for OR.
		for ep.token.Operator == r.Factor && ep.token.Rune == '|' {
			ep.getToken()
			*res.Childs = append(*res.Childs, ep.term()) // Append the found alternative to the OR (It must not be ungrouped here, so use normal append() instead of r.AppendPossibleSequence()).
		}
		if len(*res.Childs) == 1 {
			return (*res.Childs)[0]
		}
	}
	return res
}

// also works like getToken(), but advances before that as much as it itself knows
func (ep *ebnfParser) production() *r.Rule {
	if ep.traceEnabled {
		fmt.Printf("Prod(%d)  ", ep.sdx)
	}
	pos := ep.sdx
	ep.getToken()

	if !(ep.token.Operator == r.Factor && ep.token.Rune == '}') {
		if ep.token.Operator == r.Error {
			ep.error("Invalid EBNF (missing closing '}')", pos)
			return ep.token
		}
		if ep.token.Operator != r.Ident {
			ep.error(fmt.Sprintf("Ident expected but got %s", PprintRuleOnly(ep.token)), pos)
			return ep.token
		}

		ident := ep.token.String
		idx := ep.addIdent(ident)
		ep.getToken()

		var tag *r.Rule
		foundTag := false
		if ep.token.Operator == r.Factor && ep.token.Rune == '<' {
			tag = ep.tag()
			foundTag = true
		}

		ep.matchToken('=')
		if ep.token.Operator == r.Error {
			return ep.token
		}
		if foundTag {
			tag.Childs = r.AppendPossibleSequence(tag.Childs, ep.expression()) // Fill the TAG.
			ep.productions = append(ep.productions, &r.Rule{Operator: r.Production, String: ident, Int: idx, Childs: &r.Rules{tag}, Pos: pos})
		} else {
			ep.productions = append(ep.productions, &r.Rule{Operator: r.Production, String: ident, Int: idx, Childs: &r.Rules{ep.expression()}, Pos: pos})
		}
		ep.ididx[idx] = len(ep.productions) - 1
	}

	return ep.token
}

func (ep *ebnfParser) parse(srcEbnf string) {
	ep.src = []rune(srcEbnf)
	ep.sdx = 0
	ep.aGrammar = r.Rules{}
	ep.productions = ep.productions[:0]
	ep.ididx = ep.ididx[:0]
	ep.idents = ep.idents[:0]

	pos := ep.sdx
	ep.getToken()

	// Title
	if ep.token.Operator == r.Token {
		ep.aGrammar = append(ep.aGrammar, ep.token)
		pos = ep.sdx
		ep.getToken()
	}

	// Prolog
	if ep.token.Operator == r.Factor && ep.token.Rune == '<' {
		ep.aGrammar = append(ep.aGrammar, ep.tag())
		pos = ep.sdx
		// ep.getToken()
	}

	// Main
	if !(ep.token.Operator == r.Factor && ep.token.Rune == '{') {
		ep.error("Invalid EBNF (missing opening '{')", pos)
		return
	}
	for {
		// ep.token = ep.production()
		ep.production()
		if (ep.token.Operator == r.Factor && ep.token.Rune == '}') || ep.token.Operator == r.Error {
			break
		}
	}
	ep.aGrammar = append(ep.aGrammar, &r.Rule{Operator: r.Sequence, Childs: &ep.productions, Pos: pos})
	pos = ep.sdx
	ep.getToken()

	// Epilog
	if ep.token.Operator == r.Factor && ep.token.Rune == '<' {
		ep.aGrammar = append(ep.aGrammar, ep.tag())
		pos = ep.sdx
		// ep.getToken()
	}

	// Entry point
	if ep.token.Operator == r.Ident {
		ep.aGrammar = append(ep.aGrammar, ep.token)
		pos = ep.sdx
		ep.getToken()
	} else {
		ep.error("Invalid EBNF (missing entry point)", pos)
		return
	}

	// Comment
	if ep.token.Operator == r.Token {
		ep.aGrammar = append(ep.aGrammar, ep.token)
		pos = ep.sdx
		ep.getToken()
	}

	// End of parsing
	if ep.token.Operator != r.Error {
		ep.error("Invalid EBNF (Not at EOF)", pos)
		return
	}

	ep.token = &r.Rule{Operator: r.Success}
	ep.resolveNameIdx(GetProductions(&ep.aGrammar))
	ep.verifyGrammar()

	// Also resolve the index of the start rule.
	tmpStart := GetStartRule(&ep.aGrammar)
	tmpStart.Int = ep.ididx[tmpStart.Int]
}

func (ep *ebnfParser) resolveNameIdx(productions *r.Rules) {
	for i := range *productions {
		rule := (*productions)[i]
		if rule.Childs != nil && len(*rule.Childs) > 0 {
			ep.resolveNameIdx(rule.Childs)
		}
		if rule.Operator == r.Production || rule.Operator == r.Ident {
			rule.Int = ep.ididx[rule.Int]
		}
	}
}

var ep ebnfParser

func ParseAEBNF(srcEbnf string, traceEnabled bool) (g *r.Rules, e error) {
	defer func() {
		if err := recover(); err != nil {
			if ep.token.Operator == r.Error && ep.token.String != "" {
				e = fmt.Errorf("%s", ep.token.String)
			} else {
				e = fmt.Errorf("%s", err)
			}
		}
	}()

	// var ep ebnfParser
	ep.traceEnabled = traceEnabled
	ep.parse(srcEbnf)

	var err error
	if ep.token.Operator != r.Success {
		if ep.token.String != "" {
			err = fmt.Errorf("%s", ep.token.String)
		} else {
			err = fmt.Errorf("Error while parsing")
		}
	}

	return &ep.aGrammar, err
}
