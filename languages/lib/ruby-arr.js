// ruby-arr.js -- Array index arithmetic of MRI's array.c, for ruby-interpreter.abnf
// (include) and languages/lib/ruby-rt.metajs (import). Go twins in
// abnf/jsrtrubystr.go. Written in the subset both engines accept.

// Enumerator::Lazy, and the endless Enumerator of a blockless Array#cycle or of
// each_slice / each_cons on an endless range: an Enumerator object that also
// carries __rlazy, a SOURCE - kind "arr" (items), "cycle" (items, forever) or
// "range" (lo, lo+1, ... up to hi; hi null is endless) or "each" (the #each of
// a user Enumerable or of a Range that cannot be walked by counting, src; a
// non-null hi is then its size, a Float range's) - and the OPS {k, f, n},
// applied to one value at a time: map / select / reject / take_while /
// drop_while / flat_map / chunk_while
// / slice_when (f the block), take / drop / with_index (n the count or offset),
// with_object (f the memo),
// step (every n-th value from the first, a String range's #step),
// slice / cons (n the width), uniq (f the key block or null) and zip (f a list
// of functions answering an argument's i-th value or nil).
// rbaLazyFirst pulls the first n results (all of them when n < 0); call(f, v, m)
// runs a block, test(v) is Ruby truthiness, eql(a, b) is #eql?. A take op ends
// the run once its count has passed it, so nothing past it is evaluated. An
// "each" source is PUSHED: drive(src, step) runs its #each, calls step(v, m) per
// value, leaves #each once step answers true and answers what #each answered,
// which a run to the end records as the Lazy's ev. A multi-value yield arrives
// packed as an Array with m true; the block of a map, take_while or drop_while
// gets its values spread (`yield 1, 2' gives `map { |x| }' 1) and that of a
// select or reject the Array (MRI's lazy.c), until a map or a slice makes one
// value of it. With emit non-null each result goes to emit(v, m) instead of the
// answer, and the run ends once emit answers true. Go twin: rubyLazyPull in
// abnf/jsrtrubystr.go.
function rbaLazyFirst(lz, n, call, test, eql, drive, emit) {
    var ops = lz.ops
    var st = {out: [], n: n, cnt: [], on: [], buf: [], done: n == 0, full: false, stopAt: 0 - 1, emit: emit}
    var j = 0
    while (j < ops.length) {
        st.cnt.push(0)
        st.on.push(false)
        st.buf.push([])
        // take(0) makes a Lazy over nothing (lazy_take), so no value is pulled.
        if (ops[j].k == "take" && ops[j].n == 0) { st.done = true }
        j = j + 1
    }
    if (lz.kind == "each" && !st.done) {
        var ev = anytype
        ev = drive(lz.src, function(v, m) {
            rbaLazyRun(st, ops, v, m, 0, call, test, eql)
            return st.done
        })
        if (!st.done) { lz.ev = ev }
    }
    var i = 0
    while (lz.kind != "each" && !st.done) {
        var v = anytype
        if (lz.kind == "range") {
            v = lz.lo + i
            if (lz.hi !== null && (lz.excl ? v >= lz.hi : v > lz.hi)) { break }
        } else {
            var len = lz.items.length
            if (len == 0 || (lz.kind == "arr" && i >= len)) { break }
            v = lz.items[i % len]
        }
        i = i + 1
        rbaLazyRun(st, ops, v, false, 0, call, test, eql)
    }
    // A slice or chunk op still holding values hands them on as a last, short
    // one when the run ended above it: the source ran out, or a take / take_while
    // before it stopped.
    j = 0
    while (j < ops.length && !st.full) {
        if (rbaLazyBuffers(ops[j].k) && st.buf[j].length > 0 && j > st.stopAt) {
            var rest = st.buf[j]
            st.buf[j] = []
            rbaLazyRun(st, ops, rest, false, j + 1, call, test, eql)
        }
        j = j + 1
    }
    return st.out
}
// One value (m: a packed multi-value yield) through ops[from..], into the
// result unless an op holds it back.
function rbaLazyRun(st, ops, v, m, from, call, test, eql) {
    var cur = anytype
    cur = v
    var multi = m
    var j = from
    while (j < ops.length) {
        var op = ops[j]
        if (op.k == "map") {
            cur = call(op.f, cur, multi)
            multi = false
        } else if (op.k == "select") {
            if (!test(call(op.f, cur, false))) { return }
        } else if (op.k == "reject") {
            if (test(call(op.f, cur, false))) { return }
        } else if (op.k == "take_while") {
            if (!test(call(op.f, cur, multi))) {
                rbaLazyStop(st, j)
                return
            }
        } else if (op.k == "drop_while") {
            if (!st.on[j]) {
                if (test(call(op.f, cur, multi))) { return }
                st.on[j] = true
            }
        } else if (op.k == "take") {
            // A count already reached also turns away the rest of a flat_map's
            // Array.
            if (st.cnt[j] >= op.n) {
                rbaLazyStop(st, j)
                return
            }
            st.cnt[j] = st.cnt[j] + 1
            if (st.cnt[j] >= op.n) { rbaLazyStop(st, j) }
        } else if (op.k == "slice") {
            st.buf[j].push(cur)
            if (st.buf[j].length < op.n) { return }
            cur = st.buf[j]
            st.buf[j] = []
            multi = false
        } else if (op.k == "cons") {
            st.buf[j].push(cur)
            if (st.buf[j].length > op.n) { st.buf[j] = st.buf[j].slice(1) }
            if (st.buf[j].length < op.n) { return }
            cur = st.buf[j].slice(0)
            multi = false
        } else if (op.k == "flat_map") {
            // An Array result is spliced in, one value per element; any other
            // passes on as one value.
            var fr = anytype
            fr = call(op.f, cur, multi)
            if (Array.isArray(fr)) {
                var q = 0
                while (q < fr.length && !st.full) {
                    rbaLazyRun(st, ops, fr[q], false, j + 1, call, test, eql)
                    q = q + 1
                }
                return
            }
            cur = fr
            multi = false
        } else if (op.k == "drop") {
            if (st.cnt[j] < op.n) {
                st.cnt[j] = st.cnt[j] + 1
                return
            }
        } else if (op.k == "step") {
            var sk = st.cnt[j]
            st.cnt[j] = sk + 1
            if (sk % op.n != 0) { return }
        } else if (op.k == "uniq") {
            var key = anytype
            key = op.f === null ? cur : call(op.f, cur, multi)
            var u = 0
            while (u < st.buf[j].length) {
                if (eql(st.buf[j][u], key)) { return }
                u = u + 1
            }
            st.buf[j].push(key)
        } else if (op.k == "zip") {
            // A multi-value yield zips its first value.
            var row = [multi ? cur[0] : cur]
            var z = 0
            while (z < op.f.length) {
                row.push(call(op.f[z], st.cnt[j], false))
                z = z + 1
            }
            st.cnt[j] = st.cnt[j] + 1
            cur = row
            multi = false
        } else if (op.k == "with_index") {
            cur = [cur, st.cnt[j] + op.n]
            st.cnt[j] = st.cnt[j] + 1
            multi = false
        } else if (op.k == "with_object") {
            cur = [cur, op.f]
            multi = false
        } else if (op.k == "chunk_while" || op.k == "slice_when") {
            // A chunk goes on once the block, given its last value and this one,
            // splits them: answers false for chunk_while, true for slice_when.
            var cb = st.buf[j]
            if (cb.length > 0 && test(call(op.f, [cb[cb.length - 1], cur], true)) == (op.k == "slice_when")) {
                st.buf[j] = [cur]
                cur = cb
                multi = false
            } else {
                cb.push(cur)
                return
            }
        }
        j = j + 1
    }
    if (st.emit !== null) {
        if (st.emit(cur, multi) === true) {
            st.done = true
            st.full = true
        }
        return
    }
    st.out.push(cur)
    if (st.n >= 0 && st.out.length >= st.n) {
        st.done = true
        st.full = true
    }
}
function rbaLazyStop(st, j) {
    st.done = true
    if (st.stopAt < 0) { st.stopAt = j }
}

