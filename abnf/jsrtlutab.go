package abnf

// jsrtlutab.go -- Lua's base library (next, tonumber, assert, the raw* family)
// and table library for the compiler grammar (languages/lua-to-llvm-ir.abnf).
// Twin of the same js_lu* functions in languages/lib/lua-rt.metajs and of the
// globalScope library in lua-interpreter.abnf. Each extern takes the call's
// ARGUMENT ARRAY, so an absent argument can be told from a nil one.

// luIsTable is Lua's type(v) == "table".
func (rt *jsrt) luIsTable(v interface{}) bool { return rt.luTypeName(v) == "table" }

// luCheckTab is luaL_checktype(L, n, LUA_TTABLE).
func (rt *jsrt) luCheckTab(fn string, n int, args []interface{}) interface{} {
	v := luArgAt(args, n)
	if !rt.luIsTable(v) {
		rt.luArgTypeErr(fn, n, "table", args)
	}
	return v
}

// luGeti / luSeti are t[i] for an integer index, addressed the way the emitted
// module's own t[i] addresses it (luKey).
func (rt *jsrt) luGeti(t interface{}, i int64) interface{} {
	return rt.getMember(t, luKey(giNorm(i, 64, false)))
}

func (rt *jsrt) luSeti(t interface{}, i int64, v interface{}) {
	rt.setMember(t, luKey(giNorm(i, 64, false)), v)
}

// luBorder is #t of a table: the first n with t[n + 1] nil.
func (rt *jsrt) luBorder(t interface{}) int64 {
	n := int64(0)
	for !luIsNil(rt.luGeti(t, n+1)) {
		n++
	}
	return n
}

// luStr2Int is lbaselib's b_str2int: the whole string, blanks around it allowed,
// as an integer in base `base`, wrapping at 64 bits.
func luStr2Int(s string, base uint64) (int64, bool) {
	i := 0
	isSpace := func(c byte) bool { return c == ' ' || (c >= 9 && c <= 13) }
	isAlnum := func(c byte) bool { return (c >= '0' && c <= '9') || (c|32 >= 'a' && c|32 <= 'z') }
	for i < len(s) && isSpace(s[i]) {
		i++
	}
	neg := false
	if i < len(s) && (s[i] == '-' || s[i] == '+') {
		neg = s[i] == '-'
		i++
	}
	if i >= len(s) || !isAlnum(s[i]) {
		return 0, false
	}
	var n uint64
	for i < len(s) && isAlnum(s[i]) {
		d := uint64(s[i] - '0')
		if s[i] > '9' {
			d = uint64(s[i]|32-'a') + 10
		}
		if d >= base {
			return 0, false
		}
		n = n*base + d
		i++
	}
	for i < len(s) && isSpace(s[i]) {
		i++
	}
	if neg {
		n = -n
	}
	return int64(n), i == len(s)
}

// luSortLess is sort_comp: the comparator's first result, or a < b.
func (rt *jsrt) luSortLess(cmp, a, b interface{}) bool {
	if luIsNil(cmp) {
		return rt.luCmp(0, a, b)
	}
	r := luFirstOf(rt.call(cmp, jsUndef, []interface{}{a, b}))
	return !luIsNil(r) && r != false
}

