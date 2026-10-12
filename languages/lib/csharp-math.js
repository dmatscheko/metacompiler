// csharp-math.js -- System.Math's and System.MathF's static members, for csharp-interpreter.abnf
// (include) and languages/lib/csharp-rt.metajs (import). abnf/jsrtcsmath.go is
// the Go twin's port; the three must answer alike. It sits on the hooks of
// lib/csharp-strings.js and lib/csharp-linq.js (csxkNumKind, csxkDbl, csxkFlo,
// csxkFlo32, csxkLt, csxkStr, csxkThrow) and adds three:
//   csxkArith(op, a, b)   C#'s + or - on two integral operands, at their
//                         promoted type
//   csxkIntType(v)        an integral value's type, coded as its width plus 1
//                         when unsigned (int 32, uint 33, byte 9); a char is
//                         17, since it converts exactly as ushort does; 0 for
//                         anything else
//   csxkToInt(v, t)       the integral v converted to the type coded t
// and uses csxIsEnumBox and csxkRefSet for MidpointRounding and out arguments.
//
// Overloads are chosen by the arguments' kinds, as overload resolution would:
// Abs, Sign, Max, Min and Clamp keep an integral argument integral; any double
// makes the answer a double, and float arguments alone a float. Every other
// member answers a double. MathF converts every float parameter's argument to
// float and answers a float, computing in float where .NET does (Round's
// scaling). Written in the subset both engines accept: no exponent literals,
// and a negative zero is spelled 0 / (0 - 1).

// The integral overload of Abs / Max / Min / Clamp that overload resolution
// picks (ECMA-334 12.6.4): among the overloads' types every argument converts
// to implicitly (10.2.3), the narrowest, which converts to all the others.
// Max(uint, long) is Max(long, long), Max(sbyte, byte) is Max(short, short),
// Abs(byte) is Abs(short). A wider signed type holds every narrower type, a
// wider unsigned one only the unsigned ones. 0 when none applies (ulong
// against a signed type is ambiguous, a compile-time error). cands lists the
// member's integral overloads, narrowest first.
function csmIntConv(s, t) {
    if (s == t) { return true }
    if (t - t % 2 <= s - s % 2) { return false }
    return t % 2 == 0 || s % 2 == 1
}
function csmIntOverload(types, cands) {
    var c = 0
    while (c < cands.length) {
        var ok = true
        var i = 0
        while (i < types.length) {
            if (!csmIntConv(types[i], cands[c])) { ok = false }
            i = i + 1
        }
        if (ok) { return cands[c] }
        c = c + 1
    }
    return 0
}
function csmAllInts() { return [8, 9, 16, 17, 32, 33, 64, 65] }
function csmAtType(v, t) { return t == 0 ? v : csxkToInt(v, t) }

function csmIsFlo(k) { return k == "d" || k == "f" }

// The answer of a floating overload: float when no argument is a double.
function csmFloOf(hasD, x) { return hasD ? csxkFlo(x) : csxkFlo32(x) }
// The float nearest x, as a double.
function csmF32(x) { return csxkDbl(csxkFlo32(x)) }

// Round(double): to the nearest integer, a midpoint to the even one, the sign of
// a zero kept. Every double of magnitude 2^52 or more is already integral.
function csmRoundEven(d) {
    if (d !== d || d == 0 || Math.abs(d) >= 4503599627370496) { return d }
    var r = Math.floor(d)
    var f = d - r
    if (f > 0.5 || (f == 0.5 && r - 2 * Math.floor(r / 2) != 0)) { r = r + 1 }
    if (r == 0 && d < 0) { return 0 / (0 - 1) }
    return r
}

// Round under a MidpointRounding mode: 0 ToEven, 1 AwayFromZero (a fraction of
// exactly one half moves away from zero), 2 ToZero, 3 ToNegativeInfinity,
// 4 ToPositiveInfinity. The sign of a zero is kept.
function csmRoundMode(d, mode) {
    if (mode == 0) { return csmRoundEven(d) }
    if (d !== d || d == 0 || Math.abs(d) >= 4503599627370496) { return d }
    var r = Math.trunc(d)
    if (mode == 1) {
        var f = d - r
        if (f >= 0.5) { r = r + 1 }
        else if (f <= 0 - 0.5) { r = r - 1 }
    }
    else if (mode == 3) { r = Math.floor(d) }
    else if (mode == 4) { r = Math.ceil(d) }
    if (r == 0 && d < 0) { return 0 / (0 - 1) }
    return r
}

