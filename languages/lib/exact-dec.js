// exact-dec.js -- the exact decimal expansion of a double, and half-to-even
// rounding of a digit string: the digit machinery behind every float formatter
// in the interpreter halves.
//
// Included by lib/interp-core.js, and directly by bash-interpreter.abnf, which
// has no shared core.

// The EXACT decimal of a finite double a >= 0, as {i: <integer digits>, f:
// <fraction digits>}. A double is m * 2^e with m an integer below 2^53, so for
// e < 0 its decimal is m * 5^-e with the point -e digits from the right; the
// bignum is a little-endian array of base-10000 limbs, and every limb product
// stays exact. Twin of rtExactDec in lib/runtime.metajs.
function coreExactDec(a) {
    if (a == 0) { return {i: "0", f: ""} }
    var m = a
    var e = 0
    while (m != Math.floor(m)) { m = m * 2; e = e - 1 }
    while (m > 9007199254740992) { m = m / 2; e = e + 1 }
    var limbs = []
    while (m > 0) {
        var q = Math.floor(m / 10000)
        limbs.push(m - q * 10000)
        m = q
    }
    var k = e >= 0 ? 2 : 5
    var n = e >= 0 ? e : 0 - e
    for (var j = 0; j < n; j++) {
        var carry = 0
        for (var li = 0; li < limbs.length; li++) {
            var v = limbs[li] * k + carry
            carry = Math.floor(v / 10000)
            limbs[li] = v - carry * 10000
        }
        if (carry > 0) { limbs.push(carry) }
    }
    var all = "" + limbs[limbs.length - 1]
    for (var t = limbs.length - 2; t >= 0; t--) {
        var s = "" + limbs[t]
        while (s.length < 4) { s = "0" + s }
        all = all + s
    }
    if (e >= 0) { return {i: all, f: ""} }
    while (all.length <= n) { all = "0" + all }
    return {i: all.substring(0, all.length - n), f: all.substring(all.length - n)}
}
// The digit string rounded to `keep` digits, half to EVEN - one digit longer
// when the carry ran all the way out (999 -> 1000). Twin of rtRoundHalfEven in
// lib/runtime.metajs.
function coreRoundHalfEven(digits, keep) {
    if (keep >= digits.length) {
        var out = digits
        while (out.length < keep) { out = out + "0" }
        return out
    }
    var head = digits.substring(0, keep)
    var d = digits.charCodeAt(keep) - 48
    var up = d > 5
    if (d == 5) {
        var rest = false
        for (var i = keep + 1; i < digits.length; i++) { if (digits.charAt(i) != "0") { rest = true } }
        if (rest) { up = true } else if (keep > 0) {
            var last = head.charCodeAt(keep - 1) - 48
            up = last - Math.floor(last / 2) * 2 == 1
        }
    }
    if (!up) { return head == "" ? "0" : head }
    var res = ""
    var carry = 1
    for (var at = head.length - 1; at >= 0; at--) {
        var v = head.charCodeAt(at) - 48 + carry
        carry = 0
        if (v == 10) { v = 0; carry = 1 }
        res = "" + v + res
    }
    return carry == 1 ? "1" + res : res
}
