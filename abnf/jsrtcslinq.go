package abnf

// System.Linq.Enumerable's operators: the Go twin's port of
// languages/lib/csharp-linq.js, whose header has the rules (deferred queries,
// the pull iterator, the index overloads, what is not modelled). A query is a
// {__linq: operator, src, ...} *jsObject, as there.

const csqMsgEmpty = "Sequence contains no elements"

func csqIsQuery(v interface{}) (*jsObject, bool) {
	o, ok := v.(*jsObject)
	if !ok {
		return nil, false
	}
	_, ok = o.props["__linq"].(string)
	return o, ok
}

// csqIsUserSeq: an instance of a user class implementing IEnumerable<T>, which
// the Enumerable extension methods read through its own GetEnumerator.
func (rt *jsrt) csqIsUserSeq(v interface{}) bool { return rt.csUserMethod(v, "GetEnumerator") != nil }

func (rt *jsrt) csqIsLazy(v interface{}) bool {
	if _, ok := csqIsQuery(v); ok {
		return true
	}
	if _, ok := csSeqOf(v); ok {
		return true
	}
	_, ok := v.(*jsGenerator)
	return ok || rt.csqIsUserSeq(v)
}

func csqMake(op string, src interface{}) *jsObject {
	q := newJSObject()
	q.set("__linq", op)
	q.set("src", src)
	return q
}

func (rt *jsrt) csqSrcChk(v interface{}, param string) {
	if isUndefOrNull(v) {
		rt.csThrow("ArgumentNullException", "Value cannot be null. (Parameter '"+param+"')")
	}
}

// csqArity is 2 for a lambda that declares two parameters (js_csarity).
func csqArity(f interface{}) int {
	if jsvFnMeta(f).length == 2 {
		return 2
	}
	return 0
}

func (rt *jsrt) csqCall(f, x interface{}, i int) interface{} {
	if csqArity(f) == 2 {
		return rt.csxCall(f, []interface{}{x, float64(i)})
	}
	return rt.csxCall(f, []interface{}{x})
}

func csqTrue(v interface{}) bool { b, _ := v.(bool); return b }

func csqNullish(v interface{}) bool { return v == nil || isUndefOrNull(v) }

// ----- enumeration -----

type csqIter struct {
	k    string
	q    *jsObject
	a    []interface{} // "a": the elements
	g    *jsGenerator  // "g": a user iterator
	e    interface{}   // "e": a user enumerator's next() adapter (csEnumSrc)
	s, t *csqIter      // the source, and a second sequence or an inner one
	i    int
	cur  interface{}
	x    interface{}
	done bool
	seen []interface{}
	b    []interface{}
}

func csqArrIt(a []interface{}) *csqIter { return &csqIter{k: "a", a: a} }

func (rt *jsrt) csqOpen(v interface{}) *csqIter {
	if a, ok := v.(*jsArray); ok {
		return csqArrIt(a.elems)
	}
	if q, ok := csqIsQuery(v); ok {
		return rt.csqOpenQuery(q)
	}
	if g, ok := v.(*jsGenerator); ok {
		return &csqIter{k: "g", g: g}
	}
	if sq, ok := csSeqOf(v); ok {
		return &csqIter{k: "g", g: rt.csSeqOpen(sq)}
	}
	if rt.csqIsUserSeq(v) {
		src := rt.csEnumSrc(rt.csxMcall(v, "GetEnumerator", nil))
		if g, ok := src.(*jsGenerator); ok {
			return &csqIter{k: "g", g: g}
		}
		return &csqIter{k: "e", e: src}
	}
	rt.csqSrcChk(v, "source")
	_, isStr := v.(string)
	if _, _, isDict := dictParts(v); isStr || isDict {
		return csqArrIt(rt.csxForSrc(v).(*jsArray).elems)
	}
	rt.fail("a LINQ source that is not a collection, a string, a dictionary or an iterator is not modelled")
	return nil
}

func (rt *jsrt) csqToArray(v interface{}) []interface{} {
	it := rt.csqOpen(v)
	out := []interface{}{}
	for rt.csqNext(it) {
		out = append(out, it.cur)
	}
	return out
}

