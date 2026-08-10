package abnf

// -why: which extern did the emitter choose, for which line of the program?
//
// Layer-2 work starts with a question of the form "what does kotlin's `+`
// actually lower to". Answering it by reading the grammar is unreliable, and the
// reason is structural, not a matter of effort: an emitter reaches its extern
// through several overridable indirections. python's binExt says `%` is
// "js_fmod" and that entry is TRUE as documentation, but the call the module
// actually contains is js_pybin("%", l, r) - the dunder dispatcher the grammar
// installed one layer up. Reading binExt gives the wrong answer with no hint
// that it is wrong.
//
// The module is the only place the answer is not a claim. So -why does not ask
// the grammar anything: it walks the module the grammar just emitted, and
// attributes each extern call to a source line using the js_srcpos statement
// markers compile-core.js's stmtPos already emits under c.tracing (the same
// mechanism -trace, -cfgraph and -callgraph use). Nothing in the emitter is
// hooked, wrapped or replaced, so with the flag off the module is byte for byte
// the module of a build that does not have this file.
//
// What it cannot see is listed on WhyEnabled below.

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/value"
)

// WhyEnabled (-why) prints the extern-per-source-line report for every module
// the run emits. WhyFilter (-why=NAME) keeps only the lines whose report
// mentions an extern whose name contains NAME - the inverted question, "which
// constructs reach js_ktadd".
//
// Four things the report cannot show:
//
//   - Granularity is the STATEMENT, because js_srcpos is. Two operators on one
//     line share one entry; the order within the entry is emission order, which
//     usually reads as evaluation order but is not guaranteed to.
//   - A grammar that never applies stmtPos has no markers at all (ruby, php and
//     the self-contained-IR languages). Their calls land in the unattributed
//     group, which is still grouped by emitted function - coarse, not useless.
//   - It reports the STATIC choice. An emitter that branches at run time (the
//     kotlin `+` String probe, the operator-convention probe) shows every arm,
//     with no indication of which one a given execution takes.
//   - It sees the module, so it sees the extern NAME, not its body. Whether
//     js_ktadd is the Go twin in abnf/jsrtkotlin.go or the layer-2 shim in
//     lib/kotlin-rt.metajs is a question for -exe, not for this.
//
// It costs one extra js_srcpos call per statement in the module it reports on
// (exactly what -trace costs), and one linear walk of the module.
var (
	WhyEnabled bool
	WhyFilter  string
	WhyOut     io.Writer = os.Stderr // Overridden by the tests.
)

// whyDone guards against reporting the same module twice: a compiler grammar
// that both runs its module and links it with -exe reaches two hooks.
var whyDone = map[*ir.Module]bool{}

// whyEntry is one source statement's worth of emitted extern calls.
type whyEntry struct {
	file  string
	line  int
	calls []string // Display forms, in emission order.
}

// maybeDumpWhy writes the -why report for a module, once.
func maybeDumpWhy(m *ir.Module) {
	if !WhyEnabled || m == nil || whyDone[m] {
		return
	}
	whyDone[m] = true
	writeWhy(m, WhyOut)
}

// whyDisplay renders one call as `js_get(_,"length")`: every argument that is a
// decodable js_str_mem string literal is shown, the rest as `_`, and a trailing
// run of `_` is dropped. A call with no string argument shows as its bare name,
// which is the common case and keeps the report readable.
func whyDisplay(callee string, args []value.Value, strOf map[value.Value]string) string {
	shown := make([]string, len(args))
	last := -1
	for i, a := range args {
		if s, ok := strOf[a]; ok {
			shown[i] = strconv.Quote(s)
			last = i
			continue
		}
		if ci, ok := a.(*constant.Int); ok {
			shown[i] = ci.X.String()
			continue
		}
		shown[i] = "_"
	}
	if last < 0 {
		return callee
	}
	return callee + "(" + strings.Join(shown[:last+1], ",") + ")"
}

// whyHelpers answers the module-level functions a grammar emitted for itself:
// defined in the module, but carrying no js_srcpos marker anywhere, so no source
// statement is theirs. swift routes every member read through one of these, and
// without expanding them the report would answer "sw_get" to a question whose
// real answer is js_swget - so the report follows them, transitively.
//
// A user function compiled from the program IS defined in the module too, but it
// has markers of its own and therefore its own attributed lines; it is excluded
// by the same test, with no name matching involved.
func whyHelpers(m *ir.Module) map[string]*ir.Func {
	out := map[string]*ir.Func{}
	for _, f := range m.Funcs {
		if len(f.Blocks) == 0 {
			continue
		}
		marked := false
		for _, b := range f.Blocks {
			for _, inst := range b.Insts {
				if call, ok := inst.(*ir.InstCall); ok &&
					strings.TrimPrefix(call.Callee.Ident(), "@") == "js_srcpos" {
					marked = true
				}
			}
		}
		if !marked {
			out[f.Name()] = f
		}
	}
	return out
}

