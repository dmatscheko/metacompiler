// Full-syntax test: JavaScript - part 2 of 4, sections 36-43 and 101-108.
//
// THE JS RATCHET IS FOUR FILES: as one file its -frozen interpreter leg ran into
// ./test.sh --full's time brake (peak memory grew superlinearly with the file), and
// the fourth is a Module, the one goal symbol in which 'export' may stand.
//
//   tests/js-test-full.js     sections 01-35, the ECMAScript-2022 syntax walk
//   tests/js-test-full.2.js   sections 36-43 and 101-108, engine agreement, generators,
//                             Error, Math, builtin constructors, integrity levels
//   tests/js-test-full.3.js   sections 44-56, 58-71, 73-100, 109-114, the call/class/
//                             private-name model, iterators, the arguments object and
//                             the rest of the builtin surface (see its header)
//   tests/js-test-full.4.mjs  sections 57 and 72, the Module goal
//
// The section NUMBERS are global (grammar comments and docs cite them), the four
// files together run the assertions in order, and ./test.sh --full reports them as the
// units js, js2, js3 and js4 - one per FILE, because the check-count agreement rule
// compares the two GRAMMARS of one file and never two files of one language. Add a
// new section to whichever file it belongs in, and keep the numbers globally unique
// across the four.
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
    // Number::toString is the SHORTEST digits that read back, closest to the value
    // when several are that short; these doubles need all seventeen (node v24).
    var sd = [0.0015425378683206609, 1.7385704740834339e+92, -7.209885073274179e-254]
    check("tpl9", `${sd[0]}` === "0.0015425378683206609" && String(sd[1]) === "1.7385704740834339e+92" &&
                  "" + sd[2] === "-7.209885073274179e-254")

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
    // Exhausting the loop normally leaves it done as well, and runs the finally once.
    var b = g37()
    var total = 0
    for (var z of b) { total = total + z }
    check("itc5", total === 6 && b.next().done === true && fins === 2)
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
        [Symbol.iterator]: function () { return this },
        next: function () { this.n = this.n + 1; return { value: this.n, done: this.n > 5 } },
        return: function () { closed = closed + 1; return { value: undefined, done: true } }
    }
    for (var p of iter) { break }
    check("itc14", closed === 1)
    // An iterator WITHOUT a return member is left alone rather than aborting,
    // and an array (the index arm) never looks for one.
    var bare = { n: 0, [Symbol.iterator]: function () { return this },
        next: function () { this.n = this.n + 1; return { value: this.n, done: this.n > 3 } } }
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

    // OWN is not enumerable: node keeps `stack` and a given `message` on the instance
    // (getOwnPropertyNames lists them first) and `name` on the prototype until the
    // program writes it; `new Error()` assigns no own message at all.
    var hop = Object.prototype.hasOwnProperty
    var eo = new Error(xs[0])
    check("err22", !hop.call(eo, "name") && hop.call(eo, "message") && hop.call(eo, "stack"))
    check("err23", Object.getOwnPropertyNames(eo).join(",") === "stack,message"
        && Object.getOwnPropertyNames(new Error()).join(",") === "stack" && !hop.call(new Error(), "message"))
    check("err24", Object.getOwnPropertyNames(m2).join(",") === "stack,message,code"
        && hop.call(ew, "name") && Object.getOwnPropertyNames(ew).join(",") === "stack,message,q,name")
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
    var near1 = [0.99999, 0.9999999]
    check("math04b", Math.asin(near1[0]) === 1.5663241871131188 && Math.asin(near1[1]) === 1.5703491131957876)
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

    // ECMAScript's pow is not C99's: a NaN exponent is NaN even for base 1, and
    // (+-1) ** (+-Infinity) is NaN where C says 1. '**', '**=' and Math.pow (also as a
    // detached value) answer alike. Values read off node 24.
    var pw = [1, -1, NaN, Infinity, -Infinity, 0, 2]
    var pdet = Math.pow
    function n42(x) { return x !== x }
    check("math42", n42(Math.pow(pw[0], pw[2])) && n42(Math.pow(pw[0], pw[3])) && n42(Math.pow(pw[1], pw[4]))
        && n42(pdet(pw[1], pw[3])) && n42(pw[0] ** pw[2]) && n42(pw[1] ** pw[3]) && n42(pw[0] ** pw[4]))
    var pe = pw[0]
    pe **= pw[2]
    check("math43", n42(pe) && Math.pow(pw[2], pw[5]) === 1 && pw[1] ** pw[6] === 1 && Math.pow(pw[0], pw[6]) === 1)

    // pow is correctly rounded: a large y*log(x) keeps every digit, and x ** -0.5
    // is not 1/sqrt(x). Values read off node 24.
    var px = [1.9687173381259235, 2, 1.0000005275492378, 0.9303244752705773]
    var py = [596.2128067674018, -0.5, -48986194.85211566, 5961.696604822106]
    check("math47", px[0] ** py[0] === 2.488229202381568e+175 && Math.pow(px[0], py[0]) === 2.488229202381568e+175
        && px[1] ** py[1] === 0.7071067811865476)
    check("math48", Math.pow(px[2], py[2]) === 5.979866080580326e-12 && px[3] ** py[3] === 1.018795851248988e-187)

    // Math's members are NOT ENUMERABLE: for-in, Object.keys and JSON see none of
    // them, while a property the program adds is an ordinary one and a write over a
    // member keeps it hidden. Values read off node 24.
    var mk = 0
    for (var mx in Math) { mk = mk + 1 }
    check("math44", mk === 0 && Object.keys(Math).length === 0 && JSON.stringify(Math) === "{}" && ("abs" in Math))
    var flo = Math.floor
    Math.zz = a[0]
    Math.floor = Math.ceil
    var mks = ""
    for (var my in Math) { mks = mks + my }
    check("math45", mks === "zz" && Object.keys(Math).join(",") === "zz" && Math.floor(a[2]) === 1)
    Math.floor = flo
    delete Math.zz
    check("math46", Object.keys(Math).length === 0 && Math.floor(a[2]) === 0)
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

