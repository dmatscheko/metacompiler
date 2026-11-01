// ABORT PROBE: main waits in a select whose only arm can never become ready, with
// no default and no goroutine able to fill it. go writes, on stderr with exit
// status 2, `fatal error: all goroutines are asleep - deadlock!`; the engines here
// name the same deadlock in their own abort format. See the "SHOULD ABORT" rows in
// .vscode/launch.json.
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
