// js-arrmove.js -- the order in which an Array method that moves elements Sets and
// deletes its receiver's indices (ECMA-262 23.1.3, each method's algorithm). It is
// observable on an array with an index accessor (the setter calls) or an index made
// read-only or non-configurable by Object.defineProperty (where the first refused
// Set or delete raises, after the writes before it were made).
//
// Included by both js and typescript interpreters (include("lib/js-arrmove.js")) and
// imported by languages/lib/js-rt.metajs; the Go twin is jsvMoveOrder in
// abnf/jsrtjsprint.go. Written in the restricted subset of regex.js's head comment.
// Every name is prefixed jsam.

// A relative index argument (ToIntegerOrInfinity, then counted from the end when
// negative) clamped to 0..n.
function jsamRel(x, n) {
    var v = x
    if (v != v) { v = 0 }
    if (v < 0) { v = Math.ceil(v) } else { v = Math.floor(v) }
    if (v < 0) { v = v + n }
    if (v < 0) { v = 0 }
    if (v > n) { v = n }
    return v
}

// The indices method name touches on an array of length n that it leaves m long, in
// order; each is a Set where the result holds an element and a delete where it does
// not. p, q and r are the numeric arguments: splice's start and item count, fill's
// start and end, copyWithin's target, start and end. V8's sort writes nothing back
// for fewer than two elements.
function jsamOrder(name, n, m, p, q, r) {
    var out = []
    var i = 0
    if (name == "reverse") {
        while (i < Math.floor(n / 2)) {
            out.push(i)
            out.push(n - 1 - i)
            i = i + 1
        }
    }
    if (name == "shift" || (name == "sort" && n > 1)) {
        while (i < n) {
            out.push(i)
            i = i + 1
        }
    }
    if (name == "unshift") {
        var argc = m - n
        if (argc > 0) {
            i = n
            while (i > 0) {
                out.push(i + argc - 1)
                i = i - 1
            }
            while (i < argc) {
                out.push(i)
                i = i + 1
            }
        }
    }
    if (name == "splice") {
        var st = jsamRel(p, n)
        var ic = q
        var dc = n - m + ic
        if (ic < dc) {
            i = st
            while (i < n - dc) {
                out.push(i + ic)
                i = i + 1
            }
            i = n
            while (i > n - dc + ic) {
                out.push(i - 1)
                i = i - 1
            }
        }
        if (ic > dc) {
            i = n - dc
            while (i > st) {
                out.push(i + ic - 1)
                i = i - 1
            }
        }
        i = 0
        while (i < ic) {
            out.push(st + i)
            i = i + 1
        }
    }
    if (name == "fill") {
        i = jsamRel(p, n)
        var fe = jsamRel(q, n)
        while (i < fe) {
            out.push(i)
            i = i + 1
        }
    }
    if (name == "copyWithin") {
        var to = jsamRel(p, n)
        var fr = jsamRel(q, n)
        var cnt = Math.min(jsamRel(r, n) - fr, n - to)
        // An overlapping copy to a higher index runs backwards from its last slot.
        if (fr < to && to < fr + cnt) {
            i = cnt
            while (i > 0) {
                out.push(to + i - 1)
                i = i - 1
            }
        } else {
            while (i < cnt) {
                out.push(to + i)
                i = i + 1
            }
        }
    }
    return out
}
