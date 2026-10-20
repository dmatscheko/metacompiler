//go:build ignore

//go:debug panicnil=0

// The directive above: the repository's go.mod says go 1.16, under which a
// toolchain recovers panic(nil) as nil; this file asserts the Go 1.21+ answer
// (a *runtime.PanicNilError), as the grammars implement it.

// Full-syntax test: Go (Go 1.22 core language) - part 4 of 4, sections 100 and
// 101 (encoding/json and reflect, whose packages are the largest written in Go);
// tests/go-test-full.2.go says why the ratchet is four files.
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
	"encoding/json"
	"fmt"
	"os"
	"reflect"
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

// Nesting is checked without recursion (1200 levels here; past 10000 Go refuses
// with "exceeded max depth"), and a ,string field's inner literal is checked
// before it is read (only a scalar field reads it).
type Q100 struct {
	N int     `json:",string"`
	B bool    `json:",string"`
	T string  `json:",string"`
	L []int   `json:",string"`
	F float64 `json:",string"`
}

func s100b() string {
	deep := strings.Repeat(`[{"a":`, 600) + "1" + strings.Repeat("}]", 600)
	var v interface{}
	err := json.Unmarshal([]byte(deep+"x"), &v)
	out := fmt.Sprint(json.Valid([]byte(deep)), err, "|")
	for _, in := range []string{`{"N":"12"}`, `{"N":"1x"}`, `{"N":"[1,2]"}`, `{"N":12}`, `{"B":"1"}`, `{"T":"\"hi\""}`, `{"T":"hi"}`, `{"L":"[1]"}`, `{"N":"{"}`, `{"F":"2.5"}`} {
		var q Q100
		err := json.Unmarshal([]byte(in), &q)
		out += fmt.Sprint(err, q, ";")
	}
	return out
}

func s100() {
	check("json1", s100str() == "{\"name\":\"Ann \\u003c\\u0026\\u003e\",\"age\":30,\"tags\":[\"a\",\"b\"],\"home\":{\"street\":\"Main\"},\"scores\":{\"a\":2,\"z\":1},\"ratio\":0.5,\"Count\":\"42\",\"Small\":-3,\"Raw\":\"aGkh\"}<nil>|[null,1,\"two\",3.5,true,null,{\"10\":\"x\",\"9\":\"y\"},\"\\u2028\\u0001\\t\",1e+21,1e-7]|{\n \"t\": 21.5,\n \"unit\": \"C\"\n}|<nil>Bob41 [x y]S{H } map[q:7] 17 5hi!|<nil> [1 2.5 s map[a:[] b:false] <nil>]|<nil> map[a:[1 2] b:[]] <nil> [{1.5 F} {-2 }]|json: cannot unmarshal string into Go value of type float64 json: cannot unmarshal number 300 into Go value of type int8 json: cannot unmarshal string into Go struct field Person100.age of type int <nil> 2.5|invalid character '}' looking for beginning of object key string 8;invalid character '2' after array element 4;unexpected end of JSON input 3;unexpected end of JSON input 0;invalid escape sequence `\\x` in string 3;invalid character '1' looking for beginning of object key string 2;invalid character 'x' after top-level value 4;invalid character '\\x01' in string 3;invalid character ']' in numeric literal 3;{\"x\":[1]}\nabtrue false")
	check("json2", s100b() == "true invalid character 'x' after top-level value|<nil> {12 false  [] 0};json: cannot unmarshal number 1x into Go struct field Q100.N of type int {0 false  [] 0};json: cannot unmarshal number [1,2] into Go struct field Q100.N of type int {0 false  [] 0};json: cannot unmarshal number into Go struct field Q100.N of type int {0 false  [] 0};json: cannot unmarshal string \"1\" into Go struct field Q100.B of type bool: invalid syntax {0 false  [] 0};<nil> {0 false hi [] 0};json: cannot unmarshal string into Go struct field Q100.T of type string: invalid character 'h' looking for beginning of object key string {0 false  [] 0};json: cannot unmarshal string into Go struct field Q100.L of type []int {0 false  [] 0};json: cannot unmarshal number { into Go struct field Q100.N of type int {0 false  [] 0};<nil> {0 false  [] 2.5};")
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

func main() {
	s100() // SECTION-CALL 100
	s101() // SECTION-CALL 101
	fmt.Println("full:", checks, "checks,", fails, "failures")
	os.Exit(fails)
}
