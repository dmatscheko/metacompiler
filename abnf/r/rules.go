package r

import (
	"fmt"
	"strings"
)

// ----------------------------------------------------------------------------
// Rule definition for parser and compiler

type Object = interface{}
type OperatorID int

const (
	Error OperatorID = iota // This marks an invalid command. Every operation that encounters such command, should return to its caller with error.
	Success
	// Group types:
	Sequence // Basic sequence of rules. Can be broken apart.
	Group    // A group that must not be broken apart.
	// Action types:
	Token  // A terminal symbol (fixed text that must be in the target text).
	Number // A plain number. Created e.g. by the inline command :number().
	Or     // Alternative rules. The first matching child wins.
	Optional
	Repeat
	Range   // A char range ("a"..."z" or "\x00"..b"\xff"). Int holds the RangeType* constant.
	Times   // A counted repetition (e.g. 3...5 ( X )). CodeChilds holds the count parameters.
	Tag     // The annotation rule that carries JS code. Int is reserved for the UID for caching the compiled code.
	Command // A parser command like :whitespace(). Int is reserved for the code UID of :script().
	CharOf  // Exactly one char out of String must be in the target text. Int holds CharType* flags.
	CharsOf // One or more chars out of String (in any order) must be in the target text. Int holds CharType* flags.
	Not     // Negative lookahead: matches (consuming nothing) exactly when its single child does NOT match.
	// Link types:
	Production // Int is reserved for the position of the Production inside the grammar rules.
	Identifier // Int is reserved for the position of the identified Production (-1 if unresolved).
)

func (id OperatorID) String() string {
	return [...]string{"Error", "Success", "Sequence", "Group", "Token", "Number", "Or", "Optional", "Repeat", "Range", "Times", "Tag", "Command", "CharOf", "CharsOf", "Not", "Production", "Identifier"}[id]
}

type Rule struct {
	Operator   OperatorID
	String     string // The text of Token, the name of Identifier | Production | Command, or the char set of CharOf | CharsOf. If a String is set anywhere else (e.g. in a Sequence), it can be handled like a comment and discarded.
	Int        int    // The value of Number, the production position of Identifier | Production, the range type of Range, or the code UID of Tag | Command :script().
	Pos        int    // The position in the source text where this Rule was defined (in a grammar) or where it has matched (in an ASG).
	Childs     *Rules // The child rules. Used by most Operators.
	CodeChilds *Rules // The parameters or the code. Only used when Operator == Tag | Command | Range | Times.
}

// Type of a Range String. JS-Mapping: abnf.rangeType
const (
	RangeTypeRune int = iota
	RangeTypeByte
)

// Flags for the Int field of CharOf and CharsOf. JS-Mapping: abnf.charType
// The zero value (CharTypeRune) is the plain rune based set match, so all
// serialized grammars from before these flags keep their meaning.
const (
	CharTypeRune    int = 0      // Match whole runes of the set (the default).
	CharTypeByte    int = 1 << 0 // Match single bytes of the set instead of runes.
	CharTypeNegated int = 1 << 1 // Match exactly the chars that are NOT in the set.
)

// Encoding of a :number() in the target text. JS-Mapping: abnf.numberType
const (
	NumberTypeLittleEndian int = iota
	NumberTypeBigEndian
	NumberTypeBCD
	NumberTypeASCII
)

// -----------------------------------------
// Multiple rules:

type Rules []*Rule

// func (rules *Rules) AppendIfNotEmpty(elems ...*Rule) {
// 	if len(elems) == 1 {
// 		el := elems[0]
// 		if el == nil || (len(*el.Childs) == 0 && (el.Operator == Group || el.Operator == Or || el.Operator == Sequence)) {
// 			return
// 		}
// 	}
// 	*rules = append(*rules, elems...)
// }

func (rules *Rules) Append(elems ...*Rule) {
	*rules = append(*rules, elems...)
}

// Appends one rule to target but dissolves basic Sequence groups into their childs.
func AppendPossibleSequence(target *Rules, source *Rule) *Rules {
	// This costs time and does not make the result that much smaller:
	// if source.Childs != nil && len(*source.Childs) == 0 && (source.Operator == Sequence || source.Operator == Group) {
	// 	return target
	// }
	if target == nil {
		target = &Rules{}
	}
	if source.Operator == Sequence { // || (source.Operator == Or && len(*source.Childs) == 1)
		// target.AppendIfNotEmpty(*source.Childs...)
		*target = append(*target, *source.Childs...)
	} else {
		// target.AppendIfNotEmpty(source)
		*target = append(*target, source)
	}
	return target
}

