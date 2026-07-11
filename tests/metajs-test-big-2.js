/* MetaJS big test 2 - Recursion, control flow and a tiny interpreter.
 *
 * Heavy recursion (factorial, fast power, Ackermann, Towers of Hanoi, Collatz,
 * Pascal's triangle, mutual recursion, recursive tree walks), deep branching
 * (triangle classification, leap years, letter grades via switch), and a
 * complete recursive-descent evaluator for integer arithmetic with +, -, *, /
 * and parentheses. Two finite state machines round it out: a mod-3 DFA over
 * binary strings and a balanced-parenthesis checker.
 *
 * Self checking: main() returns the number of failed checks, so the run exits 0
 * exactly when all is correct, identically under both engines and both the
 * interpreter and the compiler. **/

var failures = 0;
var checks = 0;

function check(name, got, want) {
    checks = checks + 1;
    if (got !== want) {
        println("FAIL " + name + ": got " + got + " want " + want);
        failures = failures + 1;
    }
}

// ----- plain recursion -----

function factorial(n) {
    if (n <= 1) { return 1; }
    return n * factorial(n - 1);
}

function power(base, exp) {
    if (exp == 0) { return 1; }
    var half = power(base, Math.floor(exp / 2));
    if (exp % 2 == 0) { return half * half; }
    return half * half * base;
}

function ackermann(m, n) {
    if (m == 0) { return n + 1; }
    if (n == 0) { return ackermann(m - 1, 1); }
    return ackermann(m - 1, ackermann(m, n - 1));
}

function fibRec(n) {
    if (n < 2) { return n; }
    return fibRec(n - 1) + fibRec(n - 2);
}

function choose(n, k) {
    if (k == 0 || k == n) { return 1; }
    return choose(n - 1, k - 1) + choose(n - 1, k);
}

function sumDigits(n) {
    if (n < 10) { return n; }
    return (n % 10) + sumDigits(Math.floor(n / 10));
}

function gcdRec(a, b) {
    if (b == 0) { return a; }
    return gcdRec(b, a % b);
}

function reverseStr(s) {
    if (s.length <= 1) { return s; }
    return reverseStr(s.substring(1)) + s.charAt(0);
}

// ----- mutual recursion -----

function isEven(n) { return n == 0 ? true : isOdd(n - 1); }
function isOdd(n) { return n == 0 ? false : isEven(n - 1); }

// ----- Towers of Hanoi: record every move -----

function hanoi(n, from, to, via, moves) {
    if (n == 0) { return; }
    hanoi(n - 1, from, via, to, moves);
    moves.push(from + to);
    hanoi(n - 1, via, to, from, moves);
}

function hanoiMoves(n) {
    var moves = [];
    hanoi(n, "A", "C", "B", moves);
    return moves;
}

// ----- iterative control flow -----

function collatzLength(n) {
    var steps = 0;
    var v = n;
    while (v != 1) {
        if (v % 2 == 0) { v = v / 2; }
        else { v = 3 * v + 1; }
        steps++;
    }
    return steps;
}

function reverseNumber(n) {
    var r = 0;
    var v = n;
    while (v > 0) {
        r = r * 10 + (v % 10);
        v = Math.floor(v / 10);
    }
    return r;
}

function digitalRoot(n) {
    var v = n;
    while (v >= 10) { v = sumDigits(v); }
    return v;
}

function isPalindromeNumber(n) {
    return n == reverseNumber(n);
}

// ----- recursive walk over a nested array tree (numbers or arrays) -----

function treeSum(node) {
    if (typeof node == "object") {
        var s = 0;
        for (var i = 0; i < node.length; i++) { s += treeSum(node[i]); }
        return s;
    }
    return node;
}

function treeDepth(node) {
    if (typeof node != "object") { return 0; }
    var best = 0;
    for (var i = 0; i < node.length; i++) {
        var d = treeDepth(node[i]);
        if (d > best) { best = d; }
    }
    return 1 + best;
}

function treeCount(node) {
    if (typeof node != "object") { return 1; }
    var c = 0;
    for (var i = 0; i < node.length; i++) { c += treeCount(node[i]); }
    return c;
}

// ----- deep branching -----

function classifyTriangle(a, b, c) {
    if (a + b <= c || a + c <= b || b + c <= a) { return "invalid"; }
    if (a == b && b == c) { return "equilateral"; }
    if (a == b || b == c || a == c) { return "isosceles"; }
    return "scalene";
}

function isLeap(y) {
    if (y % 400 == 0) { return true; }
    if (y % 100 == 0) { return false; }
    return y % 4 == 0;
}

