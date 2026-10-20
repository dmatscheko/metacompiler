package abnf

// Python's DESCRIPTOR surface - the GO TWIN half. The other two halves are in
// languages/lib/python-rt.metajs (the native binary) and in
// languages/python-interpreter.abnf (the interpreter).
//
// Four things live here, and they are one mechanism seen from four sides: a
// callable that is MINTED AT THE ATTRIBUTE READ and remembers what it came from.
// abnf/jsrt.go's pyBoundInfo is that record; its field `kind` tells the three
// callable shapes apart.
//
//	kind ""   a bound method        <bound method C.m of ...>
//	                                <built-in method count of list object>
//	kind "d"  an unbound METHOD DESCRIPTOR read off a builtin TYPE object -
//	          list.count, str.upper, dict.get - which CPython prints
//	          <method 'count' of 'list' objects>, gives no __self__ at all
//	          (__objclass__ instead), and calls with the receiver as its FIRST
//	          argument, so list.count([1,2,2], 2) is 2. `self` holds the TYPE
//	          OBJECT for this kind, not a receiver.
//	kind "m"  a MODULE-level builtin function read off a module object -
//	          math.sqrt - printed <built-in function sqrt>, whose __self__ IS
//	          the module. This is what makes a module member a FIRST-CLASS
//	          VALUE (`f = math.sqrt`) and not only something you can call
//	          through the module.
//
// WHY MINTED ON DEMAND RATHER THAN STORED. The module object could simply hold
// twenty-five closures, and then `math.sqrt` would be an ordinary property read
// with no arm anywhere. It would also be twenty-five cells and their key strings
// held by the root scope for the whole run, in the NATIVE build too. Minting at
// the read costs a program that never reads one exactly nothing.
//
// staticmethod / classmethod are the fourth thing, and they are descriptors of a
// different sort: the decorator only WRAPS, and the unwrapping happens at the
// class-member lookup, which is the one place that knows both the receiver and
// its class.

// ---------------------------------------------------------------- wrappers --

// pyStaticWrap / pyClassWrap are what staticmethod(f) and classmethod(f) return.
// A plain object with a __-prefixed slot, so keysOf hides it and it never
// reaches __dict__ or a class's member enumeration.
func pyStaticWrap(f interface{}) *jsObject {
	o := newJSObject()
	o.set("__pysm", f)
	return o
}

func pyClassWrap(f interface{}) *jsObject {
	o := newJSObject()
	o.set("__pycm", f)
	return o
}

// pyDescKind answers "s" for a staticmethod wrapper, "c" for a classmethod one
// and "" for anything else.
func pyDescKind(v interface{}) string {
	o, ok := v.(*jsObject)
	if !ok {
		return ""
	}
	if _, has := o.props["__pysm"]; has {
		return "s"
	}
	if _, has := o.props["__pycm"]; has {
		return "c"
	}
	return ""
}

// pyDescGet is the member as the attribute read should hand it back. `owner` is
// the class a classmethod binds to: the receiver's OWN class for an instance
// access and the class itself for a class access, which is what makes a
// subclass instance pass the subclass as cls.
func (rt *jsrt) pyDescGet(v interface{}, owner interface{}, name string) interface{} {
	o := v.(*jsObject)
	if f, has := o.props["__pysm"]; has {
		return f
	}
	return rt.pyBindMethod(owner, o.props["__pycm"], name)
}

// pyDescAttr is an attribute of a staticmethod / classmethod object itself:
// the wrapped function as __func__ (and __wrapped__), and its
// __isabstractmethod__, which abc's abstract-method scan reads through the
// wrapper. ok is false for any other name. Twin of pyDescAttr in
// python-interpreter.abnf and lib/python-rt.metajs.
func (rt *jsrt) pyDescAttr(o *jsObject, name string) (interface{}, bool) {
	f, has := o.props["__pysm"]
	if !has {
		f = o.props["__pycm"]
	}
	switch name {
	case "__func__", "__wrapped__":
		return f, true
	case "__isabstractmethod__":
		return rt.pyHasAttr(f, name) && rt.truthy(rt.pyGetAttr(f, name)), true
	}
	return nil, false
}

// ---------------------------------------------------------------- property --

