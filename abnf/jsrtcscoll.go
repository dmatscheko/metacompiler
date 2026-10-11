package abnf

import (
	"math"
	"sync"
)

// List<T>'s and Dictionary<TKey,TValue>'s members: the Go twin's port of
// languages/lib/csharp-collections.js, whose header has the rules (default(T),
// the comparers, the stable sort, the exception classes). A List<T> is a
// *jsArray and a Dictionary a {__dict, keys, vals} *jsObject, as there.

const (
	csxMsgIdxLess   = "Index was out of range. Must be non-negative and less than the size of the collection."
	csxMsgIdxLessEq = "Index was out of range. Must be non-negative and less than or equal to the size of the collection."
	csxMsgCount     = "Count must be positive and count must refer to a location within the string/array/collection."
	csxMsgNonNeg    = "Non-negative number required."
	csxMsgOffLen    = "Offset and length were out of bounds for the array or count is greater than the number of elements from index to the end of the source collection."
)

func (rt *jsrt) csxArgRange(msg, param string) {
	rt.csThrow("ArgumentOutOfRangeException", msg+" (Parameter '"+param+"')")
}

// ----- the value-model hooks -----

func csxIsNum(v interface{}) bool {
	switch v.(type) {
	case float64, jsGInt, jsJFlo, jsChar:
		return true
	}
	return false
}

func (rt *jsrt) csxLt(a, b interface{}) bool {
	_, af := a.(jsJFlo)
	_, bf := b.(jsJFlo)
	if af || bf {
		return rt.toNumber(a) < rt.toNumber(b)
	}
	return rt.csCmp("<", a, b)
}

// csxMcall is the method call the shared file makes: a user method (the
// receiver prepended, as every instance call passes it), else System.Object's
// Equals.
func (rt *jsrt) csxMcall(o interface{}, name string, args []interface{}) interface{} {
	if m := rt.csUserMethod(o, name); m != nil {
		return rt.call(m, jsUndef, append([]interface{}{o}, args...))
	}
	if name == "Equals" && len(args) == 1 {
		return rt.csObjEquals(o, args[0])
	}
	rt.fail("unknown method '%s'", name)
	return nil
}

// csxCall invokes a delegate; a multicast one runs in order and the last result
// is the call's.
func (rt *jsrt) csxCall(f interface{}, args []interface{}) interface{} {
	if arr, ok := f.(*jsArray); ok {
		var r interface{} = jsUndef
		for _, g := range append([]interface{}{}, arr.elems...) {
			r = rt.call(g, jsUndef, args)
		}
		return r
	}
	return rt.call(f, jsUndef, args)
}

func (rt *jsrt) csxPred(f, v interface{}) bool {
	b, _ := rt.csxCall(f, []interface{}{v}).(bool)
	return b
}

// csxForSrc is what a foreach over v visits: a string's chars, a dictionary's
// KeyValuePairs, else v itself. js_csforsrc.
func (rt *jsrt) csxForSrc(v interface{}) interface{} {
	if _, _, ok := dictParts(v); ok {
		return rt.csxDictPairs(v.(*jsObject))
	}
	if _, ok := csqIsQuery(v); ok {
		return &jsArray{elems: rt.csqToArray(v)}
	}
	s, ok := v.(string)
	if !ok {
		return v
	}
	units := strUnits(s)
	out := &jsArray{elems: make([]interface{}, len(units))}
	for i, c := range units {
		out.elems[i] = jsChar{code: int32(c)}
	}
	return out
}

func (rt *jsrt) csxRefSet(box, v interface{}) {
	if o, ok := box.(*jsObject); ok {
		if b, isRef := o.props["__csref"].(bool); isRef && b {
			rt.call(o.props["wb"], jsUndef, []interface{}{v})
		}
	}
}

// ----- Comparer<T>.Default and EqualityComparer<T>.Default -----

