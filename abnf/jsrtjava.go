package abnf

// jsrtjava.go -- Java's VALUE RENDERING for the compiler grammar
// (languages/java-to-llvm-ir.abnf).
//
// java-to-llvm-ir.abnf used to bind System.out.println straight to the shared
// `println` host function, which hands the raw runtime value to Go's `%v`. That
// printed `<nil>` for a null reference, `[1 2 3]` for an array and a full
// `map[__class:map[__isclass:true __name:Box ...] x:0]` dump - containing raw Go
// POINTER addresses, so the compiler's stdout was not even reproducible between
// two runs - for an instance. Printing an ENUM CONSTANT crashed the process
// outright with a Go stack overflow, because the constant's `__class` points at a
// descriptor that points back at the constant and `toGoNatural` has no cycle
// guard. Real java 24 prints
//
//	null / [I@1dbd16a6 / Main$Box@7344699f / RED / P[x=1, y=2]
//
// The interpreter half was wrong in its own way: its `jstr` fell through to
// JavaScript's ToString, so an array printed "1,2,3", an instance "Box@obj", a
// record "P@obj" and a `double[]` "[object Object]". Both halves now render
// through the same rules; this file is the Go twin of `jstr` / `jIdHash` /
// `jArrayTag` in java-interpreter.abnf and the two MUST stay in step, because
// ./test.sh --cross diffs them.
//
// Registration is additive: an init() appends a registrar to rxExtraExterns, so
// nothing here runs unless a program calls one of the js_jv* externs, and only the
// Java compiler grammar emits them. No existing extern is renamed, removed or
// rebound, and abnf/jsrt.go's printArgs is untouched.
//
// Known simplifications, all DELIBERATE and all shared by the two halves:
//
//   - The identity HASH after the `@` is a per-run counter, not an address: the
//     k-th distinct object rendered in a run gets 0x1a2b3c00+k, printed with %x.
//     Real java prints System.identityHashCode, which differs between two runs of
//     the SAME program, so no assertion can ever depend on the digits; what the two
//     halves DO have to agree on is that the same object renders the same way twice
//     and that a program's output is reproducible, which a counter gives.
//   - An array's ELEMENT TYPE is inferred from the elements, because this value
//     model does not carry the declared type. plain number -> [I, boxed double ->
//     [D, String -> [Ljava.lang.String;, boolean -> [Z, char -> [C, and empty or
//     mixed -> [Ljava.lang.Object;. So `Object[] o = {1, 2}` renders as `[I@...`
//     where real java says `[Ljava.lang.Object;@...`, and a nested `int[][]` renders
//     as `[Ljava.lang.Object;@...` where real java says `[[I@...`.
//   - `Object#toString` IS a callable member now: `b.toString()` on a class that
//     declares none routes to js_jvstr, the same rendering println / `+` /
//     String.valueOf use, and `b.equals(o)` routes to js_jvaleq. Both dispatch to a
//     declared or inherited method first, so an override is unaffected - the shape
//     js_jhash established for hashCode.

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode/utf16"
)

// ----------------------------------------------------------------------------
// SIZED INTEGERS
//
// Java's integral types have fixed widths and wrap silently, and the width has
// to survive into the next operation - `byte b = 127; b++` is -128, and nothing
// about the value 127 says so. The box that carries it is jsGInt from
// jsrtint.go, the same one Go and Swift use; only the RULES are Java's, and they
// live here so no shared file has to learn a second dialect.
//
// The invariant is Java's and it is deliberately NOT Go's. There a plain number
// is a 64 bit int; here
//
//	a plain number  ==  an `int`, i.e. a SIGNED 32 BIT value
//	a jsGInt        ==  a long (w 64), a short (w 16) or a byte (w 8)
//	a jsJFlo        ==  a double / float                        (jsrtjvm.go)
//	a jsChar        ==  a char
//
// int is the type of every unsuffixed integer literal and of every arithmetic
// result with no long operand, so keeping IT unboxed is what makes the change
// affordable. The cost is that a 64 bit value stays boxed even when it would fit
// in a double, because a plain number means `int` here - giNorm unboxes such a
// result, so jvBox boxes it straight back.
//
// languages/java-interpreter.abnf implements exactly these rules on top of the
// sz* layer in lib/interp-core.js, and ./test.sh --cross diffs the two halves,
// so the two MUST stay in step.

// jvBox is "read this as a long": always a box, never a plain number.
func jvBox(v int64) jsGInt { return jsGInt{v: v, w: 64, u: false} }

func jvIsLong(v interface{}) bool {
	b, ok := v.(jsGInt)
	return ok && b.w == 64
}

// jvIsIntegral says whether a value is an operand of INTEGER arithmetic - which
// a char is, Java's char being an integral type, and a double is not.
func jvIsIntegral(v interface{}) bool {
	switch v.(type) {
	case jsGInt, float64, jsChar:
		return true
	}
	return false
}

// jvW is JLS 5.6.2 binary numeric promotion for two integral operands: long when
// either side is long, int otherwise - so `byte + byte` is an int and
// `short * short` is an int, which is the rule people are surprised by.
func jvW(l, r interface{}) uint8 {
	if jvIsLong(l) || jvIsLong(r) {
		return 64
	}
	return 32
}

// jvI is the int reading of an integral operand: the low 32 bits of its value.
func jvI(rt *jsrt, v interface{}) int32 { return int32(giVal(rt, v)) }

// The five runtime-raised exception class descriptors, each registered by NAME
// by js_jvexc at module init, because the externs that raise them have no access
// to the program's scope; without one the raise aborts the run instead of
// throwing. One compile runs one program, so a package-level holder is
// per-program - the same reasoning the jvpIdent counter below records, and it
// keeps this out of the shared jsrt struct.
//
// ArithmeticException is "/ by zero" (JLS 15.17.2). The other four: an out-of-range array
// access, an out-of-range string index or range, a negative array length and a
// null array reference all THROW rather than abort. Same reasoning, same
// per-program lifetime; the layer-2 twin is the block of jv*Cls slots in
// languages/lib/java-rt.metajs.
var jvArithExcCls *jsObject
var jvAioobeCls *jsObject
var jvSioobeCls *jsObject
var jvNegArrCls *jsObject
var jvNpeCls *jsObject

func (rt *jsrt) jvThrowCls(cls *jsObject, msg interface{}) {
	if cls == nil {
		rt.fail("java: %v", msg)
		return
	}
	o := newJSObject()
	o.set("__class", cls)
	// __msg is the field the emitted builtin exception constructor writes and
	// getMessage() reads (see emitBuiltinExc in java-to-llvm-ir.abnf).
	o.set("__msg", msg)
	// jsThrown carries the VALUE, not its handle - js_throw stores u(a[0]).
	panic(&jsThrown{value: o})
}

func (rt *jsrt) jvThrowArith(msg string) { rt.jvThrowCls(jvArithExcCls, msg) }

