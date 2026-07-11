// Swift subset self test: recursion and control flow.
//
// Theme: recursion and branching pushed hard - iterative vs recursive Fibonacci,
// factorial, the Ackermann function, fast exponentiation, Euclid's gcd, Collatz
// step counts, Tower of Hanoi move counts, Pascal's triangle by recurrence,
// mutual recursion (isEven/isOdd), recursive digit sums and digital roots,
// recursive array folds, and triple-nested loops counting Pythagorean triples,
// plus a classifier that stacks guard/if/switch. Top level code runs in source
// order, counts failed checks and ends with exit(failures), so the run exits 0
// exactly when every check passes. The interpreter and the LLVM-IR compiler must
// agree on every result. Int arithmetic is 32 bit, so magnitudes stay small.

var fails = 0

func check(_ name: String, _ got: Int, _ want: Int) {
    if got != want {
        print("FAIL \(name): got \(got) want \(want)")
        fails += 1
    }
}
func checkB(_ name: String, _ got: Bool, _ want: Bool) {
    check(name, got ? 1 : 0, want ? 1 : 0)
}

// ---- Fibonacci: recursive and iterative must agree ----

func fibRec(_ n: Int) -> Int {
    if n < 2 {
        return n
    }
    return fibRec(n - 1) + fibRec(n - 2)
}
func fibIter(_ n: Int) -> Int {
    if n < 2 {
        return n
    }
    var a = 0
    var b = 1
    var i = 2
    while i <= n {
        let c = a + b
        a = b
        b = c
        i += 1
    }
    return b
}

// ---- factorial (kept within 32 bit) ----

func factorial(_ n: Int) -> Int {
    if n <= 1 {
        return 1
    }
    return n * factorial(n - 1)
}

// ---- the Ackermann function: deep, doubly recursive ----

func ackermann(_ m: Int, _ n: Int) -> Int {
    if m == 0 {
        return n + 1
    }
    if n == 0 {
        return ackermann(m - 1, 1)
    }
    return ackermann(m - 1, ackermann(m, n - 1))
}

// ---- fast exponentiation by squaring (recursive) ----

func power(_ base: Int, _ exp: Int) -> Int {
    if exp == 0 {
        return 1
    }
    let half = power(base, exp / 2)
    if exp % 2 == 0 {
        return half * half
    }
    return half * half * base
}

// ---- Euclid's gcd, recursive ----

func gcdRec(_ a: Int, _ b: Int) -> Int {
    if b == 0 {
        return a < 0 ? -a : a
    }
    return gcdRec(b, a % b)
}

// ---- Collatz step count ----

func collatzSteps(_ n: Int) -> Int {
    var x = n
    var steps = 0
    while x != 1 {
        if x % 2 == 0 {
            x = x / 2
        } else {
            x = 3 * x + 1
        }
        steps += 1
    }
    return steps
}

// ---- Tower of Hanoi move count ----

func hanoi(_ n: Int) -> Int {
    if n == 0 {
        return 0
    }
    return 2 * hanoi(n - 1) + 1
}

// ---- Pascal's triangle by the binomial recurrence ----

func choose(_ n: Int, _ k: Int) -> Int {
    if k < 0 || k > n {
        return 0
    }
    if k == 0 || k == n {
        return 1
    }
    return choose(n - 1, k - 1) + choose(n - 1, k)
}

// ---- mutual recursion ----

func isEven(_ n: Int) -> Bool {
    if n == 0 {
        return true
    }
    return isOdd(n - 1)
}
func isOdd(_ n: Int) -> Bool {
    if n == 0 {
        return false
    }
    return isEven(n - 1)
}

// ---- recursive digit sum and digital root ----

func sumDigits(_ n: Int) -> Int {
    let x = n < 0 ? -n : n
    if x < 10 {
        return x
    }
    return x % 10 + sumDigits(x / 10)
}
func digitalRoot(_ n: Int) -> Int {
    let s = sumDigits(n)
    if s < 10 {
        return s
    }
    return digitalRoot(s)
}

// ---- iterative number reversal and palindrome test ----

func reverseNum(_ n: Int) -> Int {
    var x = n
    var r = 0
    while x > 0 {
        r = r * 10 + x % 10
        x = x / 10
    }
    return r
}
func isPalindromeNum(_ n: Int) -> Bool {
    return reverseNum(n) == n
}

// ---- recursive array folds (index-threaded) ----

func arraySum(_ a: [Int], _ i: Int) -> Int {
    let n = a.count
    if i >= n {
        return 0
    }
    return a[i] + arraySum(a, i + 1)
}
func arrayMax(_ a: [Int], _ i: Int) -> Int {
    let last = a.count - 1
    if i >= last {
        return a[last]
    }
    let rest = arrayMax(a, i + 1)
    return a[i] > rest ? a[i] : rest
}

// ---- a classifier that stacks guard, if and switch ----

