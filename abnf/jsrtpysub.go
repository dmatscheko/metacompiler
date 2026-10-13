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
// bool's are int's; a tuple's are its own dunders, and its two named methods
// are not delegates.
func pyBuiltinDelegate(tname, name string) (*hostFunc, bool) {
	switch tname {
	case "bool":
		return pyBuiltinDelegate("int", name)
	case "tuple":
		if !strings.HasPrefix(name, "__") && !pyIsTupleMethod(name) {
			return nil, false
		}
	case "str", "int", "float", "list", "dict", "set", "frozenset", "complex", "bytes", "bytearray":
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

// ------------------------------------------------------- slice objects --

// slice(stop) / slice(start, stop[, step]) is an object carrying __pyslice, the
// [start, stop, step] array, an omitted bound None. Subscripting a sequence with
// one is slicing it (js_pyidx), and the slice syntax hands one to an instance's
// __getitem__, __setitem__ and __delitem__ (js_pyslice3, js_pysetslice,
// js_pydelslice). Twin of pySliceNew in languages/python-interpreter.abnf and in
// lib/python-rt.metajs.
func pySliceOf(lo, hi, st interface{}) *jsObject {
	o := newJSObject()
	o.set("__pyslice", &jsArray{elems: []interface{}{lo, hi, st}})
	return o
}

// pySliceParts is a slice object's [start, stop, step].
func pySliceParts(v interface{}) ([]interface{}, bool) {
	if o, ok := v.(*jsObject); ok {
		if b, ok := o.props["__pyslice"].(*jsArray); ok {
			return b.elems, true
		}
	}
	return nil, false
}

// pySliceField is the bound an attribute name reads, or -1.
func pySliceField(name string) int {
	switch name {
	case "start":
		return 0
	case "stop":
		return 1
	case "step":
		return 2
	}
	return -1
}

// pySliceIndicesOf is s.indices(n): (start, stop, step) clamped to a sequence
// of length n, CPython's PySlice_GetIndicesEx.
func (rt *jsrt) pySliceIndicesOf(b []interface{}, nv interface{}) interface{} {
	toInt := func(v interface{}) float64 {
		if pyNumTypeName(v) != "int" {
			rt.pyRaise("TypeError", "slice indices must be integers or None or have an __index__ method")
		}
		return rt.toNumber(v)
	}
	if pyNumTypeName(nv) != "int" {
		rt.pyRaise("TypeError", "'"+pyTypeName(rt, nv)+"' object cannot be interpreted as an integer")
	}
	n := rt.toNumber(nv)
	if n < 0 {
		rt.pyRaise("ValueError", "length should not be negative")
	}
	step := 1.0
	if !isUndefOrNull(b[2]) {
		step = toInt(b[2])
	}
	if step == 0 {
		rt.pyRaise("ValueError", "slice step cannot be zero")
	}
	lower, upper := 0.0, n
	if step < 0 {
		lower, upper = -1, n-1
	}
	bound := func(v interface{}, dflt float64) float64 {
		if isUndefOrNull(v) {
			return dflt
		}
		x := toInt(v)
		if x < 0 {
			if x += n; x < lower {
				x = lower
			}
		} else if x > upper {
			x = upper
		}
		return x
	}
	start, stop := bound(b[0], lower), bound(b[1], upper)
	if step < 0 {
		start, stop = bound(b[0], upper), bound(b[1], lower)
	}
	return pyTup(&jsArray{elems: []interface{}{start, stop, step}})
}

// pyStrDecode is str(object, encoding[, errors]): the decoding form, for a
// bytes value only.
func (rt *jsrt) pyStrDecode(v, enc, errs interface{}) string {
	switch b := v.(type) {
	case string:
		panic(&jsThrown{value: rt.pyExcInstance("TypeError", "decoding str is not supported")})
	case *jsArray:
		if pyIsBytes(b) {
			return rt.pyBytesDecode(b.elems, enc, errs)
		}
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
	case "tuple":
		return pyIsTupleMethod(name)
	case "dict":
		return pyIsDictMethod(name)
	case "set":
		return pyIsSetMethod(name)
	case "frozenset":
		return pyIsSetMethod(name) && !pySetMutator(name)
	case "complex":
		return name == "conjugate"
	case "int", "float":
		return pyIsNumMethod(pyDescTypeSample(tname), name)
	case "bytes":
		return pyIsBytesMethod(name, pyBytesKind)
	case "bytearray":
		return pyIsBytesMethod(name, pyBArrKind)
	}
	return false
}

func pyBDelMake(tname, name string) *hostFunc {
	// mkSelf's fn sees the receiver as passed (an in-place operator answers it)
	// and its builtin value; mk's sees the value only.
	mkSelf := func(fn func(rt *jsrt, self, v interface{}, args []interface{}) interface{}) *hostFunc {
		return jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			if len(args) == 0 {
				panic(&jsThrown{value: rt.pyExcInstance("TypeError",
					"descriptor '"+name+"' of '"+tname+"' object needs an argument")})
			}
			return fn(rt, args[0], pyBDelVal(args[0]), args[1:])
		})
	}
	mk := func(fn func(rt *jsrt, v interface{}, args []interface{}) interface{}) *hostFunc {
		return mkSelf(func(rt *jsrt, _, v interface{}, args []interface{}) interface{} { return fn(rt, v, args) })
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
	seq := tname == "list" || tname == "tuple" || tname == "str" || tname == "bytes" || tname == "bytearray"
	setLike := tname == "set" || tname == "frozenset"
	switch {
	case name == "__repr__" && tname == "bytearray":
		// A bytearray's repr names the receiver's own type: BA(b'hi').
		return mkSelf(func(rt *jsrt, self, v interface{}, _ []interface{}) interface{} {
			return pyTypeName(rt, self) + rt.pyRepr(v)[len("bytearray"):]
		})
	case name == "__repr__":
		return mk(func(rt *jsrt, v interface{}, _ []interface{}) interface{} { return rt.pyRepr(v) })
	case name == "__str__" && tname == "str":
		return mk(func(rt *jsrt, v interface{}, _ []interface{}) interface{} { return v })
	case name == "__str__" && tname != "bytes":
		// Every other builtin's __str__ is object's, which renders through the
		// type's __repr__ - a subclass's own override included.
		return mkSelf(func(rt *jsrt, self, v interface{}, _ []interface{}) interface{} {
			if _, cls, ok := pyInstance(self); ok {
				if r, found := pyLookup(cls, "__repr__"); found {
					return rt.call(r, jsUndef, []interface{}{self})
				}
			}
			return rt.pyRepr(v)
		})
	case name == "__str__":
		return mk(func(rt *jsrt, v interface{}, _ []interface{}) interface{} { return rt.pyString(v) })
	case name == "__eq__" || name == "__ne__":
		neg := name == "__ne__"
		return mk(func(rt *jsrt, v interface{}, a []interface{}) interface{} {
			o := pyUnwrapBuiltin(arg(a))
			if !pyBDelAccepts(tname, o) {
				return pyNotImplemented
			}
			return rt.pyEqual(v, o) != neg
		})
	case name == "__hash__":
		// list, dict and set have __hash__ = None: no delegate (see pyBHashNone).
		if pyBHashNone(tname) {
			return nil
		}
		return mk(func(rt *jsrt, v interface{}, _ []interface{}) interface{} { return rt.pyExt("js_pyhash", v) })
	case name == "__format__" && !num:
		return mk(func(rt *jsrt, v interface{}, a []interface{}) interface{} {
			return rt.pyExt("js_pyformatfn", &jsArray{elems: append([]interface{}{v}, a...)})
		})
	case name == "__init__" && (tname == "list" || tname == "dict" || tname == "set"):
		// list.__init__ / dict.__init__ / set.__init__ refill the value in place,
		// which is what super().__init__(xs) in a subclass's own __init__ reaches.
		return mk(func(rt *jsrt, v interface{}, a []interface{}) interface{} {
			fresh := rt.callPyValue(rt.pyBuiltinClass(tname), a, jsUndef)
			rt.pyMCall(v, "clear", nil, jsUndef)
			ext := "extend"
			if tname != "list" {
				ext = "update"
			}
			rt.pyMCall(v, ext, []interface{}{fresh}, jsUndef)
			return jsUndef
		})
	case name == "__init__":
		// object.__init__: nothing to do for an immutable value.
		return mk(func(rt *jsrt, v interface{}, _ []interface{}) interface{} { return jsUndef })
	case num && name == "__bool__":
		return mk(func(rt *jsrt, v interface{}, _ []interface{}) interface{} { return rt.pyTruthyOf(v) })
	case num && name == "__neg__":
		return mk(func(rt *jsrt, v interface{}, _ []interface{}) interface{} { return rt.pyExt("js_pyunary", "-", v) })
	case num:
		if f := pyNumDunder(tname, name); f != nil {
			return mk(f)
		}
	case tname == "complex" && name == "__bool__":
		return mk(func(rt *jsrt, v interface{}, _ []interface{}) interface{} { return rt.pyTruthyOf(v) })
	case tname == "complex" && (name == "__neg__" || name == "__pos__"):
		op := "+"
		if name == "__neg__" {
			op = "-"
		}
		return mk(func(rt *jsrt, v interface{}, _ []interface{}) interface{} { return rt.pyExt("js_pyunary", op, v) })
	case tname == "complex" && name == "__abs__":
		return mk(func(rt *jsrt, v interface{}, _ []interface{}) interface{} { return rt.pyExt("js_pyabs", v) })
	case tname == "complex" && name == "__complex__":
		return mk(func(rt *jsrt, v interface{}, _ []interface{}) interface{} { return v })
	case tname != "complex" && name == "__len__":
		return mk(func(rt *jsrt, v interface{}, _ []interface{}) interface{} { return rt.pyExt("js_pylenof", v) })
	case tname != "complex" && name == "__iter__":
		return mk(func(rt *jsrt, v interface{}, _ []interface{}) interface{} { return rt.pyExt("js_pyiterfn", v) })
	case tname == "dict" && name == "__getitem__":
		// dict's asks the class's __missing__ for an absent key. Twin of
		// pyBGetItem in languages/python-interpreter.abnf.
		return mkSelf(func(rt *jsrt, self, v interface{}, a []interface{}) interface{} {
			if _, cls, ok := pyInstance(self); ok {
				if keys, _, isDict := dictParts(v); isDict && rt.pyDictFind(keys, arg(a)) < 0 {
					if ms, found := pyLookup(cls, "__missing__"); found {
						return rt.call(ms, jsUndef, []interface{}{self, arg(a)})
					}
				}
			}
			return rt.pyExt("js_pyidx", v, arg(a))
		})
	case seq && name == "__getitem__":
		return mk(func(rt *jsrt, v interface{}, a []interface{}) interface{} { return rt.pyExt("js_pyidx", v, arg(a)) })
	case (tname == "list" || tname == "dict" || tname == "bytearray") && name == "__setitem__":
		return mk(func(rt *jsrt, v interface{}, a []interface{}) interface{} {
			if len(a) < 2 {
				a = append(a, jsUndef, jsUndef)
			}
			return rt.pyExt("js_pyset", v, a[0], a[1])
		})
	case (tname == "list" || tname == "dict" || tname == "bytearray") && name == "__delitem__":
		return mk(func(rt *jsrt, v interface{}, a []interface{}) interface{} {
			return rt.pyExt("js_pydel_item", v, arg(a))
		})
	case (tname == "list" || tname == "dict") && name == "__reversed__":
		return mk(func(rt *jsrt, v interface{}, _ []interface{}) interface{} { return rt.pyExt("js_pyreversed", v) })
	case tname != "complex" && name == "__contains__":
		return mk(func(rt *jsrt, v interface{}, a []interface{}) interface{} { return rt.pyContains(arg(a), v) })
	}
	if op := pyBDelInplace(tname, name); op != "" {
		// The augmented operator on the value itself, which every engine performs
		// in place for these receivers; the receiver is the answer.
		return mkSelf(func(rt *jsrt, self, v interface{}, a []interface{}) interface{} {
			o := pyUnwrapBuiltin(arg(a))
			if setLike && !pyBDelAccepts(tname, o) {
				return pyNotImplemented
			}
			rt.pyExt("js_pybin", op+"=", v, o)
			return self
		})
	}
	var ops []string
	switch {
	case num:
		ops = []string{"+", "*", "<", ">", "<=", ">=", "-", "/", "//", "%", "**"}
		if tname == "int" {
			ops = append(ops, "|", "&", "^", "<<", ">>")
		}
	case seq:
		ops = []string{"+", "*", "<", ">", "<=", ">="}
		if tname == "str" || tname == "bytes" || tname == "bytearray" {
			ops = append(ops, "%")
		}
	case tname == "dict":
		ops = []string{"|", "<", ">", "<=", ">="}
	case setLike:
		ops = []string{"|", "&", "-", "^", "<", ">", "<=", ">="}
	case tname == "complex":
		ops = []string{"+", "-", "*", "/", "**", "<", ">", "<=", ">="}
	}
	for _, op := range ops {
		dn := pyDunderName[op]
		op := op
		ordered := op == "<" || op == ">" || op == "<=" || op == ">="
		// A sequence reflects only its repetition (and str its formatting).
		reflects := !ordered && (!seq || op == "*" || op == "%")
		reflected := reflects && name == "__r"+dn[2:]
		if name != dn && !reflected {
			continue
		}
		// object's orderings, which dict and complex inherit, decline everything.
		if ordered && (tname == "dict" || tname == "complex") {
			return mk(func(rt *jsrt, v interface{}, a []interface{}) interface{} { return pyNotImplemented })
		}
		// A number's, a dict's and a set's operators decline a foreign operand;
		// str's and list's `+`, `*` and `%` raise their own TypeError instead,
		// and only their orderings and a reflected % ("%d".__rmod__(3)) decline.
		gated := !seq || ordered || (reflected && op == "%")
		return mk(func(rt *jsrt, v interface{}, a []interface{}) interface{} {
			o := pyUnwrapBuiltin(arg(a))
			if gated && !pyBDelAccepts(tname, o) {
				return pyNotImplemented
			}
			if reflected {
				return rt.pyExt("js_pybin", op, o, v)
			}
			return rt.pyExt("js_pybin", op, v, o)
		})
	}
	return nil
}

// pyBHashNone names the builtin types whose __hash__ is None: the mutable
// containers. hasattr(list, "__hash__") is True and list.__hash__ is None.
// Twin of pyBHashNone in languages/python-interpreter.abnf and
// lib/python-rt.metajs.
func pyBHashNone(tname string) bool {
	return tname == "list" || tname == "dict" || tname == "set" || tname == "bytearray"
}

// pyBDelInplace is the binary operator of tname's in-place dunder name, or "":
// list's and bytearray's += and *=, dict's |=, set's |= &= -= ^=.
func pyBDelInplace(tname, name string) string {
	switch {
	case (tname == "list" || tname == "bytearray") && name == "__iadd__":
		return "+"
	case (tname == "list" || tname == "bytearray") && name == "__imul__":
		return "*"
	case (tname == "dict" || tname == "set") && name == "__ior__":
		return "|"
	case tname == "set" && name == "__iand__":
		return "&"
	case tname == "set" && name == "__isub__":
		return "-"
	case tname == "set" && name == "__ixor__":
		return "^"
	}
	return ""
}

// pyBDelAccepts says whether builtin type tname's binary dunder serves the
// other operand o (a subclass instance already unwrapped): int serves ints (a
// bool is one), float ints and floats, complex every number, a container its
// own kind (a set and a frozenset each other). For anything else the dunder
// answers NotImplemented, so int.__add__(5, 2.5) declines. Twin of
// pyBDelAccepts in languages/python-interpreter.abnf and lib/python-rt.metajs.
func pyBDelAccepts(tname string, o interface{}) bool {
	nk := pyNumTypeName(o)
	switch tname {
	case "int":
		return nk == "int"
	case "float":
		return nk == "int" || nk == "float"
	case "complex":
		return nk != ""
	case "str":
		_, ok := o.(string)
		return ok
	case "list", "tuple", "bytes", "bytearray":
		a, ok := o.(*jsArray)
		return ok && pySeqTName(a) == tname
	case "set", "frozenset":
		_, isSet := pySetElems(o)
		return isSet
	}
	if _, isSet := pySetElems(o); isSet {
		return false
	}
	_, _, ok := dictParts(o)
	return ok
}

// pyNumDunder is int's and float's conversion and rounding dunders, each the
// builtin it serves applied to the value: floor and ceil step int()'s
// truncation (which also raises CPython's error for an infinity or a NaN). __index__ and __invert__ are int's alone. Twin of
// pyNumDunder in languages/python-interpreter.abnf and lib/python-rt.metajs.
func pyNumDunder(tname, name string) func(rt *jsrt, v interface{}, a []interface{}) interface{} {
	isInt := tname == "int"
	toInt := func(rt *jsrt, v interface{}, _ []interface{}) interface{} { return rt.pyIntNew([]interface{}{v}) }
	switch {
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
		return func(rt *jsrt, v interface{}, a []interface{}) interface{} {
			return rt.pyExt("js_pydivmod", v, argAt(a, 0))
		}
	case name == "__rdivmod__":
		return func(rt *jsrt, v interface{}, a []interface{}) interface{} {
			return rt.pyExt("js_pydivmod", argAt(a, 0), v)
		}
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

// pyTupKind is the jsArray.kind a Python TUPLE carries (arrKind / arrKindSet in
// the other two engines; 16 stays clear of kotlin's 1-3, since the shared js_py*
// externs see both). Indexing, len, iteration and unpacking treat it as any
// array; what differs - repr, type, ==, ordering, hashing, immutability - asks
// pyIsTup. Twin of pyTup / isTup in languages/python-interpreter.abnf and pyTup
// / pyIsTup in lib/python-rt.metajs.
const pyTupKind = 16

func pyTup(a *jsArray) *jsArray {
	a.kind = pyTupKind
	return a
}

func pyIsTup(v interface{}) bool {
	a, ok := v.(*jsArray)
	return ok && a.kind == pyTupKind
}

// pyValDunder is the delegate through which a builtin VALUE answers the dunder
// name - (5).__add__, "s".__add__, [].__len__ - for the receiver sites only: a
// type object's own dunders stay delegates, which unwrap a subclass instance.
// Twin of pyValDunder in languages/python-interpreter.abnf and lib/python-rt.metajs.
func pyValDunder(v interface{}, name string) (*hostFunc, bool) {
	tn := pyNumTypeName(v)
	switch v.(type) {
	case string:
		tn = "str"
	case *jsArray:
		tn = pySeqTName(v.(*jsArray))
	}
	if _, isSet := pySetElems(v); isSet {
		tn = "set"
		if pySetFrozen(v) {
			tn = "frozenset"
		}
	} else if _, _, isDict := dictParts(v); isDict {
		tn = "dict"
	}
	if tn == "" || len(name) <= 4 || !strings.HasPrefix(name, "__") || !strings.HasSuffix(name, "__") {
		return nil, false
	}
	return pyBuiltinDelegate(tn, name)
}

// pyIsTupleMethod names a tuple's two methods. Twin of pyIsTupleMethod in
// languages/python-interpreter.abnf and lib/python-rt.metajs.
func pyIsTupleMethod(name string) bool { return name == "count" || name == "index" }

// pyValHashNone says whether v is a value whose __hash__ is None (pyBHashNone):
// a list, a bytearray, a dict or a set. Twin of pyValHashNone in
// languages/python-interpreter.abnf and lib/python-rt.metajs.
func pyValHashNone(v interface{}) bool {
	if a, isArr := v.(*jsArray); isArr {
		return pyBHashNone(pySeqTName(a))
	}
	if _, isSet := pySetElems(v); isSet {
		return !pySetFrozen(v)
	}
	_, _, isDict := dictParts(v)
	return isDict
}

func pyValDunderAttr(v interface{}, name string) bool {
	_, ok := pyValDunder(v, name)
	return ok
}

// pyNotImplemented is CPython's NotImplemented singleton, the builtin name
// NotImplemented (js_pynotimpl). A binary dunder answers it to decline an
// operand, and pyDunderBin then offers the operation to the other operand.
// Twins: pyNotImpl in languages/python-interpreter.abnf and lib/python-rt.metajs.
var pyNotImplemented = pyConstObj("NotImplemented")

// pyConstObj is a named singleton - NotImplemented, Ellipsis - which prints as
// its name.
func pyConstObj(name string) *jsObject {
	o := newJSObject()
	o.set("__pyconst", name)
	return o
}

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
// subclass of a builtin type keeps its value in __val: the immutable types build
// it from the arguments (their __new__), list/dict/set start empty and are
// filled by __init__ (list.__init__ unless the subclass overrides it).
func (rt *jsrt) pyNewInstance(cls *jsObject, args []interface{}, kw interface{}) interface{} {
	// A class's own __new__ makes the instance, and __init__ runs only when
	// what it made is an instance of the class.
	if nw, found := pyLookup(cls, "__new__"); found {
		if pyDescKind(nw) != "" {
			nw = rt.pyDescGet(nw, cls, "__new__")
		}
		made := rt.call(nw, jsUndef, rt.pyBindCall(nw, append([]interface{}{cls}, args...), kw))
		if rt.pyIsInstance(made, cls) {
			if init, found := pyLookup(cls, "__init__"); found && isCallable(init) {
				rt.call(init, jsUndef, rt.pyBindCall(init, append([]interface{}{made}, args...), kw))
			}
		}
		return made
	}
	inst := newJSObject()
	inst.set("__class", cls)
	bt := pyBuiltinBase(cls)
	if bt != "" {
		inst.set("__val", rt.pyBuiltinVal(bt, args, kw))
	}
	if init, found := pyLookup(cls, "__init__"); found && isCallable(init) {
		self := append([]interface{}{inst}, args...)
		rt.call(init, jsUndef, rt.pyBindCall(init, self, kw))
	} else if bt == "" && (len(args) > 0 || pyKwCount(kw) > 0) && !pyIsExcClass(cls) {
		// Neither __init__ nor __new__ anywhere but object's.
		rt.pyRaise("TypeError", rt.toString(cls.props["__name"])+"() takes no arguments")
	}
	return inst
}

// pyBuiltinVal is the value a fresh instance of a subclass of builtin type bt
// holds: the immutable types build it from the arguments (their __new__),
// list/dict/set start empty and are filled by __init__.
func (rt *jsrt) pyBuiltinVal(bt string, args []interface{}, kw interface{}) interface{} {
	switch bt {
	case "list":
		return &jsArray{}
	case "dict":
		return pyDescTypeSample("dict")
	case "set":
		return newPySet(&jsArray{})
	}
	return rt.callPyValue(rt.pyBuiltinClass(bt), args, kw)
}

// pyKwCount is how many keyword arguments a call passed.
func pyKwCount(kw interface{}) int {
	if keys, _, ok := dictParts(kw); ok {
		return len(keys.elems)
	}
	return 0
}

// pyIsExcClass reports whether cls derives from BaseException.
func pyIsExcClass(cls *jsObject) bool {
	for _, k := range pyMRO(cls) {
		if k.props["__name"] == "BaseException" {
			return true
		}
	}
	return false
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

// pyDrainIter is what an __iter__ answered, as a slice: a generator or a
// cursor, or an instance whose __next__ is called until it raises StopIteration.
func (rt *jsrt) pyDrainIter(it interface{}) []interface{} {
	if _, _, ok := pyInstance(it); !ok {
		return rt.pyElemsOf(rt.pyBuiltinIterator(it))
	}
	nx := rt.pyUserNextFn(it)
	var out []interface{}
	for {
		v, stop := rt.pyNextOrStop(nx, it)
		if stop {
			return out
		}
		out = append(out, v)
	}
}

// pyNonIterator is CPython's TypeError for an __iter__ that answered something
// that is not an iterator.
func (rt *jsrt) pyNonIterator(it interface{}) {
	panic(&jsThrown{value: rt.pyExcInstance("TypeError",
		"iter() returned non-iterator of type '"+pyTypeName(rt, it)+"'")})
}

// pyBuiltinIterator is a non-instance __iter__ result when it is an iterator (a
// generator or a cursor), and CPython's TypeError otherwise.
func (rt *jsrt) pyBuiltinIterator(it interface{}) interface{} {
	if _, ok := it.(*jsGenerator); !ok && pyIterName(rt, it) == "" {
		rt.pyNonIterator(it)
	}
	return it
}

// pyUserNextFn is the __next__ of an instance an __iter__ answered.
func (rt *jsrt) pyUserNextFn(it interface{}) interface{} {
	_, cls, _ := pyInstance(it)
	nx, found := pyLookup(cls, "__next__")
	if !found || !isCallable(nx) {
		rt.pyNonIterator(it)
	}
	return nx
}

// pyUserStepper is what an __iter__ answered, as something a compiled loop can
// step one value at a time: a generator or a cursor is its own, and a user
// iterator gets a cursor-shaped wrapper whose "next" runs ONE __next__, so an
// endless one ends at a `break`. The __pyit name is the instance's type, which
// is what a send() into a `yield from` over it names (js_pyyfsend). Twin of
// pyAUserStepper in languages/lib/python-rt.metajs; the interpreter steps the
// instance directly (pyUserStep).
func (rt *jsrt) pyUserStepper(it interface{}) interface{} {
	if _, _, ok := pyInstance(it); !ok {
		return rt.pyBuiltinIterator(it)
	}
	nx := rt.pyUserNextFn(it)
	ad := newJSObject()
	ad.set("__pyit", pyTypeName(rt, it))
	ad.set("next", jsHostFunc("pyuser.next", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		out := newJSObject()
		v, stop := rt.pyNextOrStop(nx, it)
		if stop {
			v = jsUndef
		}
		out.set("value", v)
		out.set("done", stop)
		return out
	}))
	return ad
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

// pyBytesDecode is bytes.decode(encoding="utf-8", errors="strict") of a list
// of byte values: UTF-8, ASCII or Latin-1, and LookupError for any other codec.
// Each undecodable run [from, to) - CPython's UnicodeDecodeError start and end -
// goes to the error handler (bad).
func (rt *jsrt) pyBytesDecode(bs []interface{}, encV, errV interface{}) string {
	enc, errs := "utf-8", "strict"
	if !isUndefOrNull(encV) {
		enc = rt.pyString(encV)
	}
	if !isUndefOrNull(errV) {
		errs = rt.pyString(errV)
	}
	e := strings.ReplaceAll(strings.ToLower(enc), "_", "-")
	b := make([]int, len(bs))
	for i, x := range bs {
		b[i] = int(rt.toNumber(x))
	}
	var sb strings.Builder
	// bad writes what the handler puts in place of b[from:to]; "strict"
	// raises. A handler name is only looked up once it is needed, as CPython
	// does.
	bad := func(codec string, from, to int, why string) {
		switch errs {
		case "strict":
			where := fmt.Sprintf("byte 0x%02x in position %d", b[from], from)
			if to-from > 1 {
				where = fmt.Sprintf("bytes in position %d-%d", from, to-1)
			}
			panic(&jsThrown{value: rt.pyExcInstance("UnicodeDecodeError",
				"'"+codec+"' codec can't decode "+where+": "+why)})
		case "ignore":
		case "replace":
			sb.WriteRune(0xFFFD)
		case "backslashreplace":
			for _, c := range b[from:to] {
				fmt.Fprintf(&sb, "\\x%02x", c)
			}
		default:
			panic(&jsThrown{value: rt.pyExcInstance("LookupError", "unknown error handler name '"+errs+"'")})
		}
	}
	switch e {
	case "latin-1", "latin1", "iso-8859-1", "iso8859-1":
		for _, c := range b {
			sb.WriteRune(rune(c))
		}
		return sb.String()
	case "ascii", "us-ascii":
		for i, c := range b {
			if c > 127 {
				bad("ascii", i, i+1, "ordinal not in range(128)")
			} else {
				sb.WriteRune(rune(c))
			}
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
		}
		why, to := "", i+1
		if need == 0 {
			why = "invalid start byte"
		}
		cp := b0 & (0x3F >> need)
		for k := 1; why == "" && k <= need; k++ {
			if i+k >= len(b) {
				why, to = "unexpected end of data", len(b)
				break
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
				why, to = "invalid continuation byte", i+k
				break
			}
			cp = cp<<6 | c&0x3F
		}
		if why != "" {
			bad("utf-8", i, to, why)
			i = to
			continue
		}
		sb.WriteRune(rune(cp))
		i += need + 1
	}
	return sb.String()
}

// pyBytesNew is bytes(source, encoding, errors), with the keywords already in
// their slots: a str under an encoding (str.encode), a count of zero bytes, or
// an iterable of ints in range(256). None and an absent argument are the same
// value here. bytearray(...) (name) builds the mutable kind, under its own error
// texts. Twins:
// pyBytesNew in languages/python-interpreter.abnf and lib/python-rt.metajs.
func (rt *jsrt) pyBytesNew(args []interface{}, name string) interface{} {
	raise := func(cls, msg string) { panic(&jsThrown{value: rt.pyExcInstance(cls, msg)}) }
	if len(args) > 3 {
		raise("TypeError", fmt.Sprintf("%s() takes at most 3 arguments (%d given)", name, len(args)))
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
		r := rt.pysEncode(s, enc, errs).(*jsArray)
		r.kind = pyBytesKind
		if name == "bytearray" {
			r.kind = pyBArrKind
		}
		return r
	}
	if enc != jsUndef {
		raise("TypeError", "encoding without a string argument")
	}
	if errs != jsUndef {
		raise("TypeError", "errors without a string argument")
	}
	out := &jsArray{kind: pyBytesKind}
	if name == "bytearray" {
		out.kind = pyBArrKind
	}
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
		raise("TypeError", "cannot convert '"+tn+"' object to "+name)
	}
	for _, e := range rt.pyElemsOf(v) {
		et := pyTypeName(rt, e)
		if et != "int" && et != "bool" {
			raise("TypeError", "'"+et+"' object cannot be interpreted as an integer")
		}
		b := rt.toNumber(e)
		if b < 0 || b > 255 {
			raise("ValueError", strings.TrimSuffix(name, "array")+" must be in range(0, 256)")
		}
		out.elems = append(out.elems, b)
	}
	return out
}

func init() {
	rxExtraExterns = append(rxExtraExterns, func(rt *jsrt, m map[string]func(args []uint64) uint64) {
		// bytes(...) and bytearray(...), bound as [positionals, keywords].
		bytesNew := func(name string) func(a []uint64) uint64 {
			return func(a []uint64) uint64 {
				el := rt.unwrap(a[0]).(*jsArray).elems
				pos := el[0].(*jsArray).elems
				if keys, _, ok := dictParts(el[1]); ok && len(keys.elems) > 0 {
					pos = rt.pyKwPos(name, pos, el[1], []string{"source", "encoding", "errors"}, 0)
				}
				return rt.wrap(rt.pyBytesNew(pos, name))
			}
		}
		m["js_pybytes"] = bytesNew("bytes")
		// slice(...), bound as [positionals, keywords].
		m["js_pyslicenew"] = func(a []uint64) uint64 {
			el := rt.unwrap(a[0]).(*jsArray).elems
			if keys, _, ok := dictParts(el[1]); ok && len(keys.elems) > 0 {
				rt.pyRaise("TypeError", "slice() takes no keyword arguments")
			}
			pos := el[0].(*jsArray).elems
			switch {
			case len(pos) == 0:
				rt.pyRaise("TypeError", "slice expected at least 1 argument, got 0")
			case len(pos) > 3:
				rt.pyRaise("TypeError", fmt.Sprintf("slice expected at most 3 arguments, got %d", len(pos)))
			case len(pos) == 1:
				return rt.wrap(pySliceOf(jsUndef, pos[0], jsUndef))
			}
			return rt.wrap(pySliceOf(pos[0], pos[1], argAt(pos, 2)))
		}
		m["js_pybytearray"] = bytesNew("bytearray")
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

// pyUnionArgs is what an operand of X | Y contributes to a union: None stands
// as NoneType, a union its own arguments, a type object or a generic alias
// itself; nil for anything else.
func (rt *jsrt) pyUnionArgs(v interface{}) []interface{} {
	if isUndefOrNull(v) {
		return []interface{}{rt.pyBuiltinClass("NoneType")}
	}
	if u, ok := pyUnionOf(v); ok {
		return u.elems
	}
	if _, isCls := rt.pyClassName(v); isCls {
		return []interface{}{v}
	}
	if o, ok := v.(*jsObject); ok && o.props["__galias"] != nil {
		return []interface{}{v}
	}
	return nil
}

// pyUnionOther is what the operand beside a UNION contributes when it is not a
// type: CPython's Union[...] takes any value as an argument but a tuple (and a
// str, which it would make a ForwardRef this value model has no type for).
func (rt *jsrt) pyUnionOther(v interface{}) []interface{} {
	if pyIsTup(v) {
		rt.pyRaise("TypeError", "Union[arg, ...]: each arg must be a type. Got "+rt.pyRepr(v)+".")
	}
	if _, isStr := v.(string); isStr {
		return nil
	}
	return []interface{}{v}
}

// pyUnion is X | Y over type objects and None, CPython's typing.Union: an
// object whose __pyunion holds the arguments flattened and deduplicated in
// order; a single argument is that type itself. nil when an operand is not a
// type, None or a union, or both are None. Twin of pyUnion in
// languages/python-interpreter.abnf and languages/lib/python-rt.metajs.
func (rt *jsrt) pyUnion(l, r interface{}) interface{} {
	if isUndefOrNull(l) && isUndefOrNull(r) {
		return nil
	}
	a, b := rt.pyUnionArgs(l), rt.pyUnionArgs(r)
	if _, isU := pyUnionOf(r); a == nil && isU {
		a = rt.pyUnionOther(l)
	}
	if _, isU := pyUnionOf(l); b == nil && isU {
		b = rt.pyUnionOther(r)
	}
	if a == nil || b == nil {
		return nil
	}
	var out []interface{}
	for _, t := range append(append([]interface{}{}, a...), b...) {
		seen := false
		for _, s := range out {
			seen = seen || s == t || rt.pyEqual(s, t)
		}
		if !seen {
			out = append(out, t)
		}
	}
	if len(out) == 1 {
		return out[0]
	}
	u := newJSObject()
	u.set("__pyunion", &jsArray{elems: out})
	return u
}

// pyUnionOf reports whether v is a union, and its arguments.
func pyUnionOf(v interface{}) (*jsArray, bool) {
	if o, ok := v.(*jsObject); ok {
		a, isU := o.props["__pyunion"].(*jsArray)
		return a, isU
	}
	return nil, false
}

// pyUnionStr renders a union as CPython does: int | None.
func (rt *jsrt) pyUnionStr(u *jsArray) string {
	parts := make([]string, len(u.elems))
	for i, c := range u.elems {
		parts[i] = rt.pyTypeArgRepr(c)
	}
	return strings.Join(parts, " | ")
}

// pyUnionEq: two unions are equal when they hold the same types, in any order.
func pyUnionEq(a, b *jsArray) bool {
	if len(a.elems) != len(b.elems) {
		return false
	}
	for _, x := range a.elems {
		found := false
		for _, y := range b.elems {
			found = found || x == y
		}
		if !found {
			return false
		}
	}
	return true
}

// pyObjMethodNames are object's own methods, the four orderings among them.
// CPython's object type carries them and every class inherits them: A.__init__ is object.__init__, and
// super().__init__() works in a class whose only base is object. The class
// chains here leave object out, so an attribute READ that finds nothing on the
// chain asks pyObjectMethod last; the runtime's own dispatch (==, hash(), str())
// keeps its defaults, which these functions agree with. They render as slot
// wrappers ("~" in the builtin render table), __new__ as a builtin. Twin of
// pyObjectMethod in languages/python-interpreter.abnf and
// languages/lib/python-rt.metajs.
var pyObjMethodNames = []string{"__init__", "__new__", "__eq__", "__ne__", "__hash__", "__repr__", "__str__",
	"__format__", "__setattr__", "__getattribute__", "__delattr__", "__lt__", "__le__", "__gt__", "__ge__"}

// pyObjectMethod is object's method called name, one function per name, or nil.
func (rt *jsrt) pyObjectMethod(name string) interface{} {
	if rt.pyObjMethods == nil {
		rt.pyObjMethods = map[string]interface{}{}
		if rt.pyBuiltinRender == nil {
			rt.pyBuiltinRender = map[interface{}]string{}
		}
		for _, n := range pyObjMethodNames {
			h := jsHostFunc(n, pyObjMethodBody(n))
			rt.pyObjMethods[n] = h
			rt.pyBuiltinRender[h] = "~" + n
		}
		rt.pyBuiltinRender[rt.pyObjMethods["__new__"]] = "__new__"
	}
	return rt.pyObjMethods[name]
}

func pyObjMethodBody(name string) func(rt *jsrt, this uint64, args []interface{}) interface{} {
	arg := func(args []interface{}, i int) interface{} {
		if i < len(args) {
			return args[i]
		}
		return jsUndef
	}
	switch name {
	case "__init__":
		// Extra arguments are an error once the class defines its own __init__,
		// or does not define __new__ (CPython's object_init).
		return func(rt *jsrt, this uint64, args []interface{}) interface{} {
			if _, cls, ok := pyInstance(arg(args, 0)); ok && len(args) > 1 {
				if _, own := pyLookup(cls, "__init__"); own {
					rt.pyRaise("TypeError", "object.__init__() takes exactly one argument (the instance to initialize)")
				}
				if _, nw := pyLookup(cls, "__new__"); !nw {
					rt.pyRaise("TypeError", rt.toString(cls.props["__name"])+
						".__init__() takes exactly one argument (the instance to initialize)")
				}
			}
			return jsUndef
		}
	case "__new__":
		return func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return rt.pyObjNew(arg(args, 0), pyArgsAfterFirst(args))
		}
	case "__eq__":
		return func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return pyObjEq(rt, arg(args, 0), arg(args, 1))
		}
	case "__ne__":
		// The negation of the receiver type's own __eq__, NotImplemented passed on.
		return func(rt *jsrt, this uint64, args []interface{}) interface{} {
			a, b := arg(args, 0), arg(args, 1)
			var r interface{}
			if _, cls, ok := pyInstance(a); !ok {
				r = rt.pyEqual(a, b)
			} else if eq, found := pyLookup(cls, "__eq__"); found {
				r = rt.call(eq, jsUndef, []interface{}{a, b})
			} else {
				r = pyObjEq(rt, a, b)
			}
			if r == pyNotImplemented {
				return r
			}
			return !rt.pyTruthyOf(r)
		}
	case "__hash__":
		return func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return rt.pyHashInt(rt.pyHashId(arg(args, 0)))
		}
	case "__repr__":
		return func(rt *jsrt, this uint64, args []interface{}) interface{} {
			if _, cls, ok := pyInstance(arg(args, 0)); ok {
				return "<" + rt.pyClsQual(cls) + " object>"
			}
			return "<" + rt.pyOwnerName(arg(args, 0)) + " object>"
		}
	case "__str__":
		return func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return rt.pyRepr(arg(args, 0))
		}
	case "__format__":
		return func(rt *jsrt, this uint64, args []interface{}) interface{} {
			if spec := arg(args, 1); !isUndefOrNull(spec) && rt.toString(spec) != "" {
				rt.pyRaise("TypeError", "unsupported format string passed to "+rt.pyOwnerName(arg(args, 0))+".__format__")
			}
			return rt.pyString(arg(args, 0))
		}
	case "__setattr__":
		return func(rt *jsrt, this uint64, args []interface{}) interface{} {
			rt.pySetAttrRaw(arg(args, 0), rt.toString(arg(args, 1)), arg(args, 2))
			return jsUndef
		}
	case "__getattribute__":
		return func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return rt.pyGetAttrRaw(arg(args, 0), rt.toString(arg(args, 1)))
		}
	case "__lt__", "__le__", "__gt__", "__ge__":
		// object's four orderings decline every pair.
		return func(rt *jsrt, this uint64, args []interface{}) interface{} { return pyNotImplemented }
	}
	return func(rt *jsrt, this uint64, args []interface{}) interface{} { // __delattr__
		rt.pyDelAttrRaw(arg(args, 0), rt.toString(arg(args, 1)))
		return jsUndef
	}
}

// pyArgsAfterFirst is args without its first element.
func pyArgsAfterFirst(args []interface{}) []interface{} {
	if len(args) == 0 {
		return nil
	}
	return args[1:]
}

// pyObjEq is object.__eq__: True for the very same object, else NotImplemented.
func pyObjEq(rt *jsrt, a, b interface{}) interface{} {
	if rt.strictEq(a, b) {
		return true
	}
	return pyNotImplemented
}

// pyObjNew is object.__new__(cls, *rest): a bare instance of cls, its __init__
// not run. A builtin value type makes its own instances, and extra arguments
// follow CPython's object_new.
func (rt *jsrt) pyObjNew(c interface{}, rest []interface{}) interface{} {
	name, isCls := rt.pyClassName(c)
	if !isCls {
		rt.pyRaise("TypeError", "object.__new__(X): X is not a type object ("+pyTypeName(rt, c)+")")
	}
	cls := c.(*jsObject)
	if _, user := pyClassObj(cls); (!user && name != "object") || (user && pyBuiltinBase(cls) != "") {
		// CPython names the first class on the base chain whose __new__ is not
		// one a class body defined.
		sb := cls
		for {
			_, ownNew := sb.props["__new__"]
			base, hasBase := sb.props["__super"].(*jsObject)
			if _, user := pyClassObj(sb); !user || !ownNew || !hasBase {
				break
			}
			sb = base
		}
		rt.pyRaise("TypeError", "object.__new__("+name+") is not safe, use "+rt.toString(sb.props["__name"])+".__new__()")
	}
	inst := newJSObject()
	inst.set("__class", cls)
	if _, user := pyClassObj(cls); user && pyIsExcClass(cls) {
		inst.set("args", pyTup(&jsArray{elems: append([]interface{}{}, rest...)}))
		return inst
	}
	if len(rest) > 0 {
		if _, own := pyLookup(cls, "__new__"); own {
			rt.pyRaise("TypeError", "object.__new__() takes exactly one argument (the type to instantiate)")
		}
		if _, init := pyLookup(cls, "__init__"); !init {
			rt.pyRaise("TypeError", name+"() takes no arguments")
		}
	}
	return inst
}

// pyBuiltinNew is T.__new__(cls, *args) of a builtin value type: T's conversion
// for T itself, and for a subclass a bare instance holding it, __init__ not run.
// One function per type.
func (rt *jsrt) pyBuiltinNew(t *jsObject) interface{} {
	if f, ok := rt.pyBNews[t]; ok {
		return f
	}
	tn, _ := t.props["__name"].(string)
	h := jsHostFunc("__new__", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		var c interface{} = jsUndef
		if len(args) > 0 {
			c = args[0]
		}
		rest := pyArgsAfterFirst(args)
		name, isCls := rt.pyClassName(c)
		if !isCls {
			rt.pyRaise("TypeError", tn+".__new__(X): X is not a type object ("+pyTypeName(rt, c)+")")
		}
		if c == t {
			return rt.pyBuiltinVal(tn, rest, jsUndef)
		}
		cls, user := pyClassObj(c)
		if !user || pyBuiltinBase(cls) != tn {
			rt.pyRaise("TypeError", tn+".__new__("+name+"): "+name+" is not a subtype of "+tn)
		}
		inst := newJSObject()
		inst.set("__class", cls)
		inst.set("__val", rt.pyBuiltinVal(tn, rest, jsUndef))
		return inst
	})
	if rt.pyBNews == nil {
		rt.pyBNews = map[*jsObject]interface{}{}
	}
	rt.pyBNews[t] = h
	if rt.pyBuiltinRender == nil {
		rt.pyBuiltinRender = map[interface{}]string{}
	}
	rt.pyBuiltinRender[h] = "__new__"
	return h
}

