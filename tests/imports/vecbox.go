// A second project package, imported by tests/go-test-multifile.go beside mathbox.
//
// It declares `type Vec struct` on purpose, because mathbox declares one too: the
// compiler's struct field tables must be keyed by package, not by bare type name.

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
