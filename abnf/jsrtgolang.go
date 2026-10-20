package abnf

// jsrtgolang.go -- Go's VALUE RENDERING (fmt's %v) for the compiler grammar
// (languages/go-to-llvm-ir.abnf).
//
// The shared `println` host function hands the RAW runtime value to Go's own %v
// (recurring shape 1 in docs/abnf-dialect-gotchas.md), and this grammar's value
// model puts a class descriptor on every struct and a {a,o,n,c} header on every
// slice, so fmt.Println / fmt.Print / fmt.Sprint need their own renderer: real
// go prints `map[a:1]`, `{1}` and `[1 2]`. This file is the Go twin of the
// interpreter half's `gstr` and the two MUST stay in step - ./test.sh --cross
// diffs them.
//
// Registration is additive: an init() appends a registrar to rxExtraExterns, so
// nothing here runs unless a program calls one of the three js_go* externs, and
// only the Go compiler grammar emits them.
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
//     switches in both halves.
//   - A nil MAP prints `map[]`, as go does: the zero value of a map type is a
//     MARKED EMPTY MAP (__nil) and not the null value, so that it can also be
//     READ like an empty one, which is what go does. (A nil SLICE prints `[]` and
//     a nil pointer / nil interface `<nil>`, both as go does. The zero value of a
//     SLICE type carries the same __nil mark for the same reason, so
//     `var s []int; s == nil` is true as well - see giIsNilZero in abnf/jsrt.go.)
//   - Only a String() method declared on the type ITSELF is a Stringer; a String()
//     promoted from an embedded struct is not found. Go's method set rules are not
//     modelled either, so a String() with a POINTER receiver is used for the value
//     form too (go prints `{2}` there).
//   - fmt's format verbs are those of languages/lib/go-fmt.js, gaps included (see
//     its head). %p and the %v of a function or a channel print an address
//     this runtime makes up, one per referent (gopAddr).

