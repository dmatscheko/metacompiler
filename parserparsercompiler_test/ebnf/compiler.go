package ebnf

import (
	"bytes"
	"fmt"
	"reflect"
	"text/template"
)

// ----------------------------------------------------------------------------
// Dynamic parse tree compiler

// TODO: make nicely named constants for all TAG array indizes!!! (also do that for all other constant array indizes)
// TODO: set and read objVars (in addition to the vars and codeVars)
// TODO: use the verifier part of the other ebnf-parser project. Also test if every ident is available (exists)!
// TODO: remoce the integer list of idents and replace with hashmap. also panic if an entry is already in the hashmap!

// TODO: implement this!!!
// object type:
type group2 int

const ( // iota is reset to 0
	no    group2 = iota // can be broken apart
	yes                 // a group, but nils can be deleted
	fixed               // a group where position is imporant. keep nil elements in this group
)

type sequence2 struct {
	group group2
	data  []object
}

/// instead of this:
type noGroup struct{}    // can be broken apart
type group struct{}      // a group where nils can be deleted
type fixedGroup struct{} // keep nil elements in this group

type compiler struct {
	globalVars map[string]object
	newGrammar Grammar

	traceEnabled bool
}

func appendObj(target []interface{}, elems ...interface{}) []interface{} {
	if elems == nil || len(elems) == 0 {
		return target
	}
	if elems[0] == nil {
		return target
	}

	switch elems[0].(type) {
	case group:
		if len(elems) == 1 {
			return target
		} else if len(elems) == 2 {
			return appendObj(target, elems[1])
		}
		return append(target, elems)
	}

	if len(elems) == 1 {
		if seq, ok := elems[0].(sequence); ok {
			return appendObj(target, seq...)
		}
	}

	return append(target, elems...)
}

func simplifyObj(obj interface{}, mustBeGroup ...bool /* = false */) object {
	enforceGroup := len(mustBeGroup) > 0 && mustBeGroup[0] == true

	// If the object is NOT an array, return the one element.
	o, ok := obj.([]interface{})
	if !ok {
		return obj
	}

	// If the array is empty, return nil.
	if len(o) == 0 {
		return nil
	}

	// If the object is a group, do not remove elements, even if they are nil. Still check their child arrays.
	switch o[0].(type) {
	case group:
		if len(o) == 1 { // If the object IS a group and one element, return nil, because it would be an empty group.
			return nil
		} else if len(o) == 2 { // If the object IS a group and two elements, return only the second element, because it is a single element.
			return simplifyObj(o[1], true)
		}

		res := sequence{group{}}

		for i := 1; i < len(o); i++ { // Do not drop nil elements in a group.
			// We are not allowed to delete nils, but we can do this inside the NON group childs
			res = appendObj(res, simplifyObj(o[i]))
		}
		return res
	}

	// If the object is NOT a group and one element, return the one element without the array.
	if len(o) == 1 {
		return simplifyObj(o[0], enforceGroup)
	}

	var res sequence

	if enforceGroup {
		// There was no group, so enforce one.
		res = sequence{group{}}
	} else {
		res = sequence{}
	}

	// There might have been a group, but we are in a sub array, so we are still allowed to drop nil elements.
	for i := 0; i < len(o); i++ {
		tmp := simplifyObj(o[i])
		if tmp != nil {
			res = appendObj(res, tmp)
		}
	}

	// Last cleaning for length.
	if len(res) == 0 {
		return nil
	} else if len(res) == 1 {
		if enforceGroup {
			return res
		}
		return res[0]
	} else if len(res) == 2 && enforceGroup {
		return res[1]
	}
	return res
}

