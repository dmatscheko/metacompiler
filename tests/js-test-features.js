// Fast feature-matrix test for the JavaScript interpreter (js-interpreter.abnf) and
// the LLVM-IR compiler (js-to-llvm-ir.abnf). It replaces the four algorithm-themed
// js-test-big-* stress tests: instead of large loops (Ackermann, sieves, 12x12
// tables) every implemented construct is exercised with the SMALLEST program that
// can prove it works - loops run 0, 1 or 3 times, recursion stays below depth 6.
// A failed check prints its id (so a diff pinpoints it) and main() returns the
// failure count; exit 0 and byte-identical output on all four legs
// (interpreter/compiler x goja/-frozen) mean everything passed.

var failures = 0;
var checks = 0;

function check(id, cond) {
    checks = checks + 1;
    if (!cond) { println("FAIL " + id); failures = failures + 1; }
}

// ----- numbers, arithmetic, precedence -----
check("arith-precedence", 2 + 3 * 4 === 14);
check("arith-paren", (2 + 3) * 4 === 20);
check("arith-unary-minus", -3 + 5 === 2);
check("arith-mod", 7 % 3 === 1);
check("arith-mod-neg", -7 % 3 === -1);
check("arith-float-div", 7 / 2 === 3.5);
check("arith-float", 0.1 + 0.2 !== 0.3);
check("arith-chain", 20 - 5 - 3 === 12);
check("arith-compound", (function() { var x = 5; x += 3; x -= 2; x *= 4; x /= 6; return x === 4; })());
check("arith-incdec", (function() { var i = 5; var a = i++; var b = ++i; return a === 5 && b === 7 && i === 7; })());
check("arith-decrement", (function() { var i = 5; i--; --i; return i === 3; })());

// ----- bitwise and shifts -----
check("bit-and-or-xor", (6 & 3) === 2 && (6 | 3) === 7 && (6 ^ 3) === 5);
check("bit-not", (~5) === -6);
check("bit-shl", (1 << 4) === 16);
check("bit-shr-neg", (-8 >> 1) === -4);
check("bit-ushr", (-1 >>> 28) === 15);

// ----- special numbers and conversions -----
check("num-nan", (0 / 0) !== (0 / 0));
check("num-infinity", 1 / 0 > 1000000);
check("num-parseint", parseInt("42") === 42);
check("num-parseint-radix", parseInt("ff", 16) === 255);
check("num-parseint-prefix", parseInt("12px") === 12);
check("num-parsefloat", parseFloat("2.5x") === 2.5);
check("num-tostring", "" + 42 === "42" && "" + 2.5 === "2.5");
check("math-floor", Math.floor(2.9) === 2 && Math.floor(-2.1) === -3);
check("math-abs", Math.abs(-7) === 7);
check("math-max-variadic", Math.max(1, 3, 2) === 3 && Math.min(3, 1, 2) === 1);

// ----- equality and coercion -----
check("eq-loose-numstr", 1 == "1");
check("eq-strict-numstr", !(1 === "1"));
check("eq-null-undef", null == undefined && !(null === undefined));
check("eq-bool-num", 0 == false && 1 == true);
check("ne", 1 != 2 && 1 !== "1");

// ----- booleans, truthiness, short-circuit -----
check("truthy-values", !!"x" && !!1 && !!{} && !![]);
check("falsy-values", !"" && !0 && !null && !undefined);
check("logic-and-value", (0 && "no") === 0 && (1 && "yes") === "yes");
check("logic-or-value", (0 || "fb") === "fb" && ("a" || "b") === "a");
var sideEffects = 0;
function bump() { sideEffects = sideEffects + 1; return true; }
var noRun = false && bump();
var oneRun = true && bump();
var skipRun = true || bump();
check("logic-short-circuit", sideEffects === 1);
check("ternary", (5 > 3 ? "a" : "b") === "a" && (5 < 3 ? "a" : "b") === "b");
check("not-chain", !!!false === true);

