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
// `var x = e`: the local's static type is e's (JLS 14.4.1); "var" while e's
// type cannot be read. dtext is the declarator's whitespace-free text.
function jVarTy(dtext) {
    var e = dtext.indexOf("=")
    if (e < 0) { return "var" }
    var t = jStaticTy(dtext.substring(e + 1, dtext.length))
    if (t == "" || t == "null" || t == "fn") { return "var" }
    return t
}
// An enhanced for's variable is declared with its FeType (`for (Integer x :
// l)` makes l.remove(x) the remove(Object) overload); dims written after the
// name (`for (int r[] : m)`) extend it.
var jFeTy = ""
var jFeName = ""
function jFeDecl(name) {
    jFeName = name
    jTyDecl(name, jFeTy)
}
function jFeDims(text) { jTyDecl(jFeName, jFeTy + jDimsOf(text)) }
// A field's type, by NAME across the whole program: "?" once two
// declarations of the name disagree, and then no conversion is made. The
// grammars read it once the walk has seen every field.
var jFieldTys = {}
// The same per declaring class ("Cls.name"), which a bare name in that class'
// own body reads first (jOwnFieldTy).
var jFieldTysC = {}
function jNoteFieldTy(name, ty) {
    var have = jFieldTys["$" + name]
    jFieldTys["$" + name] = (have == undefined || have == ty) ? ty : "?"
    jFieldTysC["$" + curClassName + "." + name] = ty
}
function jOwnFieldTy(name) {
    var t = jFieldTysC["$" + curClassName + "." + name]
    return t == undefined ? jFieldTyOf(name) : t
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
// The binary name java.lang.Class#getName gives the array type ty, which an
// array created with that type carries (jArrTag in each grammar):
// "String[][]" is "[[Ljava.lang.String;", "Main.P[]" is "[LMain$P;". "" when
// ty is no array type or names no type this program knows, and for a
// one-dimensional Object[]: lib/java-util.java builds every array it hands
// back as one, whatever the caller's type was, so the element-type inference
// of an untagged array stays the better answer there. Read once every class is
// declared, and memoized.
var jArrDescs = {}
var jPrimDescs = {"$int": "I", "$long": "J", "$short": "S", "$byte": "B", "$char": "C",
                  "$float": "F", "$double": "D", "$boolean": "Z"}
function jArrDesc(ty) {
    var memo = jArrDescs["$" + ty]
    if (memo != undefined) { return memo }
    var t = jStripTargs(ty)
    var dims = ""
    while (jElemTy(t) != "") {
        dims = dims + "["
        t = jElemTy(t)
    }
    var d = ""
    if (dims != "") {
        var from = 0
        for (var i = 0; i < t.length; i++) { if (t.charAt(i) == ".") { from = i + 1 } }
        var simple = t.substring(from)
        var bin = jBinName["$" + simple]
        if (jPrimDescs["$" + t] != undefined) { d = dims + jPrimDescs["$" + t] }
        else if (jLangStatics["$" + simple] != undefined) { d = dims + "Ljava.lang." + simple + ";" }
        else if (bin != undefined) { d = dims + "L" + bin + ";" }
        if (d == "[Ljava.lang.Object;") { d = "" }
    }
    jArrDescs["$" + ty] = d
    return d
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
// The same per declaring class ("Cls.name"), which a call whose receiver type
// is known reads first (jClsMethodTyOf).
var jMethodTysC = {}
function jNoteMethodTy(name) {
    var have = jMethodTys["$" + name]
    jMethodTys["$" + name] = (have == undefined || have == jRetTy()) ? jRetTy() : "?"
    var key = "$" + curClassName + "." + name
    var hc = jMethodTysC[key]
    jMethodTysC[key] = (hc == undefined || hc == jRetTy()) ? jRetTy() : "?"
}
// A method's result type as class cls or a superclass declares it, else by
// name across the program.
function jClsMethodTyOf(cls, name) {
    var c = cls
    var guard = 0
    while (c != "" && guard < 64) {
        guard++
        var t = jMethodTysC["$" + c + "." + name]
        if (t != undefined) { return t == "?" ? "" : t }
        var rec = jOvCls["$" + c]
        if (rec == undefined) { break }
        c = rec.sup
    }
    return jMethodTyOf(name)
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
    var args = jSplitArgs(inner)
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
// An argument list's (whitespace-free) text split at its top-level commas.
function jSplitArgs(inner) {
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
    return args
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
// The public static methods of the java.lang types a value of this model is an
// instance of, as JDK 24's reflection lists them. A reference Type::m naming
// any other method of one of these types is unbound (JLS 15.13.1): its first
// argument is the receiver. A name that is both static and instance
// (Integer::toString, Double::isNaN) is ambiguous wherever both apply, so the
// static reading is the only one a valid program can mean.
var jLangStatics = {
    "$String": "|copyValueOf|format|join|valueOf|",
    "$Integer": "|bitCount|compare|compareUnsigned|compress|decode|divideUnsigned|expand|getInteger|hashCode|highestOneBit|lowestOneBit|max|min|numberOfLeadingZeros|numberOfTrailingZeros|parseInt|parseUnsignedInt|remainderUnsigned|reverse|reverseBytes|rotateLeft|rotateRight|signum|sum|toBinaryString|toHexString|toOctalString|toString|toUnsignedLong|toUnsignedString|valueOf|",
    "$Long": "|bitCount|compare|compareUnsigned|compress|decode|divideUnsigned|expand|getLong|hashCode|highestOneBit|lowestOneBit|max|min|numberOfLeadingZeros|numberOfTrailingZeros|parseLong|parseUnsignedLong|remainderUnsigned|reverse|reverseBytes|rotateLeft|rotateRight|signum|sum|toBinaryString|toHexString|toOctalString|toString|toUnsignedString|valueOf|",
    "$Short": "|compare|compareUnsigned|decode|hashCode|parseShort|reverseBytes|toString|toUnsignedInt|toUnsignedLong|valueOf|",
    "$Byte": "|compare|compareUnsigned|decode|hashCode|parseByte|toString|toUnsignedInt|toUnsignedLong|valueOf|",
    "$Double": "|compare|doubleToLongBits|doubleToRawLongBits|hashCode|isFinite|isInfinite|isNaN|longBitsToDouble|max|min|parseDouble|sum|toHexString|toString|valueOf|",
    "$Float": "|compare|float16ToFloat|floatToFloat16|floatToIntBits|floatToRawIntBits|hashCode|intBitsToFloat|isFinite|isInfinite|isNaN|max|min|parseFloat|sum|toHexString|toString|valueOf|",
    "$Character": "|charCount|codePointAt|codePointBefore|codePointCount|codePointOf|compare|digit|forDigit|getDirectionality|getName|getNumericValue|getType|hashCode|highSurrogate|isAlphabetic|isBmpCodePoint|isDefined|isDigit|isEmoji|isEmojiComponent|isEmojiModifier|isEmojiModifierBase|isEmojiPresentation|isExtendedPictographic|isHighSurrogate|isISOControl|isIdentifierIgnorable|isIdeographic|isJavaIdentifierPart|isJavaIdentifierStart|isJavaLetter|isJavaLetterOrDigit|isLetter|isLetterOrDigit|isLowSurrogate|isLowerCase|isMirrored|isSpace|isSpaceChar|isSupplementaryCodePoint|isSurrogate|isSurrogatePair|isTitleCase|isUnicodeIdentifierPart|isUnicodeIdentifierStart|isUpperCase|isValidCodePoint|isWhitespace|lowSurrogate|offsetByCodePoints|reverseBytes|toChars|toCodePoint|toLowerCase|toString|toTitleCase|toUpperCase|valueOf|",
    "$Boolean": "|compare|getBoolean|hashCode|logicalAnd|logicalOr|logicalXor|parseBoolean|toString|valueOf|",
    "$Object": "|", "$Number": "|", "$Comparable": "|", "$CharSequence": "|compare|"
}
function jUnboundRef(tgt, name) {
    var st = jLangStatics["$" + tgt]
    return st != undefined && st.indexOf("|" + name + "|") < 0
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
// javac where the static type is wider than the value - the call sites whose
// static types are readable bypass the dispatcher (jOvStaticSig, below).
var jOvCls = {}
var jOvNames = []
var jOvIfaces = {}
// Each declared type's direct supertypes - its superclass and the interfaces
// it implements, or an interface's super-interfaces - as bare names.
var jOvSupers = {}
function jOvAddSuper(name, sup) {
    if (jOvSupers["$" + name] == undefined) { jOvSupers["$" + name] = [] }
    jOvSupers["$" + name].push(jOvBare(sup))
}
function jOvIfaceSupers(name, supers) {
    for (var i = 0; i < supers.length; i++) { jOvAddSuper(name, supers[i]) }
}
// The type variables a class or method declares, each mapped to its erasure
// (JLS 4.6: its first bound, else Object), and each generic class' type
// parameters in order. A static type that is a type variable reads as its
// erasure; a call's result typed by a class' type parameter reads as the
// receiver's type argument (List<String>.get is a String).
var jTyVars = {}
var jClsTyParams = {}
var jLastTyVar = ""
function jNoteTyVar(name, ofClass) {
    jTyVars["$" + name] = "Object"
    jLastTyVar = name
    if (ofClass) {
        if (jClsTyParams["$" + curClassName] == undefined) { jClsTyParams["$" + curClassName] = [] }
        jClsTyParams["$" + curClassName].push(name)
    }
}
function jNoteTyBound(b) {
    if (jLastTyVar != "") { jTyVars["$" + jLastTyVar] = jOvBare(b) }
    jLastTyVar = ""
}
// A declared type name that is a type variable (and no class or interface).
function jTyVarErasure(b) {
    if (jOvCls["$" + b] != undefined || jOvIfaces["$" + b] == true) { return "" }
    var e = jTyVars["$" + b]
    return e == undefined ? "" : e
}
// The type arguments of a parameterized type's text: "Map<String,List<Integer>>"
// -> ["String", "List<Integer>"]; [] when it has none.
function jTyArgs(ty) {
    var lt = ty.indexOf("<")
    if (lt < 0 || ty.charAt(ty.length - 1) != ">") { return [] }
    var out = []
    var depth = 0
    var from = lt + 1
    for (var i = lt + 1; i < ty.length - 1; i++) {
        var c = ty.charAt(i)
        if (c == "<") { depth++ } else if (c == ">") { depth-- } else if (c == "," && depth == 0) {
            out.push(ty.substring(from, i))
            from = i + 1
        }
    }
    out.push(ty.substring(from, ty.length - 1))
    return out
}
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
    var ctors = []
    for (var i = 0; i < members.length; i++) {
        var m = members[i]
        if (m.ext != undefined) {
            sup = jOvBare(m.ext)
            jOvAddSuper(name, sup)
        }
        if (m.impl != undefined) { jOvAddSuper(name, m.impl) }
        if (m.impls != undefined) { jOvIfaceSupers(name, m.impls) }
        if (m.kind == "method") {
            var v = jOvIsVar(m)
            ms.push({name: m.name, raw: jOvRawOf(m.ptys, v), ptys: m.ptys, isVar: v, statik: m.statik})
        }
        if (m.kind == "ctor" && m.ptys != undefined) { ctors.push(m) }
    }
    var cts = jOvCtorCands(ctors)
    for (var k = 0; k < cts.length; k++) { cts[k].n = ctors[k].params.length }
    if (jOvCls["$" + name] == undefined) { jOvNames.push(name) }
    jOvCls["$" + name] = {sup: sup, ms: ms, cts: cts}
}
// The signature `new cls(args)`, `this(args)` or `super(args)` resolves to
// among cls' constructors of that many parameters, "" when there are fewer
// than two of them (one needs no choosing) or an argument type is unknown.
// Its implementation is the class' "__ctor#<n>#<sig>" slot.
function jOvCtorSig(cls, atys) {
    if (atys == null) { return "" }
    var rec = jOvCls["$" + jOvBare(cls)]
    if (rec == undefined) { return "" }
    var same = []
    for (var i = 0; i < rec.cts.length; i++) {
        if (rec.cts[i].n == atys.length) { same.push(rec.cts[i]) }
    }
    if (same.length < 2) { return "" }
    var stoks = []
    for (var k = 0; k < atys.length; k++) {
        var s = jOvSTok(atys[k])
        if (s == "") { return "" }
        stoks.push(s)
    }
    return jOvPickStatic(jOvSpec(same), stoks)
}
function jOvIface(name, members) {
    jOvIfaces["$" + name] = true
    jOvIfaceDeclare(name, members)
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
// Does any class or interface dispatch over `mname`? Then every declaration of
// it is stored under its "name#sig" slot as well, so a class's override of an
// interface's overload is found before the interface's own. Asked only once
// the walk is complete.
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
        for (var f = 0; f < jOvIfaceNames.length; f++) {
            var ims = jOvIfaceMs["$" + jOvIfaceNames[f]]
            for (var k = 0; k < ims.length; k++) {
                if (jOvIfaceBodyCands(jOvIfaceNames[f], ims[k].name) != null) { jOvSlotNames["$" + ims[k].name] = true }
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
// name, a declared interface that is not functional as "#" + its name, and
// everything else (a functional interface, a type variable, a library type) as
// "*", which takes any reference and any lambda.
function jOvTok(raw) {
    if (raw.length > 2 && raw.substring(raw.length - 2) == "[]") { return jOvTok(raw.substring(0, raw.length - 2)) + "[]" }
    if (jOvIsPrim(raw) || jOvCore.indexOf("," + raw + ",") >= 0) { return raw }
    if (jOvIfaces["$" + raw] == true) { return jSamUser["$" + raw] == undefined ? "#" + raw : "*" }
    if (jOvCls["$" + raw] != undefined) { return raw }
    var er = jTyVarErasure(raw)
    if (er != "") { return jOvTok(er) }
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
    return jOvExtends(jOvUntag(s), jOvUntag(t), 0)
}
// A declared interface's token without its "#".
function jOvUntag(t) { return t.charAt(0) == "#" ? t.substring(1) : t }
// Does declared type s extend or implement t, directly or through its supertypes?
function jOvExtends(s, t, depth) {
    var sups = jOvSupers["$" + s]
    if (sups == undefined || depth > 64) { return false }
    for (var i = 0; i < sups.length; i++) {
        if (sups[i] == t || jOvExtends(sups[i], t, depth + 1)) { return true }
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
    return k == "obj" && isA(v, jOvUntag(t))
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

// ----- java.util -----
//
// Does this import name a type lib/java-util.java declares (`java.util.List`,
// `java.util.*`, `import static java.util.List.of`, `import module
// java.base`)? Then the grammar compiles that file with the program
// (jLoadUtil); without such an import none of its names is visible, as in Java.
function jUtilWants(path) {
    if (path == "java.util.*" || path == "java.base") { return true }
    if (path.length < 11 || path.substring(0, 10) != "java.util.") { return false }
    var rest = path.substring(10)
    var dot = rest.indexOf(".")
    if (dot >= 0) { rest = rest.substring(0, dot) }
    return ",ArrayList,List,Collection,Comparator,Iterator,NoSuchElementException,ConcurrentModificationException,AbstractList,Arrays,Optional,OptionalInt,OptionalLong,OptionalDouble,Map,HashMap,".indexOf("," + rest + ",") >= 0
}

// Does this import, or the class a `new` creates, name a type lib/java-map.java
// declares? Then the grammar compiles it, after lib/java-util.java, with the
// program. A java.util.* import alone does not: the program then creates a
// HashMap (makeNew asks with the simple name) or uses java.util.stream, which
// wants it too, before it can hold one.
var jMapWant = false
function jMapWants(path) {
    return path == "java.util.Map" || path == "java.util.HashMap" || path == "java.util.Map.Entry"
        || path == "java.util.Set" || path == "java.util.HashSet"
        || path == "HashMap" || path == "HashSet" || path == "java.base"
}

// Does this import name a java.util.stream type? Then the grammar compiles
// lib/java-stream.java, and lib/java-util.java and lib/java-map.java it builds
// on, with the program. A call of a method named stream (Arrays.stream, a
// Collection's stream()) sets jStreamWant instead (jMcallSfx), because neither
// needs an import.
var jStreamWant = false
function jStreamWants(path) {
    return path == "java.util.stream.*" || (path.length > 17 && path.substring(0, 17) == "java.util.stream.")
}

// Does `new cls(...)` create a type lib/java-lang.java declares? Then the
// grammar compiles that file with the program.
function jLangWants(cls) { return cls == "StringBuilder" }

// ----- Static overload selection (JLS 15.12.2 is static) -----
//
// javac picks an overload from the arguments' STATIC types, which the value
// model does not carry: `Object o = "s"; f(o)` calls f(Object) although o
// holds a String, and an Integer is a plain int at run time. A call site whose
// receiver type and argument types all read from the source text is resolved,
// once the walk is complete, to the "name#sig" slot it must call. A site with
// any type this cannot read keeps the dispatcher's runtime pick, so a shape it
// does not understand never changes what was resolved before.
var jOvIfaceMs = {}
var jOvIfaceNames = []
// An interface's methods as jOvDeclare records a class': a call through an
// interface-typed receiver resolves over these.
function jOvIfaceDeclare(name, members) {
    var ms = []
    for (var i = 0; i < members.length; i++) {
        var m = members[i]
        if ((m.kind == "amethod" || m.kind == "abstract" || m.kind == "method") && m.ptys != undefined) {
            var v = m.params != undefined && jOvIsVar(m)
            ms.push({name: m.name, raw: jOvRawOf(m.ptys, v), ptys: m.ptys, isVar: v, statik: m.statik == true,
                     body: m.kind == "method"})
        }
    }
    if (jOvIfaceMs["$" + name] == undefined) { jOvIfaceNames.push(name) }
    jOvIfaceMs["$" + name] = ms
}
function jOvIfaceCands(name, mname) {
    var ms = jOvIfaceMs["$" + name]
    if (ms == undefined) { return null }
    var out = []
    var seen = {}
    for (var i = 0; i < ms.length; i++) {
        if (ms[i].name != mname || seen["$" + ms[i].raw] == true) { continue }
        seen["$" + ms[i].raw] = true
        out.push(ms[i])
    }
    if (out.length < 2) { return null }
    return out
}
// The overloads of `mname` an interface declares with a body, when there are
// two or more of one kind (all static or all default): the interface's own
// dispatcher picks among them, as jOvCands' does for a class.
function jOvIfaceBodyCands(name, mname) {
    var all = jOvIfaceCands(name, mname)
    if (all == null) { return null }
    var out = []
    for (var i = 0; i < all.length; i++) {
        if (all[i].body) { out.push(all[i]) }
    }
    if (out.length < 2) { return null }
    for (var k = 1; k < out.length; k++) { if (out[k].statik != out[0].statik) { return null } }
    return out
}
// The static types of a call's arguments, read from the call's own text
// (`name(args)` or `.name(args)`), or null when the split does not yield the
// `n` arguments the grammar parsed.
function jCallArgTys(text, n) {
    var p = text.indexOf("(")
    if (p < 0) { return null }
    var args = jSplitArgs(text.substring(p + 1, jSkipGroup(text, p) - 1))
    if (args.length != n) { return null }
    var out = []
    for (var i = 0; i < n; i++) { out.push(jStaticTy(args[i])) }
    return out
}
// A few java.lang results a call site reads often.
var jLibRetTys = {
    "$Integer.valueOf": "Integer", "$Long.valueOf": "Long", "$Double.valueOf": "Double",
    "$Float.valueOf": "Float", "$Short.valueOf": "Short", "$Byte.valueOf": "Byte",
    "$Character.valueOf": "Character", "$Boolean.valueOf": "Boolean", "$String.valueOf": "String",
    "$Integer.parseInt": "int", "$Long.parseLong": "long", "$Double.parseDouble": "double",
    "$String.length": "int", "$String.charAt": "char", "$String.indexOf": "int",
    "$String.equals": "boolean", "$String.isEmpty": "boolean", "$String.substring": "String",
    "$String.trim": "String", "$String.toUpperCase": "String", "$String.toLowerCase": "String",
    "$String.concat": "String", "$String.repeat": "String", "$String.toString": "String"
}
// The static type of an expression's whitespace-free text, "" when unknown.
// "null" is the null literal's type and "fn" a lambda's or method
// reference's, which only a functional interface accepts.
function jStaticTy(t) {
    if (t.length == 0) { return "" }
    var pt = jExprTy(t)
    if (pt != "") { return pt }
    if (t == "null") { return "null" }
    if (t == "true" || t == "false") { return "boolean" }
    if (jTopFind(t, "->") >= 0 || jTopFind(t, "::") >= 0) { return "fn" }
    if (jTopFind(t, "?") >= 0) { return jCondTy(t) }
    if (jTopOp(t, 1) >= 0) { return jConcatTy(t) }
    var c0 = t.charAt(0)
    if (c0 == "\"") { return (t.charAt(1) != "\"" && jSkipQuote(t, 0) == t.length) ? "String" : "" }
    if (c0 == "(") {
        var g = jSkipGroup(t, 0)
        var inner = t.substring(1, g - 1)
        if (g == t.length) { return jStaticTy(inner) }
        // A cast: a type in the parentheses and an operand after them.
        if (jIsTypeText(inner) && (jIsIdChar(t.charCodeAt(g)) || "\"'(!~".indexOf(t.charAt(g)) >= 0)) { return inner }
        return ""
    }
    var nt = jNewTy(t)
    if (nt != "") { return nt }
    return jChainTy(t)
}
// `new T(...)`, `new T(...){...}`, `new T[n]...` and `new T[]{...}` - the
// text reads "newT(...)" once its whitespace is gone, so a method of that
// name is not mistaken for one.
function jNewTy(t) {
    if (t.length < 5 || t.substring(0, 3) != "new") { return "" }
    var e = jIdentEnd(t, 3)
    if (e == 3 || jMethodTys["$" + t.substring(0, e)] != undefined) { return "" }
    while (t.charAt(e) == "." && jIdentEnd(t, e + 1) > e + 1) { e = jIdentEnd(t, e + 1) }
    var ty = t.substring(3, e)
    if (t.charAt(e) == "<") {
        var a = jSkipAngle(t, e)
        if (a < 0) { return "" }
        ty = ty + t.substring(e, a)
        e = a
    }
    if (t.charAt(e) == "(") {
        e = jSkipGroup(t, e)
        if (t.charAt(e) == "{") { e = jSkipGroup(t, e) }
        return e == t.length ? ty : ""
    }
    var dims = ""
    while (t.charAt(e) == "[") {
        e = jSkipGroup(t, e)
        dims = dims + "[]"
    }
    if (dims == "") { return "" }
    if (t.charAt(e) == "{") { e = jSkipGroup(t, e) }
    return e == t.length ? ty + dims : ""
}
// A name or a call, then `.f`, `.m(...)` and `[i]` suffixes. A class name
// followed by a member is a static access.
function jChainTy(t) {
    var i = jIdentEnd(t, 0)
    if (i == 0) { return "" }
    var nm = t.substring(0, i)
    var ty = ""
    var isCls = false
    if (t.charAt(i) == "(") {
        ty = jClsMethodTyOf(curClassName, nm)
        i = jSkipGroup(t, i)
    } else {
        var lt = jTyOf(nm)
        if (lt != undefined) { ty = lt }
        else if (jFieldTys["$" + nm] != undefined) { ty = jOwnFieldTy(nm) }
        else if (jIsClassName(nm)) { ty = nm; isCls = true }
    }
    while (i < t.length && ty != "") {
        var c = t.charAt(i)
        if (c == "[") {
            if (isCls) { return "" }
            ty = jElemTy(ty)
            i = jSkipGroup(t, i)
        } else if (c == ".") {
            var e = jIdentEnd(t, i + 1)
            if (e == i + 1) { return "" }
            var mn = t.substring(i + 1, e)
            i = e
            if (t.charAt(i) == "(") {
                i = jSkipGroup(t, i)
                ty = jMemberRetTy(ty, mn)
            } else if (isCls) {
                ty = jOvCls["$" + ty] != undefined ? jFieldTyOf(mn) : ""
            } else if (mn == "length" && jElemTy(ty) != "") {
                ty = "int"
            } else {
                ty = jFieldTyOf(mn)
            }
            isCls = false
        } else {
            return ""
        }
    }
    if (isCls) { return "" }
    return ty
}
// The result type of `recv.mn(...)`: java.lang's from jLibRetTys, a declared
// class' or interface's from the method's declared result type.
function jMemberRetTy(ty, mn) {
    var b = jOvBare(ty)
    var lib = jLibRetTys["$" + b + "." + mn]
    if (lib != undefined) { return lib }
    if (jOvCls["$" + b] != undefined || jOvIfaces["$" + b] == true) {
        var rt = jClsMethodTyOf(b, mn)
        var tps = jClsTyParams["$" + b]
        if (tps != undefined) {
            var targs = jTyArgs(ty)
            for (var i = 0; i < tps.length && i < targs.length; i++) {
                if (tps[i] == rt) { return targs[i] }
            }
        }
        return rt
    }
    return ""
}
// A type name a static access can start with: a declared type, or a
// java.lang one this subset provides.
function jIsClassName(nm) {
    if (jOvCls["$" + nm] != undefined || jOvIfaces["$" + nm] == true) { return true }
    return ",String,Integer,Long,Double,Float,Short,Byte,Character,Boolean,Math,StrictMath,System,".indexOf("," + nm + ",") >= 0
}
// Is `t` the text of a type: a (qualified) name with optional type
// arguments and array brackets?
function jIsTypeText(t) {
    var e = jIdentEnd(t, 0)
    if (e == 0) { return false }
    while (t.charAt(e) == "." && jIdentEnd(t, e + 1) > e + 1) { e = jIdentEnd(t, e + 1) }
    if (t.charAt(e) == "<") { e = jSkipAngle(t, e) }
    if (e < 0) { return false }
    while (t.substring(e, e + 2) == "[]") { e = e + 2 }
    return e == t.length
}
// The index just past the <...> that opens at i, -1 when it does not close
// over type-argument characters only.
function jSkipAngle(t, i) {
    var depth = 0
    for (var j = i; j < t.length; j++) {
        var c = t.charAt(j)
        if (c == "<") { depth++ }
        else if (c == ">") {
            depth--
            if (depth == 0) { return j + 1 }
        } else if (!jIsIdChar(t.charCodeAt(j)) && ",.?[]".indexOf(c) < 0) { return -1 }
    }
    return -1
}
// The first top-level occurrence of `s` in t (outside groups and literals).
function jTopFind(t, s) {
    var i = 0
    while (i < t.length) {
        var c = t.charAt(i)
        if (c == "(" || c == "[" || c == "{") { i = jSkipGroup(t, i) }
        else if (c == "\"" || c == "'") { i = jSkipQuote(t, i) }
        else if (t.substring(i, i + s.length) == s) { return i }
        else { i++ }
    }
    return -1
}
// The first top-level binary operator at or after `from`, -1 when there is
// none: a sign or `!` / `~` at the start or after another operator is unary.
function jTopOp(t, from) {
    var i = 0
    var prevOp = true
    while (i < t.length) {
        var c = t.charAt(i)
        if (c == "(" || c == "[" || c == "{") { i = jSkipGroup(t, i); prevOp = false }
        else if (c == "\"" || c == "'") { i = jSkipQuote(t, i); prevOp = false }
        else if ("+-*/%&|^<>=!?:~".indexOf(c) >= 0) {
            if (!prevOp && i >= from) { return i }
            prevOp = true
            i++
        } else {
            prevOp = false
            i++
        }
    }
    return -1
}
// The two arms' texts of `c ? a : b`, null when t is no conditional.
function jCondArms(t) {
    var q = jTopFind(t, "?")
    if (q < 0) { return null }
    var rest = t.substring(q + 1)
    var depth = 0
    var i = 0
    var col = -1
    while (i < rest.length && col < 0) {
        var c = rest.charAt(i)
        if (c == "(" || c == "[" || c == "{") { i = jSkipGroup(rest, i) }
        else if (c == "\"" || c == "'") { i = jSkipQuote(rest, i) }
        else {
            if (c == "?") { depth++ }
            if (c == ":") { if (depth == 0) { col = i } else { depth-- } }
            i++
        }
    }
    if (col < 0) { return null }
    return {a: rest.substring(0, col), b: rest.substring(col + 1)}
}
// `c ? a : b` (JLS 15.25): a numeric conditional's type (jCondNumTy), the
// arms' common type when one arm's type is the other's or a supertype of it,
// "" for any other pair (a boxing, a least upper bound).
function jCondTy(t) {
    var arms = jCondArms(t)
    if (arms == null) { return "" }
    var a = jStaticTy(arms.a)
    var b = jStaticTy(arms.b)
    if (a == b) { return a }
    var nt = jCondNumTy(a, b, arms.b, arms.a)
    if (nt != "") { return nt }
    var sa = jOvSTok(a)
    var sb = jOvSTok(b)
    if (sa == "" || sb == "" || jOvIsPrim(sa) || jOvIsPrim(sb)) { return "" }
    if (sa == "null") { return b }
    if (sb == "null") { return a }
    if (jOvSAccepts(sb, sa, 1, jOvBare(b))) { return b }
    if (jOvSAccepts(sa, sb, 1, jOvBare(a))) { return a }
    return ""
}
// The type of a conditional whose arms have the different static types a and
// b, each a primitive or its box (JLS 15.25.2): byte and short make short; a
// byte, short or char arm with an int CONSTANT arm representable in it keeps
// its type; boolean arms are boolean; otherwise binary numeric promotion.
// btext and atext are the arms' texts, b's first. "" for any other pair.
function jCondNumTy(a, b, btext, atext) {
    var ua = jIsPrimTy(a) || a == "boolean" ? a : jOvUnbox(a)
    var ub = jIsPrimTy(b) || b == "boolean" ? b : jOvUnbox(b)
    if (ua == "" || ub == "") { return "" }
    if (ua == "boolean" || ub == "boolean") { return ua == ub ? "boolean" : "" }
    if (ua == ub) { return ua }
    if ((ua == "byte" && ub == "short") || (ua == "short" && ub == "byte")) { return "short" }
    if (jCondFits(a, b, btext)) { return a }
    if (jCondFits(b, a, atext)) { return b }
    return jPromote(ua, ub)
}
// Is the arm of type ty and text `text` an int constant representable in the
// primitive narrow type nt (byte, short or char)?
function jCondFits(nt, ty, text) {
    if (ty != "int" || (nt != "byte" && nt != "short" && nt != "char")) { return false }
    var neg = text.charAt(0) == "-"
    var from = neg ? 1 : 0
    var lit = jNumLit(text, from)
    if (lit.end != text.length || lit.ty != "int" || text.charAt(from) == "0" && text.length > from + 1) { return false }
    var v = 0
    for (var i = from; i < text.length; i++) {
        if (text.charAt(i) != "_") { v = v * 10 + text.charCodeAt(i) - 48 }
    }
    if (neg) { v = -v }
    if (nt == "char") { return v >= 0 && v <= 65535 }
    if (nt == "short") { return v >= -32768 && v <= 32767 }
    return v >= -128 && v <= 127
}
// The primitive type `c ? a : b` (its whitespace-free text) converts both
// arms to, with the arms that need the conversion: {ty, a, b}, a and b the
// type to convert that arm to or "" when its static type already is ty.
function jCondConv(t) {
    var none = {ty: "", a: "", b: ""}
    var arms = jCondArms(t)
    if (arms == null) { return none }
    var a = jStaticTy(arms.a)
    var b = jStaticTy(arms.b)
    if (a == b) { return none }
    var nt = jCondNumTy(a, b, arms.b, arms.a)
    if (!jIsPrimTy(nt)) { return none }
    return {ty: nt, a: a == nt ? "" : nt, b: b == nt ? "" : nt}
}
// May the condition written as t (whitespace-free) be a Boolean box, which
// unboxes on the way and makes a null an NPE? Its static type is Boolean, or
// it is unknown and t is a primary (a name, a call, a field, a cast) rather
// than an operator's boolean result.
function jCondMayBox(t) {
    var ty = jStaticTy(t)
    if (ty != "") { return ty == "Boolean" }
    return jTopOp(t, 1) < 0 && t.charAt(0) != "!" && jTopFind(t, "?") < 0
}
// Which operand of `a == b` / `a != b` (its whitespace-free text) unboxes:
// a box whose other operand is a primitive (JLS 15.21.1, 15.21.2).
function jEqUnbox(t) {
    var at = jTopFind(t, "==")
    if (at < 0) { at = jTopFind(t, "!=") }
    if (at < 0) { return {l: false, r: false} }
    var a = jStaticTy(t.substring(0, at))
    var b = jStaticTy(t.substring(at + 2))
    var pa = jIsPrimTy(a) || a == "boolean"
    var pb = jIsPrimTy(b) || b == "boolean"
    return {l: pb && jOvUnbox(a) != "", r: pa && jOvUnbox(b) != ""}
}
// A top-level `+` chain is a String concatenation when any operand is a
// String (JLS 15.18.1); any other top-level operator makes it unknown.
function jConcatTy(t) {
    var parts = []
    var from = 0
    var i = jTopOp(t, 1)
    while (i >= 0) {
        if (t.charAt(i) != "+" || t.charAt(i + 1) == "+" || t.charAt(i + 1) == "=") { return "" }
        parts.push(t.substring(from, i))
        from = i + 1
        var nx = jTopOp(t.substring(from), 1)
        i = nx < 0 ? -1 : from + nx
    }
    parts.push(t.substring(from))
    var str = false
    for (var k = 0; k < parts.length; k++) {
        if (jStaticTy(parts[k]) == "String") { str = true }
    }
    return str ? "String" : ""
}
// A static type as the token a spec entry is matched with (jOvTok's
// vocabulary), "" when it is not one this can judge: a type variable or a
// library type. A declared interface is "#Name", which no class token takes.
function jOvSTok(ty) {
    if (ty == "null" || ty == "fn" || ty == "") { return ty }
    var b = jOvBare(ty)
    if (jOvIsArr(b)) {
        var et = jOvSTok(b.substring(0, b.length - 2))
        return (et == "" || et == "null" || et == "fn") ? "" : et + "[]"
    }
    if (jOvIsPrim(b) || jOvCore.indexOf("," + b + ",") >= 0) { return b }
    if (jOvIfaces["$" + b] == true) { return "#" + b }
    if (jOvCls["$" + b] != undefined) { return b }
    var er = jTyVarErasure(b)
    if (er != "") { return jOvSTok(er) }
    return ""
}
// The primitive a box unboxes to (JLS 5.1.8), "" for anything else.
function jOvUnbox(s) {
    var p = ",Integer:int,Long:long,Double:double,Float:float,Short:short,Byte:byte,Character:char,Boolean:boolean,"
    var at = p.indexOf("," + s + ":")
    if (at < 0) { return "" }
    var from = at + s.length + 2
    return p.substring(from, p.indexOf(",", from))
}
// The java.lang and java.util interfaces a parameter is commonly declared as,
// each with the java.lang types this subset models that implement it ("fn"
// for a functional interface). jOvTok reads all of them as "*"; a name not
// listed (a type variable, most often) takes any reference.
var jLibIfaces = {
    "$Comparable": ",String,Integer,Long,Double,Float,Short,Byte,Character,Boolean,",
    "$CharSequence": ",String,",
    "$Serializable": ",String,Integer,Long,Double,Float,Short,Byte,Character,Boolean,Number,",
    "$Runnable": ",fn,", "$Comparator": ",fn,", "$Iterable": ",fn,", "$Callable": ",fn,",
    "$Supplier": ",fn,", "$Consumer": ",fn,", "$BiConsumer": ",fn,", "$Function": ",fn,",
    "$BiFunction": ",fn,", "$Predicate": ",fn,", "$BiPredicate": ",fn,",
    "$UnaryOperator": ",fn,", "$BinaryOperator": ",fn,",
    "$Collection": ",", "$List": ",", "$Map": ",", "$Set": ",", "$Iterator": ","
}
// Does a parameter of the library-or-variable type `raw` (token "*") take an
// argument of static token s?
function jOvLibAccepts(raw, s) {
    var impl = jLibIfaces["$" + raw]
    if (impl == undefined) { impl = jSamLib["$" + raw] != undefined ? ",fn," : undefined }
    if (impl == undefined) { return true }
    if (s == "fn" || jOvCore.indexOf("," + s + ",") >= 0) { return impl.indexOf("," + s + ",") >= 0 }
    return !jOvIsArr(s)
}
// Can an argument of static token s be passed to a parameter of token t in
// this phase (1: subtyping and widening, 2: also boxing and unboxing)? raw is
// the parameter's declared type, which a "*" token is judged by.
function jOvSAccepts(t, s, phase, raw) {
    if (s == "null") { return !jOvIsPrim(t) }
    if (jOvIsArr(t)) {
        if (!jOvIsArr(s)) { return false }
        var et = t.substring(0, t.length - 2)
        var es = s.substring(0, s.length - 2)
        if (jOvIsPrim(et) || jOvIsPrim(es)) { return et == es }
        return jOvSAccepts(et, es, 1, raw.substring(0, raw.length - 2))
    }
    if (jOvIsArr(s)) { return t == "Object" || (t == "*" && jOvLibAccepts(raw, s)) }
    if (s == "fn") { return t == "*" && jOvLibAccepts(raw, s) }
    if (jOvIsPrim(t)) {
        if (jOvIsPrim(s)) { return jOvWiden(s, t) }
        if (phase < 2) { return false }
        var u = jOvUnbox(s)
        return u != "" && jOvWiden(u, t)
    }
    if (jOvIsPrim(s)) {
        if (phase < 2) { return false }
        if (t == "*") { return jOvLibAccepts(raw, jOvBox(s)) }
        if (t == "Object") { return true }
        if (t == "Number") { return s != "char" && s != "boolean" }
        return t == jOvBox(s)
    }
    if (t == "*") { return jOvLibAccepts(raw, s) }
    if (t == "Object" || t == s) { return true }
    return jOvSub(s, t)
}
// jOvPick's three phases over static tokens; "" when no entry applies.
function jOvPickStatic(spec, stoks) {
    var ents = spec.split("|")
    var n = stoks.length
    for (var phase = 1; phase <= 3; phase++) {
        for (var e = 0; e < ents.length; e++) {
            var at = ents[e].indexOf("@")
            var toks = at == 0 ? [] : ents[e].substring(0, at).split(",")
            var raws = at == 0 ? [] : ents[e].substring(at + 1).split(",")
            var p = toks.length
            var isVar = p > 0 && toks[p - 1].indexOf("...") > 0
            var ok = true
            if (phase < 3) {
                if (p != n) { continue }
                for (var i = 0; i < p && ok; i++) {
                    var t = (isVar && i == p - 1) ? toks[i].substring(0, toks[i].length - 3) + "[]" : toks[i]
                    var r = (isVar && i == p - 1) ? raws[i].substring(0, raws[i].length - 3) + "[]" : raws[i]
                    ok = jOvSAccepts(t, stoks[i], phase, r)
                }
            } else {
                if (!isVar || n < p - 1) { continue }
                for (var j = 0; j < n && ok; j++) {
                    var tv = j < p - 1 ? toks[j] : toks[p - 1].substring(0, toks[p - 1].length - 3)
                    var rv = j < p - 1 ? raws[j] : raws[p - 1].substring(0, raws[p - 1].length - 3)
                    ok = jOvSAccepts(tv, stoks[j], 2, rv)
                }
            }
            if (ok) { return ents[e].substring(at + 1) }
        }
    }
    return ""
}
// The signature a call of `mname` on a receiver of static type rty resolves
// to, "" when the call keeps the runtime pick: the receiver type is unknown
// or not overloaded on mname, an argument type is unknown, or the
// candidates' static-ness is not the receiver's (wantStatic: true for a
// class name, false for an instance, null for an unqualified call).
function jOvStaticSig(rty, mname, atys, wantStatic) {
    if (atys == null || rty == "") { return "" }
    var b = jOvBare(rty)
    var cands = jOvCands(b, mname)
    if (cands == null) { cands = jOvIfaceCands(b, mname) }
    if (cands == null) { return "" }
    if (wantStatic != null && cands[0].statik != wantStatic) { return "" }
    // A candidate declared over a type variable (List<E>.add(int, E)) is
    // overridden under its SUBSTITUTED signature (add(int, String)), which no
    // "name#sig" slot connects to it, so such a call keeps the runtime pick.
    // A static method is never overridden, so its slot is the declared one.
    for (var c = 0; c < cands.length && cands[0].statik != true; c++) {
        for (var p = 0; p < cands[c].ptys.length; p++) {
            var pb = jOvBare(cands[c].ptys[p])
            while (jOvIsArr(pb)) { pb = pb.substring(0, pb.length - 2) }
            if (jTyVarErasure(pb) != "") { return "" }
        }
    }
    var stoks = []
    for (var i = 0; i < atys.length; i++) {
        var s = jOvSTok(atys[i])
        if (s == "") { return "" }
        stoks.push(s)
    }
    return jOvPickStatic(jOvSpec(cands), stoks)
}
// The receiver of a member call, from the text before its `.name(`: a class
// name is a static access ("static:" + name), anything else jStaticTy's type.
// A bare name that is no local is kept as "static?:" + name: a class declared
// further down is not yet known when the walk passes the call, so jSfxSig
// decides once the walk is complete.
function jRecvTy(t) {
    var e = jIdentEnd(t, 0)
    if (e == t.length && e > 0 && jTyOf(t) == undefined && jFieldTys["$" + t] == undefined) {
        if (jIsClassName(t)) { return "static:" + t }
        return "static?:" + t
    }
    return jStaticTy(t)
}
// A `.name(args)` suffix record, with its text and its arguments' static types.
function jMcallSfx(name, args, text) {
    if (name == "stream") { jStreamWant = true }
    return {kind: "mcall", name: name, args: args, stext: text, atys: jCallArgTys(text, args.length)}
}
// Gives each `.m(...)` suffix of a member chain `rty`, its receiver's static
// type: jRecvTy of the chain's text up to that suffix, measured by taking the
// suffixes' own texts off its end. A suffix that has no text to measure makes
// every receiver before it unknown.
function jChainRecvTys(sfxs, text) {
    var starts = []
    for (var k = 0; k < sfxs.length; k++) { starts.push(-1) }
    var end = text.length
    for (var i = sfxs.length - 1; i >= 0; i--) {
        var s = sfxs[i]
        var st = ""
        if (s.kind == "field") { st = "." + s.name } else if (s.stext != undefined) { st = s.stext } else { break }
        if (end < st.length || text.substring(end - st.length, end) != st) { break }
        end = end - st.length
        starts[i] = end
    }
    for (var j = 0; j < sfxs.length; j++) {
        if (sfxs[j].kind == "mcall") { sfxs[j].rty = starts[j] > 0 ? jRecvTy(text.substring(0, starts[j])) : "" }
    }
}
// The static signature of a `.m(...)` suffix, resolved once the walk is
// complete and cached on the record; "" keeps the runtime pick.
function jSfxSig(s) {
    if (s.sig == undefined) {
        var rty = s.rty
        if (rty != undefined && rty.length > 8 && rty.substring(0, 8) == "static?:") {
            var nm = rty.substring(8)
            rty = (jFieldTys["$" + nm] == undefined && jIsClassName(nm)) ? "static:" + nm : ""
        }
        s.sig = (rty == undefined || rty == "") ? "" : jOvMemberSig(rty, s.name, s.atys)
    }
    return s.sig
}
// jOvStaticSig for a member call whose receiver jRecvTy read as rty.
function jOvMemberSig(rty, mname, atys) {
    if (rty.length > 7 && rty.substring(0, 7) == "static:") { return jOvStaticSig(rty.substring(7), mname, atys, true) }
    return jOvStaticSig(rty, mname, atys, false)
}