// pyClsNew is the __new__ a class answers: its own, a builtin base's, or
// object's.
func (rt *jsrt) pyClsNew(cls *jsObject) interface{} {
	if own, found := pyLookup(cls, "__new__"); found {
		return own
	}
	bt := pyBuiltinClsName(cls)
	if bt == "" {
		bt = pyBuiltinBase(cls)
	}
	if bt != "" && bt != "object" {
		return rt.pyBuiltinNew(rt.pyBuiltinClass(bt))
	}
	return rt.pyObjectMethod("__new__")
}

// pyClassFallback is what a class answers for a name its chain does not hold:
// its __new__, its type (every class here is an instance of type), or one of
// object's methods; nil otherwise.
func (rt *jsrt) pyClassFallback(cls *jsObject, name string) interface{} {
	switch name {
	case "__new__":
		return rt.pyClsNew(cls)
	case "__class__":
		return rt.pyBuiltinClass("type")
	}
	return rt.pyObjectMethod(name)
}

// pyBindObjectMethod binds one of object's methods to an instance: a builtin
// bound method (a method-wrapper for a slot) with no __func__.
func (rt *jsrt) pyBindObjectMethod(self, m interface{}, name string) interface{} {
	b := rt.pyBindMethod(self, m, name)
	b.(*hostFunc).pybm.fn = nil
	return b
}

