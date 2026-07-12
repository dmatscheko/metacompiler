/* MetaJS feature-matrix test for the interpreter (metajs-interpreter.abnf) and
 * the LLVM-IR compiler (metajs-to-llvm-ir.abnf). It replaces the four
 * algorithm-themed metajs-test-big-* stress tests: instead of sorts, Ackermann
 * and DP tables, every implemented construct is exercised with the SMALLEST
 * program that can prove it works - loops run 0, 1, 3 or 4 times, recursion
 * stays below depth 6. The MetaJS discipline rules are covered on their legal
 * side: declaration before use, the type pin of the first non-null value,
 * var v = anytype as the declared exemption, and members staying free.
 * A failed check prints its id (so a diff pinpoints it) and main() returns the
 * failure count; exit 0 and byte-identical output on all four legs
 * (interpreter/compiler x goja/-frozen) mean everything passed. **/

var failures = 0;
var checks = 0;

function check(name, got, want) {
    checks = checks + 1;
    if (got !== want) {
        println("FAIL " + name + ": got " + got + " want " + want);
        failures = failures + 1;
    }
}

// ----- numbers, arithmetic, precedence -----
check("arith-precedence", 2 + 3 * 4, 14);
check("arith-paren", (2 + 3) * 4, 20);
check("arith-left-assoc", 20 - 5 - 3, 12);
check("arith-unary-minus", -3 + 5, 2);
check("arith-double-neg", -(-5), 5);
check("arith-float-div", 7 / 2, 3.5);
check("arith-mod", 7 % 3, 1);
check("arith-mod-neg", -7 % 3, -1);
check("arith-float", 0.5 * 4, 2);
check("arith-float-inexact", 0.1 + 0.2 !== 0.3, true);
check("hex-literal", 0x1A, 26);
check("hex-upper", 0XFF, 255);
check("hex-arith", 0x10 + 0x0f, 31);
check("num-to-string", "" + 42, "42");
check("float-to-string", "" + 2.5, "2.5");

// ----- assignment: compound, chained, as an expression, inc/dec -----
var ca = 5;
ca += 3;  check("plus-assign", ca, 8);
ca -= 2;  check("minus-assign", ca, 6);
ca *= 4;  check("times-assign", ca, 24);
ca /= 6;  check("div-assign", ca, 4);
ca %= 3;  check("mod-assign", ca, 1);
var cs = "hi";
cs += "!";
check("string-plus-assign", cs, "hi!");
var q;
check("undef-init", q, undefined);
check("assign-expr", (q = 5) + 1, 6);
var aa = 0;
var bb = 0;
aa = bb = 3;
check("chained-assign", aa + bb, 6);
var m1 = 1, m2 = "two";
check("multi-declarator", m1 + m2, "1two");
var pn = 5;
check("post-inc-value", pn++, 5);
check("pre-inc-value", ++pn, 7);
check("post-dec-value", pn--, 7);
check("pre-dec-value", --pn, 5);
check("mixed-incdec", ++pn + pn++, 12);
check("mixed-incdec-effect", pn, 7);

// ----- bitwise and shifts -----
check("bit-and", 255 & 300, 44);
check("bit-or", 6 | 3, 7);
check("bit-xor", 5 ^ 1, 4);
check("bit-not", ~5, -6);
check("bit-shl", 1 << 4, 16);
check("bit-shr-neg", -8 >> 1, -4);
check("bit-ushr", -8 >>> 28, 15);
check("bit-precedence", 1 | 2 & 3, 3);
check("bit-int-idiom", (7 / 2) | 0, 3);
check("bit-int-idiom-neg", (-7 / 2) | 0, -3);
check("bit-string-coerce", "12" | 0, 12);

