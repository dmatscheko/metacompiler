// Package bufio: Go's bufio package WRITTEN IN GO, compiled with a program that
// imports "bufio" (core.importFile in both go grammars): a Scanner over any
// io.Reader with the four split functions, and the buffered Reader and Writer.
package bufio

import (
	"errors"
	"io"
	"unicode/utf8"
)

const MaxScanTokenSize = 64 * 1024

var ErrTooLong = errors.New("bufio.Scanner: token too long")
var ErrNegativeAdvance = errors.New("bufio.Scanner: SplitFunc returns negative advance count")
var ErrAdvanceTooFar = errors.New("bufio.Scanner: SplitFunc returns advance count beyond input")
var ErrFinalToken = errors.New("final token")

type SplitFunc func(data []byte, atEOF bool) (advance int, token []byte, err error)

type Scanner struct {
	r     io.Reader
	split SplitFunc
	buf   []byte // read and not yet consumed
	tok   []byte
	err   error
	eof   bool
	done  bool
}

func NewScanner(r io.Reader) *Scanner { return &Scanner{r: r, split: ScanLines} }

func (s *Scanner) Split(split SplitFunc) { s.split = split }
func (s *Scanner) Buffer(buf []byte, max int) {}
func (s *Scanner) Text() string           { return string(s.tok) }
func (s *Scanner) Bytes() []byte          { return s.tok }
func (s *Scanner) Err() error             { return s.err }

// The next token: the split function over what is buffered, reading on while it
// asks for more; at the end of the input it is asked once more with atEOF set.
func (s *Scanner) Scan() bool {
	if s.done {
		return false
	}
	for {
		if len(s.buf) > 0 || s.eof {
			adv, tok, err := s.split(s.buf, s.eof)
			if err != nil {
				s.done = true
				if err == ErrFinalToken {
					s.tok = tok
					return tok != nil
				}
				s.err = err
				return false
			}
			if adv < 0 {
				s.err = ErrNegativeAdvance
				s.done = true
				return false
			}
			if adv > len(s.buf) {
				s.err = ErrAdvanceTooFar
				s.done = true
				return false
			}
			s.buf = s.buf[adv:]
			if tok != nil {
				s.tok = tok
				return true
			}
			if adv > 0 {
				continue
			}
			if s.eof {
				s.done = true
				s.tok = nil
				return false
			}
		}
		chunk := make([]byte, 4096)
		n, err := s.r.Read(chunk)
		s.buf = append(s.buf, chunk[:n]...)
		if err != nil {
			if err != io.EOF && err.Error() != "EOF" {
				s.err = err
			}
			s.eof = true
		}
	}
}

func indexByte(b []byte, c byte) int {
	for i := 0; i < len(b); i++ {
		if b[i] == c {
			return i
		}
	}
	return -1
}

func dropCR(data []byte) []byte {
	if len(data) > 0 && data[len(data)-1] == '\r' {
		return data[0 : len(data)-1]
	}
	return data
}

func ScanLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := indexByte(data, '\n'); i >= 0 {
		return i + 1, dropCR(data[0:i]), nil
	}
	if atEOF {
		return len(data), dropCR(data), nil
	}
	return 0, nil, nil
}

func ScanBytes(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	return 1, data[0:1], nil
}

func ScanRunes(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if data[0] < utf8.RuneSelf {
		return 1, data[0:1], nil
	}
	if !atEOF && !utf8.FullRune(data) {
		return 0, nil, nil
	}
	_, width := utf8.DecodeRune(data)
	if width > 1 {
		return width, data[0:width], nil
	}
	return 1, []byte(string(utf8.RuneError)), nil
}

func isSpace(r rune) bool {
	switch r {
	case ' ', '\t', '\n', '\v', '\f', '\r', 0x85, 0xA0, 0x1680, 0x2028, 0x2029, 0x202f, 0x205f, 0x3000:
		return true
	}
	return r >= 0x2000 && r <= 0x200a
}

func ScanWords(data []byte, atEOF bool) (advance int, token []byte, err error) {
	start := 0
	for start < len(data) {
		r, width := utf8.DecodeRune(data[start:])
		if !isSpace(r) {
			break
		}
		start += width
	}
	for i := start; i < len(data); {
		r, width := utf8.DecodeRune(data[i:])
		if isSpace(r) {
			return i + width, data[start:i], nil
		}
		i += width
	}
	if atEOF && len(data) > start {
		return len(data), data[start:], nil
	}
	return start, nil, nil
}

