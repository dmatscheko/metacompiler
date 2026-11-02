package abnf

import (
	"strings"
	"testing"
)

// TestRubyGcbTablesAgree pins String#grapheme_clusters across the three ruby
// engines: rbsGcbTable in languages/lib/ruby-str.js is the table the interpreter
// half and layer 2 share, and rubyGcbTable in jsrtrubygcb.go is the Go twin's
// copy. Each engine reads only its own, so nothing else would see them drift.
func TestRubyGcbTablesAgree(t *testing.T) {
	src := readRepoFile(t, "../languages/lib/ruby-str.js")
	head := "        rbsGcbTable = \"\"\n"
	start := strings.Index(src, head)
	if start < 0 {
		t.Fatal("rbsGcbTable not found in ruby-str.js - the parse broke, not the code")
	}
	var js strings.Builder
	for _, line := range strings.Split(src[start+len(head):], "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "+ \"") || !strings.HasSuffix(line, "\"") {
			break
		}
		js.WriteString(line[3 : len(line)-1])
	}
	if js.Len() < 1000 {
		t.Fatalf("parsed only %d chars of rbsGcbTable - the parse broke, not the code", js.Len())
	}
	if js.String() != rubyGcbTable {
		t.Errorf("rbsGcbTable (%d chars) and rubyGcbTable (%d chars) differ", js.Len(), len(rubyGcbTable))
	}
}
