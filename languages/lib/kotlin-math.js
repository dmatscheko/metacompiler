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

// java.lang.Integer / java.lang.Long statics that are a Kotlin Int / Long member of
// the same meaning applied to the first argument, as java name, kotlin name pairs.
var ktmJavaInt = ["toString", "toString", "bitCount", "countOneBits",
                  "numberOfLeadingZeros", "countLeadingZeroBits",
                  "numberOfTrailingZeros", "countTrailingZeroBits",
                  "highestOneBit", "takeHighestOneBit", "lowestOneBit", "takeLowestOneBit",
                  "rotateLeft", "rotateLeft", "rotateRight", "rotateRight"]

// The digit statics of Integer / Long and their radixes: the two's complement
// digits at the class width (toHexString(-1) is ffffffff).
var ktmJavaIntRadixNames = ["toHexString", "toOctalString", "toBinaryString"]
var ktmJavaIntRadixes = [16, 8, 2]

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

// kotlin.random.Random(seed): kotlin's XorWowRandom (Marsaglia's xorwow), its stream
// value for value. Every value is an Int held as a number, and every step that can
// leave the Int range ends in `| 0`; an unsigned shift is kxwUshr, so no bit
// operator ever sees an operand outside the Int range. The seed's two 32-bit words
// (each 0..2^32-1) are the constructor's seed1 (low) and seed2 (high), as
// Random(seed: Long) passes them; Random(seed: Int) sign-extends, which gives the
// same words. The constructor discards 64 draws. Go twin: kxw* in
// abnf/jsrtktmath.go.
function kxwNew(hi32, lo32) {
    var s1 = lo32 >= 2147483648 ? lo32 - 4294967296 : lo32
    var s2 = hi32 >= 2147483648 ? hi32 - 4294967296 : hi32
    var box = {__ktrnd: true, x: s1, y: s2, z: 0, w: 0, v: s1 ^ (0 - 1), a: ((s1 << 10) | 0) ^ kxwUshr(s2, 4)}
    var i = 0
    while (i < 64) {
        kxwNext(box)
        i = i + 1
    }
    return box
}

// x ushr k for an Int x and k in 1..31.
function kxwUshr(x, k) {
    if (x < 0) { return Math.floor((x + 4294967296) / Math.pow(2, k)) }
    return Math.floor(x / Math.pow(2, k))
}

function kxwNext(box) {
    var t = box["x"]
    t = t ^ kxwUshr(t, 2)
    box["x"] = box["y"]
    box["y"] = box["z"]
    box["z"] = box["w"]
    var v0 = box["v"]
    box["w"] = v0
    t = t ^ ((t << 1) | 0) ^ v0 ^ ((v0 << 4) | 0)
    box["v"] = t
    box["a"] = (box["a"] + 362437) | 0
    return (t + box["a"]) | 0
}

// nextBits(b): the top b bits of one draw (b in 0..31).
function kxwBits(box, b) {
    var r = kxwNext(box)
    if (b == 0) { return 0 }
    return kxwUshr(r, 32 - b)
}

// The exponent of n when n is a power of two in 1..2^31, else -1.
function kxwLog2(n) {
    var u = n
    var b = 0
    while (u > 1) {
        u = u / 2
        b = b + 1
    }
    if (u != 1) { return 0 - 1 }
    return b
}

// nextInt(from, until), until > from: a power-of-two width takes the top bits,
// any other rejects the draws whose remainder would be biased (`bits - v + (n - 1)`
// overflows an Int); a width past 2^31 draws until one lands in range.
function kxwInt(box, from, until) {
    var n = until - from
    if (n > 2147483648) {
        var r = kxwNext(box)
        while (r < from || r >= until) { r = kxwNext(box) }
        return r
    }
    if (kxwLog2(n) >= 0) { return from + kxwBits(box, kxwLog2(n)) }
    var bits = kxwUshr(kxwNext(box), 1)
    var rnd = bits % n
    while (bits - rnd + (n - 1) > 2147483647) {
        bits = kxwUshr(kxwNext(box), 1)
        rnd = bits % n
    }
    return from + rnd
}

