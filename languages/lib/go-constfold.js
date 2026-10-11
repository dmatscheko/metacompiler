// go-constfold.js -- Go's untyped-constant folder, shared by go-interpreter.abnf and
// go-to-llvm-ir.abnf through include(). Each grammar supplies the one function that
// differs between the halves, cMaterialize (a folded constant as a value or as IR),
// plus the literal thunks and everything else this file only calls.
//
// Go evaluates a constant expression EXACTLY and rounds ONCE, where the constant
// reaches a typed location: 0.1 + 0.2 is the double nearest 3/10 (0.3), not the
// sum of the two nearest doubles (0.30000000000000004), and (1 << 100) >> 98 is 4
// rather than the 0 a 64 bit accumulator answers. There is one numeric type in
// this value model, so the exactness cannot ride ON the value - it is a FRONT END
// fold, keyed by source position exactly like the grammars' convByPos and for the
// same reason: a thunk that crosses the tag stack comes back as a fresh function
// object under goja, so a table keyed by function identity finds nothing.
//
// A folded constant is {s, n, d, f}: the sign (-1/0/1), the non-negative
// magnitudes n and d (little-endian base-1e7 limb arrays; d is never zero and is
// always 1 for an integer constant), and f = "this is an untyped FLOAT constant".
// Go's constant kinds order int < float, so an operator mixing them answers a
// float and '/' on two INTEGER constants is integer division.
//
// What is deliberately NOT folded - each falls back to the unfolded path rather
// than being half-folded:
//   - a named constant the parse-time scope declines (see "NAMED constants" below).
//   - anything under a written-out conversion: float64(0.1) + float64(0.2) really
//     IS 0.30000000000000004 in Go, because each conversion rounds and the
//     arithmetic is then float64 arithmetic. tyAt already reports that case.
//   - '&' '|' '^' '&^' with a negative operand, a shift count over 4096, and any
//     division or remainder by zero (which is a compile error in Go, not a value).
// An intermediate wider than CB_LIMIT limbs also abandons the fold, so a
// pathological constant expression cannot make the front end quadratic.
var CB_BASE = 10000000
var CB_LIMIT = 700
var cbZeroF = 0
var cbInf = 1 / cbZeroF
function cbTrim(a) {
    var n = a.length
    while (n > 0 && a[n - 1] === 0) { n = n - 1 }
    if (n === a.length) { return a }
    var r = []
    var i = 0
    while (i < n) { r.push(a[i]); i = i + 1 }
    return r
}
function cbIsZero(a) { return a.length === 0 }
function cbFromSmall(v) {
    var r = []
    var x = v
    while (x > 0) {
        var q = Math.floor(x / CB_BASE)
        r.push(x - q * CB_BASE)
        x = q
    }
    return r
}
function cbOne() { return cbFromSmall(1) }
function cbEven(a) {
    if (cbIsZero(a)) { return true }
    return (a[0] - Math.floor(a[0] / 2) * 2) === 0
}
function cbCmp(a, b) {
    if (a.length !== b.length) { return a.length > b.length ? 1 : 0 - 1 }
    var i = a.length - 1
    while (i >= 0) {
        if (a[i] !== b[i]) { return a[i] > b[i] ? 1 : 0 - 1 }
        i = i - 1
    }
    return 0
}
function cbAdd(a, b) {
    var r = []
    var carry = 0
    var n = a.length > b.length ? a.length : b.length
    var i = 0
    while (i < n) {
        var s = carry + (i < a.length ? a[i] : 0) + (i < b.length ? b[i] : 0)
        if (s >= CB_BASE) { s = s - CB_BASE; carry = 1 } else { carry = 0 }
        r.push(s)
        i = i + 1
    }
    if (carry > 0) { r.push(carry) }
    return r
}
// a - b, with a >= b already established.
function cbSub(a, b) {
    var r = []
    var borrow = 0
    var i = 0
    while (i < a.length) {
        var s = a[i] - borrow - (i < b.length ? b[i] : 0)
        if (s < 0) { s = s + CB_BASE; borrow = 1 } else { borrow = 0 }
        r.push(s)
        i = i + 1
    }
    return cbTrim(r)
}
// Schoolbook, carrying inside the inner loop: one column is at most
// 1e7 + (1e7-1)^2 + 1e7, which a double still holds exactly.
function cbMul(a, b) {
    if (cbIsZero(a) || cbIsZero(b)) { return [] }
    var r = []
    var n = a.length + b.length
    var i = 0
    while (i < n) { r.push(0); i = i + 1 }
    i = 0
    while (i < a.length) {
        var carry = 0
        var ai = a[i]
        var j = 0
        while (j < b.length) {
            var t = r[i + j] + ai * b[j] + carry
            carry = Math.floor(t / CB_BASE)
            r[i + j] = t - carry * CB_BASE
            j = j + 1
        }
        var k = i + b.length
        while (carry > 0) {
            var t2 = r[k] + carry
            carry = Math.floor(t2 / CB_BASE)
            r[k] = t2 - carry * CB_BASE
            k = k + 1
        }
        i = i + 1
    }
    return cbTrim(r)
}
// Multiply / divide by a SMALL m: m * 1e7 has to stay under 2^53, so m < 9e8.
function cbMulSmall(a, m) {
    if (m === 0 || cbIsZero(a)) { return [] }
    var r = []
    var carry = 0
    var i = 0
    while (i < a.length) {
        var t = a[i] * m + carry
        carry = Math.floor(t / CB_BASE)
        r.push(t - carry * CB_BASE)
        i = i + 1
    }
    while (carry > 0) {
        var c2 = Math.floor(carry / CB_BASE)
        r.push(carry - c2 * CB_BASE)
        carry = c2
    }
    return cbTrim(r)
}
function cbDivSmall(a, m) {
    var r = []
    var i = 0
    while (i < a.length) { r.push(0); i = i + 1 }
    var rem = 0
    i = a.length - 1
    while (i >= 0) {
        var cur = rem * CB_BASE + a[i]
        var q = Math.floor(cur / m)
        r[i] = q
        rem = cur - q * m
        i = i - 1
    }
    return {q: cbTrim(r), r: rem}
}
// The number of significant BITS, counted 24 at a time so that the cost is
// linear in the limb count rather than in the value.
function cbBitLen(a) {
    if (cbIsZero(a)) { return 0 }
    var bits = 0
    var v = a
    while (true) {
        var d = cbDivSmall(v, 16777216)
        if (cbIsZero(d.q)) {
            var x = d.r
            while (x > 0) { bits = bits + 1; x = Math.floor(x / 2) }
            return bits
        }
        v = d.q
        bits = bits + 24
    }
}
function cbShl(a, k) {
    var v = a
    var left = k
    while (left >= 23) { v = cbMulSmall(v, 8388608); left = left - 23 }
    while (left > 0) { v = cbMulSmall(v, 2); left = left - 1 }
    return v
}
function cbShr(a, k) {
    var v = a
    var left = k
    while (left >= 24 && !cbIsZero(v)) { v = cbDivSmall(v, 16777216).q; left = left - 24 }
    while (left > 0 && !cbIsZero(v)) { v = cbDivSmall(v, 2).q; left = left - 1 }
    return v
}
// Long division one BIT at a time: no quotient-digit estimation to get wrong, and
// a constant expression is small enough that the bit loop is not worth avoiding.
function cbDivMod(a, b) {
    if (cbCmp(a, b) < 0) { return {q: [], r: a} }
    var shift = cbBitLen(a) - cbBitLen(b)
    var d = cbShl(b, shift)
    var rem = a
    var q = []
    var one = cbOne()
    var i = shift
    while (i >= 0) {
        q = cbMulSmall(q, 2)
        if (cbCmp(rem, d) >= 0) { rem = cbSub(rem, d); q = cbAdd(q, one) }
        d = cbDivSmall(d, 2).q
        i = i - 1
    }
    return {q: cbTrim(q), r: rem}
}
// Binary GCD: only halving, comparison and subtraction, so it needs no divisor
// estimation either. It is what keeps a rational's denominator from growing
// without bound across a chain of operators.
function cbGcd(a, b) {
    if (cbIsZero(a)) { return b }
    if (cbIsZero(b)) { return a }
    var x = a
    var y = b
    var shift = 0
    while (cbEven(x) && cbEven(y)) {
        x = cbDivSmall(x, 2).q
        y = cbDivSmall(y, 2).q
        shift = shift + 1
    }
    while (cbEven(x)) { x = cbDivSmall(x, 2).q }
    while (!cbIsZero(y)) {
        while (cbEven(y)) { y = cbDivSmall(y, 2).q }
        if (cbCmp(x, y) > 0) {
            var t = x
            x = y
            y = t
        }
        y = cbSub(y, x)
    }
    return cbShl(x, shift)
}
function cbToDec(a) {
    if (cbIsZero(a)) { return "0" }
    var s = "" + a[a.length - 1]
    var i = a.length - 2
    while (i >= 0) {
        var t = "" + a[i]
        while (t.length < 7) { t = "0" + t }
        s = s + t
        i = i - 1
    }
    return s
}
function cbFromDec(s) {
    var r = []
    var end = s.length
    while (end > 0) {
        var start = end - 7
        if (start < 0) { start = 0 }
        r.push(parseInt(s.substring(start, end), 10))
        end = start
    }
    return cbTrim(r)
}
function cbFromRadix(s, radix) {
    var r = []
    var i = 0
    while (i < s.length) {
        var cc = s.charCodeAt(i)
        var d = 0 - 1
        if (cc >= 48 && cc <= 57) { d = cc - 48 }
        else if (cc >= 97 && cc <= 122) { d = cc - 87 }
        else if (cc >= 65 && cc <= 90) { d = cc - 55 }
        i = i + 1
        if (d < 0 || d >= radix) { continue }
        r = cbMulSmall(r, radix)
        if (d > 0) { r = cbAdd(r, cbFromSmall(d)) }
    }
    return r
}
// Exact for a magnitude at or below 2^53, which is all cToF64 ever asks for.
function cbToNum(a) {
    var v = 0
    var i = a.length - 1
    while (i >= 0) {
        v = v * CB_BASE + a[i]
        i = i - 1
    }
    return v
}
function cbPow10(k) {
    var r = []
    var q = Math.floor(k / 7)
    var i = 0
    while (i < q) { r.push(0); i = i + 1 }
    r.push(1)
    var rest = k - q * 7
    var m = 1
    i = 0
    while (i < rest) { m = m * 10; i = i + 1 }
    if (m > 1) { r = cbMulSmall(r, m) }
    return r
}
// 24 bit chunks, so that '&' '|' '^' can use the host's 32 bit bitwise operators
// on a value of arbitrary width.
function cbToChunks(a) {
    var r = []
    var v = a
    while (!cbIsZero(v)) {
        var d = cbDivSmall(v, 16777216)
        r.push(d.r)
        v = d.q
    }
    return r
}
function cbFromChunks(ch) {
    var v = []
    var i = ch.length - 1
    while (i >= 0) {
        v = cbMulSmall(v, 16777216)
        if (ch[i] > 0) { v = cbAdd(v, cbFromSmall(ch[i])) }
        i = i - 1
    }
    return v
}

