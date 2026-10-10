package abnf

import (
	"testing"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
)

// A function definition is named by the line of the statement that creates its
// closure, not by the first statement of its body - when the marker can be that
// statement's (it does not follow the body, and the closure is not a member
// stored into a class descriptor).
//
//	line 1: def f():
//	line 2:     return 1
//	line 3: class C:
//	line 4:     def m(): return 2
//	line 5: g = lambda: 3
func TestCallgraphNamesADefinitionByItsOwnLine(t *testing.T) {
	src := "def f():\n    return 1\nclass C:\n    def m(): return 2\ng = lambda: 3\n"
	SetTraceSource("cg.py", src)
	line := func(n int) int64 { // the offset where line n starts
		return int64(traceLineStarts[n-1])
	}

	m := ir.NewModule()
	i64 := types.I64
	ext := func(name string, n int) *ir.Func {
		params := make([]*ir.Param, n)
		for i := range params {
			params[i] = ir.NewParam("", i64)
		}
		return m.NewFunc(name, i64, params...)
	}
	jsStrMem := m.NewFunc("js_str_mem", i64, ir.NewParam("", types.I8Ptr), ir.NewParam("", i64))
	srcpos := ext("js_srcpos", 1)
	closure := ext("js_closure", 2)
	decl := ext("js_tdecl", 3)
	set := ext("js_set", 3)
	objNew := ext("js_obj_new", 0)
	k := func(v int64) constant.Constant { return constant.NewInt(i64, v) }

	body := func(name string, markerLine int) {
		f := m.NewFunc(name, i64, ir.NewParam("env", i64), ir.NewParam("args", i64))
		b := f.NewBlock("")
		b.NewCall(srcpos, k(line(markerLine)))
		b.NewRet(k(0))
	}
	body("jsf_901", 2) // f: its body is on line 2
	body("jsf_902", 4) // m
	body("jsf_903", 5) // g: the marker of a one-line body is the definition's line

	main := m.NewFunc("jsmain", i64, ir.NewParam("env", i64), ir.NewParam("args", i64))
	b := main.NewBlock("")
	b.NewCall(srcpos, k(line(1)))
	b.NewCall(decl, k(0), whyStr(m, b, jsStrMem, 0, "f"), b.NewCall(closure, k(901), k(0)))
	b.NewCall(srcpos, k(line(3)))
	obj := b.NewCall(objNew)
	b.NewCall(set, obj, whyStr(m, b, jsStrMem, 1, "m"), b.NewCall(closure, k(902), k(0)))
	b.NewCall(srcpos, k(line(5)))
	b.NewCall(decl, k(0), whyStr(m, b, jsStrMem, 2, "g"), b.NewCall(closure, k(903), k(0)))
	b.NewRet(k(0))

	defs, _ := extractCallGraph(m)
	got := map[string]int{}
	for _, d := range defs {
		got[d.Name] = d.Line
	}
	for name, want := range map[string]int{
		"f": 1, // the def statement, not its body's line 2
		"m": 4, // a member of the class statement: its body, never the class's line 3
		"g": 5,
	} {
		if got[name] != want {
			t.Errorf("%s: line %d, want %d (all: %v)", name, got[name], want, got)
		}
	}
}
