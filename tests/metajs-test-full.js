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
// The HOST GLOBALS used to be asserted only at the INTERSECTION of the two
// halves, which was smaller than either: metajs-interpreter.abnf bound eleven
// names while the compiler half (abnf/jsrt.go standardJSBindings) also had
// Infinity, NaN, Array, Object, byteLen, sprint and rawSet - and no check on the
// difference was possible, because it would fail on one half by construction.
//
// THAT IS CLOSED (docs/todo.md 4.1). All three engines now bind the SAME 48
// names - the C floor's seed_root list in languages/lib/runtime.c is the
// authority, the interpreter gained the six it was missing, and the two that only
// the grammar's own tag scripts ever wanted (Object, rawSet) left the program set.
// SECTION 31 asserts every one of the 48 on both halves, which is only possible
// because they agree; abnf/hostglobals_test.go parses the three sources and fails
// `go test ./abnf/` if a future name lands in fewer than three of them.
//
// SECTION 32 is the other half of that item: string CASE MAPPING. All four
// engines now use the SIMPLE per-rune mapping (Go's strings.ToUpper), so
// "ss".toUpperCase() keeps one character. goja used to give real JavaScript's
// full mapping ("SS") - a live goja-vs--frozen divergence in this half that no
// gate could reach - and installGojaCaseMapping in abnf/commonscript.go now
// routes it through the same Go function the other three engines call. The
// direction was forced: full case mapping in the C floor is `case_map`, measured
// at 135x and reverted.
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
    // ToInt32 OUTSIDE the int64 range. It is a MODULO, not a range test, and
    // all three engines used to get it wrong in three different ways: the C
    // floor answered 0 from an explicit `d_in_long` test, abnf/jsrt.go's
    // rt.toInt32 ended in `int32(int64(f))` - implementation defined in Go, -1
    // here on arm64 - and goja's own operator does the same int64 conversion.
    // Every value below is checked against node; see docs/runtime-next-plan.md
    // part 2. Written as decimal literals because MetaJS has no exponent form.
    check("bit16", (10000000000000000000 | 0) === -1981284352)
    check("bit17", (100000000000000000000 | 0) === 1661992960)
    check("bit18", ((0 - 10000000000000000000) | 0) === 1981284352)
    check("bit19", ((0 - 100000000000000000000) | 0) === -1661992960)
    // A value whose low 32 bits are genuinely zero, so 0 is the RIGHT answer -
    // the assertion that says the fix did not just replace one constant with
    // another. 2^63 exactly, and a value far above the 2^84 cutoff.
    check("bit20", (9223372036854775808 | 0) === 0 &&
        (1000000000000000019884624838656 | 0) === 0)
    // The int64 boundary from below is unchanged, which is what says the fast
    // path still runs: 2^63 - 1024 is the largest double under 2^63.
    check("bit21", (9223372036854774784 | 0) === -1024 && (4294967295 | 0) === -1)
    // The other five operators take the same reading.
    check("bit22", (10000000000000000000 & 4294967295) === -1981284352 &&
        (10000000000000000000 >> 4) === -123830272 &&
        (10000000000000000000 >>> 28) === 8)
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

    // ----- the try-barrier POOL, which coroutines made per-stack -------------
    //
    // languages/lib/runtime.c used to hold one global `long JB[512]` of jmp_bufs
    // indexed by JB_DEPTH. A generator body runs on its OWN C stack, and a
    // setjmp is only valid on the stack that took it, so that one pool became a
    // pointer (JBP) to the pool of whichever stack is RUNNING - main's, or the
    // coroutine's - swapped by the side that gains control, exactly as
    // GC_STACK_BASE is. The assertions below are what that refactor has to keep
    // true, and they run in all three engines.
    //
    // WHERE THEY DISCRIMINATE, stated like SECTION 25's: the pool only exists in
    // the NATIVE binary. In the other two engines a throw is a Go panic or a
    // host-engine exception, so these are ordinary correctness checks there.
    // Measured against a floor whose pool is indexed wrongly, the deep nest
    // below is the one that fails first.
    function nest(n) {
        if (n === 0) { throw "floor" }
        try { return nest(n - 1) } catch (e) { throw e }
        return "unreached"
    }
    var deepCaught = ""
    try { nest(200) } catch (e) { deepCaught = e }
    check("exc13", deepCaught === "floor")
    // A SECOND deep nest: the depth has to have been restored, or the pool is
    // now indexed past whatever the first one left behind.
    deepCaught = ""
    try { nest(200) } catch (e) { deepCaught = e }
    check("exc14", deepCaught === "floor")
    // Two sequential try statements at the SAME depth share one jmp_buf buffer
    // and must each setjmp it again - the reuse the pool was introduced for.
    var reuse = ""
    var ri = 0
    while (ri < 50) {
        try { throw "r" + ri } catch (e) { reuse = e } finally { reuse = reuse + "!" }
        ri = ri + 1
    }
    check("exc15", reuse === "r49!")
    // A thrown value that is reachable ONLY through the throw machinery while a
    // finally clause allocates heavily. In the native binary the pool is scanned
    // as a root range (gc_scan_jb) for exactly this; run that binary under
    // MEC_GC=stress to make it a collector assertion rather than a plain one.
    function throwAcrossGarbage() {
        var junk = []
        try {
            throw { tag: "carried", n: 41 + 1 }
        } finally {
            var gi = 0
            while (gi < 3000) { junk = [gi, gi + 1, "pad"]; gi = gi + 1 }
        }
        return null
    }
    var carried = anytype
    try { carried = throwAcrossGarbage() } catch (e) { carried = e }
    check("exc16", carried.tag === "carried" && carried.n === 42)
    // The same, one frame further out and with the catch clause allocating too:
    // the value has to survive a collection triggered INSIDE the handler.
    var handled = 0
    try {
        throw { n: 7 }
    } catch (e) {
        var gj = 0
        var pad = []
        while (gj < 3000) { pad = [gj]; gj = gj + 1 }
        handled = e.n
    }
    check("exc17", handled === 7)
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

