package abnf

// jsrtcsharp.go -- C#'s VALUE RENDERING for the compiler grammar
// (languages/csharp-to-llvm-ir.abnf).
//
// csharp-to-llvm-ir.abnf used to bind Console.WriteLine / Console.Write straight
// to the shared `println` / `print` host functions, which hand the raw runtime
// value to Go's `%v`. That printed `<nil>` for a null reference, `[1 2 3]` for an
// array and a full `map[__class:map[__isclass:true __name:Box ...] x:0]` dump -
// including a raw Go POINTER address for the constructor - for a class instance,
// so the compiler's stdout was not even reproducible between two runs of the same
// program. The interpreter half was wrong in its own way: its `csStr` fell through
// to JavaScript's ToString, so an array printed `1,2,3` and an instance
// `[object Object]`.
//
// On top of that both halves agreed on ONE wrong answer, which is why
// `./test.sh --cross` could never flag it: a bool converted to a string printed
// JavaScript's `true` / `false`. `System.Boolean.ToString()` returns
// `Boolean.TrueString` / `Boolean.FalseString`, which are the strings "True" and
// "False" (the LITERALS `true` / `false` are unchanged - only the string
// CONVERSION is capitalised).
//
// This file is the Go twin of `csStr` in csharp-interpreter.abnf and the two MUST
// stay in step, because ./test.sh --cross diffs them.
//
// Registration is additive: an init() appends a registrar to rxExtraExterns, so
// nothing here runs unless a program calls one of the js_cs* externs registered
// below, and only the C# compiler grammar emits them. No existing extern is
// renamed, removed or rebound, and abnf/jsrt.go is untouched.
//
// Spec basis. There is no dotnet on this machine, so every answer below is CITED
// rather than executed:
//
//   - bool     "True" / "False"        System.Boolean.ToString(), which is
//                                      documented to return Boolean.TrueString /
//                                      Boolean.FalseString ("True" / "False").
//                                      A .NET BCL library contract, not an
//                                      ECMA-334 language rule.
//   - null     ""                      ECMA-334 12.4.7 (string concatenation:
//                                      a null operand contributes the empty
//                                      string) and System.Console.WriteLine(object),
//                                      which writes only a line terminator when the
//                                      argument is null. Also String.Concat, which
//                                      calls ToString() on non-null arguments only.
//   - array    "System.Int32[]"        System.Object.ToString() returns the fully
//                                      qualified type name (ECMA-334 8.2.3 lists
//                                      ToString among the members every type
//                                      inherits from object); an array type's name
//                                      is the element type's name plus "[]".
//   - instance "Box"                   System.Object.ToString() returns
//                                      GetType().ToString(), the fully qualified
//                                      name. See the simplification below.
//   - double   csFloStr                unchanged from the commit that boxed the
//                                      float (see jvmFloText / floCS in
//                                      abnf/jsrtjvm.go); ECMA-335 "G" round-trip.
//   - char     the glyph               System.Char.ToString().
//
// A user-defined ToString() override wins over ALL of the above, in both halves -
// System.Object.ToString() is virtual and every conversion site (Console.WriteLine,
// String.Concat, an interpolation hole) calls the virtual method.
//
// Known simplifications, all DELIBERATE and all shared by the two halves:
//
//   - An ARRAY's element type is INFERRED from its elements, because the declared
//     element type is not carried by this value model. The rule, in both halves:
//     plain number -> System.Int32[], float box -> System.Double[], string ->
//     System.String[], bool -> System.Boolean[], char -> System.Char[], and an
//     empty or mixed array -> System.Object[]. Real C# would answer from the
//     DECLARED type, so `object[] o = new object[]{1}` prints System.Int32[] here
//     and System.Object[] there.
//   - A List<T> is the same plain array in this runtime, so it renders like an
//     array. Real C# prints "System.Collections.Generic.List`1[System.Int32]".
//   - An INSTANCE prints its BARE declared name. Real C# prints the fully
//     qualified name, which for a nested class in a namespace would be
//     `Demo.Program+Box`; this runtime has no namespace and no nesting concept at
//     all, so the bare name is the reproducible part of that answer. (The same
//     simplification abnf/jsrtswift.go records for Swift's module qualification.)
//   - A RECORD prints its type name like any other instance. Real C# synthesizes
//     a `Box { X = 1 }` ToString for a record.
//   - A Dictionary and a tuple are left exactly as they rendered before (the
//     generic "[object Object]"), because both halves already agreed there and
//     neither shape carries the type information the real answer needs.

import (
	"fmt"
)

