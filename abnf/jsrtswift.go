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
	"math"
	"os"
	"strconv"
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

		// ----- Parameter packs (SE-0393) -----
		// A `repeat E` expansion produces a TUPLE of the per-element results, and
		// `for x in repeat each p` is the one place a tuple is walked as a
		// sequence. Both directions live here rather than in emitted IR because
		// the element count is a runtime property of the pack, and because
		// swift-interpreter.abnf's newTupleVal / tupleArray must agree with them
		// exactly - ./test.sh --cross diffs the two halves.
		m["js_swtuple"] = func(a []uint64) uint64 {
			arr, ok := u(a[0]).(*jsArray)
			if !ok {
				rt.fail("js_swtuple needs an array")
			}
			o := newJSObject()
			o.set("__tuple", float64(len(arr.elems)))
			for i, e := range arr.elems {
				o.set(itoa(i), e)
			}
			return rt.wrap(o)
		}
		m["js_swtuparr"] = func(a []uint64) uint64 {
			o, ok := u(a[0]).(*jsObject)
			if !ok {
				return a[0]
			}
			n, ok := swNumProp(o, "__tuple")
			if !ok {
				return a[0]
			}
			out := &jsArray{elems: make([]interface{}, 0, n)}
			for i := 0; i < n; i++ {
				out.elems = append(out.elems, o.props[itoa(i)])
			}
			return rt.wrap(out)
		}

		// ----- Swift's arithmetic (see the "Integers" note below) -----
		w := rt.wrap
		// One binary arithmetic or bitwise operator: (op, left, right). Two
		// INTEGRAL operands go to the sized-integer layer of jsrtint.go, which
		// evaluates at the result type's width and wraps to it; an operand that
		// is a FLOAT BOX pulls the whole operation into floating point and the
		// result is a box again (see the "Doubles" note below).
		// swift-interpreter.abnf implements the identical rule with szArith and
		// its {__flo} box, and ./test.sh --cross diffs the two.
		m["js_swarith"] = func(a []uint64) uint64 {
			op, l, r := rt.toString(u(a[0])), u(a[1]), u(a[2])
			if swIsWhole(l) && swIsWhole(r) {
				// AN INTEGER DIVISION BY ZERO IS SWIFT'S, NOT GO'S
				// (docs/todo.md 1.1). `integer divide by zero` is the
				// text of GO'S OWN runtime panic and reaches this
				// through the SHARED sized-integer layer
				// (abnf/jsrtint.go, languages/lib/runtime.c's
				// si_arith) - correct for go and for nobody else.
				// swiftc 6.1.2 says
				//   Swift/IntegerTypes.swift:9012: Fatal error: Division by zero
				// so the guard sits in FRONT of the shared
				// primitive, in swift's own file, and the floor's
				// Go-correct wording is left exactly as it is.
				// Twins: swGiArith in languages/lib/swift-rt.metajs
				// and swDivGuard in languages/swift-interpreter.abnf.
				if (op == "/" || op == "%") && swNum(rt, r) == 0 {
					rt.swAbort("Fatal error: Division by zero")
				}
				return w(rt.giArith(op, l, r))
			}
			if op == "+" {
				_, ls := l.(string)
				_, rs := r.(string)
				if ls || rs {
					return rt.wrapStr(strConcat(rt.swDesc(l), rt.swDesc(r)))
				}
				// Array + Array is CONCATENATION in Swift, where the shared
				// jsAdd coerced both to text and answered "1,23".
				if la, ok := l.(*jsArray); ok {
					if ra, ok2 := r.(*jsArray); ok2 {
						out := make([]interface{}, 0, len(la.elems)+len(ra.elems))
						out = append(out, la.elems...)
						out = append(out, ra.elems...)
						return w(&jsArray{elems: out})
					}
				}
			}
			x, y := swNum(rt, l), swNum(rt, r)
			// A Float operand makes BOTH operands Floats and rounds the
			// result: swift forbids every mixed pair, so the only other
			// side an untyped literal can be is a Float too. Computing in
			// double and rounding once is exactly binary32 + - * / % -
			// double has more than 2*24+2 significand bits, so the second
			// rounding cannot move the result (Figueroa's theorem).
			f32 := swIsF32(l) || swIsF32(r)
			if f32 {
				x, y = jvmFround(x), jvmFround(y)
			}
			switch op {
			case "+":
				return w(swReFlo(x+y, f32))
			case "-":
				return w(swReFlo(x-y, f32))
			case "*":
				return w(swReFlo(x*y, f32))
			case "/":
				return w(swReFlo(x/y, f32))
			case "%":
				return w(swReFlo(math.Mod(x, y), f32))
			}
			return w(rt.giArith(op, l, r))
		}
		// ----- Swift's Double: the boxed float (see the "Doubles" note below) -----
		// A float literal, Double(x) and Float(x) all arrive here. A String
		// source is FAILABLE, like Int("x").
		m["js_swflo"] = func(a []uint64) uint64 {
			v := u(a[0])
			if s, ok := v.(string); ok {
				if !swFloText(s) {
					return jsHNull
				}
				f, err := strconv.ParseFloat(s, 64)
				if err != nil {
					return jsHNull
				}
				return w(swMkFlo(f))
			}
			if isNullish(v) {
				rt.fail("Double conversion of nil")
			}
			if b, ok := v.(bool); ok {
				if b {
					return w(swMkFlo(1))
				}
				return w(swMkFlo(0))
			}
			return w(swMkFlo(swNum(rt, v)))
		}
		// Float(x): the same conversion at the 32 bit width, and the one new
		// extern the width needs. Every Float in a program comes from here or
		// from js_swadoptf32 (a `let x: Float` annotation); a Float that is
		// already one keeps its width through every operator above. The layer-2
		// twin is js_swflo32 in languages/lib/swift-rt.metajs.
		m["js_swflo32"] = func(a []uint64) uint64 {
			v := u(a[0])
			if s, ok := v.(string); ok {
				if !swFloText(s) {
					return jsHNull
				}
				// Read at 64 and rounded to 32, where swiftc rounds once. A
				// decimal that lands exactly on a float midpoint AFTER a double
				// rounding would differ; no value in the corpus or the probe
				// does. It is spelled this way because layer 2 and the
				// interpreter have only a double parser, and the three engines
				// must answer the same string.
				f, err := strconv.ParseFloat(s, 64)
				if err != nil {
					return jsHNull
				}
				return w(swMkF32(f))
			}
			if isNullish(v) {
				rt.fail("Float conversion of nil")
			}
			if b, ok := v.(bool); ok {
				if b {
					return w(swMkF32(1))
				}
				return w(swMkF32(0))
			}
			return w(swMkF32(swNum(rt, v)))
		}
		// `let x: Float = 1` and `var x: Float` - the declared-type adoption, the
		// float half of the rule that makes `let d: Double = 3` hold 3.0. It has
		// to pass nil through, which js_swflo32 would abort on.
		m["js_swadoptf32"] = func(a []uint64) uint64 {
			if isNullish(u(a[0])) {
				return a[0]
			}
			return m["js_swflo32"](a)
		}
		// ----- Declared-type adoption (docs/todo.md 1.1) -----
		//
		// An untyped literal takes the FLOATNESS and the WIDTH of the type it
		// is written at. `let d: Double = 3` has always done this (the
		// js_swflo/js_swadoptf32/js_swintconv calls makeDecl emits); js_swadopt
		// is what the OTHER declared-type sites emit - a parameter, a return
		// type, a stored property, and the element type of an array annotation.
		//
		// It is not a Float story: an Int literal that does not adopt stays an
		// Int and INTEGER-DIVIDES, so `func g(_ x: Double) -> Double { x / 2 }`
		// called g(3) answered 1 where swiftc 6.1.2 says 1.5.
		//
		// A value that is neither a bare number nor a sized integer nor a float
		// box is returned UNCHANGED: at these sites the only value a
		// type-correct Swift program can present is numeric, and passing
		// everything else through keeps the adoption from inventing conversions
		// (a String is NOT parsed here, where the failable `let x: Int = "5"`
		// site is a deliberate conversion). The twin of swAdoptTy in
		// languages/swift-interpreter.abnf and of js_swadopt in
		// languages/lib/swift-rt.metajs.
		swAdopt := func(v interface{}, ty string) interface{} {
			if isNullish(v) {
				return v
			}
			_, isNum := v.(float64)
			if !isNum {
				_, isNum = v.(jsGInt)
			}
			switch ty {
			case "Double":
				if swIsFlo(v) {
					return v
				}
				if isNum {
					return swMkFlo(swNum(rt, v))
				}
				return v
			case "Float":
				if swIsF32(v) {
					return v
				}
				if f, ok := v.(jsJFlo); ok {
					return swMkF32(f.f)
				}
				if isNum {
					return swMkF32(swNum(rt, v))
				}
				return v
			}
			bits, uns, ok := swAdoptWidth(ty)
			if !ok || swIsFlo(v) || !isNum {
				return v
			}
			return rt.giConv(v, bits, uns)
		}
		m["js_swadopt"] = func(a []uint64) uint64 {
			ty, _ := u(a[1]).(string)
			return w(swAdopt(u(a[0]), ty))
		}
		// The WRITE half of the same rule (docs/todo.md 1.1). `var d: Double = 0;
		// d = 3` answered 1, because only the INITIAL value ever adopted: the
		// declaration knows the annotation and the assignment three lines later
		// does not.
		//
		// No var-type table is needed, because this value model already carries
		// the two things an annotation decides - FLOATNESS (the jsJFlo box, at
		// its own width) and integer WIDTH (jsGInt) - on the value itself, and
		// Swift is statically typed, so the slot's type is exactly the type of
		// what it already holds. So a write adopts the type of the OLD value.
		// The one shape it cannot see is a slot that holds nil, which is a
		// `var d: Double` never yet assigned; there the write is untouched
		// unless `ty` names the annotation - the LAST RESORT, "" almost
		// everywhere, from a walk-time register in the emitter.
		// Twins: swAdoptLike in languages/swift-interpreter.abnf and
		// js_swadoptlike in languages/lib/swift-rt.metajs.
		swAdoptLike := func(old, v interface{}, ty string) interface{} {
			if isNullish(old) {
				return swAdopt(v, ty)
			}
			if isNullish(v) {
				return v
			}
			if swIsF32(old) {
				return swAdopt(v, "Float")
			}
			if swIsFlo(old) {
				return swAdopt(v, "Double")
			}
			if gi, ok := old.(jsGInt); ok {
				if swIsFlo(v) {
					return v
				}
				_, isNum := v.(float64)
				if !isNum {
					_, isNum = v.(jsGInt)
				}
				if !isNum {
					return v
				}
				return rt.giConv(v, gi.w, gi.u)
			}
			return v
		}
		m["js_swadoptlike"] = func(a []uint64) uint64 {
			ty, _ := u(a[2]).(string)
			return w(swAdoptLike(u(a[0]), u(a[1]), ty))
		}
		// The STRUCTURAL declared type (docs/todo.md 1.3). `let a: [Double] =
		// [3, 4]` types the array's ELEMENTS, so a[0] / 2 is 1.5; so does a
		// dictionary's VALUE type (`[String: Double]`), a tuple's element types
		// (`(Double, Int)`) and an Optional annotation (`Double?`), each of which
		// answered 1 before because only the array form was walked. The walk is
		// over the annotation TEXT, brackets and all, and recurses on the three
		// containers, so `[String: [Double]]` falls out of the same arms. A leaf
		// this does not recognise adopts nothing, so a user type, a generic and a
		// function type all pass through untouched.
		//
		// Twins: swAdoptDeep in languages/swift-interpreter.abnf and
		// js_swadoptdeep in languages/lib/swift-rt.metajs.
		//
		// fmap is the emitter's map from a function-type LEAF's canonical text
		// to a maker closure that answers the adopting wrapper for a value of
		// that type. Layer 2 builds no closures and neither does this walk, so a
		// nested `[(Double) -> Double]` had no way to adopt at all; the emitter
		// knows every such leaf at compile time and hands the makers down. It is
		// nil for every annotation with no function-type leaf, which is nearly
		// all of them.
		var swAdoptDeep func(v interface{}, ty string, fmap *jsObject) interface{}
		swAdoptDeep = func(v interface{}, ty0 string, fmap *jsObject) interface{} {
			if ty0 == "" || isNullish(v) {
				return v
			}
			// The fast path: a plain type NAME is the overwhelming majority of
			// the sites this serves (every declared parameter, return type and
			// stored property), and it must not pay for the container walk.
			if c := ty0[0]; c != '[' && c != '(' {
				if e := ty0[len(ty0)-1]; e != '?' && e != '!' {
					return swAdopt(v, ty0)
				}
			}
			ty := strings.TrimRight(strings.TrimSpace(ty0), "?!")
			if ty == "" {
				return v
			}
			if fmap != nil {
				if mk, ok := fmap.props[ty]; ok && isCallable(mk) {
					return rt.call(mk, jsUndef, []interface{}{v})
				}
			}
			if len(ty) >= 2 && ty[0] == '[' && ty[len(ty)-1] == ']' {
				inner := ty[1 : len(ty)-1]
				if kv := swTySplit(inner, ':'); len(kv) == 2 {
					// A dictionary is {__dict, keys, vals} here too: only the
					// vals array is rewritten, in place - the keys keep their
					// own types and their insertion order.
					o, ok := v.(*jsObject)
					if !ok {
						return v
					}
					if tag, has := o.props["__dict"].(bool); !has || !tag {
						return v
					}
					vals, ok := o.props["vals"].(*jsArray)
					if !ok {
						return v
					}
					vt := strings.TrimSpace(kv[1])
					for i, e := range vals.elems {
						vals.elems[i] = swAdoptDeep(e, vt, fmap)
					}
					return v
				}
				arr, ok := v.(*jsArray)
				if !ok {
					return v
				}
				out := make([]interface{}, len(arr.elems))
				for i, e := range arr.elems {
					out[i] = swAdoptDeep(e, inner, fmap)
				}
				return &jsArray{elems: out}
			}
			if len(ty) >= 2 && ty[0] == '(' && ty[len(ty)-1] == ')' {
				o, ok := v.(*jsObject)
				if !ok {
					return v
				}
				n, ok := swNumProp(o, "__tuple")
				if !ok {
					return v
				}
				tys := swTySplit(ty[1:len(ty)-1], ',')
				// A LABELLED element lives under its label as well as under its
				// index, and both copies have to adopt or t.lo and t.0 disagree.
				lbls := swStrArray(o.props["__tlabels"])
				for i := 0; i < n && i < len(tys); i++ {
					nv := swAdoptDeep(o.props[itoa(i)], swTyElem(tys[i]), fmap)
					o.set(itoa(i), nv)
					if i < len(lbls) && lbls[i] != "" {
						o.set(lbls[i], nv)
					}
				}
				return v
			}
			return swAdopt(v, ty)
		}
		// A property/element read that cannot abort, for the write-site ADOPTION
		// only: the value a write is about to overwrite, or undefined when there
		// is none.
		//
		// The emitter used to lower this as sw_safeget, `js_get` behind a
		// nil/typeof guard, and that had a run-vs-native divergence NO GATE CAN
		// SEE: getMember above has an ADDITIVE arm that answers a {__dict, keys,
		// vals} handle's own entries by key, and the C floor's js_get does not -
		// it reads the object's properties, where a dictionary keeps nothing. So
		// `var d: [String: Double]; d["a"] = 3` adopted under llvm.Run and NOT in
		// the native binary. Answering the dict arm here and in layer 2 makes the
		// two agree by construction. An ARRAY still falls to the ordinary indexed
		// read. Twin: js_swsafeget in languages/lib/swift-rt.metajs.
		getM := m["js_get"]
		m["js_swsafeget"] = func(a []uint64) uint64 {
			o := u(a[0])
			if isNullish(o) {
				return jsHUndefined
			}
			if keys, vals, isDict := dictParts(o); isDict {
				if i := rt.dictMemberIdx(keys, u(a[1])); i >= 0 {
					return w(vals.elems[i])
				}
				return jsHUndefined
			}
			if rt.typeOf(o) != "object" {
				return jsHUndefined
			}
			return getM(a)
		}
		baseIs := m["js_is_type"]
		// js_swfits(v, T) is whether an argument can be passed to a parameter of
		// declared type T, for OVERLOAD SELECTION only. It is NOT the dynamic type
		// test: selection sees the argument BEFORE the declared type adopts it, so
		// an untyped integer literal is still a plain number at a `Double`
		// parameter and Swift converts it. The shared js_is_type gets that wrong in
		// both directions since Swift's Double became a box - it answered "an
		// integral number" for `Double`, so `f(3.0)` chose the `Int` overload, and
		// it has no arm for the name `Bool` at all, so a `Bool` overload was never
		// selectable and the dispatcher fell back to its first entry. The
		// interpreter's swArgFits has said this since 8ff0999; this half had not,
		// which is a live halves divergence --cross never reached. Twins:
		// swArgFits in languages/swift-interpreter.abnf and js_swfits in
		// languages/lib/swift-rt.metajs.
		m["js_swfits"] = func(a []uint64) uint64 {
			ty := rt.toString(u(a[1]))
			t := ty
			if i := strings.IndexByte(t, '<'); i >= 0 {
				t = t[:i]
			}
			t = strings.TrimRight(t, "?!")
			// An `inout` argument arrives as the one-slot {__ref, v}
			// write-back box the call site built, so selection was asking
			// whether a BOX is an Int and every answer was no - two
			// `inout` overloads both ran the first entry, in all three
			// engines (docs/todo.md 1.2). The declared type belongs to
			// what the box HOLDS. Twins: swUnref in
			// languages/lib/swift-rt.metajs and swArgFits in
			// languages/swift-interpreter.abnf.
			v := u(a[0])
			if o, ok := v.(*jsObject); ok {
				if _, isRef := o.props["__ref"]; isRef {
					v = o.props["v"]
					a = []uint64{w(v), a[1]}
				}
			}
			if t == "Double" || t == "Float" {
				if swIsFlo(v) {
					return boolH(true)
				}
				_, isNum := v.(float64)
				return boolH(isNum)
			}
			if t == "Bool" {
				_, ok := v.(bool)
				return boolH(ok)
			}
			if _, _, isInt := swAdoptWidth(t); isInt {
				if swIsFlo(v) {
					return boolH(false)
				}
				if giIsInt(v) {
					return boolH(true)
				}
				n, ok := v.(float64)
				return boolH(ok && math.Floor(n) == n)
			}
			return baseIs(a)
		}
		m["js_swadoptdeep"] = func(a []uint64) uint64 {
			ty, _ := u(a[1]).(string)
			var fmap *jsObject
			if len(a) > 2 {
				fmap, _ = u(a[2]).(*jsObject)
			}
			return w(swAdoptDeep(u(a[0]), ty, fmap))
		}
		// js_swis(v, T) is `v is T` / `v as? T` / `v as! T` (docs/todo.md 1.2).
		//
		// The shared js_is_type predates the float box and gets every numeric name
		// in Swift wrong in BOTH directions: it answers "an integral number" for
		// Int and for Double alike, so `3 is Double` was TRUE and `3.0 is Double`
		// was FALSE - swiftc 6.1.2 says exactly the reverse - and `3 is Float` was
		// true too. It also has no arm for the name `Bool` (its boolean arm spells
		// the Java/Kotlin name `Boolean`), so `true is Bool` was false. Since the
		// same probe drives `as?`, each of those was a wrong VALUE and not merely a
		// wrong flag.
		//
		// The value model already carries the answer: a jsJFlo is a Double, the
		// same box at style floJavaF is a Float, a plain integral float64 or a
		// jsGInt is an integer of the width the box names. Every non-numeric,
		// non-Bool name falls through to the shared probe unchanged.
		//
		// ONE RESIDUE, and it is the value model rather than this test: `Int` and
		// `Int64` are the same width and signedness here and normalise to the same
		// representation, so `8 as Int64 is Int` answers true where swiftc says
		// false. Telling them apart needs a nominal type on the value that nothing
		// else in this front end would use. Twins: swIsNamed in
		// languages/swift-interpreter.abnf and js_swis in lib/swift-rt.metajs.
		m["js_swis"] = func(a []uint64) uint64 {
			tname := rt.toString(u(a[1]))
			t := tname
			if i := strings.IndexByte(t, '<'); i >= 0 {
				t = t[:i]
			}
			opt := strings.HasSuffix(t, "?") || strings.HasSuffix(t, "!")
			t = strings.TrimRight(t, "?!")
			bits, uns, isInt := swAdoptWidth(t)
			isD := t == "Double"
			isF := t == "Float"
			isB := t == "Bool"
			if !isInt && !isD && !isF && !isB {
				return baseIs(a)
			}
			v := u(a[0])
			if isNullish(v) {
				return boolH(opt)
			}
			if isB {
				_, ok := v.(bool)
				return boolH(ok)
			}
			// The box tests come FIRST in all three engines: in the compiled
			// halves a float box and a sized box are floor tags and `typeof`
			// answers "number" for both, so an isD arm above them claimed a big
			// sized integer.
			if swIsFlo(v) {
				if swIsF32(v) {
					return boolH(isF)
				}
				return boolH(isD)
			}
			if gi, ok := v.(jsGInt); ok {
				return boolH(gi.w == bits && gi.u == uns)
			}
			n, ok := v.(float64)
			if !ok {
				return boolH(false)
			}
			if isD {
				return boolH(math.Floor(n) != n)
			}
			if isF {
				return boolH(false)
			}
			// A plain number is a signed 64 bit Int, and only an integral one is
			// an Int at all - a fraction that reached here unboxed is a Double.
			return boolH(math.Floor(n) == n && bits == 64 && !uns)
		}
		// js_swconf(v, "P", [builtin names]) is `v is P` for a USER PROTOCOL, which
		// no name walk can answer: a conforming type's descriptor is named for
		// itself, never for the protocol it adopts (docs/todo.md 1.1 - `x is P` was
		// false for every conforming type in all THREE engines, so --cross was blind
		// to it, and the swift catch clause had to exempt user protocols from
		// filtering because filtering on a false answer turns a caught error into an
		// uncaught one). The emitter writes a __conf_<P> marker on the very table a
		// value reaches through __class at every adoption site, inherited protocols
		// included, and the walk here is the same __class/__super walk swFindMember
		// does - which is what makes a SUBCLASS of a conforming class conform too.
		//
		// The third argument is the statically known list of BUILTIN type names an
		// `extension Int: P` conformed: a builtin value has no descriptor to hang a
		// marker on, and the emitter knows the list at emit time, so it is passed in
		// rather than kept in a runtime registry. Twins: swConforms in
		// languages/swift-interpreter.abnf and js_swconf in lib/swift-rt.metajs.
		m["js_swconf"] = func(a []uint64) uint64 {
			key := "__conf_" + rt.toString(u(a[1]))
			if o, ok := u(a[0]).(*jsObject); ok {
				for cls, n := o.props["__class"], 0; cls != nil && n < 64; n++ {
					clsObj, ok := cls.(*jsObject)
					if !ok {
						break
					}
					if v, ok := clsObj.props[key]; ok && v == true {
						return boolH(true)
					}
					cls = clsObj.props["__super"]
				}
			}
			if len(a) > 2 {
				if arr, ok := u(a[2]).(*jsArray); ok {
					for _, bn := range arr.elems {
						s, ok := bn.(string)
						if !ok {
							continue
						}
						if m["js_swis"]([]uint64{a[0], w(s)}) == boolH(true) {
							return boolH(true)
						}
					}
				}
			}
			return boolH(false)
		}
		// js_swextcall(recv, name, args, tbl) is a method an `extension Int { ... }`
		// added. The emitter decides by NAME which call sites can mean one, and that
		// decision used to be FINAL: `extension Int { func pid() }` beside a
		// `struct S { func pid() }` made `S().pid()` answer the Int one in the
		// compiled halves, where the interpreter (whose mcall consults
		// builtinTableFor(target), null for an instance) and swiftc both answer S's.
		// The receiver decides here instead, by exactly the interpreter's rule: a
		// value that HAS a builtin member table takes the $ext$ table and everything
		// else takes the ordinary member call. Twin: js_swextcall in
		// languages/lib/swift-rt.metajs.
		m["js_swextcall"] = func(a []uint64) uint64 {
			v := u(a[0])
			builtin := false
			switch v.(type) {
			case float64, string, bool, jsGInt, jsJFlo, *jsArray:
				builtin = true
			}
			if o, ok := v.(*jsObject); ok {
				if tag, has := o.props["__dict"]; has && tag == true {
					builtin = true
				}
			}
			if builtin {
				if tbl, ok := u(a[3]).(*jsObject); ok {
					if f, ok := tbl.props[rt.toString(u(a[1]))]; ok && isCallable(f) {
						args := []interface{}{v}
						if arr, ok := u(a[2]).(*jsArray); ok {
							args = append(args, arr.elems...)
						}
						return w(rt.call(f, nil, args))
					}
				}
			}
			return m["js_swmcall"]([]uint64{a[0], a[1], a[2]})
		}
		// String(x) / String(describing: x): the text print would have written.
		m["js_swstr"] = func(a []uint64) uint64 { return rt.wrapStr(rt.swDesc(u(a[0]))) }
		// abs/max/min, Double-aware: a boxed operand keeps its box (abs(-1.5) is
		// 1.5, not the 1 an integer truncation gives) and compares by VALUE.
		m["js_swabs"] = func(a []uint64) uint64 {
			v := u(a[0])
			if f, ok := v.(jsJFlo); ok {
				// abs keeps the operand's WIDTH: abs of a Float is a Float.
				return w(swReFlo(math.Abs(f.f), f.sty == floJavaF))
			}
			if giIsInt(v) {
				n := giVal(rt, v)
				if n < 0 {
					n = -n
				}
				wd, un := giWidthOf(v, v)
				return w(giNorm(n, wd, un))
			}
			return rt.wrapNum(math.Abs(math.Trunc(rt.toNumber(v))))
		}
		// Swift's own min/max are `y < x ? y : x` and `y >= x ? y : x`, so a
		// TIE keeps the LEFT operand for min and takes the right for max, and
		// a NaN operand keeps the left one because every comparison against it
		// is false. Reducing the two doubles to a -1/0/1 comparison threw both
		// away: min(-0.0, 0.0) answered 0.0 where swift 6.1.2 answers -0.0,
		// min(0.0, -0.0) answered -0.0 where swift answers 0.0, and
		// min(nan, 1.0) answered 1.0 where swift answers nan. So the float
		// branch tests the operands directly. The two branches below cannot
		// reach either case - a NaN and a signed zero are always a Double here,
		// hence always a box.
		swPick := func(wantMax bool, want func(int) bool) func(a []uint64) uint64 {
			return func(a []uint64) uint64 {
				l, r := u(a[0]), u(a[1])
				if swIsFlo(l) || swIsFlo(r) {
					x, y := swNum(rt, l), swNum(rt, r)
					if wantMax {
						if y >= x {
							return a[1]
						}
						return a[0]
					}
					if y < x {
						return a[1]
					}
					return a[0]
				}
				if giIsIntegral(l) && giIsIntegral(r) {
					if want(rt.giCmp(l, r)) {
						return a[0]
					}
					return a[1]
				}
				if want(rt.jsCompare(l, r)) {
					return a[0]
				}
				return a[1]
			}
		}
		m["js_swmax"] = swPick(true, func(c int) bool { return c > 0 })
		m["js_swmin"] = swPick(false, func(c int) bool { return c < 0 })
		// The Double methods this subset carries, in front of the runtime's own
		// js_mcall (a float box is not a shape memberCall knows). Verified
		// against swift 6.1.2: 2.0.rounded() is 2.0, 2.5.squareRoot() is
		// 1.5811388300841898 and 7.0.truncatingRemainder(dividingBy: 2.0) is 1.0.
		baseMcall := m["js_mcall"]
		// One deterministic generator per runtime: see swRndState.
		swRnd := &swRndState{s: 20260810}
		m["js_swmcall"] = func(a []uint64) uint64 {
			f, isFlo := u(a[0]).(jsJFlo)
			if !isFlo {
				// insert(_:at:) existed in no engine and aborted
				// with *unknown Array method*, found while closing
				// docs/todo.md 1.2's append bullet. Twins:
				// js_swmcall in languages/lib/swift-rt.metajs and
				// arrayMethod in languages/swift-interpreter.abnf.
				if arr, isArr := u(a[0]).(*jsArray); isArr && rt.toString(u(a[1])) == "insert" {
					if ia, _ := u(a[2]).(*jsArray); ia != nil && len(ia.elems) > 1 {
						at := jsToInt(rt.toNumber(ia.elems[1]))
						if at < 0 {
							at = 0
						}
						if at > len(arr.elems) {
							at = len(arr.elems)
						}
						arr.elems = append(arr.elems, nil)
						copy(arr.elems[at+1:], arr.elems[at:])
						arr.elems[at] = ia.elems[0]
						return jsHUndefined
					}
				}
				// A STORED PROPERTY holding a closure is callable as
				// `q.fn(3)`, and no such name is in the member table -
				// so memberCall aborted with "unknown method 'fn' on
				// an instance". It is consulted only after the
				// __class/__super walk has declined, so a real method
				// of the same name still wins, and it is called
				// WITHOUT a receiver: a function-typed field is a
				// plain value, not a method. Twins: js_mcall in
				// languages/lib/swift-rt.metajs, swMethodCall in
				// languages/swift-interpreter.abnf.
				if fn := swFieldFn(rt, u(a[0]), rt.toString(u(a[1]))); fn != nil {
					args, _ := u(a[2]).(*jsArray)
					var elems []interface{}
					if args != nil {
						elems = args.elems
					}
					return w(rt.call(fn, jsUndef, elems))
				}
				// The Sequence/Collection members (docs/todo.md 2.10), in
				// FRONT of abnf/jsrt.go's shared list table - that table
				// refused with "unknown list method 'sorted'" where the
				// interpreter said "unknown Array method: sorted", so the
				// two engines disagreed about the wording of a refusal as
				// well as about the member. Every Array receiver lands
				// here now, including the refusal.
				if arr, isArr := u(a[0]).(*jsArray); isArr {
					name := rt.toString(u(a[1]))
					var elems []interface{}
					if ia, _ := u(a[2]).(*jsArray); ia != nil {
						elems = ia.elems
					}
					if v, ok := rt.swSeqMethod(arr, name, elems, swRnd); ok {
						return w(v)
					}
					if !swSharedListMethod(name) {
						rt.fail("unknown Array method: %s", name)
					}
				}
				return baseMcall(a)
			}
			name := rt.toString(u(a[1]))
			args, _ := u(a[2]).(*jsArray)
			arg0 := func() float64 {
				if args == nil || len(args.elems) == 0 {
					return 0
				}
				return swNum(rt, args.elems[0])
			}
			// Every one of these keeps the receiver's WIDTH: a Float
			// method answers a Float. squareRoot is correctly rounded at
			// 32 bits by the same double-then-round argument as the
			// operators (sqrt is in Figueroa's list).
			f32 := f.sty == floJavaF
			switch name {
			case "rounded":
				return w(swReFlo(swRoundHalfAway(f.f), f32))
			case "squareRoot":
				return w(swReFlo(math.Sqrt(f.f), f32))
			case "truncatingRemainder":
				return w(swReFlo(math.Mod(f.f, arg0()), f32))
			case "isMultiple":
				return boolH(math.Mod(f.f, arg0()) == 0)
			case "description":
				return rt.wrapStr(swFloDesc(f))
			}
			rt.fail("unknown Double method: %s", name)
			return 0
		}
		// Swift's SMART SHIFT: an over-shift answers 0 (or the sign, for a
		// signed >>) and a NEGATIVE count shifts the other way - `256 >> -2` is
		// 1024, verified against swift 6. giArith already gets the over-shift
		// right; only the negative count is Swift specific.
		m["js_swshift"] = func(a []uint64) uint64 {
			op, l, r := rt.toString(u(a[0])), u(a[1]), u(a[2])
			n := giVal(rt, r)
			if n < 0 {
				if op == "<<" {
					op = ">>"
				} else {
					op = "<<"
				}
				n = -n
			}
			return w(rt.giArith(op, l, giNorm(n, 64, false)))
		}
		// -x and ~x, both at the operand's own width. A fractional Double keeps
		// the floating point negation.
		m["js_swneg"] = func(a []uint64) uint64 {
			v := u(a[0])
			if !swIsWhole(v) {
				// A float negates as a float, so -0.0 keeps its sign and
				// -1.0/0.0 is -inf. Negation is exact, so the width only
				// has to be carried, never re-rounded.
				return w(swReFlo(-swNum(rt, v), swIsF32(v)))
			}
			wd, un := giWidthOf(v, v)
			return w(giNorm(-giVal(rt, v), wd, un))
		}
		m["js_swnot"] = func(a []uint64) uint64 {
			v := u(a[0])
			wd, un := giWidthOf(v, v)
			return w(giNorm(^giVal(rt, v), wd, un))
		}
		// Int(x) / Int8(x) / UInt64(x): (value, bits, unsigned). A String source
		// is FAILABLE - Swift answers nil for anything that is not a complete,
		// in-range integer - and is the one conversion that does not wrap.
		m["js_swintconv"] = func(a []uint64) uint64 {
			v := u(a[0])
			bits := uint8(jsToInt(rt.toNumber(u(a[1]))))
			uns := rt.truthy(u(a[2]))
			if s, ok := v.(string); ok {
				r, ok2 := swIntFromStr(s, bits, uns)
				if !ok2 {
					return jsHNull
				}
				return w(r)
			}
			if isNullish(v) {
				rt.fail("integer conversion of nil")
			}
			return w(rt.giConv(v, bits, uns))
		}
		// The four ordered comparisons. A box has to go through giCmp rather than
		// the shared jsCompare, because a UInt64 above 2^63 reads as a NEGATIVE
		// int64 there, so UInt64.max < 0 would hold.
		rel := func(name string, want func(int) bool) {
			m[name] = func(a []uint64) uint64 {
				l, r := u(a[0]), u(a[1])
				// A float box is an object to the shared comparison, so it has
				// to be unboxed first: `1.5 < 2.0` between two boxes is not a
				// numeric comparison at all.
				if swIsFlo(l) || swIsFlo(r) {
					x, y := swNum(rt, l), swNum(rt, r)
					// A Float operand makes the other side a Float too,
					// so `16777217 < f` compares two binary32 values -
					// swift will only ever have put a literal there.
					if swIsF32(l) || swIsF32(r) {
						x, y = jvmFround(x), jvmFround(y)
					}
					c := 0
					if x < y {
						c = -1
					} else if x > y {
						c = 1
					} else if x != y {
						return jsHFalse // a NaN operand: every ordering is false
					}
					return boolH(want(c))
				}
				if giIsInt(l) || giIsInt(r) {
					if giIsIntegral(l) && giIsIntegral(r) {
						return boolH(want(rt.giCmp(l, r)))
					}
				}
				return boolH(want(rt.jsCompare(l, r)))
			}
		}
		// The members a sized integer answers for itself. Everything else falls
		// straight through to the runtime's own js_get, so this is only ever a
		// type switch in front of the read the grammar already emitted.
		baseGet := m["js_get"]
		m["js_swget"] = func(a []uint64) uint64 {
			// The Double members. .isNaN / .isInfinite are the only way to spot
			// the values a comparison cannot: NaN != NaN.
			if f, ok := u(a[0]).(jsJFlo); ok {
				switch rt.toString(u(a[1])) {
				case "isNaN":
					return boolH(math.IsNaN(f.f))
				case "isInfinite":
					return boolH(math.IsInf(f.f, 0))
				case "isFinite":
					return boolH(!math.IsNaN(f.f) && !math.IsInf(f.f, 0))
				case "isZero":
					return boolH(f.f == 0)
				case "description":
					return rt.wrapStr(swFloDesc(f))
				case "magnitude":
					return w(swReFlo(math.Abs(f.f), f.sty == floJavaF))
				}
			}
			if b, ok := u(a[0]).(jsGInt); ok {
				switch rt.toString(u(a[1])) {
				case "description":
					return rt.wrapStr(giStr(b))
				case "bitWidth":
					return rt.wrapNum(float64(b.w))
				case "magnitude":
					if !b.u && b.v < 0 {
						return w(giNorm(-b.v, b.w, b.u))
					}
					return a[0]
				}
			}
			// indices is a PROPERTY on every Collection and answered nil in
			// all three engines. Only an Array is answered: a String's
			// indices are String.Index values with no printable form here.
			// swiftc 6.1.2 prints `0..<3` for [3, 1, 2].indices.
			if arr, ok := u(a[0]).(*jsArray); ok && rt.toString(u(a[1])) == "indices" {
				return w(swMkRange(0, float64(len(arr.elems)), false))
			}
			// The same three on a value that stayed a plain number, which is
			// where every integer inside +/- 2^53 lives.
			if f, ok := u(a[0]).(float64); ok {
				switch rt.toString(u(a[1])) {
				case "description":
					return rt.wrapStr(rt.swDesc(f))
				case "bitWidth":
					return rt.wrapNum(64)
				case "magnitude":
					return rt.wrapNum(math.Abs(f))
				}
			}
			return baseGet(a)
		}
		rel("js_swlt", func(c int) bool { return c < 0 })
		rel("js_swle", func(c int) bool { return c <= 0 })
		rel("js_swgt", func(c int) bool { return c > 0 })
		rel("js_swge", func(c int) bool { return c >= 0 })

		// Every Swift subscript, and the reason it is not the shared js_pyget any
		// more. js_pyget's Go twin is CPYTHON'S: a missing dictionary key raised a
		// catchable KeyError and `nil[0]` said "'NoneType' object is not
		// subscriptable", so llvm.Run answered a SWIFT program in Python while the
		// native binary ran layer 2's own copy and said "list index out of range" /
		// "KeyError: 9". Three engines, three answers (docs/todo.md 1.12).
		//
		// Rebinding m["js_pyget"] here instead of taking a new name would be chapter
		// 4's rxExtraExterns trap - this map serves EVERY language, so python would
		// have started reporting Swift's messages. The hits still delegate, so only
		// the failure paths are Swift's.
		//
		// The four messages are swiftc 6.1.2's own, read off the toolchain, and the
		// twins of js_swidx in languages/lib/swift-rt.metajs and swIndexFail in
		// languages/swift-interpreter.abnf:
		//
		//	xs[7]  Swift/ContiguousArrayBuffer.swift:692: Fatal error: Index out of range
		//	s[i]   Swift/StringCharacterView.swift:158: Fatal error: String index is out of bounds
		//	d[9]!  m.swift:3: Fatal error: Unexpectedly found nil while unwrapping an Optional value
		//
		// A missing key is the third one because this value model has no Optional
		// (manual chapter 7.9): `d[k]` here IS Swift's `d[k]!`, and a nil base is the
		// same force unwrap. They are rt.fail and not a throw on purpose - none of
		// these is catchable in Swift, where they trap the process.
		const swNilUnwrap = "Fatal error: Unexpectedly found nil while unwrapping an Optional value"
		basePyget := m["js_pyget"]
		m["js_swidx"] = func(a []uint64) uint64 {
			base := u(a[0])
			// a[1...] / a[0..<2] / s[1...3]: a RANGE subscript answers the
			// slice. Swift's ArraySlice/Substring keep the ORIGINAL indices,
			// which a plain array/string cannot; the printed form, the count
			// and every walk agree, and the re-indexing is the one measured
			// divergence (docs/todo.md 2.10). Twins: swSlice in
			// languages/lib/swift-rt.metajs and swift-interpreter.abnf.
			if r, ok := u(a[1]).(*jsObject); ok && swIsRange(r) {
				switch c := base.(type) {
				case *jsArray:
					lo, hi := rt.swSliceBounds(r, len(c.elems))
					out := make([]interface{}, hi-lo)
					copy(out, c.elems[lo:hi])
					return w(&jsArray{elems: out})
				case string:
					lo, hi := rt.swSliceBounds(r, rt.strLen(c))
					return rt.wrapStr(rt.strRange(c, lo, hi))
				}
			}
			if keys, _, ok := dictParts(base); ok {
				if rt.pyDictFind(keys, u(a[1])) < 0 {
					rt.swAbort("%s", swNilUnwrap)
				}
				return basePyget(a)
			}
			if base == nil || base == jsNull {
				rt.swAbort("%s", swNilUnwrap)
			}
			switch o := base.(type) {
			case *jsArray:
				idx := int(rt.toNumber(u(a[1])))
				if idx < 0 {
					idx += len(o.elems)
				}
				if idx < 0 || idx >= len(o.elems) {
					rt.swAbort("Fatal error: Index out of range")
				}
			case string:
				idx := int(rt.toNumber(u(a[1])))
				n := rt.strLen(o)
				if idx < 0 {
					idx += n
				}
				if idx < 0 || idx >= n {
					rt.swAbort("Fatal error: String index is out of bounds")
				}
			}
			return basePyget(a)
		}

		// THE TOP OF THE PROGRAM: an error nobody caught. swift-to-llvm-ir.abnf
		// wraps the whole program in one js_try whose catch calls this, so the
		// value never reaches runJSModule's uncaught-throw handler (nor, natively,
		// the floor's in languages/lib/runtime.c) - both render it through
		// excText/exc_text, which knows the js {name,message} and python
		// {__class,args} shapes and nothing else, so every uncaught Swift `throw'
		// printed "uncaught exception: [object Object]" in both compiled halves.
		// Teaching those handlers Swift is not an option: exc_text is the FLOOR,
		// shared by all sixteen languages. Same shape as ruby's js_rtop.
		//
		// rt.fail's text is "js runtime error: " + this, which is what the floor's
		// fail() host builtin prints too, so llvm.Run and the native binary are
		// byte-identical; swRunTop in swift-interpreter.abnf prints the same line
		// behind its own engine prefix. The wording is swiftc 6.1.2's
		// ("Swift/ErrorType.swift:254: Fatal error: Error raised at top level:
		// m.E.boom") minus the standard-library frame, with swDesc rendering the
		// value - so `m.E.boom' reads `boom', this runtime having no module concept
		// (manual chapter 7.9). docs/todo.md 1.12.
		m["js_swtop"] = func(a []uint64) uint64 {
			rt.swAbort("Fatal error: Error raised at top level: %s", rt.swDesc(u(a[0])))
			return 0
		}
	})
}

