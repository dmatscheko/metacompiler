// python-regex.js -- the `re` module's own layer over lib/regex.js's P dialect:
// Python's (?P...) spellings, and re.error (re.PatternError) for a pattern the
// engine rejects - CPython's message, its position, and the exception's fields.
// For python-interpreter.abnf (include) and languages/lib/python-rt.metajs
// (import), each after lib/regex.js. Go twin: abnf/jsrtregexpy.go.

// pyRePat translates Python's P-spellings into the neutral engine's:
// (?P<name>...) is (?<name>...) and (?P=name) is the named backreference \k<name>.
// Escapes and character classes are stepped over, so `[(?P<]` is left alone. An
// unknown name is the ENGINE's error to report, which is why nothing is counted
// here. Answers {out, err}; err is "" on success.
function pyRePat(src) {
    var out = ""
    var inCls = false
    var i = 0
    while (i < src.length) {
        var ch = src.charAt(i)
        if (ch == "\\") {
            out = out + src.substring(i, i + 2)
            i = i + 2
            continue
        }
        if (inCls) {
            if (ch == "]") { inCls = false }
            out = out + ch
            i = i + 1
            continue
        }
        if (ch == "[") {
            inCls = true
            out = out + ch
            i = i + 1
            continue
        }
        if (ch != "(") {
            out = out + ch
            i = i + 1
            continue
        }
        if (src.substring(i, i + 4) == "(?P<") {
            var j = i + 4
            while (j < src.length && src.charAt(j) != ">") { j = j + 1 }
            if (j >= src.length) { return {out: "", err: "unterminated group name"} }
            out = out + "(?<" + src.substring(i + 4, j) + ">"
            i = j + 1
            continue
        }
        if (src.substring(i, i + 4) == "(?P=") {
            var k = i + 4
            while (k < src.length && src.charAt(k) != ")") { k = k + 1 }
            if (k >= src.length) { return {out: "", err: "unterminated group name"} }
            out = out + "\\k<" + src.substring(i + 4, k) + ">"
            i = k + 1
            continue
        }
        out = out + ch
        i = i + 1
    }
    return {out: out, err: ""}
}

// The error the whole pipeline (pyRePat, then the engine under the neutral flags
// nf) reports for pattern p; "" when it compiles.
function pyReCompileErr(p, nf) {
    var tr = pyRePat(p)
    if (tr.err != "") { return tr.err }
    return rxCompile(tr.out, nf).err
}

// Where an end-of-pattern error points: for a group, the innermost '(' still
// open; for a set, the '[' still open. -1 when the scan finds none. A ']' right
// after '[' or '[^' is a literal, as in CPython.
function pyReOpenAt(p, group) {
    var opens = []
    var cls = -1
    var i = 0
    while (i < p.length) {
        var ch = p.charAt(i)
        if (ch == "\\") {
            i = i + 2
            continue
        }
        if (cls >= 0) {
            var first = cls + 1
            if (p.charAt(first) == "^") { first = first + 1 }
            if (ch == "]" && i > first) { cls = -1 }
        } else if (ch == "[") {
            cls = i
        } else if (ch == "(") {
            opens.push(i)
        } else if (ch == ")" && opens.length > 0) {
            opens.pop()
        }
        i = i + 1
    }
    if (!group) { return cls }
    if (opens.length == 0) { return -1 }
    return opens[opens.length - 1]
}

// CPython's message and position (-1 for None) for the engine error err on
// pattern pat. Most kinds are located by the shortest prefix of the pattern
// that fails the same way; an end-of-pattern kind by pyReOpenAt. Any other kind
// keeps the engine's text, at the engine's own index at (-1: none).
function pyReErrInfo(pat, nf, err, at) {
    var n = pat.length
    if (err == "unterminated group") {
        return {msg: "missing ), unterminated subpattern", pos: pyReOpenAt(pat, true)}
    }
    if (err == "unterminated character class") {
        return {msg: "unterminated character set", pos: pyReOpenAt(pat, false)}
    }
    if (err == "trailing backslash in pattern") { return {msg: "bad escape (end of pattern)", pos: n - 1} }
    var k = 1
    while (k < n && pyReCompileErr(pat.substring(0, k), nf) != err) { k = k + 1 }
    if (err == "unmatched ) in pattern") { return {msg: "unbalanced parenthesis", pos: k - 1} }
    if (err == "quantifier with nothing to repeat") { return {msg: "nothing to repeat", pos: k - 1} }
    if (err == "multiple repeat") { return {msg: err, pos: k - 1} }
    if (err == "quantifier bounds are out of order") {
        return {msg: "min repeat greater than max repeat", pos: pyReLastAt(pat, "{", k) + 1}
    }
    if (err == "character class range is out of order") {
        var dash = pyReLastAt(pat, "-", k)
        var lo = dash - 1
        if (lo > 0 && pat.charAt(lo - 1) == "\\") { lo = lo - 1 }
        return {msg: "bad character range " + pat.substring(lo, k), pos: lo}
    }
    if (err == "unterminated group name") {
        var close = pat.substring(k - 4, k) == "(?P<" ? ">" : ")"
        return {msg: "missing " + close + ", unterminated name", pos: k}
    }
    if (err.indexOf("unknown group name '") == 0) {
        return {msg: err, pos: k - 1 - (err.length - 21)}
    }
    if (err.indexOf("invalid group reference ") == 0) {
        return {msg: err, pos: k - (err.length - 24)}
    }
    if (err.indexOf("unsupported group form (?") == 0) {
        return {msg: "unknown extension ?" + pat.charAt(k - 1), pos: k - 2}
    }
    return {msg: err, pos: at}
}

// The index of the last ch in p before index k, or -1.
function pyReLastAt(p, ch, k) {
    var i = k - 1
    while (i >= 0 && p.charAt(i) != ch) { i = i - 1 }
    return i
}

// str() of a re.error built from msg, pattern and pos (pos -1: None): CPython
// appends the position, and the line and column once the pattern spans lines.
function pyReErrText(msg, pattern, pos) {
    if (pos < 0) { return msg }
    var out = msg + " at position " + pos
    if (pattern.indexOf("\n") < 0) { return out }
    return out + " (line " + pyReErrLine(pattern, pos) + ", column " + pyReErrCol(pattern, pos) + ")"
}

// re.error's lineno and colno: 1-based, counted up to pos.
function pyReErrLine(pattern, pos) {
    var line = 1
    var i = 0
    while (i < pos && i < pattern.length) {
        if (pattern.charAt(i) == "\n") { line = line + 1 }
        i = i + 1
    }
    return line
}

function pyReErrCol(pattern, pos) {
    var i = pos - 1
    if (i >= pattern.length) { i = pattern.length - 1 }
    while (i >= 0 && pattern.charAt(i) != "\n") { i = i - 1 }
    return pos - i
}
