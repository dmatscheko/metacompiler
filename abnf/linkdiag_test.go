package abnf

import (
	"strings"
	"testing"
)

// TestDuplicateSymbolIsNotReportedAsUnresolved pins the classification the -exe
// link failure report depends on.
//
// The undefined-symbol report in buildExecutable is built by filtering the
// module's declared-but-undefined names by whether the linker text MENTIONS them,
// and a duplicate-symbol error mentions exactly those same names - a name the
// module declares as an extern is a name some input defines. So a build whose real
// problem was two runtime inputs defining the same function was announced as
// "N unresolved symbol(s) ... defined by neither the module nor any linked input",
// the precise opposite of the truth. duplicateSymbols is what separates the two,
// and it is asked FIRST.
//
// The two linker spellings below are the real ones: the ld64 text was captured
// from clang 17 on this machine (a duplicate `mec_helper` across two -rt inputs),
// the GNU ld text is that linker's documented wording.
func TestDuplicateSymbolIsNotReportedAsUnresolved(t *testing.T) {
	ld64 := "duplicate symbol '_mec_helper' in:\n" +
		"    /var/folders/T/rt2-6af1d3.o\n" +
		"    /var/folders/T/rt1-c7b552.o\n" +
		"ld: 1 duplicate symbols\n" +
		"clang: error: linker command failed with exit code 1 (use -v to see invocation)\n"
	gnu := "/usr/bin/ld: rt2.o: in function `mec_helper':\n" +
		"rt2.c:(.text+0x0): multiple definition of `mec_helper'; rt1.o:rt1.c:(.text+0x0): first defined here\n" +
		"clang: error: linker command failed with exit code 1\n"
	undef := "Undefined symbols for architecture arm64:\n" +
		"  \"_mec_absent\", referenced from:\n" +
		"      _main in mec-123.o\n" +
		"ld: symbol(s) not found for architecture arm64\n"
	gnuUndef := "/usr/bin/ld: mec-123.o: in function `main':\n" +
		"mec.c:(.text+0x10): undefined reference to `mec_absent'\n"

	for _, tc := range []struct {
		name string
		out  string
		want []string
	}{
		{"ld64 duplicate", ld64, []string{"mec_helper"}},
		{"gnu duplicate", gnu, []string{"mec_helper"}},
		{"ld64 undefined", undef, nil},
		{"gnu undefined", gnuUndef, nil},
		{"success", "", nil},
	} {
		got := duplicateSymbols(tc.out)
		if len(got) != len(tc.want) {
			t.Fatalf("%s: duplicateSymbols = %v, want %v", tc.name, got, tc.want)
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Fatalf("%s: duplicateSymbols = %v, want %v", tc.name, got, tc.want)
			}
		}
	}

	// The trap itself: the SAME name the duplicate text blames also passes the
	// mentionsSymbol filter the unresolved report uses, in both spellings. That is
	// why the duplicate case has to be decided before that report is reached, and
	// this assertion fails the day mentionsSymbol is "fixed" instead.
	if !mentionsSymbol(ld64, "mec_helper") || !mentionsSymbol(gnu, "mec_helper") {
		t.Fatal("mentionsSymbol no longer matches a duplicate-symbol diagnostic;" +
			" the ordering in buildExecutable was justified by exactly that overlap")
	}

	// Several duplicates come back sorted and deduplicated, and no object-file
	// path ever leaks into the report (this path is byte-compared by the matrix).
	multi := "duplicate symbol '_zz' in:\n    a.o\n    b.o\n" +
		"duplicate symbol '_aa' in:\n    a.o\n    b.o\n" +
		"duplicate symbol '_zz' in:\n    a.o\n    c.o\n"
	got := duplicateSymbols(multi)
	if len(got) != 2 || got[0] != "aa" || got[1] != "zz" {
		t.Fatalf("duplicateSymbols(multi) = %v, want [aa zz]", got)
	}
	for _, n := range got {
		if strings.ContainsAny(n, "./ ") {
			t.Fatalf("a file path leaked into the symbol name: %q", n)
		}
	}
}
