// go-regexp.js -- the three primitives lib/go-regexp.go (Go's regexp package)
// is written over, on lib/regex.js's R dialect, with Go's byte offsets. For
// go-interpreter.abnf (include) and lib/go-rt.metajs (import), each after
// lib/regex.js, lib/java-chars.js and lib/java-regex.js. Each of the two defines
// goRxText(s), which reads a Go string into goRxT (the engine's text: UTF-16
// units, a byte that is not part of a character as U+DC00 + byte) and goRxOff
// (the byte offset of each unit and of the end; empty when every unit is one
// byte). Go twin: abnf/jsrtgoregexp.go.

// The Go string the last search read, and whether there was one.
var goRxS = ""
var goRxHave = false
var goRxT = ""
var goRxOff = []

// goRxError: Go's error text for the pattern, "" when it compiles.
function goRxError(expr) {
    var re = rxGet(expr, "R")
    return re.ok ? "" : re.err
}

// goRxNames: Regexp.SubexpNames, "" for the whole match and an unnamed group.
function goRxNames(expr) {
    var re = rxGet(expr, "R")
    var out = [""]
    var i = 1
    while (i <= re.ngroups) {
        out.push(rxNameAt(re, i))
        i = i + 1
    }
    return out
}

// goRxExec: the leftmost match of the pattern in s at or after byte pos, as
// Go's submatch index pairs (-1 for a group that did not take part); empty when
// there is none.
function goRxExec(expr, s, pos) {
    var re = rxGet(expr, "R")
    if (!goRxHave || s != goRxS) {
        goRxText(s)
        goRxS = s
        goRxHave = true
    }
    var out = []
    var m = rxSearch(re, goRxT, goRxUnit(pos))
    if (m == null) { return out }
    var caps = m.caps
    var i = 0
    while (i < caps.length) {
        out.push(caps[i] < 0 ? 0 - 1 : goRxOff.length == 0 ? caps[i] : goRxOff[caps[i]])
        i = i + 1
    }
    return out
}

// The unit at byte offset b, where a character starts.
function goRxUnit(b) {
    if (goRxOff.length == 0) { return b }
    var lo = 0
    var hi = goRxOff.length - 1
    while (lo < hi) {
        var mid = Math.floor((lo + hi) / 2)
        if (goRxOff[mid] < b) { lo = mid + 1 } else { hi = mid }
    }
    return lo
}