// property(fget=None, fset=None, fdel=None, doc=None) is a data descriptor: a
// plain object marked __pyprop holding the three accessors and the doc. The
// class-member lookups of an instance read, write, delete and method call run
// the accessor (pyPropGet / pyPropSet / pyPropDel); read off the CLASS it is
// the property object itself, whose own attributes pyPropAttr answers.
// getter / setter / deleter answer a COPY with one accessor replaced, which is
// what makes the decorator chain `@x.setter def x(...)` rebind x. Twin of
// pyProp* in lib/python-rt.metajs and python-interpreter.abnf.
func newPyProp(fget, fset, fdel, doc interface{}) *jsObject {
	p := newJSObject()
	p.set("__pyprop", true)
	p.set("fget", fget)
	p.set("fset", fset)
	p.set("fdel", fdel)
	p.set("doc", doc)
	return p
}

func pyPropOf(v interface{}) (*jsObject, bool) {
	o, ok := v.(*jsObject)
	if !ok {
		return nil, false
	}
	_, has := o.props["__pyprop"]
	return o, has
}

// pyPropName is CPython's name of a property: the class-body name
// __set_name__ gave it (pySetNames), else its getter's __name__, else none.
func (rt *jsrt) pyPropName(p *jsObject) (string, bool) {
	if n, ok := p.props["pname"].(string); ok {
		return n, true
	}
	if f := p.props["fget"]; isCallable(f) {
		return rt.toString(rt.pyGetAttr(f, "__name__")), true
	}
	return "", false
}

// pyPropErr is CPython's AttributeError for an accessor the property lacks.
func (rt *jsrt) pyPropErr(p, cls *jsObject, what string) {
	head := "property "
	if n, ok := rt.pyPropName(p); ok {
		head += "'" + n + "' "
	}
	panic(&jsThrown{value: rt.pyExcInstance("AttributeError",
		head+"of '"+rt.toString(cls.props["__name"])+"' object has no "+what)})
}

func (rt *jsrt) pyPropGet(p *jsObject, obj interface{}, cls *jsObject) interface{} {
	if isUndefOrNull(p.props["fget"]) {
		rt.pyPropErr(p, cls, "getter")
	}
	return rt.callPyValue(p.props["fget"], []interface{}{obj}, jsUndef)
}

func (rt *jsrt) pyPropSet(p *jsObject, obj interface{}, cls *jsObject, v interface{}) {
	if isUndefOrNull(p.props["fset"]) {
		rt.pyPropErr(p, cls, "setter")
	}
	rt.callPyValue(p.props["fset"], []interface{}{obj, v}, jsUndef)
}

func (rt *jsrt) pyPropDel(p *jsObject, obj interface{}, cls *jsObject) {
	if isUndefOrNull(p.props["fdel"]) {
		rt.pyPropErr(p, cls, "deleter")
	}
	rt.callPyValue(p.props["fdel"], []interface{}{obj}, jsUndef)
}

// pySetNames is the __set_name__ pass type.__new__ runs once a class body is
// a class: a property member learns its name, and a member whose class
// defines __set_name__ is called with (owner, name). __-prefixed members are
// skipped, because layer 2's keysOf cannot list them. Then the nearest base
// class's __init_subclass__ (implicitly a classmethod) is called with the new
// class and the class statement's keywords kw (a dict, or undefined); with no
// such hook, keywords are object.__init_subclass__'s TypeError. Twin of
// js_pysetnames in lib/python-rt.metajs and pySetNames in
// python-interpreter.abnf.
func (rt *jsrt) pySetNames(cls *jsObject, kw interface{}) {
	for _, k := range append([]string{}, cls.keys...) {
		if len(k) >= 2 && k[0] == '_' && k[1] == '_' {
			continue
		}
		v := cls.props[k]
		if p, isProp := pyPropOf(v); isProp {
			p.set("pname", k)
		} else if _, vcls, isInst := pyInstance(v); isInst {
			if f, ok := pyLookup(vcls, "__set_name__"); ok {
				rt.callPyValue(f, []interface{}{v, cls, k}, jsUndef)
			}
		}
	}
	for _, base := range pyMRO(cls)[1:] {
		_, isUser := base.props["__mro"]
		h, has := base.props["__init_subclass__"]
		if !isUser || !has {
			continue
		}
		if pyDescKind(h) == "c" {
			rt.callPyValue(rt.pyDescGet(h, cls, "__init_subclass__"), nil, kw)
		} else {
			rt.callPyValue(h, []interface{}{cls}, kw)
		}
		return
	}
	if keys, _, ok := dictParts(kw); ok && len(keys.elems) > 0 {
		panic(&jsThrown{value: rt.pyExcInstance("TypeError",
			rt.pyClsQual(cls)+".__init_subclass__() takes no keyword arguments")})
	}
}

