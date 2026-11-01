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
	"math/cmplx"
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
	// 70,000 iterations of a loop: the matcher's backtrack choices live on a
	// stack of its own, not the native one.
	long := strings.Repeat("ab", 35000)
	check("rx6", regexp.MustCompile(`^.*$`).MatchString(long) && len(regexp.MustCompile(`(ab)+`).FindString(long)) == 70000)
	// Linear work on a long subject is never cut off by the matcher's step limit.
	check("rx7", len(regexp.MustCompile(`(?:a|b){2,}`).FindString(long)) == 70000)
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

// ===== SECTION 124: a value whose type is a type parameter, in an interface =====
// An int64, a uint64, an array of them, a defined type or a pointer bound to T
// reaches fmt, an interface result or a type switch as the type T is bound to:
// %T, String() and the case taken follow the instantiation - also through a
// variadic ...T and an any(x) conversion, which boxes like any other interface
// store. The expected values are go run's.
type c124 float64

func (c c124) String() string { return fmt.Sprintf("%.1fC", float64(c)) }

type p124 struct{ x int }

func z124[T any]() string {
	var z T
	return fmt.Sprintf("%T", z)
}

func v124[T any](v T) string { return fmt.Sprintf("%T %v", v, v) }

func a124[T any](v T) any { return v }

func k124(x any) string {
	switch x.(type) {
	case int:
		return "int"
	case int64:
		return "int64"
	case uint64:
		return "uint64"
	}
	return "other"
}

func l124[T any](xs ...T) []any {
	out := []any{}
	for _, x := range xs {
		out = append(out, x)
	}
	return append(out, any(xs[0]))
}

func s124() {
	var w int64 = 5
	ls := l124(uint64(1), 2)
	check("tpconv124", fmt.Sprintf("%T %T %v %v %T %T %T", any(w), interface{}(c124(1)), any(w) == any(5), any(w) == any(int64(5)),
		ls[0], ls[1], ls[2]) == "int64 main.c124 false true uint64 uint64 uint64")
	check("tpwide124", z124[int64]()+" "+z124[[2]int64]()+" "+z124[uint64]()+" "+z124[uintptr]()+" "+z124[int]() ==
		"int64 [2]int64 uint64 uintptr int")
	check("tpval124", v124(int64(5))+"|"+v124(c124(2))+"|"+v124([2]uint64{1, 2})+"|"+v124(&p124{3}) ==
		"int64 5|main.c124 2.0C|[2]uint64 [1 2]|*main.p124 &{3}")
	check("tpret124", k124(a124(int64(1)))+" "+k124(a124[uint64](2))+" "+k124(a124(3))+" "+fmt.Sprintf("%T", a124(c124(1))) ==
		"int64 uint64 int main.c124")
}

// ===== SECTION 125: the bytes of a surrogate's UTF-8 =====
// ED A0 80 spells a surrogate, which UTF-8 forbids: to Go it is three bytes, each
// a rune of its own (U+FFFD, width 1), and two such triples concatenate to six
// bytes rather than to one astral character. The expected values are go run's.
func s125() {
	hs := []string{"\xed\xa0\x80", "\xed\xb0\x80"}
	s := hs[0] + hs[1]
	var b strings.Builder
	b.WriteString(hs[0])
	b.WriteString(hs[1])
	check("surcat125", len(s) == 6 && s[3] == 0xed && fmt.Sprint([]byte(s)) == "[237 160 128 237 176 128]")
	check("surjoin125", fmt.Sprintf("%s%s", hs[0], hs[1]) == s && strings.Join(hs, "") == s && b.String() == s)
	check("surwid125", fmt.Sprintf("%-4s|%.1s|%3.2s|", hs[0], hs[0], hs[0]) == hs[0]+" |\xed| \xed\xa0|")
	sp := strings.Split(hs[0], "")
	check("sursplit125", len(sp) == 3 && sp[0] == "\xed" && sp[2] == "\x80" && fmt.Sprint(strings.SplitN("a😀c", "", 2)) == "[a 😀c]")
	r := []rune{0x1F600, 'x'}
	check("surrune125", string(r[0]) == "\xf0\x9f\x98\x80" && string(r) == "😀x" && fmt.Sprintf("😀%c|%3s|", r[0], "😀") == "😀😀|  😀|")
}

// ===== SECTION 126: make of a defined type =====
// make takes any type whose underlying type is a slice, map or channel, and the
// value keeps the defined type (%T, its methods). The expected values are go run's.
type vec126 []float64
type cnt126 map[string]int
type ch126 chan int
type grid126 [][]int

func (v vec126) sum() float64 {
	s := 0.0
	for _, x := range v {
		s += x
	}
	return s
}

func s126() {
	v := make(vec126, 3)
	v[1] = 2.5
	v = append(v, 4)
	w := make(vec126, 2, 10)
	check("mkslice126", fmt.Sprintf("%v %d %v %T", v, len(v), v.sum(), v) == "[0 2.5 0 4] 4 6.5 main.vec126" && len(w) == 2 && cap(w) == 10)
	m := make(cnt126)
	m["a"]++
	m2 := make(cnt126, 8)
	check("mkmap126", fmt.Sprintf("%v %d %d %T", m, m["zz"], len(m2), m) == "map[a:1] 0 0 main.cnt126")
	c := make(ch126, 2)
	c <- 5
	g := make(grid126, 2)
	g[0] = make([]int, 3)
	check("mkchan126", <-c == 5 && cap(c) == 2 && fmt.Sprint(g) == "[[0 0 0] []]")
}

