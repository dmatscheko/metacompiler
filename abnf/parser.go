package abnf

import (
	"encoding/binary"
	"encoding/hex"
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
	Src      string   // The target text that gets parsed.
	Sdx      int      // The current parse position inside Src (byte index).
	agrammar *r.Rules // The a-grammar that describes the target text.

	opts         *Parseropts
	blockList    map[int]bool     // Marks the (rule, position) pairs that are currently being applied (-lb). Used to stop left recursions.
	foundList    map[int]*r.Rules // Caches the results of successfully applied (rule, position) pairs (-lf).
	foundSdxList map[int]int      // The parse position right behind each cached foundList result.
	traceCount   int

	initialSpaces *r.Rule // The rule that describes skippable whitespace (nil skips nothing). Set via :whitespace(), see applyCommand().

	lastParsePosition int // The furthest position that could be parsed. Only used for error messages.

	ps scriptRuleRunner // The JS subsystem for dynamic :script() rules.

	fileName string // Where Src came from. Used for messages and to resolve relative paths.

	includedFiles map[string]bool // The :include() files already added (each file is included once; also ends include cycles).

	rangeCache      [256]*r.Rule // Reusable single-char Token rules, see the comment in case r.CharOf of apply().
	referencesCache *references  // Resolves production names and assigns the tag code UIDs.
}

// Parseropts are the command line options that influence the parser.
type Parseropts struct {
	UseBlockList, UseFoundList, TraceEnabled, PreventDefaultOutput bool
}

// getRulePosId maps the pair (rule, position in the target text) to one unique int,
// used as key for the block and found lists.
func (pa *parser) getRulePosId(rule *r.Rule, pos int) int {
	// Cantor pairing function.
	a := rule.Int
	ab := pos + rule.Int
	return ((ab * (ab + 1)) >> 1) + a
}