// maybe store only beginning and end instead of matched string?
// childCodeResultStrJoined : maybe makle this also a tree of strings and flatten later!
func (co *compiler) handleTagCode(code string, childMatchStrJoined string, childCodeResultStrJoined string, childCodeResultObjTree object, localVars map[string]object, localTree object) (string, object) { // => (codeResultStrJoined, codeResultObjTree) // TODO: the result should be an object. write a serializer for the end result. maybe it needs multiple passes. For example to be able to call functions that are defined by the EBNF. A good place for functions is e.g. the preamble.J
	codeResultObjTree := childCodeResultObjTree
	vars := map[string]object{}

	funcMap := template.FuncMap{
		"inc": func(name string) int {
			if co.globalVars[name] == nil {
				co.globalVars[name] = 0
				return 0
			}

			co.globalVars[name] = co.globalVars[name].(int) + 1

			return co.globalVars[name].(int)
		},

		// "mkSlice": func(args ...interface{}) []interface{} {
		// 	return args
		// },

		"notNil": func(o interface{}) bool {
			return o != nil
		},

		"notEmpty": func(s interface{}) bool {
			return s != nil && s != ""
		},

		"childObj": func() object {
			return childCodeResultObjTree
		},

		// TODO: maybe it can overwrite len!!!!
		"lenx": func(o interface{}) int {
			if o == nil {
				return 0
			}
			if seq, ok := o.([]interface{}); ok {
				if len(seq) > 0 {
					if _, ok := seq[0].(group); ok {
						return len(seq) - 1
					}
				}
				return len(seq)
			}
			return 1
		},

		// "objCount": func(o interface{}) int {
		// 	if o == nil {
		// 		return 0
		// 	}
		// 	if seq, ok := o.([]interface{}); ok {
		// 		return len(seq)
		// 	}
		// 	return 1
		// },

		// "strip": func(o interface{}) interface{} {
		// 	if seq, ok := o.(sequence); ok {
		// 		if len(seq) == 1 {
		// 			return seq[0]
		// 		}
		// 	}
		// 	return o
		// },

		"indexNil": func(o interface{}, args ...int) interface{} {
			tmpObj := o
			for i := 0; i < len(args); i++ {
				if tmpObj == nil {
					return ""
				} else if seq, ok := tmpObj.([]interface{}); ok && len(seq) > args[i] {
					tmpObj = seq[args[i]]
				} else {
					return ""
				}
			}
			return tmpObj
		},

		// "typeOf": func(o interface{}, typeName string) bool {
		// 	if o == nil {
		// 		return 0
		// 	}
		// 	if seq, ok := o.([]interface{}); ok {
		// 		return len(seq)
		// 	}
		// 	return 1
		// },

		// TODO!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!! after this is implemented, add "functions" or "variables" like directChildCount, leftCount, depth, ... also add functions to access objects (nodes)
		// {{ upstream a b c d }}
		"upstream": func(args ...interface{}) string {
			// Only if no upstream is called, everything is put upstream unmodified.

			// seq := sequence{}
			// seq = append(seq, args...)

			// if a sequence is fully nil, returl nil
			allNil := true
			for _, o := range args {
				if o != nil {
					allNil = false
					break
				}
			}
			if allNil {
				codeResultObjTree = nil
				return ""
			}

			// objResult = args
			// return args

			codeResultObjTree = simplifyObj(args)
			return ""
		},

		"group": func(args ...interface{}) object {
			// Only if no upstream is called, everything is put upstream unmodified.

			// if a sequence is fully nil, returl nil
			allNil := true
			for _, o := range args {
				if o != nil {
					allNil = false
					break
				}
			}
			if allNil {
				return nil
			}

			return appendObj(sequence{group{}}, args...)
		},

		// TODO: allow multiple arrays of unique names. Something like ' {{ident "groupfoo" .childStr}} '
		// Sets a global unique name and returns its index. Example: ' {{ident .childStr}} '
		"ident": func(name string) int {
			idents := co.globalVars["idents"].([]string)
			// ididx := params["ididx"].([]int)

			k := -1
			for i, id := range idents {
				if id == name {
					k = i
					break
				}
			}
			if k == -1 {
				idents = append(idents, name)
				k = len(idents) - 1
				// ididx = append(ididx, -1)
			}

			co.globalVars["idents"] = idents
			// params["ididx"] = ididx

			return k
		},

		"addProduction": func(ident string, idx int, production object) string {
			co.newGrammar.productions = append(co.newGrammar.productions, sequence{ident, idx, production})
			return ""
		},

		// <"" "{{ if (eq .globalVars.or true) }}{ \"OR\", {{.childCode}} }{{end}}">
		// <"" "{{ if .childCode }}{{ setGlobal \"or\" true }}{{end}}, {{.childCode}}">
		// this variables will only be visible upstream
		"setLocal": func(name string, data interface{}) string {
			localVars[name] = data
			return ""
		},

		// using vars:
		// <"" "{ \"{{.vars.trololo}}\", {{inc \"counter\"}}, {{.codeVars.expression}} }, ">
		// foo <"trololo"> = bar <"expression" "{{.childCode}}">

		// <"" "{{ if (eq .globalVars.or true) }}{ \"OR\", {{.childCode}} }{{end}}">
		// <"" "{{ if .childCode }}{{ setGlobal \"or\" true }}{{end}}, {{.childCode}}">
		"setGlobal": func(name string, data interface{}) string {
			co.globalVars[name] = data
			return ""
		},

		// {{deleteLocalVar "foo"}}
		"deleteLocalVar": func(s string) string {
			delete(localVars, s)
			return ""
		},

		// {{deleteLocalVar "foo"}}
		"deleteGlobalVar": func(s string) string {
			delete(co.globalVars, s)
			return ""
		},

		"exists": func(name string, data interface{}) bool {
			v := reflect.ValueOf(data)
			if v.Kind() == reflect.Ptr {
				v = v.Elem()
			}
			if v.Kind() != reflect.Struct {
				return false
			}
			return v.FieldByName(name).IsValid()
		},
	}

	// funcMap := template.FuncMap{
	//
	// func MakeFuncMap(u *user) map[string]interface{} {
	// 	return map[string]interface{} {
	// 		"User": func() *user {return u}, //Can be accessed by "User." within your template
	// 	}
	// }

	tmpl, err := template.New("").Funcs(funcMap).Parse(code)
	if err != nil {
		panic(err)
	}

	// a tag looks like <"CODE">
	vars["childStr"] = childMatchStrJoined       // (string) The collective matched strings of all child nodes.
	vars["childCode"] = childCodeResultStrJoined // (string) The collective textual output of all child tag nodes code.
	vars["childObj"] = childCodeResultObjTree    // (object tree) The collective generated object tree of all child tag nodes code.
	vars["locals"] = localVars                   // (objects map) The local (going upstream) variables, that can be set with {{ setLocal "foo" true }}, accessed with {{ .locals.foo }}, or deleted with {{ deleteLocal "foo" }}
	vars["globals"] = co.globalVars              // (objects map) The global variables, that can be set with {{ setGlobal "foo" true }}, accessed with {{ .globals.foo }}, or deleted with {{ deletGlobal "foo" }}
	vars["subTree"] = localTree                  // (object tree) The current sub tree of the parser grammar
	if co.globalVars["idents"] == nil {          // (string list) The global list of unique names. Set by {{ident "someName"}}. It is exposed to the scripting language on purpose.
		co.globalVars["idents"] = []string{}
	}
	// if params["ididx"] == nil {
	// 	params["ididx"] = []int{}
	// }
	// params["directChildCount"] = directChildCount // The amount of direct child nodes.

	if co.traceEnabled {
		fmt.Printf("\n### handleTagCode input:\n    TagCode: %s\n    ChildMatchedStr: %s\n    ChildCodeResultStrJoined: %s\n    ChildCodeResultObjTree: %s\n    GlobalVars: %s\n    LocalVars: %s\n    LocalGrammarTree: %s\n", code, childMatchStrJoined, childCodeResultStrJoined, jsonizeObject(childCodeResultObjTree), jsonizeObject(co.globalVars), jsonizeObject(localVars), jsonizeObject(localTree))
	}

	var tpl bytes.Buffer
	err = tmpl.Execute(&tpl, vars) // TODO: variables should maybe contain implicit variables (e.g.: (underscore + name) of all objects below)
	if err != nil {
		panic(err)
	}
	codeResultStrJoined := tpl.String()

	if co.traceEnabled {
		fmt.Printf("  ## Result:\n    CodeResultStrJoined: %s\n    CodeResultObjTree: %s\n    GlobalVars: %s\n    LocalVars: %s\n", codeResultStrJoined, jsonizeObject(codeResultObjTree), jsonizeObject(co.globalVars), jsonizeObject(localVars))
	}

	// activate if necessary:
	// if seq, ok := objResult.(sequence); ok {
	// 	if len(seq) == 0 {
	// 		objResult = nil
	// 	} else if len(seq) == 1 {
	// 		objResult = seq[0]
	// 	}
	// }

	return codeResultStrJoined, codeResultObjTree // localVars (the modification is already seen by the caller, because its a pointer)
}

