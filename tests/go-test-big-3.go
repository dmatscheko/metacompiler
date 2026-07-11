//go:build ignore

// Go subset big test 3: OBJECT AND FUNCTIONAL FEATURES.
//
// Struct methods with value and pointer receivers (a 2D vector algebra and a
// tagged Shape whose area()/perimeter() dispatch on a kind field, standing in for
// interfaces), higher-order functions over slices (map / filter / reduce / any /
// all), function composition, currying and repeated application through closures,
// a memoizing wrapper that caches into a captured map, counter and accumulator
// factories, a function-valued dispatch table, a function pipeline held in a
// slice, and defer (LIFO order, arguments captured early). main() ends with
// os.Exit(fails).

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

func checkS(name string, got string, want string) {
	if got != want {
		fmt.Println("FAIL", name, "got", got, "want", want)
		fails++
	}
}

func checkBool(name string, got bool, want bool) {
	g := 0
	w := 0
	if got {
		g = 1
	}
	if want {
		w = 1
	}
	check(name, g, w)
}

// ----- 2D vector algebra: value receivers return new vectors -----

type Vec2 struct {
	x int
	y int
}

func (v Vec2) add(w Vec2) Vec2 {
	return Vec2{v.x + w.x, v.y + w.y}
}

func (v Vec2) sub(w Vec2) Vec2 {
	return Vec2{v.x - w.x, v.y - w.y}
}

func (v Vec2) scale(f int) Vec2 {
	return Vec2{v.x * f, v.y * f}
}

func (v Vec2) dot(w Vec2) int {
	return v.x*w.x + v.y*w.y
}

func (v Vec2) cross(w Vec2) int {
	return v.x*w.y - v.y*w.x
}

func (v Vec2) lenSq() int {
	return v.x*v.x + v.y*v.y
}

func (v Vec2) manhattan() int {
	ax := v.x
	if ax < 0 {
		ax = -ax
	}
	ay := v.y
	if ay < 0 {
		ay = -ay
	}
	return ax + ay
}

func (v Vec2) equals(w Vec2) bool {
	return v.x == w.x && v.y == w.y
}

// pointer receiver mutates in place
func (v *Vec2) translate(dx int, dy int) {
	v.x += dx
	v.y += dy
}

// ----- tagged shape: dispatch on a kind field (no interfaces in the subset) -----

type Shape struct {
	kind string
	a    int
	b    int
}

func (s Shape) area() int {
	switch s.kind {
	case "rect":
		return s.a * s.b
	case "square":
		return s.a * s.a
	case "tri":
		return s.a * s.b / 2
	default:
		return 0
	}
}

func (s Shape) perimeter() int {
	switch s.kind {
	case "rect":
		return 2 * (s.a + s.b)
	case "square":
		return 4 * s.a
	default:
		return 0
	}
}

// ----- higher-order functions over int slices -----

func mapInts(f func(int) int, s []int) []int {
	out := []int{}
	for _, v := range s {
		out = append(out, f(v))
	}
	return out
}

func filterInts(pred func(int) bool, s []int) []int {
	out := []int{}
	for _, v := range s {
		if pred(v) {
			out = append(out, v)
		}
	}
	return out
}

func reduceInts(f func(int, int) int, init int, s []int) int {
	acc := init
	for _, v := range s {
		acc = f(acc, v)
	}
	return acc
}

func anyInts(pred func(int) bool, s []int) bool {
	for _, v := range s {
		if pred(v) {
			return true
		}
	}
	return false
}

func allInts(pred func(int) bool, s []int) bool {
	for _, v := range s {
		if !pred(v) {
			return false
		}
	}
	return true
}

