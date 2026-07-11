<?php
// PHP subset BIG test 2 -- Recursion & control flow.
//
// Theme: number-theory and combinatorial routines written both recursively and
// iteratively, cross-checked against each other, plus a small event-driven state
// machine. Deep branching, nested loops, break/continue and mutual recursion.
// The program counts failures in $fails and ends with exit($fails); the
// interpreter and the LLVM-IR compiler run the same file and must agree.

$fails = 0;

function check($name, $got, $want) {
    global $fails;
    if ($got !== $want) {
        echo "FAIL " . $name . "\n";
        $fails = $fails + 1;
    }
}

// ----- factorial: recursive and iterative -----
function factRec($n) {
    if ($n <= 1) {
        return 1;
    }
    return $n * factRec($n - 1);
}

function factIter($n) {
    $acc = 1;
    for ($i = 2; $i <= $n; $i++) {
        $acc *= $i;
    }
    return $acc;
}

check("fact rec 5", factRec(5), 120);
check("fact rec 0", factRec(0), 1);
check("fact iter 6", factIter(6), 720);
check("fact agree", factRec(7), factIter(7));

// ----- Fibonacci: recursive, iterative, and via fast doubling recursion -----
function fibRec($n) {
    if ($n < 2) {
        return $n;
    }
    return fibRec($n - 1) + fibRec($n - 2);
}

function fibIter($n) {
    $a = 0;
    $b = 1;
    for ($i = 0; $i < $n; $i++) {
        $t = $a + $b;
        $a = $b;
        $b = $t;
    }
    return $a;
}

check("fib rec 10", fibRec(10), 55);
check("fib iter 20", fibIter(20), 6765);
for ($k = 0; $k <= 15; $k++) {
    check("fib agree " . $k, fibRec($k), fibIter($k));
}

// ----- greatest common divisor / least common multiple -----
function gcdRec($a, $b) {
    if ($b === 0) {
        return $a;
    }
    return gcdRec($b, $a % $b);
}

function gcdIter($a, $b) {
    while ($b !== 0) {
        $t = $a % $b;
        $a = $b;
        $b = $t;
    }
    return $a;
}

function lcm($a, $b) {
    return intdiv($a * $b, gcdRec($a, $b));
}

check("gcd rec", gcdRec(48, 36), 12);
check("gcd iter", gcdIter(1071, 462), 21);
check("gcd agree", gcdRec(270, 192), gcdIter(270, 192));
check("gcd coprime", gcdRec(17, 5), 1);
check("lcm", lcm(4, 6), 12);
check("lcm big", lcm(21, 6), 42);

// ----- fast exponentiation (recursive, exponent by squaring) -----
function power($base, $exp) {
    if ($exp === 0) {
        return 1;
    }
    $half = power($base, intdiv($exp, 2));
    if ($exp % 2 === 0) {
        return $half * $half;
    }
    return $half * $half * $base;
}

check("pow 2^10", power(2, 10), 1024);
check("pow 3^5", power(3, 5), 243);
check("pow 5^0", power(5, 0), 1);
check("pow 7^3", power(7, 3), 343);
check("pow 10^4", power(10, 4), 10000);

// ----- Ackermann function (deep nested recursion, kept small) -----
function ackermann($m, $n) {
    if ($m === 0) {
        return $n + 1;
    }
    if ($n === 0) {
        return ackermann($m - 1, 1);
    }
    return ackermann($m - 1, ackermann($m, $n - 1));
}

check("ack 0 0", ackermann(0, 0), 1);
check("ack 2 3", ackermann(2, 3), 9);
check("ack 3 3", ackermann(3, 3), 61);

// ----- Collatz sequence length -----
function collatzLength($n) {
    $steps = 0;
    while ($n !== 1) {
        if ($n % 2 === 0) {
            $n = intdiv($n, 2);
        } else {
            $n = 3 * $n + 1;
        }
        $steps++;
    }
    return $steps;
}

check("collatz 1", collatzLength(1), 0);
check("collatz 6", collatzLength(6), 8);
check("collatz 27", collatzLength(27), 111);

// ----- digit manipulation -----
function sumDigits($n) {
    if ($n < 0) {
        $n = -$n;
    }
    if ($n < 10) {
        return $n;
    }
    return $n % 10 + sumDigits(intdiv($n, 10));
}

function reverseNumber($n) {
    $r = 0;
    while ($n > 0) {
        $r = $r * 10 + $n % 10;
        $n = intdiv($n, 10);
    }
    return $r;
}

function isPalindromeNumber($n) {
    return $n === reverseNumber($n);
}

check("sum digits", sumDigits(12345), 15);
check("sum digits neg", sumDigits(-987), 24);
check("reverse number", reverseNumber(12345), 54321);
check("reverse trailing", reverseNumber(1200), 21);
check("palindrome yes", isPalindromeNumber(12321), true);
check("palindrome no", isPalindromeNumber(12345), false);

// ----- primality: trial division vs Sieve of Eratosthenes -----
function isPrime($n) {
    if ($n < 2) {
        return false;
    }
    $i = 2;
    while ($i * $i <= $n) {
        if ($n % $i === 0) {
            return false;
        }
        $i++;
    }
    return true;
}

