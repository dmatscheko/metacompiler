package ebnf

import (
	"fmt"

	"./r"
)

// ----------------------------------------------------------------------------
// Grammar verification

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
	startProduction := r.Rule{Operator: r.Error}
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
		ep.error(fmt.Sprintf("Invalid EBNF (undefined: %s)", ep.idents[k]), 0)
		return
	}
}

func (ep *ebnfParser) verifyGrammar() {
	ep.verifyAllUsedNamesDefined()
	ep.verifyAllNamesUsed()
}
