package abnf

import (
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"14.gy/mec/abnf/r"
	"github.com/dop251/goja"
	"github.com/llir/llvm/ir/constant"
)

// ----------------------------------------------------------------------------
// Scripting subsystem code for parser and compiler

// One cached, precompiled JS program together with the source it was compiled from.
// The source is kept to detect UID collisions: Two different a-grammars that were numbered
// independently can carry the same tag UIDs for different code.
type cachedProgram struct {
	src string
	p   *goja.Program
}

type commonscript struct {
	vm              *goja.Runtime
	codeCache       []cachedProgram              // Compiled programs by tag UID (for tags and :script() commands with a UID > 0).
	codeCacheBySrc  map[string]*goja.Program     // Compiled programs by name plus source text (for everything without a UID).
	referencesCache *references                  // Keeps the tag UIDs stable over multiple correctReferencesAndIDs() calls.
	export          func(goja.Value) interface{} // Export for a value stored on the Go side (gojaExporter).
}

// UnescapeTilde resolves the only escape sequence of a tags raw Code string: It replaces
// every \~ with a ~ (tilde) and leaves everything else untouched.
func UnescapeTilde(s string) string {
	// Is it trivial? Avoid allocation.
	if !strings.ContainsRune(s, '\\') {
		if utf8.ValidString(s) {
			return s
		}
	}

	buf := make([]byte, 0, 3*len(s)/2) // Try to avoid more allocations.
	for pos := 0; pos+1 < len(s); pos++ {
		if s[pos] == '\\' && s[pos+1] == '~' {
			buf = append(buf, s[:pos]...)
			s = s[pos+1:]
			pos = 0
		}
	}
	buf = append(buf, s...)
	return string(buf)
}

// floPrecStr is the host global floPrec(v, n): the correctly rounded n
// SIGNIFICANT DIGIT decimal of |v|, written DIGITS "e" EXP - the digits with no
// point and the trailing zeros stripped, and EXP the decimal exponent of the
// first digit. It is the digit string of C's "%.*e".
//
// WHY IT IS A HOST GLOBAL AND NOT SCRIPT CODE. The dialect has no toPrecision and
// no toExponential: a script can only ask for the host's SHORTEST round-tripping
// form ("" + n). A %g formatter needs the opposite question - the value rounded
// to a FIXED number of digits - and Lua's tostring is exactly that caller: it
// tries "%.15g" and, when that does not read back, "%.17g", so 1e15 prints
// "1e+15" while 1e15+1 prints "1000000000000001.0". Without this the script
// halves could only guess, and a guess is a --cross divergence against the Go
// twin, which has strconv.
//
// The twins are host id 70 of languages/lib/runtime.c (over shortest_digits with
// g_mindig = n) and the floPrec binding of languages/metajs-interpreter.abnf
// (which is this function, reached through the host globals). n is clamped to
// [1, 17] in every engine: seventeen digits always round-trip a binary64, so
// nothing above it can be asked for meaningfully.
func floPrecStr(v float64, n int) string {
	if math.IsNaN(v) || math.IsInf(v, 0) || v == 0 {
		return "0e0"
	}
	if n < 1 {
		n = 1
	}
	if n > 17 {
		n = 17
	}
	s := strconv.FormatFloat(math.Abs(v), 'e', n-1, 64)
	i := strings.IndexByte(s, 'e')
	mant := strings.Replace(s[:i], ".", "", 1)
	exp, _ := strconv.Atoi(s[i+1:])
	for len(mant) > 1 && mant[len(mant)-1] == '0' {
		mant = mant[:len(mant)-1]
	}
	return mant + "e" + strconv.Itoa(exp)
}

// scriptName is the name of one script module: the file (or grammar module) it
// belongs to, plus the source position of the node it was written on. The name
// is only ever needed when a script is COMPILED for the first time (it is baked
// into the program for stack traces and for the relative path resolution of
// load()/store()/include()) or when an error is reported - but the compile walk
// runs a tag script many thousand times per grammar, so the walk passes the parts
// and the engines format on demand.
type scriptName struct {
	name string // The module/file name; the whole name when pos < 0.
	kind string // What the position names, e.g. ":tag:pos:".
	pos  int    // Source position of the node, or < 0 for a name without one.
}

// nodeScript names the script of an ASG node at a source position.
func nodeScript(name, kind string, pos int) scriptName {
	return scriptName{name: name, kind: kind, pos: pos}
}

// fileScript names a script that is already fully named (a start script, an
// include, an eval).
func fileScript(name string) scriptName {
	return scriptName{name: name, pos: -1}
}

func (n scriptName) String() string {
	if n.pos < 0 {
		return n.name
	}
	return n.name + n.kind + strconv.Itoa(n.pos)
}

