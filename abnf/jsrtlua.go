package abnf

// jsrtlua.go -- LUA'S NUMBER MODEL for the compiler grammar
// (languages/lua-to-llvm-ir.abnf).
//
// Lua 5.3+ has a genuine integer SUBTYPE, and it is observable in every
// direction: `3` is an integer and `3.0` is a float, `print(3.0)` writes "3.0",
// `math.type` names the two apart, `/` always produces a float while `//` and
// `%` keep the subtype, and - the part a host double cannot fake - an integer is
// a REAL 64 bit two's complement value that WRAPS:
//
//	math.maxinteger          real lua: 9223372036854775807   plain double: 9223372036854776000
//	9223372036854775807 + 1  real lua: -9223372036854775808  plain double: 9223372036854776000
//	1 << 63                  real lua: -9223372036854775808
//	0xffffffffffffffff       real lua: -1
//
// lua-interpreter.abnf answers all of these on the shared sized-integer layer of
// languages/lib/interp-core.js (szArith / szCmp / szStr, at width 64, signed).
// This file is the compiler half's twin of that layer, so ./test.sh --cross
// keeps the two in step. It reuses jsGInt from abnf/jsrtint.go rather than
// introducing a second 64 bit box - Go's sized integers and Lua's one integer
// type are the same machine value, only the surrounding rules differ.
//
// The value model, identical in both halves:
//
//	a plain float64        ==  an integer whose value is exact in a double
//	                           (|v| <= 2^53, integral), OR a float whose value is
//	                           NOT integral (1.5, inf, nan) - which is unambiguous
//	jsGInt{w:64,u:false}   ==  an integer outside that range
//	*jsObject with "__f"   ==  a float whose value IS integral (1.0, 1e100), the
//	                           only case the plain reading could not tell from an
//	                           integer. The Lua compiler grammar already created
//	                           this box in IR before this file existed, and it
//	                           stays exactly that shape so the IR that reads it
//	                           directly keeps working.
//
// Registration is additive: an init() appends a registrar to rxExtraExterns, so
// nothing here runs unless a program calls one of the js_lu* externs, which only
// the Lua compiler grammar emits.

import (
	"math"
	"strconv"
	"strings"
)

// luMinIntF is 2^63 exactly - the first magnitude that is NOT a Lua integer.
const luMinIntF = 9223372036854775808.0

// ----- the two boxes -----

// luFloBox reads the {__f} float box the Lua grammars use for an integral-valued
// float.
func luFloBox(v interface{}) (float64, bool) {
	o, ok := v.(*jsObject)
	if !ok {
		return 0, false
	}
	f, ok := o.props["__f"]
	if !ok {
		return 0, false
	}
	g, ok := f.(float64)
	return g, ok
}

// luMkFlo boxes a float that would otherwise read back as an integer. The
// infinities and NaN are floats already and never need the box.
func luMkFlo(f float64) interface{} {
	if !math.IsInf(f, 0) && !math.IsNaN(f) && f == math.Trunc(f) {
		o := newJSObject()
		o.set("__f", f)
		return o
	}
	return f
}

// luIsInt is Lua's `math.type(v) == "integer"`.
func luIsInt(v interface{}) bool {
	switch t := v.(type) {
	case jsGInt:
		return true
	case float64:
		return !math.IsInf(t, 0) && !math.IsNaN(t) && t == math.Trunc(t)
	}
	return false
}

// luIsNum is Lua's `type(v) == "number"` - either subtype, either box.
func luIsNum(v interface{}) bool {
	switch v.(type) {
	case jsGInt, float64:
		return true
	}
	_, ok := luFloBox(v)
	return ok
}

// luNumOf is the double reading of a Lua number. It loses precision above 2^53
// by construction, so only the consumers that genuinely want a double (a length,
// an index, a float operand) may use it.
func luNumOf(v interface{}) float64 {
	switch t := v.(type) {
	case jsGInt:
		return float64(t.v)
	case float64:
		return t
	}
	if f, ok := luFloBox(v); ok {
		return f
	}
	return math.NaN()
}

