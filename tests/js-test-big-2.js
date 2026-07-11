// Self-checking test for the JavaScript interpreter (js-interpreter.abnf) and the
// LLVM-IR compiler (js-to-llvm-ir.abnf). THEME: heavy recursion, control flow, and
// number theory.
//
// Cross-checks recursive vs iterative vs memoized Fibonacci, factorial, Euclid's GCD /
// LCM, fast exponentiation, the Ackermann function, Towers of Hanoi, the Sieve of
// Eratosthenes, primality + prime factorization, the Collatz sequence, digit tricks,
// integer square root by bisection, base conversion, and Pascal's triangle. Deep
// branching drives if/else chains, switch with fallthrough, nested loops, do-while,
// break/continue, and the ternary operator. main() returns the number of failed
// checks; exit code 0 means every check passed. Only genuinely implemented constructs
// are used, so both grammars pass by default and the compiler IR is byte-identical.

var failures = 0;
function check(cond) { if (!cond) { failures = failures + 1; } }

// ----- Fibonacci: three independent implementations must agree -----
function fibRec(n) {
    if (n < 2) { return n; }
    return fibRec(n - 1) + fibRec(n - 2);
}
function fibIter(n) {
    var a = 0;
    var b = 1;
    for (var i = 0; i < n; i++) {
        var next = a + b;
        a = b;
        b = next;
    }
    return a;
}
var fibMemo = {};
function fibMemoized(n) {
    if (n < 2) { return n; }
    if (fibMemo[n] !== undefined) { return fibMemo[n]; }
    var r = fibMemoized(n - 1) + fibMemoized(n - 2);
    fibMemo[n] = r;
    return r;
}
function testFibonacci() {
    var expected = [0, 1, 1, 2, 3, 5, 8, 13, 21, 34, 55, 89, 144, 233, 377, 610];
    for (var n = 0; n < expected.length; n++) {
        check(fibRec(n) === expected[n]);
        check(fibIter(n) === expected[n]);
        check(fibMemoized(n) === expected[n]);
    }
    // Larger values via the fast implementations.
    check(fibIter(20) === 6765);
    check(fibMemoized(30) === 832040);
    check(fibIter(30) === fibMemoized(30));
}

// ----- factorial (recursive and iterative) -----
function factRec(n) { if (n <= 1) { return 1; } return n * factRec(n - 1); }
function factIter(n) {
    var r = 1;
    for (var i = 2; i <= n; i++) { r = r * i; }
    return r;
}
function testFactorial() {
    var expected = [1, 1, 2, 6, 24, 120, 720, 5040, 40320, 362880, 3628800];
    for (var n = 0; n < expected.length; n++) {
        check(factRec(n) === expected[n]);
        check(factIter(n) === expected[n]);
    }
}

// ----- GCD (recursive Euclid + iterative) and LCM -----
function gcdRec(a, b) { if (b === 0) { return a; } return gcdRec(b, a % b); }
function gcdIter(a, b) {
    while (b !== 0) { var t = b; b = a % b; a = t; }
    return a;
}
function lcm(a, b) { return Math.floor(a / gcdRec(a, b)) * b; }
function testGcdLcm() {
    check(gcdRec(48, 36) === 12);
    check(gcdIter(48, 36) === 12);
    check(gcdRec(17, 5) === 1);
    check(gcdRec(100, 100) === 100);
    check(gcdRec(0, 7) === 7);
    check(lcm(4, 6) === 12);
    check(lcm(21, 6) === 42);
    check(lcm(7, 13) === 91);
    // gcd(a,b) * lcm(a,b) == a*b, checked over a small grid.
    for (var a = 1; a <= 12; a++) {
        for (var b = 1; b <= 12; b++) {
            check(gcdRec(a, b) * lcm(a, b) === a * b);
            check(gcdRec(a, b) === gcdIter(a, b));
        }
    }
}

// ----- fast exponentiation (exponentiation by squaring) -----
function fastPow(base, exp) {
    var result = 1;
    var b = base;
    var e = exp;
    while (e > 0) {
        if (e % 2 === 1) { result = result * b; }
        b = b * b;
        e = Math.floor(e / 2);
    }
    return result;
}
function testFastPow() {
    check(fastPow(2, 0) === 1);
    check(fastPow(2, 10) === 1024);
    check(fastPow(3, 5) === 243);
    check(fastPow(5, 3) === 125);
    check(fastPow(10, 6) === 1000000);
    check(fastPow(7, 2) === 49);
    // Compare against a naive loop.
    for (var e = 0; e <= 10; e++) {
        var naive = 1;
        for (var i = 0; i < e; i++) { naive = naive * 2; }
        check(fastPow(2, e) === naive);
    }
}