// nextLong() as its two 32-bit words [hi, lo] (each 0..2^32-1): the first draw
// shifted up 32, plus the second sign-extended.
function kxwLong(box) {
    var a = kxwNext(box)
    var b = kxwNext(box)
    var h = b < 0 ? a - 1 : a
    return [h < 0 ? h + 4294967296 : h, b < 0 ? b + 4294967296 : b]
}

// nextLong(0, n) for n below 2^31: a power of two takes the top bits of one Int
// draw, any other the remainder of nextLong() ushr 1 by n, rejecting the draw when
// `bits - v + (n - 1)` overflows a Long. The 63-bit bits [bh, bl] are reduced 16
// bits at a time so every product stays exact in a double.
function kxwLongBelow(box, n) {
    if (kxwLog2(n) >= 0) { return kxwBits(box, kxwLog2(n)) }
    var v = 0 - 1
    while (v < 0) {
        var p = kxwLong(box)
        var bh = Math.floor(p[0] / 2)
        var bl = Math.floor(p[1] / 2) + (p[0] % 2) * 2147483648
        v = bh % n
        v = (v * 65536 + Math.floor(bl / 65536)) % n
        v = (v * 65536 + bl % 65536) % n
        var dl = bl - v
        var dh = bh
        if (dl < 0) {
            dl = dl + 4294967296
            dh = dh - 1
        }
        if (dh + Math.floor((dl + n - 1) / 4294967296) >= 2147483648) { v = 0 - 1 }
    }
    return v
}

// A Long's two's complement words [hi, lo] from a number of magnitude below 2^53.
function kxwWords(x) {
    var y = x < 0 ? 0 - x : x
    var hi = Math.floor(y / 4294967296)
    var lo = y % 4294967296
    if (x >= 0) { return [hi, lo] }
    if (lo == 0) { return [(4294967296 - hi) % 4294967296, 0] }
    return [4294967295 - hi, 4294967296 - lo]
}

// MutableList.shuffle(random), in place: from the last index down to 1, each
// element trades places with the one at nextInt(i + 1).
function kxwShuffle(box, es) {
    var i = es.length - 1
    while (i > 0) {
        var j = kxwInt(box, 0, i + 1)
        var t = es[i]
        es[i] = es[j]
        es[j] = t
        i = i - 1
    }
}

// One member call, answered as {t, v} as jrndCall answers it, with t "unsupported"
// for a nextLong range of 2^31 or more.
function kxwCall(box, name, args) {
    if (name == "nextInt" && args.length == 0) { return {t: "int", v: kxwNext(box)} }
    if (name == "nextLong" && args.length == 0) { return {t: "long", v: kxwLong(box)} }
    if (name == "nextInt" || name == "nextLong") {
        var from = 0
        var until = args[0]
        if (args.length > 1) {
            from = args[0]
            until = args[1]
        }
        if (until <= from) { return {t: "err", v: "Random range is empty: [" + from + ", " + until + ")."} }
        if (name == "nextInt") { return {t: "int", v: kxwInt(box, from, until)} }
        if (until - from >= 2147483648) { return {t: "unsupported", v: name} }
        return {t: "long", v: kxwWords(from + kxwLongBelow(box, until - from))}
    }
    if (name == "nextDouble") {
        var d = (kxwBits(box, 26) * 134217728 + kxwBits(box, 27)) / 9007199254740992
        if (args.length == 1) { d = d * args[0] }
        if (args.length > 1) { d = args[0] + d * (args[1] - args[0]) }
        return {t: "double", v: d}
    }
    if (name == "nextFloat" && args.length == 0) { return {t: "float", v: kxwBits(box, 24) / 16777216} }
    if (name == "nextBoolean" && args.length == 0) { return {t: "bool", v: kxwBits(box, 1) != 0} }
    return null
}

// The JVM classes of the objects a collection of JVM class jc hands out, "" where
// none is known. Go twin: ktSubListClass, ktReversedClass, ktIterClass and
// ktValuesClass in abnf/jsrtkotlin.go.
var ktJuAbs = "java.util.AbstractList$"