// ===== SECTION 127: complex128 locations and math/cmplx =====
// A real constant put in a complex128 location (a declaration, an assignment,
// an argument, a result, an element, a field, a comparison operand) becomes
// (x+0i); the zero complex128 is (0+0i); a complex map key compares by value.
// A constant result takes the result type, so a float64 func's `return 1` is
// 1.0. The expected values are go run's.
type cx127 struct{ z complex128 }

type deg127 float64

func one127() complex128      { return 1 }
func half127() float64        { return 1 }
func int127() int             { return 2.0 }
func pair127() (int, float64) { return 4, 5 }
func deg127f() deg127         { return 7 }

func s127() {
	var z complex128 = 3
	var zero complex128
	xs := []complex128{1, 2.5}
	xs[0] = 7
	p := cx127{z: 2}
	check("cxloc127", fmt.Sprint(z, zero, xs, p, one127()) == "(3+0i) (0+0i) [(7+0i) (2.5+0i)] {(2+0i)} (1+0i)" && half127()/2 == 0.5)
	pa, pb := pair127()
	check("kret127", int127()/4 == 0 && pa/3 == 1 && pb/2 == 2.5 && deg127f()/2 == 3.5)
	m := map[complex128]int{1i: 2, 3: 4}
	m[2i] = 6
	k := 2.0
	check("cxkey127", m[1i] == 2 && m[3] == 4 && m[2i] == 6 && len(m) == 3 && complex(k, 0) == 2 && z == 3)
	w := complex(3, -4)
	r, th := cmplx.Polar(complex(0, 2))
	check("cmplx127", cmplx.Abs(w) == 5 && cmplx.Conj(w) == 3+4i && cmplx.Sqrt(-4) == 2i && r == 2 && th == math.Pi/2 &&
		fmt.Sprint(cmplx.Sqrt(w), cmplx.Phase(w), cmplx.Sqrt(complex(1.5, -2.25))) == "(2-1i) -0.9272952180016122 (1.449855761204885-0.7759392555471052i)")
	check("cmplx127b", fmt.Sprint(cmplx.Exp(complex(1, 2)), cmplx.Log(complex(-1, 0)), cmplx.Pow(1i, 2), cmplx.Pow(complex(2, 1), complex(0.5, -1))) ==
		"(-1.1312043837568135+2.4717266720048183i) (0+3.141592653589793i) (-1+1.2246467991473515e-16i) (1.99781201148972-1.288710124239901i)")
	check("cmplx127c", cmplx.IsInf(cmplx.Inf()) && cmplx.IsNaN(cmplx.NaN()) && !cmplx.IsNaN(complex(math.NaN(), math.Inf(1))) &&
		fmt.Sprintf("%.4f %.4f", cmplx.Rect(2, math.Pi/2), cmplx.Log10(100)) == "(0.0000+2.0000i) (2.0000+0.0000i)")
	check("atan2127", math.Atan2(1, -1) == 3*math.Pi/4 && math.Atan2(math.Copysign(0, -1), -2) == -math.Pi && fmt.Sprint(math.Atan2(3, 7), math.Atan2(-5, -0.5)) == "0.40489178628508343 -1.6704649792860586")
}

// ===== SECTION 128: a multiply fused into an add =====
// The spec lets x*y + z round once, and the rule is decided from the source: a
// float product written as a direct operand of + or - fuses, through parentheses
// and unary minus, in v += x*y, for float32 too, but not through an explicit
// float64() conversion. With a product on both sides the right one fuses. A
// product held in a local does not fuse; a product of locals does, whatever values
// gc knows they hold.
// A complex multiply is a*c - b*d and a*d + b*c, each fusing its right product,
// in v *= w too. Integers do not fuse. go run on arm64 agrees except where gc's
// optimiser decides otherwise, which the spec permits: fma128c, d, e, g, h and i
// hold this rule's answers (go run's differ in the last place).
var fma128v = []float64{1.0000000009313226, -1, 0.37, 30, 1.0000000004656613, 0.1, 0.3, 3.7, 2.9, 1.000244140625}

//go:noinline
func dot128(a, b, c, d float64) float64 { return a*b + c*d }

//go:noinline
func dots128(a, b, c, d float64) (float64, float64) { return a*b + c*d, a*c + b*d }

//go:noinline
func cx128(z, w complex128) (complex128, complex128) { return z * w, z / w }

//go:noinline
func cxs128(z, w complex128) (complex128, complex128) { return z * w, w * w }

//go:noinline
func kn128(c float64) (float64, float64, float64, float64) {
	a := 1.1
	b := 1.9
	d := float64(a)
	s := c
	s += a * b
	return a*b + c, s, c - a*b, -d*b + c
}

//go:noinline
func knr128(c float64) (float64, float64, float64) {
	a := 1.1
	b := 1.9
	if c > 100 {
		a = 2
	}
	f, r := 1.1, 0.0
	for i := 0; i < 2; i++ {
		if i == 0 {
			r = f*b + c
		}
		f = f + c
	}
	g, q := 1.1, 0.0
	for i := 0; i < 2; i++ {
		h := g + 0
		if i == 0 {
			q = h*b + c
		}
		g = g + c
	}
	return a*b + c, r, q
}

//go:noinline
func ckn128(x, y, u, v float64) (complex128, complex128, float64) {
	z := complex(1.1, 2.3)
	w := complex(0.7, -1.9)
	a, b := 0.7, -1.9
	return z * w, z * complex(a, b), x*y + u*v
}