// Is the value part of the java.lang Throwable hierarchy at all? The 64-deep cap
// is the one every __class/__super walk here carries (manual chapter 7.9).
func jvIsThrowable(v interface{}) bool {
	o, ok := v.(*jsObject)
	if !ok {
		return false
	}
	cls := o.props["__class"]
	for guard := 0; guard < 64; guard++ {
		co, isObj := cls.(*jsObject)
		if !isObj {
			return false
		}
		if nm, _ := co.props["__name"].(string); nm == "Throwable" {
			return true
		}
		cls = co.props["__super"]
	}
	return false
}

// The three bounds messages are java 24.0.2's, verified against the real
// toolchain. jArrIdx/jStrIdx/jSubstr in languages/java-interpreter.abnf and
// jvThrowAioobe/jvThrowSioobe/jvThrowRange in languages/lib/java-rt.metajs spell
// the same three, and tests/java-test-full.java SECTION 34 pins them.
func (rt *jsrt) jvThrowAioobe(i, n int) {
	rt.jvThrowCls(jvAioobeCls, fmt.Sprintf("Index %d out of bounds for length %d", i, n))
}
func (rt *jsrt) jvThrowSioobe(i, n int) {
	rt.jvThrowCls(jvSioobeCls, fmt.Sprintf("Index %d out of bounds for length %d", i, n))
}
func (rt *jsrt) jvThrowRange(b, e, n int) {
	rt.jvThrowCls(jvSioobeCls, fmt.Sprintf("Range [%d, %d) out of bounds for length %d", b, e, n))
}

// jvArith is one binary arithmetic or bitwise operator on two INTEGRAL operands.
// The int case is native int32 arithmetic; only a long operand pays for the
// 64 bit box.
func (rt *jsrt) jvArith(op string, l, r interface{}) interface{} {
	if jvW(l, r) == 64 {
		a, b := jvBox(giVal(rt, l)), jvBox(giVal(rt, r))
		if (op == "/" || op == "%") && b.v == 0 {
			rt.jvThrowArith("/ by zero")
		}
		return jvBox(giVal(rt, rt.giArith(op, a, b)))
	}
	a, b := jvI(rt, l), jvI(rt, r)
	var x int32
	switch op {
	case "+":
		x = a + b
	case "-":
		x = a - b
	case "*":
		x = a * b
	case "/":
		if b == 0 {
			rt.jvThrowArith("/ by zero")
		}
		// Integer.MIN_VALUE / -1 overflows back to itself (JLS 15.17.2); Go's own
		// int32 division panics on exactly that pair, so it is special-cased.
		if a == math.MinInt32 && b == -1 {
			x = a
		} else {
			x = a / b
		}
	case "%":
		if b == 0 {
			rt.jvThrowArith("/ by zero")
		}
		if a == math.MinInt32 && b == -1 {
			x = 0
		} else {
			x = a % b
		}
	case "&":
		x = a & b
	case "|":
		x = a | b
	case "^":
		x = a ^ b
	default:
		rt.fail("js_jvarith: unknown operator %q", op)
	}
	return float64(x)
}

// jvShift is << >> >>>. The result type is the LEFT operand's alone (JLS 15.19 -
// each operand is promoted separately), and the COUNT is MASKED: & 31 for an int,
// & 63 for a long. That is why `1 << 32` is 1 rather than 0, and it is the one
// place Java and Go disagree outright, Go shifting everything out instead.
func (rt *jsrt) jvShift(op string, l, r interface{}) interface{} {
	n := giVal(rt, r)
	if jvIsLong(l) {
		s := n & 63
		a := giVal(rt, l)
		switch op {
		case "<<":
			return jvBox(a << uint(s))
		case ">>":
			return jvBox(a >> uint(s))
		}
		// >>> is the LOGICAL shift: read the operand as unsigned, shift, read the
		// result back as signed. C# spells this ">>" on an unsigned type.
		return jvBox(int64(uint64(a) >> uint(s)))
	}
	a := jvI(rt, l)
	s := uint(n & 31)
	switch op {
	case "<<":
		return float64(a << s)
	case ">>":
		return float64(a >> s)
	}
	return float64(int32(uint32(a) >> s))
}

// jvCmp is the ordered comparison at the PROMOTED width. giCmp on its own is not
// usable: it compares at the widest operand's width, so `byte b = 5; b < 1000`
// would truncate 1000 to a byte and answer false where Java promotes both to int.
func (rt *jsrt) jvCmp(op string, l, r interface{}) bool {
	var c int
	if jvW(l, r) == 64 {
		a, b := giVal(rt, l), giVal(rt, r)
		if a < b {
			c = -1
		} else if a > b {
			c = 1
		}
	} else {
		a, b := jvI(rt, l), jvI(rt, r)
		if a < b {
			c = -1
		} else if a > b {
			c = 1
		}
	}
	switch op {
	case "<":
		return c < 0
	case ">":
		return c > 0
	case "<=":
		return c <= 0
	case ">=":
		return c >= 0
	case "==":
		return c == 0
	}
	return c != 0
}

// jvToLong is the exact 64 bit reading of an INTEGRAL value.
func jvToLong(rt *jsrt, v interface{}) int64 {
	switch t := v.(type) {
	case jsGInt:
		return t.v
	case jsChar:
		return int64(t.code)
	case float64:
		return int64(t) // an int is exact
	}
	return jvNarrowFloat(rt.toNumber(v), 64)
}

// jvNarrowFloat is the JLS 5.1.3 narrowing of a FLOATING POINT value. It
// SATURATES at the target's range rather than wrapping the way Go does - and for
// a target narrower than int it saturates at the INT range first and only then
// truncates the bits, so (int) 1e20 is Integer.MAX_VALUE and (byte) 1e20 is
// (byte) Integer.MAX_VALUE == -1.
func jvNarrowFloat(f float64, w uint8) int64 {
	if math.IsNaN(f) {
		return 0
	}
	if w == 64 {
		if f >= 9223372036854775808 {
			return math.MaxInt64
		}
		if f < -9223372036854775808 {
			return math.MinInt64
		}
		return int64(math.Trunc(f))
	}
	if f >= 2147483648 {
		return math.MaxInt32
	}
	if f < -2147483648 {
		return math.MinInt32
	}
	return int64(math.Trunc(f))
}

// jvNarrow is (byte) / (short) / (int) / (long) / a declared integral type. An
// integral source is a pure bit truncation; a floating point one saturates first
// (see jvNarrowFloat).
func jvNarrow(rt *jsrt, v interface{}, w uint8) interface{} {
	var p int64
	if f, isFlo := v.(jsJFlo); isFlo {
		p = jvNarrowFloat(f.f, w)
	} else {
		p = jvToLong(rt, v)
	}
	switch w {
	case 64:
		return jvBox(p)
	case 32:
		return float64(int32(p))
	case 16:
		return jsGInt{v: int64(int16(p)), w: 16, u: false}
	}
	return jsGInt{v: int64(int8(p)), w: 8, u: false}
}