// luI64 turns an integral double into an integer value, or answers false when it
// does not fit the signed 64 bit range.
func luI64(n float64) (interface{}, bool) {
	if n >= luMinIntF || n < -luMinIntF {
		return nil, false
	}
	return giNorm(int64(n), 64, false), true
}

// luInt64 is the exact 64 bit payload of a value already known to be an integer.
func luInt64(v interface{}) int64 {
	if b, ok := v.(jsGInt); ok {
		return b.v
	}
	if f, ok := v.(float64); ok {
		return int64(f)
	}
	return 0
}

// ----- reading a literal / coercing a string -----

// luDigit is the value of one digit in base 16, or -1.
func luDigit(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	}
	return -1
}

// luParseInt reads a run of digits in `radix` as an exact 64 bit integer,
// answering whether the value fit. A hexadecimal literal WRAPS in Lua rather
// than overflowing to a float, so the caller decides what the overflow means.
func luParseInt(s string, radix int) (int64, bool) {
	var acc uint64
	fits := true
	for i := 0; i < len(s); i++ {
		d := luDigit(s[i])
		if d < 0 || d >= radix {
			continue
		}
		next := acc*uint64(radix) + uint64(d)
		if next/uint64(radix) != acc || next < acc {
			fits = false
		}
		acc = next
	}
	if acc > 9223372036854775807 {
		fits = false
	}
	return int64(acc), fits
}

// luParseNum reads the text of a Lua numeral - the lexer's spelling and the one
// a string goes through when it coerces in arithmetic. Surrounding blanks and a
// leading sign are tolerated (the coercion needs that); anything that is not a
// complete numeral answers false.
//
// The subtype comes from the SPELLING: a fraction dot or an exponent makes a
// float, everything else an integer. A decimal integer that does not fit 64 bits
// becomes a FLOAT (9223372036854775808 is 9.2233720368548e+18 in real Lua); a
// hexadecimal one WRAPS (0xffffffffffffffff is -1).
func luParseNum(s string) (interface{}, bool) {
	i, n := 0, len(s)
	for i < n && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r') {
		i++
	}
	for n > i && (s[n-1] == ' ' || s[n-1] == '\t' || s[n-1] == '\n' || s[n-1] == '\r') {
		n--
	}
	neg := false
	if i < n && (s[i] == '+' || s[i] == '-') {
		neg = s[i] == '-'
		i++
	}
	if i >= n {
		return nil, false
	}
	var v interface{}
	var ok bool
	if s[i] == '0' && i+1 < n && (s[i+1] == 'x' || s[i+1] == 'X') {
		v, ok = luParseHex(s[i+2 : n])
	} else {
		v, ok = luParseDec(s[i:n])
	}
	if !ok {
		return nil, false
	}
	if !neg {
		return v, true
	}
	if f, isf := luFloBox(v); isf {
		return luMkFlo(-f), true
	}
	if luIsInt(v) {
		return giNorm(-luInt64(v), 64, false), true
	}
	return -luNumOf(v), true
}

// luParseHex reads the body of a 0x numeral: an integer, unless the spelling
// carries a '.' or a binary 'p' exponent - then it is a float whose value is the
// hexadecimal mantissa times two to the exponent.
func luParseHex(s string) (interface{}, bool) {
	i, n := 0, len(s)
	start := i
	f := 0.0
	digits := 0
	isFloat := false
	for i < n && luDigit(s[i]) >= 0 {
		f = f*16 + float64(luDigit(s[i]))
		i++
		digits++
	}
	intEnd := i
	if i < n && s[i] == '.' {
		isFloat = true
		i++
		scale := 1.0
		for i < n && luDigit(s[i]) >= 0 {
			scale /= 16
			f += float64(luDigit(s[i])) * scale
			i++
			digits++
		}
	}
	if digits == 0 {
		return nil, false
	}
	if i < n && (s[i] == 'p' || s[i] == 'P') {
		isFloat = true
		i++
		esign := 1
		if i < n && (s[i] == '+' || s[i] == '-') {
			if s[i] == '-' {
				esign = -1
			}
			i++
		}
		e, edig := 0, 0
		for i < n && s[i] >= '0' && s[i] <= '9' {
			e = e*10 + int(s[i]-'0')
			i++
			edig++
		}
		if edig == 0 {
			return nil, false
		}
		f *= math.Pow(2, float64(esign*e))
	}
	if i < n {
		return nil, false
	}
	if isFloat {
		return luMkFlo(f), true
	}
	// A hexadecimal INTEGER wraps to 64 bits: 0x10000000000000000 is 0.
	iv, _ := luParseInt(s[start:intEnd], 16)
	return giNorm(iv, 64, false), true
}

