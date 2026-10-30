// ruby-arr.js -- Array index arithmetic of MRI's array.c, for ruby-interpreter.abnf
// (include) and languages/lib/ruby-rt.metajs (import). Go twins in
// abnf/jsrtrubystr.go. Written in the subset both engines accept.

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
