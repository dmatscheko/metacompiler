// js-callee.js -- the callee TEXT of node's "X is not a function" TypeError, for the
// expressions inside a parenthesized callee or an index: '(o.a + 1)()' reads
// "(o.a + 1) is not a function" and 'xs[i+1]()' reads "xs[(i + 1)] is not a function".
//
// V8 does not quote the source there; it re-prints its AST (CallPrinter), and this
// file re-prints the same shape from the callee's text. Included by the js and
// typescript grammars, both halves (include("lib/js-callee.js")), in the restricted
// subset of regex.js's head comment. Every name is prefixed cp.
//
// What the printing does, each rule read off node 24:
//   - a binary or comparison operator is parenthesized and spaced: "(a + b)"; a run
//     of the SAME left-associative operator stays flat, "(i - 2 - 3)", any other
//     nesting shows its grouping, "((i % 2) * 3)"
//   - a unary operator is parenthesized, "(-i)", "(!a)", "(typeof a)", and so is an
//     update, "(i++)", "(++i)"
//   - a number literal prints as its value, "2e3" reads "2000" and "1e21" "1e+21";
//     two NUMBER literals fold, as V8's parser folds them: "(1 + 2)" reads "3",
//     "(1 / 4)" "0.25", and a signed literal is one literal, "-1"; two string
//     literals concatenate; a BigInt literal prints "(intermediate value)"
//   - an assignment prints its target alone, a conditional "(intermediate value)"
//     three times, a comma "(a , b)", a call "f(...)", a string-literal index as a
//     dotted name ('o["a b"]' reads "o.a b")
//   - an array literal prints its elements, "[x,(...ys)]", and a hole as
//     "(intermediate value)"; an object literal prints "(intermediate value)" once
//     per property, "{(intermediate value)}"; a template literal prints its
//     substitutions only, and one without any as a string; a regular expression
//     prints as written
//   - a function, an arrow or a class prints "(intermediate value)" once per
//     statement of its body (at least once), plus once more when its parameter list
//     is not simple (a default, a pattern or a rest), V8's parameter-init block
//
// cpPrint answers null for anything it does not parse - '?.', 'new', a template or
// string with an escape, a body statement it cannot count (a nested block or a
// keyword statement, or a line break with no ';' before it) - and the grammar keeps
// its own answer then.
//
// THE SOURCE AS WRITTEN. up.in arrives with the whitespace between its tokens removed
// (docs/abnf-dialect-gotchas.md), which glues a word operator to its operand
// ('typeofx', 'ainb'). So the call-site tags also hand over up.pos, the node's END
// byte offset, and cpSrc reads the span back out of the source (c.srcSlice).

var CP_PUNCT = [">>>=", "===", "!==", "**=", "<<=", ">>=", ">>>", "&&=", "||=", "??=", "...",
    "=>", "==", "!=", "<=", ">=", "&&", "||", "??", "?.", "++", "--", "+=", "-=", "*=", "/=",
    "%=", "&=", "|=", "^=", "**", "<<", ">>", "+", "-", "*", "/", "%", "<", ">", "=", "!",
    "~", "&", "|", "^", "?", ":", ",", ".", "(", ")", "[", "]", "{", "}", ";"]
// Binding power of each binary operator; ** is the one right-associative level.
var CP_BIN_OPS = ["??", "||", "&&", "|", "^", "&", "==", "!=", "===", "!==", "<", ">", "<=", ">=",
    "instanceof", "in", "<<", ">>", ">>>", "+", "-", "*", "/", "%", "**"]
var CP_BIN_PREC = [1, 2, 3, 4, 5, 6, 7, 7, 7, 7, 8, 8, 8, 8, 8, 8, 9, 9, 9, 10, 10, 11, 11, 11, 12]

// The printed form of the expression text src, or null.
function cpPrint(src) {
    var toks = cpLex(src)
    if (toks === null) { return null }
    var p = {}
    p["t"] = toks
    p["i"] = 0
    p["bad"] = false
    var n = cpExpr(p)
    if (p["bad"] || p["i"] != toks.length) { return null }
    return cpShow(n)
}

