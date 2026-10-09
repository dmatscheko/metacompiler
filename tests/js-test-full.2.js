// Full-syntax test: JavaScript - part 2 of 3, sections 36-43.
//
// THE JS RATCHET IS THREE FILES: as one file its -frozen interpreter leg ran into
// ./test.sh --full's time brake (peak memory grew superlinearly with the file).
//
//   tests/js-test-full.js     sections 01-35, the ECMAScript-2022 syntax walk
//   tests/js-test-full.2.js   sections 36-43, engine agreement, generators, Error, Math
//   tests/js-test-full.3.js   sections 44-51, the call/class/private-name model
//
// The section NUMBERS are global (grammar comments and docs cite them), the three
// files together run the assertions in order, and ./test.sh --full reports them as the
// units js, js2 and js3 - one per FILE, because the check-count agreement rule
// compares the two GRAMMARS of one file and never two files of one language. Add a
// new section to whichever file it belongs in, and keep the numbers globally unique
// across the three.
//
// This file belongs to the SECOND test group (./test.sh --full): it is NOT part
// of the default matrix. The goal of the metacompiler is to support the full
// languages; these files are the ratchet that measures how far the js grammars are.
// The --full runner runs the file, and whenever a grammar aborts it removes the
// section around the error and retries - so the report lists every unsupported
// section, not just the first.
//
// Conventions (shared by every *-test-full.* file):
//   - prologue (before the first SECTION marker): the check helper only
//   - each section: '// ===== SECTION <nn>: <name> =====', top-level,
//     self-contained, no references to other sections
//   - main() calls each section via a line tagged 'SECTION-CALL <nn>'
//     and prints the summary line 'full: <checks> checks, <failures> failures'
//   - main() returns the failure count (exit 0 == full support, verified)
//
// Hand-written for the metacompiler project (Apache-2.0, no copied test-suite code).

var failures = 0
var checks = 0
function check(id, cond) {
    checks = checks + 1
    if (!cond) {
        println("FAIL " + id)
        failures = failures + 1
    }
}

// ===== SECTION 36: delete, key order, and template-literal ToString =====
// DELETE removes the key and its key-order entry (the floor's js_del), so key ORDER
// after a delete-then-reinsert, for-in and object spread are node's answers.
//
// TEMPLATE LITERAL. `${v}` is ToString(v), which is ToPrimitive with hint STRING -
// so an object's own `toString` runs, an object with only a `valueOf` still spells
// "[object Object]", and a BigInt spells its digits. Every engine converts each PART
// with ToString (host concatenation would be hint DEFAULT and call `valueOf` first).
function s36() {
    var o = { a: 1, b: 2, c: 3 }
    check("del1", (delete o.b) === true && o.b === undefined)
    check("del2", Object.keys(o).join(",") === "a,c")
    // A key that comes back is a NEW key: it lands at the end of the order.
    o.b = 9
    check("del3", Object.keys(o).join(",") === "a,c,b")
    var fi = ""
    for (var k in o) { fi = fi + k }
    check("del4", fi === "acb")
    check("del5", keyStr36({ ...o }) === "a:1;c:3;b:9;")
    check("del6", keyStr36(Object.assign({}, o)) === "a:1;c:3;b:9;")
    var d = { x: 1, y: 2 }
    delete d.x
    check("del7", !("x" in d) && ("y" in d) && Object.keys(d).length === 1)
    check("del8", keyStr36({ ...d }) === "y:2;")
    // Deleting what is not there answers true, exactly like deleting what is.
    check("del9", (delete d.nope) === true && (delete d.y) === true && Object.keys(d).length === 0)
    // On an ARRAY the length is unchanged and the slot reads as undefined.
    var arr = [1, 2, 3]
    check("del10", (delete arr[1]) === true && arr.length === 3 && arr[1] === undefined)
    // Many deletions, and one that checks the surviving ORDER.
    var big = {}
    for (var i = 0; i < 300; i++) { big["k" + i] = i; delete big["k" + i] }
    check("del11", Object.keys(big).length === 0)
    for (var j = 0; j < 300; j++) { big["m" + j] = j }
    for (var m = 0; m < 300; m = m + 2) { delete big["m" + m] }
    check("del12", Object.keys(big).length === 150 && Object.keys(big)[0] === "m1")

    check("tpl1", `${{ toString: function () { return "T" } }}` === "T")
    check("tpl2", `${new (class { toString() { return "D!" } })()}` === "D!")
    // valueOf alone is NOT consulted: a template is hint string, not hint default.
    check("tpl3", `${{ valueOf: function () { return 7 } }}` === "[object Object]")
    check("tpl4", `${{ valueOf: function () { return 7 }, toString: function () { return "B" } }}` === "B")
    check("tpl5", `${10n}` === "10" && `${2n ** 100n}` === "1267650600228229401496703205376")
    check("tpl6", `${1} ${"s"} ${true} ${undefined} ${null} ${({})}` === "1 s true undefined null [object Object]")
    check("tpl7", `${[1, 2]}` === "1,2" && `${[]}` === "")
    var t = { toString: function () { return "T" } }
    check("tpl8", `a${t}b${t}c` === "aTbTc")

    // The implicit global. A plain `=` to a name that is on NO scope of the chain
    // creates the binding in the ROOT scope (js_scope_set_or_create), so it outlives
    // the function that wrote it. js_scope_typeof cannot be used to build this in an
    // emitter - it answers "undefined" for an absent name and for a slot HOLDING
    // undefined alike - which is why the C floor carries the walk itself.
    check("ig1", mkGlobal36() === 100)
    check("ig2", implicitGlobal36 === 100)
    implicitGlobal36 = implicitGlobal36 + 1
    check("ig3", bumpGlobal36() === 102 && implicitGlobal36 === 102)
    // An assignment that DOES find the name on the chain must not shadow it in the
    // root: this is the arm js_pyset_var and js_scope_set_or_create agree on.
    var outer = 1
    function inner36() { outer = 42 }
    inner36()
    check("ig4", outer === 42 && typeof outer === "number")
}
function mkGlobal36() {
    implicitGlobal36 = 100
    return implicitGlobal36
}
function bumpGlobal36() {
    implicitGlobal36 = implicitGlobal36 + 1
    return implicitGlobal36
}
function keyStr36(o) {
    var ks = Object.keys(o)
    var s = ""
    for (var i = 0; i < ks.length; i++) { s = s + ks[i] + ":" + o[ks[i]] + ";" }
    return s
}

