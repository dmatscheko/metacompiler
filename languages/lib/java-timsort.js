// java-timsort.js -- java.util.TimSort (JDK 24), the sort behind Arrays.sort(Object[]),
// List.sort and so kotlin's sorted / sortedBy / sortedWith / sortWith, for
// kotlin-interpreter.abnf (include) and languages/lib/kotlin-rt.metajs (import).
// Go twin: abnf/jsrtjtimsort.go.
//
// Followed comparison for comparison - countRunAndMakeAscending, binary insertion
// below MIN_MERGE (32), minRunLength, mergeCollapse, and the galloping mergeLo /
// mergeHi with their adaptive minGallop - because the sequence of compare(x, y)
// calls is observable: a selector or compareTo with side effects sees it.
// ComparableTimSort is the same algorithm over compareTo, so it is this one with
// cmp = compareTo.
//
// jtsSort(a, cmp) sorts the array a in place; cmp(x, y) answers a number whose
// sign orders x against y. It answers false when the comparator was found to
// violate its contract (java's IllegalArgumentException "Comparison method
// violates its general contract!"), leaving a as java leaves it at that point.

function jtsSort(a, cmp) {
    var n = a.length
    if (n < 2) { return true }
    if (n < 32) {
        jtsBinarySort(a, 0, n, jtsCountRun(a, 0, n, cmp), cmp)
        return true
    }
    var ts = {a: a, cmp: cmp, minGallop: 7, base: [], len: [], bad: false}
    var minRun = jtsMinRun(n)
    var lo = 0
    var rem = n
    while (rem > 0) {
        var r = jtsCountRun(a, lo, lo + rem, cmp)
        if (r < minRun) {
            var force = minRun
            if (rem <= minRun) { force = rem }
            jtsBinarySort(a, lo, lo + force, lo + r, cmp)
            r = force
        }
        ts.base.push(lo)
        ts.len.push(r)
        jtsMergeCollapse(ts)
        if (ts.bad) { return false }
        lo = lo + r
        rem = rem - r
    }
    while (ts.base.length > 1) {
        var k = ts.base.length - 2
        if (k > 0 && ts.len[k - 1] < ts.len[k + 1]) { k = k - 1 }
        jtsMergeAt(ts, k)
        if (ts.bad) { return false }
    }
    return true
}

function jtsMinRun(n0) {
    var n = n0
    var r = 0
    while (n >= 32) {
        if (n % 2 == 1) { r = 1 }
        n = Math.floor(n / 2)
    }
    return n + r
}

// System.arraycopy: n elements from src[sp] to dst[dp], correct when the two
// ranges overlap in one array.
function jtsCopy(src, sp, dst, dp, n) {
    var i = 0
    if (src === dst && dp > sp) {
        i = n - 1
        while (i >= 0) {
            dst[dp + i] = src[sp + i]
            i = i - 1
        }
        return
    }
    while (i < n) {
        dst[dp + i] = src[sp + i]
        i = i + 1
    }
}

function jtsCountRun(a, lo, hi, cmp) {
    var runHi = lo + 1
    if (runHi == hi) { return 1 }
    runHi = runHi + 1
    if (cmp(a[lo + 1], a[lo]) < 0) {
        while (runHi < hi && cmp(a[runHi], a[runHi - 1]) < 0) { runHi = runHi + 1 }
        var i = lo
        var j = runHi - 1
        while (i < j) {
            var t = anytype
            t = a[i]
            a[i] = a[j]
            a[j] = t
            i = i + 1
            j = j - 1
        }
    } else {
        while (runHi < hi && cmp(a[runHi], a[runHi - 1]) >= 0) { runHi = runHi + 1 }
    }
    return runHi - lo
}

function jtsBinarySort(a, lo, hi, start0, cmp) {
    var start = start0
    if (start == lo) { start = start + 1 }
    while (start < hi) {
        var pivot = anytype
        pivot = a[start]
        var left = lo
        var right = start
        while (left < right) {
            var mid = Math.floor((left + right) / 2)
            if (cmp(pivot, a[mid]) < 0) { right = mid } else { left = mid + 1 }
        }
        jtsCopy(a, left, a, left + 1, start - left)
        a[left] = pivot
        start = start + 1
    }
}

function jtsMergeCollapse(ts) {
    var rl = ts.len
    while (ts.base.length > 1) {
        var n = ts.base.length - 2
        if ((n > 0 && rl[n - 1] <= rl[n] + rl[n + 1]) || (n > 1 && rl[n - 2] <= rl[n] + rl[n - 1])) {
            if (rl[n - 1] < rl[n + 1]) { n = n - 1 }
        } else if (rl[n] > rl[n + 1]) {
            return
        }
        jtsMergeAt(ts, n)
        if (ts.bad) { return }
    }
}

