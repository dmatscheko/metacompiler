package abnf

import (
	"fmt"
	"strings"
)

// Python's bytes and bytearray: arrays of byte values carrying kind 17 and 18,
// the way a tuple carries pyTupKind. Indexing, len and iteration are the list
// machinery's; what differs - type, repr, ==, hashing, immutability, the
// methods - asks pyIsBytes. Twins: pyBytes / isBytes in
// languages/python-interpreter.abnf and lib/python-rt.metajs.
const (
	pyBytesKind = 17
	pyBArrKind  = 18
)

// pyIsBytes says whether v is bytes or a bytearray.
func pyIsBytes(v interface{}) bool {
	a, ok := v.(*jsArray)
	return ok && (a.kind == pyBytesKind || a.kind == pyBArrKind)
}

// pySeqCls is the kind == compares: a bytearray equals bytes of the same
// bytes, and neither equals a list.
func pySeqCls(a *jsArray) int {
	if a.kind == pyBArrKind {
		return pyBytesKind
	}
	return a.kind
}

// pySeqTName is the builtin type name of an array.
func pySeqTName(a *jsArray) string {
	switch a.kind {
	case pyTupKind:
		return "tuple"
	case pyBytesKind:
		return "bytes"
	case pyBArrKind:
		return "bytearray"
	}
	return "list"
}

// pySeqKindCh is the kind letter + and the comparisons pair operands by: l a
// list, u a tuple, b bytes or a bytearray.
func pySeqKindCh(a *jsArray) byte {
	switch pySeqCls(a) {
	case pyTupKind:
		return 'u'
	case pyBytesKind:
		return 'b'
	}
	return 'l'
}

// pySeqLike gives out the kind of o: a slice of a tuple is a tuple, of bytes
// bytes.
func pySeqLike(o, out *jsArray) *jsArray {
	if o.kind >= pyTupKind && o.kind <= pyBArrKind {
		out.kind = o.kind
	}
	return out
}

// pyNoSliceDel is del of a slice of an immutable sequence: CPython's
// mapping-protocol text, "does not" where an index deletion says "doesn't".
func (rt *jsrt) pyNoSliceDel(a *jsArray) {
	rt.pyRaise("TypeError", "'"+pySeqTName(a)+"' object does not support item deletion")
}

// pyBytesRepr is b'...' with the quote CPython picks (' unless the bytes hold
// a ' and no "), \t \n \r and \\ escaped, and \xNN outside printable ASCII.
func pyBytesRepr(a *jsArray) string {
	q := byte('\'')
	hasS, hasD := false, false
	for _, e := range a.elems {
		switch pyByteOf(e) {
		case '\'':
			hasS = true
		case '"':
			hasD = true
		}
	}
	if hasS && !hasD {
		q = '"'
	}
	var sb strings.Builder
	sb.WriteString("b")
	sb.WriteByte(q)
	for _, e := range a.elems {
		c := pyByteOf(e)
		switch {
		case c == q || c == '\\':
			sb.WriteByte('\\')
			sb.WriteByte(c)
		case c == '\t':
			sb.WriteString(`\t`)
		case c == '\n':
			sb.WriteString(`\n`)
		case c == '\r':
			sb.WriteString(`\r`)
		case c < 32 || c > 126:
			fmt.Fprintf(&sb, `\x%02x`, c)
		default:
			sb.WriteByte(c)
		}
	}
	sb.WriteByte(q)
	if a.kind == pyBArrKind {
		return "bytearray(" + sb.String() + ")"
	}
	return sb.String()
}

func pyByteOf(e interface{}) byte {
	switch t := e.(type) {
	case float64:
		return byte(int(t))
	case bool:
		if t {
			return 1
		}
	}
	return 0
}

// pyBytesToS is the bytes as the str methods see them: printable ASCII and the
// controls stay themselves, U+001C..U+001F (whitespace to str, not to bytes)
// and every byte past 127 move to U+F700 + byte, which no str method treats as
// a letter, digit or space. pySToBytes undoes it.
func pyBytesToS(a *jsArray) string {
	us := make([]uint16, len(a.elems))
	for i, e := range a.elems {
		c := uint16(pyByteOf(e))
		if c > 127 || (c >= 28 && c <= 31) {
			c += 0xF700
		}
		us[i] = c
	}
	return strFromUnits(us)
}

