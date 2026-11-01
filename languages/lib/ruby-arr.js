// ruby-arr.js -- Array index arithmetic of MRI's array.c, for ruby-interpreter.abnf
// (include) and languages/lib/ruby-rt.metajs (import). Go twins in
// abnf/jsrtrubystr.go. Written in the subset both engines accept.

// Enumerator::Lazy, and the endless Enumerator of a blockless Array#cycle or of
// each_slice / each_cons on an endless range: an Enumerator object that also
// carries __rlazy, a SOURCE - kind "arr" (items), "cycle" (items, forever) or
// "range" (lo, lo+1, ... up to hi; hi null is endless) or "each" (the #each of
// a user Enumerable, src) - and the OPS map / select / reject / take_while /
// drop_while / take / slice / cons, {k, f, n}, applied to one value at a time.
// rbaLazyFirst pulls the first n results (all of them when n < 0); call(f, v)
// runs a block, test(v) is Ruby truthiness. A take op ends the run once its count
// has passed it, so nothing past it is evaluated. An "each" source is PUSHED:
// drive(src, step) runs its #each, calls step(v) per value and leaves #each once
// step answers true. Go twin: rubyLazyFirst in abnf/jsrtrubystr.go.
function rbaLazyFirst(lz, n, call, test, drive) {
    var ops = lz.ops
    var st = {out: [], n: n, cnt: [], on: [], buf: [], done: n == 0, full: false, stopAt: 0 - 1}
    var j = 0
    while (j < ops.length) {
        st.cnt.push(0)
        st.on.push(false)
        st.buf.push([])
        j = j + 1
    }
    if (lz.kind == "each" && !st.done) {
        drive(lz.src, function(v) {
            rbaLazyRun(st, ops, v, 0, call, test)
            return st.done
        })
    }
    var i = 0
    while (lz.kind != "each" && !st.done) {
        var v = anytype
        if (lz.kind == "range") {
            v = lz.lo + i
            if (lz.hi !== null && (lz.excl ? v >= lz.hi : v > lz.hi)) { break }
        } else {
            var len = lz.items.length
            if (len == 0 || (lz.kind == "arr" && i >= len)) { break }
            v = lz.items[i % len]
        }
        i = i + 1
        rbaLazyRun(st, ops, v, 0, call, test)
    }
    // A slice op still holding values hands them on as a last, short slice when
    // the run ended above it: the source ran out, or a take / take_while before
    // it stopped.
    j = 0
    while (j < ops.length && !st.full) {
        if (ops[j].k == "slice" && st.buf[j].length > 0 && j > st.stopAt) {
            var rest = st.buf[j]
            st.buf[j] = []
            rbaLazyRun(st, ops, rest, j + 1, call, test)
        }
        j = j + 1
    }
    return st.out
}
// One value through ops[from..], into st.out unless an op holds it back.
function rbaLazyRun(st, ops, v, from, call, test) {
    var cur = anytype
    cur = v
    var j = from
    while (j < ops.length) {
        var op = ops[j]
        if (op.k == "map") {
            cur = call(op.f, cur)
        } else if (op.k == "select") {
            if (!test(call(op.f, cur))) { return }
        } else if (op.k == "reject") {
            if (test(call(op.f, cur))) { return }
        } else if (op.k == "take_while") {
            if (!test(call(op.f, cur))) {
                rbaLazyStop(st, j)
                return
            }
        } else if (op.k == "drop_while") {
            if (!st.on[j]) {
                if (test(call(op.f, cur))) { return }
                st.on[j] = true
            }
        } else if (op.k == "take") {
            if (op.n <= 0) {
                rbaLazyStop(st, j)
                return
            }
            st.cnt[j] = st.cnt[j] + 1
            if (st.cnt[j] >= op.n) { rbaLazyStop(st, j) }
        } else if (op.k == "slice") {
            st.buf[j].push(cur)
            if (st.buf[j].length < op.n) { return }
            cur = st.buf[j]
            st.buf[j] = []
        } else if (op.k == "cons") {
            st.buf[j].push(cur)
            if (st.buf[j].length > op.n) { st.buf[j] = st.buf[j].slice(1) }
            if (st.buf[j].length < op.n) { return }
            cur = st.buf[j].slice(0)
        }
        j = j + 1
    }
    st.out.push(cur)
    if (st.n >= 0 && st.out.length >= st.n) {
        st.done = true
        st.full = true
    }
}
function rbaLazyStop(st, j) {
    st.done = true
    if (st.stopAt < 0) { st.stopAt = j }
}

