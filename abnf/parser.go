package abnf

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"14.gy/mec/abnf/r"
)

// TODO: Maybe switch to "text/scanner" (If it improves performance).

// func RuneToStr(r rune) string {
// 	if r > utf8.MaxRune {
// 		r = utf8.RuneError
// 	}
// 	buf := make([]byte, 0, utf8.UTFMax)
// 	size := utf8.EncodeRune(buf[:utf8.UTFMax], r)
// 	return string(buf[:size])
// }

// ----------------------------------------------------------------------------
// agrammar parser

type parser struct {
	Src      string
	Sdx      int
	agrammar *r.Rules

	blockList    map[int]bool
	useBlockList bool
	foundList    map[int]*r.Rules
	foundSdxList map[int]int
	useFoundList bool

	spaces *r.Rule

	lastParsePosition int

	rangeCache [256]*r.Rule

	traceEnabled bool
	traceCount   int

	ps *parserscript

	fileName string
}

func (pa *parser) getRulePosId(rule *r.Rule, pos int) int {
	// Cantor pairing function
	a := rule.Int
	ab := pos + rule.Int
	return ((ab * (ab + 1)) >> 1) + a
}

func (pa *parser) ruleEnter(rule *r.Rule, skipSpaceRule *r.Rule, depth int) (bool, *r.Rules, int) { // => (isBlocked, rules, rulesSdx, rulesCh)
	var isBlocked bool = false
	var foundRule *r.Rules = nil
	var foundSdx int = -1
	pa.traceCount++

	if rule.Operator == r.Identifier && (pa.useBlockList || pa.useFoundList) {
		id := pa.getRulePosId(rule, pa.Sdx)

		if pa.useFoundList {
			foundRule = pa.foundList[id]
			if foundRule != nil { // Is really only useful on massive backtracking...
				foundSdx = pa.foundSdxList[id]
				isBlocked = true // If the result is there already, block the apply() from trying it again.
			}
		}
		if !isBlocked && pa.useBlockList {
			isBlocked = pa.blockList[id] // True if loop, because in this case, the current rule on the current position in the text to parse is its own parent (= loop).
			// TODO: Maybe only block at identifier:
			// if !isBlocked && rule.Operator != r.Or { // Could r.Or in r.Or's loop forever? No because only r.Ident's can create loops and they would be marked. But the result could be found very late because the r.Or can get stuck for a long time if there is another r.Or as child (it would not get blocked from list of the first r.Or options).
			pa.blockList[id] = true // Enter the current rule rule in the block list because it was not already blocked.
			// }
		}
	}

	if !pa.traceEnabled {
		return isBlocked, foundRule, foundSdx
	}

	c := "EOF"
	if pa.Sdx < len(pa.Src) {
		c = fmt.Sprintf("%q", pa.Src[pa.Sdx])
	}
	space := times(" ", depth)
	msg := ""
	if foundRule != nil {
		msg = "\n" + space + "<INSTANT EXIT (ALREADY FOUND)\n"
	} else if isBlocked {
		msg = "\n" + space + "<INSTANT EXIT (LOOP)\n"
	}
	skip := "  spaces:〰️  " // Read spaces.
	if skipSpaceRule != nil {
		skip = "  spaces:➰  " // Skip spaces.
	}
	fmt.Print(space, ">", depth, "  (", pa.traceCount, ")  ", LinePosFromStrPos(string(pa.Src), pa.Sdx), "  char:", c, skip, rule.ToString(), msg, "\n")
	return isBlocked, foundRule, foundSdx
}

func (pa *parser) ruleExit(rule *r.Rule, skipSpaceRule *r.Rule, depth int, found *r.Rules, wasSdx int) {
	if rule.Operator == r.Identifier && (pa.useBlockList || pa.useFoundList) {
		id := pa.getRulePosId(rule, wasSdx)
		if pa.useBlockList {
			pa.blockList[id] = false // Exit of the rule. It must be unblocked so it can be called again from a parent.
		}
		if pa.useFoundList && found != nil {
			pa.foundList[id] = found
			pa.foundSdxList[id] = pa.Sdx
		}
	}

	if !pa.traceEnabled {
		return
	}
	pa.traceCount++

	c := "EOF"
	if pa.Sdx < len(pa.Src) {
		c = fmt.Sprintf("%q", pa.Src[pa.Sdx])
	}
	skip := "  spaces:〰️  " // Read spaces.
	if skipSpaceRule != nil {
		skip = "  spaces:➰  " // Skip spaces.
	}
	fmt.Print(times(" ", depth), "<", depth, "  (", pa.traceCount, ")  ", LinePosFromStrPos(string(pa.Src), pa.Sdx), "  char:", c, skip, rule.ToString(), " found:", found != nil, "\n")
}