// ----- the exact rational a constant expression is evaluated over -----
function cMk(s, n, d, f) {
    var nn = cbTrim(n)
    if (cbIsZero(nn)) { return {s: 0, n: [], d: cbOne(), f: f} }
    return {s: s, n: nn, d: cbTrim(d), f: f}
}
function cInt(s, mag) { return cMk(s, mag, cbOne(), false) }
function cTooBig(c) {
    if (cIsCx(c)) { return cTooBig(c.re) || cTooBig(c.im) }
    return c.n.length > CB_LIMIT || c.d.length > CB_LIMIT
}
function cReduce(c) {
    if (c.s === 0) { return c }
    if (c.d.length === 1 && c.d[0] === 1) { return c }
    var g = cbGcd(c.n, c.d)
    if (g.length === 1 && g[0] === 1) { return c }
    return {s: c.s, n: cbDivMod(c.n, g).q, d: cbDivMod(c.d, g).q, f: c.f}
}
function cNeg(c) {
    if (cIsCx(c)) { return cxMk(cNeg(c.re), cNeg(c.im)) }
    if (c.s === 0) { return c }
    return {s: 0 - c.s, n: c.n, d: c.d, f: c.f}
}
function cAddSub(a, b, sub) {
    var f = a.f || b.f
    var bs = sub ? 0 - b.s : b.s
    var n1 = cbMul(a.n, b.d)
    var n2 = cbMul(b.n, a.d)
    var dd = cbMul(a.d, b.d)
    var sa = a.s < 0 ? 0 - 1 : 1
    var sb = bs < 0 ? 0 - 1 : 1
    var rn = []
    var rs = 1
    if (sa === sb) {
        rn = cbAdd(n1, n2)
        rs = sa
    } else {
        var cp = cbCmp(n1, n2)
        if (cp === 0) { return cMk(0, [], cbOne(), f) }
        if (cp > 0) { rn = cbSub(n1, n2); rs = sa } else { rn = cbSub(n2, n1); rs = sb }
    }
    return cReduce(cMk(rs, rn, dd, f))
}
function cMul2(a, b) {
    var f = a.f || b.f
    if (a.s === 0 || b.s === 0) { return cMk(0, [], cbOne(), f) }
    return cReduce(cMk(a.s * b.s, cbMul(a.n, b.n), cbMul(a.d, b.d), f))
}
function cDiv2(a, b) {
    if (b.s === 0) { return null }
    var f = a.f || b.f
    if (!f) {
        // Two untyped INTEGER constants: Go divides them as integers, truncating
        // toward zero, so 1/3 is 0 while 1/3.0 is a third.
        return cInt(a.s * b.s, cbDivMod(a.n, b.n).q)
    }
    if (a.s === 0) { return cMk(0, [], cbOne(), true) }
    return cReduce(cMk(a.s * b.s, cbMul(a.n, b.d), cbMul(a.d, b.n), true))
}
function cMod2(a, b) {
    if (a.f || b.f) { return null }
    if (b.s === 0) { return null }
    // The remainder takes the DIVIDEND's sign, as everywhere else in Go.
    return cInt(a.s, cbDivMod(a.n, b.n).r)
}
function cShift(a, b, left) {
    if (a.f || b.f) { return null }
    if (b.s < 0) { return null }
    if (b.n.length > 1) { return null }
    var k = b.s === 0 ? 0 : b.n[0]
    if (k > 4096) { return null }
    if (left) { return cInt(a.s, cbShl(a.n, k)) }
    if (a.s >= 0) { return cInt(a.s, cbShr(a.n, k)) }
    // '>>' is an ARITHMETIC shift, so a negative constant floors rather than
    // truncating: -1 >> 1 is -1, not 0.
    var p = cbShl(cbOne(), k)
    return cInt(0 - 1, cbShr(cbAdd(a.n, cbSub(p, cbOne())), k))
}
function cBits(op, a, b) {
    if (a.f || b.f) { return null }
    // A negative operand would need two's complement at infinite width; there is
    // no corpus for it, so it keeps the unfolded path.
    if (a.s < 0 || b.s < 0) { return null }
    var x = cbToChunks(a.n)
    var y = cbToChunks(b.n)
    var n = x.length > y.length ? x.length : y.length
    var r = []
    var i = 0
    while (i < n) {
        var xv = i < x.length ? x[i] : 0
        var yv = i < y.length ? y[i] : 0
        var o = 0
        if (op == "&") { o = xv & yv }
        else if (op == "|") { o = xv | yv }
        else if (op == "^") { o = xv ^ yv }
        else { o = xv & (16777215 - yv) }
        r.push(o)
        i = i + 1
    }
    return cInt(1, cbFromChunks(r))
}
function cNot(c) {
    if (c.f) { return null }
    // ^x is -x-1 at arbitrary precision.
    return cAddSub(cInt(0 - 1, cbOne()), c, true)
}
// ----- a COMPLEX constant -----
// (0.1 + 0.2i) * (0.1 + 0.2i) is exactly -0.03 + 0.04i and Go prints it; multiplying
// four already-rounded doubles would answer -0.030000000000000006+0.04000000000000001i.
// A complex constant is a PAIR of the exact rationals above, and it carries s/n/d as
// a zero so that any reader that does not know about it reads a zero rather than a
// missing field. Both parts are marked FLOAT: there is no integer division anywhere
// inside complex arithmetic, and cDiv2 decides which one it does off exactly that
// flag.
function cAsF(c) { return {s: c.s, n: c.n, d: c.d, f: true} }
function cIsCx(c) { return c != null && c.cx === true }
function cxMk(re, im) {
    if (re == null || im == null) { return null }
    return {cx: true, f: true, s: 0, n: [], d: cbOne(), re: cAsF(re), im: cAsF(im)}
}
// A real constant IS a complex one with a zero imaginary part, which is also Go's
// rule for mixing the two kinds (constant kinds order int < float < complex).
function cxOf(c) { return cIsCx(c) ? c : cxMk(c, cMk(0, [], cbOne(), true)) }
function cxApply(op, x, y) {
    var a = cxOf(x)
    var b = cxOf(y)
    if (a == null || b == null) { return null }
    if (op == "+") { return cxMk(cAddSub(a.re, b.re, false), cAddSub(a.im, b.im, false)) }
    if (op == "-") { return cxMk(cAddSub(a.re, b.re, true), cAddSub(a.im, b.im, true)) }
    if (op == "*") {
        return cxMk(cAddSub(cMul2(a.re, b.re), cMul2(a.im, b.im), true),
                    cAddSub(cMul2(a.re, b.im), cMul2(a.im, b.re), false))
    }
    if (op == "/") {
        var den = cAddSub(cMul2(b.re, b.re), cMul2(b.im, b.im), false)
        if (den.s === 0) { return null }
        var nr = cAddSub(cMul2(a.re, b.re), cMul2(a.im, b.im), false)
        var ni = cAddSub(cMul2(a.im, b.re), cMul2(a.re, b.im), true)
        return cxMk(cDiv2(nr, den), cDiv2(ni, den))
    }
    return null
}
// 2i / 2.5i / 1e3i: the literal is the IMAGINARY part, read exactly the way a float
// literal is. Only '==' and '!=' are legal on a complex constant, so the ordering
// comparisons answer null and fall back to the unfolded path.
function goImagLitC(text, pos) {
    var im = cFromFloatText(text.substring(0, text.length - 1))
    regConst(pos, im == null ? null : cxMk(cMk(0, [], cbOne(), true), im))
    return makeImag(parseImag(text))
}
function cApply(op, a, b) {
    if (cIsCx(a) || cIsCx(b)) { return cxApply(op, a, b) }
    if (op == "+") { return cAddSub(a, b, false) }
    if (op == "-") { return cAddSub(a, b, true) }
    if (op == "*") { return cMul2(a, b) }
    if (op == "/") { return cDiv2(a, b) }
    if (op == "%") { return cMod2(a, b) }
    if (op == "<<") { return cShift(a, b, true) }
    if (op == ">>") { return cShift(a, b, false) }
    if (op == "&" || op == "|" || op == "^" || op == "&^") { return cBits(op, a, b) }
    return null
}

