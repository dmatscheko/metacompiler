package abnf

import (
	"os"
	"path/filepath"
)

// ImportRoots holds the -i include directories, in command-line order. An
// import that names a project file is searched relative to the imported-from
// program's directory first, then relative to each root.
var ImportRoots []string

// findImportFile resolves a grammar-supplied relative path (already mapped
// from the language's import syntax, e.g. "a/b/C.kt") against the current
// source file's directory and the -i roots. Returns the first existing
// regular file as a cleaned path, or "" when nothing matches.
func findImportFile(relPath string) string {
	if relPath == "" || filepath.IsAbs(relPath) {
		return ""
	}
	dirs := make([]string, 0, len(ImportRoots)+1)
	if traceSrcName != "" {
		dirs = append(dirs, filepath.Dir(traceSrcName))
	}
	dirs = append(dirs, ImportRoots...)
	for _, dir := range dirs {
		p := filepath.Join(dir, filepath.Clean(relPath))
		if st, err := os.Stat(p); err == nil && st.Mode().IsRegular() {
			return p
		}
	}
	return ""
}

// readImportFile loads a file previously located by findImportFile.
func readImportFile(path string) string {
	dat, err := os.ReadFile(path)
	if err != nil {
		panic("import file vanished: " + path)
	}
	return string(dat)
}

// pushTraceSource swaps the file/line attribution to an imported file for the
// duration of its nested compile walk, so warnings and errors inside it carry
// the right name and line numbers. popTraceSource restores the outer file.
type savedTraceSource struct {
	name   string
	starts []int
}

var traceSourceStack []savedTraceSource

func pushTraceSource(name, text string) {
	attributeModuleFuncs() // stamp functions compiled under the OUTGOING source
	traceSourceStack = append(traceSourceStack, savedTraceSource{traceSrcName, traceLineStarts})
	SetTraceSource(name, text)
}

func popTraceSource() {
	if len(traceSourceStack) == 0 {
		return
	}
	attributeModuleFuncs() // stamp functions compiled under the source being popped
	s := traceSourceStack[len(traceSourceStack)-1]
	traceSourceStack = traceSourceStack[:len(traceSourceStack)-1]
	traceSrcName, traceLineStarts = s.name, s.starts
}

// pushTraceSourceFile re-pushes a source by NAME, reusing the line-start table
// remembered when the file was first read (SetTraceSource). buildMain uses it to
// emit each imported file's deferred top-level items under that file's source -
// so the call graph attributes those functions to their real file - without
// having to carry the source text around a second time. popTraceSource undoes it.
func pushTraceSourceFile(name string) {
	attributeModuleFuncs() // stamp functions compiled under the OUTGOING source
	traceSourceStack = append(traceSourceStack, savedTraceSource{traceSrcName, traceLineStarts})
	traceSrcName, traceLineStarts = name, sourceLineStarts[name]
}