// ----- Reader -----

type Reader struct {
	rd  io.Reader
	buf []byte
	err error
}

func NewReader(rd io.Reader) *Reader                { return &Reader{rd: rd} }
func NewReaderSize(rd io.Reader, size int) *Reader { return &Reader{rd: rd} }

// Reads one more chunk; false once the input is done and nothing came.
func (b *Reader) fill() bool {
	if b.err != nil {
		return false
	}
	chunk := make([]byte, 4096)
	n, err := b.rd.Read(chunk)
	b.buf = append(b.buf, chunk[:n]...)
	if err != nil {
		if err.Error() == "EOF" {
			err = io.EOF
		}
		b.err = err
	}
	return n > 0 || err == nil
}

func (b *Reader) takeErr() error {
	err := b.err
	if len(b.buf) == 0 {
		b.err = nil
	}
	return err
}

func (b *Reader) ReadBytes(delim byte) ([]byte, error) {
	for {
		if i := indexByte(b.buf, delim); i >= 0 {
			line := b.buf[:i+1]
			b.buf = b.buf[i+1:]
			return line, nil
		}
		if !b.fill() {
			line := b.buf
			b.buf = nil
			return line, b.takeErr()
		}
	}
}

func (b *Reader) ReadString(delim byte) (string, error) {
	line, err := b.ReadBytes(delim)
	return string(line), err
}

func (b *Reader) ReadLine() (line []byte, isPrefix bool, err error) {
	l, err := b.ReadBytes('\n')
	if len(l) == 0 && err != nil {
		return nil, false, err
	}
	if len(l) > 0 && l[len(l)-1] == '\n' {
		l = dropCR(l[:len(l)-1])
	}
	return l, false, nil
}

func (b *Reader) ReadByte() (byte, error) {
	for len(b.buf) == 0 {
		if !b.fill() {
			return 0, b.takeErr()
		}
	}
	c := b.buf[0]
	b.buf = b.buf[1:]
	return c, nil
}

func (b *Reader) ReadRune() (r rune, size int, err error) {
	for len(b.buf) < utf8.UTFMax && !utf8.FullRune(b.buf) && b.fill() {
	}
	if len(b.buf) == 0 {
		return 0, 0, b.takeErr()
	}
	r, size = utf8.DecodeRune(b.buf)
	b.buf = b.buf[size:]
	return r, size, nil
}

func (b *Reader) Read(p []byte) (n int, err error) {
	if len(b.buf) == 0 && !b.fill() {
		return 0, b.takeErr()
	}
	for n < len(p) && n < len(b.buf) {
		p[n] = b.buf[n]
		n++
	}
	b.buf = b.buf[n:]
	return n, nil
}

func (b *Reader) Buffered() int { return len(b.buf) }

// ----- Writer -----

type Writer struct {
	wr  io.Writer
	buf []byte
	err error
}

func NewWriter(w io.Writer) *Writer                { return &Writer{wr: w} }
func NewWriterSize(w io.Writer, size int) *Writer { return &Writer{wr: w} }

func (b *Writer) Write(p []byte) (int, error) {
	b.buf = append(b.buf, p...)
	return len(p), nil
}

func (b *Writer) WriteString(s string) (int, error) {
	b.buf = append(b.buf, s...)
	return len(s), nil
}

func (b *Writer) WriteByte(c byte) error {
	b.buf = append(b.buf, c)
	return nil
}

func (b *Writer) WriteRune(r rune) (int, error) {
	s := string(r)
	b.buf = append(b.buf, s...)
	return len(s), nil
}

func (b *Writer) Flush() error {
	if b.err != nil {
		return b.err
	}
	if len(b.buf) == 0 {
		return nil
	}
	_, err := b.wr.Write(b.buf)
	b.buf = b.buf[:0]
	b.err = err
	return err
}

func (b *Writer) Buffered() int  { return len(b.buf) }
func (b *Writer) Available() int { return 4096 - len(b.buf) }
func (b *Writer) Size() int      { return 4096 }

type ReadWriter struct {
	*Reader
	*Writer
}

func NewReadWriter(r *Reader, w *Writer) *ReadWriter { return &ReadWriter{r, w} }
