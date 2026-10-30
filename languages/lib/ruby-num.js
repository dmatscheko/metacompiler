// ruby-num.js -- MRI's string-to-Float conversion (rb_cstr_to_dbl and its
// ruby_strtod), behind String#to_f and Kernel#Float, for ruby-interpreter.abnf
// (include) and languages/lib/ruby-rt.metajs (import), each of which also brings
// lib/hex-float.js. Go twin: rubyStrToDbl in abnf/jsrt.go.
//
// Written in the subset both engines accept: no exponent literals.

// MRI's ISSPACE, by character code: a MetaJS string literal has no \f or \v
// escape (natively "\f" is "f").
function rbnSpace(c) {
    var cc = c.charCodeAt(0)
    return cc == 32 || (cc >= 9 && cc <= 13)
}
function rbnDigit(c) { return c >= "0" && c <= "9" }
function rbnHexDigit(c) { return rbnDigit(c) || (c >= "a" && c <= "f") || (c >= "A" && c <= "F") }

// ruby_strtod on s from p0: {v, end}, end == p0 when nothing converts. A sign,
// then either "0x", hex digits, an optional point and hex digits and a binary
// exponent p[+-]digits that is required after a point; or digits with an
// optional point that needs a digit after it, and an optional e[+-]digits. No
// inf or nan. Out of range is an infinity or a zero.
function rbnStrtod(s, p0) {
    var p = p0
    var neg = false
    if (p < s.length && (s.charAt(p) == "+" || s.charAt(p) == "-")) {
        neg = s.charAt(p) == "-"
        p = p + 1
    }
    var q = p
    if (p + 1 < s.length && s.charAt(p) == "0" && (s.charAt(p + 1) == "x" || s.charAt(p + 1) == "X")) {
        q = p + 2
        if (q >= s.length || !rbnHexDigit(s.charAt(q))) { return {v: 0, end: p0} }
        while (q < s.length && rbnHexDigit(s.charAt(q))) { q = q + 1 }
        var point = q < s.length && s.charAt(q) == "."
        if (point) {
            q = q + 1
            if (q >= s.length || !rbnHexDigit(s.charAt(q))) { return {v: 0, end: p0} }
            while (q < s.length && rbnHexDigit(s.charAt(q))) { q = q + 1 }
        }
        if (q < s.length && (s.charAt(q) == "p" || s.charAt(q) == "P")) {
            q = q + 1
            if (q < s.length && (s.charAt(q) == "+" || s.charAt(q) == "-")) { q = q + 1 }
            var e0 = q
            while (q < s.length && rbnDigit(s.charAt(q))) { q = q + 1 }
            if (q == e0) { return {v: 0, end: p0} }
        } else if (point) {
            return {v: 0, end: p0}
        }
        var h = hexFloatScan(s.substring(p + 2, q), 0)
        var hv = h["v"]
        if (hv == 0) { return {v: neg ? 0 / (0 - 1) : 0, end: q} }
        return {v: neg ? -hv : hv, end: q}
    }
    var nd = 0
    while (q < s.length && rbnDigit(s.charAt(q))) {
        q = q + 1
        nd = nd + 1
    }
    if (q + 1 < s.length && s.charAt(q) == "." && rbnDigit(s.charAt(q + 1))) {
        q = q + 1
        while (q < s.length && rbnDigit(s.charAt(q))) {
            q = q + 1
            nd = nd + 1
        }
    }
    if (nd == 0) { return {v: 0, end: p0} }
    if (q < s.length && (s.charAt(q) == "e" || s.charAt(q) == "E")) {
        var k = q + 1
        if (k < s.length && (s.charAt(k) == "+" || s.charAt(k) == "-")) { k = k + 1 }
        var k0 = k
        while (k < s.length && rbnDigit(s.charAt(k))) { k = k + 1 }
        if (k > k0) { q = k }
    }
    return {v: parseFloat(s.substring(p0, q)), end: q}
}

