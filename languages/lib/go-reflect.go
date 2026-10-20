// Package reflect: the part of Go's reflect package a program reads values
// with, WRITTEN IN GO, compiled with a program that imports "reflect"
// (core.importFile in both go grammars), over the run-time type primitives
// both grammars bind (goRf*, see lib/go-json.go).
//
// Provided: Kind, TypeOf and its Type (Kind, String, Name, NumField, Field,
// FieldByName, Elem, Key, Len), StructField and StructTag, ValueOf and its
// Value (Kind, Type, Interface, the scalar getters, Len, Index, NumField,
// Field, FieldByName, MapKeys, MapIndex, Elem, IsNil, IsValid, IsZero, and Set
// and SetInt, SetUint, SetFloat, SetString, SetBool through a pointer's Elem),
// and DeepEqual. A type is known by its spelling and the values it holds: a
// func's or chan's element type, methods and the rest of the package are not.
package reflect

import (
	"fmt"
	"strconv"
	"strings"
)

type Kind uint

const (
	Invalid Kind = iota
	Bool
	Int
	Int8
	Int16
	Int32
	Int64
	Uint
	Uint8
	Uint16
	Uint32
	Uint64
	Uintptr
	Float32
	Float64
	Complex64
	Complex128
	Array
	Chan
	Func
	Interface
	Map
	Pointer
	Slice
	String
	Struct
	UnsafePointer
)

const Ptr = Pointer

var kindNames = []string{"invalid", "bool", "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16",
	"uint32", "uint64", "uintptr", "float32", "float64", "complex64", "complex128", "array", "chan", "func",
	"interface", "map", "ptr", "slice", "string", "struct", "unsafe.Pointer"}

func (k Kind) String() string {
	if int(k) < len(kindNames) {
		return kindNames[k]
	}
	return "kind" + strconv.Itoa(int(k))
}

// kindOf maps a goRfKind answer onto a Kind.
func kindOf(k string) Kind {
	switch k {
	case "nil":
		return Invalid
	case "ptr":
		return Pointer
	case "other":
		return Interface
	}
	for i, n := range kindNames {
		if n == k {
			return Kind(i)
		}
	}
	return Invalid
}

// ----- struct tags -----

type StructTag string

func (tag StructTag) Get(key string) string {
	v, _ := tag.Lookup(key)
	return v
}

// Lookup reads the conventional `key:"value" key2:"value2"` form.
func (tag StructTag) Lookup(key string) (value string, ok bool) {
	for tag != "" {
		i := 0
		for i < len(tag) && tag[i] == ' ' {
			i++
		}
		tag = tag[i:]
		if tag == "" {
			break
		}
		i = 0
		for i < len(tag) && tag[i] > ' ' && tag[i] != ':' && tag[i] != '"' && tag[i] != 0x7f {
			i++
		}
		if i == 0 || i+1 >= len(tag) || tag[i] != ':' || tag[i+1] != '"' {
			break
		}
		name := string(tag[:i])
		tag = tag[i+1:]
		i = 1
		for i < len(tag) && tag[i] != '"' {
			if tag[i] == '\\' {
				i++
			}
			i++
		}
		if i >= len(tag) {
			break
		}
		qvalue := string(tag[:i+1])
		tag = tag[i+1:]
		if key == name {
			value, err := strconv.Unquote(qvalue)
			if err != nil {
				break
			}
			return value, true
		}
	}
	return "", false
}

// ----- types -----

// A Type is a type text, as the program spells it, with a value of the type
// when one is at hand (sample) or can be built (goRfZero). Go's Type is an
// interface; a pointer to one struct is what lets a method's result keep its
// static type here (a Kind prints through its String method).
type Type = *rtype

type rtype struct {
	text   string
	sample interface{}
	kind   Kind
}

type StructField struct {
	Name      string
	PkgPath   string
	Type      *rtype
	Tag       StructTag
	Index     []int
	Anonymous bool
}

func (f StructField) IsExported() bool { return f.PkgPath == "" }

func TypeOf(i any) *rtype {
	k := kindOf(goRfKind(i))
	if k == Invalid {
		return nil
	}
	return &rtype{text: fmt.Sprintf("%T", i), sample: i, kind: k}
}

