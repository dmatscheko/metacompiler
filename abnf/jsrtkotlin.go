package abnf

// jsrtkotlin.go -- Kotlin's VALUE RENDERING for the compiler grammar
// (languages/kotlin-to-llvm-ir.abnf).
//
// kotlin-to-llvm-ir.abnf used to bind Kotlin's println/print straight to the
// shared `println` host function, which hands the RAW runtime value to Go's %v.
// So the compiler printed
//
//	<nil>                      for a null reference
//	[1 2]                      for a list or an array
//	map[__class:map[__ctor:0x… __isclass:true __name:D …] a:1 b:q]
//	                           for a class instance - including a raw Go POINTER
//	                           address, so its stdout was not even reproducible
//	                           between two runs of the same program
//
// and an enum entry (whose descriptor points back at the entry) sent Go's %v into
// an infinite recursion: `println(E.A)` died with a stack overflow.
//
// Kotlin/JVM's Int/Long/Double/String ARE java.lang.*, and println IS
// System.out.println, so the answers below were settled with java 24 (there is no
// kotlinc on this machine - see the commit message for the probe and its output):
//
//	null / [1, 2] / D(a=1, b=q) / C@<hash> / A (an enum entry) / 1.5
//
// ktpRender is the Go twin of `kstr` in kotlin-interpreter.abnf and the two MUST
// stay in step, because ./test.sh --cross diffs the two halves.
//
// Registration is additive: an init() appends a registrar to rxExtraExterns, so
// nothing here runs unless a program calls one of the js_kt* externs, and only the
// Kotlin compiler grammar emits them. abnf/jsrt.go, its extern table and printArgs
// are untouched, so every other language keeps the rendering it has.
//
// Known simplifications, all DELIBERATE and all shared by the two halves:
//
//   - An ARRAY renders exactly like a List, `[1, 2]`. Real Kotlin prints
//     `[Ljava.lang.Integer;@1b6d3586` for an Array, because an Array is a JVM array
//     and inherits Object.toString. In this value model arrayOf and listOf both
//     build a plain host array and are indistinguishable at runtime, so no renderer
//     can tell them apart; the List answer is the one that is right more often.
//   - A class without a toString renders as `Name@1`: a FIXED synthetic hash. Real
//     Kotlin prints the identity hash, which differs between two runs of the real
//     toolchain and therefore cannot be asserted anyway; a constant is the only
//     value the two halves can agree on without an object-identity counter.
//     kotlin-interpreter.abnf's defaultToString has always answered `Name@1`.
//   - A RANGE prints as the list of its elements (`[1, 2, 3]`), where real Kotlin
//     prints `1..3`. The compiler has no first-class range value at all - `1..3`
//     is materialised into a list at the point of use - so `1..3` is not
//     reachable there, and the interpreter was made to agree with the compiler
//     rather than the other way round.
//   - There is no Map in either Kotlin half (mapOf is not a builtin in either
//     grammar: both answer "unknown name: mapOf"), so java.util.AbstractMap's
//     `{a=1, b=2}` rendering has nothing to render and is not implemented.

import (
	"fmt"
	"strings"
)

func init() {
	rxExtraExterns = append(rxExtraExterns, func(rt *jsrt, m map[string]func(args []uint64) uint64) {
		u := rt.unwrap

		// Kotlin's println / print, as a VALUE: the extern takes no arguments and
		// returns the host function, which buildMain declares into the program
		// scope under the names `println` and `print`. Binding a value rather than
		// intercepting the call site keeps `::println` and `list.forEach(::println)`
		// working, and keeps the shared println host function untouched for the
		// twelve other languages that use it.
		m["js_ktprint"] = func(a []uint64) uint64 {
			return rt.wrap(jsHostFunc("println", func(rt *jsrt, this uint64, args []interface{}) interface{} {
				fmt.Fprint(outWriter, wtf8Clean(ktpLine(rt, args)+"\n"))
				return jsUndef
			}))
		}
		m["js_ktwrite"] = func(a []uint64) uint64 {
			return rt.wrap(jsHostFunc("print", func(rt *jsrt, this uint64, args []interface{}) interface{} {
				fmt.Fprint(outWriter, wtf8Clean(ktpLine(rt, args)))
				return jsUndef
			}))
		}

		// The text of a value: what println writes, what "$v" interpolates and what
		// v.toString() answers.
		m["js_ktstr"] = func(a []uint64) uint64 { return rt.wrapStr(rt.ktpRender(u(a[0]), 0)) }
	})
}

