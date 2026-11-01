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

// The diagnostic for the first option letter keyword kw does not take (bash 5.3: export
// and readonly take -aAfnp, the others -aAfFgiIlnprtux), its usage line included, or "".
// flags are the option words joined, their signs kept.
function bashDeclBadOpt(kw, flags) {
    var ok = (kw == "export" || kw == "readonly") ? "aAfnp" : "aAfFgiIlnprtux"
    var i
    for (i = 0; i < flags.length; i++) {
        var ch = flags.charAt(i)
        if (ch == "-" || ch == "+" || ok.indexOf(ch) >= 0) { continue }
        var use = "declare [-aAfFgiIlnrtux] [name[=value] ...] or declare -p [-aAfFilnrtux] [name ...]"
        if (kw == "typeset") { use = "typeset [-aAfFgiIlnrtux] name[=value] ... or typeset -p [-aAfFilnrtux] [name ...]" }
        if (kw == "local") { use = "local [option] name[=value] ..." }
        if (kw == "export") { use = "export [-fn] [name[=value] ...] or export -p [-f]" }
        if (kw == "readonly") { use = "readonly [-aAf] [name[=value] ...] or readonly -p" }
        return kw + ": -" + ch + ": invalid option\n" + kw + ": usage: " + use + "\n"
    }
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
// never evaluate SUB. Answers {text, operands, bad: [the fields that are no operand, bash's
// "not a valid identifier"]}, or null.
function bashDeclText(items, fieldsOf, quote) {
    var text = items[0].dkw
    var xr = text == "export" || text == "readonly"
    var arr = false
    var opts = true
    var operands = 0
    var bad = []
    var i
    var k
    for (i = 1; i < items.length; i++) {
        var it = items[i]
        if (it.dflag != undefined) {
            text = text + " " + it.dflag
            if (it.dflag.charAt(0) == "-" && (it.dflag.indexOf("a") >= 0 || it.dflag.indexOf("A") >= 0)) { arr = true }
            continue
        }
        if (it.dname != undefined || it.a != undefined) {
            if (it.dname == undefined && it.src == undefined) { return null }
            opts = false
            operands = operands + 1
            text = text + " " + ((it.dname == undefined) ? it.src : ((it.darr != undefined) ? it.darr : it.dname))
            continue
        }
        var fs = fieldsOf(it.dyn)
        if (fs == null) { return null }
        for (k = 0; k < fs.length; k++) {
            var f = fs[k]
            if (opts && f == "--") { opts = false; continue }
            if (opts && f.length > 1 && (f.charAt(0) == "-" || (f.charAt(0) == "+" && !xr)) && bashDeclFlagText(f)) {
                text = text + " " + f
                if (f.charAt(0) == "-" && (f.indexOf("a") >= 0 || f.indexOf("A") >= 0)) { arr = true }
                continue
            }
            opts = false
            var op = bashDeclOperand(f, arr, quote, !xr)
            if (op == "") { bad.push(f); continue }
            operands = operands + 1
            text = text + " " + op
        }
    }
    return {text: text, operands: operands, bad: bad}
}
function bashDeclFlagText(f) {
    var i
    for (i = 1; i < f.length; i++) { if ("aAirxulgnpfF".indexOf(f.charAt(i)) < 0) { return false } }
    return true
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
        j = f.indexOf("]", j)
        if (j < 0) { return "" }
        j = j + 1
        if (j == f.length) { return (elem && j > i + 2) ? f.slice(0, i) + "[0]" : "" }
    }
    var opLen = (f.charAt(j) == "+" && f.charAt(j + 1) == "=") ? 2 : 1
    if (f.charAt(j) != "=" && opLen == 1) { return "" }
    var v = f.slice(j + opLen)
    if (arr && v.length >= 2 && v.charAt(0) == "(" && v.charAt(v.length - 1) == ")") { return f.slice(0, j + opLen) + v }
    return f.slice(0, j + opLen) + quote(v)
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