package r

// TODO: move to ints later:
// type operatorID int

// const ( // iota is reset to 0
// 	basic operatorID = iota // can be broken apart
// 	group                   // a group, but nils can be deleted
// 	fixed                   // a group where the position is imporant. keep nil elements in this group. (Mostly when it is an operator-sequence rather than a simple group-sequence)
// 	// "TERMINAL" ...
// 	or
// )

type Object = interface{}
type OperatorID string

const (
	Invalid OperatorID = "INVALID" // This marks an invalid command. Every operation that encounters such command, should return to its caller with error.
	// Groups types:
	Sequence OperatorID = "SEQUENCE" // Basic sequence of objects. Can be broken apart.
	Group    OperatorID = "GROUP"    // A group that must not be broken apart.
	// Action types:
	Terminal   OperatorID = "TERMINAL"
	Or         OperatorID = "OR"
	Ident      OperatorID = "IDENT"
	Optional   OperatorID = "OPTIONAL"
	Repeat     OperatorID = "REPEAT"
	Range      OperatorID = "RANGE"
	SkipSpaces OperatorID = "SKIPSPACES"
	Tag        OperatorID = "TAG"
	// Special operator:
	Production OperatorID = "PRODUCTION"
	Factor     OperatorID = "FACTOR"
)

// TODO: This should be called Command
type Rule struct {
	Operator  OperatorID
	String    string // Only used when Operator == seq.Terminal || seq.Ident || seq.Production. If a String is in seq.Basic, then this string can be handles like a comment and discarded.
	Int       int    // Only used when Operator == seq.Ident || seq.Production (at production it is probably not necessary)
	Bool      bool   // Only used when Operator == seq.SkipSpaces
	Rune      rune   // Only used when Operator == seq.Factor  // TODO: Maybe always convert runes into strings...
	Pos       int    // The position where this Sequence has matched.
	Childs    []Rule // Used by most Operators
	TagChilds []Rule // Only used when Operator == seq.Tag
}

/*
// For compiler:
const (
	// Groups types:
	// Basic OperatorID = "SEQUENCE" // Basic sequence of objects. Can be broken apart
	// Group OperatorID = "GROUP"    // A group, but nils can be deleted
	Fixed OperatorID = "FIXED" // A group where the position is imporant. keep nil elements in this group. (Mostly when it is an operator-sequence rather than a simple group-sequence)
)

// TODO: This can be called Sequence for real
type Sequence struct {
	Operator OperatorID
	Objects  []Object
}
*/

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
func AppendArrayOfPossibleSequences(target []Rule, source []Rule) []Rule {
	for i := 0; i < len(source); i++ {
		target = AppendPossibleSequence(target, source[i])
	}
	return target
}