func init() {
	rxExtraExterns = append(rxExtraExterns, func(rt *jsrt, m map[string]func(args []uint64) uint64) {
		u := rt.unwrap

		// The whole Console object, built here rather than in the grammar so that
		// WriteLine / Write render through cspStr instead of through the shared
		// println host function (which is what leaked Go's %v).
		m["js_csconsole"] = func(a []uint64) uint64 {
			o := newJSObject()
			o.set("WriteLine", jsHostFunc("js_csprint", func(rt *jsrt, this uint64, args []interface{}) interface{} {
				fmt.Fprint(outWriter, wtf8Clean(rt.cspStr(argAt(args, 0)))+"\n")
				return jsUndef
			}))
			o.set("Write", jsHostFunc("js_cswrite", func(rt *jsrt, this uint64, args []interface{}) interface{} {
				fmt.Fprint(outWriter, wtf8Clean(rt.cspStr(argAt(args, 0))))
				return jsUndef
			}))
			return rt.wrap(o)
		}

		// The rendering step of a string CONVERSION, used by '+', by an
		// interpolation hole and by .ToString(). It converts only the values whose
		// C# string form differs from the shared runtime's - a bool, an array and a
		// class instance - and passes everything else through UNCHANGED, so that
		// `1 + 2` stays an integer addition and a boxed double stays a double. The
		// caller wraps the result in js_csadd, which supplies C#'s null rule and
		// the numeric cases.
		m["js_csstr"] = func(a []uint64) uint64 {
			v := u(a[0])
			switch t := v.(type) {
			case bool:
				return rt.wrapStr(cspBoolStr(t))
			case *jsArray:
				return rt.wrapStr(cspArrayName(t))
			case *jsObject:
				if s, ok := rt.cspObjStr(t, 0); ok {
					return rt.wrapStr(s)
				}
			}
			return a[0]
		}
	})
}

// cspBoolStr is System.Boolean.ToString(): Boolean.TrueString / Boolean.FalseString.
func cspBoolStr(b bool) string {
	if b {
		return "True"
	}
	return "False"
}

// cspStr is the full C# string conversion - the twin of csStr in
// csharp-interpreter.abnf.
func (rt *jsrt) cspStr(v interface{}) string { return rt.cspRender(v, 0) }

func (rt *jsrt) cspRender(v interface{}, depth int) string {
	if depth > 32 {
		return "..."
	}
	switch t := v.(type) {
	case jsNullT, jsUndefT:
		return "" // Console.WriteLine(null) writes an empty line.
	case bool:
		return cspBoolStr(t)
	case string:
		return t
	case *jsArray:
		return cspArrayName(t)
	case *jsObject:
		if s, ok := rt.cspObjStr(t, depth); ok {
			return s
		}
	}
	return rt.toString(v) // numbers, char, the boxed double (csFloStr), everything else.
}

// cspObjStr renders the object shapes the two C# grammars build. It reports false
// for a shape it does not know, which keeps the generic rendering both halves
// already agreed on.
func (rt *jsrt) cspObjStr(o *jsObject, depth int) (string, bool) {
	// A user ToString() override wins over every built-in rendering, exactly as
	// the virtual System.Object.ToString() does in C#.
	if mth, ok := cspFindToString(o); ok {
		return rt.cspRender(rt.call(mth, jsUndef, []interface{}{o}), depth+1), true
	}
	// The type descriptor itself.
	if _, ok := o.props["__isclass"]; ok {
		if n, ok := o.props["__name"].(string); ok {
			return n, true
		}
	}
	// An instance: its type's name (see the simplification note in the header).
	if cls, ok := o.props["__class"].(*jsObject); ok {
		if n, ok := cls.props["__name"].(string); ok && n != "" {
			return n, true
		}
	}
	return "", false
}

// cspFindToString walks the __class / __super chain for a user-declared ToString,
// the same walk the method dispatch does in both C# grammars.
func cspFindToString(o *jsObject) (interface{}, bool) {
	for cls := o.props["__class"]; cls != nil; {
		clsObj, ok := cls.(*jsObject)
		if !ok {
			return nil, false
		}
		if mth, ok := clsObj.props["ToString"]; ok && isCallable(mth) {
			return mth, true
		}
		cls = clsObj.props["__super"]
	}
	return nil, false
}

// cspArrayName is Object.ToString() of an array: the element type's full name plus
// "[]". The DECLARED element type is not available in this value model, so it is
// inferred from the elements - see the header for the exact rule.
func cspArrayName(a *jsArray) string {
	if len(a.elems) == 0 {
		return "System.Object[]"
	}
	name := ""
	for _, e := range a.elems {
		n := cspElemName(e)
		if n == "" || (name != "" && n != name) {
			return "System.Object[]"
		}
		name = n
	}
	return name + "[]"
}

func cspElemName(e interface{}) string {
	switch e.(type) {
	case jsJFlo:
		return "System.Double"
	case string:
		return "System.String"
	case bool:
		return "System.Boolean"
	case jsChar:
		return "System.Char"
	case float64:
		return "System.Int32"
	}
	return ""
}
