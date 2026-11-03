// bash-deparse.js -- a function definition as bash prints it (`type f`, `declare -f f`,
// `command -V f`). Included by bash-interpreter.abnf and bash-to-llvm-ir.abnf.
//
// bash keeps no source text of a function: it reprints the parsed body (print_cmd.c).
// This re-parses the definition's source and prints it the same way, measured against
// bash 5.3: the head is `NAME () ` and the body `{ ` ... `}` with every command on its own
// line, indented four blanks a level and ended by ";" unless it is the last of its list;
// `a & b` stays on one line; comments and blank lines are gone. A word is printed as
// written, except that a $( ) inside it is printed again (`$(echo  hi)` is `$(echo hi)`, a
// newline in it kept); blanks between words become one. Redirections print as `> file`,
// a duplication with its descriptor (`>&2` is `1>&2`); `elif` is an `else` holding an
// `if`; `for x` is `for x in "$@"`; `[[ a ]]` is `[[ -n a ]]`; (( )) is kept as written.
//
// Not modelled, and refused (bashDeparse answers {err}): here-documents, process
// substitution, coprocesses, and a compound command inside a $( ).

// {text} or {err: what is not supported} for the definition src, NAME() BODY or
// function NAME [()] BODY.
function bashDeparse(src) {
    var st = {s: src, i: 0, err: "", peeked: null}
    dpSkipNl(st)
    var fd = dpFuncDef(st)
    if (st.err == "" && fd == null) { st.err = "a function definition" }
    if (st.err != "") { return {err: st.err} }
    dpPrintErr = ""
    var text = dpPrintFunc(fd, "", true)
    if (dpPrintErr != "") { return {err: dpPrintErr} }
    return {text: text}
}

// Whether the text of a bad ${...}, t, is [!]NAME[ with that "[" never closed: in double
// quotes bash then looks past the "}" for the "]" and reports the whole word.
function bashSubUnclosed(t) {
    var i = (t.charAt(0) == "!") ? 1 : 0
    var n0 = i
    while (i < t.length) {
        var cc = t.charCodeAt(i)
        var w = (cc >= 97 && cc <= 122) || (cc >= 65 && cc <= 90) || cc == 95
        if (!w && !(i > n0 && cc >= 48 && cc <= 57)) { break }
        i = i + 1
    }
    if (i == n0 || t.charAt(i) != "[") { return false }
    var depth = 0
    for (; i < t.length; i++) {
        if (t.charAt(i) == "[") { depth = depth + 1 }
        else if (t.charAt(i) == "]") { depth = depth - 1; if (depth == 0) { return false } }
    }
    return true
}

// The text a "bad substitution" error names (bash 5.3): dq, the double-quoted string the
// ${...} is in with its quotes, else word, its whole word; each with a $( ) in it printed
// again (dpComsub). "" when neither was claimed. A word this printer cannot read (a
// here-document body) is kept as written.
function bashBadSubWord(dq, word) {
    var w = (dq != "") ? dq : word
    if (w == "") { return "" }
    var st = {s: w, i: 0, err: "", peeked: null}
    var d = dpWordText(st)
    return (st.err == "" && st.i == w.length) ? d : w
}

// The inside of a double-quoted string s ("..." or $"..."), with \" shown as ", which is
// how bash's parser leaves it for an error message.
function bashDqBody(s) {
    var i = (s.charAt(0) == "$") ? 2 : 1
    var out = ""
    for (; i < s.length - 1; i++) {
        var ch = s.charAt(i)
        if (ch == "\\") {
            out = out + ((s.charAt(i + 1) == "\"") ? "\"" : s.slice(i, i + 2))
            i = i + 1
            continue
        }
        out = out + ch
    }
    return out
}

// An expansion's value in a subscript of an arithmetic text that holds an expansion,
// backslash-quoted so it is not expanded again: \ $ ` " ' as bash quotes them, and [ ]
// so a bracket it holds neither nests nor ends the subscript (bash 5.3: with g='1]',
// $(( a[$g+1] )) is an error at "]+1"). Twin of rt_arsubq.
function bashArSubEsc(s) {
    var out = ""
    var i
    for (i = 0; i < s.length; i++) {
        var ch = s.charAt(i)
        if (ch == "\\" || ch == "$" || ch == "`" || ch == "\"" || ch == "'" || ch == "[" || ch == "]") { out = out + "\\" }
        out = out + ch
    }
    return out
}

// A subscript's text with bashArSubEsc's quoting of [ and ] taken off (all: of every
// character, an associative array's key, which is not expanded again). Twin of ar_unesc.
function bashArUnEsc(s, all) {
    var out = ""
    var i
    for (i = 0; i < s.length; i++) {
        var ch = s.charAt(i)
        if (ch == "\\" && i + 1 < s.length) {
            var nx = s.charAt(i + 1)
            out = out + ((all || nx == "[" || nx == "]") ? nx : ch + nx)
            i = i + 1
            continue
        }
        out = out + ch
    }
    return out
}

// names sorted by their characters' codes, as `declare -f` lists functions.
function bashSortNames(names) {
    var out = names.slice(0)
    var i
    for (i = 1; i < out.length; i++) {
        var v = out[i]
        var j = i - 1
        while (j >= 0 && out[j] > v) { out[j + 1] = out[j]; j = j - 1 }
        out[j + 1] = v
    }
    return out
}

// ---- lexer

function dpIsBlank(ch) { return ch == " " || ch == "\t" || ch == "\r" }
function dpIsMeta(ch) { return ch == " " || ch == "\t" || ch == "\r" || ch == "\n" || ch == ";" || ch == "&" || ch == "|" || ch == "<" || ch == ">" || ch == "(" || ch == ")" }
function dpIsDigits(w) {
    if (w.length == 0) { return false }
    var k
    for (k = 0; k < w.length; k++) { var c = w.charAt(k); if (c < "0" || c > "9") { return false } }
    return true
}
function dpIsAsgPrefix(w) {
    var k = 0
    while (k < w.length) {
        var c = w.charAt(k)
        if (!(c == "_" || (c >= "a" && c <= "z") || (c >= "A" && c <= "Z") || (k > 0 && c >= "0" && c <= "9"))) { break }
        k = k + 1
    }
    if (k == 0) { return false }
    if (w.charAt(k) == "[") { var e = w.indexOf("]", k); if (e < 0) { return false }; k = e + 1 }
    if (w.charAt(k) == "+") { k = k + 1 }
    return k == w.length - 1 && w.charAt(k) == "="
}

// Blanks, line continuations and a comment; never a newline.
function dpSkip(st) {
    var s = st.s
    while (st.i < s.length) {
        var ch = s.charAt(st.i)
        if (dpIsBlank(ch)) { st.i = st.i + 1 }
        else if (ch == "\\" && s.charAt(st.i + 1) == "\n") { st.i = st.i + 2 }
        else if (ch == "#") { while (st.i < s.length && s.charAt(st.i) != "\n") { st.i = st.i + 1 } }
        else { return }
    }
}

