package abnf

import (
	"fmt"
	"strings"
	"sync"
)

// Python subclasses of builtin types, the iteration protocol of an instance and
// bytes.decode - the GO TWIN half. The other two halves are in
// languages/lib/python-rt.metajs and languages/python-interpreter.abnf.
//
// An instance of `class S(str)` keeps its builtin value in a hidden __val slot
// (pyNewInstance), and the builtin TYPE in its resolution order answers a lookup
// with a DELEGATE that applies the builtin operation to that value: s.upper(),
// len(s), s + "c", s[0], str(s), iteration. Every dispatch site already asks the
// class for a dunder through pyLookup, so the delegates are found there, by
// super() and by hasattr alike, and a class's own definitions still win. Like
// CPython, a delegated method answers a plain builtin value (S("ab").upper() is
// a str). Delegates are made once per (type, name), so reading one twice gives
// the same function.

var (
	pyBDelMu      sync.Mutex
	pyBDelCache   = map[string]*hostFunc{}
	pyBDelMethods = map[*hostFunc]bool{}
)

// pyBuiltinDelegate is the delegate the builtin type tname answers for name.
func pyBuiltinDelegate(tname, name string) (*hostFunc, bool) {
	switch tname {
	case "str", "int", "float", "list", "dict":
	default:
		return nil, false
	}
	key := tname + "." + name
	pyBDelMu.Lock()
	defer pyBDelMu.Unlock()
	f, seen := pyBDelCache[key]
	if !seen {
		f = pyBDelMake(tname, name)
		pyBDelCache[key] = f
	}
	return f, f != nil
}

// pyIsBDelMethod says whether v is a delegate for one of a builtin type's
// named methods.
func pyIsBDelMethod(v interface{}) bool {
	hf, ok := v.(*hostFunc)
	if !ok {
		return false
	}
	pyBDelMu.Lock()
	defer pyBDelMu.Unlock()
	return pyBDelMethods[hf]
}

// pyStrDecode is str(object, encoding[, errors]): the decoding form, for a
// bytes value only.
func (rt *jsrt) pyStrDecode(v, enc interface{}) string {
	switch b := v.(type) {
	case string:
		panic(&jsThrown{value: rt.pyExcInstance("TypeError", "decoding str is not supported")})
	case *jsArray:
		return rt.pyBytesDecode(b.elems, enc)
	}
	panic(&jsThrown{value: rt.pyExcInstance("TypeError",
		"decoding to str: need a bytes-like object, "+pyTypeName(rt, v)+" found")})
}

// pyBDelVal is the builtin value a delegate works on: an instance's __val, or a
// plain value of the type itself (str.__repr__("x")).
func pyBDelVal(self interface{}) interface{} {
	return pyUnwrapBuiltin(self)
}

// pyBuiltinHasMethod says whether the builtin type tname has a method name,
// by the same tables hasattr() and the dispatcher use.
func pyBuiltinHasMethod(tname, name string) bool {
	switch tname {
	case "str":
		return pysIsStrMethod(name)
	case "list":
		return pyIsListMethod(name)
	case "dict":
		return pyIsDictMethod(name)
	case "int", "float":
		return pyIsNumMethod(pyDescTypeSample(tname), name)
	}
	return false
}

