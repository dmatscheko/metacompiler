// java-strings.js -- java.lang.String's methods and String.format over plain
// strings, for java-interpreter.abnf (include) and languages/lib/java-rt.metajs
// (import), both of which also include lib/regex.js. abnf/jsrtjavastr.go is the
// Go twin's port; the three must answer alike.
//
// Every function here takes and answers plain strings, numbers and arrays of
// strings. What depends on the value model is asked through hooks each importer
// defines:
//   jxkKind(v)       "null" "bool" "char" "int" "float" "str" or "obj"
//   jxkStr(v)        String.valueOf(v)
//   jxkCode(v)       a char's code, or an int's value
//   jxkDec(v)        an integral value's signed decimal digits
//   jxkRadix(v, r)   its two's complement digits in radix r at its own width
//   jxkDbl(v)        a float's value as a double
//   jxkDblStr(a)     Double.toString(a) of a double a > 0
//   jxkClass(v)      v.getClass().getName() of a non-null v
//   jxkHash(v)       v.hashCode() of a non-null v
//   jxkThrow(c, msg) throw a new built-in exception of simple class name c with
//                    message msg
//
// It also holds Integer's unsigned and bit members over plain int values, and
// Math.ulp, Math.IEEEremainder and Math.multiplyHigh.
//   jxkFail(msg)     a format conversion this subset does not support
//
// Case mapping and the literal search are lib/utf16-strings.js's, which both
// importers include first.

