package abnf

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"./r"
)

// TODO: switch to "text/scanner".

func RuneToStr(r rune) string {
	var utf8ChBuf [utf8.UTFMax]byte

	if r > utf8.MaxRune {
		r = utf8.RuneError
	}
	buf := utf8ChBuf[:0]
	w := utf8.EncodeRune(buf[:utf8.UTFMax], r)
	return string(buf[:w])
}

// ----------------------------------------------------------------------------
// Dynamic grammar parser
//
// Idea partially taken from: https://rosettacode.org/wiki/Parse_EBNF

type agrammarParser struct {
	src         []rune
	ch          rune
	sdx         int
	agrammar    *r.Rules
	productions *r.Rules

	blockList    map[int]bool
	useBlockList bool
	foundList    map[int]*r.Rules
	foundSdxList map[int]int
	foundChList  map[int]rune
	useFoundList bool

	spaces string

	lastParsePosition int

	rangeCache [256]*r.Rule

	traceEnabled bool
	traceCount   int
}

func (gp *agrammarParser) skipSpaces(spaces string) {
	for {
		if gp.sdx >= len(gp.src) {
			break
		}
		gp.ch = gp.src[gp.sdx]
		if strings.IndexRune(spaces, gp.ch) == -1 {
			break
		}
		gp.sdx++
	}
}

func (gp *agrammarParser) getRulePosId(rule *r.Rule, pos int) int {
	// Cantor pairing function
	a := rule.Int
	ab := pos + rule.Int
	return ((ab * (ab + 1)) >> 1) + a
}

