/* Kotlin subset self test - THEME 2: heavy recursion & control flow.
 *
 * Exercises: single and mutual recursion (gcd, factorial, fast power, fibonacci
 * recursive-vs-iterative, Ackermann, digit sum, Tower-of-Hanoi move count,
 * isEven/isOdd), iterative control flow (Collatz, nested loops with break /
 * continue), a subject/subject-less `when` classifier chain, a turnstile finite
 * state machine driven by an input list, and a stack-machine RPN evaluator over a
 * token list. main() counts failed checks and ends with exitProcess(fails). **/

var fails = 0

fun check(name: String, got: Int, want: Int) {
    if (got != want) { println("FAIL $name: got $got want $want"); fails++ }
}
fun checkB(name: String, got: Boolean, want: Boolean) {
    check(name, if (got) 1 else 0, if (want) 1 else 0)
}

// ----- recursion -----

fun gcd(a: Int, b: Int): Int = if (b == 0) a else gcd(b, a % b)
fun lcm(a: Int, b: Int): Int = a / gcd(a, b) * b

fun factorial(n: Int): Int = if (n <= 1) 1 else n * factorial(n - 1)

fun power(base: Int, exp: Int): Int {
    if (exp == 0) { return 1 }
    val half = power(base, exp / 2)
    val sq = half * half
    return if (exp % 2 == 0) sq else sq * base
}

fun fibRec(n: Int): Int {
    if (n < 2) { return n }
    return fibRec(n - 1) + fibRec(n - 2)
}
fun fibIter(n: Int): Int {
    var a = 0
    var b = 1
    var i = 0
    while (i < n) {
        val t = a + b
        a = b
        b = t
        i += 1
    }
    return a
}

fun ack(m: Int, n: Int): Int {
    if (m == 0) { return n + 1 }
    if (n == 0) { return ack(m - 1, 1) }
    return ack(m - 1, ack(m, n - 1))
}

fun digitSum(n: Int): Int = if (n < 10) n else n % 10 + digitSum(n / 10)

fun hanoi(n: Int): Int = if (n == 0) 0 else 2 * hanoi(n - 1) + 1

// mutual recursion
fun isEven(n: Int): Boolean = if (n == 0) true else isOdd(n - 1)
fun isOdd(n: Int): Boolean = if (n == 0) false else isEven(n - 1)

// ----- iterative control flow -----

fun collatz(n: Int): Int {
    var x = n
    var steps = 0
    while (x != 1) {
        if (x % 2 == 0) { x = x / 2 } else { x = 3 * x + 1 }
        steps += 1
    }
    return steps
}

// count coprime pairs (i,j), 1<=i<j<=n, using nested loops with continue
fun coprimePairs(n: Int): Int {
    var c = 0
    for (i in 1..n) {
        for (j in i + 1..n) {
            if (gcd(i, j) != 1) { continue }
            c += 1
        }
    }
    return c
}

// first index in a matrix scan whose value hits a target; break stops the inner scan
fun firstHit(rows: Int, cols: Int, target: Int): Int {
    var found = -1
    for (i in 0 until rows) {
        var stop = false
        for (j in 0 until cols) {
            val v = i * cols + j
            if (v == target) { found = i * 100 + j; stop = true; break }
        }
        if (stop) { break }
    }
    return found
}

// ----- when-based classifier -----

fun classify(n: Int): Int = when {
    n < 0 -> -1
    n == 0 -> 0
    n < 10 -> 1
    n < 100 -> 2
    n < 1000 -> 3
    else -> 4
}

fun weekPart(day: Int): String = when (day) {
    1, 2, 3, 4, 5 -> "work"
    6, 7 -> "rest"
    else -> "invalid"
}

// ----- finite state machine: a turnstile -----
// state 0 = LOCKED, state 1 = UNLOCKED; input 0 = PUSH, input 1 = COIN.
fun turnstile(inputs: List<Int>): Int {
    var state = 0
    for (inp in inputs) {
        state = when (state) {
            0 -> if (inp == 1) 1 else 0          // coin unlocks, push stays locked
            else -> if (inp == 0) 0 else 1       // push locks, coin stays unlocked
        }
    }
    return state
}

