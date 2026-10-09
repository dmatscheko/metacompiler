package abnf

import (
	"math"
	"math/big"
)

// Python's `math` module - the GO TWIN half, i.e. what `mec
// languages/python-to-llvm-ir.abnf prog.py` (llvm.Run) executes. The other two
// halves are js_pymath_module in languages/lib/python-rt.metajs (the native
// binary) and hostGlobals["math"] in languages/python-interpreter.abnf.
//
// WHY A FILE OF ITS OWN. Python has no abnf/jsrtpython.go: its Go-twin arms are
// spread over abnf/jsrt.go and abnf/jsrtregexpy.go. A new module is additive by
// nature, so it goes in a python-only file rather than into the shared one,
// exactly as abnf/pystrmethod.go does for the str surface.
//
// WHAT IS IN SCOPE, AND WHY THAT LINE. Everything here is a pure function of
// doubles or of integers, which is what this value model can carry exactly and
// what the C floor already offers through Math (runtime.c seeds thirty-one
// members, host ids 51-93). DELIBERATELY ABSENT: the complex-number surface
// (cmath's job), the statistics helpers (fsum, prod, dist, comb, perm - each is
// a real algorithm rather than a rename, and none is in the corpus), nextafter /
// ulp / ldexp / frexp / modf (they read a double's BIT LAYOUT, which is manual
// chapter 7.7's rule about d_frexp / d_ldexp: the floor owns those and MetaJS
// cannot express them). Adding any of them later is one arm in three engines.
//
// THE TYPE OF EACH ANSWER IS PART OF THE SPECIFICATION and CPython is not
// uniform about it: sqrt/exp/log/pow/fabs/hypot/atan2 and the trigonometric
// family all answer a FLOAT (so math.pow(2, 10) is 1024.0, not 1024), while
// floor/ceil/trunc/gcd/factorial answer an INT and isnan/isinf/isfinite answer a
// bool. A python float here is the *jsPyFlo box (manual 7.6); a plain float64 is
// an int. Getting that wrong is invisible until something prints.
//
// NOT A MODULE OBJECT IN CPYTHON'S SENSE: `import math` binds nothing in this
// project's flat import model, so `math` is a global that is always there - the
// same deviation `re` has had all along, stated in both grammars' headers.

// pyMathIsMod says whether a value is THE math module object, so the method-call
// dispatcher in abnf/jsrtregexpy.go can route math.sqrt(2) at it.
func pyMathIsMod(v interface{}) bool {
	o, ok := v.(*jsObject)
	if !ok {
		return false
	}
	n, has := o.props["__pymod"]
	return has && n == "math"
}

// pyMathFlo boxes a double as a python float. Every arm that CPython documents
// as returning a float goes through it.
func pyMathFlo(f float64) interface{} { return &jsPyFlo{f: f} }

func pyMathArg(rt *jsrt, args []interface{}, i int) float64 {
	if i >= len(args) {
		rt.fail("math: missing argument")
	}
	return rt.pyToF(args[i])
}

// pyMathInt is CPython's rule for the integral answers: floor/ceil/trunc raise
// on a non-finite operand rather than answering an infinity, because the result
// must be an int and there is no such int. CPython splits the two - an INFINITY
// is an OverflowError and a NaN a ValueError, with different texts.
func pyMathInt(rt *jsrt, f float64) interface{} {
	if math.IsNaN(f) {
		panic(&jsThrown{value: rt.pyExcInstance("ValueError",
			"cannot convert float NaN to integer")})
	}
	if math.IsInf(f, 0) {
		panic(&jsThrown{value: rt.pyExcInstance("OverflowError",
			"cannot convert float infinity to integer")})
	}
	return f
}

// pyMathDom raises CPython 3.14's own domain-error texts, which quote the
// offending operand as a FLOAT repr ("expected a nonnegative input, got -1.0").
// Taken from the oracle rather than invented, which is what makes these rows
// assertable against python3.
func pyMathDom(rt *jsrt, kind string, v float64) {
	got := rt.pyRepr(pyMathFlo(v))
	switch kind {
	case "nonneg":
		panic(&jsThrown{value: rt.pyExcInstance("ValueError", "expected a nonnegative input, got "+got)})
	case "pos":
		panic(&jsThrown{value: rt.pyExcInstance("ValueError", "expected a positive input, got "+got)})
	case "unit":
		panic(&jsThrown{value: rt.pyExcInstance("ValueError",
			"expected a number in range from -1 up to 1, got "+got)})
	}
	panic(&jsThrown{value: rt.pyExcInstance("ValueError", "expected argument value > -1, got "+got)})
}

