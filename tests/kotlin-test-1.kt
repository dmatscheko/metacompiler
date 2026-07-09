/* Kotlin subset self test.
 * main() counts failed checks and ends with exitProcess(fails), so the
 * metacompiler run exits with 0 exactly when everything works. **/

var fails = 0

fun check(name: String, got: Int, want: Int) {
    if (got != want) {
        println("FAIL $name: got $got want $want")
        fails++
    }
}
fun checkB(name: String, got: Boolean, want: Boolean) {
    check(name, if (got) 1 else 0, if (want) 1 else 0)
}
fun checkS(name: String, got: String, want: String) {
    if (got != want) {
        println("FAIL $name: got $got want $want")
        fails++
    }
}

fun add(a: Int, b: Int): Int = a + b        // expression body

fun fib(n: Int): Int {
    if (n < 2) { return n }
    return fib(n - 1) + fib(n - 2)
}

fun classify(n: Int): String = when {
    n < 0 -> "negative"
    n == 0 -> "zero"
    else -> "positive"
}

class Counter(val start: Int, var step: Int) {
    var value = start                        // property initializer sees the ctor params

    fun next(): Int {
        value += step                        // implicit property access
        return value
    }
    fun reset() {
        value = start
    }
}

data class Point(val x: Int, val y: Int) {
    fun manhattan(): Int = abs(x) + abs(y)
}

fun main() {
    // arithmetic
    check("precedence", 1 + 2 * 3, 7)
    check("division", 7 / 2, 3)
    check("negative division", -7 / 2, -3)
    check("modulo", 7 % 3, 1)
    check("expression body", add(20, 22), 42)

    // val/var and if as expression
    val a = 10
    var b = 20
    b = if (a > 5) b + 1 else b - 1
    check("if expression", b, 21)
    val big = if (a > b) a else b
    check("max via if", big, 21)

    // when with and without subject
    val w1 = when (a) {
        1, 2 -> 100
        10 -> 200
        else -> 300
    }
    check("when subject", w1, 200)
    checkS("when no subject", classify(-5), "negative")
    checkS("when zero", classify(0), "zero")

    // strings and templates
    val name = "world"
    checkS("template", "hello $name!", "hello world!")
    checkS("template expr", "sum=${a + b}", "sum=31")
    checkS("concat", "a" + 1, "a1")
    check("string length", name.length, 5)
    checkS("nested template", "x=${if (a > 5) "big" else "small"}", "x=big")

    // loops and ranges
    var sum = 0
    for (i in 1..10) { sum += i }
    check("range for", sum, 55)

    var evens = 0
    for (i in 0..20) {
        if (i % 2 == 1) { continue }
        if (i > 10) { break }
        evens += i
    }
    check("break continue", evens, 30)

    var w = 0
    while (w < 5) { w++ }
    check("while", w, 5)

    var dc = 0
    do { dc++ } while (false)
    check("do while", dc, 1)

    // lists
    val list = listOf(3, 1, 4, 1, 5)
    check("list size", list.size, 5)
    check("list index", list[2], 4)
    var lsum = 0
    for (x in list) { lsum += x }
    check("list for", lsum, 14)
    val ml = mutableListOf(1, 2)
    ml.add(3)
    check("mutable add", ml.size, 3)
    checkB("contains", ml.contains(2), true)
    check("get", ml.get(2), 3)

    // classes
    val c = Counter(10, 1)
    check("property param", c.start, 10)
    check("property init", c.value, 10)
    check("method", c.next(), 11)
    c.step = 5
    check("property write", c.next(), 16)
    c.reset()
    check("reset", c.value, 10)

    val p = Point(3, -4)
    check("data class", p.manhattan(), 7)
    check("property read", p.x - p.y, 7)

    // when with range branches
    val g1 = when (85) { in 90..100 -> "A"; in 80..89 -> "B"; else -> "?" }
    checkS("when in range", g1, "B")
    val g2 = when (77) { in 90..100 -> "A"; in 80..89 -> "B"; else -> "C" }
    checkS("when range miss", g2, "C")
    val g3 = when (5) { !in 0..3 -> "out"; else -> "in" }
    checkS("when not in", g3, "out")
    val g4 = when (2) { !in 0..3 -> "out"; else -> "in" }
    checkS("when not in miss", g4, "in")

    // elvis
    var maybe: Int? = null
    check("elvis null", maybe ?: 42, 42)
    maybe = 7
    check("elvis value", maybe ?: 42, 7)

    // functions and recursion
    check("fib", fib(10), 55)
    check("builtin max", max(3, 9), 9)
    check("builtin abs", abs(-6), 6)

    if (fails == 0) { println("Kotlin subset self test passed") }
    exitProcess(fails)
}