// ===== SECTION 24: sized integers (the sint* host globals) =====
// The one primitive type MetaJS itself does not have. It is tag 13 of
// languages/lib/runtime.c natively, jsGInt of abnf/jsrtint.go under llvm.Run,
// and the {__sz} pair box of lib/interp-core.js in the interpreter half - three
// implementations of one specification, reached from a MetaJS source program
// through the eleven sint* names, which is what makes this section able to pin
// them at all. See docs/runtime-next-plan.md part 2.
//
// The invariant every assertion below rests on: an integer that a double holds
// exactly IS a plain number, and only the rest is boxed. That is why `typeof` is
// "number" throughout - a version that always boxed would still pass sintStr and
// fail sint01/sint03.
function s24() {
    // ----- the invariant, and typeof across the 2^53 boundary -----
    check("sint01", sint(0, 5, 64, 0) === 5 && sintIs(sint(0, 5, 64, 0)) === false)
    check("sint02", typeof sint(0, 5, 64, 0) === "number")
    var exact = sint(2097152, 0, 64, 0)             // 2^53, the largest exact one
    check("sint03", exact === 9007199254740992 && sintIs(exact) === false)
    var over = sint(2097152, 1, 64, 0)              // 2^53 + 1 is not
    check("sint04", sintIs(over) === true)
    check("sint05", typeof over === "number")       // NOT "object" - the whole point
    check("sint06", sintStr(over) === "9007199254740993")
    // A box therefore pins the type class "number", so the variable may go on to
    // hold an ordinary one. This is the assertion that fails if typeof does not.
    var pinned = over
    pinned = 7
    check("sint07", pinned === 7)
    // The missing arguments default to 64 bits, signed.
    check("sint08", sint(0, 7) === 7 && sintNum(sint(0, 255, 8)) === -1)
    check("sint09", sintWidth(5) === 64 && sintUns(5) === false)
    check("sint10", sintWidth(over) === 64 && sintUns(over) === false)

    // ----- arithmetic at and beyond the 64 bit boundary -----
    var mx = sint(2147483647, 4294967295, 64, 0)    // int64 max
    check("sint11", sintStr(mx) === "9223372036854775807" && sintIs(mx) === true)
    var mn = sintOp(0, mx, 1)                       // wraps to int64 min
    check("sint12", sintStr(mn) === "-9223372036854775808")
    check("sint13", sintStr(sintOp(1, mn, 1)) === "9223372036854775807")
    check("sint14", sintCmp(mx, mn) === 1 && sintCmp(mn, mx) === -1 && sintCmp(mx, mx) === 0)
    var mn2 = sint(2147483648, 0, 64, 0)
    check("sint15", sintStr(mn2) === "-9223372036854775808" && mn2 === mn)
    // int64min / -1 has no answer in range; the hardware wrap keeps it, and the
    // remainder is 0 (si_apply and giArith both say so explicitly).
    check("sint16", sintStr(sintOp(3, mn2, -1)) === "-9223372036854775808")
    check("sint17", sintOp(4, mn2, -1) === 0)
    check("sint18", sintOp(1, over, 1) === 9007199254740992 && sintIs(sintOp(1, over, 1)) === false)
    check("sint19", sintStr(sintOp(0, over, 1)) === "9007199254740994")
    check("sint20", sintOp(0, exact, 1) === over)
    // The PLAIN operators are not sized-integer operators: they read the box
    // through to_number and compute in doubles, which is where the low bit goes.
    check("sint21", over - 1 === 9007199254740991)
    check("sint22", "" + over === "9007199254740993")
    // A box compares by VALUE under ===, and under == it is neither a number nor
    // a string, so the loose comparison is false. Both halves are odd here in
    // exactly the same way.
    var b8 = sintConv(5, 8, 0)
    check("sint23", (b8 === 5) === true && (b8 == 5) === false)
    check("sint24", b8 < 6 && b8 > 4 && b8 <= 5 && b8 >= 5)

    // ----- signedness: the 64 bit unsigned reading -----
    var um = sint(4294967295, 4294967295, 64, 1)
    check("sint25", sintStr(um) === "18446744073709551615")
    check("sint26", sintWidth(um) === 64 && sintUns(um) === true)
    check("sint27", sintCmp(um, 0) === 1)           // unsigned, so ABOVE zero
    check("sint28", sintStr(sintConv(um, 64, 0)) === "-1" && sintCmp(sintConv(um, 64, 0), 0) === -1)
    check("sint29", sintNum(um) > 0 && um > 0)
    check("sint30", sintOp(0, um, 1) === 0 && sintIs(sintOp(0, um, 1)) === true)
    check("sint31", sintStr(sintOp(0, um, 1)) === "0")

    // ----- width, and the wrap every operator does on overflow -----
    var i8 = sint(0, 127, 8, 0)
    check("sint32", sintWidth(i8) === 8 && sintUns(i8) === false && sintIs(i8) === true)
    check("sint33", sintNum(sintOp(0, i8, 1)) === -128 && sintStr(sintOp(0, i8, 1)) === "-128")
    check("sint34", sintNum(sintOp(0, sint(0, 255, 8, 1), 1)) === 0)
    check("sint35", sintNum(sintOp(1, sint(0, 0, 8, 1), 1)) === 255)
    check("sint36", sintNum(sintOp(0, sint(0, 32767, 16, 0), 1)) === -32768)
    check("sint37", sintNum(sintOp(1, sint(0, 0, 32, 1), 1)) === 4294967295)
    check("sint38", sintNum(sintOp(0, sint(0, 2147483647, 32, 0), 1)) === -2147483648)

    // ----- conversion between widths and signednesses -----
    check("sint39", sintNum(sintConv(300, 8, 0)) === 44)
    check("sint40", sintNum(sintConv(-1, 8, 1)) === 255 && sintNum(sintConv(-1, 16, 1)) === 65535)
    check("sint41", sintNum(sintConv(-1, 32, 1)) === 4294967295)
    check("sint42", sintStr(sintConv(-1, 64, 1)) === "18446744073709551615")
    check("sint43", sintConv(-1, 64, 0) === -1 && sintIs(sintConv(-1, 64, 0)) === false)
    check("sint44", sintNum(sintConv(i8, 32, 0)) === 127 && sintWidth(sintConv(i8, 32, 0)) === 32)
    check("sint45", sintNum(sintConv(sintConv(200, 8, 1), 8, 0)) === -56)
    check("sint46", sintStr(sintConv(mx, 64, 1)) === "9223372036854775807")

    // ----- shifts: a count at or above the width is defined, unlike in C -----
    var sh = sintOp(9, 1, 62)
    check("sint47", sintStr(sh) === "4611686018427387904" && sintIs(sh) === true)
    check("sint48", sintOp(10, sh, 62) === 1)
    check("sint49", sintNum(sintOp(9, sint(0, 3, 8, 0), 8)) === 0)
    check("sint50", sintNum(sintOp(10, sint(0, 240, 8, 0), 100)) === -1)   // signed: clamps to w-1
    check("sint51", sintNum(sintOp(10, sint(0, 240, 8, 1), 100)) === 0)    // unsigned: everything out
    check("sint52", sintNum(sintOp(10, sint(0, 240, 8, 0), 1)) === -8)
    check("sint53", sintNum(sintOp(10, sint(0, 240, 8, 1), 1)) === 120)

    // ----- the bitwise operators, and division's truncation toward zero -----
    check("sint54", sintOp(5, 12, 10) === 8 && sintOp(6, 12, 10) === 14)
    check("sint55", sintOp(7, 12, 10) === 6 && sintOp(8, 12, 10) === 4)
    check("sint56", sintOp(3, -7, 2) === -3 && sintOp(4, -7, 2) === -1)
    check("sint57", sintNum(sintOp(3, sint(0, 200, 8, 1), 3)) === 66)
    check("sint58", sintNum(sintOp(3, sint(0, 200, 8, 0), 3)) === -18)
    check("sint59", sintNum(sintOp(4, sint(0, 200, 8, 0), 3)) === -2)

    // ----- the two halves back out, and the text of a non-box -----
    check("sint60", sintHi(mx) === 2147483647 && sintLo(mx) === 4294967295)
    check("sint61", sintHi(5) === 0 && sintLo(5) === 5)
    check("sint62", sintHi(-1) === 4294967295 && sintLo(-1) === 4294967295)
    check("sint63", sintStr(sint(sintHi(um), sintLo(um), 64, 1)) === "18446744073709551615")
    check("sint64", sintStr(42) === "42" && sintStr("x") === "x" && sintNum(3.5) === 3.5)
    // a compound assignment reaches the same operator path
    var acc = 0
    acc += b8
    check("sint65", acc === 5)

    // ----- the four rules that only a 64 BIT operand can see -----
    // Measured: with an 8 bit operand each of these passes whatever the
    // implementation does, because the truncation to the width hides it. They
    // are the assertions that discriminate, so they are spelled at 64 bits.
    // A shift count at or above 64: the clamp to w-1 and the "everything out"
    // rule are real here, where the pair arithmetic would otherwise take the
    // count modulo 64 and shift by zero.
    check("sint66", sintOp(10, mx, 64) === 0 && sintOp(10, mn, 64) === -1)
    check("sint67", sintStr(sintOp(9, mx, 63)) === "-9223372036854775808" && sintOp(9, mx, 64) === 0)
    // Unsigned division and an unsigned right shift, where the raw 64 bit value
    // is NEGATIVE read as a signed one - the only place the two paths differ.
    check("sint68", sintStr(sintOp(3, um, 2)) === "9223372036854775807")
    check("sint69", sintStr(sintOp(4, um, 10)) === "5")
    check("sint70", sintStr(sintOp(10, um, 1)) === "9223372036854775807" && sintOp(10, um, 63) === 1)
    // The LEFT operand decides the result type where both are boxes, so this one
    // pair compares unsigned one way round and signed the other - and answers 1
    // both times, for two different reasons.
    check("sint71", sintCmp(um, mx) === 1 && sintCmp(mx, um) === 1)

    // ----- sintRaw: the one door past si_norm -----
    // si_norm UNBOXES a signed 64 bit value a double holds exactly, and that is
    // exactly what a statically typed language's `long` must not do: Java's
    // `1000000L * 1000000L` is 1000000000000 where `1000000 * 1000000` is
    // -727379968. Every other sint* builtin goes through si_norm, so until this
    // existed layer 2 had NO constructor for a small boxed value, and java
    // worked around it by marking the box UNSIGNED - the one shape si_norm will
    // not unbox - and re-supplying the six operations whose signed reading the
    // floor then got wrong, in ~90 lines. See docs/runtime-next-plan.md part 3.
    var r5 = sintRaw(0, 5, 64, 0)
    check("sint72", sintIs(r5) === true && sintIs(sint(0, 5, 64, 0)) === false)
    check("sint73", typeof r5 === "number" && sintStr(r5) === "5" && sintNum(r5) === 5)
    // SIGNED and small, the shape sint() cannot build at all. The six
    // operations java had to re-supply are asserted on a raw SIGNED box below,
    // and every one of them is the plain sint* builtin.
    check("sint74", sintWidth(r5) === 64 && sintUns(r5) === false)
    var rn1 = sintRaw(4294967295, 4294967295, 64, 0)        // -1, as a signed box
    check("sint75", sintStr(rn1) === "-1" && sintNum(rn1) === -1)
    check("sint76", sintOp(3, rn1, 2) === 0 && sintStr(sintOp(4, rn1, 2)) === "-1")
    check("sint77", sintStr(sintOp(10, rn1, 1)) === "-1")
    check("sint78", sintCmp(rn1, r5) === -1 && sintCmp(r5, rn1) === 1)
    // si_trunc IS still applied, so the cell keeps the invariant its own header
    // states: 200 at 8 bits signed is -56, not 200 wearing an int8 label.
    check("sint79", sintNum(sintRaw(0, 200, 8, 0)) === -56 && sintWidth(sintRaw(0, 200, 8, 0)) === 8)
    check("sint80", sintNum(sintRaw(0, 200, 8, 1)) === 200 && sintUns(sintRaw(0, 200, 8, 1)) === true)
    // The defaults are sint()'s: 64 bits, signed.
    check("sint81", sintWidth(sintRaw(0, 7)) === 64 && sintUns(sintRaw(0, 7)) === false &&
        sintIs(sintRaw(0, 7)) === true)
    check("sint82", sintHi(r5) === 0 && sintLo(r5) === 5)
    check("sint83", sintStr(sintRaw(2147483647, 4294967295, 64, 0)) === "9223372036854775807")
    // A box, so === compares by value and == is false - the same faithfully odd
    // pair sint23 pins, now reachable at a value a double holds exactly.
    check("sint84", (r5 === 5) === true && (r5 == 5) === false)
    var rmin = sintRaw(2147483648, 0, 64, 0)
    check("sint85", sintStr(rmin) === "-9223372036854775808" && sintIs(rmin) === true)
    check("sint86", sintStr(sintOp(3, rmin, -1)) === "-9223372036854775808" && sintOp(4, rmin, -1) === 0)
    // sintConv still normalises: sintRaw is the ONLY door past si_norm, and a
    // raw box converted back to signed 64 bits is a plain number again.
    check("sint87", sintConv(r5, 64, 0) === 5 && sintIs(sintConv(r5, 64, 0)) === false)
    // Measured: the two rules above have to be asserted where the width is WIDE
    // enough for the truncation to change the value, and at 64 bits for the
    // signedness - the same rule sint66-sint71 record. At 8 bits an unsigned
    // box's payload is never negative, so "unsigned ignored" hides; at 64 bits
    // it is the whole difference between -1 and 18446744073709551615.
    check("sint88", sintNum(sintRaw(0, 4294967295, 32, 0)) === -1 &&
        sintNum(sintRaw(0, 4294967295, 32, 1)) === 4294967295 &&
        sintWidth(sintRaw(0, 4294967295, 32, 0)) === 32)
    var ru = sintRaw(4294967295, 4294967295, 64, 1)
    check("sint89", sintStr(ru) === "18446744073709551615" && sintUns(ru) === true &&
        sintCmp(ru, 0) === 1 && sintNum(ru) > 0)
}

