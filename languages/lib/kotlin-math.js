// kotlin-math.js -- kotlin.math on doubles, as Kotlin/JVM's MathJVM.kt implements
// it over java.lang.Math, for kotlin-interpreter.abnf (include) and
// languages/lib/kotlin-rt.metajs (import). The transcendentals are StrictMath's
// fdlibm (lib/fdlibm.js, loaded by the includer first), as java.lang.Math is in
// this project. Each engine boxes the answer itself (a Float argument answers a
// Float). Go twin: abnf/jsrtktmath.go.
//
// Written in the subset both engines accept: no exponent literals, and a negative
// zero is spelled 0 / (0 - 1).

// The top-level names: functions, then the two constants.
var ktmNames = ["sqrt", "cbrt", "exp", "expm1", "ln", "ln1p", "log10", "log2", "log",
                "sin", "cos", "tan", "asin", "acos", "atan", "atan2", "sinh", "cosh",
                "tanh", "asinh", "acosh", "atanh", "hypot", "floor", "ceil", "truncate",
                "round", "sign", "PI", "E"]

function ktmIsName(n) {
    var i = 0
    while (i < ktmNames.length) {
        if (ktmNames[i] == n) { return true }
        i = i + 1
    }
    return false
}

// The functions of two arguments.
function ktmIsBinary(n) { return n == "atan2" || n == "hypot" || n == "log" }

// MathJVM.kt's constants: LN2 is Math.log(2.0); the Taylor bounds are the square
// root of ulp(1.0) (2^-26), its square root (2^-13) and their reciprocals.
var ktmTaylor2 = 1 / 67108864
var ktmTaylorN = 1 / 8192

function ktmLn2() { return fdlibm().log(2) }

function ktmAsinh(x) {
    if (x >= ktmTaylorN) {
        if (x > 8192) {
            if (x > 67108864) { return fdlibm().log(x) + ktmLn2() }
            return fdlibm().log(x * 2 + 1 / (x * 2))
        }
        return fdlibm().log(x + Math.sqrt(x * x + 1))
    }
    if (x <= 0 - ktmTaylorN) { return -ktmAsinh(-x) }
    var r = x
    if (Math.abs(x) >= ktmTaylor2) { r = r - x * x * x / 6 }
    return r
}

function ktmAcosh(x) {
    if (x < 1) { return 0 / 0 }
    if (x > 67108864) { return fdlibm().log(x) + ktmLn2() }
    if (x - 1 >= ktmTaylorN) { return fdlibm().log(x + Math.sqrt(x * x - 1)) }
    var y = Math.sqrt(x - 1)
    var r = y
    if (y >= ktmTaylor2) { r = r - y * y * y / 12 }
    return Math.sqrt(2) * r
}

function ktmAtanh(x) {
    if (Math.abs(x) < ktmTaylorN) {
        var r = x
        if (Math.abs(x) > ktmTaylor2) { r = r + x * x * x / 3 }
        return r
    }
    return fdlibm().log((1 + x) / (1 - x)) / 2
}

// Math.rint: the nearest integer, a tie to the EVEN one, the sign of a zero kept
// (rint(-0.4) is -0.0). From 2^52 up every double is already integral.
function ktmRint(x) {
    if (x !== x) { return x }
    if (x >= 4503599627370496 || x <= 0 - 4503599627370496) { return x }
    var f = Math.floor(x)
    var d = x - f
    var r = f
    if (d > 0.5) { r = f + 1 } else if (d == 0.5 && f % 2 != 0) { r = f + 1 }
    if (r == 0 && (x < 0 || 1 / x < 0)) { return 0 / (0 - 1) }
    return r
}

// Math.round's closest integer with ties toward +Infinity. x - floor(x) is exact,
// so 0.49999999999999994 rounds to 0, where floor(x + 0.5) answers 1.
function ktmRoundHalfUp(x) {
    if (x !== x) { return x }
    if (x >= 4503599627370496 || x <= 0 - 4503599627370496) { return x }
    var f = Math.floor(x)
    if (x - f >= 0.5) { return f + 1 }
    return f
}