func (rt *jsrt) csxNumCmp(a, b interface{}) int {
	x, y := rt.toNumber(a), rt.toNumber(b)
	an, bn := math.IsNaN(x), math.IsNaN(y)
	if an || bn {
		switch {
		case an && bn:
			return 0
		case an:
			return -1
		}
		return 1
	}
	if rt.csxLt(a, b) {
		return -1
	}
	if rt.csxLt(b, a) {
		return 1
	}
	return 0
}

func (rt *jsrt) csxCmp(a, b interface{}) int {
	if isUndefOrNull(a) {
		if isUndefOrNull(b) {
			return 0
		}
		return -1
	}
	if isUndefOrNull(b) {
		return 1
	}
	if csxIsNum(a) && csxIsNum(b) {
		return rt.csxNumCmp(a, b)
	}
	ab, aok := a.(bool)
	bb, bok := b.(bool)
	if aok && bok {
		switch {
		case ab == bb:
			return 0
		case ab:
			return 1
		}
		return -1
	}
	if _, ok := a.(string); ok {
		rt.fail("the default string order is the culture's collation, which is not modelled: pass a Comparison")
	}
	if rt.csUserMethod(a, "CompareTo") != nil {
		return jsToInt(rt.toNumber(rt.csxMcall(a, "CompareTo", []interface{}{b})))
	}
	rt.csThrow("InvalidOperationException", "Failed to compare two elements in the array.")
	return 0
}

func (rt *jsrt) csxEq(a, b interface{}) bool {
	if isUndefOrNull(a) {
		return isUndefOrNull(b)
	}
	if csxIsNum(a) && csxIsNum(b) {
		return rt.csxNumCmp(a, b) == 0
	}
	r, _ := rt.csxMcall(a, "Equals", []interface{}{b}).(bool)
	return r
}

func (rt *jsrt) csxCmpWith(cmp, a, b interface{}) int {
	if isUndefOrNull(cmp) {
		return rt.csxCmp(a, b)
	}
	if k := csxStrComparerKind(cmp); k >= 0 {
		return int(csxCmpStr(k, a, b))
	}
	if cspIsDeleg(cmp) {
		return jsToInt(rt.toNumber(rt.csxCall(cmp, []interface{}{a, b})))
	}
	return jsToInt(rt.toNumber(rt.csxMcall(cmp, "Compare", []interface{}{a, b})))
}

func csxZeroLike(v interface{}) interface{} {
	switch v.(type) {
	case jsChar:
		return jsChar{code: 0}
	case bool:
		return false
	case float64, jsGInt, jsJFlo:
		return float64(0)
	}
	return jsNull
}

func csxDefaultOf(ty string) interface{} {
	switch {
	case ty == "" || ty[len(ty)-1] == '?':
		return jsNull
	case ty == "bool":
		return false
	case ty == "char":
		return jsChar{code: 0}
	}
	for _, c := range ty {
		if c == '[' {
			return jsNull
		}
	}
	switch ty {
	case "int", "uint", "long", "ulong", "short", "ushort", "byte", "sbyte", "double", "float", "decimal":
		return float64(0)
	}
	return jsNull
}

// ----- the kind a List<T> or T[] carries: lib/csharp-collections.js's -----

var csxTyMu sync.Mutex
var csxTyNames = []string{""}

func csxTyCode(ty string) int {
	csxTyMu.Lock()
	defer csxTyMu.Unlock()
	for i, t := range csxTyNames {
		if t == ty {
			return i
		}
	}
	csxTyNames = append(csxTyNames, ty)
	return len(csxTyNames) - 1
}

func csxTyped(a *jsArray, ety string, isList bool) *jsArray {
	k := 2 * csxTyCode(ety)
	if isList {
		k++
	}
	a.kind = k
	return a
}

func csxIsList(a *jsArray) bool { return a.kind%2 == 1 }

func csxElemTy(a *jsArray) string {
	csxTyMu.Lock()
	defer csxTyMu.Unlock()
	if i := a.kind / 2; i < len(csxTyNames) {
		return csxTyNames[i]
	}
	return ""
}