import (
	"fmt"
	"math"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"unicode/utf8"
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
		m["js_goerrors"] = func(a []uint64) uint64 {
			arr, _ := u(a[0]).(*jsArray)
			if arr == nil {
				rt.fail("js_goerrors needs an argument array")
			}
			arg := func(i int) interface{} {
				if i < len(arr.elems) {
					return arr.elems[i]
				}
				return jsUndef
			}
			switch int(rt.toNumber(u(a[1]))) {
			case 0:
				return rt.wrap(goErrNew("*errors.errorString", rt.toString(arg(0)), jsNull))
			case 1:
				var idx []int
				var rest []interface{}
				if len(arr.elems) > 1 {
					rest = arr.elems[1:]
				}
				msg := rt.gopFormat(rt.gopStr(arg(0)), rest, &idx)
				return rt.wrap(goErrorf(msg, rest, idx))
			case 2:
				return rt.wrap(rt.goErrIs(arg(0), arg(1)))
			case 4:
				return rt.wrap(rt.goErrAs(arg(0), arg(1)))
			case 5:
				var errs []interface{}
				for _, e := range arr.elems {
					if !isUndefOrNull(e) {
						errs = append(errs, e)
					}
				}
				return rt.wrap(rt.goErrJoin(errs))
			default:
				un := rt.goErrUnwrap(arg(0))
				if _, isList := un.(*jsArray); isList || goIsSlice(un) {
					return rt.wrap(jsNull)
				}
				return rt.wrap(un)
			}
		}
		m["js_gostr"] = func(a []uint64) uint64 {
			mode := int(rt.toNumber(u(a[1])))
			if mode == gopOne {
				return rt.wrapStr(rt.gopStr(u(a[0])))
			}
			// fmt.Sprintln (mode gopLn) ends its text with the newline.
			out, nl := rt.gopCompose(u(a[0]), mode)
			if nl {
				out += "\n"
			}
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
		return rt.gopFormat(rt.gopStr(arr.elems[0]), arr.elems[1:], nil), false
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
		// fmt asks reflect's Kind, so a defined string type's box is a string too.
		if dv, ok := goDtVal(e); ok {
			_, isStr = dv.(string)
		}
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

// A depth at or above gopNoMeth means "this subtree came through an UNEXPORTED
// FIELD": fmt cannot call Error() or String() on it, because reflect's
// CanInterface is false there and transitively below it. It rides on the depth
// because the method arms are already gated on `depth < 32`, so nothing but the
// recursion cap below has to know about it. See the memberwise arm in gopObj.
const gopNoMeth = 64

func (rt *jsrt) gopRender(v interface{}, depth int) string {
	d := depth
	if d >= gopNoMeth {
		d -= gopNoMeth
	}
	if d > 32 {
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
		// jvmFloText and not goFloStr: a float32 is the SAME box at style
		// floGoF, and its shortest round-tripping form is read at 24 bits
		// (see abnf/jsrtjvm.go).
		return jvmFloText(t)
	case *jsArray:
		// A Go ARRAY is a plain host array; a Go SLICE is the {a,o,n,c} header
		// handled in gopObj. Both print `[e e e]`, space separated.
		parts := make([]string, len(t.elems))
		for i, e := range t.elems {
			parts[i] = rt.gopRender(e, depth+1)
		}
		return "[" + strings.Join(parts, " ") + "]"
	case *jsObject:
		// fmt prints a channel as its address (gfAddrText in lib/go-fmt.js).
		if c, _ := t.props["__chan"].(bool); c {
			return fmt.Sprintf("0x%x", rt.gopAddr(t, ""))
		}
		if s, ok := rt.gopObj(t, depth); ok {
			return s
		}
	}
	if isCallable(v) {
		return fmt.Sprintf("0x%x", rt.gopAddr(v, ""))
	}
	return rt.toString(v)
}

// goNameExported is Go's visibility rule: an identifier is exported exactly when
// it starts with an upper-case letter.
func goNameExported(name string) bool {
	return len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z'
}

// gopObj renders the object shapes the Go grammars build. It reports false for a
// shape it does not know, which falls back to the generic ToString.
func (rt *jsrt) gopObj(o *jsObject, depth int) (string, bool) {
	cls, _ := o.props["__class"].(*jsObject)

	// THE error INTERFACE WINS OVER fmt.Stringer, WHICH WINS OVER EVERY BUILT-IN
	// RENDERING - exactly the order fmt's handleMethods uses (error first, then
	// Stringer): a value with `Error() string` prints `missing b`, not the
	// memberwise `{missing b}`. Guarded by depth so a
	// method that prints its own receiver terminates. The twins are gstr in
	// languages/go-interpreter.abnf and gopObj in languages/lib/go-rt.metajs.
	if cls != nil && depth < 32 {
		if mth := goPrintMeth(o, cls, "Error"); mth != nil {
			return rt.gopRender(rt.call(mth, jsUndef, []interface{}{o}), depth+1), true
		}
		if mth := goPrintMeth(o, cls, "String"); mth != nil {
			return rt.gopRender(rt.call(mth, jsUndef, []interface{}{o}), depth+1), true
		}
	}

	// A map: map[k:v k:v], with the keys SORTED, which is what fmt has done since
	// go 1.12. The value model keeps insertion order, so the order is computed here.
	if keys, vals, ok := dictParts(o); ok {
		order := rt.gopKeyOrder(keys.elems)
		kt, vt := goMapTys(o)
		parts := make([]string, 0, len(order))
		for _, i := range order {
			var val interface{} = jsUndef
			if i < len(vals.elems) {
				val = vals.elems[i]
			}
			parts = append(parts, rt.gopRender(rt.goSlotBox(keys.elems[i], kt), depth+1)+":"+rt.gopRender(rt.goSlotBox(val, vt), depth+1))
		}
		return "map[" + strings.Join(parts, " ") + "]", true
	}

	if cls == nil {
		return "", false
	}
	if dv, ok := goDtVal(o); ok {
		dv = rt.goDtShown(o, dv)
		if strings.HasPrefix(rt.toString(cls.props["__name"]), "*") && goPtrShowsAmp(dv) {
			return "&" + rt.gopRender(dv, depth), true
		}
		return rt.gopRender(dv, depth), true
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
		et, _ := o.props["et"].(string)
		ec := rt.goSlotCls(et)
		parts := make([]string, 0, n)
		for i := 0; i < n; i++ {
			var e interface{} = jsUndef
			if off+i < len(back.elems) {
				e = back.elems[off+i]
			}
			parts = append(parts, rt.gopRender(goSlotIn(e, ec), depth+1))
		}
		return "[" + strings.Join(parts, " ") + "]", true
	}

	// A complex: Go spells it (re+imi), with the sign always written out. BOTH
	// PARTS ARE float64 AND RENDER AS ONE - a complex128 has no other component
	// type - so goFloStr is what has to be used and rt.toString is not
	// (`(1.234567125e+06-0.5i)`, `+Inf`, and the sign of a -0.0 part). The sign of
	// the imaginary part is written out by hand, so the MAGNITUDE is what is
	// rendered - and a -0.0 imaginary part prints as `-0i`, which go agrees with
	// because it takes the same route. The twins are cxStr in go-interpreter.abnf
	// and goCxStr in languages/lib/go-rt.metajs.
	if name == "$cx" {
		return "(" + goFloStr(rt.toNumber(o.props["re"])) +
			goCxImag(goFloStr(rt.toNumber(o.props["im"]))) + "i)", true
	}

	// A pointer CELL (&n, new(int), &p, &s.f): printed as & followed by the
	// pointee - see the header note.
	if name == "$ptr" {
		if pv, ok := rt.goCellRead(o); ok {
			return "&" + rt.gopRender(pv, depth+1), true
		}
		return "&<nil>", true
	}

	// A struct instance: its field values in declaration order, space separated.
	if tag, ok := cls.props["__isclass"]; ok && tag == true {
		flds, _ := cls.props["__fields"].(*jsArray)
		if flds == nil {
			return "{}", true
		}
		fts := rt.goFieldTys(cls)
		parts := make([]string, 0, len(flds.elems))
		for fi, f := range flds.elems {
			fn, _ := f.(string)
			var val interface{} = jsUndef
			if fv, ok := o.props[fn]; ok {
				val = fv
			}
			// AN UNEXPORTED FIELD IS NOT INTERFACEABLE, so fmt never reaches
			// its Error() or String(): printValue calls handleMethods only
			// when reflect says CanInterface, which is false for anything
			// read through a lower-case field - and transitively so for
			// everything below it. `Unexp{err: E{"bad"}}` prints `{{bad}}`
			// in go while the exported `Exp{Err: E{"bad"}}` prints `{bad}`.
			// Raising the depth to the method guard's own limit says exactly
			// that, transitivity included: depth only grows on the way down.
			fd := depth + 1
			if fd < gopNoMeth && !goNameExported(fn) {
				fd += gopNoMeth
			}
			parts = append(parts, rt.gopRender(rt.goSlotBox(val, goTyAt(fts, fi)), fd))
		}
		return "{" + strings.Join(parts, " ") + "}", true
	}
	return "", false
}

// gopKeyOrder answers the indices of a map's keys in fmt's print order (fmt sorts
// map keys since go 1.12): an insertion sort over gopKeyCmp, which keeps the order
// stable and mirrors the interpreter half's own loop.
func (rt *jsrt) gopKeyOrder(keys []interface{}) []int {
	order := make([]int, len(keys))
	for i := range order {
		order[i] = i
	}
	for i := 1; i < len(order); i++ {
		j := i
		for j > 0 && rt.gopKeyCmp(keys[order[j]], keys[order[j-1]]) < 0 {
			order[j], order[j-1] = order[j-1], order[j]
			j--
		}
	}
	return order
}

// gopKeyCmp is fmt's key order (internal/fmtsort): numbers numerically with NaN
// first, strings bytewise, false before true, arrays element by element and
// structs field by field; anything else by its rendered text. Twins: gopKeyCmp in
// lib/go-rt.metajs, gKeyCmp in languages/go-interpreter.abnf.
func (rt *jsrt) gopKeyCmp(a, b interface{}) int {
	if giIsNumeric(a) && giIsNumeric(b) {
		if giIsInt(a) || giIsInt(b) {
			return rt.giCmp(a, b)
		}
		x, y := rt.toNumber(a), rt.toNumber(b)
		switch {
		case x != x && y != y:
			return 0
		case x != x, x < y:
			return -1
		case y != y, x > y:
			return 1
		}
		return 0
	}
	switch x := a.(type) {
	case bool:
		if y, ok := b.(bool); ok {
			if x == y {
				return 0
			}
			if !x {
				return -1
			}
			return 1
		}
	case string:
		if y, ok := b.(string); ok {
			return strings.Compare(x, y)
		}
	case *jsArray:
		if y, ok := b.(*jsArray); ok {
			for i := 0; i < len(x.elems) && i < len(y.elems); i++ {
				if c := rt.gopKeyCmp(x.elems[i], y.elems[i]); c != 0 {
					return c
				}
			}
			return len(x.elems) - len(y.elems)
		}
	case *jsObject:
		ac, bc := goClassOf(a), goClassOf(b)
		if ac != nil && ac == bc {
			if fs, ok := ac.props["__fields"].(*jsArray); ok {
				y := b.(*jsObject)
				for _, f := range fs.elems {
					k := rt.toString(f)
					if c := rt.gopKeyCmp(x.props[k], y.props[k]); c != 0 {
						return c
					}
				}
				return 0
			}
		}
	}
	return strings.Compare(fmt.Sprint(a), fmt.Sprint(b))
}

// ----------------------------------------------------------------------------
// Format verbs

// gopFlags is one directive's flags, width and precision (gfState in
// languages/lib/go-fmt.js).
type gopFlags struct {
	sharp, zero, plus, minus, space bool
	wid, prec                       int
	widPresent, precPresent, plusV  bool
	sharpV                          bool
}

// gopFormat is fmt.Sprintf: the twin of gfFormat in languages/lib/go-fmt.js, which
// the interpreter half and layer 2 share. The parser, the walk over composites and
// the notes follow that file line for line; a SCALAR is handed to Go's own fmt with
// its directive respelled, so this twin checks go-fmt.js's port of fmt/format.go
// and strconv. %q and %U are formatted here the way go-fmt.js does, gaps included
// (see the head of that file).
//
// With wrap non-nil it is fmt.Errorf's formatter: every %w records its operand index
// in wrap, and an operand that is an error prints as %v would (a non-error one keeps
// the bad verb, as in fmt).
func (rt *jsrt) gopFormat(format string, args []interface{}, wrap *[]int) string {
	var out strings.Builder
	argi := 0
	n := len(format)
	i := 0
	for i < n {
		c := format[i]
		if c != '%' {
			out.WriteByte(c)
			i++
			continue
		}
		i++
		f := gopFlags{}
	flags:
		for i < n {
			switch format[i] {
			case '#':
				f.sharp = true
			case '0':
				f.zero = !f.minus
			case '+':
				f.plus = true
			case '-':
				f.minus = true
				f.zero = false
			case ' ':
				f.space = true
			default:
				break flags
			}
			i++
		}
		if i < n && format[i] == '*' {
			i++
			w, ok := 0, false
			if argi < len(args) {
				w, ok = rt.gopIntArg(args[argi])
				argi++
			}
			if !ok {
				out.WriteString("%!(BADWIDTH)")
			} else {
				f.widPresent = true
				f.wid = w
				if f.wid < 0 {
					f.minus = true
					f.zero = false
					f.wid = -f.wid
				}
			}
		} else {
			for i < n && format[i] >= '0' && format[i] <= '9' {
				f.wid = f.wid*10 + int(format[i]-'0')
				f.widPresent = true
				i++
			}
		}
		if i < n && format[i] == '.' {
			i++
			f.precPresent = true
			if i < n && format[i] == '*' {
				i++
				pr, ok := 0, false
				if argi < len(args) {
					pr, ok = rt.gopIntArg(args[argi])
					argi++
				}
				if !ok {
					f.precPresent = false
					out.WriteString("%!(BADPREC)")
				} else if pr < 0 {
					f.precPresent = false
				} else {
					f.prec = pr
				}
			} else {
				for i < n && format[i] >= '0' && format[i] <= '9' {
					f.prec = f.prec*10 + int(format[i]-'0')
					i++
				}
			}
		}
		if i >= n {
			out.WriteString("%!(NOVERB)")
			break
		}
		verb, size := utf8.DecodeRuneInString(format[i:])
		i += size
		if verb == '%' {
			out.WriteByte('%')
			continue
		}
		if argi >= len(args) {
			out.WriteString("%!" + string(verb) + "(MISSING)")
			continue
		}
		if verb == 'w' && wrap != nil {
			*wrap = append(*wrap, argi)
			if goMeth(args[argi], "Error") != nil {
				verb = 'v'
			}
		}
		switch verb {
		case 'T':
			out.WriteString(rt.gopTypeVerb(f, args[argi]))
		case 'p':
			out.WriteString(rt.gopPointer(f, args[argi]))
		default:
			if verb == 'v' {
				f.plusV = f.plus
				f.plus = false
				f.sharpV = f.sharp
				f.sharp = false
			}
			if f.sharpV {
				out.WriteString(rt.gopGoSyn(f, args[argi], "", 0))
			} else {
				out.WriteString(rt.gopArg(f, args[argi], verb, 0, false))
			}
		}
		argi++
	}
	// Operands no directive used: %!(EXTRA type=value, ...).
	if argi < len(args) {
		out.WriteString("%!(EXTRA ")
		for i, a := range args[argi:] {
			if i > 0 {
				out.WriteString(", ")
			}
			if gopKind(a) == "nil" {
				out.WriteString("<nil>")
				continue
			}
			out.WriteString(rt.gopTyPrefix(a) + rt.gopArg(gopFlags{}, a, 'v', 0, false))
		}
		out.WriteByte(')')
	}
	return out.String()
}

// gopGoSyn is gfGoSyn in languages/lib/go-fmt.js: %#v, v as a Go literal of its
// type, slot being the type the enclosing composite declares for it.
func (rt *jsrt) gopGoSyn(f gopFlags, v interface{}, slot string, depth int) string {
	if dv, ok := goDtVal(v); ok {
		dv = rt.goDtShown(v, dv)
		if strings.HasPrefix(rt.gopTypeName(v), "*") && gopKind(dv) == "struct" {
			return "&" + rt.gopGoSyn(f, dv, slot, depth)
		}
		return rt.gopGoSyn(f, dv, slot, depth)
	}
	k := gopKind(v)
	switch k {
	case "nil":
		if depth == 0 {
			return gopPadRunes(f, "<nil>", true)
		}
		return gopNilOf(slot)
	case "bool", "float", "complex", "string":
		// Go's own %#v of the scalar: strconv.Quote for a string, %g for a float.
		g := f
		g.sharpV = false
		return fmt.Sprintf(strings.Replace(gopSpec(g, 'v'), "%", "%#", 1), rt.gopScalar(v))
	case "int":
		g := f
		g.plusV = false
		if strings.HasPrefix(gopFmtTy(v), "u") || strings.HasPrefix(slot, "u") && strings.Contains(gopPredecl, " "+slot+" ") {
			g.sharp = true
			return fmt.Sprintf(gopSpec(g, 'x'), uint64(giVal(rt, v)))
		}
		return fmt.Sprintf(gopSpec(g, 'd'), rt.gopScalar(v))
	}
	ty := rt.gopTypeName(v)
	var b strings.Builder
	switch k {
	case "struct":
		o := v.(*jsObject)
		cls := goClassOf(v)
		flds, _ := cls.props["__fields"].(*jsArray)
		var tys []interface{}
		if ft, ok := cls.props["__ftypes"].(*jsArray); ok {
			tys = ft.elems
		}
		b.WriteString(ty + "{")
		if flds != nil {
			for i, fl := range flds.elems {
				if i > 0 {
					b.WriteString(", ")
				}
				name := rt.toString(fl)
				ft := ""
				if i < len(tys) {
					if t := rt.toString(tys[i]); t != "" {
						ft = gopQualTy(t)
					}
				}
				var fv interface{} = jsUndef
				if x, ok := o.props[name]; ok {
					fv = x
				}
				b.WriteString(name + ":" + rt.gopGoSyn(f, fv, ft, depth+1))
			}
		}
		b.WriteByte('}')
		return b.String()
	case "list":
		es, _, isArr := rt.gopListParts(v)
		et := ty[strings.IndexByte(ty, ']')+1:]
		// A []byte operand spells its type []byte, an element or field []uint8.
		if ty == "[]uint8" && depth == 0 {
			ty = "[]byte"
		}
		if !isArr && giIsNilZero(v) {
			return ty + "(nil)"
		}
		b.WriteString(ty + "{")
		for i, e := range es {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(rt.gopGoSyn(f, e, et, depth+1))
		}
		b.WriteByte('}')
		return b.String()
	case "map":
		if giIsNilZero(v) {
			return ty + "(nil)"
		}
		kt := gopMapKeyTy(ty)
		vt := ty[len(kt)+5:]
		keys, vals, _ := dictParts(v)
		b.WriteString(ty + "{")
		for n, i := range rt.gopKeyOrder(keys.elems) {
			if n > 0 {
				b.WriteString(", ")
			}
			var val interface{} = jsUndef
			if i < len(vals.elems) {
				val = vals.elems[i]
			}
			b.WriteString(rt.gopGoSyn(f, keys.elems[i], kt, depth+1) + ":" + rt.gopGoSyn(f, val, vt, depth+1))
		}
		b.WriteByte('}')
		return b.String()
	}
	if strings.HasPrefix(ty, "chan ") {
		return "(" + ty + ")(" + rt.gopStr(v) + ")"
	}
	return rt.gopStr(v)
}

// gopNilOf is gfNilOf: a nil in a slot of the given type.
func gopNilOf(slot string) string {
	switch {
	case slot == "" || slot == "interface {}":
		return "interface {}(nil)"
	case strings.HasPrefix(slot, "*") || strings.HasPrefix(slot, "func") || strings.HasPrefix(slot, "chan"):
		return "(" + slot + ")(nil)"
	}
	return slot + "(nil)"
}

// gopMapKeyTy is gfMapKeyTy: K of a "map[K]V" type text.
func gopMapKeyTy(t string) string {
	d := 0
	for i := 3; i < len(t); i++ {
		switch t[i] {
		case '[':
			d++
		case ']':
			d--
			if d == 0 {
				return t[4:i]
			}
		}
	}
	return ""
}

// gopTyPrefix is gfTyPrefix: "type=" before a noted value, "" for an unknown type.
func (rt *jsrt) gopTyPrefix(v interface{}) string {
	if t := rt.gopTypeName(v); t != "" {
		return t + "="
	}
	return ""
}

// gopTypeVerb is gfTypeVerb: %T.
func (rt *jsrt) gopTypeVerb(f gopFlags, v interface{}) string {
	if gopKind(v) == "nil" {
		return gopPadRunes(f, "<nil>", true)
	}
	t := rt.gopTypeName(v)
	if t == "" {
		return rt.gopBad(f, 'T', v)
	}
	return gopPadRunes(f, gopTrunc(f, t), true)
}

// gopBad is gfBad: %!verb(type=value).
func (rt *jsrt) gopBad(f gopFlags, verb rune, v interface{}) string {
	if gopKind(v) == "nil" {
		return "%!" + string(verb) + "(<nil>)"
	}
	g := f
	g.plusV = false
	return "%!" + string(verb) + "(" + rt.gopTyPrefix(v) + rt.gopArg(g, v, 'v', 0, true) + ")"
}

// Go keeps every panic that has started and not finished on a chain, and reports an
// unrecovered one with all of them. goChains.active is, per runtime, the panics whose
// frames are running their deferred calls (js_gorundefers), oldest first: the frame's
// goPanics index and the thrown value. A panic's own chain is the active list it was
// raised under, taken when it first reaches a frame's defers or js_gotop - nothing
// between its raise and there changes the list - and kept by its thrown value, a
// fresh *jsThrown per panic that is re-raised as itself. A goAbort is one entry,
// rec when it was recovered by then. Twins: goActive in languages/go-interpreter.abnf
// and lib/go-rt.metajs.
type goAbort struct {
	v     interface{}
	rec   bool
	fresh bool // raised with a value computed at the panic (see $panicf)
}

type goActiveEnt struct {
	at     int
	orig   interface{}
	frozen bool // aborted by a newer panic of the same frame: rec is final
	rec    bool
}

var goChains = struct {
	sync.Mutex
	active map[*jsrt][]goActiveEnt
	chain  map[*jsThrown][]goAbort
	fresh  map[*jsThrown]bool
}{active: map[*jsrt][]goActiveEnt{}, chain: map[*jsThrown][]goAbort{}, fresh: map[*jsThrown]bool{}}

// goChainOf is the chain of the thrown value p, taken now if it has none yet. The
// caller holds goChains.
func (rt *jsrt) goChainOf(p interface{}) []goAbort {
	t, isT := p.(*jsThrown)
	if isT {
		if c, has := goChains.chain[t]; has {
			return c
		}
	}
	act := goChains.active[rt]
	c := make([]goAbort, len(act))
	for i, e := range act {
		v := e.orig
		fresh := false
		if et, ok := v.(*jsThrown); ok {
			v = et.value
			fresh = goChains.fresh[et]
		}
		rec := e.rec
		if !e.frozen {
			rec = e.at >= len(rt.goPanics) || rt.goPanics[e.at] == nil
		}
		c[i] = goAbort{v: v, rec: rec, fresh: fresh}
	}
	if isT {
		goChains.chain[t] = c
	}
	return c
}

// goCopyCoded copies v as code says: "S" a struct (its fields by the descriptor's
// __fcopy codes), "A" + code an array of such elements, "?" a value of a type the
// front end did not know (a struct as "S", an array with its nested arrays), ""
// nothing (goIsStructVal says what a struct is).
func (rt *jsrt) goCopyCoded(v interface{}, code string) interface{} {
	if code == "" {
		return v
	}
	if arr, ok := v.(*jsArray); ok {
		if code[0] != 'A' && code != "?" {
			return v
		}
		ec := "?"
		if code[0] == 'A' {
			ec = code[1:]
		}
		out := &jsArray{elems: make([]interface{}, len(arr.elems))}
		for i, e := range arr.elems {
			if _, isArr := e.(*jsArray); ec == "?" && !isArr {
				out.elems[i] = e
				continue
			}
			out.elems[i] = rt.goCopyCoded(e, ec)
		}
		return out
	}
	o, ok := v.(*jsObject)
	if !ok || !goIsStructVal(v) || (code != "S" && code != "?") {
		return v
	}
	cls := goClassOf(v)
	fs, _ := cls.props["__fields"].(*jsArray)
	fc, _ := cls.props["__fcopy"].(*jsArray)
	out := newJSObject()
	for _, k := range o.keys {
		pv := o.props[k]
		if fs != nil && fc != nil {
			for i, f := range fs.elems {
				if f == k {
					if c, _ := argAt(fc.elems, i).(string); c != "" {
						pv = rt.goCopyCoded(pv, c)
					}
					break
				}
			}
		}
		out.set(k, pv)
	}
	return out
}

// goPtrEq is == between pointers: two struct objects are the same pointer only when
// they are the same object; anything else (a cell, nil) compares as == does.
func (rt *jsrt) goPtrEq(a, b interface{}) bool {
	if goIsStructVal(a) && goIsStructVal(b) {
		return a == b
	}
	return rt.goValueEq(a, b)
}

// goIsStructVal: an instance of a declared struct, not of a "$..." runtime
// descriptor (a slice header, a pointer cell, a complex number).
func goIsStructVal(v interface{}) bool {
	cls := goClassOf(v)
	if cls == nil {
		return false
	}
	nm, _ := cls.props["__name"].(string)
	return !strings.HasPrefix(nm, "$")
}

// goDeferCall calls one deferred function and answers the panic it raised, or nil.
func (rt *jsrt) goDeferCall(f interface{}, args []interface{}) (caught interface{}) {
	defer func() { caught = recover() }()
	rt.call(f, jsUndef, args)
	return nil
}

// goPanicReport is an unrecovered panic as Go reports it, its chain first (the last
// entry is the unrecovered panic), oldest first: each later one indented by a tab and a
// recovered one marked " [recovered]"; a panic raised again with the very value it
// aborted is printed once, marked " [recovered, repanicked]" (go 1.23+). Twin of
// goPanicReport in languages/go-interpreter.abnf and lib/go-rt.metajs.
func (rt *jsrt) goPanicReport(ps []goAbort) string {
	out := ""
	same := func(k int) bool { return k > 0 && !ps[k].fresh && rt.strictEq(ps[k].v, ps[k-1].v) }
	for k, p := range ps {
		if same(k) {
			continue
		}
		if k > 0 {
			out += "\t"
		}
		out += "panic: " + rt.goPanicText(p.v)
		if p.rec && k+1 < len(ps) && same(k+1) {
			out += " [recovered, repanicked]"
		} else if p.rec {
			out += " [recovered]"
		}
		out += "\n"
	}
	return out
}

// goPanicText is a panic value as the runtime prints it: an error's Error() or a
// Stringer's String(), and the value of a defined type as main.T(v), a string kind
// quoted. Twin of goPanicText in languages/go-interpreter.abnf and lib/go-rt.metajs.
func (rt *jsrt) goPanicText(v interface{}) string {
	if dv, ok := goDtVal(v); ok {
		cls := goClassOf(v)
		if _, def := cls.props["__pkg"]; def && goMeth(v, "Error") == nil && goMeth(v, "String") == nil {
			if s, isStr := dv.(string); isStr {
				return rt.gopTypeName(v) + "(\"" + s + "\")"
			}
			return rt.gopTypeName(v) + "(" + rt.gopStr(dv) + ")"
		}
	}
	return rt.gopStr(v)
}

// goDtVal is the value inside the box a value of a defined non-struct type gets in
// an interface (js_godtbox), and false for anything else.
func goDtVal(v interface{}) (interface{}, bool) {
	cls := goClassOf(v)
	if cls == nil || cls.props["__dt"] != true {
		return nil, false
	}
	if dv, ok := v.(*jsObject).props["__dv"]; ok {
		return dv, true
	}
	return jsUndef, true
}

// goPtrShowsAmp: a pointer's box prints a struct, a map or a slice behind a '&',
// unless it has Error() or String(). Twin of goPtrShowsAmp in
// languages/go-interpreter.abnf and lib/go-rt.metajs.
func goPtrShowsAmp(dv interface{}) bool {
	if k := gopKind(dv); k == "list" || k == "map" {
		return true
	}
	return goIsStructVal(dv) && goMeth(dv, "Error") == nil && goMeth(dv, "String") == nil
}

// goPtrBoxed is the struct inside a pointer's box (goPtrBoxTy in
// languages/lib/go-constfold.js), and false for anything else: the box's methods
// are the struct's, called on the struct, and two boxes are equal when they hold
// the same one. Twins: isPtrBox in languages/go-interpreter.abnf, goPtrBoxed in
// lib/go-rt.metajs.
func goPtrBoxed(v interface{}) (interface{}, bool) {
	dv, ok := goDtVal(v)
	if !ok {
		return nil, false
	}
	nm, _ := goClassOf(v).props["__name"].(string)
	return dv, strings.HasPrefix(nm, "*")
}

// goDtShown is the value box v (holding dv) shows fmt: an array type's box shows
// its elements boxed as their element type prints them. Twins: goDtShown in
// languages/go-interpreter.abnf and lib/go-rt.metajs.
func (rt *jsrt) goDtShown(v, dv interface{}) interface{} {
	arr, isArr := dv.(*jsArray)
	name := rt.toString(goClassOf(v).props["__name"])
	if !isArr || !strings.HasPrefix(name, "[") {
		return dv
	}
	cls := rt.goSlotCls(name[strings.Index(name, "]")+1:])
	if cls == nil {
		return dv
	}
	out := &jsArray{elems: make([]interface{}, len(arr.elems))}
	for i, e := range arr.elems {
		out.elems[i] = goSlotIn(e, cls)
	}
	return out
}

// goDtUnbox is v as type want once a type test has matched: the value inside
// want's box.
func goDtUnbox(rt *jsrt, v interface{}, want string) interface{} {
	if dv, ok := goDtVal(v); ok && rt.toString(goClassOf(v).props["__name"]) == want {
		return dv
	}
	return v
}

// goDtReg is, per runtime, the box descriptor of every defined type fmt prints
// through Error() or String() (js_godtreg), by type name.
var goDtReg = struct {
	sync.Mutex
	m map[*jsrt]map[string]*jsObject
}{m: map[*jsrt]map[string]*jsObject{}}

// goSlotBox is an element of a composite as fmt reaches it: boxed when its slot's
// type ty is registered, so fmt calls ty's method on it. goSlotCls answers the
// registered descriptor (nil for none) and goSlotIn boxes in it. Twins: goSlotBox
// in languages/go-interpreter.abnf and lib/go-rt.metajs.
func (rt *jsrt) goSlotBox(v interface{}, ty string) interface{} {
	return goSlotIn(v, rt.goSlotCls(ty))
}

func (rt *jsrt) goSlotCls(ty string) *jsObject {
	if ty == "" {
		return nil
	}
	goDtReg.Lock()
	cls := goDtReg.m[rt][ty]
	goDtReg.Unlock()
	if cls != nil {
		return cls
	}
	// An array of a registered type prints its elements through that type's
	// method, so its own box descriptor is made on first use.
	et := goArrElemTy(ty)
	if et == "" || rt.goSlotCls(et) == nil {
		return nil
	}
	cls = newJSObject()
	cls.set("__isclass", true)
	cls.set("__dt", true)
	cls.set("__name", ty)
	cls.set("__fields", &jsArray{elems: []interface{}{"__dv"}})
	goDtReg.Lock()
	goDtReg.m[rt][ty] = cls
	goDtReg.Unlock()
	return cls
}

// goArrElemTy is "T" for an array type "[N]T" of literal length N, else "".
// Twin of goArrElemTy in languages/lib/go-constfold.js.
func goArrElemTy(t string) string {
	i := 1
	for i < len(t) && t[i] >= '0' && t[i] <= '9' {
		i++
	}
	if len(t) < 4 || t[0] != '[' || i == 1 || i >= len(t)-1 || t[i] != ']' {
		return ""
	}
	return t[i+1:]
}

func goSlotIn(v interface{}, cls *jsObject) interface{} {
	if cls == nil {
		return v
	}
	if _, isBox := goDtVal(v); isBox || goIsStructVal(v) {
		return v
	}
	o := newJSObject()
	o.set("__class", cls)
	o.set("__dv", v)
	return o
}

// goMapTys is a map's key and value types, "" when it records none.
func goMapTys(o *jsObject) (string, string) {
	mt, _ := o.props["mty"].(string)
	if mt == "" {
		return "", ""
	}
	kt := gopMapKeyTy(mt)
	return kt, mt[len(kt)+5:]
}

// goFieldTys is a struct descriptor's declared field types.
func (rt *jsrt) goFieldTys(cls *jsObject) []string {
	ft, ok := cls.props["__ftypes"].(*jsArray)
	if !ok {
		return nil
	}
	out := make([]string, len(ft.elems))
	for i, t := range ft.elems {
		out[i] = rt.toString(t)
	}
	return out
}

func goTyAt(tys []string, i int) string {
	if i < len(tys) {
		return tys[i]
	}
	return ""
}

// gopAddrs is gfAddr's table, one per runtime: the referents %p has met, in order.
var gopAddrs = struct {
	sync.Mutex
	m map[*jsrt][]gopAddrEnt
}{m: map[*jsrt][]gopAddrEnt{}}

type gopAddrEnt struct {
	o   interface{}
	key string
}

// gopAddr is gfAddr. Only pointer-shaped referents are compared, so an
// incomparable value can never reach ==.
func (rt *jsrt) gopAddr(o interface{}, key string) int64 {
	gopAddrs.Lock()
	defer gopAddrs.Unlock()
	tab := gopAddrs.m[rt]
	if reflect.ValueOf(o).Kind() == reflect.Pointer {
		for i, e := range tab {
			if e.o == o && e.key == key {
				return 824633786368 + int64(i)*4096
			}
		}
	}
	gopAddrs.m[rt] = append(tab, gopAddrEnt{o, key})
	return 824633786368 + int64(len(tab))*4096
}

// gopRef is gfHRef: the referent %p takes an address from, nil for a nil one, and
// ok=false for a value that is not a reference. A struct counts as a pointer:
// &T{} and T{} are one object in this value model.
func (rt *jsrt) gopRef(v interface{}) (o interface{}, key string, off int64, ok bool) {
	if isCallable(v) {
		return v, "", 0, true
	}
	ob, isObj := v.(*jsObject)
	if !isObj {
		return nil, "", 0, false
	}
	if giIsNilZero(v) {
		return nil, "", 0, true
	}
	if _, _, isMap := dictParts(v); isMap {
		return v, "", 0, true
	}
	if c, _ := ob.props["__chan"].(bool); c {
		return v, "", 0, true
	}
	cls := goClassOf(v)
	if cls == nil {
		return nil, "", 0, false
	}
	switch cls.props["__name"] {
	case "$sl":
		return ob.props["a"], "", int64(rt.toNumber(ob.props["o"])) * 8, true
	case "$ptr":
		if h, isArr := ob.props["h"].(*jsArray); isArr {
			return h, "", int64(rt.toNumber(ob.props["k"])) * 8, true
		}
		if h, isObj := ob.props["h"].(*jsObject); isObj {
			return h, rt.toString(ob.props["k"]), 0, true
		}
		return ob.props["o"], rt.toString(ob.props["k"]), 0, true
	}
	if gopKind(v) == "struct" {
		return v, "", 0, true
	}
	return nil, "", 0, false
}

// gopPointer is gfPointer: fmtPointer's %p.
func (rt *jsrt) gopPointer(f gopFlags, v interface{}) string {
	// A box's address is the value's inside it.
	if dv, ok := goDtVal(v); ok {
		v = dv
	}
	if gopKind(v) == "nil" {
		return rt.gopBad(f, 'p', v)
	}
	o, key, off, ok := rt.gopRef(v)
	if !ok {
		return rt.gopBad(f, 'p', v)
	}
	var a int64
	if o != nil {
		a = rt.gopAddr(o, key) + off
	}
	g := f
	g.sharp = !f.sharp
	return fmt.Sprintf(gopSpec(g, 'x'), a)
}

// gopIntArg is the integer a '*' takes from its operand (gfIntArg).
func (rt *jsrt) gopIntArg(v interface{}) (int, bool) {
	if gopKind(v) != "int" {
		return 0, false
	}
	x := giVal(rt, v)
	if x > 1000000 || x < -1000000 {
		return 0, false
	}
	return int(x), true
}

// gopFmtTy is the type name fmt writes in a bad-verb note for a SCALAR operand,
// "nil" for nil and "" for anything else (a slice, map or struct). A 64 bit box
// answers "int" or "uint": the value model does not keep int64/uint64 apart from
// int/uint. The twins are gFmtTy in languages/go-interpreter.abnf and gopFmtTy in
// languages/lib/go-rt.metajs.
func gopFmtTy(a interface{}) string {
	switch t := a.(type) {
	case jsNullT, jsUndefT:
		return "nil"
	case bool:
		return "bool"
	case string:
		return "string"
	case jsJFlo:
		if t.sty == floGoF {
			return "float32"
		}
		return "float64"
	case jsGInt:
		base := "int"
		if t.u {
			base = "uint"
		}
		if t.w == 64 {
			return base
		}
		return base + strconv.Itoa(int(t.w))
	case float64:
		if t-t == 0 && t == math.Trunc(t) {
			return "int"
		}
		return "float64"
	}
	if cls := goClassOf(a); cls != nil && cls.props["__name"] == "$cx" {
		return "complex128"
	}
	return ""
}

// gopQualTy is gfQualTy in languages/lib/go-fmt.js: a type as written in package
// main, respelled the way reflect prints it.
func gopQualTy(t string) string {
	isID := func(c byte) bool {
		return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_' || c >= '0' && c <= '9' || c > 127
	}
	var b strings.Builder
	i := 0
	for i < len(t) {
		c := t[i]
		if !isID(c) || c >= '0' && c <= '9' {
			b.WriteByte(c)
			i++
			continue
		}
		j := i
		for j < len(t) && isID(t[j]) {
			j++
		}
		id := t[i:j]
		dotted := i > 0 && t[i-1] == '.' || j < len(t) && t[j] == '.'
		switch {
		case id == "byte":
			id = "uint8"
		case id == "rune":
			id = "int32"
		case id == "any" || id == "interface" && strings.HasPrefix(t[j:], "{}"):
			id = "interface {}"
			if strings.HasPrefix(t[j:], "{}") {
				j += 2
			}
		case len(id) > 4 && strings.HasPrefix(id, "chan") && strings.Contains(gopPredecl, " "+id[4:]+" "):
			id = "chan " + id[4:]
		case !dotted && !strings.Contains(gopPredecl, " "+id+" "):
			id = "main." + id
		}
		b.WriteString(id)
		i = j
	}
	return gopLtyStrip(b.String())
}

// gopLtyStrip is gfLtyStrip in languages/lib/go-fmt.js: each local type's key
// (name + "__L" + n) back to its name.
func gopLtyStrip(t string) string {
	if !strings.Contains(t, "__L") {
		return t
	}
	var b strings.Builder
	i := 0
	for k := strings.Index(t, "__L"); k >= 0; k = strings.Index(t[i:], "__L") {
		k += i
		e := k + 3
		for e < len(t) && t[e] >= '0' && t[e] <= '9' {
			e++
		}
		if k > 0 && e > k+3 && (e == len(t) || !(t[e] == '_' || t[e] >= 'a' && t[e] <= 'z' || t[e] >= 'A' && t[e] <= 'Z' || t[e] > 127)) {
			b.WriteString(t[i:k])
		} else {
			b.WriteString(t[i:e])
		}
		i = e
	}
	return b.String() + t[i:]
}

const gopPredecl = " bool string int int8 int16 int32 int64 uint uint8 uint16 uint32 uint64 uintptr float32 float64 complex64 complex128 error map chan func struct interface "

// gopTargs is gfTargs: an instantiated generic struct's type arguments as %T
// writes them after its name ("[string,main.V]"), "" when it records none or one
// is not known.
func gopTargs(v interface{}) string {
	o, ok := v.(*jsObject)
	if !ok {
		return ""
	}
	arr, ok := o.props["__targs"].(*jsArray)
	if !ok {
		return ""
	}
	parts := make([]string, 0, len(arr.elems))
	for _, e := range arr.elems {
		s, isStr := e.(string)
		if !isStr || s == "" {
			return ""
		}
		parts = append(parts, s)
	}
	return gopQualTy("[" + strings.Join(parts, ",") + "]")
}

// gopTypeName is gfTypeName: fmt's %T, "" where the value model cannot tell.
func (rt *jsrt) gopTypeName(v interface{}) string {
	switch k := gopKind(v); k {
	case "nil":
		return "<nil>"
	case "bool", "string":
		return k
	case "int", "float", "complex":
		return gopFmtTy(v)
	case "struct":
		cls := goClassOf(v)
		if cls.props["__dt"] == true {
			name := rt.toString(cls.props["__name"])
			// A pointer's box whose static type did not spell the type arguments.
			if dv, isPtr := goPtrBoxed(v); isPtr && !strings.Contains(name, "[") && goIsStructVal(dv) {
				return gopQualTy(name) + gopTargs(dv)
			}
			return gopQualTy(name)
		}
		if pkg, ok := cls.props["__pkg"].(string); ok {
			return gopLtyStrip(pkg + "." + rt.toString(cls.props["__name"]) + gopTargs(v))
		}
		if name := rt.toString(cls.props["__name"]); name != "struct" {
			return name + gopTargs(v)
		}
		return rt.gopAnonTy(v)
	case "list":
		es, et, isArr := rt.gopListParts(v)
		ets := ""
		if et != "" {
			ets = gopQualTy(et)
		} else {
			ets = rt.gopCommonTy(es)
		}
		if isArr {
			return "[" + strconv.Itoa(len(es)) + "]" + ets
		}
		return "[]" + ets
	case "map":
		o := v.(*jsObject)
		if mt, ok := o.props["mty"].(string); ok {
			return gopQualTy(mt)
		}
		keys, vals, _ := dictParts(v)
		return "map[" + rt.gopCommonTy(keys.elems) + "]" + rt.gopCommonTy(vals.elems)
	}
	if o, ok := v.(*jsObject); ok {
		if c, _ := o.props["__chan"].(bool); c {
			if et, ok := o.props["czero"].(string); ok && et != "" {
				return "chan " + gopQualTy(et)
			}
		}
	}
	return ""
}

func (rt *jsrt) gopCommonTy(es []interface{}) string {
	if len(es) == 0 {
		return "interface {}"
	}
	t := rt.gopTypeName(es[0])
	if t == "" || t == "<nil>" {
		return "interface {}"
	}
	for _, e := range es[1:] {
		if rt.gopTypeName(e) != t {
			return "interface {}"
		}
	}
	return t
}

// gopAnonTy is gfAnonTy: struct { A int; B string }.
func (rt *jsrt) gopAnonTy(v interface{}) string {
	o := v.(*jsObject)
	cls := goClassOf(v)
	flds, _ := cls.props["__fields"].(*jsArray)
	if flds == nil || len(flds.elems) == 0 {
		return "struct {}"
	}
	var tys []interface{}
	if ft, ok := cls.props["__ftypes"].(*jsArray); ok {
		tys = ft.elems
	}
	parts := make([]string, len(flds.elems))
	for i, f := range flds.elems {
		name := rt.toString(f)
		ty := ""
		if i < len(tys) {
			ty = rt.toString(tys[i])
		}
		if ty != "" {
			ty = gopQualTy(ty)
		} else {
			var fv interface{} = jsUndef
			if x, ok := o.props[name]; ok {
				fv = x
			}
			ty = rt.gopTypeName(fv)
		}
		parts[i] = name + " " + ty
	}
	return "struct { " + strings.Join(parts, "; ") + " }"
}

// gopListParts answers a list's elements, its recorded element type ("" when it
// has none) and whether it is a fixed-length array rather than a slice header.
func (rt *jsrt) gopListParts(v interface{}) ([]interface{}, string, bool) {
	if arr, ok := v.(*jsArray); ok {
		return arr.elems, "", true
	}
	o := v.(*jsObject)
	var es []interface{}
	if back, ok := o.props["a"].(*jsArray); ok {
		off := int(rt.toNumber(o.props["o"]))
		n := int(rt.toNumber(o.props["n"]))
		for i := 0; i < n; i++ {
			var e interface{} = jsUndef
			if off+i < len(back.elems) {
				e = back.elems[off+i]
			}
			es = append(es, e)
		}
	}
	et, _ := o.props["et"].(string)
	return es, et, false
}

// gopKind is gfHKind.
func gopKind(v interface{}) string {
	switch ty := gopFmtTy(v); ty {
	case "nil", "bool", "string":
		return ty
	case "float64", "float32":
		return "float"
	case "complex128":
		return "complex"
	case "":
	default:
		return "int"
	}
	if _, ok := v.(*jsArray); ok {
		return "list"
	}
	if _, _, ok := dictParts(v); ok {
		return "map"
	}
	cls := goClassOf(v)
	if cls == nil {
		return "other"
	}
	if cls.props["__name"] == "$sl" {
		return "list"
	}
	if _, ok := cls.props["__fields"].(*jsArray); ok && cls.props["__isclass"] == true {
		return "struct"
	}
	return "other"
}

// gopScalar is v as the Go value of its own type, for Go's fmt to format.
func (rt *jsrt) gopScalar(v interface{}) interface{} {
	switch t := v.(type) {
	case jsJFlo:
		if t.sty == floGoF {
			return float32(t.f)
		}
		return t.f
	case jsGInt:
		switch {
		case t.u && t.w == 8:
			return uint8(t.v)
		case t.u && t.w == 16:
			return uint16(t.v)
		case t.u && t.w == 32:
			return uint32(t.v)
		case t.u:
			return uint(uint64(t.v))
		case t.w == 8:
			return int8(t.v)
		case t.w == 16:
			return int16(t.v)
		case t.w == 32:
			return int32(t.v)
		}
		return int(t.v)
	case float64:
		if gopFmtTy(t) == "int" {
			return int(t)
		}
		return t
	case jsNullT, jsUndefT:
		return nil
	}
	if gopFmtTy(v) == "complex128" {
		o := v.(*jsObject)
		return complex(rt.toNumber(o.props["re"]), rt.toNumber(o.props["im"]))
	}
	return v
}

// gopSpec respells a directive for Go's fmt.
func gopSpec(f gopFlags, verb rune) string {
	b := []byte{'%'}
	if f.sharp {
		b = append(b, '#')
	}
	if f.zero {
		b = append(b, '0')
	}
	if f.plus || f.plusV {
		b = append(b, '+')
	}
	if f.minus {
		b = append(b, '-')
	}
	if f.space {
		b = append(b, ' ')
	}
	if f.widPresent {
		b = strconv.AppendInt(b, int64(f.wid), 10)
	}
	if f.precPresent {
		b = append(b, '.')
		b = strconv.AppendInt(b, int64(f.prec), 10)
	}
	return string(utf8.AppendRune(b, verb))
}

func gopPadRunes(f gopFlags, s string, zeroOK bool) string {
	if !f.widPresent || f.wid == 0 {
		return s
	}
	w := f.wid - utf8.RuneCountInString(s)
	if w <= 0 {
		return s
	}
	if f.minus {
		return s + strings.Repeat(" ", w)
	}
	if f.zero && zeroOK {
		return strings.Repeat("0", w) + s
	}
	return strings.Repeat(" ", w) + s
}

// gopTrunc is f.truncateString.
func gopTrunc(f gopFlags, s string) string {
	if !f.precPresent {
		return s
	}
	n := 0
	for i := range s {
		if n == f.prec {
			return s[:i]
		}
		n++
	}
	return s
}

// gopQuote is fmtQ (gfQuote).
func gopQuote(f gopFlags, s0 string) string {
	s := gopTrunc(f, s0)
	if f.sharp {
		ok := true
		for _, c := range s {
			if c == '`' || c == 127 || c == 0xFEFF || (c < 32 && c != '\t') {
				ok = false
				break
			}
		}
		if ok {
			return gopPadRunes(f, "`"+s+"`", true)
		}
	}
	return gopPadRunes(f, `"`+gopQBody(f, s, '"')+`"`, true)
}

// gopQBody is gfQBody: with + every rune outside ASCII is escaped.
func gopQBody(f gopFlags, s string, delim byte) string {
	q := gopQuoteBody(s, delim)
	if !f.plus {
		return q
	}
	var b strings.Builder
	for _, c := range q {
		switch {
		case c < 128:
			b.WriteRune(c)
		case c < 0x10000:
			fmt.Fprintf(&b, `\u%04x`, c)
		default:
			fmt.Fprintf(&b, `\U%08x`, c)
		}
	}
	return b.String()
}

// gopRuneOf is rune(c) for %c %q %U (gfRuneOf).
func (rt *jsrt) gopRuneOf(v interface{}) rune {
	if b, ok := v.(jsGInt); ok && b.u {
		if uint64(b.v) > 0x10FFFF {
			return utf8.RuneError
		}
	}
	x := giVal(rt, v)
	if x < 0 || x > 0x10FFFF || (x >= 0xD800 && x <= 0xDFFF) {
		return utf8.RuneError
	}
	return rune(x)
}

// gopArg is gfArg: printArg and printValue in one.
func (rt *jsrt) gopArg(f gopFlags, v interface{}, verb rune, depth int, noMeth bool) string {
	k := gopKind(v)
	if verb == 'v' && !f.widPresent && !f.precPresent && !f.space && !f.plus && !f.sharp && !f.plusV {
		return rt.gopStr(v)
	}
	if k == "nil" {
		if depth > 0 {
			return "<nil>"
		}
		if verb == 'v' {
			return gopPadRunes(f, "<nil>", true)
		}
		return "%!" + string(verb) + "(<nil>)"
	}
	if !noMeth && (verb == 'v' || verb == 's' || verb == 'x' || verb == 'X' || verb == 'q') {
		if ms, ok := rt.gopMeth(v); ok {
			return rt.gopString(f, ms, verb)
		}
	}
	if dv, ok := goDtVal(v); ok {
		dv = rt.goDtShown(v, dv)
		if strings.HasPrefix(rt.gopTypeName(v), "*") && gopKind(dv) == "struct" {
			if !noMeth && (verb == 'v' || verb == 's' || verb == 'x' || verb == 'X' || verb == 'q') {
				if ms, ok := rt.gopMeth(dv); ok {
					return rt.gopString(f, ms, verb)
				}
			}
			return "&" + rt.gopArg(f, dv, verb, depth, noMeth)
		}
		return rt.gopArg(f, dv, verb, depth, noMeth)
	}
	switch k {
	case "bool", "int", "float", "complex":
		switch {
		case k == "int" && verb == 'q':
			return gopPadRunes(f, "'"+gopQBody(f, string(rt.gopRuneOf(v)), '\'')+"'", true)
		case k == "int" && verb == 'U':
			return rt.gopUnicode(f, v)
		case k == "int" && verb == 'c':
			return gopPadRunes(f, string(rt.gopRuneOf(v)), true)
		}
		return fmt.Sprintf(gopSpec(f, verb), rt.gopScalar(v))
	case "string":
		return rt.gopString(f, v.(string), verb)
	case "list":
		return rt.gopList(f, v, verb, depth, noMeth)
	case "map":
		keys, vals, _ := dictParts(v)
		kt, vt := goMapTys(v.(*jsObject))
		var b strings.Builder
		b.WriteString("map[")
		for n, i := range rt.gopKeyOrder(keys.elems) {
			if n > 0 {
				b.WriteByte(' ')
			}
			var val interface{} = jsUndef
			if i < len(vals.elems) {
				val = vals.elems[i]
			}
			b.WriteString(rt.gopArg(f, rt.goSlotBox(keys.elems[i], kt), verb, depth+1, noMeth))
			b.WriteByte(':')
			b.WriteString(rt.gopArg(f, rt.goSlotBox(val, vt), verb, depth+1, noMeth))
		}
		b.WriteByte(']')
		return b.String()
	case "struct":
		o := v.(*jsObject)
		flds := goClassOf(v).props["__fields"].(*jsArray)
		fts := rt.goFieldTys(goClassOf(v))
		var b strings.Builder
		b.WriteByte('{')
		for n, fl := range flds.elems {
			if n > 0 {
				b.WriteByte(' ')
			}
			name := rt.toString(fl)
			if f.plusV {
				b.WriteString(name + ":")
			}
			var fv interface{} = jsUndef
			if x, ok := o.props[name]; ok {
				fv = x
			}
			b.WriteString(rt.gopArg(f, rt.goSlotBox(fv, goTyAt(fts, n)), verb, depth+1, noMeth || !goNameExported(name)))
		}
		b.WriteByte('}')
		return b.String()
	}
	if verb == 'v' {
		return gopPadRunes(f, rt.gopStr(v), true)
	}
	return rt.gopBad(f, verb, v)
}

// gopMeth is gfHMeth: the text of Error(), else String().
func (rt *jsrt) gopMeth(v interface{}) (string, bool) {
	cls := goClassOf(v)
	if cls == nil {
		return "", false
	}
	o, _ := v.(*jsObject)
	for _, name := range []string{"Error", "String"} {
		if mth := goPrintMeth(o, cls, name); mth != nil {
			return rt.gopStr(rt.call(mth, jsUndef, []interface{}{v})), true
		}
	}
	return "", false
}

// goPrintMeth is the method name of cls that fmt calls on o, or nil. An fmt operand
// statically of a struct type is a copy marked __vv (goVvThunk in
// go-to-llvm-ir.abnf), whose method set lacks the pointer-receiver methods. Twins:
// goPrintMeth in languages/go-interpreter.abnf and lib/go-rt.metajs.
func goPrintMeth(o *jsObject, cls *jsObject, name string) interface{} {
	mth, ok := cls.props[name]
	if !ok || !isCallable(mth) {
		return nil
	}
	if o != nil && o.props["__vv"] == true && cls.props["__ptr$"+name] == true {
		return nil
	}
	return mth
}

// gopString is fmtString (gfString).
func (rt *jsrt) gopString(f gopFlags, s string, verb rune) string {
	if verb == 'q' {
		return gopQuote(f, s)
	}
	return fmt.Sprintf(gopSpec(f, verb), s)
}

// gopUnicode is fmtUnicode (gfUnicode).
func (rt *jsrt) gopUnicode(f gopFlags, v interface{}) string {
	mag := uint64(giVal(rt, v))
	hex := strings.ToUpper(strconv.FormatUint(mag, 16))
	prec := 4
	if f.precPresent && f.prec > 4 {
		prec = f.prec
	}
	for len(hex) < prec {
		hex = "0" + hex
	}
	out := "U+" + hex
	if f.sharp && mag <= 0x10FFFF && strconv.IsPrint(rune(mag)) {
		out += " '" + string(rune(mag)) + "'"
	}
	return gopPadRunes(f, out, false)
}

// gopList is gfList: a byte slice is a string to %s %q %x %X.
func (rt *jsrt) gopList(f gopFlags, v interface{}, verb rune, depth int, noMeth bool) string {
	var es []interface{}
	isBytes := false
	if arr, ok := v.(*jsArray); ok {
		es = arr.elems
	} else {
		o := v.(*jsObject)
		if back, ok := o.props["a"].(*jsArray); ok {
			off := int(rt.toNumber(o.props["o"]))
			n := int(rt.toNumber(o.props["n"]))
			for i := 0; i < n; i++ {
				var e interface{} = jsUndef
				if off+i < len(back.elems) {
					e = back.elems[off+i]
				}
				es = append(es, e)
			}
		}
		et, _ := o.props["et"].(string)
		if et == "byte" || et == "uint8" {
			isBytes = true
		}
		if ec := rt.goSlotCls(et); ec != nil {
			for i, e := range es {
				es[i] = goSlotIn(e, ec)
			}
		}
	}
	if !isBytes && len(es) > 0 {
		isBytes = true
		for _, e := range es {
			if gopFmtTy(e) != "uint8" {
				isBytes = false
				break
			}
		}
	}
	if isBytes && (verb == 's' || verb == 'q' || verb == 'x' || verb == 'X') {
		bs := make([]byte, len(es))
		for i, e := range es {
			bs[i] = byte(giVal(rt, e))
		}
		if verb == 'q' {
			return gopQuote(f, string(bs))
		}
		return fmt.Sprintf(gopSpec(f, verb), bs)
	}
	var b strings.Builder
	b.WriteByte('[')
	for i, e := range es {
		if i > 0 {
			b.WriteByte(' ')
		}
		// A byte slice's elements are uint8 whatever the engine holds them as.
		if isBytes && gopKind(e) == "int" {
			e = jsGInt{v: giVal(rt, e), w: 8, u: true}
		}
		b.WriteString(rt.gopArg(f, e, verb, depth+1, noMeth))
	}
	b.WriteByte(']')
	return b.String()
}

// gopQuoteBody is the body of strconv.Quote / QuoteRune: the letter escapes, the
// backslash and whichever delimiter is in force, \xHH for the other ASCII controls
// and \uHHHH / \UHHHHHHHH for any other rune strconv.IsPrint refuses (gfIsPrint in
// languages/lib/go-fmt.js carries the same table), and \xHH for a byte that is
// not part of a character. The twins are gQuoteBody in
// languages/go-interpreter.abnf and gopQuoteBody in languages/lib/go-rt.metajs.
func gopQuoteBody(s string, delim byte) string {
	named := map[rune]string{'\n': `\n`, '\t': `\t`, '\r': `\r`, 7: `\a`, 8: `\b`, 12: `\f`, 11: `\v`}
	out := ""
	for i := 0; i < len(s); {
		c, w := utf8.DecodeRuneInString(s[i:])
		if c == utf8.RuneError && w == 1 {
			out += fmt.Sprintf(`\x%02x`, s[i])
		} else if c == '\\' {
			out += `\\`
		} else if c == rune(delim) {
			out += `\` + string(c)
		} else if e, ok := named[c]; ok {
			out += e
		} else if c < 32 || c == 127 {
			out += fmt.Sprintf(`\x%02x`, c)
		} else if !strconv.IsPrint(c) && c < 0x10000 {
			out += fmt.Sprintf(`\u%04x`, c)
		} else if !strconv.IsPrint(c) {
			out += fmt.Sprintf(`\U%08x`, c)
		} else {
			out += string(c)
		}
		i += w
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

// ----- float32: the one operator extern the width needs ---------------------
//
// goF32Fix is js_gf32fix (registered in abnf/jsrt.go next to js_gflo32). It is
// the WIDTH applied to a binary operator's RESULT, and go-to-llvm-ir.abnf wraps
// every arithmetic operator and every comparison in it - but ONLY in a program
// whose source text mentions `float32` at all (usesF32 there, the gate
// usesComplex already established). Every other Go program's IR is unaffected.
//
// The extern exists because the C floor cannot carry the width. runtime.c is
// compiled by our own c-to-llvm-ir.abnf, which computes every float at DOUBLE
// precision (ctF("flt") is 8 and float arithmetic is emitted as integer
// soft-float), so si_apply / jf_arith can never round to 24 bits.
//
// IT IS A POST-FIX AND NOT A REPLACEMENT, and that shape is forced by layer 2
// rather than chosen here: the operators it corrects are FLOOR externs, and a
// layer-2 file cannot call a floor extern by name (the native binary answers
// `variable not defined: js_gilt`). So the floor's own answer is computed first
// and handed in as `rr`, exactly as java's jvFloFix takes rtjArith's, and this
// half implements the identical rule so the two engines cannot drift.
//
// Two of the arms only exist because of the untyped constant. `f == 0.1` and
// `f < 0.1` put a float32 next to a style-1 box, and Go converts the constant to
// float32 at compile time, so both sides have to be rounded before the compare
// or every such test is false.
func (rt *jsrt) goF32Fix(opv, rr, l, r interface{}) uint64 {
	if !goIsF32(l) && !goIsF32(r) {
		return rt.wrap(rr)
	}
	a, b := jvmFround(rt.toNumber(l)), jvmFround(rt.toNumber(r))
	boolH := func(v bool) uint64 {
		if v {
			return jsHTrue
		}
		return jsHFalse
	}
	switch rt.toString(opv) {
	case "+":
		return rt.wrap(jsJFlo{f: jvmFround(a + b), sty: floGoF})
	case "-":
		return rt.wrap(jsJFlo{f: jvmFround(a - b), sty: floGoF})
	case "*":
		return rt.wrap(jsJFlo{f: jvmFround(a * b), sty: floGoF})
	case "/":
		return rt.wrap(jsJFlo{f: jvmFround(a / b), sty: floGoF})
	case "%":
		return rt.wrap(jsJFlo{f: jvmFround(math.Mod(a, b)), sty: floGoF})
	case "==":
		return boolH(a == b)
	case "!=":
		return boolH(a != b)
	case "<":
		return boolH(a < b)
	case "<=":
		return boolH(a <= b)
	case ">":
		return boolH(a > b)
	}
	return boolH(a >= b)
}

// goIsF32 is the WIDTH question asked of a runtime value: a tag-14 box carrying
// Go's 32-bit style. The twins are goIsF32 in languages/lib/go-rt.metajs (which
// reads floStyle(v) == 4) and v.__f32 in languages/go-interpreter.abnf.
func goIsF32(v interface{}) bool {
	f, ok := v.(jsJFlo)
	return ok && f.sty == floGoF
}

// goCxDiv is Go's OWN complex division and not the textbook formula. The gc
// runtime's complex128div (runtime/complex.go) is SMITH'S ALGORITHM - R. L.
// Smith, "Algorithm 116: Complex division", CACM 5(8):435, 1962 - followed by the
// C99 G.5.1 fixups that recover the infinities a NaN pair hides. The difference
// shows on ORDINARY values and not only on overflow: (0.1+0.2i)/(3+4i) is
// (0.044000000000000004+0.008i) in go where the textbook formula gives
// (...+0.008000000000000002i), and (0.1+0.2i)/(-0.0+0i) is (+Inf+Infi) where the
// textbook formula gives (NaN+NaNi). 55 of 100 rows of a 600-line probe.
//
// It answers a plain {re, im} object rather than a $cx, because the class
// descriptor lives in the emitted module's scope and an extern cannot reach it.
// The twins are cxDiv in languages/go-interpreter.abnf and js_gocxdiv in
// languages/lib/go-rt.metajs; keep the three in step.
func goCxDiv(a, b, c, d float64) (float64, float64) {
	var e, f float64
	// Scaling by the LARGER component keeps the ratio at most 1, which is what
	// stops 1e300/1e300 overflowing the denominator.
	//
	// EVERY PRODUCT IS WRAPPED IN float64(), AND THAT IS LOAD-BEARING RATHER THAN
	// NOISE. Go's spec lets an implementation fuse `x*y + z` into a single FMA,
	// and on arm64 gc does - but neither MetaJS nor the emitted IR has a fused
	// multiply-add, so layer 2 and the interpreter round the product first. An
	// explicit conversion is exactly what the spec says forces the rounding, so
	// this keeps the three engines byte-identical. The residual disagreement with
	// real go is the FMA; see docs/working-on-this-project.md 7.4.
	if math.Abs(c) >= math.Abs(d) {
		ratio := d / c
		denom := c + float64(ratio*d)
		e = (a + float64(b*ratio)) / denom
		f = (b - float64(a*ratio)) / denom
	} else {
		ratio := c / d
		denom := d + float64(ratio*c)
		e = (float64(a*ratio) + b) / denom
		f = (float64(b*ratio) - a) / denom
	}
	if e != e && f != f {
		switch {
		case c == 0 && d == 0 && (a == a || b == b):
			si := math.Copysign(math.Inf(1), c)
			e, f = si*a, si*b
		case (math.IsInf(a, 0) || math.IsInf(b, 0)) && goIsFinite(c) && goIsFinite(d):
			a, b = goInf2One(a), goInf2One(b)
			e = math.Inf(1) * (float64(a*c) + float64(b*d))
			f = math.Inf(1) * (float64(b*c) - float64(a*d))
		case (math.IsInf(c, 0) || math.IsInf(d, 0)) && goIsFinite(a) && goIsFinite(b):
			c, d = goInf2One(c), goInf2One(d)
			e = 0 * (float64(a*c) + float64(b*d))
			f = 0 * (float64(b*c) - float64(a*d))
		}
	}
	return e, f
}

func goIsFinite(x float64) bool { return x == x && !math.IsInf(x, 0) }

// Go's inf2one: 1 with the argument's SIGN for an infinity, and the argument
// itself - a zero, keeping its sign - otherwise.
func goInf2One(x float64) float64 {
	g := 0.0
	if math.IsInf(x, 0) {
		g = 1.0
	}
	return math.Copysign(g, x)
}

// goCxImag puts the ALWAYS-WRITTEN sign in front of an already rendered
// imaginary part. It works on the TEXT rather than on the number because
// goFloStr writes a sign of its own for two of its four special cases: -0.0
// renders "-0" (go prints `(-0-0i)` for a negated complex zero) and an infinity
// renders "+Inf"/"-Inf", where go's complex form is `(0+Infi)` / `(0-Infi)` -
// one plus, not two. A NaN carries no sign in either engine and takes the "+".
// The twins are goCxSign in languages/lib/go-rt.metajs and cxSign in
// languages/go-interpreter.abnf.
func goCxImag(s string) string {
	if strings.HasPrefix(s, "-") {
		return "-" + s[1:]
	}
	if strings.HasPrefix(s, "+") {
		return "+" + s[1:]
	}
	return "+" + s
}

// giAdoptText applies a declared type TEXT to a value. A float64 / float32 slot
// BOXES (jsJFlo), exactly as a `var d float64 = 1` declaration does - without it
// a float64 struct field assigned a plain 1 divides as an integer.
func giAdoptText(rt *jsrt, ty string, v interface{}) interface{} {
	if !giNumeric(v) {
		return v
	}
	switch giTyWord(ty) {
	case "float64":
		return jsJFlo{f: giFloat(rt, v), sty: floGo}
	case "float32":
		// A float32 SLOT rounds what is put in it, which is what makes
		// `var f float32 = 0.1; f == float32(0.1)` hold and what a struct
		// field, a []float32 element and a float32 parameter all need.
		return jsJFlo{f: jvmFround(giFloat(rt, v)), sty: floGoF}
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
		if i := rt.goMapKeyIdx(x, keys, k, false); i >= 0 {
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

// goStdinRest is the part of standard input's current line os.Stdin's Read has
// not handed out yet (js_gostdin).
var goStdinRest string

func init() {
	rxExtraExterns = append(rxExtraExterns, func(rt *jsrt, m map[string]func(args []uint64) uint64) {
		u := rt.unwrap
		w := rt.wrap
		baseSet := m["js_set"]
		basePyset := m["js_pyset"]
		baseMapGet := m["js_map_get"]
		baseGolen := m["js_golen"]

		// js_gofdwrite(fd, s): os.Stdout's (1) and os.Stderr's (2) Write. Standard
		// error goes the way eprintln writes it, ending every write with one newline,
		// so a write's own trailing newline is dropped first. Twin of js_gofdwrite in
		// lib/go-rt.metajs, goFdWrite in go-interpreter.abnf.
		m["js_gofdwrite"] = func(a []uint64) uint64 {
			s := wtf8Clean(rt.toString(u(a[1])))
			if rt.toNumber(u(a[0])) == 1 {
				fmt.Fprint(outWriter, s)
			} else {
				fmt.Fprintln(warnWriter, strings.TrimSuffix(s, "\n"))
			}
			return 0
		}
		// js_gorawstr(s): a string constant's text with each held byte (U+DC80 ..
		// U+DCFF, which utf8Decode in go-to-llvm-ir.abnf makes of a byte that is
		// not part of a character) back as that one byte - phpRaw's rule. Twin of
		// js_gorawstr in lib/go-rt.metajs.
		m["js_gorawstr"] = func(a []uint64) uint64 { return w(phpRaw(rt.toString(u(a[0])))) }
		// js_gostdin(room, eof): os.Stdin's Read - standard input a line at a time
		// (readIn), as [the next bytes of it that fit in room, nil], or ["", eof]
		// at its end. Twin of js_gostdin in lib/go-rt.metajs, goStdinRead in
		// go-interpreter.abnf.
		m["js_gostdin"] = func(a []uint64) uint64 {
			room := int(rt.toNumber(u(a[0])))
			if room > 0 && goStdinRest == "" {
				goStdinRest = readIn(0)
			}
			if room > 0 && goStdinRest == "" {
				return w(&jsArray{elems: []interface{}{"", u(a[1])}})
			}
			cut := room
			if cut > len(goStdinRest) {
				cut = len(goStdinRest)
			}
			chunk := goStdinRest[:cut]
			goStdinRest = goStdinRest[cut:]
			return w(&jsArray{elems: []interface{}{chunk, jsNull}})
		}
		// js_gogetenv(name, lookup): os.Getenv's answer ("" when unset), or with
		// lookup os.LookupEnv's [value, ok]. Twin of js_gogetenv in
		// lib/go-rt.metajs, hostGlobals["os"] in go-interpreter.abnf.
		m["js_gogetenv"] = func(a []uint64) uint64 {
			v, ok := os.LookupEnv(rt.toString(u(a[0])))
			if rt.truthy(u(a[1])) {
				return w(&jsArray{elems: []interface{}{v, ok}})
			}
			return w(v)
		}
		// goMathSqrt(x): the correctly rounded square root lib/go-math.go's Sqrt is,
		// a float64. Twin of js_gosqrt in lib/go-rt.metajs, goMathSqrt in
		// go-interpreter.abnf.
		m["js_gosqrt"] = func(a []uint64) uint64 {
			args, _ := u(a[0]).(*jsArray)
			if args == nil || len(args.elems) == 0 {
				rt.fail("goMathSqrt() needs an argument")
			}
			return w(jsJFlo{f: math.Sqrt(rt.toNumber(args.elems[0])), sty: floGo})
		}
		// cap(x) of anything but a slice header: a channel's buffer size, else
		// what len answers. Twin of js_gocap in lib/go-rt.metajs, goCap in
		// go-interpreter.abnf.
		m["js_gocap"] = func(a []uint64) uint64 {
			if o, _, ok := goChanParts(u(a[0])); ok {
				return rt.wrapNum(rt.toNumber(o.props["cap"]))
			}
			return baseGolen(a)
		}

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
				// A promoted field is written in the embedded struct that holds it.
				if o, ok := u(a[0]).(*jsObject); ok {
					key := rt.toString(u(a[1]))
					if _, own := o.props[key]; !own {
						if h := rt.goPromField(o, key, 16); h != nil {
							return m["js_gosetfield"]([]uint64{w(h), a[1], a[2]})
						}
					}
				}
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
			ui := int(rt.toNumber(u(a[1])))
			idx := int(rt.toNumber(h.props["o"])) + ui
			v := u(a[2])
			if ty, ok := h.props["et"].(string); ok {
				v = giAdoptText(rt, ty, v)
			} else if idx >= 0 && idx < len(arr.elems) {
				v = giAdoptLike(rt, arr.elems[idx], v)
			}
			// THE BOUND IS THE HEADER'S LENGTH, NOT THE BACKING ARRAY'S, and the
			// index reported is the one the program wrote. `base[0:2]` of a five
			// element slice has cap 5, so a check against the storage would let
			// sub[4] through.
			if ui < 0 || ui >= int(rt.toNumber(h.props["n"])) {
				goIdxFail(ui, int(rt.toNumber(h.props["n"])))
			}
			arr.dropIdx()
			arr.elems[idx] = v
			return 0
		}
		// The bounds of a slice EXPRESSION s[lo:hi:mx]: Go requires
		// 0 <= lo <= hi <= mx <= cap, and violating it panics. The emitter calls
		// this before it builds the new header (see $sslice) with (lo, hi, mx or
		// undefined for the two-index form, cap, whether the operand is a slice
		// rather than an array); the twin is js_gislbounds in
		// languages/lib/go-rt.metajs.
		m["js_gislbounds"] = func(a []uint64) uint64 {
			lo := int(rt.toNumber(u(a[0])))
			hi := int(rt.toNumber(u(a[1])))
			cp := int(rt.toNumber(u(a[3])))
			three := !isUndefOrNull(u(a[2]))
			mx := cp
			if three {
				mx = int(rt.toNumber(u(a[2])))
			}
			word := "length"
			if rt.truthy(u(a[4])) {
				word = "capacity"
			}
			goSliceFail(lo, hi, mx, cp, three, word)
			return 0
		}
		// x[i] THROUGH A SLICE HEADER: the read that mirrors js_gisetsl. It exists
		// for the BOUND - js_map_get sees only the backing array and cannot know
		// the header's length (`base[0:2][4]` panics in go). The twin is
		// js_gislget in languages/lib/go-rt.metajs.
		m["js_gislget"] = func(a []uint64) uint64 {
			h, ok := u(a[0]).(*jsObject)
			if !ok {
				return baseMapGet(a)
			}
			arr, ok := h.props["a"].(*jsArray)
			if !ok {
				return baseMapGet(a)
			}
			ui := int(rt.toNumber(u(a[1])))
			if ui < 0 || ui >= int(rt.toNumber(h.props["n"])) {
				goIdxFail(ui, int(rt.toNumber(h.props["n"])))
			}
			return w(arr.elems[int(rt.toNumber(h.props["o"]))+ui])
		}
		// js_goaddr(x, k, pcls, as): &x.f / &x[i]. A cell of class pcls over the
		// struct that holds field k (the embedded one, for a promoted field) or
		// over x's backing array at the element's offset, carrying as for
		// errors.As; the value itself when it is a reference already, as makeAddrVar
		// decides for a variable. A package variable gets the scope cell &name
		// would. Twins: js_goaddr in lib/go-rt.metajs, goAddrOf in
		// go-interpreter.abnf.
		m["js_goaddr"] = func(a []uint64) uint64 {
			x := u(a[0])
			goNilDeref(rt, x)
			if c := goClassOf(x); c != nil && c.props["__name"] == "$ptr" {
				x = rt.goCellLoad(x.(*jsObject))
			}
			var h, key, v interface{}
			if arr, ok := x.(*jsArray); ok {
				i := int(rt.toNumber(u(a[1])))
				if i < 0 || i >= len(arr.elems) {
					goIdxFail(i, len(arr.elems))
				}
				h, key, v = arr, float64(i), arr.elems[i]
			} else if xo, ok := x.(*jsObject); ok && goClassOf(x) != nil && goClassOf(x).props["__name"] == "$sl" {
				v = u(m["js_gislget"](a[:2]))
				h, key = xo.props["a"], float64(int(rt.toNumber(xo.props["o"]))+int(rt.toNumber(u(a[1]))))
			} else if ok {
				if sc, name, isPkg := goPkgSlot(rt, x, u(a[1])); isPkg {
					if _, has := sc.get(name); has {
						cell := newJSObject()
						cell.set("__class", u(a[2]))
						cell.set("o", sc)
						cell.set("k", name)
						return w(cell)
					}
				}
				name := rt.toString(u(a[1]))
				if fv, own := xo.props[name]; own {
					h, key, v = xo, name, fv
				} else if hh := rt.goPromField(xo, name, 16); hh != nil {
					h, key, v = hh, name, hh.props[name]
				}
			}
			if h == nil {
				return m["js_gofield"](a[:2])
			}
			if vo, isObj := v.(*jsObject); isObj {
				if vc := goClassOf(vo); vc == nil || (vc.props["__name"] != "$ptr" && vc.props["__name"] != "$sl") {
					return w(v)
				}
			}
			cell := newJSObject()
			cell.set("__class", u(a[2]))
			cell.set("h", h)
			cell.set("k", key)
			if as, ok := u(a[3]).(string); ok {
				cell.set("as", as)
			}
			return w(cell)
		}
		// *p and *p = v for a cell js_goaddr made. Twins: js_gocellget /
		// js_gocellset in lib/go-rt.metajs.
		m["js_gocellget"] = func(a []uint64) uint64 { return w(rt.goCellLoad(u(a[0]).(*jsObject))) }
		m["js_gocellset"] = func(a []uint64) uint64 {
			rt.goCellStore(u(a[0]).(*jsObject), u(a[1]))
			return 0
		}
		// THE OUTERMOST js_gotry FRAME, popped by jsmain after the entry function
		// returns. An unrecovered panic is reported HERE rather than being let out
		// of the module, so the floor's uncaught-throw handler never adds its own
		// prefix. This is the twin of runEntry in languages/go-interpreter.abnf and
		// js_gotop in languages/lib/go-rt.metajs - same wording, same stream, same
		// exit code, so the three engines print an unrecovered panic identically.
		m["js_gotop"] = func(a []uint64) uint64 {
			n := len(rt.goPanics) - 1
			if n < 0 {
				return 0
			}
			p := rt.goPanics[n]
			rt.goPanics = rt.goPanics[:n]
			if p == nil {
				return 0
			}
			if t, ok := p.(*jsThrown); ok {
				goChains.Lock()
				chain := rt.goChainOf(t)
				fresh := goChains.fresh[t]
				goChains.Unlock()
				if len(chain) > 0 {
					fmt.Fprint(warnWriter, wtf8Clean(rt.goPanicReport(append(chain[:len(chain):len(chain)], goAbort{v: t.value, fresh: fresh}))))
					os.Exit(2)
				}
			}
			// THE WORDING IS GO'S OWN (go 1.26.5, on stderr with exit status 2):
			//
			//     panic: abort probe
			//     panic: runtime error: index out of range [5] with length 0
			//
			// plus a blank line and a "goroutine 1 [running]:" trace, which
			// no engine here can produce. "panic: " prefixes BOTH kinds; a
			// runtime panic's value is an error whose Error() already reads
			// "runtime error: <msg>" (see goRTErr).
			text := ""
			if exc, ok := p.(*jsThrown); ok {
				text = rt.goPanicText(exc.value)
			} else if s, ok := p.(string); ok {
				text = s
			} else {
				text = fmt.Sprint(p)
			}
			fmt.Fprint(warnWriter, wtf8Clean("panic: "+text+"\n"))
			os.Exit(2)
			return 0
		}
		// The frame's deferred calls, LIFO. While they run for the frame's panic,
		// that panic is ACTIVE (goChains). A deferred call that panics does not stop
		// the others: its panic becomes the frame's (js_gorepanic raises it) and the
		// remaining calls run for it, while the panic it aborted stays on the chain
		// with its recovered bit frozen. Twins: goFrameDefers in
		// languages/go-interpreter.abnf, js_gorundefers in lib/go-rt.metajs.
		// js_gocopy(v, code): the copy a store of a struct VALUE makes, by the code
		// goCopyCodeC in languages/go-to-llvm-ir.abnf chose. js_goptreq / js_goptrne:
		// == on pointers, identity for two structs. Twins: js_gocopy / js_goptreq in
		// lib/go-rt.metajs, goCopyBy / goPtrEq in languages/go-interpreter.abnf.
		m["js_gocopy"] = func(a []uint64) uint64 {
			code, _ := u(a[1]).(string)
			return w(rt.goCopyCoded(u(a[0]), code))
		}
		m["js_goptreq"] = func(a []uint64) uint64 { return w(rt.goPtrEq(u(a[0]), u(a[1]))) }
		m["js_goptrne"] = func(a []uint64) uint64 { return w(!rt.goPtrEq(u(a[0]), u(a[1]))) }
		// js_gomakesl(n, c) / js_gomakech(n): the run-time panics of make. Twins:
		// js_gomakesl / js_gomakech in lib/go-rt.metajs, makeMakeCall in
		// languages/go-interpreter.abnf.
		m["js_gomakesl"] = func(a []uint64) uint64 {
			n, c := rt.toNumber(u(a[0])), rt.toNumber(u(a[1]))
			if n < 0 {
				goRTFail("makeslice: len out of range")
			}
			if c < n {
				goRTFail("makeslice: cap out of range")
			}
			return w(jsUndef)
		}
		m["js_gomakech"] = func(a []uint64) uint64 {
			if rt.toNumber(u(a[0])) < 0 {
				panic(goRTErr("runtime.plainError", "makechan: size out of range"))
			}
			return w(jsUndef)
		}
		// js_goifeq / js_goifne: == on two interface values. Twins: js_goifeq in
		// lib/go-rt.metajs, goIfEq in languages/go-interpreter.abnf.
		m["js_goifeq"] = func(a []uint64) uint64 { return w(rt.goIfEq(u(a[0]), u(a[1]))) }
		m["js_goifne"] = func(a []uint64) uint64 { return w(!rt.goIfEq(u(a[0]), u(a[1]))) }
		// panic(v) and $panicf(v): the second argument says the value was computed
		// at the panic, which goPanicReport needs (see PanicCall in the grammars).
		basePanic := m["js_gopanic"]
		m["js_gopanic"] = func(a []uint64) uint64 {
			if len(a) > 1 && rt.truthy(u(a[1])) {
				defer func() {
					c := recover()
					if t, ok := c.(*jsThrown); ok {
						goChains.Lock()
						goChains.fresh[t] = true
						goChains.Unlock()
					}
					panic(c)
				}()
			}
			return basePanic(a)
		}
		baseRunDefers := m["js_rundefers"]
		m["js_gorundefers"] = func(a []uint64) uint64 {
			at := len(rt.goPanics) - 1
			if at < 0 {
				return baseRunDefers(a)
			}
			arr, ok := u(a[0]).(*jsArray)
			if !ok {
				rt.fail("js_gorundefers needs the defer list")
			}
			rt.goDeferAt = append(rt.goDeferAt, at)
			defer func() { rt.goDeferAt = rt.goDeferAt[:len(rt.goDeferAt)-1] }()
			goChains.Lock()
			base := len(goChains.active[rt])
			if p := rt.goPanics[at]; p != nil {
				rt.goChainOf(p)
				goChains.active[rt] = append(goChains.active[rt], goActiveEnt{at: at, orig: p})
			}
			goChains.Unlock()
			for i := len(arr.elems) - 1; i >= 0; i-- {
				pair, ok := arr.elems[i].(*jsArray)
				if !ok || len(pair.elems) != 2 {
					rt.fail("broken defer entry")
				}
				args, _ := pair.elems[1].(*jsArray)
				c := rt.goDeferCall(pair.elems[0], args.elems)
				if c == nil {
					continue
				}
				goChains.Lock()
				rt.goChainOf(c)
				act := goChains.active[rt]
				if top := len(act) - 1; top >= base {
					act[top].frozen = true
					act[top].rec = rt.goPanics[at] == nil
				}
				rt.goPanics[at] = c
				goChains.active[rt] = append(act, goActiveEnt{at: at, orig: c})
				goChains.Unlock()
			}
			goChains.Lock()
			act := goChains.active[rt]
			for _, e := range act[base:] {
				if t, ok := e.orig.(*jsThrown); ok && e.orig != rt.goPanics[at] {
					delete(goChains.chain, t)
					delete(goChains.fresh, t)
				}
			}
			goChains.active[rt] = act[:base]
			goChains.Unlock()
			return 0
		}
		// The DIVISOR of an integer '/' or '%', checked before the operator runs.
		// Go's integer divide by zero is a recoverable run-time panic; the floor's
		// si_arith raises it with die(), which no js_gotry can catch. A layer-2 file
		// cannot call a floor extern by name, so the emitter (emitDivZ in
		// languages/go-to-llvm-ir.abnf) emits this as a PRE-CHECK instead. A float
		// operand on either side is passed over: a float64 divided by zero is an
		// infinity in Go, not a panic. The twin is js_gidivz in
		// languages/lib/go-rt.metajs.
		// The floor's switch from dying to throwing on an integer divide by zero
		// (js_gidivthrow in languages/lib/runtime.c). js_giarith here already
		// raises go's run-time panic, so there is nothing to switch.
		m["js_gidivthrow"] = func(a []uint64) uint64 { return w(jsUndef) }
		m["js_gidivz"] = func(a []uint64) uint64 {
			v := u(a[1])
			if jvmIsFlo(u(a[0])) || jvmIsFlo(v) {
				return 0
			}
			if rt.toNumber(v) == 0 {
				goRTFail("integer divide by zero")
			}
			return 0
		}
		// x[i] = v, at the width the slot already carries. Delegates the store.
		m["js_gisetat"] = func(a []uint64) uint64 {
			// A store into a NIL MAP is a panic in Go, and it is the one thing a
			// nil map does not share with an empty one (see giIsNilMap).
			if giIsNilMap(u(a[0])) {
				panic(goRTErr("runtime.plainError", "assignment to entry in nil map"))
			}
			// An out of range ARRAY store panics with the same wording the READ
			// uses, not js_pyset's "list assignment index out of range" (whose
			// layer-2 twin uses the floor's hard-aborting `fail`).
			if arr, ok := u(a[0]).(*jsArray); ok {
				i := int(rt.toNumber(u(a[1])))
				if i < 0 || i >= len(arr.elems) {
					goIdxFail(i, len(arr.elems))
				}
			}
			cur := rt.giSlotOf(u(a[0]), u(a[1]))
			if cur == nil {
				return basePyset(a)
			}
			return basePyset([]uint64{a[0], a[1], w(giAdoptLike(rt, cur, u(a[2])))})
		}

		// Is v an instance of the descriptor cls, by identity? Twin of js_goiscls
		// in languages/lib/go-rt.metajs.
		m["js_goiscls"] = func(a []uint64) uint64 {
			if o, ok := u(a[0]).(*jsObject); ok {
				if c, ok := o.props["__class"].(*jsObject); ok && c == u(a[1]) {
					return jsHTrue
				}
			}
			return jsHFalse
		}
		// One arm of a type switch, and the ok of the comma-ok assertion: does v
		// have dynamic type `want` (see goTypeIs)?
		m["js_gotypeis"] = func(a []uint64) uint64 {
			if goTypeIs(rt, u(a[0]), rt.toString(u(a[1]))) == 1 {
				return jsHTrue
			}
			return jsHFalse
		}
		// The single-result x.(T) for a T the emitter can decide (assertTyWant):
		// the value, or go's TypeAssertionError panic. The arguments are the value,
		// the type tested and the type as written. The static type of x is not
		// known here, and interface{} is what it nearly always is (recover()
		// answers one), so the text names it. The twins are js_goassert in
		// languages/lib/go-rt.metajs and goAssertType in
		// languages/go-interpreter.abnf.
		m["js_goassert"] = func(a []uint64) uint64 {
			v := u(a[0])
			if goTypeIs(rt, v, rt.toString(u(a[1]))) == 0 {
				goConvFail("interface {} is " + goTypeText(rt, v) + ", not " + gopQualTy(rt.toString(u(a[2]))))
			}
			return w(goDtUnbox(rt, v, rt.toString(u(a[1]))))
		}
		// js_godtbox(v, cls): v in the box of a defined non-struct type, whose
		// descriptor the emitter declared ($dt$T). A box stays itself, and so
		// does a struct instance unless a pointer's static type boxes it (a value
		// of `type Q Point` is a Point already). js_godtas(v, want, written): x.(T) for such a T -
		// the value inside its box, the conversion panic for another box, and the
		// identity for a value that is not one (its type is not known). Twins:
		// goDtBox / goAssertType in languages/go-interpreter.abnf, js_godtbox /
		// js_godtas in lib/go-rt.metajs.
		m["js_godtbox"] = func(a []uint64) uint64 {
			v := u(a[0])
			if _, isBox := goDtVal(v); isBox {
				return a[0]
			}
			cls, _ := u(a[1]).(*jsObject)
			if goIsStructVal(v) && (cls == nil || !strings.HasPrefix(rt.toString(cls.props["__name"]), "*")) {
				// A struct VALUE in an interface is a copy marked __vv, which no
				// type test reads as a pointer (goVvBox in go-interpreter.abnf).
				if o, _ := v.(*jsObject); cls != nil && cls.props["__sv"] == true && o.props["__vv"] != true {
					c := rt.goCopyCoded(v, "S").(*jsObject)
					c.set("__vv", true)
					return w(c)
				}
				return a[0]
			}
			o := newJSObject()
			o.set("__class", u(a[1]))
			o.set("__dv", v)
			return w(o)
		}
		// js_godtreg(cls): cls is the box descriptor of a type fmt prints through
		// its method inside a composite (goSlotBox).
		m["js_godtreg"] = func(a []uint64) uint64 {
			cls, _ := u(a[0]).(*jsObject)
			goDtReg.Lock()
			if goDtReg.m[rt] == nil {
				goDtReg.m[rt] = map[string]*jsObject{}
			}
			goDtReg.m[rt][rt.toString(cls.props["__name"])] = cls
			goDtReg.Unlock()
			return 0
		}
		m["js_godtas"] = func(a []uint64) uint64 {
			v := u(a[0])
			if _, ok := goDtVal(v); !ok {
				return a[0]
			}
			want := rt.toString(u(a[1]))
			if rt.toString(goClassOf(v).props["__name"]) != want {
				goConvFail("interface {} is " + goTypeText(rt, v) + ", not " + gopQualTy(rt.toString(u(a[2]))))
			}
			return w(goDtUnbox(rt, v, want))
		}
		// A DECLARED INTERFACE IS MATCHED STRUCTURALLY: a value satisfies it
		// exactly when it has every method. js_gotypeis can only compare type
		// NAMES, which is not enough for `v.(Sp)` and `case Sp:`. The method set
		// arrives as a comma-joined string the emitter built at parse time. The
		// twins are js_goifis / js_goifas in languages/lib/go-rt.metajs and
		// typeMatches / goAssertIface in languages/go-interpreter.abnf.
		m["js_goifis"] = func(a []uint64) uint64 {
			v := u(a[0])
			if goClassOf(v) == nil || goMissingMethod(rt, v, rt.toString(u(a[1]))) != "" {
				return jsHFalse
			}
			return jsHTrue
		}
		m["js_goifas"] = func(a []uint64) uint64 {
			v := u(a[0])
			// A NIL INTERFACE ASSERTED TO ANYTHING PANICS, with a DIFFERENT text:
			// go says `interface conversion: interface is nil, not main.I` and
			// names no method, because there is no dynamic type to name.
			// goMissingMethod answers "" for a value that is not an instance, so
			// nil needs this arm.
			if v == nil || v == jsNull || v == jsUndef {
				goConvFail("interface is nil, not " + gopQualTy(rt.toString(u(a[1]))))
			}
			miss := goMissingMethod(rt, v, rt.toString(u(a[2])))
			if miss == "" {
				return a[0]
			}
			// Go's own text is `interface conversion: main.Dog is not main.Sp:
			// missing method Fly`.
			goConvFail(goTypeText(rt, v) + " is not " + gopQualTy(rt.toString(u(a[1]))) + ": missing method " + miss)
			return a[0]
		}

		// A MULTI-VALUE RESULT IS MARKED with a __gomv slot, because Go EXPANDS a
		// multi-value call that is the sole argument of a call -
		// `fmt.Println(f())` prints `3 3 3`, not `[3 3 3]` - and an unmarked
		// array could not be told from a Go ARRAY value ([3]int is a plain array
		// in this value model, by design). The twins are js_gomv / js_gomvargs in
		// languages/lib/go-rt.metajs and goMvArr / goMvArgs in
		// languages/go-interpreter.abnf.
		m["js_gomv"] = func(a []uint64) uint64 {
			if arr, ok := goMvArr(u(a[0])); ok {
				return w(arr)
			}
			return a[0]
		}
		m["js_gomvargs"] = func(a []uint64) uint64 {
			args, ok := u(a[0]).(*jsArray)
			if !ok || len(args.elems) != 1 {
				return a[0]
			}
			if arr, isMv := goMvArr(args.elems[0]); isMv {
				return w(arr)
			}
			return a[0]
		}

		// A NIL POINTER DEREFERENCE IS A RECOVERABLE RUNTIME PANIC with ONE
		// wording (goNilDeref) for reading a member of nil, writing one and
		// calling a method on one - not the shared getMember / setMember /
		// memberCall messages. The twins are goNilDeref in
		// languages/go-interpreter.abnf and in languages/lib/go-rt.metajs.
		// Wrapping rather than editing js_gofield / js_gomcall in abnf/jsrt.go
		// keeps this go-local: both names are emitted only by go's grammar.
		baseField := m["js_gofield"]
		baseMcall := m["js_gomcall"]
		baseSetField := m["js_gosetfield"]
		m["js_gofield"] = func(a []uint64) uint64 {
			goNilDeref(rt, u(a[0]))
			if sc, name, ok := goPkgSlot(rt, u(a[0]), u(a[1])); ok {
				if v, has := sc.get(name); has {
					return w(v)
				}
			}
			return baseField(a)
		}
		// js_godtcall(recv, name, args, fn, dyn): a method of a DEFINED type, fn,
		// called with the receiver first; dyn = the static type is unknown, so a
		// struct's own method wins, and with no fn the call is an ordinary one.
		// Twins: js_godtcall in lib/go-rt.metajs, goDtCall in go-interpreter.abnf.
		m["js_godtcall"] = func(a []uint64) uint64 {
			args, ok := u(a[2]).(*jsArray)
			if !ok {
				rt.fail("js_godtcall args must be an array")
			}
			if rt.truthy(u(a[4])) {
				if fn, recv, found := rt.goMethod(u(a[0]), rt.toString(u(a[1]))); found {
					return w(rt.call(fn, jsUndef, append([]interface{}{recv}, args.elems...)))
				}
				if !isCallable(u(a[3])) {
					return m["js_gomcall"](a[:3])
				}
			}
			return w(rt.call(u(a[3]), jsUndef, append([]interface{}{u(a[0])}, args.elems...)))
		}
		// js_gorplan(v): the PLAN of a range loop, made once per loop, as
		// [keys, base, off, n, m]: the i-th key is keys[i] (i itself when keys is
		// null) and the i-th value base[off+i], so the emitted loop reads both with
		// js_get alone. For a map, m is the map, keys a copy of its keys and the
		// value is read live: an entry deleted before it is reached is skipped. An int is its own bound; a slice reads its backing array in
		// place; nil ranges zero times. A string ranges by RUNE (i the rune's BYTE
		// offset, the value its code point) and a channel yields each received value
		// until it is closed and drained - under the cooperative model every producer
		// runs to completion inside the first receive, so the sequence is known here.
		// Both, and a map, read as keys/vals. Twin of js_gorplan in lib/go-rt.metajs.
		m["js_gorplan"] = func(a []uint64) uint64 {
			plan := func(keys, base interface{}, off, n interface{}) uint64 {
				return w(&jsArray{elems: []interface{}{keys, base, off, n, jsNull}})
			}
			v := u(a[0])
			switch x := v.(type) {
			case jsUndefT, jsNullT:
				return plan(jsNull, jsNull, 0.0, 0.0)
			case float64:
				return plan(jsNull, jsNull, 0.0, x)
			case jsGInt:
				return plan(jsNull, jsNull, 0.0, float64(giVal(rt, x)))
			case *jsArray:
				return plan(jsNull, x, 0.0, float64(len(x.elems)))
			case string:
				keys := &jsArray{}
				vals := &jsArray{}
				for i, r := range x {
					keys.elems = append(keys.elems, float64(i))
					vals.elems = append(vals.elems, giNorm(int64(r), 32, false))
				}
				return plan(keys, vals, 0.0, float64(len(keys.elems)))
			}
			if goIsSlice(v) {
				o := v.(*jsObject)
				return plan(jsNull, o.props["a"], o.props["o"], o.props["n"])
			}
			if _, _, isChan := goChanParts(v); isChan {
				keys := &jsArray{}
				for {
					cv, ok := rt.goChanRecv(v)
					if !ok {
						break
					}
					keys.elems = append(keys.elems, cv)
				}
				return plan(keys, keys, 0.0, float64(len(keys.elems)))
			}
			if keys, _, ok := dictParts(v); ok {
				snap := &jsArray{elems: append([]interface{}(nil), keys.elems...)}
				return w(&jsArray{elems: []interface{}{snap, jsNull, 0.0, float64(len(snap.elems)), v}})
			}
			rt.fail("range over a %s", rt.typeOf(v))
			return 0
		}
		// js_godtbind(recv, fn): the method value recv.M of a defined type, its
		// receiver already evaluated. Twins: js_godtbind in lib/go-rt.metajs,
		// goDtBind in go-interpreter.abnf.
		m["js_godtbind"] = func(a []uint64) uint64 {
			recv, fn := u(a[0]), u(a[1])
			return w(jsHostFunc("", func(rt *jsrt, this uint64, args []interface{}) interface{} {
				return rt.call(fn, jsUndef, append([]interface{}{recv}, args...))
			}))
		}
		m["js_gomcall"] = func(a []uint64) uint64 {
			if dv, isPtr := goPtrBoxed(u(a[0])); isPtr {
				a = []uint64{w(dv), a[1], a[2]}
			}
			goNilDeref(rt, u(a[0]))
			// A FUNC-TYPED FIELD IS CALLABLE: `type H struct { f func(int) int }`
			// makes h.f(4) a call of the field's value, with NO receiver - it is a
			// field, not a method. rt.memberCall walks the __class chain for an
			// instance and never looks at the own slot, so this arm is needed (layer
			// 2's goMemberCall reads the slot through goGetMember). Checked
			// only AFTER rt.goMethod missed, so a real method still wins - which is
			// what layer 2's js_gomcall does too.
			if o, ok := u(a[0]).(*jsObject); ok {
				name := rt.toString(u(a[1]))
				if _, _, found := rt.goMethod(u(a[0]), name); !found {
					if fn, has := o.props[name]; has && isCallable(fn) {
						args, isArr := u(a[2]).(*jsArray)
						if isArr {
							return w(rt.call(fn, jsUndef, args.elems))
						}
					}
				}
			}
			if sc, name, ok := goPkgSlot(rt, u(a[0]), u(a[1])); ok {
				if fn, has := sc.get(name); has && isCallable(fn) {
					if args, isArr := u(a[2]).(*jsArray); isArr {
						return w(rt.call(fn, jsUndef, args.elems))
					}
				}
			}
			return baseMcall(a)
		}
		m["js_gosetfield"] = func(a []uint64) uint64 {
			goNilDeref(rt, u(a[0]))
			if sc, name, ok := goPkgSlot(rt, u(a[0]), u(a[1])); ok {
				if _, has := sc.get(name); has {
					rt.scopeSet(sc, name, u(a[2]))
					return 0
				}
			}
			return baseSetField(a)
		}
	})
}

// goPkgSlot: the SCOPE behind an imported package object, and the member name, when
// the receiver is one. buildPackages (languages/go-to-llvm-ir.abnf) puts the package's
// own runtime scope on the object as __pkgscope beside the snapshot of its exported
// names; reading the slot out of the scope instead of off the object is what makes
// `pkga.Counter` see a write the package made after the object was assembled, and what
// makes `pkga.Counter = 50` reach the variable the package's own functions resolve.
// The twin is the __pkgscope arm of js_gofield / js_gosetfield / js_gomcall in
// languages/lib/go-rt.metajs; the interpreter half binds the scope directly.
// Answers false for every other receiver, at the cost of one map lookup on a
// member access whose receiver is an object.
func goPkgSlot(rt *jsrt, recv, key interface{}) (*jsScope, string, bool) {
	o, ok := recv.(*jsObject)
	if !ok {
		return nil, "", false
	}
	sc, ok := o.props["__pkgscope"].(*jsScope)
	if !ok {
		return nil, "", false
	}
	return sc, rt.toString(key), true
}

// goHasMethod: a method of v's own descriptor, or one PROMOTED from an embedded
// struct. rt.goMethod already answers exactly that, receiver semantics included.
func goHasMethod(rt *jsrt, v interface{}, name string) bool {
	_, _, found := rt.goMethod(v, name)
	return found
}

// goMissingMethod answers "" when v has every method in the comma-joined set, and
// otherwise the FIRST one missing - which is what Go's `missing method X` panic
// text names. A value that is not a struct INSTANCE answers "": a method on a
// defined non-struct type is not reachable from the value in this model, so
// refusing there would panic programs go accepts. See the registrations above.
func goMissingMethod(rt *jsrt, v interface{}, csv string) string {
	if goClassOf(v) == nil {
		return ""
	}
	for _, name := range strings.Split(csv, ",") {
		if name == "" {
			continue
		}
		if !goHasMethod(rt, v, name) {
			return name
		}
	}
	return ""
}

// goTypeNameOf is the dynamic type name of a value, in the spelling a case label
// uses: what js_gotypeis compares and what the interface-conversion panic names.
// A float box knows its width and a sized-integer box its width and sign; a
// 64-bit integer does not (int and int64 are one plain number, uint and uint64
// one box), so it answers int / uint. A plain array is "slice", which is what the
// emitter's $istype asks to tell an array from everything else. The twins are
// goTypeName in languages/go-interpreter.abnf and goTypeNameOf in
// languages/lib/go-rt.metajs.
func goTypeNameOf(rt *jsrt, v interface{}) string {
	switch t := v.(type) {
	case jsUndefT, jsNullT:
		return "nil"
	case bool:
		return "bool"
	case string:
		return "string"
	case jsJFlo:
		if t.sty == floGoF {
			return "float32"
		}
		return "float64"
	case jsGInt:
		base := "int"
		if t.u {
			base = "uint"
		}
		if t.w == 64 {
			return base
		}
		return base + strconv.Itoa(int(t.w))
	case float64:
		if t == math.Trunc(t) {
			return "int"
		}
		return "float64"
	case *jsArray:
		return "slice"
	}
	if cls := goClassOf(v); cls != nil {
		name := rt.toString(cls.props["__name"])
		if name == "$cx" {
			return "complex128"
		}
		return name
	}
	if _, _, isDict := dictParts(v); isDict {
		return "map"
	}
	return "any"
}

// goTypeText is the dynamic type a TypeAssertionError names: fmt's %T, except
// that nil is "nil". The twins are goTypeText in languages/go-interpreter.abnf and
// languages/lib/go-rt.metajs.
func goTypeText(rt *jsrt, v interface{}) string {
	if isUndefOrNull(v) {
		return "nil"
	}
	if t := rt.gopTypeName(v); t != "" {
		return t
	}
	return goTypeNameOf(rt, v)
}

// goTypeIs answers whether v's dynamic type is `want`: 1 yes, 0 no, and 2 when the
// value model cannot tell - an int64, and a rune from a literal or a range over a
// string, are the same plain number as an int; a uint64 is the same box as a uint.
// A type switch and the comma-ok form take only a 1; the
// single-result assertion panics only on a 0. The twins are goTypeIs in
// languages/go-interpreter.abnf and languages/lib/go-rt.metajs.
func goTypeIs(rt *jsrt, v interface{}, want string) int {
	if want == "any" || want == "interface{}" {
		if isUndefOrNull(v) {
			return 0
		}
		return 1
	}
	switch want {
	case "byte":
		want = "uint8"
	case "rune":
		want = "int32"
	}
	// *T tests as T, and another package's type by its %T name (goTypeIs in
	// languages/go-interpreter.abnf).
	inst := false
	if o, ok := v.(*jsObject); ok {
		_, inst = o.props["__class"].(*jsObject)
	}
	if _, isBox := goDtVal(v); inst && !isBox && strings.HasPrefix(want, "*") && v.(*jsObject).props["__vv"] != true {
		want = want[1:]
	}
	got := goTypeNameOf(rt, v)
	if got == want {
		return 1
	}
	if inst && strings.Contains(want, ".") && rt.gopTypeName(v) == want {
		return 1
	}
	if strings.HasPrefix(want, "[") || strings.HasPrefix(want, "map[") {
		if rt.gopTypeName(v) == gopQualTy(want) {
			return 1
		}
		return 0
	}
	if (got == "int" && (want == "int64" || want == "int32")) || (got == "uint" && (want == "uint64" || want == "uintptr")) {
		return 2
	}
	return 0
}

// goRTClasses holds one descriptor per runtime error type. It is built once, at
// package initialisation, so concurrent runtimes only ever read it.
var goRTClasses = goRTClassTable("runtime.errorString", "runtime.boundsError",
	"runtime.plainError", "*runtime.TypeAssertionError", "*runtime.PanicNilError")

func goRTClassTable(names ...string) map[string]*jsObject {
	out := map[string]*jsObject{}
	for _, name := range names {
		cls := newJSObject()
		cls.set("__isclass", true)
		cls.set("__name", name)
		cls.set("__fields", &jsArray{elems: []interface{}{"msg"}})
		cls.set("Error", &hostFunc{name: "Error", fn: func(rt *jsrt, this uint64, a []interface{}) interface{} {
			if len(a) > 0 {
				if o, ok := a[0].(*jsObject); ok {
					return o.props["msg"]
				}
			}
			return ""
		}})
		out[name] = cls
	}
	return out
}

// goRTErr is the value of a run-time panic: an instance of a runtime error type
// (runtime.errorString, runtime.boundsError, runtime.plainError,
// *runtime.TypeAssertionError) whose Error() answers its text. So recover() hands
// back an `error` - r.(error) holds, r.(string) does not - and fmt prints the text
// through the Error() arm. The twins are goRTErr in languages/go-interpreter.abnf
// and languages/lib/go-rt.metajs.
func goRTErr(name, text string) *jsThrown {
	o := newJSObject()
	o.set("__class", goRTClasses[name])
	o.set("msg", text)
	return &jsThrown{value: o}
}

// goPanicNil is the value of panic(nil): a *runtime.PanicNilError since go 1.21
// (runtime/panic.go).
func goPanicNil() interface{} {
	return goRTErr("*runtime.PanicNilError", "runtime error: panic called with nil argument").value
}

// errors.New and fmt.Errorf answer instances of go's own unexported types:
// *errors.errorString, and *fmt.wrapError for one %w (Unwrap answers its error) or
// *fmt.wrapErrors for several (errors.Is walks its list; it has no Unwrap method,
// because only compiled code can build a slice). Each is a POINTER in go, so two
// calls with the same text are different errors: the seq field gives every instance
// its own identity under goValueEq's field-wise comparison. The twins are goErrNew
// in languages/go-interpreter.abnf and languages/lib/go-rt.metajs.
var goErrClasses = goErrClassTable()

var goErrSeq atomic.Int64

func goErrClassTable() map[string]*jsObject {
	out := map[string]*jsObject{}
	// A method answering one field of its receiver.
	field := func(name, key string) *hostFunc {
		return &hostFunc{name: name, fn: func(rt *jsrt, this uint64, a []interface{}) interface{} {
			if len(a) > 0 {
				if o, ok := a[0].(*jsObject); ok {
					if v, had := o.props[key]; had {
						return v
					}
				}
			}
			return jsNull
		}}
	}
	for _, name := range []string{"*errors.errorString", "*fmt.wrapError", "*fmt.wrapErrors", "*errors.joinError"} {
		cls := newJSObject()
		cls.set("__isclass", true)
		cls.set("__name", name)
		cls.set("__fields", &jsArray{elems: []interface{}{"msg", "seq"}})
		cls.set("Error", field("Error", "msg"))
		if name == "*fmt.wrapError" {
			cls.set("Unwrap", field("Unwrap", "err"))
		}
		out[name] = cls
	}
	return out
}

// goErrNew is one instance of the error type name; err is a wrapError's error, or
// a wrapErrors' *jsArray.
func goErrNew(name, msg string, err interface{}) *jsObject {
	o := newJSObject()
	o.set("__class", goErrClasses[name])
	o.set("msg", msg)
	o.set("seq", float64(goErrSeq.Add(1)))
	if name == "*fmt.wrapError" {
		o.set("err", err)
	} else if name == "*fmt.wrapErrors" || name == "*errors.joinError" {
		o.set("errs", err)
	}
	return o
}

// goErrorf is fmt.Errorf's result for the text msg and the operand indexes of its
// %w verbs (fmt/errors.go): an errorString for none, a wrapError for one (its error,
// or nil when that operand is not one), a wrapErrors of the error operands for more.
// With no explicit argument indexes (gopFormat has none) idx is ascending and
// distinct.
func goErrorf(msg string, args []interface{}, idx []int) *jsObject {
	switch len(idx) {
	case 0:
		return goErrNew("*errors.errorString", msg, jsNull)
	case 1:
		var w interface{} = jsNull
		if goMeth(args[idx[0]], "Error") != nil {
			w = args[idx[0]]
		}
		return goErrNew("*fmt.wrapError", msg, w)
	}
	errs := &jsArray{}
	for _, n := range idx {
		if goMeth(args[n], "Error") != nil {
			errs.elems = append(errs.elems, args[n])
		}
	}
	return goErrNew("*fmt.wrapErrors", msg, errs)
}

// goMeth is a method of v's dynamic type, or nil. A pointer's box answers its
// struct's, called with the box as receiver.
func goMeth(v interface{}, name string) interface{} {
	if dv, isPtr := goPtrBoxed(v); isPtr {
		m := goMeth(dv, name)
		if m == nil {
			return nil
		}
		return jsHostFunc("", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			if len(args) > 0 {
				if d, ok := goPtrBoxed(args[0]); ok {
					args = append([]interface{}{d}, args[1:]...)
				}
			}
			return rt.call(m, jsUndef, args)
		})
	}
	o, ok := v.(*jsObject)
	if !ok {
		return nil
	}
	cls, _ := o.props["__class"].(*jsObject)
	if cls == nil {
		return nil
	}
	if m, ok := cls.props[name]; ok && isCallable(m) {
		return m
	}
	return nil
}