// The double NEAREST the exact value, ties to even - the ONE rounding Go performs
// when an untyped float constant reaches a typed location. Math.floor(v*10^p+0.5)
// and parseFloat of a rendered decimal both round twice; this divides the exact
// rational out to the target ulp and rounds there.
function cToF64(c) {
    if (c.s === 0) { return 0 }
    var k = cbBitLen(c.n) - cbBitLen(c.d)
    if (k > 1200) { return c.s < 0 ? 0 - cbInf : cbInf }
    if (k < 0 - 1200) { return 0 }
    var big = false
    if (k >= 0) { big = cbCmp(c.n, cbShl(c.d, k)) >= 0 }
    else { big = cbCmp(cbShl(c.n, 0 - k), c.d) >= 0 }
    var e = big ? k : k - 1
    if (e > 1023) { return c.s < 0 ? 0 - cbInf : cbInf }
    var ulp = e - 52
    if (ulp < 0 - 1074) { ulp = 0 - 1074 }
    var num = c.n
    var den = c.d
    if (ulp >= 0) { den = cbShl(den, ulp) } else { num = cbShl(num, 0 - ulp) }
    var dm = cbDivMod(num, den)
    var q = dm.q
    var cp = cbCmp(cbMulSmall(dm.r, 2), den)
    var up = false
    if (cp > 0) { up = true }
    else if (cp === 0) { up = !cbEven(q) }
    if (up) { q = cbAdd(q, cbOne()) }
    var v = cLdexp(cbToNum(q), ulp)
    return c.s < 0 ? 0 - v : v
}
function cPow2(k) {
    var v = 1
    var i = 0
    while (i < k) { v = v * 2; i = i + 1 }
    return v
}
// Scaling by a power of two in steps of 2^30. Every intermediate is exact,
// because the FINAL value is representable and each step before it is larger.
function cLdexp(m, e) {
    var v = m
    var left = e
    while (left > 0) {
        var s = left > 30 ? 30 : left
        v = v * cPow2(s)
        left = left - s
    }
    while (left < 0) {
        var s2 = left < 0 - 30 ? 30 : 0 - left
        v = v / cPow2(s2)
        left = left + s2
    }
    return v
}

