// js-genlog.js -- the PULL LOG of the replay generators in the js and typescript
// INTERPRETERS (include("lib/js-genlog.js") from their startScript). Every name is
// prefixed gl. The compiler halves run a generator as a real coroutine and need none
// of this.
//
// A replayed body is only sound if every value it reads comes out the same on every
// run, and a ONE-SHOT source does not: `for (const x of it) yield x` over an iterator
// made outside the generator would pull it again on every replay and skip elements.
// So a pull - a next() call - on an iterator the replay did NOT create is answered
// from the generator's log when an earlier run already made it, and made live (and
// logged) otherwise.
//
// "Created by the run" is a serial: an object that can be an iterator (an object
// literal, a constructed instance, a generator, a built-in iterator) made while a
// replay is active is stamped (__genborn), each run records the serial it started at
// (floor), and an object with no stamp was made outside every run. A pull is logged by
// EVERY active run it is external to, because an inner generator the outer one created
// is re-created by the outer replay and must find the same answers in the outer log.
//
// A run is {log, lp, floor, ...}: log is its generator's list of entries, kept across
// runs, and lp the position this run has read up to. An entry is {raised, v, end}:
// what the pull answered or raised, and the log position after it - a live pull may
// itself pull (a hand-written next() reading another external iterator), and those
// inner entries follow the outer one, so a replay answering the outer pull from the
// log skips them with it.

var glSerial = 0

function glStamp(o) {
    glSerial = glSerial + 1
    o["__genborn"] = glSerial
}

// The outermost active run o is external to, or -1 when no run is (o was made inside
// all of them, or it is not an object).
function glExtRun(runs, o) {
    if (o == undefined || o == null || typeof o != "object") { return 0 - 1 }
    var born = 0
    if (o["__genborn"] !== undefined) { born = o["__genborn"] }
    for (var i = 0; i < runs.length; i++) {
        if (born <= runs[i].floor) { return i }
    }
    return 0 - 1
}

// One logged pull for the runs from index i in: the entry a run already holds at
// this position, else live() - its value or what it raised.
function glPull(runs, i, live) {
    var hit = anytype
    hit = null
    var j
    for (j = i; j < runs.length; j++) {
        if (hit == null && runs[j].lp < runs[j].log.length) { hit = runs[j].log[runs[j].lp] }
    }
    if (hit == null) {
        // Each run reserves its slot BEFORE the live call, so the pulls live() makes
        // land after it.
        var slots = []
        for (j = i; j < runs.length; j++) {
            var e = {raised: false, v: undefined, end: 0}
            runs[j].log.push(e)
            runs[j].lp = runs[j].lp + 1
            slots.push(e)
        }
        hit = {raised: false, v: undefined, end: 0}
        try { hit.v = live() }
        catch (ex) { hit.raised = true; hit.v = ex }
        for (j = i; j < runs.length; j++) {
            slots[j - i].raised = hit.raised
            slots[j - i].v = hit.v
            slots[j - i].end = runs[j].lp
        }
    } else {
        for (j = i; j < runs.length; j++) {
            var rn = runs[j]
            if (rn.lp < rn.log.length) { rn.lp = rn.log[rn.lp].end }
            else {
                rn.log.push({raised: hit.raised, v: hit.v, end: rn.lp + 1})
                rn.lp = rn.lp + 1
            }
        }
    }
    if (hit.raised) { throw hit.v }
    return hit.v
}
