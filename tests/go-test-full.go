//go:build ignore

//go:debug panicnil=0

// The directive above: the repository's go.mod says go 1.16, under which a
// toolchain recovers panic(nil) as nil; this file asserts the Go 1.21+ answer
// (a *runtime.PanicNilError), as the grammars implement it.

// Full-syntax test: Go (Go 1.22 core language) - part 1 of 2, sections 01 to 82;
// tests/go-test-full.2.go has the rest, and says why the ratchet is two files.
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
// harness, mirroring the feature-matrix file; SECTION 46 adds fmt.Errorf and
// the errors package, SECTIONS 76, 77 and 80 the math package and fmt.Fprint* on
// an io.Writer), packages and modules, cgo,
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
	"errors"
	"fmt"
	f27 "fmt"
	"io"
	"math"
	"os"
	"strings"
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
	// A TWO-name multi-assign whose first element is an index, a map index, a
	// type assertion or a channel receive: each of those is also the prefix of a
	// comma-ok form, which must not swallow it and reject the statement at the
	// comma. Every operand comes out of a container so nothing folds.
	xs05 := []int{10, 20, 30}
	ys05 := []int{40, 50, 60}
	mp05 := map[string]int{"k": 7}
	var iv05 interface{} = 11
	ch05 := make(chan int, 2)
	ch05 <- 13
	i05 := 1
	p1, p2 := xs05[i05], ys05[i05]
	check("dec7", p1 == 20 && p2 == 50)
	p3, p4 := mp05["k"], xs05[0]
	check("dec8", p3 == 7 && p4 == 10)
	p5, p6 := iv05.(int), ys05[2]
	check("dec9", p5 == 11 && p6 == 60)
	p7, p8 := <-ch05, xs05[2]
	check("dec10", p7 == 13 && p8 == 30)
	p9, p10 := xs05[i05], 5 // index first, literal second
	check("dec11", p9 == 20 && p10 == 5)
	var q1, q2 int
	q1, q2 = xs05[0], mp05["k"] // the '=' spelling of the same shape
	check("dec12", q1 == 10 && q2 == 7)
	q3, q4, q5 := xs05[0], mp05["k"], ys05[i05] // three names, unaffected
	check("dec13", q3 == 10 && q4 == 7 && q5 == 50)
	// and the comma-ok forms themselves still read as comma-ok
	cv, cok := mp05["k"]
	check("dec14", cv == 7 && cok)
	tv, tok := iv05.(int)
	check("dec15", tv == 11 && tok)
	ch05 <- 15
	rv, rok := <-ch05
	check("dec16", rv == 15 && rok)
	// The comma-ok base is a whole postfix expression, not a bare name; a base
	// carrying a suffix must not fall through to a two-name destructure of the
	// single value (<nil> <nil>).
	nst05 := map[string]map[string]int{}
	nst05["o"] = mp05
	n1, n2 := nst05["o"]["k"]
	check("dec17", n1 == 7 && n2)
	ms05 := []map[string]int{}
	ms05 = append(ms05, mp05)
	n3, n4 := ms05[0]["k"]
	check("dec18", n3 == 7 && n4)
	box05 := struct {
		mp map[string]int
		iv interface{}
	}{mp05, 21}
	n5, n6 := box05.mp["k"]
	check("dec19", n5 == 7 && n6)
	n7, n8 := box05.iv.(int)
	check("dec20", n7 == 21 && n8)
	ivs05 := []interface{}{31, "s"}
	n9, n10 := ivs05[0].(int)
	check("dec21", n9 == 31 && n10)
	n11, n12 := ivs05[1].(int) // a failed assertion is the zero value, not a panic
	check("dec22", n11 == 0 && !n12)
	// The '=' spelling of comma-ok assigns to variables that already exist: it must
	// neither destructure one value nor declare a SHADOW.
	var e1 int
	var e2 bool
	e1, e2 = mp05["k"]
	check("dec23", e1 == 7 && e2)
	e1, e2 = mp05["absent"]
	check("dec24", e1 == 0 && !e2)
	var e3 int
	var e4 bool
	e3, e4 = iv05.(int)
	check("dec25", e3 == 11 && e4)
	{ // the assignment must reach the OUTER variable, not declare a new one here
		e1, e2 = mp05["k"]
		e3, e4 = ivs05[1].(int)
	}
	check("dec26", e1 == 7 && e2 && e3 == 0 && !e4)
	// A MISSING OUTER KEY YIELDS A NIL MAP, and a nil map READS like an empty one:
	// it indexes to the zero value, has length 0, ranges over nothing and prints
	// map[]. The zero value of a map type is a MARKED empty map, which keeps
	// `== nil` true and keeps a STORE a panic.
	n13, n14 := nst05["absent"]["k"]
	check("dec27", n13 == 0 && !n14)
	check("dec28", nst05["absent"]["k"] == 0 && len(nst05["absent"]) == 0)
	var nilm05 map[string]int
	check("dec29", nilm05 == nil && len(nilm05) == 0 && nilm05["k"] == 0)
	nv05, nok05 := nilm05["k"]
	check("dec30", nv05 == 0 && !nok05)
	cnt05 := 0
	// The one-variable range over a nil map.
	for k05 := range nilm05 {
		_ = k05
		cnt05++
	}
	check("dec31", cnt05 == 0 && fmt.Sprint(nilm05) == "map[]")
	// A for-INIT may be a comma-ok form, and it records its names for the Go 1.22
	// per-iteration copy like a ':=' does. The closure below is what per-iteration
	// means.
	fs05 := []func() int{}
	for fv05, fok05 := mp05["k"]; fok05; fok05 = false {
		fs05 = append(fs05, func() int { return fv05 })
	}
	check("dec32", len(fs05) == 1 && fs05[0]() == 7)
	tn05 := 0
	for tv05, tok05 := iv05.(int); tok05; tok05 = false {
		tn05 = tv05
	}
	check("dec33", tn05 == 11)
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
	// AN OUT OF RANGE INDEX IS A RECOVERABLE PANIC, and the bound is the LENGTH,
	// not the backing array: `s[1:3]` has cap 4 here. Every index comes out of a
	// slice so nothing folds. The message text is the engine's, not go's.
	ix07 := []int{0, 2, 3, 7, -1}
	check("slc8", panics07(func() { _ = s[ix07[3]] }) && panics07(func() { _ = s[ix07[4]] }))
	check("slc9", panics07(func() { _ = sub[ix07[1]] }) && sub[ix07[0]] == 30)
	check("slc10", panics07(func() { sub[ix07[1]] = 1 }) && panics07(func() { s[ix07[3]] = 1 }))
	arr07 := [3]int{1, 2, 3}
	check("slc11", panics07(func() { _ = arr07[ix07[2]] }) && panics07(func() { arr07[ix07[2]] = 1 }))
	str07 := []string{"abc"}
	check("slc12", str07[0][ix07[1]] == 99 && panics07(func() { _ = str07[0][ix07[2]] }))
	// A slice EXPRESSION is bounded by the capacity, and s[len:] is legal. Go's
	// full rule is 0 <= lo <= hi <= max <= cap, the three-index form included.
	check("slc13", len(s[ix07[2]:]) == 2 && panics07(func() { _ = s[ix07[3]:] }))
	check("slc14", panics07(func() { _ = s[ix07[0]:ix07[2]:ix07[1]] }) &&
		cap(s[ix07[0]:ix07[1]:ix07[2]]) == 3)
}

// panics07 reports whether f panics. recover() has to see a runtime panic as an
// ordinary Go value, which is what makes the assertions above testable in all
// three engines - the tree walker, llvm.Run and the native binary.
func panics07(f func()) (did bool) {
	defer func() {
		if r := recover(); r != nil {
			did = true
		}
	}()
	f()
	return false
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
	// NESTED COMPOSITE LITERALS whose element type is a MAP, including a doubly
	// nested one: a map type's VALUE text has to be read by bracket depth, since
	// the LAST ']' runs straight past the key of an inner map.
	nst08 := map[string]map[string]int{"o": {"k": 1}, "p": {"z": 2}}
	check("map7", nst08["o"]["k"] == 1 && nst08["p"]["z"] == 2 && len(nst08) == 2)
	sl08 := []map[string]int{{"a": 1}, {"b": 2}}
	check("map8", sl08[0]["a"] == 1 && sl08[1]["b"] == 2)
	deep08 := []map[string]map[string]int{{"a": {"b": 3}}}
	check("map9", deep08[0]["a"]["b"] == 3)
	mix08 := map[string]map[string][]int{"a": {"b": {4, 5}}}
	check("map10", mix08["a"]["b"][1] == 5)
	ar08 := [2]map[string]int{{"a": 1}, {"b": 2}}
	check("map11", ar08[1]["b"] == 2 && len(ar08) == 2)
	// A STORE into a nil map is the one thing it does not share with an empty one.
	var nm08 map[string]int
	check("map12", panics07(func() { nm08["k"] = 1 }))
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
// checked against `go run`. The assertions run through VARIABLES wherever the
// answer matters: Go folds untyped constants at arbitrary precision, so
// `1.0/3.0 == 0.3333333333333333` is false in real Go and would test the wrong
// thing.
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
	// DOUBLES THE SCRIPT HOST MIS-PRINTS. goja's Number-to-String is not always the
	// shortest ROUND-TRIPPING form (~1 in 9,000 doubles) while the frozen host's
	// is exact, so the interpreter half takes a double's digits from floPrec by
	// ROUND-TRIP SEARCH rather than from `"" + a`. Every value here is one goja
	// gets wrong, read out of a slice so nothing folds; the first is the one
	// docs/working-on-this-project.md chapter 4 names.
	gm23 := []float64{
		0.0013060363926342689,
		-2.0313047603008369e-255,
		1.4614656893494439e+288,
		1.6666157901962729e+09,
		-1.8473743683904479e+18,
		2.4600138641387099e+67,
	}
	check("flt24", fs23(gm23[0]) == "0.0013060363926342689")
	check("flt25", fs23(gm23[1]) == "-2.0313047603008369e-255")
	check("flt26", fs23(gm23[2]) == "1.4614656893494439e+288")
	check("flt27", fs23(gm23[3]) == "1.6666157901962729e+09")
	check("flt28", fs23(gm23[4]) == "-1.8473743683904479e+18")
	check("flt29", fs23(gm23[5]) == "2.4600138641387099e+67")
}

// ===== SECTION 24: fmt's value rendering =====
// What fmt.Println actually writes for each kind of value. Every expected string
// below was taken from `go run`. The assertions go through fmt.Sprint /
// fmt.Sprintf so they compare strings.
//
// Two renderings are deliberately NOT asserted here, because both halves
// knowingly differ from go (they agree with each other, which is what
// ./test.sh --cross measures):
//
//	&P{1}           go: &{1}     here: {1}    - a struct value IS a reference in
//	                                            this model, so a struct and a
//	                                            pointer to it are one value
//	&n (n an int)   go: 0xc00...  here: &5    - an address is not reproducible
//
// A nil map prints map[], as in go (the zero value of a map type is a MARKED
// EMPTY MAP), and it is asserted below.
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
	// A verb that does not apply to its operand is go's bad-verb note,
	// %!<verb>(<type>=<value>), naming the operand's dynamic type.
	bf := []float64{1.7, 1.7e92, -2}
	bstr := []string{"hi", ""}
	bb := []bool{true, false}
	bi := []int{42, -3}
	check("fmt27", fmt.Sprintf("%d|%d|%d", bf[0], bf[1], bf[2]) == "%!d(float64=1.7)|%!d(float64=1.7e+92)|%!d(float64=-2)")
	check("fmt28", fmt.Sprintf("%d|%d|%d", bstr[0], bstr[1], bb[0]) == "%!d(string=hi)|%!d(string=)|%!d(bool=true)")
	check("fmt29", fmt.Sprintf("%s|%s|%s", bi[0], bf[0], bb[1]) == "%!s(int=42)|%!s(float64=1.7)|%!s(bool=false)")
	check("fmt30", fmt.Sprintf("%f|%f|%z", bi[1], bstr[0], bi[0]) == "%!f(int=-3)|%!f(string=hi)|%!z(int=42)")
	check("fmt31", fmt.Sprintf("%t|%t|%c|%q|%q", bi[0], bstr[0], bf[0], bf[0], bb[0]) == "%!t(int=42)|%!t(string=hi)|%!c(float64=1.7)|%!q(float64=1.7)|%!q(bool=true)")
	var i8b int8 = -5
	var u8b byte = 65
	var f32b float32 = 0.1
	var nile error
	check("fmt32", fmt.Sprintf("%s|%s|%d|%t", i8b, u8b, f32b, f32b) == "%!s(int8=-5)|%!s(uint8=65)|%!d(float32=0.1)|%!t(float32=0.1)")
	check("fmt33", fmt.Sprintf("%d|%s|%d %s", nile, nile, bi[0], bf[0]) == "%!d(<nil>)|%!s(<nil>)|42 %!s(float64=1.7)")
	check("fmt34", fmt.Sprintf("%c|%q|%d|%d", u8b, u8b, u8b, bi) == "A|'A'|65|[42 -3]")
	// The two BUILTINS. Real Go writes both to stderr and formats a pointer or a
	// slice as an address; here they write to stdout and render the value, which
	// is what makes them reproducible. Only their EXISTENCE can be asserted.
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

	// AND NOT runs at the full width too, not over a 32-bit bitwise pair.
	var an int64 = 1<<40 | 7
	var am int64 = 7
	check("int31", fmt.Sprint(an&^am) == "1099511627776")
	var au uint64 = 18446744073709551615
	check("int32", fmt.Sprint(au&^1) == "18446744073709551614")
	var aa int64 = 1 << 40
	aa &^= 1 << 40
	check("int33", aa == 0)

	// A shift takes its result type from the LEFT operand alone - the count is a
	// separate operand with a type of its own, so a signed value shifted by an
	// unsigned count stays signed and keeps its own width.
	var sv int = -8
	var sc8 uint8 = 1
	check("int34", sv>>sc8 == -4)
	var sv64 int64 = -1024
	var sc16 uint16 = 3
	check("int35", sv64>>sc16 == -128)
	var sl int = 1
	var sc32 uint32 = 40
	check("int36", fmt.Sprint(sl<<sc32) == "1099511627776")
	var sm int64 = -8
	sm >>= sc8
	check("int37", sm == -4)
	// and the count's own width still does not survive onto an unsigned left
	// operand: uint8(0xF0) >> int(1) stays a uint8.
	var su8 uint8 = 0xF0
	var sci int = 1
	check("int38", su8>>sci == 120)
}

// ===== SECTION 26: sized integers in declared slots =====
type sized26 struct {
	b  uint8
	i  int8
	u  uint16
	n  int
	xs []uint8
}

type wide26 struct {
	a int32
	b int64
}

type outer26 struct {
	in sized26
	f  float64
}

func step26(b uint8, i int8) (uint8, int8) {
	b++
	i++
	return b, i
}

func sum26(vals ...uint8) uint8 {
	var t uint8
	for _, v := range vals {
		t += v
	}
	return t
}

func s26() {
	// A struct field keeps its DECLARED width: the zero value is already a
	// uint8, and an assignment into the field converts to it.
	var s sized26
	check("slt1", s.b == 0 && s.i == 0 && s.u == 0 && s.n == 0)
	s.b = 255
	s.b++
	check("slt2", s.b == 0)
	s.i = 127
	s.i++
	check("slt3", s.i == -128)
	s.u = 65535
	s.u += 2
	check("slt4", s.u == 1)
	s.b--
	check("slt5", s.b == 255)
	check("slt6", fmt.Sprint(s.b, s.i, s.u) == "255 -128 1")

	// A composite literal writes the field type too, and so does a nested one.
	t := sized26{b: 255, i: 127}
	t.b++
	t.i++
	check("slt7", t.b == 0 && t.i == -128)
	o := outer26{in: sized26{b: 200}, f: 1}
	o.in.b += 100
	check("slt8", o.in.b == 44)

	// A PARAMETER is a slot with a declared type of its own.
	rb, ri := step26(255, 127)
	check("slt9", rb == 0 && ri == -128)
	check("slt10", sum26(200, 100) == 44)

	// A SLICE element type: literal, make, append and a re-slice all carry it.
	xs := []uint8{255}
	xs[0]++
	check("slt11", xs[0] == 0)
	ys := make([]uint8, 2)
	ys[0] = 255
	ys[0]++
	check("slt12", ys[0] == 0 && ys[1] == 0)
	var zs []uint8
	zs = append(zs, 255)
	zs[0]++
	check("slt13", zs[0] == 0 && len(zs) == 1)
	zs = append(zs, 200, 100)
	zs[1] += 100
	check("slt14", zs[1] == 44 && zs[2] == 100)
	ws := zs[1:]
	ws[0]++
	check("slt15", ws[0] == 45 && zs[1] == 45)
	s.xs = append(s.xs, 255)
	s.xs[0]++
	check("slt16", s.xs[0] == 0)

	// An ARRAY element type, including one indexed past a gap.
	var ar [3]uint8
	ar[0] = 255
	ar[0]++
	check("slt17", ar[0] == 0 && ar[1] == 0)
	br := [3]int8{2: 127}
	br[2]++
	check("slt18", br[2] == -128 && br[0] == 0)

	// A MAP value type, reached through the map's own zero value.
	m := map[string]int8{}
	m["a"] = 127
	m["a"]++
	check("slt19", m["a"] == -128)
	m2 := map[string]uint8{"k": 255}
	m2["k"]++
	check("slt20", m2["k"] == 0 && m2["missing"] == 0)

	// A wider slot does NOT wrap where a narrow one does, so the width really
	// travels with the slot and is not a global setting.
	var wide wide26
	wide.a = 2147483647
	wide.a++
	wide.b = 2147483647
	wide.b++
	check("slt21", wide.a == -2147483648 && wide.b == 2147483648)

	// `var s T` of a struct type is a zero VALUE, not nil, so a field of it can be
	// written on the very next line.
	var one sized26
	one.b = 255
	check("slt22", one.b == 255 && one.n == 0 && len(one.xs) == 0)

	// A []byte's elements are uint8, so they wrap, and the string round trip
	// still reads them as bytes.
	bs := []byte("AB")
	bs[0] += 200
	check("slt23", bs[0] == 9 && string(bs[1:]) == "B")

	// float64 and string fields are unaffected by any of this.
	o.f = 1
	check("slt24", o.f/2 == 0.5)
	check("slt25", fmt.Sprintf("%d %v %d", xs[0], ys, m["a"]) == "0 [0 0] -128")
}

// ===== SECTION 27: unnamed receivers, type aliases, named func types =====
// A method may leave its receiver unnamed when the body does not use it; `type A = B`
// declares an ALIAS (one type with two names, unlike `type A B`); a function TYPE may
// name its parameters and results; and a declaration may carry the pre-1.0 trailing
// semicolon. Verified against `go run`.
type pt27 struct{ x, y int }
type alias27 = pt27
type ints27 = int
type marker27 struct{}

func (marker27) tag() string   { return "value" }
func (*marker27) ptag() string { return "pointer" }

type binop27 func(a, b int) int
type seq27 = func(yield func(v int) bool)

func triple27(n int) int { return 3 * n }

func s27() {
	a := alias27{1, 2}
	check("ali1", a.x+a.y == 3)
	var b pt27 = a // an alias is the SAME type, so no conversion is involved
	check("ali2", b.y == 2)
	var n ints27 = 7
	check("ali3", n == 7)
	check("ali4", f27.Sprint(n) == "7") // aliased import binds the package object
	check("ali5", triple27(4) == 12)
	m := marker27{}
	check("rcv1", m.tag() == "value")
	check("rcv2", m.ptag() == "pointer") // &m taken automatically
	var op binop27 = func(a, b int) int { return a * b }
	check("fnt1", op(6, 7) == 42)
	total := 0
	var it seq27 = func(yield func(v int) bool) {
		for i := 1; i <= 3; i++ {
			if !yield(i) {
				return
			}
		}
	}
	it(func(v int) bool { total += v; return true })
	check("fnt2", total == 6)
}

// ===== SECTION 28: trailing commas, generic instantiation, parenthesized conversions =====
// A call's argument list may end with a comma when it is spread over lines; a generic
// instantiation may carry several type arguments or one that is not an expression
// (struct{}, [62]byte, map[string][]int); a conversion's target type may be
// parenthesized; and `return` before a `case` label ends at the line break. Verified
// against `go run`.
type pair28[A any, B any] struct {
	a A
	b B
}

