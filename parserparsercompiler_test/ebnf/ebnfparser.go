package ebnf

import (
	"fmt"
	"strings"

	"./seq"
)

type Grammar = struct {
	Productions []seq.Sequence
	start       int
	Extras      map[string]seq.Sequence
}

// ----------------------------------------------------------------------------
// EBNF parser

type ebnfParser struct {
	src     []rune
	ch      rune
	sdx     int
	token   seq.Sequence
	isSeq   bool // Only true if the String in ep.token is valid.
	err     bool
	ididx   []int
	idents  []string
	grammar Grammar
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

func (ep *ebnfParser) invalid(msg string, pos int) seq.Sequence {
	ep.err = true
	fmt.Printf("Error at position %d: %s\n", pos, msg)
	ep.sdx = len(ep.src) // set to eof
	// ep.token = seq.Sequence{Operator: seq.Invalid, String: msg} // TODO: maybe store the message later
	return seq.Sequence{Operator: seq.Invalid, String: msg, Pos: pos}
}

func (ep *ebnfParser) getToken() {
	// Yields a single character token, one of {}()[]|=.;
	// or {"TERMINAL",string} or {"IDENT", string} or -1.
	ep.skipSpaces()
	if ep.sdx >= len(ep.src) {
		ep.token = seq.Sequence{Operator: seq.Invalid, String: "Error while parsing EBNF (beyond EOF)", Pos: ep.sdx}
		ep.isSeq = false
		return
	}
	tokstart := ep.sdx
	if strings.IndexRune("{}()[]<>|=.;+-,", ep.ch) >= 0 {
		ep.token = seq.Sequence{Operator: seq.Factor, Rune: ep.ch, Pos: tokstart}
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

				ep.token = seq.Sequence{Operator: seq.Terminal, String: string(tokenrunes), Pos: tokstart}
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

				ep.token = seq.Sequence{Operator: seq.Terminal, String: string(tokenrunes), Pos: tokstart}
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
		ep.token = seq.Sequence{Operator: seq.Ident, String: string(ep.src[tokstart:ep.sdx]), Pos: tokstart}
		ep.isSeq = true
	} else {
		ep.token = ep.invalid(fmt.Sprintf("Invalid char '%c'", ep.ch), ep.sdx)
		ep.isSeq = false
	}
}

// also works like getToken()
func (ep *ebnfParser) matchToken(ch rune) {
	if ep.token.Operator == seq.Factor && ep.token.Rune == ch {
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
func (ep *ebnfParser) factor() seq.Sequence {
	pos := ep.sdx
	var res seq.Sequence

	valid := true
	if ep.token.Operator == seq.Terminal {
		res = ep.token
		ep.getToken()
	} else if ep.token.Operator == seq.Ident {
		ep.token.Int = ep.addIdent(ep.token.String) // Complete the IDENT command with the index of the target.
		res = ep.token
		ep.getToken()
	} else if ep.token.Operator == seq.Factor {

		switch ep.token.Rune {
		case '[':
			ep.getToken()
			res = seq.Sequence{Operator: seq.Optional, Childs: []seq.Sequence{ep.expression()}, Pos: pos}
			ep.matchToken(']')
		case '(':
			ep.getToken()
			res = ep.expression()
			ep.matchToken(')')
		case '{':
			ep.getToken()
			res = seq.Sequence{Operator: seq.Repeat, Childs: []seq.Sequence{ep.expression()}, Pos: pos}
			ep.matchToken('}')
		case '+':
			res = seq.Sequence{Operator: seq.SkipSpaces, Bool: true, Pos: pos}
			ep.getToken()
		case '-':
			res = seq.Sequence{Operator: seq.SkipSpaces, Bool: false, Pos: pos}
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
	} else if ep.token.Operator == seq.Tag {
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

	// if res.Operator == seq.Basic && len(res.Childs) == 1 {
	// 	return res.Childs[0]
	// }
	return res
}

// also works like getToken(), but advances before that as much as it itself knows
// TODO: allow multiple strings/text, separated by ";"!
func (ep *ebnfParser) tag() seq.Sequence {
	pos := ep.sdx

	res := seq.Sequence{Operator: seq.Tag, Pos: pos}
	for {
		ep.getToken()
		res.TagChilds = seq.AppendPossibleSequence(res.TagChilds, ep.token)
		ep.getToken()
		if !(ep.token.Operator == seq.Factor && ep.ch == ',') {
			break
		}
	}

	ep.matchToken('>')
	return res
}

// also works like getToken(), but advances before that as much as it itself knows (= implements sequence)
func (ep *ebnfParser) term() seq.Sequence {
	pos := ep.sdx

	firstFactor := ep.factor()
	if firstFactor.Operator == seq.Tag {
		return ep.invalid("TAG is invalid at this position", pos)
	}

	res := []seq.Sequence{firstFactor}

	for {
		if (ep.token.Operator == seq.Factor && strings.IndexRune("})]>|.;", ep.token.Rune) >= 0) || ep.token.Operator == seq.Invalid {
			break
		}

		newFactor := ep.factor()

		// MOVE EVERYTHING THAT THE TAG DESRIBES (which is to the left) INTO INTO THE TAG!
		// If newFactor is a TAG, merge this TAG with the last command in firstFactor.
		if newFactor.Operator == seq.Tag {
			lastFactor := res[len(res)-1]
			res = res[:len(res)-1]                                                      // Remove the last factor from the result, because it will be appended again, but as a child of TAG.
			newFactor.Childs = seq.AppendPossibleSequence(newFactor.Childs, lastFactor) // Fill the TAG.
			res = append(res, newFactor)
		} else {
			// If newFactor is not a TAG, only append newFactor to the firstFactor.
			res = seq.AppendPossibleSequence(res, newFactor)
		}
	}

	if len(res) == 1 {
		return res[0]
	}
	return seq.Sequence{Operator: seq.Basic, Childs: res, Pos: pos}
}

// also works like getToken(), but advances before that as much as it itself knows
func (ep *ebnfParser) expression() seq.Sequence {
	pos := ep.sdx
	res := ep.term()

	if ep.token.Operator == seq.Factor && ep.token.Rune == '|' {
		res = seq.Sequence{Operator: seq.Or, Childs: []seq.Sequence{res}, Pos: pos} // Override the result (a factor) with the command for OR.
		for ep.token.Operator == seq.Factor && ep.token.Rune == '|' {
			ep.getToken()
			res.Childs = append(res.Childs, ep.term()) // Append the found alternative to the OR (It must not be ungrouped here, so use normal append() instead of seq.AppendPossibleSequence()).
		}
		if len(res.Childs) == 1 {
			return res.Childs[0]
		}
	}
	return res
}

// also works like getToken(), but advances before that as much as it itself knows
func (ep *ebnfParser) production() seq.Sequence {
	pos := ep.sdx
	// Returns a token or seq.Invalid; the real result is left in 'productions' etc, ...
	ep.getToken()

	if !(ep.token.Operator == seq.Factor && ep.token.Rune == '}') {
		if ep.token.Operator == seq.Invalid {
			return ep.invalid("Invalid EBNF (missing closing })", pos)
		}
		if ep.token.Operator != seq.Ident {
			return seq.Sequence{Operator: seq.Invalid, String: fmt.Sprintf("Ident expected but got %s", ep.token.Operator), Pos: pos}
		}

		ident := ep.token.String
		idx := ep.addIdent(ident)
		ep.getToken()

		var tag seq.Sequence
		foundTag := false
		if ep.token.Operator == seq.Factor && ep.token.Rune == '<' {
			tag = ep.tag()
			foundTag = true
		}

		ep.matchToken('=')
		if ep.token.Operator == seq.Invalid {
			return ep.token
		}
		if foundTag {
			tag.Childs = seq.AppendPossibleSequence(tag.Childs, ep.expression()) // Fill the TAG.
			ep.grammar.Productions = append(ep.grammar.Productions, seq.Sequence{Operator: seq.Production, String: ident, Int: idx, Childs: []seq.Sequence{tag}, Pos: pos})
		} else {
			ep.grammar.Productions = append(ep.grammar.Productions, seq.Sequence{Operator: seq.Production, String: ident, Int: idx, Childs: []seq.Sequence{ep.expression()}, Pos: pos})
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
	ep.grammar.Extras = make(map[string]seq.Sequence)
	ep.grammar.Productions = ep.grammar.Productions[:0]
	ep.ididx = ep.ididx[:0]
	ep.idents = ep.idents[:0]

	pos := ep.sdx
	ep.getToken()

	// Title
	if ep.token.Operator == seq.Terminal {
		ep.grammar.Extras["title"] = ep.token
		// title := seq.Sequence{Operator: seq.Production, String: "title", Int: -1, Childs: []seq.Sequence{ep.token}, Pos: pos}
		// ep.grammar.extras = append(ep.grammar.extras, title)
		pos = ep.sdx
		ep.getToken()
	}

	// Prolog
	if ep.token.Operator == seq.Factor && ep.token.Rune == '<' {
		ep.grammar.Extras["prolog.code"] = ep.tag()
		// tag := seq.Sequence{Operator: seq.Production, String: "prolog.code", Int: -1, Childs: []seq.Sequence{ep.tag()}, Pos: pos}
		// ep.grammar.extras = append(ep.grammar.extras, tag)
		pos = ep.sdx
		// ep.getToken()
	}

	PprintProductions(&ep.grammar.Productions, ">>>    ")

	// Main
	if !(ep.token.Operator == seq.Factor && ep.token.Rune == '{') {
		ep.invalid("Invalid EBNF (missing opening {)", pos)
		return
	}
	for {
		ep.token = ep.production()
		if (ep.token.Operator == seq.Factor && ep.token.Rune == '}') || ep.token.Operator == seq.Invalid {
			break
		}
	}
	pos = ep.sdx
	ep.getToken()

	// TODO: prolog and epilog code and id is not yet used!!!!

	// Epilog
	if ep.token.Operator == seq.Factor && ep.token.Rune == '<' {
		ep.grammar.Extras["epilog.code"] = ep.tag()
		// tag := seq.Sequence{Operator: seq.Production, String: "epilog.code", Int: -1, Childs: []seq.Sequence{ep.tag()}, Pos: pos}
		// ep.grammar.extras = append(ep.grammar.extras, tag)
		pos = ep.sdx
		// ep.getToken()
	}

	// Entry point
	if ep.token.Operator == seq.Ident {
		start := ep.token
		start.Int = ep.addIdent(ep.token.String)
		ep.grammar.Extras["start"] = start
		// start := seq.Sequence{Operator: seq.Production, String: "start", Int: -1, Childs: []seq.Sequence{ep.token}, Pos: pos}
		// ep.grammar.extras = append(ep.grammar.extras, start)
		pos = ep.sdx
		ep.getToken()
	} else {
		ep.invalid("Invalid EBNF (missing entry point)", pos)
	}

	// Comment
	if ep.token.Operator == seq.Terminal {
		ep.grammar.Extras["comment"] = ep.token
		// comment := seq.Sequence{Operator: seq.Production, String: "comment", Int: -1, Childs: []seq.Sequence{ep.token}, Pos: pos}
		// ep.grammar.extras = append(ep.grammar.extras, comment)
		pos = ep.sdx
		ep.getToken()
	}

	// End of parsing
	if ep.token.Operator != seq.Invalid {
		ep.invalid("Invalid EBNF (missing EOF?)", pos)
		return
	}
	if ep.err {
		return
	}

	ep.token = seq.Sequence{}
	ep.verifyGrammar()
	ep.resolveIdIdx(&ep.grammar.Productions)

	// Also resolve the index of the start rule.
	tmpStart := ep.grammar.Extras["start"]
	tmpStart.Int = ep.ididx[tmpStart.Int]
	ep.grammar.Extras["start"] = tmpStart
}

func (ep *ebnfParser) resolveIdIdx(productions *[]seq.Sequence) {
	for i := range *productions {
		rule := &(*productions)[i]
		if len(rule.Childs) > 0 {
			ep.resolveIdIdx(&rule.Childs)
		}
		if rule.Operator == seq.Production || rule.Operator == seq.Ident {
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
