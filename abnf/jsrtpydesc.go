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
// off the module and a call made through it run the very same body.
func (rt *jsrt) pyModuleFn(mod *jsObject, name string) interface{} {
	mname := rt.toString(mod.props["__pymod"])
	h := jsHostFunc("pymod."+mname+"."+name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
		if m := pyExternMaps[rt]; m != nil {
			if f := m["js_pyrxmcall"]; f != nil {
				return rt.unwrap(f([]uint64{rt.wrap(mod), rt.wrap(name),
					rt.wrap(&jsArray{elems: args}), rt.wrap(jsUndef)}))
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
	if rt.toString(mod.props["__pymod"]) == "math" && pyMathHas(name) {
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
		m["js_pybuiltins_module"] = func(a []uint64) uint64 { return w(rt.pyBuiltinsModule()) }
	})
}
