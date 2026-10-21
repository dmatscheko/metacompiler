// ruby-magic.js -- the `# frozen_string_literal: true' magic comment, for
// ruby-interpreter.abnf and ruby-to-llvm-ir.abnf (include). It runs while the tags
// run and reads the file being parsed through c.srcSlice.
//
// MRI reads magic comments in the comment section in front of the first line of
// code (blank lines included). A comment is either exactly
// `key: value' or an Emacs `-*- key: value; ... -*-' line; the key compares
// case-blind with `-' and `_' alike, and only the values true and false count.
// The last one wins.

var rbMgFiles = {}

// Whether the file being parsed asked for frozen string literals.
function rbFrozenLits() {
    var f = "$" + c.curFile()
    if (rbMgFiles[f] === undefined) { rbMgFiles[f] = rbMgScan() }
    return rbMgFiles[f]
}

// Whether a literal's source text interpolates (#{..}, #@ivar, #$global): such a
// literal builds a new String every time, where a plain one is one shared object.
function rbMgInterp(src) {
    return src.indexOf("#{") >= 0 || src.indexOf("#@") >= 0 || src.indexOf("#$") >= 0
}

// c.srcSlice counts BYTES, so the scan reads a prefix and re-reads a longer one
// when the comment section runs past it.
function rbMgScan() {
    var n = 4096
    while (true) {
        var r = rbMgScanText(c.srcSlice(0, n), c.srcSlice(n, n + 1) == "")
        if (r >= 0) { return r == 1 }
        n = n * 4
    }
    return false
}

// 1 / 0 when the comment section of text decides, -1 when it runs past the end of
// a text that is not the whole file.
function rbMgScanText(text, whole) {
    var on = false
    var pos = 0
    while (pos <= text.length) {
        var nl = text.substring(pos).indexOf("\n")
        if (nl < 0) {
            if (!whole) { return -1 }
            nl = text.length
        } else {
            nl = nl + pos
        }
        var t = rbMgTrim(text.substring(pos, nl))
        pos = nl + 1
        if (t.indexOf("#") == 0) {
            var v = rbMgComment(t.substring(1))
            if (v != "") { on = v == "true" }
        } else if (t != "") {
            return on ? 1 : 0
        }
    }
    return on ? 1 : 0
}

// The frozen_string_literal value one comment body sets ("true" / "false"), or "".
function rbMgComment(body) {
    var em = body.indexOf("-*-")
    if (em < 0) { return rbMgPair(body) }
    var rest = body.substring(em + 3)
    var end = rest.indexOf("-*-")
    if (end < 0) { return "" }
    rest = rest.substring(0, end)
    var got = ""
    while (rest != "") {
        var semi = rest.indexOf(";")
        var part = rest
        if (semi >= 0) { part = rest.substring(0, semi); rest = rest.substring(semi + 1) }
        else { rest = "" }
        var pv = rbMgPair(part)
        if (pv != "") { got = pv }
    }
    return got
}

function rbMgPair(s) {
    var colon = s.indexOf(":")
    if (colon < 0) { return "" }
    if (rbMgKey(rbMgTrim(s.substring(0, colon))) != "frozen_string_literal") { return "" }
    var v = rbMgKey(rbMgTrim(s.substring(colon + 1)))
    if (v == "true" || v == "false") { return v }
    return ""
}

// Lower case, with `-' read as `_'.
function rbMgKey(s) {
    var out = ""
    for (var i = 0; i < s.length; i++) {
        var cc = s.charCodeAt(i)
        if (cc >= 65 && cc <= 90) { out += String.fromCharCode(cc + 32) }
        else if (cc == 45) { out += "_" }
        else { out += s.charAt(i) }
    }
    return out
}

function rbMgTrim(s) {
    var a = 0
    var b = s.length
    while (a < b && rbMgSpace(s.charCodeAt(a))) { a++ }
    while (b > a && rbMgSpace(s.charCodeAt(b - 1))) { b-- }
    return s.substring(a, b)
}
function rbMgSpace(cc) { return cc == 32 || cc == 9 || cc == 13 }
