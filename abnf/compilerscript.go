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
	curUp     map[string]r.Object // The upStream of the tag that is running right now: what 'up', push() and pop() work on.

	traceEnabled bool
	traceCount   int

	asgReference      *r.Rules // Exposed to JS as c.asg. Only compiled when the script passes it into c.compile().
	aGrammarReference *r.Rules // Exposed to JS as c.agrammar. Never used by the compiler itself.

	co *compiler
}

// ltrObject exposes the ltr map (the global left-to-right variables) to goja.
// A plain vm.Set of the Go map would read ltr.in straight out of the map, which
// forces the walk to keep it materialized on every Token; this dynamic object
// materializes the ltrText accumulator on READ instead. Everything else behaves
// exactly like goja's own map wrapper: reads go through vm.ToValue, writes store
// the exported value back into the same map.
type ltrObject struct {
	vm  *goja.Runtime
	ltr map[string]r.Object
}

func (o *ltrObject) Get(key string) goja.Value {
	v, ok := o.ltr[key]
	if !ok {
		return nil // Not there: JS sees undefined.
	}
	if text, isText := v.(*ltrText); isText {
		return o.vm.ToValue(text.String())
	}
	return o.vm.ToValue(v)
}

func (o *ltrObject) Set(key string, val goja.Value) bool {
	o.ltr[key] = val.Export()
	return true
}

func (o *ltrObject) Has(key string) bool {
	_, ok := o.ltr[key]
	return ok
}

func (o *ltrObject) Delete(key string) bool {
	delete(o.ltr, key)
	return true
}

func (o *ltrObject) Keys() []string {
	keys := make([]string, 0, len(o.ltr))
	for k := range o.ltr {
		keys = append(keys, k)
	}
	return keys
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
func (cs *compilerscript) HandleTagCode(tag *r.Rule, name scriptName, upStream map[string]r.Object, localASG *r.Rules, slot int, depth int) goja.Value { // => (changes upStream)
	if !(slot < len(*tag.CodeChilds)) { // If the tag has no slot with that number.
		return nil
	}

	// A script can start a nested compile (c.compile inside a startScript or
	// tag), whose tags point up/push/pop at their own upStream. Restore the
	// caller's afterwards like the frozen engine does (walkEngine saves curUp),
	// so the rest of the outer script does not keep operating on the LAST inner
	// tag's upStream - that leaked, and goja and -frozen printed different up.in.
	// Restoring 'up' costs nothing: it puts a goja Value that already exists
	// back into the global object.
	savedUp, savedUpVal := cs.curUp, cs.vm.Get("up")
	defer func() {
		cs.curUp = savedUp
		cs.vm.Set("up", savedUpVal)
	}()

	cs.curUp = upStream
	// 'up' is the local variable bag of THIS tag; the map 'ltr' (left to right)
	// holds the global ones. It is rebound per tag on purpose: a script that
	// captures the object (let u = up) must keep seeing the map of the tag it
	// captured it in, which is what the frozen engine does with its root var.
	// The wrapper goja builds for a Go map is cheap; the two host functions
	// above are not, which is why only they moved out of this path.
	cs.vm.Set("up", upStream)
	cs.compilerFuncMap["localAsg"] = localASG // The local part of the abstract semantic graph.
	// The node's source position is exposed as up.pos (set in compiler.go), which
	// is what abnf-of-abnf stamps onto the rules it builds; there is no c.Pos.

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
func (cs *compilerscript) RunTagCode(tag *r.Rule, name scriptName, upStream map[string]r.Object, localASG *r.Rules, slot int, depth int) (r.Object, bool) {
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

	cs.vm.Set("ltr", cs.vm.NewDynamicObject(&ltrObject{vm: cs.vm, ltr: cs.LtrStream}))

	// The local stack functions are installed ONCE and work on the upStream of
	// the tag that is running (cs.curUp, set by HandleTagCode). Re-setting them
	// per tag execution made goja build two fresh native function objects every
	// time - the single largest allocation source of the compile phase. ('up'
	// itself is still rebound per tag, see HandleTagCode.)
	cs.vm.Set("pop", func() interface{} {
		stack, ok := cs.curUp["stack"].([]interface{})
		if !ok {
			return nil
		}
		if len(stack) > 0 {
			res := stack[len(stack)-1]
			cs.curUp["stack"] = stack[:len(stack)-1]
			return res
		}
		return nil
	})

	cs.vm.Set("push", func(v interface{}) {
		stack, ok := cs.curUp["stack"].([]interface{})
		if !ok {
			stack = []interface{}{}
		}
		cs.curUp["stack"] = append(stack, v)
	})

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
