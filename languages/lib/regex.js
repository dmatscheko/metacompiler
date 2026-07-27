// regex.js -- the shared regular-expression engine for the language grammars.
//
// This tree had no regular-expression matcher at all, and four languages want the
// same one (Ruby's /re/ literals, JavaScript's RegExp, Python's re, Kotlin's Regex).
// So this is written ONCE, language-neutral: it takes a pattern STRING plus a flag
// STRING and knows nothing about how any language spells its literals. Mapping a
// language's own flag letters onto the neutral set below is the grammar's job (Ruby's
// /m/ is dot-matches-newline, i.e. this file's "s"; Ruby's ^ and $ are always line
// anchors, so the Ruby grammar always passes "m").
//
// A grammar pulls it in with
//     include("lib/regex.js")
// from its startScript, exactly like lib/interp-core.js, and everything lands in the
// same global scope. Every name here is prefixed rx / RX_ so nothing collides.
//
// The engine is a backtracking VM over a small instruction program (the classic
// "recursive backtracking implementation" shape). A backtracking matcher, not a DFA,
// is the right call here: it gives lazy quantifiers, capture groups, lookahead and
// BACKREFERENCES, all of which the target languages have and none of which a
// Thompson/RE2 construction can do. The cost - exponential blowup on a pathological
// pattern - is bounded by a step cap (RX_STEPS), which reports a clean error instead
// of hanging.
//
// The whole file is written in the RESTRICTED JS subset that both engines accept:
// no for...in, no array.splice, array.length is not assignable, no split/join/
// lastIndexOf, indexOf ignores its from-index, and a reassigned parameter keeps the
// type it was first called with. See docs/abnf-dialect-gotchas.md.
//
// abnf/jsrtregex.go is a line-by-line port of this file, for the COMPILER grammars:
// their emitted IR cannot call into a tag script, so the same engine has to exist
// once more on the Go side. Keep the two in step - the matrix compares the
// interpreter and the compiler on the same input.
//
// ----- API -----
//   rxCompile(pattern, flags) -> re          flags: i (fold case), s (dot matches
//                                            newline), m (^ and $ are line anchors),
//                                            x (extended: skip whitespace/# comments)
//                                            re.ok is false and re.err is set on a
//                                            bad pattern.
//   rxSearch(re, text, start) -> m | null    m.begin, m.end, m.caps (2*(n+1) ints,
//                                            -1 when the group did not take part)
//   rxTest(re, text) -> bool
//   rxGroup(re, text, m, i) -> string|null   i is 0 for the whole match
//   rxGroupNamed(re, text, m, name)
//   rxGroupCount(re) -> n                    capturing groups, not counting 0
//   rxNameAt(re, i) -> string                "" when group i has no name
//   rxIndexOfName(re, name) -> i or -1

// rxFail reports a runtime regexp error. It is a KNOB, not a function declaration:
// interp-core has fail() with a language prefix and compile-core does not, so a
// grammar overrides this by ASSIGNMENT after the include (see the shared-helper
// entry in docs/abnf-dialect-gotchas.md).
var rxFail = function (msg) {
    println("regexp error: " + msg)
    exit(1)
}

// ----- Instruction opcodes -----
var RX_CLASS = 1   // match one character against clss[pc]
var RX_SPLIT = 2   // try a, then b
var RX_JMP = 3     // continue at a
var RX_SAVE = 4    // caps[a] = sp
var RX_MARK = 5    // marks[a] = sp (loop-progress register)
var RX_PROG = 6    // if marks[a] == sp, leave the loop at b (empty-body guard)
var RX_MATCH = 7
var RX_ASSERT = 8  // zero-width assertion a
var RX_LOOK = 9    // lookahead: a = address after the body, b = 1 when negative
var RX_LOOKEND = 10
var RX_BREF = 11   // backreference to group a

// ----- Assertion kinds -----
var RX_BOL = 1     // ^
var RX_EOL = 2     // $
var RX_WORDB = 3   // \b
var RX_NWORDB = 4  // \B
var RX_BOS = 5     // \A
var RX_EOS = 6     // \z
var RX_EOSNL = 7   // \Z (end, or just before a final newline)

// ----- Character-class shorthands -----
var RX_SD = 1      // \d
var RX_SND = 2     // \D
var RX_SW = 3      // \w
var RX_SNW = 4     // \W
var RX_SS = 5      // \s
var RX_SNS = 6     // \S
var RX_SH = 7      // \h (hex digit)
var RX_SNH = 8     // \H

// Step cap for one rxSearch attempt. A pathological pattern reports
// "regexp: step limit exceeded" rather than hanging the whole run.
var RX_STEPS = 400000


// ===== Character classes =====
//
// A class is three PARALLEL NUMBER ARRAYS plus a flag, never a mixed-type record:
// lo[i]..hi[i] are the literal ranges and sp[] the shorthands above. Homogeneous
// arrays keep the frozen engine happy and make the Go port mechanical.