// This is only a helper for apply() and does not need ruleEnter() and ruleExit().
func (pa *parser) applyAsSequence(rules *r.Rules, skipSpaceRule *r.Rule, skipSpaces bool, depth int, pos int) *r.Rules {
	if rules == nil {
		return nil
	}
	var newProductions *r.Rules // = nil
	if len(*rules) == 1 {
		newProductions = pa.apply((*rules)[0], skipSpaceRule, skipSpaces, depth)
	} else {
		newRule := &r.Rule{Operator: r.Sequence, Childs: rules, Pos: pos}
		newProductions = pa.apply(newRule, skipSpaceRule, skipSpaces, depth)
	}
	return newProductions
}

func collectReferencesSlow(rules *r.Rules, references map[string]int) {
	if rules == nil {
		return
	}
	for i, rule := range *rules {
		if rule.Operator != r.Production {
			continue
		}
		references[rule.String] = i
	}
}
func correctReferencesSlow(rules *r.Rules, references map[string]int) {
	if rules == nil {
		return
	}
	if references == nil {
		references = map[string]int{}
		collectReferencesSlow(rules, references)
	}
	for i := 0; i < len(*rules); i++ {
		if (*rules)[i].Operator == r.Identifier {
			ref, ok := references[(*rules)[i].String]
			if !ok {
				panic("Production " + (*rules)[i].String + "not found")
			}
			(*rules)[i].Int = ref
		}
		if (*rules)[i].Childs != nil && len(*(*rules)[i].Childs) > 0 {
			correctReferencesSlow((*rules)[i].Childs, references)
		}
	}
}

// Almost like apply() but without parsing the target text.
func (pa *parser) resolveRulesToToken(rules *r.Rules) *r.Rules {
	if rules == nil {
		return nil
	}
	newProductions := &r.Rules{} // = nil

	for _, rule := range *rules {
		switch rule.Operator {
		case r.Token:
			*newProductions = append(*newProductions, rule)
		case r.Identifier:
			newProductions = r.AppendArrayOfPossibleSequences(newProductions, pa.resolveRulesToToken((*pa.agrammar)[rule.Int].Childs))
		default:
			if rule.Childs != nil && len(*rule.Childs) > 0 {
				newProductions = r.AppendArrayOfPossibleSequences(newProductions, pa.resolveRulesToToken(rule.Childs))
				continue
			}
			panic("Only Token and Identifier of Token (also as Sequence) are allowed as parameter. Found " + rule.Serialize())
		}
	}

	return newProductions
}

func flattenToken(rules *r.Rules) *r.Rule {
	if rules == nil {
		return nil
	}
	if len(*rules) == 1 && (*rules)[0].Operator == r.Token {
		return (*rules)[0]
	}

	length := 0
	for _, rule := range *rules {
		if rule.Operator != r.Token {
			panic("Const must only contain Token. Contains: " + rule.ToString())
		}
		length += len(rule.String)
	}

	buf := make([]byte, 0, length)
	for _, rule := range *rules {
		buf = append(buf, rule.String...)
	}

	return &r.Rule{Operator: r.Token, String: string(buf)}
}

func (pa *parser) resolveParameterToToken(rules *r.Rules) {
	if rules == nil {
		return
	}
	for i := range *rules {
		resRule := flattenToken(pa.resolveRulesToToken(&r.Rules{(*rules)[i]}))
		if resRule == nil {
			panic("Parameter is empty. Rule: " + (*rules)[i].ToString())
		}
		(*rules)[i] = resRule
	}
}