// The end (exclusive) of the bracketed text that opens just before at, `open`/`close`
// being "(" / ")" or "{" / "}", quotes and backslashes respected; -1 when it is unclosed.
function dpMatch(s, at, open, close) {
    var d = 1
    var k = at
    while (k < s.length) {
        var ch = s.charAt(k)
        if (ch == "\\") { k = k + 2; continue }
        if (ch == "'" && open == "(") { var q = s.indexOf("'", k + 1); if (q < 0) { return -1 }; k = q + 1; continue }
        if (ch == "\"") {
            k = k + 1
            while (k < s.length && s.charAt(k) != "\"") { if (s.charAt(k) == "\\") { k = k + 1 }; k = k + 1 }
            k = k + 1
            continue
        }
        if (ch == "`") { var b = s.indexOf("`", k + 1); if (b < 0) { return -1 }; k = b + 1; continue }
        if (ch == open) { d = d + 1 }
        else if (ch == close) { d = d - 1; if (d == 0) { return k + 1 } }
        k = k + 1
    }
    return -1
}

// A $( ) at s[at] = "$(": its text printed again, or "" with st.err set.
function dpComsub(st, s, at) {
    var e = dpMatch(s, at + 2, "(", ")")
    if (e < 0) { st.err = "an unclosed $("; return {text: "", end: s.length} }
    var inner = {s: s.slice(at + 2, e - 1), i: 0, err: "", peeked: null}
    var items = dpList(inner, [])
    if (inner.err == "" && dpPeek(inner).t != "eof") { inner.err = "a $( ) this printer cannot read" }
    if (inner.err != "") { st.err = inner.err; return {text: "", end: e} }
    return {text: "$(" + dpPrintList(items, "", true) + ")", end: e}
}

// One word from st.i: its printed text.
function dpWordText(st) {
    var s = st.s
    var out = ""
    while (st.i < s.length) {
        var ch = s.charAt(st.i)
        if (dpIsMeta(ch)) {
            // NAME=( ... ): an array assignment, its elements one blank apart.
            if (ch == "(" && dpIsAsgPrefix(out)) {
                st.i = st.i + 1
                var els = []
                while (true) {
                    while (st.i < s.length && (dpIsBlank(s.charAt(st.i)) || s.charAt(st.i) == "\n")) { st.i = st.i + 1 }
                    if (st.i >= s.length) { st.err = "an unclosed array assignment"; return out }
                    if (s.charAt(st.i) == ")") { st.i = st.i + 1; break }
                    var el = dpWordText(st)
                    if (st.err != "") { return out }
                    if (el == "") { st.err = "an array element"; return out }
                    els.push(el)
                }
                out = out + "(" + els.join(" ") + ")"
                continue
            }
            if ((ch == "<" || ch == ">") && s.charAt(st.i + 1) == "(") { st.err = "process substitution"; return out }
            return out
        }
        if (ch == "\\") {
            if (s.charAt(st.i + 1) == "\n") { st.i = st.i + 2; continue }
            out = out + s.slice(st.i, st.i + 2)
            st.i = st.i + 2
            continue
        }
        if (ch == "'") {
            var q = s.indexOf("'", st.i + 1)
            if (q < 0) { st.err = "an unclosed quote"; return out }
            out = out + s.slice(st.i, q + 1)
            st.i = q + 1
            continue
        }
        if (ch == "\"") {
            out = out + "\""
            st.i = st.i + 1
            while (st.i < s.length && s.charAt(st.i) != "\"") {
                var dc = s.charAt(st.i)
                if (dc == "\\") { out = out + s.slice(st.i, st.i + 2); st.i = st.i + 2 }
                else if (dc == "$" || dc == "`") { out = out + dpDollar(st); if (st.err != "") { return out } }
                else { out = out + dc; st.i = st.i + 1 }
            }
            if (st.i >= s.length) { st.err = "an unclosed quote"; return out }
            out = out + "\""
            st.i = st.i + 1
            continue
        }
        if (ch == "$" || ch == "`") { out = out + dpDollar(st); if (st.err != "") { return out }; continue }
        out = out + ch
        st.i = st.i + 1
    }
    return out
}

// A $-form or a `...` at st.i, printed: $( ) again, everything else as written.
function dpDollar(st) {
    var s = st.s
    var at = st.i
    var ch = s.charAt(at)
    var e = -1
    if (ch == "`") {
        e = at + 1
        while (e < s.length && s.charAt(e) != "`") { if (s.charAt(e) == "\\") { e = e + 1 }; e = e + 1 }
        if (e >= s.length) { st.err = "an unclosed `"; return "" }
        st.i = e + 1
        return s.slice(at, e + 1)
    }
    var n = s.charAt(at + 1)
    if (n == "(" && s.charAt(at + 2) == "(") {
        e = dpMatch(s, at + 2, "(", ")")
    } else if (n == "(") {
        var cs = dpComsub(st, s, at)
        st.i = cs.end
        return cs.text
    } else if (n == "{") {
        e = dpMatch(s, at + 2, "{", "}")
    } else if (n == "'") {
        e = at + 2
        while (e < s.length && s.charAt(e) != "'") { if (s.charAt(e) == "\\") { e = e + 1 }; e = e + 1 }
        e = (e < s.length) ? e + 1 : -1
    } else {
        st.i = at + 1
        return "$"
    }
    if (e < 0) { st.err = "an unclosed $-expansion"; return "" }
    st.i = e
    return s.slice(at, e)
}

// The next token: {t: "eof" | "nl" | "op" | "w" | "rd"}; a redirection carries its
// descriptor, operator and target word.
function dpPeek(st) {
    if (st.peeked != null) { return st.peeked }
    dpSkip(st)
    var s = st.s
    var tk = null
    if (st.i >= s.length) { tk = {t: "eof"} }
    else {
        var ch = s.charAt(st.i)
        var two = s.slice(st.i, st.i + 2)
        var three = s.slice(st.i, st.i + 3)
        if (ch == "\n") { st.i = st.i + 1; tk = {t: "nl"} }
        else if (three == ";;&") { st.i = st.i + 3; tk = {t: "op", v: three} }
        else if (two == ";;" || two == ";&" || two == "&&" || two == "||" || two == "|&") { st.i = st.i + 2; tk = {t: "op", v: two} }
        else if (two == "&>" || ch == "<" || ch == ">") { tk = dpRedir(st, "") }
        else if (ch == ";" || ch == "&" || ch == "|" || ch == "(" || ch == ")") { st.i = st.i + 1; tk = {t: "op", v: ch} }
        else {
            var w = dpWordText(st)
            if (dpIsDigits(w) && (s.charAt(st.i) == "<" || s.charAt(st.i) == ">")) { tk = dpRedir(st, w) }
            else { tk = {t: "w", v: w} }
        }
    }
    st.peeked = tk
    return tk
}
function dpNext(st) { var tk = dpPeek(st); st.peeked = null; return tk }
function dpRedir(st, fd) {
    var s = st.s
    var ops = ["&>>", "<<<", "<<-", "&>", ">>", ">&", ">|", "<&", "<>", "<<", "<", ">"]
    var k
    var op = ""
    for (k = 0; k < ops.length; k++) {
        if (s.slice(st.i, st.i + ops[k].length) == ops[k]) { op = ops[k]; break }
    }
    st.i = st.i + op.length
    if (op == "<<" || op == "<<-") { st.err = "a here-document" }
    dpSkip(st)
    var w = dpWordText(st)
    if (w == "" && st.err == "") { st.err = "a redirection without a target" }
    return {t: "rd", fd: fd, op: op, w: w}
}
function dpIsWord(tk, v) { return tk.t == "w" && tk.v == v }
function dpIsOp(tk, v) { return tk.t == "op" && tk.v == v }
function dpSkipNl(st) { while (dpPeek(st).t == "nl") { dpNext(st) } }
function dpExpectWord(st, v) {
    dpSkipNl(st)
    var tk = dpNext(st)
    if (!dpIsWord(tk, v) && st.err == "") { st.err = "`" + v + "' where this printer expected it" }
}