// Appends all rules of source to target but dissolves basic Sequence groups into their childs.
func AppendArrayOfPossibleSequences(target *Rules, source *Rules) *Rules {
	if source == nil {
		return target
	}
	for _, rule := range *source {
		target = AppendPossibleSequence(target, rule)
	}
	return target
}

// Serialize converts one rule into its Go literal form (as used in agrammar.go).
// The output can be deserialized by compiling it as Go code.
func (rule *Rule) Serialize() string {
	if rule == nil {
		return "<nil>"
	}
	res := "&r.Rule{"

	op := rule.Operator
	res += fmt.Sprintf("Operator:r.%s", op.String())

	if op == Token || op == Identifier || op == Production || op == Command || op == CharOf || op == CharsOf {
		res += fmt.Sprintf(", String:%q", rule.String)
	}
	if op == Number || op == Range || ((op == CharOf || op == CharsOf) && rule.Int != 0) {
		res += fmt.Sprintf(", Int:%d", rule.Int)
	}
	if rule.CodeChilds != nil && (op == Tag || op == Command || op == Range || op == Times) {
		res += ", CodeChilds:&r.Rules{"
		for i := range *rule.CodeChilds {
			if i > 0 {
				res += ", "
			}
			res += ((*rule.CodeChilds)[i]).Serialize()
		}
		res += "}"
	}
	if rule.Childs != nil && (op == Tag || op == Identifier || op == Production || op == Group || op == Sequence || op == Or || op == Optional || op == Repeat || op == Not) {
		res += ", Childs:&r.Rules{"
		for i := range *rule.Childs {
			if i > 0 {
				res += ", "
			}
			res += ((*rule.Childs)[i]).Serialize()
		}
		res += "}"
	}
	if !(op == Token || op == Number || op == Identifier || op == Production || op == Tag || op == Command || op == Range || op == Times || op == Group || op == Sequence || op == Or || op == Optional || op == Repeat || op == CharOf || op == CharsOf || op == Not) {
		panic("wrong rule type: " + op.String())
	}

	// res += ", Pos: " + strconv.Itoa(rule.Pos)

	res += "}"
	return res
}

// Serialize converts the rules into their Go literal form (as used in agrammar.go).
func (rules *Rules) Serialize() string {
	if rules == nil {
		return "<nil>"
	}
	res := "&r.Rules{"
	for i := range *rules {
		r := (*rules)[i]
		if i > 0 {
			res += ", "
		}
		res += r.Serialize()
	}
	res += "}"
	return res
}

// SerializePretty is Serialize() laid out for reading: every structural brace
// closes on its own line, indented by its nesting depth. Braces inside string
// literals (e.g. the JS of a tag) are left untouched. This is the layout of the
// example dump in the README.
func (rules *Rules) SerializePretty() string {
	return prettyBraces(rules.Serialize())
}