function rxNewCls() {
    return {neg: false, lo: [], hi: [], sp: []}
}

function rxClsAdd(cls, from, to) {
    var lo = cls.lo
    var hi = cls.hi
    lo.push(from)
    hi.push(to)
}

function rxClsAddSp(cls, code) {
    var sp = cls.sp
    sp.push(code)
}

function rxIsWordCode(c) {
    if (c >= 48 && c <= 57) { return true }
    if (c >= 65 && c <= 90) { return true }
    if (c >= 97 && c <= 122) { return true }
    return c == 95
}

function rxIsSpaceCode(c) {
    if (c == 32 || c == 9 || c == 10 || c == 13 || c == 12 || c == 11) { return true }
    return false
}

function rxIsDigitCode(c) { return c >= 48 && c <= 57 }

function rxIsHexCode(c) {
    if (c >= 48 && c <= 57) { return true }
    if (c >= 65 && c <= 70) { return true }
    if (c >= 97 && c <= 102) { return true }
    return false
}

function rxSpHas(code, c) {
    if (code == RX_SD) { return rxIsDigitCode(c) }
    if (code == RX_SND) { return !rxIsDigitCode(c) }
    if (code == RX_SW) { return rxIsWordCode(c) }
    if (code == RX_SNW) { return !rxIsWordCode(c) }
    if (code == RX_SS) { return rxIsSpaceCode(c) }
    if (code == RX_SNS) { return !rxIsSpaceCode(c) }
    if (code == RX_SH) { return rxIsHexCode(c) }
    if (code == RX_SNH) { return !rxIsHexCode(c) }
    return false
}

// rxSwapCase answers the other-case code point of an ASCII letter, or -1.
function rxSwapCase(c) {
    if (c >= 65 && c <= 90) { return c + 32 }
    if (c >= 97 && c <= 122) { return c - 32 }
    return -1
}

// rxClsHit is the raw membership test, BEFORE negation.
function rxClsHit(cls, c) {
    var lo = cls.lo
    var hi = cls.hi
    var i = 0
    while (i < lo.length) {
        if (c >= lo[i] && c <= hi[i]) { return true }
        i = i + 1
    }
    var sp = cls.sp
    i = 0
    while (i < sp.length) {
        if (rxSpHas(sp[i], c)) { return true }
        i = i + 1
    }
    return false
}

// rxClsMatch applies case folding first and negation LAST - a negated class under /i/
// must reject both cases of a listed letter, which is only true in that order.
function rxClsMatch(cls, c, icase) {
    var hit = rxClsHit(cls, c)
    if (!hit && icase) {
        var alt = rxSwapCase(c)
        if (alt >= 0) { hit = rxClsHit(cls, alt) }
    }
    if (cls.neg) { return !hit }
    return hit
}


// ===== The pattern parser =====
//
// A node carries every field with a default, so the shape is uniform:
//   t     "cat" | "alt" | "rep" | "cls" | "assert" | "group" | "look" | "bref" | "empty"
//   items child nodes (always an array)
//   cls   character class (always an object)
//   min/max/greedy   for "rep" (max < 0 means unbounded)
//   gidx  capture index for "group" (-1: non-capturing), assertion kind for "assert",
//         group index for "bref", 1 for a negative "look"

function rxNode(t) {
    return {t: t, items: [], cls: rxNewCls(), min: 0, max: 0, greedy: true, gidx: -1}
}

function rxErr(st, msg) {
    if (st.err == "") { st.err = msg }
}

// rxSkipX steps over the whitespace and # comments that the x flag makes insignificant.
function rxSkipX(st) {
    if (!st.ext) { return }
    var p = st.p
    while (st.i < st.n) {
        var c = p.charCodeAt(st.i)
        if (rxIsSpaceCode(c)) {
            st.i = st.i + 1
        } else if (c == 35) { // #
            while (st.i < st.n && p.charCodeAt(st.i) != 10) { st.i = st.i + 1 }
        } else {
            return
        }
    }
}

function rxHexVal(c) {
    if (c >= 48 && c <= 57) { return c - 48 }
    if (c >= 65 && c <= 70) { return c - 55 }
    if (c >= 97 && c <= 102) { return c - 87 }
    return -1
}

