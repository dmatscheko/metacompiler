package abnf

import (
	"regexp"
	"strings"
	"testing"
)

// TestPyPrintableTablesAgree pins the Go twin's printable runs to the table
// languages/lib/python-printable.js gives the interpreter half and layer 2. Each
// engine reads only its own copy, so nothing else would see them drift apart.
func TestPyPrintableTablesAgree(t *testing.T) {
	src := readRepoFile(t, "../languages/lib/python-printable.js")
	head := "function pyPrintRuns() {\n    var t = "
	start := strings.Index(src, head)
	end := strings.Index(src, "\n    var at = 0")
	if start < 0 || end < start {
		t.Fatal("pyPrintRuns not found in python-printable.js - the parse broke, not the code")
	}
	js := ""
	for _, m := range regexp.MustCompile(`"([^"]*)"`).FindAllStringSubmatch(src[start:end], -1) {
		js += m[1]
	}
	if js != pyPrintRuns {
		t.Errorf("python-printable.js's pyPrintRuns (%d chars) differs from the Go twin's (%d chars)",
			len(js), len(pyPrintRuns))
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
}
