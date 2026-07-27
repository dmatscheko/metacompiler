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
// cross-module import linkage, `with`, eval/Function, the standard library
// (Symbol, Proxy, Promise combinators, JSON, RegExp methods, ...).
// Async/generator SYNTAX is covered; where running it needs an event loop,
// functions are only defined and type-checked, never awaited.
// `export` IS covered (section 34): an import here runs the referenced file
// into the shared global scope, so an export is a transparent prefix on its
// declaration and the specifier forms are no-ops.
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

// ===== SECTION 21: interpreter/compiler agreement ratchet =====
// Assigning arr.length TRUNCATES (or pads) in JavaScript; the compiler runtime used
// to abort with "invalid array index length". NaN and Infinity are global BINDINGS;
// the interpreter half raised "variable not defined" for both. Values are node v24's.
function s21() {
    var a = [1, 2, 3]
    a.length = 1
    check("agr1", a.length === 1 && a[0] === 1)
    var b = [1]
    b.length = 3
    check("agr2", b.length === 3 && b[2] === undefined)
    var c = [1, 2]
    c.length = 0
    check("agr3", c.length === 0 && c.join(",") === "")
    check("agr4", NaN !== NaN && ("" + NaN) === "NaN")
    check("agr5", Infinity > 1e308 && ("" + Infinity) === "Infinity")
    check("agr6", -Infinity < -1e308)
    check("agr7", typeof undefined === "undefined")
}
// ===== SECTION 22: value rendering (String / ToString) =====
// The two halves used to hand the raw value to the shared println host function, so
// Go's %v reached stdout: "<nil>" for null, "[1 2 3]" for an array and a map dump for
// an object, where node's String() says "null", "1,2,3" and "[object Object]". Values
// are node v24's. The println lines at the end ratchet the PRINT path itself, which no
// check() can reach; node's console.log uses its own inspect format, so only the
// check() assertions are settled against node.
function s22() {
    class Tagged { constructor(n) { this.n = n } toString() { return "T(" + this.n + ")" } }
    check("str1", String(null) === "null" && String(undefined) === "undefined")
    check("str2", String(true) === "true" && String(1.5) === "1.5")
    check("str3", String([1, 2, 3]) === "1,2,3")
    check("str4", String([1, [2, 3]]) === "1,2,3")
    check("str5", String([1, null, undefined, 2]) === "1,,,2")
    check("str6", String({}) === "[object Object]")
    check("str7", String({ a: 1 }) === "[object Object]")
    check("str8", String(new Tagged(4)) === "T(4)")
    check("str9", ("" + new Tagged(5)) === "T(5)")
    check("str10", String("x") === "x" && String([]) === "")
    check("str11", String.fromCharCode(65, 66) === "AB")
    println(null, undefined, [1, 2, 3], { a: 1 }, "end")
    println(1, "x", true, [1, [2, 3]], new Tagged(6))
}

// ===== SECTION 23: ToPrimitive in ==, the relational operators and + =====
// The compiler half compared an OBJECT operand without ever taking its primitive
// form, so [1] == 1 and [2] > 1 were false. Values are node v24's.
function s23() {
    class Tagged { constructor(n) { this.n = n } toString() { return "T(" + this.n + ")" } }
    check("prim1", [1] == 1)
    check("prim2", [] == false)
    check("prim3", [2] > 1)
    check("prim4", [1] <= 1 && [1] >= 1)
    check("prim5", ({}) == "[object Object]")
    check("prim6", null == undefined)
    check("prim7", !(null == 0))
    check("prim8", !(NaN == NaN))
    check("prim9", [] == "")
    check("prim10", ([1, 2] + [3]) === "1,23")
    check("prim11", ("x" + {}) === "x[object Object]")
    check("prim12", (1 + [2]) === "12")
    var box = { valueOf: function () { return 42 } }
    check("prim13", box == 42 && box > 41 && (box + 1) === 43)
    check("prim14", ("" + new Tagged(7)) === "T(7)")
    var a1 = [1]
    var a2 = [1]
    check("prim15", !(a1 == a2) && a1 == a1)
}