// An index: "." + name for a string literal, "[" + expr + "]" otherwise; null when the
// text does not parse.
function cpIndex(src) {
    var toks = cpLex(src)
    if (toks === null) { return null }
    var p = {}
    p["t"] = toks
    p["i"] = 0
    p["bad"] = false
    var n = cpExpr(p)
    if (p["bad"] || p["i"] != toks.length) { return null }
    if (n["k"] == "str") { return "." + n["v"] }
    return "[" + cpShow(n) + "]"
}

// The source text of a node whose tokens are txt and whose end offset is end, or null:
// when txt cannot hold a glued word operator (the common case, which costs nothing),
// and when the span does not line up - a line comment inside it, or a node that is not
// from the current source. Walking backwards from the end, every character of txt is
// matched and the whitespace and block comments between them skipped.
function cpSrc(txt, end) {
    if (!cpGlued(txt)) { return null }
    var from = end - txt.length * 4 - 256
    var src = c.srcSlice(from, end)
    var i = src.length - 1
    var j = txt.length - 1
    while (j >= 0) {
        if (i < 0) { return null }
        var sc = src.charCodeAt(i)
        if (sc == txt.charCodeAt(j)) { j = j - 1 }
        else if (sc == 47 && i > 0 && src.charCodeAt(i - 1) == 42) {
            var k = i - 2
            while (k > 0 && !(src.charCodeAt(k - 1) == 47 && src.charCodeAt(k) == 42)) { k = k - 1 }
            if (k <= 0) { return null }
            i = k - 1
        }
        else if (!(sc == 32 || sc == 9 || sc == 10 || sc == 13)) { return null }
        i = i - 1
    }
    return src.substring(i + 1)
}

// The text a call-site tag records for a primary or an index suffix: the source as
// written for a parenthesized primary, an object literal and a bracketed index (the
// forms cpPrint and cpIndex re-print), up.in for everything else.
function cpTxt(txt, end) {
    if (txt.length < 2) { return txt }
    var c0 = txt.charCodeAt(0)
    if (c0 != 40 && c0 != 91 && c0 != 63 && c0 != 123) { return txt }
    var src = cpSrc(txt, end)
    if (src === null) { return txt }
    return src
}

// Whether txt may hold a word operator (typeof, void, delete, in, instanceof) or a
// function, whose words and line breaks the printing reads.
function cpGlued(t) {
    return t.indexOf("in") >= 0 || t.indexOf("typeof") >= 0 || t.indexOf("void") >= 0 || t.indexOf("delete") >= 0 ||
        t.indexOf("=>") >= 0 || t.indexOf("function") >= 0 || t.indexOf("class") >= 0 || t.indexOf("async") >= 0
}

// The JavaScript text of a number, or null when the host printer cannot be trusted
// with it: goja's is not always exact, and a text that does not read back is declined.
function cpNumText(v) {
    if (v !== v) { return "NaN" }
    if (v - v !== 0) { return v > 0 ? "Infinity" : "-Infinity" }
    var t = "" + v
    if (parseFloat(t) !== v) { return null }
    return t
}

// The radix a 0x / 0o / 0b prefix letter names, else 0.
function cpRadix(c) {
    if (c == 120 || c == 88) { return 16 }
    if (c == 111 || c == 79) { return 8 }
    if (c == 98 || c == 66) { return 2 }
    return 0
}
function cpDigitVal(c) {
    if (c >= 48 && c <= 57) { return c - 48 }
    if (c >= 97 && c <= 102) { return c - 87 }
    if (c >= 65 && c <= 70) { return c - 55 }
    return 0 - 1
}

// The text V8 prints for a callee primary that has no parentheses: a lone number
// literal as its value ("2e3" reads "2000", "0x10" "16"), a lone string literal in
// double quotes, an object, array, template or regular-expression literal or an arrow
// re-printed (cpPrint), anything else as it is.
function cpLiteral(t) {
    var c0 = t.charCodeAt(0)
    if (c0 == 123 || c0 == 91 || c0 == 96 || c0 == 47 || t.indexOf("=>") >= 0) {
        var pr = cpPrint(t)
        if (pr !== null) { return pr }
        return t
    }
    if (!cpIsDigit(c0) && c0 != 46 && c0 != 39) { return t }
    var toks = cpLex(t)
    if (toks === null || toks.length != 1) { return t }
    if (toks[0]["k"] == "big") { return "(intermediate value)" }
    if (toks[0]["k"] != "num" && toks[0]["k"] != "str") { return t }
    return cpShow(toks[0])
}

