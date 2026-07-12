// Multi-file Kotlin test: geo.Vec lives in tests/imports/geo/Vec.kt and is found
// via the -i include root (mec -i tests/imports ...). The imported file is parsed
// with the same grammar; its class, function and top-level property register like
// the main file's. kotlin.math is a builtin no-op import, mixed in on purpose.
import kotlin.math.abs
import geo.Vec

var fails = 0

fun check(id: String, cond: Boolean) {
    if (!cond) {
        println("FAIL " + id)
        fails = fails + 1
    }
}

fun main() {
    val a = Vec(3, 4)
    val b = Vec(2, -1)
    check("imported class", a.dot(b) == 2)
    check("imported fun", vecScale(a, 2).dot(b) == 4)
    check("imported property", geoOrigin.x == 0 && geoOrigin.y == 0)
    check("builtin still works", abs(-7) == 7)

    if (fails == 0) {
        println("kotlin multifile test passed")
    }
    exitProcess(fails)
}
