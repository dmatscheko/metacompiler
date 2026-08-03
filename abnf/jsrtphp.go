package abnf

// PHP's value model for the compiler grammar (languages/php-to-llvm-ir.abnf).
//
// php-interpreter.abnf is the reference implementation: it models a PHP value as
//
//	null / bool / int (a plain number) / float (a BOX, so 3.0 !== 3 stays true) /
//	string (byte based) / array (ONE ordered key->value map, so === compares the
//	keys AND their order while == does not) / object
//
// The compiler used to lower PHP onto the plain JS value semantics of the handle
// runtime, which made three answers silently WRONG rather than merely missing:
// `3.0 === 3` was true (no boxed float), `"0"` was truthy (JS truthiness) and
// `==` was strict (binExt["=="] was js_seq). The js_ph* externs below give the
// compiler the same model the interpreter has, so the two agree again.
//
// This file is append-only with respect to the rest of the runtime: nothing here
// runs unless a program calls one of the js_ph* externs, which only the PHP
// compiler grammar emits. The one exception is the jsPhpFlo box, which
// rt.truthy / rt.toNumber / rt.toString / rt.toGoNatural learn about in jsrt.go -
// each as one extra case for a type nothing else creates.

import (
	"math"
	"strconv"
	"strings"
)

// jsPhpFlo is PHP's float, boxed so that it stays distinguishable from an int.
// Ruby's jsFlo is the same idea but renders differently (Ruby keeps the decimal
// point, PHP prints 3.0 as "3"), so the two boxes stay separate types.
type jsPhpFlo struct{ f float64 }

func phpMkFlo(f float64) jsPhpFlo { return jsPhpFlo{f: f} }

func phpIsFlo(v interface{}) bool { _, ok := v.(jsPhpFlo); return ok }

// ----- PHP's int is 64 bit two's complement ----------------------------------
//
// The value model matches php-interpreter.abnf's exactly: an int is a PLAIN
// float64 while it stays inside [-2^53, 2^53] and the shared sized-integer box
// jsGInt{w: 64, u: false} of jsrtint.go otherwise, so giNorm / giStr / giFromFloat
// are reused unchanged. What is PHP's OWN is the overflow rule: PHP does NOT
// wrap, it silently promotes to a float computed in double precision, so
// PHP_INT_MAX + 1 is the float 9.2233720368548E+18. The arithmetic below is
// therefore not giArith - it is the overflow-detecting phpIntArith.
//
// Verified against tests/reference/php/php-src-tests/lang/operators/
// *_basiclong_64bit.phpt, whose --EXPECT-- blocks this file reproduces byte for
// byte through the same test runner the interpreter half passes.

const phpIntMax = int64(9223372036854775807)
const phpIntMin = int64(-9223372036854775808)

// phpIsInt: every plain float64 in this model is an integer (a PHP float is a
// jsPhpFlo box), and a jsGInt is one that left the exact range.
func phpIsInt(v interface{}) bool {
	switch t := v.(type) {
	case float64:
		return true
	case jsGInt:
		return t.w == 64 && !t.u
	}
	return false
}

// phpMkInt applies the invariant: a plain number while the value is exact.
func phpMkInt(v int64) interface{} { return giNorm(v, 64, false) }

// phpLongOf reads an INT operand's 64 bit value (the caller has already
// established that it is one).
func phpLongOf(v interface{}) int64 {
	switch t := v.(type) {
	case jsGInt:
		return t.v
	case float64:
		return int64(t)
	}
	return 0
}

// phpFloToLong is the (int) reading of a float: truncation toward zero, then a
// WRAP modulo 2^64 rather than a saturation - (int)(PHP_INT_MAX * 2 + 4) is
// int(0), which tests/int_overflow_64bit.phpt records.
func phpFloToLong(f float64) int64 {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0
	}
	return giFromFloat(f)
}

func phpIsNum(v interface{}) bool {
	switch v.(type) {
	case float64, jsGInt, jsPhpFlo:
		return true
	}
	return false
}

func phpNumOf(v interface{}) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case jsGInt:
		return float64(t.v)
	case jsPhpFlo:
		return t.f
	}
	return 0
}

// phpTrunc rounds toward zero: PHP's (int) cast and intdiv().
func phpTrunc(n float64) float64 {
	if n != n || math.IsInf(n, 0) {
		return 0
	}
	return math.Trunc(n)
}

// ----- float rendering: PHP has TWO, and they differ -------------------------
//
// echo / (string) / interpolation render with precision=14; var_dump, var_export
// and json use serialize_precision=-1, which means the SHORTEST round-tripping
// digits laid out at ndigit=17. The same value is 9.2233720368548E+18 to echo
// and 9.223372036854776E+18 to var_dump, and confusing the two is the classic
// PHP float trap. Both are zend_gcvt(value, ndigit, '.', 'E'): exponential
// notation exactly when `decpt < 0 ? decpt < -3 : decpt > ndigit`, a mantissa
// that always keeps one digit after the point ("1.0E+25"), and an exponent that
// is never zero padded. This is the byte-for-byte twin of floStr / floDump in
// php-interpreter.abnf.
func phpFloStr(n float64) string  { return phpFloFormat(n, 14) }
func phpFloDump(n float64) string { return phpFloFormat(n, 17) }

// phpFloSplit answers the shortest decimal digits of a positive finite double
// and the position of the decimal point: value == 0.<digits> * 10^<decpt>.
func phpFloSplit(a float64) (string, int) {
	s := strconv.FormatFloat(a, 'e', -1, 64) // "d.dddde±dd"
	mant, exps := s, "0"
	if i := strings.IndexByte(s, 'e'); i >= 0 {
		mant, exps = s[:i], s[i+1:]
	}
	exp, _ := strconv.Atoi(exps)
	digs := strings.Replace(mant, ".", "", 1)
	decpt := exp + 1
	for len(digs) > 0 && digs[0] == '0' {
		digs = digs[1:]
		decpt--
	}
	for len(digs) > 0 && digs[len(digs)-1] == '0' {
		digs = digs[:len(digs)-1]
	}
	if digs == "" {
		return "0", 1
	}
	return digs, decpt
}

// phpFloRound rounds a digit string to nd significant digits, half up, and drops
// the trailing zeros dtoa's mode 2 drops.
func phpFloRound(d string, decpt, nd int) (string, int) {
	if len(d) > nd {
		keep := []byte(d[:nd])
		if d[nd] >= '5' {
			i := nd - 1
			for i >= 0 {
				if keep[i] == '9' {
					keep[i] = '0'
					i--
					continue
				}
				keep[i]++
				break
			}
			if i < 0 {
				keep = append([]byte{'1'}, keep...)[:nd]
				decpt++
			}
		}
		d = string(keep)
	}
	for len(d) > 1 && d[len(d)-1] == '0' {
		d = d[:len(d)-1]
	}
	if d == "0" {
		decpt = 1
	}
	return d, decpt
}

func phpFloFormat(n float64, nd int) string {
	if math.IsNaN(n) {
		return "NAN"
	}
	if math.IsInf(n, 1) {
		return "INF"
	}
	if math.IsInf(n, -1) {
		return "-INF"
	}
	sign, a := "", n
	if n < 0 || (n == 0 && math.Signbit(n)) {
		sign, a = "-", -n
	}
	if a == 0 {
		return sign + "0"
	}
	d, decpt := phpFloSplit(a)
	d, decpt = phpFloRound(d, decpt, nd)
	exponential := decpt > nd
	if decpt < 0 {
		exponential = decpt < -3
	}
	if exponential {
		e2 := decpt - 1
		frac := "0"
		if len(d) > 1 {
			frac = d[1:]
		}
		es, ea := "+", e2
		if e2 < 0 {
			es, ea = "-", -e2
		}
		return sign + d[:1] + "." + frac + "E" + es + strconv.Itoa(ea)
	}
	if decpt <= 0 {
		return sign + "0." + strings.Repeat("0", -decpt) + d
	}
	if decpt >= len(d) {
		return sign + d + strings.Repeat("0", decpt-len(d))
	}
	return sign + d[:decpt] + "." + d[decpt:]
}

