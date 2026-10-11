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

// {k, v}: the numeric key and element (value) types a collection type fixes - a
// Pair's first and second - or null when it fixes none.
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
    if (ktOneArgColl["$" + base] != undefined && targs.length == 1) { v = ktNumTy(targs[0]) }
    else if (ktTwoArgColl["$" + base] != undefined && targs.length == 2) { k = ktNumTy(targs[0]); v = ktNumTy(targs[1]) }
    if (k == "" && v == "") { return null }
    return {k: k, v: v}
}
function ktTyElems(ty) { return ty == "" ? null : ktElemTypes(ty) }
// The element types a collection's integer literals are retyped to, or null.
function ktLitElems(ty) {
    var el = ktTyElems(ty)
    if (el == null) { return null }
    var k = ktLitTy(el.k)
    var v = ktLitTy(el.v)
    if (k == "" && v == "") { return null }
    return {k: k, v: v}
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
    var d = {ty: ty, at: ktAt, from: -1, to: -1}
    l.push(d)
    ktOpen.push(d)
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
    var l = ktDecls["$" + name]
    if (l == undefined) { return "" }
    var best = -1
    var ty = ""
    for (var i = 0; i < l.length; i++) {
        var d = l[i]
        if (d.at > ktAt || d.at < best) { continue }
        if (d.from >= 0 && !(d.from <= ktAt && ktAt <= d.to)) { continue }
        best = d.at
        ty = d.ty
    }
    return ty
}

function ktIdChar(c) {
    return (c >= "a" && c <= "z") || (c >= "A" && c <= "Z") || (c >= "0" && c <= "9") || c == "_"
}
// The index of the bracket closing the one at `at`, or -1 (also for a quote, whose
// contents could hold a bracket).
function ktMatchClose(s, at, open, close) {
    var depth = 0
    for (var i = at; i < s.length; i++) {
        var c = s.charAt(i)
        if (c == "\"" || c == "'") { return -1 }
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
    return ktBuilderType(s.substring(0, i), targs)
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
// Whether a text is one call of a collection builder, typed or not.
function ktIsBuilder(s) {
    if (ktCallEnd(s) != s.length) { return false }
    var i = 0
    while (i < s.length && ktIdChar(s.charAt(i))) { i++ }
    return ktBuilderKind["$" + s.substring(0, i)] != undefined
}
// The static type of an initializer's text: "Long" for a Long literal, the builder
// type of a single builder call (mutableListOf<Long>(), LongArray(2)), "" else.
function ktExprType(t) {
    var s = (t.length > 0 && t.charAt(0) == "-") ? t.substring(1) : t
    if (s.length == 0) { return "" }
    var c0 = s.charAt(0)
    if (c0 >= "0" && c0 <= "9") { return ktIsLongLit(s) ? "Long" : "" }
    var e = ktCallEnd(s)
    return e == s.length ? ktCallType(s) : ""
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
function ktMArgTypes(name, nargs, el) {
    var v = ktLitTy(el.v)
    if (name == "add" || name == "set") { return nargs == 1 ? [v] : ["", v] }
    if (name == "put") { return [ktLitTy(el.k), v] }
    return []
}
// The type a method's RESULT takes: sum() of an empty collection is its element
// type's zero, not an Int.
function ktMResType(name, nargs, el) {
    return (name == "sum" && nargs == 0 && el.v != "Int") ? el.v : ""
}
