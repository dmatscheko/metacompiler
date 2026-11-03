// bash-synerr.js -- the syntax error bash reports for a text the grammar cannot parse.
// Included by bash-interpreter.abnf and bash-to-llvm-ir.abnf.
//
// The grammar takes what it cannot parse as one trailing piece (SynErr); this re-reads the
// command line that piece starts on the way bash's parser does and names the first thing
// it cannot take, in bash 5.3's words:
//   syntax error near unexpected token `TOK'     (then the line it stands on, quoted)
//   syntax error: unexpected end of file [from `KW' command on line N]
//   unexpected EOF while looking for matching `C'
// A token is a word, an operator or `newline'; a reserved word counts only in command
// position. The end of file names the innermost compound command still open. A quote, a
// ${, a backquote, (( and NAME[ in command position report the line they open on. A $( )
// is parsed as a list and reports its end-of-file line; a token it cannot take is reported
// "while looking for matching `)'", which ends an eval's (sub)shell. A text not ending in
// a newline reads as if it did. A here-document's body is skipped (a body the pre-pass
// moved, `<<A\x02body\x03`, too). A [[ ]] has its own messages (bseCondTerm). Not
// modelled: the parts of an arithmetic for, read to their "))".

// The report for text t whose parse stopped at offset p, read from offset from (the start
// of p's command line). lineAt(off) is the line of an offset. Answers {ls, sub, code}: the
// message lines, each "line N: ...", whether the error stands inside a $( ), and the status
// it fails with (2; 1 for a NAME=( array never closed).
function bashSynErr(t, from, p, lineAt) {
    var st = {s: t, i: from, k: "", v: "", pv: "", plain: false, nlEnd: false, code: 2, at: 0, err: "", eAt: 0, eTok: "", eMsg: "", open: [], openAt: [], hd: [], hdDash: []}
    bseNext(st, true)
    bseList(st, [])
    if (st.err == "" && st.k != "e") { bseTokErr(st) }
    if (st.err == "") {
        // The checker took the rest: name what the parse stopped at.
        st.i = p
        bseNext(st, false)
        bseTokErr(st)
    }
    var eofLn = lineAt(t.length) + ((t.length > 0 && t.charAt(t.length - 1) != "\n") ? 1 : 0)
    if (st.err == "tok") {
        var ln = lineAt(st.eAt)
        var sub = st.open.indexOf("$(") >= 0
        return {ls: ["line " + ln + ": syntax error near unexpected token `" + st.eTok + "'" + (sub ? " while looking for matching `)'" : ""),
                     "line " + ln + ": `" + bseLineText(t, st.eAt, ln, lineAt) + "'"], sub: sub, code: 2}
    }
    if (st.err == "cond") {
        var cl = "line " + lineAt(st.eAt) + ": "
        var out = []
        var ms = (st.eMsg != "") ? st.eMsg.split("\n") : []
        var m
        for (m = 0; m < ms.length; m++) { out.push(cl + ms[m]) }
        out.push(cl + "syntax error near `" + st.eTok + "'")
        out.push(cl + "`" + bseLineText(t, st.eAt, lineAt(st.eAt), lineAt) + "'")
        return {ls: out, sub: false, code: 2}
    }
    if (st.err == "eof") {
        var n = st.open.length
        var msg = "syntax error: unexpected end of file"
        if (n > 0 && st.open[n - 1] == "$(") { msg = "unexpected EOF while looking for matching `)'" }
        else if (n > 0) { msg = msg + " from `" + st.open[n - 1] + "' command on line " + lineAt(st.openAt[n - 1]) }
        return {ls: ["line " + eofLn + ": " + msg], sub: false, code: 2}
    }
    return {ls: ["line " + lineAt(st.eAt) + ": unexpected EOF while looking for matching `" + st.eTok + "'"], sub: false, code: st.code}
}