func make28[A any, B any](x A, y B) pair28[A, B] { return pair28[A, B]{x, y} }

func size28[T any](n int) int { return n }

func sum28(a, b, c int) int { return a + b + c }

// A bare `return` directly in front of a `case` label: Go's semicolon insertion ends
// the statement at the line break, so the `case` is NOT the returned expression.
func early28(i int) (n int) {
	switch i {
	case 0:
		n = 1
	case 1:
		return
	default:
		n = 9
	}
	n += 100
	return
}

func s28() {
	check("trl1", sum28(
		1,
		2,
		3,
	) == 6)
	p := make28[int, string](7, "z")
	check("gen1", p.a == 7 && p.b == "z")
	check("gen2", size28[struct{}](4) == 4)
	check("gen3", size28[[62]byte](5) == 5)
	check("gen4", size28[map[string][]int](6) == 6)
	var ch chan int = (chan int)(nil)
	check("cnv1", ch == nil)
	var fn func() = (func())(nil)
	check("cnv2", fn == nil)
	var mp map[string]int = (map[string]int)(nil)
	check("cnv3", mp == nil)
	check("cnv4", string(([]byte)("hi")) == "hi")
	check("cnv5", ([]int{5, 6})[1] == 6)
	check("ret1", early28(0) == 101)
	check("ret2", early28(1) == 0)
	check("ret3", early28(2) == 109)
}

// ===== SECTION 29: parenthesized composite literals and assignment targets =====
// Go's composite-literal restriction is SYNTACTIC and applies only to a BARE literal in an
// if/for/switch header: `if T{1} == T{2} {` is a syntax error while `if (T{1}) == (T{2}) {`
// is legal, and so is `switch (T{1}) {`. An assignment target may likewise be
// parenthesized - `(*p).f = v`, `(*sp)[i] = v`, `(v) = e` - which is how Go binds a
// dereference to the field or index rather than to the whole expression. Verified
// against `go run`.
type pt29 struct {
	x int
	y int
}

func s29() {
	// a parenthesized composite literal in a header, where a bare one is illegal
	if (pt29{1, 2}) == (pt29{1, 2}) {
		check("plit1", true)
	} else {
		check("plit1", false)
	}
	if (pt29{1, 2}) != (pt29{3, 4}) {
		check("plit2", true)
	} else {
		check("plit2", false)
	}
	switch (pt29{5, 6}) {
	case pt29{5, 6}:
		check("plit3", true)
	default:
		check("plit3", false)
	}
	n := 0
	for i := 0; i < 3; i++ {
		if (pt29{i, i}) == (pt29{1, 1}) {
			n++
		}
	}
	check("plit4", n == 1)
	if q := 2; (pt29{q, q}) == (pt29{2, 2}) {
		check("plit5", true)
	} else {
		check("plit5", false)
	}

	// parenthesized assignment targets
	p := &pt29{7, 8}
	(*p).x = 9
	check("ptgt1", p.x == 9)
	(*p).y += 3
	check("ptgt2", p.y == 11)
	check("ptgt3", (*p).x+(*p).y == 20)
	var v int
	(v) = 41
	(v)++
	check("ptgt4", v == 42)
	sl := []int{1, 2, 3}
	sp := &sl
	(*sp)[1] = 20
	check("ptgt5", sl[1] == 20)
	(*sp)[2] += 7
	check("ptgt6", sl[2] == 10)
	st := []pt29{{1, 2}}
	(st)[0].x = 5
	check("ptgt7", st[0].x == 5)
}

// ===== SECTION 30: defined types, and pointers to arrays =====
// A composite literal of a DEFINED type is a literal of what the type is defined over:
// `type L []int; L{1, 2, 3}` is the []int literal, `type A [4]int; A{7, 8}` keeps the
// declared length and zeroes the rest, and `type Q P; Q{1, 2}` is P's literal. `var l L`
// zeroes the same way. An ARRAY is a VALUE - assigning one copies it - so &arr must be a
// real pointer: h := &arr; h[0] = 5 is seen by arr, and len/cap/index/slice/range all
// reach through it. Verified against `go run`.
type ints30 []int
type grid30 [][]int
type quad30 [4]int
type lookup30 map[string]int
type base30 struct {
	x int
	y int
}
type alias30 base30
type deep30 ints30
type temp30 float64
type byte30 uint8
type rec30 struct{ t temp30 }

func half30(t temp30) temp30 { return t / 2 }

func sum30(p *[3]int) int {
	t := 0
	for _, v := range p {
		t += v
	}
	return t
}

func s30() {
	// a composite literal of a defined SLICE type is a literal of what it is defined over
	a := ints30{1, 2, 3}
	check("dfl1", len(a) == 3)
	check("dfl2", a[0] == 1 && a[2] == 3)
	a = append(a, 4)
	check("dfl3", len(a) == 4 && a[3] == 4)
	check("dfl4", cap(a) >= 4)
	var z ints30
	check("dfl5", len(z) == 0)
	z = append(z, 9)
	check("dfl6", len(z) == 1 && z[0] == 9)
	e := ints30{}
	check("dfl7", len(e) == 0)

	// a chain of defined types resolves to the spelling at the end of it
	d := deep30{5, 6}
	check("dfl8", len(d) == 2 && d[1] == 6)

	// a defined ARRAY type keeps its length, so the unwritten slots are zero
	q := quad30{7, 8}
	check("dfl9", len(q) == 4)
	check("dfl10", q[0] == 7 && q[1] == 8 && q[2] == 0 && q[3] == 0)
	var qz quad30
	check("dfl11", len(qz) == 4 && qz[3] == 0)

	// a defined MAP type zeroes and builds empty
	lk := lookup30{}
	lk["a"] = 1
	check("dfl12", len(lk) == 1 && lk["a"] == 1)
	check("dfl13", lk["missing"] == 0)

	// a defined STRUCT type is a literal of the struct it is defined over
	al := alias30{1, 2}
	check("dfl14", al.x == 1 && al.y == 2)
	al.y = 5
	check("dfl15", al.y == 5)

	// nested brace groups inside a defined slice-of-slice type
	g := grid30{{1, 2}, {3}}
	check("dfl16", len(g) == 2)
	check("dfl17", len(g[0]) == 2 && g[0][1] == 2)
	check("dfl18", len(g[1]) == 1 && g[1][0] == 3)

	// an ARRAY is a value: assigning one copies it, so the copy moves alone
	arr := [3]int{1, 2, 3}
	cp := arr
	cp[0] = 99
	check("parr1", arr[0] == 1 && cp[0] == 99)

	// a POINTER to an array is a reference: a write through it is seen by the array
	h := &arr
	h[0] = 5
	check("parr2", arr[0] == 5)
	h[2] += 4
	check("parr3", arr[2] == 7)
	check("parr4", h[1] == 2)
	check("parr5", len(h) == 3 && cap(h) == 3)
	check("parr6", sum30(h) == 14)
	check("parr7", (*h)[0] == 5)
	sl := h[:]
	check("parr8", len(sl) == 3 && sl[0] == 5)
	sl[1] = 20
	check("parr9", arr[1] == 20)
	tot := 0
	for i, v := range h {
		tot += i * v
	}
	check("parr10", tot == 0*5+1*20+2*7)

	// a pointer to an array of a defined array type, and a pointer taken twice
	g2 := &qz
	g2[1] = 8
	check("parr11", qz[1] == 8)
	hh := &arr
	hh[0] = 11
	check("parr12", h[0] == 11)

	// a slot of a DEFINED numeric type takes what it is defined over: a float, a width
	var t temp30 = 37
	r := rec30{t: 37}
	ts := []temp30{3}
	check("dfl19", t/2 == 18.5 && r.t/2 == 18.5 && half30(37) == 18.5 && ts[0]/2 == 1.5)
	var by byte30 = 255
	by++
	check("dfl20", by == 0)
}

// ===== SECTION 31: local type declarations, and labels with no statement =====
// A `type` declaration is a STATEMENT too: Go scopes a local type to its block, and every
// top-level shape (struct, defined slice/int/map, alias) is legal there. A LABEL may also
// stand alone - `_:` at the end of a block is a labeled EMPTY statement, for which Go
// writes no semicolon; gc rejects the same thing in front of a 'case' or 'default'
// ("missing statement after label"), so the grammar does too. Verified against
// `go run`.
func s31() {
	// a type declaration in STATEMENT position, in every shape the top level has
	type T31 struct {
		a int
		b string
	}
	type S31 []T31
	type N31 int
	type M31 map[string]int
	type A31 = T31

	var e S31
	e = append(e, T31{1, "foo"})
	check("lty1", len(e) == 1)
	check("lty2", e[0].a == 1 && e[0].b == "foo")
	var n N31 = 7
	check("lty3", int(n)+1 == 8)
	m := M31{}
	m["k"] = 3
	check("lty4", m["k"] == 3)
	var al A31
	al.a = 9
	check("lty5", al.a == 9)
	lit := S31{{2, "x"}}
	check("lty6", len(lit) == 1 && lit[0].a == 2 && lit[0].b == "x")

	// a label with NOTHING after it is a labeled EMPTY statement. Go writes no
	// semicolon for it, and the blank '_' label needs no goto to be legal.
	n2 := 0
outer:
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if j == 1 {
				continue outer
			}
			n2 += i
		}
	_:
	}
	check("elbl1", n2 == 3)
	switch n2 {
	default:
		n2 = 5
	case 3:
		n2 = 4
	_:
	}
	check("elbl2", n2 == 4)
_:
}

// ===== SECTION 32: conversions to a written-out array, map, chan or slice type =====
// A conversion's target type may be spelled out in full: `[4]byte(s)`, `[3]int(a)`,
// `map[string]int(m)`, `chan int(c)`, and it may carry the trailing comma a call may
// carry. Converting a SLICE to an ARRAY copies (an array is a value); every other form
// here is a RE-TYPING that keeps the same backing store, so []byte(bs) is bs.
// Verified against `go run`.
func s32() {
	// a conversion whose target type is an ARRAY type written out. Converting a slice
	// to an array COPIES (an array is a value), so the result does not alias the slice.
	s := []byte{1, 2, 3, 4}
	a := [4]byte(s)
	check("cvt1", len(a) == 4 && a[0] == 1 && a[3] == 4)
	a[0] = 9
	check("cvt2", s[0] == 1 && a[0] == 9)
	b := [3]int{1, 2, 3}
	b2 := [3]int(b)
	b2[0] = 7
	check("cvt3", b[0] == 1 && b2[0] == 7)

	// a MAP type written out, and a CHAN type written out
	m := map[string]int{"x": 1}
	m2 := map[string]int(m)
	check("cvt4", m2["x"] == 1)
	m2["y"] = 2
	check("cvt5", m["y"] == 2) // a map conversion is a re-typing: still the same map
	var c chan int
	c2 := chan int(c)
	check("cvt6", c2 == nil)

	// a conversion may carry the trailing comma a call may carry
	check("cvt7", int(1.0) == 1)
	check("cvt8", len([]byte("foo")) == 3)
	check("cvt9", string([]rune{104, 105}) == "hi")

	// []T(x) is a RE-TYPING of the same header and keeps aliasing
	sl := []int{1, 2, 3}
	al := []int(sl)
	al[0] = 8
	check("cvt10", sl[0] == 8)
	bs := []byte{104, 105}
	b3 := []byte(bs)
	b3[0] = 106
	check("cvt11", string(bs) == "ji" && string(b3) == "ji" && len(b3) == 2)
	rs := []rune("hi")
	r3 := []rune(rs)
	check("cvt12", string(r3) == "hi" && len(r3) == 2)
	check("cvt13", string([]byte("ok")) == "ok")
}

// ===== SECTION 33: untyped constants fold at arbitrary precision =====
// Go evaluates a constant expression EXACTLY and rounds ONCE, where the value
// reaches a typed location - so 0.1 + 0.2 is the double nearest three tenths and
// not the sum of the two nearest doubles, and (1 << 100) >> 98 is 4 rather than
// what a 64 bit accumulator answers. Every expression below is a CONSTANT one on
// purpose: the subject is the front end, not the runtime. cst5 is the contrast
// that must NOT fold (a written-out conversion rounds first, which is Go's rule
// too), and cst16/cst19/cst31/cst32 are the reason a comparison of two constants
// has to be exact as well - both sides round to the same double and Go still says
// they are different constants.
func s33() {
	check("cst1", 0.1+0.2 == 0.3)
	check("cst2", 0.3-0.1 == 0.2)
	check("cst3", 1.1*3 == 3.3)
	check("cst4", 2.675*100 == 267.5)
	check("cst5", float64(0.1)+float64(0.2) != 0.3)
	check("cst6", 0.1+0.2+0.3 == 0.6)
	check("cst7", (0.1+0.2)*3 == 0.9)
	check("cst8", 1.0/3.0*3.0 == 1.0)
	check("cst9", -0.1+0.2 == 0.1)
	check("cst10", 1e16+2 == 1.0000000000000002e16)
	check("cst11", 0.1*0.1*0.1 == 0.001)
	check("cst12", 1e308*10/10 == 1e308)
	check("cst13", 1e-300/1e20 == 1e-320)
	check("cst14", 1.0/7*7 == 1.0)
	check("cst15", 123456789012345678901234567890.0*1.0 != 1.2345678901234568e29)
	check("cst16", 2.0/3.0 != 0.6666666666666666)
	check("cst17", 0x1p4 == 16.0)
	check("cst18", 0x1.8p1 == 3.0)
	check("cst19", 1e22+1 != 1e22)
	check("cst20", (1<<100)>>98 == 4)
	check("cst21", 100000000000000000000/10000000000000000000 == 10)
	check("cst22", 1<<62 == 4611686018427387904)
	check("cst23", 1<<200>>200 == 1)
	check("cst24", 0xFFFFFFFFFFFFFFFF/3 == 6148914691236517205)
	check("cst25", 9007199254740993*3/3 == 9007199254740993)
	check("cst26", -7/2 == -3 && -7%2 == -1)
	check("cst27", -9>>2 == -3)
	check("cst28", ^0 == -1 && ^255 == -256)
	check("cst29", 255&^15 == 240)
	check("cst30", 1/3 == 0 && 1/3.0 > 0.33)
	check("cst31", 9007199254740993.0*1.0 != 9007199254740992.0)
	check("cst32", 3*0.1 != 0.30000000000000004)
}

// ===== SECTION 34: a NAMED constant folds too, and what deliberately does not =====
// SECTION 33 folds constant LITERALS; a name needs a scope, and getting that scope
// wrong is a silent wrong answer rather than a missing feature. So half of what is
// below are the CONTROLS: every one of nc10-nc18 is a name the fold has to DECLINE,
// because a type, an iota, a parameter, a local, a range variable, an inner block or
// a switch case owns that name there and Go answers with the binding, not the const.
const c34a = 0.1
const c34b = 0.2
const c34s = c34a + c34b
const c34kb = 1024
const c34mb = c34kb * c34kb
const c34hi = (1 << 100) >> 98
const c34rA = 'A'
const c34rz = 'z'
const c34gap = c34rz - c34rA
const c34third = 1.0 / 3.0
const c34ty float64 = 0.1
const c34tz float64 = 0.2
const c34z = 0.1 + 0.2i
const c34w = 0.2 + 0.1i

const (
	c34i0 = iota
	c34i1
	c34i2
)
const (
	c34k0 = 1 << (10 * iota)
	c34k1
	c34k2
)

func s34param(c34a int, c34b int) int { return c34a * c34b }

func s34local() int {
	c34a := 5
	c34b := 7
	return c34a * c34b
}

func s34rangeVar(xs []int) int {
	t := 0
	for c34a := range xs {
		t += c34a
	}
	return t
}

func s34inner() bool {
	ok := true
	{
		const c34a = 2
		if c34a*c34b != 0.4 {
			ok = false
		}
	}
	return ok && c34a*c34b == 0.02
}

func s34switch(n int) float64 {
	switch n {
	case 1:
		const c34a = 100.0
		return c34a * c34b
	}
	return c34a * c34b
}

func s34() {
	check("nc1", c34s == 0.3)
	check("nc2", c34a+c34b == 0.3)
	check("nc3", c34a*c34b == 0.02)
	check("nc4", c34mb == 1048576)
	check("nc5", c34hi == 4)
	check("nc6", c34gap == 57 && c34rA+1 == 66)
	check("nc7", c34third*3.0 == 1.0)
	check("nc8", c34a+c34b+c34third != 0.6333333333333333)
	check("nc9", c34kb*c34kb*c34kb == 1073741824)
	// A TYPED const is converted - and so rounded - AT its declaration, so this pair
	// really is float64 arithmetic and must keep the unfolded answer.
	check("nc10", c34ty+c34tz != 0.3)
	check("nc11", float64(c34a)+float64(c34b) != 0.3)
	// iota: one source position carrying a different value per spec, which a table
	// keyed by source position cannot express - declined, not half-folded.
	check("nc12", c34i0 == 0 && c34i1 == 1 && c34i2 == 2)
	check("nc13", c34k0 == 1 && c34k1 == 1024 && c34k2 == 1048576)
	// Each of these binds c34a/c34b to something that is NOT the const above.
	check("nc14", s34param(3, 4) == 12)
	check("nc15", s34local() == 35)
	check("nc16", s34rangeVar([]int{9, 9, 9}) == 3)
	check("nc17", s34inner())
	check("nc18", s34switch(1) == 20.0 && s34switch(2) == 0.02)
	// A COMPLEX constant is a PAIR of the same exact rationals, so it folds too
	// (multiplying four already-rounded doubles drifts in both parts).
	check("nc19", (0.1+0.2i)*(0.1+0.2i) == -0.03+0.04i)
	check("nc20", c34z*c34z == -0.03+0.04i && c34z+c34w == 0.3+0.3i)
	check("nc21", real(c34z*c34w) == 0 && imag(c34z*c34w) == 0.05 && 1+0i == 1)
	check("nc22", c34z/c34w == 0.8+0.6i && -c34z+c34z == 0)
}

// ===== SECTION 35: float32 is a real binary32 =====
// A Go float32 is an IEEE-754 binary32, not a float64 spelled differently:
// float32(1)/float32(3) is 0.33333334 and float32(1e20)*float32(1e20) is +Inf.
// Every expected value below was read off `go run`. The operands come out of a
// SLICE on purpose: a probe of literals tests the grammar's constant folder
// rather than the runtime.
func fs35(f float32) string { return fmt.Sprint(f) }

func s35() {
	fs := []float32{1, 3, 0.1, 0.2, 1e20, 16777217}
	// The headline: 24 significant bits, not 53.
	check("f32a", fs35(fs[0]/fs[1]) == "0.33333334")
	check("f32b", fs35(fs[2]+fs[3]) == "0.3")
	// A float64 with the same source text keeps all 53.
	ds := []float64{1, 3, 0.1, 0.2}
	check("f32c", fs23(ds[0]/ds[1]) == "0.3333333333333333" && fs23(ds[2]+ds[3]) == "0.30000000000000004")
	// binary32 OVERFLOWS where a double does not, and it says so Go's way.
	check("f32d", fs35(fs[4]*fs[4]) == "+Inf" && fs23(1e20*1e20) == "1e+40")
	// 16777217 is the first integer a binary32 cannot hold: it rounds to
	// 16777216, which is also what makes the +1 below vanish.
	check("f32e", fs35(fs[5]) == "1.6777216e+07" && fs35(fs[5]+1) == "1.6777216e+07")
	// A conversion ROUNDS, and the rounding is observable through float64().
	check("f32f", float64(float32(0.1)) == 0.10000000149011612)
	// A DECLARED float32 slot rounds its initializer, so it equals the
	// converted constant and not the double.
	var f float32 = 0.1
	check("f32g", f == float32(0.1) && float64(f) == 0.10000000149011612)
	// An untyped constant next to a float32 is converted TO float32 (Go has no
	// implicit float32/float64 conversion at all), so this comparison holds.
	check("f32h", f == 0.1 && !(f < 0.1) && !(f > 0.1))
	// The width survives a parameter, a struct field, a slice element and a
	// zero value - every place a declared type is adopted.
	check("f32i", fs35(add35(fs[2], fs[3])) == "0.3")
	pt := pt35{fs[2], 0.1}
	check("f32j", fs35(pt.a+pt.a) == "0.2" && fs23(pt.b+pt.b) == "0.2")
	check("f32k", fs35(fs[2]*fs[3]) == "0.020000001")
	var zero float32
	check("f32l", fs35(zero) == "0" && fs35(-f) == "-0.1")
	// fmt's %v for a float32 is the same 'g' window as a float64, read at 24
	// bits: scientific below 1e-4 and from 1e6.
	check("f32m", fs35(fs[4]) == "1e+20" && fs35(fs[2]/1000000) == "1e-07")
}

