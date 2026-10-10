// lua-strlib.js -- Lua's pattern matching and the byte-level half of its string
// library: string.find, match, gmatch, gsub, byte and reverse, as lstrlib.c (Lua
// 5.5) does them. Lua patterns are NOT regular expressions: no alternation, a
// quantifier applies to one character class only, and %b, %f and back-references
// (%1) exist instead. The names below follow lstrlib.c.
//
// Written once for the interpreter half (lua-interpreter.abnf includes it) and for
// layer 2 (languages/lib/lua-rt.metajs imports it), in the restricted subset both
// accept - see the head of regex.js. abnf/jsrtluastr.go is the Go twin the compiler
// half runs under llvm.Run, a line-by-line port of this file.
//
// A Lua string is a BYTE string held one host character per byte (charCodeAt is
// 0..255), so every index here is a byte offset. Character classes are the C
// locale's: ASCII only.
//
// Everything here is value-model free: subjects, patterns and captures are host
// strings, positions are plain integers. What touches a Lua value - argument
// checks, number-to-string, calling a replacement function, the multiple results -
// stays with the includer. Errors go through lsFail, a KNOB: by default it throws
// layer 2's {__luerr} marker, and the interpreter overrides it by ASSIGNMENT.

var lsFail = function(msg) {
    var o = {}
    o["__luerr"] = msg
    throw o
}

var LS_CAP_UNFINISHED = 0 - 1
var LS_CAP_POSITION = 0 - 2
var LS_MAXCAPTURES = 32
var LS_MAXCCALLS = 200

// ----- positions -----
// posrelatI: a start position, 1-based, negative counting from the end, clamped to
// at least 1. lsEndPos (getendpos): an end position clamped into 0..len.
function lsPosI(pos, len) {
    if (pos > 0) { return pos }
    if (pos === 0) { return 1 }
    if (pos < 0 - len) { return 1 }
    return len + pos + 1
}
function lsEndPos(pos, len) {
    if (pos > len) { return len }
    if (pos >= 0) { return pos }
    if (pos < 0 - len) { return 0 }
    return len + pos + 1
}

// ----- character classes (C locale) -----
function lsIsAlpha(c) { return (c >= 65 && c <= 90) || (c >= 97 && c <= 122) }
function lsIsDigit(c) { return c >= 48 && c <= 57 }
function lsIsLower(c) { return c >= 97 && c <= 122 }
function lsIsUpper(c) { return c >= 65 && c <= 90 }
function lsIsSpace(c) { return c === 32 || (c >= 9 && c <= 13) }
function lsIsCntrl(c) { return c < 32 || c === 127 }
function lsIsPunct(c) { return c > 32 && c < 127 && !lsIsAlpha(c) && !lsIsDigit(c) }
function lsIsXDigit(c) { return lsIsDigit(c) || (c >= 65 && c <= 70) || (c >= 97 && c <= 102) }

function lsMatchClass(c, cl) {
    var lower = lsIsUpper(cl) ? cl + 32 : cl
    var res = false
    if (lower === 97) { res = lsIsAlpha(c) }
    else if (lower === 99) { res = lsIsCntrl(c) }
    else if (lower === 100) { res = lsIsDigit(c) }
    else if (lower === 103) { res = c > 32 && c < 127 }
    else if (lower === 108) { res = lsIsLower(c) }
    else if (lower === 112) { res = lsIsPunct(c) }
    else if (lower === 115) { res = lsIsSpace(c) }
    else if (lower === 117) { res = lsIsUpper(c) }
    else if (lower === 119) { res = lsIsAlpha(c) || lsIsDigit(c) }
    else if (lower === 120) { res = lsIsXDigit(c) }
    else { return cl === c }
    if (lsIsUpper(cl)) { return !res }
    return res
}

// ----- the match state -----
function lsNewState(src, pat) {
    return {src: src, pat: pat, slen: src.length, plen: pat.length, level: 0,
            cinit: [], clen: [], depth: LS_MAXCCALLS}
}
function lsReprep(ms) {
    ms.level = 0
    ms.depth = LS_MAXCCALLS
}
// The pattern byte at p, or 0 past its end (C's terminating NUL).
function lsP(ms, p) { return p < ms.plen ? ms.pat.charCodeAt(p) : 0 }
function lsS(ms, s) { return s < ms.slen ? ms.src.charCodeAt(s) : 0 }

