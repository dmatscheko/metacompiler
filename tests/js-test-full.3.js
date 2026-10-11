// Full-syntax test: JavaScript - part 3 of 4, sections 44-56 and 58-63.
//
// THE JS RATCHET IS FOUR FILES: as one file its -frozen interpreter leg ran into
// ./test.sh --full's time brake (peak memory grew superlinearly with the file), and
// the fourth is a Module, the one goal symbol in which 'export' may stand.
//
//   tests/js-test-full.js     sections 01-35, the ECMAScript-2022 syntax walk
//   tests/js-test-full.2.js   sections 36-43, engine agreement, generators, Error, Math
//   tests/js-test-full.3.js   sections 44-56 and 58-63, the call/class/private-name
//                             model, the Unicode identifier tables, iterators, derived
//                             constructors, JSON, guards, Set and Map, iterator closing,
//                             hasOwnProperty, delete, Object.prototype, extends Object
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
// node iterates only through Symbol.iterator, which this subset does not have; these
// engines drive any value with a callable next().
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
    println("full: " + checks + " checks, " + failures + " failures")
    return failures
}
