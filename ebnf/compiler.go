package ebnf

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"./r"
	"github.com/dop251/goja"
)

type compiler struct {
	vm              *goja.Runtime
	compilerFuncMap map[string]r.Object

	asg    r.Rules
	extras *map[string]r.Rule

	stack     []r.Object          // global stack.
	ltrStream map[string]r.Object // global variables.

	traceEnabled         bool
	traceCount           int
	preventDefaultOutput bool
}

// ----------------------------------------------------------------------------
// Dynamic ASG compiler

func (co *compiler) sprintStack(space string) string {
	res := ""
	for _, elem := range co.stack {
		if s, ok := elem.(*string); ok {
			res = res + space + jsonizeObject(*s) + "\n"
		} else {
			res = res + space + jsonizeObject(elem) + "\n"
		}
	}
	return res
}

func (co *compiler) traceTop(tag *r.Rule, depth int, upStream map[string]r.Object) {
	co.traceCount++
	space := "  "

	code := tag.TagChilds[0].String

	fmt.Print(">>>>>>>>>> Code block. Depth:", depth, "  Run # (", co.traceCount, "), ", PprintRuleOnly(tag), "\n")
	removeSpace1 := regexp.MustCompile(`[ \t]+`)
	code = removeSpace1.ReplaceAllString(code, " ")
	removeSpace2 := regexp.MustCompile(`[\n\r]\s+`)
	code = removeSpace2.ReplaceAllString(code, "\n")
	code = strings.ReplaceAll(code, "\n", "\n"+space)

	fmt.Print(space, "--\n", space, code, "\n")

	fmt.Print(space, "---\n", space, ">>>>Before call:\n")
	fmt.Print(space, ">>stack:\n", co.sprintStack(space), space, "--\n")
	fmt.Print(space, ">>ltr: ", jsonizeObject(co.ltrStream), "\n", space, "--\n")
	fmt.Print(space, ">>up: ", jsonizeObject(upStream), "\n")
	fmt.Print(space, "---\n", space, ">>>>Code output:\n")
}

func (co *compiler) traceBottom(upStream map[string]r.Object) {
	space := "  "
	fmt.Print(space, "---\n", space, ">>>>After call:\n")
	fmt.Print(space, ">>stack:\n", co.sprintStack(space), space, "--\n")
	fmt.Print(space, ">>ltr: ", jsonizeObject(co.ltrStream), "\n", space, "--\n")
	fmt.Print(space, ">>up: ", jsonizeObject(upStream), "\n", space, "--\n\n\n")
}

// RunScript executes the given string in the global context.
func (co *compiler) Run(name, src string) (goja.Value, error) {
	p, err := goja.Compile(name, src, true)

	if err != nil {
		return nil, err
	}

	return co.vm.RunProgram(p)
}

func (co *compiler) handleTagCode(tag *r.Rule, name string, upStream map[string]r.Object, localASG r.Rules, depth int) { // => (changes upStream)
	co.vm.Set("up", upStream)                 // Basically the local variables. The map 'ltr' (left to right) holds the global variables.
	co.compilerFuncMap["localAsg"] = localASG // The local part of the abstract syntax graph.

	co.vm.Set("pop", func() interface{} {
		stack, ok := upStream["stack"].([]interface{})
		if !ok {
			return nil
		}
		if len(stack) > 0 {
			res := stack[len(stack)-1]
			upStream["stack"] = stack[:len(stack)-1]
			return res
		}
		return nil
	})

	co.vm.Set("push", func(v interface{}) {
		stack, ok := upStream["stack"].([]interface{})
		if !ok {
			stack = []interface{}{}
		}
		upStream["stack"] = append(stack, v)
	})

	if co.traceEnabled {
		co.traceTop(tag, depth, upStream)
	}

	code := tag.TagChilds[0].String

	// TODO: store precompiled data!
	_, err := co.Run(name, code)
	if err != nil {
		panic(err.Error() + "\nError was in TAG " + PprintRuleOnly(tag) + "\nCode:\n" + PprintSrc(code))
	}

	if co.traceEnabled {
		co.traceBottom(upStream)
	}
}