// pyPropAttr is an attribute of the property object itself.
func (rt *jsrt) pyPropAttr(p *jsObject, name string) interface{} {
	switch name {
	case "fget", "fset", "fdel":
		return p.props[name]
	case "__doc__":
		return p.props["doc"]
	case "__name__":
		if n, ok := rt.pyPropName(p); ok {
			return n
		}
	case "getter", "setter", "deleter":
		return jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			q := newPyProp(p.props["fget"], p.props["fset"], p.props["fdel"], p.props["doc"])
			if n, ok := p.props["pname"]; ok {
				q.set("pname", n)
			}
			q.set("f"+name[:3], argAt(args, 0))
			return q
		})
	case "__get__":
		return jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			obj := argAt(args, 0)
			if isUndefOrNull(obj) {
				return p
			}
			_, cls, _ := pyInstance(obj)
			return rt.pyPropGet(p, obj, cls)
		})
	}
	panic(&jsThrown{value: rt.pyExcInstance("AttributeError", "'property' object has no attribute '"+name+"'")})
}

// ------------------------------------------------- the unbound descriptor --

// pyDescTypeSample answers a VALUE of the named builtin type, so the existing
// per-type "is this one of my methods" tables (pyBuiltinAttr) can be asked about
// a TYPE OBJECT. Reusing them is what makes the whole method family fall out of
// one arm instead of a hand-written name list that would drift.
func pyDescTypeSample(tname string) interface{} {
	switch tname {
	case "list":
		return &jsArray{}
	case "tuple":
		return &jsArray{kind: pyTupKind}
	case "bytes":
		return &jsArray{kind: pyBytesKind}
	case "bytearray":
		return &jsArray{kind: pyBArrKind}
	case "str":
		return ""
	case "set":
		return newPySet(&jsArray{})
	case "frozenset":
		fs := newPySet(&jsArray{})
		fs.set("__frozen", true)
		return fs
	case "complex":
		return newPyComplex(0, 0)
	case "dict":
		d := newJSObject()
		d.set("__dict", true)
		d.set("keys", &jsArray{})
		d.set("vals", &jsArray{})
		return d
	case "int":
		return 0.0
	case "float":
		return &jsPyFlo{}
	}
	return nil
}

// pyUnboundDesc mints list.count / str.upper / dict.get read off the TYPE
// OBJECT. Calling it takes the receiver as argument zero and then re-enters the
// ordinary bound-method path, so the descriptor and `xs.count(...)` cannot drift.
func (rt *jsrt) pyUnboundDesc(cls *jsObject, name string) interface{} {
	tname := rt.toString(cls.props["__name"])
	h := jsHostFunc("pydesc."+tname+"."+name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
		if len(args) == 0 {
			panic(&jsThrown{value: rt.pyExcInstance("TypeError",
				"unbound method "+tname+"."+name+"() needs an argument")})
		}
		return rt.unwrap(rt.pyBoundBuiltinCall(args[0], name, args[1:]))
	})
	h.pybm = &pyBoundInfo{name: name, self: cls, fn: nil, typ: tname, kind: "d"}
	return h
}

// pyBoundBuiltinCall is "call NAME on this receiver", routed through the same
// extern a written `xs.count(2)` takes - which matters for a STRING, whose
// methods are installed by abnf/pystrmethod.go's registrar by WRAPPING the
// js_pymcall extern rather than living in pyMethodCall.
func (rt *jsrt) pyBoundBuiltinCall(recv interface{}, name string, args []interface{}) uint64 {
	if _, isStr := recv.(string); isStr {
		if m := pyExternMaps[rt]; m != nil {
			if f := m["js_pymcall"]; f != nil {
				return f([]uint64{rt.wrap(recv), rt.wrap(name),
					rt.wrap(&jsArray{elems: args}), rt.wrap(jsUndef)})
			}
		}
	}
	return rt.wrap(rt.pyMethodCall(recv, name, args, jsUndef))
}

