/* Kotlin delegation with `by`. The grammar PARSES every delegated form a real-world
 * file uses. `by lazy { ... }` is special-cased: its initializer is run once and the
 * value becomes a plain field/binding (evaluated eagerly at the declaration -
 * single-threaded and deterministic - not on first access). Every other delegate goes
 * through the getValue/setValue protocol, and a `by` expression with no getValue - the
 * DI-style `by inject()` below, which answers a plain Int - is reported as not
 * implemented when the declaration runs, so a plain run stops there with a clean
 * file:line message (this file SHOULD fail by default). With -warn-unsupported it is
 * warned and its field omitted, while the lazy properties and ordinary members drive
 * the self-check; main() ends with exitProcess(fails) so that run exits 0 when the
 * checks pass. Interpreter and compiler must agree byte-for-byte. */

package demo.delegation

import kotlin.system.exitProcess

// A trivial "provider" so the delegate expressions reference real functions; neither
// answers a delegate.
fun inject(): Int { return 0 }
fun provide(): Int { return 0 }

// Interface delegation (class C : I by impl) against an interface this file does not
// declare: the class still lowers with its own primary-constructor property and
// methods.
class Service(val name: String) : Greeter by provide() {
    // A member property delegate in the Koin/DI style (by inject()): not a delegate.
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
