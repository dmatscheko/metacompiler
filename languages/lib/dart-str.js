// dart-str.js -- dart:core's String members, the RegExp Pattern and Match, and the
// List members that travel with them.
//
// Written once for the interpreter half (dart-interpreter.abnf includes it) and for
// layer 2 (languages/lib/dart-rt.metajs imports it), in the restricted subset both
// accept - see the head of regex.js. abnf/jsrtdart.go's dartStrMember is the Go twin
// the compiler half runs under llvm.Run. There is no dart toolchain on this machine:
// every rule below is dart:core's documentation of String, RegExp, Match and List.
//
// A Dart String is a sequence of UTF-16 code units, and every index, length and
// position here counts them. A Pattern argument is a String or a RegExp; any other
// receiver or argument shape answers a miss, and the includer's own dispatch goes on.
// A RegExp runs on lib/regex.js, which the includer brings in too.
//
// Hooks the includer defines:
//   dsHNum(v)          an int argument as a plain number
//   dsHIndexErr(i, s)  throw dart's IndexError for index i of s (does not return)
//   dsHRangeErr(v, lo, hi, name)  throw RangeError.range(v, lo, hi, name), or
//                      RangeError.value(v, name) when lo is null (does not return)
//   dsHCall(f, argv)   call the closure f
//   dsHStr(v)          v's toString()
//   dsHEq(a, b)        Dart's a == b
//   dsHCmp(a, b)       a.compareTo(b), as a plain number
//   dsHThrow(name, argv)  throw dart:core's error `name` built from argv (does not
//                      return)

// dsMember answers {hit: true, v} for a String member dart:core declares, and
// {hit: false} for any other name.
function dsMember(s, name, args) {
    var n = s.length
    var miss = {hit: false, v: undefined}
    if (name == "codeUnitAt") {
        var ci = dsHNum(args[0])
        if (ci < 0 || ci >= n) { dsHIndexErr(args[0], s) }
        return {hit: true, v: s.charCodeAt(ci)}
    }
    if (name == "codeUnits") { return {hit: true, v: dsUnits(s)} }
    if (name == "runes") { return {hit: true, v: dsRunes(s)} }
    if (name == "toUpperCase") { return {hit: true, v: dsCase(s, true)} }
    if (name == "toLowerCase") { return {hit: true, v: dsCase(s, false)} }
    if (name == "trim") { return {hit: true, v: dsTrim(s, true, true)} }
    if (name == "trimLeft") { return {hit: true, v: dsTrim(s, true, false)} }
    if (name == "trimRight") { return {hit: true, v: dsTrim(s, false, true)} }
    if (name == "padLeft" || name == "padRight") {
        var pad = " "
        if (args.length > 1 && typeof args[1] == "string") { pad = args[1] }
        var delta = dsHNum(args[0]) - n
        if (delta <= 0) { return {hit: true, v: s} }
        if (name == "padLeft") { return {hit: true, v: dsRepeat(pad, delta) + s} }
        return {hit: true, v: s + dsRepeat(pad, delta)}
    }
    if (name == "replaceRange") {
        var rs = dsHNum(args[0])
        var re = n
        if (args.length > 1 && args[1] !== null && args[1] !== undefined) { re = dsHNum(args[1]) }
        return {hit: true, v: s.substring(0, rs) + args[2] + s.substring(re, n)}
    }
    if (name == "compareTo") {
        if (typeof args[0] != "string") { return miss }
        return {hit: true, v: dsCompare(s, args[0])}
    }
    // Everything below takes a Pattern first.
    if (args.length > 0 && typeof args[0] == "object" && dsIsRegExp(args[0])) { return dsReStrMember(s, name, args) }
    if (args.length == 0 || typeof args[0] != "string") { return miss }
    var p = args[0]
    if (name == "indexOf") { return {hit: true, v: dsFind(s, p, dsOpt(args, 1, 0))} }
    if (name == "lastIndexOf") { return {hit: true, v: dsFindLast(s, p, dsOpt(args, 1, n))} }
    if (name == "contains") { return {hit: true, v: dsFind(s, p, dsOpt(args, 1, 0)) >= 0} }
    if (name == "startsWith") { return {hit: true, v: dsAt(s, dsOpt(args, 1, 0), p)} }
    if (name == "endsWith") { return {hit: true, v: dsAt(s, n - p.length, p)} }
    if (name == "split") { return {hit: true, v: dsSplit(s, p)} }
    if (name == "replaceAll") { return {hit: true, v: dsReplaceAll(s, p, args[1])} }
    // The receiver is the PATTERN of these two (String implements Pattern).
    if (name == "allMatches") {
        var a0 = dsOpt(args, 1, 0)
        if (a0 < 0 || a0 > p.length) { dsHRangeErr(a0, 0, p.length, null) }
        return {hit: true, v: dsAllMatches(s, p, a0)}
    }
    if (name == "matchAsPrefix") {
        var ms = dsOpt(args, 1, 0)
        if (ms < 0 || ms > p.length) { dsHRangeErr(ms, 0, p.length, null) }
        if (!dsAt(p, ms, s)) { return {hit: true, v: null} }
        return {hit: true, v: dsMatch(ms, p, s)}
    }
    if (name == "replaceAllMapped" || name == "splitMapJoin") {
        return {hit: true, v: dsMapJoin(s, dsAllMatches(p, s, 0), name, args)}
    }
    if (name == "replaceFirst") {
        var fi = dsFind(s, p, dsOpt(args, 2, 0))
        if (fi < 0) { return {hit: true, v: s} }
        return {hit: true, v: s.substring(0, fi) + args[1] + s.substring(fi + p.length, n)}
    }
    return miss
}