// The line of t holding offset at, as written: a continuation joined into it and a
// here-document body moved into it (both from other lines) are left out.
function bseLineText(t, at, ln, lineAt) {
    var s = at
    while (s > 0 && t.charAt(s - 1) != "\n") { s = s - 1 }
    var e = at
    while (e < t.length && t.charAt(e) != "\n") { e = e + 1 }
    var out = ""
    var k
    for (k = s; k < e; k++) {
        var ch = t.charAt(k)
        if (ch == "\x02") { while (k < e && t.charAt(k) != "\x03") { k = k + 1 } }
        else if (lineAt(k) == ln) { out = out + ch }
    }
    return out
}

function bseTokErr(st) {
    if (st.err != "") { return }
    if (st.k == "e") { st.err = "eof"; return }
    st.err = "tok"
    st.eAt = st.at
    st.eTok = (st.k == "n") ? "newline" : st.v
}
// An error of the [[ ]] parser: its message (or none), then "syntax error near" the token,
// or the one before it when that is a newline or the end.
function bseCondErr(st, msg) {
    if (st.err != "") { return }
    st.err = "cond"
    st.eMsg = msg
    st.eAt = st.at
    st.eTok = (st.k == "n" || st.k == "e") ? st.pv : st.v
}
// The token a [[ ]] message names; the end of the text is read as a newline.
function bseCondTok(st) { return (st.k == "n" || st.k == "e") ? "newline" : st.v }
function bseMatchErr(st, ch, at) {
    if (st.err != "") { return }
    st.err = "match"
    st.eTok = ch
    st.eAt = at
}

// ---- tokens ----

function bseBlank(ch) { return ch == " " || ch == "\t" || ch == "\r" }
function bseMeta(ch) { return ch == " " || ch == "\t" || ch == "\r" || ch == "\n" || ch == ";" || ch == "&" || ch == "|" || ch == "<" || ch == ">" || ch == "(" || ch == ")" || ch == "\x02" }
function bseNameCh(ch, first) {
    var cc = ch.charCodeAt(0)
    return (cc >= 97 && cc <= 122) || (cc >= 65 && cc <= 90) || cc == 95 || (!first && cc >= 48 && cc <= 57)
}
var bseOps = [";;&", "<<<", "<<-", "&>>", ";;", ";&", "&&", "||", "|&", "<<", ">>", "<&", ">&", "<>", ">|", "&>", ";", "&", "|", "<", ">", "(", ")"]

// The next token into st.k (w word, o operator, n newline, e end), st.v, st.at; a word is
// plain when nothing in it is quoted or expanded. cmd: a word here is in command position
// (NAME[ opens a subscript, NAME=( an array).
function bseNext(st, cmd) {
    if (st.err != "") { st.k = "e"; return }
    var s = st.s
    if (st.k != "n") { st.pv = st.v }
    while (true) {
        while (st.i < s.length && bseBlank(s.charAt(st.i))) { st.i = st.i + 1 }
        if (s.charAt(st.i) == "\\" && s.charAt(st.i + 1) == "\n") { st.i = st.i + 2; continue }
        if (s.charAt(st.i) == "#") { while (st.i < s.length && s.charAt(st.i) != "\n") { st.i = st.i + 1 } }
        break
    }
    st.at = st.i
    // A text that does not end in a newline is read as if it did.
    if (st.i >= s.length && !st.nlEnd && s.length > 0 && s.charAt(s.length - 1) != "\n") {
        st.nlEnd = true
        st.k = "n"
        st.v = "\n"
        return
    }
    if (st.i >= s.length) { st.k = "e"; st.v = ""; return }
    var ch = s.charAt(st.i)
    if (ch == "\n") {
        st.k = "n"
        st.v = "\n"
        st.i = st.i + 1
        bseHereBodies(st)
        return
    }
    if (!((ch == "<" || ch == ">") && s.charAt(st.i + 1) == "(")) {
        var j
        for (j = 0; j < bseOps.length; j++) {
            if (s.slice(st.i, st.i + bseOps[j].length) == bseOps[j]) {
                st.k = "o"
                st.v = bseOps[j]
                st.i = st.i + bseOps[j].length
                return
            }
        }
    }
    bseWord(st, cmd)
}