// goIsSlice reports a slice header ($sl).
func goIsSlice(v interface{}) bool {
	o, ok := v.(*jsObject)
	if !ok {
		return false
	}
	cls, _ := o.props["__class"].(*jsObject)
	return cls != nil && cls.props["__name"] == "$sl"
}

// goErrUnwrap is what errors.Is follows from err: its Unwrap() result (a slice for
// an Unwrap() []error), a wrapErrors' list, or null.
func (rt *jsrt) goErrUnwrap(err interface{}) interface{} {
	if o, ok := err.(*jsObject); ok {
		if l, ok := o.props["errs"].(*jsArray); ok {
			return l
		}
	}
	if m := goMeth(err, "Unwrap"); m != nil {
		return rt.call(m, jsUndef, []interface{}{err})
	}
	return jsNull
}

// goErrEq is `err == target` between two errors. A type whose Error method has a
// POINTER receiver only implements error through a pointer, so two such errors are
// equal only when they are the same object; anything else compares as goValueEq
// does. Twin of goErrEq in languages/go-interpreter.abnf and lib/go-rt.metajs.
func (rt *jsrt) goErrEq(a, b interface{}) bool {
	if ca := goClassOf(a); ca != nil && ca == goClassOf(b) && ca.props["__ptr$Error"] == true {
		return a == b
	}
	return rt.goValueEq(a, b)
}

