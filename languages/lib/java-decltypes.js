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
// declared type ty, or "" when no conversion is needed: ty is neither a
// primitive nor a functional interface with one (jSamSig), or the value's
// static type already is ty.
function jConvTy(ty, text) {
    if (!jIsConvTy(ty)) { return "" }
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
// What an enhanced for converts each element to: its variable's declared type
// fety, unless the iterated array (`xs`, `this.xs`, `a.b`) is declared with
// that element type already.
function jIterConvTy(fety, text) {
    if (!jIsPrimTy(fety)) { return "" }
    var r = jChainTyRef(text)
    if (r != null && jElemTy(jPlaceTy(r, jTyOf(r.name))) == fety) { return "" }
    return fety
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

// ----- Overload resolution (JLS 15.12.2) -----
//
// Every class and record declaration registers its superclass and its methods'
// parameter types as the walk passes it. A method name whose candidates - the
// class' own declarations plus the inherited ones it does not override - have
// more than one signature gets a DISPATCHER in that class' slot. It picks a
// candidate from the arguments' runtime types (jOvPick here; jvOvPick in
// abnf/jsrtjava.go and languages/lib/java-rt.metajs for the compiler) and calls
// the implementation stored under "name#sig", looked up from the dispatcher's
// own class upwards - so super.m(...) stays non-virtual. The value model
// carries every primitive's type, so primitive arguments resolve as javac
// resolves them; a reference resolves by its runtime class, which differs from
// javac only where the static type is wider than the value.
var jOvCls = {}
var jOvNames = []
var jOvIfaces = {}
// A declared type with its type arguments dropped and its qualifier cut to the
// last segment: "java.util.List<String>[]" -> "List[]".
function jOvBare(ty) {
    var out = ""
    var depth = 0
    for (var i = 0; i < ty.length; i++) {
        var c = ty.charAt(i)
        if (c == "<") { depth++ } else if (c == ">") { depth-- } else if (depth == 0) {
            out = (c == ".") ? "" : out + c
        }
    }
    return out
}
// A parameter list's signature key: the bare types, "..." on a varargs one.
function jOvRawOf(ptys, isVar) {
    var parts = []
    for (var i = 0; i < ptys.length; i++) {
        parts.push(jOvBare(ptys[i]) + ((isVar && i == ptys.length - 1) ? "..." : ""))
    }
    return parts.join(",")
}
// A varargs declaration: the interpreter flags it, the compiler's last
// parameter is a {rest} record instead of a name.
function jOvIsVar(m) {
    if (m.vararg === true) { return true }
    return m.params.length > 0 && typeof m.params[m.params.length - 1] != "string"
}
function jOvDeclare(name, members) {
    var sup = ""
    var ms = []
    for (var i = 0; i < members.length; i++) {
        var m = members[i]
        if (m.ext != undefined) { sup = jOvBare(m.ext) }
        if (m.kind == "method") {
            var v = jOvIsVar(m)
            ms.push({name: m.name, raw: jOvRawOf(m.ptys, v), ptys: m.ptys, isVar: v, statik: m.statik})
        }
    }
    if (jOvCls["$" + name] == undefined) { jOvNames.push(name) }
    jOvCls["$" + name] = {sup: sup, ms: ms}
}
function jOvIface(name, members) {
    jOvIfaces["$" + name] = true
    var abs = []
    for (var i = 0; i < members.length; i++) {
        if (members[i].kind == "amethod" || members[i].kind == "abstract") { abs.push(members[i]) }
    }
    if (abs.length == 1) { jSamUser["$" + name] = abs[0].ptys.join(",") + ">" + abs[0].ret }
}

// ----- Functional interfaces with primitive positions -----
//
// A lambda stored in a slot of a functional interface type takes that
// interface's parameter and result types: `DoubleUnaryOperator op = x -> x+1`
// sees op.applyAsDouble(2) as x = 2.0 and answers 3.0, `IntToDoubleFunction
// h = i -> i / 2` answers h.applyAsDouble(3) as 1.0. jSamSig is the signature
// as "params>result", one entry per parameter, "" where nothing converts; null
// when no position is a primitive. A declared interface with ONE abstract
// method is read from its declaration (jOvIface), the java.util.function
// primitive specializations from this table.
var jSamUser = {}
var jSamLib = {
    "$DoubleUnaryOperator": "double>double", "$DoubleBinaryOperator": "double,double>double",
    "$IntUnaryOperator": "int>int", "$IntBinaryOperator": "int,int>int",
    "$LongUnaryOperator": "long>long", "$LongBinaryOperator": "long,long>long",
    "$DoublePredicate": "double>", "$IntPredicate": "int>", "$LongPredicate": "long>",
    "$DoubleFunction": "double>", "$IntFunction": "int>", "$LongFunction": "long>",
    "$DoubleConsumer": "double>", "$IntConsumer": "int>", "$LongConsumer": "long>",
    "$DoubleSupplier": ">double", "$IntSupplier": ">int", "$LongSupplier": ">long",
    "$ToDoubleFunction": ">double", "$ToIntFunction": ">int", "$ToLongFunction": ">long",
    "$ToDoubleBiFunction": ">double", "$ToIntBiFunction": ">int", "$ToLongBiFunction": ">long",
    "$IntToDoubleFunction": "int>double", "$IntToLongFunction": "int>long",
    "$LongToDoubleFunction": "long>double", "$LongToIntFunction": "long>int",
    "$DoubleToIntFunction": "double>int", "$DoubleToLongFunction": "double>long",
    "$ObjDoubleConsumer": ",double>", "$ObjIntConsumer": ",int>", "$ObjLongConsumer": ",long>"
}
function jSamSig(ty) {
    if (ty.length > 2 && ty.substring(ty.length - 2) == "[]") { return null }
    var b = jOvBare(ty)
    var sig = jSamUser["$" + b]
    if (sig == undefined) {
        if (jOvCls["$" + b] != undefined || jOvIfaces["$" + b] == true) { return null }
        sig = jSamLib["$" + b]
        if (sig == undefined) { return null }
    }
    var parts = sig.split(">")
    var ps = parts[0] == "" ? [] : parts[0].split(",")
    var prim = jIsPrimTy(parts[1])
    for (var i = 0; i < ps.length; i++) {
        if (!jIsPrimTy(ps[i])) { ps[i] = "" } else { prim = true }
    }
    if (!prim) { return null }
    return {ps: ps, ret: jIsPrimTy(parts[1]) ? parts[1] : ""}
}
// A declared type a stored value converts to: a primitive or a functional
// interface with a primitive position.
function jIsConvTy(ty) { return jIsPrimTy(ty) || jSamSig(ty) != null }
// The candidates for `mname` in class cls: its own declarations, then each
// superclass' that no nearer declaration overrides. null unless that is two or
// more signatures of one kind (all static or all instance).
function jOvCands(cls, mname) {
    var out = []
    var seen = {}
    var c = cls
    var guard = 0
    while (c != "" && guard < 64) {
        guard++
        var rec = jOvCls["$" + c]
        if (rec == undefined) { break }
        for (var i = 0; i < rec.ms.length; i++) {
            var m = rec.ms[i]
            if (m.name != mname || seen["$" + m.raw] == true) { continue }
            seen["$" + m.raw] = true
            out.push(m)
        }
        c = rec.sup
    }
    if (out.length < 2) { return null }
    for (var k = 1; k < out.length; k++) { if (out[k].statik != out[0].statik) { return null } }
    return out
}
// Does any class dispatch over `mname`? Then every declaration of it is stored
// under its "name#sig" slot as well. Asked only once the walk is complete.
var jOvSlotNames = null
function jOvNeedsSlot(mname) {
    if (jOvSlotNames == null) {
        jOvSlotNames = {}
        for (var c = 0; c < jOvNames.length; c++) {
            var ms = jOvCls["$" + jOvNames[c]].ms
            for (var i = 0; i < ms.length; i++) {
                if (jOvCands(jOvNames[c], ms[i].name) != null) { jOvSlotNames["$" + ms[i].name] = true }
            }
        }
    }
    return jOvSlotNames["$" + mname] == true
}
var jOvPrims = ",byte,short,char,int,long,float,double,boolean,"
var jOvCore = ",String,Object,Integer,Long,Double,Float,Short,Byte,Character,Boolean,Number,"
function jOvIsPrim(t) { return jOvPrims.indexOf("," + t + ",") >= 0 }
// The token a parameter type is matched with: a primitive or a java.lang name
// as itself, an array as its element's token plus "[]", a declared class by
// name, and everything else (an interface, a type variable, a library type) as
// "*", which takes any reference and any lambda.
function jOvTok(raw) {
    if (raw.length > 2 && raw.substring(raw.length - 2) == "[]") { return jOvTok(raw.substring(0, raw.length - 2)) + "[]" }
    if (jOvIsPrim(raw) || jOvCore.indexOf("," + raw + ",") >= 0) { return raw }
    if (jOvCls["$" + raw] != undefined && jOvIfaces["$" + raw] != true) { return raw }
    return "*"
}
// Widening primitive conversion (JLS 5.1.2), identity included.
function jOvWiden(k, t) {
    if (k == t) { return true }
    if (k == "boolean" || t == "boolean" || t == "char" || t == "byte") { return false }
    if (k == "char") { return t != "short" }
    var order = ",byte,short,int,long,float,double,"
    return order.indexOf("," + t + ",") > order.indexOf("," + k + ",")
}
// Is token s at least as specific as t (JLS 15.12.2.5: s <: t)?
function jOvSub(s, t) {
    if (s == t) { return true }
    if (jOvIsArr(s) && jOvIsArr(t)) {
        var es = s.substring(0, s.length - 2)
        var et = t.substring(0, t.length - 2)
        return !jOvIsPrim(es) && !jOvIsPrim(et) && jOvSub(es, et)
    }
    var sp = jOvIsPrim(s)
    var tp = jOvIsPrim(t)
    if (sp && tp) { return jOvWiden(s, t) }
    if (sp || tp) { return false }
    if (t == "*" || t == "Object") { return true }
    if (t == "Number") { return ",Integer,Long,Double,Float,Short,Byte,".indexOf("," + s + ",") >= 0 }
    var c = s
    var guard = 0
    while (guard < 64) {
        guard++
        var rec = jOvCls["$" + c]
        if (rec == undefined || rec.sup == "") { return false }
        c = rec.sup
        if (c == t) { return true }
    }
    return false
}
function jOvIsArr(t) { return t.length > 2 && t.substring(t.length - 2) == "[]" }
function jOvMore(a, b) {
    if (a.length != b.length) { return false }
    for (var i = 0; i < a.length; i++) { if (!jOvSub(a[i], b[i])) { return false } }
    return true
}
// A dispatcher's specification: one "tokens@sig" entry per candidate, "|"
// between them, MOST SPECIFIC FIRST, so the first applicable candidate of a
// phase is the one JLS 15.12.2.5 chooses.
function jOvSpec(cands) {
    var rest = []
    for (var i = 0; i < cands.length; i++) {
        var toks = []
        for (var k = 0; k < cands[i].ptys.length; k++) {
            var tk = jOvTok(jOvBare(cands[i].ptys[k]))
            if (cands[i].isVar && k == cands[i].ptys.length - 1) { tk = tk + "..." }
            toks.push(tk)
        }
        rest.push({toks: toks, raw: cands[i].raw})
    }
    var parts = []
    while (rest.length > 0) {
        var pick = 0
        var found = false
        for (var r = 0; r < rest.length && !found; r++) {
            var beaten = false
            for (var q = 0; q < rest.length; q++) {
                if (q != r && jOvMore(rest[q].toks, rest[r].toks) && !jOvMore(rest[r].toks, rest[q].toks)) { beaten = true }
            }
            if (!beaten) { pick = r; found = true }
        }
        parts.push(rest[pick].toks.join(",") + "@" + rest[pick].raw)
        var left = []
        for (var l = 0; l < rest.length; l++) { if (l != pick) { left.push(rest[l]) } }
        rest = left
    }
    return parts.join("|")
}
// One arity's constructors, as jOvSpec's records. A record's canonical
// constructor carries no declared types; its components match anything.
function jOvCtorCands(ctors) {
    var out = []
    for (var i = 0; i < ctors.length; i++) {
        var v = jOvIsVar(ctors[i])
        var pt = ctors[i].ptys
        if (pt == undefined) {
            pt = []
            for (var k = 0; k < ctors[i].params.length; k++) { pt.push("Object") }
        }
        out.push({raw: jOvRawOf(pt, v), ptys: pt, isVar: v})
    }
    return out
}
// The interpreter's pick over a jOvSpec string. kindOf(v) names a value's
// runtime type: "null", a primitive, "String", "[]", "fn" (a lambda) or "obj";
// isA(v, cls) tests a class token. Phase 1 matches without boxing, phase 2
// with it, phase 3 expands a varargs parameter (JLS 15.12.2.2-4). Answers the
// chosen candidate's sig; with none applicable, the first of the right arity.
function jOvPick(spec, args, off, kindOf, isA) {
    var ents = spec.split("|")
    var n = args.length - off
    for (var phase = 1; phase <= 3; phase++) {
        for (var e = 0; e < ents.length; e++) {
            var at = ents[e].indexOf("@")
            var toks = at == 0 ? [] : ents[e].substring(0, at).split(",")
            var p = toks.length
            var isVar = p > 0 && toks[p - 1].indexOf("...") > 0
            var ok = true
            if (phase < 3) {
                if (p != n) { continue }
                for (var i = 0; i < p && ok; i++) {
                    var t = (isVar && i == p - 1) ? toks[i].substring(0, toks[i].length - 3) + "[]" : toks[i]
                    ok = jOvAccepts(t, args[off + i], phase, kindOf, isA)
                }
            } else {
                if (!isVar || n < p - 1) { continue }
                for (var j = 0; j < n && ok; j++) {
                    var tv = j < p - 1 ? toks[j] : toks[p - 1].substring(0, toks[p - 1].length - 3)
                    ok = jOvAccepts(tv, args[off + j], 2, kindOf, isA)
                }
            }
            if (ok) { return ents[e].substring(at + 1) }
        }
    }
    for (var f = 0; f < ents.length; f++) {
        var af = ents[f].indexOf("@")
        var nf = af == 0 ? 0 : ents[f].substring(0, af).split(",").length
        if (nf == n) { return ents[f].substring(af + 1) }
    }
    return ents[0].substring(ents[0].indexOf("@") + 1)
}
// Can the argument v be passed to a parameter of token t in this phase? An
// array is judged by its first element, the only type evidence it carries:
// a primitive element type must match exactly (int[] is no long[]). Twin of
// jvOvAccepts in abnf/jsrtjava.go and languages/lib/java-rt.metajs.
function jOvAccepts(t, v, phase, kindOf, isA) {
    var k = kindOf(v)
    if (jOvIsArr(t)) {
        if (k == "null") { return true }
        if (k != "[]") { return false }
        if (v.length == 0) { return true }
        var et = t.substring(0, t.length - 2)
        if (jOvIsPrim(et)) { return kindOf(v[0]) == et }
        return jOvAccepts(et, v[0], 1, kindOf, isA)
    }
    if (jOvIsPrim(t)) { return jOvIsPrim(k) && jOvWiden(k, t) }
    if (k == "null") { return true }
    if (jOvIsPrim(k)) {
        if (phase < 2) { return false }
        if (t == "*" || t == "Object") { return true }
        if (t == "Number") { return k != "char" && k != "boolean" }
        return t == jOvBox(k)
    }
    if (t == "*") { return true }
    if (k == "fn") { return false }
    if (t == "Object") { return true }
    if (k == "String") { return t == k }
    return k == "obj" && isA(v, t)
}
function jOvBox(k) {
    if (k == "int") { return "Integer" }
    if (k == "char") { return "Character" }
    if (k == "long") { return "Long" }
    if (k == "double") { return "Double" }
    if (k == "float") { return "Float" }
    if (k == "short") { return "Short" }
    if (k == "byte") { return "Byte" }
    return "Boolean"
}
