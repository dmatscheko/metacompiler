package r

type Object = interface{}
type OperatorID int

const (
	Error OperatorID = iota // This marks an invalid command. Every operation that encounters such command, should return to its caller with error.
	Success
	// Groups types:
	Sequence // Basic sequence of objects. Can be broken apart.
	Group    // A group that must not be broken apart.
	// Action types:
	Token
	Or
	Optional
	Repeat
	Range
	SkipSpace
	Tag
	Factor
	// Link types:
	Production
	Ident
)

func (id OperatorID) String() string {
	return [...]string{"ERROR", "SUCCESS", "SEQUENCE", "GROUP", "TOKEN", "OR", "OPTIONAL", "REPEAT", "RANGE", "SKIPSPACE", "TAG", "FACTOR", "PRODUCTION", "IDENT"}[id]
}

// TODO: When reducing the size of Rule: Maybe always convert runes into strings here...
type Rule struct {
	Operator  OperatorID
	String    string // Only used when Operator == seq.Token || seq.Ident || seq.Production. If a String is in e.g. seq.Sequence, then this string can be handled like a comment and discarded.
	Int       int    // Only used when Operator == seq.Ident || seq.Production
	Bool      bool   // Only used when Operator == seq.SkipSpaces
	Rune      rune   // Only used when Operator == seq.Factor. Maybe use r.String here too if its not too much slower.
	Pos       int    // The position where this Rule has matched.
	ID        int    // Used for the block list, when applying the rule as grammar.
	Childs    *Rules // Used by most Operators
	TagChilds *Rules // Only used when Operator == seq.Tag
}

func (rule *Rule) CloneShallow() *Rule {
	var newRule = *rule
	return &newRule
}

// func (rule *Rule) CloneDeep() *Rule {
// 	// TODO:
// 	var newRule = *rule
// 	return &newRule
// }

type Rules []Rule

func (rules *Rules) Append(elems ...Rule) {
	*rules = append(*rules, elems...)
}

// func (rules *Rules) Pop() *Rule {
// 	if len(*rules) <= 0 {
// 		return nil
// 	}
// 	rule := &(*rules)[len(*rules)-1]
// 	return rule
// }

// Appends a Sequence but dissolves basic SEQUENCE groups
func AppendPossibleSequence(target *Rules, source *Rule) *Rules {
	if target == nil {
		target = &Rules{}
	}
	if source.Operator == Sequence {
		*target = append(*target, *source.Childs...)
	} else {
		*target = append(*target, *source)
	}
	return target
}

// Appends a Sequence but dissolves basic SEQUENCE groups
func AppendArrayOfPossibleSequences(target *Rules, source *Rules) *Rules {
	if source == nil {
		return target
	}
	for _, rule := range *source {
		target = AppendPossibleSequence(target, &rule)
	}
	return target
}

// TODO: document the JS methods and vars below

var EbnfFuncMap = map[string]Object{ // The LLVM function will be inside such a map.
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
	"newRange": func(Childs *Rules, Pos int) *Rule { // TODO: implement.
		return &Rule{Operator: Group, Childs: Childs, Pos: Pos}
	},

	"newRule": func(Operator OperatorID, String string, Int int, Bool bool, Rune rune, Pos int, Childs *Rules, TagChilds *Rules) *Rule {
		return &Rule{Operator: Operator, String: String, Int: Int, Bool: Bool, Rune: Rune, Pos: Pos, Childs: Childs, TagChilds: TagChilds}
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
		// "Factor": Factor, // This one is not needed
		// Link types:
		"Production": Production,
		"Ident":      Ident,
	},
}
