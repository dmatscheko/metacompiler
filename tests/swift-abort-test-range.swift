// ABORT PROBE. This program is SUPPOSED to die, and its diagnostic text is the
// thing under test. See the "SHOULD ABORT" rows in .vscode/launch.json and
// abort_norm() in test.sh.
//
// Forming a range whose lower bound exceeds its upper bound traps, before the
// range is used. An empty `3..<3` does not. swiftc 6.1.2 and all five legs
// answer "Fatal error: Range requires lowerBound <= upperBound".
let a = [3]
let empty = a[0]..<3
print("before the abort", empty, empty.isEmpty)
let r = a[0]..<1
print("never reached", r)
