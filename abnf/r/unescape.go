package r

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

// Unescape resolves backslash escapes (\n, \t, \x41, \u00e4, ...) inside the content of
// a Dquotetoken or Squotetoken string. It is a stripped down and slightly modified version
// of strconv.Unquote(), but without the surrounding quotation marks.
func Unescape(s string) (string, error) {
	// Is it trivial? Avoid allocation.
	if !strings.ContainsRune(s, '\\') {
		if utf8.ValidString(s) {
			return s, nil
		}
	}

	var runeTmp [utf8.UTFMax]byte
	buf := make([]byte, 0, 3*len(s)/2) // Try to avoid more allocations.
	for len(s) > 0 {
		// strconv.UnquoteChar with quote 0 rejects \" and \' - but both are fine in
		// grammar tokens (each quote style escapes its own delimiter), so they are
		// resolved here first.
		if len(s) >= 2 && s[0] == '\\' && (s[1] == '"' || s[1] == '\'') {
			buf = append(buf, s[1])
			s = s[2:]
			continue
		}
		c, multibyte, ss, err := strconv.UnquoteChar(s, 0)
		if err != nil {
			return "", err
		}
		s = ss
		if c < utf8.RuneSelf || !multibyte {
			buf = append(buf, byte(c))
		} else {
			n := utf8.EncodeRune(runeTmp[:], c)
			buf = append(buf, runeTmp[:n]...)
		}
	}
	return string(buf), nil
}
