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
// functions a program may name, lower-cased between commas; their keys; and the
// global constants' names between commas. The key "%base" is a piece of no
// lines: loaded before any other, it compiles the constants alone.
function phpLibIndex(text) {
    var lines = text.split("\n")
    var fns = {"%base": [1, 0]}
    var names = ","
    var consts = ","
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
        if (l.substring(0, 6) == "const ") { consts = consts + l.substring(6, l.indexOf(" ", 6)) + "," }
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
    return {lines: lines, fns: fns, classes: classes, bases: bases, inFn: inFn, names: names, named: named,
            consts: consts}
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

// The key that compiles the library's global constant `name`, or "".
function phpLibConstKey(ix, name) {
    return name.indexOf(",") < 0 && ix.consts.indexOf("," + name + ",") >= 0 ? "%base" : ""
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

// The text t, trimmed, when it spells a storage location - a variable ('$name',
// or a variable variable '$$name' / '${expr}') followed by any run of '[...]' and
// '->name' steps, brackets balanced and quoted text skipped - and null otherwise.
// A bare $GLOBALS is a read-only copy, no variable. The grammars re-parse such a
// text as a Target, which a text failing to parse as one would abort, so nothing
// else passes.
function phpLibVarText(t) {
    var a = 0
    var z = t.length
    while (a < z && phpLibSpace(t.charCodeAt(a))) { a = a + 1 }
    while (z > a && phpLibSpace(t.charCodeAt(z - 1))) { z = z - 1 }
    var w = t.substring(a, z)
    if (w.length < 2 || w.charAt(0) != "$" || w == "$GLOBALS") { return null }
    var i = 1
    while (i < w.length && w.charAt(i) == "$") { i = i + 1 }
    if (i < w.length && w.charAt(i) == "{") {
        i = phpLibSkipBracket(w, i, "{", "}")
        if (i < 0) { return null }
    } else {
        if (i >= w.length || !phpLibNameStart(w, i)) { return null }
        while (i < w.length && phpLibNameChar(w, i)) { i = i + 1 }
    }
    while (i < w.length) {
        if (w.charAt(i) == "[") {
            i = phpLibSkipBracket(w, i, "[", "]")
            if (i < 0) { return null }
        } else if (w.charAt(i) == "-" && i + 2 < w.length && w.charAt(i + 1) == ">" && phpLibNameStart(w, i + 2)) {
            i = i + 2
            while (i < w.length && phpLibNameChar(w, i)) { i = i + 1 }
        } else { return null }
    }
    return w
}
// The index just past the bracket that closes the one opening at i of w, quoted
// text skipped; -1 when it is not closed.
function phpLibSkipBracket(w, i, open, close) {
    var depth = 0
    while (i < w.length) {
        var ch = w.charAt(i)
        if (ch == "'" || ch == "\"") {
            var q = ch
            i = i + 1
            while (i < w.length && w.charAt(i) != q) {
                if (w.charAt(i) == "\\") { i = i + 1 }
                i = i + 1
            }
            if (i >= w.length) { return -1 }
        } else if (ch == open) { depth = depth + 1 }
        else if (ch == close) {
            depth = depth - 1
            if (depth == 0) { return i + 1 }
        }
        i = i + 1
    }
    return -1
}
// An ASCII letter or '_' / one or a digit, at i of s.
function phpLibNameStart(s, i) {
    var ch = s.charCodeAt(i)
    return (ch >= 97 && ch <= 122) || (ch >= 65 && ch <= 90) || ch == 95
}
function phpLibNameChar(s, i) {
    var ch = s.charCodeAt(i)
    return phpLibNameStart(s, i) || (ch >= 48 && ch <= 57)
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
// an internal function's ZEND_PARSE_PARAMETERS_START(min, max) states them. A
// variadic standing for parameters whose presence the body tests (php-src's
// "= UNKNOWN" defaults) caps max with a "/* at most N */" in the list.
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
    var cap = l.indexOf("/* at most ")
    if (rest && cap > open && cap < close) { return [min, parseInt(l.substring(cap + 11, l.indexOf(" */", cap)))] }
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

// Does the library's declaration of engine native `name` declare scalar
// parameter types, which zpp reads for the call's site (__mec_zpp)? A program's
// direct call of one from a strict_types file, or with display_errors=1, goes to
// that declaration, so that the file decides the coercions and a null argument's
// deprecation is displayed; the native itself reads any argument coercively.
function phpLibZppNative(name) {
    return ",strlen,intdiv,substr,strpos,strrpos,strcmp,ord,chr,strtolower,strtoupper,".indexOf("," + name + ",") >= 0
}

// For a declared type typ with a callable alternative, the type its other
// alternatives make (rest, "" for none), and whether one of them takes a string
// (str) or an array (arr) as it is; null for any other type. zend_check_type
// takes a string or an array as it is when its own type is in the mask, else
// when zend_is_callable accepts it, else as the scalar alternatives coerce it.
function phpCallableRest(typ) {
    var alts = phsTypeAlts(typ)
    var cb = false
    var rest = ""
    var str = false
    var arr = false
    for (var i = 0; i < alts.length; i++) {
        var lc = phsCase(alts[i], false)
        if (lc == "callable") { cb = true }
        else {
            rest = rest == "" ? alts[i] : rest + "|" + alts[i]
            if (lc == "string" || lc == "mixed") { str = true }
            if (lc == "array" || lc == "iterable" || lc == "mixed") { arr = true }
        }
    }
    return cb ? {rest: rest, str: str, arr: arr} : null
}
