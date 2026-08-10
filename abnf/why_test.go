package abnf

import (
	"strings"
	"testing"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
)

// The -why report is read off a module that was already emitted, so the test
// builds a module by hand in the shape compile-core.js produces: an i64 handle
// world, string literals via js_str_mem(GEP into a char-array global), statement
// boundaries via js_srcpos(byteOffset), and grammar helpers as module functions
// that carry no marker of their own.
//
// Everything the report claims is pinned here: the marker moves the cursor and
// is not itself reported, js_str_mem is folded into the display of its consumer,
// a helper called from a statement is expanded transitively, the filter selects
// statements, and a function with no marker cannot inherit the previous
// function's line.

// whyStr appends `js_str_mem(getelementptr @g, len)` to b and answers the call.
func whyStr(m *ir.Module, b *ir.Block, jsStrMem *ir.Func, n int, s string) *ir.InstCall {
	g := m.NewGlobalDef("s"+string(rune('a'+n)), constant.NewCharArrayFromString(s))
	zero := constant.NewInt(types.I64, 0)
	gep := constant.NewGetElementPtr(g.Typ.ElemType, g, zero, zero)
	return b.NewCall(jsStrMem, gep, constant.NewInt(types.I64, int64(len(s))))
}

// whyFixture builds the module and registers a two-line source for it.
//
//	line 1: v = o.x
//	line 2: p(v)
//
// with `sw_kget` as a grammar helper that itself calls `sw_safeget`, so the
// transitive expansion has something to follow.
func whyFixture(t *testing.T) *ir.Module {
	t.Helper()
	src := "v = o.x\np(v)\n"
	SetTraceSource("probe.x", src)

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
	jsGet := ext("js_get", 2)
	jsSafe := ext("js_safeget", 2)
	jsCall := ext("js_call", 2)
	jsNumI := ext("js_num_i", 1)

	// The helper: defined, and with no js_srcpos anywhere.
	helper := m.NewFunc("sw_kget", i64, ir.NewParam("", i64))
	hb := helper.NewBlock("")
	hb.NewCall(jsSafe, helper.Params[0], whyStr(m, hb, jsStrMem, 0, "__class"))
	hb.NewRet(constant.NewInt(i64, 0))

	// The program function, with a marker per statement.
	f := m.NewFunc("jsmain", i64)
	b := f.NewBlock("")
	b.NewCall(srcpos, constant.NewInt(i64, 0)) // offset 0 -> line 1
	b.NewCall(helper, constant.NewInt(i64, 7))
	b.NewCall(jsGet, constant.NewInt(i64, 7), whyStr(m, b, jsStrMem, 1, "x"))
	b.NewCall(srcpos, constant.NewInt(i64, 8)) // offset 8 -> line 2
	b.NewCall(jsNumI, constant.NewInt(i64, 1))
	b.NewCall(jsNumI, constant.NewInt(i64, 1))
	b.NewCall(jsNumI, constant.NewInt(i64, 1))
	b.NewCall(jsCall, constant.NewInt(i64, 1), constant.NewInt(i64, 2))
	b.NewRet(constant.NewInt(i64, 0))
	return m
}

func whyReport(t *testing.T, m *ir.Module, filter string) string {
	t.Helper()
	old := WhyFilter
	WhyFilter = filter
	defer func() { WhyFilter = old }()
	var out strings.Builder
	writeWhy(m, &out)
	return out.String()
}

func TestWhyAttributesCallsToTheStatementMarker(t *testing.T) {
	prev := WhyEnabled
	WhyEnabled = true
	defer func() { WhyEnabled = prev }()
	m := whyFixture(t)
	got := whyReport(t, m, "")

	for _, want := range []string{
		"probe.x:1",
		"v = o.x", // The statement text is quoted, so a misattribution is visible.
		// A js_str_mem argument is decoded and folded in; a constant operand shows
		// its value, and everything else is elided to `_`.
		`js_get(7,"x")`,
		"probe.x:2",
		"p(v)",
		"js_num_i x3",             // A run of identical calls collapses.
		"sw_kget",                 // The helper reached from line 1 is expanded...
		`js_safeget(_,"__class")`, // ... including its own body.
	} {
		if !strings.Contains(got, want) {
			t.Errorf("report is missing %q:\n%s", want, got)
		}
	}
	// The marker is a cursor, not a reported call, and neither is the string
	// literal it folded away.
	if strings.Contains(got, "js_srcpos") && !strings.Contains(got, "no js_srcpos markers") {
		t.Errorf("js_srcpos should never be reported as an emitted call:\n%s", got)
	}
	if strings.Contains(got, "js_str_mem") {
		t.Errorf("js_str_mem should be folded into its consumer, not listed:\n%s", got)
	}
	// The helper has no marker, so it must not inherit line 2 - the only thing
	// under line 2 is what jsmain emitted after the second marker.
	line2 := got[strings.Index(got, "probe.x:2"):]
	if strings.Contains(line2[:strings.Index(line2, "sw_kget")], "js_safeget") {
		t.Errorf("a helper's body leaked into the previous statement:\n%s", got)
	}
}

