package abnf

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
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

// Run executes the given source string in the global context.
// Compiled programs are cached: Code with a UID (ID > 0, assigned by correctReferencesAndIDs())
// is cached per UID. The comparison with the cached source is cheap (usually only a pointer
// comparison) and protects against UID collisions between independently numbered a-grammars.
// All other code (ID <= 0, e.g. the start script, includes, or tags that were built via JS and
// never got a UID) is cached by its name plus source text. The name is part of the key because
// it is compiled into the program (for stack traces and relative paths), so byte-identical
// code from two different files must not share one program.
func (cs *commonscript) Run(name, src string, ID int) (goja.Value, error) {
	var p *goja.Program
	if ID > 0 {
		if ID >= len(cs.codeCache) {
			tmp := make([]cachedProgram, ID*2)
			cs.codeCache = append(cs.codeCache, tmp...)
		} else if cs.codeCache[ID].src == src {
			p = cs.codeCache[ID].p
		}
	} else {
		p = cs.codeCacheBySrc[name+"\x00"+src]
	}

	// Compile and cache on the first run.
	if p == nil {
		var err error
		p, err = goja.Compile(name, src, true)
		if err != nil {
			return nil, err
		}
		if ID > 0 {
			cs.codeCache[ID] = cachedProgram{src: src, p: p}
		} else {
			cs.codeCacheBySrc[name+"\x00"+src] = p
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
		srcCode := string(dat)

		_, err = common.Run(includeFileName, srcCode, -1)
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
		// True when -trace/-cfg collect source positions: the compilers then
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
		"curFile":    func() string { return traceSrcName },
		"findImport": func(relPath string) string { return findImportFile(relPath) },
		"readFile":   func(path string) string { return readImportFile(path) },
		"pushSource": func(name, text string) { pushTraceSource(name, text) },
		"popSource":  func() { popTraceSource() },
	}
	vm.Set("c", compilerFuncMap)
	vm.Set("abnf", r.AbnfFuncMap)
	vm.Set("llvm", llvmFuncMap)

	return &common
}
