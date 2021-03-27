package abnf

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"14.gy/mec/abnf/r"
	"github.com/dop251/goja"
)

// ----------------------------------------------------------------------------
// Scripting subsystem code for parser and compiler

type commonscript struct {
	vm               *goja.Runtime
	codeCache        []*goja.Program
	codeCacheInclude map[string]*goja.Program
	referencesCache  *references
}

// Stripped down and slightly modified version of stconv.Unquote()
func Unescape(s string) (string, error) {
	// Is it trivial? Avoid allocation.
	if !strings.ContainsRune(s, '\\') {
		if utf8.ValidString(s) {
			return s, nil
		}
	}

	var runeTmp [utf8.UTFMax]byte
	buf := make([]byte, 0, 3*len(s)/2) // Try to avoid more allocations.
	for len(s) > 0 {
		c, multibyte, ss, err := strconv.UnquoteChar(s, 0)
		if err != nil {
			return "", err
		}
		s = ss
		if c < utf8.RuneSelf || !multibyte {
			buf = append(buf, byte(c))
		} else {
			n := utf8.EncodeRune(runeTmp[:], c)
			buf = append(buf, runeTmp[:n]...)
		}
	}
	return string(buf), nil
}

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

// Run executes the given string in the global context.
func (cs *commonscript) Run(name, src string, ID int) (goja.Value, error) {
	var p *goja.Program
	if ID >= 0 {
		if ID >= len(cs.codeCache) {
			tmp := make([]*goja.Program, ID*2)
			cs.codeCache = append(cs.codeCache, tmp...)
		} else {
			p = cs.codeCache[ID]
		}
	} else {
		p = cs.codeCacheInclude[name]
	}

	// Cache precompiled data
	if p == nil {
		var err error
		p, err = goja.Compile(name, src, true)
		if err != nil {
			return nil, err
		}
		if ID >= 0 {
			cs.codeCache[ID] = p
		} else {
			cs.codeCacheInclude[name] = p
		}
	}

	return cs.vm.RunProgram(p)
}

func (cs *commonscript) getCurrentModuleFileName() string {
	var buf [2]goja.StackFrame
	frames := cs.vm.CaptureCallStack(2, buf[:0])
	if len(frames) < 2 {
		return "."
	}
	return frames[1].SrcName()
}

// This is used by parser and compiler.
func NewCommonScript(vm *goja.Runtime, compilerFuncMap *map[string]r.Object, preventDefaultOutput bool) *commonscript {
	var common commonscript

	common.vm = vm
	common.codeCache = make([]*goja.Program, 100)
	common.codeCacheInclude = map[string]*goja.Program{}

	if preventDefaultOutput { // Script output disabled.
		vm.Set("print", func(a ...interface{}) (n int, err error) { return 0, nil })
		vm.Set("println", func(a ...interface{}) (n int, err error) { return 0, nil })
		vm.Set("printf", func(format string, a ...interface{}) (n int, err error) { return 0, nil })
	} else { // Script output enabled.
		vm.Set("print", fmt.Print)
		vm.Set("println", fmt.Println)
		vm.Set("printf", fmt.Printf)
	}

	vm.Set("sprintf", fmt.Sprintf) // Sprintf is no output.
	vm.Set("exit", os.Exit)

	vm.Set("sleep", func(d time.Duration) { time.Sleep(d * time.Millisecond) })

	vm.Set("append", func(t []interface{}, v ...interface{}) interface{} {
		tmp := append(t, v...)
		return &tmp
	})

	vm.Set("unescape", Unescape)
	vm.Set("unescapeTilde", UnescapeTilde)

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

	vm.Set("correctReferencesAndIDs", func(agrammar *r.Rules) {
		common.referencesCache = NewReferences()
		common.referencesCache.correctReferencesAndIDs(agrammar)
	})

	// vm.Set("writable", func(v interface{}) *interface{} {
	// 	return &v
	// })
	// vm.Set("nonwritable", func(v *interface{}) interface{} {
	// 	return *v
	// })

	*compilerFuncMap = map[string]r.Object{
		"parse": func(agrammar *r.Rules, srcCode string, useBlockList bool, useFoundList bool, traceEnabled bool) *r.Rules { // TODO: Implement a feature to state the start rule.
			productions, err := ParseWithAgrammar(agrammar, srcCode, common.getCurrentModuleFileName(), useBlockList, useFoundList, traceEnabled, preventDefaultOutput)
			if err != nil {
				panic(err)
			}
			return productions
		},
		"compileRunStartScript": func(asg *r.Rules, aGrammar *r.Rules, slot int, traceEnabled bool) interface{} {
			return compileASGInternal(asg, aGrammar, common.getCurrentModuleFileName(), slot, traceEnabled, false)
		},
		"ABNFagrammar": AbnfAgrammar,
	}
	vm.Set("c", compilerFuncMap)
	vm.Set("abnf", r.AbnfFuncMap)
	vm.Set("llvm", llvmFuncMap)

	return &common
}