// jvParseInt is Integer.parseInt / Long.parseLong: a decimal digit run with an
// optional sign, and a NumberFormatException - here a plain abort, since the
// message is the only part a program reads - on anything else.
func (rt *jsrt) jvParseInt(s string, width uint8) interface{} {
	t := strings.TrimSpace(s)
	neg := false
	if strings.HasPrefix(t, "+") || strings.HasPrefix(t, "-") {
		neg = t[0] == '-'
		t = t[1:]
	}
	if t == "" {
		rt.fail("java.lang.NumberFormatException: For input string: %q", s)
	}
	var acc uint64
	for i := 0; i < len(t); i++ {
		if t[i] < '0' || t[i] > '9' {
			rt.fail("java.lang.NumberFormatException: For input string: %q", s)
		}
		acc = acc*10 + uint64(t[i]-'0')
	}
	v := int64(acc)
	if neg {
		v = -v
	}
	return jvNarrow(rt, jvBox(v), width)
}

// jvHexOf is Integer.toHexString / Long.toHexString (todo.md 1.10): the UNSIGNED
// hex reading of the value AT THE GIVEN WIDTH, lowercase, no leading zeros - so
// Integer.toHexString(-1) is "ffffffff". The twin is jvToHex32 / jvToHex64 in
// languages/lib/java-rt.metajs.
func jvHexOf(rt *jsrt, v interface{}, w uint8) string {
	var p int64
	if f, isFlo := v.(jsJFlo); isFlo {
		p = jvNarrowFloat(f.f, w)
	} else {
		p = jvToLong(rt, v)
	}
	if w == 32 {
		return strconv.FormatUint(uint64(uint32(p)), 16)
	}
	return strconv.FormatUint(uint64(p), 16)
}

// jvBoolOf is Boolean.valueOf / Boolean.parseBoolean: a STRING argument is true
// iff it equalsIgnoreCase("true"), anything else is its own truth. The twin is
// jvBoolOf in languages/lib/java-rt.metajs.
func jvBoolOf(v interface{}) bool {
	if s, ok := v.(string); ok {
		return strings.EqualFold(s, "true")
	}
	b, _ := v.(bool)
	return b
}

// jvBoxType builds one of java.lang's boxed-primitive types.
func (rt *jsrt) jvBoxType(name string) *jsObject {
	o := newJSObject()
	cmp := func(a, b interface{}) interface{} {
		if rt.jvCmp("<", a, b) {
			return float64(-1)
		}
		if rt.jvCmp(">", a, b) {
			return float64(1)
		}
		return float64(0)
	}
	addCommon := func(width uint8, size, bytes float64) {
		o.set("SIZE", size)
		o.set("BYTES", bytes)
		o.set("valueOf", jsHostFunc("valueOf", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return jvNarrow(rt, argAt(args, 0), width)
		}))
		o.set("toString", jsHostFunc("toString", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return rt.jvpStr(argAt(args, 0))
		}))
		o.set("compare", jsHostFunc("compare", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return cmp(argAt(args, 0), argAt(args, 1))
		}))
		o.set("max", jsHostFunc("max", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			l, r := argAt(args, 0), argAt(args, 1)
			if rt.jvCmp(">", l, r) {
				return jvNarrow(rt, l, width)
			}
			return jvNarrow(rt, r, width)
		}))
		o.set("min", jsHostFunc("min", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			l, r := argAt(args, 0), argAt(args, 1)
			if rt.jvCmp("<", l, r) {
				return jvNarrow(rt, l, width)
			}
			return jvNarrow(rt, r, width)
		}))
	}
	switch name {
	case "Integer":
		o.set("MAX_VALUE", float64(math.MaxInt32))
		o.set("MIN_VALUE", float64(math.MinInt32))
		addCommon(32, 32, 4)
		o.set("parseInt", jsHostFunc("parseInt", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return rt.jvParseInt(rt.toString(argAt(args, 0)), 32)
		}))
		o.set("toHexString", jsHostFunc("toHexString", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return jvHexOf(rt, argAt(args, 0), 32)
		}))
	case "Long":
		o.set("MAX_VALUE", jvBox(math.MaxInt64))
		o.set("MIN_VALUE", jvBox(math.MinInt64))
		addCommon(64, 64, 8)
		o.set("parseLong", jsHostFunc("parseLong", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return rt.jvParseInt(rt.toString(argAt(args, 0)), 64)
		}))
		o.set("toHexString", jsHostFunc("toHexString", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return jvHexOf(rt, argAt(args, 0), 64)
		}))
	case "Byte":
		o.set("MAX_VALUE", jsGInt{v: 127, w: 8})
		o.set("MIN_VALUE", jsGInt{v: -128, w: 8})
		addCommon(8, 8, 1)
	case "Short":
		o.set("MAX_VALUE", jsGInt{v: 32767, w: 16})
		o.set("MIN_VALUE", jsGInt{v: -32768, w: 16})
		addCommon(16, 16, 2)
	case "Character":
		o.set("MAX_VALUE", jsChar{code: 65535})
		o.set("MIN_VALUE", jsChar{code: 0})
		o.set("SIZE", float64(16))
		o.set("BYTES", float64(2))
	// Double / Boolean carry only what todo.md 1.10 asked for; see the note on the
	// layer-2 twin in languages/lib/java-rt.metajs.
	case "Double":
		o.set("NaN", jsJFlo{f: math.NaN(), sty: floJava})
		o.set("SIZE", float64(64))
		o.set("BYTES", float64(8))
	case "Boolean":
		o.set("TRUE", true)
		o.set("FALSE", false)
		bv := jsHostFunc("valueOf", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return jvBoolOf(argAt(args, 0))
		})
		o.set("valueOf", bv)
		o.set("parseBoolean", bv)
	}
	return o
}

// Double.compare(x, y) == 0, which is Double.equals and therefore what a record
// component of type double or float is compared with. It is doubleToLongBits
// equality: every NaN collapses to one bit pattern, so NaN equals NaN, and the
// sign of a zero is part of the value, so +0.0 does not equal -0.0. ONE body
// serves both widths - a float box already holds the binary32-rounded value
// (jvmFround), and distinct binary32 values are distinct float64 values.
func jvDoubleCompareEq(x, y float64) bool {
	if math.IsNaN(x) || math.IsNaN(y) {
		return math.IsNaN(x) && math.IsNaN(y)
	}
	if x == 0 && y == 0 {
		return math.Signbit(x) == math.Signbit(y)
	}
	return x == y
}

