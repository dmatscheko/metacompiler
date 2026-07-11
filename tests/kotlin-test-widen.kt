/* Kotlin widened syntax that is PARSED but not fully lowered. Modifiers and
 * annotations (no args) are parsed and ignored; `as` is an identity cast (supported);
 * the `is` type check is not implemented (undecidable without runtime types). Without
 * flags the compile stops at the first unimplemented construct with a clean file:line
 * message (this file SHOULD fail by default). With -warn-unsupported `is` is warned and
 * replaced by a placeholder, so the program runs and self-checks: main() ends with
 * exitProcess(fails). (try/catch/finally and throw are now implemented - see
 * kotlin-test-try.kt.) **/

@Deprecated
public fun twice(x: Int): Int = x * 2

private fun classify(n: Int): Int {
    val m = n as Int              // as: identity cast (supported)
    return twice(m)
}

fun main() {
    var fails = 0
    val probe = twice(3) is Int   // is: not implemented (result unused)
    if (twice(21) != 42) { fails = fails + 1 }
    if (classify(5) != 10) { fails = fails + 1 }
    exitProcess(fails)
}