type pt35 struct {
	a float32
	b float64
}

func add35(x float32, y float32) float32 { return x + y }

const cnz36 = -0.0

// ===== SECTION 36: complex128 the way Go computes and prints it =====
// Every expected string was read off `go run`. Unary minus is component-wise,
// both PARTS are float64 and print with the float64 'g' rule (1.234567125e+06,
// +Inf), and division is Smith's algorithm plus the C99 G.5.1 fixups, as Go's
// runtime does it.
func s36() {
	big36 := []float64{1e300}
	res := []float64{0.1, 4, 1234567.125, 0}
	ims := []float64{0.2, 3, -0.5, 0}
	a := complex(res[0], ims[0])
	b := complex(res[1], ims[1])
	// UNARY MINUS, component-wise.
	check("cx1", fmt.Sprint(-a) == "(-0.1-0.2i)")
	check("cx2", -a == complex(-res[0], -ims[0]) && -(-a) == a)
	// A NEGATIVE ZERO survives the negation, which `0 - x` would lose.
	z := complex(res[3], ims[3])
	check("cx3", fmt.Sprint(-z) == "(-0-0i)")
	// Both parts print with the float64 'g' rule.
	check("cx4", fmt.Sprint(complex(res[2], ims[2])) == "(1.234567125e+06-0.5i)")
	check("cx5", fmt.Sprint(a/complex(res[3], ims[3])) == "(+Inf+Infi)")
	// SMITH'S ALGORITHM, at the magnitude that shows it: the textbook
	// formula squares each denominator component, so 1e300+1e300i gives an
	// infinite denominator and a (0+0i) quotient. Scaling by the larger
	// component keeps the ratio at 1 and the answer finite. The neighbouring
	// case a/(3+4i) is deliberately NOT asserted: go fuses its
	// `imag(n)*ratio - real(n)` into an FMA on arm64 and answers 0.008 where
	// every engine without a fused multiply-add answers 0.008000000000000002
	// - see the note in abnf/jsrtgolang.go's goCxDiv.
	check("cx6", fmt.Sprint(a/complex(big36[0], big36[0])) == "(1.5000000000000002e-301+5e-302i)")
	check("cx11", fmt.Sprint(a/b) == "(0.04+0.02i)")
	// real() and imag() are float64 VALUES, so they render as one.
	check("cx7", fmt.Sprint(real(complex(res[2], ims[2]))) == "1.234567125e+06")
	check("cx8", real(a) == 0.1 && imag(a) == 0.2 && real(a)/2 == 0.05)
	// The basic operators.
	check("cx9", a+b == complex(res[0]+res[1], ims[0]+ims[1]))
	check("cx10", fmt.Sprint(b*b) == "(7+24i)")

	// ----- THE SIGN OF A ZERO, at all six sites a complex product has -----
	// (a+bi)(c+di) is (ac-bd) + (ad+bc)i: four products and two sums, each
	// carrying its own zero sign. The ABNF tag dialect keeps an integral value
	// as an INTEGER, so the interpreter has to put the sign back by hand. The
	// zeros are built at RUNTIME (`-pz36`), because a Go CONSTANT has no signed
	// zero and `-0.0` written out is exactly 0.
	pz36 := res[3]
	nz36 := -pz36
	check("cx12", fmt.Sprint(nz36) == "-0" && fmt.Sprint(pz36) == "0")
	// The product. Its imaginary part is (-0.0)*3 + 0.0*(-2), i.e. -0 + -0,
	// which is the ONE addition in the whole set that answers a negative zero.
	check("cx13", fmt.Sprint(complex(pz36, pz36)*complex(-2, 3)) == "(-0+0i)")
	check("cx14", fmt.Sprint(complex(nz36, pz36)*complex(-2, 3)) == "(0-0i)")
	check("cx15", fmt.Sprint(complex(nz36, nz36)*complex(2, 2)) == "(0-0i)")
	// The SUM and the DIFFERENCE: -0 + -0 is the only negative sum, and
	// -0 - +0 the only negative difference.
	check("cx16", fmt.Sprint(complex(nz36, nz36)+complex(nz36, pz36)) == "(-0+0i)")
	check("cx17", fmt.Sprint(complex(nz36, pz36)-complex(pz36, pz36)) == "(-0+0i)")
	check("cx18", fmt.Sprint(complex(pz36, pz36)-complex(pz36, nz36)) == "(0+0i)")
	// The QUOTIENT, through Smith's algorithm and its ratio.
	check("cx19", fmt.Sprint(complex(pz36, pz36)/complex(-2, 3)) == "(0-0i)")
	check("cx20", fmt.Sprint(complex(nz36, pz36)/complex(2, 3)) == "(0+0i)")
	// UNARY MINUS on each part, which `0 - x` would get wrong.
	check("cx21", fmt.Sprint(-complex(pz36, nz36)) == "(-0+0i)")
	check("cx22", fmt.Sprint(-complex(nz36, pz36)) == "(0-0i)")
	// real() and imag() hand the part back with its sign intact.
	check("cx23", fmt.Sprint(real(complex(nz36, pz36))) == "-0" && fmt.Sprint(imag(complex(pz36, nz36))) == "-0")
	// A SCALAR float64 product, which loses the sign at the same place.
	check("cx24", fmt.Sprint(pz36*-2) == "-0" && fmt.Sprint(nz36*3) == "-0" && fmt.Sprint(nz36*-3) == "0")
	check("cx25", fmt.Sprint(pz36/-2) == "-0" && fmt.Sprint(nz36+nz36) == "-0" && fmt.Sprint(nz36-pz36) == "-0")

	// ----- A Go CONSTANT HAS NO SIGNED ZERO -----
	// `-0.0` is an untyped constant expression and Go evaluates it exactly, so
	// it is 0 and not -0.0 - `complex(-0.0, 0.0)` is (0+0i) before the program
	// ever runs. A RUNTIME negation is unaffected, which cx12 pins.
	k36 := -0.0
	var v36 float64 = -0.0
	check("cx26", fmt.Sprint(k36) == "0" && fmt.Sprint(v36) == "0" && fmt.Sprint(-0.0) == "0")
	check("cx27", fmt.Sprint(complex(-0.0, 0.0)) == "(0+0i)")
	check("cx28", fmt.Sprint(cnz36) == "0" && fmt.Sprint(-0.0*1) == "0")
}

// ===== SECTION 37: nil slices, recovered results, `for range x`, panic text =====
// Every expected value was read off `go run`.
//
// WHAT EACH GROUP IS PINNING, because a passing assertion says nothing about why:
//   - the zero value of a SLICE type compares equal to nil. It is still a usable
//     empty HEADER (len/cap/range/append all work on it, as they do in Go), so
//     nil-ness is a __nil MARK on the header - exactly the trade a nil MAP
//     makes. `[]int{}`, make() and an appended header must NOT be nil, which is
//     what tells a mark from a blanket "an empty slice is nil".
//   - a function whose panic was RECOVERED answers the ZERO of its declared result
//     types, not `<nil>`; and a NAMED result starts at the zero of its own type,
//     not at the number 0.
//   - integer divide by zero is a RECOVERABLE panic in every engine (the
//     interpreter, llvm.Run and a native binary).
//   - `for range x` with no variables at all parses in both halves.
//
// The prefix of an UNRECOVERED panic in a native binary is deliberately not
// asserted here: a ratchet has to exit 0, so it cannot let a panic escape.
// tests/probe.sh with the three legs is what settles that one.
type P37 struct{ N int }
type Z37 struct {
	xs []int
	n  int
}

func f37slice(a []int, i int) []int { defer func() { recover() }(); _ = a[i]; return a }
func f37str(a []int, i int) string  { defer func() { recover() }(); _ = a[i]; return "set" }
func f37two(a []int, i int) (int, string) {
	defer func() { recover() }()
	_ = a[i]
	return 7, "set"
}
func f37struct(a []int, i int) P37 { defer func() { recover() }(); _ = a[i]; return P37{9} }
func f37u8(a []int, i int) uint8   { defer func() { recover() }(); _ = a[i]; return 200 }
func f37named(a []int, i int) (n int, s string, xs []int) {
	defer func() { recover() }()
	_ = a[i]
	n, s, xs = 1, "set", []int{4}
	return
}
func f37div(a []int, i int, j int) (out string) {
	defer func() {
		if e := recover(); e != nil {
			out = "recovered"
		}
	}()
	_ = a[i] / a[j]
	return "no panic"
}

type V37 struct {
	xs []int
	s  string
}

func (v V37) count() int             { return len(v.xs) }
func (v V37) first() int             { return v.xs[0] }
func (v V37) tag() string            { return v.s }
func (v V37) mapped() map[string]int { return map[string]int{"k": 7} }
func (v V37) nilxs() bool            { return v.xs == nil }

func s37() {
	// ----- the zero value of a slice type IS nil -----
	var z37 []int
	check("nl1", z37 == nil && nil == z37)
	check("nl2", len(z37) == 0 && cap(z37) == 0)
	check("nl3", fmt.Sprint(z37) == "[]")
	// ... and everything that is NOT the zero value is not nil.
	check("nl4", []int{} != nil)
	check("nl5", make([]int, 0) != nil)
	check("nl6", append(z37, 1) != nil && z37 == nil)
	check("nl7", []int{1, 2}[0:0] != nil)
	// The mark reaches every site that builds a zero value: a struct field, a
	// map's value type, an array element and a defined type over a slice.
	var st37 Z37
	check("nl8", st37.xs == nil && st37.n == 0)
	m37 := map[string][]int{}
	check("nl9", m37["absent"] == nil && len(m37["absent"]) == 0)
	var arr37 [2][]byte
	check("nl10", arr37[0] == nil && arr37[1] == nil)
	// A nil map still compares equal to nil, and an empty one written into does not.
	var nm37 map[string]int
	check("nl11", nm37 == nil && len(nm37) == 0)
	check("nl12", map[string]int{} != nil)

	// ----- a recovered call answers the ZERO of its result types -----
	live37 := []int{5}
	check("rz1", f37slice(live37, 9) == nil)
	check("rz2", f37slice(live37, 0) != nil)
	check("rz3", f37str(live37, 9) == "" && f37str(live37, 0) == "set")
	n37, s37s := f37two(live37, 9)
	check("rz4", n37 == 0 && s37s == "")
	check("rz5", f37struct(live37, 9) == P37{0} && f37struct(live37, 0) == P37{9})
	check("rz6", f37u8(live37, 9) == 0)
	// A NAMED result starts at the zero of its OWN type, not at the number 0.
	a37, b37, c37 := f37named(live37, 9)
	check("rz7", a37 == 0 && b37 == "" && c37 == nil)
	a38, b38, c38 := f37named(live37, 0)
	check("rz8", a38 == 1 && b38 == "set" && len(c38) == 1)

	// ----- integer divide by zero is RECOVERABLE -----
	d37 := []int{6, 3, 0}
	check("dz1", f37div(d37, 0, 1) == "no panic")
	check("dz2", f37div(d37, 0, 2) == "recovered")
	// ... and a FLOAT divided by zero is an infinity, not a panic.
	fz37 := []float64{6, 0}
	check("dz3", fz37[0]/fz37[1] > 1e307 && fz37[1]/fz37[1] != fz37[1]/fz37[1])
	// The same on a SIZED integer, whose operator is a different floor arm.
	u37 := []uint8{200, 0, 3}
	check("dz4", u37[0]/u37[2] == 66 && u37[0]%u37[2] == 2)
	// The recovered VALUE prints as go's runtime.Error text, in all three engines.
	t37 := divText37(d37, 0, 2)
	check("dz5", t37 == "runtime error: integer divide by zero")

	// ----- a VALUE receiver gets a COPY that keeps its hidden slots -----
	// goCopyVal makes the copy Go's value semantics require, and the copy has to
	// keep the __-prefixed slots (keysOf hides them), or `func (v V37) ...`
	// reading a slice field loses its __class. Only clang-check / native-full /
	// a native -exe probe can see this row at all.
	v37 := V37{xs: []int{1, 2, 3}, s: "v"}
	check("vr1", v37.count() == 3 && v37.first() == 1)
	check("vr2", v37.tag() == "v" && v37.mapped()["k"] == 7)
	// ... including the __nil mark, so a copied zero slice still compares to nil.
	check("vr3", V37{}.nilxs())

	// ----- `for range x` with no variables at all -----
	c37a := 0
	for range m37 {
		c37a++
	}
	check("fr1", c37a == 0)
	m37["k"] = []int{1}
	m37["j"] = []int{2}
	c37a = 0
	for range m37 {
		c37a++
	}
	check("fr2", c37a == 2)
	c37b := 0
	for range d37 {
		c37b++
	}
	check("fr3", c37b == 3)
	c37c := 0
	for range "héllo" {
		c37c++
	}
	check("fr4", c37c == 5)
	// The two-name form still binds, and `=` over already-declared names works too.
	sum37 := 0
	ki, kv := 0, 0
	for ki, kv = range d37 {
		sum37 += ki * kv
	}
	check("fr5", sum37 == 3 && ki == 2 && kv == 0)
}

func divText37(a []int, i int, j int) (out string) {
	defer func() {
		if e := recover(); e != nil {
			out = fmt.Sprint(e)
		}
	}()
	_ = a[i] / a[j]
	return "no panic"
}

// ===== SECTION 38: nil dereference, interfaces, multi-value calls, typed nil =====
// Every expected value was read off `go run`.
//
// WHAT EACH GROUP IS PINNING, because a passing assertion says nothing about why:
//   - a NIL POINTER DEREFERENCE is a recoverable panic in every engine. The read,
//     the WRITE, a write one level down a path and a METHOD CALL are four
//     different routes into it.
//   - a DECLARED INTERFACE is satisfied STRUCTURALLY: `x.(Sp)` holds for a value
//     that satisfies Sp, `case Sp:` fires, and the single-result `x.(Sp)` panics
//     for one that does not. The concrete-type arms pin that only an interface
//     target is checked.
//   - a MULTI-VALUE call that is the sole argument of a call EXPANDS. The results
//     carry a mark, and the array cases below are what tells the mark from "any
//     array spreads".
//   - a RESULT VALUE adopts a map or slice declared type when it is nil, so
//     `return nil` from a map result prints `map[]` and not `<nil>`. A nil POINTER
//     result must still print `<nil>`, which is what separates this from "nil
//     always becomes a container".
//   - a FUNC-TYPED STRUCT FIELD is callable, with no receiver.
//   - SLICING A NIL SLICE keeps its nilness (Go carries the nil backing pointer
//     over), while slicing an empty LITERAL does not.
//   - `append(s)` with nothing to add is s itself.
type P38 struct{ X int }
type R38 struct{ In *P38 }
type H38 struct {
	f func(int) int
	g func() string
	n int
}

type Sp38 interface {
	Speak() string
	Fly() string
}
type Bird38 struct{ N string }

func (b Bird38) Speak() string { return "tweet" }
func (b Bird38) Fly() string   { return "flap" }

type Dog38 struct{ N string }

func (d Dog38) Speak() string { return "woof" }

type Emb38 struct{ Bird38 }

func f38read(p *P38) (r int) {
	defer func() {
		if e := recover(); e != nil {
			r = -1
		}
	}()
	return p.X
}
func f38write(p *P38) (r int) {
	defer func() {
		if e := recover(); e != nil {
			r = -2
		}
	}()
	p.X = 5
	return p.X
}
func f38path(q *R38) (r int) {
	defer func() {
		if e := recover(); e != nil {
			r = -3
		}
	}()
	q.In.X = 5
	return 9
}
func (p *P38) Get() int { return p.X }
func f38mcall(p *P38) (r int) {
	defer func() {
		if e := recover(); e != nil {
			r = -4
		}
	}()
	return p.Get()
}
func f38assert(v interface{}) (s string) {
	defer func() {
		if e := recover(); e != nil {
			s = "PANIC"
		}
	}()
	return v.(Sp38).Speak()
}

func f38three(n int) (int, int, int) { return n, n + 1, n + 2 }
func f38two(n int) (int, string)     { return n, "s" }
func f38twoi(n int) (int, int)       { return n, n + 1 }
func f38add(a int, b int) int        { return a + b }
func f38arr(n int) [3]int            { return [3]int{n, n, n} }

func f38map(b bool) map[string]int {
	if b {
		return map[string]int{"a": 1}
	}
	return nil
}
func f38slice(b bool) []int {
	if b {
		return []int{1}
	}
	return nil
}
func f38ptr(b bool) *P38 { return nil }
func f38named(b bool) (m map[string]int) {
	if b {
		m = map[string]int{"q": 2}
	}
	return
}
func f38pair(b bool) (map[string]int, []int) {
	if b {
		return map[string]int{"a": 1}, []int{2}
	}
	return nil, nil
}

func s38() {
	// ----- a nil pointer dereference is a RECOVERABLE panic, four routes -----
	ps38 := []*P38{nil, &P38{7}}
	check("nd1", f38read(ps38[0]) == -1)
	check("nd2", f38read(ps38[1]) == 7)
	check("nd3", f38write(ps38[0]) == -2)
	check("nd4", f38write(ps38[1]) == 5)
	check("nd5", f38mcall(ps38[0]) == -4)
	check("nd6", f38mcall(ps38[1]) == 5)
	rs38 := []*R38{&R38{nil}, &R38{&P38{2}}}
	check("nd7", f38path(rs38[0]) == -3)
	check("nd8", f38path(rs38[1]) == 9)

	// ----- a declared interface is satisfied STRUCTURALLY -----
	vs38 := []interface{}{Bird38{"b"}, Dog38{"d"}, 3, Emb38{Bird38{"e"}}}
	_, ok38a := vs38[0].(Sp38)
	check("if1", ok38a)
	_, ok38b := vs38[1].(Sp38)
	check("if2", !ok38b)
	_, ok38c := vs38[2].(Sp38)
	check("if3", !ok38c)
	// A PROMOTED method of an embedded struct counts towards the method set.
	_, ok38d := vs38[3].(Sp38)
	check("if4", ok38d)
	check("if5", f38assert(vs38[0]) == "tweet")
	check("if6", f38assert(vs38[1]) == "PANIC")
	check("if7", f38assert(vs38[3]) == "tweet")
	// The concrete-type arms are unchanged: only an INTERFACE target is checked.
	bd38, ok38e := vs38[0].(Bird38)
	check("if8", ok38e && bd38.N == "b")
	_, ok38f := vs38[1].(Bird38)
	check("if9", !ok38f)
	sw38 := ""
	for i := 0; i < 4; i++ {
		switch vs38[i].(type) {
		case Sp38:
			sw38 += "S"
		case int:
			sw38 += "i"
		default:
			sw38 += "."
		}
	}
	check("if10", sw38 == "S.iS")

	// ----- a multi-value call that is the SOLE argument expands -----
	xs38 := []int{3, 4}
	check("mv1", fmt.Sprint(f38three(xs38[0])) == "3 4 5")
	check("mv2", fmt.Sprint(f38two(xs38[1])) == "4s")
	check("mv3", f38add(f38twoi(xs38[0])) == 7)
	check("mv4", fmt.Sprintf("%v %v %v", 1, 2, 3) == "1 2 3")
	// ... and a Go ARRAY value, which is the shape the mark had to be told from,
	// does NOT expand.
	check("mv5", fmt.Sprint(f38arr(xs38[0])) == "[3 3 3]")
	arr38 := [3]int{1, 2, 3}
	check("mv6", fmt.Sprint(arr38) == "[1 2 3]")
	// Unpacking into names still works, and so does forwarding.
	a38, b38, c38 := f38three(xs38[1])
	check("mv7", a38 == 4 && b38 == 5 && c38 == 6)
	var d38, e38, g38 = f38three(xs38[0])
	check("mv8", d38 == 3 && e38 == 4 && g38 == 5)

	// ----- a nil result adopts a MAP or SLICE declared type -----
	bs38 := []bool{false, true}
	check("tn1", fmt.Sprint(f38map(bs38[0])) == "map[]")
	check("tn2", fmt.Sprint(f38map(bs38[1])) == "map[a:1]")
	check("tn3", f38map(bs38[0]) == nil && len(f38map(bs38[0])) == 0)
	check("tn4", fmt.Sprint(f38slice(bs38[0])) == "[]")
	check("tn5", f38slice(bs38[0]) == nil)
	check("tn6", fmt.Sprint(f38named(bs38[0])) == "map[]")
	// A nil POINTER result is still <nil>: only containers adopt.
	check("tn7", fmt.Sprint(f38ptr(bs38[0])) == "<nil>")
	m38, l38 := f38pair(bs38[0])
	check("tn8", fmt.Sprint(m38) == "map[]" && fmt.Sprint(l38) == "[]")
	check("tn9", m38 == nil && l38 == nil)

	// ----- a func-typed struct field is callable -----
	h38 := H38{f: func(a int) int { return a * 2 }, g: func() string { return "hi" }, n: 1}
	check("fn1", h38.f(xs38[0]) == 6)
	check("fn2", h38.g() == "hi")
	hp38 := &h38
	check("fn3", hp38.f(xs38[1]) == 8)
	hs38 := []H38{h38}
	check("fn4", hs38[0].f(xs38[0]) == 6)
	check("fn5", h38.n == 1)

	// ----- slicing a nil slice keeps its nilness -----
	var ns38 []int
	check("sl1", ns38 == nil)
	check("sl2", ns38[:] == nil)
	check("sl3", ns38[0:0] == nil)
	check("sl4", ns38[:][:] == nil)
	check("sl5", append(ns38) == nil)
	// ... and every other producer stays non-nil, which is what tells a propagated
	// mark from "an empty slice is nil".
	es38 := []int{}
	check("sl6", es38 != nil && es38[:] != nil)
	check("sl7", make([]int, 0) != nil)
	check("sl8", append(ns38, 1) != nil)
	check("sl9", len(ns38[:]) == 0 && cap(ns38[:]) == 0)
}

