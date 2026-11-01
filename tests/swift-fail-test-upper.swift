// Swift undefined upper-case name. swiftc rejects it even when never reached:
//   swift-fail-test-upper.swift:16:11: error: cannot find 'Vector3' in scope
// No declaration of the program and no type of the standard library is named
// Vector3, so this file SHOULD fail in both halves before it prints anything.
// Everything before that line names a type of the program or of the stdlib.

struct Point { var x = 0, y = 0 }
enum Mode: Int { case on = 1 }

print("compiled")
let p = Point(x: 1, y: 2)
let ints = Array<Int>(repeating: Int.max, count: 1)
print(p.x, Mode.on.rawValue, ints.count, MemoryLayout<Int>.size, String(3))
let skip = false
if skip {
    print(Vector3(x: 1))
}