// rb_cstr_to_dbl: String#to_f when badcheck is false (the leading numeral, 0.0
// when there is none) and Float(String) when it is true (the whole string, or
// null). Where ruby_strtod stops early, MRI copies the string into a 70-byte
// buffer dropping each underscore that stands between two digits, keeping at
// most 60 characters ahead of an exponent letter and skipping leading zeros, and
// reads the copy again. A string that starts "0x" is 0.0 to to_f: only Float()
// reads it as hexadecimal.
function rbnStrToDbl(s, badcheck) {
    var p = 0
    while (p < s.length && rbnSpace(s.charAt(p))) { p = p + 1 }
    if (!badcheck && p + 1 < s.length && s.charAt(p) == "0" && (s.charAt(p + 1) == "x" || s.charAt(p + 1) == "X")) { return 0 }
    var r = rbnStrtod(s, p)
    if (r["end"] == p) {
        if (badcheck) { return null }
        return r["v"]
    }
    if (r["end"] == s.length) { return r["v"] }
    var end = r["end"]
    var buf = ""
    var cap = 60
    var prev = ""
    if (s.charAt(p) == "+" || s.charAt(p) == "-") {
        prev = s.charAt(p)
        buf = buf + prev
        p = p + 1
    }
    if (s.charAt(p) == "0") {
        prev = "0"
        buf = buf + "0"
        p = p + 1
        while (p < s.length && s.charAt(p) == "0") { p = p + 1 }
    }
    while (p < end && buf.length < cap) {
        prev = s.charAt(p)
        buf = buf + prev
        p = p + 1
    }
    var dots = 0
    while (p < s.length) {
        if (s.charAt(p) == "_") {
            if (buf.length == 0 || !rbnDigit(prev) || p + 1 >= s.length || !rbnDigit(s.charAt(p + 1))) {
                if (badcheck) { return null }
                break
            }
            p = p + 1
        }
        prev = s.charAt(p)
        p = p + 1
        if (cap == 60 && (prev == "e" || prev == "E" || prev == "p" || prev == "P")) {
            cap = 69
            buf = buf + prev
            if (p < s.length && (s.charAt(p) == "+" || s.charAt(p) == "-")) {
                prev = s.charAt(p)
                buf = buf + prev
                p = p + 1
            }
            if (p < s.length && s.charAt(p) == "0") {
                prev = "0"
                buf = buf + "0"
                p = p + 1
                while (p < s.length && s.charAt(p) == "0") { p = p + 1 }
            }
            continue
        }
        if (rbnSpace(prev)) {
            while (p < s.length && rbnSpace(s.charAt(p))) { p = p + 1 }
            if (p < s.length) {
                if (badcheck) { return null }
                break
            }
        } else if (prev == ".") {
            dots = dots + 1
            if (dots > 1) {
                if (badcheck) { return null }
                break
            }
        } else if (!rbnDigit(prev)) {
            if (badcheck) { return null }
            break
        }
        if (buf.length < cap) { buf = buf + prev }
    }
    if (!badcheck && buf.length > 1 && buf.charAt(0) == "0" && (buf.charAt(1) == "x" || buf.charAt(1) == "X")) { return 0 }
    var r2 = rbnStrtod(buf, 0)
    if (badcheck) {
        if (r2["end"] == 0) { return null }
        var k = r2["end"]
        while (k < buf.length && rbnSpace(buf.charAt(k))) { k = k + 1 }
        if (k < buf.length) { return null }
    }
    return r2["v"]
}

// ----- the half: keyword of #round (numeric.c, rb_num_get_rounding_option) -----
// The mode letters the digit rounders share: "r" rounds a tie away from zero (the
// default, half: :up), "re" to the even neighbour, "rd" towards zero. "" for a
// name that is no mode, which the caller raises as ArgumentError.
// A negative Float to a non-integral power is a Complex: MRI's
// rb_dbl_complex_new_polar_pi(abs, ang), abs * e^(i*pi*ang), as [re, im]. An
// angle of an odd half is exactly imaginary. Go twin: rubyPolarPi in
// abnf/jsrtruby.go.
function rbnPolarPi(abs, ang) {
    var fi = Math.trunc(ang)
    var fr = ang - fi
    var pos = fr == 0.5
    if (pos || fr == 0 - 0.5) {
        var h = fi / 2
        var a = abs
        if (((h - Math.trunc(h)) != fr) != pos) { a = -a }
        return [0, a]
    }
    return [abs * Math.cos(Math.PI * ang), abs * Math.sin(Math.PI * ang)]
}

