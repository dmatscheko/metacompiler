package abnf

import (
	"fmt"
	"os"
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
	compilerFuncMap      map[string]r.Object // The JS object 'c' (compiler API).
	preventDefaultOutput bool

	LtrStream map[string]r.Object // The global variables ('ltr' in JS; the local variables are in upStream).
	Stack     []r.Object          // The global stack (via pushg()/popg(); the local stack is in upStream["stack"]).

	traceEnabled bool
	traceCount   int

	asgReference      *r.Rules // Exposed to JS as c.asg. Only compiled when the script passes it into c.compile().
	aGrammarReference *r.Rules // Exposed to JS as c.agrammar. Never used by the compiler itself.

	co *compiler
}

// sprintTraceStack formats the global stack for the tag trace, one element per line.
func sprintTraceStack(stack []r.Object, space string) string {
	res := ""
	for _, elem := range stack {
		if s, ok := elem.(*string); ok {
			res += space + fmt.Sprintf("%v", *s) + "\n"
		} else {
			res += space + fmt.Sprintf("%v", elem) + "\n"
		}
	}
	return res
}

// traceTagTop/traceTagBottom print the tag trace (the -vvN / c.compile(..., true)
// debug aid) around one tag execution. Both engines share them, and they write
// to STDERR: printing to stdout corrupted the -q byte identity and would leak
// into the next -pipe segment's input.
func traceTagTop(traceCount int, tag *r.Rule, slot int, depth int, stack []r.Object, ltr map[string]r.Object, upStream map[string]r.Object) {
	space := "  "

	code := (*tag.CodeChilds)[slot].String

	fmt.Fprint(os.Stderr, ">>>>>>>>>> Code block. Depth:", depth, "  Run # (", traceCount, "), ", tag.ToString(), "\n")
	removeSpace1 := regexp.MustCompile(`[ \t]+`)
	code = removeSpace1.ReplaceAllString(code, " ")
	removeSpace2 := regexp.MustCompile(`[\n\r]\s+`)
	code = removeSpace2.ReplaceAllString(code, "\n")
	code = strings.ReplaceAll(code, "\n", "\n"+space)

	fmt.Fprint(os.Stderr, space, "--\n", space, code, "\n")

	fmt.Fprint(os.Stderr, space, "---\n", space, ">>>>Before call:\n")
	fmt.Fprint(os.Stderr, space, ">>stack:\n", sprintTraceStack(stack, space), space, "--\n")
	fmt.Fprint(os.Stderr, space, ">>ltr: ", fmt.Sprintf("%v", ltr), "\n", space, "--\n")
	fmt.Fprint(os.Stderr, space, ">>up: ", fmt.Sprintf("%v", upStream), "\n")
	fmt.Fprint(os.Stderr, space, "---\n", space, ">>>>Code output:\n")
}

func traceTagBottom(stack []r.Object, ltr map[string]r.Object, upStream map[string]r.Object) {
	space := "  "
	fmt.Fprint(os.Stderr, space, "---\n", space, ">>>>After call:\n")
	fmt.Fprint(os.Stderr, space, ">>stack:\n", sprintTraceStack(stack, space), space, "--\n")
	fmt.Fprint(os.Stderr, space, ">>ltr: ", fmt.Sprintf("%v", ltr), "\n", space, "--\n")
	fmt.Fprint(os.Stderr, space, ">>up: ", fmt.Sprintf("%v", upStream), "\n", space, "--\n\n\n")
}

