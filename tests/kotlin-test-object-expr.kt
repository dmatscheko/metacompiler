/* Kotlin anonymous object expressions: object [: Supertypes] { members } used in
 * value position - the shape a lot of real-world Kotlin uses to pass an ad-hoc
 * listener/component to a framework call, e.g.
 *
 *     manager.bindComponentToCustomLifecycle(object : IHasComponent<Navigator> {
 *         override fun getComponent(): Navigator = Navigator()
 *     })
 *
 * These are GENUINE now: each evaluation builds a fresh instance whose descriptor
 * holds the methods and whose fields are the initialized properties; methods and
 * property initializers close over the expression site's scope (like a lambda) and
 * see the new object as `this`. Supertypes (including generic ones) are parsed and
 * ignored - no inheritance, but passing the object where an interface is expected
 * works because method calls dispatch by name. main() ends with exitProcess(fails);
 * exit 0 and byte-identical output on all four legs mean everything passed. */

package demo.objectexpr

import kotlin.system.exitProcess

class Navigator(val id: Int) {
    fun where(): Int { return this.id }
}

// A framework-style sink: calls the component's method like a listener registry would.
fun fire(handler: Int): Int { return handler + 100 }

fun main() {
    var fails = 0

    // An anonymous object bound to a local, with a generic supertype and an
    // overriding member calling through to another object.
    val provider = object : Comparable<Navigator> {
        override fun compareTo(other: Navigator): Int = other.where()
    }
    val nav = Navigator(7)
    if (provider.compareTo(nav) != 7) { fails = fails + 1 }

    // Properties + methods with implicit member access and per-instance state.
    val counter = object {
        var count = 0
        fun bump(): Int {
            count = count + 1
            return count
        }
    }
    counter.bump()
    if (counter.bump() != 2) { fails = fails + 1 }
    if (counter.count != 2) { fails = fails + 1 }

    // Methods capture enclosing locals (like a lambda); the result feeds a call.
    val base = 40
    val handler = object {
        fun value(): Int = base + 2
    }
    if (fire(handler.value()) != 142) { fails = fails + 1 }

    if (fails == 0) { println("Kotlin object expression OK") }
    exitProcess(fails)
}