// ----------------------------------------------------------------------------
// Integers
//
// Swift's Int is 64 bit and Int8 ... UInt64 are exactly what they say, so both
// Swift halves ride the sized-integer box of jsrtint.go / the sz* layer of
// languages/lib/interp-core.js. What is Swift specific is the SELECTION rule:
// this value model has one number type, and an operation is integer arithmetic
// exactly when both operands are integral - a Double that happens to hold a
// whole number is therefore treated as an Int, the one place the model shows.
//
// DELIBERATE DIVERGENCE, both halves: real Swift TRAPS on signed overflow
// (`Int.max + 1` is a runtime crash) and reserves &+ / &- / &* for wrapping.
// Here + - * WRAP at the operand's width, exactly like &+ / &- / &*. The
// divergence is confined to programs Swift would have killed outright, and is
// documented rather than hidden.

// ----------------------------------------------------------------------------
// Doubles
//
// Swift's Double is a DIFFERENT type from Int, and the handle runtime models
// every number as one JS double, so `7.0 / 2.0` arrived here with exactly the
// operands `7 / 2` has and divided as integers (3, not 3.5). No dynamic rule can
// repair that - 7.0 has no fraction to detect - so the type goes ON the value,
// exactly as commit 636e326 did for Java/Kotlin/Go/C#. Swift reuses THAT box,
// jsJFlo of jsrtjvm.go, which rt.truthy / rt.toNumber / rt.typeOf / rt.strictEq
// already understand; only the RENDERING is Swift's own (swFloStr below), and
// every Swift print path goes through swRender, so the box's style tag is never
// consulted for a Swift value.
//
// Invariant, in both halves:
//
//	a plain number / a jsGInt  ==  an Int (or a sized integer type)
//	a jsJFlo                   ==  a Double (or a Float)
//
// DELIBERATE SIMPLIFICATION, both halves: Float is an ALIAS of Double, i.e. it is
// modelled at 64 bit precision. Every value both types hold exactly prints and
// computes the same; a computation that LOSES precision does not - real swift
// prints `Float(1) / Float(3)` as 0.33333334 and this prints 0.3333333333333333.
// Modelling the 32 bit width honestly needs float32 rounding after every operator
// AND a float32 shortest-round-trip renderer in the frozen JS subset, which has
// neither toPrecision nor Math.fround. Aliasing was chosen over refusing `Float`
// outright because a program that writes Float is asking for floating point.
//
// DELIBERATE SIMPLIFICATION, both halves: real Swift does not implicitly convert
// between Int and Double (`x + 2.0` with `let x = 5` is a compile ERROR, and only
// an untyped literal may take the other side's type). This front end has no type
// checker, so a mixed operation evaluates in floating point - which is the right
// answer for the literal case, `1 + 2.0 == 3.0` - instead of being rejected.

