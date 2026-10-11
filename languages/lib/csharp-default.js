// csharp-default.js -- the target type of the bare `default` literal and of the
// target-typed `new(...)`, for both csharp grammars (include).
//
// The literal is TARGET-TYPED (ECMA-334 12.8.21): it is the default value of the
// type its context converts it to; `new(...)` (C# 9) creates an instance of that
// type, through the same cell. Each literal owns a cell for that type, keyed
// by the literal's end offset (csDfltCell); the context fills it during the parse
// (csDfltTarget) and the literal reads it when it runs, or, in the compiler, when
// it is emitted. A cell nothing typed is null, a reference type's default. The
// first context to type a cell wins, so `object o = c ? 5 : default` keeps the int
// the conditional gave it.
//
// The contexts: a local's declaration and assignment, a field's or property's
// initializer, an assignment to a field or property (`this.F = default`,
// `o.F = default`, an unqualified one) or to an element of a local array or List,
// a `return` / `=> default` (the enclosing member's return type; the element type
// of an iterator's IEnumerable<T> for `yield return`, the T of an async member's
// Task<T>), the other branch of a conditional, a method or constructor argument
// (the parameter's declared type), and a lambda's return (the return type of the
// delegate type its declaration, assignment, argument or `return` converts it to).
// What the parse cannot see stays untyped: a member or an overload the file does
// not declare exactly once by name, a receiver's class, a type argument.
//
// The argument, field and `==` contexts name members that may be declared after
// the use, so they are recorded and resolved once the whole file has been read
// (csDfltResolve), before anything runs.

var csDfltCells = {}
function csDfltCell(end) {
    var cell = {ty: "", set: false, lam: false}
    csDfltCells["$" + end] = cell
    return cell
}
// The offset `to` with trailing blanks and one optional `closer` (`;`) skipped.
function csDfltEnd(to, closer) {
    var e = to
    while (e > 0 && c.srcSlice(e - 1, e).trim() == "") { e-- }
    if (closer != "" && c.srcSlice(e - 1, e) == closer) {
        e--
        while (e > 0 && c.srcSlice(e - 1, e).trim() == "") { e-- }
    }
    return e
}
// The cell of the default literal that ENDS the source before offset `to`; null
// when the source there is not a default literal.
function csDfltAt(to, closer) {
    var k = "$" + csDfltEnd(to, closer)
    return csDfltCells[k] != undefined ? csDfltCells[k] : null
}
// A lambda's return cell is typed only through its delegate type (csLamSet).
function csDfltSet(cell, ty) {
    if (cell == null || cell.set || cell.lam) { return }
    cell.ty = stripWs(ty)
    cell.set = true
}
function csDfltTarget(to, closer, ty) { csDfltSet(csDfltAt(to, closer), ty) }

// ----- the type names a context hands on -----

