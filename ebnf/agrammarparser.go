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

func GetProductions(aGrammar *r.Rules) *r.Rules {
	for i := range *aGrammar {
		rule := (*aGrammar)[i]
		if rule.Operator == r.Sequence {
			return rule.Childs
		}
	}
	return nil
}

func GetStartRule(aGrammar *r.Rules) *r.Rule {
	for i := range *aGrammar {
		rule := (*aGrammar)[i]
		if rule.Operator == r.Ident {
			return rule
		}
	}
	return nil
}

func GetProlog(aGrammar *r.Rules) *r.Rule {
	for i := range *aGrammar {
		rule := (*aGrammar)[i]
		if rule.Operator == r.Sequence {
			return nil
		} else if rule.Operator == r.Tag {
			return rule
		}
	}
	return nil
}

func GetEpilog(aGrammar *r.Rules) *r.Rule {
	afterProductions := false
	for i := range *aGrammar {
		rule := (*aGrammar)[i]
		if rule.Operator == r.Sequence {
			afterProductions = true
		} else if rule.Operator == r.Tag {
			if afterProductions {
				return rule
			}
		}
	}
	return nil
}

type grammarParser struct {
	src         []rune
	ch          rune
	sdx         int
	grammar     *r.Rules
	productions *r.Rules

	blockList    map[string]bool
	foundList    map[string]*r.Rules
	foundSdxList map[string]int
	foundChList  map[string]rune
	useFoundList bool

	lastParsePosition int

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

func (gp *grammarParser) getRulePosId(rule *r.Rule, pos int) string {
	return fmt.Sprintf("%d:%d", rule.ID, pos)
}

func (gp *grammarParser) ruleEnter(rule *r.Rule, doSkipSpaces bool, depth int) (bool, *r.Rules, int, rune) { // => (isBlocked, rules, rulesSdx, rulesCh)
	gp.traceCount++
	if rule.ID == 0 {
		rule.ID = gp.traceCount
	}
	id := gp.getRulePosId(rule, gp.sdx)

	var isBlocked bool = false
	var foundRule *r.Rules = nil
	var foundSdx int = -1
	var foundCh rune = 0

	if gp.useFoundList {
		foundRule = gp.foundList[id]
		if foundRule != nil { // Is really only useful on massive backtracking...
			foundSdx = gp.foundSdxList[id]
			foundCh = gp.foundChList[id]
			isBlocked = true // If the result is there already, block the apply() from trying it again.
		}
	}
	if !isBlocked {
		isBlocked = gp.blockList[id] // True because in this case, the current rule on the current position in the text to parse is its own parent (= loop).
		// TIODO: Maybe only block at idents:
		if !isBlocked && rule.Operator != r.Or { // Could r.Or in r.Or's loop forever? No because only r.Ident's can create loops and they would be marked. But the result could be found very late because the r.Or can get stuck for a long time if there is another r.Or as child (it would not get blocked from list of the first r.Or options).
			gp.blockList[id] = true // Enter the current rule rule in the block list because it was not already blocked.
		}
	}

	if !gp.traceEnabled {
		return isBlocked, foundRule, foundSdx, foundCh
	}

	c := "EOF"
	if gp.sdx < len(gp.src) {
		c = fmt.Sprintf("%q", gp.src[gp.sdx])
	}
	space := times(" ", depth)
	msg := ""
	if foundRule != nil {
		msg = "\n" + space + "<INSTANT EXIT (ALREADY FOUND)\n"
	} else if isBlocked {
		msg = "\n" + space + "<INSTANT EXIT (LOOP)\n"
	}
	skip := "  spaces:〰️  " // Read spaces.
	if doSkipSpaces {
		skip = "  spaces:➰  " // Skip spaces.
	}
	fmt.Print(space, ">", depth, "  (", gp.traceCount, ")  ", LinePosFromStrPos(string(gp.src), gp.sdx), "  char:", c, skip, rule.ID, ":", PprintRuleOnly(rule, ""), msg, "\n")
	return isBlocked, foundRule, foundSdx, foundCh
}

func (gp *grammarParser) ruleExit(rule *r.Rule, doSkipSpaces bool, depth int, found *r.Rules, pos int) {
	gp.traceCount++
	id := gp.getRulePosId(rule, pos)

	gp.blockList[id] = false             // Exit of the rule. It must be unblocked so it can be called again from a parent.
	if gp.useFoundList && found != nil { // TODO: Make this configurable. On most EBNFs it is not necessary and comes with huge time and memory impact.
		gp.foundList[id] = found
		gp.foundSdxList[id] = gp.sdx
		gp.foundChList[id] = gp.ch
	}

	if !gp.traceEnabled {
		return
	}
	c := "EOF"
	if gp.sdx < len(gp.src) {
		c = fmt.Sprintf("%q", gp.src[gp.sdx])
	}
	skip := "  spaces:〰️  " // Read spaces.
	if doSkipSpaces {
		skip = "  spaces:➰  " // Skip spaces.
	}
	fmt.Print(times(" ", depth), "<", depth, "  (", gp.traceCount, ")  ", LinePosFromStrPos(string(gp.src), gp.sdx), "  char:", c, skip, rule.ID, ":", PprintRuleOnly(rule, ""), " found:", found != nil, "\n")
}

// This is only a helper for apply() and does not need ruleEnter() and ruleExit().
func (gp *grammarParser) applyChildSequence(rules *r.Rules, doSkipSpaces bool, depth int, pos int) *r.Rules {
	var newProductions *r.Rules
	if len(*rules) == 1 {
		newProductions = gp.apply((*rules)[0], doSkipSpaces, depth)
	} else {
		newRule := &r.Rule{Operator: r.Sequence, Childs: rules, Pos: pos} // Creating a new rule means that the rule.ID will be new. That does not matter because there is always a parent that has a known rule.ID.
		newProductions = gp.apply(newRule, doSkipSpaces, depth)
	}
	return newProductions
}

// Apply uses the rules recursive.
func (gp *grammarParser) apply(rule *r.Rule, doSkipSpaces bool, depth int) *r.Rules { // => (localProductions)
	wasSdx := gp.sdx // Start position of the rule. Return, if the rule does not match.
	localProductions := &r.Rules{}

	isBlocked, foundRule, foundSdx, foundCh := gp.ruleEnter(rule, doSkipSpaces, depth)
	if isBlocked {
		if foundRule != nil {
			gp.sdx = foundSdx
			gp.ch = foundCh
			gp.ruleExit(rule, doSkipSpaces, depth, foundRule, wasSdx)
			return foundRule
		}
		gp.ruleExit(rule, doSkipSpaces, depth, nil, wasSdx)
		return nil
	}

	switch rule.Operator {
	case r.Sequence, r.Group, r.Production: // Those are groups/sequences of rules. Iterate through them and apply.
		for i := 0; i < len(*rule.Childs); i++ {
			newProductions := gp.apply((*rule.Childs)[i], doSkipSpaces, depth+1)
			if newProductions == nil {
				gp.ruleExit(rule, doSkipSpaces, depth, nil, wasSdx)
				gp.sdx = wasSdx
				return nil
			} else if len(*newProductions) > 0 && (*newProductions)[0].Operator == r.SkipSpace { // this has to be handled in a sequence
				doSkipSpaces = (*newProductions)[0].Bool
				continue
			}
			if rule.Operator == r.Sequence {
				localProductions = r.AppendArrayOfPossibleSequences(localProductions, newProductions) // Only r.Sequence can be flattened fully
			} else {
				localProductions = r.AppendPossibleSequence(localProductions, &r.Rule{Operator: r.Group, Childs: newProductions, Pos: gp.sdx}) // TODO Childs: (*newProductions) is faster when the pointer is used. Change!
			}
		}
	case r.Token:
		if doSkipSpaces { // There can be white space in strings/text. Do not skip that.
			gp.skipSpaces()
		}
		text := []rune(rule.String)
		for i := 0; i < len(text); i++ {
			if gp.sdx >= len(gp.src) || gp.src[gp.sdx] != text[i] {
				gp.ruleExit(rule, doSkipSpaces, depth, nil, wasSdx)
				gp.sdx = wasSdx
				return nil
			}
			gp.sdx++
		}
		*localProductions = append(*localProductions, rule)
	case r.Range:
		if doSkipSpaces { // There can be white space in strings/text. Do not skip that.
			gp.skipSpaces()
		}
		a := []rune((*rule.Childs)[0].String)[0]
		b := []rune((*rule.Childs)[1].String)[0]
		ch := gp.src[gp.sdx]
		if !(ch >= a && ch <= b) {
			gp.ruleExit(rule, doSkipSpaces, depth, nil, wasSdx)
			gp.sdx = wasSdx
			return nil
		}
		*localProductions = append(*localProductions, &r.Rule{Operator: r.Token, String: fmt.Sprintf("%c", ch)})
	case r.Or:
		found := false
		for i := 0; i < len(*rule.Childs); i++ {
			newProductions := gp.apply((*rule.Childs)[i], doSkipSpaces, depth+1)
			if newProductions != nil { // The nil result is used as ERROR. So if a match is successful but has nothing to return, it should only return something empty but not nil.
				localProductions = r.AppendArrayOfPossibleSequences(localProductions, newProductions)
				found = true
				break
			}
		}
		if !found {
			gp.ruleExit(rule, doSkipSpaces, depth, nil, wasSdx)
			gp.sdx = wasSdx
			return nil
		}
	case r.Repeat:
		var newRule *r.Rule
		if len(*rule.Childs) == 1 {
			newRule = (*rule.Childs)[0]
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
		newProductions := gp.applyChildSequence((*gp.productions)[rule.Int].Childs, doSkipSpaces, depth+1, rule.Pos)
		if newProductions == nil {
			gp.ruleExit(rule, doSkipSpaces, depth, nil, wasSdx)
			gp.sdx = wasSdx
			return nil
		}
		localProductions = r.AppendArrayOfPossibleSequences(localProductions, newProductions)
	case r.Tag:
		newProductions := gp.applyChildSequence(rule.Childs, doSkipSpaces, depth+1, rule.Pos)
		if newProductions == nil {
			gp.ruleExit(rule, doSkipSpaces, depth, nil, wasSdx)
			return nil
		}
		*localProductions = append(*localProductions, &r.Rule{Operator: r.Tag, TagChilds: rule.TagChilds, Childs: newProductions, Pos: gp.sdx})
	case r.SkipSpace: // TODO: Modify SKIPSPACES so that the chars to skip can be given to the command. e.g.: {"SKIPSPACES", "\n\t :;"}
		rule.Pos = gp.sdx
		localProductions = &r.Rules{rule} // Put the responsibility for skip spaces to the parent rule (the caller), because only the parent can change its own doSkipSpaces mode.
	default: // r.Factor || r.Error
		gp.ruleExit(rule, doSkipSpaces, depth, nil, wasSdx)
		panic(fmt.Sprintf("Invalid rule in applies() function: %#v", rule))
	}

	// All failed matches should have returned already.
	// Here must only be matches which means, localProductions MUST NOT be nil here.

	if gp.sdx > gp.lastParsePosition {
		gp.lastParsePosition = gp.sdx
	}

	if len(*localProductions) == 1 {
		if (*localProductions)[0].Operator == r.Group || (*localProductions)[0].Operator == r.Sequence {
			localProductions = (*localProductions)[0].Childs
		}
	}

	gp.ruleExit(rule, doSkipSpaces, depth, localProductions, wasSdx)
	return localProductions
}

func mergeTerminals(productions *r.Rules) {
	lastWasTerminal := false
	for i := 0; i < len(*productions); i++ {
		if (*productions)[i].Operator == r.Token {
			if lastWasTerminal {
				(*productions)[i-1].String += (*productions)[i].String
				*productions = append((*productions)[0:i], (*productions)[i+1:]...)
				i--
			} else {
				(*productions)[i] = &r.Rule{Operator: r.Token, String: (*productions)[i].String, Pos: (*productions)[i].Pos} // Create a copy of the Token to be able to change it.
				lastWasTerminal = true
			}
		} else {
			lastWasTerminal = false
			if len(*(*productions)[i].Childs) > 0 {
				mergeTerminals((*productions)[i].Childs)
			}
		}
	}
}

func ParseWithGrammar(grammar *r.Rules, srcCode string, useFoundList bool, traceEnabled bool) (res *r.Rules, e error) { // => (productions, error)
	// defer func() {
	// 	if err := recover(); err != nil {
	// 		res = nil
	// 		e = fmt.Errorf("%s", err)
	// 	}
	// }()

	var gp grammarParser
	gp.grammar = grammar
	gp.src = []rune(srcCode)
	gp.sdx = 0
	gp.traceEnabled = traceEnabled
	gp.traceCount = 0
	gp.blockList = make(map[string]bool)
	gp.foundList = make(map[string]*r.Rules)
	gp.foundSdxList = make(map[string]int)
	gp.foundChList = make(map[string]rune)
	gp.useFoundList = useFoundList
	gp.lastParsePosition = 0

	gp.productions = GetProductions(gp.grammar)

	if gp.productions == nil || len(*gp.productions) <= 0 {
		return nil, fmt.Errorf("No productions to parse")
	}

	// fmt.Println(PprintRulesFlat(gp.grammar))
	// os.Exit(0)

	startRule := GetStartRule(gp.grammar)
	if startRule == nil {
		return nil, fmt.Errorf("No start rule defined")
	}

	newProductions := gp.apply((*gp.productions)[startRule.Int], true, 0)

	// // TODO: only for testing! Multiple launches
	// gp.sdx = 0
	// gp.ch = 0
	// gp.traceCount = 0
	// newProductions = gp.apply(&gp.grammar.Productions[0], true, 0)

	// Check if the position is at EOF at end of parsing. There can be spaces left, but otherwise its an error:
	gp.skipSpaces()
	if gp.sdx < len(gp.src) {
		return nil, fmt.Errorf("Not everything could be parsed. Last good parse position was %s", LinePosFromStrPos(string(gp.src), gp.lastParsePosition))
	}

	// gp.blockList = nil
	// gp.foundList = nil
	// gp.foundSdxList = nil
	// gp.foundChList = nil
	// gp.src = nil

	mergeTerminals(newProductions)
	return newProductions, nil
}
