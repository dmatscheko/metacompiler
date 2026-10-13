// lua-strlib.js -- Lua's pattern matching and the byte-level string library:
// string.find, match, gmatch, gsub, byte, reverse, sub, rep, upper, lower and
// format, as lstrlib.c (Lua 5.5) does them. Lua patterns are NOT regular expressions: no alternation, a
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
    else if (lower === 122) { res = c === 0 }   // %z, kept by lua 5.4 for the NUL byte
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

// string.sub(s, i, j), j already defaulted to -1: a negative index counts from the
// end, the start clamps up to 1 and the end down to #s, and an empty range is "".
function lsSub(s, i, j) {
    var l = s.length
    var a = lsPosI(i, l)
    var e = lsEndPos(j, l)
    if (a > e) { return "" }
    return s.substring(a - 1, e)
}

// string.rep(s, n, sep): n copies joined by sep, "" when n < 1.
function lsRep(s, n, sep) {
    // An empty piece repeats to "" at any count, which must not cost the count:
    // string.rep("", 2^40) answers at once in lua. Twin of js_lurep in
    // abnf/jsrtluastr.go.
    if (n < 1 || s.length + sep.length === 0) { return "" }
    var out = s
    var k = 1
    while (k < n) {
        out = out + sep + s
        k = k + 1
    }
    return out
}

// string.upper / string.lower: ASCII only. Lua case-maps with the C library in the C
// locale, which leaves every byte above 127 alone; a host case map would treat the
// bytes of a multi-byte character as characters of their own.
function lsCase(s, up) {
    var out = ""
    for (var i = 0; i < s.length; i++) {
        var c = s.charCodeAt(i)
        if (up && c >= 97 && c <= 122) { c = c - 32 }
        else if (!up && c >= 65 && c <= 90) { c = c + 32 }
        out = out + String.fromCharCode(c)
    }
    return out
}

// ----- string.format -----
// str_format of lstrlib.c over C's printf, as lua 5.5 on this platform (macOS
// libc) prints it. The parts that touch a Lua value are the includer's HOOKS:
//   lsFmtArgErr(n, msg)    raise "bad argument #n to 'format' (msg)"
//   lsFmtCheckStr(args, n) luaL_checklstring of argument n (1-based)
//   lsFmtNum(args, n)      luaL_checknumber, as a double
//   lsFmtInt(args, n)      luaL_checkinteger, as the 64 bit pair {h, l} of the
//                          includer's i64 layer (i64StrU, i64Neg and i64ShrU are
//                          the includer's too)
//   lsFmtToStr(args, n)    luaL_tolstring
//   lsFmtLitKind(args, n)  for %q: "s" a string, "i" an integer, "f" a float, "t"
//                          nil or a boolean, "x" anything else
//   lsExactDec(a)          the exact decimal of a finite a >= 0 as {i, f}
//   lsRoundHalfEven(ds, k) the digit string rounded to k digits, half to even
// Twin of fmtFormat in abnf/jsrtluastr.go.

var LS_FMT_F = "-+ #0"   // a A e E f g G
var LS_FMT_X = "-#0"     // o x X
var LS_FMT_I = "-+ 0"    // d i
var LS_FMT_U = "-0"      // u
var LS_FMT_C = "-"       // c s
var LS_HEX = "0123456789abcdef"

