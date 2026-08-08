// A second project package, imported by tests/go-test-multifile.go beside mathbox.
//
// IT DECLARES `type Vec struct` ON PURPOSE, because mathbox declares one too. The
// struct field tables (structFields / structFTypes in languages/go-to-llvm-ir.abnf)
// used to be keyed by the BARE type name across every package, so the second import
// overwrote the first: mathbox.NewVec(3, 4) built a vecbox Vec, its y field did not
// exist, and v.Len2() answered NaN. The tables are keyed by package now. The
// interpreter half was never wrong here - each package's descriptor is its own scope
// binding - so this is a compiler-half row and --cross was blind to it.
// docs/todo.md 1.5.

package vecbox

type Vec struct {
	a int
	b int
	c int
}

func NewVec(a int, b int, c int) Vec { return Vec{a, b, c} }

func (v Vec) Sum() int { return v.a + v.b + v.c }

// The same three names mathbox and the main file both declare, for the namespace
// check: vecbox's Total() must read ITS values.
const Scale = 1000

var Base = 2000

func helper() int { return 300 }

func Total() int { return Scale + Base + helper() }

var Counter = 0

func Bump() int {
	Counter = Counter + 1
	return Counter
}
