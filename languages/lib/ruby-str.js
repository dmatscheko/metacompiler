// ruby-str.js -- String algorithms of MRI's string.c that take a String (and a
// compiled regex.js pattern) and answer plain values, for ruby-interpreter.abnf
// (include) and languages/lib/ruby-rt.metajs (import). Positions are UTF-16 code
// units; a surrogate pair is kept whole. Go twins in abnf/jsrtruby.go.
//
// Written in the subset both engines accept: no split/join/lastIndexOf, and
// indexOf without a from-index.

function rbsIsSpace(c) { return c == 32 || (c >= 9 && c <= 13) }

// The length of the character at unit i: 2 for a surrogate pair, else 1.
function rbsCharLen(s, i) {
    var c = s.charCodeAt(i)
    if (c < 55296 || c > 56319 || i + 1 >= s.length) { return 1 }
    var d = s.charCodeAt(i + 1)
    return d >= 56320 && d <= 57343 ? 2 : 1
}

// The first unit index >= from where t occurs in s, or -1.
function rbsFind(s, t, from) {
    if (from > s.length) { return -1 }
    var k = s.substring(from, s.length).indexOf(t)
    return k < 0 ? -1 : from + k
}

// String#split (rb_str_split_m). sep is the separator String, or null for awk
// mode (no argument, nil or " "): runs of ASCII whitespace separate, leading
// whitespace is skipped. re, when not null, is a compiled pattern and wins over
// sep; its groups are spliced into the answer. lim is the limit (0 when absent):
// positive caps the field count, the last field holding the rest; zero drops
// trailing empty fields; negative keeps them.
function rbsSplit(s, sep, re, lim) {
    var out = []
    if (lim == 1) {
        if (s.length > 0) { out.push(s) }
        return out
    }
    var i = 1
    var beg = 0
    var p = 0
    if (re !== null) {
        var start = 0
        var lastNull = false
        while (true) {
            var m = rxSearch(re, s, start)
            if (m === null) { break }
            if (start == m.begin && m.begin == m.end) {
                if (s.length == 0) {
                    out.push("")
                    break
                } else if (lastNull) {
                    out.push(s.substring(beg, beg + rbsCharLen(s, beg)))
                    beg = start
                } else {
                    start = start + (start < s.length ? rbsCharLen(s, start) : 1)
                    lastNull = true
                    continue
                }
            } else {
                out.push(s.substring(beg, m.begin))
                beg = m.end
                start = m.end
            }
            lastNull = false
            var caps = m.caps
            var g = 1
            while (g <= re.ngroups) {
                if (caps[2 * g] >= 0 && caps[2 * g + 1] >= 0) { out.push(s.substring(caps[2 * g], caps[2 * g + 1])) }
                g = g + 1
            }
            i = i + 1
            if (lim > 0 && lim <= i) { break }
        }
    } else if (sep === null) {
        var skip = true
        var end = 0
        while (p < s.length) {
            var c = s.charCodeAt(p)
            p = p + 1
            if (skip) {
                if (rbsIsSpace(c)) {
                    beg = p
                } else {
                    end = p
                    skip = false
                    if (lim > 0 && lim <= i) { break }
                }
            } else if (rbsIsSpace(c)) {
                out.push(s.substring(beg, end))
                skip = true
                beg = p
                if (lim > 0) { i = i + 1 }
            } else {
                end = p
            }
        }
    } else if (sep.length == 0) {
        while (p < s.length) {
            var n = rbsCharLen(s, p)
            out.push(s.substring(p, p + n))
            p = p + n
            i = i + 1
            if (lim > 0 && lim <= i) { break }
        }
        beg = p
    } else {
        while (p < s.length) {
            var at = rbsFind(s, sep, p)
            if (at < 0) { break }
            out.push(s.substring(p, at))
            p = at + sep.length
            i = i + 1
            if (lim > 0 && lim <= i) { break }
        }
        beg = p
    }
    if (s.length > 0 && (lim > 0 || s.length > beg || lim < 0)) { out.push(s.substring(beg, s.length)) }
    if (lim == 0) {
        while (out.length > 0 && out[out.length - 1] == "") { out.pop() }
    }
    return out
}

