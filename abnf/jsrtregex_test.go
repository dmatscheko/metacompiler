package abnf

import (
	"strings"
	"testing"
)

// TestRxEreMemo: under e a search that would backtrack exponentially finishes
// within the step cap (rxMemoOps), and every search gives the answer bash 5.3
// gives, the leftmost-longest match.
func TestRxEreMemo(t *testing.T) {
	a := strings.Repeat("a", 200)
	cases := []struct {
		pat, text string
		want      int // the match's end, or -1 for none
	}{
		{"^(a+)+$", a + "b", -1},
		{"^(a|aa)*c", a + "b", -1},
		{"(a?){30}a{30}x", a + "b", -1},
		{"^((a?|b?){20})c", a + "b", -1},
		{"^(a+)+b", a + "b", 201},
		{"^(a*)*$", a, 200},
		// the longest match at the leftmost start, whatever the alternation order
		{"(a|ab)", "ab", 2},
		{"a|ab|abc", "xabcx", 4},
		{"(ab|a)(bab)?", "abab", 4},
	}
	for _, c := range cases {
		re := rxCompile(c.pat, "e")
		if !re.ok {
			t.Fatalf("%s: %s", c.pat, re.err)
		}
		m, ok := re.search([]rune(c.text), 0)
		if !ok {
			t.Errorf("%s: step limit exceeded", c.pat)
			continue
		}
		got := -1
		if m != nil {
			got = m.end
		}
		if got != c.want {
			t.Errorf("%s: end %d, want %d", c.pat, got, c.want)
		}
	}
}
