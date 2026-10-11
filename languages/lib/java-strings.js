// java-strings.js -- java.lang.String's methods and String.format over plain
// strings, for java-interpreter.abnf (include) and languages/lib/java-rt.metajs
// (import), both of which also include lib/regex.js. abnf/jsrtjavastr.go is the
// Go twin's port; the three must answer alike.
//
// Every function here takes and answers plain strings, numbers and arrays of
// strings. What depends on the value model is asked through hooks each importer
// defines:
//   jxkKind(v)       "null" "bool" "char" "int" "float" "str" or "obj"
//   jxkStr(v)        String.valueOf(v)
//   jxkCode(v)       a char's code, or an int's value
//   jxkDec(v)        an integral value's signed decimal digits
//   jxkRadix(v, r)   its two's complement digits in radix r at its own width
//   jxkDbl(v)        a float's value as a double
//   jxkDblStr(a)     Double.toString(a) of a double a > 0
//   jxkFail(msg)     a format or a pattern this subset does not support
//
// Case mapping and the literal search are lib/utf16-strings.js's, which both
// importers include first.

// Character.isWhitespace.
function jxIsWs(c) {
    if ((c >= 9 && c <= 13) || (c >= 28 && c <= 32)) { return true }
    if (c == 5760 || (c >= 8192 && c <= 8198) || (c >= 8200 && c <= 8202)) { return true }
    return c == 8232 || c == 8233 || c == 8287 || c == 12288
}
// trim() drops every unit <= ' ' at both ends; strip() drops whitespace.
function jxTrim(s) {
    var b = 0
    var e = s.length
    while (b < e && s.charCodeAt(b) <= 32) { b = b + 1 }
    while (e > b && s.charCodeAt(e - 1) <= 32) { e = e - 1 }
    return s.substring(b, e)
}
function jxStrip(s, lead, trail) {
    var b = 0
    var e = s.length
    if (lead) { while (b < e && jxIsWs(s.charCodeAt(b))) { b = b + 1 } }
    if (trail) { while (e > b && jxIsWs(s.charCodeAt(e - 1))) { e = e - 1 } }
    return s.substring(b, e)
}
function jxIsBlank(s) { return jxStrip(s, true, false).length == 0 }
// compareTo: the first differing unit's difference, else the length difference.
function jxCompare(a, b) {
    var n = a.length
    if (b.length < n) { n = b.length }
    var i = 0
    while (i < n) {
        var d = a.charCodeAt(i) - b.charCodeAt(i)
        if (d != 0) { return d }
        i = i + 1
    }
    return a.length - b.length
}
// compareToIgnoreCase / equalsIgnoreCase fold each unit to upper case and then
// to lower case, as java.lang.String does.
function jxFold(c) { return jxChLo(jxChUp(c)) }
function jxCompareIC(a, b) {
    var n = a.length
    if (b.length < n) { n = b.length }
    var i = 0
    while (i < n) {
        var x = a.charCodeAt(i)
        var y = b.charCodeAt(i)
        if (x != y) {
            x = jxFold(x)
            y = jxFold(y)
            if (x != y) { return x - y }
        }
        i = i + 1
    }
    return a.length - b.length
}
// A code point as its one or two UTF-16 units.
function jxCpStr(cp) {
    if (cp < 65536) { return String.fromCharCode(cp) }
    var v = cp - 65536
    return String.fromCharCode(55296 + Math.floor(v / 1024)) + String.fromCharCode(56320 + v % 1024)
}
function jxJoin(delim, parts) {
    var out = ""
    var i = 0
    while (i < parts.length) {
        if (i > 0) { out = out + delim }
        out = out + parts[i]
        i = i + 1
    }
    return out
}

// ----- the regex methods (java.util.regex.Pattern / Matcher) -----

