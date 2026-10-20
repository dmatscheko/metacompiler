// Package json: Go's encoding/json WRITTEN IN GO, compiled with a program that
// imports "encoding/json" (core.importFile in both go grammars), over the
// run-time type primitives both grammars bind for it and for reflect (goRf*:
// a value's kind, a struct's field names, declared types and tags, a slice's
// elements, a map's keys and values, and the values a decode builds): the
// goRf* host globals of go-interpreter.abnf, abnf/jsrtgoreflect.go and the
// js_gorf* of lib/go-rt.metajs, and goRfZero emitted by go-to-llvm-ir.abnf.
//
// The string and float encodings are encode.go's and the decoding rules
// decode.go's (Copyright The Go Authors, BSD-style license); the syntax errors
// are the ones go1.27's jsontext-based decoder reports. A Marshaler, Unmarshaler or
// TextMarshaler is honoured on a struct; a pointer is the value it points at,
// which is how this runtime holds a pointer to a struct (and a map is its own
// pointer, so a decode fills the map a variable holds). An unsupported type
// error names a func's type as fmt's %T does here, without its signature.
package json

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Marshaler interface {
	MarshalJSON() ([]byte, error)
}

type Unmarshaler interface {
	UnmarshalJSON([]byte) error
}

type textMarshaler interface {
	MarshalText() ([]byte, error)
}

type textUnmarshaler interface {
	UnmarshalText([]byte) error
}

// RawMessage is a raw encoded JSON value.
type RawMessage []byte

func (m RawMessage) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("null"), nil
	}
	return m, nil
}

// A Number is a JSON number literal.
type Number string

func (n Number) String() string            { return string(n) }
func (n Number) Float64() (float64, error) { return strconv.ParseFloat(string(n), 64) }
func (n Number) Int64() (int64, error)     { return strconv.ParseInt(string(n), 10, 64) }

// ----- errors -----

type SyntaxError struct {
	msg    string
	Offset int64
}

func (e *SyntaxError) Error() string { return e.msg }

type UnmarshalTypeError struct {
	Value  string
	Type   string
	Offset int64
	Struct string
	Field  string
	Err    error
}

func (e *UnmarshalTypeError) Error() string {
	s := "json: cannot unmarshal " + e.Value + " into Go value of type " + e.Type
	if e.Struct != "" || e.Field != "" {
		s = "json: cannot unmarshal " + e.Value + " into Go struct field " + e.Struct + "." + e.Field + " of type " + e.Type
	}
	if e.Err != nil {
		s += ": " + e.Err.Error()
	}
	return s
}

func (e *UnmarshalTypeError) Unwrap() error { return e.Err }

type InvalidUnmarshalError struct {
	Type string
}

func (e *InvalidUnmarshalError) Error() string {
	if e.Type == "" {
		return "json: Unmarshal(nil)"
	}
	if !strings.HasPrefix(e.Type, "*") {
		return "json: Unmarshal(non-pointer " + e.Type + ")"
	}
	return "json: Unmarshal(nil " + e.Type + ")"
}

type UnsupportedTypeError struct {
	Type string
}

func (e *UnsupportedTypeError) Error() string { return "json: unsupported type: " + e.Type }

type UnsupportedValueError struct {
	Str string
}

func (e *UnsupportedValueError) Error() string { return "json: unsupported value: " + e.Str }

type MarshalerError struct {
	Type       string
	Err        error
	sourceFunc string
}

func (e *MarshalerError) Error() string {
	src := e.sourceFunc
	if src == "" {
		src = "MarshalJSON"
	}
	return "json: error calling " + src + " for type " + e.Type + ": " + e.Err.Error()
}

func (e *MarshalerError) Unwrap() error { return e.Err }

// ----- struct tags -----

// tagLookup is reflect.StructTag.Lookup: the value of key in a conventional
// `key:"value" key2:"value2"` tag.
func tagLookup(tag, key string) (string, bool) {
	for tag != "" {
		i := 0
		for i < len(tag) && tag[i] == ' ' {
			i++
		}
		tag = tag[i:]
		if tag == "" {
			break
		}
		i = 0
		for i < len(tag) && tag[i] > ' ' && tag[i] != ':' && tag[i] != '"' && tag[i] != 0x7f {
			i++
		}
		if i == 0 || i+1 >= len(tag) || tag[i] != ':' || tag[i+1] != '"' {
			break
		}
		name := tag[:i]
		tag = tag[i+1:]
		i = 1
		for i < len(tag) && tag[i] != '"' {
			if tag[i] == '\\' {
				i++
			}
			i++
		}
		if i >= len(tag) {
			break
		}
		qvalue := tag[:i+1]
		tag = tag[i+1:]
		if key == name {
			value, err := strconv.Unquote(qvalue)
			if err != nil {
				break
			}
			return value, true
		}
	}
	return "", false
}

func isValidTag(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		switch {
		case strings.ContainsRune("!#$%&()*+-./:;<=>?@[]^_{|}~ ", c):
		case !unicode.IsLetter(c) && !unicode.IsDigit(c):
			return false
		}
	}
	return true
}

// field is one encoded struct field: its JSON name, its index path through
// embedded structs, its declared type and its tag options.
type field struct {
	name      string
	path      []int
	typ       string
	tagged    bool
	omitEmpty bool
	omitZero  bool
	quoted    bool
}

// fieldsOf lists the fields of struct value v that JSON reads and writes:
// exported ones, an embedded struct's promoted (shallower and tagged names
// win, an ambiguous name is dropped), in declaration order.
func fieldsOf(v interface{}) []field {
	var out []field
	collectFields(v, nil, &out, 0)
	// Go's dominance rule: of the fields sharing a name, the shallowest one
	// wins, a tagged one over an untagged one at that depth; a tie drops the name.
	var res []field
	for i := range out {
		f := out[i]
		keep := true
		for j := range out {
			if i == j || out[j].name != f.name {
				continue
			}
			g := out[j]
			if len(g.path) < len(f.path) || len(g.path) == len(f.path) && g.tagged && !f.tagged {
				keep = false
			} else if len(g.path) == len(f.path) && g.tagged == f.tagged {
				keep = false
			}
		}
		if keep {
			res = append(res, f)
		}
	}
	return res
}

func collectFields(v interface{}, path []int, out *[]field, depth int) {
	fs := goRfFields(v)
	names, types, tags, vals := fs[0], fs[1], fs[2], fs[3]
	for i := 0; i < len(names); i++ {
		name := names[i]
		typ := types[i]
		tag, _ := tagLookup(tags[i], "json")
		if tag == "-" {
			continue
		}
		tname, opts, _ := strings.Cut(tag, ",")
		if !isValidTag(tname) {
			tname = ""
		}
		p := append(append([]int{}, path...), i)
		embedded := typ == "" || typ == "*"+name
		if embedded && tname == "" && goRfKind(vals[i]) == "struct" && depth < 16 {
			collectFields(vals[i], p, out, depth+1)
			continue
		}
		if !embedded && !isExportedName(name) {
			continue
		}
		f := field{name: name, path: p, typ: typ, tagged: tname != ""}
		if tname != "" {
			f.name = tname
		}
		for opts != "" {
			var o string
			o, opts, _ = strings.Cut(opts, ",")
			switch o {
			case "omitempty":
				f.omitEmpty = true
			case "omitzero":
				f.omitZero = true
			case "string":
				f.quoted = true
			}
		}
		*out = append(*out, f)
	}
}

