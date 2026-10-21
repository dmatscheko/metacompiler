// php-preg.js -- the two natives PHP's preg_* functions (lib/php-stdlib.php)
// are written over, on lib/regex.js's C dialect: __mec_pregc compiles a pattern
// and __mec_pregx runs one pcre2_match. For php-interpreter.abnf (include) and
// lib/php-rt.metajs (import), after lib/php-strings.js, lib/regex.js,
// lib/java-chars.js and lib/java-regex.js. Go twin: abnf/jsrtphppreg.go.
//
// A pattern is named by its KEY: the modifiers that reach PCRE2, "/", and the
// pattern between the delimiters. Without u the engine reads the pattern and
// the subject as BYTES, one unit each, so every offset is already a byte offset;
// with u it reads their text, and phpRxOff maps a unit to its byte offset.

// The subject the last run read, and the engine's reading of it.
var phpRxS = ""
var phpRxUtf = false
var phpRxHave = false
var phpRxT = ""
var phpRxOff = []
var phpRxBad = false

// The bytes of s, one unit each.
function phpRxBytes(s) {
    var ascii = true
    var i = 0
    while (i < s.length) {
        if (s.charCodeAt(i) > 127) { ascii = false }
        i = i + 1
    }
    if (ascii) { return s }
    var bs = phsBytes(s)
    var out = ""
    i = 0
    while (i < bs.length) {
        out = out + String.fromCharCode(bs[i])
        i = i + 1
    }
    return out
}

// Does s hold a byte that is not part of a character (not valid UTF-8)?
function phpRxHasByte(s) {
    var i = 0
    while (i < s.length) {
        if (phsIsByte(s.charCodeAt(i))) { return true }
        i = i + 1
    }
    return false
}

// Does key read the pattern and the subject as UTF-8: the u modifier, or
// (*UTF) among the options the pattern starts with?
function phpRxKeyUtf(key) {
    var k = key.indexOf("/")
    if (key.substring(0, k).indexOf("u") >= 0) { return true }
    var i = k + 1
    while (key.charCodeAt(i) == 40 && key.charCodeAt(i + 1) == 42) {
        var e = key.indexOf(")", i)
        if (e < 0) { return false }
        if (key.substring(i + 2, e) == "UTF" || key.substring(i + 2, e) == "UTF8") { return true }
        i = e + 1
    }
    return false
}

// The compiled pattern of key; null under UTF when the pattern is not UTF-8.
function phpRxRe(key) {
    var k = key.indexOf("/")
    var mods = key.substring(0, k)
    var pat = key.substring(k + 1, key.length)
    if (!phpRxKeyUtf(key)) { return rxGet(phpRxBytes(pat), "C" + mods) }
    if (phpRxHasByte(pat)) { return null }
    return rxGet(pat, "C" + mods)
}

// __mec_pregc: the name of each group by number, "" for the whole match and an
// unnamed group; or PCRE2's error text.
function phpPregCompile(key) {
    var re = phpRxRe(key)
    if (re == null) { return "UTF-8 error" }
    if (!re.ok) { return re.err }
    var out = [""]
    var i = 1
    while (i <= re.ngroups) {
        out.push(rxNameAt(re, i))
        i = i + 1
    }
    return out
}

// Reads s as the engine does, for utf or not.
function phpRxText(s, utf) {
    if (phpRxHave && utf == phpRxUtf && s == phpRxS) { return }
    phpRxS = s
    phpRxUtf = utf
    phpRxHave = true
    phpRxOff = []
    phpRxBad = false
    if (!utf) {
        phpRxT = phpRxBytes(s)
        return
    }
    phpRxT = s
    phpRxBad = phpRxHasByte(s)
    var off = []
    var b = 0
    var ascii = true
    var i = 0
    while (i < s.length) {
        var c = s.charCodeAt(i)
        off.push(b)
        if (c < 128) { b = b + 1 }
        else if (c < 2048) {
            b = b + 2
            ascii = false
        } else if (c >= 55296 && c <= 56319 && i + 1 < s.length) {
            off.push(b)
            b = b + 4
            i = i + 1
            ascii = false
        } else {
            b = b + 3
            ascii = false
        }
        i = i + 1
    }
    off.push(b)
    if (!ascii) { phpRxOff = off }
}

// The unit at byte offset b, or -1 when b is inside a character.
function phpRxUnit(b) {
    var off = phpRxOff
    if (off.length == 0) { return b }
    var lo = 0
    var hi = off.length - 1
    while (lo < hi) {
        var mid = Math.floor((lo + hi) / 2)
        if (off[mid] < b) { lo = mid + 1 } else { hi = mid }
    }
    if (off[lo] != b) { return 0 - 1 }
    return lo
}

// __mec_pregx: the match of key in s from byte offset pos, as byte offsets
// (-1 for a group that did not take part), [] for none, or the PREG error code
// (1 a recursion that would loop, 2 the step limit, 4 the subject is not
// UTF-8, 5 the offset is inside a character). A pos of
// -o - 1 asks for the match after an empty one at o: a non-empty one at o, else
// the search from the next character (rxSearchAdv).
function phpPregExec(key, s, pos) {
    var re = phpRxRe(key)
    phpRxText(s, phpRxKeyUtf(key))
    if (phpRxBad) { return 4 }
    var retry = pos < 0
    var at = phpRxUnit(retry ? 0 - pos - 1 : pos)
    if (at < 0) { return 5 }
    var m = retry ? rxSearchAdv(re, phpRxT, at, true) : rxSearch(re, phpRxT, at)
    var out = []
    if (m == null) { return out }
    if (m.begin < 0) { return m.begin == 0 - 4 ? 1 : 2 }
    var caps = m.caps
    var off = phpRxOff
    var i = 0
    while (i < caps.length) {
        out.push(caps[i] < 0 ? 0 - 1 : off.length == 0 ? caps[i] : off[caps[i]])
        i = i + 1
    }
    return out
}