//go:noinline
func via128(x, y, z float64) (float64, float64, float64, float64, float64) {
	t := x * y
	u := -(y * x)
	v := t
	w := x * y
	if z > 100 {
		w = z
	}
	r := z
	r += t
	return t + z, z + u, v - z, w + z, r
}

//go:noinline
func via128b(a, h float64) float64 {
	t := a * a
	u := h * -h
	return t + u
}

type f128 float64
type g128 float32

//go:noinline
func gen128[T float32 | float64](a, b, c T) T { return a*b + c }

//go:noinline
func ng128(xs []float64) (complex128, complex128, complex128) {
	u := complex(xs[3], xs[5])
	v := complex(xs[4], -xs[2])
	w := complex(xs[4], xs[2])
	t := u
	t *= v
	return u * v, u * complex(real(w), -imag(w)), t
}

func s128() {
	a, m, k, t, h := fma128v[0], fma128v[1], fma128v[2], fma128v[3], fma128v[4]
	n := len(fma128v) + 2
	check("fma128a", fmt.Sprint(a*a+m, m+a*a, a*a-1, 1-a*a, -(a*a)+1, (a*a)+m, float64(a*a)+m, float64(n)*k-t) ==
		"1.8626451500983188e-09 1.8626451500983188e-09 1.8626451500983188e-09 -1.8626451500983188e-09 -1.8626451500983188e-09 1.8626451500983188e-09 1.862645149230957e-09 -25.56")
	s := m
	s += a * a
	u := 1.0
	u -= a * a
	var f, g float32 = float32(fma128v[9]), float32(fma128v[9])
	check("fma128b", fmt.Sprint(s, u, f*g-1, 1-f*g, float32(f*g)-1) == "1.8626451500983188e-09 -1.8626451500983188e-09 0.00048834085 -0.00048834085 0.00048828125")
	x, y := dots128(a, a, h, -h)
	i, j := 7, 9
	check("fma128c", fmt.Sprint(dot128(a, a, h, -h), x, y, i*j+2, 2-i*j, -(i*j)+1) == "9.313225743986381e-10 9.313225743986381e-10 -4.336808689942018e-19 65 -61 -62")
	z := complex(fma128v[5], fma128v[6])
	w := complex(fma128v[7], fma128v[8])
	p, q := cx128(z, w)
	r, o := cxs128(z, w)
	check("fma128d", fmt.Sprint(p, q, r, o) == "(-0.4999999999999999+1.4i) (0.056108597285067875+0.037104072398190045i) (-0.4999999999999999+1.4i) (5.280000000000002+21.46i)")
	check("fma128e", fmt.Sprint(kn128(fma128v[6])) == "2.39 2.39 -1.79 -1.79")
	check("fma128f", fmt.Sprint(knr128(fma128v[6])) == "2.39 2.39 2.39")
	check("fma128h", fmt.Sprint(via128(fma128v[7], fma128v[8], fma128v[6])) == "11.030000000000001 -10.43 10.43 11.030000000000001 11.030000000000001" &&
		via128b(a, h) == 9.313225746154785e-10)
	check("fma128g", fmt.Sprint(ckn128(a, a, h, -h)) == "(5.14-0.4800000000000001i) (5.14-0.4800000000000001i) 9.313225743986381e-10")
	check("fma128i", fmt.Sprint(ng128([]float64{0.3, -0.77, 1.9, 1.1, 0.7, 2.3})) ==
		"(5.14-0.4800000000000001i) (5.14-0.4800000000000001i) (5.14-0.4800000000000001i)")
	// A defined float type and a type parameter fuse at their values' width.
	fa, fm := f128(a), f128(m)
	ga, gb := g128(f), g128(g)
	check("fma128j", fa*fa+fm == 1.8626451500983188e-09 && ga*gb-1 == 0.00048834085 && gen128(a, a, m) == 1.8626451500983188e-09 &&
		fmt.Sprint(gen128(f, g, -1)) == "0.00048834085")
}

// ===== SECTION 129: a store through a pointer takes the pointee's type =====
// `*p = v` converts an untyped constant to the type p points at, as `x = v`
// does: 5 stored through a *complex128 is (5+0i), 3 through a *float64 divides
// as a float, a defined float type and a **complex128 the same. The expected
// values are go run's.
type cel129 float64

func s129() {
	var z complex128 = 1i
	p := &z
	*p = 5
	var f float64
	q := &f
	*q = 3
	var c cel129
	r := &c
	*r = 7
	check("ptrst129", fmt.Sprint(z, *p, f/2, c/2) == "(5+0i) (5+0i) 1.5 3.5")
	pp := &p
	**pp = 2
	var u uint8 = 250
	s := &u
	*s += 10
	check("ptrst129b", fmt.Sprint(z, u) == "(2+0i) 4")
}