// The type arguments of a generic type name, split at the top-level commas.
function csDfltTArgs(t) {
    var s = stripWs(t)
    var a = s.indexOf("<")
    if (a < 0 || s.charAt(s.length - 1) != ">") { return [] }
    var out = []
    var depth = 0
    var from = a + 1
    for (var i = a + 1; i < s.length - 1; i++) {
        var ch = s.charAt(i)
        if (ch == "<" || ch == "(" || ch == "[") { depth++ }
        if (ch == ">" || ch == ")" || ch == "]") { depth-- }
        if (ch == "," && depth == 0) {
            out.push(s.substring(from, i))
            from = i + 1
        }
    }
    out.push(s.substring(from, s.length - 1))
    return out
}
// The generic type's own name, without namespace and type arguments.
function csDfltTName(t) {
    var s = stripWs(t)
    var a = s.indexOf("<")
    if (a >= 0) { s = s.substring(0, a) }
    var d = csDfltLastDot(s)
    return d >= 0 ? s.substring(d + 1, s.length) : s
}
// The element type of the array type or List<T> named by ty, "" for another type.
function csDfltElemOf(ty) {
    var t = stripWs(ty)
    if (t.length > 2 && t.substring(t.length - 2, t.length) == "[]") { return t.substring(0, t.length - 2) }
    var as = csDfltTArgs(t)
    if (csDfltTName(t) == "List" && as.length == 1) { return as[0] }
    return ""
}
function csDfltIsListTy(ty) { return csDfltTName(ty) == "List" }
// A declaration of a List<T> / T[] whose initializer is a collection literal
// (`= [ ... ]`, `= { ... }`): T, which the literal's array takes; "" otherwise.
function csDfltCollInit(ty, info) {
    if (info.dsrc == undefined) { return "" }
    var s = stripWs(info.dsrc)
    if (s.charAt(0) == "=") { s = s.substring(1, s.length) }
    if (s.charAt(0) != "[" && s.charAt(0) != "{") { return "" }
    return csDfltElemOf(ty)
}
// `new T[n]` / `new T[] { ... }` spanning [start, end): the element type, one
// "[]" per bracket pair after the first.
function csDfltNewArrTy(start, end) {
    var s = stripWs(c.srcSlice(start, end))
    if (s.substring(0, 3) != "new" && s.substring(0, 10) != "stackalloc") { return "" }
    var i = s.substring(0, 3) == "new" ? 3 : 10
    var from = i
    var depth = 0
    while (i < s.length && !(s.charAt(i) == "[" && depth == 0)) {
        if (s.charAt(i) == "<") { depth++ }
        if (s.charAt(i) == ">") { depth-- }
        if ((s.charAt(i) == "{" || s.charAt(i) == "(") && depth == 0) { return "" }
        i++
    }
    var ty = s.substring(from, i)
    if (ty == "" || i >= s.length) { return "" }
    var d = 0
    while (i < s.length) {
        var ch = s.charAt(i)
        if (ch == "[" || ch == "(" || ch == "{") { d++ }
        if (ch == "]" || ch == ")" || ch == "}") { d-- }
        i++
        if (d == 0) { break }
    }
    while (i + 1 < s.length && s.charAt(i) == "[" && s.charAt(i + 1) == "]") {
        ty = ty + "[]"
        i += 2
    }
    return ty
}
// `new List<T> { ... }` spanning [start, end) (the array-literal form a collection
// initializer of a List takes): T; "" for any other creation.
function csDfltNewListTy(start, end) {
    var s = stripWs(c.srcSlice(start, end))
    var b = s.indexOf("{")
    if (s.substring(0, 3) != "new" || b < 0) { return "" }
    var ty = s.substring(3, b)
    var as = csDfltTArgs(ty)
    return csDfltTName(ty) == "List" && as.length == 1 ? as[0] : ""
}
function csDfltLastDot(s) {
    var i = s.length - 1
    while (i >= 0 && s.charAt(i) != ".") { i-- }
    return i
}
// The element type `yield return` converts to: IEnumerable<T> / IEnumerator<T> /
// IAsyncEnumerable<T> give T (ECMA-334 15.15.4); the non-generic forms object.
function csDfltYieldTy(ty) {
    var n = csDfltTName(ty)
    var as = csDfltTArgs(ty)
    if (as.length == 1 && (n == "IEnumerable" || n == "IEnumerator" || n == "IAsyncEnumerable"
        || n == "IAsyncEnumerator")) { return as[0] }
    return ""
}
// The type an async member's `return` converts to: the T of Task<T> /
// ValueTask<T> (ECMA-334 15.15.1); a plain Task returns nothing.
function csDfltAsyncTy(ty) {
    var n = csDfltTName(ty)
    var as = csDfltTArgs(ty)
    if (as.length == 1 && (n == "Task" || n == "ValueTask")) { return as[0] }
    return ""
}
// The return type of a delegate type: Func's last type argument, Predicate's
// bool, Comparison's int, Converter's second, a delegate the file declares; none
// for Action and an unknown type.
var csDelegRets = {}
function csDfltDelegRet(ty) {
    var t = stripWs(ty)
    if (t.length > 0 && t.charAt(t.length - 1) == "?") { t = t.substring(0, t.length - 1) }
    var n = csDfltTName(t)
    var as = csDfltTArgs(t)
    if (n == "Func" && as.length > 0) { return as[as.length - 1] }
    if (n == "Predicate") { return "bool" }
    if (n == "Comparison") { return "int" }
    if (n == "Converter" && as.length == 2) { return as[1] }
    var r = csDelegRets["$" + n]
    return r != undefined ? r : ""
}
// `delegate R D(...);` records R for D.
function csDfltDelegDecl(start, end) {
    var s = c.srcSlice(start, end)
    var k = s.indexOf("delegate")
    var p = s.indexOf("(", k)
    if (k < 0 || p < 0) { return }
    var raw = s.substring(k + 8, p).trim()
    // The name is the last identifier before the parameters, after any type arguments
    // of the delegate itself.
    var q = raw.length
    if (raw.charAt(q - 1) == ">") {
        var depth = 0
        while (q > 0) {
            var ch = raw.charAt(q - 1)
            if (ch == ">") { depth++ }
            if (ch == "<") { depth-- }
            q--
            if (depth == 0) { break }
        }
    }
    var e = q
    while (q > 0 && csDfltIdCh(raw.charAt(q - 1))) { q-- }
    var name = raw.substring(q, e)
    var rty = stripWs(raw.substring(0, q))
    if (rty.length > 3 && rty.substring(0, 3) == "ref") { rty = rty.substring(3, rty.length) }
    if (rty.length > 8 && rty.substring(0, 8) == "readonly") { rty = rty.substring(8, rty.length) }
    csDelegRets["$" + name] = rty == "void" ? "" : rty
}
function csDfltIdCh(ch) {
    return (ch >= "a" && ch <= "z") || (ch >= "A" && ch <= "Z") || (ch >= "0" && ch <= "9") || ch == "_"
}

