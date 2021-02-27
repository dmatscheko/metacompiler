package ebnf

import (
	"fmt"
	"strings"
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
// The rules that applies() has to deal with are BASICALLY THE SAME AS AN BNF-PARSER with annotations (NOT EBNF):
// {factors} - if rule[0] is not string,
// just apply one after the other recursively.
// {"TERMINAL", "a1"}       -- literal constants
// {"OR", <e1>, <e2>, ...}  -- (any) one of n
// {"REPEAT", <e1>}         -- as per "{}" in ebnf
// {"OPTIONAL", <e1>}       -- as per "[]" in ebnf
// {"IDENT", <name>, idx}   -- apply the sub-rule (its a link to the sub-rule) (its a production)
// {"TAG", code, <name>, idx }  ---- from dma: the semantic description in IL or something else (script language). also other things like coloring
//
// TODO: REMEMBER WHAT HAS BEEN TRIED ALREADY FOR A POSITION!
//
// func (gp *grammarParser) applies(rule sequence, doSkipSpaces bool, depth int) (object, bool) {
// 	// func (gp *grammarParser) applies(rule sequence, depth int) (object, bool) {
// 	var localProductions sequence
// 	// localProductions = localProductions[:0]

// 	wasSdx := gp.sdx // in case of failure

// 	r1 := rule[0]
// 	if _, ok := r1.(string); !ok {
// 		gp.printTrace(rule, "DESCENT INTO SEQUENCE", depth)

// 		for i := 0; i < len(rule); i++ { // if there is no string at rule[0], it is a group of rules. iterate through them and apply

// 			newProduction, ok := gp.applies(rule[i].(sequence), doSkipSpaces, depth+1)

// 			if ok && newProduction != nil {
// 				localProductions = append(localProductions, sequence{"DESCENT", newProduction})
// 			}

// 			// localProductions = append(localProductions, newProduction)
// 			// if x, ok := newProduction.(sequence); ok && len(x) > 1 {
// 			// 	if x[len(x)-1] != nil {
// 			// 		localProductions = append(localProductions, x[len(x)-1])
// 			// 	}
// 			// }

// 			if !ok {
// 				gp.sdx = wasSdx
// 				return nil, false
// 			}

// 		}
// 	} else if r1 == "SKIPSPACES" {
// 		doSkipSpaces = rule[1].(bool)
// 	} else if r1 == "TERMINAL" {
// 		if doSkipSpaces { // There can be white space in strings/text! Do not skip that.
// 			gp.skipSpaces()
// 		}
// 		gp.printTrace(rule, "TERMINAL", depth)

// 		// myStartofTerminal := gp.sdx

// 		r2 := []rune(rule[1].(string))
// 		for i := 0; i < len(r2); i++ {
// 			if gp.sdx >= len(gp.src) || gp.src[gp.sdx] != r2[i] {
// 				gp.sdx = wasSdx
// 				return nil, false
// 			}
// 			gp.sdx++
// 		}

// 		localProductions = append(localProductions, sequence{"TERMINAL", rule[1].(string)}) // TODO: only if TAG says so!
// 		// newProduction := rule[1].(string) // string(gp.src[myStartofTerminal:gp.sdx]) //    rule[1].(string)
// 		// localProductions = append(localProductions, sequence{"TERMINAL", newProduction})

// 	} else if r1 == "OR" {
// 		gp.printTrace(rule, "OR", depth)
// 		for i := 1; i < len(rule); i++ {

// 			newProduction, ok := gp.applies(rule[i].(sequence), doSkipSpaces, depth+1)
// 			// _, ok := gp.applies(rule[i].(sequence), doSkipSpaces, depth+1)
// 			if ok && newProduction != nil {
// 				localProductions = append(localProductions, sequence{"OR", newProduction})
// 				// 	// TODO: this only iterates through all aternatives until it finds one that matches. we need to collect all OR
// 				// 	localProductions = append(localProductions, sequence{"OR", newProduction})
// 			}

// 			if ok {
// 				// TODO: return only the current part: replace localProductions everywhere in this function with a locally generated object!
// 				if len(localProductions) == 1 {
// 					return localProductions[0], true
// 				}
// 				return localProductions, true
// 			}
// 		}
// 		gp.sdx = wasSdx
// 		return nil, false
// 	} else if r1 == "REPEAT" {
// 		gp.printTrace(rule, "REPEAT", depth)
// 		// for gp.applies(rule[1].(sequence)) {}
// 		for {
// 			newProduction, ok := gp.applies(rule[1].(sequence), doSkipSpaces, depth+1)
// 			if ok && newProduction != nil {
// 				localProductions = append(localProductions, sequence{"REPEAT", newProduction})
// 			} else if !ok {
// 				break
// 			}
// 		}
// 	} else if r1 == "OPTIONAL" {
// 		gp.printTrace(rule, "OPTIONAL", depth)
// 		newProduction, ok := gp.applies(rule[1].(sequence), doSkipSpaces, depth+1)
// 		if ok && newProduction != nil {
// 			localProductions = append(localProductions, sequence{"OPTIONAL", newProduction})
// 		}

// 	} else if r1 == "IDENT" { // "IDENT" identifies another block (and its index), it is basically a link: This would e.g. be an "IDENT" to the expression-block which is at position 3: { "IDENT", "expression", 3 }
// 		gp.printTrace(rule, "IDENT", depth)

// 		i := rule[2].(int)
// 		ii := gp.grammar.ididx[i]
// 		newProduction, ok := gp.applies(gp.grammar.productions[ii][2].(sequence), doSkipSpaces, depth+1)
// 		if ok {
// 			ident := "IDENT" // TODO: find what belongs here!
// 			idx := gp.addIdent(ident)
// 			if newProduction != nil {
// 				localProductions = append(localProductions, sequence{ident, idx, newProduction})
// 			}
// 		} else {
// 			gp.sdx = wasSdx
// 			return nil, false
// 		}

// 	} else {
// 		gp.printTrace(rule, "------INVALID-----", depth)
// 		panic("invalid rule in applies() function")
// 	}

// 	// // DMA
// 	// for _, elem := range rule {
// 	// 	if e, ok := elem.(string); !ok {
// 	// 		if e == "TAG" {
// 	// 			pprint("TAG", rule)
// 	// 		}
// 	// 	}
// 	// }

// 	// TODO: return only the current part: replace localProductions everywhere in this function with a locally generated object!
// 	if len(localProductions) == 1 {
// 		return localProductions[0], true
// 	}
// 	return localProductions, true
// }

func (gp *grammarParser) applies(rule sequence, doSkipSpaces bool, depth int) object {
	wasSdx := gp.sdx // in case of failure
	r1 := rule[0]

	var localProductions sequence
	localProductions = localProductions[:0]

	// gp.printTrace(rule, "A", depth)

	if _, ok := r1.(string); !ok { // "SEQUENCE" (if there is no string at rule[0], it is a group/sequence of rules. iterate through them and apply)
		for i := 0; i < len(rule); i++ {
			newProduction := gp.applies(rule[i].(sequence), doSkipSpaces, depth+1)
			if newProduction == nil {
				gp.sdx = wasSdx
				return nil
			}
			localProductions = append(localProductions, newProduction)
		}
	} else if r1 == "TERMINAL" {
		if doSkipSpaces { // There can be white space in strings/text! Do not skip that.
			gp.skipSpaces()
		}
		r2 := []rune(rule[1].(string))
		for i := 0; i < len(r2); i++ {
			if gp.sdx >= len(gp.src) || gp.src[gp.sdx] != r2[i] {
				gp.sdx = wasSdx
				return nil
			}
			gp.sdx++
		}
		localProductions = append(localProductions, rule)
		// pprint("X", rule)
	} else if r1 == "OR" {
		for i := 1; i < len(rule); i++ {
			if newProduction := gp.applies(rule[i].(sequence), doSkipSpaces, depth+1); newProduction != nil {
				return newProduction
			}
		}
		gp.sdx = wasSdx
		return nil
	} else if r1 == "REPEAT" {
		for {
			newProduction := gp.applies(rule[1].(sequence), doSkipSpaces, depth+1)
			if newProduction == nil {
				break
			}
			localProductions = append(localProductions, newProduction)
		}
	} else if r1 == "OPTIONAL" {
		newProduction := gp.applies(rule[1].(sequence), doSkipSpaces, depth+1)
		if newProduction != nil {
			localProductions = append(localProductions, newProduction)
		}
	} else if r1 == "IDENT" { // "IDENT" identifies another block (and its index), it is basically a link: This would e.g. be an "IDENT" to the expression-block which is at position 3: { "IDENT", "expression", 3 }
		i := rule[2].(int)
		ii := gp.grammar.ididx[i]
		newProduction := gp.applies(gp.grammar.productions[ii][2].(sequence), doSkipSpaces, depth+1)
		if newProduction == nil {
			gp.sdx = wasSdx
			return nil
		}
		localProductions = append(localProductions, newProduction)
	} else if r1 == "TAG" {
		newProduction := gp.applies(rule[2].(sequence), doSkipSpaces, depth+1)
		if newProduction != nil {
			return sequence{rule[0], rule[1], newProduction}
		}
		return nil
	} else if r1 == "SKIPSPACES" { // TODO: modify SKIPSPACES so that the chars to skip must be given to the command. e.g.: {"SKIPSPACES", "\n\t :;"}
		doSkipSpaces = rule[1].(bool)
		localProductions = sequence{"SKIPSPACES", doSkipSpaces}
	} else {
		panic(fmt.Sprintf("invalid rule in applies() function: %#q", r1))
	}

	if localProductions == nil { // REMOVE LATER
		localProductions = sequence{}
	}

	if len(localProductions) == 1 {
		return localProductions[0]
	}
	return localProductions
}

func (gp *grammarParser) parseWithGrammarInternal(test string) bool {
	gp.newIdents = gp.newIdents[:0]
	gp.newGrammar.ididx = gp.newGrammar.ididx[:0]
	gp.newGrammar.productions = gp.newGrammar.productions[:0]
	// ep.extras = ep.extras[:0]

	gp.src = []rune(test)
	gp.sdx = 0
	var res object
	if len(gp.grammar.productions) > 0 {
		// ORIGINAL CALL:
		// res = gp.applies(gp.grammar.productions[0][2].(sequence))

		// newProduction, ok := gp.applies(gp.grammar.productions[0][2].(sequence), true, 0)

		res = gp.applies(gp.grammar.productions[0][2].(sequence), true, 0)

		pprint("productions of new grammar", res)

		// res = ok
	}
	gp.skipSpaces()
	if gp.sdx < len(gp.src) {
		res = nil
	}

	return res != nil
}

func (gp *grammarParser) printTrace(rule sequence, action string, depth int) {
	traceEnabled := false // TODO: CHANGE THIS WHEN DEBUGGING
	// traceEnabled := true // TODO: CHANGE THIS WHEN DEBUGGING

	d := ">"
	for i := 0; i < depth; i++ {
		d += ">"
	}

	if traceEnabled {
		c := '-'
		if gp.sdx < len(gp.src) {
			c = gp.src[gp.sdx]
		}
		pprint(fmt.Sprintf("%3d%s rule for pos # %d (%c) action: %s", depth, d, gp.sdx, c, action), rule)
	}
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