// Java's pattern flags in regex.js's terms: \Q...\E quoting.
function jxRe(pattern) {
    var re = rxGet(pattern, "q")
    if (!re.ok) { jxkFail("PatternSyntaxException: " + re.err) }
    return re
}
// Matcher.find's next start: after an empty match the search moves one on.
function jxNext(m) {
    if (m.end == m.begin) { return m.end + 1 }
    return m.end
}
// Pattern.split(input, limit): a positive limit caps the pieces, zero drops the
// trailing empty ones, and an empty match at 0 makes no leading empty piece.
function jxSplit(s, pattern, limit) {
    var re = jxRe(pattern)
    var out = []
    var index = 0
    var at = 0
    while (at <= s.length) {
        var m = rxSearch(re, s, at)
        if (m == null) { break }
        if (limit <= 0 || out.length < limit - 1) {
            if (!(index == 0 && m.begin == 0 && m.end == 0)) {
                out.push(s.substring(index, m.begin))
                index = m.end
            }
        } else {
            out.push(s.substring(index, s.length))
            index = m.end
            break
        }
        at = jxNext(m)
    }
    if (index == 0) { return [s] }
    if (limit <= 0 || out.length < limit) { out.push(s.substring(index, s.length)) }
    if (limit == 0) {
        var n = out.length
        while (n > 0 && out[n - 1].length == 0) { n = n - 1 }
        var cut = []
        var i = 0
        while (i < n) {
            cut.push(out[i])
            i = i + 1
        }
        return cut
    }
    return out
}
function jxMatches(s, pattern) { return rxMatchWhole(jxRe(pattern), s, 0) != null }
// replaceAll / replaceFirst.
function jxRxReplace(s, pattern, repl, all) {
    var re = jxRe(pattern)
    var out = ""
    var tail = 0
    var at = 0
    while (at <= s.length) {
        var m = rxSearch(re, s, at)
        if (m == null) { break }
        out = out + s.substring(tail, m.begin) + jxExpand(re, s, m, repl)
        tail = m.end
        if (!all) { break }
        at = jxNext(m)
    }
    return out + s.substring(tail, s.length)
}
// Matcher.appendReplacement's template: a backslash takes the next character
// literally, $n is group n - as many digits as still name a group - and
// ${name} a named group. An unmatched group contributes nothing.
function jxExpand(re, s, m, repl) {
    var out = ""
    var i = 0
    while (i < repl.length) {
        var c = repl.charAt(i)
        if (c == "\\") {
            if (i + 1 >= repl.length) { jxkFail("IllegalArgumentException: character to be escaped is missing") }
            out = out + repl.charAt(i + 1)
            i = i + 2
            continue
        }
        if (c != "$") {
            out = out + c
            i = i + 1
            continue
        }
        i = i + 1
        if (i >= repl.length) { jxkFail("IllegalArgumentException: Illegal group reference: group index is missing") }
        var g = 0 - 1
        if (repl.charAt(i) == "{") {
            var e = i + 1
            while (e < repl.length && repl.charAt(e) != "}") { e = e + 1 }
            var nm = repl.substring(i + 1, e)
            g = rxIndexOfName(re, nm)
            if (g < 0) { jxkFail("IllegalArgumentException: No group with name {" + nm + "}") }
            i = e + 1
        } else {
            var d = repl.charCodeAt(i) - 48
            if (d < 0 || d > 9) { jxkFail("IllegalArgumentException: Illegal group reference") }
            g = d
            i = i + 1
            if (g > rxGroupCount(re)) { jxkFail("IndexOutOfBoundsException: No group " + g) }
            while (i < repl.length) {
                var d2 = repl.charCodeAt(i) - 48
                if (d2 < 0 || d2 > 9) { break }
                var ng = g * 10 + d2
                if (ng > rxGroupCount(re)) { break }
                g = ng
                i = i + 1
            }
        }
        var gs = rxGroup(re, s, m, g)
        if (gs != null) { out = out + gs }
    }
    return out
}