// The scalar inside v, without the box of a defined type: a string, bool,
// float or integer of one of the predeclared types.
func asFloat(v interface{}) (float64, bool) {
	switch x := goRfUnbox(v).(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	}
	return 0, false
}

func asString(v interface{}) string {
	s, _ := goRfUnbox(v).(string)
	return s
}

func asByte(v interface{}) byte {
	switch x := goRfUnbox(v).(type) {
	case uint8:
		return x
	case int:
		return byte(x)
	}
	return 0
}

// listLen is the length of a slice, an array or a map.
func listLen(v interface{}) int {
	if goRfKind(v) == "map" {
		return len(goRfMapKV(v)[0])
	}
	return len(goRfElems(v))
}

// typeName spells type text t as reflect does: main.T for a struct type of
// this program, interface {} for the empty interface.
func typeName(t string) string {
	if strings.HasPrefix(t, "*") {
		return "*" + typeName(t[1:])
	}
	if t == "interface{}" || t == "any" {
		return "interface {}"
	}
	if z := goRfZero(t); z != nil {
		return fmt.Sprintf("%T", z)
	}
	return t
}

func isExportedName(s string) bool {
	r, _ := utf8.DecodeRuneInString(s)
	return unicode.IsUpper(r)
}

// fieldValue follows f's index path from struct value v.
func fieldValue(v interface{}, f field) interface{} {
	for _, i := range f.path {
		if v == nil || goRfIsNil(v) {
			return nil
		}
		v = goRfFields(v)[3][i]
	}
	return v
}

// ----- encoding -----

type encodeState struct {
	buf        []byte
	escapeHTML bool
	depth      int
}

func Marshal(v any) ([]byte, error) {
	e := &encodeState{escapeHTML: true}
	if err := e.value(v, false); err != nil {
		return nil, err
	}
	return e.buf, nil
}

func MarshalIndent(v any, prefix, indent string) ([]byte, error) {
	b, err := Marshal(v)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := Indent(&buf, b, prefix, indent); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// isEmpty is omitempty's rule: false, 0, "", nil, and an empty array, slice
// or map.
func isEmpty(v interface{}) bool {
	switch k := goRfKind(v); k {
	case "nil":
		return true
	case "bool":
		return goRfUnbox(v) == false
	case "string":
		return asString(v) == ""
	case "slice", "map", "array":
		return goRfIsNil(v) || listLen(v) == 0
	case "ptr", "func", "chan":
		return goRfIsNil(v)
	case "struct":
		return false
	}
	return isNumZero(v)
}

func isNumZero(v interface{}) bool {
	if f, ok := asFloat(v); ok {
		return f == 0 && !math.Signbit(f)
	}
	return fmt.Sprint(v) == "0"
}

// isZero is omitzero's rule: the zero value of the type.
func isZero(v interface{}) bool {
	if m, ok := v.(interface{ IsZero() bool }); ok && goRfKind(v) == "struct" {
		return m.IsZero()
	}
	switch k := goRfKind(v); k {
	case "struct":
		vals := goRfFields(v)[3]
		for i := 0; i < len(vals); i++ {
			if !isZero(vals[i]) {
				return false
			}
		}
		return true
	case "array":
		els := goRfElems(v)
		for i := 0; i < len(els); i++ {
			if !isZero(els[i]) {
				return false
			}
		}
		return true
	case "slice", "map", "ptr", "func", "chan":
		return goRfIsNil(v)
	}
	return isEmpty(v)
}

func (e *encodeState) value(v interface{}, quoted bool) error {
	kind := goRfKind(v)
	if kind == "nil" || (kind == "ptr" || kind == "map" || kind == "slice") && goRfIsNil(v) {
		e.buf = append(e.buf, "null"...)
		return nil
	}
	if kind == "ptr" {
		return e.value(goRfDeref(v), quoted)
	}
	if kind == "struct" {
		if m, ok := v.(Marshaler); ok {
			b, err := m.MarshalJSON()
			if err != nil {
				return &MarshalerError{Type: fmt.Sprintf("%T", v), Err: err}
			}
			var c bytes.Buffer
			if err := compact(&c, b, e.escapeHTML); err != nil {
				return &MarshalerError{Type: fmt.Sprintf("%T", v), Err: err}
			}
			e.buf = append(e.buf, c.Bytes()...)
			return nil
		}
		if m, ok := v.(textMarshaler); ok {
			b, err := m.MarshalText()
			if err != nil {
				return &MarshalerError{Type: fmt.Sprintf("%T", v), Err: err, sourceFunc: "MarshalText"}
			}
			e.buf = appendString(e.buf, string(b), e.escapeHTML)
			return nil
		}
	}
	switch kind {
	case "bool":
		s := "false"
		if goRfUnbox(v) == true {
			s = "true"
		}
		e.scalar(s, quoted)
	case "string":
		s := asString(v)
		if quoted {
			e.buf = appendString(e.buf, string(appendString(nil, s, e.escapeHTML)), e.escapeHTML)
		} else {
			e.buf = appendString(e.buf, s, e.escapeHTML)
		}
	case "float32", "float64":
		f, _ := asFloat(v)
		bits := 64
		if kind == "float32" {
			bits = 32
		}
		if math.IsInf(f, 0) || math.IsNaN(f) {
			return &UnsupportedValueError{Str: strconv.FormatFloat(f, 'g', -1, bits)}
		}
		e.scalar(formatFloat(f, bits), quoted)
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr":
		e.scalar(fmt.Sprint(goRfUnbox(v)), quoted)
	case "slice", "array":
		if kind == "slice" && goRfIsBytes(v) {
			els := goRfElems(v)
			b := make([]byte, len(els))
			for i := range b {
				b[i] = asByte(els[i])
			}
			e.buf = append(e.buf, '"')
			e.buf = append(e.buf, base64(b)...)
			e.buf = append(e.buf, '"')
			return nil
		}
		els := goRfElems(v)
		e.buf = append(e.buf, '[')
		for i := 0; i < len(els); i++ {
			if i > 0 {
				e.buf = append(e.buf, ',')
			}
			if err := e.value(els[i], false); err != nil {
				return err
			}
		}
		e.buf = append(e.buf, ']')
	case "map":
		return e.mapValue(v)
	case "struct":
		return e.structValue(v)
	default:
		return &UnsupportedTypeError{Type: fmt.Sprintf("%T", v)}
	}
	return nil
}

func (e *encodeState) scalar(s string, quoted bool) {
	if quoted {
		e.buf = append(e.buf, '"')
	}
	e.buf = append(e.buf, s...)
	if quoted {
		e.buf = append(e.buf, '"')
	}
}

func (e *encodeState) mapValue(v interface{}) error {
	kv := goRfMapKV(v)
	ks, vs := kv[0], kv[1]
	type entry struct {
		key string
		val interface{}
	}
	var entries []entry
	for i := 0; i < len(ks); i++ {
		k := ks[i]
		var key string
		switch goRfKind(k) {
		case "string":
			key = asString(k)
		case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr":
			key = fmt.Sprint(goRfUnbox(k))
		default:
			if m, ok := k.(textMarshaler); ok {
				b, err := m.MarshalText()
				if err != nil {
					return &MarshalerError{Type: fmt.Sprintf("%T", k), Err: err, sourceFunc: "MarshalText"}
				}
				key = string(b)
			} else {
				return &UnsupportedTypeError{Type: fmt.Sprintf("%T", v)}
			}
		}
		entries = append(entries, entry{key, vs[i]})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].key < entries[j].key })
	e.buf = append(e.buf, '{')
	for i, en := range entries {
		if i > 0 {
			e.buf = append(e.buf, ',')
		}
		e.buf = appendString(e.buf, en.key, e.escapeHTML)
		e.buf = append(e.buf, ':')
		if err := e.value(en.val, false); err != nil {
			return err
		}
	}
	e.buf = append(e.buf, '}')
	return nil
}

