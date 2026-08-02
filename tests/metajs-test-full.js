// Full-syntax test: MetaJS (the JavaScript subset metajs-interpreter.abnf and
// metajs-to-llvm-ir.abnf implement, and the language every tag script in this
// repository is written in).
//
// This file belongs to the SECOND test group (./test.sh --full): it is NOT part
// of the default matrix. It walks the whole MetaJS syntax, one self-contained
// SECTION per language area. The --full runner runs the file, and whenever a
// grammar aborts it removes the section around the error and retries - so the
// report lists every unsupported section, not just the first.
//
// It has a THIRD reader: tests/clang-check.sh builds it into a NATIVE BINARY
// (metajs-to-llvm-ir.abnf carries exePath, so clang-check runs it that way) and
// requires the binary to exit 0. That makes this file the gate of phase 3 of
// docs/runtime-rework-plan.md: the same source has to answer identically under
// the Go runtime (abnf/jsrt.go, via llvm.Run) and under the C floor
// (languages/lib/runtime.c, compiled by c-to-llvm-ir.abnf and linked by clang).
//
// Conventions (shared by every *-test-full.* file):
//   - prologue (before the first SECTION marker): the check helper only
//   - each section: '// ===== SECTION <nn>: <name> =====', top-level,
//     self-contained, no references to other sections
//   - main() calls each section via a line tagged 'SECTION-CALL <nn>'
//     and prints the summary line 'full: <checks> checks, <failures> failures'
//   - main() returns the failure count (exit 0 == full support, verified)
//
// The HOST GLOBALS asserted here are the intersection of the two halves, which
// is smaller than either: metajs-interpreter.abnf binds exactly println, print,
// printf, sprintf, eprintln, parseInt, parseFloat, Math, String, exit and
// anytype, while the compiler half (abnf/jsrt.go standardJSBindings) also has
// Infinity, NaN, Array, Object, byteLen, sprint and rawSet. That difference is a
// real one and is recorded in docs/runtime-rework-plan.md rather than asserted
// here, because a check on it would fail on one half by construction.
//
// Deliberately out of scope, because MetaJS does not have them (they belong to
// tests/js-test-full.js): classes, arrow functions, template literals,
// destructuring, spread/rest, for-of/for-in, generators, async, regular
// expression literals, getters/setters, Symbol, BigInt, labelled statements,
// optional chaining, exponentiation.
//
// The last bit of an INEXACT floating point result IS asserted, in section 22.
// It used to be out of scope here: the C floor computes with the soft float
// c-to-llvm-ir.abnf emits, which truncated where IEEE-754 rounds to nearest
// even, so 0.1 + 0.2 came out one ulp below the Go runtime's answer. That is
// exactly the kind of divergence ./test.sh cannot see - both of its engines are
// the Go runtime - so it is pinned here instead, where tests/clang-check.sh runs
// this file as a NATIVE BINARY and the C floor has to give the same answer.
//
// Hand-written for the metacompiler project (Apache-2.0, no copied test-suite
// code).

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
// Condensed re-assertion of the basics every other section builds on.
function s01() {
    var n = 0
    for (var i = 0; i < 3; i++) { n = n + i }
    check("bas1", n === 3)
    var o = { a: 1, "b": 2 }
    o.c = o.a + o["b"]
    check("bas2", o.c === 3)
    var arr = [1, 2, 3]
    check("bas3", arr.length === 3 && arr[2] === 3)
    var t = 0
    try { throw "boom" } catch (e) { t = e === "boom" ? 1 : 2 } finally { t = t + 10 }
    check("bas4", t === 11)
    check("bas5", typeof check === "function")
}

// ===== SECTION 02: numeric literals and arithmetic =====
function s02() {
    check("num1", 0xff === 255)
    check("num2", 0XFF === 255)
    check("num3", 2 + 3 * 4 === 14)
    check("num4", (2 + 3) * 4 === 20)
    check("num5", 20 - 5 - 3 === 12)
    check("num6", 7 / 2 === 3.5)
    check("num7", 7 % 3 === 1)
    check("num8", -7 % 3 === -1)
    check("num9", 0.5 * 4 === 2)
    check("num10", -(-5) === 5)
    check("num11", 1000000 * 1000000 === 1000000000000)
    check("num12", 9007199254740992 + 0 === 9007199254740992)
    check("num13", 100.125 === 100.125 && 100.125 * 8 === 801)
    check("num14", 1 / 0 > 0 && -1 / 0 < 0 && 1 / (1 / 0) === 0)
    check("num15", 0 / 0 !== 0 / 0)
    check("num16", "" + 42 === "42" && "" + 2.5 === "2.5")
    check("num17", "" + (1 / 0) === "Infinity" && "" + (0 / 0) === "NaN")
    check("num18", "" + 1000000000000000000000 === "1e+21")
    check("num19", "" + 0.0000001 === "1e-7")
    check("num20", "" + -0.25 === "-0.25")
}

