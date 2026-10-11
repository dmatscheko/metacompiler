package abnf

// Python's dict views for the compiler half. keys(), values() and items()
// answer a live window on the dict, an object {__dview: the method's name,
// __vd: the dict}: it iterates, measures and prints as dict_keys([...]), and a
// keys or items view is set-like (==, & | - ^ and isdisjoint against any
// iterable). Twin of pyNewView in languages/python-interpreter.abnf, whose
// rules these are, and of the pyDView* functions in lib/python-rt.metajs.

func pyNewDView(d *jsObject, kind string) *jsObject {
	o := newJSObject()
	o.set("__dview", kind)
	o.set("__vd", d)
	return o
}

// pyDView answers a view's dict and kind.
func pyDView(v interface{}) (*jsObject, string, bool) {
	o, ok := v.(*jsObject)
	if !ok {
		return nil, "", false
	}
	kind, ok := o.props["__dview"].(string)
	if !ok {
		return nil, "", false
	}
	d, _ := o.props["__vd"].(*jsObject)
	return d, kind, true
}

func pyIsDView(v interface{}) bool {
	_, _, ok := pyDView(v)
	return ok
}

// pyDViewSetLike: a keys or an items view.
func pyDViewSetLike(v interface{}) bool {
	_, kind, ok := pyDView(v)
	return ok && kind != "values"
}

// pyDViewElems is the view's elements now, as a fresh list.
func pyDViewElems(v interface{}) []interface{} {
	d, kind, _ := pyDView(v)
	keys, vals, _ := dictParts(d)
	switch kind {
	case "keys":
		return append([]interface{}{}, keys.elems...)
	case "values":
		return append([]interface{}{}, vals.elems...)
	}
	pairs := make([]interface{}, len(keys.elems))
	for i, k := range keys.elems {
		pairs[i] = pyTup(&jsArray{elems: []interface{}{k, vals.elems[i]}})
	}
	return pairs
}

func pyDViewLen(v interface{}) int {
	d, _, _ := pyDView(v)
	keys, _, _ := dictParts(d)
	return len(keys.elems)
}

// pyDViewHas is x in view: a key is a lookup, an item a lookup and an ==, a
// value a scan.
func (rt *jsrt) pyDViewHas(v, x interface{}) bool {
	d, kind, _ := pyDView(v)
	keys, vals, _ := dictParts(d)
	switch kind {
	case "keys":
		return rt.pyDictFind(keys, x) >= 0
	case "items":
		t, ok := x.(*jsArray)
		if !ok || t.kind != pyTupKind || len(t.elems) != 2 {
			return false
		}
		i := rt.pyDictFind(keys, t.elems[0])
		return i >= 0 && rt.pyEqual(vals.elems[i], t.elems[1])
	}
	for _, e := range vals.elems {
		if rt.pyEqual(e, x) {
			return true
		}
	}
	return false
}

// pyDViewEq is == between a set-like view and a set or another set-like view:
// the same members. A values view equals only itself.
func (rt *jsrt) pyDViewEq(a, b interface{}) bool {
	_, bSet := pySetElems(b)
	if !pyDViewSetLike(a) || !(pyDViewSetLike(b) || bSet) {
		return pySameValue(a, b)
	}
	sa, _ := pySetElems(rt.pyExt("js_pyset_new", a))
	sb, _ := pySetElems(rt.pyExt("js_pyset_new", b))
	if len(sa.elems) != len(sb.elems) {
		return false
	}
	for _, e := range sa.elems {
		if !rt.pySetHas(sb, e) {
			return false
		}
	}
	return true
}

// pyDViewIterKind is the name of the cursor iter() answers for a view.
func pyDViewIterKind(kind string) string {
	switch kind {
	case "keys":
		return "dict_keyiterator"
	case "values":
		return "dict_valueiterator"
	}
	return "dict_itemiterator"
}

// pyDViewMethod is a view's one method, isdisjoint, on a keys or items view;
// any other name is CPython's AttributeError.
func (rt *jsrt) pyDViewMethod(v interface{}, name string, args []interface{}) interface{} {
	if name != "isdisjoint" || !pyDViewSetLike(v) {
		panic(&jsThrown{value: rt.pyExcInstance("AttributeError",
			"'"+pyTypeName(rt, v)+"' object has no attribute '"+name+"'")})
	}
	for _, e := range rt.pyElemsOf(argAt(args, 0)) {
		if rt.pyDViewHas(v, e) {
			return false
		}
	}
	return true
}
