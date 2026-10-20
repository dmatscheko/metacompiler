package abnf

// jsrtswift.go -- Swift's VALUE RENDERING and Swift's `==` for the compiler
// grammar (languages/swift-to-llvm-ir.abnf).
//
// Real Swift prints (swift 6.1.2)
//
//	nil / [1, 2, 3] / ["x": 1] / P(x: 1, y: 2) / a(1, "x") / (x: 1, y: "s")
//
// and both halves render through the same rules; this file is the Go twin of
// `swstr` / `swRender` in swift-interpreter.abnf and the two MUST stay in step,
// because ./test.sh --cross diffs them. It also holds the rest of Swift's own
// js_sw* externs (arithmetic, conversions, adoption, subscripts, type tests).
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

		// String.uppercased() / lowercased() (a[1] true for upper): the full
		// mapping without Final_Sigma. Twin of js_swcase in lib/swift-rt.metajs.
		m["js_swcase"] = func(a []uint64) uint64 {
			mode := caseFull
			if u(a[1]) == true {
				mode |= caseUpper
			}
			return rt.wrapStr(caseMapMode(rt.toString(u(a[0])), mode))
		}

		// String(describing:) / "\(v)" / the thing print writes.
		m["js_swdesc"] = func(a []uint64) uint64 { return rt.wrapStr(rt.swDesc(u(a[0]))) }

		// Swift's print(_:separator:terminator:). a[0] is the argument ARRAY,
		// a[1] the separator and a[2] the terminator; undefined means the
		// default (" " and "\n").
		m["js_swprint"] = func(a []uint64) uint64 {
			fmt.Fprint(outWriter, wtf8Clean(rt.swPrintText(u(a[0]), u(a[1]), u(a[2]), false)))
			return 0
		}
		// The text print (a[3] false) or debugPrint (true) writes, for a call
		// with `to:` or a debugPrint.
		m["js_swprintstr"] = func(a []uint64) uint64 {
			return rt.wrapStr(rt.swPrintText(u(a[0]), u(a[1]), u(a[2]), a[3] == jsHTrue))
		}

		// Swift's ==. `js_seq` compares an Array, a Dictionary and a tuple by
		// IDENTITY; in Swift all three are value types and compare element by
		// element.
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
			return rt.wrap(swTupleOf(arr.elems...))
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
				// AN INTEGER DIVISION BY ZERO IS SWIFT'S, NOT GO'S.
				// `integer divide by zero` is the
				// text of GO'S OWN runtime panic and reaches this
				// through the SHARED sized-integer layer
				// (abnf/jsrtint.go, languages/lib/runtime.c's
				// si_arith) - correct for go and for nobody else.
				// swiftc 6.1.2 says
				//   Swift/IntegerTypes.swift:9012: Fatal error: Division by zero
				// (`... in remainder operation` for %), so the guard sits in FRONT of the shared
				// primitive, in swift's own file, and the floor's
				// Go-correct wording is left exactly as it is.
				// Twins: swGiArith in languages/lib/swift-rt.metajs
				// and swDivGuard in languages/swift-interpreter.abnf.
				if op == "/" && swNum(rt, r) == 0 {
					rt.swAbort("Fatal error: Division by zero")
				}
				if op == "%" && swNum(rt, r) == 0 {
					rt.swAbort("Fatal error: Division by zero in remainder operation")
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
				// jsAdd coerces both to text ("1,23").
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
				f, ok := swFloParse(s)
				if !ok {
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
			return w(swMkFlo(swIntToD(rt, v)))
		}
		// Float(x): the same conversion at the 32 bit width. Every Float in a
		// program comes from here or
		// from js_swadoptf32 (a `let x: Float` annotation); a Float that is
		// already one keeps its width through every operator above. The layer-2
		// twin is js_swflo32 in languages/lib/swift-rt.metajs.
		m["js_swflo32"] = func(a []uint64) uint64 {
			v := u(a[0])
			if s, ok := v.(string); ok {
				// Read at 64 and rounded to 32, where swiftc rounds once. A
				// decimal that lands exactly on a float midpoint AFTER a double
				// rounding would differ; no value in the corpus or the probe
				// does. It is spelled this way because layer 2 and the
				// interpreter have only a double parser, and the three engines
				// must answer the same string.
				f, ok := swFloParse(s)
				if !ok {
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
			return w(swMkF32(swIntToD(rt, v)))
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
		// ----- Declared-type adoption -----
		//
		// An untyped literal takes the FLOATNESS and the WIDTH of the type it
		// is written at. `let d: Double = 3` does this through the
		// js_swflo/js_swadoptf32/js_swintconv calls makeDecl emits; js_swadopt
		// is what the OTHER declared-type sites emit - a parameter, a return
		// type, a stored property, and the element type of an array annotation.
		//
		// An Int literal that does not adopt stays an Int and INTEGER-DIVIDES,
		// so `func g(_ x: Double) -> Double { x / 2 }` called g(3) must adopt to
		// answer 1.5.
		//
		// A value that is neither a bare number nor a sized integer nor a float
		// box is returned UNCHANGED: at these sites the only value a
		// type-correct Swift program can present is numeric, and passing
		// everything else through keeps the adoption from inventing conversions
		// (a String is NOT parsed here, where the failable `let x: Int = "5"`
		// site is a deliberate conversion). The twin of swAdoptTy in
		// languages/swift-interpreter.abnf and of js_swadopt in
		// languages/lib/swift-rt.metajs.
		var swAdoptDeep func(v interface{}, ty string, fmap *jsObject) interface{}
		swAdoptAnn := func(v interface{}, ty string) interface{} { return swAdoptDeep(v, ty, nil) }
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
			if !ok {
				if s, isStr := v.(string); isStr && (ty == "Unicode.Scalar" || ty == "UnicodeScalar") {
					return swMkScalar(s)
				}
				if o, isObj := v.(*jsObject); isObj && strings.HasSuffix(ty, ">") {
					return swAdoptGen(o, ty, swAdoptAnn)
				}
				return v
			}
			if swIsFlo(v) || !isNum {
				return v
			}
			return rt.giConv(v, bits, uns)
		}
		m["js_swadopt"] = func(a []uint64) uint64 {
			ty, _ := u(a[1]).(string)
			return w(swAdopt(u(a[0]), ty))
		}
		m["js_swgensub"] = func(a []uint64) uint64 {
			ty, _ := u(a[1]).(string)
			return w(swGenSub(u(a[0]), ty))
		}
		m["js_swgenfields"] = func(a []uint64) uint64 {
			swGenFields(u(a[0]), swAdoptAnn)
			return w(jsUndef)
		}
		m["js_swsettargs"] = func(a []uint64) uint64 {
			txt, _ := u(a[2]).(string)
			swSetTargs(u(a[0]), u(a[1]), txt)
			return w(jsUndef)
		}
		// The WRITE half of the same rule: `var d: Double = 0; d = 3` must hold
		// 3.0, yet the declaration knows the annotation and the assignment three
		// lines later does not.
		//
		// No var-type table is needed, because this value model already carries
		// the two things an annotation decides - FLOATNESS (the jsJFlo box, at
		// its own width) and integer WIDTH (jsGInt) - on the value itself, and a
		// NUMERIC slot only ever holds its own type. So a write adopts the type
		// of the OLD value. The emitter does not call this for an existential
		// slot (`Any`, a protocol), which holds anything - see swKeepMode.
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
			if o, ok := old.(*jsObject); ok {
				if _, isSc := o.props["__usc"]; isSc {
					return swAdopt(v, "Unicode.Scalar")
				}
				// An instance of a generic type carries its type arguments, and
				// the slot's next value is of that same type.
				if ta, has := o.props["__targs"].(string); has {
					if nv, isObj := v.(*jsObject); isObj {
						if _, cls, _ := swGenClass(o); cls != nil {
							name, _ := cls.props["__name"].(string)
							return swAdoptGen(nv, name+"<"+ta+">", swAdoptAnn)
						}
					}
				}
			}
			return v
		}
		m["js_swadoptlike"] = func(a []uint64) uint64 {
			ty, _ := u(a[2]).(string)
			return w(swAdoptLike(u(a[0]), u(a[1]), ty))
		}
		// The STRUCTURAL declared type. `let a: [Double] =
		// [3, 4]` types the array's ELEMENTS, so a[0] / 2 is 1.5; so does a
		// dictionary's VALUE type (`[String: Double]`), a tuple's element types
		// (`(Double, Int)`) and an Optional annotation (`Double?`). The walk is
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
		// that type. Layer 2 builds no closures and neither does this walk, so
		// for a nested `[(Double) -> Double]` the emitter (which knows every such
		// leaf at compile time) hands the makers down. It is
		// nil for every annotation with no function-type leaf, which is nearly
		// all of them.
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
				// An unlabelled tuple takes the type's labels.
				var fresh []string
				if _, has := o.props["__tlabels"]; !has && len(tys) == n {
					any := false
					for _, t := range tys {
						l := ""
						if lp := swTySplit(t, ':'); len(lp) == 2 {
							l = strings.TrimSpace(lp[0])
						}
						any = any || l != ""
						fresh = append(fresh, l)
					}
					if !any {
						fresh = nil
					}
				}
				for i := 0; i < n && i < len(tys); i++ {
					nv := swAdoptDeep(o.props[itoa(i)], swTyElem(tys[i]), fmap)
					o.set(itoa(i), nv)
					if i < len(lbls) && lbls[i] != "" {
						o.set(lbls[i], nv)
					}
					if fresh != nil && fresh[i] != "" {
						o.set(fresh[i], nv)
					}
				}
				if fresh != nil {
					la := make([]interface{}, len(fresh))
					for i, l := range fresh {
						la[i] = l
					}
					o.set("__tlabels", &jsArray{elems: la})
				}
				return v
			}
			return swAdopt(v, ty)
		}
		// A property/element read that cannot abort, for the write-site ADOPTION
		// only: the value a write is about to overwrite, or undefined when there
		// is none.
		//
		// Not `js_get` behind a nil/typeof guard: getMember has an ADDITIVE arm
		// that answers a {__dict, keys, vals} handle's own entries by key, and the
		// C floor's js_get does not, so `var d: [String: Double]; d["a"] = 3`
		// would adopt under llvm.Run and not natively. Answering the dict arm here
		// and in layer 2 makes the two agree by construction. An ARRAY still falls to the ordinary indexed
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
		// parameter and Swift converts it. The shared js_is_type answers "an
		// integral number" for `Double` (so `f(3.0)` would choose the `Int`
		// overload) and has no arm for the name `Bool`. Twins:
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
			// write-back box the call site built; the declared type belongs
			// to what the box HOLDS. Twins: swUnref in
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
		// js_swcopy(v) is Swift's defining property: EVERY value type is copied
		// whenever it is stored into a location - a var/let binding, an
		// assignment, a non-inout argument. A class instance is a REFERENCE and
		// is never copied, which is what keeps `var b = a; b[0].x = 9` visible
		// through a in an array of class instances (swiftc agrees).
		//
		// COPY AT STORE IS COMPLETE, so there is no copy on READ. Two locations
		// can only alias if one stored a value the other still reaches; if every
		// store deposits a graph nobody else holds, no read can produce one. It
		// is also why `a[0].append(5)` still mutates a in place - a subscript
		// read is not a store.
		//
		// Not copy-on-write: Swift's rule is COW with the value semantics
		// OBSERVABLE, and only the observable half is implemented, eagerly.
		//
		// An enum CASE value is deliberately NOT copied: its payload is reachable
		// only through a pattern binding, which is itself a copy site, so no
		// program can observe two case values sharing one __vals array (measured
		// against swiftc 6.1.2).
		//
		// Twins: swCopy in languages/swift-interpreter.abnf and js_swcopy in
		// languages/lib/swift-rt.metajs.
		var swValCopy func(v interface{}) interface{}
		swValCopy = func(v interface{}) interface{} {
			o, isObj := v.(*jsObject)
			if !isObj {
				// An array has no descriptor and is the other value type; every
				// non-object (number, string, bool, closure, nil) is immutable
				// here and is its own copy.
				if arr, ok := v.(*jsArray); ok {
					out := make([]interface{}, len(arr.elems))
					for i, e := range arr.elems {
						out[i] = swValCopy(e)
					}
					return &jsArray{elems: out}
				}
				return v
			}
			if keys, vals, isDict := dictParts(v); isDict {
				nk := make([]interface{}, len(keys.elems))
				copy(nk, keys.elems)
				nv := make([]interface{}, len(vals.elems))
				for i, e := range vals.elems {
					nv[i] = swValCopy(e)
				}
				d := newJSObject()
				d.set("__dict", true)
				d.set("keys", &jsArray{elems: nk})
				d.set("vals", &jsArray{elems: nv})
				return d
			}
			// An INSTANCE is identified by its descriptor, and only a struct's
			// descriptor carries __struct. A class instance falls through here
			// and is returned as itself.
			if cls, has := o.props["__class"]; has && !isNullish(cls) {
				co, ok := cls.(*jsObject)
				if !ok {
					return v
				}
				if tag, has := co.props["__struct"].(bool); !has || !tag {
					return v
				}
				flds, ok := co.props["__fields"].(*jsArray)
				if !ok {
					return v
				}
				out := newJSObject()
				out.set("__class", cls)
				if ta, has := o.props["__targs"]; has {
					out.set("__targs", ta)
				}
				for _, k := range flds.elems {
					name, ok := k.(string)
					if !ok {
						continue
					}
					// An ABSENT field must stay the twin's undefined rather
					// than a raw Go nil: js_get answers jsUndef for a missing
					// property.
					fv, present := o.props[name]
					if !present {
						fv = jsUndef
					}
					out.set(name, swValCopy(fv))
				}
				return out
			}
			// A LABELLED tuple stores the same value under "0" and under its
			// label; the copy re-establishes that aliasing rather than walking
			// the raw keys, so the two spellings stay the one element.
			if n, ok := swNumProp(o, "__tuple"); ok {
				out := newJSObject()
				out.set("__tuple", float64(n))
				lbls := swStrArray(o.props["__tlabels"])
				if len(lbls) > 0 {
					out.set("__tlabels", o.props["__tlabels"])
				}
				for i := 0; i < n; i++ {
					ev, present := o.props[itoa(i)]
					if !present {
						ev = jsUndef
					}
					e := swValCopy(ev)
					out.set(itoa(i), e)
					if i < len(lbls) && lbls[i] != "" {
						out.set(lbls[i], e)
					}
				}
				return out
			}
			if els, ok := swSetElems(o); ok {
				out := make([]interface{}, len(els.elems))
				for i, e := range els.elems {
					out[i] = swValCopy(e)
				}
				return newSwSet(out)
			}
			return v
		}
		m["js_swcopy"] = func(a []uint64) uint64 { return w(swValCopy(u(a[0]))) }
		// The lazy sequences: sequence(first:next:) / sequence(state:next:) are
		// {__swseq: 1|2, st, fn}, and a user type is a Sequence by its
		// makeIterator() or, being its own iterator, by a COPY of itself. The
		// runtime's own iterators are {__swit: kind}: 1 and 2 unfold the two
		// sequence() forms, 3 walks an array's copy, 4 calls an AnyIterator's
		// closure. nil ends a walk, and a finished iterator stays finished. Twins:
		// swMkIter / swIterNext / swSeqArr / swLazyMethod in
		// languages/swift-interpreter.abnf and languages/lib/swift-rt.metajs.
		swIsLazy := func(v interface{}) bool {
			o, ok := v.(*jsObject)
			if !ok {
				return false
			}
			if _, has := o.props["__swseq"]; has {
				return true
			}
			if _, has := o.props["__swit"]; has {
				return true
			}
			if _, found := swFindMember(o, "makeIterator"); found {
				return true
			}
			_, found := swFindMember(o, "next")
			return found
		}
		noArgs := func() uint64 { return w(&jsArray{}) }
		swMkIter := func(v interface{}) interface{} {
			if !swIsLazy(v) {
				return nil
			}
			o := v.(*jsObject)
			if k, has := o.props["__swseq"]; has {
				it := newJSObject()
				it.set("__swit", k)
				it.set("st", swValCopy(o.props["st"]))
				it.set("fn", o.props["fn"])
				it.set("go", false)
				it.set("done", false)
				return it
			}
			if _, has := o.props["__swit"]; has {
				return o
			}
			if _, found := swFindMember(o, "makeIterator"); found {
				return u(m["js_swmcall"]([]uint64{w(v), rt.wrapStr("makeIterator"), noArgs()}))
			}
			return swValCopy(v)
		}
		swIterNext := func(itv interface{}) interface{} {
			it, _ := itv.(*jsObject)
			if it == nil || it.props["__swit"] == nil {
				r := u(m["js_swmcall"]([]uint64{w(itv), rt.wrapStr("next"), noArgs()}))
				if isUndefOrNull(r) {
					return jsNull
				}
				return r
			}
			if it.props["done"] == true {
				return jsNull
			}
			var r interface{}
			switch swNum(rt, it.props["__swit"]) {
			case 1:
				if it.props["go"] == true {
					r = rt.call(it.props["fn"], jsUndef, []interface{}{it.props["st"]})
				} else {
					r = it.props["st"]
				}
				it.set("go", true)
				it.set("st", r)
			case 2:
				box := newJSObject()
				box.set("__ref", float64(1))
				box.set("v", it.props["st"])
				r = rt.call(it.props["fn"], jsUndef, []interface{}{box})
				it.set("st", box.props["v"])
			case 3:
				arr := it.props["st"].(*jsArray)
				i := int(swNum(rt, it.props["i"]))
				r = jsNull
				if i < len(arr.elems) {
					r = arr.elems[i]
				}
				it.set("i", float64(i+1))
			default:
				r = rt.call(it.props["fn"], jsUndef, nil)
			}
			if isUndefOrNull(r) {
				it.set("done", true)
				return jsNull
			}
			return r
		}
		swSeqArr := func(v interface{}) interface{} {
			it := swMkIter(v)
			if it == nil {
				return v
			}
			out := &jsArray{}
			for e := swIterNext(it); e != jsNull; e = swIterNext(it) {
				out.elems = append(out.elems, e)
			}
			return out
		}
		// A Sequence member on a lazy sequence that its own type does not declare.
		// prefix and first(where:) / contains(where:) stop at the element that
		// decides them, so they answer on an endless sequence too; every other
		// member runs on the elements.
		swLazyMethod := func(recv interface{}, name string, argv uint64) (interface{}, bool) {
			if !swIsLazyName(name) {
				return nil, false
			}
			if arr, isArr := recv.(*jsArray); isArr {
				if name != "makeIterator" {
					return nil, false
				}
				it := newJSObject()
				it.set("__swit", float64(3))
				it.set("st", &jsArray{elems: append([]interface{}{}, arr.elems...)})
				it.set("i", float64(0))
				it.set("done", false)
				return it, true
			}
			if _, found := swFindMember(recv, name); found || !swIsLazy(recv) {
				return nil, false
			}
			o := recv.(*jsObject)
			if name == "next" && o.props["__swit"] != nil {
				return swIterNext(o), true
			}
			if name == "makeIterator" {
				return swMkIter(o), true
			}
			args := rt.swArgList(u(argv))
			fa := len(args) == 1 && isCallable(args[0])
			if name == "prefix" && len(args) == 1 {
				it := swMkIter(o)
				n := -1
				if !fa {
					n = int(swNum(rt, args[0]))
				}
				out := &jsArray{}
				for n < 0 || len(out.elems) < n {
					e := swIterNext(it)
					if e == jsNull || (fa && !rt.truthy(rt.call(args[0], jsUndef, []interface{}{e}))) {
						break
					}
					out.elems = append(out.elems, e)
				}
				return out, true
			}
			if (name == "first" || name == "contains") && fa {
				it := swMkIter(o)
				for e := swIterNext(it); e != jsNull; e = swIterNext(it) {
					if rt.truthy(rt.call(args[0], jsUndef, []interface{}{e})) {
						if name == "first" {
							return e, true
						}
						return true, true
					}
				}
				if name == "first" {
					return jsNull, true
				}
				return false, true
			}
			if name == "next" {
				return nil, false
			}
			return u(m["js_swmcall"]([]uint64{w(swSeqArr(o)), rt.wrapStr(name), argv})), true
		}
		m["js_swmkiter"] = func(a []uint64) uint64 {
			if it := swMkIter(u(a[0])); it != nil {
				return w(it)
			}
			return jsHNull
		}
		m["js_swnext"] = func(a []uint64) uint64 { return w(swIterNext(u(a[0]))) }
		m["js_swseqarr"] = func(a []uint64) uint64 { return w(swSeqArr(u(a[0]))) }
		m["js_swsetof"] = func(a []uint64) uint64 { return w(rt.swSetOf(u(a[0]))) }
		m["js_swadoptdeep"] = func(a []uint64) uint64 {
			ty, _ := u(a[1]).(string)
			var fmap *jsObject
			if len(a) > 2 {
				fmap, _ = u(a[2]).(*jsObject)
			}
			return w(swAdoptDeep(u(a[0]), ty, fmap))
		}
		// js_swis(v, T) is `v is T` / `v as? T` / `v as! T`.
		//
		// The shared js_is_type does not know the float box: it answers "an
		// integral number" for Int and for Double alike (`3 is Double` must be
		// false, `3.0 is Double` true), and its boolean arm spells the Java/Kotlin
		// name `Boolean`, not `Bool`. The same probe drives `as?`.
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
		// `x as Any` of an Optional x: the Optional kept inside the Any, and the
		// cast back out that unwraps it. Twins: swCastOptAny / swOptOpen in
		// languages/swift-interpreter.abnf, js_swoptany / js_swoptopen in
		// lib/swift-rt.metajs.
		m["js_swoptany"] = func(a []uint64) uint64 {
			if isUndefOrNull(u(a[0])) {
				return a[0]
			}
			o := newJSObject()
			o.set("__optany", u(a[0]))
			return w(o)
		}
		m["js_swoptopen"] = func(a []uint64) uint64 {
			if o, ok := u(a[0]).(*jsObject); ok {
				if inner, has := o.props["__optany"]; has {
					return w(inner)
				}
			}
			return a[0]
		}
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
			// answers "number" for both, so an isD arm above them would claim a
			// big sized integer.
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
		// itself, never for the protocol it adopts. The emitter writes a __conf_<P> marker on the very table a
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
		// added. The emitter decides by NAME which call sites can mean one, but that
		// cannot be final: with `extension Int { func pid() }` beside a
		// `struct S { func pid() }`, `S().pid()` must answer S's.
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
		// String(reflecting:): the debug form, as inside a collection. Twins:
		// swStrInit in languages/swift-interpreter.abnf and lib/swift-rt.metajs.
		m["js_swreflect"] = func(a []uint64) uint64 { return rt.wrapStr(rt.swDescIn(u(a[0]))) }
		// String(x) and String(describing: x) are NOT one conversion. The one
		// argument that tells them apart is a SEQUENCE - `String(s.sorted())` is
		// init(_: some Sequence<Character>) and answers "ehllo" where the
		// describing form answers `["e", "h", "l", "l", "o"]`. Every other shape
		// agrees, because an unlabelled String(x) is only valid for a
		// LosslessStringConvertible or a Character sequence. The emitter knows
		// the LABEL statically and picks the extern; this is the unlabelled one.
		// Twins: swStrInit in languages/swift-interpreter.abnf and js_swstrinit
		// in languages/lib/swift-rt.metajs.
		m["js_swstrinit"] = func(a []uint64) uint64 {
			if arr, ok := u(a[0]).(*jsArray); ok {
				var sb strings.Builder
				for _, e := range arr.elems {
					sb.WriteString(rt.swDesc(e))
				}
				return rt.wrapStr(sb.String())
			}
			return rt.wrapStr(rt.swDesc(u(a[0])))
		}
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
		// is false (min(-0.0, 0.0) is -0.0, min(nan, 1.0) is nan). A -1/0/1
		// comparison would lose both, so the float
		// branch tests the operands directly. The two branches below cannot
		// reach either case - a NaN and a signed zero are always a Double here,
		// hence always a box.
		swPick := func(wantMax bool, want func(int) bool) func(a []uint64) uint64 {
			return func(a []uint64) uint64 {
				l, r := u(a[0]), u(a[1])
				if swIsFlo(l) || swIsFlo(r) {
					// The pick is a Double: an Int operand beside a Double is
					// an Int literal, so `min(1, 2.5)` is 1.0.
					x, y := swNum(rt, l), swNum(rt, r)
					f32 := swIsF32(l) || swIsF32(r)
					if wantMax {
						if y >= x {
							return w(swReFlo(y, f32))
						}
						return w(swReFlo(x, f32))
					}
					if y < x {
						return w(swReFlo(y, f32))
					}
					return w(swReFlo(x, f32))
				}
				if giIsIntegral(l) && giIsIntegral(r) {
					if want(rt.giCmp(l, r)) {
						return a[0]
					}
					return a[1]
				}
				// A Comparable user type: `y < x` through its own `<`.
				_, lu := swFindMember(l, "<")
				_, ru := swFindMember(r, "<")
				if lu || ru {
					if rt.swSortLess(r, l) == wantMax {
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
		// zip(a, b): the pairs of a's and b's elements, as many as the shorter
		// side has. Each side arrives as an array, or as a PartialRangeFrom
		// (sw_zipside in swift-to-llvm-ir.abnf), which the other side bounds. Twin
		// of hostGlobals["zip"] in swift-interpreter.abnf.
		m["js_swzip"] = func(a []uint64) uint64 {
			l, r := u(a[0]), u(a[1])
			xs, xok := l.(*jsArray)
			ys, yok := r.(*jsArray)
			var n int
			switch {
			case !xok && !yok:
				rt.fail("zip of two unbounded ranges")
			case !xok:
				n = len(ys.elems)
			case !yok:
				n = len(xs.elems)
			default:
				n = len(xs.elems)
				if len(ys.elems) < n {
					n = len(ys.elems)
				}
			}
			side := func(v interface{}, arr *jsArray, i int) interface{} {
				if arr != nil {
					return arr.elems[i]
				}
				return swNum(rt, v.(*jsObject).props["lo"]) + float64(i)
			}
			out := &jsArray{elems: make([]interface{}, 0, n)}
			for i := 0; i < n; i++ {
				out.elems = append(out.elems, swTupleOf(side(l, xs, i), side(r, ys, i)))
			}
			return w(out)
		}
		// stride(from:to:by:) / stride(from:through:by:) as an array: an Int stride
		// adds the step, a floating-point one computes element i as start + i*step
		// in ONE rounding (the standard library's addingProduct). Twin of swStride
		// in swift-interpreter.abnf.
		plusH := rt.wrapStr("+")
		m["js_swstride"] = func(a []uint64) uint64 {
			from, end, by := u(a[0]), u(a[1]), u(a[2])
			closed := a[3] == jsHTrue
			step := swNum(rt, by)
			if step == 0 {
				rt.swAbort("Fatal error: Stride size must not be zero")
			}
			e := swNum(rt, end)
			past := func(v float64) bool {
				if step > 0 {
					return v > e || (!closed && v == e)
				}
				return v < e || (!closed && v == e)
			}
			out := &jsArray{}
			if !swIsFlo(from) && !swIsFlo(end) && !swIsFlo(by) {
				for cur := a[0]; !past(swNum(rt, u(cur))); cur = m["js_swarith"]([]uint64{plusH, cur, a[2]}) {
					out.elems = append(out.elems, u(cur))
				}
				return w(out)
			}
			// A Float stride reads all three operands at Float width: an
			// untyped literal among them is a Float too.
			f32 := swIsF32(from) || swIsF32(end) || swIsF32(by)
			s := swNum(rt, from)
			if f32 {
				s, e, step = jvmFround(s), jvmFround(e), jvmFround(step)
			}
			for i, v := 1, s; !past(v); i++ {
				out.elems = append(out.elems, swReFlo(v, f32))
				v = math.FMA(float64(i), step, s)
				if f32 {
					v = jvmFround(v)
				}
			}
			return w(out)
		}
		// fatalError / precondition / assert and the rest: kind is swiftc's words
		// ("Fatal error", "Precondition failed", "Assertion failed"), the message
		// optional. Twin of swTrap in swift-interpreter.abnf.
		m["js_swtrap"] = func(a []uint64) uint64 {
			kind := rt.toString(u(a[0]))
			if v := u(a[1]); !isNullish(v) {
				if msg := rt.swDesc(v); msg != "" {
					rt.swAbort("%s: %s", kind, msg)
				}
			}
			rt.swAbort("%s", kind)
			return 0
		}
		// The Double methods this subset carries, in front of the runtime's own
		// js_mcall (a float box is not a shape memberCall knows). Verified
		// against swift 6.1.2: 2.0.rounded() is 2.0, 2.5.squareRoot() is
		// 1.5811388300841898 and 7.0.truncatingRemainder(dividingBy: 2.0) is 1.0.
		baseMcall := m["js_mcall"]
		// One deterministic generator per runtime: see swRndState.
		swRnd := &swRndState{s: 20260810}
		// T.random(in: r), from the same stream: an element of an integer range,
		// or for Double and Float lo + u * (hi - lo) with u in [0, 1) - the
		// type's, since `Double.random(in: 0..<1)` has Int literals for bounds.
		// Twins: swRandomIn in languages/swift-interpreter.abnf and js_swrandom*
		// in languages/lib/swift-rt.metajs.
		swRandom := func(a []uint64, fk int) uint64 {
			r, ok := u(a[0]).(*jsObject)
			if !ok || !swIsRange(r) {
				rt.fail("random(in:) needs a range")
			}
			lo, hi := r.props["lo"], r.props["hi"]
			if fk > 0 {
				uf := float64(swRnd.next()&1073741823) / 1073741824
				// float64() keeps Go from fusing the multiply-add (manual ch. 4).
				return w(swReFlo(swNum(rt, lo)+float64(uf*(swNum(rt, hi)-swNum(rt, lo))), fk == 2))
			}
			n := rt.swRangeLen(r)
			if n <= 0 {
				rt.swAbort("Fatal error: Can't get random value with an empty range")
			}
			return w(swRangeAt(rt, lo, swRnd.below(n)))
		}
		m["js_swrandom"] = func(a []uint64) uint64 { return swRandom(a, 0) }
		// Unicode.Scalar(x) / UnicodeScalar(x): a code point that is a scalar
		// value, or a String of exactly one scalar, else nil. Twins: swScalarOf
		// in languages/swift-interpreter.abnf and js_swscalar in
		// languages/lib/swift-rt.metajs.
		m["js_swscalar"] = func(a []uint64) uint64 {
			v := u(a[0])
			if o, ok := v.(*jsObject); ok {
				if _, is := o.props["__usc"]; is {
					return a[0]
				}
			}
			if s, ok := v.(string); ok {
				us := strUnits(s)
				if (len(us) == 1 && !isHighSurr(rune(us[0])) && !isLowSurr(rune(us[0]))) ||
					(len(us) == 2 && isHighSurr(rune(us[0])) && isLowSurr(rune(us[1]))) {
					return w(swMkScalar(s))
				}
				return w(jsNull)
			}
			cp := swNum(rt, v)
			if cp < 0 || cp > 0x10FFFF || (cp >= 0xD800 && cp <= 0xDFFF) {
				return w(jsNull)
			}
			return w(swMkScalar(string(rune(int(cp)))))
		}
		m["js_swrandomd"] = func(a []uint64) uint64 { return swRandom(a, 1) }
		m["js_swrandomf"] = func(a []uint64) uint64 { return swRandom(a, 2) }
		m["js_swmcall"] = func(a []uint64) uint64 {
			f, isFlo := u(a[0]).(jsJFlo)
			if !isFlo {
				if v, ok := rt.swIntMethod(u(a[0]), rt.toString(u(a[1])), rt.swArgList(u(a[2]))); ok {
					return w(v)
				}
				// A Range's Sequence members run over its elements; contains
				// tests the bounds, so a one-sided range answers it too.
				// Twins: mcall in languages/swift-interpreter.abnf and js_swmcall
				// in languages/lib/swift-rt.metajs.
				if r, ok := u(a[0]).(*jsObject); ok && swIsRange(r) {
					name := rt.toString(u(a[1]))
					args := rt.swArgList(u(a[2]))
					if v, ok := rt.swRangeMethod(r, name, args); ok {
						return w(v)
					}
					if swIsIdxMoveName(name) {
						return w(rt.swCrIxMove(r, name, args))
					}
					lo, hi := r.props["lo"], r.props["hi"]
					if isNullish(lo) || isNullish(hi) {
						rt.fail("method .%s on a one-sided range", name)
					}
					n := int(swNum(rt, hi) - swNum(rt, lo))
					if r.props["closed"] == true {
						n++
					}
					elems := []interface{}{}
					for i := 0; i < n; i++ {
						elems = append(elems, swRangeAt(rt, lo, i))
					}
					res := u(m["js_swmcall"]([]uint64{w(&jsArray{elems: elems}), a[1], a[2]}))
					if (name == "firstIndex" || name == "lastIndex" || name == "index") && !isNullish(res) {
						res = rt.swCrIxAt(r, int(swNum(rt, res)))
					}
					return w(res)
				}
				if els, isSet := swSetElems(u(a[0])); isSet {
					return w(rt.swSetMethod(els, rt.toString(u(a[1])), rt.swArgList(u(a[2])),
						func(arr *jsArray) interface{} {
							return u(m["js_swmcall"]([]uint64{w(arr), a[1], a[2]}))
						}))
				}
				// insert(_:at:). Twins:
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
				// `q.fn(3)`, and no such name is in the member table.
				// It is consulted only after the
				// __class/__super walk has declined, so a real method
				// of the same name still wins, and it is called
				// WITHOUT a receiver: a function-typed field is a
				// plain value, not a method. Twins: js_mcall in
				// languages/lib/swift-rt.metajs, swMethodCall in
				// languages/swift-interpreter.abnf.
				if s, isStr := u(a[0]).(string); isStr {
					if v, ok := swStrMethod(rt, s, rt.toString(u(a[1])), rt.swArgList(u(a[2]))); ok {
						return w(v)
					}
				}
				if v, ok := swLazyMethod(u(a[0]), rt.toString(u(a[1])), a[2]); ok {
					return w(v)
				}
				if fn := swFieldFn(rt, u(a[0]), rt.toString(u(a[1]))); fn != nil {
					args, _ := u(a[2]).(*jsArray)
					var elems []interface{}
					if args != nil {
						elems = args.elems
					}
					return w(rt.call(fn, jsUndef, elems))
				}
				// So is a lazy or computed property whose value is a closure.
				if fn := swGetterFn(rt, u(a[0]), rt.toString(u(a[1]))); fn != nil {
					args, _ := u(a[2]).(*jsArray)
					var elems []interface{}
					if args != nil {
						elems = args.elems
					}
					return w(rt.call(fn, jsUndef, elems))
				}
				// The Sequence/Collection members, in FRONT of
				// abnf/jsrt.go's shared list table (whose refusal wording
				// is not Swift's). Every Array receiver lands here,
				// including the refusal.
				// mapValues / compactMapValues: the same keys, in order, over the
				// new values (compactMapValues drops a nil). Twins: the Dictionary
				// arms of mcall in languages/swift-interpreter.abnf and js_mcall in
				// languages/lib/swift-rt.metajs.
				if keys, vals, isDict := dictParts(u(a[0])); isDict {
					name := rt.toString(u(a[1]))
					if name == "mapValues" || name == "compactMapValues" {
						f := rt.swArgList(u(a[2]))[0]
						nk, nv := []interface{}{}, []interface{}{}
						for i, k := range keys.elems {
							v := rt.call(f, jsUndef, []interface{}{vals.elems[i]})
							if name == "compactMapValues" && isNullish(v) {
								continue
							}
							nk = append(nk, k)
							nv = append(nv, v)
						}
						d := newJSObject()
						d.set("__dict", true)
						d.set("keys", &jsArray{elems: nk})
						d.set("vals", &jsArray{elems: nv})
						return w(d)
					}
					if name == "merging" {
						c := swValCopy(u(a[0]))
						ck, cv, _ := dictParts(c)
						rt.swDictMut(ck, cv, "merge", rt.swArgList(u(a[2])))
						return w(c)
					}
					if v, ok := rt.swDictMut(keys, vals, name, rt.swArgList(u(a[2]))); ok {
						return w(v)
					}
				}
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
				if fn := swStaticFn(u(a[0]), rt.toString(u(a[1]))); fn != nil {
					args, _ := u(a[2]).(*jsArray)
					elems := []interface{}{u(a[0])}
					if args != nil {
						elems = append(elems, args.elems...)
					}
					return w(rt.call(fn, jsUndef, elems))
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
				var rv interface{} = jsUndef
				if args != nil && len(args.elems) > 0 {
					rv = args.elems[0]
				}
				return w(swReFlo(rt.swRoundRule(f.f, rv), f32))
			case "squareRoot":
				return w(swReFlo(math.Sqrt(f.f), f32))
			case "truncatingRemainder":
				return w(swReFlo(math.Mod(f.f, arg0()), f32))
			case "remainder": // IEEE 754's: the quotient rounded to nearest, ties to even
				return w(swReFlo(math.Remainder(f.f, arg0()), f32))
			case "map", "flatMap": // Optional's: an Optional IS its value here
				return w(rt.call(args.elems[0], jsUndef, []interface{}{f}))
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
			// An Int64 / UInt64 box says 65 (swAdoptWidth), where the over-shift
			// is 64.
			if b, ok := l.(jsGInt); ok && b.w > 64 && n >= 64 {
				if op == ">>" && !b.u && b.v < 0 {
					return w(giNorm(-1, b.w, b.u))
				}
				return w(giNorm(0, b.w, b.u))
			}
			return w(rt.giArith(op, l, giNorm(n, 64, false)))
		}
		// type(of: v). Twins: swTypeOf in swift-interpreter.abnf, js_swtypeof in
		// lib/swift-rt.metajs.
		m["js_swtypeof"] = func(a []uint64) uint64 { return rt.wrapStr(rt.swTypeOf(u(a[0]))) }
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
				r, ok2 := swIntFromStr(s, bits, uns, 10)
				if !ok2 {
					return jsHNull
				}
				return w(r)
			}
			if isNullish(v) {
				rt.fail("integer conversion of nil")
			}
			if f, ok := v.(jsJFlo); ok {
				if why := swFloIntWhy(math.Trunc(f.f), bits, uns); why != "" {
					fty := "Double"
					if swIsF32(v) {
						fty = "Float"
					}
					rt.swAbort("Fatal error: %s value cannot be converted to %s because %s", fty, swIntTyName(bits, uns), why)
				}
			}
			return w(rt.giConv(v, bits, uns))
		}
		// Int8(clamping: x), UInt8(exactly: x), Int(s, radix: r):
		// (value, bits, unsigned, first label, radix). The emitter passes the
		// label it saw; "radix" is the string form.
		m["js_swintlab"] = func(a []uint64) uint64 {
			v := u(a[0])
			bits := uint8(jsToInt(rt.toNumber(u(a[1]))))
			uns := rt.truthy(u(a[2]))
			if s, ok := v.(string); ok {
				r, ok2 := swIntFromStr(s, bits, uns, int(giVal(rt, u(a[4]))))
				if !ok2 {
					return jsHNull
				}
				return w(r)
			}
			if isNullish(v) {
				rt.fail("integer conversion of nil")
			}
			return w(rt.swIntLabeled(v, bits, uns, rt.toString(u(a[3]))))
		}
		// A member read that came back undefined: a property a builtin value
		// does not have fails loudly, anything else stays undefined. Twins:
		// swNoProp in languages/swift-interpreter.abnf and lib/swift-rt.metajs.
		m["js_swnoprop"] = func(a []uint64) uint64 {
			v := u(a[0])
			builtin := false
			switch v.(type) {
			case string, float64, bool, jsGInt, jsJFlo, *jsArray:
				builtin = true
			}
			if _, _, ok := dictParts(v); ok {
				builtin = true
			}
			if _, ok := swSetElems(v); ok {
				builtin = true
			}
			if builtin {
				rt.fail("unknown property .%s of %s", rt.toString(u(a[1])), rt.swTypeOf(v))
			}
			return w(jsUndef)
		}
		// String(v, radix: r, uppercase: up).
		m["js_swradixstr"] = func(a []uint64) uint64 {
			return rt.wrapStr(swRadixStr(u(a[0]), int(giVal(rt, u(a[1]))), rt.truthy(u(a[2]))))
		}
		// The four ordered comparisons, over swRel.
		rel := func(name string, want func(int) bool) {
			m[name] = func(a []uint64) uint64 {
				c := rt.swRel(u(a[0]), u(a[1]))
				return boolH(c != swUnordered && want(c))
			}
		}
		// A String's Characters and their count: UAX #29 (abnf/jsrtswiftchars.go).
		// Twins: js_swgsplit / js_swgcount in languages/lib/swift-rt.metajs.
		m["js_swgsplit"] = func(a []uint64) uint64 {
			cs := swcGraphemes(rt.toString(u(a[0])))
			out := make([]interface{}, len(cs))
			for i, c := range cs {
				out[i] = c
			}
			return w(&jsArray{elems: out})
		}
		m["js_swgcount"] = func(a []uint64) uint64 {
			return rt.wrapNum(float64(len(swcGraphemes(rt.toString(u(a[0]))))))
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
				case "sign": // a FloatingPointSign case, compared and printed by name
					sign := "plus"
					if math.Signbit(f.f) {
						sign = "minus"
					}
					return w(swCaseVal("FloatingPointSign", sign))
				}
			}
			if b, ok := u(a[0]).(jsGInt); ok {
				switch rt.toString(u(a[1])) {
				case "description":
					return rt.wrapStr(giStr(b))
				case "bitWidth":
					if b.w > 64 {
						return rt.wrapNum(64)
					}
					return rt.wrapNum(float64(b.w))
				case "magnitude": // the UNSIGNED type of the same width
					if !b.u && b.v < 0 {
						return w(giNorm(-b.v, b.w, true))
					}
					return w(giNorm(b.v, b.w, true))
				}
			}
			// indices is a PROPERTY on every Collection. Only an Array is
			// answered: a String's
			// indices are String.Index values with no printable form here.
			// swiftc 6.1.2 prints `0..<3` for [3, 1, 2].indices.
			if arr, ok := u(a[0]).(*jsArray); ok && rt.toString(u(a[1])) == "indices" {
				return w(swMkRange(0, float64(len(arr.elems)), false))
			}
			// A range's bounds and isEmpty, whatever the bound type. Twins:
			// swGetField in languages/swift-interpreter.abnf and js_swget in
			// languages/lib/swift-rt.metajs.
			if r, ok := u(a[0]).(*jsObject); ok && swIsRange(r) {
				switch rt.toString(u(a[1])) {
				case "lowerBound":
					return w(r.props["lo"])
				case "upperBound":
					return w(r.props["hi"])
				case "isEmpty":
					return boolH(rt.swRangeEmpty(r))
				case "startIndex":
					return w(rt.swCrIxAt(r, 0))
				case "endIndex":
					return w(rt.swCrIxAt(r, rt.swRangeLen(r)))
				// A ClosedRange's indices, as the array of its Index values.
				case "indices":
					if r.props["closed"] != true {
						return w(r)
					}
					ixs := []interface{}{}
					for p := 0; p < rt.swRangeLen(r); p++ {
						ixs = append(ixs, rt.swCrIxAt(r, p))
					}
					return w(&jsArray{elems: ixs})
				}
			}
			// A String's views, as arrays: `.utf8` the UTF-8 bytes and
			// `.unicodeScalars` one String per code point. Twins: swGetField in
			// languages/swift-interpreter.abnf and js_swget in
			// languages/lib/swift-rt.metajs.
			if s, ok := u(a[0]).(string); ok {
				name := rt.toString(u(a[1]))
				if v, ok := swcProp(s, name); ok {
					return w(v)
				}
				switch name {
				case "utf16":
					u16 := strUnits(s)
					out := make([]interface{}, len(u16))
					for i, c := range u16 {
						out[i] = float64(c)
					}
					return w(&jsArray{elems: out})
				case "utf8":
					bs := []byte(s)
					out := make([]interface{}, len(bs))
					for i, c := range bs {
						out[i] = float64(c)
					}
					return w(&jsArray{elems: out})
				case "unicodeScalars":
					out := []interface{}{}
					u16 := strUnits(s)
					for i := 0; i < len(u16); {
						n := 1
						if isHighSurr(rune(u16[i])) && i+1 < len(u16) && isLowSurr(rune(u16[i+1])) {
							n = 2
						}
						out = append(out, swMkScalar(strFromUnits(u16[i:i+n])))
						i += n
					}
					return w(&jsArray{elems: out})
				}
			}
			// `.description` of a built-in value is its String(describing:).
			if rt.toString(u(a[1])) == "description" {
				v := u(a[0])
				_, isStr := v.(string)
				_, isBool := v.(bool)
				_, isArr := v.(*jsArray)
				_, _, isDict := dictParts(v)
				_, isSet := swSetElems(v)
				if isStr || isBool || isArr || isDict || isSet || swIsRange(v) {
					return rt.wrapStr(rt.swDesc(v))
				}
			}
			// `.debugDescription` of a String, Array, Dictionary or Set is the
			// form it prints in INSIDE a collection. Twins: swGetField in
			// languages/swift-interpreter.abnf and js_swget in
			// lib/swift-rt.metajs.
			if rt.toString(u(a[1])) == "debugDescription" {
				v := u(a[0])
				_, isStr := v.(string)
				_, isArr := v.(*jsArray)
				_, _, isDict := dictParts(v)
				_, isSet := swSetElems(v)
				if isStr || isArr || isDict || isSet {
					return rt.wrapStr(rt.swRender(v, true, 0))
				}
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
					return w(giNorm(int64(math.Abs(f)), 64, true))
				}
			}
			return baseGet(a)
		}
		rel("js_swlt", func(c int) bool { return c < 0 })
		rel("js_swle", func(c int) bool { return c <= 0 })
		rel("js_swgt", func(c int) bool { return c > 0 })
		rel("js_swge", func(c int) bool { return c >= 0 })

		// Every Swift subscript. Not the shared js_pyget: its Go twin is
		// CPYTHON'S (a catchable KeyError, "'NoneType' object is not
		// subscriptable"), and rebinding m["js_pyget"] here instead of taking a
		// new name would be chapter 4's rxExtraExterns trap - this map serves EVERY
		// language, so python would report Swift's messages. The hits still
		// delegate, so only the failure paths are Swift's.
		//
		// The four messages are swiftc 6.1.2's own, read off the toolchain, and the
		// twins of js_swidx in languages/lib/swift-rt.metajs and swIndexFail in
		// languages/swift-interpreter.abnf:
		//
		//	xs[7]  Swift/ContiguousArrayBuffer.swift:692: Fatal error: Index out of range
		//	s[i]   Swift/StringCharacterView.swift:158: Fatal error: String index is out of bounds
		//	d[9]!  m.swift:3: Fatal error: Unexpectedly found nil while unwrapping an Optional value
		//
		// A missing key answers nil (`d[k]` is an Optional); the third message is
		// js_swforce's, and a nil base gets it too. They are rt.fail and not a
		// throw on purpose - none of these is catchable in Swift, where they trap
		// the process.
		const swNilUnwrap = "Fatal error: Unexpectedly found nil while unwrapping an Optional value"
		basePyget := m["js_pyget"]
		basePyset := m["js_pyset"]
		// `c[k] = v`: js_pyset, except that `d[k] = nil` REMOVES the key
		// (Dictionary's subscript setter takes a V?). Twins: js_swdset in
		// swift-rt.metajs and setElem in swift-interpreter.abnf.
		m["js_swdset"] = func(a []uint64) uint64 {
			if keys, vals, ok := dictParts(u(a[0])); ok && isNullish(u(a[2])) {
				if i := rt.pyDictFind(keys, u(a[1])); i >= 0 {
					keys.dropIdx()
					keys.elems = append(keys.elems[:i], keys.elems[i+1:]...)
					vals.elems = append(vals.elems[:i], vals.elems[i+1:]...)
				}
				return 0
			}
			// An Array element write is in range or a trap, like the read:
			// Swift never grows an array through its subscript.
			if arr, ok := u(a[0]).(*jsArray); ok {
				if idx := int(giVal(rt, u(a[1]))); idx < 0 || idx >= len(arr.elems) {
					rt.swAbort("Fatal error: Index out of range")
				}
			}
			return basePyset(a)
		}
		// `x!`: an Optional IS its value in this model, so only nil traps. Twin
		// of js_swforce in swift-rt.metajs and ForceSfx in swift-interpreter.abnf.
		m["js_swforce"] = func(a []uint64) uint64 {
			if v := u(a[0]); v == nil || v == jsNull || v == jsUndef {
				rt.swAbort("%s", swNilUnwrap)
			}
			return a[0]
		}
		m["js_swidx"] = func(a []uint64) uint64 {
			base := u(a[0])
			// a[1...] / a[0..<2] / s[1...3]: a RANGE subscript answers the
			// slice. Swift's ArraySlice/Substring keep the ORIGINAL indices,
			// which a plain array/string cannot; the printed form, the count
			// and every walk agree, and the re-indexing is the one known
			// divergence. Twins: swSlice in
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
			// A range's element at an index (see swCrIxAt).
			if r, ok := base.(*jsObject); ok && swIsRange(r) && !swIsRange(u(a[1])) &&
				!isNullish(r.props["lo"]) && !isNullish(r.props["hi"]) {
				p := rt.swCrIxPos(r, u(a[1]))
				if p < 0 || p >= rt.swRangeLen(r) {
					rt.swAbort("Fatal error: Index out of range")
				}
				return w(swRangeAt(rt, r.props["lo"], p))
			}
			if keys, _, ok := dictParts(base); ok {
				if rt.pyDictFind(keys, u(a[1])) < 0 {
					return w(jsNull)
				}
				return basePyget(a)
			}
			if base == nil || base == jsNull {
				rt.swAbort("%s", swNilUnwrap)
			}
			switch o := base.(type) {
			case *jsArray:
				idx := int(rt.toNumber(u(a[1])))
				if idx < 0 || idx >= len(o.elems) {
					rt.swAbort("Fatal error: Index out of range")
				}
			case string:
				idx := int(rt.toNumber(u(a[1])))
				n := rt.strLen(o)
				if idx < 0 || idx >= n {
					rt.swAbort("Fatal error: String index is out of bounds")
				}
			}
			return basePyget(a)
		}

		// THE TOP OF THE PROGRAM: an error nobody caught. swift-to-llvm-ir.abnf
		// wraps the whole program in one js_try whose catch calls this, so the
		// value never reaches runJSModule's uncaught-throw handler (nor, natively,
		// the floor's in languages/lib/runtime.c) - both render through
		// excText/exc_text, which knows the js {name,message} and python
		// {__class,args} shapes only, and is shared by every language. Same shape
		// as ruby's js_rtop.
		//
		// The wording is swiftc 6.1.2's
		// ("Swift/ErrorType.swift:254: Fatal error: Error raised at top level:
		// m.E.boom") minus the standard-library frame, with swDesc rendering the
		// value - so `m.E.boom' reads `boom', this runtime having no module concept
		// (manual chapter 7.9). The same line is printed by swRunTop in
		// swift-interpreter.abnf and js_swtop in languages/lib/swift-rt.metajs.
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
// every number as one JS double, so `7.0 / 2.0` would arrive with exactly the
// operands `7 / 2` has (3.5, not 3). No dynamic rule can
// repair that - 7.0 has no fraction to detect - so the type goes ON the value,
// as Java/Kotlin/Go/C# do. Swift reuses THAT box,
// jsJFlo of jsrtjvm.go, which rt.truthy / rt.toNumber / rt.typeOf / rt.strictEq
// already understand; only the RENDERING is Swift's own (swFloStr below), and
// every Swift print path goes through swRender, so the box's style tag is never
// consulted for a Swift value.
//
// Invariant, in both halves:
//
//	a plain number / a jsGInt  ==  an Int (or a sized integer type)
//	a jsJFlo                   ==  a Double (or a Float - see "Float: the 32 bit width")
//
// DELIBERATE SIMPLIFICATION, both halves: real Swift does not implicitly convert
// between Int and Double (`x + 2.0` with `let x = 5` is a compile ERROR, and only
// an untyped literal may take the other side's type). This front end has no type
// checker, so a mixed operation evaluates in floating point - which is the right
// answer for the literal case, `1 + 2.0 == 3.0` - instead of being rejected.

// swAdoptWidth is swIntWidthOf: the (bits, unsigned) an integer type name
// declares, for the declared-type adoption above. Int64 and UInt64 say 65: see
// swIntWidthOf in swift-interpreter.abnf.
func swAdoptWidth(ty string) (uint8, bool, bool) {
	switch ty {
	case "Int":
		return 64, false, true
	case "Int64":
		return 65, false, true
	case "Int32":
		return 32, false, true
	case "Int16":
		return 16, false, true
	case "Int8":
		return 8, false, true
	case "UInt":
		return 64, true, true
	case "UInt64":
		return 65, true, true
	case "UInt32":
		return 32, true, true
	case "UInt16":
		return 16, true, true
	case "UInt8":
		return 8, true, true
	}
	return 0, false, false
}

// swPrintText is the text print / debugPrint writes for an argument array, a
// separator and a terminator (undefined: " " and "\n"). debugPrint renders each
// item as it prints inside a collection - a String quoted and escaped. Twin of
// swPrintText in swift-interpreter.abnf and lib/swift-rt.metajs.
func (rt *jsrt) swPrintText(argv, sepV, termV interface{}, debug bool) string {
	args, ok := argv.(*jsArray)
	if !ok {
		rt.fail("js_swprint needs an argument array")
	}
	sep := " "
	if !isNullish(sepV) {
		sep = rt.swDesc(sepV)
	}
	term := "\n"
	if !isNullish(termV) {
		term = rt.swDesc(termV)
	}
	out := ""
	for i, e := range args.elems {
		if i > 0 {
			out += sep
		}
		out += rt.swRender(e, debug, 0)
	}
	return out + term
}

// swTupleOf builds the {__tuple: n, "0": ..., "1": ...} object a Swift tuple is.
func swTupleOf(vals ...interface{}) *jsObject {
	o := newJSObject()
	o.set("__tuple", float64(len(vals)))
	for i, e := range vals {
		o.set(itoa(i), e)
	}
	return o
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
// the floor cannot spell `(float)x` at all.
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

// swRoundRule is Double.rounded(_ rule:); the rule is an unqualified case value
// (`.down`) read by name, and no argument is .toNearestOrAwayFromZero - Go's
// math.Round, so (-2.5).rounded() is -3. Twin of swRoundRule in
// languages/swift-interpreter.abnf and lib/swift-rt.metajs.
func (rt *jsrt) swRoundRule(f float64, rv interface{}) float64 {
	rule := "toNearestOrAwayFromZero"
	if o, ok := rv.(*jsObject); ok && o.props["__ecase"] == true {
		rule, _ = o.props["__ename"].(string)
	}
	switch rule {
	case "toNearestOrAwayFromZero":
		return math.Round(f)
	case "toNearestOrEven":
		return math.RoundToEven(f)
	case "up":
		return math.Ceil(f)
	case "down":
		return math.Floor(f)
	case "towardZero":
		return math.Trunc(f)
	case "awayFromZero":
		if f < 0 {
			return math.Floor(f)
		}
		return math.Ceil(f)
	}
	rt.fail("unknown rounding rule: %s", rule)
	return f
}

// swcProp is a Character's property `name` (ok false for any other name); a
// Character is its one-cluster String. Twin of swcProp in
// languages/lib/swift-chars.js, which has the rules.
func swcProp(s, name string) (interface{}, bool) {
	switch name {
	case "isLetter", "isUppercase", "isLowercase", "isNumber", "isWholeNumber", "wholeNumberValue",
		"isWhitespace", "isNewline", "asciiValue", "isASCII", "isHexDigit", "hexDigitValue",
		"isPunctuation", "isSymbol", "isMathSymbol", "isCurrencySymbol", "isCased":
	default:
		return nil, false
	}
	u16 := strUnits(s)
	c, single := 0, len(u16) == 1
	if len(u16) > 0 {
		c = int(u16[0])
		if isHighSurr(rune(u16[0])) && len(u16) > 1 && isLowSurr(rune(u16[1])) {
			c = (c-0xD800)*1024 + int(u16[1]) - 0xDC00 + 0x10000
			single = len(u16) == 2
		}
	}
	opt := func(v int) interface{} {
		if v < 0 {
			return jsNull
		}
		return float64(v)
	}
	switch name {
	case "isCased":
		if !single {
			return caseMapMode(s, caseFull|caseUpper) != s || caseMapMode(s, caseFull) != s, true
		}
		return swcGcBits(c)&16 != 0, true
	case "isPunctuation":
		return swcGcBits(c)&1 != 0, true
	case "isSymbol":
		return swcGcBits(c)&2 != 0, true
	case "isMathSymbol":
		return swcGcBits(c)&4 != 0, true
	case "isCurrencySymbol":
		return swcGcBits(c)&8 != 0, true
	case "asciiValue", "isASCII":
		av := -1
		if s == "\r\n" {
			av = 10
		} else if single && c < 128 {
			av = c
		}
		if name == "isASCII" {
			return av >= 0, true
		}
		return opt(av), true
	case "isLetter":
		return jcIsAlphabetic(c), true
	case "isUppercase":
		return jcIsUpper(c), true
	case "isLowercase":
		return jcIsLower(c), true
	case "isWhitespace":
		return (c >= 9 && c <= 13) || c == 32 || c == 0x85 || c == 0xA0 || c == 0x1680 || (c >= 0x2000 && c <= 0x200A) ||
			c == 0x2028 || c == 0x2029 || c == 0x202F || c == 0x205F || c == 0x3000, true
	case "isNewline":
		return (c >= 10 && c <= 13) || c == 0x85 || c == 0x2028 || c == 0x2029, true
	case "isNumber":
		return swcNumVal(c) != -1, true
	case "isHexDigit", "hexDigitValue":
		hv := -1
		if single {
			hv = swcHexVal(c)
		}
		if name == "isHexDigit" {
			return hv >= 0, true
		}
		return opt(hv), true
	}
	nv := -1
	if single {
		nv = swcNumVal(c)
	}
	if name == "isWholeNumber" {
		return nv >= 0, true
	}
	return opt(nv), true
}

// swcNumVal is java's getNumericValue without its Latin-letter digits: -1 no
// number, -2 a number that is no non-negative integer.
func swcNumVal(c int) int {
	if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= 0xFF21 && c <= 0xFF3A) || (c >= 0xFF41 && c <= 0xFF5A) {
		return -1
	}
	return jcNumericValue(c)
}

// swcHexVal is a hex digit's value, ASCII or fullwidth, or -1.
func swcHexVal(c int) int {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	case c >= 0xFF10 && c <= 0xFF19:
		return c - 0xFF10
	case c >= 0xFF21 && c <= 0xFF26:
		return c - 0xFF21 + 10
	case c >= 0xFF41 && c <= 0xFF46:
		return c - 0xFF41 + 10
	}
	return -1
}