// pyNoteHook counts a class member that is an attribute hook, so the attribute
// paths only look for one once a class has defined it. Twin of pyNoteHook in
// python-interpreter.abnf and pyDNoteHook in lib/python-rt.metajs.
func (rt *jsrt) pyNoteHook(name string) {
	switch name {
	case "__getattribute__", "__getattr__", "__setattr__", "__delattr__":
		rt.pyHooks++
	}
}

// pyUserHook is the hook NAME an instance's class (not object) defines.
func (rt *jsrt) pyUserHook(obj interface{}, name string) (interface{}, bool) {
	if rt.pyHooks == 0 {
		return nil, false
	}
	_, cls, ok := pyInstance(obj)
	if !ok {
		return nil, false
	}
	return pyLookup(cls, name)
}

// pyHookGet is CPython's slot_tp_getattr_hook: obj.name through the class's
// __getattribute__ (else object's), and on an AttributeError through its
// __getattr__. ok is false when the class defines neither.
func (rt *jsrt) pyHookGet(obj interface{}, name string) (v interface{}, ok bool) {
	ga, hasGA := rt.pyUserHook(obj, "__getattribute__")
	gf, hasGF := rt.pyUserHook(obj, "__getattr__")
	if !hasGA && !hasGF {
		return nil, false
	}
	first := func() interface{} {
		if hasGA {
			return rt.callPyValue(ga, []interface{}{obj, name}, jsUndef)
		}
		return rt.pyGetAttrRaw(obj, name)
	}
	if !hasGF {
		return first(), true
	}
	if v, missed := rt.pyTryAttr(first); !missed {
		return v, true
	}
	return rt.callPyValue(gf, []interface{}{obj, name}, jsUndef), true
}