function lsClassEnd(ms, p) {
    var q = p
    var c = lsP(ms, q)
    q = q + 1
    if (c === 37) {
        if (q >= ms.plen) { lsFail("malformed pattern (ends with '%')") }
        return q + 1
    }
    if (c === 91) {
        if (lsP(ms, q) === 94) { q = q + 1 }
        while (true) {
            if (q >= ms.plen) { lsFail("malformed pattern (missing ']')") }
            var cc = lsP(ms, q)
            q = q + 1
            if (cc === 37 && q < ms.plen) { q = q + 1 }
            if (lsP(ms, q) === 93) { break }
        }
        return q + 1
    }
    return q
}

// [set] at p ('[') .. ec (its ']').
function lsMatchBracket(ms, c, p, ec) {
    var q = p + 1
    var sig = true
    if (lsP(ms, q) === 94) { sig = false; q = q + 1 }
    while (q < ec) {
        var pc = lsP(ms, q)
        if (pc === 37) {
            q = q + 1
            if (lsMatchClass(c, lsP(ms, q))) { return sig }
            q = q + 1
        } else if (lsP(ms, q + 1) === 45 && q + 2 < ec) {
            if (pc <= c && c <= lsP(ms, q + 2)) { return sig }
            q = q + 3
        } else {
            if (pc === c) { return sig }
            q = q + 1
        }
    }
    return !sig
}

function lsSingleMatch(ms, s, p, ep) {
    if (s >= ms.slen) { return false }
    var c = ms.src.charCodeAt(s)
    var pc = lsP(ms, p)
    if (pc === 46) { return true }
    if (pc === 37) { return lsMatchClass(c, lsP(ms, p + 1)) }
    if (pc === 91) { return lsMatchBracket(ms, c, p, ep - 1) }
    return pc === c
}

function lsMatchBalance(ms, s, p) {
    if (p + 1 >= ms.plen) { lsFail("malformed pattern (missing arguments to '%b')") }
    if (s >= ms.slen || ms.src.charCodeAt(s) !== lsP(ms, p)) { return 0 - 1 }
    var b = lsP(ms, p)
    var e = lsP(ms, p + 1)
    var cont = 1
    var q = s + 1
    while (q < ms.slen) {
        var c = ms.src.charCodeAt(q)
        if (c === e) {
            cont = cont - 1
            if (cont === 0) { return q + 1 }
        } else if (c === b) { cont = cont + 1 }
        q = q + 1
    }
    return 0 - 1
}

function lsMaxExpand(ms, s, p, ep) {
    var i = 0
    while (lsSingleMatch(ms, s + i, p, ep)) { i = i + 1 }
    while (i >= 0) {
        var res = lsDoMatch(ms, s + i, ep + 1)
        if (res !== 0 - 1) { return res }
        i = i - 1
    }
    return 0 - 1
}
function lsMinExpand(ms, s, p, ep) {
    var q = s
    while (true) {
        var res = lsDoMatch(ms, q, ep + 1)
        if (res !== 0 - 1) { return res }
        if (lsSingleMatch(ms, q, p, ep)) { q = q + 1 } else { return 0 - 1 }
    }
    return 0 - 1
}

function lsStartCapture(ms, s, p, what) {
    if (ms.level >= LS_MAXCAPTURES) { lsFail("too many captures") }
    var ci = ms.cinit
    var cl = ms.clen
    ci[ms.level] = s
    cl[ms.level] = what
    ms.cinit = ci
    ms.clen = cl
    ms.level = ms.level + 1
    var res = lsDoMatch(ms, s, p)
    if (res === 0 - 1) { ms.level = ms.level - 1 }
    return res
}
function lsCaptureToClose(ms) {
    var level = ms.level - 1
    while (level >= 0) {
        if (ms.clen[level] === LS_CAP_UNFINISHED) { return level }
        level = level - 1
    }
    lsFail("invalid pattern capture")
    return 0
}
function lsEndCapture(ms, s, p) {
    var l = lsCaptureToClose(ms)
    var cl = ms.clen
    cl[l] = s - ms.cinit[l]
    ms.clen = cl
    var res = lsDoMatch(ms, s, p)
    if (res === 0 - 1) {
        var cl2 = ms.clen
        cl2[l] = LS_CAP_UNFINISHED
        ms.clen = cl2
    }
    return res
}
function lsCheckCapture(ms, ch) {
    var l = ch - 49
    if (l < 0 || l >= ms.level || ms.clen[l] === LS_CAP_UNFINISHED) {
        lsFail("invalid capture index %" + (l + 1))
    }
    return l
}
function lsMatchCapture(ms, s, ch) {
    var l = lsCheckCapture(ms, ch)
    var len = ms.clen[l]
    var init = ms.cinit[l]
    if (ms.slen - s >= len && ms.src.substring(init, init + len) === ms.src.substring(s, s + len)) {
        return s + len
    }
    return 0 - 1
}

