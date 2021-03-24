package r

// ----------------------------------------------------------------------------
// ABNF mapping for LLVM IR

var AbnfFuncMap = map[string]Object{
	"newToken": func(String string, Pos int) *Rule {
		return &Rule{Operator: Token, String: String, Pos: Pos}
	},
	"newNumber": func(Int int, Pos int) *Rule {
		return &Rule{Operator: Number, Int: Int, Pos: Pos}
	},
	"newIdentifier": func(String string, Int int, Pos int) *Rule { // This is only the link.
		return &Rule{Operator: Identifier, String: String, Int: Int, Pos: Pos}
	},
	"newProduction": func(String string, Int int, Childs *Rules, Pos int) *Rule { // This is the holder of the production. This is where the link points to.
		return &Rule{Operator: Production, String: String, Int: Int, Childs: Childs, Pos: Pos}
	},
	"newTag": func(CodeChilds *Rules, Childs *Rules, Pos int) *Rule {
		return &Rule{Operator: Tag, CodeChilds: CodeChilds, Childs: Childs, Pos: Pos}
	},
	"newCommand": func(String string, CodeChilds *Rules, Pos int) *Rule {
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
	// Int 0 == Rune-Comparison, Int 1 = Byte-Comparison
	"newRange": func(CodeChilds *Rules, Int int, Pos int) *Rule {
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
	"newRule": func(Operator OperatorID, String string, Int int, Pos int, Childs *Rules, CodeChilds *Rules) *Rule {
		return &Rule{Operator: Operator, String: String, Int: Int, Pos: Pos, Childs: Childs, CodeChilds: CodeChilds}
	},
	"arrayToRules": func(rules *Rules) *Rules {
		return rules
	},
	"serializeRule": func(rule *Rule) string {
		return rule.Serialize()
	},

	"getStartRule":   GetStartRule,
	"getProductions": GetProductions,
	"getProlog":      GetStartScript,
	"getTitle":       GetTitle,
	"getDescription": GetDescription,

	"serializeRules": func(rules *Rules) string {
		return rules.Serialize()
	},

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
	},
}