// ===== SECTION 03: assignment forms =====
function s03() {
    var ca = 5
    ca += 3;  check("asg1", ca === 8)
    ca -= 2;  check("asg2", ca === 6)
    ca *= 4;  check("asg3", ca === 24)
    ca /= 6;  check("asg4", ca === 4)
    ca %= 3;  check("asg5", ca === 1)
    var cs = "hi"
    cs += "!"
    check("asg6", cs === "hi!")
    var q
    check("asg7", q === undefined)
    check("asg8", (q = 5) + 1 === 6)
    var aa = 0
    var bb = 0
    aa = bb = 3
    check("asg9", aa + bb === 6)
    var m1 = 1, m2 = "two"
    check("asg10", m1 + m2 === "1two")
    var pn = 5
    check("asg11", pn++ === 5)
    check("asg12", ++pn === 7)
    check("asg13", pn-- === 7)
    check("asg14", --pn === 5)
    var o = { n: 1 }
    o.n += 4
    check("asg15", o.n === 5)
    var ar = [1, 2]
    ar[0] += 9
    check("asg16", ar[0] === 10)
}

// ===== SECTION 04: bitwise and shifts =====
function s04() {
    check("bit1", (255 & 300) === 44)
    check("bit2", (6 | 3) === 7)
    check("bit3", (5 ^ 1) === 4)
    check("bit4", ~5 === -6)
    check("bit5", (1 << 4) === 16)
    check("bit6", (-8 >> 1) === -4)
    check("bit7", (-8 >>> 28) === 15)
    check("bit8", (1 | 2 & 3) === 3)
    check("bit9", ((7 / 2) | 0) === 3)
    check("bit10", ((-7 / 2) | 0) === -3)
    check("bit11", ("12" | 0) === 12)
    check("bit12", (1 << 31) === -2147483648)
    check("bit13", (2147483647 + 1 | 0) === -2147483648)
    check("bit14", (0xffffffff | 0) === -1)
    check("bit15", (5 << 33) === 10)
}

// ===== SECTION 05: comparison and equality =====
function s05() {
    check("cmp1", 1 < 2 && 2 <= 2 && 3 > 2 && 3 >= 3)
    check("cmp2", "a" < "b" && "abc" < "abd")
    check("cmp3", 1 == "1" && !(1 === "1"))
    check("cmp4", null == undefined && !(null === undefined))
    check("cmp5", 0 == false && !(0 === false))
    check("cmp6", "" == 0)
    check("cmp7", 0 / 0 !== 0 / 0)
    check("cmp8", !(0 / 0 < 1) && !(0 / 0 > 1) && !(0 / 0 <= 1))
    var o1 = { a: 1 }
    var o2 = { a: 1 }
    check("cmp9", o1 !== o2 && o1 === o1)
    var a1 = [1]
    check("cmp10", a1 === a1 && a1 !== [1])
    check("cmp11", true === true && false !== true)
    check("cmp12", "2" > "10" && 2 < 10)
}

// ===== SECTION 06: logical operators and truthiness =====
function s06() {
    check("log1", (1 && 2) === 2)
    check("log2", (0 && 2) === 0)
    check("log3", (0 || 7) === 7)
    check("log4", (5 || 7) === 5)
    check("log5", !0 === true && !1 === false)
    check("log6", !"" === true && !"x" === false)
    check("log7", !undefined === true && !null === true)
    check("log8", !(0 / 0) === true)
    check("log9", ![] === false && !{} === false)
    var hits = 0
    function bump() { hits = hits + 1; return true }
    var ignored = false || bump()
    check("log10", hits === 1 && ignored === true)
    ignored = true || bump()
    check("log11", hits === 1)
    check("log12", (1 ? "y" : "n") === "y" && (0 ? "y" : "n") === "n")
}

