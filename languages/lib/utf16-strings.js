// utf16-strings.js -- case mapping and literal search over a string's UTF-16
// units, shared by lib/java-strings.js (java.lang.String) and
// lib/csharp-strings.js (System.String): java-interpreter.abnf and
// csharp-interpreter.abnf include it, languages/lib/java-rt.metajs and
// languages/lib/csharp-rt.metajs import it. The Go twin's port is the
// functions of the same names in abnf/jsrtjavastr.go. No value-model hook.
//
// Case mapping here is Unicode's SIMPLE mapping (unicodeCase modes 1 and 0, one
// character for one), which is Character.toUpperCase's and .NET's ToUpper /
// ToLower. java's String.toUpperCase / toLowerCase are the FULL mapping ("ß" ->
// "SS") with the final sigma, and call unicodeCase themselves.

// A unit whose simple mapping leaves the BMP maps to itself.
function jxChUp(c) {
    var s = unicodeCase(String.fromCharCode(c), 1)
    if (s.length == 1) { return s.charCodeAt(0) }
    return c
}
function jxChLo(c) {
    var s = unicodeCase(String.fromCharCode(c), 0)
    if (s.length == 1) { return s.charCodeAt(0) }
    return c
}
function jxUpper(s) { return unicodeCase(s, 1) }
function jxLower(s) { return unicodeCase(s, 0) }
// Does t occur in s at i?
function jxAt(s, t, i) {
    if (i < 0 || i + t.length > s.length) { return false }
    var k = 0
    while (k < t.length) {
        if (s.charCodeAt(i + k) != t.charCodeAt(k)) { return false }
        k = k + 1
    }
    return true
}
// indexOf(str, from): `from` clamps to [0, length], so an empty str is found
// at the clamped position.
function jxIndexOf(s, t, from) {
    var i = from
    if (i < 0) { i = 0 }
    if (i > s.length) { i = s.length }
    while (i + t.length <= s.length) {
        if (jxAt(s, t, i)) { return i }
        i = i + 1
    }
    return 0 - 1
}
// lastIndexOf(str, from): the rightmost occurrence starting at or before from.
function jxLastIndexOf(s, t, from) {
    var i = s.length - t.length
    if (from < i) { i = from }
    while (i >= 0) {
        if (jxAt(s, t, i)) { return i }
        i = i - 1
    }
    return 0 - 1
}
// replace(target, replacement), every occurrence, literally. An empty target
// matches before every unit and at the end: "ab".replace("", "-") is "-a-b-".
function jxReplace(s, t, r) {
    var out = ""
    if (t.length == 0) {
        var k = 0
        while (k < s.length) {
            out = out + r + s.charAt(k)
            k = k + 1
        }
        return out + r
    }
    var last = 0
    var i = jxIndexOf(s, t, 0)
    while (i >= 0) {
        out = out + s.substring(last, i) + r
        last = i + t.length
        i = jxIndexOf(s, t, last)
    }
    return out + s.substring(last, s.length)
}
function jxRepeat(s, n) {
    var out = ""
    var piece = s
    var k = n
    while (k > 0) {
        if (k % 2 == 1) { out = out + piece }
        k = (k - k % 2) / 2
        if (k > 0) { piece = piece + piece }
    }
    return out
}