// ===== SECTION 25: the garbage collector (mark/sweep over the heap) =====
// languages/lib/runtime.c allocated from a bump arena that was never freed until
// docs/runtime-next-plan.md part 1b replaced it with a mark/sweep heap. The worst
// failure mode a collector has is freeing something that is still LIVE, and the
// assertions below are the shapes that would show it: data held across a call
// that allocates heavily, a closure that keeps the scope defining it, a value
// thrown across twenty frames, and - the one the C stack scan exists for - an
// argument temporary that is live in a machine register while a LATER argument
// runs a few thousand allocations.
//
// WHERE THEY DISCRIMINATE. Only in the NATIVE binary: the other two engines are
// garbage collected by Go and by the host JS engine, so there they are ordinary
// correctness checks and cannot fail for a collector reason. The measured
// discriminating power against a deliberately broken collector is in part 1b of
// docs/runtime-next-plan.md; run the native binary with MEC_GC=stress (collect at
// every allocation) or MEC_GC=poison (and never reuse a swept block) to make a
// rare use-after-free a deterministic one.
function s25() {
    // Enough churn to cross the collector's threshold in the native binary, and
    // small enough that the interpreter half still runs this file quickly.
    var churn = 2000
    function gcPad(seed, n) {
        var s = "" + seed
        var i = 0
        while (i < n) { s = "" + seed + i; i = i + 1 }
        return s
    }
    function gcAdder(x) {
        var y = x * 2
        return function (z) { return x + y + z }
    }
    function gcDeep(n, tag) {
        if (n === 0) { throw {t: tag, s: "boom" + tag} }
        var pad = "p" + n
        return gcDeep(n - 1, tag) + pad
    }
    function gcLen3(a, b, c) { return a.length + b.length + c.length }
    var i = 0

    // ----- live data reachable from a scope survives heavy churn -----
    var keep = []
    i = 0
    while (i < 120) { keep.push({id: i, s: "v" + i}); i = i + 1 }
    gcPad("k", churn)
    var ok = 0
    i = 0
    while (i < 120) { if (keep[i].id === i && keep[i].s === "v" + i) { ok = ok + 1 } i = i + 1 }
    check("gc01", ok === 120)

    // ----- a closure keeps the scope that defines it (the cycle refcounting
    // could not free, and the reason mark/sweep was chosen) -----
    var fs = []
    i = 0
    while (i < 50) { fs.push(gcAdder(i)); i = i + 1 }
    gcPad("c", churn)
    var sum = 0
    i = 0
    while (i < 50) { sum = sum + fs[i](1); i = i + 1 }
    check("gc02", sum === 3725)                      // 3i+1 summed over 0..49

    // ----- a value thrown across twenty frames, while every frame allocates -----
    var caught = 0
    i = 0
    while (i < 40) {
        try { gcDeep(20, i) } catch (e) { if (e.t === i && e.s === "boom" + i) { caught = caught + 1 } }
        i = i + 1
    }
    check("gc03", caught === 40)

    // ----- THE C-STACK ROOT TEST. The handle of the first argument is live in a
    // machine register or a spill slot - in no scope and in no object - while the
    // second and third arguments allocate. So is the argument array itself. -----
    var tail = "" + (churn - 1)                      // what gcPad's last round appends
    check("gc04", gcLen3(gcPad("a", churn), gcPad("bb", churn), gcPad("ccc", churn))
                  === 6 + 3 * tail.length)

    // ----- a buffer reallocated many times over: every doubling leaves the old
    // block garbage while the new one is live -----
    var big = []
    i = 0
    while (i < 500) { big.push("e" + i); i = i + 1 }
    gcPad("b", churn)
    check("gc05", big.length === 500 && big[0] === "e0" && big[499] === "e499")

    // ----- the same for an object's key and value buffers -----
    var o = {}
    i = 0
    while (i < 60) { o["k" + i] = "w" + i; i = i + 1 }
    gcPad("o", churn)
    var ok2 = 0
    i = 0
    while (i < 60) { if (o["k" + i] === "w" + i) { ok2 = ok2 + 1 } i = i + 1 }
    check("gc06", ok2 === 60)

    // ----- a chain of scopes two deep, reached only through two closures -----
    function gcNest(v) {
        var a = v + 0
        var g = function () { return a }
        return function () { return g() }
    }
    var f = gcNest(7)
    gcPad("n", churn)
    check("gc07", f() === 7)

    // ----- WHY gc_grow IS NOT ASSERTED HERE, measured rather than assumed -----
    // gc_grow doubles the mark stack when the marking frontier outgrows it, and
    // the Part B coverage instrumentation (docs/runtime-next-plan.md) found it
    // at ZERO calls across all 34 native corpus builds. Every shape above has a
    // frontier of a handful of pairs, because gc_drain pops as it goes; the one
    // shape that fills the stack is a single array whose slots are all distinct
    // traced blocks, which drain pushes in one go. GC_MCAP starts at 65536 longs
    // and the stack holds PAIRS, so the frontier has to pass 32,768 - measured
    // natively, 30,000 elements never call gc_grow and 33,000 always do.
    //
    // That assertion cannot live in this file. 33,000+ object allocations inside
    // main() cost more than the -frozen half's 100,000,000-instruction safety
    // valve allows (the metajs INTERPRETER is itself a MetaJS program running on
    // abnf/jsrt.go's IR interpreter there), so the ratchet reports
    // "FULL under goja, BUT -frozen fails or differs" - tried at 40,000, and it
    // is the step limit and not the collector that stops it. There is no cheaper
    // shape: the frontier IS the count of live traced blocks, so any spelling
    // costs the same 33,000 allocations. gc_grow is therefore proved reachable
    // by a native probe recorded in docs/runtime-next-plan.md and left unasserted
    // here on purpose, rather than left looking untested by accident.
}

