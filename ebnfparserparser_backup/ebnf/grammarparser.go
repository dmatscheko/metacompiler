package ebnf

import (
	"strings"
)

type grammarParser struct {
	src     []rune
	ch      rune
	sdx     int
	err     bool
	grammar Grammar
}

// TODO: DEDUPLICATE!!
func (gp *grammarParser) skipSpaces() {
	for {
		if gp.sdx >= len(gp.src) {
			break
		}
		gp.ch = gp.src[gp.sdx]
		if strings.IndexRune(" \t\r\n", gp.ch) == -1 {
			break
		}
		gp.sdx++
	}
}

// The rules that applies() has to deal with are:
// {factors} - if rule[0] is not string,
// just apply one after the other recursively.
// {"terminal", "a1"}       -- literal constants
// {"or", <e1>, <e2>, ...}  -- (any) one of n
// {"repeat", <e1>}         -- as per "{}" in ebnf
// {"optional", <e1>}       -- as per "[]" in ebnf
// {"ident", <name>, idx}   -- apply the sub-rule
func (gp *grammarParser) applies(rule sequence) bool {
	wasSdx := gp.sdx // in case of failure
	r1 := rule[0]
	if _, ok := r1.(string); !ok {
		for i := 0; i < len(rule); i++ {
			if !gp.applies(rule[i].(sequence)) {
				gp.sdx = wasSdx
				// fmt.Println("AAAAAAAAAAAAAA")
				return false
			}
		}
	} else if r1 == "terminal" {
		gp.skipSpaces()
		r2 := []rune(rule[1].(string))
		for i := 0; i < len(r2); i++ {
			if gp.sdx >= len(gp.src) || gp.src[gp.sdx] != r2[i] {
				gp.sdx = wasSdx
				// fmt.Println("BBBBBBBBBBBBBB")
				return false
			}
			gp.sdx++
		}
	} else if r1 == "or" {
		for i := 1; i < len(rule); i++ {
			if gp.applies(rule[i].(sequence)) {
				return true
			}
		}
		gp.sdx = wasSdx
		return false
	} else if r1 == "repeat" {
		for gp.applies(rule[1].(sequence)) {
		}
	} else if r1 == "optional" {
		gp.applies(rule[1].(sequence))
	} else if r1 == "ident" {
		i := rule[2].(int)
		ii := gp.grammar.ididx[i]
		if !gp.applies(gp.grammar.productions[ii][2].(sequence)) {
			gp.sdx = wasSdx
			// fmt.Println("CCCCCCCCCCCCCCCC")
			return false
		}
	} else {
		panic("invalid rule in applies() function")
	}
	return true
}

func (gp *grammarParser) checkValid(test string) bool {
	gp.src = []rune(test)
	gp.sdx = 0
	res := false
	if len(gp.grammar.productions) > 0 {
		res = gp.applies(gp.grammar.productions[0][2].(sequence))
	}
	gp.skipSpaces()
	if gp.sdx < len(gp.src) {
		res = false
	}

	return res
}

func ParseWithGrammar(grammar Grammar, srcCode string) (bool, error) {
	var gp grammarParser
	gp.err = false

	gp.grammar = grammar

	return gp.checkValid(srcCode), nil
}
