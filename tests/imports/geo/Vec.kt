// A project module imported by tests/kotlin-test-multifile.kt (via -i tests/imports).
package geo

class Vec(val x: Int, val y: Int) {
    fun dot(o: Vec): Int {
        return x * o.x + y * o.y
    }
}

fun vecScale(v: Vec, f: Int): Vec {
    return Vec(v.x * f, v.y * f)
}

val geoOrigin = Vec(0, 0)
