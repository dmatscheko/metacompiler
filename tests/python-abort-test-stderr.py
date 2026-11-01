# ABORT PROBE. This program is SUPPOSED to exit non-zero, and what it writes on
# standard error is the thing under test: the fragment without a newline must
# reach the stream before SystemExit's message, in the interpreter half,
# llvm.Run and the NATIVE binary alike. See the "SHOULD ABORT" rows in
# .vscode/launch.json and abort_norm() in test.sh.
import sys

print("before the exit")
sys.stderr.write("stderr fragment, ")
raise SystemExit("exit message")
