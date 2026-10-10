// java-decltypes.js -- Java's declared types, threaded at WALK time, shared by
// java-interpreter.abnf and java-to-llvm-ir.abnf through include(). Each grammar
// supplies the conversion itself (jConvTo / declConvert) and jArrTyped, which
// wraps a thunk of its own shape.
//
// A parameter, a returned value, a plain `x = v`, a field or array element
// store and an array initializer's elements all convert to the DECLARED type of
// their target (JLS 5.2) - `void q(double d)` sees q(2) as 2.0, `long t; t = 2`
// holds a long, `float[] a = {16777217}` holds 1.6777216E7. A value carries its
// type, a slot does not, and guessing a slot's type from the value it holds is
// wrong for an Object slot - so the declared types are recorded here as the
// tags run, which is in source order and before any program code runs.
//
// A local's type: one name -> type map per method body, opened by the ParamList
// tag (which the walk reaches before the body) and closed by the method's own
// tag. Declarations later in the walk overwrite earlier ones, which is Java's
// scoping for every name a valid program can still assign.
var jTyScopes = [{}]
function jTyPush() { jTyScopes.push({}) }
function jTyPop() { if (jTyScopes.length > 1) { jTyScopes.pop() } }
function jTyDecl(name, ty) { jTyScopes[jTyScopes.length - 1]["$" + name] = ty }
function jTyOf(name) { return jTyScopes[jTyScopes.length - 1]["$" + name] }
// A field's type, by NAME across the whole program: "?" once two
// declarations of the name disagree, and then no conversion is made. The
// grammars read it once the walk has seen every field.
var jFieldTys = {}
function jNoteFieldTy(name, ty) {
    var have = jFieldTys["$" + name]
    jFieldTys["$" + name] = (have == undefined || have == ty) ? ty : "?"
}
function jFieldTyOf(name) {
    var t = jFieldTys["$" + name]
    if (t == undefined || t == "?") { return "" }
    return t
}
// The element type of an array type, "" for anything else.
function jElemTy(ty) {
    if (ty.length > 2 && ty.substring(ty.length - 2) == "[]") { return ty.substring(0, ty.length - 2) }
    return ""
}
// The declared result type of the method whose body is being walked: the
// RetType tag pushes it before the body, a lambda's head pushes "" (its
// returns are its own), and the closing tags pop.
var jRetTys = [""]
function jRetPush(ty) { jRetTys.push(ty) }
function jRetPop() { if (jRetTys.length > 1) { jRetTys.pop() } }
function jRetTy() { return jRetTys[jRetTys.length - 1] }
// A method's result type, by NAME across the whole program, "?" once two
// declarations disagree - the jFieldTys rule. Registered as the walk passes
// the method's name, so a recursive call already sees it.
var jMethodTys = {}
function jNoteMethodTy(name) {
    var have = jMethodTys["$" + name]
    jMethodTys["$" + name] = (have == undefined || have == jRetTy()) ? jRetTy() : "?"
}
function jMethodTyOf(name) {
    var t = jMethodTys["$" + name]
    if (t == undefined || t == "?") { return "" }
    return t
}
// The type a value written as `text` converts to on its way into a slot of
// declared type ty, or "" when no conversion is needed: ty is not a
// primitive, or the value's static type already is ty.
function jConvTy(ty, text) {
    if (!jIsPrimTy(ty)) { return "" }
    if (jExprTy(text) == ty) { return "" }
    return ty
}
// The static type of an arithmetic expression's (whitespace-free) text: names,
// calls, `.f` and `[i]` typed as above, numeric and char literals, + - * / %
// & | ^, a unary sign and parentheses, promoted as JLS 5.6 does; "" for any
// other shape (a cast, a comparison, a ternary, ...). It only ever lets a store
// SKIP a conversion its value does not need, which is what keeps
// `s = s + i % 7` on a long free of a runtime call.
function jExprTy(t) {
    var ty = ""
    var operands = 0
    var ops = 0
    var i = 0
    var wantOperand = true
    while (i < t.length) {
        var c = t.charAt(i)
        if (wantOperand) {
            if (c == "(") {
                i++
            } else if (c == "-" || c == "+") {
                if (t.charAt(i + 1) == c) { return "" }
                ops++
                i++
            } else {
                var ot = ""
                var e = jIdentEnd(t, i)
                if (e > i) {
                    var nm = t.substring(i, e)
                    i = e
                    if (t.charAt(i) == "(") {
                        ot = jMethodTyOf(nm)
                        i = jSkipGroup(t, i)
                    } else {
                        var lt = jTyOf(nm)
                        ot = (lt != undefined) ? lt : jFieldTyOf(nm)
                    }
                    // `.f`, `.m(...)` and `[i]` suffixes, each typed by name.
                    var sfx = true
                    while (sfx && i < t.length) {
                        var c2 = t.charAt(i)
                        if (c2 == ".") {
                            var e2 = jIdentEnd(t, i + 1)
                            if (e2 == i + 1) { return "" }
                            var nm2 = t.substring(i + 1, e2)
                            i = e2
                            if (t.charAt(i) == "(") {
                                ot = jMethodTyOf(nm2)
                                i = jSkipGroup(t, i)
                            } else if (nm2 == "length" && jElemTy(ot) != "") {
                                ot = "int"
                            } else {
                                ot = jFieldTyOf(nm2)
                            }
                        } else if (c2 == "[") {
                            ot = jElemTy(ot)
                            i = jSkipGroup(t, i)
                        } else {
                            sfx = false
                        }
                    }
                } else if (c == "'") {
                    var q = i + 1
                    if (t.charAt(q) == "\\") { q++ }
                    while (q < t.length && t.charAt(q) != "'") { q++ }
                    ot = "char"
                    i = q + 1
                } else {
                    var lit = jNumLit(t, i)
                    if (lit.end == i) { return "" }
                    ot = lit.ty
                    i = lit.end
                }
                if (!jIsPrimTy(ot)) { return "" }
                ty = (operands == 0) ? ot : jPromote(ty, ot)
                operands++
                wantOperand = false
            }
        } else if (c == ")") {
            i++
        } else if ("+-*/%&|^".indexOf(c) >= 0) {
            var n2 = t.charAt(i + 1)
            if (n2 == c || n2 == "=") { return "" }
            ops++
            i++
            wantOperand = true
        } else {
            return ""
        }
    }
    if (wantOperand || operands == 0) { return "" }
    if (ops > 0) { return jPromote(ty, "int") }
    return ty
}
// A primitive numeric type (char included): the declared types a stored value
// converts to, and the operand types jExprTy promotes.
function jIsPrimTy(ty) {
    return ty == "int" || ty == "long" || ty == "double" || ty == "float"
        || ty == "short" || ty == "byte" || ty == "char"
}
// Binary numeric promotion of two operand types (JLS 5.6.2).
function jPromote(a, b) {
    if (a == "double" || b == "double") { return "double" }
    if (a == "float" || b == "float") { return "float" }
    if (a == "long" || b == "long") { return "long" }
    return "int"
}
// A numeric literal at i: its end and its type - L is long, f float, d or
// a decimal point or exponent double, anything else int. end == i when
// there is no literal there.
function jNumLit(t, i) {
    var j = i
    var hex = t.charAt(i) == "0" && (t.charAt(i + 1) == "x" || t.charAt(i + 1) == "X")
    var bin = t.charAt(i) == "0" && (t.charAt(i + 1) == "b" || t.charAt(i + 1) == "B")
    if (hex || bin) { j = i + 2 }
    var flt = false
    var more = true
    while (j < t.length && more) {
        var c = t.charAt(j)
        var code = t.charCodeAt(j)
        if ((code >= 48 && code <= 57) || c == "_") { j++ }
        else if (hex && ((code >= 97 && code <= 102) || (code >= 65 && code <= 70))) { j++ }
        else if (!hex && !bin && c == ".") { flt = true; j++ }
        else if (!hex && !bin && (c == "e" || c == "E")) {
            flt = true
            j++
            if (t.charAt(j) == "+" || t.charAt(j) == "-") { j++ }
        } else { more = false }
    }
    if (j == i) { return {end: i, ty: ""} }
    var s = t.charAt(j)
    if (s == "L" || s == "l") { return {end: j + 1, ty: "long"} }
    if (!hex && (s == "f" || s == "F")) { return {end: j + 1, ty: "float"} }
    if (!hex && (s == "d" || s == "D")) { return {end: j + 1, ty: "double"} }
    if (flt) { return {end: j, ty: "double"} }
    return {end: j, ty: "int"}
}
// The declared type of an assignment's place: the base name's local type
// (lty, from the walk) or field type, then one field type per `.f` and one
// array level per `[i]`.
function jPlaceTy(ref, lty) {
    var ty = (lty != undefined) ? lty : jFieldTyOf(ref.name)
    for (var i = 0; i < ref.path.length; i++) {
        if (ref.path[i].key != undefined) { ty = jFieldTyOf(ref.path[i].key) } else { ty = jElemTy(ty) }
    }
    return ty
}
// An arbitrary base with a `.f` place: the field's type by name.
function jFieldTyOfPlace(ref) {
    var last = ref.path[ref.path.length - 1]
    if (last.key != undefined) { return jFieldTyOf(last.key) }
    return ""
}
// The place `a.b[i][j]` of an assignment whose target the grammar read as
// a chain, as the name-and-path record jPlaceTy reads - recovered from the
// target's (whitespace-free) text, since the chain's own items are thunks.
// null when the base is not a plain name: a call, a cast, a parenthesis.
function jChainTyRef(text) {
    var segs = []
    var i = jIdentEnd(text, 0)
    if (i == 0) { return null }
    var name = text.substring(0, i)
    while (i < text.length) {
        var c = text.charAt(i)
        if (c == ".") {
            var e = jIdentEnd(text, i + 1)
            if (e == i + 1) { return null }
            segs.push({key: text.substring(i + 1, e)})
            i = e
        } else if (c == "[") {
            i = jSkipGroup(text, i)
            segs.push({idx: true})
        } else { return null }
    }
    return {name: name, path: segs}
}
// Every by-name call site, as the static types of its arguments (jExprTy,
// "?" once two sites disagree), keyed by method name and argument count.
// A parameter that every call site already passes its declared type needs
// no conversion where it is bound - which keeps `fib(n - 1)` free of a
// runtime call. A name a method reference takes (`C::m`), or that a call
// passes something jExprTy cannot split safely (a `<`), can be reached with
// anything, so it never skips.
var jCallTys = {}
var jCallEsc = {}
function jNoteCall(text) {
    var p = text.indexOf("(")
    if (p < 0) { return }
    var s = p
    while (s > 0 && jIsIdChar(text.charCodeAt(s - 1))) { s-- }
    var name = text.substring(s, p)
    var inner = text.substring(p + 1, jSkipGroup(text, p) - 1)
    if (inner.indexOf("<") >= 0) {
        jCallEsc["$" + name] = true
        return
    }
    var args = []
    var i = 0
    var from = 0
    while (i < inner.length) {
        var c = inner.charAt(i)
        if (c == "(" || c == "[" || c == "{") {
            i = jSkipGroup(inner, i)
        } else if (c == "\"" || c == "'") {
            i = jSkipQuote(inner, i)
        } else if (c == ",") {
            args.push(inner.substring(from, i))
            i++
            from = i
        } else {
            i++
        }
    }
    if (inner.length > 0) { args.push(inner.substring(from)) }
    var key = "$" + name + "/" + args.length
    var have = jCallTys[key]
    var tys = []
    for (var k = 0; k < args.length; k++) {
        var t = jExprTy(args[k])
        if (have != undefined && have[k] != t) { t = "?" }
        tys.push(t)
    }
    jCallTys[key] = tys
}
function jIsIdChar(c) {
    return c == 95 || c == 36 || (c >= 65 && c <= 90) || (c >= 97 && c <= 122) || (c >= 48 && c <= 57)
}
// The index just past the string or char literal that opens at i.
function jSkipQuote(text, i) {
    var q = text.charAt(i)
    var j = i + 1
    while (j < text.length && text.charAt(j) != q) {
        if (text.charAt(j) == "\\") { j++ }
        j++
    }
    return j + 1
}
function jNoteMRef(sfx) {
    jCallEsc["$" + sfx.name] = true
    return sfx
}
function jNoteRefText(text) {
    var c = text.indexOf("::")
    if (c >= 0) { jCallEsc["$" + text.substring(c + 2)] = true }
    return text
}
// The parameter types a method converts its arguments to: its declared
// ptys, with "" where every call site already passes that type. A varargs
// method converts everything.
function jBoundTys(name, ptys, isVar) {
    var out = []
    var sites = jCallTys["$" + name + "/" + ptys.length]
    var trust = !isVar && jCallEsc["$" + name] != true && sites != undefined
    for (var k = 0; k < ptys.length; k++) {
        out.push((trust && sites[k] == ptys[k]) ? "" : ptys[k])
    }
    return out
}
// The index just past the ( [ or { group that opens at i, string and char
// literals inside it skipped.
function jSkipGroup(text, i) {
    var depth = 0
    var j = i
    while (j < text.length) {
        var d = text.charAt(j)
        if (d == "\"" || d == "'") {
            j = jSkipQuote(text, j)
        } else {
            if (d == "[" || d == "(" || d == "{") { depth++ }
            if (d == "]" || d == ")" || d == "}") {
                depth--
                if (depth == 0) { return j + 1 }
            }
            j++
        }
    }
    return j
}
// The end of the identifier starting at i (i itself when there is none).
function jIdentEnd(text, i) {
    if (i >= text.length) { return i }
    var c0 = text.charCodeAt(i)
    if (c0 >= 48 && c0 <= 57) { return i }
    var j = i
    while (j < text.length && jIsIdChar(text.charCodeAt(j))) { j++ }
    return j
}
// The C-style declarator brackets, as the "[]" run they add to the type.
function jDimsOf(text) {
    var out = ""
    for (var i = 0; i < text.length; i++) { if (text.charAt(i) == "[") { out = out + "[]" } }
    return out
}
// The type an array initializer `{...}` builds, and so converts its
// elements to: set (jArrTyped) by the declaration, field or `new T[]` that owns
// it while that initializer runs, and by an outer initializer for a nested one.
var jArrCtx = ""
