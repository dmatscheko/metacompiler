package abnf

// jsrtgolang.go -- Go's VALUE RENDERING (fmt's %v) for the compiler grammar
// (languages/go-to-llvm-ir.abnf).
//
// go-to-llvm-ir.abnf used to bind fmt.Println / fmt.Print / fmt.Sprint straight
// to the shared `println` host function, which hands the RAW runtime value to
// Go's own %v. That is recurring shape 1 in docs/abnf-dialect-gotchas.md, and in
// Go it was unusually loud, because this grammar's value model puts a class
// descriptor on every struct and a {a,o,n,c} header on every slice:
//
//	map[string]int{"a": 1}  ->  map[__dict:true keys:[a] vals:[1] zero:0]
//	P{1}                    ->  map[X:1 __class:map[__isclass:true ...]]
//	[]int{1, 2} (a header)  ->  map[__class:map[__name:$sl ...] a:[1 2] c:2 n:2 o:0]
//
// where real go 1.26.5 prints `map[a:1]`, `{1}` and `[1 2]`. The interpreter half
// had its own renderer (`gstr`) which was already close, so this file is the Go
// twin of that function and the two MUST stay in step - ./test.sh --cross diffs
// them.
//
// Registration is additive: an init() appends a registrar to rxExtraExterns, so
// nothing here runs unless a program calls one of the three js_go* externs, and
// only the Go compiler grammar emits them. abnf/jsrt.go, printArgs and every
// existing extern are untouched.
//
// The three externs, each taking the callee's ARGUMENT ARRAY plus a mode number:
//
//	js_goprint(args, mode)   0 fmt.Println   2 builtin println   4 fmt.Printf
//	js_gowrite(args, mode)   1 fmt.Print     3 builtin print
//	js_gostr(args, mode)     1 fmt.Sprint    4 fmt.Sprintf       5 render ONE value
//
// The spacing rules are Go's own and they differ per entry point: Println puts a
// space between every pair of operands, Print only between two operands NEITHER
// of which is a string, the builtin println always, the builtin print never.
//
// Deliberate simplifications, all shared with the interpreter half so the two
// agree byte for byte:
//
//   - A POINTER prints as `&` followed by the pointee. Real Go prints a hex
//     ADDRESS for a pointer to a scalar (`0x14000112008`) and only uses `&{...}`
//     / `&[...]` / `&map[...]` for a pointer to a struct, array, slice or map. An
//     address is not reproducible between two runs, so this runtime prints the
//     pointee everywhere it has one.
//   - A pointer to a STRUCT is indistinguishable from the struct: in both halves
//     a struct value IS a reference, so `&P{1}` and `P{1}` are the same runtime
//     object and both print `{1}` where go prints `&{1}` for the first. Fixing
//     that needs a real pointer representation for struct pointers, which would
//     change field access, method dispatch, assignment, equality and type
//     switches in both halves - see the report in the commit that added this file.
//   - A nil MAP prints `<nil>`; go prints `map[]`. A nil map is the null value in
//     this model, with nothing to tell it from a nil interface. (A nil SLICE does
//     print `[]`, and a nil pointer / nil interface `<nil>`, both as go does.)
//   - Only a String() method declared on the type ITSELF is a Stringer; a String()
//     promoted from an embedded struct is not found. Go's method set rules are not
//     modelled either, so a String() with a POINTER receiver is used for the value
//     form too (go prints `{2}` there).
//   - fmt's format verbs are limited to %v %d %s %t %q %c and %%. Width, precision,
//     flags (%-5d, %.2f) and the remaining verbs (%f %g %e %x %o %b %p %T %U) are
//     not implemented; an unhandled verb renders as %!<verb>(<value>) and a missing
//     operand as %!<verb>(MISSING), which is close to but not identical with go's
//     own error text (go writes %!z(int=3) where this writes %!z(3)). Surplus
//     operands are ignored where go appends %!(EXTRA ...). %q escapes the
//     backslash, the delimiter and the seven letter escapes; a control character
//     go would spell with a hex or unicode escape is left verbatim.

import (
	"fmt"
	"strconv"
	"strings"
)

