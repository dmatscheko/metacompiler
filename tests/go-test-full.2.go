//go:build ignore

//go:debug panicnil=0

// The directive above: the repository's go.mod says go 1.16, under which a
// toolchain recovers panic(nil) as nil; this file asserts the Go 1.21+ answer
// (a *runtime.PanicNilError), as the grammars implement it.

// Full-syntax test: Go (Go 1.22 core language) - part 2 of 4, sections 83 to 93.
//
// THE GO RATCHET IS FOUR FILES, so that no -frozen interpreter leg comes near the
// IR interpreter's step brake (-max-steps, 100M per call): compiling a standard
// package written in Go costs the leg that imports it.
//
//   tests/go-test-full.go     sections 01 to 82
//   tests/go-test-full.2.go   sections 83 to 93
//   tests/go-test-full.3.go   sections 94 on, and every newer section, but
//   tests/go-test-full.4.go   sections 100 and 101 (encoding/json, reflect)
//
// The section NUMBERS are global, and ./test.sh --full reports the files as the
// units go, go2, go3 and go4.
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
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"iter"
	"maps"
	"math"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"unicode"
	"unicode/utf8"
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

// ===== SECTION 83: package sort, and what it leans on =====
// sort is Go source (languages/lib/go-sort.go) running Go's own algorithms, so the
// ORDER OF EQUAL ELEMENTS after Sort and Slice is go's: past twelve elements
// pdqsort is not stable. The expected values are go run's.
type R83 struct{ k, id int }

type ByK83 []R83

func (a ByK83) Len() int           { return len(a) }
func (a ByK83) Less(i, j int) bool { return a[i].k < a[j].k }
func (a ByK83) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

func ids83(a []R83) string {
	s := ""
	for _, r := range a {
		s += fmt.Sprint(r.id)
		s += " "
	}
	return s
}

func mk83() []R83 {
	ks := []int{3, 1, 2, 1, 3, 0, 2, 1, 3, 0, 1, 2, 3, 0, 1, 2, 2, 1, 0, 3}
	out := []R83{}
	for i, k := range ks {
		out = append(out, R83{k, i})
	}
	return out
}

type IS83 []int

func (a IS83) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

type Sw83 interface{ Swap(i, j int) }

// A struct FIELD named like a method some defined type declares.
type LS83 struct{ Swap func(i, j int) }

type Rev83 struct{ Sw83 }

// A := naming a result of the function assigns it: the body is the results' block.
func nr83(n int) (i int) {
	i, j := 5, n
	i += j
	return
}

func s83() {
	a := mk83()
	sort.Slice(a, func(i, j int) bool { return a[i].k < a[j].k })
	check("srt1", ids83(a) == "5 18 13 9 10 14 1 7 17 3 11 2 15 16 6 4 12 0 8 19 ")
	b := mk83()
	sort.SliceStable(b, func(i, j int) bool { return b[i].k < b[j].k })
	check("srt2", ids83(b) == "5 9 13 18 1 3 7 10 14 17 2 6 11 15 16 0 4 8 12 19 ")
	c := mk83()
	sort.Sort(sort.Reverse(ByK83(c)))
	check("srt3", ids83(c) == "0 19 4 12 8 2 16 15 6 11 10 7 14 3 17 1 9 13 5 18 " && sort.IsSorted(sort.Reverse(ByK83(c))))
	xs := []int{5, -2, 9, 0, 5, 3}
	fs := []float64{2.5, math.NaN(), -1, math.Inf(1), 0}
	ss := []string{"pear", "Apple", "fig", "apple"}
	sort.Ints(xs)
	sort.Float64s(fs)
	sort.Strings(ss)
	check("srt4", fmt.Sprint(xs, fs, ss, sort.SearchInts(xs, 4), sort.SearchStrings(ss, "fig"), sort.IntsAreSorted(xs)) ==
		"[-2 0 3 5 5 9] [NaN -1 0 2.5 +Inf] [Apple apple fig pear] 3 2 true")
	is := sort.IntSlice{3, 1, 2}
	is.Sort()
	i, found := sort.Find(len(xs), func(i int) int { return 5 - xs[i] })
	check("srt5", fmt.Sprint(is, is.Search(2), sort.Search(100, func(i int) bool { return i*i >= 50 }), i, found) == "[1 2 3] 1 8 3 true")
	// Converting a slice to a defined slice type shares its backing array; a func
	// field is called even where a defined type has a method of that name; a
	// method promoted from an embedded interface is a method value too.
	ys := []int{1, 2, 3}
	IS83(ys).Swap(0, 2)
	var sw Sw83 = IS83(ys)
	l := LS83{sw.Swap}
	l.Swap(0, 1)
	r := &Rev83{IS83(ys)}
	f := r.Swap
	f(1, 2)
	check("srt6", fmt.Sprint(ys, nr83(3)) == "[2 1 3] 8")
}