// ===== SECTION 130: math/cmplx's trigonometric functions and math's hyperbolic ones =====
// cmplx Sin, Cos, Tan, Cot, Sinh, Cosh, Tanh, Asin, Acos, Atan and their
// hyperbolic inverses, over math Sinh, Cosh, Tanh, Asin, Acos, Asinh, Acosh,
// Atanh, Log1p, Log2 and Log10; Tan near a pole takes the series. The expected
// values are go run's on arm64.
func s130() {
	z := complex(0.75, -1.25)
	w := complex(0.3, 0.2)
	check("ctrig130", fmt.Sprint(cmplx.Sin(z), cmplx.Cos(z), cmplx.Tan(z)) ==
		"(1.2872229100264918-1.1721063598927026i) (1.381738730634259+1.0919301355539746i) (0.16080778591620642-0.9753632850312357i)")
	check("chyp130", fmt.Sprint(cmplx.Sinh(z), cmplx.Cosh(z), cmplx.Tanh(z)) ==
		"(0.2592948545511628-1.2286345240950955i) (0.40824259187796885-0.7803659308458533i) (1.372607570533783-0.38579595260975064i)")
	check("cinv130", fmt.Sprint(cmplx.Asin(z), cmplx.Acos(z), cmplx.Atan(z)) ==
		"(0.45327617763879385-1.1323936316053085i) (1.1175201491561026+1.1323936316053085i) (1.1071487177940904-0.5493061443340549i)")
	check("cinvh130", fmt.Sprint(cmplx.Asinh(w), cmplx.Acosh(w), cmplx.Atanh(w), cmplx.Cot(w), cmplx.Tan(complex(math.Pi/2, 1e-9))) ==
		"(0.3010735394466424+0.19245144716045318i) (0.20772637624812318+1.2727619269517418i) (0.29574992023641433+0.21547449370018829i) (2.207905035537343-1.6061528688158473i) (61.232339957367344+9.999999999999963e+08i)")
	check("mhyp130", fmt.Sprint(math.Sinh(0.3), math.Cosh(-2.5), math.Tanh(0.4), math.Asin(0.6), math.Acos(-0.2), math.Asinh(3), math.Acosh(1.5), math.Atanh(0.3)) ==
		"0.3045202934471426 6.132289479663686 0.3799489622552249 0.6435011087932843 1.7721542475852274 1.8184464592320668 0.9624236501192069 0.30951960420311175")
	check("mlog130", fmt.Sprint(math.Log1p(1e-5), math.Log1p(-0.3), math.Log2(10), math.Log10(2), math.Log2(8)) ==
		"9.999950000333332e-06 -0.35667494393873234 3.321928094887362 0.3010299956639812 3")
}

// ===== SECTION 131: a package calls its own function; a constant before a call =====
// The main file's Abs below does not change what math's functions, which call
// math's own Abs, answer. A call's arguments run their calls first (gc's order),
// and a constant or nil argument before them is converted where it stands. The
// expected values are go run's.
func Abs(v complex128) complex128 { return v * v }

func add131(a, b float64) float64 { return a + b }

func half131(a float64) float64 { return a / 2 }

func pick131(a []int, b string) string { return fmt.Sprint(len(a), b) }

func s131() {
	x := []float64{2.5, -7.25}
	check("ownfn131", fmt.Sprint(math.Round(x[0]), math.Mod(x[1], x[0]), math.Float64bits(x[0]), Abs(2i)) == "3 -2.25 4612811918334230528 (-4+0i)")
	var z float64 = 1
	check("kfirst131", fmt.Sprint(add131(0, half131(x[0])), add131(0.5, half131(z)), pick131(nil, fmt.Sprint(z)), math.Max(0, math.Abs(x[1]))) == "1.25 1017.25")
	var p, q float64 = 0, half131(x[1])
	check("kfirst131b", fmt.Sprint(p, q) == "0 -3.625")
}

// ===== SECTION 132: complex64 =====
// complex(float32, float32) and complex64(x) make a complex64, whose parts are
// float32: + and - round in float32, * and / compute in float64 and narrow, an
// untyped constant beside one converts first, real and imag answer float32, and
// it prints at 32 bits. A complex64 slot - a var, an element, a field, a map key,
// an appended constant - holds one. The expected values are go run's.
var c132 = []float32{0.1, 0.3, 3.7, 2.9}

type s132 struct{ z complex64 }

func s132f() {
	z := complex(c132[0], c132[1])
	w := complex(c132[2], c132[3])
	check("c64ops132", fmt.Sprint(z, w, z+w, z-w, z*w, z/w, -z, 1/z, z*2, z+1) ==
		"(0.1+0.3i) (3.7+2.9i) (3.8+3.2i) (-3.6000001-2.6000001i) (-0.50000006+1.4000001i) (0.056108598+0.037104074i) (-0.1-0.3i) (0.99999994-3i) (0.2+0.6i) (1.1+0.3i)")
	check("c64parts132", fmt.Sprint(real(z), imag(w), real(z)*2) == "0.1 2.9 0.2" && fmt.Sprintf("%T %T %.3f", z, real(z), w) == "complex64 float32 (3.700+2.900i)")
	c128 := complex(float64(c132[0]), float64(c132[1]))
	var zero complex64
	ws := []complex64{1, 2i}
	ws = append(ws, -1i, 0.5)
	p := s132{3}
	m := map[complex64]int{z: 7}
	check("c64conv132", fmt.Sprint(complex64(c128), complex128(z), zero, ws, p, m[z]) ==
		"(0.1+0.3i) (0.10000000149011612+0.30000001192092896i) (0+0i) [(1+0i) (0+2i) (0-1i) (0.5+0i)] {(3+0i)} 7" && fmt.Sprintf("%T", ws[2]) == "complex64")
}

// ===== SECTION 133: a variable typed by a conversion =====
// `x := T(v)` and `var x = T(v)` declare x of type T, so a constant assigned to x
// later is converted to T: a float64 divides as one, a sized integer wraps, a
// complex prints as one. The expected values are go run's.
func s133() {
	c := float64(3)
	c = 7
	var u = uint8(3)
	u = 255
	u++
	i := int8(1)
	i = 127
	i++
	check("convty133", fmt.Sprint(c/2, u, i) == "3.5 0 -128")
	h := complex128(3)
	h = 6
	h6 := complex64(3)
	h6 = 6
	g := float32(1)
	g = 16777217
	var cc = complex64(complex(1, 2))
	q := &cc
	*q = 2
	check("convcx133", fmt.Sprint(h, h6, g, cc) == "(6+0i) (6+0i) 1.6777216e+07 (2+0i)")
}