func csxElemDefault(a *jsArray) interface{} {
	var first interface{} = jsNull
	if len(a.elems) > 0 {
		first = a.elems[0]
	}
	return csxDefaultFor(csxElemTy(a), len(a.elems) > 0, first)
}

// csxDefaultFor: lib/csharp-collections.js's.
func csxDefaultFor(ty string, seen bool, any interface{}) interface{} {
	if ty != "" {
		if d := csxDefaultOf(ty); !isUndefOrNull(d) || ty[len(ty)-1] == '?' {
			return d
		}
	}
	if seen {
		return csxZeroLike(any)
	}
	return jsNull
}

// ----- List<T> -----

func (rt *jsrt) csxElemsOf(v interface{}) []interface{} {
	if isUndefOrNull(v) {
		rt.csThrow("ArgumentNullException", "Value cannot be null. (Parameter 'collection')")
	}
	if e, ok := v.(*jsArray); ok {
		return append([]interface{}{}, e.elems...)
	}
	return rt.csqToArray(v)
}

func (rt *jsrt) csxRangeChk(n, index, count int) {
	if index < 0 {
		rt.csxArgRange(csxMsgNonNeg, "index")
	}
	if count < 0 {
		rt.csxArgRange(csxMsgNonNeg, "count")
	}
	if n-index < count {
		rt.csThrow("ArgumentException", csxMsgOffLen)
	}
}

func csxInsertAll(a *jsArray, at int, src []interface{}) {
	a.dropIdx()
	out := make([]interface{}, 0, len(a.elems)+len(src))
	out = append(out, a.elems[:at]...)
	out = append(out, src...)
	a.elems = append(out, a.elems[at:]...)
}

func csxRemoveAt(a *jsArray, i int) {
	a.dropIdx()
	a.elems = append(a.elems[:i], a.elems[i+1:]...)
}

// csxSortRange is the stable merge sort of a[b:e] (lib/csharp-collections.js's
// csxSortRange; Go's sort.SliceStable would call the comparer in another order).
func (rt *jsrt) csxSortRange(a *jsArray, b, e int, cmp interface{}) {
	n := e - b
	if n < 2 {
		return
	}
	a.dropIdx()
	src := append([]interface{}{}, a.elems[b:e]...)
	for width := 1; width < n; width *= 2 {
		dst := make([]interface{}, 0, n)
		for lo := 0; lo < n; {
			mid, hi := lo+width, lo+2*width
			if mid > n {
				mid = n
			}
			if hi > n {
				hi = n
			}
			i, j := lo, mid
			for i < mid && j < hi {
				if rt.csxCmpWith(cmp, src[j], src[i]) < 0 {
					dst = append(dst, src[j])
					j++
				} else {
					dst = append(dst, src[i])
					i++
				}
			}
			dst = append(dst, src[i:mid]...)
			dst = append(dst, src[j:hi]...)
			lo = hi
		}
		src = dst
	}
	copy(a.elems[b:e], src)
}

func csxReverseRange(a *jsArray, b, e int) {
	a.dropIdx()
	for i, j := b, e-1; i < j; i, j = i+1, j-1 {
		a.elems[i], a.elems[j] = a.elems[j], a.elems[i]
	}
}

func (rt *jsrt) csxFindIdx(a []interface{}, from, count, step int, pred interface{}) int {
	for i, k := from, 0; k < count; i, k = i+step, k+1 {
		if rt.csxPred(pred, a[i]) {
			return i
		}
	}
	return -1
}

func (rt *jsrt) csxSearch(a []interface{}, item interface{}, from, count int) int {
	for i := from; i < from+count; i++ {
		if rt.csxEq(a[i], item) {
			return i
		}
	}
	return -1
}