// String#lines / #each_line: the pieces of s, each ending after an occurrence
// of sep (kept), the rest last.
function rbsLines(s, sep) {
    var out = []
    var p = 0
    while (p < s.length && sep.length > 0) {
        var at = rbsFind(s, sep, p)
        if (at < 0) { break }
        out.push(s.substring(p, at + sep.length))
        p = at + sep.length
    }
    if (p < s.length) { out.push(s.substring(p, s.length)) }
    return out
}

// Integer#chr("UTF-8"): the character of code point cp, or "" with the
// RangeError's message in err[0]. Go twin: rubyChrUtf8 in abnf/jsrtrubystr.go.
function rbsChrUtf8(cp, err) {
    if (cp < 0 || cp > 1114111) {
        err.push(cp + " out of char range")
        return ""
    }
    if (cp >= 55296 && cp <= 57343) {
        var hx = ""
        var v = cp
        while (v > 0) {
            hx = "0123456789ABCDEF".charAt(v % 16) + hx
            v = Math.floor(v / 16)
        }
        err.push("invalid codepoint 0x" + hx + " in UTF-8")
        return ""
    }
    if (cp < 65536) { return String.fromCharCode(cp) }
    var w = cp - 65536
    return String.fromCharCode(55296 + Math.floor(w / 1024), 56320 + (w % 1024))
}

// String#ord: the code point of the first character, a surrogate pair whole.
function rbsOrd(s) {
    if (s.length == 0) { return 0 }
    var c = s.charCodeAt(0)
    if (rbsCharLen(s, 0) == 2) { return (c - 55296) * 1024 + (s.charCodeAt(1) - 56320) + 65536 }
    return c
}

// The character index of code unit u, and the unit index of character c (-1 when
// the string is shorter).
function rbsCharIdx(s, u) {
    var i = 0
    var n = 0
    while (i < u) {
        i = i + rbsCharLen(s, i)
        n = n + 1
    }
    return n
}
function rbsUnitIdx(s, c) {
    var i = 0
    var n = 0
    while (n < c && i < s.length) {
        i = i + rbsCharLen(s, i)
        n = n + 1
    }
    return n < c ? -1 : i
}

// The character count of s. byteLen equal to the unit count means ASCII, where
// a unit is a character, so the common case does not scan.
function rbsLen(s) {
    if (byteLen(s) == s.length) { return s.length }
    return rbsCharIdx(s, s.length)
}

// Characters [c, c + k) of s, cut at the end (0 <= c <= rbsLen(s), k >= 0).
function rbsSub(s, c, k) {
    var a = c
    var b = c + k
    if (byteLen(s) != s.length) {
        a = rbsUnitIdx(s, c)
        b = a
        var n = 0
        while (n < k && b < s.length) {
            b = b + rbsCharLen(s, b)
            n = n + 1
        }
    }
    if (b > s.length) { b = s.length }
    return s.substring(a, b)
}

// String#[start, len] (start and len integers): nil for a negative len or a start
// outside the string, a negative start counting from the end.
function rbsSubseq(s, start, len) {
    var n = rbsLen(s)
    if (len < 0) { return null }
    if (start < 0) { start = start + n }
    if (start < 0 || start > n) { return null }
    return rbsSub(s, start, len)
}

// String#[i] (an integer): one character, or nil outside the string.
function rbsCharAt(s, i) {
    var n = rbsLen(s)
    if (i < 0) { i = i + n }
    if (i < 0 || i >= n) { return null }
    return rbsSub(s, i, 1)
}

// String#reverse, a surrogate pair kept whole.
function rbsReverse(s) {
    var out = ""
    var i = 0
    while (i < s.length) {
        var k = rbsCharLen(s, i)
        out = s.substring(i, i + k) + out
        i = i + k
    }
    return out
}

// String#center / #ljust / #rjust. mode 0 centres, 1 pads on the right, 2 on the
// left. MRI counts CHARACTERS, and a width at or below the receiver's length
// answers the receiver.
function rbsPad(s, width, pad, mode) {
    var n = rbsLen(s)
    if (pad.length == 0 || width <= n) { return s }
    var total = width - n
    var left = 0
    if (mode == 0) { left = Math.floor(total / 2) } else if (mode == 2) { left = total }
    return rbsRepeatTo(pad, left) + s + rbsRepeatTo(pad, total - left)
}