// The bodies of the here-documents the line just ended opened.
function bseHereBodies(st) {
    var s = st.s
    var h
    for (h = 0; h < st.hd.length; h++) {
        while (st.i < s.length) {
            var e = st.i
            while (e < s.length && s.charAt(e) != "\n") { e = e + 1 }
            var l = s.slice(st.i, e)
            if (st.hdDash[h]) { while (l.charAt(0) == "\t") { l = l.slice(1) } }
            st.i = (e < s.length) ? e + 1 : e
            if (l == st.hd[h]) { break }
        }
    }
    st.hd = []
    st.hdDash = []
}

function bseWord(st, cmd) {
    var s = st.s
    var start = st.i
    var plain = true
    // NAME[ in command position: bash reads the subscript to its matching "]", past blanks
    // and lines.
    if (cmd && bseNameCh(s.charAt(st.i), true)) {
        var q = st.i
        while (q < s.length && bseNameCh(s.charAt(q), false)) { q = q + 1 }
        if (s.charAt(q) == "[") {
            var depth = 0
            var shut = false
            while (!shut && q < s.length) {
                var c0 = s.charAt(q)
                if (c0 == "\\") { q = q + 1 }
                else if (c0 == "'" || c0 == "\"") {
                    q = q + 1
                    while (q < s.length && s.charAt(q) != c0) { if (c0 == "\"" && s.charAt(q) == "\\") { q = q + 1 }; q = q + 1 }
                }
                else if (c0 == "[") { depth = depth + 1 }
                else if (c0 == "]") { depth = depth - 1; shut = depth == 0 }
                q = q + 1
            }
            if (!shut) { bseMatchErr(st, "]", start); st.k = "e"; return }
            st.i = q
        }
    }
    while (st.i < s.length && st.err == "") {
        var ch = s.charAt(st.i)
        if (bseMeta(ch)) {
            // NAME=( and NAME+=( in command position open an array, read to its ")".
            if (ch == "(" && cmd && bseIsAsgHead(s.slice(start, st.i))) {
                plain = false
                bseParens(st, st.i + 1, 1, start)
                if (st.err != "") { st.code = 1 }
                continue
            }
            break
        }
        if (ch == "\\") { plain = false; st.i = st.i + 2; continue }
        if (ch == "'") { plain = false; bseSq(st, st.i + 1, false); continue }
        if (ch == "\"") { plain = false; bseDq(st, st.i + 1); continue }
        if (ch == "`") { plain = false; bseBq(st); continue }
        if (ch == "$") {
            var n1 = s.charAt(st.i + 1)
            if (n1 == "'") { plain = false; bseSq(st, st.i + 2, true); continue }
            if (n1 == "\"") { plain = false; bseDq(st, st.i + 2); continue }
            if (n1 == "{") { plain = false; bseBrace(st); continue }
            if (n1 == "(") { plain = false; bseDollarParen(st); continue }
        }
        if ((ch == "<" || ch == ">") && s.charAt(st.i + 1) == "(") { plain = false; bseParens(st, st.i + 2, 1, st.i); continue }
        if ((ch == "?" || ch == "*" || ch == "+" || ch == "@" || ch == "!") && s.charAt(st.i + 1) == "(") { bseParens(st, st.i + 2, 1, st.i); continue }
        st.i = st.i + 1
    }
    if (st.err != "") { st.k = "e"; return }
    st.k = "w"
    st.v = s.slice(start, st.i)
    st.plain = plain
}