// ---- parser: a command is {k: kind, ...}, a list is [{c: and-or, sep: ";" "&" "\n" ""}]

// The list up to one of the words in stops (in command position), a ")" or a case
// terminator, or the end.
function dpList(st, stops) {
    var items = []
    while (st.err == "") {
        while (true) {
            var tk0 = dpPeek(st)
            if (tk0.t == "nl" || dpIsOp(tk0, ";")) { dpNext(st) } else { break }
        }
        var tk = dpPeek(st)
        if (tk.t == "eof" || (tk.t == "op" && tk.v != "(")) { break }
        if (tk.t == "w" && stops.indexOf(tk.v) >= 0) { break }
        var ao = dpAndOr(st)
        if (st.err != "") { break }
        var sep = ""
        var nt = dpPeek(st)
        if (dpIsOp(nt, ";")) { dpNext(st); sep = ";" }
        else if (dpIsOp(nt, "&")) { dpNext(st); sep = "&" }
        else if (nt.t == "nl") { dpNext(st); sep = "\n" }
        items.push({c: ao, sep: sep})
        if (sep == "") { break }
    }
    if (st.err == "" && items.length == 0) { st.err = "an empty command list" }
    return items
}
function dpAndOr(st) {
    var parts = [dpPipeline(st)]
    var ops = []
    while (st.err == "") {
        var tk = dpPeek(st)
        if (!dpIsOp(tk, "&&") && !dpIsOp(tk, "||")) { break }
        dpNext(st)
        dpSkipNl(st)
        ops.push(tk.v)
        parts.push(dpPipeline(st))
    }
    return {parts: parts, ops: ops}
}
function dpPipeline(st) {
    var bang = false
    if (dpIsWord(dpPeek(st), "!")) { dpNext(st); bang = true }
    var cmds = [dpCommand(st)]
    var seps = []
    while (st.err == "") {
        var tk = dpPeek(st)
        if (!dpIsOp(tk, "|") && !dpIsOp(tk, "|&")) { break }
        dpNext(st)
        dpSkipNl(st)
        seps.push(tk.v)
        cmds.push(dpCommand(st))
    }
    return {bang: bang, cmds: cmds, seps: seps}
}
function dpRedirs(st) {
    var rs = []
    while (dpPeek(st).t == "rd") { rs.push(dpNext(st)) }
    return rs
}
function dpCommand(st) {
    var tk = dpPeek(st)
    var c = null
    if (tk.t == "w") {
        var v = tk.v
        if (v == "if") { c = dpIf(st) }
        else if (v == "while" || v == "until") {
            dpNext(st)
            var cond = dpList(st, ["do"])
            dpExpectWord(st, "do")
            var body = dpList(st, ["done"])
            dpExpectWord(st, "done")
            c = {k: "while", kw: v, cond: cond, body: body}
        }
        else if (v == "for" || v == "select") { c = dpFor(st) }
        else if (v == "case") { c = dpCase(st) }
        else if (v == "{") {
            dpNext(st)
            var gb = dpList(st, ["}"])
            dpExpectWord(st, "}")
            c = {k: "group", body: gb}
        }
        else if (v == "[[") { dpNext(st); c = dpCond(st) }
        else if (v == "function") { return dpFuncDef(st) }
        else if (v == "coproc") { st.err = "coproc"; return null }
    } else if (dpIsOp(tk, "(")) {
        // (( ... )) as written, or a subshell.
        if (st.s.charAt(st.i) == "(") {
            var e = dpMatch(st.s, st.i + 1, "(", ")")
            if (e > 0 && st.s.charAt(e) == ")") {
                c = {k: "arith", text: st.s.slice(st.i - 1, e + 1)}
                st.i = e + 1
                st.peeked = null
            }
        }
        if (c == null) {
            dpNext(st)
            var sb = dpList(st, [])
            dpSkipNl(st)
            if (!dpIsOp(dpNext(st), ")") && st.err == "") { st.err = "an unclosed ( )" }
            c = {k: "sub", body: sb}
        }
    }
    if (c != null) {
        c.rd = dpRedirs(st)
        return c
    }
    return dpSimple(st)
}
function dpSimple(st) {
    var words = []
    var rds = []
    while (st.err == "") {
        var tk = dpPeek(st)
        if (tk.t == "rd") { rds.push(dpNext(st)); continue }
        if (tk.t != "w") { break }
        dpNext(st)
        // NAME () BODY
        if (words.length == 0 && rds.length == 0 && dpIsOp(dpPeek(st), "(")) {
            dpNext(st)
            if (!dpIsOp(dpNext(st), ")")) { st.err = "a command this printer cannot read"; return null }
            return dpFuncBody(st, tk.v)
        }
        words.push(tk.v)
    }
    if (st.err == "" && words.length == 0 && rds.length == 0) { st.err = "an empty command" }
    return {k: "simple", words: words, rd: rds}
}
function dpFuncDef(st) {
    var tk = dpNext(st)
    if (dpIsWord(tk, "function")) {
        var nt = dpNext(st)
        if (nt.t != "w") { st.err = "a function name"; return null }
        if (dpIsOp(dpPeek(st), "(")) {
            dpNext(st)
            if (!dpIsOp(dpNext(st), ")")) { st.err = "a function head"; return null }
        }
        return dpFuncBody(st, nt.v)
    }
    if (tk.t != "w") { st.err = "a function name"; return null }
    if (!dpIsOp(dpNext(st), "(") || !dpIsOp(dpNext(st), ")")) { st.err = "a function head"; return null }
    return dpFuncBody(st, tk.v)
}
function dpFuncBody(st, name) {
    dpSkipNl(st)
    var body = dpCommand(st)
    if (st.err != "") { return null }
    if (body == null || body.k == "simple" || body.k == "func") { st.err = "a function body"; return null }
    return {k: "func", name: name, body: body, rd: []}
}
function dpIf(st) {
    dpNext(st)
    var cond = dpList(st, ["then"])
    dpExpectWord(st, "then")
    var body = dpList(st, ["elif", "else", "fi"])
    dpSkipNl(st)
    var tk = dpNext(st)
    var els = null
    if (dpIsWord(tk, "elif")) {
        st.peeked = {t: "w", v: "if"}
        var nested = dpIf(st)
        nested.rd = []
        els = [{c: {parts: [{bang: false, cmds: [nested], seps: []}], ops: []}, sep: ""}]
        return {k: "if", cond: cond, body: body, els: els}
    }
    if (dpIsWord(tk, "else")) {
        els = dpList(st, ["fi"])
        dpExpectWord(st, "fi")
    } else if (!dpIsWord(tk, "fi") && st.err == "") { st.err = "an unclosed if" }
    return {k: "if", cond: cond, body: body, els: els}
}
function dpFor(st) {
    var kw = dpNext(st).v
    dpSkip(st)
    if (st.s.slice(st.i, st.i + 2) == "((") {
        var e = dpMatch(st.s, st.i + 2, "(", ")")
        if (e < 0 || st.s.charAt(e) != ")") { st.err = "an unclosed for (( ))"; return null }
        var parts = st.s.slice(st.i + 2, e - 1).split(";")
        st.i = e + 1
        var k
        for (k = 0; k < parts.length; k++) { parts[k] = dpTrim(parts[k]) }
        var t = dpPeek(st)
        if (dpIsOp(t, ";")) { dpNext(st) }
        return {k: "afor", parts: parts, body: dpLoopBody(st)}
    }
    var nm = dpNext(st)
    if (nm.t != "w") { st.err = "a for variable"; return null }
    var words = null
    dpSkipNl(st)
    if (dpIsWord(dpPeek(st), "in")) {
        dpNext(st)
        words = []
        while (dpPeek(st).t == "w") { words.push(dpNext(st).v) }
    }
    var sep = dpPeek(st)
    if (dpIsOp(sep, ";") || sep.t == "nl") { dpNext(st) }
    return {k: "for", kw: kw, name: nm.v, words: words, body: dpLoopBody(st)}
}
// A for or select body: `do LIST done`, or `{ LIST }`, which prints as the former.
function dpLoopBody(st) {
    dpSkipNl(st)
    var close = dpIsWord(dpPeek(st), "{") ? "}" : "done"
    dpExpectWord(st, (close == "}") ? "{" : "do")
    var body = dpList(st, [close])
    dpExpectWord(st, close)
    return body
}
function dpCase(st) {
    dpNext(st)
    var w = dpNext(st)
    if (w.t != "w") { st.err = "a case word"; return null }
    dpExpectWord(st, "in")
    var clauses = []
    while (st.err == "") {
        dpSkipNl(st)
        var tk = dpPeek(st)
        if (dpIsWord(tk, "esac")) { dpNext(st); break }
        if (tk.t == "eof") { st.err = "an unclosed case"; break }
        if (dpIsOp(tk, "(")) { dpNext(st) }
        var pats = []
        while (st.err == "") {
            var p = dpNext(st)
            if (p.t != "w") { st.err = "a case pattern"; break }
            pats.push(p.v)
            var sep = dpNext(st)
            if (dpIsOp(sep, ")")) { break }
            if (!dpIsOp(sep, "|")) { st.err = "a case pattern"; break }
        }
        dpSkipNl(st)
        var body = null
        var t2 = dpPeek(st)
        if (!(t2.t == "op" && (t2.v == ";;" || t2.v == ";&" || t2.v == ";;&")) && !dpIsWord(t2, "esac")) { body = dpList(st, ["esac"]) }
        dpSkipNl(st)
        var term = ";;"
        var t3 = dpPeek(st)
        if (t3.t == "op" && (t3.v == ";;" || t3.v == ";&" || t3.v == ";;&")) { dpNext(st); term = t3.v }
        clauses.push({pats: pats, body: body, term: term})
    }
    return {k: "case", word: w.v, clauses: clauses}
}
function dpTrim(t) {
    var a = 0
    var z = t.length
    while (a < z && (dpIsBlank(t.charAt(a)) || t.charAt(a) == "\n")) { a = a + 1 }
    while (z > a && (dpIsBlank(t.charAt(z - 1)) || t.charAt(z - 1) == "\n")) { z = z - 1 }
    return t.slice(a, z)
}