// The lazy name an op is reached by, or "" when the method is not one.
function rbaLazyOp(name) {
    if (name == "map" || name == "collect") { return "map" }
    if (name == "select" || name == "filter") { return "select" }
    if (name == "reject" || name == "take_while" || name == "drop_while") { return name }
    return ""
}

// The op a blockless each_slice / each_cons is on an Enumerator that pulls one
// value at a time, or "". Go twin: rubySliceOp in abnf/jsrtrubystr.go.
function rbaSliceOp(name) {
    if (name == "each_slice") { return "slice" }
    if (name == "each_cons") { return "cons" }
    return ""
}

// Array#insert's splice position on an array of n (rb_ary_insert): -1 is the end
// and another negative one counts from after the last element. -1 when it lies
// before the start, which is IndexError "index POS too small for array; minimum:
// -n-1".
function rbaInsertPos(n, pos) {
    if (pos >= 0) { return pos }
    if (pos < 0 - n - 1) { return -1 }
    return pos + n + 1
}

// t with items spliced in at pos, padded with nil when pos is past the end.
function rbaSpliced(t, pos, items) {
    var out = []
    var i = 0
    while (i < pos) {
        out.push(i < t.length ? t[i] : null)
        i = i + 1
    }
    var k = 0
    while (k < items.length) {
        out.push(items[k])
        k = k + 1
    }
    while (i < t.length) {
        out.push(t[i])
        i = i + 1
    }
    return out
}

// The length of the first of rows t that is not as long as the first row, or
// -1: Array#transpose's IndexError "element size differs (N should be W)".
function rbaRowMismatch(t) {
    var i = 1
    while (i < t.length) {
        if (t[i].length != t[0].length) { return t[i].length }
        i = i + 1
    }
    return -1
}
// Array#transpose of rows that all have one length.
function rbaTranspose(t) {
    var out = []
    if (t.length == 0) { return out }
    var w = t[0].length
    var c = 0
    while (c < w) {
        var col = []
        var r = 0
        while (r < t.length) {
            col.push(t[r][c])
            r = r + 1
        }
        out.push(col)
        c = c + 1
    }
    return out
}

// Array#chunk_while / #slice_when, given whether each adjacent pair stays
// together (keep[i] for t[i], t[i + 1]).
function rbaChunks(t, keep) {
    var out = []
    if (t.length == 0) { return out }
    var cur = [t[0]]
    var i = 1
    while (i < t.length) {
        if (!keep[i - 1]) {
            out.push(cur)
            cur = []
        }
        cur.push(t[i])
        i = i + 1
    }
    out.push(cur)
    return out
}

// The Enumerable methods whose block MRI 2.6 hands a multi-value `yield a, b` of
// #each as the values themselves rather than packed into one Array: `map { |x| }`
// sees a, `select { |x| }` sees [a, b]. Measured against /usr/bin/ruby per method.
// Go twin: rubyEnumSpreads in abnf/jsrtrubystr.go.
function rbaEnumSpreads(n) {
    return n == "map" || n == "collect" || n == "flat_map" || n == "collect_concat"
        || n == "find_index" || n == "all?" || n == "any?" || n == "none?" || n == "one?"
        || n == "count" || n == "take_while" || n == "uniq" || n == "to_h"
}

