package abnf

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"./r"
	"github.com/dop251/goja"
)

// Stripped down and slightly modified version of stconv.Unquote()
func Unescape(s string) (string, error) {
	// Is it trivial? Avoid allocation.
	if !strings.ContainsRune(s, '\\') {
		if utf8.ValidString(s) {
			return s, nil
		}
	}

	var runeTmp [utf8.UTFMax]byte
	buf := make([]byte, 0, 3*len(s)/2) // Try to avoid more allocations.
	for len(s) > 0 {
		c, multibyte, ss, err := strconv.UnquoteChar(s, 0)
		if err != nil {
			return "", err
		}
		s = ss
		if c < utf8.RuneSelf || !multibyte {
			buf = append(buf, byte(c))
		} else {
			n := utf8.EncodeRune(runeTmp[:], c)
			buf = append(buf, runeTmp[:n]...)
		}
	}
	return string(buf), nil
}

func UnescapeTilde(s string) string {
	// Is it trivial? Avoid allocation.
	if !strings.ContainsRune(s, '\\') {
		if utf8.ValidString(s) {
			return s
		}
	}

	buf := make([]byte, 0, 3*len(s)/2) // Try to avoid more allocations.
	for pos := 0; pos+1 < len(s); pos++ {
		if s[pos] == '\\' && s[pos+1] == '~' {
			buf = append(buf, s[:pos]...)
			s = s[pos+1:]
			pos = 0
		}
	}
	buf = append(buf, s...)
	return string(buf)
}

// ----------------------------------------------------------------------------
// Dynamic ASG compiler

type compiler struct {
	vm              *goja.Runtime
	compilerFuncMap map[string]r.Object

	asg      *r.Rules
	aGrammar *r.Rules

	stack     []r.Object          // global stack.
	ltrStream map[string]r.Object // global variables.

	codeCache map[string]*goja.Program

	traceEnabled         bool
	traceCount           int
	preventDefaultOutput bool
}

// JsonizeObject is ugly. Remove!
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

func (co *compiler) traceTop(tag *r.Rule, slot int, depth int, upStream map[string]r.Object) {
	co.traceCount++
	space := "  "

	code := (*tag.TagChilds)[slot].String

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

// Run executes the given string in the global context.
func (co *compiler) Run(name, src string) (goja.Value, error) {
	p := co.codeCache[src]

	// Cache precompiled data
	if p == nil {
		var err error
		p, err = goja.Compile(name, src, true)
		if err != nil {
			return nil, err
		}
		co.codeCache[src] = p
	}

	return co.vm.RunProgram(p)
}

func (co *compiler) handleTagCode(tag *r.Rule, name string, upStream map[string]r.Object, localASG *r.Rules, slot int, depth int) { // => (changes upStream)
	if !(slot < len(*tag.TagChilds)) { // If the tag has no slot with that number
		return
	}

	co.vm.Set("up", &upStream)                // Basically the local variables. The map 'ltr' (left to right) holds the global variables.
	co.compilerFuncMap["localAsg"] = localASG // The local part of the abstract syntax graph.
	co.compilerFuncMap["Pos"] = tag.Pos
	co.compilerFuncMap["ID"] = tag.ID

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
		co.traceTop(tag, slot, depth, upStream)
	}

	code := (*tag.TagChilds)[slot].String

	_, err := co.Run(name, code)
	if err != nil {
		panic(err.Error() + "\nError was in " + PprintRuleFlat(tag, false, true))
	}

	if co.traceEnabled {
		co.traceBottom(upStream)
	}
}

