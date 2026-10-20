// php-strings.js -- PHP's byte-string primitives: strlen, substr, strpos,
// strrpos, strcmp, ord, chr, string offsets, concatenation, the ASCII case maps,
// sprintf's float conversions and the math library's double operations. Everything else in
// PHP's string and array library is written in PHP itself (lib/php-stdlib.php)
// on top of these.
//
// Written once for the interpreter half (php-interpreter.abnf includes it) and for
// layer 2 (languages/lib/php-rt.metajs imports it), in the restricted subset both
// accept - see the head of regex.js. abnf/jsrtphpstr.go is the Go twin the
// compiler half runs under llvm.Run.
//
// A PHP string is a BYTE string, held here as text: its bytes are the UTF-8
// encoding of that text, so every offset below is a byte offset. A byte that is
// not part of a well-formed UTF-8 sequence - a slice through a character, chr(200),
// "\xC8" - is held as the lone surrogate U+DC00 + byte (U+DC80..U+DCFF), which
// no well-formed text contains, so it stays ONE byte. Every string is kept in the
// canonical form phsText produces: a byte run that spells a character is that
// character, which is why concatenation goes through phsJoin.
//
// Value-model free: subjects are host strings, positions plain integers, and
// argument coercion and the errors PHP raises stay with the includer.

// Is unit c a byte held as U+DC00 + byte?
function phsIsByte(c) { return c >= 56448 && c <= 56575 }