// l.subList(..): ArrayList's own SubList, java's AbstractList one (RandomAccess
// for an array-backed list), a ListBuilder's BuilderSubList, kotlin's AbstractList
// one for a ReversedListReadOnly and a primitive array's asList; an EmptyList
// answers itself.
function ktSubListClass(jc) {
    if (jc == "java.util.ArrayList" || jc == "java.util.ArrayList$SubList") { return "java.util.ArrayList$SubList" }
    if (jc == "kotlin.collections.builders.ListBuilder" || jc == "kotlin.collections.builders.ListBuilder$BuilderSubList") {
        return "kotlin.collections.builders.ListBuilder$BuilderSubList"
    }
    if (jc == "kotlin.collections.EmptyList") { return jc }
    if (jc == "kotlin.collections.ReversedListReadOnly" || jc == "kotlin.collections.AbstractList$SubList" ||
        jc.indexOf(ktaAsListPfx) == 0) { return "kotlin.collections.AbstractList$SubList" }
    if (jc == "java.util.Arrays$ArrayList" || jc == "java.util.Collections$SingletonList" || jc == ktJuAbs + "RandomAccessSubList") {
        return ktJuAbs + "RandomAccessSubList"
    }
    if (jc == "kotlin.collections.ArrayDeque" || jc == "java.util.LinkedList" || jc == "kotlin.collections.ReversedList" || jc == ktJuAbs + "SubList") {
        return ktJuAbs + "SubList"
    }
    return ""
}

// A primitive array's asList (kind k, 5 and up) is an anonymous AbstractList of
// ArraysKt___ArraysJvmKt, numbered by element type (Byte 1 .. Char 8, ktaElems
// order here); "" for an unsigned array and for an Array, whose Arrays$ArrayList
// is its kind (-4). Go twin: ktaAsListClass in abnf/jsrtkotlin.go.
var ktaAsListNums = [3, 4, 6, 5, 8, 7, 1, 2]
var ktaAsListPfx = "kotlin.collections.ArraysKt___ArraysJvmKt$asList$"
function ktaAsListClass(k) {
    if (k < 5 || k - 5 >= ktaAsListNums.length) { return "" }
    return ktaAsListPfx + ktaAsListNums[k - 5]
}

// Whether jc is one of kotlin.collections' read-only classes, which no Mutable*
// type test or cast accepts: the Empty* singletons, a primitive array's asList, a
// ReversedListReadOnly and kotlin's AbstractList$SubList. Go twin: ktRoJvmClass in
// abnf/jsrtkotlin.go.
function ktRoJvmClass(jc) {
    return jc.indexOf("kotlin.collections.Empty") == 0 || jc.indexOf(ktaAsListPfx) == 0 ||
        jc == "kotlin.collections.ReversedListReadOnly" || jc == "kotlin.collections.AbstractList$SubList"
}

// The JVM marker interface a type name denotes - java.util.RandomAccess (also
// kotlin.collections'), java.lang.Cloneable (kotlin.Cloneable) or
// java.io.Serializable - by its simple name, "" for any other name. Go twin:
// ktMarkerName in abnf/jsrtkotlin.go.
function ktMarkerName(t) {
    if (t == "RandomAccess" || t == "java.util.RandomAccess" || t == "kotlin.collections.RandomAccess") { return "RandomAccess" }
    if (t == "Cloneable" || t == "kotlin.Cloneable" || t == "java.lang.Cloneable") { return "Cloneable" }
    if (t == "Serializable" || t == "java.io.Serializable") { return "Serializable" }
    return ""
}
// Whether JVM class jc implements marker m (ktMarkerName), for the classes this
// runtime hands out. The random-access lists are the array-backed ones and their
// subLists. Every array and every java.util collection that is neither a view nor
// an immutable singleton is Cloneable and Serializable; kotlin's singletons and
// builders, java's singletons, a PriorityQueue, the boxed primitives, String,
// StringBuilder, Pair, Triple and Regex are Serializable only; a view (a subList,
// a map's keys or values, an asReversed, a primitive array's asList) is neither.
// Go twin: ktJvmMarker in abnf/jsrtkotlin.go.
function ktJvmMarker(jc, m) {
    var u = "java.util."
    var kc = "kotlin.collections."
    if (m == "RandomAccess") {
        return jc == u + "ArrayList" || jc == u + "Arrays$ArrayList" || jc == u + "Collections$SingletonList" ||
            jc == u + "ArrayList$SubList" || jc == u + "AbstractList$RandomAccessSubList" ||
            jc == kc + "EmptyList" || jc == kc + "AbstractList$SubList" ||
            jc.indexOf(kc + "builders.ListBuilder") == 0 || jc.indexOf(ktaAsListPfx) == 0
    }
    if (m == "") { return false }
    var cl = jc.charAt(0) == "[" || jc == u + "ArrayList" || jc == u + "ArrayDeque" || jc == u + "LinkedList" ||
        jc == u + "HashSet" || jc == u + "LinkedHashSet" || jc == u + "TreeSet" ||
        jc == u + "HashMap" || jc == u + "LinkedHashMap" || jc == u + "TreeMap"
    if (cl || m == "Cloneable") { return cl }
    if (jc.indexOf("java.lang.") == 0) {
        var b = jc.substring(10)
        return b == "String" || b == "Integer" || b == "Long" || b == "Short" || b == "Byte" || b == "Double" ||
            b == "Float" || b == "Character" || b == "Boolean" || b == "StringBuilder"
    }
    return jc == u + "Arrays$ArrayList" || jc.indexOf(u + "Collections$Singleton") == 0 || jc == u + "PriorityQueue" ||
        jc == kc + "EmptyList" || jc == kc + "EmptySet" || jc == kc + "EmptyMap" ||
        jc == kc + "builders.ListBuilder" || jc == kc + "builders.ListBuilder$BuilderSubList" ||
        jc == kc + "builders.SetBuilder" || jc == kc + "builders.MapBuilder" ||
        jc == "kotlin.Pair" || jc == "kotlin.Triple" || jc == "kotlin.text.Regex"
}