// The first n characters of pad repeated.
function rbsRepeatTo(pad, n) {
    var out = ""
    var i = 0
    var c = 0
    while (c < n) {
        if (i >= pad.length) { i = 0 }
        var k = rbsCharLen(pad, i)
        out = out + pad.substring(i, i + k)
        i = i + k
        c = c + 1
    }
    return out
}

// Where String#index starts searching for character position pos (negative
// counts from the end): a unit index, or -1 when pos is outside the string.
function rbsIndexStart(s, pos) {
    var n = rbsCharIdx(s, s.length)
    var p = pos < 0 ? pos + n : pos
    if (p < 0 || p > n) { return -1 }
    return rbsUnitIdx(s, p)
}
// Where String#rindex searches back from: pos is clamped to the end, and -1 says
// a negative pos reached before the start.
function rbsRindexStart(s, pos, given) {
    var n = rbsCharIdx(s, s.length)
    var p = given ? pos : n
    if (p < 0) { p = p + n }
    if (p < 0) { return -1 }
    if (p > n) { p = n }
    return rbsUnitIdx(s, p)
}
// The last unit index <= from where t starts, or -1.
function rbsRfind(s, t, from) {
    var u = from
    if (u + t.length > s.length) { u = s.length - t.length }
    while (u >= 0) {
        if (s.substring(u, u + t.length) == t) { return u }
        u = u - 1
    }
    return -1
}
// The last unit index <= from where re matches, or -1 (rb_reg_search backward).
function rbsRfindRx(re, s, from) {
    var u = from
    while (u >= 0) {
        var m = rxMatchAt(re, s, u)
        if (m !== null && m.begin >= 0) { return u }
        u = u - 1
    }
    return -1
}

// MRI's ISALNUM classes for String#succ: 1 a digit, 2 a letter, 0 neither.
function rbsSuccKind(c) {
    if (c >= 48 && c <= 57) { return 1 }
    if ((c >= 97 && c <= 122) || (c >= 65 && c <= 90)) { return 2 }
    return 0
}
// String#succ (str_succ): the rightmost alphanumeric is incremented, carrying
// leftwards through alphanumerics ("az" -> "ba", "zz" -> "aaa") and across
// non-alphanumerics ("1.9" -> "2.0") unless the next one is of the other kind
// ("a-9" -> "a-10"). A string without alphanumerics increments its last
// character.
function rbsSucc(s) {
    if (s.length == 0) { return s }
    var out = s
    var i = s.length - 1
    var skipped = false
    var lastKind = 0
    var carryAt = -1
    var carry = ""
    while (i >= 0) {
        var c = s.charCodeAt(i)
        var k = rbsSuccKind(c)
        if (skipped && lastKind != 0 && k != 0 && k != lastKind) { break }
        if (k == 0) {
            skipped = true
            i = i - 1
            continue
        }
        skipped = false
        var w = ""
        if (c == 57) {
            w = "0"
            carry = "1"
        } else if (c == 122) {
            w = "a"
            carry = "a"
        } else if (c == 90) {
            w = "A"
            carry = "A"
        } else {
            return out.substring(0, i) + String.fromCharCode(c + 1) + out.substring(i + 1, out.length)
        }
        out = out.substring(0, i) + w + out.substring(i + 1, out.length)
        lastKind = k
        carryAt = i
        i = i - 1
    }
    if (carryAt >= 0) { return out.substring(0, carryAt) + carry + out.substring(carryAt, out.length) }
    var j = s.length - 1
    var lo = s.charCodeAt(j)
    if (j > 0 && lo >= 56320 && lo <= 57343 && rbsCharLen(s, j - 1) == 2) {
        var cp = (s.charCodeAt(j - 1) - 55296) * 1024 + (lo - 56320) + 65536 + 1
        return s.substring(0, j - 1) + String.fromCharCode(55296 + Math.floor((cp - 65536) / 1024), 56320 + (cp - 65536) % 1024)
    }
    if (lo == 55295) { return s.substring(0, j) + String.fromCharCode(57344) }
    if (lo == 65535) { return s.substring(0, j) + String.fromCharCode(55296, 56320) }
    return s.substring(0, j) + String.fromCharCode(lo + 1)
}