func pySToBytes(s string, kind int) *jsArray {
	out := &jsArray{kind: kind}
	for _, c := range strUnits(s) {
		if c >= 0xF700 {
			c -= 0xF700
		}
		out.elems = append(out.elems, float64(c))
	}
	return out
}

// pyByteVal is a value stored into a bytearray: an int in range(0, 256).
func (rt *jsrt) pyByteVal(v interface{}) float64 {
	if tn := pyTypeName(rt, v); tn != "int" && tn != "bool" {
		rt.pyRaise("TypeError", "'"+tn+"' object cannot be interpreted as an integer")
	}
	b := rt.toNumber(v)
	if b < 0 || b > 255 {
		rt.pyRaise("ValueError", "byte must be in range(0, 256)")
	}
	return b
}

// pyBArrExtend is bytearray += x and bytearray.extend(x): x's bytes appended
// in place.
func (rt *jsrt) pyBArrExtend(a *jsArray, x interface{}) *jsArray {
	if _, isStr := x.(string); isStr {
		rt.pyRaise("TypeError", "expected iterable of integers; got: 'str'")
	}
	src := append([]interface{}{}, rt.pyElemsOf(x)...)
	a.dropIdx()
	for _, e := range src {
		a.elems = append(a.elems, rt.pyByteVal(e))
	}
	return a
}

// pyBytesHas is x in b: an int is a member, a bytes-like value a substring.
func (rt *jsrt) pyBytesHas(b *jsArray, x interface{}) bool {
	if xa, ok := x.(*jsArray); ok && pyIsBytes(xa) {
		return strings.Contains(pyBytesToS(b), pyBytesToS(xa))
	}
	if tn := pyTypeName(rt, x); tn != "int" && tn != "bool" {
		rt.pyRaise("TypeError", "a bytes-like object is required, not '"+tn+"'")
	}
	c := rt.pyByteVal(x)
	for _, e := range b.elems {
		if rt.toNumber(e) == c {
			return true
		}
	}
	return false
}

// pyBytesNum is int(b) / float(b): the bytes read as ASCII text, and a
// ValueError naming the bytes' own repr.
func (rt *jsrt) pyBytesNum(name string, args []interface{}) (out interface{}) {
	b := args[0].(*jsArray)
	a2 := append([]interface{}{}, args...)
	a2[0] = pyBytesToS(b)
	defer func() {
		if p := recover(); p != nil {
			if th, ok := p.(*jsThrown); ok && rt.pyIsInstance(th.value, rt.pyExcClass("ValueError")) {
				m := rt.pyString(rt.pyExcArg0(th.value))
				if shown := rt.pyRepr(a2[0]); strings.HasSuffix(m, shown) {
					rt.pyRaise("ValueError", strings.TrimSuffix(m, shown)+pyBytesRepr(b))
				}
			}
			panic(p)
		}
	}()
	if name == "int" {
		return rt.pyIntNew(a2)
	}
	return rt.pyExt("js_pyfloat", a2[0])
}

// pyExcArg0 is an exception instance's first argument.
func (rt *jsrt) pyExcArg0(exc interface{}) interface{} {
	if o, ok := exc.(*jsObject); ok {
		if args, ok := o.props["args"].(*jsArray); ok && len(args.elems) > 0 {
			return args.elems[0]
		}
	}
	return jsUndef
}

// pyBytesPctArgs is the right operand of bytes % args with each bytes-like
// value as the str formatting sees it; a str has no place in bytes.
func (rt *jsrt) pyBytesPctArgs(r interface{}) interface{} {
	one := func(v interface{}) interface{} {
		if a, ok := v.(*jsArray); ok && pyIsBytes(a) {
			return pyBytesToS(a)
		}
		if _, isStr := v.(string); isStr {
			rt.pyRaise("TypeError", "%b requires a bytes-like object, or an object that implements __bytes__, not 'str'")
		}
		return v
	}
	if t, ok := r.(*jsArray); ok && t.kind == pyTupKind {
		out := &jsArray{kind: pyTupKind}
		for _, e := range t.elems {
			out.elems = append(out.elems, one(e))
		}
		return out
	}
	return one(r)
}

