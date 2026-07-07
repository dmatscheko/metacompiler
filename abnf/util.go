package abnf

import (
	"fmt"
	"strconv"
	"strings"

	"14.gy/mec/abnf/r"
)

// times returns the string s repeated n times (an empty string for n <= 0).
func times(s string, n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat(s, n)
}

// Shorten limits very long (error) messages to a readable size by cutting out the middle.
func Shorten(s string) string {
	const maxLen = 2000
	if len(s) > maxLen {
		midpos := maxLen/2 - 4
		s = s[:midpos] + "\n[...]\n" + s[len(s)-midpos:]
	}
	return s
}

func SprintRule(rule *r.Rule) string {
	return rule.ToString()
}

// LinePosFromStrPos converts a byte position inside data into a human readable
// line and column description (both 1-based). A position at the very end of the
// data is reported as EOF, everything outside of the data is reported as invalid.
func LinePosFromStrPos(data string, pos int) string {
	if pos < 0 || pos > len(data) {
		return "position outside of the input (char pos " + strconv.Itoa(pos) + ")"
	}
	line := 1
	column := 1
	for i, ch := range data {
		if i >= pos {
			return fmt.Sprintf("ln %d, col %d", line, column)
		}
		if ch == '\n' {
			line++
			column = 1
		} else if ch != '\r' { // A carriage return does not move the column.
			column++
		}
	}
	return fmt.Sprintf("ln %d, col %d (EOF)", line, column)
}
