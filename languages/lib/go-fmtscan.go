// Package fmt's scanning functions WRITTEN IN GO: Scan, Scanln, Scanf and their
// Sscan* and Fscan* forms, after the standard library's fmt/scan.go. Compiled
// with a program that calls one of them (goUsesFmtScan in lib/go-constfold.js)
// and bound into the fmt object both grammars provide. An operand is stored
// through the run-time type primitives both grammars bind (goRf*, see
// lib/go-json.go), by the kind of the value its pointer holds. Not provided:
// an operand with its own Scan method, the %U verb, and a decimal mantissa
// with a binary exponent ("1.5p3").
package scan

import (
	"errors"
	"io"
	"os"
	"strconv"
	"unicode/utf8"
)

const eof = -1
const hugeWid = 1 << 30

// stringRunes reads a string a rune at a time; Sscan* scans one.
type stringRunes struct {
	s    string
	i    int
	last int
}

func (r *stringRunes) ReadRune() (rune, int, error) {
	if r.i >= len(r.s) {
		r.last = 0
		return 0, 0, io.EOF
	}
	c, w := utf8.DecodeRuneInString(r.s[r.i:])
	r.i += w
	r.last = w
	return c, w, nil
}

func (r *stringRunes) UnreadRune() error {
	r.i -= r.last
	r.last = 0
	return nil
}

// byteRunes reads an io.Reader that is not an io.RuneScanner a byte at a time,
// so that nothing past what the scan consumes is read from it.
type byteRunes struct {
	r       io.Reader
	pending rune
	has     bool
	last    rune
}

func (r *byteRunes) readByte() (byte, error) {
	var b [1]byte
	n, err := r.r.Read(b[:])
	if n == 1 {
		return b[0], nil
	}
	if err == nil {
		err = io.EOF
	}
	return 0, err
}

func (r *byteRunes) ReadRune() (rune, int, error) {
	if r.has {
		r.has = false
		return r.pending, utf8.RuneLen(r.pending), nil
	}
	b, err := r.readByte()
	if err != nil {
		return 0, 0, err
	}
	if b < utf8.RuneSelf {
		r.last = rune(b)
		return r.last, 1, nil
	}
	buf := []byte{b}
	for !utf8.FullRune(buf) {
		c, err := r.readByte()
		if err != nil {
			break
		}
		buf = append(buf, c)
	}
	c, w := utf8.DecodeRune(buf)
	r.last = c
	return c, w, nil
}

func (r *byteRunes) UnreadRune() error {
	r.pending = r.last
	r.has = true
	return nil
}

// scanError carries an error out of a scan; errorHandler recovers it.
type scanError struct{ err error }

type ss struct {
	rs        io.RuneScanner
	buf       []byte
	count     int
	atEOF     bool
	nlIsSpace bool
	nlIsEnd   bool
	argLimit  int
	limit     int
	maxWid    int
}

func newScanState(rs io.RuneScanner, nlIsSpace, nlIsEnd bool) *ss {
	return &ss{rs: rs, nlIsSpace: nlIsSpace, nlIsEnd: nlIsEnd, argLimit: hugeWid, limit: hugeWid, maxWid: hugeWid}
}

func runesOf(r io.Reader) io.RuneScanner {
	if rs, ok := r.(io.RuneScanner); ok {
		return rs
	}
	return &byteRunes{r: r}
}

func Scan(a ...any) (n int, err error)   { return Fscan(os.Stdin, a...) }
func Scanln(a ...any) (n int, err error) { return Fscanln(os.Stdin, a...) }
func Scanf(format string, a ...any) (n int, err error) {
	return Fscanf(os.Stdin, format, a...)
}

func Sscan(str string, a ...any) (n int, err error) {
	return newScanState(&stringRunes{s: str}, true, false).doScan(a)
}
func Sscanln(str string, a ...any) (n int, err error) {
	return newScanState(&stringRunes{s: str}, false, true).doScan(a)
}
func Sscanf(str string, format string, a ...any) (n int, err error) {
	return newScanState(&stringRunes{s: str}, false, false).doScanf(format, a)
}

