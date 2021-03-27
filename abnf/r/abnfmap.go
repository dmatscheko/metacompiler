package r

// ----------------------------------------------------------------------------
// ABNF mapping for LLVM IR

var AbnfFuncMap = map[string]Object{
	"arrayToRules": func(rules *Rules) *Rules {
		return rules
	},
	"newRule": func(Operator OperatorID, String string, Int int, Pos int, Childs *Rules, CodeChilds *Rules) *Rule {
		return &Rule{Operator: Operator, String: String, Int: Int, Pos: Pos, Childs: Childs, CodeChilds: CodeChilds}
	},
	"newToken": func(String string, Pos int) *Rule {
		return &Rule{Operator: Token, String: String, Pos: Pos}
	},
	"newNumber": func(Int int, Pos int) *Rule {
		return &Rule{Operator: Number, Int: Int, Pos: Pos}
	},
	"newIdentifier": func(String string, Pos int) *Rule { // This is only the link. Int is reserved for the position of the identified Production.
		return &Rule{Operator: Identifier, String: String, Pos: Pos}
	},
	"newProduction": func(String string, Childs *Rules, Pos int) *Rule { // This is the holder of the Production. This is where the link points to. Int is reserved for the position of the Production.
		return &Rule{Operator: Production, String: String, Childs: Childs, Pos: Pos}
	},
	"newTag": func(CodeChilds *Rules, Childs *Rules, Pos int) *Rule { // Int is reserved for the UID for caching.
		return &Rule{Operator: Tag, CodeChilds: CodeChilds, Childs: Childs, Pos: Pos}
	},
	// Command :number() -> Int == numberType.LittleEndian | Int == numberType.BigEndian | Int == numberType.BCD
	"newCommand": func(String string, CodeChilds *Rules, Pos int) *Rule {
		if CodeChilds != nil && len(*CodeChilds) == 0 {
			CodeChilds = nil
		}
		return &Rule{Operator: Command, String: String, CodeChilds: CodeChilds, Pos: Pos}
	},
	"newRepetition": func(Childs *Rules, Pos int) *Rule {
		return &Rule{Operator: Repeat, Childs: Childs, Pos: Pos}
	},
	"newOption": func(Childs *Rules, Pos int) *Rule {
		return &Rule{Operator: Optional, Childs: Childs, Pos: Pos}
	},
	"newGroup": func(Childs *Rules, Pos int) *Rule {
		return &Rule{Operator: Group, Childs: Childs, Pos: Pos}
	},
	"newSequence": func(Childs *Rules, Pos int) *Rule {
		return &Rule{Operator: Sequence, Childs: Childs, Pos: Pos}
	},
	"newAlternative": func(Childs *Rules, Pos int) *Rule {
		return &Rule{Operator: Or, Childs: Childs, Pos: Pos}
	},
	// Int == rangeType.Rune | Int == rangeType.Byte
	"newRange": func(CodeChilds *Rules, Int int, Pos int) *Rule { // TODO: Why do the parameter have to be in that (wrong) order?
		return &Rule{Operator: Range, Int: Int, CodeChilds: CodeChilds, Pos: Pos}
	},
	"newTimes": func(CodeChilds *Rules, Childs *Rules, Pos int) *Rule {
		return &Rule{Operator: Times, CodeChilds: CodeChilds, Childs: Childs, Pos: Pos}
	},
	"newCharOf": func(String string, Pos int) *Rule {
		return &Rule{Operator: CharOf, String: String, Pos: Pos}
	},
	"newCharsOf": func(String string, Pos int) *Rule {
		return &Rule{Operator: CharsOf, String: String, Pos: Pos}
	},
	"serializeRule": func(rule *Rule) string {
		return rule.Serialize()
	},
	"serializeRules": func(rules *Rules) string {
		return rules.Serialize()
	},
	"toStringRule": func(rule *Rule) string {
		return rule.ToString()
	},
	"toStringRules": func(rules *Rules) string {
		return rules.ToString()
	},

	"getStartRule":   GetStartRule,
	"getStartScript": GetStartScript,
	"getTitle":       GetTitle,
	"getDescription": GetDescription,

	"oid": map[string]OperatorID{
		"Error":   Error,
		"Success": Success,
		// Groups types:
		"Sequence": Sequence,
		"Group":    Group,
		// Action types:
		"Token":    Token,
		"Number":   Number,
		"Or":       Or,
		"Optional": Optional,
		"Repeat":   Repeat,
		"Range":    Range,
		"Times":    Times,
		"Tag":      Tag,
		"Command":  Command,
		// Link types:
		"Production": Production,
		"Identifier": Identifier,
	},

	// Type of a Range String.
	"rangeType": map[string]int{
		"Rune": RangeTypeRune,
		"Byte": RangeTypeByte,
	},

	// Encoding of a :number() in the target text.
	"numberType": map[string]int{
		"LittleEndian": NumberTypeLittleEndian,
		"BigEndian":    NumberTypeBigEndian,
		"BCD":          NumberTypeBCD,
		"ASCII":        NumberTypeASCII,
	},
}
