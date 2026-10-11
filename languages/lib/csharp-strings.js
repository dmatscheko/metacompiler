// csharp-strings.js -- System.String's and System.Char's members, for
// csharp-interpreter.abnf (include) and languages/lib/csharp-rt.metajs (import),
// both of which also include lib/java-chars.js (the Unicode classification
// table) and lib/utf16-strings.js (the per-unit case mapping and the ordinal
// search). abnf/jsrtcsstr.go is the Go twin's port; the three must answer alike.
//
// Strings are plain host strings and are worked on as UTF-16 units. What depends
// on the value model is asked through hooks each importer defines:
//   csxkIsChar(v)       is v a char
//   csxkCode(v)         a char's code
//   csxkChar(c)         the char of code c
//   csxkNum(v)          an int argument as a number
//   csxkStr(v)          C#'s string conversion of v (Concat, Join)
//   csxkThrow(t, msg)   raise the BCL exception t
//
// Comparison and search are ORDINAL: the culture-sensitive forms
// (String.Compare, CompareTo, IndexOf(string) under a culture with ignorable
// characters) are not modelled, and String.Compare / CompareTo are absent rather
// than wrong. Case mapping is per unit and simple (one unit to one unit), which
// is what .NET's ToUpper/ToLower do: "ß".ToUpper() stays "ß".

// char.IsWhiteSpace: the Unicode separators (Zs, Zl, Zp), U+0009..U+000D and U+0085.
function csxIsWs(c) {
    if ((c >= 9 && c <= 13) || c == 32 || c == 133 || c == 160) { return true }
    if (c == 5760 || (c >= 8192 && c <= 8202)) { return true }
    return c == 8232 || c == 8233 || c == 8239 || c == 8287 || c == 12288
}
// char.IsUpper / IsLower are the general categories Lu / Ll exactly (class 1 / 2
// of java-chars.js, without Java's Other_Uppercase / Other_Lowercase).
function csxIsUpper(c) {
    if (c < 128) { return c >= 65 && c <= 90 }
    return jcClass(c) % 8 == 1
}
function csxIsLower(c) {
    if (c < 128) { return c >= 97 && c <= 122 }
    return jcClass(c) % 8 == 2
}
// char.IsControl: the general category Cc.
function csxIsControl(c) { return c < 32 || (c >= 127 && c < 160) }

