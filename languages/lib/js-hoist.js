// js-hoist.js -- which scope instantiates each function declaration, and which block
// functions get an Annex B binding. Included by the js and typescript grammars, both
// halves (include("lib/js-hoist.js")), in the restricted subset of regex.js's head
// comment. Every name is prefixed hoist.
//
// A function declaration is instantiated when its scope is entered, before any
// statement runs: the function body's, the program's, or - for one directly in a
// block, a switch or an if body - that statement's (FunctionDeclarationInstantiation,
// BlockDeclarationInstantiation, Annex B.3.4). Each declaration records itself with
// its source position, and the innermost enclosing block, switch, function body or
// program claims it by span, the way the grammars claim var names (claimVars): the
// inner tags run first, so what is left in a span is exactly what it directly holds.
//
// Annex B.3.2 (sloppy code only) gives a plain function declared in a block a second,
// function-scoped binding: hoisted as undefined, and assigned the block's binding
// when the declaration is evaluated in place. It does not apply when a let, const or
// class of the name in an enclosing block, loop head or the function body would make
// `var F` an early error, nor when F names a parameter (the grammars decide that
// one). A block function of the same name in an enclosing block does not cancel it,
// which is how node behaves.

// Declarations not yet claimed: {name, at, file, plain, ab, ...} where plain says a
// plain function (not async, not a generator) and ab that its declaration performs
// the Annex B assignment. The grammar adds what instantiating it takes.
var hoistFns = []
var hoistLex = []   // {name, at, file, cst} of each let, const and class name; cst marks a const.
var hoistAnnex = [] // Block functions whose function body has not claimed them yet.

function hoistNoteLex(name, pos, cst) { hoistLex.push({name: name, at: pos, file: c.curFile(), cst: cst === true}) }

function hoistSplit(recs, from, to) {
    var file = c.curFile()
    var got = []
    var keep = []
    for (var i = 0; i < recs.length; i++) {
        var r = recs[i]
        if (r.file === file && r.at >= from && r.at <= to) { got.push(r) } else { keep.push(r) }
    }
    return {got: got, keep: keep}
}

function hoistClaimFns(from, to) {
    var sp = hoistSplit(hoistFns, from, to)
    hoistFns = sp.keep
    return sp.got
}

// The lexical names declared directly in a span and not claimed yet, without claiming
// them: at a `for` statement's own tag that is what its head declares, the names a
// per-iteration environment copies (CreatePerIterationEnvironment). constOnly keeps
// the const ones.
function hoistLexNames(from, to, constOnly) {
    var got = hoistSplit(hoistLex, from, to).got
    var names = []
    for (var i = 0; i < got.length; i++) {
        if (constOnly === true && !got[i].cst) { continue }
        if (names.indexOf(got[i].name) < 0) { names.push(got[i].name) }
    }
    return names
}

// The lexical names declared directly in a span cancel the Annex B binding of every
// block function inside it of the same name.
function hoistLexKill(from, to) {
    var sp = hoistSplit(hoistLex, from, to)
    hoistLex = sp.keep
    if (sp.got.length == 0) { return }
    var names = []
    for (var i = 0; i < sp.got.length; i++) { names.push(sp.got[i].name) }
    var keep = []
    var file = c.curFile()
    for (var j = 0; j < hoistAnnex.length; j++) {
        var r = hoistAnnex[j]
        if (r.file === file && r.at >= from && r.at <= to && names.indexOf(r.name) >= 0) { r.ab = false } else { keep.push(r) }
    }
    hoistAnnex = keep
}

// The Annex B candidates a function body (or the program) takes over.
function hoistClaimAnnex(from, to) {
    var sp = hoistSplit(hoistAnnex, from, to)
    hoistAnnex = sp.keep
    return sp.got
}

// A block, a switch or an if body claims its functions; the plain ones become Annex B
// candidates of the enclosing function.
function hoistBlockFns(from, to) {
    var fns = hoistClaimFns(from, to)
    hoistLexKill(from, to)
    for (var i = 0; i < fns.length; i++) {
        if (fns[i].plain) { fns[i].ab = true; hoistAnnex.push(fns[i]) }
    }
    return fns
}

// The distinct names of a list of records.
function hoistNames(recs) {
    var names = []
    for (var i = 0; i < recs.length; i++) {
        if (names.indexOf(recs[i].name) < 0) { names.push(recs[i].name) }
    }
    return names
}
