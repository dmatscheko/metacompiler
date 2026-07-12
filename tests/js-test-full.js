// Full-syntax test: JavaScript (ECMAScript 2022 core grammar).
//
// This file belongs to the SECOND test group (./test.sh --full): it is NOT part
// of the default matrix. The goal of the metacompiler is to support the full
// languages; this file is the ratchet that measures how far the js grammars
// are. It walks the whole practical ECMAScript syntax, one self-contained
// SECTION per language area. The --full runner runs the file, and whenever a
// grammar aborts it removes the section around the error and retries - so the
// report lists every unsupported section, not just the first.
//
// Conventions (shared by every *-test-full.* file):
//   - prologue (before the first SECTION marker): the check helper only
//   - each section: '// ===== SECTION <nn>: <name> =====', top-level,
//     self-contained, no references to other sections
//   - main() calls each section via a line tagged 'SECTION-CALL <nn>'
//     and prints the summary line 'full: <checks> checks, <failures> failures'
//   - main() returns the failure count (exit 0 == full support, verified)
//
// Deliberately out of scope (not syntax, or unrunnable in this harness):
// module import/export, `with`, eval/Function, the standard library (Symbol,
// Proxy, Promise combinators, JSON, RegExp methods, ...). Async/generator
// SYNTAX is covered; where running it needs an event loop, functions are only
// defined and type-checked, never awaited.
//
// Hand-written for the metacompiler project (Apache-2.0, no copied test-suite
// code), organized after the ECMAScript 2022 specification grammar with the
// ANTLR grammars-v4 JavaScript grammar as a coverage checklist.

var failures = 0
var checks = 0
function check(id, cond) {
    checks = checks + 1
    if (!cond) {
        println("FAIL " + id)
        failures = failures + 1
    }
}

// ===== SECTION 01: baseline =====
// Condensed re-assertion of the feature-matrix basics this file builds on.
function s01() {
    var n = 0
    for (var i = 0; i < 3; i++) { n = n + i }
    check("bas1", n === 3)
    var o = { a: 1, "b": 2 }
    o.c = o.a + o["b"]
    check("bas2", o.c === 3)
    var arr = [1, 2, 3]
    check("bas3", arr.length === 3 && arr[2] === 3)
    function add(x, y) { return x + y }
    check("bas4", add(2, 3) === 5)
    var t = 0
    try { throw "boom" } catch (e) { t = e === "boom" ? 1 : 2 } finally { t = t + 10 }
    check("bas5", t === 11)
}

// ===== SECTION 02: numeric literal forms =====
function s02() {
    check("num1", 0xff === 255)
    check("num2", 0b1010 === 10)
    check("num3", 0o17 === 15)
    check("num4", 1_000_000 === 1000000)
    check("num5", .5 === 0.5)
    check("num6", 1e3 === 1000 && 2.5e-2 === 0.025)
    check("num7", typeof 10n === "bigint")
}

// ===== SECTION 03: string escapes =====
function s03() {
    check("str1", "A" === "A")
    check("str2", "\x41" === "A")
    check("str3", "\u{1F600}".length === 2)
    check("str4", "a\
b" === "ab")
    check("str5", '\'' === "'" && "\"".length === 1)
    check("str6", "\0".length === 1)
}

// ===== SECTION 04: template literals =====
function s04() {
    var x = 6
    check("tpl1", `x is ${x}` === "x is 6")
    check("tpl2", `${x} + ${x} = ${x + x}` === "6 + 6 = 12")
    check("tpl3", `line1
line2`.length === 11)
    check("tpl4", `outer ${`inner ${x}`}` === "outer inner 6")
    function tag(strings, a, b) { return strings[0] + a + strings[1] + b }
    check("tpl5", tag`x${1}y${2}` === "x1y2")
    function rawtag(s) { return s.raw[0] }
    check("tpl6", rawtag`a\nb` === "a\\nb")
}