// ===== SECTION 39: the error interface, deep promotion, interface{}() and nil asserts =====
// Every expected value was read off `go run`.
//
// WHAT EACH GROUP PINS:
//   - fmt CONSULTS Error() BEFORE String(), which is fmt's own handleMethods
//     order. The Stringer rows beside it pin that error did not displace String(),
//     and the both-methods row pins the PRECEDENCE rather than just presence.
//   - FIELD PROMOTION IS BREADTH FIRST and more than one level deep (`C{B{A{7}}}`);
//     the shadowing row pins that a shallower field still wins over a deeper one
//     of the same name.
//   - interface{}(x) AS A CONVERSION EXPRESSION. The float and uint8 rows pin that
//     it is the IDENTITY: the ordinary conversion path would truncate a float64 and
//     drop a sized box.
//   - A COMPOSITE LITERAL OF A DEFINED MAP TYPE, `type M map[string]int; M{...}`,
//     whose keyed element's key is an expression. The struct rows pin that
//     `P{X: 1}`, whose key is a FIELD NAME and not a variable, still works.
//   - AN ELIDED ELEMENT OF POINTER TYPE, `[]*P{nil, {7}}`; a pointer to a struct
//     IS the struct in this value model, so the star is dropped.
//   - A NIL INTERFACE ASSERTED TO AN INTERFACE PANICS, and with a different text
//     from the missing-method one - it names no method, because there is no
//     dynamic type to name. The comma-ok row pins that the two-result form still
//     answers false instead of panicking.
type E39 struct{ msg string }

func (e E39) Error() string { return e.msg }

type S39 struct{ n int }

func (s S39) String() string { return "S<" + fmt.Sprint(s.n) + ">" }

type ES39 struct{ n int }

func (e ES39) Error() string  { return "E" }
func (e ES39) String() string { return "S" }

type A39 struct{ v int }
type B39 struct{ A39 }
type C39 struct{ B39 }
type Sh39 struct {
	B39
	v int
}

type Exp39 struct{ Err error }
type Unexp39 struct{ err error }
type ExpS39 struct{ V S39 }
type UnexpS39 struct{ v S39 }
type Deep39 struct{ inner Exp39 }

type M39 map[string]int
type MS39 map[string][]int
type P39 struct{ n int }
type Pt39 struct{ X, Y int }
type I39 interface{ M() int }
type D39 struct{ k int }

func (d D39) M() int { return d.k }

func f39assert(v interface{}) (s string) {
	defer func() {
		if e := recover(); e != nil {
			s = "PANIC"
		}
	}()
	_ = v.(I39)
	return "ok"
}

func s39() {
	// -- fmt honours the error interface, and it wins over Stringer.
	msgs := []string{"missing b", "boom"}
	var err error = E39{msgs[0]}
	check("er1", fmt.Sprint(err) == "missing b")
	check("er2", fmt.Sprint(E39{msgs[1]}) == "boom")
	check("er3", fmt.Sprint("x", err, "y") == "xmissing by")
	check("er4", fmt.Sprint(S39{3}) == "S<3>")
	check("er5", fmt.Sprint(ES39{1}) == "E")
	check("er6", fmt.Sprint([]error{E39{msgs[0]}}) == "[missing b]")
	// AN UNEXPORTED FIELD IS NOT INTERFACEABLE, so fmt cannot reach its Error()
	// or String() - and transitively so for everything under it.
	check("er7", fmt.Sprint(Exp39{E39{msgs[0]}}) == "{missing b}")
	check("er8", fmt.Sprint(Unexp39{E39{msgs[0]}}) == "{{missing b}}")
	check("er9", fmt.Sprint(ExpS39{S39{3}}) == "{S<3>}")
	check("er10", fmt.Sprint(UnexpS39{S39{3}}) == "{{3}}")
	check("er11", fmt.Sprint(Deep39{Exp39{E39{msgs[0]}}}) == "{{{missing b}}}")
	check("er12", fmt.Sprint(map[string]error{"k": E39{msgs[0]}}) == "map[k:missing b]")

	// -- promoted fields, two levels down and breadth first.
	ns := []int{7}
	c := C39{B39{A39{ns[0]}}}
	check("pf1", c.v == 7)
	check("pf2", c.B39.A39.v == 7)
	sh := Sh39{B39{A39{1}}, 2}
	check("pf3", sh.v == 2)
	check("pf4", sh.B39.v == 1)

	// -- interface{}(x) as a conversion EXPRESSION is the identity.
	iv := interface{}(ns[0])
	check("ic1", iv == 7)
	fs := []float64{2.5}
	check("ic2", fmt.Sprint(interface{}(fs[0])) == "2.5")
	bs := []uint8{200}
	check("ic3", fmt.Sprint(interface{}(bs[0])) == "200")
	check("ic4", fmt.Sprint((interface{})(ns[0])) == "7")

	// -- a composite literal of a DEFINED map type, keyed by an expression.
	k := "b"
	m := M39{"a": ns[0], k: 9}
	check("dm1", m["a"] == 7)
	check("dm2", m["b"] == 9)
	check("dm3", len(m) == 2)
	check("dm4", m["zz"] == 0)
	check("dm5", len(M39{}) == 0)
	ms := MS39{"a": {1, 2}}
	check("dm6", ms["a"][1] == 2)
	// The keyed STRUCT literal, whose key is a field name and not a variable.
	p := Pt39{Y: 4, X: ns[0]}
	check("dm7", p.X == 7 && p.Y == 4)
	check("dm8", fmt.Sprint(Pt39{1, 2}) == "{1 2}")

	// -- an elided element of POINTER type.
	ps := []*P39{nil, {ns[0]}}
	check("ep1", ps[0] == nil)
	check("ep2", ps[1].n == 7)
	qs := []P39{{ns[0]}, {2}}
	check("ep3", qs[0].n == 7 && qs[1].n == 2)

	// -- x.(I) on a NIL interface panics; the comma-ok form does not.
	vals := []interface{}{nil, D39{ns[0]}}
	check("na1", f39assert(vals[0]) == "PANIC")
	check("na2", f39assert(vals[1]) == "ok")
	_, ok0 := vals[0].(I39)
	_, ok1 := vals[1].(I39)
	check("na3", !ok0)
	check("na4", ok1)
}

// ===== SECTION 40: a named constant as an array length, and multi-name const specs =====
//   - `const size = 4; var a [size]int` is a four element array: len(a) is 4 and
//     a[2] is in range. The literal-length rows beside it pin that nothing about
//     `[4]int` changed, and the map row pins that `[string]` of a map type is not
//     read as a length at all.
//   - `const a, b = 0.1, 0.2`: an ungrouped const spec binds a name LIST and an
//     expression LIST, like the grouped one.
const sz40 = 4
const szt40 int = 3

const c40a, c40b = 0.1, 0.2
const c40c, c40d int = 6, 7

type Board40 struct{ cells [sz40]int }

func s40() {
	var a [sz40]int
	a[2] = 9
	check("al1", len(a) == 4)
	check("al2", a[2] == 9 && a[0] == 0)
	var b [szt40]int
	check("al3", len(b) == 3)
	var lit [4]int
	check("al4", len(lit) == 4)
	const local = 2
	var d [local]string
	check("al5", len(d) == 2)
	e := [sz40]int{1, 2}
	check("al6", len(e) == 4 && e[3] == 0)
	var bd Board40
	check("al7", len(bd.cells) == 4)
	mm := map[string]int{"k": 1}
	check("al8", mm["k"] == 1)
	var nested [sz40][2]int
	check("al9", len(nested) == 4 && len(nested[0]) == 2)
	var sl []int
	check("al10", len(sl) == 0)

	check("cm1", c40a == 0.1 && c40b == 0.2)
	check("cm2", c40c == 6 && c40d == 7)
	const p40, q40 = "x", 2
	check("cm3", p40 == "x" && q40 == 2)
	xs := []float64{10}
	check("cm4", c40a*xs[0] == 1)
}

// ===== SECTION 41: iota as an array length, and a struct-valued map key =====
// Both settled by `go run`.
//
//   - An IOTA-GROUP constant as an array length. A spec that mentions iota cannot be
//     bound to one constant, and a spec with no '=' has no expression to bind, so
//     the fold is kept ITSELF, as a function of iota, beside the constant; then
//     `const (R = iota; G; B); var a [B]int` has len 2.
//   - A STRUCT- or ARRAY-VALUED map key. Both are Go VALUE types and `==` on them is
//     element-by-element, so `m[Pair{1, 2}]` must find an entry stored under a
//     different Pair holding the same fields - for a read, a comma-ok and a delete.
//     The elided composite key `map[Pair]string{{1, 2}: "p"}` is a key form too.
const (
	r41 = iota
	g41
	b41
	n41
)

const (
	x41 = iota * 3
	y41
	z41
)

const (
	k41a = 1 << iota
	k41b
	k41c
	k41d
)

const (
	t41a int = iota + 2
	t41b
	t41c
)

type pair41 struct {
	A int
	B int
}

type named41 struct {
	P pair41
	S string
}

func s41() {
	// The values themselves.
	check("io1", r41 == 0 && g41 == 1 && b41 == 2 && n41 == 3)
	check("io2", x41 == 0 && y41 == 3 && z41 == 6)
	check("io3", k41a == 1 && k41b == 2 && k41c == 4 && k41d == 8)
	// ... and their compile-time values as array lengths.
	var a [b41]int
	check("io4", len(a) == 2)
	var bb [n41]int
	check("io5", len(bb) == 3)
	var cc [z41]int
	check("io6", len(cc) == 6)
	var dd [k41d]int
	check("io7", len(dd) == 8)
	// A TYPED iota constant reaches the array-length side table too.
	var ee [t41c]int
	check("io8", len(ee) == 4 && t41a == 2)
	// Two dimensions, and a literal whose length comes from the group.
	var ff [3][b41]int
	check("io9", len(ff) == 3 && len(ff[0]) == 2)
	// A constant EXPRESSION in the brackets, beyond a bare NAME.
	var gg2 [b41 + 1]int
	check("io12", len(gg2) == 3)
	var hh2 [(n41 * 2) - 1]int
	check("io13", len(hh2) == 5)
	var ii2 [1 << b41]int
	check("io14", len(ii2) == 4)
	var jj2 [t41c * 2]int
	check("io15", len(jj2) == 8)
	var kk2 [2][b41 + 2]int
	check("io16", len(kk2) == 2 && len(kk2[0]) == 4)
	// ... and the shapes the substitution must NOT touch: a map key type, a slice, and
	// a length whose name is a plain variable rather than a constant.
	var mm2 map[string][]int
	check("io17", len(mm2) == 0)
	nv := 3
	check("io18", len(make([]int, nv)) == 3)
	gg := [b41]string{"p", "q"}
	check("io10", len(gg) == 2 && gg[1] == "q")
	hh := make([]int, n41)
	check("io11", len(hh) == 3)

	// A struct key: every operand read out of a slice, so nothing folds.
	ks := []pair41{{1, 2}, {3, 4}, {1, 2}}
	m := map[pair41]string{}
	m[ks[0]] = "a"
	m[ks[1]] = "b"
	check("mk1", len(m) == 2)
	check("mk2", m[ks[2]] == "a")
	v, ok := m[ks[2]]
	check("mk3", v == "a" && ok)
	_, miss := m[pair41{9, 9}]
	check("mk4", !miss)
	delete(m, ks[2])
	check("mk5", len(m) == 1 && m[ks[0]] == "")
	// The elided-key literal form.
	m2 := map[pair41]string{{1, 2}: "p", {5, 6}: "q"}
	check("mk6", len(m2) == 2 && m2[ks[0]] == "p")
	// A NESTED struct key, compared field by field all the way down.
	n1 := named41{pair41{1, 2}, "x"}
	n2 := named41{pair41{1, 2}, "x"}
	n3 := named41{pair41{1, 3}, "x"}
	m3 := map[named41]int{}
	m3[n1] = 7
	check("mk7", m3[n2] == 7 && len(m3) == 1)
	check("mk8", m3[n3] == 0)
	// An ARRAY key is a value type as well.
	as := [][2]int{{1, 2}, {1, 2}, {2, 1}}
	m4 := map[[2]int]string{}
	m4[as[0]] = "arr"
	check("mk9", m4[as[1]] == "arr" && m4[as[2]] == "")
	// A string key compares as before.
	m5 := map[string]int{"a": 1}
	check("mk10", m5["a"] == 1 && m5["b"] == 0)
	// A map value keyed by a struct, ranged over: insertion order is the model's.
	tot := 0
	for _, val := range m3 {
		tot += val
	}
	check("mk11", tot == 7)
}

// ===== SECTION 42: an elided key on a NAMED map type, and keyed nested elements =====
// Both settled by `go run`.
//
//   - `type Named map[Pair]string; Named{{1, 2}: "p"}`: the NAMED spelling goes
//     through StructLit rather than MapLit, so its key production takes a brace group
//     too, built against the map's declared key type. Nested defined types come with
//     it: []Named{{{1, 2}: "a"}}.
//   - A KEYED nested struct element - map[string]P{"x": {N: 1}} - needs the key's
//     TEXT at the nested element's key production, so the field name is there to be
//     used.
type pair42 struct {
	A int
	B int
}

type named42 map[pair42]string

type tally42 map[string]int

type nums42 []int

type pt42 struct {
	N int
	X int
}

func s42() {
	ks := []pair42{{1, 2}, {3, 4}}
	// The named-map form, with an elided key and a written-out one side by side.
	m := named42{{1, 2}: "p", pair42{3, 4}: "q"}
	check("nm1", len(m) == 2 && m[ks[0]] == "p" && m[ks[1]] == "q")
	// A defined map over ordinary key types is unaffected.
	t := tally42{"k": 8}
	check("nm2", t["k"] == 8 && len(t) == 1)
	// A defined SLICE type still builds from the same production.
	n := nums42{4, 5, 6}
	check("nm3", len(n) == 3 && n[2] == 6)
	// The zero value of a defined map type reads like an empty one.
	var z named42
	check("nm4", len(z) == 0 && z[ks[0]] == "")
	// A nested brace group whose element type is a DEFINED map.
	ms := []named42{{{1, 2}: "a"}, {{1, 2}: "b"}}
	check("nm5", len(ms) == 2 && ms[0][ks[0]] == "a" && ms[1][ks[0]] == "b")
	mm := map[string]named42{"x": {{1, 2}: "z"}}
	check("nm6", mm["x"][ks[0]] == "z")
	// The literal-type spelling of the same thing, which SECTION 41 introduced.
	lit := map[pair42]string{{1, 2}: "L"}
	check("nm7", lit[ks[0]] == "L")

	// A KEYED nested struct element, in a map value, a slice element and an array.
	km := map[string]pt42{"x": {N: 1, X: 2}, "y": {3, 4}}
	check("ke1", km["x"].N == 1 && km["x"].X == 2 && km["y"].N == 3)
	ks2 := []pt42{{N: 5}, {6, 7}, {X: 8}}
	check("ke2", ks2[0].N == 5 && ks2[0].X == 0 && ks2[1].X == 7 && ks2[2].N == 0 && ks2[2].X == 8)
	var ka [2]pt42
	check("ke3", len(ka) == 2 && ka[0].N == 0)
	// ... and an INDEXED element of a slice or array, which is a different use of the
	// same key production and must keep meaning an index rather than a field name.
	idx := []int{0: 5, 3: 9}
	check("ke4", len(idx) == 4 && idx[3] == 9 && idx[0] == 5)
	arr := [5]int{4: 1}
	check("ke5", len(arr) == 5 && arr[4] == 1)
}

// ===== SECTION 43: the value recover() hands back for a run-time panic =====
// Every expected string was read off `go run`. A runtime.Error prints
// "runtime error: " and go's bounds wording; a store into a nil map and a failed
// interface conversion carry no such prefix.
func rec43(f func()) (out string) {
	defer func() { out = fmt.Sprint(recover()) }()
	f()
	return "no panic"
}

func s43() {
	ns := []int{0, 1, 2, 3, 4, 5, -1, 7}
	xs := []int{10, 20, 30}
	var arr [3]int
	var nilsl []int
	var nilm map[int]int
	str := "ab"
	type P43 struct{ X int }
	var np *P43
	check("rp1", rec43(func() { _ = ns[3] / ns[0] }) == "runtime error: integer divide by zero")
	check("rp2", rec43(func() { _ = ns[3] % ns[0] }) == "runtime error: integer divide by zero")
	check("rp3", rec43(func() { _ = xs[ns[3]] }) == "runtime error: index out of range [3] with length 3")
	check("rp4", rec43(func() { xs[ns[6]] = 1 }) == "runtime error: index out of range [-1]")
	check("rp5", rec43(func() { _ = arr[ns[7]] }) == "runtime error: index out of range [7] with length 3")
	check("rp6", rec43(func() { arr[ns[3]] = 2 }) == "runtime error: index out of range [3] with length 3")
	check("rp7", rec43(func() { _ = str[ns[2]] }) == "runtime error: index out of range [2] with length 2")
	sub := xs[0:1]
	check("rp8", rec43(func() { _ = sub[ns[2]] }) == "runtime error: index out of range [2] with length 1")
	check("rp9", rec43(func() { nilm[ns[1]] = 1 }) == "assignment to entry in nil map")
	check("rp10", rec43(func() { _ = np.X }) == "runtime error: invalid memory address or nil pointer dereference")
	check("rp11", rec43(func() { _ = xs[ns[5]:] }) == "runtime error: slice bounds out of range [5:3]")
	check("rp12", rec43(func() { _ = xs[:ns[5]] }) == "runtime error: slice bounds out of range [:5] with capacity 3")
	check("rp13", rec43(func() { _ = xs[ns[6]:] }) == "runtime error: slice bounds out of range [-1:]")
	check("rp14", rec43(func() { _ = xs[:ns[6]] }) == "runtime error: slice bounds out of range [:-1]")
	check("rp15", rec43(func() { _ = xs[ns[1]:ns[2]:ns[5]] }) == "runtime error: slice bounds out of range [::5] with capacity 3")
	check("rp16", rec43(func() { _ = xs[ns[1]:ns[3]:ns[2]] }) == "runtime error: slice bounds out of range [:3:2]")
	check("rp17", rec43(func() { _ = xs[ns[2]:ns[1]:ns[3]] }) == "runtime error: slice bounds out of range [2:1:]")
	check("rp18", rec43(func() { _ = arr[:ns[5]] }) == "runtime error: slice bounds out of range [:5] with length 3")
	check("rp19", rec43(func() { _ = nilsl[ns[1]:ns[2]] }) == "runtime error: slice bounds out of range [:2] with capacity 0")
	check("rp20", rec43(func() { _ = str[ns[5]:] }) == "runtime error: slice bounds out of range [5:2]")
	check("rp21", rec43(func() { _ = str[:ns[5]] }) == "runtime error: slice bounds out of range [:5] with length 2")
	// No panic: recover() is nil, and the deferred call overwrites the result.
	check("rp22", rec43(func() { _ = xs[ns[1]:ns[2]] }) == "<nil>")
}

