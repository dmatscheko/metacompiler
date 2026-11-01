// python-codecs.js -- str.encode and bytes.decode for CPython 3.14's utf-8,
// ascii, latin-1, utf-16, utf-32 (each with -le and -be) and cp1252 codecs and
// the strict, ignore, replace, backslashreplace, xmlcharrefreplace and
// surrogateescape error handlers, for python-interpreter.abnf (include) and
// languages/lib/python-rt.metajs (import). Go twin: abnf/jsrtpycodecs.go.
//
// Neither direction raises: each answers {err, msg, b, s} - err the exception
// class name ("" on success), b the encoded byte values, s the decoded text -
// and the caller raises in its own engine.

// The codec an encoding name selects: lower-cased, every run of characters
// other than ASCII letters, digits and "." collapsed to one "_" (CPython's
// normalize_encoding), then looked up among the codecs' names and aliases
// (encodings.aliases). "" for a codec not implemented here.
function pycdLookup(enc) {
    var n = ""
    var punct = false
    var low = enc.toLowerCase()
    for (var i = 0; i < low.length; i++) {
        var c = low.charCodeAt(i)
        if ((c >= 97 && c <= 122) || (c >= 48 && c <= 57) || c == 46) {
            if (punct && n != "") { n = n + "_" }
            n = n + String.fromCharCode(c)
            punct = false
        } else {
            punct = true
        }
    }
    var d = ""
    for (var j = 0; j < n.length; j++) {
        if (n.charAt(j) == ".") { d = d + "_" } else { d = d + n.charAt(j) }
    }
    var names = pycdNames
    for (var k = 0; k < names.length; k = k + 2) {
        var al = names[k + 1]
        for (var m = 0; m < al.length; m++) {
            if (al[m] == n || al[m] == d) { return names[k] }
        }
    }
    return ""
}
var pycdNames = [
    "utf-8", ["utf_8", "u8", "utf", "utf8", "utf8_ucs2", "utf8_ucs4", "cp65001"],
    "ascii", ["ascii", "646", "ansi_x3_4_1968", "ansi_x3_4_1986", "cp367", "csascii", "ibm367", "iso646_us",
        "iso_646_irv_1991", "iso_ir_6", "us", "us_ascii"],
    "latin-1", ["latin_1", "8859", "cp819", "csisolatin1", "ibm819", "iso8859", "iso8859_1", "iso_8859_1",
        "iso_8859_1_1987", "iso_ir_100", "l1", "latin", "latin1"],
    "utf-16", ["utf_16", "u16", "utf16"],
    "utf-16-le", ["utf_16_le", "utf_16le", "unicodelittleunmarked"],
    "utf-16-be", ["utf_16_be", "utf_16be", "unicodebigunmarked"],
    "utf-32", ["utf_32", "u32", "utf32"],
    "utf-32-le", ["utf_32_le", "utf_32le"],
    "utf-32-be", ["utf_32_be", "utf_32be"],
    "cp1252", ["cp1252", "1252", "windows_1252"]]

// cp1252's bytes 0x80..0x9f; 0 where the byte is undefined.
var pycdCp1252 = [8364, 0, 8218, 402, 8222, 8230, 8224, 8225, 710, 8240, 352, 8249, 338, 0, 381, 0,
    0, 8216, 8217, 8220, 8221, 8226, 8211, 8212, 732, 8482, 353, 8250, 339, 0, 382, 376]

// The byte cp encodes to in cp1252, or -1.
function pycdCp1252Byte(cp) {
    if (cp < 128 || (cp >= 160 && cp < 256)) { return cp }
    for (var i = 0; i < 32; i++) {
        if (pycdCp1252[i] == cp && cp != 0) { return 128 + i }
    }
    return 0 - 1
}

// A code point as repr spells it inside a string: \xhh, \uhhhh or \Uhhhhhhhh.
function pycdEsc(cp) {
    var w = 8
    var head = "\\U"
    if (cp < 256) {
        w = 2
        head = "\\x"
    } else if (cp < 65536) {
        w = 4
        head = "\\u"
    }
    return head + pycdHex(cp, w)
}
function pycdHex(v0, w) {
    var d = "0123456789abcdef"
    var hex = ""
    var v = v0
    for (var i = 0; i < w; i++) {
        hex = d.charAt(v % 16) + hex
        v = Math.floor(v / 16)
    }
    return hex
}

