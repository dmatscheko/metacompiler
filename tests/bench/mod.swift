// See mod.kt. Same loop, so the languages are comparable to each other.
//
// swift had NO bench row at all until docs/todo.md 1.2, which means no gate ever
// loaded languages/lib/swift-rt.ll - and that item's own fix puts js_swadoptdeep
// on a per-call path, which is the documented +10.3% charAt trap of a68e16d.
func main() {
    var s = 0
    var i = 0
    while i < 40000 {
        s = s + i % 7
        i = i + 1
    }
    print(s)
}

main()
