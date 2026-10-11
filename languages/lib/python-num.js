// python-num.js -- the bit and double algorithms behind int's and float's own
// methods (bit_length, bit_count, to_bytes, from_bytes, float.hex,
// float.fromhex, float.as_integer_ratio), for python-interpreter.abnf (include)
// and languages/lib/python-rt.metajs (import). An int crosses as its sign and
// its magnitude's little-endian BITS, the shape both engines' bigint code
// already converts to and from; an error crosses as a string code that the
// engine raises. Go twin: abnf/jsrtpynum.go (math/big and strconv).
//
// Written in the subset both engines accept: no exponent literals, and powers of
// two are built by exact doubling.

// The magnitude bits of an integral double, exact at every magnitude: the low bit
// is the remainder by 2, and subtracting it leaves an even value whose half is
// exact.
function pynNumBits(n) {
    var m = n < 0 ? -n : n
    var bits = []
    while (m > 0) {
        var b = m % 2
        bits.push(b)
        m = (m - b) / 2
    }
    return bits
}

// The bits of 2^k.
function pynPow2Bits(k) {
    var bits = []
    var i = 0
    while (i < k) {
        bits.push(0)
        i = i + 1
    }
    bits.push(1)
    return bits
}

// The number of one bits.
function pynBitCount(bits) {
    var c = 0
    var i = 0
    while (i < bits.length) {
        c = c + bits[i]
        i = i + 1
    }
    return c
}

// The bits with trailing (most significant) zeros dropped.
function pynBitsTrim(bits) {
    var n = bits.length
    while (n > 0 && bits[n - 1] == 0) { n = n - 1 }
    var out = []
    var i = 0
    while (i < n) {
        out.push(bits[i])
        i = i + 1
    }
    return out
}

// The two's complement of a magnitude over nb bits: invert, then add one.
function pynTwos(bits, nb) {
    var out = []
    var carry = 1
    var i = 0
    while (i < nb) {
        var b = (i < bits.length ? 1 - bits[i] : 1) + carry
        out.push(b % 2)
        carry = b >= 2 ? 1 : 0
        i = i + 1
    }
    return out
}

// int.to_bytes: the byte values of the int (neg, bits) in `length` bytes, most
// significant first unless `little`. "neg" for a negative int without `signed`
// and "big" when it does not fit: CPython's two OverflowErrors.
function pynToBytes(neg, bits, length, little, signed) {
    var nb = length * 8
    var tw = []
    if (neg) {
        if (!signed) { return "neg" }
        // -2^(nb-1) is the most negative value that fits.
        var fits = bits.length < nb
        if (bits.length == nb && pynBitCount(bits) == 1) { fits = true }
        if (!fits) { return "big" }
        tw = pynTwos(bits, nb)
    } else {
        if (bits.length > (signed ? nb - 1 : nb)) { return "big" }
        var i = 0
        while (i < nb) {
            tw.push(i < bits.length ? bits[i] : 0)
            i = i + 1
        }
    }
    var out = []
    var k = 0
    while (k < length) {
        var at = little ? k : length - 1 - k
        var byte = 0
        var j = 7
        while (j >= 0) {
            byte = byte * 2 + tw[at * 8 + j]
            j = j - 1
        }
        out.push(byte)
        k = k + 1
    }
    return out
}

// int.from_bytes: {neg, bits} of the byte values, read as two's complement when
// `signed` and the top bit is set.
function pynFromBytes(bytes, little, signed) {
    var n = bytes.length
    var bits = []
    var k = 0
    while (k < n) {
        var v = little ? bytes[k] : bytes[n - 1 - k]
        var j = 0
        while (j < 8) {
            var b = v % 2
            bits.push(b)
            v = (v - b) / 2
            j = j + 1
        }
        k = k + 1
    }
    if (signed && n > 0 && bits[n * 8 - 1] == 1) {
        return {neg: true, bits: pynBitsTrim(pynTwos(bits, n * 8))}
    }
    return {neg: false, bits: pynBitsTrim(bits)}
}