func (rt *jsrt) csqOpenQuery(q *jsObject) *csqIter {
	op := q.props["__linq"].(string)
	it := &csqIter{k: op, q: q}
	src := q.props["src"]
	switch op {
	case "range", "repeat":
		return it
	case "grp":
		return csqArrIt(src.(*jsArray).elems)
	case "reverse":
		r := rt.csqToArray(src)
		for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
			r[i], r[j] = r[j], r[i]
		}
		return csqArrIt(r)
	case "order":
		return csqArrIt(rt.csqSorted(q))
	case "group":
		return csqArrIt(rt.csqGroups(q))
	case "takelast", "skiplast":
		all := rt.csqToArray(src)
		n := jsToInt(q.props["n"].(float64))
		if n < 0 {
			n = 0
		}
		if n > len(all) {
			n = len(all)
		}
		if op == "takelast" {
			return csqArrIt(all[len(all)-n:])
		}
		return csqArrIt(all[:len(all)-n])
	}
	it.s = rt.csqOpen(src)
	switch op {
	case "intersect", "except":
		it.b = rt.csqToArray(q.props["b"])
	case "concat", "union", "zip":
		it.t = rt.csqOpen(q.props["b"])
	}
	return it
}

func (rt *jsrt) csqHas(list []interface{}, x, cmp interface{}) bool {
	for _, e := range list {
		if rt.csqEq(e, x, cmp) {
			return true
		}
	}
	return false
}

func (rt *jsrt) csqEq(a, b, cmp interface{}) bool {
	if csqNullish(cmp) {
		return rt.csxEq(a, b)
	}
	if k := csxStrComparerKind(cmp); k >= 0 {
		return csxEqStr(k, a, b)
	}
	rt.fail("an IEqualityComparer other than StringComparer's is not modelled")
	return false
}

func (it *csqIter) prop(name string) interface{} {
	v, ok := it.q.props[name]
	if !ok {
		return jsUndef
	}
	return v
}

