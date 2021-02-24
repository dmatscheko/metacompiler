package ebnf

import (
	"fmt"
	"strings"
)

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

// Adjusts Go's normal printing of slices to look more like Phix output.
func pprint(ob object, header string) {
	fmt.Printf("\n%s:\n", header)
	pp := fmt.Sprintf("%q", ob)
	pp = strings.Replace(pp, "[", "{", -1)
	pp = strings.Replace(pp, "]", "}", -1)
	pp = strings.Replace(pp, " ", ", ", -1)

	var rs []string = make([]string, 512)
	for i := 0; i < 256; i++ {
		pos := 2 * i
		rs[pos] = fmt.Sprintf("\\x%02x", i)
		rs[pos+1] = fmt.Sprintf("<%d>", i)
	}
	r := strings.NewReplacer(rs...)
	// replace all pairs
	pp = r.Replace(pp)

	fmt.Println(pp)
}