// ===== SECTION 07: strings =====
function s07() {
    var s = "hello world"
    check("str1", s.length === 11)
    check("str2", s.charAt(0) === "h" && s[1] === "e")
    check("str3", s.charCodeAt(0) === 104)
    check("str4", s.indexOf("world") === 6 && s.indexOf("zz") === -1)
    check("str5", s.substring(0, 5) === "hello")
    check("str6", s.slice(-5) === "world")
    check("str7", s.toUpperCase() === "HELLO WORLD")
    check("str8", "ABC".toLowerCase() === "abc")
    // The case mapping is the full Unicode SIMPLE mapping, not ASCII: the C floor
    // used to change only the 26 letters, which is 2,321 code points on which the two
    // engines answered differently. Latin-1, Latin Extended-A (whose pairs alternate),
    // Greek, Cyrillic and an astral block, one from each shape of the table.
    // (not the sharp s: its uppercase is the two letters SS, which is a FULL mapping,
    // and the two halves of MetaJS disagree about it - pre-existing drift, and the
    // one thing in this area that is not settled)
    check("str7a", "\u00e9\u00fc\u00e0".toUpperCase() === "\u00c9\u00dc\u00c0")
    check("str7b", "\u0101\u0103\u0105".toUpperCase() === "\u0100\u0102\u0104")
    check("str7c", "\u03b1\u03b2\u03c9".toUpperCase() === "\u0391\u0392\u03a9")
    check("str7d", "\u0430\u0431\u044f".toUpperCase() === "\u0410\u0411\u042f")
    check("str7e", "\u00c9\u0100\u0391\u0410".toLowerCase() === "\u00e9\u0101\u03b1\u0430")
    check("str7f", String.fromCharCode(55297, 56320).toUpperCase() ===
                   String.fromCharCode(55297, 56320))
    check("str7g", String.fromCharCode(55297, 56360).toUpperCase() ===
                   String.fromCharCode(55297, 56320))
    check("str7h", "\u00b5".toUpperCase() === "\u039c" && "\u0131".toUpperCase() === "I")
    check("str9", "  pad  ".trim() === "pad")
    check("str10", s.replace("world", "there") === "hello there")
    check("str11", "a,b,c".split(",").length === 3)
    check("str12", "a,b,c".split(",").join("-") === "a-b-c")
    check("str13", "abc".split("").length === 3)
    check("str14", String.fromCharCode(65, 66) === "AB")
    check("str15", ("x" + 1 + 2) === "x12" && (1 + 2 + "x") === "3x")
    check("str16", "abc".slice(1, 2) === "b")
    check("str17", "abc".substring(2, 0) === "ab")
    check("str18", "".length === 0 && "".indexOf("") === 0)
    check("str19", "tab\tnl\nq\"e\\z" .length === 12)
    check("str20", 'single' === "single")
}

// ===== SECTION 08: UTF-16 string semantics =====
// .length and the index accessors count UTF-16 code units, not bytes or runes.
function s08() {
    check("u1", "héllo".length === 5)
    check("u2", "é".charCodeAt(0) === 233)
    check("u3", "héllo".charAt(1) === "é")
    check("u4", "héllo".substring(0, 2) === "hé")
    check("u5", String.fromCharCode(233) === "é")
    check("u6", "🎉".length === 2)
    check("u7", "🎉".charCodeAt(0) === 55356 && "🎉".charCodeAt(1) === 57225)
    check("u8", "🎉".substring(0, 1) + "🎉".substring(1, 2) === "🎉")
    check("u9", "🎉"[0] + "🎉"[1] === "🎉")
    check("u10", String.fromCharCode(55356) + String.fromCharCode(57225) === "🎉")
    check("u11", String.fromCharCode(55356, 57225) === "🎉")
    check("u12", "🎉".split("").join("") === "🎉")
    check("u13", "🎉".charAt(0).charCodeAt(0) === 65533)
    check("u14", ("a" + "🎉" + "é") === "a🎉é")
    check("u15", "a🎉é".length === 4)
}

// ===== SECTION 09: arrays =====
function s09() {
    var a = [1, 2, 3]
    check("arr1", a.length === 3)
    a.push(4)
    check("arr2", a.length === 4 && a[3] === 4)
    check("arr3", a.pop() === 4 && a.length === 3)
    check("arr4", a.shift() === 1 && a[0] === 2)
    a.unshift(0)
    check("arr5", a[0] === 0 && a.length === 3)
    check("arr6", a.indexOf(3) === 2 && a.indexOf(99) === -1)
    check("arr7", a.slice(1).length === 2)
    check("arr8", a.slice(1, 2)[0] === 2)
    check("arr9", a.concat([9]).length === 4)
    check("arr10", a.join("|") === "0|2|3")
    a.reverse()
    check("arr11", a[0] === 3 && a[2] === 0)
    var e = []
    check("arr12", e.length === 0 && e.pop() === undefined)
    var g = [1]
    g[3] = 4
    check("arr13", g.length === 4 && g[2] === undefined)
    g.length = 2
    check("arr14", g.length === 2 && g[1] === undefined)
    check("arr15", [1, [2, 3]][1][0] === 2)
    check("arr16", "" + [1, 2, 3] === "1,2,3")
}

