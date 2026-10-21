// ABORT PROBE. This program is SUPPOSED to die. See the "SHOULD ABORT" rows in
// .vscode/launch.json and abort_norm() in test.sh. A recursion with no base case
// crashes a swift program: swiftc 6.1.2's binary dies of SIGSEGV (exit status
// 139) and writes nothing on stderr, which all three halves reproduce - the
// output before the crash included, which a real binary loses only when its
// stdout is a pipe it had not flushed.
func rec(_ n: Int) -> Int { return rec(n + 1) + 1 }
print("before the abort")
print(rec(0))
