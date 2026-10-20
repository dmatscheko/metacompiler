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
// extends; and for each line whether it belongs to a piece.
function phpLibIndex(text) {
    var lines = text.split("\n")
    var fns = {}
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
            if (head == "function ") { cur = "$" + name }
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
    return {lines: lines, fns: fns, classes: classes, bases: bases, inFn: inFn}
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