// collectWhy walks the module in emission order and buckets every extern call
// under the source position of the js_srcpos marker that most recently preceded
// it. Position is reset at each function boundary, so a function emitted with no
// marker of its own does not inherit the previous function's last line.
//
// Blocks are visited in creation order, which is the order the emitter made
// them: compile-core.js and every grammar call curF.NewBlock() at the point they
// emit into it. That is emission order, not execution order - the arms of a
// branch appear one after the other, which is exactly what the report wants.
func collectWhy(m *ir.Module) ([]whyEntry, map[string]int, int) {
	strOf := whyStrings(m) // So an extern's name arguments can be shown.

	// The same stamping the call graph does: functions compiled after the last
	// import popped still carry no file, and a multi-file program would attribute
	// them to whichever source happened to be current.
	attributeModuleFuncs()
	srcOf := func(f *ir.Func) (string, []int) {
		if fs, ok := compileFuncFile[f.Name()]; ok {
			return fs.file, fs.starts
		}
		return traceSrcName, traceLineStarts
	}

	index := map[string]int{} // "file:line" -> position in entries.
	var entries []whyEntry
	unattributed := map[string]int{}
	total := 0
	at := func(file string, line int) int {
		key := file + ":" + strconv.Itoa(line)
		if i, ok := index[key]; ok {
			return i
		}
		index[key] = len(entries)
		entries = append(entries, whyEntry{file: file, line: line})
		return len(entries) - 1
	}

	for _, f := range m.Funcs {
		if len(f.Blocks) == 0 {
			continue // An extern declaration, not a definition.
		}
		file, starts := srcOf(f)
		line := 0
		for _, b := range f.Blocks {
			for _, inst := range b.Insts {
				call, ok := inst.(*ir.InstCall)
				if !ok {
					continue
				}
				callee := strings.TrimPrefix(call.Callee.Ident(), "@")
				if callee == "js_srcpos" {
					// The marker itself: it moves the cursor and is not reported.
					if len(call.Args) > 0 {
						if ci, ok := call.Args[0].(*constant.Int); ok {
							line = lineOfPosIn(int(ci.X.Int64()), starts)
						}
					}
					continue
				}
				if callee == "js_str_mem" {
					continue // Folded into the display of whatever consumes it.
				}
				total++
				disp := whyDisplay(callee, call.Args, strOf)
				if line == 0 {
					unattributed[f.Name()]++
					continue
				}
				i := at(file, line)
				entries[i].calls = append(entries[i].calls, disp)
			}
		}
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].file != entries[j].file {
			return entries[i].file < entries[j].file
		}
		return entries[i].line < entries[j].line
	})
	return entries, unattributed, total
}

// runLengths collapses a run of identical display forms into `name xN`, which is
// what turns a 40-call statement into three readable lines.
func runLengths(calls []string) []string {
	var out []string
	for i := 0; i < len(calls); {
		j := i
		for j < len(calls) && calls[j] == calls[i] {
			j++
		}
		if j-i > 1 {
			out = append(out, fmt.Sprintf("%s x%d", calls[i], j-i))
		} else {
			out = append(out, calls[i])
		}
		i = j
	}
	return out
}