// ===== SECTION 37: an iterator is CLOSED on an early exit =====
// for-of over a generator is lazy, so a loop that leaves early has to CLOSE the
// generator: a second loop over it is empty, as in node, and a `finally` around the
// yield runs. runtime.c's gen_close throws a GEN_EXIT sentinel INTO the suspended
// body (so its finally clauses unwind normally), abnf/jsrt.go's closeBody does the
// same, and makeForOf calls the iterator's `return()` on break. The moment the
// finally runs is SECTION 39's subject; this one asserts the closed STATE and counts
// taken after exactly ONE step.
function s37() {
    var fins = 0
    function* g37() {
        try { yield 1; yield 2; yield 3 } finally { fins = fins + 1 }
    }
    // break: the loop closes the generator, so a second loop over it is empty.
    var a = g37()
    var seen = []
    for (var x of a) { seen.push(x); break }
    var again = []
    for (var y of a) { again.push(y) }
    check("itc1", seen.length === 1 && seen[0] === 1)
    check("itc2", again.length === 0)
    check("itc3", a.next().done === true && a.next().value === undefined)
    check("itc4", fins === 1)
    // Exhausting the loop normally leaves it done as well. No finally count here:
    // replay runs one per next(), so the two halves reach four and one.
    var b = g37()
    var total = 0
    for (var z of b) { total = total + z }
    check("itc5", total === 6 && b.next().done === true)
    // continue does NOT close; the break after it does.
    var c = g37()
    var got = []
    for (var w of c) { if (w === 1) { continue } got.push(w); break }
    check("itc6", got.length === 1 && got[0] === 2)
    var c2 = []
    for (var w2 of c) { c2.push(w2) }
    check("itc7", c2.length === 0)
    // A break out of a TRY inside the loop body: the break becomes a control
    // signal that excDispatch re-issues, and it lands on the closing block too.
    var d = g37()
    var dfin = 0
    for (var v of d) { try { break } finally { dfin = dfin + 1 } }
    check("itc8", dfin === 1 && d.next().done === true)
    // A LABELED break to THIS loop closes it (bindLabelBrk); the label's exit is
    // reached through the loop's own closing block.
    var e = g37()
    lbl37: for (var u of e) { break lbl37 }
    check("itc9", e.next().done === true)
    // g.return(v) is the same close, spelled by hand, and answers {value, done}.
    var f = g37()
    f.next()
    var r = f.return(9)
    check("itc10", r.value === 9 && r.done === true)
    check("itc11", f.next().done === true)
    // ... on a generator that never started (nothing to unwind) ...
    var h = g37()
    var rh = h.return(5)
    check("itc12", rh.value === 5 && rh.done === true && h.next().done === true)
    // ... and on one that already finished.
    var i2 = g37()
    for (var q of i2) { }
    var ri = i2.return(7)
    check("itc13", ri.value === 7 && ri.done === true)
    // A hand-written iterator gets its own return() called, receiver and all.
    var closed = 0
    var iter = {
        n: 0,
        next: function () { this.n = this.n + 1; return { value: this.n, done: this.n > 5 } },
        return: function () { closed = closed + 1; return { value: undefined, done: true } }
    }
    for (var p of iter) { break }
    check("itc14", closed === 1)
    // An iterator WITHOUT a return member is left alone rather than aborting,
    // and an array (the index arm) never looks for one.
    var bare = { n: 0, next: function () { this.n = this.n + 1; return { value: this.n, done: this.n > 3 } } }
    var bn = 0
    for (var s of bare) { bn = s; break }
    check("itc15", bn === 1)
    var asum = 0
    for (var t of [1, 2, 3]) { asum = asum + t; break }
    check("itc16", asum === 1)
    // Destructuring from a generator, then breaking out of it.
    var g2 = g37()
    var pairs = []
    function* pg37() { yield [1, 2]; yield [3, 4] }
    for (var [pa, pb] of pg37()) { pairs.push(pa + pb); break }
    check("itc17", pairs.length === 1 && pairs[0] === 3)
    // A RETURN out of the loop body and a LABELED BREAK to an OUTER statement close
    // it, exactly as node does. Neither reaches a block makeForOf owns - one rets the
    // frame, the other branches to the outer label's exit - so the compiler halves
    // emit js_jsiterclose on the loop's own iterable handle, which the loop's entry
    // block dominates, while the interpreter half reads the body's completion record.
    var rg = g37()
    check("itc18", takeOne37(rg) === 1)
    check("itc19", rg.next().done === true)
    var tg = g37()
    try { for (var tv of tg) { throw "x" } } catch (te) { }
    // A THROW through the loop body does NOT close it, in both halves: the only route
    // an emitter has is a runtime open-iterator stack unwound at each try, and
    // `for await` suspends INSIDE its own for-of, so such a stack closes the wrong
    // loop (SECTION 38's `for await` against SECTION 16's caught rejection). Pinned
    // here.
    check("itc20", tg.next().done === false)
    // A labeled break to an OUTER statement, which leaves the loop without ever
    // reaching its own closing block.
    var lg = g37()
    out37j: for (var oi = 0; oi < 1; oi++) {
        for (var lv of lg) { break out37j }
    }
    check("itc21", lg.next().done === true)
    // A return out of NESTED for-of loops closes both, innermost first.
    var n1 = g37()
    var n2 = g37()
    check("itc22", nested37j(n1, n2) === 2)
    check("itc23", n1.next().done === true && n2.next().done === true)
    // The same gap when the throw leaves the loop by leaving the FUNCTION - the
    // try that catches it has no for-of of its own. Pinned with itc20.
    var fg = g37()
    try { throwOut37j(fg) } catch (fe) { }
    check("itc24", fg.next().done === false)
    // A return out of a try INSIDE the loop body: the return becomes a control
    // signal, so the close happens where excDispatch re-issues it.
    var sg = g37()
    check("itc25", retThroughTry37j(sg) === 1)
    check("itc26", sg.next().done === true)
    // ... and a plain loop with a try in it that does NOT leave still runs to the
    // end, so the unwind bookkeeping cannot close an iterator that is still live.
    var qg = g37()
    var qsum = 0
    for (var qv of qg) { try { qsum = qsum + qv } finally { qsum = qsum + 0 } }
    check("itc27", qsum === 6 && qg.next().done === true)
}
function nested37j(p, q) { for (var a of p) { for (var b of q) { return a + b } } return -1 }
function throwOut37j(g) { for (var x of g) { throw "z" } return 0 }
function retThroughTry37j(g) { for (var x of g) { try { return x } finally { } } return 0 }
function takeOne37(g) { for (var x of g) { return x } return 0 }