// ----- host builtins -----
check("parseint", parseInt("42"), 42);
check("parseint-stops", parseInt("7.9"), 7);
check("parseint-prefix", parseInt("12px"), 12);
check("parseint-radix", parseInt("ff", 16), 255);
check("parsefloat", parseFloat("2.5"), 2.5);
check("parsefloat-prefix", parseFloat("2.5x"), 2.5);
check("math-floor", Math.floor(2.9), 2);
check("math-floor-neg", Math.floor(-2.1), -3);
check("math-abs", Math.abs(-7), 7);
check("math-max", Math.max(1, 3, 2), 3);
check("math-min", Math.min(3, 1, 2), 1);
check("math-imul", Math.imul(3, 4), 12);
check("math-imul-wrap", Math.imul(2147483647, 2), -2);
check("sprintf", sprintf("x=%d", 7), "x=7");
check("sprintf-str", sprintf("%s=%d", "n", 5), "n=5");

// ----- comparisons and equality -----
check("lt", 2 < 3, true);
check("ge", 3 >= 3, true);
check("le", 3 <= 3, true);
check("loose-eq-numstr", 1 == "1", true);
check("strict-eq-numstr", 1 === "1", false);
check("loose-null-undef", null == undefined, true);
check("strict-null-undef", null === undefined, false);
check("loose-zero-empty", 0 == "", true);
check("ne", 1 != 2, true);
check("strict-ne", 1 !== "1", true);
check("string-lt", "apple" < "banana", true);

// ----- truthiness, || and &&, ternary -----
check("or-value", 0 || "x", "x");
check("or-last", "" || 0, 0);
check("and-value", 5 && 7, 7);
check("and-stop", null && 1, null);
check("not-zero", !0, true);
check("not-empty-string", !"", true);
check("not-zero-string", !"0", false);
check("not-not", !!5, true);
var sideCalls = 0;
function bump() { sideCalls++; return true; }
var sc1 = false && bump();
var sc2 = true || bump();
check("short-circuit-skipped", sideCalls, 0);
var sc3 = bump() && true;
check("short-circuit-ran", sideCalls, 1);
check("short-circuit-values", sc1 === false && sc2 === true && sc3 === true, true);
check("ternary", 2 > 1 ? "a" : "b", "a");
check("nested-ternary", 1 > 2 ? "a" : 3 > 2 ? "c" : "b", "c");

// ----- typeof -----
check("typeof-number", typeof 1, "number");
check("typeof-string", typeof "", "string");
check("typeof-bool", typeof true, "boolean");
check("typeof-undefined", typeof undefined, "undefined");
check("typeof-null", typeof null, "object");
check("typeof-object", typeof {}, "object");
check("typeof-array", typeof [1], "object");
check("typeof-function", typeof bump, "function");

// ----- strings -----
var str = "hello";
check("str-concat", "foo" + "bar", "foobar");
check("str-num-concat", "x" + 1 + 2, "x12");
check("num-str-concat", 1 + 2 + "x", "3x");
check("str-length", str.length, 5);
check("str-empty-length", "".length, 0);
check("str-charat", "abc".charAt(0), "a");
check("str-charat-last", "abc".charAt(2), "c");
check("str-charat-oob", "abc".charAt(9), "");
check("str-charcodeat", "A".charCodeAt(0), 65);
check("str-charcodeat-pos", str.charCodeAt(1), 101);
check("str-indexof", str.indexOf("ll"), 2);
check("str-indexof-missing", str.indexOf("z"), -1);
check("str-replace-first", str.replace("l", "_"), "he_lo");
check("str-slice", str.slice(1, 3), "el");
check("str-slice-open", str.slice(3), "lo");
check("str-slice-neg", str.slice(-3), "llo");
check("str-substring", "metacompiler".substring(4, 8), "comp");
check("str-substring-open", "metajs".substring(4), "js");
check("str-substring-swap", str.substring(3, 1), "el");
check("str-case-up", "aBc".toUpperCase(), "ABC");
check("str-case-down", "DoWn".toLowerCase(), "down");
check("str-trim", "  pad  ".trim(), "pad");
check("str-split-count", "a,b,c".split(",").length, 3);
check("str-split-part", "a,b,c".split(",")[1], "b");
check("str-join", ["x", "y", "z"].join("-"), "x-y-z");
check("str-split-join", "1 2 3".split(" ").join("+"), "1+2+3");
check("str-fromcharcode", String.fromCharCode(72, 105), "Hi");
check("str-index-brackets", "abc"[1], "b");

