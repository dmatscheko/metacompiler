/* Kotlin widened syntax that is PARSED but not fully lowered. Modifiers and
 * annotations (no args) are parsed and ignored; `as` casts, `is` type tests and
 * raw strings are genuine now; an `enum class` is recognised but not implemented.
 * Without flags the compile stops at the first unimplemented construct with a
 * clean file:line message (this file SHOULD fail by default). With
 * -warn-unsupported the enum is warned and skipped, so the program runs and
 * self-checks: main() ends with exitProcess(fails). (try/catch/finally and throw
 * are implemented - see kotlin-test-try.kt.) **/

@Deprecated
public fun twice(x: Int): Int = x * 2

private fun classify(n: Int): Int {
    val m = n as Int              // as: identity cast (supported)
    return twice(m)
}

enum class Color { RED, GREEN }   // enum class: not implemented (declaration unused)

fun main() {
    var fails = 0
    if (twice(21) != 42) { fails = fails + 1 }
    if (classify(5) != 10) { fails = fails + 1 }
    if (!(twice(3) is Int)) { fails = fails + 1 }   // is: a genuine type test now
    exitProcess(fails)
}