// luParseDec reads a decimal numeral: a float exactly when it has a fraction dot
// or an exponent.
func luParseDec(s string) (interface{}, bool) {
	i, n := 0, len(s)
	isFloat := false
	digits := 0
	for i < n && s[i] >= '0' && s[i] <= '9' {
		i++
		digits++
	}
	if i < n && s[i] == '.' {
		isFloat = true
		i++
		for i < n && s[i] >= '0' && s[i] <= '9' {
			i++
			digits++
		}
	}
	if digits == 0 {
		return nil, false
	}
	if i < n && (s[i] == 'e' || s[i] == 'E') {
		isFloat = true
		i++
		if i < n && (s[i] == '+' || s[i] == '-') {
			i++
		}
		edig := 0
		for i < n && s[i] >= '0' && s[i] <= '9' {
			i++
			edig++
		}
		if edig == 0 {
			return nil, false
		}
	}
	if i < n {
		return nil, false
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil, false
	}
	if isFloat {
		return luMkFlo(f), true
	}
	// A decimal integer literal is EXACT, and one that does not fit a signed 64
	// bit integer is a FLOAT rather than a wrapped integer.
	iv, fits := luParseInt(s, 10)
	if !fits {
		return luMkFlo(f), true
	}
	return giNorm(iv, 64, false), true
}

// luToNum is Lua's automatic coercion in an arithmetic or bitwise context: a
// number passes, a string that spells a numeral converts, anything else fails.
func luToNum(v interface{}) (interface{}, bool) {
	if luIsNum(v) {
		return v, true
	}
	if s, ok := v.(string); ok {
		return luParseNum(s)
	}
	return nil, false
}

// ----- the operators -----

// luTypeName is Lua's type() for an error message.
func (rt *jsrt) luTypeName(v interface{}) string {
	if v == nil || v == jsUndef {
		return "nil"
	}
	if _, ok := v.(bool); ok {
		return "boolean"
	}
	if luIsNum(v) {
		return "number"
	}
	if _, ok := v.(string); ok {
		return "string"
	}
	if _, ok := v.(*jsClosure); ok {
		return "function"
	}
	return "table"
}

// luIDiv / luIMod are integer FLOOR division and FLOOR modulo. Go's / and %
// truncate toward zero; Lua rounds toward minus infinity, so a non-zero
// remainder whose sign differs from the divisor's moves the quotient one step
// down and the remainder one divisor up. Both wrap at 64 bits, which is what
// makes math.mininteger // -1 answer math.mininteger.
func luIDiv(a, b int64) int64 {
	if a == math.MinInt64 && b == -1 {
		return a // the hardware wrap; Go's / panics on exactly this pair
	}
	q := a / b
	if r := a % b; r != 0 && (r < 0) != (b < 0) {
		q--
	}
	return q
}

func luIMod(a, b int64) int64 {
	if a == math.MinInt64 && b == -1 {
		return 0
	}
	r := a % b
	if r != 0 && (r < 0) != (b < 0) {
		r += b
	}
	return r
}

