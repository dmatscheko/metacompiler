// bash-lines.js -- $LINENO inside a $( ) command substitution. Included by
// bash-interpreter.abnf and bash-to-llvm-ir.abnf.
//
// bash runs a $( ) from its DEPARSED text, not from the source: the first command
// is on the line of the enclosing command ($LINENO when the substitution is
// expanded), and every later command is one line further per line break the
// printer keeps. The printer drops leading lines, blank lines and comments, joins a
// line ending in && || | ( { to the next one, and a line to a following one that
// starts with ")". bash 5.3: `z=$(<nl>echo $LINENO<nl>)` on lines 2-4 prints 4.
// It also re-flows compound commands (`if a; then b; fi` prints as three lines),
// which is not modelled. A `...` substitution runs from its raw text, so there
// every newline counts.

// The line table of a $( ) whose text t (from just after "$(" up to the ")")
// starts on line l0: {first, skips}, first = the line of its first command,
// skips = the lines from first on whose line break does not count.
function bashCsLines(t, l0) {
    var first = -1
    var skips = []
    var pending = -1      // the last counting line, until the next line shows it is joined
    var ln = l0
    var i = 0
    while (i <= t.length) {
        var e = t.indexOf("\n", i)
        if (e < 0) { e = t.length }
        // The line without its blanks and its trailing # comment (outside quotes).
        var a = i
        while (a < e && (t.charAt(a) == " " || t.charAt(a) == "\t" || t.charAt(a) == "\r")) { a = a + 1 }
        var z = a
        var q = ""
        var k = a
        while (k < e) {
            var ch = t.charAt(k)
            if (q != "") { if (ch == q) { q = "" } else if (ch == "\\" && q == "\"") { k = k + 1 } }
            else if (ch == "'" || ch == "\"") { q = ch }
            else if (ch == "\\") { k = k + 1 }
            else if (ch == "#" && (k == a || t.charAt(k - 1) == " " || t.charAt(k - 1) == "\t")) { break }
            if (ch != " " && ch != "\t" && ch != "\r") { z = k + 1 }
            k = k + 1
        }
        if (z > a) {
            if (first < 0) { first = ln }
            if (pending >= 0 && t.charAt(a) == ")") { skips.push(pending) }
            pending = -1
            var l1 = t.charAt(z - 1)
            var l2 = z - 2 >= a ? t.substring(z - 2, z) : ""
            if (l1 == "|" || l1 == "(" || l1 == "{" || l2 == "&&") { skips.push(ln) } else { pending = ln }
        } else if (first >= 0) { skips.push(ln) }
        ln = ln + 1
        i = e + 1
    }
    if (first < 0) { first = l0 }
    return {first: first, skips: skips}
}

// The line of source line n of the substitution with table tab, when the enclosing
// command is on line base.
function bashCsLine(tab, base, n) {
    if (n <= tab.first) { return base }
    var k = n - tab.first
    var j
    for (j = 0; j < tab.skips.length; j++) { if (tab.skips[j] < n) { k = k - 1 } }
    return base + k
}
