package ebnf

import (
	"fmt"
	"strings"

	"./r"
)

// ----------------------------------------------------------------------------
// Dynamic grammar parser
//
// Idea partially taken from: https://rosettacode.org/wiki/Parse_EBNF

type grammarParser struct {
	src     []rune
	ch      rune
	sdx     int
	grammar Grammar

	blockList map[string]bool

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

func times(s string, n int) string {
	res := s
	for ; n > 0; n-- {
		res = res + s
	}
	return res
}

func (gp *grammarParser) getRuleId(rule *r.Rule, pos int) string {
	return fmt.Sprintf("%d:%d", rule.ID, pos)
}

func (gp *grammarParser) ruleEnter(rule *r.Rule, doSkipSpaces bool, depth int) bool {
	gp.traceCount++
	if rule.ID == 0 {
		rule.ID = gp.traceCount
	}

	blocked := gp.blockList[gp.getRuleId(rule, gp.sdx)] // True if the current rule on the current position in the text to parse is its own parent (= loop).
	if rule.Operator != r.Or {                          // Could r.Or in r.Or's loop forever? No because only r.Ident's can create loops and they would be marked. But the result could be found very late because the r.Or can get stuck for a long time if there is another r.Or as child (it would not get blocked from list of the first r.Or options).
		gp.blockList[gp.getRuleId(rule, gp.sdx)] = true // Entry of the rule. Block the current rule in the block list.
	}

	if !gp.traceEnabled {
		return !blocked
	}
	c := "EOF"
	if gp.sdx < len(gp.src) {
		c = fmt.Sprintf("%q", gp.src[gp.sdx])
	}
	space := times(" ", depth)
	msg := ""
	if blocked {
		msg = "\n" + space + "<INSTANT EXIT (LOOP)"
	}
	skip := "  noskip  "
	if doSkipSpaces {
		skip = "  skip  "
	}
	fmt.Print(space, ">", depth, "  (", gp.traceCount, ")  pos:", gp.sdx, "  char:", c, skip, rule.ID, ":", PprintRuleOnly(rule, ""), msg, "\n")
	return !blocked
}

func (gp *grammarParser) ruleExit(rule *r.Rule, doSkipSpaces bool, depth int, found bool, pos int) {
	gp.traceCount++
	gp.blockList[gp.getRuleId(rule, pos)] = false // Exit of the rule. It must be unblocked so it can be called again from a parent.

	if !gp.traceEnabled {
		return
	}
	c := "EOF"
	if gp.sdx < len(gp.src) {
		c = fmt.Sprintf("%q", gp.src[gp.sdx])
	}
	skip := "  noskip  "
	if doSkipSpaces {
		skip = "  skip  "
	}
	fmt.Print(times(" ", depth), "<", depth, "  (", gp.traceCount, ")  pos:", gp.sdx, "  char:", c, skip, rule.ID, ":", PprintRuleOnly(rule, ""), " found:", found, "\n")
}

func (gp *grammarParser) applyChildSequence(rules []r.Rule, doSkipSpaces bool, depth int, pos int) []r.Rule {
	newProductions := []r.Rule{}
	if len(rules) == 1 {
		newProductions = gp.apply(&rules[0], doSkipSpaces, depth)
	} else {
		newRule := &r.Rule{Operator: r.Sequence, Childs: rules, Pos: pos} // Creating a new rule means that the rule.ID will be new. That does not matter because there is always a parent that has a known rule.ID.
		newProductions = gp.apply(newRule, doSkipSpaces, depth)
	}
	return newProductions
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
// // ident == name             //  <=  identifies another rule (== address of the other rule)
// // string == token == terminal == text
// // or == alternative
//
// Apply uses the rules recursive.
func (gp *grammarParser) apply(rule *r.Rule, doSkipSpaces bool, depth int) []r.Rule { // => (localProductions)
	wasSdx := gp.sdx // Start position of the rule. Return, if the rule does not match.
	var localProductions []r.Rule = nil

	if !gp.ruleEnter(rule, doSkipSpaces, depth) {
		gp.ruleExit(rule, doSkipSpaces, depth, false, wasSdx)
		return nil
	}

	switch rule.Operator {
	case r.Sequence, r.Group, r.Production: // Those are groups/sequences of rules. Iterate through them and apply.
		for i := 0; i < len(rule.Childs); i++ {
			newProductions := gp.apply(&rule.Childs[i], doSkipSpaces, depth+1)
			if newProductions == nil {
				gp.ruleExit(rule, doSkipSpaces, depth, false, wasSdx)
				gp.sdx = wasSdx
				return nil
			} else if len(newProductions) > 0 && newProductions[0].Operator == r.SkipSpaces { // this has to be handled in a sequence
				doSkipSpaces = newProductions[0].Bool
				continue
			}
			if rule.Operator == r.Sequence {
				localProductions = r.AppendArrayOfPossibleSequences(localProductions, newProductions) // Only r.Sequence can be flattened fully
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
				gp.ruleExit(rule, doSkipSpaces, depth, false, wasSdx)
				gp.sdx = wasSdx
				return nil
			}
			gp.sdx++
		}
		localProductions = append(localProductions, *rule)
	case r.Or:
		found := false
		for i := 0; i < len(rule.Childs); i++ {
			newProductions := gp.apply(&rule.Childs[i], doSkipSpaces, depth+1)
			if newProductions != nil { // HERE, nil as the result array is used as not found ERROR. So if a match is successful but has nothing to return, it should only return something empty but not nil
				localProductions = r.AppendArrayOfPossibleSequences(localProductions, newProductions)
				found = true
				break // This should shortcut some parts (depth first search). // TODO: make this configurable! Sometimes it might be useful to get all variants.
			}
		}
		if !found {
			gp.ruleExit(rule, doSkipSpaces, depth, false, wasSdx)
			gp.sdx = wasSdx
			return nil
		}
	case r.Repeat:
		var newRule *r.Rule
		if len(rule.Childs) == 1 {
			newRule = &rule.Childs[0]
		} else {
			newRule = &r.Rule{Operator: r.Sequence, Childs: rule.Childs, Pos: rule.Pos}
		}
		for { // Repeat as often as possible.
			newProductions := gp.apply(newRule, doSkipSpaces, depth+1)
			if newProductions == nil {
				break
			}
			localProductions = r.AppendArrayOfPossibleSequences(localProductions, newProductions) // Only append if all child rules matched.
		}
	case r.Optional:
		newProductions := gp.applyChildSequence(rule.Childs, doSkipSpaces, depth+1, rule.Pos)
		localProductions = r.AppendArrayOfPossibleSequences(localProductions, newProductions) // If not all child rules matched, newProductions is nil anyways.
	case r.Ident: // "IDENT" identifies another rule (and its index), it is basically a link: This would e.g. be an "IDENT" to the expression-rule which is at position 3: { "IDENT", "expression", 3 }
		newProductions := gp.applyChildSequence(gp.grammar.Productions[rule.Int].Childs, doSkipSpaces, depth+1, rule.Pos)
		if newProductions == nil {
			gp.ruleExit(rule, doSkipSpaces, depth, false, wasSdx)
			gp.sdx = wasSdx
			return nil
		}
		localProductions = r.AppendArrayOfPossibleSequences(localProductions, newProductions)
	case r.Tag:
		newProductions := gp.applyChildSequence(rule.Childs, doSkipSpaces, depth+1, rule.Pos)
		if newProductions == nil {
			gp.ruleExit(rule, doSkipSpaces, depth, false, wasSdx)
			return nil
		}
		newTag := r.Rule{Operator: r.Tag, TagChilds: rule.TagChilds, Pos: gp.sdx}
		newTag.Childs = r.AppendArrayOfPossibleSequences(newTag.Childs, newProductions)
		localProductions = append(localProductions, newTag)
	case r.SkipSpaces: // TODO: Modify SKIPSPACES so that the chars to skip can be given to the command. e.g.: {"SKIPSPACES", "\n\t :;"}
		rule.Pos = gp.sdx
		gp.ruleExit(rule, doSkipSpaces, depth, true, wasSdx)
		return []r.Rule{*rule} // Put the responsibility for skip spaces to the parent rule (the caller), because only the parent can change its own doSkipSpaces mode.
	default: // r.Factor || r.Invalid
		gp.ruleExit(rule, doSkipSpaces, depth, false, wasSdx)
		panic(fmt.Sprintf("invalid rule in applies() function: %#v", rule))
	}
	gp.ruleExit(rule, doSkipSpaces, depth, true, wasSdx)

	// All failed matches should have returned already.
	// Here must only be matches.

	if len(localProductions) == 1 {
		if localProductions[0].Operator == r.Group || localProductions[0].Operator == r.Sequence {
			localProductions = localProductions[0].Childs
		}
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
	gp.blockList = make(map[string]bool)

	if len(gp.grammar.Productions) <= 0 {
		return nil, fmt.Errorf("No productions to parse")
	}

	newProductions := gp.apply(&gp.grammar.Productions[0], true, 0)
	// Check if the position is at EOF at end of parsing. There can be spaces left, but otherwise its an error:
	gp.skipSpaces()
	if gp.sdx < len(gp.src) {
		return nil, fmt.Errorf("Not everything could be parsed")
	}

	newProductions = mergeTerminals(newProductions)

	return newProductions, nil
}
