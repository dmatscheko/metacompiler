package abnf

import (
	"fmt"
	"os"
	"strings"

	"github.com/llir/llvm/asm"
	"github.com/llir/llvm/ir"
)

// ----------------------------------------------------------------------------
// Linking a separately compiled runtime into a run of the IR interpreter.
//
// docs/runtime-rework-plan.md wants every layer compiled by this repository, which
// means a language runtime is a .ll produced by one of the grammars in languages/
// (lib/runtime.ll from runtime.c, lib/batch-rt.ll from batch-rt.c) rather than IR
// text hand-written inside the emitter. For -exe that is enough on its own: clang
// links the program object against the runtime object.
//
// llvm.Run has no linker, so a language whose runtime moved out of the module would
// call declarations with no bodies. linkRuntimeModules is that linker, and it is
// deliberately the SAME shape as clang's: the runtime stays a separate module, its
// globals get their own storage, and a declaration is bound to the definition that
// carries its name - in both directions, so a runtime may also call back into a
// function the program module generates.
//
// Keeping the modules separate is not just simplicity. Our C compiler carries a
// pointer VALUE as i32* while several emitters write i8*, so `declare i32
// @rt_strlen(i8*)` against `define i32 @rt_strlen(i32*)` is a type mismatch that
// separate compilation erases and merging into one module would not - exactly as
// metajs's `js_str_mem(i8*, i64)` has always been resolved against a definition
// taking i32*. The interpreter's memory is a flat byte array and every
// getelementptr names its element type explicitly, so the pointer stays a pointer.

// linkRuntimeModules parses each .ll runtime input and wires it into the machine.
// Non-.ll entries (.c/.o/.a, which only clang can consume) are ignored, so the same
// list a grammar hands to llvm.BuildExecutable can be handed to llvm.Run.
func (ma *machine) linkRuntimeModules(m *ir.Module, paths []string) {
	defs := map[string]*ir.Func{}
	for _, f := range m.Funcs {
		if len(f.Blocks) > 0 {
			defs[f.Name()] = f
		}
	}

	var extra []*ir.Module
	for _, p := range paths {
		if !strings.HasSuffix(p, ".ll") {
			continue
		}
		data, err := os.ReadFile(p)
		if err != nil {
			panic("llvm.Run(): cannot read the runtime module " + p + ": " + err.Error())
		}
		rm, err := asm.ParseString(p, string(data))
		if err != nil {
			panic("llvm.Run(): cannot parse the runtime module " + p + ": " + err.Error())
		}
		for _, g := range rm.Globals {
			off := ma.alloc(ma.sizeOf(g.ContentType))
			ma.globals[g] = off
			if g.Init != nil {
				ma.writeConst(off, g.Init)
			}
		}
		for _, f := range rm.Funcs {
			if len(f.Blocks) == 0 {
				continue
			}
			if prev, dup := defs[f.Name()]; dup && prev != f {
				panic(fmt.Sprintf("llvm.Run(): duplicate symbol %s - it is defined by the program module and by %s", f.Name(), p))
			}
			defs[f.Name()] = f
			if _, ok := ma.funcs[f.Name()]; !ok {
				ma.funcs[f.Name()] = f
			}
		}
		extra = append(extra, rm)
	}

	// The link step proper: a block-less function object anywhere in the set gets a
	// handler that calls the definition of that name. Calls are decoded against the
	// function OBJECT, so this reaches every call site without touching one.
	bind := func(fs []*ir.Func) {
		for _, f := range fs {
			if len(f.Blocks) > 0 {
				continue
			}
			def, ok := defs[f.Name()]
			if !ok || def == f {
				continue
			}
			target := def
			ma.externBound[f] = func(args []uint64) uint64 { return ma.call(target, args) }
		}
	}
	bind(m.Funcs)
	for _, rm := range extra {
		bind(rm.Funcs)
	}
}
