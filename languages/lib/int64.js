// int64.js -- exact 64-bit integer arithmetic on a pair of 32-bit halves.
//
// Included by lib/interp-core.js, and directly by bash-interpreter.abnf, whose
// arithmetic is bash's 64-bit intmax_t.

// ----- Exact 64-bit integers (shared) -----
// A JS number carries 53 bits exactly, so an int64 value above 2^53 cannot be held
// in one: 9223372036854775807 reads back as 9223372036854776000 and
// int64max + 1 answers -1 instead of -9223372036854775808. The statically typed
// languages need the real answer, so a value that does not fit is carried as a pair
// of UNSIGNED 32-bit halves {h, l} and every operator below works on that pair.
//
// Signedness and WIDTH are deliberately not part of the pair - they belong to the
// sized-integer box a language grammar puts around it, exactly as the {__flo} box
// carries floatness. The same pair therefore serves int8 through uint64.
//
// This code must run byte-identically under goja, the frozen MetaJS engine and
// node - so no BigInt, no for..in, no splice, no string.split appears here.

function i64Make(h, l) { return {h: h >>> 0, l: l >>> 0} }

// i64FromNum truncates toward zero and wraps modulo 2^64, which is what every
// Go/Java/C# conversion to a 64-bit integer type does.
function i64FromNum(n) {
    if (n !== n) { return i64Make(0, 0) }
    var neg = n < 0
    var a = neg ? -n : n
    a = Math.floor(a)
    // Reduce modulo 2^64 before splitting: a bigger double would give a
    // meaningless high half.
    if (a >= 18446744073709551616) { a = a - Math.floor(a / 18446744073709551616) * 18446744073709551616 }
    var h = Math.floor(a / 4294967296)
    var l = a - h * 4294967296
    var v = i64Make(h, l)
    if (neg) { v = i64Neg(v) }
    return v
}

// i64ToNumS / i64ToNumU are the nearest double. Exact while |value| <= 2^53.
function i64ToNumS(x) {
    if ((x.h & 0x80000000) !== 0) {
        var n = i64Neg(x)
        return -(n.h * 4294967296 + n.l)
    }
    return x.h * 4294967296 + x.l
}
function i64ToNumU(x) { return x.h * 4294967296 + x.l }

// ----- add / sub / neg -----
function i64Add(a, b) {
    var l = a.l + b.l
    var carry = l >= 4294967296 ? 1 : 0
    var h = a.h + b.h + carry
    return i64Make(h >>> 0, (l >>> 0))
}
function i64Not(a) { return i64Make(~a.h, ~a.l) }
function i64Neg(a) { return i64Add(i64Not(a), i64Make(0, 1)) }
function i64Sub(a, b) { return i64Add(a, i64Neg(b)) }

// ----- multiply -----
// Four 16-bit limbs per operand; each partial product is below 2^32 and every
// intermediate stays below 2^53, so the double arithmetic is exact.
function i64Mul(a, b) {
    var a0 = a.l & 0xffff, a1 = a.l >>> 16, a2 = a.h & 0xffff, a3 = a.h >>> 16
    var b0 = b.l & 0xffff, b1 = b.l >>> 16, b2 = b.h & 0xffff, b3 = b.h >>> 16
    var c0 = a0 * b0
    var c1 = (c0 >>> 16) + a1 * b0
    var c2 = c1 >>> 16
    c1 = (c1 & 0xffff) + a0 * b1
    c2 = c2 + (c1 >>> 16) + a2 * b0
    var c3 = c2 >>> 16
    c2 = (c2 & 0xffff) + a1 * b1
    c3 = c3 + (c2 >>> 16)
    c2 = (c2 & 0xffff) + a0 * b2
    c3 = c3 + (c2 >>> 16)
    c3 = c3 + a3 * b0 + a2 * b1 + a1 * b2 + a0 * b3
    return i64Make(((c2 & 0xffff) | ((c3 & 0xffff) << 16)) >>> 0,
                   ((c0 & 0xffff) | ((c1 & 0xffff) << 16)) >>> 0)
}

