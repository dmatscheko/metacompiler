package abnf

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"14.gy/mec/abnf/r"
	"github.com/dop251/goja"
)

// TODO: switch to "text/scanner".

// func RuneToStr(r rune) string {
// 	if r > utf8.MaxRune {
// 		r = utf8.RuneError
// 	}
// 	buf := make([]byte, 0, utf8.UTFMax)
// 	size := utf8.EncodeRune(buf[:utf8.UTFMax], r)
// 	return string(buf[:size])
// }

// ----------------------------------------------------------------------------
// Dynamic grammar parser
//
// Idea partially taken from: https://rosettacode.org/wiki/Parse_EBNF

type agrammarParser struct {
	src         string
	sdx         int
	agrammar    *r.Rules
	productions *r.Rules

	blockList    map[int]bool
	useBlockList bool
	foundList    map[int]*r.Rules
	foundSdxList map[int]int
	useFoundList bool

	spaces string

	lastParsePosition int

	rangeCache [256]*r.Rule

	traceEnabled bool
	traceCount   int

	vm                   *goja.Runtime
	codeCache            map[string]*goja.Program
	compilerFuncMap      map[string]r.Object
	preventDefaultOutput bool
	stack                []r.Object // global stack.
}

// ----------------------------------------------------------------------------
// Dynamic script rule for parser

// Run executes the given string in the global context.
func (gp *agrammarParser) Run(name, src string) (goja.Value, error) {
	p := gp.codeCache[src]

	// Cache precompiled data
	if p == nil {
		var err error
		p, err = goja.Compile(name, src, true)
		if err != nil {
			return nil, err
		}
		gp.codeCache[src] = p
	}

	return gp.vm.RunProgram(p)
}

func (gp *agrammarParser) handleScriptRule(rule *r.Rule, localProductions *r.Rules, doSkipSpaces string, depth int) *r.Rule {
	gp.compilerFuncMap["localAsg"] = localProductions // The local part of the abstract syntax graph.

	if gp.traceEnabled {
		// co.traceTop(tag, slot, depth, upStream)
	}

	code := (*rule.CodeChilds)[0].String

	v, err := gp.Run("parserCommand@"+strconv.Itoa(rule.Pos), code)
	if err != nil {
		panic(err.Error() + "\nError was in " + PprintRuleFlat(rule, false, true))
	}

	res, ok := v.Export().(*r.Rule)

	if gp.traceEnabled {
		// gp.traceBottom(upStream)
	}

	if ok {
		return res
	}
	return nil
}

func (gp *agrammarParser) initFuncMap() {
	initFuncMapCommon(gp.vm, &gp.compilerFuncMap, gp.preventDefaultOutput)

	gp.compilerFuncMap["getSrc"] = func() string { return gp.src }
	gp.compilerFuncMap["setSrc"] = func(src string) { gp.src = src }
	gp.compilerFuncMap["getSdx"] = func() int { return gp.sdx }
	gp.compilerFuncMap["setSdx"] = func(sdx int) { gp.sdx = sdx }

	gp.vm.Set("pop", func() interface{} {
		if len(gp.stack) > 0 {
			res := gp.stack[len(gp.stack)-1]
			gp.stack = gp.stack[:len(gp.stack)-1]
			return res
		}
		return nil
	})

	gp.vm.Set("push", func(v interface{}) {
		gp.stack = append(gp.stack, v)
	})
}

// ----------------------------------------------------------------------------

func (gp *agrammarParser) skipSpaces(spaces string) {
	for {
		if gp.sdx >= len(gp.src) {
			break
		}
		ch, size := utf8.DecodeRuneInString(gp.src[gp.sdx:])
		if strings.IndexRune(spaces, ch) == -1 {
			break
		}
		gp.sdx += size
	}
}

func (gp *agrammarParser) getRulePosId(rule *r.Rule, pos int) int {
	// Cantor pairing function
	a := rule.Int
	ab := pos + rule.Int
	return ((ab * (ab + 1)) >> 1) + a
}

