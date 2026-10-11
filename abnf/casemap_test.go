package abnf

import (
	"fmt"
	"strings"
	"testing"

	"14.gy/mec/abnf/r"
	"github.com/dop251/goja"
)

// String case mapping is the same function in the two grammar hosts: goja (the
// interpreter halves under the default engine, installGojaCaseMapping) and the
// frozen engine (abnf/jsrt.go's string methods, which a compiled MetaJS program
// under llvm.Run uses too). A divergence here is invisible to the matrix unless a
// test happens to reach the character, so every BMP code point is compared, plus
// the lone surrogates, which a Go string cannot carry.
//
// The expectations are node's: the full mapping with Final_Sigma, and a lone
// surrogate has no case, passes through and ends a sigma's context.
func TestCaseMappingAgreesAcrossHosts(t *testing.T) {
	vm := goja.New()
	fm := map[string]r.Object{}
	NewCommonScript(vm, &fm, true)
	if _, err := vm.RunString(`
		function __units(s) {
			var o = [];
			for (var i = 0; i < s.length; i++) { o.push(s.charCodeAt(i)); }
			return o.join(",");
		}
		function __map(units, up) {
			var s = String.fromCharCode.apply(null, units);
			return __units(up ? s.toUpperCase() : s.toLowerCase());
		}`); err != nil {
		t.Fatal(err)
	}
	mapFn, _ := goja.AssertFunction(vm.Get("__map"))
	gojaMap := func(units []uint16, up bool) string {
		args := make([]interface{}, len(units))
		for i, u := range units {
			args[i] = int(u)
		}
		v, err := mapFn(goja.Undefined(), vm.ToValue(args), vm.ToValue(up))
		if err != nil {
			t.Fatal(err)
		}
		return v.String()
	}
	rt := newJSRT(nil)
	frozenMap := func(units []uint16, up bool) string {
		name := "toLowerCase"
		if up {
			name = "toUpperCase"
		}
		res, _ := rt.builtinMethod(&boundMethod{recv: strFromUnits(units), name: name}, nil).(string)
		var parts []string
		for _, u := range strUnits(res) {
			parts = append(parts, fmt.Sprint(u))
		}
		return strings.Join(parts, ",")
	}

	cases := [][]uint16{
		{0xD800, 'a'}, {'B', 0xDC00}, {'x', 0xDBFF, 0xDBFF, 'y'}, // lone surrogates
		{0xD801, 0xDC28}, // a pair: U+10428 maps to U+10400
		{0x00DF}, {0xFB01}, {0x0130}, {0x0131}, {0x03C2}, {0x01C5},
		{'A', 0x03A3}, {0x03A3, 'A'}, {'A', 0xD800, 0x03A3}, {'A', 0x03A3, 0xDC00}, {'A', 0x0301, 0x03A3},
	}
	for c := 0; c < 0x10000; c++ {
		if c < 0xD800 || c > 0xDFFF {
			cases = append(cases, []uint16{uint16(c)})
		}
	}
	bad := 0
	for _, u := range cases {
		for _, up := range []bool{true, false} {
			g, f := gojaMap(u, up), frozenMap(u, up)
			if g != f {
				bad++
				if bad <= 10 {
					t.Errorf("%v up=%v: goja %s, frozen %s", u, up, g, f)
				}
			}
		}
	}
	if bad > 10 {
		t.Errorf("... %d disagreements in all", bad)
	}

	for _, w := range []struct {
		in   []uint16
		up   bool
		want string
	}{
		{[]uint16{0xD800, 'a'}, true, "55296,65"},
		{[]uint16{'B', 0xDC00}, false, "98,56320"},
		{[]uint16{'x', 0xDBFF, 0xDBFF, 'y'}, true, "88,56319,56319,89"},
		{[]uint16{0xD801, 0xDC28}, true, "55297,56320"},
		{[]uint16{0x019B}, true, "42972"},
		{[]uint16{0x00DF}, true, "83,83"},
		{[]uint16{0x0130}, false, "105,775"},
		{[]uint16{0x1FB3}, true, "913,921"},
		{[]uint16{'A', 0x03A3}, false, "97,962"},
		{[]uint16{0x03A3, 'A'}, false, "963,97"},
		{[]uint16{'A', 0xD800, 0x03A3}, false, "97,55296,963"},
		{[]uint16{'A', 0x03A3, 0xDC00}, false, "97,962,56320"},
	} {
		if got := frozenMap(w.in, w.up); got != w.want {
			t.Errorf("%v up=%v: %s, node answers %s", w.in, w.up, got, w.want)
		}
	}
}
