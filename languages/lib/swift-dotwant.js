// swift-dotwant.js -- the expected type of an implicit member `.name` inside a
// larger expression, shared by swift-interpreter.abnf and swift-to-llvm-ir.abnf
// through include().
//
// Swift resolves `.name` against the type its context expects. Without a type
// checker the walk knows that type only for a slot with a DECLARED type: an
// annotated declaration, a declared return type, a parameter. swDsScan reads the
// value such a slot receives AS WRITTEN and hands every leading-dot member it can
// reach the type that position expects:
//
//     [e, ...]  as [X], Array<X>, Set<X>      each e as X
//     [k: v]    as [K: V], Dictionary<K, V>   each k as K, each v as V
//     (e)       as X                          e as X
//     (a, b)    as (X, Y)                     a as X, b as Y
//     c ? a : b as X                          a and b as X
//     a ?? b    as X                          b as X
//     { e }     as (...) -> R                 e as R, and every `return e` in it
//     { ... }() as X                          the closure as () -> X
//
// A call's arguments take their parameter types (swDsNoteCall): the candidates
// are every function, method and initializer of that name the program declares
// (swDsNoteSig), and an argument is typed only when every candidate whose labels
// fit agrees on its parameter's type.
//
// A member is identified by the byte offset of its dot in its file; each grammar
// registers its members there (swDsDots) and supplies swDsApply(dot, ty), which
// retargets one.

var swDsDots = {}
function swDsKey(at) { return c.curFile() + "@" + at }
function swDsNoteDot(at, dot) { swDsDots["$" + swDsKey(at)] = dot }

function swDsIdCh(ch) {
    return ch == "_" || (ch >= "a" && ch <= "z") || (ch >= "A" && ch <= "Z") || (ch >= "0" && ch <= "9")
}
function swDsWs(ch) { return ch == " " || ch == "\t" || ch == "\n" || ch == "\r" }

// The index just past the string literal opening at i (a `"` or `"""`), or -1.
// An interpolation `\(...)` inside it is skipped as a group.
function swDsStrEnd(s, i) {
    var triple = s.substring(i, i + 3) == "\"\"\""
    var j = i + (triple ? 3 : 1)
    while (j < s.length) {
        var ch = s.charAt(j)
        if (ch == "\\") {
            if (s.charAt(j + 1) == "(") {
                var e = swDsGroupEnd(s, j + 1)
                if (e < 0) { return -1 }
                j = e + 1
                continue
            }
            j = j + 2
            continue
        }
        if (triple) {
            if (s.substring(j, j + 3) == "\"\"\"") { return j + 3 }
        } else if (ch == "\"") { return j + 1 }
        j = j + 1
    }
    return -1
}
// Past the string literal or comment starting at j (a line comment ends AT its
// newline, which separates statements); j itself when none starts there; -1 for
// one that does not end.
function swDsSkip(s, j) {
    var ch = s.charAt(j)
    if (ch == "\"") { return swDsStrEnd(s, j) }
    if (ch != "/") { return j }
    var nx = s.charAt(j + 1)
    if (nx == "/") {
        var nl = s.indexOf("\n", j)
        return nl < 0 ? s.length : nl
    }
    if (nx == "*") {
        var ce = s.indexOf("*/", j + 2)
        return ce < 0 ? -1 : ce + 2
    }
    return j
}
// The index of the closer matching the ( [ or { at i, or -1.
function swDsGroupEnd(s, i) {
    var depth = 0
    var j = i
    while (j < s.length) {
        var sk = swDsSkip(s, j)
        if (sk < 0) { return -1 }
        if (sk != j) { j = sk; continue }
        var ch = s.charAt(j)
        if (ch == "(" || ch == "[" || ch == "{") { depth++ }
        else if (ch == ")" || ch == "]" || ch == "}") {
            depth--
            if (depth == 0) { return j }
            if (depth < 0) { return -1 }
        }
        j = j + 1
    }
    return -1
}
// The positions in [from, to) outside every group, string literal and comment.
function swDsTop(s, from, to) {
    var out = []
    var j = from
    while (j < to) {
        var sk = swDsSkip(s, j)
        if (sk < 0 || sk > to) { return out }
        if (sk != j) { j = sk; continue }
        var ch = s.charAt(j)
        out.push(j)
        if (ch == "(" || ch == "[" || ch == "{") {
            var ge = swDsGroupEnd(s, j)
            if (ge < 0 || ge >= to) { return out }
            j = ge + 1
            continue
        }
        j = j + 1
    }
    return out
}
// Splits [from, to) at the top-level occurrences of sep: [start, end, ...] pairs.
function swDsSplit(s, from, to, sep) {
    var top = swDsTop(s, from, to)
    var out = [from]
    for (var k = 0; k < top.length; k++) {
        if (s.charAt(top[k]) == sep) { out.push(top[k]); out.push(top[k] + 1) }
    }
    out.push(to)
    return out
}