func (rt *jsrt) csqNext(it *csqIter) bool {
	switch it.k {
	case "a":
		if it.i >= len(it.a) {
			return false
		}
		it.cur = it.a[it.i]
		it.i++
		return true
	case "g", "e":
		var st *jsObject
		if it.k == "g" {
			st, _ = it.g.step(rt, jsUndef).(*jsObject)
		} else {
			st, _ = rt.call(rt.getMember(it.e, "next"), jsUndef, nil).(*jsObject)
		}
		if st == nil || st.props["done"] == true {
			return false
		}
		it.cur = st.props["value"]
		return true
	}
	s := it.s
	switch it.k {
	case "where":
		for rt.csqNext(s) {
			x := s.cur
			wi := it.i
			it.i++
			if csqTrue(rt.csqCall(it.prop("f"), x, wi)) {
				it.cur = x
				return true
			}
		}
		return false
	case "select":
		if !rt.csqNext(s) {
			return false
		}
		si := it.i
		it.i++
		it.cur = rt.csqCall(it.prop("f"), s.cur, si)
		return true
	case "cast":
		if !rt.csqNext(s) {
			return false
		}
		it.cur = s.cur
		return true
	case "selmany":
		for {
			if it.t != nil {
				if rt.csqNext(it.t) {
					if g := it.prop("g"); csqNullish(g) {
						it.cur = it.t.cur
					} else {
						it.cur = rt.csxCall(g, []interface{}{it.x, it.t.cur})
					}
					return true
				}
				it.t = nil
			}
			if !rt.csqNext(s) {
				return false
			}
			mi := it.i
			it.i++
			it.x = s.cur
			it.t = rt.csqOpen(rt.csqCall(it.prop("f"), s.cur, mi))
		}
	case "take":
		if float64(it.i) >= it.prop("n").(float64) || !rt.csqNext(s) {
			return false
		}
		it.i++
		it.cur = s.cur
		return true
	case "skip":
		for float64(it.i) < it.prop("n").(float64) {
			if !rt.csqNext(s) {
				return false
			}
			it.i++
		}
		if !rt.csqNext(s) {
			return false
		}
		it.cur = s.cur
		return true
	case "takew":
		if it.done || !rt.csqNext(s) {
			return false
		}
		ti := it.i
		it.i++
		if !csqTrue(rt.csqCall(it.prop("f"), s.cur, ti)) {
			it.done = true
			return false
		}
		it.cur = s.cur
		return true
	case "skipw":
		for !it.done {
			if !rt.csqNext(s) {
				return false
			}
			ki := it.i
			it.i++
			if !csqTrue(rt.csqCall(it.prop("f"), s.cur, ki)) {
				it.done = true
				it.cur = s.cur
				return true
			}
		}
		if !rt.csqNext(s) {
			return false
		}
		it.cur = s.cur
		return true
	case "concat", "union":
		for {
			src := s
			if it.done {
				src = it.t
			}
			if !rt.csqNext(src) {
				if it.done {
					return false
				}
				it.done = true
				continue
			}
			x := src.cur
			if it.k == "concat" {
				it.cur = x
				return true
			}
			if !rt.csqHas(it.seen, x, it.prop("c")) {
				it.seen = append(it.seen, x)
				it.cur = x
				return true
			}
		}
	case "distinct", "intersect", "except":
		for rt.csqNext(s) {
			x := s.cur
			key := x
			if f := it.prop("f"); !csqNullish(f) {
				key = rt.csxCall(f, []interface{}{x})
			}
			c := it.prop("c")
			if rt.csqHas(it.seen, key, c) {
				continue
			}
			if it.k == "intersect" && !rt.csqHas(it.b, key, c) {
				continue
			}
			if it.k == "except" && rt.csqHas(it.b, key, c) {
				continue
			}
			it.seen = append(it.seen, key)
			it.cur = x
			return true
		}
		return false
	case "append", "prepend":
		at := it.i
		it.i++
		if it.k == "prepend" && at == 0 {
			it.cur = it.prop("x")
			return true
		}
		if rt.csqNext(s) {
			it.cur = s.cur
			return true
		}
		if it.k == "append" && !it.done {
			it.done = true
			it.cur = it.prop("x")
			return true
		}
		return false
	case "defempty":
		if rt.csqNext(s) {
			it.i = 1
			it.cur = s.cur
			return true
		}
		if it.i == 0 {
			it.i = 1
			if it.q.props["has"] == true {
				it.cur = it.prop("x")
			} else {
				it.cur = csqDefault(it.q.props["src"], false, nil)
			}
			return true
		}
		return false
	case "zip":
		if !rt.csqNext(s) || !rt.csqNext(it.t) {
			return false
		}
		if zf := it.prop("f"); csqNullish(zf) {
			tup := newJSObject()
			tup.set("__tuple", float64(2))
			tup.set("Item1", s.cur)
			tup.set("Item2", it.t.cur)
			it.cur = tup
		} else {
			it.cur = rt.csxCall(zf, []interface{}{s.cur, it.t.cur})
		}
		return true
	case "range":
		if float64(it.i) >= it.prop("n").(float64) {
			return false
		}
		it.cur = it.prop("x").(float64) + float64(it.i)
		it.i++
		return true
	case "repeat":
		if float64(it.i) >= it.prop("n").(float64) {
			return false
		}
		it.cur = it.prop("x")
		it.i++
		return true
	}
	rt.fail("unknown query operator: " + it.k)
	return false
}

// ----- the buffering operators -----

type csqKey struct {
	f    interface{}
	desc bool
	c    interface{}
}

func csqKeysOf(q *jsObject) []csqKey {
	var out []csqKey
	for _, e := range q.props["keys"].(*jsArray).elems {
		o := e.(*jsObject)
		out = append(out, csqKey{f: o.props["f"], desc: o.props["desc"] == true, c: o.props["c"]})
	}
	return out
}