// swCaseVal is a payload-free enum case of a builtin enum type.
func swCaseVal(etype, name string) *jsObject {
	o := newJSObject()
	o.set("__ecase", true)
	o.set("__etype", etype)
	o.set("__ename", name)
	return o
}

// swFloStr is Swift's Double.description (SwiftDtoa): the shortest decimal that
// round-trips, written as a plain decimal when the scientific exponent is >= -4
// AND the value is <= 2^53, and in computerized scientific notation - two digit
// minimum exponent, always signed - outside that, with the integral case always
// carrying a ".0". So 1e15 prints 1000000000000000.0, 1e16 prints 1e+16, and the
// upper boundary is 2^53 rather than 1e16 - see swFloDigits. The infinities are
// "inf"/"-inf" and the NaN is "nan". Every boundary here was read off swift
// 6.1.2. The JS twin is swFloStr in swift-interpreter.abnf and
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

// swFloParse is the failable Double("3.5"), read as swift reads it with strtod:
// the WHOLE string is a sign, then a decimal numeral, a hexadecimal one (0x, an
// optional p exponent), inf, infinity or nan in any case; out of range is inf or
// zero. false for anything else - "3.5x" too, where a lenient parse would answer
// 3.5. Twin of swFloParse in swift-interpreter.abnf and lib/swift-rt.metajs.
func swFloParse(s string) (float64, bool) {
	b := s
	if b != "" && (b[0] == '+' || b[0] == '-') {
		b = b[1:]
	}
	neg := len(b) < len(s) && s[0] == '-'
	switch strings.ToLower(b) {
	case "inf", "infinity":
		if neg {
			return math.Inf(-1), true
		}
		return math.Inf(1), true
	}
	if swIsNanText(strings.ToLower(b)) {
		return math.NaN(), true
	}
	if len(b) > 1 && b[0] == '0' && (b[1] == 'x' || b[1] == 'X') {
		if !swHexText(b[2:]) {
			return 0, false
		}
		// strconv reads a hexadecimal numeral only with its p exponent.
		if !strings.ContainsAny(b, "pP") {
			s += "p0"
		}
	} else if !swFloText(s) {
		return 0, false
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil && err.(*strconv.NumError).Err != strconv.ErrRange {
		return 0, false
	}
	return f, true
}

// swIsNanText reports whether w (lower-cased, unsigned) is one of strtod's NaN
// spellings: "nan", "snan", or "nan(...)" with anything but a parenthesis
// inside. The value model has ONE NaN, so the sign and payload they spell are not
// kept. Twin of swIsNanText in swift-interpreter.abnf and lib/swift-rt.metajs.
func swIsNanText(w string) bool {
	if w == "nan" || w == "snan" {
		return true
	}
	if len(w) < 5 || !strings.HasPrefix(w, "nan(") || !strings.HasSuffix(w, ")") {
		return false
	}
	return !strings.ContainsAny(w[4:len(w)-1], "()")
}

// swHexText reports whether s is hex digits with at most one point, at least one
// digit, then an optional p[+-]digits.
func swHexText(s string) bool {
	i, digits := 0, 0
	point := false
	for ; i < len(s); i++ {
		c := s[i]
		if c == '.' && !point {
			point = true
		} else if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
			digits++
		} else {
			break
		}
	}
	if digits == 0 {
		return false
	}
	if i < len(s) && (s[i] == 'p' || s[i] == 'P') {
		i++
		if i < len(s) && (s[i] == '+' || s[i] == '-') {
			i++
		}
		d := i
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
		}
		if i == d {
			return false
		}
	}
	return i == len(s)
}

