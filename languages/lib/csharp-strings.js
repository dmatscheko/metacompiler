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
// at whitespace. At most max parts (-1: no limit), the last one holding the rest.
function csxSplit(s, seps, max) {
    if (max == 0) { return [] }
    var out = []
    var any = false
    var k = 0
    while (k < seps.length) {
        if (seps[k].length > 0) { any = true }
        k = k + 1
    }
    var last = 0
    var i = 0
    while (i < s.length && out.length + 1 != max) {
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
// Split(separator(s) [, count]): a count follows one separator argument (a char,
// a string or an array of either), which the params char[] form cannot.
// A trailing StringSplitOptions argument is the options (csxSplitOpts).
function csxSplitCall(s, args) {
    var n = args.length
    var opts = 0
    if (n > 0 && csxIsEnumBox(args[n - 1])) {
        opts = args[n - 1]["v"]
        n = n - 1
    }
    var seps = []
    var i = 0
    while (i < n) {
        seps.push(args[i])
        i = i + 1
    }
    if (n == 2 && !csxkIsChar(args[1]) && typeof args[1] != "string" && !Array.isArray(args[1]) && !csxIsNull(args[1])) {
        var max = csxkNum(args[1])
        if (max < 0) { csxkThrow("ArgumentOutOfRangeException", "Count cannot be less than zero. (Parameter 'count')") }
        return csxSplitOpts(s, csxSeps([args[0]]), max, opts)
    }
    return csxSplitOpts(s, csxSeps(seps), 0 - 1, opts)
}
// IndexOfAny(anyOf [, startIndex [, count]]): the first position holding any of
// the chars.
function csxIndexOfAny(s, args) {
    var set = csxCharSet([args[0]], 0)
    var from = args.length > 1 ? csxkNum(args[1]) : 0
    if (from < 0 || from > s.length) {
        csxkThrow("ArgumentOutOfRangeException", "Index was out of range. Must be non-negative and less than or equal to the size of the collection. (Parameter 'startIndex')")
    }
    var end = s.length
    if (args.length > 2) {
        var cnt = csxkNum(args[2])
        if (cnt < 0 || from + cnt > s.length) {
            csxkThrow("ArgumentOutOfRangeException", "Count must be positive and count must refer to a location within the string/array/collection. (Parameter 'count')")
        }
        end = from + cnt
    }
    var i = from
    while (i < end) {
        if (csxInSet(set, s.charCodeAt(i))) { return i }
        i = i + 1
    }
    return 0 - 1
}
// LastIndexOfAny(anyOf [, startIndex [, count]]): searching backward from
// startIndex (the last char by default) over count chars.
function csxLastIndexOfAny(s, args) {
    var set = csxCharSet([args[0]], 0)
    if (s.length == 0) { return 0 - 1 }
    var from = args.length > 1 ? csxkNum(args[1]) : s.length - 1
    if (from < 0 || from >= s.length) {
        csxkThrow("ArgumentOutOfRangeException", "Index was out of range. Must be non-negative and less than the size of the collection. (Parameter 'startIndex')")
    }
    var cnt = from + 1
    if (args.length > 2) {
        cnt = csxkNum(args[2])
        if (cnt < 0 || from - cnt + 1 < 0) {
            csxkThrow("ArgumentOutOfRangeException", "Count must be positive and count must refer to a location within the string/array/collection. (Parameter 'count')")
        }
    }
    var i = from
    while (i > from - cnt) {
        if (csxInSet(set, s.charCodeAt(i))) { return i }
        i = i - 1
    }
    return 0 - 1
}
// The separators of a Split call: chars, strings, or one array of either. A null
// separator (or array) contributes none, so only null separators split at white
// space.
function csxSeps(args) {
    var src = args
    if (args.length == 1 && Array.isArray(args[0])) { src = args[0] }
    var out = []
    var i = 0
    while (i < src.length) {
        if (!csxIsNull(src[i])) { out.push(csxArgStr(src[i])) }
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

// A string receiver's method.
function csxStrMethod(s, name, args) {
    var n = args.length
    if (csxHasCmpArg(name, args)) { return csxStrCmpMethod(s, name, args) }
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
    if (name == "Split") { return csxSplitCall(s, args) }
    if (name == "IndexOfAny") { return csxIndexOfAny(s, args) }
    if (name == "LastIndexOfAny") { return csxLastIndexOfAny(s, args) }
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
    if (name == "Normalize") {
        csxNormalized(s, args)
        return s
    }
    if (name == "IsNormalized") { return csxNormalized(s, args) }
    return csqMethod(s, name, args)
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
// Concat's and Join's IEnumerable<T> parameter: one LINQ query or iterator
// argument is its elements (lib/csharp-linq.js).
function csxSeqParts(args, k) {
    if (args.length == k + 1 && csqIsLazy(args[k])) { return csqToArray(args[k]) }
    return csxParts(args, k)
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
// String.CompareOrdinal(a, b): UTF-16 unit by unit; the difference of the first
// pair that differs, else of the lengths. null orders before every string.
function csxCompareOrdinal(a, b) {
    if (a === null || a === undefined) { return (b === null || b === undefined) ? 0 : 0 - 1 }
    if (b === null || b === undefined) { return 1 }
    var i = 0
    while (i < a.length && i < b.length) {
        if (a.charCodeAt(i) != b.charCodeAt(i)) { return a.charCodeAt(i) - b.charCodeAt(i) }
        i = i + 1
    }
    return a.length - b.length
}
// System.String's static members.
function csxStringStatic(name, args) {
    var n = args.length
    if (name == "Concat") { return csxJoinParts("", csxSeqParts(args, 0)) }
    if (name == "Join" && n >= 1) { return csxJoinParts(csxArgStr(args[0]), csxSeqParts(args, 1)) }
    if (name == "IsNullOrEmpty" && n == 1) { return args[0] === null || args[0] === undefined || args[0].length == 0 }
    if (name == "IsNullOrWhiteSpace" && n == 1) {
        if (args[0] === null || args[0] === undefined) { return true }
        return csxTrim(args[0], [], true, false).length == 0
    }
    if (name == "Equals" && n == 2) { return args[0] === args[1] }
    if (name == "Equals" && n == 3) { return csxEqStr(csxCmpKind(args[2]), args[0], args[1]) }
    if (name == "CompareOrdinal" && n == 2) { return csxCompareOrdinal(args[0], args[1]) }
    if (name == "Compare") {
        if (n == 3 && csxIsEnumBox(args[2])) { return csxCmpStr(csxCmpKind(args[2]), args[0], args[1]) }
        fail("String.Compare without an ordinal StringComparison is the culture's collation, which is not modelled")
    }
    if (name == "Format" && n >= 1) { return csxComposite(args[0], csxParts(args, 1)) }
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

// double.Parse's reading of s - NumberStyles.Float | AllowThousands under the
// invariant culture - or null where it throws FormatException. ASCII white
// space (U+0009..U+000D, U+0020) around a sign, digits with "," group
// separators after the first digit, an optional "." and digits, and an
// exponent e[+-]digits; out of range is an infinity (.NET Core 3.0 on). Failing
// that, the string trimmed by char.IsWhiteSpace may be one of the symbols,
// ignoring case: Infinity, +Infinity, -Infinity, NaN, +NaN, -NaN.
function csxParseDouble(s) {
    var i = 0
    var j = s.length
    while (i < j && csxNumWs(s.charCodeAt(i))) { i = i + 1 }
    while (j > i && csxNumWs(s.charCodeAt(j - 1))) { j = j - 1 }
    var p = i
    var neg = false
    if (p < j && (s.charAt(p) == "+" || s.charAt(p) == "-")) {
        neg = s.charAt(p) == "-"
        p = p + 1
    }
    var num = ""
    var nd = 0
    while (p < j) {
        var c = s.charAt(p)
        if (c >= "0" && c <= "9") {
            num = num + c
            nd = nd + 1
        } else if (c != "," || nd == 0) {
            break
        }
        p = p + 1
    }
    if (p < j && s.charAt(p) == ".") {
        num = num + "."
        p = p + 1
        while (p < j && s.charAt(p) >= "0" && s.charAt(p) <= "9") {
            num = num + s.charAt(p)
            nd = nd + 1
            p = p + 1
        }
    }
    if (nd > 0 && p < j && (s.charAt(p) == "e" || s.charAt(p) == "E")) {
        var k = p + 1
        if (k < j && (s.charAt(k) == "+" || s.charAt(k) == "-")) { k = k + 1 }
        var k0 = k
        while (k < j && s.charAt(k) >= "0" && s.charAt(k) <= "9") { k = k + 1 }
        if (k > k0) {
            num = num + s.substring(p, k)
            p = k
        }
    }
    if (nd > 0 && p == j) {
        var v = parseFloat(num)
        return neg ? -v : v
    }
    var a = 0
    var b = s.length
    while (a < b && csxIsWs(s.charCodeAt(a))) { a = a + 1 }
    while (b > a && csxIsWs(s.charCodeAt(b - 1))) { b = b - 1 }
    var w = s.substring(a, b).toLowerCase()
    if (w == "infinity" || w == "+infinity") { return 1 / 0 }
    if (w == "-infinity") { return -1 / 0 }
    if (w == "nan" || w == "+nan" || w == "-nan") { return 0 / 0 }
    return null
}
function csxNumWs(c) { return c == 32 || (c >= 9 && c <= 13) }

// ----- the BCL enums a String member takes -----
//
// StringComparison, StringSplitOptions and NormalizationForm, with their
// documented values. A member is a box {__csbcl: type name, v: value, __t: the
// type object}, interned on the type object, so == compares it by identity and
// an overload can tell it from an int: Split(',', 2) is a count,
// Split(',', StringSplitOptions.RemoveEmptyEntries) the options.
function csxEnumNames(ty) {
    if (ty == "StringComparison") {
        return ["CurrentCulture", "CurrentCultureIgnoreCase", "InvariantCulture",
                "InvariantCultureIgnoreCase", "Ordinal", "OrdinalIgnoreCase"]
    }
    if (ty == "StringSplitOptions") { return ["None", "RemoveEmptyEntries", "TrimEntries"] }
    return ["FormC", "FormD", "FormKC", "FormKD"]
}
function csxEnumVals(ty) {
    if (ty == "StringComparison") { return [0, 1, 2, 3, 4, 5] }
    if (ty == "StringSplitOptions") { return [0, 1, 2] }
    return [1, 2, 5, 6]
}
function csxEnumMake(t, v) {
    var b = {}
    b["__csbcl"] = t["__csbclty"]
    b["v"] = v
    b["__t"] = t
    t["__v" + v] = b
    return b
}
// The type object `StringComparison` etc. names: its members by name.
function csxEnumType(ty) {
    var t = {}
    t["__csbclty"] = ty
    var ns = csxEnumNames(ty)
    var vs = csxEnumVals(ty)
    var i = 0
    while (i < ns.length) {
        t[ns[i]] = csxEnumMake(t, vs[i])
        i = i + 1
    }
    return t
}
function csxIsEnumBox(v) {
    if (v === null || v === undefined || typeof v != "object" || Array.isArray(v)) { return false }
    return typeof v["__csbcl"] == "string"
}
// The member of value v of b's type, made (and interned) when no name has it.
function csxEnumOf(b, v) {
    var t = b["__t"]
    var m = t["__v" + v]
    if (m === undefined) { return csxEnumMake(t, v) }
    return m
}
// `a | b` on two members (ECMA-334 12.13.3: the enum logical operators).
function csxEnumOr(a, b) { return csxEnumOf(a, a["v"] | b["v"]) }
// Enum.ToString: the member's name; a [Flags] combination (StringSplitOptions)
// is its members' names in value order joined by ", "; otherwise the number.
function csxEnumStr(b) {
    var ty = b["__csbcl"]
    var v = b["v"]
    var ns = csxEnumNames(ty)
    var vs = csxEnumVals(ty)
    var i = 0
    while (i < vs.length) {
        if (vs[i] == v) { return ns[i] }
        i = i + 1
    }
    if (ty == "StringSplitOptions" && v > 0 && v < 4) {
        var out = ""
        i = 1
        while (i < vs.length) {
            if ((v & vs[i]) != 0) { out = out + (out == "" ? "" : ", ") + ns[i] }
            i = i + 1
        }
        return out
    }
    return "" + v
}
function csxEnumVal(v) { return csxIsEnumBox(v) ? v["v"] : csxkNum(v) }

// ----- ordinal comparison under a StringComparison -----
//
// Ordinal compares UTF-16 units; OrdinalIgnoreCase compares them after the
// invariant culture's simple upper-case mapping, unit by unit. The four
// culture-sensitive comparisons are the culture's collation, which is not
// modelled, so they fail rather than answer an ordinal result.
function csxCmpKind(v) {
    var k = csxEnumVal(v)
    if (k == 4 || k == 5) { return k }
    if (k >= 0 && k <= 3) {
        fail("culture-sensitive string comparison (StringComparison." + csxEnumNames("StringComparison")[k] + ") is not modelled")
    }
    csxkThrow("ArgumentException", "The string comparison type passed in is currently not supported. (Parameter 'comparisonType')")
    return 0
}
function csxFoldUp(s) {
    var out = ""
    var i = 0
    while (i < s.length) {
        out = out + String.fromCharCode(jxChUp(s.charCodeAt(i)))
        i = i + 1
    }
    return out
}
// The string as the comparison kind k sees it.
function csxCmpForm(k, s) { return k == 5 ? csxFoldUp(s) : s }
function csxIsNull(v) { return v === null || v === undefined }
function csxCmpStr(k, a, b) {
    if (csxIsNull(a) || csxIsNull(b)) { return csxCompareOrdinal(a, b) }
    return csxCompareOrdinal(csxCmpForm(k, a), csxCmpForm(k, b))
}
function csxEqStr(k, a, b) {
    if (csxIsNull(a) || csxIsNull(b)) { return csxIsNull(a) && csxIsNull(b) }
    return csxCmpForm(k, a) === csxCmpForm(k, b)
}
// StringComparer.Ordinal / StringComparer.OrdinalIgnoreCase: an IComparer<string>
// and IEqualityComparer<string> (csxCmpWith reads __csstrcmp directly).
function csxStrComparer(k) {
    var o = {}
    o["__csstrcmp"] = k
    o["Compare"] = function(a, b) { return csxCmpStr(k, a, b) }
    o["Equals"] = function(a, b) { return csxEqStr(k, a, b) }
    return o
}
function csxStrComparerType() {
    var t = {}
    t["Ordinal"] = csxStrComparer(4)
    t["OrdinalIgnoreCase"] = csxStrComparer(5)
    return t
}
function csxIsStrComparer(v) {
    if (v === null || v === undefined || typeof v != "object" || Array.isArray(v)) { return false }
    return v["__csstrcmp"] !== undefined
}
// Does a call to member name end in a StringComparison argument?
function csxHasCmpArg(name, args) {
    if (args.length < 2 || !csxIsEnumBox(args[args.length - 1])) { return false }
    if (args[args.length - 1]["__csbcl"] != "StringComparison") { return false }
    return name == "Equals" || name == "StartsWith" || name == "EndsWith" || name == "Contains"
        || name == "IndexOf" || name == "LastIndexOf" || name == "Replace"
}
// Such a member: the receiver and the arguments are compared in the
// comparison's form; Replace replaces in the original string.
function csxStrCmpMethod(s, name, args) {
    var k = csxCmpKind(args[args.length - 1])
    var rest = []
    var i = 0
    while (i < args.length - 1) {
        rest.push(args[i])
        i = i + 1
    }
    if (name == "Equals") { return csxEqStr(k, s, rest[0]) }
    if (csxIsNull(rest[0])) { csxkThrow("ArgumentNullException", "Value cannot be null. (Parameter 'value')") }
    var fs = csxCmpForm(k, s)
    var ft = csxCmpForm(k, csxArgStr(rest[0]))
    if (name == "StartsWith") { return jxAt(fs, ft, 0) }
    if (name == "EndsWith") { return jxAt(fs, ft, fs.length - ft.length) }
    if (name == "Contains") { return jxIndexOf(fs, ft, 0) >= 0 }
    rest[0] = ft
    if (name == "IndexOf") { return csxIndexOf(fs, rest) }
    if (name == "LastIndexOf") { return csxLastIndexOf(fs, rest) }
    if (ft.length == 0) { csxkThrow("ArgumentException", "String cannot be of zero length. (Parameter 'oldValue')") }
    var r = csxIsNull(rest[1]) ? "" : csxArgStr(rest[1])
    var out = ""
    var last = 0
    var at = jxIndexOf(fs, ft, 0)
    while (at >= 0) {
        out = out + s.substring(last, at) + r
        last = at + ft.length
        at = jxIndexOf(fs, ft, last)
    }
    return out + s.substring(last, s.length)
}

// ----- String.Normalize / IsNormalized -----
//
// Unicode normalization needs the decomposition tables, which are not modelled.
// What is settled without them: no unit below U+0300 changes under form C (they
// are all NFC_Quick_Check=Yes and none composes with another), and no ASCII unit
// changes under any form, so such a string is its own normal form.
function csxNormalized(s, args) {
    var form = args.length > 0 ? csxEnumVal(args[0]) : 1
    var lim = form == 1 ? 768 : 128
    var i = 0
    while (i < s.length) {
        if (s.charCodeAt(i) >= lim) {
            fail("Unicode normalization of a string holding U+" + csxHex4(s.charCodeAt(i)) + " is not modelled")
        }
        i = i + 1
    }
    return true
}
function csxHex4(c) {
    var out = csxHexOf(c).toUpperCase()
    while (out.length < 4) { out = "0" + out }
    return out
}

// ----- Split's options -----
//
// StringSplitOptions.TrimEntries trims each part (String.Trim), RemoveEmptyEntries
// drops the empty ones, and both may be combined. With a count, once count - 1
// parts are kept the rest of the string is the last part; under
// RemoveEmptyEntries the empty parts at its front are skipped first.
function csxSplitTrim(part, opts) { return (opts & 2) != 0 ? csxTrim(part, [], true, true) : part }
function csxSplitOpts(s, seps, max, opts) {
    if (opts == 0) { return csxSplit(s, seps, max) }
    if (max == 0) { return [] }
    var rm = (opts & 1) != 0
    var all = csxSplit(s, seps, 0 - 1)
    var out = []
    var cur = 0
    var i = 0
    while (max != 1 && i < all.length - 1) {
        var part = csxSplitTrim(all[i], opts)
        if (part.length > 0 || !rm) { out.push(part) }
        cur = cur + all[i].length + csxSepLenAt(s, seps, cur + all[i].length)
        i = i + 1
        if (out.length == max - 1) {
            while (rm && i < all.length - 1 && csxSplitTrim(all[i], opts).length == 0) {
                cur = cur + all[i].length + csxSepLenAt(s, seps, cur + all[i].length)
                i = i + 1
            }
            break
        }
    }
    var last = csxSplitTrim(s.substring(cur, s.length), opts)
    if (last.length > 0 || !rm) { out.push(last) }
    return out
}
// The length of the separator csxSplit matched at position i (0 at the end).
function csxSepLenAt(s, seps, i) {
    if (i >= s.length) { return 0 }
    var j = 0
    var any = false
    while (j < seps.length) {
        if (seps[j].length > 0) {
            any = true
            if (jxAt(s, seps[j], i)) { return seps[j].length }
        }
        j = j + 1
    }
    return any ? 0 : 1
}

// ----- composite formatting and the standard numeric format strings -----
//
// String.Format, Console.WriteLine(format, args), an interpolation hole's
// ':format' and a number's ToString(format). A format item is
// {index[,alignment][:format]}, with '{{' and '}}' the literal braces.
//
// The standard numeric formats D X F N E P G R are modelled, rendered as under
// the INVARIANT culture (the culture every other number rendering here assumes:
// '.' decimal point, ',' group separator in threes, the "-n" and "n %" patterns,
// "NaN" / "Infinity"). A double is formatted from its EXACT value rounded half to
// even (the documented rule for .NET Core 2.1 and later) and keeps its sign when
// it rounds to zero; an integer is formatted from its exact decimal digits.
// Custom format strings ("0.00", "#,##0") and the currency format are not
// modelled and fail. A value that is not a number ignores its format, as a type
// that does not implement IFormattable does.
//
// Hooks: csxkFmtKind(v) is "int" for an integral value, "dbl" for a double or a
// float and "" otherwise; csxkDbl(v) is a double's value; csxkHex(v) an
// integer's two's complement hex digits at its own width.
function csxFmtBad() { csxkThrow("FormatException", "Format specifier was invalid.") }
function csxHexOf(n) {
    var ds = "0123456789abcdef"
    if (n == 0) { return "0" }
    var out = ""
    var v = n
    while (v > 0) {
        var q = Math.floor(v / 16)
        out = ds.charAt(v - q * 16) + out
        v = q
    }
    return out
}
function csxZeros(n) {
    var out = ""
    var i = 0
    while (i < n) {
        out = out + "0"
        i = i + 1
    }
    return out
}
// The digit string ds rounded half to even to keep digits: {d, carry}, carry
// saying the rounding added a leading digit (999 -> 1000).
function csxRoundEven(ds, keep) {
    if (keep >= ds.length) { return {d: ds, carry: false} }
    var head = ds.substring(0, keep)
    var c = ds.charCodeAt(keep)
    var up = c > 53
    if (c == 53) {
        var i = keep + 1
        while (i < ds.length && ds.charAt(i) == "0") { i = i + 1 }
        if (i < ds.length) { up = true } else if (keep > 0) { up = (head.charCodeAt(keep - 1) - 48) % 2 == 1 }
    }
    if (!up) { return {d: head, carry: false} }
    var out = ""
    var k = head.length - 1
    var carry = true
    while (k >= 0) {
        var d = head.charCodeAt(k) - 48
        if (carry) {
            d = d + 1
            carry = d == 10
            if (carry) { d = 0 }
        }
        out = String.fromCharCode(48 + d) + out
        k = k - 1
    }
    if (carry) { return {d: "1" + out, carry: true} }
    return {d: out, carry: false}
}
function csxStripZeros(ds) {
    var e = ds.length
    while (e > 0 && ds.charAt(e - 1) == "0") { e = e - 1 }
    return ds.substring(0, e)
}
// A number as {neg, isInt, x, ds, e}: x is a double's magnitude, ds the
// significant digits (no trailing zeros, "" for zero) of the value
// 0.ds * 10^(e + 1), so e is the exponent of the first digit. A double's ds are
// its 17-digit rounding, which csxSig refines.
function csxNumParts(v) {
    var o = {}
    o["isInt"] = csxkFmtKind(v) == "int"
    o["x"] = 0
    o["ds"] = ""
    o["e"] = 0
    if (o["isInt"]) {
        var s = csxkStr(v)
        o["neg"] = s.charAt(0) == "-"
        if (o["neg"]) { s = s.substring(1, s.length) }
        o["ds"] = csxStripZeros(s)
        o["e"] = s.length - 1
        return o
    }
    var x = csxkDbl(v)
    o["neg"] = x < 0 || (x == 0 && 1 / x < 0)
    o["x"] = o["neg"] ? 0 - x : x
    if (x != 0) {
        var t = floPrec(o["x"], 17)
        var ei = t.indexOf("e")
        o["ds"] = csxStripZeros(t.substring(0, ei))
        o["e"] = parseInt(t.substring(ei + 1, t.length), 10)
    }
    return o
}
// The value rounded half to even to n significant digits, as {ds, e}. Past the
// seventeenth digit a double's decimal is exact here only for an integer below
// 10^17, whose seventeen digits are all of it.
function csxSig(p, n) {
    var r = {}
    r["ds"] = ""
    r["e"] = p["e"]
    if (p["ds"] == "" || n < 0) { return r }
    if (n == 0) {
        // Below one unit of the last place kept: up only past the half.
        var c = p["ds"].charCodeAt(0)
        if (c > 53 || (c == 53 && p["ds"].length > 1)) {
            r["ds"] = "1"
            r["e"] = p["e"] + 1
        }
        return r
    }
    if (p["isInt"] || n >= p["ds"].length && n <= 17) {
        var q = csxRoundEven(p["ds"], n)
        r["ds"] = csxStripZeros(q["d"])
        if (q["carry"]) { r["e"] = p["e"] + 1 }
        return r
    }
    if (n > 17) {
        if (p["x"] != Math.floor(p["x"]) || p["x"] >= 100000000000000000) {
            fail("a numeric format needing more than 17 significant digits of a double is not modelled")
        }
        r["ds"] = p["ds"]
        return r
    }
    var t = floPrec(p["x"], n)
    var ei = t.indexOf("e")
    r["ds"] = csxStripZeros(t.substring(0, ei))
    r["e"] = parseInt(t.substring(ei + 1, t.length), 10)
    return r
}
// Digits ds (the value 0.ds * 10^(e+1)) in fixed point with prec fraction digits.
function csxPlace(ds, e, prec) {
    var ip = "0"
    var fp = ""
    if (ds != "") {
        if (e >= 0) {
            ip = e + 1 <= ds.length ? ds.substring(0, e + 1) : ds + csxZeros(e + 1 - ds.length)
            fp = e + 1 < ds.length ? ds.substring(e + 1, ds.length) : ""
        } else {
            fp = csxZeros(0 - e - 1) + ds
        }
    }
    fp = fp + csxZeros(prec - fp.length)
    return {i: ip, f: fp}
}
function csxGroup(ip) {
    var out = ""
    var n = ip.length
    var i = 0
    while (i < n) {
        if (i > 0 && (n - i) % 3 == 0) { out = out + "," }
        out = out + ip.charAt(i)
        i = i + 1
    }
    return out
}
// The sign a formatted number carries: a double keeps it when it rounds to zero.
function csxSignOf(p, ds) { return p["neg"] && (!p["isInt"] || ds != "") ? "-" : "" }
// Fixed point (F, N and P): prec fraction digits of the value times 10^shift.
function csxFixedFmt(p, prec, shift, group) {
    var r = csxSig(p, p["e"] + shift + 1 + prec)
    var pl = csxPlace(r["ds"], r["e"] + shift, prec)
    var body = group ? csxGroup(pl["i"]) : pl["i"]
    if (prec > 0) { body = body + "." + pl["f"] }
    return csxSignOf(p, r["ds"]) + body
}
function csxExpPart(e, letter, minDigits) {
    var s = "" + (e < 0 ? 0 - e : e)
    while (s.length < minDigits) { s = "0" + s }
    return letter + (e < 0 ? "-" : "+") + s
}
// One value under one format string.
function csxFmtValue(v, spec) {
    var kind = csxkFmtKind(v)
    if (spec == "" || kind == "") { return csxkStr(v) }
    var letter = spec.charAt(0)
    var lc = letter.toUpperCase()
    var i = 1
    while (i < spec.length && spec.charAt(i) >= "0" && spec.charAt(i) <= "9") { i = i + 1 }
    if (i < spec.length || spec.length > 10 || "DXFNEPGRC".indexOf(lc) < 0) {
        fail("the custom numeric format string \"" + spec + "\" is not modelled")
    }
    var hasPrec = spec.length > 1
    var prec = hasPrec ? parseInt(spec.substring(1, spec.length), 10) : 0
    if (lc == "C") { fail("the currency format is culture-specific and is not modelled") }
    if (lc == "D" || lc == "X") {
        if (kind != "int") { csxFmtBad() }
        var s = lc == "D" ? csxkStr(v) : csxkHex(v)
        var neg = s.charAt(0) == "-"
        if (neg) { s = s.substring(1, s.length) }
        if (letter == "X") { s = s.toUpperCase() }
        if (prec > s.length) { s = csxZeros(prec - s.length) + s }
        return neg ? "-" + s : s
    }
    if (lc == "R") {
        if (kind != "dbl") { csxFmtBad() }
        return csxkStr(v)
    }
    var x = csxkDbl(v)
    if (kind == "dbl" && x - x != 0) { return csxkStr(v) }
    var p = csxNumParts(v)
    if (lc == "F") { return csxFixedFmt(p, hasPrec ? prec : 2, 0, false) }
    if (lc == "N") { return csxFixedFmt(p, hasPrec ? prec : 2, 0, true) }
    if (lc == "P") { return csxFixedFmt(p, hasPrec ? prec : 2, 2, true) + " %" }
    if (lc == "E") {
        var ep = hasPrec ? prec : 6
        var re = csxSig(p, ep + 1)
        var md = re["ds"] == "" ? "0" : re["ds"]
        md = md + csxZeros(ep + 1 - md.length)
        var mant = ep > 0 ? md.charAt(0) + "." + md.substring(1, md.length) : md
        return csxSignOf(p, re["ds"]) + mant + csxExpPart(re["ds"] == "" ? 0 : re["e"], letter == "E" ? "E" : "e", 3)
    }
    // G: fixed point when the exponent is at least -5 and below the precision,
    // else scientific with a two-digit exponent; G or G0 is the value's own
    // ToString().
    if (prec == 0) { return csxkStr(v) }
    var rg = csxSig(p, prec)
    var sg = csxSignOf(p, rg["ds"])
    if (rg["ds"] == "") { return sg + "0" }
    var ge = rg["e"]
    var gd = rg["ds"]
    if (ge > 0 - 5 && ge < prec) {
        var pg = csxPlace(gd, ge, 0)
        return sg + pg["i"] + (pg["f"] == "" ? "" : "." + pg["f"])
    }
    var gm = gd.length > 1 ? gd.charAt(0) + "." + gd.substring(1, gd.length) : gd
    return sg + gm + csxExpPart(ge, letter == "G" ? "E" : "e", 2)
}
// An alignment: padded on the left to width al, on the right for a negative al.
function csxAlign(s, al) {
    var w = al < 0 ? 0 - al : al
    if (s.length >= w) { return s }
    return al < 0 ? s + jxRepeat(" ", w - s.length) : jxRepeat(" ", w - s.length) + s
}
function csxBadFormat() { csxkThrow("FormatException", "Input string was not in a correct format.") }
// String.Format(format, args), args being the params object[] array.
function csxComposite(fmt, args) {
    if (csxIsNull(fmt)) { csxkThrow("ArgumentNullException", "Value cannot be null. (Parameter 'format')") }
    var out = ""
    var i = 0
    var n = fmt.length
    while (i < n) {
        var c = fmt.charAt(i)
        if (c == "}") {
            if (i + 1 >= n || fmt.charAt(i + 1) != "}") { csxBadFormat() }
            out = out + "}"
            i = i + 2
        } else if (c != "{") {
            out = out + c
            i = i + 1
        } else if (i + 1 < n && fmt.charAt(i + 1) == "{") {
            out = out + "{"
            i = i + 2
        } else {
            i = i + 1
            var idx = 0
            var nd = 0
            while (i < n && fmt.charAt(i) >= "0" && fmt.charAt(i) <= "9") {
                idx = idx * 10 + (fmt.charCodeAt(i) - 48)
                nd = nd + 1
                i = i + 1
            }
            if (nd == 0) { csxBadFormat() }
            while (i < n && fmt.charAt(i) == " ") { i = i + 1 }
            var al = 0
            if (i < n && fmt.charAt(i) == ",") {
                i = i + 1
                while (i < n && fmt.charAt(i) == " ") { i = i + 1 }
                var aneg = i < n && fmt.charAt(i) == "-"
                if (aneg) { i = i + 1 }
                var an = 0
                while (i < n && fmt.charAt(i) >= "0" && fmt.charAt(i) <= "9") {
                    al = al * 10 + (fmt.charCodeAt(i) - 48)
                    an = an + 1
                    i = i + 1
                }
                if (an == 0) { csxBadFormat() }
                if (aneg) { al = 0 - al }
                while (i < n && fmt.charAt(i) == " ") { i = i + 1 }
            }
            var spec = ""
            if (i < n && fmt.charAt(i) == ":") {
                i = i + 1
                var sb = i
                while (i < n && fmt.charAt(i) != "}" && fmt.charAt(i) != "{") { i = i + 1 }
                spec = fmt.substring(sb, i)
            }
            if (i >= n || fmt.charAt(i) != "}") { csxBadFormat() }
            i = i + 1
            if (idx >= args.length) {
                csxkThrow("FormatException", "Index (zero based) must be greater than or equal to zero and less than the size of the argument list.")
            }
            var av = args[idx]
            out = out + csxAlign(csxIsNull(av) ? "" : csxFmtValue(av, spec), al)
        }
    }
    return out
}
