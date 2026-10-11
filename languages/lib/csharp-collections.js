// csharp-collections.js -- System.Collections.Generic's List<T> and
// Dictionary<TKey,TValue> members, for csharp-interpreter.abnf (include) and
// languages/lib/csharp-rt.metajs (import). abnf/jsrtcscoll.go is the Go twin's
// port; the three must answer alike. It sits on lib/csharp-strings.js's hooks
// (csxkIsChar, csxkCode, csxkChar, csxkNum, csxkStr, csxkThrow) and asks the
// value model through these:
//   csxkIsNum(v)        a number of any width, or a char (the numeric operands)
//   csxkLt(a, b)        `<` on two such operands, at their promoted type
//   csxkMcall(o, n, a)  a method call, a user override winning
//   csxkHasMethod(o, n) does a user class instance declare method n
//   csxkIsDeleg(v)      is v a delegate
//   csxkCall(f, args)   invoke a delegate
//   csxkElems(v)        the elements a foreach over v visits, as an array
//   csxkDictFind(d, k)  the position of key k in a dictionary, or -1
//   csxkRefSet(box, v)  write v through an `out` argument's reference box
//
// A List<T> is a plain array and a Dictionary a {__dict, keys, vals} object. A
// member that hands out default(T) (Find, TryGetValue) reads T where the value
// carries it: a List or array created with its element type spelled carries T
// in its kind (csxTyped), a Dictionary created by `new Dictionary<K,V>()` V's
// name in __vty; otherwise the default of an element's own kind stands in, and
// null when there is no element.
//
// Sort() and BinarySearch() order by Comparer<T>.Default: numbers and chars by
// value (Double.CompareTo puts NaN first), bools false-first, and a user type
// by its CompareTo. A string's default order is the CULTURE's collation, which
// is not modelled (lib/csharp-strings.js leaves String.CompareTo absent for the
// same reason), so it fails rather than answering an ordinal order. Sort is
// stable here; .NET's introsort is not, so elements a comparison calls equal may
// come out in a different order than .NET's (an order .NET leaves unspecified).
//
// The exception CLASSES are the documented ones; the messages are .NET's
// resource strings, which no toolchain here can settle (manual chapter 7.9).

function csxMsgIdxLess() { return "Index was out of range. Must be non-negative and less than the size of the collection." }
function csxMsgIdxLessEq() { return "Index was out of range. Must be non-negative and less than or equal to the size of the collection." }
function csxMsgCount() { return "Count must be positive and count must refer to a location within the string/array/collection." }
function csxMsgNonNeg() { return "Non-negative number required." }
function csxMsgOffLen() { return "Offset and length were out of bounds for the array or count is greater than the number of elements from index to the end of the source collection." }

function csxArgRange(msg, param) {
    csxkThrow("ArgumentOutOfRangeException", msg + " (Parameter '" + param + "')")
}

// ----- Comparer<T>.Default and EqualityComparer<T>.Default -----

// Double.CompareTo: NaN is less than every number and equal to itself.
function csxNumCmp(a, b) {
    var x = csxkCode(a)
    var y = csxkCode(b)
    var an = x !== x
    var bn = y !== y
    if (an || bn) {
        if (an && bn) { return 0 }
        return an ? 0 - 1 : 1
    }
    if (csxkLt(a, b)) { return 0 - 1 }
    if (csxkLt(b, a)) { return 1 }
    return 0
}
function csxCmp(a, b) {
    if (csxIsNull(a)) { return csxIsNull(b) ? 0 : 0 - 1 }
    if (csxIsNull(b)) { return 1 }
    if (csxkIsNum(a) && csxkIsNum(b)) { return csxNumCmp(a, b) }
    if (typeof a == "boolean" && typeof b == "boolean") {
        if (a === b) { return 0 }
        return a ? 1 : 0 - 1
    }
    if (typeof a == "string") {
        fail("the default string order is the culture's collation, which is not modelled: pass a Comparison")
    }
    if (csxkHasMethod(a, "CompareTo")) { return csxkNum(csxkMcall(a, "CompareTo", [b])) }
    csxkThrow("InvalidOperationException", "Failed to compare two elements in the array.")
    return 0
}
// Equals on two numbers compares their values: `l.Contains(5)` on a List<double>
// has had its argument converted to double by the C# compiler.
function csxEq(a, b) {
    if (csxIsNull(a)) { return csxIsNull(b) }
    if (csxkIsNum(a) && csxkIsNum(b)) { return csxNumCmp(a, b) == 0 }
    return csxkMcall(a, "Equals", [b]) === true
}
// `x.CompareTo(y)` on a number or a bool (Int32, Double, Boolean.CompareTo).
function csxValueCompareTo(recv, args) { return csxCmp(recv, args[0]) }

