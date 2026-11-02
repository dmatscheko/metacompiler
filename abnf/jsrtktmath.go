package abnf

import (
	"math"
	"math/bits"
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

// ktJavaIntMembers are the java.lang.Integer / Long statics that are a Kotlin Int /
// Long member of the same meaning applied to the first argument (lib/kotlin-math.js's
// ktmJavaInt), and ktJavaIntRadix the digit statics with their radix.
var ktJavaIntMembers = map[string]string{
	"toString": "toString", "bitCount": "countOneBits",
	"numberOfLeadingZeros": "countLeadingZeroBits", "numberOfTrailingZeros": "countTrailingZeroBits",
	"highestOneBit": "takeHighestOneBit", "lowestOneBit": "takeLowestOneBit",
	"rotateLeft": "rotateLeft", "rotateRight": "rotateRight",
}
var ktJavaIntRadix = map[string]int{"toHexString": 16, "toOctalString": 8, "toBinaryString": 2}

// ktJavaInt is java.lang.Integer (w 32) or java.lang.Long (w 64), the JVM classes
// behind Int and Long: the companion constants, the digit statics (two's complement
// at the class width), the parsers (String.toInt / toLong, whose
// NumberFormatException they share), compare / signum / max / min / sum, and the
// member statics above. Twin of kJavaInt in kotlin-interpreter.abnf and k4JavaInt
// in languages/lib/kotlin-rt.metajs.
func ktJavaInt(rt *jsrt, w uint8) *jsObject {
	kt := ktTypeObj("Int")
	parse := "toInt"
	if w == 64 {
		kt = ktTypeObj("Long")
		parse = "toLong"
	}
	conv := func(v interface{}) interface{} { return rt.ktConv(v, w, false) }
	fn := func(name string, f func(args []interface{}) interface{}) {
		kt.set(name, jsHostFunc(name, func(rt *jsrt, this uint64, args []interface{}) interface{} { return f(args) }))
	}
	kt.set("SIZE", float64(w))
	kt.set("BYTES", float64(w/8))
	tail := func(args []interface{}) []interface{} {
		if len(args) > 1 {
			return args[1:]
		}
		return nil
	}
	parser := func(args []interface{}) interface{} {
		return ktMemberCall(argAt(args, 0), parse, tail(args))
	}
	if w == 64 {
		fn("parseLong", parser)
	} else {
		fn("parseInt", parser)
	}
	fn("valueOf", func(args []interface{}) interface{} {
		if _, isStr := argAt(args, 0).(string); isStr {
			return parser(args)
		}
		return argAt(args, 0)
	})
	cmp := func(a, b interface{}) interface{} {
		return ktMemberCall(conv(a), "compareTo", []interface{}{conv(b)})
	}
	fn("compare", func(args []interface{}) interface{} { return cmp(argAt(args, 0), argAt(args, 1)) })
	fn("signum", func(args []interface{}) interface{} { return cmp(argAt(args, 0), float64(0)) })
	for _, mm := range []string{"max", "min"} {
		g := ktGlobalFn(rt, mm)
		fn(mm, func(args []interface{}) interface{} {
			return rt.call(g, jsUndef, []interface{}{conv(argAt(args, 0)), conv(argAt(args, 1))})
		})
	}
	fn("sum", func(args []interface{}) interface{} {
		return ktMemberCall(conv(argAt(args, 0)), "plus", []interface{}{conv(argAt(args, 1))})
	})
	for jn, r := range ktJavaIntRadix {
		r := r
		fn(jn, func(args []interface{}) interface{} { return jxRadix(rt, conv(argAt(args, 0)), r) })
	}
	for jn, kn := range ktJavaIntMembers {
		kn := kn
		fn(jn, func(args []interface{}) interface{} { return ktMemberCall(conv(argAt(args, 0)), kn, tail(args)) })
	}
	return kt
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
	// kotlin.math's withSign (Math.copySign) and nextUp / nextDown (Math.nextUp /
	// nextDown), at the receiver's width. Twins: ktmWithSign / ktmNext in
	// languages/lib/kotlin-math.js.
	case "withSign":
		return ktMathRes(f, math.Copysign(f.f, giFloat(rt, argAt(args, 0)))), true
	case "nextUp", "nextDown":
		to := math.Inf(1)
		if name == "nextDown" {
			to = math.Inf(-1)
		}
		if f.sty == floJavaF {
			return jsJFlo{f: float64(math.Nextafter32(float32(f.f), float32(to))), sty: floJavaF}, true
		}
		return jsJFlo{f: math.Nextafter(f.f, to)}, true
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
	// Double.ulp / Float.ulp: Math.ulp, the distance to the next larger magnitude
	// (Math.ulp(MAX_VALUE) is 2^971, not infinity).
	if f, isFlo := o.(jsJFlo); isFlo && name == "ulp" {
		a := math.Abs(f.f)
		if f.sty == floJavaF {
			a32 := float32(a)
			if a32 == math.MaxFloat32 {
				return jsJFlo{f: math.Ldexp(1, 104), sty: floJavaF}
			}
			return jsJFlo{f: float64(math.Nextafter32(a32, float32(math.Inf(1))) - a32), sty: floJavaF}
		}
		if a == math.MaxFloat64 {
			return jsJFlo{f: math.Ldexp(1, 971)}
		}
		return jsJFlo{f: math.Nextafter(a, math.Inf(1)) - a}
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

// java.util.Random, which a Kotlin/JVM program reaches as `java.util.Random(seed)`:
// java's 48-bit linear congruential generator, seed' = seed * 0x5DEECE66D + 0xB
// (mod 2^48), its stream value for value. The seed lives on the box as a double
// (48 bits are exact). Twin of jrnd* in languages/lib/kotlin-math.js.
const jrndMask = 1<<48 - 1

func jrndMake(seed int64) *jsObject {
	o := newJSObject()
	o.set("__jrnd", true)
	o.set("s", float64((uint64(seed)^0x5DEECE66D)&jrndMask))
	return o
}

func jrndIs(v interface{}) (*jsObject, bool) {
	o, isObj := v.(*jsObject)
	if !isObj {
		return nil, false
	}
	b, ok := o.props["__jrnd"].(bool)
	return o, ok && b
}

func jrndNext(o *jsObject, bits uint) int32 {
	f, _ := o.props["s"].(float64)
	s := (uint64(f)*0x5DEECE66D + 0xB) & jrndMask
	o.set("s", float64(s))
	return int32(int64(s >> (48 - bits)))
}

func jrndBelow(o *jsObject, bound int32) int32 {
	r := jrndNext(o, 31)
	m := bound - 1
	if bound&m == 0 {
		return int32((int64(bound) * int64(r)) >> 31)
	}
	for u := r; ; u = jrndNext(o, 31) {
		r = u % bound
		if u-r+m >= 0 {
			return r
		}
	}
}

// RandomSupport.boundedNextInt over nextInt(), which java.util.Random inherits.
func jrndRange(o *jsObject, origin, bound int32) int32 {
	r := jrndNext(o, 32)
	n := bound - origin
	m := n - 1
	if n&m == 0 {
		return (r & m) + origin
	}
	if n > 0 {
		for u := int32(uint32(r) >> 1); ; u = int32(uint32(jrndNext(o, 32)) >> 1) {
			r = u % n
			if u+m-r >= 0 {
				return r + origin
			}
		}
	}
	for r < origin || r >= bound {
		r = jrndNext(o, 32)
	}
	return r
}

// kotlin.random.Random(seed): kotlin's XorWowRandom, its stream value for value. The
// six Int words live on the box; seed1 is the seed's low word and seed2 its high
// one, and the constructor discards 64 draws. Twin of kxw* in
// languages/lib/kotlin-math.js.
var kxwWords = [...]string{"x", "y", "z", "w", "v", "a"}

func kxwMake(seed int64) *jsObject {
	s1, s2 := int32(seed), int32(seed>>32)
	o := newJSObject()
	o.set("__ktrnd", true)
	for i, w := range []int32{s1, s2, 0, 0, ^s1, s1<<10 ^ int32(uint32(s2)>>4)} {
		o.set(kxwWords[i], float64(w))
	}
	for i := 0; i < 64; i++ {
		kxwNext(o)
	}
	return o
}

func kxwNext(o *jsObject) int32 {
	var s [6]int32
	for i, w := range kxwWords {
		f, _ := o.props[w].(float64)
		s[i] = int32(f)
	}
	t := s[0]
	t ^= int32(uint32(t) >> 2)
	v0 := s[4]
	t = t ^ t<<1 ^ v0 ^ v0<<4
	s = [6]int32{s[1], s[2], s[3], v0, t, s[5] + 362437}
	for i, w := range kxwWords {
		o.set(w, float64(s[i]))
	}
	return t + s[5]
}

// nextBits(b) for b in 0..31.
func kxwBits(o *jsObject, b uint) int32 {
	r := kxwNext(o)
	if b == 0 {
		return 0
	}
	return int32(uint32(r) >> (32 - b))
}

// nextInt(from, until), until > from.
func kxwInt(o *jsObject, from, until int32) int32 {
	n := until - from
	if n > 0 || n == math.MinInt32 {
		if n&-n == n {
			return from + kxwBits(o, uint(31-bits.LeadingZeros32(uint32(n))))
		}
		for {
			b := int32(uint32(kxwNext(o)) >> 1)
			v := b % n
			if b-v+(n-1) >= 0 {
				return from + v
			}
		}
	}
	for {
		if r := kxwNext(o); r >= from && r < until {
			return r
		}
	}
}

func kxwLong(o *jsObject) int64 {
	hi := int64(kxwNext(o))
	return hi<<32 + int64(kxwNext(o))
}

// nextLong(0, n) for 0 < n < 2^31: a power of two takes the top bits of one Int
// draw, any other the remainder of nextLong() ushr 1, rejecting the biased draws.
func kxwLongBelow(o *jsObject, n int64) int64 {
	if n&-n == n {
		return int64(kxwBits(o, uint(63-bits.LeadingZeros64(uint64(n)))))
	}
	for {
		b := int64(uint64(kxwLong(o)) >> 1)
		v := b % n
		if b-v+(n-1) >= 0 {
			return v
		}
	}
}

// kxwShuffle is MutableList.shuffle(random), in place.
func kxwShuffle(o *jsObject, es []interface{}) {
	for i := len(es) - 1; i > 0; i-- {
		j := kxwInt(o, 0, int32(i+1))
		es[i], es[j] = es[j], es[i]
	}
}

// ktJrndMethod is the java.util.Random member surface. Twin of kJrndMethod in
// kotlin-interpreter.abnf and k4JrndMethod in languages/lib/kotlin-rt.metajs.
func (rt *jsrt) ktJrndMethod(o *jsObject, name string, args []interface{}) (interface{}, bool) {
	arg := func(i int) int32 { return int32(giVal(rt, args[i])) }
	switch {
	case name == "nextInt" && len(args) == 0:
		return float64(jrndNext(o, 32)), true
	case name == "nextInt" && len(args) == 1:
		if arg(0) <= 0 {
			rt.ktRaise("IllegalArgumentException", "bound must be positive")
		}
		return float64(jrndBelow(o, arg(0))), true
	case name == "nextInt" && len(args) == 2:
		if arg(0) >= arg(1) {
			rt.ktRaise("IllegalArgumentException", "bound must be greater than origin")
		}
		return float64(jrndRange(o, arg(0), arg(1))), true
	case name == "nextLong" && len(args) == 0:
		hi := int64(jrndNext(o, 32))
		return jsGInt{v: hi<<32 + int64(jrndNext(o, 32)), w: 64}, true
	case name == "nextDouble" && len(args) == 0:
		hi := int64(jrndNext(o, 26))
		return jvmMkFlo(float64(hi<<27+int64(jrndNext(o, 27))) * 0x1.0p-53), true
	case name == "nextFloat" && len(args) == 0:
		return jsJFlo{f: float64(float32(jrndNext(o, 24)) / (1 << 24)), sty: floJavaF}, true
	case name == "nextBoolean" && len(args) == 0:
		return jrndNext(o, 1) != 0, true
	}
	return nil, false
}
