// kotlin-decltypes.js -- the static types an integer literal adopts, shared by
// kotlin-interpreter.abnf and kotlin-to-llvm-ir.abnf through include(). Each
// grammar wraps its own thunk shape in the retyping (kAdoptThunk / ktAdoptThunk).
//
// Kotlin types an integer literal by the type it is expected to have, and a TYPE
// ARGUMENT fixes that for a collection's elements: listOf<Long>(1, 2),
// `val m: Map<Long, Int> = mapOf(1 to 2)`, Pair<Long, Int>(1, 2), `xs.add(3)` on a
// MutableList<Long>, `y = 7` into a `var y = 5L`. An Int can only reach a Long-typed
// slot as such a literal, so retyping by the static type changes nothing else.
// Types are read at WALK time from the declarations and from source TEXT (up.in,
// whitespace already stripped): a thunk carries no type, and the tag stack copies
// the values it carries, so text is the one shape a later production can read.

// The numeric types, the only type arguments that retype anything.
function ktNumTy(t) {
    var s = (t.length > 0 && t.charAt(t.length - 1) == "?") ? t.substring(0, t.length - 1) : t
    if (s == "Int" || s == "Long" || s == "Short" || s == "Byte" || s == "Float" || s == "Double") { return t }
    if (s == "UInt" || s == "ULong" || s == "UShort" || s == "UByte") { return t }
    return ""
}

// The type an integer LITERAL is retyped to: an Int literal already is an Int, and
// Kotlin gives an integer literal no Float or Double type at all, so only the other
// widths retype anything.
function ktLitTy(t) {
    var n = ktNumTy(t)
    var s = (n.length > 0 && n.charAt(n.length - 1) == "?") ? n.substring(0, n.length - 1) : n
    return (s == "Int" || s == "Float" || s == "Double") ? "" : n
}

// The type arguments of `Base<A,B<C>>`, split at the top-level commas.
function ktSplitTArgs(s) {
    var out = []
    var depth = 0
    var from = 0
    for (var i = 0; i < s.length; i++) {
        var c = s.charAt(i)
        if (c == "<") { depth++ }
        else if (c == ">") { depth-- }
        else if (c == "," && depth == 0) { out.push(s.substring(from, i)); from = i + 1 }
    }
    out.push(s.substring(from))
    return out
}

var ktPrimArrElem = {"$IntArray": "Int", "$LongArray": "Long", "$ShortArray": "Short",
                     "$ByteArray": "Byte", "$FloatArray": "Float", "$DoubleArray": "Double",
                     "$UIntArray": "UInt", "$ULongArray": "ULong", "$UShortArray": "UShort",
                     "$UByteArray": "UByte"}
var ktOneArgColl = {"$List": 1, "$MutableList": 1, "$ArrayList": 1, "$Set": 1, "$MutableSet": 1,
                    "$HashSet": 1, "$LinkedHashSet": 1, "$Collection": 1, "$MutableCollection": 1,
                    "$Iterable": 1, "$Array": 1, "$Sequence": 1}
var ktTwoArgColl = {"$Map": 1, "$MutableMap": 1, "$HashMap": 1, "$LinkedHashMap": 1, "$Pair": 1}

// {k, v}: the key and element (value) types a collection type fixes - a Pair's
// first and second - or null when it fixes none. Each is a numeric type, or a
// collection type that itself fixes numeric elements (List<List<Long>>), "" else.
function ktElemTypes(ty) {
    var t = (ty.length > 0 && ty.charAt(ty.length - 1) == "?") ? ty.substring(0, ty.length - 1) : ty
    var prim = ktPrimArrElem["$" + t]
    if (prim != undefined) { return {k: "", v: prim} }
    var lt = t.indexOf("<")
    if (lt < 0 || t.charAt(t.length - 1) != ">") { return null }
    var base = t.substring(0, lt)
    var targs = ktSplitTArgs(t.substring(lt + 1, t.length - 1))
    var k = ""
    var v = ""
    if (ktOneArgColl["$" + base] != undefined && targs.length == 1) { v = ktElemTy(targs[0]) }
    else if (ktTwoArgColl["$" + base] != undefined && targs.length == 2) { k = ktElemTy(targs[0]); v = ktElemTy(targs[1]) }
    if (k == "" && v == "") { return null }
    return {k: k, v: v}
}
function ktElemTy(t) {
    var n = ktNumTy(t)
    if (n != "") { return n }
    return ktLitElems(t) != null ? t : ""
}
function ktTyElems(ty) { return ty == "" ? null : ktElemTypes(ty) }
// The element types a collection's integer literals are retyped to, or null: a
// numeric type, or a nested collection type whose own elements are retyped.
function ktLitElems(ty) {
    var el = ktTyElems(ty)
    if (el == null) { return null }
    var k = ktLitElTy(el.k)
    var v = ktLitElTy(el.v)
    if (k == "" && v == "") { return null }
    return {k: k, v: v}
}
function ktLitElTy(t) { return t.indexOf("<") >= 0 ? t : ktLitTy(t) }