// ----- String.format (java.util.Formatter, Locale.US) -----
//
// %[index$][flags][width][.precision]conversion with the conversions
// b s c d o x e f g % n (and their upper-case forms). A double is formatted
// from its Double.toString digits rounded HALF_UP, which is Java's rule -
// %.2f of 1.005 is 1.01 and %.1f of 0.15 is 0.2 - and a float is widened to a
// double first. Missing arguments, mismatched conversions and the rarer
// conversions (h, a, t) are reported through jxkFail.
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
            j = k + 1
        }
        var flags = ""
        while (j < fmt.length && "-#+ 0,(<".indexOf(fmt.charAt(j)) >= 0) {
            flags = flags + fmt.charAt(j)
            j = j + 1
        }
        var width = 0 - 1
        k = j
        while (k < fmt.length && jxDigit(fmt.charCodeAt(k))) { k = k + 1 }
        if (k > j) { width = parseInt(fmt.substring(j, k), 10) }
        j = k
        var prec = 0 - 1
        if (j < fmt.length && fmt.charAt(j) == ".") {
            k = j + 1
            while (k < fmt.length && jxDigit(fmt.charCodeAt(k))) { k = k + 1 }
            if (k == j + 1) { jxkFail("IllegalFormatPrecisionException: " + fmt.substring(start, k)) }
            prec = parseInt(fmt.substring(j + 1, k), 10)
            j = k
        }
        if (j >= fmt.length) { jxkFail("UnknownFormatConversionException: Conversion = '%'") }
        var conv = fmt.charAt(j)
        i = j + 1
        var spec = fmt.substring(start, i)
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
        if (ai < 0 || ai >= args.length) { jxkFail("MissingFormatArgumentException: Format specifier '" + spec + "'") }
        last = ai
        out = out + jxConv(conv, flags, width, prec, args[ai], spec)
    }
    return out
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
function jxConv(conv, flags, width, prec, v, spec) {
    var lc = conv.toLowerCase()
    var up = conv != lc
    var k = jxkKind(v)
    var body = ""
    if (lc == "s") {
        body = k == "null" ? "null" : jxkStr(v)
        if (prec >= 0 && prec < body.length) { body = body.substring(0, prec) }
    } else if (lc == "b") {
        if (k == "null") { body = "false" } else if (k == "bool") { body = jxkStr(v) } else { body = "true" }
        if (prec >= 0 && prec < body.length) { body = body.substring(0, prec) }
    } else if (lc == "c") {
        if (k == "null") { body = "null" }
        else if (k == "char" || k == "int") { body = jxCpStr(jxkCode(v)) }
        else { jxBadConv(conv, k) }
    } else if (lc == "d") {
        if (k == "null") { body = "null" } else {
            if (k != "int") { jxBadConv(conv, k) }
            var dd = jxkDec(v)
            var neg = dd.charAt(0) == "-"
            if (neg) { dd = dd.substring(1) }
            if (jxHas(flags, ",")) { dd = jxGroup(dd) }
            return jxNum(neg, dd, flags, width, up)
        }
    } else if (lc == "x" || lc == "o") {
        if (k == "null") { body = "null" } else {
            if (k != "int") { jxBadConv(conv, k) }
            var rd = jxkRadix(v, lc == "x" ? 16 : 8)
            if (jxHas(flags, "#")) { rd = (lc == "x" ? "0x" : "0") + rd }
            if (jxHas(flags, "0")) {
                while (rd.length < width) { rd = (lc == "x" && jxHas(flags, "#")) ? "0x0" + rd.substring(2) : "0" + rd }
            }
            body = rd
        }
    } else if (lc == "f" || lc == "e" || lc == "g") {
        if (k == "null") { body = "null" } else {
            if (k != "float") { jxBadConv(conv, k) }
            return jxFloat(lc, jxkDbl(v), flags, width, prec, up)
        }
    } else {
        jxkFail("UnknownFormatConversionException: Conversion = '" + conv + "'")
    }
    if (up) { body = unicodeCase(body, 3) }
    return jxPad(body, flags, width)
}
function jxBadConv(conv, k) {
    jxkFail("IllegalFormatConversionException: " + conv + " != " + k)
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
        if (jxHas(nf, "0")) { nf = jxReplace(nf, "0", "") }
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
    } else {
        body = jxSci(ds, e, prec < 0 ? 6 : prec, jxHas(flags, "#"))
    }
    return jxNum(neg, body, flags, width, up)
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