// [[ ... ]]: its own tokens (< > ( ) are operators here), then or / and / not / term.
function dpCond(st) {
    var s = st.s
    var toks = []
    while (st.err == "") {
        while (st.i < s.length && (dpIsBlank(s.charAt(st.i)) || s.charAt(st.i) == "\n")) { st.i = st.i + 1 }
        if (st.i >= s.length) { st.err = "an unclosed [["; break }
        var two = s.slice(st.i, st.i + 2)
        var ch = s.charAt(st.i)
        if (two == "&&" || two == "||") { toks.push(two); st.i = st.i + 2; continue }
        if (ch == "(" || ch == ")" || ch == "<" || ch == ">") { toks.push(ch); st.i = st.i + 1; continue }
        var w = dpCondWord(st)
        if (w == "]]") { break }
        toks.push(w)
    }
    st.peeked = null
    var cs = {toks: toks, i: 0}
    var e = dpCondOr(cs)
    if (cs.i != toks.length && st.err == "") { st.err = "a [[ ]] this printer cannot read" }
    return {k: "cond", text: e}
}
// A word inside [[ ]]: up to a blank at parenthesis depth 0 (a regex keeps its groups).
function dpCondWord(st) {
    var s = st.s
    var out = ""
    var d = 0
    while (st.i < s.length && st.err == "") {
        var ch = s.charAt(st.i)
        if (d == 0 && (dpIsBlank(ch) || ch == "\n" || (ch == ")" && out == ""))) { break }
        if (ch == "(") { d = d + 1; out = out + ch; st.i = st.i + 1; continue }
        if (ch == ")") { if (d == 0) { break }; d = d - 1; out = out + ch; st.i = st.i + 1; continue }
        if (d == 0 && dpIsMeta(ch)) { break }
        var save = st.i
        var piece = dpWordText(st)
        if (st.i == save) { out = out + ch; st.i = st.i + 1 } else { out = out + piece }
    }
    return out
}
var dpCondUnary = " -a -b -c -d -e -f -g -h -k -p -r -s -t -u -w -x -G -L -N -O -S -z -n -o -v -R "
var dpCondBinary = " == = != =~ < > -eq -ne -lt -le -gt -ge -nt -ot -ef "
function dpCondOr(cs) {
    var l = dpCondAnd(cs)
    while (cs.i < cs.toks.length && cs.toks[cs.i] == "||") { cs.i = cs.i + 1; l = l + " || " + dpCondAnd(cs) }
    return l
}
function dpCondAnd(cs) {
    var l = dpCondNot(cs)
    while (cs.i < cs.toks.length && cs.toks[cs.i] == "&&") { cs.i = cs.i + 1; l = l + " && " + dpCondNot(cs) }
    return l
}
function dpCondNot(cs) {
    if (cs.i < cs.toks.length && cs.toks[cs.i] == "!") { cs.i = cs.i + 1; return "! " + dpCondNot(cs) }
    if (cs.i < cs.toks.length && cs.toks[cs.i] == "(") {
        cs.i = cs.i + 1
        var g = dpCondOr(cs)
        if (cs.i < cs.toks.length && cs.toks[cs.i] == ")") { cs.i = cs.i + 1 }
        return "( " + g + " )"
    }
    var t = cs.i < cs.toks.length ? cs.toks[cs.i] : ""
    cs.i = cs.i + 1
    if (dpCondUnary.indexOf(" " + t + " ") >= 0 && cs.i < cs.toks.length) {
        var nx = cs.toks[cs.i]
        if (dpCondBinary.indexOf(" " + nx + " ") < 0) { cs.i = cs.i + 1; return t + " " + nx }
    }
    if (cs.i < cs.toks.length && dpCondBinary.indexOf(" " + cs.toks[cs.i] + " ") >= 0) {
        var op = cs.toks[cs.i]
        var r = cs.i + 1 < cs.toks.length ? cs.toks[cs.i + 1] : ""
        cs.i = cs.i + 2
        return t + " " + op + " " + r
    }
    return "-n " + t
}

// ---- printer: ind is the current indentation; the first line is the caller's.

var dpPrintErr = ""     // what the printer met and does not model, "" when nothing

