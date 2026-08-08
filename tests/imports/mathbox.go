// A project package imported by tests/go-test-multifile.go (via -i tests/imports).
// import "mathbox" maps to this file (mathbox.go). Its exported (capitalized) functions
// are reached through the package object as mathbox.Add(...), the way fmt.Println is;
// its own names resolve unqualified within the package (Abs calls the unexported abs),
// and NewVec returns a Vec whose value-receiver method the main file calls across files.

package mathbox

// An unexported helper, called unqualified from Abs below (an intra-package call).
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func Add(a int, b int) int {
	return a + b
}

func Max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func Abs(n int) int {
	return abs(n)
}

func Sum(xs []int) int {
	total := 0
	for _, v := range xs {
		total += v
	}
	return total
}

func Greet(name string) string {
	return "hello " + name
}

// An exported struct plus a constructor and a value-receiver method: the main file
// builds one via mathbox.NewVec(...) and calls .Len2() on the returned value.
type Vec struct {
	x int
	y int
}

func NewVec(x int, y int) Vec {
	return Vec{x, y}
}

func (v Vec) Len2() int {
	return v.x*v.x + v.y*v.y
}

// ----- docs/todo.md 2.7: the package has its OWN GLOBAL NAMESPACE -----
// Every file used to declare into one shared runtime namespace, so the importing
// file's `var Scale = 1` and `func helper()` OVERWROTE this package's own - and
// since Total() resolves its unqualified names through that same namespace at CALL
// time, it answered the main file's values. The names below are deliberately the
// same as the ones tests/go-test-multifile.go declares for itself, which is what
// makes Total() a test of the namespace rather than of arithmetic.
//
// Exported CONSTS and VARS are also reachable through the package object now
// (mconst.S used to be <nil>): only exported FUNCTIONS were ever put on it.
const Scale = 10
const Label = "mathbox"
const Lo, Hi = 2, 5

var Base = 100

func helper() int { return 7 }

// Total mixes a package const, a package var and an unexported package func, all
// three of which the importing file also declares under the same name.
func Total() int {
	return Scale + Base + helper()
}

// ----- docs/todo.md 1.5: A PACKAGE VAR IS LIVE THROUGH THE QUALIFIER -----
// The compiler half assembled the package object by COPYING each exported name out of
// the package's scope, so the object was a snapshot: AddBase's write was invisible as
// mathbox.Base, and a write TO mathbox.Base never reached the variable Total() reads.
// The interpreter half binds the scope itself and was always live - a halves
// divergence --cross could not see, because no test wrote to a package var. The object
// now carries the package's own scope (__pkgscope) and go's member externs read
// through it.
func AddBase(n int) { Base = Base + n }

func BaseNow() int { return Base }

// ----- docs/todo.md 1.5: A QUALIFIED TYPE NAME -----
// mathbox.Point{X: 1, Y: 2} and `var p mathbox.Point` are ordinary Go and parsed in
// neither half: the compiler reported `composite literal not implemented` (CompLitNI's
// CompName matched `Id { "." Id }` and its SkipBlock ate the body) and the interpreter
// dropped the qualifier and died with `unknown struct type: mathbox`. A DEFINED type is
// here too, since `mathbox.Nums{1, 2}` resolves through the same qualified key.
type Point struct {
	X int
	Y int
}

func (p Point) Norm() int { return p.X*p.X + p.Y*p.Y }

type Nums []int

type Tally map[string]int

func SumPoints(ps []Point) int {
	t := 0
	for _, p := range ps {
		t += p.X + p.Y
	}
	return t
}
