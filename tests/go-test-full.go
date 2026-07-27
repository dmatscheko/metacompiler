//go:build ignore

// Full-syntax test: Go (Go 1.22 core language).
//
// This file belongs to the SECOND test group (./test.sh --full): it is NOT part
// of the default matrix. The goal of the metacompiler is to support the full
// languages; this file is the ratchet that measures how far the go grammars
// are. It walks the whole practical Go syntax, one self-contained SECTION per
// language area. The --full runner runs the file, and whenever a grammar
// aborts it removes the section around the error and retries - so the report
// lists every unsupported section, not just the first.
//
// Conventions (shared by every *-test-full.* file):
//   - prologue (before the first SECTION marker): the check helper only
//   - each section: '// ===== SECTION <nn>: <name> =====', top-level,
//     self-contained, no references to other sections
//   - main() calls each section via a line tagged 'SECTION-CALL <nn>'
//     and prints the summary line 'full: <checks> checks, <failures> failures'
//   - main() exits with the failure count (exit 0 == full support, verified)
//
// Deliberately out of scope (not syntax, or unrunnable in this harness):
// imports and the standard library (the prologue's fmt/os calls are the whole
// harness, mirroring the feature-matrix file), packages and modules, cgo,
// unsafe, build tags, reflection (struct tags appear as syntax only), and
// goroutine scheduling beyond deterministic channel operations; generic
// method constraints do not exist in Go 1.22 and are n/a. Unlike the
// feature-matrix subset this file is real Go: len() counts bytes, range over
// a string yields runes, and type assertions need an interface.
//
// Hand-written for the metacompiler project (Apache-2.0, no copied test-suite
// code), organized after the Go language specification (version 1.22).

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

// ===== SECTION 01: baseline =====
// Condensed re-assertion of the feature-matrix basics this file builds on.
func s01() {
	n := 0
	for i := 0; i < 3; i++ {
		n += i
	}
	check("bas1", n == 3)
	m := map[string]int{"a": 1}
	m["b"] = 2
	check("bas2", m["a"]+m["b"] == 3)
	sl := []int{1, 2}
	sl = append(sl, 3)
	check("bas3", len(sl) == 3 && sl[2] == 3)
	add := func(x, y int) int { return x + y }
	check("bas4", add(2, 3) == 5)
	s := "go"
	s += "!"
	check("bas5", s == "go!" && len(s) == 3)
}

// ===== SECTION 02: numeric literal forms =====
func s02() {
	check("num1", 0xFF == 255 && 0xff == 255 && 017 == 15)
	check("num2", 0b1010 == 10 && 0o17 == 15)
	check("num3", 1_000_000 == 1000000 && 0x_FF_f == 4095)
	check("num4", 1e3 == 1000.0 && 2.5e-2 == 0.025 && .5 == 0.5 && 5. == 5.0)
	check("num5", 0x1p4 == 16.0 && 0x1.8p1 == 3.0) // hex float literals
	c := 1 + 2i                                    // imaginary literal: complex128
	check("num6", real(c) == 1 && imag(c) == 2)
	check("num7", c*c == -3+4i && complex(3, 4) == 3+4i && imag(2i*2i) == 0)
}

// ===== SECTION 03: rune and string literals =====
func s03() {
	check("chr1", 'A' == 65 && 'z'-'a' == 25 && 'é' == 233)
	check("chr2", '\n' == 10 && '\t' == 9 && '\\' == 92 && '\'' == 39)
	check("chr3", '\x41' == 'A' && '\101' == 'A' && 'é' == 'é' && '\U0001F600' == 128512)
	check("str1", "a\tb"[1] == 9 && "\x41é" == "Aé" && "\"q\"" == `"q"`)
	raw := `a\nb
c`
	check("str2", len(raw) == 6 && raw[1] == '\\' && raw[4] == '\n')
	check("str3", "h\xc3\xa9" == "hé" && "日" == "日")
}

// ===== SECTION 04: constants and iota =====
const (
	red04 = iota // 0, then 1, 2 down the block
	green04
	blue04
)

const (
	_    = iota             // 0 discarded
	kb04 = 1 << (10 * iota) // repeats with iota = 1, 2
	mb04
)

const huge04 = 1 << 40 // untyped: wider than any int32

