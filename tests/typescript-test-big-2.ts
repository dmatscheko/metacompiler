// Self-checking TypeScript test #big-2: RECURSION + CONTROL FLOW + NUMBER THEORY.
//
// Themes: recursion in every shape (linear, tree, mutual, accumulator), classic
// number theory (gcd / lcm, fast modular exponentiation, sieve of Eratosthenes,
// Collatz, digit sums, Ackermann), deep and branchy control flow (nested loops, a
// switch-driven run, do/while), and a recursive-descent arithmetic expression
// evaluator built from a small hand-written parser class. Every result is checked
// against a known value; main() returns the failure count (0 == all engines agree).

let failures: number = 0;

function check(cond: boolean, _label: string): void {
    if (!cond) { failures = failures + 1; }
}

// ---- linear & tree recursion ----

function factorial(n: number): number {
    if (n <= 1) { return 1; }
    return n * factorial(n - 1);
}

// Naive tree-recursive Fibonacci.
function fibRec(n: number): number {
    if (n < 2) { return n; }
    return fibRec(n - 1) + fibRec(n - 2);
}

// Iterative Fibonacci (must agree with fibRec).
function fibIter(n: number): number {
    let a: number = 0;
    let b: number = 1;
    for (let i: number = 0; i < n; i++) {
        const t: number = a + b;
        a = b;
        b = t;
    }
    return a;
}

// Tail-style accumulator recursion.
function sumTo(n: number, acc: number): number {
    if (n === 0) { return acc; }
    return sumTo(n - 1, acc + n);
}

// Recursive digit sum.
function digitSum(n: number): number {
    if (n < 10) { return n; }
    return (n % 10) + digitSum(Math.floor(n / 10));
}

function testRecursion(): void {
    check(factorial(0) === 1, "fact0");
    check(factorial(5) === 120, "fact5");
    check(factorial(10) === 3628800, "fact10");

    check(fibIter(0) === 0, "fib0");
    check(fibIter(1) === 1, "fib1");
    check(fibIter(10) === 55, "fib10");
    check(fibIter(20) === 6765, "fib20");
    // The recursive and iterative Fibonacci must agree across a range.
    for (let n: number = 0; n <= 18; n++) {
        check(fibRec(n) === fibIter(n), "fib-agree-" + n);
    }

    check(sumTo(100, 0) === 5050, "sum100");
    check(sumTo(10, 0) === 55, "sum10");

    check(digitSum(0) === 0, "digitsum0");
    check(digitSum(9) === 9, "digitsum9");
    check(digitSum(12345) === 15, "digitsum12345");
    check(digitSum(999999) === 54, "digitsum999999");
}

// ---- mutual recursion ----

function isEven(n: number): boolean {
    if (n === 0) { return true; }
    return isOdd(n - 1);
}

function isOdd(n: number): boolean {
    if (n === 0) { return false; }
    return isEven(n - 1);
}

function testMutualRecursion(): void {
    check(isEven(0), "even0");
    check(isOdd(1), "odd1");
    check(isEven(10), "even10");
    check(!isEven(7), "not-even7");
    check(isOdd(7), "odd7");
    for (let i: number = 0; i <= 20; i++) {
        check(isEven(i) === (i % 2 === 0), "even-agree-" + i);
        check(isOdd(i) !== isEven(i), "parity-exclusive-" + i);
    }
}

// ---- number theory ----

function gcd(a: number, b: number): number {
    while (b !== 0) {
        const t: number = b;
        b = a % b;
        a = t;
    }
    return a;
}

function lcm(a: number, b: number): number {
    return Math.floor(a / gcd(a, b)) * b;
}

// Fast modular exponentiation: base^exp mod m, by repeated squaring.
function powMod(base: number, exp: number, m: number): number {
    let result: number = 1;
    let b: number = base % m;
    let e: number = exp;
    while (e > 0) {
        if (e % 2 === 1) { result = (result * b) % m; }
        b = (b * b) % m;
        e = Math.floor(e / 2);
    }
    return result;
}

// Sieve of Eratosthenes: all primes < n.
function primesBelow(n: number): number[] {
    const isComposite: boolean[] = [];
    for (let i: number = 0; i < n; i++) { isComposite.push(false); }
    const primes: number[] = [];
    for (let p: number = 2; p < n; p++) {
        if (!isComposite[p]) {
            primes.push(p);
            for (let multiple: number = p * p; multiple < n; multiple = multiple + p) {
                isComposite[multiple] = true;
            }
        }
    }
    return primes;
}

// Collatz step count until reaching 1.
function collatzSteps(n: number): number {
    let steps: number = 0;
    let x: number = n;
    while (x !== 1) {
        if (x % 2 === 0) { x = Math.floor(x / 2); } else { x = 3 * x + 1; }
        steps = steps + 1;
    }
    return steps;
}

