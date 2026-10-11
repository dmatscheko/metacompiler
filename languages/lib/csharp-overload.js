// csharp-overload.js -- ECMA-334 12.6.4's choice among applicable overloads,
// for both csharp grammars (include). An argument VALUE stands in for its static
// type, so an overload applies when each argument's run-time type is the
// parameter type or converts to it implicitly (10.2.3 numeric widening, 10.2.8
// reference conversion to a base class or object), and among the applicable
// ones the better function member (12.6.4.3) wins: its parameter types are the
// better conversion targets (12.6.4.7) - so O(string) beats O(object) for a
// string, F(long) beats F(double) for an int, G(B) beats G(A) for a C : B : A.
// The class hierarchy is each grammar's csSupers (class name -> base name).

// A parameter type as the keyword it aliases (ECMA-334 8.2.1): System.Int32
// and Int32 are int.
var csxAlias = {"$String": "string", "$Object": "object", "$Int32": "int", "$Int64": "long", "$Int16": "short",
                "$UInt16": "ushort", "$UInt32": "uint", "$UInt64": "ulong", "$Byte": "byte", "$SByte": "sbyte",
                "$Double": "double", "$Single": "float", "$Char": "char", "$Boolean": "bool", "$Decimal": "decimal"}
function csxNorm(t) {
    var s = t
    if (s.length > 7 && s.substring(0, 7) == "System.") { s = s.substring(7, s.length) }
    var al = csxAlias["$" + s]
    if (al != undefined) { return al }
    return s
}
// The implicit numeric conversions, by source type (ECMA-334 10.2.3).
var csxNumTo = {
    "$sbyte": ",short,int,long,float,double,decimal,",
    "$byte": ",short,ushort,int,uint,long,ulong,float,double,decimal,",
    "$short": ",int,long,float,double,decimal,",
    "$ushort": ",int,uint,long,ulong,float,double,decimal,",
    "$int": ",long,float,double,decimal,",
    "$uint": ",long,ulong,float,double,decimal,",
    "$long": ",float,double,decimal,",
    "$ulong": ",float,double,decimal,",
    "$char": ",ushort,int,uint,long,ulong,float,double,decimal,",
    "$float": ",double,"
}
var csxNumNames = ["sbyte", "byte", "short", "ushort", "int", "uint", "long", "ulong", "char", "float"]
// The numeric types that convert implicitly to t (t itself excluded).
function csxWidenFrom(t) {
    var out = []
    for (var i = 0; i < csxNumNames.length; i++) {
        if (csxNumTo["$" + csxNumNames[i]].indexOf("," + t + ",") >= 0) { out.push(csxNumNames[i]) }
    }
    return out
}
// Does type a convert implicitly to type b (identity included)?
function csxConv(a, b) {
    if (a == b) { return true }
    if (b == "object" || b == "Object") { return true }
    var nt = csxNumTo["$" + a]
    if (nt != undefined) { return nt.indexOf("," + b + ",") >= 0 }
    var c = a
    var guard = 0
    while (guard < 64) {
        guard = guard + 1
        var s = csSupers[c]
        if (typeof s != "string" || s == "") { return false }
        if (s == b) { return true }
        c = s
    }
    return false
}
function csxSigned(t) { return t == "sbyte" || t == "short" || t == "int" || t == "long" }
function csxUnsigned(t) { return t == "byte" || t == "ushort" || t == "uint" || t == "ulong" }
// Is a at least as good a conversion target as b (12.6.4.7)?
function csxAsGood(a, b) {
    if (a == b) { return true }
    var ab = csxConv(a, b)
    var ba = csxConv(b, a)
    if (ab && !ba) { return true }
    return !ab && !ba && csxSigned(a) && csxUnsigned(b)
}
// Is the parameter list ta more specific than tb?
function csxMore(ta, tb) {
    if (ta.length != tb.length) { return false }
    var strict = false
    for (var k = 0; k < ta.length; k++) {
        if (!csxAsGood(ta[k], tb[k])) { return false }
        if (ta[k] != tb[k]) { strict = true }
    }
    return strict
}
// The pass-1 order of a csCtorSigs list (both grammars build the same records:
// tys, typed). The first `own` candidates are the receiver class' own and are
// tried before the inherited rest, each part in its own order: an applicable
// method of a derived class hides the base's (12.6.4.1).
function csxSigOrder(sigs, own) {
    var out = csxPartOrder(sigs, 0, own)
    var inh = csxPartOrder(sigs, own, sigs.length)
    for (var i = 0; i < inh.length; i++) { out.push(inh[i]) }
    return out
}
function csxPartOrder(sigs, from, to) {
    var tys = []
    var typed = []
    for (var i = from; i < to; i++) {
        var ts = []
        for (var k = 0; k < sigs[i].tys.length; k++) { ts.push(csxNorm(sigs[i].tys[k])) }
        tys.push(ts)
        typed.push(sigs[i].typed)
    }
    var ord = csxOrder(tys, typed)
    for (var j = 0; j < ord.length; j++) { ord[j] = ord[j] + from }
    return ord
}
// The numeric types an argument of which a parameter of type t also accepts.
function csxWidenOf(t) { return csxWidenFrom(csxNorm(t)) }

