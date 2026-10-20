//go:build ignore

//go:debug panicnil=0

// The directive above: the repository's go.mod says go 1.16, under which a
// toolchain recovers panic(nil) as nil; this file asserts the Go 1.21+ answer
// (a *runtime.PanicNilError), as the grammars implement it.

// Full-syntax test: Go (Go 1.22 core language) - part 3 of 4, sections 94 on but
// 100 and 101; tests/go-test-full.2.go says why the ratchet is four files. Add a
// new section to this file.
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
	"bufio"
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
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

// ===== SECTION 94: packages io, bufio and time; strings.Reader, os.Args =====
// Go source (languages/lib/go-io.go, go-bufio.go, go-time.go) over an io.Reader.
// The expected values are go run's.
type D94 int64

const (
	ns94 D94 = 1
	us94     = 1000 * ns94
)

func (d D94) String() string { return fmt.Sprint(int64(d), "u") }

func s94() {
	sc := bufio.NewScanner(strings.NewReader("one two\nthree\n\nfour"))
	got := ""
	for sc.Scan() {
		got += "[" + sc.Text() + "]"
	}
	ws := bufio.NewScanner(strings.NewReader("  a b\tc\n d "))
	ws.Split(bufio.ScanWords)
	nw := 0
	for ws.Scan() {
		nw++
	}
	r := bufio.NewReader(strings.NewReader("x,y"))
	t1, _ := r.ReadString(',')
	t2, e2 := r.ReadString(',')
	all, _ := io.ReadAll(strings.NewReader("all"))
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	fmt.Fprintf(w, "%d", 7)
	before := buf.Len()
	w.Flush()
	check("bio1", fmt.Sprint(got, nw, t1, t2, e2 == io.EOF, string(all), before, buf.String(), sc.Err()) ==
		"[one two][three][][four]4x,ytrueall07<nil>")
	d := 90*time.Minute + 30*time.Second + 5*time.Millisecond
	t := time.Date(2024, time.February, 29, 13, 4, 5, 123000000, time.UTC)
	pd, _ := time.ParseDuration("1h15m30.5s")
	check("tim1", fmt.Sprint(d, " ", time.Duration(1500)*time.Microsecond, " ", d.Truncate(time.Hour), " ", pd) == "1h30m30.005s 1.5ms 1h0m0s 1h15m30.5s")
	check("tim2", fmt.Sprint(t, "|", t.Format(time.RFC3339), "|", t.Format(time.Kitchen), "|", t.Month(), t.Weekday(), t.YearDay(), t.Unix()) ==
		"2024-02-29 13:04:05.123 +0000 UTC|2024-02-29T13:04:05Z|1:04PM|February Thursday 60 1709211845")
	t2b := t.Add(36 * time.Hour)
	check("tim3", fmt.Sprint(t2b.Format(time.DateTime), " ", t2b.Sub(t), " ", t.AddDate(0, 1, 0).Format(time.DateOnly), " ", fmt.Sprintf("%T", time.Hour)) ==
		"2024-03-02 01:04:05 36h0m0s 2024-03-29 time.Duration")
	check("cty1", fmt.Sprintf("%T %v %v", us94, us94, 3*us94) == "main.D94 1000u 3000u")
	check("osa1", len(os.Args) == 1)
}

// ===== SECTION 95: the capacity append grows a slice to =====
// gc's runtime.growslice: doubling, then +(n+768)/4, rounded up to the malloc size
// class of the bytes. The slices escape (sink95), so gc puts no first backing array
// on the stack. The expected values are go run's.
var sink95 []interface{}

type pt95 struct{ a, b, c int }

func caps95[T any](n int) string {
	var s []T
	var z T
	out := ""
	last := -1
	for i := 0; i < n; i++ {
		s = append(s, z)
		sink95 = append(sink95[:0], s)
		if cap(s) != last {
			last = cap(s)
			out += fmt.Sprint(last, " ")
		}
	}
	return out
}

func s95() {
	check("cap1", caps95[int](1300) == "1 2 4 8 16 32 64 128 256 512 848 1280 1792 ")
	check("cap2", caps95[byte](600) == "8 16 32 64 128 256 512 896 ")
	check("cap3", caps95[string](300) == "1 2 4 8 16 32 71 143 303 ")
	check("cap4", caps95[pt95](100) == "1 2 4 8 16 32 64 128 ")
	a := []int{1, 2, 3, 4}
	b := a[1:2:2]
	b = append(b, 9)
	sink95 = append(sink95[:0], b)
	check("cap5", cap(b) == 2)
}

// ===== SECTION 96: packages path, path/filepath, flag and log; *p.f = v =====
// Go source (languages/lib/go-path.go, go-filepath.go, go-flag.go, go-log.go). The
// expected values are go run's.
type ref96 struct {
	ip *int
	xs []*int
}