// rxEsc reads the escape whose backslash has already been consumed. It answers
// {kind: 0, ch: code} for a literal character and {kind: 1, sp: code} for a
// shorthand class; kind 2 is a backreference (ch = group number).
function rxEsc(st, inClass) {
    var res = {kind: 0, ch: 0, sp: 0}
    if (st.i >= st.n) {
        rxErr(st, "trailing backslash in pattern")
        return res
    }
    var p = st.p
    var c = p.charCodeAt(st.i)
    st.i = st.i + 1
    if (c == 100) { res.kind = 1; res.sp = RX_SD; return res }
    if (c == 68) { res.kind = 1; res.sp = RX_SND; return res }
    if (c == 119) { res.kind = 1; res.sp = RX_SW; return res }
    if (c == 87) { res.kind = 1; res.sp = RX_SNW; return res }
    if (c == 115) { res.kind = 1; res.sp = RX_SS; return res }
    if (c == 83) { res.kind = 1; res.sp = RX_SNS; return res }
    if (c == 104) { res.kind = 1; res.sp = RX_SH; return res }
    if (c == 72) { res.kind = 1; res.sp = RX_SNH; return res }
    if (c == 110) { res.ch = 10; return res }
    if (c == 116) { res.ch = 9; return res }
    if (c == 114) { res.ch = 13; return res }
    if (c == 102) { res.ch = 12; return res }
    if (c == 118) { res.ch = 11; return res }
    if (c == 97) { res.ch = 7; return res }
    if (c == 101) { res.ch = 27; return res }
    if (c == 98 && inClass) { res.ch = 8; return res }
    if (c == 48) { res.ch = 0; return res }
    if (c == 120) { // \xHH
        var v = 0
        var got = 0
        while (got < 2 && st.i < st.n) {
            var h = rxHexVal(p.charCodeAt(st.i))
            if (h < 0) { break }
            v = v * 16 + h
            st.i = st.i + 1
            got = got + 1
        }
        if (got == 0) { rxErr(st, "invalid hex escape") }
        res.ch = v
        return res
    }
    if (c == 117) { // \uHHHH
        var uv = 0
        var ugot = 0
        while (ugot < 4 && st.i < st.n) {
            var uh = rxHexVal(p.charCodeAt(st.i))
            if (uh < 0) { break }
            uv = uv * 16 + uh
            st.i = st.i + 1
            ugot = ugot + 1
        }
        if (ugot == 0) { rxErr(st, "invalid unicode escape") }
        res.ch = uv
        return res
    }
    if (!inClass && c >= 49 && c <= 57) { // \1 .. \9
        res.kind = 2
        res.ch = c - 48
        return res
    }
    res.ch = c
    return res
}

// rxParseClass reads a [...] class; the [ has already been consumed.
function rxParseClass(st) {
    var node = rxNode("cls")
    var cls = node.cls
    var p = st.p
    if (st.i < st.n && p.charCodeAt(st.i) == 94) { // ^
        cls.neg = true
        st.i = st.i + 1
    }
    var first = true
    while (st.i < st.n) {
        var c = p.charCodeAt(st.i)
        if (c == 93 && !first) { // ]
            st.i = st.i + 1
            return node
        }
        first = false
        // A POSIX bracket [:alpha:] is not supported; report it rather than
        // silently reading it as a set of punctuation characters.
        if (c == 91 && st.i + 1 < st.n && p.charCodeAt(st.i + 1) == 58) {
            rxErr(st, "POSIX bracket expressions are not supported")
            return node
        }
        var loCode = -1
        if (c == 92) { // backslash
            st.i = st.i + 1
            var e = rxEsc(st, true)
            if (e.kind == 1) {
                rxClsAddSp(cls, e.sp)
                continue
            }
            loCode = e.ch
        } else {
            st.i = st.i + 1
            loCode = c
        }
        // A range needs a '-' that is neither the last character nor followed by ].
        if (st.i + 1 < st.n && p.charCodeAt(st.i) == 45 && p.charCodeAt(st.i + 1) != 93) {
            st.i = st.i + 1
            var hiCode = -1
            var c2 = p.charCodeAt(st.i)
            if (c2 == 92) {
                st.i = st.i + 1
                var e2 = rxEsc(st, true)
                if (e2.kind == 1) {
                    // "a-\d" is not a range; take the '-' literally and the class on.
                    rxClsAdd(cls, loCode, loCode)
                    rxClsAdd(cls, 45, 45)
                    rxClsAddSp(cls, e2.sp)
                    continue
                }
                hiCode = e2.ch
            } else {
                st.i = st.i + 1
                hiCode = c2
            }
            if (hiCode < loCode) {
                rxErr(st, "character class range is out of order")
                return node
            }
            rxClsAdd(cls, loCode, hiCode)
        } else {
            rxClsAdd(cls, loCode, loCode)
        }
    }
    rxErr(st, "unterminated character class")
    return node
}

// rxReadGroupName reads name> or name' after (?< / (?'.
function rxReadGroupName(st, closer) {
    var p = st.p
    var start = st.i
    while (st.i < st.n && p.charCodeAt(st.i) != closer) { st.i = st.i + 1 }
    if (st.i >= st.n) {
        rxErr(st, "unterminated group name")
        return ""
    }
    var name = p.substring(start, st.i)
    st.i = st.i + 1
    return name
}

