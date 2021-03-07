package ebnf

import (
	"fmt"

	"./r"
	"github.com/dop251/goja"
)

type compiler struct {
	// newGrammar Grammar
	// globalVars map[string]r.Object
	vm      *goja.Runtime
	funcMap map[string]r.Object

	traceEnabled bool
}

// ----------------------------------------------------------------------------
// Dynamic parse tree compiler

// TODO: make nicely named constants for all TAG array indizes!!! (also do that for all other constant array indizes)
// TODO: set and read objVars (in addition to the vars and codeVars)
// TODO: use the verifier part of the other ebnf-parser project. Also test if every ident is available (exists)!
// TODO: remoce the integer list of idents and replace with hashmap. also panic if an entry is already in the hashmap!

/// TODO: remove this:
// type noGroup struct{}    // can be broken apart
// type oldGroup struct{}   // a group where nils can be deleted
// type fixedGroup struct{} // keep nil elements in this group

/*
	// a tag looks like <"CODE">
	vars["childStr"] = childMatchStrJoined       // (string) The collective matched strings of all child nodes.
	vars["childCode"] = childCodeResultStrJoined // (string) The collective textual output of all child tag nodes code.
	vars["childObj"] = childCodeResultObjTree    // ([]r.Sequence) The collective generated list of all child tag nodes code upstream r.Sequence trees. Each TAG node can insert its objects into one of the trees in the list via upstream.
	vars["locals"] = localVars                   // (objects map) The local (going upstream) variables, that can be set with {{ setLocal "foo" true }}, accessed with {{ .locals.foo }}, or deleted with {{ deleteLocal "foo" }}
	vars["globals"] = co.globalVars              // (objects map) The global variables, that can be set with {{ setGlobal "foo" true }}, accessed with {{ .globals.foo }}, or deleted with {{ deletGlobal "foo" }}
	vars["subTree"] = localTree                  // (object tree) The current sub tree of the parser grammar
	if co.globalVars["idents"] == nil {          // (string list) The global list of unique names. Set by {{ident "someName"}}. It is exposed to the scripting language on purpose.
*/

// RunScript executes the given string in the global context.
func (co *compiler) Run(name, src string) (goja.Value, error) {
	p, err := goja.Compile(name, src, true)

	if err != nil {
		return nil, err
	}

	return co.vm.RunProgram(p)
}

func (co *compiler) handleTagCode(code string, name string, upstream map[string]r.Object, localTree []r.Rule) { // => (codeResultObjTree) // TODO: the result should be an object. write a serializer for the end result. maybe it needs multiple passes. For example to be able to call functions that are defined by the EBNF. A good place for functions is e.g. the preamble.J
	co.vm.Set("upstream", upstream) // The object(s) that are passed from the bottom roots to the top of the tree. Initially, only TERMINALs are entered into 'upstream.text'. If 'upstream.text' contains something that can be converted into string, it is concateneted with the other TERMINAL values or filled upstream.text contents.
	co.funcMap["localAST"] = localTree

	// fmt.Printf("\n\nCODE: %s\n\n", code)

	// TODO: store precompiled data!
	_, err := co.Run(name, code)
	if err != nil {
		panic(err)
	}
}

