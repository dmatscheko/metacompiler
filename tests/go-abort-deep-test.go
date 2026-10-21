// ABORT PROBE. This program is SUPPOSED to die, and its diagnostic text is the
// thing under test: the interpreter half, llvm.Run and the NATIVE binary must all
// produce the same abort message. See the "SHOULD ABORT" rows in
// .vscode/launch.json and abort_norm() in test.sh. A recursion with no base case
// is a fatal error in go, not a panic: no deferred recover runs, and go 1.27
// writes (before its goroutine trace, which is not reproduced), exit status 2:
//
//	runtime: goroutine stack exceeds 1000000000-byte limit
//	fatal error: stack overflow
package main

import "fmt"

func rec(n int) int { return rec(n+1) + 1 }

func main() {
	fmt.Println("before the abort")
	defer func() { fmt.Println("recovered", recover()) }()
	fmt.Println(rec(0))
}