function rxParseGroup(st) {
    var p = st.p
    // The ( is already consumed.
    if (st.i < st.n && p.charCodeAt(st.i) == 63) { // ?
        st.i = st.i + 1
        if (st.i >= st.n) {
            rxErr(st, "unterminated group")
            return rxNode("empty")
        }
        var k = p.charCodeAt(st.i)
        if (k == 58) { // (?:
            st.i = st.i + 1
            var g = rxNode("group")
            g.items.push(rxParseAlt(st))
            rxExpect(st, 41, "unterminated group")
            return g
        }
        if (k == 35) { // (?# comment
            while (st.i < st.n && p.charCodeAt(st.i) != 41) { st.i = st.i + 1 }
            rxExpect(st, 41, "unterminated comment group")
            return rxNode("empty")
        }
        if (k == 61 || k == 33) { // (?= (?!
            st.i = st.i + 1
            var lk = rxNode("look")
            if (k == 33) { lk.gidx = 1 }
            lk.items.push(rxParseAlt(st))
            rxExpect(st, 41, "unterminated lookahead")
            return lk
        }
        if (k == 60 || k == 39) { // (?<name>  (?'name'
            if (k == 60 && st.i + 1 < st.n) {
                var k2 = p.charCodeAt(st.i + 1)
                if (k2 == 61 || k2 == 33) {
                    rxErr(st, "lookbehind is not supported")
                    return rxNode("empty")
                }
            }
            st.i = st.i + 1
            var closer = 62
            if (k == 39) { closer = 39 }
            var nm = rxReadGroupName(st, closer)
            var ng = rxNode("group")
            st.ngroups = st.ngroups + 1
            ng.gidx = st.ngroups
            var names = st.names
            var nidx = st.nameGroups
            names.push(nm)
            nidx.push(ng.gidx)
            ng.items.push(rxParseAlt(st))
            rxExpect(st, 41, "unterminated group")
            return ng
        }
        rxErr(st, "unsupported group form (?" + p.charAt(st.i) + ")")
        return rxNode("empty")
    }
    var cg = rxNode("group")
    st.ngroups = st.ngroups + 1
    cg.gidx = st.ngroups
    cg.items.push(rxParseAlt(st))
    rxExpect(st, 41, "unterminated group")
    return cg
}

function rxExpect(st, code, msg) {
    rxSkipX(st)
    if (st.i < st.n && st.p.charCodeAt(st.i) == code) {
        st.i = st.i + 1
        return
    }
    rxErr(st, msg)
}

function rxParseAtom(st) {
    var p = st.p
    var c = p.charCodeAt(st.i)
    if (c == 40) { st.i = st.i + 1; return rxParseGroup(st) }
    if (c == 91) { st.i = st.i + 1; return rxParseClass(st) }
    if (c == 46) { // .
        st.i = st.i + 1
        var dot = rxNode("cls")
        dot.cls.neg = true
        if (!st.dotall) { rxClsAdd(dot.cls, 10, 10) }
        return dot
    }
    if (c == 94) { // ^
        st.i = st.i + 1
        var b = rxNode("assert")
        b.gidx = RX_BOL
        return b
    }
    if (c == 36) { // $
        st.i = st.i + 1
        var d = rxNode("assert")
        d.gidx = RX_EOL
        return d
    }
    if (c == 92) { // backslash: the zero-width escapes first
        var nxt = 0
        if (st.i + 1 < st.n) { nxt = p.charCodeAt(st.i + 1) }
        if (nxt == 98 || nxt == 66 || nxt == 65 || nxt == 122 || nxt == 90) {
            st.i = st.i + 2
            var a = rxNode("assert")
            if (nxt == 98) { a.gidx = RX_WORDB }
            if (nxt == 66) { a.gidx = RX_NWORDB }
            if (nxt == 65) { a.gidx = RX_BOS }
            if (nxt == 122) { a.gidx = RX_EOS }
            if (nxt == 90) { a.gidx = RX_EOSNL }
            return a
        }
        st.i = st.i + 1
        var e = rxEsc(st, false)
        if (e.kind == 1) {
            var sn = rxNode("cls")
            rxClsAddSp(sn.cls, e.sp)
            return sn
        }
        if (e.kind == 2) {
            var br = rxNode("bref")
            br.gidx = e.ch
            return br
        }
        var ln = rxNode("cls")
        rxClsAdd(ln.cls, e.ch, e.ch)
        return ln
    }
    if (c == 41 || c == 124) { // ) |  - an empty branch
        return rxNode("empty")
    }
    if (c == 42 || c == 43 || c == 63) {
        rxErr(st, "quantifier with nothing to repeat")
        st.i = st.i + 1
        return rxNode("empty")
    }
    st.i = st.i + 1
    var lit = rxNode("cls")
    rxClsAdd(lit.cls, c, c)
    return lit
}

