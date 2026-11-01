package abnf

// jsrtruby.go -- the Ruby half of Kernel#format that abnf/jsrt.go's rubyFormat
// does not reach: the ARGUMENT SELECTORS, the %a/%A hex-float directive, and
// FloatDomainError on every Float -> Integer conversion.
//
// WHY IT IS A SEPARATE FILE. rubyFormat in abnf/jsrt.go renders one directive.
// What this file adds is not inside any directive - it is the scanner AROUND
// them, which decides WHICH argument a directive gets. So this file is that
// scanner, and it delegates each directive it has resolved back to
// rubyFormat as a single "%<flags><width>.<prec><conv>" with a single argument.
// No directive body is duplicated: %a is the only one rendered here, because it
// is the only one rubyFormat has no arm for.
//
// Registration is additive in the way abnf/jsrtdart.go and abnf/jsrtcsharp.go
// are: an init() appends a registrar to rxExtraExterns, which runs AFTER the base
// table is filled, and it rebinds only the four RUBY-ONLY externs below
// (js_rformat, js_rmod, js_rmcall, js_rxmcall), each of which keeps and calls the
// base binding for everything it does not itself answer. Manual chapter 4's
// warning is about wrapping a SHARED extern, and none of these four is shared.
//
// The twins are rformat / fmtOne / hexBody / floDomainGuard in
// languages/ruby-interpreter.abnf and rbFormat / rbHexBody / rbFloDomain in
// languages/lib/ruby-rt.metajs.

import (
	"fmt"
	"math"
	"math/big"
	"strings"
	"unicode/utf16"
)

// rubyIdentKind is the jsArray kind of the key array of a Hash after
// #compare_by_identity; rubyDictFind then compares keys with rubyIdentKeyEq.
// Twin of the __ident flag of ruby-interpreter.abnf and RB_IDENT_KIND in
// lib/ruby-rt.metajs.
const rubyIdentKind = 1

// rubyIdentKeyEq is equal? on two keys, except that two Strings are one key when
// their texts are equal: a String has no identity in this value model. Twin of
// core.identEq in ruby-interpreter.abnf.
func rubyIdentKeyEq(a, b interface{}) bool {
	if as, ok := a.(string); ok {
		bs, ok := b.(string)
		return ok && as == bs
	}
	return rubyIdenticalP(a, b)
}

func init() {
	// FloatDomainError < RangeError, so `rescue FloatDomainError' and
	// `rescue RangeError' both catch what rubyFloDomain raises. rubyExcParent is a
	// plain package var and rubyBuiltinClass makes a class object for any name, so
	// this is the whole registration.
	rubyExcParent["FloatDomainError"] = "RangeError"
	rubyExcParent["Math::DomainError"] = "StandardError"
	rubyExcParent["EncodingError"] = "StandardError"
	rubyExcParent["Encoding::CompatibilityError"] = "EncodingError"
	rubyExcParent["Encoding::UndefinedConversionError"] = "EncodingError"
	rubyExcParent["Encoding::InvalidByteSequenceError"] = "EncodingError"
	rubyExcParent["Encoding::ConverterNotFoundError"] = "EncodingError"
	// Enumerator::Lazy < Enumerator: not an exception, but the table is where a
	// builtin class's superclass and ancestors are read from by name.
	rubyExcParent["Enumerator::Lazy"] = "Enumerator"
	// The numeric classes are Numerics, and Numeric brings in Comparable.
	for _, n := range []string{"Integer", "Float", "Rational", "Complex"} {
		rubyExcParent[n] = "Numeric"
	}
	rxExtraExterns = append(rxExtraExterns, func(rt *jsrt, m map[string]func(args []uint64) uint64) {
		rt.addRubyFmtExterns(m)
	})
}

func (rt *jsrt) addRubyFmtExterns(m map[string]func(args []uint64) uint64) {
	u := rt.unwrap
	w := rt.wrap

	// Kernel#format / #sprintf / #printf: (arguments, the method's name). The answer
	// is a new String, so a box. Twin of js_rformat in lib/ruby-rt.metajs.
	m["js_rformat"] = func(a []uint64) uint64 {
		arr, ok := u(a[0]).(*jsArray)
		if !ok || len(arr.elems) == 0 {
			return w(&jsRStr{})
		}
		return w(&jsRStr{s: rt.rubyFormat2(rt.rubyStr(arr.elems[0]), &jsArray{elems: arr.elems[1:]}, rt.toString(u(a[1])))})
	}

	m["js_rrespond"] = func(a []uint64) uint64 {
		arr, _ := u(a[1]).(*jsArray)
		if arr == nil {
			arr = &jsArray{}
		}
		return w(rt.rubyRespond(u(a[0]), arr.elems))
	}

	// `class String ... end` reopening a builtin class: its methods go onto the
	// builtin's own class object, and from here on rubyMethod asks it.
	m["js_rbreopen"] = func(a []uint64) uint64 {
		rt.rubyBuiltinExt = true
		return jsHUndefined
	}

	// A bare `.length` / `.size`: a String, Array, Hash, Enumerator or Range counts
	// through js_pylen, anything else (a Symbol, MatchData, a user object) is the
	// method call. Twin of rlen in ruby-interpreter.abnf.
	pylen := m["js_pylen"]
	m["js_rlen"] = func(a []uint64) uint64 {
		v := u(a[0])
		// A size / length a program added to the builtin class wins (see rubyMethod).
		if rt.rubyBuiltinExt {
			if n, isStr := u(a[1]).(string); isStr {
				if _, found := rt.rubyBuiltinExtFind(v, n); found {
					return m["js_rxmcall"]([]uint64{a[0], a[1], w(&jsArray{})})
				}
			}
		}
		counted := false
		switch t := v.(type) {
		case *jsRStr:
			if t.bin != nil {
				return w(rt.rubyBinMethod(t.bin, "size", nil))
			}
			return w(float64(rubyCharCount(t.s)))
		case string, *jsArray:
			counted = true
		default:
			_, _, isDict := dictParts(v)
			_, isRng := rubyIsRange(v)
			// A deferred Enumerator's #size, and a Lazy's over a user #each, is
			// nil: rubyEnumMethod answers it.
			counted = isDict || isRng || (rubyIsEnum(v) && v.(*jsObject).props["__rdefer"] != true &&
				v.(*jsObject).props["kind"] != "each")
		}
		if counted {
			return pylen(a[:1])
		}
		return m["js_rxmcall"]([]uint64{a[0], a[1], w(&jsArray{})})
	}

	// String#% - the same formatter reached through the % operator. Every other
	// left operand is the base binding's modulo.
	baseMod := m["js_rmod"]
	m["js_rmod"] = func(a []uint64) uint64 {
		if s, ok := rubySV(u(a[0])).(string); ok {
			return w(&jsRStr{s: rt.rubyFormat2(s, u(a[1]), "%")})
		}

		return baseMod(a)
	}

	// Float#to_i and its five relatives. The guard has to sit in front of the
	// dispatcher rather than inside it, because rubyMethod's numeric arm is in
	// abnf/jsrt.go; it fires only for a non-finite Float receiver, so every other
	// call reaches the base binding after one type switch.
	for _, name := range []string{"js_rmcall", "js_rxmcall"} {
		base := m[name]
		call := func(a []uint64) uint64 {
			nm, args := rt.rubyRealCall(rt.toString(u(a[1])), u(a[2]))
			// A String box answers through rubyMethod, which reaches the
			// pattern-taking methods on its text itself; a plain string's sub! /
			// gsub! is nil when nothing matches.
			if _, isBox := u(a[0]).(*jsRStr); isBox {
				if arr, ok := u(a[2]).(*jsArray); ok {
					return w(rt.rubyMethod(u(a[0]), rt.toString(u(a[1])), arr.elems))
				}
			}
			if s, isStr := u(a[0]).(string); isStr && (nm == "sub!" || nm == "gsub!") {
				if arr, ok := args.(*jsArray); ok && rt.rubySubMisses(s, arr.elems) {
					return jsHNull
				}
			}
			// `Kernel.format(...)' with the module named explicitly does NOT go
			// through js_rformat - rubyMethod answers it off the Kernel builtin -
			// so it needs the same interception, or llvm.Run and the native binary
			// disagree on exactly the selectors this file adds.
			if rubyIsKernelObj(u(a[0])) && (nm == "format" || nm == "sprintf") {
				if arr, ok := args.(*jsArray); ok && len(arr.elems) > 0 {
					return rt.wrapStr(rt.rubyFormat2(rt.rubyStr(arr.elems[0]),
						&jsArray{elems: arr.elems[1:]}, nm))
				}
			}
			if v, done := rubyFloPredicate(u(a[0]), nm); done {
				return w(v)
			}
			rt.rubyFloDomainGuard(u(a[0]), nm, args)
			return base(a)
		}
		m[name] = func(a []uint64) uint64 {
			h := call(a)
			if v := u(h); v != nil {
				switch v.(type) {
				case string, *jsArray:
					return w(rubyBoxOut(u(a[0]), v))
				}
			}
			return h
		}
	}

	// A local read / write the emitter could not prove bound earlier in its frame.
	// A def does not see the locals around it, so the walk stops at the frame, the
	// scope holding "$blk" (the emitter's blkSlot). Past the frame only a callable -
	// a top-level def - still answers a read; anything else is MRI's NameError. A
	// write that finds no binding binds a[3] scopes up from where it stands, past
	// the begin/rescue/ensure closures Ruby has no scope for. Twins: js_rlvar /
	// js_rlset in languages/lib/ruby-rt.metajs. A slot holding undefined is a
	// local declared ahead of its first assignment (the emitter's frameLocalNew),
	// which a read before that assignment does not see.
	m["js_rlvar"] = func(a []uint64) uint64 {
		name := rt.toString(u(a[1]))
		inFrame := true
		for sc := rt.scopeOf(a[0]); sc != nil; sc = sc.parent {
			if v, ok := sc.get(name); ok && v != jsUndef {
				if inFrame || isCallable(v) {
					return w(v)
				}
				break
			}
			if sc.has("$blk") {
				inFrame = false
			}
		}
		self := rubyScopeSelf(rt, a[0])
		// A bare name that is no local is a call on self - one the emitter could
		// not see, as in a Proc that define_method runs with another self, which
		// reaches a private or protected method too.
		if o, isObj := self.(*jsObject); isObj && rubyRestricted(o.props["__class"], name) {
			return m["js_rxmcall"]([]uint64{w(self), rt.wrapStr("$q$" + name), w(&jsArray{})})
		}
		if o, isObj := self.(*jsObject); isObj && (rubySingRespond(o, name) || (name == "name" && o.props["__isclass"] == true)) {
			return m["js_rxmcall"]([]uint64{w(self), a[1], w(&jsArray{})})
		}
		// In a method a class_eval block added to a builtin class, self is a
		// builtin value and the name may be one of its methods (`upcase' in
		// String#shout). Twin of builtinSelfMethod in ruby-interpreter.abnf.
		if rt.rubyBuiltinExt && rt.rubyRespondClass(self) != "" && rubyTruthy(rt.rubyRespond(self, []interface{}{name})) {
			return m["js_rxmcall"]([]uint64{w(self), a[1], w(&jsArray{})})
		}
		rt.rubyNameErr("NameError", "undefined local variable or method `"+name+"' for "+rt.rubyNoMethodDesc(self), name, self)
		return 0
	}
	m["js_rloop"] = func(a []uint64) uint64 { return w(rt.rubyLoop(u(a[0]))) }
	m["js_rio"] = func(a []uint64) uint64 {
		o := newJSObject()
		o.set("__class", rt.rubyBuiltinClass("IO"))
		o.set("__rio", rt.toNumber(u(a[0])))
		return w(o)
	}
	// Object#method_missing, which a `super' in a method_missing reaches:
	// NoMethodError for the missing name. Twin of js_rnomethod in
	// lib/ruby-rt.metajs.
	m["js_rnomethod"] = func(a []uint64) uint64 {
		args, _ := u(a[1]).(*jsArray)
		name := ""
		if args != nil && len(args.elems) > 0 {
			name = rt.rubyStr(args.elems[0])
		}
		rt.rubyNoMethod(u(a[0]), name)
		return 0
	}
	// A super above a top-level def: NoMethodError. Twin of noSuper in
	// ruby-interpreter.abnf.
	m["js_rnosuper"] = func(a []uint64) uint64 {
		rt.rubyNoSuper(u(a[0]), rt.rubyStr(u(a[1])))
		return 0
	}
	// super: (class to start at, self, method key, args) - the method found up the
	// __super chain from there, called on self, or the NoMethodError of a super
	// with nothing above it. Twins: makeSuper in ruby-interpreter.abnf, js_rsuper
	// in lib/ruby-rt.metajs.
	m["js_rsuper"] = func(a []uint64) uint64 {
		name := rt.toString(u(a[2]))
		for cls := u(a[0]); ; {
			o, isObj := cls.(*jsObject)
			if !isObj {
				break
			}
			if f, has := o.props[name]; has && isCallable(f) {
				args, _ := u(a[3]).(*jsArray)
				return w(rt.call(f, jsUndef, append([]interface{}{u(a[1])}, args.elems...)))
			}
			cls = o.props["__super"]
		}
		args, _ := u(a[3]).(*jsArray)
		return w(rt.rubySuperBuiltin(u(a[1]), name, args.elems))
	}
	// An explicit-receiver call of a visibility directive: NoMethodError on a
	// class or module other than self, else the method key. Twins: visCheck in
	// ruby-interpreter.abnf, js_rvisdir in lib/ruby-rt.metajs.
	m["js_rvisdir"] = func(a []uint64) uint64 {
		recv := u(a[0])
		if o, isObj := recv.(*jsObject); isObj && o.props["__isclass"] == true && !rubyIdenticalP(recv, u(a[1])) {
			name := rt.rubyStr(u(a[2]))
			rt.rubyNameErr("NoMethodError", "private method `"+name+"' called for "+rt.rubyNoMethodDesc(recv), name, recv)
		}
		return a[2]
	}
	// The key a call on the implicit self dispatches by, where visibility can
	// change at run time (emitMKey in ruby-to-llvm-ir.abnf): rubySendKey. Twin of
	// js_rvkey in lib/ruby-rt.metajs.
	m["js_rvkey"] = func(a []uint64) uint64 {
		name := rt.toString(u(a[1]))
		if key := rubySendKey(u(a[0]), name); key != name {
			return rt.wrapStr(key)
		}
		return a[1]
	}
	m["js_rinclude"] = func(a []uint64) uint64 {
		if c, isObj := u(a[0]).(*jsObject); isObj {
			rubyInclude(c, u(a[1]))
		}
		return 0
	}
	m["js_rcatch"] = func(a []uint64) uint64 { return w(rt.rubyCatch(u(a[0]), u(a[1]))) }
	m["js_rthrow"] = func(a []uint64) uint64 {
		args, _ := u(a[0]).(*jsArray)
		var tag, v interface{} = jsNull, jsNull
		if args != nil {
			pos, _ := rubyBlockArgs(args.elems)
			if len(pos) > 0 {
				tag = pos[0]
			}
			if len(pos) > 1 {
				v = pos[1]
			}
		}
		rt.rubyThrow(tag, v)
		return 0
	}
	// send(:name, ...), with or without a receiver: a top-level def and a Kernel
	// function are MRI's private Object methods, so they answer when the receiver
	// has no method of the name. Twin of
	// js_rselfsend in languages/lib/ruby-rt.metajs and selfSendDef in
	// ruby-interpreter.abnf.
	m["js_rselfsend"] = func(a []uint64) uint64 {
		if arr, ok := u(a[2]).(*jsArray); ok && len(arr.elems) > 0 {
			name := rt.rubyStr(arr.elems[0])
			_, found := rubyFindMethod(u(a[1]), name)
			if cls := rt.rubyRespondClass(u(a[1])); cls != "" && rubyResponds(cls, name) {
				found = true
			}
			if !found {
				for sc := rt.scopeOf(a[0]); sc != nil; sc = sc.parent {
					if v, ok := sc.get("$d$" + name); ok && isCallable(v) {
						return w(rt.call(v, jsUndef, arr.elems[1:]))
					}
				}
				if rubyKernelMethodIndex(name) >= 0 {
					return w(rt.rubyKernelInvoke(name, arr.elems[1:]))
				}
			}
		}
		return w(rt.rubyMethod(u(a[1]), "send", u(a[2]).(*jsArray).elems))
	}
	// The callee of a bare `name(args)` that names no def: the chain walk of
	// js_scope_get, then a method of the implicit self the emitter could not see
	// (one a class_eval block defined, private ones included), with a miss raising
	// MRI's NoMethodError on the implicit self. Twin of js_rfn in
	// languages/lib/ruby-rt.metajs.
	m["js_rfn"] = func(a []uint64) uint64 {
		name := rt.toString(u(a[1]))
		for sc := rt.scopeOf(a[0]); sc != nil; sc = sc.parent {
			if v, ok := sc.get(name); ok {
				return w(v)
			}
		}
		self := rubyScopeSelf(rt, a[0])
		if o, isObj := self.(*jsObject); isObj {
			key := name
			if rubyRestricted(o.props["__class"], name) {
				key = "$q$" + name
			}
			if _, found := rubyFindMethod(o, key); found {
				return w(&hostFunc{name: "method:" + name, fn: func(rt *jsrt, this uint64, args []interface{}) interface{} {
					return rt.rubyMethod(o, key, args)
				}})
			}
		}
		rt.rubyNoMethod(self, name)
		return 0
	}
	// An @ivar write where self is a class object. Twin of js_rcivset in
	// languages/lib/ruby-rt.metajs.
	m["js_rcivset"] = func(a []uint64) uint64 {
		if o, ok := u(a[0]).(*jsObject); ok {
			rubyCivSet(o, rt.toString(u(a[1])), u(a[2]))
		}
		return 0
	}
	m["js_rlset"] = func(a []uint64) uint64 {
		name := rt.toString(u(a[1]))
		for sc := rt.scopeOf(a[0]); sc != nil; sc = sc.parent {
			if sc.has(name) {
				sc.put(name, u(a[2]))
				return 0
			}
			if sc.has("$blk") {
				break
			}
		}
		home := rt.scopeOf(a[0])
		for d := int(rt.toNumber(u(a[3]))); d > 0 && home.parent != nil; d-- {
			home = home.parent
		}
		home.put(name, u(a[2]))
		return 0
	}
	// obj.name = v where a setter may exist: (object, slot key, value, setter key).
	// The setter is called when the object (or its class, for a class object) has
	// one; an instance without one, a Hash, or a value that is not an object
	// (`5.x = 1'), takes the call anyway, which is MRI's NoMethodError (or its
	// method_missing); anything else has the slot written. Twin of js_rattrset in
	// languages/lib/ruby-rt.metajs and attrSet in ruby-interpreter.abnf.
	m["js_rattrset"] = func(a []uint64) uint64 {
		o, ok := u(a[0]).(*jsObject)
		if ok {
			_, isCls := o.props["__isclass"]
			_, _, isDict := dictParts(o)
			ok = !isDict && !rubySingRespond(o, rt.toString(u(a[3]))) && (!rubyUserObj(o) || isCls)
		}
		if !ok {
			m["js_rxmcall"]([]uint64{a[0], a[3], w(&jsArray{elems: []interface{}{u(a[2])}})})
			return 0
		}
		return m["js_set"](a[:3])
	}
	// js_rattrset with a fast path, emitted where no def or dynamic definition
	// names the setter, so only an attr_writer can answer it: (object, slot key,
	// value, setter key, marker key). A class holding a public attr_writer of the
	// name carries the marker (emitMethods); an instance whose class chain reaches
	// a marker before any other setter of the name has the slot written directly.
	// Twin of js_rattrw in languages/lib/ruby-rt.metajs.
	m["js_rattrw"] = func(a []uint64) uint64 {
		if o, ok := u(a[0]).(*jsObject); ok {
			mk := rt.toString(u(a[4]))
			sk := rt.toString(u(a[3]))
			cls, _ := o.props["__class"].(*jsObject)
			for d := 0; cls != nil && d < 64; d++ {
				if cls.props[mk] == true {
					return m["js_set"](a[:3])
				}
				if _, has := cls.props[sk]; has {
					break
				}
				cls, _ = cls.props["__super"].(*jsObject)
			}
		}
		return m["js_rattrset"](a[:4])
	}
	// for VARS in COLL: js_rforit opens the loop (an Enumerator runs over its #to_a,
	// see rubyForEnumVal), js_rfornext answers the next
	// element or undefined when the loop is done, js_rforend closes it (from the
	// loop's ensure). A Hash yields its live entries as [key, value] pairs inside
	// an iteration over it (rubyHashIter's count), so an insertion in the body
	// raises and an entry deleted before it is reached is skipped; an Array is
	// re-measured every pass; anything else is counted once by js_pylen and
	// indexed by js_pyget. Twin of forInHash in ruby-interpreter.abnf and
	// js_rforit in languages/lib/ruby-rt.metajs.
	m["js_rforit"] = func(a []uint64) uint64 {
		c := u(a[0])
		it := newJSObject()
		it.set("v", c)
		if rubyIsEnum(c) {
			it.set("v", rubyForEnumVal(c.(*jsObject)))
			c = rt.rubyForEnumItems(c.(*jsObject), len(a) > 1 && u(a[1]) == true)
		}
		it.set("c", c)
		it.set("i", float64(0))
		if d, isObj := c.(*jsObject); isObj {
			if _, _, isDict := dictParts(d); isDict {
				lev, _ := d.props["__iter"].(float64)
				d.set("__iter", lev+1)
				it.set("lev", lev)
				return w(it)
			}
		}
		if _, isArr := c.(*jsArray); !isArr {
			it.set("n", u(pylen(a[:1])))
		}
		return w(it)
	}
	m["js_rfornext"] = func(a []uint64) uint64 {
		it := u(a[0]).(*jsObject)
		c := it.props["c"]
		i := int(it.props["i"].(float64))
		if _, isHash := it.props["lev"]; isHash {
			keys, vals, _ := dictParts(c)
			if k, has := it.props["k"]; has && i < len(keys.elems) && keys.elems[i] == k {
				i++
			}
			if i >= len(keys.elems) {
				return jsHUndefined
			}
			it.set("i", float64(i))
			it.set("k", keys.elems[i])
			return w(&jsArray{elems: []interface{}{rt.rubyKeyOut(keys.elems[i]), vals.elems[i]}})
		}
		if arr, isArr := c.(*jsArray); isArr {
			if i >= len(arr.elems) {
				return jsHUndefined
			}
			it.set("i", float64(i+1))
			return w(arr.elems[i])
		}
		if float64(i) >= rubyToF(it.props["n"]) {
			return jsHUndefined
		}
		it.set("i", float64(i+1))
		return m["js_pyget"]([]uint64{w(c), w(float64(i))})
	}
	// js_rforval: the loop's value when it runs to its end - COLL, or for an
	// Enumerator rubyForEnumVal's answer.
	m["js_rforval"] = func(a []uint64) uint64 {
		return w(u(a[0]).(*jsObject).props["v"])
	}
	m["js_rforend"] = func(a []uint64) uint64 {
		it := u(a[0]).(*jsObject)
		if lev, isHash := it.props["lev"]; isHash {
			it.props["c"].(*jsObject).set("__iter", lev)
		}
		return jsHUndefined
	}
	// js_rsplat: the Array `*v' splices. Twin of splatSeq in
	// ruby-interpreter.abnf, which states the rule, and js_rsplat in
	// languages/lib/ruby-rt.metajs.
	m["js_rsplat"] = func(a []uint64) uint64 {
		v := u(a[0])
		if _, isArr := v.(*jsArray); isArr {
			return a[0]
		}
		if isUndefOrNull(v) {
			return w(&jsArray{})
		}
		if rubyIsEnum(v) || rt.rubyRespond(v, []interface{}{jsSym{s: "to_a"}}) == true {
			if r, isArr := rt.rubyMethod(v, "to_a", nil).(*jsArray); isArr {
				return w(r)
			}
		}
		return w(&jsArray{elems: []interface{}{v}})
	}
	// (proc, method twin): see needsMethTwin in ruby-to-llvm-ir.abnf. Twin of
	// js_rmtwin in languages/lib/ruby-rt.metajs.
	m["js_rmtwin"] = func(a []uint64) uint64 {
		if rt.rubyMTwins == nil {
			rt.rubyMTwins = map[interface{}]interface{}{}
		}
		rt.rubyMTwins[u(a[0])] = u(a[1])
		return 0
	}
	// A class first assigned to a constant takes its name (`S = Struct.new(:a)').
	// Twin of js_rconstname in languages/lib/ruby-rt.metajs and constName in
	// ruby-interpreter.abnf.
	m["js_rconstname"] = func(a []uint64) uint64 {
		if o, ok := u(a[0]).(*jsObject); ok {
			if _, isCls := o.props["__isclass"]; isCls && o.props["__name"] == "" {
				o.set("__name", rt.toString(u(a[1])))
			}
		}
		return a[0]
	}
}

