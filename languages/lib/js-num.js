// js-num.js -- JavaScript's reading of a STRING as a number: StringToNumber (the
// ToNumber of a string, ES2023 7.1.4.1.1) and the globals parseFloat and parseInt
// (19.2.4, 19.2.5); the writing of one by Number#toFixed, toExponential,
// toPrecision and toLocaleString; and the code-unit order of two strings, which relations and the
// default sort use.
//
// Included by both js and typescript interpreters (include("lib/js-num.js")) and
// imported by languages/lib/js-rt.metajs, so the interpreters and layer 2 run one
// implementation; the Go twins are jsStrToNum / jsParseFloatJS in abnf/jsrtjsbig.go
// and jsToFixedJS / jsToExponentialJS / jsToPrecisionJS / jsToLocaleJS in
// abnf/jsrtjsprint.go.
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

// parseInt(string, radix) (ES2023 19.2.5), r being ToInt32(radix): leading white
// space and one sign skipped, r 0 means 10 and r outside 2..36 NaN, an 0x / 0X
// prefix read when r is 0 or 16, then the longest run of digits below r; NaN when
// there is none. A sign applies to a zero too ("-0" is -0). The value is exact for
// base ten (the decimal literal path) and a power-of-two base; in another base the
// specification leaves a value past 2^53 to the implementation, and here it is V8's
// (jsnIntGeneric).
function jsnParseInt(s, r) {
    var i = 0
    while (i < s.length && jsnIsWs(s.charCodeAt(i))) { i = i + 1 }
    var neg = false
    if (i < s.length && (s.charAt(i) == "+" || s.charAt(i) == "-")) {
        neg = s.charAt(i) == "-"
        i = i + 1
    }
    var radix = r
    var prefix = true
    if (radix != 0) {
        if (radix < 2 || radix > 36) { return NaN }
        if (radix != 16) { prefix = false }
    } else { radix = 10 }
    if (prefix && i + 1 < s.length && s.charAt(i) == "0" && (s.charAt(i + 1) == "x" || s.charAt(i + 1) == "X")) {
        i = i + 2
        radix = 16
    }
    var start = i
    while (i < s.length && jsnDigitVal(s.charCodeAt(i)) < radix) { i = i + 1 }
    if (i == start) { return NaN }
    var v = 0
    if (radix == 10) { v = jsnDecVal(s.substring(start, i)) }
    else if (radix == 2 || radix == 4 || radix == 8 || radix == 16 || radix == 32) { v = jsnIntPow2(s, start, i, radix) }
    else { v = jsnIntGeneric(s, start, i, radix) }
    if (neg) { return -v }
    return v
}

// The digits s[a..b) in a radix that is neither 10 nor a power of two, read as V8
// reads them: in chunks whose multiplier radix^k stays at most 0xFFFFFFFF / 36, each
// folded into the value as ONE fused multiply-add, v * multiplier + chunk rounded
// once - what node computes on arm64. Leading zeros are skipped first, so they do not
// shift the chunks.
function jsnIntGeneric(s, a, b, radix) {
    var v = 0
    var k = a
    while (k < b && s.charCodeAt(k) == 48) { k = k + 1 }
    while (k < b) {
        var part = 0
        var mult = 1
        while (k < b && mult * radix <= 119304647) {
            part = part * radix + jsnDigitVal(s.charCodeAt(k))
            mult = mult * radix
            k = k + 1
        }
        v = jsnFma(v, mult, part)
    }
    return v
}

// v * mult + part rounded once, half to even, for an integral v >= 0 and
// 0 <= part < mult < 2^27. Below 2^53 the product is exact and the sum rounds once;
// above, the exact sum is built in base 16 (every step exact in a double) and
// rounded by jsnIntPow2.
function jsnFma(v, mult, part) {
    if (v * mult < 9007199254740992) { return v * mult + part }
    if (v - v !== 0) { return v }
    var d = []
    var x = v
    while (x > 0) {
        var r = x % 16
        d.push(r)
        x = (x - r) / 16
    }
    var carry = part
    for (var j = 0; j < d.length; j++) {
        var t = d[j] * mult + carry
        var lo = t % 16
        d[j] = lo
        carry = (t - lo) / 16
    }
    while (carry > 0) {
        var r2 = carry % 16
        d.push(r2)
        carry = (carry - r2) / 16
    }
    var h = ""
    for (var q = d.length - 1; q >= 0; q--) { h = h + "0123456789abcdef".charAt(d[q]) }
    return jsnIntPow2(h, 0, h.length, 16)
}

