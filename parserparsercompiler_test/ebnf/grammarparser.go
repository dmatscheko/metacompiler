package ebnf

import (
	"fmt"
	"strings"
)

// ----------------------------------------------------------------------------
// Dynamic grammar parser

type grammarParser struct {
	src     []rune
	ch      rune
	sdx     int
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
// The rules that apply() has to deal with are BASICALLY THE SAME AS AN BNF-PARSER with annotations (NOT EBNF):
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
func (gp *grammarParser) apply(rule sequence, doSkipSpaces bool, depth int) object {
	wasSdx := gp.sdx // in case of failure
	r1 := rule[0]

	var localProductions sequence
	localProductions = localProductions[:0]

	// gp.printTrace(rule, "A", depth)

	if _, ok := r1.(string); !ok { // "SEQUENCE" (if there is no string at rule[0], it is a group/sequence of rules. iterate through them and apply)
		for i := 0; i < len(rule); i++ {
			newProduction := gp.apply(rule[i].(sequence), doSkipSpaces, depth+1)
			if newProduction == nil {
				gp.sdx = wasSdx
				return nil
			}

			if t, ok := newProduction.(sequence); ok && len(t) > 0 && t[0] == "SKIPSPACES" { // this has to be handled in a sequence
				doSkipSpaces = t[1].(bool)
				continue
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
		found := false
		for i := 1; i < len(rule); i++ {
			if newProduction := gp.apply(rule[i].(sequence), doSkipSpaces, depth+1); newProduction != nil {
				// return newProduction
				localProductions = append(localProductions, newProduction)
				found = true
			}
		}
		if !found {
			gp.sdx = wasSdx
			return nil
		}
	} else if r1 == "REPEAT" {
		for {
			newProduction := gp.apply(rule[1].(sequence), doSkipSpaces, depth+1)
			if newProduction == nil {
				break
			}
			localProductions = append(localProductions, newProduction)
		}
	} else if r1 == "OPTIONAL" {
		newProduction := gp.apply(rule[1].(sequence), doSkipSpaces, depth+1)
		if newProduction != nil {
			localProductions = append(localProductions, newProduction)
		}
	} else if r1 == "IDENT" { // "IDENT" identifies another block (and its index), it is basically a link: This would e.g. be an "IDENT" to the expression-block which is at position 3: { "IDENT", "expression", 3 }
		i := rule[2].(int)
		ii := gp.grammar.ididx[i]
		newProduction := gp.apply(gp.grammar.productions[ii][2].(sequence), doSkipSpaces, depth+1)
		if newProduction == nil {
			gp.sdx = wasSdx
			return nil
		}
		localProductions = append(localProductions, newProduction)
	} else if r1 == "TAG" {
		newProduction := gp.apply(rule[2].(sequence), doSkipSpaces, depth+1)
		if newProduction != nil {
			localProductions = append(localProductions, sequence{rule[0], rule[1], newProduction})
		} else {
			return nil
		}
	} else if r1 == "SKIPSPACES" { // TODO: modify SKIPSPACES so that the chars to skip must be given to the command. e.g.: {"SKIPSPACES", "\n\t :;"}
		return rule
	} else {
		panic(fmt.Sprintf("invalid rule in applies() function: %#q", r1))
	}

	// all failed matches should have returned already
	// here must only be matches

	if len(localProductions) == 1 {
		return localProductions[0]
	}
	if localProductions == nil { // must not be nil because nil is for failed match
		localProductions = sequence{}
	}
	return localProductions
}

func ParseWithGrammar(grammar Grammar, srcCode string) (err error) {
	// defer func() {
	// 	if errRecover := recover(); errRecover != nil {
	// 		success = false
	// 		err = fmt.Errorf(fmt.Sprintf("%s", errRecover))
	// 	}
	// }()

	var gp grammarParser
	gp.grammar = grammar

	gp.src = []rune(srcCode)
	gp.sdx = 0
	if len(gp.grammar.productions) <= 0 {
		return fmt.Errorf("No productions to parse")
	}

	parseTree := gp.apply(gp.grammar.productions[0][2].(sequence), true, 0)
	// pprint("productions of new parse tree (grammar) (whole tree)", parseTree)

	gp.skipSpaces()
	if gp.sdx < len(gp.src) {
		return fmt.Errorf("Not everything could be parsed")
	}

	var co compiler
	return co.CompileParseTree(parseTree)
}
