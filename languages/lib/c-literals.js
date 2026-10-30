// c-literals.js -- what c-interpreter.abnf and c-to-llvm-ir.abnf share (include):
// the values of C string literals, character constants and the standard headers'
// constants, brace-initializer items, and the block-scope name table. digVal,
// fail and hasOwn come from the including grammar.
//
// The execution character set is UTF-8, as cc's: a narrow literal holds the
// UTF-8 bytes of every character outside ASCII and of every \u / \U universal
// character name (C11 6.4.3), and a wide one (L, u, U) one element per code point.

// The code point that starts at text[i]: a UTF-16 surrogate pair joins into one.
function cCodePointAt(text, i) {
    var hi = text.charCodeAt(i)
    if (hi >= 55296 && hi <= 56319 && i + 1 < text.length) {
        var lo = text.charCodeAt(i + 1)
        if (lo >= 56320 && lo <= 57343) return (hi - 55296) * 1024 + (lo - 56320) + 65536
    }
    return hi
}
// Append one code point: its UTF-8 bytes to a narrow literal, itself to a wide one.
function cPushCodePoint(bytes, cp, wide) {
    if (wide || cp < 128) { bytes.push(cp); return }
    if (cp < 2048) {
        bytes.push(192 + (cp >> 6))
    } else if (cp < 65536) {
        bytes.push(224 + (cp >> 12))
        bytes.push(128 + ((cp >> 6) & 63))
    } else {
        bytes.push(240 + (cp >> 18))
        bytes.push(128 + ((cp >> 12) & 63))
        bytes.push(128 + ((cp >> 6) & 63))
    }
    bytes.push(128 + (cp & 63))
}
// The bytes of a string-literal RUN (adjacent literals concatenate, C11 6.4.5p5),
// with the C escape sequences decoded and the terminating null appended. An
// L / u / U prefix makes the elements wide.
function cStrBytes(text) {
    var bytes = []
    var wide = false
    var i = 0
    while (i < text.length) {
        var cc = text.charCodeAt(i)
        if (cc == 117 && i + 1 < text.length && text.charCodeAt(i + 1) == 56) { i = i + 2; continue }
        if (cc == 76 || cc == 117 || cc == 85) { wide = true; i = i + 1; continue }
        if (cc != 34) { i = i + 1; continue }
        i = i + 1
        while (i < text.length && text.charCodeAt(i) != 34) {
            if (text.charCodeAt(i) != 92) {
                var cp = cCodePointAt(text, i)
                cPushCodePoint(bytes, cp, wide)
                i = i + (cp > 65535 ? 2 : 1)
                continue
            }
            var j = i + 1
            var e = text.charCodeAt(j)
            if (e == 120 || e == 88) {
                j = j + 1
                while (j < text.length && digVal(text.charCodeAt(j)) >= 0) j = j + 1
            } else if (e >= 48 && e <= 55) {
                j = j + 1
                var n = 1
                while (j < text.length && n < 3 && text.charCodeAt(j) >= 48 && text.charCodeAt(j) <= 55) { j = j + 1; n = n + 1 }
            } else if (e == 117) {
                j = j + 5
            } else if (e == 85) {
                j = j + 9
            } else {
                j = j + 1
            }
            if (e == 117 || e == 85) { cPushCodePoint(bytes, cCharCode(text.slice(i, j)), wide) }
            else { bytes.push(cCharCode(text.slice(i, j))) }
            i = j
        }
        i = i + 1
    }
    bytes.push(0)
    return {str: bytes, wide: wide}
}
// The value of one character-constant body, including the C escape sequences:
// backslash x with hex digits, backslash with up to three octal digits, the
// universal character names \u and \U (a code point), and the simple escapes. A
// byte is answered unsigned; cCharConst gives it the type of a constant.
function cCharCode(s) {
    if (s.charCodeAt(0) != 92) return s.charCodeAt(0)
    var e = s.charCodeAt(1)
    if (e == 120 || e == 88 || e == 117 || e == 85) {
        var hv = 0
        for (var i = 2; i < s.length; i++) hv = hv * 16 + digVal(s.charCodeAt(i))
        return (e == 117 || e == 85) ? hv : (hv & 255)
    }
    if (e >= 48 && e <= 55) {
        var ov = 0
        for (var j = 1; j < s.length; j++) ov = ov * 8 + digVal(s.charCodeAt(j))
        return ov & 255
    }
    if (e == 110) return 10
    if (e == 116) return 9
    if (e == 114) return 13
    if (e == 118) return 11
    if (e == 102) return 12
    if (e == 97) return 7
    if (e == 98) return 8
    if (e == 101) return 27
    return e   // backslash, single quote, double quote, question mark: itself.
}
// An integer character constant is an int holding the value of a char (C11
// 6.4.4.4p10), and char is signed on this target: '\377' is -1.
function cCharConst(s) {
    var v = cCharCode(s)
    return v > 127 ? v - 256 : v
}

