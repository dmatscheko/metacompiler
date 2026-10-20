//go:build ignore

//go:debug panicnil=0

// The directive above: the repository's go.mod says go 1.16, under which a
// toolchain recovers panic(nil) as nil; this file asserts the Go 1.21+ answer
// (a *runtime.PanicNilError), as the grammars implement it.

// Full-syntax test: Go (Go 1.22 core language) - part 3 of 3, sections 94 on;
// tests/go-test-full.2.go says why the ratchet is three files. Add a new section
// to this file.
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
	"os"
	"path"
	"path/filepath"
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

func main() {
	s94() // SECTION-CALL 94
	s95() // SECTION-CALL 95
	s96() // SECTION-CALL 96
	s97() // SECTION-CALL 97
	fmt.Println("full:", checks, "checks,", fails, "failures")
	os.Exit(fails)
}
