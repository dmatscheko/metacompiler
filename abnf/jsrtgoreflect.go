package abnf

// The run-time type primitives lib/go-json.go and lib/go-reflect.go are
// written over (goRf*), as the go-to-llvm-ir grammar's module calls them:
// js_gorf* over the compiler half's value model. goRfZero is emitted into the
// module itself (declRfZero). Twins of the goRf* host globals in
// go-interpreter.abnf and of js_gorf* in lib/go-rt.metajs.

func init() {
	rxExtraExterns = append(rxExtraExterns, func(rt *jsrt, m map[string]func(args []uint64) uint64) {
		u := rt.unwrap
		w := rt.wrap
		arg := func(a []uint64, i int) interface{} {
			args, _ := u(a[0]).(*jsArray)
			if args == nil || i >= len(args.elems) {
				return jsUndef
			}
			return args.elems[i]
		}
		ext := func(name string, vs ...interface{}) interface{} {
			hs := make([]uint64, len(vs))
			for i, v := range vs {
				hs[i] = w(v)
			}
			return u(m[name](hs))
		}
		unbox := func(v interface{}) interface{} {
			if dv, ok := goDtVal(v); ok {
				return dv
			}
			return v
		}
		cellOf := func(v interface{}) (*jsObject, bool) {
			if cls := goClassOf(v); cls != nil && cls.props["__name"] == "$ptr" {
				return v.(*jsObject), true
			}
			return nil, false
		}
		className := func(v interface{}) string {
			if cls := goClassOf(v); cls != nil {
				s, _ := cls.props["__name"].(string)
				return s
			}
			return ""
		}
		elems := func(v interface{}) []interface{} {
			es, et, _ := rt.gopListParts(v)
			out := make([]interface{}, len(es))
			for i, e := range es {
				out[i] = rt.goSlotBox(e, et)
			}
			return out
		}
		kind := func(v interface{}) string {
			x := unbox(v)
			switch x.(type) {
			case jsNullT, jsUndefT:
				return "nil"
			case *jsClosure, *hostFunc, *boundMethod:
				return "func"
			}
			if _, ok := cellOf(x); ok {
				return "ptr"
			}
			if _, _, ok := goChanParts(x); ok {
				return "chan"
			}
			if ty := gopFmtTy(x); ty != "" {
				return ty
			}
			switch gopKind(x) {
			case "list":
				if _, ok := x.(*jsArray); ok {
					return "array"
				}
				return "slice"
			case "map":
				return "map"
			case "struct":
				return "struct"
			}
			return "other"
		}
		adopt := func(ty string, v interface{}) interface{} { return giAdoptText(rt, ty, unbox(v)) }
		m["js_gorfkind"] = func(a []uint64) uint64 { return w(kind(arg(a, 0))) }
		m["js_gorfunbox"] = func(a []uint64) uint64 { return w(unbox(arg(a, 0))) }
		m["js_gorfisnil"] = func(a []uint64) uint64 {
			x := unbox(arg(a, 0))
			switch x.(type) {
			case jsNullT, jsUndefT:
				return w(true)
			}
			o, ok := x.(*jsObject)
			k := gopKind(x)
			return w(ok && (k == "map" || k == "list") && o.props["__nil"] == true)
		}
		m["js_gorfelems"] = func(a []uint64) uint64 { return w(&jsArray{elems: elems(unbox(arg(a, 0)))}) }
		m["js_gorfisbytes"] = func(a []uint64) uint64 {
			x := unbox(arg(a, 0))
			if o, ok := x.(*jsObject); ok {
				if et, _ := o.props["et"].(string); et == "byte" || et == "uint8" {
					return w(true)
				}
			}
			es := elems(x)
			if len(es) == 0 {
				return w(false)
			}
			for _, e := range es {
				if gopFmtTy(e) != "uint8" {
					return w(false)
				}
			}
			return w(true)
		}
		m["js_gorfmapkv"] = func(a []uint64) uint64 {
			x := unbox(arg(a, 0))
			keys, vals, _ := dictParts(x)
			kt, vt := goMapTys(x.(*jsObject))
			ks, vs := &jsArray{}, &jsArray{}
			for _, i := range rt.gopKeyOrder(keys.elems) {
				var v interface{} = jsUndef
				if i < len(vals.elems) {
					v = vals.elems[i]
				}
				ks.elems = append(ks.elems, rt.goSlotBox(keys.elems[i], kt))
				vs.elems = append(vs.elems, rt.goSlotBox(v, vt))
			}
			return w(&jsArray{elems: []interface{}{ks, vs}})
		}
		m["js_gorffields"] = func(a []uint64) uint64 {
			o := unbox(arg(a, 0)).(*jsObject)
			cls := goClassOf(o)
			fs, _ := cls.props["__fields"].(*jsArray)
			fts := rt.goFieldTys(cls)
			tgs, _ := cls.props["__ftags"].(*jsArray)
			names, types, tags, vals := &jsArray{}, &jsArray{}, &jsArray{}, &jsArray{}
			for i, f := range fs.elems {
				name := rt.toString(f)
				ty, tag := goTyAt(fts, i), ""
				if tgs != nil && i < len(tgs.elems) {
					tag = rt.toString(tgs.elems[i])
				}
				var v interface{} = jsUndef
				if fv, ok := o.props[name]; ok {
					v = fv
				}
				names.elems = append(names.elems, name)
				types.elems = append(types.elems, ty)
				tags.elems = append(tags.elems, tag)
				vals.elems = append(vals.elems, rt.goSlotBox(v, ty))
			}
			return w(&jsArray{elems: []interface{}{names, types, tags, vals}})
		}
		m["js_gorfderef"] = func(a []uint64) uint64 {
			p := arg(a, 0)
			if c, ok := cellOf(p); ok {
				return w(rt.goCellLoad(c))
			}
			return w(unbox(p))
		}
		m["js_gorfelemty"] = func(a []uint64) uint64 {
			et := ""
			if o, ok := unbox(arg(a, 0)).(*jsObject); ok && className(o) == "$sl" {
				et, _ = o.props["et"].(string)
			}
			return w(et)
		}
		m["js_gorfmapty"] = func(a []uint64) uint64 {
			mt := ""
			if o, ok := unbox(arg(a, 0)).(*jsObject); ok {
				mt, _ = o.props["mty"].(string)
			}
			return w(mt)
		}
		// goRfMakeSlice(t, elems): a slice of type text t ("[]E") over the elements
		// of the Go slice elems, each adopted to E, with elems' own slice class;
		// an array ("[N]E") is the plain array.
		m["js_gorfmakeslice"] = func(a []uint64) uint64 {
			t := rt.toString(arg(a, 0))
			src := arg(a, 1)
			et := t[indexByte(t, ']')+1:]
			es, _, _ := rt.gopListParts(src)
			arr := &jsArray{}
			for _, e := range es {
				arr.elems = append(arr.elems, adopt(et, e))
			}
			if t[1] != ']' {
				return w(arr)
			}
			h := newJSObject()
			h.set("__class", goClassOf(src))
			h.set("a", arr)
			h.set("o", float64(0))
			h.set("n", float64(len(arr.elems)))
			h.set("c", float64(len(arr.elems)))
			h.set("et", et)
			return w(h)
		}
		m["js_gorfmakemap"] = func(a []uint64) uint64 {
			mo := ext("js_dict_new").(*jsObject)
			mo.set("zero", arg(a, 1))
			mo.set("mty", rt.toString(arg(a, 0)))
			return w(mo)
		}
		// A store makes a nil map non-nil: a pointer to a map is the map itself.
		m["js_gorfmapset"] = func(a []uint64) uint64 {
			mo := arg(a, 0).(*jsObject)
			kt, vt := goMapTys(mo)
			delete(mo.props, "__nil")
			ext("js_gisetat", mo, adopt(kt, arg(a, 1)), adopt(vt, arg(a, 2)))
			return w(jsUndef)
		}
		m["js_gorfsetfield"] = func(a []uint64) uint64 {
			o := unbox(arg(a, 0)).(*jsObject)
			fs := goClassOf(o).props["__fields"].(*jsArray)
			ext("js_gosetfield", o, fs.elems[int(rt.toNumber(arg(a, 1)))], unbox(arg(a, 2)))
			return w(jsUndef)
		}
		// js_gostructset(x, v): `*p = v` for a struct pointer p, which IS the struct
		// x: v's fields are copied into x in place. Twins: goStructSet in
		// languages/go-interpreter.abnf, js_gostructset in lib/go-rt.metajs.
		structSet := func(x, v interface{}) {
			goNilDeref(rt, x)
			o := x.(*jsObject)
			src, _ := v.(*jsObject)
			if fs, ok := goClassOf(o).props["__fields"].(*jsArray); ok && src != nil {
				for _, f := range fs.elems {
					name := rt.toString(f)
					o.set(name, src.props[name])
				}
			}
		}
		m["js_gostructset"] = func(a []uint64) uint64 {
			structSet(u(a[0]), u(a[1]))
			return w(jsUndef)
		}
		m["js_gorfstore"] = func(a []uint64) uint64 {
			p, v := arg(a, 0), unbox(arg(a, 1))
			if c, ok := cellOf(p); ok {
				rt.goCellStore(c, v)
				return w(jsUndef)
			}
			structSet(unbox(p), v)
			return w(jsUndef)
		}
		m["js_gorfconv"] = func(a []uint64) uint64 {
			t := rt.toString(arg(a, 0))
			x := unbox(arg(a, 1))
			if t == "bool" || t == "string" {
				return w(x)
			}
			return w(ext("js_goconv", t, x))
		}
	})
}

func indexByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}
