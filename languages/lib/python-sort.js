// python-sort.js -- CPython's list.sort (listsort in Objects/listobject.c,
// 3.14), for python-interpreter.abnf (include) and languages/lib/python-rt.metajs
// (import). Go twin: abnf/jsrtpysort.go.
//
// Timsort is followed comparison for comparison - count_run, binary insertion
// up to minrun, the powersort merge policy and the galloping merges - because
// the sequence of < calls is observable: a key or __lt__ with side effects
// sees it, and with a NaN (no total order) it decides the result, so
// sorted([3, nan, 1, 2]) stays [3, nan, 1, 2]. A raising comparison leaves the
// arrays some permutation of their input, as CPython's does.
//
// pysoSort(k, v, lt) sorts the keys k ascending by lt(x, y) (x < y, a boolean)
// and permutes the values v in lockstep; v may be k itself.

function pysoSort(k, v, lt) {
    var n = k.length
    if (n < 2) { return }
    var ms = {k: k, v: v, lt: lt, minGallop: 7, n: n, base: [], len: [], power: [], tk: [], tv: []}
    var minrun = pysoMinrun(n)
    var lo = 0
    var rem = n
    while (rem > 0) {
        var r = pysoCountRun(ms, lo, rem)
        if (r < minrun) {
            var force = minrun
            if (rem <= minrun) { force = rem }
            pysoBinarySort(ms, lo, force, r)
            r = force
        }
        pysoFoundNewRun(ms, r)
        ms.base.push(lo)
        ms.len.push(r)
        ms.power.push(0)
        lo = lo + r
        rem = rem - r
    }
    while (ms.base.length > 1) {
        var i = ms.base.length - 2
        if (i > 0 && ms.len[i - 1] < ms.len[i + 1]) { i = i - 1 }
        pysoMergeAt(ms, i)
    }
}

// n itself below 64, else a length in 32..64 that divides n into close to,
// but strictly less than, a power of two runs.
function pysoMinrun(n0) {
    var n = n0
    var r = 0
    while (n >= 64) {
        if (n % 2 == 1) { r = 1 }
        n = Math.floor(n / 2)
    }
    return n + r
}

// Reverse a[lo:lo+n] (and the values beside it).
function pysoRev(a, lo, n) {
    var i = lo
    var j = lo + n - 1
    while (i < j) {
        var t = anytype
        t = a[i]
        a[i] = a[j]
        a[j] = t
        i = i + 1
        j = j - 1
    }
}
function pysoRevKV(ms, lo, n) {
    pysoRev(ms.k, lo, n)
    if (ms.v !== ms.k) { pysoRev(ms.v, lo, n) }
}

// The length of the run at lo, at most rem long, made ascending in place: a
// descending run is reversed, its all-equal stretches reversed first so the
// whole reversal keeps them stable.
function pysoCountRun(ms, lo, rem) {
    var k = ms.k
    var lt = ms.lt
    var n = 1
    while (n < rem) {
        if (lt(k[lo + n], k[lo + n - 1])) { break }
        n = n + 1
    }
    if (n == rem) { return n }
    if (n > 1) {
        if (lt(k[lo], k[lo + n - 1])) { return n }
        pysoRevKV(ms, lo, n)
    }
    n = n + 1
    var neq = 0
    while (n < rem) {
        if (lt(k[lo + n], k[lo + n - 1])) {
            if (neq > 0) {
                neq = neq + 1
                pysoRevKV(ms, lo + n - neq, neq)
                neq = 0
            }
        } else {
            if (lt(k[lo + n - 1], k[lo + n])) { break }
            neq = neq + 1
        }
        n = n + 1
    }
    if (neq > 0) {
        neq = neq + 1
        pysoRevKV(ms, lo + n - neq, neq)
    }
    pysoRevKV(ms, lo, n)
    while (n < rem) {
        if (lt(k[lo + n], k[lo + n - 1])) { break }
        n = n + 1
    }
    return n
}

