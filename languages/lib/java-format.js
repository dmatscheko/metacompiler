// java-format.js -- java.lang.String.format (java.util.Formatter, Locale.US) over
// plain strings. Kotlin/JVM's String.format IS this formatter, so it serves
// java-interpreter.abnf and kotlin-interpreter.abnf (include) and
// lib/java-rt.metajs and lib/kotlin-rt.metajs (import). abnf/jsrtjavastr.go's
// jxFormat is the Go twin's port.
//
// What depends on the value model is asked through hooks each importer defines:
//   jxkKind(v)       "null" "bool" "char" "int" "float" "str" or "obj"
//   jxkStr(v)        String.valueOf(v)
//   jxkCode(v)       a char's code, or an int's value
//   jxkDec(v)        an integral value's signed decimal digits
//   jxkRadix(v, r)   its two's complement digits in radix r at its own width
//   jxkDbl(v)        a float's value as a double
//   jxkDblStr(a)     Double.toString(a) of a double a > 0
//   jxkClass(v)      v.getClass().getName() of a non-null v
//   jxkHash(v)       v.hashCode() of a non-null v
//   jxkThrow(c, msg) throw a new built-in exception of simple class name c with
//                    message msg
//   jxkFail(msg)     a format conversion this subset does not support

// A code point as its one or two UTF-16 units.
function jxCpStr(cp) {
    if (cp < 65536) { return String.fromCharCode(cp) }
    var v = cp - 65536
    return String.fromCharCode(55296 + Math.floor(v / 1024)) + String.fromCharCode(56320 + v % 1024)
}