// ===== SECTION 84: packages slices, maps, cmp and iter; range over a function =====
// All four are Go source (languages/lib/go-*.go). A range over a function (Go
// 1.23) runs its body once per yield; break, continue, return and defer in that
// body mean what they mean in any loop. The expected values are go run's.
func count84(n int) iter.Seq[int] {
	return func(yield func(int) bool) {
		for i := 0; i < n; i++ {
			if !yield(i) {
				return
			}
		}
	}
}

func pairs84(xs []string) func(func(int, string) bool) {
	return func(yield func(int, string) bool) {
		for i, s := range xs {
			if !yield(i, s) {
				return
			}
		}
	}
}

func first84(min int) (r int) {
	for i := range count84(100) {
		if i*i >= min {
			r = i
			return
		}
	}
	return -1
}

func defers84() (out string) {
	for i := range count84(3) {
		defer func() { out += fmt.Sprint(i) }()
	}
	return "d"
}

func s84() {
	ys := []int{3, 1, 2, 5, 4}
	slices.Sort(ys)
	zs := slices.Clone(ys)
	slices.Reverse(zs)
	at, found := slices.BinarySearch(ys, 4)
	check("slc1", fmt.Sprint(ys, zs, slices.Contains(ys, 2), slices.Index(ys, 9), slices.Max(ys), slices.Min(zs),
		slices.Equal(ys, zs), slices.Compare(ys, zs), at, found) == "[1 2 3 4 5] [5 4 3 2 1] true -1 5 1 false -1 3 true")
	rs := mk83()
	slices.SortFunc(rs, func(a, b R83) int { return cmp.Compare(a.k, b.k) })
	ss := mk83()
	slices.SortStableFunc(ss, func(a, b R83) int { return cmp.Compare(a.k, b.k) })
	check("slc2", ids83(rs) == "5 18 13 9 10 14 1 7 17 3 11 2 15 16 6 4 12 0 8 19 " &&
		ids83(ss) == "5 9 13 18 1 3 7 10 14 17 2 6 11 15 16 0 4 8 12 19 ")
	orig := []int{1, 1, 2, 2, 3}
	c := slices.Compact(orig)
	s := slices.Insert([]int{1, 2, 5}, 2, 3, 4)
	check("slc3", fmt.Sprint(c, orig, s, slices.Delete(s, 1, 3), slices.DeleteFunc([]int{1, 2, 3, 4}, func(x int) bool { return x%2 == 0 }),
		slices.Replace([]int{0, 1, 2, 3}, 1, 3, 9), slices.Concat([]int{1}, nil, []int{2}), slices.Repeat([]string{"a"}, 3), slices.Clone([]int(nil)) == nil) ==
		"[1 2 3] [1 2 3 0 0] [1 4 5 0 0] [1 4 5] [1 3] [0 9 3] [1 2] [a a a] true")
	m := map[string]int{"b": 2, "a": 1}
	m2 := maps.Clone(m)
	m2["c"] = 3
	maps.DeleteFunc(m2, func(k string, v int) bool { return v == 1 })
	check("slc4", fmt.Sprint(cmp.Compare(2, 1), cmp.Less(math.NaN(), 0.0), cmp.Or("", "x", "y"), maps.Equal(m, m2), m2) ==
		"1 truexfalse map[b:2 c:3]")
	got := ""
outer:
	for i := range count84(4) {
		for j, s := range pairs84([]string{"a", "b", "c"}) {
			if j == 1 {
				continue
			}
			if i == 1 {
				continue outer
			}
			if i == 3 {
				break outer
			}
			got += fmt.Sprint(i, s, " ")
		}
	}
	check("itr1", fmt.Sprint(got, first84(10), defers84()) == "0a 0c 2a 2c 4d210")
	keys := slices.Sorted(maps.Keys(map[string]bool{"z": true, "a": true, "m": false}))
	chunks := ""
	for ch := range slices.Chunk([]int{1, 2, 3, 4, 5}, 2) {
		chunks += fmt.Sprint(ch)
	}
	back := ""
	for i, v := range slices.Backward([]string{"x", "y"}) {
		back += fmt.Sprint(i, v)
	}
	check("itr2", fmt.Sprint(keys, slices.Collect(count84(3)), chunks, back, len(maps.Collect(maps.All(m)))) ==
		"[a m z] [0 1 2][1 2][3 4][5]1y0x2")
}