// pyTryAttr runs f, reporting an AttributeError it raises as missed.
func (rt *jsrt) pyTryAttr(f func() interface{}) (v interface{}, missed bool) {
	defer func() {
		if p := recover(); p != nil {
			if th, isThrown := p.(*jsThrown); isThrown && rt.pyIsInstance(th.value, rt.pyExcClass("AttributeError")) {
				missed = true
				return
			}
			panic(p)
		}
	}()
	return f(), false
}

// pyHookHas is hasattr() over a class with a __getattribute__ or __getattr__:
// whether reading the attribute raises AttributeError.
func (rt *jsrt) pyHookHas(obj interface{}, name string) bool {
	_, missed := rt.pyTryAttr(func() interface{} { return rt.pyGetAttr(obj, name) })
	return !missed
}

// pyHookMethod is the callable obj.name(...) calls when the class's hooks
// decide it: always under __getattribute__, and under __getattr__ only for a
// name the ordinary lookup does not find.
func (rt *jsrt) pyHookMethod(obj interface{}, name string) (interface{}, bool) {
	if rt.pyHooks == 0 {
		return nil, false
	}
	if _, ok := rt.pyUserHook(obj, "__getattribute__"); ok {
		return rt.pyGetAttr(obj, name), true
	}
	if _, ok := rt.pyUserHook(obj, "__getattr__"); ok && !rt.pyHasAttrRaw(obj, name) {
		return rt.pyGetAttr(obj, name), true
	}
	return nil, false
}

