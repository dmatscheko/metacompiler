package abnf

// jsrtjsbig.go -- ECMAScript BigInt for the JavaScript compiler grammar.
//
// abnf/jsrt.go already carries the jsBigInt VALUE (a math/big Int, typeof "bigint",
// exact strictEq, exact + - * / % between two of them). What it does not carry is the
// rest of the type's behaviour, and the rest is most of it:
//
//	2n ** 64n     was NaN         (** lowers to Math.pow, which has only doubles)
//	1n & 3n       was 1 as a NUMBER (the bitwise externs go through ToInt32)
//	1n << 10n     was 1024 as a NUMBER, and it silently truncated past 32 bits
//	10n == 10     was false       (loose equality never crossed the type line)
//	10n < 11      was false       (the relational path needs BOTH to be BigInts)
//	1n + 1        was 2           (the spec says TypeError)
//	BigInt(42)    was "variable not defined"
//	~1n           was -2 as a NUMBER
//
// Everything here is ADDITIVE in the sense abnf/jsrtjsprint.go established: only new
// js_js* externs are registered, each delegating to the extern it extends for every
// operand pair it does not claim.
//
// It is also SCOPED. The BigInt value type is shared with two other grammars:
// python-to-llvm-ir.abnf creates one for any integer literal past 2^53 and then mixes
// it with plain doubles on every line (`10**30 + 1`), and typescript-to-llvm-ir.abnf
// has the older lenient behaviour. Raising the spec's mixed-operand TypeError
// unconditionally would break Python outright. So the strict rules below are armed by
// `js_jsbigint`, the literal external that ONLY languages/js-to-llvm-ir.abnf emits;
// until a program evaluates a BigInt literal through it, every wrapper here is a
// straight pass-through to the extern it wraps.
//
// languages/js-interpreter.abnf implements the identical semantics in its tag script
// (the "BigInt" section, base 2^24 limb arithmetic); ./test.sh --full runs
// tests/js-test-full.js through both halves and compares them. Every answer below was
// settled against node v24.

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
)

// bigOf answers v as a *big.Int when it is a BigInt.
func bigOf(v interface{}) (*big.Int, bool) {
	b, ok := v.(*jsBigInt)
	if !ok {
		return nil, false
	}
	return b.v, true
}

func bigWrap(v *big.Int) interface{} { return &jsBigInt{v: v} }

// bigToFloat is the double a BigInt denotes - exact up to 2^53, the nearest double
// above it. It is what Number(x) answers and what a mixed comparison falls back on.
func bigToFloat(v *big.Int) float64 {
	f, _ := new(big.Float).SetInt(v).Float64()
	return f
}

// bigFromFloat is ToBigInt for a number: only an integral, finite value has one.
func (rt *jsrt) bigFromFloat(f float64) *big.Int {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		rt.bigRaise("RangeError: cannot convert %v to a BigInt", f)
	}
	if f != math.Trunc(f) {
		rt.bigRaise("RangeError: The number %v cannot be converted to a BigInt because it is not an integer", f)
	}
	out, _ := new(big.Float).SetFloat64(f).Int(nil)
	return out
}

// bigOfValue is ToBigInt, the conversion the BigInt(v) global performs.
func (rt *jsrt) bigOfValue(v interface{}) *big.Int {
	switch t := v.(type) {
	case *jsBigInt:
		return t.v
	case bool:
		if t {
			return big.NewInt(1)
		}
		return big.NewInt(0)
	case float64:
		return rt.bigFromFloat(t)
	case string:
		n, ok := bigFromLiteral(t)
		if !ok {
			rt.bigRaise("SyntaxError: Cannot convert %s to a BigInt", t)
		}
		return n
	}
	rt.bigRaise("TypeError: Cannot convert %s to a BigInt", rt.toString(v))
	return nil
}

