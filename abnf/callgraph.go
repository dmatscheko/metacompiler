package abnf

// The static call graph (-callgraph): extracted from a compiled module without
// running it. For the integer-IR languages the calls are direct instructions;
// for the handle-IR languages the dispatch goes through js_call/js_mcall, but
// the names survive statically - method names and js_scope_get variable names
// are string constants (emitStr globals), and a closure pairs with its source
// name where it is stored (js_tdecl("fib", js_closure(N, scope))).
//
// Because the dynamic languages never resolve names at compile time, a file
// that calls functions defined elsewhere still compiles alone: compiling every
// file of a codebase with `-callgraph graph.jsonl` (append mode) and rendering
// once with `-render static` yields a codebase wide graph whose cross-file
// edges connect by name.

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/value"
)

// CallgraphOutPath is set from the -callgraph CLI flag: a .jsonl path appends
// sdef/scall records for a later -render static, anything else writes DOT.
var CallgraphOutPath string

type cgDef struct {
	Name string
	File string
	Line int
}

type cgCall struct {
	From string
	To   string
	File string
}

var (
	cgMu        sync.Mutex
	cgFileCount int
)

// maybeDumpCallgraph extracts and writes the static call graph of a module
// about to be executed (hooked next to maybeDumpCFG).
func maybeDumpCallgraph(m *ir.Module) {
	if CallgraphOutPath == "" {
		return
	}
	defs, calls := extractCallGraph(m)
	if strings.HasSuffix(CallgraphOutPath, ".jsonl") {
		appendCallgraphRecords(defs, calls)
		return
	}
	cgMu.Lock()
	n := cgFileCount
	cgFileCount++
	cgMu.Unlock()
	path := CallgraphOutPath
	if n > 0 {
		if dot := strings.LastIndex(path, "."); dot > 0 {
			path = fmt.Sprintf("%s-%d%s", path[:dot], n+1, path[dot:])
		} else {
			path = fmt.Sprintf("%s-%d", path, n+1)
		}
	}
	var buf strings.Builder
	writeStaticDot(defs, calls, &buf)
	if err := os.WriteFile(path, []byte(buf.String()), 0644); err != nil {
		fmt.Fprintln(os.Stderr, "callgraph dump failed: ", err)
	}
}

// appendCallgraphRecords accumulates the records of one module in the shared
// .jsonl file; several mec runs on different source files merge there.
func appendCallgraphRecords(defs []cgDef, calls []cgCall) {
	f, err := os.OpenFile(CallgraphOutPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Fprintln(os.Stderr, "callgraph append failed: ", err)
		return
	}
	defer f.Close()
	for _, d := range defs {
		line, _ := json.Marshal(&TraceEvent{Ev: "sdef", Name: d.Name, Line: d.Line, Obj: d.File})
		f.Write(append(line, '\n'))
	}
	for _, c := range calls {
		line, _ := json.Marshal(&TraceEvent{Ev: "scall", Name: c.From, Key: c.To, Obj: c.File})
		f.Write(append(line, '\n'))
	}
}

