// Annotations in every form are parsed and ignored, so the annotated code still
// runs. Self-checking: main() exits with the failure count (0 == success). Both
// kotlin-interpreter.abnf and kotlin-to-llvm-ir.abnf run it and must agree.

@file:JvmName("AnnoTest")

@Suppress("UNUSED_PARAMETER")
fun square(n: Int): Int {
    return n * n
}

@Deprecated("use square", ReplaceWith("square(n)"))
fun oldSquare(n: Int): Int {
    return square(n)
}

// Annotations (with a use-site target and a private modifier) on constructor params.
class Point(@field:JvmField val x: Int, private val y: Int) {
    @get:JvmName("magnitude")
    fun mag(): Int {
        return x * x + y * y
    }
}

// An annotation array applying several at once.
@[Suppress("a") Suppress("b")]
fun tagged(): Int {
    return 42
}

fun main() {
    var fails = 0
    if (square(5) != 25) { fails = fails + 1 }
    if (oldSquare(4) != 16) { fails = fails + 1 }
    val p = Point(3, 4)
    if (p.mag() != 25) { fails = fails + 1 }
    if (tagged() != 42) { fails = fails + 1 }
    if (fails == 0) { println("kotlin annotations OK") }
    exitProcess(fails)
}