func s04() {
	check("cst1", red04 == 0 && green04 == 1 && blue04 == 2)
	check("cst2", kb04 == 1024 && mb04 == 1048576)
	check("cst3", huge04/(1<<30) == 1024)
	const local04 = 3 * 7
	check("cst4", local04 == 21)
	var f float64 = 3 // untyped constants convert implicitly
	i := 5
	check("cst5", f == 3.0 && float64(i)/2 == 2.5 && int('A') == 65)
	t := int64(7)
	check("cst6", t+1 == 8 && int(t) == 7 && uint8(t) == 7)
}

// ===== SECTION 05: declarations and assignment =====
var top05 = 10

var a05, b05 = 1, 2

func s05() {
	var i int
	var s string
	var b bool
	check("dec1", i == 0 && s == "" && b == false) // zero values
	x, y := 1, 2
	x, y = y, x
	check("dec2", x == 2 && y == 1)
	var d, e = 4, "five"
	check("dec3", d == 4 && e == "five" && top05 == 10 && a05+b05 == 3)
	n := 1
	{
		n := 2 // shadows the outer n inside this block
		check("dec4", n == 2)
	}
	check("dec5", n == 1)
	_, keep := 1, 2
	m := 3
	m, more := 4, 5 // := needs only one new variable on the left
	check("dec6", keep == 2 && m == 4 && more == 5)
}

// ===== SECTION 06: arrays =====
func s06() {
	var a [3]int
	a[1] = 5
	check("arr1", len(a) == 3 && a[0] == 0 && a[1] == 5)
	b := [3]int{1, 2, 3}
	c := b // arrays are values: c is a copy
	c[0] = 99
	check("arr2", b[0] == 1 && c[0] == 99)
	d := [...]string{"x", "y"} // length inferred from the literal
	e := [5]int{0: 1, 4: 9}    // indexed elements
	check("arr3", len(d) == 2 && d[1] == "y" && e[0] == 1 && e[3] == 0 && e[4] == 9)
	check("arr4", [2]int{1, 2} == [2]int{1, 2} && [2]int{1, 2} != [2]int{2, 1})
	grid := [2][2]int{{1, 2}, {3, 4}}
	check("arr5", grid[1][0] == 3 && grid[0][1] == 2)
	sum := 0
	for i, v := range b {
		sum += i + v
	}
	check("arr6", sum == 9)
}

// ===== SECTION 07: slices =====
func s07() {
	s := []int{2, 3, 5, 7, 11}
	sub := s[1:3]
	check("slc1", len(s) == 5 && cap(s) == 5 && len(sub) == 2 && cap(sub) == 4)
	sub[0] = 30 // slices share the backing array
	check("slc2", s[1] == 30 && sub[0] == 30)
	three := s[:2:3] // three-index slice caps the capacity
	check("slc3", len(three) == 2 && cap(three) == 3 && len(s[2:]) == 3 && len(s[:]) == 5)
	m := make([]int, 2, 8)
	check("slc4", len(m) == 2 && cap(m) == 8 && m[1] == 0)
	var ns []int // nil slice: append allocates
	ns = append(ns, 1, 2)
	ns = append(ns, []int{3, 4}...)
	check("slc5", len(ns) == 4 && ns[3] == 4 && ns != nil)
	dst := make([]int, 3)
	check("slc6", copy(dst, s) == 3 && dst[1] == 30)
	rows := [][]int{{1}, {2, 3}}
	check("slc7", rows[1][1] == 3 && len(rows[0]) == 1)
}

// ===== SECTION 08: maps =====
func s08() {
	ages := map[string]int{"ann": 30, "bob": 25}
	ages["cy"] = 35
	ages["bob"]++
	check("map1", ages["ann"] == 30 && ages["cy"] == 35 && ages["bob"] == 26 && len(ages) == 3)
	v, ok := ages["ann"]
	_, missing := ages["zed"]
	check("map2", v == 30 && ok && !missing && ages["zed"] == 0)
	delete(ages, "bob")
	check("map3", len(ages) == 2 && ages["bob"] == 0)
	letters, sum := 0, 0
	for k, n := range ages { // iteration order is unspecified: use sums
		letters += len(k)
		sum += n
	}
	check("map4", letters == 5 && sum == 65)
	made := make(map[int][]string)
	made[1] = append(made[1], "a")
	check("map5", len(made[1]) == 1 && made[1][0] == "a")
	clear(ages) // Go 1.21 builtin
	check("map6", len(ages) == 0)
}

// ===== SECTION 09: structs =====
type point09 struct{ x, y int }

type circle09 struct {
	center point09
	r      int
}

type base09 struct{ id int }

