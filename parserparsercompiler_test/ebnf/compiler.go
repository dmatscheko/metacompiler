package ebnf

import (
	"fmt"

	"./seq"
	"github.com/dop251/goja"
)

type compiler struct {
	globalVars map[string]seq.Object
	newGrammar Grammar
	vm         *goja.Runtime
	funcMap    map[string]seq.Object

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
	vars["childObj"] = childCodeResultObjTree    // ([]seq.Sequence) The collective generated list of all child tag nodes code upstream seq.Sequence trees. Each TAG node can insert its objects into one of the trees in the list via upstream.
	vars["locals"] = localVars                   // (objects map) The local (going upstream) variables, that can be set with {{ setLocal "foo" true }}, accessed with {{ .locals.foo }}, or deleted with {{ deleteLocal "foo" }}
	vars["globals"] = co.globalVars              // (objects map) The global variables, that can be set with {{ setGlobal "foo" true }}, accessed with {{ .globals.foo }}, or deleted with {{ deletGlobal "foo" }}
	vars["subTree"] = localTree                  // (object tree) The current sub tree of the parser grammar
	if co.globalVars["idents"] == nil {          // (string list) The global list of unique names. Set by {{ident "someName"}}. It is exposed to the scripting language on purpose.
*/

func (co *compiler) handleTagCode(code string, childMatchStr string, childCodeResultObjTree interface{}, childLocalVars map[string]seq.Object, localTree seq.Object) interface{} { // => (codeResultObjTree) // TODO: the result should be an object. write a serializer for the end result. maybe it needs multiple passes. For example to be able to call functions that are defined by the EBNF. A good place for functions is e.g. the preamble.J
	// co.funcMap["globals"] = co.globalVars // is set once by initFuncMap()
	co.funcMap["locals"] = childLocalVars
	co.funcMap["upstream"] = childCodeResultObjTree // The object that is passed from the bottom roots to the top of the tree.
	co.funcMap["childStr"] = childMatchStr
	co.funcMap["localAST"] = localTree

	co.vm.Set("c", co.funcMap)

	// fmt.Printf("\n\nCODE: %s\n\n", code)

	// TODO: precompile!
	_, err := co.vm.RunString(code)
	if err != nil {
		panic(err)
	}

	return co.funcMap["upstream"] // localVars (the modification is already seen by the caller, because its a pointer)
}

func (co *compiler) compile(productions []seq.Sequence, inLocalVars map[string]seq.Object) (string, interface{}, map[string]seq.Object) { // => (childMatchStr, codeResultObjTree, outLocalVars)
	if productions == nil || len(productions) == 0 {
		// fmt.Printf("## A # %#v\n", t1)
		return "", nil, inLocalVars
	}

	// ----------------------------------

	if len(productions) > 1 { // "SEQUENCE" Iterate through all rules and apply.
		matchStr := ""
		codeResultObjTree := []interface{}{}
		localVars := map[string]seq.Object{}
		for _, rule := range productions { // TODO: IMPORTANT!!! Optimize this with index to the specific production/rule, like in the grammarparser.go. And also implement a feature to state the starting rule!
			childMatchStr, childCodeResultObjTree, childLocalVars := co.compile([]seq.Sequence{rule}, inLocalVars)
			matchStr += childMatchStr
			if childCodeResultObjTree != nil {
				if arr, ok := childCodeResultObjTree.([]interface{}); ok {
					codeResultObjTree = append(codeResultObjTree, arr...)
				} else if childCodeResultObjTree != nil {
					codeResultObjTree = append(codeResultObjTree, childCodeResultObjTree)
				}
			}
			for k, v := range childLocalVars {
				localVars[k] = v
			}
		}

		outCodeResultObjTree := codeResultObjTree
		if len(outCodeResultObjTree) == 0 {
			outCodeResultObjTree = nil
		}
		return matchStr, outCodeResultObjTree, localVars
	}

	// --------------------------------------

	// There is only one production:
	rule := productions[0]

	switch rule.Operator {
	case seq.Terminal:
		return rule.String, nil, inLocalVars
	case seq.Tag:
		tagCode := rule.TagChilds[0].String

		childMatchStrJoined, childCodeResultObjTree, childLocalVars := co.compile(rule.Childs, inLocalVars) // evaluate the child productions of the TAG

		// directChildCount, ....

		// Copy, so that handleTagCode() can change them:
		outLocalVars := map[string]seq.Object{}
		for k, v := range childLocalVars {
			outLocalVars[k] = v
		}

		// outLocalVars will be changed by co.handleCode()!
		codeResultObjTree := co.handleTagCode(tagCode, childMatchStrJoined, childCodeResultObjTree, outLocalVars, productions)

		return childMatchStrJoined, codeResultObjTree, outLocalVars
	}

	// fmt.Printf("## B # %#v\n", tree)
	return "", nil, inLocalVars
}

func (co *compiler) initFuncMap() {
	co.funcMap = map[string]seq.Object{
		"globals": co.globalVars,

		"upstreamAsString": func() string {
			return "NOT IMPLEMENTED YET"
		},

		"sequence": func(Operator seq.OperatorID, String string, Int int, Bool bool, Rune rune, Pos int, Childs []seq.Sequence, TagChilds []seq.Sequence) seq.Sequence {
			return seq.Sequence{Operator: Operator, String: String, Int: Int, Bool: Bool, Rune: Rune, Pos: Pos, Childs: Childs, TagChilds: TagChilds}
		},
		// "Upstream": func(a []seq.Sequence) {
		// 	codeResultObjTree
		// },
		"GetSeqArr": func(a string, b int) []seq.Sequence {
			return []seq.Sequence{{String: a, Int: b}, {String: a, Int: b + 1}}
		},
		"Test": func() int {
			return 123
		},
		"Foo": func(a int) int {
			return a*2 + 123
		},
		"Test2": func(a []seq.Sequence) {
			fmt.Printf("\n\nTEST2: %#v", a)
		},
	}
}

// Maybe call it "abstract semantic graph" instead of AST
func CompileAST(ast []seq.Sequence, traceEnabled bool) (g Grammar, err error) {
	// defer func() {
	// 	if err := recover(); err != nil {
	// 		g = Grammar{}
	// 		err = fmt.Errorf(fmt.Sprintf("%s", err))
	// 	}
	// }()

	var co compiler
	co.traceEnabled = traceEnabled

	co.globalVars = map[string]seq.Object{} // Global variables.
	var localVars = map[string]seq.Object{} // Local variables (must be passed through compile).

	co.initFuncMap()
	co.vm = goja.New()

	// childMatchStrJoined = String tree
	// codeResultStrJoined = Joined strings of the code output tree
	// codeResultObjTree = Code object output tree (created by upstream)
	// outLocalVars = local variables that were passed up the tree
	childMatchStrJoined, codeResultObjTree, outLocalVars := co.compile(ast, localVars)

	fmt.Printf("\n\n==================\nResult:\n==================\n\nLocalVars:\n    %#v\n\nChildMatchStrJoined:\n    %s\n\nCodeResultObjTree:\n    %s\n\nNewGrammar:\n    %s\n\n", outLocalVars, childMatchStrJoined, jsonizeObject(codeResultObjTree), jsonizeObject(co.newGrammar.Productions))

	return co.newGrammar, nil
}