func (e *encodeState) structValue(v interface{}) error {
	e.depth++
	if e.depth > 1000 {
		return &UnsupportedValueError{Str: "encountered a cycle via " + fmt.Sprintf("%T", v)}
	}
	e.buf = append(e.buf, '{')
	first := true
	for _, f := range fieldsOf(v) {
		fv := fieldValue(v, f)
		if f.omitEmpty && isEmpty(fv) || f.omitZero && isZero(fv) {
			continue
		}
		if !first {
			e.buf = append(e.buf, ',')
		}
		first = false
		e.buf = appendString(e.buf, f.name, e.escapeHTML)
		e.buf = append(e.buf, ':')
		q := f.quoted
		if q {
			switch goRfKind(fv) {
			case "bool", "string", "float32", "float64", "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr":
			default:
				q = false
			}
		}
		if err := e.value(fv, q); err != nil {
			return err
		}
	}
	e.buf = append(e.buf, '}')
	e.depth--
	return nil
}

// formatFloat is encode.go's floatEncoder: ES6's number to string, %g with
// the exponent cutoffs 1e-6 and 1e21 and an exponent not padded to two digits.
func formatFloat(f float64, bits int) string {
	abs := math.Abs(f)
	format := byte('f')
	if abs != 0 {
		if bits == 64 && (abs < 1e-6 || abs >= 1e21) || bits == 32 && (float32(abs) < 1e-6 || float32(abs) >= 1e21) {
			format = 'e'
		}
	}
	s := strconv.FormatFloat(f, format, -1, bits)
	if format == 'e' {
		n := len(s)
		if n >= 4 && s[n-4] == 'e' && s[n-3] == '-' && s[n-2] == '0' {
			s = s[:n-2] + s[n-1:]
		}
	}
	return s
}

const hexDigits = "0123456789abcdef"

// appendString is encode.go's: the escapes JSON needs, <, > and & escaped
// when escapeHTML, invalid UTF-8 as �, U+2028 and U+2029 escaped.
func appendString(dst []byte, src string, escapeHTML bool) []byte {
	dst = append(dst, '"')
	start := 0
	for i := 0; i < len(src); {
		if b := src[i]; b < utf8.RuneSelf {
			if b >= ' ' && b != '"' && b != '\\' && (!escapeHTML || b != '<' && b != '>' && b != '&') {
				i++
				continue
			}
			dst = append(dst, src[start:i]...)
			switch b {
			case '\\', '"':
				dst = append(dst, '\\', b)
			case '\b':
				dst = append(dst, '\\', 'b')
			case '\f':
				dst = append(dst, '\\', 'f')
			case '\n':
				dst = append(dst, '\\', 'n')
			case '\r':
				dst = append(dst, '\\', 'r')
			case '\t':
				dst = append(dst, '\\', 't')
			default:
				dst = append(dst, '\\', 'u', '0', '0', hexDigits[b>>4], hexDigits[b&0xF])
			}
			i++
			start = i
			continue
		}
		n := len(src) - i
		if n > utf8.UTFMax {
			n = utf8.UTFMax
		}
		c, size := utf8.DecodeRuneInString(src[i : i+n])
		if c == utf8.RuneError && size == 1 {
			dst = append(dst, src[start:i]...)
			dst = append(dst, `�`...)
			i += size
			start = i
			continue
		}
		if c == ' ' || c == ' ' {
			dst = append(dst, src[start:i]...)
			dst = append(dst, '\\', 'u', '2', '0', '2', hexDigits[c&0xF])
			i += size
			start = i
			continue
		}
		i += size
	}
	dst = append(dst, src[start:]...)
	return append(dst, '"')
}

const b64 = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

// base64 is encoding/base64's StdEncoding.
func base64(src []byte) string {
	var out []byte
	for i := 0; i < len(src); i += 3 {
		n := len(src) - i
		var v uint32
		v = uint32(src[i]) << 16
		if n > 1 {
			v |= uint32(src[i+1]) << 8
		}
		if n > 2 {
			v |= uint32(src[i+2])
		}
		out = append(out, b64[v>>18&63], b64[v>>12&63])
		if n > 1 {
			out = append(out, b64[v>>6&63])
		} else {
			out = append(out, '=')
		}
		if n > 2 {
			out = append(out, b64[v&63])
		} else {
			out = append(out, '=')
		}
	}
	return string(out)
}

func unbase64(s string) ([]byte, error) {
	var out []byte
	var acc uint32
	nbits := 0
	pad := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\r' || c == '\n' {
			continue
		}
		if c == '=' {
			pad++
			continue
		}
		k := strings.IndexByte(b64, c)
		if k < 0 || pad > 0 {
			return nil, errors.New("illegal base64 data at input byte " + strconv.Itoa(i))
		}
		acc = acc<<6 | uint32(k)
		nbits += 6
		if nbits >= 8 {
			nbits -= 8
			out = append(out, byte(acc>>uint(nbits)))
		}
	}
	return out, nil
}

// ----- the syntax checker -----
//
// Go's encoding/json reads through encoding/json/jsontext and reports a syntax
// error the way its v1 always has: "invalid character 'x' <where>" with the
// offset just past that character, or "unexpected end of JSON input" with the
// offset where the unfinished token began (an unfinished number or escape) or
// the end of the input. A duplicate name, a lone surrogate escape and invalid
// UTF-8 are accepted.