func init() {
	rxExtraExterns = append(rxExtraExterns, func(rt *jsrt, m map[string]func(args []uint64) uint64) {
		u := rt.unwrap
		w := rt.wrap
		boolH := func(b bool) uint64 {
			if b {
				return jsHTrue
			}
			return jsHFalse
		}
		opOf := func(h uint64) string { return rt.toString(u(h)) }

		// One runtime-raised exception class descriptor, handed over BY NAME at
		// module init so that the externs below can THROW rather than abort.
		m["js_jvexc"] = func(a []uint64) uint64 {
			o, ok := u(a[1]).(*jsObject)
			if !ok {
				return 0
			}
			switch rt.toString(u(a[0])) {
			case "ArithmeticException":
				jvArithExcCls = o
			case "ArrayIndexOutOfBoundsException":
				jvAioobeCls = o
			case "StringIndexOutOfBoundsException":
				jvSioobeCls = o
			case "NegativeArraySizeException":
				jvNegArrCls = o
			case "NullPointerException":
				jvNpeCls = o
			}
			return 0
		}
		// `a[i]`, read or written. EVERY array access is bounds-checked in Java
		// (JLS 10.4) and a violation throws ArrayIndexOutOfBoundsException - it
		// does not answer a default and it does not extend the array on a write.
		// The emitter emits this for both halves; the write site discards the
		// value and keeps only the check (emitRefPath in java-to-llvm-ir.abnf).
		m["js_jvidx"] = func(a []uint64) uint64 {
			o := u(a[0])
			if arr, isArr := o.(*jsArray); isArr {
				i := jsToInt(rt.toNumber(u(a[1])))
				if i < 0 || i >= len(arr.elems) {
					rt.jvThrowAioobe(i, len(arr.elems))
				}
				return w(arr.elems[i])
			}
			if isUndefOrNull(o) {
				// `int[] z = null; z[0]` - a catchable NullPointerException with a
				// null message, exactly as the interpreter half throws it.
				rt.jvThrowCls(jvNpeCls, jsNull)
			}
			return w(rt.getMember(o, u(a[1])))
		}
		// Does a thrown value match ONE catch-clause type name? The emitter emits
		// one call per name, so a multi-catch `catch (A | B e)` is two calls. The
		// twin of core.excMatches in java-interpreter.abnf and of js_jvexcmatch in
		// languages/lib/java-rt.metajs. A value outside the Throwable hierarchy -
		// this subset lets a program throw any object - is caught by the four
		// general types, which is how `catch (Exception e)` is written in practice.
		m["js_jvexcmatch"] = func(a []uint64) uint64 {
			v := u(a[0])
			n := rt.toString(u(a[1]))
			if rt.isTypeName(v, n) {
				return jsHTrue
			}
			switch n {
			case "Exception", "Throwable", "RuntimeException", "Error":
				return boolH(!jvIsThrowable(v))
			}
			return jsHFalse
		}
		// One dimension of `new T[n]`: answers it, and raises
		// NegativeArraySizeException on a negative one (JLS 15.10.1). The message
		// is the offending size and nothing else.
		m["js_jvnewlen"] = func(a []uint64) uint64 {
			n := jsToInt(rt.toNumber(u(a[0])))
			if n < 0 {
				rt.jvThrowCls(jvNegArrCls, strconv.Itoa(n))
			}
			return a[0]
		}
		// java.lang.String.charAt THROWS StringIndexOutOfBoundsException where the
		// shared arm in abnf/jsrt.go fails the run uncatchably. Only the java
		// grammar emits js_jcharat, so overriding it here is a java-only change -
		// rxExtraExterns runs after the base table is filled.
		baseCharAt := m["js_jcharat"]
		m["js_jcharat"] = func(a []uint64) uint64 {
			if str, isStr := u(a[0]).(string); isStr {
				args, ok := u(a[1]).(*jsArray)
				if ok {
					i := jsToInt(rt.toNumber(argAt(args.elems, 0)))
					units := utf16.Encode([]rune(str))
					if i < 0 || i >= len(units) {
						rt.jvThrowSioobe(i, len(units))
					}
				}
			}
			return baseCharAt(a)
		}
		// java.lang.String.substring checks the RANGE - begin > end is out of
		// bounds too - where the shared memberCall CLAMPS through substringRange.
		// A SEPARATE EXTERN, not a wrapper on js_mcall: rxExtraExterns fills ONE
		// map for every language, so overriding js_mcall here made a C# Substring
		// raise java's "Range [10, 6) out of bounds" (observed). js_ktsmcall and
		// js_swmcall are the same pattern for the same reason. Only the java
		// emitter emits js_jvsub, and only for the name `substring`.
		baseMcall := m["js_mcall"]
		m["js_jvsub"] = func(a []uint64) uint64 {
			if str, isStr := u(a[0]).(string); isStr {
				if args, ok := u(a[1]).(*jsArray); ok {
					units := utf16.Encode([]rune(str))
					bi := jsToInt(rt.toNumber(argAt(args.elems, 0)))
					ei := len(units)
					if len(args.elems) > 1 {
						ei = jsToInt(rt.toNumber(argAt(args.elems, 1)))
					}
					if bi < 0 || ei > len(units) || bi > ei {
						rt.jvThrowRange(bi, ei, len(units))
					}
				}
			}
			return baseMcall([]uint64{a[0], w("substring"), a[1]})
		}
		// One binary integral operator: (op, left, right). A double operand keeps
		// the FLOAT path of jsrtjvm.go, which is where the two boxes meet.
		m["js_jvarith"] = func(a []uint64) uint64 {
			op, l, r := opOf(a[0]), u(a[1]), u(a[2])
			if jvmIsFlo(l) || jvmIsFlo(r) {
				return w(rt.jvmArith(op[0], l, r))
			}
			if !jvIsIntegral(l) || !jvIsIntegral(r) {
				// & | ^ on two booleans are Java's non-short-circuit boolean
				// operators; anything else is not a program javac accepts.
				lb, lok := l.(bool)
				rb, rok := r.(bool)
				if lok && rok {
					switch op {
					case "&":
						return boolH(lb && rb)
					case "|":
						return boolH(lb || rb)
					case "^":
						return boolH(lb != rb)
					}
				}
				return w(rt.jvmArith(op[0], l, r))
			}
			return w(rt.jvArith(op, l, r))
		}
		m["js_jvshift"] = func(a []uint64) uint64 {
			return w(rt.jvShift(opOf(a[0]), u(a[1]), u(a[2])))
		}
		// An ordered comparison. Two integers compare at the promoted width;
		// anything else (a double, a string pair) keeps the shared comparison.
		m["js_jvcmp"] = func(a []uint64) uint64 {
			op, l, r := opOf(a[0]), u(a[1]), u(a[2])
			if jvIsIntegral(l) && jvIsIntegral(r) && !jvmIsFlo(l) && !jvmIsFlo(r) {
				return boolH(rt.jvCmp(op, l, r))
			}
			// A float operand promotes the other side to float too (JLS 5.6.2),
			// so 16777217 < 16777216f is FALSE - the int converts to 1.6777216E7.
			l, r = jvmFloCmpPair(rt, l, r)
			c := rt.jsCompare(l, r)
			// jsCompare answers the SENTINEL 2 for a NaN operand, commented
			// there as "every relation is false" - and that is exactly Java's
			// rule (JLS 15.20.1: if either operand is NaN then <, <=, > and >=
			// all produce false). Read as an ordering the sentinel made `>` and
			// `>=` TRUE here, while layer 2's own copy of jsCompare dropped the
			// sentinel and answered 0, making `<=` and `>=` true - so the two
			// halves were wrong in DIFFERENT directions and disagreed with each
			// other. It is not 0 either. This is the same fix jsrtcsharp.go:477
			// took; java is its twin and was missed. languages/lib/java-rt.metajs
			// (js_jvcmp) carries the identical three lines.
			if c == 2 {
				return boolH(false)
			}
			switch op {
			case "<":
				return boolH(c < 0)
			case ">":
				return boolH(c > 0)
			case "<=":
				return boolH(c <= 0)
			}
			return boolH(c >= 0)
		}
		// A cast or a declared integral type: (value, bits).
		m["js_jvconv"] = func(a []uint64) uint64 {
			return w(jvNarrow(rt, u(a[0]), uint8(jsToInt(rt.toNumber(u(a[1]))))))
		}
		// A declared type applied to an initializer: only an INTEGRAL value
		// narrows, so `Object o = "s"` and a lambda pass through untouched.
		m["js_jvadopt"] = func(a []uint64) uint64 {
			v := u(a[0])
			if !jvIsIntegral(v) {
				return a[0]
			}
			return w(jvNarrow(rt, v, uint8(jsToInt(rt.toNumber(u(a[1]))))))
		}
		// An integral literal, passed as DIGITS: emitNum would already have
		// rounded 9223372036854775807 to 9223372036854776000 on the way into the
		// module. (text, radix, isLong).
		m["js_jvlit"] = func(a []uint64) uint64 {
			s := rt.toString(u(a[0]))
			radix := uint64(jsToInt(rt.toNumber(u(a[1]))))
			var acc uint64
			for i := 0; i < len(s); i++ {
				var d uint64
				switch c := s[i]; {
				case c >= '0' && c <= '9':
					d = uint64(c - '0')
				case c >= 'a' && c <= 'f':
					d = uint64(c-'a') + 10
				case c >= 'A' && c <= 'F':
					d = uint64(c-'A') + 10
				default:
					continue // a '_' digit separator
				}
				if d >= radix {
					continue
				}
				acc = acc*radix + d
			}
			if rt.truthy(u(a[2])) {
				return w(jvBox(int64(acc)))
			}
			return rt.wrapNum(float64(int32(acc)))
		}
		// -x and ~x, both at the PROMOTED width (JLS 5.6.1): a long stays a long,
		// everything integral else becomes an int.
		m["js_jvneg"] = func(a []uint64) uint64 {
			v := u(a[0])
			if f, ok := v.(jsJFlo); ok {
				return w(jsJFlo{f: -f.f, sty: f.sty})
			}
			if jvIsLong(v) {
				return w(jvBox(-giVal(rt, v)))
			}
			return rt.wrapNum(float64(-jvI(rt, v)))
		}
		m["js_jvnot"] = func(a []uint64) uint64 {
			v := u(a[0])
			if jvIsLong(v) {
				return w(jvBox(^giVal(rt, v)))
			}
			return rt.wrapNum(float64(^jvI(rt, v)))
		}
		// ++ / -- keep the type of what they step, which is the same implicit
		// narrowing a compound assignment carries (JLS 15.14.2).
		m["js_jvstep"] = func(a []uint64) uint64 {
			v, d := u(a[0]), giVal(rt, u(a[1]))
			switch t := v.(type) {
			case jsChar:
				return w(jsChar{code: int32((int64(t.code) + d) & 65535)})
			case jsJFlo:
				// ++/-- keeps the operand's own type, WIDTH included: a float
				// steps as a float (jvF32Round in lib/java-rt.metajs).
				if t.sty == floJavaF {
					return w(jsJFlo{f: jvmFround(t.f + float64(d)), sty: floJavaF})
				}
				return w(jsJFlo{f: t.f + float64(d), sty: t.sty})
			case jsGInt:
				return w(jvNarrow(rt, rt.jvArith("+", t, float64(d)), t.w))
			}
			return rt.wrapNum(float64(jvI(rt, v) + int32(d)))
		}
		// The result of a COMPOUND assignment: the operation, then the implicit
		// narrowing cast back to the LEFT operand's type (JLS 15.26.2). So
		// `byte b = 10; b += 300` is (byte)310 == 54 and `int i = 5; i += 1.5` is
		// (int)6.5 == 6. A String left operand keeps its concatenation.
		m["js_jvnarrowlike"] = func(a []uint64) uint64 {
			l, v := u(a[0]), u(a[1])
			switch t := l.(type) {
			case jsJFlo:
				// A float left operand narrows the result back to a float:
				// `float f = 1.1f; f += 0.1` is (float)(f + 0.1) and not a double.
				if t.sty == floJavaF {
					return w(jsJFlo{f: jvmFround(rt.toNumber(v)), sty: floJavaF})
				}
				if f, ok := v.(jsJFlo); ok {
					return w(f)
				}
				return w(jsJFlo{f: rt.toNumber(v), sty: t.sty})
			case jsChar:
				return w(jsChar{code: int32(giVal(rt, v) & 65535)})
			case jsGInt:
				return w(jvNarrow(rt, v, t.w))
			case float64:
				if jvIsIntegral(v) || jvmIsFlo(v) {
					return w(jvNarrow(rt, v, 32))
				}
			}
			return a[1]
		}
		// The boxed-primitive types Integer / Long / Byte / Short / Character, one
		// object per name. MIN_VALUE / MAX_VALUE are the whole reason a sized
		// integer had to exist: Long.MAX_VALUE cannot be written as a plain number
		// at all, and Integer.MAX_VALUE + 1 has to wrap rather than grow. The twin
		// is hostGlobals["Integer"] etc in java-interpreter.abnf.
		m["js_jvboxtype"] = func(a []uint64) uint64 { return w(rt.jvBoxType(rt.toString(u(a[0])))) }
		// A (char) cast: narrow to 16 bits and re-box. char is the one integral
		// type that is UNSIGNED, so the mask rather than a sign extension is the
		// whole answer - (char)-1 is 65535 and (char)70000 is 4464.
		m["js_jvchar"] = func(a []uint64) uint64 {
			return w(jsChar{code: int32(jvToLong(rt, u(a[0])) & 65535)})
		}
		// The plain-number reading a raw consumer needs (an array index, a length,
		// a bound): a box is not a float64 and every such consumer wants one.
		m["js_jvnum"] = func(a []uint64) uint64 {
			if b, ok := u(a[0]).(jsGInt); ok {
				return rt.wrapNum(float64(b.v))
			}
			return a[0]
		}

		// System.out.println(x) / System.out.print(x). The argument ARRAY of the
		// emitted closure arrives whole, so the no-argument println(), which
		// writes just the line terminator, is distinguishable from println(null).
		m["js_jvprint"] = func(a []uint64) uint64 {
			fmt.Fprintln(outWriter, wtf8Clean(jvpFirst(rt, u(a[0]))))
			return 0
		}
		m["js_jvwrite"] = func(a []uint64) uint64 {
			fmt.Fprint(outWriter, wtf8Clean(jvpFirst(rt, u(a[0]))))
			return 0
		}
		// ONE record component of two record instances, compared the way JLS
		// 8.10.3 says a record's canonical equals compares it - which is NOT ===
		// in either direction:
		//
		//   a float / double component  Float.compare / Double.compare == 0, so
		//                               NaN EQUALS NaN and +0.0 does NOT equal
		//                               -0.0. Both are the opposite of ===.
		//   any other primitive         ==  (js_jchareq, chars included)
		//   a reference                 Objects.equals(a, b), i.e. a == b || (a
		//                               != null && a.equals(b)) - so a nested
		//                               record compares component-wise and a
		//                               class with a user-declared equals uses
		//                               it, where identity said unequal.
		//
		// A class WITHOUT an equals (and an array, which does not override it)
		// keeps identity, because Object.equals is identity. The layer-2 twin is
		// js_jvaleq in languages/lib/java-rt.metajs and it matches arm for arm.
		m["js_jvaleq"] = func(a []uint64) uint64 {
			x, y := u(a[0]), u(a[1])
			// TWO PRIMITIVES OF DIFFERENT KINDS ARE NEVER EQUAL. Integer#equals
			// asks `instanceof Integer` before it compares, so
			// `((Object) 1).equals(1L)` is FALSE in java - while the floor's
			// strict_eq deliberately compares a sized integer and a plain number
			// BY VALUE (chapter 7.9 of the manual), and js_jchareq below falls
			// through to it. That gap was invisible while this extern only ever
			// saw two components of the SAME declared type; routing Object#equals
			// here made it a live halves divergence against the interpreter, which
			// has always been exact. The guard is right for a record component
			// too: an Object-typed component holding 1 does not equal one holding
			// 1L. Only PRIMITIVES are screened - kind 0 (an instance, an array,
			// null) must still reach the dispatch below, because a declared
			// equals(Object) legitimately compares against a String.
			kx, ky := jvEqKind(x), jvEqKind(y)
			if kx != 0 && ky != 0 && kx != ky {
				return jsHFalse
			}
			if fx, ok := x.(jsJFlo); ok {
				if fy, ok2 := y.(jsJFlo); ok2 {
					return boolH(jvDoubleCompareEq(fx.f, fy.f))
				}
			}
			if m["js_jchareq"](a) == jsHTrue {
				return jsHTrue
			}
			o, ok := x.(*jsObject)
			if !ok {
				return jsHFalse
			}
			// The __class / __super walk rt.memberCall uses, with the guard < 64
			// cap chapter 7.9 of the manual keeps everywhere else (a cyclic
			// __super hangs the oracle, not us).
			cls := o.props["__class"]
			for guard := 0; guard < 64; guard++ {
				clsObj, isObj := cls.(*jsObject)
				if !isObj {
					return jsHFalse
				}
				if eq, has := clsObj.props["equals"]; has && isCallable(eq) {
					// `=== true`, not truthiness: equals is declared boolean and
					// layer 2's twin cannot spell truthiness on an anytype
					// without disagreeing with this line for a program the
					// grammar accepts and javac would not.
					res, isBool := rt.call(eq, jsUndef, []interface{}{x, y}).(bool)
					return boolH(isBool && res)
				}
				cls = clsObj.props["__super"]
			}
			return jsHFalse
		}
		// Object#hashCode and every override of it the JLS pins exactly, plus
		// the record combination this project pins (see jvHash for what is
		// specified and what is a decision). The layer-2 twin is js_jhash in
		// languages/lib/java-rt.metajs.
		m["js_jhash"] = func(a []uint64) uint64 { return rt.wrapNum(rt.jvHash(u(a[0]))) }
		// Is this value a CLASS DESCRIPTOR? The one thing an emitted probe needs to
		// know to apply JLS 6.4.2 - a variable name obscures a type name - because
		// a class descriptor is declared in the same scope chain a bare name reads
		// (see makeVarRef in java-to-llvm-ir.abnf, docs/todo.md 2.9). The layer-2
		// twin is js_jisclass in languages/lib/java-rt.metajs.
		m["js_jisclass"] = func(a []uint64) uint64 {
			o, ok := u(a[0]).(*jsObject)
			if !ok {
				return jsHFalse
			}
			return boolH(o.props["__isclass"] == true)
		}
		// THE OUTER HOP (docs/todo.md 1.6). An inner or anonymous class body names a
		// field of its ENCLOSING instance unqualified, and javac resolves it there
		// (JLS 6.5.6.1: the innermost enclosing class that declares the name wins).
		// The instance carries that enclosing instance as __outer - every creation
		// site of a nested or anonymous class records one - so the search is `this`,
		// the statics of this' class chain, then __outer, and out.
		//
		// Answers the OWNER object, which the caller then reads or writes the name
		// on, so one extern serves both directions. The twins are jOuterOwner in
		// languages/java-interpreter.abnf (which needs no extern: it holds the
		// objects) and js_jouter in languages/lib/java-rt.metajs.
		m["js_jouter"] = func(a []uint64) uint64 {
			name := rt.toString(u(a[1]))
			v := u(a[0])
			for guard := 0; guard < 64; guard++ {
				o, ok := v.(*jsObject)
				if !ok {
					break
				}
				// "not undefined" rather than an own-key test, because the
				// layer-2 twin has no own-key primitive and the two must agree.
				// A declared field is never undefined (the emitted constructor
				// gives every one of them 0, false or null before the body runs).
				if pv, has := o.props[name]; has && pv != interface{}(jsUndef) {
					return w(o)
				}
				if own := jvStaticOwner(o.props["__class"], name); own != nil {
					return w(own)
				}
				v = o.props["__outer"]
			}
			rt.fail("unknown name: " + name)
			return jsHUndefined
		}
		// Outer.this: out along __outer until an instance whose class chain carries
		// that name. One unconditional __outer hop is wrong at two levels of
		// nesting; the twin is makeOuterThis in java-interpreter.abnf.
		m["js_jouterthis"] = func(a []uint64) uint64 {
			name := rt.toString(u(a[1]))
			v := u(a[0])
			for guard := 0; guard < 64; guard++ {
				o, ok := v.(*jsObject)
				if !ok {
					break
				}
				if jvClassIs(o.props["__class"], name) {
					return w(o)
				}
				v = o.props["__outer"]
			}
			rt.fail("no enclosing instance of " + name)
			return jsHUndefined
		}
		// String.valueOf(x) and the string form used by `+`.
		m["js_jvstr"] = func(a []uint64) uint64 { return rt.wrapStr(rt.jvpStr(u(a[0]))) }
		// Java's `+`. Identical to js_jadd (string concat / 32 bit int add / float
		// add) except that the string case renders through jvpStr, so `"" + arr`
		// and `"" + obj` agree with println instead of falling through to
		// JavaScript's ToString ("1,2,3", "[object Object]").
		m["js_jvadd"] = func(a []uint64) uint64 {
			l, r := u(a[0]), u(a[1])
			_, ls := l.(string)
			_, rs := r.(string)
			if ls || rs {
				return rt.wrapStr(rt.jvpStr(l) + rt.jvpStr(r))
			}
			if jvmIsFlo(l) || jvmIsFlo(r) {
				// jvmArith, not a hand-rolled add: the float WIDTH needs both
				// operands converted first and the sum rounded after, and
				// jvmStyleOf gets `float + double` backwards.
				return w(rt.jvmArith('+', l, r))
			}
			// Two integral operands: int + int wraps at 32 bits, anything with a
			// long in it at 64 (JLS 15.18.2 on top of the 5.6.2 promotion).
			if jvIsIntegral(l) && jvIsIntegral(r) {
				return w(rt.jvArith("+", l, r))
			}
			return w(rt.jsAdd(l, r))
		}
	})
	// js_jvsub reads its argument array in position 1, so the handle must arrive
	// as the array itself rather than as a value.
	jsThroughArgs["js_jvsub"] = 1 << 1
}

