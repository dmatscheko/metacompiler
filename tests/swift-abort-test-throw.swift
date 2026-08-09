// ABORT PROBE. This program is SUPPOSED to die, and its diagnostic text is the
// thing under test. See the "SHOULD ABORT" rows in .vscode/launch.json and
// abort_norm() in test.sh.
enum ProbeError: Error { case boom }

func raiseIt() throws { throw ProbeError.boom }

print("before the abort")
try raiseIt()
print("never reached")