// ----- types, as whitespace-free annotation text -----
function swDsUnopt(t) {
    var u = t
    while (u.length > 0 && (u.charAt(u.length - 1) == "?" || u.charAt(u.length - 1) == "!")) { u = u.substring(0, u.length - 1) }
    if (u.length > 10 && u.substring(0, 9) == "Optional<" && u.charAt(u.length - 1) == ">") { u = u.substring(9, u.length - 1) }
    if (u.length > 1 && u.charAt(0) == "(" && swDsGroupEnd(u, 0) == u.length - 1 && swDsSplit(u, 1, u.length - 1, ",").length == 2) {
        return swDsUnopt(u.substring(1, u.length - 1))
    }
    return u
}
// The generic arguments of `Name<...>` when t is one, else null.
function swDsGenArgs(t, name) {
    if (t.length <= name.length + 2 || t.substring(0, name.length + 1) != name + "<" || t.charAt(t.length - 1) != ">") { return null }
    var inner = t.substring(name.length + 1, t.length - 1)
    var depth = 0
    var parts = []
    var st = 0
    for (var i = 0; i < inner.length; i++) {
        var ch = inner.charAt(i)
        if (ch == "<" || ch == "(" || ch == "[") { depth++ }
        else if (ch == ">" || ch == ")" || ch == "]") { depth-- }
        else if (ch == "," && depth == 0) { parts.push(inner.substring(st, i)); st = i + 1 }
    }
    parts.push(inner.substring(st, inner.length))
    return parts
}
// [elem] for an array type, [key, value] for a dictionary type, else [].
function swDsCollTy(t) {
    var u = swDsUnopt(t)
    if (u.length > 2 && u.charAt(0) == "[" && swDsGroupEnd(u, 0) == u.length - 1) {
        var parts = swDsSplit(u, 1, u.length - 1, ":")
        if (parts.length == 2) { return [u.substring(1, u.length - 1)] }
        if (parts.length == 4) { return [u.substring(parts[0], parts[1]), u.substring(parts[2], parts[3])] }
        return []
    }
    var ga = swDsGenArgs(u, "Array")
    if (ga == null) { ga = swDsGenArgs(u, "Set") }
    if (ga != null && ga.length == 1) { return ga }
    var gd = swDsGenArgs(u, "Dictionary")
    if (gd != null && gd.length == 2) { return gd }
    return []
}
// A function type's result, or "": `(Int)->T`, `@escaping()throws->T`.
function swDsFnResult(t) {
    var u = swDsUnopt(t)
    while (u.length > 0 && u.charAt(0) == "@") {
        var k = 1
        while (k < u.length && swDsIdCh(u.charAt(k))) { k++ }
        u = u.substring(k, u.length)
    }
    if (u.length == 0 || u.charAt(0) != "(") { return "" }
    var e = swDsGroupEnd(u, 0)
    if (e < 0) { return "" }
    var a = u.indexOf("->", e)
    if (a < 0) { return "" }
    return u.substring(a + 2, u.length)
}
// A tuple type's element types, labels dropped, or [].
function swDsTupleTys(t) {
    var u = swDsUnopt(t)
    if (u.length < 2 || u.charAt(0) != "(" || swDsGroupEnd(u, 0) != u.length - 1 || u.indexOf("->") >= 0) { return [] }
    var parts = swDsSplit(u, 1, u.length - 1, ",")
    var out = []
    for (var i = 0; i < parts.length; i = i + 2) { out.push(swDsDropLabel(u, parts[i], parts[i + 1])) }
    return out
}
// The text of [from, to) without a leading `label:`.
function swDsDropLabel(s, from, to) {
    var a = swDsLabelEnd(s, from, to)
    return s.substring(a, to)
}
// Where the value of [from, to) starts past a leading `label:`.
function swDsLabelEnd(s, from, to) {
    var i = from
    while (i < to && swDsWs(s.charAt(i))) { i++ }
    var k = i
    while (k < to && swDsIdCh(s.charAt(k))) { k++ }
    if (k == i) { return from }
    var j = k
    while (j < to && swDsWs(s.charAt(j))) { j++ }
    if (j < to && s.charAt(j) == ":" && s.charAt(j + 1) != ":") { return j + 1 }
    return from
}

