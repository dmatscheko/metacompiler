// Package bytes: Go's bytes package WRITTEN IN GO, compiled with a program that
// imports "bytes" (core.importFile in both go grammars). The functions that mirror
// package strings run strings' own on the converted text, so the two agree by
// construction; Buffer is the byte-slice writer and reader.
package bytes

import (
	"strings"
	"unicode/utf8"
)

func Contains(b, subslice []byte) bool        { return strings.Contains(string(b), string(subslice)) }
func ContainsAny(b []byte, chars string) bool { return strings.ContainsAny(string(b), chars) }
func ContainsRune(b []byte, r rune) bool      { return strings.ContainsRune(string(b), r) }
func Count(s, sep []byte) int                 { return strings.Count(string(s), string(sep)) }
func Equal(a, b []byte) bool                  { return string(a) == string(b) }
func Compare(a, b []byte) int                 { return strings.Compare(string(a), string(b)) }
func HasPrefix(s, prefix []byte) bool         { return strings.HasPrefix(string(s), string(prefix)) }
func HasSuffix(s, suffix []byte) bool         { return strings.HasSuffix(string(s), string(suffix)) }
func Index(s, sep []byte) int                 { return strings.Index(string(s), string(sep)) }
func IndexByte(b []byte, c byte) int          { return strings.IndexByte(string(b), c) }
func IndexRune(s []byte, r rune) int          { return strings.IndexRune(string(s), r) }
func IndexAny(s []byte, chars string) int     { return strings.IndexAny(string(s), chars) }
func LastIndex(s, sep []byte) int             { return strings.LastIndex(string(s), string(sep)) }
func EqualFold(s, t []byte) bool              { return strings.EqualFold(string(s), string(t)) }
func ToUpper(s []byte) []byte                 { return []byte(strings.ToUpper(string(s))) }
func ToLower(s []byte) []byte                 { return []byte(strings.ToLower(string(s))) }
func ToTitle(s []byte) []byte                 { return []byte(strings.ToTitle(string(s))) }
func Title(s []byte) []byte                   { return []byte(strings.Title(string(s))) }
func TrimSpace(s []byte) []byte               { return []byte(strings.TrimSpace(string(s))) }
func Trim(s []byte, cutset string) []byte     { return []byte(strings.Trim(string(s), cutset)) }
func TrimLeft(s []byte, cutset string) []byte {
	return []byte(strings.TrimLeft(string(s), cutset))
}
func TrimRight(s []byte, cutset string) []byte {
	return []byte(strings.TrimRight(string(s), cutset))
}
func TrimPrefix(s, prefix []byte) []byte {
	return []byte(strings.TrimPrefix(string(s), string(prefix)))
}
func TrimSuffix(s, suffix []byte) []byte {
	return []byte(strings.TrimSuffix(string(s), string(suffix)))
}
func Repeat(b []byte, count int) []byte { return []byte(strings.Repeat(string(b), count)) }
func ReplaceAll(s, old, new []byte) []byte {
	return []byte(strings.ReplaceAll(string(s), string(old), string(new)))
}
func Replace(s, old, new []byte, n int) []byte {
	return []byte(strings.Replace(string(s), string(old), string(new), n))
}
func Runes(s []byte) []rune { return []rune(string(s)) }

func toSlices(ss []string) [][]byte {
	out := make([][]byte, len(ss))
	for i, s := range ss {
		out[i] = []byte(s)
	}
	return out
}

func Split(s, sep []byte) [][]byte { return toSlices(strings.Split(string(s), string(sep))) }
func Fields(s []byte) [][]byte     { return toSlices(strings.Fields(string(s))) }

func Join(s [][]byte, sep []byte) []byte {
	ss := make([]string, len(s))
	for i, b := range s {
		ss[i] = string(b)
	}
	return []byte(strings.Join(ss, string(sep)))
}

func Clone(b []byte) []byte {
	if b == nil {
		return nil
	}
	return append([]byte{}, b...)
}

// ----- Buffer -----

type Buffer struct {
	buf []byte
	off int
}

func NewBuffer(buf []byte) *Buffer     { return &Buffer{buf: buf} }
func NewBufferString(s string) *Buffer { return &Buffer{buf: []byte(s)} }
func (b *Buffer) Bytes() []byte        { return b.buf[b.off:] }
func (b *Buffer) String() string {
	if b == nil {
		return "<nil>"
	}
	return string(b.buf[b.off:])
}
func (b *Buffer) Len() int { return len(b.buf) - b.off }
func (b *Buffer) Cap() int { return cap(b.buf) }
func (b *Buffer) Reset() {
	b.buf = b.buf[:0]
	b.off = 0
}
func (b *Buffer) Grow(n int) {
	if n < 0 {
		panic("bytes.Buffer.Grow: negative count")
	}
}
func (b *Buffer) Truncate(n int) {
	if n == 0 {
		b.Reset()
		return
	}
	if n < 0 || n > b.Len() {
		panic("bytes.Buffer: truncation out of range")
	}
	b.buf = b.buf[:b.off+n]
}
func (b *Buffer) Write(p []byte) (int, error) {
	b.buf = append(b.buf, p...)
	return len(p), nil
}
func (b *Buffer) WriteString(s string) (int, error) {
	b.buf = append(b.buf, s...)
	return len(s), nil
}
func (b *Buffer) WriteByte(c byte) error {
	b.buf = append(b.buf, c)
	return nil
}
func (b *Buffer) WriteRune(r rune) (int, error) {
	n := len(b.buf)
	b.buf = utf8.AppendRune(b.buf, r)
	return len(b.buf) - n, nil
}
func (b *Buffer) ReadByte() (byte, error) {
	if b.off >= len(b.buf) {
		b.Reset()
		return 0, errEOF
	}
	c := b.buf[b.off]
	b.off++
	return c, nil
}
func (b *Buffer) ReadRune() (rune, int, error) {
	if b.off >= len(b.buf) {
		b.Reset()
		return 0, 0, errEOF
	}
	r, n := utf8.DecodeRune(b.buf[b.off:])
	b.off += n
	return r, n, nil
}
func (b *Buffer) ReadString(delim byte) (string, error) {
	i := IndexByte(b.buf[b.off:], delim)
	end := len(b.buf)
	var err error
	if i < 0 {
		err = errEOF
	} else {
		end = b.off + i + 1
	}
	s := string(b.buf[b.off:end])
	b.off = end
	return s, err
}
func (b *Buffer) Next(n int) []byte {
	m := b.Len()
	if n > m {
		n = m
	}
	data := b.buf[b.off : b.off+n]
	b.off += n
	return data
}

// io.EOF's text; the io package is not one this project provides.
var errEOF = eofError{}

type eofError struct{}

func (eofError) Error() string { return "EOF" }