// A Sort / BinarySearch comparer argument: none (the default), a Comparison<T>
// delegate, or an IComparer<T> object.
function csxCmpWith(cmp, a, b) {
    if (csxIsNull(cmp)) { return csxCmp(a, b) }
    if (csxIsStrComparer(cmp)) { return csxCmpStr(cmp["__csstrcmp"], a, b) }
    if (csxkIsDeleg(cmp)) { return csxkNum(csxkCall(cmp, [a, b])) }
    return csxkNum(csxkMcall(cmp, "Compare", [a, b]))
}

// default(T) where T is not known: the default of v's own kind.
function csxZeroLike(v) {
    if (csxkIsChar(v)) { return csxkChar(0) }
    if (typeof v == "boolean") { return false }
    if (csxkIsNum(v)) { return 0 }
    return null
}
// default(T) by type name (ECMA-334 9.3): the numeric zero, false, the null char;
// null for a reference type and a nullable one.
function csxDefaultOf(ty) {
    if (ty.length == 0) { return null }
    if (ty.charAt(ty.length - 1) == "?") { return null }
    if (ty.indexOf("[") >= 0) { return null }
    if (ty == "bool") { return false }
    if (ty == "char") { return csxkChar(0) }
    if (ty == "int" || ty == "uint" || ty == "long" || ty == "ulong" || ty == "short"
        || ty == "ushort" || ty == "byte" || ty == "sbyte" || ty == "double"
        || ty == "float" || ty == "decimal") { return 0 }
    return null
}

// ----- the kind a List<T> or T[] carries -----
//
// Both are plain arrays. The array's kind number (arrKind) tells them apart and
// names T where the program spelled it: bit 0 is set on a List, bits 1 to 10 hold
// T's index in csxTyNames (0: not known), and the bits from 2048 up a List's
// Capacity (csxCap). A List made with its elements has the capacity Adding them
// one by one gives. Twin: csxTyped in abnf/jsrtcscoll.go.
var csxTyNames = [""]
function csxTyCode(ty) {
    var i = 0
    while (i < csxTyNames.length) {
        if (csxTyNames[i] == ty) { return i }
        i = i + 1
    }
    if (csxTyNames.length == 1024) { return 0 }
    csxTyNames.push(ty)
    return csxTyNames.length - 1
}
function csxTyped(a, ety, isList) {
    arrKindSet(a, (isList ? 1 : 0) + 2 * csxTyCode(ety))
    if (isList && csxCapOn) { csxCapAdds(a, a.length) }
    return a
}
function csxIsList(a) { return arrKind(a) % 2 == 1 }
function csxElemTy(a) { return csxTyNames[Math.floor(arrKind(a) / 2) % 1024] }

