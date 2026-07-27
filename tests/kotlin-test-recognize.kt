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

// A class carrying nested type declarations. GENUINE in both halves since the
// compiler grew NestedType: the nested declaration lowers like a top-level one and the
// enclosing descriptor gets it under its simple name (see SECTION 27 of
// tests/kotlin-test-full.kt, which asserts the values). Here it only has to parse.
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

    // the enclosing class with a nested type still works, and the nested type itself
    // is now built: Holder.Kind is a real enum descriptor in both halves.
    if (Holder(21).doubled() != 42) { fails = fails + 1 }
    if (Holder.Kind.LEFT.name != "LEFT") { fails = fails + 1 }

    // an assignment whose left side ENDS IN A CALL is still not a modelled lvalue:
    // both sides run (the calls stay observable), but the placeholder writes nothing.
    helper(3) += 1                   // plusAssign on a call result: notImpl, sides run
    // A chain that ends in a FIELD is genuine in both halves now - the write happens.
    // boxOf makes a FRESH Holder per call, so the write is unobservable here by
    // construction; SECTION 28 of tests/kotlin-test-full.kt asserts the observable
    // form against a shared object.
    boxOf(9).tag = 100               // foo(x).field = y: genuine write, then discarded
    if (boxOf(9).tag != 9) { fails = fails + 1 }

    // a range as a VALUE (outside a for-loop / when-in) is recognised but not modelled
    // (no iterable range object): the bounds still run, the range itself is a placeholder.
    // The genuine for-loop and when-in range forms are unaffected (they bind bounds lower).
    val someRange = helper(0) until helper(3)      // a until b as a value: notImpl
    val stepped = 0..10 step 2                      // a..b step c as a value: notImpl
    var loopSum = 0
    for (i in 0 until 4) { loopSum = loopSum + i }  // for-loop range: still genuine
    if (loopSum != 6) { fails = fails + 1 }
    val inRange = when (3) { in 0..5 -> 1; else -> 0 }   // when-in range: still genuine
    if (inRange != 1) { fails = fails + 1 }

    // try/catch used as an expression value is recognised but its value is not modelled:
    // the try/catch bodies still run (helper(4) is called under exception handling), the
    // value itself is a placeholder (left unused here). The try/catch STATEMENT is unaffected.
    val fromTry = try { helper(4) } catch (e: Exception) { -1 }

    // a destructuring for-loop iterates the collection genuinely, but the per-element
    // destructuring is not modelled (a and b bind to undefined, so they are left unused);
    // the loop still runs once per pair, so the counter reaches the list size.
    var pairsSeen = 0
    for ((a, b) in listOf(10 to 1, 20 to 2, 30 to 3)) { pairsSeen = pairsSeen + 1 }
    if (pairsSeen != 3) { fails = fails + 1 }

    exitProcess(fails)
}