func (gp *agrammarParser) ruleEnter(rule *r.Rule, doSkipSpaces string, depth int) (bool, *r.Rules, int, rune) { // => (isBlocked, rules, rulesSdx, rulesCh)
	var isBlocked bool = false
	var foundRule *r.Rules = nil
	var foundSdx int = -1
	var foundCh rune = 0
	gp.traceCount++

	if rule.Operator == r.Identifier && (gp.useBlockList || gp.useFoundList) {
		id := gp.getRulePosId(rule, gp.sdx)

		if gp.useFoundList {
			foundRule = gp.foundList[id]
			if foundRule != nil { // Is really only useful on massive backtracking...
				foundSdx = gp.foundSdxList[id]
				foundCh = gp.foundChList[id]
				isBlocked = true // If the result is there already, block the apply() from trying it again.
			}
		}
		if !isBlocked && gp.useBlockList {
			isBlocked = gp.blockList[id] // True if loop, because in this case, the current rule on the current position in the text to parse is its own parent (= loop).
			// TIODO: Maybe only block at idents:
			// if !isBlocked && rule.Operator != r.Or { // Could r.Or in r.Or's loop forever? No because only r.Ident's can create loops and they would be marked. But the result could be found very late because the r.Or can get stuck for a long time if there is another r.Or as child (it would not get blocked from list of the first r.Or options).
			gp.blockList[id] = true // Enter the current rule rule in the block list because it was not already blocked.
			// }
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
	if doSkipSpaces != "" {
		skip = "  spaces:➰  " // Skip spaces.
	}
	fmt.Print(space, ">", depth, "  (", gp.traceCount, ")  ", LinePosFromStrPos(string(gp.src), gp.sdx), "  char:", c, skip, PprintRuleOnly(rule, ""), msg, "\n")
	return isBlocked, foundRule, foundSdx, foundCh
}

func (gp *agrammarParser) ruleExit(rule *r.Rule, doSkipSpaces string, depth int, found *r.Rules, pos int) {
	if rule.Operator == r.Identifier && (gp.useBlockList || gp.useFoundList) {
		id := gp.getRulePosId(rule, pos)
		if gp.useBlockList {
			gp.blockList[id] = false // Exit of the rule. It must be unblocked so it can be called again from a parent.
		}
		if gp.useFoundList && found != nil { // TODO: Make this configurable. On most EBNFs it is not necessary and comes with huge time and memory impact.
			gp.foundList[id] = found
			gp.foundSdxList[id] = gp.sdx
			gp.foundChList[id] = gp.ch
		}
	}

	if !gp.traceEnabled {
		return
	}
	gp.traceCount++

	c := "EOF"
	if gp.sdx < len(gp.src) {
		c = fmt.Sprintf("%q", gp.src[gp.sdx])
	}
	skip := "  spaces:〰️  " // Read spaces.
	if doSkipSpaces != "" {
		skip = "  spaces:➰  " // Skip spaces.
	}
	fmt.Print(times(" ", depth), "<", depth, "  (", gp.traceCount, ")  ", LinePosFromStrPos(string(gp.src), gp.sdx), "  char:", c, skip, PprintRuleOnly(rule, ""), " found:", found != nil, "\n")
}

// This is only a helper for apply() and does not need ruleEnter() and ruleExit().
func (gp *agrammarParser) applyAsSequence(rules *r.Rules, doSkipSpaces string, depth int, pos int) *r.Rules {
	if rules == nil {
		return nil
	}
	var newProductions *r.Rules // = nil
	if len(*rules) == 1 {
		newProductions = gp.apply((*rules)[0], doSkipSpaces, depth)
	} else {
		newRule := &r.Rule{Operator: r.Sequence, Childs: rules, Pos: pos}
		newProductions = gp.apply(newRule, doSkipSpaces, depth)
	}
	return newProductions
}

func (gp *agrammarParser) resolveRulesToToken(rules *r.Rules) *r.Rules {
	if rules == nil {
		return nil
	}
	newProductions := &r.Rules{} // = nil

	for _, rule := range *rules {
		switch rule.Operator {
		case r.Token:
			*newProductions = append(*newProductions, rule)
		case r.Identifier:
			newProductions = r.AppendArrayOfPossibleSequences(newProductions, gp.resolveRulesToToken((*gp.productions)[rule.Int].Childs))
		default:
			if rule.Childs != nil && len(*rule.Childs) > 0 {
				newProductions = r.AppendArrayOfPossibleSequences(newProductions, gp.resolveRulesToToken(rule.Childs))
				continue
			}
			panic("Only Token and Identifier of Token (also as Sequence) are allowed as parameter. Found " + rule.Serialize())
		}
	}

	return newProductions
}

func (gp *agrammarParser) applyCommand(rule *r.Rule) {
	rule.CodeChilds = gp.resolveRulesToToken(rule.CodeChilds)
	switch rule.String {
	case "skip":
		if rule.CodeChilds != nil && len(*rule.CodeChilds) > 0 {
			gp.spaces = (*rule.CodeChilds)[0].String
		} else {
			gp.spaces = ""
		}
	case "include":
	default:
		panic("Unknown command :'" + rule.String + "()'")
	}
}

// Apply uses the rules top down and recursively.
// This is the resolution process of the agrammar. Does the localProductions need to go into a Group or something? No. The grouping was done already in the agrammar.
// At this point, the only grouping is done by Tags. The rest can stay in flat Sequences or arrays of rules.
func (gp *agrammarParser) apply(rule *r.Rule, doSkipSpaces string, depth int) *r.Rules { // => (localProductions)
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
	outer:
		for i := range *rule.Childs {
			newProductions := gp.apply((*rule.Childs)[i], doSkipSpaces, depth+1)
			if newProductions == nil {
				gp.ruleExit(rule, doSkipSpaces, depth, nil, wasSdx)
				gp.sdx = wasSdx
				return nil
			} else if len(*newProductions) > 0 { // Commands have to be handled inside the sequence.
				doContinue := false
				for _, prod := range *newProductions { // There should be only one command but just in case for future upgradeability.
					if prod.Operator == r.Command {
						// Resolve parameter variables.
						prod.CodeChilds = gp.resolveRulesToToken(prod.CodeChilds)
						switch prod.String {
						case "skip":
							if prod.CodeChilds != nil && len(*prod.CodeChilds) > 0 {
								doSkipSpaces = (*prod.CodeChilds)[0].String
							} else {
								doSkipSpaces = ""
							}
						case "include":
						default:
							panic("Unknown command :'" + prod.String + "()'")
						}
						doContinue = true
					} else if prod.Operator == r.SkipSpace { // TODO: Remove after :skip() is implemented.
						skip := prod.Bool
						if skip {
							doSkipSpaces = " \t\r\n"
						} else {
							doSkipSpaces = ""
						}
						doContinue = true
					}
				}
				if doContinue { // TODO: If it not a command, add the result to the localProductions.
					continue outer
				}
			}

			// During parsing, the only grouping is done by Tags. The rest can stay in flat Sequences or arrays of rules. Text could be combined.. maybe in a []byte buffer like unescape.
			localProductions = r.AppendArrayOfPossibleSequences(localProductions, newProductions) // Only r.Sequence can be flattened fully

			// if rule.Operator == r.Sequence {
			// 	localProductions = r.AppendArrayOfPossibleSequences(localProductions, newProductions) // Only r.Sequence can be flattened fully
			// } else {
			// 	// The compiler does not use groups. It could still stay inside a group to signify that it belongs together:
			// 	// localProductions = r.AppendPossibleSequence(localProductions, &r.Rule{Operator: r.Group, Childs: newProductions, Pos: gp.sdx})
			// 	// However, at this point, the only grouping is done by Tags. The rest can stay in flat Sequences or arrays of rules.
			// 	localProductions = r.AppendPossibleSequence(localProductions, &r.Rule{Operator: r.Sequence, Childs: newProductions, Pos: gp.sdx})
			// }
		}
	case r.Token:
		// Only skip spaces when actually reading from the target text (Tokens)
		gp.skipSpaces(doSkipSpaces)
		text := []rune(rule.String)
		for i := range text {
			if gp.sdx >= len(gp.src) || gp.src[gp.sdx] != text[i] {
				gp.ruleExit(rule, doSkipSpaces, depth, nil, wasSdx)
				gp.sdx = wasSdx
				return nil
			}
			gp.sdx++
		}
		*localProductions = append(*localProductions, rule)
	case r.Range:
		// Only skip spaces when actually reading from the target text (Tokens)
		gp.skipSpaces(doSkipSpaces)
		a := rune((*rule.Childs)[0].String[0])
		b := rune((*rule.Childs)[1].String[0])
		if gp.sdx >= len(gp.src) || !(gp.src[gp.sdx] >= a && gp.src[gp.sdx] <= b) {
			gp.ruleExit(rule, doSkipSpaces, depth, nil, wasSdx)
			gp.sdx = wasSdx
			return nil
		}
		ch := gp.src[gp.sdx]
		gp.sdx++
		if ch >= 0 && ch <= 255 { // Cache the ch == 0...255 part of this rules and reuse them.
			if gp.rangeCache[ch] == nil {
				gp.rangeCache[ch] = &r.Rule{Operator: r.Token, String: RuneToStr(ch)}
			}
			*localProductions = append(*localProductions, gp.rangeCache[ch])
		} else {
			*localProductions = append(*localProductions, &r.Rule{Operator: r.Token, String: RuneToStr(ch)})
		}
	case r.Or:
		found := false
		for i := range *rule.Childs {
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
		newProductions := gp.applyAsSequence(rule.Childs, doSkipSpaces, depth+1, rule.Pos)
		localProductions = r.AppendArrayOfPossibleSequences(localProductions, newProductions) // If not all child rules matched, newProductions is nil anyways.
	case r.Identifier: // This identifies another rule (and its index), it is basically a link: E.g. to the expression-rule which is at position 3: { "Identifier", "expression", 3 }
		newProductions := gp.applyAsSequence((*gp.productions)[rule.Int].Childs, doSkipSpaces, depth+1, rule.Pos)
		if newProductions == nil {
			gp.ruleExit(rule, doSkipSpaces, depth, nil, wasSdx)
			gp.sdx = wasSdx
			return nil
		}
		localProductions = r.AppendArrayOfPossibleSequences(localProductions, newProductions)
	case r.Tag:
		newProductions := gp.applyAsSequence(rule.Childs, doSkipSpaces, depth+1, rule.Pos) // TODO: resolveRulesToToken for constants
		if newProductions == nil {
			gp.ruleExit(rule, doSkipSpaces, depth, nil, wasSdx)
			return nil
		}
		*localProductions = append(*localProductions, &r.Rule{Operator: r.Tag, CodeChilds: rule.CodeChilds, Childs: newProductions, Pos: gp.sdx})
	case r.SkipSpace, r.Command: // TODO: Modify SKIPSPACES so that the chars to skip can be given to the command. e.g.: {"SKIPSPACES", "\n\t :;"}
		rule.Pos = gp.sdx
		localProductions = &r.Rules{rule} // Put the responsibility for the Command to the parent rule (the caller), because only the parent can change some options like its own doSkipSpaces mode.
	default: // r.Success || r.Error
		gp.ruleExit(rule, doSkipSpaces, depth, nil, wasSdx)
		panic(fmt.Sprintf("Invalid rule in apply() function: %s", PprintRuleOnly(rule)))
	}

	// All failed matches should have returned already.
	// Here must only be matches which means, localProductions MUST NOT be nil here.

	if gp.sdx > gp.lastParsePosition {
		gp.lastParsePosition = gp.sdx
	}

	if len(*localProductions) == 1 {
		// if (*localProductions)[0].Operator == r.Group || (*localProductions)[0].Operator == r.Sequence { // If Groups are used, break them too here.
		if (*localProductions)[0].Operator == r.Sequence { // There should be only Sequences, Tags or Token left. Everything should be as flat as possible. Sequences can sometimes be simplified further.
			localProductions = (*localProductions)[0].Childs
		}
	}

	gp.ruleExit(rule, doSkipSpaces, depth, localProductions, wasSdx)
	return localProductions
}

func mergeTerminals(productions *r.Rules) {
	if productions == nil {
		return
	}
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

func ParseWithAgrammar(agrammar *r.Rules, srcCode string, useBlockList bool, useFoundList bool, traceEnabled bool) (res *r.Rules, e error) { // => (productions, error)
	// defer func() {
	// 	if err := recover(); err != nil {
	// 		res = nil
	// 		e = fmt.Errorf("%s", err)
	// 	}
	// }()

	var gp agrammarParser
	gp.agrammar = agrammar
	gp.src = []rune(srcCode)
	gp.sdx = 0
	gp.traceEnabled = traceEnabled
	gp.traceCount = 0
	gp.blockList = make(map[int]bool)
	gp.useBlockList = useBlockList
	gp.foundList = make(map[int]*r.Rules)
	gp.foundSdxList = make(map[int]int)
	gp.foundChList = make(map[int]rune)
	gp.useFoundList = useFoundList
	gp.lastParsePosition = 0

	gp.productions = r.GetProductions(gp.agrammar)

	gp.spaces = " \t\r\n" // TODO: Make this configurable via JS.

	var newProductions *r.Rules
	if !(gp.productions == nil || len(*gp.productions) <= 0) {
		startRule := r.GetStartRule(gp.agrammar)
		if startRule == nil || startRule.Int >= len(*gp.productions) || startRule.Int < 0 {
			return nil, fmt.Errorf("No valid start rule defined")
		}

		for _, rule := range *gp.productions {
			if rule.Operator == r.Command {
				gp.applyCommand(rule)
			}
		}

		// For the parsing, the start rule is necessary. For the compilation not.
		newProductions = gp.apply((*gp.productions)[startRule.Int], gp.spaces, 0)
	}

	// Check if the position is at EOF at end of parsing. There can be spaces left, but otherwise its an error:
	gp.skipSpaces(gp.spaces)
	if gp.sdx < len(gp.src) {
		return nil, fmt.Errorf("Not everything could be parsed. Last good parse position was %s", LinePosFromStrPos(string(gp.src), gp.lastParsePosition))
	}

	mergeTerminals(newProductions)
	return newProductions, nil
}
