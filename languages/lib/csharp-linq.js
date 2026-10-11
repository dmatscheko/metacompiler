// csharp-linq.js -- System.Linq.Enumerable's operators, for csharp-interpreter.abnf
// (include) and languages/lib/csharp-rt.metajs (import). abnf/jsrtcslinq.go is the
// Go twin's port; the three must answer alike. It sits on lib/csharp-strings.js's
// and lib/csharp-collections.js's hooks and adds these:
//   csxkIsGen(v)       is v a user iterator (a method with 'yield return')
//   csxkGenNext(it)    step it.g, a user iterator: false at its end, else true with
//                      the value in it.cur
//   csxkArity(f)       2 for a lambda declaring two parameters, else 0 (the
//                      overloads whose delegate also takes the element's index)
//   csxkNumKind(v)     "i" an int, "l" a sized integer, "d" a double, "f" a float
//   csxkFlo(x)         the double x; csxkFlo32(x) the float x
//   csxkAdd(a, b)      C#'s + on two sized integers
//
// A query operator answers a QUERY, {__linq: operator, src, ...}, and runs nothing:
// the source is read when the query is enumerated (deferred execution), so a query
// over a List sees the elements the List holds at that time, and enumerating it
// twice reads the source twice. Enumeration is a pull iterator (csqOpen / csqNext),
// so Take, First, TakeWhile and Any stop reading an endless iterator early. The
// operators that must see the whole source (OrderBy, GroupBy, Reverse, the set
// operators' second sequence) read it when enumeration starts. A foreach, a
// List constructor, string.Join and the other non-LINQ consumers read a query
// whole first (csqToArray), which is the same answer for side-effect-free
// delegates but never terminates on an endless one.
//
// Equality is EqualityComparer<T>.Default (csxEq), order Comparer<T>.Default
// (csxCmp: a string's default order is the culture's collation and fails; pass
// StringComparer.Ordinal). OrderBy is stable, as documented. A delegate is
// called with the element's index too exactly when it declares two parameters
// (csxkArity), which is how C# picks Where((x, i) => ...) over Where(x => ...).
//
// default(T) (FirstOrDefault and the other ...OrDefault members, DefaultIfEmpty)
// is the source's element type's when the source carries one (csqElemTy: a typed
// collection, through the operators that keep its elements), else the default of
// the kind of an element seen, else null. Not modelled: MinBy / MaxBy of an empty
// sequence, OfType<T>, the IEqualityComparer overloads other than
// StringComparer's, ToLookup, and a second enumeration of a user iterator (it
// continues where the first stopped). An iterator body that is itself driven by
// replay (the interpreter's model) re-reads its sources at every step.

function csqMsgEmpty() { return "Sequence contains no elements" }
function csqNoMatch() { csxkThrow("InvalidOperationException", "Sequence contains no matching element") }
function csqIsQuery(v) {
    if (v === null || v === undefined || typeof v != "object" || Array.isArray(v)) { return false }
    return typeof v["__linq"] == "string"
}
// A value LINQ can enumerate that is not already an array.
function csqIsLazy(v) { return csqIsQuery(v) || csxkIsGen(v) }
function csqMake(op, src) {
    var q = {}
    q["__linq"] = op
    q["src"] = src
    return q
}
function csqSrcChk(v, param) {
    if (csxIsNull(v)) { csxkThrow("ArgumentNullException", "Value cannot be null. (Parameter '" + param + "')") }
}
// A delegate called on an element and its index.
function csqCall(f, x, i) {
    if (csxkArity(f) == 2) { return csxkCall(f, [x, i]) }
    return csxkCall(f, [x])
}
function csqTrue(v) { return v === true }

// ----- enumeration -----