// ===== SECTION 85: pointer methods - a nil receiver, a generic type, &T{} =====
// Go calls a pointer method on a nil receiver like any other; only a dereference
// inside would panic. A method of a generic type is a method of every
// instantiation, and &T{} of a slice type points at a variable the method can
// grow. The expected values are go run's.
type Node85 struct {
	V    int
	Next *Node85
}

func (n *Node85) Len() int {
	if n == nil {
		return 0
	}
	return 1 + n.Next.Len()
}

type Tree85 struct {
	L, R *Tree85
	V    int
}

func (t *Tree85) Insert(v int) *Tree85 {
	if t == nil {
		return &Tree85{V: v}
	}
	if v < t.V {
		t.L = t.L.Insert(v)
	} else {
		t.R = t.R.Insert(v)
	}
	return t
}

func (t *Tree85) InOrder(out []int) []int {
	if nil != t {
		out = t.L.InOrder(out)
		out = append(out, t.V)
		out = t.R.InOrder(out)
	}
	return out
}

type Stack85[T any] []T

func (s *Stack85[T]) Push(v T) { *s = append(*s, v) }
func (s *Stack85[T]) Pop() T {
	old := *s
	v := old[len(old)-1]
	*s = old[:len(old)-1]
	return v
}
func (s Stack85[T]) Peek() T { return s[len(s)-1] }

type Ints85 []int

func (s *Ints85) Push(v int) { *s = append(*s, v) }

func s85() {
	var e *Node85
	l := &Node85{1, &Node85{2, nil}}
	var t *Tree85
	for _, v := range []int{5, 3, 8, 1, 4} {
		t = t.Insert(v)
	}
	check("nrc1", fmt.Sprint(e.Len(), l.Len(), t.InOrder(nil)) == "0 2 [1 3 4 5 8]")
	var st Stack85[string]
	st.Push("a")
	st.Push("b")
	top := st.Peek()
	popped := st.Pop()
	ps := &Ints85{}
	ps.Push(7)
	ps.Push(8)
	check("gen1", fmt.Sprint(top, popped, len(st), ps, len(*ps)) == "bb1 &[7 8] 2")
}

// ===== SECTION 86: packages sync and sync/atomic =====
// Go source (languages/lib/go-sync.go, go-atomic.go) over the cooperative
// goroutines: WaitGroup.Wait runs the queued goroutines, each in the scope of its
// `go` statement. The expected values are go run's.
type Ctr86 struct {
	mu sync.Mutex
	m  map[string]int
}

func (c *Ctr86) Inc(k string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[k]++
}

func work86(id int, jobs <-chan int, out chan<- string, wg *sync.WaitGroup) {
	defer wg.Done()
	for j := range jobs {
		out <- fmt.Sprint(id, ":", j*j)
	}
}

func s86() {
	var n int64
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			atomic.AddInt64(&n, int64(i))
		}()
	}
	wg.Wait()
	c := Ctr86{m: map[string]int{}}
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Inc("k")
		}()
	}
	wg.Wait()
	var a atomic.Int64
	a.Add(5)
	var once sync.Once
	inits := 0
	for i := 0; i < 3; i++ {
		once.Do(func() { inits++ })
	}
	check("syn1", fmt.Sprint(atomic.LoadInt64(&n), c.m, a.Load(), a.CompareAndSwap(5, 1), a.Load(), inits) == "45 map[k:3] 5 true 1 1")
	jobs := make(chan int, 4)
	out := make(chan string, 4)
	wg2 := &sync.WaitGroup{}
	wg2.Add(1)
	go work86(7, jobs, out, wg2)
	for j := 1; j <= 3; j++ {
		jobs <- j
	}
	close(jobs)
	wg2.Wait()
	close(out)
	got := ""
	for s := range out {
		got += s + " "
	}
	check("syn2", got == "7:1 7:4 7:9 ")
}