// ===== SECTION 44: the error interface and type assertions =====
// Every expected value was read off `go run`. `error` is the predeclared interface
// {Error() string}; a run-time panic recovers as an error, a panic(string) as a
// string; a type assertion to a concrete type that does not hold panics.
type E44 struct{ m string }

func (e E44) Error() string { return "e44: " + e.m }

type P44 struct{ x int }
type Q44 struct{ y int }
type S44 interface{ M44() }

// rec44 runs f and describes what recover() handed back.
func rec44(f func()) (out string) {
	defer func() {
		r := recover()
		if e, ok := r.(error); ok {
			out = "error " + e.Error()
			return
		}
		if s, ok := r.(string); ok {
			out = "string " + s
			return
		}
		if r != nil {
			out = "other"
		}
	}()
	f()
	return "no panic"
}

func kind44(v interface{}) string {
	switch t := v.(type) {
	case error:
		return "error " + t.Error()
	case float64:
		return "float64"
	case uint8:
		return "uint8"
	case string:
		return "string"
	case int:
		return "int"
	}
	return "other"
}

func s44() {
	var v interface{} = E44{"x"}
	_, ok := v.(error)
	check("er1", ok)
	var p interface{} = P44{1}
	_, ok = p.(error)
	check("er2", !ok)
	var e error = E44{"z"}
	check("er3", e.Error() == "e44: z" && fmt.Sprint(e) == "e44: z")
	ns := []int{0, 3}
	xs := []int{1, 2}
	var nilm map[string]int
	check("er4", rec44(func() { _ = ns[1] / ns[0] }) == "error runtime error: integer divide by zero")
	check("er5", rec44(func() { _ = xs[ns[1]] }) == "error runtime error: index out of range [3] with length 2")
	check("er6", rec44(func() { nilm["a"] = 1 }) == "error assignment to entry in nil map")
	check("er7", rec44(func() { panic("boom") }) == "string boom")
	check("er8", rec44(func() {}) == "no panic")

	var f64 float64 = 2.5
	var u8 uint8 = 7
	check("ts1", kind44(E44{"y"}) == "error e44: y" && kind44("s") == "string" && kind44(3) == "int")
	check("ts2", kind44(f64) == "float64" && kind44(u8) == "uint8" && kind44(P44{2}) == "other")

	var a interface{} = "s"
	n, ok := a.(int)
	check("ta1", n == 0 && !ok)
	check("ta2", rec44(func() { _ = a.(int) }) == "error interface conversion: interface {} is string, not int")
	var fa interface{} = f64
	check("ta3", rec44(func() { _ = fa.(int) }) == "error interface conversion: interface {} is float64, not int")
	check("ta4", fa.(float64) == 2.5)
	var ua interface{} = u8
	b, ok := ua.(byte)
	check("ta5", ok && b == 7 && rec44(func() { _ = ua.(string) }) == "error interface conversion: interface {} is uint8, not string")
	var na interface{}
	check("ta6", rec44(func() { _ = na.(bool) }) == "error interface conversion: interface {} is nil, not bool")
	var qa interface{} = Q44{1}
	check("ta7", rec44(func() { _ = qa.(P44) }) == "error interface conversion: interface {} is main.Q44, not main.P44" && qa.(Q44).y == 1)
	// A declared type is named with its package, and a composite as reflect spells it.
	check("ta8", rec44(func() { _ = a.(P44) }) == "error interface conversion: interface {} is string, not main.P44")
	check("ta9", rec44(func() { _ = qa.(S44) }) == "error interface conversion: main.Q44 is not main.S44: missing method M44")
	check("ta10", rec44(func() { _ = na.(S44) }) == "error interface conversion: interface is nil, not main.S44")
	var sa interface{} = []string{"x"}
	var ma interface{} = map[string]P44{}
	check("ta11", rec44(func() { _ = sa.(int) }) == "error interface conversion: interface {} is []string, not int")
	check("ta12", rec44(func() { _ = ma.(byte) }) == "error interface conversion: interface {} is map[string]main.P44, not uint8")
}

// ===== SECTION 45: fmt's verbs, flags, width and precision =====
// Every expected string was read off `go run`. The operands come out of slices so
// nothing is folded at compile time.
type P45 struct {
	A int
	b string
}

type E45 struct{ m string }

func (e E45) Error() string { return "E:" + e.m }

func s45() {
	fs := []float64{3.14159, 2.675, 1234.5678, 1e21, 0.000123, 2.5, -3.14159, 0.1, 1.0}
	is := []int{255, -255, 7, 0, 65, 5}
	check("fv1", fmt.Sprintf("%f|%.2f|%6.2f|%-8.3e|", fs[0], fs[1], fs[0], fs[2]) == "3.141590|2.67|  3.14|1.235e+03|")
	check("fv2", fmt.Sprintf("%g|%g|%.3g|%e|%E|%G", fs[3], fs[2], fs[2], fs[4], fs[2], fs[4]) == "1e+21|1234.5678|1.23e+03|1.230000e-04|1.234568E+03|0.000123")
	check("fv3", fmt.Sprintf("%.0f|%.0e|%+08.2f|%08.2f|% .2f", fs[5], fs[5], fs[6], fs[6], fs[0]) == "2|2e+00|-0003.14|-0003.14| 3.14")
	check("fv4", fmt.Sprintf("%#g|%#.0f|%x|%.2x|%X|%b", fs[8], fs[8], fs[7], fs[7], fs[6], fs[8]) == "1.00000|1.|0x1.999999999999ap-04|0x1.9ap-04|-0X1.921F9F01B866EP+01|4503599627370496p-52")
	check("fv5", fmt.Sprintf("%x|%X|%#x|%o|%#o|%O|%b|%#b", is[0], is[0], is[1], is[2]+1, is[2]+1, is[2]+1, is[5], is[5]) == "ff|FF|-0xff|10|010|0o10|101|0b101")
	check("fv6", fmt.Sprintf("%5d|%-5d|%05d|%+d|% d|%.3d|%.0d|%#08x", is[0], is[2], is[1], is[2], is[2], is[2], is[3], is[0]) == "  255|7    |-0255|+7| 7|007||0x000000ff")
	check("fv7", fmt.Sprintf("%c|%q|%U|%#U|%x|% x|%#x", is[4], is[4], is[4], is[4], "hi", "hi", "hi") == "A|'A'|U+0041|U+0041 'A'|6869|68 69|0x6869")
	check("fv8", fmt.Sprintf("%.2s|%5.1s|%-4s|%05s|%q|%#q|%+q", "hello", "hello", "ab", "ab", "a\"b", "ab", "é") == "he|    h|ab  |000ab|\"a\\\"b\"|`ab`|\"\\u00e9\"")
	check("fv9", fmt.Sprintf("%*d|%-*d|%.*f|%5t|%v|%+v", 5, is[5], 4, is[2], 2, fs[0], true, fs[0], is[5]) == "    5|7   |3.14| true|3.14159|5")
	check("fv10", fmt.Sprintf("%5.1f|%x|%s", []float64{1.25, -2}, []byte("hi"), []byte("hi")) == "[  1.2  -2.0]|6869|hi")
	check("fv11", fmt.Sprintf("%v|%.1f|%+.2e", complex(fs[8], -2), complex(1.25, 2.35), complex(fs[8], 0)) == "(1-2i)|(1.2+2.4i)|(+1.00e+00+0.00e+00i)")
	check("fv12", fmt.Sprintf("%d|%x", []byte("hi"), E45{"a"}) == "[104 105]|453a61")
	check("fv13", fmt.Sprintf("%!|%z|%d %d|%", is[2], is[2]) == "%!!(int=7)|%!z(int=7)|%!d(MISSING) %!d(MISSING)|%!(NOVERB)")
	// An integer constant past 2^53 in a float64 slot rounds to the nearest double.
	big := []float64{9007199254740993}
	check("fv14", fmt.Sprintf("%v|%.1f", big[0], big[0]) == "9.007199254740992e+15|9007199254740992.0")

	// Composites are walked element by element, so a verb that does not suit an
	// element is noted on that element.
	check("bv1", fmt.Sprintf("%s", []int{1, 2}) == "[%!s(int=1) %!s(int=2)]")
	check("bv2", fmt.Sprintf("%d", P45{1, "x"}) == "{1 %!d(string=x)}")
	check("bv3", fmt.Sprintf("%s|%d", map[string]int{"b": 2, "a": 1}, []string{"a"}) == "map[a:%!s(int=1) b:%!s(int=2)]|[%!d(string=a)]")
	check("bv4", fmt.Sprintf("%+v|%d|%5t|%-5s|", P45{1, "x"}, E45{"a"}, 1, 3) == "{A:1 b:x}|{%!d(string=a)}|%!t(int=    1)|%!s(int=3    )|")
	check("bv5", fmt.Sprintf("%d|%s|%t", fs[7], fs[7], nil) == "%!d(float64=0.1)|%!s(float64=0.1)|%!t(<nil>)")

	// %T is reflect's spelling of the dynamic type.
	ch := make(chan string)
	check("tv1", fmt.Sprintf("%T|%T|%T|%T|%T|%T", is[0], fs[0], "s", true, complex(fs[8], 1), nil) == "int|float64|string|bool|complex128|<nil>")
	check("tv2", fmt.Sprintf("%T|%T|%T|%T|%-8T|%.3T", P45{}, []P45{}, map[string][]byte{}, [2]bool{}, is[0], "s") == "main.P45|[]main.P45|map[string][]uint8|[2]bool|int     |str")
	check("tv3", fmt.Sprintf("%T|%T|%T", ch, struct{ N int }{1}, E45{}) == "chan string|struct { N int }|main.E45")
	// Operands no directive used are listed with their types.
	check("ex1", fmt.Sprintf("%d", is[0], "x", P45{1, "y"}, nil, []int{2}) == "255%!(EXTRA string=x, main.P45={1 y}, <nil>, []int=[2])")
	check("ex2", fmt.Sprintf("x", fs[5]) == "x%!(EXTRA float64=2.5)" && fmt.Sprintf("%d %", is[2], is[3]) == "7 %!(NOVERB)%!(EXTRA int=0)")
	// %p is an address for a reference and a bad verb for anything else; a function
	// and a channel print theirs under %v too.
	xs := []int{1, 2}
	ys := xs
	var nm map[string]int
	f := func() {}
	check("pv1", fmt.Sprintf("%p", xs) == fmt.Sprintf("%p", ys) && fmt.Sprintf("%p", xs) != fmt.Sprintf("%p", xs[1:]) && fmt.Sprintf("%p", xs)[:2] == "0x")
	check("pv2", fmt.Sprintf("%p|%#p|%p|%p", nm, nm, is[2], [2]int{1, 2}) == "0x0|0|%!p(int=7)|%!p([2]int=[1 2])")
	check("pv3", fmt.Sprint(f) == fmt.Sprintf("%p", f) && fmt.Sprintf("%v", ch) == fmt.Sprintf("%p", ch) && fmt.Sprint(f)[:2] == "0x")
	// %#v writes a Go literal; a nil in a typed slot is spelled with the slot's type.
	type W45 struct {
		P   P45
		S   []string
		M   map[string]int
		Err error
		Any interface{}
		U   uint8
	}
	check("gv1", fmt.Sprintf("%#v|%#v|%#v|%#v|%#v", is[0], "a\tb", fs[5], true, nil) == "255|\"a\\tb\"|2.5|true|<nil>")
	check("gv2", fmt.Sprintf("%#v", W45{P: P45{1, "x"}, M: map[string]int{"b": 2, "a": 1}, U: 7}) == "main.W45{P:main.P45{A:1, b:\"x\"}, S:[]string(nil), M:map[string]int{\"a\":1, \"b\":2}, Err:error(nil), Any:interface {}(nil), U:0x7}")
	check("gv3", fmt.Sprintf("%#v|%#v|%#v", []interface{}{1, "a", nil}, [2]bool{true}, []byte("hi")) == "[]interface {}{1, \"a\", interface {}(nil)}|[2]bool{true, false}|[]byte{0x68, 0x69}")
	check("gv4", fmt.Sprintf("%#v|%#v|%#6v", E45{"q"}, struct{ N int }{3}, is[2]) == "main.E45{m:\"q\"}|struct { N int }{N:3}|     7")
	// %q, %+q and %#U escape what strconv.IsPrint refuses.
	us := []int{0x378, 0x2028, 0x4e16}
	check("qv1", fmt.Sprintf("%q|%+q|%q", "a\u2028b\u0378", "\u200b\u00e9", rune(us[0])) == "\"a\\u2028b\\u0378\"|\"\\u200b\\u00e9\"|'\\u0378'")
	check("qv2", fmt.Sprintf("%#U|%#U|%#U", us[0], us[1], us[2]) == "U+0378|U+2028|U+4E16 '\u4e16'")
}

// ===== SECTION 46: panic(nil), any, fmt.Errorf and the errors package =====
// Every expected value was read off `go run` (go 1.27). Since go 1.21 panic(nil)
// recovers as a *runtime.PanicNilError - in a module that declares go 1.21 or
// later; this repository's go.mod says 1.16, so check the file here with
// GODEBUG=panicnil=0. errors.New and fmt.Errorf answer
// POINTERS, so two errors with the same text are different; %w wraps.
var base46 = errors.New("base")
var twin46 = errors.New("base")

type W46 struct{ k int }

func (w W46) Error() string        { return fmt.Sprint("w", w.k) }
func (w W46) Is(target error) bool { return target == twin46 }

type M46 struct{ errs []error }

func (m M46) Error() string   { return "m46" }
func (m M46) Unwrap() []error { return m.errs }

func rec46(f func()) (r any) {
	defer func() { r = recover() }()
	f()
	return "none"
}

func s46() {
	r := rec46(func() { panic(nil) })
	e, ok := r.(error)
	check("pn1", ok && e.Error() == "runtime error: panic called with nil argument")
	check("pn2", fmt.Sprint(rec46(func() { panic(any(nil)) })) == "runtime error: panic called with nil argument")
	check("pn3", fmt.Sprint(rec46(func() { panic(error(nil)) })) == "runtime error: panic called with nil argument")
	var x any = 5
	check("an1", any(nil) == nil && x == any(5) && fmt.Sprint([]any{1, "s", nil}) == "[1 s <nil>]")
	ns := []int{3, -2}
	plain := fmt.Errorf("plain %d %s", ns[0], "x")
	check("ef1", plain.Error() == "plain 3 x" && errors.Unwrap(plain) == nil)
	w := fmt.Errorf("find %d: %w", ns[1], base46)
	check("ef2", fmt.Sprint(w) == "find -2: base" && errors.Unwrap(w) == base46)
	check("ef3", errors.Is(w, base46) && !errors.Is(w, twin46) && base46 != twin46)
	w2 := fmt.Errorf("outer: %w", w)
	check("ef4", w2.Error() == "outer: find -2: base" && errors.Is(w2, base46) && errors.Unwrap(errors.Unwrap(w2)) == base46)
	check("ef5", fmt.Errorf("n: %w", nil).Error() == "n: %!w(<nil>)" && fmt.Errorf("i: %w", ns[0]).Error() == "i: %!w(int=3)")
	two := fmt.Errorf("%w and %w", base46, twin46)
	check("ef6", two.Error() == "base and base" && errors.Is(two, base46) && errors.Is(two, twin46) && errors.Unwrap(two) == nil)
	check("ef7", errors.Is(W46{1}, twin46) && !errors.Is(W46{1}, base46) && errors.Is(fmt.Errorf("x %w", W46{2}), twin46))
	m := M46{[]error{plain, base46}}
	check("ef8", errors.Is(m, base46) && !errors.Is(m, twin46) && errors.Unwrap(m) == nil)
	check("ef9", errors.New("a") != errors.New("a") && !errors.Is(nil, base46) && errors.Is(nil, nil))
	check("ef10", fmt.Sprintf("%q|%v", fmt.Errorf("q"), fmt.Errorf("%+w!", base46)) == "\"q\"|base!")
}

// ===== SECTION 47: errors.As and errors.Join, pointer errors, a spread method call =====
// Every expected value was read off `go run` (go 1.27). errors.As stores the first
// error in the tree its target's type accepts; the type of a nil pointer target is
// its declaration's. An error type with a pointer-receiver Error method is only an
// error as a pointer, so errors.Is compares those by identity.
type NF47 struct{ Name string }

func (e *NF47) Error() string { return "not found: " + e.Name }

type T47 struct{ Code int }

func (t T47) Error() string   { return fmt.Sprint("t", t.Code) }
func (t T47) Temporary() bool { return true }

type temp47 interface{ Temporary() bool }

var nfs47 = []*NF47{{"a"}, {"b"}}

func find47(i int) error { return nfs47[i] }

type acc47 struct{ n int }

func (a *acc47) add(xs ...int) int {
	for _, x := range xs {
		a.n += x
	}
	return a.n
}

func s47() {
	ks := []int{0, 1, 2}
	errs := []error{fmt.Errorf("l: %w", find47(ks[0])), fmt.Errorf("o: %w", fmt.Errorf("i: %w", T47{7})), errors.New("p")}
	var nf *NF47
	var tp T47
	var tm temp47
	var ee error
	check("as1", errors.As(errs[0], &nf) && nf.Name == "a" && !errors.As(errs[1], &nf) && nf.Name == "a")
	check("as2", errors.As(errs[1], &tp) && tp.Code == 7 && errors.As(errs[1], &tm) && tm.Temporary() && !errors.As(errs[2], &tm))
	check("as3", errors.As(errs[2], &ee) && ee.Error() == "p")
	j := errors.Join(errs[2], nil, nfs47[1])
	var nf2 *NF47
	check("jn1", j.Error() == "p\nnot found: b" && errors.As(j, &nf2) && nf2.Name == "b" && errors.Is(j, errs[2]))
	check("jn2", errors.Join(nil, nil) == nil && errors.Unwrap(j) == nil)
	check("ei1", errors.Is(errs[0], find47(ks[0])) && !errors.Is(errs[0], &NF47{"a"}) && !errors.Is(errs[0], find47(ks[1])))
	ys := []any{1, "a", 2.5}
	a := &acc47{}
	ns := []int{4, 5}
	check("sp1", fmt.Sprint(ys...) == fmt.Sprint(1, "a", 2.5) && a.add(ns...) == 9 && a.add() == 9)
}

// ===== SECTION 48: a deferred call that panics =====
// Expected values read off `go run` (go 1.27). A deferred call that panics does not
// stop the frame's other deferred calls: its panic replaces the one in flight and
// the remaining calls run for it, so a later recover() takes the NEWER value.
var log48 = ""

func f48a() (got any) {
	defer func() { got = recover() }()
	defer func() { panic("B") }()
	defer func() { log48 += "c" }()
	panic("A")
}

func f48b() (got any) {
	defer func() { got = recover() }()
	defer func() { panic("B") }()
	log48 += "b"
	return "unused"
}

func f48c() (got any) {
	defer func() { got = recover() }()
	defer func() { panic("C") }()
	defer func() { recover(); panic("B") }()
	panic("A")
}

func f48d() (out string) {
	defer func() { out += fmt.Sprint(recover()) }()
	func() {
		defer func() { out += "i" }()
		defer func() { panic("B") }()
		panic("A")
	}()
	return "x"
}

func s48() {
	check("dp1", f48a() == "B" && log48 == "c")
	check("dp2", f48b() == "B" && log48 == "cb")
	check("dp3", f48c() == "C")
	check("dp4", f48d() == "iB")
}

// ===== SECTION 49: a struct is a value, a pointer is not =====
// Expected values read off `go run` (go 1.27). Assignment, a short declaration, a
// dereference, an argument, a result, a composite-literal element, a range variable,
// a channel send and append all COPY a struct (and an array of them); a pointer stays
// the same object, and == on two pointers is identity.
type T49 struct{ a int }
type U49 struct {
	t T49
	n []int
	p *T49
}

func id49(t T49) T49   { return t }
func mod49(t T49)      { t.a = 50 }
func ret49(t *T49) T49 { return *t }