// ----- strings -----
check("str-concat", "foo" + "bar" === "foobar");
check("str-num-concat", "x" + 1 + 2 === "x12" && 1 + 2 + "x" === "3x");
check("str-length", "hello".length === 5 && "".length === 0);
check("str-charat", "abc".charAt(0) === "a" && "abc".charAt(2) === "c" && "abc".charAt(9) === "");
check("str-charcodeat", "A".charCodeAt(0) === 65);
check("str-index", "abcabc".indexOf("b") === 1 && "abc".indexOf("z") === -1);
check("str-slice", "hello".slice(1, 3) === "el" && "hello".slice(-3) === "llo");
check("str-substring", "hello".substring(1, 3) === "el" && "hello".substring(3, 1) === "el");
check("str-split", "a,b,c".split(",").length === 3 && "a,b,c".split(",")[1] === "b");
check("str-case", "aBc".toUpperCase() === "ABC" && "aBc".toLowerCase() === "abc");
check("str-trim", "  x  ".trim() === "x");
check("str-replace", "aXa".replace("X", "Y") === "aYa");
check("str-compare", "apple" < "banana" && "a" < "b" && !("b" < "a"));
check("str-escapes", "a\tb".length === 3 && "a\nb".charCodeAt(1) === 10 && "\\".length === 1 && "\"".length === 1);
check("str-unicode-len", "héllo".length === 5);
check("str-unicode-code", "é".charCodeAt(0) === 233);
check("str-unicode-slice", "héllo".slice(0, 2) === "hé");
check("str-fromcharcode", String.fromCharCode(72, 105) === "Hi");
var tpl = 6;
check("str-template", `v=${tpl + 1}!` === "v=7!");
check("str-index-brackets", "abc"[1] === "b");

// ----- typeof -----
check("typeof-all", typeof 1 === "number" && typeof "s" === "string" && typeof true === "boolean"
    && typeof undefined === "undefined" && typeof bump === "function" && typeof {} === "object");

// ----- control flow: if / while / do-while / for -----
function grade(n) {
    if (n > 10) { return "big"; }
    else if (n > 5) { return "mid"; }
    else { return "small"; }
}
check("if-elseif-else", grade(11) === "big" && grade(7) === "mid" && grade(1) === "small");

var w = 0;
while (w > 0) { w = w - 1; }           // runs zero times
check("while-zero", w === 0);
var w3 = 0;
while (w3 < 3) { w3 = w3 + 1; }        // runs three times
check("while-three", w3 === 3);

var dw = 0;
do { dw = dw + 1; } while (false);     // body runs exactly once
check("do-while-once", dw === 1);

var forSum = 0;
for (var fi = 1; fi <= 3; fi++) { forSum = forSum + fi; }
check("for-basic", forSum === 6);

var brk = "";
for (var bi = 0; bi < 9; bi++) {
    if (bi === 2) { break; }
    brk = brk + bi;
}
check("for-break", brk === "01");

var cont = "";
for (var ci = 0; ci < 4; ci++) {
    if (ci % 2 === 1) { continue; }
    cont = cont + ci;
}
check("for-continue", cont === "02");

var nested = "";
for (var oi = 0; oi < 2; oi++) {
    for (var ii = 0; ii < 3; ii++) {
        if (ii === 1) { break; }       // inner break must not end the outer loop
        nested = nested + oi + ii;
    }
}
check("nested-break", nested === "0010");

// ----- switch: match, fallthrough, default -----
function sw(x) {
    var out = "";
    switch (x) {
        case 1: out = out + "one"; break;
        case 2: out = out + "two";           // falls through
        case 3: out = out + "three"; break;
        default: out = out + "other";
    }
    return out;
}
check("switch-match", sw(1) === "one");
check("switch-fallthrough", sw(2) === "twothree");
check("switch-case3", sw(3) === "three");
check("switch-default", sw(9) === "other");

// ----- functions, closures, recursion -----
function add(a, b) { return a + b; }
check("fn-args", add(2, 3) === 5);
check("fn-missing-arg", (function(a, b) { return b === undefined; })(1));
function fib(n) { return n < 2 ? n : fib(n - 1) + fib(n - 2); }
check("fn-recursion", fib(6) === 8);
function isEven(n) { return n === 0 ? true : isOdd(n - 1); }
function isOdd(n) { return n === 0 ? false : isEven(n - 1); }
check("fn-mutual-recursion", isEven(4) && isOdd(5));

function makeCounter() {
    var c = 0;
    return function() { c = c + 1; return c; };
}
var c1 = makeCounter();
var c2 = makeCounter();
c1(); c1();
check("closure-independent", c1() === 3 && c2() === 1);

function applyTwice(f, x) { return f(f(x)); }
check("fn-higher-order", applyTwice(function(n) { return n * 2; }, 3) === 12);
var arrow1 = x => x + 1;
var arrow2 = (a, b) => { return a * b; };
check("arrow-functions", arrow1(4) === 5 && arrow2(3, 4) === 12);
check("iife", (function(n) { return n + 1; })(41) === 42);

var fnExpr = function(n) { return n + 1; };
check("fn-expression", fnExpr(41) === 42);

