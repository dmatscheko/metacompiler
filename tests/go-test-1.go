//go:build ignore

// Go subset self test.
// main() counts failed checks and ends with os.Exit(fails), so the
// metacompiler run exits with 0 exactly when everything works.

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

func add(a int, b int) int {
	return a + b
}

func fib(n int) int {
	if n < 2 {
		return n
	}
	return fib(n-1) + fib(n-2)
}

func divmod(a int, b int) (int, int) {
	return a / b, a % b
}

func minmax(a int, b int) (int, int) {
	if a < b {
		return a, b
	}
	return b, a
}

type Point struct {
	x int
	y int
}

func (p Point) manhattan() int {
	if p.x < 0 {
		if p.y < 0 {
			return -p.x - p.y
		}
		return -p.x + p.y
	}
	if p.y < 0 {
		return p.x - p.y
	}
	return p.x + p.y
}

func (p Point) scale(f int) Point {
	return Point{p.x * f, p.y * f}
}

func sum(s []int) int {
	total := 0
	for _, v := range s {
		total += v
	}
	return total
}

func apply(f func(int) int, v int) int { // a function typed parameter
	return f(v)
}

var dlog = ""
var nlog = 0

func record(s string) { dlog = dlog + s }
func recordN(n int)   { nlog = n }

func deferDemo() int { // defers run LIFO, after the return value is computed
	dlog = ""
	defer record("c")
	defer record("b")
	dlog = dlog + "a"
	return len(dlog)
}

func deferArgVal() int { // defer arguments are evaluated at defer time
	x := 1
	defer recordN(x)
	x = 50
	return x
}

func main() {
	// arithmetic
	check("precedence", 1+2*3, 7)
	check("division", 7/2, 3)
	check("negative division", -7/2, -3)
	check("modulo", 7%3, 1)
	check("call", add(20, 22), 42)

	// short declarations, parallel assignment, blank identifier
	a := 1
	var b int = 2
	a, b = b, a
	check("swap a", a, 2)
	check("swap b", b, 1)
	q, r := divmod(17, 5)
	check("divmod q", q, 3)
	check("divmod r", r, 2)
	lo, hi := minmax(9, 4)
	check("minmax lo", lo, 4)
	check("minmax hi", hi, 9)
	_, onlyHi := minmax(3, 8)
	check("blank", onlyHi, 8)

	var zero int
	check("zero value", zero, 0)
	var empty string
	checkS("zero string", empty, "")

	// strings
	s := "go"
	s += "lang"
	checkS("concat", s, "golang")
	check("len", len(s), 6)

	// if / else if / else
	grade := 0
	score := 85
	if score >= 90 {
		grade = 1
	} else if score >= 80 {
		grade = 2
	} else {
		grade = 3
	}
	check("else if", grade, 2)

	// loops
	total := 0
	for i := 0; i < 10; i++ {
		total += i
	}
	check("classic for", total, 45)

	w := 0
	for w < 5 {
		w++
	}
	check("cond for", w, 5)

	odd := 0
	for j := 0; j < 100; j++ {
		if j%2 == 0 {
			continue
		}
		if j > 10 {
			break
		}
		odd += j
	}
	check("break continue", odd, 25)

	// slices
	sl := []int{3, 1, 4}
	check("slice len", len(sl), 3)
	check("slice index", sl[2], 4)
	sl[1] = 10
	check("slice store", sl[1], 10)
	sl = append(sl, 7)
	check("append len", len(sl), 4)
	check("append value", sl[3], 7)
	check("sum range", sum(sl), 24)

	isum := 0
	vsum := 0
	for i, v := range sl {
		isum += i
		vsum += v
	}
	check("range index", isum, 6)
	check("range value", vsum, 24)

	// structs and methods
	p := Point{3, -4}
	check("field", p.x, 3)
	check("method", p.manhattan(), 7)
	p.y = 4
	check("field write", p.y, 4)
	d := p.scale(3)
	check("returned struct", d.x+d.y, 21)
	check("chained", Point{1, 2}.scale(10).manhattan(), 30)

	// switch
	sw := 0
	switch 2 {
	case 1:
		sw = 1
	case 2, 3:
		sw = 23
	default:
		sw = 9
	}
	check("switch value", sw, 23)

	switch 7 {
	case 1:
		sw = 1
	default:
		sw = 77
	}
	check("switch default", sw, 77)

	tagless := ""
	score2 := 85
	switch {
	case score2 >= 90:
		tagless = "A"
	case score2 >= 80:
		tagless = "B"
	default:
		tagless = "C"
	}
	checkS("tagless switch", tagless, "B")

	// range over an int (Go 1.22)
	rsum := 0
	for i := range 5 {
		rsum += i
	}
	check("range int", rsum, 10)

	// maps
	ages := map[string]int{"alice": 30, "bob": 25}
	check("map get", ages["alice"], 30)
	ages["carol"] = 35
	check("map set new", ages["carol"], 35)
	ages["bob"]++
	check("map incdec", ages["bob"], 26)
	ages["carol"] += 5
	check("map compound", ages["carol"], 40)
	check("map len", len(ages), 3)
	check("map zero value", ages["nobody"], 0)
	v1, ok1 := ages["alice"]
	check("comma ok value", v1, 30)
	if !ok1 {
		check("comma ok flag", 0, 1)
	}
	_, ok2 := ages["nobody"]
	if ok2 {
		check("comma ok miss", 1, 0)
	}
	delete(ages, "bob")
	check("map delete", len(ages), 2)
	check("deleted reads zero", ages["bob"], 0)
	msum := 0
	knames := ""
	for k, v := range ages {
		msum += v
		knames += k
	}
	check("range map values", msum, 70)
	check("range map keys", len(knames), 10)
	counts := make(map[string]int)
	for _, w2 := range []string{"a", "b", "a"} {
		counts[w2]++
	}
	check("map counter idiom", counts["a"], 2)
	mi := map[int]string{1: "one", 2: "two"}
	checkS("int keyed map", mi[2], "two")

	// function literals and closures
	twice := func(x int) int { return x * 2 }
	check("func literal", twice(21), 42)
	check("func literal as arg", apply(twice, 7), 14)
	acc := 0
	add2 := func(d int) { acc += d }
	add2(5)
	add2(7)
	check("closure writes", acc, 12)
	mk := func(start int) func() int {
		c := start
		return func() int {
			c++
			return c
		}
	}
	c1 := mk(10)
	c2 := mk(100)
	c1()
	check("closure counter", c1(), 12)
	check("closures independent", c2(), 101)

	// defer
	check("defer after return value", deferDemo(), 1)
	checkS("defer lifo order", dlog, "abc")
	check("defer sees final x", deferArgVal(), 50)
	check("defer args early", nlog, 1)

	// recursion
	check("fib", fib(10), 55)

	if fails == 0 {
		fmt.Println("Go subset self test passed")
	}
	os.Exit(fails)
}