func Fscan(r io.Reader, a ...any) (n int, err error) {
	return newScanState(runesOf(r), true, false).doScan(a)
}
func Fscanln(r io.Reader, a ...any) (n int, err error) {
	return newScanState(runesOf(r), false, true).doScan(a)
}
func Fscanf(r io.Reader, format string, a ...any) (n int, err error) {
	return newScanState(runesOf(r), false, false).doScanf(format, a)
}

func (s *ss) ReadRune() (r rune, size int, err error) {
	if s.atEOF || s.count >= s.argLimit {
		err = io.EOF
		return
	}
	r, size, err = s.rs.ReadRune()
	if err == nil {
		s.count++
		if s.nlIsEnd && r == '\n' {
			s.atEOF = true
		}
	} else if err == io.EOF {
		s.atEOF = true
	}
	return
}

func (s *ss) getRune() rune {
	r, _, err := s.ReadRune()
	if err != nil {
		if err == io.EOF {
			return eof
		}
		s.error(err)
	}
	return r
}

func (s *ss) mustReadRune() rune {
	r := s.getRune()
	if r == eof {
		s.error(io.ErrUnexpectedEOF)
	}
	return r
}

func (s *ss) UnreadRune() {
	s.rs.UnreadRune()
	s.atEOF = false
	s.count--
}

func (s *ss) error(err error)          { panic(scanError{err}) }
func (s *ss) errorString(err string)   { panic(scanError{errors.New(err)}) }
func (s *ss) notEOF() {
	if r := s.getRune(); r == eof {
		panic(io.EOF)
	}
	s.UnreadRune()
}

func isSpace(r rune) bool {
	switch r {
	case ' ', '\t', '\n', '\v', '\f', '\r', 0x85, 0xA0, 0x1680, 0x2028, 0x2029, 0x202F, 0x205F, 0x3000:
		return true
	}
	return r >= 0x2000 && r <= 0x200A
}

func (s *ss) SkipSpace() {
	for {
		r := s.getRune()
		if r == eof {
			return
		}
		if r == '\r' && s.peek("\n") {
			continue
		}
		if r == '\n' {
			if s.nlIsSpace {
				continue
			}
			s.errorString("unexpected newline")
			return
		}
		if !isSpace(r) {
			s.UnreadRune()
			break
		}
	}
}

func indexRune(s string, r rune) int {
	for i, c := range s {
		if c == r {
			return i
		}
	}
	return -1
}

// consume reads the next rune into the token buffer if it is one of ok.
func (s *ss) consume(ok string, accept bool) bool {
	r := s.getRune()
	if r == eof {
		return false
	}
	if indexRune(ok, r) >= 0 {
		if accept {
			s.buf = utf8.AppendRune(s.buf, r)
		}
		return true
	}
	if accept {
		s.UnreadRune()
	}
	return false
}

func (s *ss) accept(ok string) bool { return s.consume(ok, true) }

func (s *ss) peek(ok string) bool {
	r := s.getRune()
	if r != eof {
		s.UnreadRune()
	}
	return indexRune(ok, r) >= 0
}

func (s *ss) okVerb(verb rune, okVerbs, typ string) bool {
	for _, v := range okVerbs {
		if v == verb {
			return true
		}
	}
	s.errorString("bad verb '%" + string(verb) + "' for " + typ)
	return false
}

func (s *ss) token() string {
	s.SkipSpace()
	for {
		r := s.getRune()
		if r == eof {
			break
		}
		if isSpace(r) {
			s.UnreadRune()
			break
		}
		s.buf = utf8.AppendRune(s.buf, r)
	}
	return string(s.buf)
}

const (
	binaryDigits      = "01"
	octalDigits       = "01234567"
	decimalDigits     = "0123456789"
	hexadecimalDigits = "0123456789aAbBcCdDeEfF"
	sign              = "+-"
	period            = "."
	exponent          = "eEpP"
)

func (s *ss) getBase(verb rune) (int, string) {
	s.okVerb(verb, "bdoUxXv", "integer")
	switch verb {
	case 'b':
		return 2, binaryDigits
	case 'o':
		return 8, octalDigits
	case 'x', 'X', 'U':
		return 16, hexadecimalDigits
	}
	return 10, decimalDigits
}

