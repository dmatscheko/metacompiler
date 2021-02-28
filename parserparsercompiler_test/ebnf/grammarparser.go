package ebnf

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"text/template"
)

type grammarParser struct {
	src     []rune
	ch      rune
	sdx     int
	err     bool
	grammar Grammar

	// for new grammar
	newIdents  []string
	newGrammar Grammar
}

// TODO: DEDUPLICATE!!
func (gp *grammarParser) skipSpaces() {
	for {
		if gp.sdx >= len(gp.src) {
			break
		}
		gp.ch = gp.src[gp.sdx]
		if strings.IndexRune(" \t\r\n", gp.ch) == -1 {
			break
		}
		gp.sdx++
	}
}

// The self-referential EBNF is (different description form!):
//
//      EBNF = Production* .
//      Production = <ident> "=" Expression "." .
//      Expression = Sequence ("|" Sequence)* .
//      Sequence = Term+ .
//      Term = (<ident> | <string> | ("<" <ident> ">") | ("(" Expression ")")) ("*" | "+" | "?" | "!")? .
//
// The self-referential EBNF is:
//
// {
// EBNF = "{" { production } "}" .
// production  = name "=" [ expression ] "." .
// expression  = name | terminal [ "..." terminal ] | sequence | alternative | group | option | repetition .
// sequence    = expression expression { expression } .
// alternative = expression "|" expression { "|" expression } .
// group       = "(" expression ")" .
// option      = "[" expression "]" .
// repetition  = "{" expression "}" .
// }
//
// // rule == production
// // factors == non-terminal expression. a subgroup of productions/rules
// // ident == name             //  <=  identifies another block (== address of the other expression)
// // string == token == terminal == text
// // or == alternative
//
//		SOOOOOO:
//
// The rules that applies() has to deal with are BASICALLY THE SAME AS AN BNF-PARSER with annotations (NOT EBNF):
// {factors} - if rule[0] is not string,
// just apply one after the other recursively.
// {"TERMINAL", "a1"}       -- literal constants
// {"OR", <e1>, <e2>, ...}  -- (any) one of n
// {"REPEAT", <e1>}         -- as per "{}" in ebnf
// {"OPTIONAL", <e1>}       -- as per "[]" in ebnf
// {"IDENT", <name>, idx}   -- apply the sub-rule (its a link to the sub-rule) (its a production)
// {"TAG", code, <name>, idx }  ---- from dma: the semantic description in IL or something else (script language). also other things like coloring
//
// TODO: REMEMBER WHAT HAS BEEN TRIED ALREADY FOR A POSITION!
//
func (gp *grammarParser) applies(rule sequence, doSkipSpaces bool, depth int) object {
	wasSdx := gp.sdx // in case of failure
	r1 := rule[0]

	var localProductions sequence
	localProductions = localProductions[:0]

	// gp.printTrace(rule, "A", depth)

	if _, ok := r1.(string); !ok { // "SEQUENCE" (if there is no string at rule[0], it is a group/sequence of rules. iterate through them and apply)
		for i := 0; i < len(rule); i++ {
			newProduction := gp.applies(rule[i].(sequence), doSkipSpaces, depth+1)
			if newProduction == nil {
				gp.sdx = wasSdx
				return nil
			}

			if t, ok := newProduction.(sequence); ok && len(t) > 0 && t[0] == "SKIPSPACES" { // this has to be handled in a sequence
				doSkipSpaces = t[1].(bool)
				continue
			}

			localProductions = append(localProductions, newProduction)
		}
	} else if r1 == "TERMINAL" {
		if doSkipSpaces { // There can be white space in strings/text! Do not skip that.
			gp.skipSpaces()
		}
		r2 := []rune(rule[1].(string))
		for i := 0; i < len(r2); i++ {
			if gp.sdx >= len(gp.src) || gp.src[gp.sdx] != r2[i] {
				gp.sdx = wasSdx
				return nil
			}
			gp.sdx++
		}
		localProductions = append(localProductions, rule)
		// pprint("X", rule)
	} else if r1 == "OR" {
		found := false
		for i := 1; i < len(rule); i++ {
			if newProduction := gp.applies(rule[i].(sequence), doSkipSpaces, depth+1); newProduction != nil {
				// return newProduction
				localProductions = append(localProductions, newProduction)
				found = true
			}
		}
		if !found {
			gp.sdx = wasSdx
			return nil
		}
	} else if r1 == "REPEAT" {
		for {
			newProduction := gp.applies(rule[1].(sequence), doSkipSpaces, depth+1)
			if newProduction == nil {
				break
			}
			localProductions = append(localProductions, newProduction)
		}
	} else if r1 == "OPTIONAL" {
		newProduction := gp.applies(rule[1].(sequence), doSkipSpaces, depth+1)
		if newProduction != nil {
			localProductions = append(localProductions, newProduction)
		}
	} else if r1 == "IDENT" { // "IDENT" identifies another block (and its index), it is basically a link: This would e.g. be an "IDENT" to the expression-block which is at position 3: { "IDENT", "expression", 3 }
		i := rule[2].(int)
		ii := gp.grammar.ididx[i]
		newProduction := gp.applies(gp.grammar.productions[ii][2].(sequence), doSkipSpaces, depth+1)
		if newProduction == nil {
			gp.sdx = wasSdx
			return nil
		}
		localProductions = append(localProductions, newProduction)
	} else if r1 == "TAG" {
		newProduction := gp.applies(rule[2].(sequence), doSkipSpaces, depth+1)
		if newProduction != nil {
			localProductions = append(localProductions, sequence{rule[0], rule[1], newProduction})
		} else {
			return nil
		}
	} else if r1 == "SKIPSPACES" { // TODO: modify SKIPSPACES so that the chars to skip must be given to the command. e.g.: {"SKIPSPACES", "\n\t :;"}
		return rule
	} else {
		panic(fmt.Sprintf("invalid rule in applies() function: %#q", r1))
	}

	// all failed matches should have returned already
	// here must only be matches

	if len(localProductions) == 1 {
		return localProductions[0]
	}
	if localProductions == nil { // must not be nil because nil is for failed match
		localProductions = sequence{}
	}
	return localProductions
}

