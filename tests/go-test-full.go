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
	f27 "fmt"
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
	// A TWO-name multi-assign whose first element is an index, a map index, a
	// type assertion or a channel receive: each of those is also the prefix of a
	// comma-ok form, and the comma-ok rules used to swallow it and reject the
	// statement at the comma. Every operand comes out of a container so nothing
	// folds. Three names never hit it, and neither did a first element that
	// could not start a comma-ok form.
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
	// The comma-ok base is a whole postfix expression, not a bare name. A base
	// carrying a suffix used to fall through to a two-name destructure of the
	// single value, which is a SILENT wrong answer (<nil> <nil>), or not to parse.
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
	// The '=' spelling of comma-ok assigns to variables that already exist. The map
	// form did not parse in either half (it fell through to a two-name destructure
	// of one value, i.e. <nil> <nil>), and the assertion form declared a SHADOW
	// instead of assigning - both silent.
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
	// map[]. Both halves used to reject this outright ("the comma ok form needs a
	// map" / "indexing a object") because the zero value of a map type was the null
	// value; it is now a MARKED empty map, which keeps `== nil` true and keeps a
	// STORE a panic. docs/todo.md 1.9.
	n13, n14 := nst05["absent"]["k"]
	check("dec27", n13 == 0 && !n14)
	check("dec28", nst05["absent"]["k"] == 0 && len(nst05["absent"]) == 0)
	var nilm05 map[string]int
	check("dec29", nilm05 == nil && len(nilm05) == 0 && nilm05["k"] == 0)
	nv05, nok05 := nilm05["k"]
	check("dec30", nv05 == 0 && !nok05)
	cnt05 := 0
	// `for range m` with NO variables at all does not parse in the compiler half
	// (it reads `for` as a variable and dies with `variable not defined: for`) -
	// pre-existing, unrelated to the nil map, and stated here because this is where
	// it was met. The one-variable form is the one both halves take.
	for k05 := range nilm05 {
		_ = k05
		cnt05++
	}
	check("dec31", cnt05 == 0 && fmt.Sprint(nilm05) == "map[]")
	// A for-INIT may be a comma-ok form. The interpreter said `unknown name: v`:
	// only a ':=' recorded the names a Go 1.22 loop gives a per-iteration copy of,
	// so the init inherited the last ':=' ANYWHERE EARLIER and the loop frame was
	// rebuilt around those instead. The closure below is what per-iteration means.
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
	// not the backing array: `s[1:3]` has cap 4 here, and reading past its length
	// used to answer the storage in both halves and `<nil>` (exit 0!) in the
	// interpreter. Every index comes out of a slice so nothing folds.
	// docs/todo.md 1.9. The message text is the engine's, not go's, in every half.
	ix07 := []int{0, 2, 3, 7, -1}
	check("slc8", panics07(func() { _ = s[ix07[3]] }) && panics07(func() { _ = s[ix07[4]] }))
	check("slc9", panics07(func() { _ = sub[ix07[1]] }) && sub[ix07[0]] == 30)
	check("slc10", panics07(func() { sub[ix07[1]] = 1 }) && panics07(func() { s[ix07[3]] = 1 }))
	arr07 := [3]int{1, 2, 3}
	check("slc11", panics07(func() { _ = arr07[ix07[2]] }) && panics07(func() { arr07[ix07[2]] = 1 }))
	str07 := []string{"abc"}
	check("slc12", str07[0][ix07[1]] == 99 && panics07(func() { _ = str07[0][ix07[2]] }))
	// A slice EXPRESSION is bounded by the capacity, and s[len:] is legal. Go's
	// full rule is 0 <= lo <= hi <= max <= cap; the three-index form was unchecked
	// in BOTH halves and built a header whose capacity was below its length.
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
	// NESTED COMPOSITE LITERALS whose element type is a MAP. The compiler half
	// rejected every one of these ("composite literal element of unsupported type")
	// where the interpreter accepted them, and a doubly nested one then broke the
	// interpreter instead, because both halves read a map type's VALUE text by
	// scanning for the LAST ']' - which runs straight past the key of an inner map.
	// docs/todo.md 1.9.
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
	// DOUBLES THE SCRIPT HOST MIS-PRINTS. The interpreter half took a double's
	// digits from `"" + a`, and goja's Number-to-String is not always the shortest
	// ROUND-TRIPPING form (~1 in 9,000 doubles), while the frozen host's is exact -
	// so this was three things at once: wrong against `go`, a live goja-vs-`-frozen`
	// divergence no matrix entry reached, and unfixable from inside the dialect,
	// since a script cannot ask its host for more digits than the printer gave. The
	// digits now come from floPrec by ROUND-TRIP SEARCH (docs/todo.md 1.8). Every
	// value here is one goja gets wrong, read out of a slice so nothing folds; the
	// first is the one docs/working-on-this-project.md chapter 4 names.
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
//
// A nil map used to be the third: it printed <nil> where go prints map[], because
// it was the null value in this model. Since docs/todo.md 1.9 the zero value of a
// map type is a MARKED EMPTY MAP - so that it can also be READ like an empty one,
// which is what go does - and it prints map[] in both halves. It is asserted below.
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

	// AND NOT runs at the full width too. It used to be expanded in the compiler
	// grammar as `a & (b ^ -1)` over the 32-bit bitwise pair, so every operand
	// wider than an int32 was truncated; int31/int32 both answered 0 there while
	// the interpreter half, which routes &^ through the sized box, was right.
	var an int64 = 1<<40 | 7
	var am int64 = 7
	check("int31", fmt.Sprint(an&^am) == "1099511627776")
	var au uint64 = 18446744073709551615
	check("int32", fmt.Sprint(au&^1) == "18446744073709551614")
	var aa int64 = 1 << 40
	aa &^= 1 << 40
	check("int33", aa == 0)

	// A shift takes its result type from the LEFT operand alone - the count is a
	// separate operand with a type of its own. Both halves used to read the width
	// and signedness off whichever operand was boxed, so a signed value shifted by
	// an unsigned count came back UNSIGNED and narrowed to the count's width:
	// int34 answered 124 and int36 answered 0.
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

	// `var s T` of a struct type is a zero VALUE, not nil: writing a field of it
	// on the very next line used to be an assignment on null.
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
// semicolon. Verified against go1 (`go run`) before being pinned here.
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
// against go1 (`go run`) before being pinned here.
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
// dereference to the field or index rather than to the whole expression. Verified against
// go1 (`go run`) before being pinned here.
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
// reach through it. Verified against go1 (`go run`) before being pinned here.
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
}