func classify(_ n: Int) -> Int {
    guard n >= 0 else {
        return -1
    }
    if n == 0 {
        return 0
    }
    switch n % 3 {
    case 0:
        if n % 2 == 0 {
            return 6
        } else {
            return 3
        }
    case 1:
        return 1
    default:
        return 2
    }
}

// ---- triple-nested loops: count Pythagorean triples a<b<c<=N ----

func pythagoreanTriples(_ N: Int) -> Int {
    var count = 0
    for c in 1...N {
        for b in 1..<c {
            for a in 1..<b {
                if a * a + b * b == c * c {
                    count += 1
                }
            }
        }
    }
    return count
}

func main() {
    // Fibonacci: the two implementations agree across a range. The recursive
    // version stays modest (the frozen MetaJS interpreter is ~100x slower); the
    // iterative one carries the larger inputs.
    check("fibRec 10", fibRec(10), 55)
    check("fibIter 10", fibIter(10), 55)
    check("fibRec 16", fibRec(16), 987)
    check("fibIter 16", fibIter(16), 987)
    check("fibIter 20", fibIter(20), 6765)
    check("fibIter 30", fibIter(30), 832040)
    var mismatch = 0
    var f = 0
    while f <= 16 {
        if fibRec(f) != fibIter(f) {
            mismatch += 1
        }
        f += 1
    }
    check("fib agree 0..16", mismatch, 0)

    // factorial
    check("fact 5", factorial(5), 120)
    check("fact 10", factorial(10), 3628800)
    check("fact 12", factorial(12), 479001600)

    // Ackermann
    check("ack 0 0", ackermann(0, 0), 1)
    check("ack 2 3", ackermann(2, 3), 9)
    check("ack 3 3", ackermann(3, 3), 61)

    // fast exponentiation
    check("pow 2^10", power(2, 10), 1024)
    check("pow 3^7", power(3, 7), 2187)
    check("pow 5^4", power(5, 4), 625)
    check("pow 7^0", power(7, 0), 1)
    check("pow 2^30", power(2, 30), 1073741824)

    // gcd
    check("gcd 1071 462", gcdRec(1071, 462), 21)
    check("gcd 270 192", gcdRec(270, 192), 6)
    check("gcd 13 0", gcdRec(13, 0), 13)

    // Collatz
    check("collatz 1", collatzSteps(1), 0)
    check("collatz 6", collatzSteps(6), 8)
    check("collatz 27", collatzSteps(27), 111)

    // Hanoi
    check("hanoi 1", hanoi(1), 1)
    check("hanoi 10", hanoi(10), 1023)
    check("hanoi 16", hanoi(16), 65535)

    // Pascal
    check("C(10,5)", choose(10, 5), 252)
    check("C(12,6)", choose(12, 6), 924)
    check("C(6,0)", choose(6, 0), 1)
    check("C(6,7)", choose(6, 7), 0)
    // row 6 of Pascal's triangle sums to 2^6
    var rowSum = 0
    var k = 0
    while k <= 6 {
        rowSum += choose(6, k)
        k += 1
    }
    check("Pascal row 6 sum", rowSum, 64)

    // mutual recursion
    checkB("isEven 100", isEven(100), true)
    checkB("isOdd 77", isOdd(77), true)
    checkB("isEven 77", isEven(77), false)
    checkB("isOdd 100", isOdd(100), false)

    // digit sums / digital root
    check("sumDigits 12345", sumDigits(12345), 15)
    check("sumDigits neg", sumDigits(-9042), 15)
    check("digitalRoot 9875", digitalRoot(9875), 2)
    check("digitalRoot 12345", digitalRoot(12345), 6)

    // number reversal + palindrome
    check("reverse 12345", reverseNum(12345), 54321)
    check("reverse 1200", reverseNum(1200), 21)
    checkB("pal 12321", isPalindromeNum(12321), true)
    checkB("pal 12345", isPalindromeNum(12345), false)
    checkB("pal 7", isPalindromeNum(7), true)

    // recursive array folds
    let nums = [3, 8, 1, 9, 4, 7, 2, 6, 5]
    check("arraySum", arraySum(nums, 0), 45)
    check("arrayMax", arrayMax(nums, 0), 9)
    check("arraySum single", arraySum([42], 0), 42)

    // classifier: guard/if/switch stacked
    check("classify -5", classify(-5), -1)
    check("classify 0", classify(0), 0)
    check("classify 6", classify(6), 6)
    check("classify 9", classify(9), 3)
    check("classify 4", classify(4), 1)
    check("classify 5", classify(5), 2)
    check("classify 12", classify(12), 6)

    // triple-nested loops
    check("triples <=13", pythagoreanTriples(13), 3)
    check("triples <=20", pythagoreanTriples(20), 6)
    check("triples <=5", pythagoreanTriples(5), 1)

    if fails == 0 {
        print("Swift recursion self test passed")
    }
}

main()
exit(fails)