// ===== SECTION 101: the builtin globals' name and length, x.constructor, Object(1n), String wrapper keys =====
// Number, String, Array, BigInt, Promise, Symbol and RegExp carry node's own name and
// length. A value no class built answers its builtin global for `constructor` - 7 is
// a Number, [] an Array, a plain object and Object.prototype itself an Object - and a
// null-prototype object answers nothing. Object(v) wraps a BigInt and a symbol as it
// wraps a number, and a String wrapper owns its string's indices and `length`, which
// Object.keys, for-in, `in`, hasOwnProperty and getOwnPropertyNames list before its
// other own properties. Values read off node 24.
function s101() {
    const gs = [Number, String, Array, BigInt, Promise, Symbol, RegExp]
    check("nm101a", gs.map((g) => g.name + "/" + g.length).join() === "Number/1,String/1,Array/1,BigInt/1,Promise/1,Symbol/0,RegExp/2")
    const vs = [7, "s", true, [1], {}, 2n, 1.5]
    check("ct101b", vs[0].constructor === Number && vs[1].constructor === String && vs[2].constructor === Boolean &&
        vs[3].constructor === Array && vs[4].constructor === Object && vs[5].constructor === BigInt &&
        vs.map((v) => v.constructor.name).join() === "Number,String,Boolean,Array,Object,BigInt,Number")
    check("ct101c", Object.prototype.constructor === Object && Array.prototype.constructor === Array &&
        Number.prototype.constructor === Number && String.prototype.constructor === String &&
        (function () {}).constructor === Function && new Number(vs[0]).constructor === Number &&
        Object.create(null).constructor === undefined)
    const b = Object(vs[5]), sy = Object(Symbol("q"))
    check("wr101d", typeof b === "object" && b + 1n === 3n && b instanceof BigInt && typeof b.valueOf() === "bigint" &&
        b.toString() === "2" && b.constructor === BigInt && typeof sy === "object" && sy.description === "q" &&
        sy.toString() === "Symbol(q)" && typeof sy.valueOf() === "symbol" && sy instanceof Symbol && !(vs[5] instanceof BigInt))
    const sw = new String("abc"), ks = []
    for (const k in sw) ks.push(k)
    sw.extra = 5
    check("sw101e", Object.keys(sw).join() === "0,1,2,extra" && ks.join() === "0,1,2" && sw[1] === "b" && sw["2"] === "c" &&
        sw.hasOwnProperty("1") && !sw.hasOwnProperty("3") && "0" in sw && !("5" in sw) &&
        Object.getOwnPropertyNames(sw).join() === "0,1,2,length,extra" &&
        JSON.stringify(Object.entries(new String("hi"))) === '[["0","h"],["1","i"]]' && Object.values(Object("xy")).join() === "x,y")
}