func (co *compiler) compile(parseTree object, inLocalVars map[string]object) (string, string, object, map[string]object) { // => (childMatchStrJoined, codeResultStrJoined, codeResultObjTree, outLocalVars)
	if t, ok := parseTree.(sequence); ok && len(t) > 0 {
		t1 := t[0]

		if _, ok := t1.(string); !ok { // "SEQUENCE" (if there is no string at rule[0], it is a group/sequence of rules. iterate through them and apply).
			matchStr := ""
			codeResultStr := ""
			codeResultObjTree := sequence{}
			localVars := map[string]object{}
			for _, subTree := range t {
				childMatchStr, childCodeResultStr, childCodeResultObjTree, childLocalVars := co.compile(subTree, inLocalVars)
				matchStr += childMatchStr
				codeResultStr += childCodeResultStr
				if childCodeResultObjTree != nil {
					codeResultObjTree = appendObj(codeResultObjTree, childCodeResultObjTree)
				}
				for k, v := range childLocalVars {
					localVars[k] = v
				}
			}

			var outCodeResultObjTree object = codeResultObjTree
			if len(codeResultObjTree) == 1 {
				outCodeResultObjTree = codeResultObjTree[0]
			} else if len(codeResultObjTree) == 0 {
				outCodeResultObjTree = nil
			}
			return matchStr, codeResultStr, outCodeResultObjTree, localVars
		} else if t1 == "TERMINAL" {
			return t[1].(string), "", nil, inLocalVars
		} else if t1 == "TAG" {
			tagCode := getIDAndCodeFromTag(t)

			childMatchStrJoined, childCodeResultStrJoined, childCodeResultObjTree, childLocalVars := co.compile(t[2], inLocalVars) // evaluate the child productions of the TAG

			// directChildCount := 1
			// if childs, ok := t[2].(sequence); ok {
			// 	fmt.Println()
			// 	pprint("FOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOO", childs)
			// 	fmt.Println()
			// 	fmt.Println()
			// 	directChildCount = len(childs)
			// }

			// Copy, so that handleTagCode() can change them:
			outLocalVars := map[string]object{}
			for k, v := range childLocalVars {
				outLocalVars[k] = v
			}

			// outLocalVars will be changed by co.handleCode()!
			codeResultStrJoined, codeResultObjTree := co.handleTagCode(tagCode, childMatchStrJoined, childCodeResultStrJoined, childCodeResultObjTree, outLocalVars, parseTree)

			// store code generated output as a variable with the name of the ID of the tag
			// outCodeVariables := map[string]string{}
			// for k, v := range tmpCodeVars {
			// 	outCodeVariables[k] = v
			// }
			// outCodeVariables[tagID] = tmpCode

			return childMatchStrJoined, codeResultStrJoined, codeResultObjTree, outLocalVars
		}

		// fmt.Printf("## A # %#v\n", t1)

	} else {
		// fmt.Printf("## B # %#v\n", tree)
	}

	return "", "", nil, inLocalVars
}