func (pa *parser) ruleEnter(rule *r.Rule, skipSpaceRule *r.Rule, depth int) (bool, *r.Rules, int) { // => (isBlocked, foundRule, foundSdx)
	var isBlocked bool = false
	var foundRule *r.Rules = nil
	var foundSdx int = -1
	pa.traceCount++

	if rule.Operator == r.Identifier && (pa.opts.UseBlockList || pa.opts.UseFoundList) {
		id := pa.getRulePosId(rule, pa.Sdx)

		if pa.opts.UseFoundList {
			foundRule = pa.foundList[id]
			if foundRule != nil { // Is really only useful on massive backtracking...
				foundSdx = pa.foundSdxList[id]
				isBlocked = true // If the result is there already, block the apply() from trying it again.
			}
		}
		if !isBlocked && pa.opts.UseBlockList {
			isBlocked = pa.blockList[id] // True if loop, because in this case, the current rule on the current position in the text to parse is its own parent (= loop).
			pa.blockList[id] = true      // Enter the current rule in the block list (a no-op if it was already blocked).
		}
	}

	if !pa.opts.TraceEnabled {
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

// The parameter wasBlocked must be true if the corresponding ruleEnter() returned isBlocked == true.
// Such an early exit must not touch the block and found lists: The list entries belong to the still
// running outer invocation of the same rule at the same position, not to this blocked one.
func (pa *parser) ruleExit(rule *r.Rule, skipSpaceRule *r.Rule, depth int, found *r.Rules, wasSdx int, wasBlocked bool) {
	if !wasBlocked && rule.Operator == r.Identifier && (pa.opts.UseBlockList || pa.opts.UseFoundList) {
		id := pa.getRulePosId(rule, wasSdx)
		if pa.opts.UseBlockList {
			pa.blockList[id] = false // Exit of the rule. It must be unblocked so it can be called again from a parent.
		}
		if pa.opts.UseFoundList && found != nil {
			pa.foundList[id] = found
			pa.foundSdxList[id] = pa.Sdx
		}
	}

	if !pa.opts.TraceEnabled {
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
	// A sole Command child (a production that is just ':whitespace()') still
	// needs the Sequence wrapper: applied directly, the Command would escape
	// into the CALLER's sequence and change the caller's whitespace skipping,
	// while ':whitespace() X' stays scoped to this production.
	if len(*rules) == 1 && (*rules)[0].Operator != r.Command {
		newProductions = pa.apply((*rules)[0], skipSpaceRule, skipSpaces, depth)
	} else {
		newRule := &r.Rule{Operator: r.Sequence, Childs: rules, Pos: pos}
		newProductions = pa.apply(newRule, skipSpaceRule, skipSpaces, depth)
	}
	return newProductions
}

// Almost like apply() but without parsing the target text.
func (pa *parser) resolveRulesToToken(rules *r.Rules) *r.Rules {
	if rules == nil {
		return nil
	}
	newProductions := &r.Rules{}

	for _, rule := range *rules {
		switch rule.Operator {
		case r.Token:
			*newProductions = append(*newProductions, rule)
		case r.Identifier:
			if rule.Int < 0 || rule.Int >= len(*pa.agrammar) {
				panic("Unknown production name '" + rule.String + "' used as parameter.")
			}
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

// Resolves command and tag parameters in place: Every parameter that references Token
// productions (via a Name) is replaced by one Token that contains the combined text.
// Token parameters are already resolved and Number parameters (e.g. the slot number of
// :include()) have no text form, so both are kept as they are.
func (pa *parser) resolveParameterToToken(rules *r.Rules) {
	if rules == nil {
		return
	}
	for i := range *rules {
		if (*rules)[i].Operator == r.Token || (*rules)[i].Operator == r.Number {
			continue
		}
		resRule := flattenToken(pa.resolveRulesToToken(&r.Rules{(*rules)[i]}))
		if resRule == nil {
			panic("Parameter is empty. Rule: " + (*rules)[i].ToString())
		}
		(*rules)[i] = resRule
	}
}

// padBytes zero-extends b to size bytes: in front for big endian values, behind
// for little endian ones. b is already at most size long (the :number() cases).
func padBytes(b []byte, size int, front bool) []byte {
	if len(b) == size {
		return b
	}
	out := make([]byte, size)
	if front {
		copy(out[size-len(b):], b)
	} else {
		copy(out, b)
	}
	return out
}

// applyCommand executes the global commands (the LineCommands on Production level).
// It runs once for every Command rule of the a-grammar before the parsing starts.
// TODO: Maybe remove used commands.
func (pa *parser) applyCommand(rule *r.Rule) {
	switch rule.String {
	case "whitespace":
		// The parameter is kept as a rule (usually an Identifier) on purpose:
		// It is applied like any other rule whenever spaces can be skipped.
		if rule.CodeChilds != nil && len(*rule.CodeChilds) > 0 {
			pa.initialSpaces = (*rule.CodeChilds)[0]
		} else {
			pa.initialSpaces = nil
		}
	case "include":
		// :include(fileName [, slot]) parses and compiles another ABNF file and adds its
		// productions to the current a-grammar. Note that this happens when the combined
		// a-grammar is USED, so the file name is resolved relative to the file that is
		// currently being parsed.
		// Resolve parameter constants (the file name can also be given via Token productions).
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
		// Int == 1: the path was anchored to the grammar file's directory by
		// CompileASG (resolveIncludePaths). The legacy fallback for dynamic
		// file names resolves relative to the file currently being parsed.
		fullFileName := paramFileName
		if rule.Int != 1 {
			fullFileName = filepath.Dir(pa.fileName) + string(os.PathSeparator) + paramFileName
		}
		// Include every file only once: with nested includes enabled, two
		// fragments sharing a common helper fragment would otherwise define
		// its productions twice (a hard error), and a cyclic include would
		// never terminate.
		fullFileName = filepath.Clean(fullFileName)
		if pa.includedFiles == nil {
			pa.includedFiles = map[string]bool{}
		}
		if pa.includedFiles[fullFileName] {
			return
		}
		pa.includedFiles[fullFileName] = true
		dat, err := ioutil.ReadFile(fullFileName)
		if err != nil {
			panic(err)
		}
		srcCode := string(dat)

		asg, err := ParseWithAgrammar(AbnfAgrammar, srcCode, fullFileName, pa.opts)
		if err != nil {
			panic(err)
		}
		aGrammar, err := CompileASG(asg, AbnfAgrammar, fullFileName, slot, false, false)
		if err != nil {
			panic(err)
		}
		*pa.agrammar = append(*pa.agrammar, *aGrammar...)
		// Correct all references: The included productions moved to new positions and
		// previously unresolved identifiers can now point to them.
		pa.referencesCache.correctReferencesAndIDs(pa.agrammar)
	case "number":
		// :number(size, type) reads bytes from the target text, so it only makes sense
		// inside an Expression (see apply()), not as a global line command.
		panic(":number() is only allowed as inline command.")
	case "title":
		// TODO: Maybe use that information.
	case "description":
		// TODO: Maybe use that information.
	case "origin":
		// The grammar file this a-grammar was compiled from (stamped by
		// CompileASG; the start script runs under this module name).
	case "startRule":
		// This is used by ParseWithAgrammar().
	case "startScript":
		// This is used by ParseWithAgrammar().
	default:
		panic("Unknown initial line command :" + rule.String + "()")
	}
}

// apply matches one rule of the a-grammar against the target text at the current
// position pa.Sdx, top down and recursively. It returns the productions that the rule
// created for the ASG, or nil if the rule did not match. On a failed match, pa.Sdx is
// restored to the position where the rule started.
// The returned productions stay as flat as possible: The only grouping that survives is
// done by Tags (the grouping of the grammar itself was already resolved here).
// Rules can be shared and reused between grammars. So whatever you do, NEVER change a
// rule in here (that is why e.g. the Times case clones its rule before resolving it).
//
// skipSpaceRule is the rule that describes what counts as skippable whitespace right now
// (nil means nothing is skipped). skippingSpaces is true while we are already inside such
// a whitespace rule, because then whitespace must not be skipped again (that would recurse
// forever) and no productions are created.
func (pa *parser) apply(rule *r.Rule, skipSpaceRule *r.Rule, skippingSpaces bool, depth int) *r.Rules { // => (localProductions)
	wasSdx := pa.Sdx // Start position of the rule. Return, if the rule does not match.
	localProductions := &r.Rules{}

	isBlocked, foundRule, foundSdx := pa.ruleEnter(rule, skipSpaceRule, depth)
	if isBlocked {
		if foundRule != nil { // Reuse the cached result and continue behind it.
			pa.Sdx = foundSdx
			pa.ruleExit(rule, skipSpaceRule, depth, foundRule, wasSdx, true)
			return foundRule
		}
		pa.ruleExit(rule, skipSpaceRule, depth, nil, wasSdx, true)
		return nil
	}

	switch rule.Operator {
	case r.Sequence, r.Group, r.Production: // Those are groups/sequences of rules. Iterate through them and apply.
		for i := range *rule.Childs {
			newProductions := pa.apply((*rule.Childs)[i], skipSpaceRule, skippingSpaces, depth+1)
			if newProductions == nil {
				pa.ruleExit(rule, skipSpaceRule, depth, nil, wasSdx, false)
				pa.Sdx = wasSdx
				return nil
			}
			if len(*newProductions) > 0 { // Some Commands like :whitespace() have to be handled inside the sequence.
				for _, prod := range *newProductions {
					// The local commands (inside an Expression).
					if prod.Operator == r.Command {
						switch prod.String {
						case "whitespace":
							// Change what counts as whitespace from here on inside this sequence.
							// The parameter is kept as a rule, see applyCommand().
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
					// During parsing, the only grouping is done by Tags. The rest can stay in flat Sequences or arrays of rules.
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
			pa.ruleExit(rule, skipSpaceRule, depth, nil, wasSdx, false)
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
			pa.ruleExit(rule, skipSpaceRule, depth, nil, wasSdx, false)
			pa.Sdx = wasSdx
			return nil
		}
		// Int holds the charType flags: Negated inverts the set, Byte matches one byte
		// instead of one rune. A set membership equal to Negated is the mismatch.
		negated := rule.Int&r.CharTypeNegated != 0
		if rule.Int&r.CharTypeByte != 0 {
			ch := pa.Src[pa.Sdx]
			if (strings.IndexByte(rule.String, ch) >= 0) == negated {
				pa.ruleExit(rule, skipSpaceRule, depth, nil, wasSdx, false)
				pa.Sdx = wasSdx
				return nil
			}
			pa.Sdx++
			if skippingSpaces {
				return &r.Rules{}
			}
			// Cache and reuse the Token rules for all bytes, see case r.Range.
			if pa.rangeCache[ch] == nil {
				pa.rangeCache[ch] = &r.Rule{Operator: r.Token, String: string([]byte{ch})}
			}
			*localProductions = append(*localProductions, pa.rangeCache[ch])
		} else {
			ch, size := utf8.DecodeRuneInString(pa.Src[pa.Sdx:])
			if ch == utf8.RuneError && size == 1 { // An invalid encoding never matches (like in case r.Range).
				pa.ruleExit(rule, skipSpaceRule, depth, nil, wasSdx, false)
				pa.Sdx = wasSdx
				return nil
			}
			if strings.ContainsRune(rule.String, ch) == negated {
				pa.ruleExit(rule, skipSpaceRule, depth, nil, wasSdx, false)
				pa.Sdx = wasSdx
				return nil
			}
			pa.Sdx += size
			if skippingSpaces {
				return &r.Rules{}
			}
			// Cache and reuse the Token rules for the runes 0...127, because those are by far
			// the most used chars and their encoding is identical to the single byte (the byte
			// range case below uses the same cache; bytes 128...255 are NOT identical to the
			// runes 128...255 and are only ever created by the byte cases).
			if ch >= 0 && ch <= 127 {
				if pa.rangeCache[ch] == nil {
					pa.rangeCache[ch] = &r.Rule{Operator: r.Token, String: string([]rune{ch})}
				}
				*localProductions = append(*localProductions, pa.rangeCache[ch])
			} else {
				*localProductions = append(*localProductions, &r.Rule{Operator: r.Token, String: string([]rune{ch})})
			}
		}
	case r.CharsOf:
		// Only skip spaces when actually reading from the target text (Tokens)
		if !skippingSpaces && skipSpaceRule != nil { // Do not skip spaces again when we are already at skipping spaces. Would result in an infinite loop.
			pa.apply(skipSpaceRule, skipSpaceRule, true, depth+1) // Skip spaces.
		}
		length := len(pa.Src)
		startPos := pa.Sdx
		// Int holds the charType flags, see case r.CharOf.
		negated := rule.Int&r.CharTypeNegated != 0
		byteMode := rule.Int&r.CharTypeByte != 0
		for pa.Sdx < length {
			if byteMode {
				if (strings.IndexByte(rule.String, pa.Src[pa.Sdx]) >= 0) == negated {
					break
				}
				pa.Sdx++
			} else {
				ch, size := utf8.DecodeRuneInString(pa.Src[pa.Sdx:])
				if ch == utf8.RuneError && size == 1 { // An invalid encoding never matches.
					break
				}
				if strings.ContainsRune(rule.String, ch) == negated {
					break
				}
				pa.Sdx += size
			}
		}
		size := pa.Sdx - startPos
		if size == 0 {
			pa.ruleExit(rule, skipSpaceRule, depth, nil, wasSdx, false)
			pa.Sdx = wasSdx
			return nil
		}
		if skippingSpaces {
			return &r.Rules{}
		}
		*localProductions = append(*localProductions, &r.Rule{Operator: r.Token, String: pa.Src[startPos:pa.Sdx]})
	case r.Not:
		// Negative lookahead: the single child is probed and the position is restored,
		// so nothing is ever consumed. The Not matches exactly when the child does NOT
		// match here. A successful Not has zero width and leaves nothing in the ASG.
		// (Side effects of :script() rules inside the probed child are not rolled back,
		// like everywhere else.)
		probe := pa.apply((*rule.Childs)[0], skipSpaceRule, skippingSpaces, depth+1)
		pa.Sdx = wasSdx
		if probe != nil { // The child matched: the lookahead fails.
			pa.ruleExit(rule, skipSpaceRule, depth, nil, wasSdx, false)
			return nil
		}
		if skippingSpaces {
			return &r.Rules{}
		}
	case r.Range:
		// Only skip spaces when actually reading from the target text (Tokens)
		if !skippingSpaces && skipSpaceRule != nil { // Do not skip spaces again when we are already at skipping spaces. Would result in an infinite loop.
			pa.apply(skipSpaceRule, skipSpaceRule, true, depth+1) // Skip spaces.
		}
		if pa.Sdx >= len(pa.Src) {
			pa.ruleExit(rule, skipSpaceRule, depth, nil, wasSdx, false)
			pa.Sdx = wasSdx
			return nil
		}
		if rule.Int == r.RangeTypeRune { // Rune range for unicode. JS-Mapping: abnf.rangeType.Rune
			ch, size := utf8.DecodeRuneInString(pa.Src[pa.Sdx:])
			if ch == utf8.RuneError && size == 1 { // An invalid encoding never matches (like in case r.CharOf); a real 3-byte U+FFFD does.
				pa.ruleExit(rule, skipSpaceRule, depth, nil, wasSdx, false)
				pa.Sdx = wasSdx
				return nil
			}
			from, _ := utf8.DecodeRuneInString((*rule.CodeChilds)[0].String)
			to, _ := utf8.DecodeRuneInString((*rule.CodeChilds)[1].String)
			// A multi-rune bound would silently use only its first rune here; -verify (abnf/verifier.go) reports such malformed ranges.
			if !(ch >= from && ch <= to) {
				pa.ruleExit(rule, skipSpaceRule, depth, nil, wasSdx, false)
				pa.Sdx = wasSdx
				return nil
			}
			pa.Sdx += size
			if skippingSpaces {
				return &r.Rules{}
			}
			if ch >= 0 && ch <= 127 { // Cache and reuse the Token rules for the runes 0...127, see case r.CharOf.
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
			// A multi-byte bound would silently use only its first byte here; -verify (abnf/verifier.go) reports such malformed ranges.
			if !(ch >= from && ch <= to) {
				pa.ruleExit(rule, skipSpaceRule, depth, nil, wasSdx, false)
				pa.Sdx = wasSdx
				return nil
			}
			pa.Sdx++
			if skippingSpaces {
				return &r.Rules{}
			}
			// Cache and reuse the Token rules for all bytes (0...255).
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
			pa.ruleExit(rule, skipSpaceRule, depth, nil, wasSdx, false)
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
			sdxBefore := pa.Sdx
			newProductions := pa.apply(newRule, skipSpaceRule, skippingSpaces, depth+1)
			if newProductions == nil {
				break
			}
			localProductions = r.AppendArrayOfPossibleSequences(localProductions, newProductions) // Only append if all child rules matched.
			if pa.Sdx == sdxBefore {
				break // The child rules matched without consuming anything (e.g. { [ "x" ] }). Repeating them again could never consume anything either, it would only loop forever.
			}
		}
	case r.Times:
		// Clone the rule first: The parameters in CodeChilds get resolved below (a :number()
		// parameter even consumes bytes from the target text). Resolving them directly inside
		// the shared grammar rule would falsify every later application of the same rule.
		cloneRule := &r.Rule{Operator: r.Times, CodeChilds: &r.Rules{}, Childs: rule.Childs}
		*cloneRule.CodeChilds = append(*cloneRule.CodeChilds, *rule.CodeChilds...)
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
				pa.ruleExit(rule, skipSpaceRule, depth, nil, wasSdx, false)
				pa.Sdx = wasSdx
				return nil
			}
			localProductions = r.AppendArrayOfPossibleSequences(localProductions, newProductions) // Only append if all child rules matched.
		}
		// Repeat from "from" to "to" -> here it CAN be found:
		for i := from; i < to; i++ { // Repeat as often as possible.
			sdxBefore := pa.Sdx
			newProductions := pa.apply(newRule, skipSpaceRule, skippingSpaces, depth+1)
			if newProductions == nil {
				break
			}
			localProductions = r.AppendArrayOfPossibleSequences(localProductions, newProductions) // Only append if all child rules matched.
			if pa.Sdx == sdxBefore {
				break // The child rules matched without consuming anything. See case r.Repeat.
			}
		}
	case r.Optional:
		newProductions := pa.applyAsSequence(rule.Childs, skipSpaceRule, skippingSpaces, depth+1, rule.Pos)
		localProductions = r.AppendArrayOfPossibleSequences(localProductions, newProductions) // If not all child rules matched, newProductions is nil anyways.
	case r.Identifier: // This identifies another rule (and its index), it is basically a link: E.g. to the expression-rule which is at position 3: { "Identifier", "expression", 3 }
		if rule.Int < 0 || rule.Int >= len(*pa.agrammar) {
			panic("Unknown production name '" + rule.String + "'. It is used inside the grammar but never defined.")
		}
		newProductions := pa.applyAsSequence((*pa.agrammar)[rule.Int].Childs, skipSpaceRule, skippingSpaces, depth+1, rule.Pos)
		if newProductions == nil {
			pa.ruleExit(rule, skipSpaceRule, depth, nil, wasSdx, false)
			pa.Sdx = wasSdx
			return nil
		}
		localProductions = r.AppendArrayOfPossibleSequences(localProductions, newProductions)
	case r.Tag:
		newProductions := pa.applyAsSequence(rule.Childs, skipSpaceRule, skippingSpaces, depth+1, rule.Pos)
		if newProductions == nil {
			pa.ruleExit(rule, skipSpaceRule, depth, nil, wasSdx, false)
			pa.Sdx = wasSdx
			return nil
		}
		// Resolve name parameters into their code text. This changes the grammar rule itself,
		// but the resolution is idempotent, so all later applications just reuse the result.
		pa.resolveParameterToToken(rule.CodeChilds)
		// The matched childs get wrapped into a new Tag rule for the ASG. This is the only
		// grouping that the ASG keeps. Int contains the UID of the script for later caching.
		*localProductions = append(*localProductions, &r.Rule{Operator: r.Tag, Int: rule.Int, CodeChilds: rule.CodeChilds, Childs: newProductions, Pos: pa.Sdx})
	case r.Command:
		switch rule.String {
		case "whitespace":
			// Hand the Command :whitespace() up to the parent rule (the caller), because only
			// the parent can change its own skipSpaceRule. A copy is handed up instead of the
			// shared grammar rule, so its match position can be recorded without changing the grammar.
			localProductions = &r.Rules{{Operator: r.Command, String: rule.String, CodeChilds: rule.CodeChilds, Pos: pa.Sdx}}
		case "number":
			// :number(size, type) reads size bytes from the target text, interprets them as
			// type (a r.NumberType* constant) and creates a Number production from the value.
			// As a parameter of Times it defines the repeat count, standalone it e.g. allows
			// to parse TLV formats.
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
				pa.ruleExit(rule, skipSpaceRule, depth, nil, wasSdx, false)
				pa.Sdx = wasSdx
				return nil
			}
			if byteCount == 0 { // TODO: Automatically read the number as long as it is in the target text. Also implement negative number parsing for BCD and binary.
				panic("NOT IMPLEMENTED")
			}
			bytes := []byte(pa.Src[pa.Sdx : pa.Sdx+byteCount]) // An if would probably be slower.
			n := 0
			switch numberType { // TODO: Everything for signed numbers too!
			case r.NumberTypeLittleEndian:
				switch byteCount {
				case 1:
					n = int(bytes[0])
				case 2:
					n = int(binary.LittleEndian.Uint16(bytes))
				case 3, 4:
					// Odd sizes are padded up: for little endian the high zero
					// bytes go behind the data (Uint32/Uint64 need full width;
					// the bare slice panicked for 3 and 5...7 bytes).
					n = int(binary.LittleEndian.Uint32(padBytes(bytes, 4, false)))
				case 5, 6, 7, 8:
					n = int(binary.LittleEndian.Uint64(padBytes(bytes, 8, false)))
				default:
					panic(":number() needs byte count of 1 ... 8. ( e.g. :number(4) )")
				}
			case r.NumberTypeBigEndian:
				switch byteCount {
				case 1:
					n = int(bytes[0])
				case 2:
					n = int(binary.BigEndian.Uint16(bytes))
				case 3, 4:
					// For big endian the high zero bytes go in front of the data.
					n = int(binary.BigEndian.Uint32(padBytes(bytes, 4, true)))
				case 5, 6, 7, 8:
					n = int(binary.BigEndian.Uint64(padBytes(bytes, 8, true)))
				default:
					panic(":number() needs byte count of 1 ... 8. ( e.g. :number(4) )")
				}
			case r.NumberTypeBCD:
				s := hex.EncodeToString(bytes)
				if s[len(s)-1] == 'f' {
					s = s[:len(s)-1]
				}
				res, err := strconv.ParseUint(s, 10, 64)
				if err != nil {
					panic("Can not convert number: '" + string(bytes) + "'. Error: " + err.Error())
				}
				n = int(res)
			case r.NumberTypeASCII:
				res, err := strconv.ParseInt(string(bytes), 10, 64)
				if err != nil {
					panic("Can not parse int: '" + string(bytes) + "'")
				}
				n = int(res)
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
		case "script": // Int is reserved for UID for JS cache. // TODO: Maybe move upwards like :whitespace().
			// The script can be given inline as token or as the name of a production that contains the code.
			pa.resolveParameterToToken(rule.CodeChilds)
			resRule := pa.ps.HandleScriptRule(rule, localProductions, depth) // TODO: localProductions is empty here...
			if resRule != nil {
				scriptProductions := pa.apply(resRule, skipSpaceRule, skippingSpaces, depth+1)
				if scriptProductions == nil {
					pa.ruleExit(rule, skipSpaceRule, depth, nil, wasSdx, false)
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

	pa.ruleExit(rule, skipSpaceRule, depth, localProductions, wasSdx, false)
	return localProductions
}

// References ---------------

// references resolves the two kinds of links inside an a-grammar:
//   - productionReferences maps each production name to its current position inside the
//     grammar rules array. It is rebuilt on every correctReferencesAndIDs() call, because
//     the positions can change (e.g. through :include()).
//   - tagReferences maps each distinct tag code to its UID (used to cache the compiled JS).
//     These have to stay stable over multiple calls, so they are never rebuilt.
type references struct {
	productionReferences map[string]int
	tagReferences        map[string]int
	lastTag              int
}

func NewReferences() *references {
	var re references
	re.tagReferences = map[string]int{}
	re.lastTag = 0
	return &re
}

// collectProductionReferences records the array position of every production by name.
func (re *references) collectProductionReferences(rules *r.Rules) {
	if rules == nil {
		return
	}
	for i, rule := range *rules {
		if rule.Operator != r.Production {
			continue
		}
		if _, ok := re.productionReferences[rule.String]; ok {
			panic("Error: Rule " + rule.String + " is defined multiple times.")
		}
		re.productionReferences[rule.String] = i
	}
}

// correctReferencesAndIDs walks the whole a-grammar and fills in the two Int link values:
// Every Identifier gets the current array position of the production it names (-1 if that
// production does not exist), and every distinct Tag / :script() code gets its stable UID.
func (re *references) correctReferencesAndIDs(rules *r.Rules) {
	if rules == nil {
		return
	}
	clear := false
	if re.productionReferences == nil { // If this is nil, it is the outermost recursion of correctReferencesAndIDs(). This means, parameter rules holds all productions.
		re.productionReferences = map[string]int{}
		re.collectProductionReferences(rules)
		clear = true
	}
	for _, rule := range *rules {
		if rule.Operator == r.Identifier {
			// An unknown production name cannot be reported here, because it could still be
			// added later (e.g. by an :include()). So it is only marked with the invalid
			// position -1. Whoever really uses the Identifier has to check for that marker.
			if pos, ok := re.productionReferences[rule.String]; ok {
				rule.Int = pos
			} else {
				rule.Int = -1
			}
		} else if rule.Operator == r.Tag || (rule.Operator == r.Command && rule.String == "script") {
			var allCode string
			if len(*rule.CodeChilds) == 1 {
				allCode = (*rule.CodeChilds)[0].String
			} else {
				allCode = ""
				for _, child := range *rule.CodeChilds {
					allCode += child.String
				}
			}
			if pos, ok := re.tagReferences[allCode]; ok {
				rule.Int = pos
			} else {
				re.lastTag++
				re.tagReferences[allCode] = re.lastTag
				rule.Int = re.lastTag
			}
		}
		if rule.Childs != nil && len(*rule.Childs) > 0 {
			re.correctReferencesAndIDs(rule.Childs)
		}
		if rule.CodeChilds != nil && len(*rule.CodeChilds) > 0 {
			re.correctReferencesAndIDs(rule.CodeChilds)
		}
	}

	if clear { // The production positions must be recollected on the next call, because new productions could have been added in the meantime.
		re.productionReferences = nil
	}
}

// ---------------

// mergeTerminals combines neighbouring Token rules of the finished ASG into single Token
// rules (recursively). The parser creates one Token per matched char range or string, which
// would make the ASG unnecessarily large.
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
			// Not all rules have childs. E.g. a Number (from :number()) is a leaf like a Token.
			if (*productions)[i].Childs != nil && len(*(*productions)[i].Childs) > 0 {
				mergeTerminals((*productions)[i].Childs)
			}
		}
	}
}

// ParseWithAgrammar parses the target text srcCode with the given a-grammar and returns the
// resulting ASG (abstract semantic graph). fileName is where srcCode came from; it is used
// for messages and to resolve relative paths. If the a-grammar defines no :startRule(),
// (nil, nil) is returned: Nothing can be parsed then, which is fine for grammars that only
// consist of a :startScript().
func ParseWithAgrammar(agrammar *r.Rules, srcCode, fileName string, options *Parseropts) (res *r.Rules, e error) { // => (productions, error)
	defer func() {
		if err := recover(); err != nil {
			res = nil
			e = fmt.Errorf("%s", err)
		}
	}()

	startRule := r.GetStartRule(agrammar)
	if startRule == nil {
		// No start rule defined. Immediately return but this is no error. The :startScript() rule of the compiler has to do everything now.
		return nil, nil
	}

	var pa parser
	pa.agrammar = agrammar
	pa.Src = srcCode
	pa.Sdx = 0
	pa.opts = options
	pa.traceCount = 0
	pa.blockList = make(map[int]bool)
	pa.foundList = make(map[int]*r.Rules)
	pa.foundSdxList = make(map[int]int)
	pa.lastParsePosition = 0
	pa.fileName = filepath.Clean(fileName)
	pa.referencesCache = NewReferences()
	pa.referencesCache.correctReferencesAndIDs(pa.agrammar)
	pa.initialSpaces = &r.Rule{Operator: r.CharsOf, String: "\t\n\r "} // TODO: Make this configurable via JS.

	if UseFrozenScripts {
		pa.ps = newFrozenParserScript(&pa)
	} else {
		pa.ps = NewParserScript(&pa, options.PreventDefaultOutput)
	}

	// An index loop, not a range: an :include() appends the included grammar's
	// rules (with THEIR :include() commands) to *pa.agrammar, and a range over
	// the initial slice header would never visit them - nested includes were
	// silently ignored, leaving their productions undefined.
	for i := 0; i < len(*pa.agrammar); i++ {
		if rule := (*pa.agrammar)[i]; rule.Operator == r.Command {
			pa.applyCommand(rule)
		}
	}

	// The references were corrected above (and again after every :include()), so an
	// invalid position means the named start production really does not exist.
	if startRule.Int < 0 || startRule.Int >= len(*pa.agrammar) {
		panic("The production '" + startRule.String + "' of :startRule() was not found in the grammar.")
	}

	// For the parsing, the start rule is necessary. For the compilation not.
	newProductions := pa.apply((*pa.agrammar)[startRule.Int], pa.initialSpaces, false, 0)

	// Check if the position is at EOF at end of parsing. There can be spaces left, but otherwise its an error:
	if pa.initialSpaces != nil {
		pa.apply(pa.initialSpaces, pa.initialSpaces, true, 0) // Skip spaces.
	}
	if pa.Sdx < len(pa.Src) {
		panic(fmt.Sprintf("Not everything could be parsed. Last good parse position was %s\nCreated productions: %s", LinePosFromStrPos(string(pa.Src), pa.lastParsePosition), Shorten(newProductions.Serialize())))
	}

	mergeTerminals(newProductions)
	return newProductions, nil
}