func (gp *agrammarParser) ruleEnter(rule *r.Rule, doSkipSpaces string, depth int) (bool, *r.Rules, int) { // => (isBlocked, rules, rulesSdx, rulesCh)
	var isBlocked bool = false
	var foundRule *r.Rules = nil
	var foundSdx int = -1
	gp.traceCount++

	if rule.Operator == r.Identifier && (gp.useBlockList || gp.useFoundList) {
		id := gp.getRulePosId(rule, gp.sdx)

		if gp.useFoundList {
			foundRule = gp.foundList[id]
			if foundRule != nil { // Is really only useful on massive backtracking...
				foundSdx = gp.foundSdxList[id]
				isBlocked = true // If the result is there already, block the apply() from trying it again.
			}
		}
		if !isBlocked && gp.useBlockList {
			isBlocked = gp.blockList[id] // True if loop, because in this case, the current rule on the current position in the text to parse is its own parent (= loop).
			// TODO: Maybe only block at identifier:
			// if !isBlocked && rule.Operator != r.Or { // Could r.Or in r.Or's loop forever? No because only r.Ident's can create loops and they would be marked. But the result could be found very late because the r.Or can get stuck for a long time if there is another r.Or as child (it would not get blocked from list of the first r.Or options).
			gp.blockList[id] = true // Enter the current rule rule in the block list because it was not already blocked.
			// }
		}
	}

	if !gp.traceEnabled {
		return isBlocked, foundRule, foundSdx
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
	return isBlocked, foundRule, foundSdx
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

// Almost like apply() but without parsing the target text.
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
			panic("Const must only contain Token. Contains: " + PprintRuleOnly(rule))
		}
		length += len(rule.String)
	}

	buf := make([]byte, 0, length)
	for _, rule := range *rules {
		buf = append(buf, rule.String...)
	}

	return &r.Rule{Operator: r.Token, String: string(buf)}
}

func (gp *agrammarParser) resolveParameterToToken(rules *r.Rules) {
	if rules == nil {
		return
	}
	for i := range *rules {
		resRule := flattenToken(gp.resolveRulesToToken(&r.Rules{(*rules)[i]}))
		if resRule == nil {
			panic("Parameter is empty. Rule: " + PprintRuleOnly((*rules)[i]))
		}
		(*rules)[i] = resRule
	}
}

// The global commands (Production level).
func (gp *agrammarParser) applyCommand(rule *r.Rule) {
	gp.resolveParameterToToken(rule.CodeChilds)
	switch rule.String {
	case "skip":
		if rule.CodeChilds != nil && len(*rule.CodeChilds) > 0 {
			gp.spaces = (*rule.CodeChilds)[0].String
		} else {
			gp.spaces = ""
		}
	case "include":
		// :include("foo_a.bnf"); How to include only the productions? Maybe overhaul of EBNF format?
		panic("NOT IMPLEMENTED")
	case "number":
		// :number(4, LE) would mean take 4 bytes from the input (gp.src), interpret them as little endian and create a Number from it. This means it should be usable in Times expressions and should allow the parsing of TLV-formats.
		panic("NOT IMPLEMENTED")
	default:
		panic("Unknown command :'" + rule.String + "()'")
	}
}