// Character.isWhitespace.
function jxIsWs(c) {
    if ((c >= 9 && c <= 13) || (c >= 28 && c <= 32)) { return true }
    if (c == 5760 || (c >= 8192 && c <= 8198) || (c >= 8200 && c <= 8202)) { return true }
    return c == 8232 || c == 8233 || c == 8287 || c == 12288
}
// trim() drops every unit <= ' ' at both ends; strip() drops whitespace.
function jxTrim(s) {
    var b = 0
    var e = s.length
    while (b < e && s.charCodeAt(b) <= 32) { b = b + 1 }
    while (e > b && s.charCodeAt(e - 1) <= 32) { e = e - 1 }
    return s.substring(b, e)
}
function jxStrip(s, lead, trail) {
    var b = 0
    var e = s.length
    if (lead) { while (b < e && jxIsWs(s.charCodeAt(b))) { b = b + 1 } }
    if (trail) { while (e > b && jxIsWs(s.charCodeAt(e - 1))) { e = e - 1 } }
    return s.substring(b, e)
}
function jxIsBlank(s) { return jxStrip(s, true, false).length == 0 }
// compareTo: the first differing unit's difference, else the length difference.
function jxCompare(a, b) {
    var n = a.length
    if (b.length < n) { n = b.length }
    var i = 0
    while (i < n) {
        var d = a.charCodeAt(i) - b.charCodeAt(i)
        if (d != 0) { return d }
        i = i + 1
    }
    return a.length - b.length
}
// compareToIgnoreCase / equalsIgnoreCase fold each unit to upper case and then
// to lower case, as java.lang.String does; where the units differ inside a
// surrogate pair the whole code points are compared (JDK 16 and later). A
// supplementary code point folds to itself here.
function jxFold(c) { return jxChLo(jxChUp(c)) }
function jxCompareIC(a, b) {
    var n = a.length
    if (b.length < n) { n = b.length }
    var i = 0
    while (i < n) {
        var x = a.charCodeAt(i)
        var y = b.charCodeAt(i)
        if (x != y) {
            x = jxFold(x)
            y = jxFold(y)
            if (x != y) {
                var cx = jxCpAt(a, i)
                var cy = jxCpAt(b, i)
                if (cx > 65535 || cy > 65535) {
                    if (cx < 65536) { cx = jxFold(cx) }
                    if (cy < 65536) { cy = jxFold(cy) }
                    if (cx != cy) { return cx - cy }
                } else {
                    return x - y
                }
            }
        }
        i = i + 1
    }
    return a.length - b.length
}
// The code point the unit at i belongs to: a surrogate pair's whole value
// whichever half i is, the unit itself otherwise.
function jxCpAt(s, i) {
    var c = s.charCodeAt(i)
    if (c >= 55296 && c < 56320 && i + 1 < s.length) {
        var d = s.charCodeAt(i + 1)
        if (d >= 56320 && d < 57344) { return 65536 + (c - 55296) * 1024 + (d - 56320) }
    }
    if (c >= 56320 && c < 57344 && i > 0) {
        var h = s.charCodeAt(i - 1)
        if (h >= 55296 && h < 56320) { return 65536 + (h - 55296) * 1024 + (c - 56320) }
    }
    return c
}
// String.chars and codePoints as int arrays (a surrogate pair is one code
// point), and lines' parts: split at \n, \r and \r\n, with no empty last line.
function jxChars(s) {
    var out = []
    for (var i = 0; i < s.length; i++) { out.push(s.charCodeAt(i)) }
    return out
}
function jxCodePoints(s) {
    var out = []
    var i = 0
    while (i < s.length) {
        var cp = jxCpAt(s, i)
        out.push(cp)
        i = i + (cp > 65535 ? 2 : 1)
    }
    return out
}
function jxLines(s) {
    var out = []
    var start = 0
    var i = 0
    while (i < s.length) {
        var c = s.charAt(i)
        if (c == "\n" || c == "\r") {
            out.push(s.substring(start, i))
            if (c == "\r" && i + 1 < s.length && s.charAt(i + 1) == "\n") { i = i + 1 }
            start = i + 1
        }
        i = i + 1
    }
    if (start < s.length) { out.push(s.substring(start, s.length)) }
    return out
}
// The index of s's first non-whitespace unit (Character.isWhitespace), and
// the end of its last one; s.length and 0 for a blank s.
function jxFirstNonWs(s) {
    var i = 0
    while (i < s.length && jxIsWs(s.charCodeAt(i))) { i = i + 1 }
    return i
}
function jxEndNonWs(s) {
    var e = s.length
    while (e > 0 && jxIsWs(s.charCodeAt(e - 1))) { e = e - 1 }
    return e
}
// indent(n): every line of lines() with n spaces put in front, or up to -n of
// its leading whitespace taken off, and each ended by "\n". Twin of jxIndent in
// abnf/jsrtjavastr.go.
function jxIndent(s, n) {
    if (s.length == 0) { return "" }
    var ls = jxLines(s)
    var pad = n > 0 ? jxRepeat(" ", n) : ""
    var out = ""
    for (var i = 0; i < ls.length; i++) {
        var l = ls[i]
        if (n < 0) {
            var k = jxFirstNonWs(l)
            l = l.substring(0 - n < k ? 0 - n : k)
        }
        out = out + pad + l + "\n"
    }
    return out
}
// stripIndent: each line loses its trailing whitespace and the leading
// whitespace every non-blank line (and a blank last line) shares; a string
// ending in a line terminator keeps all its indentation and gets a final "\n".
// Twin of jxStripIndent in abnf/jsrtjavastr.go.
function jxStripIndent(s) {
    if (s.length == 0) { return "" }
    var last = s.charAt(s.length - 1)
    var optOut = last == "\n" || last == "\r"
    var ls = jxLines(s)
    var outdent = 0
    if (!optOut) {
        outdent = 2147483647
        for (var i = 0; i < ls.length; i++) {
            var k = jxFirstNonWs(ls[i])
            if (k != ls[i].length && k < outdent) { outdent = k }
        }
        var ll = ls[ls.length - 1]
        if (jxFirstNonWs(ll) == ll.length && ll.length < outdent) { outdent = ll.length }
    }
    var out = ""
    for (var j = 0; j < ls.length; j++) {
        var l = ls[j]
        var f = jxFirstNonWs(l)
        var e = jxEndNonWs(l)
        if (j > 0) { out = out + "\n" }
        if (f < e) { out = out + l.substring(f < outdent ? f : outdent, e) }
    }
    return optOut ? out + "\n" : out
}
// regionMatches: the len units of s from to and of o from oo, folded as
// compareToIgnoreCase folds them when ic; false when a region runs off its
// string.
function jxRegionMatches(s, ic, to, o, oo, len) {
    if (oo < 0 || to < 0 || to > s.length - len || oo > o.length - len) { return false }
    for (var i = 0; i < len; i++) {
        var a = s.charCodeAt(to + i)
        var b = o.charCodeAt(oo + i)
        if (a != b && (!ic || jxFold(a) != jxFold(b))) { return false }
    }
    return true
}
// codePointCount(b, e): a surrogate pair counts once; -1 for a range outside s.
function jxCodePointCount(s, b, e) {
    if (b < 0 || e > s.length || b > e) { return 0 - 1 }
    var n = 0
    var i = b
    while (i < e) {
        var c = s.charCodeAt(i)
        i = i + 1
        if (c >= 55296 && c < 56320 && i < e && s.charCodeAt(i) >= 56320 && s.charCodeAt(i) < 57344) { i = i + 1 }
        n = n + 1
    }
    return n
}
// ----- Integer's unsigned and bit members, on int values (plain numbers) -----
//
// An int's 32 bits read as unsigned, and back; every step is exact in a double.
function jxU32(i) { return i < 0 ? i + 4294967296 : i }
function jxI32(u) { return u >= 2147483648 ? u - 4294967296 : u }
function jxReverse32(i) {
    var u = jxU32(i)
    var r = 0
    for (var k = 0; k < 32; k++) {
        r = r * 2 + u % 2
        u = Math.floor(u / 2)
    }
    return jxI32(r)
}
function jxReverseBytes32(i) {
    var u = jxU32(i)
    var r = 0
    for (var k = 0; k < 4; k++) {
        r = r * 256 + u % 256
        u = Math.floor(u / 256)
    }
    return jxI32(r)
}
// rotateLeft by d mod 32; rotateRight(i, d) is rotateLeft(i, -d).
function jxRotl32(i, d) {
    var n = d & 31
    var u = jxU32(i)
    var cut = Math.pow(2, 32 - n)
    return jxI32((u % cut) * Math.pow(2, n) + Math.floor(u / cut))
}
// ----- Math.ulp and Math.IEEEremainder -----
//
// ulp: the gap to the next double (f32: float) away from zero at |f|'s
// exponent; the subnormal gap below the least normal; NaN and Infinity answer
// themselves. Scaling by two is exact.
function jxUlp(f, f32) {
    if (f !== f) { return f }
    var a = f < 0 ? -f : f
    if (a - a !== 0) { return a }
    var minExp = f32 ? 0 - 126 : 0 - 1022
    var mant = f32 ? 23 : 52
    if (a < Math.pow(2, minExp)) { return Math.pow(2, minExp - mant) }
    var e = 0
    while (a >= 2) {
        a = a / 2
        e = e + 1
    }
    while (a < 1) {
        a = a * 2
        e = e - 1
    }
    return Math.pow(2, e - mant)
}
// Math.nextUp: the next double (f32: float) toward +Infinity. Upward the gap is
// ulp(f); below a negative power of two it is half that, the subnormal range
// aside. -MIN_VALUE steps to -0.0, -Infinity to -MAX_VALUE. Twin of
// math.Nextafter / Nextafter32 in abnf/jsrtjava.go.
function jxNextUp(f, f32) {
    if (f !== f || (f - f !== 0 && f > 0)) { return f }
    var mant = f32 ? 23 : 52
    var maxV = (2 - Math.pow(2, 0 - mant)) * Math.pow(2, f32 ? 127 : 1023)
    if (f - f !== 0) { return -maxV }
    if (f == 0) { return Math.pow(2, f32 ? 0 - 149 : 0 - 1074) }
    var u = jxUlp(f, f32)
    if (f > 0) {
        var up = f + u
        return up > maxV ? u / 0 : up
    }
    var a = -f
    if (a > Math.pow(2, f32 ? 0 - 126 : 0 - 1022) && u * Math.pow(2, mant) == a) { u = u / 2 }
    var down = f + u
    if (down == 0) { return -down }
    return down
}
// The double of a doubleToLongBits pattern (unsigned 32-bit halves) and the
// float of a floatToIntBits one (unsigned): Double.longBitsToDouble and
// Float.intBitsToFloat. Every NaN pattern reads as NaN. Twins: math.Float64frombits
// and math.Float32frombits in abnf/jsrtjava.go.
function jxBitsDbl(hi, lo) {
    return jxBitsFloat(hi, lo, 11, 52)
}
function jxBitsFlt(bits) {
    return jxBitsFloat(bits, 0, 8, 23)
}
// The value of the IEEE-754 pattern whose top 32 bits are hi and whose low 32
// are lo (0 for a float), with an exponent field of ebits bits and mbits
// mantissa bits.
function jxBitsFloat(hi, lo, ebits, mbits) {
    var neg = hi >= 2147483648
    var h = neg ? hi - 2147483648 : hi
    var span = Math.pow(2, 31 - ebits)
    var biased = Math.floor(h / span)
    var mant = (h - biased * span) * (mbits > 31 ? 4294967296 : 1) + lo
    var bias = Math.pow(2, ebits - 1) - 1
    var v = 0
    if (biased == bias * 2 + 1) {
        if (mant != 0) { return 0 / v }
        v = 1 / v
    } else if (biased == 0) {
        v = mant * Math.pow(2, 1 - bias - mbits)
    } else {
        v = (mant + Math.pow(2, mbits)) * Math.pow(2, biased - bias - mbits)
    }
    return neg ? -v : v
}
// IEEEremainder: x - p * n, n the integer nearest x / p (ties to even), with
// x's sign - fdlibm's __ieee754_remainder, whose every step is exact.
function jxIeeeRem(x, p) {
    if (x !== x || p !== p || x - x !== 0 || p == 0) { return (x - x) * p / 0 }
    if (p - p !== 0) { return x }
    var neg = x < 0 || (x == 0 && 1 / x < 0)
    var hp = p < 0 ? -p : p
    var hx = neg ? -x : x
    if (hp < Math.pow(2, 1023)) { hx = hx % (hp + hp) }
    if (hp < Math.pow(2, 0 - 1021)) {
        if (hx + hx > hp) {
            hx = hx - hp
            if (hx + hx >= hp) { hx = hx - hp }
        }
    } else {
        var half = 0.5 * hp
        if (hx > half) {
            hx = hx - hp
            if (hx >= half) { hx = hx - hp }
        }
    }
    if (neg) { return -hx }
    return hx
}
// Math.multiplyHigh: the high 64 bits of the signed 128-bit product of two
// longs, each given as its unsigned 32-bit halves; the answer is [hi, lo]
// halves. Sixteen-bit limbs keep every partial product exact; the unsigned
// product is corrected for a negative operand by subtracting the other.
function jxMulHigh(ah, al, bh, bl) {
    var a = [al % 65536, Math.floor(al / 65536), ah % 65536, Math.floor(ah / 65536)]
    var b = [bl % 65536, Math.floor(bl / 65536), bh % 65536, Math.floor(bh / 65536)]
    var col = [0, 0, 0, 0, 0, 0, 0, 0]
    for (var i = 0; i < 4; i++) {
        for (var j = 0; j < 4; j++) { col[i + j] = col[i + j] + a[i] * b[j] }
    }
    var limbs = []
    var carry = 0
    for (var k = 0; k < 8; k++) {
        var t = col[k] + carry
        limbs.push(t % 65536)
        carry = Math.floor(t / 65536)
    }
    var r = [limbs[7] * 65536 + limbs[6], limbs[5] * 65536 + limbs[4]]
    if (ah >= 2147483648) { r = jxSub64(r, bh, bl) }
    if (bh >= 2147483648) { r = jxSub64(r, ah, al) }
    return r
}
// [h, l] - (sh, sl) modulo 2^64, all unsigned 32-bit halves.
function jxSub64(r, sh, sl) {
    var l = r[1] - sl
    var borrow = l < 0 ? 1 : 0
    var h = r[0] - sh - borrow
    if (h < 0) { h = h + 4294967296 }
    return [h, l + borrow * 4294967296]
}
// Long.parseUnsignedLong(s, radix) as unsigned 32-bit halves [hi, lo]: an
// optional "+" and then digits (Character.digit) up to 2^64 - 1, each running
// past it "exceeds range". Twin of jvParseU64 in abnf/jsrtjava.go.
function jxParseU64(s, radix) {
    if (radix < 2) { jxkThrow("NumberFormatException", "radix " + radix + " less than Character.MIN_RADIX") }
    if (radix > 36) { jxkThrow("NumberFormatException", "radix " + radix + " greater than Character.MAX_RADIX") }
    var bad = "For input string: \"" + s + "\"" + (radix == 10 ? "" : " under radix " + radix)
    if (s.length > 0 && s.charAt(0) == "-") {
        jxkThrow("NumberFormatException", "Illegal leading minus sign on unsigned string " + s + ".")
    }
    var i = s.length > 0 && s.charAt(0) == "+" ? 1 : 0
    if (i == s.length) { jxkThrow("NumberFormatException", bad) }
    var hi = 0
    var lo = 0
    while (i < s.length) {
        var d = jcDigitVal(s.charCodeAt(i))
        if (d >= radix) { jxkThrow("NumberFormatException", bad) }
        var l2 = lo * radix + d
        var carry = Math.floor(l2 / 4294967296)
        lo = l2 - carry * 4294967296
        hi = hi * radix + carry
        if (hi >= 4294967296) { jxkThrow("NumberFormatException", "String value " + s + " exceeds range of unsigned long.") }
        i = i + 1
    }
    return [hi, lo]
}
// Long.reverse / reverseBytes / rotateLeft over unsigned 32-bit halves, as
// [hi, lo]. A rotation by 32 or more swaps the halves first.
function jxReverse64(h, l) { return [jxU32(jxReverse32(jxI32(l))), jxU32(jxReverse32(jxI32(h)))] }
function jxReverseBytes64(h, l) { return [jxU32(jxReverseBytes32(jxI32(l))), jxU32(jxReverseBytes32(jxI32(h)))] }
function jxRotl64(h, l, dist) {
    var d = dist % 64
    if (d < 0) { d = d + 64 }
    var a = h
    var b = l
    if (d >= 32) {
        a = l
        b = h
        d = d - 32
    }
    if (d == 0) { return [a, b] }
    var p = Math.pow(2, d)
    var q = Math.pow(2, 32 - d)
    var ah = Math.floor(a / q)
    var bh = Math.floor(b / q)
    return [(a - ah * q) * p + bh, (b - bh * q) * p + ah]
}
// A code point as its one or two UTF-16 units.
function jxCpStr(cp) {
    if (cp < 65536) { return String.fromCharCode(cp) }
    var v = cp - 65536
    return String.fromCharCode(55296 + Math.floor(v / 1024)) + String.fromCharCode(56320 + v % 1024)
}
function jxJoin(delim, parts) {
    var out = ""
    var i = 0
    while (i < parts.length) {
        if (i > 0) { out = out + delim }
        out = out + parts[i]
        i = i + 1
    }
    return out
}