// A live list view - an Array's asList, a List's asReversed, a subList - is a list
// of its own, linked both ways to the list it reads: each side's props __lnk lists
// {a: the other list, rev: whether it reads it reversed, sl: null, or a subList's
// window, shared by both ends}. java's view shares its backing list's storage;
// here a member call, an indexed write or a `+=` on either side copies its
// contents across (ktLinkSync), through every list linked on (views form a tree,
// so `from` is the only edge to skip), a size change counting as a structural
// change of the copy. Go twin: ktLink / ktSubLink / ktLinkSync in
// abnf/jsrtkotlin.go.
function ktLink(owner, view, rev) {
    ktLinkAdd(owner, view, rev, null)
    ktLinkAdd(view, owner, rev, null)
    return view
}
// A subList's window is {up: the backing list, off, n, mod: up's arrMod when the
// two last agreed}. A write through the subList splices it back over the window;
// a write to the backing list refreshes the window, unless the backing list has
// changed structurally since: the subList is then stale for good (java's
// checkForComodification, ktLinkStale) and the backing list drops its link.
function ktSubLink(owner, view, off) {
    var sl = anytype
    sl = {up: owner, off: off, n: view.length, mod: arrMod(owner)}
    ktLinkAdd(owner, view, false, sl)
    ktLinkAdd(view, owner, false, sl)
    return view
}
function ktLinkAdd(a, b, rev, sl) {
    var p = anytype
    p = arrProps(a)
    if (p === undefined) {
        p = {}
        arrPropsSet(a, p)
    }
    if (p.__lnk === undefined) { p.__lnk = [] }
    p.__lnk.push({a: b, rev: rev, sl: sl})
}
// The props of a linked list, else null.
function ktLinkOf(a) {
    var p = anytype
    p = arrProps(a)
    if (p === undefined) { return null }
    if (p.__lnk === undefined) { return null }
    return p
}
// Whether a subList, or a subList it is a window of, outlived a structural
// change of its backing list.
function ktLinkStale(a) {
    var p = anytype
    p = ktLinkOf(a)
    if (p === null) { return false }
    var ls = anytype
    ls = p.__lnk
    for (var i = 0; i < ls.length; i++) {
        var sl = anytype
        sl = ls[i].sl
        if (sl !== null && sl.up === ls[i].a) {
            if (arrMod(sl.up) != sl.mod) { return true }
            return ktLinkStale(sl.up)
        }
    }
    return false
}
function ktLinkSync(a, from) {
    var p = anytype
    p = arrProps(a)
    var ls = anytype
    ls = p.__lnk
    var keep = anytype
    keep = []
    for (var i = 0; i < ls.length; i++) {
        var b = anytype
        b = ls[i].a
        var sl = anytype
        sl = ls[i].sl
        var live = true
        if (b !== from) {
            if (sl === null) {
                ktLinkCopy(b, a, ls[i].rev)
            } else if (sl.up === b) {
                ktLinkSplice(b, a, sl)
            } else if (arrMod(a) == sl.mod) {
                for (var j = 0; j < b.length; j++) { b[j] = a[sl.off + j] }
            } else {
                live = false
            }
            if (live) { ktLinkSync(b, a) }
        }
        if (live) { keep.push(ls[i]) }
    }
    if (keep.length != ls.length) { p.__lnk = keep }
}
// A subList's contents over its window of up; a size change moves up's tail.
function ktLinkSplice(up, sub, sl) {
    var n = sub.length
    var i = 0
    if (n == sl.n) {
        for (i = 0; i < n; i++) { up[sl.off + i] = sub[i] }
    } else {
        var tail = anytype
        tail = []
        for (i = sl.off + sl.n; i < up.length; i++) { tail.push(up[i]) }
        while (up.length > sl.off) { up.pop() }
        for (i = 0; i < n; i++) { up.push(sub[i]) }
        for (i = 0; i < tail.length; i++) { up.push(tail[i]) }
        arrModBump(up)
        sl.n = n
    }
    sl.mod = arrMod(up)
}
function ktLinkCopy(dst, src, rev) {
    var n = src.length
    var i = 0
    if (dst.length != n) {
        while (dst.length > 0) { dst.pop() }
        for (i = 0; i < n; i++) { dst.push(src[i]) }
        arrModBump(dst)
    } else {
        for (i = 0; i < n; i++) { dst[i] = src[i] }
    }
    var hi = n - 1
    i = 0
    while (rev && i < hi) {
        var t = anytype
        t = dst[i]
        dst[i] = dst[hi]
        dst[hi] = t
        i++
        hi--
    }
}

