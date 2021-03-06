package ebnf

import (
	"fmt"

	"./seq"
	"github.com/dop251/goja"
)

type compiler struct {
	// newGrammar Grammar
	// globalVars map[string]seq.Object
	vm      *goja.Runtime
	funcMap map[string]seq.Object

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

// RunScript executes the given string in the global context.
func (co *compiler) Run(name, src string) (goja.Value, error) {
	p, err := goja.Compile(name, src, true)

	if err != nil {
		return nil, err
	}

	return co.vm.RunProgram(p)
}

func (co *compiler) handleTagCode(code string, name string, upstream map[string]seq.Object, localTree []seq.Sequence) { // => (codeResultObjTree) // TODO: the result should be an object. write a serializer for the end result. maybe it needs multiple passes. For example to be able to call functions that are defined by the EBNF. A good place for functions is e.g. the preamble.J
	// co.vm.Set("globals", co.globalVars) // is set once by initFuncMap()
	co.vm.Set("upstream", upstream) // The object(s) that are passed from the bottom roots to the top of the tree. Initially, only TERMINALs are entered into 'upstream.text'. If 'upstream.text' contains something that can be converted into string, it is concateneted with the other TERMINAL values or filled upstream.text contents.
	co.funcMap["localAST"] = localTree

	// fmt.Printf("\n\nCODE: %s\n\n", code)

	// TODO: store precompiled data!
	_, err := co.Run(name, code)
	if err != nil {
		panic(err)
	}
}

func (co *compiler) compile(productions []seq.Sequence, upstream map[string]seq.Object) {
	if productions == nil || len(productions) == 0 {
		// fmt.Printf("## A # %#v\n", t1)
		return
	}

	// ----------------------------------

	upstreamNew := map[string]seq.Object{}
	if len(productions) > 1 { // "SEQUENCE" Iterate through all rules and apply.
		for _, rule := range productions { // TODO: IMPORTANT!!! Optimize this with index to the specific production/rule, like in the grammarparser.go. And also implement a feature to state the starting rule!

			// Copy, so that compile and handleTagCode can change them:
			upstreamCopy := map[string]seq.Object{}
			for k, v := range upstream {
				upstreamCopy[k] = v
			}
			co.compile([]seq.Sequence{rule}, upstreamCopy)
			for k, v := range upstreamCopy {
				if k == "text" {
					str1, ok1 := upstreamNew[k].(string)
					str2, ok2 := v.(string)
					if ok1 && ok2 {
						upstreamNew[k] = str1 + str2
					} else if ok2 {
						upstreamNew[k] = str2
					}
				} else {
					if !(v == nil && upstreamNew[k] != nil) {
						upstreamNew[k] = v
					}
				}
			}
		}

		for k, v := range upstreamNew {
			upstream[k] = v
		}
		for k := range upstream {
			if upstreamNew[k] == nil {
				delete(upstream, k)
			}
		}

		return
	}

	// --------------------------------------

	// There is only one production:
	rule := productions[0]

	switch rule.Operator {
	case seq.Terminal:
		// if str, ok := inLocalVars["text"].(string); ok && len(str) > 0 {
		// 	panic("ONLY FOR DEBUG! no this should not happen")
		// }
		upstream["text"] = rule.String
		return
	case seq.Tag:
		tagCode := rule.TagChilds[0].String
		// First collect all the data.
		// inLocalVars will be changed by co.compile()
		co.compile(rule.Childs, upstream) // Evaluate the child productions of the TAG.
		// Then run the script on it.
		// inLocalVars will be changed by co.handleTagCode()
		co.handleTagCode(tagCode, fmt.Sprintf("TAG(at char %d)", rule.Pos), upstream, productions)
		return
	default:
		if len(rule.Childs) > 0 {
			co.compile(rule.Childs, upstream) // Evaluate the child productions of groups.
		}
	}

	// fmt.Printf("## B # %#v\n", tree)
	return
}

func (co *compiler) initFuncMap() {
	// co.vm.Set("globals", co.globalVars)

	co.vm.Set("print", fmt.Print)
	co.vm.Set("println", fmt.Println)
	co.vm.Set("printf", fmt.Printf)
	co.vm.Set("sprintf", fmt.Sprintf)

	co.vm.Set("defined", func(o seq.Object) bool { return o != nil })

	co.funcMap = map[string]seq.Object{ // The LLVM function will be inside such a map.
		"objectAsString": func(object seq.Object, stripBraces bool) string {
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
	co.vm.Set("c", co.funcMap)
}

// Compiles an "abstract semantic graph". This is similar to an AST, but it also contains the semantic of the language.
func CompileASG(ast []seq.Sequence, extras *map[string]seq.Sequence, traceEnabled bool) (g Grammar, e error) {
	defer func() {
		if err := recover(); err != nil {
			g = Grammar{}
			e = fmt.Errorf(fmt.Sprintf("%s", err))
		}
	}()

	var co compiler
	co.traceEnabled = traceEnabled

	// co.globalVars = map[string]seq.Object{} // Global variables.
	var upstream = map[string]seq.Object{} // Local variables (must be passed through compile).

	co.vm = goja.New()
	co.initFuncMap()

	if prolog, ok := (*extras)["prolog.code"]; ok {
		_, err := co.Run("prolog.code", prolog.TagChilds[0].String)
		if err != nil {
			panic(err)
		}
	}

	co.compile(ast, upstream)

	if epilog, ok := (*extras)["epilog.code"]; ok {
		_, err := co.Run("epilog.code", epilog.TagChilds[0].String)
		if err != nil {
			panic(err)
		}
	}

	// fmt.Printf("\n\n==================\nResult:\n==================\n\nUpstream Vars:\n    %#v\n\n", upstream)

	// return co.newGrammar, nil
	return Grammar{}, nil // TODO: change
}