// ===== SECTION 26: boxed doubles (the flo* host globals) =====
// The SECOND primitive type MetaJS itself does not have, and the one ten of the
// eleven remaining languages need: tag 14 of languages/lib/runtime.c natively,
// jsJFlo of abnf/jsrtjvm.go under llvm.Run, and the {__fl} box of
// metajs-interpreter.abnf in the interpreter half. See docs/runtime-next-plan.md
// part 2.
//
// It exists because a statically typed language has to tell `1.0 / 3.0` from
// `1 / 3`, which are the SAME two operands when every number is one double.
// The invariant is the mirror image of the sized integer's: a plain number is an
// INTEGRAL type and a box is a double, and unlike si_norm a double NEVER
// unboxes - flo(1) stays a box, or 1.0 and 1 would stop being distinguishable.
//
// Two things this section deliberately does NOT assert, because they differ by
// construction rather than by defect:
//   - TRUTHINESS. runtime.c's truthy() answers `!zero && !nan` for tag 14, so
//     `if (flo(0))` is false there and true in the interpreter half, where the
//     condition is the host engine's own `if`. The sized integer has the same
//     hole for the same reason.
//   - THE SIGN OF A ZERO PRODUCT. goja represents an integral number as an
//     int64, so `0 * -2718281` is +0 under the goja interpreter and -0 in real
//     JS, in the frozen engine and in the C floor. Measured: `1 / (0 * -2718281)`
//     is +Inf under goja and -Inf in the other three. A -0.0 that comes from a
//     SMALL product (flo00 below) is safe and is asserted.
function s26() {
    // ----- the box, and typeof -----
    check("flo01", floIs(flo(1, 0)) === true && floIs(1) === false)
    check("flo02", typeof flo(1, 0) === "number")     // NOT "object" - the point
    check("flo03", floNum(flo(1.5, 0)) === 1.5 && floNum(2.5) === 2.5)
    check("flo04", floStyle(flo(1, 1)) === 1 && floStyle(flo(1, 2)) === 2 && floStyle(7) === 0)
    // A double NEVER unboxes, which is where it parts company with sint():
    // sint(0, 5, 64, 0) IS the plain 5, and flo(5) is not.
    check("flo05", floIs(flo(5, 0)) === true)
    // So a box pins the type class "number" and the variable may hold a plain
    // one afterwards - the assertion that fails if typeof does not.
    var pin = flo(1, 0)
    pin = 7
    check("flo06", pin === 7)

    // ----- the three renderings (jvmFloText), which are the only thing the
    // statically typed languages differ in -----
    var e20 = 100000000000000000000
    check("flo07", floStr(flo(1, 0)) === "1.0")
    check("flo08", floStr(flo(1, 1)) === "1" && floStr(flo(1, 2)) === "1")
    check("flo09", floStr(flo(1.5, 0)) === "1.5" && floStr(flo(1.5, 1)) === "1.5")
    check("flo10", floStr(flo(e20, 0)) === "1.0E20")
    check("flo11", floStr(flo(e20, 1)) === "1e+20")
    check("flo12", floStr(flo(e20, 2)) === "1E+20")
    check("flo13", floStr(flo(1 / 0, 0)) === "Infinity" && floStr(flo(1 / 0, 1)) === "+Inf")
    check("flo14", floStr(flo(0 - 1 / 0, 1)) === "-Inf" && floStr(flo(0 - 1 / 0, 2)) === "-Infinity")
    check("flo15", floStr(flo(0 / 0, 0)) === "NaN" && floStr(flo(0 / 0, 1)) === "NaN")
    check("flo16", floStr(flo(0, 0)) === "0.0" && floStr(flo(0, 1)) === "0")
    check("flo00", floStr(flo(0 * (0 - 1), 0)) === "-0.0" && floStr(flo(0 * (0 - 1), 2)) === "-0")
    // The bounds of Java's plain-decimal window, 1e-3 <= |d| < 1e7. Asserted AT
    // the boundary in both directions, because a rendering that used the decimal
    // exponent instead of the value would still pass a test taken from the
    // middle of the range.
    check("flo17", floStr(flo(0.001, 0)) === "0.001" && floStr(flo(0.0001, 0)) === "1.0E-4")
    check("flo18", floStr(flo(9999999, 0)) === "9999999.0" && floStr(flo(10000000, 0)) === "1.0E7")
    // Go's window is the decimal exponent, -4 <= e < 6 - a different rule, so
    // 1e6 renders plain in Java and scientific in Go.
    check("flo19", floStr(flo(1000000, 1)) === "1e+06" && floStr(flo(100000, 1)) === "100000")
    check("flo20", floStr(flo(0.0001, 1)) === "0.0001" && floStr(flo(0.00001, 1)) === "1e-05")
    // C#'s is [1e-5, 1e15), and its exponent carries a sign and two digits.
    check("flo21", floStr(flo(0.00001, 2)) === "0.00001" && floStr(flo(0.000001, 2)) === "1E-06")
    check("flo22", floStr(flo(1000000000000000, 2)) === "1E+15")
    check("flo23", floStr(flo(999999999999999, 2)) === "999999999999999")
    // A non-box goes through the ordinary to_string.
    check("flo24", floStr(1.5) === "1.5" && floStr("x") === "x")

    // ----- THE REASON THE TYPE EXISTS: the same two operands, two answers -----
    check("flo25", floStr(floOp(3, flo(1, 0), 3)) === "0.3333333333333333")
    check("flo26", floOp(3, 1, 3) === 0 && floIs(floOp(3, 1, 3)) === false)
    check("flo27", floIs(floOp(3, flo(1, 0), 3)) === true)
    // Two integral operands keep the 32 bit wrap the statically typed compilers
    // emit; one double makes the whole operation floating point.
    check("flo28", floOp(2, 2.5, 1.5) === 3 && floStr(floOp(2, flo(2.5, 0), 1.5)) === "3.75")
    check("flo29", floStr(floOp(4, flo(7, 0), 2)) === "1.0")     // % is a float %
    check("flo30", floStr(floOp(1, flo(1, 0), 1)) === "0.0" && floStr(floOp(0, flo(1, 0), 1)) === "2.0")
    // jvmStyleOf: the style of whichever operand is a box, the LEFT one when
    // both are.
    check("flo31", floStyle(floOp(0, flo(1, 1), flo(2, 2))) === 1)
    check("flo32", floStyle(floOp(0, 1, flo(2, 2))) === 2)
    // An operator index the table does not have answers 0, boxed when an
    // operand is - jvmArith's `var x float64` with no arm taken.
    check("flo33", floStr(floOp(9, flo(1, 0), 2)) === "0.0")

    // ----- equality: jvmNumEq, which is strict_eq's two tag-14 arms -----
    check("flo34", floEq(flo(1, 0), 1) === true && floEq(1, flo(1, 0)) === true)
    check("flo35", floEq(flo(1, 0), flo(1, 2)) === true)         // the style is not part of the value
    check("flo36", floEq(flo(0 / 0, 0), flo(0 / 0, 0)) === false)
    check("flo37", floEq(flo(1, 0), "1") === false && floEq(flo(1, 0), true) === false)
    // A boxed double against a SIZED INTEGER is false: the floor's jf_num_eq has
    // no tag 13 arm and the twin's floEq host matches it. This is the PRIMITIVE's
    // contract, not a language's - Java's and C#'s `==` promotes the integral
    // operand to double first (JLS 15.21.1 / ECMA-334 12.12.9) and so answers
    // TRUE for `0L == 0.0`, which is why lib/runtime-jvm.metajs's rtjStrictEq
    // does that promotion BEFORE it calls floEq. Both compiler halves used to get
    // that wrong; see java-test-full's flt10e-g and csharp-test-full's flt11a-c.
    check("flo38", floEq(flo(1, 0), sint(0, 1, 8, 0)) === false)
    // === is the same two arms, reached through the operator; == is NOT, because
    // loose_eq sees a box as neither a number nor a string. Faithfully odd, and
    // the sized integer does exactly the same thing.
    check("flo39", (flo(1, 0) === 1) === true && (1 === flo(1, 0)) === true)
    check("flo40", (flo(1, 0) == 1) === false)
    check("flo41", (flo(1, 0) === flo(1, 0)) === true && (flo(1, 0) === 2) === false)

    // ----- the ORDINARY operators, which do not box: js_add_v answers a plain
    // number, and only floOp / the js_jf* externs of a compiler box a result ---
    check("flo42", flo(1.5, 0) + 1 === 2.5 && floIs(flo(1.5, 0) + 1) === false)
    check("flo43", flo(1.5, 0) * 2 === 3 && flo(3, 0) - flo(1, 0) === 2)
    check("flo44", flo(1.5, 0) < 2 && flo(1.5, 0) > 1 && flo(1.5, 0) <= 1.5)
    // A string operand concatenates through the box's own rendering.
    check("flo45", "x" + flo(1, 0) === "x1.0" && "x" + flo(1, 1) === "x1")

    // ----- Math.max / Math.min / Math.abs: the DOUBLE overload is selected as
    // soon as one side is a double, so Math.max(1.5, 2) is 2.0 -----
    check("flo46", floStr(floMax(flo(1.5, 0), 2)) === "2.0" && floIs(floMax(flo(1.5, 0), 2)) === true)
    check("flo47", floStr(floMin(flo(1.5, 0), 2)) === "1.5")
    check("flo48", floMax(1, 2) === 2 && floIs(floMax(1, 2)) === false)
    check("flo49", floStr(floAbs(flo(0 - 2.5, 0))) === "2.5" && floAbs(0 - 3) === 3)
    check("flo50", floIs(floAbs(flo(0 - 2.5, 0))) === true && floIs(floAbs(0 - 3)) === false)
    // floAbs of a SIGNED 64 bit box, which is the one arm of this primitive that
    // NO LAYER-2 FILE REACHES - java-rt.metajs and kotlin-rt.metajs both handle
    // their long case before the call - so it carried a to_int32 wrap in all
    // THREE engines for as long as it existed. These rows are the only thing that
    // reaches it, and they are here because a primitive with no caller is exactly
    // the kind that goes wrong quietly. sintRaw is the one door past si_norm, so
    // it is how a big signed box is built at all (see sint72 above).
    var absBig = sintRaw(0, 3000000000, 64, 0)               // 3e9, past int32
    var absNeg = sintRaw(4294967295, 1294967296, 64, 0)      // -3e9, as a signed box
    check("flo51", sintStr(floAbs(absNeg)) === "3000000000" && sintStr(floAbs(absBig)) === "3000000000")
    // Long.MIN_VALUE maps to ITSELF under two's-complement negation, which is the
    // answer JLS 15.15.4 asks for rather than an accident.
    var absMin = sintRaw(2147483648, 0, 64, 0)               // -9223372036854775808
    check("flo52", sintStr(floAbs(absMin)) === "-9223372036854775808")
    // An UNSIGNED box is a magnitude already and must come back untouched.
    var absU = sintRaw(4294967295, 4294967295, 64, 1)        // 2^64-1
    check("flo53", sintStr(floAbs(absU)) === "18446744073709551615")
}

