package r

import (
	"fmt"
)

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
	Command
	// Link types:
	Production
	Identifier
)

func (id OperatorID) String() string {
	return [...]string{"Error", "Success", "Sequence", "Group", "Token", "Or", "Optional", "Repeat", "Range", "SkipSpace", "Tag", "Command", "Production", "Identifier"}[id]
}

type Rule struct {
	Operator   OperatorID
	String     string // Only used when Operator == seq.Token || seq.Ident || seq.Production || seq.Command. If a String is in e.g. seq.Sequence, then this string can be handled like a comment and discarded.
	Int        int    // Only used when Operator == seq.Ident || seq.Production
	Bool       bool   // Only used when Operator == seq.SkipSpaces
	Pos        int    // The position where this Rule has matched.
	Childs     *Rules // Used by most Operators
	CodeChilds *Rules // Only used when Operator == seq.Tag || seq.Command
}

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

// Appends a Sequence but dissolves basic SEQUENCE groups
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

// Appends a Sequence but dissolves basic SEQUENCE groups
func AppendArrayOfPossibleSequences(target *Rules, source *Rules) *Rules {
	if source == nil {
		return target
	}
	for _, rule := range *source {
		target = AppendPossibleSequence(target, rule)
	}
	return target
}

func (rule *Rule) Serialize() string {
	if rule == nil {
		return "<nil>"
	}
	res := "&r.Rule{"

	op := rule.Operator
	res += fmt.Sprintf("Operator: r.%s", op.String())

	if op == Token || op == Identifier || op == Production || op == Command {
		res += fmt.Sprintf(", String: %q", rule.String)
	}
	if op == Identifier || op == Production {
		res += fmt.Sprintf(", Int: %d", rule.Int)
	}
	if op == SkipSpace {
		res += fmt.Sprintf(", Bool: %t", rule.Bool)
	}
	if rule.CodeChilds != nil && (op == Tag || op == Command) {
		res += ", CodeChilds: &r.Rules{"
		for i := range *rule.CodeChilds {
			if i > 0 {
				res += ", "
			}
			res += ((*rule.CodeChilds)[i]).Serialize()
		}
		res += "}"
	}
	if rule.Childs != nil && (op == Tag || op == Range || op == Identifier || op == Production || op == Group || op == Sequence || op == Or || op == Optional || op == Repeat) {
		res += ", Childs: &r.Rules{"
		for i := range *rule.Childs {
			if i > 0 {
				res += ", "
			}
			res += ((*rule.Childs)[i]).Serialize()
		}
		res += "}"
	}
	if !(op == Token || op == Identifier || op == Production || op == Tag || op == Command || op == Range || op == SkipSpace || op == Group || op == Sequence || op == Or || op == Optional || op == Repeat) {
		panic("wrong rule type: " + op.String())
	}
	// if op == Tag {
	// 	res += ", ID: " + strconv.Itoa(rule.ID)
	// }

	// res += ", Pos: " + strconv.Itoa(rule.Pos)

	res += "}"
	return res
}

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

func GetProductions(aGrammar *Rules) *Rules {
	if aGrammar == nil {
		return nil
	}
	for i := range *aGrammar {
		rule := (*aGrammar)[i]
		if rule.Operator == Sequence {
			return rule.Childs
		}
	}
	return nil
}

func GetStartRule(aGrammar *Rules) *Rule {
	if aGrammar == nil {
		return nil
	}
	for i := range *aGrammar {
		rule := (*aGrammar)[i]
		if rule.Operator == Identifier {
			return rule
		}
	}
	return nil
}

func GetProlog(aGrammar *Rules) *Rule {
	if aGrammar == nil {
		return nil
	}
	for i := range *aGrammar {
		rule := (*aGrammar)[i]
		if rule.Operator == Sequence {
			return nil
		} else if rule.Operator == Tag {
			return rule
		}
	}
	return nil
}

func GetTitle(aGrammar *Rules) *Rule {
	if aGrammar == nil {
		return nil
	}
	for i := range *aGrammar {
		rule := (*aGrammar)[i]
		if rule.Operator == Sequence {
			return nil
		} else if rule.Operator == Token {
			return rule
		}
	}
	return nil
}

func GetDescription(aGrammar *Rules) *Rule {
	if aGrammar == nil {
		return nil
	}
	afterProductions := false
	for i := range *aGrammar {
		rule := (*aGrammar)[i]
		if rule.Operator == Sequence {
			afterProductions = true
		} else if rule.Operator == Token {
			if afterProductions {
				return rule
			}
		}
	}
	return nil
}
