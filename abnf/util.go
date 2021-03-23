package abnf

import (
	"fmt"
	"strconv"

	"14.gy/mec/abnf/r"
)

func times(s string, n int) string {
	res := s
	for ; n > 0; n-- {
		res = res + s
	}
	return res
}

func Shorten(s string) string {
	const maxLen = 800
	if len(s) > maxLen {
		s = s[:maxLen-5] + " ..."
	}
	return s
}

func SprintRule(rule *r.Rule) string {
	return rule.ToString()
}

func LinePosFromStrPos(data string, pos int) string {
	//lines := strings.Split(data, "\n")
	// chars := 0
	if pos >= 0 && pos < len(data) {
		line := 1
		column := 0
		for i, ch := range data {
			if ch != '\r' {
				column++
			}
			if i >= pos {
				return fmt.Sprintf("ln %d, col %d", line, column)
			}
			if ch == '\n' {
				line++
				column = 0
			}
		}
	}
	return "position outside of EBNF (char pos " + strconv.Itoa(pos) + ")"
}
