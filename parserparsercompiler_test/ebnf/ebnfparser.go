package ebnf

import (
	"fmt"
	"strings"
)

// type aliases for Phix types
type object = interface{}
type sequence = []object

// TODO: change to needed values!
type Grammar = struct {
	ididx       []int
	productions []sequence
}

// ----------------------------------------------------------------------------
// EBNF parser

type ebnfParser struct {
	src     []rune
	ch      rune
	sdx     int
	token   object
	isSeq   bool
	err     bool
	idents  []string
	extras  sequence
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

func (ep *ebnfParser) invalid(msg string) int {
	ep.err = true
	fmt.Println("char", ep.sdx, ": ", msg)
	ep.sdx = len(ep.src) // set to eof
	return -1
}

func (ep *ebnfParser) getToken() {
	// Yields a single character token, one of {}()[]|=.;
	// or {"TERMINAL",string} or {"IDENT", string} or -1.
	ep.skipSpaces()
	if ep.sdx >= len(ep.src) {
		ep.token = -1
		ep.isSeq = false
		return
	}
	tokstart := ep.sdx
	if strings.IndexRune("{}()[]<>|=.;+-", ep.ch) >= 0 {
		ep.sdx++
		ep.token = ep.ch
		ep.isSeq = false
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
							case 'n':
								tokenrunes[pos] = '\n'
							case 't':
								tokenrunes[pos] = '\t'
							}
						}
					}
				}
				// fmt.Printf(">>> %s\n", string(tokenrunes))

				ep.token = sequence{"TERMINAL", string(tokenrunes)}
				ep.isSeq = true
				return
			}
			if ep.src[tokend] != '\\' {
				atEscapeCh = 0
			}

		}
		ep.token = ep.invalid("no closing quote")
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
		ep.token = sequence{"IDENT", string(ep.src[tokstart:ep.sdx])}
		ep.isSeq = true
	} else {
		ep.token = ep.invalid(fmt.Sprintf("invalid ebnf (char %c)", ep.ch))
		ep.isSeq = false
	}
}

// also works like getToken()
func (ep *ebnfParser) matchToken(ch rune) {
	if ep.token != ch {
		ep.token = ep.invalid(fmt.Sprintf("invalid ebnf (%c expected)", ch))
		ep.isSeq = false
	} else {
		ep.getToken()
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
		ep.grammar.ididx = append(ep.grammar.ididx, -1)
	}
	return k
}

// also works like getToken(), but advances before that as much as it itself knows
func (ep *ebnfParser) factor() object {
	var res object
	if ep.isSeq {
		t := ep.token.([]object)
		if t[0] == "IDENT" {
			idx := ep.addIdent(t[1].(string))
			t = append(t, idx)
			ep.token = t
		}
		res = ep.token
		ep.getToken()
	} else if ep.token == '[' {
		ep.getToken()
		res = sequence{"OPTIONAL", ep.expression()}
		ep.matchToken(']')
	} else if ep.token == '(' {
		ep.getToken()
		res = ep.expression()
		ep.matchToken(')')
	} else if ep.token == '{' {
		ep.getToken()
		res = sequence{"REPEAT", ep.expression()}
		ep.matchToken('}')
	} else if ep.token == '+' {
		res = sequence{"SKIPSPACES", true} // from DMA
		ep.getToken()
	} else if ep.token == '-' {
		res = sequence{"SKIPSPACES", false} // from DMA
		ep.getToken()
	} else if ep.token == '<' { // from DMA
		tag := ep.tag()
		if tag != nil {
			// pprint("TEST:", tag)
			res = tag
		} else {
			res = sequence{"TAG", 0} // from DMA
		}
	} else {
		panic(fmt.Sprintf("invalid token in factor() function (%#q)", ep.token))
	}

	if s, ok := res.(sequence); ok && len(s) == 1 {
		return s[0]
	}
	return res
}

// also works like getToken(), but advances before that as much as it itself knows
func (ep *ebnfParser) tag() object {
	var res object
	if ep.token == '<' {
		ep.getToken()
		res = sequence{"TAG", ep.expression()} // from DMA // TODO: allow multiple strings/text, separated by ";"!
		ep.matchToken('>')
	} else {
		panic("invalid token in tag")
	}
	if s, ok := res.(sequence); ok && len(s) == 1 {
		return s[0]
	}
	return res
}