// ----- returns, yields and lambdas -----
//
// A `return default;` is typed by the enclosing function member, whose tag runs
// after its body's: the cells wait in csRetDflts with their offsets, and the member
// (csRetDrain) takes the ones inside it. A lambda takes its own (csLamDrain) into a
// record that waits for the delegate type its context converts it to, so a
// method's type never reaches a lambda's return.
var csRetDflts = []
var csLamRecs = {}
// The returned expression ending at `to`: a lambda (whose own returns its record
// already holds), else a default literal.
function csRetPush(to, closer) {
    var lam = csLamAt(to, closer)
    if (lam != null) {
        csRetDflts.push({at: to, cell: null, yld: false, lam: lam})
        return
    }
    var cell = csDfltAt(to, closer)
    if (cell != null) { csRetDflts.push({at: to, cell: cell, yld: false, lam: null}) }
}
function csRetDflt(to) { csRetPush(to, ";") }
function csYieldDflt(to) {
    var cell = csDfltAt(to, ";")
    if (cell != null) { csRetDflts.push({at: to, cell: cell, yld: true, lam: null}) }
}
// The member's declared type and whether it is async, from its tag items.
function csRetDrain(start, items) {
    var ty = ""
    var isAsync = false
    for (var i = 0; i < items.length; i++) {
        var it = items[i]
        if (it == null || typeof it != "object") { continue }
        if (ty == "" && it.ty != undefined) { ty = it.ty }
        if (ty == "" && it.cty != undefined) { ty = it.cty }
        if (it.mods != undefined) {
            for (var m = 0; m < it.mods.length; m++) { if (it.mods[m] == "async") { isAsync = true } }
        }
    }
    csRetDrainTo(start, ty, isAsync)
    return items
}
// A type member's end also ends the scope of its locals and parameters; a local
// function's does not end the enclosing member's.
function csRetDrainMember(start, items) {
    csRetDrain(start, items)
    csDfltEndMember()
    return items
}
function csDfltEndMember() { csLocalDfltTy = {} }
function csRetDrainTo(start, ty, isAsync) {
    var rty = isAsync ? csDfltAsyncTy(ty) : ty
    while (csRetDflts.length > 0 && csRetDflts[csRetDflts.length - 1].at >= start) {
        var r = csRetDflts.pop()
        if (r.lam != null) { csLamSet(r.lam, rty) }
        else { csDfltSet(r.cell, r.yld ? csDfltYieldTy(ty) : rty) }
    }
}
// An expression-bodied lambda's body is its return.
function csLamExprDflt(to) { csRetPush(to, "") }
// A lambda or anonymous method spanning [start, end): its pending returns become
// its record's, typed later by csLamSet.
function csLamDrain(start, end) {
    var rec = {cells: [], lams: [], set: false,
               isAsync: c.srcSlice(start, start + 5) == "async" && !csDfltIdCh(c.srcSlice(start + 5, start + 6))}
    while (csRetDflts.length > 0 && csRetDflts[csRetDflts.length - 1].at >= start) {
        var r = csRetDflts.pop()
        if (r.lam != null) { rec.lams.push(r.lam) } else {
            r.cell.lam = true
            rec.cells.push(r.cell)
        }
    }
    csLamRecs["$" + csDfltEnd(end, "")] = rec
}
function csLamAt(to, closer) {
    var k = "$" + csDfltEnd(to, closer)
    return csLamRecs[k] != undefined ? csLamRecs[k] : null
}
// The lambda converts to delegate type ty: its returns take the delegate's return
// type (the T of Task<T> for an async lambda).
function csLamSet(rec, ty) {
    if (rec == null || rec.set) { return }
    rec.set = true
    var rty = csDfltDelegRet(ty)
    if (rec.isAsync) { rty = csDfltAsyncTy(rty) }
    for (var i = 0; i < rec.cells.length; i++) {
        var cell = rec.cells[i]
        if (!cell.set) {
            cell.ty = stripWs(rty)
            cell.set = true
        }
    }
    for (var j = 0; j < rec.lams.length; j++) { csLamSet(rec.lams[j], rty) }
}
// A context of type ty for the expression ending at `to`: a default literal, or a
// lambda converting to the delegate type ty.
function csDfltTo(to, closer, ty) {
    csDfltTarget(to, closer, ty)
    csLamSet(csLamAt(to, closer), ty)
}
// `x = ...` where x is a local or a parameter of the member being read.
function csDfltLocalAssign(to, name) {
    var t = csLocalDfltTy["$" + name]
    if (t != undefined) { csDfltTo(to, "", t) }
}