// typeOfText is the Type a declared type text names.
func typeOfText(t string) *rtype {
	switch {
	case t == "" || t == "any" || strings.HasPrefix(t, "interface"):
		return &rtype{text: "interface {}", kind: Interface}
	case strings.HasPrefix(t, "*"):
		return &rtype{text: "*" + typeOfText(t[1:]).String(), kind: Pointer}
	case strings.HasPrefix(t, "[]"):
		return &rtype{text: "[]" + typeOfText(t[2:]).String(), kind: Slice}
	case strings.HasPrefix(t, "["):
		return &rtype{text: t[:strings.IndexByte(t, ']')+1] + typeOfText(t[strings.IndexByte(t, ']')+1:]).String(), kind: Array}
	case strings.HasPrefix(t, "map["):
		k, v := mapTypes(t)
		return &rtype{text: "map[" + typeOfText(k).String() + "]" + typeOfText(v).String(), kind: Map}
	case strings.HasPrefix(t, "func"):
		return &rtype{text: t, kind: Func}
	case strings.HasPrefix(t, "chan"):
		return &rtype{text: t, kind: Chan}
	case t == "byte":
		return &rtype{text: "uint8", kind: Uint8}
	case t == "rune":
		return &rtype{text: "int32", kind: Int32}
	}
	for i, n := range kindNames {
		if n == t && Kind(i) <= Complex128 || t == "string" {
			return &rtype{text: t, kind: kindOf(t)}
		}
	}
	z := goRfZero(t)
	if z == nil {
		return &rtype{text: t, kind: Interface}
	}
	return &rtype{text: fmt.Sprintf("%T", z), sample: z, kind: kindOf(goRfKind(z))}
}

func mapTypes(t string) (string, string) {
	depth := 0
	for i := 3; i < len(t); i++ {
		switch t[i] {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return t[4:i], t[i+1:]
			}
		}
	}
	return "", ""
}

func (t *rtype) Kind() Kind     { return t.kind }
func (t *rtype) String() string { return t.text }

func (t *rtype) Name() string {
	if strings.ContainsAny(t.text, "[]*{} ") || strings.HasPrefix(t.text, "map[") || strings.HasPrefix(t.text, "func") {
		return ""
	}
	if i := strings.LastIndexByte(t.text, '.'); i >= 0 {
		return t.text[i+1:]
	}
	return t.text
}

func (t *rtype) PkgPath() string {
	if i := strings.LastIndexByte(t.text, '.'); i >= 0 && t.Name() != "" {
		return t.text[:i]
	}
	return ""
}

func (t *rtype) mustBe(k Kind, what string) {
	if t.kind != k {
		panic("reflect: " + what + " of non-" + k.String() + " type " + t.text)
	}
}

func (t *rtype) NumField() int {
	t.mustBe(Struct, "NumField")
	return len(goRfFields(t.sample)[0])
}

func (t *rtype) Field(i int) StructField {
	t.mustBe(Struct, "Field")
	fs := goRfFields(t.sample)
	name, typ := fs[0][i], fs[1][i]
	f := StructField{Name: name, Tag: StructTag(fs[2][i]), Index: []int{i}}
	if typ == "" {
		f.Anonymous = true
		f.Type = TypeOf(fs[3][i])
		if f.Type == nil {
			f.Type = typeOfText(name)
		}
	} else {
		f.Type = typeOfText(typ)
	}
	if c := name[0]; !(c >= 'A' && c <= 'Z') {
		f.PkgPath = "main"
	}
	return f
}

func (t *rtype) FieldByName(name string) (StructField, bool) {
	t.mustBe(Struct, "FieldByName")
	for i := 0; i < t.NumField(); i++ {
		if f := t.Field(i); f.Name == name {
			return f, true
		}
	}
	return StructField{}, false
}

func (t *rtype) Elem() *rtype {
	switch {
	case strings.HasPrefix(t.text, "*"):
		return typeOfText(strings.TrimPrefix(t.text[1:], "main."))
	case strings.HasPrefix(t.text, "[]"):
		return typeOfText(strings.TrimPrefix(t.text[2:], "main."))
	case strings.HasPrefix(t.text, "["):
		return typeOfText(strings.TrimPrefix(t.text[strings.IndexByte(t.text, ']')+1:], "main."))
	case strings.HasPrefix(t.text, "map["):
		_, v := mapTypes(t.text)
		return typeOfText(strings.TrimPrefix(v, "main."))
	}
	panic("reflect: Elem of invalid type " + t.text)
}

func (t *rtype) Key() *rtype {
	t.mustBe(Map, "Key")
	k, _ := mapTypes(t.text)
	return typeOfText(strings.TrimPrefix(k, "main."))
}