// NAME=, NAME+=, NAME[SUB]= and NAME[SUB]+=: the head of an assignment word.
function bseIsAsgHead(w) {
    var i = 0
    if (w.length == 0 || !bseNameCh(w.charAt(0), true)) { return false }
    while (i < w.length && bseNameCh(w.charAt(i), false)) { i = i + 1 }
    if (w.charAt(i) == "[") {
        var e = w.indexOf("]", i)
        if (e < 0) { return false }
        i = e + 1
    }
    if (w.charAt(i) == "+") { i = i + 1 }
    return i == w.length - 1 && w.charAt(i) == "="
}
// A word whose first part is an assignment head.
function bseIsAsg(w) {
    var e = w.indexOf("=")
    return e > 0 && bseIsAsgHead(w.slice(0, e + 1))
}

function bseSq(st, k, esc) {
    var s = st.s
    var open = st.i
    while (k < s.length && s.charAt(k) != "'") { if (esc && s.charAt(k) == "\\") { k = k + 1 }; k = k + 1 }
    if (k >= s.length) { bseMatchErr(st, "'", open); return }
    st.i = k + 1
}
function bseDq(st, k) {
    var s = st.s
    var open = st.i
    st.i = k
    while (st.err == "") {
        if (st.i >= s.length) { bseMatchErr(st, "\"", open); return }
        var ch = s.charAt(st.i)
        if (ch == "\"") { st.i = st.i + 1; return }
        if (ch == "\\") { st.i = st.i + 2 }
        else if (ch == "`") { bseBq(st) }
        else if (ch == "$" && s.charAt(st.i + 1) == "{") { bseBrace(st) }
        else if (ch == "$" && s.charAt(st.i + 1) == "(") { bseDollarParen(st) }
        else { st.i = st.i + 1 }
    }
}
function bseBq(st) {
    var s = st.s
    var open = st.i
    var k = st.i + 1
    while (k < s.length && s.charAt(k) != "`") { if (s.charAt(k) == "\\") { k = k + 1 }; k = k + 1 }
    if (k >= s.length) { bseMatchErr(st, "`", open); return }
    st.i = k + 1
}
// ${ ... }: to its matching "}", over quotes and nested ${ }.
function bseBrace(st) {
    var s = st.s
    var open = st.i
    var depth = 1
    st.i = st.i + 2
    while (st.err == "") {
        if (st.i >= s.length) { bseMatchErr(st, "}", open); return }
        var ch = s.charAt(st.i)
        if (ch == "}") { depth = depth - 1; st.i = st.i + 1; if (depth == 0) { return } }
        else if (ch == "\\") { st.i = st.i + 2 }
        else if (ch == "'") { bseSq(st, st.i + 1, false) }
        else if (ch == "\"") { bseDq(st, st.i + 1) }
        else if (ch == "`") { bseBq(st) }
        else if (ch == "$" && s.charAt(st.i + 1) == "{") { depth = depth + 1; st.i = st.i + 2 }
        else if (ch == "$" && s.charAt(st.i + 1) == "(") { bseDollarParen(st) }
        else { st.i = st.i + 1 }
    }
}
// From offset k, depth parentheses open: on past the ")" that closes the last, over quotes.
// One never closed is "matching `)'" on the line of open.
function bseParens(st, k, depth, open) {
    var s = st.s
    st.i = k
    while (st.err == "") {
        if (st.i >= s.length) { bseMatchErr(st, ")", open); return }
        var ch = s.charAt(st.i)
        if (ch == ")") { depth = depth - 1; st.i = st.i + 1; if (depth == 0) { return } }
        else if (ch == "(") { depth = depth + 1; st.i = st.i + 1 }
        else if (ch == "\\") { st.i = st.i + 2 }
        else if (ch == "'") { bseSq(st, st.i + 1, false) }
        else if (ch == "\"") { bseDq(st, st.i + 1) }
        else if (ch == "`") { bseBq(st) }
        else if (ch == "$" && s.charAt(st.i + 1) == "{") { bseBrace(st) }
        else { st.i = st.i + 1 }
    }
}
// $(( )) is read to its matching parentheses; $( LIST ) is parsed as a list.
function bseDollarParen(st) {
    var s = st.s
    if (s.charAt(st.i + 2) == "(") { bseParens(st, st.i + 3, 2, st.i); return }
    var sv = {k: st.k, v: st.v, pv: st.pv, plain: st.plain, at: st.at, hd: st.hd, hdDash: st.hdDash}
    st.open.push("$(")
    st.openAt.push(st.i)
    st.i = st.i + 2
    st.hd = []
    st.hdDash = []
    bseNext(st, true)
    bseList(st, [")"])
    if (st.err != "") { return }
    if (st.k == "e") { bseTokErr(st); return }
    if (!(st.k == "o" && st.v == ")")) { bseTokErr(st); return }
    st.open.pop()
    st.openAt.pop()
    st.k = sv.k
    st.v = sv.v
    st.pv = sv.pv
    st.plain = sv.plain
    st.at = sv.at
    st.hd = sv.hd
    st.hdDash = sv.hdDash
}