// rubyStructNew is Struct.new(:a, :b [, keyword_init: true]): a new class, a
// subclass of Struct, with a reader and a writer per member and the Struct
// instance methods. The member values live in the hidden array slot __sv, in member
// order, so they are not instance variables (MRI: a Struct's `@a' is a different
// variable from its member `a', and #instance_variables does not list members).
// Twins: rbStructNew in languages/lib/ruby-rt.metajs, structNew in
// ruby-interpreter.abnf.
func (rt *jsrt) rubyStructNew(argv []interface{}) interface{} {
	pos, _ := rubyBlockArgs(argv)
	var names []string
	kwi := false
	for _, a := range pos {
		if keys, vals, isDict := dictParts(a); isDict {
			for i, k := range keys.elems {
				if s, isSym := k.(jsSym); isSym && s.s == "keyword_init" {
					kwi = rubyTruthy(vals.elems[i])
				}
			}
			continue
		}
		names = append(names, rt.rubyStr(a))
	}
	cls := newJSObject()
	cls.set("__isclass", true)
	cls.set("__name", "")
	cls.set("__super", rt.rubyBuiltinClass("Struct"))
	// putRaw's method sees the block marker among its arguments; put's does not.
	putRaw := func(n string, f func(self *jsObject, args []interface{}) interface{}) {
		cls.set(n, &hostFunc{name: "method:" + n, fn: func(rt *jsrt, this uint64, a []interface{}) interface{} {
			self, _ := a[0].(*jsObject)
			return f(self, a[1:])
		}})
		rubyIMNote(cls, n)
	}
	put := func(n string, f func(self *jsObject, args []interface{}) interface{}) {
		putRaw(n, func(self *jsObject, args []interface{}) interface{} {
			pa, _ := rubyBlockArgs(args)
			return f(self, pa)
		})
	}
	// The value array; an instance whose own initialize never reached Struct's has
	// every member nil.
	fresh := func(self *jsObject) *jsArray {
		arr := &jsArray{elems: make([]interface{}, len(names))}
		for i := range arr.elems {
			arr.elems[i] = jsNull
		}
		self.set("__sv", arr)
		return arr
	}
	sv := func(self *jsObject) *jsArray {
		if arr, ok := self.props["__sv"].(*jsArray); ok {
			return arr
		}
		return fresh(self)
	}
	vals := func(self *jsObject) *jsArray { return &jsArray{elems: append([]interface{}{}, sv(self).elems...)} }
	put("initialize", func(self *jsObject, args []interface{}) interface{} {
		vs := fresh(self)
		if kwi {
			if keys, kv, isDict := dictParts(argAt(args, 0)); isDict && len(args) > 0 {
				for i, k := range keys.elems {
					kn := rt.rubyStr(k)
					ix := strIndex(names, kn)
					if ix < 0 {
						rt.rubyRaiseErr("initialize", "ArgumentError", "unknown keywords: "+kn)
					}
					vs.elems[ix] = kv.elems[i]
				}
			}
		} else {
			if len(args) > len(names) {
				rt.rubyRaiseErr("initialize", "ArgumentError", "struct size differs")
			}
			copy(vs.elems, args)
		}
		return jsNull
	})
	for i, n := range names {
		i := i
		put(n, func(self *jsObject, args []interface{}) interface{} { return sv(self).elems[i] })
		put(n+"=", func(self *jsObject, args []interface{}) interface{} {
			sv(self).elems[i] = argAt(args, 0)
			return argAt(args, 0)
		})
	}
	members := func() interface{} {
		out := &jsArray{}
		for _, n := range names {
			out.elems = append(out.elems, jsSym{s: n})
		}
		return out
	}
	cls.set("$s$members", &hostFunc{name: "method:members", fn: func(rt *jsrt, this uint64, a []interface{}) interface{} { return members() }})
	put("members", func(self *jsObject, args []interface{}) interface{} { return members() })
	toA := func(self *jsObject, args []interface{}) interface{} { return vals(self) }
	put("to_a", toA)
	put("values", toA)
	size := func(self *jsObject, args []interface{}) interface{} { return float64(len(names)) }
	put("size", size)
	put("length", size)
	put("to_h", func(self *jsObject, args []interface{}) interface{} {
		d := rubyNewDict()
		keys, dv, _ := dictParts(d)
		for i, n := range names {
			dictAppend(keys, dv, jsSym{s: n}, sv(self).elems[i])
		}
		return d
	})
	same := func(self *jsObject, o interface{}) (*jsObject, bool) {
		oo, isObj := o.(*jsObject)
		return oo, isObj && oo.props["__class"] == self.props["__class"]
	}
	put("==", func(self *jsObject, args []interface{}) interface{} {
		o, ok := same(self, argAt(args, 0))
		return ok && rt.rubyEq(vals(self), vals(o))
	})
	put("eql?", func(self *jsObject, args []interface{}) interface{} {
		o, ok := same(self, argAt(args, 0))
		return ok && rt.rubyEqlDeep(vals(self), vals(o))
	})
	// The class name takes part, so two Struct classes' equal values hash apart.
	put("hash", func(self *jsObject, args []interface{}) interface{} {
		return rt.rubyHashOf(&jsArray{elems: append([]interface{}{rubyInstClassName(self)}, sv(self).elems...)})
	})
	put("[]", func(self *jsObject, args []interface{}) interface{} {
		return sv(self).elems[rt.rubyStructSlot(names, argAt(args, 0))]
	})
	put("[]=", func(self *jsObject, args []interface{}) interface{} {
		sv(self).elems[rt.rubyStructSlot(names, argAt(args, 0))] = argAt(args, 1)
		return argAt(args, 1)
	})
	// each is Array#each over the values, answering the struct itself when it ran
	// a block.
	putRaw("each", func(self *jsObject, args []interface{}) interface{} {
		r := rt.rubyMethod(vals(self), "each", args)
		if _, blk := rubyBlockArgs(args); blk == nil {
			return r
		}
		return self
	})
	// values_at: a position must name a member (IndexError otherwise, as #[]
	// says), and a Range selects as Array#values_at does.
	put("values_at", func(self *jsObject, args []interface{}) interface{} {
		out := &jsArray{}
		for _, k := range args {
			if _, isNum := k.(float64); isNum {
				out.elems = append(out.elems, sv(self).elems[rt.rubyStructSlot(names, k)])
			} else if sel, isArr := rt.rubyMethod(vals(self), "values_at", []interface{}{k}).(*jsArray); isArr {
				out.elems = append(out.elems, sel.elems...)
			}
		}
		return out
	})
	// dig: the member (nil for an unknown one or a position out of range), then
	// each further key into an Array / Hash, or through the value's own #dig.
	put("dig", func(self *jsObject, args []interface{}) interface{} {
		var cur interface{} = jsNull
		if f, isNum := argAt(args, 0).(float64); isNum {
			if i := int(f); i < 0 && i+len(names) >= 0 {
				cur = sv(self).elems[i+len(names)]
			} else if i >= 0 && i < len(names) {
				cur = sv(self).elems[i]
			}
		} else if ix := strIndex(names, rt.rubyStr(argAt(args, 0))); ix >= 0 {
			cur = sv(self).elems[ix]
		}
		for i := 1; i < len(args); i++ {
			if isUndefOrNull(cur) {
				return jsNull
			}
			if rubyUserObj(cur) {
				return rt.rubyMethod(cur, "dig", args[i:])
			}
			_, _, isDict := dictParts(cur)
			if _, isArr := cur.(*jsArray); !isArr && !isDict {
				rt.rubyRaiseErr("dig", "TypeError", rt.rubyRespondClass(cur)+" does not have #dig method")
			}
			cur = rt.rubyIndexGet(cur, args[i])
		}
		return cur
	})
	text := func(self *jsObject, args []interface{}) interface{} {
		var sb strings.Builder
		sb.WriteString("#<struct ")
		if cn := rubyInstClassName(self); cn != "" {
			sb.WriteString(cn + " ")
		}
		for i, n := range names {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(n + "=" + rt.rubyInspect(sv(self).elems[i]))
		}
		sb.WriteString(">")
		return sb.String()
	}
	put("inspect", text)
	put("to_s", text)
	return cls
}

// rubyStructSlot is a Struct subscript: a member position (negative from the
// end) or a member name.
func (rt *jsrt) rubyStructSlot(names []string, k interface{}) int {
	if f, isNum := k.(float64); isNum {
		i := int(f)
		if i < 0 {
			i += len(names)
		}
		if i < 0 {
			rt.rubyRaiseErr("[]", "IndexError", "offset "+rubyIntStr(f)+" too small for struct(size:"+rubyIntStr(float64(len(names)))+")")
		}
		if i >= len(names) {
			rt.rubyRaiseErr("[]", "IndexError", "offset "+rubyIntStr(f)+" too large for struct(size:"+rubyIntStr(float64(len(names)))+")")
		}
		return i
	}
	ix := strIndex(names, rt.rubyStr(k))
	if ix < 0 {
		rt.rubyRaiseErr("[]", "NameError", "no member '"+rt.rubyStr(k)+"' in struct")
	}
	return ix
}

func strIndex(xs []string, s string) int {
	for i, x := range xs {
		if x == s {
			return i
		}
	}
	return -1
}

func rubyIsKernelObj(v interface{}) bool {
	o, ok := v.(*jsObject)
	if !ok {
		return false
	}
	bn, isBuiltin := o.props["__rbuiltin"].(string)
	return isBuiltin && bn == "Kernel"
}

// rubyRealCall unwraps #send / #__send__ / #public_send, which re-enter the
// dispatcher with the real method name one argument along. The two hooks below
// sit OUTSIDE the dispatcher, so they have to do that themselves.
func (rt *jsrt) rubyRealCall(name string, args interface{}) (string, interface{}) {
	arr, _ := args.(*jsArray)
	for (name == "send" || name == "__send__" || name == "public_send") && arr != nil && len(arr.elems) > 0 {
		name = rt.rubyStr(arr.elems[0])
		arr = &jsArray{elems: arr.elems[1:]}
		args = arr
	}
	return name, args
}

// Float#nan? / #infinite? / #finite?. #infinite? answers 1, -1 or NIL and not a
// boolean. The twins are numberMethod in languages/ruby-interpreter.abnf and
// rbNumMethod in languages/lib/ruby-rt.metajs.
// Integer#fdiv, exact for a big divisor and past the double range: the quotient is formed as an integer
// of about 70 bits, rounded once and scaled by a power of two. Twin of intFdiv in
// languages/ruby-interpreter.abnf (the reasoning is there) and rbIntFdiv in
// languages/lib/ruby-rt.metajs; the digit counts choose the same scale in all three.
func rubyIntFdiv(a, b interface{}) float64 {
	x, y := rubyToF(a), rubyToF(b)
	bi, isInt := rubyBigOf(b)
	_, bigB := b.(*jsBigInt)
	if (!math.IsInf(x, 0) && !math.IsInf(y, 0) && !bigB) || !isInt || y == 0 {
		return x / y
	}
	ai, _ := rubyBigOf(a)
	pa, pb := new(big.Int).Abs(ai), new(big.Int).Abs(bi)
	s := int(math.Ceil(float64(len(pb.String())-len(pa.String()))*3.33)) + 70
	q := new(big.Int)
	if s >= 0 {
		q.Quo(new(big.Int).Lsh(pa, uint(s)), pb)
	} else {
		q.Quo(pa, new(big.Int).Lsh(pb, uint(-s)))
	}
	r, _ := new(big.Float).SetInt(q).Float64()
	// Halving is exact while the result stays normal; the rest is one multiply by
	// an exact power of two, so a subnormal answer rounds once.
	k := s
	for ; k > 0 && r/2 >= 0x1p-1022; k-- {
		r = r / 2
	}
	if k > 0 {
		pk := 1.0
		for ; k > 0; k-- {
			pk = pk / 2
		}
		r = r * pk
	}
	for ; k < 0; k++ {
		r = r * 2
	}
	if (ai.Sign() < 0) != (bi.Sign() < 0) {
		return -r
	}
	return r
}

// rubyFloPredicate is #nan? / #finite? / #infinite?. Only a Float has #nan?; an
// Integer and a Rational are always finite, and a Complex is finite when both of
// its parts are. #infinite? answers 1, -1 or nil. Twins: numberMethod in
// ruby-interpreter.abnf and rbNumMethod in lib/ruby-rt.metajs.
func rubyFloPredicate(target interface{}, name string) (interface{}, bool) {
	if name != "nan?" && name != "infinite?" && name != "finite?" {
		return nil, false
	}
	var x float64
	switch t := target.(type) {
	case jsFlo:
		x = t.f
	case float64, *jsBigInt, jsRat:
		if name == "nan?" {
			return nil, false
		}
	case jsCpx:
		if name == "nan?" {
			return nil, false
		}
		re, okRe := rubyFloPredicate(rubyReOf(t), "infinite?")
		im, okIm := rubyFloPredicate(rubyImOf(t), "infinite?")
		inf := (okRe && !isUndefOrNull(re)) || (okIm && !isUndefOrNull(im))
		if name == "finite?" {
			return !inf, true
		}
		if inf {
			return float64(1), true
		}
		return jsNull, true
	default:
		return nil, false
	}
	switch name {
	case "nan?":
		return math.IsNaN(x), true
	case "finite?":
		return !math.IsNaN(x) && !math.IsInf(x, 0), true
	}
	if math.IsInf(x, 1) {
		return float64(1), true
	}
	if math.IsInf(x, -1) {
		return float64(-1), true
	}
	return jsNull, true
}

// rubyFloDomainGuard raises FloatDomainError when a non-finite Float is asked for
// an Integer. MRI's #round / #floor / #ceil with a POSITIVE ndigits answer a Float
// and do not raise, which is why the no-argument form is the one that is guarded.
func (rt *jsrt) rubyFloDomainGuard(target interface{}, name string, args interface{}) {
	f, ok := target.(jsFlo)
	if !ok {
		return
	}
	if !math.IsNaN(f.f) && !math.IsInf(f.f, 0) {
		return
	}
	switch name {
	case "to_i", "to_int", "truncate":
	// #floor and #ceil take an ndigits too, and a POSITIVE one answers the
	// receiver rather than raising - exactly as #round does
	// (Float::INFINITY.floor(2) is Infinity). The ndigits <= 0 case still
	// raises: rubyNumDigitOp does it, on the value it is about to convert.
	case "floor", "ceil", "round":
		if arr, isArr := args.(*jsArray); isArr && len(arr.elems) > 0 {
			return
		}
	default:
		return
	}
	rubyFloDomain(rt, f.f)
}

// rubyFloDomain is the raise itself, with the float's own #to_s as the message.
func rubyFloDomain(rt *jsrt, x float64) {
	if math.IsNaN(x) {
		rt.rubyRaiseErr("", "FloatDomainError", "NaN")
	}
	if math.IsInf(x, 1) {
		rt.rubyRaiseErr("", "FloatDomainError", "Infinity")
	}
	if math.IsInf(x, -1) {
		rt.rubyRaiseErr("", "FloatDomainError", "-Infinity")
	}
}

// ===== the argument selectors =====
//
// A directive does not have to take the next argument, and MRI offers four ways
// of saying which one it takes (`"%1$s-%2$s" % ["a","b"]' is "a-b"):
//
//	%N$s          the Nth argument, 1-based, in any order and reusable
//	%<name>s      the Hash entry :name, formatted by the directive
//	%{name}       the Hash entry :name, to_s, with NO conversion letter
//	%*d / %.*f    the width / precision comes from an argument; a NEGATIVE *
//	              width left-justifies, a negative * precision means none
//
// The `*` argument is read BEFORE the value and width before precision, and `*`
// takes an explicit selector too ("%1$*2$d" % [7, 6] is "     7").
//
// "TOO FEW ARGUMENTS" IS RAISED ("%s %s" % ["a"]); the full row table, and the
// shapes that do NOT raise, are at fmtTooFew in languages/ruby-interpreter.abnf.
// rbFmtTooFew in languages/lib/ruby-rt.metajs is the other twin. Not
// implemented: MRI's four ArgumentErrors for MIXING the numbered, unnumbered and
// named styles.
// rubyFmtCheckConv is MRI's malformed-format-string rule (ruby 2.6.10; sprintf
// is unchanged in 3.x):
//
//	"%z" % 5     MRI: malformed format string - %z
//	"%5" % 5     MRI: malformed format string - %*[0-9]
//	"100%" % []  MRI: incomplete format specifier…
//
// The messages are MRI's own, chosen by what the head consumed last - see the
// table in languages/ruby-interpreter.abnf.
//
// The one row a selector still saves in MRI and not here is "%1$%" % 5, which is
// "%" there and *invalid format character - %* here; it never reaches the
// argument list, so it is a malformed-head row.
func rubyIsFmtConv(c byte) bool {
	switch c {
	case 'd', 'i', 'u', 'f', 's', 'p', 'c', 'x', 'X', 'o', 'b', 'B', 'e', 'E', 'g', 'G', 'a', 'A':
		return true
	}
	return false
}

func (rt *jsrt) rubyFmtTooFew() {
	rt.rubyRaiseErr(rt.rubyFmtFrame, "ArgumentError", "too few arguments")
}

func (rt *jsrt) rubyFmtCheckConv(c byte) {
	if rubyIsFmtConv(c) {
		return
	}
	// % as the conversion of a non-empty head ("%5%"), and a newline or a NUL that
	// is NOT directly after the % ("%5\n"), are MRI's one message.
	if c == '%' || c == '\n' || c == 0 {
		rt.rubyRaiseErr(rt.rubyFmtFrame, "ArgumentError", "invalid format character - %")
	}
	if c >= 32 && c < 127 {
		rt.rubyRaiseErr(rt.rubyFmtFrame, "ArgumentError", "malformed format string - %"+string(c))
	}
	rt.rubyRaiseErr(rt.rubyFmtFrame, "ArgumentError", "malformed format string")
}

