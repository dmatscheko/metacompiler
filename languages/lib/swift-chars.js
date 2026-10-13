// swift-chars.js -- Swift's Character, for swift-interpreter.abnf (include) and
// languages/lib/swift-rt.metajs (import). abnf/jsrtswift.go is the Go twin's copy.
// It needs java-chars.js's classification table (Unicode 16.0) in the same scope.
//
// A Character is its one-cluster String here. Its properties are those of
// swift 6.1.2's CharacterProperties.swift: most read the FIRST scalar's Unicode
// properties, the numeric and hex ones need a single-scalar Character, and
// "\r\n" is the one two-scalar Character with an ASCII value (10). isUppercase /
// isLowercase read the first scalar too, where Swift falls back to a case-mapping
// comparison for a multi-scalar Character; the two agree whenever the first
// scalar carries the case, which is every cluster a combining mark extends.

// Whether name is a Character property this file answers.
function swcIsProp(name) {
    return name == "isLetter" || name == "isUppercase" || name == "isLowercase" || name == "isNumber"
        || name == "isWholeNumber" || name == "wholeNumberValue" || name == "isWhitespace"
        || name == "isNewline" || name == "asciiValue" || name == "isASCII" || name == "isHexDigit"
        || name == "hexDigitValue"
}

// The code point at UTF-16 offset i of s.
function swcCodeAt(s, i) {
    var c = s.charCodeAt(i)
    if (c >= 55296 && c <= 56319 && i + 1 < s.length) {
        var d = s.charCodeAt(i + 1)
        if (d >= 56320 && d <= 57343) { return (c - 55296) * 1024 + (d - 56320) + 65536 }
    }
    return c
}

// Unicode's White_Space property.
function swcIsWhite(c) {
    if (c >= 9 && c <= 13) { return true }
    if (c == 32 || c == 133 || c == 160 || c == 5760) { return true }
    if (c >= 8192 && c <= 8202) { return true }
    return c == 8232 || c == 8233 || c == 8239 || c == 8287 || c == 12288
}

// The numeric value Swift's numericType / numericValue see: java's
// getNumericValue without its Latin-letter digits (a..z, A..Z and their
// fullwidth forms are 10..35 there and no number in Swift). -1 is no number,
// -2 a number that is no non-negative integer (1/2).
function swcNumVal(c) {
    if ((c >= 65 && c <= 90) || (c >= 97 && c <= 122)) { return 0 - 1 }
    if ((c >= 65313 && c <= 65338) || (c >= 65345 && c <= 65370)) { return 0 - 1 }
    return jcNumericValue(c)
}

// The value of a hex digit, ASCII or fullwidth, or -1.
function swcHexVal(c) {
    if (c >= 48 && c <= 57) { return c - 48 }
    if (c >= 65 && c <= 70) { return c - 55 }
    if (c >= 97 && c <= 102) { return c - 87 }
    if (c >= 65296 && c <= 65305) { return c - 65296 }
    if (c >= 65313 && c <= 65318) { return c - 65303 }
    if (c >= 65345 && c <= 65350) { return c - 65335 }
    return 0 - 1
}

// Character property `name` of the Character s; null is nil.
function swcProp(s, name) {
    var c = swcCodeAt(s, 0)
    var single = s.length == 1
    if (c > 65535) { single = s.length == 2 }
    if (name == "asciiValue" || name == "isASCII") {
        var av = 0 - 1
        if (s == "\r\n") { av = 10 }
        else if (single && c < 128) { av = c }
        if (name == "isASCII") { return av >= 0 }
        if (av < 0) { return null }
        return av
    }
    if (name == "isLetter") { return jcIsAlphabetic(c) }
    if (name == "isUppercase") { return jcIsUpper(c) }
    if (name == "isLowercase") { return jcIsLower(c) }
    if (name == "isWhitespace") { return swcIsWhite(c) }
    if (name == "isNewline") { return (c >= 10 && c <= 13) || c == 133 || c == 8232 || c == 8233 }
    if (name == "isNumber") { return swcNumVal(c) != 0 - 1 }
    if (name == "isHexDigit" || name == "hexDigitValue") {
        var hv = 0 - 1
        if (single) { hv = swcHexVal(c) }
        if (name == "isHexDigit") { return hv >= 0 }
        if (hv < 0) { return null }
        return hv
    }
    var nv = 0 - 1
    if (single) { nv = swcNumVal(c) }
    if (name == "isWholeNumber") { return nv >= 0 }
    if (nv < 0) { return null }
    return nv
}
