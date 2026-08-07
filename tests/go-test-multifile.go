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

	if fails == 0 {
		fmt.Println("go multifile test passed")
	}
	os.Exit(fails)
}