// do_match: the end of the match of pattern[p..] at subject offset s, or -1.
function lsDoMatch(ms, s0, p0) {
    ms.depth = ms.depth - 1
    if (ms.depth < 0) { lsFail("pattern too complex") }
    var s = s0
    var p = p0
    while (true) {
        if (p >= ms.plen) { break }
        var pc = lsP(ms, p)
        if (pc === 40) {
            if (lsP(ms, p + 1) === 41) { s = lsStartCapture(ms, s, p + 2, LS_CAP_POSITION) }
            else { s = lsStartCapture(ms, s, p + 1, LS_CAP_UNFINISHED) }
            break
        }
        if (pc === 41) {
            s = lsEndCapture(ms, s, p + 1)
            break
        }
        if (pc === 36 && p + 1 === ms.plen) {
            if (s !== ms.slen) { s = 0 - 1 }
            break
        }
        if (pc === 37) {
            var nx = lsP(ms, p + 1)
            if (nx === 98) {
                s = lsMatchBalance(ms, s, p + 2)
                if (s !== 0 - 1) { p = p + 4; continue }
                break
            }
            if (nx === 102) {
                p = p + 2
                if (lsP(ms, p) !== 91) { lsFail("missing '[' after '%f' in pattern") }
                var fep = lsClassEnd(ms, p)
                var prev = s === 0 ? 0 : ms.src.charCodeAt(s - 1)
                if (!lsMatchBracket(ms, prev, p, fep - 1) && lsMatchBracket(ms, lsS(ms, s), p, fep - 1)) {
                    p = fep
                    continue
                }
                s = 0 - 1
                break
            }
            if (lsIsDigit(nx)) {
                s = lsMatchCapture(ms, s, nx)
                if (s !== 0 - 1) { p = p + 2; continue }
                break
            }
        }
        var ep = lsClassEnd(ms, p)
        var epc = lsP(ms, ep)
        if (!lsSingleMatch(ms, s, p, ep)) {
            if (epc === 42 || epc === 63 || epc === 45) { p = ep + 1; continue }
            s = 0 - 1
            break
        }
        if (epc === 63) {
            var r = lsDoMatch(ms, s + 1, ep + 1)
            if (r !== 0 - 1) { s = r; break }
            p = ep + 1
            continue
        }
        if (epc === 43) { s = lsMaxExpand(ms, s + 1, p, ep); break }
        if (epc === 42) { s = lsMaxExpand(ms, s, p, ep); break }
        if (epc === 45) { s = lsMinExpand(ms, s, p, ep); break }
        s = s + 1
        p = ep
    }
    ms.depth = ms.depth + 1
    return s
}

// ----- captures -----
// Capture i of a match s..e: a string, or a 1-based position for a () capture.
// With no captures at all, capture 0 is the whole match.
function lsOneCapture(ms, i, s, e) {
    if (i >= ms.level) {
        if (i !== 0) { lsFail("invalid capture index %" + (i + 1)) }
        return ms.src.substring(s, e)
    }
    var l = ms.clen[i]
    var init = ms.cinit[i]
    if (l === LS_CAP_UNFINISHED) { lsFail("unfinished capture") }
    if (l === LS_CAP_POSITION) { return init + 1 }
    return ms.src.substring(init, init + l)
}
// push_captures: every capture, or the whole match when there are none and
// `whole` is set.
function lsCaptures(ms, s, e, whole) {
    var n = (ms.level === 0 && whole) ? 1 : ms.level
    var out = []
    for (var i = 0; i < n; i++) { out.push(lsOneCapture(ms, i, s, e)) }
    return out
}

// The SPECIALS of a pattern: without one, find is a plain substring search.
function lsNoSpecials(p) {
    for (var i = 0; i < p.length; i++) {
        var c = p.charCodeAt(i)
        if (c === 94 || c === 36 || c === 42 || c === 43 || c === 63 || c === 46
            || c === 40 || c === 91 || c === 37 || c === 45) { return false }
    }
    return true
}