// The global commands (Production level).
// TODO: Maybe remove used commands.
func (pa *parser) applyCommand(rule *r.Rule) {
	switch rule.String {
	case "whitespace":
		// pa.resolveParameterToToken(rule.CodeChilds)
		if rule.CodeChilds != nil && len(*rule.CodeChilds) > 0 {
			pa.spaces = (*rule.CodeChilds)[0]
		} else {
			pa.spaces = nil
		}
	case "include":
		// Resolve parameter constants.
		pa.resolveParameterToToken(rule.CodeChilds)
		if rule.CodeChilds == nil || len(*rule.CodeChilds) == 0 || (*rule.CodeChilds)[0].Operator != r.Token {
			panic("Command :include() needs at least a constant string as file name parameter.")
		}
		if len(*rule.CodeChilds) > 2 {
			panic("Too many parameters for Command :include().")
		}
		paramFileName := (*rule.CodeChilds)[0].String
		slot := 0
		if len(*rule.CodeChilds) == 2 && (*rule.CodeChilds)[1].Operator == r.Number {
			slot = (*rule.CodeChilds)[1].Int
		}
		if paramFileName == "" {
			panic("The file parameter for Command :include() must not be empty.")
		}
		fullFileName := filepath.Dir(pa.fileName) + string(os.PathSeparator) + paramFileName
		dat, err := ioutil.ReadFile(fullFileName)
		if err != nil {
			panic(err)
		}
		srcCode := string(dat)

		asg, err := ParseWithAgrammar(AbnfAgrammar, srcCode, fullFileName, pa.useBlockList, pa.useFoundList, pa.traceEnabled)
		if err != nil {
			panic(err)
		}
		aGrammar, err := CompileASG(asg, AbnfAgrammar, fullFileName, slot, false, false)
		if err != nil {
			panic(err)
		}
		*pa.agrammar = append(*pa.agrammar, *aGrammar...)
		// Correct all references
		correctReferencesSlow(pa.agrammar, nil)
	case "number":
		// :number(4, LE) would mean take 4 bytes from the input (gp.src), interpret them as little endian and create a Number from it. This means it should be usable in Times expressions and should allow the parsing of TLV-formats.
		panic(":number() is only allowed as inline command.")
	case "title":
		// TODO: Maybe use that information.
	case "description":
		// TODO: Maybe use that information.
	case "startRule":
		// This is used by ParseWithAgrammar().
	case "startScript":
		// This is used by ParseWithAgrammar().
	default:
		panic("Unknown initial line command :" + rule.String + "()")
	}
}