// ===== SECTION 134: a pointer to a complex field or element =====
// A complex is a scalar, so &s.c and &a[i] of a complex type address the field and
// the element: a store through the pointer reaches s and a. The expected values
// are go run's.
type s134 struct {
	x int
	c complex128
	h complex64
}

func s134f() {
	s := s134{}
	pc := &s.c
	*pc = 4
	ph := &s.h
	*ph = 2i
	*ph += 1
	ca := []complex128{1, 2}
	pa := &ca[0]
	*pa = 5
	var arr [2]complex64
	pr := &arr[1]
	*pr = 9
	check("cxaddr134", fmt.Sprint(s, *pc, ca, arr, pc == &s.c) == "{0 (4+0i) (1+2i)} (4+0i) [(5+0i) (2+0i)] [(0+0i) (9+0i)] true")
}

// ===== SECTION 135: break in a select, and break of a labelled switch or select =====
// An unlabelled break in a select arm ends the select, not the loop around it;
// `break L` ends the switch or select labelled L, even from a loop inside it. The
// expected values are go run's.
func s135() {
	c := make(chan int, 8)
	for i := 0; i < 6; i++ {
		c <- i
	}
	got := []int{}
	for k := 0; k < 4; k++ {
		select {
		case v := <-c:
			if v == 1 {
				break
			}
			if v == 2 {
				continue
			}
			got = append(got, v)
		}
		got = append(got, -1)
	}
	check("selbrk135", fmt.Sprint(got) == "[0 -1 -1 3 -1]")
	n := 0
L:
	select {
	case v := <-c:
		if v > 0 {
			break L
		}
		n = -1
	}
M:
	switch {
	default:
		for i := 0; ; i++ {
			if i == 2 {
				break M
			}
			n += 10
		}
		n = 99
	}
	check("lblbrk135", n == 20 && len(c) == 1)
}

// ===== SECTION 136: a struct type holding a map of itself =====
// The zero value of such a struct holds a nil map whose missing elements read as
// that zero value again, at any depth. The expected values are go run's.
type s136 struct {
	n int
	m map[int]s136
}

type t136 struct {
	a [2]map[string]t136
	l []t136
}

func s136f() {
	var s s136
	out := fmt.Sprint(len(s.m), s.n, s.m[1].n, len(s.m[1].m[2].m))
	s.m = map[int]s136{}
	s.m[1] = s136{n: 5}
	x := s.m[2]
	x.n = 3
	var t t136
	out += fmt.Sprint(" ", len(s.m), s.m[1].n, s.m[2].n, x.n, len(t.a[1]), len(t.a[0]["k"].a[1]))
	check("selfmap136", out == "0 0 0 0 1 5 0 3 0 0")
}

// ===== SECTION 137: `*p` and `x.a` as range and receive targets =====
// A range clause and a select receive spelled with `=` store into any assignable
// target - a pointer's target, a field, an element - and into names declared
// outside; a received value may be parenthesized, and an element type of a channel
// type too. The expected values are go run's.
type x137 struct{ a int }

func s137() {
	arr := []int{5, 6}
	var i, v int
	p, q := &i, &v
	sum := 0
	for *p, *q = range arr {
		sum += *p + *q
	}
	var x x137
	for x.a = range 3 {
	}
	xs := []int{0, 0}
	for xs[1] = range 4 {
	}
	check("rngtgt137", sum == 12 && i == 1 && v == 6 && x.a == 2 && xs[1] == 3)
	c := make(chan int, 4)
	c <- 7
	c <- 8
	c <- 9
	c <- 10
	select {
	case *p = <-c:
	}
	select {
	case x.a = <-c:
	}
	n, ok := 0, false
	select {
	case n, ok = <-c:
	}
	select {
	case (<-c):
	}
	var cc chan (<-chan int)
	check("seltgt137", i == 7 && x.a == 8 && n == 9 && ok && len(c) == 0 && cc == nil)
}

// ===== SECTION 138: a goroutine blocked forever, and selects that never fire =====
// A send or a receive on a nil channel blocks forever (spec "Send statements",
// "Receive operator"), so does `select {}`, a select whose arms are all on nil
// channels, and a select or receive with nothing that could ever be ready: the
// goroutine stops there, its deferred calls never run, and main goes on. A nil
// channel's arm is never chosen, so the default runs. Every arm's channel and sent
// value is evaluated once on entering a select, the chosen arm's or not. The
// expected values are go run's.
func s138() {
	var nilch chan int
	empty := make(chan int)
	var sent, deferred, recvd, sel1, sel2, sel3, sel4 bool
	done := make(chan bool, 1)
	go func() {
		defer func() { deferred = true; recover() }()
		nilch <- 7
		sent = true
	}()
	go func() {
		<-nilch
		recvd = true
	}()
	go func() {
		select {}
		sel1 = true
	}()
	go func() {
		select {
		case nilch <- 1:
		case <-nilch:
		}
		sel2 = true
	}()
	go func() {
		select {
		case v := <-empty:
			_ = v
		}
		sel3 = true
	}()
	go func() {
		<-empty
		sel4 = true
	}()
	dflt := false
	go func() {
		select {
		case nilch <- 1:
		default:
			dflt = true
		}
		done <- true
	}()
	<-done
	check("nilblk138", !sent && !deferred && !recvd && dflt)
	check("selblk138", !sel1 && !sel2 && !sel3 && !sel4)
	cnt := 0
	val := func() int { cnt++; return cnt }
	a1 := make(chan int, 1)
	select {
	case a1 <- val():
	case nilch <- val():
	default:
	}
	check("seleval138", cnt == 2 && <-a1 == 1)
}

