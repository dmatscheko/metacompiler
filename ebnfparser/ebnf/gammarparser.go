// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package ebnf is a library for EBNF grammars. The input is text ([]byte)
// satisfying the following grammar (represented itself in EBNF):
//
//	Production  = name "=" [ Expression ] "." .
//	Expression  = Alternative { "|" Alternative } .
//	Alternative = Term { Term } .
//	Term        = name | token [ "…" token ] | Group | Option | Repetition .
//	Group       = "(" Expression ")" .
//	Option      = "[" Expression "]" .
//	Repetition  = "{" Expression "}" .
//
// A name is a Go identifier, a token is a Go string, and comments
// and white space follow the same rules as for the Go language.
// Production names starting with an uppercase Unicode letter denote
// non-terminal productions (i.e., productions which allow white-space
// and comments between tokens); all other production names denote
// lexical productions.
//
package ebnf

import (
	"fmt"
	"io"
	"text/scanner"
	"unicode/utf8"
)

// ----------------------------------------------------------------------------
// Grammar parser

type grammarParser struct {
	errors errorList

	// parser part
	scanner scanner.Scanner
	pos     scanner.Position // token position
	tok     rune             // one token look-ahead
	lit     string           // token literal

	// grammar part
	worklist []*Production
	reached  Grammar // set of productions reached from (and including) the root production
	grammar  Grammar
}

func (gp *grammarParser) error(pos scanner.Position, msg string) {
	gp.errors = append(gp.errors, newError(pos, msg))
}

func (gp *grammarParser) push(prod *Production) {
	name := prod.Name.String
	if _, found := gp.reached[name]; !found {
		gp.worklist = append(gp.worklist, prod)
		gp.reached[name] = prod
	}
}

func (gp *grammarParser) verifyChar(x *Token) rune {
	s := x.String
	if utf8.RuneCountInString(s) != 1 {
		gp.error(x.Pos(), "single char expected, found "+s)
		return 0
	}
	ch, _ := utf8.DecodeRuneInString(s)
	return ch
}

// The rules that applyExpr() has to deal with are:
// {factors} - if rule[0] is not string,
// just apply one after the other recursively.
// {"terminal", "a1"}       -- literal constants
// {"or", <e1>, <e2>, ...}  -- (any) one of n
// {"repeat", <e1>}         -- as per "{}" in ebnf
// {"optional", <e1>}       -- as per "[]" in ebnf
// {"ident", <name>, idx}   -- apply the sub-rule
func (gp *grammarParser) applyExpr(expr Expression, lexical bool) {
	fmt.Printf("%v\n", expr)

	switch x := expr.(type) {
	case nil:
		// empty expression
	case Alternative:
		for _, e := range x {
			gp.applyExpr(e, lexical)
		}
	case Sequence:
		for _, e := range x {
			gp.applyExpr(e, lexical)
		}
	case *Name:
		// a production with this name must exist;
		// add it to the worklist if not yet processed
		if prod, found := gp.grammar[x.String]; found {
			gp.push(prod)
		} else {
			gp.error(x.Pos(), "missing production "+x.String)
		}
		// within a lexical production references
		// to non-lexical productions are invalid
		if lexical && !isLexical(x.String) {
			gp.error(x.Pos(), "reference to non-lexical production "+x.String)
		}
	case *Token:
		// nothing to do for now
	case *Range:
		i := gp.verifyChar(x.Begin)
		j := gp.verifyChar(x.End)
		if i >= j {
			gp.error(x.Pos(), "decreasing character range")
		}
	case *Group:
		gp.applyExpr(x.Body, lexical)
	case *Option:
		gp.applyExpr(x.Body, lexical)
	case *Repetition:
		gp.applyExpr(x.Body, lexical)
	case *Bad:
		gp.error(x.Pos(), x.Error)
	default:
		panic(fmt.Sprintf("internal error: unexpected type %T", expr))
	}
}

