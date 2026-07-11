// Self-checking test for the JavaScript interpreter and LLVM-IR compiler.
// It counts failed checks and returns that count; exit 0 means every check passed.
// This is REAL, dynamically typed JavaScript (the opposite of MetaJS): variables
// are not type pinned and may freely change type.

var failures = 0;

function check(cond) {
    if (!cond) { failures = failures + 1; }
}

// Arithmetic and precedence.
function testArithmetic() {
    check(1 + 2 * 3 === 7);
    check((1 + 2) * 3 === 9);
    check(10 - 4 - 3 === 3);
    check(7 % 3 === 1);
    check(2 * 3 % 4 === 2);
    check(7 / 2 === 3.5);
    check(-5 + 8 === 3);
}

// Comparison, loose vs strict equality (the core JS behaviour).
function testEquality() {
    check(1 == 1);
    check(1 === 1);
    check(1 == "1");        // loose: string coerces
    check((1 === "1") === false); // strict: different types
    check(0 == false);      // loose
    check((0 === false) === false);
    check(null == undefined);
    check((null === undefined) === false);
    check(2 < 3);
    check(3 <= 3);
    check(5 > 4);
    check(5 >= 5);
    check("abc" < "abd");
    check(1 != 2);
    check(1 !== "1");
}

// && and || short circuit and return the deciding operand.
function testLogical() {
    check((0 || 5) === 5);
    check((7 || 9) === 7);
    check((3 && 4) === 4);
    check((0 && 4) === 0);
    check((false || "yes") === "yes");
    check(!!"x" === true);
    check(!0 === true);
    var hit = 0;
    function bump() { hit = hit + 1; return true; }
    var r = false && bump();   // must NOT call bump
    check(hit === 0);
    r = true || bump();        // must NOT call bump
    check(hit === 0);
}

// Ternary, typeof and bitwise.
function testMisc() {
    check((5 > 3 ? "a" : "b") === "a");
    check(typeof 1 === "number");
    check(typeof "s" === "string");
    check(typeof true === "boolean");
    check(typeof undefined === "undefined");
    check(typeof null === "object");
    check(typeof testMisc === "function");
    check((6 & 3) === 2);
    check((6 | 1) === 7);
    check((6 ^ 3) === 5);
    check((1 << 4) === 16);
    check((32 >> 2) === 8);
    check((~0) === -1);
    check(0x1A === 26);
}

// Dynamic typing: one variable holds values of changing types (illegal in MetaJS).
function testDynamicTyping() {
    var v = 10;
    check(typeof v === "number");
    v = "text";
    check(typeof v === "string");
    v = [1, 2];
    check(typeof v === "object");
    v = function() { return 1; };
    check(typeof v === "function");
    check(v() === 1);
}

// Strings.
function testStrings() {
    var s = "Hello";
    check(s.length === 5);
    check(s.charAt(1) === "e");
    check(s.charCodeAt(0) === 72);
    check(s.indexOf("l") === 2);
    check(s.slice(1, 3) === "el");
    check(s.substring(0, 2) === "He");
    check(s.toUpperCase() === "HELLO");
    check(("a,b,c".split(",")).length === 3);
    check("x" + 1 + "y" === "x1y");
    check(("  hi  ".trim()) === "hi");
}

// Arrays and index/member assignment.
function testArrays() {
    var a = [1, 2, 3];
    check(a.length === 3);
    check(a[0] === 1);
    a.push(4);
    check(a.length === 4);
    check(a[3] === 4);
    check(a.pop() === 4);
    check(a.indexOf(2) === 1);
    check((a.join("-")) === "1-2-3");
    a[1] = 20;
    check(a[1] === 20);
    var b = a.concat([9]);
    check(b.length === 4);
    check((a.slice(1, 2)).length === 1);
}

// Objects.
function testObjects() {
    var o = {name: "x", value: 5};
    check(o.name === "x");
    check(o["value"] === 5);
    o.value = 6;
    check(o.value === 6);
    o.added = 7;
    check(o.added === 7);
    var key = "name";
    check(o[key] === "x");
    var nested = {inner: {n: 42}};
    check(nested.inner.n === 42);
}

// Control flow: if/else, while, do-while, for, break, continue.
function testControlFlow() {
    var sum = 0;
    var i;
    for (i = 1; i <= 5; i++) { sum = sum + i; }
    check(sum === 15);

    var w = 0;
    var k = 0;
    while (k < 4) { w = w + k; k++; }
    check(w === 6);

    var d = 0;
    var j = 0;
    do { d = d + 1; j++; } while (j < 3);
    check(d === 3);

    var col = 0;
    for (i = 0; i < 10; i++) {
        if (i === 5) { break; }
        if (i % 2 === 0) { continue; }
        col = col + i;
    }
    check(col === 4); // 1 + 3

    // else-if chain
    var grade = "?";
    var score = 75;
    if (score >= 90) { grade = "A"; }
    else if (score >= 70) { grade = "B"; }
    else { grade = "C"; }
    check(grade === "B");
}

// switch with fallthrough and default.
function testSwitch() {
    function classify(n) {
        var out = "";
        switch (n) {
            case 1:
                out = "one";
                break;
            case 2:
            case 3:
                out = "two-or-three";
                break;
            default:
                out = "many";
        }
        return out;
    }
    check(classify(1) === "one");
    check(classify(2) === "two-or-three");
    check(classify(3) === "two-or-three");
    check(classify(9) === "many");
}

// Functions: closures, recursion, function expressions, arguments.
function makeAdder(x) {
    return function(y) { return x + y; };
}
function makeCounter() {
    var n = 0;
    return function() { n = n + 1; return n; };
}
function fact(n) {
    if (n <= 1) { return 1; }
    return n * fact(n - 1);
}
function isEven(n) { if (n === 0) { return true; } return isOdd(n - 1); }
function isOdd(n) { if (n === 0) { return false; } return isEven(n - 1); }
function sumArgs() {
    var total = 0;
    var i;
    for (i = 0; i < arguments.length; i++) { total = total + arguments[i]; }
    return total;
}
function testFunctions() {
    var add5 = makeAdder(5);
    check(add5(3) === 8);
    check(add5(10) === 15);
    var c = makeCounter();
    check(c() === 1);
    check(c() === 2);
    check(c() === 3);
    check(fact(5) === 120);
    check(isEven(10) === true);
    check(isOdd(7) === true);
    check(sumArgs(1, 2, 3, 4) === 10);
    var sq = function(z) { return z * z; };
    check(sq(6) === 36);
}

// Prefix / postfix ++/-- and compound assignment.
function testIncDec() {
    var i = 5;
    check(i++ === 5);
    check(i === 6);
    check(++i === 7);
    check(i-- === 7);
    check(--i === 5);
    var t = 10;
    t += 5; check(t === 15);
    t -= 3; check(t === 12);
    t *= 2; check(t === 24);
    t /= 4; check(t === 6);
    t %= 4; check(t === 2);
    var s = "a";
    s += "b"; check(s === "ab");
}

function main() {
    testArithmetic();
    testEquality();
    testLogical();
    testMisc();
    testDynamicTyping();
    testStrings();
    testArrays();
    testObjects();
    testControlFlow();
    testSwitch();
    testFunctions();
    testIncDec();
    return failures;
}