// One code unit (utf-16) or code point (utf-32) as bytes, in the byte order le.
function pycdPushUnit(out, u, size, le) {
    var bs = []
    var v = u
    for (var i = 0; i < size; i++) {
        bs.push(v % 256)
        v = Math.floor(v / 256)
    }
    for (var j = 0; j < size; j++) { out.push(le ? bs[j] : bs[size - 1 - j]) }
}

// Encode the code points cps. A UTF-16/32 codec meets each unencodable
// character on its own; the others hand a whole run of them to the handler.
// surrogateescape takes U+DC80..U+DCFF to the byte they escape and refuses
// anything else - from the first refused character on for utf-8, ascii and
// latin-1, whose encoders escape a prefix of the run themselves.
function pycdEncode(cps, codec, errs) {
    var r = {err: "", msg: "", b: [], s: ""}
    var out = r.b
    var size = 0
    var le = true
    if (codec == "utf-16" || codec == "utf-16-le" || codec == "utf-16-be") { size = 2 }
    if (codec == "utf-32" || codec == "utf-32-le" || codec == "utf-32-be") { size = 4 }
    if (codec == "utf-16-be" || codec == "utf-32-be") { le = false }
    if (codec == "utf-16" || codec == "utf-32") { pycdPushUnit(out, 65279, size, true) }
    var why = "surrogates not allowed"
    if (codec == "ascii") { why = "ordinal not in range(128)" }
    if (codec == "latin-1") { why = "ordinal not in range(256)" }
    if (codec == "cp1252") { why = "character maps to <undefined>" }
    var errCodec = codec
    if (codec == "cp1252") { errCodec = "charmap" }
    var k = 0
    while (k < cps.length) {
        var cp = cps[k]
        if (!pycdEncBad(codec, cp)) {
            pycdEncOne(out, codec, cp, size, le)
            k = k + 1
            continue
        }
        var j = k + 1
        if (size == 0) {
            while (j < cps.length && pycdEncBad(codec, cps[j])) { j = j + 1 }
        }
        var bad = errs == "strict" || (errs == "surrogateescape" && size > 0)
        if (errs == "surrogateescape" && size == 0) {
            var q = k
            while (q < j && cps[q] >= 56448 && cps[q] <= 56575) { q = q + 1 }
            if (q < j) {
                if (codec != "cp1252") { k = q }
                bad = true
            }
        }
        if (bad) {
            var where = "character '" + pycdEsc(cps[k]) + "' in position " + k
            if (j - k > 1) { where = "characters in position " + k + "-" + (j - 1) }
            r.err = "UnicodeEncodeError"
            r.msg = "'" + errCodec + "' codec can't encode " + where + ": " + why
            return r
        }
        for (var p = k; p < j; p++) {
            var rep = ""
            if (errs == "replace") { rep = "?" }
            else if (errs == "backslashreplace") { rep = pycdEsc(cps[p]) }
            else if (errs == "xmlcharrefreplace") { rep = "&#" + cps[p] + ";" }
            else if (errs == "surrogateescape") { out.push(cps[p] - 56320) }
            else if (errs != "ignore") {
                r.err = "LookupError"
                r.msg = "unknown error handler name '" + errs + "'"
                return r
            }
            for (var t = 0; t < rep.length; t++) { pycdEncOne(out, codec, rep.charCodeAt(t), size, le) }
        }
        k = j
    }
    return r
}

// A code point the codec cannot encode.
function pycdEncBad(codec, cp) {
    if (codec == "ascii") { return cp >= 128 }
    if (codec == "latin-1") { return cp >= 256 }
    if (codec == "cp1252") { return pycdCp1252Byte(cp) < 0 }
    return cp >= 55296 && cp <= 57343
}