// ----- string escapes and quotes -----
check("escape-tab", "\t".length, 1);
check("escape-newline", "a\nb".charCodeAt(1), 10);
check("escape-hex", "\x41", "A");
check("escape-unicode", "é", "é");
check("escape-backslash", "a\\b".length, 3);
check("quotes-mixed", 'it\'s' + " " + "say \"hi\"", "it's say \"hi\"");

// ----- UTF-16 string semantics -----
check("utf16-length", "héllo".length, 5);
check("utf16-charcode", "é".charCodeAt(0), 233);
check("utf16-charat", "héllo".charAt(1), "é");
check("utf16-substring", "héllo".substring(0, 2), "hé");
check("utf16-fromcharcode", String.fromCharCode(233), "é");

// ----- control flow: if / while / do-while / for -----
function grade(n) {
    if (n > 10) { return "big"; }
    else if (n > 5) { return "mid"; }
    else { return "small"; }
}
check("if-elseif-else", grade(11) + grade(7) + grade(1), "bigmidsmall");

var w0 = 0;
while (w0 > 0) { w0 = w0 - 1; }        // runs zero times
check("while-zero", w0, 0);
var w3 = 0;
while (w3 < 3) { w3++; }               // runs three times
check("while-three", w3, 3);
var dw = 0;
do { dw++; } while (false);            // body runs exactly once
check("do-while-once", dw, 1);

var fsum = 0;
for (var fi = 1; fi <= 3; fi++) { fsum += fi; }
check("for-basic", fsum, 6);

var brk = "";
for (var bi = 0; bi < 9; bi++) {
    if (bi == 2) { break; }
    brk += bi;
}
check("for-break", brk, "01");

var cont = "";
for (var ci = 0; ci < 4; ci++) {
    if (ci % 2 == 1) { continue; }
    cont += ci;
}
check("for-continue", cont, "02");

var nested = "";
for (var oi = 0; oi < 2; oi++) {
    for (var ii = 0; ii < 3; ii++) {
        if (ii == 1) { break; }        // inner break must not end the outer loop
        nested = nested + oi + ii;
    }
}
check("nested-break", nested, "0010");

// ----- blocks: var/let/const are all block scoped in MetaJS -----
let sh = 1;
{
    let sh = 2;
    check("let-inner-shadow", sh, 2);
}
check("let-outer-after", sh, 1);
var shv = "out";
{
    var shv = "in";                    // a block redeclaration shadows, like let
    check("var-inner-shadow", shv, "in");
}
check("var-outer-after", shv, "out");
const fixed = 40 + 2;
check("const-value", fixed, 42);
;;; // empty statements are fine

// ----- switch: strict match, stacked labels, fallthrough, default -----
function classify(x) {
    var r = 0;
    switch (x) {
    case 0:
        r = 100;
        break;
    case 1:                            // stacked labels
    case 2:
        r = 12;
        break;
    case 3:
        r = 3;                         // falls through into case 4
    case 4:
        r += 40;
        break;
    default:
        r = -1;
    }
    return r;
}
check("switch-first", classify(0), 100);
check("switch-stacked", classify(1), 12);
check("switch-stacked2", classify(2), 12);
check("switch-fallthrough", classify(3), 43);
check("switch-late-entry", classify(4), 40);
check("switch-default", classify(9), -1);
check("switch-strict", classify("3"), -1);

function dayKind(d) {
    switch (d) {
    case "sat":
    case "sun": return "weekend";
    default: return "workday";
    }
}
check("switch-string", dayKind("sun"), "weekend");
check("switch-string-default", dayKind("wed"), "workday");