// prettyBraces rewrites a compact Go-literal string so that each closing brace
// sits on its own line at (depth-1)*4 spaces, tracking string literals so a '}'
// inside a quoted string is copied verbatim.
func prettyBraces(s string) string {
	var b strings.Builder
	depth := 0
	inString := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inString {
			b.WriteByte(c)
			if c == '\\' && i+1 < len(s) { // Escape: copy the next byte verbatim.
				i++
				b.WriteByte(s[i])
			} else if c == '"' {
				inString = false
			}
			continue
		}
		switch c {
		case '"':
			inString = true
			b.WriteByte(c)
		case '{':
			depth++
			b.WriteByte(c)
		case '}':
			depth--
			b.WriteByte('\n')
			for k := 0; k < depth*4; k++ {
				b.WriteByte(' ')
			}
			b.WriteByte(c)
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// ToString converts one rule into a short, human readable form: Child rules are
// abbreviated with [...]. Use Serialize() to see everything.
func (rule *Rule) ToString() string {
	if rule == nil {
		return "<nil>"
	}
	res := "Rule{"

	op := rule.Operator
	res += fmt.Sprintf("Operator:%s", op.String())

	if op == Token || op == Identifier || op == Production || op == Command || op == CharOf || op == CharsOf {
		res += fmt.Sprintf(", String:%q", rule.String)
	}
	if op == Identifier || op == Number || op == Range || ((op == CharOf || op == CharsOf) && rule.Int != 0) {
		res += fmt.Sprintf(", Int:%d", rule.Int)
	}
	if rule.CodeChilds != nil && (op == Tag || op == Command || op == Range || op == Times) {
		res += ", CodeChilds:[...]"
	}
	if rule.Childs != nil && (op == Tag || op == Identifier || op == Production || op == Group || op == Sequence || op == Or || op == Optional || op == Repeat || op == Not) {
		res += ", Childs:[...]"
	}
	if !(op == Token || op == Number || op == Identifier || op == Production || op == Tag || op == Command || op == Range || op == Times || op == Group || op == Sequence || op == Or || op == Optional || op == Repeat || op == CharOf || op == CharsOf || op == Not) {
		res += fmt.Sprintf(", String:%q", rule.String)
		res += fmt.Sprintf(", Int:%d", rule.Int)
		if rule.CodeChilds != nil {
			res += ", CodeChilds:Rules{"
			for i := range *rule.CodeChilds {
				if i > 0 {
					res += ", "
				}
				res += ((*rule.CodeChilds)[i]).ToString()
			}
			res += "}"
		}
		if rule.Childs != nil {
			res += ", Childs:Rules{"
			for i := range *rule.Childs {
				if i > 0 {
					res += ", "
				}
				res += ((*rule.Childs)[i]).ToString()
			}
			res += "}"
		}
		res += "}"
		panic("wrong rule type: " + res)
	}

	// res += ", Pos: " + strconv.Itoa(rule.Pos)

	res += "}"
	return res
}

// ToString converts the rules into a short, human readable form.
func (rules *Rules) ToString() string {
	if rules == nil {
		return "<nil>"
	}
	res := "[]{"
	for i := range *rules {
		r := (*rules)[i]
		if i > 0 {
			res += ", "
		}
		res += r.ToString()
	}
	res += "}"
	return res
}

// GetStartRule returns the Identifier that points to the top level production for the
// parser (defined via :startRule()), or nil if the a-grammar does not define one.
func GetStartRule(aGrammar *Rules) *Rule {
	if aGrammar == nil {
		return nil
	}
	for _, rule := range *aGrammar {
		if rule.Operator == Command && rule.String == "startRule" {
			return (*rule.CodeChilds)[0]
		}
	}
	return nil
}

// GetStartScript returns the JS code that controls the compile step (defined via
// :startScript()), converted into a Tag-like rule, or nil if there is none.
func GetStartScript(aGrammar *Rules) *Rule {
	if aGrammar == nil {
		return nil
	}
	for _, rule := range *aGrammar {
		if rule.Operator == Command && rule.String == "startScript" {
			return &Rule{Operator: Tag, CodeChilds: rule.CodeChilds} // Convert to something Tag-like.
		}
	}
	return nil
}

// GetOrigin returns the file the a-grammar was compiled from (stamped as an
// :origin() command by CompileASG), or "" for grammars without one (e.g. the
// embedded serialized grammars or grammars built directly by scripts).
func GetOrigin(aGrammar *Rules) string {
	if aGrammar == nil {
		return ""
	}
	for _, rule := range *aGrammar {
		if rule.Operator == Command && rule.String == "origin" {
			return (*rule.CodeChilds)[0].String
		}
	}
	return ""
}

// GetTitle returns the title of the a-grammar (defined via :title()), or nil.
func GetTitle(aGrammar *Rules) *Rule {
	if aGrammar == nil {
		return nil
	}
	for _, rule := range *aGrammar {
		if rule.Operator == Command && rule.String == "title" {
			return (*rule.CodeChilds)[0]
		}
	}
	return nil
}

// GetDescription returns the description of the a-grammar (defined via :description()), or nil.
func GetDescription(aGrammar *Rules) *Rule {
	if aGrammar == nil {
		return nil
	}
	for _, rule := range *aGrammar {
		if rule.Operator == Command && rule.String == "description" {
			return (*rule.CodeChilds)[0]
		}
	}
	return nil
}
