// python-num.js -- the bit and double algorithms behind int's and float's own
// methods (bit_length, bit_count, to_bytes, from_bytes, float.hex,
// float.fromhex, float.as_integer_ratio) and math's frexp, ldexp, modf, ulp,
// nextafter, hypot and dist, for python-interpreter.abnf (include)
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

// A bigint magnitude crosses between its DECIMAL limbs (little-endian, base
// LB: 10000 in python-rt.metajs, 1e7 in the interpreter) and its bits sixteen
// bits per pass over the limbs: one short division by 2^16, or one
// multiply-add by it, keeps every intermediate below 2^53 for either base.

// The magnitude bits of LIMBS (left untouched); zero answers the empty array.
function pynLimbsToBits(limbs, lb) {
    var ls = []
    var i = 0
    while (i < limbs.length) {
        ls.push(limbs[i])
        i = i + 1
    }
    while (ls.length > 0 && ls[ls.length - 1] == 0) { ls.pop() }
    var bits = []
    while (ls.length > 0) {
        var rem = 0
        var j = ls.length - 1
        while (j >= 0) {
            var cur = rem * lb + ls[j]
            var q = Math.floor(cur / 65536)
            ls[j] = q
            rem = cur - q * 65536
            j = j - 1
        }
        while (ls.length > 0 && ls[ls.length - 1] == 0) { ls.pop() }
        var k = 0
        while (k < 16) {
            var b = rem % 2
            bits.push(b)
            rem = (rem - b) / 2
            k = k + 1
        }
    }
    while (bits.length > 0 && bits[bits.length - 1] == 0) { bits.pop() }
    return bits
}

// LS = LS * m + a, IN PLACE, for m * LB below 2^53.
function pynLimbsMulAdd(ls, m, a, lb) {
    var carry = a
    var i = 0
    while (i < ls.length) {
        var cur = ls[i] * m + carry
        var q = Math.floor(cur / lb)
        ls[i] = cur - q * lb
        carry = q
        i = i + 1
    }
    while (carry > 0) {
        var q2 = Math.floor(carry / lb)
        ls.push(carry - q2 * lb)
        carry = q2
    }
}

// The decimal limbs of a magnitude given as BITS; zero answers [].
function pynBitsToLimbs(bits, lb) {
    var ls = []
    var top = bits.length - 1
    while (top >= 0) {
        var g = 0
        var m = 1
        var low = top - 15
        if (low < 0) { low = 0 }
        var j = top
        while (j >= low) {
            g = g * 2 + bits[j]
            m = m * 2
            j = j - 1
        }
        pynLimbsMulAdd(ls, m, g, lb)
        top = low - 1
    }
    return ls
}