// The bytes of s.
function phsBytes(s) {
    var out = []
    var i = 0
    while (i < s.length) {
        var c = s.charCodeAt(i)
        if (phsIsByte(c)) {
            out.push(c - 56320)
            i = i + 1
            continue
        }
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
// U+10FFFF), and any other byte is held as U+DC00 + byte.
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
            out = out + String.fromCharCode(56320 + b)
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

// The text of a BYTE string: one whose every unit is a byte value 0..255, as
// a string literal's escapes are assembled (the tag stack cannot carry a lone
// surrogate).
function phsFromBytes(t) {
    var bs = []
    var i = 0
    while (i < t.length) {
        bs.push(t.charCodeAt(i))
        i = i + 1
    }
    return phsText(bs, 0, bs.length)
}

// Does s hold a byte that is not part of a character?
function phsHasHeld(s) {
    var i = 0
    while (i < s.length) {
        if (phsIsByte(s.charCodeAt(i))) { return true }
        i = i + 1
    }
    return false
}

// echo s: PHP writes a string's BYTES, so one holding a held byte goes out
// through writeBytes (one unit per byte); any other string's UTF-8 text already
// is its bytes. Twin of js_phecho in abnf/jsrtphp.go.
function phsEcho(s) {
    if (phsAscii(s) || !phsHasHeld(s)) {
        print(s)
        return
    }
    var bs = phsBytes(s)
    var out = ""
    var i = 0
    while (i < bs.length) {
        out = out + String.fromCharCode(bs[i])
        i = i + 1
    }
    writeBytes(out)
}

// strlen: the byte count.
function phsLen(s) {
    if (phsAscii(s)) { return s.length }
    return phsBytes(s).length
}

// a . b: when a ends in held bytes and b starts with a held continuation byte,
// the bytes either side of the seam may spell a character, and are re-read. A
// UTF-8 sequence is at most four bytes, so three units either side suffice.
function phsJoin(a, b) {
    if (a.length == 0 || b.length == 0) { return a + b }
    var x = a.charCodeAt(a.length - 1)
    var y = b.charCodeAt(0)
    if (!phsIsByte(x) || y < 56448 || y > 56511) { return a + b }
    var i = a.length
    while (i > 0 && a.length - i < 3 && phsIsByte(a.charCodeAt(i - 1))) { i = i - 1 }
    var j = 0
    while (j < b.length && j < 3 && phsIsByte(b.charCodeAt(j))) { j = j + 1 }
    var bs = phsBytes(a.substring(i, a.length) + b.substring(0, j))
    return a.substring(0, i) + phsText(bs, 0, bs.length) + b.substring(j, b.length)
}

// $s[$i]: the one-byte string at byte offset i (from the end when negative), ""
// outside the string.
function phsOffset(s, i) {
    if (phsAscii(s)) {
        var k = i < 0 ? i + s.length : i
        if (k < 0 || k >= s.length) { return "" }
        return s.charAt(k)
    }
    var bs = phsBytes(s)
    var j = i < 0 ? i + bs.length : i
    if (j < 0 || j >= bs.length) { return "" }
    return phsText(bs, j, j + 1)
}

// $s[$i] = $c: the byte at offset i replaced by c's first byte, a write past the
// end padding with spaces. Answers null for an offset before the start.
function phsSetOffset(s, i, c) {
    var bs = phsBytes(s)
    var j = i < 0 ? i + bs.length : i
    if (j < 0) { return null }
    while (bs.length < j) { bs.push(32) }
    var cb = phsBytes(c)
    if (j == bs.length) { bs.push(cb[0]) } else { bs[j] = cb[0] }
    return phsText(bs, 0, bs.length)
}

// substr(s, start, length): PHP 8's rules - a start past the end is "", a
// negative start or length counts from the end, and hasLen false is a null
// length (to the end).
function phsSubstr(s, start, len, hasLen) {
    var ascii = phsAscii(s)
    var n = ascii ? s.length : phsLen(s)
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
    var n = ascii ? h.length : phsLen(h)
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
    if (b >= 128) { return String.fromCharCode(56320 + b) }
    return String.fromCharCode(b)
}

// The double operations PHP's math library is written on (lib/php-stdlib.php's
// __mec_math): the C library function of one or two doubles that op names.
function phsMath(op, a, b) {
    if (op == "floor") { return Math.floor(a) }
    if (op == "ceil") { return Math.ceil(a) }
    if (op == "trunc") { return Math.trunc(a) }
    if (op == "sqrt") { return Math.sqrt(a) }
    if (op == "exp") { return Math.exp(a) }
    if (op == "expm1") { return Math.expm1(a) }
    if (op == "log") { return Math.log(a) }
    if (op == "log1p") { return Math.log1p(a) }
    if (op == "log2") { return Math.log2(a) }
    if (op == "log10") { return Math.log10(a) }
    if (op == "sin") { return Math.sin(a) }
    if (op == "cos") { return Math.cos(a) }
    if (op == "tan") { return Math.tan(a) }
    if (op == "asin") { return Math.asin(a) }
    if (op == "acos") { return Math.acos(a) }
    if (op == "atan") { return Math.atan(a) }
    if (op == "sinh") { return Math.sinh(a) }
    if (op == "cosh") { return Math.cosh(a) }
    if (op == "tanh") { return Math.tanh(a) }
    if (op == "asinh") { return Math.asinh(a) }
    if (op == "acosh") { return Math.acosh(a) }
    if (op == "atanh") { return Math.atanh(a) }
    if (op == "atan2") { return Math.atan2(a, b) }
    if (op == "hypot") { return Math.hypot(a, b) }
    if (op == "pow") { return Math.pow(a, b) }
    if (op == "div") { return a / b }
    if (op == "fmod") { return a % b }
    // copysign: |a| with the sign of b, a negative zero's included.
    var m = Math.abs(a)
    if (b < 0 || (b == 0 && 1 / b < 0)) { return -m }
    return m
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

// Is the class name n (a leading "\" allowed, any letter case) one of the
// interfaces the engine itself declares - zend_interfaces.c, zend_exceptions.c,
// zend_enum.c and ext/json - which interface_exists reports though no program
// declares them?
function phsCoreIface(n) {
    var k = phsCase(n.charCodeAt(0) == 92 ? n.substring(1) : n, false)
    return k == "traversable" || k == "iterator" || k == "iteratoraggregate" || k == "arrayaccess" ||
        k == "countable" || k == "serializable" || k == "stringable" || k == "throwable" ||
        k == "unitenum" || k == "backedenum" || k == "jsonserializable"
}

// Does the engine's own interface iface extend the interface name (both any
// letter case, a leading "\" allowed)? Iterator and IteratorAggregate extend
// Traversable, BackedEnum extends UnitEnum and Throwable extends Stringable
// (zend_interfaces.c, zend_enum.c, zend_exceptions.c).
function phsCoreIfaceExtends(iface, name) {
    var q = phsCase(name.charCodeAt(0) == 92 ? name.substring(1) : name, false)
    if (q != "traversable" && q != "unitenum" && q != "stringable") { return false }
    var p = phsCase(iface.charCodeAt(0) == 92 ? iface.substring(1) : iface, false)
    if (q == "traversable") { return p == "iterator" || p == "iteratoraggregate" }
    if (q == "unitenum") { return p == "backedenum" }
    return p == "throwable"
}

// The engine's own final classes whose instances carry no descriptor: is the
// class cls ("closure" or "generator") the class name, or an implementor of it?
// A Generator implements Iterator, and so Traversable.
function phsCoreClassIs(cls, name) {
    var q = phsCase(name.charCodeAt(0) == 92 ? name.substring(1) : name, false)
    if (q == cls) { return true }
    return cls == "generator" && (q == "iterator" || q == "traversable")
}

// Is the class name n one of those classes, which class_exists reports though no
// program declares them? Answers its lower-case name, or "".
function phsCoreClass(n) {
    var k = phsCase(n.charCodeAt(0) == 92 ? n.substring(1) : n, false)
    if (k == "closure" || k == "generator") { return k }
    return ""
}

// a & b, a | b, a ^ b of two strings, byte by byte: '|' runs to the longer
// string and keeps its tail, '&' and '^' stop at the shorter. ~s is
// phsBitOp("~", s, s): every byte inverted.
function phsBitOp(op, a, b) {
    var x = phsBytes(a)
    var y = phsBytes(b)
    var n = x.length
    if (op == "|") {
        if (y.length > n) { n = y.length }
    } else if (y.length < n) {
        n = y.length
    }
    var out = []
    var i = 0
    while (i < n) {
        var p = i < x.length ? x[i] : 0
        var q = i < y.length ? y[i] : 0
        var r = 0
        var bit = 128
        while (bit >= 1) {
            var pb = p >= bit
            var qb = q >= bit
            if (pb) { p = p - bit }
            if (qb) { q = q - bit }
            var on = false
            if (op == "&") { on = pb && qb }
            else if (op == "|") { on = pb || qb }
            else if (op == "^") { on = pb != qb }
            else { on = !pb }
            if (on) { r = r + bit }
            bit = bit / 2
        }
        out.push(r)
        i = i + 1
    }
    return phsText(out, 0, out.length)
}

// $s++ on a string that is not numeric: Perl's increment over the trailing
// alphanumeric run - 'z' wraps to 'a', 'Z' to 'A' and '9' to '0' with a carry,
// and a carry out of the first character prepends '1', 'A' or 'a' after its
// kind. The run stops at the first other byte, so "a!" stays "a!". "" is "1".
// zend_operators.c increment_string.
function phsIncr(s) {
    if (s.length == 0) { return "1" }
    var bs = phsBytes(s)
    var pos = bs.length - 1
    var carry = false
    var last = 0
    while (pos >= 0) {
        var c = bs[pos]
        if (c >= 97 && c <= 122) {
            last = 97
            carry = c == 122
            bs[pos] = carry ? 97 : c + 1
        } else if (c >= 65 && c <= 90) {
            last = 65
            carry = c == 90
            bs[pos] = carry ? 65 : c + 1
        } else if (c >= 48 && c <= 57) {
            last = 49
            carry = c == 57
            bs[pos] = carry ? 48 : c + 1
        } else {
            carry = false
        }
        if (!carry) { break }
        pos = pos - 1
    }
    var t = phsText(bs, 0, bs.length)
    if (carry) { return String.fromCharCode(last) + t }
    return t
}

// The significant digits of a positive finite double a correctly rounded to nd
// (half to even on the exact decimal, as zend_dtoa's mode 2 rounds), trailing
// zeros dropped, and the decimal point's position: a ~= 0.<d> * 10^<e>. A value
// with fewer exact digits than nd keeps them all, however large nd is. exact and
// round are as for phsFmtFloat below. Twin of phpFloSplit's precision branch in
// abnf/jsrtphp.go.
function phsExactDigits(a, nd, exact, round) {
    var dec = exact(a)
    var ip = dec["i"]
    var all = ip + dec["f"]
    var lead = 0
    while (lead < all.length && all.charAt(lead) == "0") { lead = lead + 1 }
    var e = ip.length - lead
    var d = all.substring(lead, all.length)
    if (d.length > nd) {
        var r = round(d, nd)
        if (r.length > nd) {
            r = r.substring(0, nd)
            e = e + 1
        }
        d = r
    }
    var m = d.length
    while (m > 1 && d.charAt(m - 1) == "0") { m = m - 1 }
    return {d: d.substring(0, m), e: e}
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

// A closure literal's debug record (zend_closure_get_debug_info), fixed where the
// closure is written: its name, file and line, the names of its static variables
// (the use list, or an arrow function's captures, in order, with "$"), its
// parameters as var_dump keys ("$a", "&$b") and how many of them are required
// (up to the last one with no default that is not variadic). One string, the
// fields separated by newlines and the lists by commas.
function phsCloDesc(name, file, line, statics, pnames, pref, popt) {
    var nreq = 0
    var ps = ""
    var i = 0
    while (i < pnames.length) {
        if (!popt[i]) { nreq = i + 1 }
        ps = ps + (i > 0 ? "," : "") + (pref[i] ? "&" : "") + pnames[i]
        i = i + 1
    }
    var ss = ""
    var j = 0
    while (j < statics.length) {
        ss = ss + (j > 0 ? "," : "") + statics[j]
        j = j + 1
    }
    return name + "\n" + file + "\n" + line + "\n" + ss + "\n" + ps + "\n" + nreq
}

// s cut at every unit sep; "" is no fields at all.
function phsCloSplit(s, sep) {
    var out = []
    if (s == "") { return out }
    var from = 0
    var i = 0
    while (i <= s.length) {
        if (i == s.length || s.charCodeAt(i) == sep) {
            out.push(s.substring(from, i))
            from = i + 1
        }
        i = i + 1
    }
    return out
}

// The static variable names of record d, with "$".
function phsCloStatics(d) { return phsCloSplit(phsCloSplit(d, 10)[3], 44) }

// var_dump of a closure literal at indent ind: record d, handle id, the dumps of
// its static variables' values (at ind + 4) and of its bound $this (at ind + 2;
// "" for none), in zend_closure_get_debug_info's order: name, file, line,
// static, this, parameter. A record with no file is a named function's, which
// shows its name as "function" instead of the first three.
function phsCloDump(d, id, ind, statDumps, thisDump) {
    var f = phsCloSplit(d, 10)
    var st = phsCloSplit(f[3], 44)
    var ps = phsCloSplit(f[4], 44)
    var nreq = 0
    var k = 0
    while (k < f[5].length) {
        nreq = nreq * 10 + f[5].charCodeAt(k) - 48
        k = k + 1
    }
    var pad = ""
    while (pad.length < ind) { pad = pad + " " }
    var p2 = pad + "  "
    var p4 = p2 + "  "
    var n = f[1] == "" ? 1 : 3
    if (st.length > 0) { n = n + 1 }
    if (thisDump != "") { n = n + 1 }
    if (ps.length > 0) { n = n + 1 }
    var out = pad + "object(Closure)#" + id + " (" + n + ") {\n"
    if (f[1] == "") {
        out = out + p2 + "[\"function\"]=>\n" + p2 + "string(" + phsLen(f[0]) + ") \"" + f[0] + "\"\n"
    } else {
        out = out + p2 + "[\"name\"]=>\n" + p2 + "string(" + phsLen(f[0]) + ") \"" + f[0] + "\"\n"
        out = out + p2 + "[\"file\"]=>\n" + p2 + "string(" + phsLen(f[1]) + ") \"" + f[1] + "\"\n"
        out = out + p2 + "[\"line\"]=>\n" + p2 + "int(" + f[2] + ")\n"
    }
    if (st.length > 0) {
        out = out + p2 + "[\"static\"]=>\n" + p2 + "array(" + st.length + ") {\n"
        var i = 0
        while (i < st.length) {
            out = out + p4 + "[\"" + st[i].substring(1) + "\"]=>\n" + statDumps[i]
            i = i + 1
        }
        out = out + p2 + "}\n"
    }
    if (thisDump != "") { out = out + p2 + "[\"this\"]=>\n" + thisDump }
    if (ps.length > 0) {
        out = out + p2 + "[\"parameter\"]=>\n" + p2 + "array(" + ps.length + ") {\n"
        var j = 0
        while (j < ps.length) {
            out = out + p4 + "[\"" + ps[j] + "\"]=>\n" + p4 + "string(10) \"" + (j < nreq ? "<required>" : "<optional>") + "\"\n"
            j = j + 1
        }
        out = out + p2 + "}\n"
    }
    return out + pad + "}\n"
}

// The ArgumentCountError text of an internal function given `given` arguments
// outside [min, max] (max -1: variadic), zend_wrong_parameters_count_error's
// wording. Twin of phsArityMsg in abnf/jsrtphpstr.go.
function phsArityMsg(name, min, max, given) {
    var want = given < min ? min : max
    var how = min == max ? "exactly" : (given < min ? "at least" : "at most")
    return name + "() expects " + how + " " + want + " argument" + (want == 1 ? "" : "s") + ", " + given + " given"
}