function dpPrintFunc(fd, ind, top) {
    var inner = ind + "    "
    var head = top ? fd.name + " () \n" : "function " + fd.name + " () \n" + ind
    var b = fd.body
    var text = ""
    if (b.k == "group") { text = head + "{ \n" + inner + dpPrintList(b.body, inner, false) + "\n" + ind + "}" + dpPrintRedirs(b.rd) }
    else { text = head + "{ \n" + inner + dpPrintCmd(b, inner, false) + "\n" + ind + "}" }
    return text
}
function dpPrintList(items, ind, inline) {
    var out = ""
    var k
    for (k = 0; k < items.length; k++) {
        out = out + dpPrintAndOr(items[k].c, ind, inline)
        var sep = items[k].sep
        if (k == items.length - 1) { if (sep == "&") { out = out + " &" }; break }
        if (sep == "&") { out = out + " & " }
        else if (!inline) { out = out + ";\n" + ind }
        else if (sep == "\n") { out = out + "\n" }
        else { out = out + "; " }
    }
    return out
}
function dpPrintAndOr(ao, ind, inline) {
    var out = dpPrintPipe(ao.parts[0], ind, inline)
    var k
    for (k = 0; k < ao.ops.length; k++) { out = out + " " + ao.ops[k] + " " + dpPrintPipe(ao.parts[k + 1], ind, inline) }
    return out
}
function dpPrintPipe(p, ind, inline) {
    var out = p.bang ? "! " : ""
    var k
    for (k = 0; k < p.cmds.length; k++) {
        if (k > 0) { out = out + " " + p.seps[k - 1] + " " }
        out = out + dpPrintCmd(p.cmds[k], ind, inline)
    }
    return out
}
function dpPrintRedirs(rs) {
    var out = ""
    var k
    for (k = 0; k < rs.length; k++) {
        var r = rs[k]
        var fd = r.fd
        var dup = (r.op == ">&" || r.op == "<&") && (dpIsDigits(r.w) || r.w == "-")
        if (dup && fd == "") { fd = (r.op == ">&") ? "1" : "0" }
        out = out + " " + fd + r.op + (dup ? "" : " ") + r.w
    }
    return out
}
function dpPrintCmd(c, ind, inline) {
    var inner = ind + "    "
    if (c.k == "simple") {
        var w = c.words.join(" ")
        var rd = dpPrintRedirs(c.rd)
        return (w == "") ? rd.slice(1) : w + rd
    }
    if (c.k == "func") { return dpPrintFunc(c, ind, false) }
    // bash prints a compound command inside a $( ) on lines of its own layout, which is not
    // modelled.
    if (inline && c.k != "arith" && c.k != "cond" && c.k != "sub") { dpPrintErr = "a compound command inside a $( )"; return "" }
    var rds = dpPrintRedirs(c.rd)
    if (c.k == "arith") { return c.text + rds }
    if (c.k == "cond") { return "[[ " + c.text + " ]]" + rds }
    if (c.k == "sub") { return "( " + dpPrintList(c.body, ind, inline) + " )" + rds }
    if (c.k == "group") { return "{ \n" + inner + dpPrintList(c.body, inner, false) + "\n" + ind + "}" + rds }
    if (c.k == "if") {
        var t = "if " + dpPrintList(c.cond, ind, false) + "; then\n" + inner + dpPrintList(c.body, inner, false) + ";\n" + ind
        if (c.els != null) { t = t + "else\n" + inner + dpPrintList(c.els, inner, false) + ";\n" + ind }
        return t + "fi" + rds
    }
    if (c.k == "while") {
        return c.kw + " " + dpPrintList(c.cond, ind, false) + "; do\n" + inner + dpPrintList(c.body, inner, false) + ";\n" + ind + "done" + rds
    }
    if (c.k == "for") {
        var ws = (c.words == null) ? "\"$@\"" : c.words.join(" ")
        return c.kw + " " + c.name + " in " + ws + ";\n" + ind + "do\n" + inner + dpPrintList(c.body, inner, false) + ";\n" + ind + "done" + rds
    }
    if (c.k == "afor") {
        return "for ((" + c.parts.join("; ") + "))\n" + ind + "do\n" + inner + dpPrintList(c.body, inner, false) + ";\n" + ind + "done" + rds
    }
    if (c.k == "case") {
        var inner2 = inner + "    "
        var cs = "case " + c.word + " in \n"
        var k
        for (k = 0; k < c.clauses.length; k++) {
            var cl = c.clauses[k]
            cs = cs + inner + cl.pats.join(" | ") + ")\n"
            if (cl.body != null) { cs = cs + inner2 + dpPrintList(cl.body, inner2, false) }
            cs = cs + "\n" + inner + cl.term + "\n"
        }
        return cs + ind + "esac" + rds
    }
    return ""
}

// Does bash 5.3 check a declaration's NAME=(...) operand by declaring it while the words
// expand (an option that changes the value's kind or case)?
function bashDeclLitChk(flags) {
    var i
    for (i = 0; i < flags.length; i++) {
        if ("aAiluc".indexOf(flags.charAt(i)) >= 0) { return true }
    }
    return false
}

// A declaration with -p, -f or -F only reports: it declares and changes nothing.
function bashDeclQuery(flags) {
    return flags.indexOf("p") >= 0 || flags.indexOf("f") >= 0 || flags.indexOf("F") >= 0
}

// Does a declaration with these options change an attribute a readonly variable keeps?
// bash 5.3 refuses -a/-A/-i/-l/-u/-c/-n and +a/+A/+i/+l/+u/+c on one ("NAME: readonly
// variable", status 1, none of the line's attributes applied); -x, -g, -r, -t, +t and +n
// are allowed, and so is everything `readonly` and `export` do.
function bashDeclRoAttr(kw, flags, plus) {
    if (kw == "export" || kw == "readonly" || bashDeclQuery(flags)) { return false }
    var i
    for (i = 0; i < flags.length; i++) {
        if ("aAilucn".indexOf(flags.charAt(i)) >= 0) { return true }
    }
    for (i = 0; i < plus.length; i++) {
        if ("aAiluc".indexOf(plus.charAt(i)) >= 0) { return true }
    }
    return false
}

// The case attribute (u, l, c or "") a declaration with these options leaves on a name that
// held cur. A name holds at most one: each of -u, -l and -c turns the other two off, a +X
// turns X off, and off wins over on (bash 5.3: `declare -ul x` and `declare -c +c x` leave
// none, `declare -u x; declare -c x` leaves -c).
function bashDeclCase(cur, flags, plus) {
    var on = ""
    var k
    for (k = 0; k < 3; k++) { if (flags.indexOf("ulc".charAt(k)) >= 0) { on = on + "ulc".charAt(k) } }
    if (on.length > 1) { return "" }
    var r = (on != "") ? on : cur
    return (r != "" && plus.indexOf(r) >= 0) ? "" : r
}