// ===== SECTION 102: Object.freeze, seal and preventExtensions =====
// preventExtensions stops new properties, seal also stops deletes, freeze also stops
// writes, each shallow and each answering its argument; isFrozen / isSealed /
// isExtensible test the level, a primitive being frozen. Sloppy code drops a refused
// write or delete; strict code - a "use strict" body, a class body, an Array method,
// Object.assign - raises node's TypeError. An array keeps the same three levels, and a
// sealed array's shift moves its elements down before it fails. Values read off node 24.
function s102() {
    const vals = [1, 2, 3]
    const fz = Object.freeze({ a: vals[0], n: { m: vals[0] } })
    fz.a = vals[1]
    fz.b = vals[1]
    delete fz.a
    fz.n.m = vals[2]
    fz.a += 5
    check("fz102a", fz.a === 1 && fz.b === undefined && fz.n.m === 3 && Object.isFrozen(fz) && Object.isSealed(fz) &&
        !Object.isExtensible(fz) && !Object.isFrozen(fz.n) && Object.freeze(fz) === fz && Object.keys(fz).join() === "a,n")
    const sl = Object.seal({ x: vals[0] })
    sl.x = vals[1]
    sl.y = vals[2]
    delete sl.x
    const pe = Object.preventExtensions({ q: vals[0] })
    pe.q = vals[2]
    pe.r = vals[1]
    const peq = pe.q
    delete pe.q
    check("sl102b", sl.x === 2 && sl.y === undefined && Object.isSealed(sl) && !Object.isFrozen(sl) && peq === 3 &&
        pe.q === undefined && pe.r === undefined && !Object.isExtensible(pe) && Object.isFrozen(pe) &&
        !Object.isFrozen({}) && Object.isExtensible({}) && Object.isFrozen(vals[0]) && !Object.isExtensible("s") &&
        Object.isFrozen(Object.freeze({ get g() { return 2 } })))
    const err = (f) => { try { f(); return "ok" } catch (e) { return e instanceof TypeError ? e.message : "?" } }
    check("st102c", err(function () { "use strict"; fz.a = 2 }) === "Cannot assign to read only property 'a' of object '#<Object>'" &&
        err(function () { "use strict"; fz.z = 2 }) === "Cannot add property z, object is not extensible" &&
        err(function () { "use strict"; delete sl.x }) === "Cannot delete property 'x' of #<Object>" &&
        err(function () { "use strict"; fz.a++ }) === "Cannot assign to read only property 'a' of object '#<Object>'" &&
        err(function () { "use strict"; [fz.a] = [5] }) === "Cannot assign to read only property 'a' of object '#<Object>'" &&
        err(function () { "use strict"; sl.x = 4 }) === "ok" && sl.x === 4 && err(() => { fz.a = 2 }) === "ok")
    class P102 { constructor() { this.x = vals[0]; Object.freeze(this) } bump() { this.x = vals[1] } }
    check("cl102d", err(() => new P102().bump()) === "Cannot assign to read only property 'x' of object '#<P102>'")
    const fa = Object.freeze([3, 1, 2]), sa = Object.seal([3, 1, 2]), na = Object.preventExtensions([3, 1, 2])
    fa[0] = 9
    fa[5] = 9
    fa.length = 1
    sa[0] = 5
    sa[3] = 9
    na[1] = 7
    na[4] = 4
    check("ar102e", fa.join() === "3,1,2" && sa.join() === "5,1,2" && na.join() === "3,7,2" && Object.isFrozen(fa) &&
        Object.isSealed(sa) && !Object.isFrozen(sa) && !Object.isExtensible(na) && !Object.isSealed(na) &&
        !(delete fa[0]) && !(delete sa[1]) && delete na[0] && na[0] === undefined)
    check("ar102f", err(() => fa.push(4)) === "Cannot add property 3, object is not extensible" &&
        err(() => fa.pop()) === "Cannot delete property '2' of [object Array]" &&
        err(() => fa.sort()) === "Cannot assign to read only property '0' of object '[object Array]'" &&
        err(() => sa.unshift(0)) === "Cannot add property 3, object is not extensible" &&
        err(() => na.push(1)) === "Cannot add property 3, object is not extensible" &&
        err(() => sa.reverse()) === "ok" && sa.join() === "2,1,5" && na.pop() === 2 && fa.map((x) => x * 2).join() === "6,2,4" &&
        err(function () { "use strict"; fa[0] = 1 }) === "Cannot assign to read only property '0' of object '[object Array]'" &&
        err(function () { "use strict"; delete sa[0] }) === "Cannot delete property '0' of [object Array]")
    const ss = Object.seal([3, 1, 2])
    check("ar102g", err(() => ss.shift()) === "Cannot delete property '2' of [object Array]" && ss.join() === "1,2,2")
    check("df102h", err(() => Object.defineProperty(fz, "z", { value: 1 })) === "Cannot define property z, object is not extensible" &&
        err(() => Object.assign(fz, { a: 5 })) === "Cannot assign to read only property 'a' of object '#<Object>'" &&
        err(() => Object.setPrototypeOf(sl, {})) === "#<Object> is not extensible" &&
        Object.defineProperty(sl, "x", { value: 6 }).x === 6)
    const fn = Object.freeze(function f102() {})
    fn.x = 1
    check("fn102i", fn.x === undefined && Object.isFrozen(fn) && !Object.isFrozen(function () {}))
}