// ----- stack-machine RPN evaluator over a token list -----
// operands are non-negative ints; operators are encoded as -101..-104.
fun evalRpn(tokens: List<Int>): Int {
    val st = mutableListOf()
    var sp = 0
    for (t in tokens) {
        if (t <= -101) {
            val b = st[sp - 1]
            val a = st[sp - 2]
            sp -= 2
            var r = 0
            if (t == -101) { r = a + b }
            else if (t == -102) { r = a - b }
            else if (t == -103) { r = a * b }
            else { r = a / b }
            if (sp < st.size) { st[sp] = r } else { st.add(r) }
            sp += 1
        } else {
            if (sp < st.size) { st[sp] = t } else { st.add(t) }
            sp += 1
        }
    }
    return st[sp - 1]
}

fun main() {
    // gcd / lcm
    check("gcd 48 36", gcd(48, 36), 12)
    check("gcd 17 5", gcd(17, 5), 1)
    check("lcm 4 6", lcm(4, 6), 12)
    check("lcm 21 6", lcm(21, 6), 42)

    // factorial / power
    check("fact 5", factorial(5), 120)
    check("fact 7", factorial(7), 5040)
    check("pow 2^10", power(2, 10), 1024)
    check("pow 3^4", power(3, 4), 81)
    check("pow 5^0", power(5, 0), 1)
    check("pow 7^3", power(7, 3), 343)

    // fibonacci: recursive matches iterative for 0..16
    for (n in 0..16) { check("fib $n", fibRec(n), fibIter(n)) }
    check("fib 16 value", fibIter(16), 987)

    // Ackermann
    check("ack 2 3", ack(2, 3), 9)
    check("ack 3 3", ack(3, 3), 61)
    check("ack 3 4", ack(3, 4), 125)

    // digit sum / hanoi
    check("digitSum 12345", digitSum(12345), 15)
    check("digitSum 99999", digitSum(99999), 45)
    check("hanoi 10", hanoi(10), 1023)
    check("hanoi 1", hanoi(1), 1)

    // mutual recursion
    checkB("isEven 10", isEven(10), true)
    checkB("isEven 7", isEven(7), false)
    checkB("isOdd 7", isOdd(7), true)

    // Collatz
    check("collatz 27", collatz(27), 111)
    check("collatz 6", collatz(6), 8)
    check("collatz 1", collatz(1), 0)

    // nested-loop control flow
    check("coprime pairs 10", coprimePairs(10), 31)
    check("firstHit 3x4 -> 7", firstHit(3, 4, 7), 103)   // v=7 at i=1,j=3
    check("firstHit miss", firstHit(3, 4, 99), -1)

    // when classifier
    check("classify -5", classify(-5), -1)
    check("classify 0", classify(0), 0)
    check("classify 7", classify(7), 1)
    check("classify 55", classify(55), 2)
    check("classify 720", classify(720), 3)
    check("classify 9000", classify(9000), 4)
    checkB("weekPart 3", weekPart(3) == "work", true)
    checkB("weekPart 7", weekPart(7) == "rest", true)

    // finite state machine
    check("turnstile coin->push", turnstile(listOf(1, 0)), 0)
    check("turnstile coin", turnstile(listOf(1)), 1)
    check("turnstile push first", turnstile(listOf(0, 0, 1)), 1)
    check("turnstile coin coin push", turnstile(listOf(1, 1, 0)), 0)

    // RPN: (3 + 4) * 5 = 35 ; 3 4 + 2 * = 14 ; 10 2 / 3 - = 2
    check("rpn (3+4)*5", evalRpn(listOf(3, 4, -101, 5, -103)), 35)
    check("rpn 3 4 + 2 *", evalRpn(listOf(3, 4, -101, 2, -103)), 14)
    check("rpn 10 2 / 3 -", evalRpn(listOf(10, 2, -104, 3, -102)), 2)
    check("rpn nested", evalRpn(listOf(2, 3, -101, 4, 5, -101, -103)), 45)  // (2+3)*(4+5)

    if (fails == 0) { println("Kotlin big-2 (recursion & control flow) passed") }
    exitProcess(fails)
}