// The Enumerable methods MRI ends #each for once the elements so far decide the
// answer: 1 at the first element whose test holds (any? none? find detect
// find_index include? member?), 2 at the first whose test fails (all?
// take_while), 3 at the second that holds (one?), 4 after a count (first take),
// 0 never. find / detect / take_while without a block, and find_index without
// one or a value, answer an Enumerator and iterate nothing. Go twin:
// rubyEnumStop in abnf/jsrtrubystr.go.
function rbaEnumStop(n, hasBlk, nargs) {
    if (n == "first" || n == "take") { return 4 }
    if (n == "any?" || n == "none?" || n == "include?" || n == "member?") { return 1 }
    if (n == "all?") { return 2 }
    if (n == "one?") { return 3 }
    if (!hasBlk && (n != "find_index" || nargs == 0)) { return 0 }
    if (n == "find" || n == "detect" || n == "find_index") { return 1 }
    if (n == "take_while") { return 2 }
    return 0
}
// The Array method that is such a method's test of one element without a
// block: the pattern === (or truthiness) of any?, or the == of include?.
// Go twin: rubyEnumProbe in abnf/jsrtrubystr.go.
function rbaEnumProbe(n) {
    if (n == "any?" || n == "all?" || n == "none?" || n == "one?") { return "any?" }
    return "include?"
}
// Does the Enumerable method n, given nargs arguments and no block, answer an
// Enumerator? Go twin: rubyEnumDefers in abnf/jsrtruby.go.
function rbaEnumDefers(n, nargs) {
    if (n == "each_slice" || n == "each_cons" || n == "each_with_object") { return nargs == 1 }
    if (n == "cycle") { return nargs <= 1 }
    if (nargs != 0) { return false }
    return n == "map" || n == "collect" || n == "flat_map" || n == "collect_concat" || n == "select"
        || n == "filter" || n == "find_all" || n == "reject" || n == "find" || n == "detect"
        || n == "find_index" || n == "group_by" || n == "partition" || n == "min_by" || n == "max_by"
        || n == "minmax_by" || n == "sort_by" || n == "take_while" || n == "drop_while"
        || n == "each_with_index" || n == "each_entry" || n == "reverse_each"
}

// Enumerable#slice_before / #slice_after, given whether each element matched:
// a match opens a new slice before it, or closes the current one after it.
function rbaSliceAt(t, hit, after) {
    var out = []
    var cur = []
    var i = 0
    while (i < t.length) {
        if (!after && hit[i] && cur.length > 0) {
            out.push(cur)
            cur = []
        }
        cur.push(t[i])
        if (after && hit[i]) {
            out.push(cur)
            cur = []
        }
        i = i + 1
    }
    if (cur.length > 0) { out.push(cur) }
    return out
}

// Enumerable#chunk, given each element's block value and its kind: 0 an ordinary
// key, 1 dropped (nil or :_separator), 2 a chunk of its own (:_alone). Runs of
// ordinary keys that eq calls equal share one [key, [elements]] chunk; a dropped
// element ends the run.
function rbaChunk(t, keys, kinds, eq) {
    var out = []
    var cur = anytype
    cur = null
    var i = 0
    while (i < t.length) {
        if (kinds[i] == 1) {
            cur = null
        } else if (kinds[i] == 2) {
            out.push([keys[i], [t[i]]])
            cur = null
        } else if (cur !== null && eq(cur[0], keys[i])) {
            cur[1].push(t[i])
        } else {
            cur = [keys[i], [t[i]]]
            out.push(cur)
        }
        i = i + 1
    }
    return out
}

// Enumerable#minmax_by over [key, element] pairs: the first element of the least
// key and the first of the greatest, as cmp orders keys.
function rbaMinMaxBy(kp, cmp) {
    if (kp.length == 0) { return [null, null] }
    var lo = 0
    var hi = 0
    var i = 1
    while (i < kp.length) {
        if (cmp(kp[i][0], kp[lo][0]) < 0) { lo = i }
        if (cmp(kp[i][0], kp[hi][0]) > 0) { hi = i }
        i = i + 1
    }
    return [kp[lo][1], kp[hi][1]]
}

// The [begin, end) Array#fill writes on an array of n (rb_ary_fill): a negative
// start counts from the end (clamped to 0), an absent one is 0, an absent length
// runs to the end; null when the length is negative. end may lie past n, which
// grows the array.
function rbaFillSpan(n, beg, hasBeg, len, hasLen) {
    var b = hasBeg ? beg : 0
    if (b < 0) {
        b = b + n
        if (b < 0) { b = 0 }
    }
    var l = hasLen ? len : n - b
    if (l < 0) { return null }
    return {b: b, e: b + l}
}
