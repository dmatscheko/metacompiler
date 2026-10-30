// c-printf.js -- one conversion of C's printf as macOS libc prints it, for Lua's
// string.format (lib/lua-strlib.js) and Swift's String(format:) (lib/swift-format.js).
// Included by lua-interpreter.abnf and swift-interpreter.abnf, imported by
// languages/lib/lua-rt.metajs and swift-rt.metajs. The Go twin is lsPad,
// lsFmtIntConv and lsFmtFloat in abnf/jsrtluafmt.go.
//
// Value-model free: a spec is {left, plus, space, alt, zero, width, prec} (prec
// -1 when absent), an integer arrives as its digits {neg, d, u, x, o} - the sign
// and magnitude in decimal, and the bit pattern read unsigned in base 10, 16 and
// 8 - and a float as a host number. The includer supplies two HOOKS:
//   cpfExactDec(a)          the exact decimal of a finite a >= 0 as {i, f}
//   cpfRoundHalfEven(ds, k) the digit string rounded to k digits, half to even

var CPF_HEX = "0123456789abcdef"

function cpfZeros(n) {
    var s = ""
    var k = 0
    while (k < n) { s = s + "0"; k = k + 1 }
    return s
}
function cpfUpper(s) {
    var out = ""
    for (var i = 0; i < s.length; i++) {
        var c = s.charCodeAt(i)
        out = out + (c >= 97 && c <= 122 ? String.fromCharCode(c - 32) : s.charAt(i))
    }
    return out
}
// Pads `body` (after `sign`, which zero padding goes behind) to the width.
function cpfPad(sp, sign, body, zeroOk) {
    var n = sign.length + body.length
    if (n >= sp.width) { return sign + body }
    var fill = ""
    var k = n
    while (k < sp.width) { fill = fill + (sp.zero && zeroOk && !sp.left ? "0" : " "); k = k + 1 }
    if (sp.left) { return sign + body + fill }
    if (sp.zero && zeroOk) { return sign + fill + body }
    return fill + sign + body
}

// %d %i %u %o %x %X
function cpfIntConv(sp, conv, iv) {
    var digits = iv.d
    var sign = ""
    if (conv === 117) { digits = iv.u }
    else if (conv === 111) { digits = iv.o }
    else if (conv === 120) { digits = iv.x }
    else if (conv === 88) { digits = cpfUpper(iv.x) }
    else if (iv.neg) { sign = "-" }
    else if (sp.plus) { sign = "+" }
    else if (sp.space) { sign = " " }
    var zero = digits == "0"
    if (sp.prec >= 0) {
        if (sp.prec === 0 && zero) { digits = "" }
        if (digits.length < sp.prec) { digits = cpfZeros(sp.prec - digits.length) + digits }
    }
    if (sp.alt && conv === 111 && digits.charCodeAt(0) !== 48) { digits = "0" + digits }
    if (sp.alt && !zero && conv === 120) { sign = "0x" }
    if (sp.alt && !zero && conv === 88) { sign = "0X" }
    return cpfPad(sp, sign, digits, sp.prec < 0)
}