// replaceAllMapped(p, f) and splitMapJoin(p, onMatch:, onNonMatch:) over the
// matches ms of either kind of Pattern: the text between matches is kept (or
// mapped by onNonMatch) and each match replaced by its callback's answer (by its
// own text when splitMapJoin has no onMatch).
function dsMapJoin(s, ms, name, args) {
    var onM = anytype
    var onN = anytype
    onM = args[1]
    onN = null
    if (name == "splitMapJoin") {
        onM = dsNamed(args, "onMatch")
        onN = dsNamed(args, "onNonMatch")
    }
    var out = ""
    var at = 0
    var i = 0
    while (i < ms.length) {
        var gap = s.substring(at, ms[i]["start"])
        if (onN !== null) { gap = dsHStr(dsHCall(onN, [gap])) }
        var mt = dsMatchGroup(ms[i], 0)
        if (onM !== null) { mt = dsHStr(dsHCall(onM, [ms[i]])) }
        out = out + gap + mt
        at = ms[i]["end"]
        i = i + 1
    }
    var tail = s.substring(at, s.length)
    if (onN !== null) { tail = dsHStr(dsHCall(onN, [tail])) }
    return out + tail
}

// An optional positional int argument, or its default when absent or null.
function dsOpt(args, i, dflt) {
    if (args.length > i && args[i] !== null && args[i] !== undefined) { return dsHNum(args[i]) }
    return dflt
}

// Does p occur in s at code unit i?
function dsAt(s, i, p) {
    if (i < 0 || i + p.length > s.length) { return false }
    return s.substring(i, i + p.length) == p
}

// The first match of p at or after from, or -1 (String.indexOf).
function dsFind(s, p, from) {
    var i = from
    if (i < 0) { i = 0 }
    while (i + p.length <= s.length) {
        if (dsAt(s, i, p)) { return i }
        i = i + 1
    }
    return 0 - 1
}

// The last match of p at or before from, or -1 (String.lastIndexOf).
function dsFindLast(s, p, from) {
    var i = from
    if (i > s.length - p.length) { i = s.length - p.length }
    while (i >= 0) {
        if (dsAt(s, i, p)) { return i }
        i = i - 1
    }
    return 0 - 1
}

// String.split with a String pattern: every field, empty ones included. The empty
// pattern splits between code units, and splits an empty string into no fields at
// all; any other pattern leaves an empty string one empty field.
function dsSplit(s, p) {
    var out = []
    var i = 0
    if (p.length == 0) {
        while (i < s.length) {
            out.push(s.substring(i, i + 1))
            i = i + 1
        }
        return out
    }
    var at = dsFind(s, p, 0)
    while (at >= 0) {
        out.push(s.substring(i, at))
        i = at + p.length
        at = dsFind(s, p, i)
    }
    out.push(s.substring(i, s.length))
    return out
}

// String.replaceAll with a String pattern. The empty pattern matches at every code
// unit boundary, both ends included: "ab".replaceAll("", "-") is "-a-b-".
function dsReplaceAll(s, p, rep) {
    var out = ""
    var i = 0
    if (p.length == 0) {
        out = rep
        while (i < s.length) {
            out = out + s.substring(i, i + 1) + rep
            i = i + 1
        }
        return out
    }
    var at = dsFind(s, p, 0)
    while (at >= 0) {
        out = out + s.substring(i, at) + rep
        i = at + p.length
        at = dsFind(s, p, i)
    }
    return out + s.substring(i, s.length)
}

