package abnf

import (
	"strconv"

	"14.gy/mec/abnf/r"
	"github.com/dop251/goja"
)

// ----------------------------------------------------------------------------
// Parser scripting subsystem (dynamic script rule for parser)

type parserscript struct {
	vm                   *goja.Runtime
	codeCache            map[string]*goja.Program
	compilerFuncMap      map[string]r.Object
	preventDefaultOutput bool
	stack                []r.Object // global stack.

	traceEnabled bool
	traceCount   int

	pa *parser
}

// Run executes the given string in the global context.
func (ps *parserscript) Run(name, src string) (goja.Value, error) {
	p := ps.codeCache[src]

	// Cache precompiled data
	if p == nil {
		var err error
		p, err = goja.Compile(name, src, true)
		if err != nil {
			return nil, err
		}
		ps.codeCache[src] = p
	}

	return ps.vm.RunProgram(p)
}

func (ps *parserscript) HandleScriptRule(rule *r.Rule, localProductions *r.Rules, doSkipSpaces string, depth int) *r.Rule {
	ps.compilerFuncMap["localAsg"] = localProductions // The local part of the abstract syntax graph.

	if ps.traceEnabled {
		// co.traceTop(tag, slot, depth, upStream)
	}

	code := (*rule.CodeChilds)[0].String

	v, err := ps.Run("parserCommand@"+strconv.Itoa(rule.Pos), code)
	if err != nil {
		panic(err.Error() + "\nError was in " + rule.ToString() + ", Code: '" + code + "'")
	}

	res, ok := v.Export().(*r.Rule)

	if ps.traceEnabled {
		// gp.traceBottom(upStream)
	}

	if ok {
		return res
	}
	return nil
}

func (ps *parserscript) initFuncMap() {
	initFuncMapCommon(ps.vm, &ps.compilerFuncMap, ps.preventDefaultOutput)

	ps.compilerFuncMap["getSrc"] = func() string { return ps.pa.Src }
	ps.compilerFuncMap["setSrc"] = func(src string) { ps.pa.Src = src }
	ps.compilerFuncMap["getSdx"] = func() int { return ps.pa.Sdx }
	ps.compilerFuncMap["setSdx"] = func(sdx int) { ps.pa.Sdx = sdx }

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

func NewParserScript(pa *parser) *parserscript {
	var ps parserscript
	ps.pa = pa

	ps.vm = goja.New()
	ps.codeCache = map[string]*goja.Program{}
	ps.initFuncMap()

	return &ps
}
