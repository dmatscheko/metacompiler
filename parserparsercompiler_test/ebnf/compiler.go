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

type group struct{}
type fixedGroup struct{} // keep nil elements in this group

type compiler struct {
	params map[string]object

	newGrammar Grammar
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
			// We are not allowed to delete nils, but we can split NON group objects
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
func (co *compiler) handleScript(id string, code string, childStrJoined string, childCodeResultStr string, childCodeResultObject object, variables map[string]object, codeVariables map[string]string, tree object) (string, object) { // (childCode, childObject) // TODO: the result should be an object. write a serializer for the end result. maybe it needs multiple passes. For example to be able to call functions that are defined by the EBNF. A good place for functions is e.g. the preamble.J
	fmt.Printf("### TagID: %s\n    Code: %s\n    ChildStr: %s\n    ChildObject: %s\n    Variables: %s\n    CodeVariables: %s\n    Tree: %s\n", id, code, childStrJoined, jsonizeObject(childCodeResultObject), jsonizeObject(variables), jsonizeObject(codeVariables), jsonizeObject(tree))

	objResult := childCodeResultObject

	funcMap := template.FuncMap{
		"inc": func(name string) int {
			if co.params[name] == nil {
				co.params[name] = 0
				return 0
			}

			co.params[name] = co.params[name].(int) + 1

			return co.params[name].(int)
		},

		// "mkSlice": func(args ...interface{}) []interface{} {
		// 	return args
		// },

		"notNil": func(o interface{}) bool {
			return o != nil
		},

		"childObj": func() object {
			return childCodeResultObject
		},

		"objCount": func(o interface{}) int {
			if o == nil {
				return 0
			}
			if seq, ok := o.([]interface{}); ok {
				return len(seq)
			}
			return 1
		},

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
		"upstream": func(args ...interface{}) object {
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
				objResult = nil
				return nil
			}

			// objResult = args
			// return args

			objResult = simplifyObj(args)
			return objResult
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

		// Sets a global unique name and returns its index. Example: ' {{ident .childStr}} '
		"ident": func(name string) int {
			idents := co.params["idents"].([]string)
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

			co.params["idents"] = idents
			// params["ididx"] = ididx

			return k
		},

		"addProduction": func(ident string, idx int, production object) string {
			co.newGrammar.productions = append(co.newGrammar.productions, sequence{ident, idx, production})
			return ""
		},

		// using vars:
		// <"" "{ \"{{.vars.trololo}}\", {{inc \"counter\"}}, {{.codeVars.expression}} }, ">
		// foo <"trololo"> = bar <"expression" "{{.childCode}}">

		// <"" "{{ if .childCode }}{{ set \"or\" true }}{{end}}, {{.childCode}}">
		// <"" "{{ if (eq .setVars.or true) }}{ \"OR\", {{.childCode}} }{{end}}">
		"set": func(name string, data interface{}) string {
			co.params["setVars"].(map[string]object)[name] = data
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

	tmpl, err := template.New(id).Funcs(funcMap).Parse(code)
	if err != nil {
		panic(err)
	}

	// a tag looks like <"ID" "CODE">
	co.params["id"] = id                          // The tag ID can be referenced by the code part.
	co.params["childStr"] = childStrJoined        // The collective matched strings of all child nodes.
	co.params["childCode"] = childCodeResultStr   // The collective output of all child nodes code part.
	co.params["childObj"] = childCodeResultObject // The collective generated object tree of all child nodes code part.
	co.params["vars"] = variables                 // The collective matched strings from some child node identified by a tag ID. The name of the variable is the tag ID. Example: ' foo <"bar"> '  // TODO: maybe rename to strVars
	co.params["codeVars"] = codeVariables         // The code output from some child node identified by a tag ID. The name of the variable is the tag ID. Example: ' foo <"bar" "xyz{{.childCode}}"> '
	co.params["subTree"] = tree                   // The current subtree of the parser grammar
	if co.params["setVars"] == nil {              // The global variables, that can be set with {{ set \"foo\" true }}
		co.params["setVars"] = map[string]object{}
	}
	if co.params["idents"] == nil { // The global list of unique names. Set by {{ident "someName"}}.
		co.params["idents"] = []string{}
	}
	// if params["ididx"] == nil {
	// 	params["ididx"] = []int{}
	// }
	// params["directChildCount"] = directChildCount // The amount of direct child nodes.

	var tpl bytes.Buffer
	err = tmpl.Execute(&tpl, co.params) // TODO: variables should maybe contain implicit variables (e.g.: (underscore + name) of all objects below)
	if err != nil {
		panic(err)
	}

	result := tpl.String()
	fmt.Printf("    Result:\n      CodeStrResult: %s\n      CodeObjResult: %s\n\n", result, jsonizeObject(objResult))

	// activate if necessary:
	// if seq, ok := objResult.(sequence); ok {
	// 	if len(seq) == 0 {
	// 		objResult = nil
	// 	} else if len(seq) == 1 {
	// 		objResult = seq[0]
	// 	}
	// }

	return result, objResult
}

func (co *compiler) compile(tree object, inVariables map[string]object, inCodeVariables map[string]string) (string, string, object, map[string]object, map[string]string) { // => (outStr, outCode, outObj, outVariables, outCodeVariables)
	if t, ok := tree.(sequence); ok && len(t) > 0 {
		t1 := t[0]

		if _, ok := t1.(string); !ok { // "SEQUENCE" (if there is no string at rule[0], it is a group/sequence of rules. iterate through them and apply).
			outStr := ""
			outCode := ""
			outObj := sequence{}
			outVariables := map[string]object{}
			outCodeVariables := map[string]string{}
			for _, o := range t {
				tmpStr, tmpCode, tmpObj, tmpVariables, tmpCodeVariables := co.compile(o, inVariables, inCodeVariables)
				outStr += tmpStr
				outCode += tmpCode
				if tmpObj != nil {
					outObj = appendObj(outObj, tmpObj)
				}
				for k, v := range tmpVariables {
					outVariables[k] = v
				}
				for k, v := range tmpCodeVariables {
					outCodeVariables[k] = v
				}
			}

			var outObjRes object = outObj
			if len(outObj) == 1 {
				outObjRes = outObj[0]
			} else if len(outObj) == 0 {
				outObjRes = nil
			}
			return outStr, outCode, outObjRes, outVariables, outCodeVariables
		} else if t1 == "TERMINAL" {
			return t[1].(string), "", nil, inVariables, inCodeVariables
		} else if t1 == "TAG" {
			if len(t) != 3 {
				panic(fmt.Sprintf("error at TAG: %#v", t))
			}

			tagID, tagCode := getIDAndCodeFromTag(t[1])

			outStr, outCode, outObj, tmpVariables, tmpCodeVariables := co.compile(t[2], inVariables, inCodeVariables) // evaluate the child productions of the TAG

			outVariables := map[string]object{}
			for k, v := range tmpVariables {
				outVariables[k] = v
			}
			outVariables[tagID] = outStr

			// directChildCount := 1
			// if childs, ok := t[2].(sequence); ok {
			// 	fmt.Println()
			// 	pprint("FOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOO", childs)
			// 	fmt.Println()
			// 	fmt.Println()
			// 	directChildCount = len(childs)
			// }

			// TODO: HANDLE TAG SCRIPT HERE
			tmpCode, tmpObj := co.handleScript(tagID, tagCode, outStr, outCode, outObj, outVariables, tmpCodeVariables, tree)

			// store code generated output as a variable with the name of the ID of the tag
			outCodeVariables := map[string]string{}
			for k, v := range tmpCodeVariables {
				outCodeVariables[k] = v
			}
			outCodeVariables[tagID] = tmpCode

			// outCode += tmpCode
			outCode = tmpCode
			outObj = tmpObj

			return outStr, outCode, outObj, outVariables, outCodeVariables
		}

		// fmt.Printf("## A # %#v\n", t1)

	} else {
		// fmt.Printf("## B # %#v\n", tree)
	}

	return "", "", nil, inVariables, inCodeVariables
}

func (co *compiler) CompileParseTree(parseTree object) error {
	// defer func() {
	// 	if err := recover(); err != nil {
	// 		success = false
	// 		e = fmt.Errorf(fmt.Sprintf("%s", err))
	// 	}
	// }()

	// The newly generated grammar
	co.newGrammar.productions = co.newGrammar.productions[:0]
	co.newGrammar.ididx = co.newGrammar.ididx[:0]

	co.params = map[string]object{}         // Global variables.
	var variables = map[string]object{}     // Local variables from source text (must be passed through compile).
	var codeVariables = map[string]string{} // Local variables created by semantic code (must be passed through compile).

	outStr, outCode, outObj, outVariables, outCodeVariables := co.compile(parseTree, variables, codeVariables)

	fmt.Printf("\n\nvariables:\n    %#v\n\ncode variables:\n    %#v\n\noutStr:\n    %s\n\noutCode:\n    %s\n\noutObj:\n    %s\n\nnewGrammar:\n    %#v\n\n", outVariables, outCodeVariables, outStr, outCode, jsonizeObject(simplifyObj(outObj)), jsonizeObject(co.newGrammar.productions))

	return nil
}