function grade(score) {
    var bucket = Math.floor(score / 10);
    var g = "F";
    switch (bucket) {
    case 10:
    case 9:
        g = "A";
        break;
    case 8:
        g = "B";
        break;
    case 7:
        g = "C";
        break;
    case 6:
        g = "D";
        break;
    default:
        g = "F";
    }
    return g;
}

function sign(x) {
    if (x > 0) { return 1; }
    if (x < 0) { return -1; }
    return 0;
}

// ----- recursive-descent arithmetic evaluator -----
// Grammar: Expr = Term { ("+"|"-") Term } ; Term = Factor { ("*"|"/") Factor } ;
// Factor = number | "(" Expr ")".  Division floors toward zero of the quotient.

function makeCtx(s) { return {s: s, i: 0}; }

function peekCh(ctx) {
    if (ctx.i < ctx.s.length) { return ctx.s.charAt(ctx.i); }
    return "";
}

function skipSpaces(ctx) {
    while (peekCh(ctx) == " ") { ctx.i++; }
}

function isDigit(c) {
    return c >= "0" && c <= "9";
}

function parseNumber(ctx) {
    skipSpaces(ctx);
    var n = 0;
    while (isDigit(peekCh(ctx))) {
        n = n * 10 + (peekCh(ctx).charCodeAt(0) - 48);
        ctx.i++;
    }
    return n;
}

function parseFactor(ctx) {
    skipSpaces(ctx);
    if (peekCh(ctx) == "(") {
        ctx.i++;
        var v = parseExpr(ctx);
        skipSpaces(ctx);
        ctx.i++;            // consume ')'
        return v;
    }
    return parseNumber(ctx);
}

function parseTerm(ctx) {
    var v = parseFactor(ctx);
    skipSpaces(ctx);
    while (peekCh(ctx) == "*" || peekCh(ctx) == "/") {
        var op = peekCh(ctx);
        ctx.i++;
        var rhs = parseFactor(ctx);
        if (op == "*") { v = v * rhs; }
        else { v = Math.floor(v / rhs); }
        skipSpaces(ctx);
    }
    return v;
}

function parseExpr(ctx) {
    var v = parseTerm(ctx);
    skipSpaces(ctx);
    while (peekCh(ctx) == "+" || peekCh(ctx) == "-") {
        var op = peekCh(ctx);
        ctx.i++;
        var rhs = parseTerm(ctx);
        if (op == "+") { v = v + rhs; }
        else { v = v - rhs; }
        skipSpaces(ctx);
    }
    return v;
}

function evalExpr(s) {
    return parseExpr(makeCtx(s));
}

// ----- finite state machines -----

function divisibleBy3(bits) {
    var state = 0;
    for (var i = 0; i < bits.length; i++) {
        var bit = bits.charAt(i) == "1" ? 1 : 0;
        switch (state) {
        case 0:
            state = bit == 0 ? 0 : 1;
            break;
        case 1:
            state = bit == 0 ? 2 : 0;
            break;
        case 2:
            state = bit == 0 ? 1 : 2;
            break;
        }
    }
    return state == 0;
}

function balancedParens(s) {
    var depth = 0;
    for (var i = 0; i < s.length; i++) {
        var c = s.charAt(i);
        if (c == "(") { depth++; }
        else if (c == ")") {
            depth--;
            if (depth < 0) { return false; }
        }
    }
    return depth == 0;
}

