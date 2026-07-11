//go:build ignore

// Go subset completion self test.
//
// Exercises the range forms and receiver forms this pass completed:
//   - for i, ch := range s  over a STRING: i is the index, ch is the i-th
//     one-character substring (this subset yields the character, deterministic
//     across the interpreter and the compiler).
//   - for _, ch := range s  the blank index form over a string.
//   - for i := range slice  the single-variable (index only) range over a slice.
//   - for i, v := range slice  the index+element form.
//   - for k, v := range m  and  for k := range m  over a MAP, keys in insertion
//     order.
//   - struct methods with POINTER receivers (func (c *Counter) ...) that mutate
//     the receiver in place, mixed with VALUE receivers on the same type.
//   - a value-receiver method returning MULTIPLE values.
//
// main() counts failed checks and ends with os.Exit(fails), so the run exits 0
// exactly when every check holds.

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

// A type carrying both a pointer-receiver mutator and value-receiver readers.
type Counter struct {
	n int
}

func (c *Counter) inc()        { c.n++ }      // pointer receiver: mutates in place
func (c *Counter) add(d int)   { c.n = c.n + d }
func (c Counter) get() int     { return c.n } // value receiver: reads
func (c Counter) doubled() int { return c.n * 2 }

// A value-receiver method returning multiple values.
type Rect struct {
	w int
	h int
}

func (r Rect) dims() (int, int) { return r.w, r.h }
func (r Rect) area() int        { return r.w * r.h }

func main() {
	// --- range over a string: index + one-character substring ---
	rebuilt := ""
	isum := 0
	n := 0
	for i, ch := range "hello" {
		rebuilt += ch
		isum += i
		n++
	}
	checkS("string range chars", rebuilt, "hello")
	check("string range index sum", isum, 10)
	check("string range count", n, 5)

	// blank index over a string
	word := ""
	for _, ch := range "go" {
		word += ch
	}
	checkS("string range blank index", word, "go")

	// a specific character position, and character comparison
	third := ""
	for i, ch := range "abcd" {
		if i == 2 {
			third = ch
		}
	}
	checkS("string range nth char", third, "c")
	hits := 0
	for _, ch := range "banana" {
		if ch == "a" {
			hits++
		}
	}
	check("string range char equals", hits, 3)

	// --- single-variable range over a slice (indices only) ---
	sl := []int{10, 20, 30, 40}
	idxSum := 0
	for i := range sl {
		idxSum += i
	}
	check("slice range index only", idxSum, 6)

	// index + element form
	both := 0
	for i, v := range sl {
		both += i + v
	}
	check("slice range index+value", both, 106)

	// --- range over a map: values, and keys only ---
	ages := map[string]int{"al": 1, "bo": 2, "cy": 3}
	vsum := 0
	for _, v := range ages {
		vsum += v
	}
	check("map range values", vsum, 6)
	keyChars := 0
	seen := 0
	for k := range ages {
		keyChars += len(k)
		seen++
	}
	check("map range keys only count", seen, 3)
	check("map range keys only chars", keyChars, 6)

	// --- pointer receivers mutate; value receivers read ---
	c := Counter{0}
	c.inc()
	c.inc()
	c.inc()
	check("pointer receiver mutates", c.get(), 3)
	c.add(10)
	check("pointer receiver with arg", c.get(), 13)
	check("value receiver reads", c.doubled(), 26)

	// --- value-receiver method returning multiple values ---
	r := Rect{3, 4}
	w, h := r.dims()
	check("multi-return dims w", w, 3)
	check("multi-return dims h", h, 4)
	check("value receiver area", r.area(), 12)

	if fails == 0 {
		fmt.Println("Go completion self test passed")
	}
	os.Exit(fails)
}