// Binary insertion of k[lo+ok:lo+n] into the sorted k[lo:lo+ok]. Every
// comparison for one element comes before any element moves.
function pysoBinarySort(ms, lo, n, ok0) {
    var k = ms.k
    var v = ms.v
    var ok = ok0
    if (ok == 0) { ok = 1 }
    while (ok < n) {
        var l = 0
        var r = ok
        var pivot = anytype
        pivot = k[lo + ok]
        while (l < r) {
            var m = Math.floor((l + r) / 2)
            if (ms.lt(pivot, k[lo + m])) { r = m } else { l = m + 1 }
        }
        var vp = anytype
        vp = v[lo + ok]
        var j = ok
        while (j > l) {
            k[lo + j] = k[lo + j - 1]
            v[lo + j] = v[lo + j - 1]
            j = j - 1
        }
        k[lo + l] = pivot
        v[lo + l] = vp
        ok = ok + 1
    }
}

// The powersort depth of the run at s1 (length n1) followed by one of n2.
function pysoPower(s1, n1, n2, n) {
    var result = 0
    var a = 2 * s1 + n1
    var b = a + n1 + n2
    while (true) {
        result = result + 1
        if (a >= n) {
            a = a - n
            b = b - n
        } else if (b >= n) {
            break
        }
        a = a * 2
        b = b * 2
    }
    return result
}

function pysoFoundNewRun(ms, n2) {
    var top = ms.base.length - 1
    if (top < 0) { return }
    var power = pysoPower(ms.base[top], ms.len[top], n2, ms.n)
    while (ms.base.length > 1 && ms.power[ms.base.length - 2] > power) {
        pysoMergeAt(ms, ms.base.length - 2)
    }
    ms.power[ms.base.length - 1] = power
}

// Where key goes in a[a0:a0+n], left of any equal element, searched from
// a0+hint.
function pysoGallopLeft(ms, key, a, a0, n, hint) {
    var lt = ms.lt
    var p = a0 + hint
    var lastofs = 0
    var ofs = 1
    var maxofs = 0
    if (lt(a[p], key)) {
        maxofs = n - hint
        while (ofs < maxofs) {
            if (!lt(a[p + ofs], key)) { break }
            lastofs = ofs
            ofs = ofs * 2 + 1
        }
        if (ofs > maxofs) { ofs = maxofs }
        lastofs = lastofs + hint
        ofs = ofs + hint
    } else {
        maxofs = hint + 1
        while (ofs < maxofs) {
            if (lt(a[p - ofs], key)) { break }
            lastofs = ofs
            ofs = ofs * 2 + 1
        }
        if (ofs > maxofs) { ofs = maxofs }
        var t = lastofs
        lastofs = hint - ofs
        ofs = hint - t
    }
    lastofs = lastofs + 1
    while (lastofs < ofs) {
        var m = lastofs + Math.floor((ofs - lastofs) / 2)
        if (lt(a[a0 + m], key)) { lastofs = m + 1 } else { ofs = m }
    }
    return ofs
}

// The same, right of any equal element.
function pysoGallopRight(ms, key, a, a0, n, hint) {
    var lt = ms.lt
    var p = a0 + hint
    var lastofs = 0
    var ofs = 1
    var maxofs = 0
    if (lt(key, a[p])) {
        maxofs = hint + 1
        while (ofs < maxofs) {
            if (!lt(key, a[p - ofs])) { break }
            lastofs = ofs
            ofs = ofs * 2 + 1
        }
        if (ofs > maxofs) { ofs = maxofs }
        var t = lastofs
        lastofs = hint - ofs
        ofs = hint - t
    } else {
        maxofs = n - hint
        while (ofs < maxofs) {
            if (lt(key, a[p + ofs])) { break }
            lastofs = ofs
            ofs = ofs * 2 + 1
        }
        if (ofs > maxofs) { ofs = maxofs }
        lastofs = lastofs + hint
        ofs = ofs + hint
    }
    lastofs = lastofs + 1
    while (lastofs < ofs) {
        var m = lastofs + Math.floor((ofs - lastofs) / 2)
        if (lt(key, a[a0 + m])) { ofs = m } else { lastofs = m + 1 }
    }
    return ofs
}