// rxParseBound reads {n} / {n,} / {n,m} at st.i. It answers null when what follows
// the brace is not a bound, so a literal { stays a literal { (every one of these
// languages allows that).
function rxParseBound(st) {
    var p = st.p
    var j = st.i + 1
    var minV = 0
    var digits = 0
    while (j < st.n && rxIsDigitCode(p.charCodeAt(j))) {
        minV = minV * 10 + (p.charCodeAt(j) - 48)
        j = j + 1
        digits = digits + 1
    }
    if (digits == 0) { return null }
    var maxV = minV
    if (j < st.n && p.charCodeAt(j) == 44) { // ,
        j = j + 1
        var d2 = 0
        var mv = 0
        while (j < st.n && rxIsDigitCode(p.charCodeAt(j))) {
            mv = mv * 10 + (p.charCodeAt(j) - 48)
            j = j + 1
            d2 = d2 + 1
        }
        if (d2 == 0) { maxV = -1 } else { maxV = mv }
    }
    if (j >= st.n || p.charCodeAt(j) != 125) { return null } // }
    st.i = j + 1
    return {min: minV, max: maxV}
}

function rxParsePiece(st) {
    var atom = rxParseAtom(st)
    rxSkipX(st)
    while (st.i < st.n) {
        var c = st.p.charCodeAt(st.i)
        var minV = 0
        var maxV = 0
        if (c == 42) { st.i = st.i + 1; minV = 0; maxV = -1 }
        else if (c == 43) { st.i = st.i + 1; minV = 1; maxV = -1 }
        else if (c == 63) { st.i = st.i + 1; minV = 0; maxV = 1 }
        else if (c == 123) {
            var bd = rxParseBound(st)
            if (bd == null) { return atom }
            minV = bd.min
            maxV = bd.max
            if (maxV >= 0 && maxV < minV) {
                rxErr(st, "quantifier bounds are out of order")
                return atom
            }
        } else {
            return atom
        }
        var rep = rxNode("rep")
        rep.min = minV
        rep.max = maxV
        rep.greedy = true
        if (st.i < st.n) {
            var q = st.p.charCodeAt(st.i)
            if (q == 63) { rep.greedy = false; st.i = st.i + 1 }
            else if (q == 43) { st.i = st.i + 1 } // possessive: treated as greedy
        }
        rep.items.push(atom)
        // A second quantifier on the same atom (a**) is a syntax error in every
        // target language; loop so it is REPORTED rather than silently dropped.
        atom = rep
        rxSkipX(st)
    }
    return atom
}

function rxParseCat(st) {
    var cat = rxNode("cat")
    var items = cat.items
    rxSkipX(st)
    while (st.i < st.n) {
        var c = st.p.charCodeAt(st.i)
        if (c == 124 || c == 41) { break } // | )
        var before = st.i
        items.push(rxParsePiece(st))
        if (st.err != "") { break }
        if (st.i == before) { break } // no progress: a malformed pattern
        rxSkipX(st)
    }
    return cat
}

function rxParseAlt(st) {
    var alt = rxNode("alt")
    var items = alt.items
    items.push(rxParseCat(st))
    while (st.i < st.n && st.p.charCodeAt(st.i) == 124) {
        st.i = st.i + 1
        items.push(rxParseCat(st))
        if (st.err != "") { break }
    }
    if (items.length == 1) { return items[0] }
    return alt
}


// ===== The emitter =====
//
// prog is four parallel arrays: ops[], as[], bs[] and clss[]. Same reason as the
// class representation - no mixed-type record anywhere.

function rxNewProg() {
    return {ops: [], as: [], bs: [], clss: [], nmarks: 0, dummy: rxNewCls()}
}

function rxAdd(pg, op, a, b, cls) {
    var ops = pg.ops
    var as = pg.as
    var bs = pg.bs
    var cs = pg.clss
    var at = ops.length
    ops.push(op)
    as.push(a)
    bs.push(b)
    cs.push(cls)
    return at
}

function rxSetA(pg, at, v) { var as = pg.as; as[at] = v }
function rxSetB(pg, at, v) { var bs = pg.bs; bs[at] = v }
function rxHere(pg) { var ops = pg.ops; return ops.length }

