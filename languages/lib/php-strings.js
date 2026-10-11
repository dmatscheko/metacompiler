// php-strings.js -- PHP's byte-string primitives: substr, strpos, strrpos,
// strcmp, ord, chr, the ASCII case maps and sprintf's float conversions. Everything else in
// PHP's string and array library is written in PHP itself (lib/php-stdlib.php)
// on top of these.
//
// Written once for the interpreter half (php-interpreter.abnf includes it) and for
// layer 2 (languages/lib/php-rt.metajs imports it), in the restricted subset both
// accept - see the head of regex.js. abnf/jsrtphpstr.go is the Go twin the
// compiler half runs under llvm.Run.
//
// A PHP string is a BYTE string, held here as text: its bytes are the UTF-8
// encoding of that text, so every offset below is a byte offset. A slice that
// cuts a character keeps each byte of the cut character as the character
// U+0000..U+00FF of the same value - what the Go twin's string(s[i]) answers for
// a single byte, and what `$s[$i]` already answers in the compiler half.
//
// Value-model free: subjects are host strings, positions plain integers, and
// argument coercion and the errors PHP raises stay with the includer.

// The UTF-8 bytes of s.
function phsBytes(s) {
    var out = []
    var i = 0
    while (i < s.length) {
        var c = s.charCodeAt(i)
        if (c >= 55296 && c <= 56319 && i + 1 < s.length) {
            var t = s.charCodeAt(i + 1)
            if (t >= 56320 && t <= 57343) {
                c = (c - 55296) * 1024 + (t - 56320) + 65536
                i = i + 1
            }
        }
        if (c < 128) { out.push(c) }
        else if (c < 2048) {
            out.push(192 + Math.floor(c / 64))
            out.push(128 + c % 64)
        } else if (c < 65536) {
            out.push(224 + Math.floor(c / 4096))
            out.push(128 + Math.floor(c / 64) % 64)
            out.push(128 + c % 64)
        } else {
            out.push(240 + Math.floor(c / 262144))
            out.push(128 + Math.floor(c / 4096) % 64)
            out.push(128 + Math.floor(c / 64) % 64)
            out.push(128 + c % 64)
        }
        i = i + 1
    }
    return out
}

// The text of bytes bs[from, to): a well-formed UTF-8 sequence is its character
// (Go's utf8.DecodeRune rules: no overlong form, no surrogate, nothing above
// U+10FFFF), and any other byte stands for itself.
function phsText(bs, from, to) {
    var out = ""
    var i = from
    while (i < to) {
        var b = bs[i]
        var n = 0
        var cp = 0
        if (b < 128) {
            n = 1
            cp = b
        } else if (b >= 194 && b <= 223) {
            n = 2
            cp = b - 192
        } else if (b >= 224 && b <= 239) {
            n = 3
            cp = b - 224
        } else if (b >= 240 && b <= 244) {
            n = 4
            cp = b - 240
        }
        var ok = n > 0 && i + n <= to
        var k = 1
        while (ok && k < n) {
            var cb = bs[i + k]
            if (cb < 128 || cb > 191) { ok = false }
            else { cp = cp * 64 + (cb - 128) }
            k = k + 1
        }
        if (ok && n == 3 && (cp < 2048 || (cp >= 55296 && cp <= 57343))) { ok = false }
        if (ok && n == 4 && (cp < 65536 || cp > 1114111)) { ok = false }
        if (!ok) {
            out = out + String.fromCharCode(b)
            i = i + 1
        } else {
            if (cp >= 65536) {
                var v = cp - 65536
                out = out + String.fromCharCode(55296 + Math.floor(v / 1024)) + String.fromCharCode(56320 + v % 1024)
            } else {
                out = out + String.fromCharCode(cp)
            }
            i = i + n
        }
    }
    return out
}

function phsAscii(s) { return byteLen(s) == s.length }

// substr(s, start, length): PHP 8's rules - a start past the end is "", a
// negative start or length counts from the end, and hasLen false is a null
// length (to the end).
function phsSubstr(s, start, len, hasLen) {
    var ascii = phsAscii(s)
    var n = ascii ? s.length : byteLen(s)
    var f = start
    if (f > n) { return "" }
    if (f < 0) {
        if (0 - f > n) { f = 0 } else { f = n + f }
    }
    var l = n - f
    if (hasLen) {
        if (len < 0) {
            if (0 - len > n - f) { l = 0 } else { l = n - f + len }
        } else if (len < n - f) {
            l = len
        }
    }
    if (l <= 0) { return "" }
    if (ascii) { return s.substring(f, f + l) }
    return phsText(phsBytes(s), f, f + l)
}

// Does needle nb occur in hb at byte p?
function phsAt(hb, nb, p) {
    var j = 0
    while (j < nb.length) {
        if (hb[p + j] != nb[j]) { return false }
        j = j + 1
    }
    return true
}