// The text of a sub / gsub replacement template for match m of re in s
// (rb_reg_regsub): \0..\9 and \k<name> a group, \& the match, \` the text before
// it, \' the text after it, \+ the last group that matched, \\ one backslash; a
// backslash before anything else stays as written.
function rbsExpand(re, s, m, repl) {
    var out = ""
    var i = 0
    while (i < repl.length) {
        var c = repl.charCodeAt(i)
        if (c != 92 || i + 1 >= repl.length) {
            out = out + repl.charAt(i)
            i = i + 1
            continue
        }
        var d = repl.charCodeAt(i + 1)
        var g = -1
        if (d >= 48 && d <= 57) {
            g = d - 48
        } else if (d == 38) {
            g = 0
        } else if (d == 43) {
            g = re.ngroups
            while (g > 0 && m.caps[2 * g] < 0) { g = g - 1 }
        } else if (d == 96) {
            out = out + s.substring(0, m.begin)
        } else if (d == 39) {
            out = out + s.substring(m.end, s.length)
        } else if (d == 92) {
            out = out + "\\"
        } else if (d == 107 && i + 2 < repl.length && repl.charCodeAt(i + 2) == 60 && rbsFind(repl, ">", i + 3) >= 0) {
            var j = rbsFind(repl, ">", i + 3)
            g = rxIndexOfName(re, repl.substring(i + 3, j))
            i = j - 1
        } else {
            out = out + repl.substring(i, i + 2)
        }
        if (g >= 0) {
            var t = rxGroup(re, s, m, g)
            if (t !== null) { out = out + t }
        }
        i = i + 2
    }
    return out
}

// ----- Binary (ASCII-8BIT) Strings -----
// A binary String's text holds one char per byte, U+0000..U+00FF. Go twins in
// abnf/jsrtrubystr.go.

// The UTF-8 bytes of s, one char each: String#b.
function rbsBinOf(s) {
    var out = ""
    var i = 0
    while (i < s.length) {
        var c = s.charCodeAt(i)
        if (rbsCharLen(s, i) == 2) {
            c = (c - 55296) * 1024 + (s.charCodeAt(i + 1) - 56320) + 65536
            i = i + 1
        }
        i = i + 1
        if (c < 128) { out = out + String.fromCharCode(c) }
        else if (c < 2048) { out = out + String.fromCharCode(192 + Math.floor(c / 64), 128 + (c % 64)) }
        else if (c < 65536) {
            out = out + String.fromCharCode(224 + Math.floor(c / 4096), 128 + (Math.floor(c / 64) % 64), 128 + (c % 64))
        } else {
            out = out + String.fromCharCode(240 + Math.floor(c / 262144), 128 + (Math.floor(c / 4096) % 64),
                                            128 + (Math.floor(c / 64) % 64), 128 + (c % 64))
        }
    }
    return out
}

// The text a binary String's bytes spell as UTF-8, or null when they are not
// valid UTF-8 (a stray continuation byte, an overlong form, a surrogate, past
// U+10FFFF, or a sequence cut short).
function rbsBinDecode(b) {
    var out = ""
    var i = 0
    var err = []
    while (i < b.length) {
        var c = b.charCodeAt(i)
        var n = 0
        var cp = c
        if (c >= 194 && c <= 223) { n = 1; cp = c - 192 }
        else if (c >= 224 && c <= 239) { n = 2; cp = c - 224 }
        else if (c >= 240 && c <= 244) { n = 3; cp = c - 240 }
        else if (c >= 128) { return null }
        if (i + n >= b.length && n > 0) { return null }
        var k = 1
        while (k <= n) {
            var d = b.charCodeAt(i + k)
            if (d < 128 || d > 191) { return null }
            cp = cp * 64 + (d - 128)
            k = k + 1
        }
        if ((n == 2 && cp < 2048) || (n == 3 && cp < 65536)) { return null }
        var ch = rbsChrUtf8(cp, err)
        if (err.length > 0) { return null }
        out = out + ch
        i = i + n + 1
    }
    return out
}

// Whether every char of s is ASCII.
function rbsAsciiOnly(s) {
    var i = 0
    while (i < s.length) {
        if (s.charCodeAt(i) > 127) { return false }
        i = i + 1
    }
    return true
}

