//go:build ignore

// Go subset widening test.
//
// EVERY construct here is implemented in both grammars: the type assertion x.(T), the
// type switch, the interface declaration (satisfaction is structural), the goroutine,
// the channel make/send/receive, and the select. `go work(5, done)` really runs, so tally
// reaches 6 via a genuine goroutine.
//
// The file runs to os.Exit(fails) with fails == 0, with or without -warn-unsupported,
// and a run with the flag reports no unsupported construct at all. It is an ordinary
// matrix entry, not a SHOULD-FAIL guard.

package main

import (
	"fmt"
	"os"
)

var fails = 0

func check(name string, got int, want int) {
	if got != want {
		fmt.Println("FAIL", name, "got", got, "want", want)
		fails++
	}
}

// An interface type declaration.
type Stringer interface {
	String() string
}

var tally = 0

// A goroutine callee (it stays visible in call graphs); done orders it before the
// read of tally, which real go needs.
func work(n int, done chan int) {
	tally += n
	done <- 1
}

// A boxed value comes in as interface{}; the assertion recovers it unchanged.
func unbox(x interface{}) int {
	return x.(int)
}

func main() {
	// GENUINE: a type assertion on an interface value answers the dynamic value.
	check("assert via param", unbox(42), 42)
	var a interface{} = 7
	check("assert direct", a.(int), 7)
	check("assert in expr", a.(int)+3, 10)

	// The goroutine callee bumps tally to 5.
	done := make(chan int, 1)
	go work(5, done)
	<-done

	ch := make(chan int, 1)
	ch <- 9
	_ = <-ch

	select {
	case v := <-ch:
		tally += v
	default:
		tally += 0
	}

	switch a.(type) {
	case int:
		tally += 1
	default:
		tally += 2
	}

	// The goroutine callee ran and the TYPE SWITCH's 'case int' arm adds 1:
	// 5 + 1 = 6.
	check("goroutine callee ran", tally, 6)

	if fails == 0 {
		fmt.Println("Go widening test passed")
	}
	os.Exit(fails)
}