// extractCallGraph walks a module and recovers definitions and call edges.
func extractCallGraph(m *ir.Module) ([]cgDef, []cgCall) {
	strOf := map[value.Value]string{}     // js_str_mem call -> its string literal
	nameRead := map[value.Value]string{}  // js_scope_get/js_kget call -> the variable name
	closureFn := map[value.Value]string{} // js_closure call -> IR function name jsf_N

	forEachCall := func(visit func(f *ir.Func, call *ir.InstCall, callee string)) {
		for _, f := range m.Funcs {
			for _, b := range f.Blocks {
				for _, inst := range b.Insts {
					if call, ok := inst.(*ir.InstCall); ok {
						visit(f, call, strings.TrimPrefix(call.Callee.Ident(), "@"))
					}
				}
			}
		}
	}

	// Pass 1: string literals and closure creations (values compare by pointer).
	forEachCall(func(f *ir.Func, call *ir.InstCall, callee string) {
		switch callee {
		case "js_str_mem":
			if s, ok := decodeStrMem(call); ok {
				strOf[call] = s
			}
		case "js_closure":
			if len(call.Args) > 0 {
				if ci, ok := call.Args[0].(*constant.Int); ok {
					closureFn[call] = fmt.Sprintf("jsf_%d", ci.X.Int64())
				}
			}
		}
	})

	// Pass 2: name reads, and the source names of the IR functions (a closure
	// gets its name where it is stored: a declaration, an assignment, or the
	// method slot of a class descriptor).
	fnName := map[string]string{}
	forEachCall(func(f *ir.Func, call *ir.InstCall, callee string) {
		switch callee {
		case "js_scope_get", "js_kget":
			if len(call.Args) >= 2 {
				if s, ok := strOf[call.Args[1]]; ok {
					nameRead[call] = s
				}
			}
		case "js_get":
			// A member read feeding a call: 'new Counter(...)' compiles as
			// js_call(js_get(js_scope_get("Counter"), "__ctor")). Instructions
			// are visited in emission order, so the base name is already known.
			if len(call.Args) >= 2 {
				if key, ok := strOf[call.Args[1]]; ok {
					base, hasBase := nameRead[call.Args[0]]
					switch {
					case key == "__ctor" && hasBase:
						nameRead[call] = "new " + base
					case key == "__ctor":
						nameRead[call] = "(new)"
					default:
						nameRead[call] = key
					}
				}
			}
		case "js_tdecl", "js_scope_decl", "js_pyset_var", "js_tset", "js_scope_set", "js_kset", "js_set":
			if len(call.Args) >= 3 {
				if fn, ok := closureFn[call.Args[2]]; ok {
					if s, ok2 := strOf[call.Args[1]]; ok2 {
						if _, exists := fnName[fn]; !exists {
							fnName[fn] = s
						}
					}
				}
			}
		}
	})
	display := func(irName string) string {
		if s, ok := fnName[irName]; ok {
			return s
		}
		return irName
	}

	// Pass 3: definitions and edges. The module entry scaffolding (jsmain and
	// the shared-scope variant jsrun) is not program structure: its edges come
	// from "(top)" and it never registers as a definition.
	scaffold := func(n string) bool { return n == "jsmain" || n == "jsrun" }
	var defs []cgDef
	var calls []cgCall
	for _, f := range m.Funcs {
		if len(f.Blocks) == 0 {
			continue // Extern declaration, not a definition.
		}
		from := display(f.Name())
		if scaffold(f.Name()) {
			from = "(top)"
		}
		line := 0
		for _, b := range f.Blocks {
			for _, inst := range b.Insts {
				call, ok := inst.(*ir.InstCall)
				if !ok {
					continue
				}
				callee := strings.TrimPrefix(call.Callee.Ident(), "@")
				switch {
				case callee == "js_srcpos":
					if len(call.Args) > 0 {
						if ci, ok := call.Args[0].(*constant.Int); ok {
							if l := lineOfPos(int(ci.X.Int64())); l > 0 && (line == 0 || l < line) {
								line = l
							}
						}
					}
				case callee == "js_call":
					to := "(dynamic)"
					if len(call.Args) > 0 {
						if s, ok := nameRead[call.Args[0]]; ok {
							to = s
						} else if fn, ok := closureFn[call.Args[0]]; ok {
							to = display(fn)
						}
					}
					calls = append(calls, cgCall{from, to, traceSrcName})
				case callee == "js_mcall":
					to := "(dynamic)"
					if len(call.Args) >= 2 {
						if s, ok := strOf[call.Args[1]]; ok {
							to = s
						}
					}
					calls = append(calls, cgCall{from, to, traceSrcName})
				case callee == "js_supercall":
					to := "(dynamic)"
					if len(call.Args) >= 3 {
						if s, ok := strOf[call.Args[2]]; ok {
							to = s
						}
					}
					calls = append(calls, cgCall{from, to, traceSrcName})
				case strings.HasPrefix(callee, "js_"):
					// Runtime fabric, not program structure.
				case scaffold(callee):
					// jsmain calling jsrun: scaffolding, not an edge.
				default:
					// A direct call (the integer-IR languages, putchar, ...).
					calls = append(calls, cgCall{from, callee, traceSrcName})
				}
			}
		}
		if !scaffold(f.Name()) {
			defs = append(defs, cgDef{Name: from, File: traceSrcName, Line: line})
		}
	}
	return defs, calls
}

// decodeStrMem resolves the string literal behind a js_str_mem(ptr, len) call:
// the pointer is a GEP into a global char array (emitStr in compile-core.js).
func decodeStrMem(call *ir.InstCall) (string, bool) {
	if len(call.Args) < 1 {
		return "", false
	}
	gep, ok := call.Args[0].(*constant.ExprGetElementPtr)
	if !ok {
		return "", false
	}
	g, ok := gep.Src.(*ir.Global)
	if !ok {
		return "", false
	}
	arr, ok := g.Init.(*constant.CharArray)
	if !ok {
		return "", false
	}
	return string(arr.X), true
}

// writeStaticDot renders definitions and call edges as DOT: one cluster per
// source file, names merge across files, undefined callees are dashed.
func writeStaticDot(defs []cgDef, calls []cgCall, out io.Writer) {
	type defInfo struct {
		file string
		line int
	}
	defined := map[string]defInfo{}
	for _, d := range defs {
		if _, exists := defined[d.Name]; !exists {
			defined[d.Name] = defInfo{d.File, d.Line}
		}
	}
	edges := map[cgCall]int{}
	external := map[string]bool{}
	for _, c := range calls {
		edges[cgCall{c.From, c.To, ""}]++
		if _, ok := defined[c.To]; !ok {
			external[c.To] = true
		}
	}

	fmt.Fprintf(out, "digraph callgraph {\n\trankdir=LR;\n\tnode [shape=box, fontname=\"Courier\", fontsize=10];\n")

	// One cluster per file, functions sorted.
	byFile := map[string][]string{}
	for name, info := range defined {
		byFile[info.file] = append(byFile[info.file], name)
	}
	var files []string
	for file := range byFile {
		files = append(files, file)
	}
	sort.Strings(files)
	for fi, file := range files {
		names := byFile[file]
		sort.Strings(names)
		fmt.Fprintf(out, "\tsubgraph cluster_%d {\n\t\tlabel=%q;\n", fi, file)
		for _, name := range names {
			label := name
			if l := defined[name].line; l > 0 {
				label = fmt.Sprintf("%s\nL%d", name, l)
			}
			fmt.Fprintf(out, "\t\t%q [label=%q];\n", name, label)
		}
		fmt.Fprintf(out, "\t}\n")
	}

	var externals []string
	for name := range external {
		externals = append(externals, name)
	}
	sort.Strings(externals)
	for _, name := range externals {
		fmt.Fprintf(out, "\t%q [style=dashed];\n", name)
	}

	var keys []cgCall
	for k := range edges {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].From != keys[j].From {
			return keys[i].From < keys[j].From
		}
		return keys[i].To < keys[j].To
	})
	for _, k := range keys {
		attr := ""
		if n := edges[k]; n > 1 {
			attr = fmt.Sprintf(" [label=\"%d\"]", n)
		}
		fmt.Fprintf(out, "\t%q -> %q%s;\n", k.From, k.To, attr)
	}
	fmt.Fprintf(out, "}\n")
}