type checker struct {
	data []byte
	err  *SyntaxError
}

// maxDepth is how deeply objects and arrays may nest, encoding/json's limit;
// the decode that follows a check recurses no deeper than this either.
const maxDepth = 10000

func (c *checker) fail(msg string, off int) {
	if c.err == nil {
		c.err = &SyntaxError{msg: msg, Offset: int64(off)}
	}
}

func (c *checker) eof(off int) { c.fail("unexpected end of JSON input", off) }

// badChar reports the character at p, where describes what was expected.
func (c *checker) badChar(p int, where string) {
	_, n := utf8.DecodeRune(c.data[p:])
	c.fail("invalid character "+quoteRune(c.data[p:])+" "+where, p+n)
}

// quoteRune is jsonwire.QuoteRune: the character at the start of b quoted,
// a byte that does not begin one as '\xNN'.
func quoteRune(b []byte) string {
	r, n := utf8.DecodeRune(b)
	if r == utf8.RuneError && n == 1 {
		return `'\x` + strconv.FormatUint(uint64(b[0]), 16) + `'`
	}
	return strconv.QuoteRune(r)
}

func (c *checker) ws(p int) int {
	for p < len(c.data) && isSpace(c.data[p]) {
		p++
	}
	return p
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\r' || c == '\n'
}

func isDigit(c byte) bool { return '0' <= c && c <= '9' }

func isHex(c byte) bool {
	return isDigit(c) || 'a' <= c && c <= 'f' || 'A' <= c && c <= 'F'
}

// value checks the value at p and answers where it ends; -1 after an error.
// It walks nested objects and arrays with a stack of its own, not by
// recursion, so any depth is checked (and past maxDepth refused) in every
// engine.
func (c *checker) value(p int) int {
	var stack []byte
	for {
		if p < len(c.data) && c.data[p] <= ' ' {
			p = c.ws(p)
		}
		if p >= len(c.data) {
			c.eof(len(c.data))
			return -1
		}
		done := false
		switch b := c.data[p]; {
		case b == '{' || b == '[':
			if len(stack)+1 > maxDepth {
				c.fail("exceeded max depth", p+1)
				return -1
			}
			p++
			if p < len(c.data) && c.data[p] <= ' ' {
				p = c.ws(p)
			}
			if p >= len(c.data) {
				c.eof(p)
				return -1
			}
			if b == '{' && c.data[p] == '}' || b == '[' && c.data[p] == ']' {
				p++
				done = true
				break
			}
			stack = append(stack, b)
			if b == '{' {
				if p = c.key(p); p < 0 {
					return -1
				}
			}
		case b == '"':
			p = c.str(p)
			done = true
		case b == '-' || isDigit(b):
			p = c.number(p)
			done = true
		case b == 't':
			p = c.literal(p, "true")
			done = true
		case b == 'f':
			p = c.literal(p, "false")
			done = true
		case b == 'n':
			p = c.literal(p, "null")
			done = true
		default:
			c.badChar(p, "looking for beginning of value")
			return -1
		}
		if p < 0 {
			return -1
		}
		// After a value: a separator or the end of the containers it closes.
		for done {
			if len(stack) == 0 {
				return p
			}
			if p < len(c.data) && c.data[p] <= ' ' {
				p = c.ws(p)
			}
			if p >= len(c.data) {
				c.eof(p)
				return -1
			}
			top, ch := stack[len(stack)-1], c.data[p]
			switch {
			case ch == ',':
				done = false
				if top == '{' {
					if p = c.key(c.ws(p + 1)); p < 0 {
						return -1
					}
				} else {
					p++
				}
			case top == '[' && ch == ']' || top == '{' && ch == '}':
				stack = stack[:len(stack)-1]
				p++
			case top == '[':
				c.badChar(p, "after array element")
				return -1
			default:
				c.badChar(p, "after object key:value pair")
				return -1
			}
		}
	}
}

// key checks an object key at p and the ':' after it, and answers where the
// value may start.
func (c *checker) key(p int) int {
	if p >= len(c.data) {
		c.eof(p)
		return -1
	}
	if c.data[p] != '"' {
		c.badChar(p, "looking for beginning of object key string")
		return -1
	}
	if p = c.str(p); p < 0 {
		return -1
	}
	p = c.ws(p)
	if p >= len(c.data) {
		c.eof(p)
		return -1
	}
	if c.data[p] != ':' {
		c.badChar(p, "after object key")
		return -1
	}
	return p + 1
}

func (c *checker) str(p int) int {
	i := p + 1
	for i < len(c.data) {
		b := c.data[i]
		switch {
		case b == '"':
			return i + 1
		case b < ' ':
			c.badChar(i, "in string")
			return -1
		case b != '\\':
			i++
			continue
		}
		if i+1 >= len(c.data) {
			c.eof(i)
			return -1
		}
		switch e := c.data[i+1]; e {
		case '"', '\\', '/', 'b', 'f', 'n', 'r', 't':
			i += 2
		case 'u':
			for k := 2; k < 6; k++ {
				if i+k >= len(c.data) {
					c.eof(i)
					return -1
				}
				if !isHex(c.data[i+k]) {
					end := i + 6
					if end > len(c.data) {
						end = len(c.data)
					}
					c.badEscape(i, end)
					return -1
				}
			}
			i += 6
		default:
			_, n := utf8.DecodeRune(c.data[i+1:])
			c.badEscape(i, i+1+n)
			return -1
		}
	}
	c.eof(i)
	return -1
}

// badEscape reports the escape sequence data[from:to] (jsonwire's
// InvalidTextError): one character quoted, else in backquotes unless it holds
// a backquote, a space or a character that does not print.
func (c *checker) badEscape(from, to int) {
	what := string(c.data[from:to])
	var q string
	switch {
	case utf8.RuneCountInString(what) == 1:
		q = quoteRune([]byte(what))
	case strings.ContainsFunc(what, func(r rune) bool {
		return r == '`' || r == utf8.RuneError || unicode.IsSpace(r) || !unicode.IsPrint(r)
	}):
		q = strconv.Quote(what)
	default:
		q = "`" + what + "`"
	}
	c.fail("invalid escape sequence "+q+" in string", to)
}

func (c *checker) digits(p int) int {
	for p < len(c.data) && isDigit(c.data[p]) {
		p++
	}
	return p
}

