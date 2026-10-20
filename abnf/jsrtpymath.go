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
// members, host ids 51-93). The integer helpers (isqrt, comb, perm, gcd, lcm)
// work on math/big. DELIBERATELY ABSENT: the complex-number surface (cmath's
// job), dist, nextafter / ulp / ldexp / frexp / modf (they read a double's BIT
// LAYOUT, which is manual chapter 7.7's rule about d_frexp / d_ldexp: the floor
// owns those and MetaJS cannot express them). Adding any of them later is one
// arm in three engines.
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
	if f == 0 {
		return 0.0 // An int has no negative zero: math.ceil(-0.5) is 0.
	}
	return f
}

// pyMathDom raises CPython 3.14's own domain-error texts, which quote the
// offending operand as a FLOAT repr ("expected a nonnegative input, got -1.0").
// Taken from the oracle rather than invented, which is what makes these rows
// assertable against python3.
// pyMathRange is CPython's math_1 overflow rule for exp, expm1, sinh and cosh: an
// infinite result from a finite argument is OverflowError("math range error").
// Twin of pyMRange in languages/lib/python-rt.metajs and
// languages/python-interpreter.abnf.
func pyMathRange(rt *jsrt, r, x float64) interface{} {
	if math.IsInf(r, 0) && !math.IsInf(x, 0) {
		panic(&jsThrown{value: rt.pyExcInstance("OverflowError", "math range error")})
	}
	return pyMathFlo(r)
}

// pyMathPow is math.pow with CPython's errors for two finite operands: a NaN
// result ((-8) ** (1/3)) and 0 ** negative are ValueError("math domain error"),
// any other infinite result OverflowError("math range error").
func pyMathPow(rt *jsrt, x, y float64) interface{} {
	r := crPow(x, y)
	if !math.IsInf(x, 0) && !math.IsNaN(x) && !math.IsInf(y, 0) && !math.IsNaN(y) {
		if math.IsNaN(r) || (math.IsInf(r, 0) && x == 0) {
			panic(&jsThrown{value: rt.pyExcInstance("ValueError", "math domain error")})
		}
		if math.IsInf(r, 0) {
			panic(&jsThrown{value: rt.pyExcInstance("OverflowError", "math range error")})
		}
	}
	return pyMathFlo(r)
}

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
	case "finite":
		panic(&jsThrown{value: rt.pyExcInstance("ValueError", "expected a finite input, got "+got)})
	}
	panic(&jsThrown{value: rt.pyExcInstance("ValueError", "expected argument value > -1, got "+got)})
}