// luAuxSort is ltablib.c's auxsort with the middle element as the pivot (lua
// randomizes the pivot only for a badly unbalanced partition of more than 100
// elements), so a comparator sees the same calls in the same order.
func (rt *jsrt) luAuxSort(t, cmp interface{}, lo, up int64) {
	for lo < up {
		if rt.luSortLess(cmp, rt.luGeti(t, up), rt.luGeti(t, lo)) {
			rt.luSwap(t, lo, up)
		}
		if up-lo == 1 {
			return
		}
		p := lo + (up-lo)/2
		if rt.luSortLess(cmp, rt.luGeti(t, p), rt.luGeti(t, lo)) {
			rt.luSwap(t, p, lo)
		} else if rt.luSortLess(cmp, rt.luGeti(t, up), rt.luGeti(t, p)) {
			rt.luSwap(t, p, up)
		}
		if up-lo == 2 {
			return
		}
		pv := rt.luGeti(t, p)
		rt.luSwap(t, p, up-1)
		i, j := lo, up-1
		for {
			i++
			for rt.luSortLess(cmp, rt.luGeti(t, i), pv) {
				if i == up-1 {
					rt.luFail("invalid order function for sorting")
				}
				i++
			}
			j--
			for rt.luSortLess(cmp, pv, rt.luGeti(t, j)) {
				if j < i {
					rt.luFail("invalid order function for sorting")
				}
				j--
			}
			if j < i {
				break
			}
			rt.luSwap(t, i, j)
		}
		rt.luSwap(t, up-1, i)
		p = i
		if p-lo < up-p {
			rt.luAuxSort(t, cmp, lo, p-1)
			lo = p + 1
		} else {
			rt.luAuxSort(t, cmp, p+1, up)
			up = p - 1
		}
	}
}