// bigFromLiteral is StringToBigInt: the WHOLE string must be an integer literal, so
// "10" converts and "10.0" has no value at all - which is what makes 10n == "10.0"
// false in node while 10n == "10" is true. An empty (or blank) string is 0n.
func bigFromLiteral(s string) (*big.Int, bool) {
	t := ""
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\v' || r == '\f' || r == 0xa0 || r == 0xfeff {
			continue
		}
		t += string(r)
	}
	if t == "" {
		return big.NewInt(0), true
	}
	// big.Int's base-0 parser accepts exactly the literal forms BigInt() does
	// (0x/0b/0o and plain decimal) but ALSO '_' separators, which StringToBigInt
	// does not, so they are refused here.
	for i := 0; i < len(t); i++ {
		if t[i] == '_' {
			return nil, false
		}
	}
	n, ok := new(big.Int).SetString(t, 0)
	if !ok {
		return nil, false
	}
	return n, true
}

// bigRaise raises a BigInt operation's error as a JAVASCRIPT exception rather than
// an engine abort, which is what node does with every one of them: `1n + 1` inside a
// try/catch runs the catch clause. rt.fail panics a plain string that js_try
// deliberately re-panics (it is a runtime error, not a throw), so the raise has to be
// the same *jsThrown panic js_throw builds - and the thrown VALUE is the message
// string, the spelling both interpreter halves and the emitters already use for the
// iterator-throw TypeError. Every rt.fail in this file goes through here except the
// "invalid BigInt literal" one, which is a malformed-input abort and not reachable
// from a program that parsed. docs/todo.md 2.6.
// Since docs/todo.md 1.1 the value is a real Error INSTANCE whenever the emitter
// registered the hierarchy (jsErrFromText finds the class the message text
// already names), so e.message and `e instanceof TypeError` answer; `"" + e` is
// unchanged, because Error.prototype.toString rebuilds exactly this text.
func (rt *jsrt) bigRaise(format string, args ...interface{}) {
	panic(&jsThrown{value: rt.jsErrFromText(fmt.Sprintf(format, args...))})
}

// bigMixFail is the spec's TypeError for an operator that got one BigInt and one
// value of another type. It is raised only in strict mode (see the file comment).
func (rt *jsrt) bigMixFail(op string) {
	// node's message has NO operator suffix - it is the same text for every
	// operator. docs/todo.md 1.8.
	_ = op
	rt.bigRaise("TypeError: Cannot mix BigInt and other types, use explicit conversions")
}

// bigBinary is the BigInt arm of a binary arithmetic or bitwise operator. handled is
// false when NEITHER operand is a BigInt, and the caller delegates to the double
// path; when exactly one is, the mixed-operand TypeError is raised instead.
func (rt *jsrt) bigBinary(op string, l, r interface{}) (interface{}, bool) {
	x, lok := bigOf(l)
	y, rok := bigOf(r)
	if !lok && !rok {
		return nil, false
	}
	if !lok || !rok {
		rt.bigMixFail(op)
	}
	out := new(big.Int)
	switch op {
	case "+":
		out.Add(x, y)
	case "-":
		out.Sub(x, y)
	case "*":
		out.Mul(x, y)
	case "/":
		if y.Sign() == 0 {
			rt.bigRaise("RangeError: Division by zero")
		}
		out.Quo(x, y) // BigInt division truncates toward zero.
	case "%":
		if y.Sign() == 0 {
			rt.bigRaise("RangeError: Division by zero")
		}
		out.Rem(x, y) // The remainder takes the sign of the dividend.
	case "**":
		if y.Sign() < 0 {
			rt.bigRaise("RangeError: Exponent must be non-negative")
		}
		if !y.IsInt64() {
			rt.bigRaise("RangeError: BigInt exponent is too large")
		}
		out.Exp(x, y, nil)
	case "&":
		out.And(x, y)
	case "|":
		out.Or(x, y)
	case "^":
		out.Xor(x, y)
	case "<<", ">>":
		if !y.IsInt64() {
			rt.bigRaise("RangeError: BigInt shift count is too large")
		}
		n := y.Int64()
		neg := n < 0
		if neg {
			n = -n
		}
		if n > 1<<20 {
			rt.bigRaise("RangeError: BigInt shift count is too large")
		}
		// big.Int's Rsh is already the ARITHMETIC shift BigInt's >> is, so
		// -1n >> 1n is -1n rather than 0n.
		if (op == "<<") != neg {
			out.Lsh(x, uint(n))
		} else {
			out.Rsh(x, uint(n))
		}
	case ">>>":
		rt.bigRaise("TypeError: BigInts have no unsigned right shift, use >> instead")
	default:
		return nil, false
	}
	return bigWrap(out), true
}