function jtsMergeAt(ts, i) {
    var a = ts.a
    var base1 = ts.base[i]
    var len1 = ts.len[i]
    var base2 = ts.base[i + 1]
    var len2 = ts.len[i + 1]
    ts.len[i] = len1 + len2
    if (i == ts.base.length - 3) {
        ts.base[i + 1] = ts.base[i + 2]
        ts.len[i + 1] = ts.len[i + 2]
    }
    ts.base.pop()
    ts.len.pop()
    var k = jtsGallopRight(a[base2], a, base1, len1, 0, ts.cmp)
    base1 = base1 + k
    len1 = len1 - k
    if (len1 == 0) { return }
    len2 = jtsGallopLeft(a[base1 + len1 - 1], a, base2, len2, len2 - 1, ts.cmp)
    if (len2 == 0) { return }
    if (len1 <= len2) { jtsMergeLo(ts, base1, len1, base2, len2) } else { jtsMergeHi(ts, base1, len1, base2, len2) }
}

// Where key goes in a[base:base+len], left of any equal element, searched from
// base+hint.
function jtsGallopLeft(key, a, base, len, hint, cmp) {
    var lastOfs = 0
    var ofs = 1
    var maxOfs = 0
    if (cmp(key, a[base + hint]) > 0) {
        maxOfs = len - hint
        while (ofs < maxOfs && cmp(key, a[base + hint + ofs]) > 0) {
            lastOfs = ofs
            ofs = ofs * 2 + 1
        }
        if (ofs > maxOfs) { ofs = maxOfs }
        lastOfs = lastOfs + hint
        ofs = ofs + hint
    } else {
        maxOfs = hint + 1
        while (ofs < maxOfs && cmp(key, a[base + hint - ofs]) <= 0) {
            lastOfs = ofs
            ofs = ofs * 2 + 1
        }
        if (ofs > maxOfs) { ofs = maxOfs }
        var t = lastOfs
        lastOfs = hint - ofs
        ofs = hint - t
    }
    lastOfs = lastOfs + 1
    while (lastOfs < ofs) {
        var m = lastOfs + Math.floor((ofs - lastOfs) / 2)
        if (cmp(key, a[base + m]) > 0) { lastOfs = m + 1 } else { ofs = m }
    }
    return ofs
}

// Where key goes in a[base:base+len], right of any equal element.
function jtsGallopRight(key, a, base, len, hint, cmp) {
    var ofs = 1
    var lastOfs = 0
    var maxOfs = 0
    if (cmp(key, a[base + hint]) < 0) {
        maxOfs = hint + 1
        while (ofs < maxOfs && cmp(key, a[base + hint - ofs]) < 0) {
            lastOfs = ofs
            ofs = ofs * 2 + 1
        }
        if (ofs > maxOfs) { ofs = maxOfs }
        var t = lastOfs
        lastOfs = hint - ofs
        ofs = hint - t
    } else {
        maxOfs = len - hint
        while (ofs < maxOfs && cmp(key, a[base + hint + ofs]) >= 0) {
            lastOfs = ofs
            ofs = ofs * 2 + 1
        }
        if (ofs > maxOfs) { ofs = maxOfs }
        lastOfs = lastOfs + hint
        ofs = ofs + hint
    }
    lastOfs = lastOfs + 1
    while (lastOfs < ofs) {
        var m = lastOfs + Math.floor((ofs - lastOfs) / 2)
        if (cmp(key, a[base + m]) < 0) { ofs = m } else { lastOfs = m + 1 }
    }
    return ofs
}

// Merges the run at base1 into the following one, the first run being the
// shorter: it is copied out to tmp and the merge fills a from the left.
function jtsMergeLo(ts, base1, len1In, base2, len2In) {
    var a = ts.a
    var cmp = ts.cmp
    var len1 = len1In
    var len2 = len2In
    var tmp = []
    jtsCopy(a, base1, tmp, 0, len1)
    var cursor1 = 0
    var cursor2 = base2
    var dest = base1
    a[dest] = a[cursor2]
    dest = dest + 1
    cursor2 = cursor2 + 1
    len2 = len2 - 1
    if (len2 == 0) {
        jtsCopy(tmp, cursor1, a, dest, len1)
        return
    }
    if (len1 == 1) {
        jtsCopy(a, cursor2, a, dest, len2)
        a[dest + len2] = tmp[cursor1]
        return
    }
    var minGallop = ts.minGallop
    var done = false
    while (!done) {
        var count1 = 0
        var count2 = 0
        while (true) {
            if (cmp(a[cursor2], tmp[cursor1]) < 0) {
                a[dest] = a[cursor2]
                dest = dest + 1
                cursor2 = cursor2 + 1
                count2 = count2 + 1
                count1 = 0
                len2 = len2 - 1
                if (len2 == 0) { done = true; break }
            } else {
                a[dest] = tmp[cursor1]
                dest = dest + 1
                cursor1 = cursor1 + 1
                count1 = count1 + 1
                count2 = 0
                len1 = len1 - 1
                if (len1 == 1) { done = true; break }
            }
            if ((count1 | count2) >= minGallop) { break }
        }
        if (done) { break }
        while (true) {
            count1 = jtsGallopRight(a[cursor2], tmp, cursor1, len1, 0, cmp)
            if (count1 != 0) {
                jtsCopy(tmp, cursor1, a, dest, count1)
                dest = dest + count1
                cursor1 = cursor1 + count1
                len1 = len1 - count1
                if (len1 <= 1) { done = true; break }
            }
            a[dest] = a[cursor2]
            dest = dest + 1
            cursor2 = cursor2 + 1
            len2 = len2 - 1
            if (len2 == 0) { done = true; break }
            count2 = jtsGallopLeft(tmp[cursor1], a, cursor2, len2, 0, cmp)
            if (count2 != 0) {
                jtsCopy(a, cursor2, a, dest, count2)
                dest = dest + count2
                cursor2 = cursor2 + count2
                len2 = len2 - count2
                if (len2 == 0) { done = true; break }
            }
            a[dest] = tmp[cursor1]
            dest = dest + 1
            cursor1 = cursor1 + 1
            len1 = len1 - 1
            if (len1 == 1) { done = true; break }
            minGallop = minGallop - 1
            if (count1 < 7 && count2 < 7) { break }
        }
        if (done) { break }
        if (minGallop < 0) { minGallop = 0 }
        minGallop = minGallop + 2
    }
    if (minGallop < 1) { minGallop = 1 }
    ts.minGallop = minGallop
    if (len1 == 1) {
        jtsCopy(a, cursor2, a, dest, len2)
        a[dest + len2] = tmp[cursor1]
    } else if (len1 == 0) {
        ts.bad = true
    } else {
        jtsCopy(tmp, cursor1, a, dest, len1)
    }
}