// ===== SECTION 103: a class's length, builtin function values' name and length, strict arrows, the first refused write =====
// A class's `length` is its constructor's (0 for an implicit one) and every builtin
// Error constructor's is 1, Map's and Set's 0; an anonymous class expression is named
// after the binding it is assigned to, or "". Math and JSON members and the Object
// statics read as values carry node's name and length. An arrow or a generator whose
// own body opens with "use strict" is strict code: a refused write throws, and a plain
// `=` to a name no scope binds is a ReferenceError instead of an implicit global. A
// method on a frozen, sealed or non-extensible array fails at the spec's FIRST refused
// Set or delete, after the writes the level allows. Values read off node 24.
function s103() {
    class A103 { constructor(a, b) {} }
    class B103 extends A103 {}
    class D103 extends A103 { constructor(x, ...r) { super(x, 1) } }
    class E103 { constructor(a, b = 1, c) {} }
    check("cl103a", [A103, B103, D103, E103, class {}].map((c) => c.length).join() === "2,0,1,1,0" &&
        [Error, TypeError, RangeError, SyntaxError, ReferenceError, Map, Set].map((c) => c.length).join() === "1,1,1,1,1,0,0")
    const G103 = class {}
    let H103
    H103 = class extends G103 {}
    const o103 = { k: class {} }
    check("cl103g", G103.name === "G103" && H103.name === "H103" && o103.k.name === "k" && (class {}).name === "" &&
        (class N103 {}).name === "N103" && [class {}][0].name === "")
    const mk = ["max", "min", "floor", "atan2", "random", "hypot", "pow"]
    check("bi103b", mk.map((k) => Math[k].name + "/" + Math[k].length).join() === "max/2,min/2,floor/1,atan2/2,random/0,hypot/2,pow/2" &&
        JSON.stringify.name + "/" + JSON.stringify.length + " " + JSON.parse.name + "/" + JSON.parse.length === "stringify/3 parse/2")
    const ok = ["preventExtensions", "isFrozen", "isSealed", "isExtensible", "getOwnPropertyDescriptor"]
    const pe = Object.preventExtensions
    check("bi103c", ok.map((k) => typeof Object[k] + "/" + Object[k].name + "/" + Object[k].length).join() ===
        "function/preventExtensions/1,function/isFrozen/1,function/isSealed/1,function/isExtensible/1,function/getOwnPropertyDescriptor/2" &&
        !Object.isExtensible(pe({})))
    const fr = Object.freeze({ a: 1 })
    const err = (f) => { try { f(); return "ok" } catch (e) { return e.constructor.name + ": " + e.message } }
    check("st103d", err(() => { "use strict"; fr.a = 2 }) === "TypeError: Cannot assign to read only property 'a' of object '#<Object>'" &&
        err(() => { fr.a = 2 }) === "ok" && err(() => { "use strict"; strictGlobal103 = 1 }) === "ReferenceError: strictGlobal103 is not defined" &&
        typeof strictGlobal103 === "undefined" && err(() => { const g = function* () { "use strict"; fr.a = 3; yield 1 }; g().next() }) ===
        "TypeError: Cannot assign to read only property 'a' of object '#<Object>'")
    const ops = [(a) => a.fill(9, 1), (a) => a.fill(9, 4), (a) => a.copyWithin(1, 0), (a) => a.copyWithin(2, 0),
        (a) => a.splice(1, 0, 7), (a) => a.splice(0, 2, 9), (a) => a.splice(2, 2, 7), (a) => a.splice(1, 0),
        (a) => a.unshift(7, 8), (a) => a.push(), (a) => a.sort()]
    check("ar103e", ops.map((f) => err(() => f(Object.freeze([4, 3, 2, 1])))).join("|") ===
        "TypeError: Cannot assign to read only property '1' of object '[object Array]'|ok|" +
        "TypeError: Cannot assign to read only property '3' of object '[object Array]'|" +
        "TypeError: Cannot assign to read only property '2' of object '[object Array]'|" +
        "TypeError: Cannot add property 4, object is not extensible|" +
        "TypeError: Cannot assign to read only property '1' of object '[object Array]'|" +
        "TypeError: Cannot delete property '3' of [object Array]|" +
        "TypeError: Cannot assign to read only property 'length' of object '[object Array]'|" +
        "TypeError: Cannot add property 5, object is not extensible|" +
        "TypeError: Cannot assign to read only property 'length' of object '[object Array]'|" +
        "TypeError: Cannot assign to read only property '0' of object '[object Array]'")
    const sa = Object.seal([4, 3, 2, 1]), sb = Object.seal([4, 3, 2, 1]), na = Object.preventExtensions([4, 3, 2, 1])
    check("ar103f", err(() => sa.splice(0, 2, 9)) === "TypeError: Cannot delete property '3' of [object Array]" && sa.join() === "4,2,1,1" &&
        err(() => sb.splice(2, 2, 7, 8, 9)) === "TypeError: Cannot add property 4, object is not extensible" && sb.join() === "4,3,7,8" &&
        err(() => na.splice(1, 1, 7, 8)) === "TypeError: Cannot add property 4, object is not extensible" &&
        err(() => Object.freeze([5]).sort()) === "ok" && err(() => Object.freeze([5]).reverse()) === "ok" &&
        [1, 2, 3].fill(0, 0, null).join() === "1,2,3")
}

