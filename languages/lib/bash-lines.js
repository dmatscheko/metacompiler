// bash-lines.js -- $LINENO inside a $( ) command substitution. Included by
// bash-interpreter.abnf and bash-to-llvm-ir.abnf.
//
// bash runs a $( ) from its DEPARSED text, not from the source: the first command
// is on the line of the enclosing command ($LINENO when the substitution is
// expanded), and every later command is one line further per line break the
// printer keeps. The printer drops leading lines, blank lines and comments, joins a
// line ending in && || | ( { to the next one, and a line to a following one that
// starts with ")". bash 5.3: `z=$(<nl>echo $LINENO<nl>)` on lines 2-4 prints 4.
// A function definition prints as `function NAME () `, `{ ` and its body on lines
// of their own, the body's line breaks doubled (an empty line after each command)
// and its closing `}` on the line after the last one: in `p() {<nl>a<nl>b<nl>}` a
// is 2 lines below p and b 2 below a. Commands a ; separates in a body print on lines
// of their own: in `p() { a; b; }` b is 1 below a.
// The printer also re-flows other compound commands (`if a; then b; fi` prints as
// three lines), which is not modelled. A `...` substitution runs from its raw
// text, so there every newline counts.

// The line table of a $( ) whose text t (from just after "$(" up to the ")", at
// source offset off) starts on line l0: {first, skips, addL, addN, own, seps, closes}, first =
// the line of its first command, skips = the lines from first on whose line break does
// not count, addN[i] = how many more lines the break after line addL[i] counts, own =
// the lines whose commands stand 2 lines further down (a body begun on its
// definition's line), seps = the offsets of a body's ; separators (bashCsSeps), each a
// line further for every command after it, closes = {ln, at} the `}` of such a body
// closed on its line, which the commands after it on that line follow on a line of its
// own.
function bashCsLines(t, l0, off) {
    var first = -1
    var skips = []
    var addL = []
    var addN = []
    var own = []
    var seps = []
    var closes = []
    var pending = -1      // the last counting line, until the next line shows it is joined
    var body = 0          // the brace depth inside a function definition's body
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
            var s = t.substring(a, z)
            var l1 = t.charAt(z - 1)
            var l2 = z - 2 >= a ? t.substring(z - 2, z) : ""
            var hb = (body == 0) ? bashFnHeadBody(s) : -1
            if (hb >= 0 && hb < s.length) {
                var cl = bashCsSeps(t, a + hb, z, off, seps)
                if (cl >= 0) { closes.push({ln: ln, at: off + cl}) }
            }
            else if (body > 0) { bashCsSeps(t, a, z, off, seps) }
            if (hb >= 0) {
                // A definition's header: its body starts 2 printed lines below it. Closed on
                // the same line, its `}` is the line after the body.
                if (hb < s.length) { own.push(ln) }
                addL.push(ln)
                if (bashFnClosed(s, hb)) { addN.push(3) }
                else { body = 1; addN.push(hb < s.length ? 3 : 1) }
            } else if (body > 0) {
                if (s == "}" && body == 1) {
                    // The closing brace follows the last command directly.
                    if (addL.length > 0) { addN[addN.length - 1] = addN[addN.length - 1] - 1 }
                    body = 0
                    pending = ln
                } else {
                    // `cmd; }` ends the body too, its `}` a printed line of its own.
                    if (s.charAt(0) == "}") { body = body - 1 }
                    if (l1 == "{") { body = body + 1 }
                    else if (l1 == "}" && s.charAt(0) != "}" && s.charAt(0) != "{") { body = body - 1 }
                    if (l1 == "|" || l1 == "(" || l2 == "&&") { skips.push(ln) }
                    else { addL.push(ln); addN.push(1); pending = ln }
                }
            } else if (l1 == "|" || l1 == "(" || l1 == "{" || l2 == "&&") { skips.push(ln) } else { pending = ln }
        } else if (first >= 0) { skips.push(ln) }
        ln = ln + 1
        i = e + 1
    }
    if (first < 0) { first = l0 }
    return {first: first, skips: skips, addL: addL, addN: addN, own: own, seps: seps, closes: closes}
}