// also works like getToken(), but advances before that as much as it itself knows (= implements sequence)
func (ep *ebnfParser) term() object {
	factor := ep.factor()
	if f, ok := factor.(sequence); ok && len(f) > 0 {
		if f[0] == "TAG" {
			return ep.invalid("tag is invalid at this position!")
		}
	}
	res := sequence{factor}

	tokens := []object{-1, '|', '.', ';', ')', ']', '}', '>'}
outer:
	for {
		for _, t := range tokens {
			if t == ep.token {
				break outer
			}
		}

		newFactor := ep.factor()

		// if newFactor is a TAG, merge this TAG with the last factor in res
		if f, ok := newFactor.(sequence); ok && len(f) > 0 {

			if f[0] == "TAG" {
				lastFactor := res[len(res)-1]

				tmpFactor := lastFactor.([]object)
				tmpFactor = append(tmpFactor, newFactor)

				newFactor = tmpFactor

				// remove the last factor from res, because it will be appended in its new form with TAG
				res = res[:len(res)-1]
			}

		}

		res = append(res, newFactor)
		// res = append(res, ep.factor()) // original version without tags
	}
	if len(res) == 1 {
		return res[0]
	}
	return res
}

// also works like getToken(), but advances before that as much as it itself knows
func (ep *ebnfParser) expression() object {
	res := sequence{ep.term()}

	if ep.token == '|' {
		res = sequence{"OR", res[0]}
		for ep.token == '|' {
			ep.getToken()
			res = append(res, ep.term())
		}
	}
	if len(res) == 1 {
		return res[0]
	}
	return res
}

// also works like getToken(), but advances before that as much as it itself knows
func (ep *ebnfParser) production() object {
	// Returns a token or -1; the real result is left in 'productions' etc,
	ep.getToken()
	if ep.token != '}' {
		if ep.token == -1 {
			return ep.invalid("invalid ebnf (missing closing })")
		}
		if !ep.isSeq {
			return -1
		}
		t := ep.token.(sequence)
		if t[0] != "IDENT" {
			return -1
		}
		ident := t[1].(string)
		idx := ep.addIdent(ident)
		ep.getToken()

		var tag object
		if ep.token == '<' {
			tag = ep.tag()
		}

		ep.matchToken('=')
		if ep.token == -1 {
			return -1
		}
		if tag != nil {
			ep.grammar.productions = append(ep.grammar.productions, sequence{ident, idx, ep.expression(), tag})
		} else {
			ep.grammar.productions = append(ep.grammar.productions, sequence{ident, idx, ep.expression()})
		}
		ep.grammar.ididx[idx] = len(ep.grammar.productions) - 1
	}
	return ep.token
}

// ep.err == false, if the parsing went OK
func (ep *ebnfParser) parse(srcEbnf string) {
	ep.err = false
	ep.src = []rune(srcEbnf)
	ep.sdx = 0
	ep.idents = ep.idents[:0]
	ep.grammar.ididx = ep.grammar.ididx[:0]
	ep.grammar.productions = ep.grammar.productions[:0]
	ep.extras = ep.extras[:0]
	ep.getToken()
	if ep.isSeq {
		t := ep.token.(sequence)
		t[0] = "title"
		ep.extras = append(ep.extras, ep.token)
		ep.getToken()
	}
	if ep.token != '{' {
		ep.invalid("invalid ebnf (missing opening {)")
		return
	}
	for {
		ep.token = ep.production()
		if ep.token == '}' || ep.token == -1 {
			break
		}
	}
	ep.getToken()
	if ep.isSeq {
		t := ep.token.(sequence)
		t[0] = "comment"
		ep.extras = append(ep.extras, ep.token)
		ep.getToken()
	}
	if ep.token != -1 {
		ep.invalid("invalid ebnf (missing eof?)")
		return
	}
	if ep.err {
		return
	}
	k := -1
	for i, idx := range ep.grammar.ididx {
		if idx == -1 {
			k = i
			break
		}
	}
	if k != -1 {
		ep.invalid(fmt.Sprintf("invalid ebnf (undefined:%s)", ep.idents[k]))
		return
	}
}

func ParseEBNF(srcEbnf string) (g Grammar, e error) {
	pprintSrc("parse", srcEbnf)
	// fmt.Printf("parse:\n%s\n", srcEbnf)

	var ep ebnfParser

	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("%s\n  ===> Fail\n", err)
			e = fmt.Errorf("Fail")
		}
	}()

	ep.parse(srcEbnf)

	if !ep.err {
		fmt.Println("  ===> Success")
		pprint("productions", ep.grammar.productions)
		pprint("ididx", ep.grammar.ididx)
		pprint("idents", ep.idents)
		pprint("extras", ep.extras)
		return ep.grammar, nil
	} else {
		fmt.Println("  ===> Fail\n")
	}

	return Grammar{}, fmt.Errorf("Fail")
}