func (c *checker) number(start int) int {
	p := start
	if c.data[p] == '-' {
		p++
	}
	if p >= len(c.data) {
		c.eof(start)
		return -1
	}
	switch {
	case c.data[p] == '0':
		p++
	case isDigit(c.data[p]):
		p = c.digits(p)
	default:
		c.badChar(p, "in numeric literal")
		return -1
	}
	if p < len(c.data) && c.data[p] == '.' {
		p++
		if p >= len(c.data) {
			c.eof(start)
			return -1
		}
		if !isDigit(c.data[p]) {
			c.badChar(p, "in numeric literal")
			return -1
		}
		p = c.digits(p)
	}
	if p < len(c.data) && (c.data[p] == 'e' || c.data[p] == 'E') {
		p++
		if p < len(c.data) && (c.data[p] == '+' || c.data[p] == '-') {
			p++
		}
		if p >= len(c.data) {
			c.eof(start)
			return -1
		}
		if !isDigit(c.data[p]) {
			c.badChar(p, "in numeric literal")
			return -1
		}
		p = c.digits(p)
	}
	return p
}

func (c *checker) literal(p int, lit string) int {
	for k := 0; k < len(lit); k++ {
		if p+k >= len(c.data) {
			c.eof(len(c.data))
			return -1
		}
		if c.data[p+k] != lit[k] {
			c.badChar(p+k, "in literal "+lit+" (expecting "+strconv.QuoteRune(rune(lit[k]))+")")
			return -1
		}
	}
	return p + len(lit)
}

// checkValid checks that data is one JSON value, space around it allowed.
func checkValid(data []byte) error {
	c := &checker{data: data}
	end := c.value(0)
	if end >= 0 {
		end = c.ws(end)
		if end < len(data) {
			c.badChar(end, "after top-level value")
		}
	}
	if c.err != nil {
		return c.err
	}
	return nil
}

func Valid(data []byte) bool { return checkValid(data) == nil }

// ----- indenting and compacting: indent.go -----

func Compact(dst *bytes.Buffer, src []byte) error { return compact(dst, src, false) }

func compact(dst *bytes.Buffer, src []byte, escape bool) error {
	if err := checkValid(src); err != nil {
		return err
	}
	var out []byte
	inStr := false
	for i := 0; i < len(src); i++ {
		c := src[i]
		if inStr {
			out = append(out, c)
			if c == '\\' && i+1 < len(src) {
				i++
				out = append(out, src[i])
			} else if c == '"' {
				inStr = false
			}
			continue
		}
		if isSpace(c) {
			continue
		}
		if c == '"' {
			inStr = true
		}
		if escape && (c == '<' || c == '>' || c == '&') {
			out = append(out, '\\', 'u', '0', '0', hexDigits[c>>4], hexDigits[c&0xF])
			continue
		}
		out = append(out, c)
	}
	dst.Write(out)
	return nil
}

func Indent(dst *bytes.Buffer, src []byte, prefix, indent string) error {
	if err := checkValid(src); err != nil {
		return err
	}
	var out []byte
	depth := 0
	needIndent := false
	inStr := false
	newline := func() {
		out = append(out, '\n')
		out = append(out, prefix...)
		for i := 0; i < depth; i++ {
			out = append(out, indent...)
		}
	}
	for i := 0; i < len(src); i++ {
		c := src[i]
		if inStr {
			out = append(out, c)
			if c == '\\' && i+1 < len(src) {
				i++
				out = append(out, src[i])
			} else if c == '"' {
				inStr = false
			}
			continue
		}
		if isSpace(c) {
			continue
		}
		if needIndent && c != ']' && c != '}' {
			needIndent = false
			depth++
			newline()
		}
		switch c {
		case '"':
			inStr = true
			out = append(out, c)
		case '{', '[':
			needIndent = true
			out = append(out, c)
		case ',':
			out = append(out, c)
			newline()
		case ':':
			out = append(out, ':', ' ')
		case '}', ']':
			if needIndent {
				needIndent = false
			} else {
				depth--
				newline()
			}
			out = append(out, c)
		default:
			out = append(out, c)
		}
	}
	dst.Write(out)
	return nil
}

func HTMLEscape(dst *bytes.Buffer, src []byte) {
	var out []byte
	for i := 0; i < len(src); i++ {
		c := src[i]
		if c == '<' || c == '>' || c == '&' {
			out = append(out, '\\', 'u', '0', '0', hexDigits[c>>4], hexDigits[c&0xF])
		} else {
			out = append(out, c)
		}
	}
	dst.Write(out)
}

// ----- decoding -----

type decodeState struct {
	data       []byte
	off        int
	savedError error
	useNumber  bool
	disallow   bool
	errStruct  string
	errField   []string
}

func Unmarshal(data []byte, v any) error {
	if err := checkValid(data); err != nil {
		return err
	}
	d := &decodeState{data: data}
	return d.unmarshal(v)
}

func (d *decodeState) unmarshal(v any) error {
	kind := goRfKind(v)
	if kind == "nil" || kind == "ptr" && goRfIsNil(v) {
		ty := ""
		if kind != "nil" {
			ty = fmt.Sprintf("%T", v)
		}
		return &InvalidUnmarshalError{Type: ty}
	}
	d.skipSpace()
	if kind == "struct" {
		d.intoStruct(v, structType(v))
	} else if kind == "map" {
		d.mapValue(typeTextOf(v), v)
	} else if kind == "ptr" {
		cur := goRfDeref(v)
		t := typeTextOf(cur)
		nv := d.value(t, cur)
		goRfStore(v, nv)
	} else {
		return &InvalidUnmarshalError{Type: fmt.Sprintf("%T", v)}
	}
	return d.savedError
}

// typeTextOf is the type text a decode into cur's variable builds: the
// declared type the value carries, or interface{} when it carries none.
func typeTextOf(cur interface{}) string {
	switch k := goRfKind(cur); k {
	case "nil":
		return "interface{}"
	case "struct":
		return structType(cur)
	case "slice":
		et := goRfElemTy(cur)
		if et == "" {
			et = "interface{}"
		}
		return "[]" + et
	case "map":
		if mt := goRfMapTy(cur); mt != "" {
			return mt
		}
		return "map[string]interface{}"
	case "array":
		return fmt.Sprintf("[%d]interface{}", listLen(cur))
	case "ptr":
		return "*" + typeTextOf(goRfDeref(cur))
	default:
		return k
	}
}

// structType is the type text of struct value v: T for a type of this
// program, pkg.T for one of a package.
func structType(v interface{}) string {
	n := fmt.Sprintf("%T", v)
	if strings.HasPrefix(n, "main.") {
		return n[5:]
	}
	return n
}

func (d *decodeState) saveError(err error) {
	if d.savedError == nil {
		d.savedError = err
	}
}

func (d *decodeState) typeError(what, t string) { d.typeErrorWith(what, t, nil) }

func (d *decodeState) typeErrorWith(what, t string, err error) {
	e := &UnmarshalTypeError{Value: what, Type: typeName(t), Offset: int64(d.off), Err: err}
	if len(d.errField) > 0 {
		e.Struct = d.errStruct
		e.Field = strings.Join(d.errField, ".")
	}
	d.saveError(e)
}

