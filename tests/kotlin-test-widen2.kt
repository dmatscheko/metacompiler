/* Kotlin widened surface v2: type/member declarations we now PARSE but do not lower
 * (interface, object, enum class, annotation class, typealias, companion object, init
 * block, secondary constructor, property accessor, destructuring), plus a construct we
 * GENUINELY lower - the multi-statement lambda body. The declarations are reported via
 * notImpl: without flags the run stops at the FIRST one (the interface) with a clean
 * file:line message (this file SHOULD fail by default); with -warn-unsupported each is
 * warned, dropped, and the rest runs, so the genuinely lowered multi-statement lambdas
 * self-check. main() ends with exitProcess(fails), so that run exits 0 when they pass.
 * SkipBlock is stressed inside the skipped bodies: nested braces, a string holding a
 * lone } and a string template ${...}, and a char literal. The marker interface and
 * empty object below exercise the OPTIONAL body (a bodiless declaration - no { } at all). **/

interface Shape {
    fun area(): Int
    fun name(): String
}

object Registry {
    val tag = "reg}${1 + 2}"              // a string with a } and a ${...} template
    val sep = '}'                         // a char literal holding a brace
    fun describe(): String {
        val nested = { x: Int -> { x } }  // nested braces inside the skipped body
        return "registry"
    }

    companion object {
        val VERSION = 1
    }
}

enum class Color(val rgb: Int) {
    RED(0xff0000), GREEN(0x00ff00), BLUE(0x0000ff)
}

annotation class Marker

typealias IntList = List<Int>

// Bodiless declarations (no { } body): a marker interface, one with a supertype, and an
// empty object. Each still parses and is reported via notImpl, exactly like the bodied
// interface/object above - the body is optional, not required.
interface Marker

interface Tagged : Marker

object Singleton

// A top-level extension property (receiver + get() accessor). Both the receiver and the
// accessor are recognised and notImpl - the flat model has no extension/computed props.
val Int.asMarker: String
    get() = "marker"

// A top-level extension DELEGATED property (receiver + a `by` delegate): both recognised
// and notImpl. The delegate expression is parsed and discarded (never evaluated).
val Int.viaStore: String by storeDelegate("m")

class WithExtras(val n: Int) {
    var doubled: Int = 0
    init {
        doubled = n * 2
    }
    constructor() : this(0) { }
    constructor(s: String) : this(s.length)   // bodiless: pure delegation, no { }
    val label: String
        get() = "n=$n"

    companion object {
        val ORIGIN = 0
    }
}

fun main() {
    var fails = 0

    // multi-statement lambda: genuinely lowered, so it actually runs
    val transform = { x: Int ->
        val a = x + 1
        val b = a * 2
        b - 3
    }
    if (transform(10) != 19) { fails = fails + 1 }

    // multi-statement lambda whose tail is an if-expression (value preserved)
    val classify = { x: Int ->
        val doubled = x * 2
        if (doubled > 10) 1 else 0
    }
    if (classify(7) != 1) { fails = fails + 1 }
    if (classify(3) != 0) { fails = fails + 1 }

    // multi-statement lambda as a list higher-order argument
    val nums = listOf(1, 2, 3, 4)
    val mapped = nums.map { it ->
        val sq = it * it
        sq + 1
    }
    if (mapped[3] != 17) { fails = fails + 1 }

    // destructuring declaration: not implemented; its initializer still runs
    val (p, q) = listOf(10, 20)

    exitProcess(fails)
}