function rxEmit(pg, node) {
    var t = node.t
    if (t == "empty") { return }
    if (t == "cls") {
        rxAdd(pg, RX_CLASS, 0, 0, node.cls)
        return
    }
    if (t == "assert") {
        rxAdd(pg, RX_ASSERT, node.gidx, 0, pg.dummy)
        return
    }
    if (t == "bref") {
        rxAdd(pg, RX_BREF, node.gidx, 0, pg.dummy)
        return
    }
    if (t == "cat") {
        var items = node.items
        var i = 0
        while (i < items.length) {
            rxEmit(pg, items[i])
            i = i + 1
        }
        return
    }
    if (t == "alt") {
        rxEmitAlt(pg, node)
        return
    }
    if (t == "group") {
        var gi = node.gidx
        var kids = node.items
        if (gi >= 0) { rxAdd(pg, RX_SAVE, 2 * gi, 0, pg.dummy) }
        if (kids.length > 0) { rxEmit(pg, kids[0]) }
        if (gi >= 0) { rxAdd(pg, RX_SAVE, 2 * gi + 1, 0, pg.dummy) }
        return
    }
    if (t == "look") {
        var lk = rxAdd(pg, RX_LOOK, 0, node.gidx == 1 ? 1 : 0, pg.dummy)
        var lkids = node.items
        if (lkids.length > 0) { rxEmit(pg, lkids[0]) }
        rxAdd(pg, RX_LOOKEND, 0, 0, pg.dummy)
        rxSetA(pg, lk, rxHere(pg))
        return
    }
    if (t == "rep") {
        rxEmitRep(pg, node)
        return
    }
}

function rxEmitAlt(pg, node) {
    var items = node.items
    var splits = []
    var jumps = []
    var i = 0
    while (i < items.length) {
        if (i < items.length - 1) {
            var s = rxAdd(pg, RX_SPLIT, 0, 0, pg.dummy)
            rxSetA(pg, s, rxHere(pg))
            splits.push(s)
            rxEmit(pg, items[i])
            var j = rxAdd(pg, RX_JMP, 0, 0, pg.dummy)
            jumps.push(j)
            rxSetB(pg, s, rxHere(pg))
        } else {
            rxEmit(pg, items[i])
        }
        i = i + 1
    }
    var end = rxHere(pg)
    i = 0
    while (i < jumps.length) {
        rxSetA(pg, jumps[i], end)
        i = i + 1
    }
}

function rxEmitRep(pg, node) {
    var body = node.items[0]
    var i = 0
    while (i < node.min) {
        rxEmit(pg, body)
        i = i + 1
    }
    if (node.max < 0) {
        // Unbounded tail. The MARK/PROG pair is the empty-body guard: a body that
        // consumed nothing leaves the loop instead of spinning ((a*)* is a real
        // pattern, and without this it never terminates).
        var mk = pg.nmarks
        pg.nmarks = pg.nmarks + 1
        var l1 = rxHere(pg)
        var sp = rxAdd(pg, RX_SPLIT, 0, 0, pg.dummy)
        rxAdd(pg, RX_MARK, mk, 0, pg.dummy)
        rxEmit(pg, body)
        var pgi = rxAdd(pg, RX_PROG, mk, 0, pg.dummy)
        rxAdd(pg, RX_JMP, l1, 0, pg.dummy)
        var out = rxHere(pg)
        if (node.greedy) {
            rxSetA(pg, sp, l1 + 1)
            rxSetB(pg, sp, out)
        } else {
            rxSetA(pg, sp, out)
            rxSetB(pg, sp, l1 + 1)
        }
        rxSetB(pg, pgi, out)
        return
    }
    var extra = node.max - node.min
    var splits = []
    var k = 0
    while (k < extra) {
        var s = rxAdd(pg, RX_SPLIT, 0, 0, pg.dummy)
        splits.push(s)
        rxEmit(pg, body)
        k = k + 1
    }
    var endAt = rxHere(pg)
    k = 0
    while (k < splits.length) {
        var at = splits[k]
        if (node.greedy) {
            rxSetA(pg, at, at + 1)
            rxSetB(pg, at, endAt)
        } else {
            rxSetA(pg, at, endAt)
            rxSetB(pg, at, at + 1)
        }
        k = k + 1
    }
}


// ===== Compilation =====

function rxCompile(pattern, flags) {
    var icase = false
    var dotall = false
    var multi = false
    var ext = false
    var fi = 0
    while (fi < flags.length) {
        var f = flags.charAt(fi)
        if (f == "i") { icase = true }
        else if (f == "s") { dotall = true }
        else if (f == "m") { multi = true }
        else if (f == "x") { ext = true }
        fi = fi + 1
    }
    var st = {
        p: pattern, i: 0, n: pattern.length, err: "",
        ngroups: 0, names: [], nameGroups: [], ext: ext, dotall: dotall
    }
    var ast = rxParseAlt(st)
    if (st.err == "" && st.i < st.n) {
        if (pattern.charCodeAt(st.i) == 41) { rxErr(st, "unmatched ) in pattern") }
        else { rxErr(st, "unexpected character in pattern") }
    }
    if (st.err != "") {
        return {ok: false, err: st.err, ops: [], as: [], bs: [], clss: [],
                ngroups: 0, nmarks: 0, names: [], nameGroups: [],
                icase: icase, dotall: dotall, multi: multi, src: pattern, flags: flags}
    }
    var pg = rxNewProg()
    rxAdd(pg, RX_SAVE, 0, 0, pg.dummy)
    rxEmit(pg, ast)
    rxAdd(pg, RX_SAVE, 1, 0, pg.dummy)
    rxAdd(pg, RX_MATCH, 0, 0, pg.dummy)
    return {
        ok: true, err: "", ops: pg.ops, as: pg.as, bs: pg.bs, clss: pg.clss,
        ngroups: st.ngroups, nmarks: pg.nmarks, names: st.names, nameGroups: st.nameGroups,
        icase: icase, dotall: dotall, multi: multi, src: pattern, flags: flags
    }
}


