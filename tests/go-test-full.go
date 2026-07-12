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

// ===== END SECTIONS =====

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
	fmt.Println("full:", checks, "checks,", fails, "failures")
	os.Exit(fails)
}