// Round(value, digits, mode): scaled by 10^digits, rounded, scaled back, below
// 1e16 (the documented midpoint surprises, Round(2.135, 2) == 2.13, come from
// the scaling). MathF's scales in float, below 1e8, with at most 6 digits.
function csmRoundDigits(d, digits, mode, f32) {
    var most = f32 ? 6 : 15
    if (digits < 0 || digits > most) {
        csxkThrow("ArgumentOutOfRangeException", "Rounding digits must be between 0 and " + most + ", inclusive. (Parameter 'digits')")
    }
    if (!(Math.abs(d) < (f32 ? 100000000 : 10000000000000000))) { return d }
    var p = 1
    var i = 0
    while (i < digits) {
        p = p * 10
        i = i + 1
    }
    if (f32) { return csmF32(csmRoundMode(csmF32(d * p), mode) / p) }
    return csmRoundMode(d * p, mode) / p
}

// CopySign: x's magnitude with y's sign, a zero's included. A NaN y reads as
// positive: its sign bit is not observable here.
function csmCopySign(x, y) {
    var m = Math.abs(x)
    if (y < 0 || (y == 0 && 1 / y < 0)) { return -m }
    return m
}

// BigMul(int, int) is the exact long product and BigMul(uint, uint) the ulong
// one. BigMul(a, b, out low) on two longs or two ulongs answers the high 64 bits
// of the 128-bit product and writes the low 64 bits.
function csmBigMul(args) {
    var ts = [csxkIntType(args[0]), csxkIntType(args[1])]
    if (args.length == 2) {
        var t = csmIntOverload(ts, [32, 33])
        if (t == 0) { fail("Math.BigMul of two 64-bit integers (an Int128) is not modelled") }
        return csxkArith("*", csxkToInt(args[0], t + 32), csxkToInt(args[1], t + 32))
    }
    var w = csmIntOverload(ts, [64, 65])
    var a = csxkToInt(args[0], 65)
    var b = csxkToInt(args[1], 65)
    var hl = csmMul128(a, b)
    var hi = hl[0]
    // The signed product: each negative operand subtracts the other from the
    // high half (the two's complement reading of a 64-bit operand is its
    // unsigned value minus 2^64).
    if (w == 64) {
        if (csxkLt(args[0], 0)) { hi = csxkArith("-", hi, b) }
        if (csxkLt(args[1], 0)) { hi = csxkArith("-", hi, a) }
    }
    csxkRefSet(args[2], csxkToInt(hl[1], w))
    return csxkToInt(hi, w)
}
// The 128-bit product of two ulongs as [high, low], from the four products of
// their 32-bit halves (a shift is a multiplication or division by 2^32, which
// csxkArith has at every width).
function csmMul128(a, b) {
    var m = csxkToInt(4294967295, 65)
    var k = csxkToInt(4294967296, 65)
    var al = csxkArith("&", a, m)
    var ah = csxkArith("/", a, k)
    var bl = csxkArith("&", b, m)
    var bh = csxkArith("/", b, k)
    var p0 = csxkArith("*", al, bl)
    var p1 = csxkArith("*", al, bh)
    var p2 = csxkArith("*", ah, bl)
    var mid = csxkArith("+", csxkArith("+", csxkArith("/", p0, k), csxkArith("&", p1, m)), csxkArith("&", p2, m))
    var lo = csxkArith("|", csxkArith("*", mid, k), csxkArith("&", p0, m))
    var hi = csxkArith("+", csxkArith("*", ah, bh), csxkArith("/", p1, k))
    hi = csxkArith("+", csxkArith("+", hi, csxkArith("/", p2, k)), csxkArith("/", mid, k))
    return [hi, lo]
}

// DivRem(a, b, out rem) on two ints or two longs answers the quotient and
// writes a - quotient * b; DivRem(a, b) answers the tuple (Quotient,
// Remainder) at the overload's type, for every integral type.
function csmDivRem(args) {
    var ts = [csxkIntType(args[0]), csxkIntType(args[1])]
    var t = csmIntOverload(ts, args.length == 3 ? [32, 64] : csmAllInts())
    var a = csxkToInt(args[0], t)
    var b = csxkToInt(args[1], t)
    var q = csxkToInt(csxkArith("/", a, b), t)
    var r = csxkToInt(csxkArith("-", a, csxkArith("*", q, b)), t)
    if (args.length == 3) {
        csxkRefSet(args[2], r)
        return q
    }
    var tup = {}
    tup["__tuple"] = 2
    tup["Item1"] = q
    tup["Item2"] = r
    tup["Quotient"] = q
    tup["Remainder"] = r
    return tup
}

