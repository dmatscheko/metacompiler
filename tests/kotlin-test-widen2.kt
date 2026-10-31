/* Kotlin widened surface v2: type/member declarations this grammar pair recognizes
 * (interface, object, enum class, annotation class, typealias, companion object, init
 * block, secondary constructor, property accessor, destructuring), plus the
 * multi-statement lambda body, the extension DELEGATED property and the multi-argument
 * indexed access - all of them genuinely lowered, so the lowered forms self-check with
 * and without -warn-unsupported. main() ends with exitProcess(fails), so a run exits 0
 * when they pass.
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
// empty object - the body is optional, not required.
interface Marker

interface Tagged : Marker

object Singleton

// A top-level extension property (receiver + get() accessor).
val Int.asMarker: String
    get() = "marker"

// A top-level extension DELEGATED property (receiver + a `by` delegate). The delegate
// object belongs to the DECLARATION site (storeDelegate("m") runs ONCE, here), and every
// read calls its getValue with the EXTENSION RECEIVER as thisRef - which is what `stamp`
// records.
class StoreDelegate(val tag: String) {
    var reads = 0
    operator fun getValue(thisRef: Any?, property: KProperty<*>): String {
        reads = reads + 1
        return tag + ":" + thisRef.toString() + ":" + property.name
    }
}
fun storeDelegate(tag: String) = StoreDelegate(tag)
val Int.viaStore: String by storeDelegate("m")

// A MULTI-ARGUMENT indexed access (operator get(a, b)) in a top-level property
// initializer.
class Grid { operator fun get(r: Int, c: Int) = r * 10 + c }
val cell = Grid()[1, 2]

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

    // destructuring declaration
    val (p, q) = listOf(10, 20)

    // the extension delegated property: thisRef is the RECEIVER of each read, and the
    // one delegate object is shared by every read (so `reads` counts across receivers).
    if (7.viaStore != "m:7:viaStore") { fails = fails + 1 }
    if (9.viaStore != "m:9:viaStore") { fails = fails + 1 }
    if (cell != 12) { fails = fails + 1 }

    exitProcess(fails)
}