// jvEqKind is the java TYPE of a primitive value, for Object#equals: 0 means "not
// a primitive" (an instance, an array, null, undefined) and every other number is
// one boxed type. The split between 1 and 2 is Float vs Double, which
// Float.equals(Double) rejects.
func jvEqKind(v interface{}) int {
	switch t := v.(type) {
	case jsJFlo:
		if t.sty == floJavaF {
			return 1
		}
		return 2
	case jsGInt:
		return 3
	case jsChar:
		return 4
	case bool:
		return 5
	case string:
		return 6
	case float64:
		return 7
	}
	return 0
}

// jvStaticOwner finds the class descriptor in cls' __super chain that carries
// `name` as an own property. In the COMPILED world a static field is an ordinary
// property of the descriptor (the emitter js_set's it there), so the walk is the
// same __super walk method dispatch does, with the guard < 64 cap chapter 7.9 of
// the manual keeps on every one of them.
func jvStaticOwner(cls interface{}, name string) *jsObject {
	for guard := 0; guard < 64; guard++ {
		clsObj, ok := cls.(*jsObject)
		if !ok {
			return nil
		}
		if pv, has := clsObj.props[name]; has && pv != interface{}(jsUndef) {
			return clsObj
		}
		cls = clsObj.props["__super"]
	}
	return nil
}