// strpos(h, needle, offset): the byte position of the first match at or after
// offset (a negative one counts from the end), -1 when there is none, and -2
// when the offset is outside the haystack (PHP's ValueError).
function phsStrpos(h, needle, off) {
    var ascii = phsAscii(h) && phsAscii(needle)
    var n = ascii ? h.length : byteLen(h)
    var o = off
    if (o < 0) { o = o + n }
    if (o < 0 || o > n) { return 0 - 2 }
    if (ascii) {
        var at = h.substring(o, n).indexOf(needle)
        if (at < 0) { return 0 - 1 }
        return at + o
    }
    var hb = phsBytes(h)
    var nb = phsBytes(needle)
    var p = o
    while (p + nb.length <= hb.length) {
        if (phsAt(hb, nb, p)) { return p }
        p = p + 1
    }
    return 0 - 1
}

// strrpos(h, needle, offset): the byte position of the LAST match. A
// non-negative offset is where the search starts; a negative one counts from the
// end and is where the match must END by, as zend_memnrstr is given it. -1 and
// -2 as for phsStrpos.
function phsStrrpos(h, needle, off) {
    var hb = phsBytes(h)
    var nb = phsBytes(needle)
    var n = hb.length
    var lo = 0
    var hi = n
    if (off >= 0) {
        if (off > n) { return 0 - 2 }
        lo = off
    } else {
        if (0 - off > n) { return 0 - 2 }
        if (0 - off >= nb.length) { hi = n + off + nb.length }
    }
    var p = hi - nb.length
    while (p >= lo) {
        if (phsAt(hb, nb, p)) { return p }
        p = p - 1
    }
    return 0 - 1
}

// strcmp(a, b): byte order, -1 / 0 / 1 (PHP 8.2).
function phsStrcmp(a, b) {
    if (a == b) { return 0 }
    if (phsAscii(a) && phsAscii(b)) { return a < b ? 0 - 1 : 1 }
    var ab = phsBytes(a)
    var bb = phsBytes(b)
    var i = 0
    while (i < ab.length && i < bb.length) {
        if (ab[i] != bb[i]) { return ab[i] < bb[i] ? 0 - 1 : 1 }
        i = i + 1
    }
    if (ab.length == bb.length) { return 0 }
    return ab.length < bb.length ? 0 - 1 : 1
}

// ord(s): the first byte, 0 for "".
function phsOrd(s) {
    if (s.length == 0) { return 0 }
    var c = s.charCodeAt(0)
    if (c < 128) { return c }
    return phsBytes(s.substring(0, 2))[0]
}

// chr(n): the one-byte string of n modulo 256.
function phsChr(n) {
    var b = n % 256
    if (b < 0) { b = b + 256 }
    return String.fromCharCode(b)
}

// strtolower / strtoupper: ASCII only, since PHP 8.2 - every other byte stays.
function phsCase(s, up) {
    var out = ""
    var i = 0
    while (i < s.length) {
        var c = s.charCodeAt(i)
        if (up && c >= 97 && c <= 122) { c = c - 32 }
        else if (!up && c >= 65 && c <= 90) { c = c + 32 }
        out = out + String.fromCharCode(c)
        i = i + 1
    }
    return out
}

// sprintf's %f and %e of a non-negative finite double a, from its exact decimal
// (exact(a) answers {i: <integer digits>, f: <fraction digits>}) rounded half to
// even by round(digits, keep): prec fraction digits for "f"; for "e" one digit,
// prec more after the point and an exponent that is signed and never padded
// (1.500000e+3), as php_conv_fp writes it.
function phsFmtFloat(a, conv, prec, exact, round) {
    var dec = exact(a)
    var ip = dec["i"]
    var fp = dec["f"]
    if (conv == "f") {
        var kept = round(ip + fp, ip.length + prec)
        var ilen = ip.length
        if (kept.length > ip.length + prec) { ilen = ilen + 1 }
        var ints = kept.substring(0, ilen)
        if (ints == "") { ints = "0" }
        if (prec == 0) { return ints }
        return ints + "." + kept.substring(ilen, kept.length)
    }
    var all = ip + fp
    var lead = 0
    while (lead < all.length && all.charAt(lead) == "0") { lead = lead + 1 }
    var e10 = 0
    var sig = "0"
    if (lead < all.length) {
        e10 = ip.length - lead - 1
        sig = all.substring(lead, all.length)
    }
    var d = round(sig, prec + 1)
    if (d.length > prec + 1) {
        d = d.substring(0, prec + 1)
        e10 = e10 + 1
    }
    var m = d.substring(0, 1)
    if (prec > 0) { m = m + "." + d.substring(1, d.length) }
    if (e10 < 0) { return m + "e-" + (0 - e10) }
    return m + "e+" + e10
}