// phpArrParts views a PHP array as (keys, vals). A PHP array is the ordered map
// (the __dict shape js_range_len / js_range_key / js_range_val already iterate),
// but a plain *jsArray can still reach here through a runtime-built list, and it
// reads as the map 0=>e0, 1=>e1, ... - which is exactly what it means in PHP.
func phpArrParts(v interface{}) (*jsArray, *jsArray, bool) {
	if keys, vals, ok := dictParts(v); ok {
		return keys, vals, true
	}
	if arr, ok := v.(*jsArray); ok {
		keys := &jsArray{elems: make([]interface{}, len(arr.elems))}
		for i := range arr.elems {
			keys.elems[i] = float64(i)
		}
		return keys, arr, true
	}
	return nil, nil, false
}

func phpIsArr(v interface{}) bool {
	_, _, ok := phpArrParts(v)
	return ok
}

// ----- references ------------------------------------------------------------
//
// A PHP reference makes two names share ONE storage cell, and php-interpreter.abnf
// models that with a cell object that a scope slot or an array element may hold in
// place of the value. The compiler uses the same model: a CELL is a plain object
// carrying __refcell = true and the value under "v", built by the emitted code
// (js_obj_new + js_set), so no new external is needed to make one.
//
// Everything below reads THROUGH a cell, which is what makes `$b = &$a`,
// `foreach (... as &$v)`, `use (&$v)` and a by-reference parameter one mechanism.
// A program that never writes a '&' creates no cell at all, so every phpDeref here
// is a type assertion that fails immediately and the answers are the ones from
// before references existed.
func phpCellOf(v interface{}) *jsObject {
	o, ok := v.(*jsObject)
	if !ok {
		return nil
	}
	if t, has := o.props["__refcell"]; has && t == true {
		return o
	}
	return nil
}

func phpDeref(v interface{}) interface{} {
	if c := phpCellOf(v); c != nil {
		return c.props["v"]
	}
	return v
}

func phpIsObj(v interface{}) bool {
	o, ok := v.(*jsObject)
	if !ok {
		return false
	}
	if _, isCell := o.props["__refcell"]; isCell {
		return false
	}
	if tag, isDict := o.props["__dict"]; isDict && tag == true {
		return false
	}
	_, isClass := o.props["__isclass"]
	return !isClass
}

// ----- truthiness ------------------------------------------------------------
// false, 0, 0.0, "", "0", [] and null are falsy; everything else is truthy.

func (rt *jsrt) phpTest(v interface{}) bool {
	switch t := v.(type) {
	case jsUndefT, jsNullT:
		return false
	case bool:
		return t
	case float64:
		return t != 0
	case jsGInt: // An int box is outside the exact range, so it is never 0.
		return true
	case jsPhpFlo:
		return t.f != 0
	case string:
		return t != "" && t != "0"
	}
	if keys, _, ok := phpArrParts(v); ok {
		return len(keys.elems) > 0
	}
	return true
}

// ----- string coercion -------------------------------------------------------

func (rt *jsrt) phpStr(v interface{}) string {
	switch t := v.(type) {
	case jsUndefT, jsNullT:
		return ""
	case bool:
		if t {
			return "1"
		}
		return ""
	case string:
		return t
	case float64:
		return jsNumString(t)
	case jsGInt:
		return giStr(t)
	case jsPhpFlo:
		return phpFloStr(t.f)
	}
	if phpIsArr(v) {
		return "Array"
	}
	if phpIsObj(v) {
		// __toString, when the class declares one (php-interpreter.abnf's
		// objToString); otherwise PHP raises, and the compiler prints the
		// same placeholder the JS runtime would.
		if o, ok := v.(*jsObject); ok {
			if cls, has := o.props["__class"]; has {
				if m := phpClassFind(cls, "__toString"); m != nil {
					return rt.phpStr(rt.call(m, jsUndef, []interface{}{v}))
				}
			}
		}
		return "Object"
	}
	return rt.toString(v)
}

// phpClassFind walks the __super chain of a class descriptor for one member.
func phpClassFind(cls interface{}, name string) interface{} {
	guard := 0
	for guard < 64 {
		guard++
		o, ok := cls.(*jsObject)
		if !ok {
			return nil
		}
		if m, has := o.props[name]; has && isCallable(m) {
			return m
		}
		sup, has := o.props["__super"]
		if !has {
			return nil
		}
		cls = sup
	}
	return nil
}

// ----- numeric strings and arithmetic coercion -------------------------------

// phpNumStrVal answers the number a fully numeric string denotes (an int as a
// float64, a float as a jsPhpFlo), or nil when the string is not numeric.
// Leading and trailing whitespace is allowed (PHP 8).
func phpNumStrVal(s string) interface{} {
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r' || s[i] == '\v' || s[i] == '\f') {
		i++
	}
	st := i
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		i++
	}
	digits := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
		digits++
	}
	isF := false
	if i < len(s) && s[i] == '.' {
		isF = true
		i++
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
			digits++
		}
	}
	if digits == 0 {
		return nil
	}
	if i < len(s) && (s[i] == 'e' || s[i] == 'E') {
		save := i
		i++
		if i < len(s) && (s[i] == '+' || s[i] == '-') {
			i++
		}
		ed := 0
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
			ed++
		}
		if ed == 0 {
			i = save
		} else {
			isF = true
		}
	}
	en := i
	for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r' || s[i] == '\v' || s[i] == '\f') {
		i++
	}
	if i != len(s) {
		return nil
	}
	if isF {
		f, err := strconv.ParseFloat(s[st:en], 64)
		if err != nil {
			return nil
		}
		return phpMkFlo(f)
	}
	return phpIntStrVal(s[st:en])
}

// phpIntStrVal is the value a signed DIGIT RUN denotes, promoted to a float when
// it does not fit in 64 bits: is_numeric_string reports IS_DOUBLE for
// "9223372036854775808", so "9223372036854775808" + 0 is a float and not a
// saturated int.
func phpIntStrVal(txt string) interface{} {
	body, neg := txt, false
	if strings.HasPrefix(body, "+") {
		body = body[1:]
	} else if strings.HasPrefix(body, "-") {
		neg, body = true, body[1:]
	}
	if v, err := strconv.ParseInt(txt, 10, 64); err == nil {
		return phpMkInt(v)
	}
	_ = neg
	_ = body
	f, err := strconv.ParseFloat(txt, 64)
	if err != nil {
		return nil
	}
	return phpMkFlo(f)
}

// phpLeadingNum: a non-numeric string still contributes its numeric PREFIX
// ("12abc" is 12 in an arithmetic context).
func phpLeadingNum(s string) interface{} {
	i := 0
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		i++
	}
	d := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
		d++
	}
	isF := false
	if i < len(s) && s[i] == '.' {
		j := i + 1
		fd := 0
		for j < len(s) && s[j] >= '0' && s[j] <= '9' {
			j++
			fd++
		}
		if fd > 0 || d > 0 {
			isF = true
			i = j
			d += fd
		}
	}
	if d == 0 {
		return float64(0)
	}
	if isF {
		f, err := strconv.ParseFloat(s[:i], 64)
		if err != nil {
			return float64(0)
		}
		return phpMkFlo(f)
	}
	if v := phpIntStrVal(s[:i]); v != nil {
		return v
	}
	return float64(0)
}