// The option letters keyword kw takes (bash 5.3: export and readonly -aAfnp, the others
// -aAcfFgiIlnprtux, though the usage line leaves -c out).
function bashDeclOptSet(kw) {
    return (kw == "export" || kw == "readonly") ? "aAfnp" : "aAcfFgiIlnprtux"
}
// The rest of bash's "invalid option" report of keyword kw, its usage line included.
function bashDeclUsage(kw) {
    var use = "declare [-aAfFgiIlnrtux] [name[=value] ...] or declare -p [-aAfFilnrtux] [name ...]"
    if (kw == "typeset") { use = "typeset [-aAfFgiIlnrtux] name[=value] ... or typeset -p [-aAfFilnrtux] [name ...]" }
    if (kw == "local") { use = "local [option] name[=value] ..." }
    if (kw == "export") { use = "export [-fn] [name[=value] ...] or export -p [-f]" }
    if (kw == "readonly") { use = "readonly [-aAf] [name[=value] ...] or readonly -p" }
    return ": invalid option\n" + kw + ": usage: " + use + "\n"
}
// The diagnostic for the first option letter keyword kw does not take (bashDeclOptSet), its
// usage line included, or "". flags are the option words, each with its sign and a space
// before it. The message names the letter with its word's sign; a sign after a word's first
// character is a letter like any other (bash 5.3: `declare +x=3` is "+=: invalid option",
// `declare --x` is "--: invalid option"). Twin of rt_opt_bad in bash-rt.c.
function bashDeclBadOpt(kw, flags) {
    var ok = bashDeclOptSet(kw)
    var sign = "-"
    var i
    for (i = 0; i < flags.length; i++) {
        var ch = flags.charAt(i)
        if (ch == " ") { i = i + 1; sign = flags.charAt(i); continue }
        if (ok.indexOf(ch) < 0) { return kw + ": " + sign + ch + bashDeclUsage(kw) }
    }
    // -a and -A both given (as -X, whatever +X follows) are refused without the usage line,
    // except by a query (bash 5.3: `declare -aA +a q` and `local -aA l` fail with status 2,
    // `declare -aAp q` prints).
    if (kw == "export" || kw == "readonly" || flags.indexOf("p") >= 0 || flags.indexOf("f") >= 0 || flags.indexOf("F") >= 0) { return "" }
    var on = ""
    sign = "-"
    for (i = 0; i < flags.length; i++) {
        var fc = flags.charAt(i)
        if (fc == "-" || fc == "+") { sign = fc } else if (sign == "-") { on = on + fc }
    }
    if (on.indexOf("a") >= 0 && on.indexOf("A") >= 0) { return kw + ": -a: invalid option\n" }
    return ""
}

// EXPORT AND READONLY TAKE NO +X: their options end at the first word not starting with
// "-", so `export +x v=1` has the operand +x, which is no name (bash 5.3: "`+x': not a
// valid identifier", status 1, and v is still exported). The option item it, next after
// core, becomes a computed operand of its one constant word (mkConst) from such a +X on.
function bashDeclPlusOp(core, it, mkConst) {
    if (it.dflag == undefined || (core[0].dkw != "export" && core[0].dkw != "readonly")) { return it }
    var ended = it.dflag.charAt(0) == "+"
    var k
    for (k = 1; k < core.length; k++) { if (core[k].dyn != undefined) { ended = true } }
    if (!ended) { return it }
    return {dyn: [mkConst(it.dflag)]}
}

// NAME[SUB]=(list) AS A DECLARATION OPERAND. bash meets it while the words are expanded,
// before the builtin runs: it first declares NAME the way `declare NAME[SUB]` does, with the
// declaration's -a/-A, -i/-l/-u and scope (local stays local; export, readonly and -g are
// global), then reports "NAME[SUB]: cannot assign list to array member" and abandons the line
// (bash 5.3: `declare -r q[1]=(x)` leaves `declare -a q`). That first declaration reports a
// refused -a/-A conversion as a literal's does ("NAME: cannot convert ...", abandoning the
// line at top level) and, in a function, goes on to the next such operand; when every one was
// refused, the builtin reports each again ("declare: NAME: ...") and fails. items are a
// declaration's. Answers one {pre: the first declaration, its operand marked {dlist}, late:
// the builtin's, lbl: the message's NAME[SUB], SUB as written} per operand written so.
function bashDeclListElems(items) {
    var kw = items[0].dkw
    var fl = ""
    var out = []
    var i
    for (i = 1; i < items.length; i++) {
        var it = items[i]
        if (it.dflag != undefined && it.dflag.charAt(0) == "-") { fl = fl + it.dflag }
        if (it.a == undefined || it.a.idxT == null || it.a.lit == null) { continue }
        var o = (fl.indexOf("g") >= 0 || kw == "export" || kw == "readonly") ? "g" : ""
        o = o + ((fl.indexOf("A") >= 0) ? "A" : ((fl.indexOf("a") >= 0) ? "a" : ""))
        var t = "ilu"
        var k
        for (k = 0; k < t.length; k++) { if (fl.indexOf(t.charAt(k)) >= 0) { o = o + t.charAt(k) } }
        var head = [{dkw: (kw == "local") ? "local" : "declare"}]
        if (o != "") { head.push({dflag: "-" + o}) }
        var pre = head.concat([{dname: it.a.name, darr: it.a.name + "[0]", dlist: true}])
        var late = head.concat([{dname: it.a.name, darr: it.a.name + "[0]"}])
        out.push({pre: pre, late: late, lbl: it.a.name + "[" + it.a.isrc + "]"})
    }
    return out
}

// Does the subscript text of a NAME[SUB] string need expanding (a $, a quote, a backslash or
// a backquote)? Without one it is its own expansion.
function bashSubNeedsX(t) {
    var i
    for (i = 0; i < t.length; i++) { if ("$'\"\\`".indexOf(t.charAt(i)) >= 0) { return true } }
    return false
}