// luArith is one of + - * / // %, by index (0..5). '/' is ALWAYS a float in Lua;
// the others keep the integer subtype exactly when both operands are integers,
// and the integer path is exact and wrapping at 64 bits.
func (rt *jsrt) luArith(op int, lv, rv interface{}) interface{} {
	l, ok := luToNum(lv)
	if !ok {
		rt.fail("attempt to perform arithmetic on a %s value", rt.luTypeName(lv))
	}
	r, ok := luToNum(rv)
	if !ok {
		rt.fail("attempt to perform arithmetic on a %s value", rt.luTypeName(rv))
	}
	if op == 3 {
		return luMkFlo(luNumOf(l) / luNumOf(r))
	}
	if luIsInt(l) && luIsInt(r) {
		a, b := luInt64(l), luInt64(r)
		switch op {
		case 0:
			return giNorm(a+b, 64, false)
		case 1:
			return giNorm(a-b, 64, false)
		case 2:
			return giNorm(a*b, 64, false)
		}
		if b == 0 {
			// The two messages are lua 5.5's own, verified against the
			// installed one.
			if op == 4 {
				rt.fail("attempt to divide by zero")
			}
			rt.fail("attempt to perform 'n%%0'")
		}
		if op == 4 {
			return giNorm(luIDiv(a, b), 64, false)
		}
		return giNorm(luIMod(a, b), 64, false)
	}
	x, y := luNumOf(l), luNumOf(r)
	switch op {
	case 0:
		return luMkFlo(x + y)
	case 1:
		return luMkFlo(x - y)
	case 2:
		return luMkFlo(x * y)
	case 4:
		return luMkFlo(math.Floor(x / y))
	}
	// Lua's float modulo follows the sign of the DIVISOR (-0.5 % 2 is 1.5).
	//
	// It is luai_nummod - fmod, then one correction - and NOT x - Floor(x/y)*y.
	// The two agree only while x/y fits 53 bits: real lua 5.5 (the oracle here)
	// answers 10 % 3.14159265358979 with 0.57522203923063, and the floor form
	// gives 0.5752220392306295. The floor form was also ARCHITECTURE DEPENDENT:
	// on arm64 Go contracts x - Floor(x/y)*y into a fused multiply-subtract,
	// which goja (lua-interpreter.abnf) and the C floor cannot do, so the two
	// halves of this language answered differently in 128 of a 12,557-line
	// differential probe - invisible to --cross, which printed none of them.
	m := math.Mod(x, y)
	if m != 0 && (m < 0) != (y < 0) {
		m += y
	}
	return luMkFlo(m)
}

// luToIntOp is a bitwise operand: an integer, a float whose value is an exact
// integer inside the 64 bit range, or a string that spells one. Anything else is
// an error, and Lua distinguishes the two reasons.
func (rt *jsrt) luToIntOp(v interface{}) int64 {
	a, ok := luToNum(v)
	if !ok {
		rt.fail("attempt to perform bitwise operation on a %s value", rt.luTypeName(v))
	}
	if luIsInt(a) {
		return luInt64(a)
	}
	n := luNumOf(a)
	if math.IsNaN(n) || math.IsInf(n, 0) || n != math.Trunc(n) {
		rt.fail("number has no integer representation")
	}
	iv, fits := luI64(n)
	if !fits {
		rt.fail("number has no integer representation")
	}
	return luInt64(iv)
}

// luBit is one of & | ~ << >>, by index (0..4), exact at 64 bits. The shifts are
// LOGICAL in both directions - Lua's >> never sign-extends - a count of 64 or
// more clears the value, and a negative count shifts the other way.
func (rt *jsrt) luBit(op int, lv, rv interface{}) interface{} {
	a := rt.luToIntOp(lv)
	if op == 3 || op == 4 {
		k := rt.luToIntOp(rv)
		left := op == 3
		if k < 0 {
			k = -k
			left = !left
		}
		if k >= 64 || k < 0 { // k < 0 only for math.mininteger, whose negation wraps
			return float64(0)
		}
		if left {
			return giNorm(int64(uint64(a)<<uint(k)), 64, false)
		}
		return giNorm(int64(uint64(a)>>uint(k)), 64, false)
	}
	b := rt.luToIntOp(rv)
	switch op {
	case 0:
		return giNorm(a&b, 64, false)
	case 1:
		return giNorm(a|b, 64, false)
	}
	return giNorm(a^b, 64, false) // Lua spells exclusive-or '~'
}

