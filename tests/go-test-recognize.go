//go:build ignore

// Go subset RECOGNITION test - the second widening round.
//
// An earlier pass (see go-test-widen.go) covered goroutines, channels, select, type
// switches and type assertions. This file exercises the constructs added by the second
// round: they all newly PARSE. The ones that cannot be lowered are ACCEPTED and routed
// to notImplemented, so WITHOUT a flag the compile stops at the first of them (the
// grouped const block) with a clean file:line message - this file SHOULD FAIL by
// default. WITH -warn-unsupported every not-implemented construct warns and the rest
// runs, so main() reaches os.Exit(fails) with fails == 0.
//
// GENUINELY handled and checked below: grouped var, single const, nil, float / hex /
// octal / binary / underscored-decimal / rune / raw-string literals, pointer types with
// & and * as the identity, and array / variadic parameter types.
//
// Accepted + not implemented (present for recognition; their results are discarded, so
// they never affect fails): a grouped const block, a grouped type block, slice
// expressions, keyed / empty / nested composite literals, and fallthrough.

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

// Grouped var: each name is bound in source order (GENUINE).
var (
	base   = 10
	factor = 20
)

// A single const bound like a var (GENUINE).
const limit = 7

// A grouped const block: accepted, reported not implemented (iota and bit shifts are
// not modeled, so kilo/mega are NOT bound - they are never read below).
const (
	kilo = 1 << 10
	mega = 1 << 20
)

// A grouped type block: accepted, reported not implemented (its members are skipped).
type (
	Meters int
	Pixels int
)

// A struct with a pointer-typed field (the *Node type parses and is ignored). Built as
// a single type declaration, so its positional literal below is GENUINE.
type Node struct {
	val  int
	next *Node
}

// A pointer parameter, read through as the identity (structs behave like references).
func headVal(p *Node) int { return p.val }

// A variadic parameter type parses (its slice-collection semantics are not modeled, so
// the call result is not asserted).
func announce(msg string, ns ...int) { _ = msg }

func main() {
	// Integer literals in every base, with '_' digit separators (GENUINE).
	check("hex", 0xFF, 255)
	check("hex underscores", 0xDE_AD, 57005)
	check("octal", 0o17, 15)
	check("binary", 0b1010, 10)
	check("decimal underscores", 1_000_000, 1000000)

	// Rune literals are code points (GENUINE).
	check("rune A", 'A', 65)
	check("rune newline", '\n', 10)

	// Grouped var and single const (GENUINE).
	check("grouped var", base+factor, 30)
	check("const limit", limit, 7)

	// Pointer type with address-of / dereference as the identity, and nil (GENUINE).
	n := 42
	p := &n
	check("deref identity", *p, 42)
	nd := Node{5, nil}
	check("field via pointer", headVal(&nd), 5)

	// Float literals compare as numbers (GENUINE).
	pi := 3.5
	if pi > 3.0 && pi < 4.0 {
	} else {
		fails++
	}

	// A raw string keeps its bytes verbatim (GENUINE).
	raw := `abc`
	check("raw string len", len(raw), 3)

	// Array-typed and variadic declarations parse (recognition only).
	var buf [4]int
	_ = buf
	announce("hi", 1, 2, 3)

	// --- Accepted + not implemented below; under -warn-unsupported these warn and
	// their results are discarded, so fails stays 0. ---

	// Slice expressions (three forms and the three-index form).
	s := []int{1, 2, 3, 4}
	_ = s[1:3]
	_ = s[:2]
	_ = s[2:]
	_ = s[0:4:4]

	// Keyed, empty and nested composite literals.
	_ = Node{val: 1}
	_ = Node{}
	_ = []Node{{1, nil}, {2, nil}}

	// fallthrough inside a switch.
	switch limit {
	case 7:
		fallthrough
	case 8:
	default:
	}

	if fails == 0 {
		fmt.Println("Go recognition test passed")
	}
	os.Exit(fails)
}