// count copies of s, back to back.
function dsRepeat(s, count) {
    var out = ""
    var i = 0
    while (i < count) {
        out = out + s
        i = i + 1
    }
    return out
}

// String.compareTo: code unit by code unit, then the shorter first; -1, 0 or 1.
function dsCompare(a, b) {
    var i = 0
    while (i < a.length && i < b.length) {
        var x = a.charCodeAt(i)
        var y = b.charCodeAt(i)
        if (x < y) { return 0 - 1 }
        if (x > y) { return 1 }
        i = i + 1
    }
    if (a.length < b.length) { return 0 - 1 }
    if (a.length > b.length) { return 1 }
    return 0
}

// White space as String.trim reads it: the Unicode White_Space characters and the
// BOM, U+FEFF.
function dsIsWs(c) {
    if (c == 32 || (c >= 9 && c <= 13)) { return true }
    if (c == 133 || c == 160 || c == 5760) { return true }
    if (c >= 8192 && c <= 8202) { return true }
    return c == 8232 || c == 8233 || c == 8239 || c == 8287 || c == 12288 || c == 65279
}

function dsTrim(s, left, right) {
    var a = 0
    var b = s.length
    if (left) {
        while (a < b && dsIsWs(s.charCodeAt(a))) { a = a + 1 }
    }
    if (right) {
        while (b > a && dsIsWs(s.charCodeAt(b - 1))) { b = b - 1 }
    }
    return s.substring(a, b)
}

// double.parse's reading of s, or null where it throws FormatException: trimmed
// as by String.trim, an optional sign, then "Infinity", "NaN" or a decimal
// floating-point literal - digits with an optional "." and digits, or "." and
// digits, then an optional e[+-]digits - or, as the documentation's '0xFF'
// example has it, an unsigned hexadecimal integer literal. The includer brings
// lib/hex-float.js.
function dsParseDouble(s) {
    var t = dsTrim(s, true, true)
    var p = 0
    var neg = false
    if (t.length > 0 && (t.charAt(0) == "+" || t.charAt(0) == "-")) {
        neg = t.charAt(0) == "-"
        p = 1
    }
    var body = t.substring(p, t.length)
    if (body == "Infinity") { return neg ? -1 / 0 : 1 / 0 }
    if (body == "NaN") { return 0 / 0 }
    if (p == 0 && body.length > 2 && body.charAt(0) == "0" && (body.charAt(1) == "x" || body.charAt(1) == "X")) {
        var h = anytype
        h = hexFloatScan(body, 2)
        if (h === null || h["end"] != body.length || body.indexOf(".") >= 0 || body.indexOf("p") >= 0 || body.indexOf("P") >= 0) { return null }
        return h["v"]
    }
    var q = 0
    var nd = 0
    while (q < body.length && body.charAt(q) >= "0" && body.charAt(q) <= "9") {
        q = q + 1
        nd = nd + 1
    }
    if (q < body.length && body.charAt(q) == ".") {
        q = q + 1
        while (q < body.length && body.charAt(q) >= "0" && body.charAt(q) <= "9") {
            q = q + 1
            nd = nd + 1
        }
    }
    if (nd == 0) { return null }
    if (q < body.length && (body.charAt(q) == "e" || body.charAt(q) == "E")) {
        q = q + 1
        if (q < body.length && (body.charAt(q) == "+" || body.charAt(q) == "-")) { q = q + 1 }
        var e0 = q
        while (q < body.length && body.charAt(q) >= "0" && body.charAt(q) <= "9") { q = q + 1 }
        if (q == e0) { return null }
    }
    if (q != body.length) { return null }
    var v = parseFloat(body)
    return neg ? -v : v
}

// The width, in code units, of the code point starting at i.
function dsCpWidth(s, i) {
    var c = s.charCodeAt(i)
    if (c >= 55296 && c <= 56319 && i + 1 < s.length) {
        var d = s.charCodeAt(i + 1)
        if (d >= 56320 && d <= 57343) { return 2 }
    }
    return 1
}

// toUpperCase / toLowerCase: the SIMPLE case mapping, code point by code point
// (unicodeCase modes 1 and 0) - a character whose full mapping is several
// characters ("ß" -> "SS") keeps its own case. Twin of dartStrCase in
// abnf/jsrtdart.go (unicode.ToUpper / ToLower).
function dsCase(s, up) { return unicodeCase(s, up ? 1 : 0) }

