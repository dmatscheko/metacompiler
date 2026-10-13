package abnf

// jsrtjsbig.go -- ECMAScript BigInt for the JavaScript compiler grammar.
//
// abnf/jsrt.go already carries the jsBigInt VALUE (a math/big Int, typeof "bigint",
// exact strictEq, exact + - * / % between two of them). This file adds the rest of
// the type's behaviour: an exact `**`, the bitwise and shift operators on the two's
// complement value (no ToInt32 truncation), `==` and `<` across the BigInt/Number
// line, the mixed-operand TypeError of the spec, the BigInt() global and `~`.
//
// Everything here is ADDITIVE in the sense abnf/jsrtjsprint.go established: only new
// js_js* externs are registered, each delegating to the extern it extends for every
// operand pair it does not claim.
//
// It is also SCOPED. The BigInt value type is shared with python-to-llvm-ir.abnf,
// which creates one for any integer literal past 2^53 and then mixes it with plain
// doubles on every line (`10**30 + 1`), so raising the spec's mixed-operand TypeError
// unconditionally would break Python outright. The strict rules below are therefore
// armed by `js_jsbigint`, the literal external only the js and typescript compilers
// emit; until a program evaluates a BigInt literal through it, every wrapper here is
// a straight pass-through to the extern it wraps.
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
	"unicode/utf16"
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
// the same *jsThrown panic js_throw builds. Every rt.fail in this file goes through
// here except the "invalid BigInt literal" one, which is a malformed-input abort and
// not reachable from a program that parsed. The value is a real Error INSTANCE
// whenever the emitter registered the hierarchy (jsErrFromText finds the class the
// message text already names), so e.message and `e instanceof TypeError` answer;
// `"" + e` is the message text either way.
func (rt *jsrt) bigRaise(format string, args ...interface{}) {
	panic(&jsThrown{value: rt.jsErrFromText(fmt.Sprintf(format, args...))})
}