// ----- closed constructed types -----
// A generic class keeps its static fields once per CLOSED type (ECMA-334
// 15.5.2), so `Gen<int>.N` and `Gen<long>.N` are two variables. A closed type
// is keyed by its type argument list with the aliases folded: 'Gen<int>' and
// 'Gen<Int32>' -> "int". "" when the text is not exactly Name<args>.
function csxClosedKey(text) {
    var t = stripWs(text)
    var a = t.indexOf("<")
    if (a <= 0 || t.charAt(t.length - 1) != ">" || t.substring(0, a).indexOf(".") >= 0) { return "" }
    var depth = 0
    var cur = ""
    var parts = []
    for (var i = a; i < t.length; i++) {
        var ch = t.charAt(i)
        if (ch == "<") {
            depth++
            if (depth == 1) { continue }
        } else if (ch == ">") {
            depth--
            if (depth == 0) {
                if (i != t.length - 1) { return "" }
                parts.push(csxNorm(cur))
                break
            }
        } else if (ch == "," && depth == 1) {
            parts.push(csxNorm(cur))
            cur = ""
            continue
        }
        cur += ch
    }
    return parts.join(",")
}
// The key of a type argument list already split ('new Gen<int>()').
function csxKeyOf(targs) {
    var parts = []
    for (var i = 0; i < targs.length; i++) { parts.push(csxNorm(targs[i])) }
    return parts.join(",")
}
// Whether a key mentions one of the type parameters tps: type arguments are
// erased, so 'Gen<T>' in a generic body is not a closed type here and names the
// class whose body it stands in.
function csxKeyNamesAny(key, tps) {
    if (tps == undefined || tps == null) { return false }
    var id = ""
    for (var i = 0; i <= key.length; i++) {
        var ch = (i < key.length) ? key.charAt(i) : ","
        if ((ch >= "a" && ch <= "z") || (ch >= "A" && ch <= "Z") || (ch >= "0" && ch <= "9") || ch == "_") {
            id += ch
            continue
        }
        for (var j = 0; j < tps.length; j++) { if (tps[j] == id) { return true } }
        id = ""
    }
    return false
}
// The order the exact-arity candidates are tried in: most specific first, and
// declaration order where neither is better, so the first applicable one is
// the better function member whenever there is one. `tys[i]` is candidate i's
// parameter types, `typed[i]` whether it takes part.
function csxOrder(tys, typed) {
    var rest = []
    for (var i = 0; i < tys.length; i++) { rest.push(i) }
    var out = []
    while (rest.length > 0) {
        var pick = 0
        for (var r = 0; r < rest.length; r++) {
            var beaten = false
            for (var q = 0; q < rest.length; q++) {
                if (q != r && typed[rest[q]] && typed[rest[r]] && csxMore(tys[rest[q]], tys[rest[r]])) { beaten = true }
            }
            if (!beaten) {
                pick = r
                break
            }
        }
        out.push(rest[pick])
        var left = []
        for (var l = 0; l < rest.length; l++) { if (l != pick) { left.push(rest[l]) } }
        rest = left
    }
    return out
}