// Run executes the given source string in the global context.
// Compiled programs are cached: Code with a UID (ID > 0, assigned by correctReferencesAndIDs())
// is cached per UID. The comparison with the cached source is cheap (usually only a pointer
// comparison) and protects against UID collisions between independently numbered a-grammars.
// All other code (ID <= 0, e.g. the start script, includes, or tags that were built via JS and
// never got a UID) is cached by its name plus source text. The name is part of the key because
// it is compiled into the program (for stack traces and relative paths), so byte-identical
// code from two different files must not share one program.
func (cs *commonscript) Run(name scriptName, src string, ID int) (goja.Value, error) {
	var p *goja.Program
	var key string
	if ID > 0 {
		if ID >= len(cs.codeCache) {
			tmp := make([]cachedProgram, ID*2)
			cs.codeCache = append(cs.codeCache, tmp...)
		} else if cs.codeCache[ID].src == src {
			p = cs.codeCache[ID].p
		}
	} else {
		key = name.String() + "\x00" + src
		p = cs.codeCacheBySrc[key]
	}

	// Compile and cache on the first run.
	if p == nil {
		var err error
		p, err = goja.Compile(name.String(), src, true)
		if err != nil {
			return nil, err
		}
		if ID > 0 {
			cs.codeCache[ID] = cachedProgram{src: src, p: p}
		} else {
			cs.codeCacheBySrc[key] = p
		}
	}

	return cs.vm.RunProgram(p)
}

// getCurrentModuleFileName returns the source name of the JS code that is currently being
// executed (e.g. "tests/foo.abnf:startScript"). File operations like load(), store() and
// include() use it to resolve their paths relative to that file.
func (cs *commonscript) getCurrentModuleFileName() string {
	var buf [2]goja.StackFrame
	frames := cs.vm.CaptureCallStack(2, buf[:0])
	if len(frames) < 2 {
		return "."
	}
	return frames[1].SrcName()
}