// Apply uses the rules top down and recursively.
// This is the resolution process of the agrammar. Does the localProductions need to go into a Group or something? No. The grouping was done already in the agrammar.
// At this point, the only grouping is done by Tags. The rest can stay in flat Sequences or arrays of rules.
// Rules can be reused. So whatever you do, NEVER change a rule here.
// TODO: If in a rule that does not change pa.Sdx, then the rule does not change pa.Sdx back. EXCEPT if the rule is in a loop that has to apply all or multiple elements!
func (pa *parser) apply(rule *r.Rule, skipSpaceRule *r.Rule, skippingSpaces bool, depth int) *r.Rules { // => (localProductions)
	wasSdx := pa.Sdx // Start position of the rule. Return, if the rule does not match.
	localProductions := &r.Rules{}

	isBlocked, foundRule, foundSdx := pa.ruleEnter(rule, skipSpaceRule, depth)
	if isBlocked {
		if foundRule != nil {
			pa.Sdx = foundSdx
			pa.ruleExit(rule, skipSpaceRule, depth, foundRule, wasSdx)
			return foundRule
		}
		pa.ruleExit(rule, skipSpaceRule, depth, nil, wasSdx)
		return nil
	}

	switch rule.Operator {
	case r.Sequence, r.Group, r.Production: // Those are groups/sequences of rules. Iterate through them and apply.
		for i := range *rule.Childs {
			newProductions := pa.apply((*rule.Childs)[i], skipSpaceRule, skippingSpaces, depth+1)
			if newProductions == nil {
				pa.ruleExit(rule, skipSpaceRule, depth, nil, wasSdx)
				pa.Sdx = wasSdx
				return nil
			}
			if len(*newProductions) > 0 { // Some Commands like :skip() have to be handled inside the sequence.
				for _, prod := range *newProductions {
					// The local commands (inside an Expression).
					if prod.Operator == r.Command {
						switch prod.String {
						case "whitespace":
							// Resolve parameter constants.
							// pa.resolveParameterToToken(prod.CodeChilds)
							if prod.CodeChilds != nil && len(*prod.CodeChilds) > 0 {
								skipSpaceRule = (*prod.CodeChilds)[0]
							} else {
								skipSpaceRule = nil
							}
							continue // Don't store the result of the :whitespace() command as Token or Tag-output.
						default:
							// All other commands should have been handled already by apply() and so this should never happen.
							panic("Unknown inline command :" + prod.String + "()'")
						}
					}
					// During parsing, the only grouping is done by Tags. The rest can stay in flat Sequences or arrays of rules. Text could be combined.. maybe in a []byte buffer like unescape.
					localProductions = r.AppendPossibleSequence(localProductions, prod)
				}
			}
		}
	case r.Token:
		// Only skip spaces when actually reading from the target text (Tokens)
		if !skippingSpaces && skipSpaceRule != nil { // Do not skip spaces again when we are already at skipping spaces. Would result in an infinite loop.
			pa.apply(skipSpaceRule, skipSpaceRule, true, depth+1) // Skip spaces.
		}
		size := len(rule.String)
		if pa.Sdx+size > len(pa.Src) || rule.String != pa.Src[pa.Sdx:pa.Sdx+size] {
			pa.ruleExit(rule, skipSpaceRule, depth, nil, wasSdx)
			pa.Sdx = wasSdx
			return nil
		}
		pa.Sdx += size
		if skippingSpaces {
			return &r.Rules{}
		}
		*localProductions = append(*localProductions, rule)
	case r.CharOf:
		// Only skip spaces when actually reading from the target text (Tokens)
		if !skippingSpaces && skipSpaceRule != nil { // Do not skip spaces again when we are already at skipping spaces. Would result in an infinite loop.
			pa.apply(skipSpaceRule, skipSpaceRule, true, depth+1) // Skip spaces.
		}
		if pa.Sdx+1 > len(pa.Src) {
			pa.ruleExit(rule, skipSpaceRule, depth, nil, wasSdx)
			pa.Sdx = wasSdx
			return nil
		}
		ch, size := utf8.DecodeRuneInString(pa.Src[pa.Sdx:])
		if !strings.ContainsRune(rule.String, ch) {
			pa.ruleExit(rule, skipSpaceRule, depth, nil, wasSdx)
			pa.Sdx = wasSdx
			return nil
		}
		pa.Sdx += size
		if skippingSpaces {
			return &r.Rules{}
		}
		if ch >= 0 && ch <= 127 { // Cache the rune == 0...127 part of this rules and reuse them, because those are the most used chars and they are binary compatible with bytes. 128...255 are NOT compatible.
			if pa.rangeCache[ch] == nil {
				pa.rangeCache[ch] = &r.Rule{Operator: r.Token, String: string([]rune{ch})}
			}
			*localProductions = append(*localProductions, pa.rangeCache[ch])
		} else {
			*localProductions = append(*localProductions, &r.Rule{Operator: r.Token, String: string([]rune{ch})})
		}
	case r.CharsOf:
		// Only skip spaces when actually reading from the target text (Tokens)
		if !skippingSpaces && skipSpaceRule != nil { // Do not skip spaces again when we are already at skipping spaces. Would result in an infinite loop.
			pa.apply(skipSpaceRule, skipSpaceRule, true, depth+1) // Skip spaces.
		}
		length := len(pa.Src)
		startPos := pa.Sdx
		for {
			if pa.Sdx >= length {
				break
			}
			ch, size := utf8.DecodeRuneInString(pa.Src[pa.Sdx:])
			if !strings.ContainsRune(rule.String, ch) {
				break
			}
			pa.Sdx += size
		}
		size := pa.Sdx - startPos
		if size == 0 {
			pa.ruleExit(rule, skipSpaceRule, depth, nil, wasSdx)
			pa.Sdx = wasSdx
			return nil
		}
		if skippingSpaces {
			return &r.Rules{}
		}
		*localProductions = append(*localProductions, &r.Rule{Operator: r.Token, String: pa.Src[startPos:pa.Sdx]})
	case r.Range:
		// Only skip spaces when actually reading from the target text (Tokens)
		if !skippingSpaces && skipSpaceRule != nil { // Do not skip spaces again when we are already at skipping spaces. Would result in an infinite loop.
			pa.apply(skipSpaceRule, skipSpaceRule, true, depth+1) // Skip spaces.
		}
		if pa.Sdx >= len(pa.Src) {
			pa.ruleExit(rule, skipSpaceRule, depth, nil, wasSdx)
			pa.Sdx = wasSdx
			return nil
		}
		if rule.Int == r.RangeTypeRune { // Rune range for unicode. JS-Mapping: abnf.rangeType.Rune
			ch, size := utf8.DecodeRuneInString(pa.Src[pa.Sdx:])
			if ch == utf8.RuneError {
				pa.ruleExit(rule, skipSpaceRule, depth, nil, wasSdx)
				pa.Sdx = wasSdx
				return nil
			}
			from, _ := utf8.DecodeRuneInString((*rule.CodeChilds)[0].String)
			to, _ := utf8.DecodeRuneInString((*rule.CodeChilds)[1].String)
			// TODO: check if len of rune is len of string. Panic otherwise. Or better: Do that in verifier.
			if !(ch >= from && ch <= to) {
				pa.ruleExit(rule, skipSpaceRule, depth, nil, wasSdx)
				pa.Sdx = wasSdx
				return nil
			}
			pa.Sdx += size
			if skippingSpaces {
				return &r.Rules{}
			}
			if ch >= 0 && ch <= 127 { // Cache the rune == 0...127 part of this rules and reuse them, because those are the most used chars and they are binary compatible with bytes. 128...255 are NOT compatible.
				if pa.rangeCache[ch] == nil {
					pa.rangeCache[ch] = &r.Rule{Operator: r.Token, String: string([]rune{ch})}
				}
				*localProductions = append(*localProductions, pa.rangeCache[ch])
			} else {
				*localProductions = append(*localProductions, &r.Rule{Operator: r.Token, String: string([]rune{ch})})
			}
		} else if rule.Int == r.RangeTypeByte { // Byte range for binary decoding. JS-Mapping: abnf.rangeType.Byte
			ch := pa.Src[pa.Sdx]
			from := (*rule.CodeChilds)[0].String[0]
			to := (*rule.CodeChilds)[1].String[0]
			// TODO: check if len of string is 1. Panic otherwise. Or better: Do that in verifier.
			if !(ch >= from && ch <= to) {
				pa.ruleExit(rule, skipSpaceRule, depth, nil, wasSdx)
				pa.Sdx = wasSdx
				return nil
			}
			pa.Sdx++
			if skippingSpaces {
				return &r.Rules{}
			}
			// Cache all bytes (0...255) of this rules and reuse them.
			if pa.rangeCache[ch] == nil {
				pa.rangeCache[ch] = &r.Rule{Operator: r.Token, String: string([]byte{ch})}
			}
			*localProductions = append(*localProductions, pa.rangeCache[ch])
		} else {
			panic(fmt.Sprintf("Not a valid Range mode: %d", rule.Int))
		}
	case r.Or:
		found := false
		for i := range *rule.Childs {
			newProductions := pa.apply((*rule.Childs)[i], skipSpaceRule, skippingSpaces, depth+1)
			if newProductions != nil { // The nil result is used as ERROR. So if a match is successful but has nothing to return, it should only return something empty but not nil.
				localProductions = r.AppendArrayOfPossibleSequences(localProductions, newProductions)
				found = true
				break
			}
			// pa.Sdx = wasSdx // Should not be necessary, because each apply returns to wasSdx if the rule could not be fully applied.
		}
		if !found {
			pa.ruleExit(rule, skipSpaceRule, depth, nil, wasSdx)
			pa.Sdx = wasSdx
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
			newProductions := pa.apply(newRule, skipSpaceRule, skippingSpaces, depth+1)
			if newProductions == nil {
				break
			}
			localProductions = r.AppendArrayOfPossibleSequences(localProductions, newProductions) // Only append if all child rules matched.
		}
	case r.Times:
		// Copy first! Otherwise it will override all other positions too.
		cloneRule := &r.Rule{Operator: r.Times, CodeChilds: &r.Rules{}, Childs: rule.Childs}
		// The CodeChilds-array will be modified, so copy each entry:
		*cloneRule.CodeChilds = append(*cloneRule.CodeChilds, *rule.CodeChilds...)
		// It's not very clean but just pretend, the clone is the original:
		rule = cloneRule
		// Resolve parameters:
		for i, child := range *rule.CodeChilds {
			if child.Operator == r.Number {
				continue
			}
			// TODO: When command is something else as :number(), resolve without checking or forwarding in pa.Src. Those parameters should only exist in and be fetched from agrammar. Make a distinction between forward (pa.Src) looking parameter and backwards (agrammar) looking parameters.
			if child.Operator != r.Command {
				panic(fmt.Sprintf("Parameter can not be used for Times: %s", rule.ToString()))
			}
			if child.String != "number" {
				panic(fmt.Sprintf("Only Command :number() can be used for Times. Command is: %s", rule.ToString()))
			}
			resRule := pa.apply(child, skipSpaceRule, skippingSpaces, depth+1)
			if resRule == nil || len(*resRule) != 1 {
				panic("Parameter needs to result in exactly one result. Rule: " + child.ToString())
			}
			(*rule.CodeChilds)[i] = (*resRule)[0]
		}
		// Define "from" and "to" range:
		from := (*rule.CodeChilds)[0].Int
		var to int
		if len(*rule.CodeChilds) > 1 {
			toRule := (*rule.CodeChilds)[1]
			if toRule.Operator == r.Number {
				to = toRule.Int
			} else { // If the B-part of the A...B is no number, it indicates that it should allow infinite times. We only allow MaxInt32 times.
				to = math.MaxInt32
			}
		} else { // If there is only one number, it must occur exactly that often.
			to = from
		}
		// Define rule to apply:
		var newRule *r.Rule
		if len(*rule.Childs) == 1 {
			newRule = (*rule.Childs)[0]
		} else {
			newRule = &r.Rule{Operator: r.Sequence, Childs: rule.Childs, Pos: rule.Pos}
		}
		// Repeat from zero to "from" -> here it HAS TO be found:
		for i := 0; i < from; i++ { // Repeat as often as possible.
			newProductions := pa.apply(newRule, skipSpaceRule, skippingSpaces, depth+1)
			if newProductions == nil {
				pa.ruleExit(rule, skipSpaceRule, depth, nil, wasSdx)
				pa.Sdx = wasSdx
				return nil
			}
			localProductions = r.AppendArrayOfPossibleSequences(localProductions, newProductions) // Only append if all child rules matched.
		}
		// Repeat from "from" to "to" -> here it CAN be found:
		for i := from; i < to; i++ { // Repeat as often as possible.
			newProductions := pa.apply(newRule, skipSpaceRule, skippingSpaces, depth+1)
			if newProductions == nil {
				break
			}
			localProductions = r.AppendArrayOfPossibleSequences(localProductions, newProductions) // Only append if all child rules matched.
		}
	case r.Optional:
		newProductions := pa.applyAsSequence(rule.Childs, skipSpaceRule, skippingSpaces, depth+1, rule.Pos)
		localProductions = r.AppendArrayOfPossibleSequences(localProductions, newProductions) // If not all child rules matched, newProductions is nil anyways.
	case r.Identifier: // This identifies another rule (and its index), it is basically a link: E.g. to the expression-rule which is at position 3: { "Identifier", "expression", 3 }
		newProductions := pa.applyAsSequence((*pa.agrammar)[rule.Int].Childs, skipSpaceRule, skippingSpaces, depth+1, rule.Pos)
		if newProductions == nil {
			pa.ruleExit(rule, skipSpaceRule, depth, nil, wasSdx)
			pa.Sdx = wasSdx
			return nil
		}
		localProductions = r.AppendArrayOfPossibleSequences(localProductions, newProductions)
	case r.Tag:
		newProductions := pa.applyAsSequence(rule.Childs, skipSpaceRule, skippingSpaces, depth+1, rule.Pos) // TODO: resolveRulesToToken for constants
		if newProductions == nil {
			pa.ruleExit(rule, skipSpaceRule, depth, nil, wasSdx)
			pa.Sdx = wasSdx
			return nil
		}
		pa.resolveParameterToToken(rule.CodeChilds)
		*localProductions = append(*localProductions, &r.Rule{Operator: r.Tag, CodeChilds: rule.CodeChilds, Childs: newProductions, Pos: pa.Sdx})
	case r.Command:
		switch rule.String {
		case "whitespace":
			rule.Pos = pa.Sdx
			localProductions = &r.Rules{rule} // Put the responsibility for the Command :skip() to the parent rule (the caller), because only the parent can change its own doSkipSpaces mode.
		case "number": // Mainly to dynamically create Times rules.
			byteCount := 0
			numberType := r.NumberTypeLittleEndian // JS-Mapping: abnf.numberType
			if rule.CodeChilds != nil && len(*rule.CodeChilds) > 0 {
				if (*rule.CodeChilds)[0].Operator == r.Number {
					byteCount = (*rule.CodeChilds)[0].Int
				}
				if len(*rule.CodeChilds) > 1 && (*rule.CodeChilds)[1].Operator == r.Number {
					numberType = (*rule.CodeChilds)[1].Int
				}
			}
			if pa.Sdx+byteCount > len(pa.Src) {
				pa.ruleExit(rule, skipSpaceRule, depth, nil, wasSdx)
				pa.Sdx = wasSdx
				return nil
			}
			n := 0
			switch numberType { // TODO: Everything for signed numbers too!
			case r.NumberTypeLittleEndian:
				switch byteCount {
				case 1:
					n = int(pa.Src[pa.Sdx])
				case 2:
					n = int(binary.LittleEndian.Uint16([]byte(pa.Src[pa.Sdx:])))
				case 3:
					n = int(binary.LittleEndian.Uint32([]byte(pa.Src[pa.Sdx : pa.Sdx+3])))
				case 4:
					n = int(binary.LittleEndian.Uint32([]byte(pa.Src[pa.Sdx:])))
				case 8:
					n = int(binary.LittleEndian.Uint64([]byte(pa.Src[pa.Sdx:])))
				default:
					panic(":number() needs byte count of 1, 2, 3, 4, 8. ( e.g. :number(4) )")
				}
			case r.NumberTypeBigEndian:
				switch byteCount {
				case 1:
					n = int(pa.Src[pa.Sdx])
				case 2:
					n = int(binary.BigEndian.Uint16([]byte(pa.Src[pa.Sdx:])))
				case 3:
					n = int(binary.BigEndian.Uint32([]byte(pa.Src[pa.Sdx : pa.Sdx+3])))
				case 4:
					n = int(binary.BigEndian.Uint32([]byte(pa.Src[pa.Sdx:])))
				case 8:
					n = int(binary.BigEndian.Uint64([]byte(pa.Src[pa.Sdx:])))
				default:
					panic(":number() needs byte count of 1, 2, 3, 4, 8. ( e.g. :number(4) )")
				}
			case r.NumberTypeBCD:
				panic("NOT IMPLEMENTED")
			case r.NumberTypeASCII:
				switch byteCount {
				case 0: // Automatically read the number as long as it is in the target text.
					panic("NOT IMPLEMENTED")
					// tmp := gp.apply(&r.Rule{Number rule.....}, doSkipSpaces, depth)
					// n = tmp[0].Int
					// gp.sdx += foundBytes
				default:
					res, err := strconv.ParseInt(pa.Src[pa.Sdx:pa.Sdx+byteCount], 10, 64)
					if err != nil {
						panic("Can not parse int: '" + pa.Src[pa.Sdx:pa.Sdx+byteCount] + "'")
					}
					n = int(res)
				}
			default:
				panic(fmt.Sprintf("Invalid number type: %d", numberType))
			}
			*localProductions = append(*localProductions, &r.Rule{Operator: r.Number, Int: n})
			pa.Sdx += byteCount
		case "done": // To end the parsing successfully at this place.
			// TODO: This should be implemented better (everything should return).
			pa.Sdx = len(pa.Src)
			panic("NOT IMPLEMENTED")
		case "include":
			panic("NOT IMPLEMENTED")
		case "script": // TODO: Maybe move upwards like :skip().
			resRule := pa.ps.HandleScriptRule(rule, localProductions, depth) // TODO: localProductions is empty here...
			if resRule != nil {
				scriptProductions := pa.apply(resRule, skipSpaceRule, skippingSpaces, depth+1)
				if scriptProductions == nil {
					pa.ruleExit(rule, skipSpaceRule, depth, nil, wasSdx)
					pa.Sdx = wasSdx
					return nil
				}
				if len(*scriptProductions) > 0 {
					*localProductions = append(*localProductions, *scriptProductions...)
				}
			}
		case "title":
			// TODO: Maybe use that information.
		case "description":
			// TODO: Maybe use that information.
		case "startRule":
			// This is used by ParseWithAgrammar().
		case "startScript":
			// This is used by ParseWithAgrammar().
		default:
			panic("Unknown line command :" + rule.String + "()")
		}
	default: // r.Success || r.Error
		panic(fmt.Sprintf("Invalid rule in apply() function: %s", rule.ToString()))
	}

	// All failed matches should have returned already.
	// Here must only be matches which means, localProductions MUST NOT be nil here.

	if pa.Sdx > pa.lastParsePosition {
		pa.lastParsePosition = pa.Sdx
	}

	if len(*localProductions) == 1 {
		// if (*localProductions)[0].Operator == r.Group || (*localProductions)[0].Operator == r.Sequence { // If Groups are used, break them too here.
		if (*localProductions)[0].Operator == r.Sequence { // There should be only Sequences, Tags or Token left. Everything should be as flat as possible. Sequences can sometimes be simplified further.
			localProductions = (*localProductions)[0].Childs
		}
	}

	pa.ruleExit(rule, skipSpaceRule, depth, localProductions, wasSdx)
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
			// if (*productions)[i].Childs == nil {
			// 	panic((*productions)[i].Operator.String())
			// }
			// if (*productions)[i].Childs != nil && len(*(*productions)[i].Childs) > 0 {
			if len(*(*productions)[i].Childs) > 0 {
				mergeTerminals((*productions)[i].Childs)
			}
		}
	}
}