// ----------------------------------------------------

var params map[string]object

func handleScript(id string, script string, childStr string, childCode string, variables map[string]object, codeVariables map[string]string, tree object) string { // TODO: maybe the result should be an object if it needs multiple passes. For example to be able to call functions that are defined by the EBNF. A good place for functions is e.g. the preamble.
	fmt.Printf("### ID: %s\n    Script: %s\n    Variables: %#v\n    Tree: %#v\n    Result: ", id, script, variables, tree)

	funcMap := template.FuncMap{
		"inc": func(name string) int {
			if params[name] == nil {
				params[name] = 0
				return 0
			}

			params[name] = params[name].(int) + 1

			return params[name].(int)
		},

		"mkSlice": func(args ...interface{}) []interface{} {
			return args
		},

		// Sets a global unique name and returns its index. Example: ' {{ident .childStr}} '
		"ident": func(name string) int {
			idents := params["idents"].([]string)
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

			params["idents"] = idents
			// params["ididx"] = ididx

			return k
		},

		// using vars:
		// <"" "{ \"{{.vars.trololo}}\", {{inc \"counter\"}}, {{.codeVars.expression}} }, ">
		// foo <"trololo"> = bar <"expression" "{{.childCode}}">

		// <"" "{{ if .childCode }}{{ set \"or\" true }}{{end}}, {{.childCode}}">
		// <"" "{{ if (eq .setVars.or true) }}{ \"OR\", {{.childCode}} }{{end}}">
		"set": func(name string, data interface{}) string {
			params["setVars"].(map[string]object)[name] = data
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
	params["id"] = id                  // The tag ID can be referenced by the code part.
	params["childStr"] = childStr      // The collective matched strings of all child nodes.
	params["childCode"] = childCode    // The collective output of all child nodes code part.
	params["vars"] = variables         // The collective matched strings from some child node identified by a tag ID. The name of the variable is the tag ID. Example: ' foo <"bar"> '  // TODO: maybe rename to strVars
	params["codeVars"] = codeVariables // The code output from some child node identified by a tag ID. The name of the variable is the tag ID. Example: ' foo <"bar" "xyz{{.childCode}}"> '
	params["subTree"] = tree           // The current subtree of the parser grammar
	if params["setVars"] == nil {      // The global variables, that can be set with {{ set \"foo\" true }}
		params["setVars"] = map[string]object{}
	}
	if params["idents"] == nil { // The global list of unique names. Set by {{ident "someName"}}.
		params["idents"] = []string{}
	}
	// if params["ididx"] == nil {
	// 	params["ididx"] = []int{}
	// }
	// params["directChildCount"] = directChildCount // The amount of direct child nodes.

	var tpl bytes.Buffer
	err = tmpl.Execute(&tpl, params) // TODO: variables should maybe contain implicit variables (e.g.: (underscore + name) of all objects below)
	if err != nil {
		panic(err)
	}

	result := tpl.String()
	fmt.Println(result)
	fmt.Println()

	return result
}

func getTextFromTerminal(terminal object) string {
	t, ok := terminal.(sequence)
	if ok && len(t) == 2 && t[0] == "TERMINAL" {
		tStr, ok := t[1].(string)
		if ok {
			return tStr
		}
	}
	panic(fmt.Sprintf("error at TAG: %#v", terminal))
}

func getIDAndCodeFromTag(tagAnnotation object) (string, string) {
	tagID := ""
	tagCode := ""

	if annotationSeq, ok := tagAnnotation.(sequence); ok {
		// we have the annotation of the TAG. The annotation can be either a single TERMINAL, or a sequence of TERMINALs.
		if _, ok := annotationSeq[0].(string); ok { // single TERMINAL
			tagID = getTextFromTerminal(annotationSeq)
		} else if len(annotationSeq) == 2 { // sequence of TERMINALs (so far there is only ID and code, so 2 elements)
			tagID = getTextFromTerminal(annotationSeq[0])
			tagCode = getTextFromTerminal(annotationSeq[1])
		} else {
			panic(fmt.Sprintf("only ID and code is allowed inside TAG: %#v", tagAnnotation))
		}
	} else {
		panic(fmt.Sprintf("error at TAG: %#v", tagAnnotation))
	}

	return tagID, tagCode
}

// TODO: move into file compiler.go

func compile(tree object, inVariables map[string]object, inCodeVariables map[string]string) (string, string, map[string]object, map[string]string) { // => (outStr, outCode, outVariables)
	if t, ok := tree.(sequence); ok && len(t) > 0 {
		t1 := t[0]

		if _, ok := t1.(string); !ok { // "SEQUENCE" (if there is no string at rule[0], it is a group/sequence of rules. iterate through them and apply).
			outStr := ""
			outCode := ""
			outVariables := map[string]object{}
			outCodeVariables := map[string]string{}
			for _, o := range t {
				tmpStr, tmpCode, tmpVariables, tmpCodeVariables := compile(o, inVariables, inCodeVariables)
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

			outStr, outCode, tmpVariables, tmpCodeVariables := compile(t[2], inVariables, inCodeVariables) // evaluate the child productions of the TAG

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
			tmpCode := handleScript(tagID, tagCode, outStr, outCode, outVariables, tmpCodeVariables, tree)

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

// ----------------------------------------------------

func (gp *grammarParser) parseWithGrammarInternal(test string) bool {
	gp.newIdents = gp.newIdents[:0]
	gp.newGrammar.ididx = gp.newGrammar.ididx[:0]
	gp.newGrammar.productions = gp.newGrammar.productions[:0]
	// ep.extras = ep.extras[:0]

	var wholeTree object

	gp.src = []rune(test)
	gp.sdx = 0
	if len(gp.grammar.productions) > 0 {
		// ORIGINAL CALL:
		// res = gp.applies(gp.grammar.productions[0][2].(sequence))

		// newProduction, ok := gp.applies(gp.grammar.productions[0][2].(sequence), true, 0)

		params = map[string]object{}
		wholeTree = gp.applies(gp.grammar.productions[0][2].(sequence), true, 0)

		fmt.Println()

		pprint("productions of new grammar (whole tree)", wholeTree)

		fmt.Println()

		// var variables map[string]object
		var variables = map[string]object{}
		var codeVariables = map[string]string{}
		outStr, outCode, outVariables, outCodeVariables := compile(wholeTree, variables, codeVariables)
		fmt.Printf("\n\nvariables:\n    %#v\n\ncode variables:\n    %#v\n\noutStr:\n    %s\n\noutCode:\n    %s\n\n", outVariables, outCodeVariables, outStr, outCode)
		// pprint("variables", variables)

		// res = ok
	}
	gp.skipSpaces()
	if gp.sdx < len(gp.src) {
		wholeTree = nil
	}

	return wholeTree != nil
}

func (gp *grammarParser) printTrace(rule sequence, action string, depth int) {
	traceEnabled := false // TODO: CHANGE THIS WHEN DEBUGGING
	// traceEnabled := true // TODO: CHANGE THIS WHEN DEBUGGING

	d := ">"
	for i := 0; i < depth; i++ {
		d += ">"
	}

	if traceEnabled {
		c := '-'
		if gp.sdx < len(gp.src) {
			c = gp.src[gp.sdx]
		}
		pprint(fmt.Sprintf("%3d%s rule for pos # %d (%c) action: %s", depth, d, gp.sdx, c, action), rule)
	}
}

func ParseWithGrammar(grammar Grammar, srcCode string) (success bool, e error) {
	var gp grammarParser
	gp.err = false

	// defer func() {
	// 	if err := recover(); err != nil {
	// 		success = false
	// 		e = fmt.Errorf(fmt.Sprintf("%s", err))
	// 	}
	// }()

	gp.grammar = grammar
	return gp.parseWithGrammarInternal(srcCode), nil
}