func s49() {
	a := T49{1}
	p := &a
	b := a
	b.a = 2
	c := *p
	c.a = 9
	d := id49(a)
	d.a = 7
	mod49(a)
	e := ret49(p)
	e.a = 8
	check("vs1", a.a == 1 && b.a == 2 && c.a == 9 && d.a == 7 && e.a == 8)
	p.a = 10
	check("vs2", a.a == 10 && b.a == 2 && c.a == 9)
	x := &T49{1}
	y := &T49{1}
	check("vs3", x != y && *x == *y && x == x && p == &a)
	u := U49{t: T49{3}, n: []int{1}, p: x}
	v := u
	v.t.a = 4
	v.n[0] = 9
	v.p.a = 5
	check("vs4", u.t.a == 3 && u.n[0] == 9 && u.p.a == 5 && v.t.a == 4)
	arr := []T49{{1}, {2}}
	f := arr[0]
	f.a = 99
	m := map[string]T49{"k": {1}}
	g := m["k"]
	g.a = 5
	check("vs5", arr[0].a == 1 && f.a == 99 && m["k"].a == 1 && g.a == 5)
	var h T49
	h = a
	h.a = 77
	ts := [2]T49{{1}, {2}}
	ts2 := ts
	ts2[0].a = 6
	check("vs6", a.a == 10 && h.a == 77 && ts[0].a == 1 && ts2[0].a == 6)
	for _, t := range arr {
		t.a = 1000
	}
	var ip interface{} = a
	a.a = 123
	ch := make(chan T49, 1)
	ch <- a
	a.a = 124
	got := <-ch
	check("vs7", arr[0].a == 1 && ip.(T49).a == 10 && got.a == 123)
	ps := []*T49{x, y}
	for _, q := range ps {
		q.a = 31
	}
	q2 := ps[0]
	q2.a = 32
	check("vs8", x.a == 32 && y.a == 31 && ps[0] == x)
	ap := []T49{}
	ap = append(ap, a, h)
	a.a = 1
	h.a = 2
	check("vs9", ap[0].a == 124 && ap[1].a == 77)
	k1 := T49{3}
	c1, d1 := k1, k1
	c1.a = 4
	var e1, f1 = k1, c1
	f1.a = 6
	check("vs10", k1.a == 3 && c1.a == 4 && d1.a == 3 && e1.a == 3 && f1.a == 6)
}

// ===== SECTION 50: methods of a defined type that is not a struct =====
// Expected values read off `go run` (go 1.27). A value of `type C float64` is its
// float64 here, so its method is found by the receiver's static type: a variable,
// a conversion, a field, an element, a range variable and a method's result. A
// pointer receiver of a slice type writes the caller's variable; one of an int
// type through &n too. An interface value takes the one type declaring the method.
type c50 float64
type f50 float64
type stack50 []int
type count50 int
type bag50 map[string]int
type arr50 [3]int
type rd50 struct{ t c50 }
type fer50 interface{ toF() f50 }

func (c c50) toF() f50           { return f50(c*9/5 + 32) }
func (f f50) toC() c50           { return c50((f - 32) * 5 / 9) }
func (c c50) desc() string       { return fmt.Sprintf("%.1fC", float64(c)) }
func (f f50) desc() string       { return fmt.Sprintf("%.1fF", float64(f)) }
func (s *stack50) push(v int)    { *s = append(*s, v) }
func (s stack50) top() int       { return s[len(s)-1] }
func (n *count50) add(k int)     { *n += count50(k) }
func (n count50) get() int       { return int(n) }
func (b bag50) put(k string)     { b[k]++ }
func (a arr50) zero()            { a[0] = 0 }
func (a arr50) sum() int         { return a[0] + a[1] + a[2] }
func late50(n int) late50T       { return late50T(n) }
func (l late50T) twice() late50T { return l * 2 }

type late50T int

func s50() {
	c := c50(100)
	check("dtm1", c.desc() == "100.0C" && c.toF().desc() == "212.0F")
	check("dtm2", c.toF().toC().desc() == "100.0C")
	var st stack50
	st.push(1)
	st.push(2)
	check("dtm3", len(st) == 2 && st.top() == 2)
	var n count50
	n.add(5)
	p := &n
	p.add(1)
	check("dtm4", n.get() == 6 && p.get() == 6)
	b := bag50{}
	b.put("a")
	b.put("a")
	check("dtm5", b["a"] == 2)
	a := arr50{1, 2, 3}
	a.zero()
	check("dtm6", a.sum() == 6)
	r := rd50{t: 40}
	check("dtm7", r.t.desc() == "40.0C")
	cs := []c50{0, 100}
	out := ""
	for _, x := range cs {
		out += x.desc()
	}
	check("dtm8", out == "0.0C100.0C" && cs[1].toF().desc() == "212.0F")
	var fr fer50 = c
	check("dtm9", fr.toF().desc() == "212.0F")
	check("dtm10", late50(3).twice() == 6)
}

// ===== SECTION 51: map keys, receiver copies, and copies of call results =====
// Expected values read off `go run` (go 1.27). A pointer map key is the pointer, a
// struct key is its value. A value receiver is a SHALLOW copy: a write through its
// pointer field or into its slice reaches the caller. A struct reached through a
// call's result is still copied by a store, and an array ranged with its value
// bound is ranged over a copy. The helpers are declared BELOW their use.
type k51 struct{ a int }
type in51 struct{ a int }
type u51 struct {
	p  *in51
	v  in51
	xs []int
	ar [2]int
}

func (u u51) touch() {
	u.p.a = 5
	u.v.a = 6
	u.xs[0] = 7
	u.ar[0] = 8
}

func s51() {
	m := map[*k51]int{}
	k1 := &k51{1}
	k2 := &k51{1}
	m[k1] = 1
	m[k2] = 2
	check("mk1", len(m) == 2 && m[k1] == 1 && m[k2] == 2)
	lit := map[*k51]int{k1: 3, k2: 4}
	check("mk2", len(lit) == 2 && lit[k1] == 3)
	v := map[k51]int{}
	v[k51{1}] = 1
	v[k51{1}] = 2
	check("mk3", len(v) == 1 && v[k51{1}] == 2)
	u := u51{p: &in51{1}, v: in51{1}, xs: []int{1}, ar: [2]int{1, 1}}
	u.touch()
	check("rc1", u.p.a == 5 && u.v.a == 1 && u.xs[0] == 7 && u.ar[0] == 1)
	w := mkw51()
	x := w.v
	x.a = 9
	check("cr1", w.v.a == 1)
	y := get51().v
	y.a = 7
	n := 0
	for _, e := range get51().vs {
		e.a = 97
		n += e.a
	}
	check("cr2", g51.v.a == 1 && g51.vs[0].a == 1 && n == 194)
	arr := mkw51().vs
	b := arr
	b[0].a = 5
	check("cr3", arr[0].a == 1)
	var i interface{} = w.v
	w.v.a = 3
	check("cr4", i.(in51).a == 1)
	ar2 := [2]in51{{1}, {2}}
	got := 0
	for j, e := range ar2 {
		ar2[1].a = 60
		if j == 1 {
			got = e.a
		}
	}
	nums := [3]int{1, 2, 3}
	sum := 0
	for _, e := range nums {
		nums[2] = 100
		sum += e
	}
	pn := &nums
	live := 0
	for _, e := range pn {
		pn[2] = 7
		live += e
	}
	check("cr5", got == 2 && sum == 6 && live == 1+2+7)
}

type w51 struct {
	v  in51
	vs [2]in51
}

func mkw51() w51  { return w51{v: in51{1}, vs: [2]in51{{1}, {2}}} }
func get51() *w51 { return &g51 }

var g51 = mkw51()

// ===== SECTION 52: package variable order, a pointer to a slice, defined types in interfaces =====
// Expected values read off `go run` (go 1.27). A package variable waits for the
// variables its initializer reaches, through the functions it calls, so g52 is
// initialised after h52 and the group's a52 after b52. &xs of a slice is the
// variable: append through the pointer reaches it. A value of a defined
// non-struct type keeps its type in an interface: fmt calls its String() and
// Error(), %T names it, a type switch and x.(T) tell it from its underlying type
// (and int64 from int), and a method called through the interface finds it.
var g52 = f52()

func f52() int { return 41 + h52 }

var h52 = 1

var (
	a52 = b52 + 1
	b52 = 2
)

type col52 int
type sq52 float64
type circ52 float64
type err52 int
type shape52 interface{ area() float64 }

const (
	red52 col52 = iota
	green52
)

func (c col52) String() string { return [...]string{"red", "green"}[c] }
func (s sq52) area() float64   { return float64(s * s) }
func (c circ52) area() float64 { x := float64(c * c); return x + x + x }
func (e err52) Error() string  { return fmt.Sprint("code ", int(e)) }
func add52(p *[]int, v int)    { *p = append(*p, v) }
func find52(c int) error       { return err52(c) }
func kind52(v any) string {
	switch t := v.(type) {
	case col52:
		return fmt.Sprint("col ", int(t)+1)
	case int:
		return "int"
	case int64:
		return "int64"
	case fmt.Stringer:
		return "stringer"
	}
	return "other"
}

func s52() {
	check("vo1", g52 == 42 && h52 == 1 && a52 == 3)
	xs := []int{}
	p := &xs
	*p = append(*p, 1)
	add52(&xs, 2)
	check("ps1", len(xs) == 2 && len(*p) == 2 && xs[1] == 2 && (*p)[0] == 1)
	c := green52
	check("dt1", fmt.Sprint(c, red52) == "green red" && fmt.Sprintf("%v|%s|%d|%T", c, c, c, c) == "green|green|1|main.col52")
	var sh shape52 = sq52(2)
	ss := []shape52{sq52(1), circ52(1)}
	check("dt2", sh.area() == 4 && ss[0].area()+ss[1].area() == 4)
	check("dt3", kind52(green52) == "col 2" && kind52(3) == "int" && kind52(int64(3)) == "int64")
	var v any = col52(1)
	_, isInt := v.(int)
	m, isCol := v.(col52)
	check("dt4", !isInt && isCol && m+1 == 2 && v == col52(1) && v != any(1))
	e := find52(7)
	var ec err52
	check("dt5", e.Error() == "code 7" && fmt.Sprint(e) == "code 7" && errors.As(e, &ec) && ec == 7)
	r := fmt.Sprint(func() (r any) {
		defer func() { r = recover() }()
		_ = v.(int)
		return nil
	}())
	check("dt6", r == "interface conversion: interface {} is main.col52, not int")
	check("dt7", fmt.Sprint(str52("a"), str52("b"), 1, 2) == "ab1 2" && fmt.Sprintf("%q|%v", str52("x"), str52("y")) == "\"x\"|y")
	pt := &pt52{3}
	var np *pt52
	var ch chan int
	check("dt8", fmt.Sprintf("%v|%+v|%#v|%T|%T|%v|%T", pt, &pt52{4}, pt, pt, np, np, ch) == "&{3}|&{a:4}|&main.pt52{a:3}|*main.pt52|*main.pt52|<nil>|chan int")
}

type str52 string
type pt52 struct{ a int }

// ===== SECTION 53: float arithmetic on a defined float type, method values, method sets in fmt =====
type circ53 float64

func (c circ53) area() float64 { return 3 * float64(c) }
func (c circ53) half() float64 { return float64(c)/2 - 0.25 }

type col53 int

func (c col53) String() string  { return fmt.Sprintf("C%d", int(c)) }
func (c col53) add(d int) col53 { return c + col53(d) }

type lst53 []int

func (l *lst53) push(x int) { *l = append(*l, x) }

type pv53 struct{ n int }

func (t *pv53) String() string { return "ptr" }

type pe53 struct{ m string }

func (e *pe53) Error() string { return "E:" + e.m }

func s53() {
	check("fm1", circ53(2).area() == 6 && circ53(3).half() == 1.25)
	c := col53(1)
	f := c.String
	c = 5
	g := c.add
	check("mv1", f() == "C1" && g(2) == 7 && c.String() == "C5")
	var l lst53
	p := l.push
	p(1)
	p(2)
	check("mv2", len(l) == 2 && l[1] == 2)
	e := col53.String
	fs := []func() string{c.String, col53(9).String}
	check("mv3", e(3) == "C3" && fs[0]()+fs[1]() == "C5C9")
	t := &pv53{5}
	tv := pv53{6}
	check("ms1", fmt.Sprint(*t) == "{5}" && fmt.Sprint(t) == "ptr" && fmt.Sprint(tv, &tv) == "{6} ptr")
	check("ms2", fmt.Sprintf("%v %s %+v|%v", tv, tv, tv, &tv) == "{6} {%!s(int=6)} {n:6}|ptr")
	var err error = &pe53{"x"}
	ev := pe53{"y"}
	var st fmt.Stringer = &pv53{1}
	check("ms3", fmt.Sprint(err) == "E:x" && fmt.Sprint(ev) == "{y}" && fmt.Sprint(st) == "ptr")
}

// ===== SECTION 54: the strings package =====
func s54() {
	check("str1", strings.Join([]string{"a", "b", "c"}, ", ") == "a, b, c" && strings.Join(nil, "x") == "" && strings.Repeat("ab", 3) == "ababab")
	check("str2", strings.Index("héllo", "l") == 3 && strings.LastIndex("abcabc", "bc") == 4 && strings.Index("abc", "") == 0 && strings.IndexRune("héllo", 'o') == 5)
	check("str3", strings.Contains("seafood", "foo") && !strings.ContainsRune("abc", 'z') && strings.ContainsAny("hello", "xyz l") && strings.Count("cheese", "e") == 3 && strings.Count("five", "") == 5)
	check("str4", strings.HasPrefix("golang", "go") && strings.HasSuffix("golang", "ng") && strings.TrimPrefix("golang", "go") == "lang" && strings.TrimSuffix("x.go", ".go") == "x")
	check("str5", fmt.Sprint(strings.Split("a,b,,c", ",")) == "[a b  c]" && fmt.Sprint(strings.SplitN("a,b,c", ",", 2)) == "[a b,c]" && fmt.Sprintf("%q", strings.SplitAfter("a,b", ",")) == `["a," "b"]` && fmt.Sprintf("%q", strings.Split("hé", "")) == `["h" "é"]`)
	check("str6", fmt.Sprintf("%q", strings.Fields("  x y\tz\n ")) == `["x" "y" "z"]` && strings.TrimSpace("\t a b \n") == "a b" && strings.Trim("xxhixx", "x") == "hi" && strings.TrimLeft("aab", "a") == "b" && strings.TrimRight("baa", "a") == "b")
	check("str7", strings.ToUpper("héllo wörld ß") == "HÉLLO WÖRLD ß" && strings.ToLower("ΑΒΓ Привет İ") == "αβγ привет i" && strings.ToUpper("ĳ ź ς") == "Ĳ Ź Σ")
	check("str8", strings.EqualFold("Go", "GO") && !strings.EqualFold("ı", "I") && strings.EqualFold("ſ", "S") && strings.Compare("a", "b") == -1)
	check("str9", strings.Replace("oink oink oink", "k", "ky", 2) == "oinky oinky oink" && strings.ReplaceAll("abab", "a", "_") == "_b_b" && strings.Replace("ab", "", "-", -1) == "-a-b-")
	r := strings.NewReplacer("a", "1", "b", "2", "ab", "X")
	r2 := strings.NewReplacer("<", "&lt;", ">", "&gt;")
	check("str10", r.Replace("abcab") == "12c12" && r2.Replace("a<b>") == "a&lt;b&gt;")
	b, a, ok := strings.Cut("k=v", "=")
	check("str11", b == "k" && a == "v" && ok && strings.Map(func(r rune) rune { return r + 1 }, "abc") == "bcd")
	var sb strings.Builder
	sb.WriteString("hé")
	sb.WriteByte('!')
	sb.WriteRune('ö')
	check("str12", sb.String() == "hé!ö" && sb.Len() == 6)
}

// ===== SECTION 55: range plans and per-iteration loop variables =====
func nilRange55(xs []string) int {
	n := 0
	for i, x := range xs {
		n += i + len(x)
	}
	return n
}

func s55() {
	var nm map[string]int
	k := 0
	for range nm {
		k++
	}
	check("rg1", nilRange55(nil) == 0 && nilRange55([]string{"ab", "c"}) == 4 && k == 0)
	var fs []func() int
	for i, v := range []int{4, 5, 6} {
		fs = append(fs, func() int { return i*10 + v })
	}
	check("rg2", fs[0]() == 4 && fs[1]() == 15 && fs[2]() == 26)
	n := 0
	for i := 0; i < 10; {
		n++
		if i < 3 {
			i += 3
			continue
		}
		i++
	}
	check("rg3", n == 8)
	var i2, v2 int
	for i2, v2 = range []int{3, 4} {
	}
	sum := 0
	for i := range 4 {
		sum += i
	}
	check("rg4", i2 == 1 && v2 == 4 && sum == 6)
}

// ===== SECTION 56: static types reaching fmt and interfaces =====
type col56 int

func (c col56) String() string { return fmt.Sprintf("c%d", int(c)) }

type box56 struct {
	v any
	s fmt.Stringer
}

type wrap56 struct{ in box56 }

type get56 interface{ get(x any) string }
type imp56 struct{}

func (imp56) get(x any) string { return fmt.Sprintf("%v %T", x, x) }

func sig56(a, b string) (int, error) { return 0, nil }

func s56() {
	check("ty1", fmt.Sprintf("%T|%T|%T", sig56, s56, func(x int) bool { return x > 0 }) == "func(string, string) (int, error)|func()|func(int) bool")
	n := 5
	f := 2.5
	s := "a"
	r := 'x'
	b := true
	check("ty2", fmt.Sprintf("%T %T %T %T %T", &n, &f, &s, &r, &b) == "*int *float64 *string *int32 *bool")
	bx := box56{col56(1), col56(2)}
	var w wrap56
	w.in.v = col56(3)
	an := struct{ x any }{col56(4)}
	check("if1", fmt.Sprint(bx.v, bx.s, w.in.v, an.x) == "c1 c2 c3 c4" && fmt.Sprintf("%T", bx.v) == "main.col56")
	ch := make(chan any, 1)
	ch <- col56(5)
	var g get56 = imp56{}
	check("if2", fmt.Sprint(<-ch) == "c5" && g.get(col56(6)) == "c6 main.col56")
}

// ===== SECTION 57: len, cap, x[i] and stores by static type =====
type sl57 []int
type ar57 [3]int
type st57 string
type bag57 struct {
	items []int
	arr   [2]int
	m     map[string][]int
}

// The type parameter S57 shadows the package-level type S57 on purpose: a type
// parameter stands for whatever instantiates it.
type S57 [2]int

func tlen57[S57 ~*[3]int](x S57) int { return len(x) + cap(x) }

func nil57(xs []int) int { return len(xs) + cap(xs) }

func set57(g ar57) ar57 {
	g[0] = 99
	return g
}

func s57() {
	t := sl57{1, 2, 3}
	check("sty1", len(t) == 3 && cap(t) == 3 && t[1] == 2 && nil57(nil) == 0)
	var ta [3]int
	check("sty2", tlen57(&ta) == 6 && len(st57("hé")) == 3 && st57("hé")[1] == 195)
	var g ar57
	g2 := set57(g)
	check("sty3", g[0] == 0 && g2[0] == 99 && len(g) == 3 && cap(g) == 3)
	b := bag57{items: []int{4, 5}, m: map[string][]int{"k": {7, 8, 9}}}
	b.items[1] = 50
	b.arr[1] = 3
	b.m["k"][2] = 90
	check("sty4", b.items[1] == 50 && b.arr[1] == 3 && len(b.m["k"]) == 3 && b.m["k"][2] == 90)
	a := [2]int{1, 2}
	var x any = a
	c := a
	a[0] = 7
	c[1] = 8
	check("sty5", fmt.Sprint(x, a, c) == "[1 2] [7 2] [1 8]")
	sub := t[0:1]
	var u8 []uint8 = make([]uint8, 2)
	u8[0] = 255
	u8[0]++
	var a8 [2]int8
	a8[0] = 127
	a8[0]++
	check("sty6", len(sub) == 1 && cap(sub) == 3 && u8[0] == 0 && a8[0] == -128)
	p := &ar57{1, 2, 3}
	check("sty7", len(p) == 3 && p[2] == 3)
}

// ===== SECTION 58: values of defined types as interface map keys and in type tests =====
type mi58 int
type ms58 string

func (s ms58) String() string { return "S(" + string(s) + ")" }

type am58 map[any]int

type hold58 struct{ m map[any]string }

type c58 int

func (c c58) String() string { return "c58" }

func sw58(v any) string {
	switch x := v.(type) {
	case ms58:
		return x.String()
	case c58:
		return x.String()
	}
	return "?"
}