function cpIsIdStart(c) { return (c >= 97 && c <= 122) || (c >= 65 && c <= 90) || c == 95 || c == 36 }
function cpIsDigit(c) { return c >= 48 && c <= 57 }
// Whether the digits ending at i are a BigInt literal's: an 'n' that ends the word.
function cpBigEnd(s, i) {
    if (i >= s.length || s.charCodeAt(i) != 110) { return false }
    return i + 1 >= s.length || !(cpIsIdStart(s.charCodeAt(i + 1)) || cpIsDigit(s.charCodeAt(i + 1)))
}

// The tokens of s, or null. A token that follows a line break carries nl: true, which
// is what tells a statement boundary without a ';' (cpBodyCount declines it).
function cpLex(s) {
    var out = []
    var nlAt = []
    var i = 0
    while (i < s.length) {
        var c = s.charCodeAt(i)
        if (c == 32 || c == 9 || c == 10 || c == 13) {
            if (c == 10 || c == 13) { nlAt.push(out.length) }
            i = i + 1
            continue
        }
        if (c == 47 && i + 1 < s.length && s.charCodeAt(i + 1) == 42) {
            i = i + 2
            while (i + 1 < s.length && !(s.charCodeAt(i) == 42 && s.charCodeAt(i + 1) == 47)) {
                if (s.charCodeAt(i) == 10) { nlAt.push(out.length) }
                i = i + 1
            }
            if (i + 1 >= s.length) { return null }
            i = i + 2
            continue
        }
        var start = i
        if (cpIsIdStart(c)) {
            while (i < s.length && (cpIsIdStart(s.charCodeAt(i)) || cpIsDigit(s.charCodeAt(i)))) { i = i + 1 }
            out.push({k: "id", v: s.substring(start, i)})
            continue
        }
        if (c == 48 && i + 1 < s.length && cpRadix(s.charCodeAt(i + 1)) > 0) {
            var rx = cpRadix(s.charCodeAt(i + 1))
            var rv = 0
            i = i + 2
            while (i < s.length && cpDigitVal(s.charCodeAt(i)) >= 0 && cpDigitVal(s.charCodeAt(i)) < rx) {
                rv = rv * rx + cpDigitVal(s.charCodeAt(i))
                i = i + 1
            }
            if (i == start + 2) { return null }
            if (cpBigEnd(s, i)) { out.push({k: "big"}); i = i + 1; continue }
            // Past 2^53 the running sum is no longer exact.
            if (rv > 9007199254740991) { return null }
            if (i < s.length && (cpIsIdStart(s.charCodeAt(i)) || cpIsDigit(s.charCodeAt(i)))) { return null }
            out.push({k: "num", v: "" + rv, f: rv})
            continue
        }
        if (cpIsDigit(c) || (c == 46 && i + 1 < s.length && cpIsDigit(s.charCodeAt(i + 1)))) {
            // A decimal literal; a legacy octal ('010'), a BigInt or a separated one is
            // left to the grammar's own answer.
            if (c == 48 && i + 1 < s.length && cpIsDigit(s.charCodeAt(i + 1))) { return null }
            while (i < s.length && cpIsDigit(s.charCodeAt(i))) { i = i + 1 }
            if (c != 46 && cpBigEnd(s, i)) { out.push({k: "big"}); i = i + 1; continue }
            if (i < s.length && s.charCodeAt(i) == 46) {
                i = i + 1
                while (i < s.length && cpIsDigit(s.charCodeAt(i))) { i = i + 1 }
            }
            if (i < s.length && (s.charCodeAt(i) == 101 || s.charCodeAt(i) == 69)) {
                var ep = i + 1
                if (ep < s.length && (s.charCodeAt(ep) == 43 || s.charCodeAt(ep) == 45)) { ep = ep + 1 }
                if (ep >= s.length || !cpIsDigit(s.charCodeAt(ep))) { return null }
                i = ep
                while (i < s.length && cpIsDigit(s.charCodeAt(i))) { i = i + 1 }
            }
            if (i < s.length && (cpIsIdStart(s.charCodeAt(i)) || s.charCodeAt(i) == 95)) { return null }
            var nv = parseFloat(s.substring(start, i))
            var nt = cpNumText(nv)
            if (nt === null) { return null }
            out.push({k: "num", v: nt, f: nv})
            continue
        }
        if (c == 34 || c == 39) {
            i = i + 1
            while (i < s.length && s.charCodeAt(i) != c) {
                if (s.charCodeAt(i) == 92) { return null }
                i = i + 1
            }
            if (i >= s.length) { return null }
            out.push({k: "str", v: s.substring(start + 1, i)})
            i = i + 1
            continue
        }
        if (c == 96) {
            var tl = cpLexTemplate(s, i)
            if (tl === null) { return null }
            out.push(tl["tok"])
            i = tl["end"]
            continue
        }
        if (c == 47 && cpRegexMayStart(out)) {
            var re = cpRegexEnd(s, i)
            if (re < 0) { return null }
            out.push({k: "rx", v: s.substring(start, re)})
            i = re
            continue
        }
        var hit = ""
        var j = 0
        while (j < CP_PUNCT.length && hit == "") {
            var pt = CP_PUNCT[j]
            if (s.substring(i, i + pt.length) == pt) { hit = pt }
            j = j + 1
        }
        if (hit == "" || hit == "?.") { return null }
        out.push({k: "p", v: hit})
        i = i + hit.length
    }
    var n = 0
    while (n < nlAt.length) {
        if (nlAt[n] < out.length) { out[nlAt[n]]["nl"] = true }
        n = n + 1
    }
    return out
}

