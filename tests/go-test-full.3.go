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
	fmt.Println("full:", checks, "checks,", fails, "failures")
	os.Exit(fails)
}