func (s *ss) scanNumber(digits string, haveDigits bool) string {
	if !haveDigits {
		s.notEOF()
		if !s.accept(digits) {
			s.errorString("expected integer")
		}
	}
	for s.accept(digits) {
	}
	return string(s.buf)
}

func (s *ss) scanRune(bitSize int) int64 {
	s.notEOF()
	r := int64(s.getRune())
	n := uint(bitSize)
	x := (r << (64 - n)) >> (64 - n)
	if x != r {
		s.errorString("overflow on character value " + string(rune(r)))
	}
	return r
}

// scanBasePrefix reads 0b, 0o, 0x or a leading 0, the prefixes %v accepts.
func (s *ss) scanBasePrefix() (base int, digits string, zeroFound bool) {
	if !s.peek("0") {
		return 0, decimalDigits + "_", false
	}
	s.accept("0")
	switch {
	case s.peek("bB"):
		s.consume("bB", true)
		return 0, binaryDigits + "_", true
	case s.peek("oO"):
		s.consume("oO", true)
		return 0, octalDigits + "_", true
	case s.peek("xX"):
		s.consume("xX", true)
		return 0, hexadecimalDigits + "_", true
	}
	return 0, octalDigits + "_", true
}

func (s *ss) scanInt(verb rune, bitSize int) int64 {
	if verb == 'c' {
		return s.scanRune(bitSize)
	}
	s.SkipSpace()
	s.notEOF()
	base, digits := s.getBase(verb)
	haveDigits := false
	s.accept(sign)
	if verb == 'v' {
		base, digits, haveDigits = s.scanBasePrefix()
	}
	tok := s.scanNumber(digits, haveDigits)
	i, err := strconv.ParseInt(tok, base, 64)
	if err != nil {
		s.error(err)
	}
	n := uint(bitSize)
	x := (i << (64 - n)) >> (64 - n)
	if x != i {
		s.errorString("integer overflow on token " + tok)
	}
	return i
}

func (s *ss) scanUint(verb rune, bitSize int) uint64 {
	if verb == 'c' {
		return uint64(s.scanRune(bitSize))
	}
	s.SkipSpace()
	s.notEOF()
	base, digits := s.getBase(verb)
	haveDigits := false
	if verb == 'v' {
		base, digits, haveDigits = s.scanBasePrefix()
	}
	tok := s.scanNumber(digits, haveDigits)
	i, err := strconv.ParseUint(tok, base, 64)
	if err != nil {
		s.error(err)
	}
	n := uint(bitSize)
	x := (i << (64 - n)) >> (64 - n)
	if x != i {
		s.errorString("unsigned integer overflow on token " + tok)
	}
	return i
}

var boolError = errors.New("syntax error scanning boolean")

func (s *ss) scanBool(verb rune) bool {
	s.SkipSpace()
	s.notEOF()
	if !s.okVerb(verb, "tv", "boolean") {
		return false
	}
	switch s.getRune() {
	case '0':
		return false
	case '1':
		return true
	case 't', 'T':
		if s.accept("rR") && (!s.accept("uU") || !s.accept("eE")) {
			s.error(boolError)
		}
		return true
	case 'f', 'F':
		if s.accept("aA") && (!s.accept("lL") || !s.accept("sS") || !s.accept("eE")) {
			s.error(boolError)
		}
		return false
	}
	return false
}

func (s *ss) floatToken() string {
	s.buf = s.buf[:0]
	if s.accept("nN") && s.accept("aA") && s.accept("nN") {
		return string(s.buf)
	}
	s.accept(sign)
	if s.accept("iI") && s.accept("nN") && s.accept("fF") {
		return string(s.buf)
	}
	digits := decimalDigits + "_"
	exp := exponent
	if s.accept("0") && s.accept("xX") {
		digits = hexadecimalDigits + "_"
		exp = "pP"
	}
	for s.accept(digits) {
	}
	if s.accept(period) {
		for s.accept(digits) {
		}
	}
	if s.accept(exp) {
		s.accept(sign)
		for s.accept(decimalDigits + "_") {
		}
	}
	return string(s.buf)
}