// ===== SECTION 24: new, instanceof and delete =====
// 'new F() instanceof F' was false in the compiler for a plain constructor FUNCTION
// (only class descriptors carried an instanceof chain), an object RETURNED by a
// constructor was discarded, and 'delete a[1]' was a silent no-op.
// NOTE: 'i in arr' stays TRUE after 'delete arr[i]' in both halves, where node says
// false. A real hole needs per-slot state on the array, and the interpreter half's
// arrays are host arrays that cannot carry any (the -frozen runtime rejects a
// non-index key on an array outright), so the two halves agree on blanking the slot.
function s24() {
    function Pt(x) { this.x = x }
    function Ret() { return { made: true } }
    class Tagged { constructor(n) { this.n = n } }
    var p = new Pt(3)
    check("new1", p.x === 3)
    check("new2", p instanceof Pt)
    check("new3", p instanceof Object)
    check("new4", !(p instanceof Tagged))
    check("new5", new Ret().made === true)
    check("new6", new Tagged(1) instanceof Tagged)
    check("new7", [] instanceof Array && !([] instanceof Pt))
    var arr = [1, 2, 3]
    delete arr[1]
    check("del1", arr.length === 3)
    check("del2", arr[1] === undefined)
    check("del3", arr.join(",") === "1,,3")
    var o = { a: 1, b: 2 }
    delete o.a
    check("del4", !("a" in o) && ("b" in o))
}

// ===== SECTION 25: the Object statics =====
// Object.keys / entries / values leaked the interpreter's hidden __keys bookkeeping
// slot ("a b __keys") and aborted outright in the compiler ("unknown method 'keys'").
function s25() {
    class Tagged { constructor(n) { this.n = n } }
    var o = { a: 1, b: 2 }
    check("obj1", Object.keys(o).join("|") === "a|b")
    check("obj2", Object.values(o).join("|") === "1|2")
    check("obj3", Object.entries(o).map(function (e) { return e[0] + "=" + e[1] }).join("|") === "a=1|b=2")
    var t = new Tagged(1)
    check("obj4", Object.keys(t).join("|") === "n")
    check("obj5", Object.keys([7, 8]).join("|") === "0|1")
    var d = Object.assign({ z: 0 }, { a: 1 }, { b: 2 })
    check("obj6", d.z === 0 && d.a === 1 && d.b === 2)
    check("obj7", Object.fromEntries([["k", 1], ["l", 2]]).l === 2)
    check("obj8", Object.freeze(o) === o)
}

// ===== SECTION 26: the String methods =====
// Everything here was missing from the compiler half outright, or implemented with
// the shared runtime's Java-flavoured semantics (indexOf ignoring its from-index,
// split taking no limit). Values are node v24's.
function s26() {
    check("sm1", "abcabc".indexOf("b", 2) === 4)
    check("sm2", "abcabc".lastIndexOf("b") === 4)
    check("sm3", "abc".includes("bc") && !"abc".includes("cb"))
    check("sm4", "abc".startsWith("ab") && "abc".startsWith("bc", 1))
    check("sm5", "abc".endsWith("bc") && "abc".endsWith("ab", 2))
    check("sm6", "a".padStart(3, "xy") === "xya" && "a".padEnd(3, "xy") === "axy")
    check("sm7", "ab".repeat(3) === "ababab" && "ab".repeat(0) === "")
    check("sm8", "  a  ".trimStart() === "a  " && "  a  ".trimEnd() === "  a")
    check("sm9", "abc".at(-1) === "c" && "abc".at(0) === "a" && "abc".at(9) === undefined)
    check("sm10", "a-b-c".replaceAll("-", "+") === "a+b+c")
    check("sm11", "a-b-c".replace("-", "+") === "a+b-c")
    check("sm12", "abc".concat("d", "e") === "abcde")
    check("sm13", "a,b,,c".split(",").join("|") === "a|b||c")
    check("sm14", "abc".split("").length === 3)
    check("sm15", "a,b,c".split(",", 2).join("|") === "a|b")
    check("sm16", "abc".slice(-2) === "bc" && "abc".substring(1, 2) === "b")
    check("sm17", "abcd".substr(1, 2) === "bc")
    check("sm18", "abc".localeCompare("abd") === -1 && "abc".localeCompare("abc") === 0)
    check("sm19", "abc".codePointAt(1) === 98)
    check("sm20", "aXbXc".replace("X", "[$&]") === "a[X]bXc")
    check("sm21", "abc".toUpperCase() === "ABC" && "ABC".toLowerCase() === "abc")
    check("sm22", "aa".replace("a", function (m, i) { return m + i }) === "a0a")
}