// ---- the grammar ----

function bseIsWord(st, w) { return st.k == "w" && st.plain && st.v == w }
function bseIsOp(st, o) { return st.k == "o" && st.v == o }
function bseAtStop(st, stops) {
    var i
    for (i = 0; i < stops.length; i++) {
        if (bseIsWord(st, stops[i]) || bseIsOp(st, stops[i])) { return true }
    }
    return false
}
function bseSkipNl(st) { while (st.err == "" && st.k == "n") { bseNext(st, true) } }

// A list up to one of stops (or the end); answers how many commands it holds.
function bseList(st, stops) {
    var n = 0
    while (st.err == "") {
        bseSkipNl(st)
        if (st.err != "" || st.k == "e" || bseAtStop(st, stops)) { return n }
        bseAndOr(st)
        n = n + 1
        if (st.err != "") { return n }
        if (bseIsOp(st, ";") || bseIsOp(st, "&")) { bseNext(st, true) }
        else if (st.k == "n") { bseNext(st, true) }
        else if (st.k == "e" || bseAtStop(st, stops)) { return n }
        else { bseTokErr(st); return n }
    }
    return n
}
function bseAndOr(st) {
    bsePipeline(st)
    while (st.err == "" && (bseIsOp(st, "&&") || bseIsOp(st, "||"))) {
        bseNext(st, true)
        bseSkipNl(st)
        bsePipeline(st)
    }
}
function bsePipeline(st) {
    while (st.err == "" && (bseIsWord(st, "!") || bseIsWord(st, "time"))) { bseNext(st, true) }
    bseCommand(st)
    while (st.err == "" && (bseIsOp(st, "|") || bseIsOp(st, "|&"))) {
        bseNext(st, true)
        bseSkipNl(st)
        bseCommand(st)
    }
}
// The reserved word w, after a list of n commands (none is an error at w).
function bseExpect(st, w, n, cmd) {
    if (st.err != "") { return }
    if (n == 0 || !bseIsWord(st, w)) { bseTokErr(st); return }
    bseNext(st, cmd)
}
function bseOpen(st, kw) { st.open.push(kw); st.openAt.push(st.at); bseNext(st, kw != "for" && kw != "select" && kw != "case" && kw != "[[") }
function bseClose(st) { if (st.err == "") { st.open.pop(); st.openAt.pop() } }