function csqArrIt(a) {
    var it = {}
    it["k"] = "a"
    it["a"] = a
    it["i"] = 0
    it["cur"] = undefined
    return it
}
// An iterator over any enumerable value: an array (List<T> and T[]), a string's
// chars, a dictionary's KeyValuePairs, a query, a user iterator.
function csqOpen(v) {
    if (Array.isArray(v)) { return csqArrIt(v) }
    if (csqIsQuery(v)) { return csqOpenQuery(v) }
    if (csxkIsGen(v)) {
        var it = {}
        it["k"] = "g"
        it["g"] = v
        it["cur"] = undefined
        return it
    }
    csqSrcChk(v, "source")
    if (typeof v == "string" || csxIsDictLike(v)) { return csqArrIt(csxkElems(v)) }
    fail("a LINQ source that is not a collection, a string, a dictionary or an iterator is not modelled")
    return csqArrIt([])
}
function csxIsDictLike(v) { return typeof v == "object" && v["__dict"] !== undefined && !Array.isArray(v) }
function csqToArray(v) {
    var it = csqOpen(v)
    var out = []
    while (csqNext(it)) { out.push(it["cur"]) }
    return out
}
// The elements of a source as an array that may be the source itself.
function csqElems(v) { return Array.isArray(v) ? v : csqToArray(v) }