function pycdEncOne(out, codec, cp, size, le) {
    if (size == 4) {
        pycdPushUnit(out, cp, 4, le)
    } else if (size == 2) {
        if (cp >= 65536) {
            pycdPushUnit(out, 55296 + Math.floor((cp - 65536) / 1024), 2, le)
            pycdPushUnit(out, 56320 + (cp - 65536) % 1024, 2, le)
        } else {
            pycdPushUnit(out, cp, 2, le)
        }
    } else if (codec == "cp1252") {
        out.push(pycdCp1252Byte(cp))
    } else if (codec != "utf-8" || cp < 128) {
        out.push(cp)
    } else if (cp < 2048) {
        out.push(192 + Math.floor(cp / 64))
        out.push(128 + cp % 64)
    } else if (cp < 65536) {
        out.push(224 + Math.floor(cp / 4096))
        out.push(128 + Math.floor(cp / 64) % 64)
        out.push(128 + cp % 64)
    } else {
        out.push(240 + Math.floor(cp / 262144))
        out.push(128 + Math.floor(cp / 4096) % 64)
        out.push(128 + Math.floor(cp / 64) % 64)
        out.push(128 + cp % 64)
    }
}

function pycdChr(cp) {
    if (cp < 65536) { return String.fromCharCode(cp) }
    var v = cp - 65536
    return String.fromCharCode(55296 + Math.floor(v / 1024)) + String.fromCharCode(56320 + v % 1024)
}

// Decode the byte values bs. Each undecodable range [from, to) - CPython's
// UnicodeDecodeError start and end - goes to pycdBad.
function pycdDecode(bs, codec, errs) {
    var r = {err: "", msg: "", b: [], s: ""}
    var n = bs.length
    var i = 0
    if (codec == "latin-1") {
        while (i < n) {
            r.s = r.s + String.fromCharCode(bs[i])
            i = i + 1
        }
        return r
    }
    if (codec == "ascii" || codec == "cp1252") {
        while (i < n) {
            var c0 = bs[i]
            if (c0 >= 128 && c0 < 160 && codec == "cp1252") { c0 = pycdCp1252[c0 - 128] }
            if (codec == "ascii" && c0 >= 128) {
                pycdBad(r, "ascii", bs, i, i + 1, "ordinal not in range(128)", errs)
            } else if (c0 == 0 && bs[i] != 0) {
                pycdBad(r, "charmap", bs, i, i + 1, "character maps to <undefined>", errs)
            } else {
                r.s = r.s + String.fromCharCode(c0)
            }
            if (r.err != "") { return r }
            i = i + 1
        }
        return r
    }
    if (codec == "utf-8") {
        pycdDecodeUtf8(r, bs, errs)
        return r
    }
    var size = 2
    if (codec == "utf-32" || codec == "utf-32-le" || codec == "utf-32-be") { size = 4 }
    var le = codec != "utf-16-be" && codec != "utf-32-be"
    if ((codec == "utf-16" && n >= 2) || (codec == "utf-32" && n >= 4)) {
        var bom = pycdUnit(bs, 0, size, true)
        if (bom == 65279) { i = size }
        if ((size == 2 && bom == 65534) || (size == 4 && bom == 4294836224)) {
            i = size
            le = false
        }
    }
    var name = "utf-16-be"
    if (size == 2 && le) { name = "utf-16-le" }
    if (size == 4) { name = le ? "utf-32-le" : "utf-32-be" }
    while (i < n) {
        if (n - i < size) {
            i = pycdBad(r, name, bs, i, n, "truncated data", errs)
        } else if (size == 4) {
            var u = pycdUnit(bs, i, 4, le)
            if (u >= 55296 && u <= 57343) {
                i = pycdBad(r, name, bs, i, i + 4, "code point in surrogate code point range(0xd800, 0xe000)", errs)
            } else if (u >= 1114112) {
                i = pycdBad(r, name, bs, i, i + 4, "code point not in range(0x110000)", errs)
            } else {
                r.s = r.s + pycdChr(u)
                i = i + 4
            }
        } else {
            var w = pycdUnit(bs, i, 2, le)
            if (w < 55296 || w > 57343) {
                r.s = r.s + String.fromCharCode(w)
                i = i + 2
            } else if (w >= 56320) {
                i = pycdBad(r, name, bs, i, i + 2, "illegal encoding", errs)
            } else if (n - i < 4) {
                i = pycdBad(r, name, bs, i, n, "unexpected end of data", errs)
            } else {
                var w2 = pycdUnit(bs, i + 2, 2, le)
                if (w2 >= 56320 && w2 <= 57343) {
                    r.s = r.s + String.fromCharCode(w) + String.fromCharCode(w2)
                    i = i + 4
                } else {
                    i = pycdBad(r, name, bs, i, i + 2, "illegal UTF-16 surrogate", errs)
                }
            }
        }
        if (r.err != "") { return r }
    }
    return r
}

