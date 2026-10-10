// fdlibm.js -- java.lang.StrictMath: fdlibm 5.3 as Java's own FdLibm.java ports
// it, for java-interpreter.abnf (include) and languages/lib/java-rt.metajs
// (import). abnf/jsrtfdlibm.go is the Go twin's copy, written against the C
// with real bit access; this one follows it line for line.
//
// fdlibm reads and writes a double's two 32 bit halves. MetaJS has no access to
// a double's bits, so fdHi / fdLo / fdW compute them with exact arithmetic
// only: scaling by a power of two is exact away from overflow, and the
// exponent is the host's log2 corrected against exact powers of two. A sign
// or a zero is never produced by arithmetic that could lose a -0.0 (`x * 0`
// and `0 - x` do in this dialect): fdNeg and fdNegZero spell those.
//
// Written in the restricted subset both engines accept (see regex.js): no
// exponent literals, so every constant is its fdlibm bit pattern through fdW.
// The constants are built by the first call (fdInit): layer 2 keeps whatever
// its top level builds alive for the whole run, and every collection scans it.

var fdZero = 0
var fdInf = 1 / fdZero
var fdNaN = fdZero / fdZero
var fdNegZero = 0 / (0 - 1)

// 2^n, exactly, for n in -1074..1023: every product of powers of two is exact
// down to the smallest subnormal, and a square past the last one used may
// overflow or vanish harmlessly.
function fdPow2(n) {
    var x = 1
    var base = n < 0 ? 0.5 : 2
    var k = n < 0 ? 0 - n : n
    while (k > 0) {
        if (k % 2 == 1) { x = x * base }
        base = base * base
        k = Math.floor(k / 2)
    }
    return x
}
// floor(log2(a)) for a finite a > 0, leaving 2^that in fdEP: the host's log is
// off by far less than one there, and the powers of two settle it exactly.
var fdEP = 0
function fdExpOf(a) {
    var e = Math.floor(Math.log(a) * 1.4426950408889634)
    if (e < 0 - 1074) { e = 0 - 1074 }
    if (e > 1023) { e = 1023 }
    var p = fdPow2(e)
    while (e > 0 - 1074 && p > a) {
        e = e - 1
        p = p / 2
    }
    while (e < 1023 && p * 2 <= a) {
        e = e + 1
        p = p * 2
    }
    fdEP = p
    return e
}
// The high word of x as a signed 32 bit integer (fdlibm's __HI).
function fdHi(x) {
    if (x !== x) { return 0x7ff80000 }
    var neg = x < 0 || (x == 0 && 1 / x < 0)
    var a = x < 0 ? -x : x
    var h = 0
    if (a == fdInf) {
        h = 0x7ff00000
    } else if (a > 0) {
        var e = fdExpOf(a)
        if (e < 0 - 1022) { h = Math.floor(a / fdPow2(0 - 1042)) }
        else { h = (e + 1023) * 1048576 + Math.floor((a / fdEP - 1) * 1048576) }
    }
    if (neg) { return h - 2147483648 }
    return h
}
// The low word of x as an unsigned 32 bit integer (fdlibm's __LO).
function fdLo(x) {
    if (x !== x) { return 0 }
    var a = x < 0 ? -x : x
    if (a == fdInf || a == 0) { return 0 }
    var m = 0
    if (fdExpOf(a) < 0 - 1022) { m = a / fdPow2(0 - 1074) }
    else { m = (a / fdEP - 1) * 4503599627370496 }
    return m - Math.floor(m / 4294967296) * 4294967296
}
// The double whose words are hi (signed or unsigned) and lo (unsigned).
function fdW(hi, lo) {
    var h = hi < 0 ? hi + 4294967296 : hi
    var neg = h >= 2147483648
    if (neg) { h = h - 2147483648 }
    var be = Math.floor(h / 1048576)
    var f = (h - be * 1048576) * 4294967296 + lo
    var v = 0
    if (be == 2047) {
        if (f != 0) { return fdNaN }
        v = fdInf
    } else if (be == 0) {
        v = f * fdPow2(0 - 1074)
    } else {
        v = (1 + f * fdPow2(0 - 52)) * fdPow2(be - 1023)
    }
    if (neg) { return fdNeg(v) }
    return v
}
function fdSetHi(x, hi) { return fdW(hi, fdLo(x)) }
// x with its low word cleared: |x| cut to its 21 leading significand bits.
function fdChop(x) {
    if (x !== x || x == 0 || x == fdInf || x == 0 - fdInf) { return x }
    var a = Math.abs(x)
    var g = fdExpOf(a) < 0 - 1022 ? fdPow2(0 - 1042) : fdEP / 1048576
    var c = Math.floor(a / g) * g
    if (x < 0) { return fdNeg(c) }
    return c
}
// -v, a zero included.
function fdNeg(v) {
    if (v == 0) {
        if (1 / v > 0) { return fdNegZero }
        return 0
    }
    return -v
}
// C's (int) of a double: toward zero.
function fdTrunc(v) {
    if (v < 0) { return 0 - Math.floor(-v) }
    return Math.floor(v)
}
// x * 2^n rounded once (scalbn).
function fdScalb(x, n) {
    if (n > 1023) { return x * fdPow2(1023) * fdPow2(n - 1023) }
    if (n >= 0 - 1022) { return x * fdPow2(n) }
    return x * fdPow2(n + 1022) * fdPow2(0 - 1022)
}
// c[0] + z*(c[1] + z*(... + z*c[n-1])).
function fdPoly(z, c) {
    var i = c.length - 1
    var q = c[i]
    i = i - 1
    while (i >= 0) {
        q = c[i] + z * q
        i = i - 1
    }
    return q
}

var fdLn2Hi = 0
var fdLn2Lo = 0
var fdTwo54 = 0
var fdHuge = 0
var fdTiny = 0
var fdInvLn2 = 0

// e_exp.c.
var fdExpP = []
var fdExpOHi = 0
var fdExpULo = 0
function fdExp(xIn) {
    if (!fdReady) { fdInit() }
    var x = xIn
    if (x !== x) { return x + x }
    var neg = x < 0
    var ax = neg ? -x : x
    if (ax >= 709.7822265625) {
        if (ax == fdInf) {
            if (neg) { return 0 }
            return x
        }
        if (x > fdExpOHi) { return fdInf }
        if (x < fdExpULo) { return 0 }
    }
    var hi = 0
    var lo = 0
    var k = 0
    if (ax >= 0.3465735912322998) {
        if (ax < 1.0397205352783203) {
            if (neg) {
                hi = x + fdLn2Hi
                lo = fdNeg(fdLn2Lo)
                k = 0 - 1
            } else {
                hi = x - fdLn2Hi
                lo = fdLn2Lo
                k = 1
            }
        } else {
            k = fdTrunc(fdInvLn2 * x + (neg ? 0 - 0.5 : 0.5))
            hi = x - k * fdLn2Hi
            lo = k * fdLn2Lo
        }
        x = hi - lo
    } else if (ax < 0.000000003725290298461914) {
        return 1 + x
    }
    var t = x * x
    var c = x - t * fdPoly(t, fdExpP)
    if (k == 0) { return 1 - ((x * c) / (c - 2) - x) }
    var y = 1 - ((lo - (x * c) / (2 - c)) - hi)
    return fdScalb(y, k)
}