// goErrJoin is errors.Join over its non-nil operands: nil for none, else an
// *errors.joinError whose text is theirs joined by newlines and whose Unwrap is the
// list. Twin of goErrJoin in languages/go-interpreter.abnf and lib/go-rt.metajs.
func (rt *jsrt) goErrJoin(errs []interface{}) interface{} {
	if len(errs) == 0 {
		return jsNull
	}
	msg := ""
	for i, e := range errs {
		if i > 0 {
			msg += "\n"
		}
		if m := goMeth(e, "Error"); m != nil {
			msg += rt.gopStr(rt.call(m, jsUndef, []interface{}{e}))
		} else {
			msg += rt.gopStr(e)
		}
	}
	return goErrNew("*errors.joinError", msg, &jsArray{elems: errs})
}

// goErrAs is errors.As (errors/wrap.go): the first error in err's tree, depth
// first, that the target's element type accepts - or whose As method takes the
// target - is stored through the target. A pointer cell carries that type as `as`
// (makeAddrVar in go-to-llvm-ir.abnf: "i:" and a method set, or "t:" and a type
// name), because a nil pointer has none; a struct variable is its own target and
// takes the match's fields. Twin of goErrAs in languages/go-interpreter.abnf and
// lib/go-rt.metajs.
func (rt *jsrt) goErrAs(err, target interface{}) bool {
	if isUndefOrNull(target) {
		panic(&jsThrown{value: "errors: target cannot be nil"})
	}
	tc := goClassOf(target)
	to, _ := target.(*jsObject)
	want := ""
	var cls *jsObject
	cell := tc != nil && tc.props["__name"] == "$ptr"
	if cell {
		if as, ok := to.props["as"].(string); ok {
			want = as
		} else {
			cls = goClassOf(rt.goCellLoad(to))
		}
	} else {
		cls = tc
	}
	if want == "" && cls == nil {
		rt.fail("errors.As: the target's type is not known here")
	}
	hit, via, ok := rt.goErrAsFind(err, want, cls, target)
	if !ok {
		return false
	}
	if via {
		return true
	}
	if cell {
		// A box goes into the variable as the value inside ("t:T", goAsWant).
		rt.goCellStore(to, goDtUnbox(rt, hit, strings.TrimPrefix(want, "t:")))
		return true
	}
	if fs, isArr := tc.props["__fields"].(*jsArray); isArr {
		ho, _ := hit.(*jsObject)
		for _, f := range fs.elems {
			k := rt.toString(f)
			if ho != nil {
				to.set(k, ho.props[k])
			}
		}
	}
	return true
}