// ----- shifts (count already masked to 0..63 by the caller) -----
function i64Shl(a, n) {
    n = n & 63
    if (n === 0) { return i64Make(a.h, a.l) }
    if (n < 32) { return i64Make((a.h << n) | (a.l >>> (32 - n)), a.l << n) }
    return i64Make(a.l << (n - 32), 0)
}
function i64ShrU(a, n) {
    n = n & 63
    if (n === 0) { return i64Make(a.h, a.l) }
    if (n < 32) { return i64Make(a.h >>> n, (a.l >>> n) | (a.h << (32 - n))) }
    return i64Make(0, a.h >>> (n - 32))
}
function i64ShrS(a, n) {
    n = n & 63
    if (n === 0) { return i64Make(a.h, a.l) }
    if (n < 32) { return i64Make(a.h >> n, (a.l >>> n) | (a.h << (32 - n))) }
    return i64Make(a.h >> 31, a.h >> (n - 32))
}

// ----- bitwise -----
function i64And(a, b) { return i64Make(a.h & b.h, a.l & b.l) }
function i64Or(a, b)  { return i64Make(a.h | b.h, a.l | b.l) }
function i64Xor(a, b) { return i64Make(a.h ^ b.h, a.l ^ b.l) }

// ----- comparison -----
function i64IsZero(a) { return a.h === 0 && a.l === 0 }
function i64Eq(a, b) { return a.h === b.h && a.l === b.l }
function i64CmpU(a, b) {
    if (a.h !== b.h) { return (a.h >>> 0) < (b.h >>> 0) ? -1 : 1 }
    if (a.l !== b.l) { return (a.l >>> 0) < (b.l >>> 0) ? -1 : 1 }
    return 0
}
function i64CmpS(a, b) {
    var an = (a.h & 0x80000000) !== 0
    var bn = (b.h & 0x80000000) !== 0
    if (an !== bn) { return an ? -1 : 1 }
    return i64CmpU(a, b)
}

// ----- division -----
// i64DivModSmall divides an UNSIGNED pair by a positive divisor below 2^31,
// walking the value in 16-bit chunks so every intermediate stays below 2^47.
// Returns {q: pair, r: number}.
function i64DivModSmall(a, d) {
    var parts = [a.h >>> 16, a.h & 0xffff, a.l >>> 16, a.l & 0xffff]
    var out = [0, 0, 0, 0]
    var rem = 0
    for (var i = 0; i < 4; i++) {
        var cur = rem * 65536 + parts[i]
        var q = Math.floor(cur / d)
        rem = cur - q * d
        out[i] = q
    }
    return {q: i64Make(((out[0] * 65536) + out[1]) >>> 0, ((out[2] * 65536) + out[3]) >>> 0), r: rem}
}

// i64DivModU is unsigned 64/64 by shift-and-subtract. Only reached when the
// divisor does not fit the small path, so the 64 iterations are rare.
function i64DivModU(a, b) {
    if (i64IsZero(b)) { return {q: i64Make(0, 0), r: i64Make(0, 0)} }
    if (b.h === 0 && (b.l >>> 0) < 2147483648 && b.l !== 0) {
        var s = i64DivModSmall(a, b.l >>> 0)
        return {q: s.q, r: i64Make(0, s.r)}
    }
    var q = i64Make(0, 0)
    var r = i64Make(0, 0)
    for (var i = 63; i >= 0; i--) {
        r = i64Shl(r, 1)
        var bit = (i >= 32) ? ((a.h >>> (i - 32)) & 1) : ((a.l >>> i) & 1)
        r = i64Or(r, i64Make(0, bit))
        if (i64CmpU(r, b) >= 0) {
            r = i64Sub(r, b)
            q = i64Or(q, i64Shl(i64Make(0, 1), i))
        }
    }
    return {q: q, r: r}
}