// ----- the scan -----
// Whether s[from...] can hold a leading-dot member at all: a `.` before a name
// that does not follow a value (`a.b`, `f().b`, `x?.b`, `1...3`). Most text
// holds none, and this one pass is what spares it the scan.
function swDsMayHold(s, from) {
    var i = s.indexOf(".", from)
    while (i >= 0) {
        var pv = i > 0 ? s.charAt(i - 1) : " "
        if (i + 1 < s.length && swDsIdCh(s.charAt(i + 1)) && !swDsIdCh(pv) && pv != ")" && pv != "]"
            && pv != "?" && pv != "!" && pv != "." && pv != "\"" && pv != ">") { return true }
        i = s.indexOf(".", i + 1)
    }
    return false
}
// The byte offset of s[i] when s[0] is at byte base.
function swDsByte(s, base, i) {
    var n = base
    for (var k = 0; k < i; k++) {
        var cc = s.charCodeAt(k)
        if (cc < 128) { n = n + 1 }
        else if (cc < 2048) { n = n + 2 }
        else if (cc >= 55296 && cc < 56320) { n = n + 4; k++ }
        else { n = n + 3 }
    }
    return n
}
// [from, to) without the whitespace and comments around it.
function swDsTrimStart(s, from, to) {
    var j = from
    while (j < to) {
        if (swDsWs(s.charAt(j))) { j++; continue }
        if (s.charAt(j) != "/") { return j }
        var sk = swDsSkip(s, j)
        if (sk <= j) { return j }
        j = sk
    }
    return to
}
function swDsTrimEnd(s, a, to) {
    var j = a
    var last = a
    while (j < to) {
        var ch = s.charAt(j)
        if (swDsWs(ch)) { j++; continue }
        var sk = swDsSkip(s, j)
        if (sk < 0) { return to }
        if (sk != j) {
            if (ch == "\"") { last = sk }
            j = sk
            continue
        }
        if (ch == "(" || ch == "[" || ch == "{") {
            var ge = swDsGroupEnd(s, j)
            if (ge < 0 || ge >= to) { return to }
            j = ge + 1
        } else { j++ }
        last = j
    }
    return last
}
// Every leading-dot member of the expression s[from, to) that ty reaches, as
// {at, ty} records pushed onto out.
function swDsScan(s, base, from, to, ty, out) {
    var t = stripWs(ty)
    if (t == "") { return }
    var a = swDsTrimStart(s, from, to)
    var z = swDsTrimEnd(s, a, to)
    if (a >= z) { return }
    var ch = s.charAt(a)
    if (ch == "." && a + 1 < z && swDsIdCh(s.charAt(a + 1))) {
        if (swDsMemberEnd(s, a + 1, z) == z) { out.push({at: swDsByte(s, base, a), ty: t, clo: false}) }
        return
    }
    // `{ ... }()`: a closure called on the spot gives its own result.
    if (ch == "{" && z - a > 3 && s.substring(z - 2, z) == "()" && swDsGroupEnd(s, a) == z - 3) {
        swDsClosure(s, base, a, z - 2, "()->" + t, out)
        return
    }
    if ((ch == "[" || ch == "(" || ch == "{") && swDsGroupEnd(s, a) == z - 1) {
        if (ch == "[") { swDsColl(s, base, a, z, t, out) }
        else if (ch == "(") { swDsParen(s, base, a, z, t, out) }
        else { swDsClosure(s, base, a, z, t, out) }
        return
    }
    var top = swDsTop(s, a, z)
    // c ? x : y - the `?` stands apart (an optional chain's is adjacent).
    for (var k = 0; k < top.length; k++) {
        var p = top[k]
        if (s.charAt(p) == "?" && p > a && swDsWs(s.charAt(p - 1)) && s.charAt(p + 1) != "?") {
            var depth = 0
            for (var m = k + 1; m < top.length; m++) {
                var q = top[m]
                var qc = s.charAt(q)
                if (qc == "?" && swDsWs(s.charAt(q - 1)) && s.charAt(q + 1) != "?") { depth++ }
                else if (qc == ":") {
                    if (depth == 0) {
                        swDsScan(s, base, p + 1, q, t, out)
                        swDsScan(s, base, q + 1, z, t, out)
                        return
                    }
                    depth--
                }
            }
            return
        }
    }
    // x ?? y: y has the whole expression's type.
    for (var k2 = top.length - 1; k2 >= 0; k2--) {
        var p2 = top[k2]
        if (s.charAt(p2) == "?" && s.charAt(p2 + 1) == "?" && p2 > a && s.charAt(p2 - 1) != "?") {
            swDsScan(s, base, p2 + 2, z, t, out)
            return
        }
    }
}
// The end of the member `.name`, `.name(args)` or `.name(args) { closure }`
// whose name starts at i.
function swDsMemberEnd(s, i, z) {
    var k = i
    while (k < z && swDsIdCh(s.charAt(k))) { k++ }
    if (k < z && s.charAt(k) == "(") {
        var e = swDsGroupEnd(s, k)
        if (e < 0) { return -1 }
        k = e + 1
    }
    var j = k
    while (j < z && swDsWs(s.charAt(j))) { j++ }
    if (j < z && s.charAt(j) == "{") {
        var ce = swDsGroupEnd(s, j)
        if (ce < 0) { return -1 }
        return ce + 1
    }
    return k
}
function swDsColl(s, base, a, z, t, out) {
    var ct = swDsCollTy(t)
    if (ct.length == 0) { return }
    var parts = swDsSplit(s, a + 1, z - 1, ",")
    for (var i = 0; i < parts.length; i = i + 2) {
        if (ct.length == 1) { swDsScan(s, base, parts[i], parts[i + 1], ct[0], out); continue }
        var kv = swDsSplit(s, parts[i], parts[i + 1], ":")
        if (kv.length != 4) { continue }
        swDsScan(s, base, kv[0], kv[1], ct[0], out)
        swDsScan(s, base, kv[2], kv[3], ct[1], out)
    }
}
function swDsParen(s, base, a, z, t, out) {
    var parts = swDsSplit(s, a + 1, z - 1, ",")
    if (parts.length == 2) {
        if (swDsLabelEnd(s, parts[0], parts[1]) == parts[0]) { swDsScan(s, base, parts[0], parts[1], t, out) }
        return
    }
    var tys = swDsTupleTys(t)
    if (tys.length * 2 != parts.length) { return }
    for (var i = 0; i < parts.length; i = i + 2) {
        swDsScan(s, base, swDsLabelEnd(s, parts[i], parts[i + 1]), parts[i + 1], tys[i / 2], out)
    }
}
// A closure's value: what its `return` statements give, which the walk parked
// under the closure's offset (swDsParkRets), or its one expression. A signature
// `params in` is skipped; a statement that merely contains `in` (`for x in xs`)
// opens with a keyword.
function swDsClosure(s, base, a, z, t, out) {
    var r = swDsFnResult(t)
    if (r == "") { return }
    out.push({at: swDsByte(s, base, a), ty: r, clo: true})
    var from = a + 1
    var top = swDsTop(s, from, z - 1)
    for (var k = 0; k < top.length; k++) {
        var p = top[k]
        if (s.substring(p, p + 2) == "in" && (p == from || !swDsIdCh(s.charAt(p - 1))) && !swDsIdCh(s.charAt(p + 2))) {
            var head = stripWs(s.substring(from, p))
            var kw = swDsFirstWord(head)
            if (head.indexOf(";") < 0 && kw != "for" && kw != "if" && kw != "case" && kw != "while" && kw != "guard"
                && kw != "let" && kw != "var" && kw != "return" && kw != "switch") { from = p + 2 }
            break
        }
    }
    var stmts = []
    var st = from
    var top2 = swDsTop(s, from, z - 1)
    for (var m = 0; m < top2.length; m++) {
        var ch = s.charAt(top2[m])
        if (ch == ";" || ch == "\n") { stmts.push(st); stmts.push(top2[m]); st = top2[m] + 1 }
    }
    stmts.push(st)
    stmts.push(z - 1)
    var live = []
    for (var i = 0; i < stmts.length; i = i + 2) {
        if (swDsTrimStart(s, stmts[i], stmts[i + 1]) < stmts[i + 1]) { live.push(stmts[i]); live.push(stmts[i + 1]) }
    }
    if (live.length == 2) {
        var b0 = swDsTrimStart(s, live[0], live[1])
        if (s.substring(b0, b0 + 6) != "return" || swDsIdCh(s.charAt(b0 + 6))) { swDsScan(s, base, live[0], live[1], r, out) }
    }
}
function swDsFirstWord(s) {
    var k = 0
    while (k < s.length && swDsIdCh(s.charAt(k))) { k++ }
    return s.substring(0, k)
}