// ===== SECTION 27: keysOf, and the sign of an underflowed power =====
// Two floor builtins that the eleven-language rollout needs and that no earlier
// section could reach.
//
// keysOf(o) is js_keys - an object's own keys in INSERTION order, skipping the
// internal __-prefixed slots. It is a builtin and not an extern because an
// extern is only callable from an emitter: MetaJS has neither for..in nor
// Object.keys, so a layer-2 runtime library could not walk an object at all,
// which blocks var_dump, `==` between objects, an (array) cast and clone in PHP
// and their equivalents in the other nine. Three implementations again: host id
// 61 of languages/lib/runtime.c, keysBindings of abnf/jsrtint.go, and - because
// the interpreter's script has no key enumeration under BOTH of its engines - a
// hidden insertion-order list kept by metajs-interpreter.abnf itself.
function s27() {
    var o = {b: 1, a: 2}
    o.c = 3
    o.b = 9                                       // a rewrite does NOT reorder
    var k = keysOf(o)
    check("keys01", k.length === 3)
    check("keys02", k[0] === "b" && k[1] === "a" && k[2] === "c")
    check("keys03", keysOf({}).length === 0)
    // The keys are usable as keys, which is the whole point for layer 2.
    var sum = 0
    var i = 0
    while (i < k.length) { sum = sum + o[k[i]]; i = i + 1 }
    check("keys04", sum === 14)                   // 9 + 2 + 3
    // A computed member write registers its key too.
    var c = {}
    c["x" + 1] = "v"
    check("keys05", keysOf(c).length === 1 && keysOf(c)[0] === "x1")
    // The internal slots are skipped, in every half.
    var d = {__class: 1, n: 2}
    check("keys06", keysOf(d).length === 1 && keysOf(d)[0] === "n")
    // A nested object keeps its own list.
    var n = {outer: {inner: 1, second: 2}}
    check("keys07", keysOf(n).length === 1 && keysOf(n.outer).length === 2)
    check("keys08", keysOf(n.outer)[1] === "second")
    // The list is a fresh array each call: mutating it does not edit the object.
    var f = keysOf(o)
    f.push("zz")
    check("keys09", keysOf(o).length === 3)
    // An object a function built, and one with many keys (the list grows).
    function mk(count) {
        var out = {}
        var j = 0
        while (j < count) { out["k" + j] = j; j = j + 1 }
        return out
    }
    var big = mk(40)
    var bk = keysOf(big)
    check("keys10", bk.length === 40 && bk[0] === "k0" && bk[39] === "k39")
    // ++ on a member counts as a write of that key.
    var p = {}
    p.n = 0
    p.n++
    check("keys11", keysOf(p).length === 1 && p.n === 1)

    // ----- the sign of a power that underflows to zero -----
    // Math.pow(-0.5, a large odd integer) is -0, not +0: d_ldexp has to put the
    // sign back on the underflowed zero. The native floor answered +0 until
    // 2026-08-03 while node and the Go twin answered -0 - and only
    // tests/clang-check.sh runs the native binary, so nothing else would see it.
    // Spelled as a division because -0 === 0 is true and MetaJS has no Infinity
    // global in both halves: 1 / -0 is -Infinity, which IS negative.
    check("pow01", 1 / Math.pow(0 - 0.5, 2147483647) < 0)
    check("pow02", 1 / Math.pow(0 - 0.5, 9007199254740991) < 0)
    check("pow03", 1 / Math.pow(0.5, 2147483647) > 0)      // the positive base stays +0
    check("pow04", Math.pow(0 - 0.5, 3) === 0 - 0.125)
    check("pow05", Math.pow(0 - 0.5, 0 - 2147483647) < 0)  // overflow keeps it too

    // ----- an INFINITE base, which is the only way into d_odd_int -----
    // The Part B coverage instrumentation (docs/runtime-next-plan.md) counted
    // every call to every body of the C floor across all 34 native corpus
    // builds and found d_odd_int at ZERO. It is reachable and nothing reached
    // it: d_pow only consults it when the base is a signed zero, and the one
    // spelling of a signed zero that needs no -0.0 literal and no Infinity
    // global is pow(-Inf, y) - d_pow answers that by recursing on 1.0/x, which
    // IS -0.0, with the exponent's sign flipped. So the four cases below walk
    // both arms of `d_sign(xb) && d_odd_int(y)` in both the y<0 and the y>0
    // branch, and an odd/even mistake in d_odd_int flips a sign in each.
    // Checked against node and against the native binary; the interpreter half
    // has neither Infinity nor -0.0 as literals, hence 1/0 and the divisions.
    var ninf = 0 - 1 / 0
    check("pow06", Math.pow(ninf, 3) === ninf)             // odd, y>0  -> -Inf
    check("pow07", Math.pow(ninf, 2) === 1 / 0)            // even, y>0 -> +Inf
    check("pow08", 1 / Math.pow(ninf, 0 - 3) < 0)          // odd, y<0  -> -0
    check("pow09", 1 / Math.pow(ninf, 0 - 2) > 0)          // even, y<0 -> +0
    // A NON-integral exponent is never odd, whichever side of zero it is.
    check("pow10", 1 / Math.pow(ninf, 0 - 2.5) > 0)
    // And the exponent above 2^53, where d_odd_int gives up and says "even".
    check("pow11", Math.pow(ninf, 9007199254740994) === 1 / 0)

    // ----- sprintf %v, which is the only way into fmt_val -----
    // Same finding, same file: fmt_top was reached by the whole corpus and
    // fmt_val was not, because every fmt_top the corpus takes gets a STRING and
    // returns at the tag-4 test one line in. %v on a non-string is the shortest
    // way past it, and sprintf is one of the eleven host globals BOTH halves
    // bind (see the header note on sprint, which only the compiler half has).
    check("fmtv01", sprintf("%v", 42) === "42")
    check("fmtv02", sprintf("%v", true) === "true")
    check("fmtv03", sprintf("%v", 1.5) === "1.5")
    check("fmtv04", sprintf("%v|%v", "hi", 7) === "hi|7")
    // An ARRAY operand is deliberately NOT asserted: the interpreter half's own
    // sprintf renders it as JavaScript does and the compiler half as Go's %v
    // does, so a check on it would fail on one half by construction - the same
    // rule the header states for the host globals. Go's [1 2 3] rendering is
    // pinned in tests/go-test-full.go, where it is the language's own answer.
    check("fmtv05", sprintf("%v", 0 - 0.125) === "-0.125")
}

