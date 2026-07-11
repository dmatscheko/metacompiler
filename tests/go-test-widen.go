//go:build ignore

// Go subset widening test.
//
// GENUINELY implemented and checked here: the single-result type assertion x.(T),
// which is the identity on the value (the target type is parsed and ignored).
//
// Accepted but reported "not implemented" (they need concurrency / channel / interface
// semantics the subset does not model): an interface type declaration, a goroutine
// (go f()), a channel make/send/receive, a select, and a type switch. Where operands
// exist (the goroutine callee, the send / receive channel and value) they still run,
// so their calls stay visible in call graphs.
//
// Without a flag the compile stops at the FIRST not-implemented construct (the
// Stringer interface) with a clean file:line message - this file SHOULD FAIL by
// default. With -warn-unsupported every not-implemented construct warns and its
// operands run, so main() reaches os.Exit(fails) with fails == 0 (every type-assertion
// check passes, and the goroutine callee ran).

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

// Accepted, reported not implemented: an interface type declaration.
type Stringer interface {
	String() string
}

var tally = 0

func work(n int) { tally += n } // a goroutine callee; stays visible in call graphs

// A boxed value comes in as interface{}; the assertion recovers it unchanged.
func unbox(x interface{}) int {
	return x.(int)
}

func main() {
	// GENUINE: type assertions x.(T) return the value unchanged (identity).
	check("assert via param", unbox(42), 42)
	a := 7
	check("assert direct", a.(int), 7)
	check("assert in expr", a.(int)+3, 10)

	// Accepted + not implemented; under -warn-unsupported these warn and their
	// operands run (the goroutine callee below bumps tally to 5).
	go work(5)

	ch := make(chan int)
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

	// The goroutine callee ran (its call is visible); the channel/select/type-switch
	// bodies are no-ops under -warn-unsupported, so tally stayed 5.
	check("goroutine callee ran", tally, 5)

	if fails == 0 {
		fmt.Println("Go widening test passed")
	}
	os.Exit(fails)
}
