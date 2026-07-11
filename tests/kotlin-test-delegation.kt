/* Kotlin delegation with `by`: the grammar now PARSES every delegated form a
 * real-world file uses - a property delegate (val/var x [: T] by <expr>, including
 * a trailing-lambda delegate like `by lazy { ... }` and a DI `by inject()`), the
 * same at local and top level, and interface delegation on a class (class C : I by
 * impl). Reading such a property dispatches through the delegate's getValue/setValue
 * protocol, which is outside the subset, so each is routed through notImpl. A plain
 * run therefore stops at the FIRST delegated property with a clean file:line message
 * (this file SHOULD fail by default). With -warn-unsupported each is warned and its
 * field simply omitted, while the ordinary properties and methods around them still
 * run and drive the self-check; main() ends with exitProcess(fails) so that run
 * exits 0 when the checks pass. Interpreter and compiler must agree byte-for-byte. */

package demo.delegation

import kotlin.system.exitProcess

// A trivial "provider" so the delegate expressions reference real functions; the
// delegate expression is parsed and discarded, so what it returns does not matter.
fun inject(): Int { return 0 }
fun provide(): Int { return 0 }

// Interface delegation (class C : I by impl): parsed and ignored - there is no
// inheritance in the subset, so the delegate expression is discarded and the class
// still lowers with its own primary-constructor property and methods.
class Service(val name: String) : Greeter by provide() {
    // A member property delegate in the Koin/DI style (by inject()): not implemented.
    private val hook: Int by inject()
    // A member property delegate with a trailing-lambda delegate (by lazy { ... }).
    val cached: Int by lazy { 40 + 2 }

    // Ordinary members are unaffected and drive the self-check.
    var calls: Int = 0
    fun ping(): Int {
        this.calls = this.calls + 1
        return this.calls
    }
}

// A top-level delegated property: same protocol gap, parsed and not implemented.
val banner: String by lazy { "hello" }

fun main() {
    var fails = 0

    val s = Service("svc")
    // The ordinary method still works even though the class carries delegated fields.
    if (s.ping() != 1) { fails = fails + 1 }
    if (s.ping() != 2) { fails = fails + 1 }
    if (s.name != "svc") { fails = fails + 1 }

    // A local delegated property (val x by lazy { ... }): not implemented; the check
    // below only uses ordinary locals, so the run still completes.
    val lazyLocal: Int by lazy { 7 }
    var total = 0
    for (i in 1..4) { total = total + i }
    if (total != 10) { fails = fails + 1 }

    if (fails == 0) { println("Kotlin delegation OK") }
    exitProcess(fails)
}