// i64DivModS is Go's / and % on signed 64-bit: truncation toward zero, so the
// remainder takes the DIVIDEND's sign. Division of the most negative value by
// -1 wraps back to itself, which is what the hardware does.
function i64DivModS(a, b) {
    var an = (a.h & 0x80000000) !== 0
    var bn = (b.h & 0x80000000) !== 0
    var ua = an ? i64Neg(a) : a
    var ub = bn ? i64Neg(b) : b
    var d = i64DivModU(ua, ub)
    var q = d.q
    var r = d.r
    if (an !== bn) { q = i64Neg(q) }
    if (an) { r = i64Neg(r) }
    return {q: q, r: r}
}

// ----- text -----
function i64StrU(a) {
    if (i64IsZero(a)) { return "0" }
    var s = ""
    var v = a
    while (!i64IsZero(v)) {
        var d = i64DivModSmall(v, 1000000000)
        v = d.q
        if (i64IsZero(v)) { s = "" + d.r + s }
        else {
            var chunk = "" + d.r
            while (chunk.length < 9) { chunk = "0" + chunk }
            s = chunk + s
        }
    }
    return s
}
function i64StrS(a) {
    if ((a.h & 0x80000000) !== 0) { return "-" + i64StrU(i64Neg(a)) }
    return i64StrU(a)
}

// i64FromStr reads a decimal digit run (no sign, no separators - the caller has
// already stripped them). Overflow wraps, like the arithmetic above.
function i64FromStr(s) {
    var v = i64Make(0, 0)
    var ten = i64Make(0, 10)
    for (var i = 0; i < s.length; i++) {
        var c = s.charCodeAt(i)
        if (c < 48 || c > 57) { continue }
        v = i64Add(i64Mul(v, ten), i64Make(0, c - 48))
    }
    return v
}
// i64FromRadix reads a digit run in base 2, 8 or 16.
function i64FromRadix(s, base) {
    var v = i64Make(0, 0)
    var bb = i64Make(0, base)
    for (var i = 0; i < s.length; i++) {
        var c = s.charCodeAt(i)
        var d = -1
        if (c >= 48 && c <= 57) { d = c - 48 }
        else if (c >= 97 && c <= 122) { d = c - 87 }
        else if (c >= 65 && c <= 90) { d = c - 55 }
        if (d < 0 || d >= base) { continue }
        v = i64Add(i64Mul(v, bb), i64Make(0, d))
    }
    return v
}

// ----- width -----
// i64Trunc reduces a pair to `w` bits and re-extends it: sign extension for a
// signed width, zero extension for an unsigned one. w is 8, 16, 32 or 64.
function i64Trunc(a, w, unsigned) {
    if (w >= 64) { return a }
    if (w === 32) {
        if (unsigned) { return i64Make(0, a.l) }
        return i64Make((a.l & 0x80000000) !== 0 ? 0xffffffff : 0, a.l)
    }
    var mask = w === 8 ? 0xff : 0xffff
    var lo = a.l & mask
    if (unsigned) { return i64Make(0, lo) }
    var sign = w === 8 ? 0x80 : 0x8000
    if ((lo & sign) !== 0) { return i64Make(0xffffffff, (lo | ~mask) >>> 0) }
    return i64Make(0, lo)
}

// i64Fits53 says whether a pair's SIGNED value lies in [-2^53, 2^53], the range a
// JS number holds exactly. It is the test the sized-integer box below uses to
// decide between a plain number and a box, and it has to read the pair rather
// than the double: converting first would round 2^53 + 1 down to 2^53 and answer
// yes for a value that does not fit.
function i64Fits53(p) {
    var q = ((p.h & 0x80000000) !== 0) ? i64Neg(p) : p
    if ((q.h & 0x80000000) !== 0) { return false }      // only -2^63 negates to itself
    if (q.h < 0x200000) { return true }
    return q.h === 0x200000 && q.l === 0
}