// Integer.decode / Long.decode: the text parseInt reads and its radix - an
// optional sign, then 0x, 0X or # for hex, or a leading 0 before more digits for
// octal. Twin of jxDecode in abnf/jsrtjavastr.go.
function jxDecode(s) {
    if (s.length == 0) { jxkThrow("NumberFormatException", "Zero length string") }
    var i = 0
    var sign = ""
    var c = s.charAt(0)
    if (c == "-" || c == "+") {
        if (c == "-") { sign = "-" }
        i = 1
    }
    var radix = 10
    var two = s.substring(i, i + 2)
    if (two == "0x" || two == "0X") {
        i = i + 2
        radix = 16
    } else if (s.substring(i, i + 1) == "#") {
        i = i + 1
        radix = 16
    } else if (s.substring(i, i + 1) == "0" && s.length > i + 1) {
        i = i + 1
        radix = 8
    }
    var d = s.substring(i, i + 1)
    if (d == "-" || d == "+") { jxkThrow("NumberFormatException", "Sign character in wrong position") }
    return {text: sign + s.substring(i), radix: radix}
}
// Character.forDigit: the lowercase digit d stands for in radix r, '\0' for a
// digit or radix out of range. Twin of jxForDigit in abnf/jsrtjavastr.go.
function jxForDigit(d, r) {
    if (r < 2 || r > 36 || d < 0 || d >= r) { return 0 }
    if (d < 10) { return 48 + d }
    return 87 + d
}