// bigMixFail is the spec's TypeError for an operator that got one BigInt and one
// value of another type. It is raised only in strict mode (see the file comment).
func (rt *jsrt) bigMixFail(op string) {
	// node's message has NO operator suffix - it is the same text for every
	// operator.
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
	// Python compiler (and any program without a BigInt literal) on the plain operators.
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
	// of the extern `base`. A symbol operand is ToNumber's TypeError.
	// An operand that is not a number goes through ToNumber first (rt.jsvToNum): the
	// shared operator's own conversion knows neither ToPrimitive nor StringToNumber.
	binary := func(name, baseName, op string) {
		base := m[baseName]
		m[name] = func(a []uint64) uint64 {
			l, r := u(a[0]), u(a[1])
			if strict {
				if res, ok := rt.bigBinary(op, l, r); ok {
					return w(res)
				}
			}
			_, lf := l.(float64)
			_, rf := r.(float64)
			if lf && rf {
				return base(a)
			}
			rt.jsvSymOperand(l, r)
			return base([]uint64{rt.wrapNum(rt.jsvToNum(l)), rt.wrapNum(rt.jsvToNum(r))})
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
				lp, rp := rt.jsvToPrimitive(l, "default"), rt.jsvToPrimitive(r, "default")
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

	// '**' has no shared external. Its double arm is JavaScript's Math.pow (jsPowD over
	// ToNumber), the same function js_jsmath installs as the program's Math.pow.
	m["js_jspow"] = func(a []uint64) uint64 {
		if strict {
			if r, ok := rt.bigBinary("**", u(a[0]), u(a[1])); ok {
				return w(r)
			}
		}
		rt.jsvSymOperand(u(a[0]), u(a[1]))
		return rt.wrapNum(jsPowD(rt.jsvToNum(u(a[0])), rt.jsvToNum(u(a[1]))))
	}
	// The program's Math global: the shared Math object's members, copied in order,
	// with pow replaced by the ECMAScript one. The shared object stays C99's pow for the
	// other languages. Twins: js_jsmath in languages/lib/js-rt.metajs, jsMathObj in
	// both *-interpreter.abnf start scripts.
	//
	// The members are NOT ENUMERABLE, as in node (`for (k in Math)` visits nothing),
	// so they go into props without a key-order entry: every key list skips them,
	// and a delete-then-write makes the name an ordinary enumerable property. Layer
	// 2 cannot hold an unlisted key and hides them by name instead (jvMathHid).
	m["js_jsmath"] = func(a []uint64) uint64 {
		o := newJSObject()
		if src, ok := u(a[0]).(*jsObject); ok {
			for _, k := range src.keys {
				o.props[k] = src.props[k]
			}
		}
		o.props["__strtag"] = "Math"
		// Every member converts its arguments with JavaScript's ToNumber first.
		for k, v := range o.props {
			if f, ok := v.(*hostFunc); ok {
				o.props[k] = rt.jsvNumArgs(f)
			}
		}
		o.props["pow"] = jsHostFunc("pow", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return jsPowD(rt.jsvToNum(argAt(args, 0)), rt.jsvToNum(argAt(args, 1)))
		})
		return w(o)
	}

	// ~x on a BigInt is exactly -x - 1 in arbitrary precision, so it gets an external
	// of its own: `x ^ -1` on the shared xor would be refused by the mixed-operand rule.
	baseBxor := m["js_bxor"]
	m["js_jsbnot"] = func(a []uint64) uint64 {
		if strict {
			if x, ok := bigOf(u(a[0])); ok {
				return w(bigWrap(new(big.Int).Not(x)))
			}
		}
		rt.jsvSymOperand(u(a[0]), nil)
		return baseBxor([]uint64{rt.wrapNum(rt.jsvToNum(u(a[0]))), rt.wrapNum(-1)})
	}

	// Unary '+' is the ONE operator a BigInt refuses outright: there is no number to
	// coerce it to without losing precision.
	m["js_jstonumber"] = func(a []uint64) uint64 {
		if strict {
			if _, ok := bigOf(u(a[0])); ok {
				rt.bigRaise("TypeError: Cannot convert a BigInt value to a number")
			}
		}
		rt.jsvSymOperand(u(a[0]), nil)
		return rt.wrapNum(rt.jsvToNum(u(a[0])))
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
		rt.jsvSymOperand(u(a[0]), nil)
		return rt.wrapNum(rt.jsvToNum(u(a[0])))
	}
	// Unary '-': the shared js_neg, which knows a BigInt, after the symbol test.
	// Twin of js_jsneg in languages/lib/js-rt.metajs.
	baseNeg := m["js_neg"]
	m["js_jsneg"] = func(a []uint64) uint64 {
		v := u(a[0])
		rt.jsvSymOperand(v, nil)
		if _, ok := v.(float64); ok {
			return baseNeg(a)
		}
		if _, ok := bigOf(v); ok {
			return baseNeg(a)
		}
		return baseNeg([]uint64{rt.wrapNum(rt.jsvToNum(v))})
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
		if r, ok := bigLooseEq(rt.jsvToPrimitive(u(a[0]), "default"), rt.jsvToPrimitive(u(a[1]), "default")); ok {
			return boolH(r)
		}
		return eqBase(a)
	}
	m["js_jsne"] = func(a []uint64) uint64 {
		if r, ok := bigLooseEq(rt.jsvToPrimitive(u(a[0]), "default"), rt.jsvToPrimitive(u(a[1]), "default")); ok {
			return boolH(!r)
		}
		return neBase(a)
	}
	rel := func(name string, want func(int) bool) {
		base := m[name]
		m[name] = func(a []uint64) uint64 {
			if c, ok := bigCompare(rt.jsvToPrimitive(u(a[0]), "number"), rt.jsvToPrimitive(u(a[1]), "number")); ok {
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
			return rt.jsvToNum(args[0])
		}))
	}
	// The parseFloat global: JavaScript's, not the shared host function's.
	m["js_jsparsefloatfn"] = func(a []uint64) uint64 {
		return w(jsHostFunc("parseFloat", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			return jsParseFloatJS(rt.jsvString(argAt(args, 0)))
		}))
	}
	// The parseInt global, the same way.
	m["js_jsparseintfn"] = func(a []uint64) uint64 {
		return w(jsHostFunc("parseInt", func(rt *jsrt, this uint64, args []interface{}) interface{} {
			s := rt.jsvString(argAt(args, 0))
			return jsParseIntJS(s, rt.toInt32(rt.jsvToNum(argAt(args, 1))))
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
			rt.bigRaise("RangeError: toString() radix argument must be between 2 and 36")
		}
		return x.Text(radix), true
	case "valueOf":
		return bigWrap(x), true
	}
	return nil, false
}

// jsPowD is ECMAScript's Number::exponentiate. It differs from C99 pow (math.Pow) in
// two places only: a NaN exponent gives NaN even for base 1, and (+-1) ** (+-Infinity)
// is NaN, not 1. Twins: jvPow in languages/lib/js-rt.metajs, jsPow in both
// *-interpreter.abnf start scripts.
func jsPowD(x, y float64) float64 {
	if math.IsNaN(y) || (math.IsInf(y, 0) && math.Abs(x) == 1) {
		return math.NaN()
	}
	return crPow(x, y)
}

// jsvToNum is JavaScript's ToNumber: an object through ToPrimitive with hint number,
// a string through StringToNumber, everything else through the shared conversion.
// A BigInt reached here (a valueOf answering one) reads NaN, as the shared conversion
// does; the operators settle a BigInt operand before they ask.
func (rt *jsrt) jsvToNum(v interface{}) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case string:
		return jsStrToNum(t)
	case *jsObject, *jsArray:
		if _, ok := bigOf(v); ok {
			return math.NaN()
		}
		rt.jsvSymOperand(v, nil)
		return rt.jsvToNum(rt.jsvToPrimitive(v, "number"))
	}
	return rt.toNumber(v)
}