// ktpLine is the text one println/print call writes. Kotlin's println has only
// single-argument overloads plus the no-argument one (which writes just the line
// terminator), so this is either "" or the one rendered argument.
func ktpLine(rt *jsrt, args []interface{}) string {
	out := ""
	for _, a := range args {
		out += rt.ktpRender(a, 0)
	}
	return out
}

// ktpRender is Kotlin's toString for a runtime value. `depth` only guards a cyclic
// object graph - an enum entry's class descriptor holds the entry, which holds the
// descriptor - so the renderer cannot recurse forever the way Go's %v did.
func (rt *jsrt) ktpRender(v interface{}, depth int) string {
	if depth > 32 {
		return "..."
	}
	switch t := v.(type) {
	case jsUndefT:
		return "kotlin.Unit" // Unit is a VALUE in Kotlin, and it prints as its name.
	case jsNullT:
		return "null"
	case jsChar:
		return string(rune(t.code))
	case jsJFlo:
		return jvmFloText(t) // 1.5 / 1.0E20 / Infinity - see jsrtjvm.go.
	case string:
		// Neither the top level nor a collection element quotes a String:
		// java.util.AbstractCollection.toString calls String.valueOf, so
		// `println(listOf("x", "y"))` is `[x, y]`.
		return t
	case *jsArray:
		parts := make([]string, len(t.elems))
		for i, e := range t.elems {
			parts[i] = rt.ktpRender(e, depth+1)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case *jsObject:
		if s, ok := rt.ktpObj(t, depth); ok {
			return s
		}
	}
	return rt.toString(v)
}

// ktpObj renders the object shapes the Kotlin compiler grammar builds. It reports
// false for a shape it does not know, which falls back to the generic ToString.
func (rt *jsrt) ktpObj(o *jsObject, depth int) (string, bool) {
	// A Regex renders as its pattern and a MatchResult as the matched text, which is
	// what their toString answers (see abnf/jsrtregexkt.go); neither carries a
	// __class, so without this they fell through to "[object Object]".
	if ktRxIsRegex(o) || ktRxIsMatch(o) {
		if v, handled := rt.ktRxMethod(o, "toString", nil); handled {
			return rt.ktpRender(v, depth+1), true
		}
	}
	cls, _ := o.props["__class"].(*jsObject)
	if cls == nil {
		// A class descriptor printed on its own (`println(E)`), which Go's %v
		// dumped as a map of pointers.
		if n, ok := ktpName(o); ok {
			return n, true
		}
		return "", false
	}
	// A user-written or synthesized toString() wins over every built-in rendering,
	// exactly as it does in Kotlin: a data class renders through the toString the
	// compiler generated for it (D(a=1, b=q)), an `override fun toString()` through
	// its own body. The walk is the __super chain memberCall follows.
	if mth, ok := ktpFindMember(o, "toString"); ok {
		return rt.ktpRender(rt.call(mth, jsUndef, []interface{}{o}), depth+1), true
	}
	// An enum entry prints its NAME (java.lang.Enum.toString); both halves store it
	// on the entry as `name`.
	if ktpIsEnum(cls) {
		if n, ok := o.props["name"].(string); ok {
			return n, true
		}
	}
	name, _ := cls.props["__name"].(string)
	if name == "" {
		return "", false
	}
	return name + "@1", true
}

// ktpName is the __name of a class descriptor.
func ktpName(o *jsObject) (string, bool) {
	if _, ok := o.props["__isclass"]; !ok {
		return "", false
	}
	n, ok := o.props["__name"].(string)
	return n, ok && n != ""
}

// ktpIsEnum reports whether a descriptor (or one of its supers - an enum entry
// with a body is an anonymous subclass) belongs to an enum class.
func ktpIsEnum(cls *jsObject) bool {
	for i := 0; cls != nil && i < 32; i++ {
		if e, ok := cls.props["__isenum"]; ok && e == true {
			return true
		}
		cls, _ = cls.props["__super"].(*jsObject)
	}
	return false
}

// ktpFindMember walks the __class / __super chain for a callable member, the same
// walk memberCall does for a js_mcall and clsFind does in kotlin-interpreter.abnf.
func ktpFindMember(o *jsObject, name string) (interface{}, bool) {
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