func (rt *jsrt) rubyFormat2(spec string, arg interface{}, frame string) string {
	outerFrame := rt.rubyFmtFrame
	rt.rubyFmtFrame = frame
	// A format argument is read, never kept, so a box is read as its text.
	args := []interface{}{rubySV(arg)}
	if a, ok := arg.(*jsArray); ok {
		args = rubySVArgs(a.elems)
	}
	ai := 0
	// The next positional argument. Running off the end is MRI's ArgumentError,
	// not a nil - see rubyFmtTooFew's note above.
	take := func() interface{} {
		if ai >= len(args) {
			rt.rubyFmtTooFew()
		}
		v := args[ai]
		ai++
		return v
	}
	// The value a `*` reads: the next positional argument, or the explicit one of
	// a "*2$" form. Answers the integer and the position just past what it ate.
	star := func(j int) (int, int) {
		ds, dn := j, 0
		for j < len(spec) && spec[j] >= '0' && spec[j] <= '9' {
			dn = dn*10 + int(spec[j]-'0')
			j++
		}
		var v interface{} = jsNull
		if j > ds && j < len(spec) && spec[j] == '$' {
			j++
			if dn == 0 || dn-1 >= len(args) {
				rt.rubyFmtTooFew()
			}
			v = args[dn-1]
		} else {
			j = ds
			v = take()
		}
		return int(math.Trunc(rubyToF(v))), j
	}
	var out strings.Builder
	for i := 0; i < len(spec); {
		if spec[i] != '%' {
			out.WriteByte(spec[i])
			i++
			continue
		}
		// A MALFORMED directive is an ArgumentError in MRI. rubyFmtCheckConv
		// carries the rule.
		if i+1 >= len(spec) {
			rt.rubyRaiseErr(rt.rubyFmtFrame, "ArgumentError", "incomplete format specifier; use %% (double %) instead")
		}
		if spec[i+1] == '%' {
			out.WriteString("%")
			i += 2
			continue
		}
		// A newline or a NUL DIRECTLY after the % is the one shape MRI writes out
		// instead of raising - "%\n" is "%\n" - and only there: "%5\n" raises.
		if spec[i+1] == '\n' || spec[i+1] == 0 {
			out.WriteString("%")
			i++
			continue
		}
		j := i + 1
		flags := ""
		width, prec, sel := 0, -1, -1
		name := ""
		haveName, brace, bad := false, false, false
		// What the directive head consumed last, which is what MRI's message at
		// the end of the string is chosen by: 1 flags, 2 a number (a width, a
		// precision or a *), 3 an argument SELECTOR or a <name>.
		tail := 0
		// ONE loop over the whole directive head, because the pieces may come in
		// any order: "% 1$d" and "%1$ d" are both " 5" in MRI.
		for j < len(spec) {
			c := spec[j]
			if c == '-' || c == '0' || c == '+' || c == ' ' || c == '#' {
				flags += string(c)
				j++
				tail = 1
				continue
			}
			if c == '<' || c == '{' {
				close := byte('>')
				if c == '{' {
					close = '}'
				}
				e := j + 1
				for e < len(spec) && spec[e] != close {
					e++
				}
				if e >= len(spec) {
					bad = true
					break
				}
				name, haveName = spec[j+1:e], true
				j = e + 1
				tail = 3
				if c == '{' {
					brace = true
					break
				}
				continue
			}
			if c == '*' {
				n, nj := star(j + 1)
				j = nj
				tail = 2
				if n < 0 {
					flags += "-"
					width = -n
				} else {
					width = n
				}
				continue
			}
			if c == '.' {
				if prec >= 0 {
					rt.rubyRaiseErr(rt.rubyFmtFrame, "ArgumentError", "precision given twice")
				}
				j++
				tail = 2
				if j < len(spec) && spec[j] == '*' {
					n, nj := star(j + 1)
					j = nj
					prec = n
					if n < 0 {
						prec = -1
					}
					continue
				}
				prec = 0
				for j < len(spec) && spec[j] >= '0' && spec[j] <= '9' {
					prec = prec*10 + int(spec[j]-'0')
					j++
				}
				continue
			}
			if c >= '1' && c <= '9' {
				// A leading 0 was taken as the zero FLAG above, so a run of digits
				// here is a width - unless a "$" makes it an argument selector.
				dn := 0
				for j < len(spec) && spec[j] >= '0' && spec[j] <= '9' {
					dn = dn*10 + int(spec[j]-'0')
					j++
				}
				if j < len(spec) && spec[j] == '$' {
					sel = dn - 1
					j++
					tail = 3
					continue
				}
				width = dn
				tail = 2
				continue
			}
			break
		}
		conv := byte('s')
		if !brace {
			if bad {
				rt.rubyRaiseErr(rt.rubyFmtFrame, "ArgumentError", "malformed name - unmatched parenthesis")
			}
			if j >= len(spec) {
				if tail == 2 {
					rt.rubyRaiseErr(rt.rubyFmtFrame, "ArgumentError", "malformed format string - %*[0-9]")
				}
				if tail != 3 {
					rt.rubyRaiseErr(rt.rubyFmtFrame, "ArgumentError", "invalid format character - %")
				}
				// A selector or a <name> that ran off the end writes a bare "%"
				// in MRI ("%1$" % 5 is "%"), but the name is still LOOKED UP -
				// "%<x>" % 5 is *one hash required*.
				if haveName {
					rt.rubyFmtNamed(args, name, "<", ">")
				}
				out.WriteString("%")
				break
			}
			conv = spec[j]
			j++
		}
		var av interface{} = jsNull
		switch {
		case haveName:
			open, close := "<", ">"
			if brace {
				open, close = "{", "}"
			}
			av = rt.rubyFmtNamed(args, name, open, close)
		case sel >= 0:
			if sel >= len(args) {
				rt.rubyFmtTooFew()
			}
			av = args[sel]
		default:
			av = take()
		}
		// AFTER the argument, because MRI resolves the name first: "%<x>z" % 5 is
		// *one hash required* and not *malformed format string - %z*.
		if !brace {
			rt.rubyFmtCheckConv(conv)
		}
		// An infinity or a NaN reaching an INTEGER directive raises, because MRI's
		// %d/%x/%o/%b convert the operand with to_i (and Go's int64 saturation would
		// be ARCHITECTURE-DEPENDENT - manual 7.9's giFromFloat note).
		switch conv {
		case 'd', 'i', 'u', 'x', 'X', 'o', 'b', 'B':
			// The OPERAND CHECK comes first: MRI raises TypeError on nil before it
			// can ever reach the to_i that raises FloatDomainError, and this
			// pre-pass runs ahead of rubyFormat's own arm. rubyFmtIntOperand is in
			// abnf/jsrt.go beside rubyFormat.
			av = rt.rubyFmtIntOperand(av)
			if _, isBig := av.(*jsBigInt); !isBig {
				rubyFloDomain(rt, math.Trunc(rubyToF(av)))
			}
		}
		if conv == 'a' || conv == 'A' {
			// %a is the one directive rendered in this file, so it needs the operand
			// check here rather than in rubyFormat: "%a" % [nil] is MRI's
			// *can't convert nil into Float*.
			out.WriteString(rubyHexDirective(rt.rubyFmtFloOperand(av), flags, width, prec, conv == 'A'))
			i = j
			continue
		}
		head := "%" + flags
		if width > 0 {
			head += itoaPos(width)
		}
		if prec >= 0 {
			head += "." + itoaPos(prec)
		}
		// One directive, one argument: the argument is WRAPPED in a one-element
		// array, because rubyFormat reads a bare *jsArray as the argument LIST and
		// `"%s" % [[1, 2]]' would otherwise print the inner array's first element.
		out.WriteString(rt.rubyFormat(head+string(conv), &jsArray{elems: []interface{}{av}}))
		i = j
	}
	rt.rubyFmtFrame = outerFrame
	return out.String()
}

