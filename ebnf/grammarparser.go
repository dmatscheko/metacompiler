package ebnf

import (
	"fmt"
	"strings"

	"./r"
)

// ----------------------------------------------------------------------------
// Dynamic grammar parser

// TODO: REMEMBER WHAT HAS BEEN TRIED ALREADY FOR A POSITION!

type grammarParser struct {
	src     []rune
	ch      rune
	sdx     int
	grammar Grammar

	traceEnabled bool
	traceCount   int
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

func (gp *grammarParser) printTrace(rule r.Rule, doSkipSpaces bool, depth int) {
	if !gp.traceEnabled {
		return
	}

	doSkipSpacesStr := "skipspaces: NO"
	if doSkipSpaces {
		doSkipSpacesStr = "skipspaces: YES"
	}

	d := ">"
	for i := 0; i < depth; i++ {
		d += ">"
	}

	c := '-'
	if gp.sdx < len(gp.src) {
		c = gp.src[gp.sdx]
	}

	gp.traceCount++

	// Pprint(fmt.Sprintf("%4d: %3d>>>> rule for pos # %d (char '%c') %s", gp.traceCount, depth, gp.sdx, c, doSkipSpacesStr), rule)
	Pprint(fmt.Sprintf("%4d: %3d%s rule for pos # %d (char '%c') %s", gp.traceCount, depth, d, gp.sdx, c, doSkipSpacesStr), rule)
	// fmt.Printf("%4d: %3d%s rule for pos # %d (char '%c') %s %s", gp.traceCount, depth, d, gp.sdx, c, doSkipSpacesStr, PprintSequence(&rule, ""))
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
func (gp *grammarParser) apply(rule r.Rule, doSkipSpaces bool, depth int) []r.Rule { // => (localProductions)
	wasSdx := gp.sdx // In case of failure
	var localProductions []r.Rule = nil

	// TODO: FOR DEBUG. Make this configurable:
	// ---------------
	if depth > 1000 {
		panic("ERROR: Too many loops!")
	}
	gp.printTrace(rule, doSkipSpaces, depth)
	// ---------------

	switch rule.Operator {
	case r.Sequence, r.Group, r.Production: // Those are groups/sequences of rules. Iterate through them and apply.
		for i := 0; i < len(rule.Childs); i++ {
			newProductions := gp.apply(rule.Childs[i], doSkipSpaces, depth+1)
			if newProductions == nil {
				gp.sdx = wasSdx
				return nil
			} else if len(newProductions) > 0 && newProductions[0].Operator == r.SkipSpaces { // this has to be handled in a sequence
				doSkipSpaces = newProductions[0].Bool
				continue
			}
			if rule.Operator == r.Sequence {
				localProductions = r.AppendArrayOfPossibleSequences(localProductions, newProductions) // Only r.Basic can be flattened fully
			} else {
				localProductions = r.AppendPossibleSequence(localProductions, r.Rule{Operator: r.Group, Childs: newProductions, Pos: gp.sdx})
			}
		}
	case r.Terminal:
		if doSkipSpaces { // There can be white space in strings/text. Do not skip that.
			gp.skipSpaces()
		}
		text := []rune(rule.String)
		for i := 0; i < len(text); i++ {
			if gp.sdx >= len(gp.src) || gp.src[gp.sdx] != text[i] {
				gp.sdx = wasSdx
				return nil
			}
			gp.sdx++
		}
		localProductions = append(localProductions, rule)
		// Pprint("X", rule)
	case r.Or:
		found := false
		for i := 0; i < len(rule.Childs); i++ {
			newProductions := gp.apply(rule.Childs[i], doSkipSpaces, depth+1)
			if newProductions != nil { // HERE, nil as the result array is used as not found ERROR. So if a match is successful but has nothing to return, it should only return something empty but not nil
				localProductions = r.AppendArrayOfPossibleSequences(localProductions, newProductions)
				found = true
				break // This should shortcut some parts (depth first search). // TODO: make this configurable! Sometimes it might be useful to get all variants.
			}
		}
		if !found {
			gp.sdx = wasSdx
			return nil
		}
	case r.Repeat:
		rule.Operator = r.Sequence
		for { // Repeat as often as possible.
			newProductions := gp.apply(rule, doSkipSpaces, depth+1)
			if newProductions == nil {
				break
			}
			localProductions = r.AppendArrayOfPossibleSequences(localProductions, newProductions) // Only append if all child rules matched.
		}
	case r.Optional:
		rule.Operator = r.Sequence
		newProductions := gp.apply(rule, doSkipSpaces, depth+1)
		localProductions = r.AppendArrayOfPossibleSequences(localProductions, newProductions) // If not all child rules matched, newProductions is nil anyways.
	case r.Ident: // "IDENT" identifies another block (and its index), it is basically a link: This would e.g. be an "IDENT" to the expression-block which is at position 3: { "IDENT", "expression", 3 }
		newRule := r.Rule{Operator: r.Sequence, Childs: gp.grammar.Productions[rule.Int].Childs, Pos: gp.sdx}
		newProductions := gp.apply(newRule, doSkipSpaces, depth+1)
		if newProductions == nil {
			gp.sdx = wasSdx
			return nil
		}
		localProductions = r.AppendArrayOfPossibleSequences(localProductions, newProductions)
	case r.Tag:
		newRule := r.Rule{Operator: r.Sequence, Childs: rule.Childs, Pos: gp.sdx}
		newProductions := gp.apply(newRule, doSkipSpaces, depth+1)
		if newProductions == nil {
			return nil
		}
		newTag := r.Rule{Operator: r.Tag, TagChilds: rule.TagChilds, Pos: gp.sdx}
		newTag.Childs = r.AppendArrayOfPossibleSequences(newTag.Childs, newProductions)
		localProductions = append(localProductions, newTag)
	case r.SkipSpaces: // TODO: modify SKIPSPACES so that the chars to skip must be given to the command. e.g.: {"SKIPSPACES", "\n\t :;"}
		rule.Pos = gp.sdx
		return []r.Rule{rule}
	default: // r.Factor || r.Invalid
		panic(fmt.Sprintf("invalid rule in applies() function: %#v", rule))
	}

	// all failed matches should have returned already
	// here must only be matches

	if len(localProductions) == 1 && localProductions[0].Operator == r.Group {
		localProductions = localProductions[0].Childs
	}
	if localProductions == nil { // Must not be nil because nil is for failed match.
		localProductions = []r.Rule{}
	}
	return localProductions
}

func mergeTerminals(productions []r.Rule) []r.Rule {
	lastWasTerminal := false
	for i := 0; i < len(productions); i++ {
		if productions[i].Operator == r.Terminal {
			if lastWasTerminal {
				productions[i-1].String += productions[i].String
				productions = append(productions[0:i], productions[i+1:]...)
				i--
			} else {
				lastWasTerminal = true
			}
		} else {
			lastWasTerminal = false
			if len(productions[i].Childs) > 0 {
				productions[i].Childs = mergeTerminals(productions[i].Childs)
			}
		}
	}
	return productions
}

func ParseWithGrammar(grammar Grammar, srcCode string, traceEnabled bool) (res []r.Rule, err error) { // => (productions, error)
	defer func() {
		if errRecover := recover(); errRecover != nil {
			res = nil
			err = fmt.Errorf(fmt.Sprintf("%s", errRecover))
		}
	}()

	var gp grammarParser
	gp.grammar = grammar
	gp.src = []rune(srcCode)
	gp.sdx = 0
	gp.traceEnabled = traceEnabled
	gp.traceCount = 0

	if len(gp.grammar.Productions) <= 0 {
		return nil, fmt.Errorf("No productions to parse")
	}

	newProductions := gp.apply(gp.grammar.Productions[0], true, 0)
	gp.skipSpaces()
	if gp.sdx < len(gp.src) {
		return nil, fmt.Errorf("Not everything could be parsed")
	}

	newProductions = mergeTerminals(newProductions)

	return newProductions, nil
}
