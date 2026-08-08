package abnf

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/dop251/goja"
)

// TestMetaJSHostGlobalsAgree pins the MetaJS HOST-GLOBAL SET across the three
// engines that implement it, and it exists because nothing else in this project
// can see them disagree (docs/todo.md 4.1).
//
// The three engines, and the manual's chapter 7.2 rule they are the whole of:
//
//	languages/lib/runtime.c        seed_root(...)   the C FLOOR, every native binary
//	abnf/jsrtint.go                programJSBindings()  the Go twin, llvm.Run
//	languages/metajs-interpreter.abnf   hostGlobals  the tree-walking half
//
// WHY A GO TEST AND NOT AN ASSERTION IN THE RATCHET. The --full runner makes both
// halves of a language run the SAME file and report the same assertion count, so
// an assertion on a global that only one half binds fails on the other half by
// construction - and it cannot be guarded either, because `typeof sprint` is
// "function" under the compiler and a hard `variable not defined: sprint` under
// the interpreter. That is exactly why eight names diverged unnoticed. This test
// reads the three sources instead of running them, so it does not care.
//
// The sets are now equal, so tests/metajs-test-full.js CAN assert every name on
// both halves and does (SECTION 31); this test is what stops the next addition
// from landing in one engine only.
func TestMetaJSHostGlobalsAgree(t *testing.T) {
	floor := floorSeedRootNames(t)
	twin := twinProgramGlobalNames()
	interp := interpreterHostGlobalNames(t)

	if len(floor) < 40 {
		t.Fatalf("parsed only %d seed_root names from runtime.c - the parse broke, not the code", len(floor))
	}
	if len(interp) < 40 {
		t.Fatalf("parsed only %d hostGlobals names from metajs-interpreter.abnf - the parse broke", len(interp))
	}

	diffNames(t, "languages/lib/runtime.c (seed_root)", floor, "abnf/jsrtint.go (programJSBindings)", twin)
	diffNames(t, "languages/lib/runtime.c (seed_root)", floor, "languages/metajs-interpreter.abnf (hostGlobals)", interp)
}

// twinGrammarOnlyGlobals are the names programJSBindings deliberately keeps
// beyond the floor's list. There is exactly one, it is `__`-prefixed - i.e. not a
// name a MetaJS program is meant to see - and its reason is at the delete() calls
// in jsrtint.go: java-to-llvm-ir.abnf reads __jmath through jvScopeProbe, which
// takes the hit arm under llvm.Run and falls back to layer 2's own java.lang
// descriptor in a native binary. Adding an entry here is a DECISION, not a fix.
var twinGrammarOnlyGlobals = map[string]bool{"__jmath": true}