// ===== SECTION 05: exponent and compound assignment =====
function s05() {
    check("exp1", 2 ** 10 === 1024)
    check("exp2", 2 ** 3 ** 2 === 512) // right-associative
    var v = 3
    v **= 2
    check("exp3", v === 9)
    var a = null, b = 1, c
    a ??= 5
    b ||= 7
    c = 2; c &&= 4
    check("exp4", a === 5 && b === 1 && c === 4)
    var s = 0
    s ||= 9
    check("exp5", s === 9)
}

// ===== SECTION 06: optional chaining and nullish coalescing =====
function s06() {
    var o = { a: { b: 2 }, f: function () { return 3 } }
    var missing = null
    check("opt1", o?.a?.b === 2)
    check("opt2", missing?.a?.b === undefined)
    check("opt3", o?.["a"]?.["b"] === 2)
    check("opt4", o.f?.() === 3)
    check("opt5", missing?.f?.() === undefined)
    check("nul1", (null ?? 5) === 5)
    check("nul2", (0 ?? 7) === 0)
    check("nul3", (undefined ?? "d") === "d")
}

// ===== SECTION 07: arrow functions =====
function s07() {
    var inc = x => x + 1
    check("arw1", inc(4) === 5)
    var add = (a, b) => a + b
    check("arw2", add(2, 3) === 5)
    var blk = (a) => { var t = a * 2; return t }
    check("arw3", blk(5) === 10)
    var zero = () => 42
    check("arw4", zero() === 42)
    var objret = () => ({ k: 1 })
    check("arw5", objret().k === 1)
    var curried = a => b => a + b
    check("arw6", curried(1)(2) === 3)
}

// ===== SECTION 08: default, rest params and trailing commas =====
function s08() {
    function withDefault(a, b = 10, c = a + b) { return a + b + c }
    check("par1", withDefault(1) === 22)
    check("par2", withDefault(1, 2) === 6)
    function withRest(first, ...rest) { return first + rest.length + rest[0] }
    check("par3", withRest(1, 10, 20) === 13)
    function trail(a, b,) { return a + b }
    check("par4", trail(1, 2,) === 3)
    var arrTrail = [1, 2, 3,]
    var objTrail = { a: 1, b: 2, }
    check("par5", arrTrail.length === 3 && objTrail.b === 2)
}

// ===== SECTION 09: enhanced object literals =====
function s09() {
    var name = "dyn"
    var val = 7
    var o = {
        val,
        [name + "Key"]: 5,
        method(x) { return x * 2 },
        get v() { return this._v + 1 },
        set v(x) { this._v = x * 10 },
    }
    check("obj1", o.val === 7)
    check("obj2", o.dynKey === 5)
    check("obj3", o.method(21) === 42)
    o.v = 2
    check("obj4", o._v === 20 && o.v === 21)
    var kw = { class: 1, new: 2, for: 3 }
    check("obj5", kw.class === 1 && kw.new === 2 && kw.for === 3)
}

// ===== SECTION 10: destructuring =====
function s10() {
    var [a, b, , d = 9] = [1, 2, 3]
    check("des1", a === 1 && b === 2 && d === 9)
    var [x, ...restArr] = [10, 20, 30]
    check("des2", x === 10 && restArr.length === 2 && restArr[1] === 30)
    var { p, q: renamed, r = 5 } = { p: 1, q: 2 }
    check("des3", p === 1 && renamed === 2 && r === 5)
    var { s: { deep } } = { s: { deep: 4 } }
    check("des4", deep === 4)
    var { m, ...restObj } = { m: 1, n: 2, o: 3 }
    check("des5", m === 1 && restObj.n === 2 && restObj.o === 3)
    function dparam({ k, j = 2 }, [first]) { return k + j + first }
    check("des6", dparam({ k: 1 }, [10]) === 13)
    var swap1 = 1, swap2 = 2
    ;[swap1, swap2] = [swap2, swap1]
    check("des7", swap1 === 2 && swap2 === 1)
    var total = 0
    for (var [k2, v2] of [[1, 2], [3, 4]]) { total = total + k2 * v2 }
    check("des8", total === 14)
}