func CompileParseTree(parseTree object, traceEnabled bool) (err error) {
	// defer func() {
	// 	if err := recover(); err != nil {
	// 		err = fmt.Errorf(fmt.Sprintf("%s", err))
	// 	}
	// }()

	var co compiler
	co.traceEnabled = traceEnabled

	// The newly generated grammar
	co.newGrammar.productions = co.newGrammar.productions[:0]
	co.newGrammar.ididx = co.newGrammar.ididx[:0]

	co.globalVars = map[string]object{} // Global variables.
	var localVars = map[string]object{} // Local variables (must be passed through compile).

	// childMatchStrJoined = String tree
	// codeResultStrJoined = Joined strings of the code output tree
	// codeResultObjTree = Code object output tree (created by upstream)
	// outLocalVars = local variables that were passed up the tree
	childMatchStrJoined, codeResultStrJoined, codeResultObjTree, outLocalVars := co.compile(parseTree, localVars)

	fmt.Printf("\n\n==================\nResult:\n==================\n\nLocalVars:\n    %#v\n\nChildMatchStrJoined:\n    %s\n\nCodeResultStrJoined:\n    %s\n\nCodeResultObjTree:\n    %s\n\nNewGrammar:\n    %s\n\n", outLocalVars, childMatchStrJoined, codeResultStrJoined, jsonizeObject(simplifyObj(codeResultObjTree)), jsonizeObject(co.newGrammar.productions))

	return nil
}