// The type a FUNCTION type's result adopts - `() -> Long`, `(Int) -> Long`, the
// nullable `(() -> Long)?` - or "" for any other type, for a result that retypes
// nothing, and for a function type with a receiver (`T.() -> Long`, whose value
// carries its own receiver protocol).
function ktFnRetTy(ty) {
    var t = (ty.length > 0 && ty.charAt(ty.length - 1) == "?") ? ty.substring(0, ty.length - 1) : ty
    if (t.length > 1 && t.charAt(0) == "(" && ktMatchClose(t, 0, "(", ")") == t.length - 1) { t = t.substring(1, t.length - 1) }
    if (t.substring(0, 7) == "suspend") { t = t.substring(7) }
    if (t.length == 0 || t.charAt(0) != "(") { return "" }
    var e = ktMatchClose(t, 0, "(", ")")
    if (e < 0 || t.substring(e + 1, e + 3) != "->") { return "" }
    return ktLitTy(t.substring(e + 3))
}
// Whether a collection type is a Map (its `in`, get and containsKey take a KEY).
function ktIsMapTy(ty) {
    var t = (ty.length > 0 && ty.charAt(ty.length - 1) == "?") ? ty.substring(0, ty.length - 1) : ty
    var lt = t.indexOf("<")
    var base = lt < 0 ? t : t.substring(0, lt)
    return base == "Map" || base == "MutableMap" || base == "HashMap" || base == "LinkedHashMap"
}

// The static type of a builder call: the collection kind it builds, completed by
// its explicit type arguments ("" when they are absent and the kind is generic).
var ktBuilderKind = {"$listOf": "List", "$listOfNotNull": "List", "$emptyList": "List",
                     "$mutableListOf": "MutableList", "$arrayListOf": "MutableList",
                     "$setOf": "Set", "$setOfNotNull": "Set", "$emptySet": "Set",
                     "$mutableSetOf": "MutableSet", "$hashSetOf": "MutableSet",
                     "$linkedSetOf": "MutableSet", "$sortedSetOf": "MutableSet",
                     "$mapOf": "Map", "$emptyMap": "Map", "$mutableMapOf": "MutableMap",
                     "$hashMapOf": "MutableMap", "$linkedMapOf": "MutableMap",
                     "$sortedMapOf": "MutableMap", "$arrayOf": "Array", "$emptyArray": "Array",
                     "$sequenceOf": "Sequence", "$Pair": "Pair",
                     "$ArrayList": "MutableList", "$HashSet": "MutableSet",
                     "$Array": "Array", "$List": "List", "$MutableList": "MutableList",
                     "$LinkedHashSet": "MutableSet", "$HashMap": "MutableMap",
                     "$LinkedHashMap": "MutableMap",
                     "$intArrayOf": "IntArray", "$longArrayOf": "LongArray",
                     "$shortArrayOf": "ShortArray", "$byteArrayOf": "ByteArray",
                     "$floatArrayOf": "FloatArray", "$doubleArrayOf": "DoubleArray",
                     "$IntArray": "IntArray", "$LongArray": "LongArray", "$ShortArray": "ShortArray",
                     "$ByteArray": "ByteArray", "$FloatArray": "FloatArray",
                     "$DoubleArray": "DoubleArray"}
function ktBuilderType(name, targs) {
    var kind = ktBuilderKind["$" + name]
    if (kind == undefined) { return "" }
    if (ktPrimArrElem["$" + kind] != undefined) { return kind }
    if (targs.length == 0) { return "" }
    return kind + "<" + targs.join(",") + ">"
}