// swFloText reports whether the WHOLE string is a decimal float:
// [+-]? digits [. digits] [(e|E) [+-] digits], at least one digit.
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
func swIntTyName(w uint8, u bool) string {
	n := "Int"
	if u {
		n = "UInt"
	}
	if w > 64 {
		return n + "64"
	}
	if w == 64 {
		return n
	}
	return n + strconv.Itoa(int(w))
}

// swFloIntWhy says why the truncated float t does not fit the integer type
// (w, u), in swiftc 6.1.2's trap wording, or "" when it fits. Twin of
// swFloIntWhy in languages/swift-interpreter.abnf.
func swFloIntWhy(t float64, w uint8, u bool) string {
	if math.IsNaN(t) || math.IsInf(t, 0) {
		return "it is either infinite or NaN"
	}
	b := int(w)
	if b > 64 {
		b = 64
	}
	if !u {
		b--
	}
	hi := math.Ldexp(1, b)
	lo := -hi
	if u {
		lo = 0
	}
	if t >= hi {
		return "the result would be greater than " + swIntTyName(w, u) + ".max"
	}
	if t < lo {
		return "the result would be less than " + swIntTyName(w, u) + ".min"
	}
	return ""
}

// swTypeOf is type(of: v): the dynamic type's name, as print shows it.
func (rt *jsrt) swTypeOf(v interface{}) string {
	switch t := v.(type) {
	case bool:
		return "Bool"
	case string:
		return "String"
	case jsGInt:
		return swIntTyName(t.w, t.u)
	case float64:
		if t == math.Trunc(t) {
			return "Int"
		}
		return "Double"
	case *jsArray:
		if len(t.elems) > 0 {
			return "Array<" + rt.swTypeOf(t.elems[0]) + ">"
		}
		return "Array<Any>"
	}
	if swIsFlo(v) {
		if swIsF32(v) {
			return "Float"
		}
		return "Double"
	}
	// A range names its kind and its bound type. Twins: swTypeOf in
	// languages/swift-interpreter.abnf, js_swtypeof in lib/swift-rt.metajs.
	if r, ok := v.(*jsObject); ok && swIsRange(r) {
		lo, hi := r.props["lo"], r.props["hi"]
		k := "Range"
		switch {
		case isNullish(hi):
			k = "PartialRangeFrom"
		case isNullish(lo) && r.props["closed"] == true:
			k = "PartialRangeThrough"
		case isNullish(lo):
			k = "PartialRangeUpTo"
		case r.props["closed"] == true:
			k = "ClosedRange"
		}
		if isNullish(lo) {
			lo = hi
		}
		return k + "<" + rt.swTypeOf(lo) + ">"
	}
	if els, ok := swSetElems(v); ok {
		if len(els.elems) > 0 {
			return "Set<" + rt.swTypeOf(els.elems[0]) + ">"
		}
		return "Set<Any>"
	}
	if keys, vals, ok := dictParts(v); ok {
		if len(keys.elems) == 0 {
			return "Dictionary<Any, Any>"
		}
		return "Dictionary<" + rt.swTypeOf(keys.elems[0]) + ", " + rt.swTypeOf(vals.elems[0]) + ">"
	}
	if o, ok := v.(*jsObject); ok {
		if cls, ok := o.props["__class"].(*jsObject); ok {
			if n, ok := cls.props["__name"].(string); ok {
				return n
			}
		}
	}
	rt.fail("type(of:) of this value is not supported")
	return ""
}