func (rt *jsrt) csxSearchBack(a []interface{}, item interface{}, from, count int) int {
	for i := from; i > from-count; i-- {
		if rt.csxEq(a[i], item) {
			return i
		}
	}
	return -1
}

func (rt *jsrt) csxListIndexOf(a []interface{}, args []interface{}) int {
	n := len(a)
	index := 0
	if len(args) > 1 {
		index = rt.csxNum(args[1])
	}
	if index > n {
		rt.csxArgRange(csxMsgIdxLessEq, "index")
	}
	if index < 0 {
		rt.csxArgRange(csxMsgIdxLessEq, "startIndex")
	}
	count := n - index
	if len(args) > 2 {
		count = rt.csxNum(args[2])
		if count < 0 || index > n-count {
			rt.csxArgRange(csxMsgCount, "count")
		}
	}
	return rt.csxSearch(a, args[0], index, count)
}

func (rt *jsrt) csxListLastIndexOf(a []interface{}, args []interface{}) int {
	n := len(a)
	if len(args) == 1 {
		if n == 0 {
			return -1
		}
		return rt.csxSearchBack(a, args[0], n-1, n)
	}
	index := rt.csxNum(args[1])
	count := index + 1
	if len(args) == 2 {
		if index >= n {
			rt.csxArgRange(csxMsgIdxLess, "index")
		}
	} else {
		count = rt.csxNum(args[2])
		if n != 0 && index < 0 {
			rt.csxArgRange(csxMsgNonNeg, "index")
		}
		if n != 0 && count < 0 {
			rt.csxArgRange(csxMsgNonNeg, "count")
		}
		if n == 0 {
			return -1
		}
		if index >= n {
			rt.csxArgRange(csxMsgIdxLess, "index")
		}
		if count > index+1 {
			rt.csxArgRange(csxMsgCount, "count")
		}
	}
	return rt.csxSearchBack(a, args[0], index, count)
}

func (rt *jsrt) csxListFindIndex(a []interface{}, args []interface{}) int {
	n := len(a)
	pred := args[len(args)-1]
	start := 0
	if len(args) > 1 {
		start = rt.csxNum(args[0])
	}
	if start < 0 || start > n {
		rt.csxArgRange(csxMsgIdxLessEq, "startIndex")
	}
	count := n - start
	if len(args) > 2 {
		count = rt.csxNum(args[1])
	}
	if count < 0 || start > n-count {
		rt.csxArgRange(csxMsgCount, "count")
	}
	return rt.csxFindIdx(a, start, count, 1, pred)
}

func (rt *jsrt) csxListFindLastIndex(a []interface{}, args []interface{}) int {
	n := len(a)
	pred := args[len(args)-1]
	start, count := n-1, n
	if len(args) > 1 {
		start = rt.csxNum(args[0])
		count = start + 1
	}
	if len(args) > 2 {
		count = rt.csxNum(args[1])
	}
	if n == 0 {
		if start != -1 {
			rt.csxArgRange(csxMsgIdxLess, "startIndex")
		}
	} else if start < 0 || start >= n {
		rt.csxArgRange(csxMsgIdxLess, "startIndex")
	}
	if count < 0 || start-count+1 < 0 {
		rt.csxArgRange(csxMsgCount, "count")
	}
	return rt.csxFindIdx(a, start, count, -1, pred)
}

func (rt *jsrt) csxBinarySearch(a []interface{}, args []interface{}) int {
	lo, hi := 0, len(a)-1
	item := args[0]
	var cmp interface{} = jsNull
	if len(args) > 1 {
		cmp = args[1]
	}
	if len(args) > 2 {
		lo = rt.csxNum(args[0])
		cnt := rt.csxNum(args[1])
		rt.csxRangeChk(len(a), lo, cnt)
		hi = lo + cnt - 1
		item = args[2]
		cmp = jsNull
		if len(args) > 3 {
			cmp = args[3]
		}
	}
	for lo <= hi {
		i := lo + (hi-lo)/2
		c := rt.csxCmpWith(cmp, a[i], item)
		if c == 0 {
			return i
		}
		if c < 0 {
			lo = i + 1
		} else {
			hi = i - 1
		}
	}
	return -lo - 1
}