// The one-argument function of the name.
function ktmMath1(name, x) {
    if (name == "sqrt") { return Math.sqrt(x) }
    if (name == "floor") { return Math.floor(x) }
    if (name == "ceil") { return Math.ceil(x) }
    if (name == "round") { return ktmRint(x) }
    if (name == "truncate") {
        if (x !== x || x == 1 / 0 || x == -1 / 0) { return x }
        if (x > 0) { return Math.floor(x) }
        return Math.ceil(x)
    }
    // Math.signum: NaN and both zeros answer themselves.
    if (name == "sign") {
        if (x > 0) { return 1 }
        if (x < 0) { return -1 }
        return x
    }
    if (name == "asinh") { return ktmAsinh(x) }
    if (name == "acosh") { return ktmAcosh(x) }
    if (name == "atanh") { return ktmAtanh(x) }
    var fd = fdlibm()
    if (name == "ln") { return fd.log(x) }
    if (name == "log2") { return fd.log(x) / ktmLn2() }
    if (name == "ln1p") { return fd.log1p(x) }
    if (name == "log10") { return fd.log10(x) }
    if (name == "exp") { return fd.exp(x) }
    if (name == "expm1") { return fd.expm1(x) }
    if (name == "cbrt") { return fd.cbrt(x) }
    if (name == "sin") { return fd.sin(x) }
    if (name == "cos") { return fd.cos(x) }
    if (name == "tan") { return fd.tan(x) }
    if (name == "asin") { return fd.asin(x) }
    if (name == "acos") { return fd.acos(x) }
    if (name == "atan") { return fd.atan(x) }
    if (name == "sinh") { return fd.sinh(x) }
    if (name == "cosh") { return fd.cosh(x) }
    return fd.tanh(x)
}

// The two-argument function of the name (pow is the Double.pow extension).
// log(x, base) is NaN for a base that is not positive or is 1, as MathJVM.kt
// checks before dividing.
function ktmMath2(name, x, y) {
    if (name == "atan2") { return fdlibm().atan2(x, y) }
    if (name == "hypot") { return fdlibm().hypot(x, y) }
    if (name == "pow") { return fdlibm().pow(x, y) }
    if (y <= 0 || y == 1) { return 0 / 0 }
    return fdlibm().log(x) / fdlibm().log(y)
}

// withSign is Math.copySign: x's magnitude with s's sign, a NaN s counting as
// positive (java's canonical NaN). ulp / nextUp / nextDown are Math.ulp / nextUp /
// nextDown, at binary32 when f32. Without access to a double's bits they scale by
// powers of two, which is exact. Go twin: ktMathMethod / ktMathProp in
// abnf/jsrtktmath.go.
function ktmWithSign(x, s) {
    var neg = s < 0 || (s == 0 && 1 / s < 0)
    var xneg = x < 0 || (x == 0 && 1 / x < 0)
    if (neg == xneg) { return x }
    return -x
}
function ktmPow2(k) {
    var p = 1
    var i = 0
    while (i < k) { p = p * 2; i = i + 1 }
    while (i > k) { p = p / 2; i = i - 1 }
    return p
}
function ktmMant(f32) { if (f32) { return 23 } return 52 }
function ktmEmin(f32) { if (f32) { return 0 - 126 } return 0 - 1022 }
function ktmMaxF(f32) { return (2 - ktmPow2(0 - ktmMant(f32))) * ktmPow2(0 - ktmEmin(f32) + 1) }
function ktmUlp(x, f32) {
    var a = x
    if (a < 0) { a = -a }
    if (a !== a || a - a !== 0) { return a }
    var p = 1
    var e = 0
    if (a >= 1) {
        while (p * 2 <= a) { p = p * 2; e = e + 1 }
    } else {
        while (p > a && e > ktmEmin(f32)) { p = p / 2; e = e - 1 }
    }
    return ktmPow2(e - ktmMant(f32))
}
// The next value below y > 0: one ulp down, half an ulp from the lowest value of
// a binade above the subnormals.
function ktmDownPos(y, f32) {
    var u = ktmUlp(y, f32)
    if (u * ktmPow2(ktmMant(f32)) == y && y > ktmPow2(ktmEmin(f32))) { return y - u / 2 }
    return y - u
}
function ktmNextUp(x, f32) {
    if (x !== x) { return x }
    if (x - x !== 0) {
        if (x > 0) { return x }
        return -ktmMaxF(f32)
    }
    if (x == 0) { return ktmPow2(ktmEmin(f32) - ktmMant(f32)) }
    if (x > 0) { return x + ktmUlp(x, f32) }
    return -ktmDownPos(-x, f32)
}
function ktmNextDown(x, f32) { return -ktmNextUp(-x, f32) }

// The value of a constant, PI or E.
function ktmConst(name) {
    if (name == "PI") { return 3.141592653589793 }
    return 2.718281828459045
}

