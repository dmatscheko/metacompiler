// Package io: Go's io package WRITTEN IN GO, compiled with a program that imports
// "io" (core.importFile in both go grammars): the reader and writer interfaces and
// the helpers over them.
package io

import "errors"

var EOF = errors.New("EOF")
var ErrUnexpectedEOF = errors.New("unexpected EOF")
var ErrShortWrite = errors.New("short write")
var ErrShortBuffer = errors.New("short buffer")

type Reader interface {
	Read(p []byte) (n int, err error)
}

type Writer interface {
	Write(p []byte) (n int, err error)
}

type Closer interface {
	Close() error
}

type StringWriter interface {
	WriteString(s string) (n int, err error)
}

type ByteReader interface {
	ReadByte() (byte, error)
}

type ByteWriter interface {
	WriteByte(c byte) error
}

type RuneReader interface {
	ReadRune() (r rune, size int, err error)
}

func WriteString(w Writer, s string) (n int, err error) {
	if sw, ok := w.(StringWriter); ok {
		return sw.WriteString(s)
	}
	return w.Write([]byte(s))
}

// isEOF: the end of input, also as a reader outside this package reports it (the
// host's os.Stdin has no io.EOF of its own to answer).
func isEOF(err error) bool { return err == EOF || err != nil && err.Error() == "EOF" }

func ReadAll(r Reader) ([]byte, error) {
	b := make([]byte, 0, 512)
	buf := make([]byte, 512)
	for {
		n, err := r.Read(buf)
		b = append(b, buf[:n]...)
		if err != nil {
			if isEOF(err) {
				return b, nil
			}
			return b, err
		}
	}
}

func ReadFull(r Reader, buf []byte) (n int, err error) { return ReadAtLeast(r, buf, len(buf)) }

func ReadAtLeast(r Reader, buf []byte, min int) (n int, err error) {
	if len(buf) < min {
		return 0, ErrShortBuffer
	}
	for n < min && err == nil {
		var nn int
		nn, err = r.Read(buf[n:])
		n += nn
	}
	if n >= min {
		err = nil
	} else if n > 0 && isEOF(err) {
		err = ErrUnexpectedEOF
	}
	return n, err
}

func Copy(dst Writer, src Reader) (written int64, err error) {
	buf := make([]byte, 512)
	for {
		n, rerr := src.Read(buf)
		if n > 0 {
			w, werr := dst.Write(buf[:n])
			written += int64(w)
			if werr != nil {
				return written, werr
			}
			if w != n {
				return written, ErrShortWrite
			}
		}
		if rerr != nil {
			if isEOF(rerr) {
				return written, nil
			}
			return written, rerr
		}
	}
}

type discard struct{}

func (discard) Write(p []byte) (int, error)         { return len(p), nil }
func (discard) WriteString(s string) (int, error) { return len(s), nil }

var Discard Writer = discard{}

type multiWriter struct{ ws []Writer }

func (t *multiWriter) Write(p []byte) (n int, err error) {
	for _, w := range t.ws {
		n, err = w.Write(p)
		if err != nil {
			return n, err
		}
		if n != len(p) {
			return n, ErrShortWrite
		}
	}
	return len(p), nil
}

func MultiWriter(writers ...Writer) Writer { return &multiWriter{writers} }