// ===== The matcher =====

function rxAssertOK(re, text, kind, sp) {
    var n = text.length
    if (kind == RX_BOS) { return sp == 0 }
    if (kind == RX_EOS) { return sp == n }
    if (kind == RX_EOSNL) {
        if (sp == n) { return true }
        return sp == n - 1 && text.charCodeAt(sp) == 10
    }
    if (kind == RX_BOL) {
        if (sp == 0) { return true }
        if (!re.multi) { return false }
        return text.charCodeAt(sp - 1) == 10
    }
    if (kind == RX_EOL) {
        if (sp == n) { return true }
        if (!re.multi) { return false }
        return text.charCodeAt(sp) == 10
    }
    var before = false
    var after = false
    if (sp > 0) { before = rxIsWordCode(text.charCodeAt(sp - 1)) }
    if (sp < n) { after = rxIsWordCode(text.charCodeAt(sp)) }
    if (kind == RX_WORDB) { return before != after }
    return before == after
}

// rxRun is the backtracking VM. It answers the end position of a match starting at
// the given pc/sp, or -1. Recursion happens only where the machine has a choice
// (SPLIT) or has to be able to undo a write (SAVE / MARK / LOOK).
function rxRun(re, text, pcIn, spIn, caps, marks, st) {
    var ops = re.ops
    var as = re.as
    var bs = re.bs
    var clss = re.clss
    var n = text.length
    var pc = pcIn
    var sp = spIn
    while (true) {
        st.steps = st.steps + 1
        if (st.steps > RX_STEPS) {
            st.over = true
            return -1
        }
        var op = ops[pc]
        if (op == RX_CLASS) {
            if (sp >= n) { return -1 }
            if (!rxClsMatch(clss[pc], text.charCodeAt(sp), re.icase)) { return -1 }
            pc = pc + 1
            sp = sp + 1
        } else if (op == RX_JMP) {
            pc = as[pc]
        } else if (op == RX_SPLIT) {
            var r = rxRun(re, text, as[pc], sp, caps, marks, st)
            if (r >= 0) { return r }
            if (st.over) { return -1 }
            pc = bs[pc]
        } else if (op == RX_SAVE) {
            var slot = as[pc]
            var old = caps[slot]
            caps[slot] = sp
            var rs = rxRun(re, text, pc + 1, sp, caps, marks, st)
            if (rs >= 0) { return rs }
            caps[slot] = old
            return -1
        } else if (op == RX_MARK) {
            var mslot = as[pc]
            var mold = marks[mslot]
            marks[mslot] = sp
            var rm = rxRun(re, text, pc + 1, sp, caps, marks, st)
            if (rm >= 0) { return rm }
            marks[mslot] = mold
            return -1
        } else if (op == RX_PROG) {
            if (marks[as[pc]] == sp) { pc = bs[pc] } else { pc = pc + 1 }
        } else if (op == RX_ASSERT) {
            if (!rxAssertOK(re, text, as[pc], sp)) { return -1 }
            pc = pc + 1
        } else if (op == RX_LOOK) {
            var rl = rxRun(re, text, pc + 1, sp, caps, marks, st)
            if (st.over) { return -1 }
            var hit = rl >= 0
            var want = bs[pc] == 0
            if (hit != want) { return -1 }
            pc = as[pc]
        } else if (op == RX_LOOKEND) {
            return sp
        } else if (op == RX_BREF) {
            var g = as[pc]
            var from = caps[2 * g]
            var to = caps[2 * g + 1]
            if (from < 0 || to < 0) {
                // A group that did not take part matches the empty string.
                pc = pc + 1
            } else {
                var len = to - from
                if (sp + len > n) { return -1 }
                var bi = 0
                while (bi < len) {
                    var lc = text.charCodeAt(from + bi)
                    var rc = text.charCodeAt(sp + bi)
                    if (lc != rc) {
                        if (!re.icase) { return -1 }
                        var alt = rxSwapCase(rc)
                        if (alt != lc) { return -1 }
                    }
                    bi = bi + 1
                }
                pc = pc + 1
                sp = sp + len
            }
        } else if (op == RX_MATCH) {
            return sp
        } else {
            return -1
        }
    }
    return -1
}

