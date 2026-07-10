/* Kotlin widened syntax: constructs we PARSE but do not fully lower. Modifiers and
 * annotations (no args) are parsed and ignored; `as` is an identity cast; try/catch/
 * finally, throw and `is` are not implemented. Without flags the compile stops at the
 * first unimplemented construct with a clean file:line message (this file SHOULD fail
 * by default). With -warn-unsupported each is warned and replaced by a placeholder -
 * the try block is still compiled and the thrown/`is` expressions still evaluated -
 * so the program runs and self-checks: main() ends with exitProcess(fails). **/

@Deprecated
public fun twice(x: Int): Int = x * 2

private fun classify(n: Int): Int {
    val m = n as Int              // as: identity cast (supported)
    try {
        return twice(m)           // try body is compiled under -warn-unsupported
    } catch (e: Exception) {
        return -1                 // catch is dropped
    } finally {
        println("finally ignored")
    }
}

private fun risky(n: Int): Int {
    if (n < 0) throw RuntimeException("neg")   // throw: not taken for n >= 0
    return n
}

fun main() {
    var fails = 0
    val probe = twice(3) is Int   // is: not implemented (result unused)
    if (twice(21) != 42) { fails = fails + 1 }
    if (classify(5) != 10) { fails = fails + 1 }
    if (risky(7) != 7) { fails = fails + 1 }
    exitProcess(fails)
}