// jvClassIs answers whether cls, or anything it extends, is named nm - what
// Outer.this walks out until it finds.
func jvClassIs(cls interface{}, nm string) bool {
	for guard := 0; guard < 64; guard++ {
		clsObj, ok := cls.(*jsObject)
		if !ok {
			return false
		}
		if n, _ := clsObj.props["__name"].(string); n == nm {
			return true
		}
		cls = clsObj.props["__super"]
	}
	return false
}

// jvpFirst renders the first element of a print call's argument array. An empty
// array is the no-argument System.out.println(), which prints nothing but the
// terminator.
func jvpFirst(rt *jsrt, args interface{}) string {
	arr, ok := args.(*jsArray)
	if !ok {
		return rt.jvpStr(args)
	}
	if len(arr.elems) == 0 {
		return ""
	}
	return rt.jvpStr(arr.elems[0])
}

// ----------------------------------------------------------------------------
// Rendering

// jvpStr is String.valueOf: what println writes and what `+` concatenates.
func (rt *jsrt) jvpStr(v interface{}) string { return rt.jvpRender(v, 0) }

func (rt *jsrt) jvpRender(v interface{}, depth int) string {
	if depth > 32 {
		return "..."
	}
	switch t := v.(type) {
	case jsNullT, jsUndefT:
		return "null"
	case string:
		return t
	case *jsArray:
		return jvpArrayTag(t) + "@" + jvpHash(t)
	case *jsObject:
		return rt.jvpObj(t, depth)
	}
	return rt.toString(v)
}

