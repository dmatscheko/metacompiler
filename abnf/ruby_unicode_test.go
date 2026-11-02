package abnf

import (
	"strings"
	"testing"
)

// TestRubyUnicodeTablesAgree pins String#grapheme_clusters and
// #unicode_normalize across the three ruby engines: languages/lib/ruby-str.js
// holds the tables the interpreter half and layer 2 share, and
// jsrtrubygcb.go / jsrtrubynorm.go the Go twin's copies. Each engine reads only
// its own, so nothing else would see them drift apart.
func TestRubyUnicodeTablesAgree(t *testing.T) {
	src := readRepoFile(t, "../languages/lib/ruby-str.js")
	for _, c := range []struct{ head, goName, goTab string }{
		{"        rbsGcbTable = \"\"\n", "rubyGcbTable", rubyGcbTable},
		{"    var ccc = \"\"\n", "rubyNormCccTable", rubyNormCccTable},
		{"    var nfd = \"\"\n", "rubyNormNfdTable", rubyNormNfdTable},
		{"    var nfkd = \"\"\n", "rubyNormNfkdTable", rubyNormNfkdTable},
		{"    var nfc = \"\"\n", "rubyNormNfcTable", rubyNormNfcTable},
	} {
		js := jsConcatAfter(t, src, c.head)
		if js != c.goTab {
			t.Errorf("the ruby-str.js table after %q (%d chars) and %s (%d chars) differ",
				strings.TrimSpace(c.head), len(js), c.goName, len(c.goTab))
		}
	}
}

// jsConcatAfter is the string the `+ "..."' lines after the line head of src
// concatenate.
func jsConcatAfter(t *testing.T, src, head string) string {
	start := strings.Index(src, head)
	if start < 0 {
		t.Fatalf("%q not found in ruby-str.js - the parse broke, not the code", strings.TrimSpace(head))
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
		t.Fatalf("parsed only %d chars after %q - the parse broke, not the code", js.Len(), strings.TrimSpace(head))
	}
	return js.String()
}
