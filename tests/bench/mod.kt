// The bench loop for tests/bench.sh: no regexp, no float and no allocation in
// the body. Every mod.* file runs the same loop.
fun main() {
    var s = 0
    var i = 0
    while (i < 40000) {
        s = s + i % 7
        i = i + 1
    }
    println(s)
}