// A named argument: a call's named arguments arrive as one trailing Map. null when
// the call has none of that name.
function dsNamed(args, key) {
    if (args.length == 0) { return null }
    var d = args[args.length - 1]
    if (typeof d != "object" || d === null || Array.isArray(d) || d["__dict"] !== true) { return null }
    var ks = d["keys"]
    var i = 0
    while (i < ks.length) {
        if (ks[i] === key) { return d["vals"][i] }
        i = i + 1
    }
    return null
}

// The Match of the String pattern p at code unit st of input - what allMatches,
// matchAsPrefix and the replaceAllMapped / splitMapJoin callbacks see. A String
// pattern has no capture groups.
function dsMatch(st, input, p) {
    return {__dmatch: true, start: st, end: st + p.length, input: input, pattern: p, groupCount: 0}
}
function dsIsMatch(v) { return typeof v == "object" && v !== null && !Array.isArray(v) && v["__dmatch"] === true }

// Pattern.allMatches for the String pattern p over s from start: non-overlapping,
// left to right, and an empty match moves one code unit on (dart:core's
// _StringAllMatchesIterator).
function dsAllMatches(p, s, start) {
    var out = []
    var i = start
    while (i + p.length <= s.length) {
        var at = dsFind(s, p, i)
        if (at < 0) { return out }
        out.push(dsMatch(at, s, p))
        var e = at + p.length
        if (e == i) { e = e + 1 }
        i = e
    }
    return out
}

// A Match's members: group(i) / [i] is the text of group i (0 the whole match, null
// for a group that took no part) and any group past groupCount a RangeError;
// groups(indexes) is the list of those groups and namedGroup(name) a named one.
// {hit: false} for any other name (start, end, input, pattern and groupCount are
// fields).
function dsMatchMember(m, name, args) {
    if (name == "group" || name == "[]") { return {hit: true, v: dsMatchGroup(m, args[0])} }
    if (name == "namedGroup" && m["caps"] !== undefined) {
        var gi = rxIndexOfName(rxObjRe(m["pattern"]), dsHStr(args[0]))
        if (gi < 0) { dsHThrow("ArgumentError", [args[0], "Not a capture group name"]) }
        return {hit: true, v: dsMatchGroup(m, gi)}
    }
    if (name == "groups") {
        var gs = args[0]
        var out = []
        var i = 0
        while (i < gs.length) {
            out.push(dsMatchGroup(m, gs[i]))
            i = i + 1
        }
        return {hit: true, v: out}
    }
    return {hit: false, v: undefined}
}
function dsMatchGroup(m, g) {
    var gi = dsHNum(g)
    var caps = m["caps"]
    if (caps === undefined) {
        if (gi != 0) { dsHRangeErr(g, null, null, null) }
        return m["input"].substring(m["start"], m["end"])
    }
    if (gi < 0 || gi > m["groupCount"]) { dsHRangeErr(g, null, null, null) }
    var from = caps[2 * gi]
    if (from < 0) { return null }
    return m["input"].substring(from, caps[2 * gi + 1])
}