func swNum(rt *jsrt, v interface{}) float64 { return giFloat(rt, v) }

// swIntToD is Double(x) / Float(x) of a number. A UInt64 box is width 65
// (swAdoptWidth), which giFloat reads signed. Twin of swIntToD in
// lib/swift-rt.metajs.
func swIntToD(rt *jsrt, v interface{}) float64 {
	if b, ok := v.(jsGInt); ok && b.u {
		return float64(uint64(b.v))
	}
	return giFloat(rt, v)
}

// swIntFromStr is Int("17") / UInt8("300") / Int("ff", radix: 16): the failable
// string initializer. It reports false for anything that is not a complete,
// in-range integer in the radix - an optional sign, then digits and letters of
// either case. Twin of swIntFromStr in languages/swift-interpreter.abnf and
// lib/swift-rt.metajs.
func swIntFromStr(s string, w uint8, u bool, radix int) (interface{}, bool) {
	neg := false
	if s != "" && (s[0] == '+' || s[0] == '-') {
		neg = s[0] == '-'
		s = s[1:]
	}
	if s == "" || s[0] == '+' || s[0] == '-' || strings.IndexByte(s, '_') >= 0 {
		return nil, false
	}
	mag, err := strconv.ParseUint(s, radix, 64)
	if err != nil {
		return nil, false // not a digit run, or beyond 64 bits: out of range for every width
	}
	v := int64(mag)
	if neg {
		v = -v
	}
	out := giNorm(v, w, u)
	if on, om := swIntMag(out); om != mag || (on != neg && mag != 0) {
		return nil, false
	}
	return out, true
}