type wrap09 struct {
	base09 // embedded: id is promoted to wrap09
	tag    string
}

type tagged09 struct {
	Name string `json:"name" xml:"n"` // tag syntax only, no reflection
}

func s09() {
	p := point09{1, 2}       // positional literal
	q := point09{y: 4, x: 3} // keyed literal
	check("stc1", p.x == 1 && q.x == 3 && q.y == 4)
	check("stc2", p == point09{1, 2} && p != q && point09{} == point09{x: 0})
	c := circle09{center: point09{5, 6}, r: 7}
	c.center.y = 60
	check("stc3", c.center.x == 5 && c.center.y == 60 && c.r == 7)
	w := wrap09{base09{9}, "t"}
	check("stc4", w.id == 9 && w.base09.id == 9 && w.tag == "t")
	anon := struct{ a, b int }{10, 20}
	check("stc5", anon.a+anon.b == 30)
	check("stc6", tagged09{Name: "n"}.Name == "n")
}

// ===== SECTION 10: pointers and new =====
type box10 struct{ v int }

func bump10(n *int) { *n += 5 }

func s10() {
	n := 1
	p := &n
	*p = 2
	check("ptr1", n == 2 && *p == 2 && p == &n)
	bump10(&n)
	check("ptr2", n == 7)
	q := new(int) // new(T) allocates a zeroed T, gives *T
	*q += 3
	check("ptr3", *q == 3)
	b := &box10{v: 4}
	b.v++ // struct pointers auto-dereference on selection
	check("ptr4", b.v == 5 && (*b).v == 5)
	var nilP *box10
	check("ptr5", nilP == nil)
	pp := &p
	**pp = 8
	check("ptr6", n == 8)
}

// ===== SECTION 11: functions =====
func named11(a int) (q, r int) {
	q, r = a/3, a%3
	return // bare return uses the named results
}

func sum11(xs ...int) int {
	t := 0
	for _, x := range xs {
		t += x
	}
	return t
}

func pre11(head int, rest ...int) int { return head*100 + len(rest) }

func twice11(f func(int) int, v int) int { return f(f(v)) }

func s11() {
	nq, nr := named11(10)
	check("fun1", nq == 3 && nr == 1)
	check("fun2", sum11() == 0 && sum11(1, 2, 3) == 6 && pre11(3, 4, 5) == 302)
	nums := []int{4, 5, 6}
	check("fun3", sum11(nums...) == 15) // spread a slice into variadic
	var op func(int, int) int = func(x, y int) int { return x * y }
	check("fun4", op(6, 7) == 42)
	adder := func(n int) func(int) int {
		return func(x int) int { return x + n }
	}
	check("fun5", twice11(adder(3), 10) == 16)
}

// ===== SECTION 12: closures and loop variables =====
func counter12() func() int {
	n := 0
	return func() int {
		n++
		return n
	}
}

func s12() {
	c1 := counter12()
	c2 := counter12()
	c1()
	check("clo1", c1() == 2 && c2() == 1) // independent captured state
	mul := 2
	f := func() int { return mul * 3 }
	mul = 5 // closures see the variable, not a snapshot
	check("clo2", f() == 15)
	var fs []func() int
	for i := 0; i < 3; i++ { // Go 1.22: i is a fresh variable per iteration
		fs = append(fs, func() int { return i })
	}
	check("clo3", fs[0]() == 0 && fs[1]() == 1 && fs[2]() == 2)
}

// ===== SECTION 13: defer =====
var log13 = ""
var seen13 = 0

func lifo13() string {
	log13 = ""
	defer func() { log13 += "c" }()
	defer func() { log13 += "b" }() // deferred calls run LIFO
	log13 += "a"
	return log13 // evaluated before the defers run
}

func note13(v int) { seen13 = v }

func args13() int {
	x := 1
	defer note13(x) // defer arguments are evaluated at defer time
	x = 50
	return x
}

func double13() (r int) {
	defer func() { r *= 2 }() // a deferred func may change named results
	return 21
}

func s13() {
	check("dfr1", lifo13() == "a" && log13 == "abc")
	check("dfr2", args13() == 50 && seen13 == 1)
	check("dfr3", double13() == 42)
}

// ===== SECTION 14: methods =====
type ctr14 struct{ n int }

func (c ctr14) get() int   { return c.n }
func (c ctr14) bumpV()     { c.n++ } // value receiver mutates a copy
func (c *ctr14) bumpP()    { c.n++ } // pointer receiver mutates the caller
func (c *ctr14) add(d int) { c.n += d }

