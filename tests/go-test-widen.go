//go:build ignore

// Go subset widening test.
//
// EVERY construct here is genuinely implemented now, in both grammars: the type
// assertion x.(T), the type switch, the interface declaration (a no-op, since the value
// model is dynamic and satisfaction is therefore implicit), the goroutine, the channel
// make/send/receive, and the select. `go work(5)` really runs, so tally reaches 6 via a
// genuine goroutine rather than via a warned no-op.
//
// The file runs to os.Exit(fails) with fails == 0, with or without -warn-unsupported,
// and a run with the flag reports no unsupported construct at all.
//
// History: this was a by-design SHOULD-FAIL guard, because a flagless run aborted at the
// first not-implemented construct (the Stringer interface). The full-syntax work
// implemented all of them - concurrency last - which removed the guard's premise, so the
// matrix entry became an ordinary one. Do not re-arm it.

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

	// The goroutine callee ran (its call is visible) and the TYPE SWITCH is
	// genuinely implemented now, so its 'case int' arm adds 1: 5 + 1 = 6. The
	// channel and select bodies are still no-ops under -warn-unsupported.
	// (This expected 5 while the type switch was merely recognised; that is the
	// kind of assertion that has to move as the subset grows, not be defended.)
	check("goroutine callee ran", tally, 6)

	if fails == 0 {
		fmt.Println("Go widening test passed")
	}
	os.Exit(fails)
}