func s96() {
	check("pth1", fmt.Sprint(path.Join("a", "b/../c", "/d"), " ", path.Base("/x/y.txt"), " ", path.Ext("f.tar.gz"), " ",
		path.Dir("/a/b/c"), " ", path.Clean("a//b/./../c"), " ", filepath.Join("dir", "f.go")) == "a/c/d y.txt .gz /a/b a/c dir/f.go")
	fs := flag.NewFlagSet("sub", flag.ContinueOnError)
	k := fs.Int("k", 1, "k")
	v := fs.Bool("v", false, "v")
	name := fs.String("name", "x", "who")
	err := fs.Parse([]string{"-k", "42", "-v", "--name=bob", "rest", "more"})
	check("flg1", fmt.Sprintf("%v %v %v %v %v %v", *k, *v, *name, err, fs.Args(), fs.NFlag()) == "42 true bob <nil> [rest more] 3")
	var sb strings.Builder
	l := log.New(&sb, "[app] ", 0)
	l.Printf("started %d", 3)
	l.Println("done", true)
	check("log1", sb.String() == "[app] started 3\n[app] done true\n")
	a, b := 1, 2
	r := &ref96{&a, []*int{&b}}
	*r.ip += 10
	*r.xs[0] = 20
	check("pfw1", a == 11 && b == 20)
}

// ===== SECTION 97: package context; a select over a nil channel =====
// Go source (languages/lib/go-context.go). The expected values are go run's.
type key97 string

func worker97(ctx context.Context, out chan<- string) {
	select {
	case <-ctx.Done():
		out <- "stopped: " + ctx.Err().Error()
	}
}

func s97() {
	bg := context.Background()
	ctx, cancel := context.WithCancel(bg)
	child, cancel2 := context.WithCancel(ctx)
	defer cancel2()
	vctx := context.WithValue(child, key97("user"), "ann")
	before := fmt.Sprint(ctx.Err(), vctx.Value(key97("user")), vctx.Value(key97("none")), vctx.Value("user"))
	out := make(chan string, 1)
	cancel()
	go worker97(vctx, out)
	got := <-out
	idle := "busy"
	select {
	case <-bg.Done():
	default:
		idle = "idle"
	}
	cc, ccancel := context.WithCancelCause(bg)
	ccancel(errors.New("boom"))
	check("ctx1", fmt.Sprintf("%s|%s|%v|%s|%v %v|%v", before, got, child.Err() == context.Canceled, idle, cc.Err(), context.Cause(cc), bg) ==
		"<nil>ann<nil> <nil>|stopped: context canceled|true|idle|context canceled boom|context.Background")
}