// ===== SECTION 10: objects =====
function s10() {
    var o = { a: 1, b: "two", c: [3], d: { e: 4 } }
    check("obj1", o.a === 1 && o.b === "two")
    check("obj2", o.c[0] === 3 && o.d.e === 4)
    check("obj3", o["a"] === 1)
    o.f = 5
    check("obj4", o.f === 5)
    o["g"] = 6
    check("obj5", o.g === 6)
    check("obj6", o.missing === undefined)
    var key = "a"
    check("obj7", o[key] === 1)
    var empty = {}
    check("obj8", typeof empty === "object")
    check("obj9", "" + o.d === "[object Object]")
    var counter = { n: 0, bump: function () { return 1 } }
    counter.n = counter.n + counter.bump()
    check("obj10", counter.n === 1)
}

// ===== SECTION 11: functions and closures =====
function s11() {
    function add(x, y) { return x + y }
    check("fn1", add(2, 3) === 5)
    check("fn2", add(2) !== add(2))          // undefined + 2 is NaN
    var mul = function (x, y) { return x * y }
    check("fn3", mul(3, 4) === 12)
    function counter() {
        var n = 0
        return function () { n = n + 1; return n }
    }
    var c1 = counter()
    var c2 = counter()
    check("fn4", c1() === 1 && c1() === 2 && c2() === 1)
    function outer(x) {
        function inner(y) { return x + y }
        return inner(10)
    }
    check("fn5", outer(5) === 15)
    function fact(n) { if (n < 2) { return 1 } return n * fact(n - 1) }
    check("fn6", fact(5) === 120)
    function noReturn() { var z = 1 }
    check("fn7", noReturn() === undefined)
    check("fn8", add.call(null, 4, 5) === 9)
    check("fn9", add.apply(null, [6, 7]) === 13)
    check("fn10", typeof add === "function")
    var arr = [function () { return 1 }, function () { return 2 }]
    check("fn11", arr[0]() + arr[1]() === 3)
    function argCount() { return arguments.length }
    check("fn12", argCount(1, 2, 3) === 3)
}

// ===== SECTION 12: control flow =====
function s12() {
    var out = ""
    var i = 0
    while (i < 3) { out = out + i; i = i + 1 }
    check("ctl1", out === "012")
    out = ""
    i = 0
    do { out = out + i; i = i + 1 } while (i < 2)
    check("ctl2", out === "01")
    out = ""
    for (i = 0; i < 4; i++) {
        if (i === 1) { continue }
        if (i === 3) { break }
        out = out + i
    }
    check("ctl3", out === "02")
    var n = 0
    for (;;) { n = n + 1; if (n > 2) { break } }
    check("ctl4", n === 3)
    var big = 0
    if (big > 1) { out = "a" } else if (big === 0) { out = "b" } else { out = "c" }
    check("ctl5", out === "b")
    var never = 0
    while (false) { never = 1 }
    check("ctl6", never === 0)
    var nested = ""
    for (i = 0; i < 2; i++) {
        var j = 0
        while (j < 2) { nested = nested + i + j; j = j + 1 }
    }
    check("ctl7", nested === "00011011")
}

// ===== SECTION 13: switch =====
function s13() {
    function classify(v) {
        var out = ""
        switch (v) {
        case 1:
            out = "one"
            break
        case 2:
        case 3:
            out = "few"
            break
        default:
            out = "many"
        }
        return out
    }
    check("sw1", classify(1) === "one")
    check("sw2", classify(2) === "few" && classify(3) === "few")
    check("sw3", classify(9) === "many")
    function fall(v) {
        var n = 0
        switch (v) {
        case 1:
            n = n + 1
        case 2:
            n = n + 10
            break
        case 4:
            n = n + 100
        }
        return n
    }
    check("sw4", fall(1) === 11)
    check("sw5", fall(2) === 10)
    check("sw6", fall(4) === 100)
    check("sw7", fall(7) === 0)
    function strict(v) { switch (v) { case "1": return "s" } return "n" }
    check("sw8", strict("1") === "s" && strict(1) === "n")
}