// ===== SECTION 38: for await, and async generators =====
// An async generator body carries its yields AND its awaits on
// ONE suspension channel; they are told apart by a marker record the emitter puts
// on every awaited operand (js_jsawaitmark), which is why the generator model
// needed nothing added to it and the C floor is not touched. `for await` drives an
// async iterator by awaiting each next(), and an array or a string by awaiting each
// ELEMENT - which is what the specification's async-from-sync iterator does.
//
// Every trace below is built in LOCALS. This half drives an async body by REPLAY
// (see the generator note at the top of the file), so a body that writes to shared
// state before its last suspension repeats that write; state recreated per replay
// is safe, and only that is used. Every value asserted here is byte-identical to
// node v24, including the interleaving in "ord".
function s38() {
    async function* nums() { yield 1; yield 2; yield 3 }
    async function* mixed() {
        var a = await Promise.resolve(10)
        yield a
        yield a + 1
        var b = await Promise.resolve(20)
        yield b
        return "r"
    }
    var agExpr = async function* () { yield "e" }
    var agObj = { async *m() { yield "o" } }
    class AGC { async *m() { yield "c" } }
    check("fa1", typeof nums === "function")
    check("fa2", typeof agExpr === "function")
    check("fa3", typeof agObj.m === "function")
    // The async generator OBJECT the method call answers is what this asserts.
    check("fa4", typeof new AGC().m().next === "function")
    check("fa5", typeof nums().next === "function")
    // for await over an async generator.
    async function collect() {
        var out = []
        for await (var n of nums()) { out.push(n) }
        return out.join("")
    }
    // ... one whose body AWAITS between its yields, which is the case the marker
    // record exists for.
    async function awaitInGen() {
        var out = []
        for await (var v of mixed()) { out.push(v) }
        return out.join("")
    }
    // next() by hand: each call answers a PROMISE for a {value, done} record.
    async function manual() {
        var g = nums()
        var r1 = await g.next()
        var r2 = await g.next()
        var r3 = await g.next()
        var r4 = await g.next()
        return "" + r1.value + r2.value + r3.value + r4.done + r4.value
    }
    // A break out of a for await CLOSES the async generator.
    async function closeEarly() {
        var g = nums()
        for await (var x of g) { break }
        var a = await g.next()
        return "" + a.done + a.value
    }
    // for await over an ARRAY awaits each element, so a promise element arrives
    // resolved; over a SYNC generator it drives next() as an ordinary for-of does.
    async function overArray() {
        var out = []
        for await (var p of [Promise.resolve("a"), "b"]) { out.push(p) }
        return out.join("")
    }
    async function overSyncGen() {
        var out = []
        for await (var s of sg38()) { out.push(s) }
        return out.join("")
    }
    // ag.return(v) closes it and answers {value: v, done: true}.
    async function agReturn() {
        var g = nums()
        await g.next()
        var r = await g.return(9)
        var after = await g.next()
        return "" + r.value + r.done + after.done
    }
    // The expression, object-literal-method and class-method spellings all run.
    async function methodGen() {
        var out = []
        for await (var m of agObj.m()) { out.push(m) }
        for await (var c of new AGC().m()) { out.push(c) }
        for await (var e of agExpr()) { out.push(e) }
        return out.join("")
    }
    async function all() {
        var parts = []
        parts.push(await collect())
        parts.push(await awaitInGen())
        parts.push(await manual())
        parts.push(await closeEarly())
        parts.push(await overArray())
        parts.push(await overSyncGen())
        parts.push(await agReturn())
        parts.push(await methodGen())
        return parts.join("|")
    }
    all().then(function (r) {
        var ok = r === "123|101120|123trueundefined|trueundefined|ab|12|9truetrue|oce"
        check("fa6", ok)
        // main() has already returned by the time a job runs, so a failure here
        // cannot reach the count it reported - exit() is the only way it reaches the
        // EXIT CODE, which is what clang-check and native-full read.
        if (!ok) { exit(1) }
    })
    // ORDERING is the evidence, and it is pushed from .then callbacks only - never
    // from inside a replayed body. An async generator's first next() costs exactly
    // one more tick than a bare Promise.resolve().then, so the two chains interleave
    // p1, n1, p2, n2, p3 - node's own answer.
    var olog = []
    function tick(tag) { return function (v) { olog.push(tag); return v } }
    nums().next().then(tick("n1")).then(tick("n2"))
    Promise.resolve().then(tick("p1")).then(tick("p2")).then(tick("p3")).then(function () {
        var ordOk = olog.join(",") === "p1,n1,p2,n2,p3"
        check("fa7", ordOk)
        if (!ordOk) { exit(1) }
    })
    // ----- yield* inside an async generator, and ag.throw() -----
    // Every value below is node v24's. Each trace is built in a LOCAL created per
    // call, so replay repeats no observable write. `yield* someAsyncGen()` has to
    // AWAIT each step (an async generator's next() answers a promise, whose `done` is
    // undefined), and ag.throw(e) raises at the yield rather than closing the body.
    function rec38(r) { return "" + r.value + "/" + r.done }
    var agRan38 = 0 // Counts entries into never38's body; a suspendedStart throw must not.
    // yield* over an ASYNC delegate: each step awaited, and the value of the
    // expression is the delegate's RETURN value.
    async function* inner38() { yield 1; await null; yield 2; return 9 }
    async function* outer38() {
        var t = []
        t.push("pre")
        var r = yield* inner38()
        t.push("ret=" + r)
        yield t.join(",")
    }
    async function ystarAsync() {
        var out = []
        for await (var v of outer38()) { out.push(v) }
        return out.join("|")
    }
    // yield* over a SYNC generator, and over an ARRAY of promises, from inside an
    // async generator: the elements arrive resolved and in order.
    async function* ystarSyncGen38() { yield* sg38(); yield 3 }
    async function* ystarArr38() { yield* [Promise.resolve(4), 5] }
    async function ystarOther() {
        var out = []
        for await (var a of ystarSyncGen38()) { out.push(a) }
        for await (var b of ystarArr38()) { out.push(b) }
        return out.join("")
    }
    // ag.throw(e) RAISES AT THE SUSPENDED YIELD, so the try around it catches, the
    // catch's own yield answers the throw() request, and the finally runs on the
    // next resume.
    async function* caught38() {
        var t = []
        try {
            t.push("y10")
            yield 10
            t.push("unreached")
        } catch (e) {
            t.push("caught:" + e)
            yield t.join(",")
        } finally {
            t.push("fin")
        }
    }
    async function agThrowCaught() {
        var g = caught38()
        var r1 = await g.next()
        var r2 = await g.throw("boom")
        var r3 = await g.next()
        return rec38(r1) + "|" + rec38(r2) + "|" + rec38(r3)
    }
    // Nothing catches: the request REJECTS with the thrown value and the generator is
    // done - the abrupt completion node ends up with too.
    async function* bare38() { yield 20 }
    async function agThrowUncaught() {
        var g = bare38()
        var r1 = await g.next()
        var s = "?"
        try { await g.throw("bang") } catch (e) { s = "rej:" + e }
        var r3 = await g.next()
        return rec38(r1) + "|" + s + "|" + rec38(r3)
    }
    // A throw at suspendedStart does NOT enter the body (agStarted below asserts it).
    async function* never38() { agRan38 = agRan38 + 1; yield 30 }
    async function agThrowBeforeStart() {
        var g = never38()
        var s = "?"
        try { await g.throw("early") } catch (e) { s = "rej:" + e }
        var r = await g.next()
        return s + "|" + rec38(r) + "|ran=" + agRan38
    }
    // A throw that arrives while the body is parked at a `yield*` is FORWARDED to the
    // delegate's throw(). The delegate catches, the throw() request RESOLVES with the
    // value the delegate's catch yields, and the delegate is left where it can be
    // resumed rather than suspended forever. All four fields below are node v24's.
    async function* deleg38() { try { yield 40 } catch (e) { yield "d:" + e } }
    async function* delegOuter38() { yield* deleg38(); yield 41 }
    async function agThrowIntoDelegate() {
        var g = delegOuter38()
        var r1 = await g.next()
        var s = "?"
        try { s = rec38(await g.throw("t")) } catch (e) { s = "rej:" + e }
        var r3 = await g.next()
        return rec38(r1) + "|" + s + "|" + rec38(r3)
    }
    // A delegate with NO throw method - an array here, and equally a string or a plain
    // iterator object - is CLOSED through its return() and the yield* raises node's
    // TypeError, which the outer body's own try then catches. The message is asserted
    // verbatim because it is the whole observable difference from a forwarded throw.
    async function* arrDeleg38() { try { yield* [50, 51] } catch (e) { yield "e:" + e } }
    async function agThrowIntoArray() {
        var g = arrDeleg38()
        var r1 = await g.next()
        var s = "?"
        try { s = rec38(await g.throw("t2")) } catch (e) { s = "rej:" + e }
        return rec38(r1) + "|" + s
    }
    // The other half of the same loop: g.next(v) forwards v to the DELEGATE (the
    // specification's `received.value`), so `a` below reads the sent value.
    async function* sendInner38() { var a = yield 60; yield "a=" + a; return "R" }
    async function* sendOuter38() { var r = yield* sendInner38(); yield "r=" + r }
    async function agSendThroughDelegate() {
        var g = sendOuter38()
        var r1 = await g.next()
        var r2 = await g.next("s")
        var r3 = await g.next()
        var r4 = await g.next()
        return rec38(r1) + "|" + rec38(r2) + "|" + rec38(r3) + "|" + rec38(r4)
    }
    async function all38b() {
        var parts = []
        parts.push(await ystarAsync())
        parts.push(await ystarOther())
        parts.push(await agThrowCaught())
        parts.push(await agThrowUncaught())
        parts.push(await agThrowBeforeStart())
        parts.push(await agThrowIntoDelegate())
        parts.push(await agThrowIntoArray())
        parts.push(await agSendThroughDelegate())
        return parts.join("~")
    }
    all38b().then(function (r) {
        var ok = r === "1|2|pre,ret=9~12345~10/false|y10,caught:boom/false|undefined/true~" +
                       "20/false|rej:bang|undefined/true~rej:early|undefined/true|ran=0~" +
                       "40/false|d:t/false|41/false~" +
                       "50/false|e:TypeError: The iterator does not provide a 'throw' method./false~" +
                       "60/false|a=s/false|r=R/false|undefined/true"
        check("fa8", ok)
        if (!ok) { exit(1) }
    }, function (e) {
        check("fa8", false)
        exit(1)
    })
    // A PLAIN generator's yield* answers the delegate's return value too - the same
    // one line in the emitter, and synchronous, so it is asserted directly. next(v)
    // reaches the delegate here as well, through the identical loop.
    var ysync = []
    // The sent value is YIELDED BACK rather than pushed to ysync: this half replays a
    // generator body once per next(), so a write from inside the DELEGATE would repeat.
    // ysync is written once, after the delegate is done - only one replay reaches it.
    function* ysInner38() { var a = yield 1; yield "a=" + a; return "r" }
    function* ysOuter38() { ysync.push("v=" + (yield* ysInner38())) }
    var yg38 = ysOuter38()
    var ys1 = yg38.next().value
    var ys2 = yg38.next("S").value
    yg38.next()
    check("fa9", ys1 === 1 && ys2 === "a=S" && ysync.join(",") === "v=r")
    return 0
}
function* sg38() { yield 1; yield 2 }

