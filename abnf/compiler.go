package abnf

import (
	"fmt"
	"strings"

	"14.gy/mec/abnf/r"
)

// ----------------------------------------------------------------------------
// ASG compiler

type compiler struct {
	cs *compilerscript
}

//
//     OUT
//      ^
//      |
//      C---.      (C) If the current Rule has childs, the childs get sent to 'compile()'. (Also the childs of Tag Rules.)
//      |    |
//      ^    v
//      *    |     (*) All upstream (up.*) values from returning 'compile()'s are combined.
//     /|    |
//    | | _  |
//    T | |  |     (T) The text of an EBNF Terminal symbol (Token) gets returned and included into 'up.in'.
//    | X |  |     (X) The script of a single TAG Rule script gets executed. This is after their childs came back from being splitted at (C).
//    | | O  |     (O) Other Rules are ignored.
//    | | |  |
//    \ | /  |
//     \|/   |
//      *    |     (*) Childs from one Rule get splitted. The splitted path always only processe one rule (That can contain childs).
//      |    |
//      ^    |
//      IN<-'
//
// 'upStream' are the variables that go up only. They are basically local variables.
// 'ltrStream' are basically global variables. The difference betwee ltrStram and global JS variables is, that they ltrStream appends variables of sibling rules when their branches meet while propagating upwards.
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
					arr1, ok1 := upStreamMerged[k].([]interface{})
					arr2, ok2 := v.([]interface{})
					if !ok1 {
						// panic(fmt.Sprintf("Left variable 'up.%s' must only contain arrays. Contains: %#v in rule %s.", k, upStreamMerged[k], PprintRuleOnly(&rule)))
						arr1 = []interface{}{arr1}
					}
					if !ok2 {
						// panic(fmt.Sprintf("Right variable 'up.%s' must only contain arrays. Contains: %#v in rule %s.", k, v, PprintRuleOnly(&rule)))
						arr2 = []interface{}{arr2}
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
	// Inside each splitted arm do this

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
		co.cs.HandleTagCode(rule, fmt.Sprintf("TAG(at char %d)", rule.Pos), upStream, localASG, slot, depth)
		return upStream
	default:
		if len(*rule.Childs) > 0 {
			return co.compile(rule.Childs, slot, depth+1) // Evaluate the child productions of groups to collect their values.
		}
	}

	return map[string]r.Object{"in": ""}
}

func compileASGInternal(asg *r.Rules, aGrammar *r.Rules, fileName string, slot int, traceEnabled bool, preventDefaultOutput bool) interface{} {
	var co compiler

	co.cs = NewCompilerScript(&co, asg, aGrammar, fileName, traceEnabled, preventDefaultOutput)

	prolog := r.GetStartScript(aGrammar)

	var res interface{}
	if prolog != nil {
		upStream := map[string]r.Object{ // Basically the local variables.
			"in": "", // This is the parser input (the terminals).
		}
		res = co.cs.HandleTagCode(prolog, "prolog.code", upStream, asg, slot, 0).Export()
	}

	// Is called fom JS compile():
	// co.compile(asg, slot)

	return res
}

// Compiles an "abstract semantic graph". This is similar to an AST, but it also contains the semantic of the language.
// The aGrammar is only needed for its prolog code. The start rule is only needed for parsing.
func CompileASG(asg *r.Rules, aGrammar *r.Rules, fileName string, slot int, traceEnabled, preventDefaultOutput bool) (res *r.Rules, e error) {
	defer func() {
		if err := recover(); err != nil {
			res = nil
			e = fmt.Errorf("%s", err)
		}
	}()

	resObj := compileASGInternal(asg, aGrammar, fileName, slot, traceEnabled, preventDefaultOutput)

	if resObj != nil { // There should be a generated a-grammar in upstream
		resultAGrammar, ok := resObj.([]interface{})
		if !ok {
			return nil, nil
		}
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
