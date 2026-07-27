package abnf

// jsrtjava.go -- Java's VALUE RENDERING for the compiler grammar
// (languages/java-to-llvm-ir.abnf).
//
// java-to-llvm-ir.abnf used to bind System.out.println straight to the shared
// `println` host function, which hands the raw runtime value to Go's `%v`. That
// printed `<nil>` for a null reference, `[1 2 3]` for an array and a full
// `map[__class:map[__isclass:true __name:Box ...] x:0]` dump - containing raw Go
// POINTER addresses, so the compiler's stdout was not even reproducible between
// two runs - for an instance. Printing an ENUM CONSTANT crashed the process
// outright with a Go stack overflow, because the constant's `__class` points at a
// descriptor that points back at the constant and `toGoNatural` has no cycle
// guard. Real java 24 prints
//
//	null / [I@1dbd16a6 / Main$Box@7344699f / RED / P[x=1, y=2]
//
// The interpreter half was wrong in its own way: its `jstr` fell through to
// JavaScript's ToString, so an array printed "1,2,3", an instance "Box@obj", a
// record "P@obj" and a `double[]` "[object Object]". Both halves now render
// through the same rules; this file is the Go twin of `jstr` / `jIdHash` /
// `jArrayTag` in java-interpreter.abnf and the two MUST stay in step, because
// ./test.sh --cross diffs them.
//
// Registration is additive: an init() appends a registrar to rxExtraExterns, so
// nothing here runs unless a program calls one of the js_jv* externs, and only the
// Java compiler grammar emits them. No existing extern is renamed, removed or
// rebound, and abnf/jsrt.go's printArgs is untouched.
//
// Known simplifications, all DELIBERATE and all shared by the two halves:
//
//   - The identity HASH after the `@` is a per-run counter, not an address: the
//     k-th distinct object rendered in a run gets 0x1a2b3c00+k, printed with %x.
//     Real java prints System.identityHashCode, which differs between two runs of
//     the SAME program, so no assertion can ever depend on the digits; what the two
//     halves DO have to agree on is that the same object renders the same way twice
//     and that a program's output is reproducible, which a counter gives.
//   - An array's ELEMENT TYPE is inferred from the elements, because this value
//     model does not carry the declared type. plain number -> [I, boxed double ->
//     [D, String -> [Ljava.lang.String;, boolean -> [Z, char -> [C, and empty or
//     mixed -> [Ljava.lang.Object;. So `Object[] o = {1, 2}` renders as `[I@...`
//     where real java says `[Ljava.lang.Object;@...`, and a nested `int[][]` renders
//     as `[Ljava.lang.Object;@...` where real java says `[[I@...`.
//   - `Object#toString` is not a callable MEMBER: `b.toString()` on a class that
//     declares none still fails to dispatch. Only the implicit rendering (println,
//     `+`, String.valueOf) goes through here.

import (
	"fmt"
	"strings"
)

func init() {
	rxExtraExterns = append(rxExtraExterns, func(rt *jsrt, m map[string]func(args []uint64) uint64) {
		u := rt.unwrap

		// System.out.println(x) / System.out.print(x). The argument ARRAY of the
		// emitted closure arrives whole, so the no-argument println(), which
		// writes just the line terminator, is distinguishable from println(null).
		m["js_jvprint"] = func(a []uint64) uint64 {
			fmt.Fprintln(outWriter, wtf8Clean(jvpFirst(rt, u(a[0]))))
			return 0
		}
		m["js_jvwrite"] = func(a []uint64) uint64 {
			fmt.Fprint(outWriter, wtf8Clean(jvpFirst(rt, u(a[0]))))
			return 0
		}
		// String.valueOf(x) and the string form used by `+`.
		m["js_jvstr"] = func(a []uint64) uint64 { return rt.wrapStr(rt.jvpStr(u(a[0]))) }
		// Java's `+`. Identical to js_jadd (string concat / 32 bit int add / float
		// add) except that the string case renders through jvpStr, so `"" + arr`
		// and `"" + obj` agree with println instead of falling through to
		// JavaScript's ToString ("1,2,3", "[object Object]").
		m["js_jvadd"] = func(a []uint64) uint64 {
			l, r := u(a[0]), u(a[1])
			_, ls := l.(string)
			_, rs := r.(string)
			if ls || rs {
				return rt.wrapStr(rt.jvpStr(l) + rt.jvpStr(r))
			}
			if jvmIsFlo(l) || jvmIsFlo(r) {
				return rt.wrap(jsJFlo{f: rt.toNumber(l) + rt.toNumber(r), sty: jvmStyleOf(l, r)})
			}
			ln, rn := rt.toNumber(l), rt.toNumber(r)
			if isInt32Value(ln) && isInt32Value(rn) {
				return rt.wrapNum(float64(int32(int64(ln) + int64(rn))))
			}
			return rt.wrapNum(ln + rn)
		}
	})
}