// jvpObj renders the object shapes the two Java grammars build.
func (rt *jsrt) jvpObj(o *jsObject, depth int) string {
	// A user toString() wins over every built-in rendering, as it does in Java.
	if f, ok := jvpFindMember(o, "toString"); ok {
		return rt.jvpRender(rt.call(f, o, []interface{}{o}), depth+1)
	}
	// The class descriptor itself (println(Box.class) has no spelling here, but
	// the interpreter half renders one this way, so the two agree).
	if tag, ok := o.props["__isclass"]; ok && tag == true {
		if n, ok := o.props["__name"].(string); ok {
			return "class " + n
		}
	}
	cls, _ := o.props["__class"].(*jsObject)
	if cls == nil {
		return rt.toString(o)
	}
	// An enum constant without an override prints its constant name (the
	// java.lang.Enum#toString contract).
	if n, ok := o.props["__ename"].(string); ok {
		return n
	}
	// A record prints its canonical form, P[x=1, y=2] (the java.lang.Record
	// contract). __record is the component name list the class carries. The name
	// here is the SIMPLE one - the generated toString uses getSimpleName, so a
	// nested record is `P[...]` and not `Main$P[...]`.
	if comps, ok := cls.props["__record"].(*jsArray); ok {
		parts := make([]string, 0, len(comps.elems))
		for _, c := range comps.elems {
			name, _ := c.(string)
			parts = append(parts, name+"="+rt.jvpRender(o.props[name], depth+1))
		}
		simple, _ := cls.props["__name"].(string)
		return simple + "[" + strings.Join(parts, ", ") + "]"
	}
	return jvpClassName(cls) + "@" + jvpHash(o)
}

// jvpClassName is the class' BINARY name: a nested type is spelled Outer$Inner,
// the way java.lang.Class#getName does, and the grammars record that as __bname
// while they walk the declaration. A class that carries none (an anonymous or
// locally built descriptor) falls back to its simple name.
func jvpClassName(cls *jsObject) string {
	if b, ok := cls.props["__bname"].(string); ok && b != "" {
		return b
	}
	if n, ok := cls.props["__name"].(string); ok {
		return n
	}
	return "Object"
}

