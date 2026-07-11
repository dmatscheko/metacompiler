/* Kotlin anonymous object expressions: object [: Supertypes] { members } used in
 * value position - the shape a lot of real-world Kotlin uses to pass an ad-hoc
 * listener/component to a framework call, e.g.
 *
 *     manager.bindComponentToCustomLifecycle(object : IHasComponent<Navigator> {
 *         override fun getComponent(): Navigator = Navigator()
 *     })
 *
 * The grammar now PARSES such an expression (a generic supertype and the member body
 * are accepted). Anonymous-class method dispatch is outside the subset, so it is
 * routed through notImpl: a plain run stops at the object expression with a clean
 * file:line message (this file SHOULD fail by default), while -warn-unsupported warns,
 * yields undefined for the object, and lets the surrounding real code drive the
 * self-check. main() ends with exitProcess(fails) so that run exits 0 when the checks
 * pass; the interpreter and compiler must agree byte-for-byte. */

package demo.objectexpr

import kotlin.system.exitProcess

// A tiny generic interface name and a class the object "implements"/returns; only the
// ordinary code below actually runs.
class Navigator(val id: Int) {
    fun where(): Int { return this.id }
}

// A framework-style sink: it just takes the (ignored) component and returns a marker.
fun register(component: Int): Int { return 100 }

fun main() {
    var fails = 0

    // An anonymous object passed straight as a call argument - the reported real-world
    // shape. It is not implemented, so the argument evaluates to undefined, but the
    // call itself still lowers.
    register(object : Comparable<Navigator> {
        override fun compareTo(other: Navigator): Int = 0
    })

    // An object expression bound to a local, with a generic supertype and an
    // overriding member returning a value.
    val provider = object : Comparable<Navigator> {
        override fun compareTo(other: Navigator): Int = other.where()
    }

    // Ordinary code around the object expression runs and self-checks.
    val nav = Navigator(7)
    if (nav.where() != 7) { fails = fails + 1 }
    var sum = 0
    for (i in 1..5) { sum = sum + i }
    if (sum != 15) { fails = fails + 1 }

    if (fails == 0) { println("Kotlin object expression OK") }
    exitProcess(fails)
}
