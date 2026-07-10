package r

import "strconv"

// ----------------------------------------------------------------------------
// Scripting subsystem mapping for the a-grammar rules
//
// This map is exposed to JS as the object 'abnf'. It contains the builder,
// serializer and getter functions plus the constants that are needed to create
// and inspect a-grammars from within tag scripts.

// tokenString returns the token text of v: the preferred form is the Token *Rule
// itself, a plain string is accepted for backward compatibility (see newCharOf).
func tokenString(v Object) string {
	switch t := v.(type) {
	case *Rule:
		return t.String
	case string:
		return t
	}
	panic("newCharOf/newCharsOf: the set must be a Token rule or a string")
}

var AbnfFuncMap = map[string]Object{
	// Converts a plain JS array of rules into a Go *Rules value. The conversion itself
	// is done by the JS runtime when it maps the JS argument onto the Go parameter.
	"arrayToRules": func(rules *Rules) *Rules {
		return rules
	},
	"newRule": func(Operator OperatorID, String string, Int int, Pos int, Childs *Rules, CodeChilds *Rules) *Rule {
		return &Rule{Operator: Operator, String: String, Int: Int, Pos: Pos, Childs: Childs, CodeChilds: CodeChilds}
	},
	"newToken": func(String string, Pos int) *Rule {
		return &Rule{Operator: Token, String: String, Pos: Pos}
	},
	// newTokenEscaped takes the still escaped source text and resolves the escapes on
	// the Go side. The raw result may contain non UTF8 bytes (e.g. from a byte set
	// token like '\xff') that must never travel through the JS engine: goja would
	// replace them with U+FFFD on the way.
	"newTokenEscaped": func(String string, Pos int) *Rule {
		s, err := Unescape(String)
		if err != nil {
			panic("newTokenEscaped: " + err.Error() + " in " + strconv.Quote(String))
		}
		return &Rule{Operator: Token, String: s, Pos: Pos}
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
	// String is the command name (without ':'), CodeChilds holds the parameters.
	// E.g. for :number(size, type), CodeChilds holds two Number rules (type is a numberType.* constant).
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
	// CodeChilds must hold the two Token [from, to]. Int is the range type (rangeType.Rune | rangeType.Byte).
	"newRange": func(CodeChilds *Rules, Int int, Pos int) *Rule {
		return &Rule{Operator: Range, Int: Int, CodeChilds: CodeChilds, Pos: Pos}
	},
	"newTimes": func(CodeChilds *Rules, Childs *Rules, Pos int) *Rule {
		return &Rule{Operator: Times, CodeChilds: CodeChilds, Childs: Childs, Pos: Pos}
	},
	// Int holds charType flags (charType.Rune | charType.Byte | charType.Negated).
	// The set should be passed as the Token *Rule itself (not as its String): the rule
	// pointer passes through the JS engine untouched, while a raw Go string with non
	// UTF8 bytes (like a @b set "\xc3") would be mangled by goja's UTF16 conversion.
	"newCharOf": func(Token Object, Int int, Pos int) *Rule {
		return &Rule{Operator: CharOf, String: tokenString(Token), Int: Int, Pos: Pos}
	},
	"newCharsOf": func(Token Object, Int int, Pos int) *Rule {
		return &Rule{Operator: CharsOf, String: tokenString(Token), Int: Int, Pos: Pos}
	},
	// Childs must hold exactly one rule; the Not matches (with zero width) when it does not.
	"newNot": func(Childs *Rules, Pos int) *Rule {
		return &Rule{Operator: Not, Childs: Childs, Pos: Pos}
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
		// Group types:
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
		"CharOf":   CharOf,
		"CharsOf":  CharsOf,
		"Not":      Not,
		// Link types:
		"Production": Production,
		"Identifier": Identifier,
	},

	// Type of a Range String.
	"rangeType": map[string]int{
		"Rune": RangeTypeRune,
		"Byte": RangeTypeByte,
	},

	// Flags for the Int field of CharOf/CharsOf (combine with |).
	"charType": map[string]int{
		"Rune":    CharTypeRune,
		"Byte":    CharTypeByte,
		"Negated": CharTypeNegated,
	},

	// Encoding of a :number() in the target text.
	"numberType": map[string]int{
		"LittleEndian": NumberTypeLittleEndian,
		"BigEndian":    NumberTypeBigEndian,
		"BCD":          NumberTypeBCD,
		"ASCII":        NumberTypeASCII,
	},
}