func pyBDelMake(tname, name string) *hostFunc {
	mk := func(fn func(rt *jsrt, self interface{}, args []interface{}) interface{}) *hostFunc {
		return jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			if len(args) == 0 {
				panic(&jsThrown{value: rt.pyExcInstance("TypeError",
					"descriptor '"+name+"' of '"+tname+"' object needs an argument")})
			}
			return fn(rt, pyBDelVal(args[0]), args[1:])
		})
	}
	arg := func(args []interface{}) interface{} {
		if len(args) > 0 {
			return args[0]
		}
		return jsUndef
	}
	if pyBuiltinHasMethod(tname, name) {
		f := mk(func(rt *jsrt, v interface{}, args []interface{}) interface{} {
			return rt.pyMCall(v, name, args, jsUndef)
		})
		pyBDelMethods[f] = true
		return f
	}
	num := tname == "int" || tname == "float"
	switch {
	case name == "__repr__":
		return mk(func(rt *jsrt, v interface{}, _ []interface{}) interface{} { return rt.pyRepr(v) })
	case name == "__str__" && tname == "str":
		return mk(func(rt *jsrt, v interface{}, _ []interface{}) interface{} { return v })
	case name == "__eq__":
		return mk(func(rt *jsrt, v interface{}, a []interface{}) interface{} { return rt.pyEqual(v, arg(a)) })
	case name == "__ne__":
		return mk(func(rt *jsrt, v interface{}, a []interface{}) interface{} { return !rt.pyEqual(v, arg(a)) })
	case num && name == "__bool__":
		return mk(func(rt *jsrt, v interface{}, _ []interface{}) interface{} { return rt.pyTruthyOf(v) })
	case num && name == "__neg__":
		return mk(func(rt *jsrt, v interface{}, _ []interface{}) interface{} { return rt.pyExt("js_pyunary", "-", v) })
	case num:
		if f := pyNumDunder(tname, name); f != nil {
			return mk(f)
		}
	case !num && name == "__len__":
		return mk(func(rt *jsrt, v interface{}, _ []interface{}) interface{} { return rt.pyExt("js_pylenof", v) })
	case !num && name == "__iter__":
		return mk(func(rt *jsrt, v interface{}, _ []interface{}) interface{} { return rt.pyExt("js_pyiterfn", v) })
	case !num && name == "__getitem__":
		return mk(func(rt *jsrt, v interface{}, a []interface{}) interface{} { return rt.pyExt("js_pyidx", v, arg(a)) })
	case (tname == "list" || tname == "dict") && name == "__setitem__":
		return mk(func(rt *jsrt, v interface{}, a []interface{}) interface{} {
			if len(a) < 2 {
				a = append(a, jsUndef, jsUndef)
			}
			return rt.pyExt("js_pyset", v, a[0], a[1])
		})
	case (tname == "list" || tname == "dict") && name == "__delitem__":
		return mk(func(rt *jsrt, v interface{}, a []interface{}) interface{} {
			return rt.pyExt("js_pydel_item", v, arg(a))
		})
	case !num && name == "__contains__":
		return mk(func(rt *jsrt, v interface{}, a []interface{}) interface{} { return rt.pyContains(arg(a), v) })
	case name == "__init__" && (tname == "list" || tname == "dict"):
		// list.__init__ / dict.__init__ refill the value in place, which is what
		// super().__init__(xs) in a subclass's own __init__ reaches.
		return mk(func(rt *jsrt, v interface{}, a []interface{}) interface{} {
			fresh := rt.callPyValue(rt.pyBuiltinClass(tname), a, jsUndef)
			rt.pyMCall(v, "clear", nil, jsUndef)
			ext := "extend"
			if tname == "dict" {
				ext = "update"
			}
			rt.pyMCall(v, ext, []interface{}{fresh}, jsUndef)
			return jsUndef
		})
	}
	ops := []string{"+", "*", "<", ">", "<=", ">="}
	if num {
		ops = append(ops, "-", "/", "//", "%", "**")
	}
	switch tname {
	case "int":
		ops = append(ops, "|", "&", "^", "<<", ">>")
	case "str":
		ops = append(ops, "%")
	case "dict":
		ops = []string{"|"}
	}
	for _, op := range ops {
		dn := pyDunderName[op]
		op := op
		ordered := op == "<" || op == ">" || op == "<=" || op == ">="
		if name == dn && ordered {
			return mk(func(rt *jsrt, v interface{}, a []interface{}) interface{} {
				if k := pyOrdKind(v); k == 0 || k != pyOrdKind(pyUnwrapBuiltin(arg(a))) {
					return pyNotImplemented
				}
				return rt.pyExt("js_pybin", op, v, arg(a))
			})
		}
		if name == dn {
			return mk(func(rt *jsrt, v interface{}, a []interface{}) interface{} { return rt.pyExt("js_pybin", op, v, arg(a)) })
		}
		if !ordered && name == "__r"+dn[2:] {
			return mk(func(rt *jsrt, v interface{}, a []interface{}) interface{} { return rt.pyExt("js_pybin", op, arg(a), v) })
		}
	}
	return nil
}

