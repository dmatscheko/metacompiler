// regex.js -- the shared regular-expression engine for the language grammars.
//
// Several languages want the same matcher (Ruby's /re/ literals, JavaScript's RegExp,
// Python's re, Kotlin's Regex, bash's [[ =~ ]]), so this is written ONCE, language-neutral: it takes a pattern STRING plus a flag
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
// The engine is a backtracking VM over a small instruction program, with an
// explicit backtrack stack (rxRun). A backtracking matcher, not a DFA,
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
//                                            x (extended: skip whitespace/# comments),
//                                            o (JavaScript Annex B: a backslash-digit
//                                            escape with no such group is a LEGACY
//                                            OCTAL character escape; without o it is
//                                            an "invalid group reference" error, which
//                                            is what Python, Ruby, Java and Kotlin do),
//                                            q (Java/Kotlin \Q...\E quote regions; in
//                                            JavaScript \Q is an identity escape and
//                                            /\Qa.b\E/ matches "Qa.bE", so this is a
//                                            flag and NOT the default)
//                                            e (POSIX ERE dialect - see rxCompile),
//                                            j (JavaScript repeat semantics: an
//                                            iteration of an unbounded loop that
//                                            consumed nothing is DISCARDED rather
//                                            than kept, so /^(a*)*$/ on "aaa"
//                                            reports "aaa" for group 1 the way node
//                                            and POSIX do, not "" the way Perl,
//                                            Python, Ruby and Java do; it also CLEARS
//                                            the captures inside a repeated atom at
//                                            the start of every iteration, so
//                                            /(?:(a)|b)+/ on "ab" leaves group 1
//                                            unset; e implies j),
//                                            p (POSIX bracket expressions [[:alpha:]]
//                                            and nothing else from the e dialect -
//                                            this is Ruby, which has [[:alpha:]] AND
//                                            \d, lazy quantifiers and (?...); e
//                                            implies p)
//                                            J (java.util.regex: counted {n,m}
//                                            loops instead of unrolled copies,
//                                            a bound that is not an int is an
//                                            error, nested classes and &&,
//                                            \p{..}, \h \v \R \cX \0oct \x{h},
//                                            lookbehind, multi-digit \nn and a
//                                            backreference to a group that did
//                                            not take part FAILS; the errors are
//                                            Java's descriptions, positioned in
//                                            re.errAt; inline flags (?idmsuxU-..)
//                                            anywhere and scoped (?..:X), atomic
//                                            groups and possessive quantifiers,
//                                            Java's line terminators for . ^ $ \Z)
//                                            u, d (J only: UNICODE_CASE and
//                                            UNIX_LINES from the start, as the
//                                            inline u and d)
//                                            n (a backreference to a group that did
//                                            not take part FAILS, as in Python and
//                                            Ruby, instead of matching empty; J and
//                                            P imply n)
//                                            P (Python: inline flags (?aiLmsux) at
//                                            the start and scoped (?imsx-imsx:...),
//                                            $ also matches before a final newline,
//                                            \Z is the very end; re.gflags answers
//                                            the inline letters for .flags; under
//                                            J it answers Pattern.flags(), the
//                                            letters in force at the pattern's
//                                            end, as rxFlagsGet spells them;
//                                            \w \W \b \B \d \D \s \S are Python's
//                                            Unicode ones, rxPyWordHit /
//                                            rxPyDigitHit / rxPySpaceHit)
//                                            a (P only: re.ASCII, as the inline a:
//                                            those are ASCII's)
//                                            R (Go's regexp: RE2 syntax with Go's
//                                            error texts, rewritten into J by
//                                            rxRe2, and Go's matching rules - see
//                                            rxRe2; re.names are Go's group names)
//                                            C (PCRE2 as PHP's preg_* compile it:
//                                            the letters after C are PHP's
//                                            modifiers i m s x n U D A u J, not
//                                            these; rewritten into J by rxPcre)
//                                            re.ok is false and re.err is set on a
//                                            bad pattern; re.errAt is the error's
//                                            index into the pattern, or -1.
//   rxSearch(re, text, start) -> m | null    m.begin, m.end, m.caps (2*(n+1) ints,
//                                            -1 when the group did not take part)
//   rxSearchFrom(re, text, start, gpos, glast)   rxSearch with java.util.regex's
//                                            Matcher state for \G (gpos, its
//                                            oldLast) and \b{g} (glast, its last):
//                                            the previous match's end, for a
//                                            Matcher's next find. rxSearch,
//                                            rxMatchAt and rxMatchWhole are a fresh
//                                            Matcher's: gpos the start, glast 0
//   rxMatchAt(re, text, at) -> m | null       the match must BEGIN at `at` (sticky)
//   rxMatchWhole(re, text, at) -> m | null    the match must run from `at` (0 for
//                                            the ordinary case) to the very END.
//                                            NOT a search plus a length check: an
//                                            alternation has to backtrack into its
//                                            longer branch.
//   re.hitEnd, re.reqEnd                      java.util.regex's Matcher.hitEnd
//                                            and requireEnd, accumulated by
//                                            every run until the caller clears
//                                            them: a run read at the end of the
//                                            text, or an anchor matched there that
//                                            more text could break. Meaningful
//                                            under J only; re.jbegin says a failed
//                                            search does not set hitEnd by itself
//                                            (rxBeginJ), re.jmin is Start's
//                                            shortest match (rxSearchFrom).
//   re.jcut                                   0, or under J the low surrogate
//                                            after the text when a Matcher's
//                                            region end splits a pair there,
//                                            which java.util.regex reads (set
//                                            and cleared by the caller)
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
var RX_CLEAR = 12  // caps[2a .. 2b+1] = -1 (per-iteration capture reset; j only)
var RX_CLASSX = 13 // match one character against the class TREE clss[pc] (J only)
var RX_CINIT = 14  // counted loop a starts: its count is 0
var RX_CLOOP = 15  // counted loop a: another iteration, or leave at b
var RX_CINC = 16   // counted loop a: one iteration done, back to b
var RX_LOOKB = 17  // lookbehind: a = address after the body, b = 2 * id + negative
var RX_LBEND = 18  // the lookbehind body must end where the lookbehind stands
var RX_ATOM = 19   // atomic group: a = address after the body (J only)
var RX_ATOMEND = 20
var RX_GPOS = 21   // \G: where the previous match ended (J only)
var RX_XGRAPH = 22 // \X: one extended grapheme cluster (J only)
var RX_GBOUND = 23 // \b{g}: a grapheme cluster boundary, as Java finds one (J only)

// ----- Assertion kinds -----
var RX_BOL = 1     // ^
var RX_EOL = 2     // $
var RX_WORDB = 3   // \b
var RX_NWORDB = 4  // \B
var RX_BOS = 5     // \A
var RX_EOS = 6     // \z
var RX_EOSNL = 7   // \Z (end, or just before a final newline)
var RX_MBOL = 8    // ^ under P with m in force: a line start whatever re.multi says
var RX_MEOL = 9    // $ under P with m in force
// java.util.regex's anchors (J), chosen as each is read: a line terminator is
// \n, \r\n, \r, U+0085, U+2028 or U+2029, or under UNIX_LINES (the U kinds) \n
// alone. Twins of Caret, UnixCaret, Dollar, UnixDollar and Bound in Pattern.java.
var RX_JBOL = 10    // ^ under MULTILINE
var RX_JUBOL = 11
var RX_JEOL = 12    // $ and \Z: the end, or before a final line terminator
var RX_JUEOL = 13
var RX_JMEOL = 14   // $ under MULTILINE: before any line terminator, or the end
var RX_JUMEOL = 15
var RX_JWORDB = 16  // \b: ASCII \w, and a non-spacing mark after a letter or digit
var RX_JNWORDB = 17
var RX_JUWORDB = 18 // \b under UNICODE_CHARACTER_CLASS: Unicode \w
var RX_JUNWORDB = 19
var RX_PUWORDB = 20  // \b under P without a: Python's Unicode \w (rxPyWordHit)
var RX_PUNWORDB = 21

// ----- Character-class shorthands -----
var RX_SD = 1      // \d
var RX_SND = 2     // \D
var RX_SW = 3      // \w
var RX_SNW = 4     // \W
var RX_SS = 5      // \s
var RX_SNS = 6     // \S
var RX_SH = 7      // \h (hex digit)
var RX_SNH = 8     // \H
var RX_SJH = 9     // Java's \h: horizontal whitespace
var RX_SJNH = 10   // Java's \H
var RX_SJV = 11    // Java's \v: vertical whitespace
var RX_SJNV = 12   // Java's \V
var RX_SPW = 13    // Python's Unicode \w (rxPyWordHit)
var RX_SPNW = 14   // Python's Unicode \W
var RX_SPD = 15    // Python's Unicode \d (rxPyDigitHit)
var RX_SPND = 16   // Python's Unicode \D
var RX_SPS = 17    // Python's Unicode \s (rxPySpaceHit)
var RX_SPNS = 18   // Python's Unicode \S

// Step cap for one rxSearch attempt. A pathological pattern reports
// "regexp: step limit exceeded" rather than hanging the whole run.
var RX_STEPS = 400000


// ===== Character classes =====
//
// A class is three PARALLEL NUMBER ARRAYS plus a flag, never a mixed-type record:
// lo[i]..hi[i] are the literal ranges and sp[] the shorthands above. Homogeneous
// arrays keep the frozen engine happy and make the Go port mechanical.
//
// Java's classes nest and intersect, so under J a class is a TREE: subs[] are
// nested classes in the union (each with its own neg), ands[] the operands of
// &&, and props[] the \p{..} properties rxPropHit decides. x marks a class that
// uses any of the three; it compiles to RX_CLASSX, so a flat class keeps the
// plain RX_CLASS path. u marks ranges read under Java's (?iu): they also take
// a character whose simple uppercase, or that uppercase's lowercase, is in
// range (Pattern.java's CIRangeU and SingleU).

