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
	vm        *goja.Runtime
	codeCache map[string]*goja.Program
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
func (cs *commonscript) Run(name, src string) (goja.Value, error) {
	p := cs.codeCache[src]

	// Cache precompiled data
	if p == nil {
		var err error
		p, err = goja.Compile(name, src, true)
		if err != nil {
			return nil, err
		}
		cs.codeCache[src] = p
	}

	return cs.vm.RunProgram(p)
}

// This is used by parser and compiler.
func initFuncMapCommon(vm *goja.Runtime, compilerFuncMap *map[string]r.Object, currentFileName string, preventDefaultOutput bool) *commonscript {
	var common commonscript

	common.vm = vm
	common.codeCache = map[string]*goja.Program{}

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

	vm.Set("include", func(fileName string) bool { // It seems impossible to get the file name of the currently calling script. So the base directory is always the directory of the first ABNF.

		if fileName == "" {
			return false
		}
		fullFileName := filepath.Dir(currentFileName) + string(os.PathSeparator) + fileName
		dat, err := ioutil.ReadFile(fullFileName)
		if err != nil {
			panic(err)
		}
		srcCode := string(dat)

		_, err = common.Run(fullFileName, srcCode)
		if err != nil {
			panic(err.Error() + "\nError was in " + fullFileName)
		}

		return true
	})

	// vm.Set("writable", func(v interface{}) *interface{} {
	// 	return &v
	// })
	// vm.Set("nonwritable", func(v *interface{}) interface{} {
	// 	return *v
	// })

	*compilerFuncMap = map[string]r.Object{
		"parse": func(agrammar *r.Rules, srcCode string, useBlockList bool, useFoundList bool, traceEnabled bool) *r.Rules { // TODO: Implement a feature to state the start rule.
			productions, _ := ParseWithAgrammar(agrammar, srcCode, currentFileName, useBlockList, useFoundList, traceEnabled)
			return productions
		},
		"compileWithProlog": func(asg *r.Rules, aGrammar *r.Rules, slot int, traceEnabled bool) interface{} {
			return compileASGInternal(asg, aGrammar, currentFileName, slot, traceEnabled, false)
		},
		"ABNFagrammar": AbnfAgrammar,
	}
	vm.Set("c", compilerFuncMap)
	vm.Set("abnf", r.AbnfFuncMap)
	vm.Set("llvm", llvmFuncMap)

	return &common
}