// NewCommonScript installs everything into the JS VM that parser and compiler scripts have
// in common: console output, file access, string helpers, and the 'c', 'abnf' and 'llvm'
// objects. Note that *compilerFuncMap is replaced with a fresh map; the caller can add its
// own entries afterwards.
func NewCommonScript(vm *goja.Runtime, compilerFuncMap *map[string]r.Object, preventDefaultOutput bool) *commonscript {
	var common commonscript

	common.vm = vm
	common.codeCache = make([]cachedProgram, 100)
	common.codeCacheBySrc = map[string]*goja.Program{}
	common.export = gojaExporter(vm)

	if preventDefaultOutput { // Script output disabled.
		vm.Set("print", func(a ...interface{}) (n int, err error) { return 0, nil })
		vm.Set("println", func(a ...interface{}) (n int, err error) { return 0, nil })
		vm.Set("printf", func(format string, a ...interface{}) (n int, err error) { return 0, nil })
		vm.Set("writeBytes", func(v goja.Value) {})
	} else { // Script output enabled (routed through outWriter so -pipe can capture it).
		vm.Set("print", func(a ...interface{}) (int, error) { return fmt.Fprint(outWriter, a...) })
		vm.Set("println", func(a ...interface{}) (int, error) { return fmt.Fprintln(outWriter, a...) })
		vm.Set("printf", func(format string, a ...interface{}) (int, error) { return fmt.Fprintf(outWriter, format, a...) })
		// writeBytes: the frozen host's binding of the same name (standardJSBindings).
		wtf8 := gojaWTF8(vm)
		vm.Set("writeBytes", func(v goja.Value) { outWriter.Write(lowBytes(wtf8(v))) })
	}

	// eprintln is the DIAGNOSTIC channel of the tag scripts (warnings). It goes
	// straight to standard error - never through outWriter, which -pipe swaps to
	// capture a producer stage's text, and never into the emitted module. It is
	// bound outside the quiet branch above on purpose: -q/-qq are about module and
	// program output, a warning is a diagnostic. The frozen host binds the same
	// name identically (standardJSBindings in jsrt.go); the two must agree or the
	// matrix goes red.
	vm.Set("eprintln", func(a ...interface{}) (int, error) { return fmt.Fprintln(warnWriter, a...) })

	vm.Set("sprintf", fmt.Sprintf) // Sprintf is no output.
	vm.Set("exit", os.Exit)

	vm.Set("sleep", func(d time.Duration) { time.Sleep(d * time.Millisecond) })
	// clockMicros() is the wall clock in microseconds since the epoch, exact in a
	// double; frozen.go binds the same name.
	vm.Set("clockMicros", func() float64 { return float64(time.Now().UnixMicro()) })

	vm.Set("append", func(t []interface{}, v ...interface{}) interface{} {
		tmp := append(t, v...)
		return &tmp
	})

	vm.Set("unescape", r.Unescape)
	vm.Set("unescapeTilde", UnescapeTilde)

	// floPrec: fixed-precision decimal digits. See floPrecStr below - the frozen
	// grammar host (frozenBaseBindings, frozen.go) and the MetaJS program host
	// (programJSBindings, jsrtint.go) bind the same name to the same function, and
	// languages/lib/runtime.c is host id 70. The three must agree or the matrix
	// goes red.
	vm.Set("floPrec", func(v float64, n int) string { return floPrecStr(v, n) })

	installGojaFnMeta(vm)
	installGojaArrKind(vm)
	installGojaCoroutines(vm)
	// osQuery(kind, arg): the machine's clock, time zone and environment; the
	// frozen and program hosts' osBindings (jsrtint.go).
	vm.Set("osQuery", func(call goja.FunctionCall) goja.Value {
		kind := int(call.Argument(0).ToInteger())
		if kind == 5 {
			if v, ok := os.LookupEnv(call.Argument(1).String()); ok {
				return vm.ToValue(v)
			}
			return goja.Undefined()
		}
		return vm.ToValue(osQuery(kind, call.Argument(1).ToFloat()))
	})
	// readIn(kind): standard input (readIn in jsrtint.go).
	vm.Set("readIn", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(readIn(int(call.Argument(0).ToInteger())))
	})

	// The WTF-8 byte length of a string. JS .length counts UTF-16 code units,
	// but the emitters need the byte count of the char arrays they emit
	// (lib/compile-core.js emitStr).
	wtf8 := gojaWTF8(vm)
	vm.Set("byteLen", func(v goja.Value) int { return len(wtf8(v)) })
	// byteAt / byteSub / bytesStr: byteBindings in jsrtint.go over the same WTF-8
	// bytes. A goja string cannot hold a byte that is not part of a character, so
	// a result that cuts one reads U+FFFD there.
	vm.Set("byteAt", func(v goja.Value, i goja.Value) int {
		b, k := wtf8(v), jsToInt(i.ToFloat())
		if k < 0 || k >= len(b) {
			return -1
		}
		return int(b[k])
	})
	vm.Set("byteSub", func(v, lo, hi goja.Value) string {
		b := wtf8(v)
		l, h := byteWindow(len(b), jsToInt(lo.ToFloat()), jsToInt(hi.ToFloat()))
		return b[l:h]
	})
	vm.Set("bytesStr", func(v goja.Value) string {
		var out []byte
		if o, ok := v.(*goja.Object); ok && o.ClassName() == "Array" {
			for _, e := range o.Export().([]interface{}) {
				out = append(out, byte(int64(jsToInt(vm.ToValue(e).ToFloat()))))
			}
		}
		return string(out)
	})

	// sprint: fmt.Sprint's text, the rendering println gives returned rather than
	// printed. Twin of the frozen host's binding (standardJSBindings, jsrt.go);
	// metajs-interpreter.abnf's hostGlobals needs it because the C floor seeds
	// sprint as host id 35.
	vm.Set("sprint", fmt.Sprint)

	// rawSet writes an OWN property, bypassing any Object.prototype accessor: a
	// plain obj[name] = v with name "__proto__" invokes the prototype SETTER
	// under a host JS engine (a TypeError for primitive values, a silent
	// reparenting for objects). The frozen engine's objects have no prototype
	// chain; the interpreter cores use rawSet for that name to keep the two
	// engines aligned.
	vm.Set("rawSet", func(obj *goja.Object, name string, v goja.Value) {
		if err := obj.DefineDataProperty(name, v, goja.FLAG_TRUE, goja.FLAG_TRUE, goja.FLAG_TRUE); err != nil {
			panic(err)
		}
	})

	// The MetaJS 'anytype' declaration marker (var v = anytype). Goja cannot pin
	// types, so here it is plain undefined: the variable starts as undefined
	// exactly like under the enforcing engines, and the name always resolves.
	vm.Set("anytype", goja.Undefined())

	vm.Set("include", func(fileName string) bool {
		if fileName == "" {
			return false
		}
		includeFileName := filepath.Dir(common.getCurrentModuleFileName()) + string(os.PathSeparator) + filepath.Clean(fileName)
		dat, err := ioutil.ReadFile(includeFileName)
		if err != nil {
			panic(err)
		}
		srcCode := StripBOM(string(dat))

		_, err = common.Run(fileScript(includeFileName), srcCode, -1)
		if err != nil {
			panic(err.Error() + "\nError was in " + includeFileName)
		}

		return true
	})

	vm.Set("moduleName", common.getCurrentModuleFileName)

	vm.Set("load", func(fileName string) string {
		loadFileName := filepath.Dir(common.getCurrentModuleFileName()) + string(os.PathSeparator) + filepath.Clean(fileName)
		dat, err := ioutil.ReadFile(loadFileName)
		if err != nil {
			panic(err)
		}
		return string(dat)
	})
	vm.Set("store", func(fileName, data string) {
		storeFileName := filepath.Dir(common.getCurrentModuleFileName()) + string(os.PathSeparator) + filepath.Clean(fileName)
		err := ioutil.WriteFile(storeFileName, []byte(data), 0644)
		if err != nil {
			panic(err)
		}
	})

	// correctReferencesAndIDs is a global (not part of abnf.*): it links a freshly
	// built a-grammar so it can be compiled directly - resolving each Identifier to its
	// Production and giving each Tag a UID. See references.correctReferencesAndIDs.
	vm.Set("correctReferencesAndIDs", func(agrammar *r.Rules) {
		// The references cache lives as long as this VM: Tag UIDs have to stay stable and
		// unique over multiple calls, otherwise the compiled-code cache would mix up the
		// tags of different a-grammars.
		if common.referencesCache == nil {
			common.referencesCache = NewReferences()
		}
		common.referencesCache.correctReferencesAndIDs(agrammar)
	})

	// vm.Set("writable", func(v interface{}) *interface{} {
	// 	return &v
	// })
	// vm.Set("nonwritable", func(v *interface{}) interface{} {
	// 	return *v
	// })

	*compilerFuncMap = map[string]r.Object{
		// The optional fileName names the source in parse errors (and fixes relative
		// paths); an imported file passes its own path so an error points at it, not
		// at the grammar. Empty/undefined falls back to the executing module's name.
		"parse": func(agrammar *r.Rules, srcCode string, options *Parseropts, fileName string) *r.Rules {
			if options == nil {
				options = &Parseropts{}
			}
			if fileName == "" || fileName == "undefined" {
				fileName = common.getCurrentModuleFileName()
			}
			productions, err := ParseWithAgrammar(agrammar, srcCode, fileName, options)
			if err != nil {
				panic(err)
			}
			return productions
		},
		// parseFrom is parse with an explicit start production: it parses srcCode from the
		// named rule (e.g. "Statement") rather than the grammar's declared :startRule().
		// Used by the -main snippet form to parse a code fragment of the target language.
		"parseFrom": func(agrammar *r.Rules, srcCode string, startRule string) *r.Rules {
			productions, err := ParseWithAgrammar(agrammar, srcCode, common.getCurrentModuleFileName(), &Parseropts{StartRule: startRule})
			if err != nil {
				panic(err)
			}
			return productions
		},
		"compileRunStartScript": func(asg *r.Rules, aGrammar *r.Rules, slot int, traceEnabled bool) interface{} {
			return compileASGInternal(asg, aGrammar, common.getCurrentModuleFileName(), slot, traceEnabled, preventDefaultOutput)
		},
		"ABNFagrammar": AbnfAgrammar,
		// True when -trace/-cfgraph/-callgraph/-why collect source positions: the
		// compilers then emit js_srcpos statement markers (see lib/compile-core.js
		// stmtPos, and TraceMarkersWanted for the four flags that ask for them).
		"tracing": TraceMarkersWanted(),
		// Import policy + source positions for clean grammar errors. warnImports
		// is the -warn-imports flag; file is the program being compiled; lineOf
		// turns an up.pos byte offset into a 1-based line (0 if unknown).
		"warnImports":     WarnUnresolvedImports,
		"warnUnsupported": WarnUnsupported,
		"rtPrims":         RuntimePrims,
		"rtLib":           RuntimeLib,
		"file":            traceSrcName,
		"lineOf":          func(pos int) int { return lineOfPos(pos) },
		"srcSlice":        func(from, to int) string { return srcSlice(from, to) },
		// The entry-point function name (-main flag, default "main").
		"mainName": EntryPoint,
		// Output path for a native executable (-exe flag); "" means run in the IR
		// interpreter instead. A -to-llvm-ir grammar hands its module to
		// llvm.BuildExecutable when this is set.
		"exePath": ExePath,
		// The runtime a native build links with the module (-rt, repeatable). A
		// grammar hands it to llvm.BuildExecutable; supplying anything here also
		// turns an unresolved symbol from a zero stub into a hard error.
		"runtime": RuntimeInputs,
		// Project-file imports (the -i include roots): findImport locates a
		// grammar-mapped relative path ("a/b/C.kt"), readFile loads it, and
		// pushSource/popSource swap the file/line attribution around the
		// imported file's nested c.parse + c.compile walk. curFile is the
		// dynamic variant of "file" (which is snapshotted at map creation), and
		// curFileAbs its absolute path with symlinks resolved.
		"curFile":        func() string { return traceSrcName },
		"curFileAbs":     func() string { return curSrcAbs() },
		"findImport":     func(relPath string) string { return findImportFile(relPath) },
		"readFile":       func(path string) string { return readImportFile(path) },
		"pushSource":     func(name, text string) { pushTraceSource(name, text) },
		"pushSourceFile": func(name string) { pushTraceSourceFile(name) },
		"popSource":      func() { popTraceSource() },
		// realpath(3) for a program the interpreter runs (PHP's realpath); "" when
		// the path does not resolve. Twin of js_realpath (abnf/jsrtphp.go).
		"realpath": func(p string) string { return realPath(p) },
	}
	// The three host API objects are wrapped for goja ONCE per member instead of
	// per access (see hostAPIObject). c.localAsg is the only entry a run rebinds
	// (per tag / per :script()), so it is the only one read through every time.
	vm.Set("c", newHostAPIObject(vm, compilerFuncMap, "localAsg"))
	vm.Set("abnf", newHostAPIObject(vm, r.AbnfFuncMap))
	vm.Set("llvm", newHostAPIObject(vm, gojaLLVMMap(wtf8)))

	installGojaCaseMapping(vm)
	installGojaCharAt(vm)
	installGojaMathSurface(vm)

	return &common
}