func equalSlice(a []int, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ----- named functions passed as values -----

func double(x int) int  { return 2 * x }
func square(x int) int  { return x * x }
func incr(x int) int    { return x + 1 }
func isEvenP(x int) bool { return x%2 == 0 }
func isPos(x int) bool   { return x > 0 }
func addOp(a int, b int) int { return a + b }
func mulOp(a int, b int) int { return a * b }
func maxOp(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func compose(f func(int) int, g func(int) int) func(int) int {
	return func(x int) int {
		return f(g(x))
	}
}

func applyN(f func(int) int, n int, x int) int {
	for i := 0; i < n; i++ {
		x = f(x)
	}
	return x
}

// ----- a memoize wrapper over a counted slow function -----

var slowCalls = 0

func slowSquare(n int) int {
	slowCalls++
	return n * n
}

// ----- defer demonstrations -----

var dlog = ""
var nlog = 0

func note(s string)  { dlog = dlog + s }
func recordN(n int)  { nlog = n }

func deferOrder() string {
	dlog = ""
	defer note("[end]")
	defer note("[mid]")
	note("[start]")
	return dlog
}

func deferArgEarly() int {
	x := 5
	defer recordN(x)
	x = 99
	return x
}

func main() {
	// vector algebra
	a := Vec2{3, 4}
	b := Vec2{1, 2}
	checkBool("vec add", a.add(b).equals(Vec2{4, 6}), true)
	checkBool("vec sub", a.sub(b).equals(Vec2{2, 2}), true)
	checkBool("vec scale", a.scale(3).equals(Vec2{9, 12}), true)
	check("vec dot", a.dot(b), 11)
	check("vec cross", a.cross(b), 2)
	check("vec lenSq", a.lenSq(), 25)
	check("vec manhattan", Vec2{-3, 4}.manhattan(), 7)
	// chained value-receiver methods do not mutate the source
	chained := a.add(b).scale(2).sub(Vec2{1, 1})
	checkBool("vec chained", chained.equals(Vec2{7, 11}), true)
	checkBool("vec source intact", a.equals(Vec2{3, 4}), true)
	// pointer receiver mutates
	m := Vec2{0, 0}
	m.translate(5, 6)
	m.translate(-2, 3)
	checkBool("vec translate", m.equals(Vec2{3, 9}), true)

	// tagged shapes and reduce over their areas
	shapes := []Shape{
		Shape{"rect", 3, 4},
		Shape{"square", 5, 0},
		Shape{"tri", 6, 8}}
	totalArea := 0
	for _, sh := range shapes {
		totalArea += sh.area()
	}
	check("shape total area", totalArea, 12+25+24)
	check("rect perimeter", shapes[0].perimeter(), 14)
	check("square perimeter", shapes[1].perimeter(), 20)
	check("tri perimeter default", shapes[2].perimeter(), 0)

	// higher-order functions
	nums := []int{1, 2, 3, 4, 5, 6}
	doubled := mapInts(double, nums)
	checkBool("map double", equalSlice(doubled, []int{2, 4, 6, 8, 10, 12}), true)
	squared := mapInts(square, nums)
	check("map square last", squared[5], 36)
	evens := filterInts(isEvenP, nums)
	checkBool("filter evens", equalSlice(evens, []int{2, 4, 6}), true)
	check("reduce sum", reduceInts(addOp, 0, nums), 21)
	check("reduce product", reduceInts(mulOp, 1, nums), 720)
	check("reduce max", reduceInts(maxOp, nums[0], nums), 6)
	checkBool("any even", anyInts(isEvenP, nums), true)
	checkBool("all positive", allInts(isPos, nums), true)
	checkBool("not all even", allInts(isEvenP, nums), false)
	// map then filter then reduce
	pipeline := reduceInts(addOp, 0, filterInts(isEvenP, mapInts(incr, nums)))
	check("map-filter-reduce", pipeline, 2+4+6) // incr -> 2,3,4,5,6,7; evens 2,4,6

	// composition and repeated application
	sqThenDouble := compose(double, square)
	check("compose", sqThenDouble(3), 18) // double(square(3)) = double(9)
	doubleThenSq := compose(square, double)
	check("compose order", doubleThenSq(3), 36) // square(double(3)) = square(6)
	check("applyN incr", applyN(incr, 10, 0), 10)
	check("applyN double", applyN(double, 5, 1), 32) // 2^5

	// currying through nested closures
	curriedAdd := func(x int) func(int) func(int) int {
		return func(y int) func(int) int {
			return func(z int) int {
				return x + y + z
			}
		}
	}
	step1 := curriedAdd(100)
	step2 := step1(20)
	check("curry", step2(3), 123)

	// counter factory: independent closures
	makeCounter := func(start int, step int) func() int {
		c := start
		return func() int {
			c += step
			return c
		}
	}
	c1 := makeCounter(0, 1)
	c2 := makeCounter(100, 10)
	c1()
	c1()
	check("counter 1", c1(), 3)
	check("counter 2", c2(), 110)

	// accumulator factory
	makeAcc := func() func(int) int {
		total := 0
		return func(d int) int {
			total += d
			return total
		}
	}
	acc := makeAcc()
	acc(10)
	acc(20)
	check("accumulator", acc(5), 35)

	// memoize wrapper caching into a captured map
	makeMemo := func() func(int) int {
		cache := make(map[int]int)
		return func(n int) int {
			v, ok := cache[n]
			if ok {
				return v
			}
			r := slowSquare(n)
			cache[n] = r
			return r
		}
	}
	memoSq := makeMemo()
	slowCalls = 0
	check("memo 4 a", memoSq(4), 16)
	check("memo 4 b", memoSq(4), 16)
	check("memo 7", memoSq(7), 49)
	check("memo 4 c", memoSq(4), 16)
	check("memo call count", slowCalls, 2) // 4 and 7 computed once each

	// function-valued dispatch table
	ops := make(map[string]func(int, int) int)
	ops["+"] = addOp
	ops["*"] = mulOp
	ops["max"] = maxOp
	fadd := ops["+"]
	fmul := ops["*"]
	fmax := ops["max"]
	check("dispatch +", fadd(6, 7), 13)
	check("dispatch *", fmul(6, 7), 42)
	check("dispatch max", fmax(6, 7), 7)

	// function pipeline held in a slice
	pipe := []func(int) int{double, incr, square}
	x := 3
	for _, fn := range pipe {
		x = fn(x)
	}
	check("pipeline", x, 49) // ((3*2)+1)^2 = 49

	// defer: LIFO order and early argument capture
	rv := deferOrder()
	checkS("defer return value", rv, "[start]")
	checkS("defer lifo log", dlog, "[start][mid][end]")
	check("defer sees final x", deferArgEarly(), 99)
	check("defer arg captured early", nlog, 5)

	if fails == 0 {
		fmt.Println("Go big test 3 (object and functional features) passed")
	}
	os.Exit(fails)
}