func (d *decodeState) skipSpace() {
	for d.off < len(d.data) && isSpace(d.data[d.off]) {
		d.off++
	}
}

// skipValue steps over one well-formed value and answers where it began.
func (d *decodeState) skipValue() int {
	d.skipSpace()
	start := d.off
	if d.off >= len(d.data) {
		return start
	}
	switch c := d.data[d.off]; c {
	case '{', '[':
		depth := 0
		inStr := false
		for d.off < len(d.data) {
			c := d.data[d.off]
			d.off++
			if inStr {
				if c == '\\' {
					d.off++
				} else if c == '"' {
					inStr = false
				}
				continue
			}
			switch c {
			case '"':
				inStr = true
			case '{', '[':
				depth++
			case '}', ']':
				depth--
			}
			if depth == 0 {
				break
			}
		}
	case '"':
		d.off++
		for d.off < len(d.data) && d.data[d.off] != '"' {
			if d.data[d.off] == '\\' {
				d.off++
			}
			d.off++
		}
		d.off++
	default:
		for d.off < len(d.data) && !isSpace(d.data[d.off]) && strings.IndexByte(",]}:", d.data[d.off]) < 0 {
			d.off++
		}
	}
	return start
}

// literalKind names the JSON value at the current offset the way a type
// error does.
func (d *decodeState) literalKind() string {
	switch d.data[d.off] {
	case '{':
		return "object"
	case '[':
		return "array"
	case '"':
		return "string"
	case 't', 'f':
		return "bool"
	case 'n':
		return "null"
	}
	return "number"
}

// value decodes the value at the current offset as type text t; cur is the
// value the destination holds, which an object or array decodes into.
func (d *decodeState) value(t string, cur interface{}) interface{} {
	d.skipSpace()
	if d.off >= len(d.data) {
		return cur
	}
	if t == "" || t == "interface{}" || t == "any" || strings.HasPrefix(t, "interface{") {
		if d.data[d.off] == 'n' {
			d.off += 4
			return nil
		}
		return d.generic()
	}
	if d.data[d.off] == 'n' {
		d.off += 4
		if strings.HasPrefix(t, "[]") || strings.HasPrefix(t, "map[") || strings.HasPrefix(t, "*") {
			return goRfZero(t)
		}
		return cur
	}
	if strings.HasPrefix(t, "*") {
		et := t[1:]
		var ecur interface{}
		if cur != nil && !goRfIsNil(cur) {
			ecur = goRfDeref(cur)
		} else {
			ecur = goRfZero(et)
		}
		v := d.value(et, ecur)
		if goRfKind(v) == "struct" {
			return v
		}
		return ptrTo(v)
	}
	if strings.HasPrefix(t, "[]") {
		return d.slice(t, cur)
	}
	if strings.HasPrefix(t, "[") {
		return d.array(t, cur)
	}
	if strings.HasPrefix(t, "map[") {
		return d.mapValue(t, cur)
	}
	switch t {
	case "string":
		if d.data[d.off] != '"' {
			d.typeError(d.literalKind(), t)
			d.skipValue()
			return cur
		}
		return d.str()
	case "bool":
		c := d.data[d.off]
		if c != 't' && c != 'f' {
			d.typeError(d.literalKind(), t)
			d.skipValue()
			return cur
		}
		d.skipValue()
		return c == 't'
	case "float64", "float32", "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr", "byte", "rune":
		return d.number(t, cur)
	case "Number", "json.Number":
		if c := d.data[d.off]; c == '"' || c == '-' || c >= '0' && c <= '9' {
			if c == '"' {
				return Number(d.str())
			}
			start := d.skipValue()
			return Number(string(d.data[start:d.off]))
		}
		d.typeError(d.literalKind(), t)
		d.skipValue()
		return cur
	case "RawMessage", "json.RawMessage":
		start := d.skipValue()
		return RawMessage(append([]byte{}, d.data[start:d.off]...))
	}
	// A named type: what it is decides how it decodes.
	target := cur
	if target == nil || goRfKind(target) != "struct" && goRfKind(goRfZero(t)) == "struct" {
		target = goRfZero(t)
	}
	switch goRfKind(target) {
	case "struct":
		if u, ok := target.(Unmarshaler); ok {
			start := d.skipValue()
			if err := u.UnmarshalJSON(d.data[start:d.off]); err != nil {
				d.saveError(err)
			}
			return target
		}
		if u, ok := target.(textUnmarshaler); ok && d.data[d.off] == '"' {
			if err := u.UnmarshalText([]byte(d.str())); err != nil {
				d.saveError(err)
			}
			return target
		}
		if d.data[d.off] != '{' {
			d.typeError(d.literalKind(), t)
			d.skipValue()
			return target
		}
		d.intoStruct(target, t)
		return target
	case "nil":
		d.skipValue()
		return cur
	}
	// A defined type over a basic or composite one decodes as that type, which
	// is how this runtime holds its values.
	return d.value(typeTextOf(target), target)
}

func ptrTo(v interface{}) interface{} {
	x := v
	return &x
}

func (d *decodeState) str() string {
	start := d.off
	d.off++
	var out []byte
	for d.off < len(d.data) && d.data[d.off] != '"' {
		c := d.data[d.off]
		if c != '\\' {
			out = append(out, c)
			d.off++
			continue
		}
		d.off++
		if d.off >= len(d.data) {
			break
		}
		switch e := d.data[d.off]; e {
		case 'b':
			out = append(out, '\b')
		case 'f':
			out = append(out, '\f')
		case 'n':
			out = append(out, '\n')
		case 'r':
			out = append(out, '\r')
		case 't':
			out = append(out, '\t')
		case 'u':
			if d.off+5 > len(d.data) {
				d.off = len(d.data)
				continue
			}
			r := hex4(d.data[d.off+1 : d.off+5])
			d.off += 4
			if r >= 0xD800 && r < 0xDC00 && d.off+6 < len(d.data) && d.data[d.off+1] == '\\' && d.data[d.off+2] == 'u' {
				r2 := hex4(d.data[d.off+3 : d.off+7])
				if r2 >= 0xDC00 && r2 < 0xE000 {
					r = (r-0xD800)<<10 + (r2 - 0xDC00) + 0x10000
					d.off += 6
				} else {
					r = utf8.RuneError
				}
			} else if r >= 0xD800 && r < 0xE000 {
				r = utf8.RuneError
			}
			out = utf8.AppendRune(out, r)
		default:
			out = append(out, e)
		}
		d.off++
	}
	d.off++
	_ = start
	// Invalid UTF-8 in the input is kept as U+FFFD, as decode.go's unquote does.
	if !utf8.Valid(out) {
		var fixed []byte
		for i := 0; i < len(out); {
			r, w := utf8.DecodeRune(out[i:])
			fixed = utf8.AppendRune(fixed, r)
			i += w
		}
		out = fixed
	}
	return string(out)
}

