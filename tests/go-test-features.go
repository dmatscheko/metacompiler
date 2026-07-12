//go:build ignore

// Fast feature-matrix test for the Go interpreter (go-interpreter.abnf) and the
// LLVM-IR compiler (go-to-llvm-ir.abnf). It replaces the four algorithm-themed
// go-test-big-* stress tests: instead of large loops (Ackermann, sieves, RPN
// calculators) every implemented construct is exercised with the SMALLEST program
// that can prove it works - loops run 0, 1, 3 or 4 times, recursion stays below
// depth 6. A failed check prints "FAIL <id>" (so a diff pinpoints it) and main()
// ends with os.Exit(fails); exit 0 and byte-identical output on all four legs
// (interpreter/compiler x goja/-frozen) mean everything passed.

package main

import (
	"fmt"
	"os"
)

var fails = 0
var checks = 0

func check(id string, cond bool) {
	checks++
	if !cond {
		fmt.Println("FAIL " + id)
		fails++
	}
}

// ----- functions: multiple returns, early return, recursion -----

func divmod(a int, b int) (int, int) {
	return a / b, a % b
}

func grade(n int) string {
	if n > 10 {
		return "big"
	} else if n > 5 {
		return "mid"
	}
	return "small"
}

func fib(n int) int {
	if n < 2 {
		return n
	}
	return fib(n-1) + fib(n-2)
}

func isEven(n int) bool {
	if n == 0 {
		return true
	}
	return isOdd(n - 1)
}

func isOdd(n int) bool {
	if n == 0 {
		return false
	}
	return isEven(n - 1)
}

func apply(f func(int) int, v int) int {
	return f(v)
}

// ----- structs and methods -----

type Point struct {
	x int
	y int
}

func (p Point) manhattan() int {
	a := p.x
	if a < 0 {
		a = -a
	}
	b := p.y
	if b < 0 {
		b = -b
	}
	return a + b
}

func (p Point) scale(f int) Point {
	return Point{p.x * f, p.y * f}
}

func (p Point) dims() (int, int) {
	return p.x, p.y
}

type Line struct {
	a Point
	b Point
}

type Counter struct {
	n int
}

func (c *Counter) inc()      { c.n++ }
func (c *Counter) add(d int) { c.n += d }
func (c Counter) get() int   { return c.n }

// ----- pointers (identity semantics) -----

type Node struct {
	val  int
	next *Node
}

func headVal(p *Node) int { return p.val }

// ----- grouped var, const, side-effect counter for short-circuit -----

var (
	base   = 10
	factor = 3
)

const limit = 7

var sideEffects = 0

func bump() bool {
	sideEffects++
	return true
}

func never() bool {
	sideEffects += 100
	return false
}

// ----- defer -----

var dlog = ""
var nlog = 0

func record(s string) { dlog += s }
func recordN(n int)   { nlog = n }

func deferDemo() int { // defers run LIFO, after the return value is computed
	dlog = ""
	defer record("c")
	defer record("b")
	dlog += "a"
	return len(dlog)
}

func deferArgVal() int { // defer arguments are evaluated at defer time
	x := 1
	defer recordN(x)
	x = 50
	return x
}

// ----- one small combined pipeline: closure + map + slice + switch -----

func classify(list []int, tag func(int) string) string {
	counts := map[string]int{}
	out := ""
	for _, n := range list {
		label := ""
		switch {
		case n < 0:
			label = "neg"
		case n%2 == 0:
			label = "even"
		default:
			label = tag(n)
		}
		counts[label]++
		out += label
	}
	return out + itoaSmall(counts["odd"])
}

func itoaSmall(n int) string { // enough for 0..4
	digits := []string{"0", "1", "2", "3", "4"}
	return digits[n]
}