// Does op k hold values back in st.buf until a later value releases them?
function rbaLazyBuffers(k) { return k == "slice" || k == "chunk_while" || k == "slice_when" }

// The op a Lazy method WITH A BLOCK appends, or "" when the method is not one.
function rbaLazyOp(name) {
    if (name == "map" || name == "collect") { return "map" }
    if (name == "select" || name == "filter" || name == "find_all") { return "select" }
    if (name == "flat_map" || name == "collect_concat") { return "flat_map" }
    if (name == "reject" || name == "take_while" || name == "drop_while" || name == "uniq") { return name }
    if (name == "chunk_while" || name == "slice_when") { return name }
    return ""
}

// The ops whose blockless call on an eager stream passes its values through
// unchanged (Enumerable's RETURN_SIZED_ENUMERATOR). Go twin: rubyPassOp in
// abnf/jsrtrubystr.go.
function rbaPassOp(op) {
    return op == "map" || op == "select" || op == "reject" || op == "flat_map"
}

// The #size of a Lazy or stream Enumerator (lazy_receiver_size and each op's
// size function): the source's count - an Array's length, a range's, inf for
// an endless range or a cycle of a non-empty Array, unknown for a user #each -
// carried through the ops. map, with_index / with_object and zip keep it,
// take(n) caps it, drop(n) and each_cons(n) lower it, each_slice(n) is
// (size + n - 1).div(n), and any other op (a Lazy's select, take_while,
// flat_map, uniq ...) makes it unknown, as does the method rm of a blockless
// find / take_while. -1 for unknown (nil), -2 for an each_slice dividing
// Infinity, which raises FloatDomainError in MRI 2.6. Go twin: rubyLazyCount
// in abnf/jsrtrubystr.go.
function rbaLazyCount(lz, inf) {
    if (lz.rm != "") { return 0 - 1 }
    var c = 0 - 1
    if (lz.kind == "arr") { c = lz.items.length }
    if (lz.kind == "cycle") { c = lz.items.length > 0 ? inf : 0 }
    if (lz.kind == "range") {
        if (lz.hi === null) { c = inf } else { c = Math.max(0, lz.hi - lz.lo + (lz.excl ? 0 : 1)) }
    }
    if (lz.kind == "each" && lz.hi !== null) { c = lz.hi }
    var i = 0
    while (i < lz.ops.length && c >= 0) {
        var k = lz.ops[i].k
        var n = lz.ops[i].n
        if (k == "take") { c = Math.min(c, n) }
        else if (k == "drop") { c = Math.max(0, c - n) }
        else if (k == "cons") { c = Math.max(0, c - n + 1) }
        else if (k == "slice") {
            if (c == inf) { return 0 - 2 }
            c = Math.floor((c + n - 1) / n)
        } else if (k != "map" && k != "with_index" && k != "with_object" && k != "zip") { c = 0 - 1 }
        i = i + 1
    }
    return c
}