// gojaWTF8 converts a goja string to the Go string the frozen engine holds for
// it: WTF-8, so a lone surrogate keeps its own three bytes (see strUnits in
// abnf/jsrt.go). goja's own conversion replaces one by U+FFFD, so only a string
// containing U+FFFD after that conversion is re-read unit by unit.
func gojaWTF8(vm *goja.Runtime) func(goja.Value) string {
	v, err := vm.RunString(`(function (s) {
		var u = [], lone = false;
		for (var i = 0; i < s.length; i++) {
			var c = s.charCodeAt(i);
			if (c >= 0xD800 && c <= 0xDBFF && i + 1 < s.length) {
				var d = s.charCodeAt(i + 1);
				if (d >= 0xDC00 && d <= 0xDFFF) { u.push(c, d); i++; continue; }
			}
			if (c >= 0xD800 && c <= 0xDFFF) lone = true;
			u.push(c);
		}
		return lone ? u : null;
	})`)
	if err != nil {
		panic(err)
	}
	units, _ := goja.AssertFunction(v)
	return func(s goja.Value) string {
		text := s.String()
		if !strings.ContainsRune(text, utf8.RuneError) {
			return text
		}
		r, err := units(goja.Undefined(), s)
		if err != nil || goja.IsNull(r) {
			return text
		}
		codes := r.Export().([]interface{})
		u := make([]uint16, len(codes))
		for i, c := range codes {
			u[i] = uint16(c.(int64))
		}
		return strFromUnits(u)
	}
}

