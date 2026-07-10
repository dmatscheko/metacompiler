/* Kotlin import handling: a package header plus imports that resolve against the
 * runtime builtins (kotlin.math.abs/max, kotlin.system.exitProcess, the
 * kotlin.collections list API). All of these are already provided, so the imports
 * compile as no-ops and the program self-checks. main() ends with exitProcess,
 * so the run exits 0 exactly when every check passes. **/

package demo.app

import kotlin.math.abs
import kotlin.math.max
import kotlin.collections.*
import kotlin.system.exitProcess

var fails = 0

fun check(name: String, got: Int, want: Int) {
    if (got != want) {
        println("FAIL $name: got $got want $want")
        fails++
    }
}

fun main() {
    check("abs", abs(-7), 7)
    check("max", max(3, 9), 9)
    val xs = listOf(10, 20, 30)
    check("size", xs.size, 3)
    check("get", xs.get(1), 20)
    exitProcess(fails)
}