function csmAbs(x) {
    var k = csxkNumKind(x)
    if (csmIsFlo(k)) { return csmFloOf(k == "d", Math.abs(csxkDbl(x))) }
    var tx = csxkIntType(x)
    var t = tx == 32 || tx == 64 ? tx : csmIntOverload([tx], [8, 16, 32, 64])
    if (!csxkLt(x, 0)) { return csmAtType(x, t) }
    // Negating MinValue wraps back to it at the overload's own width.
    var r = csmAtType(csxkArith("-", 0, x), t)
    if (csxkLt(r, 0)) { csxkThrow("OverflowException", "Negating the minimum value of a twos complement number is invalid.") }
    return r
}

function csmSign(x) {
    var d = csxkDbl(x)
    if (d !== d) { csxkThrow("ArithmeticException", "Function does not accept floating point Not-a-Number values.") }
    if (d > 0) { return 1 }
    if (d < 0) { return 0 - 1 }
    return 0
}

// Max / Min. On doubles NaN wins and +0 is greater than -0 (IEEE 754-2019
// maximum / minimum). Integral arguments answer at the overload's type.
function csmMinMax(a, b, max) {
    var ka = csxkNumKind(a)
    var kb = csxkNumKind(b)
    if (csmIsFlo(ka) || csmIsFlo(kb)) {
        var x = csxkDbl(a)
        var y = csxkDbl(b)
        var r = x
        if (y !== y) { r = y }
        else if (x == y) {
            if (x == 0 && (1 / x > 0) != max) { r = y }
        }
        else if (x === x && (y > x) == max) { r = y }
        return csmFloOf(ka == "d" || kb == "d", r)
    }
    var pick = csxkLt(a, b) == max ? b : a
    var ta = csxkIntType(a)
    var tb = csxkIntType(b)
    if (ta == tb && ta != 17) { return pick }
    return csmAtType(pick, csmIntOverload([ta, tb], csmAllInts()))
}

function csmClamp(v, lo, hi) {
    if (csxkLt(hi, lo)) {
        csxkThrow("ArgumentException", "'" + csxkStr(lo) + "' cannot be greater than " + csxkStr(hi) + ".")
    }
    var r = v
    if (csxkLt(v, lo)) { r = lo }
    else if (csxkLt(hi, v)) { r = hi }
    var kv = csxkNumKind(v)
    var kl = csxkNumKind(lo)
    var kh = csxkNumKind(hi)
    if (csmIsFlo(kv) || csmIsFlo(kl) || csmIsFlo(kh)) {
        return csmFloOf(kv == "d" || kl == "d" || kh == "d", csxkDbl(r))
    }
    var tv = csxkIntType(v)
    if (tv == csxkIntType(lo) && tv == csxkIntType(hi) && tv != 17) { return r }
    return csmAtType(r, csmIntOverload([tv, csxkIntType(lo), csxkIntType(hi)], csmAllInts()))
}

// Pow is IEEE 754's pow: 1 to any power, and -1 to an infinite one, is 1.
function csmPow(x, y) {
    if (x == 1) { return 1 }
    if (x == 0 - 1 && (y == Math.pow(2, 1024) || y == 0 - Math.pow(2, 1024))) { return 1 }
    return Math.pow(x, y)
}

// Log(a, newBase), with the documented NaN cases ahead of the quotient.
function csmLogBase(a, b) {
    if (a !== a) { return a }
    if (b !== b) { return b }
    if (b == 1) { return 0 / 0 }
    if (a != 1 && (b == 0 || b == Math.pow(2, 1024))) { return 0 / 0 }
    return Math.log(a) / Math.log(b)
}

// One of the members of one double argument; undefined for any other name.
function csmUnary(name, d) {
    if (name == "Sqrt") { return Math.sqrt(d) }
    if (name == "Floor") { return Math.floor(d) }
    if (name == "Ceiling") { return Math.ceil(d) }
    if (name == "Truncate") { return Math.trunc(d) }
    if (name == "Round") { return csmRoundEven(d) }
    if (name == "Exp") { return Math.exp(d) }
    if (name == "Log") { return Math.log(d) }
    if (name == "Log10") { return Math.log10(d) }
    if (name == "Log2") { return Math.log2(d) }
    if (name == "Sin") { return Math.sin(d) }
    if (name == "Cos") { return Math.cos(d) }
    if (name == "Tan") { return Math.tan(d) }
    if (name == "Asin") { return Math.asin(d) }
    if (name == "Acos") { return Math.acos(d) }
    if (name == "Atan") { return Math.atan(d) }
    if (name == "Sinh") { return Math.sinh(d) }
    if (name == "Cosh") { return Math.cosh(d) }
    if (name == "Tanh") { return Math.tanh(d) }
    if (name == "Cbrt") { return Math.cbrt(d) }
    return undefined
}

