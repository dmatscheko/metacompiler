package abnf

// jsrtdart.go -- the Dart half of the value model: Object.toString() rendering and
// const canonicalization.
//
// This is the Go twin of the `dstr` / `dcanon` pair in
// languages/dart-interpreter.abnf. Only languages/dart-to-llvm-ir.abnf emits these
// externs, and nothing here rebinds an existing one: the registrar below only ADDS
// js_dart* names, so every other grammar emits and runs exactly as before.
//
// Why it has to exist at all. The compiler half used to hand a RAW handle to the
// shared println host function, so Go's %v reached stdout - `print(null)` said
// "<nil>", `print([1, 2, 3])` said "[1 2 3]", a Map said
// "map[__dict:true keys:[a] vals:[1]]" and an instance dumped its whole class
// descriptor INCLUDING A GO POINTER ADDRESS, which is not even reproducible between
// two runs of the same program. (`print(SomeClass)` did not print at all: the
// descriptor's __class back-reference sent toGoNatural into a stack overflow.)
// Each language wants a different answer here, so the fix is a per-language print
// extern - the shape js_rputs / js_rprint (Ruby) and js_luaprint already have - and
// not a change to the shared printArgs.
//
// The rendering implemented here is Dart's Object.toString() and its overrides:
//
//	null            null                     Instance of 'E'   an object with no
//	true / false    true / false                               user toString()
//	[1, 2, 3]       a List                   E                 a Type (a class)
//	{a: 1}          a Map                    Col.red           an enum value
//	{1, 2}          a Set                    (1, a)            a Record
//
// Strings are NOT quoted inside a collection (`print(['a'])` is `[a]`), which is
// what makes this different from Python's repr-based rendering in pyString.
// Corpus citations, from tests/reference/dart/dart-language-tests:
//   language/regress/regress45642_test.dart:14   "should print: Instance of 'Foo'"
//   language/records/simple/literals_and_field_access_test.dart:121
//                                                x.toString() == "[1, 2, 3]"
//   language/enum/enum_test.dart:62              'Enum2.A' == Enum2.A.toString()
//   language/enum/enhanced_enums_basic_test.dart:143  a user toString() wins
//
// Const canonicalization (js_dartcanon) is keyed on the class and the FIELD VALUES,
// not on the constructor, because language/const/ct_const_test.dart:62 demands
//
//	Expect.equals(true, identical(const Point.X(5), const Point(5, 0)));
//
// - two different constructors, one canonical instance. The same rule covers const
// collections: language/canonicalize/const_test.dart asserts
// identical(const <int>[1, 2], const <int>[1, 2]) and
// identical(const {"a": 1, "b": 2}, const {"a": 1, "b": 2}).

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

func init() {
	rxExtraExterns = append(rxExtraExterns, func(rt *jsrt, m map[string]func(args []uint64) uint64) {
		rt.addDartExterns(m)
	})
}

// ===== Object.toString() =====

// dartCycleDepth caps the recursion of both the renderer and the key builder. Dart
// itself renders a self-referential collection as "[...]"; the depth cap is the
// belt to that braces, so a value model surprise degrades into a printable string
// rather than a stack overflow.
const dartCycleDepth = 64

func (rt *jsrt) dartStr(v interface{}) string { return rt.dartStrIn(v, nil, 0) }