// luEq / luCmp. Two integers compare on the exact 64 bit value, because past
// 2^53 their double readings are equal for values that are not.
func (rt *jsrt) luEq(l, r interface{}) bool {
	if luIsInt(l) && luIsInt(r) {
		return luInt64(l) == luInt64(r)
	}
	if luIsNum(l) && luIsNum(r) {
		return luNumOf(l) == luNumOf(r)
	}
	return rt.strictEq(l, r)
}

func (rt *jsrt) luCmp(op int, l, r interface{}) bool {
	c := 0
	if luIsInt(l) && luIsInt(r) {
		a, b := luInt64(l), luInt64(r)
		if a < b {
			c = -1
		} else if a > b {
			c = 1
		}
	} else if luIsNum(l) && luIsNum(r) {
		a, b := luNumOf(l), luNumOf(r)
		if a < b {
			c = -1
		} else if a > b {
			c = 1
		} else if a != b {
			return false // a NaN operand is unordered
		}
	} else {
		c = rt.jsCompare(l, r)
	}
	switch op {
	case 0:
		return c < 0
	case 1:
		return c > 0
	case 2:
		return c <= 0
	}
	return c >= 0
}

// luNumStr is how a Lua number renders: an integer without a fraction, a float
// always with one, and inf / -inf / nan rather than JavaScript's spellings.
func luNumStr(v interface{}) string {
	if b, ok := v.(jsGInt); ok {
		return strconv.FormatInt(b.v, 10)
	}
	n := luNumOf(v)
	if math.IsNaN(n) {
		return "nan" // real Lua prints "nan", not "-nan"
	}
	if math.IsInf(n, 1) {
		return "inf"
	}
	if math.IsInf(n, -1) {
		return "-inf"
	}
	s := jsNumString(n)
	if _, isFlo := luFloBox(v); isFlo {
		return luFloText(s, n)
	}
	return luExpFix(s)
}

// luFloText finishes an INTEGRAL float's rendering. Lua's own rule (lobject.c
// tostringbuff) is `if (buff[strspn(buff, "-0123456789")] == '\0')` - append ".0"
// only when the text is nothing but a sign and digits - and the box this is called
// for wraps exactly the integral floats, so the text is either all digits or in
// exponent form. Appending ".0" unconditionally produced "1e+21.0" and
// "1.7976931348623157e+308.0", which is not a number in any notation. In exponent
// form Lua goes through C's %g instead, whose exponent is always signed and at
// least two digits, so "1e-7" is written "1e-07"; and -0.0 keeps its sign, which
// jsNumString drops because JavaScript's String(-0) is "0". Verified against
// lua 5.5.0.
//
// STILL WRONG, and recorded rather than guessed: WHICH form Lua picks. Lua tries
// "%.15g", then "%.16g", then "%.17g", and takes the first that round-trips - so
// 1e15 is "1e+15" while 1e15+1 is "1000000000000001.0" and 1e14 is
// "100000000000000.0". Ours uses the shortest round-tripping form throughout, which
// is JavaScript's window (plain below 1e21) and gets the large integral floats
// plain where Lua writes them scientific. Fixing that needs rounding the EXACT
// value of the double to 15 significant digits, which Go has as
// strconv.FormatFloat(n, 'g', 15, 64) and neither JS-dialect half has at all (no
// toPrecision anywhere in metajs-to-llvm-ir.abnf), so doing it here alone would
// CREATE a --cross divergence. See docs/runtime-next-plan.md, "WHAT IS STILL OPEN"
// item 1.
func luFloText(s string, n float64) string {
	if n == 0 && math.Signbit(n) {
		s = "-0"
	}
	if !strings.ContainsRune(s, 'e') {
		return s + ".0"
	}
	return luExpFix(s)
}

// luExpFix rewrites JavaScript's exponent form into C's %g one - always signed, at
// least two digits - and leaves anything without an exponent alone. It is applied to
// the UNBOXED path too, because the box wraps only the INTEGRAL floats: a
// non-integral float like 1e-7 never reaches luFloText and still has to print
// "1e-07" rather than "1e-7". An integer's text is digits only, so this is the
// identity on it.
func luExpFix(s string) string {
	i := strings.IndexByte(s, 'e')
	if i < 0 {
		return s
	}
	mant, exp := s[:i], s[i+1:]
	sign := "+"
	if len(exp) > 0 && (exp[0] == '+' || exp[0] == '-') {
		sign = exp[:1]
		exp = exp[1:]
	}
	if len(exp) < 2 {
		exp = "0" + exp
	}
	return mant + "e" + sign + exp
}