// ----- RegExp ----------------------------------------------------------------
// dart:core's RegExp has "the same syntax and semantics as JavaScript regular
// expressions", so lib/regex.js runs it with JavaScript's repeat semantics (j) and
// Annex B escapes (o); caseSensitive: false, multiLine and dotAll add i, m and s.
// The object carries regex.js's shared __rxsrc / __rxflags slots beside the
// getters dart declares. A bad pattern is a FormatException. Twin: dartRegExp in
// abnf/jsrtdart.go.
function dsIsRegExp(v) {
    return typeof v == "object" && v !== null && !Array.isArray(v) && v["__dregexp"] === true
}
function dsRegExp(args) {
    var src = dsHStr(args[0])
    var ml = dsNamed(args, "multiLine") === true
    var cs = dsNamed(args, "caseSensitive") !== false
    var da = dsNamed(args, "dotAll") === true
    var fl = "jo"
    if (!cs) { fl = fl + "i" }
    if (ml) { fl = fl + "m" }
    if (da) { fl = fl + "s" }
    var re = rxGet(src, fl)
    if (!re.ok) { dsHThrow("FormatException", [re.err, src]) }
    var o = {__dregexp: true, __rxsrc: src, __rxflags: fl, pattern: src}
    o["isMultiLine"] = ml
    o["isCaseSensitive"] = cs
    o["isUnicode"] = dsNamed(args, "unicode") === true
    o["isDotAll"] = da
    return o
}
// The Match of regex.js's match m of the RegExp rx over s: its captures ride along
// for group(i).
function dsReMatch(rx, s, m) {
    var caps = []
    var i = 0
    while (i < m.caps.length) {
        caps.push(m.caps[i])
        i = i + 1
    }
    return {__dmatch: true, start: m.begin, end: m.end, input: s, pattern: rx,
            groupCount: rxGroupCount(rxObjRe(rx)), caps: caps}
}
// RegExp.allMatches over s from start: left to right, a search resuming at the end
// of each match and one code unit past an empty one.
function dsReAll(rx, s, start) {
    var re = rxObjRe(rx)
    var out = []
    var at = start
    while (at <= s.length) {
        var m = rxSearch(re, s, at)
        if (m === null) { return out }
        out.push(dsReMatch(rx, s, m))
        at = m.end
        if (m.end == m.begin) { at = at + 1 }
    }
    return out
}
// A RegExp's own members; {hit: false} for any other name (pattern, isMultiLine,
// isCaseSensitive, isUnicode and isDotAll are fields).
function dsReMember(rx, name, args) {
    var re = rxObjRe(rx)
    var miss = {hit: false, v: undefined}
    if (args.length == 0 || typeof args[0] != "string") { return miss }
    var s = args[0]
    if (name == "hasMatch") { return {hit: true, v: rxSearch(re, s, 0) !== null} }
    if (name == "firstMatch" || name == "stringMatch") {
        var m = rxSearch(re, s, 0)
        if (m === null) { return {hit: true, v: null} }
        if (name == "stringMatch") { return {hit: true, v: s.substring(m.begin, m.end)} }
        return {hit: true, v: dsReMatch(rx, s, m)}
    }
    if (name == "allMatches" || name == "matchAsPrefix") {
        var st = dsOpt(args, 1, 0)
        if (st < 0 || st > s.length) { dsHRangeErr(st, 0, s.length, null) }
        if (name == "allMatches") { return {hit: true, v: dsReAll(rx, s, st)} }
        var pm = rxMatchAt(re, s, st)
        if (pm === null) { return {hit: true, v: null} }
        return {hit: true, v: dsReMatch(rx, s, pm)}
    }
    return miss
}
// The String members whose Pattern is a RegExp. replaceAll's replacement is NOT
// interpreted (dart: "the replace string is not interpreted"; replaceAllMapped is
// the way to use a group).
function dsReStrMember(s, name, args) {
    var rx = args[0]
    var re = rxObjRe(rx)
    var n = s.length
    if (name == "contains" || name == "indexOf") {
        var sm = rxSearch(re, s, dsOpt(args, 1, 0))
        if (name == "contains") { return {hit: true, v: sm !== null} }
        return {hit: true, v: sm === null ? 0 - 1 : sm.begin}
    }
    if (name == "startsWith") {
        var at = dsOpt(args, 1, 0)
        if (at < 0 || at > n) { dsHRangeErr(at, 0, n, null) }
        return {hit: true, v: rxMatchAt(re, s, at) !== null}
    }
    if (name == "split") { return {hit: true, v: dsReSplit(s, dsReAll(rx, s, 0))} }
    if (name == "replaceAll" || name == "replaceFirst") {
        var ms = dsReAll(rx, s, name == "replaceFirst" ? dsOpt(args, 2, 0) : 0)
        var rep = dsHStr(args[1])
        var out = ""
        var pos = 0
        var i = 0
        while (i < ms.length) {
            out = out + s.substring(pos, ms[i]["start"]) + rep
            pos = ms[i]["end"]
            i = i + 1
            if (name == "replaceFirst") { i = ms.length }
        }
        return {hit: true, v: out + s.substring(pos, n)}
    }
    if (name == "replaceAllMapped" || name == "splitMapJoin") {
        return {hit: true, v: dsMapJoin(s, dsReAll(rx, s, 0), name, args)}
    }
    return {hit: false, v: undefined}
}
// String.split by the matches ms (dart:core's _StringBase.split): the pieces
// between them, an empty match at the very start or end of a piece cutting
// nothing, a match at the end of the string ending the split, and an empty string
// that the pattern matches splitting into no pieces at all.
function dsReSplit(s, ms) {
    var n = s.length
    var out = []
    if (n == 0 && ms.length > 0) { return out }
    var start = 0
    var prev = 0
    var i = 0
    while (true) {
        if (start == n || i >= ms.length) {
            out.push(s.substring(prev, n))
            return out
        }
        var m = ms[i]
        i = i + 1
        if (m["start"] == n) {
            out.push(s.substring(prev, n))
            return out
        }
        var e = m["end"]
        if (start == e && e == prev) {
            start = start + 1
        } else {
            out.push(s.substring(prev, m["start"]))
            start = e
            prev = e
        }
    }
    return out
}

