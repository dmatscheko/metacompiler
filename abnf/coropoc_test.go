package abnf

// coropoc_test.go - the llvm.Run half of the COROUTINE proof of concept
// (docs/runtime-next-plan.md, "Coroutines - the fifth-language wall").
//
// It parses tests/coro-poc/gen.ll - the SAME module tests/coro-poc/build.sh
// links into a native binary - and runs it through runJSModule, which is exactly
// what llvm.RunJS does for every compiler grammar.  So js_genfn and js_yield here
// are abnf/jsrt.go's goroutine and two channels, and a handle is an index into a
// Go table; in the native binary they are a pthread and one condition variable
// and a handle is a Cell *.  The two outputs have to agree byte for byte, which
// is the whole claim.
//
//	go test ./abnf/ -run TestCoroPoC -v
//
// The expected output is NOT hard-coded here on purpose: this test writes what
// it got to stdout, and build.sh diffs the two engines.  A hard-coded string
// would let the two halves agree with the test instead of with each other.

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/llir/llvm/asm"
)

func TestCoroPoC(t *testing.T) {
	// MEC_CORO_LL points the same harness at another hand-written module, which
	// is how the handoff cost of the two engines was measured against one file.
	path := os.Getenv("MEC_CORO_LL")
	if path == "" {
		path = filepath.Join("..", "tests", "coro-poc", "gen.ll")
	}
	if _, err := os.Stat(path); err != nil {
		t.Skipf("coro PoC module not present: %v", err)
	}
	m, err := asm.ParseFile(path)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("llvm.Run half failed: %v", r)
		}
	}()
	res := runJSModule(m, "jsmain")
	// '#' so build.sh can strip this line and diff the rest against the native
	// binary's stdout without the two halves needing to agree on it.
	fmt.Fprintf(os.Stderr, "# coro PoC: llvm.Run returned %d\n", res.Ret)
}