// m * 2^k for an integral m below 2^54, by exact doubling or halving: every
// intermediate keeps m's significand, so it is exact whenever the result is.
function pynScale2(m, k) {
    var v = m
    var i = 0
    if (k > 0) {
        while (i < k) {
            v = v * 2
            i = i + 1
        }
        return v
    }
    while (i > k) {
        v = v / 2
        i = i - 1
    }
    return v
}

// float.as_integer_ratio: {neg, bits, dexp}, the numerator's magnitude bits over
// the denominator 2^dexp, in lowest terms. "inf" / "nan" for CPython's
// OverflowError / ValueError.
function pynRatio(x) {
    if (x !== x) { return "nan" }
    if (x - x != 0) { return "inf" }
    var neg = x < 0
    var m = neg ? -x : x
    var e = 0
    while (m != Math.floor(m)) {
        m = m * 2
        e = e + 1
    }
    var bits = pynNumBits(m)
    // An even numerator over a power of two is not in lowest terms.
    var lo = 0
    while (e > 0 && lo < bits.length && bits[lo] == 0) {
        lo = lo + 1
        e = e - 1
    }
    var out = []
    var i = lo
    while (i < bits.length) {
        out.push(bits[i])
        i = i + 1
    }
    return {neg: neg, bits: out, dexp: e}
}

var pynHexDigits = "0123456789abcdef"

// float.hex, CPython's float_hex: frexp's mantissa shifted so a normal value
// leads with 1 and a subnormal with 0 at exponent -1022, then 13 hex digits.
function pynFloatHex(x) {
    if (x !== x) { return "nan" }
    if (x - x != 0) { return x > 0 ? "inf" : "-inf" }
    if (x == 0) { return 1 / x < 0 ? "-0x0.0p+0" : "0x0.0p+0" }
    var m = x < 0 ? -x : x
    var e = 0
    while (m >= 1) {
        m = m / 2
        e = e + 1
    }
    while (m < 0.5) {
        m = m * 2
        e = e - 1
    }
    var shift = 1
    if (-1021 - e > 0) { shift = 1 - (-1021 - e) }
    m = pynScale2(m, shift)
    e = e - shift
    var d = Math.floor(m)
    var s = pynHexDigits.charAt(d) + "."
    m = m - d
    var i = 0
    while (i < 13) {
        m = m * 16
        d = Math.floor(m)
        s = s + pynHexDigits.charAt(d)
        m = m - d
        i = i + 1
    }
    var ex = e < 0 ? "-" + (-e) : "+" + e
    return (x < 0 ? "-0x" : "0x") + s + "p" + ex
}

// The value of one hex digit, or -1.
function pynHexVal(c) {
    var i = pynHexDigits.indexOf(c.toLowerCase())
    return i
}

function pynIsSpace(c) { return c == " " || c == "\t" || c == "\n" || c == "\r" || c == "\f" || c == "\v" }

