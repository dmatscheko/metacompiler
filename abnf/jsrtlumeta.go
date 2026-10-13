package abnf

// jsrtlumeta.go -- Lua's metatables for the compiler grammar
// (languages/lua-to-llvm-ir.abnf). Twin of the metatable functions in
// languages/lib/lua-rt.metajs and of luaTm and its callers in
// lua-interpreter.abnf.
//
// A table's metatable is the hidden field __f of its object (setmetatable),
// which no program can name because luStrKey escapes the "__" prefix. It is the
// float box's slot on purpose: every table write and every read miss of the
// emitted module already reads __f to refuse a box, so the same read finds a
// metatable. A box holds a float64 there, a table a table. A
// metamethod is the RAW field of the metatable named by the event, so it is read
// at the event's escaped key. Every string shares one metatable, whose __index is
// the string library; the emitted module hands it over with js_lustrmeta.
//
// The emitted module asks for a metamethod only where the plain operation
// cannot answer - a raw miss, a store into a table that has a metatable, an
// operand that is no number - so a program without metatables pays nothing.

import "strings"

// luStrMeta is the strings' metatable, once the module has built it. One compile
// runs one program, so a package-level variable is per program.
var luStrMeta *jsObject

// luEvents names the events by the code the emitter passes to js_lutm. The
// arithmetic codes are js_luarith's operator indexes, and 8 + k is js_lubit's
// operator k.
var luEvents = []string{"__add", "__sub", "__mul", "__div", "__idiv", "__mod", "__pow", "__unm",
	"__band", "__bor", "__bxor", "__shl", "__shr", "__bnot", "__concat", "__eq", "__lt", "__le", "__len",
	"__call", "__pairs"}

// luMetaOf is v's metatable, or nil: a table's or a file's __f when it holds a
// table (a float box holds a float64 there).
func (rt *jsrt) luMetaOf(v interface{}) *jsObject {
	if _, isStr := v.(string); isStr {
		return luStrMeta
	}
	o, ok := v.(*jsObject)
	if !ok {
		return nil
	}
	mt, _ := o.props["__f"].(*jsObject)
	return mt
}

// luTm is v's metamethod for event ev, or nil.
func (rt *jsrt) luTm(v interface{}, ev string) interface{} {
	mt := rt.luMetaOf(v)
	if mt == nil {
		return nil
	}
	h := mt.props[luKeyEsc+ev]
	if luIsNil(h) {
		return nil
	}
	return h
}

// luBinTm is a binary event's metamethod: the first operand's, else the second's
// (luaT_trybinTM).
func (rt *jsrt) luBinTm(ev string, a, b interface{}) interface{} {
	if h := rt.luTm(a, ev); h != nil {
		return h
	}
	return rt.luTm(b, ev)
}

// luTmCall is the first result of calling metamethod h.
func (rt *jsrt) luTmCall(h interface{}, args ...interface{}) interface{} {
	return luFirstOf(rt.call(h, jsUndef, args))
}

func luTruthy(v interface{}) bool { return !luIsNil(v) && v != false }

// luMaxTagLoop is lvm.c's MAXTAGLOOP, the longest __index / __newindex chain.
const luMaxTagLoop = 2000

// luIndexMeta is luaV_finishget past a raw miss on t: a function answers the
// lookup, a table is searched raw and then through its own metatable, and
// anything else is indexed through its type's metatable.
func (rt *jsrt) luIndexMeta(t, k interface{}) interface{} {
	kk := luKey(k)
	for loop := 0; loop < luMaxTagLoop; loop++ {
		h := rt.luTm(t, "__index")
		if h == nil {
			if !rt.luIsTable(t) {
				rt.luFail("attempt to index a %s value", rt.luTypeName(t))
			}
			return jsUndef
		}
		if _, isFn := h.(*jsClosure); isFn {
			return rt.luTmCall(h, t, k)
		}
		t = h
		if rt.luIsTable(t) {
			if v := rt.getMember(t, kk); !luIsNil(v) {
				return v
			}
		}
	}
	rt.luFail("'__index' chain too long; possible loop")
	return nil
}

