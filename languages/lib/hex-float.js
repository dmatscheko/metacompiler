// hex-float.js -- the value of a hexadecimal floating-point numeral, rounded to
// nearest with ties to even (to 53 bits, fewer for a subnormal), for the
// string-to-double conversions that read one: Python's float.fromhex, Swift's
// Double(String) and Ruby's Float(). Included by those interpreters and imported
// by their layer 2; the Go twins read the same numerals with strconv.ParseFloat,
// which is exact.
//
// Written in the subset both engines accept: no exponent literals, and powers of
// two are built by exact doubling.

// The numeral in s from index p on, after any sign and "0x": hex digits with at
// most one point, then an optional binary exponent p[+-]digits. Answers {v, end}
// - v the magnitude, Infinity past the double range, and end the index after the
// numeral - or null when there is no digit or a "p" has no digits.
function hexFloatScan(s, p0) {
    var p = p0
    var digs = []
    var frac = 0
    var seenPoint = false
    while (p < s.length) {
        var c = s.charAt(p)
        if (c == "." && !seenPoint) {
            seenPoint = true
        } else {
            var hv = "0123456789abcdef".indexOf(c.toLowerCase())
            if (hv < 0) { break }
            digs.push(hv)
            if (seenPoint) { frac = frac + 1 }
        }
        p = p + 1
    }
    if (digs.length == 0) { return null }
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
        if (nd == 0) { return null }
        if (eneg) { pexp = -pexp }
    }
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
    if (bits.length == 0) { return {v: 0, end: p} }
    // The leading bit's binary exponent, and how many bits the result keeps.
    var lead = bits.length - 1 + pexp - 4 * frac
    if (lead > 1024) { return {v: 1 / 0, end: p} }
    // Below half the smallest subnormal everything rounds to zero.
    if (lead < -1075) { return {v: 0, end: p} }
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
    if (keep < bits.length && bits[keep] == 1) {
        var sticky = false
        var r = keep + 1
        while (r < bits.length) {
            if (bits[r] == 1) { sticky = true }
            r = r + 1
        }
        if (sticky || q % 2 == 1) { q = q + 1 }
    }
    // q * 2^(lead - keep + 1), by exact doubling or halving: every intermediate
    // keeps q's significand, so it is exact whenever the result is.
    var v = q
    var k = lead - keep + 1
    while (k > 0) {
        v = v * 2
        k = k - 1
    }
    while (k < 0) {
        v = v / 2
        k = k + 1
    }
    return {v: v, end: p}
}