// swIntMag is an integer's sign and magnitude, which tell whether a conversion
// kept its value whatever the two widths.
func swIntMag(v interface{}) (bool, uint64) {
	var x int64
	switch t := v.(type) {
	case jsGInt:
		if t.u {
			return false, giU(t.v, t.w)
		}
		x = t.v
	case float64:
		x = int64(t)
	}
	if x < 0 {
		return true, uint64(-x)
	}
	return false, uint64(x)
}

// swRadixStr is String(v, radix:, uppercase:). Twin of swRadixStr in
// languages/swift-interpreter.abnf and lib/swift-rt.metajs.
func swRadixStr(v interface{}, radix int, upper bool) string {
	neg, mag := swIntMag(v)
	out := strconv.FormatUint(mag, radix)
	if upper {
		out = strings.ToUpper(out)
	}
	if neg {
		return "-" + out
	}
	return out
}

// swIntLabeled is Int8(clamping:) / UInt8(exactly:) / Int(truncatingIfNeeded:):
// a value fits a width when its conversion keeps its sign and magnitude;
// clamping answers the bound on the value's side, exactly answers nil (and nil
// for a Double with a fraction). Twin of swIntInit in
// languages/swift-interpreter.abnf and js_swintlab in lib/swift-rt.metajs.
func (rt *jsrt) swIntLabeled(x interface{}, w uint8, u bool, label string) interface{} {
	if f, ok := x.(jsJFlo); ok && label == "exactly" {
		if f.f != math.Trunc(f.f) || math.IsInf(f.f, 0) {
			return jsNull
		}
		c := rt.giConv(f.f, w, u)
		if swIntToD(rt, c) != f.f {
			return jsNull
		}
		return c
	}
	c := rt.giConv(x, w, u)
	if label != "clamping" && label != "exactly" {
		return c
	}
	cn, cm := swIntMag(c)
	xn, xm := swIntMag(x)
	if cn == xn && cm == xm {
		return c
	}
	if label == "exactly" {
		return jsNull
	}
	if xn {
		if u {
			return giNorm(0, w, u)
		}
		return giNorm(math.MinInt64>>(64-swBits(w)), w, u)
	}
	if u {
		return giNorm(-1, w, u)
	}
	return giNorm(math.MaxInt64>>(64-swBits(w)), w, u)
}

// swBits is a width as a bit count: Int64 and UInt64 are width 65 (swAdoptWidth).
func swBits(w uint8) uint {
	if w > 64 {
		return 64
	}
	return uint(w)
}

// ----------------------------------------------------------------------------
// Rendering

// swDesc is Swift's `String(describing:)`: the TOP-LEVEL form, in which a String
// renders as itself. swDescIn is the form used for an element inside a
// collection, a tuple or a struct, where a String is quoted and escaped.
func (rt *jsrt) swDesc(v interface{}) string { return rt.swRender(v, false, 0) }

func (rt *jsrt) swDescIn(v interface{}) string { return rt.swRender(v, true, 0) }

// swQuote is Swift's debugDescription of a String: double quotes and the
// escapes Swift itself emits. Every escaped character is ASCII, so the walk is
// over bytes and leaves the rest of the encoding untouched.
func swQuote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\'':
			b.WriteString(`\'`)
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		case '\r':
			b.WriteString(`\r`)
		case 0:
			b.WriteString(`\0`)
		default:
			// Every other ASCII control character is \u{XX} (swiftc 6.1.2:
			// "bel\u{07}"); above ASCII is itself.
			if c < 32 || c == 127 {
				fmt.Fprintf(&b, "\\u{%02X}", c)
			} else {
				b.WriteByte(c)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
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
		// An Optional kept inside an Any (js_swoptany) prints as Swift's Optional.
		if inner, ok := t.props["__optany"]; ok {
			return "Optional(" + rt.swRender(inner, true, depth+1) + ")"
		}
		if s, ok := t.props["__usc"].(string); ok {
			if nested {
				return swScalarDebug(s)
			}
			return s
		}
		if ty, ok := t.props["__crix"].(string); ok {
			c := "pastEnd"
			if v, has := t.props["v"]; has {
				c = "inRange(" + rt.swRender(v, false, depth+1) + ")"
			}
			if nested {
				return "(extension in Swift):Swift.ClosedRange<Swift." + ty + ">.Index." + c
			}
			return c
		}
		// A Set prints as its members; nested in a collection it is `Set([...])`.
		if els, ok := swSetElems(t); ok {
			s := rt.swRender(els, true, depth)
			if nested {
				return "Set(" + s + ")"
			}
			return s
		}
		if s, ok := rt.swObjDesc(t, depth); ok {
			return s
		}
	}
	return rt.toString(v)
}