// ----- String.format (java.util.Formatter, Locale.US) -----
//
// %[index$][flags][width][.precision]conversion with the conversions
// b s h c d o x e f g a % n (and their upper-case forms). A double is formatted
// from its Double.toString digits rounded HALF_UP, which is Java's rule -
// %.2f of 1.005 is 1.01 and %.1f of 0.15 is 0.2 - and a float is widened to a
// double first. A malformed specifier, a missing argument and a mismatched one
// throw java.util.Formatter's exceptions (jxkThrow); the date conversion t is
// reported through jxkFail.
function jxFormat(fmt, args) {
    var out = ""
    var i = 0
    var next = 0
    var last = 0 - 1
    while (i < fmt.length) {
        var c = fmt.charAt(i)
        if (c != "%") {
            out = out + c
            i = i + 1
            continue
        }
        var start = i
        var j = i + 1
        var k = j
        while (k < fmt.length && jxDigit(fmt.charCodeAt(k))) { k = k + 1 }
        var idx = 0 - 1
        if (k > j && k < fmt.length && fmt.charAt(k) == "$") {
            idx = parseInt(fmt.substring(j, k), 10) - 1
            if (idx < 0) { jxkThrow("IllegalFormatArgumentIndexException", "Illegal format argument index = " + (idx + 1)) }
            j = k + 1
        }
        var flags = ""
        while (j < fmt.length && "-#+ 0,(<".indexOf(fmt.charAt(j)) >= 0) {
            if (jxHas(flags, fmt.charAt(j))) { jxkThrow("DuplicateFormatFlagsException", "Flags = '" + fmt.charAt(j) + "'") }
            flags = flags + fmt.charAt(j)
            j = j + 1
        }
        var width = 0 - 1
        k = j
        while (k < fmt.length && jxDigit(fmt.charCodeAt(k))) { k = k + 1 }
        if (k > j) { width = parseInt(fmt.substring(j, k), 10) }
        j = k
        var prec = 0 - 1
        var bad = false
        if (j < fmt.length && fmt.charAt(j) == ".") {
            k = j + 1
            while (k < fmt.length && jxDigit(fmt.charCodeAt(k))) { k = k + 1 }
            if (k == j + 1) { bad = true }
            prec = parseInt(fmt.substring(j + 1, k), 10)
            j = k
        }
        // A specifier that does not parse names the character after the '%'.
        if (bad || j >= fmt.length) { jxkThrow("UnknownFormatConversionException", "Conversion = '" + (start + 1 < fmt.length ? fmt.charAt(start + 1) : "%") + "'") }
        var conv = fmt.charAt(j)
        i = j + 1
        jxCheckSpec(conv, flags, width, prec, idx)
        if (conv == "n") {
            out = out + "\n"
            continue
        }
        if (conv == "%") {
            out = out + jxPad("%", flags, width)
            continue
        }
        var ai = next
        if (flags.indexOf("<") >= 0) { ai = last } else if (idx >= 0) { ai = idx } else { next = next + 1 }
        if (ai < 0 || ai >= args.length) { jxkThrow("MissingFormatArgumentException", "Format specifier '" + fmt.substring(start, i) + "'") }
        last = ai
        out = out + jxConv(conv, flags, width, prec, args[ai])
    }
    return out
}
// The checks java.util.Formatter makes of a specifier before it reads the
// argument (FormatSpecifier's constructor): the conversion, the flags and the
// precision it takes, the width '-' and '0' need.
function jxCheckSpec(conv, flags, width, prec, idx) {
    if ("bBhHsScCdoxXeEfgGaAtT%n".indexOf(conv) < 0) { jxkThrow("UnknownFormatConversionException", "Conversion = '" + conv + "'") }
    var lc = conv.toLowerCase()
    var sp = jxSpecStr(conv, flags, width, prec, idx)
    if (lc == "%" || lc == "n") {
        if (prec >= 0) { jxkThrow("IllegalFormatPrecisionException", "" + prec) }
        if (lc == "n" && width >= 0) { jxkThrow("IllegalFormatWidthException", "" + width) }
        if (flags != "" && (lc == "n" || flags != "-")) { jxkThrow("IllegalFormatFlagsException", "Flags = '" + jxFlagsCanon(flags) + "'") }
        if (width < 0 && jxHas(flags, "-")) { jxkThrow("MissingFormatWidthException", sp) }
        return undefined
    }
    if (lc == "b" || lc == "h" || lc == "s") {
        if (lc != "s" && jxHas(flags, "#")) { jxMismatch(lc, "#") }
        if (width < 0 && jxHas(flags, "-")) { jxkThrow("MissingFormatWidthException", sp) }
        jxBadFlags(lc, flags, "+ 0,(")
        return undefined
    }
    if (lc == "c") {
        if (prec >= 0) { jxkThrow("IllegalFormatPrecisionException", "" + prec) }
        jxBadFlags(lc, flags, "#+ 0,(")
        if (width < 0 && jxHas(flags, "-")) { jxkThrow("MissingFormatWidthException", sp) }
        return undefined
    }
    if (lc == "t") { return undefined }
    if (width < 0 && (jxHas(flags, "-") || jxHas(flags, "0"))) { jxkThrow("MissingFormatWidthException", sp) }
    if ((jxHas(flags, "+") && jxHas(flags, " ")) || (jxHas(flags, "-") && jxHas(flags, "0"))) {
        jxkThrow("IllegalFormatFlagsException", "Flags = '" + jxFlagsCanon(flags) + "'")
    }
    if (lc == "d" || lc == "o" || lc == "x") {
        if (prec >= 0) { jxkThrow("IllegalFormatPrecisionException", "" + prec) }
        jxBadFlags(lc, flags, lc == "d" ? "#" : ",")
    } else if (lc == "a") {
        jxBadFlags(lc, flags, "(,")
    } else if (lc == "e") {
        jxBadFlags(lc, flags, ",")
    } else if (lc == "g") {
        jxBadFlags(lc, flags, "#")
    }
    return undefined
}
function jxBadFlags(lc, flags, bad) {
    for (var i = 0; i < bad.length; i++) {
        if (jxHas(flags, bad.charAt(i))) { jxMismatch(lc, bad.charAt(i)) }
    }
    return undefined
}
function jxMismatch(lc, flag) { jxkThrow("FormatFlagsConversionMismatchException", "Conversion = " + lc + ", Flags = " + flag) }
// The flags in java.util.Formatter's own order, as its messages spell them.
function jxFlagsCanon(flags) {
    var all = "-#+ 0,(<"
    var out = ""
    for (var i = 0; i < all.length; i++) {
        if (jxHas(flags, all.charAt(i))) { out = out + all.charAt(i) }
    }
    return out
}
// The specifier as Formatter's toString writes it back.
function jxSpecStr(conv, flags, width, prec, idx) {
    var out = "%" + jxFlagsCanon(flags)
    if (idx >= 0) { out = out + (idx + 1) + "$" }
    if (width >= 0) { out = out + width }
    if (prec >= 0) { out = out + "." + prec }
    return out + conv
}
function jxDigit(c) { return c >= 48 && c <= 57 }
function jxHas(flags, f) { return flags.indexOf(f) >= 0 }
// Right-justified with spaces, or left-justified with the '-' flag.
function jxPad(s, flags, width) {
    var out = s
    if (jxHas(flags, "-")) {
        while (out.length < width) { out = out + " " }
        return out
    }
    while (out.length < width) { out = " " + out }
    return out
}
function jxConv(conv, flags, width, prec, v) {
    var lc = conv.toLowerCase()
    var up = conv != lc
    var k = jxkKind(v)
    var body = ""
    if (lc == "s") {
        // '#' is for a Formattable, which no value here is.
        if (jxHas(flags, "#")) { jxMismatch(lc, "#") }
        body = k == "null" ? "null" : jxkStr(v)
        if (prec >= 0 && prec < body.length) { body = body.substring(0, prec) }
    } else if (lc == "h") {
        body = k == "null" ? "null" : jxHex32(jxkHash(v))
        if (prec >= 0 && prec < body.length) { body = body.substring(0, prec) }
    } else if (lc == "b") {
        if (k == "null") { body = "false" } else if (k == "bool") { body = jxkStr(v) } else { body = "true" }
        if (prec >= 0 && prec < body.length) { body = body.substring(0, prec) }
    } else if (lc == "c") {
        if (k == "null") { body = "null" }
        else if (k == "char") { body = jxCpStr(jxkCode(v)) }
        else if (k == "int" && jxkClass(v) != "java.lang.Long") {
            var cp = jxkCode(v)
            if (cp < 0 || cp > 1114111) { jxkThrow("IllegalFormatCodePointException", "Code point = 0x" + jxHex32(cp)) }
            body = jxCpStr(cp)
        }
        else { jxBadConv(lc, v) }
    } else if (lc == "d") {
        if (k == "null") { body = "null" } else {
            if (k != "int") { jxBadConv(lc, v) }
            var dd = jxkDec(v)
            var neg = dd.charAt(0) == "-"
            if (neg) { dd = dd.substring(1) }
            if (jxHas(flags, ",")) { dd = jxGroup(dd) }
            return jxNum(neg, dd, flags, width, up)
        }
    } else if (lc == "x" || lc == "o") {
        if (k == "null") { body = "null" } else {
            if (k != "int") { jxBadConv(lc, v) }
            // A sign or a bracket is for a BigInteger only.
            jxBadFlags(lc, flags, "( +")
            var rd = jxkRadix(v, lc == "x" ? 16 : 8)
            if (jxHas(flags, "#")) { rd = (lc == "x" ? "0x" : "0") + rd }
            if (jxHas(flags, "0")) {
                while (rd.length < width) { rd = (lc == "x" && jxHas(flags, "#")) ? "0x0" + rd.substring(2) : "0" + rd }
            }
            body = rd
        }
    } else if (lc == "f" || lc == "e" || lc == "g" || lc == "a") {
        if (k == "null") { body = "null" } else {
            if (k != "float") { jxBadConv(lc, v) }
            if (lc == "a") { return jxHexFloat(jxkDbl(v), flags, width, prec, up) }
            return jxFloat(lc, jxkDbl(v), flags, width, prec, up)
        }
    } else {
        jxkFail("unsupported conversion %" + conv)
    }
    // A null argument prints "null", cut to the precision like a string.
    if (k == "null" && prec >= 0 && prec < body.length) { body = body.substring(0, prec) }
    if (up) { body = unicodeCase(body, 3) }
    return jxPad(body, flags, width)
}
function jxBadConv(lc, v) {
    jxkThrow("IllegalFormatConversionException", lc + " != " + jxkClass(v))
}
// An int's 32 bits as unsigned hex digits, Integer.toHexString.
function jxHex32(n) {
    var u = n < 0 ? n + 4294967296 : n
    var out = ""
    while (u >= 16) {
        out = "0123456789abcdef".charAt(u % 16) + out
        u = Math.floor(u / 16)
    }
    return "0123456789abcdef".charAt(u) + out
}
// Groups of three integer digits, Locale.US.
function jxGroup(ds) {
    var out = ""
    var n = ds.length
    var i = 0
    while (i < n) {
        if (i > 0 && (n - i) % 3 == 0) { out = out + "," }
        out = out + ds.charAt(i)
        i = i + 1
    }
    return out
}
// A number's sign and padding: '(' brackets a negative, '+' and ' ' mark a
// non-negative, '0' pads with zeros after the sign, '-' left-justifies.
function jxNum(neg, digits, flags, width, up) {
    var pre = ""
    var post = ""
    if (neg) {
        if (jxHas(flags, "(")) {
            pre = "("
            post = ")"
        } else {
            pre = "-"
        }
    } else if (jxHas(flags, "+")) {
        pre = "+"
    } else if (jxHas(flags, " ")) {
        pre = " "
    }
    var body = digits
    if (up) { body = unicodeCase(body, 3) }
    if (jxHas(flags, "0")) {
        while (pre.length + body.length + post.length < width) { body = "0" + body }
    }
    return jxPad(pre + body + post, flags, width)
}
// The digits ds rounded HALF_UP to keep digits - one longer when the carry
// runs out of the front (999 -> 1000), empty when keep < 0 or the value
// rounds away entirely.
function jxRoundUp(ds, keep) {
    if (keep < 0) { return "" }
    if (keep >= ds.length) { return ds }
    var head = ds.substring(0, keep)
    if (ds.charCodeAt(keep) < 53) { return head }
    var out = ""
    var carry = 1
    var i = keep - 1
    while (i >= 0) {
        var d = head.charCodeAt(i) - 48 + carry
        carry = 0
        if (d == 10) {
            d = 0
            carry = 1
        }
        out = "" + d + out
        i = i - 1
    }
    if (carry == 1) { out = "1" + out }
    return out
}
function jxZeros(n) {
    var out = ""
    var i = 0
    while (i < n) {
        out = out + "0"
        i = i + 1
    }
    return out
}
// %e / %f / %g of a double.
function jxFloat(lc, x, flags, width, precIn, up) {
    if (x !== x) { return jxPad(up ? "NAN" : "NaN", flags, width) }
    var neg = x < 0 || (x == 0 && 1 / x < 0)
    var a = neg ? 0 - x : x
    if (a - a !== 0) {
        var inf = up ? "INFINITY" : "Infinity"
        var nf = flags
        var z = nf.indexOf("0")
        if (z >= 0) { nf = nf.substring(0, z) + nf.substring(z + 1, nf.length) }
        return jxNum(neg, inf, nf, width, false)
    }
    var ds = "0"
    var e = 0
    if (a != 0) {
        var dg = jxDigitsOf(jxkDblStr(a))
        var ei = dg.indexOf("e")
        ds = dg.substring(0, ei)
        e = parseInt(dg.substring(ei + 1, dg.length), 10)
    }
    var prec = precIn
    var body = ""
    if (lc == "g") {
        if (prec < 0) { prec = 6 }
        if (prec == 0) { prec = 1 }
        var re = e
        if (a != 0) {
            var r = jxRoundUp(ds, prec)
            if (r.length > prec) { re = e + 1 }
        }
        if (a == 0 || (re >= 0 - 4 && re < prec)) {
            body = jxFixed(ds, e, a == 0 ? prec - 1 : prec - 1 - re, jxHas(flags, ","))
        } else {
            body = jxSci(ds, e, prec - 1, false)
        }
    } else if (lc == "f") {
        body = jxFixed(ds, e, prec < 0 ? 6 : prec, jxHas(flags, ","))
        if (prec == 0 && jxHas(flags, "#")) { body = body + "." }
    } else {
        body = jxSci(ds, e, prec < 0 ? 6 : prec, jxHas(flags, "#"))
    }
    return jxNum(neg, body, flags, width, up)
}
// %a of a double: Double.toHexString's 0x1.hhhp-e (0x0.hhhp-1022 for a
// subnormal), or, with a precision below 13, the significand rounded half-even
// to that many hex digits (at least one) after normalizing a subnormal. A
// float argument is widened first. The double is scaled by powers of two,
// which is exact, and its 52 fraction bits read off four at a time.
function jxHexFloat(x, flags, width, prec, up) {
    if (x !== x || x - x !== 0) { return jxFloat("e", x, flags, width, prec, up) }
    var neg = x < 0 || (x == 0 && 1 / x < 0)
    var a = neg ? 0 - x : x
    var lead = 1
    var e = 0
    var m = a
    var cut = prec >= 0 && prec < 13
    if (a == 0) {
        lead = 0
    } else if (a < Math.pow(2, 0 - 1022)) {
        m = a * Math.pow(2, 1022)
        e = 0 - 1022
        if (cut) {
            while (m < 1) {
                m = m * 2
                e = e - 1
            }
        } else {
            lead = 0
        }
    } else {
        while (m >= 2) {
            m = m / 2
            e = e + 1
        }
        while (m < 1) {
            m = m * 2
            e = e - 1
        }
    }
    var frac = m - lead
    var ds = []
    for (var i = 0; i < 13; i++) {
        frac = frac * 16
        var d = Math.floor(frac)
        ds.push(d)
        frac = frac - d
    }
    var n = 13
    if (cut) {
        n = prec == 0 ? 1 : prec
        var first = ds[n]
        var more = false
        for (var r = n + 1; r < 13; r++) { if (ds[r] != 0) { more = true } }
        if (first > 8 || (first == 8 && (more || ds[n - 1] % 2 == 1))) {
            var q = n - 1
            while (q >= 0 && ds[q] == 15) {
                ds[q] = 0
                q = q - 1
            }
            if (q >= 0) { ds[q] = ds[q] + 1 } else { lead = lead + 1 }
            if (lead == 2) {
                lead = 1
                e = e + 1
            }
        }
    } else if (prec < 0) {
        while (n > 1 && ds[n - 1] == 0) { n = n - 1 }
    }
    var hex = ""
    for (var h = 0; h < n; h++) { hex = hex + "0123456789abcdef".charAt(ds[h]) }
    while (hex.length < prec) { hex = hex + "0" }
    var body = "" + lead + "." + hex + "p" + e
    var sign = neg ? "-" : (jxHas(flags, "+") ? "+" : (jxHas(flags, " ") ? " " : ""))
    if (jxHas(flags, "0")) {
        while (sign.length + 2 + body.length < width) { body = "0" + body }
    }
    var out = sign + "0x" + body
    if (up) { out = unicodeCase(out, 3) }
    return jxPad(out, flags, width)
}
// Double.toString's digits ("4.9E-324", "123.45") as DIGITS "e" EXP, EXP the
// exponent of the first digit. They are the shortest round-tripping digits
// except that Java keeps two significant digits, the second read off the
// actual value - so Double.MIN_VALUE formats as 4.900000e-324.
function jxDigitsOf(t) {
    var mant = t
    var exp = 0
    var ei = t.indexOf("E")
    if (ei >= 0) {
        mant = t.substring(0, ei)
        exp = parseInt(t.substring(ei + 1, t.length), 10)
    }
    var ds = ""
    var pt = 0 - 1
    var i = 0
    while (i < mant.length) {
        var c = mant.charAt(i)
        if (c == ".") { pt = ds.length } else { ds = ds + c }
        i = i + 1
    }
    if (pt < 0) { pt = ds.length }
    var lead = 0
    while (lead < ds.length - 1 && ds.charAt(lead) == "0") { lead = lead + 1 }
    ds = ds.substring(lead, ds.length)
    var end = ds.length
    while (end > 1 && ds.charAt(end - 1) == "0") { end = end - 1 }
    return ds.substring(0, end) + "e" + (pt - lead + exp - 1)
}
// The digits ds (value 0.ds * 10^(e+1)) with prec fraction digits.
function jxFixed(ds, e, prec, group) {
    var pt = e + 1
    var r = ds == "0" ? "" : jxRoundUp(ds, pt + prec)
    if (r.length > pt + prec && pt + prec >= 0) { pt = pt + 1 }
    var ints = "0"
    var frac = ""
    if (pt <= 0) {
        frac = jxZeros(0 - pt) + r
    } else {
        while (r.length < pt) { r = r + "0" }
        ints = r.substring(0, pt)
        frac = r.substring(pt, r.length)
    }
    while (frac.length < prec) { frac = frac + "0" }
    if (frac.length > prec) { frac = frac.substring(0, prec) }
    if (group) { ints = jxGroup(ints) }
    if (prec == 0) { return ints }
    return ints + "." + frac
}
// d.ddd e+XX with prec fraction digits and at least two exponent digits.
function jxSci(ds, e, prec, point) {
    var ex = e
    var r = ds
    if (ds != "0") {
        r = jxRoundUp(ds, prec + 1)
        if (r.length > prec + 1) {
            ex = ex + 1
            r = r.substring(0, prec + 1)
        }
    } else {
        ex = 0
    }
    while (r.length < prec + 1) { r = r + "0" }
    var m = r.substring(0, 1)
    if (prec > 0 || point) { m = m + "." + r.substring(1, r.length) }
    var es = ex < 0 ? "" + (0 - ex) : "" + ex
    if (es.length < 2) { es = "0" + es }
    return m + "e" + (ex < 0 ? "-" : "+") + es
}