var sIL = 0;
for (var si = 0; si < 4; si++) {       // continue skips the tail, break only the switch
    switch (si % 3) {
    case 0: continue;
    case 1: sIL += 10; break;
    default: sIL += 1;
    }
    sIL += 100;
}
check("switch-in-loop", sIL, 211);

// ----- functions, closures, recursion -----
function add2(a, b) { return a + b; }
check("fn-args", add2(2, 3), 5);
function second(a, b) { return b; }
check("fn-missing-arg", second(1), undefined);
function early(n) {
    if (n < 0) { return "neg"; }
    return "pos";
}
check("fn-early-return", early(-1) + early(1), "negpos");

function fib(n) {
    if (n < 2) { return n; }
    return fib(n - 1) + fib(n - 2);
}
check("fn-recursion", fib(6), 8);
function isEven(n) { return n == 0 ? true : isOdd(n - 1); }
function isOdd(n) { return n == 0 ? false : isEven(n - 1); }
check("fn-mutual-recursion", isEven(4) && isOdd(5), true);

function sumAll() {                    // varargs via the arguments array
    var s = 0;
    for (var i = 0; i < arguments.length; i++) { s += arguments[i]; }
    return s;
}
check("fn-varargs", sumAll(1, 2, 3), 6);
check("fn-varargs-empty", sumAll(), 0);

function makeCounter(start) {
    var count = start;
    return function() { count += 1; return count; };
}
var c1 = makeCounter(0);
var c2 = makeCounter(100);
c1(); c1();
check("closure-counter", c1(), 3);
check("closure-independent", c2(), 101);

function makeAdder(n) { return function(x) { return x + n; }; }
var adders = [];
for (var ak = 0; ak < 3; ak++) { adders.push(makeAdder(ak * 10)); }
check("closures-in-loop", adders[0](1) + adders[1](1) + adders[2](1), 33);

function applyTwice(f, x) { return f(f(x)); }
function dbl(n) { return n * 2; }
check("fn-higher-order", applyTwice(dbl, 3), 12);
check("fn-anon-arg", applyTwice(function(n) { return n + 1; }, 5), 7);
var anon = function(v) { return v * v; };
check("fn-expression", anon(9), 81);
check("fn-iife", (function(n) { return n + 1; })(41), 42);

function noSemi(x) {                   // statements without semicolons
    var a = x + 1
    var b = a * 2
    return b
}
check("fn-no-semicolons", noSemi(4), 10);

// ----- the closure-as-object pattern (MetaJS has no this/new/classes) -----
function makeAccount(balance) {
    return {
        deposit: function(n) { balance += n; return balance; },
        get: function() { return balance; }
    };
}
var acct = makeAccount(10);
acct.deposit(5);
check("oo-closure-object", acct.get(), 15);

// ----- arrays -----
var arr = [3, 1, 4];
check("arr-length", arr.length, 3);
check("arr-index", arr[2], 4);
arr[1] = 10;
check("arr-store", arr[1], 10);
check("arr-push-returns-length", arr.push(7), 4);
check("arr-after-push", arr[3], 7);
check("arr-pop", arr.pop(), 7);
check("arr-pop-length", arr.length, 3);
check("arr-out-of-range", arr[99], undefined);
var sl = arr.slice(1);
check("arr-slice-open", sl.length, 2);
check("arr-slice-content", sl[0], 10);
var sl2 = [1, 2, 3, 4].slice(1, 3);
check("arr-slice-two", sl2[0] + sl2[1], 5);
check("arr-unshift-returns-length", sl.unshift(9), 3);
check("arr-unshift-content", sl[0], 9);
check("arr-shift", sl.shift(), 9);
check("arr-shift-length", sl.length, 2);
check("arr-indexof", [5, 6, 7].indexOf(7), 2);
check("arr-indexof-missing", [5, 6, 7].indexOf(8), -1);
var cat = [1].concat([2, 3]);
check("arr-concat-length", cat.length, 3);
check("arr-concat-content", cat[2], 3);
var nestedArr = [[1, 2], [3, 4]];
check("arr-nested", nestedArr[1][0], 3);
var el = [10];
el[0]--;
check("arr-element-dec", el[0], 9);
el[0] += 2;
check("arr-element-compound", el[0], 11);
var isum = 0;
var src = [1, 2, 3];
for (var xi = 0; xi < src.length; xi++) { isum += src[xi]; }
check("arr-iterate", isum, 6);
var drain = [3, 2, 1];
var got = [];
var dt;
while ((dt = drain.pop()) != undefined) { got.push(dt); }
check("arr-drain-pattern", got[0] * 100 + got[1] * 10 + got[2], 123);