// bigLooseEq is '==' across the BigInt/Number line, the one comparison that crosses
// it. It compares MATHEMATICALLY: 1n == 1 and 1n == "1" hold, 1n == "1.0" does not.
// handled is false when neither operand is a BigInt.
func bigLooseEq(a, b interface{}) (bool, bool) {
	x, aok := bigOf(a)
	y, bok := bigOf(b)
	if !aok && !bok {
		return false, false
	}
	if aok && bok {
		return x.Cmp(y) == 0, true
	}
	n := x
	other := b
	if !aok {
		n = y
		other = a
	}
	switch t := other.(type) {
	case string:
		p, ok := bigFromLiteral(t)
		if !ok {
			return false, true
		}
		return n.Cmp(p) == 0, true
	case bool:
		v := int64(0)
		if t {
			v = 1
		}
		return n.Cmp(big.NewInt(v)) == 0, true
	case float64:
		if math.IsNaN(t) || math.IsInf(t, 0) || t != math.Trunc(t) {
			return false, true
		}
		p, _ := new(big.Float).SetFloat64(t).Int(nil)
		return n.Cmp(p) == 0, true
	}
	return false, true
}

// bigCompare is the relational ordering with a BigInt on at least one side. Two
// BigInts compare exactly; a BigInt against a number goes through the double value,
// the one place a value past 2^53 could round (js-interpreter.abnf's biRel does the
// same, so both halves answer alike). 2 means "NaN was involved".
func bigCompare(a, b interface{}) (int, bool) {
	x, aok := bigOf(a)
	y, bok := bigOf(b)
	if !aok && !bok {
		return 0, false
	}
	if aok && bok {
		return x.Cmp(y), true
	}
	toF := func(v interface{}) float64 {
		switch t := v.(type) {
		case *jsBigInt:
			return bigToFloat(t.v)
		case float64:
			return t
		case bool:
			if t {
				return 1
			}
			return 0
		case string:
			if t == "" {
				return 0
			}
			f, err := strconv.ParseFloat(strings.TrimSpace(t), 64)
			if err != nil {
				return math.NaN()
			}
			return f
		case jsNullT:
			return 0 // ToNumber(null) is 0, so 1n > null holds.
		}
		return math.NaN()
	}
	af, bf := toF(a), toF(b)
	if math.IsNaN(af) || math.IsNaN(bf) {
		return 2, true
	}
	switch {
	case af < bf:
		return -1, true
	case af > bf:
		return 1, true
	}
	return 0, true
}

