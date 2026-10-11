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

// The same three names mathbox declares, on purpose: see the package-namespace checks
// in main().
const Scale = 1

var Base = 1

func helper() int { return 999 }

// The same defined type name vecbox declares, with a different String().
type Kind int

func (k Kind) String() string { return "mainkind" }

// Names vecbox declares too, with other underlying types and method sets.
type Lvl int

type Tag int

func kindOf(v any) string {
	switch x := v.(type) {
	case vecbox.Kind:
		return "S" + x.String() + " "
	case Kind:
		return "M" + x.String() + " "
	}
	return "?"
}

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

	// THE PACKAGE HAS ITS OWN GLOBAL NAMESPACE. Scale, Base and helper are declared
	// BOTH here and in mathbox, so mathbox.Total() answers 117 only if the package
	// resolves its own names (one shared namespace would give 1 + 1 + 999).
	check("package globals are not shared", mathbox.Total(), 117)
	check("main file keeps its own", Scale+Base+helper(), 1001)

	// An exported CONST and VAR reach the package object too, not just functions.
	check("imported exported const", mathbox.Scale, 10)
	check("imported exported const pair", mathbox.Lo*mathbox.Hi, 10)
	check("imported exported var", mathbox.Base, 100)
	checkS("imported exported string const", mathbox.Label, "mathbox")

	// TWO PACKAGES DECLARING THE SAME TYPE NAME. vecbox.Vec has three fields and
	// mathbox.Vec has two, so a flat struct-field table would let the second import
	// overwrite the first. Both are used AFTER both imports, which is what makes the
	// row a collision test rather than an ordering one.
	w := vecbox.NewVec(1, 2, 3)
	check("same type name in two packages", w.Sum(), 6)
	check("the other package's struct still whole", mathbox.NewVec(3, 4).Len2(), 25)
	check("each package keeps its own globals", vecbox.Total(), 3300)
	check("and the first package's are untouched", mathbox.Total(), 117)

	// A PACKAGE VAR IS LIVE THROUGH THE QUALIFIER, both ways (not a snapshot of the
	// package's exported names).
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

	// A QUALIFIED TYPE NAME from another package - a composite
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
	// A keyed NESTED struct element.
	check("keyed nested element", qs[1].X*10+qs[1].Y, 70)

	// AN IMPORTED DEFINED TYPE is vecbox.Kind to this file: a value of it calls the
	// package's methods, prints through String(), and is told apart from this file's
	// own Kind, in fmt, in a type switch and in the package's own code.
	k := vecbox.Make()
	checkS("imported defined type method", k.String(), "kind2")
	checkS("imported defined type in fmt", fmt.Sprint(k, k.Twice()), "kind2 kind4")
	var k2 vecbox.Kind = 5
	checkS("declared imported defined type", fmt.Sprintf("%v %d %T", k2, k2, k2), "kind5 5 vecbox.Kind")
	k.Inc()
	checkS("pointer method of an imported defined type", fmt.Sprint(k), "kind3")
	checkS("same defined type name in main", fmt.Sprint(Kind(7))+" "+vecbox.Describe(k), "mainkind d:kind3")
	var ka any = k2
	kk, ok := ka.(vecbox.Kind)
	checkS("comma-ok assertion to an imported defined type", fmt.Sprint(kk, ok), "kind5 true")
	checkS("type switch on an imported defined type", kindOf(k)+kindOf(Kind(1))+kindOf(3), "Skind3 Mmainkind ?")
	// A package type named like a main-file type keeps its own methods and
	// underlying type, and lends neither to the main file's.
	checkS("main type without the package type's String", fmt.Sprint(Lvl(1), []Lvl{2}, vecbox.Lvl(3)), "1 [2] lvl3")
	checkS("package type over another underlying type", vecbox.TagText("x")+fmt.Sprint(Tag(4)), "x!4")
	// The package's own Kind inside its own code, and arrays of defined types: an
	// array records no element type, so it is boxed as a whole on its way into fmt
	// or an interface.
	checkS("own defined type in the package's fmt", vecbox.Show(4), "kind4 [kind4] [kind4 kind1]")
	checkS("own defined type in the package's %T", vecbox.Types(3), "vecbox.Kind []vecbox.Kind [1]vecbox.Kind")
	checkS("type switch and assertion in the package", vecbox.Which(vecbox.Box(2))+vecbox.Which(Kind(2))+
		fmt.Sprint(vecbox.Unbox(vecbox.Box(6)), vecbox.Unbox(Kind(6))), "ownotherkind6 kind0")
	checkS("array of a defined type", fmt.Sprintf("%v %T %d|%v", [2]Kind{2, 4}, [2]Kind{}, [2]vecbox.Kind{1, 2}, [...]vecbox.Kind{3}),
		"[mainkind mainkind] [2]main.Kind [1 2]|[kind3]")
	var aa any = [2]vecbox.Kind{3, 1}
	ab := aa.([2]vecbox.Kind)
	_, abOk := aa.([2]Kind)
	checkS("assertion to an array of a defined type", fmt.Sprint(int(ab[0])*10+int(ab[1]), abOk, aa), "31 false [kind3 kind1]")

	if fails == 0 {
		fmt.Println("go multifile test passed")
	}
	os.Exit(fails)
}