function rbnHalfMode(name) {
    if (name == "" || name == "up") { return "r" }
    if (name == "even") { return "re" }
    if (name == "down") { return "rd" }
    return ""
}

// round_half_even / round_half_down of numeric.c: x * s rounded to an integral
// double under mode "re" / "rd". The tests of (f + 0.5) / s and (f - 0.5) / s are
// MRI's correction for a product that rounded to a midpoint x does not reach.
function rbnRoundHalf(x, s, mode) {
    var xs = x * s
    var f = 0
    var d = 0
    if (mode == "rd") {
        f = Math.trunc(xs)
        d = xs - f
        if (d >= 0.5) { f = f + 1 } else if (d <= 0 - 0.5) { f = f - 1 }
        if (x > 0 && (f - 0.5) / s >= x) { f = f - 1 }
        if (x < 0 && (f + 0.5) / s <= x) { f = f + 1 }
        return f
    }
    if (x > 0) {
        f = Math.floor(xs)
        d = xs - f
        if (d > 0.5) { d = 1 } else if (d == 0.5 || (f + 0.5) / s <= x) { d = f % 2 } else { d = 0 }
        return f + d
    }
    if (x < 0) {
        f = Math.ceil(xs)
        d = f - xs
        if (d > 0.5) { d = 1 } else if (d == 0.5 || (f - 0.5) / s >= x) { d = (0 - f) % 2 } else { d = 0 }
        return f - d
    }
    return x
}

// The doubles a Range#step over Floats visits (ruby_float_step in numeric.c): the
// count is decided once, allowing for the rounding error of (end - beg) / unit,
// so the last step is neither lost nor added, and no value passes the end.
function rbnFloatStep(beg, end, unit, excl) {
    var out = []
    if (unit - unit !== 0) {
        if (unit > 0 ? beg <= end : beg >= end) { out.push(beg) }
        return out
    }
    var n = (end - beg) / unit
    var err = (Math.abs(beg) + Math.abs(end) + Math.abs(end - beg)) / Math.abs(unit) * Math.pow(2, 0 - 52)
    if (err > 0.5) { err = 0.5 }
    if (excl) {
        if (n <= 0) { return out }
        if (n < 1) { n = 0 } else { n = Math.floor(n - err) }
    } else {
        if (n < 0) { return out }
        n = Math.floor(n + err)
    }
    var i = 0
    while (i <= n) {
        var d = i * unit + beg
        if (unit >= 0 ? end < d : d < end) { d = end }
        if (!excl || (unit >= 0 ? d < end : d > end)) { out.push(d) }
        i = i + 1
    }
    return out
}

// Integer#step(limit, by) over Integers: from, from + by, ... while not past limit.
function rbnIntStep(from, limit, by) {
    var out = []
    var v = from
    while (by > 0 ? v <= limit : v >= limit) {
        out.push(v)
        v = v + by
    }
    return out
}

// One step of Array#sum's Kahan-Babuska summation (ary_add_one in array.c):
// st.f is the running sum, st.c the compensation answered as f + c at the end. A
// NaN sticks, and infinities of opposite signs make one.
function rbnKahan(st, x) {
    var f = st.f
    if (f !== f) { return }
    if (x !== x || (x - x !== 0 && f - f === 0)) {
        st.f = x
        return
    }
    if (x - x !== 0) {
        if ((x > 0) != (f > 0)) { st.f = x - x } else { st.f = x }
        return
    }
    if (f - f !== 0) { return }
    var t = f + x
    if (Math.abs(f) >= Math.abs(x)) { st.c = st.c + ((f - t) + x) } else { st.c = st.c + ((x - t) + f) }
    st.f = t
}

// Whether an Integer rounded at a negative ndigits moves away from zero, given the
// kept digits (head) and the dropped ones (tail, not all zero): past the midpoint
// always, at it by the mode - "r" away, "rd" never, "re" to an even last digit.
function rbnHalfAway(mode, head, tail) {
    var c = tail.charCodeAt(0) - 48
    if (c != 5 || mode == "r") { return c >= 5 }
    var i = 1
    while (i < tail.length) {
        if (tail.charAt(i) != "0") { return true }
        i = i + 1
    }
    if (mode == "rd" || head.length == 0) { return false }
    return (head.charCodeAt(head.length - 1) - 48) % 2 == 1
}