// ===== SECTION 139: empty and call init statements; literals inside a header =====
// An if, switch or type switch header may have an EMPTY init statement (`if ;
// cond`), and any simple statement - a call, a send - is an init or a for post.
// Only a bare `T{...}` is barred from a header: inside an index, a call's
// arguments or after a slice or map type it is an ordinary literal, and so is
// the `{}` of `interface{}(x)`. The expected values are go run's.
type T139 struct{ a int }

func s139f(t T139) int { return t.a }

func s139log(log *string, s string) func(string) {
	return func(t string) { *log += s + t }
}

func s139() {
	got := ""
	if ; false {
		got += "x"
	}
	if ; true {
		got += "a"
	}
	switch ; {
	case true:
		got += "b"
	}
	n := 3
	switch ; n {
	case 3:
		got += "c"
	}
	var v interface{} = "s"
	switch ; v.(type) {
	case string:
		got += "d"
	}
	switch w := v; u := w.(type) {
	case string:
		got += u
	}
	if s139log(&got, "1")("2"); n > 2 {
		got += "e"
	}
	check("init139", got == "abcds12e")
	c := make(chan bool, 1)
	for ; false; c <- false {
	}
	k := 0
	c <- true
	for i := 0; i < 3; c <- true {
		<-c
		k++
		i++
	}
	check("post139", k == 3 && len(c) == 1)
	m := map[T139][]int{T139{}: {1}, T139{1}: {2}}
	hdr := ""
	if x, y := m[T139{}][0], m[T139{1}][0]; x == 1 && y == 2 {
		hdr += "i"
	}
	if s139f(T139{3}) == 3 {
		hdr += "f"
	}
	for j := 0; j < s139f(T139{2}); j++ {
		hdr += "l"
	}
	if s := []T139{{5}}; s[0].a == 5 {
		hdr += "s"
	}
	switch x := interface{}(nil).(type) {
	case nil:
		hdr += fmt.Sprint(x)
	}
	switch y := interface{}(7).(type) {
	case int:
		hdr += fmt.Sprint(y)
	}
	check("lit139", hdr == "iflls<nil>7")
}

// ===== SECTION 140: a select's send arm needs room or a receiver; range over a nil channel =====
// A send arm is ready only when its channel has room or a receiver is waiting
// (spec "Select statements", "Send statements"): on a full or unbuffered channel
// nobody receives from, the default runs, or the select blocks forever - even
// with a receive arm on the same channel. A goroutine receiving, directly or in a
// select of its own, takes the value; one draining a full channel lets the send
// in behind. A range over a nil channel blocks forever, whatever its type's name.
// The expected values are go run's.
type C140 chan int

func s140() {
	var nilch chan int
	var nilc C140
	ran, after, self := false, false, false
	go func() {
		for range nilch {
			ran = true
		}
		after = true
	}()
	go func() {
		for v := range nilc {
			_ = v
			ran = true
		}
		after = true
	}()
	go func() {
		c := make(chan int)
		select {
		case c <- 1:
		case <-c:
		}
		self = true
	}()
	full := make(chan int, 1)
	full <- 1
	got := ""
	select {
	case full <- 2:
		got = "sent"
	default:
		got = "default"
	}
	ub := make(chan int)
	select {
	case ub <- 1:
		got += " sent"
	default:
		got += " default"
	}
	check("selroom140", got == "default default" && len(full) == 1)
	check("rangenil140", !ran && !after && !self)
	res := make(chan int, 1)
	hand := make(chan int)
	go func() { res <- (<-hand) * 10 }()
	got = ""
	select {
	case hand <- 4:
		got += "handed"
	}
	got += fmt.Sprint(" ", <-res)
	hand2 := make(chan string)
	go func() {
		select {
		case s := <-hand2:
			res <- len(s)
		}
	}()
	select {
	case hand2 <- "abc":
		got += " handed2"
	}
	got += fmt.Sprint(" ", <-res)
	go func() { res <- <-full }()
	select {
	case full <- 9:
		got += " behind"
	}
	got += fmt.Sprint(" ", <-res, len(full), <-full)
	a, b := make(chan int), make(chan int)
	go func() { res <- <-b }()
	select {
	case a <- 1:
		got += " a"
	case b <- 2:
		got += " b"
	}
	got += fmt.Sprint(" ", <-res)
	check("selrecv140", got == "handed 40 handed2 3 behind 1 1 9 b 2")
}

// ===== SECTION 141: go statements and send statements =====
// A go statement evaluates the function value and the arguments when it runs, not
// when the goroutine does; a "<-" glued together is the arrow token, so `<-cc <- 2`
// sends on the channel received from cc; a send from main with a goroutine still
// to run waits for it rather than deadlocking.
func s141add(out chan int, a, b int) { out <- a + b }

