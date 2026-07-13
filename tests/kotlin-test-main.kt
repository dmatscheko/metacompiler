/* Exercises the -main flag, which names the entry-point function mec calls instead of
 * main(). The real entry here is checkAll(); main() is present but must NOT run (it would
 * print a failing marker and exit 1). Run as:
 *   mec languages/kotlin-interpreter.abnf tests/kotlin-test-main.kt -q -main checkAll
 * checkAll() self-checks and ends with exitProcess(fails), so a passing run exits 0 with
 * byte-identical output on all four legs (interpreter/compiler x goja/-frozen). */

fun checkAll() {
    var fails = 0
    if (2 + 3 != 5) { fails = fails + 1 }
    val xs = listOf(1, 2, 3)
    if (xs.size != 3) { fails = fails + 1 }
    if (xs.map { it * 2 }.sumOf { it } != 12) { fails = fails + 1 }
    println("custom entry point ran")
    exitProcess(fails)
}

fun main() {
    println("BUG: main() ran but -main checkAll was requested")
    exitProcess(1)
}
