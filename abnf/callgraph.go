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
// file of a codebase with `-callgraph-append graph.jsonl` (which keeps and adds
// to the file) and rendering once with `-render static` yields a codebase wide
// graph whose cross-file edges connect by name. Plain `-callgraph` overwrites
// the file each run, like the other exports.

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

// CallgraphOutPath is set from the -callgraph CLI flag: a .jsonl path holds
// sdef/scall records for a later -render static, anything else writes DOT.
// CallgraphAppend (from -callgraph-append) keeps the existing .jsonl and adds to
// it, so many runs on different files accumulate one codebase-wide graph; the
// default -callgraph overwrites the file up front, like -trace and -cfgraph.
var (
	CallgraphOutPath string
	CallgraphAppend  bool
)

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
	cgDead      bool // OpenCallgraph could not truncate the .jsonl; appends then skip.
)

// OpenCallgraph truncates the -callgraph .jsonl file up front, so a re-run
// overwrites the previous run's graph instead of appending to it - and a run
// that extracts no records still replaces a stale file rather than leaving the
// old one in place (the same reason OpenTrace pre-creates the -trace file). The
// modules of THIS run still accumulate, because maybeDumpCallgraph appends after
// this initial truncate. -callgraph-append (CallgraphAppend) skips the truncate
// to accumulate across many runs; the DOT form writes a whole file per module
// (os.WriteFile truncates) and needs no pre-truncation.
func OpenCallgraph() {
	if CallgraphOutPath == "" || CallgraphAppend || !strings.HasSuffix(CallgraphOutPath, ".jsonl") {
		return
	}
	f, err := os.Create(CallgraphOutPath)
	if err != nil {
		cgDead = true
		fmt.Fprintln(os.Stderr, "callgraph failed: ", err)
		return
	}
	f.Close()
}

// Per-function source attribution. A module built from several files (imports
// compiled into one module, e.g. Kotlin -i) would otherwise stamp EVERY
// function with the one global traceSrcName, collapsing the whole codebase into
// a single file. compileFuncFile records the file (and its line-start table)
// that was active while each IR function was compiled: it is filled at every
// trace-source push/pop (attributeModuleFuncs) and once before extraction, so
// each function keeps its own file and line numbers after the source stack has
// unwound. This is language-agnostic - every language builds via llvm.ir.
// NewModule and brackets imported files with pushSource/popSource.
type funcSrc struct {
	file   string
	starts []int
}

var (
	compileModule   *ir.Module // the module currently being built (set by llvm.ir.NewModule)
	compileFuncFile = map[string]funcSrc{}
)

// attributeModuleFuncs stamps every not-yet-stamped module function with the
// CURRENT trace source. Called just before the source is switched (push/pop)
// and once before extraction.
func attributeModuleFuncs() {
	if compileModule == nil || !TraceMarkersWanted() {
		return
	}
	for _, f := range compileModule.Funcs {
		n := f.Name()
		if _, ok := compileFuncFile[n]; !ok {
			compileFuncFile[n] = funcSrc{traceSrcName, traceLineStarts}
		}
	}
}

// beginCompileModule is called by the llvm.ir.NewModule wrapper: it records the
// module so attributeModuleFuncs can walk it, and clears any prior attribution.
func beginCompileModule(m *ir.Module) {
	compileModule = m
	compileFuncFile = map[string]funcSrc{}
}

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

// appendCallgraphRecords adds the records of one module to the .jsonl file. Each
// run's OpenCallgraph truncated it first (unless -callgraph-append), so this
// accumulates the modules of one run - and, with -callgraph-append, several mec
// runs on different source files merge into one codebase-wide graph.
func appendCallgraphRecords(defs []cgDef, calls []cgCall) {
	if cgDead {
		return
	}
	f, err := os.OpenFile(CallgraphOutPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Fprintln(os.Stderr, "callgraph append failed: ", err)
		return
	}
	defer f.Close()
	// Names such as "<module>" stay readable: no HTML escaping.
	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)
	// A failed write (full disk...) must not silently truncate the graph.
	for _, d := range defs {
		if err := enc.Encode(&TraceEvent{Ev: "sdef", Name: d.Name, Line: d.Line, Obj: d.File}); err != nil {
			fmt.Fprintln(os.Stderr, "callgraph write failed: ", err)
			return
		}
	}
	for _, c := range calls {
		if err := enc.Encode(&TraceEvent{Ev: "scall", Name: c.From, Key: c.To, Obj: c.File}); err != nil {
			fmt.Fprintln(os.Stderr, "callgraph write failed: ", err)
			return
		}
	}
}