func (t *rtype) Len() int {
	t.mustBe(Array, "Len")
	n, _ := strconv.Atoi(t.text[1:strings.IndexByte(t.text, ']')])
	return n
}

// ----- values -----

// A Value holds v; ptr, when set, is the pointer it was reached through
// (Elem), which is what a Set writes through.
type Value struct {
	v     interface{}
	valid bool
	ptr   interface{}
}

func ValueOf(i any) Value { return Value{v: i, valid: goRfKind(i) != "nil"} }

func (v Value) IsValid() bool { return v.valid }

func (v Value) Kind() Kind {
	if !v.valid {
		return Invalid
	}
	return kindOf(goRfKind(v.v))
}

func (v Value) Type() *rtype {
	if !v.valid {
		panic("reflect: call of reflect.Value.Type on zero Value")
	}
	return TypeOf(v.v)
}

func (v Value) Interface() any { return v.v }

func (v Value) String() string {
	if v.Kind() == String {
		return goRfUnbox(v.v).(string)
	}
	if !v.valid {
		return "<invalid Value>"
	}
	return "<" + v.Type().String() + " Value>"
}

func (v Value) panicKind(what string) {
	panic(&ValueError{Method: "reflect.Value." + what, Kind: v.Kind()})
}

type ValueError struct {
	Method string
	Kind   Kind
}

func (e *ValueError) Error() string {
	if e.Kind == Invalid {
		return "reflect: call of " + e.Method + " on zero Value"
	}
	return "reflect: call of " + e.Method + " on " + e.Kind.String() + " Value"
}

func (v Value) Bool() bool {
	if v.Kind() != Bool {
		v.panicKind("Bool")
	}
	return goRfUnbox(v.v).(bool)
}

func (v Value) Int() int64 {
	switch x := goRfUnbox(v.v).(type) {
	case int:
		return int64(x)
	case int8:
		return int64(x)
	case int16:
		return int64(x)
	case int32:
		return int64(x)
	case int64:
		return x
	}
	v.panicKind("Int")
	return 0
}

func (v Value) Uint() uint64 {
	switch x := goRfUnbox(v.v).(type) {
	case uint:
		return uint64(x)
	case uint8:
		return uint64(x)
	case uint16:
		return uint64(x)
	case uint32:
		return uint64(x)
	case uint64:
		return x
	case uintptr:
		return uint64(x)
	}
	v.panicKind("Uint")
	return 0
}

func (v Value) Float() float64 {
	switch x := goRfUnbox(v.v).(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	}
	v.panicKind("Float")
	return 0
}

func (v Value) Len() int {
	switch v.Kind() {
	case Slice, Array:
		return len(goRfElems(v.v))
	case Map:
		return len(goRfMapKV(v.v)[0])
	case String:
		return len(goRfUnbox(v.v).(string))
	}
	v.panicKind("Len")
	return 0
}

func (v Value) Index(i int) Value {
	switch v.Kind() {
	case Slice, Array:
		els := goRfElems(v.v)
		if i < 0 || i >= len(els) {
			panic("reflect: slice index out of range")
		}
		return ValueOf(els[i])
	case String:
		return ValueOf(goRfUnbox(v.v).(string)[i])
	}
	v.panicKind("Index")
	return Value{}
}

func (v Value) NumField() int {
	if v.Kind() != Struct {
		v.panicKind("NumField")
	}
	return len(goRfFields(v.v)[0])
}

func (v Value) Field(i int) Value {
	if v.Kind() != Struct {
		v.panicKind("Field")
	}
	fv := goRfFields(v.v)[3][i]
	return Value{v: fv, valid: true}
}

func (v Value) FieldByName(name string) Value {
	if v.Kind() != Struct {
		v.panicKind("FieldByName")
	}
	fs := goRfFields(v.v)
	for i := 0; i < len(fs[0]); i++ {
		if fs[0][i] == name {
			return Value{v: fs[3][i], valid: true}
		}
	}
	return Value{}
}

func (v Value) MapKeys() []Value {
	if v.Kind() != Map {
		v.panicKind("MapKeys")
	}
	ks := goRfMapKV(v.v)[0]
	out := make([]Value, len(ks))
	for i := range out {
		out[i] = ValueOf(ks[i])
	}
	return out
}

