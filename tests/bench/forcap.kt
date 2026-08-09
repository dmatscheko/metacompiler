// forcap.kt - the FOR-LOOP cost of a per-iteration loop-variable binding.
//
// WHY THIS FILE EXISTS AND WHY IT IS NOT mod.kt. tests/bench/mod.kt is the shared
// harness every language has - `s = s + i % 7`, 40,000 iterations - and kotlin's
// copy uses a WHILE loop. So when makeForRange / makeForList / makeForDestruct in
// languages/kotlin-to-llvm-ir.abnf started allocating a fresh scope per iteration
// (the Kotlin specification's `val <name> = iterator.next()` inside the body, which
// is what makes `for (i in 1..3) { fs.add({ i }) }` answer "1 2 3" instead of
// "3 3 3"), tests/bench.sh could not see the cost in EITHER direction. This
// program is the same loop written with `for`, so the difference between it and
// mod.kt is exactly one js_scope_new + one js_scope_decl per iteration against one
// js_scope_set.
//
// It is deliberately NOT called mod.<ext>: bench.sh globs tests/bench/mod.* and
// this file must not join the shared gate, whose baseline is per language and
// whose rows are meant to be comparable across languages. Measure it by hand,
// the way bench.sh does - min of N runs per build, median over several -exe
// output PATH LENGTHS, and quote the spread beside the median, because a native
// binary's instruction count moves ~4.8% with nothing but its output filename
// (docs/working-on-this-project.md chapter 4).
//
// Two loop shapes, because they lower differently: the RANGE loop is
// makeForRange (a counter and a bound in the loop's own scope) and the LIST loop
// is makeForList (an index and a materialized sequence). 40,000 iterations each,
// matching mod.kt.
fun main() {
    var s = 0
    for (i in 0 until 40000) {
        s = s + i % 7
    }
    val xs = mutableListOf<Int>()
    var k = 0
    while (k < 40000) {
        xs.add(k)
        k = k + 1
    }
    var t = 0
    for (x in xs) {
        t = t + x % 7
    }
    println(s + t)
}
