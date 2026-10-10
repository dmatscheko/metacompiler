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
//   - two NUMBER literals fold, as V8's parser folds them: "(1 + 2)" reads "3", and a
//     signed literal is one literal, "-1"; two string literals concatenate
//   - an assignment prints its target alone, a conditional "(intermediate value)"
//     three times, a comma "(a , b)", a call "f(...)", a string-literal index as a
//     dotted name ('o["a b"]' reads "o.a b")
//
// cpPrint answers null for anything it does not parse - a function, a class, an
// object, array or template literal, '=>', '?.', a regular expression - and the
// grammar keeps its own answer then. The callee text reaches the grammars with its
// whitespace removed (docs/abnf-dialect-gotchas.md), so a word operator is glued to
// its operand ('typeofx', 'ainb') and prints as the identifier it now looks like.

var CP_PUNCT = [">>>=", "===", "!==", "**=", "<<=", ">>=", ">>>", "&&=", "||=", "??=", "...",
    "=>", "==", "!=", "<=", ">=", "&&", "||", "??", "?.", "++", "--", "+=", "-=", "*=", "/=",
    "%=", "&=", "|=", "^=", "**", "<<", ">>", "+", "-", "*", "/", "%", "<", ">", "=", "!",
    "~", "&", "|", "^", "?", ":", ",", ".", "(", ")", "[", "]"]
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

function cpIsIdStart(c) { return (c >= 97 && c <= 122) || (c >= 65 && c <= 90) || c == 95 || c == 36 }
function cpIsDigit(c) { return c >= 48 && c <= 57 }

function cpLex(s) {
    var out = []
    var i = 0
    while (i < s.length) {
        var c = s.charCodeAt(i)
        if (c == 32 || c == 9 || c == 10 || c == 13) { i = i + 1; continue }
        var start = i
        if (cpIsIdStart(c)) {
            while (i < s.length && (cpIsIdStart(s.charCodeAt(i)) || cpIsDigit(s.charCodeAt(i)))) { i = i + 1 }
            out.push({k: "id", v: s.substring(start, i)})
            continue
        }
        if (cpIsDigit(c) || (c == 46 && i + 1 < s.length && cpIsDigit(s.charCodeAt(i + 1)))) {
            // A plain decimal literal only; a hex, octal, binary, BigInt or separated
            // one is left to the grammar's own answer.
            while (i < s.length && cpIsDigit(s.charCodeAt(i))) { i = i + 1 }
            if (i < s.length && s.charCodeAt(i) == 46) {
                i = i + 1
                while (i < s.length && cpIsDigit(s.charCodeAt(i))) { i = i + 1 }
            }
            if (i < s.length && (s.charCodeAt(i) == 101 || s.charCodeAt(i) == 69)) { return null }
            if (i < s.length && (cpIsIdStart(s.charCodeAt(i)) || s.charCodeAt(i) == 95)) { return null }
            out.push({k: "num", v: s.substring(start, i)})
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
        var hit = ""
        var j = 0
        while (j < CP_PUNCT.length && hit == "") {
            var pt = CP_PUNCT[j]
            if (s.substring(i, i + pt.length) == pt) { hit = pt }
            j = j + 1
        }
        if (hit == "" || hit == "=>" || hit == "?." || hit == "...") { return null }
        out.push({k: "p", v: hit})
        i = i + hit.length
    }
    return out
}

function cpPeek(p) {
    var t = p["t"]
    if (p["i"] < t.length) { return t[p["i"]] }
    return {k: "end", v: ""}
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

// Folding is limited to results whose text is unambiguous in every script host: a
// safe integer. Anything else stays an unfolded operation.
function cpFold(op, a, b) {
    var v = 0
    if (op == "+") { v = a + b }
    else if (op == "-") { v = a - b }
    else if (op == "*") { v = a * b }
    else if (op == "**") { v = Math.pow(a, b) }
    else { return null }
    if (Math.floor(v) !== v || Math.abs(v) > 9007199254740991) { return null }
    return {k: "num", v: "" + v, f: v}
}

function cpUnary(p) {
    var tok = cpPeek(p)
    var v = tok["v"]
    if (tok["k"] == "p" && (v == "!" || v == "-" || v == "+" || v == "~")) {
        p["i"] = p["i"] + 1
        var x = cpUnary(p)
        if ((v == "-" || v == "+") && x["k"] == "num") {
            var nv = cpNumOf(x)
            if (v == "-") { nv = 0 - nv }
            if (Math.floor(nv) === nv && nv !== 0) { return {k: "num", v: "" + nv, f: nv} }
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
        if (w == "function" || w == "class" || w == "new" || w == "async" || w == "await" || w == "yield" || w == "super") {
            return cpBad(p)
        }
        p["i"] = p["i"] + 1
        return {k: "id", v: w}
    }
    if (k == "num" || k == "str") {
        p["i"] = p["i"] + 1
        return tok
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

function cpShow(n) {
    var k = n["k"]
    if (k == "id" || k == "num") { return n["v"] }
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
