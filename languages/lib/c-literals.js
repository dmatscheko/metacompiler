// c-literals.js -- the values of C string literals and character constants, for
// c-interpreter.abnf and c-to-llvm-ir.abnf (include). digVal comes from the
// including grammar.
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