func (rt *jsrt) csqSorted(q *jsObject) []interface{} {
	elems := rt.csqToArray(q.props["src"])
	specs := csqKeysOf(q)
	keys := make([][]interface{}, len(elems))
	for i, x := range elems {
		for _, sp := range specs {
			k := x
			if !csqNullish(sp.f) {
				k = rt.csxCall(sp.f, []interface{}{x})
			}
			keys[i] = append(keys[i], k)
		}
	}
	idx := make([]int, len(elems))
	for i := range idx {
		idx[i] = i
	}
	cmp := func(a, b []interface{}) int {
		for j, sp := range specs {
			if c := rt.csxCmpWith(sp.c, a[j], b[j]); c != 0 {
				if sp.desc {
					return -c
				}
				return c
			}
		}
		return 0
	}
	n := len(idx)
	for width := 1; width < n; width *= 2 {
		dst := make([]int, 0, n)
		for lo := 0; lo < n; {
			mid, hi := lo+width, lo+2*width
			if mid > n {
				mid = n
			}
			if hi > n {
				hi = n
			}
			x, y := lo, mid
			for x < mid && y < hi {
				if cmp(keys[idx[y]], keys[idx[x]]) < 0 {
					dst = append(dst, idx[y])
					y++
				} else {
					dst = append(dst, idx[x])
					x++
				}
			}
			dst = append(append(dst, idx[x:mid]...), idx[y:hi]...)
			lo = hi
		}
		idx = dst
	}
	out := make([]interface{}, n)
	for i, k := range idx {
		out[i] = elems[k]
	}
	return out
}

func (rt *jsrt) csqGroups(q *jsObject) []interface{} {
	var gkeys []interface{}
	var gvals [][]interface{}
	f, g, r := q.props["f"], q.props["g"], q.props["r"]
	for _, x := range rt.csqToArray(q.props["src"]) {
		key := rt.csxCall(f, []interface{}{x})
		v := x
		if !csqNullish(g) {
			v = rt.csxCall(g, []interface{}{x})
		}
		j := 0
		for j < len(gkeys) && !rt.csxEq(gkeys[j], key) {
			j++
		}
		if j == len(gkeys) {
			gkeys = append(gkeys, key)
			gvals = append(gvals, nil)
		}
		gvals[j] = append(gvals[j], v)
	}
	out := make([]interface{}, len(gkeys))
	for i, k := range gkeys {
		grp := csqMake("grp", &jsArray{elems: append([]interface{}{}, gvals[i]...)})
		grp.set("Key", k)
		if csqNullish(r) {
			out[i] = grp
		} else {
			out[i] = rt.csxCall(r, []interface{}{k, grp})
		}
	}
	return out
}

func csqOrder(src, f interface{}, desc bool, cmp interface{}, prev *jsArray) *jsObject {
	q := csqMake("order", src)
	keys := &jsArray{}
	if prev != nil {
		keys.elems = append(keys.elems, prev.elems...)
	}
	spec := newJSObject()
	spec.set("f", f)
	spec.set("desc", desc)
	spec.set("c", cmp)
	keys.elems = append(keys.elems, spec)
	q.set("keys", keys)
	return q
}

// ----- the aggregates -----

func csqNumKind(v interface{}) string {
	switch t := v.(type) {
	case float64:
		return "i"
	case jsGInt:
		return "l"
	case jsJFlo:
		if t.sty == floCSF {
			return "f"
		}
		return "d"
	}
	return ""
}

func (rt *jsrt) csqSum(it *csqIter) interface{} {
	kind := ""
	acc := 0.0
	var lacc interface{} = float64(0)
	for rt.csqNext(it) {
		x := it.cur
		if csqNullish(x) {
			continue
		}
		nk := csqNumKind(x)
		if kind == "" {
			kind = nk
		}
		if nk == "l" {
			lacc = rt.csArith("+", lacc, x)
		} else {
			acc += rt.toNumber(x)
			if nk == "i" && (acc > 2147483647 || acc < -2147483648) {
				rt.csThrow("OverflowException", "Arithmetic operation resulted in an overflow.")
			}
		}
	}
	switch kind {
	case "l":
		return lacc
	case "d":
		return jsJFlo{f: acc, sty: floCS}
	case "f":
		return jsJFlo{f: float64(float32(acc)), sty: floCSF}
	}
	return acc
}