func (rt *jsrt) goErrAsFind(err interface{}, want string, cls *jsObject, target interface{}) (interface{}, bool, bool) {
	for !isUndefOrNull(err) {
		ok := false
		switch {
		case cls != nil:
			ok = goClassOf(err) == cls
		case strings.HasPrefix(want, "i:"):
			ok = goClassOf(err) != nil && goMissingMethod(rt, err, want[2:]) == ""
		default:
			ok = goTypeIs(rt, err, want[2:]) == 1
		}
		if ok {
			return err, false, true
		}
		if m := goMeth(err, "As"); m != nil && rt.truthy(rt.call(m, jsUndef, []interface{}{err, target})) {
			return err, true, true
		}
		un := rt.goErrUnwrap(err)
		var list []interface{}
		if l, isL := un.(*jsArray); isL {
			list = l.elems
		} else if goIsSlice(un) {
			list, _, _ = rt.gopListParts(un)
		} else {
			err = un
			continue
		}
		for _, e := range list {
			if h, v, found := rt.goErrAsFind(e, want, cls, target); found {
				return h, v, true
			}
		}
		return nil, false, false
	}
	return nil, false, false
}

// goErrIs is errors.Is (errors/wrap.go): err itself, its Is(target) method, then
// the errors it wraps, depth first.
func (rt *jsrt) goErrIs(err, target interface{}) bool {
	if isUndefOrNull(err) || isUndefOrNull(target) {
		return rt.goValueEq(err, target)
	}
	for {
		if rt.goErrEq(err, target) {
			return true
		}
		if m := goMeth(err, "Is"); m != nil && rt.truthy(rt.call(m, jsUndef, []interface{}{err, target})) {
			return true
		}
		un := rt.goErrUnwrap(err)
		var list []interface{}
		if l, ok := un.(*jsArray); ok {
			list = l.elems
		} else if goIsSlice(un) {
			list, _, _ = rt.gopListParts(un)
		} else if isUndefOrNull(un) {
			return false
		} else {
			err = un
			continue
		}
		for _, e := range list {
			if !isUndefOrNull(e) && rt.goErrIs(e, target) {
				return true
			}
		}
		return false
	}
}