// e_log.c.
var fdLgA = []
var fdLgB = []
function fdLog(xIn) {
    if (!fdReady) { fdInit() }
    var x = xIn
    var hx = fdHi(x)
    var lx = fdLo(x)
    var k = 0
    if (hx < 0x00100000) {
        if (((hx & 0x7fffffff) | lx) == 0) { return 0 - fdInf }
        if (hx < 0) { return fdNaN }
        k = k - 54
        x = x * fdTwo54
        hx = fdHi(x)
    }
    if (hx >= 0x7ff00000) { return x + x }
    k = k + (hx >> 20) - 1023
    hx = hx & 0x000fffff
    var i = (hx + 0x95f64) & 0x100000
    x = fdSetHi(x, hx | (i ^ 0x3ff00000))
    k = k + (i >> 20)
    var f = x - 1
    var dk = k
    if ((0x000fffff & (2 + hx)) < 3) {
        if (f == 0) {
            if (k == 0) { return 0 }
            return dk * fdLn2Hi + dk * fdLn2Lo
        }
        var r0 = f * f * (0.5 - 0.33333333333333333 * f)
        if (k == 0) { return f - r0 }
        return dk * fdLn2Hi - ((r0 - dk * fdLn2Lo) - f)
    }
    var s = f / (2 + f)
    var z = s * s
    i = hx - 0x6147a
    var w = z * z
    var j = 0x6b851 - hx
    var t1 = w * fdPoly(w, fdLgA)
    var t2 = z * fdPoly(w, fdLgB)
    i = i | j
    var r = t2 + t1
    if (i > 0) {
        var hfsq = 0.5 * f * f
        if (k == 0) { return f - (hfsq - s * (hfsq + r)) }
        return dk * fdLn2Hi - ((hfsq - (s * (hfsq + r) + dk * fdLn2Lo)) - f)
    }
    if (k == 0) { return f - s * (f - r) }
    return dk * fdLn2Hi - ((s * (f - r) - dk * fdLn2Lo) - f)
}

// e_log10.c.
var fdIvLn10 = 0
var fdLog10Hi = 0
var fdLog10Lo = 0
function fdLog10(xIn) {
    if (!fdReady) { fdInit() }
    var x = xIn
    var hx = fdHi(x)
    var lx = fdLo(x)
    var k = 0
    if (hx < 0x00100000) {
        if (((hx & 0x7fffffff) | lx) == 0) { return 0 - fdInf }
        if (hx < 0) { return fdNaN }
        k = k - 54
        x = x * fdTwo54
        hx = fdHi(x)
    }
    if (hx >= 0x7ff00000) { return x + x }
    k = k + (hx >> 20) - 1023
    var i = k < 0 ? 1 : 0
    hx = (hx & 0x000fffff) | ((0x3ff - i) << 20)
    var y = k + i
    x = fdSetHi(x, hx)
    var z = y * fdLog10Lo + fdIvLn10 * fdLog(x)
    return z + y * fdLog10Hi
}

// s_log1p.c.
var fdLp = []
function fdLog1p(x) {
    if (!fdReady) { fdInit() }
    var hx = fdHi(x)
    var ax = hx & 0x7fffffff
    var k = 1
    var f = 0
    var c = 0
    var hu = 0
    if (hx < 0x3FDA827A) {
        if (ax >= 0x3ff00000) {
            if (x == 0 - 1) { return 0 - fdInf }
            return fdNaN
        }
        if (ax < 0x3e200000) {
            if (fdTwo54 + x > 0 && ax < 0x3c900000) { return x }
            return x - x * x * 0.5
        }
        if (hx > 0 || hx <= 0 - 1076707645) {
            k = 0
            f = x
            hu = 1
        }
    }
    if (hx >= 0x7ff00000) { return x + x }
    if (k != 0) {
        var u = 0
        if (hx < 0x43400000) {
            u = 1 + x
            hu = fdHi(u)
            k = (hu >> 20) - 1023
            if (k > 0) { c = 1 - (u - x) } else { c = x - (u - 1) }
            c = c / u
        } else {
            u = x
            hu = fdHi(u)
            k = (hu >> 20) - 1023
            c = 0
        }
        hu = hu & 0x000fffff
        if (hu < 0x6a09e) {
            u = fdSetHi(u, hu | 0x3ff00000)
        } else {
            k = k + 1
            u = fdSetHi(u, hu | 0x3fe00000)
            hu = (0x00100000 - hu) >> 2
        }
        f = u - 1
    }
    var hfsq = 0.5 * f * f
    var dk = k
    if (hu == 0) {
        if (f == 0) {
            if (k == 0) { return 0 }
            c = c + dk * fdLn2Lo
            return dk * fdLn2Hi + c
        }
        var r0 = hfsq * (1 - 0.66666666666666666 * f)
        if (k == 0) { return f - r0 }
        return dk * fdLn2Hi - ((r0 - (dk * fdLn2Lo + c)) - f)
    }
    var s = f / (2 + f)
    var z = s * s
    var r = z * fdPoly(z, fdLp)
    if (k == 0) { return f - (hfsq - s * (hfsq + r)) }
    return dk * fdLn2Hi - ((hfsq - (s * (hfsq + r) + (dk * fdLn2Lo + c))) - f)
}

// s_expm1.c.
var fdExpm1Q = []
function fdExpm1(xIn) {
    if (!fdReady) { fdInit() }
    var x = xIn
    var hx = fdHi(x)
    var neg = hx < 0
    hx = hx & 0x7fffffff
    if (hx >= 0x4043687A) {
        if (hx >= 0x40862E42) {
            if (hx >= 0x7ff00000) {
                if (x !== x) { return x + x }
                if (neg) { return 0 - 1 }
                return x
            }
            if (x > fdExpOHi) { return fdInf }
        }
        if (neg) {
            if (x + fdTiny < 0) { return fdTiny - 1 }
        }
    }
    var hi = 0
    var lo = 0
    var c = 0
    var k = 0
    if (hx > 0x3fd62e42) {
        if (hx < 0x3FF0A2B2) {
            if (!neg) {
                hi = x - fdLn2Hi
                lo = fdLn2Lo
                k = 1
            } else {
                hi = x + fdLn2Hi
                lo = fdNeg(fdLn2Lo)
                k = 0 - 1
            }
        } else {
            k = fdTrunc(fdInvLn2 * x + (neg ? 0 - 0.5 : 0.5))
            hi = x - k * fdLn2Hi
            lo = k * fdLn2Lo
        }
        x = hi - lo
        c = (hi - x) - lo
    } else if (hx < 0x3c900000) {
        if (x == 0) { return x }
        var tt = fdHuge + x
        return x - (tt - (fdHuge + x))
    }
    var hfx = 0.5 * x
    var hxs = x * hfx
    var r1 = fdPoly(hxs, fdExpm1Q)
    var t = 3 - r1 * hfx
    var e = hxs * ((r1 - t) / (6 - x * t))
    if (k == 0) { return x - (x * e - hxs) }
    e = x * (e - c) - c
    e = e - hxs
    if (k == 0 - 1) { return 0.5 * (x - e) - 0.5 }
    if (k == 1) {
        if (x < 0 - 0.25) { return (0 - 2) * (e - (x + 0.5)) }
        return 1 + 2 * (x - e)
    }
    var y = 0
    if (k <= 0 - 2 || k > 56) {
        y = 1 - (e - x)
        y = fdSetHi(y, fdHi(y) + (k << 20))
        return y - 1
    }
    t = 1
    if (k < 20) {
        t = fdSetHi(t, 0x3ff00000 - (0x200000 >> k))
        y = t - (e - x)
        y = fdSetHi(y, fdHi(y) + (k << 20))
    } else {
        t = fdSetHi(t, (0x3ff - k) << 20)
        y = x - (e + t)
        y = y + 1
        y = fdSetHi(y, fdHi(y) + (k << 20))
    }
    return y
}