func (rt *jsrt) csqAverage(it *csqIter) interface{} {
	kind := ""
	acc, n := 0.0, 0
	for rt.csqNext(it) {
		if csqNullish(it.cur) {
			continue
		}
		if kind == "" {
			kind = csqNumKind(it.cur)
		}
		acc += rt.toNumber(it.cur)
		n++
	}
	if n == 0 {
		rt.csThrow("InvalidOperationException", csqMsgEmpty)
	}
	if kind == "f" {
		return jsJFlo{f: float64(float32(acc / float64(n))), sty: floCSF}
	}
	return jsJFlo{f: acc / float64(n), sty: floCS}
}

// csqElemTy / csqDefault: lib/csharp-linq.js's.
func csqElemTy(v interface{}) string {
	switch t := v.(type) {
	case *jsArray:
		return csxElemTy(t)
	case string:
		return "char"
	}
	q, ok := csqIsQuery(v)
	if !ok {
		return ""
	}
	switch q.props["__linq"] {
	case "range":
		return "int"
	case "select", "selmany", "zip", "group", "grp", "repeat":
		return ""
	}
	return csqElemTy(q.props["src"])
}

func csqDefault(src interface{}, seen bool, any interface{}) interface{} {
	return csxDefaultFor(csqElemTy(src), seen, any)
}

// csqMinMax: src is the source, nil when the elements were selected.
func (rt *jsrt) csqMinMax(it *csqIter, sign int, src interface{}) interface{} {
	has := false
	var best interface{}
	for rt.csqNext(it) {
		if !has || rt.csxCmp(it.cur, best)*sign > 0 {
			best = it.cur
		}
		has = true
	}
	if !has {
		if src != nil {
			if ety := csqElemTy(src); ety != "" && csqNullish(csxDefaultFor(ety, false, nil)) {
				return jsNull
			}
		}
		rt.csThrow("InvalidOperationException", csqMsgEmpty)
	}
	return best
}

func (rt *jsrt) csqMinMaxBy(it *csqIter, f interface{}, sign int) interface{} {
	has := false
	var best, bk interface{}
	for rt.csqNext(it) {
		key := rt.csxCall(f, []interface{}{it.cur})
		if !has || rt.csxCmp(key, bk)*sign > 0 {
			best, bk = it.cur, key
		}
		has = true
	}
	if !has {
		rt.fail("MinBy / MaxBy of an empty sequence needs the element type, which is not modelled")
	}
	return best
}

// csqPick: First (which 0) / Last (1) / Single (2); dflt nil throws.
func (rt *jsrt) csqPick(it *csqIter, pred interface{}, which int, orDefault bool, dflt, src interface{}) interface{} {
	has, seen := false, false
	var got, any interface{}
	for rt.csqNext(it) {
		x := it.cur
		if !seen {
			any, seen = x, true
		}
		if !csqNullish(pred) && !csqTrue(rt.csxCall(pred, []interface{}{x})) {
			continue
		}
		if has && which == 2 {
			msg := "Sequence contains more than one element"
			if !csqNullish(pred) {
				msg = "Sequence contains more than one matching element"
			}
			rt.csThrow("InvalidOperationException", msg)
		}
		got, has = x, true
		if which == 0 {
			return got
		}
	}
	if has {
		return got
	}
	if orDefault {
		if dflt != nil {
			return dflt
		}
		return csqDefault(src, seen, any)
	}
	if csqNullish(pred) {
		rt.csThrow("InvalidOperationException", csqMsgEmpty)
	}
	rt.csThrow("InvalidOperationException", "Sequence contains no matching element")
	return nil
}

func (rt *jsrt) csqAggregate(it *csqIter, args []interface{}) interface{} {
	var acc interface{}
	f := args[0]
	if len(args) > 1 {
		acc, f = args[0], args[1]
	} else {
		if !rt.csqNext(it) {
			rt.csThrow("InvalidOperationException", csqMsgEmpty)
		}
		acc = it.cur
	}
	for rt.csqNext(it) {
		acc = rt.csxCall(f, []interface{}{acc, it.cur})
	}
	if len(args) > 2 {
		return rt.csxCall(args[2], []interface{}{acc})
	}
	return acc
}