function main() {

    // ----- factorial and power -----
    check("factorial 5", factorial(5), 120);
    check("factorial 10", factorial(10), 3628800);
    check("factorial 0", factorial(0), 1);
    check("power 2^10", power(2, 10), 1024);
    check("power 3^4", power(3, 4), 81);
    check("power 2^16", power(2, 16), 65536);
    check("power n^0", power(7, 0), 1);
    check("power 5^3", power(5, 3), 125);

    // ----- Ackermann -----
    check("ack 0 0", ackermann(0, 0), 1);
    check("ack 1 5", ackermann(1, 5), 7);
    check("ack 2 3", ackermann(2, 3), 9);
    check("ack 2 4", ackermann(2, 4), 11);
    check("ack 3 3", ackermann(3, 3), 61);
    check("ack 3 4", ackermann(3, 4), 125);

    // ----- Fibonacci and binomials -----
    check("fibRec 15", fibRec(15), 610);
    check("fibRec 20", fibRec(20), 6765);
    check("choose 5 2", choose(5, 2), 10);
    check("choose 6 3", choose(6, 3), 20);
    check("choose 10 5", choose(10, 5), 252);
    check("choose n 0", choose(9, 0), 1);
    check("choose n n", choose(9, 9), 1);

    // ----- digit recursion -----
    check("sumDigits", sumDigits(12345), 15);
    check("sumDigits big", sumDigits(999999), 54);
    check("gcdRec", gcdRec(1071, 462), 21);
    check("gcdRec coprime", gcdRec(13, 8), 1);
    check("reverse string", reverseStr("hello"), "olleh");
    check("reverse palindrome", reverseStr("radar"), "radar");
    check("reverse single", reverseStr("x"), "x");

    // ----- mutual recursion -----
    check("isEven 10", isEven(10), true);
    check("isOdd 7", isOdd(7), true);
    check("isEven 7", isEven(7), false);

    // ----- Towers of Hanoi -----
    var m3 = hanoiMoves(3);
    check("hanoi 3 count", m3.length, 7);
    check("hanoi 3 sequence", m3.join(","), "AC,AB,CB,AC,BA,BC,AC");
    check("hanoi 4 count", hanoiMoves(4).length, 15);
    check("hanoi 5 count", hanoiMoves(5).length, 31);
    check("hanoi 1", hanoiMoves(1).join(","), "AC");
    check("hanoi 0", hanoiMoves(0).length, 0);

    // ----- iterative flows -----
    check("collatz 27", collatzLength(27), 111);
    check("collatz 6", collatzLength(6), 8);
    check("collatz 1", collatzLength(1), 0);
    check("reverse number", reverseNumber(12345), 54321);
    check("reverse trailing zero", reverseNumber(1200), 21);
    check("digital root", digitalRoot(12345), 6);
    check("digital root 9x", digitalRoot(999999), 9);
    check("palindrome number yes", isPalindromeNumber(12321), true);
    check("palindrome number no", isPalindromeNumber(12345), false);

    // ----- nested tree walks -----
    var tree = [1, [2, 3], [[4], 5], 6];
    check("tree sum", treeSum(tree), 21);
    check("tree leaf count", treeCount(tree), 6);
    check("tree depth", treeDepth(tree), 3);
    check("tree sum deep", treeSum([[[[10]]]]), 10);
    check("tree scalar", treeSum(42), 42);

    // ----- deep branching -----
    check("triangle equilateral", classifyTriangle(3, 3, 3), "equilateral");
    check("triangle isosceles", classifyTriangle(5, 5, 8), "isosceles");
    check("triangle scalene", classifyTriangle(4, 5, 6), "scalene");
    check("triangle invalid", classifyTriangle(1, 2, 10), "invalid");
    check("leap 2000", isLeap(2000), true);
    check("leap 1900", isLeap(1900), false);
    check("leap 2024", isLeap(2024), true);
    check("leap 2023", isLeap(2023), false);
    check("grade A", grade(95), "A");
    check("grade A hundred", grade(100), "A");
    check("grade B", grade(83), "B");
    check("grade C", grade(70), "C");
    check("grade D", grade(65), "D");
    check("grade F", grade(40), "F");
    check("sign pos", sign(9), 1);
    check("sign neg", sign(-3), -1);
    check("sign zero", sign(0), 0);

    // ----- recursive-descent evaluator -----
    check("eval add mul", evalExpr("2+3*4"), 14);
    check("eval parens", evalExpr("(2+3)*4"), 20);
    check("eval left assoc", evalExpr("10-2-3"), 5);
    check("eval div chain", evalExpr("100/4/5"), 5);
    check("eval nested", evalExpr("2*(3+(4-1))"), 12);
    check("eval double parens", evalExpr("((1+2)*(3+4))"), 21);
    check("eval mixed", evalExpr("7*8-6/2"), 53);
    check("eval sum", evalExpr("1+2+3+4+5"), 15);
    check("eval floor div", evalExpr("20/3"), 6);
    check("eval spaces", evalExpr("  12  +  8  "), 20);
    check("eval single", evalExpr("42"), 42);
    check("eval deep nest", evalExpr("((((5))))"), 5);

    // ----- finite state machines -----
    check("div3 zero", divisibleBy3("0"), true);
    check("div3 three", divisibleBy3("11"), true);
    check("div3 six", divisibleBy3("110"), true);
    check("div3 nine", divisibleBy3("1001"), true);
    check("div3 five no", divisibleBy3("101"), false);
    check("div3 seven no", divisibleBy3("111"), false);
    check("div3 empty", divisibleBy3(""), true);
    check("balanced simple", balancedParens("(())"), true);
    check("balanced nested", balancedParens("(()(()))"), true);
    check("balanced open", balancedParens("(()"), false);
    check("balanced close first", balancedParens(")("), false);
    check("balanced empty", balancedParens(""), true);
    check("balanced text", balancedParens("a(b)c(d(e)f)"), true);

    printf("%c%c %d checks\n", 79, 75, checks);
    if (failures == 0) { println("MetaJS big test 2 (recursion and control flow) passed"); }
    return failures;
}