func s141() {
	out := make(chan int, 4)
	x := 1
	f := s141add
	go f(out, x, 10)
	go func(v int) { out <- v * 100 }(x)
	xs := []int{2, 3}
	go s141add(out, xs[0], xs[1])
	x = 5
	f = nil
	xs[0] = 50
	sum := <-out + <-out + <-out
	check("goargs141", sum == 11+100+5)
	cc := make(chan chan int, 1)
	c := make(chan int, 2)
	cc <- c
	sent := false
	select {
	case <-cc <- 2:
		sent = true
	default:
	}
	cc <- c
	<-cc <- 3
	b := make(chan bool, 1)
	b <- false || true
	check("arrow141", sent && <-c == 2 && <-c == 3 && <-b)
	work := make(chan int, 1)
	done := make(chan int)
	go func() {
		t := 0
		for v := range work {
			t += v
		}
		done <- t
	}()
	for i := 1; i <= 4; i++ {
		work <- i
	}
	close(work)
	check("sendwait141", <-done == 10)
}

// ===== SECTION 142: array element types, generic inference, method expressions =====
// An array keeps its element type in an interface even with no elements, or with
// elements of an interface type; a slice expression has a static type a generic
// call infers from; G[T] alone is the instantiated function; a method expression
// of a promoted method takes the receiver as its first argument.
type s142T []s142T

func (*s142T) Count(args ...int) int { return len(args) }

type s142U struct{ *s142T }

type s142P struct{ n int }

func (p *s142P) Add(a, b int) int { return p.n + a + b }

type s142Q struct{ *s142P }

func s142kind(v interface{}) string {
	switch v.(type) {
	case [0]string:
		return "[0]string"
	case [0]int:
		return "[0]int"
	case [2]interface{}:
		return "[2]any"
	}
	return "other"
}

func s142max[T int | string](a, b T) T {
	if a > b {
		return a
	}
	return b
}

func s142fold[E any](s []E, init E, f func(E, E) E) E {
	r := init
	for _, v := range s {
		r = f(r, v)
	}
	return r
}

func s142smax[E int | string](s []E) E {
	if len(s) == 0 {
		var zero E
		return zero
	}
	return s142fold(s[1:], s[0], s142max[E])
}

func s142() {
	var e interface{} = [0]string{}
	var a interface{} = [2]any{1, 2}
	ty := fmt.Sprintf("%T %T %T %#v %#v", e, a, [1][0]int{}, e, a)
	check("arrty142", ty == "[0]string [2]interface {} [1][0]int [0]string{} [2]interface {}{1, 2}")
	_, ok := e.([0]string)
	check("arrsw142", s142kind(e)+" "+s142kind([0]int{})+" "+s142kind(a) == "[0]string [0]int [2]any" && ok && e != interface{}([0]int{}))
	ss := []string{"b", "dd", "a"}
	check("geninf142", fmt.Sprintf("%q %q %d", s142smax(ss[:0]), s142smax(ss), s142smax([]int{3, 9, 4})) == `"" "dd" 9`)
	check("mexpr142", (*s142U).Count(&s142U{}, 1, 2)+s142U.Count(s142U{}, 5) == 3 && (*s142Q).Add(&s142Q{&s142P{100}}, 1, 2) == 103 && s142Q.Add(s142Q{&s142P{7}}, 0, 1) == 8)
}

// ===== SECTION 143: a promoted method as a method-expression value =====
// T.M and (*T).M as values are functions taking the receiver first, also when M
// is promoted from an embedded field, and when it is declared on a defined
// non-struct type: its receiver is then the embedded field (its address, for a
// pointer method of a value field). The expected values are go run's.
type s143T []int

func (t *s143T) Sum() int {
	s := 0
	for _, x := range *t {
		s += x
	}
	return s
}

func (t *s143T) Push(xs ...int) { *t = append(*t, xs...) }

func (t s143T) Len() int { return len(t) }

type s143C int

func (c *s143C) Inc(n int) { *c += s143C(n) }
func (c s143C) Get() int     { return int(c) }

type s143U struct{ *s143T }

type s143W struct {
	name string
	s143T
	s143C
}

type s143D struct {
	tag int
	s143W
}

type s143P struct{ v int }

func (p *s143P) Add(n int) int { return p.v + n }
func (p s143P) Twice() int     { return 2 * p.v }

type s143Q struct{ *s143P }
type s143R struct{ s143Q }

func s143() {
	ns := []int{4, 5, 6}
	add := (*s143Q).Add
	qadd := s143Q.Add
	radd := (*s143R).Add
	rtw := s143R.Twice
	r := &s143R{s143Q{&s143P{3}}}
	check("mexprval143", add(&s143Q{&s143P{1}}, 2) == 3 && qadd(s143Q{&s143P{20}}, ns[0]) == 24 && radd(r, ns[1]) == 8 && rtw(*r) == 6 && (*s143R).Twice(r) == 6)
	t := s143T{ns[0], ns[1]}
	u := &s143U{&t}
	sum := (*s143U).Sum
	push := (*s143U).Push
	push(u, ns[2], 1)
	tsum := (*s143T).Sum
	check("mexprdt143", u.Sum() == 16 && u.Len() == 4 && (*s143U).Sum(u) == 16 && s143U.Len(*u) == 4 && sum(u) == 16 && tsum(&t) == 16 && fmt.Sprint(t) == "[4 5 6 1]")
	w := &s143W{name: "w", s143T: s143T{1, 2, 3}, s143C: 7}
	inc := (*s143W).Inc
	inc(w, ns[0])
	(*s143W).Push(w, 10)
	get := s143W.Get
	d := &s143D{tag: 1, s143W: s143W{s143T: s143T{9, 9}, s143C: 2}}
	dinc := (*s143D).Inc
	dinc(d, 40)
	fs := []func(*s143U) int{(*s143U).Sum, (*s143U).Len}
	check("mexprval2143", w.s143C == 11 && get(*w) == 11 && w.Sum() == 16 && (*s143W).Sum(w) == 16 && (*s143D).Sum(d) == 18 && d.Get() == 42 && s143D.Get(*d) == 42 && fs[0](u)+fs[1](u) == 20)
	ws := []s143W{{s143T: s143T{2, 2}, s143C: 9}}
	(&ws[0]).Inc(100)
	check("promrecv143", (&s143U{&t}).Sum() == 16 && s143U{&t}.Len() == 4 && (s143W{s143T: s143T{1}, s143C: 4}).Get() == 4 &&
		s143W{s143T: s143T{1, 1}}.Len() == 2 && (ws[0]).Len() == 2 && ws[0].Get() == 109 && (ws[0]).Sum() == 4)
}