// ===== SECTION 14: exceptions =====
function s14() {
    var t = anytype        // it holds a number, a string and an object in turn
    try { t = 1 } catch (e) { t = 2 }
    check("exc1", t === 1)
    try { throw "x" } catch (e) { t = e }
    check("exc2", t === "x")
    try { throw { code: 7 } } catch (e) { t = e.code }
    check("exc3", t === 7)
    var order = ""
    try { order = order + "t"; throw 1 } catch (e) { order = order + "c" } finally { order = order + "f" }
    check("exc4", order === "tcf")
    order = ""
    try { order = order + "t" } finally { order = order + "f" }
    check("exc5", order === "tf")
    function deep(n) { if (n === 0) { throw "bottom" } return deep(n - 1) }
    try { deep(4) } catch (e) { t = e }
    check("exc6", t === "bottom")
    function relabel() {
        try { try { throw "inner" } catch (e) { throw "re:" + e } } catch (e2) { return e2 }
        return "no"
    }
    check("exc7", relabel() === "re:inner")
    function retFromTry() { try { return 1 } finally { } }
    check("exc8", retFromTry() === 1)
    function retFromFinally() { try { return 1 } finally { return 2 } }
    check("exc9", retFromFinally() === 2)
    function breakOutOfTry() {
        var seen = 0
        var i = 0
        while (i < 5) { try { if (i === 2) { break } seen = seen + 1 } finally { } i = i + 1 }
        return seen
    }
    check("exc10", breakOutOfTry() === 2)
    function swallow() { try { throw "e" } finally { return "f" } }
    check("exc11", swallow() === "f")
    var noBinding = 0
    try { throw "y" } catch (e) { noBinding = 1 }
    check("exc12", noBinding === 1)
}

// ===== SECTION 15: typeof and the type classes =====
function s15() {
    check("typ1", typeof 1 === "number")
    check("typ2", typeof "s" === "string")
    check("typ3", typeof true === "boolean")
    check("typ4", typeof undefined === "undefined")
    check("typ5", typeof null === "object")
    check("typ6", typeof {} === "object")
    check("typ7", typeof [] === "object")
    check("typ8", typeof s15 === "function")
    check("typ9", typeof (0 / 0) === "number")
    check("typ10", typeof Math === "object")
    check("typ11", typeof Math.floor === "function")
}

// ===== SECTION 16: the MetaJS discipline (declaration and type pinning) =====
// The legal side of the rules: a declaration before every use, a variable that
// keeps the class of its first non-null value, undefined/null assignable
// always, and `var v = anytype` as the declared exemption.
function s16() {
    var n = 1
    n = 2
    check("dis1", n === 2)
    var s = "a"
    s = "b"
    check("dis2", s === "b")
    var u
    u = 5
    check("dis3", u === 5)
    var withNull = null
    withNull = "now a string"
    check("dis4", withNull === "now a string")
    var free = anytype
    check("dis5", free === undefined)
    free = 1
    check("dis6", free === 1)
    free = "text"
    check("dis7", free === "text")
    free = [1]
    check("dis8", free.length === 1)
    var o = { m: 1 }
    o.m = "members stay free"
    check("dis9", o.m === "members stay free")
    var back = 3
    back = undefined
    check("dis10", back === undefined)
    back = 4
    check("dis11", back === 4)
}

// ===== SECTION 17: the host globals =====
function s17() {
    check("host1", Math.floor(3.7) === 3 && Math.floor(-3.2) === -4)
    check("host2", Math.abs(-5) === 5 && Math.abs(5) === 5)
    check("host3", Math.max(1, 9, 3) === 9 && Math.min(1, 9, 3) === 1)
    check("host4", Math.max() === -1 / 0 && Math.min() === 1 / 0)
    check("host5", Math.imul(3, 4) === 12)
    check("host6", Math.imul(65536, 65536) === 0)
    check("host7", Math.ceil(1.2) === 2 && Math.trunc(-1.7) === -1)
    check("host8", Math.round(2.5) === 3 && Math.round(-2.5) === -2)
    check("host9", Math.sign(-3) === -1 && Math.sign(0) === 0)
    check("host10", Math.pow(2, 10) === 1024)
    check("host11", Math.sqrt(16) === 4)
    check("host12", parseInt("42") === 42 && parseInt("ff", 16) === 255)
    check("host13", parseInt("0x1A") === 26)
    check("host14", parseFloat("3.25xyz") === 3.25)
    check("host15", parseInt("nope") !== parseInt("nope"))
    check("host16", sprintf("x=%d", 7) === "x=7")
    check("host17", sprintf("%s=%d", "n", 5) === "n=5")
    check("host18", sprintf("%c%c", 79, 75) === "OK")
    check("host19", sprintf("%s|%d|%s", "a", 1, "b") === "a|1|b")
    check("host20", printf !== undefined && eprintln !== undefined && exit !== undefined)
}

