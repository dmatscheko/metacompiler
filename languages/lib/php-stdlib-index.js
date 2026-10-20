// php-stdlib-index.js -- which lines of lib/php-stdlib.php a program compiles.
//
// Both php grammars include this and compile the library ONE PIECE AT A TIME
// (phpLibRef). A piece is a function or a class. The first mention of a library
// function compiles that function, and a library function its body mentions is
// compiled when that mention is walked, the same way. A class is compiled before
// the walk of a program or an included file that spells its name
// (phpLibClassesIn), since a class cannot be compiled in the middle of another
// class's walk; the classes a piece spells follow it, and the class it extends
// precedes it. The library's constants come with the first piece. Each piece is
// compiled from the whole file with every other line blanked, so a position in it
// is still the file's own line and column.
//
// A piece's lines run from its "function " or "class " / "interface " line to the
// line before the next top-level one of those or "const ".

// The library's lines; each piece's [first, last] line under its key, "$name"
// for a function and "#name" in lower case for a class (class names are case
// insensitive); the class names in lower case; the key of the class each class
// extends; for each line whether it belongs to a piece; the names of the
// functions a program may name, lower-cased between commas; and their keys.
function phpLibIndex(text) {
    var lines = text.split("\n")
    var fns = {}
    var names = ","
    var named = []
    var classes = []
    var bases = {}
    var inFn = []
    var cur = ""
    var start = 0
    for (var i = 0; i < lines.length; i++) {
        var l = lines[i]
        var head = phpLibHead(l)
        if (cur != "" && (head != "" || l.substring(0, 6) == "const ")) {
            fns[cur] = [start, i - 1]
            cur = ""
        }
        if (head != "") {
            var j = head.length
            while (j < l.length && l.charAt(j) != "(" && l.charAt(j) != " ") { j = j + 1 }
            var name = l.substring(head.length, j)
            if (head == "function ") {
                cur = "$" + name
                if (name.substring(0, 5) != "__mec") {
                    names = names + name.toLowerCase() + ","
                    named.push(cur)
                }
            }
            else {
                cur = "#" + name.toLowerCase()
                classes.push(name.toLowerCase())
                bases[cur] = phpLibExtends(l)
            }
            start = i
        }
        inFn.push(cur != "")
    }
    if (cur != "") { fns[cur] = [start, lines.length - 1] }
    return {lines: lines, fns: fns, classes: classes, bases: bases, inFn: inFn, names: names, named: named}
}

// The keyword that opens a piece on line l, or "".
function phpLibHead(l) {
    if (l.substring(0, 9) == "function ") { return "function " }
    if (l.substring(0, 6) == "class ") { return "class " }
    if (l.substring(0, 10) == "interface ") { return "interface " }
    return ""
}

// The key of the class a class line extends ("class A extends B {"), or "".
function phpLibExtends(l) {
    var at = l.indexOf(" extends ")
    if (at < 0) { return "" }
    var j = at + 9
    while (j < l.length && l.charAt(j) != " " && l.charAt(j) != "{" && l.charAt(j) != ",") { j = j + 1 }
    return "#" + l.substring(at + 9, j).toLowerCase()
}

// The key of library function `name`, or "": a function name matches in any
// letter case, with one leading "\" of a fully qualified name dropped.
function phpLibKey(ix, name) {
    var k = "$" + phsCase(name.charCodeAt(0) == 92 ? name.substring(1) : name, false)
    return ix.fns[k] !== undefined ? k : ""
}

// Is lc, a lower-cased name, one of the library's functions a program may name,
// compiled or not? Twin of function_exists in php-to-llvm-ir.abnf.
function phpLibNamed(ix, lc) { return lc.indexOf(",") < 0 && ix.names.indexOf("," + lc + ",") >= 0 }

// The position of the callable that lc, a lower-cased function name or
// "closure::fromcallable", takes by name: -n for each of the last n arguments,
// libAt for a library function that takes a callback, 1000 for none.
function phpLibSinkAt(lc, libAt) {
    if (lc == "call_user_func" || lc == "call_user_func_array" || lc == "is_callable" || lc == "function_exists"
        || lc == "closure::fromcallable") { return 0 }
    if (lc == "iterator_apply") { return 1 }
    return libAt
}

// Can a call of lc reach a function through a name built at run time? Its
// callable is the argument at phpLibSinkAt's position or the one named after that
// parameter; dyn on an argument record is phpLibDynArg's answer for it, and a
// record without it, or a spread (field spr; lab is the label field), may be
// anything.
function phpLibDynCall(lc, libAt, args, lab, spr) {
    var at = phpLibSinkAt(lc, libAt)
    if (at == 1000) { return false }
    var pname = "callback"
    if (lc == "is_callable") { pname = "value" }
    if (lc == "function_exists") { pname = "function" }
    for (var i = 0; i < args.length; i++) {
        var a = args[i]
        if (a != null) {
            if (a[spr] !== undefined) { return true }
            var hit = false
            if (a[lab] !== undefined) { hit = a[lab] == pname }
            else if (at >= 0) { hit = i == at }
            else { hit = i >= args.length + at }
            if (hit && a.dyn !== false) { return true }
        }
    }
    return false
}