// k_sin.c, k_cos.c and k_tan.c.
var fdS1 = 0
var fdSinP = []
function fdKSin(x, y, iy) {
    var ix = fdHi(x) & 0x7fffffff
    if (ix < 0x3e400000) {
        if (fdTrunc(x) == 0) { return x }
    }
    var z = x * x
    var v = z * x
    var r = fdPoly(z, fdSinP)
    if (iy == 0) { return x + v * (fdS1 + z * r) }
    return x - ((z * (0.5 * y - v * r) - y) - v * fdS1)
}
var fdCosP = []
function fdKCos(x, y) {
    var ix = fdHi(x) & 0x7fffffff
    if (ix < 0x3e400000) {
        if (fdTrunc(x) == 0) { return 1 }
    }
    var z = x * x
    var r = z * fdPoly(z, fdCosP)
    if (ix < 0x3FD33333) { return 1 - (0.5 * z - (z * r - x * y)) }
    var qx = 0.28125
    if (ix <= 0x3fe90000) { qx = fdW(ix - 0x00200000, 0) }
    var hz = 0.5 * z - qx
    var a = 1 - qx
    return a - (hz - (z * r - x * y))
}
var fdTanT = []
var fdTanOdd = []
var fdTanEven = []
var fdPio4 = 0
var fdPio4Lo = 0
function fdKTan(xIn, yIn, iy) {
    var x = xIn
    var y = yIn
    var hx = fdHi(x)
    var ix = hx & 0x7fffffff
    var z = 0
    var w = 0
    var v = 0
    var s = 0
    var a = 0
    var t = 0
    if (ix < 0x3e300000) {
        if (fdTrunc(x) == 0) {
            if (((ix | fdLo(x)) | (iy + 1)) == 0) { return 1 / Math.abs(x) }
            if (iy == 1) { return x }
            w = x + y
            z = fdChop(w)
            v = y - (z - x)
            a = (0 - 1) / w
            t = fdChop(a)
            s = 1 + t * z
            return t + a * (s + t * v)
        }
    }
    if (ix >= 0x3FE59428) {
        if (hx < 0) {
            x = fdNeg(x)
            y = fdNeg(y)
        }
        z = fdPio4 - x
        w = fdPio4Lo - y
        x = z + w
        y = 0
    }
    z = x * x
    w = z * z
    var r = fdPoly(w, fdTanOdd)
    v = z * fdPoly(w, fdTanEven)
    s = z * x
    r = y + z * (s * (r + v) + y)
    r = r + fdTanT[0] * s
    w = x + r
    if (ix >= 0x3FE59428) {
        v = iy
        return (1 - ((hx >> 30) & 2)) * (v - 2 * (x - (w * w / (w + v) - r)))
    }
    if (iy == 1) { return w }
    z = fdChop(w)
    v = r - (z - x)
    a = (0 - 1) / w
    t = fdChop(a)
    s = 1 + t * z
    return t + a * (s + t * v)
}

// e_rem_pio2.c: n is the result, and x - n*pi/2 is fdRY0 + fdRY1.
var fdRY0 = 0
var fdRY1 = 0
var fdTwoOverPi = []
var fdNpio2Hw = []
var fdInvPio2 = 0
var fdPio2_1 = 0
var fdPio2_1t = 0
var fdPio2_2 = 0
var fdPio2_2t = 0
var fdPio2_3 = 0
var fdPio2_3t = 0
var fdTwo24 = 0
var fdTwon24 = 0
function fdRemPio2(x) {
    var hx = fdHi(x)
    var ix = hx & 0x7fffffff
    var z = 0
    if (ix <= 0x3fe921fb) {
        fdRY0 = x
        fdRY1 = 0
        return 0
    }
    if (ix < 0x4002d97c) {
        if (hx > 0) {
            z = x - fdPio2_1
            if (ix != 0x3ff921fb) {
                fdRY0 = z - fdPio2_1t
                fdRY1 = (z - fdRY0) - fdPio2_1t
            } else {
                z = z - fdPio2_2
                fdRY0 = z - fdPio2_2t
                fdRY1 = (z - fdRY0) - fdPio2_2t
            }
            return 1
        }
        z = x + fdPio2_1
        if (ix != 0x3ff921fb) {
            fdRY0 = z + fdPio2_1t
            fdRY1 = (z - fdRY0) + fdPio2_1t
        } else {
            z = z + fdPio2_2
            fdRY0 = z + fdPio2_2t
            fdRY1 = (z - fdRY0) + fdPio2_2t
        }
        return 0 - 1
    }
    var n = 0
    if (ix <= 0x413921fb) {
        var t = Math.abs(x)
        n = Math.floor(t * fdInvPio2 + 0.5)
        var fn = n
        var r = t - fn * fdPio2_1
        var w = fn * fdPio2_1t
        if (n < 32 && ix != fdNpio2Hw[n - 1]) {
            fdRY0 = r - w
        } else {
            var j = ix >> 20
            fdRY0 = r - w
            var i = j - ((fdHi(fdRY0) >> 20) & 0x7ff)
            if (i > 16) {
                t = r
                w = fn * fdPio2_2
                r = t - w
                w = fn * fdPio2_2t - ((t - r) - w)
                fdRY0 = r - w
                i = j - ((fdHi(fdRY0) >> 20) & 0x7ff)
                if (i > 49) {
                    t = r
                    w = fn * fdPio2_3
                    r = t - w
                    w = fn * fdPio2_3t - ((t - r) - w)
                    fdRY0 = r - w
                }
            }
        }
        fdRY1 = (r - fdRY0) - w
        if (hx < 0) {
            fdRY0 = fdNeg(fdRY0)
            fdRY1 = fdNeg(fdRY1)
            return 0 - n
        }
        return n
    }
    if (ix >= 0x7ff00000) {
        fdRY0 = x - x
        fdRY1 = fdRY0
        return 0
    }
    var e0 = (ix >> 20) - 1046
    z = fdW(ix - (e0 << 20), fdLo(x))
    var tx = [0, 0, 0]
    var k = 0
    while (k < 2) {
        tx[k] = Math.floor(z)
        z = (z - tx[k]) * fdTwo24
        k = k + 1
    }
    tx[2] = z
    var nx = 3
    while (tx[nx - 1] == 0) { nx = nx - 1 }
    n = fdKRemPio2(tx, nx, e0)
    if (hx < 0) {
        fdRY0 = fdNeg(fdRY0)
        fdRY1 = fdNeg(fdRY1)
        return 0 - n
    }
    return n
}