// swAdoptWidth is swIntWidthOf: the (bits, unsigned) an integer type name
// declares, for the declared-type adoption above.
func swAdoptWidth(ty string) (uint8, bool, bool) {
	switch ty {
	case "Int", "Int64":
		return 64, false, true
	case "Int32":
		return 32, false, true
	case "Int16":
		return 16, false, true
	case "Int8":
		return 8, false, true
	case "UInt", "UInt64":
		return 64, true, true
	case "UInt32":
		return 32, true, true
	case "UInt16":
		return 16, true, true
	case "UInt8":
		return 8, true, true
	}
	return 0, false, false
}

func swMkFlo(f float64) jsJFlo { return jsJFlo{f: f} }

func swIsFlo(v interface{}) bool { _, ok := v.(jsJFlo); return ok }

// ----------------------------------------------------------------------------
// Float: the 32 bit width
//
// A Swift `Float` is an IEEE-754 binary32 (swift.org's "Float represents a
// 32-bit floating-point number"), not a Double spelled differently:
// `Float(1) / Float(3)` is 0.33333334 where the Double is 0.3333333333333333.
// The VALUE still travels as a float64 - every float is exactly one - so the
// only thing that goes on the box is the WIDTH, and the tag 14 box already has a
// byte for a per-language print tag. STYLE 3 IS THE FLOAT, the same style byte
// java and kotlin use (floJavaF in jsrtjvm.go), because it means the same thing
// there: a 24 bit significand. Only the RENDERING differs, and swift's never
// goes through jvmFloText - every swift print path is swRender, whose float arm
// is swFloStr/swFloStr32 below.
//
// THE SIMPLIFICATION SWIFT GETS AND THE JVM LANGUAGES DO NOT: swift FORBIDS
// mixed Float/Double (and Float/Int) arithmetic outright - `x + y` with
// `let x: Float` and `let y: Double` is *error: binary operator '+' cannot be
// applied to operands of type 'Float' and 'Double'*, verified against swiftc
// 6.1.2, and so is `x + i` for `let i: Int`. Only an untyped LITERAL may take
// the other side's type. So "if either operand is a Float, both are" is correct
// for every program swiftc accepts, and no JLS-5.6.2 wider-type rule is needed:
// there is no Float-op-Double pair to get backwards.
//
// The width lives here and in layer 2, NOT in the C floor: languages/lib/runtime.c
// is compiled by our own c-to-llvm-ir.abnf, which computes every float at DOUBLE
// precision (ctF("flt") is 8, arithmetic is emitted as integer soft-float), so
// the floor cannot spell `(float)x` at all. Measured and recorded in e5e68b5.
//
// Three engines implement this one specification for swift: the {__flo, __f32}
// box of languages/swift-interpreter.abnf, jsJFlo{sty: floJavaF} here, and
// languages/lib/swift-rt.metajs.

