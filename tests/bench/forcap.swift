// forcap.swift - the FOR-LOOP cost of a per-iteration loop-variable binding.
//
// The exact counterpart of tests/bench/forcap.kt, and it exists for the same
// reason. tests/bench/mod.swift is the shared harness every language has -
// `s = s + i % 7`, 40,000 iterations - and swift's copy uses a WHILE loop, so
// when makeForRange / makeForList / makeForCase in
// languages/swift-to-llvm-ir.abnf started opening a fresh scope per iteration
// (Swift binds the element with a `let` inside the body, which is what makes
// `for i in 0..<3 { fs.append({ i }) }` answer 0 / 2 instead of 3 / 3 -
// docs/todo.md 1.1), tests/bench.sh could not see the cost in EITHER direction.
// This program is the same loop written with `for`, so the difference between it
// and mod.swift is exactly one js_scope_new + one js_scope_decl per iteration
// against one js_scope_set.
//
// It is deliberately NOT called mod.<ext>: bench.sh globs tests/bench/mod.* and
// this file must not join the shared gate, whose baseline is per language and
// whose rows are meant to be comparable across languages. Measure it by hand,
// the way bench.sh does - min of N runs per build, median over several -exe
// output PATH LENGTHS, and quote the spread beside the median, because a native
// binary's instruction count moves ~4.8% with nothing but its output filename
// (docs/working-on-this-project.md chapter 4) - and corroborate with the exact
// counters, which need no draws: MEC_GC_STATS collections, live bytes, and the
// binary's size.
//
// Three loop shapes, because they lower differently: the RANGE loop is
// makeForRange (a counter and a bound in the loop's own scope), the LIST loop is
// makeForList (an index and a materialized sequence) and the WILDCARD loop is
// makeForRange's `for _ in` arm, which binds no name and therefore opens no
// per-iteration scope at all - it is the row that must not have moved.
func main() {
    var s = 0
    for i in 0..<40000 {
        s = s + i % 7
    }
    // Array(0..<n), not a while loop of append(): swift's array is a VALUE, so
    // `xs.append(k)` 40,000 times is quadratic here and it buried the thing being
    // measured - 15.2 billion instructions for a program whose two loops are worth
    // about 30 million.
    let xs = Array(0..<40000)
    var t = 0
    for x in xs {
        t = t + x % 7
    }
    var u = 0
    for _ in 0..<40000 {
        u = u + 1
    }
    print(s + t + u)
}

main()