// k_rem_pio2.c at prec 2 (jk 4): the large-argument reduction over 24 bit
// chunks of 2/pi. Leaves its result in fdRY0 / fdRY1.
var fdPIo2 = []
function fdZeros(n) {
    var a = []
    var i = 0
    while (i < n) {
        a.push(0)
        i = i + 1
    }
    return a
}
function fdKRemPio2(x, nx, e0) {
    var iq = fdZeros(20)
    var f = fdZeros(20)
    var fq = fdZeros(20)
    var q = fdZeros(20)
    var jk = 4
    var jp = jk
    var jx = nx - 1
    var jv = fdTrunc((e0 - 3) / 24)
    if (jv < 0) { jv = 0 }
    var q0 = e0 - 24 * (jv + 1)
    var j = jv - jx
    var m = jx + jk
    var i = 0
    var fw = 0
    while (i <= m) {
        f[i] = j < 0 ? 0 : fdTwoOverPi[j]
        i = i + 1
        j = j + 1
    }
    i = 0
    while (i <= jk) {
        fw = 0
        j = 0
        while (j <= jx) {
            fw = fw + x[j] * f[jx + i - j]
            j = j + 1
        }
        q[i] = fw
        i = i + 1
    }
    var jz = jk
    var z = 0
    var n = 0
    var ih = 0
    var again = true
    while (again) {
        again = false
        i = 0
        z = q[jz]
        j = jz
        while (j > 0) {
            fw = fdTrunc(fdTwon24 * z)
            iq[i] = fdTrunc(z - fdTwo24 * fw)
            z = q[j - 1] + fw
            i = i + 1
            j = j - 1
        }
        z = fdScalb(z, q0)
        z = z - 8 * Math.floor(z * 0.125)
        n = fdTrunc(z)
        z = z - n
        ih = 0
        if (q0 > 0) {
            i = iq[jz - 1] >> (24 - q0)
            n = n + i
            iq[jz - 1] = iq[jz - 1] - (i << (24 - q0))
            ih = iq[jz - 1] >> (23 - q0)
        } else if (q0 == 0) {
            ih = iq[jz - 1] >> 23
        } else if (z >= 0.5) {
            ih = 2
        }
        if (ih > 0) {
            n = n + 1
            var carry = 0
            i = 0
            while (i < jz) {
                j = iq[i]
                if (carry == 0) {
                    if (j != 0) {
                        carry = 1
                        iq[i] = 0x1000000 - j
                    }
                } else {
                    iq[i] = 0xffffff - j
                }
                i = i + 1
            }
            if (q0 == 1) { iq[jz - 1] = iq[jz - 1] & 0x7fffff }
            if (q0 == 2) { iq[jz - 1] = iq[jz - 1] & 0x3fffff }
            if (ih == 2) {
                z = 1 - z
                if (carry != 0) { z = z - fdScalb(1, q0) }
            }
        }
        if (z == 0) {
            j = 0
            i = jz - 1
            while (i >= jk) {
                j = j | iq[i]
                i = i - 1
            }
            if (j == 0) {
                var kk = 1
                while (iq[jk - kk] == 0) { kk = kk + 1 }
                i = jz + 1
                while (i <= jz + kk) {
                    f[jx + i] = fdTwoOverPi[jv + i]
                    fw = 0
                    j = 0
                    while (j <= jx) {
                        fw = fw + x[j] * f[jx + i - j]
                        j = j + 1
                    }
                    q[i] = fw
                    i = i + 1
                }
                jz = jz + kk
                again = true
            }
        }
    }
    if (z == 0) {
        jz = jz - 1
        q0 = q0 - 24
        while (iq[jz] == 0) {
            jz = jz - 1
            q0 = q0 - 24
        }
    } else {
        z = fdScalb(z, 0 - q0)
        if (z >= fdTwo24) {
            fw = fdTrunc(fdTwon24 * z)
            iq[jz] = fdTrunc(z - fdTwo24 * fw)
            jz = jz + 1
            q0 = q0 + 24
            iq[jz] = fw
        } else {
            iq[jz] = fdTrunc(z)
        }
    }
    fw = fdScalb(1, q0)
    i = jz
    while (i >= 0) {
        q[i] = fw * iq[i]
        fw = fw * fdTwon24
        i = i - 1
    }
    i = jz
    while (i >= 0) {
        fw = 0
        var k = 0
        while (k <= jp && k <= jz - i) {
            fw = fw + fdPIo2[k] * q[i + k]
            k = k + 1
        }
        fq[jz - i] = fw
        i = i - 1
    }
    fw = 0
    i = jz
    while (i >= 0) {
        fw = fw + fq[i]
        i = i - 1
    }
    fdRY0 = ih == 0 ? fw : fdNeg(fw)
    fw = fq[0] - fw
    i = 1
    while (i <= jz) {
        fw = fw + fq[i]
        i = i + 1
    }
    fdRY1 = ih == 0 ? fw : fdNeg(fw)
    return n & 7
}

// s_sin.c, s_cos.c and s_tan.c.
function fdSin(x) {
    if (!fdReady) { fdInit() }
    var ix = fdHi(x) & 0x7fffffff
    if (ix <= 0x3fe921fb) { return fdKSin(x, 0, 0) }
    if (ix >= 0x7ff00000) { return x - x }
    var n = fdRemPio2(x) & 3
    if (n == 0) { return fdKSin(fdRY0, fdRY1, 1) }
    if (n == 1) { return fdKCos(fdRY0, fdRY1) }
    if (n == 2) { return fdNeg(fdKSin(fdRY0, fdRY1, 1)) }
    return fdNeg(fdKCos(fdRY0, fdRY1))
}
function fdCos(x) {
    if (!fdReady) { fdInit() }
    var ix = fdHi(x) & 0x7fffffff
    if (ix <= 0x3fe921fb) { return fdKCos(x, 0) }
    if (ix >= 0x7ff00000) { return x - x }
    var n = fdRemPio2(x) & 3
    if (n == 0) { return fdKCos(fdRY0, fdRY1) }
    if (n == 1) { return fdNeg(fdKSin(fdRY0, fdRY1, 1)) }
    if (n == 2) { return fdNeg(fdKCos(fdRY0, fdRY1)) }
    return fdKSin(fdRY0, fdRY1, 1)
}
function fdTan(x) {
    if (!fdReady) { fdInit() }
    var ix = fdHi(x) & 0x7fffffff
    if (ix <= 0x3fe921fb) { return fdKTan(x, 0, 1) }
    if (ix >= 0x7ff00000) { return x - x }
    var n = fdRemPio2(x)
    return fdKTan(fdRY0, fdRY1, 1 - ((n & 1) << 1))
}

// e_asin.c and e_acos.c, which share the rational approximation.
var fdPio2Hi = 0
var fdPio2Lo = 0
var fdPi = 0
var fdAsP = []
var fdAsQ = []
function fdAsin(x) {
    if (!fdReady) { fdInit() }
    var hx = fdHi(x)
    var ix = hx & 0x7fffffff
    var t = 0
    var w = 0
    var p = 0
    var q = 0
    if (ix >= 0x3ff00000) {
        if (((ix - 0x3ff00000) | fdLo(x)) == 0) { return x * fdPio2Hi + x * fdPio2Lo }
        return fdNaN
    } else if (ix < 0x3fe00000) {
        if (ix < 0x3e400000) {
            if (fdHuge + x > 1) { return x }
        }
        t = x * x
        p = t * fdPoly(t, fdAsP)
        q = fdPoly(t, fdAsQ)
        w = p / q
        return x + x * w
    }
    w = 1 - Math.abs(x)
    t = w * 0.5
    p = t * fdPoly(t, fdAsP)
    q = fdPoly(t, fdAsQ)
    var s = Math.sqrt(t)
    if (ix >= 0x3FEF3333) {
        w = p / q
        t = fdPio2Hi - (2 * (s + s * w) - fdPio2Lo)
    } else {
        w = fdChop(s)
        var c = (t - w * w) / (s + w)
        var r = p / q
        p = 2 * s * r - (fdPio2Lo - 2 * c)
        q = fdPio4 - 2 * w
        t = fdPio4 - (p - q)
    }
    if (hx > 0) { return t }
    return fdNeg(t)
}
function fdAcos(x) {
    if (!fdReady) { fdInit() }
    var hx = fdHi(x)
    var ix = hx & 0x7fffffff
    var z = 0
    var p = 0
    var q = 0
    var r = 0
    var s = 0
    var w = 0
    if (ix >= 0x3ff00000) {
        if (((ix - 0x3ff00000) | fdLo(x)) == 0) {
            if (hx > 0) { return 0 }
            return fdPi + 2 * fdPio2Lo
        }
        return fdNaN
    }
    if (ix < 0x3fe00000) {
        if (ix <= 0x3c600000) { return fdPio2Hi + fdPio2Lo }
        z = x * x
        p = z * fdPoly(z, fdAsP)
        q = fdPoly(z, fdAsQ)
        r = p / q
        return fdPio2Hi - (x - (fdPio2Lo - x * r))
    } else if (hx < 0) {
        z = (1 + x) * 0.5
        p = z * fdPoly(z, fdAsP)
        q = fdPoly(z, fdAsQ)
        s = Math.sqrt(z)
        r = p / q
        w = r * s - fdPio2Lo
        return fdPi - 2 * (s + w)
    }
    z = (1 - x) * 0.5
    s = Math.sqrt(z)
    var df = fdChop(s)
    var c = (z - df * df) / (s + df)
    p = z * fdPoly(z, fdAsP)
    q = fdPoly(z, fdAsQ)
    r = p / q
    w = r * s + c
    return 2 * (df + w)
}