//
//    OUT
//     ^
//     |
//     C--.      (C) If the Rule has childs, the childs get sent to 'compile()'. (Also the childs of TAG Rules.)
//     |   |
//   * ^   v     (*) All upstream values are combined.
//    /|   |
//   | | _ |
//   T | | |     (T) The text of a terminal gets sent to 'upstream.str'.
//   | X | |     (X) Here, the script of a single TAG Rule script gets executed. This is after their childs came back from being splitted at (C).
//   | | O |     (O) Other Rules are ignored.
//   | | | |
//   \ | / |
//  * \|/  |     (*) Childs from one Rule get splitted.
//     |__/
//     |
//     ^
//     IN
//
func (co *compiler) compile(productions []r.Rule, upstream map[string]r.Object) {
	if co.traceEnabled {
		fmt.Printf("\n## %#v\n--\n%#v\n", productions, upstream)
	}
	if productions == nil || len(productions) == 0 {
		return
	}

	// ----------------------------------
	// Split and collect

	upstreamMerged := map[string]r.Object{}
	if len(productions) > 1 { // "SEQUENCE" Iterate through all rules and applies.
		for _, rule := range productions { // TODO: IMPORTANT!!! Optimize this with index to the specific production/rule, like in the grammarparser.go. And also implement a feature to state the starting rule!

			// Copy, so that compile() and handleTagCode() can change them:
			upstreamEdit := map[string]r.Object{}
			for k, v := range upstream {
				upstreamEdit[k] = v
			}

			// Compile:
			co.compile([]r.Rule{rule}, upstreamEdit)

			// Merge into upstreamMerged:
			for k, v := range upstreamEdit {
				if len(k) >= 3 && k[:3] == "str" { // All upstream variables that start with 'str' are combined as string.
					// if v == nil { // Only when merging: Ignore empty/nil responses.
					// 	continue
					// }
					str1, ok1 := upstreamMerged[k].(string)
					str2, ok2 := v.(string)
					if ok1 && ok2 {
						upstreamMerged[k] = str1 + str2
					} else if ok2 {
						upstreamMerged[k] = str2
					}
					continue
				}

				if len(k) >= 3 && k[:3] == "obj" { // All upstream variables that start with 'str' are combined as string.
					if v == nil { // Only when merging: Ignore empty/nil responses.
						continue
					}
					if upstreamMerged[k] != nil {
						if mergedArr, ok := upstreamMerged[k].([]interface{}); ok { // If we can merge as array:
							if vArr, ok := v.([]interface{}); ok {
								upstreamMerged[k] = append(mergedArr, vArr...)
							} else {
								upstreamMerged[k] = append(mergedArr, v)
							}
						} else {
							upstreamMerged[k] = []interface{}{upstreamMerged[k], v}
						}
					} else {
						upstreamMerged[k] = v
					}
					continue
				}

				if upstreamMerged[k] != upstream[k] && v == upstream[k] { // If another child changed the result but the current one would not, keep the changed result of the other child (only usable when NOT merging).
					continue
				}

				// // Problem: One sets an upstream variable and it gets overwritten by another that was not updated.
				upstreamMerged[k] = v
			}
		}

		for k, v := range upstreamMerged {
			upstream[k] = v
		}
		for k := range upstream {
			if upstreamMerged[k] == nil {
				delete(upstream, k)
			}
		}

		return
	}

	// ----------------------------------
	// Inside each splitted arm do this

	// There is only one production:
	rule := productions[0]

	switch rule.Operator {
	case r.Terminal:
		// if str, ok := upstream["text"].(string); ok && len(str) > 0 {
		// 	panic("ONLY FOR DEBUG! no this should not happen")
		// }
		upstream["str"] = rule.String
		upstream["obj"] = rule.String
		return
	case r.Tag:
		tagCode := rule.TagChilds[0].String
		// First collect all the data.
		co.compile(rule.Childs, upstream) // Evaluate the child productions of the TAG to collect their values.
		// Then run the script on it.
		co.handleTagCode(tagCode, fmt.Sprintf("TAG(at char %d)", rule.Pos), upstream, productions) // TODO: maybe change "upstream" to "upstreamReplace, upstreamCombine"
		return
	default:
		if len(rule.Childs) > 0 {
			co.compile(rule.Childs, upstream) // Evaluate the child productions of groups to collect their values.
		}
	}

	return
}

func (co *compiler) initFuncMap() {
	co.vm.Set("print", fmt.Print)
	co.vm.Set("println", fmt.Println)
	co.vm.Set("printf", fmt.Printf)
	co.vm.Set("sprintf", fmt.Sprintf)

	co.vm.Set("defined", func(o r.Object) bool { return o != nil })

	co.funcMap = map[string]r.Object{ // The LLVM function will be inside such a map.
		"objectAsString": func(object r.Object, stripBraces bool) string {
			return "NOT IMPLEMENTED YET"
		},

		"sequence": func(Operator r.OperatorID, String string, Int int, Bool bool, Rune rune, Pos int, Childs []r.Rule, TagChilds []r.Rule) r.Rule {
			return r.Rule{Operator: Operator, String: String, Int: Int, Bool: Bool, Rune: Rune, Pos: Pos, Childs: Childs, TagChilds: TagChilds}
		},
		// "Upstream": func(a []r.Sequence) {
		// 	codeResultObjTree
		// },
		"GetSeqArr": func(a string, b int) []r.Rule {
			return []r.Rule{{String: a, Int: b}, {String: a, Int: b + 1}}
		},
		"Test": func() int {
			return 123
		},
		"Foo": func(a int) int {
			return a*2 + 123
		},
		"Test2": func(a []r.Rule) {
			fmt.Printf("\n\nTEST2: %#v", a)
		},
	}
	co.vm.Set("c", co.funcMap)
}

// Compiles an "abstract semantic graph". This is similar to an AST, but it also contains the semantic of the language.
func CompileASG(asg []r.Rule, extras *map[string]r.Rule, traceEnabled bool) (res map[string]r.Object, e error) {
	defer func() {
		if err := recover(); err != nil {
			res = nil
			e = fmt.Errorf(fmt.Sprintf("%s", err))
		}
	}()

	var co compiler
	co.traceEnabled = traceEnabled

	// co.globalVars = map[string]r.Object{} // Global variables.
	var upstream = map[string]r.Object{} // Local variables (must be passed through compile).

	co.vm = goja.New()
	co.initFuncMap()

	if prolog, ok := (*extras)["prolog.code"]; ok {
		co.handleTagCode(prolog.TagChilds[0].String, "prolog.code", upstream, asg)
	}

	co.compile(asg, upstream)

	if epilog, ok := (*extras)["epilog.code"]; ok {
		co.handleTagCode(epilog.TagChilds[0].String, "epilog.code", upstream, asg)
	}

	// return co.newGrammar, nil
	return upstream, nil
}