// THE DECLARATION A COMPUTED OPERAND SPELLS (`declare -a "$spec"`, `local 'q=1'`): bash
// re-reads the declaration with the operand's fields in its place. items are a declaration's
// ({dkw}, {dflag}, {dname}, {a, src}, {dyn}); fieldsOf(dyn) gives one computed operand's
// fields, or null to give up; quote(v) spells v as one shell word. A leading -X/+X field is an
// option, NAME or NAME[SUB]=VALUE / NAME+=VALUE an operand (the value quoted, or kept as an
// array literal (...) under -a/-A), and so is NAME[SUB] for declare, typeset and local, which
// never evaluate SUB. Answers {text, operands, bad: [the diagnostics, after "KW: ", of the
// fields that are no operand, in order]}, or null: bash's "not a valid identifier", and -n's
// refusal of a NAME[SUB] whose SUB the text spells 0, which names SUB as written. An option
// field with a letter the builtin does not take answers {badOpt: bashDeclBadOpt's report}
// instead: the builtin does nothing and fails with status 2. An empty field of a `declare
// -p` or `typeset -p` is the variable named "", printed or "not found" by the engine: voids
// counts them (`local -p ""` finds none).
//
// The builtin reads the SUB of a computed NAME[SUB]=VALUE as it stands, as an assignment
// statement does (AsgIdx): an associative array's as a word, an indexed array's as
// arithmetic text, quotes and all (bash 5.3: `declare 'a[\1]=x'` is an arithmetic syntax
// error, where the written `declare a[\1]=x` is a[1]). The second is spelled with its single
// quotes and backslashes escaped, so it expands to them; kindOf(NAME) answers "A", "a", or "" when the kind is
// not known yet, and then the answer is null.
//
// An array literal written after an option field was assigned as the words were expanded,
// under the options written (fix_assignment_words never sees a computed one): {pre} is that
// declaration, run first, and the literal's name alone stays in {text} (bash 5.3: with o=-A,
// `declare $o y=([p]=1)` makes y=([0]="1"), then refuses converting it). pre is "" without one.
function bashDeclText(items, fieldsOf, quote, kindOf) {
    var text = items[0].dkw
    var pre = ""
    var sflags = text
    var fopt = false
    var xr = text == "export" || text == "readonly"
    var arr = false
    var opts = true
    var operands = 0
    var bad = []
    var assoc = false
    var nref = false
    var pq = false
    var voids = 0
    var i
    var k
    for (i = 1; i < items.length; i++) {
        var it = items[i]
        if (it.dflag != undefined) {
            text = text + " " + it.dflag
            sflags = sflags + " " + it.dflag
            if (it.dflag.charAt(0) == "-" && it.dflag.indexOf("p") >= 0) { pq = true }
            if (it.dflag.charAt(0) == "-" && it.dflag.indexOf("n") >= 0) { nref = !xr }
            if (it.dflag.charAt(0) == "-" && (it.dflag.indexOf("a") >= 0 || it.dflag.indexOf("A") >= 0)) { arr = true }
            if (it.dflag.charAt(0) == "-" && it.dflag.indexOf("A") >= 0) { assoc = true }
            continue
        }
        if (it.dname != undefined || it.a != undefined) {
            if (it.dname == undefined && it.src == undefined) { return null }
            opts = false
            operands = operands + 1
            if (fopt && it.a != undefined && it.a.lit != null) {
                pre = ((pre == "") ? sflags : pre) + " " + it.src
                text = text + " " + it.a.name
                continue
            }
            text = text + " " + ((it.dname == undefined) ? it.src : ((it.darr != undefined) ? it.darr : it.dname))
            continue
        }
        var fs = fieldsOf(it.dyn)
        if (fs == null) { return null }
        for (k = 0; k < fs.length; k++) {
            var f = fs[k]
            if (opts && f == "--") { opts = false; continue }
            var optWord = opts && f.length > 1 && (f.charAt(0) == "-" || (f.charAt(0) == "+" && !xr))
            if (optWord && !bashDeclFlagText(f)) {
                var bo = bashDeclBadOpt(items[0].dkw, " " + f)
                if (bo != "") { return {text: text, pre: "", operands: 0, bad: [], badOpt: bo} }
            }
            if (optWord && bashDeclFlagText(f)) {
                text = text + " " + f
                fopt = true
                if (f.charAt(0) == "-" && (f.indexOf("a") >= 0 || f.indexOf("A") >= 0)) { arr = true }
                if (f.charAt(0) == "-" && f.indexOf("A") >= 0) { assoc = true }
                if (f.charAt(0) == "-" && f.indexOf("n") >= 0) { nref = !xr }
                if (f.charAt(0) == "-" && f.indexOf("p") >= 0) { pq = true }
                continue
            }
            opts = false
            if (f == "" && pq && items[0].dkw == "local") { bad.push(": not found"); continue }
            if (f == "" && pq && !xr) { voids = voids + 1; continue }
            var op = bashDeclOperand(f, arr, quote, !xr)
            if (op == "") { bad.push("`" + f + "': not a valid identifier"); continue }
            var br = f.indexOf("[")
            var cl = (br > 0) ? bashSubClose(f, br) : -1
            if (nref && cl == f.length - 1 && op != f) { bad.push(f + ": reference variable cannot be an array"); continue }
            if (cl > br + 1 && cl + 1 < f.length && bashSubQuoted(f.slice(br + 1, cl))) {
                var kind = assoc ? "A" : kindOf(f.slice(0, br))
                if (kind == "") { return null }
                if (kind == "a") { op = bashDeclOperand(f.slice(0, br + 1) + bashSubEsc(f.slice(br + 1, cl)) + f.slice(cl), arr, quote, !xr) }
            }
            operands = operands + 1
            text = text + " " + op
        }
    }
    return {text: text, pre: pre, operands: operands, bad: bad, voids: voids}
}
// AN OPTION WORD AFTER AN OPERAND. bash marks every array literal (NAME=(...), NAME+=(...))
// that follows an unquoted word beginning with "-" for the kind its letters name - A
// associative, else a indexed, and g global - and applies the mark as the words are expanded
// (make_cmd.c, fix_assignment_words). The builtin itself reads such a word after an operand
// as no option but "not a valid identifier" (bash 5.3: `declare w=7 -A e=([z]=3)` makes e
// associative, status 1), and -p looks it up as a name ("-A: not found"). items are a
// declaration's, each {dyn} carrying {dsrc}, its word as written. Answers null unless a
// written word marks a literal (and no other operand is computed, and it is no -f or -F
// query); else {items: the declaration without those words, each
// marked literal given {dlate: its letters}, bad: [{at: the operands before it, w: the
// word}]}.
function bashDeclLate(items) {
    var marks = ""
    var out = []
    var bad = []
    var operands = 0
    var any = false
    var ended = false
    var i
    for (i = 0; i < items.length; i++) {
        var it = items[i]
        if (it.dflag != undefined && (it.dflag.indexOf("f") >= 0 || it.dflag.indexOf("F") >= 0)) { return null }
        if (it.dyn != undefined) {
            // "--" ends the options only where the next one could be read; anywhere else it is
            // an operand, and no mark. The marking itself ignores it (bash 5.3: `declare -- -A
            // e=(a b)` makes e associative, `declare -- -- e` refuses the second "--").
            if (it.dsrc == "--") {
                if (operands == 0 && bad.length == 0 && !ended) { ended = true } else { bad.push({at: operands, w: "--"}) }
                continue
            }
            if (it.dsrc == undefined || !bashDeclOptWord(it.dsrc)) { return null }
            var w = it.dsrc.slice(1)
            if (w.indexOf("A") >= 0 && marks.indexOf("A") < 0) { marks = marks + "A" }
            if (w.indexOf("a") >= 0 && marks.indexOf("a") < 0) { marks = marks + "a" }
            if (w.indexOf("g") >= 0 && marks.indexOf("g") < 0) { marks = marks + "g" }
            bad.push({at: operands, w: it.dsrc})
            continue
        }
        if (it.a != undefined && it.a.lit != null && it.a.idxT == null && marks != "") {
            any = true
            out.push({a: it.a, src: it.src, dlate: marks})
        } else { out.push(it) }
        if (it.a != undefined || it.dname != undefined) { operands = operands + 1 }
    }
    return any ? {items: out, bad: bad} : null
}
// A written word that is "-" and then letters, digits or _ only.
function bashDeclOptWord(s) {
    var i
    if (s.length < 2 || s.charAt(0) != "-") { return false }
    for (i = 1; i < s.length; i++) {
        var cc = s.charCodeAt(i)
        if (!(cc == 95 || (cc >= 48 && cc <= 57) || (cc >= 65 && cc <= 90) || (cc >= 97 && cc <= 122))) { return false }
    }
    return true
}
function bashDeclFlagText(f) {
    var i
    for (i = 1; i < f.length; i++) { if ("aAirxulgnpfF".indexOf(f.charAt(i)) < 0) { return false } }
    return true
}
// The index of the "]" that closes the "[" at f[j], brackets inside counted; -1 for none.
// Twin of rt_sub_close in lib/bash-rt.c.
function bashSubClose(f, j) {
    var depth = 0
    for (; j < f.length; j++) {
        if (f.charAt(j) == "[") { depth = depth + 1 }
        else if (f.charAt(j) == "]") { depth = depth - 1; if (depth == 0) { return j } }
    }
    return -1
}
// One operand field as declaration text; "" when it is no operand. elem: NAME[SUB] alone is one.
function bashDeclOperand(f, arr, quote, elem) {
    var i = 0
    while (i < f.length) {
        var cc = f.charCodeAt(i)
        if (cc == 95 || (cc >= 65 && cc <= 90) || (cc >= 97 && cc <= 122) || (i > 0 && cc >= 48 && cc <= 57)) { i = i + 1 }
        else { break }
    }
    if (i == 0) { return "" }
    if (i == f.length) { return f }
    var j = i
    if (f.charAt(j) == "[") {
        j = bashSubClose(f, j)
        if (j < 0) { return "" }
        j = j + 1
        // SUB is not evaluated. It stays when the written NAME[SUB] reads it back (declare -n
        // names it when it refuses the operand), else it is spelled 0 (bashDeclText reports
        // that refusal itself).
        if (j == f.length) {
            if (!elem || j <= i + 2) { return "" }
            var sub = f.slice(i + 1, j - 1)
            var s
            for (s = 0; s < sub.length; s++) { if (" \t\r\n;&|<>()[]'\"\\$`".indexOf(sub.charAt(s)) >= 0) { sub = "0" } }
            return f.slice(0, i) + "[" + sub + "]"
        }
    }
    var opLen = (f.charAt(j) == "+" && f.charAt(j + 1) == "=") ? 2 : 1
    if (f.charAt(j) != "=" && opLen == 1) { return "" }
    var v = f.slice(j + opLen)
    // Under -a or -A such a value is the whole array's, whatever SUB says (bash 5.3:
    // `declare -a 'w[2]=(x y)'` is w=(x y)).
    if (arr && v.length >= 2 && v.charAt(0) == "(" && v.charAt(v.length - 1) == ")") { return f.slice(0, i) + f.slice(j, j + opLen) + v }
    return f.slice(0, j + opLen) + quote(v)
}