// ----- reading a literal exactly -----
// 1.25e-3 is 125 * 10^-5, read as the exact rational 1/800 - never as parseFloat's
// already-rounded double, which is the whole point of the exercise.
function cFromFloatText(s) {
    var digits = ""
    var frac = 0
    var seenDot = false
    var expPart = ""
    var i = 0
    while (i < s.length) {
        var ch = s.charAt(i)
        if (ch == "_") { i = i + 1; continue }
        if (ch == ".") { seenDot = true; i = i + 1; continue }
        if (ch == "e" || ch == "E") { expPart = s.substring(i + 1); i = s.length; continue }
        digits = digits + ch
        if (seenDot) { frac = frac + 1 }
        i = i + 1
    }
    var exp = 0
    if (expPart.length > 0) {
        var eneg = false
        var ep = expPart
        if (ep.charAt(0) == "+") { ep = ep.substring(1) }
        else if (ep.charAt(0) == "-") { eneg = true; ep = ep.substring(1) }
        exp = parseInt(ep, 10)
        if (eneg) { exp = 0 - exp }
    }
    var m = cbFromDec(digits)
    if (cbIsZero(m)) { return cMk(0, [], cbOne(), true) }
    var e2 = exp - frac
    if (e2 > 4900 || e2 < 0 - 4900) { return null }
    if (e2 >= 0) { return cMk(1, cbMul(m, cbPow10(e2)), cbOne(), true) }
    return cReduce(cMk(1, m, cbPow10(0 - e2), true))
}
// 0x1.8p3 is 0x18 * 2^(3 - 4), the 'p' exponent scaling by a power of TWO.
function cFromHexFloatText(s) {
    var digits = ""
    var frac = 0
    var seenDot = false
    var expPart = ""
    var i = 2
    while (i < s.length) {
        var ch = s.charAt(i)
        if (ch == "_") { i = i + 1; continue }
        if (ch == ".") { seenDot = true; i = i + 1; continue }
        if (ch == "p" || ch == "P") { expPart = s.substring(i + 1); i = s.length; continue }
        digits = digits + ch
        if (seenDot) { frac = frac + 1 }
        i = i + 1
    }
    var exp = 0
    if (expPart.length > 0) {
        var eneg = false
        var ep = expPart
        if (ep.charAt(0) == "+") { ep = ep.substring(1) }
        else if (ep.charAt(0) == "-") { eneg = true; ep = ep.substring(1) }
        exp = parseInt(ep, 10)
        if (eneg) { exp = 0 - exp }
    }
    var m = cbFromRadix(digits, 16)
    if (cbIsZero(m)) { return cMk(0, [], cbOne(), true) }
    var e2 = exp - 4 * frac
    if (e2 > 4900 || e2 < 0 - 4900) { return null }
    if (e2 >= 0) { return cMk(1, cbShl(m, e2), cbOne(), true) }
    return cReduce(cMk(1, m, cbShl(cbOne(), 0 - e2), true))
}
function cFromIntText(s, radix, start) {
    return cInt(1, cbFromRadix(s.substring(start), radix))
}