// Copy n keys and values from (sk, sv)[si:] to (dk, dv)[di:], ascending or,
// for an overlapping move to the right, descending.
function pysoCopy(dk, dv, di, sk, sv, si, n, down) {
    var i = 0
    while (i < n) {
        var x = i
        if (down) { x = n - 1 - i }
        dk[di + x] = sk[si + x]
        dv[di + x] = sv[si + x]
        i = i + 1
    }
}

// Merge the pending runs i and i+1.
function pysoMergeAt(ms, i) {
    var sa = ms.base[i]
    var na = ms.len[i]
    var sb = ms.base[i + 1]
    var nb = ms.len[i + 1]
    ms.len[i] = na + nb
    if (i == ms.base.length - 3) {
        ms.base[i + 1] = ms.base[i + 2]
        ms.len[i + 1] = ms.len[i + 2]
        ms.power[i + 1] = ms.power[i + 2]
    }
    ms.base.pop()
    ms.len.pop()
    ms.power.pop()
    var g = pysoGallopRight(ms, ms.k[sb], ms.k, sa, na, 0)
    sa = sa + g
    na = na - g
    if (na == 0) { return }
    nb = pysoGallopLeft(ms, ms.k[sa + na - 1], ms.k, sb, nb, nb - 1)
    if (nb <= 0) { return }
    if (na <= nb) { pysoMergeLo(ms, sa, na, sb, nb) } else { pysoMergeHi(ms, sa, na, sb, nb) }
}

// Merge k[sa:sa+na] with the k[sb:sb+nb] after it, na <= nb, through a copy of
// the first run.
function pysoMergeLo(ms, sa, na0, sb, nb0) {
    var k = ms.k
    var v = ms.v
    var tk = ms.tk
    var tv = ms.tv
    var na = na0
    var nb = nb0
    pysoCopy(tk, tv, 0, k, v, sa, na, false)
    var d = sa
    var pa = 0
    var pb = sb
    k[d] = k[pb]
    v[d] = v[pb]
    d = d + 1
    pb = pb + 1
    nb = nb - 1
    var copyB = false
    var done = nb == 0
    if (!done && na == 1) {
        copyB = true
        done = true
    }
    var mg = ms.minGallop
    try {
        while (!done) {
            var acount = 0
            var bcount = 0
            while (true) {
                if (ms.lt(k[pb], tk[pa])) {
                    k[d] = k[pb]
                    v[d] = v[pb]
                    d = d + 1
                    pb = pb + 1
                    bcount = bcount + 1
                    acount = 0
                    nb = nb - 1
                    if (nb == 0) {
                        done = true
                        break
                    }
                    if (bcount >= mg) { break }
                } else {
                    k[d] = tk[pa]
                    v[d] = tv[pa]
                    d = d + 1
                    pa = pa + 1
                    acount = acount + 1
                    bcount = 0
                    na = na - 1
                    if (na == 1) {
                        copyB = true
                        done = true
                        break
                    }
                    if (acount >= mg) { break }
                }
            }
            if (done) { break }
            mg = mg + 1
            while (true) {
                if (mg > 1) { mg = mg - 1 }
                ms.minGallop = mg
                var g = pysoGallopRight(ms, k[pb], tk, pa, na, 0)
                acount = g
                if (g > 0) {
                    pysoCopy(k, v, d, tk, tv, pa, g, false)
                    d = d + g
                    pa = pa + g
                    na = na - g
                    if (na == 1) {
                        copyB = true
                        done = true
                        break
                    }
                    if (na == 0) {
                        done = true
                        break
                    }
                }
                k[d] = k[pb]
                v[d] = v[pb]
                d = d + 1
                pb = pb + 1
                nb = nb - 1
                if (nb == 0) {
                    done = true
                    break
                }
                g = pysoGallopLeft(ms, tk[pa], k, pb, nb, 0)
                bcount = g
                if (g > 0) {
                    pysoCopy(k, v, d, k, v, pb, g, false)
                    d = d + g
                    pb = pb + g
                    nb = nb - g
                    if (nb == 0) {
                        done = true
                        break
                    }
                }
                k[d] = tk[pa]
                v[d] = tv[pa]
                d = d + 1
                pa = pa + 1
                na = na - 1
                if (na == 1) {
                    copyB = true
                    done = true
                    break
                }
                if (acount < 7 && bcount < 7) { break }
            }
            if (done) { break }
            mg = mg + 1
            ms.minGallop = mg
        }
    } catch (e) {
        pysoCopy(k, v, d, tk, tv, pa, na, false)
        throw e
    }
    if (copyB) {
        pysoCopy(k, v, d, k, v, pb, nb, false)
        k[d + nb] = tk[pa]
        v[d + nb] = tv[pa]
    } else {
        pysoCopy(k, v, d, tk, tv, pa, na, false)
    }
}