// ===== SECTION 104: a write to a function's name or length =====
// A function's and a class's own `name` and `length` are non-writable: sloppy code
// drops the write, strict code (and a class body, which is always strict) throws
// node's TypeError. V8 renders the function's source text after "of function", which
// is only asserted up to there. A static member of that name replaces the class's own
// property and is writable, and Object.defineProperty may replace a class's own one.
function s104() {
    function f104(a, b) { return a + b }
    class K104 { static w() { K104.length = 5 } }
    class N104 { static name = "own" }
    const err = (f) => { try { f(); return "ok" } catch (e) { return e.constructor.name + ": " + e.message } }
    const ro = "TypeError: Cannot assign to read only property '"
    const put = (o, k, v) => { "use strict"; o[k] = v }
    check("fn104a", err(() => { f104.name = "z"; f104.length = 9 }) === "ok" && f104.name === "f104" && f104.length === 2 &&
        err(() => { K104.name = "z" }) === "ok" && K104.name === "K104")
    const got = [[f104, "name"], [f104, "length"], [(x) => x, "name"], [Math.max, "length"], [K104, "name"],
        [f104.bind(null), "name"]].map((p) => err(() => put(p[0], p[1], 1)))
    check("fn104b", got.every((m, i) => m.indexOf(ro + (i == 1 || i == 3 ? "length" : "name") + "' of function '") === 0) &&
        f104.name === "f104" && f104.length === 2 && K104.name === "K104")
    check("fn104c", err(() => K104.w()).indexOf(ro + "length' of function '") === 0 && K104.length === 0 &&
        err(() => put(N104, "name", "z")) === "ok" && N104.name === "z" && err(() => put(f104, "q", 1)) === "ok" && f104.q === 1)
    Object.defineProperty(K104, "length", { value: 7 })
    check("fn104d", K104.length === 7)
}