// ===== SECTION 28: delKey (the floor's js_del under its MetaJS name) =====
// The other half of SECTION 27's argument. keysOf lets a layer-2 runtime WALK an
// object; delKey lets it PRUNE one. It is a builtin and not an extern for the same
// reason - an extern is only callable from an emitter - and MetaJS has no `delete`
// operator at all, so without it a runtime library can only BLANK a slot and keep a
// side table of what it blanked. JavaScript's layer 2 did exactly that, and it was
// quadratic (61 s for 4,000 delete-in-a-loop iterations, 0.05 s now), it leaked, and
// it could not repair key ORDER after a delete-then-reinsert.
//
// Three implementations again, and this section is what keeps them in step: host id
// 63 of languages/lib/runtime.c, keysBindings of abnf/jsrtint.go, and - because the
// interpreter's script has no `delete` either - the insertion-order list of
// metajs-interpreter.abnf, rewritten without the key.
function s28() {
    var o = {a: 1, b: 2, c: 3}
    check("del01", delKey(o, "b") === true)
    check("del02", o.b === undefined)
    var k = keysOf(o)
    check("del03", k.length === 2 && k[0] === "a" && k[1] === "c")
    // A key that comes BACK lands at the END of the order - it is a new key now.
    o.b = 9
    var k2 = keysOf(o)
    check("del04", k2.length === 3 && k2[0] === "a" && k2[1] === "c" && k2[2] === "b")
    check("del05", o.b === 9 && o.a === 1 && o.c === 3)
    // Deleting a key that was never there is a no-op that still answers true.
    check("del06", delKey(o, "zz") === true && keysOf(o).length === 3)
    // Anything that is not an object is a no-op, and so is an empty object.
    check("del07", delKey(5, "a") === true && delKey("s", "a") === true)
    check("del08", delKey({}, "a") === true)
    // Every key of an object, one at a time, first and last included.
    var p = {x: 1, y: 2, z: 3}
    delKey(p, "x")
    delKey(p, "z")
    check("del09", keysOf(p).length === 1 && keysOf(p)[0] === "y" && p.y === 2)
    delKey(p, "y")
    check("del10", keysOf(p).length === 0)
    // The internal __-prefixed slots are reachable by name even though keysOf hides
    // them: layer 2 stores its own bookkeeping there.
    var d = {__class: 1, n: 2}
    check("del11", delKey(d, "__class") === true && d.__class === undefined)
    check("del12", keysOf(d).length === 1 && keysOf(d)[0] === "n")
    // Many keys, deleted in a loop: this is the shape that was quadratic.
    var big = {}
    var i = 0
    while (i < 400) { big["k" + i] = i; delKey(big, "k" + i); i = i + 1 }
    check("del13", keysOf(big).length === 0)
    // Every other key of 400, so the surviving order is checked rather than assumed.
    var j = 0
    while (j < 400) { big["m" + j] = j; j = j + 1 }
    var m = 0
    while (m < 400) { delKey(big, "m" + m); m = m + 2 }
    var bk = keysOf(big)
    check("del14", bk.length === 200 && bk[0] === "m1" && bk[199] === "m399")
    check("del15", big.m1 === 1 && big.m0 === undefined && big.m399 === 399)
    // A computed key, and a numeric one (delKey takes the key as a string).
    var c = {}
    c["x" + 1] = "v"
    check("del16", delKey(c, "x1") === true && keysOf(c).length === 0)
    var n = {}
    n[7] = "seven"
    check("del17", delKey(n, "7") === true && keysOf(n).length === 0 && n[7] === undefined)
    // A nested object keeps its own list, and deleting the OUTER key does not
    // disturb the inner one.
    var nest = {outer: {inner: 1, second: 2}}
    var inner = nest.outer
    delKey(nest, "outer")
    check("del18", keysOf(nest).length === 0 && keysOf(inner).length === 2)
    check("del19", inner.inner === 1 && inner.second === 2)
    // The slot is really gone rather than blanked: an object whose only key was
    // deleted enumerates as empty, and re-adding does not duplicate the entry.
    var r = {only: 1}
    delKey(r, "only")
    r.only = 2
    check("del20", keysOf(r).length === 1 && keysOf(r)[0] === "only" && r.only === 2)
}