// ===== SECTION 87: package strconv; asserting a pointer type =====
// strconv is Go source (languages/lib/go-strconv.go): Go's own parsing, fmt's
// formats. A *T in a type assertion, a type switch or errors.As tests the struct
// value (a pointer and the struct are one value here), another package's type
// included. The expected values are go run's.
type Err87 struct{ code int }

func (e *Err87) Error() string { return "err87" }

func fail87() error { return &Err87{7} }

type Row87 struct {
	Name string
	Tags []string
	M    map[string]int
}

// `n := 3` in a block shadows the named result n; `return ..., n` sets the
// result, not the shadow.
func shadow87() (f float64, n int) {
	if true {
		n := 3
		return 1.5, n
	}
	return
}

func len87(s string) int {
	i := 0
	for ; i < len(s); i++ {
	}
	if i > 0 {
		return (&Row87{Name: s}).Name2()
	}
	return 0
}

func (r *Row87) Name2() int { return len(r.Name) * 2 }

func s87() {
	n, err := strconv.Atoi("123")
	_, err2 := strconv.Atoi("12a")
	i64, err3 := strconv.ParseInt("-0x1f", 0, 64)
	_, err4 := strconv.ParseInt("300", 10, 8)
	u, _ := strconv.ParseUint("18446744073709551615", 10, 64)
	check("scv1", fmt.Sprint(n, err, err2, i64, err3, err4, u) ==
		`123 <nil> strconv.Atoi: parsing "12a": invalid syntax -31 <nil> strconv.ParseInt: parsing "300": value out of range 18446744073709551615`)
	f1, _ := strconv.ParseFloat("0.1", 64)
	f2, _ := strconv.ParseFloat("1e23", 64)
	f3, _ := strconv.ParseFloat("2.2250738585072011e-308", 64)
	f4, e4 := strconv.ParseFloat("1e400", 64)
	f5, _ := strconv.ParseFloat("3.4028235677973366e38", 32)
	check("scv2", fmt.Sprint(f1, f2, f3, f4, e4, f5) == `0.1 1e+23 2.225073858507201e-308 +Inf strconv.ParseFloat: parsing "1e400": value out of range 3.4028234663852886e+38`)
	check("scv3", fmt.Sprint(strconv.Itoa(-45), strconv.FormatInt(-255, 16), strconv.FormatFloat(1234.5678, 'e', -1, 64),
		strconv.FormatFloat(0.000012345, 'f', -1, 64), strconv.FormatFloat(2.5, 'g', 3, 32), strconv.Quote("a\"é\n"), strconv.FormatBool(true)) ==
		`-45-ff1.2345678e+030.0000123452.5"a\"é\n"true`)
	uq, _ := strconv.Unquote(`"é\x41\101"`)
	b, _ := strconv.ParseBool("TRUE")
	check("scv4", fmt.Sprint(uq, b, strings.ToTitle("ǆa")) == "éAAtrueǅA")
	var ne *strconv.NumError
	ok1 := errors.As(err2, &ne)
	ne2, ok2 := err2.(*strconv.NumError)
	e := fail87()
	m, ok3 := e.(*Err87)
	sw := ""
	switch x := e.(type) {
	case *Err87:
		sw = fmt.Sprint("ptr ", x.code)
	}
	check("ast1", fmt.Sprint(ok1, ne.Func, ok2, ne2.Num, ok3, m.code, sw) == "trueAtoitrue12atrue 7ptr 7")
	rows := []Row87{{"a", nil, nil}}
	sf, sn := shadow87()
	check("nil1", fmt.Sprint(rows, rows[0].Tags == nil, sf, sn, len87("abc")) == "[{a [] map[]}] true 1.5 3 6")
}

// ===== SECTION 88: packages unicode and unicode/utf8 =====
// Go source (languages/lib/go-unicode.go, go-utf8.go); the category tests are
// exact through Latin-1. The expected values are go run's.
func cats88(r rune) string {
	s := ""
	for _, ok := range []bool{unicode.IsUpper(r), unicode.IsLower(r), unicode.IsLetter(r), unicode.IsDigit(r),
		unicode.IsSpace(r), unicode.IsPunct(r), unicode.IsSymbol(r), unicode.IsControl(r), unicode.IsPrint(r)} {
		if ok {
			s += "1"
		} else {
			s += "0"
		}
	}
	return s
}