// ===== SECTION 105: numeric literal forms =====
// A numeric separator stands between two digits of any run: the integer part, the
// fraction, the exponent and every radix ('0x1_0', '0.0_1', '1e1_0'). A legacy
// NonOctalDecimalIntegerLiteral ('08', '09') takes a fraction and an exponent like any
// decimal integer part, in sloppy code. A radix literal past 2^53 is rounded once, half
// to even: 0x200000000000011 is 2^57 + 17, nearer to 2^57 + 32 than to 2^57.
function s105() {
    check("nl105a", 0x1_0 === 16 && 0X1_F === 31 && 0xA_b_C === 2748 && 0b1_0 === 2 && 0o1_7 === 15 && 0x1_fn === 31n)
    check("nl105b", 0.0_1 === 0.01 && .0_1 === 0.01 && 0.1_2_3 === 0.123 && 1_0.0_1 === 10.01 && 1e1_0 === 1e10 &&
        1E+1_0 === 1e10 && 1e-1_0 === 1e-10 && .5e1_0 === 5e9 && 1.0_1e1_0 === 101e8 && 1_1.1_1e1_1 === 1111e9)
    check("nl105c", 08e1 === 80 && 09.5e1 === 95 && 09e-1 === 0.9 && 08.e1 === 80 && 08.1_2 === 8.12 && 08e1_0 === 8e10 &&
        0787 === 787 && 08. === 8)
    check("nl105d", 0x200000000000011 === 144115188075855900 && 0x200000000000019 === 144115188075855900 &&
        0o4000000000000000001 === 72057594037927940 && 0x20000000000001 === 9007199254740992)
}

// ===== SECTION 106: a function's name and length are configurable =====
// Object.defineProperty replaces a function's own `name` or `length` (non-writable
// unless the descriptor says writable), and `delete` removes it: the read then finds
// Function.prototype's ("" and 0), the property is no longer own, and a write is still
// refused by that inherited read-only one.
function s106() {
    const err = (f) => { try { f(); return "ok" } catch (e) { return e.constructor.name } }
    const put = (o, k, v) => { "use strict"; o[k] = v }
    function f106(a, b) { return a }
    Object.defineProperty(f106, "name", { value: "renamed" })
    check("fc106a", f106.name === "renamed" && f106.length === 2 && err(() => put(f106, "name", "z")) === "TypeError" &&
        f106.name === "renamed")
    Object.defineProperty(f106, "length", { value: 7, writable: true })
    check("fc106b", f106.length === 7 && err(() => put(f106, "length", 9)) === "ok" && f106.length === 9 &&
        f106.hasOwnProperty("name") && Object.getOwnPropertyNames(f106).indexOf("length") >= 0)
    function g106(a, b, c) { }
    check("fc106c", delete g106.length && g106.length === 0 && !g106.hasOwnProperty("length") &&
        Object.getOwnPropertyNames(g106).indexOf("length") < 0 && Object.getOwnPropertyNames(g106).indexOf("name") >= 0)
    check("fc106d", delete g106.name && g106.name === "" && !g106.hasOwnProperty("name") &&
        err(() => put(g106, "length", 4)) === "TypeError" && g106.length === 0)
    Object.defineProperty(g106, "length", { value: 2 })
    const h106 = (x) => x
    Object.defineProperty(h106, "name", { value: "hh", writable: true })
    h106.name = "h2"
    check("fc106e", g106.length === 2 && g106.hasOwnProperty("length") && h106.name === "h2" && Object.keys(h106).length === 0)
}

// ===== SECTION 107: a function's source text =====
// String(f), f.toString(), `${f}`, "" + f and f == "..." all read the source text of
// f's definition as written, comments and line breaks included (ES2023 20.2.3.5): a
// method's starts at its key (a `static` is left out), a class's is the whole class,
// and a builtin or a bound function is V8's native code. Inside the TypeError of a
// refused write, a text past 128 units keeps its first 111 and its last 2.
class P107 { static make() { return new this() } }
class K107 extends P107 {
    async am() { await 1 }
    *gm() { yield 1 }
    static sm(q) { return q }
    ["comp" + "uted"]() { }
}
function s107() {
    function f107(a, b = 2) {
        return a + b // a comment, é
    }
    const ar = (x) => x * 2
    const one = x => x
    const o = { m(a) { return a }, async *agm() { }, ["c" + "k"]() { } }
    const err = (f) => { try { f(); return "ok" } catch (e) { return e.message } }
    check("fs107a", String(f107) === "function f107(a, b = 2) {\n        return a + b // a comment, é\n    }" &&
        f107.toString() === String(f107) && String(ar) === "(x) => x * 2" && "" + one === "x => x" &&
        `${one}` === "x => x" && one == "x => x" && [one, ar].join("|") === "x => x|(x) => x * 2")
    check("fs107b", String(P107) === "class P107 { static make() { return new this() } }" &&
        String(P107.make) === "make() { return new this() }" && String(K107.sm) === "sm(q) { return q }" &&
        String(new K107().am) === "async am() { await 1 }" && String(new K107().gm) === "*gm() { yield 1 }" &&
        String(new K107().computed) === '["comp" + "uted"]() { }' && String(K107).indexOf("class K107 extends P107 {\n") === 0)
    check("fs107c", String(o.m) === "m(a) { return a }" && String(o.agm) === "async *agm() { }" &&
        String(o.ck) === '["c" + "k"]() { }' && String(class { }) === "class { }")
    check("fs107d", String(Math.max) === "function max() { [native code] }" &&
        String(one.bind(null)) === "function () { [native code] }" && String(Error) === "function Error() { [native code] }")
    check("fs107e", err(() => { "use strict"; one.name = "z" }) === "Cannot assign to read only property 'name' of function 'x => x'" &&
        err(() => { "use strict"; K107.name = "z" }) === "Cannot assign to read only property 'name' of function '" +
            String(K107).substring(0, 111) + "...<omitted>...\n}'")
}