func s14() {
	c := ctr14{1}
	c.bumpV()
	check("mth1", c.get() == 1) // the copy changed, c did not
	c.bumpP()                   // shorthand for (&c).bumpP()
	c.add(10)
	check("mth2", c.n == 12)
	p := &c
	p.bumpP()
	check("mth3", p.get() == 13) // value method through a pointer
	mv := c.get                  // method value: receiver copied at bind time
	c.n = 99
	check("mth4", mv() == 13)
	mp := c.bumpP // binds &c: mutations reach c
	mp()
	check("mth5", c.n == 100)
	ge := ctr14.get // method expression: receiver becomes argument
	pa := (*ctr14).add
	pa(&c, 1)
	check("mth6", ge(ctr14{7}) == 7 && c.n == 101)
}

// ===== SECTION 15: interfaces and type switches =====
type shape15 interface{ area() int }

type named15 interface {
	shape15 // interface embedding
	name() string
}

type rect15 struct{ w, h int }

func (r rect15) area() int    { return r.w * r.h }
func (r rect15) name() string { return "rect" }

func s15() {
	var s shape15 = rect15{3, 4} // satisfaction is implicit
	check("ifc1", s.area() == 12)
	var n named15 = rect15{1, 5}
	var up shape15 = n // named15 includes shape15
	check("ifc2", n.name() == "rect" && up.area() == 5)
	var e any = "hello" // any is interface{}
	str, ok := e.(string)
	_, bad := e.(int) // two-result form does not panic
	check("ifc3", str == "hello" && ok && !bad)
	check("ifc4", s.(rect15).w == 3) // single-result assertion
	var zero shape15
	check("ifc5", zero == nil)
	got := ""
	isum := 0
	for _, v := range []any{1, "s", true, nil, 2.5} {
		switch t := v.(type) {
		case int:
			isum += t
			got += "i"
		case string, bool: // multi-type case: t keeps type any
			got += "m"
		case nil:
			got += "n"
		default:
			got += "d"
		}
	}
	check("ifc6", got == "immnd" && isum == 1)
}

// ===== SECTION 16: generics =====
type num16 interface{ ~int | ~float64 } // type union with approximation

func double16[T num16](v T) T { return v + v }

func first16[T any](s []T) T { return s[0] }

func eq16[T comparable](a, b T) bool { return a == b }

type pair16[K comparable, V any] struct {
	key K
	val V
}

func (p pair16[K, V]) first() K { return p.key }

type myInt16 int

func s16() {
	check("gen1", double16(3) == 6)            // T inferred as int
	check("gen2", double16[float64](1.5) == 3) // explicit instantiation
	check("gen3", double16(myInt16(2)) == 4)   // ~int admits named types
	check("gen4", first16([]string{"a", "b"}) == "a")
	check("gen5", eq16(2, 2) && !eq16("x", "y"))
	p := pair16[string, int]{"n", 1}
	q := pair16[int, bool]{key: 3}
	check("gen6", p.first() == "n" && p.val == 1 && q.first() == 3 && !q.val)
	f := first16[int] // instantiated generic function as a value
	check("gen7", f([]int{9}) == 9)
}

// ===== SECTION 17: if and switch forms =====
func s17() {
	got := ""
	if m := 10 % 3; m == 1 { // if with init statement
		got = "one"
	} else if m == 2 {
		got = "two"
	} else {
		got = "other"
	}
	check("swt1", got == "one")
	t := 0
	switch x := 2; x { // switch with init; multi-value case
	case 1:
		t = 1
	case 2, 3:
		t = 23
	default:
		t = 9
	}
	check("swt2", t == 23)
	f := 0
	switch 1 {
	case 1:
		f++
		fallthrough // continues into the next case body
	case 2:
		f += 10
	case 3:
		f += 100
	}
	check("swt3", f == 11)
	g := ""
	switch score := 85; { // no tag: first true case wins
	case score >= 90:
		g = "A"
	case score >= 80:
		g = "B"
	default:
		g = "C"
	}
	check("swt4", g == "B")
}

// ===== SECTION 18: loops, labels and goto =====
func s18() {
	n := 0
	for { // infinite for + break
		n++
		if n == 4 {
			break
		}
	}
	check("lop1", n == 4)
	hits := 0
outer18:
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if j == 1 {
				continue outer18
			}
			if i == 2 {
				break outer18
			}
			hits++
		}
	}
	check("lop2", hits == 2)
	k := 0
