package abnf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
)

// TestRuntimeModuleWithLargeGEPIndex links a runtime module whose getelementptr
// indices llir prints in its u0x hexadecimal notation (any low-entropy constant
// >= 4096, e.g. byte 4096 of a long string literal). llir's own assembly parser
// rejects that spelling as a gep index, so the module goes through parseIR, which
// must also leave quoted text and values that do not fit the signed width alone.
func TestRuntimeModuleWithLargeGEPIndex(t *testing.T) {
	arr := types.NewArray(5000, types.I8)
	k := func(n int64) *constant.Int { return constant.NewInt(types.I32, n) }

	rt := ir.NewModule()
	buf := rt.NewGlobalDef("buf", constant.NewZeroInitializer(arr))
	get := rt.NewFunc("rt_get", types.I32)
	b := get.NewBlock("entry")
	p := b.NewGetElementPtr(arr, buf, k(0), k(0x1000))
	b.NewStore(constant.NewInt(types.I8, 7), p)
	q := b.NewGetElementPtr(arr, buf, k(0), k(0x1001))
	b.NewStore(constant.NewInt(types.I8, 5), q)
	sum := b.NewAdd(b.NewZExt(b.NewLoad(types.I8, p), types.I32), b.NewZExt(b.NewLoad(types.I8, q), types.I32))
	b.NewRet(sum)
	text := rt.String()
	if !strings.Contains(text, "u0x1000") || !strings.Contains(text, "u0x1001") {
		t.Fatalf("llir no longer prints the hex indices this test is about:\n%s", text)
	}
	path := filepath.Join(t.TempDir(), "rt.ll")
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}

	m := ir.NewModule()
	decl := m.NewFunc("rt_get", types.I32)
	mb := m.NewFunc("main", types.I32).NewBlock("entry")
	mb.NewRet(mb.NewCall(decl))
	if res := run(m, "main", "", path); res.Ret != 12 {
		t.Fatalf("main() = %d, want 12", res.Ret)
	}

	// A constant-expression index is read by the same llir helper.
	cexpr := "@buf = global [5000 x i8] zeroinitializer\n" +
		"@slot = global i8* getelementptr ([5000 x i8], [5000 x i8]* @buf, i32 0, i32 u0x1001)\n"
	if _, err := parseIR("cexpr.ll", cexpr); err != nil {
		t.Fatal(err)
	}
	for in, want := range map[string]string{
		`  %1 = getelementptr [9 x i8], [9 x i8]* @"a i32 u0x1000", i64 0, i64 u0x1000`: `  %1 = getelementptr [9 x i8], [9 x i8]* @"a i32 u0x1000", i64 0, i64 4096`,
		`  %2 = getelementptr i8, i8* %p, i16 u0x8000, i32 u0x7FFF0000`:                 `  %2 = getelementptr i8, i8* %p, i16 u0x8000, i32 2147418112`,
	} {
		if got := decimalHexInts(in); got != want {
			t.Fatalf("decimalHexInts(%q)\n = %q\nwant %q", in, got, want)
		}
	}
}