func init() {
	rxExtraExterns = append(rxExtraExterns, func(rt *jsrt, m map[string]func(args []uint64) uint64) {
		u := rt.unwrap

		m["js_goprint"] = func(a []uint64) uint64 {
			out, nl := rt.gopCompose(u(a[0]), int(rt.toNumber(u(a[1]))))
			if nl {
				out += "\n"
			}
			fmt.Fprint(outWriter, wtf8Clean(out))
			return 0
		}
		m["js_gowrite"] = func(a []uint64) uint64 {
			out, nl := rt.gopCompose(u(a[0]), int(rt.toNumber(u(a[1]))))
			if nl {
				out += "\n"
			}
			fmt.Fprint(outWriter, wtf8Clean(out))
			return 0
		}
		m["js_gostr"] = func(a []uint64) uint64 {
			mode := int(rt.toNumber(u(a[1])))
			if mode == gopOne {
				return rt.wrapStr(rt.gopStr(u(a[0])))
			}
			out, _ := rt.gopCompose(u(a[0]), mode)
			return rt.wrapStr(out)
		}
	})
}

// The modes the two grammars pass. They are duplicated as plain numbers in
// go-to-llvm-ir.abnf (declGoPrint) and in the interpreter's gCompose; keep the
// three lists in step.
const (
	gopLn     = 0 // fmt.Println      space between every pair,      newline
	gopPr     = 1 // fmt.Print/Sprint space between two non-strings, no newline
	gopBuiltn = 2 // builtin println  space between every pair,      newline
	gopBuiltp = 3 // builtin print    no space at all,               no newline
	gopFmtf   = 4 // fmt.Printf/Sprintf  args[0] is the format,      no newline
	gopOne    = 5 // js_gostr only: render exactly one value
)

// gopCompose builds the text of one print call and reports whether a newline is
// appended - the entry points differ in both spacing and terminator (see above).
func (rt *jsrt) gopCompose(argsV interface{}, mode int) (string, bool) {
	arr, ok := argsV.(*jsArray)
	if !ok {
		rt.fail("a Go print extern needs an argument array")
	}
	switch mode {
	case gopFmtf:
		if len(arr.elems) == 0 {
			return "", false
		}
		return rt.gopFormat(rt.gopStr(arr.elems[0]), arr.elems[1:]), false
	case gopBuiltp:
		out := ""
		for _, e := range arr.elems {
			out += rt.gopStr(e)
		}
		return out, false
	case gopLn, gopBuiltn:
		parts := make([]string, len(arr.elems))
		for i, e := range arr.elems {
			parts[i] = rt.gopStr(e)
		}
		return strings.Join(parts, " "), true
	}
	// gopPr: Go puts a space between two operands when NEITHER is a string.
	out := ""
	prevStr := true
	for i, e := range arr.elems {
		_, isStr := e.(string)
		if i > 0 && !prevStr && !isStr {
			out += " "
		}
		out += rt.gopStr(e)
		prevStr = isStr
	}
	return out, false
}

// ----------------------------------------------------------------------------
// Rendering

// gopStr is fmt's %v for one value.
func (rt *jsrt) gopStr(v interface{}) string { return rt.gopRender(v, 0) }

func (rt *jsrt) gopRender(v interface{}, depth int) string {
	if depth > 32 {
		return "..."
	}
	switch t := v.(type) {
	case jsNullT, jsUndefT:
		return "<nil>"
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case jsJFlo:
		return goFloStr(t.f)
	case *jsArray:
		// A Go ARRAY is a plain host array; a Go SLICE is the {a,o,n,c} header
		// handled in gopObj. Both print `[e e e]`, space separated.
		parts := make([]string, len(t.elems))
		for i, e := range t.elems {
			parts[i] = rt.gopRender(e, depth+1)
		}
		return "[" + strings.Join(parts, " ") + "]"
	case *jsObject:
		if s, ok := rt.gopObj(t, depth); ok {
			return s
		}
	}
	return rt.toString(v)
}

