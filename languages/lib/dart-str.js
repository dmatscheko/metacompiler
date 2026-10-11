// dart-str.js -- dart:core's String members.
//
// Written once for the interpreter half (dart-interpreter.abnf includes it) and for
// layer 2 (languages/lib/dart-rt.metajs imports it), in the restricted subset both
// accept - see the head of regex.js. abnf/jsrtdart.go's dartStrMember is the Go twin
// the compiler half runs under llvm.Run. There is no dart toolchain on this machine:
// every rule below is dart:core's documentation of String.
//
// A Dart String is a sequence of UTF-16 code units, and every index, length and
// position here counts them. A Pattern argument is a String; any other receiver or
// argument shape answers a miss, and the includer's own dispatch goes on.
//
// Hooks the includer defines:
//   dsHNum(v)          an int argument as a plain number
//   dsHIndexErr(i, s)  throw dart's IndexError for index i of s (does not return)

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
    if (args.length == 0 || typeof args[0] != "string") { return miss }
    var p = args[0]
    if (name == "indexOf") { return {hit: true, v: dsFind(s, p, dsOpt(args, 1, 0))} }
    if (name == "lastIndexOf") { return {hit: true, v: dsFindLast(s, p, dsOpt(args, 1, n))} }
    if (name == "contains") { return {hit: true, v: dsFind(s, p, dsOpt(args, 1, 0)) >= 0} }
    if (name == "startsWith") { return {hit: true, v: dsAt(s, dsOpt(args, 1, 0), p)} }
    if (name == "endsWith") { return {hit: true, v: dsAt(s, n - p.length, p)} }
    if (name == "split") { return {hit: true, v: dsSplit(s, p)} }
    if (name == "replaceAll") { return {hit: true, v: dsReplaceAll(s, p, args[1])} }
    if (name == "replaceFirst") {
        var fi = dsFind(s, p, dsOpt(args, 2, 0))
        if (fi < 0) { return {hit: true, v: s} }
        return {hit: true, v: s.substring(0, fi) + args[1] + s.substring(fi + p.length, n)}
    }
    return miss
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