// A binary String's #inspect: printable ASCII as itself, the named escapes, and
// every other byte as \xHH.
function rbsBinInspect(b) {
    var hx = "0123456789ABCDEF"
    var out = "\""
    var i = 0
    while (i < b.length) {
        var c = b.charCodeAt(i)
        var nx = i + 1 < b.length ? b.charCodeAt(i + 1) : 0
        if (c == 34) { out = out + "\\\"" }
        else if (c == 92) { out = out + "\\\\" }
        else if (c == 10) { out = out + "\\n" }
        else if (c == 9) { out = out + "\\t" }
        else if (c == 13) { out = out + "\\r" }
        else if (c == 12) { out = out + "\\f" }
        else if (c == 11) { out = out + "\\v" }
        else if (c == 8) { out = out + "\\b" }
        else if (c == 7) { out = out + "\\a" }
        else if (c == 27) { out = out + "\\e" }
        else if (c == 35 && (nx == 123 || nx == 36 || nx == 64)) { out = out + "\\#" }
        else if (c < 32 || c >= 127) { out = out + "\\x" + hx.charAt(Math.floor(c / 16)) + hx.charAt(c % 16) }
        else { out = out + b.charAt(i) }
        i = i + 1
    }
    return out + "\""
}

// Array#pack and String#unpack over the directives C (unsigned byte), c (signed
// byte) and U (UTF-8 character), each with an optional count or `*'. The
// directives as [letter, count] pairs, count -1 for `*'; err[0] names the first
// directive outside that set.
function rbsPackDirs(fmt, err) {
    var out = []
    var i = 0
    while (i < fmt.length) {
        var d = fmt.charAt(i)
        i = i + 1
        if (d == " ") { continue }
        if (d != "C" && d != "c" && d != "U") {
            err.push(d)
            return out
        }
        var n = 1
        if (i < fmt.length && fmt.charAt(i) == "*") {
            n = 0 - 1
            i = i + 1
        } else if (i < fmt.length && fmt.charCodeAt(i) >= 48 && fmt.charCodeAt(i) <= 57) {
            n = 0
            while (i < fmt.length && fmt.charCodeAt(i) >= 48 && fmt.charCodeAt(i) <= 57) {
                n = n * 10 + (fmt.charCodeAt(i) - 48)
                i = i + 1
            }
        }
        out.push([d, n])
    }
    return out
}

// Array#pack of the Integers in vals: the text, which is a binary String's
// unless the first directive is U. err[0] is the ArgumentError's message.
function rbsPack(vals, dirs, err) {
    var out = ""
    var k = 0
    var i = 0
    while (i < dirs.length) {
        var d = dirs[i][0]
        var n = dirs[i][1]
        if (n < 0) { n = vals.length - k }
        var j = 0
        while (j < n) {
            if (k >= vals.length) {
                err.push("too few arguments")
                return out
            }
            var v = vals[k]
            k = k + 1
            if (d == "U") {
                var ch = rbsChrUtf8(v, err)
                if (err.length > 0) {
                    err[0] = "pack(U): value out of range"
                    return out
                }
                out = out + (dirs[0][0] == "U" ? ch : rbsBinOf(ch))
            } else {
                out = out + String.fromCharCode(((v % 256) + 256) % 256)
            }
            j = j + 1
        }
        i = i + 1
    }
    return out
}

// String#unpack of the bytes b (one char per byte): an Array of Integers, nil
// for a counted C / c past the end. err[0] is the ArgumentError's message.
function rbsUnpack(b, dirs, err) {
    var out = []
    var p = 0
    var i = 0
    while (i < dirs.length) {
        var d = dirs[i][0]
        var n = dirs[i][1]
        var j = 0
        while (n < 0 ? p < b.length : j < n) {
            if (p >= b.length) {
                if (d != "U") { out.push(null) }
            } else if (d == "U") {
                var c = b.charCodeAt(p)
                var len = c < 128 ? 1 : (c >= 240 ? 4 : (c >= 224 ? 3 : 2))
                var t = rbsBinDecode(b.substring(p, p + len))
                if (t === null) {
                    err.push("malformed UTF-8 character")
                    return out
                }
                out.push(rbsOrd(t))
                p = p + len
            } else {
                var v = b.charCodeAt(p)
                out.push(d == "c" && v > 127 ? v - 256 : v)
                p = p + 1
            }
            j = j + 1
        }
        i = i + 1
    }
    return out
}
