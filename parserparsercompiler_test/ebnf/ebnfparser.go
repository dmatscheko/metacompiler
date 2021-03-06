package ebnf

import (
	"fmt"
	"strings"

	"./r"
)

type Grammar = struct {
	Productions []r.Rule
	start       int
	Extras      map[string]r.Rule

	// TODO: var loopStop!! loop prevention (also for grammarparser): whenever the parser steps deeper, get the entry of map[the current position] in a map: the entry is nil or a map of already tried rules. if a rule has been tried already, return false and don't try again
	// TODO: also make it configurable if it is used (only if it has performance impact)
	// TODO: implement hasBeenTried() for this:
	// loopStop map[int]map[r.OperatorID]bool
	// better: if possible something like this:
	// loopStop map[int][r.OperatorID]bool
}

// ----------------------------------------------------------------------------
// EBNF parser

type ebnfParser struct {
	src     []rune
	ch      rune
	sdx     int
	token   r.Rule
	isSeq   bool // Only true if the String in ep.token is valid.
	err     bool
	ididx   []int
	idents  []string
	grammar Grammar
}

// // call this whenever pos is set before for a new rule is tested
// func hasBeenTried(rule *r.Rule, pos int) bool {
// 	// if len(rule.Childs) == 0 { // this is probably only possible in grammarparser
// 	// 	return true
// 	// }
// 	// ...
// }

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

func (ep *ebnfParser) invalid(msg string, pos int) r.Rule {
	ep.err = true
	fmt.Printf("Error at position %d: %s\n", pos, msg)
	ep.sdx = len(ep.src) // set to eof
	// ep.token = r.Sequence{Operator: r.Invalid, String: msg} // TODO: maybe store the message later
	return r.Rule{Operator: r.Invalid, String: msg, Pos: pos}
}

func (ep *ebnfParser) getToken() {
	// Yields a single character token, one of {}()[]<>|=.;+-,
	// or {"TERMINAL",string} or {"IDENT", string} or -1.
	ep.skipSpaces()
	if ep.sdx >= len(ep.src) {
		ep.token = r.Rule{Operator: r.Invalid, String: "Error while parsing EBNF (beyond EOF)", Pos: ep.sdx}
		ep.isSeq = false
		return
	}
	tokstart := ep.sdx
	if strings.IndexRune("{}()[]<>|=.;+-,", ep.ch) >= 0 {
		ep.token = r.Rule{Operator: r.Factor, Rune: ep.ch, Pos: tokstart}
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
					for pos := 0; pos < len(tokenrunes); pos++ {
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
				// fmt.Printf(">>> %s\n", string(tokenrunes))

				ep.token = r.Rule{Operator: r.Terminal, String: string(tokenrunes), Pos: tokstart}
				ep.isSeq = true
				return
			}
			if ep.src[tokend] != '\\' {
				atEscapeCh = 0
			}

		}
		ep.token = ep.invalid("no closing quote", tokstart)
		ep.isSeq = false

	} else if ep.ch == '~' && len(ep.src) > ep.sdx+1 && ep.src[ep.sdx+1] == '~' {
		atEscapeCh := 0
		unescape := false
		for tokend := ep.sdx + 2; tokend < len(ep.src); tokend++ {
			if ep.src[tokend] == '\\' {
				atEscapeCh = (atEscapeCh + 1) % 2
				unescape = true
			}
			if ep.src[tokend] == '~' && atEscapeCh == 0 && len(ep.src) > tokend+1 && ep.src[tokend+1] == '~' {
				ep.sdx = tokend + 2

				tokenrunes := ep.src[tokstart+2 : tokend]
				if unescape {
					for pos := 0; pos < len(tokenrunes); pos++ {
						if tokenrunes[pos] == '\\' {
							tokenrunes = append(tokenrunes[:pos], tokenrunes[pos+1:]...)
							switch tokenrunes[pos] {
							case 'r':
								tokenrunes[pos] = '\r'
							case 'n':
								tokenrunes[pos] = '\n'
							case 't':
								tokenrunes[pos] = '\t'
							case '~':
								tokenrunes[pos] = '~'
							}
						}
					}
				}
				// fmt.Printf(">>> %s\n", string(tokenrunes))

				ep.token = r.Rule{Operator: r.Terminal, String: string(tokenrunes), Pos: tokstart}
				ep.isSeq = true
				return
			}
			if ep.src[tokend] != '\\' {
				atEscapeCh = 0
			}

		}
		ep.token = ep.invalid("no closing quote", tokstart)
		ep.isSeq = false
	} else if ep.ch >= 'a' && ep.ch <= 'z' {
		// To simplify things for the purposes of this task,
		// identifiers are strictly a-z only, not A-Z or 1-9.
		for {
			ep.sdx++
			if ep.sdx >= len(ep.src) {
				break
			}
			ep.ch = ep.src[ep.sdx]
			if ep.ch < 'a' || ep.ch > 'z' {
				break
			}
		}
		ep.token = r.Rule{Operator: r.Ident, String: string(ep.src[tokstart:ep.sdx]), Pos: tokstart}
		ep.isSeq = true
	} else {
		ep.token = ep.invalid(fmt.Sprintf("Invalid char '%c'", ep.ch), ep.sdx)
		ep.isSeq = false
	}
}