// ===== SECTION 31: local type declarations, and labels with no statement =====
// A `type` declaration is a STATEMENT too: Go scopes a local type to its block, and every
// top-level shape (struct, defined slice/int/map, alias) is legal there. A LABEL may also
// stand alone - `_:` at the end of a block is a labeled EMPTY statement, for which Go
// writes no semicolon; gc rejects the same thing in front of a 'case' or 'default'
// ("missing statement after label"), so the grammar does too. Verified against go1
// (`go run`) before being pinned here.
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
// here is a RE-TYPING that keeps the same backing store, so []byte(bs) is bs. Verified
// against go1 (`go run`) before being pinned here.
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
// they are different constants. 16 of these 32 fail before the folder exists.
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
	// A COMPLEX constant is a PAIR of the same exact rationals, so it folds too - the
	// unfolded path multiplied four already-rounded doubles and drifted in both parts.
	check("nc19", (0.1+0.2i)*(0.1+0.2i) == -0.03+0.04i)
	check("nc20", c34z*c34z == -0.03+0.04i && c34z+c34w == 0.3+0.3i)
	check("nc21", real(c34z*c34w) == 0 && imag(c34z*c34w) == 0.05 && 1+0i == 1)
	check("nc22", c34z/c34w == 0.8+0.6i && -c34z+c34z == 0)
}

