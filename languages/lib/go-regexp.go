// Package regexp: Go's regexp package WRITTEN IN GO, compiled with a program that
// imports "regexp" (core.importFile in both go grammars), over three primitives
// both grammars bind: goRxError (Go's error text for a pattern, "" when it
// compiles), goRxNames (its group names) and goRxExec (the leftmost match at or
// after a byte offset, as Go's submatch index pairs). They are lib/go-regexp.js
// and its twin abnf/jsrtgoregexp.go over lib/regex.js's R dialect, which reads
// RE2 syntax and matches leftmost-first as Go does.
//
// The finding, replacing and splitting below follow Go's regexp.go (Copyright
// The Go Authors, BSD-style license). Not provided: CompilePOSIX and Longest
// (leftmost-longest matching), LiteralPrefix, and the step cap of the
// backtracking matcher stands where Go's matcher is linear.
package regexp

import (
	"errors"
	"io"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Regexp struct {
	expr        string
	numSubexp   int
	subexpNames []string
}

func Compile(expr string) (*Regexp, error) {
	if msg := goRxError(expr); msg != "" {
		return nil, errors.New(msg)
	}
	names := goRxNames(expr)
	sn := make([]string, len(names))
	for i := range sn {
		sn[i] = names[i]
	}
	return &Regexp{expr: expr, numSubexp: len(sn) - 1, subexpNames: sn}, nil
}

func MustCompile(str string) *Regexp {
	re, err := Compile(str)
	if err != nil {
		panic(`regexp: Compile(` + quote(str) + `): ` + err.Error())
	}
	return re
}

func quote(s string) string {
	if strconv.CanBackquote(s) {
		return "`" + s + "`"
	}
	return strconv.Quote(s)
}

func (re *Regexp) String() string               { return re.expr }
func (re *Regexp) Copy() *Regexp                { c := *re; return &c }
func (re *Regexp) NumSubexp() int               { return re.numSubexp }
func (re *Regexp) SubexpNames() []string        { return re.subexpNames }
func (re *Regexp) MarshalText() ([]byte, error) { return []byte(re.expr), nil }

func (re *Regexp) SubexpIndex(name string) int {
	if name != "" {
		for i, s := range re.subexpNames {
			if name == s {
				return i
			}
		}
	}
	return -1
}

func (re *Regexp) UnmarshalText(text []byte) error {
	r, err := Compile(string(text))
	if err != nil {
		return err
	}
	re.expr, re.numSubexp, re.subexpNames = r.expr, r.numSubexp, r.subexpNames
	return nil
}

// find is the leftmost match in s at or after byte pos: 2*(NumSubexp+1) offsets,
// nil for none.
func (re *Regexp) find(s string, pos int) []int {
	a := goRxExec(re.expr, s, pos)
	if len(a) == 0 {
		return nil
	}
	m := make([]int, len(a))
	for i := range m {
		m[i] = a[i]
	}
	return m
}

func (re *Regexp) MatchString(s string) bool { return re.find(s, 0) != nil }
func (re *Regexp) Match(b []byte) bool       { return re.find(string(b), 0) != nil }

func (re *Regexp) MatchReader(r io.RuneReader) bool { return re.MatchString(readRunes(r)) }

// readRunes is everything r has left, read rune by rune.
func readRunes(r io.RuneReader) string {
	var sb strings.Builder
	for {
		c, _, err := r.ReadRune()
		if err != nil {
			return sb.String()
		}
		sb.WriteRune(c)
	}
}

func MatchString(pattern string, s string) (matched bool, err error) {
	re, err := Compile(pattern)
	if err != nil {
		return false, err
	}
	return re.MatchString(s), nil
}

func Match(pattern string, b []byte) (matched bool, err error) {
	re, err := Compile(pattern)
	if err != nil {
		return false, err
	}
	return re.Match(b), nil
}

func MatchReader(pattern string, r io.RuneReader) (matched bool, err error) {
	re, err := Compile(pattern)
	if err != nil {
		return false, err
	}
	return re.MatchReader(r), nil
}

// QuoteMeta escapes every metacharacter of s with a backslash.
func QuoteMeta(s string) string {
	var b []byte
	for i := 0; i < len(s); i++ {
		if strings.IndexByte(`\.+*?()|[]{}^$`, s[i]) >= 0 {
			b = append(b, '\\')
		}
		b = append(b, s[i])
	}
	return string(b)
}

// ----- finding -----

func (re *Regexp) FindString(s string) string {
	a := re.find(s, 0)
	if a == nil {
		return ""
	}
	return s[a[0]:a[1]]
}

func (re *Regexp) Find(b []byte) []byte {
	a := re.find(string(b), 0)
	if a == nil {
		return nil
	}
	return b[a[0]:a[1]:a[1]]
}

func (re *Regexp) FindStringIndex(s string) []int {
	a := re.find(s, 0)
	if a == nil {
		return nil
	}
	return a[0:2]
}

func (re *Regexp) FindIndex(b []byte) []int { return re.FindStringIndex(string(b)) }

func (re *Regexp) FindReaderIndex(r io.RuneReader) []int {
	return re.FindStringIndex(readRunes(r))
}

func (re *Regexp) FindStringSubmatch(s string) []string {
	a := re.find(s, 0)
	if a == nil {
		return nil
	}
	ret := make([]string, 1+re.numSubexp)
	for i := range ret {
		if 2*i < len(a) && a[2*i] >= 0 {
			ret[i] = s[a[2*i]:a[2*i+1]]
		}
	}
	return ret
}

func (re *Regexp) FindSubmatch(b []byte) [][]byte {
	a := re.find(string(b), 0)
	if a == nil {
		return nil
	}
	ret := make([][]byte, 1+re.numSubexp)
	for i := range ret {
		if 2*i < len(a) && a[2*i] >= 0 {
			ret[i] = b[a[2*i]:a[2*i+1]:a[2*i+1]]
		}
	}
	return ret
}

func (re *Regexp) FindStringSubmatchIndex(s string) []int { return re.find(s, 0) }
func (re *Regexp) FindSubmatchIndex(b []byte) []int       { return re.find(string(b), 0) }

func (re *Regexp) FindReaderSubmatchIndex(r io.RuneReader) []int {
	return re.find(readRunes(r), 0)
}

// all is every match in s, n at most when n >= 0: after an empty match the
// search moves on one character, and an empty match right where the previous
// match ended is not one (regexp.go's allMatches).
func (re *Regexp) all(s string, n int) [][]int {
	var out [][]int
	end := len(s)
	for pos, prevMatchEnd := 0, -1; pos <= end && (n < 0 || len(out) < n); {
		m := re.find(s, pos)
		if m == nil {
			break
		}
		accept := true
		if m[1] == pos {
			if m[0] == prevMatchEnd {
				accept = false
			}
			if pos < end {
				pos += width(s, pos)
			} else {
				pos = end + 1
			}
		} else {
			pos = m[1]
		}
		prevMatchEnd = m[1]
		if accept {
			out = append(out, m)
		}
	}
	return out
}

func (re *Regexp) FindAllString(s string, n int) []string {
	ms := re.all(s, n)
	if ms == nil {
		return nil
	}
	out := make([]string, len(ms))
	for i, m := range ms {
		out[i] = s[m[0]:m[1]]
	}
	return out
}

func (re *Regexp) FindAll(b []byte, n int) [][]byte {
	ms := re.all(string(b), n)
	if ms == nil {
		return nil
	}
	out := make([][]byte, len(ms))
	for i, m := range ms {
		out[i] = b[m[0]:m[1]:m[1]]
	}
	return out
}

func (re *Regexp) FindAllStringIndex(s string, n int) [][]int {
	ms := re.all(s, n)
	if ms == nil {
		return nil
	}
	out := make([][]int, len(ms))
	for i, m := range ms {
		out[i] = m[0:2]
	}
	return out
}

func (re *Regexp) FindAllIndex(b []byte, n int) [][]int { return re.FindAllStringIndex(string(b), n) }

func (re *Regexp) FindAllStringSubmatch(s string, n int) [][]string {
	ms := re.all(s, n)
	if ms == nil {
		return nil
	}
	out := make([][]string, len(ms))
	for k, m := range ms {
		sub := make([]string, 1+re.numSubexp)
		for i := range sub {
			if 2*i < len(m) && m[2*i] >= 0 {
				sub[i] = s[m[2*i]:m[2*i+1]]
			}
		}
		out[k] = sub
	}
	return out
}

func (re *Regexp) FindAllSubmatch(b []byte, n int) [][][]byte {
	ms := re.all(string(b), n)
	if ms == nil {
		return nil
	}
	out := make([][][]byte, len(ms))
	for k, m := range ms {
		sub := make([][]byte, 1+re.numSubexp)
		for i := range sub {
			if 2*i < len(m) && m[2*i] >= 0 {
				sub[i] = b[m[2*i]:m[2*i+1]:m[2*i+1]]
			}
		}
		out[k] = sub
	}
	return out
}

func (re *Regexp) FindAllStringSubmatchIndex(s string, n int) [][]int { return re.all(s, n) }
func (re *Regexp) FindAllSubmatchIndex(b []byte, n int) [][]int       { return re.all(string(b), n) }

// width is the length of the character at byte i of s, as
// utf8.DecodeRuneInString reads it; only the bytes it can span are sliced.
func width(s string, i int) int {
	if s[i] < utf8.RuneSelf {
		return 1
	}
	end := i + utf8.UTFMax
	if end > len(s) {
		end = len(s)
	}
	_, w := utf8.DecodeRuneInString(s[i:end])
	return w
}

// ----- replacing -----

// replaceAll writes src with every match replaced by what repl appends: no
// replacement for an empty match right after a previous match, and the search
// always moves on at least one character (regexp.go's replaceAll).
func (re *Regexp) replaceAll(src string, repl func(dst []byte, m []int) []byte) []byte {
	lastMatchEnd := 0
	searchPos := 0
	var buf []byte
	for searchPos <= len(src) {
		a := re.find(src, searchPos)
		if a == nil {
			break
		}
		buf = append(buf, src[lastMatchEnd:a[0]]...)
		if a[1] > lastMatchEnd || a[0] == 0 {
			buf = repl(buf, a)
		}
		lastMatchEnd = a[1]
		w := 0
		if searchPos < len(src) {
			w = width(src, searchPos)
		}
		if searchPos+w > a[1] {
			searchPos += w
		} else if searchPos+1 > a[1] {
			searchPos++
		} else {
			searchPos = a[1]
		}
	}
	return append(buf, src[lastMatchEnd:]...)
}

func (re *Regexp) ReplaceAllString(src, repl string) string {
	return string(re.replaceAll(src, func(dst []byte, m []int) []byte {
		return re.expand(dst, repl, src, m)
	}))
}

func (re *Regexp) ReplaceAllLiteralString(src, repl string) string {
	return string(re.replaceAll(src, func(dst []byte, m []int) []byte {
		return append(dst, repl...)
	}))
}

func (re *Regexp) ReplaceAllStringFunc(src string, repl func(string) string) string {
	return string(re.replaceAll(src, func(dst []byte, m []int) []byte {
		return append(dst, repl(src[m[0]:m[1]])...)
	}))
}

func (re *Regexp) ReplaceAll(src, repl []byte) []byte {
	s := string(src)
	r := string(repl)
	return re.replaceAll(s, func(dst []byte, m []int) []byte {
		return re.expand(dst, r, s, m)
	})
}

func (re *Regexp) ReplaceAllLiteral(src, repl []byte) []byte {
	return re.replaceAll(string(src), func(dst []byte, m []int) []byte {
		return append(dst, repl...)
	})
}

func (re *Regexp) ReplaceAllFunc(src []byte, repl func([]byte) []byte) []byte {
	return re.replaceAll(string(src), func(dst []byte, m []int) []byte {
		return append(dst, repl(src[m[0]:m[1]:m[1]])...)
	})
}

func (re *Regexp) Expand(dst []byte, template []byte, src []byte, match []int) []byte {
	return re.expand(dst, string(template), string(src), match)
}

func (re *Regexp) ExpandString(dst []byte, template string, src string, match []int) []byte {
	return re.expand(dst, template, src, match)
}

// expand appends template to dst with $1, ${1}, $name and ${name} replaced by
// that group's text and $$ by $; a name is as long as it can be, so $1x is
// ${1x}, and a group that is not there or did not match gives nothing.
func (re *Regexp) expand(dst []byte, template string, src string, match []int) []byte {
	for len(template) > 0 {
		before, after, ok := strings.Cut(template, "$")
		if !ok {
			break
		}
		dst = append(dst, before...)
		template = after
		if template != "" && template[0] == '$' {
			dst = append(dst, '$')
			template = template[1:]
			continue
		}
		name, num, rest, ok := extract(template)
		if !ok {
			dst = append(dst, '$')
			continue
		}
		template = rest
		if num >= 0 {
			if 2*num+1 < len(match) && match[2*num] >= 0 {
				dst = append(dst, src[match[2*num]:match[2*num+1]]...)
			}
		} else {
			for i, namei := range re.subexpNames {
				if name == namei && 2*i+1 < len(match) && match[2*i] >= 0 {
					dst = append(dst, src[match[2*i]:match[2*i+1]]...)
					break
				}
			}
		}
	}
	return append(dst, template...)
}

// extract reads the name after a $: letters, digits and _, or the same in
// braces; num is its value when it is a number without a leading zero, else -1.
func extract(str string) (name string, num int, rest string, ok bool) {
	if str == "" {
		return "", 0, "", false
	}
	brace := false
	if str[0] == '{' {
		brace = true
		str = str[1:]
	}
	i := 0
	for i < len(str) {
		r, size := utf8.DecodeRuneInString(str[i:])
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			break
		}
		i += size
	}
	if i == 0 {
		return "", 0, "", false
	}
	name = str[:i]
	if brace {
		if i >= len(str) || str[i] != '}' {
			return "", 0, "", false
		}
		i++
	}
	num = 0
	for k := 0; k < len(name); k++ {
		if name[k] < '0' || '9' < name[k] || num >= 1e8 {
			num = -1
			break
		}
		num = num*10 + int(name[k]) - '0'
	}
	if name[0] == '0' && len(name) > 1 {
		num = -1
	}
	return name, num, str[i:], true
}

// Split cuts s around the matches: at most n pieces when n > 0, the last one
// the unsplit rest; none when n == 0.
func (re *Regexp) Split(s string, n int) []string {
	if n == 0 {
		return nil
	}
	if len(re.expr) > 0 && len(s) == 0 {
		return []string{""}
	}
	matches := re.FindAllStringIndex(s, n)
	pieces := make([]string, 0, len(matches))
	beg := 0
	end := 0
	for _, match := range matches {
		if n > 0 && len(pieces) >= n-1 {
			break
		}
		end = match[0]
		if match[1] != 0 {
			pieces = append(pieces, s[beg:end])
		}
		beg = match[1]
	}
	if end != len(s) {
		pieces = append(pieces, s[beg:])
	}
	return pieces
}
