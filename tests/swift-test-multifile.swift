// Multi-file Swift test: the Vec struct and the lengthSq function live in
// tests/imports/Geom.swift, found via the -i include root (mec -i tests/imports ...).
// 'import Geom' maps to Geom.swift; its declarations register like the main file's.
// 'import Foundation' is a builtin no-op import, mixed in on purpose.
//
// Top level code runs in source order; it counts failed checks and ends with
// exit(fails), so the run exits 0 exactly when every check passes. The interpreter
// and the LLVM-IR compiler must agree on every result.
import Foundation
import Geom

var fails = 0

func check(_ name: String, _ got: Int, _ want: Int) {
    if got != want {
        print("FAIL \(name): got \(got) want \(want)")
        fails += 1
    }
}

func main() {
    let a = Vec(x: 3, y: 4)
    let b = Vec(x: 2, y: -1)

    // Imported struct: construction and stored-property reads.
    check("imported ctor and fields", a.x + a.y, 7)

    // Imported instance method (implicit + explicit self).
    check("imported method dot", a.dot(b), 2)

    // Imported method that returns the imported type, then chained.
    let c = a.scaled(by: 2)
    check("imported method returns type", c.dot(b), 4)

    // Imported global function taking the imported type.
    check("imported global function", lengthSq(a), 25)

    if fails == 0 {
        print("swift multifile test passed")
    }
}

main()
exit(fails)