// A char or a string argument as a string.
function csxArgStr(v) {
    if (csxkIsChar(v)) { return String.fromCharCode(csxkCode(v)) }
    return v
}
// The chars of a params char[] argument list from index k on, as codes: one
// array argument is the array itself.
function csxCharSet(args, k) {
    var out = []
    if (args.length == k + 1 && Array.isArray(args[k])) {
        var a = args[k]
        var j = 0
        while (j < a.length) {
            out.push(csxkCode(a[j]))
            j = j + 1
        }
        return out
    }
    var i = k
    while (i < args.length) {
        out.push(csxkCode(args[i]))
        i = i + 1
    }
    return out
}
function csxInSet(set, c) {
    var i = 0
    while (i < set.length) {
        if (set[i] == c) { return true }
        i = i + 1
    }
    return false
}
// Trim / TrimStart / TrimEnd: whitespace when no char is given, else the set.
function csxTrim(s, set, lead, trail) {
    var b = 0
    var e = s.length
    if (lead) {
        while (b < e && (set.length == 0 ? csxIsWs(s.charCodeAt(b)) : csxInSet(set, s.charCodeAt(b)))) { b = b + 1 }
    }
    if (trail) {
        while (e > b && (set.length == 0 ? csxIsWs(s.charCodeAt(e - 1)) : csxInSet(set, s.charCodeAt(e - 1)))) { e = e - 1 }
    }
    return s.substring(b, e)
}
// Split: at every occurrence of any separator, the first one listed winning at
// a position; empty entries are kept. No separator (or only empty ones) splits
// at whitespace.
function csxSplit(s, seps) {
    var out = []
    var any = false
    var k = 0
    while (k < seps.length) {
        if (seps[k].length > 0) { any = true }
        k = k + 1
    }
    var last = 0
    var i = 0
    while (i < s.length) {
        var hit = 0
        if (!any) {
            if (csxIsWs(s.charCodeAt(i))) { hit = 1 }
        } else {
            var j = 0
            while (j < seps.length && hit == 0) {
                if (seps[j].length > 0 && jxAt(s, seps[j], i)) { hit = seps[j].length }
                j = j + 1
            }
        }
        if (hit > 0) {
            out.push(s.substring(last, i))
            i = i + hit
            last = i
        } else {
            i = i + 1
        }
    }
    out.push(s.substring(last, s.length))
    return out
}
// The separators of a Split call: chars, strings, or one array of either.
function csxSeps(args) {
    var src = args
    if (args.length == 1 && Array.isArray(args[0])) { src = args[0] }
    var out = []
    var i = 0
    while (i < src.length) {
        out.push(csxArgStr(src[i]))
        i = i + 1
    }
    return out
}
function csxPad(s, args, left) {
    var n = csxkNum(args[0])
    if (n < 0) { csxkThrow("ArgumentOutOfRangeException", "Non-negative number required. (Parameter 'totalWidth')") }
    var fill = args.length > 1 ? String.fromCharCode(csxkCode(args[1])) : " "
    if (n <= s.length) { return s }
    var pad = jxRepeat(fill, n - s.length)
    return left ? pad + s : s + pad
}
// IndexOf(value [, startIndex [, count]]) / LastIndexOf(value [, startIndex]).
function csxIndexOf(s, args) {
    var t = csxArgStr(args[0])
    var from = args.length > 1 ? csxkNum(args[1]) : 0
    if (from < 0 || from > s.length) {
        csxkThrow("ArgumentOutOfRangeException", "Index was out of range. Must be non-negative and less than or equal to the size of the collection. (Parameter 'startIndex')")
    }
    var limit = s.length
    if (args.length > 2) {
        var cnt = csxkNum(args[2])
        if (cnt < 0 || from + cnt > s.length) {
            csxkThrow("ArgumentOutOfRangeException", "Count must be positive and count must refer to a location within the string/array/collection. (Parameter 'count')")
        }
        limit = from + cnt
    }
    var r = jxIndexOf(s.substring(0, limit), t, from)
    return r
}
function csxLastIndexOf(s, args) {
    var t = csxArgStr(args[0])
    if (args.length < 2) { return jxLastIndexOf(s, t, s.length) }
    var from = csxkNum(args[1])
    if (from < 0 || from >= s.length) {
        if (!(s.length == 0 && (from == 0 || from == 0 - 1))) {
            csxkThrow("ArgumentOutOfRangeException", "Index was out of range. Must be non-negative and less than the size of the collection. (Parameter 'startIndex')")
        }
    }
    // The search covers s[0..from], so a match must END at or before from.
    return jxLastIndexOf(s.substring(0, from + 1), t, from + 1)
}
// Substring(startIndex [, length]): a LENGTH, not an end index.
function csxSubstring(s, args) {
    var b = args.length > 0 ? csxkNum(args[0]) : 0
    if (b < 0) { csxkThrow("ArgumentOutOfRangeException", "StartIndex cannot be less than zero. (Parameter 'startIndex')") }
    if (b > s.length) { csxkThrow("ArgumentOutOfRangeException", "startIndex cannot be larger than length of string. (Parameter 'startIndex')") }
    if (args.length < 2) { return s.substring(b, s.length) }
    var n = csxkNum(args[1])
    if (n < 0) { csxkThrow("ArgumentOutOfRangeException", "Length cannot be less than zero. (Parameter 'length')") }
    if (b + n > s.length) { csxkThrow("ArgumentOutOfRangeException", "Index and length must refer to a location within the string. (Parameter 'length')") }
    return s.substring(b, b + n)
}
function csxRemove(s, args) {
    var b = csxkNum(args[0])
    if (b < 0) { csxkThrow("ArgumentOutOfRangeException", "StartIndex cannot be less than zero. (Parameter 'startIndex')") }
    if (args.length < 2) {
        if (b >= s.length) { csxkThrow("ArgumentOutOfRangeException", "startIndex must be less than length of string. (Parameter 'startIndex')") }
        return s.substring(0, b)
    }
    var n = csxkNum(args[1])
    if (n < 0) { csxkThrow("ArgumentOutOfRangeException", "Count cannot be less than zero. (Parameter 'count')") }
    if (b + n > s.length) { csxkThrow("ArgumentOutOfRangeException", "Index and count must refer to a location within the string. (Parameter 'count')") }
    return s.substring(0, b) + s.substring(b + n, s.length)
}
function csxToChars(s, b, n) {
    var out = []
    var i = b
    while (i < b + n) {
        out.push(csxkChar(s.charCodeAt(i)))
        i = i + 1
    }
    return out
}