// swDictMut answers the Dictionary members that change it in place. The two
// initializers that are not a merge into an empty dictionary arrive under names
// Swift source cannot spell as a member call, with their sequence already turned
// into an array by the emitter; a merge without a closure is
// uniqueKeysWithValues. Twins: swDictMut in languages/swift-interpreter.abnf and
// lib/swift-rt.metajs.
func (rt *jsrt) swDictMut(keys, vals *jsArray, name string, args []interface{}) (interface{}, bool) {
	switch name {
	case "removeValue", "updateValue":
		k := argAt(args, 0)
		if name == "updateValue" {
			k = argAt(args, 1)
		}
		var old interface{} = jsNull
		i := rt.pyDictFind(keys, k)
		if i >= 0 {
			old = vals.elems[i]
		}
		switch {
		case name == "updateValue" && i >= 0:
			vals.elems[i] = args[0]
		case name == "updateValue":
			dictAppend(keys, vals, k, args[0])
		case i >= 0:
			keys.dropIdx()
			keys.elems = append(keys.elems[:i], keys.elems[i+1:]...)
			vals.elems = append(vals.elems[:i], vals.elems[i+1:]...)
		}
		return old, true
	case "removeAll":
		keys.dropIdx()
		keys.elems, vals.elems = nil, nil
		return jsUndef, true
	case "reserveCapacity":
		return jsUndef, true
	case "merge", "init(uniqueKeysWithValues:)":
		var comb interface{}
		if len(args) > 1 {
			comb = args[1]
		}
		add := func(k, v interface{}) {
			i := rt.pyDictFind(keys, k)
			switch {
			case i < 0:
				dictAppend(keys, vals, k, v)
			case comb == nil:
				rt.swAbort("Fatal error: Duplicate values for key: '%s'", rt.swDesc(k))
			default:
				vals.elems[i] = rt.call(comb, jsUndef, []interface{}{vals.elems[i], v})
			}
		}
		if sk, sv, isDict := dictParts(argAt(args, 0)); isDict {
			for i := range sk.elems {
				add(sk.elems[i], sv.elems[i])
			}
		} else if ps, ok := argAt(args, 0).(*jsArray); ok {
			for _, p := range ps.elems {
				t, _ := p.(*jsObject)
				if t == nil {
					rt.fail("Dictionary: an element is not a (key, value) pair")
				}
				add(t.props["0"], t.props["1"])
			}
		}
		return jsUndef, true
	case "init(grouping:by:)":
		es, _ := argAt(args, 0).(*jsArray)
		if es == nil {
			return nil, false
		}
		for _, e := range es.elems {
			k := rt.call(argAt(args, 1), jsUndef, []interface{}{e})
			if i := rt.pyDictFind(keys, k); i >= 0 {
				g := vals.elems[i].(*jsArray)
				g.elems = append(g.elems, e)
			} else {
				dictAppend(keys, vals, k, &jsArray{elems: []interface{}{e}})
			}
		}
		return jsUndef, true
	}
	return nil, false
}

// ----- Set -----
// A Set is {__set: true, elems: [...]}: its members, kept free of duplicates by
// ==, in insertion order. Swift's own iteration order is seeded per process, so no
// order is the specification; insertion order is what makes the engines agree.
// Twins: the Set block of languages/swift-interpreter.abnf and of
// languages/lib/swift-rt.metajs.
func swSetElems(v interface{}) (*jsArray, bool) {
	o, ok := v.(*jsObject)
	if !ok || o.props["__set"] != true {
		return nil, false
	}
	els, ok := o.props["elems"].(*jsArray)
	return els, ok
}

func newSwSet(elems []interface{}) *jsObject {
	o := newJSObject()
	o.set("__set", true)
	o.set("elems", &jsArray{elems: elems})
	return o
}

func (rt *jsrt) swSetIdx(els []interface{}, v interface{}) int {
	for i, e := range els {
		if rt.swEqual(e, v) {
			return i
		}
	}
	return -1
}

// swSetOf builds a Set from an Array or a Set; the grammar has already turned a
// String into its Characters and a Range into its elements.
func (rt *jsrt) swSetOf(v interface{}) *jsObject {
	var src []interface{}
	if els, ok := swSetElems(v); ok {
		src = els.elems
	} else if arr, ok := v.(*jsArray); ok {
		src = arr.elems
	} else if !isNullish(v) {
		rt.fail("Set(): not a sequence")
	}
	out := make([]interface{}, 0, len(src))
	for _, e := range src {
		if rt.swSetIdx(out, e) < 0 {
			out = append(out, e)
		}
	}
	return newSwSet(out)
}

// The members of a, by whether they are in b (keep) or not.
func (rt *jsrt) swSetSel(a, b []interface{}, keep bool) []interface{} {
	out := make([]interface{}, 0, len(a))
	for _, e := range a {
		if (rt.swSetIdx(b, e) >= 0) == keep {
			out = append(out, e)
		}
	}
	return out
}

func swSetAlgebra(n string) bool {
	switch n {
	case "union", "formUnion", "intersection", "formIntersection", "subtracting", "subtract",
		"symmetricDifference", "formSymmetricDifference", "isSubset", "isSuperset",
		"isStrictSubset", "isStrictSuperset", "isDisjoint":
		return true
	}
	return false
}

// swSetMethod answers the Set members; seq answers a Sequence member over an Array.
func (rt *jsrt) swSetMethod(set *jsArray, name string, args []interface{}, seq func(*jsArray) interface{}) interface{} {
	els := set.elems
	var a0 interface{} = jsUndef
	if len(args) > 0 {
		a0 = args[0]
	}
	switch {
	case name == "insert":
		at := rt.swSetIdx(els, a0)
		mem := a0
		if at < 0 {
			set.elems = append(els, a0)
		} else {
			mem = els[at]
		}
		t := newJSObject()
		t.set("__tuple", float64(2))
		t.set("__tlabels", &jsArray{elems: []interface{}{"inserted", "memberAfterInsert"}})
		t.set("0", at < 0)
		t.set("inserted", at < 0)
		t.set("1", mem)
		t.set("memberAfterInsert", mem)
		return t
	case name == "contains" && !isCallable(a0):
		return rt.swSetIdx(els, a0) >= 0
	case name == "remove":
		k := rt.swSetIdx(els, a0)
		if k < 0 {
			return jsNull
		}
		rv := els[k]
		set.elems = append(els[:k:k], els[k+1:]...)
		return rv
	case name == "removeAll" && len(args) == 0:
		set.elems = []interface{}{}
		return jsUndef
	case !swSetAlgebra(name):
		r := seq(&jsArray{elems: append([]interface{}{}, els...)})
		if name == "filter" {
			return rt.swSetOf(r)
		}
		return r
	}
	o := rt.swSetOf(a0).props["elems"].(*jsArray).elems
	var res []interface{}
	switch name {
	case "union", "formUnion":
		res = append([]interface{}{}, els...)
		for _, e := range o {
			if rt.swSetIdx(res, e) < 0 {
				res = append(res, e)
			}
		}
	case "intersection", "formIntersection":
		res = rt.swSetSel(els, o, true)
	case "subtracting", "subtract":
		res = rt.swSetSel(els, o, false)
	case "symmetricDifference", "formSymmetricDifference":
		res = append(rt.swSetSel(els, o, false), rt.swSetSel(o, els, false)...)
	case "isSubset":
		return len(rt.swSetSel(els, o, false)) == 0
	case "isSuperset":
		return len(rt.swSetSel(o, els, false)) == 0
	case "isStrictSubset":
		return len(rt.swSetSel(els, o, false)) == 0 && len(els) < len(o)
	case "isStrictSuperset":
		return len(rt.swSetSel(o, els, false)) == 0 && len(els) > len(o)
	default:
		return len(rt.swSetSel(els, o, true)) == 0
	}
	if strings.HasPrefix(name, "form") || name == "subtract" {
		set.elems = res
		return jsUndef
	}
	return newSwSet(res)
}

// swObjDesc renders the object shapes the two Swift grammars build. It reports
// false for a shape it does not know, which falls back to the generic ToString.
func (rt *jsrt) swObjDesc(o *jsObject, depth int) (string, bool) {
	// A builtin's type value (`Int.self`, `type(of: "x")`) is its name. Twins:
	// swObjDesc in lib/swift-rt.metajs and swRender in swift-interpreter.abnf.
	if n, ok := o.props["__tname"].(string); ok {
		return n, true
	}
	// A user `description` (CustomStringConvertible) wins over every built-in
	// rendering, exactly as it does in Swift. Both halves store a computed
	// property as the method __get_<name> on the type descriptor.
	if g, ok := swFindMember(o, "__get_description"); ok {
		return rt.swDesc(rt.call(g, jsUndef, []interface{}{o})), true
	}
	if d, ok := o.props["description"]; ok && !isCallable(d) {
		return rt.swDesc(d), true
	}

	// A Range printed as itself: swiftc 6.1.2 prints `0..<3` and `1...3`. A
	// ONE-SIDED range prints
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
		// An observed property's storage prints under the property's own name.
		parts = append(parts, strings.TrimPrefix(f, "__v_")+": "+rt.swRender(val, true, depth+1))
	}
	return name + rt.swGenArgsTxt(o) + "(" + strings.Join(parts, ", ") + ")", true
}

// ----- A generic type's arguments, per instance -----
// See swGenMark in languages/lib/swift-dotwant.js: an instance carries its type
// arguments as text (__targs), its descriptor the parameter names (__tgens) and
// the generic stored properties (__gfields, declared types in __gftys).

func swGenIdCh(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// swGenClass is the descriptor of o when it is an instance of a generic type.
func swGenClass(v interface{}) (*jsObject, *jsObject, string) {
	o, ok := v.(*jsObject)
	if !ok {
		return nil, nil, ""
	}
	cls, ok := o.props["__class"].(*jsObject)
	if !ok {
		return nil, nil, ""
	}
	tg, ok := cls.props["__tgens"].(string)
	if !ok {
		return nil, nil, ""
	}
	return o, cls, tg
}

// swGenSub answers ty with each generic parameter of self's type replaced by
// self's type argument, or by "_" (which adopts nothing) when self carries none;
// "" when self is not an instance of a generic type. Twins: swGenSub in
// languages/swift-interpreter.abnf and js_swgensub in lib/swift-rt.metajs.
func swGenSub(self interface{}, ty string) string {
	o, _, tg := swGenClass(self)
	if o == nil {
		return ""
	}
	names := swTySplit(tg, ',')
	var args []string
	if ta, has := o.props["__targs"].(string); has {
		if args = swTySplit(ta, ','); len(args) != len(names) {
			args = nil
		}
	}
	var sb strings.Builder
	i := 0
	for i < len(ty) {
		if !swGenIdCh(ty[i]) {
			sb.WriteByte(ty[i])
			i++
			continue
		}
		s := i
		for i < len(ty) && swGenIdCh(ty[i]) {
			i++
		}
		tok := ty[s:i]
		if s == 0 || ty[s-1] != '.' {
			for k, n := range names {
				if n == tok {
					tok = "_"
					if args != nil {
						tok = args[k]
					}
					break
				}
			}
		}
		sb.WriteString(tok)
	}
	return sb.String()
}

// swTyHasHole reports whether type text names the placeholder `_`.
func swTyHasHole(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '_' && (i == 0 || !swGenIdCh(s[i-1])) && (i+1 == len(s) || !swGenIdCh(s[i+1])) {
			return true
		}
	}
	return false
}

func swGenBase(n string) string { return n[strings.LastIndexByte(n, '.')+1:] }

// swSetTargs puts the written type arguments txt on obj, a new instance of a
// generic type. A generic parameter of the enclosing instance's type among them
// is that instance's argument, and one left unbound binds nothing. Twins:
// swSetTargs in languages/swift-interpreter.abnf, js_swsettargs in
// lib/swift-rt.metajs.
func swSetTargs(obj, self interface{}, txt string) {
	o, _, _ := swGenClass(obj)
	if o == nil {
		return
	}
	s := swGenSub(self, txt)
	if s == "" {
		s = txt
	}
	if !swTyHasHole(s) {
		o.set("__targs", s)
	}
}

// swAdoptGen: an instance of a generic type reaching a slot that spells its type
// arguments takes them, and its generic stored properties adopt what they now
// declare; one that carries arguments already keeps them. Twins: swAdoptGen in
// languages/swift-interpreter.abnf and lib/swift-rt.metajs.
func swAdoptGen(v *jsObject, ty string, adopt func(interface{}, string) interface{}) interface{} {
	o, cls, _ := swGenClass(v)
	if o == nil {
		return v
	}
	if _, has := o.props["__targs"]; has {
		return v
	}
	name, _ := cls.props["__name"].(string)
	lt := strings.IndexByte(ty, '<')
	if lt < 0 || swGenBase(ty[:lt]) != swGenBase(name) {
		return v
	}
	args := ty[lt+1 : len(ty)-1]
	if swTyHasHole(args) {
		return v
	}
	// An enum case: its payload adopts. A case without one is a shared value and
	// has nothing to adopt.
	if ec, _ := o.props["__ecase"].(bool); ec {
		vals, ok := o.props["__vals"].(*jsArray)
		if !ok || len(vals.elems) == 0 {
			return v
		}
		o.set("__targs", args)
		ename, _ := o.props["__ename"].(string)
		cts := swStrArray(cls.props["__gcase_"+ename])
		for j := 0; j < len(vals.elems) && j < len(cts); j++ {
			if cts[j] != "" {
				vals.elems[j] = adopt(vals.elems[j], swGenSub(o, cts[j]))
			}
		}
		return v
	}
	o.set("__targs", args)
	swGenFields(o, adopt)
	return v
}

// swGenFields: the generic stored properties of v adopt what its type arguments
// make them, after a construction that bound them and in swAdoptGen. Twins:
// swGenFields in languages/swift-interpreter.abnf, js_swgenfields in
// lib/swift-rt.metajs.
func swGenFields(v interface{}, adopt func(interface{}, string) interface{}) {
	o, cls, _ := swGenClass(v)
	if o == nil {
		return
	}
	if _, has := o.props["__targs"]; !has {
		return
	}
	fs := swStrArray(cls.props["__gfields"])
	ts := swStrArray(cls.props["__gftys"])
	for i, f := range fs {
		if fv, has := o.props[f]; has && i < len(ts) {
			o.set(f, adopt(fv, swGenSub(o, ts[i])))
		}
	}
}

// swGenArgsTxt is the `<Double, Int>` a generic struct prints after its name:
// its type arguments, else the ones its stored properties declared as a bare
// parameter infer from their values; "" when one is unknown. Twins: swGenArgsTxt
// in languages/swift-interpreter.abnf and lib/swift-rt.metajs.
func (rt *jsrt) swGenArgsTxt(v interface{}) string {
	o, cls, tg := swGenClass(v)
	if o == nil {
		return ""
	}
	names := swTySplit(tg, ',')
	var given []string
	if ta, has := o.props["__targs"].(string); has {
		given = swTySplit(ta, ',')
	}
	fs := swStrArray(cls.props["__gfields"])
	ts := swStrArray(cls.props["__gftys"])
	out := make([]string, len(names))
	for k, n := range names {
		a := ""
		if given != nil {
			if k < len(given) {
				a = swTyCanon(given[k])
			}
		} else {
			for i := 0; i < len(fs) && i < len(ts) && a == ""; i++ {
				if fv, has := o.props[fs[i]]; has && ts[i] == n {
					a = rt.swValTyName(fv)
				}
			}
		}
		if a == "" {
			return ""
		}
		out[k] = a
	}
	return "<" + strings.Join(out, ", ") + ">"
}