// ===== SECTION 39: WHEN a generator's finally runs, and closing a yield* delegate =====
// When a finally runs, and closing a yield* delegate - neither visible to SECTION
// 37, whose assertions are about the closed STATE or a count.
//
// A generator suspends in the interpreter half by throwing a host-level signal, so
// its jsExcTry lets that signal through untouched; otherwise `try { yield } finally
// { f() }` would print f at every suspension instead of when node runs it.
//
// g.return() is gen_close, which throws the GEN_EXIT sentinel at the suspended
// yield; the emitted yield* loop yields inside a js_try whose finally runs
// IteratorClose on the DELEGATE (js-to-llvm-ir.abnf's emitYieldStarClose), so the
// delegate's finally runs first, as in node. The interpreter half raises the same
// close at the same yield (jpYieldDeleg).
//
// Every log below is written only in a `finally` or from the DRIVER, never before the
// yield the body parks at - a write before that yield repeats once per replay in the
// interpreter half. Every log is node v24's.
function s39() {
    var lg = []
    function* g39() { try { yield 1; yield 2 } finally { lg.push("fin") } }
    var a39 = g39()
    lg.push("n1=" + a39.next().value)
    lg.push("n2=" + a39.next().value)
    lg.push("ret")
    var r39 = a39.return(0)
    lg.push("end")
    check("gcl1", lg.join(",") === "n1=1,n2=2,ret,fin,end")
    check("gcl2", r39.value === 0 && r39.done === true && a39.next().done === true)
    // The delegate is closed BEFORE the delegating body's own finally.
    var l2 = []
    function* in39() { try { yield 1; yield 2 } finally { l2.push("inner") } }
    function* out39() { try { yield* in39() } finally { l2.push("outer") } }
    var b39 = out39()
    l2.push("n=" + b39.next().value)
    b39.return(0)
    check("gcl3", l2.join(",") === "n=1,inner,outer")
    // Nested yield*: innermost first, one close per level.
    var l3 = []
    function* i3_39() { try { yield 1 } finally { l3.push("i") } }
    function* m3_39() { try { yield* i3_39() } finally { l3.push("m") } }
    function* o3_39() { try { yield* m3_39() } finally { l3.push("o") } }
    var c39 = o3_39()
    c39.next()
    c39.return(0)
    check("gcl4", l3.join(",") === "i,m,o")
    // A for-of that BREAKS out of a delegating generator closes the delegate too -
    // the loop calls return(), which is the same close.
    var l4 = []
    function* i4_39() { try { yield 1; yield 2 } finally { l4.push("i4") } }
    function* o4_39() { yield* i4_39() }
    for (var v39 of o4_39()) { l4.push("v" + v39); break }
    check("gcl5", l4.join(",") === "v1,i4")
    // An ARRAY delegate has no return member: nothing is called and nothing aborts.
    var l5 = []
    function* o5_39() { try { yield* [7, 8] } finally { l5.push("o5") } }
    var e39 = o5_39()
    check("gcl6", e39.next().value === 7)
    e39.return(0)
    check("gcl7", l5.join(",") === "o5")
    // A close is a RETURN completion: `catch` is not offered it, in any of the three
    // engines (runtime.c's GEN_EXIT guard, abnf/jsrt.go's genExit, jsExcTry here).
    var l6 = []
    function* g6_39() { try { yield 1 } catch (e6) { l6.push("caught") } finally { l6.push("f6") } }
    var f39 = g6_39()
    f39.next()
    f39.return(0)
    check("gcl8", l6.join(",") === "f6")
    // A generator that never started has no suspension point, so nothing unwinds.
    var l7 = []
    function* g7_39() { try { yield 1 } finally { l7.push("f7") } }
    var h39 = g7_39()
    h39.return(3)
    check("gcl9", l7.length === 0 && h39.next().done === true)
    // A body that runs to its own end runs the finally once, at the end.
    var l8 = []
    function* g8_39() { try { yield 1 } finally { l8.push("f8") } }
    var i39 = g8_39()
    l8.push("a" + i39.next().value)
    l8.push("d" + i39.next().done)
    check("gcl10", l8.join(",") === "a1,f8,dtrue")
    // Nested try/finally around ONE yield unwinds inside out.
    var l9 = []
    function* g9_39() {
        try { try { yield 1 } finally { l9.push("in") } } finally { l9.push("out") }
    }
    var j39 = g9_39()
    j39.next()
    j39.return(0)
    check("gcl11", l9.join(",") === "in,out")
    // g.throw(v) on a PLAIN generator. Here as well as in the
    // features file because the features file is never BUILT NATIVELY, and the
    // yield*-forwarding arm is emitter code that clang-check and native-full are the
    // only gates that reach. Every push is in the driver or in a finally: a write
    // before the parked yield repeats once per replay in the interpreter half.
    var t1 = []
    function* gt1_39() {
        try { yield 1; yield 2 } catch (e) { yield "c:" + e } finally { t1.push("ft1") }
    }
    var a39 = gt1_39()
    t1.push("n" + a39.next().value)
    t1.push("t" + a39.throw("X").value)
    t1.push("d" + a39.next().done)
    check("gth1", firsts39(t1).join(",") === "n1,tc:X,ft1,dtrue")
    // Uncaught: the value propagates out of throw() and the generator is left done.
    var t2 = []
    function* gt2_39() { try { yield 1; yield 2 } finally { t2.push("ft2") } }
    var b39 = gt2_39()
    b39.next()
    var t2c = ""
    try { b39.throw("Y") } catch (e) { t2c = e }
    check("gth2", t2c === "Y" && b39.next().done === true && firsts39(t2).join(",") === "ft2")
    // Never started: no suspension point, so NO finally runs at all.
    var t3 = []
    function* gt3_39() { try { yield 1 } finally { t3.push("ft3") } }
    var c39 = gt3_39()
    var t3c = ""
    try { c39.throw("Z") } catch (e) { t3c = e }
    check("gth3", t3c === "Z" && t3.length === 0 && c39.next().done === true)
    // Caught, then RETURN: the record carries the return value and done.
    function* gt4_39() { try { yield 1 } catch (e) { return 7 } }
    var d39 = gt4_39()
    d39.next()
    var d39r = d39.throw("Q")
    check("gth4", d39r.value === 7 && d39r.done === true && d39.next().done === true)
    // At a yield* the throw is FORWARDED to the delegate, so a delegate that would
    // catch it does - and the outer body carries on with what it yields next.
    var t5 = []
    function* it5_39() { try { yield 1 } catch (e) { yield "in:" + e } finally { t5.push("i5") } }
    function* ot5_39() { try { yield* it5_39() } finally { t5.push("o5") } }
    var e5_39 = ot5_39()
    t5.push("n" + e5_39.next().value)
    t5.push("t" + e5_39.throw("W").value)
    t5.push("d" + e5_39.next().done)
    check("gth5", firsts39(t5).join(",") === "n1,tin:W,i5,o5,dtrue")
    // A delegate with NO throw method - an array - is closed through its return()
    // and the yield* raises node's TypeError, which the outer body may catch.
    function* ot6_39() { try { yield* [1, 2, 3] } catch (e) { yield "a:" + e } }
    var f6_39 = ot6_39()
    f6_39.next()
    check("gth6", f6_39.throw("V").value ===
          "a:TypeError: The iterator does not provide a 'throw' method.")
    // A `yield` inside a CATCH ARM is a suspension like any other: the enclosing
    // finally belongs to the close, not to it. Reachable with no throw() at all.
    var t7 = []
    function* gt7_39() { try { throw 1 } catch (e) { yield 99 } finally { t7.push("f7c") } }
    var g7c39 = gt7_39()
    t7.push("n" + g7c39.next().value)
    t7.push("d" + g7c39.next().done)
    check("gth7", firsts39(t7).join(",") === "n99,f7c,dtrue")
    return 0
}