// ===== SECTION 18: number to string, the exact cases =====
// Every value here is exactly representable, so the answer is a fact about the
// FORMATTER, not about the arithmetic.
function s18() {
    check("fmt1", "" + 0 === "0")
    check("fmt2", "" + -1 === "-1")
    check("fmt3", "" + 0.5 === "0.5")
    check("fmt4", "" + 0.25 === "0.25")
    check("fmt5", "" + 1.5 === "1.5")
    check("fmt6", "" + 123456789 === "123456789")
    check("fmt7", "" + 4503599627370496 === "4503599627370496")
    check("fmt8", "" + 0.001 === "0.001")
    check("fmt9", "" + 0.000001 === "0.000001")
    check("fmt10", "" + 100000000000000000000 === "100000000000000000000")
    check("fmt11", "" + true === "true" && "" + false === "false")
    check("fmt12", "" + null === "null" && "" + undefined === "undefined")
    check("fmt13", "" + [] === "" && "" + [1] === "1")
    check("fmt14", "" + [null, 1] === ",1")
    check("fmt15", "" + 125.0 === "125")
}

// ===== SECTION 19: scope and hoisting-free declaration order =====
function s19() {
    var outer = 1
    function reads() { return outer }
    check("sco1", reads() === 1)
    outer = 2
    check("sco2", reads() === 2)
    var shadowed = "outer"
    function shadows() {
        var shadowed = "inner"
        return shadowed
    }
    check("sco3", shadows() === "inner" && shadowed === "outer")
    var acc = 0
    function makeAdder(by) { return function (x) { return x + by } }
    var add3 = makeAdder(3)
    var add4 = makeAdder(4)
    check("sco4", add3(1) === 4 && add4(1) === 5)
    var i = 0
    var fns = []
    while (i < 3) { fns.push(makeAdder(i)); i = i + 1 }
    acc = fns[0](0) + fns[1](0) + fns[2](0)
    check("sco5", acc === 3)
    var blockVar = 1
    if (true) { blockVar = 2 }
    check("sco6", blockVar === 2)
}

// ===== SECTION 20: comments, semicolon insertion, empty statements =====
function s20() {
    // a line comment
    /* a block
       comment */
    var a = 1 // trailing
    var b = 2 /* inline */ + 3
    ;
    ;;
    check("lex1", a === 1 && b === 5)
    var noSemi = 7
    var alsoNoSemi = 8
    check("lex2", noSemi + alsoNoSemi === 15)
    var multi = 1 +
        2 +
        3
    check("lex3", multi === 6)
    var str = "// not a comment"
    check("lex4", str.length === 16)
}

// ===== SECTION 21: a whole small program =====
// Nothing new syntactically; it exercises the pieces together the way real
// MetaJS code (a tag script) does.
function s21() {
    function Node(name) { return { name: name, kids: [] } }
    function addKid(parent, kid) { parent.kids.push(kid); return parent }
    function render(node, depth) {
        var pad = ""
        var i = 0
        while (i < depth) { pad = pad + "  "; i = i + 1 }
        var out = pad + node.name + "\n"
        i = 0
        while (i < node.kids.length) { out = out + render(node.kids[i], depth + 1); i = i + 1 }
        return out
    }
    var root = Node("root")
    addKid(root, Node("a"))
    addKid(addKid(root, Node("b")), Node("ignored"))
    var text = render(root, 0)
    check("prog1", text.indexOf("root") === 0)
    check("prog2", text.split("\n").length === 5)

    function sortNums(list) {
        var i = 0
        while (i < list.length) {
            var j = i + 1
            while (j < list.length) {
                if (list[j] < list[i]) {
                    var t = list[i]
                    list[i] = list[j]
                    list[j] = t
                }
                j = j + 1
            }
            i = i + 1
        }
        return list
    }
    check("prog3", sortNums([3, 1, 2]).join(",") === "1,2,3")

    function countWords(s) {
        var counts = {}
        var words = s.split(" ")
        var i = 0
        while (i < words.length) {
            var w = words[i]
            if (counts[w] === undefined) { counts[w] = 1 } else { counts[w] = counts[w] + 1 }
            i = i + 1
        }
        return counts
    }
    var c = countWords("a b a c a")
    check("prog4", c.a === 3 && c.b === 1 && c.c === 1)

    function fib(n) {
        var a = 0
        var b = 1
        var i = 0
        while (i < n) { var t = a + b; a = b; b = t; i = i + 1 }
        return a
    }
    check("prog5", fib(10) === 55)
}