func (rt *jsrt) dartStrIn(v interface{}, seen []interface{}, depth int) string {
	if depth > dartCycleDepth {
		return "..."
	}
	switch t := v.(type) {
	case jsUndefT, jsNullT:
		return "null"
	case bool:
		if t {
			return "true"
		}
		return "false"
	case string:
		return t
	case *jsArray:
		if dartSeen(seen, v) {
			return "[...]"
		}
		seen = append(seen, v)
		parts := make([]string, 0, len(t.elems))
		for _, e := range t.elems {
			parts = append(parts, rt.dartStrIn(e, seen, depth+1))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case *jsObject:
		if dartSeen(seen, v) {
			return "{...}"
		}
		return rt.dartObjStr(t, append(seen, v), depth)
	}
	if isCallable(v) {
		return "Closure"
	}
	return rt.toString(v)
}

func dartSeen(seen []interface{}, v interface{}) bool {
	for _, s := range seen {
		if s == v {
			return true
		}
	}
	return false
}

func (rt *jsrt) dartObjStr(o *jsObject, seen []interface{}, depth int) string {
	// A class descriptor used as a value is a Type, and Dart prints a Type as its
	// name. This case has to come first: the descriptor's own members would
	// otherwise be walked, and it points back at itself through every instance.
	if t, ok := o.props["__isclass"]; ok && t == true {
		if n, isStr := o.props["__name"].(string); isStr {
			return n
		}
	}
	if els, ok := pySetElems(o); ok {
		parts := make([]string, 0, len(els.elems))
		for _, e := range els.elems {
			parts = append(parts, rt.dartStrIn(e, seen, depth+1))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	}
	if keys, vals, ok := dictParts(o); ok {
		parts := make([]string, 0, len(keys.elems))
		for i := range keys.elems {
			parts = append(parts, rt.dartStrIn(keys.elems[i], seen, depth+1)+": "+
				rt.dartStrIn(vals.elems[i], seen, depth+1))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	}
	cls, hasCls := o.props["__class"]
	if !hasCls {
		return rt.toString(o)
	}
	// A user toString() anywhere on the __super chain wins, for a plain class and
	// for an enum alike (enhanced_enums_basic_test.dart:143).
	for c := cls; c != nil; {
		co, isObj := c.(*jsObject)
		if !isObj {
			break
		}
		if mth, ok := co.props["toString"]; ok && isCallable(mth) {
			return rt.toString(rt.call(mth, jsUndef, []interface{}{o}))
		}
		c = co.props["__super"]
	}
	name := "Object"
	if co, isObj := cls.(*jsObject); isObj {
		if n, isStr := co.props["__name"].(string); isStr {
			name = n
		}
	}
	// An enum value prints as EnumName.valueName. __enumName is written by the enum
	// emitter of dart-to-llvm-ir.abnf beside the public `name`, so an ordinary class
	// that happens to declare a field called `name` is not mistaken for one.
	if en, ok := o.props["__enumName"].(string); ok {
		return name + "." + en
	}
	// A record renders as its field list: positional fields (keyed "$1", "$2", ...)
	// as bare values, named ones as "name: value".
	if name == "Record" {
		if keys, vals, ok := dictParts(o.props["__f"]); ok {
			parts := make([]string, 0, len(keys.elems))
			for i := range keys.elems {
				k := rt.toString(keys.elems[i])
				val := rt.dartStrIn(vals.elems[i], seen, depth+1)
				if strings.HasPrefix(k, "$") {
					parts = append(parts, val)
				} else {
					parts = append(parts, k+": "+val)
				}
			}
			return "(" + strings.Join(parts, ", ") + ")"
		}
	}
	return "Instance of '" + name + "'"
}

// ===== const canonicalization =====

// dartCanon returns the canonical instance for a const value: the first value
// registered under its structural key. Children are canonicalized FIRST, so a
// nested const creation is canonical too and the key of the parent is built out of
// values that are already shared.
//
// A value the language cannot make const (a closure, a plain host object) is
// returned unchanged and never enters the cache.
func (rt *jsrt) dartCanon(v interface{}, consts map[string]interface{}, depth int) interface{} {
	if depth > dartCycleDepth {
		return v
	}
	switch t := v.(type) {
	case *jsArray:
		for i, e := range t.elems {
			t.elems[i] = rt.dartCanon(e, consts, depth+1)
		}
	case *jsObject:
		if isc, ok := t.props["__isclass"]; ok && isc == true {
			return v // a Type is already canonical
		}
		if els, ok := pySetElems(t); ok {
			for i, e := range els.elems {
				els.elems[i] = rt.dartCanon(e, consts, depth+1)
			}
		} else if keys, vals, ok := dictParts(t); ok {
			for i := range keys.elems {
				keys.elems[i] = rt.dartCanon(keys.elems[i], consts, depth+1)
				vals.elems[i] = rt.dartCanon(vals.elems[i], consts, depth+1)
			}
		} else if _, hasCls := t.props["__class"]; hasCls {
			for _, k := range t.keys {
				if strings.HasPrefix(k, "__") {
					continue
				}
				t.props[k] = rt.dartCanon(t.props[k], consts, depth+1)
			}
			if f, ok := t.props["__f"]; ok { // a record's field dict
				rt.dartCanon(f, consts, depth+1)
			}
		} else {
			return v
		}
	default:
		return v
	}
	k := rt.dartConstKey(v, 0)
	if c, ok := consts[k]; ok {
		return c
	}
	consts[k] = v
	return v
}

// dartConstKey is the structural identity of a const value. It is never printed;
// it only has to separate values that are not identical and join values that are.
func (rt *jsrt) dartConstKey(v interface{}, depth int) string {
	if depth > dartCycleDepth {
		return "!"
	}
	switch t := v.(type) {
	case jsUndefT, jsNullT:
		return "n"
	case bool:
		if t {
			return "b1"
		}
		return "b0"
	case float64:
		return "d" + strconv.FormatFloat(t, 'g', -1, 64)
	case string:
		return "s" + strconv.Itoa(len(t)) + ":" + t
	case *jsArray:
		parts := make([]string, 0, len(t.elems))
		for _, e := range t.elems {
			parts = append(parts, rt.dartConstKey(e, depth+1))
		}
		return "L(" + strings.Join(parts, ",") + ")"
	case *jsObject:
		if isc, ok := t.props["__isclass"]; ok && isc == true {
			return "T:" + rt.toString(t.props["__name"])
		}
		if els, ok := pySetElems(t); ok {
			parts := make([]string, 0, len(els.elems))
			for _, e := range els.elems {
				parts = append(parts, rt.dartConstKey(e, depth+1))
			}
			return "S(" + strings.Join(parts, ",") + ")"
		}
		if keys, vals, ok := dictParts(t); ok {
			parts := make([]string, 0, len(keys.elems))
			for i := range keys.elems {
				parts = append(parts, rt.dartConstKey(keys.elems[i], depth+1)+"=>"+
					rt.dartConstKey(vals.elems[i], depth+1))
			}
			return "M(" + strings.Join(parts, ",") + ")"
		}
		if cls, ok := t.props["__class"]; ok {
			name := ""
			if co, isObj := cls.(*jsObject); isObj {
				name = rt.toString(co.props["__name"])
			}
			// The field ORDER of an instance depends on which constructor ran, and
			// two different constructors must produce the same key
			// (ct_const_test.dart:62), so the names are sorted.
			names := make([]string, 0, len(t.keys))
			for _, k := range t.keys {
				if !strings.HasPrefix(k, "__") {
					names = append(names, k)
				}
			}
			sort.Strings(names)
			parts := make([]string, 0, len(names))
			for _, k := range names {
				parts = append(parts, k+"="+rt.dartConstKey(t.props[k], depth+1))
			}
			return "C:" + name + "(" + strings.Join(parts, ",") + ")"
		}
		return fmt.Sprintf("O:%p", t)
	}
	return fmt.Sprintf("X:%p", v)
}

// ===== The externs =====

func (rt *jsrt) addDartExterns(m map[string]func(args []uint64) uint64) {
	u := rt.unwrap
	w := rt.wrap
	// One canonical-const table per runtime, so a value canonicalized in one
	// program cannot leak into the next.
	consts := map[string]interface{}{}

	// print(x): Dart's print writes x.toString() followed by a newline.
	m["js_dartprint"] = func(a []uint64) uint64 {
		fmt.Fprintln(outWriter, wtf8Clean(rt.dartStr(u(a[0]))))
		return 0
	}
	// The same rendering as a value: string interpolation and an explicit
	// .toString() both go through it.
	m["js_dartstr"] = func(a []uint64) uint64 { return rt.wrapStr(rt.dartStr(u(a[0]))) }
	// const e: the canonical value for e.
	m["js_dartcanon"] = func(a []uint64) uint64 { return w(rt.dartCanon(u(a[0]), consts, 0)) }
}