// Retargets every member swDsScan finds in the source span "from:to" (a
// recorded `iat`) for a slot of type ty.
function swDsWantSpan(iat, ty) {
    if (iat == undefined || ty == undefined || ty == "") { return }
    var c0 = iat.indexOf(":")
    var from = parseInt(iat.substring(0, c0), 10)
    var to = parseInt(iat.substring(c0 + 1, iat.length), 10)
    swDsWantText(c.srcSlice(from, to), from, ty, c.curFile())
}
// The same for the text src, which starts at byte base of file.
function swDsWantText(src, base, ty, file) {
    if (!swDsMayHold(src, 0)) { return }
    var out = []
    swDsScan(src, base, 0, src.length, ty, out)
    for (var i = 0; i < out.length; i++) {
        var k = "$" + file + "@" + out[i].at
        if (out[i].clo !== true) {
            if (hasOwn(swDsDots, k)) { swDsApply(swDsDots[k], out[i].ty) }
        } else if (hasOwn(swDsParked, k)) {
            var ps = swDsParked[k]
            for (var j = 0; j < ps.length; j++) { swDsWantText(ps[j].txt, ps[j].at, out[i].ty, file) }
        }
    }
}

// A returned value (a `return` or a single-expression body) waits for the
// declaration enclosing it, whose result type is known only once it completes:
// swDsClaimRets hands the ones inside [from, to] that type, "" for none.
var swDsRets = []
function swDsNoteRet(rec, pos) {
    if (rec != undefined && rec != null && rec.iat != undefined) { swDsRets.push({pos: pos, iat: rec.iat}) }
}
function swDsClaimRets(from, to, ann) {
    if (swDsRets.length == 0) { return }
    var keep = []
    for (var i = 0; i < swDsRets.length; i++) {
        var e = swDsRets[i]
        if (e.pos >= from && e.pos <= to) { swDsWantSpan(e.iat, ann) } else { keep.push(e) }
    }
    swDsRets = keep
}
// A closure with no declared result type parks its returns under its own offset:
// the slot it is passed to or stored in may still give it one.
var swDsParked = {}
function swDsParkRets(from, to) {
    if (swDsRets.length == 0) { return }
    var keep = []
    var mine = []
    for (var i = 0; i < swDsRets.length; i++) {
        var e = swDsRets[i]
        if (e.pos >= from && e.pos <= to) {
            var c0 = e.iat.indexOf(":")
            var a = parseInt(e.iat.substring(0, c0), 10)
            mine.push({txt: c.srcSlice(a, parseInt(e.iat.substring(c0 + 1, e.iat.length), 10)), at: a})
        } else { keep.push(e) }
    }
    swDsRets = keep
    if (mine.length > 0) { swDsParked["$" + swDsKey(from)] = mine }
}