// A '/' starts a regular expression where an operand is expected: first, or after an
// operator or an opening bracket.
function cpRegexMayStart(out) {
    if (out.length == 0) { return true }
    var last = out[out.length - 1]
    if (last["k"] != "p") { return false }
    var v = last["v"]
    return v != ")" && v != "]" && v != "}"
}

// The offset just past the regular expression starting at s[i] (its flags included),
// or -1. A '/' inside a class or after a backslash does not end it.
function cpRegexEnd(s, i) {
    var j = i + 1
    var inClass = false
    while (j < s.length) {
        var c = s.charCodeAt(j)
        if (c == 10 || c == 13) { return 0 - 1 }
        if (c == 92) { j = j + 1 }
        else if (c == 91) { inClass = true }
        else if (c == 93) { inClass = false }
        else if (c == 47 && !inClass) { break }
        j = j + 1
    }
    if (j >= s.length) { return 0 - 1 }
    j = j + 1
    while (j < s.length && cpIsIdStart(s.charCodeAt(j))) { j = j + 1 }
    return j
}

// The template literal starting at s[i] as {tok, end}, or null: a "tpl" token whose
// parts are the token lists of its substitutions, and whose v is its text when it has
// none. A backslash declines it, as in a string.
function cpLexTemplate(s, i) {
    var j = i + 1
    var parts = []
    var text = ""
    while (j < s.length && s.charCodeAt(j) != 96) {
        var c = s.charCodeAt(j)
        if (c == 92) { return null }
        if (c == 36 && j + 1 < s.length && s.charCodeAt(j + 1) == 123) {
            var k = j + 2
            var depth = 1
            while (k < s.length && depth > 0) {
                var d = s.charCodeAt(k)
                if (d == 123) { depth = depth + 1 }
                else if (d == 125) { depth = depth - 1 }
                else if (d == 96) {
                    k = k + 1
                    while (k < s.length && s.charCodeAt(k) != 96) { k = k + 1 }
                }
                k = k + 1
            }
            if (depth > 0) { return null }
            var sub = cpLex(s.substring(j + 2, k - 1))
            if (sub === null) { return null }
            parts.push(sub)
            j = k
        } else {
            text = text + s.charAt(j)
            j = j + 1
        }
    }
    if (j >= s.length) { return null }
    var r = {}
    r["tok"] = {k: "tpl", v: text, parts: parts}
    r["end"] = j + 1
    return r
}

function cpPeek(p) { return cpAt(p, 0) }
function cpAt(p, k) {
    var t = p["t"]
    if (p["i"] + k < t.length) { return t[p["i"] + k] }
    return {k: "end", v: ""}
}
// The index of the bracket closing the one at token i, or -1.
function cpMatch(t, i) {
    var depth = 0
    var j = i
    while (j < t.length) {
        var tok = t[j]
        if (tok["k"] == "p") {
            var v = tok["v"]
            if (v == "(" || v == "[" || v == "{") { depth = depth + 1 }
            else if (v == ")" || v == "]" || v == "}") {
                depth = depth - 1
                if (depth == 0) { return j }
            }
        }
        j = j + 1
    }
    return 0 - 1
}
function cpIsP(tok, v) { return (tok["k"] == "p" || tok["k"] == "id") && tok["v"] == v }
function cpBad(p) {
    p["bad"] = true
    return {k: "id", v: ""}
}