// ----- literals and locals -----

// The type of a literal's source text, or "" when it is not a literal: what the
// other branch of `c ? 5 : default` gives the default.
function csLitTypeName(t) {
    if (t == "true" || t == "false") { return "bool" }
    if (t.length == 0) { return "" }
    var c0 = t.charAt(0)
    if (c0 == "'") { return "char" }
    if (c0 == "\"" || c0 == "$" || c0 == "@") { return "string" }
    var s = (c0 == "-" || c0 == "+") ? t.substring(1, t.length) : t
    if (s.length == 0) { return "" }
    var d0 = s.charAt(0)
    if (!(d0 >= "0" && d0 <= "9") && d0 != ".") { return "" }
    if (s.length > 1 && (s.charAt(1) == "x" || s.charAt(1) == "X" || s.charAt(1) == "b" || s.charAt(1) == "B")) { return "int" }
    var last = s.charAt(s.length - 1)
    if (last == "f" || last == "F") { return "float" }
    if (last == "m" || last == "M") { return "decimal" }
    if (last == "d" || last == "D" || s.indexOf(".") >= 0 || s.indexOf("e") >= 0 || s.indexOf("E") >= 0) { return "double" }
    return "int"
}
// Every local's and typed parameter's declared type as written (`int?` kept),
// keyed "$" + name. `var` records the initializer's type when it is a literal.
var csLocalDfltTy = {}
function csLocalDfltOf(name) {
    var t = csLocalDfltTy["$" + name]
    return t != undefined ? t : ""
}
// `var x = new T(...)` takes T.
function csDeclDfltTy(ty, info) {
    var t = stripWs(ty)
    if (t != "var") { return t }
    var raw = info.dsrc.trim()
    var src = raw.substring(1, raw.length).trim()
    if (src.length > 4 && src.substring(0, 3) == "new" && !csDfltIdCh(src.charAt(3))) {
        var depth = 0
        var i = 3
        while (i < src.length) {
            var ch = src.charAt(i)
            if (ch == "<") { depth++ }
            if (ch == ">") { depth-- }
            if (depth == 0 && (ch == "(" || ch == "{" || ch == "[")) { break }
            i++
        }
        var nt = stripWs(src.substring(3, i))
        return i < src.length && src.charAt(i) == "[" && nt != "" ? nt + "[]" : nt
    }
    return csLitTypeName(stripWs(src))
}
// A method's, local function's or constructor's typed parameters, as locals.
function csDfltParams(pinfo) {
    if (pinfo == undefined || pinfo == null || pinfo.ptypes == undefined) { return }
    for (var i = 0; i < pinfo.params.length; i++) {
        var t = pinfo.ptypes[i]
        if (t != undefined && t != "") { csLocalDfltTy["$" + pinfo.params[i]] = t }
    }
}
// The static type of an operand's source text as far as a parse can tell: a
// literal's, or a local's declared type.
function csOperandType(t) {
    var lt = csLitTypeName(t)
    if (lt != "") { return lt }
    return csLocalDfltOf(t)
}
// `c ? x : default`: the conditional's type is the other branch's (ECMA-334
// 12.18), so a default branch takes it when a parse can tell it. Otherwise the
// cell stays open for the context outside. Answers [cond, then, else].
function csCondDflt(items) {
    if (items.length == 1) { return items }
    var p1 = items[2]
    var p2 = items[4]
    var s1 = stripWs(c.srcSlice(p1.at, p1.e))
    var s2 = stripWs(c.srcSlice(p2.at, p2.e))
    var ty = ""
    if (s1 == "default") {
        ty = csOperandType(s2)
        if (ty != "") { csDfltSet(csDfltAt(p1.e, ""), ty) }
    } else if (s2 == "default") {
        ty = csOperandType(s1)
        if (ty != "") { csDfltSet(csDfltAt(p2.e, ""), ty) }
    }
    return [items[0], items[1], items[3]]
}
function csDfltNullableTy(ty) { return ty.length > 1 && ty.charAt(ty.length - 1) == "?" }

