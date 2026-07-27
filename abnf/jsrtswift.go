package abnf

// jsrtswift.go -- Swift's VALUE RENDERING and Swift's `==` for the compiler
// grammar (languages/swift-to-llvm-ir.abnf).
//
// swift-to-llvm-ir.abnf used to bind Swift's `print` straight to the shared
// `println` host function, which hands the raw runtime value to Go's `%v`. That
// printed `<nil>` for nil, `[1 2 3]` for an array, `map[__dict:true keys:[x]
// vals:[1]]` for a dictionary and - worst - a map containing a raw Go POINTER
// address for a struct or class instance, so the compiler's stdout was not even
// reproducible between two runs of the same program. Real Swift prints
//
//	nil / [1, 2, 3] / ["x": 1] / P(x: 1, y: 2) / a(1, "x") / (x: 1, y: "s")
//
// (verified against swift 6.1.2 - see the probe output in the commit that added
// this file). The interpreter half was wrong in its own way: its `swstr` fell
// through to JavaScript's ToString, so an array printed "1,2,3" and everything
// else "[object Object]". Both halves now render through the same rules; this
// file is the Go twin of `swDesc` / `swDescIn` in swift-interpreter.abnf and the
// two MUST stay in step, because ./test.sh --cross diffs them.
//
// Registration is additive: an init() appends to rxExtraExterns, so nothing here
// runs unless a program calls one of the js_sw* externs, and only the Swift
// compiler grammar emits them. No existing extern is renamed, removed or
// rebound.
//
// Known simplifications, both DELIBERATE and both shared by the two halves:
//
//   - A class instance prints as its bare type name. Real Swift prints the
//     module-qualified name (`main.C` / `sp2.C`), and this runtime has no module
//     concept at all; the bare name is the reproducible part of that answer.
//   - There is no Optional BOX in this value model - `Int?` is just the value or
//     nil - so a non-nil optional prints as the wrapped value where real Swift
//     prints `Optional(3)`. Recovering that needs the declared type at the print
//     site, which is a type-checker job this front end does not do. `nil` itself
//     prints as `nil`, which is right.
//   - A Dictionary prints in INSERTION order. Swift's order is unspecified (it is
//     the hash order and varies per run), so any fixed order is as correct as any
//     other and insertion order is the one both halves can reproduce.

import (
	"fmt"
	"strings"
)

func init() {
	rxExtraExterns = append(rxExtraExterns, func(rt *jsrt, m map[string]func(args []uint64) uint64) {
		u := rt.unwrap
		boolH := func(b bool) uint64 {
			if b {
				return jsHTrue
			}
			return jsHFalse
		}

		// String(describing:) / "\(v)" / the thing print writes.
		m["js_swdesc"] = func(a []uint64) uint64 { return rt.wrapStr(rt.swDesc(u(a[0]))) }

		// Swift's print(_:separator:terminator:). a[0] is the argument ARRAY,
		// a[1] the separator and a[2] the terminator; undefined means the
		// default (" " and "\n"). The labelled arguments used to be passed
		// positionally to the shared println, so `print(1, 2, separator: "-")`
		// printed `1 2 -`.
		m["js_swprint"] = func(a []uint64) uint64 {
			args, ok := u(a[0]).(*jsArray)
			if !ok {
				rt.fail("js_swprint needs an argument array")
			}
			sep := " "
			if s := u(a[1]); !isNullish(s) {
				sep = rt.swDesc(s)
			}
			term := "\n"
			if t := u(a[2]); !isNullish(t) {
				term = rt.swDesc(t)
			}
			out := ""
			for i, e := range args.elems {
				if i > 0 {
					out += sep
				}
				out += rt.swDesc(e)
			}
			fmt.Fprint(outWriter, wtf8Clean(out+term))
			return 0
		}

		// Swift's ==. `js_seq` compares an Array, a Dictionary and a tuple by
		// IDENTITY, so `[1, 2] == [1, 2]` was always false in the compiler; in
		// Swift all three are value types and compare element by element.
		m["js_sweq"] = func(a []uint64) uint64 { return boolH(rt.swEqual(u(a[0]), u(a[1]))) }
	})
}

// ----------------------------------------------------------------------------
// Rendering

// swDesc is Swift's `String(describing:)`: the TOP-LEVEL form, in which a String
// renders as itself. swDescIn is the form used for an element inside a
// collection, a tuple or a struct, where a String is quoted and escaped.
func (rt *jsrt) swDesc(v interface{}) string { return rt.swRender(v, false, 0) }

func (rt *jsrt) swDescIn(v interface{}) string { return rt.swRender(v, true, 0) }