// The size-byte unit at bs[i], in the byte order le.
function pycdUnit(bs, i, size, le) {
    var v = 0
    for (var j = 0; j < size; j++) {
        var b = le ? bs[i + size - 1 - j] : bs[i + j]
        v = v * 256 + b
    }
    return v
}

function pycdDecodeUtf8(r, bs, errs) {
    var n = bs.length
    var i = 0
    while (i < n) {
        var b0 = bs[i]
        if (b0 < 128) {
            r.s = r.s + String.fromCharCode(b0)
            i = i + 1
            continue
        }
        var need = 0
        var cp = 0
        if (b0 >= 194 && b0 <= 223) {
            need = 1
            cp = b0 % 32
        } else if (b0 >= 224 && b0 <= 239) {
            need = 2
            cp = b0 % 16
        } else if (b0 >= 240 && b0 <= 244) {
            need = 3
            cp = b0 % 8
        }
        var why = ""
        if (need == 0) { why = "invalid start byte" }
        var to = i + 1
        var k = 1
        while (why == "" && k <= need) {
            if (i + k >= n) {
                why = "unexpected end of data"
                to = n
            } else {
                var c = bs[i + k]
                var lo = 128
                var hi = 191
                if (k == 1 && b0 == 224) { lo = 160 }
                if (k == 1 && b0 == 237) { hi = 159 }
                if (k == 1 && b0 == 240) { lo = 144 }
                if (k == 1 && b0 == 244) { hi = 143 }
                if (c < lo || c > hi) {
                    why = "invalid continuation byte"
                    to = i + k
                } else {
                    cp = cp * 64 + c % 64
                    k = k + 1
                }
            }
        }
        if (why != "") {
            i = pycdBad(r, "utf-8", bs, i, to, why, errs)
            if (r.err != "") { return }
            continue
        }
        r.s = r.s + pycdChr(cp)
        i = i + need + 1
    }
}

// The error handler errs on the undecodable bytes [from, to): its text is
// appended to r.s, or r.err set; the answer is where decoding resumes.
// surrogateescape gives each of up to four leading bytes b >= 128 as U+DC00+b
// and resumes after them, refusing a range that starts with an ASCII byte. A
// handler name is only looked up once it is needed, as CPython does.
function pycdBad(r, codec, bs, from, to, why, errs) {
    var esc = 0
    if (errs == "surrogateescape") {
        while (esc < 4 && from + esc < to && bs[from + esc] >= 128) {
            r.s = r.s + String.fromCharCode(56320 + bs[from + esc])
            esc = esc + 1
        }
        if (esc > 0) { return from + esc }
    }
    if (errs == "strict" || errs == "surrogateescape") {
        var where = "byte 0x" + pycdHex(bs[from], 2) + " in position " + from
        if (to - from > 1) { where = "bytes in position " + from + "-" + (to - 1) }
        r.err = "UnicodeDecodeError"
        r.msg = "'" + codec + "' codec can't decode " + where + ": " + why
    } else if (errs == "replace") {
        r.s = r.s + String.fromCharCode(65533)
    } else if (errs == "backslashreplace") {
        for (var i = from; i < to; i++) { r.s = r.s + "\\x" + pycdHex(bs[i], 2) }
    } else if (errs == "xmlcharrefreplace" || errs == "namereplace") {
        r.err = "TypeError"
        r.msg = "don't know how to handle UnicodeDecodeError in error callback"
    } else if (errs != "ignore") {
        r.err = "LookupError"
        r.msg = "unknown error handler name '" + errs + "'"
    }
    return to
}
