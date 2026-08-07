// See mod.kt. Same loop, so the languages are comparable to each other.
//
// js and typescript had NO bench row at all until this file, and docs/todo.md 7.7
// records what that cost: 673a8b1 lost ~20% on a yield*-heavy workload (5.35 s ->
// 6.40 s over 300,000 steps) and no gate in the project could see it. The loop is
// deliberately the same one every other row runs - no regexp, no float, no
// allocation in the body - so a js number moved by a change to the js value model
// is comparable to the kotlin and python numbers beside it.
function main() {
    var s = 0
    var i = 0
    while (i < 40000) {
        s = s + i % 7
        i = i + 1
    }
    println(s)
    return 0
}