// ===== SECTION 11: spread =====
function s11() {
    var base = [2, 3]
    var arr = [1, ...base, 4]
    check("spr1", arr.length === 4 && arr[1] === 2 && arr[3] === 4)
    function sum3(a, b, c) { return a + b + c }
    check("spr2", sum3(...base, 10) === 15)
    var o1 = { a: 1, b: 2 }
    var o2 = { ...o1, b: 3, c: 4 }
    check("spr3", o2.a === 1 && o2.b === 3 && o2.c === 4)
    var chars = [..."ab"]
    check("spr4", chars.length === 2 && chars[0] === "a")
}

// ===== SECTION 12: this binding =====
function s12() {
    var obj = {
        v: 4,
        m: function () { return this.v },
        arrow: function () { var f = () => this.v; return f() },
    }
    check("ths1", obj.m() === 4)
    check("ths2", obj.arrow() === 4)
    function Ctor(n) { this.n = n }
    var inst = new Ctor(6)
    check("ths3", inst.n === 6)
}

// ===== SECTION 13: classes (full) =====
function s13() {
    var Point = class {
        static origin = "0,0"
        static { this.frozen = true }
        static dist(a, b) { return (a.x - b.x) + (a.y - b.y) }
        #secret = 41
        tag = "pt"
        constructor(x, y) { this.x = x; this.y = y }
        get sum() { return this.x + this.y }
        set sum(v) { this.x = v; this.y = 0 }
        ["comp" + "uted"]() { return 9 }
        #bump() { return this.#secret + 1 }
        reveal() { return this.#bump() }
    }
    var pt = new Point(3, 4)
    check("cls1", pt.sum === 7 && pt.tag === "pt")
    pt.sum = 10
    check("cls2", pt.x === 10 && pt.y === 0)
    check("cls3", Point.origin === "0,0" && Point.frozen === true)
    check("cls4", Point.dist({ x: 5, y: 5 }, { x: 2, y: 1 }) === 7)
    check("cls5", pt.computed() === 9)
    check("cls6", pt.reveal() === 42)
    class Base {
        constructor() { this.kind = "base" }
        who() { return "base" }
    }
    class Derived extends Base {
        constructor() { super(); this.kind = "derived" }
        who() { return "derived+" + super.who() }
    }
    var dd = new Derived()
    check("cls7", dd.kind === "derived" && dd.who() === "derived+base")
    check("cls8", dd instanceof Derived && dd instanceof Base)
    function NT() { this.viaNew = new.target !== undefined }
    check("cls9", (new NT()).viaNew === true)
}

// ===== SECTION 14: iteration statements =====
function s14() {
    var sum = 0
    for (var v of [1, 2, 3]) { sum = sum + v }
    check("itr1", sum === 6)
    var sc = ""
    for (var ch of "ab") { sc = sc + ch + "." }
    check("itr2", sc === "a.b.")
    var keys = ""
    var obj = { x: 1, y: 2 }
    for (var k in obj) { keys = keys + k }
    check("itr3", keys === "xy")
    var n = 0
    do { n = n + 1 } while (n < 3)
    check("itr4", n === 3)
    var hits = 0
    outer:
    for (var i = 0; i < 3; i++) {
        for (var j = 0; j < 3; j++) {
            if (j === 1) { continue outer }
            if (i === 2) { break outer }
            hits = hits + 1
        }
    }
    check("itr5", hits === 2)
}

// ===== SECTION 15: generators =====
function s15() {
    function* gen() {
        yield 1
        yield 2
        yield* [3, 4]
        return 99
    }
    var out = []
    for (var v of gen()) { out[out.length] = v }
    check("gen1", out.length === 4 && out[0] === 1 && out[3] === 4)
    var it = gen()
    var first = it.next()
    check("gen2", first.value === 1 && first.done === false)
    it.next(); it.next(); it.next()
    var last = it.next()
    check("gen3", last.value === 99 && last.done === true)
    var genExpr = function* () { yield 5 }
    check("gen4", genExpr().next().value === 5)
    var withSend = function* () { var got = yield 1; yield got * 2 }
    var sit = withSend()
    sit.next()
    check("gen5", sit.next(21).value === 42)
}

// ===== SECTION 16: async syntax =====
// Defined and type-checked only: running them needs an event loop.
function s16() {
    async function af() { return 5 }
    check("asy1", typeof af === "function")
    var aArrow = async (x) => x + 1
    check("asy2", typeof aArrow === "function")
    async function withAwait(p) { var r = await p; return r + 1 }
    check("asy3", typeof withAwait === "function")
    async function* agen() { yield 1 }
    check("asy4", typeof agen === "function")
    async function loopAwait(xs) {
        var sum = 0
        for await (var x of xs) { sum = sum + x }
        return sum
    }
    check("asy5", typeof loopAwait === "function")
    var obj = { async m() { return 1 } }
    check("asy6", typeof obj.m === "function")
}

// ===== SECTION 17: regular expression literals =====
function s17() {
    var re = /ab+c/
    check("rex1", typeof re === "object")
    var flags = /x/gimsy
    check("rex2", typeof flags === "object")
    var cls = /[a-z0-9_$]+\/\d{2,3}/
    check("rex3", typeof cls === "object")
    var named = /(?<year>[0-9]{4})-(?<month>[0-9]{2})/
    check("rex4", typeof named === "object")
    var division = 8 / 2 / 2 // Still plain division after a value.
    check("rex5", division === 2)
}

// ===== SECTION 18: operators, misc =====
function s18() {
    var c = (1, 2, 3)
    check("msc1", c === 3)
    check("msc2", void 0 === undefined)
    var o = { a: 1 }
    check("msc3", "a" in o && !("b" in o))
    delete o.a
    check("msc4", !("a" in o))
    check("msc5", typeof notDeclaredAnywhere === "undefined")
    var grade = 87 >= 90 ? "A" : 87 >= 80 ? "B" : "C"
    check("msc6", grade === "B")
    var bit = (5 & 3) | (4 ^ 1) | (1 << 4) | (32 >> 1) | (-8 >>> 28)
    check("msc7", bit === (1 | 5 | 16 | 16 | 15))
    check("msc8", ~5 === -6)
    var neg = -(-5)
    check("msc9", neg === +5)
}

// ===== SECTION 19: exception refinements and directives =====
function s19() {
    var caught = 0
    try { throw 1 } catch { caught = 1 } // optional catch binding
    check("exc1", caught === 1)
    function strict() {
        "use strict"
        return 1
    }
    check("exc2", strict() === 1)
    var reached = 0
    l1: { reached = 1; break l1; reached = 2 } // labeled block with break
    check("exc3", reached === 1)
}

// ===== END SECTIONS =====

function main() {
    s01() // SECTION-CALL 01
    s02() // SECTION-CALL 02
    s03() // SECTION-CALL 03
    s04() // SECTION-CALL 04
    s05() // SECTION-CALL 05
    s06() // SECTION-CALL 06
    s07() // SECTION-CALL 07
    s08() // SECTION-CALL 08
    s09() // SECTION-CALL 09
    s10() // SECTION-CALL 10
    s11() // SECTION-CALL 11
    s12() // SECTION-CALL 12
    s13() // SECTION-CALL 13
    s14() // SECTION-CALL 14
    s15() // SECTION-CALL 15
    s16() // SECTION-CALL 16
    s17() // SECTION-CALL 17
    s18() // SECTION-CALL 18
    s19() // SECTION-CALL 19
    println("full: " + checks + " checks, " + failures + " failures")
    return failures
}
