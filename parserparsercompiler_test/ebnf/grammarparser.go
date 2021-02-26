package ebnf

import (
	"strings"
	"fmt"
)

type grammarParser struct {
	src     []rune
	ch      rune
	sdx     int
	err     bool
	grammar Grammar

	// for new grammar
	newIdents  []string
	newGrammar Grammar
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

// TODO DEDUPLICATE!
func (gp *grammarParser) addIdent(ident string) int {
	k := -1
	for i, id := range gp.newIdents {
		if id == ident {
			k = i
			break
		}
	}
	if k == -1 {
		gp.newIdents = append(gp.newIdents, ident)
		k = len(gp.newIdents) - 1
		gp.newGrammar.ididx = append(gp.newGrammar.ididx, -1)
	}
	return k
}

// The self-referential EBNF is (different description form!):
//
//      EBNF = Production* .
//      Production = <ident> "=" Expression "." .
//      Expression = Sequence ("|" Sequence)* .
//      Sequence = Term+ .
//      Term = (<ident> | <string> | ("<" <ident> ">") | ("(" Expression ")")) ("*" | "+" | "?" | "!")? .
//
// The self-referential EBNF is:
//
// {
// EBNF = "{" { production } "}" .
// production  = name "=" [ expression ] "." .
// expression  = name | terminal [ "..." terminal ] | sequence | alternative | group | option | repetition .
// sequence    = expression expression { expression } .
// alternative = expression "|" expression { "|" expression } .
// group       = "(" expression ")" .
// option      = "[" expression "]" .
// repetition  = "{" expression "}" .
// }
//
// // rule == production
// // factors == non-terminal expression. a subgroup of productions/rules
// // ident == name             //  <=  identifies another block (== address of the other expression)
// // string == token == terminal == text
// // or == alternative
//
//		SOOOOOO:
//
// The rules that applies() has to deal with are:
// {factors} - if rule[0] is not string,
// just apply one after the other recursively.
// {"TERMINAL", "a1"}       -- literal constants
// {"OR", <e1>, <e2>, ...}  -- (any) one of n
// {"REPEAT", <e1>}         -- as per "{}" in ebnf
// {"OPTIONAL", <e1>}       -- as per "[]" in ebnf
// {"IDENT", <name>, idx}   -- apply the sub-rule (its a link to the sub-rule)
// {"TAG", code, <name>, idx }  ---- from dma: the semantic description in IL or something else (script language). also other things like coloring
func (gp *grammarParser) applies(rule sequence) (object, bool) {
	
	// DMA: remove again - test
	foo := '-'
	if gp.sdx < len(gp.src) {
		foo = gp.src[gp.sdx]
	}
	pprint(fmt.Sprintf("rule for pos # %d (%c)", gp.sdx, foo), rule)
	
	var localProductions sequence
	localProductions = localProductions[:0]

	wasSdx := gp.sdx // in case of failure

	r1 := rule[0]
	if _, ok := r1.(string); !ok {
		for i := 0; i < len(rule); i++ { // if there is no string at rule[0], it is a group of rules. iterate through them and apply

			newProduction, ok := gp.applies(rule[i].(sequence))

			if ok && len(newProduction.(sequence)) > 1 {
				if newProduction.(sequence)[1] != nil {
					localProductions = append(localProductions, newProduction.(sequence)[1])
				}
			}

			if !ok {
				gp.sdx = wasSdx
				return nil, false
			}

		}
	} else if r1 == "TERMINAL" {
		gp.skipSpaces()

		// myStartofTerminal := gp.sdx

		r2 := []rune(rule[1].(string))
		for i := 0; i < len(r2); i++ {
			if gp.sdx >= len(gp.src) || gp.src[gp.sdx] != r2[i] {
				gp.sdx = wasSdx
				return nil, false
			}
			gp.sdx++
		}

		newProduction := rule[1].(string) // string(gp.src[myStartofTerminal:gp.sdx]) //    rule[1].(string)
		localProductions = append(localProductions, sequence{"TERMINAL", newProduction})

	} else if r1 == "OR" {
		for i := 1; i < len(rule); i++ {

			newProduction, ok := gp.applies(rule[i].(sequence))
			if ok {
				// TODO: this only iterates through all aternatives until it finds one that matches. we need to collect all OR
				localProductions = append(localProductions, sequence{"OR", newProduction})
			}

			if ok {
				// TODO: return only the current part: replace localProductions everywhere in this function with a locally generated object!
				if len(localProductions) == 1 {
					return localProductions[0], true
				}
				return localProductions, true
			}
		}
		gp.sdx = wasSdx
		return nil, false
	} else if r1 == "REPEAT" {
		// for gp.applies(rule[1].(sequence)) {}
		for {
			newProduction, ok := gp.applies(rule[1].(sequence))
			if ok {
				localProductions = append(localProductions, sequence{"REPEAT", newProduction})
			} else {
				break
			}
		}
	} else if r1 == "OPTIONAL" {

		newProduction, ok := gp.applies(rule[1].(sequence))
		if ok {
			localProductions = append(localProductions, sequence{"OPTIONAL", newProduction})
		}

	} else if r1 == "IDENT" { // "IDENT" identifies another block (and its index): this is a "IDENT" to the expression-block which is at position 3: { "IDENT", "expression", 3 }
		i := rule[2].(int)
		ii := gp.grammar.ididx[i]

		newProduction, ok := gp.applies(gp.grammar.productions[ii][2].(sequence))
		if ok {
			ident := "IDENT" // TODO: find what belongs here!
			idx := gp.addIdent(ident)
			localProductions = append(localProductions, sequence{ident, idx, newProduction})
		} else {
			gp.sdx = wasSdx
			return nil, false
		}

	} else if r1 == "TAG" {		// from DMA

		newProduction := "TAG"
		localProductions = append(localProductions, sequence{newProduction})

	} else {
		panic("invalid rule in applies() function")
	}

	// TODO: return only the current part: replace localProductions everywhere in this function with a locally generated object!
	if len(localProductions) == 1 {
		return localProductions[0], true
	}
	return localProductions, true
}

func (gp *grammarParser) parseWithGrammarInternal(test string) bool {
	gp.newIdents = gp.newIdents[:0]
	gp.newGrammar.ididx = gp.newGrammar.ididx[:0]
	gp.newGrammar.productions = gp.newGrammar.productions[:0]
	// ep.extras = ep.extras[:0]

	gp.src = []rune(test)
	gp.sdx = 0
	res := false
	if len(gp.grammar.productions) > 0 {
		// ORIGINAL CALL:
		// res = gp.applies(gp.grammar.productions[0][2].(sequence))
		// ident := "DMA_TEST_START"
		// idx := gp.addIdent(ident)
		newProduction, ok := gp.applies(gp.grammar.productions[0][2].(sequence))
		// if ok {
		// 	gp.newGrammar.productions = newProduction
		// }

		pprint("productions of new grammar", newProduction)

		// pprint("productions of new grammar", gp.newGrammar.productions)

		res = ok
	}
	gp.skipSpaces()
	if gp.sdx < len(gp.src) {
		res = false
	}

	return res
}

func ParseWithGrammar(grammar Grammar, srcCode string) (success bool, e error) {
	var gp grammarParser
	gp.err = false

	// defer func() {
	// 	if err := recover(); err != nil {
	// 		success = false
	// 		e = fmt.Errorf(fmt.Sprintf("%s", err))
	// 	}
	// }()

	gp.grammar = grammar
	return gp.parseWithGrammarInternal(srcCode), nil
}
