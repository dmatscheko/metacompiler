package ebnf

import (
	"bytes"
	"fmt"
	"reflect"
	"text/template"
)

// ----------------------------------------------------------------------------
// Dynamic parse tree compiler

type compiler struct {
	params map[string]object
}

// maybe store only beginning and end instead of matched string?
func (co *compiler) handleScript(id string, script string, childStr string, childCode string, variables map[string]object, codeVariables map[string]string, tree object) string { // TODO: the result should be an object. write a serializer for the end result. maybe it needs multiple passes. For example to be able to call functions that are defined by the EBNF. A good place for functions is e.g. the preamble.
	fmt.Printf("### ID: %s\n    Script: %s\n    Variables: %#v\n    Tree: %#v\n    Result: ", id, script, variables, tree)

	funcMap := template.FuncMap{
		"inc": func(name string) int {
			if co.params[name] == nil {
				co.params[name] = 0
				return 0
			}

			co.params[name] = co.params[name].(int) + 1

			return co.params[name].(int)
		},

		"mkSlice": func(args ...interface{}) []interface{} {
			return args
		},

		// TODO!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!! after this is implemented, add "functions" or "variables" like directChildCount, leftCount, depth, ... also add functions to access objects (nodes)
		// {{ upstream a b c d }}
		"upstream": func(args ...interface{}) string {
			// This should be returned by function handleScript() instead of the string.
			// Only if no upstream is called, everything is put upstream unmodified.
			return ""
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

	tmpl, err := template.New(id).Funcs(funcMap).Parse(script)
	if err != nil {
		panic(err)
	}

	// a tag looks like <"ID" "CODE">
	co.params["id"] = id                  // The tag ID can be referenced by the code part.
	co.params["childStr"] = childStr      // The collective matched strings of all child nodes.
	co.params["childCode"] = childCode    // The collective output of all child nodes code part.
	co.params["vars"] = variables         // The collective matched strings from some child node identified by a tag ID. The name of the variable is the tag ID. Example: ' foo <"bar"> '  // TODO: maybe rename to strVars
	co.params["codeVars"] = codeVariables // The code output from some child node identified by a tag ID. The name of the variable is the tag ID. Example: ' foo <"bar" "xyz{{.childCode}}"> '
	co.params["subTree"] = tree           // The current subtree of the parser grammar
	if co.params["setVars"] == nil {      // The global variables, that can be set with {{ set \"foo\" true }}
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
	fmt.Println(result)
	fmt.Println()

	return result
}

func (co *compiler) compile(tree object, inVariables map[string]object, inCodeVariables map[string]string) (string, string, map[string]object, map[string]string) { // => (outStr, outCode, outVariables)
	if t, ok := tree.(sequence); ok && len(t) > 0 {
		t1 := t[0]

		if _, ok := t1.(string); !ok { // "SEQUENCE" (if there is no string at rule[0], it is a group/sequence of rules. iterate through them and apply).
			outStr := ""
			outCode := ""
			outVariables := map[string]object{}
			outCodeVariables := map[string]string{}
			for _, o := range t {
				tmpStr, tmpCode, tmpVariables, tmpCodeVariables := co.compile(o, inVariables, inCodeVariables)
				outStr += tmpStr
				outCode += tmpCode
				for k, v := range tmpVariables {
					outVariables[k] = v
				}
				for k, v := range tmpCodeVariables {
					outCodeVariables[k] = v
				}
			}
			return outStr, outCode, outVariables, outCodeVariables
		} else if t1 == "TERMINAL" {
			return t[1].(string), "", inVariables, inCodeVariables
		} else if t1 == "TAG" {
			if len(t) != 3 { // TODO: maybe also 2, if tag is allowed to hold no child productions (e.g. useful at preamble).
				panic(fmt.Sprintf("error at TAG: %#v", t))
			}

			tagID, tagCode := getIDAndCodeFromTag(t[1])

			outStr, outCode, tmpVariables, tmpCodeVariables := co.compile(t[2], inVariables, inCodeVariables) // evaluate the child productions of the TAG

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
			tmpCode := co.handleScript(tagID, tagCode, outStr, outCode, outVariables, tmpCodeVariables, tree)

			outCodeVariables := map[string]string{}
			for k, v := range tmpCodeVariables {
				outCodeVariables[k] = v
			}
			outCodeVariables[tagID] = tmpCode

			// outCode += tmpCode
			outCode = tmpCode

			return outStr, outCode, outVariables, outCodeVariables
		}

		// fmt.Printf("## A # %#v\n", t1)

	} else {
		// fmt.Printf("## B # %#v\n", tree)
	}

	return "", "", inVariables, inCodeVariables
}

func (co *compiler) CompileParseTree(parseTree object) error {
	// defer func() {
	// 	if err := recover(); err != nil {
	// 		success = false
	// 		e = fmt.Errorf(fmt.Sprintf("%s", err))
	// 	}
	// }()

	co.params = map[string]object{}         // Global variables.
	var variables = map[string]object{}     // Local variables from source text (must be passed through compile).
	var codeVariables = map[string]string{} // Local variables created by semantic code (must be passed through compile).

	outStr, outCode, outVariables, outCodeVariables := co.compile(parseTree, variables, codeVariables)

	fmt.Printf("\n\nvariables:\n    %#v\n\ncode variables:\n    %#v\n\noutStr:\n    %s\n\noutCode:\n    %s\n\n", outVariables, outCodeVariables, outStr, outCode)

	return nil
}