// ===== SECTION 22: INEXACT floating point, to the last bit =====
// The point of this section is the one divergence ./test.sh is structurally
// blind to. Its two engines - goja and -frozen - are both the Go runtime, so a
// wrong answer in the C floor cannot make them disagree. tests/clang-check.sh
// builds this file as a native binary and compares its output, and that is the
// only place these assertions bite.
//
// Every value here is INEXACT: it needs correct round-to-nearest-ties-to-even in
// the emitted soft float (add, subtract, multiply, divide, and the conversion of
// an integer above 2^53), correct rounding in Math.sqrt, and an exact shortest
// decimal in the formatter. Under the truncating soft float the C floor shipped
// with until 2026-08-02, 0.1 + 0.2 printed 0.29999999999999991 and 10 / 3
// printed 3.333333333333333. The oracle is the Go runtime, which is JS.
function s22() {
    check("ieee1", 0.1 + 0.2 === 0.30000000000000004)
    check("ieee2", "" + (0.1 + 0.2) === "0.30000000000000004")
    check("ieee3", 0.1 + 0.2 > 0.3)
    check("ieee4", "" + (10 / 3) === "3.3333333333333335")
    check("ieee5", "" + (1 / 3) === "0.3333333333333333")
    check("ieee6", "" + (2 / 3) === "0.6666666666666666")
    check("ieee7", "" + (100 / 7) === "14.285714285714286")
    check("ieee8", "" + (0.1 * 0.2) === "0.020000000000000004")
    check("ieee9", "" + (0.1 + 0.7) === "0.7999999999999999")
    check("ieee10", "" + (0.1 - 0.3) === "-0.19999999999999998")
    check("ieee11", "" + (0.3 / 0.1) === "2.9999999999999996")
    check("ieee12", "" + (1.0000000000000002 * 1.0000000000000002) === "1.0000000000000004")
    // sqrt of a non-square, and the product that shows its last bit is right
    check("ieee13", "" + Math.sqrt(2) === "1.4142135623730951")
    check("ieee14", "" + Math.sqrt(3) === "1.7320508075688772")
    check("ieee15", "" + Math.sqrt(5) === "2.23606797749979")
    check("ieee16", "" + (Math.sqrt(2) * Math.sqrt(2)) === "2.0000000000000004")
    // an accumulation: the error has to land in the same place both times
    var s = 0
    for (var i = 0; i < 10; i++) { s = s + 0.1 }
    check("ieee17", "" + s === "0.9999999999999999")
    var t = 0
    for (var j = 0; j < 100; j++) { t = t + 0.01 }
    check("ieee18", "" + t === "1.0000000000000007")
    // above 2^53 the integer-to-double conversion has to round, not drop bits
    check("ieee19", "" + (123456789 * 987654321) === "121932631112635260")
    check("ieee20", "" + 9007199254740993 === "9007199254740992")
    check("ieee21", "" + (10000000000000000 + 1) === "10000000000000000")
    check("ieee22", "" + (10000000000000000 + 3) === "10000000000000004")
    // exact cases stay exact
    check("ieee23", 1 / 1048576 * 1048576 === 1)
    check("ieee24", 4 / 3 * 3 === 4)
    // Math.pow with a NON-INTEGER exponent. It used to answer NaN outright: repeated
    // multiplication cannot reach one, and an approximation would have disagreed with
    // the Go twin in the last bits. The C floor now carries a faithful port of Go's
    // own math.Pow (and the math.Exp, math.Log, math.Frexp, math.Ldexp and math.Modf
    // underneath it), which gives the same bits because the arithmetic under it is
    // now correctly rounded.
    check("pow1", "" + Math.pow(2, 0.5) === "1.4142135623730951")
    check("pow2", "" + Math.pow(2, 1.5) === "2.82842712474619")
    check("pow3", "" + Math.pow(10, 1 / 3) === "2.154434690031884")
    check("pow4", "" + Math.pow(0.5, -2.25) === "4.756828460010884")
    check("pow5", "" + Math.pow(7.25, 3.5) === "1026.0842537594017")
    check("pow6", "" + Math.pow(2, -0.5) === "0.7071067811865475")
    check("pow7", "" + Math.pow(1.0000001, 1000000) === "1.1051709126081526")
    check("pow8", Math.pow(3, 3) === 27 && Math.pow(2, 10) === 1024)
    check("pow9", Math.pow(2, -3) === 0.125 && Math.pow(0, 0) === 1)
    check("pow10", Math.pow(-2, 3) === -8 && "" + Math.pow(-2, 0.5) === "NaN")
    // and the decimal PARSE of a literal no fast path can reach
    check("ieee25", "" + parseFloat("1e100") === "1e+100")
    check("ieee26", "" + parseFloat("1.7976931348623157e308") === "1.7976931348623157e+308")
    check("ieee27", "" + parseFloat("0.9999999999999999") === "0.9999999999999999")
    check("ieee28", "" + parseFloat("123456789123456789") === "123456789123456780")
}