func ParseWithAgrammar(agrammar *r.Rules, srcCode, fileName string, useBlockList, useFoundList, traceEnabled bool) (res *r.Rules, e error) { // => (productions, error)
	// defer func() {
	// 	if err := recover(); err != nil {
	// 		res = nil
	// 		e = fmt.Errorf("%s", err)
	// 	}
	// }()

	startRule := r.GetStartRule(agrammar)
	if startRule == nil || startRule.Int >= len(*agrammar) || startRule.Int < 0 {
		// No valid start rule defined. Imeediately return but this is no error. The startScript() rule of the compiler has to do everything now.
		return nil, nil
	}

	var pa parser
	pa.agrammar = agrammar
	pa.Src = srcCode
	pa.Sdx = 0
	pa.traceEnabled = traceEnabled
	pa.traceCount = 0
	pa.blockList = make(map[int]bool)
	pa.useBlockList = useBlockList
	pa.foundList = make(map[int]*r.Rules)
	pa.foundSdxList = make(map[int]int)
	pa.useFoundList = useFoundList
	pa.lastParsePosition = 0

	pa.fileName = filepath.Clean(fileName)

	pa.ps = NewParserScript(&pa)

	pa.spaces = &r.Rule{Operator: r.CharsOf, String: "\t\n\r "} // TODO: Make this configurable via JS.

	for _, rule := range *pa.agrammar {
		if rule.Operator == r.Command {
			pa.applyCommand(rule)
		}
	}

	// For the parsing, the start rule is necessary. For the compilation not.
	newProductions := pa.apply((*pa.agrammar)[startRule.Int], pa.spaces, false, 0)

	// Check if the position is at EOF at end of parsing. There can be spaces left, but otherwise its an error:
	if pa.spaces != nil {
		pa.apply(pa.spaces, pa.spaces, true, 0) // Skip spaces.
	}
	if pa.Sdx < len(pa.Src) {
		panic(fmt.Sprintf("Not everything could be parsed. Last good parse position was %s\nCreated productions: %s", LinePosFromStrPos(string(pa.Src), pa.lastParsePosition), Shorten(newProductions.Serialize())))
	}

	mergeTerminals(newProductions)
	return newProductions, nil
}
