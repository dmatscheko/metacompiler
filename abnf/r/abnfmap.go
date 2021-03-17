package r

var AbnfFuncMap = map[string]Object{
	"newToken": func(String string, Pos int) *Rule {
		return &Rule{Operator: Token, String: String, Pos: Pos}
	},
	"newName": func(String string, Int int, Pos int) *Rule { // This is only the link.
		return &Rule{Operator: Ident, String: String, Int: Int, Pos: Pos}
	},
	"newProduction": func(String string, Int int, Childs *Rules, Pos int) *Rule { // This is the holder of the production. This is where the link points to.
		return &Rule{Operator: Production, String: String, Int: Int, Childs: Childs, Pos: Pos}
	},
	"newTag": func(TagChilds *Rules, Childs *Rules, Pos int) *Rule {
		return &Rule{Operator: Tag, TagChilds: TagChilds, Childs: Childs, Pos: Pos}
	},
	"newSkipSpace": func(Bool bool, Pos int) *Rule {
		return &Rule{Operator: SkipSpace, Bool: Bool, Pos: Pos}
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
	"newRange": func(Childs *Rules, Pos int) *Rule {
		return &Rule{Operator: Range, Childs: Childs, Pos: Pos}
	},
	"newRule": func(Operator OperatorID, String string, Int int, Bool bool, Rune rune, Pos int, Childs *Rules, TagChilds *Rules) *Rule {
		return &Rule{Operator: Operator, String: String, Int: Int, Bool: Bool, Pos: Pos, Childs: Childs, TagChilds: TagChilds}
	},
	"arrayToRules": func(rules *Rules) *Rules {
		return rules
	},
	"serializeRule": func(rule *Rule) string {
		return rule.Serialize()
	},

	"getStartRule":   GetStartRule,
	"getProductions": GetProductions,
	"getProlog":      GetProlog,
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
		"Token":     Token,
		"Or":        Or,
		"Optional":  Optional,
		"Repeat":    Repeat,
		"Range":     Range,
		"SkipSpace": SkipSpace,
		"Tag":       Tag,
		// Link types:
		"Production": Production,
		"Ident":      Ident,
	},
}