function bseCommand(st) {
    if (st.err != "") { return }
    if (st.k == "w" && st.plain) {
        var w = st.v
        if (w == "if") { bseIf(st); bseRedirs(st); return }
        if (w == "while" || w == "until") {
            bseOpen(st, w)
            var n = bseList(st, ["do"])
            bseExpect(st, "do", n, true)
            n = bseList(st, ["done"])
            bseExpect(st, "done", n, false)
            bseClose(st)
            bseRedirs(st)
            return
        }
        if (w == "for" || w == "select") { bseFor(st, w); bseRedirs(st); return }
        if (w == "case") { bseCase(st); bseRedirs(st); return }
        if (w == "{") { bseGroup(st); bseRedirs(st); return }
        if (w == "[[") { bseCond(st); bseRedirs(st); return }
        if (w == "function") {
            bseNext(st, false)
            if (st.k != "w") { bseTokErr(st); return }
            bseNext(st, false)
            if (bseIsOp(st, "(")) {
                bseNext(st, false)
                if (!bseIsOp(st, ")")) { bseTokErr(st); return }
                bseNext(st, true)
            }
            bseSkipNl(st)
            bseFuncBody(st)
            return
        }
        if (w == "then" || w == "else" || w == "elif" || w == "fi" || w == "do" || w == "done" || w == "esac" || w == "}" || w == "in") { bseTokErr(st); return }
    }
    if (bseIsOp(st, "(")) {
        if (st.s.charAt(st.i) == "(") { bseArith(st); bseRedirs(st); return }
        bseOpen(st, "(")
        var m = bseList(st, [")"])
        if (st.err == "" && st.k == "e") { bseTokErr(st); return }
        if (st.err == "" && (m == 0 || !bseIsOp(st, ")"))) { bseTokErr(st); return }
        bseNext(st, false)
        bseClose(st)
        bseRedirs(st)
        return
    }
    if (st.k == "w" || bseIsRedir(st)) { bseSimple(st); return }
    bseTokErr(st)
}
// (( ... )): to the matching "))".
function bseArith(st) {
    bseParens(st, st.i + 1, 2, st.at)
    bseNext(st, false)
}
function bseIsRedir(st) {
    if (st.k != "o") { return false }
    var v = st.v
    return v == "<" || v == ">" || v == ">>" || v == "<<" || v == "<<-" || v == "<<<" || v == "<&" || v == ">&" || v == "<>" || v == ">|" || v == "&>" || v == "&>>"
}
function bseRedir(st) {
    var op = st.v
    bseNext(st, false)
    if (st.err != "") { return }
    if (st.k != "w") { bseTokErr(st); return }
    if (op == "<<" || op == "<<-") {
        if (st.s.charAt(st.i) == "\x02") {
            var e = st.s.indexOf("\x03", st.i)
            st.i = (e < 0) ? st.s.length : e + 1
        } else {
            var d = ""
            var k
            for (k = 0; k < st.v.length; k++) {
                var ch = st.v.charAt(k)
                if (ch == "\\") { k = k + 1; d = d + st.v.charAt(k) }
                else if (ch != "'" && ch != "\"") { d = d + ch }
            }
            st.hd.push(d)
            st.hdDash.push(op == "<<-")
        }
    }
    bseNext(st, false)
}
function bseRedirs(st) { while (st.err == "" && bseIsRedir(st)) { bseRedir(st) } }

// Words and redirections; a "(" after a lone word that is no assignment defines a function.
function bseSimple(st) {
    var words = 0
    var asg = false
    var cmd = true
    var decl = false
    while (st.err == "") {
        if (bseIsRedir(st)) { bseRedir(st); continue }
        if (st.k != "w") { break }
        if (words == 0) { asg = bseIsAsg(st.v) }
        if (cmd && !bseIsAsg(st.v)) {
            cmd = false
            decl = st.v == "declare" || st.v == "typeset" || st.v == "local" || st.v == "export" || st.v == "readonly"
        }
        words = words + 1
        bseNext(st, cmd || decl)
    }
    if (st.err != "" || !bseIsOp(st, "(")) { return }
    if (words != 1 || asg) { bseTokErr(st); return }
    bseNext(st, false)
    if (!bseIsOp(st, ")")) { bseTokErr(st); return }
    bseNext(st, true)
    bseSkipNl(st)
    bseFuncBody(st)
}
function bseFuncBody(st) {
    if (st.err != "") { return }
    var ok = bseIsOp(st, "(")
    if (st.k == "w" && st.plain) {
        var w = st.v
        ok = w == "{" || w == "if" || w == "while" || w == "until" || w == "for" || w == "select" || w == "case" || w == "[["
    }
    if (!ok) { bseTokErr(st); return }
    bseCommand(st)
}

