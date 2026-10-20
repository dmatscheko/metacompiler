/* Kotlin import handling: a package header plus imports that resolve against the
 * runtime builtins (kotlin.math.abs/max, kotlin.system.exitProcess, the
 * kotlin.collections list API). All of these are already provided, so the imports
 * compile as no-ops and the program self-checks. `import java.util.Random` binds
 * Random to the JVM generator (java: new Random(1).nextInt(100) is 85, then 88).
 * The java.util collections kotlin.collections typealiases (ArrayList, HashMap,
 * HashSet, LinkedHashMap, LinkedHashSet) import as the same constructors, by name,
 * under an alias, or through `java.util.*`.
 * main() ends with exitProcess, so the run exits 0 exactly when every check
 * passes. **/

package demo.app

import kotlin.math.abs
import kotlin.math.max
import kotlin.collections.*
import kotlin.system.exitProcess
import java.util.Random
import java.util.ArrayList
import java.util.HashMap
import java.util.LinkedHashSet as LHS
import java.util.*

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
    val r = Random(1)
    check("jrandom1", r.nextInt(100), 85)
    check("jrandom2", r.nextInt(100), 88)
    val al = ArrayList<Int>(xs)
    al.add(40)
    check("jarraylist", al.size, 4)
    val hm = HashMap<String, Int>()
    hm["a"] = 1
    check("jhashmap", hm.getValue("a"), 1)
    val ls = LHS<Int>(listOf(2, 2, 3))
    check("jlinkedhashset", ls.size, 2)
    val lm = LinkedHashMap<Int, Int>()
    lm[1] = 5
    check("jstarimport", lm.getValue(1) + HashSet<Int>(listOf(1, 1)).size, 6)
    exitProcess(fails)
}
