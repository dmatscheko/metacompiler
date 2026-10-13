// js-norm.js -- String.prototype.normalize: the Unicode normalization forms NFC, NFD,
// NFKC and NFKD (UAX #15). Included by the js and typescript interpreters
// (include("lib/js-norm.js")) and imported by languages/lib/js-rt.metajs, in the
// restricted subset of regex.js's head comment. Every name is prefixed jsnorm.
// abnf/jsrtjsnorm.go is the Go twin's port.
//
// The tables are the text of lib/unicode-norm.txt, handed over once by jsnormLoad
// (the interpreters load() it, a compiled program registers it at boot): one line
// per entry,
//   c<cp> <a> <b>     a canonical decomposition that is also a primary composite
//   d<cp> <x>...      a canonical decomposition that never recomposes (a singleton,
//                     a non-starter decomposition or a composition exclusion)
//   k<cp> <x>...      a compatibility decomposition
//   n<lo> <hi> <ccc>  a run of code points with that nonzero canonical combining class
// code points in hex, ccc in decimal. Hangul syllables decompose and compose
// arithmetically (Unicode 3.12) and have no lines. A lone surrogate is a starter with
// no decomposition and passes through.

var jsnormIdx = {}       // "$" + cp -> index into the two arrays below
var jsnormKind = []      // 99 (c), 100 (d) or 107 (k)
var jsnormMap = []       // the decomposition, one level deep
var jsnormComp = {}      // "$" + a + "," + b -> the primary composite
var jsnormLo = []        // the combining-class runs, ascending
var jsnormHi = []
var jsnormCcc = []
var jsnormReady = [false]

function jsnormHex(t, i, end) {
    var v = 0
    for (var j = i; j < end; j++) {
        var c = t.charCodeAt(j)
        v = v * 16 + ((c <= 57) ? c - 48 : c - 87)
    }
    return v
}

// The numbers of one line, from i (after the kind letter) to end; the last one in
// decimal when dec says so.
function jsnormNums(t, i, end, dec) {
    var out = []
    var s = i
    for (var j = i; j <= end; j++) {
        if (j == end || t.charCodeAt(j) == 32) {
            if (j > s) {
                if (dec && j == end) { out.push(jsnormDec(t, s, j)) } else { out.push(jsnormHex(t, s, j)) }
            }
            s = j + 1
        }
    }
    return out
}
function jsnormDec(t, i, end) {
    var v = 0
    for (var j = i; j < end; j++) { v = v * 10 + (t.charCodeAt(j) - 48) }
    return v
}

function jsnormLoad(t) {
    if (jsnormReady[0]) { return }
    jsnormReady[0] = true
    var i = 0
    while (i < t.length) {
        var end = i
        while (end < t.length && t.charCodeAt(end) != 10) { end = end + 1 }
        var kind = t.charCodeAt(i)
        if (kind == 110) {
            var r = jsnormNums(t, i + 1, end, true)
            jsnormLo.push(r[0])
            jsnormHi.push(r[1])
            jsnormCcc.push(r[2])
        } else if (kind == 99 || kind == 100 || kind == 107) {
            var ns = jsnormNums(t, i + 1, end, false)
            var m = []
            for (var k = 1; k < ns.length; k++) { m.push(ns[k]) }
            jsnormIdx["$" + ns[0]] = jsnormKind.length
            jsnormKind.push(kind)
            jsnormMap.push(m)
            if (kind == 99) { jsnormComp["$" + m[0] + "," + m[1]] = ns[0] }
        }
        i = end + 1
    }
}

function jsnormClass(cp) {
    if (cp < 768) { return 0 }
    var lo = 0
    var hi = jsnormLo.length - 1
    while (lo <= hi) {
        var mid = Math.floor((lo + hi) / 2)
        if (cp < jsnormLo[mid]) { hi = mid - 1 }
        else if (cp > jsnormHi[mid]) { lo = mid + 1 }
        else { return jsnormCcc[mid] }
    }
    return 0
}

// The full decomposition of cp, appended to out.
function jsnormDecomp(cp, compat, out) {
    if (cp >= 44032 && cp <= 55203) {
        var si = cp - 44032
        out.push(4352 + Math.floor(si / 588))
        out.push(4449 + Math.floor((si % 588) / 28))
        if (si % 28 != 0) { out.push(4519 + si % 28) }
        return
    }
    var e = jsnormIdx["$" + cp]
    if (e == undefined || (jsnormKind[e] == 107 && !compat)) {
        out.push(cp)
        return
    }
    var m = jsnormMap[e]
    for (var i = 0; i < m.length; i++) { jsnormDecomp(m[i], compat, out) }
}

function jsnormCompose(a, b) {
    if (a >= 4352 && a < 4371 && b >= 4449 && b < 4470) { return 44032 + ((a - 4352) * 21 + (b - 4449)) * 28 }
    if (a >= 44032 && a <= 55203 && (a - 44032) % 28 == 0 && b > 4519 && b < 4547) { return a + (b - 4519) }
    var p = jsnormComp["$" + a + "," + b]
    if (p == undefined) { return -1 }
    return p
}

// s normalized to form, one of "NFC", "NFD", "NFKC", "NFKD"; the caller has checked it.
function jsnormApply(s, form) {
    var quick = true
    for (var q = 0; q < s.length; q++) {
        if (s.charCodeAt(q) >= 160) { quick = false; break }
    }
    if (quick) { return s }
    var compat = form == "NFKC" || form == "NFKD"
    var cps = []
    var i = 0
    while (i < s.length) {
        var c = s.charCodeAt(i)
        if (c >= 55296 && c <= 56319 && i + 1 < s.length) {
            var d = s.charCodeAt(i + 1)
            if (d >= 56320 && d <= 57343) {
                jsnormDecomp(65536 + (c - 55296) * 1024 + (d - 56320), compat, cps)
                i = i + 2
                continue
            }
        }
        jsnormDecomp(c, compat, cps)
        i = i + 1
    }
    // Canonical ordering: a stable sort of each run of nonzero classes.
    var cls = []
    for (var k = 0; k < cps.length; k++) { cls.push(jsnormClass(cps[k])) }
    for (var x = 1; x < cps.length; x++) {
        var y = x
        while (y > 0 && cls[y] != 0 && cls[y - 1] > cls[y]) {
            var tc = cps[y]
            cps[y] = cps[y - 1]
            cps[y - 1] = tc
            var tk = cls[y]
            cls[y] = cls[y - 1]
            cls[y - 1] = tk
            y = y - 1
        }
    }
    if (form == "NFC" || form == "NFKC") { cps = jsnormRecompose(cps, cls) }
    var out = ""
    for (var w = 0; w < cps.length; w++) {
        var v = cps[w]
        if (v >= 65536) {
            out = out + String.fromCharCode(55296 + Math.floor((v - 65536) / 1024)) + String.fromCharCode(56320 + (v - 65536) % 1024)
        } else {
            out = out + String.fromCharCode(v)
        }
    }
    return out
}

// Canonical composition: a character combines with the last starter unless a
// character between them has class 0 or a class not below its own (blocked).
function jsnormRecompose(cps, cls) {
    var out = []
    var si = -1
    var last = 0
    for (var i = 0; i < cps.length; i++) {
        var cp = cps[i]
        var cc = cls[i]
        if (si >= 0 && (out.length - 1 == si || (last != 0 && last < cc))) {
            var p = jsnormCompose(out[si], cp)
            if (p >= 0) {
                out[si] = p
                continue
            }
        }
        if (cc == 0) { si = out.length }
        last = cc
        out.push(cp)
    }
    return out
}
