package abnf

// The unified tracing layer: one event model through all compiled languages.
//
// Every dynamic language compiles to handle-IR whose semantics run through the
// js_* externals of one shared runtime (abnf/jsrt.go) - so hooking that runtime
// traces Java, Kotlin, Go, Python, Lisp and MetaJS programs alike, under goja
// and under -frozen (llvm.RunJS is engine independent). The integer-IR
// languages (TinyC, C subset) have no jsrt events, but their modules still get
// the -cfg control flow dump.
//
//	-cfg out.dot      write the CFG of every executed module (Mermaid with .mmd)
//	-trace out.jsonl  stream runtime events (decl/read/write/mread/mwrite/call/ret)
//	-render calls     turn a -trace file into a dynamic call graph (DOT, stdout)
//	-render vars      ... into a function/variable access graph (DOT, stdout)
//
// Only the program runtime is traced (rt.enableTrace in runJSModule); the
// frozen engine's tag-script runtime stays silent, so a goja run and a -frozen
// run of the same program produce the same event stream.

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
)

// CFGOutPath and TraceOutPath are set from the -cfg and -trace CLI flags.
var (
	CFGOutPath   string
	TraceOutPath string
)

// TraceMarkersWanted reports whether the compilers should emit js_srcpos
// statement markers (the c.tracing value the tag scripts read): positions
// serve the event stream, the CFG line annotations, and the call graph
// definition lines.
func TraceMarkersWanted() bool {
	return TraceOutPath != "" || CFGOutPath != "" || CallgraphOutPath != ""
}

// SetTraceSource registers the program source, so events and CFG labels can
// carry line numbers instead of raw byte offsets.
func SetTraceSource(name, text string) {
	traceSrcName = name
	traceLineStarts = []int{0}
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' {
			traceLineStarts = append(traceLineStarts, i+1)
		}
	}
}

var (
	traceSrcName    string
	traceLineStarts []int // Byte offset of every line start; nil = no source known.
)