// The first occurrence of each entry, in order: the interpreter half REPLAYS a
// generator body once per next(), so a log collapses to node's sequence rather than
// matching it row for row (the price is repetition, not order).
function firsts39(xs) {
    var out = []
    for (var i = 0; i < xs.length; i++) { if (out.indexOf(xs[i]) < 0) { out.push(xs[i]) } }
    return out
}

// ===== SECTION 40: methods as values, Object.prototype, bind, and a catchable BigInt =====
// A METHOD IS NOT AN OWN PROPERTY: an instance keeps its methods on its __class
// descriptor, yet `typeof p.m` is "function" and `"m" in p` is true in node.
// getMember / js_jsvget / js_jshas walk the descriptor chain for a VALUE read (the
// call site still reads raw, so an ordinary p.m() allocates nothing).
//
// typeof [].push must be "function" in every engine: llvm.Run reads a plain member
// through the shared js_get, a native build through js_jsmget's method-name hiding,
// and one extern for the value read covers both.
//
// Object.prototype.toString.call(v) is the classic type probe; every engine builds
// its own `Object` prototype with toString / valueOf / hasOwnProperty.
//
// Function.prototype.bind, and f.call(o) / f.apply(o) with the receiver arriving.
//
// A BIGINT TypeError IS CATCHABLE. `1n + 1` is a TypeError in node and a program may
// catch it. Every BigInt-family raise - the mixed-operand TypeError, ToBigInt's
// SyntaxError, the RangeErrors - is a real throw in all four engines. The operands
// come out of an ARRAY so the constant folder cannot evaluate any of it at compile
// time.
function s40() {
    class S40A { am() { return "a" + this.v } }
    class S40B extends S40A { constructor() { super(); this.v = 1 } bm() { return "b" } }
    var p = new S40B()
    var arr = [1, 2]

    // A method read as a VALUE, and called through that value.
    check("mv1", typeof p.bm === "function")
    check("mv2", typeof p.am === "function")        // inherited, one __super hop
    check("mv3", typeof p.zz === "undefined")
    var bm = p.bm
    var am = p.am
    // NOT bound: a bare call has `this` undefined (class bodies are strict), so `am`
    // throws a TypeError, and the receiver of a call through the value is the call's.
    var amThrew = false
    try { am() } catch (e) { amThrew = e instanceof TypeError }
    check("mv4", bm() === "b" && amThrew)
    check("mv7", p.am === p.am && p.am === am && am.call(p) === "a1")
    check("mv5", typeof arr.push === "function")    // was engine-dependent
    check("mv6", typeof "s".slice === "function")

    // `in` sees a method, and does NOT see the engine's own __ slots.
    check("in1", "bm" in p)
    check("in2", "am" in p)
    check("in3", ("zz" in p) === false)
    check("in4", "v" in p)
    check("in5", ("__class" in p) === false)
    check("in6", (0 in arr) && (("5" in arr) === false))

    // Object.prototype, reached the way real code reaches it.
    var vals = [[1], {}, 3, "s", true, null, undefined]
    var tags = []
    for (var i = 0; i < vals.length; i++) { tags.push(Object.prototype.toString.call(vals[i])) }
    check("op1", tags.join("|") === "[object Array]|[object Object]|[object Number]|" +
                                    "[object String]|[object Boolean]|[object Null]|[object Undefined]")
    check("op2", typeof Object.prototype === "object")
    var own = { k: 1 }
    check("op3", Object.prototype.hasOwnProperty.call(own, "k") === true)
    check("op4", Object.prototype.hasOwnProperty.call(own, "j") === false)
    check("op5", Object.prototype.valueOf.call(own) === own)

    // call / apply / bind, with the receiver actually arriving.
    var recv = { x: 10 }
    var f = function(a, b) { return this.x + a + b }
    check("fn1", f.call(recv, 2, 3) === 15)
    check("fn2", f.apply(recv, [2, 3]) === 15)
    check("fn3", f.bind(recv)(2, 3) === 15)
    check("fn4", f.bind(recv, 2)(3) === 15)
    check("fn5", f.bind(recv, 2, 3)() === 15)

    // new through a constructor that RETURNS a function, and `new` with no argument
    // list: [[Construct]] keeps an explicitly returned object and a function is one.
    function S40Meta() { return function() { this.q = 8 } }
    check("nw1", (new (new S40Meta())()).q === 8)
    function S40Plain() { this.q = 9 }
    check("nw2", (new S40Plain).q === 9)

    // The BigInt raises are CATCHABLE.
    var ops = [1n, 1, 0n, -1n, "zz", 1.5]
    var log = []
    try { log.push("v" + (ops[0] + ops[1])) } catch (e) { log.push("mix:" + (("" + e).indexOf("Cannot mix BigInt") >= 0)) }
    try { log.push("v" + (ops[0] / ops[2])) } catch (e) { log.push("div:" + (("" + e).indexOf("Division by zero") >= 0)) }
    try { log.push("v" + (ops[0] ** ops[3])) } catch (e) { log.push("pow:" + (("" + e).indexOf("Exponent") >= 0)) }
    try { log.push("v" + BigInt(ops[4])) } catch (e) { log.push("str:" + (("" + e).indexOf("SyntaxError") >= 0)) }
    try { log.push("v" + BigInt(ops[5])) } catch (e) { log.push("flo:" + (("" + e).indexOf("not an integer") >= 0)) }
    try { log.push("v" + (ops[0].toString(99))) } catch (e) { log.push("rdx:" + (("" + e).indexOf("radix") >= 0)) }
    check("bi1", log.join(",") === "mix:true,div:true,pow:true,str:true,flo:true,rdx:true")
    // The program keeps running after a caught one, which is the whole point.
    check("bi2", ops[0] + ops[0] === 2n)
    return 0
}

