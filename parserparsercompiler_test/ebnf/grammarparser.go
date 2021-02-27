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
			if newProduction := gp.applies(rule[i].(sequence), doSkipSpaces, depth+1); newProduction != nil {
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

// ----------------------------------------------------

var variables map[string]object

func Walk(tree object) string {
	if t, ok := tree.(sequence); ok && len(t) > 0 {
		t1 := t[0]

		if _, ok := t1.(string); !ok { // "SEQUENCE" (if there is no string at rule[0], it is a group/sequence of rules. iterate through them and apply)
			res := ""
			for _, o := range t {
				res += Walk(o)
			}
			return res
		} else if t1 == "TERMINAL" {
			return t[1].(string)
		} else if t1 == "TAG" {
			tagAnnotation, ok := t[1].(sequence)
			if !ok || len(tagAnnotation) < 2 || tagAnnotation[0] != "TERMINAL" {
				panic(fmt.Sprintf("error in tree at tag %#v", t))
			}
			tagAnnotationString := tagAnnotation[1].(string)

			res := Walk(t[2])
			variables[tagAnnotationString] = res

			return res
		}

		fmt.Printf("### %#v\n", t1)

	} else {
		// if tree != nil {
		// 	fmt.Printf("### %#v\n", tree)
		// }
	}

	return ""
}

// ----------------------------------------------------

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

		variables = map[string]object{}
		fmt.Printf("\n\n%s\n\n", Walk(res))
		pprint("variables", variables)

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