// ----- the position-keyed table, and the fold itself -----
// The table maps a source position to the constant of the OUTERMOST node that
// ends there, and cNone is the "ends here, and is not a constant" entry. It is
// load-bearing: a prefix operator and a non-folding binary run end at the SAME
// offset as their last operand, so `ci % 2` would otherwise be read back as the
// constant 2 that the literal registered, and `ci%2 == 1` would be false. Every
// node that ends where an operand ends either re-registers its own value or clears.
var constByPos = {}
var cNone = {s: 0, n: [], d: [], f: false, none: true}
function regConst(pos, c) {
    if (pos == undefined || pos == null) { return }
    stByPos["p" + pos] = ""
    constByPos["p" + pos] = (c == null) ? cNone : c
    if (c == null) { cFnByPos["p" + pos] = cFnNone } else { cFnByPos["p" + pos] = cKonstFn(c) }
}
function constAt(pos) {
    if (pos == undefined || pos == null) { return null }
    var c = constByPos["p" + pos]
    if (c == undefined) { return null }
    if (c.none === true) { return null }
    return c
}
// ----- the IOTA-AWARE parallel channel: a position -> function(iota) -> constant --
//
// WHY A SECOND CHANNEL AND NOT A SECOND FOLD. Inside `const ( A = 1 << iota; B; C )`
// a spec with no '=' REPEATS the previous spec's expression list, and that list is
// PARSED ONCE. constByPos can therefore only ever hold A's value; B and C need the
// same expression evaluated at a different iota, and nothing survives the parse that
// could be re-evaluated. So every fold site registers, beside its constant, the fold
// ITSELF as a closure over iota. cBindSpec is the only reader.
//
// constByPos is deliberately NOT given iota's value: a run mentioning iota must keep
// failing to fold, or goFoldConst would materialize A's constant INTO THE THUNK and
// makeConstGroup - which re-emits that same thunk for B and C - would hand all three
// specs A's value. The emission path is unchanged by this whole channel; only the
// COMPILE-TIME binding (cBindConst, cArrLenTbl, and so goArrConstLen) gains a value.
//
// The channel never holds null: an absent or unfoldable position answers cFnNone,
// which answers null for every iota, so a composition needs no per-operand test.
var cFnByPos = {}
function cFnNone(io) { return null }
function cKonstFn(c) { return function(io) { return c } }
function cIotaFn(io) { return cInt(1, cbFromSmall(io)) }
function cNegFn(f) {
    return function(io) {
        var v = f(io)
        if (v == null) { return null }
        return cNeg(v)
    }
}
function cNotFn(f) {
    return function(io) {
        var v = f(io)
        if (v == null) { return null }
        return cNot(v)
    }
}
function regCFn(pos, f) {
    if (pos == undefined || pos == null) { return }
    cFnByPos["p" + pos] = f
}
function cfnAt(pos) {
    if (pos == undefined || pos == null) { return cFnNone }
    var f = cFnByPos["p" + pos]
    if (f == undefined) { return cFnNone }
    return f
}
// goFoldConst's run, kept as a function of iota instead of evaluated once. Built
// whether or not the constant fold succeeded, because the reason it failed may be
// exactly that an operand was iota.
function goFoldCFn(items, ops, ty) {
    if (ty != null) { return cFnNone }
    var fs = [cfnAt(items[1].at)]
    var k = 0
    while (k < ops.length) {
        fs.push(cfnAt(items[3 * k + 4].at))
        k = k + 1
    }
    return function(io) {
        var acc = fs[0](io)
        if (acc == null) { return null }
        var j = 0
        while (j < ops.length) {
            var rhs = fs[j + 1](io)
            if (rhs == null) { return null }
            var nx = cApply(ops[j], acc, rhs)
            if (nx == null) { return null }
            if (cTooBig(nx)) { return null }
            acc = nx
            j = j + 1
        }
        return acc
    }
}
// The thunk is handed straight back; the point is the clear. Used by every unary
// form whose value is NOT its operand's constant (!x, &x, *x, <-x).
function goClearC(t, pos) {
    regConst(pos, null)
    return t
}
// items = [t0, {at}, op1, t1, {at}, ...]: operand k is at 3k and its end position
// at 3k + 1, so the right operand of ops[k] carries its position at 3k + 4. A
// written-out conversion anywhere in the run (ty != null) is Go's own reason not
// to fold: the conversion has already rounded.
function goFoldConst(items, ops, pos, ty) {
    if (ty != null) { return null }
    var acc = constAt(items[1].at)
    if (acc == null) { return null }
    var k = 0
    while (k < ops.length) {
        var rhs = constAt(items[3 * k + 4].at)
        if (rhs == null) { return null }
        var nx = cApply(ops[k], acc, rhs)
        if (nx == null) { return null }
        if (cTooBig(nx)) { return null }
        acc = nx
        k = k + 1
    }
    regConst(pos, acc)
    return cMaterialize(acc)
}