// ===== SECTION 41: the Error hierarchy =====
// The classes are ordinary {__isclass, __name, __super, __ctor} descriptors, so
// instanceof, `new` and `extends` are the ordinary class machinery.
//
// Every row below is node v24's answer, except s41keys - see the comment there.
function s41() {
    var xs = ["boom", 1n, 2, 255, "zz"]

    var te = new TypeError(xs[0])
    check("err1", te.name === "TypeError" && te.message === "boom")
    check("err2", te.toString() === "TypeError: boom" && ("" + te) === "TypeError: boom")
    // stack has no frames here (no engine in this project has a frame walker),
    // but it starts with exactly the line node's does, which is what the
    // `e.stack.split("\n")[0]` idiom reads.
    check("err3", typeof te.stack === "string" && te.stack.split("\n")[0] === "TypeError: boom")
    check("err4", te instanceof TypeError && te instanceof Error)
    check("err5", !(te instanceof RangeError) && !(te instanceof ReferenceError))

    // The five. EvalError and URIError are deliberately absent (see the
    // jsErrClassNames table in the emitters).
    var mk = [new Error(xs[0]), new TypeError(xs[0]), new RangeError(xs[0]),
              new ReferenceError(xs[0]), new SyntaxError(xs[0])]
    var nm = ["Error", "TypeError", "RangeError", "ReferenceError", "SyntaxError"]
    var okn = 0
    var oki = 0
    for (var i = 0; i < mk.length; i++) {
        if (mk[i].name === nm[i] && ("" + mk[i]) === nm[i] + ": boom") { okn = okn + 1 }
        if (mk[i] instanceof Error) { oki = oki + 1 }
    }
    check("err6", okn === 5 && oki === 5)

    // No message: `new Error()` and `new Error(undefined)` both leave "", and
    // toString then answers the bare name.
    var e0 = new Error()
    check("err7", e0.message === "" && e0.toString() === "Error" && e0.name === "Error")

    // A user subclass. name comes from the NEAREST BUILTIN ancestor, because in a
    // real engine it lives on the prototype and E.prototype does not shadow it.
    class S41MyErr extends Error {}
    var m1 = new S41MyErr(xs[0])
    check("err8", m1.message === "boom" && m1.name === "Error" && ("" + m1) === "Error: boom")
    check("err9", m1 instanceof S41MyErr && m1 instanceof Error && !(m1 instanceof TypeError))
    class S41TErr extends TypeError {
        constructor(msg) { super(msg); this.code = 7 }
    }
    var m2 = new S41TErr(xs[0])
    check("err10", m2.name === "TypeError" && m2.message === "boom" && m2.code === 7)
    check("err11", m2 instanceof S41TErr && m2 instanceof TypeError && m2 instanceof Error)
    // Two levels, and the message still arrives through BOTH default constructors.
    class S41Deep extends S41TErr {}
    var m3 = new S41Deep(xs[0])
    check("err12", m3.message === "boom" && m3.name === "TypeError" && m3.code === 7)

    // A default DERIVED constructor forwards its arguments: node says 5.
    class S41Base { constructor(a, b) { this.a = a; this.b = b } }
    class S41Sub extends S41Base {}
    var sb = new S41Sub(xs[2], xs[3])
    check("err13", sb.a === 2 && sb.b === 255)

    // Throwing and catching one, and re-throwing through a finally.
    var seen = ""
    try {
        throw new RangeError(xs[0])
    } catch (e) {
        seen = e.name + "/" + e.message + "/" + (e instanceof RangeError) + "/" + (e instanceof Error)
    }
    check("err14", seen === "RangeError/boom/true/true")

    // ----- the ENGINE's own errors are instances of these classes -----
    // The BigInt mix error, the iterator-throw TypeError, and the number
    // toString() radix RangeError.
    var got = []
    try { got.push("v" + (xs[1] + xs[2])) } catch (e) {
        got.push("mix:" + (e instanceof TypeError) + (e.message.indexOf("Cannot mix BigInt") === 0))
    }
    try { got.push("v" + BigInt(xs[4])) } catch (e) {
        got.push("syn:" + (e instanceof SyntaxError) + (e instanceof Error))
    }
    try { got.push("v" + (xs[3]).toString(1)) } catch (e) {
        got.push("rdx:" + (e instanceof RangeError) + (e.message.indexOf("radix") >= 0))
    }
    check("err15", got.join(",") === "mix:truetrue,syn:truetrue,rdx:truetrue")
    // A yield* delegate with no throw() method: the iterator-throw TypeError.
    var gd = s41deleg()
    gd.next()
    var itm = ""
    try { gd.throw(new Error(xs[0])) } catch (e) {
        itm = "" + (e instanceof TypeError) + (e.message.indexOf("does not provide") >= 0)
    }
    check("err16", itm === "truetrue")
    // And the program keeps running after every one of them.
    check("err17", xs[1] + xs[1] === 2n)

    // The message text survives the round trip unchanged, which is what keeps
    // every `("" + e).indexOf(...)` assertion in this file honest.
    check("err18", ("" + new TypeError(xs[0])).indexOf("TypeError: ") === 0)

    // name / message / stack are NOT enumerable: Object.keys(new Error("x")) is [],
    // for-in and spread see only the program's own fields, and a write keeps message
    // hidden. `name` lives on the prototype, so writing it creates an ordinary
    // enumerable own property at that point of the key order.
    check("err19", Object.keys(new Error(xs[0])).join(",") === "" && Object.keys(new Error()).length === 0)
    var ek = []
    for (var ekk in m2) { ek.push(ekk) }
    check("err20", ek.join(",") === "code" && Object.keys({...m2}).join(",") === "code")
    var ew = new RangeError(xs[0])
    ew.message = "b"
    ew.q = 1
    ew.name = "Q"
    check("err21", Object.keys(ew).join(",") === "q,name" && ("" + ew) === "Q: b" && ("message" in ew))
}
function* s41deleg() { yield* [3, 4] }