// ===== SECTION 29: the scope API (scopeNew/Parent/Get/Has/Decl) =====
// The third instance of SECTION 27's argument, and the largest. A SCOPE is the C
// floor's private storage: an emitter hands a layer-2 function a scope handle -
// js_pyset_var's `s`, and every `defined?` / `isset` / `global` / `nonlocal`
// probe - and layer 2 could do NOTHING with it, because MetaJS cannot open a cell
// and an extern is not callable from MetaJS source. Six languages answered that
// by lowering the probe into their EMITTER instead (docs/runtime-next-plan.md:
// swift, dart, go, ruby, kotlin's nine helpers, python's six), which is the same
// IR-building written out once per language for one language-neutral question.
// Three migration reports asked for this API by name.
//
// Three implementations, and this section is what keeps them in step: host ids
// 64..68 of languages/lib/runtime.c (the REAL tag-11 scope, so a layer-2 file
// gets the emitter's own chain), scopeBindings of abnf/jsrtint.go (*jsScope), and
// a box of its own in metajs-interpreter.abnf - whose own chain is an array of
// {v,t} objects with no parent link, so it has no scope to lend.
//
// The two asymmetries with the js_scope_* EXTERNS are asserted here, because they
// are what let the three halves agree at all: an absent parent is undefined and
// not the root scope (sco05/sco06 - a builtin's arguments are values, so the
// handle 0 that means "global" to an emitter never arrives, and a chain running
// into the host globals has no twin in the interpreter half), and scopeHas is the
// OWN-scope test with no chain walk (sco03/sco04), which is the one question
// js_scope_typeof cannot answer and the one python asked for by name.
function s29() {
    var root = scopeNew()
    scopeDecl(root, "x", 1)
    scopeDecl(root, "y", "hi")
    var inner = scopeNew(root)
    scopeDecl(inner, "x", 2)
    // scopeGet walks the CHAIN, and the innermost binding shadows.
    check("sco01", scopeGet(inner, "x") === 2)
    check("sco02", scopeGet(inner, "y") === "hi")
    // scopeHas does NOT walk it. This pair is the whole point of the name.
    check("sco03", scopeHas(inner, "x") === true && scopeHas(root, "x") === true)
    check("sco04", scopeHas(inner, "y") === false && scopeHas(root, "y") === true)
    check("sco05", scopeParent(inner) === root)
    check("sco06", scopeParent(root) === undefined)
    // scopeNew(null) and scopeNew(undefined) are the same "no parent".
    check("sco07", scopeParent(scopeNew(null)) === undefined)
    // scopeDecl OVERWRITES in this scope and never reaches the parent's binding -
    // it is scope_put, not js_scope_set.
    scopeDecl(inner, "x", 7)
    check("sco08", scopeGet(inner, "x") === 7 && scopeGet(root, "x") === 1)
    // A name declared only in the inner scope is invisible from the outer one,
    // which is what makes scopeHas+scopeParent a complete containment test.
    scopeDecl(inner, "z", 5)
    check("sco09", scopeHas(inner, "z") === true && scopeHas(root, "z") === false)
    check("sco10", scopeGet(inner, "z") === 5)
    check("sco11", scopeDecl(inner, "w", 0) === undefined && scopeGet(inner, "w") === 0)
    // Three levels: scopeParent in a loop IS the chain walk, and it reaches the
    // same binding scopeGet does. This is the loop a layer-2 containment probe
    // writes, so it is asserted rather than described.
    var deep = scopeNew(inner)
    var hops = 0
    var cur = deep
    while (cur !== undefined && !scopeHas(cur, "y")) { cur = scopeParent(cur); hops = hops + 1 }
    check("sco12", hops === 2 && cur === root && scopeGet(deep, "y") === "hi")
    // A slot really holding undefined is BOUND - the distinction js_scope_typeof
    // cannot make, and the reason scopeHas exists at all.
    scopeDecl(deep, "u", undefined)
    check("sco13", scopeHas(deep, "u") === true && scopeGet(deep, "u") === undefined)
    // Values of every class survive the round trip, keys are strings.
    scopeDecl(deep, "f", check)
    scopeDecl(deep, "o", {k: 3})
    check("sco14", typeof scopeGet(deep, "f") === "function" && scopeGet(deep, "o").k === 3)
    check("sco15", typeof root === "object" && root !== inner)
    // A scope is not an object to the program: it has no readable members, and
    // two distinct scopes never compare equal.
    check("sco16", scopeNew() !== scopeNew())
    // The chain is by REFERENCE: a binding added to the parent after the child
    // was made is visible from the child.
    scopeDecl(root, "late", 42)
    check("sco17", scopeGet(deep, "late") === 42 && scopeHas(deep, "late") === false)
}

// ===== SECTION 30: isGenerator =====
// The floor's tag-15 test, host id 69 - the value js_gen_create makes and
// js_gen_next drives. It exists because layer 2 had no way to ask "is this a
// generator" and languages/lib/php-rt.metajs had to GUESS: exclude __dict,
// __refcell, __isclass, __class and length one at a time, then test whether
// v["next"] is callable. PHP's own migration report calls that "the one guess in
// this port", and it is wrong in principle for any object with a callable `next`.
//
// A NUMERIC js_tag(v) was considered and rejected: js_genfn is a tag 16 CELL in
// the C floor and a *hostFunc in abnf/jsrt.go, so one number would be 16 in one
// half and 8 in the other, and metajs-interpreter.abnf has no tag numbering at
// all - a closure, a host function and a bound method are one JS function there.
// Every other distinction a tag would carry already has a name layer 2 can use
// (typeof, sintIs, floIs, and `typeof v.length == "number"` for an array).
//
// WHAT THIS SECTION CAN AND CANNOT PROVE, stated rather than implied: MetaJS has
// no generators - no `function*`, and metajs-to-llvm-ir.abnf emits no js_genfn -
// so no MetaJS program can construct one in ANY engine. The true arm is reachable
// only from an emitter, i.e. only from a layer-2 file of a language that has
// generators. What is pinned here is that all three halves answer false for every
// value a MetaJS program can build, INCLUDING the shape php-rt.metajs's guess
// gets wrong (gen05: an ordinary object with a callable `next`).
function s30() {
    check("gen01", isGenerator(1) === false && isGenerator("s") === false)
    check("gen02", isGenerator(undefined) === false && isGenerator(null) === false)
    check("gen03", isGenerator(true) === false && isGenerator([1, 2]) === false)
    check("gen04", isGenerator({}) === false && isGenerator(check) === false)
    var fake = {next: function() { return {done: true} }}
    check("gen05", isGenerator(fake) === false && typeof fake.next === "function")
    check("gen06", isGenerator(scopeNew()) === false)
    check("gen07", isGenerator(sint(0, 5)) === false && isGenerator(flo(1.5, 0)) === false)
    // It is a REFINEMENT of typeof and never contradicts it: everything that
    // could be a generator answers "object" there.
    check("gen08", typeof fake === "object" && isGenerator(fake) === false)
}