func s88() {
	got := ""
	for _, r := range []rune{'A', 'z', '5', ' ', '!', '$', '\n', 0xA0, 'é', 'ß', 0xD7, 0xBF, 'Σ', '世'} {
		got += cats88(r) + " "
	}
	check("uni1", got == "101000001 011000001 000100001 000010001 000001001 000000101 000010010 000010000 011000001 011000001 000000101 000001001 101000001 001000001 ")
	check("uni2", string([]rune{unicode.ToUpper('é'), unicode.ToLower('Σ'), unicode.ToTitle('ǆ'), unicode.ToUpper('q')}) == "ÉσǅQ")
	s := "héllo, 世界\xff"
	var dec []string
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		dec = append(dec, fmt.Sprint(r, ":", size))
		i += size
	}
	buf := make([]byte, 4)
	n := utf8.EncodeRune(buf, '€')
	check("utf1", fmt.Sprint(utf8.RuneCountInString(s), utf8.ValidString(s), utf8.RuneLen('世'), dec, buf[:n],
		string(utf8.AppendRune(nil, 'é'))) == "10 false 3 [104:1 233:2 108:1 108:1 111:1 44:1 32:1 19990:3 30028:3 65533:1] [226 130 172]é")
}

// ===== SECTION 89: package bytes =====
// Go source (languages/lib/go-bytes.go) over package strings. The expected
// values are go run's.
func s89() {
	var buf bytes.Buffer
	buf.WriteString("hi ")
	buf.WriteByte('w')
	buf.WriteRune('é')
	fmt.Fprintf(&buf, "%d", 42)
	line, _ := bytes.NewBufferString("a,b").ReadString(',')
	check("byt1", fmt.Sprint(buf.String(), buf.Len(), bytes.Contains([]byte("abc"), []byte("bc")), string(bytes.ToUpper([]byte("ab"))),
		bytes.Equal([]byte("x"), []byte("x")), len(bytes.Fields([]byte(" a b "))), line) == "hi wé428 trueABtrue 2a,")
}

// ===== SECTION 90: order of evaluation =====
// gc runs every call of a statement first, left to right, and reads the plain
// operands after them. The expected values are go run's.
func pair90(a, b int) int { return a*100 + b }
func s90() {
	x := 5
	f := func() int { x = 7; return 1 }
	got := fmt.Sprint(x, f())
	x = 5
	got += fmt.Sprint(" ", pair90(x, f()))
	x = 5
	y := x + f()
	x = 5
	a, b := x, f()
	x = 5
	r := func() (int, int) { return x, f() }
	r1, r2 := r()
	check("ord1", fmt.Sprint(got, " ", y, a, b, r1, r2) == "7 1 701 8 7 1 7 1")
	x = 5
	check("ord2", fmt.Sprint(x, x+f(), x == 7 && f() == 1) == "7 8 true")
	type p90 struct{ a, b int }
	x = 5
	s := []int{x, f(), x}
	x = 5
	check("ord3", fmt.Sprint(s, p90{x, f()}) == "[7 1 7] {7 1}")
}

// ===== SECTION 91: forward goto, late typed constants, range of an untyped func =====
// The expected values are go run's.
func find91(xs []int, t int) int {
	i := 0
	for i = 0; i < len(xs); i++ {
		if xs[i] == t {
			goto found
		}
	}
	return -1
found:
	return i
}

func back91() int {
	n := 0
again:
	n++
	if n < 3 {
		goto again
	}
	return n
}

func show91() string {
	return fmt.Sprintf("%v %T %v %T", Late91, Late91, Kind91(2), Late91+1)
}

type Kind91 uint8

func (k Kind91) String() string { return fmt.Sprintf("K%d", uint8(k)) }

const Late91 Kind91 = 9

func count91(n int) func(func(int) bool) {
	return func(yield func(int) bool) {
		for i := 0; i < n; i++ {
			if !yield(i) {
				return
			}
		}
	}
}

func s91() {
	check("gto1", fmt.Sprint(find91([]int{4, 5, 6}, 5), find91(nil, 1), back91()) == "1 -1 3")
	check("lck1", show91() == "K9 main.Kind91 K2 main.Kind91")
	mk := count91
	got := ""
	for v := range mk(3) {
		got += fmt.Sprint(v)
	}
	check("rfn1", got == "012")
	check("len1", fmt.Sprintf("%v %T", len("ab")<<2, len("abc")) == "8 int")
}

// ===== SECTION 92: pointers and struct values in interfaces =====
// An interface keeps whether it holds a *T or a T. The expected values are go
// run's.
type V92 struct{ A int }