// ----- List<T>.Capacity -----
//
// Kept only when the program names Capacity, TrimExcess or EnsureCapacity
// (csxCapOn, set before it runs), so no other List.Add pays for it. The growth is
// List<T>'s: an empty list holds 0, a full one grows to twice its capacity (4
// from 0), or to the size needed when that is more; the documented example reads
// 0, then 8 after five Adds, then 5 after TrimExcess, and Clear keeps it.
var csxCapOn = false
function csxCap(a) { return Math.floor(arrKind(a) / 2048) }
function csxCapSet(a, c) { arrKindSet(a, arrKind(a) % 2048 + 2048 * c) }
function csxCapGrow(a, need) {
    var cap = csxCap(a)
    if (cap >= need) { return }
    var nc = cap == 0 ? 4 : 2 * cap
    if (nc < need) { nc = need }
    csxCapSet(a, nc)
}
// The capacity n more Adds leave, from a.length - n elements.
function csxCapAdds(a, n) {
    var k = a.length - n
    while (k < a.length) {
        k = k + 1
        csxCapGrow(a, k)
    }
}
// AddRange / InsertRange: a collection whose size is known grows the list once,
// any other sequence one Add at a time.
function csxCapRange(a, src, n) {
    if (Array.isArray(src) || csxIsDictLike(src)) {
        if (n > 0) { csxCapGrow(a, a.length + n) }
        return
    }
    var k = 0
    while (k < n) {
        k = k + 1
        csxCapGrow(a, a.length + k)
    }
}
function csxCapMethod(a, name, args) {
    if (name == "Capacity") { return csxCap(a) }
    if (name == "set_Capacity") {
        var c = csxkNum(args[0])
        if (c < a.length) { csxArgRange("capacity was less than the current size.", "value") }
        csxCapSet(a, c)
        return undefined
    }
    if (name == "TrimExcess") {
        if (a.length < Math.floor(csxCap(a) * 0.9)) { csxCapSet(a, a.length) }
        return undefined
    }
    var need = csxkNum(args[0])
    if (need < 0) { csxArgRange(csxMsgNonNeg(), "capacity") }
    csxCapGrow(a, need)
    return csxCap(a)
}
// default(T) for a's elements: by T when it is known, else the default of the
// first element's own kind, else null.
function csxElemDefault(a) { return csxDefaultFor(csxElemTy(a), a.length > 0, a.length > 0 ? a[0] : null) }
// default(T) for T named ty (a value type, or a nullable one), else the default
// of the kind of an element seen (a type parameter or a reference type), else null.
function csxDefaultFor(ty, seen, any) {
    if (ty != "") {
        var d = anytype
        d = csxDefaultOf(ty)
        if (d !== null || ty.charAt(ty.length - 1) == "?") { return d }
    }
    return seen ? csxZeroLike(any) : null
}
// The collection a declaration of type List<ety> / ety[] initializes: an untagged
// array (a collection expression, `List<int> l = [];`) takes the declared type.
function csxTypedAs(a, ety, isList) {
    if (!Array.isArray(a) || arrKind(a) != 0) { return a }
    return csxTyped(a, ety, isList)
}

// ----- List<T> -----