func hex4(b []byte) rune {
	var r rune
	for _, c := range b {
		var v rune
		switch {
		case '0' <= c && c <= '9':
			v = rune(c - '0')
		case 'a' <= c && c <= 'f':
			v = rune(c - 'a' + 10)
		default:
			v = rune(c - 'A' + 10)
		}
		r = r*16 + v
	}
	return r
}

func (d *decodeState) number(t string, cur interface{}) interface{} {
	c := d.data[d.off]
	if c != '-' && (c < '0' || c > '9') {
		d.typeError(d.literalKind(), t)
		d.skipValue()
		return cur
	}
	start := d.skipValue()
	s := string(d.data[start:d.off])
	switch t {
	case "float64", "float32":
		bits := 64
		if t == "float32" {
			bits = 32
		}
		f, err := strconv.ParseFloat(s, bits)
		if err != nil {
			d.typeError("number "+s, t)
			return cur
		}
		return goRfConv(t, f)
	case "uint", "uint8", "uint16", "uint32", "uint64", "uintptr", "byte":
		n, err := strconv.ParseUint(s, 10, intBits(t))
		if err != nil {
			d.typeError("number "+s, t)
			return cur
		}
		return goRfConv(t, n)
	}
	n, err := strconv.ParseInt(s, 10, intBits(t))
	if err != nil {
		d.typeError("number "+s, t)
		return cur
	}
	return goRfConv(t, n)
}

// intBits is the width of integer type t.
func intBits(t string) int {
	switch t {
	case "int8", "uint8", "byte":
		return 8
	case "int16", "uint16":
		return 16
	case "int32", "uint32", "rune":
		return 32
	}
	return 64
}

// generic decodes into interface{}: map[string]interface{}, []interface{},
// float64 (or Number), string, bool or nil.
func (d *decodeState) generic() interface{} {
	d.skipSpace()
	if d.off >= len(d.data) {
		return nil
	}
	switch c := d.data[d.off]; c {
	case '{':
		m := make(map[string]interface{})
		d.off++
		for {
			d.skipSpace()
			if d.data[d.off] == '}' {
				d.off++
				return m
			}
			k := d.str()
			d.skipSpace()
			d.off++
			m[k] = d.generic()
			d.skipSpace()
			if d.data[d.off] == ',' {
				d.off++
			}
		}
	case '[':
		a := []interface{}{}
		d.off++
		for {
			d.skipSpace()
			if d.data[d.off] == ']' {
				d.off++
				return a
			}
			a = append(a, d.generic())
			d.skipSpace()
			if d.data[d.off] == ',' {
				d.off++
			}
		}
	case '"':
		return d.str()
	case 't':
		d.off += 4
		return true
	case 'f':
		d.off += 5
		return false
	case 'n':
		d.off += 4
		return nil
	}
	start := d.skipValue()
	s := string(d.data[start:d.off])
	if d.useNumber {
		return Number(s)
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		d.typeError("number "+s, "float64")
	}
	return f
}

// elemType is the element type text of "[]E" or "[N]E".
func elemType(t string) string {
	return t[strings.IndexByte(t, ']')+1:]
}

// mapTypes splits "map[K]V" at the bracket that closes K.
func mapTypes(t string) (string, string) {
	depth := 0
	for i := 3; i < len(t); i++ {
		switch t[i] {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return t[4:i], t[i+1:]
			}
		}
	}
	return "string", "interface{}"
}

func (d *decodeState) slice(t string, cur interface{}) interface{} {
	et := elemType(t)
	if d.data[d.off] == '"' && (et == "byte" || et == "uint8") {
		b, err := unbase64(d.str())
		if err != nil {
			d.saveError(err)
			return cur
		}
		els := make([]interface{}, len(b))
		for i := range b {
			els[i] = b[i]
		}
		return goRfMakeSlice(t, els)
	}
	if d.data[d.off] != '[' {
		d.typeError(d.literalKind(), t)
		d.skipValue()
		return cur
	}
	var old []interface{}
	if cur != nil && goRfKind(cur) == "slice" {
		e := goRfElems(cur)
		for i := 0; i < len(e); i++ {
			old = append(old, e[i])
		}
	}
	els := []interface{}{}
	d.off++
	for i := 0; ; i++ {
		d.skipSpace()
		if d.data[d.off] == ']' {
			d.off++
			break
		}
		var ec interface{}
		if i < len(old) {
			ec = old[i]
		}
		els = append(els, d.value(et, ec))
		d.skipSpace()
		if d.data[d.off] == ',' {
			d.off++
		}
	}
	return goRfMakeSlice(t, els)
}

func (d *decodeState) array(t string, cur interface{}) interface{} {
	et := elemType(t)
	n, _ := strconv.Atoi(t[1:strings.IndexByte(t, ']')])
	if d.data[d.off] != '[' {
		d.typeError(d.literalKind(), t)
		d.skipValue()
		return cur
	}
	var els []interface{}
	if cur != nil && goRfKind(cur) == "array" {
		e := goRfElems(cur)
		for i := 0; i < len(e); i++ {
			els = append(els, e[i])
		}
		n = len(els)
	}
	d.off++
	i := 0
	for ; ; i++ {
		d.skipSpace()
		if d.data[d.off] == ']' {
			d.off++
			break
		}
		if i < n && i < len(els) {
			els[i] = d.value(et, els[i])
		} else {
			d.skipValue()
		}
		d.skipSpace()
		if d.data[d.off] == ',' {
			d.off++
		}
	}
	for ; i < len(els); i++ {
		els[i] = goRfZero(et)
	}
	return goRfMakeSlice(t, els)
}

func (d *decodeState) mapValue(t string, cur interface{}) interface{} {
	if d.data[d.off] != '{' {
		d.typeError(d.literalKind(), t)
		d.skipValue()
		return cur
	}
	kt, vt := mapTypes(t)
	// A map is its own pointer here, so a decode fills the map the
	// destination holds, a nil one included (goRfMapSet makes it non-nil).
	m := cur
	if goRfKind(m) != "map" {
		m = goRfMakeMap(t, goRfZero(vt))
	}
	d.off++
	for {
		d.skipSpace()
		if d.data[d.off] == '}' {
			d.off++
			return m
		}
		ks := d.str()
		d.skipSpace()
		d.off++
		var key interface{}
		ok := true
		switch kt {
		case "string":
			key = ks
		case "int", "int8", "int16", "int32", "int64":
			n, err := strconv.ParseInt(ks, 10, intBits(kt))
			ok = err == nil
			key = goRfConv(kt, n)
		case "uint", "uint8", "uint16", "uint32", "uint64", "uintptr":
			n, err := strconv.ParseUint(ks, 10, intBits(kt))
			ok = err == nil
			key = goRfConv(kt, n)
		default:
			key = goRfConv(kt, ks)
		}
		if ok {
			goRfMapSet(m, key, d.value(vt, nil))
		} else {
			d.typeError("number "+ks, kt)
			d.skipValue()
		}
		d.skipSpace()
		if d.data[d.off] == ',' {
			d.off++
		}
	}
}

