package abnf

import (
	"math"
	"time"
)

// kotlin.math for the compiler half: the top-level functions and constants, the
// Double/Float extension members (pow, roundToInt, roundToLong) and
// System.currentTimeMillis / nanoTime. Kotlin/JVM implements the functions on
// java.lang.Math; the transcendentals here are StrictMath's fdlibm, as java's own
// Math is in this project (jvMathMembers). Twin of kMathFn in
// kotlin-interpreter.abnf and k4MathFn in languages/lib/kotlin-rt.metajs.

// ktMathNames are the names buildMain may declare from js_ktglobal.
var ktMathNames = map[string]bool{
	"sqrt": true, "cbrt": true, "exp": true, "expm1": true, "ln": true, "ln1p": true,
	"log10": true, "log2": true, "log": true, "sin": true, "cos": true, "tan": true,
	"asin": true, "acos": true, "atan": true, "atan2": true, "sinh": true, "cosh": true,
	"tanh": true, "asinh": true, "acosh": true, "atanh": true, "hypot": true,
	"floor": true, "ceil": true, "truncate": true, "round": true, "sign": true,
	"PI": true, "E": true,
}

// MathJVM.kt's constants: LN2 is Math.log(2.0); the Taylor bounds are the square
// root of ulp(1.0) (2^-26), its square root (2^-13) and their reciprocals.
var ktLN2 = fdLog(2)

const (
	ktTaylor2  = 1.0 / 67108864
	ktTaylorN  = 1.0 / 8192
	ktUTaylor2 = 67108864.0
	ktUTaylorN = 8192.0
)

func ktAsinh(x float64) float64 {
	switch {
	case x >= ktTaylorN:
		if x > ktUTaylorN {
			if x > ktUTaylor2 {
				return fdLog(x) + ktLN2
			}
			return fdLog(float64(x*2) + 1/(x*2))
		}
		return fdLog(x + math.Sqrt(float64(x*x)+1))
	case x <= -ktTaylorN:
		return -ktAsinh(-x)
	}
	r := x
	if math.Abs(x) >= ktTaylor2 {
		r -= x * x * x / 6
	}
	return r
}

func ktAcosh(x float64) float64 {
	switch {
	case x < 1:
		return math.NaN()
	case x > ktUTaylor2:
		return fdLog(x) + ktLN2
	case x-1 >= ktTaylorN:
		return fdLog(x + math.Sqrt(float64(x*x)-1))
	}
	y := math.Sqrt(x - 1)
	r := y
	if y >= ktTaylor2 {
		r -= y * y * y / 12
	}
	return math.Sqrt(2) * r
}

func ktAtanh(x float64) float64 {
	if math.Abs(x) < ktTaylorN {
		r := x
		if math.Abs(x) > ktTaylor2 {
			r += x * x * x / 3
		}
		return r
	}
	return fdLog((1+x)/(1-x)) / 2
}

// ktMath1 is the one-argument function of the name, on doubles.
func ktMath1(name string, x float64) float64 {
	switch name {
	case "sqrt":
		return math.Sqrt(x)
	case "cbrt":
		return fdCbrt(x)
	case "exp":
		return fdExp(x)
	case "expm1":
		return fdExpm1(x)
	case "ln":
		return fdLog(x)
	case "ln1p":
		return fdLog1p(x)
	case "log10":
		return fdLog10(x)
	case "log2":
		return fdLog(x) / ktLN2
	case "sin":
		return fdSin(x)
	case "cos":
		return fdCos(x)
	case "tan":
		return fdTan(x)
	case "asin":
		return fdAsin(x)
	case "acos":
		return fdAcos(x)
	case "atan":
		return fdAtan(x)
	case "sinh":
		return fdSinh(x)
	case "cosh":
		return fdCosh(x)
	case "tanh":
		return fdTanh(x)
	case "asinh":
		return ktAsinh(x)
	case "acosh":
		return ktAcosh(x)
	case "atanh":
		return ktAtanh(x)
	case "floor":
		return math.Floor(x)
	case "ceil":
		return math.Ceil(x)
	case "truncate":
		if math.IsNaN(x) || math.IsInf(x, 0) {
			return x
		}
		if x > 0 {
			return math.Floor(x)
		}
		return math.Ceil(x)
	case "round":
		return math.RoundToEven(x)
	case "sign":
		if x > 0 {
			return 1
		}
		if x < 0 {
			return -1
		}
		return x
	}
	return math.NaN()
}