// The Enumerable methods whose blockless Enumerator has no size and whose #each
// with a block runs the method itself (RETURN_ENUMERATOR, not the sized form).
// Go twin: rubyUnsizedOp in abnf/jsrtrubystr.go.
function rbaUnsizedOp(name) {
    return name == "find" || name == "detect" || name == "find_index" || name == "take_while" || name == "drop_while"
}

// The unsized ops that stop yielding at the first falsy block answer. Every
// Enumerable and Enumerator method iterates with a block answering nil, so it
// sees only the first value (`[1, 2].take_while.first(2)` is [1]); only #each and
// the rbaRerunsMethod ones given a block hand the method the caller's answers.
// Go twin: rubyStopsOnNil in abnf/jsrtrubystr.go.
function rbaStopsOnNil(op) { return op == "take_while" || op == "drop_while" }

// The Enumerable methods a Lazy answers EAGERLY, over the Array of all its
// values: MRI's Lazy inherits them from Enumerable unchanged. The ones that can
// stop early (find, include?, any?, ...) are answered by rbaLazyScan instead.
function rbaLazyEager(name) {
    return name == "sum" || name == "count" || name == "min" || name == "max" || name == "minmax"
        || name == "min_by" || name == "max_by" || name == "minmax_by" || name == "sort" || name == "sort_by"
        || name == "reduce" || name == "inject" || name == "group_by" || name == "partition" || name == "to_h"
        || name == "one?"
}

// The rbaLazyEager methods that call their block once per value, in order, and
// nothing else: they are run one value at a time, the block's answers kept and
// handed back to the Array method in the same order, so a `break` in the block
// ends an endless source as it does in MRI. Go twin: rubyLazyKeyed in
// abnf/jsrtrubystr.go.
function rbaLazyKeyed(name) {
    return name == "count" || name == "sum" || name == "min_by" || name == "max_by" || name == "minmax_by"
        || name == "sort_by" || name == "group_by" || name == "partition" || name == "to_h"
}