// pyBytesHex is b.hex(): two lowercase hex digits a byte.
func pyBytesHex(a *jsArray) string {
	var sb strings.Builder
	for _, e := range a.elems {
		fmt.Fprintf(&sb, "%02x", pyByteOf(e))
	}
	return sb.String()
}

// pyBytesFromHex is bytes.fromhex(s): pairs of hex digits, spaces between
// pairs skipped.
func (rt *jsrt) pyBytesFromHex(v interface{}, kind int) *jsArray {
	s, ok := v.(string)
	if !ok {
		rt.pyRaise("TypeError", "fromhex() argument must be str, not "+pyTypeName(rt, v))
	}
	digit := func(i int) int {
		if i >= len(s) {
			return -1
		}
		return strings.IndexByte("0123456789abcdef", strings.ToLower(s[i : i+1])[0])
	}
	out := &jsArray{kind: kind}
	for i := 0; i < len(s); {
		if s[i] == ' ' {
			i++
			continue
		}
		hi, lo := digit(i), digit(i+1)
		if hi < 0 || lo < 0 {
			at := i
			if hi >= 0 {
				at = i + 1
			}
			rt.pyRaise("ValueError", fmt.Sprintf("non-hexadecimal number found in fromhex() arg at position %d", at))
		}
		out.elems = append(out.elems, float64(hi*16+lo))
		i += 2
	}
	return out
}

// pyBytesStrMethod names the str methods a bytes value answers through
// pyBytesToS.
func pyBytesStrMethod(name string) bool {
	switch name {
	case "count", "find", "rfind", "index", "rindex", "startswith", "endswith", "split", "rsplit",
		"join", "strip", "lstrip", "rstrip", "replace", "upper", "lower", "swapcase", "capitalize", "title",
		"isdigit", "isalpha", "isalnum", "isspace", "islower", "isupper", "istitle", "isascii",
		"partition", "rpartition", "removeprefix", "removesuffix", "center", "ljust", "rjust", "zfill", "expandtabs":
		return true
	}
	return false
}

// pyBArrMutator names the mutators a bytearray adds.
func pyBArrMutator(name string) bool {
	switch name {
	case "append", "extend", "pop", "insert", "remove", "reverse", "clear", "copy":
		return true
	}
	return false
}

// pyIsBytesMethod is the method name set of bytes (kind 17) or a bytearray.
func pyIsBytesMethod(name string, kind int) bool {
	return name == "decode" || name == "hex" || pyBytesStrMethod(name) || (kind == pyBArrKind && pyBArrMutator(name))
}

// pyBytesArgS is one argument of a str-delegated bytes method, as the str
// method sees it. An int stands for its byte where CPython accepts one (the
// search methods), a tuple of bytes for each of its members (startswith).
func (rt *jsrt) pyBytesArgS(name string, v interface{}, first bool) interface{} {
	if a, ok := v.(*jsArray); ok && pyIsBytes(a) {
		return pyBytesToS(a)
	}
	tn := pyTypeName(rt, v)
	search := name == "count" || name == "find" || name == "rfind" || name == "index" || name == "rindex"
	if first && search && (tn == "int" || tn == "bool") {
		return pyBytesToS(&jsArray{elems: []interface{}{rt.pyByteVal(v)}})
	}
	if t, ok := v.(*jsArray); ok && first && (name == "startswith" || name == "endswith") && t.kind == pyTupKind {
		out := &jsArray{kind: pyTupKind}
		for _, e := range t.elems {
			out.elems = append(out.elems, rt.pyBytesArgS(name, e, false))
		}
		return out
	}
	if first && name == "join" {
		out := &jsArray{}
		for j, e := range rt.pyElemsOf(v) {
			ea, ok := e.(*jsArray)
			if !ok || !pyIsBytes(ea) {
				rt.pyRaise("TypeError", fmt.Sprintf("sequence item %d: expected a bytes-like object, %s found", j, pyTypeName(rt, e)))
			}
			out.elems = append(out.elems, pyBytesToS(ea))
		}
		return out
	}
	if tn == "str" {
		switch {
		case search:
			rt.pyRaise("TypeError", "argument should be integer or bytes-like object, not 'str'")
		case name == "startswith" || name == "endswith":
			rt.pyRaise("TypeError", name+" first arg must be bytes or a tuple of bytes, not str")
		}
		rt.pyRaise("TypeError", "a bytes-like object is required, not 'str'")
	}
	return v
}