// The declared names and their static types, for the sites that retype by a NAME
// (an assignment, an indexed write, a method argument, an elvis operand). Each
// declaration is kept with its source offset, and every finished body (a function,
// a lambda, a class) with its extent; a use sees the closest earlier declaration of
// the name that no finished body it is outside of encloses - Kotlin's scoping, read
// off positions so a backtracked parse attempt leaves nothing wrong behind. ktAt is
// the offset of the production being tagged, set by the tag before it calls in.
// A declaration's innermost finished body is the only one that matters (every
// outer body that holds it holds that one too), so it is stamped on the
// declaration when the body finishes, and ktOpen keeps the declarations no
// finished body holds yet.
var ktAt = 0
var ktDecls = {}
var ktOpen = []
function ktNoteDecl(name, ty) {
    if (name == null || name == undefined) { return }
    var l = ktDecls["$" + name]
    if (l == undefined) { l = []; ktDecls["$" + name] = l }
    var d = {ty: ty, at: ktAt, from: -1, to: -1, vr: ktIsVarAt(ktAt)}
    l.push(d)
    ktOpen.push(d)
}
// Whether the declaration at source offset `at` is spelled `var`: a word `var`
// before its name ends (modifiers may precede it, `private var x: T`). A `val`, a
// function parameter, a loop or lambda variable cannot be reassigned.
function ktIsVarAt(at) {
    var s = c.srcSlice(at, at + 200)
    var i = 0
    while (i < s.length) {
        var ch = s.charAt(i)
        if (ch == ":" || ch == "=" || ch == "(" || ch == ")" || ch == "," || ch == "{" || ch == ";") { return false }
        if (!ktIdChar(ch)) { i++; continue }
        var j = i
        while (j < s.length && ktIdChar(s.charAt(j))) { j++ }
        if (s.substring(i, j) == "var") { return true }
        i = j
    }
    return false
}
function ktNoteBody(from, to) {
    var still = []
    for (var i = 0; i < ktOpen.length; i++) {
        var d = ktOpen[i]
        if (from <= d.at && d.at <= to) { d.from = from; d.to = to } else { still.push(d) }
    }
    ktOpen = still
}
function ktDeclTypeOf(name) {
    var d = ktDeclAt(name)
    return d == null ? "" : d.ty
}
// The declaration of `name` a use at ktAt sees, or null.
function ktDeclAt(name) {
    var l = ktDecls["$" + name]
    if (l == undefined) { return null }
    var best = -1
    var hit = null
    for (var i = 0; i < l.length; i++) {
        var d = l[i]
        if (d.at > ktAt || d.at < best) { continue }
        if (d.from >= 0 && !(d.from <= ktAt && ktAt <= d.to)) { continue }
        best = d.at
        hit = d
    }
    return hit
}
// Whether the target of a compound assignment `ref op= v` at offset `at` is a
// `val`: a local or parameter, a top-level property, or a property of the class the
// path's owner has (by name alone when that owner is not known here). Kotlin
// resolves `a += b` on a val as a.plusAssign(b), in place on a mutable collection,
// and on a var whose type has no plusAssign as a = a + b (a var of a MUTABLE
// collection type is ambiguous and rejected, so only a read-only type reaches the
// plus form). Read when every declaration has been seen: a member or a top-level
// property may be declared after its use.
function ktAssignIsVal(ref, at) {
    var n = ref.path.length
    if (n == 0) {
        var saved = ktAt
        ktAt = at
        var d = ktDeclAt(ref.name)
        ktAt = saved
        if (d != null) { return !d.vr }
        var l = ktDecls["$" + ref.name]
        if (l != undefined) { for (var t = 0; t < l.length; t++) { if (l[t].from < 0) { return !l[t].vr } } }
        var cls = ktClassAt(at)
        return cls != "" && ktMemberIsVal(cls, ref.name)
    }
    var key = ref.path[n - 1].key
    if (key == undefined) { return false }
    var names = []
    for (var i = 0; i < n - 1; i++) {
        if (ref.path[i].key == undefined) { return ktMemberIsVal("", key) }
        names.push(ref.path[i].key)
    }
    return ktMemberIsVal(ktBaseName(ktFieldChainType(ref.name, at, names)), key)
}
// Whether property `name` of class `cls` is a val; for a class not known here, or
// one that inherits the property, whether every property of that name is.
function ktMemberIsVal(cls, name) {
    var l = ktMemberVars["$" + name]
    if (l == undefined) { return false }
    var e = cls == "" ? undefined : ktClassExt["$" + cls]
    if (e != undefined) {
        for (var i = 0; i < l.length; i++) { if (e.from <= l[i].at && l[i].at <= e.to) { return !l[i].vr } }
    }
    for (var j = 0; j < l.length; j++) { if (l[j].vr) { return false } }
    return true
}

// An equality chain's operators, renamed "==." / "!=." (equality that tells the
// integer types apart) unless an operand is an integer LITERAL, which keeps "==" /
// "!=" (by value): a literal whose expected type was not seen here is still an Int.
// The operands are read off the source (from..to); a text the scan cannot split is
// treated as having a literal.
function ktEqOps(items, from, to) {
    if (items.length < 3) { return items }
    if (ktEqHasLit(c.srcSlice(from, to))) { return items }
    var out = []
    for (var i = 0; i < items.length; i++) {
        var it = items[i]
        if (it === "==") { out.push("==.") }
        else if (it === "!=") { out.push("!=.") }
        else { out.push(it) }
    }
    return out
}
function ktEqHasLit(raw) {
    var depth = 0
    var from = 0
    var ops = 0
    var i = 0
    while (i < raw.length) {
        var ch = raw.charAt(i)
        if (ch == "\"" || ch == "'" || (ch == "/" && (raw.charAt(i + 1) == "/" || raw.charAt(i + 1) == "*"))) { return true }
        if (ch == "(" || ch == "[" || ch == "{") { depth++ }
        else if (ch == ")" || ch == "]" || ch == "}") { depth-- }
        else if (depth == 0 && (ch == "=" || ch == "!") && raw.charAt(i + 1) == "=") {
            var prev = i > 0 ? raw.charAt(i - 1) : ""
            if (ch == "=" && (prev == "<" || prev == ">" || prev == "!" || prev == "=")) { i++; continue }
            if (ktIsIntLitText(raw.substring(from, i))) { return true }
            ops++
            i = i + 2
            if (raw.charAt(i) == "=") { i++ }
            from = i
            continue
        }
        i++
    }
    if (ops == 0) { return true }
    return ktIsIntLitText(raw.substring(from))
}
// Whether an operand's text is an integer literal, signed and parenthesized or not.
function ktIsIntLitText(t) {
    var s = ""
    for (var i = 0; i < t.length; i++) {
        var ch = t.charAt(i)
        if (ch != " " && ch != "\t" && ch != "\n" && ch != "\r" && ch != "(" && ch != ")") { s = s + ch }
    }
    if (s.length > 0 && (s.charAt(0) == "-" || s.charAt(0) == "+")) { s = s.substring(1) }
    if (s.length == 0) { return false }
    var c0 = s.charAt(0)
    if (!(c0 >= "0" && c0 <= "9")) { return false }
    for (var j = 1; j < s.length; j++) {
        var cj = s.charAt(j)
        var ok = (cj >= "0" && cj <= "9") || (cj >= "a" && cj <= "f") || (cj >= "A" && cj <= "F") || cj == "_" ||
                 cj == "x" || cj == "X" || cj == "l" || cj == "L" || cj == "u" || cj == "U"
        if (!ok) { return false }
    }
    return true
}

