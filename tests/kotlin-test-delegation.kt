/* Kotlin delegation with `by`. The grammar PARSES every delegated form a real-world
 * file uses, and the `by lazy { ... }` form is now GENUINE: its initializer is run
 * once and the value becomes a plain field/binding (evaluated eagerly at the
 * declaration - single-threaded and deterministic - not on first access). Every OTHER
 * delegate still needs the getValue/setValue protocol, which is outside the subset:
 * a DI `by inject()`, a custom delegate, and interface delegation (class C : I by impl,
 * which parses and is ignored). Those route through notImpl, so a plain run stops at
 * the FIRST such delegate with a clean file:line message (this file SHOULD fail by
 * default). With -warn-unsupported each unsupported delegate is warned and its field
 * omitted, while the lazy properties and ordinary members drive the self-check;
 * main() ends with exitProcess(fails) so that run exits 0 when the checks pass.
 * Interpreter and compiler must agree byte-for-byte. */

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
    // A member property delegate with a trailing-lambda delegate (by lazy { ... }):
    // GENUINE - the initializer runs once and `cached` becomes an ordinary field.
    val cached: Int by lazy { 40 + 2 }

    // Ordinary members are unaffected and drive the self-check.
    var calls: Int = 0
    fun ping(): Int {
        this.calls = this.calls + 1
        return this.calls
    }
}

// A top-level lazy property: genuine.
val banner: String by lazy { "hel" + "lo" }

fun main() {
    var fails = 0

    val s = Service("svc")
    // The ordinary method still works even though the class carries delegated fields.
    if (s.ping() != 1) { fails = fails + 1 }
    if (s.ping() != 2) { fails = fails + 1 }
    if (s.name != "svc") { fails = fails + 1 }
    // the genuine lazy member computed its value
    if (s.cached != 42) { fails = fails + 1 }
    // the genuine top-level lazy property
    if (banner != "hello") { fails = fails + 1 }

    // A local lazy property (val x by lazy { ... }): genuine.
    val lazyLocal: Int by lazy { 3 + 4 }
    if (lazyLocal != 7) { fails = fails + 1 }
    var total = 0
    for (i in 1..4) { total = total + i }
    if (total != 10) { fails = fails + 1 }

    if (fails == 0) { println("Kotlin delegation OK") }
    exitProcess(fails)
}