// The source offsets (t[0] at off) of the ; separators in the body text t[from, to) that
// a command follows on the line, a subshell's included: each is a printed line break.
// One inside $( ), ${ } or [ ] is not, and a `}` the text does not open ends the body:
// answers its offset in t, else -1.
function bashCsSeps(t, from, to, off, seps) {
    var q = ""
    var st = []
    var k = from
    while (k < to) {
        var ch = t.charAt(k)
        var top = (st.length > 0) ? st[st.length - 1] : ""
        if (q != "") { if (ch == q) { q = "" } else if (ch == "\\" && q == "\"") { k = k + 1 } }
        else if (ch == "'" || ch == "\"") { q = ch }
        else if (ch == "\\") { k = k + 1 }
        else if (ch == "$" && (t.charAt(k + 1) == "(" || t.charAt(k + 1) == "{")) { st.push("o"); k = k + 1 }
        else if (ch == "(" || ch == "{") { st.push((top == "o") ? "o" : "p") }
        else if (ch == "[") { st.push("o") }
        else if (ch == ")" || ch == "]" || ch == "}") {
            if (st.length == 0) { return k }
            st.pop()
        }
        else if (ch == ";" && top != "o" && t.charAt(k - 1) != ";") {
            var j = k + 1
            while (j < to && (t.charAt(j) == " " || t.charAt(j) == "\t")) { j = j + 1 }
            if (j < to && ";)}".indexOf(t.charAt(j)) < 0) { seps.push(off + k) }
        }
        k = k + 1
    }
    return -1
}

// Whether the line s opens a function definition's body - `NAME() {`, `function NAME {`
// or `function NAME() {`, possibly with body text after the brace - answering the index
// of the text after that "{" (s.length when nothing follows it), else -1.
function bashFnHeadBody(s) {
    var p = s.indexOf("{")
    if (p < 1) { return -1 }
    var head = s.substring(0, p)
    var n = head.length
    while (n > 0 && head.charAt(n - 1) == " ") { n = n - 1 }
    head = head.substring(0, n)
    var kw = head.length > 9 && head.substring(0, 9) == "function "
    if (!kw && (n < 3 || head.substring(n - 2, n) != "()")) { return -1 }
    var j
    for (j = 0; j < n; j++) {
        var c = head.charAt(j)
        if (c == "'" || c == "\"" || c == "$" || c == "=" || c == ";" || c == "&" || c == "|") { return -1 }
    }
    var r = p + 1
    while (r < s.length && s.charAt(r) == " ") { r = r + 1 }
    return r
}
// Whether a body begun at index b of the header line s closes on that line (`; }`).
function bashFnClosed(s, b) {
    var c = s.lastIndexOf("}")
    if (c <= b) { return false }
    var d = c - 1
    while (d > b && s.charAt(d) == " ") { d = d - 1 }
    return s.charAt(d) == ";" || s.charAt(d) == "&"
}

// The line of a command on source line n, its first word ending at source offset pos,
// of the substitution with table tab, when the enclosing command is on line base.
function bashCsLine(tab, base, n, pos) {
    if (n <= tab.first && tab.own.indexOf(n) < 0) { return base }
    var k = n - tab.first
    var j
    for (j = 0; j < tab.skips.length; j++) { if (tab.skips[j] < n) { k = k - 1 } }
    for (j = 0; j < tab.addL.length; j++) { if (tab.addL[j] < n) { k = k + tab.addN[j] } }
    if (tab.own.indexOf(n) >= 0) { k = k + 2 }
    for (j = 0; j < tab.seps.length; j++) { if (tab.seps[j] < pos) { k = k + 1 } }
    for (j = 0; j < tab.closes.length; j++) { if (tab.closes[j].ln == n && tab.closes[j].at < pos) { k = k + 1 } }
    return base + k
}