// The decimal digits of a finite a > 0 rounded to `n` significant digits, and
// the exponent of the first: {ds, e}.
function cpfSigDigits(a, n) {
    var dec = cpfExactDec(a)
    var all = dec.i + dec.f
    var e = dec.i.length - 1
    var lead = 0
    while (lead < all.length && all.charCodeAt(lead) === 48) { lead = lead + 1 }
    e = e - lead
    var ds = cpfRoundHalfEven(all.substring(lead), n)
    if (ds.length > n) { ds = ds.substring(0, n); e = e + 1 }
    return {ds: ds, e: e}
}
function cpfExpText(e) {
    var a = e < 0 ? 0 - e : e
    var s = "" + a
    if (s.length < 2) { s = "0" + s }
    return (e < 0 ? "-" : "+") + s
}
// %e of a finite a >= 0 with `prec` digits after the point.
function cpfE(a, prec, alt) {
    var ds = ""
    var e = 0
    if (a === 0) { ds = cpfZeros(prec + 1) } else {
        var r = cpfSigDigits(a, prec + 1)
        ds = r.ds
        e = r.e
    }
    var body = ds.substring(0, 1)
    if (prec > 0 || alt) { body = body + "." + ds.substring(1) }
    return body + "e" + cpfExpText(e)
}
// %f of a finite a >= 0.
function cpfF(a, prec, alt) {
    var dec = cpfExactDec(a)
    var kept = cpfRoundHalfEven(dec.i + dec.f, dec.i.length + prec)
    var ilen = dec.i.length
    if (kept.length > dec.i.length + prec) { ilen = ilen + 1 }
    var ints = kept.substring(0, ilen)
    if (ints == "") { ints = "0" }
    if (prec > 0) { return ints + "." + kept.substring(ilen) }
    return alt ? ints + "." : ints
}
// %g: %e or %f by the exponent the value has at P significant digits, trailing
// zeros (and a bare point) removed unless '#'.
function cpfG(a, prec, alt) {
    var P = prec < 0 ? 6 : (prec === 0 ? 1 : prec)
    var x = 0
    if (a !== 0) { x = cpfSigDigits(a, P).e }
    var s = ""
    if (x < 0 - 4 || x >= P) { s = cpfE(a, P - 1, alt) } else { s = cpfF(a, P - 1 - x, alt) }
    if (alt) { return s }
    var ei = s.indexOf("e")
    var mant = ei < 0 ? s : s.substring(0, ei)
    var tail = ei < 0 ? "" : s.substring(ei)
    if (mant.indexOf(".") >= 0) {
        var k = mant.length
        while (k > 0 && mant.charCodeAt(k - 1) === 48) { k = k - 1 }
        if (k > 0 && mant.charCodeAt(k - 1) === 46) { k = k - 1 }
        mant = mant.substring(0, k)
    }
    return mant + tail
}
// %a of a finite a >= 0: 0x1.<hex>p<exp>, normalized (a subnormal too). With a
// precision the hex digits are cut and rounded as this platform's libc rounds
// them, which reads only the next two dropped digits: up from 0x90, down below
// 0x80, and in between up exactly when the second digit is odd (measured over
// every two-digit tail); a carry makes the lead digit 2.
function cpfA(a, prec, alt) {
    var lead = "1"
    var ds = ""
    var e = 0
    if (a === 0) { lead = "0" } else {
        var m = a
        while (m >= 2) { m = m / 2; e = e + 1 }
        while (m < 1) { m = m * 2; e = e - 1 }
        var f = m - 1
        while (f > 0) {
            f = f * 16
            var d = Math.floor(f)
            ds = ds + CPF_HEX.charAt(d)
            f = f - d
        }
    }
    if (prec >= 0) {
        if (ds.length > prec) {
            var next = CPF_HEX.indexOf(ds.charAt(prec)) * 16
            if (prec + 1 < ds.length) { next = next + CPF_HEX.indexOf(ds.charAt(prec + 1)) }
            var up = next >= 144 || (next >= 128 && next % 2 === 1)
            ds = ds.substring(0, prec)
            if (up) {
                var codes = []
                for (var i = 0; i < ds.length; i++) { codes.push(CPF_HEX.indexOf(ds.charAt(i))) }
                var at = codes.length - 1
                var carry = true
                while (at >= 0 && carry) {
                    if (codes[at] === 15) { codes[at] = 0 } else { codes[at] = codes[at] + 1; carry = false }
                    at = at - 1
                }
                ds = ""
                for (var j = 0; j < codes.length; j++) { ds = ds + CPF_HEX.charAt(codes[j]) }
                if (carry) { lead = lead == "0" ? "1" : "2" }
            }
        }
        while (ds.length < prec) { ds = ds + "0" }
    }
    var body = "0x" + lead
    if (ds.length > 0 || alt) { body = body + "." + ds }
    return body + "p" + (e < 0 ? "-" : "+") + (e < 0 ? 0 - e : e)
}
// A float conversion (a A e E f F g G): the sign, inf / nan (which take no zero
// padding, and nan no sign at all), the digits, the case.
function cpfFloat(sp, conv, v) {
    var lowc = conv >= 97 ? conv : conv + 32
    var neg = v < 0 || (v === 0 && 1 / v < 0)
    var a = neg ? 0 - v : v
    var sign = ""
    if (v !== v) { neg = false }
    else if (neg) { sign = "-" }
    else if (sp.plus) { sign = "+" }
    else if (sp.space) { sign = " " }
    var body = ""
    var finite = v === v && a - a === 0
    if (!finite) { body = v !== v ? "nan" : "inf" }
    else if (lowc === 101) { body = cpfE(a, sp.prec < 0 ? 6 : sp.prec, sp.alt) }
    else if (lowc === 102) { body = cpfF(a, sp.prec < 0 ? 6 : sp.prec, sp.alt) }
    else if (lowc === 103) { body = cpfG(a, sp.prec, sp.alt) }
    else { body = cpfA(a, sp.prec, sp.alt) }
    if (conv < 97) { body = cpfUpper(body) }
    if (finite && lowc === 97 && sp.zero && !sp.left) {
        // The zero padding of %a goes behind the "0x".
        return sign + body.substring(0, 2) + cpfZeros(sp.width - sign.length - body.length) + body.substring(2)
    }
    return cpfPad(sp, sign, body, finite)
}
