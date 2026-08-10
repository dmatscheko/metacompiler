// idxhot.kt - the cost of the BOUNDS CHECK on Kotlin's indexed read.
//
// WHY THIS FILE EXISTS AND WHY IT IS NOT mod.kt. tests/bench/mod.kt is the shared
// harness every language has - `s = s + i % 7`, 40,000 iterations - and it does
// not index anything at all, so it cannot see the cost of a check inside
// js_ktindex in EITHER direction. docs/todo.md 2.9 is exactly that change: the
// out-of-range list/array read used to answer kotlin.Unit and the out-of-range
// string read used to abort, and both now raise a catchable throwable, which
// costs two comparisons and a length read per indexed access.
//
// It is deliberately NOT called mod.<ext>, for the same reason forcap.kt is not:
// bench.sh globs tests/bench/mod.* and this file must not join the shared gate,
// whose baseline is per language and whose rows are meant to be comparable
// ACROSS languages. Measure it by hand the way bench.sh does - min of N runs per
// build, median over several -exe output PATH LENGTHS, and quote the spread
// beside the median, because a native binary's instruction count moves ~4.8% with
// nothing but its output filename (docs/working-on-this-project.md chapter 4).
//
// Three loops, because they lower to three different reads:
//   the LIST read   a[i]  -> js_ktindex's ktIsArr arm  (the new length compare)
//   the STRING read s[i]  -> js_ktindex's string arm   (the compare that used to
//                            be there already, as a fail() rather than a raise)
//   the CONTROL           -> mod.kt's body, so a run of this file also carries a
//                            row that the change cannot touch.
// 40,000 iterations each, matching mod.kt.
fun main() {
    val xs = mutableListOf<Int>()
    var k = 0
    while (k < 200) {
        xs.add(k)
        k = k + 1
    }
    val s = "0123456789012345678901234567890123456789"
    var a = 0
    var i = 0
    while (i < 40000) {
        a = a + xs[i % 200]
        i = i + 1
    }
    var b = 0
    var j = 0
    while (j < 40000) {
        b = b + s[j % 40].code
        j = j + 1
    }
    var c = 0
    var n = 0
    while (n < 40000) {
        c = c + n % 7
        n = n + 1
    }
    println(a + b + c)
}