// java.lang.Math, which a Kotlin/JVM program reaches as `Math`. These members are a
// kotlin.math function of the same meaning, as java name, kotlin name pairs (rint is
// kotlin's round, signum its sign); each answers a Double whatever its argument.
var ktmJava = ["sqrt", "sqrt", "cbrt", "cbrt", "exp", "exp", "expm1", "expm1", "log", "ln",
               "log10", "log10", "log1p", "ln1p", "sin", "sin", "cos", "cos", "tan", "tan",
               "asin", "asin", "acos", "acos", "atan", "atan", "sinh", "sinh", "cosh", "cosh",
               "tanh", "tanh", "floor", "floor", "ceil", "ceil", "rint", "round",
               "signum", "sign", "atan2", "atan2", "hypot", "hypot", "pow", "pow"]

// Math.toRadians / toDegrees: one multiplication by java's own constant.
function ktmJavaAngle(name, x) {
    if (name == "toRadians") { return x * 0.017453292519943295 }
    return x * 57.29577951308232
}

// java.util.Random, which a Kotlin/JVM program reaches as `java.util.Random(seed)`:
// java's 48-bit linear congruential generator, seed' = seed * 0x5DEECE66D + 0xB
// (mod 2^48), kept as two 24-bit halves so every product stays exact in a double.
// Each engine boxes the answers and raises the errors itself. Go twin: jrnd* in
// abnf/jsrtktmath.go.

// A generator over the seed's 64-bit two's complement words (each 0..2^32-1),
// scrambled as java's constructor does: (seed ^ 0x5DEECE66D) mod 2^48.
function jrndNew(hi32, lo32) {
    var lo = lo32 % 16777216
    var hi = Math.floor(lo32 / 16777216) + (hi32 % 65536) * 256
    return {__jrnd: true, hi: hi ^ 1502, lo: lo ^ 15525485}
}

// next(bits): step the seed, answer its top `bits` bits (an Int, so 32 of them
// read signed).
function jrndNext(box, bits) {
    var sl = box["lo"]
    var sh = box["hi"]
    var lo = sl * 15525485 + 11
    var mid = (sh * 15525485 + sl * 1502) % 16777216
    box["lo"] = lo % 16777216
    box["hi"] = (mid + Math.floor(lo / 16777216)) % 16777216
    var v = Math.floor((box["hi"] * 16777216 + box["lo"]) / Math.pow(2, 48 - bits))
    if (bits == 32 && v >= 2147483648) { v = v - 4294967296 }
    return v
}

// nextInt(bound): a power of two takes the top bits, anything else rejects the
// draws whose 31-bit value would bias the remainder (u - r + m overflows an Int).
function jrndBelow(box, bound) {
    var r = jrndNext(box, 31)
    var m = bound - 1
    if ((bound & m) == 0) { return Math.floor(r / (2147483648 / bound)) }
    var u = r
    r = u % bound
    while (u - r + m > 2147483647) {
        u = jrndNext(box, 31)
        r = u % bound
    }
    return r
}

// nextInt(origin, bound): RandomSupport.boundedNextInt over nextInt(), which
// java.util.Random inherits from RandomGenerator.
function jrndRange(box, origin, bound) {
    var r = jrndNext(box, 32)
    var n = bound - origin
    var m = n - 1
    // A width of 2^31 is a power of two too: java's n overflows to MIN_VALUE there.
    if (n <= 2147483648 && (n & m) == 0) { return (r & m) + origin }
    if (n <= 2147483647) {
        var u = Math.floor((r < 0 ? r + 4294967296 : r) / 2)
        r = u % n
        while (u + m - r > 2147483647) {
            var t = jrndNext(box, 32)
            u = Math.floor((t < 0 ? t + 4294967296 : t) / 2)
            r = u % n
        }
        return r + origin
    }
    while (r < origin || r >= bound) { r = jrndNext(box, 32) }
    return r
}

// One member call, answered as {t, v} for the engine to box: t is "int", "long"
// (v then [hi32, lo32], the 64-bit two's complement words), "double", "float",
// "bool", or "err" with IllegalArgumentException's message; null for a name this
// generator does not answer.
function jrndCall(box, name, args) {
    if (name == "nextInt" && args.length == 0) { return {t: "int", v: jrndNext(box, 32)} }
    if (name == "nextInt" && args.length == 1) {
        if (args[0] <= 0) { return {t: "err", v: "bound must be positive"} }
        return {t: "int", v: jrndBelow(box, args[0])}
    }
    if (name == "nextInt" && args.length == 2) {
        if (args[0] >= args[1]) { return {t: "err", v: "bound must be greater than origin"} }
        return {t: "int", v: jrndRange(box, args[0], args[1])}
    }
    if (name == "nextLong" && args.length == 0) {
        var a = jrndNext(box, 32)
        var b = jrndNext(box, 32)
        var h = b < 0 ? a - 1 : a
        return {t: "long", v: [h < 0 ? h + 4294967296 : h, b < 0 ? b + 4294967296 : b]}
    }
    if (name == "nextDouble" && args.length == 0) {
        var d = jrndNext(box, 26) * 134217728 + jrndNext(box, 27)
        return {t: "double", v: d / 9007199254740992}
    }
    if (name == "nextFloat" && args.length == 0) { return {t: "float", v: jrndNext(box, 24) / 16777216} }
    if (name == "nextBoolean" && args.length == 0) { return {t: "bool", v: jrndNext(box, 1) != 0} }
    return null
}

