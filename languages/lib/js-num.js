// js-num.js -- JavaScript's reading of a STRING as a number: StringToNumber (the
// ToNumber of a string, ES2023 7.1.4.1.1) and the global parseFloat (19.2.4).
//
// Included by both js and typescript interpreters (include("lib/js-num.js")) and
// imported by languages/lib/js-rt.metajs, so the interpreters and layer 2 run one
// implementation; the Go twin is jsStrToNum / jsParseFloatJS in abnf/jsrtjsbig.go.
// Written in the restricted subset of regex.js's head comment. Every name is
// prefixed jsn.
//
// The host's own conversions are not JavaScript's: they disagree on "0x10" (16),
// "1_0" (NaN), "inf" (NaN), "Infinity", the Unicode white space, and between goja
// and -frozen. Only a VALIDATED decimal literal is handed to the host's parseFloat,
// which every host reads correctly.

// WhiteSpace and LineTerminator (ES2023 12.2, 12.3): what StringToNumber trims and
// parseFloat skips.
function jsnIsWs(c) {
    if (c == 32 || (c >= 9 && c <= 13) || c == 160 || c == 5760 || c == 8232 || c == 8233) { return true }
    if (c >= 8192 && c <= 8202) { return true }
    return c == 8239 || c == 8287 || c == 12288 || c == 65279
}

function jsnDigitVal(c) {
    if (c >= 48 && c <= 57) { return c - 48 }
    if (c >= 97 && c <= 122) { return c - 87 }
    if (c >= 65 && c <= 90) { return c - 55 }
    return 99
}

// The length of the longest StrUnsignedDecimalLiteral at s[i..]: digits with an
// optional fraction (at least one digit in all), then an optional exponent with at
// least one digit; "Infinity" too. 0 when there is none.
function jsnDecLen(s, i) {
    if (s.substring(i, i + 8) == "Infinity") { return 8 }
    var j = i
    var nd = 0
    while (j < s.length && jsnDigitVal(s.charCodeAt(j)) < 10) { j = j + 1; nd = nd + 1 }
    if (j < s.length && s.charCodeAt(j) == 46) {
        var k = j + 1
        while (k < s.length && jsnDigitVal(s.charCodeAt(k)) < 10) { k = k + 1; nd = nd + 1 }
        if (nd == 0) { return 0 }
        j = k
    }
    if (nd == 0) { return 0 }
    if (j < s.length && (s.charCodeAt(j) == 101 || s.charCodeAt(j) == 69)) {
        var e = j + 1
        if (e < s.length && (s.charCodeAt(e) == 43 || s.charCodeAt(e) == 45)) { e = e + 1 }
        var ed = e
        while (ed < s.length && jsnDigitVal(s.charCodeAt(ed)) < 10) { ed = ed + 1 }
        if (ed > e) { j = ed }
    }
    return j - i
}

// The value of a validated decimal literal t with its sign.
function jsnDecVal(t) {
    var neg = t.charAt(0) == "-"
    var body = t
    if (t.charAt(0) == "-" || t.charAt(0) == "+") { body = t.substring(1) }
    var v = 0
    if (body == "Infinity") { v = Infinity } else { v = parseFloat(body) }
    if (neg) { return -v }
    return v
}

// StringToNumber: white space trimmed at both ends, "" is 0, then exactly one of a
// signed decimal literal (or Infinity) or an UNSIGNED 0x / 0o / 0b integer.
function jsnStrToNum(s) {
    var i = 0
    var j = s.length
    while (i < j && jsnIsWs(s.charCodeAt(i))) { i = i + 1 }
    while (j > i && jsnIsWs(s.charCodeAt(j - 1))) { j = j - 1 }
    if (i == j) { return 0 }
    var t = s.substring(i, j)
    if (t.length > 2 && t.charAt(0) == "0") {
        var p = t.charAt(1)
        var radix = 0
        if (p == "x" || p == "X") { radix = 16 }
        else if (p == "o" || p == "O") { radix = 8 }
        else if (p == "b" || p == "B") { radix = 2 }
        if (radix > 0) {
            var v = 0
            for (var k = 2; k < t.length; k++) {
                var d = jsnDigitVal(t.charCodeAt(k))
                if (d >= radix) { return NaN }
                v = v * radix + d
            }
            return v
        }
    }
    var at = 0
    if (t.charAt(0) == "+" || t.charAt(0) == "-") { at = 1 }
    var n = jsnDecLen(t, at)
    if (n == 0 || at + n != t.length) { return NaN }
    return jsnDecVal(t)
}

// parseFloat(string): leading white space skipped, then the longest prefix that is
// a signed decimal literal or Infinity; NaN when there is none.
function jsnParseFloat(s) {
    var i = 0
    while (i < s.length && jsnIsWs(s.charCodeAt(i))) { i = i + 1 }
    var at = i
    if (at < s.length && (s.charAt(at) == "+" || s.charAt(at) == "-")) { at = at + 1 }
    var n = jsnDecLen(s, at)
    if (n == 0) { return NaN }
    return jsnDecVal(s.substring(i, at + n))
}
