// python-chars.js -- str.isalpha / isdecimal / isdigit / isnumeric of one code
// point, for python-interpreter.abnf (include) and languages/lib/python-rt.metajs
// (import), each after lib/java-chars.js. Go twin: pyCpAlpha .. pyCpNumeric in
// abnf/pystrmethod.go.
//
// CPython 3.14 and java-chars.js's table are both Unicode 16.0, and over every
// code point isalpha IS Character.isLetter (the L* categories) and isdecimal IS
// Character.isDigit (Nd). isdigit adds the Numeric_Type=Digit characters
// (superscripts, circled digits, ...) of pyDigitRuns. isnumeric adds every
// other non-letter with a numeric value (Nl, No, fractions) and the Han
// letters whose value comes from Unihan, pyHanNumerals.

// First and last code point of each run.
var pyDigitRuns = [178, 179, 185, 185, 4969, 4977, 6618, 6618, 8304, 8304, 8308, 8313, 8320, 8329,
                   9312, 9320, 9332, 9340, 9352, 9360, 9450, 9450, 9461, 9469, 9471, 9471,
                   10102, 10110, 10112, 10120, 10122, 10130, 68160, 68163, 69216, 69224,
                   69714, 69722, 127232, 127242]

var pyHanNumerals = [13317, 13443, 14378, 15181, 19968, 19971, 19975, 19977, 20004, 20061, 20108,
                     20116, 20118, 20140, 20159, 20160, 20191, 20200, 20237, 20336, 20457, 20486,
                     20740, 20806, 20841, 20843, 20845, 21313, 21315, 21316, 21317, 21324, 21441,
                     21442, 21443, 21444, 22235, 22769, 22777, 24186, 24318, 24319, 24332, 24333,
                     24334, 24336, 25296, 25342, 25420, 26578, 27934, 28422, 29590, 30334, 30357,
                     31213, 32902, 33836, 36014, 36019, 36144, 37390, 38057, 38433, 38470, 38476,
                     38520, 38646, 63851, 63859, 63864, 63922, 63953, 63955, 63997, 131073, 131172,
                     131298, 131361, 133418, 133507, 133516, 133532, 133866, 133885, 133913, 140176,
                     141720, 146203, 156269, 194704]

// The code point of a one-character string (one unit, or a surrogate pair).
function pyCpOf(ch) {
    var c = ch.charCodeAt(0)
    if (ch.length < 2) { return c }
    return (c - 55296) * 1024 + ch.charCodeAt(1) - 56320 + 65536
}

function pyCpAlpha(c) {
    if (c < 128) { return (c >= 65 && c <= 90) || (c >= 97 && c <= 122) }
    return jcIsLetter(c)
}

function pyCpDecimal(c) {
    if (c < 128) { return c >= 48 && c <= 57 }
    return jcIsDigit(c)
}

function pyCpDigit(c) {
    if (pyCpDecimal(c)) { return true }
    var i = 0
    while (i < pyDigitRuns.length) {
        if (c < pyDigitRuns[i]) { return false }
        if (c <= pyDigitRuns[i + 1]) { return true }
        i = i + 2
    }
    return false
}

function pyCpNumeric(c) {
    if (c < 128) { return c >= 48 && c <= 57 }
    if (jcIsLetter(c)) { return pyHanNumerals.indexOf(c) >= 0 }
    return jcClass(c) % 8 == 6 || jcNumericValue(c) != 0 - 1
}

// \N{NAME}: the code point of a Unicode character name, matched without regard
// to case as CPython does, or -1. A small table of the names a program is likely
// to spell out: the full one is ~34,000 names and 1 MB of text beyond the
// algorithmic ones, so a name outside it is refused as unknown. Used by string
// literals and by the `re` module (pyRePat). Twins: pyUniNames in
// python-to-llvm-ir.abnf and pyUniNameCp in abnf/jsrtregexpy.go.
var pyUniNameList = ["BULLET", "DEGREE SIGN", "EURO SIGN", "GREEK SMALL LETTER ALPHA",
                     "GREEK CAPITAL LETTER DELTA", "AMPERSAND", "LEFT CURLY BRACKET",
                     "RIGHT CURLY BRACKET", "EXCLAMATION MARK", "LATIN SMALL LETTER A WITH DIAERESIS",
                     "GREEK SMALL LETTER PI", "EM DASH", "SNOWMAN", "NO-BREAK SPACE"]
var pyUniNameCps = [8226, 176, 8364, 945, 916, 38, 123, 125, 33, 228, 960, 8212, 9731, 160]

function pyUniNameCp(name) {
    var i = pyUniNameList.indexOf(name.toUpperCase())
    return i < 0 ? 0 - 1 : pyUniNameCps[i]
}