// swMkF32 builds a Float box, rounding to binary32 first. jvmFround is
// `float64(float32(x))`, which is the whole of the width in Go.
func swMkF32(f float64) jsJFlo { return jsJFlo{f: jvmFround(f), sty: floJavaF} }

// swIsF32 reports whether a value is a Float rather than a Double.
func swIsF32(v interface{}) bool {
	f, ok := v.(jsJFlo)
	return ok && f.sty == floJavaF
}

// swReFlo rebuilds a box at the width of the operand(s) it came from: a Float
// result rounds, a Double result does not.
func swReFlo(f float64, f32 bool) jsJFlo {
	if f32 {
		return swMkF32(f)
	}
	return swMkFlo(f)
}

// swIsNumeric reports whether a value takes part in numeric equality at all, so
// that 1.0 == "x" is false rather than 1.0 == 0.
func swIsNumeric(v interface{}) bool {
	switch v.(type) {
	case jsJFlo, jsGInt, float64:
		return true
	}
	return false
}

// swRoundHalfAway is Double.rounded(), i.e. .toNearestOrAwayFromZero - which is
// what Go's math.Round does, and NOT what a half-to-even or half-up rule gives:
// (-2.5).rounded() is -3.
func swRoundHalfAway(f float64) float64 { return math.Round(f) }