func s58() {
	m := map[any]int{mi58(1): 1, 1: 2}
	var k mi58 = 1
	check("mk1", len(m) == 2 && m[k] == 1 && m[1] == 2 && m[mi58(1)] == 1)
	m[k] = 10
	m[ms58("a")] = 3
	m["a"] = 4
	_, ok := m[mi58(2)]
	_, ok2 := m[k]
	check("mk2", len(m) == 4 && m[mi58(1)] == 10 && m[1] == 2 && m[ms58("a")] == 3 && m["a"] == 4 && !ok && ok2)
	delete(m, k)
	m[k]++
	m[k] += 5
	check("mk3", len(m) == 4 && m[mi58(1)] == 6 && m[1] == 2)
	am := am58{mi58(1): 1, 1: 2}
	h := hold58{m: map[any]string{}}
	h.m[k] = "boxed"
	h.m[1] = "plain"
	nested := map[string]map[any]int{"o": {mi58(1): 1}}
	nested["o"][1] = 2
	nested["o"][mi58(1)] += 10
	check("mk4", len(am) == 2 && am[mi58(1)] == 1 && len(h.m) == 2 && h.m[k] == "boxed" && nested["o"][mi58(1)] == 11 && nested["o"][1] == 2)
	keyTy := ""
	for kk := range map[any]bool{mi58(3): true} {
		keyTy = fmt.Sprintf("%v %T", kk, kk)
	}
	sm := map[fmt.Stringer]int{ms58("x"): 1}
	sm[ms58("y")] = 2
	check("mk5", keyTy == "3 main.mi58" && len(sm) == 2 && sm[ms58("x")] == 1)
	var a any = ms58("q")
	t1, ok3 := a.(ms58)
	t2 := a.(ms58)
	check("ty58", fmt.Sprint(t1, ok3, t2) == "S(q)trueS(q)" && sw58(c58(1))+sw58(ms58("w"))+sw58(2) == "c58S(w)?")
	// A defined-type key boxes in a type switch subject, a comma-ok assertion, a
	// deferred call's target and a panic operand as it does in an expression.
	km := map[any]any{mi58(1): "x", 1: 2}
	tsw := "?"
	switch v := km[mi58(1)].(type) {
	case string:
		tsw = "string " + v
	case int:
		tsw = "int"
	}
	_, ok4 := km[mi58(1)].(string)
	dv := ""
	fm := map[any]func(){mi58(1): func() { dv = "boxed" }, 1: func() { dv = "plain" }}
	func() {
		defer fm[mi58(1)]()
	}()
	pv := func() (r any) {
		defer func() { r = recover() }()
		panic(km[mi58(1)])
	}()
	check("mk6", tsw == "string x" && ok4 && dv == "boxed" && pv == "x")
}

// ===== SECTION 59: elements of a defined type inside a composite fmt prints =====
type col59 int

func (c col59) String() string { return fmt.Sprintf("c%d", int(c)) }

type err59 float64

func (e err59) Error() string { return fmt.Sprintf("e%.1f", float64(e)) }

type pk59 int

func (p *pk59) String() string { return "ptr" }

type rec59 struct {
	C  col59
	cs []col59
	M  map[col59]string
}

func s59() {
	src := []int{3, 1, 2}
	ks := []col59{}
	for _, v := range src {
		ks = append(ks, col59(v))
	}
	check("dc1", fmt.Sprint(ks, ks[1:]) == "[c3 c1 c2] [c1 c2]" && fmt.Sprintf("%v|%d|%s|%q", ks, ks, ks, ks[:1]) == "[c3 c1 c2]|[3 1 2]|[c3 c1 c2]|[\"c3\"]")
	m := map[col59]col59{2: 5, 1: 4}
	check("dc2", fmt.Sprint(m) == "map[c1:c4 c2:c5]" && fmt.Sprintf("%+v", map[string][]col59{"a": {7}}) == "map[a:[c7]]")
	r := rec59{C: 1, cs: []col59{2}, M: map[col59]string{3: "x"}}
	check("dc3", fmt.Sprint(r) == "{c1 [2] map[c3:x]}" && fmt.Sprintf("%+v", r) == "{C:c1 cs:[2] M:map[c3:x]}")
	var a any = ks
	check("dc4", fmt.Sprint(a, []err59{1.5}, []pk59{1}, [][]col59{{4}}) == "[c3 c1 c2] [e1.5] [1] [[c4]]")
}

// ===== SECTION 60: static type of arithmetic, composite type tests, nested arrays =====
type k60 int

func (k k60) String() string { return fmt.Sprintf("k%d", int(k)) }

type w60 int64

type g60 struct {
	Cells [2]k60
	Rows  [][2]k60
	hid   [2]k60
}

func ty60(v any) string {
	switch v.(type) {
	case []int:
		return "ints"
	case []k60:
		return "ks"
	case []string:
		return "strs"
	case [][]int:
		return "intss"
	case [2]int:
		return "arr"
	case map[string]int:
		return "msi"
	case []any:
		return "anys"
	}
	return "other"
}

func s60() {
	y := []k60{1, 2, 3}
	kk := k60(5)
	z := kk + y[1]
	check("ar1", fmt.Sprint(y[0]+y[2], kk+1, kk*2, 1+kk, y[0]<<2, z) == "k4 k6 k10 k6 k4 k7")
	var a, b w60 = 2, 3
	var p any = a * b
	_, isW := p.(w60)
	var i1, i2 int64 = 4, 5
	var q any = i1 + i2
	_, is64 := q.(int64)
	check("ar2", isW && is64 && fmt.Sprintf("%v %d %T", y[0]+y[1], y[0]+y[1], kk-1) == "k3 3 main.k60")
	got := ""
	for _, v := range []any{[]int{1}, []k60{1}, []string{"x"}, [][]int{{1}}, [2]int{}, map[string]int{}, []any{1}, make([]int, 1), 3} {
		got += ty60(v) + " "
	}
	_, ok1 := any([]int{}).([]int)
	_, ok2 := any([]int{}).([]k60)
	_, ok3 := any(map[string]int{}).(map[string]int)
	check("ct1", got == "ints ks strs intss arr msi anys ints other " && ok1 && !ok2 && ok3)
	var n [2][2]k60
	n[0][1] = 7
	check("na1", fmt.Sprint(n, n[0]) == "[[k0 k7] [k0 k0]] [k0 k7]" && fmt.Sprintf("%d|%#v", n, n) == "[[0 7] [0 0]]|[2][2]main.k60{[2]main.k60{0, 7}, [2]main.k60{0, 0}}")
	g := g60{}
	g.Cells[1] = 4
	g.hid[0] = 2
	g.Rows = append(g.Rows, [2]k60{5, 6})
	check("na2", fmt.Sprint(g) == "{[k0 k4] [[k5 k6]] [2 0]}" && fmt.Sprintf("%+v", map[string][2]k60{"x": {1, 2}}) == "map[x:[k1 k2]]")
}

// ===== SECTION 61: ordered comparisons with a NaN operand =====
func s61() {
	zs := []float64{0, 1}
	a := zs[0] / zs[0]
	b := zs[1]
	check("nan1", fmt.Sprint(a > b, a >= b, b > a, b >= a, a < b, a <= b, b < a, b <= a) == "false false false false false false false false")
	m := a
	for _, v := range []float64{1, 2} {
		if v > m {
			m = v
		}
	}
	check("nan2", m != m)
}

// ===== SECTION 62: the two phases of a tuple assignment =====
type sv62 struct{ v []int }

var tr62 = ""

func ix62(i int) int { tr62 += fmt.Sprint("i", i, " "); return i }
func vl62(i int) int { tr62 += fmt.Sprint("v", i, " "); return i }

func s62() {
	a := []int{1, 2, 3}
	i := 0
	i, a[i] = 1, 9
	m := map[int]int{}
	n := 1
	n, m[n] = 5, n
	check("tup1", fmt.Sprint(i, a, n, m) == "1 [9 2 3] 5 map[1:1]")
	s := []int{7}
	old := s
	s, s[0] = []int{9}, 100
	ps := &sv62{v: []int{1}}
	ops := ps
	ps, ps.v[0] = &sv62{v: []int{2}}, 42
	check("tup2", fmt.Sprint(s, old, ps.v, ops.v) == "[9] [100] [2] [42]")
	d := []int{0, 0}
	d[ix62(0)], d[ix62(1)] = vl62(5), vl62(6)
	check("tup3", fmt.Sprint(d) == "[5 6]" && tr62 == "i0 i1 v5 v6 ")
	c := []int{10, 20, 30}
	k := 0
	k, c[k], c[k+1] = 2, c[k+2], c[k]
	x, y := 1, 2
	px := &x
	px, *px = &y, 7
	check("tup4", fmt.Sprint(k, c, x, y) == "2 [30 10 30] 7 2")
	tr62 = ""
	e := []int{1, 2}
	e[ix62(1)] += vl62(3)
	e[ix62(0)]++
	check("tup5", fmt.Sprint(e) == "[2 5]" && tr62 == "i1 v3 i0 ")
}

// ===== SECTION 63: copy between overlapping slices =====
func s63() {
	x := []int{1, 2, 3, 4, 5}
	n := copy(x[1:], x)
	y := []int{1, 2, 3, 4, 5}
	m := copy(y, y[2:])
	check("cpy1", fmt.Sprint(n, x, m, y) == "4 [1 1 2 3 4] 3 [3 4 5 4 5]")
	b := []byte("abcde")
	copy(b[2:], b)
	check("cpy2", string(b) == "ababc")
}

// ===== SECTION 64: deleting and updating map entries during range =====
func s64() {
	m := map[int]int{}
	for i := 0; i < 10; i++ {
		m[i] = i * i
	}
	cnt := 0
	for k := range m {
		delete(m, k)
		delete(m, 9-k)
		cnt++
	}
	check("mrd1", cnt == 5 && len(m) == 0 && fmt.Sprint(m) == "map[]")
	m3 := map[int]int{1: 1, 2: 2}
	for k := range m3 {
		delete(m3, k)
		m3[k] = 5
	}
	check("mrd2", fmt.Sprint(m3) == "map[1:5 2:5]")
	m4 := map[string]int{"a": 1, "b": 2, "c": 3}
	sum := 0
	for _, v := range m4 {
		m4["a"], m4["b"], m4["c"] = 100, 100, 100
		sum += v
	}
	check("mrd3", sum > 200)
}

// ===== SECTION 65: interface equality and map keys by dynamic type =====
func s65() {
	fs := []float64{1, 0, -1}
	is := []int{1}
	var a interface{} = fs[0]
	var b interface{} = is[0]
	var c interface{} = float32(1)
	var d interface{} = uint8(1)
	check("ieq1", fmt.Sprint(a == b, b == a, a == c, b == d, a == fs[0], a == 1, a == 1.0, b == 1) == "false false false false true false true true")
	m := map[interface{}]string{}
	m[a] = "float"
	m[b] = "int"
	m[c] = "f32"
	m[d] = "u8"
	check("ieq2", len(m) == 4 && m[a]+" "+m[b]+" "+m[1]+" "+m[1.0]+" "+m[uint8(1)] == "float int int float u8")
	got := ""
	switch b {
	case 1.0:
		got = "float"
	case 1:
		got = "int"
	}
	check("ieq3", got == "int")
	nan := fs[1] / fs[1]
	negz := fs[1] / fs[2]
	fm := map[float64]int{}
	fm[nan] = 1
	fm[nan] = 2
	fm[0.0] = 3
	fm[negz] = 4
	fm[fs[0]] = 5
	fm[1] = 6
	check("fkey1", fmt.Sprint(len(fm), fm[0], fm[nan], fm[1]) == "4 4 0 6")
}

// ===== SECTION 66: clear, min and max =====
type p66 struct {
	A int
	B string
}

func s66() {
	s := []string{"a", "b"}
	clear(s)
	bs := []bool{true}
	clear(bs)
	ps := []p66{{1, "x"}}
	clear(ps)
	pp := []*p66{{1, "x"}}
	clear(pp)
	ss := [][]int{{1}}
	clear(ss)
	check("clr1", fmt.Sprintf("%q %v %v %v %v", s, bs, ps, ss, s[0] == "" && pp[0] == nil && ss[0] == nil) == `["" ""] [false] [{0 }] [[]] true`)
	zs := []float64{0, -1, 1}
	nan := zs[0] / zs[0]
	one := zs[2]
	check("mm1", fmt.Sprint(max(nan, one), max(one, nan), min(nan, one), min(one, nan)) == "NaN NaN NaN NaN")
	z := zs[0]
	nz := zs[0] / zs[1]
	check("mm2", fmt.Sprint(1/max(z, nz), 1/max(nz, z), 1/min(z, nz), 1/min(nz, z)) == "+Inf +Inf -Inf -Inf")
	is := []int8{3, -2}
	us := []uint64{18446744073709551615, 18446744073709551614}
	check("mm3", fmt.Sprint(min(is[0], is[1]), max(is[0], is[1]), min(us[0], us[1])) == "-2 3 18446744073709551614")
}

// ===== SECTION 67: run-time panics of close, send and make =====
func rc67(f func()) (s string) {
	defer func() { s = fmt.Sprint(recover()) }()
	f()
	return "none"
}

func s67() {
	ns := []int{3, 2, -1, 0}
	got := rc67(func() { c := make(chan int); close(c); close(c) }) + "|" +
		rc67(func() { var c chan int; close(c) }) + "|" +
		rc67(func() { c := make(chan int, 1); close(c); c <- 1 })
	check("pch1", got == "close of closed channel|close of nil channel|send on closed channel")
	got = rc67(func() { _ = make([]int, ns[0], ns[1]) }) + "|" +
		rc67(func() { _ = make([]int, ns[2]) }) + "|" +
		rc67(func() { _ = make([]int, ns[3], ns[2]) }) + "|" +
		rc67(func() { _ = make(chan int, ns[2]) }) + "|" +
		rc67(func() { _ = make([]int, ns[1], ns[0]) })
	check("pmk1", got == "runtime error: makeslice: cap out of range|runtime error: makeslice: len out of range|"+
		"runtime error: makeslice: cap out of range|makechan: size out of range|<nil>")
}

// ===== SECTION 68: the dynamic type of runes and bytes =====
func k68(v any) string {
	switch v.(type) {
	case int:
		return "int"
	case int32:
		return "int32"
	case uint8:
		return "uint8"
	}
	return "other"
}

func r68() rune { return 'r' }
func i68() int  { return 'i' }

func s68() {
	s := "az"
	got := ""
	for _, c := range s {
		got += k68(c) + " "
	}
	check("rt1", got == "int32 int32 " && fmt.Sprintf("%T %T %T", 'z', []rune(s)[0], s[0]) == "int32 int32 uint8")
	x := 'q'
	var y = 'a' + 1
	var z any = 'w'
	check("rt2", fmt.Sprintf("%T %T %T %T %T", x, y, z, r68(), i68()) == "int32 int32 int32 int32 int")
	freq := map[rune]int{}
	for _, c := range "abca" {
		freq[c]++
	}
	n := 0
	for i := 0; i < len(s); i++ {
		n = n*1000 + int(s[i]-'a')
	}
	check("rt3", fmt.Sprint(freq, freq['a'], n) == "map[97:2 98:1 99:1] 2 25")
}

// ===== SECTION 69: fmt's order of composite and float map keys =====
type k69 struct {
	A int
	B string
}

func s69() {
	m := map[k69]int{{1, "a"}: 1, {0, "b"}: 2, {1, ""}: 3, {-1, "z"}: 4, {10, "a"}: 5, {2, "a"}: 6}
	check("mko1", fmt.Sprint(m) == "map[{-1 z}:4 {0 b}:2 {1 }:3 {1 a}:1 {2 a}:6 {10 a}:5]")
	a := map[[2]int]string{{1, 1}: "a", {1, 2}: "b", {10, 0}: "c", {2, 1}: "d"}
	check("mko2", fmt.Sprint(a) == "map[[1 1]:a [1 2]:b [2 1]:d [10 0]:c]")
	zs := []float64{0}
	nan := zs[0] / zs[0]
	fm := map[float64]int{3: 1, nan: 2, -1: 3}
	r := map[rune]int{'d': 1, 'a': 2}
	check("mko3", fmt.Sprint(fm, r) == "map[NaN:2 -1:3 3:1] map[97:2 100:1]")
}

// ===== SECTION 70: conversion of nil to a slice type =====
type l70 []int

func s70() {
	z := []int(nil)
	var e []int
	a := append(z, e...)
	check("nsl1", fmt.Sprint(z == nil, len(z), z, a == nil) == "true 0 [] true")
	b := []byte(nil)
	l := l70(nil)
	check("nsl2", fmt.Sprint(b == nil, len(b), l == nil, l) == "true 0 true []")
}

// ===== SECTION 71: string of an invalid code point =====
func s71() {
	got := ""
	for _, r := range []int{-1, 0xD800, 0x110000, 0x10FFFF} {
		got += fmt.Sprintf("%x ", string(rune(r)))
	}
	check("ivr1", got == "efbfbd efbfbd efbfbd f48fbfbf " && string([]rune{0xDFFF, 'a'}) == "�a")
}

// ===== SECTION 72: Unicode case of digraphs and of the letterlike symbols =====
func s72() {
	check("ucs1", strings.ToUpper("ǅungla") == "ǄUNGLA" && strings.ToLower("ǅungla") == "ǆungla" &&
		strings.ToUpper("ǳ") == "Ǳ" && strings.ToLower("ẞΩ") == "ßω")
	check("ucs2", strings.EqualFold("ǅ", "ǆ") && strings.EqualFold("K", "k") &&
		strings.EqualFold("Å", "å") && !strings.EqualFold("ǲ", "dz"))
}

// ===== SECTION 73: make(map, hint), fmt.Sprintln, strings.Title =====
var n73 = 0

func h73() int { n73++; return 8 }

func s73() {
	m := make(map[string]int, h73())
	m["a"] = 1
	check("feat1", len(m) == 1 && n73 == 1)
	check("feat2", fmt.Sprintln("a", 1, []int{2}) == "a 1 [2]\n" && fmt.Sprintln() == "\n")
	check("feat3", strings.Title("hello wOrld-foo_bar x.y ǆz") == "Hello WOrld-Foo_bar X.Y ǅz")
}

// ===== SECTION 74: indexing several long strings in turn =====
// s[i] is a byte of the UTF-8 encoding; four strings indexed round robin, two of
// them equal in length and different in content, then two more.
func s74() {
	ss := []string{strings.Repeat("aé", 300), strings.Repeat("日x", 200), "plain ascii",
		strings.Repeat("a", 1) + strings.Repeat("éa", 300)}
	sum := 0
	for r := 0; r < 3; r++ {
		for i := 0; i < 600; i++ {
			for k, s := range ss {
				sum += int(s[i%len(s)]) * (k + 1)
			}
		}
	}
	check("sidx1", sum == 2514786)
	check("sidx2", ss[1][1] == 151 && "€€"[5] == 172 && strings.Repeat("ü", 7)[13] == 188 && ss[3][1] == 195 && ss[3][2] == 169)
}

// ===== SECTION 75: []byte holds the UTF-8 bytes; a channel's len and cap =====
// A string spread into append hands over its bytes, and a '<-' opening a line is a
// receive statement, not the continuation of the line before.
func s75() {
	b := []byte("az")
	check("byt1", fmt.Sprintf("%T %T", b[0], b) == "uint8 []uint8" && b[0]+200 == 41)
	u := []byte("hé日")
	check("byt2", len(u) == 6 && fmt.Sprint(u) == "[104 195 169 230 151 165]" && string(u) == "hé日")
	u = append([]byte("x:"), "€"...)
	check("byt3", len(u) == 5 && u[2] == 226 && string(u) == "x:€")
	var sb strings.Builder
	sb.Write([]byte{0xe2, 0x82, 0xac})
	check("byt4", sb.String() == "€" && string([]rune("é€")) == "é€" && len([]rune("é€")) == 2)
	c := make(chan int, 3)
	c <- 1
	c <- 2
	check("chc1", cap(c) == 3 && len(c) == 2)
	<-c
	check("chc2", len(c) == 1)
	d := make(chan string)
	var e chan int
	check("chc3", cap(d) == 0 && len(d) == 0 && cap(e) == 0 && len(e) == 0)
}