// s_atan.c and e_atan2.c.
var fdAtanHi = []
var fdAtanLo = []
var fdAtOdd = []
var fdAtEven = []
function fdAtan(xIn) {
    if (!fdReady) { fdInit() }
    var x = xIn
    var hx = fdHi(x)
    var ix = hx & 0x7fffffff
    var id = 0
    if (ix >= 0x44100000) {
        if (x !== x) { return x + x }
        if (hx > 0) { return fdAtanHi[3] + fdAtanLo[3] }
        return fdNeg(fdAtanHi[3]) - fdAtanLo[3]
    }
    if (ix < 0x3fdc0000) {
        if (ix < 0x3e200000) {
            if (fdHuge + x > 1) { return x }
        }
        id = 0 - 1
    } else {
        x = Math.abs(x)
        if (ix < 0x3ff30000) {
            if (ix < 0x3fe60000) {
                id = 0
                x = (2 * x - 1) / (2 + x)
            } else {
                id = 1
                x = (x - 1) / (x + 1)
            }
        } else {
            if (ix < 0x40038000) {
                id = 2
                x = (x - 1.5) / (1 + 1.5 * x)
            } else {
                id = 3
                x = (0 - 1) / x
            }
        }
    }
    var z = x * x
    var w = z * z
    var s1 = z * fdPoly(w, fdAtOdd)
    var s2 = w * fdPoly(w, fdAtEven)
    if (id < 0) { return x - x * (s1 + s2) }
    z = fdAtanHi[id] - ((x * (s1 + s2) - fdAtanLo[id]) - x)
    if (hx < 0) { return fdNeg(z) }
    return z
}
var fdPiLo = 0
function fdAtan2(y, x) {
    if (!fdReady) { fdInit() }
    if (x !== x || y !== y) { return x + y }
    var hx = fdHi(x)
    var ix = hx & 0x7fffffff
    var lx = fdLo(x)
    var hy = fdHi(y)
    var iy = hy & 0x7fffffff
    var ly = fdLo(y)
    if (((hx - 0x3ff00000) | lx) == 0) { return fdAtan(y) }
    var m = ((hy >> 31) & 1) | ((hx >> 30) & 2)
    if ((iy | ly) == 0) {
        if (m == 0 || m == 1) { return y }
        if (m == 2) { return fdPi + fdTiny }
        return fdNeg(fdPi) - fdTiny
    }
    if ((ix | lx) == 0) {
        if (hy < 0) { return fdNeg(fdPio2Hi) - fdTiny }
        return fdPio2Hi + fdTiny
    }
    if (ix == 0x7ff00000) {
        if (iy == 0x7ff00000) {
            if (m == 0) { return fdPio4 + fdTiny }
            if (m == 1) { return fdNeg(fdPio4) - fdTiny }
            if (m == 2) { return 3 * fdPio4 + fdTiny }
            return (0 - 3) * fdPio4 - fdTiny
        }
        if (m == 0) { return 0 }
        if (m == 1) { return fdNegZero }
        if (m == 2) { return fdPi + fdTiny }
        return fdNeg(fdPi) - fdTiny
    }
    if (iy == 0x7ff00000) {
        if (hy < 0) { return fdNeg(fdPio2Hi) - fdTiny }
        return fdPio2Hi + fdTiny
    }
    var k = (iy - ix) >> 20
    var z = 0
    if (k > 60) {
        z = fdPio2Hi + 0.5 * fdPiLo
    } else if (hx < 0 && k < 0 - 60) {
        z = 0
    } else {
        z = fdAtan(Math.abs(y / x))
    }
    if (m == 0) { return z }
    if (m == 1) { return fdNeg(z) }
    if (m == 2) { return fdPi - (z - fdPiLo) }
    return (z - fdPiLo) - fdPi
}

// e_sinh.c, e_cosh.c and s_tanh.c.
var fdShuge = 0
function fdSinh(x) {
    if (!fdReady) { fdInit() }
    var jx = fdHi(x)
    var ix = jx & 0x7fffffff
    if (ix >= 0x7ff00000) { return x + x }
    var h = 0.5
    if (jx < 0) { h = 0 - 0.5 }
    var t = 0
    if (ix < 0x40360000) {
        if (ix < 0x3e300000) {
            if (fdShuge + x > 1) { return x }
        }
        t = fdExpm1(Math.abs(x))
        if (ix < 0x3ff00000) { return h * (2 * t - t * t / (t + 1)) }
        return h * (t + t / (t + 1))
    }
    if (ix < 0x40862E42) { return h * fdExp(Math.abs(x)) }
    var lx = fdLo(x)
    if (ix < 0x408633CE || (ix == 0x408633ce && lx <= 0x8fb9f87d)) {
        var w = fdExp(0.5 * Math.abs(x))
        t = h * w
        return t * w
    }
    return x * fdShuge
}
function fdCosh(x) {
    if (!fdReady) { fdInit() }
    var ix = fdHi(x) & 0x7fffffff
    if (ix >= 0x7ff00000) { return x * x }
    var t = 0
    var w = 0
    if (ix < 0x3fd62e43) {
        t = fdExpm1(Math.abs(x))
        w = 1 + t
        if (ix < 0x3c800000) { return w }
        return 1 + (t * t) / (w + w)
    }
    if (ix < 0x40360000) {
        t = fdExp(Math.abs(x))
        return 0.5 * t + 0.5 / t
    }
    if (ix < 0x40862E42) { return 0.5 * fdExp(Math.abs(x)) }
    var lx = fdLo(x)
    if (ix < 0x408633CE || (ix == 0x408633ce && lx <= 0x8fb9f87d)) {
        w = fdExp(0.5 * Math.abs(x))
        t = 0.5 * w
        return t * w
    }
    return fdInf
}
function fdTanh(x) {
    if (!fdReady) { fdInit() }
    var jx = fdHi(x)
    var ix = jx & 0x7fffffff
    if (ix >= 0x7ff00000) {
        if (jx >= 0) { return 1 / x + 1 }
        return 1 / x - 1
    }
    var z = 0
    var t = 0
    if (ix < 0x40360000) {
        if (ix < 0x3c800000) {
            if (x == 0) { return x }
            return x * (1 + x)
        }
        if (ix >= 0x3ff00000) {
            t = fdExpm1(2 * Math.abs(x))
            z = 1 - 2 / (t + 2)
        } else {
            t = fdExpm1((0 - 2) * Math.abs(x))
            z = fdNeg(t) / (t + 2)
        }
    } else {
        z = 1 - fdTiny
    }
    if (jx >= 0) { return z }
    return fdNeg(z)
}