// ----- members: fields, properties, parameters -----

// Every field's and property's declared type, and every method's and
// constructor's parameter list, by name, each with the class declaring it
// (curClassName while the member is read).
var csDfltMembers = {}
var csDfltCallees = {}
function csDfltMember(name, ty) {
    var k = "$" + name
    if (csDfltMembers[k] == undefined) { csDfltMembers[k] = [] }
    csDfltMembers[k].push({cls: curClassName, ty: stripWs(ty), p: null})
}
function csDfltCallee(name, pinfo) {
    if (name == undefined || name == "" || pinfo == undefined || pinfo == null || pinfo.ptypes == undefined) { return }
    var k = "$" + name
    if (csDfltCallees[k] == undefined) { csDfltCallees[k] = [] }
    csDfltCallees[k].push({cls: curClassName, ty: "", p: pinfo})
}
// The declarations of a name that class cls sees: its own, else its nearest base
// class's; without a class (or with none declaring it), all of them.
function csDfltVisible(ds, cls) {
    var c = cls
    var guard = 0
    while (c != "" && guard < 64) {
        guard++
        var own = []
        for (var i = 0; i < ds.length; i++) { if (ds[i].cls == c) { own.push(ds[i]) } }
        if (own.length > 0) { return own }
        var s = csSupers[c]
        c = typeof s == "string" ? s : ""
    }
    return ds
}
// The one type the visible declarations of member `name` agree on, or "".
function csDfltMemberTy(cls, name) {
    var all = csDfltMembers["$" + name]
    if (all == undefined) { return "" }
    var ds = csDfltVisible(all, cls)
    for (var i = 1; i < ds.length; i++) { if (ds[i].ty != ds[0].ty) { return "" } }
    return ds[0].ty
}
// The one declared type argument position idx (or the parameter named pname) has
// in every visible overload of `name` that takes nargs arguments, or "".
function csDfltParamTy(cls, name, idx, pname, nargs) {
    var all = csDfltCallees["$" + name]
    if (all == undefined) { return "" }
    var cs = csDfltVisible(all, cls)
    var ty = ""
    var seen = false
    for (var i = 0; i < cs.length; i++) {
        var p = cs[i].p
        if (p.params.length < nargs && p.variadic < 0) { continue }
        var j = idx
        if (pname != "") {
            j = -1
            for (var k = 0; k < p.params.length; k++) { if (p.params[k] == pname) { j = k } }
        }
        if (j < 0 || j >= p.params.length || j == p.variadic) { return "" }
        var t = p.ptypes[j]
        if (t == undefined || t == "") { return "" }
        if (seen && t != ty) { return "" }
        ty = t
        seen = true
    }
    return ty
}

