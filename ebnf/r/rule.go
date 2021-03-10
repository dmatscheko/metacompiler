package r

// TODO: move to ints later:
// type operatorID int

type Object = interface{}
type OperatorID int

const (
	Success OperatorID = iota // This marks an invalid command. Every operation that encounters such command, should return to its caller with error.
	Error
	// Groups types:
	Sequence // Basic sequence of objects. Can be broken apart.
	Group    // A group that must not be broken apart.
	// Action types:
	Terminal
	Or
	Optional
	Repeat
	Range
	SkipSpaces
	Tag
	Factor
	// Link types:
	Production
	Ident
)

func (id OperatorID) String() string {
	return [...]string{"SUCCESS", "ERROR", "SEQUENCE", "GROUP", "TERMINAL", "OR", "OPTIONAL", "REPEAT", "RANGE", "SKIPSPACES", "TAG", "FACTOR", "PRODUCTION", "IDENT"}[id]
}

// TODO: When reducing the size of Rule: Maybe always convert runes into strings here...
type Rule struct {
	Operator  OperatorID
	String    string // Only used when Operator == seq.Terminal || seq.Ident || seq.Production. If a String is in e.g. seq.Sequence, then this string can be handled like a comment and discarded.
	Int       int    // Only used when Operator == seq.Ident || seq.Production
	Bool      bool   // Only used when Operator == seq.SkipSpaces
	Rune      rune   // Only used when Operator == seq.Factor
	Pos       int    // The position where this Rule has matched.
	ID        int    // Used for the block list, when applying the rule as grammar.
	Childs    []Rule // Used by most Operators
	TagChilds []Rule // Only used when Operator == seq.Tag
}

// Appends a Sequence but dissolves basic SEQUENCE groups
func AppendPossibleSequence(target []Rule, source Rule) []Rule {
	if source.Operator == Sequence {
		target = append(target, source.Childs...)
	} else {
		target = append(target, source)
	}
	return target
}

// Appends a Sequence but dissolves basic SEQUENCE groups
func AppendArrayOfPossibleSequences(target []Rule, source *[]Rule) []Rule {
	if source == nil {
		return target
	}
	for _, rule := range *source {
		target = AppendPossibleSequence(target, rule)
	}
	return target
}