// s_cbrt.c.
var fdCbrtC = 0
var fdCbrtD = 0
var fdCbrtE = 0
var fdCbrtF = 0
var fdCbrtG = 0
function fdCbrt(xIn) {
    if (!fdReady) { fdInit() }
    var x = xIn
    if (x == 0 || x !== x || x == fdInf || x == 0 - fdInf) { return x + x }
    var neg = x < 0
    x = Math.abs(x)
    var t = 0
    if (x < fdPow2(0 - 1022)) {
        t = fdTwo54 * x
        t = fdSetHi(t, fdTrunc(fdHi(t) / 3) + 696219795)
    } else {
        t = fdW(fdTrunc(fdHi(x) / 3) + 715094163, 0)
    }
    var r = t * t / x
    var s = fdCbrtC + r * t
    t = t * (fdCbrtG + fdCbrtF / (s + fdCbrtE + fdCbrtD / s))
    t = fdChop(t)
    t = fdSetHi(t, fdHi(t) + 1)
    s = t * t
    r = x / s
    var w = t + t
    r = (r - t) / (w + r)
    t = t + t * r
    if (neg) { return fdNeg(t) }
    return t
}

// e_hypot.c as FdLibm.Hypot spells it.
var fdHypBig = 0
function fdHypot(x, y) {
    if (!fdReady) { fdInit() }
    var a = Math.abs(x)
    var b = Math.abs(y)
    if (a == fdInf || b == fdInf) { return fdInf }
    if (a !== a || b !== b) { return a + b }
    if (b > a) {
        var tmp = a
        a = b
        b = tmp
    }
    var ha = fdHi(a)
    var hb = fdHi(b)
    if (ha - hb > 0x3c00000) { return a + b }
    var k = 0
    if (a > fdHypBig) {
        ha = ha - 0x25800000
        hb = hb - 0x25800000
        a = a * fdPow2(0 - 600)
        b = b * fdPow2(0 - 600)
        k = k + 600
    }
    var t1 = 0
    var t2 = 0
    if (b < fdPow2(0 - 500)) {
        if (b < fdPow2(0 - 1022)) {
            if (b == 0) { return a }
            t1 = fdPow2(1022)
            b = b * t1
            a = a * t1
            k = k - 1022
        } else {
            ha = ha + 0x25800000
            hb = hb + 0x25800000
            a = a * fdPow2(600)
            b = b * fdPow2(600)
            k = k - 600
        }
    }
    var w = a - b
    if (w > b) {
        t1 = fdW(ha, 0)
        t2 = a - t1
        w = Math.sqrt(t1 * t1 - (b * fdNeg(b) - t2 * (a + t1)))
    } else {
        a = a + a
        var y1 = fdW(hb, 0)
        var y2 = b - y1
        t1 = fdW(ha + 0x00100000, 0)
        t2 = a - t1
        w = Math.sqrt(t1 * y1 - (w * fdNeg(w) - (t1 * y2 + t2 * b)))
    }
    if (k != 0) { return fdScalb(1, k) * w }
    return w
}

// e_pow.c as FdLibm.Pow spells it.
var fdMaxD = 0
var fdPowYBig = 0
var fdPowXLo = 0
var fdPowXHi = 0
var fdPowInvLn2 = 0
var fdPowInvLn2H = 0
var fdPowInvLn2L = 0
var fdPowCp = 0
var fdPowCpH = 0
var fdPowCpL = 0
var fdPowL = []
var fdPowDpH = []
var fdPowDpL = []
var fdPowOvt = 0
var fdPowP = []
var fdPowLg2 = 0
var fdPowLg2H = 0
var fdPowLg2L = 0
// s * z, the sign of a zero included.
function fdSigned(s, z) {
    if (s < 0) { return fdNeg(z) }
    return z
}
function fdPow(x, y) {
    if (!fdReady) { fdInit() }
    if (y == 0) { return 1 }
    if (x !== x || y !== y) { return x + y }
    var yAbs = Math.abs(y)
    var xAbs = Math.abs(x)
    if (y == 2) { return x * x }
    if (y == 0.5) {
        if (x >= fdNeg(fdMaxD)) { return Math.sqrt(x + 0) }
    } else if (yAbs == 1) {
        if (y == 1) { return x }
        return 1 / x
    } else if (yAbs == fdInf) {
        if (xAbs == 1) { return y - y }
        if (xAbs > 1) {
            if (y >= 0) { return y }
            return 0
        }
        if (y < 0) { return fdNeg(y) }
        return 0
    }
    var hx = fdHi(x)
    var ix = hx & 0x7fffffff
    var yIsInt = 0
    if (hx < 0) {
        if (yAbs >= fdPow2(53)) {
            yIsInt = 2
        } else if (yAbs >= 1) {
            var yl = Math.floor(yAbs)
            if (yl == yAbs) { yIsInt = 2 - (yl - 2 * Math.floor(yl / 2)) }
        }
    }
    var z = 0
    if (xAbs == 0 || xAbs == fdInf || xAbs == 1) {
        z = xAbs
        if (y < 0) { z = 1 / z }
        if (hx < 0) {
            if (((ix - 0x3ff00000) | yIsInt) == 0) {
                z = fdNaN
            } else if (yIsInt == 1) {
                z = fdNeg(z)
            }
        }
        return z
    }
    var n = (hx >> 31) + 1
    if ((n | yIsInt) == 0) { return fdNaN }
    var s = 1
    if ((n | (yIsInt - 1)) == 0) { s = 0 - 1 }
    var pH = 0
    var pL = 0
    var t1 = 0
    var t2 = 0
    var t = 0
    var u = 0
    var v = 0
    var w = 0
    if (yAbs > fdPowYBig) {
        if (xAbs < fdPowXLo) {
            if (y < 0) { return s * fdInf }
            return fdSigned(s, 0)
        }
        if (xAbs > fdPowXHi) {
            if (y > 0) { return s * fdInf }
            return fdSigned(s, 0)
        }
        t = xAbs - 1
        w = t * t * (0.5 - t * (0.3333333333333333333333 - t * 0.25))
        u = fdPowInvLn2H * t
        v = t * fdPowInvLn2L - w * fdPowInvLn2
        t1 = u + v
        t1 = fdChop(t1)
        t2 = v - (t1 - u)
    } else {
        n = 0
        if (ix < 0x00100000) {
            xAbs = xAbs * fdPow2(53)
            n = n - 53
            ix = fdHi(xAbs)
        }
        n = n + (ix >> 20) - 0x3ff
        var j0 = ix & 0x000fffff
        ix = j0 | 0x3ff00000
        var k0 = 0
        if (j0 <= 0x3988E) {
            k0 = 0
        } else if (j0 < 0xBB67A) {
            k0 = 1
        } else {
            k0 = 0
            n = n + 1
            ix = ix - 0x00100000
        }
        xAbs = fdSetHi(xAbs, ix)
        var bp = k0 == 0 ? 1 : 1.5
        u = xAbs - bp
        v = 1 / (xAbs + bp)
        var ss = u * v
        var sH = fdChop(ss)
        var tH = fdW(((ix >> 1) | 0x20000000) + 0x00080000 + (k0 << 18), 0)
        var tL = xAbs - (tH - bp)
        var sL = v * ((u - sH * tH) - sH * tL)
        var s2 = ss * ss
        var r0 = s2 * s2 * fdPoly(s2, fdPowL)
        r0 = r0 + sL * (sH + ss)
        s2 = sH * sH
        tH = 3 + s2 + r0
        tH = fdChop(tH)
        tL = r0 - ((tH - 3) - s2)
        u = sH * tH
        v = sL * tH + tL * ss
        pH = u + v
        pH = fdChop(pH)
        pL = v - (pH - u)
        var zH = fdPowCpH * pH
        var zL = fdPowCpL * pH + pL * fdPowCp + fdPowDpL[k0]
        t = n
        t1 = ((zH + zL) + fdPowDpH[k0]) + t
        t1 = fdChop(t1)
        t2 = zL - (((t1 - t) - fdPowDpH[k0]) - zH)
    }
    var y1 = fdChop(y)
    pL = (y - y1) * t1 + y * t2
    pH = y1 * t1
    z = pL + pH
    var j = fdHi(z)
    var i = fdLo(z)
    if (j >= 0x40900000) {
        if (((j - 0x40900000) | i) != 0) { return s * fdInf }
        if (pL + fdPowOvt > z - pH) { return s * fdInf }
    } else if ((j & 0x7fffffff) >= 0x4090cc00) {
        if (((j - (0 - 1064252416)) | i) != 0) { return fdSigned(s, 0) }
        if (pL <= z - pH) { return fdSigned(s, 0) }
    }
    i = j & 0x7fffffff
    var k = (i >> 20) - 0x3ff
    n = 0
    if (i > 0x3fe00000) {
        n = j + (0x00100000 >> (k + 1))
        k = ((n & 0x7fffffff) >> 20) - 0x3ff
        t = fdW(n & ~(0x000fffff >> k), 0)
        n = ((n & 0x000fffff) | 0x00100000) >> (20 - k)
        if (j < 0) { n = 0 - n }
        pH = pH - t
    }
    t = pL + pH
    t = fdChop(t)
    u = t * fdPowLg2H
    v = (pL - (t - pH)) * fdPowLg2 + t * fdPowLg2L
    z = u + v
    w = v - (z - u)
    t = z * z
    t1 = z - t * fdPoly(t, fdPowP)
    var r = (z * t1) / (t1 - 2) - (w + z * w)
    z = 1 - (r - z)
    j = fdHi(z) + (n << 20)
    if ((j >> 20) <= 0) {
        z = fdScalb(z, n)
    } else {
        z = fdSetHi(z, fdHi(z) + (n << 20))
    }
    return fdSigned(s, z)
}