func itoaPos(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// rubyFmtNamed resolves %<name> / %{name} against the one Hash the right operand
// has to be. A SCAN and not rubyDictFind: the key is the Symbol :name, and a
// String key does NOT match - MRI's `"%{x}" % {"x" => 1}' is a KeyError.
func (rt *jsrt) rubyFmtNamed(args []interface{}, name, open, close string) interface{} {
	var h *jsObject
	if len(args) == 1 {
		if o, ok := args[0].(*jsObject); ok {
			if d, isDict := o.props["__dict"].(bool); isDict && d {
				h = o
			}
		}
	}
	if h == nil {
		rt.rubyRaiseErr(rt.rubyFmtFrame, "ArgumentError", "one hash required")
	}
	keys, _ := h.props["keys"].(*jsArray)
	vals, _ := h.props["vals"].(*jsArray)
	if keys != nil && vals != nil {
		for i, k := range keys.elems {
			if s, ok := k.(jsSym); ok && s.s == name && i < len(vals.elems) {
				return vals.elems[i]
			}
		}
	}
	rt.rubyRaiseErr(rt.rubyFmtFrame, "KeyError", "key"+open+name+close+" not found")
	return jsNull
}

// ===== %a / %A =====
//
// MRI writes a finite double as [-]0x1.<hex fraction>p<+|-><decimal binary
// exponent>, normalised so the leading digit is 1 for every non-zero value - a
// SUBNORMAL is normalised too ("%a" % 5.0e-324 is "0x1p-1074") - and zero is
// "0x0p+0". Without a precision the fraction's trailing zeros go; with one it is
// rounded half to even, or zero-padded, to exactly that many digits.
//
// Written on the same pure ARITHMETIC as the other two engines rather than on
// math.Frexp or strconv's 'x' format, deliberately: this is the twin the two
// MetaJS halves have to agree with digit for digit, and the cheapest way to be
// sure of that is to run the same algorithm. Scaling a double by a power of two
// is exact as long as the result neither overflows nor falls into the subnormals,
// and both loops are driven so that it never does.
//
// ONE DELIBERATE DIVERGENCE FROM THE ORACLE, and MRI 2.6.10 is the one that is
// wrong: at a precision of EXACTLY 13 - the full digit count, where no rounding
// can be needed - it rounds the last digit to even anyway. "%.13a" % (1.0/3) is
// 0x1.5555555555554p-2 there against the value's own 0x1.5555555555555p-2. C's
// printf, python3's float.hex and MRI's own %a with no precision
// all print the exact digits, and MRI is right again at 12 and at 14.
func rubyHexNorm(x float64) (float64, int) {
	pv, pk := []float64{}, []int{}
	for v, k := 2.0, 1; k <= 512; v, k = v*v, k*2 {
		pv = append(pv, v)
		pk = append(pk, k)
	}
	e := 0
	for i := len(pv) - 1; i >= 0; i-- {
		for x >= pv[i] {
			x /= pv[i]
			e += pk[i]
		}
	}
	for i := len(pv) - 1; i >= 0; i-- {
		for x*pv[i] < 2 {
			x *= pv[i]
			e -= pk[i]
		}
	}
	return x, e
}

// rubyHexBody is the finite-value body WITHOUT the sign - "0x1.8p+0". prec < 0
// means "as many digits as the value needs".
func rubyHexBody(fv float64, prec int, alt, upper bool) string {
	lead, frac, ex := "1", "", 0
	if fv != 0 {
		if fv < 0 {
			fv = -fv
		}
		m, e := rubyHexNorm(fv)
		ex = e
		// The 13 hex digits of m - 1: 13 x 4 is exactly the 52 fraction bits of a
		// double, so nothing is lost (m - 1 is exact and every *16 is a scale by a
		// power of two).
		d := make([]int, 13)
		f := m - 1
		for i := 0; i < 13; i++ {
			f *= 16
			d[i] = int(math.Floor(f))
			f -= float64(d[i])
		}
		n := 13
		if prec >= 0 && prec < 13 {
			up := false
			if d[prec] > 8 {
				up = true
			} else if d[prec] == 8 {
				rest := false
				for q := prec + 1; q < 13; q++ {
					if d[q] != 0 {
						rest = true
					}
				}
				last := 1
				if prec > 0 {
					last = d[prec-1]
				}
				up = rest || last%2 == 1
			}
			n = prec
			if up {
				c := n - 1
				for c >= 0 && d[c] == 15 {
					d[c] = 0
					c--
				}
				if c < 0 {
					ex++
				} else {
					d[c]++
				}
			}
		}
		for k := 0; k < n; k++ {
			frac += string("0123456789abcdef"[d[k]])
		}
		if prec < 0 {
			frac = strings.TrimRight(frac, "0")
		} else {
			for len(frac) < prec {
				frac += "0"
			}
		}
	} else {
		lead = "0"
	}
	if prec > 0 && frac == "" {
		frac = strings.Repeat("0", prec)
	}
	out := "0x" + lead
	if frac != "" {
		out += "." + frac
	} else if alt {
		out += "."
	}
	sign, ae := "+", ex
	if ex < 0 {
		sign, ae = "-", -ex
	}
	out += "p" + sign + itoaPos(ae)
	if upper {
		return strings.ToUpper(out)
	}
	return out
}

// rubyHexDirective is one whole %a/%A field: the body, the sign, the + / space
// flags and the padding. %a is a NUMERIC directive - it takes a sign and honours
// the 0 flag - and the zeros go behind BOTH the sign and the "0x", which is what
// `keep' counts: "%020a" % -0.0 is "-0x00000000000000p+0".
func rubyHexDirective(f float64, flags string, width, prec int, upper bool) string {
	left := strings.Contains(flags, "-")
	zero := strings.Contains(flags, "0")
	plus := strings.Contains(flags, "+")
	space := strings.Contains(flags, " ")
	alt := strings.Contains(flags, "#")
	body, keep, nonFinite := "", 0, false
	if w, bad := rubyFloatWord(f); bad {
		body, nonFinite = w, true
	} else {
		body = rubyHexBody(f, prec, alt, upper)
		keep = 2
		if f < 0 || (f == 0 && math.Signbit(f)) {
			body, keep = "-"+body, 3
		}
	}
	if !strings.HasPrefix(body, "-") {
		if plus {
			body, keep = "+"+body, keep+1
		} else if space {
			body, keep = " "+body, keep+1
		}
	}
	pad := " "
	if zero && !left && !nonFinite {
		pad = "0"
	}
	if nonFinite {
		keep = 0
	}
	for rubyCharCount(body) < width {
		if left {
			body += " "
		} else if pad != " " {
			body = body[:keep] + pad + body[keep:]
		} else {
			body = pad + body
		}
	}
	return body
}

// ===== Numeric#div / #divmod / #modulo / #rationalize =====
//
// Twin of numDivOp in languages/ruby-interpreter.abnf, which states the rules,
// and rbNumDivOp in languages/lib/ruby-rt.metajs.

func (rt *jsrt) rubyNumDivOp(t interface{}, name string, args []interface{}) interface{} {
	y := argAt(args, 0)
	switch name {
	case "rationalize":
		return rt.rubyRationalize(t, args)
	case "modulo":
		return rt.rubyBinAs(name, "%", t, y)
	}
	_, tf := t.(jsFlo)
	_, yf := y.(jsFlo)
	if name == "divmod" && (tf || yf) {
		return rt.rubyFloDivmod(rubyToF(t), rubyToF(y))
	}
	if rubyToF(y) == 0 {
		rt.rubyRaiseErr(name, "ZeroDivisionError", "divided by 0")
	}
	q := rt.rubyNumBin("/", t, y)
	switch qv := q.(type) {
	case jsFlo:
		rubyFloDomain(rt, qv.f)
		q = rt.rubyIntFromF(math.Floor(qv.f))
	case jsRat:
		q = rt.rubyRatDigitOp(qv, 'f', 0)
	}
	if name == "div" {
		return q
	}
	return &jsArray{elems: []interface{}{q, rt.rubyNumBin("-", t, rt.rubyNumBin("*", y, q))}}
}

// rubyBinAs is rubyBin for a builtin METHOD that answers through an operator
// (Integer#modulo through %): an error the operator raises is reported in the
// method's own frame, as MRI does - `5.modulo(0)' raises in `modulo', not `%'.
// Only an error that names its frame is renamed. Twins: binAs in
// ruby-interpreter.abnf, rbBinAs in lib/ruby-rt.metajs.
func (rt *jsrt) rubyBinAs(name, op string, l, r interface{}) interface{} {
	defer func() {
		if p := recover(); p != nil {
			if th, isThrown := p.(*jsThrown); isThrown {
				if o, isObj := th.value.(*jsObject); isObj {
					if f, framed := o.props["__rframe"].(string); framed && f != "" {
						o.set("__rframe", name)
					}
				}
			}
			panic(p)
		}
	}()
	return rt.rubyBin(op, l, r)
}

// rubyFloDivmod is MRI's flodivmod.
func (rt *jsrt) rubyFloDivmod(x, y float64) interface{} {
	if y != y {
		rubyFloDomain(rt, y)
	}
	if y == 0 {
		rt.rubyRaiseErr("divmod", "ZeroDivisionError", "divided by 0")
	}
	xInf, yInf := math.IsInf(x, 0), math.IsInf(y, 0)
	mod := x
	if !(x == 0 || (yInf && !xInf)) {
		mod = math.Mod(x, y)
	}
	div := x
	if !(xInf && !yInf) {
		if dq := (x - mod) / y; dq < 0 {
			div = -math.Floor(0.5 - dq)
		} else {
			div = math.Floor(dq + 0.5)
		}
	}
	if y*mod < 0 {
		mod += y
		div--
	}
	rubyFloDomain(rt, div)
	return &jsArray{elems: []interface{}{rt.rubyIntFromF(div), jsFlo{f: mod}}}
}

func (rt *jsrt) rubyRationalize(t interface{}, args []interface{}) interface{} {
	switch v := t.(type) {
	case float64, *jsBigInt:
		return rt.rubyAsRat(t)
	case jsRat:
		if len(args) == 0 {
			return v
		}
		if _, ef := args[0].(jsFlo); !ef {
			er := rt.rubyAsRat(args[0])
			if er.n == 0 {
				return v
			}
			vn, vd := rubyRatParts(v)
			en, ed := rubyRatParts(er)
			d := new(big.Int).Mul(vd, ed)
			c := new(big.Int).Mul(new(big.Int).Abs(vn), ed)
			e := new(big.Int).Mul(new(big.Int).Abs(en), vd)
			r, _ := rubyRatSimplest(new(big.Rat).SetFrac(new(big.Int).Sub(c, e), d), new(big.Rat).SetFrac(new(big.Int).Add(c, e), d))
			if v.n < 0 {
				r = rubyRatNeg(r)
			}
			return r
		}
	}
	x := rubyToF(t)
	rubyFloDomain(rt, x)
	ax := math.Abs(x)
	var r jsRat
	ok := true
	switch {
	case len(args) > 0:
		ef := math.Abs(rubyToF(args[0]))
		lo, hi := ax-ef, ax+ef
		if lo == hi {
			return rt.rubyAsRat(t)
		}
		r, ok = rubyFloSimplest(lo, hi)
	case ax == 0:
		return rubyMkRat(0, 1)
	default:
		f, n := ax, 0
		for f < 4503599627370496 {
			f, n = f*2, n-1
		}
		for f >= 9007199254740992 {
			f, n = f/2, n+1
		}
		if n >= 0 {
			return rubyMkRat(x, 1)
		}
		den := new(big.Int).Lsh(big.NewInt(1), uint(1-n))
		twoF := big.NewInt(2 * int64(f))
		r, ok = rubyRatSimplest(new(big.Rat).SetFrac(new(big.Int).Sub(twoF, big.NewInt(1)), den),
			new(big.Rat).SetFrac(new(big.Int).Add(twoF, big.NewInt(1)), den))
	}
	if !ok {
		return rt.rubyAsRat(t)
	}
	if x < 0 {
		r = rubyRatNeg(r)
	}
	return r
}

// ----- Rational parts past 2^53 --------------------------------------------
//
// A Rational's parts are Integers of any size, as in MRI: 10**20 + Rational(1, 3)
// is (300000000000000000001/3). A pair that fits the doubles keeps the plain
// float64 form every other site reads; past 2^53 the exact parts ride along as
// bn / bd, and n / d hold only their nearest doubles. Twins: mkRat / mkRatBig in
// languages/ruby-interpreter.abnf and rbMkRat in languages/lib/ruby-rt.metajs.

var rubyRatLim = new(big.Int).Lsh(big.NewInt(1), 53)

// rubyRatParts is the exact pair, whichever form the Rational has.
func rubyRatParts(r jsRat) (*big.Int, *big.Int) {
	if r.bn != nil {
		return r.bn, r.bd
	}
	return rubyExactInt(r.n), rubyExactInt(r.d)
}

// rubyMkRatB reduces an exact pair, sign on the numerator.
func rubyMkRatB(n0, d0 *big.Int) jsRat {
	n, d := new(big.Int).Set(n0), new(big.Int).Set(d0)
	if d.Sign() < 0 {
		n.Neg(n)
		d.Neg(d)
	}
	if g := new(big.Int).GCD(nil, nil, new(big.Int).Abs(n), d); g.Cmp(big.NewInt(1)) > 0 {
		n.Quo(n, g)
		d.Quo(d, g)
	}
	if n.CmpAbs(rubyRatLim) <= 0 && d.CmpAbs(rubyRatLim) <= 0 {
		return jsRat{n: rubyBigIntToF(n), d: rubyBigIntToF(d)}
	}
	return jsRat{n: rubyBigIntToF(n), d: rubyBigIntToF(d), bn: n, bd: d}
}

func (rt *jsrt) rubyRatArith(op string, p, q jsRat) jsRat {
	pn, pd := rubyRatParts(p)
	qn, qd := rubyRatParts(q)
	mul := func(a, b *big.Int) *big.Int { return new(big.Int).Mul(a, b) }
	switch op {
	case "+":
		return rubyMkRatB(new(big.Int).Add(mul(pn, qd), mul(qn, pd)), mul(pd, qd))
	case "-":
		return rubyMkRatB(new(big.Int).Sub(mul(pn, qd), mul(qn, pd)), mul(pd, qd))
	case "*":
		return rubyMkRatB(mul(pn, qn), mul(pd, qd))
	}
	return rubyMkRatB(mul(pn, qd), mul(pd, qn))
}

func rubyRatNeg(r jsRat) jsRat {
	if r.bn != nil {
		return jsRat{n: -r.n, d: r.d, bn: new(big.Int).Neg(r.bn), bd: r.bd}
	}
	return jsRat{n: -r.n, d: r.d}
}

func rubyRatToF(r jsRat) float64 {
	if r.bn != nil {
		return rubyIntFdiv(&jsBigInt{v: r.bn}, &jsBigInt{v: r.bd})
	}
	return r.n / r.d
}

// rubyRatPart is a numerator or a denominator as a Ruby Integer.
func (rt *jsrt) rubyRatPart(r jsRat, num bool) interface{} {
	if r.bn == nil {
		if num {
			return r.n
		}
		return r.d
	}
	if num {
		return rt.rubyBigNarrow(r.bn)
	}
	return rt.rubyBigNarrow(r.bd)
}

// rubyRatExact compares a Rational with an Integer or a Rational EXACTLY, by cross
// multiplication; ok is false for any other pair.
func (rt *jsrt) rubyRatExact(l, r interface{}) (int, bool) {
	_, lr := l.(jsRat)
	_, rr := r.(jsRat)
	if !lr && !rr {
		return 0, false
	}
	for _, v := range []interface{}{l, r} {
		switch v.(type) {
		case jsRat, float64, *jsBigInt:
		default:
			return 0, false
		}
	}
	pn, pd := rubyRatParts(rt.rubyAsRat(l))
	qn, qd := rubyRatParts(rt.rubyAsRat(r))
	return new(big.Int).Mul(pn, qd).Cmp(new(big.Int).Mul(qn, pd)), true
}

// rubyIntExponent is an Integer exponent, a Rational one over 1 included.
func (rt *jsrt) rubyIntExponent(r interface{}) (*big.Int, bool) {
	if q, ok := r.(jsRat); ok {
		n, d := rubyRatParts(q)
		return n, d.Cmp(big.NewInt(1)) == 0
	}
	switch r.(type) {
	case float64, *jsBigInt:
		return rubyBigOf(r)
	}
	return nil, false
}

// rubyRatPow is a Rational to an Integer power, exactly. Twin of ratPow in
// languages/ruby-interpreter.abnf.
func (rt *jsrt) rubyRatPow(p jsRat, k *big.Int) interface{} {
	n, d := rubyRatParts(p)
	if k.Sign() < 0 {
		if n.Sign() == 0 {
			rt.rubyRaiseErr("**", "ZeroDivisionError", "divided by 0")
		}
		n, d = d, n
		k = new(big.Int).Neg(k)
	}
	return rubyMkRatB(new(big.Int).Exp(n, k, nil), new(big.Int).Exp(d, k, nil))
}

// rubyRatIntOpB is rubyRatIntOp on exact parts (d > 0, so Div is the floor).
func rubyRatIntOpB(n, d *big.Int, mode byte) *big.Int {
	switch mode {
	case 'f':
		return new(big.Int).Div(n, d)
	case 'c':
		return new(big.Int).Neg(new(big.Int).Div(new(big.Int).Neg(n), d))
	case 't':
		if n.Sign() < 0 {
			return rubyRatIntOpB(n, d, 'c')
		}
		return new(big.Int).Div(n, d)
	}
	n2 := new(big.Int).Lsh(n, 1)
	d2 := new(big.Int).Lsh(d, 1)
	var k *big.Int
	if n.Sign() >= 0 {
		k = new(big.Int).Div(new(big.Int).Add(n2, d), d2)
	} else {
		k = new(big.Int).Neg(new(big.Int).Div(new(big.Int).Sub(d, n2), d2))
	}
	// half: :even / :down move an exact tie (2n/d odd) back toward zero.
	q, r := new(big.Int).QuoRem(n2, d, new(big.Int))
	if mode == 'r' || r.Sign() != 0 || q.Bit(0) == 0 {
		return k
	}
	if mode == 'd' || k.Bit(0) != 0 {
		return k.Sub(k, big.NewInt(int64(n.Sign())))
	}
	return k
}

// rubyProcCompose is Proc#>> (g after f) and Proc#<< (f after g); g is anything
// with #call, and the composition is a lambda when f is. Twins: procCompose in
// ruby-interpreter.abnf, rbProcCompose in lib/ruby-rt.metajs.
func (rt *jsrt) rubyProcCompose(f, g interface{}, fFirst bool) interface{} {
	callG := func(a []interface{}) interface{} {
		if isCallable(g) {
			return rt.call(g, jsUndef, a)
		}
		return rt.rubyMethod(g, "call", a)
	}
	var comp interface{} = &hostFunc{name: "compose", fn: func(rt *jsrt, this uint64, a []interface{}) interface{} {
		if fFirst {
			return callG([]interface{}{rt.call(f, jsUndef, a)})
		}
		return rt.call(f, jsUndef, []interface{}{callG(a)})
	}}
	rt.rubyMarkLambdaLike(comp, f)
	return comp
}

// rubyProcCurry is Proc#curry: collect arguments until there are n, then call f
// with them all. Twins: procCurry in ruby-interpreter.abnf, rbProcCurry in
// lib/ruby-rt.metajs.
func (rt *jsrt) rubyProcCurry(f interface{}, n int, got []interface{}) interface{} {
	var cf interface{} = &hostFunc{name: "curry", fn: func(rt *jsrt, this uint64, a []interface{}) interface{} {
		all := append(append([]interface{}{}, got...), a...)
		if len(all) >= n {
			return rt.call(f, jsUndef, all)
		}
		return rt.rubyProcCurry(f, n, all)
	}}
	rt.rubyMarkLambdaLike(cf, f)
	return cf
}

// rubyMarkLambdaLike makes g a lambda when f is one.
func (rt *jsrt) rubyMarkLambdaLike(g, f interface{}) {
	if rt.rubyLambdas[f] {
		if rt.rubyLambdas == nil {
			rt.rubyLambdas = map[interface{}]bool{}
		}
		rt.rubyLambdas[g] = true
	}
}

// rubyStopIter is Enumerator#next past the end: StopIteration, whose #result is
// what the iteration answered, every yield having given nil - the receiver of an
// each, and otherwise the method run again with a block answering nil. Twin of
// enumResult in ruby-interpreter.abnf, rbStopIter in lib/ruby-rt.metajs.
func (rt *jsrt) rubyStopIter(name string, t *jsObject) {
	e := rt.rubyExc(rt.rubyBuiltinClass("StopIteration"), []interface{}{"iteration reached an end"})
	var res interface{} = jsNull
	if src, has := t.props["src"]; has {
		res = src
		if t.props["op"] != "each" {
			meth, _ := t.props["meth"].(string)
			var args []interface{}
			if ma, isArr := t.props["margs"].(*jsArray); isArr {
				args = append(args, ma.elems...)
			}
			nilBlk := &hostFunc{name: "", fn: func(*jsrt, uint64, []interface{}) interface{} { return jsNull }}
			res = rt.rubyMethod(src, meth, append(args, rubyBlkWrap(nilBlk)))
		}
	}
	e.set("__rframe", name)
	e.set("__rresult", res)
	panic(&jsThrown{value: e})
}

// rubyObjCmp is Object#<=> on a user object whose class defines no <=>: is it
// 0 (else nil)? Twin of objCmp in ruby-interpreter.abnf, which states the rule,
// and rbObjCmp in lib/ruby-rt.metajs.
// rubyForEnumItems: what `for` over an Enumerator binds, one entry per yield.
// Twin of forEnumItems in ruby-interpreter.abnf, which states the rule, and
// rbForEnumItems in languages/lib/ruby-rt.metajs.
func (rt *jsrt) rubyForEnumItems(e *jsObject, whole bool) interface{} {
	meth, _ := e.props["meth"].(string)
	if whole && e.props["__rdefer"] == true && rubyForPassMeth(meth) {
		all := &jsArray{}
		rt.rubyLazyDrive(e.props["src"], func(v interface{}, multi bool) bool {
			if multi {
				v = v.(*jsArray).elems[0]
			}
			all.elems = append(all.elems, v)
			return false
		})
		return all
	}
	all := rt.rubyMethod(e, "to_a", nil)
	arr, isArr := all.(*jsArray)
	if !whole || !isArr || !rubyForPairMeth(meth) {
		return all
	}
	firsts := &jsArray{}
	for _, x := range arr.elems {
		if xa, ok := x.(*jsArray); ok && len(xa.elems) > 0 {
			x = xa.elems[0]
		}
		firsts.elems = append(firsts.elems, x)
	}
	return firsts
}

// Twins of rbaForPairMeth / rbaForPassMeth in languages/lib/ruby-arr.js.
func rubyForPairMeth(m string) bool {
	return m == "each_with_index" || m == "each_with_object" || m == "with_index" || m == "with_object"
}

func rubyForPassMeth(m string) bool {
	return m == "each" || m == "map" || m == "collect" || m == "flat_map" || m == "collect_concat"
}

// rubyForEnumVal: what `for` over an Enumerator answers. Twin of forEnumVal in
// ruby-interpreter.abnf, which states the rule, and rbForEnumVal in
// languages/lib/ruby-rt.metajs.
func rubyForEnumVal(e *jsObject) interface{} {
	m, isStr := e.props["meth"].(string)
	if !isStr {
		return e
	}
	if m == "each_slice" || m == "each_cons" {
		return jsNull
	}
	if !strings.HasPrefix(m, "each") || (m == "each" && e.props["__rdefer"] == true) {
		return e
	}
	return e.props["src"]
}

func (rt *jsrt) rubyObjCmp(l, r interface{}) bool {
	if rubyIdenticalP(l, r) {
		return true
	}
	o := l.(*jsObject)
	m, ok := rubyFindMethod(l, "==")
	if !ok || o.props["__rcmpbusy"] == true {
		return false
	}
	o.set("__rcmpbusy", true)
	defer o.set("__rcmpbusy", false)
	return rubyTruthy(rt.call(m, jsUndef, []interface{}{l, r}))
}

// rubyCatch / rubyThrow are Kernel#catch / #throw. Twins: the catch / throw host
// globals of ruby-interpreter.abnf, which state the rule, and js_rcatch /
// js_rthrow in lib/ruby-rt.metajs. A throw to a running catch is a js_pyexc
// "__rthrow" marker carrying [tag, value]; no rescue clause takes one.
func (rt *jsrt) rubyCatch(tag, blk interface{}) (res interface{}) {
	if tag == jsUndef {
		o := newJSObject()
		o.set("__class", rt.rubyBuiltinClass("Object"))
		tag = o
	}
	rt.rubyCatchTags = append(rt.rubyCatchTags, tag)
	depth := len(rt.rubyCatchTags)
	defer func() {
		rt.rubyCatchTags = rt.rubyCatchTags[:depth-1]
		if p := recover(); p != nil {
			if th, isThrown := p.(*jsThrown); isThrown && rt.isTypeName(th.value, "__rthrow") {
				if args, ok := th.value.(*jsObject).props["args"].(*jsArray); ok && len(args.elems) == 2 && rubyIdenticalP(args.elems[0], tag) {
					res = args.elems[1]
					return
				}
			}
			panic(p)
		}
	}()
	return rt.call(blk, jsUndef, []interface{}{tag})
}

func (rt *jsrt) rubyThrow(tag, v interface{}) {
	for i := len(rt.rubyCatchTags) - 1; i >= 0; i-- {
		if rubyIdenticalP(rt.rubyCatchTags[i], tag) {
			cls := newJSObject()
			cls.set("__name", "__rthrow")
			mark := newJSObject()
			mark.set("__class", cls)
			mark.set("args", &jsArray{elems: []interface{}{tag, v}})
			panic(&jsThrown{value: mark})
		}
	}
	e := rt.rubyExc(rt.rubyBuiltinClass("UncaughtThrowError"), []interface{}{"uncaught throw " + rt.rubyInspect(tag)})
	e.set("__rtag", tag)
	e.set("__rval", v)
	e.set("__rframe", "throw")
	panic(&jsThrown{value: e})
}

// rubyPutsLine is a line puts writes: the text, and a newline unless it already
// ends in one. Twin of putsLine in ruby-interpreter.abnf.
func rubyPutsLine(s string) string {
	if strings.HasSuffix(s, "\n") {
		return s
	}
	return s + "\n"
}

// rubyIOMethod answers the methods of $stdout / STDOUT (fd 1) and $stderr /
// STDERR (fd 2). Twin of ioMethod in ruby-interpreter.abnf, which states the
// rules; standard error is unbuffered there, and here too.
func (rt *jsrt) rubyIOMethod(o *jsObject, fd int, name string, args []interface{}) interface{} {
	pos, _ := rubyBlockArgs(args)
	// raw: s is a binary String's bytes, written to standard output as they are.
	write := func(s string, raw bool) {
		if fd != 2 {
			if raw {
				outWriter.Write([]byte(s))
			} else {
				fmt.Fprint(outWriter, wtf8Clean(s))
			}
			return
		}
		fmt.Fprint(warnWriter, wtf8Clean(s))
	}
	var puts func(v interface{})
	puts = func(v interface{}) {
		if arr, isArr := v.(*jsArray); isArr {
			for _, e := range arr.elems {
				puts(e)
			}
			return
		}
		if b, isBin := rubyFwd(v).(*jsRBin); isBin {
			write(rubyPutsLine(b.b), true)
			return
		}
		write(rubyPutsLine(rt.rubyStr(v)), false)
	}
	switch name {
	case "puts":
		if len(pos) == 0 {
			write("\n", false)
		}
		for _, v := range pos {
			puts(v)
		}
		return jsNull
	case "print", "write", "<<":
		n := 0
		for _, v := range pos {
			if b, isBin := rubyFwd(v).(*jsRBin); isBin {
				write(b.b, true)
				n += len(b.b)
				continue
			}
			s := rt.rubyStr(v)
			write(s, false)
			n += len(s)
		}
		if name == "write" {
			return float64(n)
		}
		if name == "<<" {
			return o
		}
		return jsNull
	case "printf":
		if len(pos) > 0 {
			write(rt.rubyFormatArgs(pos, "printf"), false)
		}
		return jsNull
	case "flush":
		return o
	case "sync", "sync=":
		return true
	case "fileno":
		return float64(fd)
	case "tty?", "isatty":
		return false
	}
	rt.rubyNoMethod(o, name)
	return nil
}

// rubyPolarPi is MRI's rb_dbl_complex_new_polar_pi: abs * e^(i*pi*ang), which a
// negative Float to a non-integral power is. Twin of rbnPolarPi in
// lib/ruby-num.js.
func rubyPolarPi(abs, ang float64) (float64, float64) {
	fi := math.Trunc(ang)
	fr := ang - fi
	pos := fr == 0.5
	if pos || fr == -0.5 {
		h := fi / 2
		if (h-math.Trunc(h) != fr) != pos {
			abs = -abs
		}
		return 0, abs
	}
	return abs * math.Cos(math.Pi*ang), abs * math.Sin(math.Pi*ang)
}

// rubyLoop is Kernel#loop: the block again and again, until a break (the
// caller's marker catch) or a StopIteration, whose #result is the answer. Twin
// of js_rloop in lib/ruby-rt.metajs and loop in ruby-interpreter.abnf.
func (rt *jsrt) rubyLoop(f interface{}) (res interface{}) {
	stop := rt.rubyBuiltinClass("StopIteration")
	defer func() {
		if p := recover(); p != nil {
			if th, isThrown := p.(*jsThrown); isThrown && rt.rubyIsA(th.value, stop) {
				res = jsNull
				if o, isObj := th.value.(*jsObject); isObj {
					if r, had := o.props["__rresult"]; had {
						res = r
					}
				}
				return
			}
			panic(p)
		}
	}()
	for {
		rt.call(f, jsUndef, nil)
	}
}

// rubyMainObj is MRI's top-level `self`: an Object whose singleton #to_s and
// #inspect answer "main". Twin of mainObj in ruby-interpreter.abnf.
func (rt *jsrt) rubyMainObj() *jsObject {
	o := newJSObject()
	o.set("__class", rt.rubyBuiltinClass("Object"))
	name := &hostFunc{name: "main", fn: func(rt *jsrt, this uint64, a []interface{}) interface{} { return "main" }}
	rubySingPut(o, "to_s", name)
	rubySingPut(o, "inspect", name)
	return o
}

// rubyRespondNames is Object#respond_to? for a value that is not a user object:
// per builtin class, the names that dispatch in all three ruby engines, row "*"
// holding the ones every class answers. Twin of rbRespondRows in
// languages/lib/ruby-respond.js (whose comment says how the list is made);
// ruby_respond_test.go keeps the two equal.
var rubyRespondNames = map[string][]string{
	"*": {"!", "!=", "==", "===", "class", "hash", "inspect", "itself", "object_id",
		"respond_to?", "tap", "to_s"},
	"private": {"format", "lambda", "loop", "p", "print", "printf", "proc", "puts", "raise",
		"rand", "require", "sprintf"},
	"Integer": {"%", "&", "*", "**", "+", "+@", "-", "-@", "/", "<", "<=", "<=>", ">", ">=",
		"[]", "^", "abs", "abs2", "allbits?", "angle", "anybits?", "arg", "bit_length", "ceil",
		"chr", "clone", "coerce", "conj", "conjugate", "denominator", "digits", "div", "divmod",
		"dup", "eql?", "equal?", "even?", "fdiv", "finite?", "floor", "frozen?", "gcd", "gcdlcm",
		"i", "imag", "imaginary", "infinite?", "instance_of?", "instance_variables", "integer?",
		"is_a?", "kind_of?", "lcm", "magnitude", "modulo", "negative?", "next", "nil?", "nobits?",
		"nonzero?", "numerator", "odd?", "ord", "phase", "polar", "positive?", "pow", "pred",
		"public_send", "quo", "rationalize", "real", "real?", "rect", "rectangular", "remainder",
		"round", "send", "size", "succ", "to_c", "to_f", "to_i", "to_int", "to_r", "truncate",
		"zero?", "|"},
	"Float": {"%", "*", "**", "+", "+@", "-", "-@", "/", "<", "<=", "<=>", ">", ">=", "abs",
		"abs2", "angle", "arg", "ceil", "clone", "coerce", "conj", "conjugate", "denominator",
		"div", "divmod", "dup", "eql?", "equal?", "fdiv", "finite?", "floor", "frozen?", "i",
		"imag", "imaginary", "infinite?", "instance_of?", "instance_variables", "integer?",
		"is_a?", "kind_of?", "magnitude", "modulo", "nan?", "negative?", "next_float", "nil?",
		"nonzero?", "numerator", "phase", "polar", "positive?", "prev_float", "public_send",
		"quo", "rationalize", "real", "real?", "rect", "rectangular", "remainder", "round",
		"send", "to_c", "to_f", "to_i", "to_int", "to_r", "truncate", "zero?"},
	"Rational": {"%", "*", "**", "+", "+@", "-", "-@", "/", "<", "<=", "<=>", ">", ">=", "abs",
		"abs2", "angle", "arg", "ceil", "clone", "coerce", "conj", "conjugate", "denominator",
		"div", "divmod", "dup", "eql?", "equal?", "fdiv", "finite?", "floor", "frozen?", "i",
		"imag", "imaginary", "infinite?", "instance_of?", "instance_variables", "integer?",
		"is_a?", "kind_of?", "magnitude", "negative?", "nil?", "nonzero?", "numerator", "phase",
		"polar", "positive?", "public_send", "quo", "rationalize", "real", "real?", "rect",
		"rectangular", "remainder", "round", "send", "to_c", "to_f", "to_i", "to_int", "to_r",
		"truncate", "zero?"},
	"Complex": {"*", "+", "+@", "-", "-@", "/", "abs", "abs2", "angle", "arg", "clone",
		"coerce", "conj", "conjugate", "denominator", "dup", "eql?", "equal?", "fdiv", "finite?",
		"frozen?", "imag", "imaginary", "infinite?", "instance_of?", "instance_variables",
		"integer?", "is_a?", "kind_of?", "magnitude", "nil?", "nonzero?", "numerator", "phase",
		"polar", "public_send", "quo", "rationalize", "real", "real?", "rect", "rectangular",
		"send", "to_c", "to_f", "to_i", "to_int", "to_r", "zero?"},
	"String": {"%", "*", "+", "+@", "<", "<<", "<=", "<=>", ">", ">=", "[]", "ascii_only?",
		"b", "bytes", "bytesize", "capitalize", "casecmp", "casecmp?", "center", "chars", "chomp",
		"chomp!", "chop", "chop!", "chr", "clear", "clone", "codepoints", "concat", "count",
		"delete", "delete!", "delete_prefix", "delete_prefix!", "delete_suffix", "delete_suffix!",
		"downcase", "dump", "dup", "each_byte", "each_char", "each_codepoint", "each_line",
		"empty?", "encode", "end_with?", "equal?", "force_encoding", "freeze", "frozen?",
		"getbyte", "hex", "include?", "index", "insert", "instance_of?", "instance_variables",
		"intern", "is_a?", "kind_of?", "length", "lines", "ljust", "lstrip", "lstrip!", "next",
		"nil?", "oct", "ord", "partition", "prepend", "public_send", "replace", "reverse",
		"reverse!", "rindex", "rjust", "rpartition", "rstrip", "rstrip!", "scrub", "send", "size",
		"slice", "slice!", "split", "squeeze", "squeeze!", "start_with?", "strip", "strip!",
		"succ", "sum", "swapcase", "swapcase!", "to_f", "to_i", "to_str", "to_sym", "tr", "tr!",
		"tr_s", "tr_s!", "unicode_normalize", "unicode_normalized?", "upcase", "upto",
		"valid_encoding?"},
	"Symbol": {"<", "<=", "<=>", ">", ">=", "[]", "capitalize", "downcase", "empty?", "equal?",
		"frozen?", "id2name", "instance_of?", "instance_variables", "is_a?", "kind_of?", "length",
		"name", "next", "nil?", "public_send", "send", "size", "succ", "swapcase", "to_proc",
		"to_sym", "upcase"},
	"Array": {"&", "*", "+", "-", "<<", "<=>", "[]", "all?", "any?", "append", "assoc",
		"chain", "chunk", "clear", "clone", "collect", "collect!", "collect_concat", "combination", "compact",
		"compact!", "concat", "count", "cycle", "delete", "delete_at", "delete_if", "detect",
		"drop", "drop_while", "dup", "each", "each_cons", "each_index", "each_slice",
		"each_with_index", "each_with_object", "empty?", "entries", "equal?", "fetch", "fill",
		"filter", "filter!", "find", "find_index", "first", "flat_map", "flatten", "flatten!",
		"freeze", "frozen?", "grep", "grep_v", "group_by", "include?", "index", "inject", "insert", "instance_of?",
		"instance_variables", "is_a?", "itself", "join", "keep_if", "kind_of?", "last", "lazy", "length",
		"map", "map!", "max", "max_by", "member?", "min", "min_by", "minmax", "minmax_by", "nil?", "none?", "one?",
		"partition", "permutation", "pop", "prepend", "product", "public_send", "push", "reduce",
		"reject", "reject!", "reverse", "reverse!", "reverse_each", "rindex", "rotate", "sample",
		"select", "select!", "send", "shift", "shuffle", "shuffle!", "size", "slice", "slice_after", "slice_before", "sort",
		"sort!", "sort_by", "sort_by!", "sum", "take", "take_while", "tally", "to_a", "to_h", "uniq",
		"uniq!", "unshift", "values_at", "zip", "|"},
	"Hash": {"<", "<=", "<=>", ">", ">=", "[]", "all?", "any?", "assoc", "chain", "chunk", "clear", "clone",
		"collect", "collect_concat", "count", "default", "delete", "delete_if", "detect", "drop",
		"drop_while", "dup", "each", "each_cons", "each_key", "each_pair", "each_slice",
		"each_value", "each_with_index", "each_with_object", "empty?", "entries", "equal?",
		"fetch", "filter", "filter!", "find", "find_index", "first", "flat_map", "freeze",
		"frozen?", "grep", "grep_v", "group_by", "has_key?", "include?", "inject", "instance_of?",
		"instance_variables", "invert", "is_a?", "itself", "keep_if", "key", "key?", "keys",
		"kind_of?", "length", "map", "max", "max_by", "member?", "merge", "merge!", "min", "min_by",
		"minmax", "minmax_by", "nil?", "none?", "one?", "partition", "public_send", "reduce", "reject",
		"reject!", "select", "select!", "send", "size", "slice_after", "slice_before", "sort", "sort_by", "sum", "take",
		"take_while", "tally", "to_a", "to_h", "transform_keys", "transform_values", "uniq",
		"update", "values", "values_at", "zip"},
	"Range": {"<=>", "all?", "any?", "begin", "chain", "chunk", "clone", "collect", "collect_concat", "count",
		"cover?", "cycle", "detect", "drop", "drop_while", "dup", "each", "each_cons",
		"each_slice", "each_with_index", "each_with_object", "end", "entries", "eql?", "equal?",
		"exclude_end?", "filter", "find", "find_index", "first", "flat_map", "freeze", "frozen?",
		"grep", "grep_v", "group_by", "hash", "include?", "inject", "instance_of?", "instance_variables", "is_a?",
		"itself", "kind_of?", "last", "lazy", "map", "max", "max_by", "member?", "min", "min_by",
		"minmax", "minmax_by", "nil?", "none?", "object_id", "one?", "partition", "public_send", "reduce",
		"reject", "reverse_each", "select", "send", "size", "slice_after", "slice_before", "sort", "sort_by", "step", "sum",
		"take", "take_while", "tally", "to_a", "uniq", "zip"},
	"NilClass": {"&", "<=>", "^", "equal?", "frozen?", "instance_of?", "instance_variables",
		"is_a?", "kind_of?", "nil?", "public_send", "send", "to_a", "to_f", "to_i", "|"},
	"TrueClass": {"&", "<=>", "^", "equal?", "frozen?", "instance_of?", "instance_variables",
		"is_a?", "kind_of?", "nil?", "public_send", "send", "|"},
	"FalseClass": {"&", "<=>", "^", "equal?", "frozen?", "instance_of?", "instance_variables",
		"is_a?", "kind_of?", "nil?", "public_send", "send", "|"},
	"Proc": {"<<", "<=>", ">>", "[]", "arity", "call", "curry", "equal?", "frozen?", "instance_of?",
		"instance_variables", "is_a?", "kind_of?", "lambda?", "nil?", "public_send", "send",
		"to_proc", "yield"},
	"Method": {"<<", "<=>", ">>", "[]", "arity", "call", "curry", "equal?", "frozen?", "instance_of?",
		"instance_variables", "is_a?", "kind_of?", "name", "nil?", "public_send", "send",
		"to_proc"},
	"Regexp": {"<=>", "=~", "equal?", "frozen?", "instance_of?", "instance_variables", "is_a?",
		"kind_of?", "match", "match?", "names", "nil?", "options", "public_send", "send",
		"source"},
	"MatchData": {"<=>", "[]", "begin", "captures", "end", "equal?", "frozen?", "instance_of?",
		"instance_variables", "is_a?", "kind_of?", "length", "named_captures", "names", "nil?",
		"post_match", "pre_match", "public_send", "send", "size", "to_a"},
	"Enumerator": {"<=>", "all?", "any?", "chain", "chunk", "clone", "collect", "collect_concat", "count",
		"cycle", "detect", "drop", "drop_while", "dup", "each", "each_cons", "each_slice",
		"each_with_index", "each_with_object", "entries", "equal?", "filter", "find",
		"find_index", "first", "flat_map", "freeze", "frozen?", "grep", "grep_v", "group_by", "include?", "inject",
		"instance_of?", "instance_variables", "is_a?", "itself", "kind_of?", "lazy", "map", "max",
		"max_by", "member?", "min", "min_by", "minmax", "minmax_by", "next", "nil?", "none?", "one?", "partition", "peek",
		"public_send", "reduce", "reject", "reverse_each", "rewind", "select", "send", "size",
		"slice_after", "slice_before", "sort", "sort_by", "sum", "take", "take_while", "tally", "to_a", "uniq", "with_index",
		"with_object", "zip"},
	"Enumerable": {"all?", "any?", "chain", "chunk", "chunk_while", "collect", "collect_concat", "count",
		"cycle", "detect", "drop", "drop_while", "each_cons", "each_entry", "each_slice",
		"each_with_index", "each_with_object", "entries", "filter", "find", "find_all",
		"find_index", "first", "flat_map", "grep", "grep_v", "group_by", "include?", "inject", "lazy", "map",
		"max", "max_by", "member?", "min", "min_by", "minmax", "minmax_by", "none?", "one?", "partition", "reduce",
		"reject", "reverse_each", "select", "slice_after", "slice_before", "slice_when", "sort", "sort_by", "sum", "take",
		"take_while", "to_a", "to_h", "uniq", "zip"},
	"Class": {"<", "<=", ">", ">=", "ancestors", "class_eval", "constants", "equal?",
		"frozen?", "include?", "instance_methods", "instance_of?",
		"instance_variable_get", "instance_variable_set", "instance_variables", "is_a?",
		"kind_of?", "method_defined?", "module_eval", "name", "new", "nil?",
		"private_instance_methods", "private_method_defined?", "public_instance_methods",
		"public_method_defined?", "public_send", "send", "singleton_methods",
		"superclass"},
	"Module": {"<", "<=", ">", ">=", "ancestors", "class_eval", "constants", "equal?",
		"frozen?", "include?", "instance_methods", "instance_of?",
		"instance_variable_get", "instance_variable_set", "instance_variables", "is_a?",
		"kind_of?", "method_defined?", "module_eval", "name", "nil?",
		"private_instance_methods", "private_method_defined?", "public_instance_methods",
		"public_method_defined?", "public_send", "send", "singleton_methods"},
}

// rubyOwnNames is Module#instance_methods(false) of a builtin class: the names of
// its rubyRespondNames row (or of row "*") that MRI defines on the class itself.
// Twin of rbOwnRows in languages/lib/ruby-respond.js; ruby_respond_test.go keeps
// the two equal.
var rubyOwnNames = map[string][]string{
	"Integer": {"%", "&", "*", "**", "+", "-", "-@", "/", "<", "<=", "<=>", "==", "===", ">",
		">=", "[]", "^", "abs", "allbits?", "anybits?", "bit_length", "ceil", "chr", "coerce",
		"denominator", "digits", "div", "divmod", "even?", "fdiv", "floor", "gcd", "gcdlcm",
		"inspect", "integer?", "lcm", "magnitude", "modulo", "next", "nobits?", "numerator",
		"odd?", "ord", "pow", "pred", "rationalize", "remainder", "round", "size", "succ",
		"to_f", "to_i", "to_int", "to_r", "to_s", "truncate", "|"},
	"Float": {"%", "*", "**", "+", "-", "-@", "/", "<", "<=", "<=>", "==", "===", ">", ">=",
		"abs", "angle", "arg", "ceil", "coerce", "denominator", "divmod", "eql?", "fdiv",
		"finite?", "floor", "hash", "infinite?", "inspect", "magnitude", "modulo", "nan?",
		"negative?", "next_float", "numerator", "phase", "positive?", "prev_float", "quo",
		"rationalize", "round", "to_f", "to_i", "to_int", "to_r", "to_s", "truncate", "zero?"},
	"Rational": {"*", "**", "+", "-", "-@", "/", "<=>", "==", "abs", "ceil", "coerce",
		"denominator", "fdiv", "floor", "hash", "inspect", "magnitude", "negative?", "numerator",
		"positive?", "quo", "rationalize", "round", "to_f", "to_i", "to_r", "to_s", "truncate"},
	"Complex": {"*", "+", "-", "-@", "/", "==", "abs", "abs2", "angle", "arg", "coerce",
		"conj", "conjugate", "denominator", "eql?", "fdiv", "finite?", "hash", "imag",
		"imaginary", "infinite?", "inspect", "magnitude", "numerator", "phase", "polar", "quo",
		"rationalize", "real", "real?", "rect", "rectangular", "to_c", "to_f", "to_i", "to_r",
		"to_s"},
	"String": {"%", "*", "+", "+@", "<<", "<=>", "==", "===", "[]", "ascii_only?", "b",
		"bytes", "bytesize", "capitalize", "casecmp", "casecmp?", "center", "chars", "chomp",
		"chomp!", "chop", "chop!", "chr", "clear", "codepoints", "concat", "count", "delete",
		"delete!", "delete_prefix", "delete_prefix!", "delete_suffix", "delete_suffix!",
		"downcase", "dump", "each_byte", "each_char", "each_codepoint", "each_line", "empty?",
		"encode", "end_with?", "force_encoding", "freeze", "getbyte", "hash", "hex", "include?",
		"index", "insert", "inspect", "intern", "length", "lines", "ljust", "lstrip", "lstrip!",
		"next", "oct", "ord", "partition", "prepend", "replace", "reverse", "reverse!", "rindex",
		"rjust", "rpartition", "rstrip", "rstrip!", "scrub", "size", "slice", "slice!", "split",
		"squeeze", "squeeze!", "start_with?", "strip", "strip!", "succ", "sum", "swapcase",
		"swapcase!", "to_f", "to_i", "to_s", "to_str", "to_sym", "tr", "tr!", "tr_s", "tr_s!",
		"unicode_normalize", "unicode_normalized?", "upcase", "upto", "valid_encoding?"},
	"Symbol": {"<=>", "==", "===", "[]", "capitalize", "downcase", "empty?", "id2name",
		"inspect", "length", "next", "size", "succ", "swapcase", "to_proc", "to_s", "to_sym",
		"upcase"},
	"Array": {"&", "*", "+", "-", "<<", "<=>", "==", "[]", "all?", "any?", "append", "assoc",
		"clear", "collect", "collect!", "combination", "compact", "compact!", "concat", "count",
		"cycle", "delete", "delete_at", "delete_if", "drop", "drop_while", "each", "each_index",
		"empty?", "fetch", "fill", "filter", "filter!", "find_index", "first", "flatten",
		"flatten!", "hash", "include?", "index", "insert", "inspect", "join", "keep_if", "last",
		"length", "map", "map!", "max", "min", "none?", "one?", "permutation", "pop", "prepend",
		"product", "push", "reject", "reject!", "reverse", "reverse!", "reverse_each", "rindex",
		"rotate", "sample", "select", "select!", "shift", "shuffle", "shuffle!", "size", "slice",
		"sort", "sort!", "sort_by!", "sum", "take", "take_while", "to_a", "to_h", "to_s", "uniq",
		"uniq!", "unshift", "values_at", "zip", "|"},
	"Hash": {"<", "<=", "==", ">", ">=", "[]", "any?", "assoc", "clear", "default", "delete",
		"delete_if", "each", "each_key", "each_pair", "each_value", "empty?", "fetch", "filter",
		"filter!", "has_key?", "hash", "include?", "inspect", "invert", "keep_if", "key", "key?",
		"keys", "length", "merge", "merge!", "reject", "reject!", "select", "select!", "size",
		"to_a", "to_h", "to_s", "transform_keys", "transform_values", "update", "values",
		"values_at"},
	"Range": {"==", "===", "begin", "cover?", "each", "end", "entries", "eql?",
		"exclude_end?", "first", "hash", "include?", "inspect", "last", "max", "member?", "min",
		"size", "step", "to_a", "to_s"},
	"NilClass":   {"&", "===", "^", "inspect", "nil?", "to_a", "to_f", "to_i", "to_s", "|"},
	"TrueClass":  {"&", "===", "^", "inspect", "to_s", "|"},
	"FalseClass": {"&", "===", "^", "inspect", "to_s", "|"},
	"Proc": {"<<", "===", ">>", "[]", "arity", "call", "curry", "hash", "inspect", "lambda?",
		"to_proc", "to_s", "yield"},
	"Method": {"<<", "==", "===", ">>", "[]", "arity", "call", "curry", "hash", "inspect",
		"name", "to_proc", "to_s"},
	"Regexp": {"==", "===", "=~", "hash", "inspect", "match", "match?", "names", "options",
		"source", "to_s"},
	"MatchData": {"==", "[]", "begin", "captures", "end", "hash", "inspect", "length",
		"named_captures", "names", "post_match", "pre_match", "size", "to_a", "to_s"},
	"Enumerator": {"each", "each_with_index", "each_with_object", "inspect", "next", "peek",
		"rewind", "size", "with_index", "with_object"},
}

var rubyRespondSet = func() map[string]bool {
	out := map[string]bool{}
	for cls, names := range rubyRespondNames {
		for _, n := range names {
			out[cls+"#"+n] = true
		}
	}
	return out
}()

// rubyResponds answers respond_to? for a value of builtin class cls.
func rubyResponds(cls, name string) bool {
	return rubyRespondSet["*#"+name] || rubyRespondSet[cls+"#"+name]
}

// rubyRespond is `x.respond_to?(n, include_all)` (js_rrespond): a builtin answers
// from the table, a user instance or class from its own tables. A private or
// protected method counts only with include_all, and so do the private methods
// every object has: the "private" row and the top-level defs, which are private
// methods of Object recorded as "$p$" slots on the Object class object. Twin of
// js_rrespond in lib/ruby-rt.metajs and respondTo in ruby-interpreter.abnf.
func (rt *jsrt) rubyRespond(target interface{}, args []interface{}) interface{} {
	pos, _ := rubyBlockArgs(args)
	n := rt.rubyStr(argAt(pos, 0))
	all := rubyTruthy(argAt(pos, 1))
	if cls := rt.rubyRespondClass(target); cls != "" {
		if rt.rubyBuiltinExt {
			if _, found := rt.rubyBuiltinExtFind(target, n); found {
				return true
			}
		}
		return rubyResponds(cls, n) || (all && rt.rubyRespondsPrivate(n))
	}
	o, isObj := target.(*jsObject)
	if !isObj {
		return rt.rubyMethod(target, "respond_to?", args)
	}
	// A class or module object: a private class method (Class#new included), then
	// the Class / Module row.
	if o.props["__isclass"] == true {
		if rubySendKey(o, n) != n {
			return all
		}
		if rubyResponds(rubyClassRow(o), n) {
			return true
		}
	}
	if rubyResponds("", n) {
		return true
	}
	if _, ok := rubySingFind(o, n); ok {
		return true
	}
	if rubyRestricted(o.props["__class"], n) {
		return all
	}
	if rubySingRespond(o, n) || (all && rt.rubyRespondsPrivate(n)) {
		return true
	}
	if rt.rubyEnumAnswers(o, n) {
		return true
	}
	// respond_to_missing? - see respondTo in ruby-interpreter.abnf.
	if rubyUserObj(o) {
		if _, has := rubyFindMethod(o, "respond_to_missing?"); has {
			return rubyTruthy(rt.rubyMethod(o, "respond_to_missing?", []interface{}{jsSym{s: n}, all}))
		}
	}
	return false
}

// rubyClassRow is the respond_to? row of a class or module object.
func rubyClassRow(o *jsObject) string {
	if o.props["__ismodule"] == true {
		return "Module"
	}
	return "Class"
}

// rubyRespondsPrivate: is n a private method every object has?
// rubyEnumAnswers: an Enumerable method on an instance of a class that includes
// Enumerable. An includer without #each still answers them, and the call fails
// on each, as MRI's does. Twins: enumAnswers in ruby-interpreter.abnf,
// rbEnumAnswers in lib/ruby-rt.metajs.
func (rt *jsrt) rubyEnumAnswers(t interface{}, name string) bool {
	if !rubyRespondSet["Enumerable#"+name] || !rubyUserObj(t) {
		return false
	}
	return rt.rubyIsA(t, rt.rubyBuiltinClass("Enumerable"))
}

// rubyEnumMixin answers such a method. Called without a block, #lazy and the
// methods rubyEnumDefers names do not run #each at all: they answer a Lazy over
// the receiver, or a DEFERRED Enumerator that runs the method when it is used
// (rubyEnumForce). Twins: enumMixin in ruby-interpreter.abnf, rbEnumMixin in
// lib/ruby-rt.metajs.
func (rt *jsrt) rubyEnumMixin(t interface{}, name string, args []interface{}) interface{} {
	if pos, blk := rubyBlockArgs(args); blk == nil {
		if name == "lazy" {
			lz := rubyMkLazy("each", &jsArray{}, 0, jsNull, false, true, "#<Enumerator::Lazy: "+rt.rubyInspect(t)+">")
			lz.set("src", t)
			return lz
		}
		if rubyEnumDefers(name, len(pos)) {
			e := rubyEnumAt(rubyMkEnumOp(&jsArray{}, name), t, name, pos)
			e.set("__rdefer", true)
			return e
		}
	}
	return rt.rubyEnumRun(t, name, args)
}

// rubyEnumDefers: does the Enumerable method name, given nargs arguments and no
// block, answer an Enumerator? Twin of rbaEnumDefers in lib/ruby-arr.js.
func rubyEnumDefers(name string, nargs int) bool {
	switch name {
	case "each_slice", "each_cons", "each_with_object":
		return nargs == 1
	case "cycle":
		return nargs <= 1
	case "map", "collect", "flat_map", "collect_concat", "select", "filter", "find_all", "reject",
		"find", "detect", "find_index", "group_by", "partition", "min_by", "max_by", "minmax_by",
		"sort_by", "take_while", "drop_while", "each_with_index", "each_entry", "reverse_each":
		return nargs == 0
	}
	return false
}

// rubyEnumForce runs a deferred Enumerator's method, once, and takes over the
// element list and pending op of the Enumerator it answers.
func (rt *jsrt) rubyEnumForce(t *jsObject) {
	t.set("__rdefer", false)
	meth, _ := t.props["meth"].(string)
	margs := append([]interface{}{}, t.props["margs"].(*jsArray).elems...)
	if e, isObj := rt.rubyEnumRun(t.props["src"], meth, margs).(*jsObject); isObj && rubyIsEnum(e) {
		t.set("items", e.props["items"])
		t.set("op", e.props["op"])
	}
}

// rubyEnumRun runs such a method: the values #each yields, packed as
// Enumerable#to_a packs them, run as an Array. The methods rubyEnumSpreads names
// get a multi-value yield's values as their block's arguments; the k-th call of
// each of them is for the k-th element. A method rubyEnumStop names ends #each
// early. Twins: enumRun in ruby-interpreter.abnf, which states the rules,
// rbEnumRun in lib/ruby-rt.metajs.
func (rt *jsrt) rubyEnumRun(t interface{}, name string, args []interface{}) interface{} {
	items := &jsArray{}
	var multi []bool
	var blk interface{}
	pos := args
	if n := len(args); n > 0 {
		if b, isBlk := rubyBlkOf(args[n-1]); isBlk {
			blk, pos = b, args[:n-1]
		}
	}
	kind := rubyEnumStop(name, blk != nil, len(pos))
	useBlk := kind > 0 && kind < 4 && blk != nil && (len(pos) == 0 || name == "find" || name == "detect")
	limit := -1
	if kind == 4 {
		switch {
		case len(pos) > 0:
			limit = int(math.Trunc(rt.rubyImplicitInt(name, pos[0], "")))
		case name == "first":
			limit = 1
		default:
			limit = 0
		}
		if limit < 0 {
			rt.rubyRaiseErr(name, "ArgumentError", "attempt to take negative size")
		}
		if len(pos) > 0 {
			args = append([]interface{}{float64(limit)}, args[1:]...)
		}
	}
	var answers []interface{}
	hits := 0
	tag := newJSObject()
	collect := &hostFunc{name: "", fn: func(rt *jsrt, _ uint64, a []interface{}) interface{} {
		var it interface{} = jsNull
		switch len(a) {
		case 0:
		case 1:
			it = a[0]
		default:
			it = &jsArray{elems: append([]interface{}{}, a...)}
		}
		hit := false
		if kind > 0 && kind < 4 {
			if useBlk {
				ba := []interface{}{it}
				if len(a) > 1 && rubyEnumSpreads(name) {
					ba = a
				}
				res := rt.call(blk, jsUndef, ba)
				answers = append(answers, res)
				hit = rubyTruthy(res)
			} else {
				hit = rubyTruthy(rt.rubyDelegArr(&jsArray{elems: []interface{}{it}}, t, rubyEnumProbe(name), pos))
			}
			if hit {
				hits++
			}
			switch kind {
			case 2:
				hit = !hit
			case 3:
				hit = hits > 1
			}
		}
		items.elems = append(items.elems, it)
		multi = append(multi, len(a) > 1)
		if kind == 4 {
			hit = len(items.elems) >= limit
		}
		if hit {
			rt.rubyThrow(tag, jsNull)
		}
		return jsNull
	}}
	if limit != 0 {
		rt.rubyCatch(tag, &hostFunc{name: "", fn: func(rt *jsrt, _ uint64, _ []interface{}) interface{} {
			return rt.rubyMethod(t, "each", []interface{}{rubyBlkWrap(collect)})
		}})
	}
	if useBlk {
		k := 0
		replay := &hostFunc{name: "", fn: func(_ *jsrt, _ uint64, _ []interface{}) interface{} {
			i := k
			k++
			return answers[i]
		}}
		args = append(append([]interface{}{}, pos...), rubyBlkWrap(replay))
	} else if n := len(args); n > 0 && rubyEnumSpreads(name) {
		if blk, isBlk := rubyBlkOf(args[n-1]); isBlk {
			k := 0
			spread := &hostFunc{name: "", fn: func(rt *jsrt, _ uint64, a []interface{}) interface{} {
				it := k
				k++
				if it < len(items.elems) && multi[it] && len(a) == 1 {
					if got, isArr := a[0].(*jsArray); isArr && got == items.elems[it].(*jsArray) {
						return rt.call(blk, jsUndef, got.elems)
					}
				}
				return rt.call(blk, jsUndef, a)
			}}
			args = append(append([]interface{}{}, args[:n-1]...), rubyBlkWrap(spread))
		}
	}
	return rt.rubyDelegArr(items, t, name, args)
}

// rubyMkUnbound is Module#instance_method on a user class: an UnboundMethod
// carrying __rubm (the name), __rown (the owner text of its #inspect) and __rcls
// (the class). Twins: mkUnbound in ruby-interpreter.abnf, rbMkUnbound in
// lib/ruby-rt.metajs.
func (rt *jsrt) rubyMkUnbound(cls *jsObject, n string) *jsObject {
	probe := newJSObject()
	probe.set("__class", cls)
	name, _ := cls.props["__name"].(string)
	own := ""
	if _, found := rubyFindMethod(probe, n); found {
		own = name
	} else if rubyRespondSet["Enumerable#"+n] && rt.rubyAncestorHas(cls, rt.rubyBuiltinClass("Enumerable")) {
		own = name + "(Enumerable)"
	}
	if own == "" {
		rt.rubyNameErr("NameError", "undefined method `"+n+"' for class `"+name+"'", n, cls)
	}
	u := newJSObject()
	u.set("__class", rt.rubyBuiltinClass("UnboundMethod"))
	u.set("__rubm", n)
	u.set("__rown", own)
	u.set("__rmod", own != name)
	u.set("__rcls", cls)
	return u
}

// rubyUnboundMethod: #name, #to_s and #bind, which answers the Method the
// receiver's #method would. A class's method binds only to an instance of it; a
// module's binds to any value, and then the Method keeps the UnboundMethod's
// owner. Twins: unboundMethod in ruby-interpreter.abnf, rbUnboundMethod in
// lib/ruby-rt.metajs.
func (rt *jsrt) rubyUnboundMethod(u *jsObject, name string, args []interface{}) interface{} {
	n, _ := u.props["__rubm"].(string)
	switch name {
	case "name":
		return jsSym{s: n}
	case "to_s":
		return rt.rubyInspect(u)
	case "bind":
		pos, _ := rubyBlockArgs(args)
		cls := u.props["__rcls"].(*jsObject)
		recv := argAt(pos, 0)
		if u.props["__rmod"] == true && !rt.rubyEnumAnswers(recv, n) {
			return rt.rubyMarkMethodVal(&hostFunc{name: "method:" + n,
				fn: func(rt *jsrt, this uint64, a []interface{}) interface{} {
					return rt.rubyMethod(recv, n, a)
				}}, rt.rubyStr(u.props["__rown"]), n)
		}
		if !rt.rubyIsA(recv, cls) {
			cn, _ := cls.props["__name"].(string)
			rt.rubyRaiseErr(name, "TypeError", "bind argument must be an instance of "+cn)
		}
		return rt.rubyMethod(argAt(pos, 0), "method", []interface{}{jsSym{s: n}})
	}
	rt.rubyNoMethod(u, name)
	return nil
}

func (rt *jsrt) rubyRespondsPrivate(n string) bool {
	if rubyRespondSet["private#"+n] {
		return true
	}
	_, top := rt.rubyBuiltinClass("Object").props["$p$"+n]
	return top
}

// rubyVisIMethods is Module#private_instance_methods (want "private") and
// #protected_instance_methods: the names each ancestor gives that visibility
// itself - its private ones are its __ipn list (ownIPNames in
// ruby-to-llvm-ir.abnf), the rest of __imn is protected under "$q$" - nearest
// first and once each, a nearer one of another visibility hiding it. With
// inheritance a class's private ones take in Object's: the top-level defs (its
// "$p$" slots) and the private Kernel methods. Twins: visInstanceMethods in
// ruby-interpreter.abnf, rbVisIMethods in lib/ruby-rt.metajs.
func (rt *jsrt) rubyVisIMethods(o *jsObject, want string, inherit bool) *jsArray {
	out := &jsArray{}
	seen := map[string]bool{}
	note := func(n, vis string) {
		if !seen[n] {
			seen[n] = true
			if vis == want {
				out.elems = append(out.elems, jsSym{s: n})
			}
		}
	}
	anc := []interface{}{o}
	if inherit {
		anc = rt.rubyAncestors(o).elems
	}
	for _, a := range anc {
		ao, isObj := a.(*jsObject)
		if !isObj {
			continue
		}
		if bn, isBuiltin := ao.props["__rbuiltin"].(string); isBuiltin {
			if bn == "Object" {
				for _, k := range ao.keys {
					if strings.HasPrefix(k, "$p$") {
						note(k[3:], "private")
					}
				}
				for _, n := range rubyRespondNames["private"] {
					note(n, "private")
				}
			}
			continue
		}
		if ns, isArr := ao.props["__ipn"].(*jsArray); isArr {
			for _, e := range ns.elems {
				note(e.(string), "private")
			}
		}
		if ns, isArr := ao.props["__imn"].(*jsArray); isArr {
			for _, e := range ns.elems {
				n := e.(string)
				if q, has := ao.props["$q$"+n]; has && !rubyIdenticalP(ao.props[n], q) {
					note(n, "protected")
				} else {
					note(n, "public")
				}
			}
		}
	}
	return out
}

// rubyRestricted: is n a private or protected method of a class chain (an
// instance's __class)? The emitter keeps such a method under "$q$" + n (methodKey
// in ruby-to-llvm-ir.abnf), and a nearer class holding only the plain name
// overrides it as a public one. A level whose plain name holds the "$q$" method
// itself was made public at run time (rubySetVis).
func rubyRestricted(cls interface{}, n string) bool {
	for cls != nil {
		clsObj, ok := cls.(*jsObject)
		if !ok {
			break
		}
		if q, has := clsObj.props["$q$"+n]; has {
			return !rubyIdenticalP(clsObj.props[n], q)
		}
		if _, has := clsObj.props[n]; has {
			return false
		}
		cls = clsObj.props["__super"]
	}
	return false
}

// rubySendKey is the key a call that visibility does not stop - #send, or one on
// the implicit self - reaches method n of target by: "$q$" + n for a restricted
// instance method (rubyRestricted), or for a private class method, kept under
// "$s$$q$" + n behind a raising "$s$" + n (emitMethods in ruby-to-llvm-ir.abnf).
// Twin of js_rvkey in lib/ruby-rt.metajs.
func rubySendKey(target interface{}, n string) string {
	o, isObj := target.(*jsObject)
	if !isObj {
		return n
	}
	if o.props["__isclass"] != true {
		if rubyRestricted(o.props["__class"], n) {
			return "$q$" + n
		}
		return n
	}
	for c := interface{}(o); c != nil; {
		co, ok := c.(*jsObject)
		if !ok {
			break
		}
		if _, has := co.props["$s$$q$"+n]; has {
			return "$q$" + n
		}
		if _, has := co.props["$s$"+n]; has {
			return n
		}
		c = co.props["__super"]
	}
	return n
}

// rubySetVis is Module#private / #protected / #public with method names (reached
// through send): the layout emitMethods gives a restricted method, made at run
// time - the method under "$q$" + n, "$pv$" + n marking a private one, and the
// plain name bound to a stub that raises. Public binds the plain name to the
// method itself. Answers the class, as MRI 2.6 does. Twins: setVisOn in
// ruby-interpreter.abnf, rbSetVis in lib/ruby-rt.metajs.
func (rt *jsrt) rubySetVis(c *jsObject, mode string, names []interface{}) interface{} {
	for _, nv := range names {
		n := rt.rubyStr(nv)
		key := n
		if rubyRestricted(c, n) {
			key = "$q$" + n
		}
		var f interface{}
		for cls := c; cls != nil && f == nil; {
			if v, has := cls.props[key]; has && isCallable(v) {
				f = v
			}
			cls, _ = cls.props["__super"].(*jsObject)
		}
		if f == nil {
			if rt.rubyMethodVis(c, n, true) != "" {
				continue
			}
			rt.rubyNameErr("NameError", "undefined method `"+n+"' for class `"+rt.rubyStr(c)+"'", n, c)
		}
		rubyVisRelist(c, n, mode == "private")
		if mode != "private" {
			c.del("$pv$" + n)
		}
		if mode == "public" {
			c.set(n, f)
			if _, has := c.props["$q$"+n]; has {
				c.set("$q$"+n, f)
			}
			continue
		}
		c.set("$q$"+n, f)
		if mode == "private" {
			c.set("$pv$"+n, true)
		}
		vis, name := mode, n
		c.set(n, &hostFunc{name: n, fn: func(rt *jsrt, this uint64, args []interface{}) interface{} {
			recv := argAt(args, 0)
			rt.rubyNameErr("NoMethodError", vis+" method `"+name+"' called for "+rt.rubyNoMethodDesc(recv), name, recv)
			return nil
		}})
	}
	return c
}

// rubyVisRelist keeps a class's own-name lists in step with a visibility set at
// run time: a private name is in __ipn, a public or protected one in __imn
// (ownIPNames / ownIMNames in ruby-to-llvm-ir.abnf). Twin of rbVisRelist in
// lib/ruby-rt.metajs.
func rubyVisRelist(c *jsObject, n string, private bool) {
	from, to := "__imn", "__ipn"
	if !private {
		from, to = to, from
	}
	if l, ok := c.props[from].(*jsArray); ok {
		kept := &jsArray{}
		for _, e := range l.elems {
			if e != n {
				kept.elems = append(kept.elems, e)
			}
		}
		c.set(from, kept)
	}
	out := &jsArray{}
	if l, ok := c.props[to].(*jsArray); ok {
		out.elems = append(out.elems, l.elems...)
	}
	for _, e := range out.elems {
		if e == n {
			return
		}
	}
	c.set(to, &jsArray{elems: append(out.elems, n)})
}

// rubyIMNHas: is n one of the public instance methods class c's bodies added (its
// __imn list), as a reopening body adds them to a builtin class.
func rubyIMNHas(c *jsObject, n string) bool {
	if ns, isArr := c.props["__imn"].(*jsArray); isArr {
		for _, e := range ns.elems {
			if e == n {
				return true
			}
		}
	}
	return false
}

// rubyMethodVis is the visibility of instance method n of a class, "" when it
// has none: Module#method_defined? and its public / private / protected twins,
// which see the class alone when inherit is false. A restricted method is kept
// under "$q$" + n, a private one also marked "$pv$" + n (emitMethods in
// ruby-to-llvm-ir.abnf). initialize and its kin are private whatever the body
// says; past the program's classes the methods a reopening body added to the
// builtin class and then the builtin table decide, with the private
// Kernel methods and the top-level defs. Twins: methodVisIn in
// ruby-interpreter.abnf, rbMethodVis in lib/ruby-rt.metajs.
func (rt *jsrt) rubyMethodVis(o *jsObject, n string, inherit bool) string {
	c := o
	for c != nil {
		if _, b := c.props["__rbuiltin"]; b {
			break
		}
		if q, has := c.props["$q$"+n]; has && !rubyIdenticalP(c.props[n], q) {
			if _, p := c.props["$pv$"+n]; p {
				return "private"
			}
			return "protected"
		}
		if v, has := c.props[n]; has && isCallable(v) {
			if n == "initialize" || n == "initialize_copy" || n == "respond_to_missing?" {
				return "private"
			}
			return "public"
		}
		if !inherit {
			return ""
		}
		c, _ = c.props["__super"].(*jsObject)
	}
	if c != nil && rubyIMNHas(c, n) {
		return "public"
	}
	if n == "initialize" || rt.rubyRespondsPrivate(n) {
		return "private"
	}
	bn := "Object"
	if c != nil {
		bn, _ = c.props["__rbuiltin"].(string)
	}
	if rubyResponds(bn, n) {
		return "public"
	}
	return ""
}

// rubyRespondClass is the builtin class respond_to? looks up, "" for a user
// instance or class (or anything else that answers from its own tables).
func (rt *jsrt) rubyRespondClass(v interface{}) string {
	if rxIsRegexObj(v) {
		return "Regexp"
	}
	if rxIsMatchObj(v) {
		return "MatchData"
	}
	if o, isObj := v.(*jsObject); isObj {
		if _, isCls := o.props["__isclass"]; isCls || rubyUserObj(o) {
			return ""
		}
	}
	return rt.rubyClassNameRT(v)
}

// rubyIvarAccess is Object#instance_variable_get / _set on an instance: an @ivar
// is the slot of its name without the "@". Twin of ivarAccess in
// ruby-interpreter.abnf.
func (rt *jsrt) rubyIvarAccess(o *jsObject, name string, args []interface{}) interface{} {
	pos, _ := rubyBlockArgs(args)
	n := rt.rubyStr(argAt(pos, 0))
	if !rubyIvarNameOk(n) {
		rt.rubyRaiseErr(name, "NameError", "`"+n+"' is not allowed as an instance variable name")
	}
	if name == "instance_variable_set" {
		v := argAt(pos, 1)
		if rubyClassObj(o) {
			rubyCivSet(o, n[1:], v)
		} else {
			o.set(n[1:], v)
		}
		return v
	}
	if v, has := o.props[n[1:]]; has {
		return v
	}
	return jsNull
}

// rubyIvarNames is Object#instance_variables on an instance: its @ivar slots in
// assignment order, the hidden __ slots left out. Twin of ivarNames in
// ruby-interpreter.abnf and rbIvarNames in lib/ruby-rt.metajs.
func rubyIvarNames(o *jsObject) *jsArray {
	out := &jsArray{}
	if rubyClassObj(o) {
		if l, ok := o.props["__civn"].(*jsArray); ok {
			for _, k := range l.elems {
				name, _ := k.(string)
				out.elems = append(out.elems, jsSym{s: "@" + name})
			}
		}
		return out
	}
	for _, k := range o.keys {
		if len(k) >= 2 && k[0] == '_' && k[1] == '_' {
			continue
		}
		out.elems = append(out.elems, jsSym{s: "@" + k})
	}
	return out
}

// rubyIvarNameOk: "@" and then an identifier (a letter, "_" or a non-ASCII
// character first, digits allowed after it).
func rubyIvarNameOk(n string) bool {
	if len(n) < 2 || n[0] != '@' {
		return false
	}
	for i := 1; i < len(n); i++ {
		c := n[i]
		ok := c == '_' || c >= 128 || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (i > 1 && c >= '0' && c <= '9')
		if !ok {
			return false
		}
	}
	return true
}

// rubyIntBit is Integer#[]: bit i of the infinite two's complement, 0 for a
// negative i. Twin of intBitAt in ruby-interpreter.abnf.
func (rt *jsrt) rubyIntBit(n, i interface{}) interface{} {
	if rubyToF(i) < 0 {
		return 0.0
	}
	return rt.rubyBin("&", rt.rubyBin(">>", n, i), 1.0)
}

func rubyExactInt(x float64) *big.Int {
	b, _ := big.NewFloat(x).Int(nil)
	return b
}

func rubyBigIntToF(x *big.Int) float64 {
	f, _ := new(big.Float).SetInt(x).Float64()
	return f
}

// rubyRatSimplest is MRI's nurat_rationalize_internal over exact bounds a < b;
// the answer is exact (ok is always true).
func rubyRatSimplest(a, b *big.Rat) (jsRat, bool) {
	p0, p1, q0, q1 := big.NewInt(0), big.NewInt(1), big.NewInt(1), big.NewInt(0)
	var c *big.Int
	for {
		c = new(big.Int).Neg(new(big.Int).Div(new(big.Int).Neg(a.Num()), a.Denom()))
		if new(big.Rat).SetInt(c).Cmp(b) < 0 {
			break
		}
		k := new(big.Int).Sub(c, big.NewInt(1))
		p2 := new(big.Int).Add(new(big.Int).Mul(k, p1), p0)
		q2 := new(big.Int).Add(new(big.Int).Mul(k, q1), q0)
		kr := new(big.Rat).SetInt(k)
		t := new(big.Rat).Inv(new(big.Rat).Sub(b, kr))
		b = new(big.Rat).Inv(new(big.Rat).Sub(a, kr))
		a = t
		p0, q0, p1, q1 = p1, q1, p2, q2
	}
	p := new(big.Int).Add(new(big.Int).Mul(c, p1), p0)
	q := new(big.Int).Add(new(big.Int).Mul(c, q1), q0)
	return rubyMkRatB(p, q), true
}

func rubyFiniteRat(p, q float64) (jsRat, bool) {
	if math.IsInf(p, 0) || math.IsInf(q, 0) {
		return jsRat{}, false
	}
	return rubyMkRat(p, q), true
}

// rubyFloSimplest is the same walk in doubles, as MRI does once eps made the
// bounds Floats. float64() on each product stops arm64 fusing it into an FMA
// (manual chapter 4), which the other two engines cannot do.
func rubyFloSimplest(a, b float64) (jsRat, bool) {
	p0, p1, q0, q1 := 0.0, 1.0, 1.0, 0.0
	c := 0.0
	for {
		c = math.Ceil(a)
		if c < b {
			break
		}
		k := c - 1
		p2 := float64(k*p1) + p0
		q2 := float64(k*q1) + q0
		t := 1 / (b - k)
		b = 1 / (a - k)
		a = t
		p0, q0, p1, q1 = p1, q1, p2, q2
	}
	return rubyFiniteRat(float64(c*p1)+p0, float64(c*q1)+q0)
}

// rubyCpxAbs is MRI's Complex#abs: with one part zero it is the other part's
// own abs, a Float only if either part is one. Twin of cpxAbs in
// languages/ruby-interpreter.abnf.
func (rt *jsrt) rubyCpxAbs(t jsCpx) interface{} {
	_, reF := t.re.(jsFlo)
	_, imF := t.im.(jsFlo)
	if rt.rubyEq(t.re, 0.0) {
		a := rt.rubyRealAbs(t.im)
		if reF && !imF {
			return jsFlo{f: rubyToF(a)}
		}
		return a
	}
	if rt.rubyEq(t.im, 0.0) {
		a := rt.rubyRealAbs(t.re)
		if !reF && imF {
			return jsFlo{f: rubyToF(a)}
		}
		return a
	}
	return jsFlo{f: rubyHypot(rubyToF(t.re), rubyToF(t.im))}
}

// rubyCpxMethod is Complex's own surface. Twin of cpxMethod in
// languages/ruby-interpreter.abnf, which states the rules.
func (rt *jsrt) rubyCpxMethod(t jsCpx, name string) (interface{}, bool) {
	switch name {
	case "conjugate", "conj":
		return rt.rubyCpxConj(t), true
	case "rect", "rectangular":
		return &jsArray{elems: []interface{}{t.re, t.im}}, true
	case "abs", "magnitude":
		return rt.rubyCpxAbs(t), true
	case "polar":
		return &jsArray{elems: []interface{}{rt.rubyCpxAbs(t), jsFlo{f: math.Atan2(rubyToF(t.im), rubyToF(t.re))}}}, true
	case "arg", "angle", "phase":
		return jsFlo{f: math.Atan2(rubyToF(t.im), rubyToF(t.re))}, true
	case "zero?":
		return rt.rubyEq(t.re, 0.0) && rt.rubyEq(t.im, 0.0), true
	case "real?":
		return false, true
	case "round", "floor", "ceil", "truncate", "positive?", "negative?", "div", "divmod", "modulo", "remainder":
		rt.rubyNoMethod(t, name)
	}
	into := map[string]string{"to_i": "Integer", "to_int": "Integer", "to_f": "Float", "to_r": "Rational", "rationalize": "Rational"}[name]
	if into != "" {
		if _, imF := t.im.(jsFlo); imF || !rt.rubyEq(t.im, 0.0) {
			rt.rubyRaiseErr(name, "RangeError", "can't convert "+rt.rubyStr(t)+" into "+into)
		}
		return rt.rubyMethod(t.re, name, nil), true
	}
	return nil, false
}

// rubyCpxBin is Complex arithmetic on the TYPED parts. Twin of cpxBin in
// languages/ruby-interpreter.abnf, which states the rules.
func (rt *jsrt) rubyCpxBin(op string, l, r interface{}) interface{} {
	rc, rIsCpx := r.(jsCpx)
	if !rIsCpx {
		lc := l.(jsCpx)
		switch op {
		case "+":
			return jsCpx{re: rt.rubyNumBin("+", lc.re, r), im: lc.im}
		case "-":
			return jsCpx{re: rt.rubyNumBin("-", lc.re, r), im: lc.im}
		case "*":
			return jsCpx{re: rt.rubyNumBin("*", lc.re, r), im: rt.rubyNumBin("*", lc.im, r)}
		case "/":
			return jsCpx{re: rt.rubyRealQuo(lc.re, r), im: rt.rubyRealQuo(lc.im, r)}
		}
		rt.fail("Complex does not support '%s'", op)
	}
	a, aIsCpx := l.(jsCpx)
	if !aIsCpx {
		a = jsCpx{re: l, im: 0.0}
	}
	switch op {
	case "+":
		return jsCpx{re: rt.rubyNumBin("+", a.re, rc.re), im: rt.rubyNumBin("+", a.im, rc.im)}
	case "-":
		return jsCpx{re: rt.rubyNumBin("-", a.re, rc.re), im: rt.rubyNumBin("-", a.im, rc.im)}
	case "*":
		return rt.rubyCpxMul(a, rc)
	case "/":
		nb := rt.rubyNumBin
		q := rt.rubyRealQuo
		var x, y interface{}
		if c, _ := rt.rubySpaceship(rt.rubyRealAbs(rc.re), rt.rubyRealAbs(rc.im)); c > 0 {
			q1 := q(rc.im, rc.re)
			n1 := nb("*", rc.re, nb("+", 1.0, nb("*", q1, q1)))
			x = nb("+", q(a.re, n1), q(nb("*", a.im, q1), n1))
			y = nb("-", q(a.im, n1), q(nb("*", a.re, q1), n1))
		} else {
			q2 := q(rc.re, rc.im)
			n2 := nb("*", rc.im, nb("+", 1.0, nb("*", q2, q2)))
			x = nb("+", q(nb("*", a.re, q2), n2), q(a.im, n2))
			y = nb("-", q(nb("*", a.im, q2), n2), q(a.re, n2))
		}
		for _, p := range []interface{}{a.re, a.im, rc.re, rc.im} {
			if _, f := p.(jsFlo); f {
				return jsCpx{re: x, im: y}
			}
		}
		return jsCpx{re: rubyRatCanon(x), im: rubyRatCanon(y)}
	}
	rt.fail("Complex does not support '%s'", op)
	return nil
}

func rubyRatCanon(v interface{}) interface{} {
	if r, ok := v.(jsRat); ok && r.d == 1 {
		if r.bn != nil {
			return &jsBigInt{v: r.bn}
		}
		return r.n
	}
	return v
}

func (rt *jsrt) rubyCpxMul(a, b jsCpx) jsCpx {
	nb := rt.rubyNumBin
	return jsCpx{re: nb("-", nb("*", a.re, b.re), nb("*", a.im, b.im)),
		im: nb("+", nb("*", a.re, b.im), nb("*", a.im, b.re))}
}

func (rt *jsrt) rubyCpxConj(c jsCpx) jsCpx { return jsCpx{re: c.re, im: rt.rubyNeg(c.im)} }

// Numeric#quo on two reals. Twin of realQuo in languages/ruby-interpreter.abnf.
func (rt *jsrt) rubyRealQuo(a, b interface{}) interface{} {
	_, af := a.(jsFlo)
	_, bf := b.(jsFlo)
	if af || bf {
		return rt.rubyNumBin("/", a, b)
	}
	if rt.rubyEq(b, 0.0) {
		rt.rubyRaiseErr("", "ZeroDivisionError", "divided by 0")
	}
	return rt.rubyNumBin("/", rt.rubyAsRat(a), b)
}

func (rt *jsrt) rubyRealAbs(v interface{}) interface{} {
	if f, ok := v.(jsFlo); ok {
		return jsFlo{f: math.Abs(f.f)}
	}
	if c, _ := rt.rubySpaceship(v, 0.0); c < 0 {
		return rt.rubyNeg(v)
	}
	return v
}

// rubyCpxText is Complex#to_s (insp false) and the inside of #inspect: MRI's
// f_format. Twin of cpxText in languages/ruby-interpreter.abnf.
func (rt *jsrt) rubyCpxText(t jsCpx, insp bool) string {
	render := rt.rubyStr
	if insp {
		render = rt.rubyInspect
	}
	pos := true
	if f, ok := t.im.(jsFlo); ok {
		pos = !math.Signbit(f.f)
	} else if c, _ := rt.rubySpaceship(t.im, 0.0); c < 0 {
		pos = false
	}
	sign := "+"
	if !pos {
		sign = "-"
	}
	it := render(rt.rubyRealAbs(t.im))
	out := render(t.re) + sign + it
	if lc := it[len(it)-1]; lc < '0' || lc > '9' {
		out += "*"
	}
	return out + "i"
}

// rubyNumMore is Numeric's remaining public methods for an Integer, a Float and a
// Rational, reached once rubyNumMethod has no arm of its own. Twin of numMore in
// languages/ruby-interpreter.abnf, which states the rules.
func (rt *jsrt) rubyNumMore(t interface{}, name string, args []interface{}) (interface{}, bool) {
	pos, _ := rubyBlockArgs(args)
	a0 := argAt(pos, 0)
	_, isF := t.(jsFlo)
	ti, isI := rubyIntBig(t)
	sign := func(v interface{}) int { c, _ := rt.rubySpaceship(v, 0.0); return c }
	switch name {
	case "dup", "clone":
		return t, true
	case "nonzero?":
		if rt.rubyEq(t, 0.0) {
			return jsNull, true
		}
		return t, true
	}
	if c, isCpx := t.(jsCpx); isCpx {
		_, cd := a0.(jsCpx)
		switch name {
		case "abs2":
			return rt.rubyNumBin("+", rt.rubyNumBin("*", c.re, c.re), rt.rubyNumBin("*", c.im, c.im)), true
		case "coerce":
			if cd {
				return &jsArray{elems: []interface{}{a0, t}}, true
			}
			return &jsArray{elems: []interface{}{jsCpx{re: a0, im: 0.0}, t}}, true
		case "quo":
			if !cd {
				re, _ := rt.rubyNumMore(c.re, "quo", pos)
				im, _ := rt.rubyNumMore(c.im, "quo", pos)
				return jsCpx{re: re, im: im}, true
			}
		case "fdiv":
			if !cd {
				return jsCpx{re: jsFlo{f: rubyToF(c.re) / rubyToF(a0)}, im: jsFlo{f: rubyToF(c.im) / rubyToF(a0)}}, true
			}
		}
		return nil, false
	}
	switch name {
	case "abs2":
		return rt.rubyNumBin("*", t, t), true
	case "conj", "conjugate":
		return t, true
	case "real?":
		return true, true
	case "rect", "rectangular":
		return &jsArray{elems: []interface{}{t, 0.0}}, true
	case "i":
		return jsCpx{re: 0.0, im: t}, true
	case "arg", "angle", "phase", "polar":
		var ar interface{} = 0.0
		if f, ok := t.(jsFlo); ok {
			if f.f != f.f {
				ar = t
			} else if math.Signbit(f.f) {
				ar = jsFlo{f: math.Pi}
			}
		} else if sign(t) < 0 {
			ar = jsFlo{f: math.Pi}
		}
		if name == "polar" {
			return &jsArray{elems: []interface{}{rt.rubyMethod(t, "abs", nil), ar}}, true
		}
		return ar, true
	case "coerce":
		_, coI := rubyIntBig(a0)
		if isI {
			if coI {
				return &jsArray{elems: []interface{}{a0, t}}, true
			}
			return &jsArray{elems: []interface{}{jsFlo{f: rubyToF(a0)}, jsFlo{f: rubyToF(t)}}}, true
		}
		if _, isR := t.(jsRat); isR {
			if coI {
				return &jsArray{elems: []interface{}{rt.rubyAsRat(a0), t}}, true
			}
			if _, coF := a0.(jsFlo); coF {
				return &jsArray{elems: []interface{}{a0, jsFlo{f: rubyToF(t)}}}, true
			}
			return &jsArray{elems: []interface{}{a0, t}}, true
		}
		return &jsArray{elems: []interface{}{jsFlo{f: rubyToF(a0)}, t}}, true
	case "quo":
		if _, qF := a0.(jsFlo); isF || qF {
			return rt.rubyNumBin("/", jsFlo{f: rubyToF(t)}, a0), true
		}
		if rt.rubyEq(a0, 0.0) {
			rt.rubyRaiseErr(name, "ZeroDivisionError", "divided by 0")
		}
		return rt.rubyNumBin("/", rt.rubyAsRat(t), rt.rubyAsRat(a0)), true
	case "remainder":
		m := rt.rubyNumBin("%", t, a0)
		if !rt.rubyEq(m, 0.0) && (sign(t) < 0) != (sign(a0) < 0) {
			return rt.rubyNumBin("-", m, a0), true
		}
		return m, true
	}
	if f, ok := t.(jsFlo); ok {
		switch name {
		case "next_float":
			return jsFlo{f: math.Nextafter(f.f, math.Inf(1))}, true
		case "prev_float":
			return jsFlo{f: math.Nextafter(f.f, math.Inf(-1))}, true
		}
		return nil, false
	}
	if !isI {
		return nil, false
	}
	narrow := rt.rubyBigNarrow
	switch name {
	case "ord":
		return t, true
	case "gcd", "lcm", "gcdlcm":
		y, yI := rubyIntBig(a0)
		if !yI {
			rt.rubyRaiseErr(name, "TypeError", "not an integer")
		}
		g := new(big.Int).GCD(nil, nil, new(big.Int).Abs(ti), new(big.Int).Abs(y))
		if name == "gcd" {
			return narrow(g), true
		}
		l := new(big.Int)
		if g.Sign() != 0 {
			l.Abs(l.Mul(new(big.Int).Quo(ti, g), y))
		}
		if name == "lcm" {
			return narrow(l), true
		}
		return &jsArray{elems: []interface{}{narrow(g), narrow(l)}}, true
	case "allbits?", "anybits?", "nobits?":
		m, _ := rubyIntBig(a0)
		and := new(big.Int).And(ti, m)
		switch name {
		case "allbits?":
			return and.Cmp(m) == 0, true
		case "anybits?":
			return and.Sign() != 0, true
		}
		return and.Sign() == 0, true
	case "bit_length":
		return float64(rubyBitLen(ti)), true
	case "size":
		if n := (new(big.Int).Abs(ti).BitLen() + 7) / 8; n > 8 {
			return float64(n), true
		}
		return 8.0, true
	case "digits":
		base := big.NewInt(10)
		if len(pos) > 0 {
			base, _ = rubyIntBig(a0)
		}
		if base.Sign() < 0 {
			rt.rubyRaiseErr(name, "ArgumentError", "negative radix")
		}
		if base.Cmp(big.NewInt(2)) < 0 {
			rt.rubyRaiseErr(name, "ArgumentError", "invalid radix "+base.String())
		}
		if ti.Sign() < 0 {
			rt.rubyRaiseErr(name, "Math::DomainError", "out of domain")
		}
		out := &jsArray{}
		v := new(big.Int).Set(ti)
		for {
			q, r := new(big.Int).QuoRem(v, base, new(big.Int))
			out.elems = append(out.elems, narrow(r))
			if q.Sign() == 0 {
				return out, true
			}
			v = q
		}
	case "pow":
		if len(pos) < 2 {
			return rt.rubyBinAs(name, "**", t, a0), true
		}
		e, eI := rubyIntBig(a0)
		if !eI {
			rt.rubyRaiseErr(name, "TypeError", "Integer#pow() 2nd argument not allowed unless a 1st argument is integer")
		}
		m, mI := rubyIntBig(argAt(pos, 1))
		if !mI {
			rt.rubyRaiseErr(name, "TypeError", "Integer#pow() 2nd argument not allowed unless all arguments are integers")
		}
		if e.Sign() < 0 {
			rt.rubyRaiseErr(name, "RangeError", "Integer#pow() 1st argument cannot be negative when 2nd argument specified")
		}
		if e.Sign() == 0 {
			if m.Sign() > 0 {
				return 1.0, true
			}
			return narrow(new(big.Int).Add(m, big.NewInt(1))), true
		}
		if m.Sign() == 0 {
			rt.rubyRaiseErr(name, "ZeroDivisionError", "divided by 0")
		}
		am := new(big.Int).Abs(m)
		r := new(big.Int).Exp(new(big.Int).Mod(ti, am), e, am)
		r.Mod(r, am)
		if m.Sign() < 0 && r.Sign() != 0 {
			r.Add(r, m)
		}
		return narrow(r), true
	case "[]":
		return rt.rubyIntBit(t, a0), true
	}
	return nil, false
}

// rubyIntBig is an Integer value as a big.Int; ok=false for anything else.
func rubyIntBig(v interface{}) (*big.Int, bool) {
	switch v.(type) {
	case float64, *jsBigInt:
		return rubyBigOf(v)
	}
	return nil, false
}

// rubyBitLen is Integer#bit_length: the bits of x, or of ~x for a negative x.
func rubyBitLen(x *big.Int) int {
	if x.Sign() < 0 {
		return new(big.Int).Not(x).BitLen()
	}
	return x.BitLen()
}

// rubySuperBuiltin is a super that no compiled class or module above answers:
// the builtin Object / Kernel / Exception method of that name on a user
// instance, else NoMethodError. Comparable#== is compiled into the class
// (comparableMethods), so == here is identity. Twin of superBuiltin in
// ruby-interpreter.abnf, which states the rules.
func (rt *jsrt) rubySuperBuiltin(self interface{}, name string, args []interface{}) interface{} {
	// A class method ("$s$" key, or a method of a module the class extends):
	// Class#new constructs, and the hooks are empty.
	sn, isSing := strings.CutPrefix(name, "$s$")
	if so, isObj := self.(*jsObject); isSing || (isObj && so.props["__isclass"] == true) {
		if sn == "new" {
			return rt.rubyNew(self, args)
		}
		if rubyHookName(sn) {
			return jsNull
		}
		if isSing {
			rt.rubyNoSuper(self, sn)
		}
	}
	o, isObj := self.(*jsObject)
	if !isObj || !rubyUserObj(o) || o.props["__isclass"] == true {
		rt.rubyNoSuper(self, name)
	}
	pos, _ := rubyBlockArgs(args)
	switch name {
	case "initialize":
		if len(pos) > 0 {
			rt.rubyRaiseErr(name, "ArgumentError", fmt.Sprintf("wrong number of arguments (given %d, expected 0)", len(pos)))
		}
		return jsNull
	case "to_s", "inspect":
		if msg, ok := rt.rubyExcMessage(o); ok && name == "to_s" && rubyExcInst(o) {
			return msg
		}
		cn, _ := o.props["__class"].(*jsObject).props["__name"].(string)
		return "#<" + cn + ">"
	case "message":
		if rubyExcInst(o) {
			return rt.rubyStr(o)
		}
	case "hash":
		return rubyStrHash("o" + rt.rubyInspect(rt.rubyObjectId(o)))
	case "object_id":
		return rt.rubyObjectId(o)
	case "==", "equal?", "eql?":
		return rubyIdenticalP(o, argAt(pos, 0))
	case "!=":
		return !rt.rubyEq(o, argAt(pos, 0))
	case "frozen?":
		fz, _ := o.props["__fz"].(bool)
		return fz
	case "freeze":
		o.set("__fz", true)
		return o
	case "dup", "clone":
		return rubySingCopy(o, name == "clone")
	case "respond_to?":
		return rt.rubyRespond(o, pos)
	case "respond_to_missing?":
		return false
	}
	rt.rubyNoSuper(self, name)
	return nil
}

// rubyNew is Class#new: an instance of a compiled class with its initialize run.
// Proc.new { } is its block, Array / Hash / Regexp / Struct have their own
// constructors, and an exception class makes the {__class, args} shape the rescue
// clauses read. Twins: js_rnew in lib/ruby-rt.metajs, newInstance in
// ruby-interpreter.abnf.
func (rt *jsrt) rubyNew(clsV interface{}, argv []interface{}) interface{} {
	if cls, isObj := clsV.(*jsObject); isObj {
		if bn, isBuiltin := cls.props["__rbuiltin"].(string); isBuiltin {
			switch bn {
			case "Proc":
				if _, blk := rubyBlockArgs(argv); blk != nil {
					return blk
				}
				return jsNull
			case "Array":
				return rt.rubyArrNew(argv)
			case "Hash":
				return rubyNewHash(argv)
			case "Regexp":
				return rt.rubyRegexNew(argv)
			case "Struct":
				return rt.rubyStructNew(argv)
			}
			return rt.rubyExc(cls, argv)
		}
	}
	inst := newJSObject()
	inst.set("__class", clsV)
	rt.rubyMethod(inst, "initialize", argv)
	return inst
}

// rubyHookName: the Class / Module callbacks, empty unless a program defines them.
// Twins: isHookName in ruby-interpreter.abnf, rbHookName in lib/ruby-rt.metajs.
func rubyHookName(n string) bool {
	switch n {
	case "inherited", "included", "extended", "prepended", "method_added":
		return true
	}
	return false
}

// rubyHook runs the class method `name` of a module or class (a class's own or
// an inherited one) with arg, for a run-time extend / include / define_method.
// Twins: runHook in ruby-interpreter.abnf, rbHook in lib/ruby-rt.metajs.
func (rt *jsrt) rubyHook(recv interface{}, name string, arg interface{}) {
	for c := recv; ; {
		o, isObj := c.(*jsObject)
		if !isObj {
			return
		}
		if f, has := o.props["$s$"+name]; has && isCallable(f) {
			rt.call(f, jsUndef, []interface{}{recv, arg})
			return
		}
		c = o.props["__super"]
	}
}

// rubyNoSuper raises the NoMethodError of a super with nothing above it. Twin of
// noSuper in ruby-interpreter.abnf.
func (rt *jsrt) rubyNoSuper(self interface{}, name string) {
	rt.rubyNameErr("NoMethodError", "super: no superclass method `"+name+"' for "+rt.rubyNoMethodDesc(self), name, self)
}

// rubyObjectId is Object#object_id. Twin of objectId in
// languages/ruby-interpreter.abnf, which states the rules.
func (rt *jsrt) rubyObjectId(v interface{}) interface{} {
	if isUndefOrNull(v) {
		return 8.0
	}
	if b, ok := v.(bool); ok {
		if b {
			return 20.0
		}
		return 0.0
	}
	if _, isI := rubyIntBig(v); isI {
		return rt.rubyNumBin("+", rt.rubyNumBin("*", v, 2.0), 1.0)
	}
	valued := false
	switch v.(type) {
	case string, jsRat, jsCpx:
		valued = true
	}
	for i, o := range rt.rubyOids {
		if rubyIdenticalP(o, v) || (valued && rt.rubyEql(o, v)) {
			return float64(16 + 8*i)
		}
	}
	rt.rubyOids = append(rt.rubyOids, v)
	return float64(8 + 8*len(rt.rubyOids))
}

// rubyStrHash is the 31-bit polynomial hash over a string's UTF-16 units.
func rubyStrHash(s string) float64 {
	h := int64(0)
	for _, u := range utf16.Encode([]rune(s)) {
		h = (h*31 + int64(u)) % 2147483647
	}
	return float64(h)
}

// rubyHashOf is Object#hash, by value. Twin of valHash in
// languages/ruby-interpreter.abnf, which states the rules.
func (rt *jsrt) rubyHashOf(v interface{}) float64 {
	if a, ok := v.(*jsArray); ok {
		h := int64(7)
		for _, e := range a.elems {
			h = (h*31 + int64(rt.rubyHashOf(e))) % 2147483647
		}
		return float64(h)
	}
	if keys, vals, isDict := dictParts(v); isDict {
		h := int64(11)
		for i := range keys.elems {
			h = (h + (int64(rt.rubyHashOf(keys.elems[i]))*31+int64(rt.rubyHashOf(vals.elems[i])))%2147483647) % 2147483647
		}
		return float64(h)
	}
	if _, hasOwn := rubyFindMethod(v, "hash"); hasOwn && rubyUserObj(v) {
		return rubyToF(rt.rubyNumBin("%", rt.rubyMethod(v, "hash", nil), 2147483647.0))
	}
	if isUndefOrNull(v) {
		return rubyStrHash("n" + rt.rubyInspect(v))
	}
	switch t := v.(type) {
	case bool:
		return rubyStrHash("n" + rt.rubyInspect(v))
	case float64, *jsBigInt:
		return rubyStrHash("i" + rt.rubyInspect(v))
	case string:
		return rubyStrHash("s" + t)
	case *jsRStr:
		if t.bin != nil {
			return rubyStrHash("s" + rubyBinText(t.bin.b))
		}
		return rubyStrHash("s" + t.s)
	case *jsRBin:
		if t.str != nil {
			return rt.rubyHashOf(t.str)
		}
		return rubyStrHash("s" + rubyBinText(t.b))
	case jsSym:
		return rubyStrHash("y" + t.s)
	case jsFlo:
		return rubyStrHash("f" + rt.rubyInspect(v))
	case jsRat:
		return rubyStrHash("r" + rt.rubyInspect(v))
	case jsCpx:
		return rubyStrHash("c" + rt.rubyInspect(v))
	}
	if _, isRng := rubyIsRange(v); isRng {
		return rubyStrHash("g" + rt.rubyInspect(v))
	}
	if rxIsRegexObj(v) {
		return rubyStrHash("x" + rt.rubyInspect(v))
	}
	return rubyStrHash("o" + rt.rubyInspect(rt.rubyObjectId(v)))
}

// rubyEqlDeep is #eql? on a builtin value. Twin of valEql in
// languages/ruby-interpreter.abnf.
func (rt *jsrt) rubyEqlDeep(a, b interface{}) bool {
	la, aArr := a.(*jsArray)
	lb, bArr := b.(*jsArray)
	if !aArr || !bArr {
		return rt.rubyEql(a, b)
	}
	if len(la.elems) != len(lb.elems) {
		return false
	}
	for i := range la.elems {
		if !rt.rubyEqlDeep(la.elems[i], lb.elems[i]) {
			return false
		}
	}
	return true
}

// rubyClassObj: a class or module object.
func rubyClassObj(o *jsObject) bool {
	_, isCls := o.props["__isclass"]
	return isCls
}

// rubyCivSet writes a class object's @ivar. Its instance methods are plain keys
// of the same object, so the @ivar names are recorded in __civn, in assignment
// order, for #instance_variables. Twin of js_rcivset in lib/ruby-rt.metajs.
func rubyCivSet(o *jsObject, k string, v interface{}) {
	l, ok := o.props["__civn"].(*jsArray)
	if !ok {
		l = &jsArray{}
		o.set("__civn", l)
	}
	found := false
	for _, e := range l.elems {
		if e == interface{}(k) {
			found = true
			break
		}
	}
	if !found {
		l.elems = append(l.elems, k)
	}
	o.set(k, v)
}

// rubyScopeSelf is the implicit receiver at scope h: the nearest `self` binding,
// or the main object where that is nil (the top level and its defs, see
// emitIvarSelf in ruby-to-llvm-ir.abnf). Twin of rbScopeSelf in
// languages/lib/ruby-rt.metajs.
func rubyScopeSelf(rt *jsrt, h uint64) interface{} {
	var self, main interface{} = jsNull, jsNull
	found := false
	for sc := rt.scopeOf(h); sc != nil; sc = sc.parent {
		if v, ok := sc.get("self"); ok && !found {
			self, found = v, true
		}
		if v, ok := sc.get("$main"); ok {
			main = v
		}
	}
	if isNullish(self) {
		return main
	}
	return self
}

// rubyCasecmp is String#casecmp: ASCII folded to UPPER case, then MRI's byte
// order, which is code-point order and therefore Go's string order. Twins:
// rbCasecmp in ruby-interpreter.abnf and languages/lib/ruby-rt.metajs.
func rubyCasecmp(a, b string) int {
	up := func(s string) string {
		return strings.Map(func(r rune) rune {
			if r >= 'a' && r <= 'z' {
				return r - 32
			}
			return r
		}, s)
	}
	return strings.Compare(up(a), up(b))
}

// rubySymInspect is Symbol#inspect: the name bare after the colon when MRI's
// symname test passes (rubySymPlain), else quoted as :"a b". Twins: symInspect in
// ruby-interpreter.abnf, rbSymInspect in languages/lib/ruby-rt.metajs.
func rubySymInspect(s string) string {
	if rubySymPlain(s) {
		return ":" + s
	}
	return ":" + rubyStrInspect(s)
}

func rubySymIdCh(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c >= 0x80
}

// rubySymPlain: an operator method name, an identifier with one trailing ? ! or =,
// an @ivar, an @@cvar, or a global ($name, $1, $-w, $~ and the other one-character
// specials). Twins: rbSymPlain in ruby-interpreter.abnf and
// languages/lib/ruby-rt.metajs.
func rubySymPlain(s string) bool {
	n := len(s)
	if n == 0 {
		return false
	}
	switch s {
	case "+", "-", "*", "/", "%", "**", "==", "===", "!=", "=~", "!~", "!", "~", "+@", "-@", "[]", "[]=",
		"<", "<=", ">", ">=", "<=>", "<<", ">>", "&", "|", "^", "`":
		return true
	}
	i := 0
	switch s[0] {
	case '$':
		if n == 1 {
			return false
		}
		if n == 2 && strings.IndexByte("~*$?!@/\\;,.=:<>\"&`'+0", s[1]) >= 0 {
			return true
		}
		if s[1] == '-' {
			return n == 3 && rubySymIdCh(s[2])
		}
		if s[1] >= '0' && s[1] <= '9' {
			k := 2
			for k < n && s[k] >= '0' && s[k] <= '9' {
				k++
			}
			return k == n
		}
		i = 1
	case '@':
		i = 1
		if n > 1 && s[1] == '@' {
			i = 2
		}
	}
	if i >= n || !rubySymIdCh(s[i]) || s[i] >= '0' && s[i] <= '9' {
		return false
	}
	for i < n && rubySymIdCh(s[i]) {
		i++
	}
	if i == n {
		return true
	}
	t := s[i]
	return s[0] != '$' && s[0] != '@' && i == n-1 && (t == '?' || t == '!' || t == '=')
}

// rubyBuiltinExtFind is a method a program added to a builtin class by reopening
// it, for a builtin value (not an instance or a class object); it wins over the
// builtin method of the same name, as in MRI. Twins: the builtinReopened arm of
// mcall in ruby-interpreter.abnf, rbBuiltinExtFind in languages/lib/ruby-rt.metajs.
func (rt *jsrt) rubyBuiltinExtFind(v interface{}, name string) (interface{}, bool) {
	if o, isObj := v.(*jsObject); isObj {
		if _, inst := o.props["__class"]; inst {
			return nil, false
		}
		if _, cls := o.props["__isclass"]; cls {
			return nil, false
		}
	}
	// The value's class, then its builtin superclasses (a method a program gave
	// Numeric reaches an Integer).
	for cn := rubyClassName(v); cn != ""; cn = rubyExcParent[cn] {
		if bm, has := rt.rubyBuiltinClass(cn).props[name]; has && isCallable(bm) {
			return bm, true
		}
	}
	return nil, false
}

// jsRStr is a MUTABLE Ruby String: what a string literal, a method call, a String
// operator and to_s evaluate to, and what a mutator makes of a plain string it
// reaches through a variable or an element. The rule, and what it leaves out, are
// at isRStr in languages/ruby-interpreter.abnf. It is a pointer, so every alias
// shares it; fz is #frozen?. Twin of the {__rs, __fz} box of lib/ruby-rt.metajs.
//
// A String re-encoded in place (<< of a binary String, force_encoding) becomes
// the binary String bin: the other engines change the box's own tags
// (becomeBin in ruby-interpreter.abnf), and a Go value cannot change its type,
// so every alias reaches bin through this one (rubyFwd).
type jsRStr struct {
	s   string
	fz  bool
	id  jsHandleID
	bin *jsRBin
}

// rubyFwd is what a value stands for: the binary String a re-encoded box
// became, or the String a re-encoded binary String became, else the value
// itself.
func rubyFwd(v interface{}) interface{} {
	switch t := v.(type) {
	case *jsRStr:
		if t.bin != nil {
			return rubyFwd(t.bin)
		}
	case *jsRBin:
		if t.str != nil {
			return rubyFwd(t.str)
		}
	}
	return v
}

// rubyFwdSelf is a method answer of the binary String b became, with that
// String itself answered as b, the object the program holds.
func rubyFwdSelf(b *jsRStr, r interface{}) interface{} {
	if bn, ok := r.(*jsRBin); ok && bn == b.bin {
		return b
	}
	return r
}

// rubyBoxOut is a method call's answer as the program sees it: a String the call
// built is a new object, so a plain string becomes a box, and an Array a String
// method answers (split, chars, lines, scan, partition) holds boxes - such an
// Array is always new. Twin of boxOut in ruby-interpreter.abnf and the tail of
// js_rxmcall in lib/ruby-rt.metajs.
func rubyBoxOut(recv, v interface{}) interface{} {
	switch t := v.(type) {
	case string:
		return &jsRStr{s: t}
	case *jsArray:
		switch recv.(type) {
		case string, *jsRStr:
			for i, e := range t.elems {
				if s, ok := e.(string); ok {
					t.elems[i] = &jsRStr{s: s}
				}
			}
		}
	}
	return v
}

// rubySV reads a String box as its text and hands every other value back.
func rubySV(v interface{}) interface{} {
	switch b := rubyFwd(v).(type) {
	case *jsRStr:
		return b.s
	case *jsRBin:
		return b
	}
	return v
}

// rubyKeyOut is a Hash key as a reader sees it. A String key is stored as its
// text (dictAppend) and read out as a FROZEN String, which is what MRI's key - a
// frozen copy of the String it was stored under - is. MRI deduplicates that copy
// (the fstring table a frozen literal is also in), so every read of one text is the
// same object. Twin of keyOut in ruby-interpreter.abnf and rbKeyOut in
// lib/ruby-rt.metajs.
func (rt *jsrt) rubyKeyOut(k interface{}) interface{} {
	if s, ok := k.(string); ok {
		return rt.rubyFzLit(s, true)
	}
	return k
}

// rubySVArgs is rubySV over an argument list, copying only when a box is in it.
func rubySVArgs(args []interface{}) []interface{} {
	for i, a := range args {
		switch a.(type) {
		case *jsRStr, *jsRBin:
			out := append([]interface{}{}, args[:i]...)
			for _, b := range args[i:] {
				if bn, isBin := rubyFwd(b).(*jsRBin); isBin {
					out = append(out, rubyBinText(bn.b))
					continue
				}
				out = append(out, rubySV(b))
			}
			return out
		}
	}
	return args
}

// rubyCaseEqv is Ruby's === on a value that is not a user object with its own: a
// Regexp matches, a class tests is_a?, a Range tests membership, everything else
// is ==. Twin of js_rxwhen (regexp half, then js_rwhen).
func (rt *jsrt) rubyCaseEqv(l, r interface{}) bool {
	if rxIsRegexObj(l) {
		return rt.rxMatchOp(l, r) != jsNull
	}
	if lo, ok := l.(*jsObject); ok {
		if _, isCls := lo.props["__isclass"]; isCls {
			return rt.rubyIsA(r, l)
		}
	}
	if lo, isRng := rubyIsRange(l); isRng {
		return rt.rubyOpenRangeCover(lo, r)
	}
	return rt.rubyEq(l, r)
}

// rubyStrMutator: the names that change a String in place - every bang method and
// the mutators that have none. Twin of strMutator in ruby-interpreter.abnf.
func rubyStrMutator(n string) bool {
	if len(n) > 1 && n[len(n)-1] == '!' {
		return true
	}
	switch n {
	case "<<", "concat", "replace", "insert", "prepend", "clear", "[]=", "setbyte":
		return true
	}
	return false
}

// rubyStrCall runs a method on a plain string, the pattern-taking ones included
// (they live in the regexp half).
func (rt *jsrt) rubyStrCall(s, name string, args []interface{}) interface{} {
	if rxWantsPattern(name) {
		if v, ok := rt.rxRegexMethod(s, name, args); ok {
			return v
		}
	}
	return rt.rubyMethod(s, name, args)
}

// rubySubMisses: sub! / gsub! answer nil, and change nothing, when the pattern
// does not match at all. A String pattern matches literally.
func (rt *jsrt) rubySubMisses(s string, args []interface{}) bool {
	pos, _ := rubyBlockArgs(args)
	reObj := argAt(pos, 0)
	if !rxIsRegexObj(reObj) {
		reObj = rxNewRegexObj(rxQuoteText(rt.rubyStr(reObj)), "mpnOU")
	}
	m, ok := rxObjRe(reObj).search([]rune(s), 0)
	if !ok {
		rt.fail("regexp: step limit exceeded")
	}
	return m == nil
}

// rubyStrAppendText is what `s << x' / s.concat(x) appends: an Integer is a
// codepoint.
func (rt *jsrt) rubyStrAppendText(r interface{}) string {
	if f, ok := r.(float64); ok {
		if f < 0 || f > 1114111 {
			return ""
		}
		return string(rune(int(f)))
	}
	return rt.rubyStr(r)
}

// rubyStrMutate runs a mutator on a box. A bang method answers the box, or nil when
// nothing changed - which the plain string's own method already answers for the
// text, so it computes it. Twin of strMutate in ruby-interpreter.abnf.
func (rt *jsrt) rubyStrMutate(b *jsRStr, name string, args []interface{}) interface{} {
	if b.fz {
		rt.rubyRaiseErr(name, "FrozenError", "can't modify frozen String")
	}
	if b.bin != nil {
		return rubyFwdSelf(b, rt.rubyBinMethod(b.bin, name, args))
	}
	switch name {
	case "<<", "concat":
		for i, a := range args {
			a = rubyFwd(a)
			if _, isBin := a.(*jsRBin); !isBin {
				b.s += rt.rubyStrAppendText(a)
				continue
			}
			cv := rt.rubyBinCat(b.s, a)
			if nb, isBin := cv.(*jsRBin); isBin {
				b.bin = nb
				if i+1 < len(args) {
					rt.rubyBinMethod(nb, name, args[i+1:])
				}
				return b
			}
			b.s = rt.rubyStr(cv)
		}
		return b
	case "replace":
		b.s = rt.rubyStr(argAt(args, 0))
		return b
	case "clear":
		b.s = ""
		return b
	case "[]=":
		var k2 interface{} = jsUndef
		if len(args) > 2 {
			k2 = args[1]
		}
		b.s = rt.rubyStr(rt.rubySetSlice(b.s, argAt(args, 0), k2, args[len(args)-1]))
		return args[len(args)-1]
	case "slice!":
		cut := rt.rubyStrCall(b.s, "slice", args)
		if isUndefOrNull(cut) {
			return jsNull
		}
		var k2 interface{} = jsUndef
		if len(args) > 1 {
			k2 = args[1]
		}
		b.s = rt.rubyStr(rt.rubySetSlice(b.s, argAt(args, 0), k2, ""))
		return cut
	case "sub!", "gsub!":
		if rt.rubySubMisses(b.s, args) {
			return jsNull
		}
	}
	r := rt.rubyStrCall(b.s, name, args)
	if isUndefOrNull(r) {
		return jsNull
	}
	b.s = rt.rubyStr(r)
	return b
}

// rubyRStrMethod answers the methods a box answers itself - the mutators and the
// ones that depend on WHICH object it is - or done=false, and rubyMethod then
// answers the name on the text. Twin of the isRStr arm of mcall in
// ruby-interpreter.abnf.
func (rt *jsrt) rubyRStrMethod(b *jsRStr, name string, args []interface{}) (interface{}, bool) {
	if rubyStrMutator(name) {
		return rt.rubyStrMutate(b, name, args), true
	}
	if name == "force_encoding" {
		if b.fz {
			rt.rubyRaiseErr(name, "FrozenError", "can't modify frozen String")
		}
		pos, _ := rubyBlockArgs(args)
		bs, _ := rt.rubyStrBE(b)
		switch nv := rubyEncStr(bs, rt.rubyEncArg(argAt(pos, 0))).(type) {
		case *jsRBin:
			b.bin = nv
		case *jsRStr:
			b.bin, b.s = nil, nv.s
		}
		return b, true
	}
	if b.bin != nil {
		switch name {
		case "freeze", "frozen?", "equal?", "object_id", "itself", "tap":
		default:
			return rubyFwdSelf(b, rt.rubyBinMethod(b.bin, name, args)), true
		}
	}
	switch name {
	case "freeze":
		// The binary String this box became answers clone, +@ and -@.
		b.fz = true
		if b.bin != nil {
			b.bin.fz = true
		}
		return b, true
	case "frozen?":
		return b.fz, true
	case "dup", "+@":
		if name == "+@" && !b.fz {
			return b, true
		}
		return &jsRStr{s: b.s}, true
	case "clone":
		return &jsRStr{s: b.s, fz: b.fz}, true
	case "-@":
		if b.fz {
			return b, true
		}
		return &jsRStr{s: b.s, fz: true}, true
	case "to_s", "to_str", "itself":
		return b, true
	case "tap":
		_, blk := rubyBlockArgs(args)
		rt.call(blk, jsUndef, []interface{}{b})
		return b, true
	case "equal?":
		return rubyIdenticalP(b, argAt(args, 0)), true
	case "object_id":
		return rt.rubyObjectId(b), true
	}
	return nil, false
}

// rubyInclude mixes the program's module m into the class c at run time. An include
// is spliced by copying: each method m holds (plain-named, with its $q$ / $pv$ / $aw$
// visibility slots) is copied onto c unless c defines it itself (its __imn), c is
// recorded among m's includers for a method m gains later (rubyIncludersPut), and
// the ancestry goes in front of c's __mixorder with the $mix$ record is_a? reads.
// An already included module is left alone, as MRI does. Twins: js_rinclude in
// lib/ruby-rt.metajs, includeInto in ruby-interpreter.abnf.
func rubyInclude(c *jsObject, mv interface{}) {
	m, isObj := mv.(*jsObject)
	if !isObj {
		return
	}
	nm, _ := m.props["__name"].(string)
	if c.props["$mix$"+nm] == interface{}(m) {
		return
	}
	own := map[string]bool{}
	if ns, has := c.props["__imn"].(*jsArray); has {
		for _, e := range ns.elems {
			if s, isStr := e.(string); isStr {
				own[s] = true
			}
		}
	}
	for k := range m.props {
		base := k
		for _, pre := range []string{"$q$", "$pv$", "$aw$"} {
			if strings.HasPrefix(base, pre) {
				base = base[len(pre):]
				break
			}
		}
		if base == "" || strings.HasPrefix(base, "$") || strings.HasPrefix(base, "__") || own[base] {
			continue
		}
		c.set(k, m.props[k])
	}
	if incl, has := m.props["__incl"].(*jsArray); has {
		incl.elems = append(incl.elems, c)
	}
	order := &jsArray{elems: []interface{}{m}}
	c.set("$mix$"+nm, m)
	if mo, has := m.props["__mixorder"].(*jsArray); has {
		for _, e := range mo.elems {
			if eo, isO := e.(*jsObject); isO {
				en, _ := eo.props["__name"].(string)
				c.set("$mix$"+en, eo)
				order.elems = append(order.elems, eo)
			}
		}
	}
	rubyPushAll(order, c.props["__mixorder"])
	c.set("__mixorder", order)
}

// rubyStrPromote: a mutator is about to reach v. A plain string becomes a box, and
// when v was read out of an Array or a Hash (o, k) the box replaces it there. Twin
// of promoteElem in ruby-interpreter.abnf and rbStrPromote in lib/ruby-rt.metajs.
func (rt *jsrt) rubyStrPromote(o, k, v interface{}) interface{} {
	s, isStr := v.(string)
	if !isStr {
		return v
	}
	nb := &jsRStr{s: s}
	if arr, ok := o.(*jsArray); ok {
		i := int(math.Trunc(rubyToF(k)))
		if i < 0 {
			i += len(arr.elems)
		}
		if i >= 0 && i < len(arr.elems) && arr.elems[i] == v {
			arr.elems[i] = nb
		}
	} else if keys, vals, ok := dictParts(o); ok {
		if i := rt.rubyDictFind(keys, k); i >= 0 && vals.elems[i] == v {
			vals.elems[i] = nb
		}
	}
	return nb
}

// rubyPctLit is a %-literal's value: a String is a fresh box, an Array (%w / %W) a
// fresh Array of fresh boxes, anything else (%i, %r, %s) itself.
func rubyPctLit(v interface{}) interface{} {
	switch t := v.(type) {
	case string:
		return &jsRStr{s: t}
	case *jsArray:
		out := &jsArray{}
		for _, e := range t.elems {
			if s, ok := e.(string); ok {
				out.elems = append(out.elems, &jsRStr{s: s})
			} else {
				out.elems = append(out.elems, e)
			}
		}
		return out
	}
	return v
}

// rubyFzLit is a literal's value under `# frozen_string_literal: true`: a frozen
// String - the same object for the same text when the literal does not
// interpolate - and for %w an Array of them. Twin of fzLit in
// ruby-interpreter.abnf and js_rsfz in lib/ruby-rt.metajs.
func (rt *jsrt) rubyFzLit(v interface{}, shared bool) interface{} {
	switch t := v.(type) {
	case *jsArray:
		out := &jsArray{}
		for _, e := range t.elems {
			out.elems = append(out.elems, rt.rubyFzLit(e, shared))
		}
		return out
	case string:
		if !shared {
			return &jsRStr{s: t, fz: true}
		}
		if b, ok := rt.rubyFzLits[t]; ok {
			return b
		}
		if rt.rubyFzLits == nil {
			rt.rubyFzLits = map[string]*jsRStr{}
		}
		b := &jsRStr{s: t, fz: true}
		rt.rubyFzLits[t] = b
		return b
	}
	return v
}

func init() {
	rxExtraExterns = append(rxExtraExterns, func(rt *jsrt, m map[string]func(args []uint64) uint64) {
		u := rt.unwrap
		w := rt.wrap
		m["js_rsnew"] = func(a []uint64) uint64 {
			if bn, isBox := rubyFwd(u(a[0])).(*jsRBin); isBox {
				return w(bn)
			}
			return w(&jsRStr{s: rt.rubyStr(u(a[0]))})
		}
		// One piece of an interpolated String joining the text so far: plain text
		// while neither side is a box, else rubyBinCat's encoding rule. Twins:
		// makeTemplate in ruby-interpreter.abnf, js_rtpl in lib/ruby-rt.metajs.
		m["js_rtpl"] = func(a []uint64) uint64 {
			acc, x := rubyFwd(u(a[0])), rubyFwd(u(a[1]))
			_, accBox := acc.(*jsRBin)
			_, xBox := x.(*jsRBin)
			if !accBox && !xBox {
				return w(rt.toString(acc) + rt.rubyStr(x))
			}
			return w(rt.rubyBinCat(acc, x))
		}
		// A String literal of invalid UTF-8: its bytes, one char each, in
		// encoding a[1]. Twin of js_renc in lib/ruby-rt.metajs.
		m["js_renc"] = func(a []uint64) uint64 {
			return w(rubyEncStr(rubyBinOfText(rt.toString(u(a[0]))), rt.toString(u(a[1]))))
		}
		m["js_rsfz"] = func(a []uint64) uint64 { return w(rt.rubyFzLit(u(a[0]), u(a[1]) == true)) }

		m["js_rspct"] = func(a []uint64) uint64 { return w(rubyPctLit(u(a[0]))) }
		m["js_rsprom"] = func(a []uint64) uint64 { return w(rt.rubyStrPromote(u(a[0]), u(a[1]), u(a[2]))) }
		// A bare `.to_s': a new String of js_rstr's text, except that a String box
		// answers itself.
		m["js_rtos"] = func(a []uint64) uint64 {
			switch u(a[0]).(type) {
			case *jsRStr, *jsRBin:
				return a[0]
			}
			return w(&jsRStr{s: rt.rubyStr(u(a[0]))})
		}
		// Unary plus: the identity on a number, String#+@ on a String.
		m["js_rpos"] = func(a []uint64) uint64 {
			switch u(a[0]).(type) {
			case string, *jsRStr, *jsRBin:
				return w(rt.rubyMethod(u(a[0]), "+@", nil))
			}
			return a[0]
		}
	})
}