// pyNumDunder is int's and float's conversion and rounding dunders, each the
// builtin it serves applied to the value: str(x) of a number is its repr, and
// floor and ceil step int()'s truncation (which also raises CPython's error for
// an infinity or a NaN). __index__ and __invert__ are int's alone. Twin of
// pyNumDunder in languages/python-interpreter.abnf and lib/python-rt.metajs.
func pyNumDunder(tname, name string) func(rt *jsrt, v interface{}, a []interface{}) interface{} {
	isInt := tname == "int"
	toInt := func(rt *jsrt, v interface{}, _ []interface{}) interface{} { return rt.pyIntNew([]interface{}{v}) }
	switch {
	case name == "__str__":
		return func(rt *jsrt, v interface{}, _ []interface{}) interface{} { return rt.pyRepr(v) }
	case name == "__int__" || name == "__trunc__" || (isInt && (name == "__index__" || name == "__pos__")):
		return toInt
	case name == "__pos__":
		return func(rt *jsrt, v interface{}, _ []interface{}) interface{} { return v }
	case name == "__floor__" || name == "__ceil__":
		op, step := "<", "-"
		if name == "__ceil__" {
			op, step = ">", "+"
		}
		return func(rt *jsrt, v interface{}, _ []interface{}) interface{} {
			t := rt.pyIntNew([]interface{}{v})
			if rt.pyTruthyOf(rt.pyExt("js_pybin", op, v, t)) {
				return rt.pyExt("js_pybin", step, t, 1.0)
			}
			return t
		}
	case name == "__float__":
		return func(rt *jsrt, v interface{}, _ []interface{}) interface{} { return rt.pyExt("js_pyfloat", v) }
	case name == "__abs__":
		return func(rt *jsrt, v interface{}, _ []interface{}) interface{} { return rt.pyExt("js_pyabs", v) }
	case isInt && name == "__invert__":
		return func(rt *jsrt, v interface{}, _ []interface{}) interface{} { return rt.pyExt("js_pyunary", "~", v) }
	case name == "__round__":
		return func(rt *jsrt, v interface{}, a []interface{}) interface{} {
			return rt.pyExt("js_pyround", &jsArray{elems: append([]interface{}{v}, a...)})
		}
	case name == "__divmod__":
		return func(rt *jsrt, v interface{}, a []interface{}) interface{} { return rt.pyExt("js_pydivmod", v, argAt(a, 0)) }
	case name == "__rdivmod__":
		return func(rt *jsrt, v interface{}, a []interface{}) interface{} { return rt.pyExt("js_pydivmod", argAt(a, 0), v) }
	case name == "__format__":
		return func(rt *jsrt, v interface{}, a []interface{}) interface{} {
			return rt.pyExt("js_pyformatfn", &jsArray{elems: append([]interface{}{v}, a...)})
		}
	}
	return nil
}

// pyNumTypeName is the builtin numeric type v is - "int" (a bool included),
// "float", "complex" - or "".
func pyNumTypeName(v interface{}) string {
	switch v.(type) {
	case float64, bool, *jsBigInt:
		return "int"
	case *jsPyFlo:
		return "float"
	}
	if isPyComplex(v) {
		return "complex"
	}
	return ""
}

// pyNumDunderAttr says whether a NUMBER answers the dunder name through its
// type's delegate - (5).__add__ - for the receiver sites only: a type object's
// own dunders stay delegates, which unwrap a subclass instance.
func pyNumDunderAttr(v interface{}, name string) bool {
	tn := pyNumTypeName(v)
	if tn == "" || len(name) <= 4 || !strings.HasPrefix(name, "__") || !strings.HasSuffix(name, "__") {
		return false
	}
	_, ok := pyBuiltinDelegate(tn, name)
	return ok
}