// pyDelAttr is `del obj.name`, through the class's __delattr__ when it defines
// one.
func (rt *jsrt) pyDelAttr(obj interface{}, name string) {
	if f, ok := rt.pyUserHook(obj, "__delattr__"); ok {
		rt.callPyValue(f, []interface{}{obj, name}, jsUndef)
		return
	}
	rt.pyDelAttrRaw(obj, name)
}

// pyDelAttrRaw is object.__delattr__.
func (rt *jsrt) pyDelAttrRaw(obj interface{}, name string) {
	if isCallable(obj) {
		d, i := rt.pyFuncAttrIdx(obj, name)
		if i < 0 {
			rt.pyRaise("AttributeError", "'"+pyTypeName(rt, obj)+"' object has no attribute '"+name+"'")
		}
		keys, vals, _ := dictParts(d)
		keys.dropIdx()
		keys.elems = append(keys.elems[:i], keys.elems[i+1:]...)
		vals.elems = append(vals.elems[:i], vals.elems[i+1:]...)
		return
	}
	_, cls, isInst := pyInstance(obj)
	if !isInst {
		rt.pyNoDictAttr(obj, name)
	} else if pyIsObjectCls(cls) {
		panic(&jsThrown{value: rt.pyAttrError(cls, name)})
	} else if m, found := pyLookup(cls, name); found {
		if p, isProp := pyPropOf(m); isProp {
			rt.pyPropDel(p, obj, cls)
			return
		}
	}
	o, ok := obj.(*jsObject)
	if !ok {
		rt.fail("attribute deletion on a %s", rt.typeOf(obj))
	}
	// Only an OWN attribute can be deleted: a class's member is not the
	// instance's, nor a base's the subclass's.
	if _, has := o.props[name]; !has {
		if _, isCls := pyClassObj(o); isCls {
			rt.pyRaise("AttributeError", "type object '"+rt.toString(o.props["__name"])+"' has no attribute '"+name+"'")
		}
		if isInst {
			rt.pyRaise("AttributeError", "'"+pyTypeName(rt, obj)+"' object has no attribute '"+name+"'")
		}
	}
	delete(o.props, name)
	for i, k := range o.keys {
		if k == name {
			o.keys = append(o.keys[:i], o.keys[i+1:]...)
			break
		}
	}
	if d, has := o.props["__pydict"].(*jsObject); has {
		pyInstDictPull(o, d)
	}
}