// pyMathLog2 is log2 of x as log(m) / log(2) + e over x = m * 2^e, rewritten
// for x >= 1 as CPython's own fallback does so the sum does not cancel just
// above 1.0, where math.Log2's Log(frac)/Ln2 + exp loses every digit. An
// infinity or a NaN is itself. Twin of pynLog2 in languages/lib/python-num.js.
func pyMathLog2(x float64) float64 {
	if math.IsInf(x, 0) || math.IsNaN(x) {
		return x
	}
	m, e := math.Frexp(x)
	if x >= 1 {
		return math.Log(2*m)/math.Log(2) + float64(e-1)
	}
	return math.Log(m)/math.Log(2) + float64(e)
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

// pyModHas says whether `name` is a function the runtime module `mod` (math, re
// or sys) offers. It exists for the ATTRIBUTE read (`f = math.sqrt`, `from re
// import sub`) and for hasattr; the call path does not need it, because each
// module's call dispatch raises the AttributeError itself. Keep in step with
// pyMHas in languages/lib/python-rt.metajs and pyModHas in
// languages/python-interpreter.abnf.
func pyModHas(mod, name string) bool {
	switch mod {
	case "math":
		switch name {
		case "sqrt", "cbrt", "exp", "expm1", "log", "log1p", "log2", "log10", "pow",
			"fabs", "fmod", "copysign", "atan2", "hypot",
			"sin", "cos", "tan", "asin", "acos", "atan", "sinh", "cosh", "tanh",
			"degrees", "radians", "floor", "ceil", "trunc",
			"gcd", "lcm", "factorial", "isnan", "isinf", "isfinite", "isclose",
			"isqrt", "comb", "perm", "prod", "fsum",
			"frexp", "ldexp", "modf", "ulp", "nextafter", "dist", "remainder":
			return true
		}
	case "re":
		switch name {
		case "compile", "search", "match", "fullmatch", "findall", "finditer",
			"sub", "subn", "split", "escape", "purge":
			return true
		}
	case "sys":
		switch name {
		case "exit", "get_int_max_str_digits", "set_int_max_str_digits":
			return true
		}
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

// pyMathKwArgs is math.NAME's arguments with its keyword arguments placed: only
// isclose takes any, rel_tol and abs_tol, which land third and fourth. Twin of
// pyMathKwArgs in python-interpreter.abnf and lib/python-rt.metajs.
func (rt *jsrt) pyMathKwArgs(name string, args []interface{}, kw interface{}) []interface{} {
	keys, vals, ok := dictParts(kw)
	if !ok || len(keys.elems) == 0 || !pyModHas("math", name) {
		return args
	}
	if name == "prod" && len(keys.elems) == 1 && keys.elems[0] == "start" {
		return []interface{}{argAt(args, 0), vals.elems[0]}
	}
	if name != "isclose" {
		rt.pyRaise("TypeError", "math."+name+"() takes no keyword arguments")
	}
	out := []interface{}{argAt(args, 0), argAt(args, 1), pyMathFlo(1e-09), float64(0)}
	for i, k := range keys.elems {
		switch rt.toString(k) {
		case "rel_tol":
			out[2] = vals.elems[i]
		case "abs_tol":
			out[3] = vals.elems[i]
		default:
			rt.pyRaise("TypeError", "isclose() got an unexpected keyword argument '"+rt.toString(k)+"'")
		}
	}
	return out
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
	case "cbrt":
		return pyMathFlo(math.Cbrt(pyMathArg(rt, args, 0)))
	case "exp":
		x := pyMathArg(rt, args, 0)
		return pyMathRange(rt, math.Exp(x), x)
	case "expm1":
		x := pyMathArg(rt, args, 0)
		return pyMathRange(rt, math.Expm1(x), x)
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
		return pyMathFlo(pyMathLog2(l2))
	case "log10":
		l10 := pyMathArg(rt, args, 0)
		if l10 <= 0 {
			pyMathDom(rt, "pos", l10)
		}
		return pyMathFlo(math.Log10(l10))
	case "pow":
		return pyMathPow(rt, pyMathArg(rt, args, 0), pyMathArg(rt, args, 1))
	case "fabs":
		return pyMathFlo(math.Abs(pyMathArg(rt, args, 0)))
	case "fmod", "remainder":
		// C's fmod and IEEE's remainder; a NaN from two non-NaN arguments is
		// CPython's domain error.
		x, y := pyMathArg(rt, args, 0), pyMathArg(rt, args, 1)
		r := math.Mod(x, y)
		if name == "remainder" {
			r = math.Remainder(x, y)
		}
		if math.IsNaN(r) && !math.IsNaN(x) && !math.IsNaN(y) {
			rt.pyRaise("ValueError", "math domain error")
		}
		return pyMathFlo(r)
	case "copysign":
		return pyMathFlo(math.Copysign(pyMathArg(rt, args, 0), pyMathArg(rt, args, 1)))
	case "atan2":
		return pyMathFlo(fdAtan2(pyMathArg(rt, args, 0), pyMathArg(rt, args, 1)))
	case "hypot":
		xs := make([]float64, len(args))
		for i := range args {
			xs[i] = rt.pyToF(args[i])
		}
		return pyMathFlo(pyMathNorm(xs))
	case "dist":
		p, q := rt.pyElemsOf(argAt(args, 0)), rt.pyElemsOf(argAt(args, 1))
		if len(p) != len(q) {
			rt.pyRaise("ValueError", "both points must have the same number of dimensions")
		}
		xs := make([]float64, len(p))
		for i := range p {
			xs[i] = rt.pyToF(p[i]) - rt.pyToF(q[i])
		}
		return pyMathFlo(pyMathNorm(xs))
	case "frexp":
		m, e := math.Frexp(pyMathArg(rt, args, 0))
		return pyTup(&jsArray{elems: []interface{}{pyMathFlo(m), float64(e)}})
	case "ldexp":
		x := pyMathArg(rt, args, 0)
		r := math.Ldexp(x, rt.pyMathLdexpExp(argAt(args, 1)))
		if math.IsInf(r, 0) && !math.IsInf(x, 0) {
			rt.pyRaise("OverflowError", "math range error")
		}
		return pyMathFlo(r)
	case "modf":
		x := pyMathArg(rt, args, 0)
		ip, fp := math.Modf(x)
		if math.IsInf(x, 0) {
			fp = math.Copysign(0, x)
		}
		return pyTup(&jsArray{elems: []interface{}{pyMathFlo(fp), pyMathFlo(ip)}})
	case "ulp":
		x := math.Abs(pyMathArg(rt, args, 0))
		if math.IsNaN(x) || math.IsInf(x, 0) {
			return pyMathFlo(x)
		}
		if x == math.MaxFloat64 {
			return pyMathFlo(x - math.Nextafter(x, 0))
		}
		return pyMathFlo(math.Nextafter(x, math.Inf(1)) - x)
	case "nextafter":
		// x == y answers y, so nextafter(-0.0, 0.0) is 0.0 (Go's answers x).
		x, y := pyMathArg(rt, args, 0), pyMathArg(rt, args, 1)
		if x == y {
			return pyMathFlo(y)
		}
		return pyMathFlo(math.Nextafter(x, y))
	case "sin", "cos", "tan":
		// An infinity is CPython's domain error for the three.
		ta := pyMathArg(rt, args, 0)
		if math.IsInf(ta, 0) {
			pyMathDom(rt, "finite", ta)
		}
		switch name {
		case "sin":
			return pyMathFlo(fdSin(ta))
		case "cos":
			return pyMathFlo(fdCos(ta))
		}
		return pyMathFlo(fdTan(ta))
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
		x := pyMathArg(rt, args, 0)
		return pyMathRange(rt, math.Sinh(x), x)
	case "cosh":
		x := pyMathArg(rt, args, 0)
		return pyMathRange(rt, math.Cosh(x), x)
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
		for _, a := range args {
			if _, isBig := a.(*jsBigInt); isBig {
				return rt.pyBigNarrow(rt.pyMathGcdBig(args, true))
			}
		}
		g := math.Abs(math.Trunc(rt.pyToF(args[0])))
		for i := 1; i < len(args); i++ {
			g = pyMathGcd(g, rt.pyToF(args[i]))
		}
		return g
	case "lcm":
		// Always exact: two small operands can have a multiple past 2^53.
		if len(args) == 0 {
			return 1.0
		}
		return rt.pyBigNarrow(rt.pyMathGcdBig(args, false))
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
		// CPython's tolerances (rel_tol and abs_tol, placed by pyMathKwArgs),
		// and its rule that two infinities of the same sign are close while a
		// NaN is close to nothing.
		a, b := pyMathArg(rt, args, 0), pyMathArg(rt, args, 1)
		rel, abs := 1e-09, 0.0
		if len(args) > 2 {
			rel = pyMathArg(rt, args, 2)
		}
		if len(args) > 3 {
			abs = pyMathArg(rt, args, 3)
		}
		if rel < 0 || abs < 0 {
			rt.pyRaise("ValueError", "tolerances must be non-negative")
		}
		if a == b {
			return true
		}
		if math.IsInf(a, 0) || math.IsInf(b, 0) || math.IsNaN(a) || math.IsNaN(b) {
			return false
		}
		d := math.Abs(a - b)
		return d <= math.Abs(rel*b) || d <= math.Abs(rel*a) || d <= abs
	case "isqrt":
		n := rt.pyMathIntArg(argAt(args, 0))
		if n.Sign() < 0 {
			rt.pyRaise("ValueError", "isqrt() argument must be nonnegative")
		}
		return rt.pyBigNarrow(new(big.Int).Sqrt(n))
	case "comb", "perm":
		n := rt.pyMathIntArg(argAt(args, 0))
		k := n
		if a := argAt(args, 1); name == "comb" || !isUndefOrNull(a) {
			k = rt.pyMathIntArg(a)
		} else if n.Sign() < 0 {
			// perm(n) is factorial(n), its error text included.
			rt.pyRaise("ValueError", "factorial() not defined for negative values")
		}
		return rt.pyBigNarrow(rt.pyMathComb(n, k, name == "comb"))
	case "prod":
		var acc interface{} = 1.0
		if len(args) > 1 {
			acc = args[1]
		}
		next := rt.pyStepper(argAt(args, 0))
		for x, ok := next(); ok; x, ok = next() {
			acc = rt.pyExt("js_pybin", "*", acc, x)
		}
		return acc
	case "fsum":
		var fs []float64
		next := rt.pyStepper(argAt(args, 0))
		for x, ok := next(); ok; x, ok = next() {
			fs = append(fs, pyMathArg(rt, []interface{}{x}, 0))
		}
		return pyMathFlo(rt.pyMathFsum(fs))
	}
	panic(&jsThrown{value: rt.pyExcInstance("AttributeError",
		"module 'math' has no attribute '"+name+"'")})
}

// pyMathNorm is hypot / dist over the coordinates xs: an infinity wins over a
// NaN, as CPython's math_hypot decides, then CPython's vector_norm. Twin of
// pyMNorm in languages/python-interpreter.abnf and lib/python-rt.metajs, and of
// pynVecNorm in lib/python-num.js, which use Dekker's product for the fma.
func pyMathNorm(xs []float64) float64 {
	mx, nan := 0.0, false
	abs := make([]float64, len(xs))
	for i, x := range xs {
		abs[i] = math.Abs(x)
		if math.IsNaN(x) {
			nan = true
		} else if abs[i] > mx {
			mx = abs[i]
		}
	}
	if math.IsInf(mx, 0) {
		return mx
	}
	if nan {
		return math.NaN()
	}
	return pyVecNorm(abs, mx)
}

func pyVecNorm(vec []float64, mx float64) float64 {
	if mx == 0 || len(vec) <= 1 {
		return mx
	}
	_, me := math.Frexp(mx)
	if me < -1022 {
		big := make([]float64, len(vec))
		for i, v := range vec {
			big[i] = math.Ldexp(v, 1022)
		}
		return pyVecNorm(big, math.Ldexp(mx, 1022)) * math.Ldexp(1, -1022)
	}
	// The float64() conversions keep Go from fusing a product into the add
	// that follows it, which the other engines cannot do.
	scale := math.Ldexp(1, -me)
	csum, frac1, frac2 := 1.0, 0.0, 0.0
	for _, v := range vec {
		x := float64(v * scale)
		hi := float64(x * x)
		lo := math.FMA(x, x, -hi)
		s := csum + hi
		frac2 += (csum - s) + hi
		csum = s
		frac1 += lo
	}
	h := math.Sqrt(csum - 1 + (frac1 + frac2))
	phi := float64(-h * h)
	plo := math.FMA(-h, h, -phi)
	s := csum + phi
	frac2 += (csum - s) + phi
	csum = s
	frac1 += plo
	c := csum - 1 + (frac1 + frac2)
	h += c / (2 * h)
	return h / scale
}

// pyMathLdexpExp is ldexp's exponent: an int (a bool is one), clamped to where
// every finite x has overflowed or underflowed. Twin of pyMLdexpExp in
// languages/python-interpreter.abnf and lib/python-rt.metajs.
func (rt *jsrt) pyMathLdexpExp(v interface{}) int {
	if b, ok := v.(bool); ok {
		if b {
			return 1
		}
		return 0
	}
	if _, isFlo := v.(*jsPyFlo); !isFlo {
		if bi, ok := pyBigOperand(v); ok {
			if bi.Cmp(big.NewInt(2200)) > 0 {
				return 2200
			}
			if bi.Cmp(big.NewInt(-2200)) < 0 {
				return -2200
			}
			return int(bi.Int64())
		}
	}
	rt.pyRaise("TypeError", "Expected an int as second argument to ldexp.")
	return 0
}

// pyMathGcdBig is gcd / lcm once an operand is past the exact-double range.
// Twin of pyMGcdBig in languages/python-interpreter.abnf and
// lib/python-rt.metajs.
func (rt *jsrt) pyMathGcdBig(args []interface{}, gcd bool) *big.Int {
	acc := new(big.Int).Abs(rt.pyMathIntArg(args[0]))
	for _, a := range args[1:] {
		b := new(big.Int).Abs(rt.pyMathIntArg(a))
		g := new(big.Int).GCD(nil, nil, acc, b)
		switch {
		case gcd:
			acc = g
		case g.Sign() == 0:
			acc = new(big.Int)
		default:
			acc = new(big.Int).Mul(new(big.Int).Quo(acc, g), b)
		}
	}
	return acc
}

// pyMathIntArg is the integer argument of isqrt / comb / perm: an int or a
// bool. Twin of pyMIntArg in languages/python-interpreter.abnf and
// lib/python-rt.metajs.
func (rt *jsrt) pyMathIntArg(v interface{}) *big.Int {
	if _, isFlo := v.(*jsPyFlo); !isFlo {
		if b, ok := pyBigOperand(v); ok {
			return b
		}
	}
	rt.pyRaise("TypeError", "'"+pyTypeName(rt, v)+"' object cannot be interpreted as an integer")
	return nil
}

// pyMathComb is comb(n, k) (choose) or perm(n, k) (choose, ordered). Twin of
// pyMComb in languages/python-interpreter.abnf and lib/python-rt.metajs.
func (rt *jsrt) pyMathComb(n, k *big.Int, choose bool) *big.Int {
	if n.Sign() < 0 {
		rt.pyRaise("ValueError", "n must be a non-negative integer")
	}
	if k.Sign() < 0 {
		rt.pyRaise("ValueError", "k must be a non-negative integer")
	}
	if k.Cmp(n) > 0 {
		return new(big.Int)
	}
	steps := k
	if rest := new(big.Int).Sub(n, k); choose && rest.Cmp(k) < 0 {
		steps = rest
	}
	r := big.NewInt(1)
	i := new(big.Int)
	for i.Cmp(steps) < 0 {
		r.Mul(r, new(big.Int).Sub(n, i))
		i.Add(i, big.NewInt(1))
		if choose {
			r.Quo(r, i)
		}
	}
	return r
}

// pyMathFsum is Shewchuk's exactly rounded sum as CPython's math_fsum computes
// it. Twin of pyMFsum in languages/python-interpreter.abnf and
// lib/python-rt.metajs, which explain it.
func (rt *jsrt) pyMathFsum(xs []float64) float64 {
	var partials []float64
	special, infSum := 0.0, 0.0
	for _, x := range xs {
		xsave := x
		n := 0
		for _, y := range partials {
			if math.Abs(x) < math.Abs(y) {
				x, y = y, x
			}
			hi := x + y
			lo := y - (hi - x)
			if lo != 0 {
				partials[n] = lo
				n++
			}
			x = hi
		}
		partials = partials[:n]
		if x != 0 {
			if math.IsInf(x, 0) || math.IsNaN(x) {
				if !math.IsInf(xsave, 0) && !math.IsNaN(xsave) {
					rt.pyRaise("OverflowError", "intermediate overflow in fsum")
				}
				if math.IsInf(xsave, 0) {
					infSum += xsave
				}
				special += xsave
				partials = partials[:0]
			} else {
				partials = append(partials, x)
			}
		}
	}
	if special != 0 {
		if math.IsNaN(infSum) {
			rt.pyRaise("ValueError", "-inf + inf in fsum")
		}
		return special
	}
	m := len(partials)
	if m == 0 {
		return 0
	}
	m--
	hi, lo := partials[m], 0.0
	for m > 0 {
		x := hi
		m--
		y := partials[m]
		hi = x + y
		lo = y - (hi - x)
		if lo != 0 {
			break
		}
	}
	if m > 0 && ((lo < 0 && partials[m-1] < 0) || (lo > 0 && partials[m-1] > 0)) {
		y := lo * 2
		x := hi + y
		if y == x-hi {
			hi = x
		}
	}
	return hi
}

func init() {
	rxExtraExterns = append(rxExtraExterns, func(rt *jsrt, m map[string]func(args []uint64) uint64) {
		m["js_pymath_module"] = func(a []uint64) uint64 { return rt.wrap(pyMathModule(rt)) }
		m["js_pysys_module"] = func(a []uint64) uint64 {
			o := newJSObject()
			o.set("__pymod", "sys")
			return rt.wrap(o)
		}
	})
}