// goRTFail raises a run-time panic whose text go prefixes with "runtime error: "
// (a runtime.errorString); goBoundsFail is the same for runtime.boundsError.
func goRTFail(msg string) { panic(goRTErr("runtime.errorString", "runtime error: "+msg)) }

func goBoundsFail(msg string) { panic(goRTErr("runtime.boundsError", "runtime error: "+msg)) }

// goConvFail is a failed type assertion, a *runtime.TypeAssertionError.
func goConvFail(msg string) {
	panic(goRTErr("*runtime.TypeAssertionError", "interface conversion: "+msg))
}

// goIdxFail is go's index-out-of-range panic (runtime/error.go, boundsErrorFmt).
func goIdxFail(i, n int) {
	if i < 0 {
		goBoundsFail(fmt.Sprintf("index out of range [%d]", i))
	}
	goBoundsFail(fmt.Sprintf("index out of range [%d] with length %d", i, n))
}

// goSliceFail panics when s[lo:hi] (three false) or s[lo:hi:mx] breaks
// 0 <= lo <= hi <= mx <= cp, checking in go's own order so the bound named is the
// one go names. word is "capacity" for a slice, "length" for an array or string.
// The twins are goSliceCheck in languages/go-interpreter.abnf and
// languages/lib/go-rt.metajs.
func goSliceFail(lo, hi, mx, cp int, three bool, word string) {
	p := "slice bounds out of range ["
	if three {
		switch {
		case mx < 0:
			goBoundsFail(fmt.Sprintf("%s::%d]", p, mx))
		case mx > cp:
			goBoundsFail(fmt.Sprintf("%s::%d] with %s %d", p, mx, word, cp))
		case hi < 0:
			goBoundsFail(fmt.Sprintf("%s:%d:]", p, hi))
		case hi > mx:
			goBoundsFail(fmt.Sprintf("%s:%d:%d]", p, hi, mx))
		case lo < 0:
			goBoundsFail(fmt.Sprintf("%s%d::]", p, lo))
		case lo > hi:
			goBoundsFail(fmt.Sprintf("%s%d:%d:]", p, lo, hi))
		}
		return
	}
	switch {
	case hi < 0:
		goBoundsFail(fmt.Sprintf("%s:%d]", p, hi))
	case hi > cp:
		goBoundsFail(fmt.Sprintf("%s:%d] with %s %d", p, hi, word, cp))
	case lo < 0:
		goBoundsFail(fmt.Sprintf("%s%d:]", p, lo))
	case lo > hi:
		goBoundsFail(fmt.Sprintf("%s%d:%d]", p, lo, hi))
	}
}