// lineOfPos converts a byte offset to a 1-based line number (0 = unknown).
func lineOfPos(pos int) int {
	if traceLineStarts == nil || pos < 0 {
		return 0
	}
	lo, hi := 0, len(traceLineStarts)-1
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if traceLineStarts[mid] <= pos {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return lo + 1
}

// ----------------------------------------------------------------------------
// The runtime event stream (-trace)

// TraceEvent is one line of the JSONL stream. Ev is one of: stmt (a statement
// begins), decl, read, write (scope variables), mread, mwrite (object
// members), call, ret.
type TraceEvent struct {
	Seq   int64  `json:"seq"`
	Ev    string `json:"ev"`
	Depth int    `json:"depth"`
	Line  int    `json:"line,omitempty"` // 1-based source line of the executing statement.
	Name  string `json:"name,omitempty"` // Variable or callee name.
	Key   string `json:"key,omitempty"`  // Member key of mread/mwrite.
	Obj   string `json:"obj,omitempty"`  // Container of mread/mwrite: "type#handle".
	Val   string `json:"val,omitempty"`  // The value, rendered and capped.
}

var (
	traceMu   sync.Mutex
	traceFile *os.File // Unbuffered on purpose: exit() ends the process abruptly.
	traceSeq  int64
	traceDead bool
)

// OpenTrace creates (and truncates) the -trace file up front. Without the
// eager create, a run that emits no events (interpreter-engine grammars trace
// nothing) silently left a STALE file from an earlier run in place, and a
// later -render rendered the wrong program.
func OpenTrace() {
	traceMu.Lock()
	defer traceMu.Unlock()
	if traceFile != nil || traceDead || TraceOutPath == "" {
		return
	}
	f, err := os.Create(TraceOutPath)
	if err != nil {
		traceDead = true
		fmt.Fprintln(os.Stderr, "trace failed: ", err)
		return
	}
	traceFile = f
}

func traceEmit(ev *TraceEvent) {
	traceMu.Lock()
	defer traceMu.Unlock()
	if traceFile == nil {
		if traceDead || TraceOutPath == "" {
			return
		}
		f, err := os.Create(TraceOutPath)
		if err != nil {
			traceDead = true
			fmt.Fprintln(os.Stderr, "trace failed: ", err)
			return
		}
		traceFile = f
	}
	traceSeq++
	ev.Seq = traceSeq
	line, err := json.Marshal(ev)
	if err != nil {
		return
	}
	// A failed write (full disk...) must not silently truncate the stream
	// with exit 0: report it once and stop tracing.
	if _, err := traceFile.Write(append(line, '\n')); err != nil {
		fmt.Fprintln(os.Stderr, "trace write failed: ", err)
		traceFile.Close()
		traceFile = nil
		traceDead = true
	}
}

// CloseTrace closes the stream (writes are unbuffered, nothing to flush).
func CloseTrace() {
	traceMu.Lock()
	defer traceMu.Unlock()
	if traceFile != nil {
		traceFile.Close()
		traceFile = nil
	}
}

// enableTrace marks this runtime as the traced program runtime. Only
// runJSModule calls it: the frozen engine's tag-script runtime stays untraced,
// so both engines produce the same stream for the same program.
func (rt *jsrt) enableTrace() {
	if TraceOutPath == "" {
		return
	}
	rt.traced = true
	rt.traceNames = map[*jsClosure]string{}
}

// traceVal renders a value for the stream, capped so events stay one-liners.
func (rt *jsrt) traceVal(v interface{}) string {
	s := rt.toString(v)
	if len(s) > 100 {
		s = s[:100] + "..."
	}
	return s
}

// noteClosureName remembers under which name a closure was stored; call events
// use it (a method closure gets its name when the class descriptor is built).
func (rt *jsrt) noteClosureName(name string, v interface{}) {
	if c, ok := v.(*jsClosure); ok && name != "" {
		if _, exists := rt.traceNames[c]; !exists {
			rt.traceNames[c] = name
		}
	}
}

// trVar traces a scope variable event: decl, read or write.
func (rt *jsrt) trVar(ev, name string, v interface{}) {
	rt.noteClosureName(name, v)
	traceEmit(&TraceEvent{Ev: ev, Depth: rt.traceDepth, Line: lineOfPos(rt.curPos), Name: name, Val: rt.traceVal(v)})
}

// trMember traces an object member event: mread or mwrite.
func (rt *jsrt) trMember(ev string, objH uint64, key interface{}, v interface{}) {
	keyS := rt.toString(key)
	rt.noteClosureName(keyS, v)
	obj := fmt.Sprintf("%s#%d", rt.typeOf(rt.unwrap(objH)), objH)
	traceEmit(&TraceEvent{Ev: ev, Depth: rt.traceDepth, Line: lineOfPos(rt.curPos), Key: keyS, Obj: obj, Val: rt.traceVal(v)})
}

// calleeName resolves what a call event should be called.
func (rt *jsrt) calleeName(callee interface{}) string {
	switch c := callee.(type) {
	case *jsClosure:
		if n, ok := rt.traceNames[c]; ok {
			return n
		}
		return c.fn.Name()
	case *hostFunc:
		return c.name
	case *boundMethod:
		return c.name
	}
	return "(host)"
}

// ----------------------------------------------------------------------------
// The control flow dump (-cfg)

var (
	cfgMu    sync.Mutex
	cfgCount int
)

// maybeDumpCFG writes the block graph of a module about to be executed.
// Multiple executed modules in one run get numbered files.
func maybeDumpCFG(m *ir.Module) {
	if CFGOutPath == "" {
		return
	}
	cfgMu.Lock()
	n := cfgCount
	cfgCount++
	cfgMu.Unlock()

	path := CFGOutPath
	if n > 0 {
		if dot := strings.LastIndex(path, "."); dot > 0 {
			path = fmt.Sprintf("%s-%d%s", path[:dot], n+1, path[dot:])
		} else {
			path = fmt.Sprintf("%s-%d", path, n+1)
		}
	}
	var buf strings.Builder
	if strings.HasSuffix(path, ".mmd") {
		writeCFGMermaid(m, &buf)
	} else {
		writeCFGDot(m, &buf)
	}
	if err := os.WriteFile(path, []byte(buf.String()), 0644); err != nil {
		fmt.Fprintln(os.Stderr, "cfg dump failed: ", err)
	}
}

type cfgEdge struct {
	to    *ir.Block
	label string
}

// blockEdges lists the successors of a block from its terminator.
func blockEdges(b *ir.Block) []cfgEdge {
	switch term := b.Term.(type) {
	case *ir.TermBr:
		return []cfgEdge{{term.Target.(*ir.Block), ""}}
	case *ir.TermCondBr:
		return []cfgEdge{
			{term.TargetTrue.(*ir.Block), "T"},
			{term.TargetFalse.(*ir.Block), "F"},
		}
	}
	return nil // ret and everything else: no successors.
}

// blockTitle names a block for display (many are unnamed in the emitted IR).
func blockTitle(b *ir.Block, idx int) string {
	if b.LocalName != "" {
		return b.LocalName
	}
	return fmt.Sprintf("b%d", idx)
}

// blockCalls lists the non-js_* functions a block calls (the js_* externals
// are the fabric of handle-IR, not the interesting structure).
func blockCalls(b *ir.Block) []string {
	var out []string
	seen := map[string]bool{}
	for _, inst := range b.Insts {
		if call, ok := inst.(*ir.InstCall); ok {
			name := strings.TrimPrefix(call.Callee.Ident(), "@")
			if strings.HasPrefix(name, "js_") || seen[name] {
				continue
			}
			seen[name] = true
			out = append(out, name)
		}
	}
	return out
}

// blockLines derives the source line range of a block from its js_srcpos
// markers (present when the module was compiled with -trace or -cfg active).
func blockLines(b *ir.Block) string {
	lo, hi := 0, 0
	for _, inst := range b.Insts {
		call, ok := inst.(*ir.InstCall)
		if !ok || strings.TrimPrefix(call.Callee.Ident(), "@") != "js_srcpos" || len(call.Args) == 0 {
			continue
		}
		ci, ok := call.Args[0].(*constant.Int)
		if !ok {
			continue
		}
		line := lineOfPos(int(ci.X.Int64()))
		if line == 0 {
			continue
		}
		if lo == 0 || line < lo {
			lo = line
		}
		if line > hi {
			hi = line
		}
	}
	switch {
	case lo == 0:
		return ""
	case lo == hi:
		return fmt.Sprintf("L%d", lo)
	default:
		return fmt.Sprintf("L%d-%d", lo, hi)
	}
}

func writeCFGDot(m *ir.Module, buf *strings.Builder) {
	buf.WriteString("digraph CFG {\n\trankdir=TB;\n\tnode [shape=box, fontname=\"Courier\", fontsize=10];\n")
	for fi, f := range m.Funcs {
		if len(f.Blocks) == 0 {
			continue
		}
		ids := map[*ir.Block]string{}
		for bi, b := range f.Blocks {
			ids[b] = fmt.Sprintf("f%d.%d", fi, bi)
		}
		fmt.Fprintf(buf, "\tsubgraph cluster_%d {\n\t\tlabel=%q;\n", fi, f.Name())
		for bi, b := range f.Blocks {
			label := fmt.Sprintf("%s\n%d inst", blockTitle(b, bi), len(b.Insts))
			if lines := blockLines(b); lines != "" {
				label = fmt.Sprintf("%s %s\n%d inst", blockTitle(b, bi), lines, len(b.Insts))
			}
			if calls := blockCalls(b); len(calls) > 0 {
				label += "\ncalls: " + strings.Join(calls, ", ")
			}
			extra := ""
			if _, isRet := b.Term.(*ir.TermRet); isRet {
				extra = ", peripheries=2"
			}
			fmt.Fprintf(buf, "\t\t%q [label=%q%s];\n", ids[b], label, extra)
		}
		buf.WriteString("\t}\n")
		for _, b := range f.Blocks {
			for _, e := range blockEdges(b) {
				attr := ""
				if e.label != "" {
					attr = fmt.Sprintf(" [label=%q]", e.label)
				}
				fmt.Fprintf(buf, "\t%q -> %q%s;\n", ids[b], ids[e.to], attr)
			}
		}
	}
	buf.WriteString("}\n")
}

func writeCFGMermaid(m *ir.Module, buf *strings.Builder) {
	buf.WriteString("flowchart TD\n")
	for fi, f := range m.Funcs {
		if len(f.Blocks) == 0 {
			continue
		}
		ids := map[*ir.Block]string{}
		for bi, b := range f.Blocks {
			ids[b] = fmt.Sprintf("f%db%d", fi, bi)
		}
		fmt.Fprintf(buf, "  subgraph sg%d[%q]\n", fi, f.Name())
		for bi, b := range f.Blocks {
			title := blockTitle(b, bi)
			if lines := blockLines(b); lines != "" {
				title += " " + lines
			}
			fmt.Fprintf(buf, "    %s[%q]\n", ids[b], fmt.Sprintf("%s (%d)", title, len(b.Insts)))
		}
		buf.WriteString("  end\n")
		for _, b := range f.Blocks {
			for _, e := range blockEdges(b) {
				if e.label != "" {
					fmt.Fprintf(buf, "  %s -->|%s| %s\n", ids[b], e.label, ids[e.to])
				} else {
					fmt.Fprintf(buf, "  %s --> %s\n", ids[b], ids[e.to])
				}
			}
		}
	}
}

// ----------------------------------------------------------------------------
// Trace rendering (-render)

// RenderTrace reads a -trace / -callgraph JSONL file and writes a DOT graph to
// stdout. Kinds: "calls" (dynamic call graph, edge labels are call counts),
// "vars" (which function touches which variable; decl/write solid, read
// dashed), and "static" (the -callgraph records: definitions clustered per
// source file, cross-file edges merged by name, undefined callees dashed).
func RenderTrace(kind, path string) error {
	if path == "" {
		return fmt.Errorf("-render needs the -trace <file> flag as its input")
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	type edgeKey struct{ from, to, mode string }
	edges := map[edgeKey]int{}
	varNodes := map[string]bool{}
	fnNodes := map[string]bool{"(top)": true}
	stack := []string{"(top)"}
	var sDefs []cgDef
	var sCalls []cgCall

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1024*1024), 16*1024*1024)
	for sc.Scan() {
		var ev TraceEvent
		if json.Unmarshal(sc.Bytes(), &ev) != nil {
			continue
		}
		cur := stack[len(stack)-1]
		switch ev.Ev {
		case "call":
			edges[edgeKey{cur, ev.Name, "call"}]++
			fnNodes[ev.Name] = true
			stack = append(stack, ev.Name)
		case "ret":
			if len(stack) > 1 {
				stack = stack[:len(stack)-1]
			}
		case "decl", "write":
			edges[edgeKey{cur, ev.Name, "write"}]++
			varNodes[ev.Name] = true
		case "read":
			edges[edgeKey{cur, ev.Name, "read"}]++
			varNodes[ev.Name] = true
		case "sdef":
			sDefs = append(sDefs, cgDef{Name: ev.Name, File: ev.Obj, Line: ev.Line})
		case "scall":
			sCalls = append(sCalls, cgCall{From: ev.Name, To: ev.Key, File: ev.Obj})
		}
	}
	if err := sc.Err(); err != nil {
		return err
	}

	if kind == "static" {
		out := &strings.Builder{}
		writeStaticDot(sDefs, sCalls, out)
		_, err = os.Stdout.WriteString(out.String())
		return err
	}

	var keys []edgeKey
	for k := range edges {
		if kind == "calls" && k.mode != "call" {
			continue
		}
		if kind == "vars" && k.mode == "call" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := keys[i], keys[j]
		if a.from != b.from {
			return a.from < b.from
		}
		if a.to != b.to {
			return a.to < b.to
		}
		return a.mode < b.mode
	})

	out := &strings.Builder{}
	switch kind {
	case "calls":
		out.WriteString("digraph calls {\n\trankdir=LR;\n\tnode [shape=box, fontname=\"Courier\"];\n")
		for _, k := range keys {
			fmt.Fprintf(out, "\t%q -> %q [label=\"%d\"];\n", k.from, k.to, edges[k])
		}
	case "vars":
		out.WriteString("digraph vars {\n\trankdir=LR;\n\tnode [fontname=\"Courier\"];\n")
		var fns, vars []string
		for n := range fnNodes {
			fns = append(fns, n)
		}
		for n := range varNodes {
			vars = append(vars, n)
		}
		sort.Strings(fns)
		sort.Strings(vars)
		for _, n := range fns {
			fmt.Fprintf(out, "\t%q [shape=box];\n", n)
		}
		for _, n := range vars {
			fmt.Fprintf(out, "\tvar_%s [label=%q, shape=ellipse];\n", dotID(n), n)
		}
		for _, k := range keys {
			style := "solid"
			if k.mode == "read" {
				style = "dashed"
			}
			fmt.Fprintf(out, "\t%q -> var_%s [label=\"%d\", style=%s];\n", k.from, dotID(k.to), edges[k], style)
		}
	default:
		return fmt.Errorf("unknown -render kind %q (use 'calls' or 'vars')", kind)
	}
	out.WriteString("}\n")
	_, err = os.Stdout.WriteString(out.String())
	return err
}

// dotID makes a name safe as a bare DOT identifier suffix.
func dotID(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			fmt.Fprintf(&b, "_%02x", r)
		}
	}
	return b.String()
}