// ===== SECTION 35: float32 is a real binary32 =====
// A Go float32 is an IEEE-754 binary32, not a float64 spelled differently, and
// every expected value below was read off `go run` before it was written down.
// It used to be an alias of the float64 box, so float32(1)/float32(3) answered
// 0.3333333333333333 and float32(1e20)*float32(1e20) answered 1e+40 - both
// halves agreeing, and both wrong, which is the defect class byte-identity
// cannot see. The operands come out of a SLICE on purpose: a probe of literals
// tests the grammar's constant folder rather than the runtime.
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
// Every expected string was read off `go run` before it was written down. Three
// separate defects live here: unary minus on a complex answered 0 in both
// compiled halves (js_gineg read the {re,im} box as NaN); both PARTS are
// float64 and were printed with the integer rendering, so 1234567.125 came out
// verbatim where go writes 1.234567125e+06 and an infinity came out
// "Infinity"; and division used the textbook formula where Go's runtime uses
// Smith's algorithm plus the C99 G.5.1 fixups.
func s36() {
	big36 := []float64{1e300}
	res := []float64{0.1, 4, 1234567.125, 0}
	ims := []float64{0.2, 3, -0.5, 0}
	a := complex(res[0], ims[0])
	b := complex(res[1], ims[1])
	// UNARY MINUS. This was `0` in llvm.Run and in the native binary.
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
	// The operators that already worked, kept as a regression fence.
	check("cx9", a+b == complex(res[0]+res[1], ims[0]+ims[1]))
	check("cx10", fmt.Sprint(b*b) == "(7+24i)")

	// ----- THE SIGN OF A ZERO, at all six sites a complex product has -----
	// (a+bi)(c+di) is (ac-bd) + (ad+bc)i: four products and two sums, each
	// carrying its own zero sign. The ABNF tag dialect keeps an integral value
	// as an INTEGER, so a bare `-0.0 * 3` answers +0 in the interpreter where
	// both compiled halves answer -0.0 - a halves divergence, not a cosmetic
	// one. The zeros are built at RUNTIME (`-pz36`), because a Go CONSTANT has
	// no signed zero and `-0.0` written out is exactly 0.
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
	// A SCALAR float64 product loses the sign at the same place, so it is
	// pinned here too: this is what made the complex product diverge.
	check("cx24", fmt.Sprint(pz36*-2) == "-0" && fmt.Sprint(nz36*3) == "-0" && fmt.Sprint(nz36*-3) == "0")
	check("cx25", fmt.Sprint(pz36/-2) == "-0" && fmt.Sprint(nz36+nz36) == "-0" && fmt.Sprint(nz36-pz36) == "-0")

	// ----- A Go CONSTANT HAS NO SIGNED ZERO -----
	// `-0.0` is an untyped constant expression and Go evaluates it exactly, so
	// it is 0 and not -0.0 - `complex(-0.0, 0.0)` is (0+0i) before the program
	// ever runs. Every engine here answered -0 until unary minus became a fold
	// site; it was the root of every remaining difference in a 346-value
	// complex probe. A RUNTIME negation is unaffected, which cx12 pins.
	k36 := -0.0
	var v36 float64 = -0.0
	check("cx26", fmt.Sprint(k36) == "0" && fmt.Sprint(v36) == "0" && fmt.Sprint(-0.0) == "0")
	check("cx27", fmt.Sprint(complex(-0.0, 0.0)) == "(0+0i)")
	check("cx28", fmt.Sprint(cnz36) == "0" && fmt.Sprint(-0.0*1) == "0")
}

