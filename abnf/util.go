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

// errMsgMaxLen bounds a very long (error) message: Shorten and ShortenColored both
// cut out the middle once the message exceeds this many characters.
const errMsgMaxLen = 2000

// Shorten limits very long (error) messages to a readable size by cutting out the middle.
func Shorten(s string) string {
	if len(s) > errMsgMaxLen {
		midpos := errMsgMaxLen/2 - 4
		s = s[:midpos] + "\n[...]\n" + s[len(s)-midpos:]
	}
	return s
}

// ShortenColored is Shorten for a string that carries ANSI color escapes: it counts
// only visible characters, never cuts inside an escape sequence, and resets the color
// at the cut and before the tail so no color bleeds across the [...] gap or beyond.
func ShortenColored(s string) string {
	// atoms: each is one whole ANSI escape (ESC '[' ... final byte) or a single byte.
	var atoms []string
	visible := 0
	for i := 0; i < len(s); {
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && (s[j] < '@' || s[j] > '~') { // CSI ends on 0x40..0x7e.
				j++
			}
			if j < len(s) {
				j++ // Include the final byte (e.g. 'm').
			}
			atoms = append(atoms, s[i:j])
			i = j
			continue
		}
		atoms = append(atoms, s[i:i+1])
		visible++
		i++
	}
	if visible <= errMsgMaxLen {
		return s
	}
	isEsc := func(a string) bool { return len(a) > 1 && a[0] == 0x1b }
	budget := errMsgMaxLen/2 - 4

	var head strings.Builder
	hv := 0
	k := 0
	for ; k < len(atoms) && hv < budget; k++ {
		head.WriteString(atoms[k])
		if !isEsc(atoms[k]) {
			hv++
		}
	}
	head.WriteString("\x1b[0m") // Close any color left open by the cut.

	tv := 0
	t := len(atoms)
	for ; t > 0 && tv < budget; t-- {
		if !isEsc(atoms[t-1]) {
			tv++
		}
	}
	var tail strings.Builder
	tail.WriteString("\x1b[0m") // Clear any color the tail would otherwise inherit.
	for ; t < len(atoms); t++ {
		tail.WriteString(atoms[t])
	}
	return head.String() + "\n[...]\n" + tail.String()
}

// wrapScriptError builds the panic message for a tag or :script whose JS code threw:
// the original error, a short form of the rule it ran in, and the offending code. Every
// tag/script error site (goja and frozen, parser and compiler) shares this one wording.
func wrapScriptError(err interface{}, ruleText, code string) string {
	return fmt.Sprintf("%s\nError was in %s, Code: '%s'", err, ruleText, code)
}

func SprintRule(rule *r.Rule) string {
	return rule.ToString()
}

// lineCol returns the 1-based line and column of byte position pos in data, and
// whether pos sits at the very end (EOF). Columns count runes; a '\r' does not
// advance the column.
func lineCol(data string, pos int) (line, column int, eof bool) {
	line, column = 1, 1
	for i, ch := range data {
		if i >= pos {
			return line, column, false
		}
		if ch == '\n' {
			line++
			column = 1
		} else if ch != '\r' {
			column++
		}
	}
	return line, column, true
}

// LinePosFromStrPos converts a byte position inside data into a human readable
// line and column description (both 1-based). A position at the very end of the
// data is reported as EOF, everything outside of the data is reported as invalid.
func LinePosFromStrPos(data string, pos int) string {
	if pos < 0 || pos > len(data) {
		return "char pos " + strconv.Itoa(pos) + ", outside the input"
	}
	line, column, eof := lineCol(data, pos)
	if eof {
		return fmt.Sprintf("ln %d, col %d (EOF)", line, column)
	}
	return fmt.Sprintf("ln %d, col %d", line, column)
}

// FileLinePos formats a source position as a clickable "file:line:col" (with a
// trailing " (EOF)" when pos is at the end of data). An out-of-range position
// falls back to the file name plus the raw byte offset.
func FileLinePos(fileName, data string, pos int) string {
	if pos < 0 || pos > len(data) {
		return fmt.Sprintf("%s (char pos %d, outside the input)", fileName, pos)
	}
	line, column, eof := lineCol(data, pos)
	if eof {
		return fmt.Sprintf("%s:%d:%d (EOF)", fileName, line, column)
	}
	return fmt.Sprintf("%s:%d:%d", fileName, line, column)
}