// jsvNumArgs wraps a Math member so that its arguments reach it as numbers.
func (rt *jsrt) jsvNumArgs(f *hostFunc) *hostFunc {
	return jsHostFunc(f.name, func(rt *jsrt, this uint64, args []interface{}) interface{} {
		conv := make([]interface{}, len(args))
		for i, x := range args {
			conv[i] = rt.jsvToNum(x)
		}
		return f.fn(rt, this, conv)
	})
}

// jsIsWsJS is WhiteSpace or LineTerminator (ES2023 12.2, 12.3). Twin of jsnIsWs in
// languages/lib/js-num.js.
func jsIsWsJS(c uint16) bool {
	switch {
	case c == 32 || (c >= 9 && c <= 13) || c == 160 || c == 5760 || c == 8232 || c == 8233:
		return true
	case c >= 8192 && c <= 8202:
		return true
	}
	return c == 8239 || c == 8287 || c == 12288 || c == 65279
}

func jsDigitValJS(c uint16) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'z':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'Z':
		return int(c-'A') + 10
	}
	return 99
}

// jsDecLenJS is the length of the longest StrUnsignedDecimalLiteral at u[i:]. Twin of
// jsnDecLen in languages/lib/js-num.js.
func jsDecLenJS(u []uint16, i int) int {
	if i+8 <= len(u) && string(utf16.Decode(u[i:i+8])) == "Infinity" {
		return 8
	}
	j, nd := i, 0
	for j < len(u) && jsDigitValJS(u[j]) < 10 {
		j++
		nd++
	}
	if j < len(u) && u[j] == '.' {
		k := j + 1
		for k < len(u) && jsDigitValJS(u[k]) < 10 {
			k++
			nd++
		}
		if nd == 0 {
			return 0
		}
		j = k
	}
	if nd == 0 {
		return 0
	}
	if j < len(u) && (u[j] == 'e' || u[j] == 'E') {
		e := j + 1
		if e < len(u) && (u[e] == '+' || u[e] == '-') {
			e++
		}
		ed := e
		for ed < len(u) && jsDigitValJS(u[ed]) < 10 {
			ed++
		}
		if ed > e {
			j = ed
		}
	}
	return j - i
}

// jsDecValJS is the value of a validated, possibly signed decimal literal.
func jsDecValJS(t string) float64 {
	neg := strings.HasPrefix(t, "-")
	body := strings.TrimLeft(t, "+-")
	v := math.Inf(1)
	if body != "Infinity" {
		v, _ = strconv.ParseFloat(body, 64) // "1e400" is +Inf with ErrRange: the value is right
	}
	if neg {
		return -v
	}
	return v
}

