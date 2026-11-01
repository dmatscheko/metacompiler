// ABORT PROBE: main waits in a select whose only arm can never become ready, with
// no default and no goroutine able to fill it. go writes, on stderr with exit
// status 2, `fatal error: all goroutines are asleep - deadlock!` and a goroutine
// trace; the engines here write the same line and status without the trace. See
// the deadlock rows in .vscode/launch.json.
package main

import "fmt"

func main() {
	c := make(chan int)
	go func() {
		fmt.Println("the goroutine ran")
	}()
	fmt.Println("before the deadlock")
	select {
	case v := <-c:
		fmt.Println("never received", v)
	}
	fmt.Println("never reached")
}