// The digits s[a..b) in a power-of-two base, rounded once, half to even: the first
// 53 significant bits, then the first dropped bit and whether any later one is set.
function jsnIntPow2(s, a, b, radix) {
    var v = 0
    var nb = 0
    var exp = 0
    var rnd = 0
    var sticky = false
    for (var i = a; i < b; i++) {
        var d = jsnDigitVal(s.charCodeAt(i))
        for (var p = radix / 2; p >= 1; p = p / 2) {
            var bit = 0
            if (d >= p) { bit = 1; d = d - p }
            if (nb < 53) {
                if (nb > 0 || bit == 1) { v = v * 2 + bit; nb = nb + 1 }
            } else {
                if (exp == 0) { rnd = bit } else if (bit == 1) { sticky = true }
                exp = exp + 1
            }
        }
    }
    if (rnd == 1 && (sticky || v % 2 == 1)) { v = v + 1 }
    if (exp > 0) { v = v * Math.pow(2, exp) }
    return v
}

// Number#toString(radix) for a radix other than 10, which ECMA-262 leaves to the
// implementation: V8's DoubleToRadixCString. The fraction gets digits only while the
// remainder is at least delta, half the gap to the next double, and the last digit is
// rounded half to even, carrying back through the digits already written; the
// integer part is exact while it is below 2^53 times the radix and zero-filled above.
// Twin of jsvRadixString in abnf/jsrtjsprint.go.
function jsnRadixStr(x, radix) {
    if (x !== x) { return "NaN" }
    if (x - x !== 0) {
        if (x > 0) { return "Infinity" }
        return "-Infinity"
    }
    if (x == 0) { return "0" }
    var digits = "0123456789abcdefghijklmnopqrstuvwxyz"
    var v = x
    if (v < 0) { v = -v }
    var integer = Math.floor(v)
    var fraction = v - integer
    var delta = jsnHalfGap(v)
    var fd = []
    if (fraction >= delta) {
        while (true) {
            fraction = fraction * radix
            delta = delta * radix
            var digit = Math.floor(fraction)
            fd.push(digit)
            fraction = fraction - digit
            if ((fraction > 0.5 || (fraction == 0.5 && digit % 2 == 1)) && fraction + delta > 1) {
                while (true) {
                    if (fd.length == 0) {
                        integer = integer + 1
                        break
                    }
                    var last = fd.pop()
                    if (last + 1 < radix) {
                        fd.push(last + 1)
                        break
                    }
                }
                break
            }
            if (fraction < delta) { break }
        }
    }
    var out = ""
    while (integer / radix >= 9007199254740992) {
        integer = integer / radix
        out = out + "0"
    }
    while (true) {
        var rem = integer % radix
        out = digits.charAt(rem) + out
        integer = (integer - rem) / radix
        if (integer <= 0) { break }
    }
    if (fd.length > 0) {
        out = out + "."
        for (var i = 0; i < fd.length; i++) { out = out + digits.charAt(fd[i]) }
    }
    if (x < 0) { return "-" + out }
    return out
}

// One code point of String.fromCodePoint (ES2023 22.1.2.2) as UTF-16 text, or ""
// when n is not an integral code point, for which the caller raises node's
// "RangeError: Invalid code point <n>".
function jsnCodePoint(n) {
    if (n !== n || n - n !== 0 || n != Math.floor(n) || n < 0 || n > 1114111) { return "" }
    if (n < 65536) { return String.fromCharCode(n) }
    var c = n - 65536
    return String.fromCharCode(55296 + Math.floor(c / 1024)) + String.fromCharCode(56320 + c % 1024)
}

// Half the gap from a positive finite v to the next double up, and at least the
// smallest subnormal: 2^(e-53) for 2^e <= v < 2^(e+1), found by exact scaling.
function jsnHalfGap(v) {
    var p = 1
    while (p > v) { p = p / 2 }
    while (p * 2 <= v) { p = p * 2 }
    var d = p / 9007199254740992
    if (d == 0) {
        d = p
        while (d / 2 > 0) { d = d / 2 }
    }
    return d
}

// JavaScript's order of two strings, -1, 0 or 1: by UTF-16 code units (ECMA-262
// IsLessThan), where both hosts' own `<` - and the floor's - is the code point
// order, which puts an astral character above U+E000..U+FFFF. The two orders agree
// whenever one side is ASCII, which the byte count tells. Twin of jsvStrCmp in
// abnf/jsrtjsprint.go.
function jsnStrCmp(a, b) {
    if (a === b) { return 0 }
    if (byteLen(a) == a.length || byteLen(b) == b.length) {
        if (a < b) { return -1 }
        return 1
    }
    var n = a.length
    if (b.length < n) { n = b.length }
    for (var i = 0; i < n; i++) {
        var d = a.charCodeAt(i) - b.charCodeAt(i)
        if (d < 0) { return -1 }
        if (d > 0) { return 1 }
    }
    if (a.length < b.length) { return -1 }
    return 1
}

