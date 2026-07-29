package abnf

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"14.gy/mec/abnf/r"
	"github.com/dop251/goja"
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
	codeCache       []cachedProgram          // Compiled programs by tag UID (for tags and :script() commands with a UID > 0).
	codeCacheBySrc  map[string]*goja.Program // Compiled programs by name plus source text (for everything without a UID).
	referencesCache *references              // Keeps the tag UIDs stable over multiple correctReferencesAndIDs() calls.
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

// scriptName is the name of one script module: the file (or grammar module) it
// belongs to, plus the source position of the node it was written on. The name
// is only ever needed when a script is COMPILED for the first time (it is baked
// into the program for stack traces and for the relative path resolution of
// load()/store()/include()) or when an error is reported - but the compile walk
// runs a tag script ten thousand times per grammar, so formatting the name per
// execution was pure waste. The walk therefore passes the parts and the engines
// format on demand.
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

	if preventDefaultOutput { // Script output disabled.
		vm.Set("print", func(a ...interface{}) (n int, err error) { return 0, nil })
		vm.Set("println", func(a ...interface{}) (n int, err error) { return 0, nil })
		vm.Set("printf", func(format string, a ...interface{}) (n int, err error) { return 0, nil })
	} else { // Script output enabled (routed through outWriter so -pipe can capture it).
		vm.Set("print", func(a ...interface{}) (int, error) { return fmt.Fprint(outWriter, a...) })
		vm.Set("println", func(a ...interface{}) (int, error) { return fmt.Fprintln(outWriter, a...) })
		vm.Set("printf", func(format string, a ...interface{}) (int, error) { return fmt.Fprintf(outWriter, format, a...) })
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

	vm.Set("append", func(t []interface{}, v ...interface{}) interface{} {
		tmp := append(t, v...)
		return &tmp
	})

	vm.Set("unescape", r.Unescape)
	vm.Set("unescapeTilde", UnescapeTilde)

	// The UTF-8 byte length of a string. JS .length counts UTF-16 code units,
	// but the emitters need the byte count of the char arrays they emit
	// (lib/compile-core.js emitStr). goja exports the string as UTF-8.
	vm.Set("byteLen", func(s string) int { return len(s) })

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
		// True when -trace/-cfgraph collect source positions: the compilers then
		// emit js_srcpos statement markers (see lib/compile-core.js stmtPos).
		"tracing": TraceMarkersWanted(),
		// Import policy + source positions for clean grammar errors. warnImports
		// is the -warn-imports flag; file is the program being compiled; lineOf
		// turns an up.pos byte offset into a 1-based line (0 if unknown).
		"warnImports":     WarnUnresolvedImports,
		"warnUnsupported": WarnUnsupported,
		"file":            traceSrcName,
		"lineOf":          func(pos int) int { return lineOfPos(pos) },
		// The entry-point function name (-main flag, default "main").
		"mainName": EntryPoint,
		// Output path for a native executable (-exe flag); "" means run in the IR
		// interpreter instead. A -to-llvm-ir grammar hands its module to
		// llvm.BuildExecutable when this is set.
		"exePath": ExePath,
		// Project-file imports (the -i include roots): findImport locates a
		// grammar-mapped relative path ("a/b/C.kt"), readFile loads it, and
		// pushSource/popSource swap the file/line attribution around the
		// imported file's nested c.parse + c.compile walk. curFile is the
		// dynamic variant of "file" (which is snapshotted at map creation).
		"curFile":        func() string { return traceSrcName },
		"findImport":     func(relPath string) string { return findImportFile(relPath) },
		"readFile":       func(path string) string { return readImportFile(path) },
		"pushSource":     func(name, text string) { pushTraceSource(name, text) },
		"pushSourceFile": func(name string) { pushTraceSourceFile(name) },
		"popSource":      func() { popTraceSource() },
	}
	// The three host API objects are wrapped for goja ONCE per member instead of
	// per access (see hostAPIObject). c.localAsg is the only entry a run rebinds
	// (per tag / per :script()), so it is the only one read through every time.
	vm.Set("c", newHostAPIObject(vm, compilerFuncMap, "localAsg"))
	vm.Set("abnf", newHostAPIObject(vm, r.AbnfFuncMap))
	vm.Set("llvm", newHostAPIObject(vm, llvmFuncMap))

	return &common
}

// hostAPIObject exposes one of the host API maps (c, abnf, llvm) to goja.
// Binding the Go map itself made goja wrap the accessed member again on EVERY
// read: a tag script that calls abnf.newRule() a thousand times had goja build a
// thousand native function objects for that one Go function, and
// llvm.ir.NewModule() rebuilt the wrapper of the nested 'ir' map on top of it.
// Here each member is wrapped once and the same goja Value is handed out
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