// A string receiver's method. The compiler half lowers IndexOf and Contains to
// the shared runtime's indexOf / contains (a List has both), so those spellings
// arrive too.
function csxStrMethod(s, name0, args) {
    var name = name0
    if (name == "indexOf") { name = "IndexOf" }
    if (name == "contains") { name = "Contains" }
    var n = args.length
    if (name == "Substring") { return csxSubstring(s, args) }
    if (name == "IndexOf") { return csxIndexOf(s, args) }
    if (name == "LastIndexOf") { return csxLastIndexOf(s, args) }
    if (name == "Contains" && n == 1) { return jxIndexOf(s, csxArgStr(args[0]), 0) >= 0 }
    if (name == "StartsWith" && n == 1) { return jxAt(s, csxArgStr(args[0]), 0) }
    if (name == "EndsWith" && n == 1) {
        var t = csxArgStr(args[0])
        return jxAt(s, t, s.length - t.length)
    }
    if ((name == "ToUpper" || name == "ToUpperInvariant") && n == 0) { return jxUpper(s) }
    if ((name == "ToLower" || name == "ToLowerInvariant") && n == 0) { return jxLower(s) }
    if (name == "Trim") { return csxTrim(s, csxCharSet(args, 0), true, true) }
    if (name == "TrimStart") { return csxTrim(s, csxCharSet(args, 0), true, false) }
    if (name == "TrimEnd") { return csxTrim(s, csxCharSet(args, 0), false, true) }
    if (name == "Replace" && n == 2) {
        var o = csxArgStr(args[0])
        if (o.length == 0) { csxkThrow("ArgumentException", "String cannot be of zero length. (Parameter 'oldValue')") }
        var r = args[1] === null || args[1] === undefined ? "" : csxArgStr(args[1])
        return jxReplace(s, o, r)
    }
    if (name == "Split") { return csxSplit(s, csxSeps(args)) }
    if (name == "ToCharArray") {
        if (n == 0) { return csxToChars(s, 0, s.length) }
        var b = csxkNum(args[0])
        var c = csxkNum(args[1])
        if (b < 0 || c < 0 || b + c > s.length) {
            csxkThrow("ArgumentOutOfRangeException", "Index was out of range. Must be non-negative and less than the size of the collection. (Parameter 'startIndex')")
        }
        return csxToChars(s, b, c)
    }
    if (name == "PadLeft") { return csxPad(s, args, true) }
    if (name == "PadRight") { return csxPad(s, args, false) }
    if (name == "Insert" && n == 2) {
        var at = csxkNum(args[0])
        if (at < 0 || at > s.length) {
            csxkThrow("ArgumentOutOfRangeException", "Index was out of range. Must be non-negative and less than or equal to the size of the collection. (Parameter 'startIndex')")
        }
        return s.substring(0, at) + args[1] + s.substring(at, s.length)
    }
    if (name == "Remove") { return csxRemove(s, args) }
    fail("unknown string method: " + name0)
    return undefined
}