func (rt *jsrt) csxListCtor(a *jsArray, args []interface{}) interface{} {
	if len(args) == 0 {
		return a
	}
	if csxIsNum(args[0]) {
		if rt.csxNum(args[0]) < 0 {
			rt.csxArgRange(csxMsgNonNeg, "capacity")
		}
		return a
	}
	csxInsertAll(a, 0, rt.csxElemsOf(args[0]))
	return a
}

func (rt *jsrt) csxListMethod(a *jsArray, name string, args []interface{}) interface{} {
	n := len(args)
	arg := func(i int) interface{} { return argAt(args, i) }
	switch name {
	case "__ctor":
		return rt.csxListCtor(a, args)
	case "Add":
		a.dropIdx()
		a.elems = append(a.elems, arg(0))
		return jsUndef
	case "Count":
		if n > 0 {
			return rt.csqMethod(a, name, args)
		}
		return float64(len(a.elems))
	case "Contains":
		if n != 1 {
			return rt.csqMethod(a, name, args)
		}
		return rt.csxSearch(a.elems, arg(0), 0, len(a.elems)) >= 0
	case "IndexOf":
		return float64(rt.csxListIndexOf(a.elems, args))
	case "LastIndexOf":
		return float64(rt.csxListLastIndexOf(a.elems, args))
	case "Clear":
		a.dropIdx()
		a.elems = nil
		return jsUndef
	case "AddRange":
		csxInsertAll(a, len(a.elems), rt.csxElemsOf(arg(0)))
		return jsUndef
	case "Insert", "InsertRange":
		at := rt.csxNum(arg(0))
		if at < 0 || at > len(a.elems) {
			if name == "Insert" {
				rt.csxArgRange("Index must be within the bounds of the List.", "index")
			}
			rt.csxArgRange(csxMsgIdxLessEq, "index")
		}
		if name == "Insert" {
			csxInsertAll(a, at, []interface{}{arg(1)})
		} else {
			csxInsertAll(a, at, rt.csxElemsOf(arg(1)))
		}
		return jsUndef
	case "Remove":
		i := rt.csxSearch(a.elems, arg(0), 0, len(a.elems))
		if i < 0 {
			return false
		}
		csxRemoveAt(a, i)
		return true
	case "RemoveAt":
		k := rt.csxNum(arg(0))
		if k < 0 || k >= len(a.elems) {
			rt.csxArgRange(csxMsgIdxLess, "index")
		}
		csxRemoveAt(a, k)
		return jsUndef
	case "RemoveRange":
		b, c := rt.csxNum(arg(0)), rt.csxNum(arg(1))
		rt.csxRangeChk(len(a.elems), b, c)
		a.dropIdx()
		a.elems = append(a.elems[:b], a.elems[b+c:]...)
		return jsUndef
	case "RemoveAll":
		var keep []interface{}
		gone := 0
		for _, e := range append([]interface{}{}, a.elems...) {
			if rt.csxPred(arg(0), e) {
				gone++
			} else {
				keep = append(keep, e)
			}
		}
		a.dropIdx()
		a.elems = keep
		return float64(gone)
	case "GetRange", "Slice":
		b, c := rt.csxNum(arg(0)), rt.csxNum(arg(1))
		rt.csxRangeChk(len(a.elems), b, c)
		return csxTyped(&jsArray{elems: append([]interface{}{}, a.elems[b:b+c]...)}, csxElemTy(a), true)
	case "ToArray":
		return csxTyped(&jsArray{elems: append([]interface{}{}, a.elems...)}, csxElemTy(a), false)
	case "Reverse":
		// An array has no Reverse() of its own: it is Enumerable.Reverse.
		if n == 0 && !csxIsList(a) {
			return rt.csqMethod(a, name, args)
		}
		if n == 0 {
			csxReverseRange(a, 0, len(a.elems))
			return jsUndef
		}
		b, c := rt.csxNum(arg(0)), rt.csxNum(arg(1))
		rt.csxRangeChk(len(a.elems), b, c)
		csxReverseRange(a, b, b+c)
		return jsUndef
	case "Sort":
		if n == 3 {
			b, c := rt.csxNum(arg(0)), rt.csxNum(arg(1))
			rt.csxRangeChk(len(a.elems), b, c)
			rt.csxSortRange(a, b, b+c, arg(2))
			return jsUndef
		}
		var cmp interface{} = jsNull
		if n == 1 {
			cmp = arg(0)
		}
		rt.csxSortRange(a, 0, len(a.elems), cmp)
		return jsUndef
	case "BinarySearch":
		return float64(rt.csxBinarySearch(a.elems, args))
	case "Exists":
		return rt.csxFindIdx(a.elems, 0, len(a.elems), 1, arg(0)) >= 0
	case "FindIndex":
		return float64(rt.csxListFindIndex(a.elems, args))
	case "FindLastIndex":
		return float64(rt.csxListFindLastIndex(a.elems, args))
	case "Find", "FindLast":
		var fi int
		if name == "Find" {
			fi = rt.csxFindIdx(a.elems, 0, len(a.elems), 1, arg(0))
		} else {
			fi = rt.csxFindIdx(a.elems, len(a.elems)-1, len(a.elems), -1, arg(0))
		}
		if fi >= 0 {
			return a.elems[fi]
		}
		return csxElemDefault(a)
	case "FindAll":
		out := &jsArray{}
		for _, e := range append([]interface{}{}, a.elems...) {
			if rt.csxPred(arg(0), e) {
				out.elems = append(out.elems, e)
			}
		}
		return csxTyped(out, csxElemTy(a), true)
	case "TrueForAll":
		for _, e := range append([]interface{}{}, a.elems...) {
			if !rt.csxPred(arg(0), e) {
				return false
			}
		}
		return true
	case "ForEach":
		for i := 0; i < len(a.elems); i++ {
			rt.csxCall(arg(0), []interface{}{a.elems[i]})
		}
		return jsUndef
	case "ConvertAll":
		out := &jsArray{}
		for _, e := range append([]interface{}{}, a.elems...) {
			out.elems = append(out.elems, rt.csxCall(arg(0), []interface{}{e}))
		}
		return csxTyped(out, "", true)
	case "CopyTo":
		dst, _ := arg(0).(*jsArray)
		di, si, cn := 0, 0, len(a.elems)
		if n == 2 {
			di = rt.csxNum(arg(1))
		}
		if n == 4 {
			si = rt.csxNum(arg(0))
			dst, _ = arg(1).(*jsArray)
			di = rt.csxNum(arg(2))
			cn = rt.csxNum(arg(3))
			rt.csxRangeChk(len(a.elems), si, cn)
		}
		if di < 0 {
			rt.csxArgRange(csxMsgNonNeg, "arrayIndex")
		}
		if dst == nil || len(dst.elems)-di < cn {
			rt.csThrow("ArgumentException", "Destination array was not long enough. Check the destination index, length, and the array's lower bounds. (Parameter 'destinationArray')")
		}
		dst.dropIdx()
		copy(dst.elems[di:di+cn], a.elems[si:si+cn])
		return jsUndef
	}
	return rt.csqMethod(a, name, args)
}