// The JVM classes of Kotlin's arrays, which a Formatter mismatch names. A host
// array carries a kind number (arrKind): 1 is an Array<T>, and the primitive
// arrays are 5 + their element type's index here. Go twin: ktaElems in
// abnf/jsrtkotlin.go.
var ktaElems = ["Int", "Long", "Double", "Float", "Char", "Boolean", "Byte", "Short",
                "UInt", "ULong", "UByte", "UShort"]
var ktaClasses = ["[I", "[J", "[D", "[F", "[C", "[Z", "[B", "[S",
                  "kotlin.UIntArray", "kotlin.ULongArray", "kotlin.UByteArray", "kotlin.UShortArray"]

// copyOfRange(from, to)'s bounds failure on an array of `size` elements and kind
// `ak`, as [class, message], or null for a valid range. Kotlin checks toIndex
// against the size, java.util.Arrays.copyOfRange then rejects from > to, and
// System.arraycopy a negative from. The last raises ArrayIndexOutOfBoundsException,
// answered as its supertype IndexOutOfBoundsException like every array read here.
// Go twin: ktaCopyRangeFault in abnf/jsrtkotlin.go.
function ktaCopyRangeFault(from, to, size, ak) {
    if (to > size) { return ["IndexOutOfBoundsException", "toIndex (" + to + ") is greater than size (" + size + ")."] }
    if (from > to) { return ["IllegalArgumentException", from + " > " + to] }
    if (from >= 0) { return null }
    var jt = "object array"
    if (ak >= 5) {
        jt = ktaElems[ak - 5]
        if (jt.charAt(0) == "U") { jt = jt.substring(1) }
        jt = jt.toLowerCase()
    }
    return ["IndexOutOfBoundsException", "arraycopy: source index " + from + " out of bounds for " + jt + "[" + size + "]"]
}

// The kind of the primitive array of element type `elem` ("Int"), 0 for none.
function ktaKindOf(elem) {
    var i = 0
    while (i < ktaElems.length) {
        if (ktaElems[i] == elem) { return 5 + i }
        i = i + 1
    }
    return 0
}

// The kind a primitive array builder's answer carries - IntArray(n) and
// intArrayOf(...) alike - 0 for any other name.
function ktaBuilderKind(name) {
    var i = 0
    while (i < ktaElems.length) {
        if (name == ktaElems[i] + "Array" || name == ktaElems[i].toLowerCase() + "ArrayOf") { return 5 + i }
        i = i + 1
    }
    return 0
}

// The kind an array-making member answers on a receiver of kind `recv`: to<E>Array
// its primitive array, toTypedArray an Array<T>, and the copying members (copyOf,
// sortedArray, plus, ...) the receiver's own array kind.
function ktaResultKind(recv, name) {
    if (name == "toTypedArray") { return 1 }
    if (name.length > 7 && name.substring(0, 2) == "to" && name.substring(name.length - 5, name.length) == "Array") {
        var k = ktaKindOf(name.substring(2, name.length - 5))
        if (k != 0) { return k }
    }
    if (recv == 1 || recv >= 5) { return recv }
    return 1
}

// An Array<T>'s class, [L<T>; - T read off the elements' classes (a static type
// argument is not kept): a java.lang class every non-null element shares, else
// java.lang.Object.
function ktaRefClass(classes) {
    var c = ""
    var i = 0
    while (i < classes.length) {
        var e = classes[i]
        if (e.length < 11 || e.substring(0, 10) != "java.lang.") { return "[Ljava.lang.Object;" }
        if (c != "" && c != e) { return "[Ljava.lang.Object;" }
        c = e
        i = i + 1
    }
    if (c == "") { return "[Ljava.lang.Object;" }
    return "[L" + c + ";"
}

