// fmt writes the BYTES a Go string holds, valid UTF-8 or not: three bytes that
// spell a surrogate in UTF-8 (ED A0 80) are bytes like any others, written as
// they are and never joined with the next three into an astral character.
// Nothing inside a program can observe its own output, so this file is checked
// from outside: tests/native-full.sh compares the native binary's stdout with
// tests/go-bytes.expected, which is what `go run` writes, and --cross holds the
// two halves to the same bytes.
package main

import (
	"fmt"
	"strings"
)

func main() {
	parts := []string{"\xed\xa0\x80", "\xed\xb0\x80", "\xff", "😀"}
	s := parts[0] + parts[1]
	fmt.Println(s, len(s))
	fmt.Println(parts[0])
	fmt.Print(parts[2], parts[3], "\n")
	fmt.Printf("%s|%-4s|%.1s|😀%d\n", s, parts[0], parts[0], 1)
	var b strings.Builder
	b.WriteString(parts[0])
	b.WriteString(parts[1])
	fmt.Println(b.String() == s, fmt.Sprint(parts[0], parts[1]) == s)
	fmt.Println(strings.Join(parts[:2], ""), strings.Split(parts[0], ""))
}
