// ABORT PROBE. This program is SUPPOSED to die, and its diagnostic text is the
// thing under test. See the "SHOULD ABORT" rows in .vscode/launch.json and
// abort_norm() in test.sh.
//
// removeLast() on an empty array traps (popLast() would answer nil). swiftc
// 6.1.2 and all five legs answer "Fatal error: Can't remove last element from
// an empty collection".
var e = [4]
print("before the abort", e.removeLast(), e.popLast() as Any)
e.removeLast()
print("never reached", e)