// The mirror image: the second run is the shorter, copied out to tmp, and the
// merge fills a from the right.
function jtsMergeHi(ts, base1, len1In, base2, len2In) {
    var a = ts.a
    var cmp = ts.cmp
    var len1 = len1In
    var len2 = len2In
    var tmp = []
    jtsCopy(a, base2, tmp, 0, len2)
    var cursor1 = base1 + len1 - 1
    var cursor2 = len2 - 1
    var dest = base2 + len2 - 1
    a[dest] = a[cursor1]
    dest = dest - 1
    cursor1 = cursor1 - 1
    len1 = len1 - 1
    if (len1 == 0) {
        jtsCopy(tmp, 0, a, dest - (len2 - 1), len2)
        return
    }
    if (len2 == 1) {
        dest = dest - len1
        cursor1 = cursor1 - len1
        jtsCopy(a, cursor1 + 1, a, dest + 1, len1)
        a[dest] = tmp[cursor2]
        return
    }
    var minGallop = ts.minGallop
    var done = false
    while (!done) {
        var count1 = 0
        var count2 = 0
        while (true) {
            if (cmp(tmp[cursor2], a[cursor1]) < 0) {
                a[dest] = a[cursor1]
                dest = dest - 1
                cursor1 = cursor1 - 1
                count1 = count1 + 1
                count2 = 0
                len1 = len1 - 1
                if (len1 == 0) { done = true; break }
            } else {
                a[dest] = tmp[cursor2]
                dest = dest - 1
                cursor2 = cursor2 - 1
                count2 = count2 + 1
                count1 = 0
                len2 = len2 - 1
                if (len2 == 1) { done = true; break }
            }
            if ((count1 | count2) >= minGallop) { break }
        }
        if (done) { break }
        while (true) {
            count1 = len1 - jtsGallopRight(tmp[cursor2], a, base1, len1, len1 - 1, cmp)
            if (count1 != 0) {
                dest = dest - count1
                cursor1 = cursor1 - count1
                len1 = len1 - count1
                jtsCopy(a, cursor1 + 1, a, dest + 1, count1)
                if (len1 == 0) { done = true; break }
            }
            a[dest] = tmp[cursor2]
            dest = dest - 1
            cursor2 = cursor2 - 1
            len2 = len2 - 1
            if (len2 == 1) { done = true; break }
            count2 = len2 - jtsGallopLeft(a[cursor1], tmp, 0, len2, len2 - 1, cmp)
            if (count2 != 0) {
                dest = dest - count2
                cursor2 = cursor2 - count2
                len2 = len2 - count2
                jtsCopy(tmp, cursor2 + 1, a, dest + 1, count2)
                if (len2 <= 1) { done = true; break }
            }
            a[dest] = a[cursor1]
            dest = dest - 1
            cursor1 = cursor1 - 1
            len1 = len1 - 1
            if (len1 == 0) { done = true; break }
            minGallop = minGallop - 1
            if (count1 < 7 && count2 < 7) { break }
        }
        if (done) { break }
        if (minGallop < 0) { minGallop = 0 }
        minGallop = minGallop + 2
    }
    if (minGallop < 1) { minGallop = 1 }
    ts.minGallop = minGallop
    if (len2 == 1) {
        dest = dest - len1
        cursor1 = cursor1 - len1
        jtsCopy(a, cursor1 + 1, a, dest + 1, len1)
        a[dest] = tmp[cursor2]
    } else if (len2 == 0) {
        ts.bad = true
    } else {
        jtsCopy(tmp, 0, a, dest - (len2 - 1), len2)
    }
}