// jsStrToNum is StringToNumber (ES2023 7.1.4.1.1). Twin of jsnStrToNum in
// languages/lib/js-num.js.
func jsStrToNum(s string) float64 {
	u := utf16.Encode([]rune(s))
	i, j := 0, len(u)
	for i < j && jsIsWsJS(u[i]) {
		i++
	}
	for j > i && jsIsWsJS(u[j-1]) {
		j--
	}
	if i == j {
		return 0
	}
	t := u[i:j]
	if len(t) > 2 && t[0] == '0' {
		radix := 0
		switch t[1] {
		case 'x', 'X':
			radix = 16
		case 'o', 'O':
			radix = 8
		case 'b', 'B':
			radix = 2
		}
		if radix > 0 {
			v := 0.0
			for _, c := range t[2:] {
				d := jsDigitValJS(c)
				if d >= radix {
					return math.NaN()
				}
				v = v*float64(radix) + float64(d)
			}
			return v
		}
	}
	at := 0
	if t[0] == '+' || t[0] == '-' {
		at = 1
	}
	n := jsDecLenJS(t, at)
	if n == 0 || at+n != len(t) {
		return math.NaN()
	}
	return jsDecValJS(string(utf16.Decode(t)))
}

// jsParseIntJS is the global parseInt (ES2023 19.2.5), r being ToInt32(radix). Twin
// of jsnParseInt in languages/lib/js-num.js, where the rules are.
func jsParseIntJS(s string, r int32) float64 {
	u := utf16.Encode([]rune(s))
	i := 0
	for i < len(u) && jsIsWsJS(u[i]) {
		i++
	}
	neg := false
	if i < len(u) && (u[i] == '+' || u[i] == '-') {
		neg = u[i] == '-'
		i++
	}
	radix, prefix := int(r), true
	if radix != 0 {
		if radix < 2 || radix > 36 {
			return math.NaN()
		}
		prefix = radix == 16
	} else {
		radix = 10
	}
	if prefix && i+1 < len(u) && u[i] == '0' && (u[i+1] == 'x' || u[i+1] == 'X') {
		i += 2
		radix = 16
	}
	digit := func(c uint16) int {
		if c >= 128 || digitValue(byte(c)) < 0 {
			return 99
		}
		return digitValue(byte(c))
	}
	start := i
	for i < len(u) && digit(u[i]) < radix {
		i++
	}
	if i == start {
		return math.NaN()
	}
	var v float64
	switch radix {
	case 10:
		v = jsDecValJS(string(utf16.Decode(u[start:i])))
	case 2, 4, 8, 16, 32:
		// The first 53 significant bits, then the first dropped bit and whether
		// any later one is set: one rounding, half to even.
		nb, exp, rnd, sticky := 0, 0, 0, false
		for _, c := range u[start:i] {
			d := digit(c)
			for p := radix / 2; p >= 1; p /= 2 {
				bit := 0
				if d >= p {
					bit, d = 1, d-p
				}
				if nb < 53 {
					if nb > 0 || bit == 1 {
						v = v*2 + float64(bit)
						nb++
					}
				} else {
					if exp == 0 {
						rnd = bit
					} else if bit == 1 {
						sticky = true
					}
					exp++
				}
			}
		}
		if rnd == 1 && (sticky || math.Mod(v, 2) == 1) {
			v++
		}
		if exp > 0 {
			v *= math.Pow(2, float64(exp))
		}
	default:
		// V8's chunks, each folded in with one rounding (jsnIntGeneric).
		k := start
		for k < i && u[k] == '0' {
			k++
		}
		for k < i {
			part, mult := 0, 1
			for k < i && mult*radix <= 119304647 {
				part = part*radix + digit(u[k])
				mult *= radix
				k++
			}
			v = math.FMA(v, float64(mult), float64(part))
		}
	}
	if neg {
		return -v
	}
	return v
}

// jsParseFloatJS is the global parseFloat (ES2023 19.2.4). Twin of jsnParseFloat in
// languages/lib/js-num.js.
func jsParseFloatJS(s string) float64 {
	u := utf16.Encode([]rune(s))
	i := 0
	for i < len(u) && jsIsWsJS(u[i]) {
		i++
	}
	at := i
	if at < len(u) && (u[at] == '+' || u[at] == '-') {
		at++
	}
	n := jsDecLenJS(u, at)
	if n == 0 {
		return math.NaN()
	}
	return jsDecValJS(string(utf16.Decode(u[i : at+n])))
}
