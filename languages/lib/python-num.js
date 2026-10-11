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

function pynIsSpace(c) { return c == " " || c == "\t" || c == "\n" || c == "\r" || c == "\f" || c == "\v" }

// float.fromhex: the double nearest the hexadecimal string (ties to even), or
// "invalid" / "overflow" for CPython's ValueError / OverflowError. The numeral
// itself is lib/hex-float.js's.
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
    var r = anytype
    r = hexFloatScan(s, p)
    if (r === null || r["end"] != s.length) { return "invalid" }
    var v = r["v"]
    if (v - v != 0) { return "overflow" }
    if (v == 0) { return neg ? 0 / (0 - 1) : 0 }
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

// ---- hash() ----
//
// CPython hashes in Py_uhash_t: 64-bit unsigned arithmetic that wraps. A double
// holds 53 bits, so a 64-bit word here is four 16-bit limbs, least significant
// first; a product of two limbs is below 2^32 and a column of four of them plus
// a carry below 2^35, so every step is exact. Go twin: the pyHash* functions in
// abnf/jsrtpynum.go, on uint64.

// The word of a non-negative integral double.
function pynWNum(n) {
    var out = []
    var i = 0
    while (i < 4) {
        var q = Math.floor(n / 65536)
        out.push(n - q * 65536)
        n = q
        i = i + 1
    }
    return out
}
// The word of four limbs written most significant first.
function pynW(a3, a2, a1, a0) { return [a0, a1, a2, a3] }
function pynWAdd(a, b) {
    var out = []
    var c = 0
    var i = 0
    while (i < 4) {
        var s = a[i] + b[i] + c
        c = s >= 65536 ? 1 : 0
        out.push(s - c * 65536)
        i = i + 1
    }
    return out
}
function pynWMul(a, b) {
    var out = []
    var c = 0
    var k = 0
    while (k < 4) {
        var s = c
        var i = 0
        while (i <= k) {
            s = s + a[i] * b[k - i]
            i = i + 1
        }
        c = Math.floor(s / 65536)
        out.push(s - c * 65536)
        k = k + 1
    }
    return out
}
function pynWNeg(a) {
    return pynWAdd([65535 - a[0], 65535 - a[1], 65535 - a[2], 65535 - a[3]], [1, 0, 0, 0])
}
function pynWXor(a, b) { return [a[0] ^ b[0], a[1] ^ b[1], a[2] ^ b[2], a[3] ^ b[3]] }
function pynWOr(a, b) { return [a[0] | b[0], a[1] | b[1], a[2] | b[2], a[3] | b[3]] }
function pynWAnd(a, b) { return [a[0] & b[0], a[1] & b[1], a[2] & b[2], a[3] & b[3]] }
// a shifted left (k > 0) or right (k < 0) by |k| < 64 bits.
function pynWShift(a, k) {
    var out = []
    var i = 0
    if (k >= 0) {
        var q = Math.floor(k / 16)
        var s = k - q * 16
        while (i < 4) {
            var hi = i - q >= 0 ? a[i - q] : 0
            var lo = i - q - 1 >= 0 ? a[i - q - 1] : 0
            out.push(((hi << s) & 65535) | (lo >> (16 - s)))
            i = i + 1
        }
        return out
    }
    var q2 = Math.floor((0 - k) / 16)
    var s2 = 0 - k - q2 * 16
    while (i < 4) {
        var lo2 = i + q2 < 4 ? a[i + q2] : 0
        var hi2 = i + q2 + 1 < 4 ? a[i + q2 + 1] : 0
        out.push((lo2 >> s2) | ((hi2 << (16 - s2)) & 65535))
        i = i + 1
    }
    return out
}
function pynWEq(a, b) { return a[0] == b[0] && a[1] == b[1] && a[2] == b[2] && a[3] == b[3] }
function pynWGe(a, b) {
    var i = 3
    while (i >= 0) {
        if (a[i] != b[i]) { return a[i] > b[i] }
        i = i - 1
    }
    return true
}
var pynWP = pynW(8191, 65535, 65535, 65535)
var pynWM1 = pynW(65535, 65535, 65535, 65535)
// The modulus 2^61 - 1 taken once from a word below 2^62.
function pynWModP(x) {
    var r = pynWAdd(pynWAnd(x, pynWP), pynWShift(x, -61))
    if (pynWGe(r, pynWP)) { r = pynWAdd(r, pynWNeg(pynWP)) }
    return r
}
// -1 is the error return of CPython's hash slots, so a hash of -1 is -2.
function pynWFix(x) {
    if (pynWEq(x, pynWM1)) { return pynW(65535, 65535, 65535, 65534) }
    return x
}