// The arguments of a call spanning [start, end): each one that is a default
// literal, a `new(...)` or a lambda waits for its parameter's type.
var csDfltLater = []
function csDfltArgs(cls, name, start, end) {
    var s = c.srcSlice(start, end)
    if (s.indexOf("default") < 0 && s.indexOf("=>") < 0 && s.indexOf("delegate") < 0 && s.indexOf("new") < 0) { return }
    var p = csDfltOpenParen(s)
    if (p < 0) { return }
    var ends = []
    var froms = []
    var depth = 0
    var from = p + 1
    var i = p + 1
    while (i < s.length) {
        var ch = s.charAt(i)
        if (ch == "\"" || ch == "'") {
            i = csDfltSkipQuoted(s, i)
            continue
        }
        if (ch == "(" || ch == "[" || ch == "{") { depth++ }
        if (ch == ")" || ch == "]" || ch == "}") {
            if (depth == 0) {
                froms.push(from)
                ends.push(i)
                break
            }
            depth--
        }
        if (ch == "," && depth == 0) {
            froms.push(from)
            ends.push(i)
            from = i + 1
        }
        i++
    }
    for (var a = 0; a < ends.length; a++) {
        var text = s.substring(froms[a], ends[a]).trim()
        var pname = csDfltArgName(text)
        var at = start + ends[a]
        var cell = csDfltAt(at, "")
        var lam = csLamAt(at, "")
        if (cell != null || lam != null) {
            csDfltLater.push({kind: "arg", cell: cell, lam: lam, cls: cls, name: name, idx: a, pname: pname, nargs: ends.length})
        }
    }
}
// The position of the argument list's "(": the first one outside type arguments.
function csDfltOpenParen(s) {
    var depth = 0
    for (var i = 0; i < s.length; i++) {
        var ch = s.charAt(i)
        if (ch == "<") { depth++ }
        if (ch == ">") { depth-- }
        if (ch == "(" && depth <= 0) { return i }
    }
    return -1
}
// The position after the string or char literal starting at i ('@' verbatim and
// '$' interpolated strings included).
function csDfltSkipQuoted(s, i) {
    var q = s.charAt(i)
    var verb = i > 0 && (s.charAt(i - 1) == "@" || (i > 1 && s.charAt(i - 2) == "@"))
    var j = i + 1
    while (j < s.length) {
        var ch = s.charAt(j)
        if (ch == "\\" && !verb) {
            j += 2
            continue
        }
        if (ch == q) {
            if (verb && j + 1 < s.length && s.charAt(j + 1) == q) {
                j += 2
                continue
            }
            return j + 1
        }
        j++
    }
    return j
}
// `name: value`: the parameter name of a named argument, else "".
function csDfltArgName(t) {
    var i = 0
    while (i < t.length && csDfltIdCh(t.charAt(i))) { i++ }
    if (i == 0) { return "" }
    var j = i
    while (j < t.length && t.charAt(j).trim() == "") { j++ }
    if (j < t.length && t.charAt(j) == ":" && (j + 1 >= t.length || t.charAt(j + 1) != ":")) { return t.substring(0, i) }
    return ""
}
// An assignment spanning [start, end) whose right side is a default literal or a
// lambda: a field or property target (`F`, `this.F`, `o.F`, `C.F`) waits for the
// member's type; an element of a local array or List takes the element type, an
// entry of a local Dictionary the value type.
function csDfltAssign(start, end) {
    var cell = csDfltAt(end, "")
    var lam = csLamAt(end, "")
    if (cell == null && lam == null) { return }
    var s = c.srcSlice(start, end)
    var eq = csDfltAssignOp(s)
    if (eq < 0) { return }
    var tgt = stripWs(s.substring(0, eq))
    var br = tgt.indexOf("[")
    if (br > 0 && tgt.charAt(tgt.length - 1) == "]") {
        var lt = csLocalDfltOf(tgt.substring(0, br))
        var ety = ""
        if (lt.length > 2 && lt.substring(lt.length - 2, lt.length) == "[]") { ety = lt.substring(0, lt.length - 2) }
        else if (csDfltTName(lt) == "List" && csDfltTArgs(lt).length == 1) { ety = csDfltTArgs(lt)[0] }
        else if (csDfltTName(lt) == "Dictionary" && csDfltTArgs(lt).length == 2) { ety = csDfltTArgs(lt)[1] }
        if (ety != "") {
            csDfltSet(cell, ety)
            csLamSet(lam, ety)
        }
        return
    }
    for (var i = 0; i < tgt.length; i++) {
        if (!csDfltIdCh(tgt.charAt(i)) && tgt.charAt(i) != ".") { return }
    }
    var dot = csDfltLastDot(tgt)
    if (dot < 0 && csLocalDfltTy["$" + tgt] != undefined) { return }
    csDfltLater.push({kind: "member", cell: cell, lam: lam, cls: csDfltRecvCls(tgt, dot),
                      name: tgt.substring(dot + 1, tgt.length)})
}
// The class a member access `tgt` (its last dot at `dot`) reads from: the enclosing
// class for `F` and `this.F`, a local's declared class for `o.F`, the class itself
// for `C.F`; "" for a longer path.
function csDfltRecvCls(tgt, dot) {
    if (dot < 0) { return curClassName }
    var head = tgt.substring(0, dot)
    if (head == "this" || head == "base") { return curClassName }
    if (csDfltLastDot(head) >= 0) { return "" }
    var lt = csLocalDfltTy["$" + head]
    if (lt != undefined) { return csDfltTName(lt) }
    return head
}
// The offset of a plain `=` assignment operator in s, or -1 for a compound one.
function csDfltAssignOp(s) {
    var depth = 0
    for (var i = 0; i < s.length; i++) {
        var ch = s.charAt(i)
        if (ch == "(" || ch == "[" || ch == "{") { depth++ }
        if (ch == ")" || ch == "]" || ch == "}") { depth-- }
        if (ch == "=" && depth == 0) {
            var prev = i > 0 ? s.charAt(i - 1) : ""
            var next = i + 1 < s.length ? s.charAt(i + 1) : ""
            if (next == "=" || next == ">") { return -1 }
            if ("+-*/%&|^<>!?=".indexOf(prev) >= 0 && prev != "") { return -1 }
            return i
        }
    }
    return -1
}
// The nullable test for an `==` operand: 1 a nullable local or parameter, 0 a local
// of another type or an operand that names no member, and from 2 on the index (+2)
// of a field or property access (`F`, `this.F`, `o.F`) that csDfltNullableMember
// decides once the file is read.
var csDfltNullMembers = []
function csDfltNullKind(t) {
    var lt = csLocalDfltTy["$" + t]
    if (lt != undefined) { return csDfltNullableTy(lt) ? 1 : 0 }
    for (var i = 0; i < t.length; i++) { if (!csDfltIdCh(t.charAt(i)) && t.charAt(i) != ".") { return 0 } }
    var dot = csDfltLastDot(t)
    csDfltNullMembers.push({cls: csDfltRecvCls(t, dot), name: t.substring(dot + 1, t.length)})
    return csDfltNullMembers.length + 1
}
function csDfltNullableMember(k) {
    var m = csDfltNullMembers[k - 2]
    return csDfltNullableTy(csDfltMemberTy(m.cls, m.name))
}
// The contexts waiting for the members the whole file declares.
function csDfltResolve() {
    for (var i = 0; i < csDfltLater.length; i++) {
        var r = csDfltLater[i]
        var ty = r.kind == "arg" ? csDfltParamTy(r.cls, r.name, r.idx, r.pname, r.nargs) : csDfltMemberTy(r.cls, r.name)
        if (ty == "") { continue }
        csDfltSet(r.cell, ty)
        csLamSet(r.lam, ty)
    }
    csDfltLater = []
}
// Whether the program names a List's capacity members, which is when the
// runtime keeps capacities (csxCapOn, lib/csharp-collections.js).
var csCapUsed = false
function csCapNote(name) {
    if (name == "Capacity" || name == "TrimExcess" || name == "EnsureCapacity") { csCapUsed = true }
    return name
}
// A call's tag: its arguments wait for the parameter types (csDfltArgs). An
// unqualified call names a member of the enclosing class; a member call's
// receiver class is not known to the parse.
function csDfltSelfCall(name, start, end) {
    csDfltArgs(curClassName, name, start, end)
    return name
}
function csDfltCall(name, start, end) {
    csDfltArgs("", csCapNote(name), start, end)
    return name
}
function csDfltNew(items, start, end) {
    if (items.length > 0 && items[0] != null && typeof items[0] == "object" && items[0].cname != undefined) {
        csDfltArgs(items[0].cname, items[0].cname, start, end)
    }
    return items
}
