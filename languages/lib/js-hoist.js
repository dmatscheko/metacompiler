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

// The temporal dead zone. A let, const or class binding exists from its scope's entry
// but is uninitialized until its declaration runs, and reading, writing or typeof-ing
// it before then is a ReferenceError (ECMA-262 9.1.1.1). A compiler records each
// reference to a name (hoistNoteRef), and the scope that declares the name claims the
// references in its span, inner scopes first like the declarations. A reference can
// run while its binding is uninitialized only when it stands textually before the end
// of the declaration, inside a function declaration of that scope (instantiated at
// its entry), or anywhere in a switch's case block (a case label jumps past the
// declaration); every other one runs after the declaration has. Those references get
// chk, and their names are created uninitialized at scope entry, so a scope with none
// keeps its cheap entry. A reference claimed by a scope whose name it in fact shadows
// inside (a parameter, a var) only gets a check that never fires.
// hoistNoteRef answers the reference's index in hoistRefAll: a value pushed on the tag
// stack is deep-copied, so a target reference holds the index, not the record.
var hoistRefs = {}     // "$" + name -> the references to it no scope has claimed, {at, file, chk}
var hoistRefAll = []   // Every reference recorded, by index.
var hoistTdzCount = 0  // How many bindings some scope creates uninitialized.

function hoistNoteLex(name, pos, cst) { hoistLex.push({name: name, at: pos, file: c.curFile(), cst: cst === true}) }

function hoistNoteRef(name, pos) {
    var rec = {at: pos, file: c.curFile(), chk: false}
    var k = "$" + name
    if (hoistRefs[k] == undefined) { hoistRefs[k] = [] }
    hoistRefs[k].push(rec)
    hoistRefAll.push(rec)
    return hoistRefAll.length - 1
}

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
// block function inside it of the same name. Answers {names, tdz}: the distinct names,
// and those the scope creates uninitialized; fns are the function declarations it
// instantiates and sw says it is a switch's case block.
function hoistLexKill(from, to, fns, sw) {
    var sp = hoistSplit(hoistLex, from, to)
    hoistLex = sp.keep
    var out = {names: [], tdz: []}
    if (sp.got.length == 0) { return out }
    var names = out.names
    for (var i = 0; i < sp.got.length; i++) {
        var nm = sp.got[i].name
        if (names.indexOf(nm) >= 0) { continue }
        names.push(nm)
        if (hoistTdzRefs(sp.got[i], from, to, fns, sw === true)) { out.tdz.push(nm) }
    }
    var keep = []
    var file = c.curFile()
    for (var j = 0; j < hoistAnnex.length; j++) {
        var r = hoistAnnex[j]
        if (r.file === file && r.at >= from && r.at <= to && names.indexOf(r.name) >= 0) { r.ab = false } else { keep.push(r) }
    }
    hoistAnnex = keep
    return out
}

// Claims the references to the declaration r in a span, marking the ones that can
// reach its uninitialized binding; answers whether there is one.
function hoistTdzRefs(r, from, to, fns, sw) {
    var k = "$" + r.name
    if (hoistRefs[k] == undefined) { return false }
    var sp = hoistSplit(hoistRefs[k], from, to)
    hoistRefs[k] = sp.keep
    var need = false
    for (var i = 0; i < sp.got.length; i++) {
        var p = sp.got[i].at
        if (sw || p <= r.at || hoistInFns(p, fns)) { sp.got[i].chk = true; need = true }
    }
    if (need) { hoistTdzCount = hoistTdzCount + 1 }
    return need
}
function hoistInFns(p, fns) {
    if (fns == undefined) { return false }
    for (var i = 0; i < fns.length; i++) {
        if (fns[i].at <= p && p <= fns[i].to) { return true }
    }
    return false
}

// The Annex B candidates a function body (or the program) takes over.
function hoistClaimAnnex(from, to) {
    var sp = hoistSplit(hoistAnnex, from, to)
    hoistAnnex = sp.keep
    return sp.got
}

// A block, a switch (sw) or an if body claims its functions and its lexical names,
// answering {fns, names, tdz}; the plain functions become Annex B candidates of the
// enclosing function.
function hoistBlockFns(from, to, sw) {
    var fns = hoistClaimFns(from, to)
    var lex = hoistLexKill(from, to, fns, sw)
    for (var i = 0; i < fns.length; i++) {
        if (fns[i].plain) { fns[i].ab = true; hoistAnnex.push(fns[i]) }
    }
    return {fns: fns, names: lex.names, tdz: lex.tdz}
}

// The distinct names of a list of records.
function hoistNames(recs) {
    var names = []
    for (var i = 0; i < recs.length; i++) {
        if (names.indexOf(recs[i].name) < 0) { names.push(recs[i].name) }
    }
    return names
}