func (rt *jsrt) csqToDict(it *csqIter, args []interface{}) interface{} {
	d := newJSObject()
	d.set("__dict", true)
	d.set("keys", &jsArray{})
	d.set("vals", &jsArray{})
	var vf interface{}
	if len(args) > 1 && cspIsDeleg(args[1]) {
		vf = args[1]
	}
	if last := args[len(args)-1]; len(args) > 1 && !csqNullish(last) && !cspIsDeleg(last) {
		rt.fail("a Dictionary key comparer is not modelled")
	}
	for rt.csqNext(it) {
		x := it.cur
		v := x
		if vf != nil {
			v = rt.csxCall(vf, []interface{}{x})
		}
		rt.csxDictMethod(d, "Add", []interface{}{rt.csxCall(args[0], []interface{}{x}), v})
	}
	return d
}

// ----- the operator dispatch -----

var csqLazyOps = map[string]string{
	"Where": "where", "Select": "select", "TakeWhile": "takew", "SkipWhile": "skipw",
	"Take": "take", "Skip": "skip", "TakeLast": "takelast", "SkipLast": "skiplast",
	"Concat": "concat", "Union": "union", "Intersect": "intersect", "Except": "except", "Zip": "zip",
	"Append": "append", "Prepend": "prepend", "DefaultIfEmpty": "defempty",
}