// ----- objects -----
var obj = {n: 1, name: "thing", "spaced key": true};
check("obj-get", obj.n, 1);
check("obj-computed-get", obj["name"], "thing");
check("obj-string-key", obj["spaced key"], true);
obj.n = 5;
check("obj-set", obj.n, 5);
obj.fresh = "new";
check("obj-add", obj.fresh, "new");
obj["k" + 1] = 11;
check("obj-computed-set", obj.k1, 11);
check("obj-missing", obj.nothing, undefined);
obj.n++;
check("obj-member-inc", obj.n, 6);
obj.n += 10;
check("obj-member-compound", obj.n, 16);
var deep = {a: {b: [{c: 42}]}};
check("obj-deep-get", deep.a.b[0].c, 42);
deep.a.b[0].c = 43;
check("obj-deep-set", deep.a.b[0].c, 43);
var calc = {add: function(p, r) { return p + r; }, base: 100};
check("obj-method", calc.add(2, 3), 5);
check("obj-mixed", calc.base + calc.add(1, 1), 102);

// ----- assignment targets that reach through calls -----
var boxData = {n: 1, arr: [0, 0]};
function boxFn() { return boxData; }
boxFn().n = 99;
check("call-target-member", boxData.n, 99);
boxFn()["k" + 1] = 5;
check("call-target-computed", boxData.k1, 5);
boxFn().arr[1] = 7;
check("call-target-nested", boxData.arr[1], 7);
boxFn().n++;
check("call-target-incdec", boxData.n, 100);
boxFn().n += 10;
check("call-target-compound", boxData.n, 110);

// ----- the type pin: first non-null value decides the type class -----
var tn = 1;
tn = 2.5;
tn = -7;                               // int and float are one number class
check("pin-number-stays", tn, -7);
var th = 0xff;
th = 0x100;
check("pin-hex-number", th, 256);
var ts = "a";
ts += "b";
ts = "renewed";
check("pin-string-stays", ts, "renewed");
var tb = true;
tb = 1 > 2;
check("pin-bool-stays", tb, false);
var late;
check("pin-late-undefined", late, undefined);
late = "now a string";
late = "still a string";               // untyped until the first real value
check("pin-late-typing", late, "still a string");
var tu = 5;
tu = undefined;
tu = 7;                                // undefined never unpins
check("pin-undefined-no-unpin", tu, 7);
var to = {a: 1};
to = [1, 2, 3];
to = null;
to = {back: true};                     // arrays/objects/null share the object class
check("pin-object-class-shared", to.back, true);
var tf = function(x) { return x * 2; };
tf = fib;
check("pin-function-stays", tf(6), 8);
function ident(x) { return x; }
check("pin-param-fresh-num", ident(4), 4);
check("pin-param-fresh-str", ident("four"), "four");
var freeBox = {v: 1};
freeBox.v = "free";                    // members are not variables: no pin
check("pin-members-free", freeBox.v, "free");
freeBox.v = [1, 2];
check("pin-members-free-arr", freeBox.v.length, 2);

// ----- anytype: the declared exemption from pinning -----
var any = anytype;
check("anytype-starts-undefined", any, undefined);
any = 1;
any = "one";
any = [1, 2];
check("anytype-retypes", any.length, 2);
var any = 7;                           // a redeclaration with a value pins again
any = 8;
check("anytype-redecl-repins", any, 8);