// jvpFirst renders the first element of a print call's argument array. An empty
// array is the no-argument System.out.println(), which prints nothing but the
// terminator.
func jvpFirst(rt *jsrt, args interface{}) string {
	arr, ok := args.(*jsArray)
	if !ok {
		return rt.jvpStr(args)
	}
	if len(arr.elems) == 0 {
		return ""
	}
	return rt.jvpStr(arr.elems[0])
}

// ----------------------------------------------------------------------------
// Rendering

// jvpStr is String.valueOf: what println writes and what `+` concatenates.
func (rt *jsrt) jvpStr(v interface{}) string { return rt.jvpRender(v, 0) }

func (rt *jsrt) jvpRender(v interface{}, depth int) string {
	if depth > 32 {
		return "..."
	}
	switch t := v.(type) {
	case jsNullT, jsUndefT:
		return "null"
	case string:
		return t
	case *jsArray:
		return jvpArrayTag(t) + "@" + jvpHash(t)
	case *jsObject:
		return rt.jvpObj(t, depth)
	}
	return rt.toString(v)
}

// jvpObj renders the object shapes the two Java grammars build.
func (rt *jsrt) jvpObj(o *jsObject, depth int) string {
	// A user toString() wins over every built-in rendering, as it does in Java.
	if f, ok := jvpFindMember(o, "toString"); ok {
		return rt.jvpRender(rt.call(f, o, []interface{}{o}), depth+1)
	}
	// The class descriptor itself (println(Box.class) has no spelling here, but
	// the interpreter half renders one this way, so the two agree).
	if tag, ok := o.props["__isclass"]; ok && tag == true {
		if n, ok := o.props["__name"].(string); ok {
			return "class " + n
		}
	}
	cls, _ := o.props["__class"].(*jsObject)
	if cls == nil {
		return rt.toString(o)
	}
	// An enum constant without an override prints its constant name (the
	// java.lang.Enum#toString contract).
	if n, ok := o.props["__ename"].(string); ok {
		return n
	}
	// A record prints its canonical form, P[x=1, y=2] (the java.lang.Record
	// contract). __record is the component name list the class carries. The name
	// here is the SIMPLE one - the generated toString uses getSimpleName, so a
	// nested record is `P[...]` and not `Main$P[...]`.
	if comps, ok := cls.props["__record"].(*jsArray); ok {
		parts := make([]string, 0, len(comps.elems))
		for _, c := range comps.elems {
			name, _ := c.(string)
			parts = append(parts, name+"="+rt.jvpRender(o.props[name], depth+1))
		}
		simple, _ := cls.props["__name"].(string)
		return simple + "[" + strings.Join(parts, ", ") + "]"
	}
	return jvpClassName(cls) + "@" + jvpHash(o)
}

// jvpClassName is the class' BINARY name: a nested type is spelled Outer$Inner,
// the way java.lang.Class#getName does, and the grammars record that as __bname
// while they walk the declaration. A class that carries none (an anonymous or
// locally built descriptor) falls back to its simple name.
func jvpClassName(cls *jsObject) string {
	if b, ok := cls.props["__bname"].(string); ok && b != "" {
		return b
	}
	if n, ok := cls.props["__name"].(string); ok {
		return n
	}
	return "Object"
}

// jvpFindMember walks the __class / __super chain for a callable member, the same
// walk the two grammars' method dispatch does.
func jvpFindMember(o *jsObject, name string) (interface{}, bool) {
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

// jvpArrayTag is the array's class name as getName spells it. The element type is
// INFERRED from the elements (see the header): this value model does not carry the
// declared one.
func jvpArrayTag(a *jsArray) string {
	tag := ""
	for _, e := range a.elems {
		t := jvpElemTag(e)
		if t == "" || (tag != "" && t != tag) {
			return "[Ljava.lang.Object;"
		}
		tag = t
	}
	if tag == "" {
		return "[Ljava.lang.Object;"
	}
	return tag
}

func jvpElemTag(v interface{}) string {
	switch v.(type) {
	case jsJFlo:
		return "[D"
	case jsChar:
		return "[C"
	case bool:
		return "[Z"
	case string:
		return "[Ljava.lang.String;"
	case float64:
		return "[I"
	}
	return ""
}

// ----------------------------------------------------------------------------
// Identity hash

// jvpIdent gives every object rendered with an `@` a stable per-run number. Real
// java prints System.identityHashCode, which is an address-derived value that
// changes between two runs of the same program - so it is not reproducible and no
// test can assert it. What the two halves must agree on is that the same object
// renders identically twice within a run and that the run as a whole is
// deterministic, which the counter provides. One compile runs one program, so a
// package-level counter is per-program.
var jvpIdent = map[interface{}]int{}
var jvpIdentN int

func jvpHash(v interface{}) string {
	n, ok := jvpIdent[v]
	if !ok {
		jvpIdentN++
		n = jvpIdentN
		jvpIdent[v] = n
	}
	return fmt.Sprintf("%x", 0x1a2b3c00+n)
}