// swFloStr is Swift's Double.description (SwiftDtoa): the shortest decimal that
// round-trips, written as a plain decimal when the scientific exponent is >= -4
// AND the value is <= 2^53, and in computerized scientific notation - two digit
// minimum exponent, always signed - outside that, with the integral case always
// carrying a ".0". So 1e15 prints 1000000000000000.0, 1e16 prints 1e+16, and the
// upper boundary is 2^53 rather than 1e16 - see swFloDigits. The infinities are
// "inf"/"-inf" and the NaN is "nan". Every boundary here was read off swift 6.1.2
// rather than remembered. The JS twin is swFloStr in swift-interpreter.abnf and
// the two MUST stay in step, because ./test.sh --cross diffs them.
func swFloStr(d float64) string { return swFloStrW(d, false) }

// swFloStr32 is Float.description: the SAME SwiftDtoa rule read at 24 significant
// bits instead of 53, with the plain/scientific boundary moved from 2^53 down to
// 2^24. Read off swift 6.1.2: 16777216.0 is the last plain value and 16777218.0
// prints 1.6777218e+07, exactly as 9007199254740992.0 / 9.007199254740994e+15 do
// for a Double. There is no two-significant-digit minimum (Float.leastNonzero-
// Magnitude prints "1e-45", not java's "1.4E-45") and no forced ".0" on a
// one-digit mantissa ("1e+20").
func swFloStr32(d float64) string { return swFloStrW(d, true) }

