// ruby-arr.js -- Array index arithmetic of MRI's array.c, for ruby-interpreter.abnf
// (include) and languages/lib/ruby-rt.metajs (import). Go twins in
// abnf/jsrtrubystr.go. Written in the subset both engines accept.

// Enumerator::Lazy, and the endless Enumerator of a blockless Array#cycle: an
// Enumerator object that also carries __rlazy, a SOURCE - kind "arr" (items),
// "cycle" (items, forever) or "range" (lo, lo+1, ... up to hi; hi null is
// endless) - and the OPS map / select / reject / take_while / drop_while / take,
// {k, f, n}, applied to one value at a time. rbaLazyFirst pulls the first n
// results (all of them when n < 0); call(f, v) runs a block, test(v) is Ruby
// truthiness. A take op ends the run once its count has passed it, so nothing
// past it is evaluated. Go twin: rubyLazyFirst in abnf/jsrtrubystr.go.
function rbaLazyFirst(lz, n, call, test) {
    var out = []
    var ops = lz.ops
    var cnt = []
    var on = []
    var j = 0
    while (j < ops.length) {
        cnt.push(0)
        on.push(false)
        j = j + 1
    }
    var i = 0
    var done = n == 0
    while (!done) {
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
        var keep = true
        j = 0
        while (keep && j < ops.length) {
            var op = ops[j]
            if (op.k == "map") {
                v = call(op.f, v)
            } else if (op.k == "select") {
                keep = test(call(op.f, v))
            } else if (op.k == "reject") {
                keep = !test(call(op.f, v))
            } else if (op.k == "take_while") {
                if (!test(call(op.f, v))) {
                    keep = false
                    done = true
                }
            } else if (op.k == "drop_while") {
                if (!on[j]) {
                    if (test(call(op.f, v))) { keep = false } else { on[j] = true }
                }
            } else if (op.k == "take") {
                if (op.n <= 0) {
                    keep = false
                    done = true
                } else {
                    cnt[j] = cnt[j] + 1
                    if (cnt[j] >= op.n) { done = true }
                }
            }
            j = j + 1
        }
        if (keep) {
            out.push(v)
            if (n >= 0 && out.length >= n) { done = true }
        }
    }
    return out
}

// The lazy name an op is reached by, or "" when the method is not one.
function rbaLazyOp(name) {
    if (name == "map" || name == "collect") { return "map" }
    if (name == "select" || name == "filter") { return "select" }
    if (name == "reject" || name == "take_while" || name == "drop_while") { return name }
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