function countPrimesTrial($limit) {
    $count = 0;
    for ($n = 2; $n <= $limit; $n++) {
        if (isPrime($n)) {
            $count++;
        }
    }
    return $count;
}

function sievePrimes($limit) {
    $sieve = [];
    for ($i = 0; $i <= $limit; $i++) {
        $sieve[] = true;
    }
    $sieve[0] = false;
    if ($limit >= 1) {
        $sieve[1] = false;
    }
    for ($p = 2; $p * $p <= $limit; $p++) {
        if ($sieve[$p]) {
            for ($m = $p * $p; $m <= $limit; $m += $p) {
                $sieve[$m] = false;
            }
        }
    }
    $primes = [];
    for ($i = 2; $i <= $limit; $i++) {
        if ($sieve[$i]) {
            $primes[] = $i;
        }
    }
    return $primes;
}

check("is prime 2", isPrime(2), true);
check("is prime 17", isPrime(17), true);
check("is prime 1", isPrime(1), false);
check("is prime 91", isPrime(91), false);
check("is prime 97", isPrime(97), true);

$primes = sievePrimes(50);
check("sieve count", count($primes), 15);
check("sieve first", $primes[0], 2);
check("sieve last", $primes[count($primes) - 1], 47);
check("sieve matches trial", count($primes), countPrimesTrial(50));
check("count primes 100", countPrimesTrial(100), 25);

// The nth prime, read out of the sieve.
check("6th prime", $primes[5], 13);

// ----- mutual recursion: even / odd -----
function isEvenR($n) {
    if ($n === 0) {
        return true;
    }
    return isOddR($n - 1);
}

function isOddR($n) {
    if ($n === 0) {
        return false;
    }
    return isEvenR($n - 1);
}

check("mutual even 10", isEvenR(10), true);
check("mutual odd 7", isOddR(7), true);
check("mutual even 7", isEvenR(7), false);
check("mutual odd 0", isOddR(0), false);

// ----- Towers of Hanoi: count of moves (should be 2^n - 1) -----
function hanoiMoves($n) {
    if ($n === 0) {
        return 0;
    }
    return 2 * hanoiMoves($n - 1) + 1;
}

check("hanoi 1", hanoiMoves(1), 1);
check("hanoi 3", hanoiMoves(3), 7);
check("hanoi 10", hanoiMoves(10), 1023);
check("hanoi matches pow", hanoiMoves(8), power(2, 8) - 1);

// ----- nested loops: multiplication table and Pythagorean triples -----
$tableSum = 0;
for ($i = 1; $i <= 9; $i++) {
    for ($j = 1; $j <= 9; $j++) {
        $tableSum += $i * $j;
    }
}
check("times table sum", $tableSum, 2025);

$triples = 0;
for ($a = 1; $a <= 20; $a++) {
    for ($b = $a; $b <= 20; $b++) {
        for ($c = $b; $c <= 20; $c++) {
            if ($a * $a + $b * $b === $c * $c) {
                $triples++;
            }
        }
    }
}
check("pythagorean triples", $triples, 6);

// break / continue: sum of numbers 1..100 that are not multiples of 3, stop at 50.
$acc = 0;
$n = 0;
while (true) {
    $n++;
    if ($n > 100) {
        break;
    }
    if ($n > 50) {
        break;
    }
    if ($n % 3 === 0) {
        continue;
    }
    $acc += $n;
}
check("break continue acc", $acc, 867);

// A labelled search using a flag across nested loops.
$found = 0;
$target = 56;
for ($i = 1; $i <= 10; $i++) {
    $done = false;
    for ($j = 1; $j <= 10; $j++) {
        if ($i * $j === $target) {
            $found = $i * 100 + $j;
            $done = true;
            break;
        }
    }
    if ($done) {
        break;
    }
}
check("nested search", $found, 708);

// ----- a small event-driven state machine: a turnstile -----
// States: "locked" and "unlocked". A "coin" unlocks; a "push" through an
// unlocked turnstile records a pass and relocks. Illegal events are ignored.
function runTurnstile($events) {
    $state = "locked";
    $passes = 0;
    $rejected = 0;
    foreach ($events as $ev) {
        if ($state === "locked") {
            if ($ev === "coin") {
                $state = "unlocked";
            } else {
                $rejected++;
            }
        } else {
            if ($ev === "push") {
                $passes++;
                $state = "locked";
            } else {
                // an extra coin while unlocked is wasted but harmless
                $rejected++;
            }
        }
    }
    return $passes * 1000 + $rejected;
}

// coin, push -> 1 pass. push (rejected). coin, coin (2nd wasted), push -> 1 pass.
$events = ["coin", "push", "push", "coin", "coin", "push", "coin"];
// passes: (coin,push)=1, push rejected=1, (coin, coin wasted=1, push)=2 passes,
// trailing coin leaves unlocked. passes=2, rejected=2.
check("turnstile", runTurnstile($events), 2002);
check("turnstile empty", runTurnstile([]), 0);
check("turnstile all push", runTurnstile(["push", "push"]), 2);

// ----- done -----
if ($fails === 0) {
    echo "php-test-big-2 (recursion & control flow) passed\n";
}
exit($fails);