// ----- the regex methods (java.util.regex.Pattern / Matcher) -----

// Java's pattern flags in regex.js's terms: \Q...\E quoting and the J dialect.
function jxRe(pattern) {
    var re = rxGet(pattern, "qJ")
    if (!re.ok) { jxkThrow("PatternSyntaxException", jxReMsg(pattern, re.err, re.errAt)) }
    return re
}
// Matcher.find's next start: after an empty match the search moves one on.
function jxNext(m) {
    if (m.end == m.begin) { return m.end + 1 }
    return m.end
}
// Pattern.split(input, limit): a positive limit caps the pieces, zero drops the
// trailing empty ones, and an empty match at 0 makes no leading empty piece.
function jxSplit(s, pattern, limit) {
    var re = jxRe(pattern)
    var out = []
    var index = 0
    var at = 0
    while (at <= s.length) {
        var m = rxSearch(re, s, at)
        if (m == null) { break }
        if (limit <= 0 || out.length < limit - 1) {
            if (!(index == 0 && m.begin == 0 && m.end == 0)) {
                out.push(s.substring(index, m.begin))
                index = m.end
            }
        } else {
            out.push(s.substring(index, s.length))
            index = m.end
            break
        }
        at = jxNext(m)
    }
    if (index == 0) { return [s] }
    if (limit <= 0 || out.length < limit) { out.push(s.substring(index, s.length)) }
    if (limit == 0) {
        var n = out.length
        while (n > 0 && out[n - 1].length == 0) { n = n - 1 }
        var cut = []
        var i = 0
        while (i < n) {
            cut.push(out[i])
            i = i + 1
        }
        return cut
    }
    return out
}
function jxMatches(s, pattern) { return rxMatchWhole(jxRe(pattern), s, 0) != null }
// replaceAll / replaceFirst.
function jxRxReplace(s, pattern, repl, all) {
    var re = jxRe(pattern)
    var out = ""
    var tail = 0
    var at = 0
    while (at <= s.length) {
        var m = rxSearch(re, s, at)
        if (m == null) { break }
        out = out + s.substring(tail, m.begin) + jxExpand(re, s, m, repl)
        tail = m.end
        if (!all) { break }
        at = jxNext(m)
    }
    return out + s.substring(tail, s.length)
}
// Matcher.appendReplacement's template: a backslash takes the next character
// literally, $n is group n - as many digits as still name a group - and
// ${name} a named group. An unmatched group contributes nothing.
function jxExpand(re, s, m, repl) {
    var out = ""
    var i = 0
    while (i < repl.length) {
        var c = repl.charAt(i)
        if (c == "\\") {
            if (i + 1 >= repl.length) { jxkThrow("IllegalArgumentException", "character to be escaped is missing") }
            out = out + repl.charAt(i + 1)
            i = i + 2
            continue
        }
        if (c != "$") {
            out = out + c
            i = i + 1
            continue
        }
        i = i + 1
        if (i >= repl.length) { jxkThrow("IllegalArgumentException", "Illegal group reference: group index is missing") }
        var g = 0 - 1
        if (repl.charAt(i) == "{") {
            var e = i + 1
            while (e < repl.length && repl.charAt(e) != "}") { e = e + 1 }
            var nm = repl.substring(i + 1, e)
            g = rxIndexOfName(re, nm)
            if (g < 0) { jxkThrow("IllegalArgumentException", "No group with name {" + nm + "}") }
            i = e + 1
        } else {
            var d = repl.charCodeAt(i) - 48
            if (d < 0 || d > 9) { jxkThrow("IllegalArgumentException", "Illegal group reference") }
            g = d
            i = i + 1
            if (g > rxGroupCount(re)) { jxkThrow("IndexOutOfBoundsException", "No group " + g) }
            while (i < repl.length) {
                var d2 = repl.charCodeAt(i) - 48
                if (d2 < 0 || d2 > 9) { break }
                var ng = g * 10 + d2
                if (ng > rxGroupCount(re)) { break }
                g = ng
                i = i + 1
            }
        }
        var gs = rxGroup(re, s, m, g)
        if (gs != null) { out = out + gs }
    }
    return out
}