// Merge k[sa:sa+na] with the k[sb:sb+nb] after it, na > nb, from the right
// through a copy of the second run.
function pysoMergeHi(ms, sa, na0, sb, nb0) {
    var k = ms.k
    var v = ms.v
    var tk = ms.tk
    var tv = ms.tv
    var na = na0
    var nb = nb0
    pysoCopy(tk, tv, 0, k, v, sb, nb, false)
    var d = sb + nb - 1
    var pa = sa + na - 1
    var pb = nb - 1
    k[d] = k[pa]
    v[d] = v[pa]
    d = d - 1
    pa = pa - 1
    na = na - 1
    var copyA = false
    var done = na == 0
    if (!done && nb == 1) {
        copyA = true
        done = true
    }
    var mg = ms.minGallop
    try {
        while (!done) {
            var acount = 0
            var bcount = 0
            while (true) {
                if (ms.lt(tk[pb], k[pa])) {
                    k[d] = k[pa]
                    v[d] = v[pa]
                    d = d - 1
                    pa = pa - 1
                    acount = acount + 1
                    bcount = 0
                    na = na - 1
                    if (na == 0) {
                        done = true
                        break
                    }
                    if (acount >= mg) { break }
                } else {
                    k[d] = tk[pb]
                    v[d] = tv[pb]
                    d = d - 1
                    pb = pb - 1
                    bcount = bcount + 1
                    acount = 0
                    nb = nb - 1
                    if (nb == 1) {
                        copyA = true
                        done = true
                        break
                    }
                    if (bcount >= mg) { break }
                }
            }
            if (done) { break }
            mg = mg + 1
            while (true) {
                if (mg > 1) { mg = mg - 1 }
                ms.minGallop = mg
                var g = na - pysoGallopRight(ms, tk[pb], k, sa, na, na - 1)
                acount = g
                if (g > 0) {
                    d = d - g
                    pa = pa - g
                    pysoCopy(k, v, d + 1, k, v, pa + 1, g, true)
                    na = na - g
                    if (na == 0) {
                        done = true
                        break
                    }
                }
                k[d] = tk[pb]
                v[d] = tv[pb]
                d = d - 1
                pb = pb - 1
                nb = nb - 1
                if (nb == 1) {
                    copyA = true
                    done = true
                    break
                }
                g = nb - pysoGallopLeft(ms, k[pa], tk, 0, nb, nb - 1)
                bcount = g
                if (g > 0) {
                    d = d - g
                    pb = pb - g
                    pysoCopy(k, v, d + 1, tk, tv, pb + 1, g, false)
                    nb = nb - g
                    if (nb == 1) {
                        copyA = true
                        done = true
                        break
                    }
                    if (nb == 0) {
                        done = true
                        break
                    }
                }
                k[d] = k[pa]
                v[d] = v[pa]
                d = d - 1
                pa = pa - 1
                na = na - 1
                if (na == 0) {
                    done = true
                    break
                }
                if (acount < 7 && bcount < 7) { break }
            }
            if (done) { break }
            mg = mg + 1
            ms.minGallop = mg
        }
    } catch (e) {
        pysoCopy(k, v, d - nb + 1, tk, tv, 0, nb, false)
        throw e
    }
    if (copyA) {
        pysoCopy(k, v, d + 1 - na, k, v, pa + 1 - na, na, true)
        d = d - na
        k[d] = tk[pb]
        v[d] = tv[pb]
    } else {
        pysoCopy(k, v, d - nb + 1, tk, tv, 0, nb, false)
    }
}
