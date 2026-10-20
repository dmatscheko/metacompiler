// Full-syntax test: JavaScript - part 3 of 4, sections 44-56, 58-71 and 73-100.
//
// THE JS RATCHET IS FOUR FILES: as one file its -frozen interpreter leg ran into
// ./test.sh --full's time brake (peak memory grew superlinearly with the file), and
// the fourth is a Module, the one goal symbol in which 'export' may stand.
//
//   tests/js-test-full.js     sections 01-35, the ECMAScript-2022 syntax walk
//   tests/js-test-full.2.js   sections 36-43, 101 and 102, engine agreement, generators,
//                             Error, Math, builtin constructors, integrity levels
//   tests/js-test-full.3.js   sections 44-56, 58-71 and 73-100, the call/class/private-name
//                             model, the Unicode identifier tables, iterators, derived
//                             constructors, JSON, guards, Set and Map, iterator closing,
//                             hasOwnProperty, delete, Object.prototype, extends Object,
//                             g.return() at a yield*, builtin methods as values,
//                             the own names of Math / JSON / Object and `in`,
//                             super reaching Object.prototype, g.return(v) as a
//                             return completion, where yield and return stand,
//                             prototypes, static methods as values, Boolean,
//                             symbols, with, for-in and `in` through the
//                             prototype chain, inherited data properties and
//                             Object.create, a replaced constructor prototype,
//                             getPrototypeOf / setPrototypeOf / defineProperty,
//                             toLocaleString, primitive wrapper objects and property
//                             attributes
//   (section 72 is in tests/js-test-full.4.mjs, beside 57: it needs a file that
//   names no builtin class but in a `new`)
//   tests/js-test-full.4.mjs  section 57, export in a Module
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

// ===== SECTION 44: calling a non-function is a catchable TypeError, worded as node does =====
// `xs[1]()` with undefined in the slot is a catchable TypeError in node, worded
// `xs[1] is not a function` - the callee's SOURCE TEXT, which V8 recovers by
// re-printing the AST and which no runtime value carries, since `xs[1]` and
// `xs[i]` reach the runtime as the same key.
//
// So the text is built at PARSE time (siteTexts, one copy per grammar) and the
// runtime's job is only to say "nothing claimed this call": the floor's js_call
// and the shared memberCall are used by every language's layer-2 internals and
// still abort, and js/ts opt out through js_rt_misscall - the fourth of the
// js_rt_miss* family - plus a raw callability test at the call site itself.
//
// Every message below is node v24's, byte for byte.
function s44() {
    var xs = [1, undefined, null, "s", {}, [1, 2], 7]
    var o = {a: 1, g: 5, sub: {}}
    var i = 1
    var n = xs[6]
    var s = xs[3]
    var u = xs[1]
    var nl = xs[2]
    var arr = xs[5]

    function msg(f) {
        try { f(); return "none" }
        catch (e) { return (e instanceof TypeError) + "|" + (e instanceof Error) + "|" + e.message }
    }

    // A member call whose member is missing or is not callable.
    check("nfn1", msg(function () { return xs[1]() }) === "true|true|xs[1] is not a function")
    check("nfn2", msg(function () { return xs[i]() }) === "true|true|xs[i] is not a function")
    check("nfn3", msg(function () { return o.nope() }) === "true|true|o.nope is not a function")
    // node prints a string-literal index as a dot access.
    check("nfn4", msg(function () { return o["nope"]() }) === "true|true|o.nope is not a function")
    check("nfn5", msg(function () { return o.g() }) === "true|true|o.g is not a function")
    check("nfn6", msg(function () { return o.sub.q() }) === "true|true|o.sub.q is not a function")
    check("nfn7", msg(function () { return o.g?.() }) === "true|true|o.g is not a function")
    check("nfn8", msg(function () { return s.nope() }) === "true|true|s.nope is not a function")
    check("nfn9", msg(function () { return arr.nope() }) === "true|true|arr.nope is not a function")
    check("nfn10", msg(function () { return n.nope() }) === "true|true|n.nope is not a function")
    check("nfn11", msg(function () { return Math.nope() }) === "true|true|Math.nope is not a function")

    // A PLAIN call of a value that is not callable, which is the floor's js_call.
    check("nfn12", msg(function () { return u() }) === "true|true|u is not a function")
    check("nfn13", msg(function () { return nl() }) === "true|true|nl is not a function")
    check("nfn14", msg(function () { return n() }) === "true|true|n is not a function")
    check("nfn15", msg(function () { return s() }) === "true|true|s is not a function")
    check("nfn16", msg(function () { return o() }) === "true|true|o is not a function")
    check("nfn17", msg(function () { return arr() }) === "true|true|arr is not a function")

    // The parentheses of a parenthesised callee are not part of node's text, and
    // anything a CALL produced is "(intermediate value)".
    check("nfn18", msg(function () { return (o.g)() }) === "true|true|o.g is not a function")
    var f5 = function () { return 5 }
    check("nfn19", msg(function () { return f5()() }) === "true|true|f5(...) is not a function")
    // A literal receiver is printed as written.
    check("nfn20", msg(function () { return "abc".nope() }) === "true|true|\"abc\".nope is not a function")
    check("nfn21", msg(function () { return [].nope() }) === "true|true|[].nope is not a function")

    // A class instance whose method does not exist.
    class C { m() { return 1 } }
    var c = new C()
    check("nfn22", msg(function () { return c.zz() }) === "true|true|c.zz is not a function")

    // A generator's member table, and a tagged template.
    check("nfn23", msg(function () { var g = xs[3]; return g`x` }) === "true|true|g is not a function")

    // `new` on something that is not a constructor - node's other wording.
    check("nfn24", msg(function () { return new n() }) === "true|true|n is not a constructor")
    check("nfn25", msg(function () { return new o.nope() }) === "true|true|o.nope is not a constructor")
    check("nfn26", msg(function () { return new o() }) === "true|true|o is not a constructor")

    // ?.() short circuits on a nullish callee instead of raising.
    check("nfn27", u?.() === undefined && o.nope?.() === undefined && xs[1]?.() === undefined)

    // The program CONTINUES, the throw travels the ordinary try/finally machinery,
    // and it can be caught, wrapped and re-thrown.
    var k = 0
    for (var j = 0; j < 3; j++) {
        try { xs[1]() } catch (e) { k = k + 1 }
    }
    check("nfn28", k === 3)
    var log = []
    try {
        try { log.push("t"); o.nope() }
        finally { log.push("f") }
    } catch (e) { log.push("c:" + e.name) }
    check("nfn29", log.join(",") === "t,f,c:TypeError")
    var r = "none"
    try {
        try { xs[1]() } catch (e) { throw new TypeError("re:" + e.message) }
    } catch (e2) { r = e2.message }
    check("nfn30", r === "re:xs[1] is not a function")

    // A call in the MIDDLE of an assignment target, which is its own walk in every
    // engine.
    check("nfn31", msg(function () { n().x = 1 }) === "true|true|n is not a function")
    check("nfn32", msg(function () { o.sub.f().y = 1 }) === "true|true|o.sub.f is not a function")
    // super.m() with nothing on the __super chain: node has no source text for a
    // super receiver either and prints "(intermediate value)".
    class SA { }
    class SB extends SA { go() { return super.zz() } }
    check("nfn33", msg(function () { return new SB().go() }) === "true|true|(intermediate value).zz is not a function")
    // An operation inside the callee's parentheses or an index is RE-PRINTED, as V8
    // prints its AST: spaced and parenthesized, a same-operator run flat, two number
    // literals folded, an assignment as its target (lib/js-callee.js).
    var k = "a"
    check("nfn34", msg(function () { return (o.a + 1)() }) === "true|true|(o.a + 1) is not a function"
        && msg(function () { return (o.a*i-3)() }) === "true|true|((o.a * i) - 3) is not a function")
    check("nfn35", msg(function () { return xs[i+1]() }) === "true|true|xs[(i + 1)] is not a function"
        && msg(function () { return xs[1 + 1]() }) === "true|true|xs[2] is not a function")
    check("nfn36", msg(function () { return (i - 2 - 3)() }) === "true|true|(i - 2 - 3) is not a function"
        && msg(function () { return (1 + 2)() }) === "true|true|3 is not a function")
    check("nfn37", msg(function () { return (o.a && o.g || i)() }) === "true|true|((o.a && o.g) || i) is not a function"
        && msg(function () { return (-o.a)() }) === "true|true|(-o.a) is not a function")
    check("nfn38", msg(function () { return (o.a = 3)() }) === "true|true|o.a is not a function"
        && msg(function () { return o["a b"]() }) === "true|true|o.a b is not a function"
        && msg(function () { return (i ? o.a : o.g)() }) === "true|true|(intermediate value)(intermediate value)(intermediate value) is not a function")
    check("nfn39", msg(function () { return (o.sub.x + o[k])() }) === "true|true|(o.sub.x + o[k]) is not a function"
        && msg(function () { return (i++)() }) === "true|true|(i++) is not a function")
    // A word operator keeps its spacing, which the source as written carries.
    check("nfn40", msg(function () { return (typeof o.a)() }) === "true|true|(typeof o.a) is not a function"
        && msg(function () { return ("a" in o)() }) === "true|true|(\"a\" in o) is not a function"
        && msg(function () { return (o instanceof Object)() }) === "true|true|(o instanceof Object) is not a function")
    check("nfn41", msg(function () { return (void i, delete o.zz)() }) === "true|true|((void i) , (delete o.zz)) is not a function"
        && msg(function () { return xs[typeof /* c */ i]() }) === "true|true|xs[(typeof i)] is not a function"
        && msg(function () { return (typeof typeof i in o)() }) === "true|true|((typeof (typeof i)) in o) is not a function")
    // A number literal prints as its value, and two of them fold, exponents included.
    check("nfn42", msg(function () { return (1e21 + 1)() }) === "true|true|1e+21 is not a function"
        && msg(function () { return (2e3 + i)() }) === "true|true|(2000 + i) is not a function"
        && msg(function () { return xs[0.5 + 0.5]() }) === "true|true|xs[1] is not a function"
        && msg(function () { return (1 / 4)() }) === "true|true|0.25 is not a function")
    check("nfn43", msg(function () { return 2e3.nope() }) === "true|true|2000.nope is not a function"
        && msg(function () { return 0x10.nope() }) === "true|true|16.nope is not a function"
        && msg(function () { return ('x')() }) === "true|true|\"x\" is not a function"
        && msg(function () { return (1e300 * 1e300)() }) === "true|true|Infinity is not a function")
    // A function, an arrow or a class is "(intermediate value)" once per body statement,
    // an object literal once per property, an array literal prints its elements.
    check("nfn44", msg(function () { return (i => 1, "a" in o)() }) === "true|true|((intermediate value) , (\"a\" in o)) is not a function"
        && msg(function () { return (function (a) { i; o }, 5)() }) === "true|true|((intermediate value)(intermediate value) , 5) is not a function"
        && msg(function () { return (class {}, (a = 1) => a, i)() }) === "true|true|((intermediate value) , (intermediate value)(intermediate value) , i) is not a function")
    check("nfn45", msg(function () { return ({a: 1, b: 2}, [i + 1, , "s"])() }) === "true|true|({(intermediate value)(intermediate value)} , [(i + 1),(intermediate value),\"s\"]) is not a function"
        && msg(function () { return ({}).a() }) === "true|true|{}.a is not a function"
        && msg(function () { return (x => 1)()() }) === "true|true|(intermediate value)(...) is not a function")
    check("nfn46", msg(function () { return (`t`, /r/g, `a${i}b${o.a}`)() }) === "true|true|(\"t\" , /r/g , io.a) is not a function")
}

// ===== SECTION 45: the top-level 'this' is the global object (the Script goal) =====
// These grammars parse the SCRIPT goal symbol - the Annex B HTML-like comments are
// accepted (B.1.3 admits them in a Script only), the code is sloppy unless a
// "use strict" directive says otherwise, and top-level `return` is a syntax error,
// which rules out the CommonJS wrapper node's `--check` puts around a file. So the
// top-level `this` is the GLOBAL OBJECT, and every value below is
// `node -e 'vm.runInThisContext(src)'`'s, except where a comment says otherwise.
var s45this = this
var s45arrow = () => this
var s45nested = () => (() => this)()
function s45plain() { return this }
function s45strict() { "use strict"; return this }
function s45outer() { "use strict"; function inner() { return this } return inner() }
function s45sloppyOuter() { function inner2() { return this } return inner2() }
function s45arrowIn() { var a = () => this; return a() }
function s45arrowInStrict() { "use strict"; var a = () => this; return a() }
class s45K { mk() { function inner3() { return this } return inner3() } }
var s45litm = {q: function () { return this }}
class S45C {
    static tag = "S45"
    static who() { return this }
    static viaThis() { return this.tag }
}

function s45() {
    // The top level.
    check("gthis1", typeof s45this === "object")
    check("gthis2", (s45this === undefined) === false)
    check("gthis3", (s45this === null) === false)
    check("gthis4", typeof globalThis === "object")
    check("gthis5", s45this === globalThis)
    check("gthis6", !!s45this)
    // An arrow at the top level binds no `this` of its own, so it walks out to the
    // global object - at any nesting depth.
    check("gthis7", s45arrow() === s45this)
    check("gthis8", s45nested() === s45this)
    // The global object is a PLAIN object here, not a live view of the scope chain:
    // a global variable is not a property of it. node's is (`globalThis.s45this` is
    // the same value), and closing that gap needs an object identity for the root
    // scope in all four engines.
    check("gthis9", globalThis.s45this === undefined)
    // `"" + this` is "[object global]" in node and "[object Object]" here, for the
    // same reason: nothing in the value model marks this object as the global one.
    check("gthis10", ("" + s45this) === "[object Object]")

    // A STATIC method's `this` is the class.
    check("gthis11", S45C.who() === S45C)
    check("gthis12", S45C.viaThis() === "S45")
    check("gthis13", typeof S45C.who() === "function")
    // An instance method's and a constructor's are unchanged by any of that.
    class D { constructor() { this.v = 1 } m() { return this } }
    var d = new D()
    check("gthis14", d.m() === d && d.v === 1)
    var lit = {t: "L", m: function () { return this.t }}
    check("gthis15", lit.m() === "L")

    // OrdinaryCallBindThis (ES2023 10.2.1.2), the per-function STRICT-MODE flag.
    // Every value below is `new vm.Script(src)`'s.
    check("gthis16", s45plain() === globalThis)
    check("gthis18", s45plain.call(null) === globalThis)
    check("gthis19", s45plain.call(undefined) === globalThis)
    check("gthis20", s45plain.apply(null, []) === globalThis)
    // A STRICT function keeps whatever the call site handed over, undefined and
    // null included - which is what a flag that is per FUNCTION and not per file
    // is for: this one sits in the same sloppy Script as the row above it.
    check("gthis21", s45strict() === undefined)
    check("gthis22", s45strict.call(null) === null)
    check("gthis23", s45strict.call(7) === 7)
    // Strictness is INHERITED lexically, in both directions: a plain function
    // declared inside a strict one is strict, and one declared inside a CLASS
    // BODY is strict too (class code always is).
    check("gthis24", s45outer() === undefined)
    check("gthis25", (new s45K()).mk() === undefined)
    // ...while a function nested in an ordinary sloppy one stays sloppy.
    check("gthis26", s45sloppyOuter() === globalThis)
    // An ARROW binds no `this` of its own, so it reports the enclosing FUNCTION's -
    // which is the global object when that function is sloppy and undefined when
    // it is strict.
    check("gthis27", s45arrowIn() === globalThis)
    check("gthis28", s45arrowInStrict() === undefined)
    // An object-literal method, and a getter, are functions of the enclosing code's
    // strictness like any other.
    check("gthis29", s45litm.q.call(null) === globalThis)
    // A sloppy GENERATOR binds `this` the same way in the interpreter half and
    // under llvm.Run - and NOT in the native build, which is why there is no
    // assertion here. `js_this` is one global slot in layer 2, and the emitter arms
    // it with the RECEIVER of the `g.next()` call, so a generator body reads the
    // generator object instead of the receiver its function was called with.
    // Closing it needs a per-coroutine `this` in the C floor's control block.
    // Recorded rather than asserted, since clang-check and native-full run this file
    // natively.
    // Not asserted: a sloppy body boxes a primitive receiver (s45plain.call(7) is a
    // Number object in node), and this value model has no wrapper objects.
    var detached = lit.m
    check("gthis17", typeof detached === "function")
}

// ===== SECTION 46: parentheses do not drop the receiver =====
// `(o.m)()` calls m with `this` === o. A parenthesized expression evaluates to whatever
// the inner expression evaluates to, and a MemberExpression evaluates to a REFERENCE,
// which carries its base - so the parentheses are transparent to the receiver exactly
// as they are to an assignment target ((a.b).c = 1 has always assigned where a.b.c = 1
// does). The matrix and --cross cannot see a both-halves-agree defect here, so every
// row is node's answer.
//
// The negative rows are the point of the section as much as the positive ones. Only a
// parenthesized MemberExpression is a Reference: `(0, o.m)()` is the IDIOM for calling a
// method WITHOUT its receiver, and a fix that made it pass `o` would be a regression
// dressed as a feature. The conditional, comma, `||` and assignment forms are the same
// case. Every value here is `new vm.Script(src)`'s.
//
// Every operand is read out of s46a so the constant folder cannot answer any of it.
var s46a = ["m", "n", 1, 2, 3];
function s46tag(v) {
    if (v === undefined) { return "undefined" }
    if (v === globalThis) { return "GLOBAL" }
    if (v !== null && v.nm !== undefined) { return v.nm }
    return typeof v;
}
class S46B { who() { return "B/" + s46tag(this) } static swho() { return "SB/" + s46tag(this) } }
// `super.m` is a Reference too, and its thisValue is the CURRENT `this` - which is what
// makes the parenthesized and the computed spelling agree with the plain call.
class S46D extends S46B {
    who() { return "D" }
    static sPlain() { return super.swho() }
    static sParen() { return (super.swho)() }
    static sIndex() { return super[s46a[1] === "n" ? "swho" : "no"]() }
}
class S46K { constructor() { this.nm = "k" } m() { return s46tag(this) } static sm() { return this } }

function s46() {
    var o = {nm: "o", m: function () { return s46tag(this) }, n: function () { return s46tag(this) }};
    var p = {nm: "p"};
    var k0 = s46a[0];
    check("paren01", o.m() === "o");
    check("paren02", (o.m)() === "o");
    check("paren03", ((o.m))() === "o");
    check("paren04", (o?.m)() === "o");
    check("paren05", (o[k0])() === "o");
    check("paren06", ((o).m)() === "o");
    check("paren07", (o.m).call(p) === "p");
    check("paren08", (o.m)?.() === "o");
    check("paren09", (o?.m)?.() === "o");
    // A TAGGED template behind a parenthesized member is a call like any other.
    check("paren10", (o.m)`x` === "o");
    var w = {nm: "w", g: function () { return {nm: "inner", h: function () { return s46tag(this) }} }};
    check("paren11", (w.g().h)() === "inner");
    check("paren12", (w.g)().nm === "inner");
    // As a STATEMENT, not only in an expression.
    var arr = [s46a[2]];
    (arr.push)(s46a[3]);
    check("paren13", arr.length === 2 && arr[1] === 2);
    check("paren14", ("abc".toUpperCase)() === "ABC");
    var kk = new S46K();
    check("paren15", (kk.m)() === "k");
    check("paren16", (S46K.sm)() === S46K);
    check("paren17", ((S46K.sm))() === S46K);
    check("paren18", S46D.sPlain() === "SB/" + s46tag(S46D));
    check("paren19", S46D.sParen() === S46D.sPlain());
    check("paren20", S46D.sIndex() === S46D.sPlain());
    // ...and the NEGATIVE side: a value, not a Reference, so `this` is the global object
    // in this sloppy Script.
    var f = o.m;
    check("paren21", f() === "GLOBAL");
    check("paren22", (f)() === "GLOBAL");
    check("paren23", (0, o.m)() === "GLOBAL");
    check("paren24", (s46a[0] === "m" ? o.m : o.n)() === "GLOBAL");
    check("paren25", (o.m, o.n)() === "GLOBAL");
    check("paren26", (o.m || o.n)() === "GLOBAL");
    check("paren27", (o.m = o.n, o.n)() === "GLOBAL");
    // V8 prints the callee's source text without the parentheses; so does siteTexts.
    check("paren28", (function () {
        try { (o.nope)(); return "no" } catch (e) { return e.message }
    })() === "o.nope is not a function");
}

// ===== SECTION 47: a class's STATIC and INSTANCE namespaces =====
// `class C { static m(){} m(){} }` is TWO members in JavaScript - one on the
// constructor and one on the prototype - so `C.m()` must answer the STATIC method,
// `class F extends E {}` must resolve an inherited static to E's static, and an
// instance must not see a static (`typeof c.sonly` is "undefined" in node). Every
// static member lives in a separate `__statics` table on the descriptor; instance
// members stay in the descriptor, which is what keeps the shared js_get / js_set /
// js_mcall - and therefore the other fifteen languages - untouched.
//
// Every value is `new vm.Script(src)`'s, and every operand comes out of s47a so the
// constant folder cannot answer any of it.
var s47a = [1, 2, 3, "m", "who"];
class S47C {
    static m() { return "S-m" }
    m() { return "I-m" }
    static get g() { return "S-g" }
    get g() { return "I-g" }
    static sf = s47a[0];
    static sonly() { return "S-only" }
    ionly() { return "I-only" }
    constructor() { this.f = s47a[1] }
    static set sv(v) { S47C.svSeen = v }
}
class S47E { static x() { return "E-static-x" } static only() { return "E-only" } }
class S47F extends S47E { x() { return "F-inst-x" } }
class S47H { static n() { return "H-S-n" } n() { return "H-I-n" } static viaThis() { return this.n() } }
// The four spellings of a super member. `super.m` is a Reference whose thisValue is the
// CURRENT `this`, so which SIDE it resolves on is decided by the receiver: a static
// method's is the class, an instance method's is the instance. The parenthesized and
// computed forms have to agree with the plain one.
class S47G extends S47C {
    static viaSuper() { return super.m() }
    static viaSuperIdx() { return super[s47a[3]]() }
    viaSuperI() { return super.m() }
    viaSuperParen() { return (super.m)() }
}
class S47B { who() { return "B/" + (this === undefined ? "undef" : this.nm) } }
class S47D extends S47B { who() { return "D" } paren() { return (super.who)() } }