// Name-access and call helpers some compilers emit into the module as IR
// functions rather than calling the js_scope_get/js_scope_set/js_call externs
// directly. The name helpers take (scope, name, ...): a reader's result is the
// variable, so a js_call of it calls that name. All of them are runtime fabric,
// neither definitions nor callees.
var (
	cgNameReaders = map[string]bool{
		"py_getvar": true, "sw_kget": true, "dt_kget": true, "lua_getvar": true,
		"kt_ktget": true, "kt_ktgetp": true, "kt_bareref": true,
	}
	cgNameWriters = map[string]bool{
		"py_setvar": true, "sw_kset": true, "dt_kset": true, "kt_kset": true,
	}
	// (scope, name, args): resolve the name and call it.
	cgNameCallers = map[string]bool{"dt_kcall": true, "kt_barecall": true}
	// (fn, args, ...): call a function value, as js_call does.
	cgValueCallers = map[string]bool{"js_pycall": true, "sw_callval": true, "sw_dyncall": true}
)

// cgTailCaller is lua's tail call: its argument is an array whose first element
// (the first js_arr_push into it) is the function to call.
const cgTailCaller = "lua_tramp"

// cgPassThrough answers its first argument (js's temporal-dead-zone check of a
// `let`/`class` binding), so a name read through it is still that name.
const cgPassThrough = "jstdzchk"

func cgHelper(n string) bool {
	return cgNameReaders[n] || cgNameWriters[n] || cgNameCallers[n] || cgValueCallers[n] || n == cgTailCaller
}

// cgMethodCall: every language's method-call extern takes (receiver, method
// name, ...): js_mcall and its per-language variants (js_pymcall, js_swmcall,
// js_phmcallv, ..., dart's js_dartnm).
func cgMethodCall(n string) bool {
	return n == "js_dartnm" || strings.HasPrefix(n, "js_") && (strings.HasSuffix(n, "mcall") || strings.HasSuffix(n, "mcallv"))
}

// cgEntryFuncs are the module entry functions the handle-IR compilers emit: the
// shared jsmain (and its shared-scope variant jsrun), php's and dart's own.
var cgEntryFuncs = map[string]bool{"jsmain": true, "jsrun": true, "phpmain": true, "dartmain": true}

// cgUserFunc: in a handle-IR module every function of the program is a closure
// body jsf_N; any other defined function (jsnum.N constant builders, sw_*,
// lua_*, php_* helpers, ...) is runtime fabric.
func cgUserFunc(n string) bool {
	if !strings.HasPrefix(n, "jsf_") {
		return false
	}
	for _, r := range n[4:] {
		if r < '0' || r > '9' {
			return false
		}
	}
	return len(n) > 4
}

