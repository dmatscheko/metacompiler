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
