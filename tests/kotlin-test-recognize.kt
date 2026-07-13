/* Kotlin widened RECOGNITION surface: constructs a real-world Kotlin file uses that the
 * grammar now PARSES. Some are genuinely lowered (the non-null assertion x!!, referential
 * equality === / !==, default parameter values and vararg parameters, loop labels with
 * break@l / continue@l, bare callable references ::fn, labelled returns targeting the
 * innermost lambda), others are recognised-but-not-implemented and routed through
 * notImpl (qualified callable references Type::member and anonymous functions
 * fun(...) {...}). Because the notImpl constructs abort at the FIRST one hit, a
 * plain run stops with a clean file:line message (this file SHOULD fail by default). With
 * -warn-unsupported each is warned and replaced by a placeholder, the genuinely lowered
 * features drive the self-check, and main() ends with exitProcess(fails) so that run
 * exits 0 when they pass. */

package demo.recognize

import kotlin.math.abs

// Default parameter value (= 2) and a vararg parameter are parsed; the default is not
// supplied at call sites, so callers pass the argument explicitly.
fun weightedSum(base: Int, factor: Int = 2, vararg extra: Int): Int {
    return base * factor
}

fun helper(x: Int): Int = x * 3
fun boxOf(k: Int): Holder = Holder(k)

// A class carrying a nested type declaration: recognised but not lowered (the subset
// has a flat class model). Under -warn-unsupported the nested type is skipped and the
// enclosing class's own members still work.
class Holder(val tag: Int) {
    data class Slot(val name: String, val on: Boolean = true)
    enum class Kind { LEFT, RIGHT }
    fun doubled(): Int = tag * 2
}

fun main() {
    var fails = 0

    // ---- genuinely lowered widened features drive the self-check ----

    // non-null assertion: the identity for a non-null value, and it still chains
    val boxed: Int = 41
    val unwrapped = boxed!!
    if (unwrapped + 1 != 42) { fails = fails + 1 }
    if (boxed!! != 41) { fails = fails + 1 }

    // referential equality === / !== (lowered like structural == / !=)
    val s = "kotlin"
    if (!(s === s)) { fails = fails + 1 }
    if (s !== "other") { } else { fails = fails + 1 }

    // default parameter value is parsed; the call passes factor explicitly
    if (weightedSum(10, 2) != 20) { fails = fails + 1 }

    // a label on a loop without labelled jumps changes nothing
    var product = 1
    outer@ for (i in 1..3) {
        product = product * i
    }
    if (product != 6) { fails = fails + 1 }

    // ---- recognised-but-not-implemented constructs (exercised, results unused) ----

    val pi = 3.14159                       // floating-point literal
    val g = 6.674e-11                      // floating-point literal with exponent
    val ratio = 2.0f                       // floating-point literal with an f suffix
    val doc = """
        Raw "triple-quoted" string
        spanning multiple lines.
    """                                    // triple-quoted raw string literal
    // a bare callable reference is genuine: the function value, stored and called
    val ref = ::helper
    if (ref(2) != 6) { fails = fails + 1 }
    val kref = String::length              // type-qualified callable reference: notImpl
    val adder = fun(x: Int): Int {         // anonymous function expression
        return x + 100
    }

    // labelled break / continue are genuine: continue@grid resumes the outer
    // loop, break@grid leaves it (2 visits for i=1,2, then one before the break)
    var visits = 0
    grid@ for (i in 1..3) {
        for (j in 1..3) {
            visits = visits + 1
            if (j == 2) continue@grid
            if (i == 3) break@grid
        }
    }
    if (visits != 5) { fails = fails + 1 }

    // labelled return inside a lambda is genuine: it skips the rest of the body
    val nums = listOf(1, 2, 3, 4)
    var lrSum = 0
    nums.forEach { n ->
        println(n)
        if (n == 2) return@forEach
        lrSum = lrSum + n
    }
    if (lrSum != 8) { fails = fails + 1 }

    // a destructuring lambda parameter is recognised but not lowered: the names bind
    // to undefined under -warn-unsupported, so the map still runs (returns a constant).
    val destructured = nums.map { (a, b) -> 0 }
    if (destructured.size != 4) { fails = fails + 1 }

    // the enclosing class with a nested type still works (the nested type was skipped)
    if (Holder(21).doubled() != 42) { fails = fails + 1 }

    // an assignment whose left side contains a call is recognised but not written: the
    // target is a call result / plusAssign, not a modelled lvalue. Both sides still run
    // (the calls stay observable), but the placeholder writes nothing.
    helper(3) += 1                   // plusAssign on a call result: notImpl, sides run
    boxOf(9).tag = 100               // foo(x).field = y: write not modelled, sides run
    if (boxOf(9).tag != 9) { fails = fails + 1 }

    exitProcess(fails)
}
