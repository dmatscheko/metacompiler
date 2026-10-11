// php-stdlib-index.js -- which lines of lib/php-stdlib.php a program compiles.
//
// Both php grammars include this and compile the library ONE FUNCTION AT A TIME
// (phpLibRef): the first mention of a library function compiles that function,
// and a library function its body mentions is compiled when that mention is
// walked, the same way. The library's constants and classes come with the first
// one. Each piece is compiled from the whole file with every other line blanked,
// so a position in it is still the file's own line and column.
//
// A function's lines run from its "function " line to the line before the next
// top-level "function ", "const " or "class " line.

// The library's lines, each function's [first, last] line under "$name", and for
// each line whether it belongs to a function.
function phpLibIndex(text) {
    var lines = text.split("\n")
    var fns = {}
    var inFn = []
    var cur = ""
    var start = 0
    for (var i = 0; i < lines.length; i++) {
        var l = lines[i]
        var isFn = l.substring(0, 9) == "function "
        if (cur != "" && (isFn || l.substring(0, 6) == "const " || l.substring(0, 6) == "class ")) {
            fns["$" + cur] = [start, i - 1]
            cur = ""
        }
        if (isFn) {
            var j = 9
            while (j < l.length && l.charAt(j) != "(") { j = j + 1 }
            cur = l.substring(9, j)
            start = i
        }
        inFn.push(cur != "")
    }
    if (cur != "") { fns["$" + cur] = [start, lines.length - 1] }
    return {lines: lines, fns: fns, inFn: inFn}
}

// Is `name` one of the library's functions?
function phpLibHas(ix, name) { return ix.fns["$" + name] !== undefined }

// The text that compiles library function `name`, and the constants and classes
// too when `base`. Line 0, the open tag, is always kept.
function phpLibText(ix, name, base) {
    var span = ix.fns["$" + name]
    if (!base) {
        // Without the base only the function's own lines are kept, so the blanked
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