retry18:
	if k < 3 {
		k++
		goto retry18
	}
	check("lop3", k == 3)
	r := 0
	for i := range 3 { // Go 1.22: range over an int
		r += i
	}
	check("lop4", r == 3)
}

// ===== SECTION 19: channels and goroutines =====
func s19() {
	ch := make(chan int, 2) // buffered: send then receive, one goroutine
	ch <- 1
	ch <- 2
	check("chn1", <-ch == 1 && <-ch == 2)
	done := make(chan string)
	go func() { done <- "hi" }() // blocking receive: deterministic
	check("chn2", <-done == "hi")
	src := make(chan int, 3)
	src <- 3
	src <- 4
	close(src)
	sum := 0
	for v := range src { // range drains until close
		sum += v
	}
	check("chn3", sum == 7)
	after, open := <-src // closed channel: zero value, not open
	check("chn4", after == 0 && !open)
	sel := 0
	ready := make(chan int, 1)
	ready <- 5
	select { // the ready case wins over default
	case v := <-ready:
		sel = v
	default:
		sel = -1
	}
	check("chn5", sel == 5)
}

// ===== SECTION 20: panic and recover =====
func safe20(f func()) (msg string) {
	defer func() {
		if r := recover(); r != nil { // recover turns the panic into a value
			if s, ok := r.(string); ok {
				msg = "caught:" + s
			}
		}
	}()
	f()
	return "clean"
}

func s20() {
	check("pnc1", safe20(func() {}) == "clean")
	check("pnc2", safe20(func() { panic("boom") }) == "caught:boom")
	steps := ""
	check("pnc3", safe20(func() {
		defer func() { steps += "d" }() // defers still run while panicking
		steps += "a"
		panic("x")
	}) == "caught:x" && steps == "ad")
}

// ===== SECTION 21: strings, bytes and runes =====
func s21() {
	s := "héllo" // é is 2 bytes in UTF-8: len counts bytes
	check("sbr1", len(s) == 6 && len("hello") == 5)
	runes, offs := 0, 0
	var second rune
	for i, r := range s { // range decodes runes; i is the byte offset
		runes++
		offs += i
		if i == 1 {
			second = r
		}
	}
	check("sbr2", runes == 5 && offs == 13 && second == 'é') // 0+1+3+4+5
	b := []byte("abc")
	b[0] = 'A'
	check("sbr3", len(b) == 3 && b[2] == 99 && string(b) == "Abc")
	rs := []rune(s)
	check("sbr4", len(rs) == 5 && rs[1] == 'é' && string(rs[:2]) == "hé")
	check("sbr5", string(rune(65)) == "A" && string(rune(233)) == "é")
	check("sbr6", s[0] == 'h' && s[1] == 0xc3 && s[1:3] == "é") // bytes when indexed
}

// ===== SECTION 22: operators and newer builtins =====
func s22() {
	check("opr1", 5&3 == 1 && 5|3 == 7 && 5^3 == 6 && 5&^3 == 4) // &^ is AND NOT
	check("opr2", 1<<4 == 16 && 32>>2 == 8 && -8>>1 == -4)
	check("opr3", ^5 == -6 && ^uint8(0xF0) == 0x0F) // unary ^ complements
	sh := uint(2)
	i := 7
	check("opr4", i<<sh == 28 && i>>sh == 1)
	check("opr5", min(3, 1, 2) == 1 && max(3, 1, 2) == 3) // Go 1.21 builtins
	check("opr6", min(2.5, 2.0) == 2.0 && max("a", "b") == "b")
	sl := []int{7, 8}
	clear(sl) // Go 1.21: zeroes slice elements in place
	check("opr7", sl[0] == 0 && sl[1] == 0 && len(sl) == 2)
	x := 5
	x &^= 1
	x |= 8
	x ^= 2
	x <<= 1
	x >>= 1
	check("opr8", x == 14) // 5 &^1 = 4, |8 = 12, ^2 = 14, <<1 = 28, >>1 = 14
	check("opr9", 7/2 == 3 && -7/2 == -3 && 7%3 == 1 && -7%3 == -1)
}

// ===== SECTION 23: floating point arithmetic =====
// float64/float32 are a DIFFERENT type from int, and every value below was
// checked against `go run` (go 1.26) before it was written down. The section
// exists because both grammars used to evaluate every arithmetic operator in 32
// bit integers: a/b with a = 1.0 and b = 3.0 was 0, 2.5 * 1.5 was 3, and +Inf /
// -Inf / NaN did not exist at all. Note that the assertions run through
// VARIABLES wherever the answer matters: Go folds untyped constants at
// arbitrary precision, so `1.0/3.0 == 0.3333333333333333` is false in real Go
// and would test the wrong thing.
type point23 struct {
	x float64
}