// gojaExporter returns goja's Export for a value that is STORED on the Go side
// (the tag stacks, ltr) and read back by a script later. Export turns a string
// holding a lone surrogate into U+FFFD, so such a string is kept as the goja
// value itself, which ToValue hands back unchanged; every other part of the
// export is goja's own. Only an export that contains U+FFFD can have lost a
// unit, so anything else costs one walk over the exported value.
func gojaExporter(vm *goja.Runtime) func(goja.Value) interface{} {
	v, err := vm.RunString(`(function (s) {
		for (var i = 0; i < s.length; i++) {
			var c = s.charCodeAt(i);
			if (c < 0xD800 || c > 0xDFFF) continue;
			if (c <= 0xDBFF && i + 1 < s.length) {
				var d = s.charCodeAt(i + 1);
				if (d >= 0xDC00 && d <= 0xDFFF) { i++; continue; }
			}
			return true;
		}
		return false;
	})`)
	if err != nil {
		panic(err)
	}
	loneFn, _ := goja.AssertFunction(v)
	lone := func(s goja.Value) bool {
		r, err := loneFn(goja.Undefined(), s)
		return err == nil && r.ToBoolean()
	}
	// An export is a graph (goja exports a shared or cyclic object once). A walk
	// first runs without bookkeeping on a budget of visits; past it, the walk
	// restarts and visits each map and slice once.
	type walk struct {
		budget int
		seen   map[uintptr]bool
	}
	first := func(w *walk, x interface{}) bool {
		if w.seen == nil {
			w.budget--
			return w.budget >= 0
		}
		p := reflect.ValueOf(x).Pointer()
		if p == 0 || w.seen[p] {
			return false
		}
		w.seen[p] = true
		return true
	}
	var lost func(w *walk, e interface{}) bool
	lost = func(w *walk, e interface{}) bool {
		switch x := e.(type) {
		case string:
			return strings.Contains(x, "\uFFFD")
		case map[string]interface{}:
			if first(w, x) {
				for _, ev := range x {
					if lost(w, ev) {
						return true
					}
				}
			}
		case []interface{}:
			if len(x) > 0 && first(w, x) {
				for _, ev := range x {
					if lost(w, ev) {
						return true
					}
				}
			}
		}
		return false
	}
	// restore puts the goja string back wherever the export lost a unit,
	// reading the original value at the same key or index.
	var restore func(w *walk, e interface{}, v goja.Value) interface{}
	restore = func(w *walk, e interface{}, v goja.Value) interface{} {
		switch x := e.(type) {
		case string:
			if strings.Contains(x, "\uFFFD") && lone(v) {
				return v
			}
		case map[string]interface{}:
			if o, ok := v.(*goja.Object); ok && first(w, x) {
				for k, ev := range x {
					x[k] = restore(w, ev, o.Get(k))
				}
			}
		case []interface{}:
			if o, ok := v.(*goja.Object); ok && len(x) > 0 && first(w, x) {
				for i, ev := range x {
					x[i] = restore(w, ev, o.Get(strconv.Itoa(i)))
				}
			}
		}
		return e
	}
	return func(v goja.Value) interface{} {
		if v == nil {
			return nil
		}
		e := v.Export()
		w := walk{budget: 256}
		found := lost(&w, e)
		if !found && w.budget < 0 {
			found = lost(&walk{seen: map[uintptr]bool{}}, e)
		}
		if found {
			e = restore(&walk{seen: map[uintptr]bool{}}, e, v)
		}
		return e
	}
}