// ---------------------------------------------------- the module function --

// pyModuleFn mints `math.sqrt` as a VALUE. It re-enters js_pyrxmcall, the one
// extern the python emitter routes every method call through, so a value read
// off the module and a call made through it run the very same body. A keyword
// call arrives as the positionals, pyKwMark and the keyword dict (pyBindCallR).
func (rt *jsrt) pyModuleFn(mod *jsObject, name string) interface{} {
	mname := rt.toString(mod.props["__pymod"])
	h := jsHostFunc("pymod."+mname+"."+name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
		var kw interface{} = jsUndef
		if n := len(args); n >= 2 && args[n-2] == pyKwMark {
			args, kw = args[:n-2], args[n-1]
		}
		if m := pyExternMaps[rt]; m != nil {
			if f := m["js_pyrxmcall"]; f != nil {
				return rt.unwrap(f([]uint64{rt.wrap(mod), rt.wrap(name),
					rt.wrap(&jsArray{elems: args}), rt.wrap(kw)}))
			}
		}
		rt.fail("module function %s.%s is not reachable", mname, name)
		return jsUndef
	})
	h.pybm = &pyBoundInfo{name: name, self: mod, fn: nil, typ: mname, kind: "m"}
	return h
}

// pyModuleAttr answers a module object's attribute, or nil when the module has
// no such member. `builtins.len` is not handled here: the emitter fills the
// builtins module's own slots with the very bindings the global names resolve
// to, so it is an ordinary property read and `builtins.len is len` holds.
func (rt *jsrt) pyModuleAttr(mod *jsObject, name string) (interface{}, bool) {
	if v, has := mod.props[name]; has {
		return v, true
	}
	// Every module names itself.
	if name == "__name__" {
		return mod.props["__pymod"], true
	}
	if pyModHas(rt.toString(mod.props["__pymod"]), name) {
		return rt.pyModuleFn(mod, name), true
	}
	return nil, false
}

// pyBuiltinsModule is the one `builtins` module object of a run. len.__self__ is
// it, so it has to be a singleton rather than a fresh object per call.
func (rt *jsrt) pyBuiltinsModule() *jsObject {
	if rt.pyBuiltinsMod == nil {
		o := newJSObject()
		o.set("__pymod", "builtins")
		rt.pyBuiltinsMod = o
	}
	return rt.pyBuiltinsMod
}

// pyIsModule says whether v is one of the module objects this runtime hands out.
func pyIsModule(v interface{}) (*jsObject, bool) {
	o, ok := v.(*jsObject)
	if !ok {
		return nil, false
	}
	if _, has := o.props["__pymod"]; has {
		return o, true
	}
	return nil, false
}

func init() {
	rxExtraExterns = append(rxExtraExterns, func(rt *jsrt, m map[string]func(args []uint64) uint64) {
		u := rt.unwrap
		w := rt.wrap
		m["js_pystaticmethod"] = func(a []uint64) uint64 { return w(pyStaticWrap(u(a[0]))) }
		m["js_pyclassmethod"] = func(a []uint64) uint64 { return w(pyClassWrap(u(a[0]))) }
		// property(...), bound as [positionals, keywords].
		m["js_pyproperty"] = func(a []uint64) uint64 {
			el := u(a[0]).(*jsArray).elems
			args := rt.pyKwPos("property", el[0].(*jsArray).elems, el[1], []string{"fget", "fset", "fdel", "doc"}, 0)
			return w(newPyProp(argAt(args, 0), argAt(args, 1), argAt(args, 2), argAt(args, 3)))
		}
		m["js_pybuiltins_module"] = func(a []uint64) uint64 { return w(rt.pyBuiltinsModule()) }
	})
}

// ----------------------------------------------------------- metaclasses ---
//
// A class a metaclass made carries it in "__meta" (pyTypeNew); type(C) answers
// it, isinstance(C, M) holds, an attribute C lacks is looked up on it, and an
// operator on the CLASS - iter(C), C[k], len(C), x in C, C(...), repr(C) - asks
// its methods. Only a metaclass's own methods count: type's are the default
// behaviour. Twins: pyMetaFind, pyMetaOf, pyTypeNew and pyTypeDelegate in
// languages/python-interpreter.abnf and lib/python-rt.metajs.