// goMvArr answers the plain array behind a marked multi-value result, and
// whether v was one. See the js_gomv registration above.
func goMvArr(v interface{}) (*jsArray, bool) {
	o, ok := v.(*jsObject)
	if !ok {
		return nil, false
	}
	arr, isArr := o.props["__gomv"].(*jsArray)
	return arr, isArr
}

// goCellLoad and goCellStore read and write through a pointer cell: o and the
// name for a variable's (o the scope, or the one-slot object of new(T)), h and the
// field name or the element's offset for one js_goaddr made. A cell over nothing
// reads undefined. Twins: goCellLoad / goCellStore in lib/go-rt.metajs.
func (rt *jsrt) goCellLoad(c *jsObject) interface{} {
	v, _ := rt.goCellRead(c)
	return v
}

// goCellRead is goCellLoad, and whether the cell holds anything at all.
func (rt *jsrt) goCellRead(c *jsObject) (interface{}, bool) {
	switch o := c.props["o"].(type) {
	case *jsScope:
		return rt.scopeGet(o, rt.toString(c.props["k"])), true
	case *jsObject:
		return o.props[rt.toString(c.props["k"])], true
	}
	switch h := c.props["h"].(type) {
	case *jsObject:
		return h.props[rt.toString(c.props["k"])], true
	case *jsArray:
		return h.elems[int(rt.toNumber(c.props["k"]))], true
	}
	return jsUndef, false
}