function arraysEqual(a: number[], b: number[]): boolean {
    if (a.length !== b.length) { return false; }
    for (let i: number = 0; i < a.length; i++) {
        if (a[i] !== b[i]) { return false; }
    }
    return true;
}

function testNumberTheory(): void {
    check(gcd(48, 18) === 6, "gcd-48-18");
    check(gcd(17, 5) === 1, "gcd-coprime");
    check(gcd(100, 100) === 100, "gcd-equal");
    check(gcd(0, 7) === 7, "gcd-zero");
    check(lcm(4, 6) === 12, "lcm-4-6");
    check(lcm(21, 6) === 42, "lcm-21-6");

    check(powMod(2, 10, 1000) === 24, "powmod-1024");
    check(powMod(3, 5, 7) === 5, "powmod-3^5%7");   // 243 % 7 = 5
    check(powMod(7, 0, 13) === 1, "powmod-exp0");
    check(powMod(2, 20, 1000000) === 48576, "powmod-big");   // 1048576 % 1e6

    const primes: number[] = primesBelow(30);
    check(arraysEqual(primes, [2, 3, 5, 7, 11, 13, 17, 19, 23, 29]), "primes-below-30");
    check(primesBelow(2).length === 0, "primes-below-2");
    check(primesBelow(3).length === 1, "primes-below-3");
    check(primesBelow(100).length === 25, "primes-below-100-count");

    check(collatzSteps(1) === 0, "collatz-1");
    check(collatzSteps(6) === 8, "collatz-6");
    check(collatzSteps(27) === 111, "collatz-27");
}

// ---- Ackermann (deep tree recursion) ----
function ackermann(m: number, n: number): number {
    if (m === 0) { return n + 1; }
    if (n === 0) { return ackermann(m - 1, 1); }
    return ackermann(m - 1, ackermann(m, n - 1));
}

// ---- Tower of Hanoi: count moves, and record them ----
function hanoiCount(n: number): number {
    if (n === 0) { return 0; }
    return 2 * hanoiCount(n - 1) + 1;
}

function hanoiMoves(n: number, from: number, to: number, via: number, moves: number[]): void {
    if (n === 0) { return; }
    hanoiMoves(n - 1, from, via, to, moves);
    moves.push(from * 10 + to);      // encode a move "from->to" as one number
    hanoiMoves(n - 1, via, to, from, moves);
}

function testDeepRecursion(): void {
    check(ackermann(0, 0) === 1, "ack-0-0");
    check(ackermann(2, 3) === 9, "ack-2-3");
    check(ackermann(3, 3) === 61, "ack-3-3");
    check(ackermann(3, 4) === 125, "ack-3-4");

    // hanoiCount(n) must equal 2^n - 1.
    for (let n: number = 0; n <= 10; n++) {
        check(hanoiCount(n) === (1 << n) - 1, "hanoi-count-" + n);
    }
    const moves: number[] = [];
    hanoiMoves(3, 1, 3, 2, moves);
    check(moves.length === 7, "hanoi-moves-len");
    // The classic optimal solution for 3 disks (peg 1 -> 3, spare 2).
    check(arraysEqual(moves, [13, 12, 32, 13, 21, 23, 13]), "hanoi-moves-seq");
}

// ---- binomial coefficients via recursion (Pascal's triangle) ----
function binomial(n: number, k: number): number {
    if (k === 0 || k === n) { return 1; }
    if (k < 0 || k > n) { return 0; }
    return binomial(n - 1, k - 1) + binomial(n - 1, k);
}

function testPascal(): void {
    check(binomial(0, 0) === 1, "binom-0-0");
    check(binomial(5, 2) === 10, "binom-5-2");
    check(binomial(6, 3) === 20, "binom-6-3");
    check(binomial(10, 5) === 252, "binom-10-5");
    // Row n sums to 2^n.
    for (let n: number = 0; n <= 8; n++) {
        let rowSum: number = 0;
        for (let k: number = 0; k <= n; k++) { rowSum = rowSum + binomial(n, k); }
        check(rowSum === (1 << n), "pascal-rowsum-" + n);
    }
    // Symmetry: C(n,k) == C(n,n-k).
    check(binomial(9, 2) === binomial(9, 7), "binom-symmetry");
}

// ---- deep nested loops + a switch-driven classifier ----

// Count Pythagorean triples (a<b<c, a^2+b^2==c^2) with c <= limit.
function countPythagorean(limit: number): number {
    let count: number = 0;
    for (let a: number = 1; a <= limit; a++) {
        for (let b: number = a + 1; b <= limit; b++) {
            for (let c: number = b + 1; c <= limit; c++) {
                if (a * a + b * b === c * c) { count = count + 1; }
            }
        }
    }
    return count;
}

