/* Kotlin widened syntax. Modifiers and annotations (no args) are parsed and
 * ignored; `as` casts, `is` type tests, raw strings and `enum class` are all
 * genuine now, in the interpreter and the compiler alike, so the file runs and
 * self-checks with or without -warn-unsupported: main() ends with
 * exitProcess(fails). (try/catch/finally and throw are implemented - see
 * kotlin-test-try.kt.)
 *
 * History: this file used to be a by-design SHOULD-FAIL guard, because the
 * `enum class` below was recognised but not lowered and aborted a flagless run.
 * The full-syntax work implemented it, which removed the guard's premise, so the
 * matrix entry became an ordinary one. The still-red guards are widen2 (which
 * keeps a genuinely unimplemented declaration) and the recognition tests. **/

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