// gojaLLVMMap is llvmFuncMap with the string constant taking its bytes through
// gojaWTF8, so goja emits the same char array as the frozen engine.
func gojaLLVMMap(wtf8 func(goja.Value) string) map[string]r.Object {
	m := make(map[string]r.Object, len(llvmFuncMap))
	for k, v := range llvmFuncMap {
		m[k] = v
	}
	cm := map[string]r.Object{}
	for k, v := range llvmFuncMap["constant"].(map[string]r.Object) {
		cm[k] = v
	}
	cm["NewCharArrayFromString"] = func(s goja.Value) *constant.CharArray {
		return constant.NewCharArrayFromString(wtf8(s))
	}
	m["constant"] = cm
	return m
}

// installGojaFnMeta binds fnMeta(f, key) and fnMetaSet(f, name, length), the goja
// twin of fnMetaBindings (jsrtint.go), which the frozen grammar host binds. The pair
// is kept ON the function object under a private symbol, so it is O(1), invisible to
// every string-keyed enumeration, and does not keep the function alive.
func installGojaFnMeta(vm *goja.Runtime) {
	sym := goja.NewSymbol("fnMeta")
	fnObj := func(v goja.Value) *goja.Object {
		o, ok := v.(*goja.Object)
		if !ok {
			return nil
		}
		if _, isFn := goja.AssertFunction(o); !isFn {
			return nil
		}
		return o
	}
	vm.Set("fnMeta", func(call goja.FunctionCall) goja.Value {
		o := fnObj(call.Argument(0))
		if o == nil {
			return goja.Undefined()
		}
		mi := jsvFnInfo{}
		if v := o.GetSymbol(sym); v != nil {
			if p, ok := v.Export().(*jsvFnInfo); ok {
				mi = *p
			}
		}
		switch call.Argument(1).String() {
		case "name":
			return vm.ToValue(mi.name)
		case "length":
			return vm.ToValue(mi.length)
		case "src":
			return vm.ToValue(mi.src)
		}
		return goja.Undefined()
	})
	vm.Set("fnMetaSet", func(call goja.FunctionCall) goja.Value {
		f := call.Argument(0)
		if o := fnObj(f); o != nil {
			mi := &jsvFnInfo{name: call.Argument(1).String(), length: int(call.Argument(2).ToFloat())}
			if len(call.Arguments) > 3 {
				mi.src = int(call.Argument(3).ToFloat())
			} else if v := o.GetSymbol(sym); v != nil {
				if p, ok := v.Export().(*jsvFnInfo); ok {
					mi.src = p.src
				}
			}
			if err := o.DefineDataPropertySymbol(sym, vm.ToValue(mi), goja.FLAG_TRUE, goja.FLAG_FALSE, goja.FLAG_TRUE); err != nil {
				panic(err)
			}
		}
		return f
	})
}

// installGojaArrKind binds arrKind(a) and arrKindSet(a, k), the goja twin of
// arrKindBindings (jsrtint.go). The number is kept ON the array under a private
// symbol, so a copy (slice, concat, a literal) starts untagged, as jsArray's does.
func installGojaArrKind(vm *goja.Runtime) {
	sym := goja.NewSymbol("arrKind")
	arrObj := func(v goja.Value) *goja.Object {
		o, ok := v.(*goja.Object)
		if !ok || o.ClassName() != "Array" {
			return nil
		}
		return o
	}
	vm.Set("arrKind", func(call goja.FunctionCall) goja.Value {
		if o := arrObj(call.Argument(0)); o != nil {
			if v := o.GetSymbol(sym); v != nil {
				return v
			}
		}
		return vm.ToValue(0)
	})
	vm.Set("arrKindSet", func(call goja.FunctionCall) goja.Value {
		a := call.Argument(0)
		if o := arrObj(a); o != nil {
			if err := o.DefineDataPropertySymbol(sym, vm.ToValue(call.Argument(1).ToInteger()), goja.FLAG_TRUE, goja.FLAG_FALSE, goja.FLAG_TRUE); err != nil {
				panic(err)
			}
		}
		return a
	})
}