func swFloStrW(d float64, f32 bool) string {
	if math.IsNaN(d) {
		return "nan"
	}
	if math.IsInf(d, 1) {
		return "inf"
	}
	if math.IsInf(d, -1) {
		return "-inf"
	}
	neg := math.Signbit(d)
	a := math.Abs(d)
	s := swFloDigitsW(a, f32)
	if neg {
		return "-" + s
	}
	return s
}

// swFloDesc renders a box in ITS OWN width. Every swift rendering path goes
// through here or through swRender's float arm, which calls it.
func swFloDesc(v jsJFlo) string {
	if v.sty == floJavaF {
		return swFloStr32(v.f)
	}
	return swFloStr(v.f)
}

// swFloDigits renders a >= 0. strconv's 'e' form gives the shortest round-tripping
// digit run and the scientific exponent directly, which is exactly the pair
// SwiftDtoa formats from.
func swFloDigits(a float64) string { return swFloDigitsW(a, false) }

// swFloDigitsW is swFloDigits at either width. `bitSize` is what decides
// "shortest run of digits that round-trips", and the plain/scientific boundary
// moves with it - 2^53 for a Double, 2^24 for a Float.
func swFloDigitsW(a float64, f32 bool) string {
	if a == 0 {
		return "0.0"
	}
	bits, bound := 64, 9007199254740992.0
	if f32 {
		bits, bound = 32, 16777216.0
	}
	s := strconv.FormatFloat(a, 'e', -1, bits)
	i := strings.IndexByte(s, 'e')
	mant, exp := s[:i], s[i+1:]
	e10, _ := strconv.Atoi(exp)
	digits := strings.Replace(mant, ".", "", 1)
	// THE UPPER BOUNDARY IS ON THE VALUE, NOT THE EXPONENT. SwiftDtoa switches to
	// scientific when |v| exceeds 2^53, not when the decimal exponent reaches 16, and
	// the two rules part between 2^53 and 1e16. Read off swift 6.1.2:
	//   9007199254740992.0 (2^53) -> 9007199254740992.0   plain, the last plain value
	//   9007199254740994.0        -> 9.007199254740994e+15 the first scientific one
	//   9999999999999000.0        -> 9.999999999999e+15
	//   1234567890123456.0        -> 1234567890123456.0
	// `e10 >= 16` is subsumed: any |v| with e10 >= 16 is >= 1e16 > 2^53.
	if e10 < -4 || a > bound {
		out := digits[:1]
		if len(digits) > 1 {
			out += "." + digits[1:]
		}
		sign := "+"
		if e10 < 0 {
			sign = "-"
			e10 = -e10
		}
		if e10 < 10 {
			return out + "e" + sign + "0" + strconv.Itoa(e10)
		}
		return out + "e" + sign + strconv.Itoa(e10)
	}
	pt := e10 + 1 // digits before the decimal point
	if pt <= 0 {
		return "0." + strings.Repeat("0", -pt) + digits
	}
	if pt >= len(digits) {
		return digits + strings.Repeat("0", pt-len(digits)) + ".0"
	}
	return digits[:pt] + "." + digits[pt:]
}

// swFloText reports whether the WHOLE string is a decimal float, for the failable
// Double("3.5"): swift answers nil for "3.5x", where a lenient parse stops at the
// first bad character and answers 3.5.
func swFloText(s string) bool {
	i := 0
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		i++
	}
	d0 := i
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	seen := i > d0
	if i < len(s) && s[i] == '.' {
		i++
		d1 := i
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
		}
		if i > d1 {
			seen = true
		}
	}
	if !seen {
		return false
	}
	if i < len(s) && (s[i] == 'e' || s[i] == 'E') {
		i++
		if i < len(s) && (s[i] == '+' || s[i] == '-') {
			i++
		}
		d2 := i
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
		}
		if i == d2 {
			return false
		}
	}
	return i == len(s)
}

// swIsWhole is the interpreter's swWhole: a sized-integer box, or a Double whose
// value is a whole number.
func swIsWhole(v interface{}) bool {
	switch t := v.(type) {
	case jsGInt:
		return true
	case float64:
		return !math.IsNaN(t) && !math.IsInf(t, 0) && t == math.Trunc(t)
	}
	return false
}

// swNum is the plain-number reading of an operand, for the Double side of a
// mixed operation.
func swNum(rt *jsrt, v interface{}) float64 { return giFloat(rt, v) }

// swIntFromStr is Int("17") / UInt8("300"): the failable string initializer. It
// reports false for anything that is not a complete, in-range decimal integer.
func swIntFromStr(s string, w uint8, u bool) (interface{}, bool) {
	i, neg := 0, false
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		neg = s[i] == '-'
		i++
	}
	if i >= len(s) {
		return nil, false
	}
	var acc uint64
	for k := i; k < len(s); k++ {
		c := s[k]
		if c < '0' || c > '9' {
			return nil, false
		}
		d := uint64(c - '0')
		if acc > (1<<64-1-d)/10 {
			return nil, false // beyond 64 bits: out of range for every width
		}
		acc = acc*10 + d
	}
	v := int64(acc)
	if neg {
		v = -v
	}
	// The range check: the truncated value has to read back as what came in.
	out := giNorm(v, w, u)
	var back string
	if b, ok := out.(jsGInt); ok {
		back = giStr(b)
	} else {
		back = strconv.FormatInt(int64(out.(float64)), 10)
	}
	if back != swIntCanon(s, neg) {
		return nil, false
	}
	return out, true
}