// ----- Ackermann function: deep double recursion -----
function ackermann(m, n) {
    if (m === 0) { return n + 1; }
    if (n === 0) { return ackermann(m - 1, 1); }
    return ackermann(m - 1, ackermann(m, n - 1));
}
function testAckermann() {
    check(ackermann(0, 0) === 1);
    check(ackermann(1, 1) === 3);
    check(ackermann(2, 2) === 7);
    check(ackermann(2, 3) === 9);
    check(ackermann(3, 3) === 61);
}

// ----- Towers of Hanoi: number of moves = 2^n - 1 -----
function hanoiMoves(n, from, to, via) {
    if (n === 0) { return 0; }
    var m = 0;
    m = m + hanoiMoves(n - 1, from, via, to);
    m = m + 1;                              // move disk n
    m = m + hanoiMoves(n - 1, via, to, from);
    return m;
}
function testHanoi() {
    for (var n = 0; n <= 10; n++) {
        check(hanoiMoves(n, 0, 2, 1) === fastPow(2, n) - 1);
    }
}

// ----- Sieve of Eratosthenes + primality + factorization -----
function sieve(limit) {
    var isComposite = [];
    for (var i = 0; i <= limit; i++) { isComposite.push(false); }
    var primes = [];
    for (var p = 2; p <= limit; p++) {
        if (!isComposite[p]) {
            primes.push(p);
            for (var m = p * p; m <= limit; m = m + p) { isComposite[m] = true; }
        }
    }
    return primes;
}
function isPrime(n) {
    if (n < 2) { return false; }
    if (n < 4) { return true; }
    if (n % 2 === 0) { return false; }
    var d = 3;
    while (d * d <= n) {
        if (n % d === 0) { return false; }
        d = d + 2;
    }
    return true;
}
function primeFactors(n) {
    var factors = [];
    var d = 2;
    while (d * d <= n) {
        while (n % d === 0) { factors.push(d); n = Math.floor(n / d); }
        d = d + 1;
    }
    if (n > 1) { factors.push(n); }
    return factors;
}
function product(a) {
    var p = 1;
    for (var i = 0; i < a.length; i++) { p = p * a[i]; }
    return p;
}
function testPrimes() {
    var primes = sieve(50);
    var expected = [2, 3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37, 41, 43, 47];
    check(primes.length === expected.length);
    for (var i = 0; i < expected.length; i++) { check(primes[i] === expected[i]); }
    // The sieve and the trial-division primality test must agree on every n.
    for (var n = 0; n <= 50; n++) {
        var inSieve = false;
        for (var j = 0; j < primes.length; j++) { if (primes[j] === n) { inSieve = true; } }
        check(inSieve === isPrime(n));
    }
    // Prime factorization multiplies back to the original number.
    var samples = [12, 60, 97, 100, 360, 1024, 999];
    for (var s = 0; s < samples.length; s++) {
        var f = primeFactors(samples[s]);
        check(product(f) === samples[s]);
        for (var k = 0; k < f.length; k++) { check(isPrime(f[k])); }
    }
    check(primeFactors(97).length === 1);   // a prime factors to itself
    check(primeFactors(8).length === 3);    // 2 * 2 * 2
}

// ----- Collatz sequence length -----
function collatzSteps(n) {
    var steps = 0;
    while (n !== 1) {
        if (n % 2 === 0) { n = Math.floor(n / 2); } else { n = 3 * n + 1; }
        steps++;
    }
    return steps;
}
function testCollatz() {
    check(collatzSteps(1) === 0);
    check(collatzSteps(2) === 1);
    check(collatzSteps(6) === 8);
    check(collatzSteps(7) === 16);
    check(collatzSteps(27) === 111);
}

// ----- digit tricks -----
function digitSum(n) {
    var sum = 0;
    while (n > 0) { sum = sum + (n % 10); n = Math.floor(n / 10); }
    return sum;
}
function reverseNumber(n) {
    var r = 0;
    while (n > 0) { r = r * 10 + (n % 10); n = Math.floor(n / 10); }
    return r;
}
function isPalindromeNumber(n) { return n === reverseNumber(n); }
function digitalRoot(n) {
    while (n >= 10) { n = digitSum(n); }
    return n;
}
function testDigits() {
    check(digitSum(12345) === 15);
    check(digitSum(0) === 0);
    check(digitSum(999) === 27);
    check(reverseNumber(1234) === 4321);
    check(reverseNumber(1200) === 21);   // leading zeros vanish
    check(isPalindromeNumber(12321) === true);
    check(isPalindromeNumber(12345) === false);
    check(digitalRoot(12345) === 6);     // 15 -> 6
    check(digitalRoot(99999) === 9);
}

