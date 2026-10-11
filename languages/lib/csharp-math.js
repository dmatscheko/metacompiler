// csharp-math.js -- System.Math's static members, for csharp-interpreter.abnf
// (include) and languages/lib/csharp-rt.metajs (import). abnf/jsrtcsmath.go is
// the Go twin's port; the three must answer alike. It sits on the hooks of
// lib/csharp-strings.js and lib/csharp-linq.js (csxkNumKind, csxkDbl, csxkFlo,
// csxkFlo32, csxkLt, csxkStr, csxkThrow) and adds one:
//   csxkArith(op, a, b)   C#'s + or - on two integral operands, at their
//                         promoted type
//
// Overloads are chosen by the arguments' kinds, as overload resolution would:
// Abs, Sign, Max, Min and Clamp keep an integral argument integral; any double
// makes the answer a double, and float arguments alone a float. Every other
// member answers a double. Written in the subset both engines accept: no
// exponent literals, and a negative zero is spelled 0 / (0 - 1).

function csmIsFlo(k) { return k == "d" || k == "f" }

// The answer of a floating overload: float when no argument is a double.
function csmFloOf(hasD, x) { return hasD ? csxkFlo(x) : csxkFlo32(x) }

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

// Round(double, digits): scaled by 10^digits, rounded, scaled back, below 1e16
// (the documented midpoint surprises, Round(2.135, 2) == 2.13, come from the
// scaling).
function csmRoundDigits(d, digits) {
    if (digits < 0 || digits > 15) {
        csxkThrow("ArgumentOutOfRangeException", "Rounding digits must be between 0 and 15, inclusive. (Parameter 'digits')")
    }
    if (!(Math.abs(d) < 10000000000000000)) { return d }
    var p = 1
    var i = 0
    while (i < digits) {
        p = p * 10
        i = i + 1
    }
    return csmRoundEven(d * p) / p
}

function csmAbs(x) {
    var k = csxkNumKind(x)
    if (csmIsFlo(k)) { return csmFloOf(k == "d", Math.abs(csxkDbl(x))) }
    if (!csxkLt(x, 0)) { return x }
    var r = csxkArith("-", 0, x)
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
// maximum / minimum). Two integral arguments of different types answer at their
// promoted type: the pick plus a zero of the other's type.
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
    var lt = csxkLt(a, b)
    var pick = lt == max ? b : a
    if (ka == kb) { return pick }
    var other = lt == max ? a : b
    return csxkArith("+", pick, csxkArith("-", other, other))
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
    return r
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

function csmStatic(name, args) {
    var n = args.length
    if (n == 1) {
        if (name == "Abs") { return csmAbs(args[0]) }
        if (name == "Sign") { return csmSign(args[0]) }
        var u = csmUnary(name, csxkDbl(args[0]))
        if (u !== undefined) { return csxkFlo(u) }
    }
    if (n == 2) {
        if (name == "Max") { return csmMinMax(args[0], args[1], true) }
        if (name == "Min") { return csmMinMax(args[0], args[1], false) }
        var a = csxkDbl(args[0])
        var b = csxkDbl(args[1])
        if (name == "Pow") { return csxkFlo(csmPow(a, b)) }
        if (name == "Atan2") { return csxkFlo(Math.atan2(a, b)) }
        if (name == "Log") { return csxkFlo(csmLogBase(a, b)) }
        if (name == "Round") { return csxkFlo(csmRoundDigits(a, b)) }
    }
    if (n == 3 && name == "Clamp") { return csmClamp(args[0], args[1], args[2]) }
    fail("Math." + name + " with " + n + " arguments is not modelled")
    return undefined
}
