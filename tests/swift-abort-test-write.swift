// ABORT PROBE. This program is SUPPOSED to die, and its diagnostic text is the
// thing under test. See the "SHOULD ABORT" rows in .vscode/launch.json and
// abort_norm() in test.sh.
//
// The subscript-WRITE half of swift-abort-test.swift: an Array element write
// out of range traps like the read does - a negative index does not wrap from
// the end, and a write past the end does not grow the array. swiftc 6.1.2 and
// all five legs answer "Fatal error: Index out of range".
var a = [1, 2]
let idx = [2, 5, -1]
a[1] = 7
print("before the abort", a)
a[idx[2]] = 9
print("never reached", a)