// gopObj renders the object shapes the Go grammars build. It reports false for a
// shape it does not know, which falls back to the generic ToString.
func (rt *jsrt) gopObj(o *jsObject, depth int) (string, bool) {
	cls, _ := o.props["__class"].(*jsObject)

	// fmt.Stringer wins over every built-in rendering, exactly as it does in Go.
	// Guarded by depth so a String() that prints its own receiver terminates.
	if cls != nil && depth < 32 {
		if mth, ok := cls.props["String"]; ok && isCallable(mth) {
			return rt.gopRender(rt.call(mth, jsUndef, []interface{}{o}), depth+1), true
		}
	}

	// A map: map[k:v k:v], with the keys SORTED, which is what fmt has done since
	// go 1.12. The value model keeps insertion order, so the order is computed here.
	if keys, vals, ok := dictParts(o); ok {
		order := gopKeyOrder(keys.elems)
		parts := make([]string, 0, len(order))
		for _, i := range order {
			var val interface{} = jsUndef
			if i < len(vals.elems) {
				val = vals.elems[i]
			}
			parts = append(parts, rt.gopRender(keys.elems[i], depth+1)+":"+rt.gopRender(val, depth+1))
		}
		return "map[" + strings.Join(parts, " ") + "]", true
	}

	if cls == nil {
		return "", false
	}
	name, _ := cls.props["__name"].(string)

	// A slice header prints only its own window [o, o+n) of the backing array.
	if name == "$sl" {
		back, _ := o.props["a"].(*jsArray)
		if back == nil {
			return "[]", true
		}
		off := int(rt.toNumber(o.props["o"]))
		n := int(rt.toNumber(o.props["n"]))
		parts := make([]string, 0, n)
		for i := 0; i < n; i++ {
			var e interface{} = jsUndef
			if off+i < len(back.elems) {
				e = back.elems[off+i]
			}
			parts = append(parts, rt.gopRender(e, depth+1))
		}
		return "[" + strings.Join(parts, " ") + "]", true
	}

	// A complex: Go spells it (re+imi), with the sign always written out. The twin
	// is cxStr in go-interpreter.abnf, which concatenates the two JS numbers, so
	// toString is what has to be used here rather than a float rendering.
	if name == "$cx" {
		re := rt.toString(o.props["re"])
		im := rt.toNumber(o.props["im"])
		sign := "+"
		if im < 0 {
			sign = "-"
			im = -im
		}
		return "(" + re + sign + rt.toString(im) + "i)", true
	}

	// A pointer CELL (&n, new(int), &p): the scope that holds the variable plus
	// its name. Printed as & followed by the pointee - see the header note.
	if name == "$ptr" {
		if sc, ok := o.props["o"].(*jsScope); ok {
			key := rt.toString(o.props["k"])
			return "&" + rt.gopRender(rt.scopeGet(sc, key), depth+1), true
		}
		if inner, ok := o.props["o"].(*jsObject); ok {
			key := rt.toString(o.props["k"])
			return "&" + rt.gopRender(inner.props[key], depth+1), true
		}
		return "&<nil>", true
	}

	// A struct instance: its field values in declaration order, space separated.
	if tag, ok := cls.props["__isclass"]; ok && tag == true {
		flds, _ := cls.props["__fields"].(*jsArray)
		if flds == nil {
			return "{}", true
		}
		parts := make([]string, 0, len(flds.elems))
		for _, f := range flds.elems {
			fn, _ := f.(string)
			var val interface{} = jsUndef
			if fv, ok := o.props[fn]; ok {
				val = fv
			}
			parts = append(parts, rt.gopRender(val, depth+1))
		}
		return "{" + strings.Join(parts, " ") + "}", true
	}
	return "", false
}

// gopKeyOrder answers the indices of a map's keys in fmt's print order. fmt sorts
// map keys (go 1.12+): numerically for a numeric key type, bytewise for a string
// key, false before true for a bool. A Go map has ONE key type, so the kind of the
// first key decides; anything else falls back to the rendered text. Insertion sort,
// which keeps the order stable and mirrors the interpreter half's own loop.
func gopKeyOrder(keys []interface{}) []int {
	order := make([]int, len(keys))
	for i := range order {
		order[i] = i
	}
	num := make([]float64, len(keys))
	txt := make([]string, len(keys))
	numeric := len(keys) > 0
	for i, k := range keys {
		switch t := k.(type) {
		case float64:
			num[i] = t
		case jsJFlo:
			num[i] = t.f
		case bool:
			numeric = false
			if t {
				txt[i] = "1"
			} else {
				txt[i] = "0"
			}
		case string:
			numeric = false
			txt[i] = t
		default:
			numeric = false
			txt[i] = fmt.Sprint(k)
		}
	}
	less := func(a, b int) bool {
		if numeric {
			return num[a] < num[b]
		}
		return txt[a] < txt[b]
	}
	for i := 1; i < len(order); i++ {
		j := i
		for j > 0 && less(order[j], order[j-1]) {
			order[j], order[j-1] = order[j-1], order[j]
			j--
		}
	}
	return order
}

// ----------------------------------------------------------------------------
// Format verbs

// gopFormat is fmt.Sprintf over the verb subset documented at the top of the file.
func (rt *jsrt) gopFormat(format string, args []interface{}) string {
	out := ""
	next := 0
	for i := 0; i < len(format); i++ {
		if format[i] != '%' || i+1 >= len(format) {
			out += string(format[i])
			continue
		}
		i++
		verb := format[i]
		if verb == '%' {
			out += "%"
			continue
		}
		if next >= len(args) {
			out += "%!" + string(verb) + "(MISSING)"
			continue
		}
		a := args[next]
		next++
		out += rt.gopVerb(verb, a)
	}
	return out
}