// ----- arrays -----
var arr = [10, 20, 30];
check("arr-literal-index", arr.length === 3 && arr[0] === 10 && arr[2] === 30);
arr[1] = 21;
check("arr-write", arr[1] === 21);
arr.push(40);
check("arr-push", arr.length === 4 && arr[3] === 40);
check("arr-pop", arr.pop() === 40 && arr.length === 3);
check("arr-oob", arr[9] === undefined);
check("arr-indexof", [5, 6, 7].indexOf(6) === 1 && [5].indexOf(9) === -1);
check("arr-join", [1, 2, 3].join("-") === "1-2-3");
check("arr-slice", [1, 2, 3, 4].slice(1, 3).join(",") === "2,3");
check("arr-concat", [1].concat([2, 3]).length === 3);
check("arr-nested", [[1, 2], [3]][0][1] === 2);
var arrSum = 0;
var src = [1, 2, 3];
for (var ai = 0; ai < src.length; ai++) { arrSum = arrSum + src[ai]; }
check("arr-iterate", arrSum === 6);
var ofSum = 0;
for (var v of [4, 5, 6]) { ofSum = ofSum + v; }
check("for-of", ofSum === 15);

// ----- objects -----
var obj = { a: 1, "b-key": 2 };
check("obj-literal", obj.a === 1 && obj["b-key"] === 2);
obj.a = 10;
obj.c = 3;
check("obj-write-add", obj.a === 10 && obj.c === 3);
check("obj-missing", obj.zzz === undefined);
var nestedObj = { in1: { in2: { val: 7 } } };
check("obj-nested", nestedObj.in1.in2.val === 7);
var keys = "";
for (var k in { x: 1, y: 2, z: 3 }) { keys = keys + k; }
check("for-in-order", keys === "xyz");
var methBase = 4;
var methObj = { twice: function() { return methBase * 2; } };
check("obj-method", methObj.twice() === 8);

// ----- destructuring (deep coverage lives in js-test-complete.js) -----
var [d1, d2] = [1, 2];
var { a: dA, c: dC } = { a: "x", c: "y" };
check("destructuring", d1 === 1 && d2 === 2 && dA === "x" && dC === "y");

// ----- classes, inheritance, super -----
class Animal {
    constructor(name) { this.name = name; }
    speak() { return this.name + " makes a sound"; }
    kind() { return "animal"; }
}
class Dog extends Animal {
    constructor(name) { super(name); this.tricks = 1; }
    speak() { return this.name + " barks"; }
    describe() { return super.speak() + ", " + this.kind(); }
}
var pet = new Dog("Rex");
check("class-fields", pet.name === "Rex" && pet.tricks === 1);
check("class-override", pet.speak() === "Rex barks");
check("class-super-inherit", pet.describe() === "Rex makes a sound, animal");
check("class-base", new Animal("Cat").speak() === "Cat makes a sound");

// ----- exceptions: throw / catch / finally / control flow -----
var exOrder = "";
try {
    exOrder = exOrder + "t";
    throw "boom";
} catch (e) {
    exOrder = exOrder + "c" + e;
} finally {
    exOrder = exOrder + "f";
}
check("try-throw-catch-finally", exOrder === "tcboomf");

var noThrow = "";
try { noThrow = noThrow + "t"; } finally { noThrow = noThrow + "f"; }
check("try-no-throw", noThrow === "tf");

function nestedThrow() {
    try {
        try { throw 1; } finally { return "inner-finally-wins"; }
    } catch (e) {
        return "outer-caught";
    }
}
check("finally-overrides-throw", nestedThrow() === "inner-finally-wins");

function retAcrossTry() {
    try { return "from-try"; } finally { bump(); }
}
check("return-across-try", retAcrossTry() === "from-try");

function retInFinally() {
    try { return 1; } finally { return 2; }
}
check("return-in-finally", retInFinally() === 2);

function breakInFinally() {
    var i = 0;
    while (true) {
        i = i + 1;
        try { bump(); } finally { break; }
    }
    return i;
}
check("break-in-finally", breakInFinally() === 1);

function throwValue() {
    try { throw { code: 42 }; } catch (e) { return e.code; }
}
check("throw-object", throwValue() === 42);

function rethrow() {
    try {
        try { throw "deep"; } catch (e) { throw e + "er"; }
    } catch (e2) { return e2; }
}
check("rethrow", rethrow() === "deeper");

// ----- dynamic typing: a variable may change its type -----
var dyn = 1;
dyn = "now a string";
dyn = [dyn];
check("dynamic-retype", dyn[0] === "now a string");

// ----- everything combined in one small pipeline (3-element data flow) -----
function transform(list) {
    var out = [];
    for (var i = 0; i < list.length; i++) {
        var n = list[i];
        try {
            if (n < 0) { throw "neg"; }
            out.push(n % 2 === 0 ? "e" + n : "o" + n);
        } catch (e) {
            out.push("x");
        }
    }
    return out.join("");
}
check("combined-pipeline", transform([1, 2, -3]) === "o1e2x");

function main() {
    println("features: " + checks + " checks, " + failures + " failures");
    return failures;
}