func (s *ss) scanFloat(verb rune, bitSize int) float64 {
	if !s.okVerb(verb, "beEfFgGv", "float"+strconv.Itoa(bitSize)) {
		return 0
	}
	s.SkipSpace()
	s.notEOF()
	f, err := strconv.ParseFloat(s.floatToken(), bitSize)
	if err != nil {
		s.error(err)
	}
	return f
}

func (s *ss) convertString(verb rune) string {
	if !s.okVerb(verb, "svqxX", "string") {
		return ""
	}
	s.SkipSpace()
	s.notEOF()
	switch verb {
	case 'q':
		return s.quotedString()
	case 'x', 'X':
		return s.hexString()
	}
	return s.token()
}

func (s *ss) quotedString() string {
	s.notEOF()
	quote := s.getRune()
	switch quote {
	case '`':
		for {
			r := s.mustReadRune()
			if r == quote {
				break
			}
			s.buf = utf8.AppendRune(s.buf, r)
		}
		return string(s.buf)
	case '"':
		s.buf = append(s.buf, '"')
		for {
			r := s.mustReadRune()
			s.buf = utf8.AppendRune(s.buf, r)
			if r == '\\' {
				s.buf = utf8.AppendRune(s.buf, s.mustReadRune())
			} else if r == '"' {
				break
			}
		}
		result, err := strconv.Unquote(string(s.buf))
		if err != nil {
			s.error(err)
		}
		return result
	}
	s.errorString("expected quoted string")
	return ""
}

func hexDigit(d rune) (int, bool) {
	switch {
	case d >= '0' && d <= '9':
		return int(d - '0'), true
	case d >= 'a' && d <= 'f':
		return 10 + int(d-'a'), true
	case d >= 'A' && d <= 'F':
		return 10 + int(d-'A'), true
	}
	return -1, false
}

func (s *ss) hexByte() (byte, bool) {
	rune1 := s.getRune()
	if rune1 == eof {
		return 0, false
	}
	value1, ok := hexDigit(rune1)
	if !ok {
		s.UnreadRune()
		return 0, false
	}
	value2, ok := hexDigit(s.mustReadRune())
	if !ok {
		s.errorString("illegal hex digit")
		return 0, false
	}
	return byte(value1<<4 | value2), true
}

func (s *ss) hexString() string {
	s.notEOF()
	for {
		b, ok := s.hexByte()
		if !ok {
			break
		}
		s.buf = append(s.buf, b)
	}
	if len(s.buf) == 0 {
		s.errorString("no hex data for %x string")
		return ""
	}
	return string(s.buf)
}

// scanOne scans one operand by the kind of the value its pointer holds.
func (s *ss) scanOne(verb rune, arg any) {
	s.buf = s.buf[:0]
	if goRfKind(arg) != "ptr" {
		s.errorString("type not a pointer: " + typeText(arg))
	}
	kind := goRfKind(goRfDeref(arg))
	var v any
	switch kind {
	case "bool":
		v = s.scanBool(verb)
	case "int", "int64":
		v = s.scanInt(verb, 64)
	case "int8":
		v = s.scanInt(verb, 8)
	case "int16":
		v = s.scanInt(verb, 16)
	case "int32":
		v = s.scanInt(verb, 32)
	case "uint", "uint64", "uintptr":
		v = s.scanUint(verb, 64)
	case "uint8":
		v = s.scanUint(verb, 8)
	case "uint16":
		v = s.scanUint(verb, 16)
	case "uint32":
		v = s.scanUint(verb, 32)
	case "float32":
		v = s.scanFloat(verb, 32)
	case "float64":
		v = s.scanFloat(verb, 64)
	case "string":
		v = s.convertString(verb)
	default:
		s.errorString("can't scan type: " + typeText(arg))
	}
	goRfStore(arg, goRfConv(kind, v))
}

func typeText(arg any) string {
	switch k := goRfKind(arg); k {
	case "ptr":
		return "*" + typeText(goRfDeref(arg))
	case "nil":
		return "<nil>"
	default:
		return k
	}
}