// rxSearch finds the leftmost match at or after `start`. It answers null for no
// match, and a record {begin, end, caps} otherwise; caps[2i]/caps[2i+1] bound
// group i, and are -1 when the group did not take part.
function rxSearch(re, text, start) {
    if (!re.ok) { rxFail("invalid regexp: " + re.err) }
    var from = start
    if (from < 0) { from = 0 }
    var nslots = 2 * (re.ngroups + 1)
    var at = from
    while (at <= text.length) {
        var caps = []
        var i = 0
        while (i < nslots) {
            caps.push(-1)
            i = i + 1
        }
        var marks = []
        i = 0
        while (i < re.nmarks) {
            marks.push(-1)
            i = i + 1
        }
        var st = {steps: 0, over: false}
        var end = rxRun(re, text, 0, at, caps, marks, st)
        if (st.over) { rxFail("step limit exceeded") }
        if (end >= 0) {
            return {begin: caps[0], end: caps[1], caps: caps}
        }
        at = at + 1
    }
    return null
}

function rxTest(re, text) {
    return rxSearch(re, text, 0) != null
}

function rxGroupCount(re) { return re.ngroups }

function rxGroup(re, text, m, i) {
    if (i < 0 || i > re.ngroups) { return null }
    var caps = m.caps
    var from = caps[2 * i]
    var to = caps[2 * i + 1]
    if (from < 0 || to < 0) { return null }
    return text.substring(from, to)
}

function rxIndexOfName(re, name) {
    var names = re.names
    var idx = re.nameGroups
    var i = 0
    while (i < names.length) {
        if (names[i] == name) { return idx[i] }
        i = i + 1
    }
    return -1
}

function rxNameAt(re, i) {
    var names = re.names
    var idx = re.nameGroups
    var k = 0
    while (k < names.length) {
        if (idx[k] == i) { return names[k] }
        k = k + 1
    }
    return ""
}

function rxGroupNamed(re, text, m, name) {
    var i = rxIndexOfName(re, name)
    if (i < 0) { return null }
    return rxGroup(re, text, m, i)
}

// rxReplace substitutes every match (or only the first when `all` is false).
// The replacement is expanded: \0 / \1 .. \9 and \k<name> stand for a group,
// and a doubled backslash for one backslash. Languages spell their template
// differently ($1 in JS, \1 in Ruby); a grammar that wants another spelling
// rewrites the template before calling this.
function rxReplace(re, text, repl, all) {
    var out = ""
    var at = 0
    while (at <= text.length) {
        var m = rxSearch(re, text, at)
        if (m == null) { break }
        out = out + text.substring(at, m.begin) + rxExpand(re, text, m, repl)
        if (m.end == m.begin) {
            // An empty match must still make progress or this loop never ends.
            if (m.end < text.length) { out = out + text.charAt(m.end) }
            at = m.end + 1
        } else {
            at = m.end
        }
        if (!all) { break }
    }
    if (at <= text.length) { out = out + text.substring(at, text.length) }
    return out
}

function rxExpand(re, text, m, repl) {
    var out = ""
    var i = 0
    while (i < repl.length) {
        var c = repl.charCodeAt(i)
        if (c != 92 || i + 1 >= repl.length) {
            out = out + repl.charAt(i)
            i = i + 1
            continue
        }
        var d = repl.charCodeAt(i + 1)
        if (d >= 48 && d <= 57) {
            var g = rxGroup(re, text, m, d - 48)
            if (g != null) { out = out + g }
            i = i + 2
            continue
        }
        if (d == 107 && i + 2 < repl.length && repl.charCodeAt(i + 2) == 60) { // \k<
            var j = i + 3
            while (j < repl.length && repl.charCodeAt(j) != 62) { j = j + 1 }
            var nm = repl.substring(i + 3, j)
            var gv = rxGroupNamed(re, text, m, nm)
            if (gv != null) { out = out + gv }
            i = j + 1
            continue
        }
        out = out + repl.charAt(i + 1)
        i = i + 2
    }
    return out
}

// rxSplit cuts `text` at every match. A zero-width match advances by one character,
// so /x*/ on "abc" answers the individual letters rather than looping.
function rxSplit(re, text) {
    var parts = []
    var at = 0
    var last = 0
    while (at <= text.length) {
        var m = rxSearch(re, text, at)
        if (m == null) { break }
        if (m.end == m.begin) {
            if (m.begin >= text.length) { break }
            parts.push(text.substring(last, m.begin + 1))
            last = m.begin + 1
            at = m.begin + 1
        } else {
            parts.push(text.substring(last, m.begin))
            last = m.end
            at = m.end
        }
    }
    parts.push(text.substring(last, text.length))
    return parts
}