function cpExpr(p) {
    var n = cpAssign(p)
    if (!cpIsP(cpPeek(p), ",")) { return n }
    var xs = [n]
    while (cpIsP(cpPeek(p), ",") && !p["bad"]) {
        p["i"] = p["i"] + 1
        xs.push(cpAssign(p))
    }
    return {k: "nary", op: ",", xs: xs}
}

function cpIsAssignOp(v) {
    return v == "=" || v == "+=" || v == "-=" || v == "*=" || v == "/=" || v == "%=" || v == "**=" ||
        v == "<<=" || v == ">>=" || v == ">>>=" || v == "&=" || v == "|=" || v == "^=" ||
        v == "&&=" || v == "||=" || v == "??="
}

function cpAssign(p) {
    var t = cpCond(p)
    var tok = cpPeek(p)
    if (tok["k"] == "p" && cpIsAssignOp(tok["v"])) {
        p["i"] = p["i"] + 1
        cpAssign(p)
        return {k: "asg", t: t}
    }
    return t
}

function cpCond(p) {
    var c = cpBin(p, 1)
    if (!cpIsP(cpPeek(p), "?")) { return c }
    p["i"] = p["i"] + 1
    cpAssign(p)
    if (!cpIsP(cpPeek(p), ":")) { return cpBad(p) }
    p["i"] = p["i"] + 1
    cpAssign(p)
    return {k: "cond"}
}

function cpPrec(tok) {
    if (tok["k"] != "p" && tok["k"] != "id") { return 0 }
    var j = 0
    while (j < CP_BIN_OPS.length) {
        if (CP_BIN_OPS[j] == tok["v"]) { return CP_BIN_PREC[j] }
        j = j + 1
    }
    return 0
}

function cpBin(p, minPrec) {
    var l = cpUnary(p)
    while (!p["bad"]) {
        var tok = cpPeek(p)
        var pr = cpPrec(tok)
        if (pr == 0 || pr < minPrec) { return l }
        var op = tok["v"]
        p["i"] = p["i"] + 1
        var r = anytype // The right operand's node.
        if (op == "**") { r = cpBin(p, pr) } else { r = cpBin(p, pr + 1) }
        l = cpCombine(op, l, r)
    }
    return l
}

// One binary node, folding two number (or, for '+', two string) literals the way V8's
// parser does, and extending a run of the same left-associative operator flat.
function cpCombine(op, l, r) {
    if (l["k"] == "num" && r["k"] == "num") {
        var f = cpFold(op, cpNumOf(l), cpNumOf(r))
        if (f !== null) { return f }
    }
    if (op == "+" && l["k"] == "str" && r["k"] == "str") { return {k: "str", v: l["v"] + r["v"]} }
    if (l["k"] == "nary" && l["op"] == op && op != "**" && l["paren"] !== true) {
        var xs = []
        var i = 0
        while (i < l["xs"].length) {
            xs.push(l["xs"][i])
            i = i + 1
        }
        xs.push(r)
        return {k: "nary", op: op, xs: xs}
    }
    return {k: "nary", op: op, xs: [l, r]}
}

function cpNumOf(n) {
    if (n["f"] !== undefined) { return n["f"] }
    return parseFloat(n["v"])
}

// The arithmetic operators fold; a result cpNumText declines stays an unfolded
// operation. ECMAScript's ** is NaN for a NaN exponent and for (+-1) ** (+-Infinity),
// where C's pow (the frozen host's) answers 1.
function cpFold(op, a, b) {
    var v = 0
    if (op == "+") { v = a + b }
    else if (op == "-") { v = a - b }
    else if (op == "*") { v = a * b }
    else if (op == "/") { v = a / b }
    else if (op == "%") { v = a % b }
    else if (op == "**") {
        if (b !== b || ((a == 1 || a == -1) && b - b !== 0)) { v = b - b }
        else { v = Math.pow(a, b) }
    }
    else { return null }
    var t = cpNumText(v)
    if (t === null) { return null }
    return {k: "num", v: t, f: v}
}