// float.fromhex: the double nearest the hexadecimal string (ties to even), or
// "invalid" / "overflow" for CPython's ValueError / OverflowError. The digits'
// bits are rounded to the 53 (fewer for a subnormal) the result can hold.
function pynFromHex(src) {
    var s = src
    var a = 0
    var b = s.length
    while (a < b && pynIsSpace(s.charAt(a))) { a = a + 1 }
    while (b > a && pynIsSpace(s.charAt(b - 1))) { b = b - 1 }
    s = s.substring(a, b)
    var neg = false
    var p = 0
    if (s.charAt(0) == "-" || s.charAt(0) == "+") {
        neg = s.charAt(0) == "-"
        p = 1
    }
    var word = s.substring(p).toLowerCase()
    if (word == "inf" || word == "infinity") { return neg ? -1 / 0 : 1 / 0 }
    if (word == "nan") { return 0 / 0 }
    if (word.length >= 2 && word.charAt(0) == "0" && word.charAt(1) == "x") { p = p + 2 }
    // The digits, most significant first, and how many follow the point.
    var digs = []
    var frac = 0
    var seenPoint = false
    while (p < s.length) {
        var c = s.charAt(p)
        if (c == "." && !seenPoint) {
            seenPoint = true
        } else {
            var hv = pynHexVal(c)
            if (hv < 0) { break }
            digs.push(hv)
            if (seenPoint) { frac = frac + 1 }
        }
        p = p + 1
    }
    if (digs.length == 0) { return "invalid" }
    var pexp = 0
    if (p < s.length && (s.charAt(p) == "p" || s.charAt(p) == "P")) {
        p = p + 1
        var eneg = false
        if (p < s.length && (s.charAt(p) == "-" || s.charAt(p) == "+")) {
            eneg = s.charAt(p) == "-"
            p = p + 1
        }
        var nd = 0
        while (p < s.length && s.charAt(p) >= "0" && s.charAt(p) <= "9") {
            // Past any double's range the exact value no longer matters.
            if (pexp < 100000000) { pexp = pexp * 10 + (s.charCodeAt(p) - 48) }
            nd = nd + 1
            p = p + 1
        }
        if (nd == 0) { return "invalid" }
        if (eneg) { pexp = -pexp }
    }
    if (p != s.length) { return "invalid" }
    // The value's bits, most significant first, without leading zeros.
    var bits = []
    var i = 0
    while (i < digs.length) {
        var pw = 8
        while (pw >= 1) {
            var bit = Math.floor(digs[i] / pw) % 2
            if (bits.length > 0 || bit == 1) { bits.push(bit) }
            pw = pw / 2
        }
        i = i + 1
    }
    var zero = neg ? 0 / (0 - 1) : 0
    if (bits.length == 0) { return zero }
    // The leading bit's binary exponent, and how many bits the result keeps.
    var lead = bits.length - 1 + pexp - 4 * frac
    if (lead > 1024) { return "overflow" }
    // Below half the smallest subnormal everything rounds to zero.
    if (lead < -1075) { return zero }
    var keep = 53
    if (lead < -1022) { keep = lead + 1075 }
    var q = 0
    i = 0
    while (i < keep && i < bits.length) {
        q = q * 2 + bits[i]
        i = i + 1
    }
    while (i < keep) {
        q = q * 2
        i = i + 1
    }
    // Round to nearest, ties to even, on the bits past the kept ones.
    if (keep < bits.length && bits[keep] == 1) {
        var sticky = false
        var r = keep + 1
        while (r < bits.length) {
            if (bits[r] == 1) { sticky = true }
            r = r + 1
        }
        if (sticky || q % 2 == 1) { q = q + 1 }
    }
    var v = pynScale2(q, lead - keep + 1)
    if (v - v != 0) { return "overflow" }
    return neg ? -v : v
}

// round(i, -k) on the decimal DIGITS d of |i| (no sign, no leading zero) for
// k > 0: the digits of the nearest multiple of 10^k, a tie going to the even
// multiple. sticky says the true value lies just above |i| (a float's dropped
// fraction), which breaks a tie upward. Go twin: pyRoundTens in
// abnf/jsrtpynum.go.
function pynRoundTens(d, k, sticky) {
    var cut = d.length - k
    if (cut < 0) { return "0" }
    var head = d.substring(0, cut)
    var tail = d.substring(cut, d.length)
    // tail against the half, 5 followed by k-1 zeros.
    var c = 0
    if (tail.charAt(0) > "5") { c = 1 }
    if (tail.charAt(0) < "5") { c = -1 }
    var i = 1
    while (c == 0 && i < k) {
        if (tail.charAt(i) != "0") { c = 1 }
        i = i + 1
    }
    var last = head.length > 0 ? head.charCodeAt(head.length - 1) - 48 : 0
    if (c > 0 || (c == 0 && (sticky || last % 2 == 1))) { head = pynDecInc(head) }
    if (head == "" || head == "0") { return "0" }
    var out = head
    var z = 0
    while (z < k) {
        out = out + "0"
        z = z + 1
    }
    return out
}

// The decimal digit string d plus one ("" is zero).
function pynDecInc(d) {
    var out = ""
    var j = d.length - 1
    while (j >= 0 && d.charAt(j) == "9") {
        out = "0" + out
        j = j - 1
    }
    if (j < 0) { return "1" + out }
    return d.substring(0, j) + String.fromCharCode(d.charCodeAt(j) + 1) + out
}