// The constants of the standard headers this front end treats as built in
// (core.stdlibImports), spelled as the reference toolchain's headers spell them on
// this target (LP64, signed char) so each one gets the C type that spelling gives
// it: INT_MIN is an int, UINT_MAX an unsigned int, ULONG_MAX an unsigned long. A
// leading '-' negates and a trailing '-1' subtracts one, the only two shapes used.
var cStdConsts = {
    "limits.h": {CHAR_BIT: "8", SCHAR_MIN: "-128", SCHAR_MAX: "127", UCHAR_MAX: "255",
        CHAR_MIN: "-128", CHAR_MAX: "127", SHRT_MIN: "-32768", SHRT_MAX: "32767",
        USHRT_MAX: "65535", INT_MIN: "-2147483647-1", INT_MAX: "2147483647",
        UINT_MAX: "0xffffffff", LONG_MIN: "-0x7fffffffffffffffL-1",
        LONG_MAX: "0x7fffffffffffffffL", ULONG_MAX: "0xffffffffffffffffUL",
        LLONG_MIN: "-0x7fffffffffffffffLL-1", LLONG_MAX: "0x7fffffffffffffffLL",
        ULLONG_MAX: "0xffffffffffffffffULL"},
    "stdint.h": {INT8_MIN: "-128", INT8_MAX: "127", UINT8_MAX: "255",
        INT16_MIN: "-32768", INT16_MAX: "32767", UINT16_MAX: "65535",
        INT32_MIN: "-2147483647-1", INT32_MAX: "2147483647", UINT32_MAX: "4294967295U",
        INT64_MIN: "-9223372036854775807LL-1", INT64_MAX: "9223372036854775807LL",
        UINT64_MAX: "18446744073709551615ULL", SIZE_MAX: "18446744073709551615UL",
        INTPTR_MIN: "-9223372036854775807L-1", INTPTR_MAX: "9223372036854775807L",
        UINTPTR_MAX: "18446744073709551615UL", INTMAX_MIN: "-9223372036854775807L-1",
        INTMAX_MAX: "9223372036854775807L", UINTMAX_MAX: "18446744073709551615UL",
        PTRDIFF_MIN: "-9223372036854775807L-1", PTRDIFF_MAX: "9223372036854775807L"},
    "stdbool.h": {"true": "1", "false": "0", __bool_true_false_are_defined: "1"}
}
// The constant tables of the headers included so far.
var cStdIncluded = []
// '#include <h>': its constants become visible, and <stdbool.h> defines bool as
// _Bool (C11 7.18), from here on like any typedef.
function cNoteSysHeader(h) {
    if (hasOwn(cStdConsts, h)) cStdIncluded.push(cStdConsts[h])
    if (h == "stdbool.h") typedefs["bool"] = ctBool
}
// The constant thunk for a name an included standard header defines, or undefined.
// makeNumLit, makeNeg and foldBinary come from the including grammar.
function cStdConst(name) {
    for (var i = 0; i < cStdIncluded.length; i++) {
        if (!hasOwn(cStdIncluded[i], name)) continue
        var s = cStdIncluded[i][name]
        if (s.charCodeAt(0) != 45) return makeNumLit(parseCInt(s), s)
        var parts = s.slice(1).split("-")
        var t = makeNeg(makeNumLit(parseCInt(parts[0]), parts[0]))
        if (parts.length == 1) return t
        return foldBinary([t, "-", makeNumLit(parseCInt(parts[1]), parts[1])])
    }
    return undefined
}

// One brace-initializer item: an optional designator chain ('[i]', '.field',
// '.b[1]') plus its value. The first designator picks the member of the list's
// own object (at / fld); the rest (sub) walk into that member.
function makeDItem(items) {
    var it = {v: items[items.length - 1]}
    if (items.length > 1) {
        if (items[0].fld != undefined) { it.fld = items[0].fld } else { it.at = items[0].at }
        if (items.length > 2) it.sub = items.slice(1, items.length - 1)
    }
    return it
}
// An undesignated item after a chained designator continues INSIDE the member the
// chain walked into (C11 6.7.9p17-20), which is brace elision - not supported, so
// refuse it instead of filling the next member.
function noElisionAfter(list, i) {
    if (i > 0 && list[i].at == undefined && list[i].fld == undefined && list[i - 1].sub != undefined) {
        fail("an initializer after a nested designator (brace elision) is not supported")
    }
}

// The names declared so far in each enclosing block, innermost last. The tags
// fill it while they walk the tree, so a second declaration of one name in one
// block (C11 6.7p3) is a compile error even in code that never runs; a nested
// block opens a table of its own, which is what keeps shadowing legal. Only two
// extern declarations may repeat a name, since both denote the one object with
// linkage (6.7p3).
var blockNames = []
function openBlockScope() { blockNames.push({}) }
function closeBlockScope() { blockNames.pop() }
function noteBlockName(name, ext) {
    if (blockNames.length == 0) { return }
    var top = blockNames[blockNames.length - 1]
    if (hasOwn(top, name) && !(ext && top[name] == "ext")) fail("redefinition of '" + name + "'")
    top[name] = ext ? "ext" : "obj"
}
// Whether a declaration's keyword run ('extern int', 'const extern long') holds
// the extern storage class. up.in arrives with the separators removed, so the
// run is read keyword by keyword and the scan stops at anything else.
var declKws = ["extern", "const", "volatile", "unsigned", "signed", "short", "long",
               "int", "char", "double", "float", "_Bool", "void"]
function isExternTy(text) {
    var i = 0
    while (i < text.length) {
        var hit = ""
        for (var k = 0; k < declKws.length; k++) {
            if (hit == "" && text.slice(i, i + declKws[k].length) == declKws[k]) { hit = declKws[k] }
        }
        if (hit == "") { return false }
        if (hit == "extern") { return true }
        i = i + hit.length
    }
    return false
}