function bseIf(st) {
    bseOpen(st, "if")
    var n = bseList(st, ["then"])
    bseExpect(st, "then", n, true)
    n = bseList(st, ["elif", "else", "fi"])
    while (st.err == "" && bseIsWord(st, "elif")) {
        bseExpect(st, "elif", n, true)
        n = bseList(st, ["then"])
        bseExpect(st, "then", n, true)
        n = bseList(st, ["elif", "else", "fi"])
    }
    if (st.err == "" && bseIsWord(st, "else")) {
        bseExpect(st, "else", n, true)
        n = bseList(st, ["fi"])
    }
    bseExpect(st, "fi", n, false)
    bseClose(st)
}
function bseGroup(st) {
    bseOpen(st, "{")
    var n = bseList(st, ["}"])
    bseExpect(st, "}", n, false)
    bseClose(st)
}
function bseFor(st, kw) {
    bseOpen(st, kw)
    if (kw == "for" && bseIsOp(st, "(") && st.s.charAt(st.i) == "(") { bseArith(st) }
    else {
        if (st.k != "w") { bseTokErr(st); return }
        bseNext(st, false)
        bseSkipNl(st)
        if (bseIsWord(st, "in")) {
            bseNext(st, false)
            while (st.err == "" && st.k == "w") { bseNext(st, false) }
            if (st.err == "" && !bseIsOp(st, ";") && st.k != "n") { bseTokErr(st); return }
        }
    }
    if (bseIsOp(st, ";")) { bseNext(st, true) }
    bseSkipNl(st)
    if (bseIsWord(st, "{")) { bseGroup(st) }
    else {
        bseExpect(st, "do", 1, true)
        var n = bseList(st, ["done"])
        bseExpect(st, "done", n, false)
    }
    bseClose(st)
}
function bseCase(st) {
    bseOpen(st, "case")
    if (st.k != "w") { bseTokErr(st); return }
    bseNext(st, false)
    bseSkipNl(st)
    bseExpect(st, "in", 1, false)
    while (st.err == "") {
        bseSkipNl(st)
        if (bseIsWord(st, "esac")) { bseNext(st, false); break }
        if (bseIsOp(st, "(")) { bseNext(st, false) }
        if (st.k != "w") { bseTokErr(st); return }
        bseNext(st, false)
        while (st.err == "" && bseIsOp(st, "|")) {
            bseNext(st, false)
            if (st.k != "w") { bseTokErr(st); return }
            bseNext(st, false)
        }
        if (st.err == "" && !bseIsOp(st, ")")) { bseTokErr(st); return }
        bseNext(st, true)
        bseList(st, [";;", ";&", ";;&", "esac"])
        if (st.err != "") { return }
        if (bseIsOp(st, ";;") || bseIsOp(st, ";&") || bseIsOp(st, ";;&")) { bseNext(st, false); continue }
        if (bseIsWord(st, "esac")) { bseNext(st, false); break }
        bseTokErr(st)
    }
    bseClose(st)
}
// [[ EXPR ]], parsed as bash's cond_expr: || over && over terms. A newline may stand before
// a term, after && and || and after a complete test, not after a lone operand.
function bseCond(st) {
    bseOpen(st, "[[")
    bseCondOr(st)
    if (st.err != "") { return }
    if (!bseIsWord(st, "]]")) {
        bseCondErr(st, "syntax error in conditional expression: unexpected token `" + bseCondTok(st) + "'")
        return
    }
    bseNext(st, false)
    bseClose(st)
}
function bseCondOr(st) {
    bseCondAnd(st)
    while (st.err == "" && bseIsOp(st, "||")) { bseNext(st, false); bseCondAnd(st) }
}
function bseCondAnd(st) {
    bseCondTerm(st)
    while (st.err == "" && bseIsOp(st, "&&")) { bseNext(st, false); bseCondTerm(st) }
}
var bseUnops = "abcdefghkprstuwxOGLSNnzovR"
var bseBinops = ["==", "=", "!=", "-eq", "-ne", "-lt", "-le", "-gt", "-ge", "-nt", "-ot", "-ef"]
// An operand: a word that is not "]]".
function bseCondArg(st) { return st.k == "w" && !bseIsWord(st, "]]") }
function bseCondTerm(st) {
    bseSkipNl(st)
    if (st.err != "") { return }
    if (bseIsWord(st, "]]")) { bseCondErr(st, ""); return }
    if (bseIsWord(st, "!")) { bseNext(st, false); bseCondTerm(st); return }
    if (bseIsOp(st, "(")) {
        bseNext(st, false)
        bseCondOr(st)
        // An error inside is followed by this one, which then names no token.
        if (st.err == "cond") { st.eMsg = ((st.eMsg == "") ? "" : st.eMsg + "\n") + "expected `)'" }
        if (st.err != "") { return }
        if (!bseIsOp(st, ")")) {
            bseCondErr(st, "unexpected token `" + bseCondTok(st) + "', expected `)'")
            return
        }
        bseNext(st, false)
        bseSkipNl(st)
        return
    }
    if (st.k != "w") {
        bseCondErr(st, "unexpected token `" + bseCondTok(st) + "' in conditional command")
        return
    }
    var w = st.v
    if (w.length == 2 && w.charAt(0) == "-" && bseUnops.indexOf(w.charAt(1)) >= 0) {
        bseNext(st, false)
        if (!bseCondArg(st)) {
            bseCondErr(st, "unexpected argument `" + bseCondTok(st) + "' to conditional unary operator")
            return
        }
        bseNext(st, false)
        bseSkipNl(st)
        return
    }
    bseNext(st, false)
    if (st.err != "") { return }
    var bin = bseIsOp(st, "<") || bseIsOp(st, ">")
    var k
    for (k = 0; k < bseBinops.length; k++) { if (bseIsWord(st, bseBinops[k])) { bin = true } }
    if (bseIsWord(st, "=~")) {
        bseRegex(st)
        if (st.err != "") { return }
        bin = true
    } else if (!bin) {
        if (bseIsWord(st, "]]") || bseIsOp(st, "&&") || bseIsOp(st, "||") || bseIsOp(st, ")")) { return }
        bseCondErr(st, "unexpected token `" + bseCondTok(st) + "', conditional binary operator expected")
        return
    } else { bseNext(st, false) }
    if (!bseCondArg(st)) {
        bseCondErr(st, "unexpected argument `" + bseCondTok(st) + "' to conditional binary operator")
        return
    }
    bseNext(st, false)
    bseSkipNl(st)
}
// The right side of =~: up to a blank outside parentheses and quotes, which it may hold. An
// empty one is read as the next token.
function bseRegex(st) {
    var s = st.s
    st.pv = st.v
    while (st.i < s.length && bseBlank(s.charAt(st.i))) { st.i = st.i + 1 }
    var start = st.i
    var depth = 0
    while (st.i < s.length && st.err == "") {
        var ch = s.charAt(st.i)
        if (depth == 0 && (bseBlank(ch) || ch == "\n")) { break }
        if (ch == "(") { depth = depth + 1; st.i = st.i + 1 }
        else if (ch == ")") { if (depth == 0) { break }; depth = depth - 1; st.i = st.i + 1 }
        else if (ch == "\\") { st.i = st.i + 2 }
        else if (ch == "'") { bseSq(st, st.i + 1, false) }
        else if (ch == "\"") { bseDq(st, st.i + 1) }
        else if (depth == 0 && (ch == ";" || ch == "&" || ch == "|" || ch == "<" || ch == ">")) { break }
        else { st.i = st.i + 1 }
    }
    if (st.err != "") { return }
    if (st.i == start) { bseNext(st, false); return }
    st.k = "w"
    st.v = s.slice(start, st.i)
    st.plain = false
    st.at = start
}