// luSetIndexMeta is luaV_finishset on t, which has a metatable: a key the table
// already holds, or a table with no __newindex, is stored raw; a function takes
// the store, and a table is stored into in its turn.
func (rt *jsrt) luSetIndexMeta(t, k, v interface{}) {
	for loop := 0; loop < luMaxTagLoop; loop++ {
		var h interface{}
		if rt.luIsTable(t) {
			if luIsNil(rt.getMember(t, luKey(k))) {
				h = rt.luTm(t, "__newindex")
			}
			if h == nil {
				rt.setMember(t, rt.luWriteKey(k), v)
				return
			}
		} else if h = rt.luTm(t, "__newindex"); h == nil {
			rt.luFail("attempt to index a %s value", rt.luTypeName(t))
		}
		if _, isFn := h.(*jsClosure); isFn {
			rt.call(h, jsUndef, []interface{}{t, k, v})
			return
		}
		t = h
	}
	rt.luFail("'__newindex' chain too long; possible loop")
}

// luObjStr is luaL_tolstring of a table or a function: its __tostring's answer,
// which has to be a string or a number; else the identity, under the
// metatable's __name when that is a string.
func (rt *jsrt) luObjStr(v interface{}) string {
	if h := rt.luTm(v, "__tostring"); h != nil {
		s := rt.luTmCall(h, v)
		if str, ok := s.(string); ok {
			return str
		}
		if luIsNum(s) {
			return luNumStr(s)
		}
		rt.luFail("'__tostring' must return a string")
	}
	id := rt.luIdent(v)
	if nm, ok := rt.luTm(v, "__name").(string); ok && !luIsFile(v) {
		return nm + id[strings.Index(id, ":"):]
	}
	return id
}

func init() {
	rxExtraExterns = append(rxExtraExterns, func(rt *jsrt, m map[string]func(args []uint64) uint64) {
		u := rt.unwrap
		w := rt.wrap
		argsOf := func(h uint64) []interface{} {
			if arr, ok := u(h).(*jsArray); ok {
				return arr.elems
			}
			return nil
		}
		// (event code as a raw handle, a, b) -> a's metamethod, else b's, or nil.
		m["js_lutm"] = func(a []uint64) uint64 {
			if h := rt.luBinTm(luEvents[int(a[0])], u(a[1]), u(a[2])); h != nil {
				return w(h)
			}
			return w(jsUndef)
		}
		m["js_luindex"] = func(a []uint64) uint64 { return w(rt.luIndexMeta(u(a[0]), u(a[1]))) }
		m["js_lusetindex"] = func(a []uint64) uint64 {
			rt.luSetIndexMeta(u(a[0]), u(a[1]), u(a[2]))
			return w(jsUndef)
		}
		m["js_lustrmeta"] = func(a []uint64) uint64 {
			luStrMeta, _ = u(a[0]).(*jsObject)
			return w(jsUndef)
		}
		// setmetatable(t, mt) and getmetatable(v), lbaselib's: a __metatable
		// field protects the metatable and is what getmetatable answers instead.
		m["js_lusetmt"] = func(a []uint64) uint64 {
			args := argsOf(a[0])
			t := rt.luCheckTab("setmetatable", 1, args)
			mt := luArgAt(args, 2)
			if len(args) < 2 || (!luIsNil(mt) && !rt.luIsTable(mt)) {
				rt.luArgTypeErr("setmetatable", 2, "nil or table", args)
			}
			if rt.luTm(t, "__metatable") != nil {
				rt.luFail("cannot change a protected metatable")
			}
			if luIsNil(mt) {
				mt = jsUndef
			}
			rt.setMember(t, "__f", mt)
			return w(t)
		}
		m["js_lugetmt"] = func(a []uint64) uint64 {
			args := argsOf(a[0])
			rt.luCheckAny("getmetatable", 1, args)
			mt := rt.luMetaOf(args[0])
			if mt == nil {
				return w(jsUndef)
			}
			if prot := rt.luTm(args[0], "__metatable"); prot != nil {
				return w(prot)
			}
			return w(mt)
		}
	})
}