// ----- List ------------------------------------------------------------------
// dart:core's List members and the Iterable members a List inherits. Those that
// answer an Iterable in dart:core (map, where, reversed, skip, take, expand, ...;
// printed in parentheses there) answer a List here, eagerly - the same trade runes
// and allMatches make. Element equality is the includer's Dart ==. {hit: false} for
// any other name. Twin: dartListMember in abnf/jsrtdart.go.
function dsListMember(o, name, args) {
    var out = []
    var n = o.length
    var i = 0
    var r = anytype
    if (name == "map" || name == "where" || name == "expand") {
        while (i < n) {
            r = dsHCall(args[0], [o[i]])
            if (name == "map") { out.push(r) }
            else if (name == "expand") { dsPushAll(out, r) }
            else if (r === true) { out.push(o[i]) }
            i = i + 1
        }
        return {hit: true, v: out}
    }
    if (name == "toList" || name == "reversed") {
        while (i < n) {
            out.push(name == "toList" ? o[i] : o[n - 1 - i])
            i = i + 1
        }
        return {hit: true, v: out}
    }
    if (name == "first" || name == "last" || name == "single") {
        if (n == 0) { dsHThrow("StateError", ["No element"]) }
        if (name == "single" && n > 1) { dsHThrow("StateError", ["Too many elements"]) }
        return {hit: true, v: name == "last" ? o[n - 1] : o[0]}
    }
    if (name == "join") {
        var sep = ""
        if (args.length > 0 && typeof args[0] == "string") { sep = args[0] }
        var js = ""
        while (i < n) {
            if (i > 0) { js = js + sep }
            js = js + dsHStr(o[i])
            i = i + 1
        }
        return {hit: true, v: js}
    }
    if (name == "contains") { return {hit: true, v: dsListFind(o, args[0], 0) >= 0} }
    if (name == "indexOf") { return {hit: true, v: dsListFind(o, args[0], dsOpt(args, 1, 0))} }
    if (name == "lastIndexOf") {
        var li = dsLastFrom(n, args)
        while (li >= 0 && !dsHEq(o[li], args[0])) { li = li - 1 }
        return {hit: true, v: li}
    }
    if (name == "indexWhere") {
        i = dsOpt(args, 1, 0)
        if (i < 0) { i = 0 }
        while (i < n && dsHCall(args[0], [o[i]]) !== true) { i = i + 1 }
        return {hit: true, v: i < n ? i : 0 - 1}
    }
    if (name == "lastIndexWhere") {
        var wi = dsLastFrom(n, args)
        while (wi >= 0 && dsHCall(args[0], [o[wi]]) !== true) { wi = wi - 1 }
        return {hit: true, v: wi}
    }
    if (name == "any" || name == "every") {
        var want = name == "any"
        while (i < n) {
            if ((dsHCall(args[0], [o[i]]) === true) === want) { return {hit: true, v: want} }
            i = i + 1
        }
        return {hit: true, v: !want}
    }
    if (name == "forEach") {
        while (i < n) {
            dsHCall(args[0], [o[i]])
            i = i + 1
        }
        return {hit: true, v: null}
    }
    if (name == "fold" || name == "reduce") {
        var acc = anytype
        if (name == "fold") { acc = args[0] } else {
            if (n == 0) { dsHThrow("StateError", ["No element"]) }
            acc = o[0]
            i = 1
        }
        var f = args[args.length - 1]
        while (i < n) {
            acc = dsHCall(f, [acc, o[i]])
            i = i + 1
        }
        return {hit: true, v: acc}
    }
    if (name == "firstWhere" || name == "lastWhere" || name == "singleWhere") {
        var found = 0
        var fv = anytype
        fv = null
        while (i < n) {
            var e = anytype
            e = name == "lastWhere" ? o[n - 1 - i] : o[i]
            if (dsHCall(args[0], [e]) === true) {
                if (found > 0) { dsHThrow("StateError", ["Too many elements"]) }
                fv = e
                found = 1
                if (name != "singleWhere") { i = n }
            }
            i = i + 1
        }
        if (found > 0) { return {hit: true, v: fv} }
        var orElse = dsNamed(args, "orElse")
        if (orElse !== null) { return {hit: true, v: dsHCall(orElse, [])} }
        dsHThrow("StateError", ["No element"])
    }
    if (name == "elementAt") {
        var ei = dsHNum(args[0])
        if (ei < 0 || ei >= n) { dsHIndexErr(args[0], o) }
        return {hit: true, v: o[ei]}
    }
    if (name == "sublist" || name == "getRange") {
        var se = dsRangeEnd(o, args, name == "sublist")
        i = dsHNum(args[0])
        while (i < se) {
            out.push(o[i])
            i = i + 1
        }
        return {hit: true, v: out}
    }
    if (name == "skip" || name == "take" || name == "skipWhile" || name == "takeWhile") {
        var cut = 0
        if (name == "skip" || name == "take") {
            cut = dsHNum(args[0])
            if (cut < 0) { dsHRangeErr(cut, 0, null, "count") }
            if (cut > n) { cut = n }
        } else {
            while (cut < n && dsHCall(args[0], [o[cut]]) === true) { cut = cut + 1 }
        }
        if (name == "skip" || name == "skipWhile") { i = cut } else { n = cut }
        while (i < n) {
            out.push(o[i])
            i = i + 1
        }
        return {hit: true, v: out}
    }
    if (name == "followedBy") {
        dsPushAll(out, o)
        dsPushAll(out, args[0])
        return {hit: true, v: out}
    }
    if (name == "sort") {
        var cmp = anytype
        cmp = null
        if (args.length > 0) { cmp = args[0] }
        dsSort(o, cmp)
        return {hit: true, v: null}
    }
    return dsListMutate(o, name, args)
}