// Has a subscript a single quote or a backslash, which arithmetic keeps and an expansion as
// a word removes? (Arithmetic removes double quotes itself.)
function bashSubQuoted(t) { return t.indexOf("'") >= 0 || t.indexOf("\\") >= 0 }
// t with each single quote and backslash escaped: a word that expands to them.
function bashSubEsc(t) {
    var out = ""
    var i
    for (i = 0; i < t.length; i++) {
        var ch = t.charAt(i)
        out = out + ((ch == "'" || ch == "\\") ? "\\" + ch : ch)
    }
    return out
}
// v in single quotes, each ' in it spelled '\'' (bash's sh_single_quote). Twin of rt_sq in
// lib/bash-rt.c.
function bashSq(v) {
    var out = "'"
    var i
    for (i = 0; i < v.length; i++) {
        var ch = v.charAt(i)
        if (ch == "'") { out = out + "'\\''" } else { out = out + ch }
    }
    return out + "'"
}
// The synopses bash prints after a usage error of mapfile and read (optUsage).
var bashMfSyn = " [-d delim] [-n count] [-O origin] [-s count] [-t] [-u fd] [-C callback] [-c quantum] [array]"
var bashReadSyn = " [-Eers] [-a array] [-d delim] [-i text] [-n nchars] [-N nchars] [-p prompt] [-t timeout] [-u fd] [name ...]"
// An option value as bash's legal_number reads it - blanks, an optional sign, decimal
// digits, blanks - or null. Twin of rt_mf_num in lib/bash-rt.c.
function mfNum(s) {
    if (s == null) { return null }
    var i = 0
    while (i < s.length && (s.charAt(i) == " " || s.charAt(i) == "\t" || s.charAt(i) == "\n")) { i = i + 1 }
    var neg = false
    if (i < s.length && (s.charAt(i) == "-" || s.charAt(i) == "+")) { neg = s.charAt(i) == "-"; i = i + 1 }
    var d0 = i
    var n = 0
    while (i < s.length && s.charCodeAt(i) >= 48 && s.charCodeAt(i) <= 57) { n = n * 10 + (s.charCodeAt(i) - 48); i = i + 1 }
    if (i == d0) { return null }
    while (i < s.length && (s.charAt(i) == " " || s.charAt(i) == "\t" || s.charAt(i) == "\n")) { i = i + 1 }
    if (i < s.length) { return null }
    return neg ? -n : n
}
// THE OPERANDS OF `read`, as getopt reads "Eea:d:i:n:N:p:rst:u:": letters bundle (`-ra arr`),
// a value is the rest of its word or the next word, and the first word that is no option ends
// them - every word after it is a name, even one starting with "-". Answers {raw: -r,
// arr: the -a name or null, dlm: the -d delimiter ("" for -d '', "\n" without -d), nch: the
// -n/-N count (-1 without), whole: -N was given, names}. -N reads past every delimiter
// (dlm "", whatever -d says) and splits nothing; the later of -n and -N sets the count.
// The first bad option stops the parse: {bad} holds the letter of an unknown option or one
// without its value (usage, status 2), or {badNum} a count that is no int from 0 up
// ("invalid number", status 1). Twin of rt_rd_parse in lib/bash-rt.c.
function bashReadOpts(words) {
    var o = {raw: false, arr: null, dlm: "\n", nch: -1, whole: false, names: [], bad: null, missing: false, badNum: null}
    var i = 0
    while (i < words.length && o.bad == null && o.badNum == null) {
        var f = words[i]
        if (f.length < 2 || f.charAt(0) != "-") { break }
        i = i + 1
        if (f == "--") { break }
        var j = 1
        while (j < f.length) {
            var ch = f.charAt(j)
            if (ch == "r") { o.raw = true }
            if ("adinNptu".indexOf(ch) < 0) {
                if ("Eers".indexOf(ch) < 0) { o.bad = "-" + ch; break }
                j = j + 1
                continue
            }
            var v = f.slice(j + 1)
            if (v == "") {
                if (i >= words.length) { o.bad = "-" + ch; o.missing = true; break }
                v = words[i]
                i = i + 1
            }
            if (ch == "a") { o.arr = v }
            if (ch == "d") { o.dlm = v.charAt(0) }
            if (ch == "n" || ch == "N") {
                var c = mfNum(v)
                if (c == null || c < 0 || c > 2147483647) { o.badNum = v; break }
                o.nch = c
                if (ch == "N") { o.whole = true }
            }
            j = f.length
        }
    }
    if (o.whole) { o.dlm = "" }
    while (i < words.length) { o.names.push(words[i]); i = i + 1 }
    return o
}
// A name read and printf -v may store into: a NAME, or NAME[SUB] with SUB not empty and its
// brackets balanced up to the last character, as bash's valid_array_reference (`b[]` and
// `b[1]x` are "not a valid identifier"). Twin of rt_isasgref in lib/bash-rt.c.
function bashIsRef(t) {
    var i = 0
    while (i < t.length && t.charAt(i) != "[") {
        var cc = t.charCodeAt(i)
        if (!(cc == 95 || (cc >= 65 && cc <= 90) || (cc >= 97 && cc <= 122) || (i > 0 && cc >= 48 && cc <= 57))) { return false }
        i = i + 1
    }
    if (i == t.length) { return i > 0 }
    if (i == 0 || t.charAt(i + 1) == "]") { return false }
    var d = 0
    while (i < t.length) {
        if (t.charAt(i) == "[") { d = d + 1 }
        if (t.charAt(i) == "]") {
            d = d - 1
            if (d == 0) { return i == t.length - 1 }
        }
        i = i + 1
    }
    return false
}
// The diagnostic of `declare -n` (who) refusing the value v, which names no variable;
// "" when v is a NAME or NAME[SUB]. Twin of rt_nref_bad in lib/bash-rt.c.
function bashRefBad(who, v) {
    if (bashIsRef(v)) { return "" }
    return who + ": `" + v + "': " + ((v == "") ? "not a valid identifier" : "invalid variable name for name reference") + "\n"
}