// ===== SECTION 98: package regexp =====
// Go source (languages/lib/go-regexp.go) over lib/regex.js's R dialect: RE2
// syntax, Go's error texts, leftmost-first matching with Go's empty-iteration
// rule, byte offsets, FindAll's and ReplaceAll's empty-match steps, $ templates.
// The expected values are go run's.
func s98() {
	pats := []string{`(a*)*`, `(|a)+`, `()*`, `(?:(a)|b)+`, `(?U)a+`, `(?i)k+`, `(?m)^\w`, `\pL+`, `\p{Greek}+`, `[[:^alpha:]]+`, `\Qa.b\E+`, `(?s).$`, `\bx\b`}
	texts := []string{"aaa", "aa", "x", "ab", "aaa", "\u212akK", "a\nb", "1héllo2", "abγδε", "ab12cd", "a.bbb", "a\n", "x ax x"}
	out := ""
	for i, p := range pats {
		out += fmt.Sprint(regexp.MustCompile(p).FindAllStringSubmatchIndex(texts[i], -1), ";")
	}
	check("rx1", out == "[[0 3 0 3]];[[0 0 0 0] [1 1 1 1] [2 2 2 2]];[[0 0 0 0] [1 1 1 1]];[[0 2 0 1]];[[0 1] [1 2] [2 3]];[[0 5]];[[0 1] [2 3]];[[1 7]];[[2 8]];[[2 4]];[[0 5]];[[1 2]];[[0 1] [5 6]];")
	re := regexp.MustCompile(`(?P<key>\w+)=(?P<val>\w*)`)
	src := "a=1, bb=22, c=, é=x"
	check("rx2", fmt.Sprint(re.ReplaceAllString(src, "$val:$key"), "|", re.ReplaceAllString(src, "${key}x$1x$$ $9"), "|",
		re.SubexpNames(), re.SubexpIndex("val"), re.NumSubexp(), "|", re.FindAllString(src, 2), "|", re.ReplaceAllStringFunc(src, strings.ToUpper)) ==
		"1:a, 22:bb, :c, é=x|ax$ , bbx$ , cx$ , é=x|[ key val] 2 2|[a=1 bb=22]|A=1, BB=22, C=, é=x")
	a := regexp.MustCompile(`a*`)
	check("rx3", fmt.Sprint(a.Split("abaabaccadaaae", 5), len(a.Split("abaabaccadaaae", -1)), a.ReplaceAllString("baaac", "X"),
		regexp.MustCompile(`x*`).FindAllStringIndex("éxé\U0001F600", -1), regexp.MustCompile(`.`).FindAllStringIndex("a\xff\xc3é", -1),
		regexp.QuoteMeta(`1.5+[x]`)) == "[ b b c cadaaae] 7XbXcX[[0 0] [2 3] [5 5] [9 9]] [[0 1] [1 2] [2 3] [3 5]]1\\.5\\+\\[x\\]")
	errs := ""
	for _, p := range []string{`a**`, `(?=a)`, `(?<=a)`, `\1`, `a{1001}`, `[z-a]`, `(`, `x)`, `[a`, `\p{Foo}`, `[[:foo:]]`, `(?z)`, `(a{2}){501}`, `\x{110000}`, `a\`} {
		_, err := regexp.Compile(p)
		errs += err.Error() + "\n"
	}
	check("rx4", errs == "error parsing regexp: invalid nested repetition operator: `**`\nerror parsing regexp: invalid or unsupported Perl syntax: `(?=`\nerror parsing regexp: invalid named capture: `(?<=a)`\nerror parsing regexp: invalid escape sequence: `\\1`\nerror parsing regexp: invalid repeat count: `{1001}`\nerror parsing regexp: invalid character class range: `z-a`\nerror parsing regexp: missing closing ): `(`\nerror parsing regexp: unexpected ): `x)`\nerror parsing regexp: missing closing ]: `[a`\nerror parsing regexp: invalid character class range: `\\p{Foo}`\nerror parsing regexp: invalid character class range: `[:foo:]`\nerror parsing regexp: invalid or unsupported Perl syntax: `(?z`\nerror parsing regexp: invalid repeat count: `{501}`\nerror parsing regexp: invalid escape sequence: `\\x{110000`\nerror parsing regexp: trailing backslash at end of expression: ``\n")
	defer func() {
		check("rx5", fmt.Sprint(recover()) == "regexp: Compile(`a\tb(`): error parsing regexp: missing closing ): `a\tb(`")
	}()
	regexp.MustCompile("a\tb(")
}

// ===== SECTION 99: os.Stdin =====
// Nothing here reads standard input, which the harness leaves to its caller:
// reading it is probed with input fed through a pipe. A Read into no room
// answers (0, nil) without reading. The expected values are go run's.
func s99() {
	var r io.Reader = os.Stdin
	k, err := r.Read(make([]byte, 0))
	check("stdin1", k == 0 && err == nil)
}

// ===== SECTION 102: a pointer to a sized integer, a float or a complex variable =====
// &x of such a variable is a cell, so a write through it reaches x. The expected
// values are go run's.
func set102(p *float64, q *int8, r *uint64) {
	*p = 2.5
	*q = 7
	*r = 9
}

func s102() {
	var f float64
	var i int8
	var u uint64
	var c complex128
	set102(&f, &i, &u)
	pc := &c
	*pc = 1 + 2i
	check("ptr102", fmt.Sprint(f, i, u, c) == "2.5 7 9 (1+2i)")
}

// ===== SECTION 103: a package function named like another package's method =====
// math.Abs and math.Round are the package's functions, though time.Duration
// declares methods Abs and Round. The expected values are go run's.
func s103() {
	check("pkgfn103", fmt.Sprint(math.Abs(-2.5), math.Round(2.5), math.Abs(1e21) >= 1e21, time.Duration(-90).Abs()) == "2.5 3 true 90ns")
}

// ===== SECTION 104: only an EMBEDDED field's methods are the struct's =====
// A named field of a type with a method does not give the struct that method;
// an embedded one does. The expected values are go run's.
type mj104 interface{ MJ() string }

type p104 struct{ X int }

func (p p104) MJ() string { return "p" }

type named104 struct{ Q p104 }

type embed104 struct{ p104 }

func s104() {
	var a interface{} = named104{p104{1}}
	var b interface{} = embed104{p104{1}}
	_, okA := a.(mj104)
	mb, okB := b.(mj104)
	check("embed104", !okA && okB && mb.MJ() == "p")
}

// ===== SECTION 105: `*p = v` through a pointer to a struct =====
// The store copies v into the struct p points at, so every other pointer to it
// sees the new value, and a store through a nil pointer panics. The expected
// values are go run's.
type in105 struct{ v int }

type t105 struct {
	x  int
	in in105
}

func (t *t105) set(o *t105) { *t = *o }

func s105() {
	a := &t105{1, in105{1}}
	alias := a
	b := &t105{5, in105{6}}
	a.set(b)
	b.in.v = 99
	out := fmt.Sprint(alias.x, alias.in.v)
	*alias = t105{7, in105{8}}
	out += fmt.Sprint(" ", a.x, a.in.v)
	pp := &a
	**pp = t105{3, in105{4}}
	out += fmt.Sprint(" ", alias.x, alias.in.v)
	func() {
		defer func() { out += fmt.Sprint(" ", recover()) }()
		var np *t105
		*np = t105{}
	}()
	check("pstore105", out == "5 6 7 8 3 4 runtime error: invalid memory address or nil pointer dereference")
}

// ===== SECTION 106: a zero embedded struct; a promoted field written =====
// S{} holds a zero Base, so Base's methods and fields are S's; a write to a
// promoted field writes the embedded struct's. The expected values are go run's.
type base106 struct{ n int }

func (b base106) Hello() string { return fmt.Sprint("hi", b.n) }

type s106 struct {
	base106
	k int
}

func s106f() {
	s := s106{}
	out := s.Hello() + fmt.Sprint(" ", s.n, " ")
	s.n = 4
	s.n++
	s.n += 2
	out += fmt.Sprint(s.n, s.base106.n, s.Hello())
	p := &s106{k: 1}
	p.n = 6
	u := s
	u.n = 100
	out += fmt.Sprint(" ", p.base106.n, p.Hello(), s.n, u.n)
	check("embed106", out == "hi0 0 7 7hi7 6hi67 100")
	check("embed106v", fmt.Sprintf("%+v", s106{}) == "{base106:{n:0} k:0}")
}

// ===== SECTION 107: a package's typed constants =====
// time.March is a time.Month however it is spelled, and so is a constant
// expression over one, or over a typed constant of the program's own. The
// expected values are go run's.
type color107 int

const (
	red107 color107 = iota + 1
	green107
)

func (c color107) String() string { return [...]string{"?", "R", "G", "B"}[c] }

func s107() {
	check("pkgconst107", fmt.Sprint(time.March, " ", time.February*2, " ", time.Saturday-1, " ", time.Monday) == "March April Friday Monday")
	check("pkgconst107f", fmt.Sprintf("%v %d %s %T", time.March, time.March, time.March, time.March) == "March 3 March time.Month")
	check("pkgconst107c", fmt.Sprint(green107+1, red107*2) == "B G")
}

// ===== SECTION 108: fmt's Scan family =====
// Sscan treats a newline as space, Sscanln stops at one, Sscanf matches its
// format; an operand is stored by its pointer's type, and a failure answers the
// count stored so far and the error. Fscan reads only what it needs, so two
// calls over one reader see consecutive tokens. The expected values are go run's.
type onlyReader108 struct{ r io.Reader }

func (o onlyReader108) Read(p []byte) (int, error) { return o.r.Read(p) }

func s108() {
	var a, b int
	var s string
	var f float64
	var f32 float32
	var u8 uint8
	var r rune
	var t bool
	out := ""
	n, err := fmt.Sscanf("12 34 hello", "%d %d %s", &a, &b, &s)
	out += fmt.Sprint(n, err, a, b, s, ";")
	n, err = fmt.Sscan("0x1F\n-2.5e1 true", &a, &f, &t)
	out += fmt.Sprint(n, err, a, f, t, ";")
	n, err = fmt.Sscanln("aa bb\ncc", &s, &s)
	out += fmt.Sprint(n, err, s, ";")
	n, err = fmt.Sscanf("7", "%d %d", &a, &b)
	out += fmt.Sprint(n, err, ";")
	n, err = fmt.Sscan("abc", &a)
	out += fmt.Sprint(n, err, ";")
	n, err = fmt.Sscan("256", &u8)
	out += fmt.Sprint(n, err, ";")
	n, err = fmt.Sscan("3.4e39", &f32)
	out += fmt.Sprint(n, err, ";")
	n, err = fmt.Sscanf("x:9 5%", "%c:%d %d%%", &r, &a, &b)
	out += fmt.Sprint(n, err, string(r), a, b, ";")
	n, err = fmt.Sscanf("12345", "%3d%d", &a, &b)
	out += fmt.Sprint(n, err, a, b, ";")
	n, err = fmt.Sscan("5", a)
	out += fmt.Sprint(n, err, ";")
	check("scan108", out == "3 <nil> 12 34hello;3 <nil> 31 -25 true;2 <nil>bb;1 EOF;0 expected integer;0 unsigned integer overflow on token 256;0 strconv.ParseFloat: parsing \"3.4e39\": value out of range;3 <nil>x9 5;2 <nil> 123 45;0 type not a pointer: int;")
	rd := onlyReader108{strings.NewReader("1 2\n3 4\n")}
	out = ""
	for {
		n, err := fmt.Fscan(rd, &a, &b)
		out += fmt.Sprint(n, err, a, b, ";")
		if err != nil {
			break
		}
	}
	sr := strings.NewReader("5 6\n7\n")
	for {
		n, err := fmt.Fscanln(sr, &a, &b)
		out += fmt.Sprint(n, err, a, b, ";")
		if err != nil {
			break
		}
	}
	check("fscan108", out == "2 <nil> 1 2;2 <nil> 3 4;0 EOF 3 4;2 <nil> 5 6;1 unexpected newline 7 6;")
}

// ===== SECTION 109: a func and os.Stdin in an interface; a package's interface =====
// A func value keeps its signature in an interface, for %T, an assertion and a
// type switch; os.Stdin, os.Stdout and os.Stderr are *os.File, which is both an
// io.Reader and an io.Writer; io.Reader is satisfied by any type with Read. The
// expected values are go run's.
func sq109(x int) int { return x * x }

type t109 struct{}

func (t109) M(s string) bool { return s != "" }

func (t109) Read(p []byte) (int, error) { return 0, nil }

func show109(v any) string { return fmt.Sprintf("%T", v) }

func s109() {
	var f any = sq109
	var t t109
	out := show109(f) + " " + show109(t.M) + " " + show109(func(a string, b ...int) (int, error) { return 0, nil })
	g := f.(func(int) int)
	out += fmt.Sprint(" ", g(3))
	switch h := f.(type) {
	case func(int) int:
		out += fmt.Sprint(" sw", h(4))
	}
	_, isStr := f.(func(string) string)
	out += fmt.Sprint(" ", isStr)
	check("functy109", out == "func(int) int func(string) bool func(string, ...int) (int, error) 9 sw16 false")
	var in, o, e any = os.Stdin, os.Stdout, os.Stderr
	_, r1 := in.(io.Reader)
	_, w1 := in.(io.Writer)
	_, w2 := o.(io.Writer)
	_, s2 := o.(io.StringWriter)
	var tr any = t
	_, r3 := tr.(io.Reader)
	check("osfile109", fmt.Sprintf("%T %T %T", in, o, e) == "*os.File *os.File *os.File" && r1 && w1 && w2 && s2 && r3)
}

// ===== SECTION 110: a store through `*(x)` =====
// x is any expression answering a pointer: an assertion, a call, an index. The
// expected values are go run's.
type v110 struct{ p any }

type p110 struct{ x int }

func id110(p *int) *int { return p }

func s110() {
	k := 1
	v := v110{&k}
	*(v.p.(*int)) = 5
	out := fmt.Sprint(k)
	*(id110(&k)) += 2
	*(v.p.(*int))++
	ps := []*int{&k}
	out += fmt.Sprint(" ", k)
	*(ps[0]) = 42
	s := &p110{1}
	w := v110{s}
	*(w.p.(*p110)) = p110{9}
	out += fmt.Sprint(" ", k, " ", s.x)
	check("derefx110", out == "5 8 42 9")
}

// ===== SECTION 111: pointers to struct fields and to elements =====
// &x.f addresses the field (the embedded struct's, when promoted) and &s[i] the
// element in the backing array, so a store through either reaches x and s, and a
// later append that reallocates leaves the old element behind. `*(a).p` is
// `*((a).p)`. The expected values are go run's.
type in111 struct {
	x int
	b uint8
}

type o111 struct {
	in111
	name string
	arr  [3]int
	fs   []int
	p    *int
}

func inc111(p *int) { *p++ }

func (o *o111) xp() *int { return &o.x }

func s111() {
	var o o111
	inc111(&o.x)
	inc111(&o.in111.x)
	pb := &o.b
	*pb += 200
	*pb += 200
	pn := &o.name
	*pn = "n"
	pf := &o.fs
	*pf = append(*pf, 4)
	inc111(&o.arr[1])
	pa := &o.arr
	inc111(&pa[2])
	*(o.xp()) += 10
	out := fmt.Sprint(o.x, " ", o.b, " ", o.name, " ", o.fs, " ", o.arr)
	xs := []int{1, 2, 3}
	inc111(&xs[1])
	e := &xs[2]
	xs = append(xs, 4)
	xs[2] = 100
	*e = 7
	out += fmt.Sprint(" ", xs, " ", *e)
	g := [][]int{{1}, {2}}
	inc111(&g[1][0])
	out += fmt.Sprint(" ", g)
	var o2 o111
	out += fmt.Sprint(" ", &o.x == &o.x, &o.x == &o2.x, &o.arr[0] == &o.arr[1])
	check("addr111", out == "12 144 n [4] [0 1 1] [1 3 100 4] 7 [[1] [3]] true false false")
	r := struct{ a, b int }{}
	n, err := fmt.Sscan("5 6", &r.a, &xs[0])
	check("scan111", n == 2 && err == nil && r.a == 5 && xs[0] == 6)
	k := 1
	o.p = &k
	*(o).p = 5
	po := &o
	*(po).p += 2
	*(*po).p *= 3
	check("derefx111", k == 21)
	func() {
		defer func() { out = fmt.Sprint(recover()) }()
		var np *o111
		_ = &np.name
	}()
	check("nilfield111", out == "runtime error: invalid memory address or nil pointer dereference")
	func() {
		defer func() { out = fmt.Sprint(recover()) }()
		i := 3
		_ = &xs[i+2]
	}()
	check("oob111", out == "runtime error: index out of range [5] with length 4")
}

// ===== SECTION 112: an anonymous struct with embedded fields =====
// An embedded field of a struct literal's own type is named after its type, and
// promotes its fields and methods, through a pointer too; two literals of one
// anonymous type are comparable. The expected values are go run's.
type in112 struct{ n int }

func (i in112) Get() int { return i.n }

func (i *in112) Bump() { i.n++ }

func s112() {
	r := struct{ *strings.Reader }{strings.NewReader("abc")}
	c, _ := r.ReadByte()
	out := fmt.Sprint(string(c), r.Len(), r.Reader.Len())
	v := struct {
		in112
		k int
	}{in112{4}, 5}
	v.Bump()
	p := struct{ *in112 }{&in112{7}}
	p.Bump()
	out += fmt.Sprint(" ", v.n, v.Get(), v.in112.n, v.k, " ", p.n, p.Get())
	out += fmt.Sprintf(" %+v", v)
	e1 := struct{ in112 }{in112{2}}
	e2 := struct{ in112 }{in112{2}}
	check("anonembed112", out == "a2 2 5 5 5 5 8 8 {in112:{n:5} k:5}" && e1 == e2)
}

// ===== SECTION 113: %T of fmt's, errors' and os' functions =====
// The expected values are go run's.
func s113() {
	out := fmt.Sprintf("%T|%T|%T|%T", fmt.Println, fmt.Sprintf, fmt.Fprintf, fmt.Sscan)
	out += fmt.Sprintf("|%T|%T|%T", errors.New, errors.As, os.Exit)
	check("hostfnty113", out == "func(...interface {}) (int, error)|func(string, ...interface {}) string|"+
		"func(io.Writer, string, ...interface {}) (int, error)|func(string, ...interface {}) (int, error)|"+
		"func(string) error|func(error, interface {}) bool|func(int)")
}

// ===== SECTION 114: a statement that opens with '(' after a selector =====
// Go ends a line after an identifier, so `pm := w.cnt` is a statement of its own
// and the next line's `(pm)` is not a call of w.cnt. The expected values are go
// run's.
type w114 struct{ cnt map[string]int }

func s114() {
	w := w114{map[string]int{}}
	pm := w.cnt
	(pm)["a"]++
	pp := &w.cnt
	(*pp)["a"] += 2
	check("selnl114", w.cnt["a"] == 3)
}

// ===== SECTION 115: a pointer as a map key =====
// &n taken twice is one pointer, and so is &o.x; two &T{} are two. The expected
// values are go run's.
type p115 struct{ x, y int }

func s115() {
	n := 1
	var o p115
	m := map[*int]string{}
	m[&n] = "n"
	m[&n] += "!"
	m[&o.x] = "x"
	m[&o.y] = "y"
	delete(m, &o.y)
	a, b := &p115{}, &p115{}
	mp := map[*p115]int{a: 1, b: 2}
	check("ptrkey115", len(m) == 2 && m[&n] == "n!" && m[&o.x] == "x" && len(mp) == 2 && mp[a] == 1)
}

// ===== SECTION 116: a store through `*` of a call =====
// `*o.xp() += 1` stores through the pointer the call answers. The expected values
// are go run's.
type o116 struct{ x int }

func (o *o116) xp() *int { return &o.x }

func id116(p *int) *int { return p }

func s116() {
	var o o116
	n := 1
	*o.xp() += 10
	*o.xp()++
	*id116(&n) = 5
	fs := []func() *int{o.xp}
	*fs[0]() *= 2
	check("derefcall116", o.x == 22 && n == 5)
}

// ===== SECTION 117: an anonymous struct type as a declared type =====
// A variable, a field, a slice element and a parameter of an inline struct type
// get a zero instance with every field at its own type's zero, and a value of one
// copies on assignment like any struct. The expected values are go run's.
type o117 struct {
	Name string
	In   struct {
		V    int
		X, Y int8
	}
}

var cfg117 struct {
	Port int
	Tags []string
}

func bump117(p struct{ n int }) int {
	p.n++
	return p.n
}

func s117() {
	var r struct {
		a int
		s string
		b uint8
		w struct{ z int }
	}
	r.a = 5
	r.b = 255
	r.b++
	out := fmt.Sprintf("%v %+v", r, r)
	var o o117
	o.In.V = 3
	o.In.X = 127
	o.In.X++
	o2 := o
	o2.In.V = 4
	out += fmt.Sprint(" ", o, o2.In.V)
	a := struct{ x int }{1}
	b := a
	b.x = 2
	out += fmt.Sprint(" ", a.x, b.x, bump117(struct{ n int }{6}))
	cfg117.Port = 80
	cfg117.Tags = append(cfg117.Tags, "t")
	out += fmt.Sprintf(" %+v", cfg117)
	ps := make([]struct{ a, b int }, 2)
	ps[1].b = 9
	var e1, e2 struct{ v int }
	out += fmt.Sprint(" ", ps, e1 == e2)
	check("anonty117", out == "{5  0 {0}} {a:5 s: b:0 w:{z:0}} { {3 -128 0}} 4 1 2 7 {Port:80 Tags:[t]} [{0 0} {0 9}] true")
}

// ===== SECTION 118: an embedded type of another package =====
// `strings.Builder` embedded in a struct is the field Builder, and its methods are
// the struct's, through a pointer too. The expected values are go run's.
type s118 struct {
	strings.Builder
	n int
}

type p118 struct{ *strings.Builder }

func s118f() {
	var s s118
	s.WriteString("hi")
	s.WriteByte('!')
	p := p118{&strings.Builder{}}
	p.WriteString("ab")
	check("pkgembed118", s.String() == "hi!" && s.Len() == 3 && s.Builder.Len() == 3 && s.n == 0 && p.String() == "ab")
}

// ===== SECTION 119: fmt.Append, Appendf and Appendln; os.Getenv and os.LookupEnv =====
// fmt.Append* append what Sprint* make of their operands to a []byte, which grows
// by append's rule. The expected values are go run's.
func s119() {
	buf := fmt.Append(nil, "x", 1, 2)
	out := fmt.Sprint(len(buf), cap(buf), string(buf))
	buf = fmt.Appendf(buf, "-%d-%s", 7, "q")
	buf = fmt.Appendln(buf, "end", 3)
	out += fmt.Sprintf(" %q %d %d", string(buf), len(buf), cap(buf))
	small := make([]byte, 0, 2)
	grown := fmt.Append(small, "héllo")
	out += fmt.Sprint(" ", len(grown), len(small), fmt.Append(nil) == nil)
	v, ok := os.LookupEnv("MEC_UNSET_VARIABLE_119")
	check("fmtappend119", out == `4 8x1 2 "x1 2-7-qend 3\n" 14 16 6 0 true`)
	check("getenv119", os.Getenv("MEC_UNSET_VARIABLE_119") == "" && v == "" && !ok && os.Getenv("PATH") != "")
}

// ===== SECTION 120: the capacity append grows into holds zero values =====
// s[:cap(s)] after a growing append shows the slots past the old length: the
// element type's zero, its own instance for a struct or an array, nil for a
// pointer. The expected values are go run's.
type p120 struct{ x, y int }

func s120() {
	s := append([]p120{}, p120{1, 2})
	s = append(s, p120{3, 4})
	s = append(s, p120{5, 6})
	s = s[:cap(s)]
	as := append([][2]int{}, [2]int{1, 2})
	as = append(as, [2]int{3, 4})
	as = append(as, [2]int{5, 6})
	as = as[:cap(as)]
	as[3][0] = 7
	ps := append([]*p120{}, &p120{})
	ps = append(ps, &p120{})
	ps = append(ps, &p120{})
	ps = ps[:cap(ps)]
	out := fmt.Sprint(len(s), s, as, len(ps), ps[3] == nil)
	check("growzero120", out == "4 [{1 2} {3 4} {5 6} {0 0}] [[1 2] [3 4] [5 6] [7 0]] 4 true")
	// A type declared inside a function pads with its own zero too, as does a
	// top-level type whose name a local declaration shadows.
	type in120 struct {
		b []int
		c bool
	}
	type loc120 struct {
		in in120
		n  int
	}
	var ls []loc120
	for i := 0; i < 3; i++ {
		ls = append(ls, loc120{n: i})
		func() { ls = append(ls, loc120{n: 10 + i}) }()
	}
	ls = ls[:cap(ls)]
	type p120 struct{ x, y int }
	sh := append([]p120{}, p120{1, 2})
	sh = append(sh, p120{3, 4})
	sh = append(sh, p120{5, 6})
	sh = sh[:cap(sh)]
	out = fmt.Sprint(len(ls), ls[7].in.b == nil, ls[6:], sh)
	check("growzeroloc120", out == "8 true [{{[] false} 0} {{[] false} 0}] [{1 2} {3 4} {5 6} {0 0}]")
}

// ===== SECTION 121: a local type is its own type; a generic function's local type =====
// A type declared inside a function is scoped to its block: a top-level type of the
// same name, and another function's local type of that name, are different types
// with their own fields, zero values and conversions. Inside a generic function a
// local type depends on the type parameters, so its zero value and %T follow the
// instantiation (main.W121[string]). A type argument that shares its name with a
// type parameter elsewhere is still the type. The expected values are go run's.
type T121 struct{ x, y string }
type n121 string

func s121loc() string {
	type T121 struct{ z int }
	type n121 int
	var t T121
	t.z = 3
	ts := []T121{t, {4}}
	m := map[string]T121{"a": {5}}
	var i interface{} = t
	kind := "other"
	switch i.(type) {
	case T121:
		kind = "local"
	}
	x := n121(5)
	return fmt.Sprintf("%v %v %v %T %T %#v %s %v %T", t, ts, m, ts, &t, t, kind, x+2, x)
}

func s121other() string {
	type T121 struct{ q float64 }
	return fmt.Sprint(T121{1.5}, T121{})
}

func s121gen[T any]() string {
	type W121 struct {
		v T
		n int
	}
	var w W121
	ws := []W121{w, {n: 2}}
	var z T
	return fmt.Sprintf("%v|%v|%T|%v|%T|%v", w.v, w, w, ws, ws, z)
}

// A type parameter named like the top-level type it is instantiated with.
func s121z[T121 any]() string {
	var z T121
	return fmt.Sprint(z)
}

func s121() {
	xs := append([]T121{}, T121{"p", "q"}, T121{"r", "t"})
	out := fmt.Sprint(xs, s121loc(), s121other(), T121{"a", "b"}, n121("s"))
	check("ltyshadow121", out == "[{p q} {r t}]{3} [{3} {4}] map[a:{5}] []main.T121 *main.T121 main.T121{z:3} local 7 main.n121{1.5} {0}{a b}s")
	check("ltygen121", s121gen[string]() == "|{ 0}|main.W121[string]|[{ 0} { 2}]|[]main.W121[string]|")
	check("ltygen121b", s121gen[int]() == "0|{0 0}|main.W121[int]|[{0 0} {0 2}]|[]main.W121[int]|0")
	check("ltytp121", s121z[T121]() == "{ }")
	check("ltygen121c", s121gen[T121]() == "{ }|{{ } 0}|main.W121[main.T121]|[{{ } 0} {{ } 2}]|[]main.W121[main.T121]|{ }")
}

// ===== SECTION 122: append's growth inside a generic function =====
// Inside a generic function a slice's element type is the instantiated one: the
// slots growth adds past the length hold its zero (a struct's own instance, a
// string's "") and the capacity follows its size - also for a local type, and
// for a type argument built from the caller's own type parameter. A string, bool
// or defined element type pads with its own zero too, and a struct of an
// instantiated generic type is a value. Every slice escapes into sink122, so
// gc's stack buffer for a first small append does not apply. The expected
// values are go run's.
type b122[T any] struct {
	v T
	p *int
}

type pr122[K comparable, V any] struct {
	k K
	v V
}

type n122 string

var sink122 any

func g122[T any](x T) string {
	ws := []b122[T]{}
	ws = append(ws, b122[T]{v: x}, b122[T]{v: x}, b122[T]{v: x}, b122[T]{v: x}, b122[T]{v: x})
	sink122 = ws
	ms := make([]b122[T], 0)
	ms = append(ms, b122[T]{v: x})
	ms = append(ms, b122[T]{v: x})
	ms = append(ms, b122[T]{v: x})
	ms = ms[:cap(ms)]
	sink122 = ms
	ts := []T{x, x, x}
	ts = append(ts, x)
	ts = ts[:cap(ts)]
	sink122 = ts
	type w122 struct {
		v T
		n int
	}
	ls := []w122{{x, 1}}
	ls = append(ls, w122{x, 2})
	ls = append(ls, w122{x, 3})
	ls = ls[:cap(ls)]
	sink122 = ls
	return fmt.Sprintf("%d %d %T %v %v|%d %T %#v|%d %v", cap(ws), len(ms), ms[3], ms[3].v, ms[3].p == nil,
		len(ts), ts[5], ts[5], len(ls), ls[3])
}

func h122[K comparable, V any](k K, v V) string {
	ps := []pr122[K, V]{{k, v}, {k, v}, {k, v}}
	ps = append(ps, pr122[K, V]{k, v})
	ps = ps[:cap(ps)]
	sink122 = ps
	return fmt.Sprintf("%d %T %v", len(ps), ps[5], ps[5])
}

func o122[T any](x T) string { return i122[b122[T]]() }

func i122[U any]() string {
	var z U
	us := []U{z, z, z}
	us = append(us, z)
	us = us[:cap(us)]
	sink122 = us
	return fmt.Sprintf("%T %v %d %v", z, z, len(us), us[5])
}

func s122() {
	s := []string{"a", "b", "c"}
	s = append(s, "d")
	s = s[:cap(s)]
	n := []n122{"a", "b", "c"}
	n = append(n, "d")
	n = n[:cap(n)]
	bs := []bool{true, true, true}
	bs = append(bs, true)
	bs = bs[:cap(bs)]
	a := b122[int]{v: 1}
	ws := []b122[int]{a}
	a.v = 9
	var arr [2]b122[string]
	arr[0].v = "x"
	cp := arr
	cp[0].v = "y"
	out := fmt.Sprintf("%q %q %v %v %v %v %v", s[5], n[5], bs[7], a.v, ws[0].v, arr[0].v, cp[0].v)
	check("growpad122", out == `"" "" false 9 1 x y`)
	check("growgen122a", g122(5) == "5 4 main.b122[int] 0 true|6 int 0|4 {0 0}")
	check("growgen122b", g122("a") == `5 4 main.b122[string]  true|6 string ""|4 { 0}`)
	check("growgen122c", g122([3]int32{1, 2, 3}) == "5 4 main.b122[[3]int32] [0 0 0] true|6 [3]int32 [3]int32{0, 0, 0}|4 {[0 0 0] 0}")
	check("growgen122d", h122("k", 1)+"|"+h122(2, true) == "6 main.pr122[string,int] { 0}|6 main.pr122[int,bool] {0 false}")
	check("growgen122e", o122(7)+"|"+o122("q") == "main.b122[int] {0 <nil>} 6 {0 <nil>}|main.b122[string] { <nil>} 6 { <nil>}")
}

// ===== SECTION 123: long append loops and the element store =====
// Hundreds of appends grow by gc's rule; each padded slot is a zero of its own;
// an append inside the capacity writes into the array a slice was cut from; the
// stored element keeps the element type's width. The expected values are go run's.
var sink123 any

func s123() {
	var xs []int
	for i := 0; i < 600; i++ {
		xs = append(xs, i*2)
	}
	sum := 0
	for _, x := range xs {
		sum += x
	}
	sink123 = xs
	ss := []string{}
	for i := 0; i < 300; i++ {
		ss = append(ss, fmt.Sprint(i%10))
	}
	sink123 = ss
	check("aploop123", fmt.Sprint(len(xs), cap(xs), xs[599], sum, len(ss), cap(ss), " ", ss[299]) ==
		"600 848 1198 359400 300 303 9")
	s := append([]int{}, 1, 2, 3)
	s = append(s, 4)
	t := s[:cap(s)]
	t[4] = 9
	sink123 = s
	var arr [5]int
	w := arr[1:2]
	w = append(w, 7)
	w = append(w, 8, 9, 10, 11)
	w[0] = 5
	bs := []byte("ab")
	bs = append(bs, 200, 100)
	bs[2] += bs[3]
	u := []uint16{1}
	u = append(u, 65535)
	u[1]++
	m := append([]int{}, 1, 2, 3, 4, 5, 6, 7)
	check("apstore123", fmt.Sprint(t, arr, w, bs, u, len(m), cap(m)) ==
		"[1 2 3 4 9 0] [0 0 7 0 0] [5 7 8 9 10 11] [97 98 44 100] [1 0] 7 8")
}

func main() {
	s94()   // SECTION-CALL 94
	s95()   // SECTION-CALL 95
	s96()   // SECTION-CALL 96
	s97()   // SECTION-CALL 97
	s98()   // SECTION-CALL 98
	s99()   // SECTION-CALL 99
	s102()  // SECTION-CALL 102
	s103()  // SECTION-CALL 103
	s104()  // SECTION-CALL 104
	s105()  // SECTION-CALL 105
	s106f() // SECTION-CALL 106
	s107()  // SECTION-CALL 107
	s108()  // SECTION-CALL 108
	s109()  // SECTION-CALL 109
	s110()  // SECTION-CALL 110
	s111()  // SECTION-CALL 111
	s112()  // SECTION-CALL 112
	s113()  // SECTION-CALL 113
	s114()  // SECTION-CALL 114
	s115()  // SECTION-CALL 115
	s116()  // SECTION-CALL 116
	s117()  // SECTION-CALL 117
	s118f() // SECTION-CALL 118
	s119()  // SECTION-CALL 119
	s120()  // SECTION-CALL 120
	s121()  // SECTION-CALL 121
	s122()  // SECTION-CALL 122
	s123()  // SECTION-CALL 123
	fmt.Println("full:", checks, "checks,", fails, "failures")
	os.Exit(fails)
}