func main() {
	// ----- numbers, arithmetic, precedence -----
	check("arith-precedence", 2+3*4 == 14)
	check("arith-paren", (2+3)*4 == 20)
	check("arith-unary-minus", -3+5 == 2)
	check("arith-int-div", 7/2 == 3)
	check("arith-int-div-neg", -7/2 == -3)
	check("arith-mod", 7%3 == 1)
	check("arith-mod-neg", -7%3 == -1)
	check("arith-chain", 20-5-3 == 12)
	x := 5
	x += 3
	x -= 2
	x *= 4
	x /= 3
	x %= 5
	check("arith-compound", x == 3)
	i := 5
	i++
	i++
	i--
	check("arith-incdec", i == 6)

	// integer literals in other bases, rune literals
	check("num-hex", 0xFF == 255)
	check("num-octal", 0o17 == 15)
	check("num-binary", 0b1010 == 10)
	check("num-underscore", 1_000 == 1000)
	check("num-rune", 'A' == 65)
	check("num-rune-escape", '\n' == 10)

	// float literals compare as numbers (arithmetic is integer-only in the subset)
	pi := 3.5
	check("float-compare", pi > 3.0 && pi < 4.0 && pi == 3.5)

	// ----- declarations, multiple assignment, zero values -----
	a := 1
	var b int = 2
	a, b = b, a
	check("swap", a == 2 && b == 1)
	q, r := divmod(17, 5)
	check("multi-return", q == 3 && r == 2)
	_, onlyR := divmod(9, 4)
	check("blank-ident", onlyR == 1)
	var zeroI int
	var zeroS string
	var zeroB bool
	check("zero-values", zeroI == 0 && zeroS == "" && zeroB == false)
	check("grouped-var", base+factor == 13)
	check("const", limit == 7)

	// ----- strings -----
	s := "go"
	s += "lang"
	check("str-concat", s == "golang" && "foo"+"bar" == "foobar")
	check("str-len", len(s) == 6 && len("") == 0)
	check("str-compare", "apple" < "banana" && !("b" < "a") && "a" != "b")
	check("str-escapes", len("a\tb") == 3 && len("\\") == 1 && len("\"") == 1)
	raw := `a\nb`
	check("str-raw", len(raw) == 4)

	rebuilt := ""
	isum := 0
	for j, ch := range "abc" {
		rebuilt += ch
		isum += j
	}
	check("str-range", rebuilt == "abc" && isum == 3)
	third := ""
	for j, ch := range "abcd" {
		if j == 2 {
			third = ch
		}
	}
	check("str-range-nth", third == "c")
	hits := 0
	for _, ch := range "banana" {
		if ch == "a" {
			hits++
		}
	}
	check("str-range-eq", hits == 3)
	uni := 0
	acc := ""
	for _, ch := range "héllo" {
		uni++
		acc += ch
	}
	check("str-unicode", uni == 5 && acc == "héllo" && len("héllo") == 5)
	rev := ""
	for _, ch := range "abc" {
		rev = ch + rev
	}
	check("str-build-reverse", rev == "cba")

	// ----- booleans, short-circuit, logic -----
	check("bool-ops", true && !false || false)
	sideEffects = 0
	ok1 := never() && bump() // never() is false: bump must not run
	ok2 := bump() || never() // bump() is true: never must not run
	check("logic-short-circuit", !ok1 && ok2 && sideEffects == 101)
	check("cmp-chain", 1 < 2 && 2 <= 2 && 3 > 2 && 3 >= 3 && 1 == 1 && 1 != 2)

	// ----- control flow -----
	check("if-elseif-else", grade(11) == "big" && grade(7) == "mid" && grade(1) == "small")

	w := 0
	for w > 0 { // runs zero times
		w--
	}
	check("while-zero", w == 0)
	w3 := 0
	for w3 < 3 { // runs three times
		w3++
	}
	check("while-three", w3 == 3)

	forSum := 0
	for fi := 1; fi <= 3; fi++ {
		forSum += fi
	}
	check("for-basic", forSum == 6)

	brk := 0
	for bi := 0; bi < 9; bi++ {
		if bi == 2 {
			break
		}
		brk = brk*10 + bi + 1
	}
	check("for-break", brk == 12)

	cont := 0
	for ci := 0; ci < 4; ci++ {
		if ci%2 == 1 {
			continue
		}
		cont += ci
	}
	check("for-continue", cont == 2)

	nested := 0
	for oi := 0; oi < 2; oi++ {
		for ii := 0; ii < 3; ii++ {
			if ii == 1 {
				break // must not end the outer loop
			}
			nested++
		}
	}
	check("nested-break", nested == 2)

	rsum := 0
	for j := range 4 { // range over an int (Go 1.22)
		rsum += j
	}
	check("range-int", rsum == 6)

	// ----- switch: tag, multi-value case, default, tagless, implicit break -----
	sw := ""
	switch 2 {
	case 1:
		sw = "one"
	case 2, 3:
		sw = "two-or-three"
	default:
		sw = "other"
	}
	check("switch-multi", sw == "two-or-three")

	switch 9 {
	case 1:
		sw = "one"
	default:
		sw = "default"
	}
	check("switch-default", sw == "default")

	hit := 0
	switch 1 {
	case 1:
		hit++ // implicit break: case 2 must NOT run
	case 2:
		hit += 10
	}
	check("switch-implicit-break", hit == 1)

	tagless := ""
	score := 85
	switch {
	case score >= 90:
		tagless = "A"
	case score >= 80:
		tagless = "B"
	default:
		tagless = "C"
	}
	check("switch-tagless", tagless == "B")

	// ----- functions, closures, recursion -----
	check("fn-recursion", fib(6) == 8)
	check("fn-mutual-recursion", isEven(4) && isOdd(5))

	twice := func(n int) int { return n * 2 }
	check("fn-literal", twice(21) == 42)
	check("fn-higher-order", apply(twice, 7) == 14)

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
	check("closure-independent", c1() == 12 && c2() == 101)

	accSum := 0
	addTo := func(d int) { accSum += d }
	addTo(5)
	addTo(7)
	check("closure-writes-outer", accSum == 12)

	// ----- defer -----
	check("defer-after-return", deferDemo() == 1)
	check("defer-lifo", dlog == "abc")
	check("defer-final-x", deferArgVal() == 50)
	check("defer-args-early", nlog == 1)

	// ----- slices -----
	sl := []int{3, 1, 4}
	check("slice-literal", len(sl) == 3 && sl[0] == 3 && sl[2] == 4)
	sl[1] = 10
	check("slice-write", sl[1] == 10)
	sl = append(sl, 7)
	check("slice-append", len(sl) == 4 && sl[3] == 7)
	empty := []int{}
	check("slice-empty", len(empty) == 0)
	ssum := 0
	sidx := 0
	for j, v := range sl {
		sidx += j
		ssum += v
	}
	check("slice-range", sidx == 6 && ssum == 24)
	only := 0
	for j := range sl {
		only += j
	}
	check("slice-range-index-only", only == 6)
	words := []string{"a", "b", "a"}
	check("slice-strings", words[1] == "b" && len(words) == 3)
	rows := [][]int{[]int{5, 6}, []int{7}}
	rows = append(rows, []int{8, 9})
	check("slice-nested", rows[0][1] == 6 && rows[2][0] == 8 && len(rows) == 3)

	// ----- maps -----
	ages := map[string]int{"alice": 30, "bob": 25}
	check("map-get", ages["alice"] == 30)
	ages["carol"] = 35
	check("map-set-new", ages["carol"] == 35 && len(ages) == 3)
	ages["bob"]++
	ages["carol"] += 5
	check("map-compound", ages["bob"] == 26 && ages["carol"] == 40)
	check("map-missing-zero", ages["nobody"] == 0)
	v1, ok3 := ages["alice"]
	_, ok4 := ages["nobody"]
	check("map-comma-ok", v1 == 30 && ok3 && !ok4)
	delete(ages, "bob")
	check("map-delete", len(ages) == 2 && ages["bob"] == 0)
	korder := ""
	vsum := 0
	for k, v := range ages { // insertion order in this subset
		korder += k
		vsum += v
	}
	check("map-range-order", korder == "alicecarol" && vsum == 70)
	made := make(map[string]int)
	for _, wrd := range words {
		made[wrd]++
	}
	check("map-counter-idiom", made["a"] == 2 && made["b"] == 1)
	mi := map[int]string{1: "one", 2: "two"}
	check("map-int-keys", mi[2] == "two" && mi[9] == "")

	// ----- structs and methods -----
	p := Point{3, -4}
	check("struct-fields", p.x == 3 && p.y == -4)
	p.y = 4
	check("struct-field-write", p.y == 4)
	check("struct-method", p.manhattan() == 7)
	d := p.scale(3)
	check("struct-method-returns-struct", d.x == 9 && d.y == 12)
	check("struct-method-chain", Point{1, 2}.scale(10).manhattan() == 30)
	pw, ph := Point{3, 4}.dims()
	check("struct-method-multi-return", pw == 3 && ph == 4)

	ln := Line{Point{1, 2}, Point{3, 4}}
	check("struct-nested", ln.a.x == 1 && ln.b.y == 4)
	ln.b.x = 30
	check("struct-nested-write", ln.b.x == 30)

	cnt := Counter{0}
	cnt.inc()
	cnt.inc()
	cnt.inc()
	check("ptr-receiver-mutates", cnt.get() == 3)
	cnt.add(10)
	check("ptr-receiver-arg", cnt.get() == 13)

	// ----- pointers as identity, nil -----
	n := 42
	ptr := &n
	check("ptr-deref-identity", *ptr == 42)
	nd := Node{5, nil}
	check("ptr-struct-param", headVal(&nd) == 5)

	// ----- type assertion is the identity -----
	av := 7
	check("type-assert", av.(int) == 7 && av.(int)+3 == 10)

	// ----- combined pipeline -----
	tag := func(n int) string { return "odd" }
	check("combined-pipeline", classify([]int{1, 2, -3, 4}, tag) == "oddevennegeven1")

	fmt.Println("features:", checks, "checks,", fails, "failures")
	os.Exit(fails)
}