// phpToNum is the arithmetic coercion: everything becomes an int or a float box.
func (rt *jsrt) phpToNum(v interface{}) interface{} {
	switch t := v.(type) {
	case float64:
		return t
	case jsGInt:
		return t
	case jsPhpFlo:
		return t
	case bool:
		if t {
			return float64(1)
		}
		return float64(0)
	case jsUndefT, jsNullT:
		return float64(0)
	case string:
		if n := phpNumStrVal(t); n != nil {
			return n
		}
		return phpLeadingNum(t)
	}
	if phpIsArr(v) {
		rt.fail("unsupported operand type: array used as a number")
	}
	return float64(0)
}

// phpToLong is the (int) reading of ANY value.
func (rt *jsrt) phpToLong(v interface{}) int64 {
	n := rt.phpToNum(v)
	if f, ok := n.(jsPhpFlo); ok {
		return phpFloToLong(f.f)
	}
	return phpLongOf(n)
}

// phpArith: an int stays an int unless the exact 64 bit answer does not FIT, and
// any float operand makes the result a float.
func (rt *jsrt) phpArith(op string, l, r interface{}) interface{} {
	a := rt.phpToNum(l)
	b := rt.phpToNum(r)
	// '%' is the one arithmetic operator that is INTEGER only: PHP casts both
	// operands to int first, so 7.5 % 2 is 1 and not 1.5.
	if op == "%" {
		y := rt.phpToLong(b)
		if y == 0 {
			rt.phpRaise("DivisionByZeroError", "Modulo by zero")
		}
		if y == -1 { // MIN % -1 traps on the hardware; PHP answers 0.
			return float64(0)
		}
		return phpMkInt(rt.phpToLong(a) % y)
	}
	if phpIsFlo(a) || phpIsFlo(b) {
		return rt.phpFloArith(op, phpNumOf(a), phpNumOf(b))
	}
	return rt.phpIntArith(op, phpLongOf(a), phpLongOf(b))
}

func (rt *jsrt) phpFloArith(op string, x, y float64) interface{} {
	switch op {
	case "+":
		return phpMkFlo(x + y)
	case "-":
		return phpMkFlo(x - y)
	case "*":
		return phpMkFlo(x * y)
	case "/":
		if y == 0 {
			rt.phpRaise("DivisionByZeroError", "Division by zero")
		}
		return phpMkFlo(x / y)
	case "**":
		return phpMkFlo(math.Pow(x, y))
	}
	rt.fail("unknown arithmetic operator %s", op)
	return nil
}

// phpAddOv / phpSubOv / phpMulOv are the three operators that can leave 64 bits.
// On overflow the operation is evaluated AGAIN in double precision, exactly what
// ZEND_SIGNED_ADD_OVERFLOW's slow path does, so the answer is
// float(9.223372036854776E+18) and not a rounded 64 bit value.
func phpAddOv(x, y int64) (int64, bool) {
	r := x + y
	return r, (x >= 0) == (y >= 0) && (r >= 0) != (x >= 0)
}

func phpSubOv(x, y int64) (int64, bool) {
	r := x - y
	return r, (x >= 0) != (y >= 0) && (r >= 0) != (x >= 0)
}

func phpMulOv(x, y int64) (int64, bool) {
	if x == 0 || y == 0 {
		return 0, false
	}
	if (x == -1 && y == phpIntMin) || (y == -1 && x == phpIntMin) {
		return phpIntMin, true
	}
	r := x * y
	return r, r/x != y
}

func (rt *jsrt) phpIntArith(op string, x, y int64) interface{} {
	switch op {
	case "+":
		if r, ov := phpAddOv(x, y); !ov {
			return phpMkInt(r)
		}
		return phpMkFlo(float64(x) + float64(y))
	case "-":
		if r, ov := phpSubOv(x, y); !ov {
			return phpMkInt(r)
		}
		return phpMkFlo(float64(x) - float64(y))
	case "*":
		if r, ov := phpMulOv(x, y); !ov {
			return phpMkInt(r)
		}
		return phpMkFlo(float64(x) * float64(y))
	case "/":
		// Division stays an int only when it comes out EXACT: 6/3 is int(2) where
		// 7/2 is float(3.5). intdiv() is the truncating one.
		if y == 0 {
			rt.phpRaise("DivisionByZeroError", "Division by zero")
		}
		if y == -1 && x == phpIntMin {
			return phpMkFlo(-float64(phpIntMin))
		}
		if x%y == 0 {
			return phpMkInt(x / y)
		}
		return phpMkFlo(float64(x) / float64(y))
	case "**":
		if y < 0 {
			return phpMkFlo(math.Pow(float64(x), float64(y)))
		}
		// Squaring, with the same overflow test every multiplication gets: the
		// first product that leaves 64 bits makes the WHOLE result a float.
		acc, base, e := int64(1), x, y
		for e > 0 {
			if e&1 == 1 {
				r, ov := phpMulOv(acc, base)
				if ov {
					return phpMkFlo(math.Pow(float64(x), float64(y)))
				}
				acc = r
			}
			e >>= 1
			if e > 0 {
				r, ov := phpMulOv(base, base)
				if ov {
					return phpMkFlo(math.Pow(float64(x), float64(y)))
				}
				base = r
			}
		}
		return phpMkInt(acc)
	}
	rt.fail("unknown arithmetic operator %s", op)
	return nil
}

// The bitwise operators are exact 64 bit and cast their operands to int. Unlike
// the arithmetic ones they WRAP - PHP_INT_MAX << 1 is int(-2), not a float
// (lang/operators/bitwiseShiftLeft_basiclong_64bit.phpt) - and a negative shift
// count is an ArithmeticError rather than a shift the other way.
func (rt *jsrt) phpBit(op string, l, r interface{}) interface{} {
	x := rt.phpToLong(l)
	switch op {
	case "&":
		return phpMkInt(x & rt.phpToLong(r))
	case "|":
		return phpMkInt(x | rt.phpToLong(r))
	case "^":
		return phpMkInt(x ^ rt.phpToLong(r))
	case "<<", ">>":
		n := rt.phpToLong(r)
		if n < 0 {
			rt.phpRaise("ArithmeticError", "Bit shift by negative number")
		}
		if n >= 64 {
			if op == "<<" || x >= 0 {
				return float64(0)
			}
			return float64(-1)
		}
		if op == "<<" {
			return phpMkInt(x << uint(n))
		}
		return phpMkInt(x >> uint(n))
	}
	rt.fail("unknown bit operator %s", op)
	return nil
}

// phpNeg is unary minus. It keeps the operand's TYPE (so -0.0 stays -0), and the
// one int that cannot be negated, PHP_INT_MIN, promotes like any other overflow.
func (rt *jsrt) phpNeg(v interface{}) interface{} {
	a := rt.phpToNum(v)
	if f, ok := a.(jsPhpFlo); ok {
		return phpMkFlo(-f.f)
	}
	x := phpLongOf(a)
	if x == phpIntMin {
		return phpMkFlo(-float64(phpIntMin))
	}
	return phpMkInt(-x)
}

// ----- the engine's own arithmetic errors ------------------------------------
//
// PHP made division by zero and a negative shift count CATCHABLE in 8.0, and the
// corpus tests exactly that (divide_basiclong_64bit.phpt catches
// DivisionByZeroError). The class descriptors live in the emitted module, not
// here, so phpmain hands its scope over once through js_phrtinit and the raise
// looks the class up there; without one (an imported fragment, a unit test) the
// error stays the fatal it was.
var phpMainScope = map[*jsrt]uint64{}

