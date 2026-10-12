// python-suggest.js -- CPython's "Did you mean" suggestion
// (_Py_CalculateSuggestions in Python/suggestions.c), for
// python-interpreter.abnf (include) and languages/lib/python-rt.metajs (import).
// Go twin: abnf/jsrtpysuggest.go.
//
// A Levenshtein distance where an insertion, a deletion or a substitution costs
// 2 and a substitution that only flips ASCII case costs 1. CPython measures
// UTF-8 bytes; this measures UTF-16 code units, which is the same for an ASCII
// name.

// The message suffix: ". Did you mean 'x'?", or "" when nothing is close.
function pyDidYouMean(name, cands) {
    var s = pysgSuggest(name, cands)
    if (s == "") { return "" }
    return ". Did you mean '" + s + "'?"
}

// The closest of cands to name within a third of the characters involved, the
// first one on a tie; "" when none qualifies.
function pysgSuggest(name, cands) {
    if (cands.length >= 750) { return "" }
    var best = ""
    var bestDist = 0 - 1
    for (var i = 0; i < cands.length; i++) {
        var item = cands[i]
        if (item == name) { continue }
        var maxd = Math.floor((name.length + item.length + 3) * 2 / 6)
        if (bestDist >= 0 && bestDist - 1 < maxd) { maxd = bestDist - 1 }
        var d = pysgDistance(name, item, maxd)
        if (d > maxd) { continue }
        if (bestDist < 0 || d < bestDist) {
            best = item
            bestDist = d
        }
    }
    return best
}

function pysgSubCost(a, b) {
    if (a % 32 != b % 32) { return 2 }
    if (a == b) { return 0 }
    if (a >= 65 && a <= 90) { a = a + 32 }
    if (b >= 65 && b <= 90) { b = b + 32 }
    if (a == b) { return 1 }
    return 2
}

// levenshtein_distance: common affixes trimmed, one row updated in place, and
// maxCost + 1 as soon as no cell of a row is within reach.
function pysgDistance(a0, b0, maxCost) {
    var sa = 0
    var ea = a0.length
    var sb = 0
    var eb = b0.length
    while (sa < ea && sb < eb && a0.charCodeAt(sa) == b0.charCodeAt(sb)) {
        sa = sa + 1
        sb = sb + 1
    }
    while (sa < ea && sb < eb && a0.charCodeAt(ea - 1) == b0.charCodeAt(eb - 1)) {
        ea = ea - 1
        eb = eb - 1
    }
    var an = ea - sa
    var bn = eb - sb
    if (an == 0 || bn == 0) { return (an + bn) * 2 }
    if (an > 40 || bn > 40) { return maxCost + 1 }
    var a = a0.substring(sa, ea)
    var b = b0.substring(sb, eb)
    if (bn < an) {
        var t = a
        a = b
        b = t
        an = a.length
        bn = b.length
    }
    if ((bn - an) * 2 > maxCost) { return maxCost + 1 }
    var buf = []
    for (var i = 0; i < an; i++) { buf.push((i + 1) * 2) }
    var result = 0
    for (var bi = 0; bi < bn; bi++) {
        var code = b.charCodeAt(bi)
        var dist = bi * 2
        result = dist
        var minimum = 0 - 1
        for (var ai = 0; ai < an; ai++) {
            var sub = dist + pysgSubCost(code, a.charCodeAt(ai))
            dist = buf[ai]
            var insdel = (result < dist ? result : dist) + 2
            result = insdel < sub ? insdel : sub
            buf[ai] = result
            if (minimum < 0 || result < minimum) { minimum = result }
        }
        if (minimum > maxCost) { return maxCost + 1 }
    }
    return result
}
