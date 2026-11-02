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
        if (c == "<" || c == "(") { depth++ }
        else if (c == ")" || (c == ">" && (i == 0 || s.charAt(i - 1) != "-"))) { depth-- }
        else if (c == "," && depth == 0) { out.push(s.substring(from, i)); from = i + 1 }
    }
    out.push(s.substring(from))
    return out
}

var ktPrimArrElem = {"$IntArray": "Int", "$LongArray": "Long", "$ShortArray": "Short",
                     "$ByteArray": "Byte", "$FloatArray": "Float", "$DoubleArray": "Double",
                     "$UIntArray": "UInt", "$ULongArray": "ULong", "$UShortArray": "UShort",
                     "$UByteArray": "UByte", "$CharArray": "Char", "$BooleanArray": "Boolean"}
// A parameter's static type: a vararg parameter of element type T is an IntArray
// for Int (each primitive and unsigned type has its own array) and an Array<T>
// otherwise.
function ktParamTy(ty, vararg) {
    if (!vararg || ty == "") { return ty }
    if (ktPrimArrElem["$" + ty + "Array"] == ty) { return ty + "Array" }
    return "Array<" + ty + ">"
}
var ktOneArgColl = {"$List": 1, "$MutableList": 1, "$ArrayList": 1, "$Set": 1, "$MutableSet": 1,
                    "$HashSet": 1, "$LinkedHashSet": 1, "$Collection": 1, "$MutableCollection": 1,
                    "$Iterable": 1, "$Array": 1, "$Sequence": 1}
var ktTwoArgColl = {"$Map": 1, "$MutableMap": 1, "$HashMap": 1, "$LinkedHashMap": 1, "$Pair": 1}

// {k, v}: the key and element (value) types a collection type fixes - a Pair's
// first and second - or null when it fixes none. Each is a numeric type, or a
// collection type that itself fixes numeric elements (List<List<Long>>), "" else;
// with fns also a plain function type, which a callable reference or a lambda
// element adopts (List<(Int) -> Int>).
function ktElemTypes(ty, fns) {
    var t = (ty.length > 0 && ty.charAt(ty.length - 1) == "?") ? ty.substring(0, ty.length - 1) : ty
    var prim = ktPrimArrElem["$" + t]
    if (prim != undefined) { return {k: "", v: prim} }
    var lt = t.indexOf("<")
    if (lt < 0 || t.charAt(t.length - 1) != ">") { return null }
    var base = t.substring(0, lt)
    var targs = ktSplitTArgs(t.substring(lt + 1, t.length - 1))
    var k = ""
    var v = ""
    if (ktOneArgColl["$" + base] != undefined && targs.length == 1) { v = ktElemTy(targs[0], fns) }
    else if (ktTwoArgColl["$" + base] != undefined && targs.length == 2) { k = ktElemTy(targs[0], fns); v = ktElemTy(targs[1], fns) }
    if (k == "" && v == "") { return null }
    return {k: k, v: v}
}
function ktElemTy(t, fns) {
    var n = ktNumTy(t)
    if (n != "") { return n }
    if (fns && ktFnTyArity(t) >= 0) { return t }
    return ktLitElemsOf(t, fns) != null ? t : ""
}
// fns: also function element types (ktElemTypes).
function ktTyElems(ty, fns) { return ty == "" ? null : ktElemTypes(ty, fns == true) }
// The element types a collection's integer literals, callable references and
// lambdas are retyped to, or null: a numeric type, a function type, or a nested
// collection type whose own elements are retyped.
function ktLitElems(ty) { return ktLitElemsOf(ty, true) }
function ktLitElemsOf(ty, fns) {
    var el = ty == "" ? null : ktElemTypes(ty, fns)
    if (el == null) { return null }
    var k = ktLitElTy(el.k)
    var v = ktLitElTy(el.v)
    if (k == "" && v == "") { return null }
    return {k: k, v: v}
}
function ktLitElTy(t) { return t.indexOf("<") >= 0 || ktFnTyArity(t) >= 0 ? t : ktLitTy(t) }

// The type a FUNCTION type's result adopts - `() -> Long`, `(Int) -> Long`, the
// nullable `(() -> Long)?` - or "" for any other type, for a result that retypes
// nothing, and for a function type with a receiver (`T.() -> Long`, whose value
// carries its own receiver protocol).
function ktFnRetTy(ty) {
    var t = ktPlainFnTy(ty)
    return t == "" ? "" : ktLitTy(t.substring(ktMatchClose(t, 0, "(", ")") + 3))
}
// The parameter count of a plain function type (as ktFnRetTy reads it), which is
// the kotlin.jvm.functions class of a callable reference of that type; -1 for any
// other type.
function ktFnTyArity(ty) {
    var t = ktPlainFnTy(ty)
    return t == "" ? -1 : ktTyParamCount(t, 0, ktMatchClose(t, 0, "(", ")"))
}
// A plain function type `(A) -> T` without its nullable parentheses and
// `suspend`, or "".
function ktPlainFnTy(ty) {
    var t = (ty.length > 0 && ty.charAt(ty.length - 1) == "?") ? ty.substring(0, ty.length - 1) : ty
    if (t.length > 1 && t.charAt(0) == "(" && ktMatchClose(t, 0, "(", ")") == t.length - 1) { t = t.substring(1, t.length - 1) }
    if (t.substring(0, 7) == "suspend") { t = t.substring(7) }
    if (t.length == 0 || t.charAt(0) != "(") { return "" }
    var e = ktMatchClose(t, 0, "(", ")")
    return (e < 0 || t.substring(e + 1, e + 3) != "->") ? "" : t
}
// The arity of a receiver function type `R.(A, B) -> T`, the kotlin.jvm.functions
// class every value of it has: its parameters and the receiver.
function ktRecvFnArity(t) {
    var i = t.indexOf(".(")
    var e = i < 0 ? -1 : ktMatchClose(t, i + 1, "(", ")")
    return e < 0 ? 1 : ktTyParamCount(t, i + 1, e) + 1
}
// The parameter count of the function-type parameter list t[open..close].
function ktTyParamCount(t, open, close) {
    var n = 1
    var depth = 0
    var any = false
    for (var j = open + 1; j < close; j++) {
        var c = t.charAt(j)
        if (!ktIsWs(c)) { any = true }
        if (c == "(" || c == "<") { depth++ }
        else if (c == ")" || (c == ">" && t.charAt(j - 1) != "-")) { depth-- }
        else if (c == "," && depth == 0) { n++ }
    }
    return any ? n : 0
}
// The parameter count of the member a `::` names, from the shapes of the program's
// methods of its name (l, [min, max, names, ...] each): theirs when every one
// agrees, else -1: the receiver's class decides, from the parameter count its
// method of the name recorded on its closure. 0 is a property's and an unknown
// builtin's. A class that declares no `name` method still inherits
// Any.equals(other).
function ktRefParams(l, name) {
    if (l == undefined) { return name == "equals" ? 1 : 0 }
    var n = l[0][2].length
    for (var i = 1; i < l.length; i++) { if (l[i][2].length != n) { return 0 - 1 } }
    return n
}
// The parameter count of a builtin member a `::` names, as an arity spec: "n" on
// every receiver, followed by <row><n> pairs where a member-table row
// (ktBuiltinMembers in abnf/jsrtkotlin.go) differs - `find` is Iterable.find(pred)
// but Regex.find(input, startIndex). Measured with kotlinc 2.2.21 as `T::name is
// FunctionN` over every row and Int, Long and Double. An absent name takes no
// parameters, is a property, or is overloaded so that kotlinc rejects a reference
// to it without an expected type.
var ktBuiltinArityTab = {}
var ktBatn = ("add:1 addAll:1 addFirst:1 addLast:1 all:1 also:1 and:1 apply:1 binarySearch:3 " +
    "coerceAtLeast:1 coerceAtMost:1 commonPrefixWith:2 commonSuffixWith:2 compareTo:1 compute:2 " +
    "computeIfAbsent:2 computeIfPresent:2 contains:1 containsAll:1 containsKey:1 containsMatchIn:1 " +
    "containsValue:1 contentDeepEquals:1 contentEquals:1 copyOfRange:2 delete:2 deleteCharAt:1 " +
    "deleteRange:2 downTo:1 drop:1 dropLast:1 dropLastWhile:1 dropWhile:1 elementAt:1 " +
    "elementAtOrElse:2 elementAtOrNull:1 ensureCapacity:1 equals:1 fill:1A3 filter:1 " +
    "filterIndexed:1 filterKeys:1 filterNot:1 filterValues:1 find:1X2 findAll:2 findLast:1 floor:1 " +
    "floorDiv:1 forEach:1 forEachIndexed:1 get:1 getOrDefault:2U1 getOrElse:2U1 getOrNull:1U0 " +
    "getOrPut:2 ifBlank:1 ifEmpty:1 indexOf:1 indexOfFirst:1 indexOfLast:1 intersect:1 " +
    "joinToString:6 lastIndexOf:1 matchEntire:1 matches:1 maxWith:1 maxWithOrNull:1 merge:3 " +
    "minWith:1 minWithOrNull:1 mod:1 offer:1 offerFirst:1 offerLast:1 onEach:1 onEachIndexed:1 " +
    "onFailure:1 onSuccess:1 or:1 padEnd:2 padStart:2 partition:1 plus:1 push:1 put:2 putAll:1 " +
    "putIfAbsent:2 rangeTo:1 rangeUntil:1 reduce:1 reduceIndexed:1 reduceIndexedOrNull:1 " +
    "reduceOrNull:1 reduceRight:1 reduceRightIndexed:1 reduceRightIndexedOrNull:1 " +
    "reduceRightOrNull:1 remove:1 removeAll:1 removeAt:1 removeIf:1 removePrefix:1 removeSuffix:1 " +
    "repeat:1 replace:3 replaceAll:1 replaceFirst:2 replaceIndent:1 retainAll:1 rotateLeft:1 " +
    "rotateRight:1 runningReduce:1 set:2 setCharAt:2 setLength:1 setRange:3 setValue:3 shl:1 " +
    "shr:1 slice:1 sortedWith:1 sortWith:1 split:2 subList:2 subSequence:2 subtract:1 take:1 " +
    "takeIf:1 takeLast:1 takeLastWhile:1 takeUnless:1 takeWhile:1 toByteArray:1 toCharArray:4 " +
    "trimMargin:1 union:1 until:1 ushr:1 xor:1").split(" ")
for (var ktBati = 0; ktBati < ktBatn.length; ktBati++) {
    var ktBatc = ktBatn[ktBati].indexOf(":")
    ktBuiltinArityTab["$" + ktBatn[ktBati].substring(0, ktBatc)] = ktBatn[ktBati].substring(ktBatc + 1)
}
function ktBuiltinArity(name) {
    var s = ktBuiltinArityTab["$" + name]
    return s == undefined ? "0" : s
}
// The parameter count of a java.lang static `base::name` - M Math, I Integer and
// java.lang.Long, S System - measured with kotlinc 2.2.21 as `Math::sqrt is
// FunctionN`; -1 for any other name. An overloaded static (abs, parseInt) is absent:
// only an expected function type picks its overload (kAdoptPlan).
var ktJavaStaticTab = {}
var ktJstn = ("M.sqrt:1 M.cbrt:1 M.exp:1 M.expm1:1 M.log:1 M.log10:1 M.log1p:1 M.sin:1 M.cos:1 " +
    "M.tan:1 M.asin:1 M.acos:1 M.atan:1 M.sinh:1 M.cosh:1 M.tanh:1 M.floor:1 M.ceil:1 M.rint:1 " +
    "M.atan2:2 M.hypot:2 M.pow:2 M.toRadians:1 M.toDegrees:1 I.compare:2 I.signum:1 I.max:2 " +
    "I.min:2 I.sum:2 I.bitCount:1 I.numberOfLeadingZeros:1 I.numberOfTrailingZeros:1 " +
    "I.highestOneBit:1 I.lowestOneBit:1 I.rotateLeft:2 I.rotateRight:2 I.toHexString:1 " +
    "I.toOctalString:1 I.toBinaryString:1 S.currentTimeMillis:0 S.nanoTime:0 " +
    "S.identityHashCode:1").split(" ")