function csqOpenQuery(q) {
    var op = q["__linq"]
    var it = {}
    it["k"] = op
    it["q"] = q
    it["cur"] = undefined
    it["i"] = 0
    if (op == "range" || op == "repeat") { return it }
    if (op == "grp") { return csqArrIt(q["src"]) }
    if (op == "reverse") {
        var r = csqToArray(q["src"])
        csxReverseRange(r, 0, r.length)
        return csqArrIt(r)
    }
    if (op == "order") { return csqArrIt(csqSorted(q)) }
    if (op == "group") { return csqArrIt(csqGroups(q)) }
    if (op == "takelast" || op == "skiplast") {
        var all = csqToArray(q["src"])
        var n = q["n"]
        if (n < 0) { n = 0 }
        if (n > all.length) { n = all.length }
        if (op == "takelast") { return csqArrIt(csxSlice(all, all.length - n, all.length)) }
        return csqArrIt(csxSlice(all, 0, all.length - n))
    }
    it["s"] = csqOpen(q["src"])
    if (op == "union" || op == "intersect" || op == "except" || op == "distinct") { it["seen"] = [] }
    if (op == "intersect" || op == "except") { it["b"] = csqToArray(q["b"]) }
    if (op == "concat" || op == "union" || op == "zip") { it["t"] = csqOpen(q["b"]) }
    return it
}
// Is x in the list under the comparer (null: EqualityComparer<T>.Default)?
function csqHas(list, x, cmp) {
    var i = 0
    while (i < list.length) {
        if (csqEq(list[i], x, cmp)) { return true }
        i = i + 1
    }
    return false
}
function csqEq(a, b, cmp) {
    if (csxIsNull(cmp)) { return csxEq(a, b) }
    if (csxIsStrComparer(cmp)) { return csxEqStr(cmp["__csstrcmp"], a, b) }
    fail("an IEqualityComparer other than StringComparer's is not modelled")
    return false
}
// One step of an iterator: false at the end, else true with the value in it.cur.
function csqNext(it) {
    var k = it["k"]
    var x = anytype
    if (k == "a") {
        var a = it["a"]
        var ai = it["i"]
        if (ai >= a.length) { return false }
        it["cur"] = a[ai]
        it["i"] = ai + 1
        return true
    }
    if (k == "g") { return csxkGenNext(it) }
    var q = it["q"]
    var s = it["s"]
    if (k == "where") {
        while (csqNext(s)) {
            x = s["cur"]
            var wi = it["i"]
            it["i"] = wi + 1
            if (csqTrue(csqCall(q["f"], x, wi))) {
                it["cur"] = x
                return true
            }
        }
        return false
    }
    if (k == "select") {
        if (!csqNext(s)) { return false }
        var si = it["i"]
        it["i"] = si + 1
        it["cur"] = csqCall(q["f"], s["cur"], si)
        return true
    }
    if (k == "cast") {
        if (!csqNext(s)) { return false }
        it["cur"] = s["cur"]
        return true
    }
    if (k == "selmany") {
        while (true) {
            if (it["t"] !== undefined) {
                var inner = it["t"]
                if (csqNext(inner)) {
                    var g = q["g"]
                    it["cur"] = csxIsNull(g) ? inner["cur"] : csxkCall(g, [it["x"], inner["cur"]])
                    return true
                }
                it["t"] = undefined
            }
            if (!csqNext(s)) { return false }
            var mi = it["i"]
            it["i"] = mi + 1
            it["x"] = s["cur"]
            it["t"] = csqOpen(csqCall(q["f"], s["cur"], mi))
        }
    }
    if (k == "take") {
        if (it["i"] >= q["n"] || !csqNext(s)) { return false }
        it["i"] = it["i"] + 1
        it["cur"] = s["cur"]
        return true
    }
    if (k == "skip") {
        while (it["i"] < q["n"]) {
            if (!csqNext(s)) { return false }
            it["i"] = it["i"] + 1
        }
        if (!csqNext(s)) { return false }
        it["cur"] = s["cur"]
        return true
    }
    if (k == "takew") {
        if (it["done"] === true || !csqNext(s)) { return false }
        var ti = it["i"]
        it["i"] = ti + 1
        if (!csqTrue(csqCall(q["f"], s["cur"], ti))) {
            it["done"] = true
            return false
        }
        it["cur"] = s["cur"]
        return true
    }
    if (k == "skipw") {
        while (it["done"] !== true) {
            if (!csqNext(s)) { return false }
            var ki = it["i"]
            it["i"] = ki + 1
            if (!csqTrue(csqCall(q["f"], s["cur"], ki))) {
                it["done"] = true
                it["cur"] = s["cur"]
                return true
            }
        }
        if (!csqNext(s)) { return false }
        it["cur"] = s["cur"]
        return true
    }
    if (k == "concat" || k == "union") {
        while (true) {
            var src = s
            if (it["done"] === true) { src = it["t"] }
            if (!csqNext(src)) {
                if (it["done"] === true) { return false }
                it["done"] = true
                continue
            }
            x = src["cur"]
            if (k == "concat") {
                it["cur"] = x
                return true
            }
            if (!csqHas(it["seen"], x, q["c"])) {
                it["seen"].push(x)
                it["cur"] = x
                return true
            }
        }
    }
    if (k == "distinct" || k == "intersect" || k == "except") {
        while (csqNext(s)) {
            x = s["cur"]
            var key = anytype
            key = csxIsNull(q["f"]) ? x : csxkCall(q["f"], [x])
            if (csqHas(it["seen"], key, q["c"])) { continue }
            if (k == "intersect" && !csqHas(it["b"], key, q["c"])) { continue }
            if (k == "except" && csqHas(it["b"], key, q["c"])) { continue }
            it["seen"].push(key)
            it["cur"] = x
            return true
        }
        return false
    }
    if (k == "append" || k == "prepend") {
        var at = it["i"]
        it["i"] = at + 1
        if (k == "prepend" && at == 0) {
            it["cur"] = q["x"]
            return true
        }
        if (csqNext(s)) {
            it["cur"] = s["cur"]
            return true
        }
        if (k == "append" && it["done"] !== true) {
            it["done"] = true
            it["cur"] = q["x"]
            return true
        }
        return false
    }
    if (k == "defempty") {
        if (csqNext(s)) {
            it["i"] = 1
            it["cur"] = s["cur"]
            return true
        }
        if (it["i"] == 0) {
            it["i"] = 1
            it["cur"] = q["has"] ? q["x"] : csqDefault(q["src"], false, null)
            return true
        }
        return false
    }
    if (k == "zip") {
        var t = it["t"]
        if (!csqNext(s) || !csqNext(t)) { return false }
        var zf = q["f"]
        if (csxIsNull(zf)) {
            var tup = {}
            tup["__tuple"] = 2
            tup["Item1"] = s["cur"]
            tup["Item2"] = t["cur"]
            it["cur"] = tup
        } else {
            it["cur"] = csxkCall(zf, [s["cur"], t["cur"]])
        }
        return true
    }
    if (k == "range") {
        if (it["i"] >= q["n"]) { return false }
        it["cur"] = q["x"] + it["i"]
        it["i"] = it["i"] + 1
        return true
    }
    if (k == "repeat") {
        if (it["i"] >= q["n"]) { return false }
        it["cur"] = q["x"]
        it["i"] = it["i"] + 1
        return true
    }
    fail("unknown query operator: " + k)
    return false
}

// ----- the buffering operators -----