// ----- Dictionary<TKey,TValue> -----

func csxKvp(k, v interface{}) *jsObject {
	o := newJSObject()
	o.set("__kvp", true)
	o.set("__tuple", float64(2))
	o.set("Item1", k)
	o.set("Item2", v)
	o.set("Key", k)
	o.set("Value", v)
	return o
}

func csxIsKvp(o *jsObject) bool {
	b, ok := o.props["__kvp"].(bool)
	return ok && b
}

func (rt *jsrt) csxDictPairs(d *jsObject) *jsArray {
	keys, vals, _ := dictParts(d)
	out := &jsArray{elems: make([]interface{}, len(keys.elems))}
	for i, k := range keys.elems {
		out.elems[i] = csxKvp(k, vals.elems[i])
	}
	return out
}

func (rt *jsrt) csxKeyChk(k interface{}) {
	if isUndefOrNull(k) {
		rt.csThrow("ArgumentNullException", "Value cannot be null. (Parameter 'key')")
	}
}

func csxDictDefault(d *jsObject, vals *jsArray) interface{} {
	if t, ok := d.props["__vty"].(string); ok {
		return csxDefaultOf(t)
	}
	if len(vals.elems) > 0 {
		return csxZeroLike(vals.elems[0])
	}
	return jsNull
}