// l.asReversed(): a ReversedList over a MutableList, else a ReversedListReadOnly.
// kotlin picks the overload by the receiver's static type, which a value does not
// carry, so the mutable classes stand for a MutableList receiver.
function ktReversedClass(jc) {
    if (jc == "java.util.ArrayList" || jc == "java.util.ArrayList$SubList" || jc == "kotlin.collections.ArrayDeque" ||
        jc == "java.util.LinkedList" || jc == ktJuAbs + "SubList" || jc == "kotlin.collections.ReversedList") {
        return "kotlin.collections.ReversedList"
    }
    return "kotlin.collections.ReversedListReadOnly"
}

// A map's values view: LinkedHashMap's LinkedValues, a MapBuilderValues, a
// SingletonMap's SingletonSet, an EmptyMap's EmptyList, else the map's own Values.
function ktValuesClass(mjc) {
    if (mjc == "java.util.LinkedHashMap") { return mjc + "$LinkedValues" }
    if (mjc == "kotlin.collections.builders.MapBuilder") { return mjc + "Values" }
    if (mjc == "java.util.Collections$SingletonMap") { return "java.util.Collections$SingletonSet" }
    if (mjc == "kotlin.collections.EmptyMap") { return "kotlin.collections.EmptyList" }
    return mjc + "$Values"
}