func (rt *jsrt) phpRaise(clsName, msg string) {
	h, has := phpMainScope[rt]
	if !has {
		rt.fail("%s", msg)
	}
	// TWO shapes, because layer 2 cannot open a scope (WALL 3 of
	// docs/runtime-next-plan.md part 3). php-to-llvm-ir.abnf now hands over a PHP
	// ARRAY of name => descriptor, which is a plain value the MetaJS runtime
	// library can read too; a module built before that change still hands over the
	// SCOPE, and is still accepted.
	if keys, vals, ok := phpArrParts(rt.unwrap(h)); ok {
		var found interface{}
		for i := range keys.elems {
			if rt.phpStr(keys.elems[i]) == clsName {
				found = phpDeref(vals.elems[i])
			}
		}
		if found == nil || isNullish(found) {
			rt.fail("%s", msg)
		}
		o := newJSObject()
		o.set("__class", found)
		o.set("message", msg)
		o.set("code", float64(0))
		panic(&jsThrown{value: o})
	}
	cls := rt.scopeGet(rt.scopeOf(h), clsName)
	if cls == nil || isNullish(cls) {
		rt.fail("%s", msg)
	}
	o := newJSObject()
	o.set("__class", cls)
	o.set("message", msg)
	o.set("code", float64(0))
	panic(&jsThrown{value: o})
}

// ----- comparison ------------------------------------------------------------

// phpIdentical is ===: the same type, and for arrays the same keys in the same
// ORDER. A boxed float is only identical to another boxed float.
func (rt *jsrt) phpIdentical(l, r interface{}) bool {
	if phpIsFlo(l) || phpIsFlo(r) {
		return phpIsFlo(l) && phpIsFlo(r) && l.(jsPhpFlo).f == r.(jsPhpFlo).f
	}
	lk, lv, la := phpArrParts(l)
	rk, rv, ra := phpArrParts(r)
	if la || ra {
		if !(la && ra) {
			return false
		}
		if len(lk.elems) != len(rk.elems) {
			return false
		}
		for i := range lk.elems {
			if !phpKeyEq(lk.elems[i], rk.elems[i]) {
				return false
			}
			if !rt.phpIdentical(phpDeref(lv.elems[i]), phpDeref(rv.elems[i])) {
				return false
			}
		}
		return true
	}
	if isNullish(l) || isNullish(r) {
		return isNullish(l) && isNullish(r)
	}
	// Two ints compare by VALUE: above 2^53 one of them is a box, and two boxes
	// holding the same 64 bit value are different Go values only by accident.
	if phpIsInt(l) || phpIsInt(r) {
		return phpIsInt(l) && phpIsInt(r) && phpLongOf(l) == phpLongOf(r)
	}
	return rt.strictEq(l, r)
}

func isNullish(v interface{}) bool {
	switch v.(type) {
	case jsUndefT, jsNullT:
		return true
	}
	return false
}

// phpKeyEq compares two normalized array keys (an int or a string).
func phpKeyEq(a, b interface{}) bool {
	aIsN, bIsN := phpIsInt(a), phpIsInt(b)
	if aIsN || bIsN {
		return aIsN && bIsN && phpLongOf(a) == phpLongOf(b)
	}
	as, aIsS := a.(string)
	bs, bIsS := b.(string)
	if aIsS && bIsS {
		return as == bs
	}
	return a == b
}

// phpLoose is PHP 8's ==: a bool on either side compares truthiness, null
// compares as "empty", two strings compare numerically only when BOTH are
// numeric, and a number against a NON-numeric string compares as strings (the
// PHP 8 change that made 0 == "a" false).
func (rt *jsrt) phpLoose(l, r interface{}) bool {
	if isNullish(l) {
		l = jsNull
	}
	if isNullish(r) {
		r = jsNull
	}
	_, lb := l.(bool)
	_, rb := r.(bool)
	if lb || rb {
		return rt.phpTest(l) == rt.phpTest(r)
	}
	lIsNull, rIsNull := l == interface{}(jsNull), r == interface{}(jsNull)
	if lIsNull && rIsNull {
		return true
	}
	if lIsNull {
		return rt.phpLooseNull(r)
	}
	if rIsNull {
		return rt.phpLooseNull(l)
	}
	lk, lv, la := phpArrParts(l)
	rk, rv, ra := phpArrParts(r)
	if la || ra {
		if !(la && ra) {
			return false
		}
		if len(lk.elems) != len(rk.elems) {
			return false
		}
		for i := range lk.elems {
			j := phpKeyIndex(rk, lk.elems[i])
			if j < 0 {
				return false
			}
			if !rt.phpLoose(phpDeref(lv.elems[i]), phpDeref(rv.elems[j])) {
				return false
			}
		}
		return true
	}
	if phpIsObj(l) || phpIsObj(r) {
		if !(phpIsObj(l) && phpIsObj(r)) {
			return false
		}
		lo, ro := l.(*jsObject), r.(*jsObject)
		if lo == ro {
			return true
		}
		if lo.props["__class"] != ro.props["__class"] {
			return false
		}
		return rt.phpObjPropsEq(lo, ro)
	}
	ls, lIsS := l.(string)
	rs, rIsS := r.(string)
	if lIsS && rIsS {
		nl, nr := phpNumStrVal(ls), phpNumStrVal(rs)
		if nl != nil && nr != nil {
			return phpNumEq(nl, nr)
		}
		return ls == rs
	}
	if lIsS && phpIsNum(r) {
		return rt.phpStrNumLoose(ls, r)
	}
	if rIsS && phpIsNum(l) {
		return rt.phpStrNumLoose(rs, l)
	}
	return phpNumEq(rt.phpToNum(l), rt.phpToNum(r))
}

// phpNumEq / phpNumCmp compare two numbers the way PHP does: int against int at
// 64 bits (a double would collapse 2^63 and 2^63 - 1 onto the same value),
// anything else as doubles, which is the LONG-to-DOUBLE widening
// compare_function performs.
func phpNumEq(a, b interface{}) bool {
	if phpIsInt(a) && phpIsInt(b) {
		return phpLongOf(a) == phpLongOf(b)
	}
	return phpNumOf(a) == phpNumOf(b)
}

func phpNumCmp(a, b interface{}) float64 {
	if phpIsInt(a) && phpIsInt(b) {
		x, y := phpLongOf(a), phpLongOf(b)
		if x < y {
			return -1
		}
		if x > y {
			return 1
		}
		return 0
	}
	return phpCmpNum(phpNumOf(a), phpNumOf(b))
}

func (rt *jsrt) phpLooseNull(v interface{}) bool {
	if s, ok := v.(string); ok {
		return s == ""
	}
	if keys, _, ok := phpArrParts(v); ok {
		return len(keys.elems) == 0
	}
	if phpIsNum(v) {
		return phpNumOf(v) == 0
	}
	return false
}

func (rt *jsrt) phpStrNumLoose(s string, n interface{}) bool {
	if ns := phpNumStrVal(s); ns != nil {
		return phpNumEq(ns, n)
	}
	return s == rt.phpStr(n)
}

// phpObjPropsEq compares two instances property by property with ==. Instance
// properties live directly on the object here (js_set), so the class markers are
// skipped by name.
func (rt *jsrt) phpObjPropsEq(a, b *jsObject) bool {
	na, nb := 0, 0
	for _, k := range a.keys {
		if phpHiddenProp(k) {
			continue
		}
		na++
		bv, has := b.props[k]
		if !has {
			return false
		}
		if !rt.phpLoose(a.props[k], bv) {
			return false
		}
	}
	for _, k := range b.keys {
		if !phpHiddenProp(k) {
			nb++
		}
	}
	return na == nb
}

func phpHiddenProp(k string) bool {
	return len(k) > 1 && k[0] == '_' && k[1] == '_'
}