function cpUnary(p) {
    var tok = cpPeek(p)
    var v = tok["v"]
    if (tok["k"] == "p" && (v == "!" || v == "-" || v == "+" || v == "~")) {
        p["i"] = p["i"] + 1
        var x = cpUnary(p)
        if ((v == "-" || v == "+") && x["k"] == "num") {
            var nv = cpNumOf(x)
            if (v == "-") { nv = -nv }
            var nt = cpNumText(nv)
            if (nt !== null) { return {k: "num", v: nt, f: nv} }
        }
        return {k: "un", op: v, x: x}
    }
    if (tok["k"] == "id" && (v == "typeof" || v == "void" || v == "delete")) {
        p["i"] = p["i"] + 1
        return {k: "un", op: v + " ", x: cpUnary(p)}
    }
    if (tok["k"] == "p" && (v == "++" || v == "--")) {
        p["i"] = p["i"] + 1
        return {k: "un", op: v, x: cpUnary(p)}
    }
    var e = cpPostfix(p)
    var after = cpPeek(p)
    if (after["k"] == "p" && (after["v"] == "++" || after["v"] == "--")) {
        p["i"] = p["i"] + 1
        return {k: "post", op: after["v"], x: e}
    }
    return e
}

function cpPostfix(p) {
    var n = cpPrimary(p)
    while (!p["bad"]) {
        var tok = cpPeek(p)
        if (cpIsP(tok, ".")) {
            p["i"] = p["i"] + 1
            var nm = cpPeek(p)
            if (nm["k"] != "id") { return cpBad(p) }
            p["i"] = p["i"] + 1
            n = {k: "mem", o: n, v: nm["v"]}
        } else if (cpIsP(tok, "[")) {
            p["i"] = p["i"] + 1
            var e = cpExpr(p)
            if (!cpIsP(cpPeek(p), "]")) { return cpBad(p) }
            p["i"] = p["i"] + 1
            n = {k: "idx", o: n, e: e}
        } else if (cpIsP(tok, "(")) {
            p["i"] = p["i"] + 1
            if (!cpIsP(cpPeek(p), ")")) {
                cpExpr(p)
                if (!cpIsP(cpPeek(p), ")")) { return cpBad(p) }
            }
            p["i"] = p["i"] + 1
            n = {k: "call", o: n}
        } else {
            return n
        }
    }
    return n
}

function cpPrimary(p) {
    var tok = cpPeek(p)
    var k = tok["k"]
    if (k == "id") {
        var w = tok["v"]
        if (w == "function") { return cpFunction(p) }
        if (w == "class") { return cpClass(p) }
        if (w == "async") {
            var an = cpAt(p, 1)
            if (cpIsP(an, "function") || (an["k"] == "id" && cpIsP(cpAt(p, 2), "=>")) ||
                (cpIsP(an, "(") && cpArrowAt(p["t"], p["i"] + 1))) {
                p["i"] = p["i"] + 1
                if (cpIsP(an, "function")) { return cpFunction(p) }
                return cpArrow(p)
            }
        }
        if (cpIsP(cpAt(p, 1), "=>")) { return cpArrow(p) }
        if (w == "new" || w == "await" || w == "yield" || w == "super") {
            return cpBad(p)
        }
        p["i"] = p["i"] + 1
        return {k: "id", v: w}
    }
    if (k == "num" || k == "str" || k == "rx") {
        p["i"] = p["i"] + 1
        return tok
    }
    if (k == "big") {
        p["i"] = p["i"] + 1
        return {k: "iv", n: 1}
    }
    if (k == "tpl") {
        p["i"] = p["i"] + 1
        var parts = tok["parts"]
        if (parts.length == 0) { return {k: "str", v: tok["v"]} }
        var xs = []
        var pi = 0
        while (pi < parts.length) {
            var q = {}
            q["t"] = parts[pi]
            q["i"] = 0
            q["bad"] = false
            var x = cpExpr(q)
            if (q["bad"] || q["i"] != parts[pi].length) { return cpBad(p) }
            xs.push(x)
            pi = pi + 1
        }
        return {k: "tpl", xs: xs}
    }
    if (cpIsP(tok, "(") && cpArrowAt(p["t"], p["i"])) { return cpArrow(p) }
    if (cpIsP(tok, "[")) { return cpArray(p) }
    if (cpIsP(tok, "{")) {
        // An object literal: one "(intermediate value)" per property, so only the
        // depth-0 commas are counted (a trailing one ends no property).
        var t = p["t"]
        var close = cpMatch(t, p["i"])
        if (close < 0) { return cpBad(p) }
        var props = 0
        if (close > p["i"] + 1) {
            props = 1
            var depth = 0
            var j = p["i"] + 1
            while (j < close) {
                var v = t[j]["v"]
                if (t[j]["k"] == "p") {
                    if (v == "(" || v == "[" || v == "{") { depth = depth + 1 }
                    else if (v == ")" || v == "]" || v == "}") { depth = depth - 1 }
                    else if (v == "," && depth == 0 && j + 1 < close) { props = props + 1 }
                }
                j = j + 1
            }
        }
        p["i"] = close + 1
        return {k: "obj", n: props}
    }
    if (cpIsP(tok, "(")) {
        p["i"] = p["i"] + 1
        var n = cpExpr(p)
        if (!cpIsP(cpPeek(p), ")")) { return cpBad(p) }
        p["i"] = p["i"] + 1
        // A parenthesized operation keeps its own grouping in a longer run.
        if (n["k"] == "nary") { return {k: "nary", op: n["op"], xs: n["xs"], paren: true} }
        return n
    }
    return cpBad(p)
}