// pyNotImplemented is CPython's NotImplemented, answered only by the delegates:
// an ORDERING of a builtin subclass's value with a value of another kind
// declines, so pyDunderBin tries the other operand and the ordering then raises
// with the operands' own types ('S' and 'int', not 'str' and 'int'). Twins:
// pyNotImpl in languages/python-interpreter.abnf and lib/python-rt.metajs.
var pyNotImplemented = newJSObject()

// pyDunderName is the dunder each binary operator a delegate serves is asked by.
var pyDunderName = map[string]string{
	"+": "__add__", "-": "__sub__", "*": "__mul__", "/": "__truediv__", "//": "__floordiv__",
	"%": "__mod__", "**": "__pow__", "|": "__or__", "&": "__and__", "^": "__xor__",
	"<<": "__lshift__", ">>": "__rshift__", "<": "__lt__", ">": "__gt__", "<=": "__le__", ">=": "__ge__",
}

// pyClassMember is one class's own member name, or the delegate a builtin type
// on a resolution order answers for it.
func pyClassMember(k *jsObject, name string) (interface{}, bool) {
	if v, ok := k.props[name]; ok {
		return v, true
	}
	if tn := pyBuiltinClsName(k); tn != "" {
		if f, ok := pyBuiltinDelegate(tn, name); ok {
			return f, true
		}
	}
	return nil, false
}

// pyBuiltinBase is the first builtin type on cls's resolution order that
// pyBuiltinDelegate serves, or "" - the kind of value pyNewInstance keeps.
func pyBuiltinBase(cls *jsObject) string {
	if _, has := cls.props["__mro"]; !has {
		return ""
	}
	for _, k := range pyMRO(cls) {
		if tn := pyBuiltinClsName(k); tn != "" {
			if _, ok := pyBuiltinDelegate(tn, "__repr__"); ok {
				return tn
			}
		}
	}
	return ""
}

// pyNewInstance is a call of a user class: a fresh instance and its __init__. A
// subclass of a builtin type keeps its value in __val: str/int/float build it
// from the arguments (their __new__), list/dict start empty and are filled by
// __init__ (list.__init__ unless the subclass overrides it).
func (rt *jsrt) pyNewInstance(cls *jsObject, args []interface{}, kw interface{}) *jsObject {
	inst := newJSObject()
	inst.set("__class", cls)
	switch bt := pyBuiltinBase(cls); bt {
	case "":
	case "list":
		inst.set("__val", &jsArray{})
	case "dict":
		inst.set("__val", pyDescTypeSample("dict"))
	default:
		inst.set("__val", rt.callPyValue(rt.pyBuiltinClass(bt), args, kw))
	}
	if init, found := pyLookup(cls, "__init__"); found && isCallable(init) {
		self := append([]interface{}{inst}, args...)
		rt.call(init, jsUndef, rt.pyBindCall(init, self, kw))
	}
	return inst
}

// pyUnwrapBuiltin is the builtin value of an instance of a builtin subclass;
// anything else is itself.
func pyUnwrapBuiltin(v interface{}) interface{} {
	if o, ok := v.(*jsObject); ok {
		if val, has := o.props["__val"]; has {
			if _, isInst := o.props["__class"]; isInst {
				return val
			}
		}
	}
	return v
}

// pyUserTruthy is the truthiness of an instance: __bool__, then __len__, else
// true.
func (rt *jsrt) pyUserTruthy(cls *jsObject, v interface{}) bool {
	if m, found := pyLookup(cls, "__bool__"); found && isCallable(m) {
		return rt.pyTruthyOf(rt.call(m, jsUndef, []interface{}{v}))
	}
	if m, found := pyLookup(cls, "__len__"); found && isCallable(m) {
		return rt.pyTruthyOf(rt.call(m, jsUndef, []interface{}{v}))
	}
	return true
}

// pyUserIter is what an instance's __iter__ answers, or false when its class
// has none.
func (rt *jsrt) pyUserIter(v interface{}) (interface{}, bool) {
	if _, cls, ok := pyInstance(v); ok {
		if m, found := pyLookup(cls, "__iter__"); found && isCallable(m) {
			return rt.call(m, jsUndef, []interface{}{v}), true
		}
	}
	return nil, false
}