// The declared types of class PROPERTIES, for a receiver that is a member chain
// (`b.xs.add(3)`, `this.m[1] = v`): each property with its source offset, and each
// class with its extent, so a property is found in the class a chain step names.
// Read lazily (at emit / first run), when every class has been seen.
var ktMemberTys = {}
var ktMemberVars = {}
var ktClassExt = {}
var ktClassNames = []
function ktNoteMember(name, ty) {
    if (name == null || name == undefined) { return }
    var kl = ktMemberVars["$" + name]
    if (kl == undefined) { kl = []; ktMemberVars["$" + name] = kl }
    kl.push({vr: ktIsVarAt(ktAt), at: ktAt})
    if (ty == "") { return }
    var l = ktMemberTys["$" + name]
    if (l == undefined) { l = []; ktMemberTys["$" + name] = l }
    l.push({ty: ty, at: ktAt})
}
// A class declaration's extent, its name read off its text (`classS91(...){...}`,
// `dataclassP(...)`).
function ktNoteClass(txt, from, to) {
    var i = txt.indexOf("class")
    if (i < 0) { return }
    var j = i + 5
    while (j < txt.length && ktIdChar(txt.charAt(j))) { j++ }
    if (j == i + 5) { return }
    var name = txt.substring(i + 5, j)
    if (ktClassExt["$" + name] == undefined) { ktClassNames.push(name) }
    ktClassExt["$" + name] = {from: from, to: to}
}
// The innermost class whose body holds offset `at`, or "".
function ktClassAt(at) {
    var best = ""
    var size = 0
    for (var i = 0; i < ktClassNames.length; i++) {
        var e = ktClassExt["$" + ktClassNames[i]]
        if (e.from <= at && at <= e.to && (best == "" || e.to - e.from < size)) { best = ktClassNames[i]; size = e.to - e.from }
    }
    return best
}
// A type's class name: `Box<Long>?` is Box.
function ktBaseName(ty) {
    var t = (ty.length > 0 && ty.charAt(ty.length - 1) == "?") ? ty.substring(0, ty.length - 1) : ty
    var lt = t.indexOf("<")
    return lt < 0 ? t : t.substring(0, lt)
}
// The type an extension call tests its receiver against, from the receiver's text
// (`String?.`, `a.Box<T>.`): the simple base name, keeping a `?` so a null receiver
// still matches, or "" for an annotated or parenthesized receiver.
function ktExtRecvTy(txt) {
    if (txt.indexOf("@") >= 0 || txt.indexOf("(") >= 0) { return "" }
    var t = txt.substring(0, txt.length - 1)
    var segs = ktBaseName(t).split(".")
    return segs[segs.length - 1] + ((t.length > 0 && t.charAt(t.length - 1) == "?") ? "?" : "")
}
// Whether a call tests an extension's receiver type at all: the value types whose
// runtime test is exact, and the classes, interfaces, objects and enums the program
// declares. Any other receiver (a type parameter, Any, a collection) is not tested,
// and the extension then applies to every receiver of its name. Read at the call,
// when every type has been seen.
var ktExtValueTys = {"$String": true, "$Char": true, "$Boolean": true, "$Int": true, "$Long": true,
                     "$Short": true, "$Byte": true, "$Double": true, "$Float": true}