// ^x is -x-1 at arbitrary precision; a written-out unsigned conversion complements
// within its width instead, which is not a constant fold.
function goComplC(items) {
    if (tyAt(items[1].at) != null) { return }
    var inner = cfnAt(items[1].at)
    var c = constAt(items[1].at)
    if (c != null) { regConst(items[1].at, cNot(c)) }
    else { regCFn(items[1].at, cNotFn(inner)) }
}
// -x and (x) do not move the END position of what they wrap (a prefix operator
// and the group's ')' respectively), so both simply re-register at their own key.
function goNegC(t, pos) {
    var inner = cfnAt(pos)
    var c = constAt(pos)
    if (c == null) { regCFn(pos, cNegFn(inner)); return makeNeg(t) }
    var nc = cNeg(c)
    regConst(pos, nc)
    // UNARY MINUS IS A FOLD SITE because a Go CONSTANT HAS NO SIGNED ZERO:
    // `-0.0` is exactly the constant 0 and `complex(-0.0, 0.0)` is (0+0i)
    // before the program runs. cNeg hands a zero back unchanged, but
    // cMaterialize is otherwise reached only from goFoldConst (a BINARY run),
    // so without this the runtime thunk would answer -0.0 for `x := -0.0`,
    // `var y float64 = -0.0`, `const c = -0.0` and the bare literal.
    //
    // Only a ZERO materialises here. Folding every unary minus would also
    // re-route `-9223372036854775808` and every other magnitude through
    // cMaterialize's own wrap rules for no observable gain; the zero is the
    // one difference Go's "constants have no signed zero" rule forces.
    if (nc.s === 0) { return cMaterialize(nc) }
    return makeNeg(t)
}
// items = [thunk, {at}] - PGInner tags the inner expression's end position.
function goParenC(items, pos) {
    var c = constAt(items[1].at)
    if (c != null) { regConst(pos, c) }
    else { regCFn(pos, cfnAt(items[1].at)) }
    return items[0]
}
function cFromRadixText(text) {
    var radix = 10
    var start = 0
    if (text.length >= 2 && text.charAt(0) == "0") {
        var p = text.charAt(1)
        if (p == "x" || p == "X") { radix = 16; start = 2 }
        else if (p == "o" || p == "O") { radix = 8; start = 2 }
        else if (p == "b" || p == "B") { radix = 2; start = 2 }
    }
    return cFromIntText(text, radix, start)
}
// ----- NAMED constants: a parse-time scope, and what it declines -----
// `const s = a + b` has to fold too, or Go's single rounding is only half
// implemented - but resolving a NAME needs a scope, and a name that is also
// declared as a variable somewhere would otherwise fold WRONGLY, which is a silent
// wrong answer in every program that writes it. cScopes is that scope, built at
// parse time and carrying exactly one thing per name: the folded constant of a
// `const` declaration, or NULL for "some other binder owns this name here" - a var,
// a parameter, a receiver, a named result, a range or type-switch variable, an
// assignment target, a func, a type, an import alias. A frame is pushed at every
// '{' that opens a Block and at every select arm and popped with it, and a switch
// body is truncated back to its own mark at each 'case', because Go gives each case
// clause its own scope while the grammars parse that body flat.
//
// POISON LEAKING OUTWARD IS SAFE and a CONST LEAKING OUTWARD IS NOT, which is the
// asymmetry the whole design rests on: a for/if init or a parameter list poisons
// into the enclosing frame (a decline), while a const only ever binds inside the
// frame of the block that declares it.
//
// WHAT IT DECLINES, each falling back to the unfolded path rather than to a
// half-fold:
//   - a spec that carries a TYPE (const x float64 = 0.1). Go converts - and so
//     rounds - at the declaration, so x + 0.2 really is float64 arithmetic there.
//   - IOTA, and with it any spec that REPEATS the previous spec's expression list.
//     One source position would have to carry a different constant per spec, and
//     this table is keyed by source position; the compiled half re-emits the very
//     same thunk per spec, which is the same statement from the other side.
//   - a name used EARLIER in the file than its declaration. Package scope has no
//     order in Go; this fold runs in source order, so a forward reference declines.
//   - a name declared with := , var, or as a parameter anywhere its frame can see,
//     and every name at all once the frame that bound it has been popped.
// A frame is {n: names, v: constants, t: static types} as parallel arrays rather than
// an object, because a frame has to be TRUNCATED (the switch-case rule) and
// enumerated, and neither is expressible on an object in the frozen script subset.
// The t column is the STATIC TYPES section's (below); a binder that does not set it
// leaves "", which means "not known".
var cScopes = [{n: [], v: [], t: []}]
var cPend = []
var cPendT = []
var cSwMarks = []
function cTop() { return cScopes[cScopes.length - 1] }
// A parameter or receiver name is collected here and applied to the NEXT frame,
// which is the function body: poisoning it where the tag fires would poison it in
// the PACKAGE frame instead and kill `const n` for the whole file.
function cPendName(name) { cPend.push(name); cPendT.push(""); return name }
function cScopeEnter() {
    var f = {n: [], v: [], t: []}
    var i = 0
    while (i < cPend.length) { f.n.push(cPend[i]); f.v.push(null); f.t.push(cPendT[i]); i = i + 1 }
    cPend = []
    cPendT = []
    cScopes.push(f)
}
function cScopeExit() { if (cScopes.length > 1) { cScopes.pop() } }
function cBindConst(name, c) {
    var f = cTop()
    f.n.push(name)
    f.v.push(c == undefined ? null : c)
    f.t.push("")
}
// Returns the name so it can be written inline in a tag that also pushes it.
function cPoisonName(name) { cBindConst(name, null); return name }
function cLookup(name) {
    var s = cScopes.length - 1
    while (s >= 0) {
        var f = cScopes[s]
        var i = f.n.length - 1
        while (i >= 0) {
            if (f.n[i] === name) { return f.v[i] }
            i = i - 1
        }
        s = s - 1
    }
    return null
}
// ----- STATIC TYPES: which stores copy a struct -----
// A struct and a pointer to it are ONE object in this value model, so a store
// cannot tell a copy (b := a) from an alias (q := p) by the value. What the front
// end knows of the static type decides, and it knows it for a few forms only:
//   - a binding keeps the type text it was declared with: a typed var, a parameter,
//     a receiver (cPendTy), or the type of the single expression a := binds
//     (cBindTy); an assignment target keeps the one it had (cAssignName), and every
//     other binder - range, type switch, comma-ok, named result - leaves "".
//   - an expression reports its type against its END position, like constByPos: a
//     name its binding's, &x a pointer ("*" + x's), *p what p points at ("?" when
//     that is not known: a value of some type), a composite literal its own type
//     marked fresh ("=T": a new value, nothing to copy). regConst clears the entry,
//     so a node ending where a reporting operand ends reads as "" unless it reports.
// "" means "not known", and an unknown store does not copy - the pointer reading,
// which can only lose a copy, never an alias. Readers: goStCopyTy (the stores in
// both grammars) and goCmpFold's pointer comparison.
var stByPos = {}
function regSt(pos, ty) {
    if (pos == undefined || pos == null) { return }
    stByPos["p" + pos] = ty
}
function stAt(pos) {
    if (pos == undefined || pos == null) { return "" }
    var t = stByPos["p" + pos]
    if (t == undefined) { return "" }
    return t
}
function cTyLookup(name) {
    var s = cScopes.length - 1
    while (s >= 0) {
        var f = cScopes[s]
        var i = f.n.length - 1
        while (i >= 0) {
            if (f.n[i] === name) { return f.t[i] }
            i = i - 1
        }
        s = s - 1
    }
    return ""
}
// The newest binding of name in the innermost frame takes the type ty.
function cBindTy(name, ty) {
    var f = cTop()
    var i = f.n.length - 1
    while (i >= 0) {
        if (f.n[i] === name) { f.t[i] = ty; return }
        i = i - 1
    }
}
// A pending parameter or receiver name takes its declared type.
function cPendTy(name, ty) {
    var i = cPend.length - 1
    while (i >= 0) {
        if (cPend[i] === name) { cPendT[i] = ty; return }
        i = i - 1
    }
}
// An assignment target: a poison binding (it is not a constant here) that keeps the
// static type the name already had.
function cAssignName(name) {
    var t = cTyLookup(name)
    cBindConst(name, null)
    cBindTy(name, t)
    return name
}
// A type with "=" (fresh) taken off: what a binding of that expression holds.
function stBare(t) { return (t.length > 0 && t.charAt(0) == "=") ? t.substring(1) : t }
// The type a store must copy with, or "" when it copies nothing: a fresh value, a
// pointer, a slice, map, channel, function or interface, and an unknown type are
// the value itself. The grammar resolves the text to a struct or an array of them.
function goStCopyTy(t) {
    if (t == "" || t.charAt(0) == "=" || t.charAt(0) == "*") { return "" }
    if (t == "any" || t == "error" || t.indexOf("[]") == 0 || t.indexOf("map[") == 0) { return "" }
    if (t.indexOf("chan") == 0 || t.indexOf("<-") == 0 || t.indexOf("func") == 0) { return "" }
    if (t.indexOf("interface") == 0) { return "" }
    return t
}
// *x and &x, at the position they and their operand share.
function goDerefC(t, pos) {
    var it = stAt(pos)
    regConst(pos, null)
    var b = stBare(it)
    regSt(pos, (b.length > 1 && b.charAt(0) == "*") ? b.substring(1) : "?")
    return t
}
function goAddrC(t, pos) {
    var it = stBare(stAt(pos))
    regConst(pos, null)
    regSt(pos, "*" + (it == "" ? "?" : it))
    return t
}
function goAddrNameC(t, name, pos) {
    var it = cTyLookup(name)
    regSt(pos, "*" + (it == "" ? "?" : it))
    return t
}
// A composite literal of type ty: a fresh value.
function goLitC(t, ty, pos) {
    regSt(pos, "=" + ty)
    return t
}
// The element type of a slice, array or map type text, or "".
function stElemTy(t) {
    var b = stBare(t)
    if (b.length > 0 && b.charAt(0) == "*") { b = b.substring(1) }
    if (b.indexOf("map[") == 0) {
        var d = 0
        var i = 3
        while (i < b.length) {
            var ch = b.charAt(i)
            if (ch == "[") { d = d + 1 }
            if (ch == "]") { d = d - 1; if (d == 0) { return b.substring(i + 1) } }
            i = i + 1
        }
        return ""
    }
    if (b.length > 1 && b.charAt(0) == "[") {
        var cl = b.indexOf("]")
        if (cl > 0) { return b.substring(cl + 1) }
    }
    return ""
}
// A selector/index chain, from its matched TEXT (whitespace already stripped): a
// name, then `.field` and `[index]` steps, typed through the binding's type, the
// field types of the struct declarations (goFieldTyHook, one per grammar) and the
// element types. A call, a conversion, a slice or an assertion anywhere ends it
// unknown.
function goChainC(t, text, pos) {
    var ty = stChainTy(text)
    if (ty != "") { regSt(pos, ty) }
    return t
}
// The type of a chain's text, or "" (a bare name answers its binding's).
function stChainTy(text) {
    var i = 0
    while (i < text.length && cIsIdCh(text.charCodeAt(i))) { i = i + 1 }
    if (i == 0) { return "" }
    var ty = stBare(cTyLookup(text.substring(0, i)))
    while (i < text.length && ty != "") {
        var ch = text.charAt(i)
        if (ch == ".") {
            var j = i + 1
            while (j < text.length && cIsIdCh(text.charCodeAt(j))) { j = j + 1 }
            if (j == i + 1) { return "" }
            var sn = ty.charAt(0) == "*" ? ty.substring(1) : ty
            ty = goFieldTyHook(sn, text.substring(i + 1, j))
            i = j
        } else if (ch == "[") {
            var d = 0
            var k = i
            var q = 0
            while (k < text.length) {
                var c2 = text.charCodeAt(k)
                if (q != 0) { if (c2 == q) { q = 0 } else if (c2 == 92) { k = k + 1 } }
                else if (c2 == 34 || c2 == 96 || c2 == 39) { q = c2 }
                else if (c2 == 91) { d = d + 1 }
                else if (c2 == 93) { d = d - 1; if (d == 0) { break } }
                else if (c2 == 58 && d == 1) { return "" }
                k = k + 1
            }
            if (k >= text.length) { return "" }
            ty = stElemTy(ty)
            i = k + 1
        } else { return "" }
    }
    return ty
}
// append(s, v, ...): each appended struct value is a copy, by s's element type.
// items = ["append", s, v..., {spread}?]; text is the call's.
function goAppendC(items, text) {
    if (items.length < 3 || items[0] !== "append") { return items }
    var a = text.indexOf("(") + 1
    var e = a
    var d = 0
    while (e < text.length) {
        var cc = text.charCodeAt(e)
        if (cc == 40 || cc == 91 || cc == 123) { d = d + 1 }
        if (cc == 41 || cc == 93 || cc == 125) { if (d == 0) { break } d = d - 1 }
        if (cc == 44 && d == 0) { break }
        e = e + 1
    }
    var cty = goStCopyTy(stElemTy(stChainTy(text.substring(a, e))))
    if (cty == "") { return items }
    var i = 2
    while (i < items.length) {
        if (typeof items[i] == "function") { items[i] = goStoreThunk(items[i], cty) }
        i = i + 1
    }
    return items
}
function cIsIdCh(cc) { return (cc >= 48 && cc <= 57) || (cc >= 65 && cc <= 90) || (cc >= 97 && cc <= 122) || cc == 95 }
// `for k, v := range x`: the names are noted as they are bound, and the ranged
// expression's element type becomes v's - and the copy each iteration makes of a
// struct element ({rcopy}, read by makeForRange in both grammars).
var cRangeNames = []
function cRangeStart(lbl) { cRangeNames = []; return lbl }
function cRangeVar(name) { cPoisonName(name); cRangeNames.push(name); return name }
function goRangeC(pos) {
    var et = stElemTy(stAt(pos))
    if (cRangeNames.length > 1) { cBindTy(cRangeNames[1], et) }
    cRangeNames = []
    return {rcopy: goStCopyTy(et)}
}

