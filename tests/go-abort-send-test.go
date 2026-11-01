// ABORT PROBE: main sends on a full buffered channel after every goroutine has
// finished, so nothing can ever receive. go writes, on stderr with exit status 2,
// `fatal error: all goroutines are asleep - deadlock!` and a goroutine trace; the
// engines here write the same line and status without the trace. The sends before
// it, which a goroutine does receive, must not deadlock. See the send deadlock rows
// in .vscode/launch.json.
package main

import "fmt"

func main() {
	u := make(chan string)
	got := make(chan bool, 1)
	go func() {
		fmt.Println("received", <-u)
		got <- true
	}()
	u <- "handed over"
	<-got
	c := make(chan int, 1)
	c <- 1
	fmt.Println("before the deadlock", len(c))
	c <- 2
	fmt.Println("never reached")
}