// extractCallGraph walks a module and recovers definitions and call edges.
func extractCallGraph(m *ir.Module) ([]cgDef, []cgCall) {
	strOf := map[value.Value]string{}          // js_str_mem call -> its string literal
	nameRead := map[value.Value]string{}       // js_scope_get/js_kget call -> the variable name
	scopeRead := map[value.Value]bool{}        // nameRead entries that read a variable, not a member
	boundTo := map[string][]string{}           // variable name -> the jsf_N closures stored in it
	firstPush := map[value.Value]value.Value{} // array -> the first value pushed into it
	closureFn := map[value.Value]string{}      // js_closure call -> IR function name jsf_N
	// jsf_N -> the statement that creates its closure: the js_srcpos marker right
	// before the js_closure in the creating function. That statement is the
	// definition (`def f`, `function f`), so its line names the function rather
	// than the first statement of its body. Not when another closure came between
	// them, nor for a closure stored into an object member - a class statement
	// creates all its methods after ONE marker, its own - unless the marker is
	// ADJACENT to the closure (nothing but string constants between): a
	// declaration's own marker (defPos in lib/compile-core.js).
	type cgSite struct {
		in  *ir.Func
		pos int
		adj bool
	}
	closureSite := map[string]cgSite{}

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

	// The module entry scaffolding is not program structure: its edges come from
	// "(top)" and it never registers as a definition. Several entries end by
	// running the program as a closure through js_try / js_gotry (entryRun): the
	// last such call of a block that returns. A top-level try statement of the
	// program is never that call - its block goes on to dispatch the try's
	// control signal.
	handleIR := false
	entryThunk := map[string]bool{}
	definedFn := map[string]bool{}
	hasMarker := map[string]bool{} // the function emits a statement marker
	entryRun := map[value.Value]bool{}
	for _, f := range m.Funcs {
		if len(f.Blocks) > 0 {
			definedFn[f.Name()] = true
			handleIR = handleIR || cgUserFunc(f.Name()) || cgEntryFuncs[f.Name()]
		}
		if cgEntryFuncs[f.Name()] {
			for _, b := range f.Blocks {
				if _, ok := b.Term.(*ir.TermRet); !ok {
					continue
				}
				for i := len(b.Insts) - 1; i >= 0; i-- {
					if call, ok := b.Insts[i].(*ir.InstCall); ok {
						if n := call.Callee.Ident(); n == "@js_try" || n == "@js_gotry" {
							entryRun[call] = true
							break
						}
					}
				}
			}
		}
	}
	forEachCall(func(f *ir.Func, call *ir.InstCall, callee string) {
		hasMarker[f.Name()] = hasMarker[f.Name()] || callee == "js_srcpos"
	})
	scaffold := func(n string) bool { return cgEntryFuncs[n] || entryThunk[n] }
	fnName := map[string]string{} // jsf_N -> its source name
	// A try statement runs its body, handler and finally block as closures
	// through js_try, and go runs every function body as a closure through
	// js_gotry (its frame holds the defer list): those closures are part of the
	// function that creates them, which bodyOf maps them to.
	bodyOf := map[string]string{}
	// An unnamed closure the entry function creates before its first statement
	// marker is a runtime hook (js_rt_deep's overflow reporter, python's uncaught
	// handler) or a prelude initialiser: runtime fabric, not program code.
	hook := map[string]bool{}

	// Pass 1: string literals and closure creations (values compare by pointer).
	var posFn *ir.Func
	lastPos := -1
	adj := false
	marked := false // posFn has emitted a statement marker
	forEachCall(func(f *ir.Func, call *ir.InstCall, callee string) {
		if f != posFn {
			posFn, lastPos, adj, marked = f, -1, false, false
		}
		switch callee {
		case "js_str_mem":
			if s, ok := decodeStrMem(call); ok {
				strOf[call] = s
			}
			return
		case "js_srcpos":
			if len(call.Args) > 0 {
				if ci, ok := call.Args[0].(*constant.Int); ok {
					lastPos = int(ci.X.Int64())
					adj = true
					marked = true
					return
				}
			}
		case "js_closure":
			if len(call.Args) > 0 {
				if ci, ok := call.Args[0].(*constant.Int); ok {
					fn := fmt.Sprintf("jsf_%d", ci.X.Int64())
					closureFn[call] = fn
					if _, seen := closureSite[fn]; !seen && lastPos >= 0 {
						closureSite[fn] = cgSite{f, lastPos, adj}
					}
					if cgEntryFuncs[f.Name()] && !marked {
						hook[fn] = true
					}
					lastPos = -1
				}
			}
		case "js_jsfnmeta", "js_pysig", "js_pyfnname", "js_rarity":
			// js/ts/php wrap a closure to give it `name`/`length`, python its
			// signature and `__name__`, ruby its arity shape, and store the
			// wrapper's result, which is the same function.
			if len(call.Args) > 0 {
				if fn, ok := closureFn[call.Args[0]]; ok {
					closureFn[call] = fn
				}
			}
			return
		case "js_try", "js_gotry":
			// The body the entry runs is the module top level (python and swift
			// run the whole program inside one js_try); it is named as python's
			// traceback names it. A body without statements only calls main
			// (java, kotlin, C#, go): scaffolding. The entry's handler reports an
			// uncaught exception: a runtime hook.
			for i, a := range call.Args {
				fn, ok := closureFn[a]
				switch {
				case !ok:
				case !entryRun[call]:
					bodyOf[fn] = f.Name()
				case i == 0 && hasMarker[fn]:
					fnName[fn] = "<module>"
				case i == 0:
					entryThunk[fn] = true
				default:
					hook[fn] = true
				}
			}
		}
		adj = false
	})

	// Pass 2: name reads, and the source names of the IR functions (a closure
	// gets its name where it is stored: a declaration, an assignment, or the
	// method slot of a class descriptor).
	forEachCall(func(f *ir.Func, call *ir.InstCall, callee string) {
		if cgNameReaders[callee] {
			callee = "js_scope_get"
		}
		switch callee {
		case "js_scope_get", "js_kget", "js_ktget":
			if len(call.Args) >= 2 {
				if s, ok := strOf[call.Args[1]]; ok {
					nameRead[call] = s
					scopeRead[call] = true
				}
			}
		case "js_arr_push":
			if len(call.Args) >= 2 {
				if _, seen := firstPush[call.Args[0]]; !seen {
					firstPush[call.Args[0]] = call.Args[1]
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
		case cgPassThrough:
			if len(call.Args) >= 1 {
				if s, ok := nameRead[call.Args[0]]; ok {
					nameRead[call] = s
					scopeRead[call] = scopeRead[call.Args[0]]
				}
			}
		case "js_tdecl", "js_scope_decl", "js_pyset_var", "py_setvar", "sw_kset", "dt_kset", "kt_kset",
			"js_tset", "js_scope_set", "js_kset", "js_set", "js_phmset":
			if len(call.Args) >= 3 {
				if fn, ok := closureFn[call.Args[2]]; ok {
					member := callee == "js_set" || callee == "js_phmset"
					if member && !closureSite[fn].adj {
						delete(closureSite, fn)
					}
					if s, ok2 := strOf[call.Args[1]]; ok2 {
						if _, exists := fnName[fn]; !exists {
							// A ruby top-level def is stored only under "$d$" + its
							// name; the plain slot belongs to a local of that name.
							fnName[fn] = strings.TrimPrefix(s, "$d$")
						}
						if !member {
							boundTo[s] = append(boundTo[s], fn)
						}
					}
				}
			}
		}
	})
	// root is the function a closure body belongs to (itself, if none).
	root := func(n string) string {
		for {
			frame, ok := bodyOf[n]
			if !ok {
				return n
			}
			n = frame
		}
	}
	display := func(irName string) string {
		irName = root(irName)
		if s, ok := fnName[irName]; ok {
			return s
		}
		return irName
	}
	// Runtime fabric: neither a definition, nor a caller, nor a callee.
	fabric := func(n string) bool {
		n = root(n)
		if !handleIR || scaffold(n) {
			return false
		}
		if !cgUserFunc(n) {
			return true
		}
		_, named := fnName[n]
		return hook[n] && !named
	}
	// A function also bound under a compiler-made second name (a swift overload
	// as "f$1" beside "f", a ruby method as "$d$f") is called through that name:
	// a read of it resolves to the function's source name, unless the name is
	// itself some function's source name or holds functions of different names.
	ownName := map[string]bool{}
	for _, s := range fnName {
		ownName[s] = true
	}
	alias := map[string]string{}
	for s, fns := range boundTo {
		if ownName[s] {
			continue
		}
		d := display(fns[0])
		same := true
		for _, fn := range fns[1:] {
			same = same && display(fn) == d
		}
		if same {
			alias[s] = d
		}
	}
	callTarget := func(v value.Value) (string, bool) {
		if s, ok := nameRead[v]; ok {
			if a, ok := alias[s]; ok && scopeRead[v] {
				return a, true
			}
			return s, true
		}
		if fn, ok := closureFn[v]; ok {
			return display(fn), true
		}
		return "", false
	}

	// Pass 3: definitions and edges.
	// Stamp any functions still unattributed (the main file's, compiled after
	// the last import popped) with the final source, so every function has one.
	attributeModuleFuncs()
	// Each function's own file + line-start table (falls back to the global
	// source for the single-file case, where compileFuncFile mirrors it anyway).
	srcOf := func(f *ir.Func) (string, []int) {
		if fs, ok := compileFuncFile[f.Name()]; ok {
			return fs.file, fs.starts
		}
		return traceSrcName, traceLineStarts
	}

	var defs []cgDef
	var calls []cgCall
	defAt := map[string]int{} // IR function -> its index in defs
	bodyLine := map[string]int{}
	// A call through a name the same function first tests with typeof (python's
	// `if main is defined: main()` entry) happens only when something defines
	// that name: conditional[i] marks such a call in calls.
	conditional := map[int]bool{}
	unattributed := 0
	for _, f := range m.Funcs {
		if len(f.Blocks) == 0 || cgHelper(f.Name()) || fabric(f.Name()) {
			continue // Extern declaration or runtime fabric, not a definition.
		}
		from := display(f.Name())
		typeofTested := map[string]bool{}
		if scaffold(root(f.Name())) {
			from = "(top)"
		}
		file, starts := srcOf(f)
		if file == "" && !scaffold(root(f.Name())) {
			unattributed++
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
							if l := lineOfPosIn(int(ci.X.Int64()), starts); l > 0 && (line == 0 || l < line) {
								line = l
							}
						}
					}
				case callee == "js_scope_typeof":
					if len(call.Args) >= 2 {
						if s, ok := strOf[call.Args[1]]; ok {
							typeofTested[s] = true
						}
					}
				case callee == "js_call" || cgValueCallers[callee] || callee == cgTailCaller:
					to := "(dynamic)"
					if len(call.Args) > 0 {
						fv := call.Args[0]
						if callee == cgTailCaller {
							fv = firstPush[fv]
						}
						if fn, ok := closureFn[fv]; ok && fabric(fn) {
							break // the entry running a prelude initialiser
						}
						if s, ok := callTarget(fv); ok {
							to = s
						}
						if s, ok := nameRead[fv]; ok && scopeRead[fv] && typeofTested[s] {
							conditional[len(calls)] = true
						}
					}
					calls = append(calls, cgCall{from, to, file})
				case cgNameCallers[callee]:
					to := "(dynamic)"
					if len(call.Args) >= 2 {
						if s, ok := strOf[call.Args[1]]; ok {
							to = s
							if a, ok := alias[s]; ok {
								to = a
							}
						}
					}
					calls = append(calls, cgCall{from, to, file})
				case cgHelper(callee):
					// Runtime fabric, like the js_ externs.
				case cgMethodCall(callee):
					to := "(dynamic)"
					if len(call.Args) >= 2 {
						if s, ok := strOf[call.Args[1]]; ok {
							to = s
						}
					}
					calls = append(calls, cgCall{from, to, file})
				case callee == "js_supercall" || callee == "js_jvsuper":
					to := "(dynamic)"
					if len(call.Args) >= 3 {
						if s, ok := strOf[call.Args[2]]; ok {
							to = s
						}
					}
					calls = append(calls, cgCall{from, to, file})
				case strings.HasPrefix(callee, "js_"):
					// Runtime fabric, not program structure.
				case scaffold(callee) || definedFn[callee] && fabric(callee):
					// jsmain calling jsrun, or a runtime helper: not an edge.
				default:
					// A direct call (the integer-IR languages, putchar, ...).
					calls = append(calls, cgCall{from, callee, file})
				}
			}
		}
		// A marker is where its statement begins (up.start, or the first token
		// in python, ruby and php), so the defining statement's line precedes
		// the body's; it is used only when it does not follow the body's.
		if site, ok := closureSite[f.Name()]; ok {
			_, siteStarts := srcOf(site.in)
			if l := lineOfPosIn(site.pos, siteStarts); l > 0 && (line == 0 || l <= line) {
				line = l
			}
		}
		if frame := root(f.Name()); frame != f.Name() {
			// Part of the creating function's definition, not one of its own.
			if l, seen := bodyLine[frame]; line > 0 && (!seen || line < l) {
				bodyLine[frame] = line
			}
		} else if !scaffold(f.Name()) {
			defAt[f.Name()] = len(defs)
			defs = append(defs, cgDef{Name: from, File: file, Line: line})
		}
	}
	for frame, l := range bodyLine {
		if i, ok := defAt[frame]; ok && (defs[i].Line == 0 || l < defs[i].Line) {
			defs[i].Line = l
		}
	}
	if len(conditional) > 0 {
		defName := map[string]bool{}
		for _, d := range defs {
			defName[d.Name] = true
		}
		kept := calls[:0]
		for i, c := range calls {
			if !conditional[i] || defName[c.To] {
				kept = append(kept, c)
			}
		}
		calls = kept
	}
	if unattributed > 0 {
		fmt.Fprintf(os.Stderr, "callgraph: %d function(s) had no source file (attribution incomplete)\n", unattributed)
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