func (gp *grammarParser) next() {
	gp.tok = gp.scanner.Scan()
	gp.pos = gp.scanner.Position
	gp.lit = gp.scanner.TokenText()
}

func (gp *grammarParser) errorExpected(pos scanner.Position, msg string) {
	msg = `expected "` + msg + `"`
	if pos.Offset == gp.pos.Offset {
		// the error happened at the current position;
		// make the error message more specific
		msg += ", found " + scanner.TokenString(gp.tok)
		if gp.tok < 0 {
			msg += " " + gp.lit
		}
	}
	gp.error(pos, msg)
}

func (gp *grammarParser) expect(tok rune) scanner.Position {
	pos := gp.pos
	if gp.tok != tok {
		gp.errorExpected(pos, scanner.TokenString(tok))
	}
	gp.next() // make progress in any case
	return pos
}

// The rules that applies() has to deal with are:
// {factors} - if rule[0] is not string,
// just apply one after the other recursively.
// {"terminal", "a1"}       -- literal constants
// {"or", <e1>, <e2>, ...}  -- (any) one of n
// {"repeat", <e1>}         -- as per "{}" in ebnf
// {"optional", <e1>}       -- as per "[]" in ebnf
// {"ident", <name>, idx}   -- apply the sub-rule
func (gp *grammarParser) apply(rule sequence) bool {
	wasSdx := sdx // in case of failure
	r1 := rule[0]
	if _, ok := r1.(string); !ok {
		for i := 0; i < len(rule); i++ {
			if !apply(rule[i].(sequence)) {
				sdx = wasSdx
				// fmt.Println("AAAAAAAAAAAAAA")
				return false
			}
		}
	} else if r1 == "terminal" {
		skipSpaces()
		r2 := []rune(rule[1].(string))
		for i := 0; i < len(r2); i++ {
			if sdx >= len(src) || src[sdx] != r2[i] {
				sdx = wasSdx
				// fmt.Println("BBBBBBBBBBBBBB")
				return false
			}
			sdx++
		}
	} else if r1 == "or" {
		for i := 1; i < len(rule); i++ {
			if apply(rule[i].(sequence)) {
				return true
			}
		}
		sdx = wasSdx
		return false
	} else if r1 == "repeat" {
		for apply(rule[1].(sequence)) {
		}
	} else if r1 == "optional" {
		apply(rule[1].(sequence))
	} else if r1 == "ident" {
		i := rule[2].(int)
		ii := ididx[i]
		if !apply(productions[ii][2].(sequence)) {
			sdx = wasSdx
			// fmt.Println("CCCCCCCCCCCCCCCC")
			return false
		}
	} else {
		panic("invalid rule in apply() function")
	}
	return true
}

func (gp *grammarParser) parseProduction() *Production {
	name := gp.parseIdentifier()
	gp.expect('=')
	var expr Expression
	if gp.tok != '.' {
		expr = gp.parseExpression()
	}
	gp.expect('.')
	return &Production{name, expr}
}

func (gp *grammarParser) parse(filename string, src io.Reader) {
	gp.scanner.Init(src)
	gp.scanner.Filename = filename
	gp.next() // initializes pos, tok, lit

	for gp.tok != scanner.EOF {
		prod := gp.apply()
		name := prod.Name.String
		if _, found := gp.grammar[name]; !found {
			gp.grammar[name] = prod
		} else {
			gp.error(prod.Pos(), name+" declared already")
		}
	}
}

// ParseWithGrammar parses according to a grammar:
//	- all productions used are defined
//	- all productions defined are used when beginning at start
//	- lexical productions refer only to other lexical productions
//
// Position information is interpreted relative to the file set fset.
//
func ParseWithGrammar(grammar Grammar, start string, filename string, src io.Reader) error {
	var gp grammarParser
	gp.grammar = grammar

	// gp.apply(grammar, start)
	return gp.errors.Err()
}