// FizzBuzz collapsed to a numeric code via a switch on (n%3, n%5).
function fizzCode(n: number): number {
    const key: number = (n % 3 === 0 ? 1 : 0) + (n % 5 === 0 ? 2 : 0);
    let out: number = 0;
    switch (key) {
        case 3: out = 15; break;   // FizzBuzz
        case 1: out = 3; break;    // Fizz
        case 2: out = 5; break;    // Buzz
        default: out = n;          // the number itself
    }
    return out;
}

function testLoopsAndSwitch(): void {
    check(countPythagorean(20) === 6, "pythagorean-20");
    check(countPythagorean(5) === 1, "pythagorean-5");   // just (3,4,5)
    check(countPythagorean(4) === 0, "pythagorean-4");

    check(fizzCode(1) === 1, "fizz-1");
    check(fizzCode(3) === 3, "fizz-3");
    check(fizzCode(5) === 5, "fizz-5");
    check(fizzCode(15) === 15, "fizz-15");
    check(fizzCode(30) === 15, "fizz-30");
    check(fizzCode(9) === 3, "fizz-9");
    check(fizzCode(20) === 5, "fizz-20");

    // do/while accumulation.
    let product: number = 1;
    let i: number = 1;
    do {
        product = product * i;
        i++;
    } while (i <= 5);
    check(product === 120, "dowhile-product");
}

// ---- recursive-descent arithmetic evaluator ----
// Grammar:  expr := term (('+'|'-') term)*
//           term := factor (('*'|'/') factor)*
//           factor := number | '(' expr ')' | '-' factor
class Parser {
    private src: string;
    private pos: number;

    constructor(src: string) {
        this.src = src;
        this.pos = 0;
    }

    private peek(): string {
        if (this.pos >= this.src.length) { return ""; }
        return this.src.charAt(this.pos);
    }

    private advance(): void {
        this.pos = this.pos + 1;
    }

    private skipSpaces(): void {
        while (this.peek() === " ") { this.advance(); }
    }

    private isDigit(ch: string): boolean {
        if (ch === "") { return false; }
        const code: number = ch.charCodeAt(0);
        return code >= 48 && code <= 57;
    }

    private parseNumber(): number {
        let value: number = 0;
        while (this.isDigit(this.peek())) {
            value = value * 10 + (this.peek().charCodeAt(0) - 48);
            this.advance();
        }
        return value;
    }

    private parseFactor(): number {
        this.skipSpaces();
        const ch: string = this.peek();
        if (ch === "(") {
            this.advance();
            const v: number = this.parseExpr();
            this.skipSpaces();
            if (this.peek() === ")") { this.advance(); }
            return v;
        }
        if (ch === "-") {
            this.advance();
            return -this.parseFactor();
        }
        return this.parseNumber();
    }

    private parseTerm(): number {
        let v: number = this.parseFactor();
        while (true) {
            this.skipSpaces();
            const op: string = this.peek();
            if (op === "*") { this.advance(); v = v * this.parseFactor(); }
            else if (op === "/") { this.advance(); v = v / this.parseFactor(); }
            else { break; }
        }
        return v;
    }

    parseExpr(): number {
        let v: number = this.parseTerm();
        while (true) {
            this.skipSpaces();
            const op: string = this.peek();
            if (op === "+") { this.advance(); v = v + this.parseTerm(); }
            else if (op === "-") { this.advance(); v = v - this.parseTerm(); }
            else { break; }
        }
        return v;
    }
}

function evalExpr(src: string): number {
    const p: Parser = new Parser(src);
    return p.parseExpr();
}

function testExpressionEvaluator(): void {
    check(evalExpr("1+2*3") === 7, "eval-precedence");
    check(evalExpr("(1+2)*3") === 9, "eval-parens");
    check(evalExpr("10-4-3") === 3, "eval-left-assoc");
    check(evalExpr("2*3+4*5") === 26, "eval-mixed");
    check(evalExpr("100/4/5") === 5, "eval-div");
    check(evalExpr("-(3+4)") === -7, "eval-unary-minus");
    check(evalExpr("((2))") === 2, "eval-nested-parens");
    check(evalExpr("2+3*4-10/2") === 9, "eval-combined");
    check(evalExpr("  7 - 2 * 2  ") === 3, "eval-spaces");
    check(evalExpr("42") === 42, "eval-single-number");
    check(evalExpr("2*(3+(4-1))") === 12, "eval-deep");
    check(evalExpr("10/4") === 2.5, "eval-fractional");
}

function main(): number {
    testRecursion();
    testMutualRecursion();
    testNumberTheory();
    testDeepRecursion();
    testPascal();
    testLoopsAndSwitch();
    testExpressionEvaluator();
    return failures;
}