// System.Double's constants: NaN, the two infinities, Epsilon (the least
// positive subnormal, 2^-1074) and MaxValue / MinValue (+-(2^53 - 1) * 2^971).
function csmDoubleConstNames() { return ["NaN", "PositiveInfinity", "NegativeInfinity", "Epsilon", "MaxValue", "MinValue"] }
// System.Single's (f32) are the binary32 extremes: Epsilon 2^-149, MaxValue
// (2^24-1)*2^104.
function csmDoubleConst(name, f32) {
    if (name == "NaN") { return 0 / 0 }
    if (name == "PositiveInfinity") { return Math.pow(2, 1024) }
    if (name == "NegativeInfinity") { return 0 - Math.pow(2, 1024) }
    if (name == "Epsilon") { return f32 ? Math.pow(2, 0 - 149) : Math.pow(2, 0 - 1074) }
    var m = f32 ? (Math.pow(2, 24) - 1) * Math.pow(2, 104) : (Math.pow(2, 53) - 1) * Math.pow(2, 971)
    return name == "MaxValue" ? m : 0 - m
}
// System.Double's classification members, each of one double.
function csmDoubleIsNames() { return ["IsNaN", "IsInfinity", "IsPositiveInfinity", "IsNegativeInfinity", "IsFinite"] }
function csmDoubleIs(name, d) {
    var inf = Math.pow(2, 1024)
    if (name == "IsNaN") { return d !== d }
    if (name == "IsInfinity") { return d == inf || d == 0 - inf }
    if (name == "IsPositiveInfinity") { return d == inf }
    if (name == "IsNegativeInfinity") { return d == 0 - inf }
    return d === d && d != inf && d != 0 - inf
}

// MathF's arguments: each one of a float parameter converted to float (Round's
// digits and mode are not).
function csmF32Args(name, args) {
    var out = []
    var i = 0
    while (i < args.length) {
        var a = args[i]
        if (name == "Round" && i > 0) { out.push(a) } else { out.push(csxkFlo32(csxkDbl(a))) }
        i = i + 1
    }
    return out
}

// Math.name(args), or MathF's when f32.
function csmStatic(name, args0, f32) {
    var args = f32 ? csmF32Args(name, args0) : args0
    var n = args.length
    if (name == "Round" && n > 1 && csxIsEnumBox(args[n - 1])) {
        var digits = n == 3 ? csxkNum(args[1]) : 0
        return csmFloOf(!f32, csmRoundDigits(csxkDbl(args[0]), digits, args[n - 1]["v"], f32))
    }
    if (name == "BigMul" && !f32) { return csmBigMul(args) }
    if (name == "DivRem" && !f32) { return csmDivRem(args) }
    if (n == 1) {
        if (name == "Abs") { return csmAbs(args[0]) }
        if (name == "Sign") { return csmSign(args[0]) }
        var u = csmUnary(name, csxkDbl(args[0]))
        if (u !== undefined) { return csmFloOf(!f32, u) }
    }
    if (n == 2) {
        if (name == "Max") { return csmMinMax(args[0], args[1], true) }
        if (name == "Min") { return csmMinMax(args[0], args[1], false) }
        var a = csxkDbl(args[0])
        var b = csxkDbl(args[1])
        if (name == "Pow") { return csmFloOf(!f32, csmPow(a, b)) }
        if (name == "Atan2") { return csmFloOf(!f32, Math.atan2(a, b)) }
        if (name == "Log") { return csmFloOf(!f32, csmLogBase(a, b)) }
        if (name == "Round") { return csmFloOf(!f32, csmRoundDigits(a, b, 0, f32)) }
        if (name == "CopySign") { return csmFloOf(!f32, csmCopySign(a, b)) }
    }
    if (n == 3 && name == "Clamp") { return csmClamp(args[0], args[1], args[2]) }
    fail((f32 ? "MathF." : "Math.") + name + " with " + n + " arguments is not modelled")
    return undefined
}