func errorHandler(errp *error) {
	if e := recover(); e != nil {
		if se, ok := e.(scanError); ok {
			*errp = se.err
		} else if eof, ok := e.(error); ok && eof == io.EOF {
			*errp = eof
		} else {
			panic(e)
		}
	}
}

func (s *ss) doScan(a []any) (numProcessed int, err error) {
	defer errorHandler(&err)
	for _, arg := range a {
		s.scanOne('v', arg)
		numProcessed++
	}
	if s.nlIsEnd {
		for {
			r := s.getRune()
			if r == '\n' || r == eof {
				break
			}
			if !isSpace(r) {
				s.errorString("expected newline")
				break
			}
		}
	}
	return
}

// advance matches the format's spaces and literal text against the input, up
// to the next verb: the bytes of format consumed, or -1 when a literal does
// not match.
func (s *ss) advance(format string) (i int) {
	for i < len(format) {
		fmtc, w := utf8.DecodeRuneInString(format[i:])
		// A newline in the format matches zero or more spaces and then a newline
		// or the end of the input; spaces before it collapse into it, spaces
		// after it match zero or more spaces after the input's newline, and any
		// other run of spaces matches one or more spaces or the end of input.
		if isSpace(fmtc) {
			newlines := 0
			trailingSpace := false
			for isSpace(fmtc) && i < len(format) {
				if fmtc == '\n' {
					newlines++
					trailingSpace = false
				} else {
					trailingSpace = true
				}
				i += w
				fmtc, w = utf8.DecodeRuneInString(format[i:])
			}
			for j := 0; j < newlines; j++ {
				inputc := s.getRune()
				for isSpace(inputc) && inputc != '\n' {
					inputc = s.getRune()
				}
				if inputc != '\n' && inputc != eof {
					s.errorString("newline in format does not match input")
				}
			}
			if trailingSpace {
				inputc := s.getRune()
				if newlines == 0 {
					if !isSpace(inputc) && inputc != eof {
						s.errorString("expected space in input to match format")
					}
					if inputc == '\n' {
						s.errorString("newline in input does not match format")
					}
				}
				for isSpace(inputc) && inputc != '\n' {
					inputc = s.getRune()
				}
				if inputc != eof {
					s.UnreadRune()
				}
			}
			continue
		}
		if fmtc == '%' {
			if i+w == len(format) {
				s.errorString("missing verb: % at end of format string")
			}
			nextc, _ := utf8.DecodeRuneInString(format[i+w:])
			if nextc != '%' {
				return
			}
			i += w
		}
		inputc := s.mustReadRune()
		if fmtc != inputc {
			s.UnreadRune()
			return -1
		}
		i += w
	}
	return
}

func (s *ss) doScanf(format string, a []any) (numProcessed int, err error) {
	defer errorHandler(&err)
	end := len(format) - 1
	for i := 0; i <= end; {
		w := s.advance(format[i:])
		if w > 0 {
			i += w
			continue
		}
		if format[i] != '%' {
			if w < 0 {
				s.errorString("input does not match format")
			}
			break
		}
		i++
		// A width: the operand reads at most that many runes.
		wid, widPresent := 0, false
		for i <= end && format[i] >= '0' && format[i] <= '9' {
			wid = wid*10 + int(format[i]-'0')
			widPresent = true
			i++
		}
		s.maxWid = hugeWid
		if widPresent {
			s.maxWid = wid
		}
		c, cw := utf8.DecodeRuneInString(format[i:])
		i += cw
		if c != 'c' {
			s.SkipSpace()
		}
		if c == '%' {
			s.SkipSpace()
			s.notEOF()
			if !s.accept("%") {
				s.errorString("missing literal %")
			}
			continue
		}
		s.argLimit = s.limit
		if f := s.count + s.maxWid; f < s.argLimit {
			s.argLimit = f
		}
		if numProcessed >= len(a) {
			s.errorString("too few operands for format '%" + format[i-cw:] + "'")
			break
		}
		s.scanOne(c, a[numProcessed])
		numProcessed++
		s.argLimit = s.limit
	}
	if numProcessed < len(a) {
		s.errorString("too many operands")
	}
	return
}
