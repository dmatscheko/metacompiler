package ebnf

import (
	"fmt"

	"./r"
)

// ----------------------------------------------------------------------------
// a-Grammar verification

var reachedNames map[string]bool

func (ep *ebnfParser) collectReachableNames(productions *r.Rules, prod *r.Rule) {
	for _, child := range prod.Childs {
		if child.Operator == r.Ident {
			i := child.Int
			// ii := ep.ididx[i]
			nextProd := &(*productions)[i] // [ii]
			if nextProd.Operator == r.Production {
				if reachedNames[nextProd.String] {
					continue
				}
				reachedNames[nextProd.String] = true
			} else {
				panic(fmt.Sprintf("Not a production: '%s' at position %d.", nextProd.String, nextProd.Pos))
			}
			ep.collectReachableNames(productions, nextProd)
		} else if len(child.Childs) > 0 {
			ep.collectReachableNames(productions, &child)
		}
	}
}

// Checks if all defined productiona are used.
func (ep *ebnfParser) verifyAllNamesUsed() {
	reachedNames = make(map[string]bool)

	// Get name of start production.
	startRule := GetStartRule(&ep.aGrammar)
	if startRule == nil {
		panic("No start production defined.")
	}

	productions := GetProductions(&ep.aGrammar)

	// Get start production.
	startProduction := &(*productions)[startRule.Int]

	if startProduction.Operator != r.Production && startProduction.String != startRule.String {
		panic(fmt.Sprintf("Defined start production (%s) not found.", startRule.String))
	}
	reachedNames[startProduction.String] = true
	ep.collectReachableNames(productions, startProduction)

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