func (p point23) half() float64 { return p.x / 2 }

func fs23(f float64) string { return fmt.Sprint(f) }

func s23() {
	a, b := 1.0, 3.0
	check("flt1", a/b == 0.3333333333333333)
	check("flt2", 7/2 == 3 && -7/2 == -3)
	check("flt3", float64(7)/2 == 3.5 && a/2 == 0.5)
	check("flt4", 2.5*1.5 == 3.75 && 7.0-0.5 == 6.5 && 3.0*2.0 == 6.0)
	check("flt5", fs23(a) == "1" && fs23(2.5) == "2.5" && fs23(a/b) == "0.3333333333333333")
	// The infinities and NaN, which integer arithmetic could not produce.
	var z float64 = 0
	check("flt6", fs23(a/z) == "+Inf" && fs23(-a/z) == "-Inf" && fs23(z/z) == "NaN")
	check("flt7", z/z != z/z)
	// fmt's %v is FormatFloat(f, 'g', -1, 64): scientific below 1e-4 and from 1e6.
	check("flt8", fs23(1e20) == "1e+20" && fs23(1.5e-8) == "1.5e-08")
	check("flt9", fs23(1e-3) == "0.001" && fs23(100000.0) == "100000" && fs23(1e6) == "1e+06")
	// A float64 declaration converts its initializer; ++ and the compound
	// operators keep the variable a float64.
	var d float64 = 1
	check("flt10", fs23(d) == "1")
	d += 0.25
	check("flt11", d == 1.25)
	d++
	check("flt12", d == 2.25)
	d *= 2
	check("flt13", d == 4.5)
	d /= 3
	check("flt14", d == 1.5)
	d--
	check("flt15", d == 0.5 && fs23(d) == "0.5")
	check("flt16", fs23(-2.5) == "-2.5" && -d == -0.5)
	n29 := 2.9
	check("flt17", int(d*4) == 2 && float64(int(n29)) == 2.0)
	check("flt18", d > 0.25 && d < 1 && d <= 0.5 && d >= 0.5 && d == 0.5 && d != 0.4)
	// A float64 survives a struct field, a slice element and a method boundary.
	p := point23{x: 5.0}
	check("flt19", p.half() == 2.5 && fs23(p.x) == "5")
	xs := []float64{0.5, 1.5}
	check("flt20", xs[0]+xs[1] == 2.0 && fs23(xs[0]*2) == "1")
	var f32 float32 = 3
	check("flt21", f32/2 == 1.5)
	check("flt22", min(2.5, 2.0) == 2.0 && max(2.5, 2.0) == 2.5)
	sum := 0.0
	for i := 0; i < 4; i++ {
		sum += 0.25
	}
	check("flt23", sum == 1.0 && fs23(sum) == "1")
}

// ===== SECTION 24: fmt's value rendering =====
// What fmt.Println actually writes for each kind of value. Every expected string
// below was taken from `go run` (go 1.26.5) before it was written down. The
// section exists because the COMPILER half used to hand the raw runtime value to
// Go's %v - so a map printed as map[__dict:true keys:[a] vals:[1] zero:0] and a
// struct as its class descriptor - and because both halves printed a map in
// INSERTION order where fmt sorts the keys, and neither half honoured
// fmt.Stringer. The assertions go through fmt.Sprint / fmt.Sprintf so they
// compare strings.
//
// Three renderings are deliberately NOT asserted here, because both halves
// knowingly differ from go (they agree with each other, which is what
// ./test.sh --cross measures):
//
//	&P{1}           go: &{1}     here: {1}    - a struct value IS a reference in
//	                                            this model, so a struct and a
//	                                            pointer to it are one value
//	&n (n an int)   go: 0xc00...  here: &5    - an address is not reproducible
//	var m map[K]V   go: map[]     here: <nil> - a nil map is the null value here
type stringer24 struct{ n int }

func (s stringer24) String() string { return "S<" + fmt.Sprint(s.n) + ">" }

type pt24 struct {
	x int
	y int
}

type nest24 struct {
	a int
	b pt24
}