function csxRemoveAt(a, i) {
    var j = i
    while (j + 1 < a.length) {
        a[j] = a[j + 1]
        j = j + 1
    }
    a.pop()
}
function csxInsertAll(a, at, src) {
    var n = src.length
    var tail = []
    var i = at
    while (i < a.length) {
        tail.push(a[i])
        i = i + 1
    }
    while (a.length > at) { a.pop() }
    i = 0
    while (i < n) {
        a.push(src[i])
        i = i + 1
    }
    i = 0
    while (i < tail.length) {
        a.push(tail[i])
        i = i + 1
    }
}
// The elements of an IEnumerable<T> argument, copied (so `l.AddRange(l)` reads
// the list before it grows).
function csxElemsOf(v) {
    if (csxIsNull(v)) { csxkThrow("ArgumentNullException", "Value cannot be null. (Parameter 'collection')") }
    if (!Array.isArray(v)) { return csqToArray(v) }
    var out = []
    var i = 0
    while (i < v.length) {
        out.push(v[i])
        i = i + 1
    }
    return out
}
// (index, count) naming a range of a list of length n: RemoveRange, GetRange,
// Reverse, Sort.
function csxRangeChk(n, index, count) {
    if (index < 0) { csxArgRange(csxMsgNonNeg(), "index") }
    if (count < 0) { csxArgRange(csxMsgNonNeg(), "count") }
    if (n - index < count) { csxkThrow("ArgumentException", csxMsgOffLen()) }
}
function csxSlice(a, b, e) {
    var out = []
    var i = b
    while (i < e) {
        out.push(a[i])
        i = i + 1
    }
    return out
}
// Stable merge sort of a[b..e) by the comparer.
function csxSortRange(a, b, e, cmp) {
    var n = e - b
    if (n < 2) { return }
    var src = csxSlice(a, b, e)
    var width = 1
    while (width < n) {
        var dst = []
        var lo = 0
        while (lo < n) {
            var mid = lo + width < n ? lo + width : n
            var hi = lo + 2 * width < n ? lo + 2 * width : n
            var i = lo
            var j = mid
            while (i < mid && j < hi) {
                if (csxCmpWith(cmp, src[j], src[i]) < 0) {
                    dst.push(src[j])
                    j = j + 1
                } else {
                    dst.push(src[i])
                    i = i + 1
                }
            }
            while (i < mid) {
                dst.push(src[i])
                i = i + 1
            }
            while (j < hi) {
                dst.push(src[j])
                j = j + 1
            }
            lo = hi
        }
        src = dst
        width = width * 2
    }
    var k = 0
    while (k < n) {
        a[b + k] = src[k]
        k = k + 1
    }
}
function csxReverseRange(a, b, e) {
    var i = b
    var j = e - 1
    while (i < j) {
        var t = a[i]
        a[i] = a[j]
        a[j] = t
        i = i + 1
        j = j - 1
    }
}
// FindIndex over [b, e) forward, or (e, b] backward when step is -1.
function csxFindIdx(a, from, count, step, pred) {
    var i = from
    var k = 0
    while (k < count) {
        if (csxkCall(pred, [a[i]]) === true) { return i }
        i = i + step
        k = k + 1
    }
    return 0 - 1
}
function csxSearch(a, item, from, count) {
    var i = from
    while (i < from + count) {
        if (csxEq(a[i], item)) { return i }
        i = i + 1
    }
    return 0 - 1
}
function csxSearchBack(a, item, from, count) {
    var i = from
    while (i > from - count) {
        if (csxEq(a[i], item)) { return i }
        i = i - 1
    }
    return 0 - 1
}
// List<T>.IndexOf(item [, index [, count]]).
function csxListIndexOf(a, args) {
    var n = a.length
    var index = args.length > 1 ? csxkNum(args[1]) : 0
    if (index > n) { csxArgRange(csxMsgIdxLessEq(), "index") }
    if (index < 0) { csxArgRange(csxMsgIdxLessEq(), "startIndex") }
    var count = n - index
    if (args.length > 2) {
        count = csxkNum(args[2])
        if (count < 0 || index > n - count) { csxArgRange(csxMsgCount(), "count") }
    }
    return csxSearch(a, args[0], index, count)
}
// List<T>.LastIndexOf(item [, index [, count]]): backward from index.
function csxListLastIndexOf(a, args) {
    var n = a.length
    if (args.length == 1) {
        if (n == 0) { return 0 - 1 }
        return csxSearchBack(a, args[0], n - 1, n)
    }
    var index = csxkNum(args[1])
    var count = index + 1
    if (args.length == 2) {
        if (index >= n) { csxArgRange(csxMsgIdxLess(), "index") }
    } else {
        count = csxkNum(args[2])
        if (n != 0 && index < 0) { csxArgRange(csxMsgNonNeg(), "index") }
        if (n != 0 && count < 0) { csxArgRange(csxMsgNonNeg(), "count") }
        if (n == 0) { return 0 - 1 }
        if (index >= n) { csxArgRange(csxMsgIdxLess(), "index") }
        if (count > index + 1) { csxArgRange(csxMsgCount(), "count") }
    }
    return csxSearchBack(a, args[0], index, count)
}
// FindIndex(match) / (startIndex, match) / (startIndex, count, match).
function csxListFindIndex(a, args) {
    var n = a.length
    var pred = args[args.length - 1]
    var start = args.length > 1 ? csxkNum(args[0]) : 0
    if (start < 0 || start > n) { csxArgRange(csxMsgIdxLessEq(), "startIndex") }
    var count = args.length > 2 ? csxkNum(args[1]) : n - start
    if (count < 0 || start > n - count) { csxArgRange(csxMsgCount(), "count") }
    return csxFindIdx(a, start, count, 1, pred)
}
// FindLastIndex(match) / (startIndex, match) / (startIndex, count, match).
function csxListFindLastIndex(a, args) {
    var n = a.length
    var pred = args[args.length - 1]
    var start = args.length > 1 ? csxkNum(args[0]) : n - 1
    var count = args.length > 2 ? csxkNum(args[1]) : (args.length > 1 ? start + 1 : n)
    if (n == 0) {
        if (start != 0 - 1) { csxArgRange(csxMsgIdxLess(), "startIndex") }
    } else if (start < 0 || start >= n) {
        csxArgRange(csxMsgIdxLess(), "startIndex")
    }
    if (count < 0 || start - count + 1 < 0) { csxArgRange(csxMsgCount(), "count") }
    return csxFindIdx(a, start, count, 0 - 1, pred)
}
// BinarySearch([index, count,] item [, comparer]): the index of a match, or the
// bitwise complement of the insertion point. The probe order is .NET's, so which
// of several equal elements answers is .NET's too.
function csxBinarySearch(a, args) {
    var lo = 0
    var hi = a.length - 1
    var item = args[0]
    var cmp = args.length > 1 ? args[1] : null
    if (args.length > 2) {
        lo = csxkNum(args[0])
        var cnt = csxkNum(args[1])
        csxRangeChk(a.length, lo, cnt)
        hi = lo + cnt - 1
        item = args[2]
        cmp = args.length > 3 ? args[3] : null
    }
    while (lo <= hi) {
        var i = lo + Math.floor((hi - lo) / 2)
        var c = csxCmpWith(cmp, a[i], item)
        if (c == 0) { return i }
        if (c < 0) { lo = i + 1 } else { hi = i - 1 }
    }
    return 0 - lo - 1
}
function csxListMethod(a, name, args) {
    var n = args.length
    var i = 0
    // `__ctor` is the constructor call the compiler half emits (csxListCtor).
    if (name == "__ctor") { return csxListCtor(a, args) }
    if (name == "Add") {
        if (csxCapOn) { csxCapGrow(a, a.length + 1) }
        a.push(args[0])
        return undefined
    }
    if (name == "Count" && n == 0) { return a.length }
    if (name == "Contains" && n == 1) { return csxSearch(a, args[0], 0, a.length) >= 0 }
    if (name == "IndexOf") { return csxListIndexOf(a, args) }
    if (name == "LastIndexOf") { return csxListLastIndexOf(a, args) }
    if (name == "Clear") {
        while (a.length > 0) { a.pop() }
        return undefined
    }
    if (name == "AddRange") {
        var more = csxElemsOf(args[0])
        if (csxCapOn) { csxCapRange(a, args[0], more.length) }
        csxInsertAll(a, a.length, more)
        return undefined
    }
    if (name == "Insert" || name == "InsertRange") {
        var at = csxkNum(args[0])
        if (at < 0 || at > a.length) {
            if (name == "Insert") { csxArgRange("Index must be within the bounds of the List.", "index") }
            csxArgRange(csxMsgIdxLessEq(), "index")
        }
        var ins = name == "Insert" ? [args[1]] : csxElemsOf(args[1])
        if (csxCapOn) { csxCapRange(a, name == "Insert" ? ins : args[1], ins.length) }
        csxInsertAll(a, at, ins)
        return undefined
    }
    if (name == "Capacity" || name == "set_Capacity" || name == "TrimExcess" || name == "EnsureCapacity") {
        return csxCapMethod(a, name, args)
    }
    if (name == "Remove") {
        var ri = csxSearch(a, args[0], 0, a.length)
        if (ri < 0) { return false }
        csxRemoveAt(a, ri)
        return true
    }
    if (name == "RemoveAt") {
        var k = csxkNum(args[0])
        if (k < 0 || k >= a.length) { csxArgRange(csxMsgIdxLess(), "index") }
        csxRemoveAt(a, k)
        return undefined
    }
    if (name == "RemoveRange") {
        var rb = csxkNum(args[0])
        var rc = csxkNum(args[1])
        csxRangeChk(a.length, rb, rc)
        var kept = csxSlice(a, rb + rc, a.length)
        while (a.length > rb) { a.pop() }
        csxInsertAll(a, rb, kept)
        return undefined
    }
    if (name == "RemoveAll") {
        var keep = []
        var gone = 0
        i = 0
        while (i < a.length) {
            if (csxkCall(args[0], [a[i]]) === true) { gone = gone + 1 } else { keep.push(a[i]) }
            i = i + 1
        }
        while (a.length > 0) { a.pop() }
        csxInsertAll(a, 0, keep)
        return gone
    }
    if (name == "GetRange" || name == "Slice") {
        var gb = csxkNum(args[0])
        var gc = csxkNum(args[1])
        csxRangeChk(a.length, gb, gc)
        return csxTyped(csxSlice(a, gb, gb + gc), csxElemTy(a), true)
    }
    if (name == "ToArray") { return csxTyped(csxSlice(a, 0, a.length), csxElemTy(a), false) }
    if (name == "Reverse") {
        // An array has no Reverse() of its own: it is Enumerable.Reverse.
        if (n == 0 && !csxIsList(a)) { return csqMethod(a, name, args) }
        if (n == 0) {
            csxReverseRange(a, 0, a.length)
            return undefined
        }
        var vb = csxkNum(args[0])
        var vc = csxkNum(args[1])
        csxRangeChk(a.length, vb, vc)
        csxReverseRange(a, vb, vb + vc)
        return undefined
    }
    if (name == "Sort") {
        if (n == 3) {
            var sb = csxkNum(args[0])
            var sc = csxkNum(args[1])
            csxRangeChk(a.length, sb, sc)
            csxSortRange(a, sb, sb + sc, args[2])
            return undefined
        }
        csxSortRange(a, 0, a.length, n == 1 ? args[0] : null)
        return undefined
    }
    if (name == "BinarySearch") { return csxBinarySearch(a, args) }
    if (name == "Exists") { return csxFindIdx(a, 0, a.length, 1, args[0]) >= 0 }
    if (name == "FindIndex") { return csxListFindIndex(a, args) }
    if (name == "FindLastIndex") { return csxListFindLastIndex(a, args) }
    if (name == "Find" || name == "FindLast") {
        var fi = name == "Find" ? csxFindIdx(a, 0, a.length, 1, args[0])
                                : csxFindIdx(a, a.length - 1, a.length, 0 - 1, args[0])
        if (fi >= 0) { return a[fi] }
        return csxElemDefault(a)
    }
    if (name == "FindAll") {
        var hits = []
        i = 0
        while (i < a.length) {
            if (csxkCall(args[0], [a[i]]) === true) { hits.push(a[i]) }
            i = i + 1
        }
        return csxTyped(hits, csxElemTy(a), true)
    }
    if (name == "TrueForAll") {
        i = 0
        while (i < a.length) {
            if (csxkCall(args[0], [a[i]]) !== true) { return false }
            i = i + 1
        }
        return true
    }
    if (name == "ForEach") {
        i = 0
        while (i < a.length) {
            csxkCall(args[0], [a[i]])
            i = i + 1
        }
        return undefined
    }
    if (name == "ConvertAll") {
        var conv = []
        i = 0
        while (i < a.length) {
            conv.push(csxkCall(args[0], [a[i]]))
            i = i + 1
        }
        return csxTyped(conv, "", true)
    }
    if (name == "CopyTo") {
        var dst = args[0]
        var di = 0
        var si = 0
        var cn = a.length
        if (n == 2) { di = csxkNum(args[1]) }
        if (n == 4) {
            si = csxkNum(args[0])
            dst = args[1]
            di = csxkNum(args[2])
            cn = csxkNum(args[3])
            csxRangeChk(a.length, si, cn)
        }
        if (di < 0) { csxArgRange(csxMsgNonNeg(), "arrayIndex") }
        if (dst.length - di < cn) {
            csxkThrow("ArgumentException", "Destination array was not long enough. Check the destination index, length, and the array's lower bounds. (Parameter 'destinationArray')")
        }
        i = 0
        while (i < cn) {
            dst[di + i] = a[si + i]
            i = i + 1
        }
        return undefined
    }
    return csqMethod(a, name, args)
}

