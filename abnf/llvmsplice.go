package abnf

import (
	"github.com/llir/llvm/asm"
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/types"
)

// ----------------------------------------------------------------------------
// Splicing a separately compiled runtime module into the module being emitted.
//
// WHY THIS EXISTS. A compiler grammar that moves its runtime out of hand-emitted
// IR and into C (compiled by languages/c-to-llvm-ir.abnf into a checked-in .ll)
// has two consumers of that runtime, not one:
//
//   - `-exe`, which hands the .ll to clang alongside the module (c.runtime), and
//   - `llvm.Run`, the built-in IR interpreter, which links NOTHING: it executes
//     exactly the module it is given, so a runtime function that is only a
//     `declare` is an unresolved external there.
//
// The MetaJS and Lua compilers get away with declaring their runtime because a Go
// twin (abnf/jsrt*.go) answers the same externs under llvm.Run. Bash has no Go
// twin - its runtime has always been part of the emitted module - so the C
// version has to reach llvm.Run as real definitions. Splice does that: it parses
// the runtime .ll and appends its type definitions, globals and functions to the
// module being built, which keeps the module self-contained exactly as it was
// when the runtime was hand-emitted.
//
// A name already present in the destination wins and the source copy is skipped,
// so a grammar may splice first and then keep using its own declaration of, say,
// putchar.

// spliceIR parses the LLVM IR file at path and appends everything in it to m.
// It returns "" on success or a human-readable error string.
func spliceIR(m *ir.Module, path string) string {
	src, err := asm.ParseFile(path)
	if err != nil {
		return "cannot parse " + path + ": " + err.Error()
	}
	haveType := map[string]bool{}
	for _, t := range m.TypeDefs {
		haveType[typeDefName(t)] = true
	}
	for _, t := range src.TypeDefs {
		if n := typeDefName(t); n != "" && !haveType[n] {
			haveType[n] = true
			m.TypeDefs = append(m.TypeDefs, t)
		}
	}
	haveGlobal := map[string]bool{}
	for _, g := range m.Globals {
		haveGlobal[g.Name()] = true
	}
	for _, g := range src.Globals {
		if !haveGlobal[g.Name()] {
			haveGlobal[g.Name()] = true
			m.Globals = append(m.Globals, g)
		}
	}
	haveFunc := map[string]bool{}
	for _, f := range m.Funcs {
		haveFunc[f.Name()] = true
	}
	for _, f := range src.Funcs {
		if !haveFunc[f.Name()] {
			haveFunc[f.Name()] = true
			m.Funcs = append(m.Funcs, f)
		}
	}
	return ""
}

// typeDefName is the name of a named type definition, or "" if it has none.
func typeDefName(t types.Type) string {
	if n, ok := t.(interface{ Name() string }); ok {
		return n.Name()
	}
	return ""
}

// spliceFunc returns the function named name in m. A grammar uses it to reach a
// function that came in through spliceIR. A missing name is a hard error rather
// than a null the grammar would have to test for: it means the runtime .ll is out
// of date with the grammar, and silently emitting a call to nothing is exactly the
// failure mode this whole layer exists to remove.
func spliceFunc(m *ir.Module, name string) *ir.Func {
	for _, f := range m.Funcs {
		if f.Name() == name {
			return f
		}
	}
	panic("llvm.SpliceFunc: no function @" + name + " in the module - regenerate the runtime .ll")
}

// spliceGlobal returns the global named name in m; a missing name is a hard error
// for the same reason spliceFunc's is.
func spliceGlobal(m *ir.Module, name string) *ir.Global {
	for _, g := range m.Globals {
		if g.Name() == name {
			return g
		}
	}
	panic("llvm.SpliceGlobal: no global @" + name + " in the module - regenerate the runtime .ll")
}
