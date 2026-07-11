/* Kotlin subset self test - THEME 3: object-oriented & functional features.
 *
 * Exercises the language's signature surface: classes with primary-constructor
 * properties and methods that return fresh instances (an immutable Fraction with
 * gcd-reduce / add / mul / cross-multiply equality), a nullable-reference singly
 * linked list built through a class (cons / length / sum / reverse / contains),
 * an `object` singleton holding mutable state, a `companion object` factory,
 * `when` expressions, elvis and safe calls on nullable object refs, and a heavy
 * dose of first-class functions - lambdas, closures that capture and mutate, a
 * function factory, function composition, and the list higher-order pipeline.
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

fun gcdOf(a: Int, b: Int): Int = if (b == 0) a else gcdOf(b, a % b)

// ----- an immutable Fraction; methods return new Fractions -----

class Fraction(val num: Int, val den: Int) {
    fun reduce(): Fraction {
        val g0 = gcdOf(abs(num), abs(den))
        val g = if (g0 == 0) 1 else g0
        return Fraction(num / g, den / g)
    }
    fun add(o: Fraction): Fraction =
        Fraction(num * o.den + o.num * den, den * o.den).reduce()
    fun mul(o: Fraction): Fraction =
        Fraction(num * o.num, den * o.den).reduce()
    fun eq(o: Fraction): Boolean = num * o.den == o.num * den
    fun show(): String = "$num/$den"
}

// ----- singly linked list via a class with a nullable next -----

class Node(val value: Int) {
    var next: Node? = null
}

fun cons(v: Int, tail: Node?): Node {
    val n = Node(v)
    n.next = tail
    return n
}
fun fromList(a: List<Int>): Node? {
    var h: Node? = null
    for (i in a.size - 1 downTo 0) { h = cons(a[i], h) }
    return h
}
fun length(head: Node?): Int {
    var n = head
    var c = 0
    while (n != null) {
        c += 1
        n = n?.next
    }
    return c
}
fun sumNodes(head: Node?): Int {
    var s = 0
    var n = head
    while (n != null) {
        s += n?.value ?: 0
        n = n?.next
    }
    return s
}
fun contains(head: Node?, target: Int): Boolean {
    var n = head
    while (n != null) {
        if ((n?.value ?: 0) == target) { return true }
        n = n?.next
    }
    return false
}
fun reverse(head: Node?): Node? {
    var prev: Node? = null
    var cur = head
    while (cur != null) {
        val nx = cur.next    // cur is non-null inside the loop
        cur.next = prev
        prev = cur
        cur = nx
    }
    return prev
}
// fold a list into a decimal string "a>b>c"
fun showList(head: Node?): String {
    var s = ""
    var n = head
    var first = true
    while (n != null) {
        if (!first) { s = s + ">" }
        s = s + "${n?.value ?: 0}"
        first = false
        n = n?.next
    }
    return s
}

// ----- singleton object with mutable state -----

object IdGen {
    var last = 0
    fun next(): Int {
        last += 1
        return last
    }
    fun reset() { last = 0 }
}

// ----- companion-object factory + instance side -----

class Widget(val id: Int, val name: String) {
    fun tag(): String = "$name#$id"

    companion object {
        var made = 0
        fun create(name: String): Widget {
            made += 1
            return Widget(made, name)
        }
        fun count(): Int = made
    }
}

// ----- first-class functions -----

fun adder(base: Int): (Int) -> Int = { x -> base + x }
fun compose(f: (Int) -> Int, g: (Int) -> Int): (Int) -> Int = { x -> f(g(x)) }
fun applyN(f: (Int) -> Int, n: Int, start: Int): Int {
    var acc = start
    var i = 0
    while (i < n) {
        acc = f(acc)
        i += 1
    }
    return acc
}

fun main() {
    // ---- Fraction arithmetic ----
    val half = Fraction(1, 2)
    val third = Fraction(1, 3)
    val sum = half.add(third)          // 5/6
    checkS("frac add show", sum.show(), "5/6")
    check("frac add num", sum.num, 5)
    check("frac add den", sum.den, 6)

    val prod = half.mul(Fraction(2, 3)) // 1/3
    checkS("frac mul show", prod.show(), "1/3")
    checkB("frac mul eq third", prod.eq(third), true)

    val reduced = Fraction(6, 8).reduce()
    checkS("frac reduce", reduced.show(), "3/4")
    checkB("frac cross eq", Fraction(2, 4).eq(half), true)
    checkB("frac cross neq", Fraction(2, 5).eq(half), false)

    // a running sum of 1/2 + 1/3 + 1/6 = 1
    val whole = half.add(third).add(Fraction(1, 6))
    checkS("frac sum to one", whole.show(), "1/1")

    // ---- linked list ----
    val ll = fromList(listOf(3, 1, 4, 1, 5, 9))
    check("list length", length(ll), 6)
    check("list sum", sumNodes(ll), 23)
    checkB("list contains", contains(ll, 4), true)
    checkB("list missing", contains(ll, 7), false)
    checkS("list show", showList(ll), "3>1>4>1>5>9")

    // safe-call chain and elvis over nodes (before the in-place reverse mutates ll)
    check("hop 2", ll?.next?.next?.value ?: -1, 4)
    val short = fromList(listOf(42))
    check("hop past end", short?.next?.value ?: -1, -1)

    val rev = reverse(ll)
    checkS("list reversed", showList(rev), "9>5>1>4>1>3")
    check("reversed length", length(rev), 6)
    check("reversed sum equal", sumNodes(rev), 23)

    check("empty list length", length(null), 0)
    check("empty list sum", sumNodes(null), 0)
    checkB("empty list contains", contains(null, 1), false)

    // ---- object singleton ----
    IdGen.reset()
    check("idgen 1", IdGen.next(), 1)
    check("idgen 2", IdGen.next(), 2)
    check("idgen 3", IdGen.next(), 3)
    check("idgen state", IdGen.last, 3)
    IdGen.reset()
    check("idgen after reset", IdGen.next(), 1)

    // ---- companion factory ----
    Widget.made = 0
    val w1 = Widget.create("alpha")
    val w2 = Widget.create("beta")
    check("widget1 id", w1.id, 1)
    check("widget2 id", w2.id, 2)
    checkS("widget tag", w2.tag(), "beta#2")
    check("widget count", Widget.count(), 2)

    // ---- first-class functions ----
    val add10 = adder(10)
    check("adder", add10(5), 15)

    val inc = { x: Int -> x + 1 }
    val dbl = { x: Int -> x * 2 }
    val incThenDbl = compose(dbl, inc)   // dbl(inc(x))
    check("compose", incThenDbl(4), 10)  // (4+1)*2
    val dblThenInc = compose(inc, dbl)   // inc(dbl(x))
    check("compose order", dblThenInc(4), 9)

    check("applyN inc 5", applyN(inc, 5, 0), 5)
    check("applyN dbl 4", applyN(dbl, 4, 1), 16) // 1 -> 2 -> 4 -> 8 -> 16

    // closure that captures and mutates an outer var
    var running = 0
    val accumulate = { x: Int -> running += x; running }
    check("closure acc 1", accumulate(10), 10)
    check("closure acc 2", accumulate(5), 15)
    check("closure sees var", running, 15)

    // higher-order pipeline over a list
    val data = listOf(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
    check("pipeline sum evens sq", data.filter { it % 2 == 0 }.map { it * it }.sumOf { it }, 4 + 16 + 36 + 64 + 100)
    check("pipeline count > 5", data.count { it > 5 }, 5)
    checkB("pipeline any > 9", data.any { it > 9 }, true)
    checkB("pipeline any > 99", data.any { it > 99 }, false)
    check("pipeline map+sumOf", data.map { it + 1 }.sumOf { it }, 65)

    // a when expression driving a small scoring function
    val scores = listOf(95, 82, 71, 64, 100, 58)
    var honors = 0
    for (s in scores) {
        val g = when {
            s >= 90 -> 4
            s >= 80 -> 3
            s >= 70 -> 2
            s >= 60 -> 1
            else -> 0
        }
        if (g >= 3) { honors += 1 }
    }
    check("honors count", honors, 3)

    if (fails == 0) { println("Kotlin big-3 (OO & functional) passed") }
    exitProcess(fails)
}