// The List members that change its length or contents in place.
function dsListMutate(o, name, args) {
    var n = o.length
    var i = 0
    if (name == "addAll") {
        dsPushAll(o, dsCopy(args[0]))
        return {hit: true, v: null}
    }
    if (name == "insert" || name == "insertAll") {
        var at = dsHNum(args[0])
        if (at < 0 || at > n) { dsHRangeErr(at, 0, n, "index") }
        var ins = []
        if (name == "insert") { ins.push(args[1]) } else { dsPushAll(ins, args[1]) }
        dsSplice(o, at, at, ins)
        return {hit: true, v: null}
    }
    if (name == "remove") {
        var ri = dsListFind(o, args[0], 0)
        if (ri < 0) { return {hit: true, v: false} }
        dsSplice(o, ri, ri + 1, [])
        return {hit: true, v: true}
    }
    if (name == "removeAt") {
        var ai = dsHNum(args[0])
        if (ai < 0 || ai >= n) { dsHIndexErr(args[0], o) }
        var rv = anytype
        rv = o[ai]
        dsSplice(o, ai, ai + 1, [])
        return {hit: true, v: rv}
    }
    if (name == "removeWhere" || name == "retainWhere") {
        var keep = []
        while (i < n) {
            if ((dsHCall(args[0], [o[i]]) === true) === (name == "retainWhere")) { keep.push(o[i]) }
            i = i + 1
        }
        dsSplice(o, 0, n, keep)
        return {hit: true, v: null}
    }
    if (name == "clear") {
        o.length = 0
        return {hit: true, v: null}
    }
    if (name == "removeRange" || name == "replaceRange") {
        var re = dsRangeEnd(o, args, false)
        var repl = []
        if (name == "replaceRange") { dsPushAll(repl, args[2]) }
        dsSplice(o, dsHNum(args[0]), re, repl)
        return {hit: true, v: null}
    }
    if (name == "fillRange" || name == "setRange") {
        var fe = dsRangeEnd(o, args, false)
        i = dsHNum(args[0])
        var src = []
        var skip = 0
        if (name == "setRange") {
            dsPushAll(src, args[2])
            skip = dsOpt(args, 3, 0)
            if (skip < 0) { dsHRangeErr(skip, 0, null, "skipCount") }
            if (src.length - skip < fe - i) { dsHThrow("StateError", ["Too few elements"]) }
        }
        var k = skip
        while (i < fe) {
            if (name == "setRange") { o[i] = src[k] } else { o[i] = args.length > 2 ? args[2] : null }
            i = i + 1
            k = k + 1
        }
        return {hit: true, v: null}
    }
    if (name == "setAll") {
        var sa = dsHNum(args[0])
        var vals = []
        dsPushAll(vals, args[1])
        if (sa < 0 || sa > n) { dsHRangeErr(sa, 0, n, "index") }
        if (sa + vals.length > n) { dsHRangeErr(sa + vals.length, sa, n, "end") }
        while (i < vals.length) {
            o[sa + i] = vals[i]
            i = i + 1
        }
        return {hit: true, v: null}
    }
    return {hit: false, v: undefined}
}

// The first index at or after from whose element == v, or -1.
function dsListFind(o, v, from) {
    var i = from
    if (i < 0) { i = 0 }
    while (i < o.length) {
        if (dsHEq(o[i], v)) { return i }
        i = i + 1
    }
    return 0 - 1
}

