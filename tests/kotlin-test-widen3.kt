/* Compact self-check for the Kotlin parse-surface widenings, in a small geometry theme
 * deliberately unlike any real application. Every construct here is genuine now and is
 * asserted, including the abstract (bodiless) member function, which the full-syntax
 * work implemented. main() runs to exitProcess(fails), which is 0 when every check
 * passes, with or without -warn-unsupported.
 *
 * History: the abstract member used to be recognised-but-not-lowered, which made a
 * flagless run abort, so this file was a by-design SHOULD-FAIL guard. Implementing it
 * removed that premise and the matrix entry became an ordinary one.
 *
 * Covers: labelled receiver this@Box; calls with a trailing lambda, the no-paren form,
 * and explicit type arguments (top-level and method); `when` with a val-binding subject
 * and is-type arms (never match, else wins) on consecutive lines (a postfix `is` must
 * not swallow the following arrow); a statement-level annotation; the infix `to` (a
 * Pair read via .first/.second); and a top-level extension function (genuine: the
 * receiver binds to `this`). */

// A genuine top-level extension function; the abstract (bodiless) member only warns.
fun Int.stepped(): Int = this + 1

abstract class Shape {
    abstract fun sides(): Int
}

// A generic top-level function; the call-site type argument is parsed and ignored.
fun <T> castTo(value: Int): Int = value

// Higher-order helpers driven by trailing lambdas.
fun repeatSum(k: Int, body: () -> Int): Int {
    var s = 0
    var i = 0
    while (i < k) {
        s = s + body()
        i = i + 1
    }
    return s
}
fun once(body: () -> Int): Int = body()
fun add2(a: Int, b: Int): Int = a + b

class Box(val side: Int) {
    // A generic method (type argument ignored) that reads the labelled receiver.
    fun <T> area(): Int = this@Box.side * side
}

// A `when` whose subject binds a name read in the arms; the is-arms are genuine type
// tests now (an Int subject hits `is Number`, never `is CharSequence`). The two
// `is ... ->` on consecutive lines exercise the arrow guard.
fun grade(n: Int): Int = when (val doubled = n + n) {
    0 -> -1
    is CharSequence -> doubled - 1
    is Number -> doubled
    else -> doubled + 1
}

fun main() {
    var fails = 0

    // labelled this@Box + a generic method call
    if (Box(4).area<Int>() != 16) { fails = fails + 1 }
    // a generic top-level call
    if (castTo<Int>(7) != 7) { fails = fails + 1 }
    // a trailing lambda after parentheses, and the no-paren form
    if (repeatSum(3) { 2 } != 6) { fails = fails + 1 }
    if (once { 9 } != 9) { fails = fails + 1 }
    // no-paren trailing-lambda calls in argument position
    if (add2(once { 3 }, once { 4 }) != 7) { fails = fails + 1 }
    // a trailing lambda closing over a local
    val base = 10
    if (once { base } != 10) { fails = fails + 1 }
    // when subject binding: the Int subject matches `is Number`; value arm still wins first
    if (grade(5) != 10) { fails = fails + 1 }
    if (grade(0) != -1) { fails = fails + 1 }

    // a statement-level annotation: the val is declared, the annotation ignored
    @Suppress("UNUSED_VARIABLE")
    val flagged = 42
    if (flagged != 42) { fails = fails + 1 }

    // the infix `to` is genuine now: a Pair with first/second
    val paired = 3 to 4
    if (paired.first + paired.second != 7) { fails = fails + 1 }

    // the top-level extension function is genuine now: receiver bound to `this`
    if (41.stepped() != 42) { fails = fails + 1 }

    exitProcess(fails)
}