function ktExtTested(ty) {
    var b = ktBaseName(ty)
    return ktExtValueTys["$" + b] == true || ktClassExt["$" + b] != undefined || ktDeclTypeNames["$" + b] == true
}
// The interfaces, objects and enum classes the program declares (the classes are
// ktClassExt), each noted from its declaration's text after keyword kw
// (`interfaceShape{...}`, `objectOrigin{...}`, `enumclassColor{...}`).
var ktDeclTypeNames = {}
function ktNoteTypeName(txt, kw) {
    var i = txt.indexOf(kw)
    if (i < 0) { return }
    var j = i + kw.length
    while (j < txt.length && ktIdChar(txt.charAt(j))) { j++ }
    if (j > i + kw.length) { ktDeclTypeNames["$" + txt.substring(i + kw.length, j)] = true }
}
// The declared type of property `name` of class `cls`; for a class not known here,
// the name's type when every declaration of it agrees, else "".
function ktMemberTypeIn(cls, name) {
    var l = ktMemberTys["$" + name]
    if (l == undefined) { return "" }
    var e = cls == "" ? undefined : ktClassExt["$" + cls]
    if (e != undefined) {
        for (var i = 0; i < l.length; i++) { if (e.from <= l[i].at && l[i].at <= e.to) { return l[i].ty } }
        return ""
    }
    for (var j = 1; j < l.length; j++) { if (l[j].ty != l[0].ty) { return "" } }
    return l[0].ty
}
// The declared type of `p.f1...fn` at offset `at`: p a local, a parameter, a
// property or `this`, each step a property of the class the step before it has.
// "" for p means an owner not known here, so the steps go by name alone.
function ktFieldChainType(p, at, names) {
    var ty = ""
    if (p == "this") { ty = ktClassAt(at) }
    else if (p != "") {
        var saved = ktAt
        ktAt = at
        ty = ktDeclTypeOf(p)
        ktAt = saved
    }
    for (var i = 0; i < names.length; i++) { ty = ktMemberTypeIn(ktBaseName(ty), names[i]) }
    return ty
}
// The receiver type of a postfix chain's step i (a call or an index after `.name`):
// the whole chain when every step before it is a property read, else the last
// property by name alone.
function ktSfxChainType(p, at, sfx, i) {
    var names = []
    var whole = true
    for (var j = 0; j < i; j++) {
        if (sfx[j].kind == "field") { names.push(sfx[j].name) } else { whole = false }
    }
    if (whole) { return ktFieldChainType(p, at, names) }
    return ktFieldChainType("", at, [sfx[i - 1].name])
}
// The name a postfix chain's text starts with ("" when it starts with a call or a
// literal): the owner ktSfxChainType begins at.
function ktLeadName(t) {
    if (t == undefined) { return "" }
    var i = 0
    while (i < t.length && ktIdChar(t.charAt(i))) { i++ }
    if (i == 0 || i == t.length || t.charAt(i) == "(" || t.charAt(i) == "{" || t.charAt(i) == "<") { return "" }
    var c0 = t.charAt(0)
    return (c0 >= "0" && c0 <= "9") ? "" : t.substring(0, i)
}

// Generic functions and classes by name: their type parameters and the declared
// types of their value parameters, read off the declaration's source text, for a
// call with explicit type arguments (`ident<Long>(5)`, `Box<Long>(3)`) - a
// parameter declared with a type parameter adopts the argument's type. A name
// declared generic twice answers nothing.
var ktGenerics = {}
function ktNoteGeneric(raw, kw) {
    var i = raw.indexOf(kw)
    if (i < 0) { return }
    i = ktSkipWs(raw, i + kw.length)
    var tps = []
    if (raw.charAt(i) == "<") {
        var e = ktMatchClose(raw, i, "<", ">")
        if (e < 0) { return }
        tps = ktTParamNames(raw.substring(i + 1, e))
        i = ktSkipWs(raw, e + 1)
    }
    var n0 = i
    while (i < raw.length && ktIdChar(raw.charAt(i))) { i++ }
    var name = raw.substring(n0, i)
    i = ktSkipWs(raw, i)
    if (raw.charAt(i) == "<") {
        var e2 = ktMatchClose(raw, i, "<", ">")
        if (e2 < 0) { return }
        tps = ktTParamNames(raw.substring(i + 1, e2))
        i = e2 + 1
    }
    if (name == "" || tps.length == 0) { return }
    // A class's primary constructor may be preceded by modifiers and `constructor`.
    while (i < raw.length && raw.charAt(i) != "(" && raw.charAt(i) != "{" && raw.charAt(i) != ":" && raw.charAt(i) != "=") { i++ }
    var ptys = []
    if (raw.charAt(i) == "(") {
        var pe = ktMatchClose(raw, i, "(", ")")
        if (pe < 0) { return }
        var parts = ktSplitTop(raw.substring(i + 1, pe), ",")
        for (var p = 0; p < parts.length; p++) {
            var colon = ktSplitTop(parts[p], ":")
            ptys.push(colon.length < 2 ? "" : stripWs(ktSplitTop(colon[1], "=")[0]))
        }
    }
    ktGenerics["$" + name] = ktGenerics["$" + name] == undefined ? {tps: tps, ptys: ptys} : "#"
}
function ktSkipWs(s, i) {
    while (i < s.length && (s.charAt(i) == " " || s.charAt(i) == "\t" || s.charAt(i) == "\n" || s.charAt(i) == "\r")) { i++ }
    return i
}
// `out T : Comparable<T>, reified U` -> ["T", "U"]: each entry's last word before
// its bound.
function ktTParamNames(s) {
    var out = []
    var parts = ktSplitTop(s, ",")
    for (var i = 0; i < parts.length; i++) {
        var w = ktSplitTop(parts[i], ":")[0]
        var e = w.length
        while (e > 0 && !ktIdChar(w.charAt(e - 1))) { e-- }
        var b = e
        while (b > 0 && ktIdChar(w.charAt(b - 1))) { b-- }
        if (e > b) { out.push(w.substring(b, e)) }
    }
    return out
}
// A text split at the separator where no bracket of any kind is open and outside
// string and char literals.
function ktSplitTop(s, sep) {
    var out = []
    var depth = 0
    var from = 0
    var i = 0
    while (i < s.length) {
        var ch = s.charAt(i)
        if (ch == "\"" || ch == "'") {
            var q = ktQuoteEnd(s, i)
            if (q < 0) { break }
            i = q + 1
            continue
        }
        if (ch == "(" || ch == "[" || ch == "{" || ch == "<") { depth++ }
        else if (ch == ")" || ch == "]" || ch == "}" || (ch == ">" && s.charAt(i - 1) != "-")) { depth-- }
        else if (depth == 0 && ch == sep) { out.push(s.substring(from, i)); from = i + 1 }
        i++
    }
    out.push(s.substring(from))
    return out
}
// The types the positional arguments of `name<targs>(...)` adopt (nargs of them),
// or null: a generic's parameter types with its type parameters replaced by the
// type arguments; maxOf / minOf take their one type argument for every argument.
function ktGenericArgTys(name, targs, nargs) {
    var out = []
    if ((name == "maxOf" || name == "minOf") && targs.length == 1) {
        for (var m = 0; m < nargs; m++) { out.push(targs[0]) }
        return out
    }
    var g = ktGenerics["$" + name]
    if (g == undefined || g == "#" || g.tps.length != targs.length) { return null }
    for (var i = 0; i < nargs && i < g.ptys.length; i++) { out.push(ktSubstTParams(g.ptys[i], g.tps, targs)) }
    return out
}
// A type text with every type parameter name replaced by its type argument.
function ktSubstTParams(ty, tps, targs) {
    var out = ""
    var i = 0
    while (i < ty.length) {
        if (!ktIdChar(ty.charAt(i))) { out = out + ty.charAt(i); i++; continue }
        var b = i
        while (i < ty.length && ktIdChar(ty.charAt(i))) { i++ }
        var w = ty.substring(b, i)
        for (var k = 0; k < tps.length; k++) { if (tps[k] == w) { w = targs[k] } }
        out = out + w
    }
    return out
}