// ===== SECTION 23: value INTERNING and the small-integer cache =====
//
// languages/lib/runtime.c shares cells: one string cell per string-literal
// ADDRESS (str_intern), one number cell per small integer (mk_num/js_num_i), and
// a scope that starts with no buffers at all. Phase 4a of
// docs/runtime-rework-plan.md did that to stop the arena growing 18 KB per loop
// iteration, and the whole argument for its being safe is that a shared cell is
// INDISTINGUISHABLE from a fresh one. This section is that claim, written down.
// ./test.sh cannot see it - both of its engines are the Go runtime - so it is
// here, where tests/clang-check.sh runs the file as a native binary.
function s23() {
    // literals, computed strings and the coercions that used to allocate
    var a = "abc"
    var b = "abc"
    var c = "ab" + "c"
    check("intern01", a === b)
    check("intern02", a === c)
    check("intern03", a < "abd")
    check("intern04", typeof a === "string" && typeof 1 === "number")
    check("intern05", typeof true === "boolean" && typeof null === "object")
    check("intern06", typeof undefined === "undefined")
    check("intern07", "" + null === "null" && "" + true === "true")
    check("intern08", "" + undefined === "undefined" && "" + false === "false")
    // every integer across and beyond the cached window round-trips
    var bad = -1
    var i = 0
    while (i <= 1030) {
        var n = i * 1
        if (n !== i) { bad = i; i = 1031 }
        i = i + 1
    }
    check("intern09", bad === -1)
    // (spelled as a STRING: the interpreter half does not bind Infinity - see the
    // host-globals note in the header)
    check("intern10", 0 === 0 && -0 === 0 && "" + (1 / -0) === "-Infinity")
    check("intern11", "" + -0 === "0" && "" + (0 - 0.0) === "0")
    check("intern12", 1024 + 1 === 1025 && 1025 - 1 === 1024)
    check("intern13", -256 + 0 === -256 && -257 + 0 === -257)
    check("intern14", "" + (0.1 + 0.2) === "0.30000000000000004")
    check("intern15", 10 % 3 === 1 && "" + (10 % 0.1) === "0.09999999999999945")
    // the method-name compare that no longer builds a string per candidate
    var arr = [3, 1, 2]
    check("intern16", arr.length === 3 && arr.indexOf(2) === 2)
    check("intern17", arr.join("-") === "3-1-2" && arr.slice(1).join(",") === "1,2")
    check("intern18", arr.concat([9]).join(",") === "3,1,2,9")
    check("intern19", "hello".charAt(1) === "e" && "hello".charCodeAt(0) === 104)
    check("intern20", "hello".slice(1, 3) === "el" && "hello".substring(0, 2) === "he")
    check("intern21", "HeLLo".toUpperCase() === "HELLO" && "  x ".trim() === "x")
    check("intern22", "hello".replace("l", "L") === "heLlo")
    // a method name used as an ORDINARY key, and a key that is a PREFIX of one
    var o = {}
    o["length"] = 5
    o.push = 1
    check("intern23", o.length === 5 && o.push === 1)
    check("intern24", arr["pu"] === undefined && "hello"["cha"] === undefined)
    check("intern25", [].length === 0 && "".length === 0)
    // blocks that declare nothing (an empty scope), and captured scopes
    function mk(k) {
        var q = 0
        { { q = k * 2 } }
        return function () { return q }
    }
    check("intern26", mk(21)() === 42 && mk(3)() === 6)
    var fs = []
    var j = 0
    while (j < 5) {
        (function (v) { fs.push(function () { return v * 1 }) })(j)
        j = j + 1
    }
    var t = 0
    j = 0
    while (j < 5) { t = t + fs[j](); j = j + 1 }
    check("intern27", t === 10)
    // a NUL inside a string, against the C-literal compare in str_eq_c
    var z = "ab" + String.fromCharCode(0) + "cd"
    check("intern28", z.length === 5 && z !== "ab" && z.indexOf("cd") === 3)
}

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
    println("full: " + checks + " checks, " + failures + " failures")
    return failures
}