// ktMath2 is the two-argument function of the name. log(x, base) is NaN for a
// base that is not positive or is 1, as MathJVM.kt checks before dividing.
func ktMath2(name string, x, y float64) float64 {
	switch name {
	case "atan2":
		return fdAtan2(x, y)
	case "hypot":
		return fdHypot(x, y)
	case "pow":
		return fdPow(x, y)
	}
	if y <= 0 || y == 1 {
		return math.NaN()
	}
	return fdLog(x) / fdLog(y)
}

// ktMathRes boxes a result as the first operand's type: a Float overload answers a
// Float (the double result narrowed), every other operand the Double.
func ktMathRes(like interface{}, f float64) interface{} {
	if l, ok := like.(jsJFlo); ok && l.sty == floJavaF {
		return jsJFlo{f: jvmFround(f), sty: floJavaF}
	}
	return jsJFlo{f: f}
}

// ktMathFn answers the value of a kotlin.math name, or nil.
func ktMathFn(name string) interface{} {
	if !ktMathNames[name] {
		return nil
	}
	switch name {
	case "PI":
		return jsJFlo{f: math.Pi}
	case "E":
		return jsJFlo{f: math.E}
	case "atan2", "hypot", "log":
		return jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			a := argAt(args, 0)
			return ktMathRes(a, ktMath2(name, giFloat(rt, a), giFloat(rt, argAt(args, 1))))
		})
	}
	return jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
		a := argAt(args, 0)
		return ktMathRes(a, ktMath1(name, giFloat(rt, a)))
	})
}

// ktJavaMathNames are java.lang.Math's members that are a kotlin.math function of
// the same meaning (rint is kotlin's round, signum its sign).
var ktJavaMathNames = map[string]string{
	"sqrt": "sqrt", "cbrt": "cbrt", "exp": "exp", "expm1": "expm1", "log": "ln",
	"log10": "log10", "log1p": "ln1p", "sin": "sin", "cos": "cos", "tan": "tan",
	"asin": "asin", "acos": "acos", "atan": "atan", "sinh": "sinh", "cosh": "cosh",
	"tanh": "tanh", "floor": "floor", "ceil": "ceil", "rint": "round", "signum": "sign",
	"atan2": "atan2", "hypot": "hypot", "pow": "pow",
}

// ktJavaMath is java.lang.Math, which a Kotlin/JVM program reaches as `Math`: its
// functions answer a Double whatever the argument (Math.sqrt(9) is 3.0), round a
// Long (an Int for a Float) rounding half up, toRadians / toDegrees multiply by
// java's own constant, and abs/max/min keep the argument's type exactly as
// kotlin.math's do. Twin of kJavaMath in kotlin-interpreter.abnf and k4JavaMath in
// languages/lib/kotlin-rt.metajs.
func ktJavaMath(rt *jsrt) *jsObject {
	o := newJSObject()
	o.set("PI", jsJFlo{f: math.Pi})
	o.set("E", jsJFlo{f: math.E})
	for _, n := range []string{"abs", "max", "min"} {
		o.set(n, ktGlobalFn(rt, n))
	}
	for jn, kn := range ktJavaMathNames {
		kn := kn
		if kn == "atan2" || kn == "hypot" || kn == "pow" {
			o.set(jn, jsHostFunc(jn, func(rt *jsrt, this uint64, args []interface{}) interface{} {
				return jsJFlo{f: ktMath2(kn, giFloat(rt, argAt(args, 0)), giFloat(rt, argAt(args, 1)))}
			}))
			continue
		}
		o.set(jn, jsHostFunc(jn, func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return jsJFlo{f: ktMath1(kn, giFloat(rt, argAt(args, 0)))}
		}))
	}
	o.set("toRadians", jsHostFunc("toRadians", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return jsJFlo{f: giFloat(rt, argAt(args, 0)) * 0.017453292519943295}
	}))
	o.set("toDegrees", jsHostFunc("toDegrees", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return jsJFlo{f: giFloat(rt, argAt(args, 0)) * 57.29577951308232}
	}))
	o.set("floorDiv", jsHostFunc("floorDiv", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return rt.ktFloorDivMod(argAt(args, 0), argAt(args, 1), false)
	}))
	o.set("floorMod", jsHostFunc("floorMod", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return rt.ktFloorDivMod(argAt(args, 0), argAt(args, 1), true)
	}))
	o.set("round", jsHostFunc("round", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		a := argAt(args, 0)
		w := uint8(64)
		if f, ok := a.(jsJFlo); ok && f.sty == floJavaF {
			w = 32
		}
		return rt.ktConv(jsJFlo{f: jvRoundHalfUp(giFloat(rt, a))}, w, false)
	}))
	return o
}

