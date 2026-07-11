/* Kotlin subset self test - THEME 4: string & number processing.
 *
 * Exercises number theory and text formatting: Sieve of Eratosthenes (Int flags
 * in a list), prime factorization with a product cross-check, gcd/lcm, proper
 * divisor sum and perfect-number test, digit work (reverse, palindrome, digit
 * sum/count), recursive and iterative base conversion to arbitrary radix strings,
 * Roman numerals via greedy subtraction over parallel value/symbol lists, a
 * string repeat, a comma joiner over an Int list, and FizzBuzz classification -
 * all cross-checked by value and by string length/equality. Strings in the subset
 * support templates, concatenation, .length and == , which is all this uses.
 * main() counts failed checks and ends with exitProcess(fails). **/

var fails = 0

fun check(name: String, got: Int, want: Int) {
    if (got != want) { println("FAIL $name: got $got want $want"); fails++ }
}
fun checkB(name: String, got: Boolean, want: Boolean) {
    check(name, if (got) 1 else 0, if (want) 1 else 0)
}
fun checkS(name: String, got: String, want: String) {
    if (got != want) { println("FAIL $name: got $got want $want"); fails++ }
}

// ----- number theory -----

fun gcd(a: Int, b: Int): Int = if (b == 0) a else gcd(b, a % b)
fun lcm(a: Int, b: Int): Int = a / gcd(a, b) * b

fun sieve(n: Int): List<Int> {
    // comp[i] == 1 marks a composite; index 0/1 are ignored below.
    val comp = mutableListOf()
    for (i in 0..n) { comp.add(0) }
    var p = 2
    while (p * p <= n) {
        if (comp[p] == 0) {
            var m = p * p
            while (m <= n) {
                comp[m] = 1
                m += p
            }
        }
        p += 1
    }
    val primes = mutableListOf()
    for (i in 2..n) {
        if (comp[i] == 0) { primes.add(i) }
    }
    return primes
}

fun isPrime(n: Int): Boolean {
    if (n < 2) { return false }
    var d = 2
    while (d * d <= n) {
        if (n % d == 0) { return false }
        d += 1
    }
    return true
}

fun primeFactors(n: Int): List<Int> {
    val f = mutableListOf()
    var x = n
    var d = 2
    while (d * d <= x) {
        while (x % d == 0) {
            f.add(d)
            x = x / d
        }
        d += 1
    }
    if (x > 1) { f.add(x) }
    return f
}

fun product(a: List<Int>): Int {
    var p = 1
    for (x in a) { p *= x }
    return p
}

fun sumProperDivisors(n: Int): Int {
    var s = 0
    for (i in 1 until n) {
        if (n % i == 0) { s += i }
    }
    return s
}
fun isPerfect(n: Int): Boolean = sumProperDivisors(n) == n

// ----- digit work -----

fun reverseNumber(n: Int): Int {
    var x = n
    var r = 0
    while (x > 0) {
        r = r * 10 + x % 10
        x = x / 10
    }
    return r
}
fun isPalindromeNumber(n: Int): Boolean = reverseNumber(n) == n

fun digitSum(n: Int): Int {
    var x = n
    var s = 0
    while (x > 0) {
        s += x % 10
        x = x / 10
    }
    return s
}
fun digitCount(n: Int): Int = "$n".length

// ----- base conversion -----

// recursive: n in binary as a string
fun toBinary(n: Int): String = if (n < 2) "$n" else toBinary(n / 2) + "${n % 2}"

// iterative: n in an arbitrary radix (2..16) using a digit-symbol list
fun toBase(n: Int, base: Int): String {
    if (n == 0) { return "0" }
    val digits = listOf("0", "1", "2", "3", "4", "5", "6", "7",
                        "8", "9", "A", "B", "C", "D", "E", "F")
    var x = n
    var s = ""
    while (x > 0) {
        s = digits[x % base] + s
        x = x / base
    }
    return s
}

// ----- Roman numerals -----

fun toRoman(n: Int): String {
    val vals = listOf(1000, 900, 500, 400, 100, 90, 50, 40, 10, 9, 5, 4, 1)
    val syms = listOf("M", "CM", "D", "CD", "C", "XC", "L", "XL", "X", "IX", "V", "IV", "I")
    var x = n
    var s = ""
    var i = 0
    while (i < vals.size) {
        while (x >= vals[i]) {
            s = s + syms[i]
            x = x - vals[i]
        }
        i += 1
    }
    return s
}

// ----- string helpers -----

fun repeatStr(s: String, n: Int): String {
    var r = ""
    var i = 0
    while (i < n) {
        r = r + s
        i += 1
    }
    return r
}
fun joinInts(a: List<Int>): String {
    var s = ""
    for (i in 0 until a.size) {
        if (i > 0) { s = s + "," }
        s = s + "${a[i]}"
    }
    return s
}

fun fizz(n: Int): String = when {
    n % 15 == 0 -> "FizzBuzz"
    n % 3 == 0 -> "Fizz"
    n % 5 == 0 -> "Buzz"
    else -> "$n"
}