func (rt *jsrt) luSwap(t interface{}, i, j int64) {
	a, b := rt.luGeti(t, i), rt.luGeti(t, j)
	rt.luSeti(t, i, b)
	rt.luSeti(t, j, a)
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
		ival := func(d float64) int64 { return int64(d) }

		// next(t [, k]): the key after k in the table's insertion order, skipping
		// keys whose value is nil, and its value; nil past the last one.
		m["js_lunext"] = func(a []uint64) uint64 {
			args := argsOf(a[0])
			t := rt.luCheckTab("next", 1, args)
			o, _ := t.(*jsObject)
			var keys []string
			if o != nil {
				for _, k := range o.keys {
					if len(k) < 2 || k[0] != '_' || k[1] != '_' {
						keys = append(keys, k)
					}
				}
			}
			i := 0
			if k := luArgAt(args, 2); !luIsNil(k) {
				kk := rt.toString(luKey(k))
				for i < len(keys) && keys[i] != kk {
					i++
				}
				if i == len(keys) {
					// A runtime error raised inside a C function: lua gives it no
					// position, so it travels as the bare string.
					panic(&jsThrown{value: "invalid key to 'next'"})
				}
				i++
			}
			for ; i < len(keys); i++ {
				v := o.props[keys[i]]
				if !luIsNil(v) {
					return w(luMk([]interface{}{rt.luKeyBack(keys[i]), v}))
				}
			}
			return w(jsUndef)
		}
		// tonumber(v [, base]).
		m["js_lutonumber"] = func(a []uint64) uint64 {
			args := argsOf(a[0])
			if luIsNil(luArgAt(args, 2)) {
				if v, ok := luToNum(luArgAt(args, 1)); ok {
					return w(v)
				}
				rt.luCheckAny("tonumber", 1, args)
				return w(jsUndef)
			}
			base := rt.luCheckInt("tonumber", 2, args)
			s, ok := luArgAt(args, 1).(string)
			if !ok {
				rt.luArgTypeErr("tonumber", 1, "string", args)
			}
			if base < 2 || base > 36 {
				rt.luArgErr("tonumber", 2, "base out of range")
			}
			if n, ok := luStr2Int(s, uint64(base)); ok {
				return w(giNorm(n, 64, false))
			}
			return w(jsUndef)
		}
		// assert(v [, msg, ...]): every argument back when v is truthy, else
		// error(msg) - a string gains the caller's position - with
		// "assertion failed!" for a missing msg.
		m["js_luassert"] = func(a []uint64) uint64 {
			args := argsOf(a[0])
			if v := luArgAt(args, 1); !luIsNil(v) && v != false {
				return w(luMk(append([]interface{}{}, args...)))
			}
			rt.luCheckAny("assert", 1, args)
			var msg interface{} = "assertion failed!"
			if len(args) > 1 {
				msg = args[1]
			}
			if s, ok := msg.(string); ok {
				rt.luFail("%s", s)
			}
			panic(&jsThrown{value: msg})
		}
		m["js_lurawequal"] = func(a []uint64) uint64 {
			args := argsOf(a[0])
			rt.luCheckAny("rawequal", 1, args)
			rt.luCheckAny("rawequal", 2, args)
			return w(rt.luEq(args[0], args[1]))
		}
		m["js_lurawget"] = func(a []uint64) uint64 {
			args := argsOf(a[0])
			t := rt.luCheckTab("rawget", 1, args)
			rt.luCheckAny("rawget", 2, args)
			return w(rt.getMember(t, luKey(args[1])))
		}
		m["js_lurawset"] = func(a []uint64) uint64 {
			args := argsOf(a[0])
			t := rt.luCheckTab("rawset", 1, args)
			rt.luCheckAny("rawset", 2, args)
			rt.luCheckAny("rawset", 3, args)
			rt.setMember(t, rt.luWriteKey(args[1]), args[2])
			return w(t)
		}
		m["js_lurawlen"] = func(a []uint64) uint64 {
			args := argsOf(a[0])
			v := luArgAt(args, 1)
			if s, ok := v.(string); ok {
				return w(float64(len(strUnits(s))))
			}
			if !rt.luIsTable(v) {
				rt.luArgTypeErr("rawlen", 1, "table or string", args)
			}
			return w(giNorm(rt.luBorder(v), 64, false))
		}

		// ----- the io library -----
		// io.write (from 0) and file:write (from 1, the file is self): each
		// argument is written as it is checked, a number as tostring renders it.
		// The caller answers the file. Twin of luaWrite in lua-interpreter.abnf.
		m["js_luwrite"] = func(a []uint64) uint64 {
			args := argsOf(a[0])
			from := int(rt.toNumber(u(a[1])))
			qual := "io.write"
			if from == 1 {
				qual = "?"
				if !luIsFile(luArgAt(args, 1)) {
					rt.luArgTypeErr(qual, 1, "FILE*", args)
				}
			}
			for i := from; i < len(args); i++ {
				outWriter.Write(lowBytes(rt.luCheckStr(qual, i+1, args)))
			}
			return w(jsUndef)
		}
		m["js_luiotype"] = func(a []uint64) uint64 {
			args := argsOf(a[0])
			rt.luCheckAny("io.type", 1, args)
			if luIsFile(args[0]) {
				return rt.wrapStr("file")
			}
			return w(jsUndef)
		}

		// ----- the table library -----
		m["js_luconcat"] = func(a []uint64) uint64 {
			args := argsOf(a[0])
			t := rt.luCheckTab("table.concat", 1, args)
			sep := rt.luOptStr("table.concat", 2, args, "")
			i := ival(rt.luOptInt("table.concat", 3, args, 1))
			last := int64(0)
			if luIsNil(luArgAt(args, 4)) {
				last = rt.luBorder(t)
			} else {
				last = ival(rt.luCheckInt("table.concat", 4, args))
			}
			out := ""
			for ; i <= last; i++ {
				v := rt.luGeti(t, i)
				s, ok := v.(string)
				if !ok {
					if !luIsNum(v) {
						rt.luFail("invalid value (%s) at index %d in table for 'concat'", rt.luTypeName(v), i)
					}
					s = luNumStr(v)
				}
				out += s
				if i < last {
					out += sep
				}
			}
			return rt.wrapStr(out)
		}
		m["js_luinsert"] = func(a []uint64) uint64 {
			args := argsOf(a[0])
			t := rt.luCheckTab("table.insert", 1, args)
			e := rt.luBorder(t) + 1
			switch len(args) {
			case 2:
				rt.luSeti(t, e, args[1])
			case 3:
				pos := ival(rt.luCheckInt("table.insert", 2, args))
				if uint64(pos)-1 >= uint64(e) {
					rt.luArgErr("table.insert", 2, "position out of bounds")
				}
				for i := e; i > pos; i-- {
					rt.luSeti(t, i, rt.luGeti(t, i-1))
				}
				rt.luSeti(t, pos, args[2])
			default:
				rt.luFail("wrong number of arguments to 'insert'")
			}
			return w(jsUndef)
		}
		m["js_luremove"] = func(a []uint64) uint64 {
			args := argsOf(a[0])
			t := rt.luCheckTab("table.remove", 1, args)
			size := rt.luBorder(t)
			pos := size
			if !luIsNil(luArgAt(args, 2)) {
				pos = ival(rt.luCheckInt("table.remove", 2, args))
			}
			if pos != size && uint64(pos)-1 > uint64(size) {
				rt.luArgErr("table.remove", 2, "position out of bounds")
			}
			v := rt.luGeti(t, pos)
			for ; pos < size; pos++ {
				rt.luSeti(t, pos, rt.luGeti(t, pos+1))
			}
			rt.luSeti(t, pos, jsUndef)
			return w(v)
		}
		m["js_luunpack"] = func(a []uint64) uint64 {
			args := argsOf(a[0])
			t := rt.luCheckTab("table.unpack", 1, args)
			i := ival(rt.luOptInt("table.unpack", 2, args, 1))
			var e int64
			if luIsNil(luArgAt(args, 3)) {
				e = rt.luBorder(t)
			} else {
				e = ival(rt.luCheckInt("table.unpack", 3, args))
			}
			if i > e {
				return w(luMk(nil))
			}
			// lua's stack limit (LUAI_MAXSTACK) is what refuses a huge range.
			if uint64(e)-uint64(i) >= 1000000 {
				rt.luFail("too many results to unpack")
			}
			out := make([]interface{}, 0, e-i+1)
			for ; i <= e; i++ {
				out = append(out, rt.luGeti(t, i))
			}
			return w(luMk(out))
		}
		m["js_lupack"] = func(a []uint64) uint64 {
			args := argsOf(a[0])
			t := newJSObject()
			for i, v := range args {
				rt.luSeti(t, int64(i+1), v)
			}
			t.set("n", float64(len(args)))
			return w(t)
		}
		m["js_lusort"] = func(a []uint64) uint64 {
			args := argsOf(a[0])
			t := rt.luCheckTab("table.sort", 1, args)
			n := rt.luBorder(t)
			if n > 1 {
				if n >= 2147483647 {
					rt.luArgErr("table.sort", 1, "array too big")
				}
				cmp := luArgAt(args, 2)
				if !luIsNil(cmp) && rt.luTypeName(cmp) != "function" {
					rt.luArgTypeErr("table.sort", 2, "function", args)
				}
				rt.luAuxSort(t, cmp, 1, n)
			}
			return w(jsUndef)
		}
		m["js_lumove"] = func(a []uint64) uint64 {
			args := argsOf(a[0])
			f := ival(rt.luCheckInt("table.move", 2, args))
			e := ival(rt.luCheckInt("table.move", 3, args))
			t := ival(rt.luCheckInt("table.move", 4, args))
			tt := 1
			if !luIsNil(luArgAt(args, 5)) {
				tt = 5
			}
			src := rt.luCheckTab("table.move", 1, args)
			dst := rt.luCheckTab("table.move", tt, args)
			if e >= f {
				n := e - f + 1
				if t > e || t <= f || (tt != 1 && src != dst) {
					for i := int64(0); i < n; i++ {
						rt.luSeti(dst, t+i, rt.luGeti(src, f+i))
					}
				} else {
					for i := n - 1; i >= 0; i-- {
						rt.luSeti(dst, t+i, rt.luGeti(src, f+i))
					}
				}
			}
			return w(dst)
		}
	})
}
