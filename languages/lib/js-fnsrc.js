// js-fnsrc.js -- the source text of a function or a class: what String(f) and
// Function.prototype.toString answer (ES2023 20.2.3.5), the slice of the file its
// definition spans, as written. Included by both js grammars (include("lib/js-fnsrc.js")),
// in the restricted subset of regex.js's head comment. Every name is prefixed fs.
//
// A definition records a SPAN, not its text: fsSpan answers an id for a byte range of
// the current file, whose text is read once. The interpreter slices a span when a
// function is printed (fsText); the compiler hands the module the files' text once
// (fsBlob) with each span's offsets in it (fsFrom / fsTo), and the runtime slices that.
//
// A definition's tag marks its span (fsMark) and the builder it calls takes it (fsTake),
// in the same tag, so a mark never outlives the definition that made it.

var fsNames = []     // the files spans were taken in, by c.curFile()
var fsTexts = []     // their text
var fsMaps = []      // per file: [] while ASCII, else unit index by byte offset
var fsFile = []      // per span: index into the three arrays above
var fsFrom = []      // per span: first and past-the-end UTF-16 unit
var fsTo = []
var fsLast = -1      // the file of the latest span, which the next one is nearly always in
var fsNext = -1

// The file index of the current source, its text read on first use.
function fsCur() {
    var nm = c.curFile()
    if (fsLast >= 0 && fsNames[fsLast] === nm) { return fsLast }
    var i = fsNames.indexOf(nm)
    if (i < 0) {
        var t = c.srcSlice(0, 1073741824)
        var mp = []
        // c.srcSlice and up.pos count BYTES; the text is UTF-16. A non-ASCII file gets
        // the unit index of every byte offset, once.
        if (byteLen(t) != t.length) {
            for (var k = 0; k < t.length; k++) {
                var cc = t.charCodeAt(k)
                var nb = 3
                if (cc < 128) { nb = 1 } else if (cc < 2048) { nb = 2 }
                else if (cc >= 55296 && cc < 56320) { nb = 4 } else if (cc >= 56320 && cc < 57344) { nb = 0 }
                for (var q = 0; q < nb; q++) { mp.push(k) }
            }
            mp.push(t.length)
        }
        fsNames.push(nm)
        fsTexts.push(t)
        fsMaps.push(mp)
        i = fsNames.length - 1
    }
    fsLast = i
    return i
}
function fsUnit(fi, b) {
    var mp = fsMaps[fi]
    if (mp.length == 0) { return b }
    return mp[b]
}
function fsIsSpace(cc) { return cc == 32 || cc == 9 || cc == 10 || cc == 13 }

// The span [from, to) of the current file, as an id; -1 for lib/js-builtins.js, whose
// classes and functions are the engine's own and print as native code. A class MEMBER
// starts at its key: V8 leaves a leading `static` out ('static m() {}' prints
// 'm() {}'), unless `static` is the key itself ('static() {}').
function fsSpan(from, to, member) {
    if (c.curFile() == "js-builtins.js") { return -1 }
    var fi = fsCur()
    var s = fsUnit(fi, from)
    var e = fsUnit(fi, to)
    if (member) {
        var t = fsTexts[fi]
        if (t.substring(s, s + 6) == "static") {
            var k = s + 6
            while (k < e && fsIsSpace(t.charCodeAt(k))) { k = k + 1 }
            if (k > s + 6 && t.charCodeAt(k) != 40) { s = k }
            else if (k == s + 6 && (t.charCodeAt(k) == 91 || t.charCodeAt(k) == 42)) { s = k }
        }
    }
    fsFile.push(fi)
    fsFrom.push(s)
    fsTo.push(e)
    return fsFile.length - 1
}
function fsMark(from, to, member) { fsNext = fsSpan(from, to, member) }
function fsTake() {
    var r = fsNext
    fsNext = -1
    return r
}
// The text of span id.
function fsText(id) { return fsTexts[fsFile[id]].substring(fsFrom[id], fsTo[id]) }
// V8's text for a function with no source of its own: a builtin, a bound function.
function fsNative(name) {
    if (name.substring(0, 6) == "bound ") { name = "" }
    return "function " + name + "() { [native code] }"
}

// A function's text inside a V8 error message: past 128 UTF-16 units, its first 111
// and its last 2 around "...<omitted>...". Twins: jsvElide in abnf/jsrtjsprint.go,
// jvElide in languages/lib/js-rt.metajs.
function fsElide(t) {
    if (t.length <= 128) { return t }
    return t.substring(0, 111) + "...<omitted>..." + t.substring(t.length - 2, t.length)
}

// For the compiler: every file's text, one after the other, and a span's offsets in it.
function fsBlob() {
    var out = ""
    for (var i = 0; i < fsTexts.length; i++) { out = out + fsTexts[i] }
    return out
}
function fsBase(fi) {
    var n = 0
    for (var i = 0; i < fi; i++) { n = n + fsTexts[i].length }
    return n
}
