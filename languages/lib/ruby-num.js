// ruby-num.js -- MRI's string-to-number conversions: rb_cstr_to_dbl and its
// ruby_strtod (String#to_f, Kernel#Float), parse_rat (String#to_r,
// Kernel#Rational) and parse_comp (String#to_c, Kernel#Complex), for
// ruby-interpreter.abnf (include) and languages/lib/ruby-rt.metajs (import),
// each of which also brings lib/hex-float.js. Go twins: rubyStrToDbl in
// abnf/jsrt.go, rubyStrToRat / rubyStrToCpx in abnf/jsrtruby.go.
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

// How many doubles rbnFloatStep visits for a finite unit (ruby_float_step_size),
// which is also the #size of a Float range's Enumerator. Go twin:
// rubyFloatStepSize in abnf/jsrtrubystr.go.
function rbnFloatStepSize(beg, end, unit, excl) {
    var n = (end - beg) / unit
    var err = (Math.abs(beg) + Math.abs(end) + Math.abs(end - beg)) / Math.abs(unit) * Math.pow(2, 0 - 52)
    if (err > 0.5) { err = 0.5 }
    if (excl) {
        if (n <= 0) { return 0 }
        if (n < 1) { n = 0 } else { n = Math.floor(n - err) }
    } else {
        if (n < 0) { return 0 }
        n = Math.floor(n + err)
    }
    return n + 1
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
    var n = rbnFloatStepSize(beg, end, unit, excl) - 1
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

// ----- String#to_r / #to_c, Kernel#Rational(String) / #Complex(String) -----

// rb_int_parse_cstr(s from p, base 10, RB_INT_PARSE_UNDERSCORE): {e, d}, where
// the digits stop and the digits read (an underscore only between two digits),
// or null without a digit at p.
function rbnDigitsUs(s, p) {
    if (p >= s.length || !rbnDigit(s.charAt(p))) { return null }
    var d = ""
    var i = p
    while (i < s.length) {
        var c = s.charAt(i)
        if (rbnDigit(c)) { d = d + c }
        else if (c != "_" || i + 1 >= s.length || !rbnDigit(s.charAt(i + 1))) { break }
        i = i + 1
    }
    return {e: i, d: d}
}

// The value of a digit string as a double: an exponent only needs to be known
// to lie past rbnRatExpMax.
function rbnDigitsNum(d) {
    var v = 0
    var i = 0
    while (i < d.length) {
        v = v * 10 + (d.charCodeAt(i) - 48)
        i = i + 1
    }
    return v
}

// read_num of rational.c, from st.p: {d, x}, the significand's digits and the
// power of ten dividing it, with st.p past what was read; null when nothing
// was. A point needs no digit after it ("5." is 5), and an exponent letter is
// only read after a digit and before at least one more character.
function rbnRatNum(s, st) {
    var d = ""
    var ok = false
    if (st.p < s.length && s.charAt(st.p) != ".") {
        var a = rbnDigitsUs(s, st.p)
        if (a === null) { return null }
        st.p = a.e
        d = a.d
        ok = true
    }
    var fn = 0
    if (st.p < s.length && s.charAt(st.p) == ".") {
        st.p = st.p + 1
        var f = rbnDigitsUs(s, st.p)
        if (f === null) { return {d: d == "" ? "0" : d, x: 0} }
        st.p = f.e
        d = d + f.d
        fn = f.d.length
        ok = true
    }
    if (!ok) { return null }
    var x = fn
    if (st.p + 1 < s.length && (s.charAt(st.p) == "e" || s.charAt(st.p) == "E")) {
        st.p = st.p + 1
        var sg = s.charAt(st.p)
        if (sg == "+" || sg == "-") { st.p = st.p + 1 }
        var ex = rbnDigitsUs(s, st.p)
        if (ex === null) { return {d: d, x: x} }
        st.p = ex.e
        var e = rbnDigitsNum(ex.d)
        if (e != 0) { x = sg == "-" ? e + fn : fn - e }
    }
    return {d: d, x: x}
}

function rbnWsTail(s, p) {
    var i = p
    while (i < s.length && rbnSpace(s.charAt(i))) { i = i + 1 }
    return i == s.length
}

function rbnZeroDigits(d) {
    var i = 0
    while (i < d.length) {
        if (d.charAt(i) != "0") { return false }
        i = i + 1
    }
    return true
}

// From this power of ten MRI's 10**n is a Float ("in a**b, b may be too big":
// the Integer would pass BIGLEN_LIMIT, 32M bits), and parse_rat answers a Float.
var rbnRatExpMax = 9942067

// parse_rat of rational.c, behind String#to_r (strict false) and Rational(String)
// (strict true): leading whitespace, a sign, a significand, then "/" and a
// second one. Answers null when strict and s is no Rational; {zd: true} for a
// zero denominator (ZeroDivisionError even when not strict); {f: true, inf,
// neg} where MRI answers an infinite (inf) or a zero Float; else {neg, n, d,
// m}: the Rational (-1)^neg * n / d * 10^m, n and d decimal digit strings.
// Not strict, what is no number is 0 and the text after one is ignored.
function rbnStrToRat(s, strict) {
    var st = {p: 0}
    while (st.p < s.length && rbnSpace(s.charAt(st.p))) { st.p = st.p + 1 }
    var neg = false
    if (st.p < s.length && (s.charAt(st.p) == "+" || s.charAt(st.p) == "-")) {
        neg = s.charAt(st.p) == "-"
        st.p = st.p + 1
    }
    var num = rbnRatNum(s, st)
    if (num === null) {
        if (strict) { return null }
        return {zd: false, f: false, neg: false, n: "0", d: "1", m: 0}
    }
    var den = "1"
    var x = num.x
    if (st.p < s.length && s.charAt(st.p) == "/") {
        st.p = st.p + 1
        var dn = rbnRatNum(s, st)
        if (dn === null) {
            if (strict) { return null }
        } else if (rbnZeroDigits(dn.d)) {
            return {zd: true}
        } else if (strict && !rbnWsTail(s, st.p)) {
            return null
        } else {
            den = dn.d
            x = x - dn.x
        }
    } else if (strict && !rbnWsTail(s, st.p)) {
        return null
    }
    if (x <= 0 - rbnRatExpMax || x >= rbnRatExpMax) { return {zd: false, f: true, inf: x < 0, neg: neg} }
    return {zd: false, f: false, neg: neg, n: num.d, d: den, m: 0 - x}
}

// String#to_c and Complex(String): parse_comp of complex.c. Each part is read
// into a buffer, digits and underscores apart, and becomes a number the way
// str2num makes one (rbnCpxKind). Answers {ok, re, im, polar}: the two
// buffers, polar when they are a modulus and an angle (the "r@theta" form), ok
// false where Complex(String) raises.
function rbnStrToCpx(s, strict) {
    var st = {p: 0, b: ""}
    while (st.p < s.length && rbnSpace(s.charAt(st.p))) { st.p = st.p + 1 }
    var r = rbnCpxRead(s, st, strict)
    if (r.ok) {
        while (st.p < s.length && rbnSpace(s.charAt(st.p))) { st.p = st.p + 1 }
        if (strict && st.p < s.length) { r.ok = false }
    }
    return r
}

function rbnImagUnit(c) { return c == "i" || c == "I" || c == "j" || c == "J" }

// read_sign of complex.c: a sign is copied into the buffer.
function rbnCpxSign(s, st) {
    var c = s.charAt(st.p)
    if (st.p < s.length && (c == "+" || c == "-")) {
        st.b = st.b + c
        st.p = st.p + 1
        return c
    }
    return ""
}

// read_digits of complex.c. A doubled underscore ends the read only when
// strict; after a trailing underscore the position steps back over it onto the
// last digit, as MRI's do-while does.
function rbnCpxDigits(s, st, strict) {
    if (st.p >= s.length || !rbnDigit(s.charAt(st.p))) { return false }
    var us = true
    while (st.p < s.length && (rbnDigit(s.charAt(st.p)) || s.charAt(st.p) == "_")) {
        if (s.charAt(st.p) == "_") {
            if (strict && us) { return false }
            us = true
        } else {
            st.b = st.b + s.charAt(st.p)
            us = false
        }
        st.p = st.p + 1
    }
    if (us) {
        st.p = st.p - 1
        while (s.charAt(st.p) == "_") { st.p = st.p - 1 }
    }
    return true
}

// The buffer without the character a failed read put there.
function rbnCpxDrop(st) { st.b = st.b.substring(0, st.b.length - 1) }

// read_num of complex.c: digits, then "." and digits, then e[sign]digits.
function rbnCpxNum(s, st, strict) {
    if (st.p >= s.length || s.charAt(st.p) != ".") {
        if (!rbnCpxDigits(s, st, strict)) { return false }
    }
    if (st.p < s.length && s.charAt(st.p) == ".") {
        st.b = st.b + "."
        st.p = st.p + 1
        if (!rbnCpxDigits(s, st, strict)) {
            rbnCpxDrop(st)
            return false
        }
    }
    if (st.p < s.length && (s.charAt(st.p) == "e" || s.charAt(st.p) == "E")) {
        st.b = st.b + s.charAt(st.p)
        st.p = st.p + 1
        rbnCpxSign(s, st)
        if (!rbnCpxDigits(s, st, strict)) {
            rbnCpxDrop(st)
            return false
        }
    }
    return true
}

// read_rat_nos of complex.c: a number, then "/" and digits.
function rbnCpxRat(s, st, strict) {
    if (!rbnCpxNum(s, st, strict)) { return false }
    if (st.p < s.length && s.charAt(st.p) == "/") {
        st.b = st.b + "/"
        st.p = st.p + 1
        if (!rbnCpxDigits(s, st, strict)) {
            rbnCpxDrop(st)
            return false
        }
    }
    return true
}

// The buffer read so far, the next part starting a fresh one.
function rbnCpxTake(st) {
    var t = st.b
    st.b = ""
    return t
}

// read_comp of complex.c.
function rbnCpxRead(s, st, strict) {
    var sign = rbnCpxSign(s, st)
    if (st.p < s.length && rbnImagUnit(s.charAt(st.p))) {
        st.p = st.p + 1
        return {ok: true, re: "0", im: sign == "-" ? "-1" : "1", polar: false}
    }
    if (!rbnCpxRat(s, st, strict)) { return {ok: false, re: rbnCpxTake(st), im: "0", polar: false} }
    var num = rbnCpxTake(st)
    if (st.p < s.length && rbnImagUnit(s.charAt(st.p))) {
        st.p = st.p + 1
        return {ok: true, re: "0", im: num, polar: false}
    }
    if (st.p < s.length && s.charAt(st.p) == "@") {
        st.p = st.p + 1
        rbnCpxSign(s, st)
        var pok = rbnCpxRat(s, st, strict)
        var ang = rbnCpxTake(st)
        if (ang.length < 1 || !rbnDigit(ang.charAt(ang.length - 1))) { return {ok: false, re: num, im: "0", polar: false} }
        return {ok: pok, re: num, im: ang, polar: true}
    }
    if (st.p < s.length && (s.charAt(st.p) == "+" || s.charAt(st.p) == "-")) {
        var num2 = rbnCpxSign(s, st) == "-" ? "-1" : "1"
        if (st.p >= s.length || !rbnImagUnit(s.charAt(st.p))) {
            if (!rbnCpxRat(s, st, strict)) { return {ok: false, re: num, im: "0", polar: false} }
            num2 = rbnCpxTake(st)
        }
        if (st.p >= s.length || !rbnImagUnit(s.charAt(st.p))) { return {ok: false, re: num, im: "0", polar: false} }
        st.p = st.p + 1
        return {ok: true, re: num, im: num2, polar: false}
    }
    return {ok: true, re: num, im: "0", polar: false}
}

// str2num of complex.c: the kind of number a part's buffer makes - "r" a
// Rational (rb_cstr_to_rat), "f" a Float (rb_cstr_to_dbl), "i" an Integer
// (rb_cstr_to_inum), each lenient.
function rbnCpxKind(t) {
    if (t.indexOf("/") >= 0) { return "r" }
    if (t.indexOf(".") >= 0 || t.indexOf("e") >= 0 || t.indexOf("E") >= 0) { return "f" }
    return "i"
}