// writeWhy renders the report. Shape (kotlin, `println(a + b)` on line 4):
//
//	-why p/a.kt: 359 call(s) emitted, 13 distinct callee(s) across 6 attributed statement(s)
//	p/a.kt:4               println(a + b)
//	                         kt_ktgetp(_,"println") js_arr_new kt_ktget(_,"a") kt_ktget(_,"b")
//	                         js_is_type(_,"String") js_truthy x2 js_ktstr x2
//	                         js_jadd js_ktadd js_arr_push js_call
//	kt_ktget               (grammar helper, no source line of its own)
//	                         kt_ktgetp
//	-why: 319 call(s) in 11 function(s) carry no source position (...)
//
// The statement TEXT is quoted next to its position on purpose: a grammar whose
// stmtPos fires at the wrong position then shows the wrong line, visibly,
// instead of quietly mislabelling the answer. (python's does - it reports the
// position AFTER the statement, which -trace and -cfgraph already inherit.)
func writeWhy(m *ir.Module, out io.Writer) {
	entries, unattributed, total := collectWhy(m)
	helpers := whyHelpers(m)
	distinct := map[string]bool{}
	for _, e := range entries {
		for _, c := range e.calls {
			distinct[externOf(c)] = true
		}
	}
	name := m.SourceFilename
	if name == "" {
		name = traceSrcName
	}
	if name == "" {
		name = "module"
	}
	fmt.Fprintf(out, "-why %s: %d call(s) emitted, %d distinct callee(s) across %d attributed statement(s)\n",
		name, total, len(distinct), len(entries))
	if len(entries) == 0 {
		fmt.Fprintf(out, "-why: no js_srcpos markers in this module - the grammar does not apply "+
			"compile-core's stmtPos, so nothing can be attributed to a source line. "+
			"The helper listing below is the whole report.\n")
	}

	shown := 0
	want := map[string]bool{} // Helpers reachable from what was printed.
	for _, e := range entries {
		if WhyFilter != "" && !whyMatches(e.calls, WhyFilter) {
			continue
		}
		shown++
		head := fmt.Sprintf("%s:%d", e.file, e.line)
		fmt.Fprintf(out, "%-22s %s\n", head, strings.TrimSpace(sourceLine(e.file, e.line)))
		for _, chunk := range wrapCalls(runLengths(e.calls), 80) {
			fmt.Fprintf(out, "%-22s   %s\n", "", chunk)
		}
		for _, c := range e.calls {
			if _, ok := helpers[externOf(c)]; ok {
				want[externOf(c)] = true
			}
		}
	}
	if WhyFilter != "" {
		fmt.Fprintf(out, "-why: filter %q matched %d of %d statement(s)\n", WhyFilter, shown, len(entries))
	}

	// The helper bodies, transitively: a member read that lands on sw_get is not
	// an answer until sw_get's own body is on the page. With a filter, a helper
	// that MATCHES it is seeded too - that is the whole report for a grammar
	// with no markers (ruby, php), where nothing else can point at one.
	strOf := whyStrings(m)
	bodies := map[string][]string{}
	for n, f := range helpers {
		var calls []string
		for _, b := range f.Blocks {
			for _, inst := range b.Insts {
				call, ok := inst.(*ir.InstCall)
				if !ok {
					continue
				}
				callee := strings.TrimPrefix(call.Callee.Ident(), "@")
				if callee == "js_str_mem" || callee == "js_srcpos" {
					continue
				}
				calls = append(calls, whyDisplay(callee, call.Args, strOf))
			}
		}
		bodies[n] = calls
		if WhyFilter != "" && whyMatches(calls, WhyFilter) {
			want[n] = true
		}
		// A grammar with no markers has no statement to reach a helper FROM, so
		// an unfiltered report there is the helper list itself - coarse, but it
		// is the whole of what the module can be asked.
		if len(entries) == 0 && WhyFilter == "" {
			want[n] = true
		}
	}
	printed := map[string]bool{}
	for len(want) > 0 {
		var batch []string
		for n := range want {
			if !printed[n] {
				batch = append(batch, n)
			}
		}
		sort.Strings(batch)
		want = map[string]bool{}
		for _, n := range batch {
			printed[n] = true
			fmt.Fprintf(out, "%-22s (grammar helper, no source line of its own)\n", n)
			for _, chunk := range wrapCalls(runLengths(bodies[n]), 80) {
				fmt.Fprintf(out, "%-22s   %s\n", "", chunk)
			}
			for _, c := range bodies[n] {
				if _, isHelper := helpers[externOf(c)]; isHelper && !printed[externOf(c)] {
					want[externOf(c)] = true
				}
			}
		}
	}

	if len(unattributed) > 0 {
		n, fns := 0, 0
		for _, c := range unattributed {
			n += c
			fns++
		}
		fmt.Fprintf(out, "-why: %d call(s) in %d function(s) carry no source position "+
			"(language prelude, grammar helpers, or a grammar that does not apply stmtPos)\n", n, fns)
	}
}

// whyStrings maps every js_str_mem call in the module to its literal.
func whyStrings(m *ir.Module) map[value.Value]string {
	strOf := map[value.Value]string{}
	for _, f := range m.Funcs {
		for _, b := range f.Blocks {
			for _, inst := range b.Insts {
				if call, ok := inst.(*ir.InstCall); ok &&
					strings.TrimPrefix(call.Callee.Ident(), "@") == "js_str_mem" {
					if s, ok := decodeStrMem(call); ok {
						strOf[call] = s
					}
				}
			}
		}
	}
	return strOf
}

// externOf strips the argument display back to the extern name.
func externOf(disp string) string {
	if i := strings.IndexByte(disp, '('); i > 0 {
		return disp[:i]
	}
	return disp
}

func whyMatches(calls []string, filter string) bool {
	for _, c := range calls {
		if strings.Contains(c, filter) {
			return true
		}
	}
	return false
}

// wrapCalls packs display forms into lines of at most width characters.
func wrapCalls(items []string, width int) []string {
	var out []string
	cur := ""
	for _, it := range items {
		if cur == "" {
			cur = it
			continue
		}
		if len(cur)+1+len(it) > width {
			out = append(out, cur)
			cur = it
			continue
		}
		cur += " " + it
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

// sourceLine answers the text of one 1-based line of a registered source, or ""
// when the source is not known (an imported file whose text was not kept).
func sourceLine(file string, line int) string {
	text, ok := sourceTexts[file]
	if !ok {
		return ""
	}
	starts := sourceLineStarts[file]
	if line < 1 || line > len(starts) {
		return ""
	}
	from := starts[line-1]
	to := len(text)
	if line < len(starts) {
		to = starts[line]
	}
	if from > len(text) {
		return ""
	}
	return strings.TrimRight(text[from:to], "\r\n")
}