// TODO: correct the wording:

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
//    T | |  |     (T) The text of a EBNF Terminal symbol (Token) gets returned and included into 'up.in'.
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

		for _, rule := range *localASG { // TODO: IMPORTANT!!! Optimize this with index to the specific production/rule, like in the grammarparser.go. And also implement a feature to state the starting rule!
			// Compile:
			upStreamNew := co.compile(&r.Rules{rule}, slot, depth+1)

			for k, v := range upStreamNew {
				if k == "in" || strings.HasPrefix(k, "str") {
					str1, ok1 := upStreamMerged[k].(string)
					str2, ok2 := v.(string)
					if !ok1 {
						panic(fmt.Sprintf("Left variable 'up.%s' must only contain strings. Contains: %#v in rule %s.", k, upStreamMerged[k], PprintRuleOnly(rule)))
					}
					if !ok2 {
						panic(fmt.Sprintf("Right variable 'up.%s' must only contain strings. Contains: %#v in rule %s.", k, v, PprintRuleOnly(rule)))
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
	rule := (*localASG)[0]

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
		upStream := co.compile(rule.Childs, slot, depth+1) // Evaluate the child productions of the TAG to collect their values.
		// Then run the script on it.
		co.handleTagCode(rule, fmt.Sprintf("TAG(at char %d)", rule.Pos), upStream, localASG, slot, depth)
		return upStream
	default:
		if len(*rule.Childs) > 0 {
			return co.compile(rule.Childs, slot, depth+1) // Evaluate the child productions of groups to collect their values.
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

	co.vm.Set("sleep", func(d time.Duration) { time.Sleep(d * time.Millisecond) })

	co.vm.Set("append", func(t []interface{}, v ...interface{}) interface{} {
		tmp := append(t, v...)
		return &tmp
	})

	co.vm.Set("unescape", Unescape)
	co.vm.Set("unescapeTilde", UnescapeTilde)

	// co.vm.Set("writable", func(v interface{}) *interface{} {
	// 	return &v
	// })
	// co.vm.Set("nonwritable", func(v *interface{}) interface{} {
	// 	return *v
	// })

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

	co.compilerFuncMap = map[string]r.Object{
		"parse": func(agrammar *r.Rules, srcCode string, useBlockList bool, useFoundList bool, traceEnabled bool) *r.Rules {
			productions, _ := ParseWithAgrammar(agrammar, srcCode, useBlockList, useFoundList, traceEnabled)
			return productions
		},
		"compile": func(asg *r.Rules, slot int, traceEnabled bool) map[string]r.Object {
			co.traceEnabled = traceEnabled
			return co.compile(asg, slot, 0)
		},
		"compileWithProlog": func(asg *r.Rules, aGrammar *r.Rules, slot int, traceEnabled bool) map[string]r.Object {
			return compileASGInternal(asg, aGrammar, slot, traceEnabled, false)
		},
		"asg":          co.asg,
		"agrammar":     co.aGrammar,
		"ABNFagrammar": AbnfAgrammar,
	}
	co.vm.Set("c", co.compilerFuncMap)
	r.AbnfFuncMap["sprintProductions"] = PprintRulesFlat
	co.vm.Set("abnf", r.AbnfFuncMap)
	co.vm.Set("llvm", llvmFuncMap)
}

func compileASGInternal(asg *r.Rules, aGrammar *r.Rules, slot int, traceEnabled bool, preventDefaultOutput bool) map[string]r.Object {
	var co compiler
	co.traceEnabled = traceEnabled
	co.traceCount = 0
	co.preventDefaultOutput = preventDefaultOutput
	co.asg = asg
	co.aGrammar = aGrammar
	co.codeCache = map[string]*goja.Program{}
	co.ltrStream = map[string]r.Object{ // Basically like global variables.
		"in":    "", // This is the parser input (the terminals).
		"stack": &co.stack,
	}
	upStream := map[string]r.Object{ // Basically the local variables.
		"in": "", // This is the parser input (the terminals).
	}

	co.vm = goja.New()
	co.initFuncMap()

	prolog := r.GetProlog(aGrammar)

	if prolog != nil {
		co.handleTagCode(prolog, "prolog.code", upStream, asg, slot, 0)
	}

	// Is called fom JS compile():
	// co.compile(asg, slot)

	return co.ltrStream
}

// Compiles an "abstract semantic graph". This is similar to an AST, but it also contains the semantic of the language.
// The aGrammar is only needed for its prolog and epilog definition and its start rule definition.
func CompileASG(asg *r.Rules, aGrammar *r.Rules, slot int, traceEnabled bool, preventDefaultOutput bool) (res *r.Rules, e error) {
	defer func() {
		if err := recover(); err != nil {
			res = nil
			e = fmt.Errorf("%s", err)
		}
	}()

	resObj := compileASGInternal(asg, aGrammar, slot, traceEnabled, preventDefaultOutput)

	if resObj["agrammar"] != nil { // There should be a generated a-grammar in upstream
		resultAGrammar, ok := resObj["agrammar"].([]interface{})
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