// Can argument text t evaluate to a function name built at run time? A string
// literal names its function where it is written (phpLibRef), and a closure, an
// array literal (a method callable) or a first-class callable 'f(...)' is no
// name at all; anything else may be one.
function phpLibDynArg(t) {
    var a = 0
    var z = t.length
    while (a < z && phpLibSpace(t.charCodeAt(a))) { a = a + 1 }
    while (z > a && phpLibSpace(t.charCodeAt(z - 1))) { z = z - 1 }
    if (a == z) { return false }
    var w = t.substring(a, z)
    var q = w.charAt(0)
    if (q == "'" || q == "\"" || q == "[") { return phpLibCloseAt(w, 0) != w.length - 1 }
    var lead = phpLibLead(w)
    if (lead.substring(0, 6) == "static") { lead = lead.substring(6) }
    if (lead.substring(0, 9) == "function(" || lead.substring(0, 10) == "function&(" || lead.substring(0, 3) == "fn("
        || lead.substring(0, 4) == "fn&(") { return false }
    if (lead.substring(0, 6) == "array(") { return phpLibCloseAt(w, w.indexOf("(")) != w.length - 1 }
    if (w.length > 5 && w.substring(w.length - 5) == "(...)") {
        for (var i = 0; i < w.length - 5; i++) {
            var ch = w.charCodeAt(i)
            if (!phpLibWordChar(w, i) && ch != 36 && ch != 92 && ch != 58 && ch != 45 && ch != 62) { return true }
        }
        return false
    }
    return true
}

// The first 12 non-blank characters of w, lower-cased.
function phpLibLead(w) {
    var out = ""
    for (var i = 0; i < w.length && out.length < 12; i++) {
        if (!phpLibSpace(w.charCodeAt(i))) { out = out + w.charAt(i).toLowerCase() }
    }
    return out
}

function phpLibSpace(ch) { return ch == 32 || ch == 9 || ch == 10 || ch == 13 }

// The index in w of what closes the quote or bracket at i: the matching quote,
// past escapes, or the bracket that balances it, past quoted text; -1 if none.
// A '"' literal holding a '$' or '{' may interpolate, so it closes nowhere.
function phpLibCloseAt(w, i) {
    var open = w.charAt(i)
    if (open == "'" || open == "\"") {
        var j = i + 1
        while (j < w.length) {
            var d = w.charAt(j)
            if (d == "\\") { j = j + 2 }
            else if (d == open) { return j }
            else if (open == "\"" && (d == "$" || d == "{")) { return -1 }
            else { j = j + 1 }
        }
        return -1
    }
    var close = open == "[" ? "]" : ")"
    var depth = 0
    var k = i
    while (k < w.length) {
        var e = w.charAt(k)
        if (e == "'" || e == "\"") {
            var end = phpLibCloseAt(w, k)
            if (end < 0) { return -1 }
            k = end + 1
        } else {
            if (e == open) { depth = depth + 1 }
            else if (e == close) {
                depth = depth - 1
                if (depth == 0) { return k }
            }
            k = k + 1
        }
    }
    return -1
}

// The keys of the library classes text spells as a whole word, in any case.
function phpLibClassesIn(ix, text) {
    var out = []
    var low = text.toLowerCase()
    for (var k = 0; k < ix.classes.length; k++) {
        var nm = ix.classes[k]
        var rest = low
        var at = rest.indexOf(nm)
        while (at >= 0) {
            if (!phpLibWordChar(rest, at - 1) && !phpLibWordChar(rest, at + nm.length)) {
                out.push("#" + nm)
                at = -1
            } else {
                rest = rest.substring(at + 1)
                at = rest.indexOf(nm)
            }
        }
    }
    return out
}

// Is the character at i of s part of a name?
function phpLibWordChar(s, i) {
    if (i < 0 || i >= s.length) { return false }
    var ch = s.charCodeAt(i)
    return (ch >= 97 && ch <= 122) || (ch >= 48 && ch <= 57) || ch == 95 || (ch >= 65 && ch <= 90) || ch >= 128
}

// The text that compiles library piece `key`, and the constants too when `base`.
// Line 0, the open tag, is always kept.
function phpLibText(ix, key, base) {
    var span = ix.fns[key]
    if (!base) {
        // Without the base only the piece's own lines are kept, so the blanked
        // lines before it are newlines and the ones after it can go.
        var nl = "\n"
        while (nl.length < span[0]) { nl = nl + nl }
        return ix.lines[0] + nl.substring(0, span[0]) + ix.lines.slice(span[0], span[1] + 1).join("\n")
    }
    var out = [ix.lines[0]]
    for (var i = 1; i < ix.lines.length; i++) {
        if ((i >= span[0] && i <= span[1]) || (base && !ix.inFn[i])) { out.push(ix.lines[i]) }
        else { out.push("") }
    }
    return out.join("\n")
}

// The argument counts library function `key` accepts, from its declaration line:
// [min, max], max -1 for a variadic one. min counts up to the last parameter
// without a default (zend_function.required_num_args), max every parameter, as
// an internal function's ZEND_PARSE_PARAMETERS_START(min, max) states them.
function phpLibArity(ix, key) {
    var l = ix.lines[ix.fns[key][0]]
    var open = l.indexOf("(")
    var close = phpLibCloseAt(l, open)
    var min = 0
    var n = 0
    var opt = false
    var rest = false
    var k = open + 1
    while (k <= close) {
        var ch = l.charAt(k)
        if (ch == "," || k == close) {
            if (k > open + 1) {
                n = n + 1
                if (!opt && !rest) { min = n }
            }
            opt = false
        } else if (ch == "=") { opt = true }
        else if (l.substring(k, k + 3) == "...") { rest = true }
        if (ch == "'" || ch == "\"" || ch == "[" || ch == "(") { k = phpLibCloseAt(l, k) + 1 }
        else { k = k + 1 }
    }
    return [min, rest ? -1 : n]
}

// Is n arguments a count library function `name` refuses? A direct call of an
// engine native that is goes to the library's declaration instead, whose arity
// check raises the error.
function phpLibArityBad(ix, name, n) {
    var key = phpLibKey(ix, name)
    if (key == "") { return false }
    var ar = phpLibArity(ix, key)
    return n < ar[0] || (ar[1] >= 0 && n > ar[1])
}