// str_find_aux: string.find (find true) and string.match. init is the third
// argument as an integer, already defaulted to 1. Answers null for no match, else
// the result values: find's start and end (1-based) then the captures; match's
// captures, or the whole match.
function lsFindAux(s, p, initArg, plain, find) {
    var ls = s.length
    var init = lsPosI(initArg, ls) - 1
    if (init > ls) { return null }
    if (find && (plain || lsNoSpecials(p))) {
        var at = s.indexOf(p, init)
        if (at < 0) { return null }
        return [at + 1, at + p.length]
    }
    var anchor = p.length > 0 && p.charCodeAt(0) === 94
    var pat = anchor ? p.substring(1) : p
    var ms = lsNewState(s, pat)
    var s1 = init
    while (true) {
        lsReprep(ms)
        var e = lsDoMatch(ms, s1, 0)
        if (e !== 0 - 1) {
            if (find) {
                var r = [s1 + 1, e]
                var cs = lsCaptures(ms, 0 - 1, 0, false)
                for (var i = 0; i < cs.length; i++) { r.push(cs[i]) }
                return r
            }
            return lsCaptures(ms, s1, e, true)
        }
        s1 = s1 + 1
        if (anchor || s1 > ls) { break }
    }
    return null
}

// gmatch: the iteration state, and one step of it - the captures of the next
// match, or null when there is none. A match may not end where the previous one
// did, which is what keeps an empty match from repeating.
function lsGmatchNew(s, p, initArg) {
    var ls = s.length
    var init = lsPosI(initArg, ls) - 1
    if (init > ls) { init = ls + 1 }
    return {s: s, p: p, pos: init, last: 0 - 1}
}
function lsGmatchNext(gm) {
    var ms = lsNewState(gm.s, gm.p)
    var src = gm.pos
    while (src <= ms.slen) {
        lsReprep(ms)
        var e = lsDoMatch(ms, src, 0)
        if (e !== 0 - 1 && e !== gm.last) {
            gm.pos = e
            gm.last = e
            return lsCaptures(ms, src, e, true)
        }
        src = src + 1
    }
    return null
}

// add_s: a replacement string with %0..%9 and %%. A capture's value is a string or
// a position; `numStr` renders the position the way tostring renders an integer.
function lsAddS(ms, repl, s, e) {
    var out = ""
    var i = 0
    while (i < repl.length) {
        var c = repl.charAt(i)
        if (c !== "%") { out = out + c; i = i + 1; continue }
        i = i + 1
        var d = i < repl.length ? repl.charCodeAt(i) : 0
        if (d === 37) { out = out + "%" }
        else if (lsIsDigit(d)) {
            if (d === 48) { out = out + ms.src.substring(s, e) }
            else {
                var cv = anytype
                cv = lsOneCapture(ms, d - 49, s, e)
                out = out + cv
            }
        } else { lsFail("invalid use of '%' in replacement string") }
        i = i + 1
    }
    return out
}

// gsub. `repl` is the replacement string (a number already rendered), or null when
// `cb` decides: cb(captures) answers the replacement text, or null to keep the
// match as it was. maxN < 0 means no limit. Answers {r, n}.
function lsGsub(s, p, repl, cb, maxN) {
    var anchor = p.length > 0 && p.charCodeAt(0) === 94
    var pat = anchor ? p.substring(1) : p
    var ms = lsNewState(s, pat)
    var src = 0
    var last = 0 - 1
    var n = 0
    var out = ""
    while (maxN < 0 || n < maxN) {
        lsReprep(ms)
        var e = lsDoMatch(ms, src, 0)
        if (e !== 0 - 1 && e !== last) {
            n = n + 1
            if (repl !== null) { out = out + lsAddS(ms, repl, src, e) }
            else {
                var rv = anytype
                rv = cb(lsCaptures(ms, src, e, true))
                if (rv === null) { out = out + s.substring(src, e) } else { out = out + rv }
            }
            src = e
            last = e
        } else if (src < ms.slen) {
            out = out + s.charAt(src)
            src = src + 1
        } else { break }
        if (anchor) { break }
    }
    if (src < ms.slen) { out = out + s.substring(src, ms.slen) }
    return {r: out, n: n}
}

// string.byte(s, i, j) with i and j already integers (j defaulted to i): the byte
// values of the range.
function lsByte(s, i, j) {
    var l = s.length
    var pose = lsEndPos(j, l)
    var pi = lsPosI(i, l)
    var out = []
    var k = pi
    while (k <= pose) {
        out.push(s.charCodeAt(k - 1))
        k = k + 1
    }
    return out
}

function lsReverse(s) {
    var out = ""
    var i = s.length - 1
    while (i >= 0) {
        out = out + s.charAt(i)
        i = i - 1
    }
    return out
}