// ----- String.format (java.util.Formatter, Locale.US) -----
//
// %[index$][flags][width][.precision]conversion with the conversions
// b s h c d o x e f g a % n (and their upper-case forms). A double is formatted
// from its Double.toString digits rounded HALF_UP, which is Java's rule -
// %.2f of 1.005 is 1.01 and %.1f of 0.15 is 0.2 - and a float is widened to a
// double first. A malformed specifier, a missing argument and a mismatched one
// throw java.util.Formatter's exceptions (jxkThrow); the date conversion t is
// reported through jxkFail.
function jxFormat(fmt, args) {
    var out = ""
    var i = 0
    var next = 0
    var last = 0 - 1
    while (i < fmt.length) {
        var c = fmt.charAt(i)
        if (c != "%") {
            out = out + c
            i = i + 1
            continue
        }
        var start = i
        var j = i + 1
        var k = j
        while (k < fmt.length && jxDigit(fmt.charCodeAt(k))) { k = k + 1 }
        var idx = 0 - 1
        if (k > j && k < fmt.length && fmt.charAt(k) == "$") {
            idx = parseInt(fmt.substring(j, k), 10) - 1
            if (idx < 0) { jxkThrow("IllegalFormatArgumentIndexException", "Illegal format argument index = " + (idx + 1)) }
            j = k + 1
        }
        var flags = ""
        while (j < fmt.length && "-#+ 0,(<".indexOf(fmt.charAt(j)) >= 0) {
            if (jxHas(flags, fmt.charAt(j))) { jxkThrow("DuplicateFormatFlagsException", "Flags = '" + fmt.charAt(j) + "'") }
            flags = flags + fmt.charAt(j)
            j = j + 1
        }
        var width = 0 - 1
        k = j
        while (k < fmt.length && jxDigit(fmt.charCodeAt(k))) { k = k + 1 }
        if (k > j) { width = parseInt(fmt.substring(j, k), 10) }
        j = k
        var prec = 0 - 1
        var bad = false
        if (j < fmt.length && fmt.charAt(j) == ".") {
            k = j + 1
            while (k < fmt.length && jxDigit(fmt.charCodeAt(k))) { k = k + 1 }
            if (k == j + 1) { bad = true }
            prec = parseInt(fmt.substring(j + 1, k), 10)
            j = k
        }
        // A specifier that does not parse names the character after the '%'.
        if (bad || j >= fmt.length) { jxkThrow("UnknownFormatConversionException", "Conversion = '" + (start + 1 < fmt.length ? fmt.charAt(start + 1) : "%") + "'") }
        var conv = fmt.charAt(j)
        i = j + 1
        jxCheckSpec(conv, flags, width, prec, idx)
        if (conv == "n") {
            out = out + "\n"
            continue
        }
        if (conv == "%") {
            out = out + jxPad("%", flags, width)
            continue
        }
        var ai = next
        if (flags.indexOf("<") >= 0) { ai = last } else if (idx >= 0) { ai = idx } else { next = next + 1 }
        if (ai < 0 || ai >= args.length) { jxkThrow("MissingFormatArgumentException", "Format specifier '" + fmt.substring(start, i) + "'") }
        last = ai
        out = out + jxConv(conv, flags, width, prec, args[ai])
    }
    return out
}
// The checks java.util.Formatter makes of a specifier before it reads the
// argument (FormatSpecifier's constructor): the conversion, the flags and the
// precision it takes, the width '-' and '0' need.
function jxCheckSpec(conv, flags, width, prec, idx) {
    if ("bBhHsScCdoxXeEfgGaAtT%n".indexOf(conv) < 0) { jxkThrow("UnknownFormatConversionException", "Conversion = '" + conv + "'") }
    var lc = conv.toLowerCase()
    var sp = jxSpecStr(conv, flags, width, prec, idx)
    if (lc == "%" || lc == "n") {
        if (prec >= 0) { jxkThrow("IllegalFormatPrecisionException", "" + prec) }
        if (lc == "n" && width >= 0) { jxkThrow("IllegalFormatWidthException", "" + width) }
        if (flags != "" && (lc == "n" || flags != "-")) { jxkThrow("IllegalFormatFlagsException", "Flags = '" + jxFlagsCanon(flags) + "'") }
        if (width < 0 && jxHas(flags, "-")) { jxkThrow("MissingFormatWidthException", sp) }
        return undefined
    }
    if (lc == "b" || lc == "h" || lc == "s") {
        if (lc != "s" && jxHas(flags, "#")) { jxMismatch(lc, "#") }
        if (width < 0 && jxHas(flags, "-")) { jxkThrow("MissingFormatWidthException", sp) }
        jxBadFlags(lc, flags, "+ 0,(")
        return undefined
    }
    if (lc == "c") {
        if (prec >= 0) { jxkThrow("IllegalFormatPrecisionException", "" + prec) }
        jxBadFlags(lc, flags, "#+ 0,(")
        if (width < 0 && jxHas(flags, "-")) { jxkThrow("MissingFormatWidthException", sp) }
        return undefined
    }
    if (lc == "t") { return undefined }
    if (width < 0 && (jxHas(flags, "-") || jxHas(flags, "0"))) { jxkThrow("MissingFormatWidthException", sp) }
    if ((jxHas(flags, "+") && jxHas(flags, " ")) || (jxHas(flags, "-") && jxHas(flags, "0"))) {
        jxkThrow("IllegalFormatFlagsException", "Flags = '" + jxFlagsCanon(flags) + "'")
    }
    if (lc == "d" || lc == "o" || lc == "x") {
        if (prec >= 0) { jxkThrow("IllegalFormatPrecisionException", "" + prec) }
        jxBadFlags(lc, flags, lc == "d" ? "#" : ",")
    } else if (lc == "a") {
        jxBadFlags(lc, flags, "(,")
    } else if (lc == "e") {
        jxBadFlags(lc, flags, ",")
    } else if (lc == "g") {
        jxBadFlags(lc, flags, "#")
    }
    return undefined
}
function jxBadFlags(lc, flags, bad) {
    for (var i = 0; i < bad.length; i++) {
        if (jxHas(flags, bad.charAt(i))) { jxMismatch(lc, bad.charAt(i)) }
    }
    return undefined
}
function jxMismatch(lc, flag) { jxkThrow("FormatFlagsConversionMismatchException", "Conversion = " + lc + ", Flags = " + flag) }
// The flags in java.util.Formatter's own order, as its messages spell them.
function jxFlagsCanon(flags) {
    var all = "-#+ 0,(<"
    var out = ""
    for (var i = 0; i < all.length; i++) {
        if (jxHas(flags, all.charAt(i))) { out = out + all.charAt(i) }
    }
    return out
}
// The specifier as Formatter's toString writes it back.
function jxSpecStr(conv, flags, width, prec, idx) {
    var out = "%" + jxFlagsCanon(flags)
    if (idx >= 0) { out = out + (idx + 1) + "$" }
    if (width >= 0) { out = out + width }
    if (prec >= 0) { out = out + "." + prec }
    return out + conv
}
function jxDigit(c) { return c >= 48 && c <= 57 }
function jxHas(flags, f) { return flags.indexOf(f) >= 0 }
// Right-justified with spaces, or left-justified with the '-' flag.
function jxPad(s, flags, width) {
    var out = s
    if (jxHas(flags, "-")) {
        while (out.length < width) { out = out + " " }
        return out
    }
    while (out.length < width) { out = " " + out }
    return out
}
function jxConv(conv, flags, width, prec, v) {
    var lc = conv.toLowerCase()
    var up = conv != lc
    var k = jxkKind(v)
    var body = ""
    if (lc == "s") {
        // '#' is for a Formattable, which no value here is.
        if (jxHas(flags, "#")) { jxMismatch(lc, "#") }
        body = k == "null" ? "null" : jxkStr(v)
        if (prec >= 0 && prec < body.length) { body = body.substring(0, prec) }
    } else if (lc == "h") {
        body = k == "null" ? "null" : jxHex32(jxkHash(v))
        if (prec >= 0 && prec < body.length) { body = body.substring(0, prec) }
    } else if (lc == "b") {
        if (k == "null") { body = "false" } else if (k == "bool") { body = jxkStr(v) } else { body = "true" }
        if (prec >= 0 && prec < body.length) { body = body.substring(0, prec) }
    } else if (lc == "c") {
        if (k == "null") { body = "null" }
        else if (k == "char") { body = jxCpStr(jxkCode(v)) }
        else if (k == "int" && jxkClass(v) != "java.lang.Long") {
            var cp = jxkCode(v)
            if (cp < 0 || cp > 1114111) { jxkThrow("IllegalFormatCodePointException", "Code point = 0x" + jxHex32(cp)) }
            body = jxCpStr(cp)
        }
        else { jxBadConv(lc, v) }
    } else if (lc == "d") {
        if (k == "null") { body = "null" } else {
            if (k != "int") { jxBadConv(lc, v) }
            var dd = jxkDec(v)
            var neg = dd.charAt(0) == "-"
            if (neg) { dd = dd.substring(1) }
            if (jxHas(flags, ",")) { dd = jxGroup(dd) }
            return jxNum(neg, dd, flags, width, up)
        }
    } else if (lc == "x" || lc == "o") {
        if (k == "null") { body = "null" } else {
            if (k != "int") { jxBadConv(lc, v) }
            // A sign or a bracket is for a BigInteger only.
            jxBadFlags(lc, flags, "( +")
            var rd = jxkRadix(v, lc == "x" ? 16 : 8)
            if (jxHas(flags, "#")) { rd = (lc == "x" ? "0x" : "0") + rd }
            if (jxHas(flags, "0")) {
                while (rd.length < width) { rd = (lc == "x" && jxHas(flags, "#")) ? "0x0" + rd.substring(2) : "0" + rd }
            }
            body = rd
        }
    } else if (lc == "f" || lc == "e" || lc == "g" || lc == "a") {
        if (k == "null") { body = "null" } else {
            if (k != "float") { jxBadConv(lc, v) }
            if (lc == "a") { return jxHexFloat(jxkDbl(v), flags, width, prec, up) }
            return jxFloat(lc, jxkDbl(v), flags, width, prec, up)
        }
    } else {
        jxkFail("unsupported conversion %" + conv)
    }
    // A null argument prints "null", cut to the precision like a string.
    if (k == "null" && prec >= 0 && prec < body.length) { body = body.substring(0, prec) }
    if (up) { body = unicodeCase(body, 3) }
    return jxPad(body, flags, width)
}
function jxBadConv(lc, v) {
    jxkThrow("IllegalFormatConversionException", lc + " != " + jxkClass(v))
}
// An int's 32 bits as unsigned hex digits, Integer.toHexString.
function jxHex32(n) {
    var u = n < 0 ? n + 4294967296 : n
    var out = ""
    while (u >= 16) {
        out = "0123456789abcdef".charAt(u % 16) + out
        u = Math.floor(u / 16)
    }
    return "0123456789abcdef".charAt(u) + out
}
// Groups of three integer digits, Locale.US.
function jxGroup(ds) {
    var out = ""
    var n = ds.length
    var i = 0
    while (i < n) {
        if (i > 0 && (n - i) % 3 == 0) { out = out + "," }
        out = out + ds.charAt(i)
        i = i + 1
    }
    return out
}
// A number's sign and padding: '(' brackets a negative, '+' and ' ' mark a
// non-negative, '0' pads with zeros after the sign, '-' left-justifies.
function jxNum(neg, digits, flags, width, up) {
    var pre = ""
    var post = ""
    if (neg) {
        if (jxHas(flags, "(")) {
            pre = "("
            post = ")"
        } else {
            pre = "-"
        }
    } else if (jxHas(flags, "+")) {
        pre = "+"
    } else if (jxHas(flags, " ")) {
        pre = " "
    }
    var body = digits
    if (up) { body = unicodeCase(body, 3) }
    if (jxHas(flags, "0")) {
        while (pre.length + body.length + post.length < width) { body = "0" + body }
    }
    return jxPad(pre + body + post, flags, width)
}
// The digits ds rounded HALF_UP to keep digits - one longer when the carry
// runs out of the front (999 -> 1000), empty when keep < 0 or the value
// rounds away entirely.
function jxRoundUp(ds, keep) {
    if (keep < 0) { return "" }
    if (keep >= ds.length) { return ds }
    var head = ds.substring(0, keep)
    if (ds.charCodeAt(keep) < 53) { return head }
    var out = ""
    var carry = 1
    var i = keep - 1
    while (i >= 0) {
        var d = head.charCodeAt(i) - 48 + carry
        carry = 0
        if (d == 10) {
            d = 0
            carry = 1
        }
        out = "" + d + out
        i = i - 1
    }
    if (carry == 1) { out = "1" + out }
    return out
}
function jxZeros(n) {
    var out = ""
    var i = 0
    while (i < n) {
        out = out + "0"
        i = i + 1
    }
    return out
}
// %e / %f / %g of a double.
function jxFloat(lc, x, flags, width, precIn, up) {
    if (x !== x) { return jxPad(up ? "NAN" : "NaN", flags, width) }
    var neg = x < 0 || (x == 0 && 1 / x < 0)
    var a = neg ? 0 - x : x
    if (a - a !== 0) {
        var inf = up ? "INFINITY" : "Infinity"
        var nf = flags
        if (jxHas(nf, "0")) { nf = jxReplace(nf, "0", "") }
        return jxNum(neg, inf, nf, width, false)
    }
    var ds = "0"
    var e = 0
    if (a != 0) {
        var dg = jxDigitsOf(jxkDblStr(a))
        var ei = dg.indexOf("e")
        ds = dg.substring(0, ei)
        e = parseInt(dg.substring(ei + 1, dg.length), 10)
    }
    var prec = precIn
    var body = ""
    if (lc == "g") {
        if (prec < 0) { prec = 6 }
        if (prec == 0) { prec = 1 }
        var re = e
        if (a != 0) {
            var r = jxRoundUp(ds, prec)
            if (r.length > prec) { re = e + 1 }
        }
        if (a == 0 || (re >= 0 - 4 && re < prec)) {
            body = jxFixed(ds, e, a == 0 ? prec - 1 : prec - 1 - re, jxHas(flags, ","))
        } else {
            body = jxSci(ds, e, prec - 1, false)
        }
    } else if (lc == "f") {
        body = jxFixed(ds, e, prec < 0 ? 6 : prec, jxHas(flags, ","))
        if (prec == 0 && jxHas(flags, "#")) { body = body + "." }
    } else {
        body = jxSci(ds, e, prec < 0 ? 6 : prec, jxHas(flags, "#"))
    }
    return jxNum(neg, body, flags, width, up)
}
// %a of a double: Double.toHexString's 0x1.hhhp-e (0x0.hhhp-1022 for a
// subnormal), or, with a precision below 13, the significand rounded half-even
// to that many hex digits (at least one) after normalizing a subnormal. A
// float argument is widened first. The double is scaled by powers of two,
// which is exact, and its 52 fraction bits read off four at a time.
function jxHexFloat(x, flags, width, prec, up) {
    if (x !== x || x - x !== 0) { return jxFloat("e", x, flags, width, prec, up) }
    var neg = x < 0 || (x == 0 && 1 / x < 0)
    var a = neg ? 0 - x : x
    var lead = 1
    var e = 0
    var m = a
    var cut = prec >= 0 && prec < 13
    if (a == 0) {
        lead = 0
    } else if (a < Math.pow(2, 0 - 1022)) {
        m = a * Math.pow(2, 1022)
        e = 0 - 1022
        if (cut) {
            while (m < 1) {
                m = m * 2
                e = e - 1
            }
        } else {
            lead = 0
        }
    } else {
        while (m >= 2) {
            m = m / 2
            e = e + 1
        }
        while (m < 1) {
            m = m * 2
            e = e - 1
        }
    }
    var frac = m - lead
    var ds = []
    for (var i = 0; i < 13; i++) {
        frac = frac * 16
        var d = Math.floor(frac)
        ds.push(d)
        frac = frac - d
    }
    var n = 13
    if (cut) {
        n = prec == 0 ? 1 : prec
        var first = ds[n]
        var more = false
        for (var r = n + 1; r < 13; r++) { if (ds[r] != 0) { more = true } }
        if (first > 8 || (first == 8 && (more || ds[n - 1] % 2 == 1))) {
            var q = n - 1
            while (q >= 0 && ds[q] == 15) {
                ds[q] = 0
                q = q - 1
            }
            if (q >= 0) { ds[q] = ds[q] + 1 } else { lead = lead + 1 }
            if (lead == 2) {
                lead = 1
                e = e + 1
            }
        }
    } else if (prec < 0) {
        while (n > 1 && ds[n - 1] == 0) { n = n - 1 }
    }
    var hex = ""
    for (var h = 0; h < n; h++) { hex = hex + "0123456789abcdef".charAt(ds[h]) }
    while (hex.length < prec) { hex = hex + "0" }
    var body = "" + lead + "." + hex + "p" + e
    var sign = neg ? "-" : (jxHas(flags, "+") ? "+" : (jxHas(flags, " ") ? " " : ""))
    if (jxHas(flags, "0")) {
        while (sign.length + 2 + body.length < width) { body = "0" + body }
    }
    var out = sign + "0x" + body
    if (up) { out = unicodeCase(out, 3) }
    return jxPad(out, flags, width)
}
// Double.toString's digits ("4.9E-324", "123.45") as DIGITS "e" EXP, EXP the
// exponent of the first digit. They are the shortest round-tripping digits
// except that Java keeps two significant digits, the second read off the
// actual value - so Double.MIN_VALUE formats as 4.900000e-324.
function jxDigitsOf(t) {
    var mant = t
    var exp = 0
    var ei = t.indexOf("E")
    if (ei >= 0) {
        mant = t.substring(0, ei)
        exp = parseInt(t.substring(ei + 1, t.length), 10)
    }
    var ds = ""
    var pt = 0 - 1
    var i = 0
    while (i < mant.length) {
        var c = mant.charAt(i)
        if (c == ".") { pt = ds.length } else { ds = ds + c }
        i = i + 1
    }
    if (pt < 0) { pt = ds.length }
    var lead = 0
    while (lead < ds.length - 1 && ds.charAt(lead) == "0") { lead = lead + 1 }
    ds = ds.substring(lead, ds.length)
    var end = ds.length
    while (end > 1 && ds.charAt(end - 1) == "0") { end = end - 1 }
    return ds.substring(0, end) + "e" + (pt - lead + exp - 1)
}
// The digits ds (value 0.ds * 10^(e+1)) with prec fraction digits.
function jxFixed(ds, e, prec, group) {
    var pt = e + 1
    var r = ds == "0" ? "" : jxRoundUp(ds, pt + prec)
    if (r.length > pt + prec && pt + prec >= 0) { pt = pt + 1 }
    var ints = "0"
    var frac = ""
    if (pt <= 0) {
        frac = jxZeros(0 - pt) + r
    } else {
        while (r.length < pt) { r = r + "0" }
        ints = r.substring(0, pt)
        frac = r.substring(pt, r.length)
    }
    while (frac.length < prec) { frac = frac + "0" }
    if (frac.length > prec) { frac = frac.substring(0, prec) }
    if (group) { ints = jxGroup(ints) }
    if (prec == 0) { return ints }
    return ints + "." + frac
}
// d.ddd e+XX with prec fraction digits and at least two exponent digits.
function jxSci(ds, e, prec, point) {
    var ex = e
    var r = ds
    if (ds != "0") {
        r = jxRoundUp(ds, prec + 1)
        if (r.length > prec + 1) {
            ex = ex + 1
            r = r.substring(0, prec + 1)
        }
    } else {
        ex = 0
    }
    while (r.length < prec + 1) { r = r + "0" }
    var m = r.substring(0, 1)
    if (prec > 0 || point) { m = m + "." + r.substring(1, r.length) }
    var es = ex < 0 ? "" + (0 - ex) : "" + ex
    if (es.length < 2) { es = "0" + es }
    return m + "e" + (ex < 0 ? "-" : "+") + es
}