// pyMathGcd is Euclid on the absolute values; CPython's math.gcd(0, 0) is 0.
func pyMathGcd(a, b float64) float64 {
	a = math.Abs(math.Trunc(a))
	b = math.Abs(math.Trunc(b))
	for b != 0 {
		a, b = b, math.Mod(a, b)
	}
	return a
}

// pyMathHas says whether `name` is a function this math module offers. It exists
// for the ATTRIBUTE read (`f = math.sqrt`) and for hasattr; the call path does
// not need it, because pyMathCall's own fall-through raises the AttributeError.
// Keep in step with pyMHas in languages/lib/python-rt.metajs and pyMathNames in
// languages/python-interpreter.abnf.
func pyMathHas(name string) bool {
	switch name {
	case "sqrt", "exp", "expm1", "log", "log1p", "log2", "log10", "pow",
		"fabs", "fmod", "copysign", "atan2", "hypot",
		"sin", "cos", "tan", "asin", "acos", "atan", "sinh", "cosh", "tanh",
		"degrees", "radians", "floor", "ceil", "trunc",
		"gcd", "lcm", "factorial", "isnan", "isinf", "isfinite", "isclose":
		return true
	}
	return false
}

func pyMathModule(rt *jsrt) *jsObject {
	o := newJSObject()
	o.set("__pymod", "math")
	o.set("pi", pyMathFlo(math.Pi))
	o.set("e", pyMathFlo(math.E))
	o.set("tau", pyMathFlo(2*math.Pi))
	o.set("inf", pyMathFlo(math.Inf(1)))
	o.set("nan", pyMathFlo(math.NaN()))
	return o
}

