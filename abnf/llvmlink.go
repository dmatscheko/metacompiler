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
//
// GLOBALS are bound the same way, and for the same reason. Bash's runtime state is
// SHARED with the emitted program - the program loads last_status, writes gvars[],
// reads read_eof - so a declaration on the program side has to name the runtime's
// one definition rather than getting storage of its own. A global whose initializer
// is nil is the declaration (`@gvars = external global [1024 x i8*]`, which is what
// ir.NewGlobal builds); one with an initializer is the definition. The two may
// disagree on the ELEMENT type exactly as a function signature may: `[1024 x i8*]`
// against `[1024 x i32*]` is the same 8192 bytes, every getelementptr names the type
// it is stepping through, and clang resolves the two objects by symbol name. What may
// NOT disagree is the size, so that is checked rather than assumed - a declaration
// narrower than its definition would silently address the wrong slot.

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
	// Where each global name is DEFINED, and which module said so, so that a
	// duplicate can name both sides.
	type gdef struct {
		g    *ir.Global
		from string
	}
	gdefs := map[string]gdef{}
	for _, g := range m.Globals {
		if g.Init != nil {
			gdefs[g.Name()] = gdef{g, "the program module"}
		}
	}

	type rtmod struct {
		m    *ir.Module
		path string
	}
	var extra []rtmod
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
			if g.Init == nil { // a declaration; it is bound below, like a function's
				continue
			}
			if prev, dup := gdefs[g.Name()]; dup {
				panic(fmt.Sprintf("llvm.Run(): duplicate global @%s - it is defined by %s and by %s",
					g.Name(), prev.from, p))
			}
			gdefs[g.Name()] = gdef{g, p}
			off := ma.alloc(ma.sizeOf(g.ContentType))
			ma.globals[g] = off
			ma.writeConst(off, g.Init)
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
		extra = append(extra, rtmod{rm, p})
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
	// The same, for globals: an initializer-less global anywhere in the set is
	// re-pointed at the storage of the definition carrying its name. newMachine has
	// already given the program module's declarations storage of their own, which
	// this overwrites - the abandoned bytes are the cost of not having to know, at
	// load time, whether a link is coming.
	bindG := func(gs []*ir.Global, where string) {
		for _, g := range gs {
			if g.Init != nil {
				continue
			}
			def, ok := gdefs[g.Name()]
			if !ok {
				panic(fmt.Sprintf("llvm.Run(): unresolved global @%s declared by %s - "+
					"this build links a runtime, so it is NOT given storage of its own", g.Name(), where))
			}
			if def.g == g {
				continue
			}
			if dn, gn := ma.sizeOf(def.g.ContentType), ma.sizeOf(g.ContentType); dn != gn {
				panic(fmt.Sprintf("llvm.Run(): global @%s is declared %d bytes by %s but defined %d bytes by %s",
					g.Name(), gn, where, dn, def.from))
			}
			ma.globals[g] = ma.globals[def.g]
		}
	}
	bind(m.Funcs)
	bindG(m.Globals, "the program module")
	for _, rm := range extra {
		bind(rm.m.Funcs)
		bindG(rm.m.Globals, rm.path)
	}
}