func (rt *jsrt) csqMethod(recv interface{}, name string, args []interface{}) interface{} {
	n := len(args)
	arg := func(i int) interface{} {
		if i < n {
			return args[i]
		}
		return jsNull
	}
	if q, ok := csqIsQuery(recv); ok && q.props["__linq"] == "order" && (name == "ThenBy" || name == "ThenByDescending") {
		return csqOrder(q.props["src"], arg(0), name == "ThenByDescending", arg(1), q.props["keys"].(*jsArray))
	}
	if op, ok := csqLazyOps[name]; ok {
		q := csqMake(op, recv)
		switch name {
		case "Where", "Select", "TakeWhile", "SkipWhile":
			param := "predicate"
			if name == "Select" {
				param = "selector"
			}
			rt.csqSrcChk(arg(0), param)
			q.set("f", arg(0))
		case "Take", "Skip", "TakeLast", "SkipLast":
			q.set("n", float64(rt.csxNum(arg(0))))
		case "Append", "Prepend", "DefaultIfEmpty":
			q.set("x", arg(0))
			q.set("has", n > 0)
		default:
			rt.csqSrcChk(arg(0), "second")
			q.set("b", arg(0))
			q.set("c", jsNull)
			q.set("f", jsNull)
			if n > 1 {
				if name == "Zip" {
					q.set("f", args[1])
				} else {
					q.set("c", args[1])
				}
			}
		}
		return q
	}
	switch name {
	case "SelectMany":
		q := csqMake("selmany", recv)
		q.set("f", arg(0))
		q.set("g", arg(1))
		return q
	case "Distinct", "DistinctBy":
		q := csqMake("distinct", recv)
		if name == "DistinctBy" {
			q.set("f", arg(0))
			q.set("c", arg(1))
		} else {
			q.set("f", jsNull)
			q.set("c", arg(0))
		}
		return q
	case "Reverse":
		return csqMake("reverse", recv)
	case "Cast", "AsEnumerable":
		return csqMake("cast", recv)
	case "OrderBy", "OrderByDescending":
		return csqOrder(recv, arg(0), name == "OrderByDescending", arg(1), nil)
	case "Order", "OrderDescending":
		return csqOrder(recv, jsNull, name == "OrderDescending", arg(0), nil)
	case "GroupBy":
		q := csqMake("group", recv)
		q.set("f", arg(0))
		q.set("g", jsNull)
		q.set("r", jsNull)
		if n == 2 && csqArity(args[1]) == 2 {
			q.set("r", args[1])
		} else if n > 1 {
			q.set("g", args[1])
		}
		if n > 2 {
			q.set("r", args[2])
		}
		return q
	}
	it := rt.csqOpen(recv)
	switch name {
	case "ToList", "ToArray":
		out := &jsArray{}
		for rt.csqNext(it) {
			out.elems = append(out.elems, it.cur)
		}
		return csxTyped(out, csqElemTy(recv), name == "ToList")
	case "ToDictionary":
		return rt.csqToDict(it, args)
	case "Count", "LongCount":
		c := 0
		for rt.csqNext(it) {
			if n == 0 || csqTrue(rt.csxCall(args[0], []interface{}{it.cur})) {
				c++
			}
		}
		return float64(c)
	case "Any":
		for rt.csqNext(it) {
			if n == 0 || csqTrue(rt.csxCall(args[0], []interface{}{it.cur})) {
				return true
			}
		}
		return false
	case "All":
		for rt.csqNext(it) {
			if !csqTrue(rt.csxCall(args[0], []interface{}{it.cur})) {
				return false
			}
		}
		return true
	case "Contains":
		for rt.csqNext(it) {
			if rt.csqEq(it.cur, args[0], arg(1)) {
				return true
			}
		}
		return false
	case "Sum", "Average", "Min", "Max":
		var src interface{} = recv
		if n > 0 {
			sq := csqMake("select", recv)
			sq.set("f", args[0])
			it = rt.csqOpen(sq)
			src = nil
		}
		switch name {
		case "Sum":
			return rt.csqSum(it)
		case "Average":
			return rt.csqAverage(it)
		case "Max":
			return rt.csqMinMax(it, 1, src)
		}
		return rt.csqMinMax(it, -1, src)
	case "MinBy":
		return rt.csqMinMaxBy(it, arg(0), -1)
	case "MaxBy":
		return rt.csqMinMaxBy(it, arg(0), 1)
	case "First", "FirstOrDefault", "Last", "LastOrDefault", "Single", "SingleOrDefault":
		which := map[byte]int{'F': 0, 'L': 1, 'S': 2}[name[0]]
		orDef := len(name) > 9 && name[len(name)-9:] == "OrDefault"
		var pred, dflt interface{} = jsNull, nil
		if n > 0 && cspIsDeleg(args[0]) {
			pred = args[0]
		} else if n > 0 {
			dflt = args[0]
		}
		if n > 1 {
			dflt = args[1]
		}
		return rt.csqPick(it, pred, which, orDef, dflt, recv)
	case "ElementAt", "ElementAtOrDefault":
		at := rt.csxNum(args[0])
		var any interface{} = jsNull
		ei := 0
		for ; at >= 0 && rt.csqNext(it); ei++ {
			if ei == at {
				return it.cur
			}
			any = it.cur
		}
		if name == "ElementAt" {
			rt.csxArgRange(csxMsgIdxLess, "index")
		}
		return csqDefault(recv, ei > 0, any)
	case "Aggregate":
		return rt.csqAggregate(it, args)
	case "SequenceEqual":
		ot := rt.csqOpen(args[0])
		for {
			more := rt.csqNext(it)
			if more != rt.csqNext(ot) {
				return false
			}
			if !more {
				return true
			}
			if !rt.csqEq(it.cur, ot.cur, arg(1)) {
				return false
			}
		}
	case "GetEnumerator":
		rt.fail("GetEnumerator of a LINQ query is not modelled")
	}
	rt.fail("unknown method: " + name)
	return nil
}

func (rt *jsrt) csqStatic(name string, args []interface{}) interface{} {
	switch name {
	case "Range":
		start, count := rt.csxNum(argAt(args, 0)), rt.csxNum(argAt(args, 1))
		if count < 0 || int64(start)+int64(count)-1 > 2147483647 {
			rt.csxArgRange("Specified argument was out of the range of valid values.", "count")
		}
		q := csqMake("range", jsNull)
		q.set("x", float64(start))
		q.set("n", float64(count))
		return q
	case "Repeat":
		rc := rt.csxNum(argAt(args, 1))
		if rc < 0 {
			rt.csxArgRange("Specified argument was out of the range of valid values.", "count")
		}
		q := csqMake("repeat", jsNull)
		q.set("x", argAt(args, 0))
		q.set("n", float64(rc))
		return q
	case "Empty":
		return csqMake("cast", &jsArray{})
	}
	if len(args) == 0 {
		rt.fail("unknown method: " + name)
	}
	return rt.csqMethod(args[0], name, args[1:])
}
