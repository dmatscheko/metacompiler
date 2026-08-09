// ABORT PROBE. This program is SUPPOSED to die, and its diagnostic text is the
// thing under test. See the "SHOULD ABORT" rows in .vscode/launch.json and
// abort_norm() in test.sh.
//
// Swift has TWO abort probes because neither one alone is green on all three
// implementations today, and the split keeps what IS gateable gated:
//
//   * this file (array index out of range) is the INTERPRETER row. Its compiler
//     halves diverge from each other - llvm.Run says "uncaught exception: list
//     index out of range" where the native binary says "list index out of range"
//     (and on a missing dictionary key, "uncaught exception: 9" vs "KeyError: 9").
//   * swift-abort-test-throw.swift (an uncaught `throw`) is the NATIVE row. Its
//     two compiler halves agree; its INTERPRETER half does not produce a
//     diagnostic at all - it dumps the engine's own stack, differently under goja
//     and under -frozen.
//
// Both are reported in docs/todo.md 4.1's round; promote the rows when a swift
// owner closes them.
print("before the abort")
let xs = [1, 2, 3]
print(xs[7])
print("never reached")
