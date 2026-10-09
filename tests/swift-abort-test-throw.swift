// ABORT PROBE. This program is SUPPOSED to die, and its diagnostic text is the
// thing under test. See the "SHOULD ABORT" rows in .vscode/launch.json and
// abort_norm() in test.sh.
//
// The uncaught-`throw` half of the pair; swift-abort-test.swift is the
// out-of-range half. All five legs answer
// "Fatal error: Error raised at top level: boom", so it carries the interpreter
// SHOULD FAIL row as well as the ALL HALVES one.
enum ProbeError: Error { case boom }

func raiseIt() throws { throw ProbeError.boom }

print("before the abort")
try raiseIt()
print("never reached")
