// ABORT PROBE. This program is SUPPOSED to die, and its diagnostic text is the
// thing under test. See the "SHOULD ABORT" rows in .vscode/launch.json and
// abort_norm() in test.sh.
//
// Swift has TWO abort probes: this one (an array index out of range, raised
// inside the runtime) and swift-abort-test-throw.swift (an uncaught `throw`,
// raised by the program). They reach the diagnostic by different routes and both
// are worth a row.
//
// Both carry the FULL PAIR of rows since docs/todo.md 6.2 - an interpreter
// SHOULD FAIL row for the goja-vs-frozen byte-identity check, and a native
// SHOULD ABORT, ALL HALVES row that compares the abort TEXT across all three
// implementations. This file's header used to say the compiler halves diverged
// ("uncaught exception: list index out of range" under llvm.Run against "list
// index out of range" natively, and "uncaught exception: 9" against "KeyError:
// 9" on a missing dictionary key). Measured at 22e7b44: all five legs already
// answered "Fatal error: Index out of range", so the divergence had been closed
// and only this comment still claimed it.
print("before the abort")
let xs = [1, 2, 3]
print(xs[7])
print("never reached")