//
//     OUT
//      ^
//      |
//      C---.      (C) If the current Rule has childs, the childs get sent to 'compile()'. (Also the childs of TAG Rules.)
//      |    |
//      ^    v
//      *    |     (*) All upstream (up.*) values from returning 'compile()'s are combined.
//     /|    |
//    | | _  |
//    T | |  |     (T) The text of a Terminal symbol gets returned and included into 'up.in'.
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
func (co *compiler) compile(localASG r.Rules, depth int) map[string]r.Object { // => (upStream)
	if localASG == nil || len(localASG) == 0 {
		return map[string]r.Object{"in": ""}
	}

	// ----------------------------------
	// Split and collect

	if len(localASG) > 1 { // "SEQUENCE" Iterate through all rules and applies.

		upStreamMerged := map[string]r.Object{"in": "", "stack": []interface{}{}}

		for _, rule := range localASG { // TODO: IMPORTANT!!! Optimize this with index to the specific production/rule, like in the grammarparser.go. And also implement a feature to state the starting rule!
			// Compile:
			upStreamNew := co.compile(r.Rules{rule}, depth+1)

			for k, v := range upStreamNew {
				if k == "in" || strings.HasPrefix(k, "str") {
					str1, ok1 := upStreamMerged[k].(string)
					str2, ok2 := v.(string)
					if !ok1 {
						panic(fmt.Sprintf("Left variable 'up.%s' must only contain strings. Contains: %#v in rule %s.", k, upStreamMerged[k], PprintRuleOnly(&rule)))
					}
					if !ok2 {
						panic(fmt.Sprintf("Right variable 'up.%s' must only contain strings. Contains: %#v in rule %s.", k, v, PprintRuleOnly(&rule)))
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
					panic("Array missing in upStreamMerged.")
				}
			}
		}

		return upStreamMerged
	}

	// ----------------------------------
	// Inside each splitted arm do this

	// There is only one production:
	rule := localASG[0]

	switch rule.Operator {
	case r.Token:
		if str, ok := co.ltrStream["in"].(string); ok {
			co.ltrStream["in"] = str + rule.String
		} else {
			panic("Variable 'ltr.in' must only contain strings.")
		}
		return map[string]r.Object{"in": rule.String, "stack": []interface{}{}}
	case r.Tag:
		// First collect all the data.
		upStream := co.compile(rule.Childs, depth+1) // Evaluate the child productions of the TAG to collect their values.
		// Then run the script on it.
		co.handleTagCode(&rule, fmt.Sprintf("TAG(at char %d)", rule.Pos), upStream, localASG, depth)
		return upStream
	default:
		if len(rule.Childs) > 0 {
			return co.compile(rule.Childs, depth+1) // Evaluate the child productions of groups to collect their values.
		}
	}

	return map[string]r.Object{"in": ""}
}

func (co *compiler) initFuncMap() {
	if co.preventDefaultOutput { // Script output disabled.
		co.vm.Set("print", func(a ...interface{}) (n int, err error) { return 0, nil })
		co.vm.Set("println", func(a ...interface{}) (n int, err error) { return 0, nil })
		co.vm.Set("printf", func(format string, a ...interface{}) (n int, err error) { return 0, nil })
	} else { // Script output enabled.
		co.vm.Set("print", fmt.Print)
		co.vm.Set("println", fmt.Println)
		co.vm.Set("printf", fmt.Printf)
	}
	co.vm.Set("sprintf", fmt.Sprintf) // Sprintf is no output.
	co.vm.Set("exit", os.Exit)

	co.vm.Set("append", func(t []interface{}, v ...interface{}) interface{} {
		return append(t, v...)
	})

	co.vm.Set("ltr", co.ltrStream)

	co.vm.Set("popg", func() interface{} {
		if len(co.stack) > 0 {
			res := co.stack[len(co.stack)-1]
			co.stack = co.stack[:len(co.stack)-1]
			return res
		}
		return nil
	})

	co.vm.Set("pushg", func(v interface{}) {
		co.stack = append(co.stack, v)
	})

	co.compilerFuncMap = map[string]r.Object{ // The LLVM function will be inside such a map.
		"compile": func(localASG r.Rules) map[string]r.Object {
			res := co.compile(localASG, 0)
			if epilog, ok := (*co.extras)["epilog.code"]; ok {
				co.handleTagCode(&epilog, "epilog.code", res, localASG, 0)
			}
			return res
		},
		"asg": co.asg,

		// "newToken": func(String string, Pos int) r.Rule {
		// 	return r.Rule{Operator: r.Token, String: String, Pos: Pos}
		// },
		// "newName": func(String string, Int int, Pos int) r.Rule {
		// 	return r.Rule{Operator: r.Ident, String: String, Int: Int, Pos: Pos}
		// },
		// "newTag": func(TagChilds r.Rules, Childs r.Rules, Pos int) r.Rule {
		// 	return r.Rule{Operator: r.Tag, TagChilds: TagChilds, Childs: Childs, Pos: Pos}
		// },
		// "newSkipSpace": func(Bool bool, Pos int) r.Rule {
		// 	return r.Rule{Operator: r.SkipSpace, Bool: Bool, Pos: Pos}
		// },

		// "newRepetition": func(Childs r.Rules, Pos int) r.Rule {
		// 	return r.Rule{Operator: r.Repeat, Childs: Childs, Pos: Pos}
		// },
		// "newOption": func(Childs r.Rules, Pos int) r.Rule {
		// 	return r.Rule{Operator: r.Optional, Childs: Childs, Pos: Pos}
		// },
		// "newGroup": func(Childs r.Rules, Pos int) r.Rule {
		// 	return r.Rule{Operator: r.Group, Childs: Childs, Pos: Pos}
		// },

		// "newSequence": func(Childs r.Rules, Pos int) r.Rule {
		// 	return r.Rule{Operator: r.Sequence, Childs: Childs, Pos: Pos}
		// },
		// "newAlternative": func(Childs r.Rules, Pos int) r.Rule {
		// 	return r.Rule{Operator: r.Or, Childs: Childs, Pos: Pos}
		// },

		// "newRule": func(Operator r.OperatorID, String string, Int int, Bool bool, Rune rune, Pos int, Childs r.Rules, TagChilds r.Rules) r.Rule {
		// 	return r.Rule{Operator: Operator, String: String, Int: Int, Bool: Bool, Rune: Rune, Pos: Pos, Childs: Childs, TagChilds: TagChilds}
		// },

		// "oid": map[string]r.OperatorID{
		// 	"Error":   r.Error,
		// 	"Success": r.Success,
		// 	// Groups types:
		// 	"Sequence": r.Sequence,
		// 	"Group":    r.Group,
		// 	// Action types:
		// 	"Token":     r.Token,
		// 	"Or":        r.Or,
		// 	"Optional":  r.Optional,
		// 	"Repeat":    r.Repeat,
		// 	"Range":     r.Range,
		// 	"SkipSpace": r.SkipSpace,
		// 	"Tag":       r.Tag,
		// 	// "Factor": r.Factor, // This one is not needed
		// 	// Link types:
		// 	"Production": r.Production,
		// 	"Ident":      r.Ident,
		// },
	}
	co.vm.Set("c", co.compilerFuncMap)
	r.EbnfFuncMap["sprintProductions"] = PprintProductionsFlat
	co.vm.Set("ebnf", r.EbnfFuncMap)
	co.vm.Set("llvm", llvmFuncMap)
}

// Compiles an "abstract semantic graph". This is similar to an AST, but it also contains the semantic of the language.
func CompileASG(asg r.Rules, extras *map[string]r.Rule, traceEnabled bool, preventDefaultOutput bool) (res map[string]r.Object, e error) {
	defer func() {
		if err := recover(); err != nil {
			res = nil
			e = fmt.Errorf("%s", err)
		}
	}()

	var co compiler
	co.traceEnabled = traceEnabled
	co.traceCount = 0
	co.preventDefaultOutput = preventDefaultOutput
	co.asg = asg
	co.extras = extras
	co.ltrStream = map[string]r.Object{ // Basically like global variables.
		"in": "", // This is the parser input (the terminals).
	}
	upStream := map[string]r.Object{ // Basically the local variables.
		"in": "", // This is the parser input (the terminals).
	}

	co.vm = goja.New()
	co.initFuncMap()

	if prolog, ok := (*extras)["prolog.code"]; ok {
		co.handleTagCode(&prolog, "prolog.code", upStream, asg, 0)
	}

	// Is called fom JS compile().
	// co.compile(asg)

	// Is called from JS compile().
	// if epilog, ok := (*extras)["epilog.code"]; ok {
	// 	co.handleTagCode(epilog.TagChilds[0].String, "epilog.code", upStream, asg, 0)
	// }

	return upStream, nil
}
