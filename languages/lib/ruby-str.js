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
// trailing empty fields; negative keeps them. last[0] receives the result of
// the final pattern search (null when it failed), which MRI leaves in $~; it is
// untouched when no search ran.
function rbsSplit(s, sep, re, lim, last) {
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
            last[0] = m
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

// String#chars: one element per CHARACTER, a surrogate pair taken whole as one
// substring (rebuilt from its two halves it would be two U+FFFD).
function rbsChars(s) {
    var out = []
    var i = 0
    while (i < s.length) {
        var k = rbsCharLen(s, i)
        out.push(s.substring(i, i + k))
        i = i + k
    }
    return out
}

// The index of character ch in the list, or -1.
function rbsListIdx(list, ch) {
    var i = 0
    while (i < list.length) {
        if (list[i] == ch) { return i }
        i = i + 1
    }
    return -1
}

// MRI's character-set notation, shared by tr / tr_s / count / delete / squeeze
// (tr_setup_table in string.c): a leading ^ negates, `a-z` is a range of code
// points, and a backslash escapes the next character. Answers {neg, chars} with
// every range expanded, one character per entry.
function rbsSetExpand(spec) {
    var cs = rbsChars(spec)
    var neg = false
    var i = 0
    if (cs.length > 1 && cs[0] == "^") { neg = true; i = 1 }
    var chars = []
    while (i < cs.length) {
        if (cs[i] == "\\" && i + 1 < cs.length) {
            chars.push(cs[i + 1])
            i = i + 2
            continue
        }
        if (i + 2 < cs.length && cs[i + 1] == "-") {
            var k = rbsOrd(cs[i])
            var b = rbsOrd(cs[i + 2])
            while (k <= b) {
                if (k < 65536) { chars.push(String.fromCharCode(k)) }
                else { chars.push(String.fromCharCode(55296 + Math.floor((k - 65536) / 1024), 56320 + ((k - 65536) % 1024))) }
                k = k + 1
            }
            i = i + 3
            continue
        }
        chars.push(cs[i])
        i = i + 1
    }
    return {neg: neg, chars: chars}
}

// Is ch in EVERY one of the expanded sets? MRI intersects the sets of count /
// delete / squeeze when more than one is given.
function rbsInSets(sets, ch) {
    var i = 0
    while (i < sets.length) {
        var found = rbsListIdx(sets[i].chars, ch) >= 0
        if (sets[i].neg) { found = !found }
        if (!found) { return false }
        i = i + 1
    }
    return sets.length > 0
}
function rbsSetsOf(specs) {
    var out = []
    var i = 0
    while (i < specs.length) {
        out.push(rbsSetExpand(specs[i]))
        i = i + 1
    }
    return out
}

// String#tr / #tr_s. An empty `to` DELETES; a shorter one repeats its last
// character; a negated `from` maps everything outside the set to that last
// character. tr_s additionally squeezes the runs it produced.
function rbsTr(s, from, to, sq) {
    var f = rbsSetExpand(from)
    var t = rbsSetExpand(to)
    var tc = t.chars
    var last = tc.length > 0 ? tc[tc.length - 1] : ""
    var cs = rbsChars(s)
    var out = ""
    var prev = ""
    var i = 0
    while (i < cs.length) {
        var ch = cs[i]
        i = i + 1
        var k = rbsListIdx(f.chars, ch)
        var rep = ch
        var did = false
        if (f.neg) {
            if (k < 0) { rep = last; did = true }
        } else if (k >= 0) {
            if (tc.length == 0) { rep = "" } else if (k >= tc.length) { rep = last } else { rep = tc[k] }
            did = true
        }
        if (rep == "") { prev = ""; continue }
        if (sq && did && rep == prev) { continue }
        out = out + rep
        prev = did ? rep : ""
    }
    return out
}

// String#squeeze: collapse a run of the same character, restricted to the sets
// when any are given.
function rbsSqueeze(s, specs) {
    var sets = rbsSetsOf(specs)
    var cs = rbsChars(s)
    var out = ""
    var prev = ""
    var i = 0
    while (i < cs.length) {
        var ch = cs[i]
        i = i + 1
        if (ch == prev && (sets.length == 0 || rbsInSets(sets, ch))) { continue }
        out = out + ch
        prev = ch
    }
    return out
}

// String#count / #delete: the characters in every set, counted or removed.
function rbsCount(s, specs) {
    var sets = rbsSetsOf(specs)
    var cs = rbsChars(s)
    var n = 0
    var i = 0
    while (i < cs.length) {
        if (rbsInSets(sets, cs[i])) { n = n + 1 }
        i = i + 1
    }
    return n
}
function rbsDelete(s, specs) {
    var sets = rbsSetsOf(specs)
    var cs = rbsChars(s)
    var out = ""
    var i = 0
    while (i < cs.length) {
        if (!rbsInSets(sets, cs[i])) { out = out + cs[i] }
        i = i + 1
    }
    return out
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
// character within its UTF-8 length (enc_succ_char): past the largest one of
// that length it wraps to the smallest and the carry moves one character left,
// and a string the carry runs off gains "\x01" in front ("\x7f" -> "\x01\x00").
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
    var tail = ""
    var j = s.length
    var err = []
    while (j > 0) {
        var at = j - 1
        var cp = s.charCodeAt(at)
        if (at > 0 && cp >= 56320 && cp <= 57343 && rbsCharLen(s, at - 1) == 2) {
            at = at - 1
            cp = (s.charCodeAt(at) - 55296) * 1024 + (cp - 56320) + 65536
        }
        var wrap = rbsSuccWrap(cp)
        if (wrap < 0) { return s.substring(0, at) + rbsChrUtf8(cp == 55295 ? 57344 : cp + 1, err) + tail }
        tail = rbsChrUtf8(wrap, err) + tail
        j = at
    }
    return "\x01" + tail
}
// The smallest code point of cp's UTF-8 length when cp is the largest, else -1.
function rbsSuccWrap(cp) {
    if (cp == 127) { return 0 }
    if (cp == 2047) { return 128 }
    if (cp == 65535) { return 2048 }
    if (cp == 1114111) { return 65536 }
    return 0 - 1
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

// String#setbyte over the bytes b: byte i (counted from the end when negative)
// becomes v mod 256. Answers null when i is outside b. Go twin: rubySetByte in
// abnf/jsrtrubystr.go.
function rbsSetByte(b, i, v) {
    var k = i < 0 ? i + b.length : i
    if (k < 0 || k >= b.length) { return null }
    var m = v % 256
    if (m < 0) { m = m + 256 }
    return b.substring(0, k) + String.fromCharCode(m) + b.substring(k + 1)
}

// The text a binary String's bytes spell as UTF-8, or null when they are not
// valid UTF-8 (a stray continuation byte, an overlong form, a surrogate, past
// U+10FFFF, or a sequence cut short).
function rbsBinDecode(b) {
    var out = ""
    var i = 0
    var err = []
    while (i < b.length) {
        var n = rbsU8At(b, i)
        if (n == 0) { return null }
        out = out + (n == 1 ? b.charAt(i) : rbsChrUtf8(rbsU8Cp(b, i, n), err))
        i = i + n
    }
    return out
}

// A string literal whose \xHH or \nnn escapes past 127 are BYTES (raw lists
// where each landed in out, one char each): the literal is read as UTF-8 bytes,
// which spell its text when they are valid ("\xC3\xA9" is "é"). Invalid ones
// leave out as it is, and inv, when given, receives the bytes for a String
// literal to box.
function rbsEscBytes(out, raw, inv) {
    var b = ""
    var r = 0
    var i = 0
    while (i < out.length) {
        if (r < raw.length && raw[r] == i) {
            b = b + out.charAt(i)
            r = r + 1
            i = i + 1
        } else {
            var k = rbsCharLen(out, i)
            b = b + rbsBinOf(out.substring(i, i + k))
            i = i + k
        }
    }
    var t = rbsBinDecode(b)
    if (t !== null) { return t }
    if (inv !== undefined && inv !== null) { inv.push(b) }
    return out
}

// The byte length of the UTF-8 character at byte i of b, 0 when no valid one
// starts there.
function rbsU8At(b, i) {
    var c = b.charCodeAt(i)
    if (c < 128) { return 1 }
    var n = 0
    if (c >= 194 && c <= 223) { n = 2 }
    else if (c >= 224 && c <= 239) { n = 3 }
    else if (c >= 240 && c <= 244) { n = 4 }
    else { return 0 }
    if (i + n > b.length) { return 0 }
    var k = 1
    while (k < n) {
        var d = b.charCodeAt(i + k)
        if (d < 128 || d > 191) { return 0 }
        k = k + 1
    }
    var cp = rbsU8Cp(b, i, n)
    if ((n == 3 && (cp < 2048 || (cp >= 55296 && cp <= 57343))) || (n == 4 && (cp < 65536 || cp > 1114111))) { return 0 }
    return n
}

// The code point of the n-byte UTF-8 character at byte i of b.
function rbsU8Cp(b, i, n) {
    var cp = b.charCodeAt(i) % (n == 2 ? 32 : (n == 3 ? 16 : 8))
    var k = 1
    while (k < n) {
        cp = cp * 64 + (b.charCodeAt(i + k) - 128)
        k = k + 1
    }
    return cp
}

// The characters of UTF-8 bytes b where a byte that starts no valid character
// counts as one: String#size of a String with an invalid byte.
function rbsU8Len(b) {
    var n = 0
    var i = 0
    while (i < b.length) {
        var k = rbsU8At(b, i)
        i = i + (k == 0 ? 1 : k)
        n = n + 1
    }
    return n
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

// upcase / downcase / swapcase / capitalize (name) of the bytes b of a String
// that is not UTF-8: MRI maps the ASCII letters only, every other byte stays.
function rbsAsciiCase(b, name) {
    var out = ""
    var i = 0
    while (i < b.length) {
        var c = b.charCodeAt(i)
        var up = name == "upcase" || (name == "capitalize" && i == 0)
        var down = name == "downcase" || (name == "capitalize" && i > 0)
        if ((up || name == "swapcase") && c >= 97 && c <= 122) { c = c - 32 }
        else if ((down || name == "swapcase") && c >= 65 && c <= 90) { c = c + 32 }
        out = out + String.fromCharCode(c)
        i = i + 1
    }
    return out
}

// A binary String's #inspect: printable ASCII as itself, the named escapes, and
// every other byte as \xHH.
function rbsBinInspect(b) { return rbsDump(b, false) }

// String#dump of the bytes b: rbsBinInspect's form, where u8 (a UTF-8 String)
// writes each valid multi-byte character as \uXXXX, or \u{X} past U+FFFF. Go
// twin: rubyDump in abnf/jsrtrubystr.go.
function rbsDump(b, u8) {
    var hx = "0123456789ABCDEF"
    var out = "\""
    var i = 0
    while (i < b.length) {
        var c = b.charCodeAt(i)
        var nx = i + 1 < b.length ? b.charCodeAt(i + 1) : 0
        var n = u8 && c >= 128 ? rbsU8At(b, i) : 0
        if (n > 1) {
            var cp = rbsHexUp(rbsU8Cp(b, i, n))
            out = out + (cp.length > 4 ? "\\u{" + cp + "}" : "\\u" + "0000".substring(cp.length) + cp)
            i = i + n
            continue
        }
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

// The encoding a name selects (case does not matter): ASCII-8BIT, US-ASCII or
// UTF-8, or "" for any other. Go twin: rubyEncCanon in abnf/jsrtrubystr.go.
function rbsEncCanon(n) {
    var u = n.toUpperCase()
    if (u == "BINARY" || u == "ASCII-8BIT") { return "ASCII-8BIT" }
    if (u == "US-ASCII" || u == "ASCII") { return "US-ASCII" }
    if (u == "UTF-8") { return "UTF-8" }
    return ""
}

// The value of the hex digits of s from i up to (not including) lim, and how
// many there were.
function rbsHexRun(s, i, lim) {
    var v = 0
    var k = i
    while (k < lim) {
        var d = "0123456789abcdef".indexOf(s.charAt(k).toLowerCase())
        if (d < 0) { break }
        v = v * 16 + d
        k = k + 1
    }
    return {v: v, n: k - i}
}

// String#undump of the bytes s, MRI's str_undump: s must be a double-quoted
// String#dump form, optionally followed by .force_encoding("NAME") (with or
// without .dup). Answers {ok: true, b: the bytes, enc: the encoding name, "" for
// the receiver's own} or {ok: false, msg: the RuntimeError's message}. A \u
// escape makes the result UTF-8 and cannot be mixed with \x. Go twin:
// rubyUndump in abnf/jsrtrubystr.go.
function rbsUndump(s) {
    var badForm = "invalid dumped string; not wrapped with '\"' nor '\"...\".force_encoding(\"...\")' form"
    if (!rbsAsciiOnly(s)) { return {ok: false, msg: "non-ASCII character detected"} }
    if (s.indexOf(String.fromCharCode(0)) >= 0) { return {ok: false, msg: "string contains null byte"} }
    if (s.length < 2 || s.charAt(0) != "\"") { return {ok: false, msg: badForm} }
    var n = s.length
    var out = ""
    var enc = ""
    var utf8 = false
    var hexed = false
    var i = 1
    while (true) {
        if (i >= n) { return {ok: false, msg: "unterminated dumped string"} }
        var c = s.charAt(i)
        if (c == "\"") { break }
        if (c != "\\") {
            out = out + c
            i = i + 1
            continue
        }
        i = i + 1
        if (i >= n) { return {ok: false, msg: "invalid escape"} }
        var e = s.charAt(i)
        var named = "nrtfvbae".indexOf(e)
        if (e == "\\" || e == "\"" || e == "#") {
            out = out + e
            i = i + 1
        } else if (named >= 0) {
            out = out + String.fromCharCode([10, 13, 9, 12, 11, 8, 7, 27][named])
            i = i + 1
        } else if (e == "u") {
            if (hexed) { return {ok: false, msg: "hex escape and Unicode escape are mixed"} }
            utf8 = true
            enc = "UTF-8"
            i = i + 1
            if (i >= n) { return {ok: false, msg: "invalid Unicode escape"} }
            var braced = s.charAt(i) == "{"
            if (braced) { i = i + 1 }
            while (true) {
                if (braced && i >= n) { return {ok: false, msg: "unterminated Unicode escape"} }
                var ch = s.charAt(i)
                if (braced && ch == "}") { i = i + 1; break }
                var cc = s.charCodeAt(i)
                if (braced && (cc == 32 || (cc >= 9 && cc <= 13))) { i = i + 1; continue }
                var h = rbsHexRun(s, i, braced || i + 4 > n ? n : i + 4)
                if (braced ? h.n == 0 || h.n > 6 : h.n != 4) { return {ok: false, msg: "invalid Unicode escape"} }
                if (h.v > 1114111) { return {ok: false, msg: "invalid Unicode codepoint (too large)"} }
                if (h.v >= 55296 && h.v <= 57343) { return {ok: false, msg: "invalid Unicode codepoint"} }
                out = out + rbsBinOf(rbsChrUtf8(h.v, []))
                i = i + h.n
                if (!braced) { break }
            }
        } else if (e == "x") {
            if (utf8) { return {ok: false, msg: "hex escape and Unicode escape are mixed"} }
            hexed = true
            i = i + 1
            var h2 = rbsHexRun(s, i, i + 2 < n ? i + 2 : n)
            if (h2.n != 2) { return {ok: false, msg: "invalid hex escape"} }
            out = out + String.fromCharCode(h2.v)
            i = i + 2
        } else {
            out = out + "\\" + e
            i = i + 1
        }
    }
    var rest = s.substring(i + 1)
    if (rest == "") { return {ok: true, b: out, enc: enc} }
    if (rest.length > 4 && rest.substring(0, 4) == ".dup") { rest = rest.substring(4) }
    var fe = ".force_encoding(\""
    if (rest.length <= fe.length || rest.substring(0, fe.length) != fe) { return {ok: false, msg: badForm} }
    if (utf8) { return {ok: false, msg: "dumped string contained Unicode escape but used force_encoding"} }
    rest = rest.substring(fe.length)
    var q = rest.indexOf("\"")
    if (q < 0 || rest.substring(q) != "\")") { return {ok: false, msg: badForm} }
    var en = rbsEncCanon(rest.substring(0, q))
    if (en == "") { return {ok: false, msg: "dumped string has unknown encoding name"} }
    return {ok: true, b: out, enc: en}
}

// The invalid UTF-8 sequence at byte i of b (where rbsU8At answers 0) as MRI's
// transcoder delimits it: a lead byte and the continuation bytes that may
// follow it. [its length, 0 for a byte that starts no character, 1 for a
// sequence the bytes end inside, 2 for one the next byte breaks].
function rbsU8Bad(b, i) {
    var c = b.charCodeAt(i)
    var n = 0
    if (c >= 194 && c <= 223) { n = 2 }
    else if (c >= 224 && c <= 239) { n = 3 }
    else if (c >= 240 && c <= 244) { n = 4 }
    else { return [1, 0] }
    var k = 1
    while (k < n) {
        if (i + k >= b.length) { return [k, 1] }
        var d = b.charCodeAt(i + k)
        var lo = 128
        var hi = 191
        if (k == 1 && c == 224) { lo = 160 }
        if (k == 1 && c == 237) { hi = 159 }
        if (k == 1 && c == 240) { lo = 144 }
        if (k == 1 && c == 244) { hi = 143 }
        if (d < lo || d > hi) { return [k, 2] }
        k = k + 1
    }
    return [n, 2]
}

// String#encode over the modelled encodings UTF-8, US-ASCII and ASCII-8BIT:
// the bytes b of encoding src written in dst. inv and undef are what replaces an
// invalid byte sequence and a character dst has no code for, null to raise;
// err receives MRI's message and class then. Bytes in one encoding are copied
// as they are unless inv is given (String#scrub's case).
// The bytes b of a String in encoding enc cut at its invalid sequences (as
// rbsU8Bad delimits them): the even entries are the valid runs, the odd ones
// the sequences #scrub hands its block. Go twin: rubyScrubSplit in
// abnf/jsrtrubystr.go.
function rbsScrubSplit(b, enc) {
    var parts = []
    var run = ""
    var i = 0
    while (i < b.length) {
        var c = b.charCodeAt(i)
        var n = 1
        var bad = c >= 128 && enc == "US-ASCII"
        if (c >= 128 && enc == "UTF-8") {
            n = rbsU8At(b, i)
            if (n == 0) {
                n = rbsU8Bad(b, i)[0]
                bad = true
            }
        }
        if (bad) {
            parts.push(run)
            parts.push(b.substring(i, i + n))
            run = ""
        } else {
            run = run + b.substring(i, i + n)
        }
        i = i + n
    }
    parts.push(run)
    return parts
}

function rbsEncode(b, src, dst, inv, undef, err) {
    if (src == dst && inv === null) { return b }
    var out = ""
    var i = 0
    while (i < b.length) {
        var c = b.charCodeAt(i)
        if (c < 128) {
            out = out + b.charAt(i)
            i = i + 1
        } else if (src == "ASCII-8BIT") {
            if (dst == "ASCII-8BIT") { out = out + b.charAt(i) }
            else if (undef !== null) { out = out + undef }
            else {
                err.push(rbsBinInspect(b.charAt(i)) + " from ASCII-8BIT to " + dst)
                err.push("Encoding::UndefinedConversionError")
                return out
            }
            i = i + 1
        } else if (src == "US-ASCII") {
            if (inv === null) {
                err.push(rbsBinInspect(b.charAt(i)) + " on US-ASCII")
                err.push("Encoding::InvalidByteSequenceError")
                return out
            }
            out = out + inv
            i = i + 1
        } else {
            var n = rbsU8At(b, i)
            if (n > 0 && dst == "UTF-8") { out = out + b.substring(i, i + n) }
            else if (n > 0 && undef !== null) { out = out + undef }
            else if (n > 0) {
                var hx = rbsHexUp(rbsU8Cp(b, i, n))
                while (hx.length < 4) { hx = "0" + hx }
                err.push("U+" + hx + " from UTF-8 to " + dst)
                err.push("Encoding::UndefinedConversionError")
                return out
            } else {
                var bad = rbsU8Bad(b, i)
                n = bad[0]
                if (inv !== null) { out = out + inv }
                else {
                    var q = rbsBinInspect(b.substring(i, i + n))
                    if (bad[1] == 1) { q = "incomplete " + q }
                    if (bad[1] == 2) { q = q + " followed by " + rbsBinInspect(b.charAt(i + n)) }
                    err.push(q + " on UTF-8")
                    err.push("Encoding::InvalidByteSequenceError")
                    return out
                }
            }
            i = i + n
        }
    }
    return out
}

// Upper-case hex digits of a non-negative integer.
function rbsHexUp(v) {
    var hx = "0123456789ABCDEF"
    var out = ""
    var x = v
    while (x >= 16) {
        out = hx.charAt(x % 16) + out
        x = Math.floor(x / 16)
    }
    return hx.charAt(x) + out
}

// ----- Array#pack / String#unpack -----
// Integers (C c S s L l Q q J j I i n N v V U, with the _ ! < > modifiers),
// floats (e E g G d D f F), Strings (a A Z B b H h m) and positions (x X @),
// each with a count or `*'. Go twins: rubyPack / rubyUnpack in
// abnf/jsrtrubystr.go.
//
// The engine hands rbsPack its values as a number (an Integer), {flo: number}
// (a Float), {big: decimal text} (an Integer past 2^53), {str: bytes} (a
// String, one char per byte), null (nil) or {cls: class name}; rbsUnpack answers
// numbers, {flo: number}, {big: decimal text}, {str: bytes, enc: encoding name}
// and null. An error is err[0] (the message) and err[1] (the class,
// ArgumentError when absent).

// The byte width of an integer directive, 0 for any other one, -1 for one that
// is not modelled.
function rbsPackSize(d) {
    if (d == "C" || d == "c") { return 1 }
    if (d == "S" || d == "s" || d == "n" || d == "v") { return 2 }
    if (d == "L" || d == "l" || d == "N" || d == "V" || d == "I" || d == "i") { return 4 }
    if (d == "Q" || d == "q" || d == "J" || d == "j") { return 8 }
    if (d == "e" || d == "f" || d == "F" || d == "g") { return 4 }
    if (d == "E" || d == "d" || d == "D" || d == "G") { return 8 }
    if (d == "U" || d == "a" || d == "A" || d == "Z" || d == "B" || d == "b" || d == "H" || d == "h"
        || d == "m" || d == "x" || d == "X" || d == "@") { return 0 }
    return 0 - 1
}

// The directives as [letter, count, byte width, big-endian, signed] records;
// count -1 for `*'. err[0] names the first directive that is not modelled, or
// is MRI's message with err[1] its class for a modifier after a type that
// takes none.
function rbsPackDirs(fmt, err) {
    var out = []
    var i = 0
    while (i < fmt.length) {
        var d = fmt.charAt(i)
        i = i + 1
        if (d == " " || d == "\t" || d == "\n") { continue }
        var size = rbsPackSize(d)
        if (size < 0) {
            err.push(d)
            return out
        }
        var be = d == "n" || d == "N" || d == "g" || d == "G"
        var signed = d == "c" || d == "s" || d == "l" || d == "q" || d == "j" || d == "i"
        var native = "sSiIlLqQjJ"
        while (i < fmt.length && (fmt.charAt(i) == "_" || fmt.charAt(i) == "!" || fmt.charAt(i) == "<" || fmt.charAt(i) == ">")) {
            var mo = fmt.charAt(i)
            if (native.indexOf(d) < 0) {
                err.push("'" + mo + "' allowed only after types " + native)
                err.push("ArgumentError")
                return out
            }
            if (mo == "<" || mo == ">") { be = mo == ">" }
            else if (d == "l" || d == "L") { size = 8 }
            i = i + 1
        }
        var n = d == "m" ? 45 : 1
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
        out.push([d, n, size, be, signed])
    }
    return out
}

// The value's class for a TypeError message.
function rbsPackCls(v) {
    if (v === null) { return "nil" }
    if (typeof v == "number") { return "Integer" }
    if (typeof v.big == "string") { return "Integer" }
    if (typeof v.flo == "number") { return "Float" }
    if (typeof v.str == "string") { return "String" }
    return v.cls
}

// A pack value for an integer directive: a Float truncated.
function rbsPackIntVal(v) {
    if (v !== null && typeof v == "object" && typeof v.flo == "number") { return Math.trunc(v.flo) }
    return v
}

// A float directive: e E g G d D f F.
function rbsFloDir(d) {
    return d == "e" || d == "E" || d == "g" || d == "G" || d == "d" || d == "D" || d == "f" || d == "F"
}

// x * 2^k, exact while the result is a double.
function rbsScale2(x, k) {
    var v = x
    var n = k
    while (n > 0) { v = v * 2; n = n - 1 }
    while (n < 0) { v = v / 2; n = n + 1 }
    return v
}

// The size (4 or 8) little-endian bytes of the IEEE 754 binary32 / binary64 of
// x, half to even. A value past the largest binary32 is an infinity there
// (MRI's VALUE_to_float), and a NaN is the quiet one with the sign clear.
function rbsFloBytes(x, size) {
    var mb = size == 4 ? 23 : 52
    var bias = size == 4 ? 127 : 1023
    var emax = size == 4 ? 255 : 2047
    var top = rbsScale2(1, mb)
    var zero = 0
    var sign = x < 0 || (x === 0 && 1 / x < 0) ? 1 : 0
    var a = sign == 1 ? -x : x
    var ex = 0
    var f = 0
    if (x !== x) {
        sign = 0
        ex = emax
        f = top / 2
    } else if (a !== 0 && (a / 2 === a || (size == 4 && a > rbsScale2(16777215, 104)))) {
        ex = emax
    } else if (a !== 0) {
        var e = 0
        var t = a
        while (t >= 2) { t = t / 2; e = e + 1 }
        while (t < 1) { t = t * 2; e = e - 1 }
        var q = e - mb
        if (q < 1 - bias - mb) { q = 1 - bias - mb }
        var s = rbsScale2(a, -q)
        var m = Math.floor(s)
        var d = s - m
        if (d > 0.5 || (d === 0.5 && m % 2 == 1)) { m = m + 1 }
        ex = q + mb + bias
        if (m >= top * 2) {
            m = m / 2
            ex = ex + 1
        }
        if (m < top) { ex = 0 }
        f = m < top ? m : m - top
        if (ex >= emax) {
            ex = emax
            f = 0
        }
    }
    var bits = []
    var v = f
    while (bits.length < mb) {
        bits.push(v % 2)
        v = Math.floor(v / 2)
    }
    v = ex
    while (bits.length < size * 8 - 1) {
        bits.push(v % 2)
        v = Math.floor(v / 2)
    }
    bits.push(sign)
    var out = []
    var k = 0
    while (k < size) {
        var by = 0
        var j = 7
        while (j >= 0) {
            by = by * 2 + bits[k * 8 + j]
            j = j - 1
        }
        out.push(by)
        k = k + 1
    }
    return out
}

// The Float the size bytes of b at p spell (big-endian when be).
function rbsFloOf(b, p, size, be) {
    var mb = size == 4 ? 23 : 52
    var bias = size == 4 ? 127 : 1023
    var emax = size == 4 ? 255 : 2047
    var bits = []
    var q = 0
    while (q < size) {
        var by = b.charCodeAt(be ? p + size - 1 - q : p + q)
        var j = 0
        while (j < 8) {
            bits.push(by % 2)
            by = Math.floor(by / 2)
            j = j + 1
        }
        q = q + 1
    }
    var f = 0
    var i = mb - 1
    while (i >= 0) {
        f = f * 2 + bits[i]
        i = i - 1
    }
    var ex = 0
    i = size * 8 - 2
    while (i >= mb) {
        ex = ex * 2 + bits[i]
        i = i - 1
    }
    var zero = 0
    var v = 0
    if (ex == emax) { v = f == 0 ? 1 / zero : zero / zero }
    else if (ex == 0) { v = rbsScale2(f, 1 - bias - mb) }
    else { v = rbsScale2(rbsScale2(1, mb) + f, ex - bias - mb) }
    return bits[size * 8 - 1] == 1 ? -v : v
}

// The k little-endian bytes of the decimal integer text dec modulo 2^(8k), two's
// complement for a negative one.
function rbsDecBytes(dec, k) {
    var neg = dec.charAt(0) == "-"
    var digits = []
    var i = neg ? 1 : 0
    while (i < dec.length) {
        digits.push(dec.charCodeAt(i) - 48)
        i = i + 1
    }
    var out = []
    while (out.length < k) {
        var rem = 0
        var j = 0
        while (j < digits.length) {
            var cur = rem * 10 + digits[j]
            digits[j] = Math.floor(cur / 256)
            rem = cur % 256
            j = j + 1
        }
        out.push(rem)
    }
    if (neg) {
        var carry = 1
        var q = 0
        while (q < k) {
            var b = 255 - out[q] + carry
            out[q] = b % 256
            carry = b >= 256 ? 1 : 0
            q = q + 1
        }
    }
    return out
}

// The decimal text of the little-endian bytes b (two's complement when neg).
function rbsBytesDec(b, neg) {
    var bytes = []
    var i = 0
    while (i < b.length) {
        bytes.push(neg ? 255 - b[i] : b[i])
        i = i + 1
    }
    if (neg) {
        var carry = 1
        var q = 0
        while (q < bytes.length && carry > 0) {
            var s = bytes[q] + carry
            bytes[q] = s % 256
            carry = s >= 256 ? 1 : 0
            q = q + 1
        }
    }
    var out = ""
    var nonzero = true
    while (nonzero) {
        var rem = 0
        nonzero = false
        var j = bytes.length - 1
        while (j >= 0) {
            var cur = rem * 256 + bytes[j]
            bytes[j] = Math.floor(cur / 10)
            rem = cur % 10
            if (bytes[j] > 0) { nonzero = true }
            j = j - 1
        }
        out = String.fromCharCode(48 + rem) + out
    }
    return (neg ? "-" : "") + out
}

// The k bytes of an Integer, little-endian.
function rbsIntBytes(v, k) {
    if (typeof v != "number") { return rbsDecBytes(v.big, k) }
    var out = []
    var x = v
    if (k == 8) {
        var hi = Math.floor(x / 4294967296)
        var lo = x - hi * 4294967296
        out = rbsIntBytes(lo, 4)
        var hb = rbsIntBytes(hi, 4)
        var t = 0
        while (t < 4) {
            out.push(hb[t])
            t = t + 1
        }
        return out
    }
    var m = Math.pow(2, 8 * k)
    x = ((x % m) + m) % m
    var i = 0
    while (i < k) {
        out.push(x % 256)
        x = Math.floor(x / 256)
        i = i + 1
    }
    return out
}

function rbsB64Char(n) {
    return "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/".charAt(n)
}

// The base64 text of the bytes s: `m' wraps a line every `per' input bytes, and
// per 0 (`m0') writes no newline.
function rbsB64(s, per) {
    var out = ""
    var i = 0
    var line = 0
    while (i < s.length) {
        var b0 = s.charCodeAt(i)
        var b1 = i + 1 < s.length ? s.charCodeAt(i + 1) : 0
        var b2 = i + 2 < s.length ? s.charCodeAt(i + 2) : 0
        out = out + rbsB64Char(Math.floor(b0 / 4)) + rbsB64Char((b0 % 4) * 16 + Math.floor(b1 / 16))
        out = out + (i + 1 < s.length ? rbsB64Char((b1 % 16) * 4 + Math.floor(b2 / 64)) : "=")
        out = out + (i + 2 < s.length ? rbsB64Char(b2 % 64) : "=")
        i = i + 3
        line = line + 3
        if (per > 0 && (line >= per || i >= s.length)) {
            out = out + "\n"
            line = 0
        }
    }
    return out
}

// The bytes base64 text s spells; anything not in the alphabet is skipped.
function rbsUnB64(s) {
    var out = ""
    var acc = 0
    var bits = 0
    var i = 0
    while (i < s.length) {
        var c = s.charCodeAt(i)
        i = i + 1
        var v = 0 - 1
        if (c >= 65 && c <= 90) { v = c - 65 }
        else if (c >= 97 && c <= 122) { v = c - 71 }
        else if (c >= 48 && c <= 57) { v = c + 4 }
        else if (c == 43) { v = 62 }
        else if (c == 47) { v = 63 }
        else if (c == 61) { break }
        if (v < 0) { continue }
        acc = acc * 64 + v
        bits = bits + 6
        if (bits >= 8) {
            bits = bits - 8
            var p = Math.pow(2, bits)
            out = out + String.fromCharCode(Math.floor(acc / p))
            acc = acc % p
        }
    }
    return out
}

// The nibble value of a hex digit as MRI reads it for H / h.
function rbsNibble(c) {
    if ((c >= 65 && c <= 90) || (c >= 97 && c <= 122)) { return ((c % 8) + 9) % 16 }
    return c % 16
}

// Array#pack: the bytes, one char per byte.
function rbsPack(vals, dirs, err) {
    var out = ""
    var k = 0
    var i = 0
    while (i < dirs.length) {
        var d = dirs[i][0]
        var n = dirs[i][1]
        var size = dirs[i][2]
        if (d == "x") {
            out = out + rbsRepeatTo(String.fromCharCode(0), n < 0 ? 0 : n)
            i = i + 1
            continue
        }
        if (d == "X") {
            var back = n < 0 ? 0 : n
            if (back > out.length) {
                err.push("X outside of string")
                return out
            }
            out = out.substring(0, out.length - back)
            i = i + 1
            continue
        }
        if (d == "@") {
            var at = n < 0 ? 0 : n
            out = at <= out.length ? out.substring(0, at) : out + rbsRepeatTo(String.fromCharCode(0), at - out.length)
            i = i + 1
            continue
        }
        if (size == 0 && d != "U") {
            if (k >= vals.length) {
                err.push("too few arguments")
                return out
            }
            var sv = vals[k]
            k = k + 1
            if (sv === null || typeof sv != "object" || typeof sv.str != "string") {
                err.push("no implicit conversion of " + rbsPackCls(sv) + " into String")
                err.push("TypeError")
                return out
            }
            out = out + rbsPackStr(d, n, sv.str)
            i = i + 1
            continue
        }
        if (n < 0) { n = vals.length - k }
        var j = 0
        while (j < n) {
            if (k >= vals.length) {
                err.push("too few arguments")
                return out
            }
            if (rbsFloDir(d)) {
                var fv = vals[k]
                k = k + 1
                var fx = 0
                if (typeof fv == "number") { fx = fv }
                else if (fv !== null && typeof fv.flo == "number") { fx = fv.flo }
                else if (fv !== null && typeof fv.big == "string") { fx = parseFloat(fv.big) }
                else {
                    err.push("can't convert " + rbsPackCls(fv) + " into Float")
                    err.push("TypeError")
                    return out
                }
                var fb = rbsFloBytes(fx, size)
                var fq = 0
                while (fq < size) {
                    out = out + String.fromCharCode(fb[dirs[i][3] ? size - 1 - fq : fq])
                    fq = fq + 1
                }
                j = j + 1
                continue
            }
            var v = rbsPackIntVal(vals[k])
            k = k + 1
            if (v === null || (typeof v != "number" && typeof v.big != "string")) {
                err.push("no implicit conversion of " + rbsPackCls(v) + " into Integer")
                err.push("TypeError")
                return out
            }
            if (d == "U") {
                if (typeof v != "number" || v < 0 || v > 2147483647) {
                    err.push("pack(U): value out of range")
                    err.push("RangeError")
                    return out
                }
                out = out + rbsU8Enc(v)
            } else {
                var bs = rbsIntBytes(v, size)
                var q = 0
                while (q < size) {
                    out = out + String.fromCharCode(bs[dirs[i][3] ? size - 1 - q : q])
                    q = q + 1
                }
            }
            j = j + 1
        }
        i = i + 1
    }
    return out
}

// The encoding of pack's String: UTF-8 when U comes first, US-ASCII when only
// m does, else ASCII-8BIT (MRI's pack.c).
function rbsPackEnc(dirs) {
    var e = "US-ASCII"
    var i = 0
    while (i < dirs.length) {
        var d = dirs[i][0]
        if (d == "U") {
            if (e == "US-ASCII") { e = "UTF-8" }
        } else if (d != "m") { e = "ASCII-8BIT" }
        i = i + 1
    }
    return e
}

// The bytes pack(U) writes for cp: UTF-8's scheme, extended to six bytes up to
// 2^31 - 1 as MRI does, so a surrogate or a value past U+10FFFF is written too.
function rbsU8Enc(cp) {
    if (cp < 128) { return String.fromCharCode(cp) }
    var n = 6
    var lead = 252
    if (cp < 2048) { n = 2; lead = 192 }
    else if (cp < 65536) { n = 3; lead = 224 }
    else if (cp < 2097152) { n = 4; lead = 240 }
    else if (cp < 67108864) { n = 5; lead = 248 }
    var tail = ""
    var v = cp
    var k = 1
    while (k < n) {
        tail = String.fromCharCode(128 + v % 64) + tail
        v = Math.floor(v / 64)
        k = k + 1
    }
    return String.fromCharCode(lead + v) + tail
}

// One String directive of pack over the bytes s.
function rbsPackStr(d, n, s) {
    if (d == "a" || d == "A" || d == "Z") {
        if (n < 0) { return d == "Z" ? s + String.fromCharCode(0) : s }
        if (s.length >= n) { return s.substring(0, n) }
        return s + rbsRepeatTo(d == "A" ? " " : String.fromCharCode(0), n - s.length)
    }
    if (d == "m") { return rbsB64(s, n == 0 ? 0 : (n < 3 ? 45 : n - n % 3)) }
    var out = ""
    var cnt = n < 0 ? s.length : n
    if (d == "B" || d == "b") {
        var acc = 0
        var bits = 0
        var i = 0
        while (i < cnt) {
            var bit = i < s.length ? s.charCodeAt(i) % 2 : 0
            acc = d == "B" ? acc * 2 + bit : acc + bit * Math.pow(2, bits)
            bits = bits + 1
            if (bits == 8) {
                out = out + String.fromCharCode(acc)
                acc = 0
                bits = 0
            }
            i = i + 1
        }
        if (bits > 0) { out = out + String.fromCharCode(d == "B" ? acc * Math.pow(2, 8 - bits) : acc) }
        return out
    }
    var hacc = 0
    var half = false
    var h = 0
    while (h < cnt) {
        var nib = h < s.length ? rbsNibble(s.charCodeAt(h)) : 0
        if (!half) { hacc = d == "H" ? nib * 16 : nib }
        else {
            hacc = d == "H" ? hacc + nib : hacc + nib * 16
            out = out + String.fromCharCode(hacc)
        }
        half = !half
        h = h + 1
    }
    if (half) { out = out + String.fromCharCode(hacc) }
    return out
}

// String#unpack of the bytes b (one char per byte).
function rbsUnpack(b, dirs, err) {
    var out = []
    var p = 0
    var i = 0
    while (i < dirs.length) {
        var d = dirs[i][0]
        var n = dirs[i][1]
        var size = dirs[i][2]
        if (d == "x" || d == "X" || d == "@") {
            var to = d == "@" ? (n < 0 ? 0 : n) : (d == "x" ? p + (n < 0 ? b.length - p : n) : p - (n < 0 ? 0 : n))
            if (to < 0 || to > b.length) {
                err.push(d + " outside of string")
                return out
            }
            p = to
        } else if (size == 0 && d != "U") {
            var r = rbsUnpackStr(d, n, b, p)
            out.push(r[0])
            p = r[1]
        } else {
            var j = 0
            while (n < 0 ? p + (d == "U" ? 1 : size) <= b.length : j < n) {
                if (d == "U") {
                    if (p >= b.length) { break }
                    var c = b.charCodeAt(p)
                    var len = c < 128 ? 1 : (c >= 240 ? 4 : (c >= 224 ? 3 : 2))
                    var t = rbsBinDecode(b.substring(p, p + len))
                    if (t === null) {
                        err.push("malformed UTF-8 character")
                        return out
                    }
                    out.push(rbsOrd(t))
                    p = p + len
                } else if (p + size > b.length) {
                    out.push(null)
                    p = b.length
                } else if (rbsFloDir(d)) {
                    out.push({flo: rbsFloOf(b, p, size, dirs[i][3])})
                    p = p + size
                } else {
                    out.push(rbsIntOf(b, p, size, dirs[i][3], dirs[i][4]))
                    p = p + size
                }
                j = j + 1
            }
        }
        i = i + 1
    }
    return out
}

// The integer of the size bytes of b at p.
function rbsIntOf(b, p, size, be, signed) {
    var le = []
    var q = 0
    while (q < size) {
        le.push(b.charCodeAt(be ? p + size - 1 - q : p + q))
        q = q + 1
    }
    var neg = signed && le[size - 1] >= 128
    if (size == 8) {
        var lo = le[0] + le[1] * 256 + le[2] * 65536 + le[3] * 16777216
        var hi = le[4] + le[5] * 256 + le[6] * 65536 + le[7] * 16777216
        if (neg) { hi = hi - 4294967296 }
        if (hi < 2097152 && hi >= 0 - 2097152) { return hi * 4294967296 + lo }
        return {big: rbsBytesDec(le, neg)}
    }
    var v = 0
    var k = size - 1
    while (k >= 0) {
        v = v * 256 + le[k]
        k = k - 1
    }
    return neg ? v - Math.pow(2, 8 * size) : v
}

// One String directive of unpack at byte p: [the String, the next position].
function rbsUnpackStr(d, n, b, p) {
    var rest = b.length - p
    if (d == "a" || d == "A" || d == "Z") {
        var cnt = n < 0 || n > rest ? rest : n
        var s = b.substring(p, p + cnt)
        var next = p + cnt
        if (d == "Z") {
            var z = s.indexOf(String.fromCharCode(0))
            if (z >= 0) {
                s = s.substring(0, z)
                if (n < 0) { next = p + z + 1 }
            }
        } else if (d == "A") {
            var e = s.length
            while (e > 0 && (s.charCodeAt(e - 1) == 32 || s.charCodeAt(e - 1) == 0)) { e = e - 1 }
            s = s.substring(0, e)
        }
        return [{str: s, enc: "ASCII-8BIT"}, next]
    }
    if (d == "m") { return [{str: rbsUnB64(b.substring(p, b.length)), enc: "ASCII-8BIT"}, b.length] }
    var out = ""
    if (d == "B" || d == "b") {
        var bits = n < 0 || n > rest * 8 ? rest * 8 : n
        var i = 0
        while (i < bits) {
            var by = b.charCodeAt(p + Math.floor(i / 8))
            var sh = d == "B" ? 7 - i % 8 : i % 8
            out = out + (Math.floor(by / Math.pow(2, sh)) % 2 == 1 ? "1" : "0")
            i = i + 1
        }
        return [{str: out, enc: "US-ASCII"}, p + Math.floor((bits + 7) / 8)]
    }
    var hx = "0123456789abcdef"
    var nibs = n < 0 || n > rest * 2 ? rest * 2 : n
    var h = 0
    while (h < nibs) {
        var hb = b.charCodeAt(p + Math.floor(h / 2))
        var hi = Math.floor(hb / 16)
        var lo = hb % 16
        out = out + hx.charAt((h % 2 == 0) == (d == "H") ? hi : lo)
        h = h + 1
    }
    return [{str: out, enc: "US-ASCII"}, p + Math.floor((nibs + 1) / 2)]
}