func (v *V92) Inc() int { v.A++; return v.A }

type E92 struct{ Code int }

func (e *E92) Error() string { return fmt.Sprint("code ", e.Code) }

var base92 = &E92{1}

func find92(n int) error {
	if n == 0 {
		return base92
	}
	return &E92{n}
}

func kind92(i interface{}) string {
	switch x := i.(type) {
	case *V92:
		return fmt.Sprint("ptr", x.A)
	case V92:
		return fmt.Sprint("val", x.A)
	}
	return "other"
}

func s92() {
	var j interface{} = &V92{2}
	_, okP := j.(*V92)
	_, okV := j.(V92)
	j.(*V92).Inc()
	var k interface{} = V92{4}
	_, kP := k.(*V92)
	check("ifp1", fmt.Sprintf("%v %v %T %v %v %T %v", okP, okV, j, j, kP, k, k) == "true false *main.V92 &{3} false main.V92 {4}")
	_, err := strconv.Atoi("x")
	e0 := find92(0)
	var e3, e4 error = &E92{9}, &E92{9}
	var target *E92
	check("ifp2", fmt.Sprintf("%T|%v %v %v|%v %v", err, e0 == base92, errors.Is(e0, base92), e3 == e4,
		errors.As(fmt.Errorf("w: %w", e3), &target), target.Code) == "*strconv.NumError|true true false|true 9")
	var np *V92
	var ni interface{} = np
	check("ifp3", fmt.Sprintf("%v %v %v %v %v", kind92(&V92{7}), kind92(V92{8}), ni == nil, &map[string]int{"a": 1}, &[]int{1}) ==
		"ptr7 val8 false &map[a:1] &[1]")
}

// ===== SECTION 93: generic instantiation: zero values, %T, new(T) =====
// A generic function or type knows its type arguments at run time. The expected
// values are go run's.
type Pair93[K comparable, V any] struct {
	Key K
	Val V
}

func mkPair93[K comparable, V any](k K, v V) Pair93[K, V] { return Pair93[K, V]{k, v} }

type Stack93[T any] struct{ xs []T }

func (s *Stack93[T]) Pop() (T, bool) {
	var zero T
	if len(s.xs) == 0 {
		return zero, false
	}
	v := s.xs[len(s.xs)-1]
	s.xs = s.xs[:len(s.xs)-1]
	return v, true
}

type Opt93[T any] struct {
	v  T
	ok bool
}

func none93[T any]() Opt93[T] { return Opt93[T]{} }

func zero93[T any]() (z T) { return }

func fill93[T any](n int, x T) []T {
	s := make([]T, n)
	s[0] = x
	return s
}

func map93[T, U any](xs []T, f func(T) U) []U {
	out := make([]U, 0, len(xs))
	for _, x := range xs {
		out = append(out, f(x))
	}
	return out
}

type P93 struct{ X int }

func (p *P93) Move() { p.X++ }

func s93() {
	var s Stack93[string]
	v, ok := s.Pop()
	si := &Stack93[int]{}
	n, _ := si.Pop()
	check("gen1", fmt.Sprintf("%q %v %v %T %T", v, ok, n, s, si) == `"" false 0 main.Stack93[string] *main.Stack93[int]`)
	check("gen2", fmt.Sprintf("%T %v %T %v", mkPair93("a", 1.5), none93[string](), map93([]int{1}, strconv.Itoa), fill93(2, 7)) ==
		"main.Pair93[string,float64] { false} []string [7 0]")
	check("gen3", fmt.Sprintf("%q %v %v", zero93[string](), zero93[int](), zero93[bool]()) == `"" 0 false`)
	p := new(P93)
	p.Move()
	check("new1", fmt.Sprintf("%T %v", p, p) == "*main.P93 &{1}")
}

func main() {
	s83() // SECTION-CALL 83
	s84() // SECTION-CALL 84
	s85() // SECTION-CALL 85
	s86() // SECTION-CALL 86
	s87() // SECTION-CALL 87
	s88() // SECTION-CALL 88
	s89() // SECTION-CALL 89
	s90() // SECTION-CALL 90
	s91() // SECTION-CALL 91
	s92() // SECTION-CALL 92
	s93() // SECTION-CALL 93
	fmt.Println("full:", checks, "checks,", fails, "failures")
	os.Exit(fails)
}