// An array literal: its elements, a hole and a spread included.
function cpArray(p) {
    p["i"] = p["i"] + 1
    var xs = []
    while (!p["bad"] && !cpIsP(cpPeek(p), "]")) {
        if (cpIsP(cpPeek(p), ",")) {
            xs.push({k: "iv", n: 1})
            p["i"] = p["i"] + 1
        } else {
            if (cpIsP(cpPeek(p), "...")) {
                p["i"] = p["i"] + 1
                xs.push({k: "spread", x: cpAssign(p)})
            } else {
                xs.push(cpAssign(p))
            }
            if (cpIsP(cpPeek(p), ",")) { p["i"] = p["i"] + 1 }
            else if (!cpIsP(cpPeek(p), "]")) { return cpBad(p) }
        }
    }
    if (p["bad"]) { return cpBad(p) }
    p["i"] = p["i"] + 1
    return {k: "arr", xs: xs}
}

// Whether the "(" at token i opens an arrow function's parameter list.
function cpArrowAt(t, i) {
    var close = cpMatch(t, i)
    return close >= 0 && close + 1 < t.length && cpIsP(t[close + 1], "=>")
}

// Whether the parameter list from token i to the closing bracket at token close is
// NOT simple: a default, a destructuring pattern or a rest parameter.
function cpParamsComplex(t, i, close) {
    var j = i + 1
    while (j < close) {
        var v = t[j]["v"]
        if (t[j]["k"] == "p" && (v == "=" || v == "[" || v == "{" || v == "...")) { return true }
        j = j + 1
    }
    return false
}

// A function's printed count: one per body statement and one for a parameter list
// that is not simple, never fewer than one.
function cpFnNode(stmts, complex) {
    var n = stmts
    if (complex) { n = n + 1 }
    if (n == 0) { n = 1 }
    return {k: "iv", n: n}
}

// An arrow function, at its parameter (a name or a parenthesized list).
function cpArrow(p) {
    var t = p["t"]
    var complex = false
    if (cpIsP(cpPeek(p), "(")) {
        var close = cpMatch(t, p["i"])
        complex = cpParamsComplex(t, p["i"], close)
        p["i"] = close + 1
    } else {
        p["i"] = p["i"] + 1
    }
    p["i"] = p["i"] + 1
    if (cpIsP(cpPeek(p), "{")) { return cpFnNode(cpBodyCount(p), complex) }
    cpAssign(p)
    return cpFnNode(1, complex)
}

