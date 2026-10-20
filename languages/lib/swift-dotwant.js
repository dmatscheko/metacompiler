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
// fit agrees on its parameter's type. A parameter typed by the function's own
// generic parameter X takes the call's expected type when the function returns
// X (`let g: T = gen(.make(5))`), and no type otherwise.
//
// Two more positions take the type of a neighbouring expression, as the grammar's
// swExprAnn answers it: an operand of == / != (swDsNoteEq), and the element
// argument of a collection method (swDsRecvCall, `xs.append(.zero)`).
//
// A member is identified by the byte offset of its dot in its file; each grammar
// registers its members there (swDsDots) and supplies swDsApply(dot, ty), which
// retargets one.
//
// The generic parameter names read for that (swDsGenNames) also mark which
// parameter binds each one as a type VALUE in the body (swDsMarkTypeParams).

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
// A variable of function type is called as a function is, so its parameter
// types type the arguments too: `let cl: (Any) -> Void` keeps cl(c ? 1.5 : 3)'s
// Int arm.
function swDsNoteFnVar(name, ann) {
    if (swDsFnResult(ann) == "") { return }
    var u = swDsUnopt(ann)
    var p = u.indexOf("(")
    var parts = swDsSplit(u, p + 1, swDsGroupEnd(u, p), ",")
    var params = []
    for (var i = 0; i < parts.length; i = i + 2) {
        var ty = stripWs(swDsDropLabel(u, parts[i], parts[i + 1]))
        if (ty != "") { params.push({label: "", type: ty, hasDefault: false, variadic: false}) }
    }
    swDsNoteSig(name, params)
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
    // `f(args)`: its expected result, for a generic f returning its own parameter.
    if (swDsIdCh(ch)) {
        var ne = a
        while (ne < z && swDsIdCh(s.charAt(ne))) { ne++ }
        if (ne < z && s.charAt(ne) == "(" && swDsGroupEnd(s, ne) == z - 1) {
            swDsCallWant["$" + c.curFile() + "@" + swDsByte(s, base, a)] = t
            return
        }
    }
    var top = swDsTop(s, a, z)
    // c ? x : y - the `?` stands apart (an optional chain's is adjacent).
    for (var k = 0; k < top.length; k++) {
        var p = top[k]
        if (s.charAt(p) == "?" && p > a && swDsWs(s.charAt(p - 1)) && s.charAt(p + 1) != "?") {
            if (swLitExist(t)) { out.push({at: swDsByte(s, base, a), lit: true}) }
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
    if (swLitExistIn(ct[ct.length - 1])) { out.push({at: swDsByte(s, base, a), lit: true}) }
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
    if (!swDsMayHold(src, 0) && !swLitMayHold(src, ty)) { return }
    var out = []
    swDsScan(src, base, 0, src.length, ty, out)
    for (var i = 0; i < out.length; i++) {
        var k = "$" + file + "@" + out[i].at
        if (out[i].lit === true) {
            if (hasOwn(swLitCells, k)) { swLitCells[k].off = true }
        } else if (out[i].clo !== true) {
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
    if (hasOwn(swDsWait, "$" + name)) {
        var ws = swDsWait["$" + name]
        for (var w = 0; w < ws.length; w++) { swDsCalls.push(ws[w]) }
        swDsWait["$" + name] = []
    }
    if (!hasOwn(swDsSigs, "$" + name)) { swDsSigs["$" + name] = [] }
    swDsSigs["$" + name].push(params)
}
// "name#k" -> the generic parameter names of the k-th signature of name, and its
// declared result type, for a function whose result IS one of them (or an
// Optional of one); the
// declaration spans [from, to) and rtxt is its result type as written.
var swDsSigGen = {}
function swDsNoteGenRet(name, from, to, rtxt) {
    var r = swDsUnopt(stripWs(rtxt))
    if (r == "" || !swDsIdCh(r.charAt(0))) { return }
    for (var i = 0; i < r.length; i++) { if (!swDsIdCh(r.charAt(i))) { return } }
    var gens = swDsGenNames(from, to)
    if (gens == null) { return }
    swDsSigGen["$" + name + "#" + (swDsSigs["$" + name].length - 1)] = {gens: gens, ret: r}
}
// The generic parameter names of the declaration spanning [from, to) - its
// `<X: P, Y>` ahead of the parameter list - or null when it has none.
function swDsGenNames(from, to) {
    var src = c.srcSlice(from, to)
    var lt = src.indexOf("<")
    var op = src.indexOf("(")
    if (lt < 0 || (op >= 0 && op < lt)) { return null }
    var gt = lt + 1
    var depth = 1
    while (gt < src.length && depth > 0) {
        if (src.charAt(gt) == "<") { depth++ } else if (src.charAt(gt) == ">") { depth-- }
        gt++
    }
    var gens = []
    var parts = swDsGenArgs("G" + stripWs(src.substring(lt, gt)), "G")
    if (parts == null) { return null }
    for (var j = 0; j < parts.length; j++) {
        var gn = parts[j]
        var col = gn.indexOf(":")
        gens.push(col < 0 ? gn : gn.substring(0, col))
    }
    return gens
}
// A generic parameter is also a VALUE in the body: X in `func f<X: P>(_ t: X.Type)`
// is the type t holds (`X.zz` reads its static zz), in `func f<X: P>(_ v: X)`
// the type of v, and in `func f<X: P>(_ xs: [X])` or `(_ xs: X...)` the type of
// the first element (an empty array leaves X unbound). The parameter that binds each
// one is marked for the prologue - tmeta for a metatype, tval for a value, telem
// for an element, in that order of preference. gens is swDsGenNames' answer.
function swDsMarkTypeParams(params, gens) {
    if (gens == null) { return }
    for (var g = 0; g < gens.length; g++) {
        var meta = null
        var val = null
        var elem = null
        for (var i = 0; i < params.length; i++) {
            var p = params[i]
            if (typeof p != "object" || p.inout || p.type == undefined) { continue }
            var t = swDsUnopt(p.type)
            if (p.variadic) {
                if (elem == null && t == gens[g]) { elem = p }
                continue
            }
            if (meta == null && t == gens[g] + ".Type") { meta = p }
            if (val == null && t == gens[g]) { val = p }
            if (elem == null && p.type == "[" + gens[g] + "]") { elem = p }
        }
        if (meta != null) { meta.tmeta = gens[g] }
        else if (val != null) { val.tval = gens[g] }
        else if (elem != null) { elem.telem = gens[g] }
    }
}

// ----- Type aliases -----
// `typealias Real = Double`, `typealias Grid<T> = [[T]]`: an annotation naming an
// alias adopts as the type it stands for, so `let r: Real = 3` holds 3.0. The
// walk expands the aliases declared above the annotation (swAliasExpand) and
// classifies the result as the annotation forms of the grammar would (swAnnRecOf).
var swAliasDefs = {}
function swAliasNote(name, params, target) { swAliasDefs["$" + name] = {ps: params, to: target} }
function swAliasExpand(t) {
    var cur = t
    for (var guard = 0; guard < 16; guard++) {
        var nx = swAliasStep(cur)
        if (nx == cur) { return cur }
        cur = nx
    }
    return cur
}
// One pass replacing each alias in t, its generic arguments substituted, by its
// target. A function type followed by `?` is parenthesised, as written in Swift.
function swAliasStep(t) {
    var out = ""
    var i = 0
    while (i < t.length) {
        if (!swDsIdCh(t.charAt(i))) { out = out + t.charAt(i); i++; continue }
        var s = i
        while (i < t.length && swDsIdCh(t.charAt(i))) { i++ }
        var tok = t.substring(s, i)
        if ((s > 0 && t.charAt(s - 1) == ".") || !hasOwn(swAliasDefs, "$" + tok)) { out = out + tok; continue }
        var d = swAliasDefs["$" + tok]
        var to = d.to
        if (d.ps.length > 0) {
            var e = swAngleEnd(t, i)
            if (e < 0) { out = out + tok; continue }
            var args = swDsGenArgs(tok + t.substring(i, e + 1), tok)
            if (args == null || args.length != d.ps.length) { out = out + tok; continue }
            to = swTySubst(to, d.ps, args)
            i = e + 1
        }
        if (to.indexOf("->") >= 0 && i < t.length && (t.charAt(i) == "?" || t.charAt(i) == "!")) { to = "(" + to + ")" }
        out = out + to
    }
    return out
}
// The index of the `>` closing the `<` at t[i], or -1.
function swAngleEnd(t, i) {
    if (i >= t.length || t.charAt(i) != "<") { return -1 }
    var depth = 0
    for (var j = i; j < t.length; j++) {
        var ch = t.charAt(j)
        if (ch == "-" && j + 1 < t.length && t.charAt(j + 1) == ">") { j++; continue }
        if (ch == "<" || ch == "[" || ch == "(") { depth++ }
        else if (ch == ">" || ch == "]" || ch == ")") { depth--; if (depth == 0) { return ch == ">" ? j : -1 } }
    }
    return -1
}
// ty with each whole identifier names[k] replaced by args[k].
function swTySubst(ty, names, args) {
    var out = ""
    var i = 0
    while (i < ty.length) {
        if (!swDsIdCh(ty.charAt(i))) { out = out + ty.charAt(i); i++; continue }
        var s = i
        while (i < ty.length && swDsIdCh(ty.charAt(i))) { i++ }
        var tok = ty.substring(s, i)
        if (s == 0 || ty.charAt(s - 1) != ".") {
            for (var k = 0; k < names.length; k++) { if (names[k] == tok) { tok = args[k]; break } }
        }
        out = out + tok
    }
    return out
}
// The record the grammar's annotation forms push for type text e (DeclAnnType):
// {itype} for a numeric name, {tlabels, atype} for a labelled tuple, {atype} for
// a function type, a structural or Optional numeric one, else swGenAnnRec's.
var swNumTyNames = "|Int64|Int32|Int16|Int8|Int|UInt64|UInt32|UInt16|UInt8|UInt|Double|Float|"
function swAnnRecOf(e) {
    if (swNumTyNames.indexOf("|" + e + "|") >= 0) { return {itype: e} }
    if (e == "Unicode.Scalar" || e == "UnicodeScalar") { return {atype: e} }
    var core = e
    while (core.length > 1 && (core.charAt(core.length - 1) == "?" || core.charAt(core.length - 1) == "!")) { core = core.substring(0, core.length - 1) }
    if (core.charAt(0) == "(") {
        var close = swDsGroupEnd(core, 0)
        if (close > 0 && core.substring(close + 1).indexOf("->") >= 0) { return {atype: e} }
        if (close == core.length - 1 && core == e) {
            var parts = swTySplit(core.substring(1, close), ",")
            var labels = []
            for (var i = 0; i < parts.length; i++) {
                var lp = swTySplit(parts[i], ":")
                labels.push(lp.length == 2 && swDsIdCh(lp[0].charAt(0)) ? lp[0] : "")
            }
            if (labels[0] != "") { return {tlabels: labels, atype: e} }
        }
        return {atype: e}
    }
    if (core.charAt(0) == "[" || swNumTyNames.indexOf("|" + core + "|") >= 0) { return {atype: e} }
    return swGenAnnRec(e)
}
// The record an annotation that no other form claimed pushes, its aliases expanded.
function swOtherAnnRec(t) {
    var e = swAliasExpand(t)
    return e == t ? swGenAnnRec(t) : swAnnRecOf(e)
}

// `var w, h: Double` declares BOTH as Double: a binding with neither an annotation
// nor an initializer takes the annotation of the binding after it. items are the
// {dbind} records of one declaration (each the item list of one binding, its name
// first); mk builds each binding once the annotations are shared.
function swBindsShared(items, mk) {
    var binds = []
    for (var i = 0; i < items.length; i++) { if (items[i] != undefined && items[i].dbind != undefined) { binds.push(items[i].dbind) } }
    for (var j = binds.length - 2; j >= 0; j--) {
        if (binds[j].length != 1) { continue }
        var nx = binds[j + 1]
        var k = 1
        while (k < nx.length && swIsAnnRec(nx[k])) { binds[j].push(nx[k]); k++ }
    }
    var out = []
    for (var b = 0; b < binds.length; b++) { out.push(mk(binds[b])) }
    return out
}
function swIsAnnRec(r) {
    return r != undefined && r != null && typeof r == "object"
        && (r.itype != undefined || r.atype != undefined || r.otype != undefined || r.tlabels != undefined)
}

// ----- A generic TYPE's parameters -----
// `struct Box<T> { var v: T }` binds T per INSTANCE: `Box<Double>(v: 2)` and
// `let b: Box<Double> = Box(v: 3)` hold 2.0 and 3.0, and so does a later
// `b.put(4)` whose parameter is a T. The instance carries its type arguments as
// text (__targs, "Double"), its descriptor the parameter names (__tgens, "T") and
// the generic stored properties (__gfields, their declared types in __gftys).
// Every adoption site inside the type whose declared type names one of its
// parameters carries that text behind a leading "=", which no type text starts
// with, and the adoption resolves it through self at run time: swGenSub in the
// interpreter, js_swgensub in the compiled halves.

// "<T:Equatable,U>" (whitespace already stripped) -> ["T", "U"].
function swGenParamNames(txt) {
    var parts = swDsGenArgs("G" + txt, "G")
    var out = []
    if (parts == null) { return out }
    for (var i = 0; i < parts.length; i++) {
        var col = parts[i].indexOf(":")
        out.push(col < 0 ? parts[i] : parts[i].substring(0, col))
    }
    return out
}
// Whether type text ty names one of names as a whole identifier - not as the
// member of a qualified name (`Outer.T`).
function swTyNames(ty, names) {
    if (names == null || names.length == 0 || ty == undefined || ty == null) { return false }
    var i = 0
    while (i < ty.length) {
        if (!swDsIdCh(ty.charAt(i))) { i++; continue }
        var s = i
        while (i < ty.length && swDsIdCh(ty.charAt(i))) { i++ }
        if (s > 0 && ty.charAt(s - 1) == ".") { continue }
        var tok = ty.substring(s, i)
        for (var k = 0; k < names.length; k++) { if (names[k] == tok) { return true } }
    }
    return false
}
// Whether ty applies a generic type of the program anywhere (`Box<Double>`,
// `[Pair<Int, Double>]`). The standard library's collections adopt through their
// own arms.
function swTyHasUserGen(ty) {
    if (ty == undefined || ty == null) { return false }
    var lt = ty.indexOf("<")
    while (lt > 0) {
        var s = lt
        while (s > 0 && swDsIdCh(ty.charAt(s - 1))) { s-- }
        var nm = ty.substring(s, lt)
        if (nm != "" && nm != "Array" && nm != "Set" && nm != "Dictionary" && nm != "Optional") { return true }
        lt = ty.indexOf("<", lt + 1)
    }
    return false
}
// A parameter that applies a program generic type to a generic parameter of its
// own function (`func f<T>(_ b: Box<T>)`) adopts nothing: nothing binds that T.
function swGenUnbind(params, gens) {
    if (gens == null) { return }
    for (var i = 0; i < params.length; i++) {
        var p = params[i]
        if (typeof p != "object" || p.adopt == undefined) { continue }
        if (swTyHasUserGen(p.adopt) && swTyNames(p.adopt, gens)) { p.adopt = "" }
    }
}
// The record an annotation that no numeric or structural form claimed pushes: one
// APPLYING a generic type of the program (`Box<Double>`, `Box<Double>?`) is an
// adopting {atype}, as `[Double]` is; anything else - one naming a generic
// parameter in scope included - is only text, {otype}.
function swGenAnnRec(t) {
    var c0 = t.charAt(0)
    if (swDsIdCh(c0) && swTyHasUserGen(t) && !swGenInScope(t)) { return {atype: t} }
    return {otype: t}
}
// Whether an element type is one applied generic type (`Box<Double>`), which
// adopts as a plain leaf does.
function swGenLeafTy(e) { return swDsIdCh(e.charAt(0)) && e.charAt(e.length - 1) == ">" && swTyHasUserGen(e) }
// The generic parameters of the functions whose declarations the walk is inside
// (`func map<U>(...)`): a type text naming one of them is no concrete type, so
// `Wrapper<U>(...)` there binds nothing. FunGenParams opens an entry and makeFun
// closes it - the one whose parameter list lies inside its declaration [from, to).
var swGenScope = []
function swGenOpen(names, pos) { swGenScope.push({names: names, pos: pos}) }
function swGenClose(from, to) {
    if (swGenScope.length == 0) { return }
    var top = swGenScope[swGenScope.length - 1]
    if (top.pos >= from && top.pos < to) { swGenScope.pop() }
}
function swGenInScope(ty) {
    for (var i = 0; i < swGenScope.length; i++) { if (swTyNames(ty, swGenScope[i].names)) { return true } }
    return false
}
// The generic parameter names of each declared type, for its extensions.
var swGenTypes = {}
// Marks the adoption sites of a generic type's members (mem is splitMembers'
// answer, tgens the parameter names): a method's or initializer's parameter and
// result, and a stored property. A site that also names a generic
// parameter of the METHOD (`func map<U>(_ x: U) -> Box<U>`) adopts nothing, as
// nothing on the instance binds U. A function type is left alone too: its
// adopting wrapper is built from the annotation at walk time.
function swGenMark(mem, tgens) {
    if (tgens == null || tgens.length == 0) { return }
    // A static member has no instance to resolve through, so its marked sites
    // adopt nothing - rather than bind the parameter's NAME.
    var fns = []
    for (var m0 = 0; m0 < mem.methods.length; m0++) { fns.push(mem.methods[m0]) }
    for (var m1 = 0; m1 < mem.statMethods.length; m1++) { fns.push(mem.statMethods[m1]) }
    for (var m = 0; m < fns.length; m++) {
        var f = fns[m]
        var own = f.gens == undefined ? null : f.gens
        for (var i = 0; i < f.params.length; i++) {
            var p = f.params[i]
            if (typeof p != "object" || p.variadic || p.type == undefined) { continue }
            if (swGenSite(p.type, tgens, own)) { p.adopt = "=" + p.type }
        }
        var rt = f.rtxt == undefined ? "" : f.rtxt
        if (swGenSite(rt, tgens, own)) { f.rtype = "=" + rt }
    }
    for (var k = 0; k < mem.fields.length; k++) {
        var fd = mem.fields[k]
        if (swGenSite(fd.ann, tgens, null)) { fd.ty = "=" + fd.ann }
    }
    // An enum payload naming a parameter adopts when the case value meets its
    // type arguments (swAdoptGen), not at construction.
    for (var c = 0; c < mem.cases.length; c++) {
        var ca = mem.cases[c].cassoc
        if (ca == null) { continue }
        for (var j = 0; j < ca.length; j++) { if (swTyNames(ca[j].cty, tgens)) { ca[j].cty = "" } }
    }
}
function swGenSite(ty, tgens, own) {
    return swTyNames(ty, tgens) && !swTyNames(ty, own) && ty.indexOf("->") < 0
}
// `class DBox: Box<Double>`: a class that is not generic itself but inherits from
// a generic one with its arguments spelled. Its descriptor records the
// superclass's parameter names and every instance is constructed bound to those
// arguments (fixed, __ftargs), so Box's `init(_ v: T)` adopts a Double. null
// for any other class.
function swGenSuperDesc(mem, superName) {
    if (superName == null || (mem.tgens != null && mem.tgens.length > 0) || !hasOwn(swGenTypes, "$" + superName)) { return null }
    var stg = swGenTypes["$" + superName]
    for (var i = 0; i < mem.inherits.length; i++) {
        var args = swDsGenArgs(mem.inherits[i], superName)
        if (args == null || args.length != stg.length) { continue }
        var tg = ""
        var fixed = ""
        for (var k = 0; k < stg.length; k++) {
            tg = tg + (k > 0 ? "," : "") + stg[k]
            fixed = fixed + (k > 0 ? "," : "") + args[k]
        }
        return {tg: tg, fs: [], ts: [], fixed: fixed}
    }
    return null
}
// What a generic ENUM's member table records: {tg: "T,U", cases: [{name, tys}]}
// for each case whose payload names a parameter, tys its payload types ("" for
// one that does not), or null.
function swGenEnumDesc(mem) {
    if (mem.tgens == null || mem.tgens.length == 0) { return null }
    var tg = ""
    for (var i = 0; i < mem.tgens.length; i++) { tg = tg + (i > 0 ? "," : "") + mem.tgens[i] }
    var cases = []
    for (var k = 0; k < mem.cases.length; k++) {
        var ca = mem.cases[k].cassoc
        if (ca == null) { continue }
        var tys = []
        var any = false
        for (var j = 0; j < ca.length; j++) {
            var t = swGenSite(ca[j].ctxt, mem.tgens, null) ? ca[j].ctxt : ""
            if (t != "") { any = true }
            tys.push(t)
        }
        if (any) { cases.push({name: mem.cases[k].cname, tys: tys}) }
    }
    return {tg: tg, cases: cases}
}
// What a generic type's descriptor records (after swGenMark): {tg: "T,U", fs: the
// generic stored properties' storage names, ts: their declared types}, or null.
function swGenDesc(mem) {
    if (mem.tgens == null || mem.tgens.length == 0) { return null }
    var tg = ""
    for (var i = 0; i < mem.tgens.length; i++) { tg = tg + (i > 0 ? "," : "") + mem.tgens[i] }
    var fs = []
    var ts = []
    for (var k = 0; k < mem.fields.length; k++) {
        var t = mem.fields[k].ty
        if (t != undefined && t.length > 0 && t.charAt(0) == "=") { fs.push(mem.fields[k].store); ts.push(t.substring(1)) }
    }
    return {tg: tg, fs: fs, ts: ts}
}
// file@offset of a call -> the type its value is expected to have.
var swDsCallWant = {}
// A parameter type as the k-th candidate of the call at key binds it: its generic
// parameter X is the call's expected type when the result is X, else unknown.
function swDsGenSubst(name, k, key, pt) {
    var gk = "$" + name + "#" + k
    if (!hasOwn(swDsSigGen, gk)) { return pt }
    var g = swDsSigGen[gk]
    var bare = swDsUnopt(pt)
    for (var i = 0; i < g.gens.length; i++) {
        if (g.gens[i] != bare) { continue }
        if (bare == g.ret && hasOwn(swDsCallWant, key)) { return swDsCallWant[key] }
        return ""
    }
    return pt
}
// A call `name(args)` spanning [from, to), trailing closures included: its
// arguments are typed once the walk is complete (swDsFlush), when every
// declaration is known. A call with no dot, `?` or `[` in it holds no implicit
// member and no literal (swLitCells) and is not kept. The first trailing closure
// carries the label "*". A call flushed before any function of its name is
// declared waits in swDsWait for the first one (a function body may call one
// declared below it).
var swDsCalls = []
var swDsWait = {}
function swDsNoteCall(name, from, to) {
    var src = c.srcSlice(from, to)
    var op = src.indexOf(name) + name.length
    if (name.length == 0 || op < name.length) { return }
    if (!swDsMayHold(src, op) && src.indexOf("?", op) < 0 && src.indexOf("[", op) < 0) { return }
    var args = swDsCallArgs(src, from, op)
    if (args.length > 0) {
        swDsCalls.push({name: name, file: c.curFile(), args: args, key: "$" + c.curFile() + "@" + swDsByte(src, from, op - name.length)})
    }
}
// The arguments of the call whose name ends at src[op] (src starts at byte from):
// {lbl, txt, at} each.
function swDsCallArgs(src, from, op) {
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
    return args
}
// A collection method `.name(args)` spanning "from:to" on a receiver of type rty:
// the argument that is an element takes the element type.
function swDsRecvCall(span, name, rty) {
    var ct = swDsCollTy(rty)
    if (ct.length != 1 || span == undefined) { return }
    var c0 = span.indexOf(":")
    var from = parseInt(span.substring(0, c0), 10)
    var src = c.srcSlice(from, parseInt(span.substring(c0 + 1, span.length), 10))
    var op = src.indexOf(name) + name.length
    if (op < name.length || !swDsMayHold(src, op)) { return }
    var args = swDsCallArgs(src, from, op)
    if (args.length == 0) { return }
    var want = ""
    if (name == "append" || name == "insert" || name == "contains" || name == "remove") {
        if (args[0].lbl == "") { want = ct[0] }
        else if (args[0].lbl == "contentsOf") { want = "[" + ct[0] + "]" }
    } else if ((name == "firstIndex" || name == "lastIndex") && args[0].lbl == "of") { want = ct[0] }
    if (want != "") { swDsWantText(args[0].txt, args[0].at, want, c.curFile()) }
}
// An == / != spanning [from, to): a leading-dot operand takes the other operand's
// type.
function swDsNoteEq(from, to) {
    var src = c.srcSlice(from, to)
    if (!swDsMayHold(src, 0)) { return }
    var top = swDsTop(src, 0, src.length)
    var ops = []
    for (var k = 0; k < top.length; k++) {
        var p = top[k]
        var two = src.substring(p, p + 2)
        if ((two == "==" || two == "!=") && src.charAt(p + 2) != "=" && (p == 0 || "=!<>".indexOf(src.charAt(p - 1)) < 0)) {
            ops.push(p)
            k++
        }
    }
    if (ops.length != 1) { return }
    var at = ops[0]
    swDsEqSide(src, from, 0, at, at + 2, src.length)
    swDsEqSide(src, from, at + 2, src.length, 0, at)
}
function swDsEqSide(src, from, a, z, oa, oz) {
    var s0 = swDsTrimStart(src, a, z)
    if (s0 >= z || src.charAt(s0) != "." || !swDsIdCh(src.charAt(s0 + 1))) { return }
    var ty = swExprAnn(stripWs(src.substring(oa, oz)))
    if (ty != "") { swDsWantText(src.substring(a, z), swDsByte(src, from, a), ty, c.curFile()) }
}
// A chain of n operands joined by binary `+` spanning [from, to): when one operand
// is an Array of an adoptable element type (swExprAnn), that is the type of every
// operand, so an array LITERAL among them adopts it - `[Double]() + [1]` is
// `[1.0]` under swiftc 6.1.2. Answers the Array type, or "".
function swDsConcatTy(from, to, n) {
    var src = c.srcSlice(from, to)
    if (src.indexOf("[") < 0) { return "" }
    var top = swDsTop(src, 0, src.length)
    var cuts = [0]
    for (var k = 0; k < top.length; k++) {
        var p = top[k]
        if (src.charAt(p) == "+" && src.charAt(p + 1) != "=" && (p == 0 || "&+".indexOf(src.charAt(p - 1)) < 0)) {
            cuts.push(p)
            cuts.push(p + 1)
        }
    }
    cuts.push(src.length)
    if (cuts.length != 2 * n) { return "" }
    var floLit = false
    for (var i = 0; i < cuts.length; i += 2) {
        var part = stripWs(src.substring(cuts[i], cuts[i + 1]))
        var ty = swExprAnn(part)
        if (ty.charAt(0) == "[" && swElemTyOf(ty) != "" && swDsSplit(ty, 1, ty.length - 1, ":").length == 2) { return ty }
        if (swDsHasFloElem(part)) { floLit = true }
    }
    // With no operand typed, an Array literal holding a float literal makes the
    // chain [Double]: `[2.5] + [3]` is [2.5, 3.0].
    return floLit ? "[Double]" : ""
}
// Is s an Array literal with a float literal (`2.5`, `-1e3`) among its elements?
function swDsHasFloElem(s) {
    if (s.length < 3 || s.charAt(0) != "[" || swDsGroupEnd(s, 0) != s.length - 1) { return false }
    var cuts = swDsSplit(s, 1, s.length - 1, ",")
    for (var i = 0; i < cuts.length; i += 2) {
        var e = s.substring(cuts[i], cuts[i + 1])
        var j = e.charAt(0) == "-" ? 1 : 0
        var digits = 0
        var flo = false
        while (j < e.length) {
            var ch = e.charAt(j)
            if (ch >= "0" && ch <= "9") { digits++ }
            else if (ch == "." || ch == "e" || ch == "E") { flo = true }
            else if (ch != "_" && ch != "-" && ch != "+") { break }
            j++
        }
        if (j == e.length && digits > 0 && flo) { return true }
    }
    return false
}
// ----- the type a literal takes from its neighbours -----
// An integer literal has no type of its own: next to a Double it IS a Double.
// `c ? 1.5 : 3`, `[1.5, 3]`, `["a": 1.5, "b": 3]` and `[[1, 2.5], [3, 4]]` are
// all Double throughout (swiftc 6.1.2). swLitTy reads the type an operand's TEXT
// shows - "#" for an integer literal, "Double" for a float literal, a collection
// literal's joined element types, else the grammar's swExprAnn - and swLitJoin
// joins two, "" when they do not visibly agree. A join that is "" or holds no
// floating-point leaf adopts nothing, so a literal with a String or an untyped
// operand among its elements (`[Any]`'s usual shape) is left alone.
function swLitTy(t) {
    if (t == "") { return "" }
    var ar = swArithTy(t)
    if (ar != null) { return ar }
    if (t.indexOf("..") >= 0) { return swExprAnn(t) }
    var a = t.charAt(0) == "-" ? 1 : 0
    if (a < t.length && t.charAt(a) >= "0" && t.charAt(a) <= "9") {
        var hex = t.substring(a, a + 2) == "0x"
        var flo = false
        for (var i = a; i < t.length; i++) {
            var ch = t.charAt(i)
            if (ch == "." || ch == "p" || (!hex && (ch == "e" || ch == "E"))) { flo = true }
            else if (!(swDsIdCh(ch) || ch == "+" || ch == "-")) { return swExprAnn(t) }
        }
        return flo ? "Double" : "#"
    }
    if (t.charAt(0) == "(" && swDsGroupEnd(t, 0) == t.length - 1) {
        var tp = swDsSplit(t, 1, t.length - 1, ",")
        if (tp.length == 2) { return swLitTy(t.substring(1, t.length - 1)) }
        // A tuple's element types, its labels dropped (adoption is positional).
        var tt = "("
        for (var q = 0; q < tp.length; q = q + 2) {
            var et = swLitTy(stripWs(t.substring(swDsLabelEnd(t, tp[q], tp[q + 1]), tp[q + 1])))
            if (et == "") { return "" }
            tt = tt + (q > 0 ? "," : "") + et
        }
        return tt + ")"
    }
    if (t.charAt(0) != "[" || t.length < 3 || swDsGroupEnd(t, 0) != t.length - 1) { return swExprAnn(t) }
    var parts = swDsSplit(t, 1, t.length - 1, ",")
    var kt = "?"
    var vt = "?"
    var dict = false
    for (var p = 0; p < parts.length; p = p + 2) {
        if (swDsTrimStart(t, parts[p], parts[p + 1]) >= parts[p + 1]) { continue }
        var kv = swDsSplit(t, parts[p], parts[p + 1], ":")
        if (p == 0) { dict = kv.length == 4 && swDsSplit(t, parts[p], parts[p + 1], "?").length == 2 }
        if (dict) {
            if (kv.length != 4) { return "" }
            kt = swLitJoin(kt, swLitTy(stripWs(t.substring(kv[0], kv[1]))))
            vt = swLitJoin(vt, swLitTy(stripWs(t.substring(kv[2], kv[3]))))
        } else {
            vt = swLitJoin(vt, swLitTy(stripWs(t.substring(parts[p], parts[p + 1]))))
        }
    }
    if (vt == "" || vt == "?" || kt == "") { return "" }
    return dict ? "[" + kt + ":" + vt + "]" : "[" + vt + "]"
}
function swLitIsNum(t) { return t == "Double" || t == "Float" || t == "CGFloat" || swIntWidthOf(t) != null }
// The type of an arithmetic expression's text from its operands'
// (`Double(s) / Double(n)` is a Double, so `c ? 0 : Double(s) / Double(n)` makes
// the 0 a Double): floating point when an operand is, "#" when every operand is
// an integer literal, "" otherwise - and null when t is not a + - * / chain at
// its top level (a comparison, a ternary, a range, a generic name all decline).
function swArithTy(t) {
    var ops = []
    var depth = 0
    var last = ""
    var j = 0
    while (j < t.length) {
        var sk = swDsSkip(t, j)
        if (sk < 0) { return null }
        if (sk != j) { j = sk; last = "\""; continue }
        var ch = t.charAt(j)
        if (ch == "(" || ch == "[" || ch == "{") { depth++ }
        else if (ch == ")" || ch == "]" || ch == "}") { depth-- }
        else if (depth == 0) {
            if (ch == "<" || ch == ">" || ch == "=" || ch == "!" || ch == "&" || ch == "|" || ch == "?"
                || ch == ":" || ch == "." || ch == "%" || ch == "^" || ch == "~") { return null }
            var bin = swDsIdCh(last) || last == ")" || last == "]" || last == "\""
            if ((ch == "+" || ch == "-" || ch == "*" || ch == "/") && bin && !swNumExpSign(t, j)) { ops.push(j) }
        }
        if (ch != " " && ch != "\t") { last = ch }
        j++
    }
    if (ops.length == 0) { return null }
    var out = "#"
    var from = 0
    for (var k = 0; k <= ops.length; k++) {
        var to = k < ops.length ? ops[k] : t.length
        var pt = swLitCoalTy(stripWs(t.substring(from, to)))
        if (pt == "") { pt = swLitTy(stripWs(t.substring(from, to))) }
        if (pt == "Double" || pt == "Float") { return pt }
        if (pt != "#") { out = "" }
        from = to + 1
    }
    return out
}
// Whether the sign at t[j] belongs to a numeral's exponent (`1e-5`).
function swNumExpSign(t, j) {
    if (j < 2 || (t.charAt(j - 1) != "e" && t.charAt(j - 1) != "E")) { return false }
    var s = j - 1
    while (s > 0 && (swDsIdCh(t.charAt(s - 1)) || t.charAt(s - 1) == ".")) { s-- }
    var c0 = t.charAt(s)
    return c0 >= "0" && c0 <= "9" && t.substring(s, s + 2) != "0x"
}
function swLitJoin(a, b) {
    if (a == "?") { return b }
    if (a == "" || b == "") { return "" }
    if (a == b) { return a }
    if (a == "#" && swLitIsNum(swDsUnopt(b))) { return b }
    if (b == "#" && swLitIsNum(swDsUnopt(a))) { return a }
    if (a.charAt(0) == "(" && b.charAt(0) == "(" && a.charAt(a.length - 1) == ")" && b.charAt(b.length - 1) == ")") {
        var ap = swDsSplit(a, 1, a.length - 1, ",")
        var bp = swDsSplit(b, 1, b.length - 1, ",")
        if (ap.length != bp.length) { return "" }
        var jt = "("
        for (var p = 0; p < ap.length; p = p + 2) {
            var ej = swLitJoin(a.substring(ap[p], ap[p + 1]), b.substring(bp[p], bp[p + 1]))
            if (ej == "") { return "" }
            jt = jt + (p > 0 ? "," : "") + ej
        }
        return jt + ")"
    }
    if (a.charAt(0) == "[" && b.charAt(0) == "[" && a.charAt(a.length - 1) == "]" && b.charAt(b.length - 1) == "]") {
        var ak = swDsSplit(a, 1, a.length - 1, ":")
        var bk = swDsSplit(b, 1, b.length - 1, ":")
        if (ak.length != bk.length) { return "" }
        if (ak.length == 2) {
            var e = swLitJoin(a.substring(1, a.length - 1), b.substring(1, b.length - 1))
            return e == "" ? "" : "[" + e + "]"
        }
        if (ak.length != 4) { return "" }
        var k = swLitJoin(a.substring(ak[0], ak[1]), b.substring(bk[0], bk[1]))
        var v = swLitJoin(a.substring(ak[2], ak[3]), b.substring(bk[2], bk[3]))
        return k == "" || v == "" ? "" : "[" + k + ":" + v + "]"
    }
    return ""
}
// The type operands of the texts txts adopt to, or "" for none: their join, when
// it has a floating-point leaf (an Int literal needs no adoption to be an Int).
function swLitCtxTy(txts) {
    var j = "?"
    for (var i = 0; i < txts.length; i++) { j = swLitJoin(j, swLitTy(txts[i])) }
    if (j == "" || j == "?") { return "" }
    var out = ""
    var flo = false
    for (var c = 0; c < j.length; c++) {
        if (j.charAt(c) == "#") { out = out + "Int" } else { out = out + j.charAt(c) }
        if (j.substring(c, c + 6) == "Double" || j.substring(c, c + 5) == "Float") { flo = true }
    }
    return flo ? out : ""
}
// The type the right side of `a ?? b` adopts from a's text: a's own type when it
// is floating point, and Double(...) / Float(...)'s result. "" for none.
function swLitCoalTy(t) {
    if (t.substring(0, 7) == "Double(") { return "Double" }
    if (t.substring(0, 6) == "Float(") { return "Float" }
    var a = swDsUnopt(swLitTy(t))
    return a == "Double" || a == "Float" ? a : ""
}
// Splits a literal's items - each operand followed by its {ltxt} text record -
// into the operands, adopted to the type their texts join to (swLitCtxTy).
function swLitOperands(items) {
    var ts = swLitPlain(items)
    var txts = []
    for (var i = 0; i < items.length; i++) {
        if (items[i] != undefined && items[i].ltxt != undefined) { txts.push(items[i].ltxt) }
    }
    if (txts.length != ts.length) { return ts }
    return swLitAdoptAll(ts, txts, swLitCtxTy(txts))
}
// The operands alone, unadopted.
function swLitPlain(items) {
    var ts = []
    for (var i = 0; i < items.length; i++) {
        if (items[i] == undefined || items[i].ltxt == undefined) { ts.push(items[i]) }
    }
    return ts
}
// An EXISTENTIAL context types each operand on its own: `f(c ? 1.5 : 3)` with
// `func f(_ x: Any)` prints 3, and `let a: [[Any]] = [[1, 2.5]]` holds the Int
// 1 (swiftc 6.1.2). Every ternary and collection literal registers a switch,
// {off}, under its offset (swDsKey); the scan of a typed slot (swDsScan) turns
// off a ternary whose own type is existential and a collection literal whose
// ELEMENTS' is (swLitExist). A literal that is itself the existential value
// still joins: `let a: Any = [1, 2.5]` is [1.0, 2.5].
// print is a host function with no signature, so its argument is matched by
// text against the literal built last (swLitStop): a ternary argument only.
var swLitLast = null
var swLitCells = {}
function swLitNote(txt, tern) {
    var cell = {off: false}
    swLitLast = {txt: txt, cell: cell, tern: tern}
    swLitCells["$" + swDsKey(up.start)] = cell
    return cell
}
function swLitStop(txt) {
    if (swLitLast == null || txt == undefined || swLitLast.txt != txt) { return }
    if (swLitLast.tern) { swLitLast.cell.off = true }
}
// Whether the type t is existential: Any, AnyObject, AnyHashable or `any P`,
// optional or not, or a variadic parameter's `Any...`.
function swLitExist(t) {
    var u = swDsUnopt(t)
    if (u.length > 3 && u.substring(u.length - 3, u.length) == "...") { u = swDsUnopt(u.substring(0, u.length - 3)) }
    if (u == "Any" || u == "AnyObject" || u == "AnyHashable") { return true }
    return u.length > 3 && u.substring(0, 3) == "any" && u.charAt(3) >= "A" && u.charAt(3) <= "Z"
}
// Whether the element type t is existential or a collection of them at any depth:
// `[[1, 2.5], [3]]` typed [[Any]] must not join its elements to [[Double]].
function swLitExistIn(t) {
    if (swLitExist(t)) { return true }
    var ct = swDsCollTy(t)
    return ct.length > 0 && swLitExistIn(ct[ct.length - 1])
}
// Whether the text src of a slot typed ty can hold a literal the scan switches off.
function swLitMayHold(src, ty) {
    if (ty.indexOf("Any") < 0 && ty.indexOf("any") < 0) { return false }
    return src.indexOf("?") >= 0 || src.indexOf("[") >= 0
}
// A Dictionary literal's entries {dk, dv, dkt, dvt}: the keys and the values
// each adopt to the type their texts join to, in place.
function swLitDictAdopt(items) {
    var ks = []
    var vs = []
    var kts = []
    var vts = []
    for (var i = 0; i < items.length; i++) {
        if (items[i].dkt == undefined) { return items }
        ks.push(items[i].dk)
        vs.push(items[i].dv)
        kts.push(items[i].dkt)
        vts.push(items[i].dvt)
    }
    var ka = swLitAdoptAll(ks, kts, swLitCtxTy(kts))
    var va = swLitAdoptAll(vs, vts, swLitCtxTy(vts))
    var out = []
    for (var j = 0; j < items.length; j++) { out.push({dk: ka[j], dv: va[j]}) }
    return out
}
// Wraps each operand thunk ts[i] whose own text is not already of type ty in the
// grammar's swAdoptThunk; ts unchanged when ty is "".
function swLitAdoptAll(ts, txts, ty) {
    if (ty == "") { return ts }
    var out = []
    for (var i = 0; i < ts.length; i++) { out.push(swLitTy(txts[i]) == ty ? ts[i] : swAdoptThunk(ts[i], ty)) }
    return out
}

// An additive chain [operand, op, operand, ...] spanning [from, to): an Array
// concatenation wraps every operand in the grammar's swAdoptThunk.
function swFoldAdd(items, from, to) {
    if (items.length < 3) { return foldBinary(items) }
    for (var i = 1; i < items.length; i += 2) { if (items[i] != "+") { return foldBinary(items) } }
    var ty = swDsConcatTy(from, to, (items.length + 1) / 2)
    if (ty == "") { return foldBinary(items) }
    var adopted = []
    for (var j = 0; j < items.length; j++) { adopted.push(j % 2 == 1 ? items[j] : swAdoptThunk(items[j], ty)) }
    return foldBinary(adopted)
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
        if (!hasOwn(swDsSigs, "$" + cl.name)) {
            if (!hasOwn(swDsWait, "$" + cl.name)) { swDsWait["$" + cl.name] = [] }
            swDsWait["$" + cl.name].push(cl)
            continue
        }
        var cands = swDsSigs["$" + cl.name]
        var tys = []
        var fits = 0
        for (var k = 0; k < cands.length; k++) {
            var bound = swDsBind(cands[k], cl.args)
            if (bound == null) { continue }
            for (var j = 0; j < cl.args.length; j++) {
                var pt = swDsGenSubst(cl.name, k, cl.key, swDsParamTy(bound[j]))
                if (fits == 0) { tys.push(pt) } else if (tys[j] != pt) { tys[j] = "" }
            }
            fits++
        }
        for (var a = 0; a < tys.length; a++) {
            if (tys[a] != "") { swDsWantText(cl.args[a].txt, cl.args[a].at, tys[a], cl.file) }
        }
    }
}