// pyMathCall is math.<name>(args...). The dispatcher is by name rather than by
// twenty-five stored closures because the same shape has to exist in MetaJS
// layer 2, where a closure per member would be twenty-five floor cells held
// forever.
func (rt *jsrt) pyMathCall(name string, args []interface{}) interface{} {
	switch name {
	// ---- float in, float out
	case "sqrt":
		x := pyMathArg(rt, args, 0)
		if x < 0 {
			pyMathDom(rt, "nonneg", x)
		}
		return pyMathFlo(math.Sqrt(x))
	case "exp":
		return pyMathFlo(math.Exp(pyMathArg(rt, args, 0)))
	case "expm1":
		return pyMathFlo(math.Expm1(pyMathArg(rt, args, 0)))
	case "log":
		x := pyMathArg(rt, args, 0)
		if x <= 0 {
			pyMathDom(rt, "pos", x)
		}
		// math.log(x, base) is the two-argument form CPython offers.
		if len(args) > 1 {
			// math.log(x, 1) divides by log(1) == 0: CPython raises
			// ZeroDivisionError, not a domain error.
			lb := math.Log(pyMathArg(rt, args, 1))
			if lb == 0 {
				panic(&jsThrown{value: rt.pyExcInstance("ZeroDivisionError", "division by zero")})
			}
			return pyMathFlo(math.Log(x) / lb)
		}
		return pyMathFlo(math.Log(x))
	case "log1p":
		l1 := pyMathArg(rt, args, 0)
		if l1 <= -1 {
			pyMathDom(rt, "gt", l1)
		}
		return pyMathFlo(math.Log1p(l1))
	case "log2":
		l2 := pyMathArg(rt, args, 0)
		if l2 <= 0 {
			pyMathDom(rt, "pos", l2)
		}
		return pyMathFlo(math.Log2(l2))
	case "log10":
		l10 := pyMathArg(rt, args, 0)
		if l10 <= 0 {
			pyMathDom(rt, "pos", l10)
		}
		return pyMathFlo(math.Log10(l10))
	case "pow":
		return pyMathFlo(math.Pow(pyMathArg(rt, args, 0), pyMathArg(rt, args, 1)))
	case "fabs":
		return pyMathFlo(math.Abs(pyMathArg(rt, args, 0)))
	case "fmod":
		return pyMathFlo(math.Mod(pyMathArg(rt, args, 0), pyMathArg(rt, args, 1)))
	case "copysign":
		return pyMathFlo(math.Copysign(pyMathArg(rt, args, 0), pyMathArg(rt, args, 1)))
	case "atan2":
		return pyMathFlo(math.Atan2(pyMathArg(rt, args, 0), pyMathArg(rt, args, 1)))
	case "hypot":
		// VARIADIC since 3.8, and scaled so hypot(1e300, 1e300) does not overflow.
		mx := 0.0
		for i := range args {
			a := math.Abs(rt.pyToF(args[i]))
			if math.IsInf(a, 0) {
				return pyMathFlo(math.Inf(1))
			}
			if a > mx {
				mx = a
			}
		}
		if mx == 0 {
			return pyMathFlo(0)
		}
		sum := 0.0
		for i := range args {
			t := rt.pyToF(args[i]) / mx
			sum += t * t
		}
		return pyMathFlo(mx * math.Sqrt(sum))
	case "sin":
		return pyMathFlo(math.Sin(pyMathArg(rt, args, 0)))
	case "cos":
		return pyMathFlo(math.Cos(pyMathArg(rt, args, 0)))
	case "tan":
		return pyMathFlo(math.Tan(pyMathArg(rt, args, 0)))
	case "asin":
		as1 := pyMathArg(rt, args, 0)
		if as1 < -1 || as1 > 1 {
			pyMathDom(rt, "unit", as1)
		}
		return pyMathFlo(math.Asin(as1))
	case "acos":
		ac1 := pyMathArg(rt, args, 0)
		if ac1 < -1 || ac1 > 1 {
			pyMathDom(rt, "unit", ac1)
		}
		return pyMathFlo(math.Acos(ac1))
	case "atan":
		return pyMathFlo(math.Atan(pyMathArg(rt, args, 0)))
	case "sinh":
		return pyMathFlo(math.Sinh(pyMathArg(rt, args, 0)))
	case "cosh":
		return pyMathFlo(math.Cosh(pyMathArg(rt, args, 0)))
	case "tanh":
		return pyMathFlo(math.Tanh(pyMathArg(rt, args, 0)))
	case "degrees":
		return pyMathFlo(pyMathArg(rt, args, 0) * (180 / math.Pi))
	case "radians":
		return pyMathFlo(pyMathArg(rt, args, 0) * (math.Pi / 180))
	// ---- float in, INT out
	case "floor":
		return pyMathInt(rt, math.Floor(pyMathArg(rt, args, 0)))
	case "ceil":
		return pyMathInt(rt, math.Ceil(pyMathArg(rt, args, 0)))
	case "trunc":
		return pyMathInt(rt, math.Trunc(pyMathArg(rt, args, 0)))
	case "gcd":
		if len(args) == 0 {
			return 0.0
		}
		g := math.Abs(math.Trunc(rt.pyToF(args[0])))
		for i := 1; i < len(args); i++ {
			g = pyMathGcd(g, rt.pyToF(args[i]))
		}
		return g
	case "lcm":
		if len(args) == 0 {
			return 1.0
		}
		l := math.Abs(math.Trunc(rt.pyToF(args[0])))
		for i := 1; i < len(args); i++ {
			b := math.Abs(math.Trunc(rt.pyToF(args[i])))
			if l == 0 || b == 0 {
				return 0.0
			}
			l = l / pyMathGcd(l, b) * b
		}
		return l
	case "factorial":
		n := math.Trunc(pyMathArg(rt, args, 0))
		if n < 0 || n != pyMathArg(rt, args, 0) {
			panic(&jsThrown{value: rt.pyExcInstance("ValueError",
				"factorial() not defined for negative values")})
		}
		// Beyond 20! the product leaves the exact-double range, so it is handed
		// to the arbitrary-precision integer this value model already carries -
		// which is what makes math.factorial(30) CPython's exact answer rather
		// than a rounded double.
		if n <= 20 {
			r := 1.0
			for i := 2.0; i <= n; i++ {
				r *= i
			}
			return r
		}
		acc := big.NewInt(1)
		for i := int64(2); i <= int64(n); i++ {
			acc.Mul(acc, big.NewInt(i))
		}
		return rt.pyBigNarrow(acc)
	// ---- predicates
	case "isnan":
		return math.IsNaN(pyMathArg(rt, args, 0))
	case "isinf":
		return math.IsInf(pyMathArg(rt, args, 0), 0)
	case "isfinite":
		f := pyMathArg(rt, args, 0)
		return !math.IsNaN(f) && !math.IsInf(f, 0)
	case "isclose":
		// CPython's default tolerances, and its rule that two infinities of the
		// same sign are close while a NaN is close to nothing.
		a, b := pyMathArg(rt, args, 0), pyMathArg(rt, args, 1)
		if a == b {
			return true
		}
		if math.IsInf(a, 0) || math.IsInf(b, 0) || math.IsNaN(a) || math.IsNaN(b) {
			return false
		}
		d := math.Abs(a - b)
		return d <= 1e-09*math.Max(math.Abs(a), math.Abs(b)) || d <= 0.0
	}
	panic(&jsThrown{value: rt.pyExcInstance("AttributeError",
		"module 'math' has no attribute '"+name+"'")})
}

func init() {
	rxExtraExterns = append(rxExtraExterns, func(rt *jsrt, m map[string]func(args []uint64) uint64) {
		m["js_pymath_module"] = func(a []uint64) uint64 { return rt.wrap(pyMathModule(rt)) }
	})
}