// lastIndexOf / lastIndexWhere's start: the last index by default and clamped to
// it; a negative start searches nothing.
function dsLastFrom(n, args) {
    var s = dsOpt(args, 1, n - 1)
    if (s >= n) { s = n - 1 }
    if (s < 0) { s = 0 - 1 }
    return s
}

// RangeError.checkValidRange(start, end, length): 0 <= start <= length, then
// start <= end <= length, each a RangeError naming the bad argument. An absent end
// is the length when endOpt. Answers the end.
function dsRangeEnd(o, args, endOpt) {
    var n = o.length
    var s = dsHNum(args[0])
    if (s < 0 || s > n) { dsHRangeErr(s, 0, n, "start") }
    var e = n
    if (!endOpt || (args.length > 1 && args[1] !== null && args[1] !== undefined)) { e = dsHNum(args[1]) }
    if (e < s || e > n) { dsHRangeErr(e, s, n, "end") }
    return e
}

function dsPushAll(out, xs) {
    var i = 0
    while (i < xs.length) {
        out.push(xs[i])
        i = i + 1
    }
}

function dsCopy(xs) {
    var out = []
    dsPushAll(out, xs)
    return out
}

// Replace o[from:to] with ins, in place.
function dsSplice(o, from, to, ins) {
    var tail = []
    var i = to
    while (i < o.length) {
        tail.push(o[i])
        i = i + 1
    }
    o.length = from
    dsPushAll(o, ins)
    dsPushAll(o, tail)
}

// List.sort: by cmp, or by the elements' compareTo when cmp is null. A stable
// bottom-up merge sort: dart:core promises no stability, but its implementation
// insertion-sorts any list of up to 32 elements, which is stable, so the two agree
// on every list that short.
function dsSort(o, cmp) {
    var n = o.length
    var src = dsCopy(o)
    var dst = dsCopy(o)
    var w = 1
    while (w < n) {
        var lo = 0
        while (lo < n) {
            var mid = lo + w
            if (mid > n) { mid = n }
            var hi = mid + w
            if (hi > n) { hi = n }
            var a = lo
            var b = mid
            var k = lo
            while (k < hi) {
                if (a < mid && (b >= hi || dsSortCmp(cmp, src[a], src[b]) <= 0)) {
                    dst[k] = src[a]
                    a = a + 1
                } else {
                    dst[k] = src[b]
                    b = b + 1
                }
                k = k + 1
            }
            lo = hi
        }
        var t = src
        src = dst
        dst = t
        w = w * 2
    }
    var i = 0
    while (i < n) {
        o[i] = src[i]
        i = i + 1
    }
}

function dsSortCmp(cmp, a, b) {
    if (cmp === null || cmp === undefined) { return dsHCmp(a, b) }
    return dsHNum(dsHCall(cmp, [a, b]))
}

// String.runes: the code points, as ints; an unpaired surrogate is one of its own.
// dart:core answers an Iterable (printed in parentheses); this is a List.
function dsRunes(s) {
    var out = []
    var i = 0
    while (i < s.length) {
        var w = dsCpWidth(s, i)
        if (w == 2) { out.push((s.charCodeAt(i) - 55296) * 1024 + (s.charCodeAt(i + 1) - 56320) + 65536) }
        else { out.push(s.charCodeAt(i)) }
        i = i + w
    }
    return out
}

// The static String.fromCharCode(c) and String.fromCharCodes(codes, [start, end]):
// a code past U+FFFF becomes a surrogate pair and any other is one code unit.
// {hit: false} for any other name.
function dsStatic(name, args) {
    if (name == "fromCharCode") { return {hit: true, v: dsFromCode(dsHNum(args[0]))} }
    if (name == "fromCharCodes") {
        var cs = args[0]
        var e = dsOpt(args, 2, cs.length)
        var out = ""
        var i = dsOpt(args, 1, 0)
        while (i < e) {
            out = out + dsFromCode(dsHNum(cs[i]))
            i = i + 1
        }
        return {hit: true, v: out}
    }
    return {hit: false, v: undefined}
}
function dsFromCode(c) {
    if (c < 0 || c > 1114111) { dsHRangeErr(c, 0, 1114111, null) }
    if (c < 65536) { return String.fromCharCode(c) }
    var u = c - 65536
    return String.fromCharCode(55296 + Math.floor(u / 1024)) + String.fromCharCode(56320 + u % 1024)
}

// String.codeUnits: the UTF-16 code units, as ints.
function dsUnits(s) {
    var out = []
    var i = 0
    while (i < s.length) {
        out.push(s.charCodeAt(i))
        i = i + 1
    }
    return out
}