// find / detect / include? / member? / find_index / any? / all? / none? on a
// Lazy: the run ends at the first value hit(v, m) accepts, so an endless source
// still answers. [index, value] of that value, or [] when none was accepted.
function rbaLazyScan(lz, call, test, eql, drive, hit) {
    var res = []
    var i = [0]
    rbaLazyFirst(lz, -1, call, test, eql, drive, function(v, m) {
        if (test(hit(v, m))) {
            res.push(i[0])
            res.push(v)
            return true
        }
        i[0] = i[0] + 1
        return false
    })
    return res
}

// The Enumerator methods that run the method the Enumerator was made from
// again rather than its #each (enumerator.c). Go twin: rubyRerunsMethod in
// abnf/jsrtrubystr.go.
function rbaRerunsMethod(name) {
    return name == "with_index" || name == "each_with_index" || name == "with_object" || name == "each_with_object"
}

// The op a blockless each_slice / each_cons is on an Enumerator that pulls one
// value at a time, or "". Go twin: rubySliceOp in abnf/jsrtrubystr.go.
function rbaSliceOp(name) {
    if (name == "each_slice") { return "slice" }
    if (name == "each_cons") { return "cons" }
    return ""
}

// The block methods of an Integer range without an end that MRI answers one
// element at a time, so a break, a hit or a take_while stop can end them: they
// run on the Enumerator #lazy walks. Go twin: rubyRangeStreams in
// abnf/jsrtrubystr.go.
function rbaRangeStreams(name) {
    return name == "each" || name == "each_with_index" || name == "each_with_object" || name == "find"
        || name == "detect" || name == "find_index" || name == "any?" || name == "all?" || name == "none?"
        || (rbaLazyOp(name) != "" && name != "drop_while" && name != "uniq" && name != "chunk_while" && name != "slice_when")
}

// Array#insert's splice position on an array of n (rb_ary_insert): -1 is the end
// and another negative one counts from after the last element. -1 when it lies
// before the start, which is IndexError "index POS too small for array; minimum:
// -n-1".
function rbaInsertPos(n, pos) {
    if (pos >= 0) { return pos }
    if (pos < 0 - n - 1) { return -1 }
    return pos + n + 1
}

// t with items spliced in at pos, padded with nil when pos is past the end.
function rbaSpliced(t, pos, items) {
    var out = []
    var i = 0
    while (i < pos) {
        out.push(i < t.length ? t[i] : null)
        i = i + 1
    }
    var k = 0
    while (k < items.length) {
        out.push(items[k])
        k = k + 1
    }
    while (i < t.length) {
        out.push(t[i])
        i = i + 1
    }
    return out
}

// The length of the first of rows t that is not as long as the first row, or
// -1: Array#transpose's IndexError "element size differs (N should be W)".
function rbaRowMismatch(t) {
    var i = 1
    while (i < t.length) {
        if (t[i].length != t[0].length) { return t[i].length }
        i = i + 1
    }
    return -1
}
// Array#transpose of rows that all have one length.
function rbaTranspose(t) {
    var out = []
    if (t.length == 0) { return out }
    var w = t[0].length
    var c = 0
    while (c < w) {
        var col = []
        var r = 0
        while (r < t.length) {
            col.push(t[r][c])
            r = r + 1
        }
        out.push(col)
        c = c + 1
    }
    return out
}

// Array#chunk_while / #slice_when, given whether each adjacent pair stays
// together (keep[i] for t[i], t[i + 1]).
function rbaChunks(t, keep) {
    var out = []
    if (t.length == 0) { return out }
    var cur = [t[0]]
    var i = 1
    while (i < t.length) {
        if (!keep[i - 1]) {
            out.push(cur)
            cur = []
        }
        cur.push(t[i])
        i = i + 1
    }
    out.push(cur)
    return out
}

// The Enumerable methods whose block MRI 2.6 hands a multi-value `yield a, b` of
// #each as the values themselves rather than packed into one Array: `map { |x| }`
// sees a, `select { |x| }` sees [a, b]. Measured against /usr/bin/ruby per method.
// Go twin: rubyEnumSpreads in abnf/jsrtrubystr.go.
function rbaEnumSpreads(n) {
    return n == "map" || n == "collect" || n == "flat_map" || n == "collect_concat"
        || n == "find_index" || n == "all?" || n == "any?" || n == "none?" || n == "one?"
        || n == "count" || n == "take_while" || n == "uniq" || n == "to_h"
}