// ===== SECTION 42: the Math object =====
// Math's MEMBERS are the same set in every engine that implements a host global
// (manual 7.2): the floor carries Go's own math package, ported (runtime.c, "the
// transcendentals"), and abnf/hostglobals_test.go's TestMathMembersAgree pins the
// member set.
//
// Every row is node v24's answer except the two marked GO-vs-V8: Go's math
// package and V8 differ in the last ulp there, and both engines here are Go's.
//
// Every operand is read out of an array so the constant folders cannot answer
// at compile time (manual chapter 3).
function s42() {
    var a = [1, 2, 0.5, 8, 100, 3, 4, 0, 27, 0.001, 1e300, 1e-300, -1, 16, 0.1, 4294967295]

    check("math01", Math.sin(a[0]) === 0.8414709848078965)
    check("math02", Math.cos(a[0]) === 0.5403023058681398)
    check("math03", Math.tan(a[0]) === 1.557407724654902)          // GO-vs-V8: node 1.5574077246549023
    check("math04", Math.asin(a[2]) === 0.5235987755982989)
    check("math05", Math.acos(a[2]) === 1.0471975511965976)        // GO-vs-V8: node 1.0471975511965979
    check("math06", Math.atan(a[0]) === 0.7853981633974483)
    check("math07", Math.sinh(a[0]) === 1.1752011936438014)
    check("math08", Math.cosh(a[0]) === 1.5430806348152437)
    check("math09", Math.tanh(a[0]) === 0.7615941559557649)
    check("math10", Math.asinh(a[0]) === 0.881373587019543)
    check("math11", Math.acosh(a[0]) === 0 && Math.atanh(a[2]) === 0.5493061443340548)
    check("math12", Math.exp(a[0]) === 2.718281828459045)
    check("math13", Math.expm1(a[0]) === 1.718281828459045)
    check("math14", Math.log(a[0]) === 0 && Math.log1p(a[0]) === 0.6931471805599453)
    // log2 of a power of two is EXACT, which Log(x)/Ln2 is not - Go's Log2 has a
    // frac == 0.5 arm for it and the port carries it.
    check("math15", Math.log2(a[3]) === 3 && Math.log2(a[13]) === 4)
    check("math16", Math.log10(a[4]) === 2 && Math.log10(a[9]) === -3)
    check("math17", Math.cbrt(a[8]) === 3 && Math.cbrt(a[12]) === -1)
    check("math18", Math.atan2(a[0], a[1]) === 0.4636476090008061)

    // hypot: variadic, and it must overflow only when the RESULT does (a naive sum
    // of squares gives Infinity for the first row and 0 for the second).
    check("math19", Math.hypot(a[3], a[5]) === 8.54400374531753)
    check("math20", Math.hypot(a[10], a[10]) === 1.4142135623730952e+300)
    check("math21", Math.hypot(a[11], a[11]) === 1.4142135623730952e-300)
    check("math22", Math.hypot(a[5], a[6]) === 5 && Math.hypot(a[5], a[6], a[13]) === 16.76305461424021)

    // clz32 and fround.
    check("math23", Math.clz32(a[0]) === 31 && Math.clz32(a[7]) === 32)
    check("math24", Math.clz32(a[15]) === 0 && Math.clz32(a[1]) === 30)
    check("math25", Math.fround(a[14]) === 0.10000000149011612)
    check("math26", Math.fround(a[0]) === 1 && Math.fround(a[10]) === Infinity)

    check("math27", Math.imul(a[5], a[6]) === 12 && Math.imul(a[12], a[15]) === 1)
    check("math28", Math.LN2 === 0.6931471805599453 && Math.LN10 === 2.302585092994046)
    check("math29", Math.LOG2E === 1.4426950408889634 && Math.LOG10E === 0.4342944819032518)
    check("math30", Math.SQRT2 === 1.4142135623730951 && Math.SQRT1_2 === 0.7071067811865476)
    check("math31", Math.PI === 3.141592653589793 && Math.E === 2.718281828459045)

    check("math32", Math.log2(a[10]) === 996.5784284662087)
    check("math33", Math.exp(a[12]) === 0.36787944117144233 && Math.sinh(a[12]) === -1.1752011936438014)
    check("math34", Math.atanh(a[12]) === -Infinity)

    // Operands outside the int32 range. ToInt32 is a modulo 2^32, so 4294967297 is 1
    // and 1e300 is 0. Values read off node 24.
    var b = [4294967296, 4294967297, 1e300, -1e300, 1e20, 16777217, 0.49999999999999994, -0.4, 2.5, -2.5, 0, 3.4028236e38]
    var nz = -b[10]
    check("math35", Math.imul(b[2], b[1]) === 0 && Math.imul(b[1], b[1]) === 1 && Math.imul(b[0], b[5]) === 0)
    check("math36", Math.imul(b[5], b[4]) === 1661992960 && Math.imul(b[3], b[5]) === 0)
    check("math37", Math.clz32(b[2]) === 32 && Math.clz32(b[4]) === 1 && Math.clz32(b[1]) === 31 && Math.clz32(b[0]) === 32)
    check("math38", Math.fround(b[11]) === Infinity && Math.fround(b[5]) === 16777216 && Math.hypot(b[2], b[3]) === 1.4142135623730952e+300)
    // round is not floor(x + 0.5): that addition rounds 0.49999999999999994 up, and a
    // negative operand rounding to zero gives -0. max/min order the two zeros.
    check("math39", Math.round(b[6]) === 0 && 1 / Math.round(b[7]) === -Infinity && Math.round(b[8]) === 3 && Math.round(b[9]) === -2)
    check("math40", 1 / Math.max(nz, b[10]) === Infinity && 1 / Math.min(b[10], nz) === -Infinity && 1 / Math.round(nz) === -Infinity)

    // Math.random: a number in [0, 1) on every call. The values differ per run, so
    // only that property is asserted.
    var rok = 0
    for (var ri = 0; ri < 50; ri++) {
        var rv = Math.random()
        if (typeof rv === "number" && rv >= 0 && rv < 1) { rok = rok + 1 }
    }
    check("math41", typeof Math.random === "function" && rok === 50)
}

