package abnf

import (
	"fmt"
	"path/filepath"
	"strings"

	"14.gy/mec/abnf/r"
)

// ----------------------------------------------------------------------------
// ASG compiler

type compiler struct {
	cs       *compilerscript
	fileName string
}

//	 OUT
//	  ^
//	  |
//	  C---.      (C) If the current Rule has childs, the childs get sent to 'compile()'. (Also the childs of Tag Rules.)
//	  |    |
//	  ^    v
//	  *    |     (*) All upstream (up.*) values from returning 'compile()'s are combined.
//	 /|    |
//	| | _  |
//	T | |  |     (T) The text of an EBNF Terminal symbol (Token) gets returned and included into 'up.in'.
//	| X |  |     (X) The script of a single Tag Rule gets executed. This is after its childs came back from being split at (C).
//	| | O  |     (O) Other Rules are ignored.
//	| | |  |
//	\ | /  |
//	 \|/   |
//	  *    |     (*) Childs from one Rule get split. Each split path only processes one rule (that can contain childs).
//	  |    |
//	  ^    |
//	  IN<-'
//
// 'upStream' holds the variables that only go up (basically the local variables, 'up' in JS).
// When the branches of sibling rules meet while propagating upwards, their upStream maps are
// merged: 'in' and 'str*' are concatenated as strings, 'stack' and 'arr*' are appended as
// arrays, and all other keys are collected into arrays of their values.
// 'ltrStream' holds the global variables ('ltr' in JS). It is one single map that all rules
// share from left to right, e.g. 'ltr.in' collects the text of all Token seen so far.
func (co *compiler) compile(localASG *r.Rules, slot int, depth int) map[string]r.Object { // => (upStream)
	if localASG == nil || len(*localASG) == 0 {
		return map[string]r.Object{"in": ""}
	}

	// ----------------------------------
	// Split and collect

	if len(*localASG) > 1 { // "SEQUENCE" Iterate through all rules and applies.

		upStreamMerged := map[string]r.Object{"in": "", "stack": []interface{}{}}

		for _, rule := range *localASG {
			// Compile:
			upStreamNew := co.compile(&r.Rules{rule}, slot, depth+1)

			for k, v := range upStreamNew {
				if k == "in" || strings.HasPrefix(k, "str") {
					str1, ok1 := upStreamMerged[k].(string)
					str2, ok2 := v.(string)
					if !ok1 {
						panic(fmt.Sprintf("Left variable 'up.%s' must only contain strings. Contains: %#v in rule %s.", k, upStreamMerged[k], rule.ToString()))
					}
					if !ok2 {
						panic(fmt.Sprintf("Right variable 'up.%s' must only contain strings. Contains: %#v in rule %s.", k, v, rule.ToString()))
					}
					upStreamMerged[k] = str1 + str2
					continue
				} else if k == "stack" || strings.HasPrefix(k, "arr") {
					// Both sides must be arrays before they can be appended.
					// A missing left side starts empty, everything else that is not an array gets wrapped into one.
					arr1, ok1 := upStreamMerged[k].([]interface{})
					arr2, ok2 := v.([]interface{})
					if !ok1 {
						if old, exists := upStreamMerged[k]; exists {
							arr1 = []interface{}{old}
						} else {
							arr1 = []interface{}{}
						}
					}
					if !ok2 {
						arr2 = []interface{}{v}
					}
					upStreamMerged[k] = append(arr1, arr2...)
					continue
				}
				// If upStreamMerged[k] already holds an array, it must stay that array and must NOT get filled with newer v.
				// So if upStreamMerged[k] has no previous entry, create an array inside and add the array v one object.
				if _, ok := upStreamMerged[k]; !ok {
					upStreamMerged[k] = []interface{}{}
				}
				if arr, ok := upStreamMerged[k].([]interface{}); ok {
					upStreamMerged[k] = append(arr, v)
				} else {
					panic("Array missing in upStreamMerged")
				}
			}
		}

		return upStreamMerged
	}

	// ----------------------------------
	// Inside each split arm do this

	// There is only one production:
	rule := (*localASG)[0]

	switch rule.Operator {
	case r.Token:
		if str, ok := co.cs.LtrStream["in"].(string); ok {
			co.cs.LtrStream["in"] = str + rule.String
		} else {
			panic("Variable 'ltr.in' must only contain strings")
		}
		return map[string]r.Object{"in": rule.String, "stack": []interface{}{}}
	case r.Tag:
		// First collect all the data.
		upStream := co.compile(rule.Childs, slot, depth+1) // Evaluate the child productions of the TAG to collect their values.
		// Then run the script on it.
		co.cs.HandleTagCode(rule, fmt.Sprintf("%s:tag:pos:%d", co.fileName, rule.Pos), upStream, localASG, slot, depth)
		return upStream
	default:
		// Not all rules have childs. E.g. a Number (from :number()) is a leaf like a Token, but without text.
		if rule.Childs != nil && len(*rule.Childs) > 0 {
			return co.compile(rule.Childs, slot, depth+1) // Evaluate the child productions of groups to collect their values.
		}
	}

	return map[string]r.Object{"in": ""}
}

func compileASGInternal(asg *r.Rules, aGrammar *r.Rules, fileName string, slot int, traceEnabled bool, preventDefaultOutput bool) interface{} {
	var co compiler

	co.cs = NewCompilerScript(&co, asg, aGrammar, traceEnabled, preventDefaultOutput)
	co.fileName = filepath.Clean(fileName)

	startScript := r.GetStartScript(aGrammar)

	var res interface{}
	if startScript != nil {
		upStream := map[string]r.Object{ // Basically the local variables.
			"in": "", // This is the parser input (the terminals).
		}
		// The actual co.compile() of the ASG is called from inside the start script (via the JS function c.compile()).
		v := co.cs.HandleTagCode(startScript, co.fileName+":startScript", upStream, asg, slot, 0)
		if v != nil { // HandleTagCode() returns nil if the start script has no code for the requested slot.
			res = v.Export()
		}
	}

	return res
}

// CompileASG compiles an "abstract semantic graph". This is similar to an AST, but it also contains the semantic of the language.
// The aGrammar is only needed for its start script (the parser needs it for everything else, the ASG already contains the rest).
func CompileASG(asg *r.Rules, aGrammar *r.Rules, fileName string, slot int, traceEnabled, preventDefaultOutput bool) (res *r.Rules, e error) {
	defer func() {
		if err := recover(); err != nil {
			res = nil
			e = fmt.Errorf("%s", err)
		}
	}()

	resObj := compileASGInternal(asg, aGrammar, fileName, slot, traceEnabled, preventDefaultOutput)

	// If the start script returned an a-grammar, convert and return it. Everything else
	// (e.g. a number or a string from a calculator grammar) results in res == nil.
	switch resultAGrammar := resObj.(type) {
	case *r.Rules: // The script used e.g. abnf.arrayToRules() or returned c.agrammar.
		res = resultAGrammar
	case []interface{}: // The script returned a plain JS array that hopefully contains rules.
		res = &r.Rules{}
		for _, rule := range resultAGrammar {
			if r, ok := rule.(*r.Rule); ok {
				*res = append(*res, r)
			} else {
				return nil, nil
			}
		}
	}
	return res, nil
}