// ===== SECTION 27: the Array methods =====
// map/filter/forEach existed in the shared runtime with KOTLIN's signature (the
// callback got no index); everything else was missing. Values are node v24's.
function s27() {
    var a = [1, 2, 3]
    check("am1", a.map(function (x, i) { return x * 10 + i }).join("|") === "10|21|32")
    check("am2", a.filter(function (x, i) { return i > 0 }).join("|") === "2|3")
    check("am3", a.reduce(function (s, x) { return s + x }, 0) === 6)
    check("am4", a.reduce(function (s, x) { return s + x }) === 6)
    check("am5", a.reduceRight(function (s, x) { return s + "" + x }) === "321")
    check("am6", a.some(function (x) { return x > 2 }) && !a.some(function (x) { return x > 9 }))
    check("am7", a.every(function (x) { return x > 0 }) && !a.every(function (x) { return x > 1 }))
    check("am8", a.find(function (x) { return x > 1 }) === 2)
    check("am9", a.findIndex(function (x) { return x > 1 }) === 1)
    check("am10", a.findLast(function (x) { return x < 3 }) === 2)
    check("am11", a.findLastIndex(function (x) { return x < 3 }) === 1)
    check("am12", a.includes(2) && !a.includes(9))
    check("am13", a.lastIndexOf(3) === 2 && a.indexOf(9) === -1)
    check("am14", a.at(-1) === 3 && a.at(5) === undefined)
    check("am15", [1, [2, [3]]].flat().join("|") === "1|2|3")
    check("am16", [1, [2, [3]]].flat(2).join("|") === "1|2|3")
    check("am17", a.flatMap(function (x) { return [x, x] }).join("|") === "1|1|2|2|3|3")
    var f = [1, 2, 3, 4]
    check("am18", f.fill(0, 1, 3).join("|") === "1|0|0|4")
    var g = [1, 2, 3, 4]
    check("am19", g.splice(1, 2).join("|") === "2|3" && g.join("|") === "1|4")
    var h = [3, 1, 2]
    check("am20", h.sort().join("|") === "1|2|3")
    check("am21", [10, 9].sort().join("|") === "10|9")
    check("am22", [10, 9].sort(function (x, y) { return x - y }).join("|") === "9|10")
    check("am23", [1, 2].concat([3], 4).join("|") === "1|2|3|4")
    check("am24", [1, 2].toString() === "1,2")
    check("am25", [1, 2, 3].slice(1, -1).join("|") === "2")
    var r = [1, 2, 3]
    check("am26", r.reverse().join("|") === "3|2|1" && r[0] === 3)
    check("am27", [1, null, undefined, 2].join("-") === "1---2")
}

// ===== SECTION 28: the Number methods =====
// toFixed and toString(radix) had no implementation at all in either half's runtime.
// Values are node v24's; note that (1.005).toFixed(2) is "1.00" because the exact
// binary value of 1.005 is below it.
function s28() {
    check("nm1", (1.2345).toFixed(2) === "1.23")
    check("nm2", (2.5).toFixed(0) === "3")
    check("nm3", (1.005).toFixed(2) === "1.00")
    check("nm4", (-1.5).toFixed(0) === "-2")
    check("nm5", (3).toFixed(0) === "3" && (3).toFixed(2) === "3.00")
    check("nm6", (255).toString(16) === "ff")
    check("nm7", (10).toString(2) === "1010")
    check("nm8", (255).toString() === "255" && (255).toString(10) === "255")
    check("nm9", (0.5).toString(2) === "0.1")
}

// ===== SECTION 29: super as a property =====
class S29Base {
    constructor() { this.tag = "base" }
    get label() { return "B:" + this.tag }
    get box() { return this.boxv }
    set label(v) { this.tag = v }
    kind() { return "base" }
}
class S29Sub extends S29Base {
    constructor() { super(); this.own = 1 }
    readGet() { return super.label }
    readComputed(k) { return super[k]() }
    writeSetter() { super.label = "set"; return this.tag }
    writePlain() { super.fresh = 5; return this.fresh }
    compound() { super.n = 3; return super.n }
    viaPath() { this.boxv = { v: 0 }; super.box.v = 9; return this.boxv.v }
    destruct() { [super.d] = [11]; return this.d }
    destructObj() { ({ q: super.e } = { q: 12 }); return this.e }
    kind() { return "sub<" + super.kind() + ">" }
}
function s29() {
    var s = new S29Sub()
    check("sup1", s.readGet() === "B:base")
    check("sup2", s.readComputed("kind") === "base")
    check("sup3", s.writeSetter() === "set")
    check("sup4", s.writePlain() === 5)
    // 'super.n = 3' stores on the RECEIVER; 'super.n' then reads the class chain, which
    // has no n at all - so the read is undefined, exactly as in node.
    check("sup5", s.compound() === undefined)
    check("sup6", s.viaPath() === 9)
    check("sup7", s.destruct() === 11)
    check("sup8", s.destructObj() === 12)
    check("sup9", s.kind() === "sub<base>")
}