func phpCmpNum(a, b float64) float64 {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func phpCmpStr(a, b string) float64 {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

// phpCmp is the spaceship, and the engine behind < > <= >=.
func (rt *jsrt) phpCmp(l, r interface{}) float64 {
	if isNullish(l) {
		l = jsNull
	}
	if isNullish(r) {
		r = jsNull
	}
	_, lb := l.(bool)
	_, rb := r.(bool)
	if lb || rb || l == interface{}(jsNull) || r == interface{}(jsNull) {
		lt, rt2 := 0.0, 0.0
		if rt.phpTest(l) {
			lt = 1
		}
		if rt.phpTest(r) {
			rt2 = 1
		}
		return phpCmpNum(lt, rt2)
	}
	lk, lv, la := phpArrParts(l)
	rk, rv, ra := phpArrParts(r)
	if la && ra {
		if len(lk.elems) != len(rk.elems) {
			return phpCmpNum(float64(len(lk.elems)), float64(len(rk.elems)))
		}
		for i := range lk.elems {
			j := phpKeyIndex(rk, lk.elems[i])
			if j < 0 {
				return 1
			}
			if cv := rt.phpCmp(phpDeref(lv.elems[i]), phpDeref(rv.elems[j])); cv != 0 {
				return cv
			}
		}
		return 0
	}
	ls, lIsS := l.(string)
	rs, rIsS := r.(string)
	if lIsS && rIsS {
		nl, nr := phpNumStrVal(ls), phpNumStrVal(rs)
		if nl != nil && nr != nil {
			return phpNumCmp(nl, nr)
		}
		return phpCmpStr(ls, rs)
	}
	if lIsS && phpIsNum(r) {
		x := phpNumStrVal(ls)
		if x == nil {
			return phpCmpStr(ls, rt.phpStr(r))
		}
		return phpNumCmp(x, r)
	}
	if rIsS && phpIsNum(l) {
		y := phpNumStrVal(rs)
		if y == nil {
			return phpCmpStr(rt.phpStr(l), rs)
		}
		return phpNumCmp(l, y)
	}
	return phpNumCmp(rt.phpToNum(l), rt.phpToNum(r))
}

// ----- arrays: ONE ordered map, with PHP's key normalization -----------------

// phpNormKey turns a subscript into PHP's stored key: an integer stays an
// integer, a canonical decimal string folds to one ("01" does NOT), a bool is
// 0/1, null is "" and a float truncates.
func phpNormKey(v interface{}) interface{} {
	switch t := v.(type) {
	case float64:
		return t
	case jsGInt:
		return v
	case jsPhpFlo:
		return phpMkInt(phpFloToLong(t.f))
	case bool:
		if t {
			return float64(1)
		}
		return float64(0)
	case jsUndefT, jsNullT:
		return ""
	case string:
		// A canonical decimal folds to an int key only while it FITS: PHP keeps
		// "9223372036854775808" a STRING key.
		if phpIsCanonInt(t) {
			if n, err := strconv.ParseInt(t, 10, 64); err == nil {
				return phpMkInt(n)
			}
		}
		return t
	}
	return v
}

// phpNumericKey answers whether a subscript addresses a string OFFSET: PHP indexes
// a string by an integer (or an integer-ish string) only.
func phpNumericKey(k interface{}) bool {
	switch t := k.(type) {
	case float64, jsPhpFlo, bool:
		return true
	case string:
		return phpNumStrVal(t) != nil
	}
	return false
}

func phpIsCanonInt(s string) bool {
	if s == "" {
		return false
	}
	i := 0
	if s[0] == '-' {
		if len(s) == 1 {
			return false
		}
		i = 1
	}
	if s[i] == '0' && len(s) > i+1 {
		return false
	}
	for j := i; j < len(s); j++ {
		if s[j] < '0' || s[j] > '9' {
			return false
		}
	}
	return true
}

// phpKeyIndex is a linear lookup by normalized key. It stays linear on purpose:
// the shared dictFind index keys on the raw Go value, and PHP's 5 / "5" folding
// would make a hit there answer for the wrong key.
func phpKeyIndex(keys *jsArray, k interface{}) int {
	nk := phpNormKey(k)
	for i := range keys.elems {
		if phpKeyEq(keys.elems[i], nk) {
			return i
		}
	}
	return -1
}

// phpNewArr builds an empty PHP array: the ordered map shape the runtime's own
// dict helpers and js_range_* already understand, plus __next, the next
// automatic integer key.
func phpNewArr() *jsObject {
	o := newJSObject()
	o.set("__dict", true)
	o.set("keys", &jsArray{})
	o.set("vals", &jsArray{})
	o.set("__next", float64(0))
	return o
}

// phpAsArr views any value as a PHP array for a WRITE. A plain *jsArray reaching
// a write site means the program built a list through a non-PHP path; it is
// promoted to the ordered map in place of the caller's value.
func phpAsArr(v interface{}) (*jsObject, bool) {
	if o, ok := v.(*jsObject); ok {
		if tag, isDict := o.props["__dict"]; isDict && tag == true {
			return o, true
		}
	}
	return nil, false
}

func phpArrNext(o *jsObject) float64 {
	if n, ok := o.props["__next"].(float64); ok {
		return n
	}
	return 0
}

func (rt *jsrt) phpArrSet(av interface{}, k, v interface{}) {
	o, ok := phpAsArr(av)
	if !ok {
		if arr, isArr := av.(*jsArray); isArr { // a raw list still indexes by position
			i := int(rt.phpToLong(k))
			for len(arr.elems) <= i {
				arr.elems = append(arr.elems, jsNull)
			}
			arr.dropIdx()
			arr.elems[i] = v
			return
		}
		rt.fail("cannot use a %s as an array", rt.typeOf(av))
	}
	keys, _ := o.props["keys"].(*jsArray)
	vals, _ := o.props["vals"].(*jsArray)
	nk := phpNormKey(k)
	if i := phpKeyIndex(keys, nk); i >= 0 {
		if c := phpCellOf(vals.elems[i]); c != nil && phpCellOf(v) == nil {
			c.set("v", v)
		} else {
			vals.elems[i] = v
		}
	} else {
		keys.dropIdx()
		vals.dropIdx()
		keys.elems = append(keys.elems, nk)
		vals.elems = append(vals.elems, v)
	}
	if f, isN := nk.(float64); isN && f >= phpArrNext(o) {
		o.set("__next", f+1)
	}
}

func (rt *jsrt) phpArrPush(av interface{}, v interface{}) {
	if o, ok := phpAsArr(av); ok {
		rt.phpArrSet(av, phpArrNext(o), v)
		return
	}
	if arr, isArr := av.(*jsArray); isArr {
		arr.dropIdx()
		arr.elems = append(arr.elems, v)
		return
	}
	rt.fail("cannot append to a %s", rt.typeOf(av))
}

// phpArrGet reads $a[k]. A missing key is null (PHP warns and yields null; the
// interpreter answers null too), a string indexes its BYTES.
func (rt *jsrt) phpArrGet(av interface{}, k interface{}) interface{} {
	if keys, vals, ok := phpArrParts(av); ok {
		if i := phpKeyIndex(keys, k); i >= 0 {
			return phpDeref(vals.elems[i])
		}
		return jsNull
	}
	if s, ok := av.(string); ok {
		i := int(rt.phpToLong(k))
		if i < 0 {
			i += len(s)
		}
		if i < 0 || i >= len(s) {
			return ""
		}
		return string(s[i])
	}
	if isNullish(av) {
		return jsNull
	}
	if o, ok := av.(*jsObject); ok { // ArrayAccess-ish: read the property
		if pv, has := o.props[rt.phpStr(k)]; has {
			return pv
		}
		return jsNull
	}
	return jsNull
}

func (rt *jsrt) phpArrHas(av interface{}, k interface{}) bool {
	if keys, vals, ok := phpArrParts(av); ok {
		i := phpKeyIndex(keys, k)
		return i >= 0 && !isNullish(vals.elems[i])
	}
	if s, ok := av.(string); ok {
		if !phpNumericKey(k) {
			return false
		}
		i := int(rt.phpToLong(k))
		return i >= 0 && i < len(s)
	}
	if o, ok := av.(*jsObject); ok {
		pv, has := o.props[rt.phpStr(k)]
		return has && !isNullish(pv)
	}
	return false
}

func (rt *jsrt) phpArrUnset(av interface{}, k interface{}) {
	o, ok := phpAsArr(av)
	if !ok {
		return
	}
	keys, _ := o.props["keys"].(*jsArray)
	vals, _ := o.props["vals"].(*jsArray)
	i := phpKeyIndex(keys, k)
	if i < 0 {
		return
	}
	keys.dropIdx()
	vals.dropIdx()
	keys.elems = append(keys.elems[:i], keys.elems[i+1:]...)
	vals.elems = append(vals.elems[:i], vals.elems[i+1:]...)
}

// phpElemCell is the storage CELL of an array element, created on demand - the twin
// of php-interpreter.abnf's arrCell. It must answer the SAME cell for the same
// element every time: building a fresh one per '&' silently detaches every alias
// taken earlier, so `$x = &$a[0]; $y = &$a[0]; $y = 9;` left $x behind.
func (rt *jsrt) phpElemCell(av interface{}, k interface{}) interface{} {
	keys, vals, ok := phpArrParts(av)
	if !ok {
		return jsNull
	}
	nk := phpNormKey(k)
	i := phpKeyIndex(keys, nk)
	if i < 0 {
		rt.phpArrSet(av, nk, jsNull)
		keys, vals, ok = phpArrParts(av)
		if !ok {
			return jsNull
		}
		i = phpKeyIndex(keys, nk)
		if i < 0 {
			return jsNull
		}
	}
	if c := phpCellOf(vals.elems[i]); c != nil {
		return c
	}
	cell := newJSObject()
	cell.set("__refcell", true)
	cell.set("v", vals.elems[i])
	vals.elems[i] = cell
	return cell
}

// phpArrCopy is PHP's value assignment: an array is copied (deeply, since a
// nested array is a value too), an object is not.
func (rt *jsrt) phpArrCopy(v interface{}) interface{} {
	keys, vals, ok := phpArrParts(v)
	if !ok {
		return v
	}
	out := phpNewArr()
	ok2, _ := out.props["keys"].(*jsArray)
	ov, _ := out.props["vals"].(*jsArray)
	ok2.elems = append(ok2.elems, keys.elems...)
	for _, e := range vals.elems {
		ov.elems = append(ov.elems, rt.phpArrCopy(phpDeref(e)))
	}
	if o, isO := phpAsArr(v); isO {
		out.set("__next", phpArrNext(o))
	} else {
		out.set("__next", float64(len(keys.elems)))
	}
	return out
}

func (rt *jsrt) phpLen(v interface{}) float64 {
	if keys, _, ok := phpArrParts(v); ok {
		return float64(len(keys.elems))
	}
	if s, ok := v.(string); ok {
		return float64(len(s))
	}
	if isNullish(v) {
		return 0
	}
	return 1
}

// ----- casts -----------------------------------------------------------------

func (rt *jsrt) phpCast(kind string, v interface{}) interface{} {
	switch kind {
	case "int", "integer":
		// (int) of an out-of-range float WRAPS modulo 2^64 rather than saturating:
		// (int)(PHP_INT_MAX * 2 + 4) is int(0). Corpus: tests/int_overflow_64bit.phpt.
		return phpMkInt(rt.phpToLong(v))
	case "float", "double", "real":
		return phpMkFlo(phpNumOf(rt.phpToNum(v)))
	case "string", "binary":
		return rt.phpStr(v)
	case "bool", "boolean":
		return rt.phpTest(v)
	case "array":
		if phpIsArr(v) {
			return v
		}
		out := phpNewArr()
		if isNullish(v) {
			return out
		}
		if o, ok := v.(*jsObject); ok && phpIsObj(v) {
			for _, k := range o.keys {
				if phpHiddenProp(k) {
					continue
				}
				rt.phpArrSet(out, k, o.props[k])
			}
			return out
		}
		rt.phpArrPush(out, v)
		return out
	case "object":
		return v
	case "unset":
		return jsNull
	}
	return v
}

// phpJoinKeys / phpJoinVals back array_keys() and array_values().
func (rt *jsrt) phpKeysOf(v interface{}) interface{} {
	out := phpNewArr()
	if keys, _, ok := phpArrParts(v); ok {
		for _, k := range keys.elems {
			rt.phpArrPush(out, k)
		}
	}
	return out
}

func (rt *jsrt) phpValsOf(v interface{}) interface{} {
	out := phpNewArr()
	if _, vals, ok := phpArrParts(v); ok {
		for _, e := range vals.elems {
			rt.phpArrPush(out, phpDeref(e))
		}
	}
	return out
}

func (rt *jsrt) phpInArray(needle, hay interface{}) bool {
	if _, vals, ok := phpArrParts(hay); ok {
		for _, e := range vals.elems {
			if rt.phpLoose(phpDeref(e), needle) {
				return true
			}
		}
	}
	return false
}

// phpImplode is the string form a template / echo needs for an array-free path;
// kept next to the other coercions so both engines share one spelling.
func phpImplode(parts []string, sep string) string { return strings.Join(parts, sep) }

// ----- classes: constants, static properties, :: dispatch --------------------
//
// A class descriptor is the same *jsObject the compiler already builds (__name,
// __super, one property per method). Two prefixed key spaces are added on top,
// because PHP keeps constants, static properties and methods in SEPARATE
// namespaces: "k:" for a class constant and "s:" for a static property. Both are
// resolved by walking the __super chain, which is what makes a static property
// SHARED with the subclasses (the write lands on the declaring class) and a
// constant inherited.

const phpConstPfx = "k:"
const phpStaticPfx = "s:"

// phpClassOf resolves the left side of '::' to a class descriptor: an instance
// yields its class, a descriptor is itself.
func (rt *jsrt) phpClassOf(v interface{}) *jsObject {
	o, ok := v.(*jsObject)
	if !ok {
		rt.fail(":: on a %s", rt.typeOf(v))
	}
	if _, isClass := o.props["__isclass"]; isClass {
		return o
	}
	if cls, has := o.props["__class"]; has {
		if c, ok := cls.(*jsObject); ok {
			return c
		}
	}
	rt.fail(":: on a value that is not an object or a class")
	return nil
}

// phpOwner walks the __super chain for the class that actually declares `key`.
func phpOwner(cls *jsObject, key string) *jsObject {
	for guard := 0; cls != nil && guard < 64; guard++ {
		if _, has := cls.props[key]; has {
			return cls
		}
		sup, has := cls.props["__super"]
		if !has {
			return nil
		}
		cls, _ = sup.(*jsObject)
	}
	return nil
}

func (rt *jsrt) phpConst(target interface{}, name string) interface{} {
	cls := rt.phpClassOf(target)
	if v, ok := phpConstIn(cls, name, 0); ok {
		return v
	}
	rt.fail("undefined constant %s::%s", rt.phpStr(cls.props["__name"]), name)
	return nil
}

// phpConstIn walks a class's own chain and then its interfaces: PHP inherits a
// constant from a base class AND from every implemented interface.
func phpConstIn(cls *jsObject, name string, depth int) (interface{}, bool) {
	if cls == nil || depth > 16 {
		return nil, false
	}
	if o := phpOwner(cls, phpConstPfx+name); o != nil {
		return o.props[phpConstPfx+name], true
	}
	for k := cls; k != nil; {
		if ifs, has := k.props["__ifaces"]; has {
			if _, vals, ok := phpArrParts(ifs); ok {
				for _, iv := range vals.elems {
					if io, isO := iv.(*jsObject); isO {
						if v, found := phpConstIn(io, name, depth+1); found {
							return v, true
						}
					}
				}
			}
		}
		sup, has := k.props["__super"]
		if !has {
			break
		}
		k, _ = sup.(*jsObject)
	}
	return nil, false
}

func (rt *jsrt) phpStaticGet(target interface{}, name string) interface{} {
	cls := rt.phpClassOf(target)
	if o := phpOwner(cls, phpStaticPfx+name); o != nil {
		return o.props[phpStaticPfx+name]
	}
	return jsNull
}

func (rt *jsrt) phpStaticSet(target interface{}, name string, v interface{}) {
	cls := rt.phpClassOf(target)
	o := phpOwner(cls, phpStaticPfx+name)
	if o == nil {
		o = cls // First write declares it here.
	}
	o.set(phpStaticPfx+name, v)
}

// phpIsA is the runtime type test behind instanceof and a typed catch: a name
// matches the class itself, any base class, and any implemented interface (which
// may itself extend others).
func (rt *jsrt) phpIsA(v interface{}, name string) bool {
	// A generator IS an instance of PHP's Generator class. It is not a *jsObject, so
	// it carries no descriptor to walk - the name is answered directly, which also
	// gives the compiler a safe "is this a generator" probe for foreach.
	if _, isGen := v.(*jsGenerator); isGen {
		return name == "Generator" || name == "Traversable" || name == "Iterator"
	}
	o, ok := v.(*jsObject)
	if !ok {
		return false
	}
	var cls *jsObject
	if _, isClass := o.props["__isclass"]; isClass {
		cls = o
	} else if c, has := o.props["__class"]; has {
		cls, _ = c.(*jsObject)
	}
	for guard := 0; cls != nil && guard < 64; guard++ {
		if rt.phpStr(cls.props["__name"]) == name {
			return true
		}
		if ifs, has := cls.props["__ifaces"]; has {
			if _, vals, ok := phpArrParts(ifs); ok {
				for _, iv := range vals.elems {
					// An entry is a descriptor (so an interface that extends
					// other interfaces matches those too) or, for the built-in
					// exception classes, a bare name.
					if io, isO := iv.(*jsObject); isO {
						if rt.phpIsA(io, name) {
							return true
						}
					} else if rt.phpStr(iv) == name {
						return true
					}
				}
			}
		}
		sup, has := cls.props["__super"]
		if !has {
			return false
		}
		cls, _ = sup.(*jsObject)
	}
	return false
}

// phpClone is PHP's `clone`: a shallow copy of the instance (array properties are
// values, so they copy) whose __clone hook, if the class declares one, runs on the
// COPY.
func (rt *jsrt) phpClone(v interface{}) interface{} {
	o, ok := v.(*jsObject)
	if !ok || !phpIsObj(v) {
		return v
	}
	out := newJSObject()
	for _, k := range o.keys {
		out.set(k, rt.phpArrCopy(o.props[k]))
	}
	if cls, has := o.props["__class"]; has {
		if m := phpClassFind(cls, "__clone"); m != nil {
			rt.call(m, jsUndef, []interface{}{out})
		}
	}
	return out
}

// ----- the externs this file owns --------------------------------------------
//
// Registration is additive: the init() below appends a registrar to
// rxExtraExterns, which jsrtregex.go runs LAST - so the handful of entries that
// share a name with one in jsrt.go REPLACE it, and the PHP-only externs are
// created here rather than in the shared table.

// phpHole is the PHP 8.5 partial-application placeholder: one shared sentinel,
// because it never reaches user data - it only ever sits in the bound-argument
// array a partial closure carries, and js_phpfa replaces every occurrence.
var phpHole = func() *jsObject {
	o := newJSObject()
	o.set("__pfahole", true)
	return o
}()

func phpIsHole(v interface{}) bool {
	o, ok := v.(*jsObject)
	if !ok {
		return false
	}
	t, has := o.props["__pfahole"]
	return has && t == true
}

// phpDumpIds hands out var_dump's object handles (#1, #2, ...) on first sight,
// which is enough for a single run's transcript to be stable.
var phpDumpIds = map[*jsrt]map[*jsObject]int{}

func (rt *jsrt) phpDumpID(o *jsObject) int {
	m, has := phpDumpIds[rt]
	if !has {
		m = map[*jsObject]int{}
		phpDumpIds[rt] = m
	}
	if id, seen := m[o]; seen {
		return id
	}
	id := len(m) + 1
	m[o] = id
	return id
}

// phpTypeName is gettype()'s historical spelling; phpDebugType is
// get_debug_type()'s modern one.
func (rt *jsrt) phpTypeName(v interface{}) string {
	switch v.(type) {
	case jsUndefT, jsNullT:
		return "NULL"
	case bool:
		return "boolean"
	case float64, jsGInt:
		return "integer"
	case jsPhpFlo:
		return "double"
	case string:
		return "string"
	}
	if phpIsArr(v) {
		return "array"
	}
	if phpIsObj(v) {
		return "object"
	}
	return "unknown type"
}

func (rt *jsrt) phpDebugType(v interface{}) string {
	switch v.(type) {
	case jsUndefT, jsNullT:
		return "null"
	case bool:
		return "bool"
	case float64, jsGInt:
		return "int"
	case jsPhpFlo:
		return "float"
	case string:
		return "string"
	}
	if phpIsArr(v) {
		return "array"
	}
	if phpIsObj(v) {
		return rt.phpClassName(v)
	}
	return "unknown"
}

func (rt *jsrt) phpClassName(v interface{}) string {
	if o, ok := v.(*jsObject); ok {
		if cls, has := o.props["__class"]; has {
			if c, isO := cls.(*jsObject); isO {
				return rt.phpStr(c.props["__name"])
			}
		}
	}
	return "stdClass"
}

// phpDump is var_dump's rendering. A float goes through phpFloDump
// (serialize_precision=-1), NOT through the precision=14 renderer echo uses -
// that is the trap the two renderers exist to keep apart.
func (rt *jsrt) phpDump(v interface{}, ind int) string {
	pad := strings.Repeat(" ", ind)
	switch t := v.(type) {
	case jsUndefT, jsNullT:
		return pad + "NULL\n"
	case bool:
		if t {
			return pad + "bool(true)\n"
		}
		return pad + "bool(false)\n"
	case float64, jsGInt:
		return pad + "int(" + rt.phpStr(v) + ")\n"
	case jsPhpFlo:
		return pad + "float(" + phpFloDump(t.f) + ")\n"
	case string:
		return pad + "string(" + strconv.Itoa(len(t)) + ") \"" + t + "\"\n"
	}
	if keys, vals, ok := phpArrParts(v); ok {
		out := pad + "array(" + strconv.Itoa(len(keys.elems)) + ") {\n"
		for i := range keys.elems {
			out += rt.phpDumpKey(keys.elems[i], ind+2)
			out += rt.phpDump(phpDeref(vals.elems[i]), ind+2)
		}
		return out + pad + "}\n"
	}
	if phpIsObj(v) {
		o := v.(*jsObject)
		names := []string{}
		for _, k := range o.keys {
			if !phpHiddenProp(k) {
				names = append(names, k)
			}
		}
		out := pad + "object(" + rt.phpClassName(v) + ")#" + strconv.Itoa(rt.phpDumpID(o)) +
			" (" + strconv.Itoa(len(names)) + ") {\n"
		for _, k := range names {
			out += pad + "  [\"" + k + "\"]=>\n"
			out += rt.phpDump(phpDeref(o.props[k]), ind+2)
		}
		return out + pad + "}\n"
	}
	return pad + "NULL\n"
}

func (rt *jsrt) phpDumpKey(k interface{}, ind int) string {
	if s, ok := k.(string); ok {
		return strings.Repeat(" ", ind) + "[\"" + s + "\"]=>\n"
	}
	return strings.Repeat(" ", ind) + "[" + rt.phpStr(k) + "]=>\n"
}

// phpLit is an integer literal that a double cannot hold: the emitter passes the
// DIGITS, because a literal above 2^53 would already have been rounded on the
// way through the grammar's own JS numbers. A literal that does not FIT in 64
// bits is a float, not a wrapped int - 0xFFFFFFFFFFFFFFFF is
// float(1.8446744073709552E+19) and a long enough digit run is float(INF).
// Corpus: lang/integer_literals/*_64bit.phpt.
func phpLit(text string, radix int) interface{} {
	if v, err := strconv.ParseInt(text, radix, 64); err == nil {
		return phpMkInt(v)
	}
	if radix == 10 {
		f, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return float64(0)
		}
		return phpMkFlo(f)
	}
	// A power-of-two radix accumulates, exactly as php-interpreter.abnf's
	// radixFloat does, which is what zend_hex_strtod amounts to.
	v := 0.0
	for i := 0; i < len(text); i++ {
		d := 0
		c := text[i]
		switch {
		case c >= '0' && c <= '9':
			d = int(c - '0')
		case c >= 'a' && c <= 'f':
			d = int(c-'a') + 10
		case c >= 'A' && c <= 'F':
			d = int(c-'A') + 10
		default:
			continue
		}
		v = v*float64(radix) + float64(d)
	}
	return phpMkFlo(v)
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

		// phpmain hands its scope over once, so phpRaise can find the emitted
		// DivisionByZeroError / ArithmeticError descriptors.
		m["js_phrtinit"] = func(a []uint64) uint64 {
			phpMainScope[rt] = a[0]
			return 0
		}
		// The four that REPLACE a jsrt.go entry: each of them was 32 bit or
		// double based and is now exact 64 bit.
		m["js_phneg"] = func(a []uint64) uint64 { return w(rt.phpNeg(u(a[0]))) }
		m["js_phbnot"] = func(a []uint64) uint64 { return w(phpMkInt(^rt.phpToLong(u(a[0])))) }
		m["js_phintdiv"] = func(a []uint64) uint64 {
			y := rt.phpToLong(u(a[1]))
			if y == 0 {
				rt.phpRaise("DivisionByZeroError", "Division by zero")
			}
			x := rt.phpToLong(u(a[0]))
			if y == -1 && x == phpIntMin {
				rt.phpRaise("ArithmeticError", "Division of PHP_INT_MIN by -1 is not an integer")
			}
			return w(phpMkInt(x / y))
		}
		m["js_phabs"] = func(a []uint64) uint64 {
			n := rt.phpToNum(u(a[0]))
			if f, isF := n.(jsPhpFlo); isF {
				if f.f < 0 {
					return w(phpMkFlo(-f.f))
				}
				return w(n)
			}
			if phpLongOf(n) < 0 {
				return w(rt.phpNeg(n))
			}
			return w(n)
		}
		// A literal the grammar's own JS numbers cannot carry: (digits, radix).
		m["js_philit"] = func(a []uint64) uint64 {
			return w(phpLit(rt.toString(u(a[0])), jsToInt(rt.toNumber(u(a[1])))))
		}
		// The type predicates and var_dump: the whole point of the boxes is that
		// the int / float distinction is OBSERVABLE.
		m["js_phis"] = func(a []uint64) uint64 {
			v := u(a[1])
			switch rt.toString(u(a[0])) {
			case "int":
				return boolH(phpIsInt(v))
			case "float":
				return boolH(phpIsFlo(v))
			case "string":
				_, ok := v.(string)
				return boolH(ok)
			case "bool":
				_, ok := v.(bool)
				return boolH(ok)
			case "array":
				return boolH(phpIsArr(v))
			case "object":
				return boolH(phpIsObj(v))
			case "null":
				return boolH(isNullish(v))
			case "numeric":
				if s, ok := v.(string); ok {
					return boolH(phpNumStrVal(s) != nil)
				}
				return boolH(phpIsNum(v))
			}
			return jsHFalse
		}
		// PHP 8.5 PARTIAL FUNCTION APPLICATION. `foo(1, ?, 3)` binds what is
		// there and leaves a hole per `?`; the compiler stores the bound run in
		// one PHP array with the sentinel below in each hole, and js_phpfa fills
		// the holes - in source order - from the arguments the partial closure is
		// finally called with. Anything left over is appended, which is what the
		// trailing-`...` form is for (superfluous_args_are_forwarded.phpt).
		m["js_phhole"] = func(a []uint64) uint64 { return w(phpHole) }
		m["js_phpfa"] = func(a []uint64) uint64 {
			out := phpNewArr()
			bk, bv, ok := phpArrParts(u(a[0]))
			if !ok {
				return w(out)
			}
			var sup []interface{}
			var supKeys, supVals []interface{}
			if sk, sv, ok2 := phpArrParts(u(a[1])); ok2 {
				for i := range sk.elems {
					if _, isNum := sk.elems[i].(float64); isNum {
						sup = append(sup, phpDeref(sv.elems[i]))
					} else {
						supKeys = append(supKeys, sk.elems[i])
						supVals = append(supVals, phpDeref(sv.elems[i]))
					}
				}
			}
			k := 0
			for i := range bk.elems {
				v := bv.elems[i]
				if phpIsHole(v) {
					v = interface{}(jsNull)
					if k < len(sup) {
						v = sup[k]
					}
					k++
				}
				if _, isNum := bk.elems[i].(float64); isNum {
					rt.phpArrPush(out, v)
				} else {
					rt.phpArrSet(out, bk.elems[i], v)
				}
			}
			// Only the trailing-`...` form forwards what is left over:
			// superfluous_args_are_forwarded.phpt calls both `f(?, ...)` and
			// `f(?)` with three arguments and expects three and one.
			if rt.phpToLong(u(a[2])) != 0 {
				for ; k < len(sup); k++ {
					rt.phpArrPush(out, sup[k])
				}
			}
			for i := range supKeys {
				rt.phpArrSet(out, supKeys[i], supVals[i])
			}
			return w(out)
		}
		// AUTO-VIVIFICATION. `$a[0]["k"] = 1` creates the intermediate array in
		// PHP, and php-interpreter.abnf's resolveRefPhp does the same; these three
		// are the compiler half's version of that walk, one per step shape.
		//
		// They only ever run on an INTERMEDIATE step of an assignment target, and
		// they only create when the container really is a PHP array - a property
		// step on an object, a string, or anything else is left exactly as
		// js_phget would have read it, which is what the interpreter does too
		// (its guard is `isArr(o)`).
		m["js_phviv"] = func(a []uint64) uint64 {
			cont, key := u(a[0]), u(a[1])
			cur := rt.phpArrGet(cont, key)
			if isNullish(cur) {
				if _, ok := phpAsArr(cont); ok {
					na := phpNewArr()
					rt.phpArrSet(cont, key, na)
					return w(na)
				}
			}
			return w(cur)
		}
		// The `[]` step: `$c[]["p"] = 7` appends a fresh array and descends into it.
		m["js_phvivpush"] = func(a []uint64) uint64 {
			cont := u(a[0])
			if _, ok := phpAsArr(cont); ok {
				na := phpNewArr()
				rt.phpArrPush(cont, na)
				return w(na)
			}
			return w(jsNull)
		}
		// The ROOT of the walk: `$e = null; $e["x"]["y"] = 5` makes $e an array.
		// A pure function of the value - the caller writes the answer back with
		// its own setVarV, so a $e that is a REFERENCE has the array installed
		// through its cell. Only a target with more than one step gets this,
		// exactly as in the interpreter (resolveRefPhp guards on path.length > 1),
		// so `$a[0] = 1` on a null $a stays an error in both halves.
		m["js_phvivroot"] = func(a []uint64) uint64 {
			if cur := u(a[0]); !isNullish(cur) {
				return a[0]
			}
			return w(phpNewArr())
		}
		m["js_phtype"] = func(a []uint64) uint64 { return rt.wrapStr(rt.phpTypeName(u(a[0]))) }
		m["js_phdtype"] = func(a []uint64) uint64 { return rt.wrapStr(rt.phpDebugType(u(a[0]))) }
		m["js_phdump"] = func(a []uint64) uint64 { return rt.wrapStr(rt.phpDump(u(a[0]), 0)) }
		m["js_phgetclass"] = func(a []uint64) uint64 {
			if phpIsObj(u(a[0])) {
				return rt.wrapStr(rt.phpClassName(u(a[0])))
			}
			return jsHFalse
		}
	})
}