func s24() {
	// nil, the scalars and a rune (which is an integer, so it prints as one).
	var e error = nil
	var np *pt24 = nil
	check("fmt1", fmt.Sprint(e) == "<nil>" && fmt.Sprint(np) == "<nil>")
	check("fmt2", fmt.Sprint(true) == "true" && fmt.Sprint(false) == "false")
	check("fmt3", fmt.Sprint(42) == "42" && fmt.Sprint(-7) == "-7")
	check("fmt4", fmt.Sprint("hi") == "hi" && fmt.Sprint('a') == "97")
	// A SLICE and an ARRAY print the same way - space separated, no commas - and a
	// slice shows only its own window, not the whole backing array.
	sl := []int{1, 2, 3, 4}
	ar := [3]int{7, 8, 9}
	check("fmt5", fmt.Sprint(sl) == "[1 2 3 4]" && fmt.Sprint(ar) == "[7 8 9]")
	check("fmt6", fmt.Sprint(sl[1:3]) == "[2 3]" && fmt.Sprint(sl[:0]) == "[]")
	var nilsl []int
	check("fmt7", fmt.Sprint(nilsl) == "[]" && fmt.Sprint([]string{"a", "b"}) == "[a b]")
	// A map prints map[k:v ...] with the keys SORTED (fmt has done that since Go
	// 1.12), numerically for a numeric key type and bytewise for a string one.
	ms := map[string]int{"c": 3, "a": 1, "b": 2}
	check("fmt8", fmt.Sprint(ms) == "map[a:1 b:2 c:3]")
	mi := map[int]string{33: "x", 2: "b", 10: "j"}
	check("fmt9", fmt.Sprint(mi) == "map[2:b 10:j 33:x]")
	check("fmt10", fmt.Sprint(map[string]int{}) == "map[]")
	// A struct is its field values in declaration order, braced.
	check("fmt11", fmt.Sprint(pt24{1, 2}) == "{1 2}")
	check("fmt12", fmt.Sprint(nest24{1, pt24{2, 3}}) == "{1 {2 3}}")
	check("fmt13", fmt.Sprint([]pt24{{1, 2}, {3, 4}}) == "[{1 2} {3 4}]")
	check("fmt14", fmt.Sprint(map[string]pt24{"k": {9, 8}}) == "map[k:{9 8}]")
	// fmt.Stringer wins over the memberwise form, at every nesting depth.
	check("fmt15", fmt.Sprint(stringer24{3}) == "S<3>")
	check("fmt16", fmt.Sprint([]stringer24{{1}, {2}}) == "[S<1> S<2>]")
	// Println separates every pair of operands; Print (and Sprint) only two
	// operands NEITHER of which is a string.
	check("fmt17", fmt.Sprint(1, 2) == "1 2" && fmt.Sprint(1, "a", 2) == "1a2")
	check("fmt18", fmt.Sprint("a", "b") == "ab" && fmt.Sprint() == "")
	// The format verbs both halves implement.
	check("fmt19", fmt.Sprintf("%v|%d|%s|%t", ms, 5, "z", true) == "map[a:1 b:2 c:3]|5|z|true")
	check("fmt20", fmt.Sprintf("%v %v", pt24{1, 2}, sl) == "{1 2} [1 2 3 4]")
	check("fmt21", fmt.Sprintf("%q %q", "a\"b", 'a') == "\"a\\\"b\" 'a'")
	check("fmt22", fmt.Sprintf("%c%c%%", 65, 0x4e2d) == "A中%")
	check("fmt23", fmt.Sprintf("%s|%v", stringer24{4}, stringer24{5}) == "S<4>|S<5>")
	check("fmt24", fmt.Sprintf("no verbs") == "no verbs" && fmt.Sprintf("%d", -3) == "-3")
	// A float64 renders through the same path, boxed value and all.
	half := 1.0 / 2.0
	check("fmt25", fmt.Sprint(half) == "0.5" && fmt.Sprint([]float64{1.5, 2}) == "[1.5 2]")
	check("fmt26", fmt.Sprintf("%v %v", half, pt24{}) == "0.5 {0 0}")
	// The two BUILTINS. Real Go writes both to stderr and formats a pointer or a
	// slice as an address; here they write to stdout and render the value, which
	// is what makes them reproducible. Only their EXISTENCE can be asserted - the
	// interpreter half used to abort with "unknown name: println" right here.
	println("s24: builtin println", 1, true)
	print("s24: builtin print", 2, "\n")
}

// ===== END SECTIONS =====