func TestWhyFilterSelectsStatements(t *testing.T) {
	prev := WhyEnabled
	WhyEnabled = true
	defer func() { WhyEnabled = prev }()
	m := whyFixture(t)
	got := whyReport(t, m, "js_num_i")
	if !strings.Contains(got, "matched 1 of 2 statement(s)") {
		t.Errorf("filter should keep exactly the js_num_i statement:\n%s", got)
	}
	if strings.Contains(got, "v = o.x") {
		t.Errorf("filtered-out statement still printed:\n%s", got)
	}
}

// A grammar that never applies stmtPos (ruby, php) emits no markers at all. The
// report must say so rather than print an empty page, and must fall back to the
// helper listing - which is then every function in the module.
func TestWhyWithoutMarkersFallsBackToHelpers(t *testing.T) {
	prev := WhyEnabled
	WhyEnabled = true
	defer func() { WhyEnabled = prev }()
	m := ir.NewModule()
	i64 := types.I64
	radd := m.NewFunc("js_radd", i64, ir.NewParam("", i64), ir.NewParam("", i64))
	f := m.NewFunc("jsmain", i64)
	b := f.NewBlock("")
	b.NewCall(radd, constant.NewInt(i64, 1), constant.NewInt(i64, 2))
	b.NewRet(constant.NewInt(i64, 0))

	got := whyReport(t, m, "")
	if !strings.Contains(got, "no js_srcpos markers") {
		t.Errorf("a module with no markers must say so:\n%s", got)
	}
	if !strings.Contains(got, "jsmain") || !strings.Contains(got, "js_radd") {
		t.Errorf("the helper fallback should still name the call:\n%s", got)
	}
}

// -why is the fourth consumer of the statement markers, so it has to turn them
// on the same way -trace, -cfgraph and -callgraph do. If this drifts, the report
// silently reports nothing but the prelude.
func TestWhyRequestsStatementMarkers(t *testing.T) {
	prevWhy, prevTrace, prevCFG, prevCG := WhyEnabled, TraceOutPath, CFGOutPath, CallgraphOutPath
	defer func() {
		WhyEnabled, TraceOutPath, CFGOutPath, CallgraphOutPath = prevWhy, prevTrace, prevCFG, prevCG
	}()
	WhyEnabled, TraceOutPath, CFGOutPath, CallgraphOutPath = false, "", "", ""
	if TraceMarkersWanted() {
		t.Fatal("no exporter is on, yet the markers are wanted - emission would change for every run")
	}
	WhyEnabled = true
	if !TraceMarkersWanted() {
		t.Fatal("-why needs js_srcpos markers or it can attribute nothing")
	}
}

// maybeDumpWhy is reached from three places (llvm.Run, llvm.RunJS and the -exe
// link), and a grammar that both runs and links its module reaches two of them.
func TestWhyReportsAModuleOnlyOnce(t *testing.T) {
	prevWhy, prevOut := WhyEnabled, WhyOut
	defer func() { WhyEnabled, WhyOut = prevWhy, prevOut }()
	WhyEnabled = true
	var out strings.Builder
	WhyOut = &out
	m := whyFixture(t)
	maybeDumpWhy(m)
	first := out.Len()
	maybeDumpWhy(m)
	if out.Len() != first {
		t.Errorf("the same module was reported twice (%d then %d bytes)", first, out.Len())
	}
	if first == 0 {
		t.Error("nothing was reported at all")
	}
}