// The word of an int's two's complement, from its sign and magnitude bits (the
// low 64 bits).
function pynWOfInt(neg, bits) {
    var w = [0, 0, 0, 0]
    var i = 0
    while (i < bits.length && i < 64) {
        if (bits[i] == 1) { w[Math.floor(i / 16)] = w[Math.floor(i / 16)] + pynScale2(1, i - Math.floor(i / 16) * 16) }
        i = i + 1
    }
    if (neg) { return pynWNeg(w) }
    return w
}
// Does the int fit a signed 64-bit word (-2^63 is the one magnitude of 64 bits)?
function pynFitsI64(neg, bits) {
    if (bits.length < 64) { return true }
    return neg && bits.length == 64 && pynBitCount(bits) == 1
}
// A word read as a signed hash: {neg, bits}.
function pynWSigned(x) {
    var neg = x[3] >= 32768
    var m = neg ? pynWNeg(x) : x
    var bits = []
    var i = 0
    while (i < 4) {
        var v = m[i]
        var j = 0
        while (j < 16) {
            bits.push(v % 2)
            v = Math.floor(v / 2)
            j = j + 1
        }
        i = i + 1
    }
    return {neg: neg, bits: pynBitsTrim(bits)}
}

// hash(int): the magnitude modulo 2^61 - 1, signed (long_hash).
function pynHashInt(neg, bits) {
    var x = [0, 0, 0, 0]
    var i = bits.length - 1
    while (i >= 0) {
        x = pynWShift(x, 1)
        if (bits[i] == 1) { x[0] = x[0] + 1 }
        if (pynWGe(x, pynWP)) { x = pynWAdd(x, pynWNeg(pynWP)) }
        i = i - 1
    }
    if (neg) { x = pynWNeg(x) }
    return pynWFix(x)
}

// hash(float) for a finite x (_Py_HashDouble): the value modulo 2^61 - 1, which
// is the int's hash for an integral x. The mantissa enters 28 bits at a time and
// the binary exponent as a rotation, since 2^61 = 1 modulo 2^61 - 1.
function pynHashFloat(v) {
    var m = v < 0 ? 0 - v : v
    if (m == 0) { return [0, 0, 0, 0] }
    var e = 0
    while (m >= 1) {
        m = m / 2
        e = e + 1
    }
    while (m < 0.5) {
        m = m * 2
        e = e - 1
    }
    var x = [0, 0, 0, 0]
    while (m != 0) {
        x = pynWOr(pynWAnd(pynWShift(x, 28), pynWP), pynWShift(x, -33))
        m = m * 268435456
        e = e - 28
        var y = Math.floor(m)
        m = m - y
        x = pynWAdd(x, pynWNum(y))
        if (pynWGe(x, pynWP)) { x = pynWAdd(x, pynWNeg(pynWP)) }
    }
    if (e >= 0) { e = e % 61 } else { e = 60 - (-1 - e) % 61 }
    x = pynWOr(pynWAnd(pynWShift(x, e), pynWP), pynWShift(x, e - 61))
    if (v < 0) { x = pynWNeg(x) }
    return pynWFix(x)
}

// hash(complex): the real part's hash plus 1000003 times the imaginary part's.
function pynHashCplx(hr, hi) { return pynWFix(pynWAdd(hr, pynWMul(hi, pynWNum(1000003)))) }

// hash(tuple), tuplehash: xxHash's round over the items' hashes.
function pynHashTuple(lanes) {
    var p1 = pynW(40503, 31153, 34283, 51847)
    var p2 = pynW(49842, 44605, 10196, 60239)
    var p5 = pynW(10196, 60207, 5718, 26565)
    var acc = p5
    var i = 0
    while (i < lanes.length) {
        acc = pynWAdd(acc, pynWMul(lanes[i], p2))
        acc = pynWOr(pynWShift(acc, 31), pynWShift(acc, -33))
        acc = pynWMul(acc, p1)
        i = i + 1
    }
    acc = pynWAdd(acc, pynWXor(pynWNum(lanes.length), pynWXor(p5, pynWNum(3527539))))
    if (pynWEq(acc, pynWM1)) { return pynWNum(1546275796) }
    return acc
}

// hash(frozenset), frozenset_hash: the members' shuffled hashes XORed (so the
// order cannot matter), then the size and a final dispersal.
function pynHashFrozen(lanes) {
    var h = [0, 0, 0, 0]
    var i = 0
    while (i < lanes.length) {
        var l = lanes[i]
        h = pynWXor(h, pynWMul(pynWXor(pynWXor(l, pynWNum(89869747)), pynWShift(l, 16)), pynWNum(3644798167)))
        i = i + 1
    }
    h = pynWXor(h, pynWMul(pynWNum(lanes.length + 1), pynWNum(1927868237)))
    h = pynWXor(h, pynWXor(pynWShift(h, -11), pynWShift(h, -25)))
    h = pynWAdd(pynWMul(h, pynWNum(69069)), pynWNum(907133923))
    if (pynWEq(h, pynWM1)) { return pynWNum(590923713) }
    return h
}

// hash(str) and hash(bytes): CPython's is SipHash under a per-process random
// key, so no value is reproducible; this is a fixed polynomial over the code
// units instead, keeping only what is guaranteed - equal strings hash equal, and
// hash("") is 0.
function pynHashUnits(units) {
    var h = 0
    var i = 0
    while (i < units.length) {
        h = (h * 31 + units[i]) % 2147483647
        i = i + 1
    }
    return pynWNum(h)
}