// The type the left operand of `x in c` adopts, c being the source text from..to:
// c's element type, or its key type for a map; "" when c's type is not known.
function ktInElemTy(from, to) {
    var t = stripWs(c.srcSlice(from, to))
    var i = 0
    while (i < t.length && ktIdChar(t.charAt(i))) { i++ }
    var ty = ""
    if (i == t.length) {
        var saved = ktAt
        ktAt = from
        ty = ktDeclTypeOf(t)
        ktAt = saved
    } else if (ktCallEnd(t) == t.length) {
        ty = ktCallType(t)
    }
    var el = ktTyElems(ty)
    if (el == null) { return "" }
    return ktLitTy(ktIsMapTy(ty) ? el.k : el.v)
}

// The positions in a postfix chain whose receiver is a member READ: a call or an
// index directly after a `.name` step.
function ktChainSites(sfx) {
    var out = []
    for (var i = 1; i < sfx.length; i++) {
        var k = sfx[i].kind
        if ((k == "mcall" || k == "index") && sfx[i - 1].kind == "field") { out.push(i) }
    }
    return out
}

function ktIdChar(c) {
    return (c >= "a" && c <= "z") || (c >= "A" && c <= "Z") || (c >= "0" && c <= "9") || c == "_"
}
// The index of the quote closing the string or char literal opening at `at` (a raw
// """ string's last quote; a template's ${...} may hold quotes of its own), or -1.
function ktQuoteEnd(s, at) {
    var q = s.charAt(at)
    if (q == "\"" && s.substring(at, at + 3) == "\"\"\"") {
        var e = s.indexOf("\"\"\"", at + 3)
        return e < 0 ? -1 : e + 2
    }
    var i = at + 1
    while (i < s.length) {
        var ch = s.charAt(i)
        if (ch == "\\") { i = i + 2; continue }
        if (ch == q) { return i }
        if (q == "\"" && ch == "$" && s.charAt(i + 1) == "{") {
            var e2 = ktMatchClose(s, i + 1, "{", "}")
            if (e2 < 0) { return -1 }
            i = e2 + 1
            continue
        }
        i++
    }
    return -1
}
// The index of the bracket closing the one at `at`, or -1. String and char literals
// are stepped over, so a bracket inside one does not count.
function ktMatchClose(s, at, open, close) {
    var depth = 0
    for (var i = at; i < s.length; i++) {
        var c = s.charAt(i)
        if (c == "\"" || c == "'") {
            i = ktQuoteEnd(s, i)
            if (i < 0) { return -1 }
            continue
        }
        if (c == open) { depth++ }
        else if (c == close) { depth--; if (depth == 0) { return i } }
    }
    return -1
}
// Where the call a text starts with ends - name, type arguments, then argument
// lists and trailing lambdas - or -1 when it does not start with one.
function ktCallEnd(s) {
    var i = 0
    while (i < s.length && ktIdChar(s.charAt(i))) { i++ }
    if (i == 0) { return -1 }
    if (i < s.length && s.charAt(i) == "<") {
        i = ktMatchClose(s, i, "<", ">")
        if (i < 0) { return -1 }
        i++
    }
    var groups = 0
    while (i < s.length && (s.charAt(i) == "(" || s.charAt(i) == "{")) {
        var e = (s.charAt(i) == "(") ? ktMatchClose(s, i, "(", ")") : ktMatchClose(s, i, "{", "}")
        if (e < 0) { return -1 }
        i = e + 1
        groups++
    }
    return groups == 0 ? -1 : i
}
// The builder type of a call's text.
function ktCallType(s) {
    var i = 0
    while (i < s.length && ktIdChar(s.charAt(i))) { i++ }
    var targs = []
    if (i < s.length && s.charAt(i) == "<") { targs = ktSplitTArgs(s.substring(i + 1, ktMatchClose(s, i, "<", ">"))) }
    else if (ktOneArgColl["$" + ktBuilderKind["$" + s.substring(0, i)]] != undefined && ktAllLongLits(s, i)) { targs = ["Long"] }
    return ktBuilderType(s.substring(0, i), targs)
}
// Whether a call's one argument list, opening at `at`, is a non-empty run of Long
// literals - `setOf(1L, 2L)` is a Set<Long> with no type argument written.
function ktAllLongLits(s, at) {
    if (s.charAt(at) != "(") { return false }
    var e = ktMatchClose(s, at, "(", ")")
    if (e != s.length - 1 || e == at + 1) { return false }
    var args = ktSplitTArgs(s.substring(at + 1, e))
    for (var i = 0; i < args.length; i++) {
        var a = args[i]
        if (a.length > 0 && a.charAt(0) == "-") { a = a.substring(1) }
        if (a.length == 0 || !ktIsLongLit(a)) { return false }
    }
    return true
}
// A Long literal's text: an L suffix, or no suffix and a value past Int.
function ktIsLongLit(s) {
    var body = ""
    for (var i = 0; i < s.length; i++) { if (s.charAt(i) != "_") { body = body + s.charAt(i) } }
    var last = body.charAt(body.length - 1)
    if (last == "L" || last == "l") { body = body.substring(0, body.length - 1) }
    else if (!(last >= "0" && last <= "9") && !(last >= "a" && last <= "f") && !(last >= "A" && last <= "F")) { return false }
    var radix = 10
    if (body.length > 2 && body.charAt(0) == "0" && (body.charAt(1) == "x" || body.charAt(1) == "X")) { radix = 16; body = body.substring(2) }
    else if (body.length > 2 && body.charAt(0) == "0" && (body.charAt(1) == "b" || body.charAt(1) == "B")) { radix = 2; body = body.substring(2) }
    for (var j = 0; j < body.length; j++) {
        var c = body.charAt(j)
        var ok = (c >= "0" && c <= "9") || (radix == 16 && ((c >= "a" && c <= "f") || (c >= "A" && c <= "F")))
        if (!ok) { return false }
    }
    if (body.length == 0) { return false }
    return last == "L" || last == "l" || parseInt(body, radix) > 2147483647
}
// The static type of an initializer's text: "Long" for a Long literal, the builder
// type of a single builder call (mutableListOf<Long>(), LongArray(2)), the class
// of a constructor call, "" else.
function ktExprType(t) {
    var s = (t.length > 0 && t.charAt(0) == "-") ? t.substring(1) : t
    if (s.length == 0) { return "" }
    var c0 = s.charAt(0)
    if (c0 >= "0" && c0 <= "9") { return ktIsLongLit(s) ? "Long" : "" }
    var e = ktCallEnd(s)
    if (e != s.length) { return "" }
    var bt = ktCallType(s)
    if (bt != "") { return bt }
    // A constructor call types the declaration by its class: `val b = Box(...)`.
    var pa = s.indexOf("(")
    return (c0 >= "A" && c0 <= "Z" && pa > 0) ? ktBaseName(s.substring(0, pa)) : ""
}
// The static type of the operand an expression's text starts with when it is
// directly followed by `sep` - the receiver of `xs.add(3)` or of
// `listOf<Long>().sum()`, the left operand of `n ?: 0`: a declared name's type or a
// builder call's, else "".
function ktLeadType(t, sep) {
    if (t == undefined) { return "" }
    var i = 0
    while (i < t.length && ktIdChar(t.charAt(i))) { i++ }
    if (i == 0) { return "" }
    if (t.substring(i, i + sep.length) == sep) { return ktDeclTypeOf(t.substring(0, i)) }
    var e = ktCallEnd(t)
    if (e < 0 || t.substring(e, e + sep.length) != sep) { return "" }
    return ktCallType(t.substring(0, e))
}
// A declaration's initializer text: what follows `val`/`var`, the name, the
// declared type when there is one and `=`, without a trailing `;`.
function ktInitText(dtext, name, dtype) {
    if (dtext == undefined) { return "" }
    var t = dtext.substring(4 + name.length + (dtype == "" ? 0 : 1 + dtype.length))
    if (t.length > 0 && t.charAt(t.length - 1) == ";") { t = t.substring(0, t.length - 1) }
    return t
}
// The types a method's positional arguments take on a collection whose element
// (key) types are known: add(x), add(i, x) and set(i, x) retype x by the element
// type, put(k, v) k by the key type and v by the value type. "" leaves one alone.
// On a map, get / containsKey / getOrDefault / remove / getValue / getOrElse /
// getOrPut retype the KEY, containsValue the value; on a list or set, contains /
// indexOf / lastIndexOf / remove retype the element, and addAll / removeAll /
// containsAll / retainAll retype the collection argument's elements.
function ktMArgTypes(name, nargs, el, isMap) {
    var v = ktLitElTy(el.v)
    var k = ktLitElTy(el.k)
    if (isMap) {
        if (name == "put") { return [k, v] }
        if (name == "get" || name == "containsKey" || name == "remove" || name == "getValue") { return [k] }
        if (name == "getOrDefault") { return [k, v] }
        if (name == "getOrElse" || name == "getOrPut") { return [k, "()->" + v] }
        if (name == "containsValue") { return [v] }
        return []
    }
    if (name == "add" || name == "set") { return nargs == 1 ? [v] : ["", v] }
    if (name == "getOrElse") { return ["", "(Int)->" + v] }
    if (name == "contains" || name == "indexOf" || name == "lastIndexOf" || name == "remove") { return [v] }
    if (v != "" && (name == "addAll" || name == "removeAll" || name == "containsAll" || name == "retainAll")) {
        return nargs == 1 ? ["List<" + v + ">"] : ["", "List<" + v + ">"]
    }
    if (name == "put") { return [k, v] }
    return []
}
// The type the right operand of `xs += x` / `xs -= x` adopts on a collection of
// type ty: a map's entry Pair (or Map) by the map type; on a list or set the element
// - or a collection of elements, which the "List<E>" step retypes - "" else.
function ktCompoundTys(ty) {
    var el = ktTyElems(ty)
    if (el == null) { return [] }
    if (ktIsMapTy(ty)) { return [ty] }
    var v = ktLitElTy(el.v)
    return v == "" ? [] : ["List<" + v + ">", v]
}

