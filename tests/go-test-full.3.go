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
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path"
	"path/filepath"
	"reflect"
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

// ===== SECTION 100: package encoding/json =====
// Go source (languages/lib/go-json.go) over the run-time type primitives (goRf*):
// field names, json tags (a name, omitempty, string, -), maps with sorted keys,
// []byte as base64, the float format, decoding into structs, maps, slices and
// interface{}, type errors, and go1.27's syntax errors with their offsets. The
// expected values are go run's.
type Addr100 struct {
	Street string `json:"street"`
	City   string `json:"city,omitempty"`
}

type Person100 struct {
	Name   string         `json:"name"`
	Age    int            `json:"age"`
	Email  string         `json:"email,omitempty"`
	Tags   []string       `json:"tags"`
	Addr   *Addr100       `json:"addr,omitempty"`
	Home   Addr100        `json:"home"`
	Scores map[string]int `json:"scores"`
	Ratio  float64        `json:"ratio"`
	secret string
	Skip   string `json:"-"`
	Count  int64  `json:",string"`
	Small  int8
	Raw    []byte
}

type Temp100 float64

type Reading100 struct {
	T    Temp100 `json:"t"`
	Unit string  `json:"unit"`
}

func s100str() string {
	p := Person100{Name: "Ann <&>", Age: 30, Tags: []string{"a", "b"}, Home: Addr100{Street: "Main"},
		Scores: map[string]int{"z": 1, "a": 2}, Ratio: 0.5, secret: "x", Skip: "s", Count: 42, Small: -3, Raw: []byte("hi!")}
	b, err := json.Marshal(p)
	out := fmt.Sprint(string(b), err, "|")
	b, _ = json.Marshal([]interface{}{nil, 1, "two", 3.5, true, []string(nil), map[int]string{10: "x", 9: "y"}, " \x01\t", 1e21, 1e-7})
	out += string(b) + "|"
	b, _ = json.MarshalIndent(Reading100{T: 21.5, Unit: "C"}, "", " ")
	out += string(b) + "|"
	var q Person100
	err = json.Unmarshal([]byte(`{"name":"Bob","AGE":41,"tags":["x","y"],"addr":{"street":"S"},"home":{"street":"H"},"scores":{"q":7},"Count":"17","zz":[1,{}],"Small":5,"Raw":"aGkh"}`), &q)
	out += fmt.Sprint(err, q.Name, q.Age, q.Tags, q.Addr.Street, q.Home, q.Scores, q.Count, q.Small, string(q.Raw), "|")
	var anyv interface{}
	err = json.Unmarshal([]byte(` [1, 2.5, "s", {"b":false,"a":[]}, null] `), &anyv)
	out += fmt.Sprint(err, anyv, "|")
	var m map[string][]int
	err = json.Unmarshal([]byte(`{"a":[1,2],"b":[]}`), &m)
	var rs []Reading100
	err2 := json.Unmarshal([]byte(`[{"t":1.5,"unit":"F"},{"t":-2}]`), &rs)
	out += fmt.Sprint(err, m, err2, rs, "|")
	var f float64
	var i8 int8
	out += fmt.Sprint(json.Unmarshal([]byte(`"s"`), &f), json.Unmarshal([]byte(`300`), &i8), json.Unmarshal([]byte(`{"age":"old"}`), &q), json.Unmarshal([]byte(`2.5`), &f), f, "|")
	for _, bad := range []string{`{"a":1,}`, `[1 2]`, `tru`, `1.`, `"\x"`, `{1:2}`, `[1]x`, "\"a\x01\"", `[-]`} {
		var v interface{}
		err := json.Unmarshal([]byte(bad), &v)
		se, _ := err.(*json.SyntaxError)
		out += fmt.Sprint(err, se.Offset, ";")
	}
	var sb strings.Builder
	enc := json.NewEncoder(&sb)
	enc.Encode(map[string]interface{}{"x": []int{1}})
	dec := json.NewDecoder(strings.NewReader(`{"street":"a"} {"street":"b"}`))
	for dec.More() {
		var a Addr100
		dec.Decode(&a)
		sb.WriteString(a.Street)
	}
	out += sb.String() + fmt.Sprint(json.Valid([]byte(`{"a":[]}`)), json.Valid([]byte(`{`)))
	return out
}