// The switch body's mark: its frame DEPTH as well as its length, so a truncation
// that would land in the wrong frame does nothing at all instead.
function cSwEnter() { cSwMarks.push({d: cScopes.length, k: cTop().n.length}) }
function cSwTrunc(m) {
    if (cScopes.length !== m.d) { return }
    var f = cTop()
    while (f.n.length > m.k) { f.n.pop(); f.v.pop(); f.t.pop() }
}
function cSwCase() { if (cSwMarks.length > 0) { cSwTrunc(cSwMarks[cSwMarks.length - 1]) } }
function cSwExit() { if (cSwMarks.length > 0) { cSwTrunc(cSwMarks.pop()) } }
// The spec index INSIDE the current const group, which is what `iota` means, and
// the group's last seen expression list as functions of it. CGrpOn resets both at
// every `const`, grouped or not; CSpecOn steps the index once per spec.
var cIotaIdx = 0
var cLastFns = []
function cSpecVal(fns, n, ok, io) {
    if (!ok) { return null }
    return fns[n](io)
}
// A bare name registers its constant against its own end position, exactly as a
// literal does, so every fold above it reads names and literals the same way. The
// thunk is UNCHANGED - the name still reads its variable - so a fold is the only
// thing that can consume it, and a context the grammars do not fold is unaffected.
// The clear on the miss is the constByPos invariant: a node that ends where an
// operand ends either re-registers or clears.
function goNameC(name, pos) {
    if (name === "iota") {
        // No CONSTANT (see cFnByPos: folding iota would poison the re-emitted
        // thunk a repeating spec shares), but a fn channel entry, which is what
        // gives `const ( R = iota; G; B )` its compile-time 0, 1, 2.
        regConst(pos, null)
        regCFn(pos, cIotaFn)
        return makeVarRef(name)
    }
    regConst(pos, cLookup(name))
    regSt(pos, cTyLookup(name))
    return makeVarRef(name)
}
// A const spec binds its names and hands the item list on with the position markers
// CExprItem pushed taken back out, so the builders below see the plain shape.
// items = [name..., {ty}?, thunk, {cat}, thunk, {cat}, ...].
function cBindSpec(items) {
    var names = []
    var i = 0
    for (; i < items.length; i++) {
        if (typeof items[i] != "string") { break }
        names.push(items[i])
    }
    var typed = false
    if (i < items.length && items[i] != undefined && items[i].ty != undefined) { typed = true; i++ }
    var rest = []
    var ats = []
    var k = i
    while (k < items.length) {
        if (typeof items[k] == "object" && items[k] != null && items[k].cat !== undefined) { ats.push(items[k].cat) }
        else { rest.push(items[k]) }
        k = k + 1
    }
    // The spec's expression list AS FUNCTIONS OF IOTA. A spec that HAS one records
    // it as the group's current list; a spec with no '=' at all repeats it, which
    // is Go's rule and the whole reason the fn channel exists. Anything else (a
    // name count that does not match the expression count) binds nothing.
    var fns = []
    if (ats.length === names.length && rest.length === names.length) {
        var af = 0
        while (af < ats.length) { fns.push(cfnAt(ats[af])); af = af + 1 }
        cLastFns = fns
    } else if (ats.length === 0 && rest.length === 0 && cLastFns.length === names.length) {
        fns = cLastFns
    }
    var okn = fns.length === names.length
    var n = 0
    while (n < names.length) {
        cBindConst(names[n], typed ? null : cSpecVal(fns, n, okn, cIotaIdx))
        n = n + 1
    }
    // A TYPED const is deliberately NOT an untyped folding constant (cBindConst
    // gets null above, so `const k int = 3` never folds into an expression). But
    // `const size int = 3; var a [size]int` is still a three element array in go,
    // so the integer value is kept in a side table that ONLY goArrConstLen reads -
    // it cannot reach the folder and therefore cannot change any expression's
    // value. Flat rather than scoped, which is the one thing it gives up: a
    // function-local typed const shadowing a package-level one of the same name
    // would take the inner value for both.
    if (typed) {
        var tn = 0
        while (tn < names.length) {
            var td = goConstIntText(cSpecVal(fns, tn, okn, cIotaIdx))
            if (td.length > 0) { cArrLenTbl["$" + names[tn]] = td }
            tn = tn + 1
        }
    }
    var out = []
    var q = 0
    while (q < i) { out.push(items[q]); q = q + 1 }
    var z = 0
    while (z < rest.length) { out.push(rest[z]); z = z + 1 }
    return out
}
// A rune literal is an ordinary INTEGER constant in Go ('A' is 65). Its value is
// built inside the rule rather than read back off the matched text, so it is
// registered here at the literal's own end position, which makes 'z' - 'a' and
// 1 << ('c' - 'a') fold like any other pair.
function goRuneLitC(v, pos) {
    regConst(pos, cInt(1, cbFromSmall(v)))
    return makeConst(v)
}

