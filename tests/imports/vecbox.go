// A second project package, imported by tests/go-test-multifile.go beside mathbox.
//
// It declares `type Vec struct` on purpose, because mathbox declares one too: the
// compiler's struct field tables must be keyed by package, not by bare type name.

package vecbox

import "fmt"

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

// A DEFINED NON-STRUCT TYPE with methods. Seen from the main file it is vecbox.Kind,
// and tests/go-test-multifile.go declares a Kind of its own beside it.
type Kind int

func (k Kind) String() string { return "kind" + string(rune('0'+int(k))) }

func (k Kind) Twice() Kind { return k * 2 }

func (k *Kind) Inc() { *k = *k + 1 }

func Make() Kind { return Kind(2) }

func Describe(k Kind) string { return "d:" + k.String() }

// Names the main file declares too, with other underlying types and method sets.
type Lvl int

func (l Lvl) String() string { return "lvl" + string(rune('0'+int(l))) }

type Tag string

func TagText(t Tag) string { return string(t) + "!" }

// The package's OWN defined type in its own code is vecbox.Kind, not the main
// file's Kind: fmt reaches its String() for a value, a slice element and an array
// element, %T names it, and a type switch and an assertion compare against it.
func Show(k Kind) string { return fmt.Sprint(k, []Kind{k}, [2]Kind{k, 1}) }

func Types(k Kind) string { return fmt.Sprintf("%T %T %T", k, []Kind{k}, [1]Kind{k}) }

func Box(n int) any { return Kind(n) }

func Which(v any) string {
	switch v.(type) {
	case Kind:
		return "own"
	}
	return "other"
}

func Unbox(v any) Kind {
	k, _ := v.(Kind)
	return k
}
