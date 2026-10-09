// See mod.kt.
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
