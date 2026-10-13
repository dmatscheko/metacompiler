package abnf

import (
	"regexp"
	"strings"
	"testing"
)

// TestPyPrintableTablesAgree pins the Go twin's run tables (printable,
// XID_Start, XID_Continue) to the ones languages/lib/python-printable.js gives
// the interpreter half and layer 2. Each engine reads only its own copy, so
// nothing else would see them drift apart.
func TestPyPrintableTablesAgree(t *testing.T) {
	src := readRepoFile(t, "../languages/lib/python-printable.js")
	for _, tb := range []struct{ fn, goT string }{
		{"pyPrintRuns", pyPrintRuns}, {"pyXidStartRuns", pyXidStartRuns}, {"pyXidContRuns", pyXidContRuns},
	} {
		head := "function " + tb.fn + "() {\n    return "
		start := strings.Index(src, head)
		end := -1
		if start >= 0 {
			end = strings.Index(src[start:], "\n}")
		}
		if end < 0 {
			t.Fatalf("%s not found in python-printable.js - the parse broke, not the code", tb.fn)
		}
		js := ""
		for _, m := range regexp.MustCompile(`"([^"]*)"`).FindAllStringSubmatch(src[start:start+end], -1) {
			js += m[1]
		}
		if js != tb.goT {
			t.Errorf("python-printable.js's %s (%d chars) differs from the Go twin's (%d chars)",
				tb.fn, len(js), len(tb.goT))
		}
	}
	for _, c := range []struct {
		cp   rune
		want bool
	}{{' ', true}, {0xa0, false}, {0xad, false}, {0xa1, true}, {0x378, false},
		{0xd7ff, false}, {0xe000, false}, {0x1f600, true}, {0x10ffff, false}} {
		if got := pyIsPrintable(c.cp); got != c.want {
			t.Errorf("pyIsPrintable(U+%04X) = %v, want %v", c.cp, got, c.want)
		}
	}
	for _, c := range []struct {
		cp          rune
		start, cont bool
	}{{'_', true, true}, {'1', false, true}, {0xe9, true, true}, {0x1f600, false, false},
		{0xb7, false, true}, {0x301, false, true}, {0x2167, true, true}, {0x2081, false, false}, {0x200d, false, true}} {
		if s, k := pyIsXidStart(c.cp), pyIsXidCont(c.cp); s != c.start || k != c.cont {
			t.Errorf("U+%04X: XID_Start %v XID_Continue %v, want %v %v", c.cp, s, k, c.start, c.cont)
		}
	}
}