// swValTyName is the type name of a value as a generic argument, "" when the
// value cannot say.
func (rt *jsrt) swValTyName(x interface{}) string {
	switch t := x.(type) {
	case bool, string, float64, jsGInt, jsJFlo:
		return rt.swTypeOf(x)
	case *jsArray:
		if len(t.elems) == 0 {
			return ""
		}
		if e := rt.swValTyName(t.elems[0]); e != "" {
			return "Array<" + e + ">"
		}
		return ""
	case *jsObject:
		cls, ok := t.props["__class"].(*jsObject)
		if !ok {
			return ""
		}
		name, _ := cls.props["__name"].(string)
		if _, gen := cls.props["__tgens"].(string); !gen {
			return name
		}
		if g := rt.swGenArgsTxt(t); g != "" {
			return name + g
		}
	}
	return ""
}

// swTyCanon spells a type written with the sugar as type(of:) and print name it:
// [T] is Array<T>, [K: V] Dictionary<K, V> and T? Optional<T>, all the way down.
// Twins: swTyCanon in both swift grammars and lib/swift-rt.metajs.
func swTyCanon(t string) string {
	n := len(t)
	if n > 1 && t[n-1] == '?' {
		return "Optional<" + swTyCanon(t[:n-1]) + ">"
	}
	if n > 1 && t[0] == '[' && t[n-1] == ']' {
		if kv := swTySplit(t[1:n-1], ':'); len(kv) == 2 {
			return "Dictionary<" + swTyCanon(kv[0]) + ", " + swTyCanon(kv[1]) + ">"
		}
		return "Array<" + swTyCanon(t[1:n-1]) + ">"
	}
	if lt := strings.IndexByte(t, '<'); lt > 0 && t[n-1] == '>' {
		parts := swTySplit(t[lt+1:n-1], ',')
		for i := range parts {
			parts[i] = swTyCanon(parts[i])
		}
		return t[:lt+1] + strings.Join(parts, ", ") + ">"
	}
	return t
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

// swGetterFn answers the value of the lazy or computed property `name` of an
// instance whose member table has no method of that name, when that value is a
// callable: `c.bump()` on `lazy var bump: () -> Int`. nil otherwise. Twins: the
// js_mcall instance arm in languages/lib/swift-rt.metajs, mcall in
// languages/swift-interpreter.abnf.
func swGetterFn(rt *jsrt, v interface{}, name string) interface{} {
	o, ok := v.(*jsObject)
	if !ok {
		return nil
	}
	if _, hasCls := o.props["__class"]; !hasCls {
		return nil
	}
	if _, found := swFindMember(v, name); found {
		return nil
	}
	g, found := swFindMember(v, "__get_"+name)
	if !found || !isCallable(g) {
		return nil
	}
	if fv := rt.call(g, jsUndef, []interface{}{v}); isCallable(fv) {
		return fv
	}
	return nil
}

// swStaticFn answers the static method `name` a TYPE DESCRIPTOR (an object with
// no __class) declares or inherits through __super, stored as __static_<name>.
// It is called with the descriptor it was reached through as its receiver, so an
// inherited `class func` sees the subclass as self. Twins: js_mcall in
// languages/lib/swift-rt.metajs, mcall in languages/swift-interpreter.abnf.
func swStaticFn(v interface{}, name string) interface{} {
	o, ok := v.(*jsObject)
	if !ok {
		return nil
	}
	if _, hasCls := o.props["__class"]; hasCls {
		return nil
	}
	key := "__static_" + name
	for c, n := o, 0; c != nil && n < 64; n++ {
		if f, ok := c.props[key]; ok && isCallable(f) {
			return f
		}
		c, _ = c.props["__super"].(*jsObject)
	}
	return nil
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
		// The `>` of a function type's `->` closes nothing.
		case c == '-' && i+1 < len(s) && s[i+1] == '>':
			i++
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
// they compare element by element (js_seq compares them by identity). An enum
// case compares by name plus associated
// values (which the compiler also encodes in __eqkey; this path agrees with it
// and additionally handles the shapes __eqkey cannot round-trip).
// A Unicode.Scalar: {__usc, value, isASCII}. Twins: swMkScalar / swScalarDebug /
// swScalarText in languages/swift-interpreter.abnf, whose comment has the rule,
// and in lib/swift-rt.metajs.
func swMkScalar(s string) *jsObject {
	cp := 0
	u16 := strUnits(s)
	if len(u16) > 1 && isHighSurr(rune(u16[0])) && isLowSurr(rune(u16[1])) {
		cp = int(surrPair(rune(u16[0]), rune(u16[1])))
	} else if len(u16) > 0 {
		cp = int(u16[0])
	}
	o := newJSObject()
	o.set("__usc", s)
	o.set("value", float64(cp))
	o.set("isASCII", cp < 128)
	return o
}

func swScalarDebug(s string) string {
	o := swMkScalar(s)
	cp := int(o.props["value"].(float64))
	e := s
	switch {
	case s == "\\":
		e = "\\\\"
	case s == "'":
		e = "\\'"
	case s == "\"":
		e = "\\\""
	case cp >= 32 && cp < 127:
	case cp == 0:
		e = "\\0"
	case cp == 10:
		e = "\\n"
	case cp == 13:
		e = "\\r"
	case cp == 9:
		e = "\\t"
	case cp < 128:
		e = fmt.Sprintf("\\u{%02X}", cp)
	case cp < 65536:
		e = fmt.Sprintf("\\u{%04X}", cp)
	default:
		e = fmt.Sprintf("\\u{%08X}", cp)
	}
	return "\"" + e + "\""
}

func swIsScalar(v interface{}) bool {
	o, ok := v.(*jsObject)
	if !ok {
		return false
	}
	_, ok = o.props["__usc"]
	return ok
}

func swScalarText(v interface{}) (string, bool) {
	switch t := v.(type) {
	case string:
		return t, true
	case *jsObject:
		s, ok := t.props["__usc"].(string)
		return s, ok
	}
	return "", false
}

// swIsValueKey: a Dictionary key that is an object or an array, i.e. one whose
// == is not identity (a Set, Array, Dictionary, struct or enum case).
func swIsValueKey(v interface{}) bool {
	switch v.(type) {
	case *jsObject, *jsArray:
		return true
	}
	return false
}

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
	// A Unicode.Scalar against a scalar or a String literal: by scalar.
	if swIsScalar(l) || swIsScalar(r) {
		lt, lok := swScalarText(l)
		rtx, rok := swScalarText(r)
		return lok && rok && lt == rtx
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
	// A Set meets an Array only where the array is a LITERAL its context typed as a
	// Set (`[ss].contains([1])`, `ss == [1, 2]`): Swift has no Set-to-Array ==, so
	// the array compares as the Set it denotes. Twins: swEqual in
	// languages/lib/swift-rt.metajs, swEqD in languages/swift-interpreter.abnf.
	if ra, ok := r.(*jsArray); ok {
		if _, isSet := swSetElems(l); isSet {
			return rt.swEqualDepth(l, rt.swSetOf(ra), depth+1)
		}
	}
	if la, ok := l.(*jsArray); ok {
		ra, ok := r.(*jsArray)
		if !ok {
			if _, isSet := swSetElems(r); isSet {
				return rt.swEqualDepth(rt.swSetOf(la), r, depth+1)
			}
			return false
		}
		if len(la.elems) != len(ra.elems) {
			return false
		}
		for i := range la.elems {
			if !rt.swEqualDepth(la.elems[i], ra.elems[i], depth+1) {
				return false
			}
		}
		return true
	}
	if li, ok := swCrIx(l); ok {
		ri, ok := swCrIx(r)
		return ok && rt.swCrIxCmp(li, ri) == 0
	}
	if _, ok := swCrIx(r); ok {
		return false
	}
	// Two RANGES compare by their bounds - a Range is Equatable in Swift and
	// `0..<3 == 0..<3` is true there, not an object IDENTITY test.
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
		// A type's own `==` decides for its values, inside a collection, a tuple
		// or a Set too. Twins: swEqD in languages/swift-interpreter.abnf, swEqual
		// in languages/lib/swift-rt.metajs.
		_, lc := lo.props["__class"]
		_, rc := ro.props["__class"]
		if lc || rc {
			f, ok := swFindMember(l, "==")
			if !ok {
				f, ok = swFindMember(r, "==")
			}
			if ok && isCallable(f) {
				return rt.truthy(rt.call(f, jsUndef, []interface{}{l, r}))
			}
		}
		// Two Sets are equal when they hold the same members, in any order.
		if lel, ok := swSetElems(lo); ok {
			rel, ok := swSetElems(ro)
			return ok && len(lel.elems) == len(rel.elems) && len(rt.swSetSel(lel.elems, rel.elems, false)) == 0
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
		// A struct's synthesized ==: same type, every stored property equal.
		// Twin of swStructEq in languages/swift-interpreter.abnf.
		if cls, ok := lo.props["__class"].(*jsObject); ok && cls.props["__struct"] == true && cls == ro.props["__class"] {
			for _, f := range swStrArray(cls.props["__fields"]) {
				if !rt.swEqualDepth(lo.props[f], ro.props[f], depth+1) {
					return false
				}
			}
			return true
		}
		return false
	}
	return rt.strictEq(l, r)
}

// ----- The Sequence/Collection members ----------------------------------------
// The Go twin of swSeqMethod in languages/lib/swift-rt.metajs and arraySeqMethod
// in languages/swift-interpreter.abnf. Every member here is checked against
// swiftc 6.1.2; the ones that cannot be matched exactly are recorded at their
// arms in the two files above.
//
// It sits in FRONT of abnf/jsrt.go's shared list table, whose refusal wording
// ("unknown list method 'sorted'") is not the interpreter's "unknown Array
// method: sorted". Every Array receiver lands here and no swift value can reach
// the shared wording.
//
// Nothing here rebinds a SHARED extern (manual chapter 4's rxExtraExterns trap) -
// it is reached only from m["js_swmcall"], which is swift's own name.

// swSortLess is Swift's <, over the values a Swift Array can hold: a UInt64 above
// 2^63 is not negative, a Double is a box, and a String pair compares as text.
// A user type's own `static func <` WINS (`[P(n: 3), P(n: 1)].sorted()` is
// [1, 3]): Comparable is how
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

// swUnordered is swRel's answer for a NaN operand, on which every ordering is
// false.
const swUnordered = 2

// swRel is Swift's ordering of two values: -1, 0, 1, or swUnordered. A box has
// to go through giCmp rather than the shared jsCompare, because a UInt64 above
// 2^63 reads as a NEGATIVE int64 there, so UInt64.max < 0 would hold. Twin of
// swRel in languages/lib/swift-rt.metajs.
func (rt *jsrt) swRel(l, r interface{}) int {
	// A float box is an object to the shared comparison, so it has to be
	// unboxed first: `1.5 < 2.0` between two boxes is not a numeric
	// comparison at all.
	if swIsFlo(l) || swIsFlo(r) {
		x, y := swNum(rt, l), swNum(rt, r)
		// A Float operand makes the other side a Float too, so `16777217 < f`
		// compares two binary32 values - swift will only ever have put a
		// literal there.
		if swIsF32(l) || swIsF32(r) {
			x, y = jvmFround(x), jvmFround(y)
		}
		switch {
		case x < y:
			return -1
		case x > y:
			return 1
		case x != y:
			return swUnordered
		}
		return 0
	}
	if giIsInt(l) || giIsInt(r) {
		if giIsIntegral(l) && giIsIntegral(r) {
			return rt.giCmp(l, r)
		}
	}
	if li, ok := swCrIx(l); ok {
		if ri, ok := swCrIx(r); ok {
			return rt.swCrIxCmp(li, ri)
		}
	}
	if lo, ok := l.(*jsObject); ok {
		if ro, ok := r.(*jsObject); ok {
			if c, ok := rt.swTupRel(lo, ro); ok {
				return c
			}
		}
	}
	// Two Strings order by their NFC scalars (swcStrCmp).
	if ls, ok := l.(string); ok {
		if rs, ok := r.(string); ok {
			return swcStrCmp(ls, rs)
		}
	}
	return rt.jsCompare(l, r)
}

// swTupRel orders two tuples lexicographically: the first unequal pair decides
// (the stdlib's tuple comparison operators), swUnordered when that pair is.
// ok is false unless both are tuples. Twins: swTupRel in
// languages/swift-interpreter.abnf and lib/swift-rt.metajs.
func (rt *jsrt) swTupRel(l, r *jsObject) (int, bool) {
	ln, lok := swNumProp(l, "__tuple")
	rn, rok := swNumProp(r, "__tuple")
	if !lok || !rok {
		return 0, false
	}
	for i := 0; i < ln && i < rn; i++ {
		k := strconv.Itoa(i)
		a, b := l.props[k], r.props[k]
		if rt.swEqualDepth(a, b, 0) {
			continue
		}
		c := rt.swRel(a, b)
		if c == 0 {
			return swUnordered, true
		}
		return c, true
	}
	return 0, true
}

// swRangeLess is `a < b` (`a <= b` when orEq) between two range bounds.
func (rt *jsrt) swRangeLess(a, b interface{}, orEq bool) bool {
	c := rt.swRel(a, b)
	return c != swUnordered && (c < 0 || orEq && c == 0)
}

// swRangeAt is the i-th element of a range starting at lo, of lo's own integer
// type. Twins: rangeAt in languages/swift-interpreter.abnf and swRangeAt in
// languages/lib/swift-rt.metajs.
func swRangeAt(rt *jsrt, lo interface{}, i int) interface{} {
	if g, ok := lo.(jsGInt); ok {
		return giNorm(g.v+int64(i), g.w, g.u)
	}
	return swNum(rt, lo) + float64(i)
}

// swRangeHas is a range's bounds test, under the bound type's own ordering; a
// missing bound admits everything on its side.
func (rt *jsrt) swRangeHas(r *jsObject, v interface{}) bool {
	if lo := r.props["lo"]; !isNullish(lo) && !rt.swRangeLess(lo, v, true) {
		return false
	}
	hi := r.props["hi"]
	return isNullish(hi) || rt.swRangeLess(v, hi, r.props["closed"] == true)
}

// swRangeEmpty is Range.isEmpty: a half-open range with equal bounds. A
// ClosedRange always holds its lowerBound.
func (rt *jsrt) swRangeEmpty(r *jsObject) bool {
	return r.props["closed"] != true && rt.swRel(r.props["lo"], r.props["hi"]) == 0
}

// swRangeMethod answers the members a range answers from its bounds alone, so
// they hold for a Double or a String range too: contains, clamped(to:) and
// overlaps, as the Swift standard library spells them (Range.swift,
// ClosedRange.swift). Twins: rangeMethod in languages/swift-interpreter.abnf and
// swRangeMethod in languages/lib/swift-rt.metajs.
func (rt *jsrt) swRangeMethod(r *jsObject, name string, args []interface{}) (interface{}, bool) {
	if len(args) != 1 {
		return nil, false
	}
	if name == "contains" && !isCallable(args[0]) {
		return rt.swRangeHas(r, args[0]), true
	}
	o, ok := args[0].(*jsObject)
	if !ok || !swIsRange(o) {
		return nil, false
	}
	lo, hi, olo, ohi := r.props["lo"], r.props["hi"], o.props["lo"], o.props["hi"]
	switch name {
	case "clamped":
		nlo, nhi := lo, hi
		if rt.swRangeLess(lo, olo, false) {
			nlo = olo
		} else if rt.swRangeLess(ohi, lo, false) {
			nlo = ohi
		}
		if rt.swRangeLess(ohi, hi, false) {
			nhi = ohi
		} else if rt.swRangeLess(hi, olo, false) {
			nhi = olo
		}
		c := newJSObject()
		c.set("__range", true)
		c.set("lo", nlo)
		c.set("hi", nhi)
		c.set("closed", r.props["closed"])
		return c, true
	case "overlaps":
		// Disjoint when one range ends before the other starts - at its
		// upperBound itself for a half-open range - or either is empty.
		return !(rt.swRangeLess(ohi, lo, o.props["closed"] != true) ||
			rt.swRangeLess(hi, olo, r.props["closed"] != true) ||
			rt.swRangeEmpty(r) || rt.swRangeEmpty(o)), true
	}
	return nil, false
}

// swIntMethod is the BinaryInteger members on an Int of any width (ok false
// for any other receiver or name); the answer keeps the receiver's type, and
// map/flatMap are Optional's. Twin of swIntMethod in
// languages/swift-interpreter.abnf and lib/swift-rt.metajs.
func (rt *jsrt) swIntMethod(v interface{}, name string, args []interface{}) (interface{}, bool) {
	var wd uint8 = 64
	uns := false
	switch t := v.(type) {
	case jsGInt:
		wd, uns = t.w, t.u
	case float64:
	default:
		return nil, false
	}
	switch name {
	case "signum":
		c := rt.giCmp(v, 0.0)
		return giNorm(int64(c), wd, uns), true
	case "isMultiple":
		if giVal(rt, args[0]) == 0 {
			return rt.giCmp(v, 0.0) == 0, true
		}
		return giVal(rt, rt.giArith("%", v, args[0])) == 0, true
	case "quotientAndRemainder":
		if giVal(rt, args[0]) == 0 {
			rt.swAbort("Fatal error: Division by zero")
		}
		q, r := rt.giArith("/", v, args[0]), rt.giArith("%", v, args[0])
		t := newJSObject()
		t.set("__tuple", float64(2))
		t.set("__tlabels", &jsArray{elems: []interface{}{"quotient", "remainder"}})
		t.set("0", q)
		t.set("quotient", q)
		t.set("1", r)
		t.set("remainder", r)
		return t, true
	case "map", "flatMap":
		return rt.call(args[0], jsUndef, []interface{}{v}), true
	}
	return nil, false
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

// A ClosedRange's Index is Swift's enum `inRange(Bound)` / `pastEnd`, held as an
// object {__crix: bound type name, v: bound} with no `v` for pastEnd; a
// half-open Range<Int> is its own indices. An index stands for a POSITION (its
// offset from lowerBound, the count for the end), and the moves, distance and
// ordering are position arithmetic. Twins: swCrIx* in
// languages/swift-interpreter.abnf and lib/swift-rt.metajs.
func swCrIx(v interface{}) (*jsObject, bool) {
	o, ok := v.(*jsObject)
	if !ok {
		return nil, false
	}
	_, has := o.props["__crix"]
	return o, has
}

func (rt *jsrt) swRangeLen(r *jsObject) int {
	n := int(swNum(rt, r.props["hi"]) - swNum(rt, r.props["lo"]))
	if r.props["closed"] == true {
		n++
	}
	if n < 0 {
		return 0
	}
	return n
}

func (rt *jsrt) swCrIxAt(r *jsObject, pos int) interface{} {
	if r.props["closed"] != true {
		return swRangeAt(rt, r.props["lo"], pos)
	}
	o := newJSObject()
	o.set("__crix", rt.swTypeOf(r.props["lo"]))
	if pos != rt.swRangeLen(r) {
		o.set("v", swRangeAt(rt, r.props["lo"], pos))
	}
	return o
}

func (rt *jsrt) swCrIxPos(r *jsObject, i interface{}) int {
	if o, ok := swCrIx(i); ok {
		v, has := o.props["v"]
		if !has {
			return rt.swRangeLen(r)
		}
		i = v
	}
	return int(swNum(rt, i) - swNum(rt, r.props["lo"]))
}

func swIsIdxMoveName(n string) bool {
	return n == "$idxAfter" || n == "$idxBefore" || n == "$idxOffset" || n == "$distance"
}

func (rt *jsrt) swCrIxMove(r *jsObject, n string, args []interface{}) interface{} {
	p0 := rt.swCrIxPos(r, args[0])
	switch n {
	case "$idxAfter":
		return rt.swCrIxAt(r, p0+1)
	case "$idxBefore":
		return rt.swCrIxAt(r, p0-1)
	case "$idxOffset":
		return rt.swCrIxAt(r, p0+int(swNum(rt, args[1])))
	}
	return float64(rt.swCrIxPos(r, args[1]) - p0)
}

// swCrIxCmp is -1, 0 or 1; pastEnd follows every inRange.
func (rt *jsrt) swCrIxCmp(a, b *jsObject) int {
	av, ah := a.props["v"]
	bv, bh := b.props["v"]
	if !ah || !bh {
		return swB2i(!ah) - swB2i(!bh)
	}
	if rt.swRangeLess(av, bv, false) {
		return -1
	}
	if rt.swRangeLess(bv, av, false) {
		return 1
	}
	return 0
}

func swB2i(b bool) int {
	if b {
		return 1
	}
	return 0
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
	case "$appendSeq":
		// append(contentsOf:): the emitter hands over the elements as an array
		// (rtLabelMeth in languages/swift-to-llvm-ir.abnf).
		if xs, ok := args[0].(*jsArray); ok {
			o.elems = append(o.elems, xs.elems...)
		}
		return jsUndef, true
	case "removeSubrange", "replaceSubrange":
		// In place; a replacement that is not an array is one element (a String
		// receiver's text, joined back by rtStrMut). Twins: arrayMethod in
		// languages/swift-interpreter.abnf, swSeqMethod in lib/swift-rt.metajs.
		r, _ := args[0].(*jsObject)
		if r == nil {
			rt.fail("%s needs a range", name)
		}
		lo, hi := rt.swSliceBounds(r, n)
		var ins []interface{}
		if name == "replaceSubrange" {
			if xs, ok := args[1].(*jsArray); ok {
				ins = xs.elems
			} else {
				ins = []interface{}{args[1]}
			}
		}
		out := make([]interface{}, 0, n-(hi-lo)+len(ins))
		out = append(out, o.elems[:lo]...)
		out = append(out, ins...)
		o.elems = append(out, o.elems[hi:]...)
		return jsUndef, true
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
	case "removeLast":
		// removeLast() answers the element; removeLast(k) answers Void. Twin of
		// arrayMethod in languages/swift-interpreter.abnf.
		if len(args) == 0 {
			if n == 0 {
				rt.swAbort("Fatal error: Can't remove last element from an empty collection")
			}
			l := o.elems[n-1]
			o.elems = o.elems[:n-1]
			return l, true
		}
		k := int(swNum(rt, args[0]))
		if k < 0 || k > n {
			rt.swAbort("Fatal error: Can't remove more items from a collection than it contains")
		}
		o.elems = o.elems[:n-k]
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
	case "randomElement":
		if n == 0 {
			return jsNull, true
		}
		return o.elems[rnd.below(n)], true
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
			// Sequence.min/max(by:) replace the running answer only when
			// the next element orders strictly before / after it, so a tie
			// (or a NaN) keeps the EARLIER element.
			if name == "min" {
				if less(o.elems[i], best) {
					best = o.elems[i]
				}
			} else if less(best, o.elems[i]) {
				best = o.elems[i]
			}
		}
		return best, true
	case "prefix", "drop":
		// prefix(while:) / drop(while:): the leading run the predicate holds
		// for, and what follows it.
		if len(args) > 0 && isCallable(args[0]) {
			run := 0
			for run < n && rt.truthy(rt.call(args[0], jsUndef, []interface{}{o.elems[run]})) {
				run++
			}
			if name == "prefix" {
				return &jsArray{elems: append([]interface{}{}, o.elems[:run]...)}, true
			}
			return &jsArray{elems: append([]interface{}{}, o.elems[run:]...)}, true
		}
		if name == "drop" {
			return nil, false
		}
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
	case "first", "last":
		// first(where:) / last(where:); the property forms never reach here.
		if len(args) == 0 || !isCallable(args[0]) {
			return nil, false
		}
		for i := 0; i < n; i++ {
			e := o.elems[i]
			if name == "last" {
				e = o.elems[n-1-i]
			}
			if rt.truthy(rt.call(args[0], jsUndef, []interface{}{e})) {
				return e, true
			}
		}
		return jsNull, true
	case "split":
		return rt.swSplitSeq(o.elems, args), true
	case "starts", "elementsEqual":
		// Element by element with ==.
		var other []interface{}
		if len(args) > 0 {
			if oa, ok := args[0].(*jsArray); ok {
				other = oa.elems
			} else if els, ok := swSetElems(args[0]); ok {
				other = els.elems
			}
		}
		if len(other) > n || (name == "elementsEqual" && len(other) != n) {
			return false, true
		}
		for i := range other {
			if !rt.swEqual(o.elems[i], other[i]) {
				return false, true
			}
		}
		return true, true
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
		// contains(_:) is a value test by Swift's == and contains(where:) a
		// PREDICATE; the label does not reach here, so the closure argument
		// tells them apart. Twins: arrayMethod in languages/swift-interpreter.abnf
		// and swSeqMethod in lib/swift-rt.metajs.
		if len(args) == 0 {
			return nil, false
		}
		for _, e := range o.elems {
			if isCallable(args[0]) {
				if rt.truthy(rt.call(args[0], jsUndef, []interface{}{e})) {
					return true, true
				}
			} else if rt.swEqual(e, args[0]) {
				return true, true
			}
		}
		return false, true
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

// swSplitSeq is Collection.split, step for step; the arguments are told apart by
// type because the labels do not reach here. Twins: swSplitSeq in
// languages/swift-interpreter.abnf and lib/swift-rt.metajs.
func (rt *jsrt) swSplitSeq(t []interface{}, args []interface{}) *jsArray {
	var pred interface{}
	var sep interface{} = jsUndef
	maxS := -1
	omit := true
	for _, a := range args {
		if isCallable(a) {
			pred = a
		}
	}
	for j, a := range args {
		if isCallable(a) {
			continue
		}
		if b, ok := a.(bool); ok {
			omit = b
		} else if j == 0 && pred == nil {
			sep = a
		} else {
			maxS = int(swNum(rt, a))
		}
	}
	n := len(t)
	piece := func(a, b int) interface{} { return &jsArray{elems: append([]interface{}{}, t[a:b]...)} }
	res := []interface{}{}
	if maxS == 0 || n == 0 {
		if n > 0 || !omit {
			res = append(res, piece(0, n))
		}
		return &jsArray{elems: res}
	}
	start, end := 0, 0
	for end < n {
		var isSep bool
		if pred != nil {
			isSep = rt.truthy(rt.call(pred, jsUndef, []interface{}{t[end]}))
		} else {
			isSep = rt.swEqual(t[end], sep)
		}
		if isSep {
			did := false
			if !(start == end && omit) {
				res = append(res, piece(start, end))
				did = true
			}
			end++
			start = end
			if did && len(res) == maxS {
				break
			}
			continue
		}
		end++
	}
	if start != n || !omit {
		res = append(res, piece(start, n))
	}
	return &jsArray{elems: res}
}

// swStrMethod answers the String members that take a String argument and compare
// text, with no Character walk. Twins: the String arm of mcall in
// languages/swift-interpreter.abnf and of js_mcall in languages/lib/swift-rt.metajs.
func swStrMethod(rt *jsrt, s, name string, args []interface{}) (interface{}, bool) {
	if len(args) == 0 {
		return nil, false
	}
	p, isStr := args[0].(string)
	if !isStr {
		return nil, false
	}
	switch name {
	case "hasPrefix", "starts":
		return swcHasPrefix(s, p), true
	case "hasSuffix":
		return swcHasSuffix(s, p), true
	case "contains":
		return swcContains(s, p), true
	case "trimmingPrefix":
		return swcTrimPrefix(s, p), true
	case "replacing":
		if len(args) > 1 {
			if r, ok := args[1].(string); ok {
				n := -1
				if len(args) > 2 {
					n = int(swNum(rt, args[2]))
				}
				return swcReplacing(s, p, r, n), true
			}
		}
	}
	return nil, false
}

// swSharedListMethod names the members abnf/jsrt.go's shared list table already
// answers for an Array receiver, so they keep reaching it - the swift-side table
// above only ADDS members and takes over the refusal. Keeping the list explicit is
// what makes "unknown Array method: X" reachable without re-implementing the eleven
// the shared table already has.
// swIsLazyName: the members a lazy sequence answers without declaring them - its
// iterator's and the Sequence members of swIsSeqMember in
// languages/swift-interpreter.abnf, which is the list to keep this in step with.
func swIsLazyName(name string) bool {
	switch name {
	case "makeIterator", "next", "map", "filter", "sorted", "reversed", "enumerated",
		"allSatisfy", "compactMap", "flatMap", "reduce", "prefix", "suffix", "dropFirst",
		"dropLast", "min", "max", "forEach", "shuffled", "randomElement", "first", "last", "count",
		"contains", "split", "drop", "firstIndex", "lastIndex", "elementsEqual":
		return true
	}
	return false
}

func swSharedListMethod(name string) bool {
	switch name {
	case "add", "append", "pop", "size", "get", "contains", "map",
		"filter", "forEach", "count", "any", "insert", "popLast":
		return true
	}
	return false
}

// swAbort is HOW A SWIFT PROGRAM DIES. rt.fail prefixes "js runtime error: ";
// a message ALREADY in swiftc's own
// wording is printed bare on stderr and exits 133, a trap's status (see "How a
// Swift program DIES" in languages/swift-interpreter.abnf). The pattern is
// js_jvtop in abnf/jsrtjvm.go, and the
// twins are swAbort in languages/lib/swift-rt.metajs and the fail override in
// languages/swift-interpreter.abnf. An engine-level complaint no toolchain would
// say ("unknown Array method: x") still goes through rt.fail and keeps its prefix.
func (rt *jsrt) swAbort(format string, args ...interface{}) {
	fmt.Fprint(warnWriter, wtf8Clean(fmt.Sprintf(format, args...)+"\n"))
	os.Exit(133)
}