// pyClassMeta is the metaclass of a class a metaclass made, or nil.
func pyClassMeta(v interface{}) *jsObject {
	if c, ok := pyClassObj(v); ok {
		if m, ok := c.props["__meta"].(*jsObject); ok {
			return m
		}
	}
	return nil
}

// pyMetaFind is a class's metaclass method name, or nil.
func pyMetaFind(v interface{}, name string) interface{} {
	if m := pyClassMeta(v); m != nil {
		if f, found := pyLookup(m, name); found {
			return f
		}
	}
	return nil
}

// pyCheckHooked is the isinstance (hook "__instancecheck__") or issubclass
// ("__subclasscheck__") builtin: a target whose type defines the hook - a
// class's metaclass, or an instance's class - answers for itself (abc.ABCMeta's
// register() and typing's aliases rest on it), and a tuple asks each of its
// classes in turn. Twin of pyCheckHooked in
// python-interpreter.abnf and lib/python-rt.metajs.
func (rt *jsrt) pyCheckHooked(v, target interface{}, hook string) bool {
	if arr, ok := target.(*jsArray); ok {
		for _, e := range arr.elems {
			if rt.pyCheckHooked(v, e, hook) {
				return true
			}
		}
		return false
	}
	if m := pyMetaFind(target, hook); m != nil {
		return rt.truthy(rt.call(m, jsUndef, []interface{}{target, v}))
	}
	if _, cls, ok := pyInstance(target); ok {
		if m, found := pyLookup(cls, hook); found && isCallable(m) {
			return rt.truthy(rt.call(m, jsUndef, []interface{}{target, v}))
		}
	}
	if hook == "__instancecheck__" {
		return rt.pyIsInstance(v, target)
	}
	return rt.pyIsSubclass(v, target)
}

// pyIsMetaclass says whether cls derives from type.
func pyIsMetaclass(cls *jsObject) bool {
	for _, k := range pyMRO(cls) {
		if pyBuiltinClsName(k) == "type" {
			return true
		}
	}
	return false
}

// pyMetaOf is the metaclass a class statement uses: the metaclass= keyword,
// else the most derived metaclass among the bases; nil for type.
func (rt *jsrt) pyMetaOf(bases *jsArray, kw interface{}) interface{} {
	var meta interface{}
	if keys, vals, ok := dictParts(kw); ok {
		for i, k := range keys.elems {
			if rt.toString(k) == "metaclass" {
				meta = vals.elems[i]
			}
		}
	}
	for _, b := range bases.elems {
		bm := pyClassMeta(b)
		if bm == nil {
			continue
		}
		if mc, ok := meta.(*jsObject); !ok || pyIsSubclass(bm, mc) {
			meta = bm
		}
	}
	if m, ok := meta.(*jsObject); ok && pyBuiltinClsName(m) == "type" {
		return nil
	}
	return meta
}

// pyIsSubclass says whether sub has sup in its MRO.
func pyIsSubclass(sub, sup *jsObject) bool {
	for _, k := range pyMRO(sub) {
		if k == sup {
			return true
		}
	}
	return false
}