// The Enumerable methods MRI ends #each for once the elements so far decide the
// answer: 1 at the first element whose test holds (any? none? find detect
// find_index include? member?), 2 at the first whose test fails (all?
// take_while), 3 at the second that holds (one?), 4 after a count (first take),
// 0 never. find / detect / take_while without a block, and find_index without
// one or a value, answer an Enumerator and iterate nothing. Go twin:
// rubyEnumStop in abnf/jsrtrubystr.go.
function rbaEnumStop(n, hasBlk, nargs) {
    if (n == "first" || n == "take") { return 4 }
    if (n == "any?" || n == "none?" || n == "include?" || n == "member?") { return 1 }
    if (n == "all?") { return 2 }
    if (n == "one?") { return 3 }
    if (!hasBlk && (n != "find_index" || nargs == 0)) { return 0 }
    if (n == "find" || n == "detect" || n == "find_index") { return 1 }
    if (n == "take_while") { return 2 }
    return 0
}
// The Array method that is such a method's test of one element without a
// block: the pattern === (or truthiness) of any?, or the == of include?.
// Go twin: rubyEnumProbe in abnf/jsrtrubystr.go.
function rbaEnumProbe(n) {
    if (n == "any?" || n == "all?" || n == "none?" || n == "one?") { return "any?" }
    return "include?"
}
// Does the Enumerable method n, given nargs arguments and no block, answer an
// Enumerator? Go twin: rubyEnumDefers in abnf/jsrtruby.go.
function rbaEnumDefers(n, nargs) {
    if (n == "each_slice" || n == "each_cons" || n == "each_with_object") { return nargs == 1 }
    if (n == "cycle") { return nargs <= 1 }
    if (nargs != 0) { return false }
    return n == "map" || n == "collect" || n == "flat_map" || n == "collect_concat" || n == "select"
        || n == "filter" || n == "find_all" || n == "reject" || n == "find" || n == "detect"
        || n == "find_index" || n == "group_by" || n == "partition" || n == "min_by" || n == "max_by"
        || n == "minmax_by" || n == "sort_by" || n == "take_while" || n == "drop_while"
        || n == "each_with_index" || n == "each_entry" || n == "reverse_each"
}
// For `for` over an Enumerator (forEnumItems in ruby-interpreter.abnf): does the
// method m always yield two values (the element and an index or memo)? And does
// it, made without a block over a user #each, yield what that #each yields? Go
// twins: rubyForPairMeth / rubyForPassMeth in abnf/jsrtruby.go.
function rbaForPairMeth(m) {
    return m == "each_with_index" || m == "each_with_object" || m == "with_index" || m == "with_object"
}
function rbaForPassMeth(m) {
    return m == "each" || m == "map" || m == "collect" || m == "flat_map" || m == "collect_concat"
}