// A function expression, at 'function'.
function cpFunction(p) {
    var t = p["t"]
    p["i"] = p["i"] + 1
    if (cpIsP(cpPeek(p), "*")) { p["i"] = p["i"] + 1 }
    if (cpPeek(p)["k"] == "id") { p["i"] = p["i"] + 1 }
    if (!cpIsP(cpPeek(p), "(")) { return cpBad(p) }
    var close = cpMatch(t, p["i"])
    if (close < 0) { return cpBad(p) }
    var complex = cpParamsComplex(t, p["i"], close)
    p["i"] = close + 1
    if (!cpIsP(cpPeek(p), "{")) { return cpBad(p) }
    return cpFnNode(cpBodyCount(p), complex)
}

// A class expression, at 'class': one "(intermediate value)" whatever its body.
function cpClass(p) {
    var t = p["t"]
    var j = p["i"] + 1
    while (j < t.length && !cpIsP(t[j], "{")) {
        if (cpIsP(t[j], "(") || cpIsP(t[j], "[")) {
            j = cpMatch(t, j)
            if (j < 0) { return cpBad(p) }
        }
        j = j + 1
    }
    var close = cpMatch(t, j)
    if (close < 0) { return cpBad(p) }
    p["i"] = close + 1
    return {k: "iv", n: 1}
}

// The number of statements in the block body at the "{" here, which it steps past.
// An empty statement is not one. A statement this cannot delimit by its ';' - a
// nested block or object at statement level, a keyword statement, or a line break
// with no ';' before it (ASI) - declines the whole print.
function cpBodyCount(p) {
    var t = p["t"]
    var close = cpMatch(t, p["i"])
    if (close < 0) {
        cpBad(p)
        return 0
    }
    var n = 0
    var open = false
    var depth = 0
    var j = p["i"] + 1
    while (j < close) {
        var tok = t[j]
        var v = tok["v"]
        if (depth == 0) {
            if (cpIsP(tok, "{") || (tok["k"] == "id" && (v == "if" || v == "for" || v == "while" || v == "do" ||
                v == "switch" || v == "try" || v == "function" || v == "class" || v == "with")) ||
                (open && tok["nl"] === true)) {
                cpBad(p)
                return 0
            }
        }
        if (tok["k"] == "p" && (v == "(" || v == "[" || v == "{")) { depth = depth + 1 }
        else if (tok["k"] == "p" && (v == ")" || v == "]" || v == "}")) { depth = depth - 1 }
        if (depth == 0 && cpIsP(tok, ";")) {
            if (open) { n = n + 1 }
            open = false
        } else {
            open = true
        }
        j = j + 1
    }
    if (open) { n = n + 1 }
    p["i"] = close + 1
    return n
}

function cpShow(n) {
    var k = n["k"]
    if (k == "id" || k == "num" || k == "rx") { return n["v"] }
    if (k == "iv") {
        var iv = ""
        var r = 0
        while (r < n["n"]) {
            iv = iv + "(intermediate value)"
            r = r + 1
        }
        return iv
    }
    if (k == "obj") { return "{" + cpShow({k: "iv", n: n["n"]}) + "}" }
    if (k == "spread") { return "(..." + cpShow(n["x"]) + ")" }
    if (k == "arr" || k == "tpl") {
        var ao = ""
        var e = 0
        while (e < n["xs"].length) {
            if (k == "arr" && e > 0) { ao = ao + "," }
            ao = ao + cpShow(n["xs"][e])
            e = e + 1
        }
        if (k == "arr") { return "[" + ao + "]" }
        return ao
    }
    if (k == "str") { return "\"" + n["v"] + "\"" }
    if (k == "mem") { return cpShow(n["o"]) + "." + n["v"] }
    if (k == "idx") {
        if (n["e"]["k"] == "str") { return cpShow(n["o"]) + "." + n["e"]["v"] }
        return cpShow(n["o"]) + "[" + cpShow(n["e"]) + "]"
    }
    if (k == "call") { return cpShow(n["o"]) + "(...)" }
    if (k == "asg") { return cpShow(n["t"]) }
    if (k == "cond") { return "(intermediate value)(intermediate value)(intermediate value)" }
    if (k == "un") { return "(" + n["op"] + cpShow(n["x"]) + ")" }
    if (k == "post") { return "(" + cpShow(n["x"]) + n["op"] + ")" }
    var xs = n["xs"]
    var out = "(" + cpShow(xs[0])
    var i = 1
    while (i < xs.length) {
        out = out + " " + n["op"] + " " + cpShow(xs[i])
        i = i + 1
    }
    return out + ")"
}