// ===== SECTION 144: a method promoted from a defined non-struct type, called through an interface =====
// A struct embedding a defined non-struct type has its methods in its method set,
// so it satisfies an interface by them and a call through the interface reaches
// the embedded field (its address, for a pointer method of a value field); a
// promoted String makes the struct print through it. A method expression's %T is
// M's signature with the receiver first. The expected values are go run's.
type s144L []int

func (l *s144L) Push(v int) { *l = append(*l, v) }
func (l s144L) Len() int    { return len(l) }

type s144C int

func (c *s144C) Inc()          { *c++ }
func (c s144C) Get() int       { return int(c) }
func (c s144C) String() string { return fmt.Sprintf("C%d", int(c)) }

type s144U struct{ *s144L }

type s144B struct {
	s144L
	name string
}

type s144O struct {
	s144B
	k int
}

type s144H struct{ s144C }

type s144Pusher interface {
	Push(int)
	Len() int
}

type s144Incer interface {
	Inc()
	Get() int
}

type s144P struct{ x int }

func (p *s144P) Add(n int) int { return p.x + n }
func (p s144P) Twice() int     { return p.x * 2 }

func s144fill(p s144Pusher, vs []int) int {
	for _, v := range vs {
		p.Push(v)
	}
	return p.Len()
}

func s144() {
	vs := []int{7, 8}
	l := s144L{1}
	u := &s144U{&l}
	b := &s144B{name: "b"}
	o := &s144O{}
	n := s144fill(u, vs)*100 + s144fill(b, vs)*10 + s144fill(o, vs)
	check("promdt144", n == 322 && fmt.Sprint(l, b.s144L, o.s144L) == "[1 7 8] [7 8] [7 8]")
	h := &s144H{}
	var i s144Incer = h
	i.Inc()
	i.Inc()
	var s fmt.Stringer = s144H{s144C(5)}
	check("promdtcall144", i.Get() == 2 && h.s144C == 2 && s.String() == "C5" && fmt.Sprint(h, " ", s144H{s144C(3)}) == "C2 C3")
	check("mexprty144", fmt.Sprintf("%T %T", (*s144P).Add, s144P.Twice) == "func(*main.s144P, int) int func(main.s144P) int")
}

// ===== SECTION 145: a struct value's method set in a type switch and an assertion =====
// A struct VALUE in an interface has only the value-receiver methods (Go spec
// "Method sets"), so it does not satisfy an interface needing a pointer method,
// while a pointer to it does. A by-value embedded struct lends a value only its
// value methods; an embedded *T lends all of T's. The expected values are go run's.
type s145S struct{ n int }

func (s *s145S) Inc()    { s.n++ }
func (s s145S) Get() int { return s.n }

type s145Incer interface {
	Inc()
	Get() int
}

type s145Getter interface{ Get() int }

type s145E struct{ s145S }

type s145P struct{ *s145S }

func s145kind(v interface{}) string {
	switch v.(type) {
	case s145Incer:
		return "i"
	case s145Getter:
		return "g"
	}
	return "o"
}

func s145() {
	s := s145S{6}
	ps := &s
	vs := []interface{}{s145S{7}, &s145S{8}}
	k := s145kind(s145S{1}) + s145kind(&s145S{2}) + s145kind(s) + s145kind(*ps) + s145kind(ps) + s145kind(vs[0]) + s145kind(vs[1])
	check("vmset145", k == "giggigi")
	var g s145Getter = s145S{3}
	_, ok1 := g.(s145Incer)
	var g2 s145Getter = &s145S{4}
	_, ok2 := g2.(s145Incer)
	var x interface{} = s145S{5}
	_, ok3 := x.(s145Incer)
	check("vmsetok145", !ok1 && ok2 && !ok3)
	check("vmsetemb145", s145kind(s145E{})+s145kind(&s145E{})+s145kind(s145P{&s145S{}}) == "gii")
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
	s124()  // SECTION-CALL 124
	s125()  // SECTION-CALL 125
	s126()  // SECTION-CALL 126
	s127()  // SECTION-CALL 127
	s128()  // SECTION-CALL 128
	s129()  // SECTION-CALL 129
	s130()  // SECTION-CALL 130
	s131()  // SECTION-CALL 131
	s132f() // SECTION-CALL 132
	s133()  // SECTION-CALL 133
	s134f() // SECTION-CALL 134
	s135()  // SECTION-CALL 135
	s136f() // SECTION-CALL 136
	s137()  // SECTION-CALL 137
	s138()  // SECTION-CALL 138
	s139()  // SECTION-CALL 139
	s140()  // SECTION-CALL 140
	s141()  // SECTION-CALL 141
	s142()  // SECTION-CALL 142
	s143()  // SECTION-CALL 143
	s144()  // SECTION-CALL 144
	s145()  // SECTION-CALL 145
	fmt.Println("full:", checks, "checks,", fails, "failures")
	os.Exit(fails)
}