function rxNewCls() {
    return {neg: false, lo: [], hi: [], sp: [], subs: [], ands: [], props: [], x: false, u: false, pu: false,
            cpg: false}
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

// rxCpBackN steps rxCpBack k times from i, or answers -1 past the start.
function rxCpBackN(s, i, k) {
    var j = i
    var m = 0
    while (m < k) {
        if (j <= 0) { return 0 - 1 }
        j = rxCpBack(s, j)
        m = m + 1
    }
    return j
}

// rxCpBack is where the code point ending just before UTF-16 index i of s starts:
// i - 2 after a surrogate pair, else i - 1.
function rxCpBack(s, i) {
    if (i >= 2) {
        var d = s.charCodeAt(i - 1)
        var c = s.charCodeAt(i - 2)
        if (d >= 56320 && d <= 57343 && c >= 55296 && c <= 56319) { return i - 2 }
    }
    return i - 1
}

// rxCpAt is the code point at UTF-16 index i of s, a surrogate pair read as one.
// The J and P dialects read both pattern and text this way, as java.util.regex
// and CPython read code points; a lone surrogate is a code point of its own.
function rxCpAt(s, i) {
    var c = s.charCodeAt(i)
    if (c >= 55296 && c <= 56319 && i + 1 < s.length) {
        var d = s.charCodeAt(i + 1)
        if (d >= 56320 && d <= 57343) { return (c - 55296) * 1024 + (d - 56320) + 65536 }
    }
    return c
}

// rxCpStr is code point c as UTF-16 text.
function rxCpStr(c) {
    if (c < 65536) { return String.fromCharCode(c) }
    var v = c - 65536
    return String.fromCharCode(55296 + Math.floor(v / 1024)) + String.fromCharCode(56320 + v % 1024)
}

function rxIsHexCode(c) {
    if (c >= 48 && c <= 57) { return true }
    if (c >= 65 && c <= 70) { return true }
    if (c >= 97 && c <= 102) { return true }
    return false
}

// rxPosixAdd adds the POSIX bracket class `name` ([:alpha:] and friends) to cls and
// answers whether the name is known. These exist only in the ERE dialect (flag e):
// JavaScript, Python and Java all read [[:alpha:]] as an ordinary nested-looking
// class, and Ruby is the only one of the five that has them - so the engine keeps
// reporting them as unsupported unless the caller asked for ERE.
//
// The definitions are the C locale's, and deliberately the SAME ones the emitted ERE
// engine in bash-to-llvm-ir.abnf carries in its re_ctype (so the two halves cannot
// drift). Every one of them is expressible as plain ranges, which is why no new
// shorthand opcode is needed.
function rxPosixAdd(cls, name) {
    if (name == "alpha") { rxClsAdd(cls, 65, 90); rxClsAdd(cls, 97, 122); return true }
    if (name == "digit") { rxClsAdd(cls, 48, 57); return true }
    if (name == "alnum") {
        rxClsAdd(cls, 48, 57); rxClsAdd(cls, 65, 90); rxClsAdd(cls, 97, 122)
        return true
    }
    if (name == "upper") { rxClsAdd(cls, 65, 90); return true }
    if (name == "lower") { rxClsAdd(cls, 97, 122); return true }
    if (name == "space") { rxClsAdd(cls, 9, 13); rxClsAdd(cls, 32, 32); return true }
    if (name == "blank") { rxClsAdd(cls, 9, 9); rxClsAdd(cls, 32, 32); return true }
    if (name == "punct") {
        rxClsAdd(cls, 33, 47); rxClsAdd(cls, 58, 64)
        rxClsAdd(cls, 91, 96); rxClsAdd(cls, 123, 126)
        return true
    }
    if (name == "print") { rxClsAdd(cls, 32, 126); return true }
    if (name == "graph") { rxClsAdd(cls, 33, 126); return true }
    if (name == "cntrl") { rxClsAdd(cls, 0, 31); rxClsAdd(cls, 127, 127); return true }
    if (name == "xdigit") {
        rxClsAdd(cls, 48, 57); rxClsAdd(cls, 65, 70); rxClsAdd(cls, 97, 102)
        return true
    }
    if (name == "word") {
        rxClsAdd(cls, 48, 57); rxClsAdd(cls, 65, 90)
        rxClsAdd(cls, 95, 95); rxClsAdd(cls, 97, 122)
        return true
    }
    if (name == "ascii") { rxClsAdd(cls, 0, 127); return true }
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
    if (code == RX_SJH) { return rxIsHSpaceJ(c) }
    if (code == RX_SJNH) { return !rxIsHSpaceJ(c) }
    if (code == RX_SJV) { return rxIsVSpaceJ(c) }
    if (code == RX_SJNV) { return !rxIsVSpaceJ(c) }
    if (code == RX_SPW) { return rxPyWordHit(c) }
    if (code == RX_SPNW) { return !rxPyWordHit(c) }
    if (code == RX_SPD) { return rxPyDigitHit(c) }
    if (code == RX_SPND) { return !rxPyDigitHit(c) }
    if (code == RX_SPS) { return rxPySpaceHit(c) }
    if (code == RX_SPNS) { return !rxPySpaceHit(c) }
    return false
}

// java.util.regex's \h: space, tab, U+00A0, U+1680, U+180E, U+2000..U+200A, U+202F,
// U+205F and U+3000; \v: U+000A..U+000D, U+0085, U+2028 and U+2029.
function rxIsHSpaceJ(c) {
    if (c == 32 || c == 9 || c == 160 || c == 5760 || c == 6158) { return true }
    if (c >= 8192 && c <= 8202) { return true }
    return c == 8239 || c == 8287 || c == 12288
}
function rxIsVSpaceJ(c) {
    if (c >= 10 && c <= 13) { return true }
    return c == 133 || c == 8232 || c == 8233
}

// rxPropHit answers whether code point c has the \p{..} property id (RX_P*).
// It is a KNOB like rxFail: the Unicode tables live with the language (Java's in
// lib/java-chars.js and lib/java-regex.js), so a grammar that passes J assigns
// its own after the include.
var rxPropHit = function (id, c) { return false }

// rxPyWordHit answers whether c is a word character of a Python str pattern:
// "_" or what str.isalnum() accepts. A KNOB like rxPropHit, assigned by the
// grammars that pass P (the tables are lib/python-chars.js's).
var rxPyWordHit = function (c) { return rxIsWordCode(c) }
// \d and \s of a str pattern: str.isdecimal() (Unicode Nd) and str.isspace().
// KNOBS like rxPyWordHit.
var rxPyDigitHit = function (c) { return rxIsDigitCode(c) }
var rxPySpaceHit = function (c) { return rxIsSpaceCode(c) }

// rxClsHitX is rxClsHit for a class TREE: the union of the class's own ranges,
// its properties and its nested classes, intersected with every && operand.
function rxClsHitX(cls, c) {
    var hit = rxClsHit(cls, c)
    var props = cls.props
    var i = 0
    while (!hit && i < props.length) {
        hit = rxPropHit(props[i], c)
        i = i + 1
    }
    var subs = cls.subs
    i = 0
    while (!hit && i < subs.length) {
        hit = rxClsHitX(subs[i], c) != subs[i].neg
        i = i + 1
    }
    var ands = cls.ands
    i = 0
    while (hit && i < ands.length) {
        hit = rxClsHitX(ands[i], c) != ands[i].neg
        i = i + 1
    }
    return hit
}

// rxSwapCase answers the other-case code point of an ASCII letter, or -1.
function rxSwapCase(c) {
    if (c >= 65 && c <= 90) { return c + 32 }
    if (c >= 97 && c <= 122) { return c - 32 }
    return -1
}

// rxClsFold adds the other case of every ASCII letter in the class's ranges, which
// is rxClsMatch's icase rule decided at parse time: P resolves a scoped (?i:...)
// that way, so the matcher's re.icase stays one flag for the whole pattern.
// Shorthands are closed under case already.
function rxClsFold(cls) {
    var lo = cls.lo
    var hi = cls.hi
    var n = lo.length
    var i = 0
    while (i < n) {
        var a = lo[i]
        var b = hi[i]
        if (a < 65) { a = 65 }
        if (b > 90) { b = 90 }
        if (a <= b) { rxClsAdd(cls, a + 32, b + 32) }
        a = lo[i]
        b = hi[i]
        if (a < 97) { a = 97 }
        if (b > 122) { b = 122 }
        if (a <= b) { rxClsAdd(cls, a - 32, b - 32) }
        i = i + 1
    }
}

// rxClsFoldP is Python's Unicode IGNORECASE of a class: CPython folds every
// character of a set, and two characters fold together when their lowercase or
// their uppercase agree (_ignorecase_fixes). A short non-ASCII range gets its
// characters' other cases here; the matcher (pu, rxClsHitU) tries the text
// character's uppercase and its lowercase.
function rxClsFoldP(cls) {
    var lo = cls.lo
    var hi = cls.hi
    var n = lo.length
    var i = 0
    while (i < n) {
        var a = lo[i] < 128 ? 128 : lo[i]
        if (hi[i] - a < 256) {
            var c = a
            while (c <= hi[i]) {
                var l = rxCaseU(c, 0)
                var u = rxCaseU(c, 1)
                rxClsAdd(cls, l, l)
                rxClsAdd(cls, u, u)
                var ul = rxCaseU(l, 1)
                var lu = rxCaseU(u, 0)
                rxClsAdd(cls, ul, ul)
                rxClsAdd(cls, lu, lu)
                c = c + 1
            }
        }
        i = i + 1
    }
    cls.u = true
    cls.pu = true
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
    // A pu class (P's Unicode case folding) already holds the other case of its
    // ASCII letters (rxClsFold), and no other character's case is ASCII.
    if (cls.u && !(cls.pu && c < 128)) { return rxClsHitU(cls, c) }
    return false
}

// rxClsHitU is the rest of a u class's range test: the simple uppercase of c,
// then that uppercase's lowercase.
function rxClsHitU(cls, c) {
    var lo = cls.lo
    var hi = cls.hi
    var k = 0
    var v = rxCaseU(c, 1)
    while (k < 2) {
        var i = 0
        while (i < lo.length) {
            if (v >= lo[i] && v <= hi[i]) { return true }
            i = i + 1
        }
        v = rxCaseU(v, 0)
        k = k + 1
    }
    return false
}

// rxCaseU is java.lang.Character's simple case mapping of code point c: mode 1
// uppercase, 0 lowercase. A mapping to more than one code point keeps c.
function rxCaseU(c, mode) {
    var s = unicodeCase(rxCpStr(c), mode)
    var r = rxCpAt(s, 0)
    if (s.length == (r > 65535 ? 2 : 1)) { return r }
    return c
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

function rxClsMatchX(cls, c, icase) {
    var hit = rxClsHitX(cls, c)
    if (!hit && icase) {
        var alt = rxSwapCase(c)
        if (alt >= 0) { hit = rxClsHitX(cls, alt) }
    }
    return hit != cls.neg
}


// ===== The pattern parser =====
//
// A node carries every field with a default, so the shape is uniform:
//   t     "cat" | "alt" | "rep" | "cls" | "assert" | "group" | "look" | "lookb" | "bref"
//         | "atom" (an atomic group, J) | "op" (one J instruction, gidx) | "empty"
//   items child nodes (always an array)
//   cls   character class (always an object)
//   min/max/greedy   for "rep" (max < 0 means unbounded); min of a "bref" is 1
//         for a case-insensitive one, 2 under Java's UNICODE_CASE
//   gidx  capture index for "group" (-1: non-capturing), assertion kind for "assert",
//         group index for "bref", 1 for a negative "look"

// rxIsMetaCh answers whether a character has to be backslashed to stay a literal.
// Whitespace and # are included because the x (extended) flag would otherwise drop
// them, and - because it is a range operator inside a character class.
// \u000c and \u000b, NOT \f and \v: this file is layer 2, so it is parsed BOTH by
// goja (real JavaScript, where \f and \v are formfeed and vertical tab) AND by the
// frozen MetaJS parser, whose string literal has NEITHER and DROPS THE BACKSLASH -
// turning them into the letters f and v. The \uXXXX spelling means the same thing in
// both parsers. (The Go twin abnf/jsrtregex.go can use \f and \v.) tools/metajs-lint
// catches the class.
function rxIsMetaCh(ch) {
    return "\\^$.|?*+()[]{}-#/ \t\n\r\u000c\u000b".indexOf(ch) >= 0
}

// rxExpandQuotes rewrites every \Q...\E region into the equivalent escaped text, as
// a PRE-PASS over the pattern rather than a parser mode. An unterminated \Q runs to
// the end of the pattern (Java's rule). Doing it here means a quantifier after \E
// binds to the LAST quoted character - \Qab\E+ is ab+ - which is also Java's rule,
// and it keeps the parser itself free of a quoting state that every rule would have
// to respect.
function rxExpandQuotes(p) {
    if (p.indexOf("\\Q") < 0) { return p }
    var out = ""
    var i = 0
    while (i < p.length) {
        if (p.charAt(i) != "\\" || i + 1 >= p.length) {
            out += p.charAt(i)
            i = i + 1
            continue
        }
        if (p.charAt(i + 1) != "Q") {
            out += p.substring(i, i + 2)
            i = i + 2
            continue
        }
        i = i + 2
        while (i < p.length) {
            if (p.charAt(i) == "\\" && i + 1 < p.length && p.charAt(i + 1) == "E") {
                i = i + 2
                break
            }
            var ch = p.charAt(i)
            if (rxIsMetaCh(ch)) { out += "\\" }
            out += ch
            i = i + 1
        }
    }
    return out
}

// rxCountGroups is a cheap PRE-PASS over the pattern, counting the capturing groups
// before the real parse starts. A backslash-digit escape can only be read correctly
// once the total is known: \1 in a pattern with one group is a backreference, and in a
// pattern with none it is either a legacy octal escape (JavaScript, flag o) or an
// error (everywhere else) - and the octal reading has to CONSUME the following digits,
// which the parser cannot undo after the fact.
function rxCountGroups(p) {
    var n = 0
    var i = 0
    var incls = false
    while (i < p.length) {
        var c = p.charCodeAt(i)
        if (c == 92) { i = i + 2; continue }
        if (incls) {
            if (c == 93) { incls = false }
            i = i + 1
            continue
        }
        if (c == 91) { incls = true; i = i + 1; continue }
        if (c != 40) { i = i + 1; continue }
        if (i + 1 >= p.length || p.charCodeAt(i + 1) != 63) { n = n + 1; i = i + 1; continue }
        // (?<name> and (?'name' capture; (?<= and (?<! are lookbehind, and every other
        // (? form is non-capturing.
        var c2 = i + 2 < p.length ? p.charCodeAt(i + 2) : 0
        var c3 = i + 3 < p.length ? p.charCodeAt(i + 3) : 0
        if (c2 == 39 || (c2 == 60 && c3 != 61 && c3 != 33)) { n = n + 1 }
        i = i + 2
    }
    return n
}

function rxNode(t) {
    // bname carries the NAME of a \k<name> backreference until rxCompile resolves it
    // to a group number; it stays "" for every other node.
    // cnt marks a J {n,m} repeat, which is emitted as a counted loop; closed marks
    // a capturing group whose ")" has been read (P's open-group reference check).
    return {t: t, items: [], cls: rxNewCls(), min: 0, max: 0, greedy: true, gidx: -1, bname: "",
            cnt: false, closed: false}
}

// rxNameIdx is the group NUMBER a name was declared for, or -1.
function rxNameIdx(names, groups, name) {
    var i = 0
    while (i < names.length) {
        if (names[i] == name) { return groups[i] }
        i = i + 1
    }
    return -1
}

function rxErr(st, msg) {
    if (st.err == "") { st.err = msg }
}

// rxErrAt is rxErr with the error's index into the pattern (re.errAt).
function rxErrAt(st, msg, at) {
    if (st.err == "") {
        st.err = msg
        st.errAt = at
    }
}

// The octal digit at index i of the pattern, or -1.
function rxOctAt(st, i) {
    if (i >= st.n) { return -1 }
    var d = st.p.charCodeAt(i) - 48
    if (d < 0 || d > 7) { return -1 }
    return d
}

// The \p{..} property ids rxPropHit answers for (the names that read them are
// lib/java-regex.js's jxPropCls). RX_PGC + t is general category t, in
// java.lang.Character.getType's numbering; RX_PSC + n is the script of ordinal
// n, RX_PBLK + n the n-th block and RX_PBIN + b bit b of the emoji /
// ideographic / mirrored table.
var RX_PL = 1        // isLetter
var RX_PLU = 2       // gc Lu
var RX_PLL = 3       // gc Ll
var RX_PLT = 4       // gc Lt
var RX_PND = 5       // isDigit (gc Nd)
var RX_PNL = 6       // gc Nl
var RX_PALPHA = 7    // isAlphabetic
var RX_PUPPER = 8    // isUpperCase
var RX_PLOWER = 9    // isLowerCase
var RX_PLD = 10      // isLetterOrDigit
var RX_PWORD = 11    // Unicode \w: Alphabetic, marks, Nd, Pc, Join_Control
var RX_PWHITE = 12   // White_Space
var RX_PHEX = 13     // Hex_Digit
var RX_PJOIN = 14    // Join_Control
var RX_PNONCH = 15   // Noncharacter_Code_Point
var RX_PASSIGNED = 16 // gc is not Cn (also isDefined)
var RX_PGRAPH = 17   // Unicode \p{Graph}
var RX_PPRINT = 18   // Unicode \p{Print}
var RX_PBLANK = 19   // Unicode \p{Blank}
var RX_PCASED = 20   // upper, lower or title case: the CASE_INSENSITIVE reading of each
var RX_PALNUM = 21   // Unicode \p{Alnum}
var RX_PJSTART = 22  // isJavaIdentifierStart
var RX_PJPART = 23   // isJavaIdentifierPart
var RX_PUSTART = 24  // isUnicodeIdentifierStart
var RX_PUPART = 25   // isUnicodeIdentifierPart
var RX_PIGNORE = 26  // isIdentifierIgnorable
var RX_PJWHITE = 27  // isWhitespace
var RX_PBIN = 32
var RX_PGC = 64
var RX_PMN = 70      // gc Mn
var RX_PSC = 128
var RX_PBLK = 512

// rxGraphNext answers where the extended grapheme cluster that starts at i
// ends (limit at most): Java's Grapheme.nextBoundary. A KNOB like rxPropHit
// (lib/java-regex.js's jxGraphNext).
var rxGraphNext = function (text, i, limit) { return i + 1 }

// rxPropCls answers the class of \p{name}, or reports Java's error at index at.
// A KNOB like rxPropHit: the names are java.util.regex's, resolved over its
// tables (lib/java-regex.js's jxPropCls).
var rxPropCls = function (st, name, at) {
    rxErr(st, "unsupported character property \\p{" + name + "}")
    return rxNewCls()
}

// rxEscJ reads the escapes java.util.regex spells differently, for rxEsc under J:
// c (already consumed) is the character after the backslash. It answers false for
// the escapes every dialect shares. Each error stands where Java reports it: the
// character it could not use, or the end of the pattern.
function rxEscJ(st, c, inClass, res) {
    var p = st.p
    // UNICODE_CHARACTER_CLASS: \d \w \s and their complements are Unicode's.
    if (st.uclass && (c == 100 || c == 68 || c == 119 || c == 87 || c == 115 || c == 83)) {
        res.kind = 3
        var uc = rxNewCls()
        var uprops = uc.props
        uprops.push(c == 100 || c == 68 ? RX_PND : c == 119 || c == 87 ? RX_PWORD : RX_PWHITE)
        uc.x = true
        uc.neg = c == 68 || c == 87 || c == 83
        st.pcls = uc
        return true
    }
    if (c == 104 || c == 72 || c == 118 || c == 86) { // \h \H \v \V
        res.kind = 1
        if (c == 104) { res.sp = RX_SJH }
        if (c == 72) { res.sp = RX_SJNH }
        if (c == 118) { res.sp = RX_SJV }
        if (c == 86) { res.sp = RX_SJNV }
        return true
    }
    if (c == 99) { // \cX is X with bit 6 flipped
        if (st.i >= st.n) {
            rxErrAt(st, "Illegal control escape sequence", st.i - 1)
            return true
        }
        var cc = p.charCodeAt(st.i)
        st.i = st.i + 1
        if (Math.floor(cc / 64) % 2 == 1) { res.ch = cc - 64 } else { res.ch = cc + 64 }
        return true
    }
    if (c == 48) { // \0n, \0nn, \0mnn (m <= 3)
        var on = rxOctAt(st, st.i)
        if (on < 0) {
            rxErrAt(st, "Illegal octal escape sequence", st.i)
            return true
        }
        st.i = st.i + 1
        res.ch = on
        var om = rxOctAt(st, st.i)
        if (om < 0) { return true }
        st.i = st.i + 1
        res.ch = on * 8 + om
        var oo = rxOctAt(st, st.i)
        if (oo < 0 || on > 3) { return true }
        st.i = st.i + 1
        res.ch = on * 64 + om * 8 + oo
        return true
    }
    if (c == 120) { // \xhh and \x{h...}
        var x0 = st.i
        var xa = x0 < st.n ? rxHexVal(p.charCodeAt(x0)) : -1
        if (xa >= 0) {
            var xb = x0 + 1 < st.n ? rxHexVal(p.charCodeAt(x0 + 1)) : -1
            if (xb < 0) {
                rxErrAt(st, "Illegal hexadecimal escape sequence", x0 + 1)
                return true
            }
            st.i = x0 + 2
            res.ch = xa * 16 + xb
            return true
        }
        if (x0 + 1 < st.n && p.charCodeAt(x0) == 123 && rxHexVal(p.charCodeAt(x0 + 1)) >= 0) {
            var xv = 0
            st.i = x0 + 1
            while (st.i < st.n && rxHexVal(p.charCodeAt(st.i)) >= 0) {
                xv = xv * 16 + rxHexVal(p.charCodeAt(st.i))
                if (xv > 1114111) {
                    rxErrAt(st, "Hexadecimal codepoint is too big", st.i)
                    return true
                }
                st.i = st.i + 1
            }
            if (st.i >= st.n || p.charCodeAt(st.i) != 125) {
                rxErrAt(st, "Unclosed hexadecimal escape sequence", st.i)
                return true
            }
            st.i = st.i + 1
            res.ch = xv
            return true
        }
        rxErrAt(st, "Illegal hexadecimal escape sequence", x0)
        return true
    }
    if (c == 117) { // \uhhhh: exactly four
        var uv = 0
        var uk = 0
        while (uk < 4) {
            var uh = st.i < st.n ? rxHexVal(p.charCodeAt(st.i)) : -1
            if (uh < 0) {
                rxErrAt(st, "Illegal Unicode escape sequence", st.i)
                return true
            }
            uv = uv * 16 + uh
            st.i = st.i + 1
            uk = uk + 1
        }
        res.ch = uv
        // A high surrogate escape and a low surrogate escape after it are one
        // code point, as Pattern.u() reads them.
        if (uv >= 55296 && uv <= 56319 && st.i + 5 < st.n && p.charCodeAt(st.i) == 92
            && p.charCodeAt(st.i + 1) == 117) {
            var lv = 0
            var lk = 2
            while (lk < 6 && rxHexVal(p.charCodeAt(st.i + lk)) >= 0) {
                lv = lv * 16 + rxHexVal(p.charCodeAt(st.i + lk))
                lk = lk + 1
            }
            if (lk == 6 && lv >= 56320 && lv <= 57343) {
                res.ch = (uv - 55296) * 1024 + (lv - 56320) + 65536
                st.i = st.i + 6
            }
        }
        return true
    }
    if (c == 112 || c == 80) { // \pX, \p{Name}, and \P for the complement
        res.kind = 3
        var pname = ""
        var pat = st.i
        if (st.i < st.n && p.charCodeAt(st.i) == 123) {
            var pe = st.i + 1
            while (pe < st.n && p.charCodeAt(pe) != 125) { pe = pe + 1 }
            if (pe >= st.n) {
                rxErrAt(st, "Unclosed character family", st.n)
                return true
            }
            if (pe == st.i + 1) {
                rxErrAt(st, "Empty character family", pe)
                return true
            }
            pname = p.substring(st.i + 1, pe)
            pat = pe
            st.i = pe + 1
        } else if (st.i < st.n) {
            pname = p.charAt(st.i)
            st.i = st.i + 1
        } else {
            pname = String.fromCharCode(0)
        }
        st.pcls = rxPropCls(st, pname, pat)
        st.supp = true
        st.pcls.neg = c == 80
        return true
    }
    if (c >= 49 && c <= 57) { // \n..: as many digits as name a group opened so far
        if (inClass) {
            rxErrAt(st, "Illegal/unsupported escape sequence", st.i - 1)
            return true
        }
        // Under C every digit is the number's (rxPcre writes (?:\n)).
        var g = c - 48
        while (st.i < st.n && rxIsDigitCode(p.charCodeAt(st.i))
               && (st.pcre || g * 10 + p.charCodeAt(st.i) - 48 <= st.ngroups)) {
            g = g * 10 + p.charCodeAt(st.i) - 48
            st.i = st.i + 1
        }
        res.kind = 2
        res.ch = g
        return true
    }
    // The letters java.util.regex gives no meaning (in a class, also the
    // assertions and \k, \R, \X, which only stand outside one) are errors;
    // \N{name} is Java's but not this engine's: it has no character name table.
    if ((c >= 65 && c <= 90) || (c >= 97 && c <= 122)) {
        if ("dDwWsSntrfae".indexOf(String.fromCharCode(c)) >= 0) { return false }
        if (!inClass && c == 78) {
            rxErr(st, "unsupported escape \\" + String.fromCharCode(c))
            return true
        }
        rxErrAt(st, "Illegal/unsupported escape sequence", st.i - 1)
        return true
    }
    return false
}

// rxSkipX steps over the whitespace and # comments that the x flag makes insignificant.
function rxSkipX(st) {
    if (!st.ext) { return }
    var p = st.p
    while (st.i < st.n) {
        var c = p.charCodeAt(st.i)
        if (rxIsSpaceCode(c)) {
            st.i = st.i + 1
        } else if (c == 35) { // # (under J up to any line terminator)
            while (st.i < st.n && p.charCodeAt(st.i) != 10
                   && !(st.java && rxIsLineSepJ(p.charCodeAt(st.i), st.unix))) {
                st.i = st.i + 1
            }
        } else {
            return
        }
    }
}

// rxSkipXAt is rxSkipX from index j: where the next significant character is.
function rxSkipXAt(st, j) {
    st.i = j
    rxSkipX(st)
    return st.i
}

function rxHexVal(c) {
    if (c >= 48 && c <= 57) { return c - 48 }
    if (c >= 65 && c <= 70) { return c - 55 }
    if (c >= 97 && c <= 102) { return c - 87 }
    return -1
}

// rxEsc reads the escape whose backslash has already been consumed. It answers
// {kind: 0, ch: code} for a literal character and {kind: 1, sp: code} for a
// shorthand class; kind 2 is a backreference (ch = group number), and kind 3 a
// \p{..} property, whose class is left in st.pcls (J only).
function rxEsc(st, inClass) {
    var res = {kind: 0, ch: 0, sp: 0}
    if (st.i >= st.n) {
        rxErr(st, "trailing backslash in pattern")
        return res
    }
    var p = st.p
    var c = p.charCodeAt(st.i)
    st.i = st.i + 1
    if (st.java && rxEscJ(st, c, inClass, res)) { return res }
    if (st.py && rxBadEscP(c, inClass)) {
        rxErr(st, "bad escape \\" + String.fromCharCode(c))
        return res
    }
    if (st.py && c == 85) { // \UHHHHHHHH
        var bv = 0
        var bgot = 0
        while (bgot < 8 && st.i < st.n && rxHexVal(p.charCodeAt(st.i)) >= 0) {
            bv = bv * 16 + rxHexVal(p.charCodeAt(st.i))
            st.i = st.i + 1
            bgot = bgot + 1
        }
        res.ch = bv
        return res
    }
    if (c == 100) { res.kind = 1; res.sp = st.pyuni ? RX_SPD : RX_SD; return res }
    if (c == 68) { res.kind = 1; res.sp = st.pyuni ? RX_SPND : RX_SND; return res }
    if (c == 119) { res.kind = 1; res.sp = st.pyuni ? RX_SPW : RX_SW; return res }
    if (c == 87) { res.kind = 1; res.sp = st.pyuni ? RX_SPNW : RX_SNW; return res }
    if (c == 115) { res.kind = 1; res.sp = st.pyuni ? RX_SPS : RX_SS; return res }
    if (c == 83) { res.kind = 1; res.sp = st.pyuni ? RX_SPNS : RX_SNS; return res }
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
        // A backreference only if the pattern HAS that group (rxCountGroups counted
        // them before the parse began).
        // Python's group must be opened before the reference and closed by it.
        if (st.py && c - 48 <= st.ntotal) {
            if (c - 48 > st.ngroups) {
                rxErr(st, "invalid group reference " + (c - 48))
            } else if (!st.gnodes[c - 48].closed) {
                rxErr(st, "cannot refer to an open group")
            }
        }
        if (c - 48 <= st.ntotal) {
            res.kind = 2
            res.ch = c - 48
            return res
        }
        if (!st.octal) {
            rxErr(st, "invalid group reference " + (c - 48))
            return res
        }
        // JavaScript Annex B. 8 and 9 are not octal digits, so they are an identity
        // escape; 0-3 admit three octal digits and 4-7 only two, which is what makes
        // node read /\400/ as the two characters \40 and "0".
        if (c > 55) { res.ch = c; return res }
        var oval = c - 48
        var omax = c <= 51 ? 3 : 2
        var ogot = 1
        while (ogot < omax && st.i < st.n) {
            var oc = p.charCodeAt(st.i)
            if (oc < 48 || oc > 55) { break }
            oval = oval * 8 + (oc - 48)
            st.i = st.i + 1
            ogot = ogot + 1
        }
        res.ch = oval
        return res
    }
    if ((st.java || st.py) && c >= 55296 && c <= 56319) {
        c = rxCpAt(p, st.i - 1)
        if (c > 65535) { st.i = st.i + 1 }
    }
    res.ch = c
    return res
}

// rxBadEscP: is \c an escape Python's re refuses ("bad escape")? An ASCII
// letter it gives no meaning, and 8 or 9 in a set. \k is the named
// backreference python-regex.js writes for (?P=name).
function rxBadEscP(c, inClass) {
    if (c >= 48 && c <= 57) { return inClass && c >= 56 }
    if (!((c >= 65 && c <= 90) || (c >= 97 && c <= 122))) { return false }
    var ok = inClass ? "abdfnrstuvwxDNSUW" : "abdfknrstuvwxzABDNSUWZ"
    return ok.indexOf(String.fromCharCode(c)) < 0
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
        // A POSIX bracket [:alpha:] is a real class under the "e" (POSIX ERE) and
        // "p" (POSIX classes only, which is Ruby) dialects, and is not supported
        // anywhere else; without one of those, report it rather than silently
        // reading it as a set of punctuation characters.
        if (c == 91 && st.i + 1 < st.n && p.charCodeAt(st.i + 1) == 58) {
            if (!st.posix) {
                rxErr(st, "POSIX bracket expressions are not supported")
                return node
            }
            var pj = st.i + 2
            while (pj + 1 < st.n && !(p.charCodeAt(pj) == 58 && p.charCodeAt(pj + 1) == 93)) {
                pj = pj + 1
            }
            if (pj + 1 >= st.n) {
                rxErr(st, "Unmatched [, [^, [:, [., or [=")
                return node
            }
            if (!rxPosixAdd(cls, p.substring(st.i + 2, pj))) {
                rxErr(st, "Invalid character class name")
                return node
            }
            st.i = pj + 2
            // A POSIX class is a SET, so it cannot be the low end of a range: bash
            // says "invalid character range" for [[:alpha:]-z] and Ruby refuses it
            // with "unmatched range specifier in char-class". A trailing "-]" is
            // still a literal hyphen.
            if (st.posix && st.i + 1 < st.n && p.charCodeAt(st.i) == 45
                && p.charCodeAt(st.i + 1) != 93) {
                rxErr(st, "Invalid range end")
                return node
            }
            continue
        }
        var loCode = -1
        // In a POSIX ERE a backslash inside a bracket expression is an ORDINARY
        // CHARACTER: [a\-b] is {a} plus the range \ .. b, and [a\] closes on the ],
        // leaving a literal backslash in the set. Every other dialect here escapes.
        if (c == 92 && !st.ere) { // backslash
            st.i = st.i + 1
            var e = rxEsc(st, true)
            if (e.kind == 1) {
                rxClsAddSp(cls, e.sp)
                continue
            }
            loCode = e.ch
        } else {
            if (st.py) { c = rxCpAt(p, st.i) }
            st.i = st.i + (c > 65535 ? 2 : 1)
            loCode = c
        }
        // A range needs a '-' that is neither the last character nor followed by ].
        if (st.i + 1 < st.n && p.charCodeAt(st.i) == 45 && p.charCodeAt(st.i + 1) != 93) {
            st.i = st.i + 1
            var hiCode = -1
            var c2 = p.charCodeAt(st.i)
            // ... nor the high end: [a-[:digit:]] is the same error.
            if (st.posix && c2 == 91 && st.i + 1 < st.n && p.charCodeAt(st.i + 1) == 58) {
                rxErr(st, "Invalid range end")
                return node
            }
            if (c2 == 92 && !st.ere) {
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
                if (st.py) { c2 = rxCpAt(p, st.i) }
                st.i = st.i + (c2 > 65535 ? 2 : 1)
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

// rxParseClassJ reads a java.util.regex class; the [ has already been consumed.
function rxParseClassJ(st) {
    var node = rxNode("cls")
    var cls = rxClassJ(st, true)
    rxClsSimplify(cls)
    node.cls = cls
    return node
}

// rxClassJ is java.util.regex's clazz(), its precedence included: the items of a
// class are a union, && intersects everything before it with everything after it
// up to the class's ] (so [a-z&&def] is a-z AND def), a leading ^ complements the
// whole class, and "-" before a nested class is a literal. st.i is at the first
// character after the [ - or, for the right operand of && (consume false), at the
// operand's first character, its ] being the enclosing class's.
//
// Java's single characters below U+0100 collect in ONE set (bits) that the
// unions and intersections built so far keep referring to, so a character added
// after an && still joins them: [a-cx&&&y] holds x. The trees here share the
// set the same way, and rxClsSimplify flattens them once the class is complete.
function rxClassJ(st, consume) {
    var p = st.p
    var prev = rxNewCls()
    var hasPrev = false
    var curr = rxNewCls()
    var hasCurr = false
    var bits = rxNewCls()
    var hasBits = false
    var neg = false
    if (st.i < st.n && p.charCodeAt(st.i) == 94 && p.charCodeAt(st.i - 1) == 91) { // ^
        neg = true
        st.i = st.i + 1
    }
    while (st.err == "") {
        rxSkipX(st)
        if (st.i >= st.n) {
            rxErr(st, "unterminated character class")
            return prev
        }
        var c = p.charCodeAt(st.i)
        if (c == 91) { // [
            st.i = st.i + 1
            curr = rxClassJ(st, true)
            hasCurr = true
            if (hasPrev) { prev = rxClsOr(prev, curr) } else { prev = curr }
            hasPrev = true
            continue
        }
        if (c == 38 && st.i + 1 < st.n && p.charCodeAt(st.i + 1) == 38) { // &&
            st.i = st.i + 2
            var right = rxNewCls()
            var hasRight = false
            rxSkipX(st)
            while (st.err == "" && st.i < st.n && p.charCodeAt(st.i) != 93
                   && p.charCodeAt(st.i) != 38) {
                var r = rxNewCls()
                if (p.charCodeAt(st.i) == 91) {
                    st.i = st.i + 1
                    r = rxClassJ(st, true)
                } else {
                    r = rxClassJ(st, false)
                }
                if (hasRight) { right = rxClsOr(right, r) } else { right = r }
                hasRight = true
                rxSkipX(st)
            }
            if (st.err != "") { return prev }
            if (hasBits) {
                if (hasPrev) { prev = rxClsOr(prev, bits) } else {
                    prev = bits
                    curr = bits
                    hasPrev = true
                    hasCurr = true
                }
                hasBits = false
            }
            if (hasRight) {
                curr = right
                hasCurr = true
            }
            if (!hasPrev) {
                if (!hasRight) {
                    rxErrAt(st, "Bad class syntax", st.i - 1)
                    return prev
                }
                prev = right
                hasPrev = true
            } else {
                if (!hasCurr) {
                    rxErrAt(st, "Bad intersection syntax", st.i - 1)
                    return prev
                }
                prev = rxClsAnd(prev, curr)
            }
            continue
        }
        if (c == 93 && (hasPrev || hasBits)) { // ]
            if (consume) { st.i = st.i + 1 }
            if (!hasPrev) { prev = bits } else if (hasBits) { prev = rxClsOr(prev, bits) }
            if (neg) { return rxClsNot(prev) }
            return prev
        }
        var item = rxRangeJ(st, bits)
        if (st.err != "") { return prev }
        if (item.lo.length + item.sp.length + item.subs.length + item.props.length == 0) {
            hasBits = true
        } else {
            curr = item
            hasCurr = true
            if (hasPrev) { prev = rxClsOr(prev, curr) } else { prev = curr }
            hasPrev = true
        }
    }
    return prev
}

// rxRangeJ is java.util.regex's range(): one character, range, escape or
// property of a class. A single character below U+0100 goes into bits and the
// answer is an EMPTY class; anything else is answered as a class of its own.
function rxRangeJ(st, bits) {
    var p = st.p
    var ch = rxCpAt(p, st.i)
    if (ch == 92) { // backslash
        st.i = st.i + 1
        // \v as either end of a range is the one character U+000B.
        var vt = st.i + 1 < st.n && p.charCodeAt(st.i) == 118 && p.charCodeAt(st.i + 1) == 45
        var e = rxEsc(st, true)
        if (vt) {
            ch = 11
        } else if (e.kind == 1) {
            var sh = rxNewCls()
            rxClsAddSp(sh, e.sp)
            return sh
        } else if (e.kind == 3) {
            return st.pcls
        } else {
            ch = e.ch
        }
    } else {
        st.i = st.i + (ch > 65535 ? 2 : 1)
    }
    rxSkipX(st)
    if (st.i < st.n && p.charCodeAt(st.i) == 45) { // -
        var end = st.i + 1 < st.n ? p.charCodeAt(st.i + 1) : 0
        if (end != 91 && end != 93) {
            st.i = st.i + 1
            rxSkipX(st)
            var m = -1
            if (st.i >= st.n) {
                st.i = st.i + 1
            } else if (p.charCodeAt(st.i) == 92) {
                st.i = st.i + 1
                if (st.i < st.n && p.charCodeAt(st.i) == 118) {
                    st.i = st.i + 1
                    m = 11
                } else {
                    var e2 = rxEsc(st, true)
                    if (e2.kind == 0) { m = e2.ch }
                }
            } else {
                m = rxCpAt(p, st.i)
                st.i = st.i + (m > 65535 ? 2 : 1)
            }
            if (m < ch) {
                rxErrAt(st, "Illegal character range", st.i - 1)
            }
            var rg = rxNewCls()
            rxClsAdd(rg, ch, m)
            if (st.icase) {
                rxFoldJ(st, rg, false)
                st.supp = true
            }
            return rg
        }
    }
    // Pattern.bitsOrSingle: a character below U+0100 joins bits with its other
    // cases, except under (?iu) the ones whose case partners are not all in
    // Latin-1, which are single predicates like any other character.
    var excl = st.icase && st.ucase && "\u00ff\u00b5IiSsKk\u00c5\u00e5".indexOf(String.fromCharCode(ch)) >= 0
    if (ch < 256 && !excl) {
        rxClsAdd(bits, ch, ch)
        if (st.icase && ch < 128) {
            var sw = rxSwapCase(ch)
            if (sw >= 0) { rxClsAdd(bits, sw, sw) }
        } else if (st.icase && st.ucase) {
            rxClsAdd(bits, rxCaseU(ch, 0), rxCaseU(ch, 0))
            rxClsAdd(bits, rxCaseU(ch, 1), rxCaseU(ch, 1))
        }
        return rxNewCls()
    }
    var one = rxNewCls()
    rxClsAdd(one, ch, ch)
    if (st.icase) { rxFoldJ(st, one, true) }
    return one
}

// The union, intersection and complement of class trees, BY REFERENCE (see
// rxClassJ).
function rxClsOr(a, b) {
    var u = rxNewCls()
    var subs = u.subs
    subs.push(a)
    subs.push(b)
    u.x = true
    return u
}
function rxClsAnd(a, b) {
    var u = rxNewCls()
    var subs = u.subs
    subs.push(a)
    var ands = u.ands
    ands.push(b)
    u.x = true
    return u
}
function rxClsNot(a) {
    var u = rxNewCls()
    var subs = u.subs
    subs.push(a)
    u.neg = true
    u.x = true
    return u
}

// rxClsSimplify folds every positive flat member of a union into its parent's
// own ranges, so that a class without &&, \p{..} or a negated member ends up
// flat and runs as a plain RX_CLASS. A u member stays a member: its ranges
// fold, the parent's do not.
function rxClsSimplify(cls) {
    if (!cls.x) { return }
    var subs = cls.subs
    var keep = []
    var i = 0
    while (i < subs.length) {
        var s = subs[i]
        rxClsSimplify(s)
        if (!s.x && !s.neg && !s.u) {
            var lo = s.lo
            var hi = s.hi
            var k = 0
            while (k < lo.length) {
                rxClsAdd(cls, lo[k], hi[k])
                k = k + 1
            }
            var sp = s.sp
            k = 0
            while (k < sp.length) {
                rxClsAddSp(cls, sp[k])
                k = k + 1
            }
        } else {
            keep.push(s)
        }
        i = i + 1
    }
    cls.subs = keep
    var ands = cls.ands
    i = 0
    while (i < ands.length) {
        rxClsSimplify(ands[i])
        i = i + 1
    }
    cls.x = keep.length + ands.length + cls.props.length > 0
}

// rxReadGroupName reads name> or name' after (?< / (?'.
function rxReadGroupName(st, closer) {
    var p = st.p
    var start = st.i
    if (st.java) { return rxReadGroupNameJ(st) }
    while (st.i < st.n && p.charCodeAt(st.i) != closer) { st.i = st.i + 1 }
    if (st.i >= st.n) {
        rxErr(st, "unterminated group name")
        return ""
    }
    var name = p.substring(start, st.i)
    st.i = st.i + 1
    return name
}

// rxReadGroupNameJ reads a java.util.regex group name, an ASCII letter and then
// letters and digits, and its closing >.
function rxReadGroupNameJ(st) {
    var p = st.p
    var start = st.i
    var c = st.i < st.n ? p.charCodeAt(st.i) : 0
    if (!((c >= 65 && c <= 90) || (c >= 97 && c <= 122))) {
        rxErrAt(st, "capturing group name does not start with a Latin letter", st.i)
        return ""
    }
    while (st.i < st.n && (rxIsWordCode(p.charCodeAt(st.i)) && p.charCodeAt(st.i) != 95)) {
        st.i = st.i + 1
    }
    if (st.i >= st.n || p.charCodeAt(st.i) != 62) {
        rxErrAt(st, "named capturing group is missing trailing '>'", st.i)
        return ""
    }
    st.i = st.i + 1
    return p.substring(start, st.i - 1)
}

function rxParseGroup(st) {
    var p = st.p
    // The ( is already consumed. A POSIX ERE has no (? forms at all: the ? is simply
    // a quantifier sitting where an atom belongs, which is the error bash reports for
    // (?:a) - "repetition-operator operand invalid".
    if (st.i < st.n && p.charCodeAt(st.i) == 63 && !st.ere) { // ?
        st.i = st.i + 1
        if (st.i >= st.n) {
            if (st.java) { rxErrAt(st, "Unknown inline modifier", st.i) } else { rxErr(st, "unterminated group") }
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
        if (k == 35 && !st.java) { // (?# comment; java.util.regex has none
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
        if (k == 60 || (k == 39 && !st.java)) { // (?<name>  (?'name'
            if (k == 60 && st.i + 1 < st.n) {
                var k2 = p.charCodeAt(st.i + 1)
                if ((k2 == 61 || k2 == 33) && (st.java || st.py)) { // (?<= (?<!
                    st.i = st.i + 2
                    var lb = rxNode("lookb")
                    if (k2 == 33) { lb.gidx = 1 }
                    lb.items.push(rxParseAlt(st))
                    rxExpect(st, 41, "unterminated group")
                    return lb
                }
                if (k2 == 61 || k2 == 33) {
                    rxErr(st, "lookbehind is not supported")
                    return rxNode("empty")
                }
            }
            st.i = st.i + 1
            var closer = 62
            if (k == 39) { closer = 39 }
            var nm = rxReadGroupName(st, closer)
            if (st.java && rxNameIdx(st.names, st.nameGroups, nm) >= 0) {
                rxErrAt(st, "Named capturing group <" + nm + "> is already defined", st.i - 1)
            }
            if (st.py && rxNameIdx(st.names, st.nameGroups, nm) >= 0) {
                rxErr(st, "redefinition of group name '" + nm + "' as group " + (st.ngroups + 1) + "; was group "
                          + rxNameIdx(st.names, st.nameGroups, nm))
            }
            var ng = rxNode("group")
            st.ngroups = st.ngroups + 1
            ng.gidx = st.ngroups
            var ngn = st.gnodes
            ngn.push(ng)
            var names = st.names
            var nidx = st.nameGroups
            names.push(nm)
            nidx.push(ng.gidx)
            ng.items.push(rxParseAlt(st))
            rxExpect(st, 41, "unterminated group")
            ng.closed = true
            return ng
        }
        if (st.py && "aiLmsux-".indexOf(p.charAt(st.i)) >= 0) { return rxParseFlagsP(st) }
        if (st.java) {
            if (k == 62) { // (?>X): X matched once, never backtracked into
                st.i = st.i + 1
                var at = rxNode("atom")
                at.items.push(rxParseAlt(st))
                rxExpect(st, 41, "unterminated group")
                return at
            }
            if (k == 36 || k == 64) { // (?$ (?@
                rxErrAt(st, "Unknown group type", st.i)
                return rxNode("empty")
            }
            return rxParseFlagsJ(st)
        }
        rxErr(st, "unsupported group form (?" + p.charAt(st.i) + ")")
        return rxNode("empty")
    }
    var cg = rxNode("group")
    st.ngroups = st.ngroups + 1
    cg.gidx = st.ngroups
    var cgn = st.gnodes
    cgn.push(cg)
    st.depth = st.depth + 1
    cg.items.push(rxParseAlt(st))
    st.depth = st.depth - 1
    rxExpect(st, 41, st.ere ? "Unmatched ( or \\(" : "unterminated group")
    cg.closed = true
    return cg
}

// rxParseFlagsP reads Python's inline flags; st.i is at the first letter after
// "(?". (?aiLmsux) sets flags for the whole pattern and is legal only where
// nothing but other such groups precedes it (st.gend); (?imsx-imsx:...) scopes them
// to a non-capturing group. Under P, i decides case folding per atom
// (rxParsePiece) and m picks the kind of ^ and $ as they are read, so neither needs
// state in the matcher. a and u are accepted and change nothing here.
function rxParseFlagsP(st) {
    var p = st.p
    var open = st.i - 2
    var on = ""
    var off = ""
    var neg = false
    while (st.i < st.n) {
        var c = p.charAt(st.i)
        if (c == "-" && !neg) {
            neg = true
        } else if ("aiLmsux".indexOf(c) >= 0) {
            if (neg) {
                if ("imsx".indexOf(c) < 0) {
                    rxErr(st, "bad inline flag: cannot turn off flags 'a', 'u' and 'L'")
                    return rxNode("empty")
                }
                off = off + c
            } else {
                on = on + c
            }
        } else {
            break
        }
        st.i = st.i + 1
    }
    if (on.indexOf("L") >= 0) {
        rxErr(st, "bad inline flag: cannot use 'L' flag with a str pattern")
        return rxNode("empty")
    }
    var k = st.i < st.n ? p.charCodeAt(st.i) : 0
    if (k == 41 && !neg) {
        st.i = st.i + 1
        if (open != st.gend) {
            rxErrAt(st, "global flags not at the start of the expression", open)
            return rxNode("empty")
        }
        st.gend = st.i
        st.gflags = st.gflags + on
        rxFlagsSet(st, on, true)
        return rxNode("empty")
    }
    if (k != 58) {
        rxErr(st, neg ? "missing :" : "missing -, : or )")
        return rxNode("empty")
    }
    st.i = st.i + 1
    var icase = st.icase
    var multi = st.multi
    var dotall = st.dotall
    var ext = st.ext
    var pyuni = st.pyuni
    rxFlagsSet(st, on, true)
    rxFlagsSet(st, off, false)
    var g = rxNode("group")
    g.items.push(rxParseAlt(st))
    rxExpect(st, 41, "missing ), unterminated subpattern")
    st.icase = icase
    st.multi = multi
    st.dotall = dotall
    st.ext = ext
    st.pyuni = pyuni
    return g
}

// rxParseFlagsJ reads java.util.regex's (?idmsuxU-idmsuxU) and (?..:X); st.i is
// at the first character after "(?" (Pattern.addFlag / subFlag). Like P, the
// flags are decided per atom as the parser reads it. The inline form changes
// them up to the end of the enclosing group, the scoped one only for X: every
// group puts back what was in force when it opened (rxParseAtom), except an
// inline one, which sets st.inl. c (CANON_EQ) is accepted and changes nothing.
function rxParseFlagsJ(st) {
    var p = st.p
    var on = true
    while (st.i < st.n) {
        var c = p.charAt(st.i)
        if (c == "-" && on) {
            on = false
        } else if ("idmsuxcU".indexOf(c) >= 0) {
            rxFlagsSet(st, c, on)
        } else {
            break
        }
        st.i = st.i + 1
    }
    var k = st.i < st.n ? p.charCodeAt(st.i) : 0
    if (k == 41) {
        st.i = st.i + 1
        st.inl = true
        return rxNode("empty")
    }
    if (k != 58) {
        rxErrAt(st, "Unknown inline modifier", st.i)
        return rxNode("empty")
    }
    st.i = st.i + 1
    var g = rxNode("group")
    g.items.push(rxParseAlt(st))
    rxExpect(st, 41, "unterminated group")
    return g
}

// rxFlagsSet switches the parse-time flags named by letters on or off. U is
// Java's UNICODE_CHARACTER_CLASS, which implies UNICODE_CASE both ways.
function rxFlagsSet(st, letters, v) {
    if (letters.indexOf("i") >= 0) { st.icase = v }
    if (letters.indexOf("m") >= 0) { st.multi = v }
    if (letters.indexOf("s") >= 0) { st.dotall = v }
    if (letters.indexOf("x") >= 0) { st.ext = v }
    if (letters.indexOf("d") >= 0) { st.unix = v }
    if (letters.indexOf("u") >= 0) { st.ucase = v }
    if (letters.indexOf("U") >= 0) {
        st.uclass = v
        st.ucase = v
    }
    if (letters.indexOf("c") >= 0) { st.canon = v }
    if (letters.indexOf("a") >= 0) { st.pyuni = !v }
}

// The flags in force, as rxFlagsSet's letters, and back.
function rxFlagsGet(st) {
    var s = ""
    if (st.icase) { s = s + "i" }
    if (st.multi) { s = s + "m" }
    if (st.dotall) { s = s + "s" }
    if (st.ext) { s = s + "x" }
    if (st.unix) { s = s + "d" }
    if (st.ucase) { s = s + "u" }
    if (st.uclass) { s = s + "U" }
    if (st.canon) { s = s + "c" }
    return s
}
function rxFlagsPut(st, letters) {
    rxFlagsSet(st, "imsxdUc", false)
    rxFlagsSet(st, letters, true)
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
    if (c == 40) {
        st.i = st.i + 1
        if (!st.java) { return rxParseGroup(st) }
        var fl = rxFlagsGet(st)
        var g = rxParseGroup(st)
        if (!st.inl) { rxFlagsPut(st, fl) }
        st.inl = false
        return g
    }
    if (c == 91) {
        st.i = st.i + 1
        if (st.java) { return rxParseClassJ(st) }
        return rxParseClass(st)
    }
    // Java reads a { where an atom belongs as a quantifier of NOTHING: {2}a and
    // a{1}{2} are legal, and a bare { is "Illegal repetition".
    if (c == 123 && st.java) { return rxNode("empty") }
    if (c == 46) { // .
        st.i = st.i + 1
        var dot = rxNode("cls")
        dot.cls.neg = true
        if (!st.dotall) { rxClsAdd(dot.cls, 10, 10) }
        if (st.java && !st.dotall && !st.unix) { rxAddLineSepJ(dot.cls) }
        return dot
    }
    if (c == 94) { // ^
        st.i = st.i + 1
        var b = rxNode("assert")
        b.gidx = RX_BOL
        if (st.py) { b.gidx = st.multi ? RX_MBOL : RX_BOS }
        if (st.java) { b.gidx = !st.multi ? RX_BOS : st.unix ? RX_JUBOL : RX_JBOL }
        return b
    }
    if (c == 36) { // $
        st.i = st.i + 1
        var d = rxNode("assert")
        d.gidx = RX_EOL
        if (st.py) { d.gidx = st.multi ? RX_MEOL : RX_EOSNL }
        if (st.java) { d.gidx = rxEolJ(st, st.multi) }
        return d
    }
    if (c == 92 && st.ere) {
        // A POSIX ERE has no escape SEQUENCES: a backslash always quotes the next
        // character, so \d is the letter d and \1 is the digit 1.
        if (st.i + 1 >= st.n) {
            rxErr(st, "Trailing backslash")
            st.i = st.i + 1
            return rxNode("empty")
        }
        var qc = p.charCodeAt(st.i + 1)
        st.i = st.i + 2
        var qn = rxNode("cls")
        rxClsAdd(qn.cls, qc, qc)
        return qn
    }
    if (c == 92) { // backslash: the zero-width escapes first
        var nxt = 0
        if (st.i + 1 < st.n) { nxt = p.charCodeAt(st.i + 1) }
        // \k<name>: a NAMED backreference. The group it names may be declared LATER
        // in the pattern (a forward reference is legal), so the node is parked on
        // st.pending and resolved once the whole pattern has been read.
        if (nxt == 107 && st.i + 2 < st.n && p.charCodeAt(st.i + 2) == 60) {
            st.i = st.i + 3
            var knm = rxReadGroupName(st, 62)
            var kb = rxNode("bref")
            kb.bname = knm
            if (st.java) {
                // Java resolves the name as it reads it: the group must already
                // be open.
                kb.gidx = rxNameIdx(st.names, st.nameGroups, knm)
                if (kb.gidx < 0 && st.err == "") {
                    rxErrAt(st, "named capturing group <" + knm + "> does not exist", st.i - 1)
                }
                return kb
            }
            if (st.py) {
                // So does Python, and the group must already be closed as well.
                kb.gidx = rxNameIdx(st.names, st.nameGroups, knm)
                if (kb.gidx < 0) {
                    rxErr(st, "unknown group name '" + knm + "'")
                } else if (!st.gnodes[kb.gidx].closed) {
                    rxErr(st, "cannot refer to an open group")
                }
                return kb
            }
            st.pending.push(kb)
            return kb
        }
        if (nxt == 107 && st.java) {
            rxErrAt(st, "\\k is not followed by '<' for named capturing group", st.i + 2)
            return rxNode("empty")
        }
        if (nxt == 82 && st.java) { // Java's \R: \r\n or one vertical-whitespace character
            st.i = st.i + 2
            return rxLinebreakJ()
        }
        if (st.pcre && nxt == 75) { // \K: the match starts here, group 0's SAVE
            st.i = st.i + 2
            var kn = rxNode("op")
            kn.gidx = RX_SAVE
            return kn
        }
        if (st.java && (nxt == 71 || nxt == 88)) { // \G \X
            st.i = st.i + 2
            var gn = rxNode("op")
            gn.gidx = nxt == 71 ? RX_GPOS : RX_XGRAPH
            return gn
        }
        if (st.java && nxt == 98 && st.i + 3 < st.n && p.charCodeAt(st.i + 2) == 123
            && p.charCodeAt(st.i + 3) == 103) { // \b{g}; any other \b{ is \b and a bound
            if (st.i + 4 >= st.n || p.charCodeAt(st.i + 4) != 125) {
                rxErrAt(st, "Illegal/unsupported escape sequence", st.i + 4)
                return rxNode("empty")
            }
            st.i = st.i + 5
            var gb = rxNode("op")
            gb.gidx = RX_GBOUND
            return gb
        }
        if (nxt == 98 || nxt == 66 || nxt == 65 || nxt == 122 || nxt == 90) {
            st.i = st.i + 2
            var a = rxNode("assert")
            if (nxt == 98) { a.gidx = RX_WORDB }
            if (nxt == 66) { a.gidx = RX_NWORDB }
            if (nxt == 65) { a.gidx = RX_BOS }
            if (nxt == 122) { a.gidx = RX_EOS }
            if (nxt == 90) { a.gidx = st.py ? RX_EOS : RX_EOSNL }
            if (st.pyuni && nxt == 98) { a.gidx = RX_PUWORDB }
            if (st.pyuni && nxt == 66) { a.gidx = RX_PUNWORDB }
            // Go's \b and \B are ASCII \w ones (R).
            if (st.java && !st.re2) {
                if (nxt == 98) { a.gidx = st.uclass ? RX_JUWORDB : RX_JWORDB }
                if (nxt == 66) { a.gidx = st.uclass ? RX_JUNWORDB : RX_JNWORDB }
                if (nxt == 90) { a.gidx = rxEolJ(st, false) }
            }
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
        if (e.kind == 3) {
            var pn = rxNode("cls")
            pn.cls = st.pcls
            return pn
        }
        var ln = rxNode("cls")
        rxClsAdd(ln.cls, e.ch, e.ch)
        if (st.java && st.icase) { rxFoldJ(st, ln.cls, true) }
        return ln
    }
    // A ) only closes a group when one is OPEN. At the top level of a POSIX ERE it is
    // an ordinary character - [[ "a)b" =~ ")" ]] matches in bash, it is not an error -
    // where every other dialect here calls it "unmatched )".
    if (c == 41 && (!st.ere || st.depth > 0)) { return rxNode("empty") }
    if (c == 124) { // | - an empty branch
        return rxNode("empty")
    }
    if (c == 42 || c == 43 || c == 63) {
        rxErr(st, st.ere ? "Invalid preceding regular expression"
                         : "quantifier with nothing to repeat")
        st.i = st.i + 1
        return rxNode("empty")
    }
    // A { that opens an interval where no atom precedes it is the same error; a { that
    // is NOT followed by a digit is an ordinary character (bash takes "a{b" literally).
    if (c == 123 && st.ere && st.i + 1 < st.n && rxIsDigitCode(p.charCodeAt(st.i + 1))) {
        rxErr(st, "Invalid preceding regular expression")
        st.i = st.i + 1
        return rxNode("empty")
    }
    if (st.java || st.py) { c = rxCpAt(p, st.i) }
    st.i = st.i + (c > 65535 ? 2 : 1)
    var lit = rxNode("cls")
    rxClsAdd(lit.cls, c, c)
    if (st.java && st.icase) { rxFoldJ(st, lit.cls, true) }
    return lit
}

// rxEolJ is the kind of a J $ (multi true) or \Z (multi false).
function rxEolJ(st, multi) {
    if (st.unix) { return multi ? RX_JUMEOL : RX_JUEOL }
    return multi ? RX_JMEOL : RX_JEOL
}

// rxAddLineSepJ adds Java's line terminators besides \n: \r, U+0085, U+2028
// and U+2029.
function rxAddLineSepJ(cls) {
    rxClsAdd(cls, 13, 13)
    rxClsAdd(cls, 133, 133)
    rxClsAdd(cls, 8232, 8233)
}

// rxFoldJ makes a J class read under CASE_INSENSITIVE match either case: its
// ranges get their ASCII letters' other case, or under UNICODE_CASE are marked
// u (rxClsHitU). A single character (single) is turned into the lowercase of
// its uppercase, which is what Java's SingleU compares; an unmarked one keeps
// only itself, because it has no other case. \p{..} classes and shorthands are
// never folded: Java decides their case-insensitive reading by name
// (jxPropCls in lib/java-regex.js).
function rxFoldJ(st, cls, single) {
    if (!st.ucase) {
        rxClsFold(cls)
        return
    }
    if (single) {
        var c = cls.lo[0]
        var up = rxCaseU(c, 1)
        var low = rxCaseU(up, 0)
        if (up == low) { return }
        var lo = cls.lo
        var hi = cls.hi
        lo[0] = low
        hi[0] = low
    }
    cls.u = true
}

// rxLinebreakJ is the node of Java's \R: CR LF, or one character of \v.
function rxLinebreakJ() {
    var crlf = rxNode("cat")
    var cr = rxNode("cls")
    rxClsAdd(cr.cls, 13, 13)
    var lf = rxNode("cls")
    rxClsAdd(lf.cls, 10, 10)
    var ci = crlf.items
    ci.push(cr)
    ci.push(lf)
    var one = rxNode("cls")
    rxClsAddSp(one.cls, RX_SJV)
    var alt = rxNode("alt")
    var ai = alt.items
    ai.push(crlf)
    ai.push(one)
    var g = rxNode("group")
    var gi = g.items
    gi.push(alt)
    return g
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

// rxParseBoundJ reads a java.util.regex bound at st.i, where a { is always one:
// it answers null, with Java's error, for anything that does not read as
// {n}, {n,} or {n,m} with n and m ints. Under x whitespace and comments may
// follow each digit and the comma (Pattern.read skips them).
function rxParseBoundJ(st) {
    var p = st.p
    var j = st.i + 1
    if (j >= st.n || !rxIsDigitCode(p.charCodeAt(j))) {
        rxErrAt(st, "Illegal repetition", st.i + 1)
        return null
    }
    var minV = 0
    while (j < st.n && rxIsDigitCode(p.charCodeAt(j))) {
        minV = minV * 10 + (p.charCodeAt(j) - 48)
        if (minV > 2147483647) {
            rxErrAt(st, "Illegal repetition range", j)
            return null
        }
        j = rxSkipXAt(st, j + 1)
    }
    var maxV = minV
    if (j < st.n && p.charCodeAt(j) == 44) { // ,
        j = rxSkipXAt(st, j + 1)
        maxV = -1
        if (j < st.n && rxIsDigitCode(p.charCodeAt(j))) { maxV = 0 }
        while (j < st.n && rxIsDigitCode(p.charCodeAt(j))) {
            maxV = maxV * 10 + (p.charCodeAt(j) - 48)
            if (maxV > 2147483647) {
                rxErrAt(st, "Illegal repetition range", j)
                return null
            }
            j = rxSkipXAt(st, j + 1)
        }
    }
    if (j >= st.n || p.charCodeAt(j) != 125) { // }
        rxErrAt(st, "Unclosed counted closure", j)
        return null
    }
    if (maxV >= 0 && maxV < minV) {
        rxErrAt(st, "Illegal repetition range", j)
        return null
    }
    st.i = j + 1
    return {min: minV, max: maxV}
}

function rxParsePiece(st) {
    var atom = rxParseAtom(st)
    // Python's IGNORECASE of a str pattern folds case by Unicode (rxClsFoldP),
    // by ASCII under re.ASCII / (?a).
    if (st.py && st.icase) {
        if (atom.t == "cls") {
            rxClsFold(atom.cls)
            if (st.pyuni) { rxClsFoldP(atom.cls) }
        }
        if (atom.t == "bref") { atom.min = st.pyuni ? 2 : 1 }
    }
    if (st.java && st.icase && atom.t == "bref") { atom.min = st.ucase ? 2 : 1 }
    rxSkipX(st)
    var seen = 0
    while (st.i < st.n) {
        var c = st.p.charCodeAt(st.i)
        var minV = 0
        var maxV = 0
        var counted = false
        if (c == 42) { st.i = st.i + 1; minV = 0; maxV = -1 }
        else if (c == 43) { st.i = st.i + 1; minV = 1; maxV = -1 }
        else if (c == 63) { st.i = st.i + 1; minV = 0; maxV = 1 }
        else if (c == 123 && st.java) {
            // A second {..} starts a new piece over an empty atom (rxParseAtom).
            if (seen > 0) { return atom }
            var bj = rxParseBoundJ(st)
            if (bj == null) { return atom }
            minV = bj.min
            maxV = bj.max
            counted = true
        }
        else if (c == 123) {
            // In a POSIX ERE a { opens an interval EXACTLY WHEN a digit follows, and an
            // interval that then fails to close is an error: bash matches "a{b" and
            // "a{" literally but rejects "a{1" outright. Everywhere else an unreadable
            // brace stays a literal brace.
            var ivl = st.ere && st.i + 1 < st.n && rxIsDigitCode(st.p.charCodeAt(st.i + 1))
            var bd = rxParseBound(st)
            if (bd == null) {
                if (ivl) { rxErr(st, "Invalid content of \\{\\}") }
                return atom
            }
            minV = bd.min
            maxV = bd.max
            if (maxV >= 0 && maxV < minV) {
                rxErr(st, "quantifier bounds are out of order")
                return atom
            }
        } else {
            return atom
        }
        // A second quantifier on the same atom - a**, a?*, a{2}{3}, and in an ERE also
        // a*? - is a syntax error in bash, node and python3, and a** is one in java too
        // (java reads the {3} of a{2}{3} as a new piece, see above). Ruby 2.6 is the
        // one dissenter: it warns about the "redundant nested repeat operator" and
        // folds it.
        seen = seen + 1
        if (seen > 1) {
            rxErr(st, st.ere ? "Invalid preceding regular expression"
                             : "multiple repeat")
            return atom
        }
        var rep = rxNode("rep")
        rep.min = minV
        rep.max = maxV
        rep.greedy = true
        rep.cnt = counted && !st.re2
        // A POSIX ERE has no lazy or possessive form, so the ? of "a*?" is a SECOND
        // quantifier there rather than a modifier of the first.
        if (st.java) { rxSkipX(st) }
        var poss = false
        if (st.i < st.n && !st.ere) {
            var q = st.p.charCodeAt(st.i)
            if (q == 63) { rep.greedy = false; st.i = st.i + 1 }
            else if (q == 43) { st.i = st.i + 1; poss = st.java }
        }
        // Outside J a possessive quantifier is read as greedy. Under J, X*+ is
        // Pattern.java's possessive Curly: every iteration of X is its first
        // success, and the loop never gives one back - (?>(?>X)*).
        // Under J a greedy X*, X+ or X{n,} of one class is Pattern's
        // CharPropertyGreedy, which reads a pair a region end splits as its
        // high half (RX_CLASS).
        if (st.java && rep.greedy && !poss && maxV < 0 && atom.t == "cls") { atom.cls.cpg = true }
        if (poss && atom.t != "cls") {
            var inner = rxNode("atom")
            inner.items.push(atom)
            atom = inner
        }
        rep.items.push(atom)
        atom = rep
        if (poss) {
            var outer = rxNode("atom")
            outer.items.push(rep)
            atom = outer
        }
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
        if (c == 124) { break } // |
        if (c == 41 && (!st.ere || st.depth > 0)) { break } // )
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
    // A POSIX ERE branch may not be empty: bash rejects "", "a|", "|a", "(a|)" and
    // "(|a)" alike with "empty (sub)expression". The one exception is a group whose
    // WHOLE body is empty - "()" and "(())" match the empty string in bash 5.3 - so
    // the emptiness is only an error for a branch of a real alternation, or for the
    // top level of the pattern.
    if (st.ere && st.err == "") {
        var ei = 0
        while (ei < items.length) {
            if (items[ei].items.length == 0 && (items.length > 1 || st.depth == 0)) {
                rxErr(st, "empty (sub)expression")
                ei = items.length
            }
            ei = ei + 1
        }
    }
    if (items.length == 1) { return items[0] }
    return alt
}


// ===== The emitter =====
//
// prog is four parallel arrays: ops[], as[], bs[] and clss[]. Same reason as the
// class representation - no mixed-type record anywhere.

// cmin/cmax/cgreedy describe counted loop i (RX_CINIT .. RX_CINC), lbmin/lbmax
// the length range of lookbehind i (-1: unbounded).
function rxNewProg() {
    return {ops: [], as: [], bs: [], clss: [], nmarks: 0, jsrep: false, java: false, re2: false, dummy: rxNewCls(),
            cmin: [], cmax: [], cgreedy: [], lbmin: [], lbmax: []}
}

// The smallest / largest group NUMBER declared inside a subtree, or -1 for none.
// Only a "group" node carries a group number in gidx: "assert" keeps the assertion
// kind there, "bref" the group it points at and "look" the negation flag.
function rxLoG(node) {
    var best = -1
    if (node.t == "group" && node.gidx >= 0) { best = node.gidx }
    var items = node.items
    var i = 0
    while (i < items.length) {
        var k = rxLoG(items[i])
        if (k >= 0 && (best < 0 || k < best)) { best = k }
        i = i + 1
    }
    return best
}

function rxHiG(node) {
    var best = -1
    if (node.t == "group" && node.gidx >= 0) { best = node.gidx }
    var items = node.items
    var i = 0
    while (i < items.length) {
        var k = rxHiG(items[i])
        if (k > best) { best = k }
        i = i + 1
    }
    return best
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
        rxAdd(pg, node.cls.x ? RX_CLASSX : RX_CLASS, 0, 0, node.cls)
        return
    }
    if (t == "assert") {
        rxAdd(pg, RX_ASSERT, node.gidx, 0, pg.dummy)
        return
    }
    if (t == "bref") {
        rxAdd(pg, RX_BREF, node.gidx, node.min, pg.dummy)
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
    if (t == "op") {
        rxAdd(pg, node.gidx, 0, 0, pg.dummy)
        return
    }
    if (t == "atom") {
        var am = rxAdd(pg, RX_ATOM, 0, 0, pg.dummy)
        rxEmit(pg, node.items[0])
        rxAdd(pg, RX_ATOMEND, 0, 0, pg.dummy)
        rxSetA(pg, am, rxHere(pg))
        return
    }
    if (t == "lookb") {
        var lbs = pg.lbmin
        var id = lbs.length
        lbs.push(rxMinLen(node.items[0]))
        var lbx = pg.lbmax
        // Lengths count characters; a J character is up to two UTF-16 units.
        var lbm = rxMaxLen(node.items[0])
        lbx.push(pg.java && lbm > 0 ? rxLenCap(2 * lbm) : lbm)
        var lb = rxAdd(pg, RX_LOOKB, 0, 2 * id + (node.gidx == 1 ? 1 : 0), pg.dummy)
        rxEmit(pg, node.items[0])
        rxAdd(pg, RX_LBEND, 0, 0, pg.dummy)
        rxSetA(pg, lb, rxHere(pg))
        return
    }
    if (t == "rep") {
        if (node.cnt) {
            rxEmitCount(pg, node)
            return
        }
        rxEmitRep(pg, node)
        return
    }
}

// The fewest / most characters a node can match; -1 is unbounded. A lookbehind
// tries only the start positions this range allows.
function rxMinLen(node) {
    var t = node.t
    if (t == "cls") { return 1 }
    // \X counts one toward the shortest, and nothing toward the longest, as
    // Java's XGrapheme.study does.
    if (t == "op") { return node.gidx == RX_XGRAPH ? 1 : 0 }
    var items = node.items
    if (t == "cat") {
        var sum = 0
        var i = 0
        while (i < items.length) {
            sum = rxLenCap(sum + rxMinLen(items[i]))
            i = i + 1
        }
        return sum
    }
    if (t == "alt") {
        var best = -1
        var k = 0
        while (k < items.length) {
            var m = rxMinLen(items[k])
            if (best < 0 || m < best) { best = m }
            k = k + 1
        }
        return best < 0 ? 0 : best
    }
    if (t == "group" || t == "atom") { return items.length > 0 ? rxMinLen(items[0]) : 0 }
    if (t == "rep") { return rxLenCap(node.min * rxMinLen(items[0])) }
    return 0
}

// rxLookbVarP: does a look-behind under node match a varying number of characters?
function rxLookbVarP(st, node) {
    if (node.t == "lookb" && rxWidthP(st, node.items[0], 0) < 0) { return true }
    var items = node.items
    var i = 0
    while (i < items.length) {
        if (rxLookbVarP(st, items[i])) { return true }
        i = i + 1
    }
    return false
}

// rxLookbLongC: does a lookbehind under C reach past PCRE2's limit of 255
// characters (MAX_VARLOOKBEHIND), or have no limit at all?
function rxLookbLongC(node) {
    if (node.t == "lookb") {
        var m = rxMaxLen(node.items[0])
        if (m < 0 || m > 255) { return true }
    }
    var items = node.items
    var i = 0
    while (i < items.length) {
        if (rxLookbLongC(items[i])) { return true }
        i = i + 1
    }
    return false
}

// rxWidthP is the one number of characters a node always matches under P, or -1
// when it can vary: Python's look-behind must be fixed-width. A backreference is
// as wide as the group it names (st.gnodes); depth stops a group that refers to
// itself.
function rxWidthP(st, node, depth) {
    var t = node.t
    if (t == "cls") { return 1 }
    var items = node.items
    if (t == "cat" || t == "alt") {
        var acc = 0
        var i = 0
        while (i < items.length) {
            var w = rxWidthP(st, items[i], depth)
            if (w < 0) { return -1 }
            if (t == "cat") { acc = rxLenCap(acc + w) } else if (i > 0 && w != acc) { return -1 } else { acc = w }
            i = i + 1
        }
        return acc
    }
    if (t == "group") { return rxWidthP(st, items[0], depth) }
    if (t == "rep") {
        var b = rxWidthP(st, items[0], depth)
        if (b == 0) { return 0 }
        if (b < 0 || node.min != node.max) { return -1 }
        return rxLenCap(node.min * b)
    }
    if (t == "bref") {
        var g = node.bname != "" ? rxNameIdx(st.names, st.nameGroups, node.bname) : node.gidx
        var gn = st.gnodes
        if (g < 1 || g >= gn.length || depth > st.ngroups) { return -1 }
        return rxWidthP(st, gn[g], depth + 1)
    }
    return 0
}

// Lengths saturate at the int range, which no text reaches, so that the products
// of nested bounds cannot overflow.
function rxLenCap(n) { return n > 2147483647 ? 2147483647 : n }

function rxMaxLen(node) {
    var t = node.t
    if (t == "cls") { return 1 }
    if (t == "bref") { return -1 }
    var items = node.items
    if (t == "cat" || t == "alt") {
        var acc = 0
        var i = 0
        while (i < items.length) {
            var m = rxMaxLen(items[i])
            if (m < 0) { return -1 }
            if (t == "cat") { acc = rxLenCap(acc + m) } else if (m > acc) { acc = m }
            i = i + 1
        }
        return acc
    }
    if (t == "group" || t == "atom") { return items.length > 0 ? rxMaxLen(items[0]) : 0 }
    if (t == "rep") {
        var b = rxMaxLen(items[0])
        if (b == 0) { return 0 }
        if (b < 0 || node.max < 0) { return -1 }
        return rxLenCap(node.max * b)
    }
    return 0
}

// rxEmitCount emits a J {n,m} repeat as ONE copy of the body inside a counted
// loop, so a{2147483647} costs what a{2} does. An iteration that consumed
// nothing leaves the loop, minimum reached or not (java.util.regex's Loop).
function rxEmitCount(pg, node) {
    var cm = pg.cmin
    var c = cm.length
    cm.push(node.min)
    var cx = pg.cmax
    cx.push(node.max)
    var cg = pg.cgreedy
    cg.push(node.greedy ? 1 : 0)
    rxAdd(pg, RX_CINIT, c, 0, pg.dummy)
    var top = rxAdd(pg, RX_CLOOP, c, 0, pg.dummy)
    rxEmit(pg, node.items[0])
    rxAdd(pg, RX_CINC, c, top, pg.dummy)
    rxSetB(pg, top, rxHere(pg))
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

// Under the j dialect every iteration of a repeat starts by CLEARING the captures
// declared inside the repeated atom, so a group that took part in an earlier
// iteration but not the last one reads as "did not take part":
// /(?:(a)|b)+/.exec("ab") leaves group 1 undefined in node, and bash reports the
// same group empty. Perl, Python, Ruby and Java carry the earlier value forward,
// so nothing is emitted for them. The group numbers inside a subtree are
// contiguous ('(' is counted left to right), so one lo..hi range covers them.
function rxEmitClear(pg, body) {
    if (!pg.jsrep) { return }
    var lo = rxLoG(body)
    if (lo < 0) { return }
    rxAdd(pg, RX_CLEAR, lo, rxHiG(body), pg.dummy)
}

function rxEmitRep(pg, node) {
    if (pg.re2 && node.max < 0) {
        rxEmitRepRe2(pg, node)
        return
    }
    var body = node.items[0]
    var i = 0
    while (i < node.min) {
        rxEmitClear(pg, body)
        rxEmit(pg, body)
        i = i + 1
    }
    if (node.max < 0) {
        // Unbounded tail. The MARK/PROG pair is the empty-body guard: a body that
        // consumed nothing leaves the loop instead of spinning ((a*)* is a real
        // pattern, and without this it never terminates). A body that always
        // consumes something needs none.
        var guard = rxMinLen(body) == 0
        var mk = pg.nmarks
        if (guard) { pg.nmarks = pg.nmarks + 1 }
        var l1 = rxHere(pg)
        var sp = rxAdd(pg, RX_SPLIT, 0, 0, pg.dummy)
        if (guard) { rxAdd(pg, RX_MARK, mk, 0, pg.dummy) }
        rxEmitClear(pg, body)
        rxEmit(pg, body)
        if (guard) {
            var pgi = rxAdd(pg, RX_PROG, mk, 0, pg.dummy)
            rxSetB(pg, pgi, rxHere(pg) + 1)
        }
        rxAdd(pg, RX_JMP, l1, 0, pg.dummy)
        var out = rxHere(pg)
        if (node.greedy) {
            rxSetA(pg, sp, l1 + 1)
            rxSetB(pg, sp, out)
        } else {
            rxSetA(pg, sp, out)
            rxSetB(pg, sp, l1 + 1)
        }
        return
    }
    var extra = node.max - node.min
    var splits = []
    var k = 0
    while (k < extra) {
        var s = rxAdd(pg, RX_SPLIT, 0, 0, pg.dummy)
        splits.push(s)
        rxEmitClear(pg, body)
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


// rxEmitRepRe2 is an unbounded repeat under R, compiled as Go compiles it:
// x{n,} is n-1 copies of x and then x+, x* is (x+)?, and Go's matcher never
// enters the same instruction at the same position twice. So the first
// iteration of the x+ may match empty, and is then the last one (the loop
// stands where that iteration began); a later iteration that matches empty
// brings the loop back to where it already stood and FAILS (PROG with b = -1),
// which undoes it, captures included. regexp/syntax's compile.go.
function rxEmitRepRe2(pg, node) {
    var body = node.items[0]
    var i = 1
    while (i < node.min) {
        rxEmit(pg, body)
        i = i + 1
    }
    var skip = -1
    if (node.min == 0) { skip = rxAdd(pg, RX_SPLIT, 0, 0, pg.dummy) }
    var m0 = pg.nmarks
    pg.nmarks = m0 + 2
    rxAdd(pg, RX_MARK, m0, 0, pg.dummy)
    rxEmit(pg, body)
    var keep = rxAdd(pg, RX_PROG, m0, 0, pg.dummy)
    var l1 = rxHere(pg)
    var sp = rxAdd(pg, RX_SPLIT, 0, 0, pg.dummy)
    rxAdd(pg, RX_MARK, m0 + 1, 0, pg.dummy)
    rxEmit(pg, body)
    rxAdd(pg, RX_PROG, m0 + 1, 0 - 1, pg.dummy)
    rxAdd(pg, RX_JMP, l1, 0, pg.dummy)
    var out = rxHere(pg)
    rxSetB(pg, keep, out)
    rxSplitTo(pg, sp, l1 + 1, out, node.greedy)
    if (skip >= 0) { rxSplitTo(pg, skip, skip + 1, out, node.greedy) }
}

// rxSplitTo points the SPLIT at `at` into a loop body and out of it, the body
// first when greedy.
function rxSplitTo(pg, at, body, out, greedy) {
    if (greedy) {
        rxSetA(pg, at, body)
        rxSetB(pg, at, out)
    } else {
        rxSetA(pg, at, out)
        rxSetB(pg, at, body)
    }
}

// ===== The RE2 dialect (flag R): Go's regexp/syntax =====
//
// rxRe2 reads a pattern the way Go's regexp.Compile does (regexp/syntax's Parse
// with the Perl flags) and answers {out, err, names, nameGroups}: err is Go's
// "error parsing regexp: CODE: `EXPR`" for a pattern Go rejects, and otherwise
// out is the same pattern spelled in the J dialect so that J reads exactly what
// Go means. A literal is \x{h}, inside (?iu:..) under (?i); . and the Perl and
// POSIX classes are explicit classes; ^ and $ are \A and \z, or under (?m) an
// alternation with a newline look-around; \pN and \p{Greek} are J's properties;
// (?U) is applied to every quantifier; a named group is a plain group whose
// name the result carries, since Go allows any [A-Za-z0-9_]+. Matching is J's
// with re2 set (rxEmitRepRe2, the ASCII \b, code-point steps).
// Twin: rxRe2 in abnf/jsrtregex.go.

// Go's unicode.Categories keys, unicode.CategoryAliases (Name=Key) and
// unicode.Scripts keys (Unicode 17), for \p{..}.
var RX_RE2_CATS = " C Cc Cf Cn Co Cs L Ll Lm Lo Lt Lu M Mc Me Mn N Nd Nl No P Pc Pd Pe Pf Pi Po Ps S Sc Sk Sm So Z Zl Zp Zs "
var RX_RE2_CALIAS = " Cased_Letter=LC Close_Punctuation=Pe Combining_Mark=M Connector_Punctuation=Pc Control=Cc"
    + " Currency_Symbol=Sc Dash_Punctuation=Pd Decimal_Number=Nd Enclosing_Mark=Me Final_Punctuation=Pf"
    + " Format=Cf Initial_Punctuation=Pi Letter=L Letter_Number=Nl Line_Separator=Zl Lowercase_Letter=Ll"
    + " Mark=M Math_Symbol=Sm Modifier_Letter=Lm Modifier_Symbol=Sk Nonspacing_Mark=Mn Number=N"
    + " Open_Punctuation=Ps Other=C Other_Letter=Lo Other_Number=No Other_Punctuation=Po Other_Symbol=So"
    + " Paragraph_Separator=Zp Private_Use=Co Punctuation=P Separator=Z Space_Separator=Zs Spacing_Mark=Mc"
    + " Surrogate=Cs Symbol=S Titlecase_Letter=Lt Unassigned=Cn Uppercase_Letter=Lu cntrl=Cc digit=Nd punct=P "
var RX_RE2_SCRIPTS = " Adlam Ahom Anatolian_Hieroglyphs Arabic Armenian Avestan Balinese Bamum Bassa_Vah Batak"
    + " Bengali Beria_Erfe Bhaiksuki Bopomofo Brahmi Braille Buginese Buhid Canadian_Aboriginal Carian"
    + " Caucasian_Albanian Chakma Cham Cherokee Chorasmian Common Coptic Cuneiform Cypriot Cypro_Minoan"
    + " Cyrillic Deseret Devanagari Dives_Akuru Dogra Duployan Egyptian_Hieroglyphs Elbasan Elymaic"
    + " Ethiopic Garay Georgian Glagolitic Gothic Grantha Greek Gujarati Gunjala_Gondi Gurmukhi"
    + " Gurung_Khema Han Hangul Hanifi_Rohingya Hanunoo Hatran Hebrew Hiragana Imperial_Aramaic Inherited"
    + " Inscriptional_Pahlavi Inscriptional_Parthian Javanese Kaithi Kannada Katakana Kawi Kayah_Li"
    + " Kharoshthi Khitan_Small_Script Khmer Khojki Khudawadi Kirat_Rai Lao Latin Lepcha Limbu Linear_A"
    + " Linear_B Lisu Lycian Lydian Mahajani Makasar Malayalam Mandaic Manichaean Marchen Masaram_Gondi"
    + " Medefaidrin Meetei_Mayek Mende_Kikakui Meroitic_Cursive Meroitic_Hieroglyphs Miao Modi Mongolian"
    + " Mro Multani Myanmar Nabataean Nag_Mundari Nandinagari New_Tai_Lue Newa Nko Nushu"
    + " Nyiakeng_Puachue_Hmong Ogham Ol_Chiki Ol_Onal Old_Hungarian Old_Italic Old_North_Arabian"
    + " Old_Permic Old_Persian Old_Sogdian Old_South_Arabian Old_Turkic Old_Uyghur Oriya Osage Osmanya"
    + " Pahawh_Hmong Palmyrene Pau_Cin_Hau Phags_Pa Phoenician Psalter_Pahlavi Rejang Runic Samaritan"
    + " Saurashtra Sharada Shavian Siddham Sidetic SignWriting Sinhala Sogdian Sora_Sompeng Soyombo"
    + " Sundanese Sunuwar Syloti_Nagri Syriac Tagalog Tagbanwa Tai_Le Tai_Tham Tai_Viet Tai_Yo Takri"
    + " Tamil Tangsa Tangut Telugu Thaana Thai Tibetan Tifinagh Tirhuta Todhri Tolong_Siki Toto"
    + " Tulu_Tigalari Ugaritic Vai Vithkuqi Wancho Warang_Citi Yezidi Yi Zanabazar_Square "
// The scripts Unicode 17 added, which J's tables (Java's Unicode) do not know:
// their characters are unassigned there, so \p{Sidetic} matches nothing.
var RX_RE2_NEWSC = " Beria_Erfe Sidetic Tai_Yo Tolong_Siki "
// regexp/syntax's perl_groups.go: the ranges of \d \s \w and of the POSIX
// [:name:] classes, as pairs of characters (low, high); "" for another name.
function rxRe2Spec(name) {
    if (name == "d" || name == "digit") { return "09" }
    if (name == "s") { return "\t\n\f\r  " }
    if (name == "w" || name == "word") { return "09AZ__az" }
    if (name == "alnum") { return "09AZaz" }
    if (name == "alpha") { return "AZaz" }
    if (name == "ascii") { return "\u0000\u007f" }
    if (name == "blank") { return "\t\t  " }
    if (name == "cntrl") { return "\u0000\u001f\u007f\u007f" }
    if (name == "graph") { return "!~" }
    if (name == "lower") { return "az" }
    if (name == "print") { return " ~" }
    if (name == "punct") { return "!/:@[`{~" }
    if (name == "space") { return "\t\r  " }
    if (name == "upper") { return "AZ" }
    if (name == "xdigit") { return "09AFaf" }
    return ""
}

function rxRe2Fail(t, code, expr) {
    if (t.err == "") { t.err = "error parsing regexp: " + code + ": `" + expr + "`" }
}

// The parse state: the flags in force, and one frame per open group holding the
// alternatives read so far (alts, nalts), the atoms of the current one (cur,
// with each atom's need) and the flags to restore at its ')'.
function rxRe2Open(t, cap) {
    var f = {alts: "", nalts: 0, cur: [], needs: [], need: 0, cap: cap,
             fold: t.fold, multi: t.multi, dotnl: t.dotnl, lazy: t.lazy}
    var fr = t.frames
    if (t.depth + 1 < fr.length) { fr[t.depth + 1] = f } else { fr.push(f) }
    t.depth = t.depth + 1
}

function rxRe2Push(t, text, need) {
    var f = t.frames[t.depth]
    var cur = f.cur
    cur.push(text)
    var needs = f.needs
    needs.push(need)
}

// The text of a group's body: its alternatives and the current one.
function rxRe2Body(f) {
    var cur = f.cur
    var s = ""
    var i = 0
    while (i < cur.length) {
        s = s + cur[i]
        i = i + 1
    }
    if (f.nalts == 0) { return s }
    return f.alts + "|" + s
}

// A group's need: the largest over its atoms. need is what regexp/syntax's
// repeatIsValid asks of the repeats inside an atom - the least n for which they
// fit in n copies - so a counted repeat {..m} over an atom of need k needs
// max(m, m * k), and Go rejects one whose need passes 1000.
function rxRe2Need(f) {
    var n = f.need
    var needs = f.needs
    var i = 0
    while (i < needs.length) {
        if (needs[i] > n) { n = needs[i] }
        i = i + 1
    }
    return n
}

function rxRe2Bar(t) {
    var f = t.frames[t.depth]
    f.need = rxRe2Need(f)
    f.alts = rxRe2Body(f)
    f.nalts = f.nalts + 1
    f.cur = []
    f.needs = []
}

function rxRe2Close(t) {
    if (t.depth == 0) {
        rxRe2Fail(t, "unexpected )", t.p)
        return
    }
    var f = t.frames[t.depth]
    t.depth = t.depth - 1
    t.fold = f.fold
    t.multi = f.multi
    t.dotnl = f.dotnl
    t.lazy = f.lazy
    var body = rxRe2Body(f)
    rxRe2Push(t, f.cap > 0 ? "(" + body + ")" : "(?:" + body + ")", rxRe2Need(f))
    t.i = t.i + 1
}

function rxRe2Hex(c) {
    var d = "0123456789ABCDEF"
    var s = ""
    var v = c
    while (v >= 16) {
        s = d.charAt(v % 16) + s
        v = Math.floor(v / 16)
    }
    return "\\x{" + d.charAt(v) + s + "}"
}

function rxRe2Lit(t, c) {
    var s = rxRe2Hex(c)
    if (rxIsWordCode(c) && c != 95) { s = rxCpStr(c) }
    rxRe2Push(t, t.fold ? "(?iu:" + s + ")" : s, 0)
}

// A class under construction: lo/hi range pairs and J property items (props).
function rxRe2Cls() { return {lo: [], hi: [], props: ""} }

function rxRe2Range(cls, a, b) {
    var lo = cls.lo
    var hi = cls.hi
    lo.push(a)
    hi.push(b)
}

// rxRe2Group adds the ranges of spec (rxRe2Spec), or every character outside
// them when neg.
function rxRe2Group(cls, spec, neg) {
    var from = 0
    var k = 0
    while (k < spec.length) {
        var a = spec.charCodeAt(k)
        var b = spec.charCodeAt(k + 1)
        k = k + 2
        if (!neg) { rxRe2Range(cls, a, b) }
        else {
            if (a > from) { rxRe2Range(cls, from, a - 1) }
            from = b + 1
        }
    }
    if (neg) { rxRe2Range(cls, from, 1114111) }
}

function rxRe2ClsText(cls, neg) {
    var s = neg ? "[^" : "["
    var lo = cls.lo
    var hi = cls.hi
    var i = 0
    while (i < lo.length) {
        s = s + rxRe2Hex(lo[i])
        if (hi[i] != lo[i]) { s = s + "-" + rxRe2Hex(hi[i]) }
        i = i + 1
    }
    return s + cls.props + "]"
}

function rxRe2PushCls(t, cls, neg) {
    var s = rxRe2ClsText(cls, neg)
    rxRe2Push(t, t.fold ? "(?iu:" + s + ")" : s, 0)
}

// regexp/syntax's canonicalName: no _ - or space, the first letter upper case
// and the others lower case.
function rxRe2Canon(name) {
    var out = ""
    var first = true
    var i = 0
    while (i < name.length) {
        var c = name.charCodeAt(i)
        i = i + 1
        if (c == 95 || c == 45 || c == 32) { continue }
        if (first) {
            if (c >= 97 && c <= 122) { c = c - 32 }
            first = false
        } else if (c >= 65 && c <= 90) { c = c + 32 }
        out = out + String.fromCharCode(c)
    }
    return out
}

// rxRe2Word finds the word in table (space-separated) whose canonical name is
// canon, from index 1 on, and answers it, or "".
function rxRe2Word(table, canon, sep) {
    var i = 1
    while (i < table.length) {
        var e = table.indexOf(" ", i)
        var w = table.substring(i, e)
        var key = w
        var k = sep.length > 0 ? w.indexOf(sep) : 0 - 1
        if (k >= 0) { key = w.substring(0, k) }
        if (rxRe2Canon(key) == canon) { return w }
        i = e + 1
    }
    return ""
}

// The J item for Go's \p{name} (sign -1: \P{name}), or "" when Go knows no such
// name: regexp/syntax's unicodeTable.
function rxRe2Prop(name, sign) {
    var p = sign > 0 ? "\\p{" : "\\P{"
    var c = rxRe2Canon(name)
    if (c == "Any") { return p + "all}" }
    if (c == "Assigned") { return (sign > 0 ? "\\P{" : "\\p{") + "Cn}" }
    if (c == "Ascii") { return p + "ASCII}" }
    if (c == "Lc") { return p + "LC}" }
    var cat = rxRe2Word(RX_RE2_CATS, c, "")
    if (cat == "") {
        var al = rxRe2Word(RX_RE2_CALIAS, c, "=")
        if (al != "") { cat = al.substring(al.indexOf("=") + 1, al.length) }
    }
    if (cat == "LC") { return p + "LC}" }
    if (cat != "") { return p + cat + "}" }
    var sc = rxRe2Word(RX_RE2_SCRIPTS, c, "")
    if (sc == "") { return "" }
    if (RX_RE2_NEWSC.indexOf(" " + sc + " ") >= 0) { return (sign > 0 ? "\\P{" : "\\p{") + "all}" }
    return p + "Is" + sc.toUpperCase() + "}"
}

// \p.. or \P.. at index at: its J item goes into cls, and the index after it
// is the answer (regexp/syntax's parseUnicodeClass).
function rxRe2Uni(t, at, cls) {
    var p = t.p
    var sign = p.charCodeAt(at + 1) == 80 ? 0 - 1 : 1
    var j = at + 2
    var seq = ""
    var name = ""
    if (j >= t.n) {
        seq = p.substring(at, j)
    } else if (p.charCodeAt(j) != 123) {
        j = j + (rxCpAt(p, j) > 65535 ? 2 : 1)
        seq = p.substring(at, j)
        name = p.substring(at + 2, j)
    } else {
        var end = p.indexOf("}", at)
        if (end < 0) {
            rxRe2Fail(t, "invalid character class range", p.substring(at, t.n))
            return j
        }
        seq = p.substring(at, end + 1)
        name = p.substring(at + 3, end)
        j = end + 1
    }
    if (name.length > 0 && name.charCodeAt(0) == 94) {
        sign = 0 - sign
        name = name.substring(1, name.length)
    }
    var item = rxRe2Prop(name, sign)
    if (item == "") {
        rxRe2Fail(t, "invalid character class range", seq)
        return j
    }
    cls.props = cls.props + item
    return j
}

// The Perl class \d \D \s \S \w \W whose letter is c, into cls; false for any
// other letter.
function rxRe2Perl(cls, c) {
    var lc = c >= 65 && c <= 90 ? c + 32 : c
    if (lc != 100 && lc != 115 && lc != 119) { return false }
    rxRe2Group(cls, rxRe2Spec(String.fromCharCode(lc)), lc != c)
    return true
}

// The escape at index at (regexp/syntax's parseEscape): its character, with the
// index after it in t.k, or -1 after reporting Go's error.
function rxRe2Esc(t, at) {
    var p = t.p
    var j = at + 1
    if (j >= t.n) {
        rxRe2Fail(t, "trailing backslash at end of expression", "")
        return 0 - 1
    }
    var c = rxCpAt(p, j)
    j = j + (c > 65535 ? 2 : 1)
    t.k = j
    if (c < 128 && !(rxIsWordCode(c) && c != 95)) { return c }
    var octal = c == 48
    if (c >= 49 && c <= 55) { octal = j < t.n && p.charCodeAt(j) >= 48 && p.charCodeAt(j) <= 55 }
    if (octal) {
        var r = c - 48
        var n = 1
        while (n < 3 && j < t.n && p.charCodeAt(j) >= 48 && p.charCodeAt(j) <= 55) {
            r = r * 8 + p.charCodeAt(j) - 48
            j = j + 1
            n = n + 1
        }
        t.k = j
        return r
    }
    if (c == 120 && j < t.n) {
        var x = rxCpAt(p, j)
        j = j + (x > 65535 ? 2 : 1)
        if (x == 123) {
            var nh = 0
            var v = 0
            while (true) {
                if (j >= t.n) { break }
                var h = rxCpAt(p, j)
                j = j + (h > 65535 ? 2 : 1)
                if (h == 125) {
                    if (nh == 0) { break }
                    t.k = j
                    return v
                }
                var hv = rxIsHexCode(h) ? rxHexVal(h) : 0 - 1
                if (hv < 0) { break }
                v = v * 16 + hv
                if (v > 1114111) { break }
                nh = nh + 1
            }
        } else {
            var y = -1
            if (j < t.n) {
                var yc = rxCpAt(p, j)
                j = j + (yc > 65535 ? 2 : 1)
                if (rxIsHexCode(yc)) { y = rxHexVal(yc) }
            }
            if (rxIsHexCode(x) && y >= 0) {
                t.k = j
                return rxHexVal(x) * 16 + y
            }
        }
    }
    var simple = "a\u0007f\u000cn\nr\rt\tv\u000b"
    var s = 0
    while (s < simple.length) {
        if (simple.charCodeAt(s) == c) {
            t.k = j
            return simple.charCodeAt(s + 1)
        }
        s = s + 2
    }
    rxRe2Fail(t, "invalid escape sequence", p.substring(at, j))
    return 0 - 1
}

// One character of a class at index at, with the index after it in t.k; -1
// after an error.
function rxRe2ClassChar(t, at, whole) {
    if (at >= t.n) {
        rxRe2Fail(t, "missing closing ]", t.p.substring(whole, t.n))
        return 0 - 1
    }
    if (t.p.charCodeAt(at) == 92) { return rxRe2Esc(t, at) }
    var c = rxCpAt(t.p, at)
    t.k = at + (c > 65535 ? 2 : 1)
    return c
}

// A bracketed class at t.i (regexp/syntax's parseClass).
function rxRe2Class(t) {
    var p = t.p
    var s = t.i
    var j = s + 1
    var neg = false
    if (j < t.n && p.charCodeAt(j) == 94) {
        neg = true
        j = j + 1
    }
    var cls = rxRe2Cls()
    var first = true
    while (t.err == "" && (j >= t.n || p.charCodeAt(j) != 93 || first)) {
        first = false
        if (t.n - j > 2 && p.charCodeAt(j) == 91 && p.charCodeAt(j + 1) == 58) {
            var e = p.indexOf(":]", j + 2)
            if (e >= 0) {
                var name = p.substring(j, e + 2)
                var pn = name.substring(2, name.length - 2)
                var pneg = pn.length > 0 && pn.charCodeAt(0) == 94
                if (pneg) { pn = pn.substring(1, pn.length) }
                var spec = pn.length > 1 ? rxRe2Spec(pn) : ""
                if (spec == "") {
                    rxRe2Fail(t, "invalid character class range", name)
                    return
                }
                rxRe2Group(cls, spec, pneg)
                j = e + 2
                continue
            }
        }
        if (j + 1 < t.n && p.charCodeAt(j) == 92) {
            var d = p.charCodeAt(j + 1)
            if (d == 112 || d == 80) {
                j = rxRe2Uni(t, j, cls)
                continue
            }
            if (rxRe2Perl(cls, d)) {
                j = j + 2
                continue
            }
        }
        var rs = j
        var lo = rxRe2ClassChar(t, j, s)
        if (lo < 0) { return }
        j = t.k
        var hi = lo
        if (t.n - j >= 2 && p.charCodeAt(j) == 45 && p.charCodeAt(j + 1) != 93) {
            hi = rxRe2ClassChar(t, j + 1, s)
            if (hi < 0) { return }
            j = t.k
            if (hi < lo) {
                rxRe2Fail(t, "invalid character class range", p.substring(rs, j))
                return
            }
        }
        rxRe2Range(cls, lo, hi)
    }
    if (t.err != "") { return }
    t.i = j + 1
    rxRe2PushCls(t, cls, neg)
}

// regexp/syntax's parseRepeat at t.i: {min}, {min,} or {min,max}, as
// {min, max (-1: none), end}; null when the text is not one, and then '{' is a
// literal. A number of more than eight digits reads as -1, which Go rejects.
function rxRe2Bound(t) {
    var p = t.p
    var j = t.i + 1
    var lo = rxRe2Int(t, j)
    if (lo == 0 - 2) { return null }
    j = t.k
    var hi = lo
    if (j >= t.n) { return null }
    if (p.charCodeAt(j) == 44) {
        j = j + 1
        if (j >= t.n) { return null }
        if (p.charCodeAt(j) == 125) { hi = 0 - 1 }
        else {
            hi = rxRe2Int(t, j)
            if (hi == 0 - 2) { return null }
            j = t.k
            if (hi == 0 - 1) { lo = 0 - 1 }
        }
    }
    if (j >= t.n || p.charCodeAt(j) != 125) { return null }
    return {min: lo, max: hi, end: j + 1}
}

// A decimal number at index at with no leading zero, the index after it in t.k:
// -2 when there is none, -1 when it has more than eight digits.
function rxRe2Int(t, at) {
    var p = t.p
    if (at >= t.n || !rxIsDigitCode(p.charCodeAt(at))) { return 0 - 2 }
    if (at + 1 < t.n && p.charCodeAt(at) == 48 && rxIsDigitCode(p.charCodeAt(at + 1))) { return 0 - 2 }
    var j = at
    var v = 0
    while (j < t.n && rxIsDigitCode(p.charCodeAt(j))) {
        if (v >= 0) { v = v >= 100000000 ? 0 - 1 : v * 10 + p.charCodeAt(j) - 48 }
        j = j + 1
    }
    t.k = j
    return v
}

// A repetition operator spelled from index op to t.i (regexp/syntax's repeat):
// a trailing ? makes it lazy, a repeat may not follow a repeat, and it applies
// to the atom before it.
function rxRe2Repeat(t, op, min, max, counted) {
    var p = t.p
    var q = p.substring(op, t.i)
    var lazy = false
    if (t.i < t.n && p.charCodeAt(t.i) == 63) {
        lazy = true
        t.i = t.i + 1
    }
    if (t.lastRep >= 0) {
        rxRe2Fail(t, "invalid nested repetition operator", p.substring(t.lastRep, t.i))
        return
    }
    var f = t.frames[t.depth]
    var cur = f.cur
    if (cur.length == 0) {
        rxRe2Fail(t, "missing argument to repetition operator", p.substring(op, t.i))
        return
    }
    var atom = cur.pop()
    var needs = f.needs
    var need = needs.pop()
    if (counted) {
        var m = max < 0 ? min : max
        need = m == 0 ? 0 : (need > 1 ? m * need : m)
        if ((min >= 2 || max >= 2) && need > 1000) {
            rxRe2Fail(t, "invalid repeat count", p.substring(op, t.i))
            return
        }
    }
    rxRe2Push(t, "(?:" + atom + ")" + q + (lazy == t.lazy ? "" : "?"), need)
}

// (?..: a named group, a flag change, or a non-capturing group with flags
// (regexp/syntax's parsePerlFlags). A flag change lasts to the end of the
// enclosing group.
function rxRe2Flags(t) {
    var p = t.p
    var i = t.i
    var named = t.n - i > 4 && p.charCodeAt(i + 2) == 80 && p.charCodeAt(i + 3) == 60
    var start = i + 4
    if (!named && t.n - i > 3 && p.charCodeAt(i + 2) == 60) {
        named = true
        start = i + 3
    }
    if (named) {
        var end = p.indexOf(">", i)
        if (end < 0) {
            rxRe2Fail(t, "invalid named capture", p.substring(i, t.n))
            return
        }
        var name = p.substring(start, end)
        var ok = name.length > 0
        var k = 0
        while (k < name.length) {
            if (!rxIsWordCode(name.charCodeAt(k))) { ok = false }
            k = k + 1
        }
        if (!ok) {
            rxRe2Fail(t, "invalid named capture", p.substring(i, end + 1))
            return
        }
        t.ncap = t.ncap + 1
        var names = t.names
        names.push(name)
        var groups = t.nameGroups
        groups.push(t.ncap)
        rxRe2Open(t, t.ncap)
        t.i = end + 1
        return
    }
    var j = i + 2
    var fold = t.fold
    var multi = t.multi
    var dotnl = t.dotnl
    var ung = t.lazy
    var neg = false
    var saw = false
    while (j < t.n) {
        var c = rxCpAt(p, j)
        j = j + (c > 65535 ? 2 : 1)
        if (c == 105) { fold = !neg }
        else if (c == 109) { multi = !neg }
        else if (c == 115) { dotnl = !neg }
        else if (c == 85) { ung = !neg }
        else if (c == 45) {
            if (neg) { break }
            neg = true
            saw = false
            continue
        } else if (c == 58 || c == 41) {
            if (neg && !saw) { break }
            if (c == 58) { rxRe2Open(t, 0) }
            t.fold = fold
            t.multi = multi
            t.dotnl = dotnl
            t.lazy = ung
            t.i = j
            return
        } else { break }
        saw = true
    }
    rxRe2Fail(t, "invalid or unsupported Perl syntax", p.substring(i, j))
}

// A backslash at t.i outside a class.
function rxRe2Escape(t) {
    var p = t.p
    var i = t.i
    if (i + 1 < t.n) {
        var d = p.charCodeAt(i + 1)
        var as = d == 65 ? "\\A" : d == 98 ? "\\b" : d == 66 ? "\\B" : d == 122 ? "\\z" : ""
        if (as != "") {
            rxRe2Push(t, as, 0)
            t.i = i + 2
            return
        }
        if (d == 67) {
            rxRe2Fail(t, "invalid escape sequence", "\\C")
            return
        }
        if (d == 81) {
            var e = p.indexOf("\\E", i + 2)
            var stop = e < 0 ? t.n : e
            var j = i + 2
            while (j < stop) {
                var c = rxCpAt(p, j)
                rxRe2Lit(t, c)
                j = j + (c > 65535 ? 2 : 1)
            }
            t.i = e < 0 ? t.n : e + 2
            return
        }
        var cls = rxRe2Cls()
        if (d == 112 || d == 80) {
            t.i = rxRe2Uni(t, i, cls)
            if (t.err == "") { rxRe2PushCls(t, cls, false) }
            return
        }
        if (rxRe2Perl(cls, d)) {
            t.i = i + 2
            rxRe2PushCls(t, cls, false)
            return
        }
    }
    var r = rxRe2Esc(t, i)
    if (r < 0) { return }
    t.i = t.k
    rxRe2Lit(t, r)
}

function rxRe2(pattern) {
    var t = {p: pattern, i: 0, k: 0, n: pattern.length, err: "", fold: false, multi: false, dotnl: false,
             lazy: false, frames: [], depth: 0 - 1, ncap: 0, names: [], nameGroups: [], lastRep: 0 - 1}
    rxRe2Open(t, 0)
    var p = pattern
    while (t.i < t.n && t.err == "") {
        var rep = 0 - 1
        var c = p.charCodeAt(t.i)
        if (c == 40) {
            if (t.i + 1 < t.n && p.charCodeAt(t.i + 1) == 63) { rxRe2Flags(t) }
            else {
                t.ncap = t.ncap + 1
                rxRe2Open(t, t.ncap)
                t.i = t.i + 1
            }
        } else if (c == 124) {
            rxRe2Bar(t)
            t.i = t.i + 1
        } else if (c == 41) {
            rxRe2Close(t)
        } else if (c == 94 || c == 36) {
            var anc = c == 94 ? "\\A" : "\\z"
            if (t.multi) { anc = c == 94 ? "(?:\\A|(?<=\\x{A}))" : "(?:\\z|(?=\\x{A}))" }
            rxRe2Push(t, anc, 0)
            t.i = t.i + 1
        } else if (c == 46) {
            rxRe2Push(t, t.dotnl ? "[\\x{0}-\\x{10FFFF}]" : "[^\\x{A}]", 0)
            t.i = t.i + 1
        } else if (c == 91) {
            rxRe2Class(t)
        } else if (c == 42 || c == 43 || c == 63) {
            rep = t.i
            t.i = t.i + 1
            rxRe2Repeat(t, rep, c == 43 ? 1 : 0, c == 63 ? 1 : 0 - 1, false)
        } else if (c == 123) {
            var b = rxRe2Bound(t)
            if (b == null) {
                rxRe2Lit(t, 123)
                t.i = t.i + 1
            } else if (b.min < 0 || b.min > 1000 || b.max > 1000 || (b.max >= 0 && b.min > b.max)) {
                rxRe2Fail(t, "invalid repeat count", p.substring(t.i, b.end))
            } else {
                rep = t.i
                t.i = b.end
                rxRe2Repeat(t, rep, b.min, b.max, true)
            }
        } else if (c == 92) {
            rxRe2Escape(t)
        } else {
            var cp = rxCpAt(p, t.i)
            if (cp >= 55296 && cp <= 57343) {
                rxRe2Fail(t, "invalid UTF-8", p.substring(t.i, t.n))
            }
            rxRe2Lit(t, cp)
            t.i = t.i + (cp > 65535 ? 2 : 1)
        }
        t.lastRep = rep
    }
    if (t.err == "" && t.depth != 0) { rxRe2Fail(t, "missing closing )", p) }
    return {out: t.err == "" ? rxRe2Body(t.frames[0]) : "", err: t.err, names: t.names, nameGroups: t.nameGroups}
}

// ===== The C dialect: PCRE2, as PHP's preg_* compile it =====
//
// rxPcre reads a PCRE2 pattern and writes its J spelling, which rxCompile then
// compiles with java.util.regex's machinery and UNIX_LINES (PHP's PCRE2 is
// built with NEWLINE_DEFAULT LF). mods are PHP's pattern modifiers: i m s x n
// U D A u J. Without u the pattern and the subject are BYTES (each unit one
// byte) and \d \w \s \b are ASCII; with u (PCRE2_UTF | PCRE2_UCP) they are code
// points and the classes Unicode's. What the J spelling cannot say is spelled
// out: every literal as \x{..}, a capture always a plain "(" (the names stay in
// t.names), a backreference as (?:\n), ^ $ . per the m s D in force, U by
// swapping each quantifier's greed, \K as J's \K under C. Recursion, subroutine
// calls, conditionals, branch reset and (*VERB)s are not implemented: their
// re.err starts with "unsupported".

// The J items of PCRE2's UCP \w (L, N, Mn, Pc) and \s (Z, \h, \v).
var RX_PCRE_UW = "\\p{L}\\p{N}_\\p{Mn}\\p{Pc}"
var RX_PCRE_US = "\\p{Z}\\h\\v"

function rxPcreFail(t, msg, at) {
    if (t.err == "") {
        t.err = msg
        t.errAt = at
    }
}

// Appends J text; okq: may a quantifier follow it?
function rxPcreEmit(t, s, okq) {
    t.out = t.out + s
    t.okq = okq
}

// A literal character: an ASCII letter or digit as itself, anything else as
// \x{..}; one atom either way.
function rxPcreLitText(c) {
    if (rxIsWordCode(c) && c != 95) { return rxCpStr(c) }
    return rxRe2Hex(c)
}

// The frame of flags in force; one per open group, restored at its ")".
function rxPcreFrame(t) { return {i: t.fi, m: t.fm, s: t.fs, x: t.fx, U: t.fU, n: t.fn, J: t.fJ} }

function rxPcreOpen(t) {
    var fr = t.frames
    fr.push(rxPcreFrame(t))
}

function rxPcreClose(t) {
    var fr = t.frames
    if (fr.length == 0) {
        rxPcreFail(t, "unmatched closing parenthesis", t.i)
        return
    }
    var f = fr.pop()
    t.fi = f.i
    t.fm = f.m
    t.fs = f.s
    t.fx = f.x
    t.fU = f.U
    t.fn = f.n
    t.fJ = f.J
    t.i = t.i + 1
    rxPcreEmit(t, ")", true)
}

// x-mode white space: what isspace() takes in the C locale, and under u
// Unicode's Pattern_White_Space beyond it (U+0085, U+200E, U+200F, U+2028,
// U+2029).
function rxPcreIsXSpace(t, c) {
    if (c == 32 || (c >= 9 && c <= 13)) { return true }
    return t.utf && (c == 133 || c == 8206 || c == 8207 || c == 8232 || c == 8233)
}

// Steps over what changes nothing: x-mode white space and # comments, (?#..)
// comments and \E, and \Q\E pairs.
function rxPcreSkip(t) {
    var p = t.p
    while (t.i < t.n) {
        var c = p.charCodeAt(t.i)
        if (c == 92 && t.i + 1 < t.n && p.charCodeAt(t.i + 1) == 69) {
            t.i = t.i + 2
        } else if (c == 92 && t.i + 3 < t.n && p.charCodeAt(t.i + 1) == 81 && p.charCodeAt(t.i + 2) == 92
                   && p.charCodeAt(t.i + 3) == 69) {
            t.i = t.i + 4
        } else if (t.fx && rxPcreIsXSpace(t, c)) {
            t.i = t.i + 1
        } else if (t.fx && c == 35) {
            while (t.i < t.n && p.charCodeAt(t.i) != 10) { t.i = t.i + 1 }
        } else if (c == 40 && t.i + 2 < t.n && p.charCodeAt(t.i + 1) == 63 && p.charCodeAt(t.i + 2) == 35) {
            var e = t.i + 3
            while (e < t.n && p.charCodeAt(e) != 41) { e = e + 1 }
            if (e >= t.n) {
                rxPcreFail(t, "missing ) at end of (?# comment", t.n)
                return
            }
            t.i = e + 1
        } else {
            return
        }
    }
}

// read_repeat_counts at the "{" at t.i: {n} {n,} {n,m} {,m}, spaces and tabs
// allowed inside; null when it is not a quantifier (the { is then a literal).
function rxPcreBound(t) {
    var p = t.p
    var j = t.i + 1
    var minV = 0 - 1
    var maxV = 0 - 1
    while (j < t.n && (p.charCodeAt(j) == 32 || p.charCodeAt(j) == 9)) { j = j + 1 }
    if (j < t.n && rxIsDigitCode(p.charCodeAt(j))) {
        minV = 0
        while (j < t.n && rxIsDigitCode(p.charCodeAt(j))) {
            minV = minV * 10 + p.charCodeAt(j) - 48
            if (minV > 99999999) { minV = 99999999 }
            j = j + 1
        }
    }
    while (j < t.n && (p.charCodeAt(j) == 32 || p.charCodeAt(j) == 9)) { j = j + 1 }
    if (j >= t.n) { return null }
    if (p.charCodeAt(j) == 125) {
        if (minV < 0) { return null }
        return {min: minV, max: minV, end: j + 1}
    }
    if (p.charCodeAt(j) != 44) { return null }
    j = j + 1
    while (j < t.n && (p.charCodeAt(j) == 32 || p.charCodeAt(j) == 9)) { j = j + 1 }
    var hasMax = false
    if (j < t.n && rxIsDigitCode(p.charCodeAt(j))) {
        hasMax = true
        maxV = 0
        while (j < t.n && rxIsDigitCode(p.charCodeAt(j))) {
            maxV = maxV * 10 + p.charCodeAt(j) - 48
            if (maxV > 99999999) { maxV = 99999999 }
            j = j + 1
        }
    }
    if (minV < 0 && !hasMax) { return null }
    while (j < t.n && (p.charCodeAt(j) == 32 || p.charCodeAt(j) == 9)) { j = j + 1 }
    if (j >= t.n || p.charCodeAt(j) != 125) { return null }
    if (minV < 0) { minV = 0 }
    return {min: minV, max: maxV, end: j + 1}
}

// A quantifier at t.i: its text with greed decided (U swaps it, a following ?
// swaps it back, a following + makes it possessive).
function rxPcreQuant(t) {
    var p = t.p
    var c = p.charCodeAt(t.i)
    var q = ""
    var at = t.i
    if (c == 42) { q = "*"; t.i = t.i + 1 }
    else if (c == 43) { q = "+"; t.i = t.i + 1 }
    else if (c == 63) { q = "?"; t.i = t.i + 1 }
    else {
        var b = rxPcreBound(t)
        if (b.min > 65535 || b.max > 65535) {
            rxPcreFail(t, "number too big in {} quantifier", b.end - 1)
            return
        }
        if (b.max >= 0 && b.max < b.min) {
            rxPcreFail(t, "numbers out of order in {} quantifier", b.end - 1)
            return
        }
        q = "{" + b.min + (b.max == b.min ? "" : b.max < 0 ? "," : "," + b.max) + "}"
        t.i = b.end
    }
    if (!t.okq) {
        rxPcreFail(t, "quantifier does not follow a repeatable item", at)
        return
    }
    rxPcreSkip(t)
    var lazy = t.fU
    var poss = false
    if (t.i < t.n && p.charCodeAt(t.i) == 63) {
        lazy = !lazy
        t.i = t.i + 1
    } else if (t.i < t.n && p.charCodeAt(t.i) == 43) {
        poss = true
        lazy = false
        t.i = t.i + 1
    }
    rxPcreEmit(t, q + (lazy ? "?" : poss ? "+" : ""), false)
}

// A group name after (?< (?' (?P< or \k< \k' \k{ \g{, up to term: a word
// character that is not a digit, then word characters, at most 32.
function rxPcreName(t, term) {
    var p = t.p
    var s = t.i
    while (t.i < t.n && t.i - s <= 32) {
        var c = p.charCodeAt(t.i)
        if (!(rxIsWordCode(c) || (t.utf && c > 127 && rxPropHit(RX_PLD, rxCpAt(p, t.i))))) { break }
        t.i = t.i + (c >= 55296 && c <= 56319 ? 2 : 1)
    }
    if (t.i == s) {
        rxPcreFail(t, "subpattern name expected", t.i)
        return ""
    }
    if (rxIsDigitCode(p.charCodeAt(s))) {
        rxPcreFail(t, "subpattern name must start with a non-digit", s)
        return ""
    }
    if (t.i - s > 32) {
        rxPcreFail(t, "subpattern name is too long (maximum 32 code units)", t.i)
        return ""
    }
    if (t.i >= t.n || p.charCodeAt(t.i) != term) {
        rxPcreFail(t, "syntax error in subpattern name (missing terminator?)", t.i)
        return ""
    }
    var nm = p.substring(s, t.i)
    t.i = t.i + 1
    return nm
}

// A backreference: group n now, or the group a name stands for, resolved when
// the whole pattern is read (a forward reference is legal).
function rxPcreRef(t, n, name, at) {
    var parts = t.parts
    parts.push(t.out)
    var refs = t.refs
    refs.push({part: parts.length, n: n, name: name, at: at})
    parts.push("")
    t.out = ""
    rxPcreEmit(t, "", true)
}

// The digits at t.i as a number (capped).
function rxPcreNum(t) {
    var p = t.p
    var v = 0
    var j = t.i
    while (j < t.n && rxIsDigitCode(p.charCodeAt(j))) {
        v = v * 10 + p.charCodeAt(j) - 48
        if (v > 99999999) { v = 99999999 }
        j = j + 1
    }
    t.i = j
    return v
}

// The class items of the shorthand \d \D \s \S \w \W (c its letter) into cls:
// ASCII ones, or under u the UCP ones; a complement under u is a nested class.
function rxPcreShort(t, cls, c) {
    var lc = c >= 65 && c <= 90 ? c + 32 : c
    if (!t.utf) {
        rxRe2Group(cls, rxRe2Spec(String.fromCharCode(lc)), lc != c)
        return
    }
    var items = lc == 100 ? "\\p{Nd}" : lc == 119 ? RX_PCRE_UW : RX_PCRE_US
    cls.props = cls.props + (lc == c ? items : "[^" + items + "]")
}

// The \p / \P at t.i (after the letter), into cls: PCRE2's general categories,
// scripts and its X properties; "" and an error for an unknown name.
function rxPcreProp(t, cls, neg) {
    var p = t.p
    var name = ""
    if (t.i < t.n && p.charCodeAt(t.i) == 123) {
        var e = t.i + 1
        while (e < t.n && p.charCodeAt(e) != 125) { e = e + 1 }
        if (e >= t.n) {
            rxPcreFail(t, "malformed \\P or \\p sequence", e)
            return
        }
        name = p.substring(t.i + 1, e)
        t.i = e + 1
    } else if (t.i < t.n) {
        name = p.substring(t.i, t.i + 1)
        t.i = t.i + 1
    } else {
        rxPcreFail(t, "malformed \\P or \\p sequence", t.i)
        return
    }
    if (name.length > 0 && name.charCodeAt(0) == 94) {
        neg = !neg
        name = name.substring(1, name.length)
    }
    var key = rxRe2Canon(name)
    var items = ""
    if (name == "L&" || key == "Lc" || key == "L&") { items = "\\p{LC}" }
    else if (key == "Xan") { items = "\\p{L}\\p{N}" }
    else if (key == "Xwd") { items = RX_PCRE_UW }
    else if (key == "Xsp" || key == "Xps") { items = RX_PCRE_US }
    else if (key == "Xuc") { items = "$@`\\x{A0}-\\x{D7FF}\\x{E000}-\\x{10FFFF}" }
    else {
        items = rxRe2Prop(name, 1)
        if (key == "Any") { items = "\\x{0}-\\x{10FFFF}" }
    }
    if (items == "") {
        rxPcreFail(t, "unknown property after \\P or \\p", t.i)
        return
    }
    cls.props = cls.props + (neg ? "[^" + items + "]" : items)
}

// check_escape for a character: the code point an escape at t.i (after the
// backslash) stands for, or -1 when it is not a character escape (t.i then
// unmoved). inClass: \b is a backspace and a digit always octal.
function rxPcreChar(t, inClass) {
    var p = t.p
    var c = rxCpAt(p, t.i)
    var w = c > 65535 ? 2 : 1
    if (!(rxIsWordCode(c) && c != 95)) {
        t.i = t.i + w
        return c
    }
    var simple = "a\u0007e\u001bf\u000cn\nr\rt\t"
    var s = 0
    while (s < simple.length) {
        if (simple.charCodeAt(s) == c) {
            t.i = t.i + 1
            return simple.charCodeAt(s + 1)
        }
        s = s + 2
    }
    if (c == 98 && inClass) {
        t.i = t.i + 1
        return 8
    }
    if (c == 48 || (inClass && c >= 49 && c <= 55)) { // \0oo, and \ooo in a class
        var v = c - 48
        var k = 1
        t.i = t.i + 1
        while (k < 3 && t.i < t.n && p.charCodeAt(t.i) >= 48 && p.charCodeAt(t.i) <= 55) {
            v = v * 8 + p.charCodeAt(t.i) - 48
            t.i = t.i + 1
            k = k + 1
        }
        if (!t.utf && v > 255) { rxPcreFail(t, "octal value is greater than \\377 in 8-bit non-UTF-8 mode", t.i) }
        return v
    }
    if (inClass && (c == 56 || c == 57)) {
        t.i = t.i + 1
        return c
    }
    if (c == 111) { // \o{ddd}
        if (t.i + 1 >= t.n || p.charCodeAt(t.i + 1) != 123) {
            rxPcreFail(t, "missing opening brace after \\o", t.i + 1)
            return 0
        }
        return rxPcreBraced(t, 8)
    }
    if (c == 120) { // \xhh, \x{hhh}
        if (t.i + 1 < t.n && p.charCodeAt(t.i + 1) == 123) { return rxPcreBraced(t, 16) }
        t.i = t.i + 1
        var xv = 0
        var xk = 0
        while (xk < 2 && t.i < t.n && rxIsHexCode(p.charCodeAt(t.i))) {
            xv = xv * 16 + rxHexVal(p.charCodeAt(t.i))
            t.i = t.i + 1
            xk = xk + 1
        }
        return xv
    }
    if (c == 99) { // \cX
        if (t.i + 1 >= t.n) {
            rxPcreFail(t, "\\c at end of pattern", t.i + 1)
            return 0
        }
        var cc = p.charCodeAt(t.i + 1)
        if (cc >= 97 && cc <= 122) { cc = cc - 32 }
        if (cc < 32 || cc > 126) {
            rxPcreFail(t, "\\c must be followed by a printable ASCII character", t.i + 1)
            return 0
        }
        t.i = t.i + 2
        return cc >= 64 ? cc - 64 : cc + 64
    }
    if (c == 78 && t.i + 3 < t.n && p.charCodeAt(t.i + 1) == 123 && p.charCodeAt(t.i + 2) == 85
        && p.charCodeAt(t.i + 3) == 43) { // \N{U+hhh}
        if (!t.utf) {
            rxPcreFail(t, "\\N{U+dddd} is supported only in Unicode (UTF) mode", t.i + 3)
            return 0
        }
        t.i = t.i + 2
        return rxPcreBraced(t, 16)
    }
    return 0 - 1
}

// The {..} digits of \x{..} \o{..} \N{U+..} in base; t.i is at the character
// before the {.
function rxPcreBraced(t, base) {
    var p = t.p
    t.i = t.i + 2
    while (t.i < t.n && (p.charCodeAt(t.i) == 32 || p.charCodeAt(t.i) == 9)) { t.i = t.i + 1 }
    if (t.i >= t.n || p.charCodeAt(t.i) == 125) {
        rxPcreFail(t, "digits missing in \\x{} or \\o{} or \\N{U+}", t.i)
        return 0
    }
    var v = 0
    var lim = t.utf ? 1114111 : 255
    while (t.i < t.n) {
        var d = base == 16 ? (rxIsHexCode(p.charCodeAt(t.i)) ? rxHexVal(p.charCodeAt(t.i)) : 0 - 1)
                           : (p.charCodeAt(t.i) >= 48 && p.charCodeAt(t.i) <= 55 ? p.charCodeAt(t.i) - 48 : 0 - 1)
        if (d < 0) { break }
        v = v * base + d
        if (v > lim) {
            rxPcreFail(t, "character code point value in \\x{} or \\o{} is too large", t.i)
            return 0
        }
        t.i = t.i + 1
    }
    while (t.i < t.n && (p.charCodeAt(t.i) == 32 || p.charCodeAt(t.i) == 9)) { t.i = t.i + 1 }
    if (t.i >= t.n || p.charCodeAt(t.i) != 125) {
        rxPcreFail(t, base == 16 ? "\\x{ is not followed by } (or the hex digits are invalid)"
                                 : "non-octal character in \\o{} (closing brace missing?)", t.i)
        return 0
    }
    t.i = t.i + 1
    if (t.utf && v >= 55296 && v <= 57343) {
        rxPcreFail(t, "disallowed Unicode code point (>= 0xd800 && <= 0xdfff)", t.i - 1)
    }
    return v
}

// A POSIX class [:name:] / [:^name:] at t.i (the "[") into cls; false when the
// text is not one (the [ is then a literal).
function rxPcrePosix(t, cls) {
    var p = t.p
    if (t.i + 2 >= t.n || p.charCodeAt(t.i + 1) != 58) { return false }
    var j = t.i + 2
    var e = 0 - 1
    while (j + 1 < t.n) {
        var c = p.charCodeAt(j)
        if (c == 92 && (p.charCodeAt(j + 1) == 93 || p.charCodeAt(j + 1) == 92)) { j = j + 1 }
        else if ((c == 91 && p.charCodeAt(j + 1) == 58) || c == 93) { return false }
        else if (c == 58 && p.charCodeAt(j + 1) == 93) {
            e = j
            j = t.n
        }
        j = j + 1
    }
    if (e < 0) { return false }
    var name = p.substring(t.i + 2, e)
    var neg = name.length > 0 && name.charCodeAt(0) == 94
    if (neg) { name = name.substring(1, name.length) }
    var spec = rxRe2Spec(name)
    if (name == "d" || name == "s" || name == "w" || spec == "") {
        rxPcreFail(t, "unknown POSIX class name", t.i + 2)
        return true
    }
    t.i = e + 2
    if (!t.utf || name == "ascii") {
        rxRe2Group(cls, spec, neg)
        return true
    }
    var u = ""
    if (name == "alpha") { u = "\\p{L}" }
    else if (name == "lower") { u = "\\p{Ll}" }
    else if (name == "upper") { u = "\\p{Lu}" }
    else if (name == "alnum") { u = "\\p{L}\\p{N}" }
    else if (name == "blank") { u = "\\h" }
    else if (name == "cntrl") { u = "\\p{Cc}" }
    else if (name == "digit") { u = "\\p{Nd}" }
    else if (name == "space") { u = RX_PCRE_US }
    else if (name == "word") { u = RX_PCRE_UW }
    else if (name == "xdigit") { u = "0-9A-Fa-f\\x{FF10}-\\x{FF19}\\x{FF21}-\\x{FF26}\\x{FF41}-\\x{FF46}" }
    else if (name == "punct") { u = "\\p{P}$+<=>\\^`|~" } // P, and the ASCII S characters
    else {
        // graph: L M N P S Cf less U+061C U+180E U+2066..U+2069; print adds
        // Zs and gives U+180E back (PT_PXGRAPH / PT_PXPRINT in pcre2_xclass.c).
        u = name == "print" ? "[\\p{L}\\p{M}\\p{N}\\p{P}\\p{S}\\p{Cf}\\p{Zs}&&[^\\x{61C}\\x{2066}-\\x{2069}]]"
                            : "[\\p{L}\\p{M}\\p{N}\\p{P}\\p{S}\\p{Cf}&&[^\\x{61C}\\x{180E}\\x{2066}-\\x{2069}]]"
    }
    cls.props = cls.props + (neg ? "[^" + u + "]" : u)
    return true
}

// A class at t.i (the "["): PCRE2's syntax, where [ and & are literals, a ]
// first is one, and POSIX classes stand inside.
function rxPcreClass(t) {
    var p = t.p
    t.i = t.i + 1
    var neg = false
    if (t.i < t.n && p.charCodeAt(t.i) == 94) {
        neg = true
        t.i = t.i + 1
    }
    var cls = rxRe2Cls()
    var first = true
    var inq = false
    var pend = 0 - 1  // a single character that may still start a range
    var dash = false  // a "-" after pend
    while (t.err == "") {
        if (t.i >= t.n) {
            rxPcreFail(t, "missing terminating ] for character class", t.n)
            return
        }
        var c = p.charCodeAt(t.i)
        if (inq) {
            if (c == 92 && t.i + 1 < t.n && p.charCodeAt(t.i + 1) == 69) {
                inq = false
                t.i = t.i + 2
                continue
            }
            c = rxCpAt(p, t.i)
            t.i = t.i + (c > 65535 ? 2 : 1)
        } else if (c == 93 && !first) {
            t.i = t.i + 1
            break
        } else if (c == 91 && rxPcrePosix(t, cls)) {
            if (dash) {
                rxPcreFail(t, "invalid range in character class", t.i)
                return
            }
            if (pend >= 0) { rxRe2Range(cls, pend, pend) }
            pend = 0 - 1
            first = false
            if (t.i + 1 < t.n && p.charCodeAt(t.i) == 45 && p.charCodeAt(t.i + 1) != 93) {
                rxPcreFail(t, "invalid range in character class", t.i)
                return
            }
            continue
        } else if (c == 45 && pend >= 0 && !dash) {
            dash = true
            t.i = t.i + 1
            first = false
            continue
        } else if (c == 92) {
            t.i = t.i + 1
            if (t.i >= t.n) {
                rxPcreFail(t, "\\ at end of pattern", t.i)
                return
            }
            var e = p.charCodeAt(t.i)
            if (e == 81 || e == 69) {
                inq = e == 81
                t.i = t.i + 1
                first = false
                continue
            }
            if (e == 100 || e == 68 || e == 115 || e == 83 || e == 119 || e == 87 || e == 104 || e == 72
                || e == 118 || e == 86 || e == 112 || e == 80) {
                if (dash) {
                    rxPcreFail(t, "invalid range in character class", t.i + 1)
                    return
                }
                if (pend >= 0) { rxRe2Range(cls, pend, pend) }
                pend = 0 - 1
                first = false
                t.i = t.i + 1
                if (e == 112 || e == 80) { rxPcreProp(t, cls, e == 80) }
                else if (e == 104 || e == 72 || e == 118 || e == 86) {
                    cls.props = cls.props + "\\" + String.fromCharCode(e)
                } else { rxPcreShort(t, cls, e) }
                if (t.i + 1 < t.n && p.charCodeAt(t.i) == 45 && p.charCodeAt(t.i + 1) != 93) {
                    rxPcreFail(t, "invalid range in character class", t.i)
                    return
                }
                continue
            }
            if (e == 66 || e == 82 || e == 88) {
                rxPcreFail(t, "escape sequence is invalid in character class", t.i)
                return
            }
            if (e == 78) {
                rxPcreFail(t, "\\N is not supported in a class", t.i)
                return
            }
            c = rxPcreChar(t, true)
            if (c < 0) {
                rxPcreFail(t, "unrecognized character follows \\", t.i)
                return
            }
        } else {
            c = rxCpAt(p, t.i)
            t.i = t.i + (c > 65535 ? 2 : 1)
        }
        first = false
        if (dash) {
            if (c < pend) {
                rxPcreFail(t, "range out of order in character class", t.i - 1)
                return
            }
            rxRe2Range(cls, pend, c)
            pend = 0 - 1
            dash = false
        } else {
            if (pend >= 0) { rxRe2Range(cls, pend, pend) }
            pend = c
        }
    }
    if (t.err != "") { return }
    if (pend >= 0) { rxRe2Range(cls, pend, pend) }
    if (dash) { rxRe2Range(cls, 45, 45) }
    rxPcreEmit(t, rxRe2ClsText(cls, neg), true)
}

// The flags of (?imnsxJU-imnsxJU) / (?^..) at t.i (after "(?"), ending in ")"
// (they hold to the end of the enclosing group) or ":" (a group of their own).
// Only i reaches the J text.
function rxPcreFlags(t) {
    var p = t.p
    var on = true
    var hyph = true
    var saved = rxPcreFrame(t)
    var i0 = t.fi
    if (t.i < t.n && p.charCodeAt(t.i) == 94) {
        t.fi = false
        t.fm = false
        t.fn = false
        t.fs = false
        t.fx = false
        hyph = false
        t.i = t.i + 1
    }
    while (t.i < t.n && p.charCodeAt(t.i) != 41 && p.charCodeAt(t.i) != 58) {
        var c = p.charAt(t.i)
        if (c == "-" && hyph) {
            on = false
            hyph = false
        } else if (c == "i") { t.fi = on }
        else if (c == "m") { t.fm = on }
        else if (c == "n") { t.fn = on }
        else if (c == "s") { t.fs = on }
        else if (c == "x") { t.fx = on }
        else if (c == "J") { t.fJ = on }
        else if (c == "U") { t.fU = on }
        else {
            rxPcreFail(t, c == "-" ? "invalid hyphen in option setting" : "unrecognized character after (? or (?-", t.i)
            return
        }
        t.i = t.i + 1
    }
    if (t.i >= t.n) {
        rxPcreFail(t, "missing closing parenthesis", t.n)
        return
    }
    var ifl = t.fi == i0 ? "" : t.fi ? "i" : "-i"
    if (p.charCodeAt(t.i) == 41) {
        t.i = t.i + 1
        rxPcreEmit(t, ifl == "" ? "" : "(?" + ifl + ")", false)
        return
    }
    t.i = t.i + 1
    // The group's own frame keeps what was in force before the letters.
    var nf = t.frames
    nf.push(saved)
    rxPcreEmit(t, "(?" + ifl + ":", false)
}

// A named group's name, checked against the names so far, and its number.
function rxPcreNamed(t, term) {
    var nm = rxPcreName(t, term)
    if (t.err != "") { return }
    if (!t.fJ && !t.dup && rxNameIdx(t.names, t.nameGroups, nm) >= 0) {
        rxPcreFail(t, "two named subpatterns have the same name (PCRE2_DUPNAMES not set)", t.i - 1)
        return
    }
    t.ncap = t.ncap + 1
    var names = t.names
    names.push(nm)
    var ng = t.nameGroups
    ng.push(t.ncap)
    rxPcreOpen(t)
    rxPcreEmit(t, "(", false)
}

// "(" at t.i.
function rxPcreParen(t) {
    var p = t.p
    var at = t.i
    t.i = t.i + 1
    if (t.i < t.n && p.charCodeAt(t.i) == 42) {
        rxPcreFail(t, "unsupported (*VERB) or alpha assertion", at)
        return
    }
    if (t.i >= t.n || p.charCodeAt(t.i) != 63) {
        if (t.fn) {
            rxPcreOpen(t)
            rxPcreEmit(t, "(?:", false)
            return
        }
        t.ncap = t.ncap + 1
        rxPcreOpen(t)
        rxPcreEmit(t, "(", false)
        return
    }
    t.i = t.i + 1
    if (t.i >= t.n) {
        rxPcreFail(t, "missing closing parenthesis", t.n)
        return
    }
    var c = p.charCodeAt(t.i)
    var c2 = t.i + 1 < t.n ? p.charCodeAt(t.i + 1) : 0
    if (c == 58 || c == 62 || c == 61 || c == 33) { // (?: (?> (?= (?!
        t.i = t.i + 1
        rxPcreOpen(t)
        rxPcreEmit(t, "(?" + String.fromCharCode(c), false)
        return
    }
    if (c == 60 && (c2 == 61 || c2 == 33)) { // (?<= (?<!
        t.i = t.i + 2
        rxPcreOpen(t)
        rxPcreEmit(t, "(?<" + String.fromCharCode(c2), false)
        return
    }
    if (c == 60 || c == 39) { // (?<name> (?'name'
        t.i = t.i + 1
        rxPcreNamed(t, c == 60 ? 62 : 39)
        return
    }
    if (c == 80) { // (?P<name> (?P=name) (?P>name)
        if (c2 == 60) {
            t.i = t.i + 2
            rxPcreNamed(t, 62)
            return
        }
        if (c2 == 61) {
            t.i = t.i + 2
            var rn = rxPcreName(t, 41)
            if (t.err == "") { rxPcreRef(t, 0, rn, at) }
            return
        }
        if (c2 == 62) {
            rxPcreFail(t, "unsupported subroutine call", at)
            return
        }
        rxPcreFail(t, "unrecognized character after (?P", t.i + 1)
        return
    }
    if (c == 124 || c == 40 || c == 82 || c == 38 || c == 67 || c == 43 || (c == 45 && rxIsDigitCode(c2))
        || rxIsDigitCode(c)) {
        rxPcreFail(t, "unsupported branch reset, conditional, recursion or callout group", at)
        return
    }
    rxPcreFlags(t)
}

// An escape at t.i (the backslash), outside a class.
function rxPcreEscape(t) {
    var p = t.p
    var at = t.i
    t.i = t.i + 1
    if (t.i >= t.n) {
        rxPcreFail(t, "\\ at end of pattern", t.i)
        return
    }
    var c = p.charCodeAt(t.i)
    if (c == 100 || c == 68 || c == 115 || c == 83 || c == 119 || c == 87) {
        t.i = t.i + 1
        if (!t.utf) {
            rxPcreEmit(t, "\\" + String.fromCharCode(c), true)
            return
        }
        var sc = rxRe2Cls()
        rxPcreShort(t, sc, c)
        rxPcreEmit(t, rxRe2ClsText(sc, false), true)
        return
    }
    if (c == 104 || c == 72 || c == 118 || c == 86 || c == 88) { // \h \H \v \V \X
        t.i = t.i + 1
        rxPcreEmit(t, "\\" + String.fromCharCode(c), true)
        return
    }
    if (c == 112 || c == 80) {
        t.i = t.i + 1
        var pc = rxRe2Cls()
        rxPcreProp(t, pc, c == 80)
        rxPcreEmit(t, rxRe2ClsText(pc, false), true)
        return
    }
    if (c == 82) {
        t.i = t.i + 1
        rxPcreEmit(t, "(?>\\x{D}\\x{A}|[\\x{A}-\\x{D}\\x{85}\\x{2028}\\x{2029}])", true)
        return
    }
    if (c == 78 && !(t.i + 3 < t.n && p.charCodeAt(t.i + 1) == 123 && p.charCodeAt(t.i + 2) == 85
                     && p.charCodeAt(t.i + 3) == 43)) { // \N, which a {..} may quantify
        t.i = t.i + 1
        rxPcreEmit(t, "[^\\x{A}]", true)
        return
    }
    if (c == 67) { // \C
        if (t.utf) {
            rxPcreFail(t, "using \\C is incompatible with the 'u' modifier", t.i + 1)
            return
        }
        t.i = t.i + 1
        rxPcreEmit(t, "[\\x{0}-\\x{FF}]", true)
        return
    }
    if (c == 65 || c == 122 || c == 90 || c == 71 || c == 75) { // \A \z \Z \G \K
        t.i = t.i + 1
        rxPcreEmit(t, "\\" + String.fromCharCode(c), false)
        return
    }
    if (c == 98 || c == 66) { // \b \B
        t.i = t.i + 1
        if (!t.utf) { rxPcreEmit(t, "\\" + String.fromCharCode(c), false) }
        else {
            var w = "[" + RX_PCRE_UW + "]"
            rxPcreEmit(t, c == 98 ? "(?:(?<=" + w + ")(?!" + w + ")|(?<!" + w + ")(?=" + w + "))"
                                  : "(?:(?<=" + w + ")(?=" + w + ")|(?<!" + w + ")(?!" + w + "))", false)
        }
        return
    }
    if (c >= 49 && c <= 57) { // \1.. a backreference, or octal
        var s = t.i
        var n = rxPcreNum(t)
        if (n < 10 || c >= 56 || n <= t.ncap) {
            rxPcreRef(t, n, "", at)
            return
        }
        t.i = s
        if (c >= 56) {
            t.i = t.i + 1
            rxPcreEmit(t, rxPcreLitText(c), true)
            return
        }
        var ov = 0
        var ok = 0
        while (ok < 3 && t.i < t.n && p.charCodeAt(t.i) >= 48 && p.charCodeAt(t.i) <= 55) {
            ov = ov * 8 + p.charCodeAt(t.i) - 48
            t.i = t.i + 1
            ok = ok + 1
        }
        if (!t.utf && ov > 255) { rxPcreFail(t, "octal value is greater than \\377 in 8-bit non-UTF-8 mode", t.i) }
        rxPcreEmit(t, rxPcreLitText(ov), true)
        return
    }
    if (c == 103) { // \g
        t.i = t.i + 1
        var gc = t.i < t.n ? p.charCodeAt(t.i) : 0
        if (gc == 60 || gc == 39) {
            rxPcreFail(t, "unsupported subroutine call", at)
            return
        }
        var braced = gc == 123
        if (braced) { t.i = t.i + 1 }
        var sign = 0
        if (t.i < t.n && (p.charCodeAt(t.i) == 45 || p.charCodeAt(t.i) == 43)) {
            sign = p.charCodeAt(t.i) == 45 ? 0 - 1 : 1
            t.i = t.i + 1
        }
        if (t.i < t.n && rxIsDigitCode(p.charCodeAt(t.i))) {
            var gn = rxPcreNum(t)
            if (sign < 0) { gn = t.ncap - gn + 1 }
            else if (sign > 0) { gn = t.ncap + gn }
            if (braced) {
                if (t.i >= t.n || p.charCodeAt(t.i) != 125) {
                    rxPcreFail(t, "\\g is not followed by a braced, angle-bracketed, or quoted name/number or by a plain number", t.i)
                    return
                }
                t.i = t.i + 1
            }
            if (gn <= 0) {
                rxPcreFail(t, "reference to non-existent subpattern", t.i - 1)
                return
            }
            rxPcreRef(t, gn, "", at)
            return
        }
        if (braced && sign == 0) {
            var gname = rxPcreName(t, 125)
            if (t.err == "") { rxPcreRef(t, 0, gname, at) }
            return
        }
        rxPcreFail(t, "\\g is not followed by a braced, angle-bracketed, or quoted name/number or by a plain number", t.i)
        return
    }
    if (c == 107) { // \k<n> \k'n' \k{n}
        t.i = t.i + 1
        var kc = t.i < t.n ? p.charCodeAt(t.i) : 0
        if (kc != 60 && kc != 39 && kc != 123) {
            rxPcreFail(t, "\\k is not followed by a braced, angle-bracketed, or quoted name", t.i)
            return
        }
        t.i = t.i + 1
        var kname = rxPcreName(t, kc == 60 ? 62 : kc == 39 ? 39 : 125)
        if (t.err == "") { rxPcreRef(t, 0, kname, at) }
        return
    }
    var ch = rxPcreChar(t, false)
    if (t.err != "") { return }
    if (ch < 0) {
        rxPcreFail(t, c == 117 || c == 85 || c == 76 || c == 108 || c == 70 || c == 78
                      ? "PCRE2 does not support \\F, \\L, \\l, \\N{name}, \\U, or \\u"
                      : "unrecognized character follows \\", t.i)
        return
    }
    rxPcreEmit(t, rxPcreLitText(ch), true)
}

// rxPcre itself: {out, err, errAt, names, nameGroups} - see the head of this
// section.
function rxPcre(pattern, mods) {
    var t = {p: pattern, i: 0, n: pattern.length, err: "", errAt: 0 - 1, out: "", parts: [], refs: [],
             frames: [], ncap: 0, names: [], nameGroups: [], okq: false,
             utf: mods.indexOf("u") >= 0, dup: mods.indexOf("J") >= 0, dollarEnd: mods.indexOf("D") >= 0,
             fi: mods.indexOf("i") >= 0, fm: mods.indexOf("m") >= 0, fs: mods.indexOf("s") >= 0,
             fx: mods.indexOf("x") >= 0, fU: mods.indexOf("U") >= 0, fn: mods.indexOf("n") >= 0, fJ: false}
    var p = pattern
    var inq = false
    while (t.err == "") {
        if (!inq) { rxPcreSkip(t) }
        if (t.err != "" || t.i >= t.n) { break }
        var c = p.charCodeAt(t.i)
        if (inq) {
            if (c == 92 && t.i + 1 < t.n && p.charCodeAt(t.i + 1) == 69) {
                inq = false
                t.i = t.i + 2
                continue
            }
            var qc = rxCpAt(p, t.i)
            t.i = t.i + (qc > 65535 ? 2 : 1)
            rxPcreEmit(t, rxPcreLitText(qc), true)
            continue
        }
        if (c == 92 && t.i + 1 < t.n && p.charCodeAt(t.i + 1) == 81) {
            inq = true
            t.i = t.i + 2
            continue
        }
        if (c == 42 || c == 43 || c == 63 || (c == 123 && rxPcreBound(t) != null)) {
            rxPcreQuant(t)
        } else if (c == 40) {
            rxPcreParen(t)
        } else if (c == 41) {
            rxPcreClose(t)
        } else if (c == 124) {
            t.i = t.i + 1
            rxPcreEmit(t, "|", false)
        } else if (c == 94) {
            t.i = t.i + 1
            rxPcreEmit(t, t.fm ? "(?:\\A|(?m:^))" : "^", false)
        } else if (c == 36) {
            t.i = t.i + 1
            rxPcreEmit(t, t.fm ? "(?m:$)" : t.dollarEnd ? "\\z" : "$", false)
        } else if (c == 46) {
            t.i = t.i + 1
            rxPcreEmit(t, t.fs ? "(?s:.)" : ".", true)
        } else if (c == 91) {
            rxPcreClass(t)
        } else if (c == 92) {
            rxPcreEscape(t)
        } else {
            var lc = rxCpAt(p, t.i)
            t.i = t.i + (lc > 65535 ? 2 : 1)
            rxPcreEmit(t, rxPcreLitText(lc), true)
        }
    }
    if (t.err == "" && t.frames.length > 0) { rxPcreFail(t, "missing closing parenthesis", t.n) }
    var parts = t.parts
    parts.push(t.out)
    var refs = t.refs
    var ri = 0
    while (t.err == "" && ri < refs.length) {
        var r = refs[ri]
        var g = r.n
        if (r.name != "") {
            g = rxNameIdx(t.names, t.nameGroups, r.name)
            if (g < 0) { rxPcreFail(t, "reference to non-existent subpattern", r.at) }
        } else if (g > t.ncap) {
            rxPcreFail(t, "reference to non-existent subpattern", r.at)
        }
        parts[r.part] = "(?:\\" + g + ")"
        ri = ri + 1
    }
    var out = ""
    var pi = 0
    while (pi < parts.length) {
        out = out + parts[pi]
        pi = pi + 1
    }
    // i from the start; A (PCRE2_ANCHORED): only at the search's start.
    if (mods.indexOf("i") >= 0) { out = "(?i)" + out }
    if (mods.indexOf("A") >= 0) { out = "\\G(?:" + out + ")" }
    return {out: t.err == "" ? out : "", err: t.err, errAt: t.errAt, names: t.names, nameGroups: t.nameGroups,
            ncap: t.ncap}
}

// ===== Compilation =====

// rxSuppJ and rxClsBmpJ decide java.util.regex's hasSupplementary for a parsed
// pattern: does any character it matches come from a predicate that is not a
// BmpCharPredicate? Those are a single BMP character outside the surrogates,
// such a range, the ASCII shorthands, \h and \v, and unions and intersections
// of those; a complement, ., \p, a case-insensitive range and Unicode case
// folding are not. rxHasSurr adds a surrogate anywhere in the pattern text.
function rxSuppJ(node) {
    if (node.t == "cls") { return !rxClsBmpJ(node.cls) }
    var items = node.items
    var i = 0
    while (i < items.length) {
        if (rxSuppJ(items[i])) { return true }
        i = i + 1
    }
    return false
}

function rxClsBmpJ(cls) {
    if (cls.neg || cls.u || cls.props.length > 0) { return false }
    var lo = cls.lo
    var hi = cls.hi
    var i = 0
    while (i < lo.length) {
        if (hi[i] >= 55296 && !(lo[i] > 57343 && hi[i] < 65536)) { return false }
        i = i + 1
    }
    var sp = cls.sp
    i = 0
    while (i < sp.length) {
        var k = sp[i]
        if (k == RX_SND || k == RX_SNW || k == RX_SNS || k == RX_SJNH || k == RX_SJNV) { return false }
        i = i + 1
    }
    var subs = cls.subs
    i = 0
    while (i < subs.length) {
        if (!rxClsBmpJ(subs[i])) { return false }
        i = i + 1
    }
    var ands = cls.ands
    i = 0
    while (i < ands.length) {
        if (!rxClsBmpJ(ands[i])) { return false }
        i = i + 1
    }
    return true
}

function rxHasSurr(p) {
    var i = 0
    while (i < p.length) {
        var c = p.charCodeAt(i)
        if (c >= 55296 && c <= 57343) { return true }
        i = i + 1
    }
    return false
}

function rxCompile(pattern, flags) {
    var icase = false
    var dotall = false
    var multi = false
    var ext = false
    var octal = false
    var quoting = false
    var ere = false
    var jsrep = false
    var posix = false
    var java = false
    var py = false
    var bfail = false
    var ucase = false
    var unix = false
    var pyascii = false
    var re2 = false
    // C is followed by PHP's modifiers rather than these letters (rxPcre).
    var pcre = flags.length > 0 && flags.charAt(0) == "C"
    var fi = pcre ? flags.length : 0
    while (fi < flags.length) {
        var f = flags.charAt(fi)
        if (f == "i") { icase = true }
        else if (f == "s") { dotall = true }
        else if (f == "m") { multi = true }
        else if (f == "x") { ext = true }
        else if (f == "o") { octal = true }
        else if (f == "q") { quoting = true }
        else if (f == "e") { ere = true }
        else if (f == "j") { jsrep = true }
        else if (f == "p") { posix = true }
        else if (f == "J") { java = true }
        else if (f == "P") { py = true }
        else if (f == "n") { bfail = true }
        else if (f == "u") { ucase = true }
        else if (f == "d") { unix = true }
        else if (f == "a") { pyascii = true }
        else if (f == "R") { re2 = true }
        fi = fi + 1
    }
    // R reads the pattern as Go does and compiles its J spelling (rxRe2).
    var tr = rxRe2(re2 ? pattern : "")
    if (re2) { java = true }
    // C reads the pattern as PCRE2 does and compiles its J spelling (rxPcre),
    // with PHP's LF newline (UNIX_LINES) and, under u, Unicode case folding.
    var mods = pcre ? flags.substring(1, flags.length) : ""
    var tp = rxPcre(pcre ? pattern : "", mods)
    if (pcre) {
        java = true
        unix = true
        ucase = mods.indexOf("u") >= 0
    }
    if (java || py) { bfail = true }
    // The POSIX ERE dialect (bash's [[ =~ ]]). What it changes is spelled out at each
    // site; the two whole-engine consequences are here. A "." matches a newline in an
    // ERE (there is no "dot does not cross a line" rule to switch off), and an ERE has
    // POSIX capture semantics, of which the visible one is that an unbounded loop
    // whose iteration consumed nothing DISCARDS that iteration - so ^(a*)*$ on "aaa"
    // reports "aaa" for group 1, as bash and node do and Perl, Python, Ruby and Java
    // do not.
    if (ere) {
        dotall = true
        jsrep = true
        posix = true
    }
    // "p" on its own is the POSIX BRACKET EXPRESSIONS and nothing else. Ruby takes
    // [[:alpha:]] from POSIX but keeps \d, \w, \s, lazy quantifiers, (?...) groups
    // and an escaping backslash inside a bracket - so it wants this letter and NOT
    // the whole "e" dialect, which would switch all of those off.
    // \Q...\E is expanded before anything else looks at the text, so neither the
    // group counter nor the parser needs a quoting mode. re.src keeps the ORIGINAL
    // pattern, which is what a language's .source / inspect reports.
    var pat = re2 ? tr.out : pcre ? tp.out : quoting ? rxExpandQuotes(pattern) : pattern
    var st = {
        p: pat, i: 0, n: pat.length, err: "",
        ngroups: 0, names: [], nameGroups: [], pending: [], ext: ext, dotall: dotall,
        octal: octal, ntotal: ere ? 0 : rxCountGroups(pat), ere: ere, posix: posix,
        depth: 0, java: java, errAt: -1, pcls: rxNewCls(), gnodes: [rxNode("empty")],
        py: py, icase: icase, multi: multi, gend: 0, gflags: "",
        ucase: ucase && java, uclass: false, unix: unix && java, inl: false, supp: false,
        canon: false, pyuni: py && !pyascii, re2: re2, pcre: pcre
    }
    // Under P and J case folding and line anchoring are decided per atom while
    // parsing.
    if (py || java) {
        icase = false
        multi = false
    }
    var ast = rxParseAlt(st)
    // A search steps over a surrogate pair whole: under J when the pattern has
    // a supplementary character (java.util.regex's hasSupplementary), and always
    // under P, where the text is code points.
    var supp = py || java && (re2 || st.supp || rxSuppJ(ast) || rxHasSurr(pat)) || pcre && ucase
    // Resolve every \k<name> now that the whole name table is known.
    var pend = st.pending
    var pi = 0
    while (pi < pend.length) {
        var gi = rxNameIdx(st.names, st.nameGroups, pend[pi].bname)
        if (gi < 0) { rxErr(st, "unknown group name '" + pend[pi].bname + "'") }
        pend[pi].gidx = gi
        pi = pi + 1
    }
    if (st.err == "" && st.i < st.n) {
        if (pat.charCodeAt(st.i) == 41) { rxErr(st, "unmatched ) in pattern") }
        else { rxErr(st, "unexpected character in pattern") }
    }
    // CPython checks a look-behind's width when it compiles the parsed pattern,
    // so every parse error is reported first.
    if (py && st.err == "" && rxLookbVarP(st, ast)) { rxErr(st, "look-behind requires fixed-width pattern") }
    if (pcre && st.err == "" && rxLookbLongC(ast)) { rxErr(st, "length of lookbehind assertion is not limited") }
    if (tr.err != "") { st.err = tr.err }
    if (tp.err != "") {
        st.err = tp.err
        st.errAt = tp.errAt
    }
    if (st.err != "") {
        return {ok: false, err: st.err, errAt: st.errAt, ops: [], as: [], bs: [], clss: [],
                ngroups: 0, nmarks: 0, names: [], nameGroups: [],
                icase: icase, dotall: dotall, multi: multi, jsrep: jsrep, java: java,
                py: false, supp: false, bfail: bfail, gflags: "",
                cmin: [], cmax: [], cgreedy: [], lbmin: [], lbmax: [],
                src: pattern, flags: flags, hitEnd: false, reqEnd: false, jbegin: false, jmin: 0, jcut: 0}
    }
    var pg = rxNewProg()
    pg.jsrep = jsrep
    pg.java = java
    pg.re2 = re2
    rxAdd(pg, RX_SAVE, 0, 0, pg.dummy)
    rxEmit(pg, ast)
    rxAdd(pg, RX_SAVE, 1, 0, pg.dummy)
    rxAdd(pg, RX_MATCH, 0, 0, pg.dummy)
    var jbegin = java && rxBeginJ(ast)
    // Go's and PCRE2's searches have no Start node to model (R, C).
    var jmin = java && !re2 && !pcre && !jbegin ? rxMinLen(ast) : 0
    return {
        ok: true, err: "", errAt: -1, ops: pg.ops, as: pg.as, bs: pg.bs, clss: pg.clss,
        ngroups: st.ngroups, nmarks: pg.nmarks, names: re2 ? tr.names : pcre ? tp.names : st.names,
        nameGroups: re2 ? tr.nameGroups : pcre ? tp.nameGroups : st.nameGroups,
        icase: icase, dotall: dotall, multi: multi, jsrep: jsrep, java: java,
        py: py, supp: supp, bfail: bfail, gflags: java ? rxFlagsGet(st) : st.gflags, pcre: pcre,
        cmin: pg.cmin, cmax: pg.cmax, cgreedy: pg.cgreedy, lbmin: pg.lbmin, lbmax: pg.lbmax,
        src: pattern, flags: flags, hitEnd: false, reqEnd: false, jbegin: jbegin, jmin: jmin,
        jcut: 0
    }
}

// rxBeginJ: does the pattern start with java.util.regex's Begin node (\A, or
// ^ without MULTILINE, first in the top-level sequence, inline flags aside)?
// Pattern then searches from that node itself rather than through Start, whose
// failure always sets hitEnd and which tries no start that leaves fewer than
// the pattern's shortest match (re.jmin, rxSearchFrom).
function rxBeginJ(ast) {
    if (ast.t != "cat") { return false }
    var items = ast.items
    var i = 0
    while (i < items.length && items[i].t == "empty") { i = i + 1 }
    return i < items.length && items[i].t == "assert" && items[i].gidx == RX_BOS
}


// ===== The matcher =====

function rxAssertOK(re, text, kind, sp) {
    var n = text.length
    if (kind == RX_BOS) { return sp == 0 }
    if (kind == RX_EOS) {
        if (sp != n) { return false }
        // Pattern.End.
        re.hitEnd = true
        return true
    }
    if (kind == RX_EOSNL) {
        if (sp == n) { return true }
        return sp == n - 1 && text.charCodeAt(sp) == 10
    }
    if (kind == RX_BOL || kind == RX_MBOL) {
        if (sp == 0) { return true }
        if (!re.multi && kind == RX_BOL) { return false }
        return text.charCodeAt(sp - 1) == 10
    }
    if (kind == RX_EOL || kind == RX_MEOL) {
        if (sp == n) { return true }
        if (!re.multi && kind == RX_EOL) { return false }
        return text.charCodeAt(sp) == 10
    }
    if (kind == RX_PUWORDB || kind == RX_PUNWORDB) {
        var pb = sp > 0 && rxPyWordHit(rxCpAt(text, rxCpBack(text, sp)))
        var pa = sp < n && rxPyWordHit(rxCpAt(text, sp))
        if (kind == RX_PUWORDB) { return pb != pa }
        return pb == pa
    }
    if (kind >= RX_JBOL) { return rxAssertJ(re, text, kind, sp) }
    var before = false
    var after = false
    if (sp > 0) { before = rxIsWordCode(text.charCodeAt(sp - 1)) }
    if (sp < n) { after = rxIsWordCode(text.charCodeAt(sp)) }
    if (kind == RX_WORDB) { return before != after }
    return before == after
}

// rxIsLineSepJ: is c one of Java's line terminators (unix: \n alone)?
function rxIsLineSepJ(c, unix) {
    if (c == 10) { return true }
    if (unix) { return false }
    return c == 13 || c == 133 || c == 8232 || c == 8233
}

// rxAssertJ decides the RX_J* anchors (see their definitions), and sets
// re.hitEnd and re.reqEnd where Pattern's Caret, Dollar, UnixDollar and Bound
// do.
function rxAssertJ(re, text, kind, sp) {
    var n = text.length
    if (kind == RX_JBOL || kind == RX_JUBOL) {
        // Not at the very end, even after a terminator; not inside \r\n.
        if (sp == n) {
            re.hitEnd = true
            return false
        }
        if (sp == 0) { return true }
        var pc = text.charCodeAt(sp - 1)
        if (!rxIsLineSepJ(pc, kind == RX_JUBOL)) { return false }
        return !(pc == 13 && kind == RX_JBOL && text.charCodeAt(sp) == 10)
    }
    if (kind <= RX_JUMEOL) {
        var unix = kind == RX_JUEOL || kind == RX_JUMEOL
        var multi = kind == RX_JMEOL || kind == RX_JUMEOL
        if (sp < n) {
            var c = text.charCodeAt(sp)
            if (!rxIsLineSepJ(c, unix)) { return false }
            if (multi) { return !(c == 10 && !unix && sp > 0 && text.charCodeAt(sp - 1) == 13) }
            if (unix && sp != n - 1) { return false }
            if (!unix && sp == n - 2 && !(c == 13 && text.charCodeAt(sp + 1) == 10)) { return false }
            if (!unix && sp != n - 2 && (sp != n - 1 || (c == 10 && sp > 0 && text.charCodeAt(sp - 1) == 13))) { return false }
        }
        // At the end, or (without MULTILINE) before the final terminator:
        // more text could move the end.
        re.hitEnd = true
        re.reqEnd = true
        return true
    }
    if (sp >= n) {
        // Pattern.Bound read past the end: more text could break the boundary.
        re.hitEnd = true
        re.reqEnd = true
    }
    var uw = kind == RX_JUWORDB || kind == RX_JUNWORDB
    var before = false
    if (sp > 0) {
        // Character.codePointBefore: a pair only when the unit before sp is
        // its low half.
        var bc = text.charCodeAt(sp - 1)
        if (bc >= 56320 && bc <= 57343 && sp > 1 && rxCpAt(text, sp - 2) > 65535) {
            bc = rxCpAt(text, sp - 2)
        }
        before = rxIsWordJ(text, bc, sp - 1, uw)
    }
    var after = false
    if (sp < n) {
        var ac = rxCpAt(text, sp)
        // Character.codePointAt reads a pair a region end splits (re.jcut).
        if (sp + 1 == n && re.jcut != 0 && ac >= 55296 && ac <= 56319) { ac = 65536 + (ac - 55296) * 1024 + re.jcut - 56320 }
        after = rxIsWordJ(text, ac, sp, uw)
    }
    if (kind == RX_JWORDB || kind == RX_JUWORDB) { return before != after }
    return before == after
}

// rxIsWordJ is Pattern.Bound's side test for the character c: a word
// character (ASCII, or Unicode's under UNICODE_CHARACTER_CLASS), or a
// non-spacing mark that hasBaseCharacter accepts, reading the code point at
// each unit from i down to the start.
function rxIsWordJ(text, c, i, uw) {
    if (uw ? rxPropHit(RX_PWORD, c) : rxIsWordCode(c)) { return true }
    // No non-spacing mark lies below U+0300.
    if (c < 768 || !rxPropHit(RX_PMN, c)) { return false }
    var k = i
    while (k >= 0) {
        var b = rxCpAt(text, k)
        if (rxPropHit(RX_PLD, b)) { return true }
        if (b < 768 || !rxPropHit(RX_PMN, b)) { return false }
        k = k - 1
    }
    return false
}

// The backtrack stack: entry i is rxBk[i] (its kind), rxBx[i] and rxBy[i].
// RX_BT_ALT resumes at pc x, sp y; the four undo kinds put y back into slot x of
// caps, marks, st.ctr or st.cbeg; RX_BT_ITER enters another iteration of the
// counted loop whose RX_CLOOP is at x, from sp y (the lazy loop's second choice).
// One stack serves every run, as no run starts while another is in progress
// except a nested one, which works above it (rxRun's base); it only grows.
var rxBk = []
var rxBx = []
var rxBy = []
var RX_BT_ALT = 0
var RX_BT_CAP = 1
var RX_BT_MARK = 2
var RX_BT_CTR = 3
var RX_BT_CBEG = 4
var RX_BT_ITER = 5

// rxRun is the backtracking VM. It answers the end position of a match starting at
// the given pc/sp, or -1. A choice (SPLIT, a counted loop's CLOOP) pushes the
// alternative not taken onto the backtrack stack, from entry base up, and a write
// that has to be undone (SAVE, MARK, CLEAR, the counted-loop registers) pushes the
// old value. A failure leaves the forward loop, pops back to the latest choice,
// undoing the writes on the way, and runs forward again from it - so a long
// subject cannot exhaust the native stack. A write is undone only above lim:
// unless keep (the nested run of a lookaround or atomic group body, whose failure
// must leave everything as it found it), that is only when a choice lies below
// it, as a failed attempt's registers are thrown away. A nested run works above
// this one's top, and a body that succeeds leaves its entries there, never
// revisited.
function rxRun(re, text, pcIn, spIn, caps, marks, st, base, keep) {
    var lim = keep ? base - 1 : base
    var ops = re.ops
    var as = re.as
    var bs = re.bs
    var clss = re.clss
    var bk = rxBk
    var bx = rxBx
    var by = rxBy
    var btn = bk.length
    var n = text.length
    var pc = pcIn
    var sp = spIn
    var top = base
    while (true) {
        while (true) {
            st.steps = st.steps + 1
            if (st.steps > RX_STEPS) {
                st.over = true
                return -1
            }
            var op = ops[pc]
            if (op == RX_CLASS) {
                if (sp >= n) {
                    re.hitEnd = true
                    break
                }
                var cc = text.charCodeAt(sp)
                // J and P read a surrogate pair as one character.
                if (cc >= 55296 && cc <= 56319 && (re.java || re.py)) {
                    // Pattern.CharProperty reads a pair a region end splits
                    // (re.jcut) and fails past the end; CharPropertyGreedy
                    // (cls.cpg) reads its high half, as a BmpCharProperty
                    // does, but has hit the end.
                    if (sp + 1 == n && re.jcut != 0 && !rxClsBmpJ(clss[pc])) {
                        re.hitEnd = true
                        if (!clss[pc].cpg) { break }
                    } else {
                        cc = rxCpAt(text, sp)
                    }
                }
                if (!rxClsMatch(clss[pc], cc, re.icase)) {
                    break
                }
                pc = pc + 1
                sp = sp + (cc > 65535 ? 2 : 1)
            } else if (op == RX_JMP) {
                pc = as[pc]
            } else if (op == RX_SPLIT) {
                var ta = top
                if (ta == btn) { btn = rxBtGrow() }
                bk[ta] = RX_BT_ALT
                bx[ta] = bs[pc]
                by[ta] = sp
                top = ta + 1
                pc = as[pc]
            } else if (op == RX_SAVE) {
                var slot = as[pc]
                var tb = top
                if (tb > lim) {
                    if (tb == btn) { btn = rxBtGrow() }
                    bk[tb] = RX_BT_CAP
                    bx[tb] = slot
                    by[tb] = caps[slot]
                    top = tb + 1
                }
                caps[slot] = sp
                pc = pc + 1
            } else if (op == RX_CLEAR) {
                var ci = 2 * as[pc]
                var cTo = 2 * bs[pc] + 1
                while (ci <= cTo) {
                    var tc = top
                    if (tc > lim) {
                        if (tc == btn) { btn = rxBtGrow() }
                        bk[tc] = RX_BT_CAP
                        bx[tc] = ci
                        by[tc] = caps[ci]
                        top = tc + 1
                    }
                    caps[ci] = -1
                    ci = ci + 1
                }
                pc = pc + 1
            } else if (op == RX_MARK) {
                var mslot = as[pc]
                var td = top
                if (td > lim) {
                    if (td == btn) { btn = rxBtGrow() }
                    bk[td] = RX_BT_MARK
                    bx[td] = mslot
                    by[td] = marks[mslot]
                    top = td + 1
                }
                marks[mslot] = sp
                pc = pc + 1
            } else if (op == RX_PROG) {
                if (marks[as[pc]] == sp) {
                    // The iteration consumed nothing. Two readings, and they differ ONLY in
                    // what the captures inside the body are left holding:
                    //   jsrep off - leave the loop from here, KEEPING whatever that empty
                    //     iteration saved. /^(a*)*$/ on "aaa" then reports "" for group 1,
                    //     which is what Perl, Python, Ruby and Java print.
                    //   jsrep on  - FAIL the iteration, so the run backtracks through the
                    //     MARK/SAVE writes (restoring them) and the SPLIT leaves the
                    //     loop as if the body had never been entered. Group 1 keeps
                    //     "aaa", which is what node and bash print.
                    // Failing here always terminates: the SPLIT above has exactly one
                    // alternative left, and it is the loop exit.
                    // Under R a PROG whose b is -1 always fails (rxEmitRepRe2).
                    if (re.jsrep || bs[pc] < 0) {
                        break
                    }
                    pc = bs[pc]
                } else { pc = pc + 1 }
            } else if (op == RX_ASSERT) {
                if (!rxAssertOK(re, text, as[pc], sp)) {
                    break
                }
                pc = pc + 1
            } else if (op == RX_LOOK) {
                // Under C the captures are put back when the rest of the match fails
                // past here (rxUndoAll); the op then runs again, st.wrap saying so.
                if (re.pcre && st.wrap != pc) {
                    if (top > lim) { top = rxUndoAll(top, caps) }
                    btn = bk.length
                    st.wrap = pc
                    continue
                }
                st.wrap = 0 - 1
                // Pattern.Neg: more text could make the body match.
                if (sp >= n && bs[pc] != 0) { re.reqEnd = true }
                var rl = rxRun(re, text, pc + 1, sp, caps, marks, st, top, true)
                btn = bk.length
                if (st.over) { return -1 }
                var hit = rl >= 0
                var want = bs[pc] == 0
                if (hit != want) {
                    break
                }
                pc = as[pc]
            } else if (op == RX_LOOKEND) {
                return sp
            } else if (op == RX_BREF) {
                var g = as[pc]
                // A reference to a group NUMBER the pattern never defined (\1 in a
                // pattern with no groups) is out of range in caps. Treated exactly like
                // a group that did not take part. Without
                // the length test the JS read is undefined and sp silently goes NaN,
                // and the Go twin panics with an index-out-of-range.
                var from = -1
                var to = -1
                if (2 * g + 1 < caps.length) {
                    from = caps[2 * g]
                    to = caps[2 * g + 1]
                }
                if (from < 0 || to < 0) {
                    // A group that did not take part matches the empty string - except
                    // under n (Java, Python, Ruby), where the backreference fails.
                    if (re.bfail) {
                        break
                    }
                    pc = pc + 1
                } else {
                    var len = to - from
                    if (sp + len > n) {
                        re.hitEnd = true
                        break
                    }
                    var bi = 0
                    var bmis = false
                    while (bi < len && !bmis) {
                        var lc = text.charCodeAt(from + bi)
                        var rc = text.charCodeAt(sp + bi)
                        // P compares an astral character whole.
                        if (re.py && lc >= 55296 && lc <= 56319) {
                            lc = rxCpAt(text, from + bi)
                            rc = rxCpAt(text, sp + bi)
                            if (lc > 65535) { bi = bi + 1 }
                        }
                        if (lc != rc) {
                            if (!re.icase && bs[pc] == 0) {
                                bmis = true
                            } else if (bs[pc] == 2) {
                                // Java's CIBackRef under UNICODE_CASE.
                                var lu = rxCaseU(lc, 1)
                                var ru = rxCaseU(rc, 1)
                                if (lu != ru && rxCaseU(lu, 0) != rxCaseU(ru, 0)) { bmis = true }
                            } else if (rxSwapCase(rc) != lc) {
                                bmis = true
                            }
                        }
                        bi = bi + 1
                    }
                    if (bmis) {
                        break
                    }
                    pc = pc + 1
                    sp = sp + len
                }
            } else if (op == RX_MATCH) {
                // rxMatchWhole CONSTRAINS THE RUN rather than rewriting the pattern: a
                // match that stops short of mustEnd is refused right here, so the VM
                // keeps backtracking for a longer one. Rewriting the pattern to
                // \A(?:...)\z instead breaks under the x flag, where a trailing #
                // comment swallows the wrapper's own closing paren.
                // adv is the start of an attempt that may not match empty (rxSearchAdv).
                if ((st.mustEnd >= 0 && sp != st.mustEnd) || sp == st.adv) {
                    break
                }
                return sp
            } else if (op == RX_CLASSX) {
                if (sp >= n) {
                    re.hitEnd = true
                    break
                }
                var xc = rxCpAt(text, sp)
                if (sp + 1 == n && re.jcut != 0 && xc >= 55296 && xc <= 56319 && !rxClsBmpJ(clss[pc])) {
                    re.hitEnd = true
                    if (!clss[pc].cpg) { break }
                }
                if (!rxClsMatchX(clss[pc], xc, re.icase)) {
                    break
                }
                pc = pc + 1
                sp = sp + (xc > 65535 ? 2 : 1)
            } else if (op == RX_CINIT) {
                // The count goes back to its old value on backtracking, so a loop
                // re-entered by an enclosing one starts from zero.
                var cslot = as[pc]
                var ctr = st.ctr
                var tm = top
                if (tm > lim) {
                    if (tm == btn) { btn = rxBtGrow() }
                    bk[tm] = RX_BT_CTR
                    bx[tm] = cslot
                    by[tm] = ctr[cslot]
                    top = tm + 1
                }
                ctr[cslot] = 0
                pc = pc + 1
            } else if (op == RX_CLOOP) {
                // Another iteration records where it began (cbeg), so RX_CINC can tell
                // an empty one.
                var lslot = as[pc]
                var lctr = st.ctr
                var cnt = lctr[lslot]
                var lmaxs = re.cmax
                var lmins = re.cmin
                var lgreedy = re.cgreedy
                if (lmaxs[lslot] >= 0 && cnt >= lmaxs[lslot]) {
                    pc = bs[pc]
                } else if (cnt >= lmins[lslot] && lgreedy[lslot] != 1) {
                    var te = top
                    if (te == btn) { btn = rxBtGrow() }
                    bk[te] = RX_BT_ITER
                    bx[te] = pc
                    by[te] = sp
                    top = te + 1
                    pc = bs[pc]
                } else {
                    if (cnt >= lmins[lslot]) {
                        var tf = top
                        if (tf == btn) { btn = rxBtGrow() }
                        bk[tf] = RX_BT_ALT
                        bx[tf] = bs[pc]
                        by[tf] = sp
                        top = tf + 1
                    }
                    var tg = top
                    if (tg == btn) { btn = rxBtGrow() }
                    var lbeg = st.cbeg
                    bk[tg] = RX_BT_CBEG
                    bx[tg] = lslot
                    by[tg] = lbeg[lslot]
                    top = tg + 1
                    lbeg[lslot] = sp
                    pc = pc + 1
                }
            } else if (op == RX_CINC) {
                var islot = as[pc]
                var ibeg = st.cbeg
                if (ibeg[islot] == sp) {
                    pc = pc + 1
                } else {
                    var ictr = st.ctr
                    var tn = top
                    if (tn > lim) {
                        if (tn == btn) { btn = rxBtGrow() }
                        bk[tn] = RX_BT_CTR
                        bx[tn] = islot
                        by[tn] = ictr[islot]
                        top = tn + 1
                    }
                    ictr[islot] = ictr[islot] + 1
                    pc = bs[pc]
                }
            } else if (op == RX_LOOKB) {
                if (re.pcre && st.wrap != pc) {
                    if (top > lim) { top = rxUndoAll(top, caps) }
                    btn = bk.length
                    st.wrap = pc
                    continue
                }
                st.wrap = 0 - 1
                // Try the body from every start the length range allows, shortest
                // first, each run required to end exactly here (RX_LBEND).
                var lid = Math.floor(bs[pc] / 2)
                var lneg = bs[pc] % 2 == 1
                var lbmaxs = re.lbmax
                var lbmins = re.lbmin
                var lmax = lbmaxs[lid]
                var lfrom = sp - lbmins[lid]
                var lto = lmax < 0 ? 0 : sp - lmax
                // The range counts characters; under P one may be a surrogate pair.
                if (re.py) {
                    lfrom = rxCpBackN(text, sp, lbmins[lid])
                    lto = lmax < 0 ? 0 : rxCpBackN(text, sp, lmax)
                }
                if (lto < 0) { lto = 0 }
                var lsaved = st.lbt
                st.lbt = sp
                var lhit = false
                while (!lhit && lfrom >= lto) {
                    if (rxRun(re, text, pc + 1, lfrom, caps, marks, st, top, true) >= 0) { lhit = true }
                    if (st.over) { return -1 }
                    lfrom = re.py ? rxCpBack(text, lfrom) : lfrom - 1
                }
                btn = bk.length
                st.lbt = lsaved
                if (lhit == lneg) {
                    break
                }
                pc = as[pc]
            } else if (op == RX_LBEND) {
                if (sp != st.lbt) {
                    break
                }
                return sp
            } else if (op == RX_ATOM) {
                // The body's first success, and no other: the run below stops at
                // RX_ATOMEND, and nothing here is retried when the rest fails.
                // Captures the body set stay set then, as in Java; under C they
                // are taken back, as in PCRE2.
                if (re.pcre && st.wrap != pc) {
                    if (top > lim) { top = rxUndoAll(top, caps) }
                    btn = bk.length
                    st.wrap = pc
                    continue
                }
                st.wrap = 0 - 1
                var ra = rxRun(re, text, pc + 1, sp, caps, marks, st, top, true)
                btn = bk.length
                if (st.over) { return -1 }
                if (ra < 0) {
                    break
                }
                pc = as[pc]
                sp = ra
            } else if (op == RX_ATOMEND) {
                return sp
            } else if (op == RX_GPOS) {
                if (sp != st.gpos) {
                    break
                }
                pc = pc + 1
            } else if (op == RX_XGRAPH) {
                if (sp >= n) {
                    re.hitEnd = true
                    break
                }
                // Grapheme.nextBoundary reads a pair a region end splits whole.
                sp = re.jcut == 0 ? rxGraphNext(text, sp, n) : rxGraphNext(text + String.fromCharCode(re.jcut), sp, n)
                pc = pc + 1
            } else if (op == RX_GBOUND) {
                // Pattern.GraphemeBound: not between a surrogate pair, and not before
                // the first boundary after the previous match's end.
                if (sp > 0 && sp < n) {
                    var hs = text.charCodeAt(sp - 1)
                    var ls = text.charCodeAt(sp)
                    if ((hs >= 55296 && hs <= 56319 && ls >= 56320 && ls <= 57343) || rxGraphNext(text, st.glast, n) > sp) {
                        break
                    }
                } else if (sp > 0) {
                    re.hitEnd = true
                    re.reqEnd = true
                }
                pc = pc + 1
            } else {
                break
            }
        }
        // The forward run failed: pop entries until one resumes it.
        while (true) {
            if (top == base) { return -1 }
            top = top - 1
            var uk = bk[top]
            var ux = bx[top]
            var uy = by[top]
            if (uk == RX_BT_ALT) {
                pc = ux
                sp = uy
                break
            } else if (uk == RX_BT_CAP) {
                caps[ux] = uy
            } else if (uk == RX_BT_ITER) {
                // The entry just popped is the room for the iteration's own.
                var uslot = as[ux]
                var ucb = st.cbeg
                if (top > lim) {
                    bk[top] = RX_BT_CBEG
                    bx[top] = uslot
                    by[top] = ucb[uslot]
                    top = top + 1
                }
                ucb[uslot] = uy
                pc = ux + 1
                sp = uy
                break
            } else if (uk == RX_BT_MARK) {
                marks[ux] = uy
            } else if (uk == RX_BT_CTR) {
                st.ctr[ux] = uy
            } else {
                st.cbeg[ux] = uy
            }
        }
    }
    return -1
}

// rxUndoAll pushes every capture's current value onto the backtrack stack
// above top, and answers the new top: PCRE2 forgets what a lookaround or atomic
// group captured once the match backtracks past it, and what a negative
// lookaround's body captured.
function rxUndoAll(top, caps) {
    var bk = rxBk
    var bx = rxBx
    var by = rxBy
    var t = top
    var i = 0
    while (i < caps.length) {
        if (t == bk.length) { rxBtGrow() }
        bk[t] = RX_BT_CAP
        bx[t] = i
        by[t] = caps[i]
        t = t + 1
        i = i + 1
    }
    return t
}

// rxBtGrow makes room for 64 more backtrack-stack entries and answers the
// stack's new size.
function rxBtGrow() {
    var bk = rxBk
    var bx = rxBx
    var by = rxBy
    var i = 0
    while (i < 64) {
        bk.push(0)
        bx.push(0)
        by.push(0)
        i = i + 1
    }
    return bk.length
}

// rxSearch finds the leftmost match at or after `start`. It answers null for no
// match, and a record {begin, end, caps} otherwise; caps[2i]/caps[2i+1] bound
// group i, and are -1 when the group did not take part.
// rxAttempt runs the program ONCE, starting at `at`. mustEnd is -1 for an
// ordinary match and the required end position for a whole-input match; it is the
// one knob rxSearch, rxMatchAt and rxMatchWhole differ by. adv is -1, or `at`
// when an empty match there is refused (rxSearchAdv).
function rxAttempt(re, text, at, mustEnd, gpos, glast, adv) {
    var nslots = 2 * (re.ngroups + 1)
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
    var ctr = []
    var cbeg = []
    var cmin = re.cmin
    i = 0
    while (i < cmin.length) {
        ctr.push(0)
        cbeg.push(-1)
        i = i + 1
    }
    var st = {steps: 0, over: false, mustEnd: mustEnd, lbt: -1, ctr: ctr, cbeg: cbeg,
              gpos: gpos, glast: glast, adv: adv, wrap: 0 - 1}
    var end = rxRun(re, text, 0, at, caps, marks, st, 0, false)
    if (st.over) { rxFail("step limit exceeded") }
    if (end >= 0) { return {begin: caps[0], end: caps[1], caps: caps} }
    return null
}

function rxSearch(re, text, start) {
    if (!re.ok) { rxFail("invalid regexp: " + re.err) }
    var from = start
    if (from < 0) { from = 0 }
    var at = from
    while (at <= text.length) {
        var m = rxAttempt(re, text, at, -1, from, 0, -1)
        if (m != null) { return m }
        // After a failed attempt a J pattern with re.supp steps over a surrogate
        // pair whole, as java.util.regex's StartS does; any other steps one
        // unit, as Start does.
        at = re.supp && at + 1 < text.length && rxCpAt(text, at) > 65535 ? at + 2 : at + 1
    }
    return null
}

// rxSearchAdv is rxSearch with CPython's must_advance: when adv, a match that
// is empty at start itself is refused, though a longer one there is not. The
// search after an empty match asks for this, so that it moves on.
function rxSearchAdv(re, text, start, adv) {
    if (!adv || start < 0 || start > text.length) { return rxSearch(re, text, start) }
    var m = rxAttempt(re, text, start, -1, start, 0, start)
    if (m != null) { return m }
    return rxSearch(re, text, start + (re.supp && start + 1 < text.length && rxCpAt(text, start) > 65535 ? 2 : 1))
}

// rxSearchFrom is rxSearch for java.util.regex's Matcher.find, whose \G and
// \b{g} read where the previous match ended (gpos, glast; see the API). Like
// Pattern's Start it tries no start that leaves fewer than re.jmin units,
// where no match can begin and an anchor's re.reqEnd must not be set. Not
// folded into rxSearch, which every language calls once per match.
function rxSearchFrom(re, text, start, gpos, glast) {
    if (!re.ok) { rxFail("invalid regexp: " + re.err) }
    var at = start
    var last = text.length - re.jmin
    while (at <= last) {
        var m = rxAttempt(re, text, at, -1, gpos, glast, -1)
        if (m != null) { return m }
        at = re.supp && at + 1 < text.length && rxCpAt(text, at) > 65535 ? at + 2 : at + 1
    }
    return null
}

// rxMatchAt: the match must BEGIN at `at` - no scanning forward. This is what a
// sticky (JavaScript /y/) match and Python's re.match(pattern, s, pos) want, and
// it is strictly cheaper than searching and then rejecting begin != at.
function rxMatchAt(re, text, at) {
    if (!re.ok) { rxFail("invalid regexp: " + re.err) }
    if (at < 0 || at > text.length) { return null }
    return rxAttempt(re, text, at, -1, at, 0, -1)
}

// rxMatchWhole: the match must cover the WHOLE input. This cannot be a search plus
// a length check - `(a|ab)` against "ab" has to BACKTRACK into the alternation,
// and a leftmost search answers "a" and would wrongly report no match. Python's
// re.fullmatch, Kotlin's Regex.matches / matchEntire and Ruby's \A...\z idiom all
// want exactly this.
function rxMatchWhole(re, text, at) {
    if (!re.ok) { rxFail("invalid regexp: " + re.err) }
    if (at < 0 || at > text.length) { return null }
    return rxAttempt(re, text, at, text.length, at, 0, -1)
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

// rxReplace substitutes every match (or only the first when `all` is false),
// each search a Matcher's next find (rxSearchFrom).
// The replacement is expanded: \0 / \1 .. \9 and \k<name> stand for a group,
// and a doubled backslash for one backslash. Languages spell their template
// differently ($1 in JS, \1 in Ruby); a grammar that wants another spelling
// rewrites the template before calling this.
function rxReplace(re, text, repl, all) {
    var out = ""
    var at = 0
    var prev = 0
    while (at <= text.length) {
        var m = rxSearchFrom(re, text, at, prev, prev)
        if (m == null) { break }
        prev = m.end
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

// ===== The compiled-pattern cache, and the neutral match value ==============
//
// Shared by js, kotlin, python and ruby. Twins: rxCache/rxGet, rxIsRegexObj,
// rxIsMatchObj, rxObjRe and rxMatchOf in abnf/jsrtregex.go. The names are
// deliberately the Go twin's, so the two files line up when they are diffed.
//
// THE KEY IS NOT THE GO KEY, and that is deliberate. Go writes
// `flags + "\x00" + pattern`; here the key is prefixed "$" and separated by a
// space, because a MetaJS object is a string-keyed map and a pattern that
// happens to spell an Object.prototype member ("toString", "valueOf") would
// otherwise read back a function from the prototype instead of undefined. See
// docs/abnf-dialect-gotchas.md. The neutral flag string is only ever drawn from
// "ismxqjpeJnPud" (or is C and PHP's modifier letters), so the space is an
// unambiguous separator.
var rxCache = {}

function rxGet(pattern, flags) {
    var key = "$" + flags + " " + pattern
    var got = rxCache[key]
    if (got !== undefined) { return got }
    var re = rxCompile(pattern, flags)
    rxCache[key] = re
    return re
}

// rxGFlagBits answers CPython's flag bits for P's inline flag letters (re.gflags),
// which Python ORs into a pattern's .flags. Twin in abnf/jsrtregex.go.
function rxGFlagBits(letters) {
    var n = 0
    if (letters.indexOf("i") >= 0) { n = n | 2 }
    if (letters.indexOf("m") >= 0) { n = n | 8 }
    if (letters.indexOf("s") >= 0) { n = n | 16 }
    if (letters.indexOf("u") >= 0) { n = n | 32 }
    if (letters.indexOf("x") >= 0) { n = n | 64 }
    if (letters.indexOf("a") >= 0) { n = n | 256 }
    return n
}

// ----- The neutral Regexp / MatchData value -----
//
// Ruby's Regexp and MatchData, Kotlin's Regex and MatchResult and Python's
// compiled pattern and match are ONE pair of shapes: a plain object carrying
// __rxsrc / __rxflags, and, for a match, __rxmd / __rxtext / __rxcaps as well.
// Each language adds its own marker beside them (__ktsrc, __pymatch), which is
// why only the bodies that read the SHARED slots live here.
//
// The object and nil tests are spelled out rather than calling a language's
// helpers: js-rt.metajs imports this file and does NOT import runtime.metajs.
// Python does not use rxIsMatchObj: its pyERxIsMatch asks __pymatch, because a
// Python match and a compiled pattern both carry __rx* slots.
//
// rxMatchOf applies no language number coercion to __rxcaps: it is written only
// by k5NewMatch, rbRxNewMatch and pyERxMkMatch, all of which copy the engine's
// own m.caps, so every value is already a plain integer (the Go twin likewise
// does `f, _ := e.(float64)`).
function rxIsRegexObj(v) {
    if (v === null) { return false }
    if (v === undefined) { return false }
    if (typeof v != "object") { return false }
    if (v["__rxsrc"] === undefined) { return false }
    var md = v["__rxmd"]
    if (md === undefined) { return true }
    return md === null
}

function rxIsMatchObj(v) {
    if (v === null) { return false }
    if (v === undefined) { return false }
    if (typeof v != "object") { return false }
    var md = v["__rxmd"]
    if (md === undefined) { return false }
    return md !== null
}

// The program behind a Regexp or a MatchData, out of its own __rxsrc/__rxflags.
// For a match those are the source the match RAN on - which for Python's
// re.match is the ANCHORED source, and must not be confused with the pattern
// object's __src.
function rxObjRe(v) {
    var src = v["__rxsrc"]
    var flags = v["__rxflags"]
    if (typeof src != "string") { src = "" }
    if (typeof flags != "string") { flags = "" }
    return rxGet(src, flags)
}

// Go's rxMatchOf answers three values; MetaJS has one return, so the three
// travel in a record. `text` stays a STRING: this engine indexes UTF-16 code
// units where the Go engine indexes runes, and the two agree throughout the BMP.
function rxMatchOf(v) {
    var re = rxObjRe(v)
    var text = v["__rxtext"]
    if (typeof text != "string") { text = "" }
    var arr = v["__rxcaps"]
    var caps = []
    var i = 0
    while (i < arr.length) {
        caps.push(Math.trunc(arr[i]))
        i = i + 1
    }
    var begin = 0
    var end = 0
    if (caps.length >= 2) {
        begin = caps[0]
        end = caps[1]
    }
    return {re: re, text: text, m: {begin: begin, end: end, caps: caps}}
}