func s100() {
	check("json1", s100str() == "{\"name\":\"Ann \\u003c\\u0026\\u003e\",\"age\":30,\"tags\":[\"a\",\"b\"],\"home\":{\"street\":\"Main\"},\"scores\":{\"a\":2,\"z\":1},\"ratio\":0.5,\"Count\":\"42\",\"Small\":-3,\"Raw\":\"aGkh\"}<nil>|[null,1,\"two\",3.5,true,null,{\"10\":\"x\",\"9\":\"y\"},\"\\u2028\\u0001\\t\",1e+21,1e-7]|{\n \"t\": 21.5,\n \"unit\": \"C\"\n}|<nil>Bob41 [x y]S{H } map[q:7] 17 5hi!|<nil> [1 2.5 s map[a:[] b:false] <nil>]|<nil> map[a:[1 2] b:[]] <nil> [{1.5 F} {-2 }]|json: cannot unmarshal string into Go value of type float64 json: cannot unmarshal number 300 into Go value of type int8 json: cannot unmarshal string into Go struct field Person100.age of type int <nil> 2.5|invalid character '}' looking for beginning of object key string 8;invalid character '2' after array element 4;unexpected end of JSON input 3;unexpected end of JSON input 0;invalid escape sequence `\\x` in string 3;invalid character '1' looking for beginning of object key string 2;invalid character 'x' after top-level value 4;invalid character '\\x01' in string 3;invalid character ']' in numeric literal 3;{\"x\":[1]}\nabtrue false")
}

// ===== SECTION 101: package reflect =====
// Go source (languages/lib/go-reflect.go): TypeOf, a struct's fields, their types
// and tags, ValueOf and its getters, a Set through a pointer, DeepEqual. The
// expected values are go run's.
type In101 struct {
	X int `json:"x"`
}

type T101 struct {
	Name  string         `json:"name" db:"n"`
	Tags  []string       `json:"tags"`
	M     map[string]int `json:"m"`
	In    In101
	F     float64
	small int8
}

func s101str() string {
	t := T101{Name: "a", Tags: []string{"x"}, M: map[string]int{"k": 1}, In: In101{5}, F: 1.5, small: 2}
	ty := reflect.TypeOf(t)
	out := fmt.Sprint(ty, " ", ty.Kind(), " ", ty.Name(), " ", ty.NumField(), "|")
	for i := 0; i < ty.NumField(); i++ {
		f := ty.Field(i)
		out += fmt.Sprint(f.Name, " ", f.Type, " ", f.Type.Kind(), " ", f.Tag.Get("json"), " ", f.Tag.Get("db"), " ", f.IsExported(), ";")
	}
	v := reflect.ValueOf(t)
	out += fmt.Sprint(v.Kind(), v.NumField(), v.Field(0).String(), v.Field(1).Len(), v.Field(1).Index(0), v.Field(4).Float(), v.FieldByName("In").Field(0).Int(), "|")
	x := 10
	reflect.ValueOf(&x).Elem().SetInt(42)
	out += fmt.Sprint(x, reflect.DeepEqual(map[string][]int{"a": {1}}, map[string][]int{"a": {1}}), reflect.DeepEqual([]int{1}, []int{2}),
		reflect.DeepEqual([]int(nil), []int{}), reflect.TypeOf([]int{}).Elem(), reflect.TypeOf(map[string]bool{}).Key(), reflect.ValueOf(nil).IsValid())
	return out
}

func s101() {
	check("refl1", s101str() == "main.T101 struct T101 6|Name string string name n true;Tags []string slice tags  true;M map[string]int map m  true;In main.In101 struct   true;F float64 float64   true;small int8 int8   false;struct 6a1 x 1.5 5|42 true false false int string false")
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

func main() {
	s94()  // SECTION-CALL 94
	s95()  // SECTION-CALL 95
	s96()  // SECTION-CALL 96
	s97()  // SECTION-CALL 97
	s98()  // SECTION-CALL 98
	s99()  // SECTION-CALL 99
	s100() // SECTION-CALL 100
	s101() // SECTION-CALL 101
	s102() // SECTION-CALL 102
	s103() // SECTION-CALL 103
	fmt.Println("full:", checks, "checks,", fails, "failures")
	os.Exit(fails)
}