// ===== SECTION 37: nil slices, recovered results, `for range x`, panic text =====
// The five gaps of docs/todo.md 1.7, plus the three the probe for it turned up.
// Every expected value was read off `go run` before it was written down.
//
// WHAT EACH GROUP IS PINNING, because a passing assertion says nothing about why:
//   - the zero value of a SLICE type compares equal to nil. It is still a usable
//     empty HEADER (len/cap/range/append all work on it, as they do in Go), so
//     nil-ness is a __nil MARK on the header - exactly the trade a nil MAP already
//     makes. `[]int{}`, make() and an appended header must NOT be nil, which is
//     what tells a mark from a blanket "an empty slice is nil".
//   - a function whose panic was RECOVERED answers the ZERO of its declared result
//     types, not `<nil>`; and a NAMED result starts at the zero of its own type,
//     not at the number 0. Both needed the result type texts, which RetTypes did
//     not capture at all before.
//   - integer divide by zero is a RECOVERABLE panic in every engine. It used to
//     abort the interpreter (interp-core's `fail`) and abort a native binary (the
//     floor's si_arith calls die(), which no js_gotry can catch) while llvm.Run
//     recovered it - a three-way split.
//   - `for range x` with no variables at all parses in both halves.
//
// A sixth gap, the DOUBLED PREFIX of an unrecovered panic in a native binary, is
// deliberately not asserted here: a ratchet has to exit 0, so it cannot let a
// panic escape. tests/probe.sh with the three legs is what settles that one.
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
	// The recovered VALUE is a runtime-error MESSAGE and the same text in all three
	// engines. Only its TAIL is asserted: this project prefixes every runtime error
	// with "js runtime error: " where go writes "runtime error: ", a divergence
	// recorded at goRTPanic, and the file has to stay green under real `go run` too.
	t37 := divText37(d37, 0, 2)
	check("dz5", len(t37) >= 22 && t37[len(t37)-22:] == "integer divide by zero")

	// ----- a VALUE receiver gets a COPY that keeps its hidden slots -----
	// goCopyVal makes the copy Go's value semantics require, and layer 2 built it
	// through keysOf, which HIDES every __-prefixed slot - so the copy lost its
	// __class and `func (v V37) ...` reading a slice field answered
	// "len() of a object" in a NATIVE binary while llvm.Run printed the length. Only
	// clang-check / native-full / a native -exe probe can see this row at all.
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
// The six residues of docs/todo.md 1.5, plus the two the probe for it turned up
// (`append(s)` with nothing to add, and a func-typed field reached through a
// pointer). Every expected value was read off `go run` before it was written down.
//
// WHAT EACH GROUP IS PINNING, because a passing assertion says nothing about why:
//   - a NIL POINTER DEREFERENCE is a recoverable panic, and it was a THREE-WAY
//     split: llvm.Run recovered it, the interpreter aborted in its own field/store
//     paths and in the shared resolveRef, and a native binary died in the C floor's
//     get_member/set_member. The read, the WRITE, a write one level down a path and
//     a METHOD CALL are four different routes into it and each aborted somewhere.
//   - a DECLARED INTERFACE is satisfied STRUCTURALLY. Nothing recorded an
//     interface's method set, so `x.(Sp)` was false for a value that satisfies Sp,
//     `case Sp:` never fired, and the single-result `x.(Sp)` never panicked. The
//     concrete-type arms are here to pin that the change did NOT make every
//     assertion a check - only an interface target is checked.
//   - a MULTI-VALUE call that is the sole argument of a call EXPANDS. It could not,
//     because the results travelled as a bare array that is indistinguishable from
//     a Go ARRAY value; they now carry a mark, and the array cases below are what
//     tells the mark from "any array spreads".
//   - a RESULT VALUE adopts a map or slice declared type when it is nil, so
//     `return nil` from a map result prints `map[]` and not `<nil>`. A nil POINTER
//     result must still print `<nil>`, which is what separates this from "nil
//     always becomes a container".
//   - a FUNC-TYPED STRUCT FIELD is callable. Only the descriptor was consulted, so
//     the interpreter and llvm.Run both said "unknown method 'f'" while the native
//     binary answered - a live run-vs-native divergence.
//   - SLICING A NIL SLICE keeps its nilness (Go carries the nil backing pointer
//     over), while slicing an empty LITERAL does not.
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
// The six residues of docs/todo.md 1.3. Every expected value was read off
// `go run` before it was written down, and FIVE OF THE SIX WERE THE SAME SHAPE:
// all three engines agreed and all three were wrong, which is the one class
// ./test.sh --cross cannot see by construction.
//
// WHAT EACH GROUP PINS:
//   - fmt CONSULTS Error() BEFORE String(), which is fmt's own handleMethods
//     order. Error() was not consulted at all, so a value with `Error() string`
//     printed the memberwise `{missing b}` where go prints `missing b`. The
//     Stringer rows beside it pin that adding error did not displace String(),
//     and the both-methods row pins the PRECEDENCE rather than just presence.
//   - FIELD PROMOTION IS BREADTH FIRST and more than one level deep. Only the
//     directly embedded structs were searched, so a two-level `C{B{A{7}}}`
//     answered <nil> for c.v; the shadowing row pins that a shallower field
//     still wins over a deeper one of the same name.
//   - interface{}(x) AS A CONVERSION EXPRESSION. `var x interface{} = v` always
//     worked; the conversion aborted - the interpreter with `unknown name:
//     interface`, the compiler with `composite literal not implemented`. The
//     float and uint8 rows pin that it is the IDENTITY: routing it through the
//     ordinary conversion path truncated a float64 and dropped a sized box.
//   - A COMPOSITE LITERAL OF A DEFINED MAP TYPE. `type M map[string]int; M{...}`
//     ran in the interpreter and was `composite literal not implemented` in the
//     compiler, because a keyed element's key had to be an identifier and a
//     string key did not parse. The struct rows are what pin that giving the key
//     an expression form did not break `P{X: 1}`, whose key is a FIELD NAME and
//     not a variable.
//   - AN ELIDED ELEMENT OF POINTER TYPE. `[]*P{nil, {7}}` was `unknown name: *P`
//     / `composite literal element of unsupported type *P`; a pointer to a struct
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
	// or String() - and transitively so for everything under it. This was already
	// wrong for Stringer before Error() was added, in all three engines.
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
// Two residues of docs/todo.md 2.7, both of which all three engines got wrong
// together.
//   - `const size = 4; var a [size]int` was an EMPTY array in every engine, because
//     every reader of a type text understands digits only - so len(a) was 0 where go
//     says 4 and a[2] was an out-of-range panic. The literal-length rows beside it
//     pin that nothing about `[4]int` changed, and the map row pins that `[string]`
//     of a map type is not read as a length at all.
//   - `const a, b = 0.1, 0.2` did not PARSE in the compiler half - its ungrouped
//     const rule took one name and one expression where the grouped one and the
//     interpreter half both took lists.
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
// Two residues of docs/todo.md 1.5, both of which all three engines got wrong
// TOGETHER - so --cross was blind to each and only `go run` settles them.
//
//   - An IOTA-GROUP constant as an array length. `const (R = iota; G; B)` binds no
//     compile-time value at all in the old fold: a spec that mentions iota cannot be
//     bound to one constant, and a spec with no '=' has no expression to bind. So
//     `var a [B]int` was an EMPTY array (len 0 where go says 2) in all three engines,
//     while `R`, `G` and `B` themselves printed correctly - the run-time iota was
//     always right, only the compile-time value was missing. The fix keeps the fold
//     ITSELF, as a function of iota, beside the constant.
//   - A STRUCT- or ARRAY-VALUED map key. Both are Go VALUE types and `==` on them is
//     element-by-element, so `m[Pair{1, 2}]` must find an entry stored under a
//     different Pair holding the same fields; the key scan compared by identity and
//     every read, comma-ok and delete missed. The literal `map[Pair]string{{1, 2}: "p"}`
//     did not PARSE in either half either - an elided composite key was not a key form.
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
	// The values themselves were never wrong; these pin that they stay right.
	check("io1", r41 == 0 && g41 == 1 && b41 == 2 && n41 == 3)
	check("io2", x41 == 0 && y41 == 3 && z41 == 6)
	check("io3", k41a == 1 && k41b == 2 && k41c == 4 && k41d == 8)
	// ... and these are the rows that were 0 in every engine.
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
	// A constant EXPRESSION in the brackets, which the name substitution alone could
	// not read: it understood a bare NAME and left everything else as written, so
	// [b41 + 1] was the empty array in all three engines just as [b41] was.
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
	// The elided-key literal form, which neither half could parse.
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
	// A string key still compares as it always did.
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
// Two more residues of docs/todo.md 1.5, both settled by `go run`.
//
//   - `type Named map[Pair]string; Named{{1, 2}: "p"}` is legal Go and parsed in
//     NEITHER half. The literal-type spelling map[Pair]string{{1, 2}: "p"} goes through
//     MapLit, which knows the key type from the type it just parsed; the NAMED spelling
//     goes through StructLit, whose key production took an Expression only - so an
//     elided key was a parse failure and the native build failed with it. Both key
//     productions take a brace group now, built against the map's declared key type.
//     Nested defined types come with it: []Named{{{1, 2}: "a"}} reached the compiler
//     half's element builder as the unresolved name "Named" and was refused.
//   - A KEYED nested struct element - map[string]P{"x": {N: 1}} - was a live halves
//     divergence: the interpreter built it and the compiler answered "a keyed nested
//     struct element is not supported". The key's TEXT was thrown away at the nested
//     element's key production, so the field name was not there to be used.
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
	// A nested brace group whose element type is a DEFINED map: the compiler half
	// refused this with "composite literal element of unsupported type named42".
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
	fmt.Println("full:", checks, "checks,", fails, "failures")
	os.Exit(fails)
}