func (v Value) MapIndex(key Value) Value {
	if v.Kind() != Map {
		v.panicKind("MapIndex")
	}
	kv := goRfMapKV(v.v)
	for i := 0; i < len(kv[0]); i++ {
		if DeepEqual(kv[0][i], key.v) {
			return Value{v: kv[1][i], valid: true}
		}
	}
	return Value{}
}

func (v Value) IsNil() bool {
	switch v.Kind() {
	case Pointer, Map, Slice, Func, Chan, Interface, Invalid:
		return goRfIsNil(v.v)
	}
	v.panicKind("IsNil")
	return false
}

func (v Value) IsZero() bool {
	switch v.Kind() {
	case Struct:
		for i := 0; i < v.NumField(); i++ {
			if !v.Field(i).IsZero() {
				return false
			}
		}
		return true
	case Array:
		for i := 0; i < v.Len(); i++ {
			if !v.Index(i).IsZero() {
				return false
			}
		}
		return true
	case Pointer, Map, Slice, Func, Chan, Interface, Invalid:
		return goRfIsNil(v.v)
	case String:
		return v.Len() == 0
	case Bool:
		return !v.Bool()
	case Float32, Float64:
		return v.Float() == 0
	case Uint, Uint8, Uint16, Uint32, Uint64, Uintptr:
		return v.Uint() == 0
	}
	return v.Int() == 0
}

// Elem is the value a pointer points at, which a Set then writes through.
func (v Value) Elem() Value {
	switch v.Kind() {
	case Pointer:
		return Value{v: goRfDeref(v.v), valid: true, ptr: v.v}
	case Struct:
		return Value{v: v.v, valid: true, ptr: v.v}
	case Interface:
		return v
	}
	v.panicKind("Elem")
	return Value{}
}

func (v Value) CanSet() bool { return v.ptr != nil }

func (v Value) Set(x Value) {
	if v.ptr == nil {
		panic("reflect: reflect.Value.Set using unaddressable value")
	}
	goRfStore(v.ptr, x.v)
}

func (v Value) setScalar(x interface{}, what string) {
	if v.ptr == nil {
		panic("reflect: reflect.Value." + what + " using unaddressable value")
	}
	goRfStore(v.ptr, goRfConv(goKindText(v.v), x))
}

func goKindText(x interface{}) string {
	k := goRfKind(x)
	if k == "ptr" || k == "nil" || k == "other" {
		return "interface{}"
	}
	return k
}

func (v Value) SetInt(x int64)     { v.setScalar(x, "SetInt") }
func (v Value) SetUint(x uint64)   { v.setScalar(x, "SetUint") }
func (v Value) SetFloat(x float64) { v.setScalar(x, "SetFloat") }
func (v Value) SetString(x string) { v.setScalar(x, "SetString") }
func (v Value) SetBool(x bool)     { v.setScalar(x, "SetBool") }

// DeepEqual is reflect's deep equality over this runtime's values: the same
// kind, and equal elements, fields, map entries or scalars; a nil slice or map
// is not deeply equal to an empty one.
func DeepEqual(x, y any) bool {
	kx, ky := goRfKind(x), goRfKind(y)
	if kx != ky {
		return false
	}
	if fmt.Sprintf("%T", x) != fmt.Sprintf("%T", y) {
		return false
	}
	switch kx {
	case "nil":
		return true
	case "slice", "map":
		if goRfIsNil(x) != goRfIsNil(y) {
			return false
		}
	}
	switch kx {
	case "slice", "array":
		a, b := goRfElems(x), goRfElems(y)
		if len(a) != len(b) {
			return false
		}
		for i := 0; i < len(a); i++ {
			if !DeepEqual(a[i], b[i]) {
				return false
			}
		}
		return true
	case "map":
		a, b := goRfMapKV(x), goRfMapKV(y)
		if len(a[0]) != len(b[0]) {
			return false
		}
		for i := 0; i < len(a[0]); i++ {
			found := false
			for j := 0; j < len(b[0]); j++ {
				if DeepEqual(a[0][i], b[0][j]) {
					found = DeepEqual(a[1][i], b[1][j])
					break
				}
			}
			if !found {
				return false
			}
		}
		return true
	case "struct":
		a, b := goRfFields(x)[3], goRfFields(y)[3]
		for i := 0; i < len(a); i++ {
			if !DeepEqual(a[i], b[i]) {
				return false
			}
		}
		return true
	case "ptr":
		return x == y || DeepEqual(goRfDeref(x), goRfDeref(y))
	case "func":
		return false
	}
	return x == y
}