func twinProgramGlobalNames() []string {
	out := []string{}
	for k := range programJSBindings() {
		if twinGrammarOnlyGlobals[k] {
			continue
		}
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

var seedRootRe = regexp.MustCompile(`seed_root\("([^"]+)"`)

func floorSeedRootNames(t *testing.T) []string {
	src := readRepoFile(t, "../languages/lib/runtime.c")
	out := []string{}
	for _, m := range seedRootRe.FindAllStringSubmatch(src, -1) {
		out = append(out, m[1])
	}
	sort.Strings(out)
	return out
}

var interpKeyRe = regexp.MustCompile(`"([A-Za-z_$][A-Za-z0-9_$]*)"\s*:`)

// interpreterHostGlobalNames reads the `var hostGlobals = {` object literal out of
// metajs-interpreter.abnf. The literal's own entries are the lines at exactly two
// levels of indentation (8 spaces); a nested function body is deeper and a comment
// is skipped outright, so the only way this misreads is a future edit that puts a
// `"key":` of its own at the entry level - in which case it fails loudly, which is
// the point.
func interpreterHostGlobalNames(t *testing.T) []string {
	src := readRepoFile(t, "../languages/metajs-interpreter.abnf")
	lines := strings.Split(src, "\n")
	start := -1
	for i, ln := range lines {
		if strings.HasPrefix(ln, "    var hostGlobals = {") {
			start = i
			break
		}
	}
	if start < 0 {
		t.Fatalf("metajs-interpreter.abnf: no `    var hostGlobals = {` line")
	}
	out := []string{}
	closed := false
	for _, ln := range lines[start+1:] {
		if ln == "    }" {
			closed = true
			break
		}
		trimmed := strings.TrimLeft(ln, " ")
		if strings.HasPrefix(trimmed, "//") {
			continue
		}
		if len(ln)-len(trimmed) != 8 {
			continue
		}
		for _, m := range interpKeyRe.FindAllStringSubmatch(ln, -1) {
			out = append(out, m[1])
		}
	}
	if !closed {
		t.Fatalf("metajs-interpreter.abnf: hostGlobals literal never closed at `    }`")
	}
	sort.Strings(out)
	return out
}

func readRepoFile(t *testing.T, path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

func diffNames(t *testing.T, aName string, a []string, bName string, b []string) {
	t.Helper()
	set := func(xs []string) map[string]bool {
		m := map[string]bool{}
		for _, x := range xs {
			m[x] = true
		}
		return m
	}
	as, bs := set(a), set(b)
	missing := []string{}
	extra := []string{}
	for _, x := range a {
		if !bs[x] {
			missing = append(missing, x)
		}
	}
	for _, x := range b {
		if !as[x] {
			extra = append(extra, x)
		}
	}
	if len(missing) == 0 && len(extra) == 0 {
		return
	}
	t.Errorf("MetaJS host globals disagree.\n"+
		"  %s has %d names, %s has %d.\n"+
		"  only in %s: %v\n"+
		"  only in %s: %v\n"+
		"A host global means THREE engines (manual 7.2): seed_root in\n"+
		"languages/lib/runtime.c, programJSBindings in abnf/jsrtint.go, and\n"+
		"hostGlobals in languages/metajs-interpreter.abnf. Add it to all three, or\n"+
		"write down at the site why it can only exist in one.",
		aName, len(a), bName, len(b), aName, missing, bName, extra)
}

// TestMathMembersAgree pins the MEMBERS of Math across the same three engines,
// and it exists because TestMetaJSHostGlobalsAgree above could not see them: the
// three engines all bound the NAME `Math` and then offered wildly different
// objects behind it. At 1367c23 the C floor seeded eleven methods and two
// properties, standardJSBindings offered thirty-two and eight, and goja - the
// grammar host the interpreter halves run on - offered thirty-five and eight. So
//
//	Math.sin(1)     0.8414709848078965 under llvm.Run, and
//	                `js runtime error: unknown method 'sin'` in the native binary
//	Math.hypot(...) Infinity for (1e300, 1e300) under -frozen, and
//	                1.4142135623730952e+300 under goja
//	Math.clz32(1)   31 under goja, and "call of a non function value" under -frozen
//
// were all live, and no gate in this project could see any of them (docs/todo.md
// 1.1). The sets are equal now, up to the two documented exceptions below.
func TestMathMembersAgree(t *testing.T) {
	floor := floorMathMemberNames(t)
	twin := twinMathMemberNames()
	goja := gojaMathMemberNames(t)

	if len(floor) < 30 {
		t.Fatalf("parsed only %d Math members from runtime.c - the parse broke, not the code", len(floor))
	}
	diffNames(t, "languages/lib/runtime.c (seed_host/obj_put on math)", floor,
		"abnf/jsrt.go (standardJSBindings mathObj)", twin)
	diffNames(t, "abnf/jsrt.go (standardJSBindings mathObj)", twin,
		"goja's own Math (the grammar host of every interpreter half)", goja)
}

// gojaOnlyMathMembers are the members goja's Math has that the other two engines
// deliberately do not. Adding an entry here is a DECISION, not a fix.
//
//	random  a nondeterministic member CANNOT be converged: the matrix demands
//	        byte-identical output from goja and -frozen, so any Math.random that
//	        actually answered differently per run would fail it by construction,
//	        and a deterministic one would agree with neither goja nor any real
//	        toolchain. `Math.random` therefore stays a goja-only name: a program
//	        that calls it works in an interpreter half under goja and aborts
//	        under -frozen, which is the honest report of the situation.
var gojaOnlyMathMembers = map[string]bool{"random": true}

var seedHostMathRe = regexp.MustCompile(`seed_host\(math, "([^"]+)"`)
var objPutMathRe = regexp.MustCompile(`obj_put\(math, mk_cstr\("([^"]+)"\)`)

func floorMathMemberNames(t *testing.T) []string {
	src := readRepoFile(t, "../languages/lib/runtime.c")
	out := []string{}
	for _, re := range []*regexp.Regexp{seedHostMathRe, objPutMathRe} {
		for _, m := range re.FindAllStringSubmatch(src, -1) {
			out = append(out, m[1])
		}
	}
	sort.Strings(out)
	return out
}

func twinMathMemberNames() []string {
	m, _ := standardJSBindings()["Math"].(*jsObject)
	out := []string{}
	if m != nil {
		for k := range m.props {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

// gojaMathMemberNames asks goja itself rather than reading a source file, which
// is the only way to get this set: goja's Math is native to the library.
func gojaMathMemberNames(t *testing.T) []string {
	vm := goja.New()
	v, err := vm.RunString(`Object.getOwnPropertyNames(Math).join(",")`)
	if err != nil {
		t.Fatalf("goja: %v", err)
	}
	out := []string{}
	for _, n := range strings.Split(v.String(), ",") {
		if n == "" || gojaOnlyMathMembers[n] {
			continue
		}
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}
