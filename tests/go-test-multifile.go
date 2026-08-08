//go:build ignore

// Multi-file Go test: package mathbox lives in tests/imports/mathbox.go and is found
// via the -i include root (mec -i tests/imports ...). import "mathbox" maps to that
// file; its exported functions are reached as mathbox.Add(...) (the package-object path
// fmt.Println takes), its unexported helper is used across the package, and NewVec
// returns a Vec whose method this file then calls. "fmt" is a builtin no-op import,
// mixed in on purpose; os.Exit(fails) makes the run exit 0 exactly when the checks pass.

package main

import (
	"fmt"
	"mathbox"
	"os"
	"vecbox"
)

var fails = 0

// The same three names mathbox declares, on purpose: see the 2.7 checks in main().
const Scale = 1

var Base = 1

func helper() int { return 999 }

func check(name string, got int, want int) {
	if got != want {
		fmt.Println("FAIL", name, "got", got, "want", want)
		fails++
	}
}

func checkS(name string, got string, want string) {
	if got != want {
		fmt.Println("FAIL", name, "got", got, "want", want)
		fails++
	}
}

func main() {
	// Exported package functions, reached through the package object.
	check("imported Add", mathbox.Add(20, 22), 42)
	check("imported Max", mathbox.Max(9, 4), 9)
	check("imported Abs uses helper", mathbox.Abs(-7), 7)
	check("imported Sum over slice", mathbox.Sum([]int{3, 1, 4, 1}), 9)
	checkS("imported Greet", mathbox.Greet("go"), "hello go")

	// An exported struct built by a constructor, its method called across files.
	v := mathbox.NewVec(3, 4)
	check("imported struct method", v.Len2(), 25)

	// docs/todo.md 2.7: THE PACKAGE HAS ITS OWN GLOBAL NAMESPACE. Scale, Base and
	// helper are declared BOTH here and in mathbox, so mathbox.Total() answers 117
	// only if the package resolves its own names; with one shared namespace it read
	// this file's and answered 1 + 1 + 999.
	check("package globals are not shared", mathbox.Total(), 117)
	check("main file keeps its own", Scale+Base+helper(), 1001)

	// An exported CONST and VAR reach the package object too, not just functions.
	check("imported exported const", mathbox.Scale, 10)
	check("imported exported const pair", mathbox.Lo*mathbox.Hi, 10)
	check("imported exported var", mathbox.Base, 100)
	checkS("imported exported string const", mathbox.Label, "mathbox")

	// docs/todo.md 1.5: TWO PACKAGES DECLARING THE SAME TYPE NAME. vecbox.Vec has
	// three fields and mathbox.Vec has two; one flat struct-field table meant the
	// second import overwrote the first, so mathbox.NewVec built a vecbox Vec and
	// Len2() read a y that was not there. Both are used AFTER both imports, which is
	// what makes the row a collision test rather than an ordering one.
	w := vecbox.NewVec(1, 2, 3)
	check("same type name in two packages", w.Sum(), 6)
	check("the other package's struct still whole", mathbox.NewVec(3, 4).Len2(), 25)
	check("each package keeps its own globals", vecbox.Total(), 3300)
	check("and the first package's are untouched", mathbox.Total(), 117)

	// docs/todo.md 1.5: A PACKAGE VAR IS LIVE THROUGH THE QUALIFIER, both ways.
	// The compiler half read and wrote a SNAPSHOT of the package's exported names.
	mathbox.AddBase(5) // a write INSIDE the package
	check("an internal write is seen through the qualifier", mathbox.Base, 105)
	mathbox.Base = 200 // a write THROUGH the qualifier
	check("a qualified write is seen inside the package", mathbox.BaseNow(), 200)
	check("... and by a function that resolves the name unqualified", mathbox.Total(), 217)
	mathbox.Base++ // a read-modify-write through the qualifier is both directions at once
	check("increment through the qualifier", mathbox.BaseNow(), 201)
	mathbox.Base += 9
	check("compound assign through the qualifier", mathbox.BaseNow(), 210)
	// The two packages' identically named counters do not alias.
	check("counter a", vecbox.Bump(), 1)
	check("counter b", vecbox.Bump(), 2)
	check("counter a is live", vecbox.Counter, 2)
	check("mathbox is unaffected", mathbox.Base, 210)

	// docs/todo.md 1.5: A QUALIFIED TYPE NAME from another package - a composite
	// literal (keyed and positional), a zero value, an element type, a defined type,
	// and a method on a value built through the qualified descriptor.
	qp := mathbox.Point{X: 3, Y: 4}
	check("qualified keyed literal", qp.X*10+qp.Y, 34)
	qp2 := mathbox.Point{5, 6}
	check("qualified positional literal", qp2.X*10+qp2.Y, 56)
	check("method through the qualified descriptor", qp.Norm(), 25)
	var qz mathbox.Point
	check("qualified zero value", qz.X+qz.Y+1, 1)
	qs := []mathbox.Point{{1, 2}, {X: 7}}
	check("qualified element type", mathbox.SumPoints(qs), 10)
	qn := mathbox.Nums{4, 5, 6}
	check("qualified defined slice type", len(qn)*100+qn[2], 306)
	qt := mathbox.Tally{"k": 8}
	check("qualified defined map type", qt["k"]*10+len(qt), 81)
	qm := map[string]mathbox.Point{"a": {X: 2, Y: 3}}
	check("qualified type as a map value", qm["a"].Norm(), 13)
	// A keyed NESTED struct element, which the compiler half refused outright with
	// "a keyed nested struct element is not supported" while this half built it.
	check("keyed nested element", qs[1].X*10+qs[1].Y, 70)

	if fails == 0 {
		fmt.Println("go multifile test passed")
	}
	os.Exit(fails)
}