// The read-only collection classes a cast to a Mutable type exposes, as the codes
// ktRoRule takes: LB ListBuilder, L1 Collections$SingletonList, LA Arrays$ArrayList,
// SB SetBuilder, S1 Collections$SingletonSet, MB MapBuilder, M1
// Collections$SingletonMap; "" for a mutable collection, and for an EmptyList /
// EmptySet / EmptyMap, which the cast itself refuses. A List's code comes from its
// kind k (-5 a built buildList) and size n; a Set's or a Map's from its JVM class jc
// and whether its builder has finished (sealed).
function ktRoList(k, n) {
    if (k == 0 - 5) { return "LB" }
    if (k == 0 - 4 || (k == 4 && n > 1)) { return "LA" }
    if (k == 4 && n == 1) { return "L1" }
    return ""
}
function ktRoTagged(jc, sealed, isMap) {
    var c = "S"
    var single = "java.util.Collections$SingletonSet"
    if (isMap) {
        c = "M"
        single = "java.util.Collections$SingletonMap"
    }
    if (sealed) { return c + "B" }
    if (jc == single) { return c + "1" }
    return ""
}

// What mutating member `name`, called with na arguments (pred: the first is a
// function), does to the read-only collection of code ro holding n elements: 0 runs
// it, 1 raises UnsupportedOperationException first, 2 runs it on a copy and raises
// when the copy changed size, 3 raises when its argument has elements. A builder's
// mutators call checkIsMutable first (a removeIf / removeFirst, and kotlin's
// removeAll / retainAll of a predicate, only reach one on a hit); the
// java.util classes inherit AbstractList / AbstractCollection / AbstractMap, whose
// add, add(index), removeAt and put always throw and whose removals throw from the
// iterator, on a hit only; Arrays$ArrayList writes in place (set, sort, fill), and
// the Singleton classes override removeIf, replaceAll and Map's default methods to
// throw. A sort or reverse of fewer than two elements writes nothing. Go twin:
// ktRoRule in abnf/jsrtkotlin.go.
var ktRoWriters = " set fill replaceAll sort sortDescending sortBy sortByDescending sortWith reverse shuffle "
var ktRoListAdds = " add addFirst addLast removeAt "
var ktRoBulk = " addAll remove removeAll retainAll clear "
var ktRoRemovals = " removeIf removeFirst removeLast removeFirstOrNull removeLastOrNull "
var ktRoMapPuts = " put set compute computeIfPresent merge replace replaceAll "
var ktRoMapAdds = " getOrPut putIfAbsent computeIfAbsent "
function ktRoRule(ro, name, n, na, pred) {
    var w = " " + name + " "
    if (pred && (name == "removeAll" || name == "retainAll")) { return 2 }
    var builder = ro.charAt(1) == "B"
    var kind = ro.charAt(0)
    if (kind == "L") {
        if (ktRoListAdds.indexOf(w) >= 0) { return 1 }
        if (ktRoWriters.indexOf(w) >= 0) {
            if (ro == "LA") { return 0 }
            if (name == "set") { return 1 }
            if (name == "fill" || name == "replaceAll") { return n > 0 ? 1 : 0 }
            return builder && n > 1 ? 1 : 0
        }
        if (ro == "L1" && name == "removeIf") { return 1 }
        if (ktRoBulk.indexOf(w) >= 0) { return builder ? 1 : 2 }
        return ktRoRemovals.indexOf(w) >= 0 ? 2 : 0
    }
    if (kind == "S") {
        if (name == "add" || (name == "removeIf" && !builder)) { return 1 }
        if (name == "addAll") { return builder ? 1 : 3 }
        if (ktRoBulk.indexOf(w) >= 0) { return builder ? 1 : 2 }
        return name == "removeIf" ? 2 : 0
    }
    if (ktRoMapPuts.indexOf(w) >= 0) { return 1 }
    if (name == "putAll") { return builder ? 1 : 3 }
    // remove(key, value) is Map's default method, which only a SingletonMap overrides.
    if (name == "remove" && na > 1) { return builder ? 2 : 1 }
    if (name == "remove" || name == "clear") { return builder ? 1 : 2 }
    if (ktRoMapAdds.indexOf(w) >= 0) { return builder || name == "getOrPut" ? 2 : 1 }
    return 0
}

// Whether the UnsupportedOperationException a removal hit raises says "remove": an
// Arrays$ArrayList's own iterator refuses so, and remove(element), retainAll,
// removeIf and removeAll(elements) remove through it, where removeAll(predicate) is
// kotlin's in-place filter, which ends in removeAt, and clear in removeRange.
function ktRoSaysRemove(ro, name, pred) {
    if (ro != "LA") { return false }
    if (name == "removeAll") { return !pred }
    return name == "remove" || name == "retainAll" || name == "removeIf"
}