// pyBytesBack is a str method's answer back as bytes of the given kind: a
// str, or a list or tuple of them; anything else is unchanged.
func pyBytesBack(r interface{}, kind int) interface{} {
	switch t := r.(type) {
	case string:
		return pySToBytes(t, kind)
	case *jsArray:
		out := &jsArray{kind: t.kind}
		for _, e := range t.elems {
			out.elems = append(out.elems, pyBytesBack(e, kind))
		}
		return out
	}
	return r
}

// pyBytesMethod is a method of bytes or a bytearray. Twin of pyBytesMethod in
// languages/python-interpreter.abnf and lib/python-rt.metajs.
func (rt *jsrt) pyBytesMethod(b *jsArray, name string, args []interface{}, kw interface{}) interface{} {
	keys, vals, hasKw := dictParts(kw)
	hasKw = hasKw && len(keys.elems) > 0
	switch {
	case name == "decode":
		if hasKw {
			args = rt.pyKwPos("decode", args, kw, []string{"encoding", "errors"}, 0)
		}
		return rt.pyBytesDecode(b.elems, argAt(args, 0))
	case name == "hex":
		return pyBytesHex(b)
	case b.kind == pyBArrKind && pyBArrMutator(name):
		switch name {
		case "append":
			b.dropIdx()
			b.elems = append(b.elems, rt.pyByteVal(argAt(args, 0)))
			return jsUndef
		case "extend":
			rt.pyBArrExtend(b, argAt(args, 0))
			return jsUndef
		case "insert":
			args = []interface{}{argAt(args, 0), rt.pyByteVal(argAt(args, 1))}
		case "remove":
			rv := rt.pyByteVal(argAt(args, 0))
			for i, e := range b.elems {
				if rt.toNumber(e) == rv {
					b.dropIdx()
					b.elems = append(b.elems[:i], b.elems[i+1:]...)
					return jsUndef
				}
			}
			rt.pyRaise("ValueError", "value not found in bytearray")
		case "pop":
			if len(b.elems) == 0 {
				rt.pyRaise("IndexError", "pop from empty bytearray")
			}
		}
		r := rt.pyListMethod(b, name, args, kw)
		if name == "copy" {
			r.(*jsArray).kind = pyBArrKind
		}
		return r
	case pyBytesStrMethod(name):
		sa := make([]interface{}, len(args))
		for i, a := range args {
			sa[i] = rt.pyBytesArgS(name, a, i == 0)
		}
		var skw interface{} = kw
		if hasKw {
			nk, nv := &jsArray{}, &jsArray{}
			for i, k := range keys.elems {
				dictAppend(nk, nv, k, rt.pyBytesArgS(name, vals.elems[i], false))
			}
			skw = &jsObject{props: map[string]interface{}{"__dict": true, "keys": nk, "vals": nv}}
		}
		s := pyBytesToS(b)
		if name == "index" || name == "rindex" {
			at := rt.pyExt("js_pymcall", s, strings.Replace(name, "index", "find", 1), &jsArray{elems: sa}, skw)
			if rt.toNumber(at) < 0 {
				rt.pyRaise("ValueError", "subsection not found")
			}
			return at
		}
		return pyBytesBack(rt.pyExt("js_pymcall", s, name, &jsArray{elems: sa}, skw), b.kind)
	}
	rt.pyRaise("AttributeError", "'"+pySeqTName(b)+"' object has no attribute '"+name+"'")
	return nil
}

func init() {
	rxExtraExterns = append(rxExtraExterns, func(rt *jsrt, m map[string]func(args []uint64) uint64) {
		// A bytes literal: the array of its byte values, marked. Twin of js_pyb
		// in lib/python-rt.metajs.
		m["js_pyb"] = func(a []uint64) uint64 {
			if arr, ok := rt.unwrap(a[0]).(*jsArray); ok {
				arr.kind = pyBytesKind
			}
			return a[0]
		}
	})
}