for (var ktJsti = 0; ktJsti < ktJstn.length; ktJsti++) {
    var ktJstc = ktJstn[ktJsti].indexOf(":")
    ktJavaStaticTab["$" + ktJstn[ktJsti].substring(0, ktJstc)] = ktJstn[ktJsti].charCodeAt(ktJstc + 1) - 48
}
function ktJavaStaticArity(base, name) {
    var h = ktJavaStaticCls(base)
    var n = h == "" ? undefined : ktJavaStaticTab["$" + h + "." + name]
    return n == undefined ? -1 : n
}
// The ktJavaStaticTab letter of a java.lang class reference base, "" for none.
function ktJavaStaticCls(base) {
    var jl = base.substring(0, 10) == "java.lang."
    var b = jl ? base.substring(10) : base
    return b == "Math" ? "M" : (b == "Integer" || (jl && b == "Long")) ? "I" : b == "System" ? "S" : ""
}
// What arity spec `spec` gives a receiver of member-table row `kind`. Twin of
// ktSpecArity in abnf/jsrtkotlin.go and languages/lib/kotlin-rt.metajs.
function ktSpecArity(spec, kind) {
    for (var i = 1; i + 1 < spec.length; i += 2) {
        if (spec.charAt(i) == kind) { return spec.charCodeAt(i + 1) - 48 }
    }
    return spec.charCodeAt(0) - 48
}
// The member-table row of a builtin type's values, for an unbound `Regex::find`.
function ktTypeKind(ty) {
    var b = ktBaseName(ty)
    if (b == "String") { return "T" }
    if (b == "Char") { return "C" }
    var k = ktExtKinds(b)
    return k == "" ? "" : k.charAt(0)
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
                     "$DoubleArray": "DoubleArray", "$charArrayOf": "CharArray",
                     "$booleanArrayOf": "BooleanArray", "$CharArray": "CharArray",
                     "$BooleanArray": "BooleanArray"}
function ktBuilderType(name, targs) {
    var kind = ktBuilderKind["$" + name]
    if (kind == undefined) { return "" }
    if (ktPrimArrElem["$" + kind] != undefined) { return kind }
    if (targs.length == 0) { return "" }
    return kind + "<" + targs.join(",") + ">"
}

// The JVM class of the Array an Array builder called with an explicit element type
// builds (emptyArray<Int>() is [Ljava.lang.Integer;), which its elements cannot
// tell when there are none or when the type is wider than they are; raw is the
// call's `<...>`. "" for any other call, and for an element type outside
// ktArrElemJvm, whose elements decide.
var ktArrBuilders = {"$emptyArray": true, "$arrayOf": true, "$arrayOfNulls": true, "$Array": true}
function ktArrBuilderClass(name, raw) {
    if (ktArrBuilders["$" + name] != true || raw == undefined || raw.length < 2) { return "" }
    var args = ktSplitTArgs(raw.substring(1, raw.length - 1))
    if (args.length != 1) { return "" }
    var e = ktArrElemJvm(args[0])
    return e == "" ? "" : "[" + e
}
// The JVM class a declared type `Array<T>` fixes for the array a declaration binds:
// Kotlin infers the builder's T from it (val d: Array<Animal> = arrayOf(Dog()) is
// an Array<Animal>), and Array is invariant, so any other value already has it.
// "" for any other declared type. Read once every type has been seen.
function ktDeclArrClass(dt) {
    var t = (dt.length > 0 && dt.charAt(dt.length - 1) == "?") ? dt.substring(0, dt.length - 1) : dt
    if (t.substring(0, 6) != "Array<") { return "" }
    return ktArrBuilderClass("Array", t.substring(5))
}
var ktPrimArrJvm = {"$IntArray": "[I", "$LongArray": "[J", "$ShortArray": "[S", "$ByteArray": "[B",
                    "$FloatArray": "[F", "$DoubleArray": "[D", "$CharArray": "[C", "$BooleanArray": "[Z"}