// The types the right operand of `xs + x` / `xs - x` adopts when the left one is
// a name or a builder call of a known collection type - as for `xs += x`
// (ktCompoundTys); [] else. items are a sum's operands and operators, txt its
// source text and at its offset.
function ktPlusTys(items, txt, at) {
    if (items.length != 3 || (items[1] !== "+" && items[1] !== "-")) { return [] }
    var saved = ktAt
    ktAt = at
    var tys = ktCompoundTys(ktLeadType(txt, items[1]))
    ktAt = saved
    return tys
}
// The type a method's RESULT takes: sum() of an empty collection is its element
// type's zero, not an Int.
function ktMResType(name, nargs, el) {
    return (name == "sum" && nargs == 0 && el.v != "Int") ? el.v : ""
}

// The parameter names of the stdlib members a call may fill by NAME
// (`xs.joinToString(limit = 2)`), in declaration order, for a member no user
// declaration of that name claims. Each grammar places the named arguments into
// these slots and leaves a skipped slot undefined, which the member reads as its
// default.
var ktStdParamNames = {
    "$joinToString": ["separator", "prefix", "postfix", "limit", "truncated", "transform"],
    "$windowed": ["size", "step", "partialWindows", "transform"],
    "$chunked": ["size", "transform"],
    "$padStart": ["length", "padChar"],
    "$padEnd": ["length", "padChar"]
}
function ktStdParams(name) { return ktStdParamNames["$" + name] }