// Apply uses the rules top down and recursively.
// This is the resolution process of the agrammar. Does the localProductions need to go into a Group or something? No. The grouping was done already in the agrammar.
// At this point, the only grouping is done by Tags. The rest can stay in flat Sequences or arrays of rules.
// Rules can be reused. So whatever you do, NEVER change a rule here.
func (gp *agrammarParser) apply(rule *r.Rule, doSkipSpaces string, depth int) *r.Rules { // => (localProductions)
	wasSdx := gp.sdx // Start position of the rule. Return, if the rule does not match.
	localProductions := &r.Rules{}

	isBlocked, foundRule, foundSdx := gp.ruleEnter(rule, doSkipSpaces, depth)
	if isBlocked {
		if foundRule != nil {
			gp.sdx = foundSdx
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
			} else if len(*newProductions) > 0 { // Some Commands like :skip() have to be handled inside the sequence.
				noResult := false
				for _, prod := range *newProductions { // There should be only one command but just in case for future upgradeability.
					// The local commands (inside an Expression).
					if prod.Operator == r.Command {
						noResult = true
						switch prod.String {
						case "skip":
							// Resolve parameter constants.
							gp.resolveParameterToToken(prod.CodeChilds)
							if prod.CodeChilds != nil && len(*prod.CodeChilds) > 0 {
								doSkipSpaces = (*prod.CodeChilds)[0].String
							} else {
								doSkipSpaces = ""
							}
						default:
							// All other commands should have been handled already by apply() and so this should never happen.
							panic("Unknown command :'" + prod.String + "()'")
						}
					}
				}
				if noResult { // TODO: If it not a command, add the result to the localProductions.
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

		size := len(rule.String)
		if gp.sdx+size > len(gp.src) || rule.String != gp.src[gp.sdx:gp.sdx+size] {
			gp.ruleExit(rule, doSkipSpaces, depth, nil, wasSdx)
			gp.sdx = wasSdx
			return nil
		}
		gp.sdx += size
		*localProductions = append(*localProductions, rule)
	case r.Range:
		// Only skip spaces when actually reading from the target text (Tokens)
		gp.skipSpaces(doSkipSpaces)

		if gp.sdx >= len(gp.src) {
			gp.ruleExit(rule, doSkipSpaces, depth, nil, wasSdx)
			gp.sdx = wasSdx
			return nil
		}

		if rule.Int == 0 { // Rune range for unicode. JS-Mapping: abnf.rangeType.Rune

			ch, size := utf8.DecodeRuneInString(gp.src[gp.sdx:])
			if ch == utf8.RuneError {
				gp.ruleExit(rule, doSkipSpaces, depth, nil, wasSdx)
				gp.sdx = wasSdx
				return nil
			}

			from, _ := utf8.DecodeRuneInString((*rule.CodeChilds)[0].String)
			to, _ := utf8.DecodeRuneInString((*rule.CodeChilds)[1].String)
			// TODO: check if len of rune is len of string. Panic otherwise. Or better: Do that in verifier.

			if !(ch >= from && ch <= to) {
				gp.ruleExit(rule, doSkipSpaces, depth, nil, wasSdx)
				gp.sdx = wasSdx
				return nil
			}

			gp.sdx += size

			if ch >= 0 && ch <= 127 { // Cache the rune == 0...127 part of this rules and reuse them, because those are the most used chars and they are binary compatible with bytes. 128...255 are NOT compatible.
				if gp.rangeCache[ch] == nil {
					gp.rangeCache[ch] = &r.Rule{Operator: r.Token, String: string([]rune{ch})}
				}
				*localProductions = append(*localProductions, gp.rangeCache[ch])
			} else {
				*localProductions = append(*localProductions, &r.Rule{Operator: r.Token, String: string([]rune{ch})})
			}

		} else if rule.Int == 1 { // Byte range for binary decoding. JS-Mapping: abnf.rangeType.Byte

			ch := gp.src[gp.sdx]
			from := (*rule.CodeChilds)[0].String[0]
			to := (*rule.CodeChilds)[1].String[0]
			// TODO: check if len of string is 1. Panic otherwise. Or better: Do that in verifier.

			if !(ch >= from && ch <= to) {
				gp.ruleExit(rule, doSkipSpaces, depth, nil, wasSdx)
				gp.sdx = wasSdx
				return nil
			}

			gp.sdx++

			// Cache all bytes (0...255) of this rules and reuse them.
			if gp.rangeCache[ch] == nil {
				gp.rangeCache[ch] = &r.Rule{Operator: r.Token, String: string([]byte{ch})}
			}
			*localProductions = append(*localProductions, gp.rangeCache[ch])

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
	case r.Times:
		// Copy first! Otherwise it will override all other positions too.
		cloneRule := &r.Rule{CodeChilds: &r.Rules{}, Childs: rule.Childs}
		// The CodeChilds-array will be modified, so copy each entry:
		*cloneRule.CodeChilds = append(*cloneRule.CodeChilds, *rule.CodeChilds...)
		// It's not very clean but just pretend, the clone is the original:
		rule = cloneRule

		for i, child := range *rule.CodeChilds {
			if child.Operator == r.Number {
				continue
			}
			resRule := gp.apply(child, doSkipSpaces, depth+1)
			if resRule == nil || len(*resRule) != 1 {
				panic("Parameter needs to result in exactly one result. Rule: " + PprintRuleOnly(child))
			}
			(*rule.CodeChilds)[i] = (*resRule)[0]
		}

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

		var newRule *r.Rule
		if len(*rule.Childs) == 1 {
			newRule = (*rule.Childs)[0]
		} else {
			newRule = &r.Rule{Operator: r.Sequence, Childs: rule.Childs, Pos: rule.Pos}
		}
		for i := 0; i < from; i++ { // Repeat as often as possible.
			newProductions := gp.apply(newRule, doSkipSpaces, depth+1)
			if newProductions == nil {
				gp.ruleExit(rule, doSkipSpaces, depth, nil, wasSdx)
				gp.sdx = wasSdx
				return nil
			}
			localProductions = r.AppendArrayOfPossibleSequences(localProductions, newProductions) // Only append if all child rules matched.
		}
		for i := from; i < to; i++ { // Repeat as often as possible.
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
		gp.resolveParameterToToken(rule.CodeChilds)
		*localProductions = append(*localProductions, &r.Rule{Operator: r.Tag, CodeChilds: rule.CodeChilds, Childs: newProductions, Pos: gp.sdx})
	case r.Command:
		switch rule.String {
		case "skip":
			rule.Pos = gp.sdx
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
			if gp.sdx+byteCount > len(gp.src) {
				gp.ruleExit(rule, doSkipSpaces, depth, nil, wasSdx)
				gp.sdx = wasSdx
				return nil
			}
			n := 0
			switch numberType { // TODO: Everything for singed numbers too!
			case r.NumberTypeLittleEndian:
				if byteCount == 0 {
					panic(":number() needs at least the byte count. ( e.g. :number(4) )")
				} else if byteCount == 1 {
					n = int(gp.src[gp.sdx])
				} else if byteCount == 2 {
					n = int(binary.LittleEndian.Uint16([]byte(gp.src[gp.sdx:])))
				} else if byteCount == 3 {
					n = int(binary.LittleEndian.Uint32([]byte(gp.src[gp.sdx : gp.sdx+3])))
				} else if byteCount == 4 {
					n = int(binary.LittleEndian.Uint32([]byte(gp.src[gp.sdx:])))
				} else if byteCount == 8 {
					n = int(binary.LittleEndian.Uint64([]byte(gp.src[gp.sdx:])))
				} else {
					panic(":number() needs byte count of 1, 2, 3, 4, 8. ( e.g. :number(4) )")
				}
			case r.NumberTypeBigEndian:
				if byteCount == 0 {
					panic(":number() needs at least the byte count. ( e.g. :number(4) )")
				} else if byteCount == 1 {
					n = int(gp.src[gp.sdx])
				} else if byteCount == 2 {
					n = int(binary.BigEndian.Uint16([]byte(gp.src[gp.sdx:])))
				} else if byteCount == 3 {
					n = int(binary.BigEndian.Uint32([]byte(gp.src[gp.sdx : gp.sdx+3])))
				} else if byteCount == 4 {
					n = int(binary.BigEndian.Uint32([]byte(gp.src[gp.sdx:])))
				} else if byteCount == 8 {
					n = int(binary.BigEndian.Uint64([]byte(gp.src[gp.sdx:])))
				} else {
					panic(":number() needs byte count of 1, 2, 3, 4, 8. ( e.g. :number(4) )")
				}
			case r.NumberTypeBCD:
				if byteCount == 0 { // Automatically read the number as long as it is in the target text.
					panic("NOT IMPLEMENTED")
					// tmp := gp.apply(&r.Rule{Number rule.....}, doSkipSpaces, depth)
					// n = tmp[0].Int
					// gp.sdx += foundBytes
				} else {
					res, err := strconv.ParseInt(gp.src[gp.sdx:gp.sdx+byteCount], 10, 64)
					if err != nil {
						panic("Can not parse int: '" + gp.src[gp.sdx:gp.sdx+byteCount] + "'")
					}
					n = int(res)
				}
			}
			*localProductions = append(*localProductions, &r.Rule{Operator: r.Number, Int: n})
			gp.sdx += byteCount
		case "done": // To end the parsing successfully at this place.
			// TODO: This does not work.
			gp.sdx = len(gp.src)
			panic("NOT IMPLEMENTED")
		case "include":
			panic("NOT IMPLEMENTED")
		case "script": // TODO: Maybe move upwards like :skip().
			resRule := gp.handleScriptRule(rule, localProductions, doSkipSpaces, depth) // TODO: localProductions is empty here...
			if resRule != nil {
				scriptProductions := gp.apply(resRule, doSkipSpaces, depth+1)
				if scriptProductions == nil {
					gp.ruleExit(rule, doSkipSpaces, depth, nil, wasSdx)
					gp.sdx = wasSdx
					return nil
				}
				if len(*scriptProductions) > 0 {
					*localProductions = append(*localProductions, *scriptProductions...)
				}
			}
		default:
			panic("Unknown command :'" + rule.String + "()'")
		}
	default: // r.Success || r.Error
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

func ParseWithAgrammar(agrammar *r.Rules, srcCode string, useBlockList bool, useFoundList bool, traceEnabled bool) (res *r.Rules, e error) { // => (productions, error)
	// defer func() {
	// 	if err := recover(); err != nil {
	// 		res = nil
	// 		e = fmt.Errorf("%s", err)
	// 	}
	// }()

	var gp agrammarParser
	gp.agrammar = agrammar
	gp.src = srcCode
	gp.sdx = 0
	gp.traceEnabled = traceEnabled
	gp.traceCount = 0
	gp.blockList = make(map[int]bool)
	gp.useBlockList = useBlockList
	gp.foundList = make(map[int]*r.Rules)
	gp.foundSdxList = make(map[int]int)
	gp.useFoundList = useFoundList
	gp.lastParsePosition = 0

	gp.vm = goja.New()
	gp.codeCache = map[string]*goja.Program{}
	gp.initFuncMap()

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
		return nil, fmt.Errorf("Not everything could be parsed. Last good parse position was %s\nFailed productions: %s", LinePosFromStrPos(string(gp.src), gp.lastParsePosition), newProductions.Serialize())
	}

	mergeTerminals(newProductions)
	return newProductions, nil
}