// The digits of a magnitude given as BITS in base 2, 8 or 16; "0" for zero.
function pynBitsRadix(bits, base) {
    var w = base == 16 ? 4 : (base == 8 ? 3 : 1)
    if (bits.length == 0) { return "0" }
    var out = ""
    var g = Math.floor((bits.length - 1) / w)
    while (g >= 0) {
        var d = 0
        var k = w - 1
        while (k >= 0) {
            var at = g * w + k
            d = d * 2 + (at < bits.length ? bits[at] : 0)
            k = k - 1
        }
        out = out + "0123456789abcdef".charAt(d)
        g = g - 1
    }
    return out
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

// CPython's float_divmod for a nonzero divisor. The remainder is fmod (JS %, exact)
// moved to the divisor's sign, and the quotient is the exact (x - fmod) / y,
// rounded to the nearest integer, so q*y + r stays x where floor(x / y) rounds
// past it (1.0 // 0.1 is 9.0, not 10.0). A zero remainder takes the divisor's
// sign, a zero quotient the sign of x / y. Go twin: pyFloDivmod in abnf/jsrt.go.
function pynFloMod(x, y) {
    var m = x % y
    if (m != 0) {
        if ((y < 0) != (m < 0)) { return m + y }
        return m
    }
    if (y < 0) { return 0 / (0 - 1) }
    return 0
}
function pynFloDiv(x, y) {
    var m = x % y
    var d = (x - m) / y
    if (m != 0 && (y < 0) != (m < 0)) { d = d - 1 }
    if (d != 0) {
        var f = Math.floor(d)
        if (d - f > 0.5) { return f + 1 }
        return f
    }
    var q = x / y
    if (q < 0 || 1 / q < 0) { return 0 / (0 - 1) }
    return 0
}

// pow(a, b, m) for three ints and a nonzero m, over the engine's own binary
// operator OP(op, l, r), so an arbitrary precision operand stays exact. As in
// CPython the work is done modulo |m|, a negative exponent first inverts the
// base (extended Euclid), and a nonzero answer takes m's sign. null when the
// base has no inverse. Go twin: js_pypow in abnf/jsrt.go (math/big).
function pynPowMod(a, b, m, op) {
    var neg = op("<", m, 0) === true
    var mm = anytype
    mm = m
    if (neg) { mm = op("-", 0, m) }
    var base = anytype
    base = op("%", a, mm)
    var e = anytype
    e = b
    if (op("<", e, 0) === true) {
        var r0 = anytype
        r0 = base
        var r1 = anytype
        r1 = mm
        var s0 = anytype
        s0 = 1
        var s1 = anytype
        s1 = 0
        while (op("!=", r1, 0) === true) {
            var q = anytype
            q = op("//", r0, r1)
            var rn = anytype
            rn = op("-", r0, op("*", q, r1))
            r0 = r1
            r1 = rn
            var st = anytype
            st = op("-", s0, op("*", q, s1))
            s0 = s1
            s1 = st
        }
        if (op("!=", r0, 1) === true) { return null }
        base = op("%", s0, mm)
        e = op("-", 0, e)
    }
    var acc = anytype
    acc = op("%", 1, mm)
    while (op(">", e, 0) === true) {
        if (op("==", op("%", e, 2), 1) === true) { acc = op("%", op("*", acc, base), mm) }
        e = op("//", e, 2)
        base = op("%", op("*", base, base), mm)
    }
    if (neg && op("!=", acc, 0) === true) { return op("+", acc, m) }
    return acc
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
    var p5 = pynW(10196, 60207, 5718, 26565)
    var acc = pynWAdd(pynHashRounds(lanes), pynWXor(pynWNum(lanes.length), pynWXor(p5, pynWNum(3527539))))
    if (pynWEq(acc, pynWM1)) { return pynWNum(1546275796) }
    return acc
}

// hash(slice), slice_hash: tuplehash's rounds over (start, stop, step), with no
// length mixed in.
function pynHashSlice(lanes) {
    var acc = pynHashRounds(lanes)
    if (pynWEq(acc, pynWM1)) { return pynWNum(1546275796) }
    return acc
}

// xxHash's round over the lanes, from its PRIME_5 seed: the body tuplehash and
// slice_hash share.
function pynHashRounds(lanes) {
    var p1 = pynW(40503, 31153, 34283, 51847)
    var p2 = pynW(49842, 44605, 10196, 60239)
    var acc = pynW(10196, 60207, 5718, 26565)
    var i = 0
    while (i < lanes.length) {
        acc = pynWAdd(acc, pynWMul(lanes[i], p2))
        acc = pynWOr(pynWShift(acc, 31), pynWShift(acc, -33))
        acc = pynWMul(acc, p1)
        i = i + 1
    }
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

// ----- math's functions at the double level -----
//
// Exact by construction where CPython's are exact (frexp, ldexp, modf, ulp,
// nextafter): scaling by two is exact while nothing enters the subnormals, and
// the one rounding into them is done once, half to even. pynVecNorm is
// CPython's vector_norm, the algorithm behind hypot and dist, with Dekker's
// exact product in place of its fma. Go twins in abnf/jsrtpymath.go.

// x = m * 2^e with 0.5 <= |m| < 1; zero, an infinity and a NaN give (x, 0).
function pynFrexp(x) {
    if (x == 0 || x !== x || x - x != 0) { return {m: x, e: 0} }
    var neg = x < 0
    var a = neg ? -x : x
    var e = 0
    while (a >= 1) {
        a = a / 2
        e = e + 1
    }
    while (a < 0.5) {
        a = a * 2
        e = e - 1
    }
    return {m: neg ? -a : a, e: e}
}

// x * 2^n rounded once, or null when it overflows. n is already clamped by the
// caller to a range where the loops stay short.
function pynLdexp(x, n) {
    if (x == 0 || x !== x || x - x != 0) { return x }
    var f = pynFrexp(x)
    var t = f.e + n
    if (t > 1024) { return null }
    var m = f.m < 0 ? -f.m : f.m
    var r = 0
    if (t >= -1021) {
        r = pynScale2(m, t)
    } else if (t + 1074 >= -1) {
        var y = pynScale2(m, t + 1074)
        var k = Math.floor(y)
        var d = y - k
        if (d > 0.5 || (d == 0.5 && k % 2 == 1)) { k = k + 1 }
        r = pynScale2(k, 0 - 1074)
    }
    if (x < 0) { return r / (0 - 1) }
    return r
}

// modf: the fractional and the integral part, both with x's sign.
function pynModf(x) {
    if (x !== x) { return {f: x, i: x} }
    if (x - x != 0) { return {f: x > 0 ? 0 : 0 / (0 - 1), i: x} }
    var neg = x < 0 || (x == 0 && 1 / x < 0)
    var a = neg ? -x : x
    var ip = Math.floor(a)
    var fp = a - ip
    if (neg) { return {f: fp / (0 - 1), i: ip / (0 - 1)} }
    return {f: fp, i: ip}
}

// The value of the least significant bit of |x|.
function pynUlp(x) {
    if (x !== x) { return x }
    var a = x < 0 ? -x : x
    if (a - a != 0) { return a }
    var e = 0 - 1074
    if (a != 0) { e = pynFrexp(a).e - 53 }
    if (e < 0 - 1074) { e = 0 - 1074 }
    return pynScale2(1, e)
}

// The next double after x towards y.
function pynNextafter(x, y) {
    if (x !== x || y !== y) { return x + y }
    if (x == y) { return y }
    if (x == 0) {
        var t = pynScale2(1, 0 - 1074)
        if (y < 0) { return t / (0 - 1) }
        return t
    }
    var up = (y > x) == (x > 0)
    var a = x < 0 ? -x : x
    var r = 0
    if (a - a != 0) {
        r = pynScale2(9007199254740991, 971)
    } else {
        var f = pynFrexp(a)
        var e = f.e
        if (!up && f.m == 0.5) { e = e - 1 }
        e = e - 53
        if (e < 0 - 1074) { e = 0 - 1074 }
        var step = pynScale2(1, e)
        r = up ? a + step : a - step
    }
    if (x < 0) { return r / (0 - 1) }
    return r
}

// Dekker's exact product: a * b == hi + lo, what fma(a, b, -hi) answers for lo.
function pynTwoProd(a, b) {
    var p = a * b
    var t = 134217729 * a
    var ah = t - (t - a)
    var al = a - ah
    t = 134217729 * b
    var bh = t - (t - b)
    var bl = b - bh
    return {hi: p, lo: ((ah * bh - p) + ah * bl + al * bh) + al * bl}
}

// CPython's vector_norm over the absolute values vec, mx their maximum, both
// finite.
function pynVecNorm(vec, mx) {
    if (mx == 0 || vec.length <= 1) { return mx }
    var me = pynFrexp(mx).e
    if (me < 0 - 1022) {
        var big = []
        var i = 0
        while (i < vec.length) {
            big.push(pynScale2(vec[i], 1022))
            i = i + 1
        }
        return pynVecNorm(big, pynScale2(mx, 1022)) * pynScale2(1, 0 - 1022)
    }
    var scale = pynScale2(1, 0 - me)
    var csum = 1
    var frac1 = 0
    var frac2 = 0
    var j = 0
    while (j < vec.length) {
        var x = vec[j] * scale
        var pr = pynTwoProd(x, x)
        var hi = csum + pr.hi
        frac2 = frac2 + ((csum - hi) + pr.hi)
        csum = hi
        frac1 = frac1 + pr.lo
        j = j + 1
    }
    var h = Math.sqrt(csum - 1 + (frac1 + frac2))
    var pr2 = pynTwoProd(-h, h)
    var hi2 = csum + pr2.hi
    frac2 = frac2 + ((csum - hi2) + pr2.hi)
    csum = hi2
    frac1 = frac1 + pr2.lo
    var c = csum - 1 + (frac1 + frac2)
    h = h + c / (2 * h)
    return h / scale
}

// ----- a set's iteration order -----
//
// CPython's iteration order of a set of ints: the slot order of its open-
// addressed table after the members are added in this order (setobject.c's
// set_add_entry, set_table_resize and set_insert_clean: ten linear probes,
// then the next slot by perturbation with five more bits of the hash), with
// hash(i) == i and hash(-1) == -2. Answers the member indexes in slot order,
// or null when a member is not an int or a bool: a str's hash is salted per
// run in CPython and the others are not modelled, so such a set keeps
// insertion order. A removed member leaves no dummy slot here, so after a
// removal the order is the one the remaining members would have had. Go twin:
// pySetOrder in abnf/jsrtpynum.go.
function pynSetOrder(vals) {
    var hi = []
    var lo = []
    var i = 0
    while (i < vals.length) {
        var e = anytype
        e = vals[i]
        var v = 0
        if (typeof e == "boolean") { v = e ? 1 : 0 }
        else if (typeof e == "number") { v = e }
        else { return null }
        if (v == 0 - 1) { v = 0 - 2 }
        if (v >= 0) {
            hi.push(Math.floor(v / 4294967296))
            lo.push(v % 4294967296)
        } else {
            var a = 0 - v
            var al = a % 4294967296
            var ah = Math.floor(a / 4294967296)
            lo.push(al == 0 ? 0 : 4294967296 - al)
            hi.push(al == 0 ? (4294967296 - ah) % 4294967296 : 4294967295 - ah)
        }
        i = i + 1
    }
    var size = 8
    var table = pynSlots(size)
    var k = 0
    while (k < vals.length) {
        pynSetPlace(table, size, hi[k], lo[k], k)
        if ((k + 1) * 5 >= (size - 1) * 3) {
            var minused = k + 1 > 50000 ? (k + 1) * 2 : (k + 1) * 4
            var ns = 8
            while (ns <= minused) { ns = ns * 2 }
            var nt = pynSlots(ns)
            var s = 0
            while (s < size) {
                if (table[s] >= 0) { pynSetPlace(nt, ns, hi[table[s]], lo[table[s]], table[s]) }
                s = s + 1
            }
            table = nt
            size = ns
        }
        k = k + 1
    }
    var out = []
    var t = 0
    while (t < size) {
        if (table[t] >= 0) { out.push(table[t]) }
        t = t + 1
    }
    return out
}

function pynSlots(n) {
    var t = []
    var i = 0
    while (i < n) {
        t.push(0 - 1)
        i = i + 1
    }
    return t
}

// The first free slot for a hash whose unsigned 64 bits are hi:lo.
function pynSetPlace(t, size, hi, lo, k) {
    var i = lo % size
    var phi = hi
    var plo = lo
    while (true) {
        if (t[i] < 0) {
            t[i] = k
            return
        }
        if (i + 9 <= size - 1) {
            var j = 1
            while (j <= 9) {
                if (t[i + j] < 0) {
                    t[i + j] = k
                    return
                }
                j = j + 1
            }
        }
        plo = Math.floor(plo / 32) + (phi % 32) * 134217728
        phi = Math.floor(phi / 32)
        i = (i * 5 + 1 + plo) % size
    }
}
