package abnf

import (
	"strconv"

	"14.gy/mec/abnf/r"
	"github.com/dop251/goja"
)

// ----------------------------------------------------------------------------
// Parser scripting subsystem (dynamic script rule for parser)

// scriptRuleRunner executes the code of an inline :script() command. There are
// two implementations: parserscript runs it with goja, frozenParserScript
// compiles it with the frozen MetaJS bootstrap and executes the resulting IR.
type scriptRuleRunner interface {
	HandleScriptRule(rule *r.Rule, localProductions *r.Rules, depth int) *r.Rule
}

type parserscript struct {
	vm                   *goja.Runtime
	common               *commonscript
	compilerFuncMap      map[string]r.Object // The JS object 'c' (here extended with the parser state accessors).
	preventDefaultOutput bool
	stack                []r.Object // Global stack for the parser scripts (via push()/pop()). Survives between :script() calls.

	// traceEnabled bool
	// traceCount   int

	pa *parser
}

// HandleScriptRule executes the JS code of an inline :script() command. The code can inspect
// and change the parser state (c.getSdx() etc.). If it returns a rule (built via the abnf.*
// functions), the parser applies that rule at the current position; the result of every other
// return value is nil, which means there is nothing to apply.
func (ps *parserscript) HandleScriptRule(rule *r.Rule, localProductions *r.Rules, depth int) *r.Rule {
	ps.compilerFuncMap["localAsg"] = localProductions // The local part of the abstract semantic graph.

	// if ps.traceEnabled {
	// co.traceTop(tag, slot, depth, upStream)
	// }

	code := (*rule.CodeChilds)[0].String // TODO: Handle slot!

	// The :script() code lives in the GRAMMAR: run it under the grammar's
	// module name (include/load/store resolve relative to it, like tag
	// scripts), falling back to the parsed file for grammars without an
	// :origin() stamp.
	module := r.GetOrigin(ps.pa.agrammar)
	if module == "" {
		module = ps.pa.fileName
	}
	v, err := ps.common.Run(module+":parserCommand:"+strconv.Itoa(rule.Pos), code, rule.Int)
	if err != nil {
		panic(err.Error() + "\nError was in " + rule.ToString() + ", Code: '" + code + "'")
	}

	res, ok := v.Export().(*r.Rule)

	// if ps.traceEnabled {
	// gp.traceBottom(upStream)
	// }

	if ok {
		return res
	}
	return nil
}

// initFuncMap installs the parser specific JS API on top of the common one:
// accessors for the parse state and a stack that survives between the :script() calls.
func (ps *parserscript) initFuncMap() {
	ps.common = NewCommonScript(ps.vm, &ps.compilerFuncMap, ps.preventDefaultOutput)

	ps.compilerFuncMap["getSrc"] = func() string { return ps.pa.Src }
	ps.compilerFuncMap["setSrc"] = func(src string) { ps.pa.Src = src }
	ps.compilerFuncMap["getSdx"] = func() int { return ps.pa.Sdx }
	ps.compilerFuncMap["setSdx"] = func(sdx int) { ps.pa.Sdx = sdx }
	// peek returns the byte at the current parse position plus offset, or -1 outside of the
	// target text. Unlike getSrc() it does not copy the target text into the JS runtime, so
	// scripts can use it to look ahead cheaply (e.g. to check a keyword boundary).
	ps.compilerFuncMap["peek"] = func(offset int) int {
		pos := ps.pa.Sdx + offset
		if pos < 0 || pos >= len(ps.pa.Src) {
			return -1
		}
		return int(ps.pa.Src[pos])
	}

	ps.vm.Set("pop", func() interface{} {
		if len(ps.stack) > 0 {
			res := ps.stack[len(ps.stack)-1]
			ps.stack = ps.stack[:len(ps.stack)-1]
			return res
		}
		return nil
	})

	ps.vm.Set("push", func(v interface{}) {
		ps.stack = append(ps.stack, v)
	})
}

// NewParserScript creates the JS VM for the dynamic :script() rules of one parse run.
func NewParserScript(pa *parser, preventDefaultOutput bool) *parserscript {
	var ps parserscript
	ps.pa = pa
	ps.preventDefaultOutput = preventDefaultOutput

	ps.vm = goja.New()
	ps.initFuncMap()

	return &ps
}