// ----- Number#toFixed / toExponential / toPrecision (ES2023 21.1.3) -----
//
// Each rounds the EXACT binary value, and of two equally close candidates takes the
// larger: round half UP on the exact decimal expansion, which jsnExact writes out.
// A RangeError goes through the includer's jsnHRaise(message). The digit count has
// already been through ToIntegerOrInfinity (jsnToInt).

function jsnToInt(n) {
    if (n !== n) { return 0 }
    if (n - n !== 0) { return n }
    if (n < 0) { return -Math.floor(-n) }
    return Math.floor(n)
}

// Little-endian base-10000 limbs, multiplied in place by a small factor.
function jsnMulSmall(limbs, k) {
    var carry = 0
    for (var i = 0; i < limbs.length; i++) {
        var v = limbs[i] * k + carry
        carry = Math.floor(v / 10000)
        limbs[i] = v - carry * 10000
    }
    while (carry > 0) {
        var q = Math.floor(carry / 10000)
        limbs.push(carry - q * 10000)
        carry = q
    }
}

function jsnLimbStr(limbs) {
    var out = "" + limbs[limbs.length - 1]
    for (var i = limbs.length - 2; i >= 0; i--) {
        var part = "" + limbs[i]
        while (part.length < 4) { part = "0" + part }
        out = out + part
    }
    return out
}

// The exact decimal of a positive finite double as {i: integer digits ("0" when
// there are none), f: fraction digits}: a = m * 2^e with m an integer below 2^53,
// and 2^-n is 5^n / 10^n.
function jsnExact(a) {
    var m = a
    var e = 0
    while (m != Math.floor(m)) {
        m = m * 2
        e = e - 1
    }
    while (m > 9007199254740992) {
        m = m / 2
        e = e + 1
    }
    var limbs = []
    var x = m
    while (x > 0) {
        var q = Math.floor(x / 10000)
        limbs.push(x - q * 10000)
        x = q
    }
    if (limbs.length == 0) { limbs.push(0) }
    if (e >= 0) {
        for (var k = 0; k < e; k++) { jsnMulSmall(limbs, 2) }
        return {i: jsnLimbStr(limbs), f: ""}
    }
    var n = -e
    for (var j = 0; j < n; j++) { jsnMulSmall(limbs, 5) }
    var all = jsnLimbStr(limbs)
    while (all.length <= n) { all = "0" + all }
    return {i: all.substring(0, all.length - n), f: all.substring(all.length - n)}
}

// The first `keep` digits of the digit run d, rounded half up; one digit longer when
// the rounding carried out of the first.
function jsnRoundUp(d, keep) {
    var head = d
    if (keep < d.length) { head = d.substring(0, keep) }
    while (head.length < keep) { head = head + "0" }
    if (keep >= d.length || d.charCodeAt(keep) < 53) { return head }
    var out = ""
    var i = head.length - 1
    while (i >= 0 && head.charAt(i) == "9") {
        out = "0" + out
        i = i - 1
    }
    if (i < 0) { return "1" + out }
    return head.substring(0, i) + String.fromCharCode(head.charCodeAt(i) + 1) + out
}

// k significant digits of a positive finite double, rounded half up, and the decimal
// exponent of the first: {d, e}.
function jsnSig(a, k) {
    var x = jsnExact(a)
    var all = x.i + x.f
    var z = 0
    while (all.charAt(z) == "0") { z = z + 1 }
    var e = x.i.length - z - 1
    var d = jsnRoundUp(all.substring(z), k)
    if (d.length > k) {
        d = d.substring(0, k)
        e = e + 1
    }
    return {d: d, e: e}
}

// The SHORTEST digits that read back as a, which is what Number::toString prints,
// with the exponent of the first: {d, e}.
function jsnShortest(a) {
    var s = "" + a
    var ex = 0
    var at = s.indexOf("e")
    if (at >= 0) {
        var es = s.substring(at + 1)
        if (es.charAt(0) == "+") { es = es.substring(1) }
        ex = parseInt(es, 10)
        s = s.substring(0, at)
    }
    var p = s.indexOf(".")
    var ds = s
    if (p >= 0) { ds = s.substring(0, p) + s.substring(p + 1) } else { p = s.length }
    var z = 0
    while (ds.charAt(z) == "0") { z = z + 1 }
    var d = ds.substring(z)
    while (d.length > 1 && d.charAt(d.length - 1) == "0") { d = d.substring(0, d.length - 1) }
    return {d: d, e: ex + p - z - 1}
}