// OrderBy / ThenBy: a stable merge sort on the keys, computed once per element.
function csqSorted(q) {
    var elems = csqToArray(q["src"])
    var specs = q["keys"]
    var keys = []
    var i = 0
    while (i < elems.length) {
        var row = []
        var j = 0
        while (j < specs.length) {
            var f = anytype
            f = specs[j]["f"]
            row.push(csxIsNull(f) ? elems[i] : csxkCall(f, [elems[i]]))
            j = j + 1
        }
        keys.push(row)
        i = i + 1
    }
    var idx = []
    i = 0
    while (i < elems.length) {
        idx.push(i)
        i = i + 1
    }
    var width = 1
    var n = idx.length
    while (width < n) {
        var dst = []
        var lo = 0
        while (lo < n) {
            var mid = lo + width < n ? lo + width : n
            var hi = lo + 2 * width < n ? lo + 2 * width : n
            var x = lo
            var y = mid
            while (x < mid && y < hi) {
                if (csqKeyCmp(specs, keys[idx[y]], keys[idx[x]]) < 0) {
                    dst.push(idx[y])
                    y = y + 1
                } else {
                    dst.push(idx[x])
                    x = x + 1
                }
            }
            while (x < mid) {
                dst.push(idx[x])
                x = x + 1
            }
            while (y < hi) {
                dst.push(idx[y])
                y = y + 1
            }
            lo = hi
        }
        idx = dst
        width = width * 2
    }
    var out = []
    i = 0
    while (i < n) {
        out.push(elems[idx[i]])
        i = i + 1
    }
    return out
}
function csqKeyCmp(specs, a, b) {
    var j = 0
    while (j < specs.length) {
        var c = csxCmpWith(specs[j]["c"], a[j], b[j])
        if (c != 0) { return specs[j]["desc"] ? 0 - c : c }
        j = j + 1
    }
    return 0
}
// A grouping: its Key, enumerable as its elements.
function csqGrouping(key, elems) {
    var g = csqMake("grp", elems)
    g["Key"] = key
    return g
}
// GroupBy: the groups in the order their keys first occur, each group's elements
// in source order.
function csqGroups(q) {
    var elems = csqToArray(q["src"])
    var gkeys = []
    var gvals = []
    var i = 0
    while (i < elems.length) {
        var x = anytype
        x = elems[i]
        var key = anytype
        key = csxkCall(q["f"], [x])
        var v = anytype
        v = csxIsNull(q["g"]) ? x : csxkCall(q["g"], [x])
        var j = 0
        while (j < gkeys.length && !csxEq(gkeys[j], key)) { j = j + 1 }
        if (j == gkeys.length) {
            gkeys.push(key)
            gvals.push([])
        }
        gvals[j].push(v)
        i = i + 1
    }
    var out = []
    i = 0
    while (i < gkeys.length) {
        var grp = csqGrouping(gkeys[i], gvals[i])
        var r = q["r"]
        if (csxIsNull(r)) { out.push(grp) } else { out.push(csxkCall(r, [gkeys[i], grp])) }
        i = i + 1
    }
    return out
}
function csqOrder(src, f, desc, cmp, prev) {
    var q = csqMake("order", src)
    var keys = []
    if (prev !== null) {
        var i = 0
        while (i < prev.length) {
            keys.push(prev[i])
            i = i + 1
        }
    }
    var spec = {}
    spec["f"] = f
    spec["desc"] = desc
    spec["c"] = cmp
    keys.push(spec)
    q["keys"] = keys
    return q
}

// ----- the aggregates -----