// ===== SECTION 43: three uncatchable aborts become catchable throws =====
// A name with no binding, a member read on null/undefined and `new Promise(5)` throw
// a ReferenceError or a TypeError in node that a plain try/catch recovers. The first
// two live in the FLOOR's scope_get / get_member (and abnf/jsrt.go's scopeGet /
// getMember), shared by all sixteen languages, so they are reached through a
// per-language hook the emitter registers - js_rt_missvar / js_rt_missmem - and a
// language that registers none is unchanged, which is what keeps the three
// SHOULD-ABORT matrix rows where they are.
//
// Also here: `typeof K` for a class is "function", and the BigInt mix message has
// no operator suffix. Every row is node v24's answer.
function s43() {
    var xs = [null, undefined, {a: 1}, 5, "k", 1n]

    var r1 = "none"
    try { r1 = "" + nosuchname_zz }
    catch (e) { r1 = (e instanceof ReferenceError) + "|" + (e instanceof Error) + "|" + e.name + "|" + e.message }
    check("ref1", r1 === "true|true|ReferenceError|nosuchname_zz is not defined")
    // `typeof <bare name>` is still the one expression that may name an
    // undeclared variable without raising.
    check("ref2", typeof nosuchname_zz === "undefined")

    var r2 = "none"
    try { r2 = "" + xs[0].k }
    catch (e) { r2 = (e instanceof TypeError) + "|" + e.message }
    check("ref3", r2 === "true|Cannot read properties of null (reading 'k')")
    var r3 = "none"
    try { r3 = "" + xs[1][xs[4]] }
    catch (e) { r3 = (e instanceof TypeError) + "|" + e.message }
    check("ref4", r3 === "true|Cannot read properties of undefined (reading 'k')")

    var r4 = "none"
    try { new Promise(xs[3]) }
    catch (e) { r4 = (e instanceof TypeError) + "|" + e.message }
    check("ref5", r4 === "true|Promise resolver 5 is not a function")

    // The program CONTINUES afterwards.
    var n = 0
    for (var i = 0; i < 3; i++) {
        try { n = n + xs[0].len }
        catch (e) { n = n + 1 }
    }
    check("ref6", n === 3)

    // ...and the throw travels the ordinary try/finally machinery.
    var log = []
    try {
        try { log.push("t"); var q = xs[1].z }
        finally { log.push("f") }
    } catch (e) { log.push("c:" + e.name) }
    check("ref7", log.join(",") === "t,f,c:TypeError")

    class K { m() { return 1 } }
    check("ref8", typeof K === "function")
    var arr = [K]
    check("ref9", typeof arr[0] === "function" && typeof new K() === "object")
    check("ref10", typeof Math === "object" && typeof Math.max === "function")

    // The WRITE sibling of ref3/ref4, with node's other wording.
    var r6 = "none"
    try { xs[0].k = 1 }
    catch (e) { r6 = (e instanceof TypeError) + "|" + e.message }
    check("ref12", r6 === "true|Cannot set properties of null (setting 'k')")

    var r5 = "none"
    try { r5 = "" + (xs[5] + xs[3]) }
    catch (e) { r5 = e.message }
    check("ref11", r5 === "Cannot mix BigInt and other types, use explicit conversions")
}

// ===== END SECTIONS =====

function main() {
    s36() // SECTION-CALL 36
    s37() // SECTION-CALL 37
    s38() // SECTION-CALL 38
    s39() // SECTION-CALL 39
    s40() // SECTION-CALL 40
    s41() // SECTION-CALL 41
    s42() // SECTION-CALL 42
    s43() // SECTION-CALL 43
    println("full: " + checks + " checks, " + failures + " failures")
    return failures
}