// addJSBigIntExterns registers the BigInt-aware operator externs. It runs at the END
// of addJSValueExterns so that the js_js* names it wraps already exist.
func (rt *jsrt) addJSBigIntExterns(m map[string]func(args []uint64) uint64) {
	u := rt.unwrap
	w := rt.wrap
	boolH := func(b bool) uint64 {
		if b {
			return jsHTrue
		}
		return jsHFalse
	}
	// One bool per runtime, armed by the js_jsbigint literal external below. While it
	// is false every wrapper here is a straight pass-through, which is what keeps the
	// Python and TypeScript compilers byte-for-byte unchanged.
	strict := false

	// The BigInt literal of languages/js-to-llvm-ir.abnf. It differs from the shared
	// js_bigint only in arming the strict rules.
	m["js_jsbigint"] = func(a []uint64) uint64 {
		v, ok := new(big.Int).SetString(rt.toString(u(a[0])), 0)
		if !ok {
			rt.fail("invalid BigInt literal %q", rt.toString(u(a[0])))
		}
		rt.hasBigInt = true
		strict = true
		return w(bigWrap(v))
	}

	// binary(name, base, op) registers `name` as `op` with the BigInt rules in front
	// of the extern `base`.
	binary := func(name, baseName, op string) {
		base := m[baseName]
		m[name] = func(a []uint64) uint64 {
			if strict {
				if r, ok := rt.bigBinary(op, u(a[0]), u(a[1])); ok {
					return w(r)
				}
			}
			return base(a)
		}
	}
	binary("js_jssub", "js_sub", "-")
	binary("js_jsmul", "js_mul", "*")
	binary("js_jsdiv", "js_div", "/")
	binary("js_jsmod", "js_mod", "%")
	binary("js_jsband", "js_band", "&")
	binary("js_jsbor", "js_bor", "|")
	binary("js_jsbxor", "js_bxor", "^")
	binary("js_jsshl", "js_shl", "<<")
	binary("js_jsshr", "js_shr", ">>")
	binary("js_jsushr", "js_ushr", ">>>")

	// '+' is the one arithmetic operator a BigInt shares with strings, so string
	// concatenation is settled before the mixed-operand rule can fire.
	baseJSAdd := m["js_jsadd"]
	m["js_jsadd"] = func(a []uint64) uint64 {
		if strict {
			l, r := u(a[0]), u(a[1])
			_, lok := bigOf(l)
			_, rok := bigOf(r)
			if lok || rok {
				lp, rp := rt.jsvToPrimitive(l), rt.jsvToPrimitive(r)
				_, lIsStr := lp.(string)
				_, rIsStr := rp.(string)
				if !lIsStr && !rIsStr {
					if res, ok := rt.bigBinary("+", lp, rp); ok {
						return w(res)
					}
				}
			}
		}
		return baseJSAdd(a)
	}

	// '**' has no shared external: it lowers to Math.pow, which only has doubles. The
	// double arm below IS Math.pow (math.Pow over ToNumber), so a program without a
	// BigInt gets bit-for-bit the answer it got before.
	m["js_jspow"] = func(a []uint64) uint64 {
		if strict {
			if r, ok := rt.bigBinary("**", u(a[0]), u(a[1])); ok {
				return w(r)
			}
		}
		return rt.wrapNum(math.Pow(rt.toNumber(u(a[0])), rt.toNumber(u(a[1]))))
	}

	// ~x on a BigInt is exactly -x - 1 in arbitrary precision. The grammar used to
	// lower it to `x ^ -1` on the shared xor, which the mixed-operand rule would now
	// (correctly) refuse, so it gets an external of its own.
	baseBxor := m["js_bxor"]
	m["js_jsbnot"] = func(a []uint64) uint64 {
		if strict {
			if x, ok := bigOf(u(a[0])); ok {
				return w(bigWrap(new(big.Int).Not(x)))
			}
		}
		return baseBxor([]uint64{a[0], rt.wrapNum(-1)})
	}

	// Unary '+' is the ONE operator a BigInt refuses outright: there is no number to
	// coerce it to without losing precision.
	baseToNum := m["js_tonum"]
	m["js_jstonumber"] = func(a []uint64) uint64 {
		if strict {
			if _, ok := bigOf(u(a[0])); ok {
				rt.bigRaise("TypeError: Cannot convert a BigInt value to a number")
			}
		}
		return baseToNum(a)
	}

	// ++ and -- are ToNumeric-then-step: a BigInt stays a BigInt and steps by 1n,
	// everything else becomes a number and steps by 1. js_jstonumeric answers the
	// value ++ reports (the OLD one, for the postfix form) and js_jsstep applies the
	// delta to it.
	m["js_jstonumeric"] = func(a []uint64) uint64 {
		if strict {
			if _, ok := bigOf(u(a[0])); ok {
				return a[0]
			}
		}
		return baseToNum(a)
	}
	m["js_jsstep"] = func(a []uint64) uint64 {
		if strict {
			if x, ok := bigOf(u(a[0])); ok {
				return w(bigWrap(new(big.Int).Add(x, big.NewInt(int64(rt.toNumber(u(a[1])))))))
			}
		}
		return rt.wrapNum(rt.toNumber(u(a[0])) + rt.toNumber(u(a[1])))
	}

	// The equality and relational operators. Only the BigInt pairs are claimed;
	// everything else delegates to the js_js* extern of abnf/jsrtjsprint.go.
	// ToPrimitive runs FIRST, so an OBJECT on the other side reaches its primitive
	// form before the BigInt rules see it: `1n == [1n]` is true in node, because the
	// array stringifies to "1" and StringToBigInt reads it back.
	eqBase, neBase := m["js_jseq"], m["js_jsne"]
	m["js_jseq"] = func(a []uint64) uint64 {
		if r, ok := bigLooseEq(rt.jsvToPrimitive(u(a[0])), rt.jsvToPrimitive(u(a[1]))); ok {
			return boolH(r)
		}
		return eqBase(a)
	}
	m["js_jsne"] = func(a []uint64) uint64 {
		if r, ok := bigLooseEq(rt.jsvToPrimitive(u(a[0])), rt.jsvToPrimitive(u(a[1]))); ok {
			return boolH(!r)
		}
		return neBase(a)
	}
	rel := func(name string, want func(int) bool) {
		base := m[name]
		m[name] = func(a []uint64) uint64 {
			if c, ok := bigCompare(rt.jsvToPrimitive(u(a[0])), rt.jsvToPrimitive(u(a[1]))); ok {
				if c == 2 {
					return jsHFalse
				}
				return boolH(want(c))
			}
			return base(a)
		}
	}
	rel("js_jslt", func(c int) bool { return c < 0 })
	rel("js_jsgt", func(c int) bool { return c > 0 })
	rel("js_jsle", func(c int) bool { return c <= 0 })
	rel("js_jsge", func(c int) bool { return c >= 0 })

	// The BigInt(v) and Number(v) globals, declared into the program scope by the
	// grammar the way String / Object / println already are.
	m["js_jsbigintfn"] = func(a []uint64) uint64 {
		return w(jsHostFunc("BigInt", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			if len(args) == 0 {
				rt.bigRaise("TypeError: Cannot convert undefined to a BigInt")
			}
			rt.hasBigInt = true
			strict = true
			return bigWrap(rt.bigOfValue(args[0]))
		}))
	}
	m["js_jsnumberfn"] = func(a []uint64) uint64 {
		return w(jsHostFunc("Number", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			if len(args) == 0 {
				return float64(0)
			}
			if x, ok := bigOf(args[0]); ok {
				return bigToFloat(x)
			}
			return rt.toNumber(args[0])
		}))
	}
}

// jsvBigIntMethod is BigInt.prototype: toString([radix]) and valueOf, the two that
// make sense without a wrapper object. Reached from jsvMethod in jsrtjsprint.go.
func (rt *jsrt) jsvBigIntMethod(x *big.Int, name string, args []interface{}) (interface{}, bool) {
	switch name {
	case "toString":
		radix := 10
		if len(args) > 0 {
			if _, isUndef := args[0].(jsUndefT); !isUndef {
				radix = int(rt.toNumber(args[0]))
			}
		}
		if radix < 2 || radix > 36 {
			rt.bigRaise("RangeError: toString() radix must be between 2 and 36")
		}
		return x.Text(radix), true
	case "valueOf":
		return bigWrap(x), true
	}
	return nil, false
}