// ===== SECTION 108: property descriptors, builtin methods' names, deleting a class's name and length =====
// Object.getOwnPropertyDescriptor answers the attributes the value model keeps: an
// object's (defineProperty's, the integrity level's), and the fixed ones of the
// intrinsic properties - a function's and a class's name and length are configurable
// only, a class's prototype and an array's length are not configurable, a string's
// indices are read-only, and a class body's members are writable and configurable but
// not enumerable. A builtin method read as a value carries its own name and length,
// which String(f) shows. `delete` removes a class's own name or length, and the read
// then finds the parent class's, or Function.prototype's; it removes a static member
// the same way, and never the instance method of that name. Values read off node 24.
class P108 { constructor(a, b) {} }
class K108 extends P108 {}
class E108 {}
class T108 { static z = 1; static m() {} static k = 2; m() { return "inst" } }
class W108 { static z = 1 }
class V108 extends W108 { static z = 9 }
class S108 { static name = "custom" }
class A108 { static get g() { return "g" } }
class C108 {
    static sf = 3
    static sm() {}
    static get sg() { return 1 }
    f = 1
    m() {}
    get g() { return 2 }
    set g(v) {}
}
function s108() {
    const show = (d) => d === undefined ? "-" : Object.keys(d).map((k) => k + "=" +
        (typeof d[k] === "function" ? "fn " + d[k].name : String(d[k]))).join(",")
    const G = Object.getOwnPropertyDescriptor
    const o = { x: 1, get g() { return 2 }, set g(v) {} }
    check("od108a", show(G(o, "x")) === "value=1,writable=true,enumerable=true,configurable=true" &&
        show(G(o, "g")) === "get=fn get g,set=fn set g,enumerable=true,configurable=true" && G(o, "no") === undefined &&
        o.hasOwnProperty("g") && G(5, "x") === undefined && G("ab", "1").value === "b")
    const d = Object.defineProperty({}, "h", { value: 1, writable: true })
    Object.defineProperty(d, "q", { get: function qq() { return 1 }, configurable: true })
    check("od108b", show(G(d, "h")) === "value=1,writable=true,enumerable=false,configurable=false" &&
        show(G(d, "q")) === "get=fn qq,set=undefined,enumerable=false,configurable=true" &&
        show(G(Object.freeze({ a: 1 }), "a")) === "value=1,writable=false,enumerable=true,configurable=false" &&
        show(G(Object.seal({ a: 1 }), "a")) === "value=1,writable=true,enumerable=true,configurable=false" &&
        show(G(Object.freeze([7]), "0")) + " " + show(G(Object.seal([7]), "length")) ===
        "value=7,writable=false,enumerable=true,configurable=false value=1,writable=true,enumerable=false,configurable=false" &&
        show(G([5], "length")) === "value=1,writable=true,enumerable=false,configurable=false")
    function f108(a) {}
    f108.extra = 5
    const c = new C108()
    check("od108c", show(G(f108, "name")) === "value=f108,writable=false,enumerable=false,configurable=true" &&
        show(G(f108, "extra")) === "value=5,writable=true,enumerable=true,configurable=true" &&
        show(G(C108, "length")) === "value=0,writable=false,enumerable=false,configurable=true" &&
        show(G(C108, "prototype")).indexOf(",writable=false,enumerable=false,configurable=false") > 0 &&
        show(G(C108, "sf")) + " " + show(G(C108, "sm")) + " " + show(G(C108, "sg")) ===
        "value=3,writable=true,enumerable=true,configurable=true value=fn sm,writable=true,enumerable=false,configurable=true " +
        "get=fn get sg,set=undefined,enumerable=false,configurable=true" &&
        show(G(C108.prototype, "m")) + " " + show(G(C108.prototype, "g")) ===
        "value=fn m,writable=true,enumerable=false,configurable=true get=fn get g,set=fn set g,enumerable=false,configurable=true" &&
        show(G(c, "f")) === "value=1,writable=true,enumerable=true,configurable=true" && G(c, "m") === undefined)
    const sym = Symbol("t")
    const so = { 1: "one" }
    so[sym] = 9
    const err = (f) => { try { f(); return "ok" } catch (e) { return e.message } }
    check("od108d", show(G(so, sym)) === "value=9,writable=true,enumerable=true,configurable=true" && G(so, 1).value === "one" &&
        show(G(new String("hi"), "0")) === "value=h,writable=false,enumerable=true,configurable=false" &&
        err(() => G(undefined, "x")) === "Cannot convert undefined or null to object" &&
        err(() => Object.getOwnPropertyDescriptors(null)) === "Cannot convert undefined or null to object" &&
        Object.getOwnPropertyDescriptor.length === 2 && Object.getOwnPropertyDescriptors.length === 1)
    const ds = Object.getOwnPropertyDescriptors({ a: 1, get b() { return 2 } })
    const da = Object.getOwnPropertyDescriptors([5])
    check("od108e", Object.keys(ds).join() === "a,b" && show(ds.b) === "get=fn get b,set=undefined,enumerable=true,configurable=true" &&
        Object.keys(da).join() === "0,length" && da.length.writable === true &&
        Object.keys(Object.getOwnPropertyDescriptors(class Q { static z() {} })).join() === "length,name,prototype,z")
    const f = function (x) { return this.v + x }
    const pu = [].push
    const arr = [1]
    pu.call(arr, 2, 3)
    check("bn108f", [[].push, "".slice, [].map, (5).toFixed, (5).toString, true.toString, "".trimEnd, [].splice]
        .map((m) => m.name + "/" + m.length).join() ===
        "push/1,slice/2,map/1,toFixed/1,toString/1,toString/0,trimEnd/0,splice/2" &&
        String([].push) === "function push() { [native code] }" && String("".slice) === "function slice() { [native code] }" &&
        arr.join() === "1,2,3" && ({}).hasOwnProperty.name === "hasOwnProperty" && ({}).isPrototypeOf.length === 1 &&
        [f.call, f.apply, f.bind].map((m) => m.name + "/" + m.length).join() === "call/1,apply/2,bind/1" &&
        f.call.call(f, { v: 1 }, 2) === 3 && f.hasOwnProperty.name === "hasOwnProperty" && f.toString.name === "toString")
    check("dc108g", delete K108.name && delete K108.length && K108.name === "P108" && K108.length === 2 &&
        !K108.hasOwnProperty("name") && Object.getOwnPropertyNames(K108).join() === "prototype" && G(K108, "name") === undefined &&
        delete E108.name && E108.name === "" && delete E108.length && E108.length === 0 && String(E108) === "class E108 {}" &&
        err(() => { "use strict"; E108.name = "z" }).indexOf("Cannot assign to read only property 'name'") === 0)
    check("dc108h", delete T108.z && delete T108.m && T108.z === undefined && T108.m === undefined &&
        Object.keys(T108).join() === "k" && !T108.hasOwnProperty("z") && !("m" in T108) && new T108().m() === "inst" &&
        Object.getOwnPropertyNames(T108).join() === "length,name,prototype,k" && delete S108.name && S108.name === "" &&
        !S108.hasOwnProperty("name") && Object.getOwnPropertyNames(S108).join() === "length,prototype")
    T108.z = 5
    check("dc108i", Object.keys(T108).join() === "k,z" && T108.z === 5 && delete V108.z && V108.z === 1 &&
        delete V108.z && V108.z === 1 && delete A108.g && A108.g === undefined && G(A108, "g") === undefined &&
        (A108.g = 3) === 3 && A108.g === 3 && Object.keys(A108).join() === "g")
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
    s101() // SECTION-CALL 101
    s102() // SECTION-CALL 102
    s103() // SECTION-CALL 103
    s104() // SECTION-CALL 104
    s105() // SECTION-CALL 105
    s106() // SECTION-CALL 106
    s107() // SECTION-CALL 107
    s108() // SECTION-CALL 108
    println("full: " + checks + " checks, " + failures + " failures")
    return failures
}