// swQuote is Swift's debugDescription of a String: double quotes and the four
// escapes Swift itself emits.
func swQuote(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`, "\t", `\t`, "\r", `\r`, "\x00", `\0`)
	return `"` + r.Replace(s) + `"`
}

// swRender does the work. `nested` selects the quoted form for strings; `depth`
// only guards against a cyclic object graph (a class instance can point back at
// its owner) so the renderer cannot recurse forever.
func (rt *jsrt) swRender(v interface{}, nested bool, depth int) string {
	if depth > 32 {
		return "..."
	}
	switch t := v.(type) {
	case jsNullT, jsUndefT:
		return "nil"
	case string:
		if nested {
			return swQuote(t)
		}
		return t
	case *jsArray:
		parts := make([]string, len(t.elems))
		for i, e := range t.elems {
			parts[i] = rt.swRender(e, true, depth+1)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case *jsObject:
		if s, ok := rt.swObjDesc(t, depth); ok {
			return s
		}
	}
	return rt.toString(v)
}

// swObjDesc renders the object shapes the two Swift grammars build. It reports
// false for a shape it does not know, which falls back to the generic ToString.
func (rt *jsrt) swObjDesc(o *jsObject, depth int) (string, bool) {
	// A user `description` (CustomStringConvertible) wins over every built-in
	// rendering, exactly as it does in Swift. Both halves store a computed
	// property as the method __get_<name> on the type descriptor.
	if g, ok := swFindMember(o, "__get_description"); ok {
		return rt.swDesc(rt.call(g, jsUndef, []interface{}{o})), true
	}
	if d, ok := o.props["description"]; ok && !isCallable(d) {
		return rt.swDesc(d), true
	}

	// Dictionary: ["k": v, ...], and the empty one is [:].
	if keys, vals, ok := dictParts(o); ok {
		if len(keys.elems) == 0 {
			return "[:]", true
		}
		parts := make([]string, 0, len(keys.elems))
		for i, k := range keys.elems {
			var val interface{} = jsUndef
			if i < len(vals.elems) {
				val = vals.elems[i]
			}
			parts = append(parts, rt.swRender(k, true, depth+1)+": "+rt.swRender(val, true, depth+1))
		}
		return "[" + strings.Join(parts, ", ") + "]", true
	}

	// Tuple: (1, "two") / (x: 1, y: "s"). The element labels ride along as
	// __tlabels when the tuple has any (an unlabelled slot is "").
	if n, ok := swNumProp(o, "__tuple"); ok {
		labels := swStrArray(o.props["__tlabels"])
		parts := make([]string, 0, n)
		for i := 0; i < n; i++ {
			s := rt.swRender(o.props[itoa(i)], true, depth+1)
			if i < len(labels) && labels[i] != "" {
				s = labels[i] + ": " + s
			}
			parts = append(parts, s)
		}
		return "(" + strings.Join(parts, ", ") + ")", true
	}

	// An enum case: `c`, `a(1, "x")`, `b(n: 2)`. A raw-value enum still prints
	// the CASE NAME, not the raw value.
	if tag, ok := o.props["__ecase"]; ok && tag == true {
		name, _ := o.props["__ename"].(string)
		vals, _ := o.props["__vals"].(*jsArray)
		if vals == nil || len(vals.elems) == 0 {
			return name, true
		}
		labels := swStrArray(o.props["__labels"])
		parts := make([]string, 0, len(vals.elems))
		for i, e := range vals.elems {
			s := rt.swRender(e, true, depth+1)
			if i < len(labels) && labels[i] != "" {
				s = labels[i] + ": " + s
			}
			parts = append(parts, s)
		}
		return name + "(" + strings.Join(parts, ", ") + ")", true
	}

	// A Set literal, where the grammars build one.
	if tag, ok := o.props["__set"]; ok && tag == true {
		if el, ok := o.props["elems"].(*jsArray); ok {
			parts := make([]string, len(el.elems))
			for i, e := range el.elems {
				parts[i] = rt.swRender(e, true, depth+1)
			}
			return "[" + strings.Join(parts, ", ") + "]", true
		}
	}

	// The type descriptor itself (`print(P.self)`) is its name.
	if _, ok := o.props["__isclass"]; ok {
		if n, ok := o.props["__name"].(string); ok {
			return n, true
		}
	}
	if _, ok := o.props["__isenum"]; ok {
		if n, ok := o.props["__name"].(string); ok {
			return n, true
		}
	}

	// An instance. A STRUCT prints memberwise - P(x: 1, y: 2) - because Swift
	// synthesizes that reflection-based description for every struct; a CLASS
	// prints its type name (real Swift qualifies it with the module, which this
	// runtime does not have).
	cls, _ := o.props["__class"].(*jsObject)
	if cls == nil {
		return "", false
	}
	name, _ := cls.props["__name"].(string)
	if name == "" {
		return "", false
	}
	if isStruct, ok := cls.props["__struct"]; !ok || isStruct != true {
		return name, true
	}
	flds := swStrArray(cls.props["__fields"])
	parts := make([]string, 0, len(flds))
	for _, f := range flds {
		val, ok := o.props[f]
		if !ok {
			val = jsUndef
		}
		parts = append(parts, f+": "+rt.swRender(val, true, depth+1))
	}
	return name + "(" + strings.Join(parts, ", ") + ")", true
}

// swFindMember walks the __class / __super chain for a callable member, the same
// walk classFind does in swift-interpreter.abnf.
func swFindMember(v interface{}, name string) (interface{}, bool) {
	o, ok := v.(*jsObject)
	if !ok {
		return nil, false
	}
	for cls := o.props["__class"]; cls != nil; {
		clsObj, ok := cls.(*jsObject)
		if !ok {
			break
		}
		if mth, ok := clsObj.props[name]; ok && isCallable(mth) {
			return mth, true
		}
		cls = clsObj.props["__super"]
	}
	return nil, false
}

func swNumProp(o *jsObject, key string) (int, bool) {
	f, ok := o.props[key].(float64)
	if !ok {
		return 0, false
	}
	return int(f), true
}

func swStrArray(v interface{}) []string {
	arr, ok := v.(*jsArray)
	if !ok {
		return nil
	}
	out := make([]string, len(arr.elems))
	for i, e := range arr.elems {
		out[i], _ = e.(string)
	}
	return out
}

// ----------------------------------------------------------------------------
// Equality

// swEqual is Swift's ==. Array, Dictionary and tuple are VALUE types in Swift, so
// they compare element by element; js_seq compared them by identity and answered
// false for every one of them. An enum case compares by name plus associated
// values (which the compiler also encodes in __eqkey; this path agrees with it
// and additionally handles the shapes __eqkey cannot round-trip).
func (rt *jsrt) swEqual(l, r interface{}) bool {
	return rt.swEqualDepth(l, r, 0)
}

func (rt *jsrt) swEqualDepth(l, r interface{}, depth int) bool {
	if depth > 32 {
		return false
	}
	if isNullish(l) || isNullish(r) {
		return isNullish(l) && isNullish(r)
	}
	if la, ok := l.(*jsArray); ok {
		ra, ok := r.(*jsArray)
		if !ok || len(la.elems) != len(ra.elems) {
			return false
		}
		for i := range la.elems {
			if !rt.swEqualDepth(la.elems[i], ra.elems[i], depth+1) {
				return false
			}
		}
		return true
	}
	lo, lok := l.(*jsObject)
	ro, rok := r.(*jsObject)
	if lok && rok {
		if lo == ro {
			return true
		}
		if lk, lv, ok := dictParts(lo); ok {
			rk, rv, ok := dictParts(ro)
			if !ok || len(lk.elems) != len(rk.elems) {
				return false
			}
			// A Dictionary is unordered, so every entry of the left must have a
			// matching entry somewhere in the right.
			for i, k := range lk.elems {
				found := false
				for j, k2 := range rk.elems {
					if rt.swEqualDepth(k, k2, depth+1) && rt.swEqualDepth(lv.elems[i], rv.elems[j], depth+1) {
						found = true
						break
					}
				}
				if !found {
					return false
				}
			}
			return true
		}
		if ln, ok := swNumProp(lo, "__tuple"); ok {
			rn, ok := swNumProp(ro, "__tuple")
			if !ok || ln != rn {
				return false
			}
			for i := 0; i < ln; i++ {
				if !rt.swEqualDepth(lo.props[itoa(i)], ro.props[itoa(i)], depth+1) {
					return false
				}
			}
			return true
		}
		if tag, ok := lo.props["__ecase"]; ok && tag == true {
			if tag2, ok := ro.props["__ecase"]; !ok || tag2 != true {
				return false
			}
			if lo.props["__ename"] != ro.props["__ename"] {
				return false
			}
			lv, _ := lo.props["__vals"].(*jsArray)
			rv, _ := ro.props["__vals"].(*jsArray)
			if lv == nil || rv == nil {
				return lv == nil && rv == nil
			}
			if len(lv.elems) != len(rv.elems) {
				return false
			}
			for i := range lv.elems {
				if !rt.swEqualDepth(lv.elems[i], rv.elems[i], depth+1) {
					return false
				}
			}
			return true
		}
		return false
	}
	return rt.strictEq(l, r)
}