// ----- call arguments -----
// name -> every parameter list declared under it.
var swDsSigs = {}
function swDsNoteSig(name, params) {
    if (!hasOwn(swDsSigs, "$" + name)) { swDsSigs["$" + name] = [] }
    swDsSigs["$" + name].push(params)
}
// A call `name(args)` spanning [from, to), trailing closures included: its
// arguments are typed once the walk is complete (swDsFlush), when every
// declaration is known. A call with no dot in it holds no implicit member and
// is not kept. The first trailing closure carries the label "*".
var swDsCalls = []
function swDsNoteCall(name, from, to) {
    var src = c.srcSlice(from, to)
    var op = src.indexOf(name) + name.length
    if (name.length == 0 || op < name.length || !swDsMayHold(src, op)) { return }
    var args = []
    var k = op
    if (src.charAt(op) == "(") {
        var ce = swDsGroupEnd(src, op)
        if (ce < 0) { return }
        var parts = swDsSplit(src, op + 1, ce, ",")
        for (var i = 0; i < parts.length; i = i + 2) {
            var ve = swDsLabelEnd(src, parts[i], parts[i + 1])
            var lbl = ve == parts[i] ? "" : stripWs(src.substring(parts[i], ve - 1))
            var txt = src.substring(ve, parts[i + 1])
            if (stripWs(txt) == "") { continue }
            args.push({lbl: lbl, txt: txt, at: swDsByte(src, from, ve)})
        }
        k = ce + 1
    }
    var tlbl = "*"
    while (k < src.length) {
        var j = k
        while (j < src.length && swDsWs(src.charAt(j))) { j++ }
        if (tlbl != "*") {
            var le = swDsLabelEnd(src, j, src.length)
            if (le == j) { break }
            tlbl = stripWs(src.substring(j, le - 1))
            j = le
            while (j < src.length && swDsWs(src.charAt(j))) { j++ }
        }
        if (j >= src.length || src.charAt(j) != "{") { break }
        var be = swDsGroupEnd(src, j)
        if (be < 0) { break }
        args.push({lbl: tlbl, txt: src.substring(j, be + 1), at: swDsByte(src, from, j)})
        tlbl = ""
        k = be + 1
    }
    if (args.length > 0) { swDsCalls.push({name: name, file: c.curFile(), args: args}) }
}
// The parameter each argument binds in params, or null when the labels do not
// fit: in order, a parameter with a default may be skipped, a variadic one also
// takes the unlabelled arguments that follow it, and the first trailing closure
// binds the next parameter of function type (SE-0286). A parameter left over
// must have a default, or be a function a trailing closure may fill.
function swDsBind(params, args) {
    var out = []
    var p = 0
    var i = 0
    while (i < args.length) {
        if (p >= params.length) { return null }
        var prm = params[p]
        var fits = args[i].lbl == "*" ? swDsFnResult(swDsParamTy(prm)) != "" : prm.label == args[i].lbl
        if (fits) {
            out.push(prm)
            i++
            if (prm.variadic) {
                while (i < args.length && args[i].lbl == "") { out.push(prm); i++ }
            }
        } else if (!prm.hasDefault && !prm.variadic) { return null }
        p++
    }
    while (p < params.length) {
        if (!params[p].hasDefault && !params[p].variadic && swDsFnResult(swDsParamTy(params[p])) == "") { return null }
        p++
    }
    return out
}
// A memberwise initializer's parameter carries its property's annotation as ann.
function swDsParamTy(prm) {
    var t = prm.type == undefined || prm.type == null ? "" : stripWs(prm.type)
    if (t == "" && prm.ann != undefined && prm.ann != null) { t = stripWs(prm.ann) }
    if (prm.auto === true) { return swDsFnResult(t) }
    return t
}
function swDsFlush() {
    if (swDsCalls.length == 0) { return }
    var calls = swDsCalls
    swDsCalls = []
    for (var i = 0; i < calls.length; i++) {
        var cl = calls[i]
        if (!hasOwn(swDsSigs, "$" + cl.name)) { continue }
        var cands = swDsSigs["$" + cl.name]
        var tys = []
        var fits = 0
        for (var k = 0; k < cands.length; k++) {
            var bound = swDsBind(cands[k], cl.args)
            if (bound == null) { continue }
            for (var j = 0; j < cl.args.length; j++) {
                var pt = swDsParamTy(bound[j])
                if (fits == 0) { tys.push(pt) } else if (tys[j] != pt) { tys[j] = "" }
            }
            fits++
        }
        for (var a = 0; a < tys.length; a++) {
            if (tys[a] != "") { swDsWantText(cl.args[a].txt, cl.args[a].at, tys[a], cl.file) }
        }
    }
}