// ===== SECTION 25: sized integers =====
// Go's integer types have a WIDTH and a signedness, and both survive into the
// next operation: 127 + 1 is -128 at int8 and 128 at int. Everything here is
// written through VARIABLES on purpose - Go folds untyped CONSTANT expressions
// at arbitrary precision, so `check("x", int8(127)+1 == -128)` would not
// compile and `var x = 1 << 62` is a constant, not the runtime shift.
func s25() {
	var i8 int8 = 127
	i8++
	check("int1", i8 == -128)
	var u8 uint8 = 255
	u8++
	check("int2", u8 == 0)
	var i16 int16 = 32767
	i16++
	check("int3", i16 == -32768)
	var u16 uint16 = 65535
	u16++
	check("int4", u16 == 0)
	var i32 int32 = 2147483647
	i32++
	check("int5", i32 == -2147483648)
	var u32 uint32 = 4294967295
	u32++
	check("int6", u32 == 0)

	// 64 bits of precision: a double carries 53, so these are the values a
	// number-based value model cannot hold.
	var i64 int64 = 9223372036854775807
	check("int7", fmt.Sprint(i64) == "9223372036854775807")
	j := i64
	j++
	check("int8", j == -9223372036854775808)
	check("int9", fmt.Sprint(j) == "-9223372036854775808")
	var u64 uint64 = 18446744073709551615
	check("int10", fmt.Sprint(u64) == "18446744073709551615")
	k := u64
	k++
	check("int11", k == 0)

	// Shifts run at the full width, not at 32 bits.
	one := 1
	check("int12", fmt.Sprint(one<<62) == "4611686018427387904")
	check("int13", fmt.Sprint(one<<63) == "-9223372036854775808")
	var s8 uint8 = 1
	check("int14", s8<<9 == 0)

	// A product that leaves the exactly representable range.
	big := int64(1000000007)
	check("int15", fmt.Sprint(big*big) == "1000000014000000049")

	// Unsigned comparison and unsigned shift are not the signed ones.
	var um uint64 = 18446744073709551615
	check("int16", um > 0)
	check("int17", fmt.Sprint(um>>1) == "9223372036854775807")
	var neg int32 = -8
	check("int18", neg>>1 == -4)
	check("int19", fmt.Sprint(uint32(4294967288)>>1) == "2147483644")

	// Truncated division and remainder take the dividend's sign.
	n := -7
	d := 2
	check("int20", n/d == -3)
	check("int21", n%d == -1)

	// Conversions wrap; ^x complements within the operand's own width.
	c1 := 300
	c2 := -1
	check("int22", int8(c1) == 44)
	check("int23", uint8(c2) == 255)
	check("int24", fmt.Sprint(^one) == "-2")
	var m8 uint8 = 0xF0
	check("int25", ^m8 == 0x0F)
	var x8 int8 = -128
	check("int26", -x8 == -128)

	// A sized value keeps its width through arithmetic and through a struct
	// field, and float64() of a 64 bit integer is the float, not the integer.
	var w8 int8 = 100
	check("int27", w8*2 == -56)
	check("int28", fmt.Sprint(float64(i64)) == "9.223372036854776e+18")
	var arr [4]int
	var idx uint8 = 2
	arr[idx] = 9
	check("int29", arr[2] == 9)
	check("int30", fmt.Sprintf("%d %v", i8, u8) == "-128 0")
}

func main() {
	s01() // SECTION-CALL 01
	s02() // SECTION-CALL 02
	s03() // SECTION-CALL 03
	s04() // SECTION-CALL 04
	s05() // SECTION-CALL 05
	s06() // SECTION-CALL 06
	s07() // SECTION-CALL 07
	s08() // SECTION-CALL 08
	s09() // SECTION-CALL 09
	s10() // SECTION-CALL 10
	s11() // SECTION-CALL 11
	s12() // SECTION-CALL 12
	s13() // SECTION-CALL 13
	s14() // SECTION-CALL 14
	s15() // SECTION-CALL 15
	s16() // SECTION-CALL 16
	s17() // SECTION-CALL 17
	s18() // SECTION-CALL 18
	s19() // SECTION-CALL 19
	s20() // SECTION-CALL 20
	s21() // SECTION-CALL 21
	s22() // SECTION-CALL 22
	s23() // SECTION-CALL 23
	s24() // SECTION-CALL 24
	s25() // SECTION-CALL 25
	fmt.Println("full:", checks, "checks,", fails, "failures")
	os.Exit(fails)
}