// pyDrainIter is what an __iter__ answered, as a slice: a builtin iterable, or
// an instance whose __next__ is called until it raises StopIteration.
func (rt *jsrt) pyDrainIter(it interface{}) []interface{} {
	_, cls, ok := pyInstance(it)
	if !ok {
		return rt.pyElemsOf(it)
	}
	nx, found := pyLookup(cls, "__next__")
	if !found || !isCallable(nx) {
		panic(&jsThrown{value: rt.pyExcInstance("TypeError",
			"iter() returned non-iterator of type '"+pyTypeName(rt, it)+"'")})
	}
	var out []interface{}
	for {
		v, stop := rt.pyNextOrStop(nx, it)
		if stop {
			return out
		}
		out = append(out, v)
	}
}

func (rt *jsrt) pyNextOrStop(nx, it interface{}) (v interface{}, stop bool) {
	defer func() {
		if r := recover(); r != nil {
			if th, ok := r.(*jsThrown); ok && rt.pyIsInstance(th.value, rt.pyExcClass("StopIteration")) {
				stop = true
				return
			}
			panic(r)
		}
	}()
	return rt.call(nx, jsUndef, []interface{}{it}), false
}

// pyMCall is target.name(*args, **kw) through the whole method dispatcher the
// emitter calls (str's methods and the re module included).
func (rt *jsrt) pyMCall(target interface{}, name string, args []interface{}, kw interface{}) interface{} {
	return rt.pyExt("js_pyrxmcall", target, name, &jsArray{elems: args}, kw)
}

// pyExt calls one of python's externs by name with unwrapped values.
func (rt *jsrt) pyExt(name string, args ...interface{}) interface{} {
	hs := make([]uint64, len(args))
	for i, a := range args {
		if s, ok := a.(string); ok {
			hs[i] = rt.wrapStr(s)
		} else {
			hs[i] = rt.wrap(a)
		}
	}
	return rt.unwrap(pyExternMaps[rt][name](hs))
}

// pyBytesDecode is bytes.decode(encoding="utf-8") of a list of byte values:
// UTF-8 (strict, with CPython's UnicodeDecodeError texts), ASCII or Latin-1,
// and LookupError for any other codec.
func (rt *jsrt) pyBytesDecode(bs []interface{}, encV interface{}) string {
	enc := "utf-8"
	if !isUndefOrNull(encV) {
		enc = rt.pyString(encV)
	}
	e := strings.ReplaceAll(strings.ToLower(enc), "_", "-")
	b := make([]int, len(bs))
	for i, x := range bs {
		b[i] = int(rt.toNumber(x))
	}
	fail := func(codec string, from, to int, why string) {
		where := fmt.Sprintf("byte 0x%02x in position %d", b[from], from)
		if to-from > 1 {
			where = fmt.Sprintf("bytes in position %d-%d", from, to-1)
		}
		panic(&jsThrown{value: rt.pyExcInstance("UnicodeDecodeError",
			"'"+codec+"' codec can't decode "+where+": "+why)})
	}
	var sb strings.Builder
	switch e {
	case "latin-1", "latin1", "iso-8859-1", "iso8859-1":
		for _, c := range b {
			sb.WriteRune(rune(c))
		}
		return sb.String()
	case "ascii", "us-ascii":
		for i, c := range b {
			if c > 127 {
				fail("ascii", i, i+1, "ordinal not in range(128)")
			}
			sb.WriteRune(rune(c))
		}
		return sb.String()
	case "utf-8", "utf8":
	default:
		panic(&jsThrown{value: rt.pyExcInstance("LookupError", "unknown encoding: "+enc)})
	}
	for i := 0; i < len(b); {
		b0 := b[i]
		need := 0
		switch {
		case b0 < 0x80:
			sb.WriteRune(rune(b0))
			i++
			continue
		case b0 >= 0xC2 && b0 <= 0xDF:
			need = 1
		case b0 >= 0xE0 && b0 <= 0xEF:
			need = 2
		case b0 >= 0xF0 && b0 <= 0xF4:
			need = 3
		default:
			fail("utf-8", i, i+1, "invalid start byte")
		}
		cp := b0 & (0x3F >> need)
		for k := 1; k <= need; k++ {
			if i+k >= len(b) {
				fail("utf-8", i, len(b), "unexpected end of data")
			}
			c := b[i+k]
			lo, hi := 0x80, 0xBF
			if k == 1 {
				switch b0 {
				case 0xE0:
					lo = 0xA0
				case 0xED:
					hi = 0x9F
				case 0xF0:
					lo = 0x90
				case 0xF4:
					hi = 0x8F
				}
			}
			if c < lo || c > hi {
				fail("utf-8", i, i+k, "invalid continuation byte")
			}
			cp = cp<<6 | c&0x3F
		}
		sb.WriteRune(rune(cp))
		i += need + 1
	}
	return sb.String()
}