// One run of the Enumerable method m that rbaEnumDefers names, made without a
// block over a user #each, handing on each block call's arguments as MRI's
// method makes it - what Enumerator#next answers one at a time, and what a `for`
// over the Enumerator binds. drive(step) runs #each, calls step(it, multi) per
// yield (a multi-value one packed as an Array) and leaves #each once step answers
// true; out(a) takes one call's argument list. n is each_slice / each_cons's
// width or cycle's count (-1: for ever), memo each_with_object's. The block
// answers nil, so take_while stops after one element and drop_while hands on
// only its first; reverse_each and cycle's later passes need the whole run.
// Answers the packed values #each yielded. Go twin: rubyEnumPump in
// abnf/jsrtruby.go.
function rbaEnumPump(m, n, memo, drive, out) {
    var st = {items: [], buf: [], i: 0, dropped: false}
    if (m == "cycle" && n == 0) { return st.items }
    drive(function(it, multi) {
        st.items.push(it)
        if (m == "each_with_index") {
            out([it, st.i])
            st.i = st.i + 1
        } else if (m == "each_with_object") {
            out([it, memo])
        } else if (m == "each_slice" || m == "each_cons") {
            st.buf.push(it)
            if (m == "each_cons" && st.buf.length > n) {
                var w = []
                var k = 1
                while (k < st.buf.length) { w.push(st.buf[k]); k = k + 1 }
                st.buf = w
            }
            if (st.buf.length == n) {
                if (m == "each_slice") {
                    out([st.buf])
                    st.buf = []
                } else {
                    var c = []
                    var j = 0
                    while (j < n) { c.push(st.buf[j]); j = j + 1 }
                    out([c])
                }
            }
        } else if (m != "reverse_each" && !(m == "drop_while" && st.dropped)) {
            st.dropped = true
            if (multi && rbaForPassMeth(m)) { out(it) }
            else { out([it]) }
            return m == "take_while"
        }
        return false
    })
    if (m == "each_slice" && st.buf.length > 0) { out([st.buf]) }
    var r = st.items.length - 1
    while (m == "reverse_each" && r >= 0) {
        out([st.items[r]])
        r = r - 1
    }
    var pass = 1
    while (m == "cycle" && st.items.length > 0 && (n < 0 || pass < n)) {
        var q = 0
        while (q < st.items.length) {
            out([st.items[q]])
            q = q + 1
        }
        pass = pass + 1
    }
    return st.items
}

// Enumerable#slice_before / #slice_after, given whether each element matched:
// a match opens a new slice before it, or closes the current one after it.
function rbaSliceAt(t, hit, after) {
    var out = []
    var cur = []
    var i = 0
    while (i < t.length) {
        if (!after && hit[i] && cur.length > 0) {
            out.push(cur)
            cur = []
        }
        cur.push(t[i])
        if (after && hit[i]) {
            out.push(cur)
            cur = []
        }
        i = i + 1
    }
    if (cur.length > 0) { out.push(cur) }
    return out
}

// Enumerable#chunk, given each element's block value and its kind: 0 an ordinary
// key, 1 dropped (nil or :_separator), 2 a chunk of its own (:_alone). Runs of
// ordinary keys that eq calls equal share one [key, [elements]] chunk; a dropped
// element ends the run.
function rbaChunk(t, keys, kinds, eq) {
    var out = []
    var cur = anytype
    cur = null
    var i = 0
    while (i < t.length) {
        if (kinds[i] == 1) {
            cur = null
        } else if (kinds[i] == 2) {
            out.push([keys[i], [t[i]]])
            cur = null
        } else if (cur !== null && eq(cur[0], keys[i])) {
            cur[1].push(t[i])
        } else {
            cur = [keys[i], [t[i]]]
            out.push(cur)
        }
        i = i + 1
    }
    return out
}

// Enumerable#minmax_by over [key, element] pairs: the first element of the least
// key and the first of the greatest, as cmp orders keys.
function rbaMinMaxBy(kp, cmp) {
    if (kp.length == 0) { return [null, null] }
    var lo = 0
    var hi = 0
    var i = 1
    while (i < kp.length) {
        if (cmp(kp[i][0], kp[lo][0]) < 0) { lo = i }
        if (cmp(kp[i][0], kp[hi][0]) > 0) { hi = i }
        i = i + 1
    }
    return [kp[lo][1], kp[hi][1]]
}

// The [begin, end) Array#fill writes on an array of n (rb_ary_fill): a negative
// start counts from the end (clamped to 0), an absent one is 0, an absent length
// runs to the end; null when the length is negative. end may lie past n, which
// grows the array.
function rbaFillSpan(n, beg, hasBeg, len, hasLen) {
    var b = hasBeg ? beg : 0
    if (b < 0) {
        b = b + n
        if (b < 0) { b = 0 }
    }
    var l = hasLen ? len : n - b
    if (l < 0) { return null }
    return {b: b, e: b + l}
}

// The address Object#to_s and #inspect print for an object whose object_id is
// id: MRI 2.6's is twice the object_id, in sixteen hex digits, so ours is twice
// ours ("0x0000000000000020" for id 16). Go twin: rubyObjAddr in
// abnf/jsrtruby.go.
function rbaObjAddr(id) {
    var n = id * 2
    var out = ""
    var i = 0
    while (i < 16) {
        var q = Math.trunc(n / 16)
        out = "0123456789abcdef".charAt(n - q * 16) + out
        n = q
        i = i + 1
    }
    return "0x" + out
}