// installGojaCoroutines binds coNew(f), coResume(co, v) and coYield(v), the
// tag scripts' suspension primitive: f runs on a call stack of its own
// (goja.Coroutine), coResume answers {value, done} and coYield answers the next
// coResume's v. The goja twin of coroutineBindings (frozen.go).
func installGojaCoroutines(vm *goja.Runtime) {
	vm.Set("coNew", func(call goja.FunctionCall) goja.Value {
		f, ok := goja.AssertFunction(call.Argument(0))
		if !ok {
			panic(vm.NewTypeError("coNew needs a function"))
		}
		return vm.ToValue(vm.NewCoroutine(f))
	})
	vm.Set("coResume", func(call goja.FunctionCall) goja.Value {
		c, ok := call.Argument(0).Export().(*goja.Coroutine)
		if !ok {
			panic(vm.NewTypeError("coResume needs a coroutine"))
		}
		v, done := c.Resume(call.Argument(1))
		res := vm.NewObject()
		res.Set("value", v)
		res.Set("done", done)
		return res
	})
	vm.Set("coYield", func(call goja.FunctionCall) goja.Value {
		return vm.Yield(call.Argument(0))
	})
}

// installGojaCaseMapping replaces goja's String.prototype.toUpperCase and
// toLowerCase with caseMapMode (abnf/casemap.go) and binds unicodeCase(s, mode), so
// the grammar host under goja answers exactly what -frozen, llvm.Run and a native
// binary answer: goja's own methods use different Unicode tables.
func installGojaCaseMapping(vm *goja.Runtime) {
	vm.Set("__goCaseMap", caseMapMode)
	// String.prototype so it reaches a primitive receiver too; `this` is boxed
	// there, hence the explicit String(this).
	// A Go string cannot carry a lone surrogate (goja hands one over as U+FFFD),
	// and a lone surrogate has no case and stops a Final_Sigma context like any
	// uncased character, so the text between them is mapped on its own and they
	// pass through.
	if _, err := vm.RunString(`
		function unicodeCase(s, mode) {
			s = String(s);
			var out = "", start = 0;
			for (var i = 0; i < s.length; i++) {
				var c = s.charCodeAt(i);
				if (c < 0xD800 || c > 0xDFFF) continue;
				if (c <= 0xDBFF && i + 1 < s.length) {
					var d = s.charCodeAt(i + 1);
					if (d >= 0xDC00 && d <= 0xDFFF) { i++; continue; }
				}
				out += __goCaseMap(s.substring(start, i), mode) + String.fromCharCode(c);
				start = i + 1;
			}
			return start === 0 ? __goCaseMap(s, mode) : out + __goCaseMap(s.substring(start), mode);
		}
		String.prototype.toUpperCase = function () { return unicodeCase(this, 3) };
		String.prototype.toLowerCase = function () { return unicodeCase(this, 6) };
	`); err != nil {
		panic(err)
	}
}

// installGojaCharAt makes String.prototype.charAt answer a lone surrogate's own
// unit, as ECMAScript does: the pinned goja's charAt ends in a rune to string
// conversion, which writes U+FFFD. Only a U+FFFD result is re-read, through
// substring, which keeps the unit. Twin of charAt in jsrt.go and mid 21 in
// languages/lib/runtime.c.
func installGojaCharAt(vm *goja.Runtime) {
	proto := vm.Get("String").ToObject(vm).Get("prototype").ToObject(vm)
	charAt := proto.Get("charAt").Export().(func(goja.FunctionCall) goja.Value)
	substring := proto.Get("substring").Export().(func(goja.FunctionCall) goja.Value)
	fn := func(call goja.FunctionCall) goja.Value {
		c := charAt(call)
		if c.String() != "\uFFFD" {
			return c
		}
		i := call.Argument(0).ToInteger()
		return substring(goja.FunctionCall{This: call.This, Arguments: []goja.Value{vm.ToValue(i), vm.ToValue(i + 1)}})
	}
	if err := proto.DefineDataProperty("charAt", vm.ToValue(fn), goja.FLAG_TRUE, goja.FLAG_FALSE, goja.FLAG_TRUE); err != nil {
		panic(err)
	}
}