// HandleTagCode executes the JS code of the given slot of a Tag (the ASG carries multiple
// code slots when a Tag was written with multiple comma separated code strings). It returns
// the JS result value of the code, or nil if the tag has no code for that slot. The code can
// change upStream (visible to it as 'up').
func (cs *compilerscript) HandleTagCode(tag *r.Rule, name string, upStream map[string]r.Object, localASG *r.Rules, slot int, depth int) goja.Value { // => (changes upStream)
	if !(slot < len(*tag.CodeChilds)) { // If the tag has no slot with that number.
		return nil
	}

	// A script can start a nested compile (c.compile inside a startScript or
	// tag), whose tags rebind up/push/pop below. Restore the caller's bindings
	// afterwards like the frozen engine does (walkEngine saves curUp), so the
	// rest of the outer script does not keep operating on the LAST inner tag's
	// upStream - that leaked, and goja and -frozen printed different up.in.
	savedUp := cs.vm.Get("up")
	savedPop := cs.vm.Get("pop")
	savedPush := cs.vm.Get("push")
	defer func() {
		cs.vm.Set("up", savedUp)
		cs.vm.Set("pop", savedPop)
		cs.vm.Set("push", savedPush)
	}()

	cs.vm.Set("up", upStream)                 // Basically the local variables. The map 'ltr' (left to right) holds the global variables.
	cs.compilerFuncMap["localAsg"] = localASG // The local part of the abstract semantic graph.
	// The node's source position is exposed as up.pos (set in compiler.go), which
	// is what abnf-of-abnf stamps onto the rules it builds; there is no c.Pos.

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
		cs.traceCount++
		traceTagTop(cs.traceCount, tag, slot, depth, cs.Stack, cs.LtrStream, upStream)
	}

	code := (*tag.CodeChilds)[slot].String

	v, err := cs.common.Run(name, code, tag.Int)
	if err != nil {
		panic(wrapScriptError(err, tag.ToString(), code))
	}

	if cs.traceEnabled {
		traceTagBottom(cs.Stack, cs.LtrStream, upStream)
	}

	return v
}

// RunTagCode adapts HandleTagCode to the scriptEngine interface: it exports the
// goja result value to a plain Go value.
func (cs *compilerscript) RunTagCode(tag *r.Rule, name string, upStream map[string]r.Object, localASG *r.Rules, slot int, depth int) (r.Object, bool) {
	v := cs.HandleTagCode(tag, name, upStream, localASG, slot, depth)
	if v == nil {
		return nil, false
	}
	return v.Export(), true
}

// Ltr returns the global (left to right) variables.
func (cs *compilerscript) Ltr() map[string]r.Object {
	return cs.LtrStream
}

// initFuncMap installs the compiler specific JS API on top of the common one:
// the local and global stack functions, 'ltr', and the c.compile()/c.asg/c.agrammar entries.
func (cs *compilerscript) initFuncMap() {
	cs.common = NewCommonScript(cs.vm, &cs.compilerFuncMap, cs.preventDefaultOutput)

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
		// The JS parameter can only turn tracing on, not off: c.compile(c.asg) leaves out the
		// parameter, which arrives here as false and must not override the -vb2/-vvb2 command
		// line flags.
		cs.traceEnabled = cs.traceEnabled || traceEnabled
		return cs.co.compile(asg, slot, 0)
	}

	cs.compilerFuncMap["asg"] = cs.asgReference           // Just for reference (usually passed to c.compile()).
	cs.compilerFuncMap["agrammar"] = cs.aGrammarReference // Just for reference.
}

// NewCompilerScript creates the JS VM for one compile run.
func NewCompilerScript(co *compiler, asg *r.Rules, aGrammar *r.Rules, traceEnabled, preventDefaultOutput bool) *compilerscript {
	var cs compilerscript

	cs.co = co

	cs.asgReference = asg
	cs.aGrammarReference = aGrammar

	cs.traceEnabled = traceEnabled
	cs.traceCount = 0
	cs.preventDefaultOutput = preventDefaultOutput

	cs.LtrStream = map[string]r.Object{ // Basically like global variables.
		"in":    "",        // Collects the text of all Token that the compiler has seen so far (left to right).
		"stack": &cs.Stack, // The global stack is also reachable as ltr.stack.
	}

	cs.vm = goja.New()
	cs.initFuncMap()

	return &cs
}
