package abnf

import (
	"math"
	"math/rand"
	"strconv"
	"testing"

	"14.gy/mec/abnf/r"
	"github.com/dop251/goja"
)

// Number→String is the same function in the two grammar hosts: goja (the
// interpreter halves under the default engine) and the frozen engine, whose
// string concatenation ends in jsNumString. Every grammar that takes a double's
// digits from "" + n depends on the two agreeing, so the comparison runs over
// the values where a shortest-digits algorithm goes wrong: both ends of every
// binary exponent, the subnormals, the neighbours of each power of ten, and
// seventeen-digit decimals, plus random bit patterns.
func TestNumberToStringAgreesAcrossHosts(t *testing.T) {
	vm := goja.New()
	fm := map[string]r.Object{}
	NewCommonScript(vm, &fm, true)
	if _, err := vm.RunString(`function __str(xs) { return xs.map(function (x) { return "" + x; }); }`); err != nil {
		t.Fatal(err)
	}
	strFn, _ := goja.AssertFunction(vm.Get("__str"))

	var xs []float64
	add := func(v float64) {
		if !math.IsNaN(v) && !math.IsInf(v, 0) {
			xs = append(xs, v, -v)
		}
	}
	for _, v := range []float64{0.0015425378683206609, 1.7385704740834339e+92,
		7.209885073274179e-254, 1.5620869132521429e-115, 5.627494252092118e-309,
		0, 0.1, 1e21, 1e-7, 123e-20, math.MaxFloat64, math.SmallestNonzeroFloat64} {
		add(v)
	}
	for e := -1074; e <= 971; e++ {
		for _, m := range []uint64{1 << 52, 1<<52 + 1, 1<<53 - 1, 1<<53 - 2} {
			add(math.Ldexp(float64(m), e))
		}
	}
	for b := uint64(1); b < 2000; b++ {
		add(math.Float64frombits(b))
		add(math.Float64frombits(1<<52 - b))
		add(math.Float64frombits(1<<52 + b))
	}
	for k := -323; k <= 308; k++ {
		p, _ := strconv.ParseFloat("1e"+strconv.Itoa(k), 64)
		add(p)
		add(math.Nextafter(p, 0))
		add(math.Nextafter(p, math.Inf(1)))
	}
	rnd := rand.New(rand.NewSource(1))
	for i := 0; i < 100000; i++ {
		d := strconv.FormatInt(1e16+rnd.Int63n(9e16), 10)
		v, _ := strconv.ParseFloat(d+"e"+strconv.Itoa(rnd.Intn(616)-324), 64)
		add(v)
		add(math.Float64frombits(rnd.Uint64() &^ (1 << 63)))
	}

	rt := newJSRT(nil)
	bad := 0
	for lo := 0; lo < len(xs); lo += 10000 {
		hi := lo + 10000
		if hi > len(xs) {
			hi = len(xs)
		}
		args := make([]interface{}, hi-lo)
		for i, v := range xs[lo:hi] {
			args[i] = v
		}
		res, err := strFn(goja.Undefined(), vm.ToValue(args))
		if err != nil {
			t.Fatal(err)
		}
		var got []string
		if err := vm.ExportTo(res, &got); err != nil {
			t.Fatal(err)
		}
		for i, v := range xs[lo:hi] {
			if f := rt.toString(v); got[i] != f {
				bad++
				if bad <= 10 {
					t.Errorf("%v: goja %s, frozen %s", strconv.FormatFloat(v, 'g', -1, 64), got[i], f)
				}
			}
		}
	}
	if bad > 0 {
		t.Errorf("%d of %d doubles print differently under goja and the frozen host", bad, len(xs))
	}
}

// ToInt32 and ToUint32 (`x | 0`, `x >>> 0`, `~x`, a shift count) are a modulo 2^32
// in both grammar hosts, also for a magnitude outside the int64 range, where Go's
// float64-to-int64 conversion is implementation defined: the frozen host reduces
// with rt.toInt32, goja with its patched modUint32 (third_party/goja/runtime.go).
func TestToInt32AgreesAcrossHosts(t *testing.T) {
	vm := goja.New()
	fm := map[string]r.Object{}
	NewCommonScript(vm, &fm, true)
	if _, err := vm.RunString(`function __i32(xs) { return xs.map(function (x) { return [x | 0, x >>> 0, ~x, 1 << x]; }); }`); err != nil {
		t.Fatal(err)
	}
	fn, _ := goja.AssertFunction(vm.Get("__i32"))
	var xs []float64
	for e := 0; e <= 1023; e++ {
		for _, d := range []float64{-1, 0, 1, 3} {
			v := math.Ldexp(1, e) + d
			xs = append(xs, v, -v)
		}
	}
	xs = append(xs, 1e21, 1e19, 1e20, math.MaxFloat64, 0.5, -0.5, 4294967295.5, math.Ldexp(1, 64)+math.Ldexp(1, 33))
	rnd := rand.New(rand.NewSource(1))
	for i := 0; i < 20000; i++ {
		v := math.Float64frombits(rnd.Uint64())
		if !math.IsNaN(v) && !math.IsInf(v, 0) {
			xs = append(xs, v)
		}
	}
	args := make([]interface{}, len(xs))
	for i, v := range xs {
		args[i] = v
	}
	res, err := fn(goja.Undefined(), vm.ToValue(args))
	if err != nil {
		t.Fatal(err)
	}
	var got [][]float64
	if err := vm.ExportTo(res, &got); err != nil {
		t.Fatal(err)
	}
	rt := newJSRT(nil)
	bad := 0
	for i, v := range xs {
		i32 := rt.toInt32(v)
		want := []float64{float64(i32), float64(uint32(i32)), float64(^i32), float64(int32(1) << (uint32(i32) & 31))}
		for j := range want {
			if got[i][j] != want[j] {
				bad++
				if bad <= 10 {
					t.Errorf("%v op %d: goja %v, frozen %v", strconv.FormatFloat(v, 'g', -1, 64), j, got[i][j], want[j])
				}
			}
		}
	}
	if bad > 0 {
		t.Errorf("%d of %d results differ between goja and the frozen host", bad, 4*len(xs))
	}
}