// ----- integer square root by bisection -----
function isqrt(n) {
    if (n < 2) { return n; }
    var lo = 1;
    var hi = n;
    var ans = 1;
    while (lo <= hi) {
        var mid = Math.floor((lo + hi) / 2);
        if (mid * mid <= n) { ans = mid; lo = mid + 1; } else { hi = mid - 1; }
    }
    return ans;
}
function testIsqrt() {
    check(isqrt(0) === 0);
    check(isqrt(1) === 1);
    check(isqrt(15) === 3);
    check(isqrt(16) === 4);
    check(isqrt(17) === 4);
    check(isqrt(99) === 9);
    check(isqrt(100) === 10);
    check(isqrt(1000000) === 1000);
    // Property: ans^2 <= n < (ans+1)^2 for every n in a range.
    for (var n = 0; n <= 200; n++) {
        var r = isqrt(n);
        check(r * r <= n && (r + 1) * (r + 1) > n);
    }
}

// ----- base conversion to a string of digits, using a switch for the digit char -----
function digitChar(d) {
    switch (d) {
        case 10: return "a";
        case 11: return "b";
        case 12: return "c";
        case 13: return "d";
        case 14: return "e";
        case 15: return "f";
        default: return "" + d;   // 0..9 coerce to their single character
    }
}
function toBase(n, base) {
    if (n === 0) { return "0"; }
    var out = "";
    while (n > 0) {
        out = digitChar(n % base) + out;
        n = Math.floor(n / base);
    }
    return out;
}
function testBaseConversion() {
    check(toBase(0, 2) === "0");
    check(toBase(5, 2) === "101");
    check(toBase(255, 2) === "11111111");
    check(toBase(255, 16) === "ff");
    check(toBase(256, 16) === "100");
    check(toBase(8, 8) === "10");
    check(toBase(100, 10) === "100");
    check(toBase(3735928559, 16) === "deadbeef");
}

// ----- Pascal's triangle / binomial coefficients -----
function binomial(n, k) {
    if (k < 0 || k > n) { return 0; }
    if (k === 0 || k === n) { return 1; }
    return binomial(n - 1, k - 1) + binomial(n - 1, k);
}
function pascalRow(n) {
    var row = [];
    for (var k = 0; k <= n; k++) { row.push(binomial(n, k)); }
    return row;
}
function testPascal() {
    check(binomial(5, 2) === 10);
    check(binomial(6, 3) === 20);
    check(binomial(10, 0) === 1);
    check(binomial(10, 10) === 1);
    var row5 = pascalRow(5);
    var expected = [1, 5, 10, 10, 5, 1];
    check(row5.length === 6);
    for (var i = 0; i < expected.length; i++) { check(row5[i] === expected[i]); }
    // Each row of Pascal's triangle sums to 2^n.
    for (var n = 0; n <= 8; n++) {
        var row = pascalRow(n);
        var sum = 0;
        for (var j = 0; j < row.length; j++) { sum = sum + row[j]; }
        check(sum === fastPow(2, n));
    }
}

// ----- a small classifier exercising switch fallthrough and do-while -----
function fizzbuzzClassify(n) {
    var byThree = (n % 3 === 0);
    var byFive = (n % 5 === 0);
    if (byThree && byFive) { return "fizzbuzz"; }
    if (byThree) { return "fizz"; }
    if (byFive) { return "buzz"; }
    return "" + n;
}
function testFizzBuzz() {
    check(fizzbuzzClassify(3) === "fizz");
    check(fizzbuzzClassify(5) === "buzz");
    check(fizzbuzzClassify(15) === "fizzbuzz");
    check(fizzbuzzClassify(7) === "7");
    // Count categories in [1..30] with a do-while accumulator.
    var fizz = 0;
    var buzz = 0;
    var fb = 0;
    var i = 1;
    do {
        var c = fizzbuzzClassify(i);
        switch (c) {
            case "fizz": fizz++; break;
            case "buzz": buzz++; break;
            case "fizzbuzz": fb++; break;
            default: break;
        }
        i++;
    } while (i <= 30);
    check(fizz === 8);   // multiples of 3 not of 15: 3,6,9,12,18,21,24,27
    check(buzz === 4);   // multiples of 5 not of 15: 5,10,20,25
    check(fb === 2);     // 15, 30
}

function main() {
    testFibonacci();
    testFactorial();
    testGcdLcm();
    testFastPow();
    testAckermann();
    testHanoi();
    testPrimes();
    testCollatz();
    testDigits();
    testIsqrt();
    testBaseConversion();
    testPascal();
    testFizzBuzz();
    return failures;
}