// The JVM descriptor of an array element type: L<class>; for a type the program
// declares (ktProgBin), Any, a builtin value type or a collection interface
// (ktCastLang), [<descriptor> for an Array or a primitive array (ktPrimArrJvm);
// "" for any other type (a type parameter, a projection). Read once every type
// has been seen.
function ktArrElemJvm(t) {
    var s = (t.length > 0 && t.charAt(t.length - 1) == "?") ? t.substring(0, t.length - 1) : t
    if (s.substring(0, 6) == "Array<" && s.charAt(s.length - 1) == ">") {
        var inner = ktArrElemJvm(s.substring(6, s.length - 1))
        return inner == "" ? "" : "[" + inner
    }
    var pj = ktPrimArrJvm["$" + s]
    if (pj != undefined) { return pj }
    var lt = s.indexOf("<")
    var base = lt < 0 ? s : s.substring(0, lt)
    var pb = ktProgBin(base)
    if (pb != "") { return "L" + pb + ";" }
    if (base == "Any") { return "Ljava.lang.Object;" }
    var jc = ktCastLang["$" + base]
    return jc == undefined ? "" : "L" + jc + ";"
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
// finished body holds yet. lty, when given, is a type only javaClass reads
// (ktJvmPrimSfxs) - an untyped declaration's initializer's static type, a loop or
// lambda variable's element type - since retyping by an Int or a Double would only
// add work to every assignment.
var ktAt = 0
var ktDecls = {}
var ktOpen = []
function ktNoteDecl(name, ty, lty) {
    if (name == null || name == undefined) { return }
    var l = ktDecls["$" + name]
    if (l == undefined) { l = []; ktDecls["$" + name] = l }
    var d = {ty: ty, lty: lty == undefined ? "" : lty, at: ktAt, from: -1, to: -1, vr: ktIsVarAt(ktAt)}
    l.push(d)
    ktOpen.push(d)
    return d
}
// A declaration's javaClass type (lty). A lambda parameter's is judged on first
// use, not when the lambda is walked: the lambda's receiver text may read `this`
// of an extension function whose receiver is noted only once its body is walked.
function ktDeclLty(d) {
    if (d.lfn !== undefined) {
        var f = d.lfn
        d.lfn = undefined
        var saved = ktAt
        ktAt = d.at
        d.lty = f()
        ktAt = saved
    }
    return d.lty
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
// A lambda's parameter `name` declared at offset `at`, a val; the finished lambda
// body (ktNoteBody) then scopes it. A lambda with no parameter list (items, its
// parts, carry no lparams) declares the implicit `it` at its own offset.
// Either is typed (lty) for javaClass as far as ktLamOneParamTy / ktLamParamTy tell.
function ktNoteLamParam(name, at) {
    var saved = ktAt
    ktAt = at
    var d = ktNoteDecl(name, "", "")
    d.lfn = function() { return ktLamOneParamTy(at, name.length) }
    ktAt = saved
}
function ktNoteLamIt(items, at) {
    for (var i = 0; i < items.length; i++) {
        if (items[i] != undefined && items[i].lparams != undefined) { return }
    }
    var saved = ktAt
    ktAt = at
    var d = ktNoteDecl("it", "", "")
    d.lfn = function() { return ktLamParamTy(at, -1) }
    ktAt = saved
}
// The offsets of the words `it` walked so far that no lambda has claimed.
var ktItWords = []
function ktNoteIt(at) { ktItWords.push(at) }
// Whether the lambda at [from, to), parts items, reads its implicit `it`, which
// makes its arity 1 (Function1) rather than 0. A lambda without a parameter list
// claims every unclaimed `it` inside it; a nested one was walked first and has
// claimed its own, and one with a parameter list claims none, so an `it` it reads
// belongs to the lambda around it - unless it names a parameter `it`, which every
// `it` inside it reads instead.
function ktLamClaimsIt(items, from, to) {
    for (var i = 0; i < items.length; i++) {
        var lps = items[i] == undefined ? undefined : items[i].lparams
        if (lps != undefined) {
            if (lps.indexOf("it") >= 0) { ktLamTakeIt(from, to) }
            return false
        }
    }
    return ktLamTakeIt(from, to)
}
// Removes the unclaimed `it` words in [from, to); whether there were any.
function ktLamTakeIt(from, to) {
    var keep = []
    var hit = false
    for (var j = 0; j < ktItWords.length; j++) {
        if (ktItWords[j] >= from && ktItWords[j] < to) { hit = true } else { keep.push(ktItWords[j]) }
    }
    ktItWords = keep
    return hit
}
// The static type of the lambda parameter at offset at (its name n characters
// long): its written type, else ktLamParamTy's for the lambda's only parameter or
// either of exactly two (`{ i, e -> ... }`).
function ktLamOneParamTy(at, n) {
    var w0 = at > 80 ? at - 80 : 0
    var pre = c.srcSlice(w0, at)
    var post = c.srcSlice(at + n, at + n + 200)
    var k = pre.length
    while (k > 0 && ktIsWs(pre.charAt(k - 1))) { k-- }
    var pi = -1
    if (k > 0 && pre.charAt(k - 1) == ",") {
        // The second parameter: a plain name before it, the `{` before that.
        k--
        while (k > 0 && ktIsWs(pre.charAt(k - 1))) { k-- }
        var e = k
        while (k > 0 && ktIdChar(pre.charAt(k - 1))) { k-- }
        while (k > 0 && ktIsWs(pre.charAt(k - 1))) { k-- }
        if (k == e) { return "" }
        pi = 1
    }
    if (k == 0 || pre.charAt(k - 1) != "{") { return "" }
    var i = ktSkipWs(post, 0)
    var arrow = post.indexOf("->")
    var comma = post.indexOf(",")
    if (arrow < 0) { return "" }
    if (comma >= 0 && comma < arrow) {
        // The first of two: the second is a plain name or a destructuring pattern.
        var j = ktSkipWs(post, comma + 1)
        if (post.charAt(j) == "(") { j = ktMatchClose(post, j, "(", ")") + 1 }
        while (j > 0 && j < post.length && ktIdChar(post.charAt(j))) { j++ }
        if (pi >= 0 || i != comma || ktSkipWs(post, j) != arrow) { return "" }
        pi = 0
    }
    if (post.charAt(i) == ":") { return pi < 0 ? stripWs(post.substring(i + 1, arrow)) : "" }
    return (i == arrow || pi == 0) ? ktLamParamTy(w0 + k - 1, pi) : ""
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
// `val`: `this`, a local or parameter (a lambda's too), a top-level property, or a
// property of the class the path's owner has (by name alone when that owner is not
// known here). Kotlin
// resolves `a += b` on a val as a.plusAssign(b), in place on a mutable collection,
// and on a var whose type has no plusAssign as a = a + b (a var of a MUTABLE
// collection type is ambiguous and rejected, so only a read-only type reaches the
// plus form). Read when every declaration has been seen: a member or a top-level
// property may be declared after its use.
function ktAssignIsVal(ref, at) {
    var n = ref.path.length
    if (n == 0) {
        if (ref.name == "this") { return true }
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
// one that inherits the property, whether every property of that name is. A
// property no class here declares belongs to a library type - Map.keys / values /
// entries - and those are vals, so `m.keys -= k` is minusAssign (a remove).
function ktMemberIsVal(cls, name) {
    var l = ktMemberVars["$" + name]
    if (l == undefined) { return true }
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
// An operator with an operand that is a name DECLARED with a type other than Float
// or Double is "==:" / "!=:", Any.equals: Kotlin's == is IEEE 754 only when both
// operands are statically floating point, so through Any NaN equals NaN and -0.0
// is not 0.0.
// The operands are read off the source (from..to); a text the scan cannot split is
// treated as having a literal.
function ktEqOps(items, from, to) {
    if (items.length < 3) { return items }
    var opnds = ktEqOperands(c.srcSlice(from, to))
    if (opnds == null) { return items }
    var typed = opnds.length * 2 - 1 == items.length
    var out = []
    var k = 0
    for (var i = 0; i < items.length; i++) {
        var it = items[i]
        if (it === "==" || it === "!=") {
            var byEq = typed && (ktEqNonFlo(opnds[k], from) || ktEqNonFlo(opnds[k + 1], from))
            if (it === "==") { out.push(byEq ? "==:" : "==.") } else { out.push(byEq ? "!=:" : "!=.") }
        } else { out.push(it) }
        if (i % 2 == 1) { k++ }
    }
    return out
}
// Whether an operand's text is a name declared with a type other than Float or
// Double, nullable or not.
function ktEqNonFlo(t, at) {
    var s = ""
    for (var i = 0; i < t.length; i++) {
        var ch = t.charAt(i)
        if (ch != " " && ch != "\t" && ch != "\n" && ch != "\r") { s = s + ch }
    }
    if (s.length == 0 || (s.charAt(0) >= "0" && s.charAt(0) <= "9")) { return false }
    for (var j = 0; j < s.length; j++) { if (!ktIdChar(s.charAt(j))) { return false } }
    var saved = ktAt
    ktAt = at
    var d = ktDeclAt(s)
    ktAt = saved
    if (d == null || d.ty == "") { return false }
    var b = ktBaseName(d.ty)
    if (b == "Double" || b == "Float") { return false }
    return !ktFloSmartCast(s, c.srcSlice(d.at, at))
}
// Whether the source between a declaration and a use (whitespace dropped) tests
// the name with `is Double` / `is Float`, directly or as a `when` subject: a smart
// cast makes the operand statically floating point again, and the text cannot tell
// the scope of one, so any such test keeps IEEE equality.
function ktFloSmartCast(name, raw) {
    var t = ""
    for (var i = 0; i < raw.length; i++) {
        var ch = raw.charAt(i)
        if (ch != " " && ch != "\t" && ch != "\n" && ch != "\r") { t = t + ch }
    }
    var subj = t.indexOf("when(" + name + ")") >= 0
    var tests = ["isDouble", "isFloat"]
    for (var k = 0; k < tests.length; k++) {
        var at = t.indexOf(tests[k])
        while (at >= 0) {
            var from = at - name.length
            if (subj || (from >= 0 && t.substring(from, at) == name && (from == 0 || !ktIdChar(t.charAt(from - 1))))) { return true }
            at = t.indexOf(tests[k], at + 1)
        }
    }
    return false
}
// The operand texts of an equality chain, or null when an operand is an integer
// literal or the text holds a string, char or comment the scan does not split.
function ktEqOperands(raw) {
    var out = []
    var depth = 0
    var from = 0
    var i = 0
    while (i < raw.length) {
        var ch = raw.charAt(i)
        if (ch == "\"" || ch == "'" || (ch == "/" && (raw.charAt(i + 1) == "/" || raw.charAt(i + 1) == "*"))) { return null }
        if (ch == "(" || ch == "[" || ch == "{") { depth++ }
        else if (ch == ")" || ch == "]" || ch == "}") { depth-- }
        else if (depth == 0 && (ch == "=" || ch == "!") && raw.charAt(i + 1) == "=") {
            var prev = i > 0 ? raw.charAt(i - 1) : ""
            if (ch == "=" && (prev == "<" || prev == ">" || prev == "!" || prev == "=")) { i++; continue }
            var o = raw.substring(from, i)
            if (ktIsIntLitText(o)) { return null }
            out.push(o)
            i = i + 2
            if (raw.charAt(i) == "=") { i++ }
            from = i
            continue
        }
        i++
    }
    if (out.length == 0) { return null }
    var last = raw.substring(from)
    if (ktIsIntLitText(last)) { return null }
    out.push(last)
    return out
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
// lty, when given, is a type only javaClass reads, as for ktNoteDecl.
var ktMemberLtys = {}
function ktNoteMember(name, ty, lty) {
    if (name == null || name == undefined) { return }
    var kl = ktMemberVars["$" + name]
    if (kl == undefined) { kl = []; ktMemberVars["$" + name] = kl }
    kl.push({vr: ktIsVarAt(ktAt), at: ktAt})
    if (lty != undefined && lty != "") {
        var ll = ktMemberLtys["$" + name]
        if (ll == undefined) { ll = []; ktMemberLtys["$" + name] = ll }
        ll.push({ty: lty, at: ktAt})
    }
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
    ktNoteSupers(name, c.srcSlice(from, to))
}
// The supertypes a type declaration's header names (`class B : A(), I by d`), by
// simple name, which ktChainFunRet searches for an inherited method.
var ktTypeSupers = {}
function ktNoteSupers(name, raw) {
    var i = raw.indexOf(name)
    if (i < 0) { return }
    i = i + name.length
    var depth = 0
    var colon = -1
    while (i < raw.length) {
        var ch = raw.charAt(i)
        if (ch == "\"" || ch == "'") {
            var q = ktQuoteEnd(raw, i)
            if (q < 0) { return }
            i = q + 1
            continue
        }
        if (ch == "(" || ch == "[" || ch == "<") { depth++ }
        else if (ch == ")" || ch == "]" || (ch == ">" && raw.charAt(i - 1) != "-")) { depth-- }
        else if (depth == 0 && ch == "{") { break }
        else if (depth == 0 && ch == ":" && colon < 0) { colon = i }
        i++
    }
    if (colon < 0) { return }
    var items = ktSplitTop(raw.substring(colon + 1, i), ",")
    var out = []
    for (var k = 0; k < items.length; k++) {
        var t = ktTrimWs(items[k])
        var e = 0
        while (e < t.length && (ktIdChar(t.charAt(e)) || t.charAt(e) == ".")) { e++ }
        var sup = t.substring(t.lastIndexOf(".", e) + 1, e)
        if (sup != "") { out.push(sup) }
    }
    ktTypeSupers["$" + name] = out
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
// The builtin receivers an untested extension type can bind as an IMPLICIT receiver,
// as member-table rows (ktBuiltinMembers in abnf/jsrtkotlin.go): `fun
// MutableList<Int>.f()` does not take a listOf inside `with`. "" is any receiver (a
// type parameter, Any).
var ktExtKindTab = {"$MutableList": "LD", "$ArrayList": "LD", "$List": "LRD", "$MutableCollection": "LDS",
    "$Collection": "LRDS", "$MutableIterable": "LDS", "$Iterable": "LRDSG", "$MutableSet": "S", "$Set": "S",
    "$HashSet": "S", "$LinkedHashSet": "S", "$MutableMap": "M", "$Map": "M", "$HashMap": "M",
    "$LinkedHashMap": "M", "$Array": "A", "$IntArray": "A", "$LongArray": "A", "$DoubleArray": "A",
    "$FloatArray": "A", "$ShortArray": "A", "$ByteArray": "A", "$CharArray": "A", "$BooleanArray": "A",
    "$Sequence": "Q", "$IntRange": "G", "$LongRange": "G", "$CharRange": "G", "$IntProgression": "G",
    "$ArrayDeque": "D", "$StringBuilder": "B", "$CharSequence": "BT", "$Regex": "X", "$MatchResult": "H",
    "$Result": "U"}
function ktExtKinds(ty) {
    var k = ktExtKindTab["$" + ktBaseName(ty)]
    return k == undefined ? "" : k
}
// The interfaces, objects and enum classes the program declares (the classes are
// ktClassExt), each noted from its declaration's text after keyword kw
// (`interfaceShape{...}`, `objectOrigin{...}`, `enumclassColor{...}`).
var ktDeclTypeNames = {}
var ktTypeExt = {}
var ktTypeExtNames = []
function ktNoteTypeName(txt, kw, from, to) {
    var i = txt.indexOf(kw)
    if (i < 0) { return }
    var j = i + kw.length
    while (j < txt.length && ktIdChar(txt.charAt(j))) { j++ }
    if (j == i + kw.length) { return }
    var name = txt.substring(i + kw.length, j)
    ktDeclTypeNames["$" + name] = true
    if (ktTypeExt["$" + name] == undefined) { ktTypeExtNames.push(name) }
    ktTypeExt["$" + name] = {from: from, to: to}
    ktNoteSupers(name, c.srcSlice(from, to))
}
// The JVM binary name of a type the program declares, written t (`Inner`,
// `Outer.Inner`): its enclosing type declarations and itself joined by `$`
// (Outer$Inner), read off the declarations' extents; "" for any other type.
function ktProgBin(t) {
    var segs = t.split(".")
    var e = ktDeclExt(segs[0])
    if (e == undefined) { return "" }
    var out = segs.join("$")
    var outer = ktOuterDecl(e)
    while (outer != "") {
        out = outer + "$" + out
        outer = ktOuterDecl(ktDeclExt(outer))
    }
    return out
}
function ktDeclExt(name) {
    var e = ktClassExt["$" + name]
    return e != undefined ? e : ktTypeExt["$" + name]
}
// The innermost type declaration whose extent strictly holds extent e, or "".
function ktOuterDecl(e) {
    var best = ""
    var size = 0
    var n = ktClassNames.length
    for (var i = 0; i < n + ktTypeExtNames.length; i++) {
        var name = i < n ? ktClassNames[i] : ktTypeExtNames[i - n]
        var o = ktDeclExt(name)
        if (o.from <= e.from && e.to <= o.to && o.to - o.from > e.to - e.from &&
            (best == "" || o.to - o.from < size)) { best = name; size = o.to - o.from }
    }
    return best
}
// The declared type of property `name` of class `cls`; for a class not known here,
// the name's type when every declaration of it agrees, else "".
function ktMemberTypeIn(cls, name) { return ktMemberTyOf(ktMemberTys["$" + name], cls) }
function ktMemberTyOf(l, cls) {
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
// declared generic twice answers nothing. Given the declaration's offset `at`, an
// extension's receiver type is also noted as the type of `this` there, for the
// body's static types; the caller notes the body after it, which scopes it.
var ktGenerics = {}
function ktNoteGeneric(raw, kw, at) {
    var i = raw.indexOf(kw)
    if (i < 0) { return }
    i = ktSkipWs(raw, i + kw.length)
    var tps = []
    var rf = false
    if (raw.charAt(i) == "<") {
        var e = ktMatchClose(raw, i, "<", ">")
        if (e < 0) { return }
        tps = ktTParamNames(raw.substring(i + 1, e))
        rf = raw.substring(i + 1, e).indexOf("reified") >= 0
        i = ktSkipWs(raw, e + 1)
    }
    var n0 = i
    while (i < raw.length && ktIdChar(raw.charAt(i))) { i++ }
    // An extension's receiver type (`T.`, `List<T>.`, `T?.`) comes before its name.
    var recv = ""
    var r0 = n0
    var r = i
    while (kw == "fun" && r > 0) {
        r = ktSkipWs(raw, r)
        if (raw.charAt(r) == "<") { r = ktMatchClose(raw, r, "<", ">") + 1 }
        if (r > 0 && raw.charAt(r) == "?") { r++ }
        if (r <= 0 || raw.charAt(r) != ".") { break }
        recv = ktTrimWs(raw.substring(r0, r))
        n0 = r + 1
        i = n0
        while (i < raw.length && ktIdChar(raw.charAt(i))) { i++ }
        r = i
    }
    var name = raw.substring(n0, i)
    if (recv != "" && at != undefined) {
        var saved = ktAt
        ktAt = at
        ktNoteDecl("this", "", stripWs(recv))
        ktAt = saved
    }
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
    // An extension is filed apart (under ".name"): only a call in method position
    // can reach it.
    var key = (recv == "" ? "$" : ".") + name
    ktGenerics[key] = ktGenerics[key] == undefined ? {tps: tps, ptys: ptys, rf: rf, recv: recv} : "#"
}
// The type arguments a call of the reified generic function `name` written without
// any infers, one per type parameter, or null for any other callee: the static type
// (ktStaticTy) of the first positional argument whose parameter is declared as that
// type parameter (T or T?), Any when none settles it. The call's source is at..to.
function ktInferTArgs(name, at, to) {
    var g = ktGenerics["$" + name]
    if (g == undefined || g == "#" || g.rf != true) { return null }
    var raw = c.srcSlice(at, to)
    var i = raw.indexOf("(")
    var args = []
    if (i >= 0 && i == ktSkipWs(raw, name.length)) {
        var e = ktMatchClose(raw, i, "(", ")")
        if (e < 0) { return null }
        args = ktSplitTop(raw.substring(i + 1, e), ",")
    }
    var out = []
    var saved = ktAt
    ktAt = at + (i < 0 ? 0 : i)
    for (var k = 0; k < g.tps.length; k++) {
        var ty = ""
        for (var p = 0; p < g.ptys.length && p < args.length && ty == ""; p++) {
            var pt = g.ptys[p]
            if (pt != g.tps[k] && pt != g.tps[k] + "?") { continue }
            // A named argument may sit in any slot.
            var a = ktTrimWs(args[p])
            var n = 0
            while (n < a.length && ktIdChar(a.charAt(n))) { n++ }
            var eq = ktSkipWs(a, n)
            if (n > 0 && a.charAt(eq) == "=" && a.charAt(eq + 1) != "=") { break }
            ty = ktStaticTy(a)
            if (pt != g.tps[k] && ty.charAt(ty.length - 1) == "?") { ty = ty.substring(0, ty.length - 1) }
        }
        out.push(ty == "" ? "Any" : ty)
    }
    ktAt = saved
    return out
}
// The type arguments a method-position call of the reified generic extension
// `name` written without any infers, one per type parameter, or null for any other
// callee: what the static type of the receiver (the source at..to) binds the type
// parameter to in the declared receiver type (`T`, `List<T>`), Any when it binds none.
function ktInferExtTArgs(name, at, to) {
    var g = ktGenerics["." + name]
    if (g == undefined || g == "#" || g.rf != true) { return null }
    var saved = ktAt
    ktAt = at
    var ty = ktStaticTy(c.srcSlice(at, to))
    ktAt = saved
    // A non-null receiver type is reached on a nullable value only through `?.`.
    if (g.recv.charAt(g.recv.length - 1) != "?" && ty.charAt(ty.length - 1) == "?") { ty = ty.substring(0, ty.length - 1) }
    var out = []
    for (var k = 0; k < g.tps.length; k++) {
        var b = ty == "" ? "" : ktBindTParam(g.recv, ty, g.tps[k])
        out.push(b == "" ? "Any" : b)
    }
    return out
}
// The type the type parameter tp stands for when the type text pat (written with it)
// is the type ty, "" when pat does not mention it where ty spells one.
function ktBindTParam(pat, ty, tp) {
    var p = ktTrimWs(pat)
    var t = ktTrimWs(ty)
    var w = ["out ", "in "]
    for (var m = 0; m < w.length; m++) {
        if (p.substring(0, w[m].length) == w[m]) { p = ktTrimWs(p.substring(w[m].length)) }
    }
    if (p.charAt(p.length - 1) == "?") {
        p = p.substring(0, p.length - 1)
        if (t.charAt(t.length - 1) == "?") { t = t.substring(0, t.length - 1) }
    }
    if (p == tp) { return t }
    var lp = p.indexOf("<")
    var lt = t.indexOf("<")
    if (lp < 0 || lt < 0 || p.charAt(p.length - 1) != ">" || t.charAt(t.length - 1) != ">") { return "" }
    var pa = ktSplitTop(p.substring(lp + 1, p.length - 1), ",")
    var ta = ktSplitTop(t.substring(lt + 1, t.length - 1), ",")
    if (pa.length != ta.length) { return "" }
    for (var i = 0; i < pa.length; i++) {
        var b = ktBindTParam(pa[i], ta[i], tp)
        if (b != "") { return b }
    }
    return ""
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
    var end = i
    while (i < s.length && (s.charAt(i) == "(" || s.charAt(i) == "{")) {
        var e = (s.charAt(i) == "(") ? ktMatchClose(s, i, "(", ")") : ktMatchClose(s, i, "{", "}")
        if (e < 0) { return -1 }
        end = e + 1
        i = ktSkipWs(s, end)
        groups++
    }
    return groups == 0 ? -1 : end
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
// The type name a type argument's text names, without its own type arguments and
// nullability: List<Int> is List, C? is C.
function ktTArgBase(t) {
    var lt = t.indexOf("<")
    var s = lt < 0 ? t : t.substring(0, lt)
    return (s.length > 0 && s.charAt(s.length - 1) == "?") ? s.substring(0, s.length - 1) : s
}
// The end of the number, char or boolean literal `t` starts with, or -1. A number
// runs over letters, digits and `_` (5L, 0x1F, 1.5f, 2u), a `.` only before a digit
// (`5.javaClass` is 5 then a member) and a sign only after a decimal exponent.
function ktLitEnd(t) {
    var c0 = t.charAt(0)
    if (c0 == "'") {
        var q = ktQuoteEnd(t, 0)
        return q < 0 ? -1 : q + 1
    }
    var i = 0
    if (!(c0 >= "0" && c0 <= "9")) {
        while (i < t.length && ktIdChar(t.charAt(i))) { i++ }
        var w = t.substring(0, i)
        return (w == "true" || w == "false") ? i : -1
    }
    var hex = t.length > 1 && (t.charAt(1) == "x" || t.charAt(1) == "X")
    while (i < t.length) {
        var c = t.charAt(i)
        var p = i > 0 ? t.charAt(i - 1) : ""
        var d1 = t.charAt(i + 1)
        if (ktIdChar(c)) { i++ }
        else if (c == "." && d1 >= "0" && d1 <= "9") { i++ }
        else if ((c == "+" || c == "-") && !hex && (p == "e" || p == "E")) { i++ }
        else { break }
    }
    return i
}
// The static type of a literal's text: Int, Long, Double, Float, Char or Boolean,
// "" for an unsigned literal and any other text.
function ktLitType(t) {
    var s = (t.length > 1 && t.charAt(0) == "-") ? t.substring(1) : t
    if (s.length == 0 || ktLitEnd(s) != s.length) { return "" }
    var c0 = s.charAt(0)
    if (c0 == "'") { return "Char" }
    if (c0 == "t" || c0 == "f") { return "Boolean" }
    var last = s.charAt(s.length - 1)
    if (s.indexOf("u") >= 0 || s.indexOf("U") >= 0) { return "" }
    var radix = s.length > 1 && (s.charAt(1) == "x" || s.charAt(1) == "X" || s.charAt(1) == "b" || s.charAt(1) == "B")
    if (!radix && (last == "f" || last == "F")) { return "Float" }
    if (!radix && (s.indexOf(".") >= 0 || s.indexOf("e") >= 0 || s.indexOf("E") >= 0)) { return "Double" }
    return ktIsLongLit(s) ? "Long" : "Int"
}
var ktJvmPrims = {"$Int": true, "$Long": true, "$Short": true, "$Byte": true, "$Double": true,
                  "$Float": true, "$Char": true, "$Boolean": true}
// kotlinc's javaClass of a receiver whose STATIC type is a primitive is the
// primitive class (`int`), while a value typed Any, a nullable or a generic one is
// boxed (java.lang.Integer). Each non-safe `.javaClass` step of a postfix chain
// whose text is txt gets its receiver's text (jrecv) and offset (jat) here, at walk
// time; ktJvmPrimSfxs judges them (jprim) once every declaration is known. Whether
// any was noted.
function ktNoteJavaClassRecvs(sfx, txt) {
    if (txt == undefined) { return false }
    var cuts = []
    var i = 0
    while (i < txt.length) {
        var ch = txt.charAt(i)
        if (ch == "\"" || ch == "'") {
            var q = ktQuoteEnd(txt, i)
            if (q < 0) { return false }
            i = q + 1
        } else if (ch == "(" || ch == "[" || ch == "{") {
            var e = ktMatchClose(txt, i, ch, ch == "(" ? ")" : (ch == "[" ? "]" : "}"))
            if (e < 0) { return false }
            i = e + 1
        } else {
            var n = ch == "." ? ktSkipWs(txt, i + 1) : i
            if (n > i && txt.substring(n, n + 9) == "javaClass" && !ktIdChar(txt.charAt(n + 9))) { cuts.push(i) }
            i++
        }
    }
    var ci = 0
    var any = false
    for (var k = 0; k < sfx.length && ci < cuts.length; k++) {
        var s = sfx[k]
        if (s.kind != "field" || s.name != "javaClass") { continue }
        if (s.safe != true) {
            s.jrecv = txt.substring(0, cuts[ci])
            s.jat = ktAt
            any = true
        }
        ci++
    }
    return any
}
// Whether a postfix chain's text (whitespace stripped) LEADS with an unqualified
// `javaClass` that nothing declares: the extension property read off the implicit
// receiver, which each grammar's foldCallMember rewrites to `this.javaClass` so its
// primitive receiver is judged like any other (`fun Int.f() = javaClass.name` is int).
function ktImplicitJavaClass(txt) {
    if (txt == undefined || txt.substring(0, 9) != "javaClass") { return false }
    var n = txt.charAt(9)
    if (ktIdChar(n) || n == "(" || n == "{" || n == "<") { return false }
    return ktDeclAt("javaClass") == null
}
function ktJvmPrimSfxs(sfx) {
    var saved = ktAt
    for (var k = 0; k < sfx.length; k++) {
        var s = sfx[k]
        if (s.jrecv === undefined) { continue }
        ktAt = s.jat
        s.jprim = ktJvmPrims["$" + ktJvmTy(s.jrecv)] == true
    }
    ktAt = saved
}
// The type of the value an expression's text leaves on the JVM stack, where it
// differs from the static type (ktStaticTy): a cast (`as Int`), a `!!`, an
// element read off a list or an Array and a generic element result
// (`xs.first()`, ktElemFns, unless a String, a primitive array or a range
// declares its own) leave a BOXED value, while an arithmetic result, a local, a
// property or a function result of a primitive type is unboxed.
function ktJvmTy(t) {
    var s = ktTrimWs(t)
    while (s.charAt(0) == "(" && ktMatchClose(s, 0, "(", ")") == s.length - 1) { s = ktTrimWs(s.substring(1, s.length - 1)) }
    var ops = ktTopOps(s)
    if (ops.length == 1 && ops[0].op == "as") { return "" }
    if (ops.length == 0 && s.substring(s.length - 2) == "!!") { return "" }
    var lc = ops.length == 0 ? ktLastCall(s) : null
    if (lc !== null && ktElemFns["$" + lc[0]] == true) {
        var rb = lc[1].charAt(lc[1].length - 1) == "?" ? "" : ktBaseName(ktStaticTy(lc[1]))
        if (rb != "String" && ktPrimArrElem["$" + rb] == undefined && rb.indexOf("Range") <= 0 &&
            rb.indexOf("Progression") <= 0) { return "" }
    }
    if (ops.length == 0 && s.charAt(s.length - 1) == "]") {
        var lb = ktMatchOpen(s, s.length - 1, "[", "]")
        var cb = ktBaseName(ktStaticTy(s.substring(0, lb)))
        if (cb != "String" && ktPrimArrElem["$" + cb] == undefined) { return "" }
    }
    return ktStaticTy(s)
}

// ----- The static type of an expression's text -----
// What a text alone settles, read at ktAt: a literal, a declared name, a
// constructor call, a declared function's result (ktFunRet), a builder call, a
// property chain off those, an index into a typed list or array, numeric
// promotion across + - * / %, a comparison or logical operator (Boolean), a range
// and `as`; "" for anything else. Whitespace may be present, and an infix word
// (until, downTo, step, as, shl...) is only seen where it is.
function ktIsWs(c) { return c == " " || c == "\t" || c == "\n" || c == "\r" }
function ktTrimWs(t) {
    var b = ktSkipWs(t, 0)
    var e = t.length
    while (e > b && ktIsWs(t.charAt(e - 1))) { e-- }
    return t.substring(b, e)
}
function ktStaticTy(t) {
    var s = ktTrimWs(t)
    if (s.length == 0) { return "" }
    if (s.charAt(0) == "(" && ktMatchClose(s, 0, "(", ")") == s.length - 1) { return ktStaticTy(s.substring(1, s.length - 1)) }
    if (s.substring(0, 2) == "if" && ktSkipWs(s, 2) < s.length && s.charAt(ktSkipWs(s, 2)) == "(") { return ktIfTy(s) }
    if (s.substring(0, 4) == "when" && !ktIdChar(s.charAt(4))) { return ktWhenTy(s) }
    var ops = ktTopOps(s)
    if (ops.length == 0) { return ktChainTy(s) }
    for (var i = 0; i < ops.length; i++) {
        if (ops[i].op == "#" || ops[i].op == "?:") { return "" }
    }
    for (var b = 0; b < ops.length; b++) {
        if (ktBoolOps["$" + ops[b].op] == true) { return "Boolean" }
    }
    // The loosest operator splits first: an infix word, then a range, then the
    // arithmetic, each left-associative.
    var w = -1
    var r = -1
    var ar = false
    for (var j = 0; j < ops.length; j++) {
        var o = ops[j].op
        if (ktIdChar(o.charAt(0)) && !ktPrefixOp(o)) { w = j }
        else if (o == ".." || o == "..<") { r = j }
        else if (!ktPrefixOp(o)) { ar = true }
    }
    if (w >= 0) {
        var wo = ops[w].op
        var lt = ktStaticTy(s.substring(0, ops[w].at))
        var rt = s.substring(ops[w].at + wo.length)
        if (wo == "as") { return ops.length == 1 ? stripWs(rt) : "" }
        if (wo == "to") {
            var rtt = ktStaticTy(rt)
            return (lt != "" && rtt != "") ? "Pair<" + lt + "," + rtt + ">" : "Pair"
        }
        if (wo == "until") { return ktRangeTy(lt, ktStaticTy(rt), "Range") }
        if (wo == "downTo") { return ktRangeTy(lt, ktStaticTy(rt), "Progression") }
        if (wo == "step") {
            var el = ktElemOfTy(lt)
            return (lt == el + "Range" || lt == el + "Progression") ? el + "Progression" : ""
        }
        if (ktBitWords["$" + wo] == true) {
            var bt = ktNumPromote(lt, ktStaticTy(rt))
            return (bt == "Int" || bt == "Long") ? bt : ""
        }
        return ""
    }
    if (r >= 0) { return ktRangeTy(ktStaticTy(s.substring(0, ops[r].at)), ktStaticTy(s.substring(ops[r].at + ops[r].op.length)), "Range") }
    if (ar) {
        var ty = ""
        var from = 0
        var prev = ""
        for (var k = 0; k <= ops.length; k++) {
            if (k < ops.length && ktPrefixOp(ops[k].op)) { continue }
            var to = k < ops.length ? ops[k].at : s.length
            var ot = ktStaticTy(s.substring(from, to))
            ty = prev == "" ? ot : ktArithTy(ty, prev, ot)
            if (ty == "") { return "" }
            if (k < ops.length) { prev = ops[k].op; from = to + prev.length }
        }
        return ty
    }
    // Only prefix operators are left, and the first one applies to the rest: `!` is
    // Boolean.not, a sign keeps a numeric type (Short and Byte promote to Int).
    if (ops[0].op == "!") { return "Boolean" }
    return ktNumPromote(ktStaticTy(s.substring(ops[0].at + 1)), "Int")
}
function ktPrefixOp(o) { return o == "u-" || o == "u+" || o == "!" }
// `if (c) a else b` as an expression: the branches' one type, a braced branch
// read as its single expression; "" when they differ.
function ktIfTy(s) {
    var ce = ktMatchClose(s, ktSkipWs(s, 2), "(", ")")
    if (ce < 0) { return "" }
    var i = ktSkipWs(s, ce + 1)
    var a = ktBranchEnd(s, i)
    var e = ktSkipWs(s, a)
    if (a < 0 || s.substring(e, e + 4) != "else" || ktIdChar(s.charAt(e + 4))) { return "" }
    var b = ktSkipWs(s, e + 4)
    if (ktBranchEnd(s, b) != s.length) { return "" }
    var ta = ktStaticTy(ktUnbrace(s.substring(i, a)))
    return ta == ktStaticTy(ktUnbrace(s.substring(b))) ? ta : ""
}
// The type of a `when` expression whose every arm's value has the same static type,
// "" else. An arm ends at a line break or `;`, or before the word `else` of the last
// arm (`when (it) { 1 -> "a" else -> "b" }`); an arm the cut leaves without its
// `->`, or whose value is not one expression, is not told.
function ktWhenTy(s) {
    var i = ktSkipWs(s, 4)
    if (s.charAt(i) == "(") {
        var ce = ktMatchClose(s, i, "(", ")")
        if (ce < 0) { return "" }
        i = ktSkipWs(s, ce + 1)
    }
    if (s.charAt(i) != "{" || ktMatchClose(s, i, "{", "}") != s.length - 1) { return "" }
    var body = s.substring(i + 1, s.length - 1)
    var arms = []
    var from = 0
    var depth = 0
    var k = 0
    while (k < body.length) {
        var ch = body.charAt(k)
        if (ch == "\"" || ch == "'") {
            var q = ktQuoteEnd(body, k)
            if (q < 0) { return "" }
            k = q + 1
            continue
        }
        if (ch == "(" || ch == "[" || ch == "{") { depth++ }
        else if (ch == ")" || ch == "]" || ch == "}") { depth-- }
        else if (depth == 0 && (ch == "\n" || ch == ";" ||
                 (ktIsWs(ch) && body.substring(k + 1, k + 5) == "else" && !ktIdChar(body.charAt(k + 5))))) {
            arms.push(body.substring(from, k))
            from = k + 1
        }
        k++
    }
    arms.push(body.substring(from))
    var ty = ""
    for (var a = 0; a < arms.length; a++) {
        var t = ktTrimWs(arms[a])
        if (t == "") { continue }
        var ar = ktTopArrow(t)
        if (ar < 0) { return "" }
        var at = ktStaticTy(ktUnbrace(t.substring(ar + 2)))
        if (at == "" || (ty != "" && at != ty)) { return "" }
        ty = at
    }
    return ty
}
// Whether t is one plain identifier.
function ktIsIdent(t) {
    if (t.length == 0 || (t.charAt(0) >= "0" && t.charAt(0) <= "9")) { return false }
    for (var i = 0; i < t.length; i++) { if (!ktIdChar(t.charAt(i))) { return false } }
    return true
}
// The first `->` of t outside brackets and quotes, or -1.
function ktTopArrow(t) {
    var depth = 0
    var k = 0
    while (k < t.length) {
        var ch = t.charAt(k)
        if (ch == "\"" || ch == "'") {
            var q = ktQuoteEnd(t, k)
            if (q < 0) { return -1 }
            k = q + 1
            continue
        }
        if (ch == "(" || ch == "[" || ch == "{") { depth++ }
        else if (ch == ")" || ch == "]" || ch == "}") { depth-- }
        else if (depth == 0 && ch == "-" && t.charAt(k + 1) == ">") { return k }
        k++
    }
    return -1
}
// Where an if-branch starting at i ends: its braces' close, else the word `else`
// at bracket depth 0 (or the text's end); -1 when it cannot be told.
function ktBranchEnd(s, i) {
    if (s.charAt(i) == "{") {
        var e = ktMatchClose(s, i, "{", "}")
        return e < 0 ? -1 : e + 1
    }
    var depth = 0
    var k = i
    while (k < s.length) {
        var ch = s.charAt(k)
        if (ch == "\"" || ch == "'") {
            var q = ktQuoteEnd(s, k)
            if (q < 0) { return -1 }
            k = q + 1
            continue
        }
        if (ch == "(" || ch == "[" || ch == "{") { depth++ }
        else if (ch == ")" || ch == "]" || ch == "}") { depth-- }
        else if (depth == 0 && ktIsWs(ch) && s.substring(k + 1, k + 5) == "else" && !ktIdChar(s.charAt(k + 5))) { return k }
        k++
    }
    return k
}
function ktUnbrace(t) {
    var s = ktTrimWs(t)
    if (s.charAt(0) != "{") { return s }
    var inner = s.substring(1, s.length - 1)
    return (inner.indexOf(";") >= 0 || ktTrimWs(inner).indexOf("\n") >= 0) ? "" : inner
}
var ktBoolOps = {"$==": true, "$!=": true, "$===": true, "$!==": true, "$&&": true, "$||": true,
                 "$<": true, "$>": true, "$<=": true, "$>=": true, "$is": true, "$!is": true,
                 "$in": true, "$!in": true}
var ktBitWords = {"$and": true, "$or": true, "$xor": true, "$shl": true, "$shr": true, "$ushr": true}
// The result type of two numeric operands' arithmetic: Double over Float over Long
// over Int (Short and Byte promote to Int); "" unless both are numeric.
function ktNumPromote(a, b) {
    var p = ["Double", "Float", "Long"]
    if (ktNumTy(a) == "" || ktNumTy(b) == "" || a.charAt(a.length - 1) == "?" || b.charAt(b.length - 1) == "?" ||
        a.charAt(0) == "U" || b.charAt(0) == "U") { return "" }
    for (var i = 0; i < p.length; i++) { if (a == p[i] || b == p[i]) { return p[i] } }
    return "Int"
}
// `a op b`'s type: numeric promotion, String.plus, and Char +/- Int (a Char),
// Char - Char (an Int).
function ktArithTy(a, op, b) {
    if (a == "String" && op == "+") { return "String" }
    if (a == "Char") {
        if ((op == "+" || op == "-") && ktNumPromote(b, "Int") == "Int") { return "Char" }
        return (op == "-" && b == "Char") ? "Int" : ""
    }
    return ktNumPromote(a, b)
}
// A range's type over operands of types a and b - IntRange, LongRange, CharRange,
// or the Progression kind - or "".
function ktRangeTy(a, b, kind) {
    if (a == "Char" && b == "Char") { return "Char" + kind }
    var n = ktNumPromote(a, b)
    return (n == "Int" || n == "Long") ? n + kind : ""
}
// The operators of an expression's text at bracket depth 0, in order, each
// {op, at}: a symbolic binary one, `..` / `..<`, a prefix one ("u-", "u+", "!"),
// and a word between two operands (an infix call: until, as, is, shl...).
// "#" marks what no static type is read through: an assignment, `::`, `->`,
// `++` / `--`, two operands in a row.
function ktTopOps(s) {
    var out = []
    var i = 0
    var opnd = false
    var ws = false
    while (i < s.length) {
        var ch = s.charAt(i)
        if (ktIsWs(ch)) { i++; ws = true; continue }
        var wsb = ws
        ws = false
        var two = s.substring(i, i + 2)
        if (ch == "\"" || ch == "'") {
            var q = ktQuoteEnd(s, i)
            if (q < 0 || opnd) { out.push({op: "#", at: i}); return out }
            i = q + 1
            opnd = true
        } else if (ch == "(" || ch == "[" || ch == "{") {
            var e = ktMatchClose(s, i, ch, ch == "(" ? ")" : (ch == "[" ? "]" : "}"))
            if (e < 0) { out.push({op: "#", at: i}); return out }
            i = e + 1
            opnd = true
        } else if (ktIdChar(ch)) {
            var j = i + 1
            if (ch >= "0" && ch <= "9") { j = i + ktLitEnd(s.substring(i)) }
            else { while (j < s.length && ktIdChar(s.charAt(j))) { j++ } }
            if (opnd) {
                if (!wsb || !(ch >= "a" && ch <= "z")) { out.push({op: "#", at: i}); return out }
                var wd = s.substring(i, j)
                if (wd == "as" && s.charAt(j) == "?") { out.push({op: "#", at: i}); return out }
                out.push({op: wd, at: i})
                opnd = false
            } else {
                opnd = true
            }
            i = j
        } else if (two == "?." || (ch == "." && two != "..")) {
            // A member step continues the operand.
            i = ktSkipWs(s, i + (ch == "?" ? 2 : 1))
            var m = i
            while (i < s.length && ktIdChar(s.charAt(i))) { i++ }
            if (i == m || !opnd) { out.push({op: "#", at: m}); return out }
        } else if (two == "!!" && opnd) {
            i = i + 2
        } else {
            var op = ktSymOp(s, i, opnd)
            if (op == "#" || op == "?") {
                if (op == "?" && opnd) { i++; continue }
                out.push({op: "#", at: i})
                return out
            }
            if (op == "<>") {
                // A call's type arguments: `listOf<Int>(1)`.
                i = ktMatchClose(s, i, "<", ">") + 1
                continue
            }
            out.push({op: op, at: i})
            i = i + (ktPrefixOp(op) ? 1 : op.length)
            opnd = false
        }
    }
    return out
}
// The symbolic operator at s[i], given whether an operand precedes it; "<>" for a
// call's type-argument list, "?" for a nullable type's mark, "#" for anything a
// static type is not read through.
function ktSymOp(s, i, opnd) {
    var ch = s.charAt(i)
    var three = s.substring(i, i + 3)
    var two = s.substring(i, i + 2)
    if (!opnd) {
        if (ch == "-" && two != "--") { return "u-" }
        if (ch == "+" && two != "++") { return "u+" }
        if (ch == "!" && two != "!=") {
            var w = ktSkipWs(s, i + 1)
            return (s.substring(w, w + 2) == "is" || s.substring(w, w + 2) == "in") && !ktIdChar(s.charAt(w + 2)) ? "#" : "!"
        }
        return "#"
    }
    if (three == "===" || three == "!==" || three == "..<") { return three }
    if (two == "!i") {
        var nw = s.substring(i + 1, i + 3)
        return (nw == "is" || nw == "in") && !ktIdChar(s.charAt(i + 3)) ? "!" + nw : "#"
    }
    if (two == "==" || two == "!=" || two == "&&" || two == "||" || two == "<=" || two == ">=" || two == "?:" || two == "..") { return two }
    if (two == "++" || two == "--" || two == "->" || two == "::" || s.charAt(i + 1) == "=") { return "#" }
    if (ch == "<") {
        var e = ktMatchClose(s, i, "<", ">")
        return (e > 0 && s.charAt(e + 1) == "(") ? "<>" : "<"
    }
    if (ch == ">" || ch == "+" || ch == "-" || ch == "*" || ch == "/" || ch == "%") { return ch }
    if (ch == "?") { return "?" }
    return "#"
}
// The static type of a postfix chain's text: a literal, a name, `this`, a call or a
// parenthesized expression, then property reads, indexes and `!!`. A method call
// in it answers "".
function ktChainTy(s) {
    var c0 = s.charAt(0)
    var ty = ""
    var i = 0
    if (c0 == "\"") {
        i = ktQuoteEnd(s, 0) + 1
        if (i <= 0) { return "" }
        ty = "String"
    } else if (c0 == "'" || (c0 >= "0" && c0 <= "9")) {
        i = ktLitEnd(s)
        if (i <= 0) { return "" }
        ty = ktLitType(s.substring(0, i))
    } else if (c0 == "(") {
        i = ktMatchClose(s, 0, "(", ")") + 1
        if (i <= 0) { return "" }
        ty = ktStaticTy(s.substring(1, i - 1))
    } else if (ktIdChar(c0)) {
        while (i < s.length && ktIdChar(s.charAt(i))) { i++ }
        var w = s.substring(0, i)
        var ce = ktCallEnd(s)
        if (w == "true" || w == "false") { ty = "Boolean" }
        else if (w == "this") { ty = ktThisTy() }
        else if (ce > 0) {
            ty = ktCallResTy(w, s.substring(0, ce))
            i = ce
        } else {
            var d = ktDeclAt(w)
            ty = d == null ? ktNestedCtorHead(w, s, i) : (d.ty != "" ? d.ty : ktDeclLty(d))
        }
    } else {
        return ""
    }
    while (i < s.length && ty != "") {
        i = ktSkipWs(s, i)
        if (i >= s.length) { break }
        var ch = s.charAt(i)
        if (s.substring(i, i + 2) == "!!") {
            if (ty.charAt(ty.length - 1) == "?") { ty = ty.substring(0, ty.length - 1) }
            i = i + 2
        } else if (ch == "[") {
            var e = ktMatchClose(s, i, "[", "]")
            if (e < 0) { return "" }
            ty = ktIndexTy(ty)
            i = e + 1
        } else if (ch == "." || s.substring(i, i + 2) == "?.") {
            var safe = ch == "?"
            i = ktSkipWs(s, i + (safe ? 2 : 1))
            var m = i
            while (i < s.length && ktIdChar(s.charAt(i))) { i++ }
            var k = ktSkipWs(s, i)
            if (i == m) { return "" }
            if (k < s.length && (s.charAt(k) == "(" || s.charAt(k) == "{" || s.charAt(k) == "<")) {
                // The call's type arguments, argument list and trailing lambdas.
                if (s.charAt(k) == "<") { k = ktMatchClose(s, k, "<", ">") + 1 }
                while (k > 0 && k < s.length && (s.charAt(k) == "(" || s.charAt(k) == "{")) {
                    k = ktSkipWs(s, ktMatchClose(s, k, s.charAt(k), s.charAt(k) == "(" ? ")" : "}") + 1)
                }
                if (k <= 0) { return "" }
                ty = ktMethodTy(ty, s.substring(m, i), s.substring(ktSkipWs(s, i), k))
                i = k
            } else {
                ty = ktPropTy(ty, s.substring(m, i))
            }
            if (safe && ty != "") { ty = ty + "?" }
        } else {
            return ""
        }
    }
    return ty
}
// A class name w heading a chain s whose next step `.Nested(` constructs a class
// declared in it: w, so that step types as that class (ktMethodTy); "" else.
function ktNestedCtorHead(w, s, i) {
    if (ktClassExt["$" + w] == undefined) { return "" }
    var j = ktSkipWs(s, i)
    if (s.charAt(j) != ".") { return "" }
    j = ktSkipWs(s, j + 1)
    var m = j
    while (j < s.length && ktIdChar(s.charAt(j))) { j++ }
    var n = s.substring(m, j)
    var e = ktClassExt["$" + n]
    var o = ktClassExt["$" + w]
    if (e == undefined || e.from < o.from || e.to > o.to || s.charAt(ktSkipWs(s, j)) != "(") { return "" }
    return w
}
// The type a call `name(...)` (its text t) yields: a class's constructor, a
// declared function's result, a builder's collection - typed by its explicit type
// arguments, else by its arguments when they all have one static type (a map
// builder's are Pairs) - and a Pair or Triple of its arguments' types.
var ktOfBuilders = {"$listOf": true, "$mutableListOf": true, "$arrayListOf": true, "$setOf": true,
                    "$mutableSetOf": true, "$hashSetOf": true, "$linkedSetOf": true, "$arrayOf": true,
                    "$sequenceOf": true, "$mapOf": true, "$mutableMapOf": true, "$hashMapOf": true,
                    "$linkedMapOf": true}
function ktCallResTy(name, t) {
    if (ktClassExt["$" + name] != undefined) { return name }
    if (ktFunRets["$" + name] != undefined) { return ktFunRet(name) }
    var bt = ktCallType(stripWs(t))
    var tup = name == "Pair" || name == "Triple"
    if (bt != "" || (ktOfBuilders["$" + name] != true && !tup)) { return bt }
    var lp = t.indexOf("(")
    if (lp != name.length || ktMatchClose(t, lp, "(", ")") != t.length - 1) { return "" }
    var args = ktSplitTop(t.substring(lp + 1, t.length - 1), ",")
    var el = ""
    var tys = []
    for (var i = 0; i < args.length; i++) {
        var at = ktStaticTy(args[i])
        if (at == "" || (el != "" && at != el && !tup)) { return "" }
        el = at
        tys.push(at)
    }
    if (tup) { return tys.length == (name == "Pair" ? 2 : 3) ? name + "<" + tys.join(",") + ">" : "" }
    var kind = ktBuilderKind["$" + name]
    if (!ktIsMapTy(kind)) { return kind + "<" + el + ">" }
    return (ktBaseName(el) == "Pair" && el.indexOf("<") > 0) ? kind + el.substring(el.indexOf("<")) : ""
}
// The type method `name` of a value of type ty returns: a declared class's own
// function's result (ktFunRet), a list of the same elements from a collection
// operation that only selects or reorders them; "" else.
var ktSameElemFns = {"$filter": true, "$filterNot": true, "$sorted": true, "$sortedBy": true,
                     "$sortedDescending": true, "$sortedByDescending": true, "$reversed": true,
                     "$distinct": true, "$distinctBy": true, "$take": true, "$drop": true,
                     "$takeWhile": true, "$dropWhile": true, "$takeLast": true, "$dropLast": true,
                     "$toList": true, "$toMutableList": true, "$shuffled": true}
// map / mapIndexed of anything iterable give a list of the lambda's result type
// (ktLamResTy), first / last / single (with or without a predicate) an element,
// toString() a String, a number's or a string's toInt() and the like that number
// type, and a string's case, trim, substring, replace, repeat and pad functions a
// String. call is the text of the call's arguments and lambdas.
var ktElemFns = {"$first": true, "$last": true, "$single": true}
var ktNumConvFns = {"$toInt": "Int", "$toLong": "Long", "$toShort": "Short", "$toByte": "Byte",
                    "$toDouble": "Double", "$toFloat": "Float"}
// The primitive array a conversion (`"ab".toCharArray()`, `xs.toIntArray()`) makes.
var ktToPrimArr = {"$toIntArray": "IntArray", "$toLongArray": "LongArray", "$toShortArray": "ShortArray",
                   "$toByteArray": "ByteArray", "$toFloatArray": "FloatArray",
                   "$toDoubleArray": "DoubleArray", "$toCharArray": "CharArray",
                   "$toBooleanArray": "BooleanArray"}
var ktStrStrFns = {"$uppercase": true, "$lowercase": true, "$trim": true, "$substring": true,
                   "$replace": true, "$repeat": true, "$padStart": true, "$padEnd": true}
function ktMethodTy(ty, name, call) {
    var base = ktBaseName(ty)
    if (ty.charAt(ty.length - 1) == "?") { return "" }
    if (ktClassExt["$" + base] != undefined) {
        var own = ktChainFunRet(base, name)
        if (own != undefined) { return own }
        // `d.Inner()` constructs the inner class it names.
        if (ktClassExt["$" + name] != undefined && ktTrimWs(call).charAt(0) == "(") { return name }
        return ktFunRets["$" + name] != undefined ? ktFunRet(name) : ""
    }
    var args = ktTrimWs(call)
    if (name == "toString" && args == "()") { return "String" }
    if (ktToPrimArr["$" + name] != undefined && args == "()") { return ktToPrimArr["$" + name] }
    if (ktNumConvFns["$" + name] != undefined && args == "()" && (base == "String" || ktNumTy(base) != "")) { return ktNumConvFns["$" + name] }
    if (base == "String" && ktStrStrFns["$" + name] == true && args.charAt(0) == "(") { return "String" }
    var el = ktElemOfTy(ty)
    if (el == "") { return "" }
    if (name == "map" || name == "mapIndexed") {
        var rt = ktLamResTy(args, name == "map" ? [el] : ["Int", el])
        return rt == "" ? "" : "List<" + rt + ">"
    }
    if (ktElemFns["$" + name] == true) { return (args == "()" || args.charAt(0) == "{") ? el : "" }
    if (name == "sum" && args == "()") { return ktSumTy(el) }
    return (ktSameElemFns["$" + name] == true && base != "String" && base.indexOf("Range") < 0 && base.indexOf("Progression") < 0) ? "List<" + el + ">" : ""
}
// The method a postfix chain's text ends by calling (`xs.first { p }` is first),
// as its name and its receiver's text, null when it ends otherwise.
function ktLastCall(s) {
    var k = s.length
    while (k > 0 && (s.charAt(k - 1) == ")" || s.charAt(k - 1) == "}")) {
        var cl = s.charAt(k - 1)
        k = ktMatchOpen(s, k - 1, cl == ")" ? "(" : "{", cl)
        while (k > 0 && ktIsWs(s.charAt(k - 1))) { k-- }
    }
    if (k <= 0 || k == s.length) { return null }
    var e = k
    while (k > 0 && ktIdChar(s.charAt(k - 1))) { k-- }
    var d = k
    while (d > 0 && ktIsWs(s.charAt(d - 1))) { d-- }
    return (k < e && d > 0 && s.charAt(d - 1) == ".") ? [s.substring(k, e), s.substring(0, d - 1)] : null
}
// The type of `this` at ktAt: the receiver of the extension function around it
// (noted by ktNoteGeneric) unless a class declared inside that function is
// closer, else the enclosing class. Inside any brace of the function's body
// other than its own (a lambda, which may bind another `this`), it is not told.
function ktThisTy() {
    var cls = ktClassAt(ktAt)
    var d = ktDeclAt("this")
    if (d == null || (cls != "" && ktClassExt["$" + cls].from > d.from)) { return cls }
    var s = c.srcSlice(d.at, ktAt)
    var paren = 0
    var depth = 0
    var own = -1
    var i = 0
    while (i < s.length) {
        var ch = s.charAt(i)
        if (ch == "\"" || ch == "'") {
            var q = ktQuoteEnd(s, i)
            if (q < 0) { return "" }
            i = q + 1
            continue
        }
        if (ch == "(") { paren++ }
        else if (ch == ")") { paren-- }
        else if (ch == "{") {
            if (own < 0 && paren == 0) { own = 1 }
            depth++
        } else if (ch == "}") { depth-- }
        else if (ch == "=" && own < 0 && paren == 0 && depth == 0) { own = 0 }
        i++
    }
    return depth == (own < 0 ? 0 : own) ? d.lty : ""
}
// The result type of a lone lambda argument (`{ it * 2 }`, `({ i, s -> i })`)
// whose parameters have the types ptys: its body's static type when the body is
// one expression, read with each parameter (or `it`) bound to its type; "" else.
function ktLamResTy(args, ptys) {
    var t = args
    if (t.charAt(0) == "(" && ktMatchClose(t, 0, "(", ")") == t.length - 1) { t = ktTrimWs(t.substring(1, t.length - 1)) }
    // An unbound reference to a member of the parameter's own type (`Int::toString`,
    // `String::length`) is `{ it.member() }` / `{ it.member }`.
    var cc = t.indexOf("::")
    if (cc > 0 && ptys.length == 1 && ktIsIdent(t.substring(0, cc)) && ktIsIdent(t.substring(cc + 2))
        && t.substring(0, cc) == ktBaseName(ptys[0])) {
        var mt = ktMethodTy(ptys[0], t.substring(cc + 2), "()")
        return mt != "" ? mt : ktPropTy(ptys[0], t.substring(cc + 2))
    }
    if (t.charAt(0) != "{" || ktMatchClose(t, 0, "{", "}") != t.length - 1) { return "" }
    var body = t.substring(1, t.length - 1)
    var names = ["it"]
    // A parameter list is names and type text up to the first arrow; an arrow
    // after anything else belongs to a nested lambda or a `when`.
    var arrow = body.indexOf("->")
    var head = arrow < 0 ? "" : body.substring(0, arrow)
    var plain = arrow >= 0
    for (var h = 0; h < head.length && plain; h++) {
        var hc = head.charAt(h)
        plain = ktIdChar(hc) || ktIsWs(hc) || hc == "," || hc == ":" || hc == "<" || hc == ">" || hc == "?" || hc == "."
    }
    if (plain) {
        names = ktSplitTop(head, ",")
        body = body.substring(arrow + 2)
    }
    body = ktTrimWs(body)
    if (names.length != ptys.length || body == "" || ktTopBreak(body)) { return "" }
    // Each binding sits at ktAt itself, so it is the closest declaration the body sees.
    var bound = []
    for (var i = 0; i < names.length; i++) {
        var nm = ktTrimWs(names[i])
        var ty = ptys[i]
        var colon = nm.indexOf(":")
        if (colon >= 0) {
            ty = stripWs(nm.substring(colon + 1))
            nm = ktTrimWs(nm.substring(0, colon))
        }
        var l = ktDecls["$" + nm]
        if (l == undefined) {
            l = []
            ktDecls["$" + nm] = l
        }
        l.push({ty: "", lty: ty, at: ktAt, from: -1, to: -1, vr: false})
        bound.push(l)
    }
    var rt = ktStaticTy(body)
    for (var j = 0; j < bound.length; j++) { bound[j].pop() }
    return rt
}
// Whether s holds a `;` or a line break outside brackets and quotes - a statement
// boundary, so s is not one expression. A break inside a `when`'s braces is not one.
function ktTopBreak(s) {
    var depth = 0
    var k = 0
    while (k < s.length) {
        var ch = s.charAt(k)
        if (ch == "\"" || ch == "'") {
            var q = ktQuoteEnd(s, k)
            if (q < 0) { return true }
            k = q + 1
            continue
        }
        if (ch == "(" || ch == "[" || ch == "{") { depth++ }
        else if (ch == ")" || ch == "]" || ch == "}") { depth-- }
        else if (depth == 0 && (ch == ";" || ch == "\n")) { return true }
        k++
    }
    return false
}
// The type property `name` of a value of type ty has: a declared class's member,
// a collection's or array's size, a string's length, indices and lastIndex.
function ktPropTy(ty, name) {
    var base = ktBaseName(ty)
    if (ty.charAt(ty.length - 1) == "?") { return "" }
    if (ktClassExt["$" + base] != undefined) {
        var mt = ktMemberTypeIn(base, name)
        return mt != "" ? mt : ktMemberTyOf(ktMemberLtys["$" + name], base)
    }
    var seq = ktOneArgColl["$" + base] != undefined || ktTwoArgColl["$" + base] != undefined ||
              ktPrimArrElem["$" + base] != undefined || base == "String" || base == "StringBuilder"
    if (!seq || base == "Pair" || base == "Iterable" || base == "Sequence") { return "" }
    var isMap = ktTwoArgColl["$" + base] != undefined
    // A map's views: its entries, keys and values.
    var kv = (isMap && ty.indexOf("<") > 0) ? ktSplitTArgs(ty.substring(ty.indexOf("<") + 1, ty.length - 1)) : []
    if (kv.length == 2) {
        if (name == "entries") { return "Set<Map.Entry<" + kv[0] + "," + kv[1] + ">>" }
        if (name == "keys") { return "Set<" + kv[0] + ">" }
        if (name == "values") { return "Collection<" + kv[1] + ">" }
    }
    if (name == "size") { return (base == "String" || base == "StringBuilder") ? "" : "Int" }
    if (name == "length") { return (base == "String" || base == "StringBuilder") ? "Int" : "" }
    if (name == "lastIndex" && !isMap) { return "Int" }
    if (name == "indices" && !isMap) { return "IntRange" }
    return ""
}
// The element type `x[i]` reads off a value of type ty: a list's or an array's
// element, a string's Char; "" for a map (its get is nullable) and anything else.
function ktIndexTy(ty) {
    var base = ktBaseName(ty)
    if (ty.charAt(ty.length - 1) == "?") { return "" }
    if (base == "String") { return "Char" }
    if (ktPrimArrElem["$" + base] != undefined) { return ktPrimArrElem["$" + base] }
    if (base != "List" && base != "MutableList" && base != "ArrayList" && base != "Array") { return "" }
    return ktOneTArg(ty)
}
// The one type argument of `Base<A>`, "" for any other type.
function ktOneTArg(ty) {
    var lt = ty.indexOf("<")
    if (lt < 0 || ty.charAt(ty.length - 1) != ">") { return "" }
    var targs = ktSplitTArgs(ty.substring(lt + 1, ty.length - 1))
    return targs.length == 1 ? targs[0] : ""
}
// The type of the elements a value of type ty iterates over: a range's or a
// progression's, a string's Char, a primitive array's, a one-argument collection's
// or Array's type argument; "" for a map and anything else.
function ktElemOfTy(ty) {
    var base = ktBaseName(ty)
    if (ty.charAt(ty.length - 1) == "?") { return "" }
    var ks = ["Int", "Long", "Char"]
    for (var i = 0; i < ks.length; i++) {
        if (base == ks[i] + "Range" || base == ks[i] + "Progression") { return ks[i] }
    }
    if (base == "String") { return "Char" }
    if (ktPrimArrElem["$" + base] != undefined) { return ktPrimArrElem["$" + base] }
    if (ktOneArgColl["$" + base] == undefined) { return "" }
    var a = ktOneTArg(ty)
    // A variance-marked argument arrives without whitespace (`outInt`).
    return (a.substring(0, 3) == "out" && ktIsUpper(a.charAt(3))) ? a.substring(3) : a
}

// The program's functions by name, for a call's static type: {ty} the written
// result type, Unit for a block body, or {ex, at} an expression body's text and
// offset, typed on first use. Declarations of one name that disagree answer "".
// ktFunDecls keeps every declaration of a name, so a method call on a receiver of a
// known class takes that class's own declaration (ktMethodTy).
var ktFunRets = {}
var ktFunDecls = {}
function ktNoteFunRet(raw, at) {
    var i = raw.indexOf("fun")
    if (i < 0) { return }
    i = ktSkipWs(raw, i + 3)
    if (raw.charAt(i) == "<") {
        i = ktMatchClose(raw, i, "<", ">")
        if (i < 0) { return }
        i = ktSkipWs(raw, i + 1)
    }
    var n0 = i
    while (i < raw.length && ktIdChar(raw.charAt(i))) { i++ }
    var name = raw.substring(n0, i)
    i = ktSkipWs(raw, i)
    if (name == "" || raw.charAt(i) != "(") { return }
    i = ktMatchClose(raw, i, "(", ")")
    if (i < 0) { return }
    i = ktSkipWs(raw, i + 1)
    var f = {ty: "Unit", ex: "", at: 0, pos: at}
    if (raw.charAt(i) == ":") {
        var e = i + 1
        var depth = 0
        while (e < raw.length) {
            var ch = raw.charAt(e)
            if (ch == "-" && raw.charAt(e + 1) == ">") { e = e + 2; continue }
            if (ch == "(" || ch == "<" || ch == "[") { depth++ }
            else if (ch == ")" || ch == ">" || ch == "]") { depth-- }
            else if (depth == 0 && (ch == "{" || ch == "=")) { break }
            e++
        }
        f.ty = stripWs(raw.substring(i + 1, e))
    } else if (raw.charAt(i) == "=") {
        f.ty = ""
        f.ex = raw.substring(i + 1)
        f.at = at + i + 1
    }
    var old = ktFunRets["$" + name]
    var all = ktFunDecls["$" + name]
    if (all == undefined) { all = []; ktFunDecls["$" + name] = all }
    var seen = false
    for (var k = 0; k < all.length; k++) { if (all[k].pos == at) { seen = true } }
    if (!seen) { all.push(f) }
    if (old != undefined && old.pos != at && !(old.ex == "" && f.ex == "" && old.ty == f.ty)) {
        ktFunRets["$" + name] = {ty: "", ex: "", at: 0, pos: -1}
        return
    }
    ktFunRets["$" + name] = f
}
function ktFunRet(name) { return ktFunRetTy(ktFunRets["$" + name]) }
// The result type of the method `name` that type cls declares (cls is the innermost
// type around the declaration) or inherits from a supertype its header names
// (ktTypeSupers), undefined when there is none.
function ktChainFunRet(cls, name) {
    var l = ktFunDecls["$" + name]
    if (l == undefined) { return undefined }
    var queue = [cls]
    for (var g = 0; g < queue.length && g < 64; g++) {
        for (var i = 0; i < l.length; i++) {
            if (ktOuterDecl({from: l[i].pos, to: l[i].pos}) == queue[g]) { return ktFunRetTy(l[i]) }
        }
        var sups = ktTypeSupers["$" + queue[g]]
        for (var k = 0; sups != undefined && k < sups.length; k++) { queue.push(sups[k]) }
    }
    return undefined
}
function ktFunRetTy(f) {
    if (f.ex != "") {
        var ex = f.ex
        f.ex = ""
        var saved = ktAt
        ktAt = f.at
        f.ty = ktStaticTy(ex)
        ktAt = saved
    }
    return f.ty
}

// A for-loop's variable or destructuring pattern, typed (lty) by the elements of
// what it iterates (ktIterElemTy) and scoped to the loop. raw is the loop's source text, from..to its
// extent.
function ktNoteForVar(raw, from, to) {
    var p = raw.indexOf("(")
    var i = ktSkipWs(raw, p + 1)
    var n0 = i
    var dp = raw.charAt(i) == "(" ? ktMatchClose(raw, i, "(", ")") : -1
    if (dp > 0) { i = dp + 1 }
    while (i < raw.length && ktIdChar(raw.charAt(i))) { i++ }
    var close = ktMatchClose(raw, p, "(", ")")
    var k = ktSkipWs(raw, i)
    if (p < 0 || i == n0 || close < 0 || raw.substring(k, k + 2) != "in" || ktIdChar(raw.charAt(k + 2))) { return }
    var saved = ktAt
    ktAt = from + k
    var el = ktIterElemTy(ktStaticTy(raw.substring(k + 2, close)))
    ktAt = from + n0
    if (dp > 0) { ktNoteDestruct(raw.substring(n0, dp + 1), from + n0, el) } else { ktNoteDecl(raw.substring(n0, i), "", el) }
    ktAt = saved
    ktNoteBody(from, to)
}

// The static type of parameter pi of a lambda from the call it trails, lb being the
// offset of the lambda's `{`: for its only parameter (pi -1) the receiver's element
// type for a collection operation that hands over one element (ktLamElemFns), the
// receiver's own type for let / also / takeIf / takeUnless, and repeat's Int; for
// the two of an indexed operation (ktLamIdxFns) the index's Int and the element
// type. "" for any other lambda.
var ktLamElemFns = {"$forEach": true, "$onEach": true, "$map": true, "$mapNotNull": true,
                    "$flatMap": true, "$filter": true, "$filterNot": true, "$any": true, "$all": true,
                    "$none": true, "$count": true, "$find": true, "$findLast": true, "$first": true,
                    "$firstOrNull": true, "$last": true, "$lastOrNull": true, "$single": true,
                    "$singleOrNull": true, "$sumOf": true, "$takeWhile": true, "$dropWhile": true,
                    "$sortedBy": true, "$sortedByDescending": true, "$groupBy": true,
                    "$associateBy": true, "$associateWith": true, "$maxBy": true, "$minBy": true,
                    "$maxByOrNull": true, "$minByOrNull": true, "$maxOf": true, "$minOf": true,
                    "$partition": true, "$indexOfFirst": true, "$indexOfLast": true, "$distinctBy": true}
var ktLamSelfFns = {"$let": true, "$also": true, "$takeIf": true, "$takeUnless": true}
var ktLamIdxFns = {"$forEachIndexed": true, "$mapIndexed": true, "$filterIndexed": true,
                   "$onEachIndexed": true, "$mapIndexedNotNull": true}
function ktLamParamTy(lb, pi) {
    var w0 = lb > 600 ? lb - 600 : 0
    var w = c.srcSlice(w0, lb)
    var k = w.length
    while (k > 0 && ktIsWs(w.charAt(k - 1))) { k-- }
    var me = k
    if (pi < 0 && k > 0 && w.charAt(k - 1) == ")") {
        // `repeat(n) { ... }`
        var rp = ktMatchOpen(w, k - 1, "(", ")")
        return (rp >= 6 && w.substring(rp - 6, rp) == "repeat" && (rp == 6 || !ktIdChar(w.charAt(rp - 7)))) ? "Int" : ""
    }
    while (k > 0 && ktIdChar(w.charAt(k - 1))) { k-- }
    var m = "$" + w.substring(k, me)
    var fits = pi < 0 ? (ktLamElemFns[m] == true || ktLamSelfFns[m] == true) : ktLamIdxFns[m] == true
    if (k == me || k < 1 || w.charAt(k - 1) != "." || !fits) { return "" }
    if (pi == 0) { return "Int" }
    k--
    if (k > 0 && w.charAt(k - 1) == "?") { k-- }
    while (k > 0 && ktIsWs(w.charAt(k - 1))) { k-- }
    var re = k
    while (k > 0) {
        var ch = w.charAt(k - 1)
        if (ktIdChar(ch) || ch == "." || ch == "?" || ch == "!") { k--; continue }
        if (ch == ")" || ch == "]" || ch == "}" || (ch == ">" && k > 1 && w.charAt(k - 2) != "-")) {
            var op = ch == ")" ? "(" : (ch == "]" ? "[" : (ch == "}" ? "{" : "<"))
            k = ktMatchOpen(w, k - 1, op, ch)
            if (k < 0) { return "" }
            continue
        }
        if (ktIsWs(ch)) {
            var after = w.charAt(k)
            var q = k
            while (q > 0 && ktIsWs(w.charAt(q - 1))) { q-- }
            if (after == "." || after == "?" || after == "{") { k = q; continue }
        }
        break
    }
    if (k == re || (k == 0 && w0 > 0)) { return "" }
    var saved = ktAt
    ktAt = w0 + k
    var rtxt = w.substring(k, re)
    var ty = ktStaticTy(rtxt)
    ktAt = saved
    if (ktLamSelfFns[m] != true) { return ktIterElemTy(ty) }
    var sc = ktSmartCastTy(rtxt, w0 + k)
    return sc != "" ? sc : ty
}
// The type of the elements a for loop or a collection lambda sees in a value of
// type ty: a map's entries, else ktElemOfTy's.
function ktIterElemTy(ty) {
    if (!ktIsMapTy(ty) || ty.indexOf("<") < 0) { return ktElemOfTy(ty) }
    var kv = ktSplitTArgs(ty.substring(ty.indexOf("<") + 1, ty.length - 1))
    return kv.length == 2 ? "Map.Entry<" + kv[0] + "," + kv[1] + ">" : ""
}
// The type a plain local `name` is smart-cast to at offset `at` by an earlier
// unconditional `name as T` (not `as?`) in a block still open there, "" when
// there is none; a reassignment `name = ...` ends it. The source is scanned
// from the name's declaration on.
function ktSmartCastTy(name, at) {
    for (var n = 0; n < name.length; n++) { if (!ktIdChar(name.charAt(n))) { return "" } }
    var saved = ktAt
    ktAt = at
    var d = ktDeclAt(name)
    ktAt = saved
    if (d == null || d.at >= at) { return "" }
    var s = c.srcSlice(d.at, at)
    var ty = ""
    var tyDepth = -1
    var depth = 0
    var i = 0
    while (i < s.length) {
        var ch = s.charAt(i)
        if (ch == "/" && s.charAt(i + 1) == "/") {
            while (i < s.length && s.charAt(i) != "\n") { i++ }
            continue
        }
        if (ch == "\"" || ch == "'") {
            var q = ktQuoteEnd(s, i)
            if (q < 0) { return "" }
            i = q + 1
            continue
        }
        if (ch == "{") { depth++ }
        if (ch == "}") {
            depth--
            if (depth < tyDepth) { ty = ""; tyDepth = -1 }
        }
        if (!ktIdChar(ch)) { i++; continue }
        var j = i
        while (j < s.length && ktIdChar(s.charAt(j))) { j++ }
        if (s.substring(i, j) == name && (i == 0 || s.charAt(i - 1) != ".")) {
            var k = ktSkipWs(s, j)
            if (s.substring(k, k + 2) == "as" && !ktIdChar(s.charAt(k + 2)) && s.charAt(k + 2) != "?" && ktCastUncond(s, i)) {
                var t0 = ktSkipWs(s, k + 2)
                var t1 = t0
                while (t1 < s.length && (ktIdChar(s.charAt(t1)) || s.charAt(t1) == ".")) { t1++ }
                if (t1 > t0 && s.charAt(t1) != "<" && s.charAt(t1) != "?") { ty = s.substring(t0, t1); tyDepth = depth }
            } else if (s.charAt(k) == "=" && s.charAt(k + 1) != "=") {
                ty = ""
                tyDepth = -1
            }
        }
        i = j
    }
    return ty
}
// Whether the expression starting at offset i of s runs whenever the code after
// it does: not the branch of a brace-less if / else / when arm.
function ktCastUncond(s, i) {
    var k = i
    while (k > 0 && ktIsWs(s.charAt(k - 1))) { k-- }
    if (k > 0 && s.charAt(k - 1) == ")") { return false }
    if (k > 1 && s.substring(k - 2, k) == "->") { return false }
    return !(k > 3 && s.substring(k - 4, k) == "else" && (k == 4 || !ktIdChar(s.charAt(k - 5))))
}
// The component types a destructuring of a value of type ty binds: a Pair's, a
// Triple's and a map entry's type arguments; none for any other type.
function ktComponentTys(ty) {
    var base = ktBaseName(ty)
    var n = base == "Triple" ? 3 : ((base == "Pair" || base == "Map.Entry") ? 2 : 0)
    var lt = ty.indexOf("<")
    if (n == 0 || lt < 0 || ty.charAt(ty.length - 1) != ">") { return [] }
    var ts = ktSplitTArgs(ty.substring(lt + 1, ty.length - 1))
    return ts.length == n ? ts : []
}
// The names of a destructuring pattern `(a, b: T, _)` (raw, at offset from),
// declared as vals typed (lty) by the component of ty they bind, which the JVM
// local keeps under a wider written type too (kotlinc 2.2.21: `val (u: Number, _)
// = 1 to 2` holds an int), else by the written type. A name-based entry
// (`(x = first)`) is left untyped.
function ktNoteDestruct(raw, from, ty) {
    var p = ktSkipWs(raw, 0)
    if (raw.charAt(p) != "(") { return }
    var close = ktMatchClose(raw, p, "(", ")")
    if (close < 0) { return }
    var comps = ktComponentTys(ty)
    var parts = ktSplitTop(raw.substring(p + 1, close), ",")
    var off = p + 1
    var saved = ktAt
    for (var i = 0; i < parts.length; i++) {
        var part = parts[i]
        var j = ktSkipWs(part, 0)
        var n0 = j
        while (j < part.length && ktIdChar(part.charAt(j))) { j++ }
        var nm = part.substring(n0, j)
        var r = ktSkipWs(part, j)
        var wt = part.charAt(r) == ":" ? stripWs(ktSplitTop(part.substring(r + 1), "=")[0]) : ""
        if (nm != "" && nm != "_" && ktSplitTop(part, "=").length == 1) {
            ktAt = from + off + n0
            ktNoteDecl(nm, "", (i < comps.length && comps[i] != "") ? comps[i] : wt)
        }
        off = off + part.length + 1
    }
    ktAt = saved
}
// A lambda's destructured parameter at from..to: typed by its only parameter's
// type, or by the second one of an indexed operation's two (ktLamParamTy).
function ktNoteLamDestruct(from, to) {
    var w0 = from > 80 ? from - 80 : 0
    var pre = c.srcSlice(w0, from)
    var k = pre.length
    while (k > 0 && ktIsWs(pre.charAt(k - 1))) { k-- }
    var pi = -1
    if (k > 0 && pre.charAt(k - 1) == ",") {
        k--
        while (k > 0 && ktIsWs(pre.charAt(k - 1))) { k-- }
        var e = k
        while (k > 0 && ktIdChar(pre.charAt(k - 1))) { k-- }
        while (k > 0 && ktIsWs(pre.charAt(k - 1))) { k-- }
        if (k == e) { return }
        pi = 1
    } else {
        var post = c.srcSlice(to, to + 40)
        var a = ktSkipWs(post, 0)
        if (post.substring(a, a + 2) != "->") { return }
    }
    if (k == 0 || pre.charAt(k - 1) != "{") { return }
    ktNoteDestruct(c.srcSlice(from, to), from, ktLamParamTy(w0 + k - 1, pi))
}
// A destructuring declaration `val (a, b) = e` (raw, at offset from): its names
// typed by the components of e's static type.
function ktNoteValDestruct(raw, from) {
    var p = raw.indexOf("(")
    var close = p < 0 ? -1 : ktMatchClose(raw, p, "(", ")")
    if (close < 0) { return }
    var eq = ktSkipWs(raw, close + 1)
    if (raw.charAt(eq) != "=") { return }
    var init = ktTrimWs(raw.substring(eq + 1))
    if (init.charAt(init.length - 1) == ";") { init = init.substring(0, init.length - 1) }
    var saved = ktAt
    ktAt = from + eq + 1
    var ty = ktStaticTy(init)
    ktAt = saved
    ktNoteDestruct(raw.substring(p, close + 1), from + p, ty)
}
// The index of the bracket opening the one closing at `at`, scanning back, or -1.
function ktMatchOpen(s, at, open, close) {
    var depth = 0
    for (var i = at; i >= 0; i--) {
        var c = s.charAt(i)
        if (c == close) { depth++ }
        else if (c == open) { depth--; if (depth == 0) { return i } }
    }
    return -1
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
// The initializer of a declaration's source text `val x: T = e;` with its
// whitespace, which an infix call (`1 to 2`) needs, or "" without one.
function ktSrcInit(src) {
    var head = ktSplitTop(src, "=")[0]
    if (head.length >= src.length) { return "" }
    var t = ktTrimWs(src.substring(head.length + 1))
    if (t.charAt(t.length - 1) == ";") { t = t.substring(0, t.length - 1) }
    return t
}
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
// The type a method's RESULT takes: sum() of an empty collection is its result
// type's zero, not an Int.
function ktMResType(name, nargs, el) {
    var t = (name == "sum" && nargs == 0) ? ktSumTy(el.v) : ""
    return t == "Int" ? "" : t
}
// The type sum() of elements of type el yields: a Short or Byte sum widens to Int,
// a UShort or UByte sum to UInt, the other numeric types keep theirs; "" else.
function ktSumTy(el) {
    var n = ktNumTy(el)
    if (n == "" || n.charAt(n.length - 1) == "?") { return "" }
    if (n == "Short" || n == "Byte") { return "Int" }
    return (n == "UShort" || n == "UByte") ? "UInt" : n
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
// The java.* interfaces an import may name that the runtime judges by name only
// (ktJvmCastDest): the import binds nothing, since a type test reads the name.
function ktJavaTypeImport(path) {
    return path == "java.io.Serializable" || path == "java.util.RandomAccess" || path == "java.lang.Cloneable" ||
        path == "java.util.SortedSet" || path == "java.util.NavigableSet" || path == "java.util.SortedMap" ||
        path == "java.util.NavigableMap"
}

// The type kotlinc's null-cast message names - "null cannot be cast to non-null type
// kotlin.collections.List<kotlin.Int>" for `null as List<Int>`: the written type with
// every stdlib name fully qualified, its type arguments included. dest is the JVM
// class a judged cast checks against ("" when the cast checks no class); a program
// type keeps its dotted path. "" when the cast cannot check null either: a type
// parameter, a function type, a library type outside ktFqTypes.
var ktFqTypes = {}
function ktFqAdd(pkg, names) {
    for (var i = 0; i < names.length; i++) { ktFqTypes["$" + names[i]] = pkg + names[i] }
}
ktFqAdd("kotlin.", ["Any", "Unit", "String", "CharSequence", "Int", "Long", "Short", "Byte",
                    "Double", "Float", "Char", "Boolean", "Number", "Comparable", "Array",
                    "IntArray", "LongArray", "ShortArray", "ByteArray", "DoubleArray", "FloatArray",
                    "CharArray", "BooleanArray", "Pair", "Triple", "Cloneable"])
ktFqAdd("kotlin.collections.", ["List", "MutableList", "Set", "MutableSet", "Map", "MutableMap",
                                "Collection", "MutableCollection", "Iterable", "MutableIterable",
                                "Iterator", "MutableIterator"])
ktFqAdd("kotlin.sequences.", ["Sequence"])
function ktNullCastType(t, dest) {
    if (t.indexOf("->") >= 0) { return "" }
    var s = (t.length > 0 && t.charAt(t.length - 1) == "?") ? t.substring(0, t.length - 1) : t
    var lt = s.indexOf("<")
    var base = lt >= 0 ? s.substring(0, lt) : s
    var head = ktFqTypes["$" + base]
    if (dest != "" && dest.indexOf("java.lang.") != 0) { head = dest.split("$").join(".") }
    if (head == undefined) { return "" }
    return head + ktFqTArgs(s, lt)
}
// `<A, B>` of a type whose `<` is at lt, each argument qualified the same way; a
// name outside ktFqTypes (a type parameter, a program type) is kept as written.
// The text arrives without whitespace, so a variance reads `outNumber`.
function ktFqTArgs(s, lt) {
    if (lt < 0) { return "" }
    var args = ktSplitTArgs(s.substring(lt + 1, s.length - 1))
    var out = []
    for (var i = 0; i < args.length; i++) {
        var a = args[i]
        var pre = ""
        if (a.length > 3 && a.substring(0, 3) == "out" && ktIsUpper(a.charAt(3))) { pre = "out "; a = a.substring(3) }
        else if (a.length > 2 && a.substring(0, 2) == "in" && ktIsUpper(a.charAt(2))) { pre = "in "; a = a.substring(2) }
        var q = (a.length > 0 && a.charAt(a.length - 1) == "?") ? "?" : ""
        if (q != "") { a = a.substring(0, a.length - 1) }
        var alt = a.indexOf("<")
        var ab = alt >= 0 ? a.substring(0, alt) : a
        var fq = ktFqTypes["$" + ab]
        out.push(pre + (fq == undefined ? ab : fq) + ktFqTArgs(a, alt) + q)
    }
    return "<" + out.join(", ") + ">"
}
function ktIsUpper(c) { return c >= "A" && c <= "Z" }

// ----- `x as T`: the JVM class the checkcast names -----
// A builtin value type or a collection interface: its JVM class.
var ktCastLang = {"$String": "java.lang.String", "$Int": "java.lang.Integer", "$Long": "java.lang.Long",
                  "$Short": "java.lang.Short", "$Byte": "java.lang.Byte", "$Double": "java.lang.Double",
                  "$Float": "java.lang.Float", "$Char": "java.lang.Character",
                  "$Boolean": "java.lang.Boolean", "$Number": "java.lang.Number",
                  "$Comparable": "java.lang.Comparable",
                  "$List": "java.util.List", "$MutableList": "java.util.List",
                  "$Set": "java.util.Set", "$MutableSet": "java.util.Set",
                  "$Map": "java.util.Map", "$MutableMap": "java.util.Map",
                  "$Collection": "java.util.Collection", "$MutableCollection": "java.util.Collection",
                  "$Iterable": "java.lang.Iterable", "$MutableIterable": "java.lang.Iterable"}
// A concrete java.util collection class (kotlin.collections' ArrayList, HashSet,
// LinkedHashSet, HashMap and LinkedHashMap are typealiases to them; a bare
// ArrayDeque is kotlin.collections' own) or a JVM marker interface: its JVM class,
// "" for any other t. The runtime's `is` judges exactly these names (ktJvmIsA,
// ktMarkerName).
function ktJvmCastDest(t) {
    var q = t.indexOf("java.util.") == 0
    var s = t
    if (q) { s = t.substring(10) } else if (t.indexOf("kotlin.collections.") == 0) { s = t.substring(19) }
    if (s == "ArrayDeque") { return q ? "java.util.ArrayDeque" : "kotlin.collections.ArrayDeque" }
    if (s == "ArrayList" || s == "HashSet" || s == "LinkedHashSet" || s == "HashMap" || s == "LinkedHashMap" ||
        s == "TreeSet" || s == "TreeMap" || s == "PriorityQueue" || s == "LinkedList" || s == "SortedSet" ||
        s == "NavigableSet" || s == "SortedMap" || s == "NavigableMap" || s == "RandomAccess") { return "java.util." + s }
    if (t == "Cloneable" || t == "kotlin.Cloneable" || t == "java.lang.Cloneable") { return "java.lang.Cloneable" }
    if (t == "Serializable" || t == "java.io.Serializable") { return "java.io.Serializable" }
    return ""
}