// iterator(), and a Map's iterator() over its entries. A java.util set iterates its
// backing map's keys (HashSet a HashMap, TreeSet a TreeMap), and a map view's
// iterator is the map's nested Key / Value / Entry iterator.
function ktIterClass(jc) {
    var b = "kotlin.collections.builders."
    var k = "Entry"
    var m = jc
    var d = jc.indexOf("$")
    if (jc.indexOf("kotlin.collections.Empty") == 0) { return "kotlin.collections.EmptyIterator" }
    if (jc.indexOf("java.util.Collections$Singleton") == 0) { return "java.util.Collections$1" }
    if (jc == b + "ListBuilder" || jc == b + "ListBuilder$BuilderSubList") { return jc + "$Itr" }
    if (jc == b + "SetBuilder" || jc == b + "MapBuilderKeys") { return b + "MapBuilder$KeysItr" }
    if (jc == b + "MapBuilderValues") { return b + "MapBuilder$ValuesItr" }
    if (jc == b + "MapBuilder" || jc == b + "MapBuilderEntries") { return b + "MapBuilder$EntriesItr" }
    if (jc == "kotlin.collections.ReversedList" || jc == "kotlin.collections.ReversedListReadOnly") { return jc + "$listIterator$1" }
    if (jc == "kotlin.collections.ArrayDeque") { return ktJuAbs + "Itr" }
    if (jc == "java.util.ArrayList" || jc == "java.util.PriorityQueue") { return jc + "$Itr" }
    if (jc == "java.util.ArrayList$SubList") { return jc + "$1" }
    if (jc == ktJuAbs + "SubList" || jc == ktJuAbs + "RandomAccessSubList") { return ktJuAbs + "SubList$1" }
    if (jc == "java.util.Arrays$ArrayList") { return "java.util.Arrays$ArrayItr" }
    if (jc == "java.util.ArrayDeque") { return jc + "$DeqIterator" }
    if (jc == "java.util.LinkedList") { return jc + "$ListItr" }
    if (jc.indexOf("java.util.") != 0) { return "" }
    if (d < 0 && jc.substring(jc.length - 3) == "Set") {
        m = jc.substring(0, jc.length - 3) + "Map"
        k = "Key"
    }
    if (d >= 0) {
        m = jc.substring(0, d)
        if (jc.indexOf("Key") > d) { k = "Key" }
        if (jc.indexOf("Value") > d) { k = "Value" }
        if (jc.indexOf("Entry") > d) { k = "Entry" }
    }
    if (m == "java.util.LinkedHashMap") { k = "Linked" + k }
    return m + "$" + k + "Iterator"
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

// An Array<T>'s class, [L<T>; with T read off the elements (a static type argument
// is not kept), as the common supertype Kotlin infers for arrayOf's arguments:
// the one type every element's class has among its supertypes and that is a
// subtype of all the others; java.lang.Object when there is none, or only an
// intersection (Int and String share Comparable and Serializable). graphs holds
// one SUPERTYPE GRAPH per non-null element, entries [class, its direct
// supertypes...] with the element's own class first and every supertype reached
// listed; an empty graph is an element whose class does not stand for its static
// type. Go twin: ktaRefClass in abnf/jsrtkotlin.go.
function ktaRefClass(graphs) {
    var obj = "[Ljava.lang.Object;"
    if (graphs.length == 0 || graphs[0].length == 0) { return obj }
    var g0 = graphs[0]
    var common = []
    var i = 0
    while (i < g0.length) {
        var n = g0[i][0]
        var all = true
        var k = 1
        while (k < graphs.length) {
            if (!ktaHas(graphs[k], n)) { all = false }
            k = k + 1
        }
        if (all) { common.push(n) }
        i = i + 1
    }
    i = 0
    while (i < common.length) {
        var ups = []
        ktaUps(g0, common[i], ups)
        var least = true
        var j = 0
        while (j < common.length) {
            if (ups.indexOf(common[j]) < 0) { least = false }
            j = j + 1
        }
        if (least) {
            if (common[i].charAt(0) == "[") { return "[" + common[i] }
            return "[L" + common[i] + ";"
        }
        i = i + 1
    }
    return obj
}
// Whether graph g has an entry for class n.
function ktaHas(g, n) {
    var i = 0
    while (i < g.length) {
        if (g[i][0] == n) { return true }
        i = i + 1
    }
    return false
}
// Collects n and every supertype graph g reaches from it into out.
function ktaUps(g, n, out) {
    if (out.indexOf(n) >= 0) { return }
    out.push(n)
    var i = 0
    while (i < g.length) {
        if (g[i][0] == n) {
            var j = 1
            while (j < g[i].length) {
                ktaUps(g, g[i][j], out)
                j = j + 1
            }
        }
        i = i + 1
    }
}
// The direct supertypes of a library class (java.io.Serializable,
// java.lang.Comparable and the rest are leaves here), null for a class whose
// static type an element's runtime class cannot stand for - listOf(1) is a
// Collections$SingletonList typed List, a sequence some private class.
function ktaLibSups(n) {
    if (typeof n != "string") { return null }
    var nc = ["java.lang.Number", "java.lang.Comparable"]
    var cs = ["java.lang.Comparable", "java.io.Serializable"]
    var ser = ["java.io.Serializable"]
    if (n == "java.lang.Integer" || n == "java.lang.Long" || n == "java.lang.Short" || n == "java.lang.Byte" ||
        n == "java.lang.Double" || n == "java.lang.Float") { return nc }
    if (n == "java.lang.Number" || n == "java.lang.Throwable" || n == "kotlin.Pair" || n == "kotlin.Triple" ||
        n == "kotlin.text.Regex") { return ser }
    if (n == "java.lang.Character" || n == "java.lang.Boolean" || n == "java.lang.Enum") { return cs }
    if (n == "java.lang.String" || n == "java.lang.StringBuilder") {
        return ["java.lang.Comparable", "java.lang.CharSequence", "java.io.Serializable"]
    }
    if (n == "kotlin.UInt" || n == "kotlin.ULong" || n == "kotlin.UShort" || n == "kotlin.UByte") { return ["java.lang.Comparable"] }
    if (n == "kotlin.reflect.KFunction" || (n.length > 29 && n.substring(0, 29) == "kotlin.jvm.functions.Function")) {
        return ["kotlin.Function"]
    }
    var rk = ["Int", "Long", "Char"]
    var r = 0
    while (r < rk.length) {
        if (n == "kotlin.ranges." + rk[r] + "Range") {
            return ["kotlin.ranges." + rk[r] + "Progression", "kotlin.ranges.ClosedRange", "kotlin.ranges.OpenEndRange"]
        }
        if (n == "kotlin.ranges." + rk[r] + "Progression") { return ["java.lang.Iterable"] }
        r = r + 1
    }
    // Every array is Cloneable and Serializable, and an array of references an
    // Object[].
    if (n.length > 1 && n.charAt(0) == "[") {
        if (n != "[Ljava.lang.Object;" && (n.charAt(1) == "L" || n.charAt(1) == "[")) { return ["[Ljava.lang.Object;"] }
        return ["java.lang.Cloneable", "java.io.Serializable"]
    }
    if (n == "kotlin.Unit" || n == "kotlin.Function" || n == "java.io.Serializable" || n == "kotlin.ranges.ClosedRange" ||
        n == "kotlin.ranges.OpenEndRange" || (n.length > 10 && n.substring(0, 10) == "java.lang.")) { return [] }
    return null
}
// Adds library class n and its supertypes to graph g; false when n is not one
// ktaLibSups knows.
function ktaAddLib(g, n) {
    if (ktaHas(g, n)) { return true }
    var s = ktaLibSups(n)
    if (s === null) { return false }
    var e = [n]
    var i = 0
    while (i < s.length) {
        e.push(s[i])
        i = i + 1
    }
    g.push(e)
    i = 0
    while (i < s.length) {
        ktaAddLib(g, s[i])
        i = i + 1
    }
    return true
}

// The kind a member's fresh List of n elements takes when kotlin-stdlib answers it
// through emptyList() / listOf(e) / toList() by its size (_Collections.kt,
// _Arrays.kt, _Sequences.kt): 4 an EmptyList or a SingletonList at n 0 or 1, -4 an
// Arrays$ArrayList, 0 an ArrayList. recv is the receiver's kind, total its size,
// ranged whether slice was given a range. An Array's reversed, and a slice by an
// index list, are emptyList() or an ArrayList; an Array's slice by a range is an
// asList; a takeLastWhile short of the whole receiver is an ArrayList. Go twin:
// ktSizedKind in abnf/jsrtkotlin.go.
function ktSizedKind(recv, name, n, total, ranged) {
    var arr = recv == 1 || recv >= 5
    if (name == "slice") {
        if (n == 0) { return 4 }
        if (arr) { return ranged ? 0 - 4 : 0 }
        return ranged && n == 1 ? 4 : 0
    }
    if ((name == "reversed" && arr) || (name == "takeLastWhile" && n != total)) { return n == 0 ? 4 : 0 }
    if (ktSizedNames.indexOf(" " + name + " ") < 0) { return 0 }
    return n <= 1 ? 4 : 0
}
var ktSizedNames = " toList take drop takeLast dropLast takeLastWhile dropLastWhile reversed distinct "

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

// The code of a keys / entries view (a Set) or a values view (isValues, a
// Collection) of a map of code mro: a MapBuilder's views check the builder as a
// SetBuilder / ListBuilder does, a SingletonMap's are singleton collections.
function ktRoView(mro, isValues) {
    if (mro == "") { return "" }
    return (isValues ? "L" : "S") + mro.charAt(1)
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