// ===== SECTION 76: the math package =====
// Every operand is read out of a slice so nothing folds at compile time; the
// expected texts are go run's.
func s76() {
	xs := []float64{2.5, -2.5, 0.49999999999999994, -0.5, 1e-300, 5e-324, 4503599627370495.5}
	got := ""
	for _, x := range xs {
		fr, ex := math.Frexp(x)
		ip, fp := math.Modf(x)
		got += fmt.Sprint(math.Floor(x), math.Ceil(x), math.Trunc(x), math.Round(x), math.RoundToEven(x),
			math.Sqrt(math.Abs(x)), fr, ex, ip, fp, math.Mod(x, 0.75), math.Pow(x, 3), math.Ldexp(x, -1070)) + ";"
	}
	check("mth1", got == "2 3 2 3 2 1.5811388300841898 0.625 2 2 0.5 0.25 15.625 2e-322;"+
		"-3 -2 -2 -3 -2 1.5811388300841898 -0.625 2 -2 -0.5 -0.25 -15.625 -2e-322;"+
		"0 1 0 0 0 0.7071067811865475 0.9999999999999999 -1 0 0.49999999999999994 0.49999999999999994 0.12499999999999996 4e-323;"+
		"-1 -0 -0 -1 -0 0.7071067811865476 -0.5 0 -0 -0.5 -0.5 -0.125 -4e-323;"+
		"0 1 0 0 0 1e-150 0.6696928794914171 -996 0 1e-300 1e-300 0 0;"+
		"0 1 0 0 0 2.2227587494850775e-162 0.5 -1073 0 5e-324 5e-324 0 0;"+
		"4.503599627370495e+15 4.503599627370496e+15 4.503599627370495e+15 4.503599627370496e+15 4.503599627370496e+15 "+
		"6.710886399999999e+07 0.9999999999999999 52 4.503599627370495e+15 0.5 0.5 9.13438523331814e+46 3.560118173611522e-307;")
	vs := []float64{0, -1, 2, 10}
	check("mth2", fmt.Sprint(math.Pow(vs[3], -3), math.Pow(vs[2], 0.5), math.Pow(vs[1], 1e300), math.Pow(vs[0], -1),
		math.Sqrt(vs[1]), math.Max(vs[0], -vs[0]), math.Min(vs[0], -vs[0]), math.Dim(vs[2], vs[3])) ==
		"0.001 1.4142135623730951 1 +Inf NaN 0 -0 0")
	// A float64 holding the constant: the constant itself compares at arbitrary
	// precision in Go, so math.Pi == 3.141592653589793 is false there.
	pi, tiny := math.Pi, math.SmallestNonzeroFloat64
	check("mth3", math.IsNaN(math.NaN()) && math.IsInf(math.Inf(-1), -1) && !math.IsInf(math.MaxFloat64, 0) &&
		math.Signbit(math.Copysign(0, -1)) && pi == 3.141592653589793 && math.MaxInt64 == 1<<63-1 &&
		math.MinInt64 == -1<<63 && math.MaxUint8 == 255 && tiny == 5e-324 && tiny/2 == 0)
}

// ===== SECTION 77: fmt.Fprint* and io.Writer =====
// Fprint, Fprintln and Fprintf hand their text to w.Write as a []byte and answer
// what Write answers; os.Stdout is a writer too.
type w77 struct{ parts []string }

func (w *w77) Write(p []byte) (int, error) {
	w.parts = append(w.parts, string(p))
	return len(p), nil
}

func put77(w io.Writer, k string, v int) (int, error) { return fmt.Fprintf(w, "%s=%03d;", k, v) }

func s77() {
	var sb strings.Builder
	n, err := fmt.Fprintf(&sb, "%d-%s|", 5, "é")
	fmt.Fprint(&sb, "a", 1, 2, "b")
	fmt.Fprintln(&sb, "", 3.5)
	check("fpr1", sb.String() == "5-é|a1 2b 3.5\n" && n == 5 && err == nil)
	w := &w77{}
	m, _ := put77(w, "k", 7)
	put77(w, "z", -1)
	check("fpr2", m == 6 && strings.Join(w.parts, "") == "k=007;z=-01;" && len(w.parts) == 2)
	k, e2 := os.Stdout.WriteString("")
	j, _ := fmt.Fprint(os.Stdout)
	check("fpr3", k == 0 && e2 == nil && j == 0)
}

// ===== SECTION 78: a literal 0 at the end of a line =====
// The x, o or b of 0x, 0o, 0b belongs to the literal only when glued to the 0.
func s78() {
	a := 0
	x1, b1 := 3, 4
	c := 0
	o2 := 5
	d := 0
	b3 := 6
	check("rdx1", a+c+d == 0 && x1 == 3 && b1 == 4 && o2 == 5 && b3 == 6 && 0x1f+0o17+0b11 == 49)
}

// ===== SECTION 79: a string holding bytes that are not a character =====
// A Go string is bytes: a slice may cut a rune, string([]byte) and a \x escape may
// hold any byte, and each such byte counts, indexes, compares and concatenates as
// itself while ranging and []rune read it as U+FFFD. The expected values are go run's.
func s79() {
	ws := []string{"日本語", "a\xffb"}
	w := ws[0]
	a, b := w[0:1], w[1:3]
	ra := []rune(a)
	check("ibs1", len(a) == 1 && len(ra) == 1 && ra[0] == 65533 && a != "�" && a[0] == 230 && len(b) == 2)
	c := string([]byte{200})
	check("ibs2", len(c) == 1 && c == "\xc8" && []byte(c)[0] == 200 && len(string([]byte{0xe6, 0x97, 0xa5})) == 3)
	ab := a + b
	d := a
	d += b
	check("ibs3", ab == "日" && d == "日" && len(ab) == 3 && []rune(ab)[0] == 26085 && a+"" == a)
	check("ibs4", "\xe6\x97\xa5" == w[0:3] && len("x\xe6") == 2 && '\xff' == 255 && '\377' == 255 && len([]rune("\xed\xa0\x80")) == 3)
	got := ""
	for i, r := range ws[1] {
		got += fmt.Sprint(i, ":", r, " ")
	}
	check("ibs5", got == "0:97 1:65533 2:98 " && len(ws[1]) == 3 && ws[1][1] == 255)
	check("ibs6", fmt.Sprintf("%q %x %+q", w[0:4], w[0:4], a) == `"日\xe6" e697a5e6 "\xe6"` && fmt.Sprintf("%s%s", a, b) == "日")
	var sb strings.Builder
	for i := 0; i < len(w); i++ {
		sb.WriteByte(w[i])
	}
	check("ibs7", sb.String() == w && strings.Index(w, b) == 1 && strings.LastIndex(w, w[3:4]) == 3 &&
		strings.TrimSpace(" "+w[0:2]+" ") == w[0:2] && strings.ToUpper(a) == "�")
	check("ibs8", !(a < b) && b < a && ws[1][1:2] > ws[1][0:1] && "\xff" > "日")
	m := map[string]int{a: 1, w[1:2]: 5}
	m[string([]byte{0xe6})] += 1
	check("ibs9", len(m) == 2 && m[a] == 2 && m[w[3:4]] == 2 && m[w[1:2]] == 5)
	// Six long strings that are not ASCII, indexed in turn: each s[i] is one byte
	// read, not a fresh encoding of the string (the frozen interpreter's step brake
	// stops a per-index encoding).
	var ls []string
	for k := 0; k < 6; k++ {
		ls = append(ls, strings.Repeat("é", 1200)+fmt.Sprint(k))
	}
	sum := 0
	for i := 0; i < 200; i++ {
		for k := range ls {
			sum += int(ls[k][i*12])
		}
	}
	check("ibs10", sum == 234000 && ls[5][2400] == '5')
}

// ===== SECTION 80: math's transcendental functions, bits and exact constants =====
// Every result is go run's on arm64 to the last bit, which is where the FMAs its
// compiler fuses into the polynomials show; operands come out of slices.
func s80() {
	xs := []float64{0.5, 1.7, -2.3, 10, 1e-5, 3e8, 1e300, 5e-324, 0.7853981633974483, 2.5e-10}
	got := ""
	for _, x := range xs {
		got += fmt.Sprint(math.Exp(x), math.Log(math.Abs(x)), math.Sin(x), math.Cos(x), math.Tan(x), math.Atan(x),
			math.Cbrt(x), math.Hypot(x, 3)) + ";"
	}
	check("mtr1", got == "1.6487212707001282 -0.6931471805599453 0.479425538604203 0.8775825618903728 0.5463024898437905 0.4636476090008061 0.7937005259840997 3.0413812651491092;"+
		"5.473947391727199 0.5306282510621704 0.9916648104524687 -0.12884449429552464 -7.696602139459161 1.039072259536091 1.193483191927337 3.448187929913334;"+
		"0.10025884372280375 0.832909122935104 -0.7457052121767203 -0.666276021279824 1.1192136417341325 -1.1606689862534056 -1.3200061217959125 3.7802116342871597;"+
		"22026.465794806718 2.302585092994046 -0.5440211108893699 -0.8390715290764524 0.6483608274590867 1.4711276743037345 2.154434690031884 10.44030650891055;"+
		"1.00001000005 -11.512925464970229 9.999999999833334e-06 0.99999999995 1.0000000000333334e-05 9.999999999666668e-06 0.02154434690031884 3.0000000000166667;"+
		"+Inf 19.519293032620475 -0.4395520511837499 0.8982171197990818 -0.48936058052653386 1.5707963234615632 669.4329500821696 3e+08;"+
		"+Inf 690.7755278982137 -0.8178819121159087 -0.5753861119575491 1.4214488238747243 1.5707963267948966 1e+100 1e+300;"+
		"1 -744.4400719213812 5e-324 1 5e-324 5e-324 1.7031839360032603e-108 3;"+
		"2.1932800507380152 -0.2415644752704905 0.7071067811865475 0.7071067811865476 0.9999999999999998 0.6657737500283538 0.9226350743220142 3.101104686247803;"+
		"1.00000000025 -22.1095601980663 2.5e-10 1 2.5e-10 2.5e-10 0.0006299605249474366 3;")
	ys := []float64{2, 0.25, -1.5, 10, 7.7}
	check("mtr2", fmt.Sprint(math.Pow(ys[0], ys[1]), math.Pow(ys[3], ys[2]), math.Pow(ys[4], ys[4]), math.Pow(ys[1], ys[2]),
		math.Pow(-ys[0], ys[1])) == "1.189207115002721 0.031622776601683784 6.698515765072521e+06 8 NaN")
	check("mtr3", math.Float64bits(ys[2]) == 13832806255468478464 && math.Float64frombits(4614253070214989087) == 3.14 &&
		math.Float64bits(xs[7]) == 1 && math.FMA(xs[1], xs[1], -2.89) == -2.753353101070388e-16 && math.FMA(ys[4], 3, 0.1) == 23.2)
	// A package constant is exact until it lands in a typed place.
	check("mtr4", !(math.Pi == 3.141592653589793) && math.Pi > 3.141592653589793 && float64(math.Pi) == 3.141592653589793 &&
		math.MaxUint64/3 == 6148914691236517205 && uint64(math.MaxUint64) == 1<<64-1 && math.E != 2.718281828459045)
}

// ===== SECTION 81: an untyped constant converts to the type of where it lands =====
// An assignment converts like a declaration does, an integer constant beyond int64
// is held by a uint64 or rounded into a float, and one beside a float operand is
// that float. The expected values are go run's.
func u81(x uint64) uint64   { return x | 1 }
func f81(x float64) float64 { return x / 2 }

func s81() {
	var s uint64
	s = 1 << 63
	check("ucv1", fmt.Sprint(s, s|1, s == 1<<63, s > 1<<62) == "9223372036854775808 9223372036854775809 true true")
	xs := []float64{0.5}
	x := xs[0]
	check("ucv2", fmt.Sprint(18014398509481984*x, x*9223372036854775808, x-9223372036854775808, x < 9223372036854775808) ==
		"9.007199254740992e+15 4.611686018427388e+18 -9.223372036854776e+18 true")
	var u8 uint8
	u8 = 255
	u8++
	var f float64
	f = 3
	g := f / 2
	f = 1 << 64
	check("ucv3", u8 == 0 && g == 1.5 && fmt.Sprint(f) == "1.8446744073709552e+19")
	var h float64 = 1<<64 + 1
	check("ucv4", fmt.Sprint(u81(1<<63), f81(1<<64), h, u81(math.MaxUint64)) ==
		"9223372036854775809 9.223372036854776e+18 1.8446744073709552e+19 18446744073709551615")
	// No 64-bit integer holds 1 << 64, so wherever it lands is a float.
	p := P81{1 << 64}
	ys := []float64{-1 << 64}
	ch := make(chan float64, 1)
	ch <- 1 << 65
	check("ucv5", fmt.Sprint(r81(), p, ys, <-ch) == "1.8446744073709552e+19 {1.8446744073709552e+19} [-1.8446744073709552e+19] 3.6893488147419103e+19")
}

type P81 struct{ F float64 }

func r81() float64 { return 1 << 64 }

// ===== SECTION 82: constants at the edge of the type they convert to =====
// go rejects at compile time a constant its location's type cannot hold (the
// go-fail-test-overflow-*.go files); every value here is the largest or smallest
// that fits, at each kind of location, and compiles. The expected values are go run's.
type P82 struct {
	a int
	b uint8
}

type B82 struct{ x uint8 }

type E82 struct {
	B82
	y int8
}

type K82 struct{ l, r int8 }

type L82 []uint8

type M82 map[uint8]int8

const (
	A82 uint8 = iota + 253
	B82c
	C82c
)

func r82() (int8, uint16) {
	g := func() uint8 { return 255 }
	_ = g
	return -128, 65535
}

type C82 int8

const (
	T82a uint8   = 200
	T82b         = T82a + 55
	T82c C82     = -127
	T82s         = "héllo"
	T82t         = T82s + "!"
	Z82f float64 = 0
	F82h float64 = 0.5
	T82f float32 = 3.4028234663852886e38
	T82g float32 = 3.4028235677973366e38
)

func s82() {
	check("cke1", fmt.Sprint(E82{B82{255}, -128}, []P82{{1, 255}, {2, 0}}, map[string]P82{"a": {1, 255}},
		map[K82]uint8{{-128, 127}: 255}, [3]uint8{2: 255}, L82{255}, M82{255: -128}, P82{b: 255, a: 1}) ==
		"{{255} -128} [{1 255} {2 0}] map[a:{1 255}] map[{-128 127}:255] [0 0 255] [255] map[255:-128] {1 255}")
	var u8 uint8 = 1
	var i8 int8 = -1
	x := 5
	check("cke2", fmt.Sprint(u8+254, i8-127, u8 == 255, x<<62, 100+155+u8, A82, B82c, C82c) ==
		"255 -128 false 4611686018427387904 0 253 254 255")
	a, b := r82()
	ch := make(chan uint8, 1)
	ch <- 255
	u8 += 254
	_ = 1<<63 - 1
	check("cke3", fmt.Sprint(a, b, <-ch, u8) == "-128 65535 255 255")
	// A float divided by an integer constant zero is an infinity, also in a program
	// that recovers from an integer divide by zero (this one does): the constant
	// converts to float64.
	f := 2.5
	f /= 0
	g := []float64{1.5}[0]
	check("dvz1", fmt.Sprint(f, g/0, -g/0) == "+Inf +Inf -Inf")
	// Only a negative constant shift count and an INTEGER division by a constant
	// zero are compile errors (go-fail-test-shift.go, go-fail-test-divzero.go).
	c := 1i
	check("cke4", fmt.Sprint(x<<0, x>>63, c/0, -1<<2>>1) == "5 0 (NaN+Infi) -2")
	// A converted constant must fit the type, and a constant index or slice bound
	// must lie inside an array or a constant string (go-fail-test-convert.go,
	// -index.go, -slice.go); these are the extremes that do.
	check("cke5", fmt.Sprint(uint8(255), int8(-128), uint64(1<<64-1), int(2.0), U82(255), rune(0x10FFFF), uint(0)) ==
		"255 -128 18446744073709551615 2 255 1114111 0")
	arr := [3]int{1, 2, 3}
	pa := &arr
	arr[2] = 4
	mn := map[int]int{-1: 9}
	check("cke6", fmt.Sprint(arr[2], pa[2], arr[3:], arr[0:3:3], "abc"[2], "abc"[3:], "héllo"[5], mn[-1]) ==
		"4 4 [] [1 2 4] 99111 9")
	// A float constant reaches an integer location when it is integral
	// (go-fail-test-truncate.go is one that is not).
	var ui uint8 = 2.0
	var ii int = 1e3
	ui += 3.0
	check("cke7", fmt.Sprint(ui, ii, ui+250.0, ii/2.0, []int{4.0}, x<<2.0, 1<<2.0) == "5 1000 255 500 [4] 20 4")
	// A TYPED constant expression must fit its type at every step
	// (go-fail-test-typed-const.go, -typed-divzero.go), and a named constant
	// string bounds a constant index like a literal does (-const-str-index.go,
	// -const-str-slice.go); these are the extremes that compile.
	zf := 1.0
	check("cke8", fmt.Sprint(T82b, T82a/2*2, -T82c, T82c-1, -(T82c-1+1), uint8(200)+55, int8(-128)/1, zf/Z82f) ==
		"255 200 127 -128 127 255 -128 +Inf")
	check("cke9", fmt.Sprint(T82s[5], T82t[6], T82s[6:], T82t[1:7], len(T82t)) == "111 33éllo!7")
	// A spec of a const GROUP holds a value of its type, as a lone one does, and
	// arithmetic on a float32 conversion is binary32 arithmetic.
	check("cke10", fmt.Sprintf("%T %T %T %T %T", T82a, T82b, T82c, T82f, Z82f) == "uint8 uint8 main.C82 float32 float64")
	check("cke11", fmt.Sprint(T82f, float32(1)/3, float32(2)/3+0, F82h*2) == "3.4028235e+38 0.33333334 0.6666667 1")
	// A constant reaching a float32 is rounded ONCE, from its exact value: this
	// one is just below the tie between MaxFloat32 and 2^128, which it would land
	// on exactly as a float64 first.
	var f32 float32 = 3.4028235677973366e38
	check("cke12", fmt.Sprint(f32, float32(3.4028235677973366e38), T82g) == "3.4028235e+38 3.4028235e+38 3.4028235e+38")
}

type U82 uint8

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
	s26() // SECTION-CALL 26
	s27() // SECTION-CALL 27
	s28() // SECTION-CALL 28
	s29() // SECTION-CALL 29
	s30() // SECTION-CALL 30
	s31() // SECTION-CALL 31
	s32() // SECTION-CALL 32
	s33() // SECTION-CALL 33
	s34() // SECTION-CALL 34
	s35() // SECTION-CALL 35
	s36() // SECTION-CALL 36
	s37() // SECTION-CALL 37
	s38() // SECTION-CALL 38
	s39() // SECTION-CALL 39
	s40() // SECTION-CALL 40
	s41() // SECTION-CALL 41
	s42() // SECTION-CALL 42
	s43() // SECTION-CALL 43
	s44() // SECTION-CALL 44
	s45() // SECTION-CALL 45
	s46() // SECTION-CALL 46
	s47() // SECTION-CALL 47
	s48() // SECTION-CALL 48
	s49() // SECTION-CALL 49
	s50() // SECTION-CALL 50
	s51() // SECTION-CALL 51
	s52() // SECTION-CALL 52
	s53() // SECTION-CALL 53
	s54() // SECTION-CALL 54
	s55() // SECTION-CALL 55
	s56() // SECTION-CALL 56
	s57() // SECTION-CALL 57
	s58() // SECTION-CALL 58
	s59() // SECTION-CALL 59
	s60() // SECTION-CALL 60
	s61() // SECTION-CALL 61
	s62() // SECTION-CALL 62
	s63() // SECTION-CALL 63
	s64() // SECTION-CALL 64
	s65() // SECTION-CALL 65
	s66() // SECTION-CALL 66
	s67() // SECTION-CALL 67
	s68() // SECTION-CALL 68
	s69() // SECTION-CALL 69
	s70() // SECTION-CALL 70
	s71() // SECTION-CALL 71
	s72() // SECTION-CALL 72
	s73() // SECTION-CALL 73
	s74() // SECTION-CALL 74
	s75() // SECTION-CALL 75
	s76() // SECTION-CALL 76
	s77() // SECTION-CALL 77
	s78() // SECTION-CALL 78
	s79() // SECTION-CALL 79
	s80() // SECTION-CALL 80
	s81() // SECTION-CALL 81
	s82() // SECTION-CALL 82
	fmt.Println("full:", checks, "checks,", fails, "failures")
	os.Exit(fails)
}