// The java.* classes an import may name that the runtime provides, each with the
// global it is provided under (hostGlobals / js_ktglobal). Such an import binds its
// simple name, or its alias, to that global: after `import java.util.Random`,
// `Random(1)` is java.util.Random, not kotlin.random.Random, and `ArrayDeque()`
// java.util.ArrayDeque. The other collections are the classes kotlin.collections
// already typealiases (ArrayList is java.util.ArrayList), so their import binds the
// same constructor.
var ktJavaClassImports = [["java.util.Random", "java.util.Random"], ["java.util.ArrayList", "ArrayList"],
                          ["java.util.HashMap", "HashMap"], ["java.util.HashSet", "HashSet"],
                          ["java.util.LinkedHashMap", "LinkedHashMap"], ["java.util.LinkedHashSet", "LinkedHashSet"],
                          ["java.util.TreeMap", "TreeMap"], ["java.util.TreeSet", "TreeSet"],
                          ["java.util.LinkedList", "LinkedList"], ["java.util.ArrayDeque", "java.util.ArrayDeque"],
                          ["java.util.PriorityQueue", "PriorityQueue"]]
// The [simple name, global] pairs an import binds: the one class of an exact path,
// every provided class of the package for a `.*` import, none for anything else.
function ktJavaClassImport(path) {
    var star = path.length > 2 && path.substring(path.length - 2, path.length) == ".*"
    var out = []
    for (var i = 0; i < ktJavaClassImports.length; i++) {
        var cp = ktJavaClassImports[i][0]
        var dot = cp.length - 1
        while (dot > 0 && cp.charAt(dot) != ".") { dot-- }
        var hit = star ? cp.substring(0, dot) + ".*" == path : cp == path
        if (hit) { out.push([cp.substring(dot + 1, cp.length), ktJavaClassImports[i][1]]) }
    }
    return out
}