func (rt *jsrt) gopVerb(verb byte, a interface{}) string {
	switch verb {
	case 'v', 's':
		return rt.gopStr(a)
	case 'd':
		return strconv.FormatInt(int64(rt.toNumber(a)), 10)
	case 't':
		if b, ok := a.(bool); ok && b {
			return "true"
		}
		return "false"
	case 'c':
		return string(rune(int64(rt.toNumber(a))))
	case 'q':
		if s, ok := a.(string); ok {
			return `"` + gopQuoteBody(s, '"') + `"`
		}
		return "'" + gopQuoteBody(string(rune(int64(rt.toNumber(a)))), '\'') + "'"
	}
	return "%!" + string(verb) + "(" + rt.gopStr(a) + ")"
}

// gopQuoteBody is the escaping strconv.Quote / strconv.QuoteRune do, minus the \x
// and \u forms: the letter escapes, the backslash and whichever delimiter is in
// force. A control character Go would spell \x00 is left verbatim. The twin is
// gQuoteBody in languages/go-interpreter.abnf; keep the two in step.
func gopQuoteBody(s string, delim byte) string {
	named := map[byte]string{'\n': `\n`, '\t': `\t`, '\r': `\r`, 7: `\a`, 8: `\b`, 12: `\f`, 11: `\v`}
	out := ""
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\\' {
			out += `\\`
		} else if c == delim {
			out += `\` + string(c)
		} else if e, ok := named[c]; ok {
			out += e
		} else {
			out += string(c)
		}
	}
	return out
}

// ----- Widths on the SLOTS a Go program declares -----
//
// The jsGInt box of jsrtint.go carries a width, but a width only exists where the
// program wrote a type down, and a `var` declaration is not the only place that
// happens. Three kinds of slot have a declared type of their own, and a value
// flowing into one of them has to adopt it or the width is simply lost:
//
//	type S struct{ b uint8 }; s.b = 255; s.b++   -> 0    (the field's own type)
//	a := []uint8{255}; a[0]++                    -> 0    (the element type)
//	m := map[string]int8{}; m["k"] = 127; m["k"]++ -> -128
//
// A struct field's type is on the descriptor (__fields / __ftypes), so
// js_gosetfield can read it at the write. A container element has no type text at
// the write site at all - but its ZERO VALUE was built from one, so the box
// sitting in the slot IS the record of the declared element type, and js_gisetat
// reapplies it. Both replace a plain js_set / js_pyset at the same site, so the
// emitted IR keeps one call per store and the change costs nothing where no sized
// type is involved.

// giTyBits is goIntTy of the two Go grammars, in Go: the width and signedness of
// the leading TYPE WORD of a type text, and ok=false for anything that is not a
// sized integer type. Only a type that is NOT a signed 64 bit integer needs a
// box, but int/int64 answer here too - giNorm drops the box for them.
func giTyBits(ty string) (uint8, bool, bool) {
	switch giTyWord(ty) {
	case "int", "int64":
		return 64, false, true
	case "int8":
		return 8, false, true
	case "int16":
		return 16, false, true
	case "int32", "rune":
		return 32, false, true
	case "uint8", "byte":
		return 8, true, true
	case "uint16":
		return 16, true, true
	case "uint32":
		return 32, true, true
	case "uint", "uint64", "uintptr":
		return 64, true, true
	}
	return 0, false, false
}

// giNumeric is true for the two things a width may be applied to. A string, a
// struct, nil - anything else - passes a slot untouched.
func giNumeric(v interface{}) bool {
	switch v.(type) {
	case float64, jsGInt:
		return true
	}
	return false
}

// giAdoptText applies a declared type TEXT to a value. A float64 / float32 slot
// BOXES (jsJFlo), exactly as a `var d float64 = 1` declaration does - without it
// a float64 struct field assigned a plain 1 divides as an integer.
func giAdoptText(rt *jsrt, ty string, v interface{}) interface{} {
	if !giNumeric(v) {
		return v
	}
	if w := giTyWord(ty); w == "float64" || w == "float32" {
		return jsJFlo{f: giFloat(rt, v), sty: floGo}
	}
	wd, un, ok := giTyBits(ty)
	if !ok {
		return v
	}
	return giNorm(giVal(rt, v), wd, un)
}

// giTyWord is the leading TYPE WORD of a type text, which may carry trailing
// whitespace, a comment or a following name.
func giTyWord(ty string) string {
	i := 0
	for i < len(ty) {
		c := ty[i]
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_') {
			break
		}
		i++
	}
	return ty[:i]
}

// giAdoptLike applies the width the slot ALREADY carries.
func giAdoptLike(rt *jsrt, cur, v interface{}) interface{} {
	g, ok := cur.(jsGInt)
	if !ok || !giNumeric(v) {
		return v
	}
	return giNorm(giVal(rt, v), g.w, g.u)
}

// giFieldTy is the declared type text of one field of a struct value, read off
// its descriptor. "" when the value is not a struct instance or the name is not
// one of its fields (an embedded promotion, the slice header's own props).
func giFieldTy(o interface{}, name string) string {
	obj, ok := o.(*jsObject)
	if !ok {
		return ""
	}
	cls, ok := obj.props["__class"].(*jsObject)
	if !ok {
		return ""
	}
	fs, ok := cls.props["__fields"].(*jsArray)
	if !ok {
		return ""
	}
	ft, ok := cls.props["__ftypes"].(*jsArray)
	if !ok {
		return ""
	}
	for i, f := range fs.elems {
		if s, ok := f.(string); ok && s == name && i < len(ft.elems) {
			if t, ok := ft.elems[i].(string); ok {
				return t
			}
		}
	}
	return ""
}

// giSlotOf is the value a container slot holds right now: a map's entry (or its
// zero value when the key is absent - which is exactly the boxed zero built from
// the value type) and an array's element. Never fails: an out of range index or a
// non-container simply has no width to offer.
func (rt *jsrt) giSlotOf(x, k interface{}) interface{} {
	if keys, vals, ok := dictParts(x); ok {
		if i := rt.dictFind(keys, k); i >= 0 {
			return vals.elems[i]
		}
		if obj, ok := x.(*jsObject); ok {
			if z, has := obj.props["zero"]; has {
				return z
			}
		}
		return nil
	}
	if arr, ok := x.(*jsArray); ok {
		i := int(rt.toNumber(k))
		if i < 0 {
			i += len(arr.elems)
		}
		if i >= 0 && i < len(arr.elems) {
			return arr.elems[i]
		}
	}
	return nil
}

func init() {
	rxExtraExterns = append(rxExtraExterns, func(rt *jsrt, m map[string]func(args []uint64) uint64) {
		u := rt.unwrap
		w := rt.wrap
		baseSet := m["js_set"]
		basePyset := m["js_pyset"]

		// (typeText, value): the declared width of a slot whose type IS written at
		// the site - an append into a []uint8, a parameter binding.
		m["js_giadopt"] = func(a []uint64) uint64 {
			ty, ok := u(a[0]).(string)
			if !ok {
				return a[1]
			}
			return w(giAdoptText(rt, ty, u(a[1])))
		}
		// obj.name = v, at the field's declared width. Delegates the store itself.
		m["js_gosetfield"] = func(a []uint64) uint64 {
			ty := giFieldTy(u(a[0]), rt.toString(u(a[1])))
			if ty == "" {
				return baseSet(a)
			}
			return baseSet([]uint64{a[0], a[1], w(giAdoptText(rt, ty, u(a[2])))})
		}
		// s[i] = v THROUGH A SLICE HEADER: one extern instead of the three the
		// emitted IR used (read .a, add .o, js_pyset), so carrying the width here
		// costs less than the store cost before it. The width is the header's
		// recorded element type first - an empty []uint8 has no element to take one
		// from - and the box already in the slot second.
		m["js_gisetsl"] = func(a []uint64) uint64 {
			h, ok := u(a[0]).(*jsObject)
			if !ok {
				return basePyset(a)
			}
			arr, ok := h.props["a"].(*jsArray)
			if !ok {
				return basePyset(a)
			}
			idx := int(rt.toNumber(h.props["o"])) + int(rt.toNumber(u(a[1])))
			v := u(a[2])
			if ty, ok := h.props["et"].(string); ok {
				v = giAdoptText(rt, ty, v)
			} else if idx >= 0 && idx < len(arr.elems) {
				v = giAdoptLike(rt, arr.elems[idx], v)
			}
			if idx < 0 || idx >= len(arr.elems) {
				rt.fail("index %d out of range", idx)
			}
			arr.dropIdx()
			arr.elems[idx] = v
			return 0
		}
		// x[i] = v, at the width the slot already carries. Delegates the store.
		m["js_gisetat"] = func(a []uint64) uint64 {
			cur := rt.giSlotOf(u(a[0]), u(a[1]))
			if cur == nil {
				return basePyset(a)
			}
			return basePyset([]uint64{a[0], a[1], w(giAdoptLike(rt, cur, u(a[2])))})
		}
	})
}