func (rt *jsrt) goCellStore(c *jsObject, v interface{}) {
	switch o := c.props["o"].(type) {
	case *jsScope:
		rt.scopeSet(o, rt.toString(c.props["k"]), v)
		return
	case *jsObject:
		o.set(rt.toString(c.props["k"]), v)
		return
	}
	switch h := c.props["h"].(type) {
	case *jsObject:
		h.set(rt.toString(c.props["k"]), v)
	case *jsArray:
		h.elems[int(rt.toNumber(c.props["k"]))] = v
	}
}

// goNilDeref raises Go's nil-pointer-dereference panic when v is nil. See the
// wrappers above.
func goNilDeref(rt *jsrt, v interface{}) {
	if isUndefOrNull(v) {
		goRTFail("invalid memory address or nil pointer dereference")
	}
}

// goDynKind tells apart the dynamic types that share the number representation:
// a plain number (int), a float box by its style (float64, float32) and a sized
// integer by width and sign. Anything else answers -1 and is left to goValueEq.
func goDynKind(v interface{}) int {
	switch x := v.(type) {
	case float64:
		return 0
	case jsJFlo:
		return int(x.sty)
	case jsGInt:
		k := 16 + int(x.w)*2
		if x.u {
			k++
		}
		return k
	}
	return -1
}

// goIfEq is == on two interface values: the dynamic types must match as well.
func (rt *jsrt) goIfEq(a, b interface{}) bool {
	return goDynKind(a) == goDynKind(b) && rt.goValueEq(a, b)
}

// goByteElems: string(x) of a slice decodes UTF-8 when its elements are bytes and
// joins code points when they are runes; a []byte's elements are uint8 boxes.
// Twin of goByteEls in go-interpreter.abnf and lib/go-rt.metajs.
func goByteElems(els []interface{}) bool {
	for _, e := range els {
		if g, ok := e.(jsGInt); ok && g.w == 8 && g.u {
			return true
		}
	}
	return false
}
