package abnf

import (
	"fmt"
	"regexp"
	"strings"

	"14.gy/mec/abnf/r"
	"github.com/dop251/goja"
)

// ----------------------------------------------------------------------------
// ASG compiler scripting subsystem

type compilerscript struct {
	vm                   *goja.Runtime
	common               *commonscript
	compilerFuncMap      map[string]r.Object
	preventDefaultOutput bool

	LtrStream map[string]r.Object // Global variables (the local variables are in upstream).
	Stack     []r.Object          // global stack (the local stack is in upstream.stack).

	traceEnabled bool
	traceCount   int

	asgReference      *r.Rules // Only for reference. Will only be used by compiler if it is passed into the compile() function.
	aGrammarReference *r.Rules // Only for reference. Will not be used by compiler.

	co *compiler
}

func (cs *compilerscript) sprintStack(space string) string {
	res := ""
	for _, elem := range cs.Stack {
		if s, ok := elem.(*string); ok {
			res += space + fmt.Sprintf("%v", *s) + "\n"
		} else {
			res += space + fmt.Sprintf("%v", elem) + "\n"
		}
	}
	return res
}

func (cs *compilerscript) traceTop(tag *r.Rule, slot int, depth int, upStream map[string]r.Object) {
	cs.traceCount++
	space := "  "

	code := (*tag.CodeChilds)[slot].String

	fmt.Print(">>>>>>>>>> Code block. Depth:", depth, "  Run # (", cs.traceCount, "), ", tag.ToString(), "\n")
	removeSpace1 := regexp.MustCompile(`[ \t]+`)
	code = removeSpace1.ReplaceAllString(code, " ")
	removeSpace2 := regexp.MustCompile(`[\n\r]\s+`)
	code = removeSpace2.ReplaceAllString(code, "\n")
	code = strings.ReplaceAll(code, "\n", "\n"+space)

	fmt.Print(space, "--\n", space, code, "\n")

	fmt.Print(space, "---\n", space, ">>>>Before call:\n")
	fmt.Print(space, ">>stack:\n", cs.sprintStack(space), space, "--\n")
	fmt.Print(space, ">>ltr: ", fmt.Sprintf("%v", cs.LtrStream), "\n", space, "--\n")
	fmt.Print(space, ">>up: ", fmt.Sprintf("%v", upStream), "\n")
	fmt.Print(space, "---\n", space, ">>>>Code output:\n")
}

func (cs *compilerscript) traceBottom(upStream map[string]r.Object) {
	space := "  "
	fmt.Print(space, "---\n", space, ">>>>After call:\n")
	fmt.Print(space, ">>stack:\n", cs.sprintStack(space), space, "--\n")
	fmt.Print(space, ">>ltr: ", fmt.Sprintf("%v", cs.LtrStream), "\n", space, "--\n")
	fmt.Print(space, ">>up: ", fmt.Sprintf("%v", upStream), "\n", space, "--\n\n\n")
}

func (cs *compilerscript) HandleTagCode(tag *r.Rule, name string, upStream map[string]r.Object, localASG *r.Rules, slot int, depth int) goja.Value { // => (changes upStream)
	if !(slot < len(*tag.CodeChilds)) { // If the tag has no slot with that number
		return nil
	}

	cs.vm.Set("up", upStream)                 // Basically the local variables. The map 'ltr' (left to right) holds the global variables.
	cs.compilerFuncMap["localAsg"] = localASG // The local part of the abstract syntax graph.
	// co.compilerFuncMap["Pos"] = tag.Pos
	// co.compilerFuncMap["ID"] = tag.Int

	cs.vm.Set("pop", func() interface{} {
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

	cs.vm.Set("push", func(v interface{}) {
		stack, ok := upStream["stack"].([]interface{})
		if !ok {
			stack = []interface{}{}
		}
		upStream["stack"] = append(stack, v)
	})

	if cs.traceEnabled {
		cs.traceTop(tag, slot, depth, upStream)
	}

	code := (*tag.CodeChilds)[slot].String

	v, err := cs.common.Run(name, code, tag.Int)
	if err != nil {
		panic(err.Error() + "\nError was in " + tag.ToString() + ", Code: '" + code + "'")
	}

	if cs.traceEnabled {
		cs.traceBottom(upStream)
	}

	return v
}

func (cs *compilerscript) initFuncMap() {
	cs.common = initFuncMapCommon(cs.vm, &cs.compilerFuncMap, cs.preventDefaultOutput)

	cs.vm.Set("popg", func() interface{} {
		if len(cs.Stack) > 0 {
			res := cs.Stack[len(cs.Stack)-1]
			cs.Stack = cs.Stack[:len(cs.Stack)-1]
			return res
		}
		return nil
	})

	cs.vm.Set("pushg", func(v interface{}) {
		cs.Stack = append(cs.Stack, v)
	})

	cs.vm.Set("ltr", cs.LtrStream)

	cs.compilerFuncMap["compile"] = func(asg *r.Rules, slot int, traceEnabled bool) map[string]r.Object {
		cs.traceEnabled = traceEnabled
		return cs.co.compile(asg, slot, 0)
	}

	cs.compilerFuncMap["asg"] = cs.asgReference           // Just for reference.
	cs.compilerFuncMap["agrammar"] = cs.aGrammarReference // Just for reference.
}

func NewCompilerScript(co *compiler, asg *r.Rules, aGrammar *r.Rules, traceEnabled, preventDefaultOutput bool) *compilerscript {
	var cs compilerscript

	cs.co = co

	cs.asgReference = asg
	cs.aGrammarReference = aGrammar

	cs.traceEnabled = traceEnabled
	cs.traceCount = 0
	cs.preventDefaultOutput = preventDefaultOutput

	cs.LtrStream = map[string]r.Object{ // Basically like global variables.
		"in":    "", // This is the parser input (the terminals).
		"stack": &cs.Stack,
	}

	cs.vm = goja.New()
	cs.initFuncMap()

	return &cs
}