// pyTypeNew is type(name, bases, ns) and type.__new__(mcs, name, bases, ns,
// **kw): a namespace's __module__ and __qualname__ become the class's own, mcs,
// unless it is type itself, its metaclass, and kw goes to __init_subclass__.
func (rt *jsrt) pyTypeNew(name string, basesV, ns, mcs, kw interface{}) *jsObject {
	bases := &jsArray{elems: append([]interface{}{}, rt.pyElemsOf(basesV)...)}
	cls := rt.pyClassNew(name, bases)
	cls.set("__module__", "__main__")
	if m, ok := mcs.(*jsObject); ok && pyBuiltinClsName(m) != "type" {
		cls.set("__meta", m)
	}
	if keys, vals, ok := dictParts(ns); ok {
		for i, k := range keys.elems {
			switch ks := rt.toString(k); ks {
			case "__module__":
				cls.set("__module__", vals.elems[i])
			case "__qualname__":
				cls.set("__qual", vals.elems[i])
			case "__classcell__":
				cell, _ := vals.elems[i].(*jsObject)
				var sc *jsScope
				if cell != nil {
					sc, _ = cell.props["__pyccell"].(*jsScope)
				}
				if sc == nil {
					rt.pyRaise("TypeError", "__classcell__ must be a nonlocal cell, not <class '"+pyTypeName(rt, vals.elems[i])+"'>")
				}
				sc.put("__pycls__", cls)
			case "__classdictcell__":
				// CPython fills it with the class's namespace, which nothing here
				// reads back; only the cell check is observable.
				if cell, _ := vals.elems[i].(*jsObject); cell == nil || (cell.props["__pydcell"] != true && cell.props["__pyccell"] == nil) {
					rt.pyRaise("TypeError", "__classdictcell__ must be a nonlocal cell, not <class '"+pyTypeName(rt, vals.elems[i])+"'>")
				}
			case "__annotate_func__":
				// CPython calls it on the first __annotations__ read; here
				// type.__new__ calls it once, when the class is made.
				rt.pySetAttr(cls, ks, vals.elems[i])
				cls.set("__annotations__", rt.callPyValue(vals.elems[i], []interface{}{float64(1)}, jsUndef))
			default:
				rt.pySetAttr(cls, ks, vals.elems[i])
			}
		}
	}
	rt.pyEqHashNone(cls)
	rt.pySetNames(cls, kw)
	return cls
}

// pyClassCellCheck is CPython's check once a metaclass returned a class: the
// __classcell__ it was handed (now holding cell, false while empty) must name
// that class (builtin___build_class__). Twin of pyClassCellCheck in
// languages/python-interpreter.abnf and lib/python-rt.metajs.
func (rt *jsrt) pyClassCellCheck(name string, cls, cell interface{}) {
	if _, isCls := pyClassObj(cls); !isCls {
		return
	}
	if cell == false {
		rt.pyRaise("RuntimeError", "__class__ not set defining "+rt.pyRepr(name)+" as "+rt.pyRepr(cls)+
			". Was __classcell__ propagated to type.__new__?")
	}
	if cell != cls {
		rt.pyRaise("TypeError", "__class__ set to "+rt.pyRepr(cell)+" defining "+rt.pyRepr(name)+" as "+rt.pyRepr(cls))
	}
}

// pyTypeDelegate is type's own __new__, __init__ or __call__, reachable from a
// metaclass: super().__new__(mcs, name, bases, ns) makes the class and
// super().__call__(*args) an instance of it. nil for any other name.
func (rt *jsrt) pyTypeDelegate(name string) interface{} {
	switch name {
	case "__new__":
		return jsHostFunc("type.__new__", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			// A keyword call hands the dict over as one trailing argument
			// (pyBindCallR), with no pyKwMark before it.
			return rt.pyTypeNew(rt.toString(argAt(args, 1)), argAt(args, 2), argAt(args, 3), argAt(args, 0), argAt(args, 4))
		})
	case "__init__":
		return jsHostFunc("type.__init__", func(rt *jsrt, this uint64, args []interface{}) interface{} { return jsUndef })
	case "__call__":
		return jsHostFunc("type.__call__", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			args, kw := pyKwSplit(args)
			cls, _ := argAt(args, 0).(*jsObject)
			return rt.pyNewInstance(cls, pyArgsAfterFirst(args), kw)
		})
	}
	return nil
}

// pyKwSplit separates a host function's trailing pyKwMark and keyword dict.
func pyKwSplit(args []interface{}) ([]interface{}, interface{}) {
	if n := len(args); n >= 2 && args[n-2] == pyKwMark {
		return args[:n-2], args[n-1]
	}
	return args, jsUndef
}

// pyMetaRender is repr(C) or str(C) through C's metaclass, if it says.
func (rt *jsrt) pyMetaRender(v interface{}, repr bool) (string, bool) {
	f := pyMetaFind(v, "__repr__")
	if !repr {
		if s := pyMetaFind(v, "__str__"); s != nil {
			f = s
		}
	}
	if f == nil {
		return "", false
	}
	return rt.toString(rt.call(f, jsUndef, []interface{}{v})), true
}