// ----- exceptions: throw / catch / finally / control flow -----
var exLog = "";
try {
    exLog += "a";
    throw "boom";
    exLog += "X";
} catch (e) {
    exLog += "b" + e;
} finally {
    exLog += "c";
}
check("try-throw-catch-finally", exLog, "abboomc");

function risky(n) {
    if (n > 3) { throw {code: n}; }
    return n * 2;
}
var caught = -1;
try {
    var rr = risky(5);                 // the throw unwinds out of the call
    check("try-unreached", true, false);
} catch (e) {
    caught = e.code;
}
check("throw-unwinds-call", caught, 5);
check("try-no-throw-value", risky(2), 4);

var order = "";
try { order += "t"; } catch (e) { order += "c"; } finally { order += "f"; }
check("try-no-throw", order, "tf");

var noBind = "";
try { throw "nb"; } catch { noBind = "caught"; }
check("catch-no-binding", noBind, "caught");

var finTail = 0;
try { finTail += 1; } finally { finTail += 10; }
check("finally-no-catch", finTail, 11);

function relabel() {
    var result = "";
    try {
        try { throw "inner"; } catch (e) { throw "rethrown:" + e; }
    } catch (e2) {
        result = e2;
    }
    return result;
}
check("try-rethrow", relabel(), "rethrown:inner");

function withReturn(n) {
    try { if (n > 0) { return n * 10; } throw "neg"; }
    catch (e) { return -1; }
    finally { }
}
check("return-out-of-try", withReturn(4), 40);
check("return-out-of-catch", withReturn(-1), -1);

function nestedReturn() {
    try {
        try { return 9; } finally { }
    } finally { }
    return 0;
}
check("return-nested-tries", nestedReturn(), 9);

var finRan = 0;
function retAcross() {
    try { return "from-try"; } finally { finRan = 1; }
}
check("return-across-try", retAcross(), "from-try");
check("finally-ran-on-return", finRan, 1);

function retFinallyOverrides() {
    try { return 1; } finally { return 2; }
}
check("return-in-finally-overrides", retFinallyOverrides(), 2);

function finallyCancelsThrow() {
    try {
        try { throw "deep"; } finally { return "finally-wins"; }
    } catch (e) {
        return "outer-caught";
    }
}
check("finally-overrides-throw", finallyCancelsThrow(), "finally-wins");

function loopBreakTry() {
    var s = 0;
    for (var i = 0; i < 9; i++) {
        try { if (i == 3) { break; } s += i; } finally { }
    }
    return s;                          // 0+1+2 = 3
}
check("break-out-of-try", loopBreakTry(), 3);

function loopContinueTry() {
    var s = 0;
    for (var i = 0; i < 4; i++) {
        try { if (i == 2) { continue; } s += i; } catch (e) { }
    }
    return s;                          // 0+1+3 = 4
}
check("continue-out-of-try", loopContinueTry(), 4);

function breakInFinally() {
    var i = 0;
    while (i < 9) {
        i++;
        try { i += 10; } finally { break; }
    }
    return i;                          // one iteration: 1 + 10
}
check("break-in-finally", breakInFinally(), 11);

function continueInFinally() {
    var n = 0;
    var hits = 0;
    while (n < 3) {
        n++;
        try { hits += 1; throw "x"; } finally { continue; }
    }
    return hits;                       // the continue cancels each rethrow
}
check("continue-in-finally", continueInFinally(), 3);

// ----- everything combined in one small pipeline (3-element data flow) -----
function transform(list) {
    var out = [];
    for (var i = 0; i < list.length; i++) {
        var n = list[i];
        try {
            if (n < 0) { throw "neg"; }
            out.push(n % 2 == 0 ? "e" + n : "o" + n);
        } catch (e) {
            out.push("x");
        }
    }
    return out.join("");
}
check("combined-pipeline", transform([1, 2, -3]), "o1e2x");

function main() {
    println("features: " + checks + " checks, " + failures + " failures");
    return failures;
}