// ===== SECTION 31: the host-global SET, all 48 names on both halves =====
// docs/todo.md 4.1. This section exists because it COULD NOT BE WRITTEN before:
// the --full runner makes both halves run this same file and report the same
// assertion count, so a check on a name only one half bound failed on the other
// by construction, and it could not be guarded either - `typeof sprint` answered
// "function" under the compiler and a hard `variable not defined: sprint` under
// the interpreter, which is an abort and not a false.
//
// The set is the C floor's seed_root list in languages/lib/runtime.c, and that is
// the authority for the other two engines (programJSBindings in abnf/jsrtint.go,
// hostGlobals in metajs-interpreter.abnf). abnf/hostglobals_test.go parses all
// three and fails if they stop matching; this section is the runtime half of the
// same statement and adds what a source parse cannot see - that the names are
// bound to something of the right KIND, in a native binary too.
//
// WHAT CANNOT BE ASSERTED HERE: absence. `typeof Object` is not "undefined" in
// any of the three engines, it is a hard "variable not defined" abort, so the
// removal of Object and rawSet from the program set is pinned by the Go test
// only.
function s31() {
    // The six output and diagnostic names.
    check("hg01", typeof println === "function" && typeof print === "function")
    check("hg02", typeof printf === "function" && typeof sprintf === "function")
    check("hg03", typeof eprintln === "function" && typeof sprint === "function")
    // fail() is a runtime ERROR and exits, so only its binding can be checked.
    check("hg04", typeof fail === "function" && typeof exit === "function")
    check("hg05", typeof parseInt === "function" && typeof parseFloat === "function")
    check("hg06", typeof byteLen === "function" && typeof floPrec === "function")
    // The eleven sized-integer names.
    check("hg07", typeof sint === "function" && typeof sintRaw === "function")
    check("hg08", typeof sintIs === "function" && typeof sintHi === "function")
    check("hg09", typeof sintLo === "function" && typeof sintWidth === "function")
    check("hg10", typeof sintUns === "function" && typeof sintOp === "function")
    check("hg11", typeof sintCmp === "function" && typeof sintConv === "function")
    check("hg12", typeof sintStr === "function" && typeof sintNum === "function")
    // The ten boxed-double names.
    check("hg13", typeof flo === "function" && typeof floIs === "function")
    check("hg14", typeof floNum === "function" && typeof floStyle === "function")
    check("hg15", typeof floStr === "function" && typeof floOp === "function")
    check("hg16", typeof floEq === "function" && typeof floAbs === "function")
    check("hg17", typeof floMax === "function" && typeof floMin === "function")
    // Object walking, the scope API and the generator test.
    check("hg18", typeof keysOf === "function" && typeof delKey === "function")
    check("hg19", typeof scopeNew === "function" && typeof scopeParent === "function")
    check("hg20", typeof scopeGet === "function" && typeof scopeHas === "function")
    check("hg21", typeof scopeDecl === "function" && typeof isGenerator === "function")
    // The five non-function names. anytype is the declaration marker and is an
    // object in all three engines (a cell of tag 12 in the floor).
    check("hg22", typeof Infinity === "number" && typeof NaN === "number")
    check("hg23", typeof Math === "object" && typeof anytype === "object")
    // String and Array are PLAIN OBJECTS carrying one method each, not the host
    // engine's constructors: seed_host(strobj, "fromCharCode", 30) and
    // seed_host(arrobj, "isArray", 31) in the floor's boot(). The interpreter used
    // to bind the grammar host's own String here, which answered "function".
    check("hg24", typeof String === "object" && typeof Array === "object")
    check("hg25", typeof String.fromCharCode === "function" && typeof Array.isArray === "function")

    // ----- Math's MEMBERS, docs/todo.md 1.1 -----
    // hg23 above says `typeof Math === "object"` and says nothing about what is
    // IN it, and for a long time the three engines disagreed wildly: the C floor
    // seeded eleven methods and two properties, abnf/jsrt.go thirty-two and
    // eight, and goja - the grammar host THIS half runs on - thirty-five and
    // eight. abnf/hostglobals_test.go's TestMathMembersAgree is the source-parse
    // half of the statement (and the only place goja's own set can be read);
    // these rows are the runtime half, and they are what makes the assertion
    // reach the C FLOOR, which no Go test can run.
    check("hg25a", typeof Math.sin === "function" && typeof Math.cos === "function")
    check("hg25b", typeof Math.tan === "function" && typeof Math.atan2 === "function")
    check("hg25c", typeof Math.asin === "function" && typeof Math.acos === "function")
    check("hg25d", typeof Math.sinh === "function" && typeof Math.cosh === "function")
    check("hg25e", typeof Math.tanh === "function" && typeof Math.asinh === "function")
    check("hg25f", typeof Math.acosh === "function" && typeof Math.atanh === "function")
    check("hg25g", typeof Math.exp === "function" && typeof Math.expm1 === "function")
    check("hg25h", typeof Math.log === "function" && typeof Math.log1p === "function")
    check("hg25i", typeof Math.log2 === "function" && typeof Math.log10 === "function")
    check("hg25j", typeof Math.cbrt === "function" && typeof Math.hypot === "function")
    check("hg25k", typeof Math.clz32 === "function" && typeof Math.fround === "function")
    check("hg25l", typeof Math.LN2 === "number" && typeof Math.LN10 === "number")
    check("hg25m", typeof Math.LOG2E === "number" && typeof Math.LOG10E === "number")
    check("hg25n", typeof Math.SQRT2 === "number" && typeof Math.SQRT1_2 === "number")
    // And the ANSWERS, which is the part that makes the C floor's port of Go's
    // math package a measurement rather than a claim. Read out of an array so no
    // constant folder can answer at compile time.
    var mv = [1, 8, 27, 0.5, 3, 4, 100]
    check("hg25o", Math.sin(mv[0]) === 0.8414709848078965 && Math.cos(mv[0]) === 0.5403023058681398)
    check("hg25p", Math.log2(mv[1]) === 3 && Math.log10(mv[6]) === 2 && Math.cbrt(mv[2]) === 3)
    check("hg25q", Math.exp(mv[0]) === 2.718281828459045 && Math.expm1(mv[0]) === 1.718281828459045)
    check("hg25r", Math.hypot(mv[4], mv[5]) === 5 && Math.atan2(mv[0], mv[3]) === 1.1071487177940904)
    check("hg25s", Math.clz32(mv[0]) === 31 && Math.fround(mv[3]) === 0.5)
    check("hg25t", Math.LN2 === 0.6931471805599453 && Math.SQRT2 === 1.4142135623730951)

    // The six names the interpreter half gained, exercised rather than named.
    check("hg26", Infinity - 1 === Infinity && 0 - Infinity < 0)
    check("hg27", NaN !== NaN && !(NaN === NaN))
    check("hg28", Array.isArray([1, 2]) === true)
    check("hg29", Array.isArray("x") === false && Array.isArray({}) === false)
    // byteLen is UTF-8 bytes where .length is UTF-16 code units.
    check("hg30", byteLen("abc") === 3 && "abc".length === 3)
    check("hg31", byteLen("é") === 2 && "é".length === 1)
    check("hg32", byteLen("🎉") === 4 && "🎉".length === 2)
    // sprint is fmt.Sprint: a space between two adjacent operands only when
    // NEITHER of them is a string (fmt_sprint in languages/lib/runtime.c tests
    // tag 4 on both sides).
    check("hg33", sprint("a", "b") === "ab")
    check("hg34", sprint(1, 2) === "1 2")
    check("hg35", sprint("a", 1) === "a1" && sprint(1, "a") === "1a")
    check("hg36", sprint() === "")
    check("hg37", String.fromCharCode(65, 66) === "AB")
}

// ===== SECTION 32: string case mapping is the SIMPLE per-rune mapping =====
// docs/todo.md 4.1, the second half. Real JavaScript uses the FULL Unicode case
// mapping, in which a few characters change length: "ss".toUpperCase() is "SS" in
// node, "fi" (the U+FB01 ligature) becomes "FI", and "I" (U+0130) lowercases to
// "i" plus a combining dot. This project answers the SIMPLE per-rune mapping in
// all four engines instead - Go's strings.ToUpper / strings.ToLower.
//
// THE DECISION, and why it went this way rather than toward the JS spec: full
// case mapping in the C floor is `case_map`, which was built, passed every gate,
// was measured at 135x on a real workload and was reverted
// (docs/working-on-this-project.md chapter 5). Three of the four engines already
// did simple mapping; the fourth was goja, and moving goja costs nothing - it now
// calls the same Go function the frozen engine calls
// (installGojaCaseMapping in abnf/commonscript.go).
//
// Until then this was a LIVE goja-vs--frozen divergence in the interpreter half
// that ./test.sh could not see, because no matrix entry upper-cased a non-ASCII
// string. These rows are what makes it visible from now on.
function s32() {
    check("cm01", "abc".toUpperCase() === "ABC" && "ABC".toLowerCase() === "abc")
    check("cm02", "é".toUpperCase() === "É" && "É".toLowerCase() === "é")
    // The three characters where the full mapping differs from the simple one.
    check("cm03", "ß".toUpperCase() === "ß" && "ß".toUpperCase().length === 1)
    check("cm04", "ﬁ".toUpperCase() === "ﬁ" && "ﬁ".toUpperCase().length === 1)
    check("cm05", "İ".toLowerCase() === "i" && "İ".toLowerCase().length === 1)
    // Characters whose simple mapping IS the answer, so both models agree.
    check("cm06", "ı".toUpperCase() === "I" && "ǉ".toUpperCase() === "Ǉ")
    check("cm07", "I".toLowerCase() === "i" && "i".toUpperCase() === "I")
    // Case mapping never changes the code-unit count in the simple model.
    check("cm08", "ßﬁİ".toUpperCase().length === 3 && "ßﬁİ".toLowerCase().length === 3)
    // Astral pairs still map (the surrogate pair is one rune to Go): U+10428
    // DESERET SMALL LETTER LONG I upper-cases to U+10400.
    check("cm09", String.fromCharCode(55297, 56360).toUpperCase() ===
                  String.fromCharCode(55297, 56320))
    check("cm10", "".toUpperCase() === "" && " ".toUpperCase() === " ")
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
    s24() // SECTION-CALL 24
    s25() // SECTION-CALL 25
    s26() // SECTION-CALL 26
    s27() // SECTION-CALL 27
    s28() // SECTION-CALL 28
    s29() // SECTION-CALL 29
    s30() // SECTION-CALL 30
    s31() // SECTION-CALL 31
    s32() // SECTION-CALL 32
    println("full: " + checks + " checks, " + failures + " failures")
    return failures
}