function lsIndexOfCh(set, c) { return set.indexOf(String.fromCharCode(c)) >= 0 }
function lsGet2Digits(form, i) {
    var j = i
    if (j < form.length && lsIsDigit(form.charCodeAt(j))) {
        j = j + 1
        if (j < form.length && lsIsDigit(form.charCodeAt(j))) { j = j + 1 }
    }
    return j
}
// checkformat: the flags this conversion allows, a width and (if allowed) a
// precision of at most two digits each, then the conversion letter.
function lsFormCh(form, i) { return i < form.length ? form.charCodeAt(i) : 0 }
function lsCheckFormat(form, flags, precision) {
    var i = 1
    while (i < form.length && lsIndexOfCh(flags, form.charCodeAt(i))) { i = i + 1 }
    if (lsFormCh(form, i) !== 48) {
        i = lsGet2Digits(form, i)
        if (lsFormCh(form, i) === 46 && precision) { i = lsGet2Digits(form, i + 1) }
    }
    if (!lsIsAlpha(lsFormCh(form, i))) { lsFail("invalid conversion specification: '" + form + "'") }
}
// The parsed spec of a checked form: its flags, width and precision (-1 none).
function lsSpec(form) {
    var sp = {left: false, plus: false, space: false, alt: false, zero: false, width: 0, prec: 0 - 1}
    var i = 1
    while (i < form.length) {
        var c = form.charCodeAt(i)
        if (c === 45) { sp.left = true }
        else if (c === 43) { sp.plus = true }
        else if (c === 32) { sp.space = true }
        else if (c === 35) { sp.alt = true }
        else if (c === 48) { sp.zero = true }
        else { break }
        i = i + 1
    }
    var w = 0
    while (i < form.length && lsIsDigit(form.charCodeAt(i))) { w = w * 10 + form.charCodeAt(i) - 48; i = i + 1 }
    sp.width = w
    if (i < form.length && form.charCodeAt(i) === 46) {
        i = i + 1
        var p = 0
        while (i < form.length && lsIsDigit(form.charCodeAt(i))) { p = p * 10 + form.charCodeAt(i) - 48; i = i + 1 }
        sp.prec = p
    }
    return sp
}
// Pads `body` (after `sign`, which zero padding goes behind) to the width.
function lsPad(sp, sign, body, zeroOk) {
    var n = sign.length + body.length
    if (n >= sp.width) { return sign + body }
    var fill = ""
    var k = n
    while (k < sp.width) { fill = fill + (sp.zero && zeroOk && !sp.left ? "0" : " "); k = k + 1 }
    if (sp.left) { return sign + body + fill }
    if (sp.zero && zeroOk) { return sign + fill + body }
    return fill + sign + body
}
function lsZeros(n) {
    var s = ""
    var k = 0
    while (k < n) { s = s + "0"; k = k + 1 }
    return s
}
function lsUpper(s) { return lsCase(s, true) }
// The digits a 64 bit pair prints with: its sign and magnitude (d), and its bit
// pattern read unsigned in base 10, 16 and 8.
function lsIntDigits(p) {
    var neg = (p.h & 0x80000000) !== 0
    var x = ""
    var o = ""
    var k = 0
    while (k < 16) {
        var nib = k < 8 ? (p.l >>> (k * 4)) & 15 : (p.h >>> ((k - 8) * 4)) & 15
        x = LS_HEX.charAt(nib) + x
        k = k + 1
    }
    var q = 0
    while (q < 22) {
        o = LS_HEX.charAt(i64ShrU(p, q * 3).l & 7) + o
        q = q + 1
    }
    while (x.length > 1 && x.charCodeAt(0) === 48) { x = x.substring(1) }
    while (o.length > 1 && o.charCodeAt(0) === 48) { o = o.substring(1) }
    return {neg: neg, d: i64StrU(neg ? i64Neg(p) : p), u: i64StrU(p), x: x, o: o}
}

// %d %i %u %o %x %X
function lsFmtIntConv(sp, conv, iv) {
    var digits = iv.d
    var sign = ""
    if (conv === 117) { digits = iv.u }
    else if (conv === 111) { digits = iv.o }
    else if (conv === 120) { digits = iv.x }
    else if (conv === 88) { digits = lsUpper(iv.x) }
    else if (iv.neg) { sign = "-" }
    else if (sp.plus) { sign = "+" }
    else if (sp.space) { sign = " " }
    var zero = digits == "0"
    if (sp.prec >= 0) {
        if (sp.prec === 0 && zero) { digits = "" }
        if (digits.length < sp.prec) { digits = lsZeros(sp.prec - digits.length) + digits }
    }
    if (sp.alt && conv === 111 && digits.charCodeAt(0) !== 48) { digits = "0" + digits }
    if (sp.alt && !zero && conv === 120) { sign = "0x" }
    if (sp.alt && !zero && conv === 88) { sign = "0X" }
    return lsPad(sp, sign, digits, sp.prec < 0)
}