// also works like getToken()
func (ep *ebnfParser) matchToken(ch rune) {
	if ep.token.Operator == r.Factor && ep.token.Rune == ch {
		ep.getToken()
	} else {
		ep.token = ep.invalid(fmt.Sprintf("Invalid char ('%c' expected, '%c' found)", ch, ep.src[ep.sdx]), ep.sdx)
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
func (ep *ebnfParser) factor() r.Rule {
	pos := ep.sdx
	var res r.Rule

	valid := true
	if ep.token.Operator == r.Terminal {
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
			res = r.Rule{Operator: r.Optional, Childs: []r.Rule{ep.expression()}, Pos: pos}
			ep.matchToken(']')
		case '(':
			ep.getToken()
			res = ep.expression()
			ep.matchToken(')')
		case '{':
			ep.getToken()
			res = r.Rule{Operator: r.Repeat, Childs: []r.Rule{ep.expression()}, Pos: pos}
			ep.matchToken('}')
		case '+':
			res = r.Rule{Operator: r.SkipSpaces, Bool: true, Pos: pos}
			ep.getToken()
		case '-':
			res = r.Rule{Operator: r.SkipSpaces, Bool: false, Pos: pos}
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
		panic(fmt.Sprintf("invalid token in factor() function (%#v)", ep.token))
	}

	// if res.Operator == r.Basic && len(res.Childs) == 1 {
	// 	return res.Childs[0]
	// }
	return res
}

// also works like getToken(), but advances before that as much as it itself knows
// TODO: allow multiple strings/text, separated by ";"!
func (ep *ebnfParser) tag() r.Rule {
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
	return res
}

// also works like getToken(), but advances before that as much as it itself knows (= implements sequence)
func (ep *ebnfParser) term() r.Rule {
	pos := ep.sdx

	firstFactor := ep.factor()
	if firstFactor.Operator == r.Tag {
		return ep.invalid("TAG is invalid at this position", pos)
	}

	res := []r.Rule{firstFactor}

	for {
		if (ep.token.Operator == r.Factor && strings.IndexRune("})]>|.;", ep.token.Rune) >= 0) || ep.token.Operator == r.Invalid {
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
			res = r.AppendPossibleSequence(res, newFactor)
		}
	}

	if len(res) == 1 {
		return res[0]
	}
	return r.Rule{Operator: r.Sequence, Childs: res, Pos: pos}
}

// also works like getToken(), but advances before that as much as it itself knows
func (ep *ebnfParser) expression() r.Rule {
	pos := ep.sdx
	res := ep.term()

	if ep.token.Operator == r.Factor && ep.token.Rune == '|' {
		res = r.Rule{Operator: r.Or, Childs: []r.Rule{res}, Pos: pos} // Override the result (a factor) with the command for OR.
		for ep.token.Operator == r.Factor && ep.token.Rune == '|' {
			ep.getToken()
			res.Childs = append(res.Childs, ep.term()) // Append the found alternative to the OR (It must not be ungrouped here, so use normal append() instead of r.AppendPossibleSequence()).
		}
		if len(res.Childs) == 1 {
			return res.Childs[0]
		}
	}
	return res
}

// also works like getToken(), but advances before that as much as it itself knows
func (ep *ebnfParser) production() r.Rule {
	pos := ep.sdx
	// Returns a token or r.Invalid; the real result is left in 'productions' etc, ...
	ep.getToken()

	if !(ep.token.Operator == r.Factor && ep.token.Rune == '}') {
		if ep.token.Operator == r.Invalid {
			return ep.invalid("Invalid EBNF (missing closing })", pos)
		}
		if ep.token.Operator != r.Ident {
			return r.Rule{Operator: r.Invalid, String: fmt.Sprintf("Ident expected but got %s", ep.token.Operator), Pos: pos}
		}

		ident := ep.token.String
		idx := ep.addIdent(ident)
		ep.getToken()

		var tag r.Rule
		foundTag := false
		if ep.token.Operator == r.Factor && ep.token.Rune == '<' {
			tag = ep.tag()
			foundTag = true
		}

		ep.matchToken('=')
		if ep.token.Operator == r.Invalid {
			return ep.token
		}
		if foundTag {
			tag.Childs = r.AppendPossibleSequence(tag.Childs, ep.expression()) // Fill the TAG.
			ep.grammar.Productions = append(ep.grammar.Productions, r.Rule{Operator: r.Production, String: ident, Int: idx, Childs: []r.Rule{tag}, Pos: pos})
		} else {
			ep.grammar.Productions = append(ep.grammar.Productions, r.Rule{Operator: r.Production, String: ident, Int: idx, Childs: []r.Rule{ep.expression()}, Pos: pos})
		}
		ep.ididx[idx] = len(ep.grammar.Productions) - 1
	}

	return ep.token
}

// ep.err == false, if the parsing went OK
func (ep *ebnfParser) parse(srcEbnf string) {
	ep.err = false
	ep.src = []rune(srcEbnf)
	ep.sdx = 0
	ep.grammar.Extras = make(map[string]r.Rule)
	ep.grammar.Productions = ep.grammar.Productions[:0]
	ep.ididx = ep.ididx[:0]
	ep.idents = ep.idents[:0]

	pos := ep.sdx
	ep.getToken()

	// Title
	if ep.token.Operator == r.Terminal {
		ep.grammar.Extras["title"] = ep.token
		pos = ep.sdx
		ep.getToken()
	}

	// Prolog
	if ep.token.Operator == r.Factor && ep.token.Rune == '<' {
		ep.grammar.Extras["prolog.code"] = ep.tag()
		pos = ep.sdx
		// ep.getToken()
	}

	// Main
	if !(ep.token.Operator == r.Factor && ep.token.Rune == '{') {
		ep.invalid("Invalid EBNF (missing opening {)", pos)
		return
	}
	for {
		ep.token = ep.production()
		if (ep.token.Operator == r.Factor && ep.token.Rune == '}') || ep.token.Operator == r.Invalid {
			break
		}
	}
	pos = ep.sdx
	ep.getToken()

	// Epilog
	if ep.token.Operator == r.Factor && ep.token.Rune == '<' {
		ep.grammar.Extras["epilog.code"] = ep.tag()
		pos = ep.sdx
		// ep.getToken()
	}

	// Entry point
	if ep.token.Operator == r.Ident {
		start := ep.token
		start.Int = ep.addIdent(ep.token.String)
		ep.grammar.Extras["start"] = start
		pos = ep.sdx
		ep.getToken()
	} else {
		ep.invalid("Invalid EBNF (missing entry point)", pos)
	}

	// Comment
	if ep.token.Operator == r.Terminal {
		ep.grammar.Extras["comment"] = ep.token
		pos = ep.sdx
		ep.getToken()
	}

	// End of parsing
	if ep.token.Operator != r.Invalid {
		ep.invalid("Invalid EBNF (missing EOF?)", pos)
		return
	}
	if ep.err {
		return
	}

	ep.token = r.Rule{}
	ep.verifyGrammar()
	ep.resolveIdIdx(&ep.grammar.Productions)

	// Also resolve the index of the start rule.
	tmpStart := ep.grammar.Extras["start"]
	tmpStart.Int = ep.ididx[tmpStart.Int]
	ep.grammar.Extras["start"] = tmpStart
}

func (ep *ebnfParser) resolveIdIdx(productions *[]r.Rule) {
	for i := range *productions {
		rule := &(*productions)[i]
		if len(rule.Childs) > 0 {
			ep.resolveIdIdx(&rule.Childs)
		}
		if rule.Operator == r.Production || rule.Operator == r.Ident {
			rule.Int = ep.ididx[rule.Int]
		}
	}
}

func ParseEBNF(srcEbnf string) (g Grammar, e error) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("%s\n  ==> Fail\n", err)
			e = fmt.Errorf("Fail")
		}
	}()

	var ep ebnfParser
	ep.parse(srcEbnf)

	var err error = nil
	if ep.err {
		if ep.token.String != "" {
			err = fmt.Errorf("%s", ep.token.String)
		} else {
			err = fmt.Errorf("Error while parsing")
		}
	}

	return ep.grammar, err
}