fun main() {
    // ---- sieve / primes ----
    val primes = sieve(30)
    check("prime count <=30", primes.size, 10)
    check("first prime", primes[0], 2)
    check("last prime <=30", primes[9], 29)
    check("prime sum <=30", primes.sumOf { it }, 129)
    checkB("isPrime 29", isPrime(29), true)
    checkB("isPrime 91", isPrime(91), false)   // 7 * 13
    checkB("isPrime 1", isPrime(1), false)
    // every sieve entry agrees with isPrime
    var mismatch = 0
    for (x in primes) {
        if (!isPrime(x)) { mismatch += 1 }
    }
    check("sieve vs isPrime", mismatch, 0)

    // ---- factorization ----
    val f360 = primeFactors(360)
    check("factors product", product(f360), 360)
    check("factors count", f360.size, 6)       // 2,2,2,3,3,5
    check("factors of prime", primeFactors(97).size, 1)
    check("factors first", f360[0], 2)

    // ---- gcd / lcm / perfect ----
    check("gcd 24 36", gcd(24, 36), 12)
    check("lcm 6 8", lcm(6, 8), 24)
    checkB("perfect 6", isPerfect(6), true)
    checkB("perfect 28", isPerfect(28), true)
    checkB("perfect 12", isPerfect(12), false)
    check("sigma proper 12", sumProperDivisors(12), 16) // 1+2+3+4+6

    // ---- digits ----
    check("reverse 12345", reverseNumber(12345), 54321)
    check("reverse 1200", reverseNumber(1200), 21)      // trailing zeros drop
    checkB("palindrome 12321", isPalindromeNumber(12321), true)
    checkB("palindrome 12345", isPalindromeNumber(12345), false)
    check("digitSum 99999", digitSum(99999), 45)
    check("digitSum 12345", digitSum(12345), 15)
    check("digitCount 12345", digitCount(12345), 5)
    check("digitCount 7", digitCount(7), 1)

    // ---- base conversion ----
    checkS("bin 13", toBinary(13), "1101")
    checkS("bin 1", toBinary(1), "1")
    checkS("bin 0", toBinary(0), "0")
    check("bin 255 len", toBinary(255).length, 8)
    checkS("base16 255", toBase(255, 16), "FF")
    checkS("base16 4095", toBase(4095, 16), "FFF")
    checkS("base2 13", toBase(13, 2), "1101")
    checkS("base8 64", toBase(64, 8), "100")
    checkS("base3 26", toBase(26, 3), "222")
    // toBinary and toBase(,2) agree on 1..40
    var baseMismatch = 0
    for (k in 1..40) {
        if (toBinary(k) != toBase(k, 2)) { baseMismatch += 1 }
    }
    check("binary methods agree", baseMismatch, 0)

    // ---- Roman numerals ----
    checkS("roman 4", toRoman(4), "IV")
    checkS("roman 9", toRoman(9), "IX")
    checkS("roman 49", toRoman(49), "XLIX")
    checkS("roman 1994", toRoman(1994), "MCMXCIV")
    checkS("roman 2024", toRoman(2024), "MMXXIV")
    checkS("roman 3888", toRoman(3888), "MMMDCCCLXXXVIII")

    // ---- string helpers ----
    checkS("repeat ab x3", repeatStr("ab", 3), "ababab")
    check("repeat len", repeatStr("xyz", 4).length, 12)
    checkS("repeat zero", repeatStr("z", 0), "")
    checkS("join", joinInts(listOf(1, 2, 3)), "1,2,3")
    check("join len", joinInts(listOf(10, 20, 30)).length, 8) // "10,20,30"
    checkS("join single", joinInts(listOf(42)), "42")

    // ---- FizzBuzz ----
    checkS("fizz 15", fizz(15), "FizzBuzz")
    checkS("fizz 9", fizz(9), "Fizz")
    checkS("fizz 10", fizz(10), "Buzz")
    checkS("fizz 7", fizz(7), "7")
    // build the 1..5 line and compare
    var line = ""
    for (i in 1..5) {
        if (i > 1) { line = line + " " }
        line = line + fizz(i)
    }
    checkS("fizz line 1..5", line, "1 2 Fizz 4 Buzz")

    // count Fizz/Buzz/FizzBuzz hits in 1..30 (no string needed, uses ==)
    var fizzCount = 0
    var buzzCount = 0
    var fbCount = 0
    for (i in 1..30) {
        val t = fizz(i)
        if (t == "Fizz") { fizzCount += 1 }
        if (t == "Buzz") { buzzCount += 1 }
        if (t == "FizzBuzz") { fbCount += 1 }
    }
    check("fizz hits 1..30", fizzCount, 8)   // 3,6,9,12,18,21,24,27
    check("buzz hits 1..30", buzzCount, 4)   // 5,10,20,25
    check("fizzbuzz hits 1..30", fbCount, 2) // 15,30

    if (fails == 0) { println("Kotlin big-4 (string & number processing) passed") }
    exitProcess(fails)
}