// ----- Dictionary<TKey,TValue> -----

// KeyValuePair<TKey,TValue>: Key and Value, and Item1/Item2 under __tuple so a
// `foreach (var (k, v) in d)` deconstructs it like a tuple. It renders as
// "[key, value]".
function csxKvp(k, v) {
    var o = {}
    o["__kvp"] = true
    o["__tuple"] = 2
    o["Item1"] = k
    o["Item2"] = v
    o["Key"] = k
    o["Value"] = v
    return o
}
function csxIsKvp(v) {
    if (csxIsNull(v) || typeof v != "object" || Array.isArray(v)) { return false }
    return v["__kvp"] === true
}
function csxDictPairs(d) {
    var ks = d["keys"]
    var vs = d["vals"]
    var out = []
    var i = 0
    while (i < ks.length) {
        out.push(csxKvp(ks[i], vs[i]))
        i = i + 1
    }
    return out
}
function csxKeyChk(k) {
    if (csxIsNull(k)) { csxkThrow("ArgumentNullException", "Value cannot be null. (Parameter 'key')") }
}
function csxDictDel(d, i) {
    csxRemoveAt(d["keys"], i)
    csxRemoveAt(d["vals"], i)
}
// default(TValue): the declared value type's when the constructor recorded it.
function csxDictDefault(d) {
    var t = d["__vty"]
    if (typeof t == "string") { return csxDefaultOf(t) }
    var vs = d["vals"]
    return vs.length > 0 ? csxZeroLike(vs[0]) : null
}
function csxDictMethod(d, name, args) {
    var n = args.length
    var ks = d["keys"]
    var vs = d["vals"]
    if (name == "__ctor") { return csxDictCtor(d, args) }
    if (name == "Keys") { return csxSlice(ks, 0, ks.length) }
    if (name == "Values") { return csxSlice(vs, 0, vs.length) }
    if (name == "Count" && n == 0) { return ks.length }
    if (name == "Add" || name == "TryAdd") {
        csxKeyChk(args[0])
        if (csxkDictFind(d, args[0]) >= 0) {
            if (name == "TryAdd") { return false }
            csxkThrow("ArgumentException", "An item with the same key has already been added. Key: " + csxkStr(args[0]))
        }
        ks.push(args[0])
        vs.push(args[1])
        return name == "TryAdd" ? true : undefined
    }
    if (name == "ContainsKey") {
        csxKeyChk(args[0])
        return csxkDictFind(d, args[0]) >= 0
    }
    if (name == "ContainsValue") { return csxSearch(vs, args[0], 0, vs.length) >= 0 }
    if (name == "Clear") {
        while (ks.length > 0) {
            ks.pop()
            vs.pop()
        }
        return undefined
    }
    if (name == "Remove" || name == "TryGetValue") {
        csxKeyChk(args[0])
        var i = csxkDictFind(d, args[0])
        var got = anytype
        got = i >= 0 ? vs[i] : csxDictDefault(d)
        if (n > 1) { csxkRefSet(args[1], got) }
        if (i >= 0 && name == "Remove") { csxDictDel(d, i) }
        return i >= 0
    }
    if (name == "GetValueOrDefault") {
        csxKeyChk(args[0])
        var gi = csxkDictFind(d, args[0])
        if (gi >= 0) { return vs[gi] }
        return n > 1 ? args[1] : csxDictDefault(d)
    }
    return csqMethod(d, name, args)
}
// `new List<T>(collection)` copies it and `new List<T>(capacity)` is empty;
// the same for a Dictionary built from another one.
function csxListCtor(a, args) {
    if (args.length == 0) { return a }
    var src = args[0]
    if (csxkIsNum(src)) {
        if (csxkNum(src) < 0) { csxArgRange(csxMsgNonNeg(), "capacity") }
        if (csxCapOn) { csxCapSet(a, csxkNum(src)) }
        return a
    }
    // A collection whose size is known sets the capacity to it.
    var els = csxElemsOf(src)
    csxInsertAll(a, 0, els)
    if (csxCapOn) {
        if (Array.isArray(src) || csxIsDictLike(src)) { csxCapSet(a, els.length) } else { csxCapAdds(a, els.length) }
    }
    return a
}
function csxDictCtor(d, args) {
    if (args.length == 0) { return d }
    var src = args[0]
    if (csxkIsNum(src)) {
        if (csxkNum(src) < 0) { csxArgRange(csxMsgNonNeg(), "capacity") }
        return d
    }
    var ps = csxElemsOf(src)
    var i = 0
    while (i < ps.length) {
        csxDictMethod(d, "Add", [ps[i]["Key"], ps[i]["Value"]])
        i = i + 1
    }
    return d
}