// pyBytesNew is bytes(source, encoding, errors), with the keywords already in
// their slots: a list of byte values in this value model - a str under an
// encoding (str.encode), a count of zero bytes, or an iterable of ints in
// range(256). None and an absent argument are the same value here. Twins:
// pyBytesNew in languages/python-interpreter.abnf, js_pybytes in
// lib/python-rt.metajs.
func (rt *jsrt) pyBytesNew(args []interface{}) interface{} {
	raise := func(cls, msg string) { panic(&jsThrown{value: rt.pyExcInstance(cls, msg)}) }
	if len(args) > 3 {
		raise("TypeError", fmt.Sprintf("bytes() takes at most 3 arguments (%d given)", len(args)))
	}
	at := func(i int) interface{} {
		if i < len(args) {
			return args[i]
		}
		return jsUndef
	}
	v, enc, errs := at(0), at(1), at(2)
	if s, ok := v.(string); ok {
		if enc == jsUndef {
			raise("TypeError", "string argument without an encoding")
		}
		return rt.pysEncode(s, enc, errs)
	}
	if enc != jsUndef {
		raise("TypeError", "encoding without a string argument")
	}
	if errs != jsUndef {
		raise("TypeError", "errors without a string argument")
	}
	out := &jsArray{}
	if len(args) == 0 {
		return out
	}
	switch tn := pyTypeName(rt, v); tn {
	case "int", "bool":
		n := rt.toNumber(v)
		if n < 0 {
			raise("ValueError", "negative count")
		}
		for i := 0; float64(i) < n; i++ {
			out.elems = append(out.elems, float64(0))
		}
		return out
	case "NoneType", "float", "complex":
		raise("TypeError", "cannot convert '"+tn+"' object to bytes")
	}
	for _, e := range rt.pyElemsOf(v) {
		et := pyTypeName(rt, e)
		if et != "int" && et != "bool" {
			raise("TypeError", "'"+et+"' object cannot be interpreted as an integer")
		}
		b := rt.toNumber(e)
		if b < 0 || b > 255 {
			raise("ValueError", "bytes must be in range(0, 256)")
		}
		out.elems = append(out.elems, b)
	}
	return out
}

func init() {
	rxExtraExterns = append(rxExtraExterns, func(rt *jsrt, m map[string]func(args []uint64) uint64) {
		// bytes(...), bound as [positionals, keywords].
		m["js_pybytes"] = func(a []uint64) uint64 {
			el := rt.unwrap(a[0]).(*jsArray).elems
			pos := el[0].(*jsArray).elems
			if keys, _, ok := dictParts(el[1]); ok && len(keys.elems) > 0 {
				pos = rt.pyKwPos("bytes", pos, el[1], []string{"source", "encoding", "errors"}, 0)
			}
			return rt.wrap(rt.pyBytesNew(pos))
		}
		// frozenset(iterable=()). Twins: pyConvert in
		// languages/python-interpreter.abnf, js_pyfrozenset in lib/python-rt.metajs.
		m["js_pyfrozenset"] = func(a []uint64) uint64 {
			out := &jsArray{}
			if v := rt.unwrap(a[0]); !isUndefOrNull(v) {
				for _, e := range rt.pyElemsOf(v) {
					rt.pySetAdd(out, e)
				}
			}
			fs := newPySet(out)
			fs.set("__frozen", true)
			return rt.wrap(fs)
		}
	})
}
