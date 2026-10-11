// ABORT PROBE. This program is SUPPOSED to die, and its diagnostic text is the
// thing under test: the interpreter half, llvm.Run and the NATIVE binary must all
// produce the same abort message. See the "SHOULD ABORT" rows in
// .vscode/launch.json and abort_norm() in test.sh. The second deferred call
// recovers the panic and raises a NEW value with the same text, and the first one
// recovers that and raises it again unchanged; go (1.23+) reports
//
//	panic: abort probe [recovered]
//		panic: abort probe [recovered, repanicked]
package main

import "fmt"

func main() {
	fmt.Println("before the abort")
	defer func() {
		r := recover()
		panic(r)
	}()
	defer func() {
		r := recover()
		panic(fmt.Sprint(r))
	}()
	panic("abort probe")
}