// swIntCanon is the canonical decimal text of a digit run - the sign and the
// leading zeros removed - for the range check above.
func swIntCanon(s string, neg bool) string {
	d := strings.TrimLeft(strings.TrimLeft(s, "+-"), "0")
	if d == "" {
		return "0"
	}
	if neg {
		return "-" + d
	}
	return d
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
	case jsGInt:
		// A sized integer prints as its DECIMAL text, not through a double:
		// Int.max would otherwise render as 9.223372036854776e+18.
		return giStr(t)
	case jsJFlo:
		// A Double prints through SWIFT's float rendering: 1.0 is "1.0" and the
		// infinity is "inf", where the shared toString would answer the style of
		// whichever language owns the box's tag. A Float prints at 24 bits.
		return swFloDesc(t)
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

	// A Range printed as itself. Both compiled halves used to answer
	// "[object Object]" - agree-and-wrong, which byte-identity cannot see.
	// swiftc 6.1.2 prints `0..<3` and `1...3`. A ONE-SIDED range prints
	// PartialRangeFrom/Through/UpTo, whose real form names the static element
	// type and so cannot be reproduced without a type checker - the generic
	// parameter is dropped and the rest is byte-exact.
	if t, has := o.props["__range"]; has && t == true {
		rlo, hasLo := o.props["lo"]
		rhi, hasHi := o.props["hi"]
		closed := o.props["closed"] == true
		if !hasLo || isNullish(rlo) {
			nm := "PartialRangeUpTo"
			if closed {
				nm = "PartialRangeThrough"
			}
			return nm + "(upperBound: " + rt.swRender(rhi, false, depth+1) + ")", true
		}
		if !hasHi || isNullish(rhi) {
			return "PartialRangeFrom(lowerBound: " + rt.swRender(rlo, false, depth+1) + ")", true
		}
		op := "..<"
		if closed {
			op = "..."
		}
		return rt.swRender(rlo, false, depth+1) + op + rt.swRender(rhi, false, depth+1), true
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

// swFieldFn answers the OWN stored property `name` when it holds a callable and
// the member table does not answer that name at all - a function-typed field,
// called as `q.fn(3)`. nil means "not this case, take the ordinary path".
func swFieldFn(rt *jsrt, v interface{}, name string) interface{} {
	o, ok := v.(*jsObject)
	if !ok {
		return nil
	}
	if _, hasCls := o.props["__class"]; !hasCls {
		return nil
	}
	p, ok := o.props[name]
	if !ok || !isCallable(p) {
		return nil
	}
	if _, found := swFindMember(v, name); found {
		return nil
	}
	return p
}

func swNumProp(o *jsObject, key string) (int, bool) {
	f, ok := o.props[key].(float64)
	if !ok {
		return 0, false
	}
	return int(f), true
}

// swTySplit splits an annotation's text on the TOP-LEVEL occurrences of one
// separator, ignoring anything nested inside brackets, parentheses or angle
// brackets: `[String: Double]` splits on its own colon and `[String: [Int:
// Double]]` still splits on the outer one. Twins: swTySplit in
// languages/swift-interpreter.abnf, swTySplit in languages/lib/swift-rt.metajs.
func swTySplit(s string, sep byte) []string {
	out := []string{}
	depth := 0
	start := 0
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case c == '[' || c == '(' || c == '<':
			depth++
		case c == ']' || c == ')' || c == '>':
			depth--
		case c == sep && depth == 0:
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return append(out, s[start:])
}

// swTyElem is one tuple element's TYPE, with a `label:` prefix dropped: the
// element of `(x: Double, y: Int)` is written `x:Double` and the type is what
// follows the top-level colon. An element that is itself a dictionary keeps its
// own colon, which is nested and therefore invisible to swTySplit.
func swTyElem(s string) string {
	parts := swTySplit(s, ':')
	return strings.TrimSpace(parts[len(parts)-1])
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
	// A float box compares BY VALUE and ACROSS the box boundary: `1.0 == 1` holds
	// in Swift (the untyped literal takes the Double type). It has to be spelled
	// before the integer path, which would compare 1.5 against Int8(1) as the
	// truncated 1.
	if swIsFlo(l) || swIsFlo(r) {
		if !swIsNumeric(l) || !swIsNumeric(r) {
			return false
		}
		return swNum(rt, l) == swNum(rt, r)
	}
	// A sized-integer box is a value of its own, so two integers have to be
	// compared BY VALUE here rather than falling through to strictEq, which
	// would call a box and a plain number different.
	if giIsInt(l) || giIsInt(r) {
		return rt.giEq(l, r)
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
	// Two RANGES compare by their bounds - a Range is Equatable in Swift and
	// `0..<3 == 0..<3` is true there, where these engines fell through to object
	// IDENTITY and answered false. Adjacent to docs/todo.md 2.10: nothing could
	// build a comparable Range value until `indices` did.
	if swIsRange(l) || swIsRange(r) {
		lr, lrok := l.(*jsObject)
		rr, rrok := r.(*jsObject)
		if !lrok || !rrok || !swIsRange(l) || !swIsRange(r) {
			return false
		}
		return rt.swEqualDepth(lr.props["lo"], rr.props["lo"], depth+1) &&
			rt.swEqualDepth(lr.props["hi"], rr.props["hi"], depth+1) &&
			(lr.props["closed"] == true) == (rr.props["closed"] == true)
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

// ----- The Sequence/Collection members, docs/todo.md 2.10 --------------------
// The Go twin of swSeqMethod in languages/lib/swift-rt.metajs and arraySeqMethod
// in languages/swift-interpreter.abnf. Every member here was measured against
// swiftc 6.1.2 before it was written; the ones that cannot be matched exactly are
// recorded at their arms in the two files above.
//
// It sits in FRONT of abnf/jsrt.go's shared list table, which is the other half of
// the item: that table refuses with "unknown list method 'sorted'" while the
// interpreter said "unknown Array method: sorted", so the two engines disagreed
// even about the WORDING of a refusal. Every Array receiver now lands here and no
// swift value can reach the shared wording.
//
// Nothing here rebinds a SHARED extern (manual chapter 4's rxExtraExterns trap) -
// it is reached only from m["js_swmcall"], which is swift's own name.

// swSortLess is Swift's <, over the values a Swift Array can hold: a UInt64 above
// 2^63 is not negative, a Double is a box, and a String pair compares as text.
// A user type's own `static func <` WINS. `[P(n: 3), P(n: 1)].sorted()` answered
// [3, 1] in all three engines against swiftc 6.1.2's [1, 3]: Comparable is how
// Swift sorts anything that is not a built-in, and the built-in comparison sees
// two opaque objects. The operand's type is asked first, then the other's -
// exactly what sw_opfn does for a written `a < b`.
func (rt *jsrt) swSortLess(a, b interface{}) bool {
	if f, ok := swFindMember(a, "<"); ok {
		return rt.truthy(rt.call(f, jsUndef, []interface{}{a, b}))
	}
	if f, ok := swFindMember(b, "<"); ok {
		return rt.truthy(rt.call(f, jsUndef, []interface{}{a, b}))
	}
	if as, ok := a.(string); ok {
		if bs, ok2 := b.(string); ok2 {
			return as < bs
		}
	}
	if swIsFlo(a) || swIsFlo(b) {
		x, y := swNum(rt, a), swNum(rt, b)
		if swIsF32(a) || swIsF32(b) {
			x, y = jvmFround(x), jvmFround(y)
		}
		return x < y
	}
	if giIsInt(a) || giIsInt(b) {
		if giIsIntegral(a) && giIsIntegral(b) {
			return rt.giCmp(a, b) < 0
		}
	}
	return rt.jsCompare(a, b) < 0
}

// swMergeSort is STABLE. Swift's own sort is introsort and is not stable, so
// stability is not a convergence claim - it is what makes the three engines agree
// with each other on equal keys, which byte-identity requires.
func swMergeSort(xs []interface{}, less func(a, b interface{}) bool) []interface{} {
	if len(xs) < 2 {
		out := make([]interface{}, len(xs))
		copy(out, xs)
		return out
	}
	mid := len(xs) / 2
	l := swMergeSort(xs[:mid], less)
	r := swMergeSort(xs[mid:], less)
	out := make([]interface{}, 0, len(xs))
	i, j := 0, 0
	for i < len(l) && j < len(r) {
		if less(r[j], l[i]) {
			out = append(out, r[j])
			j++
		} else {
			out = append(out, l[i])
			i++
		}
	}
	out = append(out, l[i:]...)
	out = append(out, r[j:]...)
	return out
}

// THERE IS NO RANDOM SOURCE IN THIS PROJECT and there cannot be one the three
// engines agree on byte-for-byte (kotlin's ktRndMake is settled by the same
// argument, and Math.random is deleted from the goja host for it). shuffled()
// draws from a DETERMINISTIC xorshift32 with a fixed seed, so every run of every
// engine sees the same permutation; a test asserts on the SORTED result or the
// count, never on the order. The twins are swRndNext in swift-interpreter.abnf and
// swRndNext in languages/lib/swift-rt.metajs, and all three produce one stream.
type swRndState struct{ s int32 }

func (r *swRndState) next() int32 {
	x := r.s
	x = x ^ (x << 13)
	x = x ^ int32(uint32(x)>>17)
	x = x ^ (x << 5)
	if x == 0 {
		x = 1
	}
	r.s = x
	return x
}

func (r *swRndState) below(n int) int {
	if n <= 1 {
		return 0
	}
	return int(r.next()&1073741823) % n
}

func swSeqK(rt *jsrt, args []interface{}, dflt, n int) int {
	k := dflt
	if len(args) > 0 {
		k = int(swNum(rt, args[0]))
	}
	if k < 0 {
		k = 0
	}
	if k > n {
		k = n
	}
	return k
}

// swMkRange builds the {__range, lo, hi, closed} object both grammars emit.
func swMkRange(lo, hi float64, closed bool) *jsObject {
	o := newJSObject()
	o.set("__range", true)
	o.set("lo", lo)
	o.set("hi", hi)
	o.set("closed", closed)
	return o
}

func swIsRange(v interface{}) bool {
	o, ok := v.(*jsObject)
	if !ok {
		return false
	}
	t, has := o.props["__range"]
	return has && t == true
}

// swSliceBounds fills a ONE-SIDED range's missing end in from the receiver's
// length, which is the only place either can be used.
func (rt *jsrt) swSliceBounds(r *jsObject, n int) (int, int) {
	lo := 0
	if v, has := r.props["lo"]; has && !isNullish(v) {
		lo = int(swNum(rt, v))
	}
	hi := n
	if v, has := r.props["hi"]; has && !isNullish(v) {
		hi = int(swNum(rt, v))
		if c, ok := r.props["closed"]; ok && c == true {
			hi++
		}
	}
	if lo < 0 || hi > n || lo > hi {
		rt.swAbort("Fatal error: Range requires lowerBound <= upperBound")
	}
	return lo, hi
}

func (rt *jsrt) swArgList(v interface{}) []interface{} {
	if a, ok := v.(*jsArray); ok {
		return a.elems
	}
	return nil
}

func (rt *jsrt) swLessOf(args []interface{}) func(a, b interface{}) bool {
	if len(args) > 0 && isCallable(args[0]) {
		f := args[0]
		return func(a, b interface{}) bool {
			return rt.truthy(rt.call(f, jsUndef, []interface{}{a, b}))
		}
	}
	return rt.swSortLess
}

// swSeqMethod answers (value, true) for a member it knows and (nil, false) for
// everything else, so the caller can fall through to the shared table.
func (rt *jsrt) swSeqMethod(o *jsArray, name string, args []interface{}, rnd *swRndState) (interface{}, bool) {
	n := len(o.elems)
	switch name {
	case "isEmpty":
		return n == 0, true
	case "removeFirst":
		// removeFirst() answers the element; removeFirst(k) answers Void. Both
		// trap on an empty array, with swiftc 6.1.2's own two messages.
		if len(args) == 0 {
			if n == 0 {
				rt.swAbort("Fatal error: Can't remove first element from an empty collection")
			}
			f0 := o.elems[0]
			o.elems = o.elems[1:]
			return f0, true
		}
		k := int(swNum(rt, args[0]))
		if k < 0 || k > n {
			rt.swAbort("Fatal error: Can't remove more items from a collection than it contains")
		}
		o.elems = o.elems[k:]
		return jsUndef, true
	case "remove":
		i := int(swNum(rt, args[0]))
		if i < 0 || i >= n {
			rt.swAbort("Fatal error: Index out of range")
		}
		rv := o.elems[i]
		o.elems = append(o.elems[:i], o.elems[i+1:]...)
		return rv, true
	case "removeAll":
		if len(args) > 0 && isCallable(args[0]) {
			keep := make([]interface{}, 0, n)
			for _, e := range o.elems {
				if !rt.truthy(rt.call(args[0], jsUndef, []interface{}{e})) {
					keep = append(keep, e)
				}
			}
			o.elems = keep
			return jsUndef, true
		}
		o.elems = o.elems[:0]
		return jsUndef, true
	case "reversed":
		// reversed() answers a ReversedCollection in Swift, whose printed form
		// names the STATIC element type. See the twin's comment: a plain
		// reversed ARRAY is answered instead and the print is the divergence.
		out := make([]interface{}, 0, n)
		for i := n - 1; i >= 0; i-- {
			out = append(out, o.elems[i])
		}
		return &jsArray{elems: out}, true
	case "sorted":
		return &jsArray{elems: swMergeSort(o.elems, rt.swLessOf(args))}, true
	case "sort":
		o.elems = swMergeSort(o.elems, rt.swLessOf(args))
		return jsUndef, true
	case "shuffled":
		out := make([]interface{}, n)
		copy(out, o.elems)
		for i := len(out) - 1; i > 0; i-- {
			j := rnd.below(i + 1)
			out[i], out[j] = out[j], out[i]
		}
		return &jsArray{elems: out}, true
	case "swapAt":
		x, y := int(swNum(rt, args[0])), int(swNum(rt, args[1]))
		if x < 0 || x >= n || y < 0 || y >= n {
			rt.swAbort("Fatal error: Index out of range")
		}
		o.elems[x], o.elems[y] = o.elems[y], o.elems[x]
		return jsUndef, true
	case "min", "max":
		// An Optional in Swift and this value model has none (manual 7.9), so
		// `print(a.min())` reads `1` where swiftc prints `Optional(1)`. The
		// EMPTY case answers nil, which swiftc prints as `nil` too.
		if n == 0 {
			return jsNull, true
		}
		less := rt.swLessOf(args)
		best := o.elems[0]
		for i := 1; i < n; i++ {
			// Swift's own min/max keep the LEFT operand on a tie.
			if name == "min" {
				if less(o.elems[i], best) {
					best = o.elems[i]
				}
			} else if !less(o.elems[i], best) {
				best = o.elems[i]
			}
		}
		return best, true
	case "prefix":
		k := swSeqK(rt, args, 1, n)
		out := make([]interface{}, k)
		copy(out, o.elems[:k])
		return &jsArray{elems: out}, true
	case "suffix":
		k := swSeqK(rt, args, 1, n)
		out := make([]interface{}, k)
		copy(out, o.elems[n-k:])
		return &jsArray{elems: out}, true
	case "dropFirst":
		k := swSeqK(rt, args, 1, n)
		out := make([]interface{}, n-k)
		copy(out, o.elems[k:])
		return &jsArray{elems: out}, true
	case "dropLast":
		k := swSeqK(rt, args, 1, n)
		out := make([]interface{}, n-k)
		copy(out, o.elems[:n-k])
		return &jsArray{elems: out}, true
	case "firstIndex", "lastIndex", "index":
		// `of:` (a value) or `where:` (a closure), answering an Optional - the
		// same divergence as min/max, and nil for "not found", which IS what
		// swiftc prints.
		var found interface{} = jsNull
		for i := 0; i < n; i++ {
			hit := false
			if len(args) > 0 && isCallable(args[0]) {
				hit = rt.truthy(rt.call(args[0], jsUndef, []interface{}{o.elems[i]}))
			} else if len(args) > 0 {
				hit = rt.swEqual(o.elems[i], args[0])
			}
			if hit {
				if name != "lastIndex" {
					return float64(i), true
				}
				found = float64(i)
			}
		}
		return found, true
	case "allSatisfy":
		for _, e := range o.elems {
			if !rt.truthy(rt.call(args[0], jsUndef, []interface{}{e})) {
				return false, true
			}
		}
		return true, true
	case "compactMap":
		out := make([]interface{}, 0, n)
		for _, e := range o.elems {
			cv := rt.call(args[0], jsUndef, []interface{}{e})
			if !isNullish(cv) {
				out = append(out, cv)
			}
		}
		return &jsArray{elems: out}, true
	case "flatMap":
		out := make([]interface{}, 0, n)
		for _, e := range o.elems {
			fv := rt.call(args[0], jsUndef, []interface{}{e})
			if fa, ok := fv.(*jsArray); ok {
				out = append(out, fa.elems...)
			} else {
				out = append(out, fv)
			}
		}
		return &jsArray{elems: out}, true
	case "contains":
		// contains(_:) is a value test and contains(where:) a PREDICATE; the
		// label does not reach here, so the closure argument tells them apart.
		// The value form stays with the shared table.
		if len(args) > 0 && isCallable(args[0]) {
			for _, e := range o.elems {
				if rt.truthy(rt.call(args[0], jsUndef, []interface{}{e})) {
					return true, true
				}
			}
			return false, true
		}
		return nil, false
	case "indices":
		return swMkRange(0, float64(n), false), true
	case "enumerated":
		// An EnumeratedSequence in Swift, whose printed form names the static
		// type; `Array(a.enumerated())` is what programs use and THAT is
		// byte-exact - `[(offset: 0, element: 3), ...]`.
		out := make([]interface{}, 0, n)
		for i, e := range o.elems {
			pt := newJSObject()
			pt.set("__tuple", float64(2))
			pt.set("__tlabels", &jsArray{elems: []interface{}{"offset", "element"}})
			pt.set("0", float64(i))
			pt.set("offset", float64(i))
			pt.set("1", e)
			pt.set("element", e)
			out = append(out, pt)
		}
		return &jsArray{elems: out}, true
	case "joined":
		// [String] concatenates, [[T]] flattens.
		allArr := n > 0
		for _, e := range o.elems {
			if _, ok := e.(*jsArray); !ok {
				allArr = false
			}
		}
		if allArr && len(args) == 0 {
			out := make([]interface{}, 0, n)
			for _, e := range o.elems {
				out = append(out, e.(*jsArray).elems...)
			}
			return &jsArray{elems: out}, true
		}
		sep := ""
		if len(args) > 0 {
			sep = rt.swDesc(args[0])
		}
		js := ""
		for i, e := range o.elems {
			if i > 0 {
				js = strConcat(js, sep)
			}
			js = strConcat(js, rt.swDesc(e))
		}
		return js, true
	case "reduce":
		// reduce(initial, combine) and reduce(into: seed, combine). The labels
		// do not reach here, so the two forms are told apart by WHAT THE CLOSURE
		// ANSWERS: an `into` body mutates its accumulator and returns Void.
		acc := args[0]
		for _, e := range o.elems {
			r2 := rt.call(args[1], jsUndef, []interface{}{acc, e})
			if _, isUndef := r2.(jsUndefT); !isUndef {
				acc = r2
			}
		}
		return acc, true
	}
	return nil, false
}

// swSharedListMethod names the members abnf/jsrt.go's shared list table already
// answers for an Array receiver, so they keep reaching it - the swift-side table
// above only ADDS members and takes over the refusal. Keeping the list explicit is
// what makes "unknown Array method: X" reachable without re-implementing the eleven
// the shared table already has.
func swSharedListMethod(name string) bool {
	switch name {
	case "add", "append", "pop", "removeLast", "size", "get", "contains", "map",
		"filter", "forEach", "count", "any", "insert", "popLast":
		return true
	}
	return false
}

// swAbort is HOW A SWIFT PROGRAM DIES (docs/todo.md 1.1). rt.fail prefixes
// "js runtime error: ", which sat in front of an otherwise byte-correct
// `Fatal error: Error raised at top level: boom`; a message ALREADY in swiftc's own
// wording is printed bare on stderr and exits 1. The pattern is js_jvtop in
// abnf/jsrtjvm.go, which has done exactly this since docs/todo.md 1.14, and the
// twins are swAbort in languages/lib/swift-rt.metajs and the fail override in
// languages/swift-interpreter.abnf. An engine-level complaint no toolchain would
// say ("unknown Array method: x") still goes through rt.fail and keeps its prefix.
//
// THE EXIT CODE STAYS 1: swiftc traps at 133 and that is declined with the
// measurement - see the long note in languages/swift-interpreter.abnf.
func (rt *jsrt) swAbort(format string, args ...interface{}) {
	fmt.Fprint(warnWriter, wtf8Clean(fmt.Sprintf(format, args...)+"\n"))
	os.Exit(1)
}