func (rt *jsrt) csxDictCtor(d *jsObject, args []interface{}) interface{} {
	if len(args) == 0 {
		return d
	}
	if csxIsNum(args[0]) {
		if rt.csxNum(args[0]) < 0 {
			rt.csxArgRange(csxMsgNonNeg, "capacity")
		}
		return d
	}
	for _, p := range rt.csxElemsOf(args[0]) {
		po, _ := p.(*jsObject)
		if po == nil {
			rt.fail("not a KeyValuePair")
		}
		rt.csxDictMethod(d, "Add", []interface{}{po.props["Key"], po.props["Value"]})
	}
	return d
}

func (rt *jsrt) csxDictMethod(d *jsObject, name string, args []interface{}) interface{} {
	keys, vals, _ := dictParts(d)
	n := len(args)
	arg := func(i int) interface{} { return argAt(args, i) }
	switch name {
	case "__ctor":
		return rt.csxDictCtor(d, args)
	case "Keys":
		return &jsArray{elems: append([]interface{}{}, keys.elems...)}
	case "Values":
		return &jsArray{elems: append([]interface{}{}, vals.elems...)}
	case "Count":
		if n > 0 {
			return rt.csqMethod(d, name, args)
		}
		return float64(len(keys.elems))
	case "Add", "TryAdd":
		rt.csxKeyChk(arg(0))
		if rt.dictFind(keys, arg(0)) >= 0 {
			if name == "TryAdd" {
				return false
			}
			rt.csThrow("ArgumentException", "An item with the same key has already been added. Key: "+rt.cspStr(arg(0)))
		}
		keys.elems = append(keys.elems, arg(0))
		vals.dropIdx()
		vals.elems = append(vals.elems, arg(1))
		if name == "TryAdd" {
			return true
		}
		return jsUndef
	case "ContainsKey":
		rt.csxKeyChk(arg(0))
		return rt.dictFind(keys, arg(0)) >= 0
	case "ContainsValue":
		return rt.csxSearch(vals.elems, arg(0), 0, len(vals.elems)) >= 0
	case "Clear":
		keys.dropIdx()
		vals.dropIdx()
		keys.elems = nil
		vals.elems = nil
		return jsUndef
	case "Remove", "TryGetValue":
		rt.csxKeyChk(arg(0))
		i := rt.dictFind(keys, arg(0))
		var got interface{}
		if i >= 0 {
			got = vals.elems[i]
		} else {
			got = csxDictDefault(d, vals)
		}
		if n > 1 {
			rt.csxRefSet(arg(1), got)
		}
		if i >= 0 && name == "Remove" {
			csxRemoveAt(keys, i)
			csxRemoveAt(vals, i)
		}
		return i >= 0
	case "GetValueOrDefault":
		rt.csxKeyChk(arg(0))
		if i := rt.dictFind(keys, arg(0)); i >= 0 {
			return vals.elems[i]
		}
		if n > 1 {
			return arg(1)
		}
		return csxDictDefault(d, vals)
	}
	return rt.csqMethod(d, name, args)
}