// ===== SECTION 30: the general new forms =====
class S30C { constructor(x) { this.x = x === undefined ? -1 : x } }
function S30F() { this.p = 9 }
var s30ns = { inner: { C: S30C } }
var s30arr = [S30F]
function s30mk() { return S30F }
function s30() {
    check("new1", new s30ns.inner.C(3).x === 3)
    check("new2", new S30C().x === -1)
    check("new3", new S30F().p === 9)
    check("new4", new (s30mk())().p === 9)
    check("new5", new s30arr[0]().p === 9)
    check("new6", (new (class { constructor() { this.z = 8 } })()).z === 8)
    check("new7", new S30C(2).x + new s30ns.inner.C(5).x === 7)
    check("new8", new s30ns.inner.C(4) instanceof S30C)
}

// ===== SECTION 31: a class heritage EXPRESSION =====
class S31Base { constructor() { this.b = 1 } hi() { return "base" } }
function s31mixin(B) { return class extends B { hi() { return "mix+" + super.hi() } } }
class S31M extends s31mixin(S31Base) { hi() { return "M+" + super.hi() } }
var s31holder = { K: S31Base }
class S31N extends s31holder.K { }
function s31() {
    var m = new S31M()
    check("her1", m.hi() === "M+mix+base")
    check("her2", m.b === 1)
    var n = new S31N()
    check("her3", n.b === 1)
    check("her4", n.hi() === "base")
    var Q = class extends (S31Base) { hi() { return "q" } }
    check("her5", new Q().hi() === "q")
    check("her6", new Q().b === 1)
    check("her7", n instanceof S31Base)
}

// ===== SECTION 32: a member path as the for-of target =====
function s32() {
    var o = { a: 0, b: 0 }
    var out = ""
    for (o.a of [1, 2, 3]) { out = out + o.a }
    check("fom1", out === "123")
    var arr = [0, 0]
    for (arr[1] of ["x", "y"]) { out = out + arr[1] }
    check("fom2", out === "123xy")
    var p = { k: 0 }
    for ([p.k] of [[7], [8]]) { out = out + p.k }
    check("fom3", out === "123xy78")
    function box() { return o }
    for (box().b of [5, 6]) { out = out + o.b }
    check("fom4", out === "123xy7856")
    var nest = { m: { n: 0 } }
    for (nest.m.n of [4]) { check("fom5", nest.m.n === 4) }
}

// ===== SECTION 33: a destructuring catch parameter =====
function s33() {
    var r = ""
    try { throw [1, 2, 3] } catch ([a, ...b]) { r = a + ":" + b.length + ":" + b[1] }
    check("cat1", r === "1:2:3")
    try { throw { m: "msg", c: 5 } } catch ({ m, c: code }) { r = m + code }
    check("cat2", r === "msg5")
    try { throw [] } catch ([x = 9]) { r = x }
    check("cat3", r === 9)
    try { throw { a: { b: 7 } } } catch ({ a: { b } }) { r = b }
    check("cat4", r === 7)
    try { throw "plain" } catch (e) { r = e }
    check("cat5", r === "plain")
    try { throw "bare" } catch { r = "nobind" }
    check("cat6", r === "nobind")
}

// ===== SECTION 34: export, and the debugger no-op =====
export const S34K = 7
export function s34f(x) { return x * 2 }
export class S34C { constructor() { this.v = 3 } }
var s34side = 0
export { s34side as s34sideOut }
export { S34K as s34alias }
export default class S34D { m() { return "d" } }
export function s34mark() { s34side = 42; return s34side }

function s34() {
    check("exp1", S34K === 7)
    check("exp2", s34f(4) === 8)
    check("exp3", new S34C().v === 3)
    check("exp4", s34mark() === 42)
    check("exp5", new S34D().m() === "d")
    var n = 0
    debugger
    n = n + 1
    debugger;
    check("dbg1", n === 1)
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
    s21() // SECTION-CALL 21
    s22() // SECTION-CALL 22
    s23() // SECTION-CALL 23
    s24() // SECTION-CALL 24
    s25() // SECTION-CALL 25
    s26() // SECTION-CALL 26
    s27() // SECTION-CALL 27
    s28() // SECTION-CALL 28
    s29() // SECTION-CALL 29
    s30() // SECTION-CALL 30
    s31() // SECTION-CALL 31
    s32() // SECTION-CALL 32
    s33() // SECTION-CALL 33
    s34() // SECTION-CALL 34
    println("full: " + checks + " checks, " + failures + " failures")
    return failures
}