// ktMathMethod is the Double / Float extension members: pow, roundToInt and
// roundToLong. roundToInt rounds half up (Math.round) and saturates at the Int
// range; a NaN receiver throws, as MathJVM.kt does.
func (rt *jsrt) ktMathMethod(target interface{}, name string, args []interface{}) (interface{}, bool) {
	f, ok := target.(jsJFlo)
	if !ok {
		return nil, false
	}
	switch name {
	case "pow":
		return ktMathRes(f, fdPow(f.f, giFloat(rt, argAt(args, 0)))), true
	case "roundToInt", "roundToLong":
		if math.IsNaN(f.f) {
			rt.ktRaise("IllegalArgumentException", "Cannot round NaN value.")
		}
		w := uint8(32)
		if name == "roundToLong" {
			w = 64
		}
		return rt.ktConv(jsJFlo{f: jvRoundHalfUp(f.f)}, w, false), true
	}
	return nil, false
}

// ktClock is java.lang.System's two clocks as Kotlin/JVM reaches them, from
// microseconds since the epoch: currentTimeMillis in milliseconds, nanoTime in
// nanoseconds (its origin is arbitrary in Java; only differences mean anything).
// A native build reads the microseconds with gettimeofday in the emitted module;
// llvm.Run cannot call libc, so here an undefined us reads Go's clock.
func (rt *jsrt) ktClock(kind int, us interface{}) interface{} {
	t := time.Now().UnixMicro()
	if !isUndefOrNull(us) {
		t = int64(giFloat(rt, us))
	}
	if kind == 0 {
		return ktNorm(t/1000, 64, false)
	}
	return ktNorm(t*1000, 64, false)
}

// ktAbs is kotlin.math.abs in the operand's own type: abs(-1.5) is the Double
// 1.5, abs(-1.5f) a Float and abs(-3L) a Long.
func (rt *jsrt) ktAbs(v interface{}) interface{} {
	if f, isFlo := v.(jsJFlo); isFlo {
		return ktMathRes(f, math.Abs(f.f))
	}
	if rt.ktNumLess(v, float64(0)) {
		return rt.ktArith("-", ktNorm(0, 32, false), v)
	}
	return v
}

// ktMathProp is kotlin.math's absoluteValue and sign properties on a number: abs
// and signum in the number's own type, except that an integer's sign is an Int.
func (rt *jsrt) ktMathProp(o interface{}, name string) interface{} {
	if name == "absoluteValue" {
		return rt.ktAbs(o)
	}
	if f, isFlo := o.(jsJFlo); isFlo {
		return ktMathRes(f, ktMath1("sign", f.f))
	}
	if rt.ktNumLess(o, float64(0)) {
		return float64(-1)
	}
	if rt.ktNumLess(float64(0), o) {
		return float64(1)
	}
	return float64(0)
}