// luKey normalises a table key. Numbers and strings address the map the way the
// runtime's own js_get does; a float with an exact integer value normalises to
// that INTEGER key, which is Lua's own rule (t[2.0] is t[2]). An integer past
// 2^53 has to become its exact decimal TEXT, because its double reading would
// collide with its neighbour's.
func luKey(v interface{}) interface{} {
	if b, ok := v.(jsGInt); ok {
		return strconv.FormatInt(b.v, 10)
	}
	if f, ok := luFloBox(v); ok {
		if !math.IsInf(f, 0) && !math.IsNaN(f) && f == math.Trunc(f) {
			if iv, fits := luI64(f); fits {
				if b, isBox := iv.(jsGInt); isBox {
					return strconv.FormatInt(b.v, 10)
				}
				return iv
			}
		}
		return f
	}
	return v
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

		m["js_luisint"] = func(a []uint64) uint64 { return boolH(luIsInt(u(a[0]))) }
		m["js_luisnum"] = func(a []uint64) uint64 { return boolH(luIsNum(u(a[0]))) }
		// The raw double behind a Lua value, with Lua's string coercion. Every
		// consumer that genuinely wants a double (an index, a length, a repeat
		// count, a float operand) goes through it; a value that is no number at
		// all reads as NaN, which is what the callers already expected.
		m["js_lunum"] = func(a []uint64) uint64 {
			v, ok := luToNum(u(a[0]))
			if !ok {
				return rt.wrapNum(math.NaN())
			}
			return rt.wrapNum(luNumOf(v))
		}
		// The COERCION itself, keeping the subtype: '10' + 5 is the integer 15
		// and '10.0' + 5 the float 15.0.
		m["js_lutonum"] = func(a []uint64) uint64 {
			v, ok := luToNum(u(a[0]))
			if !ok {
				return w(jsUndef)
			}
			return w(v)
		}
		m["js_lumkflo"] = func(a []uint64) uint64 { return w(luMkFlo(rt.toNumber(u(a[0])))) }
		// The operator INDEX arrives as a raw handle (the emitter writes
		// handle(k), a compile time constant, exactly as the IR helpers these
		// replaced compared it with NewICmp) - it is not a wrapped value and
		// must not be unwrapped.
		m["js_luarith"] = func(a []uint64) uint64 {
			return w(rt.luArith(int(a[0]), u(a[1]), u(a[2])))
		}
		m["js_lubit"] = func(a []uint64) uint64 {
			return w(rt.luBit(int(a[0]), u(a[1]), u(a[2])))
		}
		m["js_lubnot"] = func(a []uint64) uint64 {
			return w(giNorm(^rt.luToIntOp(u(a[0])), 64, false))
		}
		m["js_lueq"] = func(a []uint64) uint64 { return boolH(rt.luEq(u(a[0]), u(a[1]))) }
		m["js_lucmp"] = func(a []uint64) uint64 {
			return boolH(rt.luCmp(int(a[0]), u(a[1]), u(a[2])))
		}
		m["js_lustr"] = func(a []uint64) uint64 { return rt.wrapStr(luNumStr(u(a[0]))) }
		m["js_lukey"] = func(a []uint64) uint64 { return w(luKey(u(a[0]))) }
		// A literal the emitter could not put in the module as a double, because
		// emitNum would already have rounded 9223372036854775807 to
		// 9223372036854776000 on the way in. (digit text, radix)
		m["js_lulit"] = func(a []uint64) uint64 {
			s := rt.toString(u(a[0]))
			radix := int(rt.toNumber(u(a[1])))
			if radix == 16 {
				iv, _ := luParseInt(s, 16)
				return w(giNorm(iv, 64, false))
			}
			iv, fits := luParseInt(s, 10)
			if !fits {
				f, _ := strconv.ParseFloat(s, 64)
				return w(luMkFlo(f))
			}
			return w(giNorm(iv, 64, false))
		}
		// math.type(x): "integer", "float", or nil for anything that is not a
		// number (a STRING is not, even one that spells a numeral).
		m["js_lutype"] = func(a []uint64) uint64 {
			v := u(a[0])
			if luIsInt(v) {
				return rt.wrapStr("integer")
			}
			if luIsNum(v) {
				return rt.wrapStr("float")
			}
			return w(jsUndef)
		}
		// math.tointeger(x): the integer x names, or nil when it names none.
		m["js_lutoint"] = func(a []uint64) uint64 {
			v, ok := luToNum(u(a[0]))
			if !ok {
				return w(jsUndef)
			}
			if luIsInt(v) {
				return w(v)
			}
			n := luNumOf(v)
			if math.IsNaN(n) || math.IsInf(n, 0) || n != math.Trunc(n) {
				return w(jsUndef)
			}
			iv, fits := luI64(n)
			if !fits {
				return w(jsUndef)
			}
			return w(iv)
		}
		// math.floor / math.ceil answer an INTEGER, and an operand that already
		// is one is handed back untouched - flooring math.maxinteger through a
		// double would round it.
		mathRound := func(up bool) func(a []uint64) uint64 {
			return func(a []uint64) uint64 {
				v := u(a[0])
				if luIsInt(v) {
					return w(v)
				}
				n, ok := luToNum(v)
				if !ok {
					rt.fail("bad argument to 'floor' (number expected)")
				}
				if luIsInt(n) {
					return w(n)
				}
				f := luNumOf(n)
				if up {
					f = math.Ceil(f)
				} else {
					f = math.Floor(f)
				}
				if iv, fits := luI64(f); fits {
					return w(iv)
				}
				return rt.wrapNum(f)
			}
		}
		m["js_lufloor"] = mathRound(false)
		m["js_luceil"] = mathRound(true)
		// math.abs keeps the subtype, and an integer WRAPS:
		// math.abs(math.mininteger) is math.mininteger.
		m["js_luabs"] = func(a []uint64) uint64 {
			v := u(a[0])
			if luIsInt(v) {
				n := luInt64(v)
				if n < 0 {
					n = -n
				}
				return w(giNorm(n, 64, false))
			}
			return w(luMkFlo(math.Abs(luNumOf(v))))
		}
		// math.max / math.min keep the WINNING argument, so the subtype survives -
		// and the tie goes to the FIRST one, which is real Lua's rule (its loop
		// replaces the running best only on a strict '<'), so math.max(1, 1.0) is
		// the integer 1.
		m["js_lumax"] = func(a []uint64) uint64 {
			if rt.luCmp(0, u(a[0]), u(a[1])) {
				return a[1]
			}
			return a[0]
		}
		m["js_lumin"] = func(a []uint64) uint64 {
			if rt.luCmp(0, u(a[1]), u(a[0])) {
				return a[1]
			}
			return a[0]
		}
		// math.fmod keeps C's fmod: the remainder takes the sign of the DIVIDEND,
		// which is where it differs from the '%' operator.
		m["js_lufmod"] = func(a []uint64) uint64 {
			l, lok := luToNum(u(a[0]))
			r, rok := luToNum(u(a[1]))
			if !lok || !rok {
				rt.fail("bad argument to 'fmod' (number expected)")
			}
			if luIsInt(l) && luIsInt(r) {
				b := luInt64(r)
				if b == 0 {
					rt.fail("bad argument #2 to 'fmod' (zero)")
				}
				if b == -1 {
					return w(float64(0))
				}
				return w(giNorm(luInt64(l)%b, 64, false))
			}
			return w(luMkFlo(math.Mod(luNumOf(l), luNumOf(r))))
		}
		m["js_lumaxint"] = func(a []uint64) uint64 { return w(giNorm(math.MaxInt64, 64, false)) }
		m["js_luminint"] = func(a []uint64) uint64 { return w(giNorm(math.MinInt64, 64, false)) }
	})
}
