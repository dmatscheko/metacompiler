// A hot loop with no regexp, no float and no allocation in the body: this is the
// harness every perf measurement in docs/runtime-merge-plan.md uses.
fun main() {
    var s = 0
    var i = 0
    while (i < 40000) {
        s = s + i % 7
        i = i + 1
    }
    println(s)
}