// jvpFindMember walks the __class / __super chain for a callable member, the same
// walk the two grammars' method dispatch does.
func jvpFindMember(o *jsObject, name string) (interface{}, bool) {
	for cls := o.props["__class"]; cls != nil; {
		clsObj, ok := cls.(*jsObject)
		if !ok {
			break
		}
		if mth, ok := clsObj.props[name]; ok && isCallable(mth) {
			return mth, true
		}
		cls = clsObj.props["__super"]
	}
	return nil, false
}

// jvpArrayTag is the array's class name as getName spells it. The element type is
// INFERRED from the elements (see the header): this value model does not carry the
// declared one.
func jvpArrayTag(a *jsArray) string {
	tag := ""
	for _, e := range a.elems {
		t := jvpElemTag(e)
		if t == "" || (tag != "" && t != tag) {
			return "[Ljava.lang.Object;"
		}
		tag = t
	}
	if tag == "" {
		return "[Ljava.lang.Object;"
	}
	return tag
}

func jvpElemTag(v interface{}) string {
	switch t := v.(type) {
	case jsGInt:
		// A sized integer carries its own width, so long[] / short[] / byte[] are
		// the one array kind whose element type is not a guess.
		switch t.w {
		case 64:
			return "[J"
		case 16:
			return "[S"
		}
		return "[B"
	case jsJFlo:
		if t.sty == floJavaF {
			return "[F"
		}
		return "[D"
	case jsChar:
		return "[C"
	case bool:
		return "[Z"
	case string:
		return "[Ljava.lang.String;"
	case float64:
		return "[I"
	}
	return ""
}

// ----------------------------------------------------------------------------
// Identity hash

// jvpIdent gives every object rendered with an `@` a stable per-run number. Real
// java prints System.identityHashCode, which is an address-derived value that
// changes between two runs of the same program - so it is not reproducible and no
// test can assert it. What the two halves must agree on is that the same object
// renders identically twice within a run and that the run as a whole is
// deterministic, which the counter provides. One compile runs one program, so a
// package-level counter is per-program.
var jvpIdent = map[interface{}]int{}
var jvpIdentN int

// The NUMBER behind that rendering, which is exactly what Object.hashCode
// answers in real java: `o.toString()` is getName() + "@" +
// Integer.toHexString(hashCode()), so the two must agree, and here they do by
// construction. jvpHash below is this value printed with %x.
func jvpIdentNum(v interface{}) float64 {
	n, ok := jvpIdent[v]
	if !ok {
		jvpIdentN++
		n = jvpIdentN
		jvpIdent[v] = n
	}
	return float64(0x1a2b3c00 + n)
}

func jvpHash(v interface{}) string {
	return fmt.Sprintf("%x", int(jvpIdentNum(v)))
}

// ----------------------------------------------------------------------------
// hashCode
//
// java.lang.Object#hashCode and the overrides of it that the JLS pins to an
// exact value. THE COMBINATION A RECORD USES IS NOT ONE OF THEM: JLS 8.10.3
// requires only that a record's hashCode be derived from the components'
// hashCodes and that equal records hash equally - the exact function "is
// unspecified". What IS specified exactly, and reproduced here bit for bit:
//
//	Boolean    true -> 1231, false -> 1237                       (JDK javadoc)
//	Integer    the value itself; Byte/Short/Character widen to it
//	Long       (int)(value ^ (value >>> 32))
//	Double     bits = doubleToLongBits(v); (int)(bits ^ (bits >>> 32))
//	Float      floatToIntBits(v)
//	String     s[0]*31^(n-1) + s[1]*31^(n-2) + ... , over UTF-16 units
//	null       0, via Objects.hashCode
//
// For the record COMBINATION we pin OpenJDK's, which is what
// java.lang.runtime.ObjectMethods generates: `h = 0; for each component h =
// h*31 + hash(component)`, at int width. Measured against java 24.0.2 on this
// machine rather than assumed - `record R(int a, String b, double c)` with
// (1, "x", 2.5) answers 1074008649 there and 1074008649 here. It is pinned
// because the invariant that MUST hold (equal records hash equally) does not
// choose a value, and two engines have to agree on one.
//
// The twin is js_jhash in languages/lib/java-rt.metajs, arm for arm; that one
// cannot call math.Float64bits and takes the bit pattern apart with exact
// arithmetic instead.
func jvHashDouble(f float64) float64 {
	bits := math.Float64bits(f)
	return float64(int32(uint32(bits>>32) ^ uint32(bits)))
}

func (rt *jsrt) jvHashString(s string) float64 {
	h := int32(0)
	n := rt.strLen(s)
	for i := 0; i < n; i++ {
		h = h*31 + int32(rt.strCodeAt(s, i))
	}
	return float64(h)
}

func (rt *jsrt) jvHash(v interface{}) float64 {
	switch x := v.(type) {
	case nil, jsUndefT, jsNullT:
		return 0
	case bool:
		if x {
			return 1231
		}
		return 1237
	case string:
		return rt.jvHashString(x)
	case float64:
		// A plain number is an INT in this value model (java never unboxes a
		// long - see chapter 7.5 of the manual), and Integer.hashCode is the
		// value itself.
		return float64(int32(jsToInt(x)))
	case jsChar:
		return float64(x.code)
	case jsGInt:
		// Long.hashCode. A byte/short/int box narrows to the same answer,
		// because its high half is the sign extension the value already has.
		u := uint64(x.v)
		return float64(int32(uint32(u>>32) ^ uint32(u)))
	case jsJFlo:
		if x.sty == floJavaF {
			return float64(int32(math.Float32bits(float32(x.f))))
		}
		return jvHashDouble(x.f)
	}
	if o, ok := v.(*jsObject); ok {
		if cls, has := o.props["__class"]; has {
			// A user-declared (or inherited) hashCode wins, exactly as an
			// override does. The __class/__super walk is js_jvaleq's, cap and
			// all.
			for guard := 0; guard < 64; guard++ {
				clsObj, isObj := cls.(*jsObject)
				if !isObj {
					break
				}
				if h, hit := clsObj.props["hashCode"]; hit && isCallable(h) {
					return rt.toNumber(rt.call(h, jsUndef, []interface{}{v}))
				}
				if comps, isRec := clsObj.props["__record"]; isRec {
					return rt.jvRecordHash(v, comps)
				}
				cls = clsObj.props["__super"]
			}
		}
	}
	return jvpIdentNum(v)
}

// h = 0; h = h*31 + Objects.hashCode(component), at int width - OpenJDK's.
func (rt *jsrt) jvRecordHash(v interface{}, comps interface{}) float64 {
	arr, ok := comps.(*jsArray)
	if !ok {
		return jvpIdentNum(v)
	}
	o, ok := v.(*jsObject)
	if !ok {
		return jvpIdentNum(v)
	}
	h := int32(0)
	for _, c := range arr.elems {
		name, _ := c.(string)
		h = h*31 + int32(rt.jvHash(o.props[name]))
	}
	return float64(h)
}