// Go compares two constants EXACTLY as well, and this is not a detail: 2.0/3.0 ==
// 0.6666666666666666 is FALSE in Go, because two thirds is not that decimal, even
// though both round to the same double. Comparing the two ROUNDED values would
// answer true, so the comparison levels fold here too. Only a SINGLE comparison
// folds: a chain would need the untyped BOOL constant Go's spec has and this front
// end does not model (and Go rejects `a < b < c` anyway).
// items = [t0, {at}, op, t1, {at}, ...] - the same shape goFold reads.
function goCmpFold(items) {
    var ts = [items[0]]
    var ops = []
    var i = 2
    while (i < items.length) {
        ops.push(items[i])
        ts.push(items[i + 1])
        i += 3
    }
    if (ops.length == 1) {
        // == between POINTERS is identity: &T{1} == &T{1} is false while *x == *y
        // compares the structs. "==p" / "!=p" are the grammars' pointer operators.
        var lt = stBare(stAt(items[1].at))
        var rt = stBare(stAt(items[4].at))
        if ((ops[0] == "==" || ops[0] == "!=") && (lt.charAt(0) == "*" || rt.charAt(0) == "*")) {
            ops[0] = ops[0] + "p"
        }
        var l = constAt(items[1].at)
        var r = constAt(items[4].at)
        // The clear has to come AFTER the two reads: this node ends exactly where
        // its right operand does, so clearing first would erase what it reads.
        regConst(items[4].at, null)
        if (l != null && r != null) {
            var got = cCompare(ops[0], l, r)
            if (got != null) { return makeConst(got.v) }
        }
    } else if (ops.length > 1) {
        regConst(items[items.length - 1].at, null)
    }
    var flat = [ts[0]]
    for (var j = 0; j < ops.length; j++) { flat.push(ops[j]); flat.push(ts[j + 1]) }
    return foldBinary(flat)
}
function cCmpExact(a, b) {
    if (a.s !== b.s) { return a.s > b.s ? 1 : 0 - 1 }
    if (a.s === 0) { return 0 }
    var c = cbCmp(cbMul(a.n, b.d), cbMul(b.n, a.d))
    return a.s < 0 ? 0 - c : c
}
// {v: bool}, or null for an operator that is not a comparison.
function cCompare(op, a, b) {
    if (cIsCx(a) || cIsCx(b)) {
        var xa = cxOf(a)
        var xb = cxOf(b)
        if (xa == null || xb == null) { return null }
        var eq = cCmpExact(xa.re, xb.re) === 0 && cCmpExact(xa.im, xb.im) === 0
        if (op == "==") { return {v: eq} }
        if (op == "!=") { return {v: !eq} }
        return null
    }
    var c = cCmpExact(a, b)
    if (op == "==") { return {v: c === 0} }
    if (op == "!=") { return {v: c !== 0} }
    if (op == "<") { return {v: c < 0} }
    if (op == "<=") { return {v: c <= 0} }
    if (op == ">") { return {v: c > 0} }
    if (op == ">=") { return {v: c >= 0} }
    return null
}

// '||' and '&&' end where their last operand ends, so a run WITH an operator has
// to clear; a bare pass-through (one operand) must not, or a parenthesised
// constant would lose its value on the way out of the group.
function goLogicC(items, isOr, pos) {
    if (items.length > 1) { regConst(pos, null) }
    return makeOrAnd(items, isOr)
}