// The constants, built at the first call of any function here.
var fdReady = false
function fdInit() {
    fdReady = true
    fdLn2Hi = fdW(0x3fe62e42, 0xfee00000)
    fdLn2Lo = fdW(0x3dea39ef, 0x35793c76)
    fdTwo54 = fdPow2(54)
    fdHuge = fdW(0x7e37e43c, 0x8800759c)
    fdTiny = fdW(0x01a56e1f, 0xc2f8f359)
    fdInvLn2 = fdW(0x3ff71547, 0x652b82fe)
    fdExpP = [fdW(0x3fc55555, 0x5555553e), fdW(0xbf66c16c, 0x16bebd93), fdW(0x3f11566a, 0xaf25de2c),
              fdW(0xbebbbd41, 0xc5d26bf1), fdW(0x3e663769, 0x72bea4d0)]
    fdExpOHi = fdW(0x40862e42, 0xfefa39ef)
    fdExpULo = fdW(0xc0874910, 0xd52d3051)
    fdLgA = [fdW(0x3fd99999, 0x9997fa04), fdW(0x3fcc71c5, 0x1d8e78af), fdW(0x3fc39a09, 0xd078c69f)]
    fdLgB = [fdW(0x3fe55555, 0x55555593), fdW(0x3fd24924, 0x94229359), fdW(0x3fc74664, 0x96cb03de),
             fdW(0x3fc2f112, 0xdf3e5244)]
    fdIvLn10 = fdW(0x3fdbcb7b, 0x1526e50e)
    fdLog10Hi = fdW(0x3fd34413, 0x509f6000)
    fdLog10Lo = fdW(0x3d59fef3, 0x11f12b36)
    fdLp = [fdW(0x3fe55555, 0x55555593), fdW(0x3fd99999, 0x9997fa04), fdW(0x3fd24924, 0x94229359),
            fdW(0x3fcc71c5, 0x1d8e78af), fdW(0x3fc74664, 0x96cb03de), fdW(0x3fc39a09, 0xd078c69f),
            fdW(0x3fc2f112, 0xdf3e5244)]
    fdExpm1Q = [1, fdW(0xbfa11111, 0x111110f4), fdW(0x3f5a01a0, 0x19fe5585), fdW(0xbf14ce19, 0x9eaadbb7),
                fdW(0x3ed0cfca, 0x86e65239), fdW(0xbe8afdb7, 0x6e09c32d)]
    fdS1 = fdW(0xbfc55555, 0x55555549)
    fdSinP = [fdW(0x3f811111, 0x1110f8a6), fdW(0xbf2a01a0, 0x19c161d5), fdW(0x3ec71de3, 0x57b1fe7d),
              fdW(0xbe5ae5e6, 0x8a2b9ceb), fdW(0x3de5d93a, 0x5acfd57c)]
    fdCosP = [fdW(0x3fa55555, 0x5555554c), fdW(0xbf56c16c, 0x16c15177), fdW(0x3efa01a0, 0x19cb1590),
              fdW(0xbe927e4f, 0x809c52ad), fdW(0x3e21ee9e, 0xbdb4b1c4), fdW(0xbda8fae9, 0xbe8838d4)]
    fdTanT = [fdW(0x3fd55555, 0x55555563), fdW(0x3fc11111, 0x1110fe7a), fdW(0x3faba1ba, 0x1bb341fe),
              fdW(0x3f9664f4, 0x8406d637), fdW(0x3f8226e3, 0xe96e8493), fdW(0x3f6d6d22, 0xc9560328),
              fdW(0x3f57dbc8, 0xfee08315), fdW(0x3f4344d8, 0xf2f26501), fdW(0x3f3026f7, 0x1a8d1068),
              fdW(0x3f147e88, 0xa03792a6), fdW(0x3f12b80f, 0x32f0a7e9), fdW(0xbef375cb, 0xdb605373),
              fdW(0x3efb2a70, 0x74bf7ad4)]
    fdTanOdd = [fdTanT[1], fdTanT[3], fdTanT[5], fdTanT[7], fdTanT[9], fdTanT[11]]
    fdTanEven = [fdTanT[2], fdTanT[4], fdTanT[6], fdTanT[8], fdTanT[10], fdTanT[12]]
    fdPio4 = fdW(0x3fe921fb, 0x54442d18)
    fdPio4Lo = fdW(0x3c81a626, 0x33145c07)
    fdTwoOverPi = [
                   0xA2F983, 0x6E4E44, 0x1529FC, 0x2757D1, 0xF534DD, 0xC0DB62,
                   0x95993C, 0x439041, 0xFE5163, 0xABDEBB, 0xC561B7, 0x246E3A,
                   0x424DD2, 0xE00649, 0x2EEA09, 0xD1921C, 0xFE1DEB, 0x1CB129,
                   0xA73EE8, 0x8235F5, 0x2EBB44, 0x84E99C, 0x7026B4, 0x5F7E41,
                   0x3991D6, 0x398353, 0x39F49C, 0x845F8B, 0xBDF928, 0x3B1FF8,
                   0x97FFDE, 0x05980F, 0xEF2F11, 0x8B5A0A, 0x6D1F6D, 0x367ECF,
                   0x27CB09, 0xB74F46, 0x3F669E, 0x5FEA2D, 0x7527BA, 0xC7EBE5,
                   0xF17B3D, 0x0739F7, 0x8A5292, 0xEA6BFB, 0x5FB11F, 0x8D5D08,
                   0x560330, 0x46FC7B, 0x6BABF0, 0xCFBC20, 0x9AF436, 0x1DA9E3,
                   0x91615E, 0xE61B08, 0x659985, 0x5F14A0, 0x68408D, 0xFFD880,
                   0x4D7327, 0x310606, 0x1556CA, 0x73A8C9, 0x60E27B, 0xC08C6B]
    fdNpio2Hw = [
                 0x3FF921FB, 0x400921FB, 0x4012D97C, 0x401921FB, 0x401F6A7A, 0x4022D97C,
                 0x4025FDBB, 0x402921FB, 0x402C463A, 0x402F6A7A, 0x4031475C, 0x4032D97C,
                 0x40346B9C, 0x4035FDBB, 0x40378FDB, 0x403921FB, 0x403AB41B, 0x403C463A,
                 0x403DD85A, 0x403F6A7A, 0x40407E4C, 0x4041475C, 0x4042106C, 0x4042D97C,
                 0x4043A28C, 0x40446B9C, 0x404534AC, 0x4045FDBB, 0x4046C6CB, 0x40478FDB,
                 0x404858EB, 0x404921FB]
    fdInvPio2 = fdW(0x3fe45f30, 0x6dc9c883)
    fdPio2_1 = fdW(0x3ff921fb, 0x54400000)
    fdPio2_1t = fdW(0x3dd0b461, 0x1a626331)
    fdPio2_2 = fdW(0x3dd0b461, 0x1a600000)
    fdPio2_2t = fdW(0x3ba3198a, 0x2e037073)
    fdPio2_3 = fdW(0x3ba3198a, 0x2e000000)
    fdPio2_3t = fdW(0x397b839a, 0x252049c1)
    fdTwo24 = 16777216
    fdTwon24 = fdPow2(0 - 24)
    fdPIo2 = [fdW(0x3ff921fb, 0x40000000), fdW(0x3e74442d, 0x00000000), fdW(0x3cf84698, 0x80000000),
              fdW(0x3b78cc51, 0x60000000), fdW(0x39f01b83, 0x80000000), fdW(0x387a2520, 0x40000000),
              fdW(0x36e38222, 0x80000000), fdW(0x3569f31d, 0x00000000)]
    fdPio2Hi = fdW(0x3ff921fb, 0x54442d18)
    fdPio2Lo = fdW(0x3c91a626, 0x33145c07)
    fdPi = fdW(0x400921fb, 0x54442d18)
    fdAsP = [fdW(0x3fc55555, 0x55555555), fdW(0xbfd4d612, 0x03eb6f7d), fdW(0x3fc9c155, 0x0e884455),
             fdW(0xbfa48228, 0xb5688f3b), fdW(0x3f49efe0, 0x7501b288), fdW(0x3f023de1, 0x0dfdf709)]
    fdAsQ = [1, fdW(0xc0033a27, 0x1c8a2d4b), fdW(0x40002ae5, 0x9c598ac8), fdW(0xbfe6066c, 0x1b8d0159),
             fdW(0x3fb3b8c5, 0xb12e9282)]
    fdAtanHi = [fdW(0x3fddac67, 0x0561bb4f), fdW(0x3fe921fb, 0x54442d18), fdW(0x3fef730b, 0xd281f69b),
                fdW(0x3ff921fb, 0x54442d18)]
    fdAtanLo = [fdW(0x3c7a2b7f, 0x222f65e2), fdW(0x3c81a626, 0x33145c07), fdW(0x3c700788, 0x7af0cbbd),
                fdW(0x3c91a626, 0x33145c07)]
    fdAtOdd = [fdW(0x3fd55555, 0x5555550d), fdW(0x3fc24924, 0x920083ff), fdW(0x3fb745cd, 0xc54c206e),
               fdW(0x3fb10d66, 0xa0d03d51), fdW(0x3fa97b4b, 0x24760deb), fdW(0x3f90ad3a, 0xe322da11)]
    fdAtEven = [fdW(0xbfc99999, 0x9998ebc4), fdW(0xbfbc71c6, 0xfe231671), fdW(0xbfb3b0f2, 0xaf749a6d),
                fdW(0xbfadde2d, 0x52defd9a), fdW(0xbfa2b444, 0x2c6a6c2f)]
    fdPiLo = fdW(0x3ca1a626, 0x33145c07)
    fdShuge = fdW(0x7fac7b1f, 0x3cac7433)
    fdCbrtC = fdW(0x3fe15f15, 0xf15f15f1)
    fdCbrtD = fdW(0xbfe691de, 0x2532c834)
    fdCbrtE = fdW(0x3ff6a0ea, 0x0ea0ea0f)
    fdCbrtF = fdW(0x3ff9b6db, 0x6db6db6e)
    fdCbrtG = fdW(0x3fd6db6d, 0xb6db6db7)
    fdHypBig = fdW(0x5f300000, 0xffffffff)
    fdMaxD = fdW(0x7fefffff, 0xffffffff)
    fdPowYBig = fdW(0x41e00000, 0xffffffff)
    fdPowXLo = fdW(0x3fefffff, 0x00000000)
    fdPowXHi = fdW(0x3ff00000, 0xffffffff)
    fdPowInvLn2 = fdW(0x3ff71547, 0x652b82fe)
    fdPowInvLn2H = fdW(0x3ff71547, 0x60000000)
    fdPowInvLn2L = fdW(0x3e54ae0b, 0xf85ddf44)
    fdPowCp = fdW(0x3feec709, 0xdc3a03fd)
    fdPowCpH = fdW(0x3feec709, 0xe0000000)
    fdPowCpL = fdW(0xbe3e2fe0, 0x145b01f5)
    fdPowL = [fdW(0x3fe33333, 0x33333303), fdW(0x3fdb6db6, 0xdb6fabff), fdW(0x3fd55555, 0x518f264d),
              fdW(0x3fd17460, 0xa91d4101), fdW(0x3fcd864a, 0x93c9db65), fdW(0x3fca7e28, 0x4a454eef)]
    fdPowDpH = [0, fdW(0x3fe2b803, 0x40000000)]
    fdPowDpL = [0, fdW(0x3e4cfdeb, 0x43cfd006)]
    fdPowOvt = fdW(0x3c971547, 0x652b82fe)
    fdPowP = [fdW(0x3fc55555, 0x5555553e), fdW(0xbf66c16c, 0x16bebd93), fdW(0x3f11566a, 0xaf25de2c),
              fdW(0xbebbbd41, 0xc5d26bf1), fdW(0x3e663769, 0x72bea4d0)]
    fdPowLg2 = fdW(0x3fe62e42, 0xfefa39ef)
    fdPowLg2H = fdW(0x3fe62e43, 0x00000000)
    fdPowLg2L = fdW(0xbe205c61, 0x0ca86c39)
}