// Sum: an int sum is checked (OverflowException), a float sum is accumulated in
// double and rounded once, as Enumerable.Sum does.
function csqSum(it) {
    var kind = ""
    var acc = 0
    var lacc = anytype
    lacc = 0
    while (csqNext(it)) {
        var x = anytype
        x = it["cur"]
        if (csxIsNull(x)) { continue }
        var nk = csxkNumKind(x)
        if (kind == "") { kind = nk }
        if (nk == "l") {
            lacc = csxkAdd(lacc, x)
        } else {
            acc = acc + csxkDbl(x)
            if (nk == "i" && (acc > 2147483647 || acc < 0 - 2147483648)) {
                csxkThrow("OverflowException", "Arithmetic operation resulted in an overflow.")
            }
        }
    }
    if (kind == "l") { return lacc }
    if (kind == "d") { return csxkFlo(acc) }
    if (kind == "f") { return csxkFlo32(acc) }
    return acc
}
function csqAverage(it) {
    var kind = ""
    var acc = 0
    var n = 0
    while (csqNext(it)) {
        var x = anytype
        x = it["cur"]
        if (csxIsNull(x)) { continue }
        if (kind == "") { kind = csxkNumKind(x) }
        acc = acc + csxkDbl(x)
        n = n + 1
    }
    if (n == 0) { csxkThrow("InvalidOperationException", csqMsgEmpty()) }
    if (kind == "f") { return csxkFlo32(acc / n) }
    return csxkFlo(acc / n)
}
// The element type of a source as far as it is known: a typed collection's, a
// string's char, Range's int, and through every operator that keeps the
// elements; "" otherwise.
function csqElemTy(v) {
    if (Array.isArray(v)) { return csxElemTy(v) }
    if (typeof v == "string") { return "char" }
    if (!csqIsQuery(v)) { return "" }
    var op = v["__linq"]
    if (op == "range") { return "int" }
    if (op == "select" || op == "selmany" || op == "zip" || op == "group" || op == "grp" || op == "repeat") { return "" }
    return csqElemTy(v["src"])
}
// default(T) for the elements of src: by T when known, else the default of the
// kind of an element seen, else null.
function csqDefault(src, seen, any) { return csxDefaultFor(csqElemTy(src), seen, any) }
// Min / Max by Comparer<T>.Default; the first of equal elements wins. An empty
// sequence of a reference or nullable type answers null (src the source, null
// when the elements were selected).
function csqMinMax(it, sign, src) {
    var has = false
    var best = anytype
    best = undefined
    while (csqNext(it)) {
        var x = anytype
        x = it["cur"]
        if (!has || csxCmp(x, best) * sign > 0) { best = x }
        has = true
    }
    if (!has) {
        var ety = src === null ? "" : csqElemTy(src)
        if (ety != "" && csxDefaultFor(ety, false, null) === null) { return null }
        csxkThrow("InvalidOperationException", csqMsgEmpty())
    }
    return best
}
// MinBy / MaxBy: the first element whose key is least / greatest.
function csqMinMaxBy(it, f, sign) {
    var has = false
    var best = anytype
    var bk = anytype
    best = undefined
    bk = undefined
    while (csqNext(it)) {
        var x = anytype
        x = it["cur"]
        var key = anytype
        key = csxkCall(f, [x])
        if (!has || csxCmp(key, bk) * sign > 0) {
            best = x
            bk = key
        }
        has = true
    }
    if (!has) { fail("MinBy / MaxBy of an empty sequence needs the element type, which is not modelled") }
    return best
}
// First / Last / Single, with or without a predicate, throwing or answering a
// default: which is 0 first, 1 last, 2 single; dflt is the default to answer, or
// undefined to throw.
function csqPick(it, pred, which, orDefault, dflt, src) {
    var has = false
    var got = anytype
    var any = anytype
    got = undefined
    any = undefined
    var seen = false
    while (csqNext(it)) {
        var x = anytype
        x = it["cur"]
        if (!seen) {
            any = x
            seen = true
        }
        if (!csxIsNull(pred) && !csqTrue(csxkCall(pred, [x]))) { continue }
        if (has && which == 2) {
            csxkThrow("InvalidOperationException", csxIsNull(pred) ? "Sequence contains more than one element"
                                                                 : "Sequence contains more than one matching element")
        }
        got = x
        has = true
        if (which == 0) { return got }
    }
    if (has) { return got }
    if (orDefault) {
        if (dflt !== undefined) { return dflt }
        return csqDefault(src, seen, any)
    }
    if (csxIsNull(pred)) { csxkThrow("InvalidOperationException", csqMsgEmpty()) }
    csqNoMatch()
    return undefined
}
function csqAggregate(it, args) {
    var acc = anytype
    var f = anytype
    f = args[0]
    if (args.length > 1) {
        acc = args[0]
        f = args[1]
    } else {
        if (!csqNext(it)) { csxkThrow("InvalidOperationException", csqMsgEmpty()) }
        acc = it["cur"]
    }
    while (csqNext(it)) { acc = csxkCall(f, [acc, it["cur"]]) }
    if (args.length > 2) { return csxkCall(args[2], [acc]) }
    return acc
}
function csqToDict(it, args) {
    var d = {__dict: true, keys: [], vals: []}
    var vf = args.length > 1 && csxkIsDeleg(args[1]) ? args[1] : null
    if (args.length > 1 && !csxIsNull(args[args.length - 1]) && !csxkIsDeleg(args[args.length - 1])) {
        fail("a Dictionary key comparer is not modelled")
    }
    while (csqNext(it)) {
        var x = anytype
        x = it["cur"]
        var k = anytype
        k = csxkCall(args[0], [x])
        csxDictMethod(d, "Add", [k, vf === null ? x : csxkCall(vf, [x])])
    }
    return d
}

