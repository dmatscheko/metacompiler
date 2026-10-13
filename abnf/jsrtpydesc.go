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
// skipped, because layer 2's keysOf cannot list them. Twin of pyDSetNames in
// lib/python-rt.metajs and pySetNames in python-interpreter.abnf.
func (rt *jsrt) pySetNames(cls *jsObject) {
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
