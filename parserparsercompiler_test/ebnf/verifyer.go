package ebnf

import (
	"fmt"

	"./r"
)

// -------------------------------------------------------- verifier for ebnfparser by DMA -----

var reachedNames map[string]bool

func (ep *ebnfParser) collectReachableNames(production r.Rule) {
	for _, child := range production.Childs {
		if child.Operator == r.Ident {
			i := child.Int
			ii := ep.ididx[i]
			nextProduction := ep.grammar.Productions[ii]
			if nextProduction.Operator == r.Production {
				if reachedNames[nextProduction.String] {
					continue
				}
				reachedNames[nextProduction.String] = true
			} else {
				panic(fmt.Sprintf("Not a production: '%s' at position %d.", nextProduction.String, nextProduction.Pos))
			}
			ep.collectReachableNames(nextProduction)
		} else if len(child.Childs) > 0 {
			ep.collectReachableNames(child)
		}
	}
}

// Checks if all defined productiona are used.
func (ep *ebnfParser) verifyAllNamesUsed() {
	reachedNames = make(map[string]bool)

	startName := ""
	// Get name of start production.
	elem, ok := ep.grammar.Extras["start"]
	if ok {
		startName = elem.String
	} else {
		panic("No start production defined.")
	}

	// Get start production.
	startProduction := r.Rule{Operator: r.Invalid}
	for _, elem := range ep.grammar.Productions {
		if elem.Operator == r.Production && elem.String == startName {
			startProduction = elem
			break
		}
	}

	if startProduction.Operator != r.Production {
		panic(fmt.Sprintf("Defined start production (%s) not found.", startName))
	}
	reachedNames[startProduction.String] = true
	ep.collectReachableNames(startProduction)

	for _, name := range ep.idents {
		if !reachedNames[name] {
			panic(fmt.Sprintf("Name '%s' defined but not used (therefore not reachable).", name))
		}
	}
}

// Checks if there is a production defined for all used names.
func (ep *ebnfParser) verifyAllUsedNamesDefined() {
	k := -1
	for i, idx := range ep.ididx {
		if idx == -1 {
			k = i
			break
		}
	}
	if k != -1 {
		ep.invalid(fmt.Sprintf("Invalid EBNF (undefined: %s)", ep.idents[k]), 0)
		return
	}
}

func (ep *ebnfParser) verifyGrammar() {
	ep.verifyAllUsedNamesDefined()
	ep.verifyAllNamesUsed()
}

// ---------------------------------------------------- end verifier -----

/*
import (
	"fmt"
	"text/scanner"
	"unicode/utf8"

	"./seq"
)

// ----------------------------------------------------------------------------
// Grammar verification

type verifier struct {
	errors errorList
	// worklist []*Production
	worklist []*r.Sequence
	// reached  Grammar // set of productions reached from (and including) the root production
	reached map[string]*r.Sequence
	grammar map[string]*r.Sequence
	// grammar Grammar
}

func (v *verifier) error(pos int, msg string) {
	v.errors = append(v.errors, newError(pos, msg))
}

func (v *verifier) push(prod *r.Sequence) {
	if prod.Operator != r.Production {
		panic("No PRODUCTION given to push()")
	}
	name := prod.String
	if _, found := v.reached[name]; !found {
		v.worklist = append(v.worklist, prod)
		v.reached[name] = prod
	}
}

func (v *verifier) verifyChar(x *r.Sequence) rune {
	if x.Operator != r.Terminal {
		panic("No TERMINAL given to verifyChar()")
	}
	s := x.String
	if len(s) != 1 {
		v.error(x.Pos, "single char expected, found "+s)
		return 0
	}
	ch, _ := utf8.DecodeRuneInString(s)
	return ch
}

func (v *verifier) verifyExpr(expr *r.Sequence) {

	switch expr.Operator {
	case r.Or:
		for _, e := range expr.Childs {
			v.verifyExpr(&e)
		}
	case r.Group, r.Optional, r.Repeat, r.Basic: // TODO: RENAME IN SEQUENCE!!!
		for _, e := range expr.Childs {
			v.verifyExpr(&e)
		}
	case r.Ident: //  TODO: MAYBE CHANGE TO "NAME"
		// a production with this name must exist;
		// add it to the worklist if not yet processed
		if prod, found := v.grammar[expr.String]; found {
			v.push(prod)
		} else {
			v.error(expr.Pos, "missing production "+expr.String)
		}
	case r.Terminal:
		// nothing to do for now
	case r.Range:
		i := v.verifyChar(expr.Begin)
		j := v.verifyChar(expr.End)
		if i >= j {
			v.error(expr.Pos, "decreasing character range")
		}
	case r.Invalid:
		v.error(expr.Pos, expr.String)
	default:
		panic(fmt.Sprintf("internal error: unexpected type %T", expr))
	}
}

func (v *verifier) verify(grammar Grammar, start string) {
	// find root production
	root, found := grammar[start]
	if !found {
		var noPos scanner.Position
		v.error(noPos, "no start production "+start)
		return
	}

	// initialize verifier
	v.worklist = v.worklist[0:0]
	v.reached = make(Grammar)
	v.grammar = grammar

	// work through the worklist
	v.push(root)
	for {
		n := len(v.worklist) - 1
		if n < 0 {
			break
		}
		prod := v.worklist[n]
		v.worklist = v.worklist[0:n]
		v.verifyExpr(prod.Expr, isLexical(prod.Name.String))
	}

	// check if all productions were reached
	if len(v.reached) < len(v.grammar) {
		for name, prod := range v.grammar {
			if _, found := v.reached[name]; !found {
				v.error(prod.Pos(), name+" is unreachable")
			}
		}
	}
}

// Verify checks that:
//	- all productions used are defined
//	- all productions defined are used when beginning at start
//	- lexical productions refer only to other lexical productions
//
// Position information is interpreted relative to the file set fset.
//
func Verify(grammar Grammar, start string) error {
	var v verifier
	v.verify(grammar, start)
	return v.errors.Err()
}
*/