// ----- the operator dispatch -----

// The LINQ operator `name` on the enumerable recv. A List<T>, a string and a
// Dictionary come here only for the names their own members do not take.
function csqMethod(recv, name, args) {
    var n = args.length
    if (csqIsQuery(recv) && recv["__linq"] == "order" && (name == "ThenBy" || name == "ThenByDescending")) {
        return csqOrder(recv["src"], args[0], name == "ThenByDescending", n > 1 ? args[1] : null, recv["keys"])
    }
    var q = anytype
    if (name == "Where" || name == "Select" || name == "TakeWhile" || name == "SkipWhile") {
        csqSrcChk(args[0], name == "Select" ? "selector" : "predicate")
        var op = name == "Where" ? "where" : (name == "Select" ? "select" : (name == "TakeWhile" ? "takew" : "skipw"))
        q = csqMake(op, recv)
        q["f"] = args[0]
        return q
    }
    if (name == "SelectMany") {
        q = csqMake("selmany", recv)
        q["f"] = args[0]
        q["g"] = n > 1 ? args[1] : null
        return q
    }
    if (name == "Take" || name == "Skip" || name == "TakeLast" || name == "SkipLast") {
        q = csqMake(name == "Take" ? "take" : (name == "Skip" ? "skip" : (name == "TakeLast" ? "takelast" : "skiplast")), recv)
        q["n"] = csxkNum(args[0])
        return q
    }
    if (name == "Concat" || name == "Union" || name == "Intersect" || name == "Except" || name == "Zip") {
        csqSrcChk(args[0], name == "Zip" ? "second" : "second")
        q = csqMake(name == "Concat" ? "concat" : (name == "Union" ? "union" : (name == "Intersect" ? "intersect" : (name == "Except" ? "except" : "zip"))), recv)
        q["b"] = args[0]
        q["c"] = null
        q["f"] = null
        if (n > 1) {
            if (name == "Zip") { q["f"] = args[1] } else { q["c"] = args[1] }
        }
        return q
    }
    if (name == "Distinct" || name == "DistinctBy") {
        q = csqMake("distinct", recv)
        q["f"] = name == "DistinctBy" ? args[0] : null
        q["c"] = name == "DistinctBy" ? (n > 1 ? args[1] : null) : (n > 0 ? args[0] : null)
        return q
    }
    if (name == "Append" || name == "Prepend" || name == "DefaultIfEmpty") {
        q = csqMake(name == "Append" ? "append" : (name == "Prepend" ? "prepend" : "defempty"), recv)
        q["x"] = n > 0 ? args[0] : null
        q["has"] = n > 0
        return q
    }
    if (name == "Reverse") { return csqMake("reverse", recv) }
    if (name == "Cast" || name == "AsEnumerable") { return csqMake("cast", recv) }
    if (name == "OrderBy" || name == "OrderByDescending") {
        return csqOrder(recv, args[0], name == "OrderByDescending", n > 1 ? args[1] : null, null)
    }
    if (name == "Order" || name == "OrderDescending") {
        return csqOrder(recv, null, name == "OrderDescending", n > 0 ? args[0] : null, null)
    }
    if (name == "GroupBy") {
        q = csqMake("group", recv)
        q["f"] = args[0]
        q["g"] = null
        q["r"] = null
        if (n == 2 && csxkArity(args[1]) == 2) { q["r"] = args[1] } else if (n > 1) { q["g"] = args[1] }
        if (n > 2) { q["r"] = args[2] }
        return q
    }
    // The immediate operators: they enumerate now.
    var it = csqOpen(recv)
    if (name == "ToList" || name == "ToArray") {
        var out = []
        while (csqNext(it)) { out.push(it["cur"]) }
        return csxTyped(out, csqElemTy(recv), name == "ToList")
    }
    if (name == "ToDictionary") { return csqToDict(it, args) }
    if (name == "Count" || name == "LongCount") {
        var c = 0
        while (csqNext(it)) {
            if (n == 0 || csqTrue(csxkCall(args[0], [it["cur"]]))) { c = c + 1 }
        }
        return c
    }
    if (name == "Any") {
        while (csqNext(it)) {
            if (n == 0 || csqTrue(csxkCall(args[0], [it["cur"]]))) { return true }
        }
        return false
    }
    if (name == "All") {
        while (csqNext(it)) {
            if (!csqTrue(csxkCall(args[0], [it["cur"]]))) { return false }
        }
        return true
    }
    if (name == "Contains") {
        while (csqNext(it)) {
            if (csqEq(it["cur"], args[0], n > 1 ? args[1] : null)) { return true }
        }
        return false
    }
    if (name == "Sum" || name == "Average" || name == "Min" || name == "Max") {
        if (n > 0) {
            var sq = csqMake("select", recv)
            sq["f"] = args[0]
            it = csqOpen(sq)
        }
        if (name == "Sum") { return csqSum(it) }
        if (name == "Average") { return csqAverage(it) }
        return csqMinMax(it, name == "Max" ? 1 : 0 - 1, n > 0 ? null : recv)
    }
    if (name == "MinBy" || name == "MaxBy") { return csqMinMaxBy(it, args[0], name == "MaxBy" ? 1 : 0 - 1) }
    if (name == "First" || name == "FirstOrDefault" || name == "Last" || name == "LastOrDefault"
        || name == "Single" || name == "SingleOrDefault") {
        var which = name.charAt(0) == "F" ? 0 : (name.charAt(0) == "L" ? 1 : 2)
        var orDef = name.length > 8 && name.substring(name.length - 9, name.length) == "OrDefault"
        var pred = anytype
        var dflt = anytype
        pred = null
        dflt = undefined
        if (n > 0 && csxkIsDeleg(args[0])) { pred = args[0] } else if (n > 0) { dflt = args[0] }
        if (n > 1) { dflt = args[1] }
        return csqPick(it, pred, which, orDef, dflt, recv)
    }
    if (name == "ElementAt" || name == "ElementAtOrDefault") {
        var at = csxkNum(args[0])
        var ei = 0
        var any = anytype
        any = null
        while (at >= 0 && csqNext(it)) {
            if (ei == at) { return it["cur"] }
            any = it["cur"]
            ei = ei + 1
        }
        if (name == "ElementAt") { csxArgRange(csxMsgIdxLess(), "index") }
        return csqDefault(recv, ei > 0, any)
    }
    if (name == "Aggregate") { return csqAggregate(it, args) }
    if (name == "SequenceEqual") {
        var ot = csqOpen(args[0])
        while (true) {
            var more = csqNext(it)
            if (more != csqNext(ot)) { return false }
            if (!more) { return true }
            if (!csqEq(it["cur"], ot["cur"], n > 1 ? args[1] : null)) { return false }
        }
    }
    if (name == "GetEnumerator") { fail("GetEnumerator of a LINQ query is not modelled") }
    fail("unknown method: " + name)
    return undefined
}
// Enumerable's static members: Range, Repeat, Empty, and any operator called
// in its static form (Enumerable.Where(xs, p)).
function csqStatic(name, args) {
    var q = anytype
    if (name == "Range") {
        var start = csxkNum(args[0])
        var count = csxkNum(args[1])
        if (count < 0 || start + count - 1 > 2147483647) { csxArgRange("Specified argument was out of the range of valid values.", "count") }
        q = csqMake("range", null)
        q["x"] = start
        q["n"] = count
        return q
    }
    if (name == "Repeat") {
        var rc = csxkNum(args[1])
        if (rc < 0) { csxArgRange("Specified argument was out of the range of valid values.", "count") }
        q = csqMake("repeat", null)
        q["x"] = args[0]
        q["n"] = rc
        return q
    }
    if (name == "Empty") { return csqMake("cast", []) }
    var rest = []
    var i = 1
    while (i < args.length) {
        rest.push(args[i])
        i = i + 1
    }
    return csqMethod(args[0], name, rest)
}
