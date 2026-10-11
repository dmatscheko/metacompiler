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

// The value of a constant, PI or E.
function ktmConst(name) {
    if (name == "PI") { return 3.141592653589793 }
    return 2.718281828459045
}