// installGojaMathSurface replaces every method of goja's Math with the Go twin's
// (standardJSBindings in jsrt.go), so the grammar host under goja answers exactly
// what -frozen, llvm.Run and - through host ids 10..93 of languages/lib/runtime.c -
// a native binary answer. goja's own methods differ: its Math.imul and Math.clz32
// do not reduce an operand outside the int64 range modulo 2^32 (`Math.imul(x,
// 1e300)` is -1, `Math.clz32(1e300)` is 0, where node and the other engines give 0
// and 32). Members goja has and the twin lacks are dropped with the rest. The
// constants are left alone: they are non-configurable and equal in both.
//
// The twin's functions only convert their arguments with rt.toNumber, which needs
// no runtime state for a float64, so a zero jsrt serves.
func installGojaMathSurface(vm *goja.Runtime) {
	if _, err := vm.RunString(`Object.getOwnPropertyNames(Math).forEach(function (k) {
		if (typeof Math[k] === "function") { delete Math[k] }
	})`); err != nil {
		panic(err)
	}
	twin := standardJSBindings()["Math"].(*jsObject)
	m := vm.Get("Math").ToObject(vm)
	rt := &jsrt{}
	for _, k := range twin.keys {
		hf, ok := twin.props[k].(*hostFunc)
		if !ok {
			continue
		}
		fn := hf.fn
		if err := m.Set(k, func(call goja.FunctionCall) goja.Value {
			args := make([]interface{}, len(call.Arguments))
			for i, a := range call.Arguments {
				args[i] = a.ToFloat()
			}
			return vm.ToValue(fn(rt, 0, args))
		}); err != nil {
			panic(err)
		}
	}
}

// hostAPIObject exposes one of the host API maps (c, abnf, llvm) to goja.
// Binding the Go map itself would make goja wrap the accessed member again on
// EVERY read (a new native function object per abnf.newRule() call, plus the
// wrapper of a nested map like llvm.ir). Here each member is wrapped once and the same goja Value is handed out
// afterwards; a nested map (abnf.oid, llvm.ir, ...) becomes a caching object of
// its own. Members whose Go value is rebound while the run goes on are named as
// live and never cached. Everything else behaves like goja's Go-map wrapper:
// reads go through vm.ToValue, writes store the exported value in the map.
type hostAPIObject struct {
	vm    *goja.Runtime
	m     reflect.Value // The Go map, or a pointer to it.
	live  map[string]bool
	cache map[string]goja.Value
}

// newHostAPIObject binds a string-keyed Go map (or a pointer to one) as a
// caching JS object.
func newHostAPIObject(vm *goja.Runtime, goMap interface{}, live ...string) *goja.Object {
	o := &hostAPIObject{vm: vm, m: reflect.ValueOf(goMap), cache: map[string]goja.Value{}}
	if len(live) > 0 {
		o.live = make(map[string]bool, len(live))
		for _, key := range live {
			o.live[key] = true
		}
	}
	return vm.NewDynamicObject(o)
}

// goMap resolves the map behind the binding. It is kept as a pointer where the
// caller has one (the 'c' map), so that replacing the whole map stays visible.
func (o *hostAPIObject) goMap() reflect.Value {
	if o.m.Kind() == reflect.Ptr {
		return o.m.Elem()
	}
	return o.m
}

func (o *hostAPIObject) Get(key string) goja.Value {
	if v, ok := o.cache[key]; ok {
		return v
	}
	mv := o.goMap().MapIndex(reflect.ValueOf(key))
	if !mv.IsValid() {
		return nil // Not there: JS sees undefined.
	}
	raw := mv.Interface()
	var val goja.Value
	if sub := reflect.ValueOf(raw); sub.Kind() == reflect.Map && sub.Type().Key().Kind() == reflect.String {
		val = newHostAPIObject(o.vm, raw) // A nested API group: cache its members too.
	} else {
		val = o.vm.ToValue(raw)
	}
	if !o.live[key] {
		o.cache[key] = val
	}
	return val
}

func (o *hostAPIObject) Set(key string, val goja.Value) bool {
	m := o.goMap()
	elem := m.Type().Elem()
	v := reflect.ValueOf(val.Export())
	switch {
	case !v.IsValid(): // null/undefined.
		v = reflect.Zero(elem)
	case v.Type().AssignableTo(elem):
	case elem.Kind() != reflect.String && v.Type().ConvertibleTo(elem):
		// A number into one of the typed groups (abnf.oid and friends). String
		// targets are left out on purpose: Go would turn a number into the
		// character of that code point instead of rejecting it.
		v = v.Convert(elem)
	default:
		return false
	}
	m.SetMapIndex(reflect.ValueOf(key), v)
	delete(o.cache, key)
	return true
}

func (o *hostAPIObject) Has(key string) bool {
	return o.goMap().MapIndex(reflect.ValueOf(key)).IsValid()
}

func (o *hostAPIObject) Delete(key string) bool {
	o.goMap().SetMapIndex(reflect.ValueOf(key), reflect.Value{})
	delete(o.cache, key)
	return true
}

func (o *hostAPIObject) Keys() []string {
	m := o.goMap()
	keys := make([]string, 0, m.Len())
	for _, k := range m.MapKeys() {
		keys = append(keys, k.String())
	}
	return keys
}