// The decimal digits of a finite a > 0 rounded to `n` significant digits, and
// the exponent of the first: {ds, e}.
function lsSigDigits(a, n) {
    var dec = lsExactDec(a)
    var all = dec.i + dec.f
    var e = dec.i.length - 1
    var lead = 0
    while (lead < all.length && all.charCodeAt(lead) === 48) { lead = lead + 1 }
    e = e - lead
    var ds = lsRoundHalfEven(all.substring(lead), n)
    if (ds.length > n) { ds = ds.substring(0, n); e = e + 1 }
    return {ds: ds, e: e}
}
function lsExpText(e) {
    var a = e < 0 ? 0 - e : e
    var s = "" + a
    if (s.length < 2) { s = "0" + s }
    return (e < 0 ? "-" : "+") + s
}
// %e of a finite a >= 0 with `prec` digits after the point.
function lsFmtE(a, prec, alt) {
    var ds = ""
    var e = 0
    if (a === 0) { ds = lsZeros(prec + 1) } else {
        var r = lsSigDigits(a, prec + 1)
        ds = r.ds
        e = r.e
    }
    var body = ds.substring(0, 1)
    if (prec > 0 || alt) { body = body + "." + ds.substring(1) }
    return body + "e" + lsExpText(e)
}
// %f of a finite a >= 0.
function lsFmtF(a, prec, alt) {
    var dec = lsExactDec(a)
    var kept = lsRoundHalfEven(dec.i + dec.f, dec.i.length + prec)
    var ilen = dec.i.length
    if (kept.length > dec.i.length + prec) { ilen = ilen + 1 }
    var ints = kept.substring(0, ilen)
    if (ints == "") { ints = "0" }
    if (prec > 0) { return ints + "." + kept.substring(ilen) }
    return alt ? ints + "." : ints
}
// %g: %e or %f by the exponent the value has at P significant digits, trailing
// zeros (and a bare point) removed unless '#'.
function lsFmtG(a, prec, alt) {
    var P = prec < 0 ? 6 : (prec === 0 ? 1 : prec)
    var x = 0
    if (a !== 0) { x = lsSigDigits(a, P).e }
    var s = ""
    if (x < 0 - 4 || x >= P) { s = lsFmtE(a, P - 1, alt) } else { s = lsFmtF(a, P - 1 - x, alt) }
    if (alt) { return s }
    var ei = s.indexOf("e")
    var mant = ei < 0 ? s : s.substring(0, ei)
    var tail = ei < 0 ? "" : s.substring(ei)
    if (mant.indexOf(".") >= 0) {
        var k = mant.length
        while (k > 0 && mant.charCodeAt(k - 1) === 48) { k = k - 1 }
        if (k > 0 && mant.charCodeAt(k - 1) === 46) { k = k - 1 }
        mant = mant.substring(0, k)
    }
    return mant + tail
}
// %a of a finite a >= 0: 0x1.<hex>p<exp>, normalized (a subnormal too). With a
// precision the hex digits are cut and rounded as this platform's libc rounds
// them, which reads only the next two dropped digits: up from 0x90, down below
// 0x80, and in between up exactly when the second digit is odd (measured over
// every two-digit tail); a carry makes the lead digit 2.
function lsFmtA(a, prec, alt) {
    var lead = "1"
    var ds = ""
    var e = 0
    if (a === 0) { lead = "0" } else {
        var m = a
        while (m >= 2) { m = m / 2; e = e + 1 }
        while (m < 1) { m = m * 2; e = e - 1 }
        var f = m - 1
        while (f > 0) {
            f = f * 16
            var d = Math.floor(f)
            ds = ds + LS_HEX.charAt(d)
            f = f - d
        }
    }
    if (prec >= 0) {
        if (ds.length > prec) {
            var next = LS_HEX.indexOf(ds.charAt(prec)) * 16
            if (prec + 1 < ds.length) { next = next + LS_HEX.indexOf(ds.charAt(prec + 1)) }
            var up = next >= 144 || (next >= 128 && next % 2 === 1)
            ds = ds.substring(0, prec)
            if (up) {
                var codes = []
                for (var i = 0; i < ds.length; i++) { codes.push(LS_HEX.indexOf(ds.charAt(i))) }
                var at = codes.length - 1
                var carry = true
                while (at >= 0 && carry) {
                    if (codes[at] === 15) { codes[at] = 0 } else { codes[at] = codes[at] + 1; carry = false }
                    at = at - 1
                }
                ds = ""
                for (var j = 0; j < codes.length; j++) { ds = ds + LS_HEX.charAt(codes[j]) }
                if (carry) { lead = lead == "0" ? "1" : "2" }
            }
        }
        while (ds.length < prec) { ds = ds + "0" }
    }
    var body = "0x" + lead
    if (ds.length > 0 || alt) { body = body + "." + ds }
    return body + "p" + (e < 0 ? "-" : "+") + (e < 0 ? 0 - e : e)
}
// A float conversion: the sign, inf / nan (which take no zero padding, and nan no
// sign at all), the digits, the case.
function lsFmtFloat(sp, conv, v) {
    var lowc = conv >= 97 ? conv : conv + 32
    var neg = v < 0 || (v === 0 && 1 / v < 0)
    var a = neg ? 0 - v : v
    var sign = ""
    if (v !== v) { neg = false }
    else if (neg) { sign = "-" }
    else if (sp.plus) { sign = "+" }
    else if (sp.space) { sign = " " }
    var body = ""
    var finite = v === v && a - a === 0
    if (!finite) { body = v !== v ? "nan" : "inf" }
    else if (lowc === 101) { body = lsFmtE(a, sp.prec < 0 ? 6 : sp.prec, sp.alt) }
    else if (lowc === 102) { body = lsFmtF(a, sp.prec < 0 ? 6 : sp.prec, sp.alt) }
    else if (lowc === 103) { body = lsFmtG(a, sp.prec, sp.alt) }
    else { body = lsFmtA(a, sp.prec, sp.alt) }
    if (conv < 97) { body = lsUpper(body) }
    if (finite && lowc === 97 && sp.zero && !sp.left) {
        // The zero padding of %a goes behind the "0x".
        return sign + body.substring(0, 2) + lsZeros(sp.width - sign.length - body.length) + body.substring(2)
    }
    return lsPad(sp, sign, body, finite)
}
// %q of a string: lstrlib's addquoted.
function lsQuoted(s) {
    var out = "\""
    for (var i = 0; i < s.length; i++) {
        var c = s.charCodeAt(i)
        if (c === 34 || c === 92 || c === 10) { out = out + "\\" + s.charAt(i) }
        else if (lsIsCntrl(c)) {
            var nx = i + 1 < s.length ? s.charCodeAt(i + 1) : 0
            var ds = "" + c
            if (lsIsDigit(nx)) { while (ds.length < 3) { ds = "0" + ds } }
            out = out + "\\" + ds
        }
        else { out = out + s.charAt(i) }
    }
    return out + "\""
}
// %q of argument n: a literal Lua reads back as the same value.
function lsLiteral(args, n) {
    var kind = lsFmtLitKind(args, n)
    if (kind == "s") { return lsQuoted(lsFmtCheckStr(args, n)) }
    if (kind == "i") {
        var iv = lsIntDigits(lsFmtInt(args, n))
        // math.mininteger has no decimal literal: its negation does not fit.
        if (iv.neg && iv.x == "8000000000000000") { return "0x8000000000000000" }
        return (iv.neg ? "-" : "") + iv.d
    }
    if (kind == "f") {
        var v = lsFmtNum(args, n)
        if (v !== v) { return "(0/0)" }
        if (v - v !== 0) { return v > 0 ? "1e9999" : "-1e9999" }
        var neg = v < 0 || (v === 0 && 1 / v < 0)
        return (neg ? "-" : "") + lsFmtA(neg ? 0 - v : v, 0 - 1, false)
    }
    if (kind == "t") { return lsFmtToStr(args, n) }
    lsFmtArgErr(n, "value has no literal form")
    return ""
}
function lsFormat(args) {
    var fmt = lsFmtCheckStr(args, 1)
    var out = ""
    var arg = 1
    var i = 0
    while (i < fmt.length) {
        var c = fmt.charCodeAt(i)
        if (c !== 37) { out = out + fmt.charAt(i); i = i + 1; continue }
        i = i + 1
        if (i < fmt.length && fmt.charCodeAt(i) === 37) { out = out + "%"; i = i + 1; continue }
        arg = arg + 1
        if (arg > args.length) { lsFmtArgErr(arg, "no value") }
        // getformat: the span of flags, digits and points, then one more character.
        var j = i
        while (j < fmt.length && (lsIndexOfCh("-+ #0123456789.", fmt.charCodeAt(j)))) { j = j + 1 }
        if (j - i + 1 >= 22) { lsFail("invalid format string to 'format'") }
        var form = "%" + fmt.substring(i, j + 1)
        var conv = j < fmt.length ? fmt.charCodeAt(j) : 0
        i = j + 1
        if (conv === 99) {
            // %c prints the low byte of the integer (C converts it to unsigned char).
            var hx = lsIntDigits(lsFmtInt(args, arg)).x
            lsCheckFormat(form, LS_FMT_C, false)
            var code = LS_HEX.indexOf(hx.charAt(hx.length - 1))
            if (hx.length > 1) { code = code + 16 * LS_HEX.indexOf(hx.charAt(hx.length - 2)) }
            out = out + lsPad(lsSpec(form), "", String.fromCharCode(code), false)
        } else if (conv === 100 || conv === 105 || conv === 117 || conv === 111 || conv === 120 || conv === 88) {
            var iv2 = lsIntDigits(lsFmtInt(args, arg))
            var fl = LS_FMT_X
            if (conv === 100 || conv === 105) { fl = LS_FMT_I } else if (conv === 117) { fl = LS_FMT_U }
            lsCheckFormat(form, fl, true)
            out = out + lsFmtIntConv(lsSpec(form), conv, iv2)
        } else if (conv === 97 || conv === 65 || conv === 101 || conv === 69 || conv === 102 || conv === 103 || conv === 71) {
            var fv = lsFmtNum(args, arg)
            lsCheckFormat(form, LS_FMT_F, true)
            out = out + lsFmtFloat(lsSpec(form), conv, fv)
        } else if (conv === 113) {
            if (form.length > 2) { lsFail("specifier '%q' cannot have modifiers") }
            out = out + lsLiteral(args, arg)
        } else if (conv === 115) {
            var s = lsFmtToStr(args, arg)
            if (form.length == 2) { out = out + s } else {
                // A modifier goes through C's %s, which would stop at a NUL.
                if (s.indexOf(String.fromCharCode(0)) >= 0) { lsFmtArgErr(arg, "string contains zeros") }
                lsCheckFormat(form, LS_FMT_C, true)
                var sp = lsSpec(form)
                if (sp.prec < 0 && s.length >= 100) { out = out + s } else {
                    if (sp.prec >= 0 && s.length > sp.prec) { s = s.substring(0, sp.prec) }
                    out = out + lsPad(sp, "", s, false)
                }
            }
        } else {
            lsFail("invalid conversion '" + form + "' to 'format'")
        }
    }
    return out
}