function jsnZeros(n) {
    var out = ""
    while (out.length < n) { out = out + "0" }
    return out
}

function jsnExpForm(d, e) {
    var m = d.charAt(0)
    if (d.length > 1) { m = m + "." + d.substring(1) }
    if (e >= 0) { return m + "e+" + e }
    return m + "e-" + (-e)
}

function jsnToFixed(x, f) {
    if (f - f !== 0 || f < 0 || f > 100) { jsnHRaise("RangeError: toFixed() digits argument must be between 0 and 100") }
    if (x !== x) { return "NaN" }
    var big = Math.pow(10, 21)
    if (x >= big || x <= -big) { return "" + x }
    var sign = ""
    var a = x
    if (x < 0) {
        sign = "-"
        a = -x
    }
    var ip = "0"
    var fp = jsnZeros(f)
    if (a != 0) {
        var ex = jsnExact(a)
        var r = jsnRoundUp(ex.i + ex.f, ex.i.length + f)
        ip = r.substring(0, r.length - f)
        fp = r.substring(r.length - f)
    }
    if (f == 0) { return sign + ip }
    return sign + ip + "." + fp
}

function jsnToExponential(x, f, fUndef) {
    if (x !== x || x - x !== 0) { return "" + x }
    if (f - f !== 0 || f < 0 || f > 100) { jsnHRaise("RangeError: toExponential() argument must be between 0 and 100") }
    var sign = ""
    var a = x
    if (x < 0) {
        sign = "-"
        a = -x
    }
    if (a == 0) {
        if (fUndef) { return sign + "0e+0" }
        return sign + jsnExpForm(jsnZeros(f + 1), 0)
    }
    var r = anytype
    if (fUndef) { r = jsnShortest(a) } else { r = jsnSig(a, f + 1) }
    return sign + jsnExpForm(r.d, r.e)
}

function jsnToPrecision(x, p) {
    if (x !== x || x - x !== 0) { return "" + x }
    if (p - p !== 0 || p < 1 || p > 100) { jsnHRaise("RangeError: toPrecision() argument must be between 1 and 100") }
    var sign = ""
    var a = x
    if (x < 0) {
        sign = "-"
        a = -x
    }
    var d = jsnZeros(p)
    var e = 0
    if (a != 0) {
        var r = jsnSig(a, p)
        d = r.d
        e = r.e
    }
    if (e < -6 || e >= p) { return sign + jsnExpForm(d, e) }
    if (e == p - 1) { return sign + d }
    if (e >= 0) { return sign + d.substring(0, e + 1) + "." + d.substring(e + 1) }
    return sign + "0." + jsnZeros(-(e + 1)) + d
}

// An integer's digit run, signed or not, grouped by threes with ",", as en-US
// writes it - also BigInt#toLocaleString.
function jsnGroup(ip) {
    var sign = ""
    if (ip.charAt(0) == "-") {
        sign = "-"
        ip = ip.substring(1)
    }
    var out = ""
    var i = ip.length
    while (i > 3) {
        out = "," + ip.substring(i - 3, i) + out
        i = i - 3
    }
    return sign + ip.substring(0, i) + out
}

// Number#toLocaleString() in node's default locale (en-US, ECMA-402
// Intl.NumberFormat): the shortest digits rounded half away from zero to at most
// three fraction digits, trailing zeros dropped, the integer part grouped. The sign
// of a negative value survives a rounding to zero ("-0"); an infinity is the sign
// and U+221E. A value below 1 is written from a leading "0" digit, which also
// absorbs a carry (0.9995 is "1").
function jsnToLocale(x) {
    if (x !== x) { return "NaN" }
    var sign = ""
    var a = x
    if (x < 0 || 1 / x < 0) {
        sign = "-"
        a = -x
    }
    if (a - a !== 0) { return sign + String.fromCharCode(8734) }
    if (a == 0) { return sign + "0" }
    var r = jsnShortest(a)
    var digits = r.d
    var il = r.e + 1
    if (r.e < 0) {
        digits = jsnZeros(-r.e) + r.d
        il = 1
    }
    var rd = jsnRoundUp(digits, il + 3)
    if (rd.length > il + 3) { il = il + 1 }
    var fp = rd.substring(il)
    while (fp.length > 0 && fp.charAt(fp.length - 1) == "0") { fp = fp.substring(0, fp.length - 1) }
    if (fp.length > 0) { return sign + jsnGroup(rd.substring(0, il)) + "." + fp }
    return sign + jsnGroup(rd.substring(0, il))
}
