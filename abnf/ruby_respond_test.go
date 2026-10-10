package abnf

import (
	"sort"
	"strings"
	"testing"
)

// TestRubyRespondTablesAgree pins Object#respond_to? for builtin values across
// the three ruby engines: languages/lib/ruby-respond.js is the table the
// interpreter half and layer 2 share, and rubyRespondNames in jsrtruby.go is the
// Go twin's copy. Nothing else can see the two drift apart - each engine reads
// only its own copy, so every gate would stay green while respond_to? answered
// differently under llvm.Run.
func TestRubyRespondTablesAgree(t *testing.T) {
	src := readRepoFile(t, "../languages/lib/ruby-respond.js")
	head := "function rbRespondRows() {\n    return ["
	start := strings.Index(src, head)
	if start < 0 {
		t.Fatal("rbRespondRows not found in ruby-respond.js - the parse broke, not the code")
	}
	js := parseRespondRows(src[start+len(head):])
	if len(js) < 10 {
		t.Fatalf("parsed only %d rows from ruby-respond.js - the parse broke, not the code", len(js))
	}
	classes := map[string]bool{}
	for cls := range js {
		classes[cls] = true
	}
	for cls := range rubyRespondNames {
		classes[cls] = true
	}
	for cls := range classes {
		a := respondSetOf(js[cls])
		b := respondSetOf(rubyRespondNames[cls])
		onlyJS, onlyGo := []string{}, []string{}
		for n := range a {
			if !b[n] {
				onlyJS = append(onlyJS, n)
			}
		}
		for n := range b {
			if !a[n] {
				onlyGo = append(onlyGo, n)
			}
		}
		sort.Strings(onlyJS)
		sort.Strings(onlyGo)
		if len(onlyJS)+len(onlyGo) > 0 {
			t.Errorf("respond_to? row %q differs: only in ruby-respond.js %v, only in rubyRespondNames %v",
				cls, onlyJS, onlyGo)
		}
	}
}

// parseRespondRows reads `["Class", "name", ...], ...]` - the rows of the table
// up to its closing bracket - into class -> names.
func parseRespondRows(s string) map[string][]string {
	out := map[string][]string{}
	var row []string
	inRow := false
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case c == '"':
			j := strings.IndexByte(s[i+1:], '"')
			if j < 0 {
				return out
			}
			if inRow {
				row = append(row, s[i+1:i+1+j])
			}
			i += j + 1
		case c == '[':
			inRow = true
			row = nil
		case c == ']':
			if !inRow {
				return out
			}
			if len(row) > 0 {
				out[row[0]] = row[1:]
			}
			inRow = false
		}
	}
	return out
}

func respondSetOf(xs []string) map[string]bool {
	out := map[string]bool{}
	for _, x := range xs {
		out[x] = true
	}
	return out
}