function s47() {
    var c = new S47C();
    check("stat01", S47C.m() === "S-m");
    check("stat02", c.m() === "I-m");
    check("stat03", S47C.g === "S-g");
    check("stat04", c.g === "I-g");
    check("stat05", S47C.sf === 1 && c.f === 2);
    // An instance cannot see a static, and the constructor cannot see an instance member.
    check("stat06", typeof S47C.sonly === "function" && typeof c.sonly === "undefined");
    check("stat07", typeof S47C.ionly === "undefined" && typeof c.ionly === "function");
    // An INHERITED static beats the subclass's own INSTANCE member of the same name.
    check("stat08", S47F.x() === "E-static-x" && (new S47F()).x() === "F-inst-x");
    check("stat09", S47F.only() === "E-only");
    // `this.n()` inside a static resolves on the static side.
    check("stat10", S47H.viaThis() === "H-S-n" && (new S47H()).n() === "H-I-n");
    check("stat11", (s47a[3] in S47C) === true && ("ionly" in S47C) === false);
    // Every class object has a `name`, and it is the class's own.
    check("stat12", S47C.name === "S47C" && S47F.name === "S47F");
    S47C.zz = s47a[2];
    check("stat13", S47C.zz === 3 && (new S47C()).zz === undefined);
    S47G.sv = s47a[0];
    check("stat14", S47G.svSeen === 1);
    check("stat15", S47G.viaSuper() === "S-m" && S47G.viaSuperIdx() === "S-m");
    check("stat16", (new S47G()).viaSuperI() === "I-m" && (new S47G()).viaSuperParen() === "I-m");
    // A parent INSTANCE method reached through a super REFERENCE runs on the current
    // receiver, not on undefined.
    var d = new S47D(); d.nm = "d";
    check("stat17", d.paren() === "B/d");
    check("stat18", (function () {
        try { S47C.ionly(); return "no" } catch (e) { return e.message }
    })() === "S47C.ionly is not a function");
}

// ===== SECTION 48: a class object's OWN KEYS, e.constructor, and a method value's
//       OVERRIDABLE receiver =====
//
//   Object.keys(<class>)  node lists the static FIELDS. A static method, a static
//                         accessor and a private static are not enumerable; a
//                         runtime `C.zz = 1` is.
//   e.constructor         node hands back the class, and the FUNCTION itself for
//                         `new F()`.
//   f.call(x)             a method read AS A VALUE - `const f = o.m`, `super.m` -
//                         runs on x, not on the receiver the READ was made with.
//
// Every operand comes out of s48a so the constant folder cannot answer any of it.
var s48a = [1, "two", 3, "m", "who", "constructor"];
class S48C {
    static sf = s48a[0];
    static sf2 = s48a[1];
    static sm() { return "sm" }
    static get sg() { return "sg" }
    m() { return "m" }
    get g() { return "g" }
    iv = s48a[0];
}
class S48Sub extends S48C { static own = s48a[2]; }
class S48B { who() { return "B/" + this.nm } }
class S48D extends S48B {
    constructor() { super(); this.nm = "d" }
    paren() { return (super.who)() }
    callx(o) { return super.who.call(o) }
    applyx(o) { return super.who.apply(o, []) }
    bindx(o) { return super.who.bind(o)() }
}
function S48F(n) { this.nm = n }

function s48() {
    // A static FIELD is enumerable; a static method, a static accessor and the
    // instance members are not, and an inherited static is not an OWN key.
    check("keys01", Object.keys(S48C).join("|") === "sf|sf2")
    check("keys02", Object.keys(S48Sub).join("|") === "own")
    check("keys03", Object.values(S48C).join("|") === "1|two")
    var en = Object.entries(S48C)
    check("keys04", en.length === 2 && en[0][0] === "sf" && en[0][1] === 1 && en[1][1] === "two")
    S48C.zz = s48a[2]
    check("keys05", Object.keys(S48C).join("|") === "sf|sf2|zz" && S48C.zz === 3)
    var fi = []
    for (var k in S48C) { fi.push(k) }
    check("keys06", fi.join("|") === "sf|sf2|zz")
    check("keys07", Object.keys(Object.assign({}, S48C)).join("|") === "sf|sf2|zz")
    check("keys08", Object.keys(new S48C()).join("|") === "iv")
    // The class object still answers every static it did before.
    check("keys09", S48C.sm() === "sm" && S48C.sg === "sg" && S48Sub.sm() === "sm")
    var c = new S48C()
    check("ctor01", c.constructor === S48C && c.constructor.name === "S48C")
    check("ctor02", (new S48Sub()).constructor === S48Sub)
    check("ctor03", (s48a[5] in c) === true)
    check("ctor04", (new S48F("f")).constructor === S48F)
    check("ctor05", Object.keys(c).join("|") === "iv")
    // A method read as a VALUE takes the receiver of the call that eventually runs
    // it, and keeps the read-time one only when the call supplies none.
    var d = new S48D()
    var o = {nm: "x"}
    check("recv01", d.callx(o) === "B/x")
    check("recv02", d.applyx(o) === "B/x")
    check("recv03", d.bindx(o) === "B/x")
    check("recv04", d.paren() === "B/d")
    var bm = d.who
    check("recv05", bm.call(o) === "B/x" && bm.apply(o, []) === "B/x" && bm.bind(o)() === "B/x")
}