// intoStruct decodes an object into the fields of struct value s, named t:
// a key matches a field's name exactly, else without regard to case; a key
// no field has is skipped.
func (d *decodeState) intoStruct(s interface{}, t string) {
	if u, ok := s.(Unmarshaler); ok {
		start := d.skipValue()
		if err := u.UnmarshalJSON(d.data[start:d.off]); err != nil {
			d.saveError(err)
		}
		return
	}
	if d.data[d.off] != '{' {
		d.typeError(d.literalKind(), t)
		d.skipValue()
		return
	}
	fields := fieldsOf(s)
	d.off++
	for {
		d.skipSpace()
		if d.data[d.off] == '}' {
			d.off++
			return
		}
		key := d.str()
		d.skipSpace()
		d.off++
		fi := -1
		for i := range fields {
			if fields[i].name == key {
				fi = i
				break
			}
		}
		if fi < 0 {
			for i := range fields {
				if strings.EqualFold(fields[i].name, key) {
					fi = i
					break
				}
			}
		}
		if fi < 0 {
			if d.disallow {
				d.saveError(fmt.Errorf("json: unknown field %q", key))
			}
			d.skipValue()
		} else {
			f := fields[fi]
			owner := s
			for _, i := range f.path[:len(f.path)-1] {
				owner = goRfFields(owner)[3][i]
			}
			idx := f.path[len(f.path)-1]
			savedS, savedF := d.errStruct, d.errField
			if len(d.errField) == 0 {
				d.errStruct = strings.TrimPrefix(t, "main.")
				if i := strings.LastIndexByte(d.errStruct, '.'); i >= 0 {
					d.errStruct = d.errStruct[i+1:]
				}
			}
			d.errField = append(append([]string{}, d.errField...), f.name)
			cur := goRfFields(owner)[3][idx]
			var v interface{}
			if f.quoted && isScalarKind(goRfKind(cur)) {
				v = d.quoted(f.typ, cur)
			} else {
				v = d.value(f.typ, cur)
			}
			goRfSetField(owner, idx, v)
			d.errStruct, d.errField = savedS, savedF
		}
		d.skipSpace()
		if d.data[d.off] == ',' {
			d.off++
		}
	}
}

func isScalarKind(k string) bool {
	switch k {
	case "bool", "string", "float32", "float64", "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr":
		return true
	}
	return false
}

// quoted decodes a ,string field of scalar type t: a JSON string holding the
// field's literal (for a string field, a quoted JSON string), or null. The
// inner literal is checked before it is read; what does not fit is a type
// error, as encoding/json reports it.
func (d *decodeState) quoted(t string, cur interface{}) interface{} {
	c := d.data[d.off]
	if c == 'n' {
		d.skipValue()
		return cur
	}
	if c != '"' {
		d.typeError(d.literalKind(), t)
		d.skipValue()
		return cur
	}
	q := d.str()
	if q == "null" {
		return cur
	}
	switch k := goRfKind(cur); k {
	case "string":
		if q == "" || q[0] != '"' {
			bad := q
			if bad == "" {
				bad = " "
			}
			c := &checker{data: []byte(bad)}
			c.badChar(0, "looking for beginning of object key string")
			d.typeErrorWith("string", t, c.err)
			return cur
		}
		if err := checkValid([]byte(q)); err != nil {
			d.typeErrorWith("string", t, err)
			return cur
		}
		sub := &decodeState{data: []byte(q)}
		return sub.str()
	case "bool":
		if q != "true" && q != "false" {
			d.typeErrorWith("string "+strconv.Quote(q), t, strconv.ErrSyntax)
			return cur
		}
		return q == "true"
	}
	nc := &checker{data: []byte(q)}
	if q == "" || (q[0] != '-' && !isDigit(q[0])) || nc.number(0) != len(q) {
		d.typeError("number "+q, t)
		return cur
	}
	sub := &decodeState{data: []byte(q), errStruct: d.errStruct, errField: d.errField}
	v := sub.number(t, cur)
	if sub.savedError != nil {
		d.saveError(sub.savedError)
	}
	return v
}

// ----- streams -----

type Encoder struct {
	w          io.Writer
	prefix     string
	indent     string
	escapeHTML bool
}

func NewEncoder(w io.Writer) *Encoder { return &Encoder{w: w, escapeHTML: true} }

func (enc *Encoder) SetIndent(prefix, indent string) {
	enc.prefix = prefix
	enc.indent = indent
}

func (enc *Encoder) SetEscapeHTML(on bool) { enc.escapeHTML = on }

func (enc *Encoder) Encode(v any) error {
	e := &encodeState{escapeHTML: enc.escapeHTML}
	if err := e.value(v, false); err != nil {
		return err
	}
	b := e.buf
	if enc.prefix != "" || enc.indent != "" {
		var buf bytes.Buffer
		if err := Indent(&buf, b, enc.prefix, enc.indent); err != nil {
			return err
		}
		b = buf.Bytes()
	}
	b = append(b, '\n')
	_, err := enc.w.Write(b)
	return err
}

type Decoder struct {
	r         io.Reader
	buf       []byte
	read      bool
	useNumber bool
	disallow  bool
}

func NewDecoder(r io.Reader) *Decoder { return &Decoder{r: r} }

func (dec *Decoder) UseNumber()             { dec.useNumber = true }
func (dec *Decoder) DisallowUnknownFields() { dec.disallow = true }

func (dec *Decoder) fill() {
	if !dec.read {
		b, _ := io.ReadAll(dec.r)
		dec.buf = b
		dec.read = true
	}
}

// More reports whether another value follows in the input.
func (dec *Decoder) More() bool {
	dec.fill()
	i := 0
	for i < len(dec.buf) && isSpace(dec.buf[i]) {
		i++
	}
	return i < len(dec.buf) && dec.buf[i] != ']' && dec.buf[i] != '}'
}

// Decode reads the next value of the input into v: io.EOF when only space is
// left, io.ErrUnexpectedEOF when the value is cut off.
func (dec *Decoder) Decode(v any) error {
	dec.fill()
	c := &checker{data: dec.buf}
	if c.ws(0) == len(dec.buf) {
		return io.EOF
	}
	n := c.value(0)
	if n < 0 {
		if c.err.msg == "unexpected end of JSON input" {
			return io.ErrUnexpectedEOF
		}
		return c.err
	}
	data := dec.buf[:n]
	dec.buf = dec.buf[n:]
	d := &decodeState{data: data, useNumber: dec.useNumber, disallow: dec.disallow}
	return d.unmarshal(v)
}