// A char receiver's method beyond System.Object's: CompareTo is the difference of
// the two codes, as System.Char's is.
function csxCharMethod(c, name, args) {
    if (name == "CompareTo" && args.length == 1) { return csxkCode(c) - csxkCode(args[0]) }
    fail("unknown char method: " + name)
    return undefined
}

// The char a char.IsX(...) call tests: (c) or (s, index).
function csxCharArg(args) {
    if (args.length == 2) {
        var s = args[0]
        var i = csxkNum(args[1])
        if (i < 0 || i >= s.length) {
            csxkThrow("ArgumentOutOfRangeException", "Index was out of range. Must be non-negative and less than the size of the collection. (Parameter 'index')")
        }
        return s.charCodeAt(i)
    }
    return csxkCode(args[0])
}
// System.Char's static members.
function csxCharStatic(name, args) {
    if (name == "ToUpper") { return csxkChar(jxChUp(csxkCode(args[0]))) }
    if (name == "ToLower") { return csxkChar(jxChLo(csxkCode(args[0]))) }
    var c = csxCharArg(args)
    if (name == "IsDigit") { return jcIsDigit(c) }
    if (name == "IsLetter") { return jcIsLetter(c) }
    if (name == "IsLetterOrDigit") { return jcIsLetter(c) || jcIsDigit(c) }
    if (name == "IsUpper") { return csxIsUpper(c) }
    if (name == "IsLower") { return csxIsLower(c) }
    if (name == "IsWhiteSpace") { return csxIsWs(c) }
    if (name == "IsControl") { return csxIsControl(c) }
    if (name == "IsSurrogate") { return c >= 55296 && c <= 57343 }
    fail("unknown char method: " + name)
    return undefined
}

// The parts of a params object[] list from index k on: one array argument is
// the array itself.
function csxParts(args, k) {
    if (args.length == k + 1 && Array.isArray(args[k])) { return args[k] }
    var out = []
    var i = k
    while (i < args.length) {
        out.push(args[i])
        i = i + 1
    }
    return out
}
function csxJoinParts(sep, parts) {
    var out = ""
    var i = 0
    while (i < parts.length) {
        if (i > 0) { out = out + sep }
        out = out + csxkStr(parts[i])
        i = i + 1
    }
    return out
}
// System.String's static members.
function csxStringStatic(name, args) {
    var n = args.length
    if (name == "Concat") { return csxJoinParts("", csxParts(args, 0)) }
    if (name == "Join" && n >= 1) { return csxJoinParts(csxArgStr(args[0]), csxParts(args, 1)) }
    if (name == "IsNullOrEmpty" && n == 1) { return args[0] === null || args[0] === undefined || args[0].length == 0 }
    if (name == "IsNullOrWhiteSpace" && n == 1) {
        if (args[0] === null || args[0] === undefined) { return true }
        return csxTrim(args[0], [], true, false).length == 0
    }
    if (name == "Equals" && n == 2) { return args[0] === args[1] }
    fail("unknown string method: " + name)
    return undefined
}

// new string(c, count) / new string(char[]) / new string(char[], start, length).
function csxNewString(args) {
    if (args.length == 2 && csxkIsChar(args[0])) {
        var n = csxkNum(args[1])
        if (n < 0) { csxkThrow("ArgumentOutOfRangeException", "Non-negative number required. (Parameter 'count')") }
        return jxRepeat(String.fromCharCode(csxkCode(args[0])), n)
    }
    var a = args[0]
    if (a === null || a === undefined) { return "" }
    var b = 0
    var e = a.length
    if (args.length == 3) {
        b = csxkNum(args[1])
        e = b + csxkNum(args[2])
        if (b < 0 || e < b || e > a.length) {
            csxkThrow("ArgumentOutOfRangeException", "Index was out of range. Must be non-negative and less than the size of the collection. (Parameter 'startIndex')")
        }
    }
    var out = ""
    var i = b
    while (i < e) {
        out = out + String.fromCharCode(csxkCode(a[i]))
        i = i + 1
    }
    return out
}