// ===== SECTION 49: the PRIVATE-NAME model, and for-in over what is not a plain
//       object =====
//
//   THE PRIVATE NAME. `#x` is stored under the mangled key '__#x' in all four
//   engines, which the engine's __-slot rule already hides: `Object.keys(p)` is "y"
//   as in node, and `o["#x"] = 1` and a private `#x` are two different slots. Every
//   __-name is skipped by the static-side test, so the mangled private spelling is
//   the one exception there (`C.#s`, a static private, must read).
//
//   for (k in [7,8])  yields the INDICES as strings, as in node. A string enumerates
//   the same way; null, undefined and a primitive enumerate nothing rather than
//   raising. `{...[7,8]}` goes through the same call.
//
//   #x in o - ES2022's ergonomic brand check.
//
// Every operand comes out of s49a so the constant folder cannot answer any of it.
var s49a = [0, 1, 2, "#x", "ab", 4];
class S49P {
    #x = s49a[1];
    y = s49a[2];
    static #s = s49a[5];
    #bump(n) { this.#x = this.#x + n; return this.#x }
    get val() { return this.#x }
    run(n) { return this.#bump(n) }
    brand(o) { return #x in o }
    brandM(o) { return #bump in o }
    static gets() { return S49P.#s }
    static bumps() { S49P.#s = S49P.#s + s49a[1]; return S49P.#s }
}

function s49() {
    var p = new S49P()
    // A private field is not an own key, is not spread and is not enumerated.
    check("priv01", Object.keys(p).join("|") === "y")
    check("priv02", Object.keys({ ...p }).join("|") === "y")
    var fi = ""
    for (var k in p) { fi = fi + k + "," }
    check("priv03", fi === "y,")
    // It is still readable and writable from inside the class body.
    check("priv04", p.val === 1 && p.run(s49a[5]) === 5 && p.val === 5)
    // A STATIC private is reachable, and it is not an own key of the class object
    // either.
    check("priv05", S49P.gets() === 4 && S49P.bumps() === 5 && S49P.gets() === 5)
    check("priv06", Object.keys(S49P).join("|") === "")
    // '#x' AS AN ORDINARY STRING KEY is a DIFFERENT SLOT from the private #x: the
    // write below must not disturb the field, and it IS enumerable.
    p[s49a[3]] = s49a[5]
    check("priv07", p.val === 5 && p[s49a[3]] === 4)
    check("priv08", Object.keys(p).join("|") === "y|#x")
    var q = {}
    q[s49a[3]] = s49a[1]
    check("priv09", Object.keys(q).join("|") === "#x" && q[s49a[3]] === 1)
    // The ergonomic brand check, over a field and over a method.
    check("priv10", p.brand(p) === true && p.brand({}) === false)
    check("priv11", p.brandM(p) === true && p.brandM({}) === false)

    // ----- for-in over what is not a plain object -----
    var arr = [s49a[5], s49a[5], s49a[5]]
    var ai = ""
    for (var i in arr) { ai = ai + i + "," }
    check("fin01", ai === "0,1,2,")
    // The key is a STRING, as node says, not a number.
    var tk = ""
    for (var i2 in arr) { tk = tk + typeof i2 }
    check("fin02", tk === "stringstringstring")
    var si = ""
    for (var j in s49a[4]) { si = si + j + "," }
    check("fin03", si === "0,1,")
    // null, undefined and a primitive enumerate NOTHING and do not raise.
    var none = [null, undefined, s49a[5], true, ""]
    var nc = 0
    for (var n = 0; n < none.length; n++) {
        for (var k2 in none[n]) { nc = nc + 1 }
    }
    check("fin04", nc === 0)
    // break and continue inside a for-in over an array.
    var cc = ""
    for (var c in arr) { if (c === "1") { continue } cc = cc + c }
    check("fin05", cc === "02")
    var bb = ""
    for (var b in arr) { if (b === "1") { break } bb = bb + b }
    check("fin06", bb === "0")
    // The same call is what object spread makes.
    var sa = { ...arr }
    check("fin07", Object.keys(sa).join("|") === "0|1|2" && sa[s49a[1]] === 4)
    var ss = { ...s49a[4] }
    check("fin08", Object.keys(ss).join("|") === "0|1" && ss[s49a[0]] === "a")
    check("fin09", Object.keys({ ...null }).length === 0 && Object.keys({ ...s49a[5] }).length === 0)
}

// ===== SECTION 50: for-in is a SNAPSHOT, a private name is per CLASS BODY, and a
//       FUNCTION carries own properties =====
// Three things every engine here could agree on wrongly, so byte-identity could not
// see them; node is the only witness. Each row below was checked against node v24.
//
//   DELETING A KEY DURING for-in. js_jskeys / jsKeys answer a SNAPSHOT, so a key the
//   body deleted before its turn came must be skipped: `for (k in o) { s += k;
//   delete o.c }` gives "ab" in node. The compilers guard each iteration with
//   js_jshas and the interpreters with propIn.
//
//   THE SAME PRIVATE NAME IN A SUBCLASS. `class A { #v } class B extends A { #v }` is
//   TWO names in JavaScript, so the key carries the CLASS BODY ('__#<body>#v'),
//   resolved to the innermost enclosing body that declares the name - so an inner
//   class reading an outer class's private name still reaches it.
//
//   AN OWN PROPERTY ON A FUNCTION. `f.p = 1`: the map lives beside the function in a
//   side table in all four engines, and for-in, Object.keys, `in`, hasOwnProperty
//   and delete all read it.
//
// Every operand comes out of s50a so the constant folder cannot answer any of it.
var s50a = [0, 1, 2, 3, "a", "b", "c", "d"];

class S50A {
    #v = s50a[1];
    getA() { return this.#v }
    setA(x) { this.#v = x }
    hasA(o) { return #v in o }
}
class S50B extends S50A {
    #v = s50a[2];
    getB() { return this.#v }
    setB(x) { this.#v = x }
    hasB(o) { return #v in o }
}
class S50C { #m() { return s50a[4] } callC() { return this.#m() } }
class S50D extends S50C { #m() { return s50a[5] } callD() { return this.#m() } }
class S50E { static #s = s50a[1]; static getE() { return S50E.#s } }
class S50F extends S50E { static #s = s50a[2]; static getF() { return S50F.#s } }
// An INNER class body reading the OUTER body's private name: the reference resolves
// to the innermost enclosing body that DECLARES it, which is the outer one here.
class S50G {
    #g = s50a[3];
    read() { return this.#g }
    make() { return class S50H { peek(o) { return o.#g } } }
}
// A method that reads a field declared LATER in the same body - which is why the
// reference key cannot be settled while the body is being walked.
class S50L { late() { return this.#l } #l = s50a[3] }

function s50fn(x) { return x }

function s50() {
    // ----- for-in is a snapshot, but a deleted key is skipped -----
    var o1 = { a: s50a[1], b: s50a[2], c: s50a[3] }
    var r1 = ""
    for (var k1 in o1) { r1 = r1 + k1; delete o1.c }
    check("fin10", r1 === "ab")
    var o2 = { a: s50a[1], b: s50a[2], c: s50a[3] }
    var r2 = ""
    for (var k2 in o2) { r2 = r2 + k2; if (k2 === s50a[4]) { delete o2.b } }
    check("fin11", r2 === "ac")
    var o3 = { a: s50a[1], b: s50a[2], c: s50a[3], d: s50a[3] }
    var r3 = ""
    for (var k3 in o3) { r3 = r3 + k3; delete o3.b; delete o3.d }
    check("fin12", r3 === "ac")
    // A key DELETED AND PUT BACK is still visited: it is present when its turn comes.
    var o4 = { a: s50a[1], b: s50a[2], c: s50a[3] }
    var r4 = ""
    for (var k4 in o4) { r4 = r4 + k4; delete o4.a; o4.a = s50a[1] }
    check("fin13", r4 === "abc")
    // A key ADDED during the loop is NOT visited - the list is still a snapshot.
    var o5 = { a: s50a[1], b: s50a[2] }
    var r5 = ""
    for (var k5 in o5) { r5 = r5 + k5; o5.z = s50a[3] }
    check("fin14", r5 === "ab")
    // The guard must not drop anything from an array, a string or a class object.
    var ar = [s50a[1], s50a[2], s50a[3]]
    var r6 = ""
    for (var k6 in ar) { r6 = r6 + k6 }
    check("fin15", r6 === "012")
    var r7 = ""
    for (var k7 in s50a[4] + s50a[5]) { r7 = r7 + k7 }
    check("fin16", r7 === "01")
    var r8 = ""
    for (var k8 in S50E) { r8 = r8 + k8 }
    check("fin17", r8 === "")

    // ----- a private name is scoped to its CLASS BODY -----
    var b = new S50B()
    check("priv12", b.getA() === 1 && b.getB() === 2)
    b.setA(s50a[3] * 10)
    check("priv13", b.getA() === 30 && b.getB() === 2)
    b.setB(s50a[2] * 10)
    check("priv14", b.getA() === 30 && b.getB() === 20)
    var d = new S50D()
    check("priv15", d.callC() === "a" && d.callD() === "b")
    check("priv16", S50E.getE() === 1 && S50F.getF() === 2)
    // The brand check is per class body too: an A has A's #v and not B's.
    check("priv17", b.hasA(b) === true && b.hasB(b) === true)
    check("priv18", b.hasA(new S50A()) === true && b.hasB(new S50A()) === false)
    var g = new S50G()
    var H = g.make()
    check("priv19", g.read() === 3 && (new H()).peek(g) === 3)
    check("priv20", (new S50L()).late() === 3)
    // Neither slot is enumerable, in either class.
    check("priv21", Object.keys(b).join("|") === "" && Object.keys({ ...b }).join("|") === "")

    // ----- a FUNCTION carries own properties -----
    s50fn.p = s50a[1]
    s50fn.q = s50a[2]
    check("fnp01", s50fn.p === 1 && s50fn.q === 2 && s50fn.zz === undefined)
    var fk = ""
    for (var k9 in s50fn) { fk = fk + k9 + "," }
    check("fnp02", fk === "p,q,")
    check("fnp03", Object.keys(s50fn).join("|") === "p|q")
    check("fnp04", (s50a[4] in s50fn) === false && ("p" in s50fn) === true)
    check("fnp05", Object.prototype.hasOwnProperty.call(s50fn, "p") === true)
    delete s50fn.p
    check("fnp06", s50fn.p === undefined && Object.keys(s50fn).join("|") === "q")
    // A function that was never written to has no keys and is not disturbed.
    check("fnp07", Object.keys(s50) .join("|") === "" && ("q" in s50) === false)
    // A property holding a function is callable, and the function is still a function.
    s50fn.m = function () { return s50a[3] * 14 }
    check("fnp08", s50fn.m() === 42 && s50fn(s50a[3]) === 3 && typeof s50fn === "function")
    // A property holding UNDEFINED is still an own key, as it is on any object.
    s50fn.u = undefined
    check("fnp09", ("u" in s50fn) === true && Object.keys(s50fn).join("|") === "q|m|u")

    // ----- a FUNCTION's Object.prototype SURFACE, as methods -----
    // All four are answered by this engine, in all four legs (the host's own
    // prototype chain exists under goja only).
    check("fnp10", s50fn.hasOwnProperty("q") === true && s50fn.hasOwnProperty(s50a[4]) === false)
    check("fnp11", s50fn.propertyIsEnumerable("q") === true && s50fn.propertyIsEnumerable(s50a[4]) === false)
    // toString agrees with String(f), which is the one spelling every leg already had;
    // node agrees with itself here too, so the row does not pin our text.
    check("fnp12", typeof s50fn.toString() === "string" && s50fn.toString() === String(s50fn))
    check("fnp13", s50fn.valueOf() === s50fn)
    // An OWN property still wins over the built-in, as it does in node.
    var s50g = function () { return s50a[1] }
    s50g.valueOf = s50a[3]
    check("fnp14", s50g.valueOf === 3 && typeof s50g.toString === "function")
    // A function with NO own properties answers false rather than throwing.
    check("fnp15", s50.hasOwnProperty("q") === false && s50.propertyIsEnumerable("q") === false)
    // The VALUES, not just the keys: Object.values / entries / assign / spread read a
    // function's own properties through a different path from for-in.
    var s50v = Object.values(s50fn)
    var s50e = Object.entries(s50fn)
    check("fnp16", s50v.length === 3 && s50v[0] === 2 && typeof s50v[1] === "function" && s50v[2] === undefined)
    check("fnp16b", s50e.length === 3 && s50e[0][0] === "q" && s50e[0][1] === 2)
    check("fnp17", Object.assign({}, s50fn).q === 2 && ({ ...s50fn }).q === 2)
}

// ===== SECTION 51: a function's own `name`, `length` and SELF NAME =====
// Every function has two own properties nothing in the program writes; node says
// `f`/1 for `function f(a){}`. Every value below was read off node 24 before it was
// asserted.
//
// `length` is the parameters BEFORE the first default and before the rest parameter,
// and a destructuring pattern counts as ONE - which is why fn9 is 3 and not 4.
//
// What is deliberately NOT asserted, because no engine here can answer it: the other
// three own names node reports on a function declaration (`arguments`, `caller`,
// `prototype` - there is no prototype chain and no call-site frame in this value
// model), and `f.constructor`, which diverges goja vs -frozen and is left unasserted
// on purpose so that no assertion pins the divergence.
function s51() {
    function fnDecl(a) { return a }
    var fnNamedExpr = function fnInner(a, b) { return a + b }
    var fnArrow = (x) => x
    var fnAnon = function () { return 1 }
    var fnDflt = function (a, b = 1, c) { return a }
    var fnRest = function (...r) { return r.length }
    var fnDestr = function ({ a }, [b], c) { return c }
    var fnDrest = function (a, ...r) { return a }
    var fnZero = () => 0
    check("fnm01", fnDecl.name === "fnDecl" && fnDecl.length === 1)
    // A NAMED function expression keeps ITS OWN name, and does not take the binding's.
    check("fnm02", fnNamedExpr.name === "fnInner" && fnNamedExpr.length === 2)
    // NamedEvaluation: an ANONYMOUS form takes the name of the binding it is
    // DIRECTLY assigned to.
    check("fnm03", fnArrow.name === "fnArrow" && fnArrow.length === 1)
    check("fnm04", fnAnon.name === "fnAnon" && fnAnon.length === 0)
    check("fnm05", fnZero.name === "fnZero" && fnZero.length === 0)
    check("fnm06", fnDflt.length === 1)
    check("fnm07", fnRest.length === 0)
    check("fnm08", fnDrest.length === 1)
    check("fnm09", fnDestr.length === 3)
    // ...and only when it IS the initializer: node names nothing in `f(function(){})`.
    function s51id(x) { return x }
    var fnVia = s51id(function () { return 2 })
    check("fnm10", fnVia.name === "" && fnVia.length === 0)
    var fnAssigned
    fnAssigned = function () { return 3 }
    check("fnm11", fnAssigned.name === "fnAssigned")
    var s51obj = {
        mExpr: function () { return 1 },
        mShort(a) { return a },
        mArrow: (a, b) => a,
        deep: { inner: function () { return 4 } }
    }
    check("fnm12", s51obj.mExpr.name === "mExpr" && s51obj.mExpr.length === 0)
    check("fnm13", s51obj.mShort.name === "mShort" && s51obj.mShort.length === 1)
    check("fnm14", s51obj.mArrow.name === "mArrow" && s51obj.mArrow.length === 2)
    check("fnm15", s51obj.deep.inner.name === "inner")
    async function s51async(a, b) { return a }
    function* s51gen(a) { yield a }
    check("fnm16", s51async.name === "s51async" && s51async.length === 2)
    check("fnm17", s51gen.name === "s51gen" && s51gen.length === 1)
    class S51C {
        constructor(a) { this.a = a }
        meth(x) { return x }
        static stat(a, b) { return a }
        async am(a) { return a }
        *gm() { yield 1 }
    }
    var s51i = new S51C(1)
    // A method read as a VALUE answers the method's own name and arity: the receiver
    // arrives as a hidden first argument here and is not a declared parameter.
    check("fnm18", s51i.meth.name === "meth" && s51i.meth.length === 1)
    check("fnm19", S51C.stat.name === "stat" && S51C.stat.length === 2)
    check("fnm20", s51i.am.name === "am" && s51i.am.length === 1)
    check("fnm21", s51i.gm.name === "gm" && s51i.gm.length === 0)
    // A BOUND function: "bound " + the target's name, and the target's length less the
    // arguments already supplied, floored at 0.
    var fnBound = fnDecl.bind(null)
    var fnBound2 = fnNamedExpr.bind(null, 1)
    check("fnm22", fnBound.name === "bound fnDecl" && fnBound.length === 1)
    check("fnm23", fnBound2.name === "bound fnInner" && fnBound2.length === 1)
    check("fnm24", fnRest.bind(null, 1, 2, 3).length === 0)
    // Both are OWN properties, and NON-ENUMERABLE: getOwnPropertyNames reports them,
    // Object.keys and for-in do not. The list is asserted on an ARROW, where node's own
    // answer is exactly ["length","name"]; a function DECLARATION has three more there
    // (`arguments`, `caller`, `prototype`) and no engine here can answer any of them -
    // there is no prototype chain and no call-site frame in this value model - so for
    // that shape only the two names' PRESENCE is asserted, which is true in node too.
    var g = Object.getOwnPropertyNames(fnZero)
    check("fnm25", g.length === 2 && g[0] === "length" && g[1] === "name")
    var gd = Object.getOwnPropertyNames(fnDecl)
    check("fnm26", gd.indexOf("length") >= 0 && gd.indexOf("name") >= 0)
    check("fnm27", Object.keys(fnDecl).length === 0)
    check("fnm28", fnDecl.hasOwnProperty("name") && fnDecl.hasOwnProperty("length"))
    check("fnm29", !fnDecl.propertyIsEnumerable("name"))
    var fnKeys = []
    for (var fk in fnDecl) { fnKeys.push(fk) }
    check("fnm30", fnKeys.length === 0)
    // Both are NON-WRITABLE: the write is dropped, silently, as it is in sloppy node.
    fnDecl.name = "zzz"
    fnDecl.length = 99
    check("fnm31", fnDecl.name === "fnDecl" && fnDecl.length === 1)
    // ...and a function can still carry own properties of its own beside them.
    fnZero.tag = 7
    var g2 = Object.getOwnPropertyNames(fnZero)
    check("fnm32", fnZero.tag === 7 && g2.length === 3 && g2[2] === "tag")
    check("fnm33", fnZero.hasOwnProperty("tag") && fnZero.propertyIsEnumerable("tag"))
    check("fnm34", Object.keys(fnZero).length === 1 && Object.keys(fnZero)[0] === "tag")
    // Closures built at ONE definition site in a loop each carry their own pair, and
    // binding one of them names only the bound value.
    var fnLoop = []
    for (var fi = 0; fi < 300; fi++) { fnLoop.push(function looped(a, b) { return a + b }) }
    var fnLoopB = fnLoop[150].bind(null, 1)
    check("fnm35", fnLoop[0].name === "looped" && fnLoop[299].length === 2 && fnLoop[150] !== fnLoop[151])
    check("fnm36", fnLoopB.name === "bound looped" && fnLoopB.length === 1 && fnLoopB(2) === 3 && fnLoop[150].name === "looped")
    // ----- a named function EXPRESSION sees its OWN NAME (ES2023 15.2.5) -----
    // The spec's funcEnv - one scope, one binding of the name to the function object,
    // between the definition site and the function's own scope - is what all nine rows
    // below come out of; every value was read off node 24 first.
    var s51fact = function fact(n) { return n < 2 ? 1 : n * fact(n - 1) }
    check("fns01", s51fact(5) === 120)
    // The name does NOT leak into the enclosing scope.
    var s51hid = function hidden51() { return 1 }
    check("fns02", typeof hidden51 === "undefined")
    // An outer variable of the same name is SHADOWED inside and untouched outside.
    var shad51 = 7
    var s51shad = function shad51() { return typeof shad51 }
    check("fns03", s51shad() === "function" && shad51 === 7)
    // A DECLARATION is unchanged: it binds in the ENCLOSING scope, and only there.
    check("fns04", s51decl(5) === 120 && typeof s51decl === "function")
    // A `var` of the same name IN THE BODY shadows the self binding, so this is 2.
    var s51var = function v51() { var v51 = 1; v51 = 2; return v51 }
    check("fns05", s51var() === 2)
    // The name IS the function - one object, bound after it exists.
    var s51id = function self51() { return self51 === s51id }
    check("fns06", s51id() === true)
    // A PARAMETER of the same name wins over the self binding.
    var s51par = function p51(p51) { return p51 }
    check("fns07", s51par(42) === 42)
    // ...and a NESTED function still sees it.
    var s51nest = function o51() { return (function () { return typeof o51 })() }
    check("fns08", s51nest() === "function")
    // A generator function EXPRESSION binds its own name too, which is what lets it
    // delegate to itself. Driven by next() rather than by spread: spreading a generator
    // answers an empty array in both halves (a separate gap).
    var s51gen = function* gen51(n) { yield n; if (n > 0) { yield* gen51(n - 1) } }
    var s51out = ""
    var s51it = s51gen(2)
    var s51r = s51it.next()
    while (!s51r.done) { s51out = s51out + s51r.value; s51r = s51it.next() }
    check("fns09", s51out === "210")
    // NOT asserted, and deliberately: an ASSIGNMENT to the name inside the body. node
    // ignores it silently in sloppy code and raises a TypeError in strict code, because
    // the binding is IMMUTABLE; the write lands on the funcEnv slot in BOTH halves here,
    // so they agree and an assertion would pin the divergence. The mechanism it would
    // take is written up at fnSelf in both grammars.
}
// The declaration s51decl is used by fns04 from inside s51, and a declaration binds in
// the scope AROUND the function it is written in - so it is a top-level sibling here,
// which is also what makes it the control for the whole group: it must keep working
// exactly as it did.
function s51decl(n) { return n < 2 ? 1 : n * s51decl(n - 1) }

// ===== SECTION 52: UNICODE IDENTIFIERS - the real ID_Start / ID_Continue tables =====
// Every name below is legal JavaScript and node runs this section unchanged. The point is
// that the two js grammars decide with node's own ID_Start / ID_Continue tables
// (lib/ident-unicode.abnf) instead of a range approximation, so the names are chosen out
// of the code points where an approximation goes wrong in either direction:
//   - U+FF41, a fullwidth 'a', is ID_Start.
//   - U+00B7, U+200C and U+203F are ID_Continue and NOT ID_Start: node parses
//     'var a<U+00B7>b' and refuses 'var <U+00B7>b', and so do we.
//   - U+2176 is a roman numeral (Nl) and IS ID_Start; U+1D7CB is a mathematical digit
//     (Nd) and is ID_Continue only, so it may not lead a name.
// An ESCAPED spelling is the same name as the raw one (uni09..uni13): every engine
// resolves a \u escape in a name back to the character it spells (idName) before any
// scope, property, label or class table sees it. The other half of that guard -
// '\u002a' being an asterisk and no identifier however it is spelled - is a PARSE
// error and so only the corpus (test262 fail/) can hold it.
function s52() {
    var ａ = 11
    check("uni01", ａ === 11)
    var a·b = 12
    check("uni02", a·b === 12)
    var c‌d = 13
    check("uni03", c‌d === 13)
    var e‿f = 14
    check("uni04", e‿f === 14)
    var ⅶ = 15
    check("uni05", ⅶ === 15)
    var 日本 = 17
    check("uni06", 日本 === 17)
    var _𝟋 = 18
    check("uni07", _𝟋 === 18)
    check("uni08", typeof ａ === "number")
    var \u0061bc = 21
    check("uni09", abc === 21)
    var x\u{79}z = 22
    check("uni10", \u0078yz === 22 && typeof xyz === "number")
    var uo = { \u006b: 23, k\u0032: 24 }
    check("uni11", uo.k === 23 && uo.\u006b\u0032 === 24)
    function \u0067(\u0070) { return p + 1 }
    check("uni12", g(1) === 2)
    class \u0055C { m\u0065() { return 25 } }
    check("uni13", new UC().me() === 25)
    var unf = function \u0062c() { return 0 }
    check("uni14", unf.name === "bc")
}

// ===== SECTION 53: an ITERATOR spreads and destructures =====
// A spread and an array pattern read an iterator (a callable `next`, the same structural
// test for-of makes - there are no symbols) through next() rather than by index. An array
// pattern takes only as many values as it has positions and then CLOSES the iterator
// (its return(), so a generator's finally runs), which is what makes destructuring an
// endless generator terminate. Values read off node 24.
function* s53g() { yield 1; yield 2 }
var s53log = []
function* s53nat() { try { var i = 0; while (true) { yield i++ } } finally { s53log.push("closed") } }
function s53add(a, b) { return a + b }
class S53P { constructor(a, b) { this.s = a + "-" + b } }
class S53Q extends S53P { constructor() { super(...s53g()) } }
function s53() {
    check("its01", [...s53g()].length === 2)
    check("its02", [0, ...s53g(), 3].join(",") === "0,1,2,3")
    check("its03", s53add(...s53g()) === 3 && Math.max(...s53g()) === 2)
    check("its04", new S53P(...s53g()).s === "1-2" && new S53Q().s === "1-2")
    var gg = s53g()
    gg.next()
    check("its05", [...gg].join(",") === "2" && [...gg].length === 0)
    var [x, y] = s53g()
    check("its06", x === 1 && y === 2)
    var [p, ...rest] = s53g()
    check("its07", p === 1 && rest.length === 1 && rest[0] === 2)
    var [a, b, c] = s53nat()
    check("its08", a + b + c === 3 && s53log.join(",") === "closed")
    var [d = 9, e = 8, h = 7] = s53g()
    check("its09", d === 1 && e === 2 && h === 7)
    var z1, z2
    [z1, z2] = s53g()
    check("its10", z1 === 1 && z2 === 2)
    function pr([m, n]) { return m * 10 + n }
    check("its11", pr(s53g()) === 12)
}

// ===== SECTION 54: a derived constructor's `this` is what super() answers =====
// A parent constructor that returns an OBJECT replaces the instance: in the derived
// constructor `this` is bound to that object once super() returns, the derived class's
// fields are initialized on it, and `new` answers it - unless the derived constructor
// returns an object of its own. A primitive return is ignored. Values read off node 24.
var s54stash = { tag: "stash" }
class S54A { constructor() { return s54stash } }
class S54B extends S54A { constructor() { super(); this.y = 2 } }
class S54C extends S54A { }
class S54D extends S54A { constructor() { super(); this.z = 3; return undefined } }
class S54E extends S54A { f = 9 }
class S54F extends S54B { constructor() { super(); this.w = 4 } }
class S54P { constructor() { this.p = 1; return 5 } }
class S54Q extends S54P { constructor() { super(); this.q = 2 } }
class S54R extends S54P { g = 7; constructor() { super(); this.h = this.g + 1 } }
function s54() {
    var b = new S54B()
    check("sup01", b === s54stash && b.y === 2 && !(b instanceof S54B))
    check("sup02", new S54C() === s54stash)
    var d = new S54D()
    check("sup03", d === s54stash && d.z === 3)
    var e = new S54E()
    check("sup04", e === s54stash && e.f === 9)
    var f = new S54F()
    check("sup05", f === s54stash && f.w === 4 && Object.keys(s54stash).join(",") === "tag,y,z,f,w")
    var q = new S54Q()
    check("sup06", q.p === 1 && q.q === 2 && q instanceof S54Q)
    var r = new S54R()
    check("sup07", r.g === 7 && r.h === 8 && Object.keys(r).join(",") === "p,g,h")
}

// ===== SECTION 55: JSON.stringify and JSON.parse =====
// One implementation (lib/json.js) serves the interpreter halves and layer 2, with
// abnf/jsrtjson.go as the llvm.Run twin. A malformed text is a SyntaxError in every
// engine, but only "Unexpected end of JSON input" is worded as node words it, so the
// other messages are not asserted. Values read off node 24.
class S55Pt { constructor(x, y) { this.x = x; this.y = y } toJSON(k) { return "Pt" + this.x + this.y + k } }
class S55Plain { constructor() { this.a = 1; this.f = function () {} } m() {} }
function s55() {
    var nz = [0, -0, NaN, Infinity, -Infinity, 1.5, 1e21, 1e-7, 123]
    check("jsn01", JSON.stringify(nz) === "[0,0,null,null,null,1.5,1e+21,1e-7,123]")
    check("jsn02", JSON.stringify({a: 1, b: "x", c: [1, "two", null, true], d: {e: {}}, u: undefined, f: function () {}})
        === '{"a":1,"b":"x","c":[1,"two",null,true],"d":{"e":{}}}')
    var ctl = "q\"b\\s/\n\t\r" + String.fromCharCode(8, 12, 1, 31, 127)
    check("jsn03", JSON.stringify(ctl) === '"q\\"b\\\\s/\\n\\t\\r\\b\\f\\u0001\\u001f' + String.fromCharCode(127) + '"')
    check("jsn04", JSON.stringify([undefined, function () {}, null]) === "[null,null,null]" && JSON.stringify(undefined) === undefined)
    check("jsn05", JSON.stringify(null) + JSON.stringify(true) + JSON.stringify(5) === "nulltrue5")
    check("jsn06", JSON.stringify({a: [1, {b: 2}], c: "d"}, null, 2) === '{\n  "a": [\n    1,\n    {\n      "b": 2\n    }\n  ],\n  "c": "d"\n}')
    check("jsn07", JSON.stringify([1, [2, []], {}], null, "--") === "[\n--1,\n--[\n----2,\n----[]\n--],\n--{}\n]")
    check("jsn08", JSON.stringify([1], null, 20) === "[\n          1\n]" && JSON.stringify([1], null, "abcdefghijkl") === "[\nabcdefghij1\n]")
    check("jsn09", JSON.stringify(new S55Pt(1, 2)) === '"Pt12"' && JSON.stringify({p: new S55Pt(3, 4)}) === '{"p":"Pt34p"}')
    check("jsn10", JSON.stringify(new S55Plain()) === '{"a":1}' && JSON.stringify({toJSON: function () { return 42 }}) === "42")
    // An Error's name, message and stack are not enumerable.
    check("jsn11", JSON.stringify(new Error("m")) === "{}" && JSON.stringify([new TypeError("t")]) === "[{}]")
    check("jsn12", JSON.stringify({a: 1, b: 2, c: 3}, ["c", "a"]) === '{"c":3,"a":1}')
    check("jsn13", JSON.stringify({a: 1, b: "s", c: [1, 2]}, function (k, v) { return typeof v == "number" ? v * 10 : v })
        === '{"a":10,"b":"s","c":[10,20]}')
    var big = anyErr(function () { return JSON.stringify(1n) })
    check("jsn14", big[0] === true && big[2] === "Do not know how to serialize a BigInt")
    var cyc = {a: 1}
    cyc.self = cyc
    check("jsn15", anyErr(function () { return JSON.stringify(cyc) })[0] === true)
    // A lone surrogate is escaped, a pair is copied.
    check("jsn16", JSON.stringify(String.fromCharCode(55357)) === '"\\ud83d"' && JSON.stringify(String.fromCharCode(55357, 56832)).length === 4)
    var p = JSON.parse('{"a": [1, 2.5, -3e2, true, false, null], "b": {"c": "x\\ny\\u0041\\"z"}, "d": ""}')
    check("jsn17", p.a.length === 6 && p.a[1] === 2.5 && p.a[2] === -300 && p.a[3] === true && p.a[5] === null)
    check("jsn18", p.b.c === "x\nyA\"z" && p.d === "" && Object.keys(p).join(",") === "a,b,d")
    check("jsn19", JSON.parse("  7 ") === 7 && JSON.parse('"s"') === "s" && JSON.parse("null") === null && JSON.parse("[]").length === 0)
    check("jsn20", JSON.stringify(JSON.parse('{"x":{"y":[1,{"z":2}]}}')) === '{"x":{"y":[1,{"z":2}]}}' && JSON.parse('{"a":1,"a":2}').a === 2)
    var bad = ["{bad", "[1,", "", "01", "[1,]", "'a'", "tru", '"a', "1 2", '{"a" 1}']
    var nbad = 0
    for (var bi = 0; bi < bad.length; bi++) {
        var r = anyErr(function () { return JSON.parse(bad[bi]) })
        if (r[1] === true) { nbad = nbad + 1 }
    }
    check("jsn21", nbad === 10 && anyErr(function () { return JSON.parse("[1,") })[2] === "Unexpected end of JSON input")
    var rv = JSON.parse('{"a": 1, "b": [2, 3], "c": {"d": 4}}', function (k, v) { return typeof v == "number" ? v + 100 : v })
    check("jsn22", JSON.stringify(rv) === '{"a":101,"b":[102,103],"c":{"d":104}}')
    var rd = JSON.parse('{"a": 1, "b": 2}', function (k, v) { return k === "a" ? undefined : v })
    check("jsn23", JSON.stringify(rd) === '{"b":2}' && Object.keys(rd).join(",") === "b")
    check("jsn24", typeof JSON === "object" && typeof JSON.stringify === "function" && typeof JSON.parse === "function")
}
// ===== SECTION 56: the valid shapes beside the syntax-error guards =====
// The grammars refuse '() => {}()', a template's '${' that is not a substitution,
// a second 'default:' in one switch, 'this' as a whole assignment target and a
// generator expression named 'yield' - parse errors, which only the corpus (test262
// fail/) can hold. What a ratchet CAN hold is that their valid neighbours still run:
// an arrow block ending its statement before a line break, a lone '$' in a template,
// a default clause between cases, 'this' at the head of a member target. Values read
// off node 24.
function s56() {
    var log = []
    var f = () => { log.push("f") }
    ;(function () { log.push("iife") })()
    var g = (() => 5)()
    var h = [() => 6][0]()
    check("grd1", log.join(",") === "iife" && g === 5 && h === 6 && typeof f === "function")
    var d = "$"
    check("grd2", `a$b` === "a$b" && `$${d}$` === "$$$" && `${d}{x}` === "${x}")
    function sw(v) {
        var out = ""
        switch (v) { case 1: out += "1"; default: out += "d"; case 2: out += "2" }
        return out
    }
    check("grd3", sw(1) === "1d2" && sw(2) === "2" && sw(9) === "d2")
    var o = { v: 0, set(x) { this.v = x; this["w"] = x + 1; return this } }
    check("grd4", o.set(4).v === 4 && o.w === 5)
    var gen = function* yielder() { yield 1 }
    check("grd5", gen().next().value === 1 && gen.name === "yielder")
    // An arrow function is an AssignmentExpression, so 'typeof () => {}' and
    // 'a || () => 1' are refused; it still stands wherever one may: parenthesized,
    // as a conditional's branch, a right-hand side, an argument, after a comma.
    var ab = [0, 1]
    var pick = ab[1] ? () => "t" : () => "f"
    var curry = x => y => x + y
    var seq = (ab[0], () => 7)
    check("grd6", typeof (() => {}) === "function" && pick() === "t" && curry(ab[1])(2) === 3
        && seq() === 7 && [1, 2].map(x => x * 2).join() === "2,4" && (ab[0] || (() => 8))() === 8)
    // A statement that starts with '{' is a Block, never an object literal: '{ a() {} }'
    // is refused, while a block holding a label or a comma expression runs.
    var bl = 0
    { lab: bl = bl + 1 }
    { bl, bl = bl + 2 }
    check("grd7", bl === 3 && ({ m() { return 4 } }).m() === 4)
}
// [isTypeError, isSyntaxError, message] of what f throws.
function anyErr(f) {
    try { f(); return [false, false, ""] }
    catch (e) { return [e instanceof TypeError, e instanceof SyntaxError, e.message] }
}

// ===== SECTION 58: Set, Map, Array.from and the Array global =====
// Set, Map, Array.from and Array.of are lib/js-builtins.js, JavaScript compiled into
// every program that names them; a Set or a Map iterates through its "@@iterator"
// method, which every iteration site asks for (this subset has no Symbol.iterator).
// The Array global is the engines' own function: Array(n), new Array(n), isArray.
// Values read off node 24.
function s58() {
    var src = [3, 1, 3, 2]
    function* g58() { yield src[0]; yield src[1]; yield src[0] }
    function err58(f) { try { f(); return "none" } catch (e) { return e.constructor.name + ": " + e.message } }
    check("col1", Array.from([1, 2]).join(",") === "1,2" && Array.from("abc", (c, i) => c + i).join() === "a0,b1,c2"
        && Array.from({length: 3, 0: "a", 2: "c"}).join("|") === "a||c" && Array.from(g58()).join() === "3,1,3")
    check("col2", Array.from([1, 2], function (x) { return x * this.k }, {k: 10}).join() === "10,20"
        && Array.from(src[0]).length === 0 && Array.of(src[0]).length === 1 && Array.of(1, 2, 3).join() === "1,2,3")
    check("col3", err58(() => Array.from(null)) === "TypeError: object null is not iterable (cannot read property Symbol(Symbol.iterator))"
        && err58(() => Array.from([1], src[0])) === "TypeError: number 3 is not a function")
    var s = new Set(g58())
    check("col4", s.size === 2 && Array.from(s).join() === "3,1" && [...s].join() === "3,1" && s.has(1) && !s.has("1"))
    check("col5", new Set([NaN, NaN, 0, -0, "0", undefined, null, undefined]).size === 5 && 1 / [...new Set([-0])][0] === Infinity
        && [...new Set("hello")].join("") === "helo")
    s.add(src[3]).add(src[0])
    check("col6", s.size === 3 && s.delete(1) === true && s.delete(1) === false && [...s].join() === "3,2")
    var seen = []
    for (var v of new Set([1, 2, 3])) { seen.push(v) }
    var live = new Set([1, 2, 3])
    var order = []
    for (var w of live) { order.push(w); if (w === 1) { live.delete(2); live.add(4) } }
    check("col7", seen.join() === "1,2,3" && order.join() === "1,3,4")
    var m = new Map([[1, "a"], ["1", "b"]])
    m.set(src[1], "c").set(src[3], "d")
    check("col8", m.size === 3 && m.get(1) === "c" && m.get("1") === "b" && m.get(2) === "d" && m.get(9) === undefined
        && [...m.keys()].join() === "1,1,2" && [...m.values()].join() === "c,b,d")
    var pairs = []
    for (var [k, val] of m) { pairs.push(k + "=" + val) }
    var each = []
    m.forEach(function (x, y, mm) { each.push(y + ":" + x + (mm === m)) })
    check("col9", pairs.join() === "1=c,1=b,2=d" && each.join() === "1:ctrue,1:btrue,2:dtrue")
    var key = {}
    var om = new Map([[key, 1], [{}, 2]])
    check("col10", om.get(key) === 1 && om.get({}) === undefined && om.size === 2 && new Map([[NaN, 7]]).get(NaN) === 7)
    m.clear()
    check("col11", m.size === 0 && m.get(1) === undefined && err58(() => new Map([1])) === "TypeError: Iterator value 1 is not an entry object"
        && err58(() => new Set(src[0])) === "TypeError: number 3 is not iterable (cannot read property Symbol(Symbol.iterator))")
    function* d58() { yield* new Set([8, 9]) }
    var [a58, b58] = new Set([5, 6, 7])
    check("col12", [...d58()].join() === "8,9" && a58 + b58 === 11 && new Set() instanceof Set && !(new Set() instanceof Map)
        && String(new Set()) === "[object Set]" && JSON.stringify(new Map([[1, 2]])) === "{}" && Object.keys(new Set([1])).length === 0)
    check("col13", typeof Set === "function" && typeof Map === "function" && typeof Array === "function"
        && Map.name + Set.name + Array.from.name + Array.of.name === "MapSetfromof")
    check("col14", Array(src[0]).length === 3 && Array(src[0]).join("-") === "--" && new Array(src[3], src[0]).join() === "2,3"
        && Array("3").length === 1 && Array().length === 0 && [] instanceof Array && Array.isArray([]) && !Array.isArray("x")
        && err58(() => Array(-1)) === "RangeError: Invalid array length" && err58(() => Array(1.5)) === "RangeError: Invalid array length")
    var isArr = Array.isArray
    check("col15", typeof isArr === "function" && [[1], [2]].every(Array.isArray) && !isArr("x"))
    check("col16", Object.keys(Array).length === 0 && typeof Array.from === "function")
}

// ===== SECTION 59: closing a class-instance iterator =====
// An iterator whose return() and throw() are CLASS methods (they live on the class,
// not on the instance) is closed by every site that abandons it: a for-of break, a
// labelled break out of it, a return out of its body, array destructuring that stops
// early, and a yield* delegate's return() / throw(). Values read off node 24.
var log59 = []
class It59 {
    constructor(n) { this.i = 0; this.n = n }
    next() { this.i++; return this.i > this.n ? {done: true, value: undefined} : {done: false, value: this.i} }
    return() { log59.push("ret" + this.i); return {done: true} }
    throw(e) { log59.push("thr"); return {done: true} }
}
// Iterable through a [Symbol.iterator] on the prototype.
if (typeof Symbol !== "undefined") { It59.prototype[Symbol.iterator] = function () { return this } }
function* del59(n) { yield* new It59(n) }
function lab59(n) { outer: for (var x of [1]) { for (var y of new It59(n)) { if (y == 2) { break outer } } } }
function ret59(n) { for (var y of new It59(n)) { if (y == 3) { return y } } }
function s59() {
    var five = [5, 1][0]
    for (var v of new It59(five)) { if (v == 2) { break } }
    var [a, b] = new It59(five)
    var [c] = new It59([5, 1][1])
    check("itc1", log59.join() === "ret2,ret2,ret1" && a + b + c === 4)
    lab59(five)
    ret59(five)
    check("itc2", log59.join() === "ret2,ret2,ret1,ret2,ret3")
    var g = del59(five)
    g.next()
    var r = g.return(7)
    var g2 = del59(five)
    g2.next()
    var t = g2.throw(new Error("x"))
    check("itc3", log59.join() === "ret2,ret2,ret1,ret2,ret3,ret1,thr" && r.done && t.done)
}

// ===== SECTION 60: hasOwnProperty on every kind of receiver =====
// `v.hasOwnProperty(k)` and Object.prototype.hasOwnProperty.call(v, k) on a plain
// object (a slot holding undefined is own), a class instance (its methods are not),
// an array and a string (length and canonical indices), Math and JSON, an Error and a
// function. Values read off node 24.
class C60 { constructor() { this.x = 1 } m() { return 1 } }
function s60() {
    var ks = ["a", "b", "x", "m", "0", "1", "01", "length", "abs", "PI", "parse", "message", "name", "nope"]
    var vals = [{a: 1, b: undefined}, new C60(), [5, 6], Math, JSON, new Error("e"), "ab", 7, function f() {}]
    var h = Object.prototype.hasOwnProperty
    var rows = []
    var same = true
    for (var i = 0; i < vals.length; i++) {
        var row = ""
        for (var j = 0; j < ks.length; j++) {
            var r = vals[i].hasOwnProperty(ks[j])
            if (r !== h.call(vals[i], ks[j])) { same = false }
            row = row + (r ? "1" : "0")
        }
        rows.push(row)
    }
    check("hop1", rows.slice(0, 3).join() === "11000000000000,00100000000000,00001101000000" && same)
    check("hop2", rows.slice(3, 6).join() === "00000000110000,00000000001000,00000000000100")
    check("hop3", rows.slice(6).join() === "00001101000000,00000000000000,00000001000010")
    check("hop4", Math.hasOwnProperty("abs") && !Math.hasOwnProperty("nope") && ({hasOwnProperty: () => 9}).hasOwnProperty() === 9)
}

// ===== SECTION 61: delete through any primary =====
// The object a delete removes from may be any primary: a comma expression, an array
// literal's element, a call's result, a parenthesized name. Values read off node 24.
function s61() {
    var o = {a: 1, b: 2, c: {d: 4, e: 5}, f: 6}
    function g() { return o }
    var r = [delete (0, o).a, delete [o][0].b, delete g().c.d, delete (o).f]
    check("del1", r.join() === "true,true,true,true" && JSON.stringify(o) === '{"c":{"e":5}}' && Object.keys(o).join() === "c")
}

// ===== SECTION 62: Object.prototype's methods on every object =====
// Every object inherits toString, valueOf, hasOwnProperty and propertyIsEnumerable,
// called or read as a value: a literal, a class instance, an Error, Math and JSON
// (whose @@toStringTag names them). propertyIsEnumerable is Object.keys' question, and
// the Object and Array globals have their own non-enumerable names. Values read off
// node 24.
class C62 { constructor() { this.q = 1 } }
function s62() {
    var o = {a: 1}
    var p = new C62()
    var vals = [o, p, new Error("e"), Math, JSON]
    var tags = []
    var types = ""
    for (var i = 0; i < vals.length; i++) {
        tags.push(vals[i].toString())
        types = types + typeof vals[i].toString + typeof vals[i].valueOf
            + typeof vals[i].hasOwnProperty + typeof vals[i].propertyIsEnumerable
    }
    check("opm1", tags.join() === "[object Object],[object Object],Error: e,[object Math],[object JSON]")
    var want = ""
    for (var j = 0; j < 20; j++) { want = want + "function" }
    check("opm2", types === want)
    check("opm3", o.valueOf() === o && p.valueOf() === p && Math.valueOf() === Math && [1, 2].valueOf().length === 2)
    check("opm4", o.propertyIsEnumerable("a") && !o.propertyIsEnumerable("b") && p.propertyIsEnumerable("q")
        && !new Error("x").propertyIsEnumerable("message") && ![1].propertyIsEnumerable("length")
        && [1].propertyIsEnumerable(0) && "ab".propertyIsEnumerable(1))
    check("opm5", String(Math) === "[object Math]" && `${JSON}` === "[object JSON]" && "" + o === "[object Object]"
        && Object.prototype.toString.call(new Error("x")) === "[object Error]")
    var ts = o.toString
    var hp = p.hasOwnProperty
    check("opm6", ts.call([1]) === "[object Array]" && hp.call(p, "q") && !hp.call(o, "q")
        && Object.prototype.propertyIsEnumerable.call(o, "a"))
    check("opm7", Object.hasOwnProperty("keys") && !Object.hasOwnProperty("toString") && Array.hasOwnProperty("isArray")
        && !Array.propertyIsEnumerable("isArray") && Object.keys(Array).length === 0)
    check("opm8", Object.getOwnPropertyNames(Array).join() === "length,name,prototype,isArray,from,fromAsync,of"
        && Object.getOwnPropertyNames(Object).length === 26)
}

// ===== SECTION 63: extends Object =====
// `class A extends Object` is a derived class whose parent constructor does nothing:
// super() is required and returns, a default constructor works, and the heritage may
// also be an expression that evaluates to Object. Values read off node 24.
class A63 extends Object {}
class B63 extends Object { constructor(x) { super(); this.x = x } m() { return this.x + 1 } }
class C63 extends B63 { m() { return super.m() * 10 } }
function o63() { return Object }
class D63 extends o63() { constructor() { super(); this.d = 1 } }
function s63() {
    var a = new A63()
    var c = new C63(2)
    check("xob1", a instanceof A63 && a instanceof Object && Object.keys(a).length === 0 && String(a) === "[object Object]")
    check("xob2", new B63(4).m() === 5 && c.m() === 30 && c instanceof B63 && JSON.stringify(c) === '{"x":2}')
    check("xob3", JSON.stringify(new D63()) === '{"d":1}' && new (class extends (Object) {})() instanceof Object)
}

// ===== SECTION 64: g.return(v) at a yield* answers the delegate's return() =====
// A close at a `yield*` hands v to the delegate's return(), and when that answers
// done the generator returns ITS value - undefined for a bare {done: true} - with the
// generator's own finally clauses still run. A delegate generator closes the same
// way. Values read off node 24.
var log64 = []
function it64(ans) {
    var o = {
        n: 0,
        next() { this.n++; return {value: this.n, done: false} },
        return(v) { log64.push("ret:" + v); return ans }
    }
    if (typeof Symbol !== "undefined") { o[Symbol.iterator] = function () { return this } }
    return o
}
function* d64(it) { try { yield* it } finally { log64.push("fin") } }
function* in64() { try { yield 1 } finally { log64.push("in") } }
function s64() {
    var g = d64(it64({done: true}))
    var a = g.next()
    var b = g.return(7)
    check("ysr1", a.value === 1 && b.value === undefined && b.done === true && log64.join() === "ret:7,fin")
    var h = d64(it64({value: 9, done: true}))
    h.next()
    var c = h.return(7)
    check("ysr2", c.value === 9 && c.done === true && h.next().done === true)
    var k = d64(in64())
    k.next()
    var e = k.return(5)
    check("ysr3", e.value === 5 && e.done === true && log64.slice(4).join() === "in,fin")
}

// ===== SECTION 65: a builtin method read as a value =====
// `[].push`, `"s".charAt` and `(5).toFixed` read without a call are the prototype's
// own function: one per kind and name, shared by every receiver, and taking `this`
// from the call site - `.call` / `.apply` / `.bind` choose it and a bare call has none,
// which is node's TypeError. String's toString/valueOf and a Number method refuse a
// receiver of another kind. Values read off node 24.
function s65() {
    var vals = [[1, 2], "abc", 5]
    var names = ["push", "valueOf", "toString", "charAt", "map", "toFixed", "nope"]
    var types = []
    for (var i = 0; i < vals.length; i++) {
        for (var j = 0; j < names.length; j++) { types.push(typeof vals[i][names[j]]) }
    }
    check("bmv1", types.join() === "function,function,function,undefined,function,undefined,undefined," +
        "undefined,function,function,function,undefined,undefined,undefined," +
        "undefined,function,function,undefined,undefined,function,undefined")
    var a = vals[0]
    var push = a.push
    var into = [9]
    check("bmv2", push.call(into, 3) === 2 && into.join() === "9,3" && a.length === 2 && a.push === [].push &&
        a.valueOf.call(a) === a && "x".slice === "y".slice && a.slice !== "".slice)
    var errs = []
    var bad = [
        function () { var f = [].push; return f(1) },
        function () { return "a".toString.call([1]) },
        function () { return (1).toFixed.call("x") },
        function () { return [].map.call(undefined, function (x) { return x }) },
        function () { return "".trimEnd.call(null) },
    ]
    for (var k = 0; k < bad.length; k++) {
        try { bad[k](); errs.push("none") } catch (e) { errs.push(e.constructor.name + ": " + e.message) }
    }
    check("bmv3", errs.join("|") === "TypeError: Cannot convert undefined or null to object|" +
        "TypeError: String.prototype.toString requires that 'this' be a String|" +
        "TypeError: Number.prototype.toFixed requires that 'this' be a Number|" +
        "TypeError: Array.prototype.map called on null or undefined|" +
        "TypeError: String.prototype.trimRight called on null or undefined")
    var up = "q".toUpperCase.bind("hi")
    check("bmv4", "ab".charAt.call(12345, 1) === "2" && [1, 2, 3].slice.apply([4, 5, 6], [1]).join() === "5,6" &&
        up() === "HI" && (2.5).toFixed.call(1.25, 1) === "1.3" && ["a", "b"].map("x".concat.bind("p")).join() === "pa0a,b,pb1a,b")
    // Array / String / Number.prototype hold those same functions, and Object is a
    // function to typeof.
    var rest = function () { return Array.prototype.slice.call(arguments, 1) }
    check("bmv5", typeof Object === "function" && Object.prototype.toString.call(Object) === "[object Function]" &&
        typeof Array.prototype === "object" && Array.prototype.slice === [].slice && rest(1, 2, 3).join() === "2,3" &&
        String.prototype.toUpperCase.call("ab") === "AB" && "x".slice === String.prototype.slice &&
        Number.prototype.toFixed.call(1.25, 1) === "1.3" && Array.prototype.join.call([4, 5], "-") === "4-5")
}

// ===== SECTION 66: the own names of Math, JSON and Object, and `in` through the prototype =====
// Math's, JSON's and Object's members are own and NOT enumerable: Object.keys and
// for-in see none of them, getOwnPropertyNames lists them in node's order. `k in o`
// also finds what o inherits - Object.prototype's names on every object, an Array's
// methods and length, a function's or a class's call / bind / name / length. Values
// read off node 24.
class C66 { constructor() { this.q = 1 } m() { return 1 } static s() { return 2 } }
function s66() {
    var fk = []
    for (var p in JSON) { fk.push(p) }
    check("own1", Object.keys(JSON).length === 0 && Object.keys(Object).length === 0 && Object.keys(Math).length === 0 &&
        fk.length === 0 && Object.values(JSON).length === 0)
    var mn = Object.getOwnPropertyNames(Math)
    check("own2", mn.length === 44 && mn[0] === "abs" && mn[7] === "atan2" && mn[25] === "pow" && mn[40] === "PI" &&
        Object.getOwnPropertyNames(JSON).join() === "parse,stringify,rawJSON,isRawJSON")
    var f = function (a) { return a }
    var keys = ["toString", "push", "length", "call", "name", "0", "q", "m", "s", "nope", "constructor", "__proto__", "hasOwnProperty"]
    var objs = [{}, [5], f, new C66(), C66]
    var rows = []
    for (var i = 0; i < objs.length; i++) {
        var row = ""
        for (var j = 0; j < keys.length; j++) { row = row + (keys[j] in objs[i] ? 1 : 0) }
        rows.push(row)
    }
    check("own3", rows.join() === "1000000000111,1110010000111,1011100000111,1000001100111,1011100010111")
}

// ===== SECTION 67: super reaches Object.prototype =====
// A class's super chain ends at Object.prototype - with or without `extends Object` -
// so `super.toString()` in a class that overrides toString is "[object Object]",
// super.valueOf / hasOwnProperty / propertyIsEnumerable run on the receiver, and a
// name no level defines is still undefined. Values read off node 24.
class A67 { toString() { return "A:" + super.toString() } t() { return typeof super.toString } n() { return typeof super.nope } }
class B67 extends Object { toString() { return "B:" + super.toString() } v() { return super.valueOf() === this } }
class P67 { }
class Q67 extends P67 { e(k) { return super.propertyIsEnumerable(k) && super.hasOwnProperty(k) } }
function s67() {
    var b = new B67()
    var q = new Q67()
    q.z = 2
    check("sup1", new A67().toString() === "A:[object Object]" && b.toString() === "B:[object Object]" && b.v() &&
        `${b}` === "B:[object Object]")
    check("sup2", q.e("z") && !q.e("y") && new A67().t() === "function" && new A67().n() === "undefined")
}

// ===== SECTION 68: g.return(v) resumes the body with a return completion =====
// The finally clauses around the parked yield run as the body's own code: one that
// returns overrides v, one that yields makes return() answer {value, done: false}
// and the generator goes on from there (a later next() finishes the return, a later
// return() replaces its value, a throw() can be caught), a catch clause never sees
// the completion, and at a yield* a delegate whose return() is not done is yielded
// from and stays delegated to. A generator not yet started runs nothing. Values
// read off node 24.
function* r68a() { try { yield 1 } finally { return 42 } }
function* r68b() { try { yield 1 } finally { yield 99; yield 100 } return 5 }
function* r68c() { try { yield 1 } catch (e) { yield "caught" } finally { yield "fin" } }
function* r68d() { try { yield 1 } finally { try { yield 2 } catch (e) { yield "c:" + e } } }
function* r68in() { try { yield 1; yield 2 } finally { yield "f" } }
function* r68out() { var r = yield* r68in(); return "o:" + r }
function* r68arg(a) { try { yield arguments.length; yield a } finally { } }
function s68() {
    var j = function (r) { return r.value + "/" + r.done }
    var a = r68a()
    a.next()
    var b = r68b()
    b.next()
    var b1 = [j(b.return(7)), j(b.next()), j(b.next()), j(b.next())]
    check("grc1", j(a.return(7)) === "42/true" && j(a.next()) === "undefined/true" &&
        b1.join() === "99/false,100/false,7/true,undefined/true")
    var c = r68c()
    c.next()
    var d = r68d()
    d.next()
    var f = r68b()
    f.next()
    check("grc2", [j(c.return(3)), j(c.next())].join() === "fin/false,3/true" &&
        [j(d.return(5)), j(d.throw("T")), j(d.next()), j(d.next())].join() === "2/false,c:T/false,5/true,undefined/true" &&
        [j(f.return(7)), j(f.return(8)), j(f.next())].join() === "99/false,8/true,undefined/true")
    var o = r68out()
    var h = r68b()
    var w = r68arg(4, 5)
    check("grc3", [j(o.next()), j(o.return(7)), j(o.next()), j(o.next())].join() === "1/false,f/false,o:7/true,undefined/true" &&
        [j(h.return(3)), j(h.next())].join() === "3/true,undefined/true" &&
        [j(w.next()), j(w.next()), j(w.return(1))].join() === "2/false,4/false,1/true")
}

// ===== SECTION 69: where `yield` and `return` stand, deep into a file =====
// Whether a `yield` is an expression or an identifier, and whether a `return` is
// inside a function, depends on the brackets and function heads before it in the
// whole file; this section sits at the END of one, past every generator above. A
// generator's nested function, arrow and generator method, then the identifier again
// once the generator has closed. Values read off node 24.
function* g69() {
    var f = function () { return 1 }
    yield f()
    yield (() => { return 2 })()
    var o = { *m() { yield 7 } }
    yield o.m().next().value
}
function s69() {
    var vals = []
    for (var v of g69()) { vals.push(v) }
    var yield = 3
    function after() { var yield = 5; return yield + 1 }
    check("ig69a", vals.join() === "1,2,7" && yield === 3 && after() === 6)
}

// ===== SECTION 70: prototypes, static methods as values, Boolean, Object.is =====
// A class's and a constructor function's prototype is an object: a function written
// onto it is a method of every instance (and replaces a class method of that name),
// its methods read as values, and `constructor` is the class. The builtin globals'
// prototypes are objects (Array's an array), a static method reads as a function,
// Boolean is ToBoolean and Object.is is SameValue. Values read off node 24.
function F70(x) { this.x = x }
class C70 { m() { return "m" + this.v } constructor() { this.v = 1 } }
class D70 extends C70 {}
class E70 extends Error {}
class G70 { get b() { return 1 } a() {} set b(v) {} static st() {} #q() {} [Symbol.iterator]() {} ["__#@iterator"]() {} *g() {} }
function s70() {
    F70.prototype.get = function () { return this.x }
    const f = new F70(7)
    check("pr70a", typeof F70.prototype === "object" && F70.prototype === F70.prototype && f.get() === 7 &&
        typeof f.get === "function" && ("get" in f) && F70.prototype.constructor === F70 &&
        Object.keys(F70.prototype).join() === "get")
    C70.prototype.n = function () { return "n" + this.v }
    const c = new C70()
    const before = c.m()
    C70.prototype.m = function () { return "p" + this.v }
    check("pr70b", c.n() === "n1" && before === "m1" && c.m() === "p1" && C70.prototype.constructor === C70 &&
        C70.prototype.n.call({v: 6}) === "n6" && new D70().n() === "n1" && typeof D70.prototype.n === "function" &&
        D70.prototype.constructor === D70)
    const p = F70.prototype
    p.twice = function () { return this.x * 2 }
    check("pr70c", new F70(4).twice() === 8 && Error.prototype.name === "Error" && TypeError.prototype.name === "TypeError" &&
        Error.prototype.message === "" && typeof Error.prototype.toString === "function" && E70.prototype.name === "Error" &&
        typeof RangeError.prototype === "object" && typeof Map.prototype.get === "function" && typeof Set.prototype === "object")
    check("pr70d", typeof RegExp.prototype === "object" && typeof Promise.prototype === "object" &&
        typeof BigInt.prototype === "object" && Array.isArray(Array.prototype) && Array.prototype.length === 0 &&
        typeof String.prototype === "object")
    const k = Object.keys
    check("st70e", typeof Object.keys === "function" && typeof Object.assign === "function" && typeof Object.fromEntries === "function" &&
        k({x: 1, y: 2}).join() === "x,y" && Object.keys.name === "keys" && Object.assign.length === 2 && Object.keys === Object.keys &&
        String.fromCharCode === String.fromCharCode && typeof Promise.resolve === "function" &&
        JSON.stringify([{a: 1}, {b: 2}].map(Object.keys)) === '[["a"],["b"]]' && [[1], 2].filter(Array.isArray).length === 1)
    check("bo70f", typeof Boolean === "function" && Boolean.name === "Boolean" && Boolean.length === 1 &&
        [0, 1, "", "a", null, NaN, [], 0n, 5n, -0].map(Boolean).join() === "false,true,false,true,false,false,true,false,true,false" &&
        Boolean() === false)
    check("is70g", Object.is(NaN, 0 / 0) && !Object.is(0, -0) && Object.is(-0, -1 / Infinity) && !Object.is(NaN, "NaN") &&
        Object.is(10n, 10n) && !Object.is({}, {}) && Object.is("a", "a"))
    // getOwnPropertyNames of a prototype: `constructor`, the class body's methods and
    // accessors in source order (no static, private or symbol-keyed one), then what was
    // written onto it; a declared method written again keeps its place.
    const gp = Object.getOwnPropertyNames
    check("pr70h", gp(G70.prototype).join() === "constructor,b,a,__#@iterator,g" && gp(D70.prototype).join() === "constructor" &&
        gp(F70.prototype).join() === "constructor,get,twice" && gp(C70.prototype).join() === "constructor,m,n" &&
        Object.keys(G70.prototype).length === 0)
}

// ===== SECTION 71: symbols, Symbol.iterator, a private method read as a value =====
// A class or an object literal with a [Symbol.iterator] method is iterable by for-of,
// a spread, destructuring, Array.from and yield*. A symbol is its own type, unique per
// Symbol() call, registered by Symbol.for, a property key that no key listing shows.
// `this.#m` read without a call is the method. Values read off node 24.
class R71 {
    constructor(n) { this.n = n }
    [Symbol.iterator]() {
        let i = 0
        const n = this.n
        return { next() { i++; return i <= n ? { value: i, done: false } : { value: undefined, done: true } } }
    }
}
function* g71() { yield* new R71(2) }
class P71 { #m() { return 1 } get() { const f = this.#m; return typeof f + f.call(this) } }
function s71() {
    const out = []
    for (const v of new R71(3)) { out.push(v) }
    const [x, y] = new R71(5)
    check("sy71a", out.join() === "1,2,3" && [...new R71(3)].join() === "1,2,3" && Array.from(new R71(2)).join() === "1,2" &&
        x === 1 && y === 2 && [...g71()].join() === "1,2")
    const obj = { *[Symbol.iterator]() { yield 7; yield 8 } }
    check("sy71b", [...obj].join() === "7,8" && Object.keys(obj).length === 0 && JSON.stringify(obj) === "{}")
    const s1 = Symbol("k")
    const o2 = {}
    o2[s1] = 5
    o2.plain = 1
    const ks = []
    for (const k in o2) { ks.push(k) }
    check("sy71c", typeof Symbol === "function" && typeof Symbol.iterator === "symbol" && typeof s1 === "symbol" &&
        String(Symbol.iterator) === "Symbol(Symbol.iterator)" && Symbol("hi").toString() === "Symbol(hi)" &&
        s1.description === "k" && Symbol().description === undefined && o2[s1] === 5 && (s1 in o2) &&
        Object.keys(o2).join() === "plain" && ks.join() === "plain" && Symbol("k") !== s1 &&
        JSON.stringify({ x: s1, y: 1 }) === '{"y":1}')
    check("sy71d", Symbol.for("app") === Symbol.for("app") && Symbol.keyFor(Symbol.for("app")) === "app" &&
        Symbol.keyFor(s1) === undefined && Symbol.iterator === Symbol.iterator && typeof new R71(1)[Symbol.iterator] === "function" &&
        new Map([[s1, "v"]]).get(s1) === "v" && new P71().get() === "function1")
}

// ===== SECTION 73: with =====
// A name inside a with body is the object's property when the object has it - read,
// written, compounded, incremented, destructured, called with the object as `this` -
// and the ordinary binding otherwise; a property added inside the body is seen at
// once, a let in the body and a parameter shadow the object, a closure keeps the
// object, and null is a TypeError. Values read off node 24.
function f73(o, a) { with (o) { return a + b } }
function mk73(q) { with (q) { return function () { return v } } }
function s73() {
    var o = { a: 1, b: 2, get2() { return this.b * 2 } }
    var a = 100
    var outer = 5
    var seen = ""
    with (o) {
        seen = a + " " + b + " " + outer + " " + get2()
        a = 10
        outer = 6
        b++
        b += 5
    }
    check("wi73a", seen === "1 2 5 4" && o.a === 10 && a === 100 && outer === 6 && o.b === 8)
    var p = {}
    var late = ""
    with (p) {
        p.added = 7
        late = added + " " + typeof added + " " + typeof nope
    }
    var q = { v: 1 }
    var g = mk73(q)
    q.v = 42
    var shadow = 0
    with ({ x: 1 }) { let x = 2; shadow = x }
    check("wi73b", late === "7 number undefined" && g() === 42 && shadow === 2 && f73({ a: 1, b: 2 }, 10) === 3 &&
        f73({ b: 3 }, 10) === 13)
    var mathed = 0
    with (Math) { mathed = max(3, 9) + floor(PI) }
    var r = { s: "x", m: 0, n: 0 }
    with (r) { s += "y"; [m, n] = [7, 8] }
    var nested = ""
    with ({ k: 1 }) { with ({ k: 2, j: 3 }) { nested = k + " " + j } nested = nested + " " + k }
    var err = ""
    try { with (null) { } } catch (e) { err = e.constructor.name + ": " + e.message }
    check("wi73c", mathed === 12 && r.s === "xy" && r.m === 7 && r.n === 8 && nested === "2 3 1" &&
        err === "TypeError: Cannot convert undefined or null to object")
}

// ===== SECTION 74: var is function scoped and hoisted =====
// A var declared in any nested block (if, a bare block, a loop body, try / catch /
// finally, switch) is one binding of the whole function, bound to undefined from its
// entry: readable after the block, before its declaration, by a closure made earlier.
// A parameter of the same name keeps its argument, a var without an initializer
// assigns nothing, and an assignment inside a catch clause whose parameter it names
// lands on the parameter. A for-in/of head without a keyword assigns the existing
// variable, with var the function's. Values read off node 24.
function s74() {
    const f1 = function () { if (true) { var w = 3 } { var z = 4 } for (var i = 0; i < 2; i++) { var q = i } return w + z + q + i }
    const f2 = function () { try { var t1 = 5 } catch (e) { var t2 = 6 } finally { var t3 = 7 } switch (1) { case 1: var t4 = 8 } return t1 + t3 + t4 + (t2 === undefined ? 100 : 0) }
    const f3 = function (a) { var r = typeof later; var a; var later = 1; var g = function () { return h }; var h = 9; return r + a + g() }
    const f4 = function () { try { throw 1 } catch (e) { var e = 2 } return e }
    const f5 = function () { var n = 1; { var n = 2 } var n; return n }
    check("vh74a", f1() === 10 && f2() === 120 && f3(7) === "undefined79" && f4() === undefined && f5() === 2)
    const f6 = function () { var k = 0; for (k of [5, 6]) {} let m = ""; for (m in { a: 1, b: 2 }) {} return k + m }
    const f7 = function () { for (var [a, b] of [[1, 2], [3, 4]]) {} for (var p in { x: 1 }) {} var [c, { d }] = [5, { d: 6 }]; return a + b + p + c + d }
    const f8 = function () { const fs = []; for (var i = 0; i < 3; i++) { fs.push(() => i) } return fs.map(f => f()).join() }
    const f9 = () => { { var av = 5 } return av }
    check("vh74b", f6() === "6b" && f7() === "7x56" && f8() === "3,3,3" && f9() === 5)
    // A default value's closure sees the parameters, not the body's vars.
    const dx = "outer"
    const f10 = function (a = () => dx) { var dx = "inner"; return a() + "," + dx }
    const f11 = function (p, q = p + 1) { var p; var r = q; return p + r }
    check("vh74c", f10() === "outer,inner" && f11(4) === 9)
}

// ===== SECTION 75: a symbol is a primitive ToString and ToNumber refuse =====
// `"" + s`, `${s}`, `s * 1` and `s < 1` throw where String(s) and s.toString() answer;
// a symbol is == to itself alone. Object spread and Object.assign copy symbol-keyed
// properties, which no key listing shows. Symbol has all thirteen well-known symbols,
// and an array's or a string's [Symbol.iterator]() is an iterator over its elements
// (a string's code points). Values read off node 24.
function s75() {
    const err = (f) => { try { return String(f()) } catch (e) { return e.constructor.name + ": " + e.message } }
    const v = [Symbol("k"), 1, "a", {}]
    check("sy75a", err(() => "" + v[0]) === "TypeError: Cannot convert a Symbol value to a string" &&
        err(() => v[0] + v[3]) === "TypeError: Cannot convert a Symbol value to a string" &&
        err(() => `${v[0]}`) === "TypeError: Cannot convert a Symbol value to a string" &&
        err(() => v[0] + v[1]) === "TypeError: Cannot convert a Symbol value to a number" &&
        err(() => v[0] * v[1]) === "TypeError: Cannot convert a Symbol value to a number" &&
        err(() => v[0] < v[1]) === "TypeError: Cannot convert a Symbol value to a number" &&
        err(() => +v[0]) === "TypeError: Cannot convert a Symbol value to a number" &&
        String(v[0]) === "Symbol(k)" && v[0].toString() === "Symbol(k)" &&
        !(v[0] == "Symbol(k)") && v[0] == v[0] && v[0] != v[1])
    const k2 = Symbol("two")
    const o = { a: 1, [v[0]]: 3 }
    o[k2] = 2
    const sp = { ...o }
    const as = Object.assign({}, o)
    delete o[k2]
    check("sy75b", sp[v[0]] === 3 && sp[k2] === 2 && Object.keys(sp).join() === "a" && as[v[0]] === 3 && as[k2] === 2 &&
        JSON.stringify(as) === '{"a":1}' && { ...o }[k2] === undefined && !(k2 in o) && (v[0] in o))
    const wk = ["asyncIterator", "hasInstance", "isConcatSpreadable", "iterator", "match", "matchAll", "replace",
        "search", "species", "split", "toPrimitive", "toStringTag", "unscopables"]
    check("sy75c", wk.every(n => typeof Symbol[n] === "symbol" && Symbol[n].description === "Symbol." + n && Symbol[n] === Symbol[n]))
    const it = [7, 8][Symbol.iterator]()
    const a1 = it.next().value
    const a2 = it.next().value
    const si = "a\u{1F600}b"[Symbol.iterator]()
    check("sy75d", typeof [1][Symbol.iterator] === "function" && a1 === 7 && a2 === 8 && it.next().done === true &&
        it[Symbol.iterator]() === it && si.next().value === "a" && si.next().value.length === 2 &&
        [...[3, 4][Symbol.iterator]()].join() === "3,4")
    const num = "TypeError: Cannot convert a Symbol value to a number"
    check("sy75e", err(() => -v[0]) === num && err(() => { let q = v[0]; q++; return q }) === num &&
        err(() => ~v[0]) === num && -v[1] === -1 && Object.is(-(-0), 0))
}

// ===== SECTION 76: the iteration protocol and the well-known symbols consulted =====
// for-of, spread, Array.from, array destructuring and yield* walk a string by code
// point (a surrogate pair is one element, a lone surrogate one of its own). An array's
// values() / keys() / entries() are iterators reading it live, and a generator object's
// [Symbol.iterator]() is itself. [Symbol.toPrimitive] is asked with the hint ("string"
// for a template and String(), "default" for + and ==, "number" for <), an object
// result is a TypeError; [Symbol.hasInstance] decides instanceof and
// [Symbol.toStringTag] names the object in Object.prototype.toString. Values read off
// node 24.
function s76() {
    const strs = ["a\u{1F600}b", "\uD800z"]
    const s0 = strs[0]
    const lens = []
    for (const ch of s0) { lens.push(ch.length) }
    const [p, q] = strs[0]
    function* gs() { yield* strs[0] }
    check("it76a", lens.join() === "1,2,1" && [...s0].length === 3 && Array.from(s0).length === 3 &&
        q.length === 2 && p === "a" && [...gs()].length === 3 && [...strs[1]].length === 2)
    const a = [7, 8]
    const v = a.values()
    const r1 = v.next()
    a.push(9)
    const ent = []
    for (const [i, x] of a.entries()) { ent.push(i + "=" + x) }
    check("it76b", typeof v.next === "function" && r1.value === 7 && r1.done === false && v.next().value === 8 &&
        v.next().value === 9 && v.next().done === true && v[Symbol.iterator]() === v &&
        [...a.keys()].join() === "0,1,2" && ent.join() === "0=7,1=8,2=9" && JSON.stringify([1].entries().next()) === '{"value":[0,1],"done":false}')
    function* g() { yield 1; yield 2 }
    const go = g()
    check("it76c", typeof go[Symbol.iterator] === "function" && go[Symbol.iterator]() === go &&
        Array.from(g()).join() === "1,2")
    class D { [Symbol.toPrimitive](h) { return h === "number" ? 7 : "D:" + h } }
    const d = [new D()][0]
    let acc = d
    acc += "x"
    const err = (f) => { try { return String(f()) } catch (e) { return e.constructor.name + ": " + e.message } }
    check("tp76d", `${d}` === "D:string" && String(d) === "D:string" && d + 1 === "D:default1" && d == "D:default" &&
        d < 8 && !(d > 7) && [d].join() === "D:string" && acc === "D:defaultx" &&
        err(() => "" + { [Symbol.toPrimitive]() { return {} } }) === "TypeError: Cannot convert object to primitive value")
    class Even { static [Symbol.hasInstance](n) { return n % 2 === 0 } }
    class T2 { get [Symbol.toStringTag]() { return "T2" } }
    const tagged = { [Symbol.toStringTag]: "Tagged" }
    check("wk76e", (2 instanceof Even) && !(3 instanceof Even) && Object.prototype.toString.call(tagged) === "[object Tagged]" &&
        String(tagged) === "[object Tagged]" && Object.prototype.toString.call(new T2()) === "[object T2]" && `${new T2()}` === "[object T2]")
}

// ===== SECTION 77: class static order, the global isNaN / isFinite, GetMethod =====
// Static fields and static blocks run in source order, after the methods exist and with
// `this` bound to the class. The global isNaN / isFinite convert with ToNumber first.
// A well-known-symbol member that is present but not callable is a TypeError. Values
// read off node 24.
function s77() {
    const sv = [0, 2, 10]
    class K1 { static { K1.st = sv[1] } static st = sv[0] }
    class K2 { static a = K2.f() + 1; static f() { return sv[2] } }
    class K3 { static a = this.name; static b = this.a + "!"; static { this.c = this.b + "?" } }
    const log = []
    class K4 { static x = (log.push("x"), 1); static { log.push("blk") } static y = (log.push("y"), 2) }
    check("st77a", K1.st === 0 && K2.a === 11 && K3.a === "K3" && K3.b === "K3!" && K3.c === "K3!?" &&
        log.join() === "x,blk,y" && Object.keys(K4).join() === "x,y")
    const nv = ["abc", "12", NaN, 3, "1e3", " 7 ", undefined, null, true, "", Infinity, "-Infinity"]
    check("nn77b", nv.map(isNaN).join() === "true,false,true,false,false,false,true,false,false,false,false,false" &&
        nv.map(isFinite).join() === "false,true,false,true,true,true,false,true,true,true,false,false" &&
        typeof isNaN === "function" && isNaN.length === 1)
    const err = (f) => { try { return String(f()) } catch (e) { return e.constructor.name + ": " + e.message } }
    const bad = [1, "x", {}, true, 2.5]
    check("gm77c", bad.map((b) => err(() => "" + { [Symbol.toPrimitive]: b })).join("|") ===
        'TypeError: number 1 is not a function|TypeError: string "x" is not a function|TypeError: object is not a function|TypeError: boolean true is not a function|TypeError: number 2.5 is not a function' &&
        err(() => ({}) instanceof { [Symbol.hasInstance]: bad[0] }) === "TypeError: number 1 is not a function" &&
        "" + { [Symbol.toPrimitive]: undefined } === "[object Object]")
}

// ===== SECTION 78: ToNumber of an object and of a string =====
// Every numeric operator, Number(), the Math members, isNaN and ++ convert an object
// with ToPrimitive (hint number: [Symbol.toPrimitive], else valueOf, else toString)
// and a string with StringToNumber: white space trimmed, "" is 0, an unsigned
// 0x / 0o / 0b integer, a signed decimal literal or Infinity, anything else NaN.
// == and the relational operators read a string against a number the same way;
// parseFloat takes the longest decimal prefix. Values read off node 24.
function s78() {
    class TP { [Symbol.toPrimitive](h) { return h === "number" ? 7 : "TP" } }
    const ov = [{ valueOf() { return 5 } }, new TP(), [5], [], {}, ["0x10"]]
    const conv = (o) => { let p = o; p++; let q = o; q -= 1; return [+o, -o, o * 2, o - 1, o / 2, o % 3, o ** 2, o | 0, o << 1, ~o, o >>> 0, p, q, Number(o), Math.max(o, 1), Math.floor(o), isNaN(o), o < 6].join(" ") }
    check("tn78a", ov.map(conv).join("|") ===
        "5 -5 10 4 2.5 2 25 5 10 -6 5 6 4 5 5 5 false true|7 -7 14 6 3.5 1 49 7 14 -8 7 8 6 7 7 7 false false|" +
        "5 -5 10 4 2.5 2 25 5 10 -6 5 6 4 5 5 5 false true|0 0 0 -1 0 0 0 0 0 -1 0 1 -1 0 1 0 false true|" +
        "NaN NaN NaN NaN NaN NaN NaN 0 0 -1 0 NaN NaN NaN NaN NaN true false|16 -16 32 15 8 1 256 16 32 -17 16 17 15 16 16 16 false false")
    const sv = ["0x10", " 0b11 ", "0o17", "0X1f", "-0x10", "1_0", "inf", "Infinity", "-Infinity", "1e400", " 12﻿", "0x", "1e", ".", "+.5e1", "", " ", "12px", "5."]
    check("tn78b", sv.map((s) => Number(s)).join() === "16,3,15,31,NaN,NaN,NaN,Infinity,-Infinity,Infinity,12,NaN,NaN,NaN,5,0,0,NaN,5" &&
        sv.map((s) => s * 1).join() === sv.map((s) => +s).join() && sv.map((s) => s | 0).join() === "16,3,15,31,0,0,0,0,0,0,12,0,0,0,5,0,0,0,5" &&
        sv.map(isNaN).join() === sv.map((s) => Number(s) !== Number(s)).join())
    const e = [sv[0], true, "1", "0x0", 16]
    check("tn78c", e[0] == 16 && 16 == e[0] && e[1] == e[2] && !(e[1] == e[3]) && "10" < "9" && !("10" < 9) && e[0] > 15 && e[0] <= e[4] &&
        Math.min(e[0], ov[0]) === 5 && Math.imul(ov[0], 2) === 10 && Math.clz32(ov[0]) === 29 && Math.hypot(ov[0], "12") === 13)
    const pf = ["Infinity", "-Infinityx", "  +1.5e3abc", "-.5", "e5", "1e400", "  7", "1e", "0x10", ".e1"]
    check("pf78d", pf.map((s) => parseFloat(s)).join() === "Infinity,-Infinity,1500,-0.5,NaN,Infinity,7,1,0,NaN" &&
        parseFloat({ toString() { return "3.5" } }) === 3.5 && Object.is(parseFloat("-0"), -0))
}

// ===== SECTION 79: Number#toFixed, toExponential and toPrecision =====
// Each rounds the EXACT binary value half up (1.005 is 1.00499999999999989...), takes
// its digit count through ToIntegerOrInfinity and raises a RangeError outside its
// range; toFixed answers ToString from 1e21 on, toExponential() the shortest digits,
// toPrecision() ToString, and toPrecision switches to exponent form below 1e-6 or at
// p digits. Values read off node 24.
function s79() {
    const xs = [0, -0, 1.5, 2.5, -2.5, 0.5, 1.25, 1.005, 123.456, 0.000123, 1e21, 5e-324, 1 / 3, -1234.5678, NaN, -Infinity, 1e-7, 999.995, -0.4]
    const fx = xs.map((x) => x.toFixed() + "/" + x.toFixed(2)).join(" ")
    check("nf79a", fx === "0/0.00 0/0.00 2/1.50 3/2.50 -3/-2.50 1/0.50 1/1.25 1/1.00 123/123.46 0/0.00 1e+21/1e+21 0/0.00 0/0.33 " +
        "-1235/-1234.57 NaN/NaN -Infinity/-Infinity 0/0.00 1000/1000.00 -0/-0.40" && xs[7].toFixed(20) === "1.00499999999999989342" &&
        xs[11].toFixed(100).length === 102)
    const ex = xs.map((x) => x.toExponential() + "/" + x.toExponential(0) + "/" + x.toExponential(3)).join(" ")
    check("nf79b", ex === "0e+0/0e+0/0.000e+0 0e+0/0e+0/0.000e+0 1.5e+0/2e+0/1.500e+0 2.5e+0/3e+0/2.500e+0 -2.5e+0/-3e+0/-2.500e+0 " +
        "5e-1/5e-1/5.000e-1 1.25e+0/1e+0/1.250e+0 1.005e+0/1e+0/1.005e+0 1.23456e+2/1e+2/1.235e+2 1.23e-4/1e-4/1.230e-4 " +
        "1e+21/1e+21/1.000e+21 5e-324/5e-324/4.941e-324 3.333333333333333e-1/3e-1/3.333e-1 -1.2345678e+3/-1e+3/-1.235e+3 " +
        "NaN/NaN/NaN -Infinity/-Infinity/-Infinity 1e-7/1e-7/1.000e-7 9.99995e+2/1e+3/1.000e+3 -4e-1/-4e-1/-4.000e-1")
    const pr = xs.map((x) => x.toPrecision() + "/" + x.toPrecision(1) + "/" + x.toPrecision(4)).join(" ")
    check("nf79c", pr === "0/0/0.000 0/0/0.000 1.5/2/1.500 2.5/3/2.500 -2.5/-3/-2.500 0.5/0.5/0.5000 1.25/1/1.250 1.005/1/1.005 " +
        "123.456/1e+2/123.5 0.000123/0.0001/0.0001230 1e+21/1e+21/1.000e+21 5e-324/5e-324/4.941e-324 0.3333333333333333/0.3/0.3333 " +
        "-1234.5678/-1e+3/-1235 NaN/NaN/NaN -Infinity/-Infinity/-Infinity 1e-7/1e-7/1.000e-7 999.995/1e+3/1000 -0.4/-0.4/-0.4000" &&
        xs[12].toPrecision(21) === "0.333333333333333314830")
    const err = (f) => { try { return f() } catch (e) { return e.constructor.name + ": " + e.message } }
    const y = xs[2]
    const tp = y.toPrecision
    check("nf79d", err(() => y.toFixed(101)) === "RangeError: toFixed() digits argument must be between 0 and 100" &&
        err(() => y.toExponential(-1)) === "RangeError: toExponential() argument must be between 0 and 100" &&
        err(() => y.toPrecision(0)) === "RangeError: toPrecision() argument must be between 1 and 100" &&
        err(() => xs[14].toPrecision(0)) === "NaN" && y.toFixed("2") === "1.50" && y.toFixed(2.7) === "1.50" &&
        y.toExponential(null) === "2e+0" && typeof y.toExponential === "function" && tp.call(xs[3], 2) === "2.5")
}

// ===== SECTION 80: function declarations are hoisted =====
// A function declaration is instantiated when its scope is entered, before any
// statement runs: callable above it in a function body and at the top level, and in a
// block from the block's first statement. Sloppy code also gives a plain function
// declared in a block (a switch, an if body) a var of the function, undefined until
// the declaration is evaluated in place, which then copies the block's binding - not
// when a let, const or class of the name encloses it, a parameter has the name, or
// the code is strict, and never for a generator. Values read off node 24.
var s80top = typeof s80tf + s80tf()
function s80tf() { return 4 }
{ function s80tb() { return 5 } }
function s80() {
    const f1 = function () { const r = typeof g + g(); function g() { return 1 } return r }
    const f2 = function () { var out = typeof h; { out += typeof h + h(); function h() { return 2 } } return out + typeof h }
    const f3 = function () { let h = 1; { function h() {} } return h }
    const f4 = function (h) { { function h() {} } return h }
    const f5 = function () { "use strict"; { function h() {} } return typeof h }
    check("fh80a", f1() === "function1" && f2() === "undefinedfunction2function" && f3() === 1 && f4(7) === 7 &&
        f5() === "undefined" && s80top === "function4" && s80tb() === 5)
    const f6 = function () { { function h() { return 1 } { function h() { return 2 } } } return h() }
    const f7 = function (k) { switch (k) { case 1: return typeof q; case 2: function q() {} } return typeof q }
    const f8 = function () { { function* gg() {} } return typeof gg }
    const f9 = function () { { h = 5; function h() {} } return h }
    const f10 = function () { function d() { return 1 } function d() { return 2 } return d() }
    check("fh80b", f6() === 2 && f7(1) === "function" && f7(2) === "function" && f7(3) === "undefined" &&
        f8() === "undefined" && f9() === 5 && f10() === 2)
    const f11 = function () { var v = typeof v2; var v2 = 3; function v2() {} return v + v2 }
    const f12 = function (c) { var r = typeof ib; if (c) function ib() {} return r + typeof ib }
    const f13 = function () { try { { function th() { return "t" } } } finally {} return th() }
    const f14 = function () { function h() { return "top" } { function h() { return "blk" } } return h() }
    const f15 = () => { return af(); function af() { return "arrow" } }
    const f16 = function () { var r; try { throw 1 } catch (h) { { function h() {} } r = typeof h } return r + typeof h }
    const f17 = function () { var fs = []; for (var i = 0; i < 2; i++) { let j = i; fs.push(lf); function lf() { return j } } return fs[0]() + fs[1]() + (fs[0] === fs[1] ? "" : "!") }
    check("fh80c", f11() === "function3" && f12(true) === "undefinedfunction" && f12(false) === "undefinedundefined" &&
        f13() === "t" && f14() === "blk" && f15() === "arrow" && f16() === "numberfunction" && f17() === "1!")
}

// ===== SECTION 81: a generator pulling an iterator made outside it =====
// Each pull a generator makes on an iterator it was handed advances that iterator
// once, however the generator is driven: a for-of, a spread, a destructuring, a
// yield* and an explicit next() over an array iterator, a Map iterator, another
// generator and a hand-written iterator (whose throw is caught where it happened),
// interleaved with the caller's own pulls. An iterator the generator makes itself
// starts over each time it is made. Values read off node 24.
function* s81each(it) { for (const x of it) yield x }
function* s81two(s) { yield s.next().value; yield s.next().value; yield* s }
function* s81pat(it) { const [a, b] = it; yield a; yield b; yield [...it].join("+") }
function* s81own() { const it = [1, 2, 3][Symbol.iterator](); yield it.next().value; yield it.next().value; yield it.next().value }
function* s81wrap(src) { const w = { next() { return src.next() } }; for (const x of { [Symbol.iterator]() { return w } }) yield x }
function* s81nums() { yield 10; yield 20; yield 30; yield 40 }
class S81Flaky {
    constructor(n) { this.n = n; this.i = 0 }
    next() { this.i++; if (this.i === 2) throw "boom"; return this.i > this.n ? { done: true, value: undefined } : { done: false, value: this.i } }
    [Symbol.iterator]() { return this }
}
function* s81try(it) {
    let r = ""
    while (true) {
        try { const s = it.next(); if (s.done) break; r += s.value; yield r }
        catch (e) { r += "!" + e; yield r }
    }
}
function s81() {
    check("gp81a", [...s81each([5, 6, 7, 8][Symbol.iterator]())].join(",") === "5,6,7,8")
    const it = [1, 2, 3, 4, 5, 6][Symbol.iterator]()
    const g = s81each(it)
    const seen = [g.next().value, it.next().value, g.next().value, g.next().value, it.next().value, g.next().done]
    check("gp81b", seen.join(",") === "1,2,3,4,5,false")
    check("gp81c", [...s81two(s81nums())].join(",") === "10,20,30,40")
    let n = 0
    const user = { next() { n++; return n > 4 ? { done: true } : { value: n * 100, done: false } }, [Symbol.iterator]() { return this } }
    check("gp81d", [...s81each(user)].join(",") === "100,200,300,400" && n === 5)
    check("gp81e", [...s81each(new Map([["a", 1], ["b", 2], ["c", 3]]).keys())].join(",") === "a,b,c")
    check("gp81f", [...s81pat([7, 8, 9, 10][Symbol.iterator]())].join(",") === "7,8,9+10")
    check("gp81g", [...s81own()].join(",") === "1,2,3")
    check("gp81h", [...s81wrap([3, 1, 4, 1, 5][Symbol.iterator]())].join(",") === "3,1,4,1,5")
    const ext = s81nums()
    const h = s81each(ext)
    check("gp81i", h.next().value + " " + ext.next().value + " " + h.next().value + " " + [...h].join(",") === "10 20 30 40")
    check("gp81j", [...s81try(new S81Flaky(4))].join(",") === "1,1!boom,1!boom3,1!boom34")
}

// ===== SECTION 82: a generator resuming itself, padStart's limit, a missing method, labeled jumps across try =====
// A generator body that resumes itself gets a catchable TypeError and the generator
// goes on; padStart / padEnd past V8's string length limit raise a RangeError instead
// of building the string; a method a builtin object lacks is a catchable TypeError; a
// labeled break or continue that crosses several try bodies closes every for-of
// iterator it leaves, innermost first, each after the finally around it. Values read
// off node 24.
var s82me = null
function* s82self() {
    let r = "none"
    try { s82me.next() } catch (e) { r = (e instanceof TypeError) + " " + e.message }
    yield r
    yield 2
}
function s82iter(name, log) {
    let i = 0
    const it = { next() { i++; return { value: i, done: i > 3 } }, return() { log.push("close " + name); return { done: true } } }
    it[Symbol.iterator] = function () { return it }
    return it
}
function s82() {
    s82me = s82self()
    check("gr82a", s82me.next().value === "true Generator is already running")
    check("gr82b", s82me.next().value === 2 && s82me.next().done === true)
    let pe = ""
    try { "abc".padStart(Infinity, "*") } catch (e) { pe = e.name + ": " + e.message }
    check("ps82a", pe === "RangeError: Invalid string length")
    pe = ""
    try { "abc".padEnd(536870889, "*") } catch (e) { pe = e.name + ": " + e.message }
    check("ps82b", pe === "RangeError: Invalid string length")
    check("ps82c", "abc".padStart(Infinity, "") === "abc" && "abc".padEnd(-Infinity, "*") === "abc")
    let me = ""
    try { String.nope(65) } catch (e) { me = e.name + ": " + e.message }
    check("mm82a", me === "TypeError: String.nope is not a function")
    me = ""
    try { /a/.nope(1) } catch (e) { me = e.name + ": " + e.message }
    check("mm82b", me === "TypeError: /a/.nope is not a function")
    me = ""
    const big = 5n
    try { big.nope(1) } catch (e) { me = e.name + ": " + e.message }
    check("mm82c", me === "TypeError: big.nope is not a function")
    let log = []
    outer: for (const a of s82iter("A", log)) {
        try {
            for (const b of s82iter("B", log)) {
                try {
                    for (const c of s82iter("C", log)) {
                        try { for (const d of s82iter("D", log)) { break outer } } finally { log.push("f3") }
                    }
                } finally { log.push("f2") }
            }
        } finally { log.push("f1") }
    }
    check("lb82a", log.join(",") === "close D,f3,close C,f2,close B,f1,close A")
    log = []
    outer2: for (const a of s82iter("A", log)) {
        try {
            for (const b of s82iter("B", log)) {
                try { for (const c of s82iter("C", log)) { continue outer2 } } finally { log.push("f2") }
            }
        } finally { log.push("f1") }
    }
    check("lb82b", log.join(",") === "close C,f2,close B,f1,close C,f2,close B,f1,close C,f2,close B,f1")
    log = []
    blk: {
        try {
            for (const b of s82iter("B", log)) {
                try { for (const c of s82iter("C", log)) { break blk } } finally { log.push("f2") }
            }
        } finally { log.push("f1") }
    }
    check("lb82c", log.join(",") === "close C,f2,close B,f1")
}

// ===== SECTION 83: Number#toString(radix), parseInt in an odd radix, JSON.parse's errors, String.fromCodePoint =====
// toString(radix) is V8's DoubleToRadixCString: fraction digits only while they are
// significant, the last rounded half to even, the integer part zero-filled past 2^53
// times the radix. parseInt in a radix other than 10 or a power of two folds V8's
// digit chunks in with one rounding each. JSON.parse names the rule a malformed text
// breaks, with its position, line and column, or quotes the text around an
// unexpected character. Values read off node 24.
function s83err(f) {
    try { f(); return "no error" } catch (e) { return e.name + ": " + e.message }
}
function s83() {
    check("rx83a", (1e21).toString(36) === "5v1j4f4ds7c000" && Math.pow(2, 60).toString(7) === "2031000661631341064200")
    check("rx83b", (0.1).toString(3) === "0.0022002200220022002200220022002201" && (-255.5).toString(7) === "-513.33333333333333334")
    check("rx83c", (1 / 3).toString(2) === "0.010101010101010101010101010101010101010101010101010101" && (0.5).toString(36) === "0.i")
    check("rx83d", s83err(function () { (5).toString(1) }) === "RangeError: toString() radix argument must be between 2 and 36")
    check("pi83a", String(parseInt("1000000000000000000000", 7)) === "558545864083284030")
    check("pi83b", String(parseInt("001210100002101012010100020021201011200121011222102221000020101", 3)) === "7.589052697856112e+28")
    check("js83a", s83err(function () { JSON.parse("{bad") }) === "SyntaxError: Expected property name or '}' in JSON at position 1 (line 1 column 2)")
    check("js83b", s83err(function () { JSON.parse("[1 2]") }) === "SyntaxError: Expected ',' or ']' after array element in JSON at position 3 (line 1 column 4)")
    check("js83c", s83err(function () { JSON.parse("\"abc") }) === "SyntaxError: Unterminated string in JSON at position 4 (line 1 column 5)")
    check("js83d", s83err(function () { JSON.parse("abc") }) === "SyntaxError: Unexpected token 'a', \"abc\" is not valid JSON")
    check("js83e", s83err(function () { JSON.parse("[1,2,3,4,5,6,7,8,9,10,11,12,13,14,x]") }) === "SyntaxError: Unexpected token 'x', ...\",12,13,14,x]\" is not valid JSON")
    check("js83f", s83err(function () { JSON.parse("{\n\"a\": 1,\n\"b\" 2\n}") }) === "SyntaxError: Expected ':' after property name in JSON at position 14 (line 3 column 5)")
    check("js83g", s83err(function () { JSON.parse("01") }) === "SyntaxError: Unexpected number in JSON at position 1 (line 1 column 2)")
    check("js83h", s83err(function () { JSON.parse("1 2") }) === "SyntaxError: Unexpected non-whitespace character after JSON at position 2 (line 1 column 3)")
    check("js83i", s83err(function () { JSON.parse("undefined") }) === "SyntaxError: \"undefined\" is not valid JSON")
    const cp = String.fromCodePoint(65, 128512)
    check("fc83a", cp.length === 3 && cp.charCodeAt(1) === 0xD83D && cp.charCodeAt(2) === 0xDE00 && String.fromCodePoint() === "")
    check("fc83b", s83err(function () { String.fromCodePoint(1.5) }) === "RangeError: Invalid code point 1.5" &&
        s83err(function () { String.fromCodePoint(0x110000) }) === "RangeError: Invalid code point 1114112")
}

// ===== SECTION 84: the temporal dead zone of let, const and class =====
// A lexical binding exists from its scope's entry, uninitialized until its
// declaration runs: a read, a write or a typeof before then is a ReferenceError, also
// through a closure or a function declaration of the scope, and in a switch's case
// block, which is entered after the discriminant. Values read off node 24.
function s84err(f) {
    try { return "" + f() } catch (e) { return e.name + ": " + e.message }
}
let s84top = s84read()
function s84read() { return s84err(function () { return s84late }) }
let s84late = 5
function s84() {
    check("tdz84a", s84err(function () { let q = "outer"; { let r = q; let q = 1; return r } }) === "ReferenceError: Cannot access 'q' before initialization")
    check("tdz84b", s84err(function () { { let r = typeof u; const u = 1; return r } }) === "ReferenceError: Cannot access 'u' before initialization")
    check("tdz84c", s84err(function () { { u = 5; let u = 1; return u } }) === "ReferenceError: Cannot access 'u' before initialization")
    check("tdz84d", s84err(function () { { x2++; let x2 = 1; return x2 } }) === "ReferenceError: Cannot access 'x2' before initialization")
    check("tdz84e", s84err(function () { { let f = function () { return v }; let r = f(); let v = 3; return r } }) === "ReferenceError: Cannot access 'v' before initialization")
    check("tdz84f", s84err(function () { { let r = new K(); class K {}; return r } }) === "ReferenceError: Cannot access 'K' before initialization")
    check("tdz84g", s84err(function () { let x = x; return x }) === "ReferenceError: Cannot access 'x' before initialization")
    check("tdz84h", s84err(function () { switch (1) { case 0: let z = 1; case 1: return z } }) === "ReferenceError: Cannot access 'z' before initialization")
    check("tdz84i", s84err(function () { { g(); let w = 1; function g() { return w } } }) === "ReferenceError: Cannot access 'w' before initialization")
    check("tdz84j", s84err(function () { let n = 1; switch (n) { case 1: let n = 2; return n } }) === "2")
    check("tdz84k", s84err(function () { switch (0) { case 0: let z = 3 } return typeof z }) === "undefined")
    check("tdz84l", s84err(function () { { let [a, b] = [a, 2]; return b } }) === "ReferenceError: Cannot access 'a' before initialization")
    check("tdz84m", s84err(function () { for (let i = i; ;) { return i } }) === "ReferenceError: Cannot access 'i' before initialization")
    check("tdz84n", s84err(function () { { q.a = 1; let q = {}; return q } }) === "ReferenceError: Cannot access 'q' before initialization")
    check("tdz84o", s84top === "ReferenceError: Cannot access 's84late' before initialization" && s84read() === "5")
    check("tdz84p", s84err(function () { let x = 1; { let f = () => x; let x = 2; return f() } }) === "2" &&
        s84err(function () { { let r = typeof undeclared84; return r } }) === "undefined")
}

// ===== SECTION 85: String.prototype.normalize, a BigInt literal as a callee's base =====
// The four Unicode normalization forms: decomposition (canonical, or compatibility for
// NFK*), canonical ordering of combining marks, and canonical composition for NFC and
// NFKC, Hangul by arithmetic. V8 prints a BigInt literal in a callee's text as
// "(intermediate value)". Values read off node 24.
function s85hex(s) {
    var o = []
    for (var i = 0; i < s.length; i++) { o.push(s.charCodeAt(i).toString(16)) }
    return o.join(" ")
}
function s85err(f) {
    try { return "" + f() } catch (e) { return e.name + ": " + e.message }
}
function s85() {
    const xs = ["é", "ẛ̣", "ﬁ", "각", "á̧", "①", "𝅗𝅥", "x\ud800y"]
    const forms = ["NFC", "NFD", "NFKC", "NFKD"]
    const got = []
    for (let i = 0; i < xs.length; i++) {
        for (let j = 0; j < forms.length; j++) { got.push(s85hex(xs[i].normalize(forms[j]))) }
    }
    check("nf85a", got.slice(0, 8).join("|") === "e9|65 301|e9|65 301|1e9b 323|17f 323 307|1e69|73 323 307")
    check("nf85b", got.slice(8, 16).join("|") === "fb01|fb01|66 69|66 69|ac01|1100 1161 11a8|ac01|1100 1161 11a8")
    check("nf85c", got.slice(16, 24).join("|") === "e1 327|61 327 301|e1 327|61 327 301|2460|2460|31|31")
    check("nf85d", got.slice(24).join("|") === "d834 dd57 d834 dd65|d834 dd57 d834 dd65|d834 dd57 d834 dd65|d834 dd57 d834 dd65|78 d800 79|78 d800 79|78 d800 79|78 d800 79")
    check("nf85e", "é".normalize() === "é" && "abc".normalize("NFD") === "abc" && typeof "".normalize === "function")
    check("nf85f", s85err(function () { return "a".normalize("nfc") }) === "RangeError: The normalization form should be one of NFC, NFD, NFKC, NFKD.")
    check("bc85a", s85err(function () { return (5n).nope() }) === "TypeError: (intermediate value).nope is not a function" &&
        s85err(function () { return 0x10n.nope() }) === "TypeError: (intermediate value).nope is not a function")
    check("bc85b", s85err(function () { return (5n + 1n).nope() }) === "TypeError: ((intermediate value) + (intermediate value)).nope is not a function" &&
        s85err(function () { return (-5n).nope() }) === "TypeError: (-(intermediate value)).nope is not a function")
}

// ===== SECTION 86: the arguments object is no Array =====
// It is indexed, has a length and iterates like an array, and Array.prototype's
// methods run on it through call, but Array.isArray and instanceof Array say no, it
// has no Array methods of its own, Object.prototype.toString tags it Arguments and
// JSON writes it as an object. An arrow has no arguments of its own. Values read off
// node 24.
function s86err(f) {
    try { return "" + f() } catch (e) { return e.name + ": " + e.message }
}
function s86a() { return Array.isArray(arguments) + " " + (arguments instanceof Array) + " " + arguments.length + " " + arguments[1] }
function s86b() { return Object.prototype.toString.call(arguments) + " " + String(arguments) + " " + JSON.stringify(arguments) }
function s86c() { return arguments.map(function (x) { return x }) }
function s86d() {
    var s = 0
    for (var x of arguments) { s += x }
    return s + " " + Array.prototype.slice.call(arguments).join("-") + " " + Array.isArray(Array.from(arguments)) + " " + [...arguments].length
}
function s86e() { var f = () => arguments.length + ":" + arguments[0]; return f(9) }
function s86() {
    check("ar86a", s86a(1, 2) === "false false 2 2")
    check("ar86b", s86b(1, "x") === "[object Arguments] [object Arguments] {\"0\":1,\"1\":\"x\"}")
    check("ar86c", s86err(function () { return s86c(1) }) === "TypeError: arguments.map is not a function")
    check("ar86d", s86d(1, 2, 3) === "6 1-2-3 true 3")
    check("ar86e", s86e(4, 5) === "2:4")
}

// ===== SECTION 87: a program's own keys that start with "__" =====
// The value model keeps its hidden slots under "__" names (__class, __keys, ...), so
// a program's own "__" key is stored escaped and listed, copied, serialized and read
// back as written, and it can neither see nor disturb the hidden slots. Values read
// off node 24.
function s87() {
    const p = JSON.parse('{"__proto__":1,"__x":2,"y":{"__z":[1]}}')
    check("dk87a", Object.keys(p).join() === "__proto__,__x,y" && JSON.stringify(p.y) === "{\"__z\":[1]}")
    const o = {__z: 3, w: 4, "__q": 5}
    o.__x = 1
    o["__y"] = 2
    let fk = ""
    for (const k in o) { fk += k }
    check("dk87b", Object.keys(o).join() === "__z,w,__q,__x,__y" && fk === "__zw__q__x__y" && o.__x + o["__y"] === 3)
    check("dk87c", JSON.stringify({__a: 1, b: {__c: 2}}) === "{\"__a\":1,\"b\":{\"__c\":2}}" &&
        ("__z" in o) && o.hasOwnProperty("__z") && !("__nope" in o))
    const h = {__class: 1, __name: "n", __keys: 7}
    check("dk87d", Object.keys(h).join() === "__class,__name,__keys" && h.__class === 1 && h.__keys === 7 && typeof h === "object")
    const {__z, ...rest} = {__z: 5, __w: 6}
    check("dk87e", __z === 5 && Object.keys(rest).join() === "__w" && Object.keys({...h}).join() === "__class,__name,__keys")
    check("dk87f", Object.entries({__e: 1}).join() === "__e,1" && Object.keys(Object.fromEntries([["__f", 1]])).join() === "__f")
    class A { constructor() { this.__p = 1 } __m() { return 1 } static __s() { return 3 } get __g() { return 4 } }
    class B extends A { __m() { return super.__m() + 1 } static t() { return super.__s() } f() { return super.__g } }
    check("dk87g", new B().__m() === 2 && B.t() === 3 && new B().f() === 4 && Object.keys(new B()).join() === "__p")
    let rk = ""
    JSON.parse('{"__r":1}', function (k, v) { rk += "[" + k + "]"; return v })
    check("dk87h", rk === "[__r][]" && JSON.stringify({__x: 1}, function (k, v) { return k === "__x" ? 5 : v }) === "{\"__x\":5}")
}

// ===== SECTION 88: the ES2015-ES2024 statics of Object, Number and String, structuredClone =====
// Object.hasOwn, Object.groupBy and Map.groupBy, Object.fromEntries over any iterable,
// Number's predicates, parsers and constants, String.raw, structuredClone. Values read
// off node 24.
function s88() {
    const v = [{a: 1}, "a", null, 5, "ab", [1, 2]]
    function err(f) { try { f(); return "none" } catch (e) { return (e instanceof TypeError ? "T:" : e.name + ":") + e.message } }
    const s = Symbol("q")
    const so = {}
    so[s] = 1
    check("es88a", Object.hasOwn(v[0], v[1]) && !Object.hasOwn(v[0], "b") && !Object.hasOwn(v[0], "toString") &&
        Object.hasOwn(v[5], "length") && Object.hasOwn(v[5], 1) && !Object.hasOwn(v[5], 2) && Object.hasOwn(v[4], 0) &&
        Object.hasOwn(so, s) && !Object.hasOwn(v[3], "a") && typeof Object.hasOwn === "function" && Object.hasOwn.length === 2 &&
        err(function () { return Object.hasOwn(v[2], "a") }) === "T:Cannot convert undefined or null to object")
    function* gen() { yield 1; yield 2; yield 3 }
    const it = {i: 0, next() { this.i++; return {value: this.i, done: this.i > 2} }, [Symbol.iterator]() { return this }}
    check("es88b", JSON.stringify(Object.groupBy([1, 2, 3, 4], function (x) { return x % 2 ? "odd" : "even" })) === "{\"odd\":[1,3],\"even\":[2,4]}" &&
        JSON.stringify(Object.groupBy(v[4], function (c, i) { return c + i })) === "{\"a0\":[\"a\"],\"b1\":[\"b\"]}" &&
        JSON.stringify(Object.groupBy(gen(), function (x) { return x > 1 })) === "{\"false\":[1],\"true\":[2,3]}" &&
        JSON.stringify(Object.groupBy(new Set([1, 1, 2]), function (x) { return "k" })) === "{\"k\":[1,2]}" &&
        JSON.stringify(Object.groupBy(it, function (x) { return "n" })) === "{\"n\":[1,2]}" &&
        err(function () { return Object.groupBy(v[2], function () {}) }) === "T:Object.groupBy called on null or undefined" &&
        err(function () { return Object.groupBy([1], v[3]) }) === "T:5 is not a function" &&
        err(function () { return Object.groupBy(v[3], function () {}) }) === "T:number 5 is not iterable (cannot read property Symbol(Symbol.iterator))")
    const k = {}
    const m = Map.groupBy([1, 2, 3, 4], function (x) { return x % 2 ? k : "even" })
    check("es88c", m.size === 2 && m.get(k).join() === "1,3" && m.get("even").join() === "2,4" && Map.groupBy.length === 2 &&
        err(function () { return Map.groupBy(v[2], function () {}) }) === "T:Map.groupBy called on null or undefined")
    function* pairs() { yield ["x", 1]; yield ["__y", 2] }
    check("es88d", JSON.stringify(Object.fromEntries(new Map([["k", 3]]))) === "{\"k\":3}" &&
        JSON.stringify(Object.fromEntries(pairs())) === "{\"x\":1,\"__y\":2}" &&
        JSON.stringify(Object.fromEntries([{0: "a", 1: 2}])) === "{\"a\":2}" &&
        err(function () { return Object.fromEntries([1]) }) === "T:Iterator value 1 is not an entry object" &&
        err(function () { return Object.fromEntries(v[2]) }) === "T:undefined is not iterable")
    const xs = [0, -0, 1.5, "3", NaN, Infinity, -7, 9007199254740992, null]
    let ps = ""
    for (let i = 0; i < xs.length; i++) {
        ps += (Number.isInteger(xs[i]) ? "i" : "-") + (Number.isSafeInteger(xs[i]) ? "s" : "-") +
            (Number.isFinite(xs[i]) ? "f" : "-") + (Number.isNaN(xs[i]) ? "n" : "-") + " "
    }
    check("es88e", ps === "isf- isf- --f- ---- ---n ---- isf- i-f- ---- " &&
        Number.parseFloat("3.5e1x") === 35 && Number.parseInt("ff", 16) === 255 && Number.parseInt.length === 2 &&
        typeof Number.isInteger === "function" && Number.isInteger.name === "isInteger")
    check("es88f", Number.MAX_SAFE_INTEGER === 9007199254740991 && Number.MIN_SAFE_INTEGER === -9007199254740991 &&
        Number.EPSILON === Math.pow(2, -52) && 1 + Number.EPSILON !== 1 && "" + Number.MAX_VALUE === "1.7976931348623157e+308" &&
        "" + Number.MIN_VALUE === "5e-324" && Number.POSITIVE_INFINITY === Infinity && Number.NEGATIVE_INFINITY === -Infinity &&
        Number.NaN !== Number.NaN)
    const n = v[3]
    check("es88g", String.raw`x\ty${n}\n${n + 1}` === "x\\ty5\\n6" && String.raw({raw: "abc"}, 1, 2, 3) === "a1b2c" &&
        String.raw({raw: ["p", "q"]}, v[4], 9) === "pabq" && String.raw({raw: []}) === "" && String.raw.length === 1 &&
        err(function () { return String.raw({}) }) === "T:Cannot convert undefined or null to object")
    const o = {a: [1, {b: n}], c: "s", __d: 2}
    o.self = o
    const c = structuredClone(o)
    c.a[1].b = 9
    const sm = structuredClone(new Map([[1, o.a]]))
    check("es88h", o.a[1].b === 5 && c.a[1].b === 9 && c.self === c && c !== o && Object.keys(c).join() === "a,c,__d,self" &&
        sm instanceof Map && sm.get(1)[0] === 1 && sm.get(1) !== o.a && structuredClone(v[1]) === "a" &&
        err(function () { return structuredClone(s) }) === "DataCloneError:Symbol(q) could not be cloned.")
}

// ===== SECTION 89: the arguments object's length, and the mapped arguments of a sloppy function =====
// `length` is the argument count whatever is written to the elements; in a sloppy
// function with plain parameters, parameter i and arguments[i] are one value below
// that count, in both directions, through closures, compound stores, destructuring and
// a var of the parameter's name. Strict code, defaults, rest parameters and class
// methods are unmapped. Values read off node 24.
function s89() {
    const v = [1, 2, 3]
    function l1() { arguments[0] = 9; return arguments.length + ":" + arguments[0] }
    function l2(p) { arguments[3] = 9; return arguments.length + ":" + Array.from(arguments).length }
    function l3() { arguments.length = 2; return arguments.length }
    check("ar89a", l1() === "0:9" && l2(v[0]) === "1:1" && l3(v[0]) === 2 && l1(v[0], v[1]) === "2:9")
    function m1(p) { p = 5; return arguments[0] }
    function m2(p) { arguments[0] = 9; return p }
    function m3(p, q) { q = 7; return arguments.length + ":" + arguments[1] }
    function m4(p) { p++; p += 10; return arguments[0] }
    function m5(p) { var p = 8; return arguments[0] + ":" + p }
    function m6(p) { (function () { p = 4 })(); return arguments[0] }
    function m7(p) { const g = () => { arguments[0] = 6 }; g(); return p }
    function m8(p) { arguments[0] += 1; arguments[0]++; return p }
    function m9(p) { [p] = [3]; return arguments[0] }
    function m10(p) { ({p} = {p: 6}); return arguments[0] }
    check("ar89b", m1(v[0]) === 5 && m2(v[0]) === 9 && m3(v[0]) === "1:undefined" && m3(v[0], v[1]) === "2:7" &&
        m4(v[0]) === 12 && m5(v[0]) === "8:8" && m6(v[0]) === 4 && m7(v[0]) === 6 && m8(v[0]) === 3 &&
        m9(v[0]) === 3 && m10(v[0]) === 6 && m1() === undefined && m2() === undefined)
    function u1(p) { "use strict"; p = 5; return arguments[0] }
    function u2(p = 0) { p = 5; return arguments[0] }
    function u3(p, ...r) { p = 5; return arguments[0] }
    function u4(p) { { let p = 3; p = 4 } return arguments[0] }
    function u5(p) { try { throw 1 } catch (p) { p = 4 } return arguments[0] }
    function* u6(p) { p = 5; yield arguments[0] }
    class C { m(p) { p = 5; return arguments[0] + ":" + arguments.length } }
    check("ar89c", u1(v[0]) === 1 && u2(v[0]) === 1 && u3(v[0]) === 1 && u4(v[0]) === 1 && u5(v[0]) === 1 &&
        new C().m(v[0], v[1]) === "1:2")
    function n1(p) { function inner(p) { p = 9; return arguments[0] } return inner(p) + ":" + arguments[0] + ":" + p }
    function n2(n) { if (n > 0) { n = n - 1; return n2(n) + "," + arguments[0] } return "0" }
    function n3(p) { arguments["0"] = 8; return p }
    const o = { m(p) { p = 6; return arguments[0] } }
    check("ar89d", n1(v[0]) === "9:1:1" && n2(v[1]) === "0,0,1" && n3(v[0]) === 8 && o.m(v[0]) === 6)
}

// ===== SECTION 90: calls inside an assignment target, the restricted '++', a body's functions =====
// A method call in the middle of a target is the ordinary method call, receiver and
// all ('k.get(1).a = 2'), under every assignment operator, in a destructuring and a
// for-of head. Two signs in a row are the '++'/'--' token, so 'x' + newline + '++y' is
// two statements. A function declared in a body that also has a var closes over the
// body's let and const. Values read off node 24.
function s90() {
    const v = [1, 2, 3]
    class K { constructor() { this.o = {a: v[0]} } get(i) { return this.o } m() { return v[0] } }
    const k = new K()
    const mp = new Map([["k", {a: v[0]}]])
    const log = []
    function key(x) { log.push("k" + x); return x }
    function obj() { log.push("o"); return k }
    k.get(v[0]).a = 2
    mp.get("k").a = 3
    mp.get("k").a += 4
    obj().get(key(1))[key("b")] = v[2]
    check("tg90a", k.o.a === 2 && mp.get("k").a === 7 && k.o.b === 3 && log.join() === "o,k1,kb")
    k.get(v[0]).a ||= 9
    k.get(v[0]).c ??= 4
    k.get(v[0]).a++
    ;++k.get(v[0]).a
    ;[k.get(1).d, k.get(2).e] = [v[1], v[2]]
    ;({x: k.get(1).f} = {x: v[0]})
    for (k.get(1).g of v) {}
    k.m.x = 5
    check("tg90b", k.o.a === 4 && k.o.c === 4 && k.o.d === 2 && k.o.e === 3 && k.o.f === 1 && k.o.g === 3 &&
        k.m.x === 5)
    let t = 5, y = 10, z = 1
    const r = typeof t--
    var a = t
    ++z
    var b = y
    --z
    var c = t + +z, d = - -z
    check("tg90c", r === "number" && t === 4 && a === 4 && b === 10 && z === 1 && c === 5 && d === 1)
    const lex = v[2]
    function seesLex() { return lex }
    var hoisted = v[1]
    check("tg90d", seesLex() === 3 && hoisted === 2)
}

// ===== SECTION 91: the arguments object through other references, past its length; "__#" keys =====
// A store through another reference to a mapped arguments object reaches the parameter,
// a generator's and an async function's arguments are mapped too, and `delete` ends an
// element's mapping. `length` is an ordinary property: writing it leaves the elements
// alone, and an iteration, a spread, apply and an Array method read up to it, while the
// own keys are the elements that hold a value. Number.parseInt and parseFloat are the
// globals. A program's own "__#" key is a key like any other. Values read off node 24.
function s91() {
    const v = [1, 2, 3]
    function a1(p) { var a = arguments; a[0] = 2; return p }
    function a2(p) { const box = {a: arguments}; box.a[0] = 4; return p }
    function a3(p) { const a = arguments; const g = () => { a[0] = 6 }; g(); return p }
    function* g1(p) { p = 5; yield arguments[0] }
    function* g2(p) { arguments[0] = 6; yield p; yield p }
    function d1(p) { delete arguments[0]; p = 3; return arguments[0] + ":" + p }
    function d2(p) { delete arguments[0]; arguments[0] = 9; return p }
    const it2 = g2(v[0])
    check("ag91a", a1(v[0]) === 2 && a2(v[0]) === 4 && a3(v[0]) === 6 && g1(v[0]).next().value === 5 &&
        it2.next().value === 6 && it2.next().value === 6 && d1(v[0]) === "undefined:3" && d2(v[0]) === 1)
    function sum() { let s = 0; for (let i = 0; i < arguments.length; i++) { s += arguments[i] } return s }
    function k1() {
        arguments[2] = 1
        return [Object.keys(arguments).join(), arguments.length, [...arguments].length, Array.from(arguments).length,
            Array.prototype.slice.call(arguments).length, JSON.stringify(arguments), sum.apply(null, arguments),
            arguments.hasOwnProperty(1), 1 in arguments, 2 in arguments].join(" ")
    }
    function k2() { arguments[2] = 1; let s = ""; for (const x of arguments) { s += x } for (const k in arguments) { s += "k" + k } return s }
    check("ag91b", k1() === "2 0 0 0 0 {\"2\":1} 0 false false true" &&
        k1(v[0]) === "0,2 1 1 1 1 {\"0\":1,\"2\":1} 1 false false true" && k2(v[0]) === "1k0k2")
    function t1(p, q) { arguments.length = 0; return arguments.length + ":" + arguments[0] + ":" + [...arguments].length + ":" + Object.keys(arguments).join("") }
    function t2(p) { arguments.length = 3; return [...arguments].join("|") + ":" + Object.keys(arguments).length + ":" + Array.from(arguments).length }
    function t3(p, q) { arguments.length = 1; q = 9; return arguments[1] + ":" + [...arguments].length }
    function t4() { return Object.getOwnPropertyNames(arguments).join() }
    check("ag91c", t1(v[0], 7) === "0:1:0:01" && t2(v[0]) === "1||:1:3" && t3(v[0], v[1]) === "9:1" &&
        t4(v[0], v[1]) === "0,1,length,callee" && Object.getOwnPropertyNames(v).join() === "0,1,2,length")
    check("ag91d", Number.parseInt === parseInt && Number.parseFloat === parseFloat && parseInt.length === 2 &&
        parseFloat.length === 1 && parseInt.name === "parseInt" && Number.parseFloat.name === "parseFloat")
    const ks = ["__#x", "__#1#y"]
    const o = {}
    o[ks[0]] = 1
    o[ks[1]] = 2
    const lit = {"__#a": 3, __b: 4}
    const ps = JSON.parse('{"__#q": 5}')
    class P { #x = 1; has(t) { return #x in t } }
    const pp = new P()
    check("ag91e", Object.keys(o).join() === "__#x,__#1#y" && JSON.stringify(o) === '{"__#x":1,"__#1#y":2}' &&
        Object.keys(lit).join() === "__#a,__b" && ps["__#q"] === 5 && Object.keys(ps).length === 1 &&
        o.hasOwnProperty(ks[0]) && (ks[1] in o) && pp.has(pp) && !pp.has(o) && Object.keys(pp).length === 0 &&
        JSON.stringify(Object.entries({"__#z": 1})) === '[["__#z",1]]')
}

// ===== SECTION 92: iterating a value that is not iterable =====
// GetIterator throws a TypeError for a value with no callable [Symbol.iterator]: a
// spread, a for-of, an array pattern and a yield* over a plain object, a number, a
// boolean, null, or an object whose only protocol is a `next` method. Values read off
// node 24.
function s92() {
    const v = [{}, {a: 1}, 5, {i: 0, next() { return {value: 1, done: true} }}, true, null]
    function te(f) { try { f(); return "-" } catch (e) { return e instanceof TypeError ? "T" : "E" } }
    function* yd(x) { yield* x }
    const r = []
    for (let i = 0; i < v.length; i++) {
        const x = v[i]
        r.push(te(() => [...x]) + te(() => { for (const e of x) {} }) + te(() => { const [p] = x }) +
            te(() => [...yd(x)]) + te(() => Math.max(...x)))
    }
    check("it92a", r.join() === "TTTTT,TTTTT,TTTTT,TTTTT,TTTTT,TTTTT")
    class K92 {}
    K92.prototype[Symbol.iterator] = function () { return [7, 8][Symbol.iterator]() }
    function F92() {}
    F92.prototype[Symbol.iterator] = function* () { yield 9 }
    check("it92b", [...[1, 2], ..."ab"].join() === "1,2,a,b" && te(() => Object.fromEntries(v[3])) === "T" &&
        te(() => new Set(v[0])) === "T" && [...new K92(), ...new F92()].join() === "7,8,9")
}

// ===== SECTION 93: a generator body runs once =====
// A generator or async body is suspended at a yield or an await and continued there, so
// the side effects before it happen once: through next(v), throw(), return(), a yield*,
// a for-of break, an async function's awaits and an async generator. Values read off
// node 24.
const log93 = []
function* g93() { log93.push("a"); const x = yield 1; log93.push("b" + x); try { yield 2 } catch (e) { log93.push("c" + e) } finally { log93.push("f") } log93.push("d"); return 9 }
function* fin93() { try { log93.push("s"); yield 1; yield 2 } finally { log93.push("fin") } }
function* in93() { log93.push("i0"); const r = yield 10; log93.push("ir" + r); return "IR" }
function* out93() { log93.push("o0"); const v = yield* in93(); log93.push("ov" + v); yield 20 }
async function af93(n) { log93.push("af" + n); const x = await n; log93.push("aw" + x); return x + 1 }
async function* ag93() { log93.push("ag0"); yield 1; log93.push("ag1"); await 0; yield 2; log93.push("ag2") }
function s93() {
    const g = g93()
    const r = [g.next(), g.next("X"), g.throw("E"), g.next()].map(function (x) { return x.value + "/" + x.done }).join()
    check("gr93a", r === "1/false,2/false,9/true,undefined/true" && log93.join() === "a,bX,cE,f,d")
    log93.length = 0
    const h = fin93()
    h.next()
    const hr = h.return(5)
    const o = out93()
    const or = [o.next(), o.next("S"), o.next()].map(function (x) { return x.value }).join()
    check("gr93b", hr.value === 5 && hr.done && or === "10,20," && log93.join() === "s,fin,o0,i0,irS,ovIR")
    log93.length = 0
    for (const x of fin93()) { break }
    check("gr93c", log93.join() === "s,fin")
    log93.length = 0
    af93(3).then(function (v) {
        const ok = v === 4 && log93.join() === "af3,ag0,aw3"
        check("gr93d", ok)
        if (!ok) { exit(1) }
    })
    const ys = []
    const run = async function () { for await (const y of ag93()) { ys.push(y) } }
    run().then(function () {
        const ok = ys.join() === "1,2" && log93.join() === "af3,ag0,aw3,ag1,ag2"
        check("gr93e", ok)
        if (!ok) { exit(1) }
    })
}

// ===== SECTION 94: a string key shaped like a symbol's; Object.getOwnPropertySymbols =====
// A program's string key "__#@iterator" is an ordinary string key, not Symbol.iterator,
// however a symbol's key is spelled inside the engine; getOwnPropertySymbols lists the
// own symbol keys in the order they were first written; iterating null or undefined is
// GetIterator's TypeError. Values read off node 24.
class K94 { ["__#@iterator"]() { return [1][Symbol.iterator]() } }
function s94() {
    const ks = ["__#@iterator", "__#@x"]
    const o = {}
    o[ks[0]] = 5
    const p = JSON.parse('{"__#@iterator":3,"a":4}')
    const q = {"__#@x": 1}
    q[Symbol.iterator] = 2
    q[ks[0]] = 3
    function te(f) { try { f(); return "-" } catch (e) { return e instanceof TypeError ? "T" : "E" } }
    check("sk94a", Object.keys(o).join() === "__#@iterator" && typeof o[Symbol.iterator] === "undefined" &&
        JSON.stringify(o) === '{"__#@iterator":5}' && ks[1] in q && Object.keys(p).join() === "__#@iterator,a")
    check("sk94b", te(() => [...p]) === "T" && te(() => [...new K94()]) === "T" && q[Symbol.iterator] === 2 &&
        q[ks[0]] === 3 && Object.keys(q).join() === "__#@x,__#@iterator" && [...new Set([1, 1, 2])].join() === "1,2")
    const s1 = Symbol("d")
    const so = {}
    so[Symbol.for("f94")] = 1
    so.a = 0
    so[s1] = 2
    so[Symbol.asyncIterator] = 3
    delete so[Symbol.for("f94")]
    const syms = Object.getOwnPropertySymbols(so)
    const gs = Object.getOwnPropertySymbols
    check("sy94c", syms.length === 2 && syms[0] === s1 && syms[1] === Symbol.asyncIterator &&
        Object.getOwnPropertySymbols({x: 1}).length === 0 && gs(so).length === 2 && gs.length === 1 &&
        te(() => Object.getOwnPropertySymbols(null)) === "T" && so.hasOwnProperty(s1) && Object.keys(structuredClone(q)).join() === "__#@x,__#@iterator")
    const nv = [null, undefined]
    const msg = []
    for (let i = 0; i < nv.length; i++) {
        try { const r = [...nv[i]] } catch (e) { msg.push(e instanceof TypeError && e.message.indexOf("is not iterable") >= 0 && e.message.indexOf("__") < 0) }
    }
    check("it94d", msg.join() === "true,true")
}

// ===== SECTION 95: the own properties of a class object and of a prototype =====
// A class object's own names are `length`, `name` and `prototype`, then its static
// members in definition order (methods and accessors, then fields and what static
// blocks write); a static method written again stays non-enumerable. A prototype owns
// `constructor` and the class body's methods and accessors, and those stay
// non-enumerable when written through it. Values read off node 24.
class K95 { static a = 1; static s() {} static get g() { return 1 } m() {} get pg() { return 1 } static { K95.c = 3 } }
class D95 extends K95 { static d() {} }
class B95 { m() {} }
function F95() {}
function s95() {
    const gp = Object.getOwnPropertyNames
    check("cl95a", gp(K95).join() === "length,name,prototype,s,g,a,c" && gp(D95).join() === "length,name,prototype,d" &&
        gp(B95).join() === "length,name,prototype" && Object.keys(K95).join() === "a,c")
    check("cl95b", K95.hasOwnProperty("s") && K95.hasOwnProperty("g") && K95.hasOwnProperty("a") &&
        K95.hasOwnProperty("prototype") && K95.hasOwnProperty("name") && !K95.hasOwnProperty("m") &&
        !D95.hasOwnProperty("s") && Object.hasOwn(D95, "d"))
    K95.s = 9
    K95.z = 1
    check("cl95c", Object.keys(K95).join() === "a,c,z" && gp(K95).join() === "length,name,prototype,s,g,a,c,z" &&
        !K95.propertyIsEnumerable("s") && K95.propertyIsEnumerable("z"))
    check("pr95d", B95.prototype.hasOwnProperty("constructor") && B95.prototype.hasOwnProperty("m") &&
        K95.prototype.hasOwnProperty("pg") && !D95.prototype.hasOwnProperty("m") &&
        F95.prototype.hasOwnProperty("constructor") && !K95.prototype.hasOwnProperty("s"))
    B95.prototype.m = function () { return 2 }
    B95.prototype.x = 1
    const ks = []
    for (const k in B95.prototype) { ks.push(k) }
    check("pr95e", Object.keys(B95.prototype).join() === "x" && ks.join() === "x" && !B95.prototype.propertyIsEnumerable("m") &&
        JSON.stringify(B95.prototype) === '{"x":1}' && Object.entries(B95.prototype).length === 1 && new B95().m() === 2 &&
        gp(B95.prototype).join() === "constructor,m,x")
}

// ===== SECTION 96: for-in and `in` through the prototype chain =====
// for-in visits the own enumerable keys, then each prototype's up the chain, a name
// met lower down shadowing one above - a declared method included, which is not
// enumerable (EnumerateObjectProperties, ECMA-262 14.7.5.9). `in` finds a declared
// method or accessor and a key written onto a prototype. A class object inherits
// Object.prototype's methods as values, and a private accessor may be static. Values
// read off node 24.
class A96 {}
A96.prototype.k1 = 1
A96.prototype.m = 2
class B96 extends A96 { m() {} get g() { return 1 } }
B96.prototype.k2 = 3
function F96() { this.a = 1 }
F96.prototype.v = 5
class P96 {
    static #w = 2
    static get #h() { return P96.#w * 10 }
    static set #h(x) { P96.#w = x }
    static h(x) { P96.#h = x; return P96.#h }
    #n = 1
    get #i() { return this.#n }
    set #i(x) { this.#n = x }
    i(x) { this.#i = x; return this.#i }
}
function s96() {
    const b = new B96()
    b.own = 4
    const ks = []
    for (const k in b) { ks.push(k) }
    const ps = []
    for (const k in B96.prototype) { ps.push(k) }
    const fs = []
    for (const k in new F96()) { fs.push(k) }
    check("fi96a", ks.join() === "own,k2,k1" && ps.join() === "k2,k1" && fs.join() === "a,v" &&
        Object.keys(b).join() === "own" && JSON.stringify(b) === '{"own":4}')
    check("in96b", "k1" in b && "m" in b && "g" in b && "m" in B96.prototype && "g" in B96.prototype &&
        "k1" in B96.prototype && !("zz" in b) && "v" in new F96() && "prototype" in B96 && !("prototype" in b))
    check("cv96c", typeof B96.hasOwnProperty === "function" && typeof B96.valueOf === "function" &&
        typeof B96.toString === "function" && typeof B96.propertyIsEnumerable === "function" &&
        B96.hasOwnProperty.call(B96, "prototype") && typeof B96.prototype.hasOwnProperty === "function")
    check("pa96d", P96.h(7) === 70 && new P96().i(3) === 3)
}

// ===== SECTION 97: inherited data properties, Object.create, Object.prototype's methods as values =====
// A property written onto a prototype is read through every object below it, by a
// plain read, a compound assignment and a destructuring pattern alike, until an own
// property shadows it. Object.create(p) makes an object whose prototype is p.
// Object.prototype's methods read as values are the very functions Object.prototype
// holds. Values read off node 24.
class A97 { m() { return "m" } }
class B97 extends A97 {}
class K97 {}
function F97() { this.own = 1 }
function s97() {
    A97.prototype.x = 1
    A97.prototype.nul = null
    B97.prototype.y = 2
    const a = new A97(), b = new B97()
    check("ip97a", a.x === 1 && b.x === 1 && b.y === 2 && a.y === undefined && a.nul === null &&
        B97.prototype.x === 1 && !b.hasOwnProperty("x") && "x" in b)
    a.x = 5
    A97.prototype.x = 7
    const b2 = new B97()
    b2.x += 10
    const b3 = new B97()
    b3.x++
    const { x: dx, y: dy } = new B97()
    check("ip97b", a.x === 5 && b.x === 7 && b2.x === 17 && new B97().x === 7 && b3.x === 8 && dx === 7 && dy === 2)
    F97.prototype.v = 2
    F97.prototype.greet = function () { return "hi " + this.own }
    const f = new F97()
    check("ip97c", f.v === 2 && f.greet() === "hi 1" && f.constructor === F97)
    const base = { p: 1, hello() { return "hello " + this.n } }
    const o = Object.create(base)
    o.n = "o"
    const o2 = Object.create(o)
    check("oc97d", o.p === 1 && o.hello() === "hello o" && Object.keys(o).join() === "n" && !o.hasOwnProperty("p") &&
        "p" in o && o2.p === 1 && o2.hello() === "hello o")
    base.p = 3
    const p3 = o.p
    o.p = 9
    check("oc97e", p3 === 3 && o.p === 9 && base.p === 3 && o2.p === 9)
    const ia = Object.create(A97.prototype)
    const ib = Object.create(b)
    check("oc97f", ia.x === 7 && ia.m() === "m" && ia instanceof A97 && ia.constructor === A97 &&
        ib.y === 2 && ib instanceof B97 && ib.constructor === B97)
    const nul = Object.create(null)
    nul.a = 3
    const ts = Object.create({ toString() { return "TS" } })
    check("oc97g", nul.a === 3 && typeof Object.create === "function" && Object.create.length === 2 &&
        String(ts) === "TS" && `${ts}` === "TS")
    let msg = ""
    try { Object.create(5) } catch (e) { msg = e instanceof TypeError ? e.message : "" }
    check("oc97h", msg === "Object prototype may only be an Object or null: 5")
    check("pm97i", typeof K97.isPrototypeOf === "function" && typeof K97.toLocaleString === "function" &&
        typeof ({}).toLocaleString === "function" && typeof a.isPrototypeOf === "function" &&
        typeof (function () {}).isPrototypeOf === "function")
    check("pm97j", K97.valueOf === K97.valueOf && ({}).valueOf === a.valueOf && a.toString === Object.prototype.toString &&
        a.isPrototypeOf === Object.prototype.isPrototypeOf)
    check("pm97k", A97.prototype.isPrototypeOf(b) && !B97.prototype.isPrototypeOf(a) &&
        A97.prototype.isPrototypeOf(B97.prototype) && Object.prototype.isPrototypeOf(a) &&
        Object.prototype.isPrototypeOf([]) && !Object.prototype.isPrototypeOf(5) &&
        base.isPrototypeOf(o2) && !o.isPrototypeOf(base) && F97.prototype.isPrototypeOf(f))
    const tl = ({}).toLocaleString
    check("pm97l", ({}).toLocaleString() === "[object Object]" && ({ toString() { return "Q" } }).toLocaleString() === "Q" &&
        tl.call([1, 2]) === "1,2" && "ab".toLocaleString() === "ab")
}

// ===== SECTION 98: replacing a constructor's prototype =====
// `F.prototype = p` makes p the prototype of every instance `new F()` builds from
// then on (an instance built before keeps the old one), and `instanceof F` asks
// whether p is on the chain; a prototype that is no object gives Object.prototype.
// The classic `H.prototype = Object.create(G.prototype)` inheritance rides on it.
// Values read off node 24.
function s98() {
    function F() {}
    F.prototype = { q: 4, m() { return this.q + 1 } }
    const f = new F()
    check("rp98a", f.q === 4 && f.m() === 5 && f instanceof F && !f.hasOwnProperty("q"))
    function G(n) { this.n = n }
    G.prototype.hi = function () { return "G" + this.n }
    function H(n) { G.call(this, n) }
    H.prototype = Object.create(G.prototype)
    H.prototype.constructor = H
    H.prototype.ho = function () { return "H" + this.n }
    const h = new H(3)
    check("rp98b", h.hi() === "G3" && h.ho() === "H3" && h instanceof H && h instanceof G && h.constructor === H)
    const old = new F()
    F.prototype = { q: 7 }
    const nw = new F()
    check("rp98c", old.q === 4 && nw.q === 7 && !(old instanceof F) && nw instanceof F)
    F.prototype = G.prototype
    const g2 = new F()
    check("rp98d", g2 instanceof G && g2 instanceof F && typeof g2.hi === "function")
    F.prototype = 5
    const z = new F()
    let msg = ""
    try { z instanceof F } catch (e) { msg = e.message }
    check("rp98e", typeof z === "object" && z.q === undefined && !(1 instanceof F) &&
        msg === "Function has non-object prototype '5' in instanceof check")
    F.prototype = { a: 1 }
    const w = new F()
    w.b = 2
    const ks = []
    for (const k in w) ks.push(k)
    check("rp98f", ks.join() === "b,a" && Object.keys(w).join() === "b" && "a" in w)
}

// ===== SECTION 99: getPrototypeOf, setPrototypeOf, defineProperty, a null prototype, toLocaleString =====
// Object.getPrototypeOf answers the object an instance, a prototype or a primitive
// inherits from; setPrototypeOf replaces it; defineProperty makes a property that is
// not enumerable unless asked, or an accessor; Object.create(null) inherits nothing,
// not even Object.prototype's methods. Number#toLocaleString is node's en-US format:
// grouped, at most three fraction digits rounded half up. Values read off node 24.
class A99 { m() { return "m" } }
class B99 extends A99 {}
class R99 extends Object {}
function s99() {
    const gp = Object.getPrototypeOf
    const base = { p: 1 }
    const o = Object.create(base)
    const b = new B99()
    check("gp99a", gp(o) === base && gp(base) === Object.prototype && gp(Object.prototype) === null &&
        gp(b) === B99.prototype && gp(B99.prototype) === A99.prototype && gp(A99.prototype) === Object.prototype &&
        gp(B99) === A99 && gp(R99) === Object && gp(R99.prototype) === Object.prototype)
    check("gp99b", gp([]) === Array.prototype && gp("s") === String.prototype && gp(5) === Number.prototype &&
        gp(Array.prototype) === Object.prototype && typeof gp === "function" && gp.length === 1)
    let msg = ""
    try { gp(null) } catch (e) { msg = e.message }
    check("gp99c", msg === "Cannot convert undefined or null to object")
    const n = Object.create(null)
    n.a = 3
    let nmsg = ""
    try { n.toString() } catch (e) { nmsg = e instanceof TypeError ? e.message : "" }
    const n2 = Object.create(n)
    check("np99d", typeof n.toString === "undefined" && !("toString" in n) && gp(n) === null && n.a === 3 &&
        Object.keys(n).join() === "a" && JSON.stringify(n) === '{"a":3}' && nmsg === "n.toString is not a function" &&
        n2.a === 3 && typeof n2.hasOwnProperty === "undefined" && gp(n2) === n)
    const s = {}
    const r1 = Object.setPrototypeOf(s, { z: 9 })
    const z1 = s.z, zin = "z" in s, zk = Object.keys(s).length
    Object.setPrototypeOf(s, A99.prototype)
    const sm = s.m(), sa = s instanceof A99
    Object.setPrototypeOf(s, null)
    const st = typeof s.toString, sg = gp(s)
    Object.setPrototypeOf(s, Object.prototype)
    check("sp99e", r1 === s && z1 === 9 && zin && zk === 0 && sm === "m" && sa && st === "undefined" && sg === null &&
        typeof s.toString === "function" && s.z === undefined && Object.setPrototypeOf(5, null) === 5 &&
        Object.setPrototypeOf.length === 2)
    let spmsg = ""
    try { Object.setPrototypeOf({}, 5) } catch (e) { spmsg = e.message }
    check("sp99f", spmsg === "Object prototype may only be an Object or null: 5")
    const d = {}
    Object.defineProperty(d, "x", { value: 5 })
    const dx = d.x, dk = Object.keys(d).length, dj = JSON.stringify(d)
    Object.defineProperty(d, "y", { value: 6, enumerable: true })
    let hidden = 1
    Object.defineProperty(d, "g", { get() { return hidden * 10 }, set(v) { hidden = v }, enumerable: true })
    d.g = 4
    check("dp99g", dx === 5 && dk === 0 && dj === "{}" && d.hasOwnProperty("x") && "x" in d && !d.propertyIsEnumerable("x") &&
        Object.keys(d).join() === "y,g" && JSON.stringify(d) === '{"y":6,"g":40}' && hidden === 4 && d.g === 40)
    const q = Object.defineProperty({ q: 1 }, "q", { value: 2 })
    const c = Object.create({ base: 1 }, { a: { value: 1, enumerable: true }, h: { value: 2 } })
    const dp = Object.defineProperties({}, { u: { value: 7, enumerable: true }, v: { get() { return 8 } } })
    check("dp99h", q.q === 2 && Object.keys(q).join() === "q" && c.a === 1 && c.h === 2 && c.base === 1 &&
        Object.keys(c).join() === "a" && dp.u === 7 && dp.v === 8 && Object.keys(dp).join() === "u" &&
        Object.defineProperty.length === 3 && Object.defineProperties.length === 2)
    let dmsg = "", dmsg2 = ""
    try { Object.defineProperty(1, "x", {}) } catch (e) { dmsg = e.message }
    try { Object.defineProperty({}, "x", 1) } catch (e) { dmsg2 = e.message }
    check("dp99i", dmsg === "Object.defineProperty called on non-object" && dmsg2 === "Property description must be an object: 1")
    const ls = [1234.5, 0, -0, 1e21, 123456789.123456, 1.0005, -1234.5678, NaN, -Infinity, 0.0001, -0.0001, 999.9995, 0.9995]
    const lo = []
    for (const x of ls) lo.push(x.toLocaleString())
    check("tl99j", lo.join("|") === "1,234.5|0|-0|1,000,000,000,000,000,000,000|123,456,789.123|1.001|-1,234.568|NaN|-\u221e|0|-0|1,000|1")
    check("tl99k", [1234.5, "a", null, undefined, [1000, 2], -0, true].toLocaleString() === "1,234.5,a,,,1,000,2,-0,true" &&
        (123456789n).toLocaleString() === "123,456,789" && (-5n).toLocaleString() === "-5" &&
        [{ toLocaleString() { return "L" } }, { toString() { return "S" } }].toLocaleString() === "L,S" &&
        typeof (5).toLocaleString === "function")
}

// ===== SECTION 100: primitive wrapper objects, property attributes, Function and Boolean prototypes =====
// new Number(7), Object("s") and Object.prototype.valueOf.call(true) are objects
// holding a primitive: typeof says "object", ToPrimitive and ToString answer the
// primitive, the methods are the primitive's, and the class tag, JSON, instanceof and
// getPrototypeOf follow its type. Object is a function (ToObject), Boolean and
// Function are globals with their prototypes. defineProperty's writable,
// configurable and enumerable default to false on a new property: a write to a
// read-only one is dropped (sloppy code), a non-configurable one survives delete and
// refuses a redefinition, and getOwnPropertyNames lists a non-enumerable one where it
// was defined. A null-prototype object has no ToPrimitive. Values read off node 24.
function s100() {
    const ns = [7, 2.5, -0, 255]
    const a = new Number(ns[0]), b = Object(ns[0]), c = ({}).valueOf.call(ns[0])
    check("wr100a", typeof a === "object" && a + 1 === 8 && typeof b === "object" && b + 1 === 8 &&
        typeof c === "object" && c + 1 === 8 && a !== b && a == 7 && !(a === 7) && a.valueOf() === 7 &&
        String(a) === "7" && "" + a === "7" && `${a}` === "7")
    check("wr100b", new Number(ns[1]).toFixed(2) === "2.50" && Number.prototype.toFixed(2) === "0.00" &&
        new Number(ns[3]).toString(16) === "ff" && Object(ns[0]).toExponential(1) === "7.0e+0" &&
        Number.prototype.toFixed.call(new Number(3), 1) === "3.0" && String(new Number(ns[2])) === "0")
    const tag = (v) => Object.prototype.toString.call(v)
    check("wr100c", tag(a) === "[object Number]" && tag(new String("x")) === "[object String]" &&
        tag(new Boolean(false)) === "[object Boolean]" && tag(Number.prototype) === "[object Number]" &&
        tag(Boolean.prototype) === "[object Boolean]")
    check("wr100d", JSON.stringify([a, new String("abc"), new Boolean(true), { k: Object(false) }]) === '[7,"abc",true,{"k":false}]' &&
        JSON.stringify({ a: new Number(ns[1]), b: [new String("q")] }) === '{"a":2.5,"b":["q"]}' &&
        JSON.stringify(new Boolean(false)) === "false")
    check("wr100e", a instanceof Number && !(ns[0] instanceof Number) && new String("ab") instanceof String &&
        new Boolean(true) instanceof Boolean && Object(true) instanceof Boolean && !(a instanceof String) &&
        a instanceof Object && !(Number.prototype instanceof Number) && Object.prototype.valueOf.call("s") instanceof String)
    const s = new String("abc")
    check("wr100f", typeof s === "object" && s.length === 3 && s + "!" === "abc!" && s.toUpperCase() === "ABC" &&
        String(s) === "abc" && s.charAt(1) === "b" && s.valueOf() === "abc" && s.slice(1) === "bc" &&
        s.indexOf("c") === 2 && s.at(-1) === "c" && String.prototype.valueOf.call(new String("q")) === "q")
    const f = new Boolean(false)
    let bmsg = ""
    try { Boolean.prototype.toString.call(1) } catch (e) { bmsg = e.message }
    check("wr100g", typeof f === "object" && !!f && f.valueOf() === false && String(f) === "false" && f == false &&
        (f ? "y" : "n") === "y" && true.toString() === "true" && false.valueOf() === false &&
        typeof true.toString === "function" && Boolean.prototype.valueOf() === false &&
        Boolean.prototype.valueOf.call(new Boolean(true)) === true && bmsg === "Boolean.prototype.toString requires that 'this' be a Boolean")
    check("wr100h", new Number(3) < new Number(4) && new Number(2) * new Number(5) === 10 && -new Number(4) === -4 &&
        new String("2") * 3 === 6 && typeof Number(new Number(5)) === "number" && Number(new Number(5)) === 5 &&
        new Number(1) !== new Number(1) && !(new Number(1) == new Number(1)) && typeof new Number() === "object" &&
        [String(new Number()), String(new String()), String(new Boolean())].join() === "0,,false")
    const o = {}, fn = function () {}
    const no = new Object()
    no.a = 1
    check("wr100i", typeof Object("s") === "object" && typeof Object(true) === "object" && typeof Object(null) === "object" &&
        typeof Object(undefined) === "object" && typeof Object() === "object" && Object(o) === o && Object(fn) === fn &&
        new Object(o) === o && JSON.stringify(no) === '{"a":1}' && typeof Object === "function" && Object.name === "Object" &&
        Object.length === 1 && [1, "a"].map(Object).map((x) => typeof x).join() === "object,object" &&
        "prototype" in Object && "keys" in Object && "isArray" in Array)
    const gp = Object.getPrototypeOf
    const w = new Number(ns[0])
    w.tag = "x"
    check("wr100j", gp(new Number(1)) === Number.prototype && gp(Object("s")) === String.prototype &&
        gp(new Boolean(true)) === Boolean.prototype && gp(true) === Boolean.prototype && gp(Number.prototype) === Object.prototype &&
        Number.prototype + 1 === 1 && String(Number.prototype) === "0" && String.prototype.length === 0 &&
        typeof Boolean.prototype === "object" && w.tag === "x" && Object.keys(w).join() === "tag" && w + 1 === 8)
    const vals = [1, 2, 3]
    const d1 = {}
    Object.defineProperty(d1, "x", { value: vals[0] })
    d1.x = vals[1]
    d1["x"] += 5
    d1.x++
    const d2 = {}
    Object.defineProperty(d2, "x", { value: vals[0], writable: true })
    d2.x = vals[1]
    const d3 = { x: vals[0] }
    Object.defineProperty(d3, "x", { writable: false })
    d3.x = vals[2]
    check("dp100k", d1.x === 1 && d2.x === 2 && d3.x === 1 && Object.keys(d3).join() === "x" &&
        !delete d1.x && d1.x === 1 && "x" in d1)
    const d4 = {}
    Object.defineProperty(d4, "x", { value: vals[0], configurable: true })
    const del4 = delete d4.x
    const d5 = {}
    Object.defineProperty(d5, "x", { value: vals[0] })
    Object.defineProperty(d5, "x", { value: vals[0] })
    const redef = (obj, desc) => { try { Object.defineProperty(obj, "x", desc); return "ok" } catch (e) { return e.message } }
    const d6 = {}
    Object.defineProperty(d6, "x", { get: () => vals[0] })
    const d7 = {}
    Object.defineProperty(d7, "x", { value: vals[0], writable: true })
    Object.defineProperty(d7, "x", { value: vals[2] })
    const re7 = redef(d7, { value: vals[1] })
    Object.defineProperty(d7, "x", { writable: false })
    d7.x = vals[0]
    check("dp100l", del4 && d4.x === undefined && !("x" in d4) && d5.x === 1 &&
        redef(d5, { value: vals[1] }) === "Cannot redefine property: x" && redef(d5, { enumerable: true }) === "Cannot redefine property: x" &&
        redef(d6, { get: () => vals[1] }) === "Cannot redefine property: x" && re7 === "ok" && d7.x === 2 &&
        redef(d7, { writable: true }) === "Cannot redefine property: x")
    const h = { a: 1 }
    Object.defineProperty(h, "h", { value: vals[0] })
    h.b = 2
    const dps = Object.defineProperties({}, { p: { value: 1 }, q: { value: 2, enumerable: true } })
    const cr = Object.create({}, { z: { value: vals[0] } })
    cr.z = 9
    const en = {}
    Object.defineProperty(en, "x", { value: vals[0], enumerable: true })
    const cp = Object.assign({}, en)
    cp.x = 4
    check("dp100m", Object.getOwnPropertyNames(h).join() === "a,h,b" && Object.keys(h).join() === "a,b" &&
        Object.getOwnPropertyNames(dps).join() === "p,q" && cr.z === 1 && Object.getOwnPropertyNames(cr).join() === "z" &&
        cp.x === 4 && JSON.stringify(h) === '{"a":1,"b":2}')
    class C100 {}
    check("fn100n", gp(function () {}) === Function.prototype && gp(() => 1) === Function.prototype &&
        typeof Function === "function" && Function.name === "Function" && (function () {}) instanceof Function &&
        (() => 1) instanceof Function && !({} instanceof Function) && gp(C100) === Function.prototype &&
        C100 instanceof Function && C100 instanceof Object && typeof Function.prototype === "function" &&
        gp(Function.prototype) === Object.prototype)
    const prim = (f) => { try { return f() } catch (e) { return e instanceof TypeError ? e.message : "?" } }
    const nul = Object.create(null)
    const own = Object.create(null)
    own.toString = () => "ok"
    check("np100o", prim(() => String(nul)) === "Cannot convert object to primitive value" &&
        prim(() => "" + nul) === "Cannot convert object to primitive value" &&
        prim(() => nul + 1) === "Cannot convert object to primitive value" &&
        prim(() => `${nul}`) === "Cannot convert object to primitive value" && String(own) === "ok")
}

// ===== END SECTIONS =====

function main() {
    s44() // SECTION-CALL 44
    s45() // SECTION-CALL 45
    s46() // SECTION-CALL 46
    s47() // SECTION-CALL 47
    s48() // SECTION-CALL 48
    s49() // SECTION-CALL 49
    s50() // SECTION-CALL 50
    s51() // SECTION-CALL 51
    s52() // SECTION-CALL 52
    s53() // SECTION-CALL 53
    s54() // SECTION-CALL 54
    s55() // SECTION-CALL 55
    s56() // SECTION-CALL 56
    s58() // SECTION-CALL 58
    s59() // SECTION-CALL 59
    s60() // SECTION-CALL 60
    s61() // SECTION-CALL 61
    s62() // SECTION-CALL 62
    s63() // SECTION-CALL 63
    s64() // SECTION-CALL 64
    s65() // SECTION-CALL 65
    s66() // SECTION-CALL 66
    s67() // SECTION-CALL 67
    s68() // SECTION-CALL 68
    s69() // SECTION-CALL 69
    s70() // SECTION-CALL 70
    s71() // SECTION-CALL 71
    s73() // SECTION-CALL 73
    s74() // SECTION-CALL 74
    s75() // SECTION-CALL 75
    s76() // SECTION-CALL 76
    s77() // SECTION-CALL 77
    s78() // SECTION-CALL 78
    s79() // SECTION-CALL 79
    s80() // SECTION-CALL 80
    s81() // SECTION-CALL 81
    s82() // SECTION-CALL 82
    s83() // SECTION-CALL 83
    s84() // SECTION-CALL 84
    s85() // SECTION-CALL 85
    s86() // SECTION-CALL 86
    s87() // SECTION-CALL 87
    s88() // SECTION-CALL 88
    s89() // SECTION-CALL 89
    s90() // SECTION-CALL 90
    s91() // SECTION-CALL 91
    s92() // SECTION-CALL 92
    s93() // SECTION-CALL 93
    s94() // SECTION-CALL 94
    s95() // SECTION-CALL 95
    s96() // SECTION-CALL 96
    s97() // SECTION-CALL 97
    s98() // SECTION-CALL 98
    s99() // SECTION-CALL 99
    s100() // SECTION-CALL 100
    println("full: " + checks + " checks, " + failures + " failures")
    return failures
}
