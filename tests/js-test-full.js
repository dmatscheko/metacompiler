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

// ===== SECTION 20: regular expressions =====
// Every expected value below was cross-checked against node v24.
function s20() {
    var re = /a(b+)c/
    check("re01", re.source === "a(b+)c" && re.flags === "" && re.global === false)
    check("re02", re.test("xxabbbc") && !re.test("ac"))
    var m = re.exec("xxabbbc")
    check("re03", m[0] === "abbbc" && m[1] === "bbb" && m.index === 2 && m.length === 2)
    check("re04", m.input === "xxabbbc" && m.groups === undefined)
    check("re05", re.exec("nope") === null)
    // Flags: i folds case, s makes '.' match a newline, m makes ^ and $ line anchors -
    // and WITHOUT m they are string anchors, which is the one place JavaScript differs
    // from Ruby.
    check("re06", /AB/i.test("xxab"))
    check("re07", /a.b/s.test("a\nb") && !/a.b/.test("a\nb"))
    check("re08", /^b$/m.test("a\nb\nc") && !/^b$/.test("a\nb\nc"))
    // g is not a match mode, it is lastIndex iteration state.
    var g = /a/g
    var hits = ""
    var one = g.exec("aXaXa")
    while (one !== null) {
        hits = hits + one.index
        one = g.exec("aXaXa")
    }
    check("re09", hits === "024" && g.lastIndex === 0)
    // y (sticky) anchors the search at lastIndex.
    var st = /a/y
    check("re10", !st.test("ba"))
    st.lastIndex = 1
    check("re11", st.test("ba") && st.lastIndex === 2)
    // Named groups reach the match through .groups.
    var nm = /(?<y>[0-9]{4})-(?<m>[0-9]{2})/.exec("on 2024-05-06")
    check("re12", nm.groups.y === "2024" && nm.groups.m === "05" && nm.index === 3)
    // String.prototype.match / matchAll / search.
    check("re13", "a1b22".match(/\d+/)[0] === "1")
    check("re14", "a1b22".match(/\d+/g).length === 2)
    check("re15", "abc".match(/z/) === null)
    check("re16", "abc".search(/c/) === 2 && "abc".search(/z/) === -1)
    var all = [... "a1b2".matchAll(/(\w)(\d)/g)]
    var joined = ""
    for (var i = 0; i < all.length; i++) { joined = joined + all[i][1] + all[i][2] }
    check("re17", joined === "a1b2")
    // replace / replaceAll, with the $-template and with a function.
    check("re18", "a1b22".replace(/\d+/, "#") === "a#b22")
    check("re19", "a1b22".replace(/\d+/g, "#") === "a#b#")
    check("re20", "one two".replace(/(\w+) (\w+)/, "$2 $1") === "two one")
    check("re21", "abc".replace(/b/, "[$&]") === "a[b]c")
    check("re22", "ab".replace(/a/, "$$") === "$b")
    check("re23", "2024-05".replace(/(?<y>\d{4})-(?<m>\d{2})/, "$<m>/$<y>") === "05/2024")
    check("re24", "aXbXc".replace(/X/g, function (w, off) { return "(" + off + ")" }) === "a(1)b(3)c")
    check("re25", "aaa".replaceAll(/a/g, "b") === "bbb")
    // split with a pattern keeps the separator's capture groups.
    check("re26", "a,b,,c".split(/,/).length === 4)
    check("re27", "a-b_c".split(/([-_])/).length === 5)
    check("re28", "a1b".split(/\d/)[1] === "b")
    // A string argument is not a pattern: these keep the plain string behaviour.
    check("re29", "a.c".replace(".", "-") === "a-c" && "a.c".split(".").length === 2)
    // new RegExp / RegExp with the flags arriving at run time.
    var dyn = new RegExp("b+", "i")
    check("re30", dyn.test("aBBc") && dyn.source === "b+" && dyn.flags === "i")
    var dyn2 = RegExp("\\d")
    check("re31", dyn2.test("x9") && !dyn2.test("xy"))
    check("re32", typeof RegExp === "function" && typeof dyn === "object")
    // Backreferences, alternation, lazy quantifiers and a group that did not take part.
    check("re33", /(a)\1/.test("aa") && !/(a)\1/.test("ab"))
    check("re34", /<(.+?)>/.exec("<a><b>")[1] === "a")
    check("re35", /(a)(b)?/.exec("a")[2] === undefined)
    check("re36", /^(cat|dog)s?$/.test("dogs"))
    // Lookahead, and the escape forms the engine reads out of the RAW body.
    check("re37", /foo(?!bar)/.test("foobaz") && !/foo(?!bar)/.test("foobar"))
    check("re38", /a\/b/.test("a/b") && /\[\]/.test("[]"))
    check("re39", /^\d{2,3}$/.test("123") && !/^\d{2,3}$/.test("1234"))
    check("re40", /[a-z0-9_$]+/.exec("__x9!")[0] === "__x9")
    // A '/' after a value is still division, which is what keeps the literal out of the
    // way of ordinary arithmetic.
    var division = 8 / 2 / 2
    check("re41", division === 2)
    check("re42", /x/.toString() === "/x/" && /x/gi.toString() === "/x/gi")
    // Each evaluation of a literal yields a NEW object, so the lastIndex of a global
    // pattern written inside a loop starts over every time round.
    var counted = 0
    for (var k = 0; k < 3; k++) { if (/z/g.test("z")) { counted = counted + 1 } }
    check("re43", counted === 3)
    // Named backreferences, including a FORWARD one (the group is declared later).
    check("re44", /(?<n>a)\k<n>/.test("aa") && !/(?<n>a)\k<n>/.test("ab"))
    check("re45", /\k<n>(?<n>a)/.test("a"))
    check("re46", /(?<w>ab)-\k<w>/.exec("zab-ab")[0] === "ab-ab")
    check("re47", "xayb".replace(/(?<c>[ab])/g, "[$<c>]") === "x[a]y[b]")
    // Annex B legacy octal: a backslash-digit escape with no such group is a
    // CHARACTER, not a backreference. \1 is U+0001, so it cannot match the empty
    // string; 8 and 9 are not octal digits and stay identity escapes; and 4-7 admit
    // only two more digits, which is why \400 reads as \40 followed by "0".
    check("re48", !/\1/.test("") && /(a)\1/.test("aa") && !/(a)\2/.test("a"))
    check("re49", /\12/.test("\n") && /\101/.test("A") && /\400/.test(" 0"))
    check("re50", /\8/.test("8") && /\9/.test("9"))
    // Sticky: the match must BEGIN at lastIndex, not merely after it.
    var sticky = /b/y
    sticky.lastIndex = 1
    check("re51", sticky.test("ab") && sticky.lastIndex === 2)
    var sticky2 = /b/y
    sticky2.lastIndex = 0
    check("re52", !sticky2.test("ab"))
    // \Q ... \E is a Java/Kotlin quote region and NOT a JavaScript one: here \Q and
    // \E are identity escapes, so this pattern is the literal text "Qa.bE". The
    // shared engine offers quote regions behind a dialect flag that JavaScript does
    // not pass, and this check is what keeps it from creeping in.
    check("re53", !/\Qa.b\E/.test("a.b") && /\Qa.b\E/.test("Qa.bE"))
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
    s20() // SECTION-CALL 20
    println("full: " + checks + " checks, " + failures + " failures")
    return failures
}
