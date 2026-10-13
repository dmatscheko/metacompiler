package abnf

// PHP's value model for the compiler grammar (languages/php-to-llvm-ir.abnf).
//
// php-interpreter.abnf is the reference implementation: it models a PHP value as
//
//	null / bool / int (a plain number) / float (a BOX, so 3.0 !== 3 stays true) /
//	string (byte based) / array (ONE ordered key->value map, so === compares the
//	keys AND their order while == does not) / object
//
// The js_ph* externs below give the compiler the same model the interpreter
// has; plain JS value semantics would be silently WRONG (`3.0 === 3` must be
// false, `"0"` is falsy, and `==` is loose).
//
// This file is append-only with respect to the rest of the runtime: nothing here
// runs unless a program calls one of the js_ph* externs, which only the PHP
// compiler grammar emits. The one exception is the jsPhpFlo box, which
// rt.truthy / rt.toNumber / rt.toString / rt.toGoNatural learn about in jsrt.go -
// each as one extra case for a type nothing else creates.

import (
	"errors"
	"math"
	"os"
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
//
// echo's ndigit is the precision ini setting, phpPrecision, and rounded selects
// dtoa's mode 2 (twin of floStr's, which explains both).
func phpFloStr(n float64) string {
	p := phpPrecision
	if p < 0 {
		return phpFloFormat(n, 17, false)
	}
	if p == 0 {
		p = 1
	}
	return phpFloFormat(n, p, true)
}
func phpFloDump(n float64) string { return phpFloFormat(n, 17, false) }

// phpConstDepth counts the constant initializers running (phpConstIn).
var phpConstDepth int

// phpPrecision is the running program's precision ini setting (js_phprecision).
// One process runs one PHP program, so it is process state.
var phpPrecision = 14

// phpFloSplit answers the decimal digits of a positive finite double and the
// position of the decimal point: value == 0.<digits> * 10^<decpt>. With
// exact false they are the shortest round-tripping digits; with exact true the
// exact decimal correctly rounded to nd digits (strconv rounds half to even, as
// zend_dtoa's mode 2 does). No double has more than 767 significant digits, so
// a larger nd is cut to that many before strconv sees it.
func phpFloSplit(a float64, nd int, exact bool) (string, int) {
	prec := -1
	if exact {
		prec = nd - 1
		if nd > 800 {
			prec = 799
		}
	}
	s := strconv.FormatFloat(a, 'e', prec, 64) // "d.dddde±dd"
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

func phpFloFormat(n float64, nd int, rounded bool) string {
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
	d, decpt := phpFloSplit(a, nd, false)
	if rounded && (nd > 15 || a < 0x1p-1022 || (len(d) == nd+1 && d[nd] == '5')) {
		d, decpt = phpFloSplit(a, nd, true)
	}
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
// is a type assertion that fails immediately.
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

// phpRefHolders is how many slots hold a reference cell, its "rc"; absent is the
// one slot that made it. It is counted where a reference is bound or unbound - '=&',
// foreach by reference, use (&$v), [&$v], global, unset() and an array copy keeping
// the cell - but not when a frame returns or an array (an array literal assigned
// and copied, too) is discarded, so it can only run high. A by-reference argument
// is not counted: its binding ends with the call. The count is read when the copy
// is made, at the assignment; PHP reads it at the copy's first write.
// Twin of phRefHolders in languages/lib/php-rt.metajs and refHolders in
// php-interpreter.abnf.
func phpRefHolders(c *jsObject) float64 {
	if n, ok := c.props["rc"].(float64); ok {
		return n
	}
	return 1
}

// phpRefAdjust adds d holders to c, when there is a cell.
func phpRefAdjust(c *jsObject, d float64) {
	if c != nil {
		c.set("rc", phpRefHolders(c)+d)
	}
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
		// __toString, found under its lower-cased name, when the class
		// declares one (php-interpreter.abnf's objToString); otherwise PHP
		// raises, and the compiler prints the same placeholder the JS
		// runtime would.
		if o, ok := v.(*jsObject); ok {
			if cls, has := o.props["__class"]; has {
				if m := phpClassFind(cls, "__tostring"); m != nil {
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

// phpPlaceArgs lays a call's arguments out in the parameter positions of a callee
// known only at run time. sig is the parameter list the emitter recorded in the
// function's meta name (withSig in php-to-llvm-ir.abnf): the names without '$',
// comma separated, the variadic one spelled "...name"; ")" for a function with no
// parameters, "" when nothing was recorded. args is the runtime list of a plain
// call, or the PHP array a call with named arguments or a spread collected
// (positional entries under integer keys, named ones under their names). A
// parameter takes its position's argument, else the argument named for it, else
// undefined (its default); the variadic one collects the remaining positions and
// every named argument no parameter took. Against a known list, a named argument
// that no parameter declares and no variadic collects raises "Unknown named
// parameter", and one naming a parameter a position already filled raises
// "overwrites previous argument", both in argument order. mask has bit i set for a
// by-reference parameter i: an element of a collected array that is a reference
// arrives there as that reference (call_user_func_array, a spread). An element
// that is none arrives as its value, where a spread in PHP first makes it a
// reference in the spread array (ZEND_SEND_UNPACK).
// Twin of js_phplacesig in languages/lib/php-rt.metajs.
func (rt *jsrt) phpPlaceArgs(sig string, mask int, args interface{}) interface{} {
	list, isList := args.(*jsArray)
	if isList && !strings.Contains(sig, "...") {
		return list
	}
	keys, vals, ok := phpArrParts(args)
	out := &jsArray{}
	if !ok {
		return out
	}
	// A runtime list may carry the CELL of a by-reference argument, which has to
	// arrive as it is; a collected PHP array holds values, but for a by-reference
	// parameter pi (-1: none) its references.
	at := func(i, pi int) interface{} {
		if isList || (pi >= 0 && pi < 53 && (mask>>uint(pi))&1 == 1) {
			return vals.elems[i]
		}
		return phpDeref(vals.elems[i])
	}
	if sig == "" {
		for i, k := range keys.elems {
			if n, isPos := k.(float64); isPos {
				out.elems = append(out.elems, at(i, int(n)))
			}
		}
		return out
	}
	params := strings.Split(sig, ",")
	if sig == ")" {
		params = nil
	}
	for _, k := range keys.elems {
		name, isName := k.(string)
		if !isName {
			continue
		}
		at, variadic := -1, false
		for pi, p := range params {
			if strings.HasPrefix(p, "...") {
				variadic = true
			} else if p == name {
				at = pi
			}
		}
		if at < 0 && !variadic {
			rt.phpRaise("Error", "Unknown named parameter $"+name)
		}
		if at >= 0 && rt.phpKeyIndex(keys, float64(at)) >= 0 {
			rt.phpRaise("Error", "Named parameter $"+name+" overwrites previous argument")
		}
	}
	taken := map[string]bool{}
	for pi, p := range params {
		if strings.HasPrefix(p, "...") {
			rest := phpNewArr()
			for i, k := range keys.elems {
				if n, isPos := k.(float64); isPos && int(n) >= pi {
					rt.phpArrPush(rest, at(i, -1))
				}
			}
			for i, k := range keys.elems {
				if s, isName := k.(string); isName && !taken[s] {
					rt.phpArrSet(rest, s, at(i, -1))
				}
			}
			out.elems = append(out.elems, rest)
			return out
		}
		var v interface{} = jsUndef
		if i := rt.phpKeyIndex(keys, float64(pi)); i >= 0 {
			v = at(i, pi)
		} else if i := rt.phpKeyIndex(keys, p); i >= 0 {
			v = at(i, pi)
			taken[p] = true
		}
		out.elems = append(out.elems, v)
	}
	// Positions past the declared parameters still travel (func_get_args).
	for i, k := range keys.elems {
		if n, isPos := k.(float64); isPos && int(n) >= len(params) {
			out.elems = append(out.elems, at(i, int(n)))
		}
	}
	return out
}

// ----- numeric strings and arithmetic coercion -------------------------------

func phpIsSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\v' || c == '\f'
}

// phpNumScan reads the numeric PREFIX both readings of a string share: leading
// whitespace, then an optionally signed decimal / float with its exponent. It
// answers the number's span and whether it is a float; ok is false when no digit
// was seen. Twin of numScan in php-interpreter.abnf and phNumScan in
// languages/lib/php-rt.metajs.
func phpNumScan(s string) (st, en int, isF, ok bool) {
	i := 0
	for i < len(s) && phpIsSpace(s[i]) {
		i++
	}
	st = i
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		i++
	}
	digits := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
		digits++
	}
	if i < len(s) && s[i] == '.' {
		isF = true
		i++
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
			digits++
		}
	}
	if digits == 0 {
		return 0, 0, false, false
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
	return st, i, isF, true
}

// phpNumSpanVal is the number s[st:en] (a span phpNumScan accepted) denotes.
func phpNumSpanVal(s string, st, en int, isF bool) interface{} {
	if isF {
		f, ok := phpStrtod(s[st:en])
		if !ok {
			return nil
		}
		return phpMkFlo(f)
	}
	return phpIntStrVal(s[st:en])
}

// phpNumStrVal answers the number a fully numeric string denotes (an int as a
// float64, a float as a jsPhpFlo), or nil when the string is not numeric.
// Leading and trailing whitespace is allowed (PHP 8).
func phpNumStrVal(s string) interface{} {
	st, en, isF, ok := phpNumScan(s)
	if !ok {
		return nil
	}
	i := en
	for i < len(s) && phpIsSpace(s[i]) {
		i++
	}
	if i != len(s) {
		return nil
	}
	return phpNumSpanVal(s, st, en, isF)
}

// phpStrtod reads a decimal number the caller has already scanned. A magnitude
// past the double range is +-INF, as zend_strtod answers; strconv.ParseFloat
// reports that as ErrRange beside the same value.
func phpStrtod(txt string) (float64, bool) {
	f, err := strconv.ParseFloat(txt, 64)
	return f, err == nil || errors.Is(err, strconv.ErrRange)
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
	f, ok := phpStrtod(txt)
	if !ok {
		return nil
	}
	return phpMkFlo(f)
}

// phpLeadingNum: a non-numeric string still contributes its numeric PREFIX,
// exponent and leading whitespace included ("12abc" is 12, " 5e1 z" is 50.0).
func phpLeadingNum(s string) interface{} {
	st, en, isF, ok := phpNumScan(s)
	if !ok {
		return float64(0)
	}
	if v := phpNumSpanVal(s, st, en, isF); v != nil {
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

// phpNoNum: a value with no number reading - an array, an object, a closure, a
// generator or a string with no numeric prefix (PHP 8's TypeError, where
// "5 apples" still reads 5). Twin of noNum in php-interpreter.abnf and phNoNum
// in lib/php-rt.metajs.
func phpNoNum(v interface{}) bool {
	switch t := v.(type) {
	case float64, jsGInt, jsPhpFlo, bool, jsUndefT, jsNullT:
		return false
	case string:
		_, _, _, ok := phpNumScan(t)
		return !ok
	}
	return true
}

// phpArrArith: an operand with no number reading. '+' of two arrays is the UNION
// (the left array, then each key of the right one it lacks); anything else is a
// TypeError. Twin of phpArrArith in php-interpreter.abnf.
func (rt *jsrt) phpArrArith(op string, l, r interface{}) interface{} {
	rk, rv, rok := phpArrParts(r)
	if op == "+" && phpIsArr(l) && rok {
		out := rt.phpArrCopy(l)
		ok, _, _ := phpArrParts(out)
		for i, k := range rk.elems {
			if rt.phpKeyIndex(ok, k) < 0 {
				rt.phpArrSet(out, k, rt.phpArrCopy(phpDeref(rv.elems[i])))
			}
		}
		return out
	}
	rt.phpBinopErr(op, l, r)
	return nil
}

// phpBinopErr is zend_binop_error: PHP 8's TypeError for an operand an operator
// cannot use.
func (rt *jsrt) phpBinopErr(op string, l, r interface{}) {
	rt.phpRaise("TypeError", "Unsupported operand types: "+rt.phpDebugType(l)+" "+op+" "+rt.phpDebugType(r))
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
	if phpNoNum(l) || phpNoNum(r) {
		return rt.phpArrArith(op, l, r)
	}
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
		return phpMkFlo(crPow(x, y))
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
			return phpMkFlo(crPow(float64(x), float64(y)))
		}
		// Squaring, with the same overflow test every multiplication gets: the
		// first product that leaves 64 bits makes the WHOLE result a float.
		acc, base, e := int64(1), x, y
		for e > 0 {
			if e&1 == 1 {
				r, ov := phpMulOv(acc, base)
				if ov {
					return phpMkFlo(crPow(float64(x), float64(y)))
				}
				acc = r
			}
			e >>= 1
			if e > 0 {
				r, ov := phpMulOv(base, base)
				if ov {
					return phpMkFlo(crPow(float64(x), float64(y)))
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
	// Two strings under &, | or ^ combine byte by byte and stay a string.
	if ls, ok := l.(string); ok && op != "<<" && op != ">>" {
		if rs, ok := r.(string); ok {
			return phpBitOp(op, ls, rs)
		}
	}
	if phpNoNum(l) || phpNoNum(r) {
		rt.phpBinopErr(op, l, r)
	}
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

// phpIncDec is PHP 8.3's increment_function / decrement_function: a number steps
// by one; null++ is 1 and null-- stays null; a bool does not change; "" becomes
// "1" or -1; a numeric string steps as its number; any other string increments
// Perl-style (phpIncr) and does not decrement. An array or an object is a
// TypeError. Twin of phpIncDec in php-interpreter.abnf and phIncDec in
// lib/php-rt.metajs.
func (rt *jsrt) phpIncDec(v interface{}, inc bool) interface{} {
	op := "-"
	if inc {
		op = "+"
	}
	switch t := v.(type) {
	case jsUndefT, jsNullT:
		if inc {
			return float64(1)
		}
		return jsNull
	case bool:
		return t
	case string:
		if t == "" {
			if inc {
				return "1"
			}
			return float64(-1)
		}
		n := phpNumStrVal(t)
		if n == nil {
			if inc {
				return phpIncr(t)
			}
			return t
		}
		return rt.phpArith(op, n, float64(1))
	}
	if phpNoNum(v) {
		verb := "decrement "
		if inc {
			verb = "increment "
		}
		rt.phpRaise("TypeError", "Cannot "+verb+rt.phpDebugType(v))
	}
	return rt.phpArith(op, v, float64(1))
}

// phpNeg is unary minus. It keeps the operand's TYPE (so -0.0 stays -0), and the
// one int that cannot be negated, PHP_INT_MIN, promotes like any other overflow.
func (rt *jsrt) phpNeg(v interface{}) interface{} {
	// PHP compiles -$x as $x * -1, so an operand with no number reading fails as
	// that multiplication.
	if phpNoNum(v) {
		rt.phpBinopErr("*", v, float64(-1))
	}
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
	o := rt.phpMkErr(clsName, msg)
	if o == nil {
		rt.fail("%s", msg)
	}
	panic(&jsThrown{value: o})
}

// phpMkErr is phpRaise minus the throw, and the twin of phMkErr in
// languages/lib/php-rt.metajs. The floor's recoverable-name hook ANSWERS the
// value to throw rather than throwing it - the slot is cleared for the duration
// of the call so a hook that itself trips the same site cannot recurse - so
// js_phmissname needs the object without the raise. nil means "no descriptor,
// keep the fatal".
func (rt *jsrt) phpMkErr(clsName, msg string) *jsObject {
	cls := rt.phpRtClass(clsName)
	if cls == nil {
		return nil
	}
	o := newJSObject()
	o.set("__class", cls)
	o.set("message", msg)
	o.set("code", float64(0))
	rt.phpStamp(o)
	return o
}

// phpRtClass is the class descriptor phpmain handed over under clsName, or nil.
func (rt *jsrt) phpRtClass(clsName string) interface{} {
	h, has := phpMainScope[rt]
	if !has {
		return nil
	}
	// TWO shapes, because layer 2 cannot open a scope. php-to-llvm-ir.abnf hands
	// over a PHP ARRAY of name => descriptor, which is a plain value the MetaJS
	// runtime library can read too; a SCOPE (an older module's shape) is still
	// accepted.
	var found interface{}
	if keys, vals, ok := phpArrParts(rt.unwrap(h)); ok {
		for i := range keys.elems {
			if rt.phpStr(keys.elems[i]) == clsName {
				found = phpDeref(vals.elems[i])
			}
		}
	} else {
		found = rt.scopeGet(rt.scopeOf(h), clsName)
	}
	if found == nil || isNullish(found) {
		return nil
	}
	return found
}

// phpFindMethod walks a class descriptor's __class/__super chain, the same single
// inheritance chain rt.memberCall's instance arm walks. It exists so js_phmcall
// can tell a MISS from a hit BEFORE the shared dispatcher aborts on it.
func (rt *jsrt) phpFindMethod(cls interface{}, name string) interface{} {
	for cls != nil {
		clsObj, ok := cls.(*jsObject)
		if !ok {
			return nil
		}
		if m, ok := clsObj.props[name]; ok && isCallable(m) {
			return m
		}
		cls = clsObj.props["__super"]
	}
	return nil
}

// phpCalledScope is the class 'static::' names inside a method body: the runtime
// class of $this when there is one, else the class the running static call bound,
// else fallback. Twin of js_phlsb in languages/lib/php-rt.metajs.
func (rt *jsrt) phpCalledScope(this, fallback interface{}) interface{} {
	if o, ok := this.(*jsObject); ok {
		if cls, has := o.props["__class"]; has {
			return cls
		}
	}
	if rt.phpLsb != nil {
		return rt.phpLsb
	}
	return fallback
}

// phpCbFail raises a callback check's failure: PHP's TypeError "<where>, <reason>",
// or "<where> <reason>" for a where ending in a colon (Closure::fromCallable's
// "Failed to create closure from callable:"). Twin of phCbFail in
// languages/lib/php-rt.metajs.
func (rt *jsrt) phpCbFail(where, reason string) {
	if strings.HasSuffix(where, ":") {
		rt.phpRaise("TypeError", where+" "+reason)
	}
	rt.phpRaise("TypeError", where+", "+reason)
}

// phpCbMeth is the method half of a callback check: target (an object, or a class
// for a static callable) has the method, or the magic method that stands in for a
// missing one. Twin of phCbMeth in languages/lib/php-rt.metajs.
func (rt *jsrt) phpCbMeth(where string, target, name interface{}) {
	if rt.phpMethodOf(target, name) != nil {
		return
	}
	magic := "__callStatic"
	if phpIsObj(target) {
		magic = "__call"
	}
	if rt.phpMethodOf(target, magic) != nil {
		return
	}
	rt.phpCbFail(where, "class "+rt.phpStr(rt.phpClassOf(target).props["__name"])+" does not have a method \""+rt.phpStr(name)+"\"")
}

// phpMethodOf is the method `name` of target - an object, or a class for a static
// method - with the lower-cased retry js_phmcall makes, or nil.
func (rt *jsrt) phpMethodOf(target, nameV interface{}) interface{} {
	cls := target
	if o, isObj := target.(*jsObject); isObj {
		if _, isClass := o.props["__isclass"]; !isClass {
			cls = o.props["__class"]
		}
	}
	name := rt.phpStr(nameV)
	if meth := rt.phpFindMethod(cls, name); meth != nil {
		return meth
	}
	return rt.phpFindMethod(cls, phpCase(name, false))
}

// phpCbFallback is where a callback naming a method cls lacks goes
// (zend_is_callable_check_func): the class whose __call runs on this, nil for
// cls's __callStatic, ok false for nowhere. form is 0 for 'C::m', 1 for
// ['C', 'm'] and 2 for 'C::m' checked from a method of an internal class
// (Closure::fromCallable). ['C', 'm'] with an object bound - any $this for a
// relative name, else this when the class scope is an instance of cls and this
// one of the scope - takes C's own __call (that object's for ['self', 'm'])
// and nothing else. Otherwise get_static_method_fallback decides, with this
// when the check runs in a function and none in a method of an internal class.
// Twin of cbFallback in php-interpreter.abnf and phCbFallback in
// languages/lib/php-rt.metajs.
func (rt *jsrt) phpCbFallback(cls *jsObject, cn string, form int, this, scope interface{}) (*jsObject, bool) {
	rn := phpCase(strings.TrimPrefix(cn, "\\"), false)
	rel := rn == "self" || rn == "parent" || rn == "static"
	if form == 1 && phpIsObj(this) {
		sc, _ := scope.(*jsObject)
		if rel || (sc != nil && rt.phpIsA(this, rt.phpStr(sc.props["__name"])) && rt.phpIsA(sc, rt.phpStr(cls.props["__name"]))) {
			vc := cls
			if rn == "self" {
				vc, _ = this.(*jsObject).props["__class"].(*jsObject)
			}
			if phpOwner(vc, "__call") == nil {
				return nil, false
			}
			return vc, true
		}
	}
	if form != 2 && rt.phpCallFallback(cls, this) {
		c, _ := this.(*jsObject).props["__class"].(*jsObject)
		return c, true
	}
	return nil, phpOwner(cls, "__callstatic") != nil
}

// phpCloHas: the methods [$closure, name] names validly. Twin of cloHasMethod
// in php-interpreter.abnf.
func phpCloHas(name string) bool {
	switch phpCase(name, false) {
	case "__invoke", "bind", "bindto", "call", "fromcallable":
		return true
	}
	return false
}

// phpCloCall is a method call on a Closure: __invoke calls it, bindTo rebinds
// it, call rebinds it to an object and that object's class and calls it at
// once. Twin of phCloCall in languages/lib/php-rt.metajs and cloMethod in
// php-interpreter.abnf.
func (rt *jsrt) phpCloCall(f interface{}, name string, args []interface{}) interface{} {
	var nthis, sv interface{} = jsNull, jsNull
	if len(args) > 0 {
		nthis = args[0]
	}
	if len(args) > 1 {
		sv = args[1]
	}
	switch phpCase(name, false) {
	case "__invoke":
		return rt.call(f, jsUndef, args)
	case "bind":
		// The static Closure::bind($closure, $newThis, $newScope), called through
		// an instance: $closure->bindTo($newThis, $newScope).
		if len(args) == 0 || !isCallable(args[0]) {
			rt.phpRaise("TypeError", "Closure::bind(): Argument #1 ($closure) must be of type Closure, "+rt.phpValueName(nthis)+" given")
		}
		return rt.phpCloCall(args[0], "bindto", args[1:])
	case "bindto":
		if len(args) < 2 {
			return rt.phpCloBind(f, nthis, jsNull, true)
		}
		if s, ok := sv.(string); ok && phpCase(s, false) == "static" {
			return rt.phpCloBind(f, nthis, jsNull, true)
		}
		if o, ok := sv.(*jsObject); ok && phpIsObj(o) {
			return rt.phpCloBind(f, nthis, o.props["__class"], false)
		}
		return rt.phpCloBind(f, nthis, sv, false)
	case "call":
		var scope interface{} = jsNull
		if o, ok := nthis.(*jsObject); ok && phpIsObj(o) {
			scope = o.props["__class"]
		}
		bf := rt.phpCloBind(f, nthis, scope, false)
		if isNullish(bf) {
			return jsNull
		}
		return rt.call(bf, jsUndef, args[1:])
	}
	rt.phpRaise("Error", "Call to undefined method Closure::"+name+"()")
	return jsUndef
}

// phpCloBind is f with its $this and class scope replaced (zend_create_closure_ex),
// made by the factory a closure literal's record keeps (cloFactory in
// php-to-llvm-ir.abnf), which also resolves a class NAME. keep leaves the class
// scope the record holds. A binding zend_valid_closure_binding refuses answers
// null (PHP also warns, which is not modelled): an instance for a static closure,
// none for a closure using $this that has one, and for a closure made from a
// callable (phpFakeBind) or a function any other scope. Twin of phCloBind in
// languages/lib/php-rt.metajs and cloBind in php-interpreter.abnf.
func (rt *jsrt) phpCloBind(f, nthis, scope interface{}, keep bool) interface{} {
	rec := phpLitRec(f)
	if rec == nil {
		if fr := phpFakeRec(f); fr != nil {
			return rt.phpFakeBind(f, fr, nthis, scope, keep)
		}
		// A function's closure has no class scope, so it takes none other. The
		// binding is a new Closure object (zend_create_closure_ex) running f.
		if keep || isNullish(scope) {
			mi := jsvFnMeta(f)
			return &hostFunc{name: "closure", meta: &mi, fn: func(rt *jsrt, this uint64, args []interface{}) interface{} {
				return rt.call(f, jsUndef, args)
			}}
		}
		return jsNull
	}
	if rec.props["f"] == nil {
		rt.fail("binding a Closure that is not a closure literal is not modelled")
	}
	if !phpIsObj(nthis) {
		nthis = jsNull
		if rec.props["ut"] == true && phpIsObj(rec.props["t"]) {
			return jsNull
		}
	} else if rec.props["st"] == true {
		return jsNull
	}
	if keep {
		scope = rec.props["sc"]
	}
	return rt.call(rec.props["f"], jsUndef, []interface{}{nthis, scope})
}

// phpLitRec is a closure literal's record (js_phclo), phpFakeRec that of a closure
// made from a callable (js_phfake); nil for anything else.
func phpLitRec(v interface{}) *jsObject {
	if rec := jsvFnMeta(v).rec; rec != nil && rec.props["fk"] != true {
		return rec
	}
	return nil
}

func phpFakeRec(v interface{}) *jsObject {
	if rec := jsvFnMeta(v).rec; rec != nil && rec.props["fk"] == true {
		return rec
	}
	return nil
}

// phpFakeBind: a closure made from a callable keeps its method's class as its
// scope: it binds an instance of that class unless the method is static, none
// unless it is, and no other scope (zend_valid_closure_binding). The record's
// factory makes the new closure, which runs the same method (js_phscall through
// its class) with the new $this. Twin of phFakeBind in
// languages/lib/php-rt.metajs and fakeBind in php-interpreter.abnf.
func (rt *jsrt) phpFakeBind(f interface{}, fr *jsObject, nthis, scope interface{}, keep bool) interface{} {
	sc, _ := fr.props["sc"].(*jsObject)
	st := fr.props["st"] == true
	if phpIsObj(nthis) {
		if st || !rt.phpIsA(nthis, rt.phpStr(sc.props["__name"])) {
			return jsNull
		}
	} else if !st {
		return jsNull
	}
	if !keep && !rt.phpSameClass(scope, sc) {
		return jsNull
	}
	var nt interface{} = jsUndef
	var lsb interface{} = sc
	if phpIsObj(nthis) {
		nt = nthis
		lsb = nthis.(*jsObject).props["__class"]
	}
	return rt.call(fr.props["f"], jsUndef, []interface{}{sc, fr.props["n"], nt, lsb, f})
}

// phpSameClass: whether a bind scope - a class, a class name, or null - is sc.
func (rt *jsrt) phpSameClass(scope interface{}, sc *jsObject) bool {
	if s, ok := scope.(string); ok {
		return phpCase(strings.TrimPrefix(s, "\\"), false) == phpCase(rt.phpStr(sc.props["__name"]), false)
	}
	o, ok := scope.(*jsObject)
	return ok && o == sc
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
			j := rt.phpKeyIndex(rk, lk.elems[i])
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
	if math.IsNaN(phpNumOf(n)) {
		return false
	}
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

// phpCmpNum is zend_compare's ZEND_THREEWAY_COMPARE: a NAN on either side
// answers 1.
func phpCmpNum(a, b float64) float64 {
	if a < b {
		return -1
	}
	if a == b {
		return 0
	}
	return 1
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

// phpCmp is the spaceship, and the engine behind < > <= >= (a > b is b < a).
// Twin of phpCmp in languages/php-interpreter.abnf.
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
			j := rt.phpKeyIndex(rk, lk.elems[i])
			if j < 0 {
				return 1
			}
			if cv := rt.phpCmp(phpDeref(lv.elems[i]), phpDeref(rv.elems[j])); cv != 0 {
				return cv
			}
		}
		return 0
	}
	if la {
		return 1
	}
	if ra {
		return -1
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
	if (lIsS && phpIsNum(r) && math.IsNaN(phpNumOf(r))) || (rIsS && phpIsNum(l) && math.IsNaN(phpNumOf(l))) {
		return 1
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
	// A leading zero is canonical only as the whole string "0": "-0" stays a
	// string key.
	if s[i] == '0' && len(s) > 1 {
		return false
	}
	for j := i; j < len(s); j++ {
		if s[j] < '0' || s[j] > '9' {
			return false
		}
	}
	return true
}

// phpKeyIndex finds key k in a key run by its normalized form. Keys are stored
// normalized (phpNormKey), and between two normalized keys PHP's key equality is
// strictEq - an int by value, a string by content - so a longer run goes through
// the shared hashed index (dictFind) and only a short one is scanned. Twin of
// phDictFind in languages/lib/php-rt.metajs.
func (rt *jsrt) phpKeyIndex(keys *jsArray, k interface{}) int {
	nk := phpNormKey(k)
	if len(keys.elems) >= 16 {
		return rt.dictFind(keys, nk)
	}
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

// phpArrPos is the array's internal pointer, 0 until one of current() and its
// family moves it (see js_phaptr).
func phpArrPos(o *jsObject) float64 {
	if n, ok := o.props["__pos"].(float64); ok {
		return n
	}
	return 0
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
		if rt.phpIsAA(av) {
			rt.memberCall(av, "offsetSet", []interface{}{k, v})
			return
		}
		rt.fail("cannot use a %s as an array", rt.typeOf(av))
	}
	keys, _ := o.props["keys"].(*jsArray)
	vals, _ := o.props["vals"].(*jsArray)
	nk := phpNormKey(k)
	if i := rt.phpKeyIndex(keys, nk); i >= 0 {
		if c := phpCellOf(vals.elems[i]); c != nil && phpCellOf(v) == nil {
			c.set("v", v)
		} else {
			vals.elems[i] = v
		}
	} else {
		dictAppend(keys, vals, nk, v)
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
	if rt.phpIsAA(av) {
		rt.memberCall(av, "offsetSet", []interface{}{jsNull, v})
		return
	}
	rt.fail("cannot append to a %s", rt.typeOf(av))
}

// phpArrGet reads $a[k]. A missing key is null (PHP warns and yields null; the
// interpreter answers null too), a string indexes its BYTES.
func (rt *jsrt) phpArrGet(av interface{}, k interface{}) interface{} {
	if keys, vals, ok := phpArrParts(av); ok {
		if i := rt.phpKeyIndex(keys, k); i >= 0 {
			return phpDeref(vals.elems[i])
		}
		return jsNull
	}
	if s, ok := av.(string); ok {
		return phpOffset(s, int(rt.phpToLong(k)))
	}
	if isNullish(av) {
		return jsNull
	}
	if rt.phpIsAA(av) {
		return rt.memberCall(av, "offsetGet", []interface{}{k})
	}
	if o, ok := av.(*jsObject); ok { // a non-ArrayAccess object: read the property
		if pv, has := o.props[rt.phpStr(k)]; has {
			return pv
		}
		return jsNull
	}
	return jsNull
}

// phpIsAA: an object whose class implements ArrayAccess, which $o[k] reads
// through offsetGet, writes through offsetSet (a null key for $o[] = v) and
// unsets through offsetUnset; isset($o[k]) asks offsetExists (js_phaisset).
// Twin of phpIsAA in languages/php-interpreter.abnf and phIsAA in
// lib/php-rt.metajs.
func (rt *jsrt) phpIsAA(v interface{}) bool {
	return phpIsObj(v) && rt.phpIsA(v, "ArrayAccess")
}

// phpAAIsset: isset($o[k]) on an ArrayAccess object - offsetExists, or the
// library's __mec_isset when the class declaring offsetExists declares that too.
// Twin of phpAAIsset in languages/php-interpreter.abnf.
func (rt *jsrt) phpAAIsset(o interface{}, k interface{}) bool {
	c, _ := o.(*jsObject).props["__class"].(*jsObject)
	for guard := 0; c != nil && guard < 64; guard++ {
		if _, ok := c.props["__ms:offsetexists"].(string); ok {
			if _, ok := c.props["__ms:__mec_isset"].(string); ok {
				return rt.phpTest(rt.memberCall(o, "__mec_isset", []interface{}{k}))
			}
			break
		}
		c, _ = c.props["__super"].(*jsObject)
	}
	return rt.phpTest(rt.memberCall(o, "offsetExists", []interface{}{k}))
}

func (rt *jsrt) phpArrHas(av interface{}, k interface{}) bool {
	if keys, vals, ok := phpArrParts(av); ok {
		i := rt.phpKeyIndex(keys, k)
		return i >= 0 && !isNullish(vals.elems[i])
	}
	if s, ok := av.(string); ok {
		if !phpNumericKey(k) {
			return false
		}
		i := int(rt.phpToLong(k))
		// A NEGATIVE string offset counts from the end (PHP 7.1+), so
		// isset("abc"[-1]) is true - and phpGet has always ANSWERED "c" for it,
		// so this arm made isset and the read disagree inside one expression.
		// Mirrored in languages/lib/php-rt.metajs's phArrHas.
		n := len(phpRaw(s))
		if i < 0 {
			return -i <= n
		}
		return i < n
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
		if rt.phpIsAA(av) {
			rt.memberCall(av, "offsetUnset", []interface{}{k})
		}
		return
	}
	keys, _ := o.props["keys"].(*jsArray)
	vals, _ := o.props["vals"].(*jsArray)
	i := rt.phpKeyIndex(keys, k)
	if i < 0 {
		return
	}
	phpRefAdjust(phpCellOf(vals.elems[i]), -1)
	keys.dropIdx()
	vals.dropIdx()
	keys.elems = append(keys.elems[:i], keys.elems[i+1:]...)
	vals.elems = append(vals.elems[:i], vals.elems[i+1:]...)
	if p := phpArrPos(o); p > float64(i) {
		o.set("__pos", p-1)
	}
}

// phpElemCell is the storage CELL of an array element, created on demand - the twin
// of php-interpreter.abnf's arrCell. It must answer the SAME cell for the same
// element every time: building a fresh one per '&' silently detaches every alias
// taken earlier, so `$x = &$a[0]; $y = &$a[0]; $y = 9;` left $x behind.
func (rt *jsrt) phpElemCell(av interface{}, k interface{}) interface{} {
	if rt.phpIsAA(av) {
		return rt.phpAACell(av, k)
	}
	keys, vals, ok := phpArrParts(av)
	if !ok {
		return jsNull
	}
	nk := phpNormKey(k)
	i := rt.phpKeyIndex(keys, nk)
	if i < 0 {
		rt.phpArrSet(av, nk, jsNull)
		keys, vals, ok = phpArrParts(av)
		if !ok {
			return jsNull
		}
		i = rt.phpKeyIndex(keys, nk)
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

// phpAACell: an ArrayAccess element fetched for writing is offsetGet's answer in
// a temporary cell (zend_std_read_dimension). Twin of phAACell in
// languages/lib/php-rt.metajs and of targetCell's arm in php-interpreter.abnf.
func (rt *jsrt) phpAACell(av interface{}, k interface{}) interface{} {
	cell := newJSObject()
	cell.set("__refcell", true)
	cell.set("v", rt.memberCall(av, "offsetGet", []interface{}{k}))
	return cell
}

// phpArrCopy is PHP's value assignment: an array is copied (deeply, since a
// nested array is a value too), an object is not. An element that is a reference
// some other slot holds stays that reference in the copy; one only its own slot
// holds becomes a plain value (zend_array_dup_element).
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
		if c := phpCellOf(e); c != nil && phpRefHolders(c) > 1 {
			phpRefAdjust(c, 1)
			ov.elems = append(ov.elems, c)
			continue
		}
		ov.elems = append(ov.elems, rt.phpArrCopy(phpDeref(e)))
	}
	if o, isO := phpAsArr(v); isO {
		out.set("__next", phpArrNext(o))
		if _, has := o.props["__pos"]; has {
			out.set("__pos", phpArrPos(o))
		}
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
		return float64(len(phpRaw(s)))
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
			return rt.phpObjArr(o, false, nil)
		}
		// A Closure or a Generator is an object with no properties.
		if phpIsGenOrClosure(v) {
			return out
		}
		rt.phpArrPush(out, v)
		return out
	case "object":
		// An array's elements become the properties of a stdClass, any other
		// non-object the one property "scalar". Twin of phpCast in
		// php-interpreter.abnf.
		if phpIsObj(v) || phpIsGenOrClosure(v) {
			return v
		}
		o := newJSObject()
		if cls := rt.phpRtClass("stdClass"); cls != nil {
			o.set("__class", cls)
		}
		rt.phpStamp(o)
		if keys, vals, ok := phpArrParts(v); ok {
			for i := range keys.elems {
				o.set(phpPropKey(rt.phpStr(keys.elems[i])), rt.phpArrCopy(phpDeref(vals.elems[i])))
			}
		} else if !isNullish(v) {
			o.set("scalar", v)
		}
		return o
	case "unset":
		return jsNull
	}
	return v
}

// phpValsOf backs array_values().
func (rt *jsrt) phpValsOf(v interface{}) interface{} {
	out := phpNewArr()
	if _, vals, ok := phpArrParts(v); ok {
		for _, e := range vals.elems {
			rt.phpArrPush(out, phpDeref(e))
		}
	}
	return out
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

// Whether the method owner holds under key is static: owner's __mst lists its
// static methods lower-cased. Twin of phStaticIn in languages/lib/php-rt.metajs.
func phpStaticIn(owner *jsObject, key string) bool {
	l, ok := owner.props["__mst"].(string)
	return ok && strings.Contains(l, ","+phpCase(key, false)+",")
}

// phpCallFallback says where a '::' call of a method cls has none of, or may not
// call, goes (get_static_method_fallback): to the __call of this - the frame's
// $this, or the one a callback binds - when cls has a __call and this is an
// instance of cls; false leaves it to __callStatic. Twin of phCallFallback in
// languages/lib/php-rt.metajs and phpCallFallback in php-interpreter.abnf.
func (rt *jsrt) phpCallFallback(cls *jsObject, this interface{}) bool {
	return phpIsObj(this) && phpOwner(cls, "__call") != nil && rt.phpIsA(this, rt.phpStr(cls.props["__name"]))
}

// phpCallVia runs this's __call for the method name, with the PHP array args.
func (rt *jsrt) phpCallVia(this interface{}, name string, args interface{}) interface{} {
	o := this.(*jsObject)
	cls, _ := o.props["__class"].(*jsObject)
	inner := &jsArray{}
	if _, vals, ok := phpArrParts(args); ok {
		inner.elems = append(inner.elems, vals.elems...)
	}
	return rt.call(phpOwner(cls, "__call").props["__call"], jsUndef, []interface{}{o, name, inner})
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

// phpConst is C::NAME; base is the class as the source wrote it, or null when
// the source named it through a value.
func (rt *jsrt) phpConst(target interface{}, name string, base interface{}) interface{} {
	cls := rt.phpClassOf(target)
	shown := rt.phpStr(cls.props["__name"])
	if !isNullish(base) {
		shown = rt.phpStr(base)
	}
	if v, ok := rt.phpConstIn(cls, name, shown+"::"+name, 0); ok {
		return v
	}
	rt.fail("undefined constant %s::%s", rt.phpStr(cls.props["__name"]), name)
	return nil
}

// phpConstIn walks a class's own chain and then its interfaces: PHP inherits a
// constant from a base class AND from every implemented interface.
//
// A declared constant is a {__lazyconst: closure} box until its first read (see
// emitConstDecl in php-to-llvm-ir.abnf): the closure computes it, called with a
// null $this, and the value replaces the box on the declaring class. A box read
// from inside another constant's initializer is marked __visiting while it runs,
// and reading a marked box is the self-referencing-constant Error naming shown;
// constFind in php-interpreter.abnf explains the rule.
func (rt *jsrt) phpConstIn(cls *jsObject, name, shown string, depth int) (interface{}, bool) {
	if cls == nil || depth > 16 {
		return nil, false
	}
	key := phpConstPfx + name
	if o := phpOwner(cls, key); o != nil {
		v := o.props[key]
		if box, isO := v.(*jsObject); isO {
			if f, lazy := box.props["__lazyconst"]; lazy {
				if box.props["__visiting"] == true {
					rt.phpRaise("Error", "Cannot declare self-referencing constant "+shown)
				}
				if phpConstDepth > 0 {
					box.set("__visiting", true)
				}
				phpConstDepth++
				func() {
					defer func() {
						phpConstDepth--
						box.set("__visiting", false)
					}()
					v = rt.call(f, jsUndef, []interface{}{jsNull})
				}()
				o.set(key, v)
			}
		}
		return v, true
	}
	for k := cls; k != nil; {
		if ifs, has := k.props["__ifaces"]; has {
			if _, vals, ok := phpArrParts(ifs); ok {
				for _, iv := range vals.elems {
					if io, isO := iv.(*jsObject); isO {
						if v, found := rt.phpConstIn(io, name, shown, depth+1); found {
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

// phpClassInit runs a class's __sinit (its constants, then its static property
// defaults; classInitBody in php-to-llvm-ir.abnf) the first time the class is
// used, its parent's first. A failure leaves it to run again on the next use.
// Twin of phClassInit in languages/lib/php-rt.metajs and classInit in
// php-interpreter.abnf.
func (rt *jsrt) phpClassInit(cls *jsObject) {
	if cls.props["__ready"] == true {
		return
	}
	if sup, ok := cls.props["__super"].(*jsObject); ok {
		rt.phpClassInit(sup)
	}
	if f, has := cls.props["__sinit"]; has && isCallable(f) {
		rt.call(f, jsUndef, []interface{}{jsNull})
		cls.set("__sinit", jsNull)
	}
	cls.set("__ready", true)
}

func (rt *jsrt) phpStaticGet(target interface{}, name string) interface{} {
	cls := rt.phpClassOf(target)
	rt.phpClassInit(cls)
	if o := phpOwner(cls, phpStaticPfx+name); o != nil {
		return o.props[phpStaticPfx+name]
	}
	return jsNull
}

// phpStaticTypePfx keys a typed static property's declared type on the class
// that declares it; such a property starts uninitialised (undefined).
const phpStaticTypePfx = "t:"

func (rt *jsrt) phpStaticSet(target interface{}, name string, v interface{}) {
	cls := rt.phpClassOf(target)
	rt.phpClassInit(cls)
	o := phpOwner(cls, phpStaticPfx+name)
	if o == nil {
		o = cls // First write declares it here.
	}
	if t, typed := o.props[phpStaticTypePfx+name]; typed {
		v = rt.phpSlotFit(o, name, rt.phpStr(t), v)
	}
	o.set(phpStaticPfx+name, v)
}

// phpIsA is the runtime type test behind instanceof and a typed catch: a name
// matches the class itself, any base class, and any implemented interface (which
// may itself extend others).
// phpSameName: two class names denote the same class when they agree ignoring
// ASCII case. Twin of sameClassName in php-interpreter.abnf.
func phpSameName(a, b string) bool {
	return a == b || (len(a) == len(b) && phpCase(a, false) == phpCase(b, false))
}

func (rt *jsrt) phpIsA(v interface{}, name string) bool {
	// A generator IS an instance of PHP's Generator class. It is not a *jsObject, so
	// it carries no descriptor to walk - the name is answered directly, which also
	// gives the compiler a safe "is this a generator" probe for foreach.
	if _, isGen := v.(*jsGenerator); isGen {
		return phpCoreClassIs("generator", name)
	}
	// A closure is an instance of PHP's final class Closure.
	if isCallable(v) {
		return phpCoreClassIs("closure", name)
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
		if phpSameName(rt.phpStr(cls.props["__name"]), name) {
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
					} else if in := rt.phpStr(iv); phpSameName(in, name) || phpCoreIfaceExtends(in, name) {
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
	rt.phpStamp(out)
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

// phpHandles counts the object handles a run has handed out: every object takes
// the next one when it is made (js_phoid), stored in its hidden __oid slot, and
// that is var_dump's #id. A closure literal takes one too, kept in its record
// (js_phclo). PHP
// also hands a freed handle to the next object, which this model, without
// refcounts, cannot see. Twin of objHandles in php-interpreter.abnf.
var phpHandles = map[*jsrt]int{}

// phpStamp gives o the next handle; a nil o only counts one.
func (rt *jsrt) phpStamp(o *jsObject) {
	n := phpHandles[rt] + 1
	phpHandles[rt] = n
	if o != nil {
		o.set("__oid", float64(n))
	}
}

// phpDumpID is o's handle; an object made where nothing stamped it takes the
// next one now.
func (rt *jsrt) phpDumpID(o *jsObject) int {
	if n, ok := o.props["__oid"].(float64); ok {
		return int(n)
	}
	rt.phpStamp(o)
	return phpHandles[rt]
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
	if phpIsObj(v) || phpIsGenOrClosure(v) {
		return "object"
	}
	return "unknown type"
}

// phpIsGenOrClosure: an instance of PHP's final class Generator or Closure, which
// carries no class descriptor here.
func phpIsGenOrClosure(v interface{}) bool {
	if _, isGen := v.(*jsGenerator); isGen {
		return true
	}
	return isCallable(v)
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
	if _, isGen := v.(*jsGenerator); isGen {
		return "Generator"
	}
	if isCallable(v) {
		return "Closure"
	}
	return "unknown"
}

// phpValueName is zend_zval_value_name, the "X given" of PHP 8.3's argument
// TypeErrors: a bool is named by its value ("true given"), anything else as
// get_debug_type does. Twin of phpValueName in php-interpreter.abnf and
// phValueName in lib/php-rt.metajs.
func (rt *jsrt) phpValueName(v interface{}) string {
	if b, ok := v.(bool); ok {
		return strconv.FormatBool(b)
	}
	return rt.phpDebugType(v)
}

// phpFullName is the name get_class and ::class answer: an anonymous class's
// carries the "\0file:line$n" suffix every display leaves off (anonSuffix in
// php-to-llvm-ir.abnf).
func (rt *jsrt) phpFullName(cls *jsObject) string {
	name := rt.phpStr(cls.props["__name"])
	if sfx, has := cls.props["__anon"].(string); has {
		name += sfx
	}
	return name
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
		return pad + "string(" + strconv.Itoa(len(phpRaw(t))) + ") \"" + t + "\"\n"
	}
	if keys, vals, ok := phpArrParts(v); ok {
		out := pad + "array(" + strconv.Itoa(len(keys.elems)) + ") {\n"
		for i := range keys.elems {
			out += rt.phpDumpKey(keys.elems[i], ind+2)
			out += rt.phpDump(phpDeref(vals.elems[i]), ind+2)
		}
		return out + pad + "}\n"
	}
	if phpIsObj(v) || phpLitRec(v) != nil {
		for _, d := range phpDumping[rt] {
			if d == v {
				return pad + "*RECURSION*\n"
			}
		}
		phpDumping[rt] = append(phpDumping[rt], v)
		out := rt.phpDumpObj(v, ind, pad)
		phpDumping[rt] = phpDumping[rt][:len(phpDumping[rt])-1]
		return out
	}
	return pad + "NULL\n"
}

// phpDumping holds the objects (and closures) var_dump is inside of: one reached
// again prints *RECURSION* (php_var_dump's Z_IS_RECURSIVE). Twin of dumping in
// php-interpreter.abnf and phDumping in lib/php-rt.metajs.
var phpDumping = map[*jsrt][]interface{}{}

func (rt *jsrt) phpDumpObj(v interface{}, ind int, pad string) string {
	if phpIsObj(v) {
		o := v.(*jsObject)
		cls, _ := o.props["__class"].(*jsObject)
		// The count leaves out an uninitialised typed property, which is then listed
		// as uninitialized(type); an emptied untyped one is not listed (php_var_dump).
		names := []string{}
		cnt := 0
		for _, k := range o.keys {
			if !phpHiddenProp(k) {
				if o.props[k] != jsUndef {
					cnt++
				} else if _, _, t := rt.phpPropDecl(cls, phpPropName(k)); t == "" {
					continue
				}
				names = append(names, k)
			}
		}
		out := pad + "object(" + rt.phpClassName(v) + ")#" + strconv.Itoa(rt.phpDumpID(o)) +
			" (" + strconv.Itoa(cnt) + ") {\n"
		for _, k := range names {
			name, vis := rt.phpKeyVis(cls, k)
			switch vis {
			case "":
				out += pad + "  [\"" + name + "\"]=>\n"
			case "*":
				out += pad + "  [\"" + name + "\":protected]=>\n"
			default:
				out += pad + "  [\"" + name + "\":\"" + vis + "\":private]=>\n"
			}
			if o.props[k] == jsUndef {
				_, _, t := rt.phpPropDecl(cls, name)
				out += pad + "  uninitialized(" + t + ")\n"
				continue
			}
			out += rt.phpDump(phpDeref(o.props[k]), ind+2)
		}
		return out + pad + "}\n"
	}
	rec := phpLitRec(v)
	d := rt.phpStr(rec.props["d"])
	env, _ := rec.props["e"].(*jsScope)
	var sd []string
	for _, n := range phpCloStatics(d) {
		var sv interface{} = jsNull
		if env != nil {
			if x, ok := env.get(n); ok {
				sv = x
			}
		}
		sd = append(sd, rt.phpDump(phpDeref(sv), ind+4))
	}
	this := ""
	if t := rec.props["t"]; t != jsUndef && t != jsNull && t != nil {
		this = rt.phpDump(t, ind+2)
	}
	return phpCloDump(d, int(rec.props["i"].(float64)), ind, sd, this)
}

// phpSigOf is the parameter list a function value's meta records (withSig in
// php-to-llvm-ir.abnf), which a closure literal keeps in its record (js_phclo).
// Twin of phSigOf in lib/php-rt.metajs.
func (rt *jsrt) phpSigOf(f interface{}) string {
	mi := jsvFnMeta(f)
	if mi.rec != nil {
		return rt.phpStr(mi.rec.props["s"])
	}
	return mi.name
}

// phpPropKey is the key property name is stored under on an object: one
// starting "__" behind a \x01, apart from the runtime's own __ slots, which a
// listing hides. phpPropName takes it back off. Twins: propKey in
// php-to-llvm-ir.abnf, phPropKey / phPropName in lib/php-rt.metajs.
func phpPropKey(name string) string {
	if strings.HasPrefix(name, "__") {
		return "\x01" + name
	}
	return name
}

// A base class's private property is invisible outside that class, so a write
// from there makes a separate dynamic property of the same name (PHP keys a
// private one by its class). That one is kept under phpShadowKey, which no
// property key can start with, and is listed as the public name. Twin of
// PROP_SHADOW in languages/php-interpreter.abnf.
func phpShadowKey(name string) string { return "\x02" + name }

// phpKeyVis is the property name an object key lists under and its visibility.
func (rt *jsrt) phpKeyVis(cls *jsObject, key string) (string, string) {
	if strings.HasPrefix(key, "\x02") {
		return key[1:], ""
	}
	name := phpPropName(key)
	return name, rt.phpVisOf(cls, name)
}

func phpPropName(key string) string {
	if strings.HasPrefix(key, "\x01__") {
		return key[1:]
	}
	return key
}

// phpVisOf is a property's visibility on an instance of cls: "" public, "*"
// protected, or the declaring class of a private one. Twin of phpVisOf in
// languages/php-interpreter.abnf.
func (rt *jsrt) phpVisOf(cls *jsObject, name string) string {
	_, vis, _ := rt.phpPropDecl(cls, name)
	return vis
}

// phpPropDecl is the nearest declaration of an instance property along the class
// chain, off the ",name=V:T," entries of each __pvis: the declaring class (nil for
// an undeclared one), its visibility and its declared type. Twin of phpPropDecl in
// languages/php-interpreter.abnf.
func (rt *jsrt) phpPropDecl(cls *jsObject, name string) (*jsObject, string, string) {
	key := "," + name + "="
	for cls != nil {
		if l, ok := cls.props["__pvis"]; ok {
			s := rt.phpStr(l)
			if at := strings.Index(s, key); at >= 0 {
				rest := s[at+len(key):]
				rest = rest[:strings.IndexByte(rest, ',')]
				c := strings.IndexByte(rest, ':')
				return cls, rest[:c], rest[c+1:]
			}
		}
		cls, _ = cls.props["__super"].(*jsObject)
	}
	return nil, "", ""
}

// phpPropAccess: may code in class scope (nil outside every class) touch property
// name of o? 0 yes, 1 no, 2 it is a private of a base class, which the scope
// cannot see at all and PHP treats as undeclared. Twin of phpPropAccess in
// languages/php-interpreter.abnf.
// A protected property is judged by its first declaration up the chain (the
// prototype), and a scope reaches its own private property on an instance of a
// subclass that redeclares the name (ZEND_ACC_CHANGED).
func (rt *jsrt) phpPropAccess(cls *jsObject, name string, scope *jsObject) int {
	k, vis, _ := rt.phpPropDecl(cls, name)
	if k == nil || vis == "" || k == scope {
		return 0
	}
	if scope != nil {
		scopeName := rt.phpStr(scope.props["__name"])
		if sk, svis, _ := rt.phpPropDecl(scope, name); sk == scope && svis == scopeName && rt.phpIsA(cls, scopeName) {
			return 0
		}
	}
	if vis != "*" {
		if k == cls {
			return 1
		}
		return 2
	}
	root := k
	for {
		sup, _ := root.props["__super"].(*jsObject)
		up, upVis, _ := rt.phpPropDecl(sup, name)
		if up == nil || upVis != "*" {
			break
		}
		root = up
	}
	if scope != nil && (rt.phpIsA(root, rt.phpStr(scope.props["__name"])) || rt.phpIsA(scope, rt.phpStr(root.props["__name"]))) {
		return 0
	}
	return 1
}

func (rt *jsrt) phpBadAccess(cls *jsObject, name string) {
	w := "private"
	if rt.phpVisOf(cls, name) == "*" {
		w = "protected"
	}
	rt.phpRaise("Error", "Cannot access "+w+" property "+rt.phpStr(cls.props["__name"])+"::$"+name)
}

// phpStrictTypes: the program declares strict_types=1 (js_phstrict).
var phpStrictTypes = map[*jsrt]bool{}

// phpTypeAlts is the alternatives of a declared type: '?T' is T|null, and
// '(A&B)|C' splits at the top-level bars only. Twin of phpTypeAlts in
// languages/php-interpreter.abnf.
func phpTypeAlts(typ string) []string {
	var out []string
	t := typ
	if strings.HasPrefix(t, "?") {
		out = append(out, "null")
		t = t[1:]
	}
	depth, from := 0, 0
	for i := 0; i <= len(t); i++ {
		ch := byte('|')
		if i < len(t) {
			ch = t[i]
		}
		switch {
		case ch == '(':
			depth++
		case ch == ')':
			depth--
		case ch == '|' && depth == 0:
			a := t[from:i]
			if strings.HasPrefix(a, "(") {
				a = a[1 : len(a)-1]
			}
			out = append(out, a)
			from = i + 1
		}
	}
	return out
}

// phpTypeIs: does v belong to the single (non-union) type a, with no coercion?
// self is the class that declared the type. Twin of phpTypeIs in
// languages/php-interpreter.abnf.
func (rt *jsrt) phpTypeIs(a string, v interface{}, self *jsObject) bool {
	if amp := strings.IndexByte(a, '&'); amp >= 0 {
		return rt.phpTypeIs(a[:amp], v, self) && rt.phpTypeIs(a[amp+1:], v, self)
	}
	switch phpCase(a, false) {
	case "mixed":
		return true
	case "null":
		return isNullish(v)
	case "int":
		return phpIsInt(v)
	case "float":
		return phpIsFlo(v)
	case "string":
		_, ok := v.(string)
		return ok
	case "bool":
		_, ok := v.(bool)
		return ok
	case "false":
		return v == false
	case "true":
		return v == true
	case "array":
		return phpIsArr(v)
	case "iterable":
		return phpIsArr(v) || rt.phpIsA(v, "Traversable")
	case "object":
		return phpIsObj(v) || phpIsGenOrClosure(v)
	case "self":
		return rt.phpIsA(v, rt.phpStr(self.props["__name"]))
	case "parent":
		sup, ok := self.props["__super"].(*jsObject)
		return ok && rt.phpIsA(v, rt.phpStr(sup.props["__name"]))
	}
	return rt.phpIsA(v, a)
}

// phpTypeFit is the value a slot of declared type typ holds once v is written
// to it, and false when v neither belongs to the type nor coerces to it. Twin of
// phpTypeFit in languages/php-interpreter.abnf, which states the rules.
func (rt *jsrt) phpTypeFit(typ string, v interface{}, self *jsObject) (interface{}, bool) {
	mask := ""
	for _, a := range phpTypeAlts(typ) {
		if rt.phpTypeIs(a, v, self) {
			return v, true
		}
		switch lc := phpCase(a, false); lc {
		case "int", "float", "string", "bool":
			mask += lc[:1]
		}
	}
	if isNullish(v) || phpIsArr(v) {
		return nil, false
	}
	has := func(c string) bool { return strings.Contains(mask, c) }
	if phpStrictTypes[rt] {
		if has("f") && phpIsInt(v) {
			return phpMkFlo(phpNumOf(v)), true
		}
		return nil, false
	}
	s, isStr := v.(string)
	b, isBool := v.(bool)
	if has("i") {
		if has("f") && isStr {
			if n := phpNumStrVal(s); n != nil {
				return n, true
			}
		} else if n := rt.phpWeakInt(v); n != nil {
			return n, true
		}
	}
	if has("f") {
		switch {
		case phpIsInt(v):
			return phpMkFlo(phpNumOf(v)), true
		case isBool:
			if b {
				return phpMkFlo(1), true
			}
			return phpMkFlo(0), true
		case isStr:
			if n := phpNumStrVal(s); n != nil {
				return phpMkFlo(phpNumOf(n)), true
			}
		}
	}
	if has("s") {
		if phpIsNum(v) || isBool || (phpIsObj(v) && rt.phpMethodOf(v, "__toString") != nil) {
			return rt.phpStr(v), true
		}
	}
	if has("b") && (phpIsNum(v) || isStr) {
		return rt.phpTest(v), true
	}
	return nil, false
}

// phpWeakInt is zend_parse_arg_long_weak: a float that fits (its fraction
// dropped), a numeric string read the same way, a bool as 0 or 1; nil for
// anything else. Twin of phpWeakInt in languages/php-interpreter.abnf.
func (rt *jsrt) phpWeakInt(v interface{}) interface{} {
	f := v
	if s, ok := v.(string); ok {
		f = phpNumStrVal(s)
	}
	switch t := f.(type) {
	case bool:
		if t {
			return float64(1)
		}
		return float64(0)
	case jsPhpFlo:
		lim := math.Pow(2, 63)
		if t.f != t.f || t.f >= lim || t.f < -lim {
			return nil
		}
		return rt.phpCast("int", t)
	}
	if phpIsInt(f) {
		return f
	}
	return nil
}

// phpTypeText is a declared type as PHP prints it (zend_type_to_string). Twin
// of phpTypeText in languages/php-interpreter.abnf.
func phpTypeText(typ string) string {
	alts := phpTypeAlts(typ)
	order := []string{"object", "array", "string", "int", "float", "bool", "false", "true"}
	var out []string
	hasNull := false
	for _, a := range alts {
		lc := phpCase(a, false)
		if lc == "null" {
			hasNull = true
			continue
		}
		builtin := false
		for _, o := range order {
			builtin = builtin || o == lc
		}
		if !builtin {
			if strings.Contains(a, "&") && len(alts) > 1 {
				a = "(" + a + ")"
			}
			out = append(out, a)
		}
	}
	for _, o := range order {
		for _, a := range alts {
			if phpCase(a, false) == o {
				out = append(out, o)
			}
		}
	}
	s := strings.Join(out, "|")
	if !hasNull {
		return s
	}
	if len(out) == 1 && !strings.Contains(s, "&") && phpCase(s, false) != "mixed" {
		return "?" + s
	}
	if len(out) == 0 {
		return "null"
	}
	return s + "|null"
}

// phpPropFit: v as the declared type of property name on an instance of cls
// takes it, or PHP's TypeError. Twin of phpPropFit in
// languages/php-interpreter.abnf.
func (rt *jsrt) phpPropFit(cls *jsObject, name string, v interface{}) interface{} {
	k, _, typ := rt.phpPropDecl(cls, name)
	if k == nil || typ == "" {
		return v
	}
	return rt.phpSlotFit(k, name, typ, v)
}

// phpSlotFit checks v against the type of property name declared by k, instance
// or static.
func (rt *jsrt) phpSlotFit(k *jsObject, name, typ string, v interface{}) interface{} {
	r, ok := rt.phpTypeFit(typ, v, k)
	if !ok {
		rt.phpRaise("TypeError", "Cannot assign "+rt.phpValueName(v)+" to property "+rt.phpStr(k.props["__name"])+"::$"+name+" of type "+phpTypeText(typ))
	}
	return r
}

// phpMethodVisOf is a class's own entry for method lc in its __mvis
// (",name=V,": "*" protected, "-" private), "" for none.
func (rt *jsrt) phpMethodVisOf(k *jsObject, lc string) string {
	if k == nil {
		return ""
	}
	l, ok := k.props["__mvis"]
	if !ok {
		return ""
	}
	s, key := rt.phpStr(l), ","+lc+"="
	if at := strings.Index(s, key); at >= 0 {
		return s[at+len(key) : at+len(key)+1]
	}
	return ""
}

// phpMethodDecl is the nearest declaration of method lc along the class chain,
// off each class's __mlc: the declaring class (nil for none) and its visibility.
// Twin of phpMethodDecl in languages/php-interpreter.abnf.
func (rt *jsrt) phpMethodDecl(cls *jsObject, lc string) (*jsObject, string) {
	key := "," + lc + ","
	for cls != nil {
		if l, ok := cls.props["__mlc"]; ok && strings.Contains(rt.phpStr(l), key) {
			return cls, rt.phpMethodVisOf(cls, lc)
		}
		cls, _ = cls.props["__super"].(*jsObject)
	}
	return nil, ""
}

// phpNonStatic is "Class::name" - the declaring class and the method as declared,
// off its __mns - when the nearest declaration of method lc is not static, else
// "". Twin of phpNonStatic in languages/php-interpreter.abnf.
func (rt *jsrt) phpNonStatic(cls *jsObject, lc string) string {
	k, _ := rt.phpMethodDecl(cls, lc)
	if k == nil {
		return ""
	}
	l, ok := k.props["__mns"]
	if !ok {
		return ""
	}
	s, key := rt.phpStr(l), ","+lc+":"
	at := strings.Index(s, key)
	if at < 0 {
		return ""
	}
	rest := s[at+len(key):]
	return rt.phpStr(k.props["__name"]) + "::" + rest[:strings.Index(rest, ",")]
}

// phpMethodAccess: may code in class scope (nil outside every class) call method
// lc on cls? 0 yes, 1 no (private), 2 no (protected), 3 call the scope's own
// private method instead. Twin of phpMethodAccess in
// languages/php-interpreter.abnf, which explains the rules.
func (rt *jsrt) phpMethodAccess(cls *jsObject, lc string, scope *jsObject, inst bool) int {
	k, vis := rt.phpMethodDecl(cls, lc)
	if k == nil {
		return 0
	}
	if inst && scope != nil && k != scope && rt.phpMethodVisOf(scope, lc) == "-" && rt.phpIsA(cls, rt.phpStr(scope.props["__name"])) {
		return 3
	}
	if vis == "" || k == scope {
		return 0
	}
	if vis == "-" {
		return 1
	}
	root := k
	if lc != "__construct" {
		for {
			sup, _ := root.props["__super"].(*jsObject)
			up, upVis := rt.phpMethodDecl(sup, lc)
			if up == nil || upVis == "-" {
				break
			}
			root = up
		}
	}
	if scope != nil && (rt.phpIsA(root, rt.phpStr(scope.props["__name"])) || rt.phpIsA(scope, rt.phpStr(root.props["__name"]))) {
		return 0
	}
	return 2
}

// phpBadMethod raises zend_bad_method_call's Error, or zend_bad_constructor_call's
// when ctor. Twin of phpBadMethod in languages/php-interpreter.abnf.
func (rt *jsrt) phpBadMethod(cls *jsObject, name string, scope *jsObject, ctor bool) {
	k, vis := rt.phpMethodDecl(cls, phpCase(name, false))
	w, from := "protected", "global scope"
	if vis == "-" {
		w = "private"
	}
	if !ctor {
		w += " method"
	}
	if scope != nil {
		from = "scope " + rt.phpStr(scope.props["__name"])
	}
	rt.phpRaise("Error", "Call to "+w+" "+rt.phpStr(k.props["__name"])+"::"+name+"() from "+from)
}

// phpMethodVis checks a call of method name on target (an object, or a class for
// a static call) from class scope: where undefined is a call, false a `new`, a
// string the callback check it words. A method the scope may not call raises,
// unless the magic method that stands in for a missing one is there. Twin of
// js_phmvis in languages/lib/php-rt.metajs.
func (rt *jsrt) phpMethodVis(target interface{}, name string, scope *jsObject, where interface{}) {
	// A Closure's methods are public.
	if isCallable(target) {
		return
	}
	cls := rt.phpClassOf(target)
	inst := phpIsObj(target)
	acc := rt.phpMethodAccess(cls, phpCase(name, false), scope, inst)
	if acc != 1 && acc != 2 {
		return
	}
	if wh, ok := where.(string); ok {
		magic := "__callStatic"
		if inst {
			magic = "__call"
		}
		if rt.phpMethodOf(target, magic) != nil {
			return
		}
		w := "protected"
		if acc == 1 {
			w = "private"
		}
		rt.phpCbFail(wh, "cannot access "+w+" method "+rt.phpStr(cls.props["__name"])+"::"+name+"()")
	}
	if b, ok := where.(bool); ok && !b {
		rt.phpBadMethod(cls, name, scope, true)
	}
	magic := "__callstatic"
	if inst {
		magic = "__call"
	}
	if phpOwner(cls, magic) == nil {
		rt.phpBadMethod(cls, name, scope, false)
	}
}

// phpInUn: did unset() empty this typed property (o's __un list)? Such a one is
// offered to the magic methods; one never initialised is not.
func (rt *jsrt) phpInUn(o *jsObject, name string) bool {
	l, ok := o.props["__un"].(string)
	return ok && strings.Contains(l, ","+name+",")
}

// phpMagicCall runs one magic method under zend_get_property_guard's guard: while
// __get (g), __set (s), __isset (i) or __unset (u) runs for one property of one
// object, the same access inside it reaches the plain property. The __mg slot
// lists the running ones. Twin of phpMagicCall in languages/php-interpreter.abnf.
func (rt *jsrt) phpMagicCall(o *jsObject, kind, name string, args ...interface{}) (interface{}, bool) {
	m := map[string]string{"g": "__get", "s": "__set", "i": "__isset", "u": "__unset"}[kind]
	cls := o.props["__class"]
	if rt.phpFindMethod(cls, m) == nil {
		return nil, false
	}
	saved, _ := o.props["__mg"].(string)
	if saved == "" {
		saved = ","
	}
	if strings.Contains(saved, ","+kind+name+",") {
		return nil, false
	}
	o.set("__mg", saved+kind+name+",")
	defer o.set("__mg", saved)
	return rt.memberCall(o, m, args), true
}

// phpStaticVisOf is the visibility class k gives its own static property name,
// off its __svis list: "" public, "*" protected, else the class (private).
func (rt *jsrt) phpStaticVisOf(k *jsObject, name string) string {
	if k == nil {
		return ""
	}
	s, _ := k.props["__svis"].(string)
	key := "," + name + "="
	at := strings.Index(s, key)
	if at < 0 {
		return ""
	}
	rest := s[at+len(key):]
	return rest[:strings.IndexByte(rest, ',')]
}

// phpObjDel removes a key from an object, as js_del does.
func phpObjDel(o *jsObject, key string) {
	if _, had := o.props[key]; !had {
		return
	}
	delete(o.props, key)
	for i, k := range o.keys {
		if k == key {
			o.keys = append(o.keys[:i], o.keys[i+1:]...)
			return
		}
	}
}

// phpInstance is o as a class instance, nil for anything else.
func phpInstance(v interface{}) (*jsObject, *jsObject) {
	o, ok := v.(*jsObject)
	if !ok || !phpIsObj(o) {
		return nil, nil
	}
	cls, _ := o.props["__class"].(*jsObject)
	if cls == nil {
		return nil, nil
	}
	return o, cls
}

// phpPropRead is $o->p past the emitter's plain read (js_phprop): visibility from
// scope (true when the emitter already knows the access is allowed), the
// uninitialised typed property, and the guarded __isset / __get fallbacks. isMode
// is the read of '??' (BP_VAR_IS). Twin of phpGetProp / phpPropMiss in
// languages/php-interpreter.abnf.
func (rt *jsrt) phpPropRead(ov interface{}, key string, scopeV interface{}, isMode bool) interface{} {
	o, cls := phpInstance(ov)
	if o == nil {
		return phpDeref(rt.getMember(ov, key))
	}
	name := phpPropName(key)
	c := 0
	if scopeV != true {
		scope, _ := scopeV.(*jsObject)
		c = rt.phpPropAccess(cls, name, scope)
	}
	var d *jsObject
	typ := ""
	if c == 0 {
		if v := phpDeref(o.props[key]); v != nil && v != jsUndef {
			return v
		}
		d, _, typ = rt.phpPropDecl(cls, name)
	}
	if c == 2 {
		if v, has := o.props[phpShadowKey(name)]; has {
			return phpDeref(v)
		}
	}
	typed := d != nil && typ != ""
	uninit := func() {
		rt.phpRaise("Error", "Typed property "+rt.phpStr(d.props["__name"])+"::$"+name+" must not be accessed before initialization")
	}
	if typed && !rt.phpInUn(o, name) {
		if isMode {
			return jsNull
		}
		uninit()
	}
	hasGet := rt.phpFindMethod(cls, "__get") != nil
	if c == 1 && !hasGet && !isMode {
		rt.phpBadAccess(cls, name)
	}
	if isMode {
		if r, ran := rt.phpMagicCall(o, "i", name, name); ran {
			if !rt.phpTest(r) {
				return jsNull
			}
			// A property __isset created is read rather than asked of __get (GH-12695).
			if v := phpDeref(o.props[key]); c == 0 && v != nil && v != jsUndef {
				return v
			}
		}
	}
	if hasGet {
		if r, ran := rt.phpMagicCall(o, "g", name, name); ran {
			return r
		}
		if c == 1 {
			rt.phpBadAccess(cls, name)
		}
	}
	if typed && !isMode {
		uninit()
	}
	return jsNull
}

// phpObjArr is an object's properties as an array: a non-public name mangled
// the way (array) does, or, for foreach (iter), only those visible from class
// scope (nil outside every class). Twin of phpObjArr in
// languages/php-interpreter.abnf.
func (rt *jsrt) phpObjArr(o *jsObject, iter bool, scope *jsObject) *jsObject {
	out := phpNewArr()
	cls, _ := o.props["__class"].(*jsObject)
	for _, k := range o.keys {
		if phpHiddenProp(k) {
			continue
		}
		// An uninitialised typed property is not listed.
		if o.props[k] == jsUndef {
			continue
		}
		name, vis := rt.phpKeyVis(cls, k)
		switch {
		case vis == "" || (iter && rt.phpVisible(o, vis, scope)):
			rt.phpArrSet(out, name, o.props[k])
		case !iter:
			rt.phpArrSet(out, "\x00"+vis+"\x00"+name, o.props[k])
		}
	}
	return out
}

// phpVisible answers whether code in class scope reads a property of o whose
// visibility is vis.
func (rt *jsrt) phpVisible(o *jsObject, vis string, scope *jsObject) bool {
	if scope == nil {
		return false
	}
	if vis == "*" {
		return rt.phpIsA(o, rt.phpStr(scope.props["__name"])) || rt.phpIsA(scope, rt.phpClassName(o))
	}
	return rt.phpStr(scope.props["__name"]) == vis
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
		f, ok := phpStrtod(text)
		if !ok {
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

// realPath is realpath(3): the canonical absolute path of an existing file, with
// ".", ".." and symbolic links resolved against the working directory, or "" when
// it does not resolve - a missing component, a non-directory followed by more
// path, a symlink loop, a NUL byte. An empty path is the working directory. It is
// js_realpath here and c.realpath for the interpreters; twin of js_realpath in
// languages/lib/runtime.c.
func realPath(p string) string {
	if strings.IndexByte(p, 0) >= 0 {
		return ""
	}
	if !strings.HasPrefix(p, "/") {
		wd, err := os.Getwd()
		if err != nil {
			return ""
		}
		p = wd + "/" + p
	}
	res := ""
	rest := strings.Split(p, "/")
	links := 0
	for len(rest) > 0 {
		c := rest[0]
		rest = rest[1:]
		if c == "" || c == "." {
			continue
		}
		if c == ".." {
			if i := strings.LastIndexByte(res, '/'); i >= 0 {
				res = res[:i]
			}
			continue
		}
		next := res + "/" + c
		fi, err := os.Lstat(next)
		if err != nil {
			return ""
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			links++
			t, err := os.Readlink(next)
			if err != nil || links > 32 {
				return ""
			}
			if strings.HasPrefix(t, "/") {
				res = ""
			}
			rest = append(strings.Split(t, "/"), rest...)
			continue
		}
		if !fi.IsDir() && len(rest) > 0 {
			return ""
		}
		res = next
	}
	if res == "" {
		return "/"
	}
	return res
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
		m["js_realpath"] = func(a []uint64) uint64 {
			if r := realPath(rt.toString(u(a[0]))); r != "" {
				return rt.wrapStr(r)
			}
			return jsHFalse
		}
		// A slot that held old now holds cell (either may be no cell; undefined for
		// an unset): the holder counts phpArrCopy reads.
		m["js_phrebind"] = func(a []uint64) uint64 {
			co, cn := phpCellOf(u(a[0])), phpCellOf(u(a[1]))
			if co != cn {
				phpRefAdjust(co, -1)
				phpRefAdjust(cn, 1)
			}
			return 0
		}
		m["js_phrtinit"] = func(a []uint64) uint64 {
			phpMainScope[rt] = a[0]
			return 0
		}
		// ----- the two RECOVERABLE-ERROR arms ---------------------------------------
		//
		// js_phmcall is php's own name for the method dispatch, and it exists
		// BECAUSE of chapter 4's rxExtraExterns trap: this map is filled once for
		// every language, so overriding m["js_mcall"] here would make a C# or java
		// method miss raise a PHP Error. The layer-2 twin is js_phmcall in
		// languages/lib/php-rt.metajs and the two texts are byte for byte the same.
		m["js_phmcall"] = func(a []uint64) uint64 {
			recv := u(a[0])
			name := rt.phpStr(u(a[1]))
			args, isArr := u(a[2]).(*jsArray)
			if !isArr {
				rt.fail("js_phmcall args must be an array")
			}
			if _, isGen := recv.(*jsGenerator); isGen {
				// The shared rt.memberCall aborts on a generator method nothing
				// claims, and this is php's own name precisely so it can raise
				// instead. The member test is the same one memberCall makes.
				if !isCallable(rt.getMember(recv, name)) {
					rt.phpRaise("Error", "Call to undefined method Generator::"+name+"()")
				}
				return w(rt.memberCall(recv, name, args.elems))
			}
			if o, isObj := recv.(*jsObject); isObj {
				if cls, has := o.props["__class"]; has && !isNullish(cls) {
					if rt.phpFindMethod(cls, name) == nil {
						// Method names are case-insensitive: a class also holds each
						// method under its lower-cased name (js_phmset), which a
						// miss retries.
						if lc := phpCase(name, false); lc != name && rt.phpFindMethod(cls, lc) != nil {
							return w(rt.memberCall(recv, lc, args.elems))
						}
						// A method the class does not declare is its __call's.
						if c, ok := cls.(*jsObject); ok && phpOwner(c, "__call") != nil {
							inner := phpNewArr()
							for _, e := range args.elems {
								rt.phpArrPush(inner, e)
							}
							return w(rt.memberCall(recv, "__call", []interface{}{name, inner}))
						}
						// PHP 8, and CATCHABLE: `catch (Error $e)` recovers.
						rt.phpRaise("Error", "Call to undefined method "+rt.phpClassName(recv)+"::"+name+"()")
					}
					return w(rt.memberCall(recv, name, args.elems))
				}
				if f, has := o.props[name]; has && isCallable(f) {
					return w(rt.memberCall(recv, name, args.elems))
				}
			}
			if isCallable(recv) {
				return w(rt.phpCloCall(recv, name, args.elems))
			}
			// A string, an int, null, an array - everything PHP will not let you
			// call a method on. phpDebugType spells the type as PHP 8 does.
			rt.phpRaise("Error", "Call to a member function "+name+"() on "+rt.phpDebugType(recv))
			return 0
		}
		// js_phmcallv(recv, name, args, scope): $o->m() for a method name some class
		// restricts, judged from class scope: the scope's own private method when
		// the instance redeclares it, __call or an Error for one it may not call,
		// js_phmcall otherwise. Twin of js_phmcallv in languages/lib/php-rt.metajs.
		m["js_phmcallv"] = func(a []uint64) uint64 {
			if o, ok := u(a[0]).(*jsObject); ok {
				if cls, ok := o.props["__class"].(*jsObject); ok {
					name := rt.phpStr(u(a[1]))
					lc := phpCase(name, false)
					scope, _ := u(a[3]).(*jsObject)
					args, _ := u(a[2]).(*jsArray)
					switch rt.phpMethodAccess(cls, lc, scope, true) {
					case 3:
						return w(rt.call(scope.props[lc], jsUndef, append([]interface{}{o}, args.elems...)))
					case 1, 2:
						if phpOwner(cls, "__call") == nil {
							rt.phpBadMethod(cls, name, scope, false)
						}
						inner := phpNewArr()
						for _, e := range args.elems {
							rt.phpArrPush(inner, e)
						}
						return m["js_phmcall"]([]uint64{a[0], rt.wrapStr("__call"), w(&jsArray{elems: []interface{}{name, inner}})})
					}
				}
			}
			return m["js_phmcall"](a[:3])
		}
		// js_phscallv(target, name, self, args, lsb, scope): C::m() for a method
		// name some class restricts: __callStatic or an Error for one the scope may
		// not call, js_phscall otherwise. Twin of js_phscallv in
		// languages/lib/php-rt.metajs.
		m["js_phscallv"] = func(a []uint64) uint64 {
			cls := rt.phpClassOf(u(a[0]))
			name := rt.phpStr(u(a[1]))
			scope, _ := u(a[5]).(*jsObject)
			if acc := rt.phpMethodAccess(cls, phpCase(name, false), scope, false); acc == 1 || acc == 2 {
				// An inaccessible method falls back as a missing one does.
				if rt.phpCallFallback(cls, u(a[2])) {
					return w(rt.phpCallVia(u(a[2]), name, u(a[3])))
				}
				if phpOwner(cls, "__callstatic") == nil {
					rt.phpBadMethod(cls, name, scope, false)
				}
				fwd := phpNewArr()
				rt.phpArrPush(fwd, name)
				rt.phpArrPush(fwd, u(a[3]))
				return m["js_phscall"]([]uint64{a[0], rt.wrapStr("__callstatic"), a[2], w(fwd), a[4]})
			}
			return m["js_phscall"](a[:5])
		}
		// js_phnscheck(target, name, this, scope): C::m() or C::m(...) of a
		// non-static method needs a $this that is an instance of C; scope, when
		// given, defers a method it may not call to js_phscallv. Twin of
		// js_phnscheck in languages/lib/php-rt.metajs.
		m["js_phnscheck"] = func(a []uint64) uint64 {
			cls := rt.phpClassOf(u(a[0]))
			lc := phpCase(rt.phpStr(u(a[1])), false)
			if u(a[3]) != jsUndef {
				scope, _ := u(a[3]).(*jsObject)
				if acc := rt.phpMethodAccess(cls, lc, scope, false); acc == 1 || acc == 2 {
					return jsHUndefined
				}
			}
			if ns := rt.phpNonStatic(cls, lc); ns != "" && !(phpIsObj(u(a[2])) && rt.phpIsA(u(a[2]), rt.phpStr(cls.props["__name"]))) {
				rt.phpRaise("Error", "Non-static method "+ns+"() cannot be called statically")
			}
			return jsHUndefined
		}
		// js_phcbns(where, cls, cn, mn, this, scope): the non-static rule for a
		// method named through its class cn (zend_is_callable_check_class). A
		// callback binds the frame's $this when the frame's class scope is an
		// instance of cls, and a relative name forwards any $this. where is the
		// callback check's argument, which raises its TypeError for a method
		// nothing binds; true answers the $this a callback binds (undefined for
		// none); false is a DYNAMIC call ('C::m'(), ['C', 'm']()), which binds none
		// and so raises PHP's Error for a non-static method, and for a missing one
		// that __call would take for this (get_static_method_fallback). Twin of
		// js_phcbns in languages/lib/php-rt.metajs and cbNonStatic / cbThis /
		// dynNonStatic in php-interpreter.abnf.
		m["js_phcbns"] = func(a []uint64) uint64 {
			cls, ok := u(a[1]).(*jsObject)
			if !ok {
				return jsHUndefined
			}
			this := u(a[4])
			switch phpCase(strings.TrimPrefix(rt.phpStr(u(a[2])), "\\"), false) {
			case "self", "parent", "static":
				if phpIsObj(this) || u(a[0]) == false || u(a[0]) == true {
					return jsHUndefined
				}
			}
			mn := rt.phpStr(u(a[3]))
			if u(a[0]) == false {
				ns := ""
				scope, _ := u(a[5]).(*jsObject)
				if rt.phpMethodOf(cls, mn) != nil && rt.phpMethodAccess(cls, phpCase(mn, false), scope, false) == 0 {
					ns = rt.phpNonStatic(cls, phpCase(mn, false))
				} else if o, ok := this.(*jsObject); ok && rt.phpCallFallback(cls, this) {
					if k, _ := rt.phpMethodDecl(o.props["__class"].(*jsObject), "__call"); k != nil {
						ns = rt.phpStr(k.props["__name"]) + "::" + mn
					}
				}
				if ns != "" {
					rt.phpRaise("Error", "Non-static method "+ns+"() cannot be called statically")
				}
				return jsHUndefined
			}
			ns := rt.phpNonStatic(cls, phpCase(mn, false))
			// The frame's $this, compatible as above, is bound to a static method too:
			// its class is then the called scope.
			compat := false
			if scope, ok := u(a[5]).(*jsObject); ok && phpIsObj(this) && rt.phpIsA(this, rt.phpStr(scope.props["__name"])) &&
				rt.phpIsA(scope, rt.phpStr(cls.props["__name"])) {
				compat = true
			}
			if u(a[0]) == true {
				if compat {
					return a[4]
				}
				return jsHUndefined
			}
			if ns == "" || compat {
				return jsHUndefined
			}
			rt.phpCbFail(rt.phpStr(u(a[0])), "non-static method "+rt.phpStr(cls.props["__name"])+"::"+ns[strings.Index(ns, "::")+2:]+"() cannot be called statically")
			return jsHUndefined
		}
		// js_phfccdecl(env, key, recv, name): js_scope_decl of the receiver a
		// method callable is bound to, which must answer name where the callable is
		// made: an instance its method or __call, a class its method or
		// __callStatic. A generator and a closure are left to the call. name
		// undefined is the key's tail after its ':'. Twin of js_phfccdecl in
		// languages/lib/php-rt.metajs.
		m["js_phfccdecl"] = func(a []uint64) uint64 {
			recv := u(a[2])
			key := rt.toString(u(a[1]))
			rt.scopeOf(a[0]).put(key, recv)
			name := key[strings.Index(key, ":")+1:]
			if u(a[3]) != jsUndef {
				name = rt.phpStr(u(a[3]))
			}
			if o, ok := recv.(*jsObject); ok {
				if _, isCls := o.props["__isclass"]; isCls {
					if rt.phpMethodOf(o, name) == nil && rt.phpMethodOf(o, "__callstatic") == nil {
						rt.phpRaise("Error", "Call to undefined method "+rt.phpStr(o.props["__name"])+"::"+name+"()")
					}
					return 0
				}
				if cls, has := o.props["__class"]; has && !isNullish(cls) {
					if rt.phpMethodOf(o, name) == nil && rt.phpMethodOf(o, "__call") == nil {
						rt.phpRaise("Error", "Call to undefined method "+rt.phpClassName(recv)+"::"+name+"()")
					}
					return 0
				}
				if f, has := o.props[name]; has && isCallable(f) {
					return 0
				}
			} else if _, isGen := recv.(*jsGenerator); isGen || isCallable(recv) {
				return 0
			}
			rt.phpRaise("Error", "Call to a member function "+name+"() on "+rt.phpDebugType(recv))
			return 0
		}
		m["js_phmvis"] = func(a []uint64) uint64 {
			scope, _ := u(a[2]).(*jsObject)
			rt.phpMethodVis(u(a[0]), rt.phpStr(u(a[1])), scope, u(a[3]))
			return jsHUndefined
		}
		// js_phcbhead(v): the class name a 'C::m' or ['C', 'm'] callable starts
		// with, else null. Twin of js_phcbhead in languages/lib/php-rt.metajs.
		m["js_phcbhead"] = func(a []uint64) uint64 {
			v := u(a[0])
			if s, ok := v.(string); ok {
				if i := strings.Index(s, "::"); i >= 0 {
					return rt.wrapStr(s[:i])
				}
			} else if phpIsArr(v) && rt.phpLen(v) == 2 {
				if s, ok := rt.phpArrGet(v, float64(0)).(string); ok {
					return rt.wrapStr(s)
				}
			}
			return jsHNull
		}
		// js_phcbvis(v, cls, scope, where): a method callback some class restricts,
		// on cls (the class a named callable denotes) or the object of [$obj, 'm'].
		// With where a string it is the callback check, raising its TypeError for a
		// method scope may not call; otherwise it answers whether scope may call
		// one, i.e. the callable must be bound in scope. Twin of js_phcbvis in
		// languages/lib/php-rt.metajs and visBound / cbVis in php-interpreter.abnf.
		m["js_phcbvis"] = func(a []uint64) uint64 {
			v := u(a[0])
			var target, mn interface{}
			if s, ok := v.(string); ok {
				if i := strings.Index(s, "::"); i >= 0 {
					mn = s[i+2:]
				}
			} else if phpIsArr(v) && rt.phpLen(v) == 2 {
				target, mn = rt.phpArrGet(v, float64(0)), rt.phpArrGet(v, float64(1))
			}
			if c, ok := u(a[1]).(*jsObject); ok {
				target = c
			}
			name, ok := mn.(string)
			isCls := false
			if o, isO := target.(*jsObject); isO {
				_, isCls = o.props["__isclass"]
			}
			if !ok || !(phpIsObj(target) || isCls) {
				return jsHFalse
			}
			cls := rt.phpClassOf(target)
			lc := phpCase(name, false)
			k, vis := rt.phpMethodDecl(cls, lc)
			if k == nil {
				return jsHFalse
			}
			scope, _ := u(a[2]).(*jsObject)
			if _, isStr := u(a[3]).(string); isStr {
				rt.phpMethodVis(target, name, scope, u(a[3]))
				return jsHFalse
			}
			acc := rt.phpMethodAccess(cls, lc, scope, phpIsObj(target))
			return boolH(acc == 3 || (acc == 0 && vis != ""))
		}
		// The emitter's door to phpRaise: a message it words at compile time.
		m["js_phraise"] = func(a []uint64) uint64 {
			rt.phpRaise(rt.phpStr(u(a[0])), rt.phpStr(u(a[1])))
			return 0
		}
		// The floor's js_rt_missvar hook, php half. Twin of js_phmissname in
		// languages/lib/php-rt.metajs.
		m["js_phmissname"] = func(a []uint64) uint64 {
			n := rt.phpStr(u(a[0]))
			if n == "" || n[0] == '$' {
				return w(jsUndef)
			}
			o := rt.phpMkErr("Error", "Call to undefined function "+n+"()")
			if o == nil {
				return w(jsUndef)
			}
			return w(o)
		}
		// (f, i) -> is f's parameter i by reference: bit i of the mask the emitter
		// recorded with js_jsfnmeta (withRefMask in php-to-llvm-ir.abnf). Twin of
		// js_phrefat in languages/lib/php-rt.metajs.
		m["js_phrefat"] = func(a []uint64) uint64 {
			i := int(rt.toNumber(u(a[1])))
			f := u(a[0])
			// An object called as a function takes its __invoke's.
			if _, isObj := f.(*jsObject); isObj {
				f = rt.phpMethodOf(f, "__invoke")
			}
			return boolH((jsvFnMeta(f).length>>uint(i))&1 == 1)
		}
		// (target, name, i) -> is parameter i of the method `name` of target (an
		// object, or a class for a static call) by reference; false when there is
		// no such method. Twin of js_phmrefat in languages/lib/php-rt.metajs.
		m["js_phmrefat"] = func(a []uint64) uint64 {
			meth := rt.phpMethodOf(u(a[0]), u(a[1]))
			if meth == nil {
				return jsHFalse
			}
			return boolH((jsvFnMeta(meth).length>>uint(int(rt.toNumber(u(a[2])))))&1 == 1)
		}
		// An object called as a function places its arguments for its __invoke.
		m["js_phplace"] = func(a []uint64) uint64 {
			f := u(a[0])
			if _, isObj := f.(*jsObject); isObj {
				f = rt.phpMethodOf(f, "__invoke")
			}
			return w(rt.phpPlaceArgs(rt.phpSigOf(f), jsvFnMeta(f).length, u(a[1])))
		}
		// A non-string callee: true for an array callback (which must have two
		// elements), false for a value the call runs as it is (a closure, an object
		// with __invoke), and PHP 8's Error for anything else. With where (a
		// callback check, php_cbcheck) a failure is that argument's TypeError
		// instead, an [$obj, 'm'] pair has its method checked, and true means
		// ['C', 'm'], whose class the caller looks up. Twin of js_phcbarr in
		// languages/lib/php-rt.metajs.
		m["js_phcbarr"] = func(a []uint64) uint64 {
			v := u(a[0])
			where, check := u(a[1]).(string)
			if isCallable(v) {
				return jsHFalse
			}
			if phpIsArr(v) {
				if !check {
					if rt.phpLen(v) != 2 {
						rt.phpRaise("Error", "Array callback must have exactly two elements")
					}
					return jsHTrue
				}
				if rt.phpLen(v) != 2 {
					rt.phpCbFail(where, "array must have exactly two members")
				}
				recv := rt.phpArrGet(v, float64(0))
				_, isCls := recv.(string)
				if !isCls && !phpIsObj(recv) && !isCallable(recv) {
					rt.phpCbFail(where, "first array member is not a valid class name or object")
				}
				mn, isStr := rt.phpArrGet(v, float64(1)).(string)
				if !isStr {
					rt.phpCbFail(where, "second array member is not a valid method")
				}
				if isCallable(recv) {
					if !phpCloHas(mn) {
						rt.phpCbFail(where, "class Closure does not have a method \""+mn+"\"")
					}
					return jsHFalse
				}
				if isCls {
					return jsHTrue
				}
				rt.phpCbMeth(where, recv, mn)
				return jsHFalse
			}
			if phpIsObj(v) {
				if rt.phpMethodOf(v, "__invoke") == nil {
					if check {
						rt.phpCbFail(where, "no array or string given")
					}
					rt.phpRaise("Error", "Object of type "+rt.phpDebugType(v)+" is not callable")
				}
				return jsHFalse
			}
			if t := rt.phpDebugType(v); t != "unknown" {
				if check {
					rt.phpCbFail(where, "no array or string given")
				}
				rt.phpRaise("Error", "Value of type "+t+" is not callable")
			}
			return jsHFalse
		}
		// count($value): an array's element count, a Countable's count(); anything
		// else is a TypeError. Twin of js_phcount in languages/lib/php-rt.metajs and
		// phpCount in php-interpreter.abnf.
		m["js_phcount"] = func(a []uint64) uint64 {
			v := u(a[0])
			if keys, _, ok := phpArrParts(v); ok {
				return rt.wrapNum(float64(len(keys.elems)))
			}
			if phpIsObj(v) && rt.phpIsA(v, "Countable") {
				return w(rt.memberCall(v, "count", nil))
			}
			rt.phpRaise("TypeError", "count(): Argument #1 ($value) must be of type Countable|array, "+rt.phpValueName(v)+" given")
			return jsHUndefined
		}
		// js_phrelcb(v, self, static, cls, this): a library function's callback named RELATIVE
		// to the class scope ('self::m', ['parent', 'm'], 'static::m'), resolved
		// against the caller's frame classes. undefined when the class answers the
		// method (the caller binds it), the absolute "C::m" when it does not (the
		// library's own check reports on that), v itself when it is no relative
		// callable or the caller has no class scope. A non-static method named
		// outright through cls (the class its name denotes, which the caller looks
		// up) answers undefined too when the frame's this binds it (js_phcbns).
		// Twin of js_phrelcb in languages/lib/php-rt.metajs and libCallable in
		// php-interpreter.abnf.
		m["js_phrelcb"] = func(a []uint64) uint64 {
			v := u(a[0])
			var cn, mn interface{}
			if s, ok := v.(string); ok && strings.Contains(s, "::") {
				i := strings.Index(s, "::")
				cn, mn = s[:i], s[i+2:]
			} else if phpIsArr(v) && rt.phpLen(v) == 2 {
				cn, mn = rt.phpArrGet(v, float64(0)), rt.phpArrGet(v, float64(1))
			}
			c, ok1 := cn.(string)
			mname, ok2 := mn.(string)
			self, ok3 := u(a[1]).(*jsObject)
			if !ok1 || !ok2 || !ok3 {
				return a[0]
			}
			cls := self
			switch phpCase(c, false) {
			case "self", "static", "parent":
			default:
				if m["js_phcbns"]([]uint64{jsHTrue, a[3], w(c), w(mname), a[4], a[1]}) != jsHUndefined {
					return jsHUndefined
				}
				return a[0]
			}
			switch phpCase(c, false) {
			case "self":
			case "static":
				if st, ok := u(a[2]).(*jsObject); ok {
					cls = st
				}
			case "parent":
				sup, ok := self.props["__super"].(*jsObject)
				if !ok {
					return a[0]
				}
				cls = sup
			default:
				return a[0]
			}
			if rt.phpMethodOf(cls, mname) != nil || rt.phpMethodOf(cls, "__callStatic") != nil {
				return jsHUndefined
			}
			return rt.wrapStr(rt.phpStr(cls.props["__name"]) + "::" + mname)
		}
		m["js_phcbfail"] = func(a []uint64) uint64 {
			rt.phpCbFail(rt.phpStr(u(a[0])), rt.phpStr(u(a[1])))
			return jsHUndefined
		}
		// js_phcbcls(where, cls, mn, cn, this, scope, form): the method half of
		// the check of a callback naming mn through its class cn: a method cls
		// lacks must go somewhere (phpCbFallback). Twin of js_phcbcls in
		// languages/lib/php-rt.metajs.
		m["js_phcbcls"] = func(a []uint64) uint64 {
			cls, _ := u(a[1]).(*jsObject)
			if cls == nil || rt.phpMethodOf(cls, u(a[2])) != nil {
				return jsHUndefined
			}
			if _, ok := rt.phpCbFallback(cls, rt.phpStr(u(a[3])), int(rt.toNumber(u(a[6]))), u(a[4]), u(a[5])); !ok {
				rt.phpCbFail(rt.phpStr(u(a[0])), "class "+rt.phpStr(cls.props["__name"])+" does not have a method \""+rt.phpStr(u(a[2]))+"\"")
			}
			return jsHUndefined
		}
		// js_phcbfb(cls, cn, mn, this, scope, form): where a callback's method
		// cls lacks goes: the class whose __call runs on this, false for cls's
		// __callStatic, undefined when cls has the method (or nothing takes it).
		// Twin of js_phcbfb in languages/lib/php-rt.metajs.
		m["js_phcbfb"] = func(a []uint64) uint64 {
			cls, _ := u(a[0]).(*jsObject)
			if cls == nil || rt.phpMethodOf(cls, u(a[2])) != nil {
				return jsHUndefined
			}
			via, ok := rt.phpCbFallback(cls, rt.phpStr(u(a[1])), int(rt.toNumber(u(a[5]))), u(a[3]), u(a[4]))
			if !ok {
				return jsHUndefined
			}
			if via == nil {
				return jsHFalse
			}
			return w(via)
		}
		// The class a callable's class NAME forwards as the late-static-binding
		// class: the caller's own for self / parent / static, none for a class named
		// outright. Twin of js_phfwd in languages/lib/php-rt.metajs.
		m["js_phfwd"] = func(a []uint64) uint64 {
			switch phpCase(rt.phpStr(u(a[0])), false) {
			case "self", "parent", "static":
				return w(rt.phpCalledScope(u(a[1]), u(a[2])))
			}
			return jsHUndefined
		}
		// (clo, target, name) -> clo, carrying the meta of the method `name` of
		// target (an object, or a class for a static method): a closure that
		// dispatches to a method places its arguments as the method would. Twin of
		// js_phbindsig in languages/lib/php-rt.metajs.
		// (recv, name) -> recv for a closure and __invoke, whose callable form
		// is the closure itself (zend_closure_from_frame,
		// zend_create_closure_from_callable); undefined otherwise. Twin of
		// js_phcloinv in languages/lib/php-rt.metajs.
		m["js_phcloinv"] = func(a []uint64) uint64 {
			if isCallable(u(a[0])) && phpCase(rt.phpStr(u(a[1])), false) == "__invoke" {
				return a[0]
			}
			return jsHUndefined
		}
		// (recv, name, this, factory) -> the record of the closure PHP makes from a
		// callable (zend_create_fake_closure): the method name that recv (an
		// object, or a class) answers, or the __call / __callStatic standing in
		// for it. It keeps the method's signature and by-reference mask (mk), the
		// method's class sc, whether it is static, the $this it is bound to -
		// recv's, or this for a static dispatch of a non-static method - and the
		// emitted factory phpFakeBind makes a rebound one with. Twin of js_phfake
		// in languages/lib/php-rt.metajs and fakeClosure in php-interpreter.abnf.
		m["js_phfake"] = func(a []uint64) uint64 {
			recv := u(a[0])
			cls, _ := recv.(*jsObject)
			if phpIsObj(recv) {
				cls, _ = cls.props["__class"].(*jsObject)
			}
			nm := rt.phpStr(u(a[1]))
			lc := phpCase(nm, false)
			meth := rt.phpMethodOf(recv, nm)
			k, _ := rt.phpMethodDecl(cls, lc)
			st := false
			if k == nil {
				st = !phpIsObj(recv) || phpOwner(cls, "__call") == nil
				magic := "__call"
				if st {
					magic = "__callstatic"
				}
				k, _ = rt.phpMethodDecl(cls, magic)
			} else {
				st = phpStaticIn(k, lc)
			}
			if k == nil {
				k = cls
			}
			var t interface{} = jsUndef
			if !st {
				t = u(a[2])
				if phpIsObj(recv) {
					t = recv
				}
			}
			sig := ""
			if meth != nil {
				sig = rt.phpSigOf(meth)
			}
			rec := newJSObject()
			rec.set("fk", true)
			rec.set("s", sig)
			rec.set("mk", float64(jsvFnMeta(meth).length))
			rec.set("n", nm)
			rec.set("sc", k)
			rec.set("st", st)
			rec.set("t", t)
			rec.set("f", u(a[3]))
			return w(rec)
		}
		// (src, this) -> the record of the closure the factory makes from the one
		// src, bound to this. Twin of js_phfakerec in languages/lib/php-rt.metajs.
		m["js_phfakerec"] = func(a []uint64) uint64 {
			src := phpFakeRec(u(a[0]))
			rec := newJSObject()
			for _, k := range []string{"fk", "s", "mk", "n", "sc", "st", "f"} {
				rec.set(k, src.props[k])
			}
			rec.set("t", u(a[1]))
			return w(rec)
		}
		m["js_phbindsig"] = func(a []uint64) uint64 {
			if meth := rt.phpMethodOf(u(a[1]), u(a[2])); meth != nil {
				jsvFnMetaSet(u(a[0]), rt.phpSigOf(meth), jsvFnMeta(meth).length)
			}
			return a[0]
		}
		// (target, name, args) -> args placed for the method `name` of target, for
		// a call site that cannot know which class's method it reaches. Twin of
		// js_phmplace in languages/lib/php-rt.metajs.
		m["js_phmplace"] = func(a []uint64) uint64 {
			meth := rt.phpMethodOf(u(a[0]), u(a[1]))
			return w(rt.phpPlaceArgs(rt.phpSigOf(meth), jsvFnMeta(meth).length, u(a[2])))
		}
		// (name, self, static) -> the class a run-time class name the scope did not
		// declare denotes: self / static / parent through the frame's classes (null
		// outside a class), else PHP 8's catchable Error. Twin of resolveClassName in
		// php-interpreter.abnf and of js_phclassmiss in languages/lib/php-rt.metajs.
		m["js_phclassmiss"] = func(a []uint64) uint64 {
			name := rt.phpStr(u(a[0]))
			where, check := u(a[3]).(string)
			// The three relative names are case-insensitive like every class name.
			if lc := phpCase(name, false); lc == "self" || lc == "static" || lc == "parent" {
				name = lc
				cls := u(a[1])
				if name == "static" {
					cls = u(a[2])
				}
				o, ok := cls.(*jsObject)
				if !ok {
					if check {
						rt.phpCbFail(where, "cannot access \""+name+"\" when no class scope is active")
					}
					rt.phpRaise("Error", "Cannot use \""+name+"\" when no class scope is active")
				}
				if name != "parent" {
					return w(o)
				}
				if sup, has := o.props["__super"]; has && sup != nil && sup != jsUndef {
					return w(sup)
				}
				if check {
					rt.phpCbFail(where, "cannot access \"parent\" when current class scope has no parent")
				}
				rt.phpRaise("Error", "Cannot use \"parent\" when current class scope has no parent")
			}
			if check {
				rt.phpCbFail(where, "class \""+name+"\" not found")
			}
			rt.phpRaise("Error", "Class \""+name+"\" not found")
			return 0
		}
		// `new C`: the class is initialized before its first instance (phpClassInit).
		m["js_phsinit"] = func(a []uint64) uint64 {
			rt.phpClassInit(rt.phpClassOf(u(a[0])))
			return 0
		}
		m["js_phplacesig"] = func(a []uint64) uint64 {
			return w(rt.phpPlaceArgs(rt.phpStr(u(a[0])), int(rt.toNumber(u(a[2]))), u(a[1])))
		}
		// The four that REPLACE a jsrt.go entry: each is exact 64 bit where the
		// shared one is 32 bit or double based.
		m["js_phneg"] = func(a []uint64) uint64 { return w(rt.phpNeg(u(a[0]))) }
		// ~: a string's bytes or an int's bits; any other operand, null and a
		// bool included, is a TypeError. Twin of phpBitNot in php-interpreter.abnf.
		m["js_phbnot"] = func(a []uint64) uint64 {
			v := u(a[0])
			if s, ok := v.(string); ok {
				return w(phpBitOp("~", s, s))
			}
			_, isBool := v.(bool)
			if v == jsUndef || v == jsNull || isBool || phpNoNum(v) {
				rt.phpRaise("TypeError", "Cannot perform bitwise not on "+rt.phpValueName(v))
			}
			return w(phpMkInt(^rt.phpToLong(v)))
		}
		m["js_phinc"] = func(a []uint64) uint64 { return w(rt.phpIncDec(u(a[0]), true)) }
		m["js_phdec"] = func(a []uint64) uint64 { return w(rt.phpIncDec(u(a[0]), false)) }
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
		// abs($num): a scalar coerces as arithmetic does; an array, an object or a
		// string with no numeric prefix is a TypeError. Twin of js_phabs in
		// languages/lib/php-rt.metajs and phpBuiltins.abs in php-interpreter.abnf.
		m["js_phabs"] = func(a []uint64) uint64 {
			bad := false
			switch t := u(a[0]).(type) {
			case float64, jsGInt, jsPhpFlo, bool, jsUndefT, jsNullT:
			case string:
				_, _, _, ok := phpNumScan(t)
				bad = !ok
			default:
				bad = true
			}
			if bad {
				rt.phpRaise("TypeError", "abs(): Argument #1 ($num) must be of type int|float, "+rt.phpValueName(u(a[0]))+" given")
			}
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
				return boolH(phpIsObj(v) || phpIsGenOrClosure(v))
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
		// through its cell. A one-step target (`$a[0] = 1`) is vivified inline by
		// the emitter (vivRootV in php-to-llvm-ir.abnf).
		m["js_phvivroot"] = func(a []uint64) uint64 {
			if cur := u(a[0]); !isNullish(cur) {
				return a[0]
			}
			return w(phpNewArr())
		}
		m["js_phtype"] = func(a []uint64) uint64 { return rt.wrapStr(rt.phpTypeName(u(a[0]))) }
		m["js_phdtype"] = func(a []uint64) uint64 { return rt.wrapStr(rt.phpDebugType(u(a[0]))) }
		// (o, key): writing a property of anything but an object raises PHP 8's
		// catchable Error. Twin of js_phsetchk in languages/lib/php-rt.metajs.
		m["js_phsetchk"] = func(a []uint64) uint64 {
			if o := u(a[0]); !phpIsObj(o) {
				rt.phpRaise("Error", "Attempt to assign property \""+phpPropName(rt.phpStr(u(a[1])))+"\" on "+rt.phpDebugType(o))
			}
			return 0
		}
		m["js_phdump"] = func(a []uint64) uint64 { return rt.wrapStr(rt.phpDump(u(a[0]), 0)) }
		// The slow halves of a property access, for a name some class restricts or
		// types and for a miss with a magic method about: (o, key, scope[, ...]),
		// scope being the running class (null outside one) or true for an access the
		// emitter knows is allowed. Twins in languages/lib/php-rt.metajs; the
		// interpreter's are phpGetProp, phpSetProp, phpPropIsset and phpUnsetProp.
		m["js_phprop"] = func(a []uint64) uint64 {
			return w(rt.phpPropRead(u(a[0]), rt.phpStr(u(a[1])), u(a[2]), u(a[3]) == true))
		}
		m["js_phpropset"] = func(a []uint64) uint64 {
			ov, key, val := u(a[0]), rt.phpStr(u(a[1])), u(a[3])
			o, cls := phpInstance(ov)
			if o == nil {
				rt.setMember(ov, key, val)
				return 0
			}
			name := phpPropName(key)
			c := 0
			if sv := u(a[2]); sv != true {
				scope, _ := sv.(*jsObject)
				c = rt.phpPropAccess(cls, name, scope)
			}
			if c == 1 {
				if _, ran := rt.phpMagicCall(o, "s", name, name, val); !ran {
					rt.phpBadAccess(cls, name)
				}
				return 0
			}
			if c == 2 {
				key = phpShadowKey(name)
			}
			raw := o.props[key]
			// __set is for a property the instance does not have. An uninitialised
			// typed one it has, and the write initialises it; one unset() emptied
			// is offered to __set again.
			if raw == jsUndef || raw == nil {
				if _, has := o.props[key]; !has || rt.phpInUn(o, name) {
					if _, ran := rt.phpMagicCall(o, "s", name, name, val); ran {
						return 0
					}
				}
			}
			if c == 0 {
				val = rt.phpPropFit(cls, name, val)
			}
			if cell := phpCellOf(raw); cell != nil {
				cell.set("v", val)
			} else {
				o.set(key, val)
			}
			return 0
		}
		// js_phptyped(o, key, v): v as the declared type of property key of o
		// takes it, or PHP's TypeError. Twin of js_phptyped in
		// languages/lib/php-rt.metajs and phpPropFit in php-interpreter.abnf.
		m["js_phptyped"] = func(a []uint64) uint64 {
			if o, cls := phpInstance(u(a[0])); o != nil {
				return w(rt.phpPropFit(cls, phpPropName(rt.phpStr(u(a[1]))), u(a[2])))
			}
			return a[2]
		}
		// declare(strict_types=1) in the program: a typed slot takes only its own
		// types, and an int for a float.
		// js_phsready(target, name, v): v, the value just read of a typed static
		// property, or PHP's Error when it is still uninitialised. Twin of
		// js_phsready in languages/lib/php-rt.metajs.
		m["js_phsready"] = func(a []uint64) uint64 {
			if v := u(a[2]); v != jsUndef && v != nil {
				return a[2]
			}
			name := rt.toString(u(a[1]))
			if o := phpOwner(rt.phpClassOf(u(a[0])), phpStaticTypePfx+name); o != nil {
				rt.phpRaise("Error", "Typed static property "+rt.phpStr(o.props["__name"])+"::$"+name+" must not be accessed before initialization")
			}
			return a[2]
		}
		m["js_phstrict"] = func(a []uint64) uint64 {
			phpStrictTypes[rt] = true
			return 0
		}
		m["js_phpropisset"] = func(a []uint64) uint64 {
			ov, key := u(a[0]), rt.phpStr(u(a[1]))
			o, cls := phpInstance(ov)
			if o == nil {
				return boolH(rt.phpArrHas(ov, key))
			}
			name := phpPropName(key)
			c := 0
			if sv := u(a[2]); sv != true {
				scope, _ := sv.(*jsObject)
				c = rt.phpPropAccess(cls, name, scope)
			}
			if c == 2 {
				key = phpShadowKey(name)
			}
			if c == 0 || c == 2 {
				raw, has := o.props[key]
				if v := phpDeref(raw); has && v != jsUndef {
					return boolH(v != jsNull)
				}
				if has && !rt.phpInUn(o, name) {
					return jsHFalse
				}
			}
			r, ran := rt.phpMagicCall(o, "i", name, name)
			return boolH(ran && rt.phpTest(r))
		}
		m["js_phpropunset"] = func(a []uint64) uint64 {
			ov, key := u(a[0]), rt.phpStr(u(a[1]))
			o, cls := phpInstance(ov)
			if o == nil {
				if obj, ok := ov.(*jsObject); ok {
					phpObjDel(obj, key)
				}
				return 0
			}
			name := phpPropName(key)
			c := 0
			if sv := u(a[2]); sv != true {
				scope, _ := sv.(*jsObject)
				c = rt.phpPropAccess(cls, name, scope)
			}
			if c == 0 {
				if raw, has := o.props[key]; has {
					_, _, typ := rt.phpPropDecl(cls, name)
					if raw != jsUndef || (typ != "" && !rt.phpInUn(o, name)) {
						if typ == "" {
							phpObjDel(o, key)
							return 0
						}
						// A typed property keeps its slot, uninitialised again.
						o.props[key] = jsUndef
						if !rt.phpInUn(o, name) {
							l, _ := o.props["__un"].(string)
							if l == "" {
								l = ","
							}
							o.set("__un", l+name+",")
						}
						return 0
					}
				}
			}
			if _, has := o.props[phpShadowKey(name)]; has && c == 2 {
				phpObjDel(o, phpShadowKey(name))
				return 0
			}
			if _, ran := rt.phpMagicCall(o, "u", name, name); !ran && c == 1 {
				rt.phpBadAccess(cls, name)
			}
			return 0
		}
		// echo v: PHP writes the string's BYTES, so a held byte goes out as that
		// one byte. Twin of phsEcho in languages/lib/php-strings.js.
		m["js_phecho"] = func(a []uint64) uint64 {
			s := rt.phpStr(u(a[0]))
			if raw := phpRaw(s); raw != s {
				outWriter.Write([]byte(raw))
			} else {
				outWriter.Write([]byte(wtf8Clean(s)))
			}
			return w(jsUndef)
		}
		// js_phclsexists(found, name, iface): class_exists / interface_exists over
		// the descriptor the name was looked up to (null when none). Twin of
		// phpClassExists in languages/php-interpreter.abnf.
		m["js_phclsexists"] = func(a []uint64) uint64 {
			o, _ := u(a[0]).(*jsObject)
			if o == nil || o.props["__isclass"] != true {
				if rt.phpTest(u(a[2])) {
					return boolH(phpCoreIface(rt.phpStr(u(a[1]))))
				}
				return boolH(phpCoreClass(rt.phpStr(u(a[1]))) != "")
			}
			if rt.phpTest(u(a[2])) {
				return boolH(o.props["__isiface"] == true)
			}
			return boolH(o.props["__isiface"] != true && o.props["__istrait"] != true)
		}
		// js_phisaof(v, found, cls, allow, onlySub): is_a / is_subclass_of, where
		// found is the descriptor a string v names (null when none). Twin of
		// phpIsAOf in languages/php-interpreter.abnf.
		m["js_phisaof"] = func(a []uint64) uint64 {
			v := u(a[0])
			name := strings.TrimPrefix(rt.phpStr(u(a[2])), "\\")
			own := rt.phpDebugType(v)
			if _, isStr := v.(string); isStr {
				cls, ok := u(a[1]).(*jsObject)
				if !rt.phpTest(u(a[3])) {
					return jsHFalse
				}
				if !ok {
					core := phpCoreClass(v.(string))
					if core == "" || (rt.phpTest(u(a[4])) && phpCoreClass(name) == core) {
						return jsHFalse
					}
					return boolH(phpCoreClassIs(core, name))
				}
				v, own = cls, rt.phpStr(cls.props["__name"])
			}
			if rt.phpTest(u(a[4])) && phpSameName(own, name) {
				return jsHFalse
			}
			return boolH(rt.phpIsA(v, name))
		}
		// js_phparent(v, found): get_parent_class of an object, of the class
		// descriptor v, or of the class a string v names (found, null when none).
		// Twin of phpParentClass in languages/php-interpreter.abnf.
		m["js_phparent"] = func(a []uint64) uint64 {
			v := u(a[0])
			var cls *jsObject
			if _, isStr := v.(string); isStr {
				cls, _ = u(a[1]).(*jsObject)
			} else if o, isO := v.(*jsObject); isO && o.props["__isclass"] == true {
				cls = o
			} else if phpIsObj(v) || phpIsGenOrClosure(v) {
				cls, _ = v.(*jsObject)
				if cls != nil {
					cls, _ = cls.props["__class"].(*jsObject)
				}
				if cls == nil {
					return jsHFalse
				}
			}
			if cls == nil {
				rt.phpRaise("TypeError", "get_parent_class(): Argument #1 ($object_or_class) must be an object or a valid class name, "+rt.phpValueName(v)+" given")
			}
			if sup, ok := cls.props["__super"].(*jsObject); ok {
				return rt.wrapStr(rt.phpStr(sup.props["__name"]))
			}
			return jsHFalse
		}
		// js_phaisset(a, k): isset($a[k]) - offsetExists for an ArrayAccess
		// object, js_phisset's answer for anything else.
		m["js_phaisset"] = func(a []uint64) uint64 {
			if rt.phpIsAA(u(a[0])) {
				return boolH(rt.phpAAIsset(u(a[0]), u(a[1])))
			}
			return boolH(rt.phpArrHas(u(a[0]), u(a[1])))
		}
		// js_phifaces(v): the interfaces the class of the object v, or the class
		// descriptor v, names itself, by declared name; false for anything else
		// (null: a string that named no class). Twin of phpIfaceNames in
		// languages/php-interpreter.abnf.
		m["js_phifaces"] = func(a []uint64) uint64 {
			v := u(a[0])
			out := phpNewArr()
			if _, isGen := v.(*jsGenerator); isGen {
				rt.phpArrPush(out, "Iterator")
				return w(out)
			}
			if isCallable(v) {
				return w(out)
			}
			cls, _ := v.(*jsObject)
			if cls != nil && cls.props["__isclass"] != true {
				if !phpIsObj(cls) {
					return jsHFalse
				}
				cls, _ = cls.props["__class"].(*jsObject)
			}
			if cls == nil {
				return jsHFalse
			}
			if _, vals, ok := phpArrParts(cls.props["__ifaces"]); ok {
				for _, iv := range vals.elems {
					if io, isO := iv.(*jsObject); isO {
						iv = io.props["__name"]
					}
					rt.phpArrPush(out, iv)
				}
			}
			return w(out)
		}
		// js_phmexists(objOrCls, name): method_exists over the __mlc lists of the
		// class chain. Twin of phpMethodExists in languages/php-interpreter.abnf.
		m["js_phmexists"] = func(a []uint64) uint64 {
			o, _ := u(a[0]).(*jsObject)
			if o != nil && o.props["__isclass"] != true {
				o, _ = o.props["__class"].(*jsObject)
			}
			key := "," + phpCase(rt.phpStr(u(a[1])), false) + ","
			for o != nil {
				if l, ok := o.props["__mlc"]; ok && strings.Contains(rt.phpStr(l), key) {
					return jsHTrue
				}
				o, _ = o.props["__super"].(*jsObject)
			}
			return jsHFalse
		}
		// js_phsvis(cls, name, scope, quiet): may code in class scope (null outside
		// one) touch static property name of cls? True when it may; otherwise false
		// when quiet (isset, '??'), else PHP's Error. Twin of phpStaticVis in
		// languages/php-interpreter.abnf.
		m["js_phsvis"] = func(a []uint64) uint64 {
			cls, _ := u(a[0]).(*jsObject)
			name := rt.phpStr(u(a[1]))
			scope, _ := u(a[2]).(*jsObject)
			key := "s:" + name
			k := cls
			for k != nil {
				if _, has := k.props[key]; has {
					break
				}
				k, _ = k.props["__super"].(*jsObject)
			}
			vis := rt.phpStaticVisOf(k, name)
			if k == nil || vis == "" || k == scope {
				return jsHTrue
			}
			if vis == "*" && scope != nil {
				root := k
				for {
					sup, _ := root.props["__super"].(*jsObject)
					for sup != nil {
						if _, has := sup.props[key]; has {
							break
						}
						sup, _ = sup.props["__super"].(*jsObject)
					}
					if sup == nil || rt.phpStaticVisOf(sup, name) != "*" {
						break
					}
					root = sup
				}
				if rt.phpIsA(root, rt.phpStr(scope.props["__name"])) || rt.phpIsA(scope, rt.phpStr(root.props["__name"])) {
					return jsHTrue
				}
			}
			if u(a[3]) == true {
				return jsHFalse
			}
			w := "private"
			if vis == "*" {
				w = "protected"
			}
			rt.phpRaise("Error", "Cannot access "+w+" property "+rt.phpStr(cls.props["__name"])+"::$"+name)
			return 0
		}
		// js_phpexists(v, found, name): property_exists - a property the class chain
		// of found (an object's class, or the class a string named) declares, static
		// or not and whatever its visibility, or one the object v has, null or not.
		// Twin of phpPropExists in languages/php-interpreter.abnf.
		m["js_phpexists"] = func(a []uint64) uint64 {
			name := rt.phpStr(u(a[2]))
			if o, ok := u(a[0]).(*jsObject); ok && phpIsObj(o) {
				if _, has := o.props[phpPropKey(name)]; has {
					return jsHTrue
				}
			}
			k, _ := u(a[1]).(*jsObject)
			if k != nil && k.props["__isclass"] != true {
				k, _ = k.props["__class"].(*jsObject)
			}
			if d, _, _ := rt.phpPropDecl(k, name); d != nil {
				return jsHTrue
			}
			for ; k != nil; k, _ = k.props["__super"].(*jsObject) {
				if _, has := k.props["s:"+name]; has {
					return jsHTrue
				}
			}
			return jsHFalse
		}
		// js_phfestep(it, first, needKey): one step of foreach's generator walk -
		// a generator's next(), or an Iterator's rewind (first) / next, valid,
		// current and, when the loop binds one, key, as the same {done, value}
		// record with the key in a {__genkv, k, v}. Twin of js_phfestep in
		// lib/php-rt.metajs and of the Iterator loop in php-interpreter.abnf.
		m["js_phfestep"] = func(a []uint64) uint64 {
			it := u(a[0])
			if _, isGen := it.(*jsGenerator); isGen {
				return w(rt.memberCall(it, "next", nil))
			}
			if rt.phpTest(u(a[1])) {
				rt.memberCall(it, "rewind", nil)
			} else {
				rt.memberCall(it, "next", nil)
			}
			st := newJSObject()
			if !rt.phpTest(rt.memberCall(it, "valid", nil)) {
				st.set("value", jsUndef)
				st.set("done", true)
				return w(st)
			}
			kv := newJSObject()
			kv.set("__genkv", true)
			kv.set("v", rt.memberCall(it, "current", nil))
			kv.set("k", jsNull)
			if rt.phpTest(u(a[2])) {
				kv.set("k", rt.memberCall(it, "key", nil))
			}
			st.set("value", kv)
			st.set("done", false)
			return w(st)
		}
		// js_phfeobj(v, scope, byref): what foreach iterates - an object's properties
		// visible from the class scope (null outside a class), by value an array
		// as it is now, anything else as is.
		m["js_phfeobj"] = func(a []uint64) uint64 {
			v := u(a[0])
			for phpIsObj(v) && rt.phpIsA(v, "IteratorAggregate") {
				v = rt.memberCall(v, "getIterator", nil)
			}
			// By value, an array is walked as it was when the loop started: the
			// body's writes to it go to the variable, not to what is iterated.
			if keys, vals, isArr := phpArrParts(v); isArr && !rt.phpTest(u(a[2])) {
				out := phpNewArr()
				ok2, _ := out.props["keys"].(*jsArray)
				ov, _ := out.props["vals"].(*jsArray)
				ok2.elems = append(ok2.elems, keys.elems...)
				ov.elems = append(ov.elems, vals.elems...)
				return w(out)
			}
			o, ok := v.(*jsObject)
			if !ok || !phpIsObj(o) || rt.phpIsA(o, "Iterator") {
				return w(v)
			}
			scope, _ := u(a[1]).(*jsObject)
			out := rt.phpObjArr(o, true, scope)
			// `as &$v` reaches each property through its own storage cell.
			if rt.phpTest(u(a[2])) {
				keys, vals, _ := phpArrParts(out)
				for i, k := range keys.elems {
					c := phpCellOf(vals.elems[i])
					if c == nil {
						c = newJSObject()
						c.set("__refcell", true)
						c.set("v", vals.elems[i])
						o.set(phpPropKey(rt.phpStr(k)), c)
					}
					vals.elems[i] = c
				}
			}
			return w(out)
		}
		m["js_phpkey"] = func(a []uint64) uint64 { return rt.wrapStr(phpPropKey(rt.phpStr(u(a[0])))) }
		m["js_phpname"] = func(a []uint64) uint64 { return rt.wrapStr(phpPropName(rt.phpStr(u(a[0])))) }
		m["js_phgetclass"] = func(a []uint64) uint64 {
			if o, ok := u(a[0]).(*jsObject); ok && phpIsObj(o) {
				if cls, isCls := o.props["__class"].(*jsObject); isCls {
					return rt.wrapStr(rt.phpFullName(cls))
				}
				return rt.wrapStr(rt.phpClassName(o))
			}
			if v := u(a[0]); phpIsGenOrClosure(v) {
				return rt.wrapStr(rt.phpDebugType(v))
			}
			rt.phpRaise("TypeError", "get_class(): Argument #1 ($object) must be of type object, "+rt.phpValueName(u(a[0]))+" given")
			return 0
		}
		m["js_phvname"] = func(a []uint64) uint64 { return rt.wrapStr(rt.phpValueName(u(a[0]))) }
		// js_phmset(cls, name, closure): a method declaration. Twin of setMethod
		// in php-interpreter.abnf, which says why every recorded spelling is set.
		m["js_phmset"] = func(a []uint64) uint64 {
			cls, ok := u(a[0]).(*jsObject)
			if !ok {
				rt.fail("js_phmset on a non-class")
			}
			name := rt.phpStr(u(a[1]))
			fn := u(a[2])
			lc := phpCase(name, false)
			key := "__ms:" + lc
			spells := ","
			for k := interface{}(cls); ; {
				ko, isObj := k.(*jsObject)
				if !isObj {
					break
				}
				if s, has := ko.props[key].(string); has {
					spells = s
					break
				}
				k = ko.props["__super"]
			}
			for _, s := range strings.Split(spells, ",") {
				if s != "" {
					cls.set(s, fn)
				}
			}
			cls.set(name, fn)
			cls.set(lc, fn)
			cls.set(key, spells+name+",")
			return 0
		}
	})
	// js_phmcall reads its argument array in position 2, exactly like the js_mcall
	// it replaces: the handle must arrive as the array itself.
	jsThroughArgs["js_phmcall"] = 1 << 2
}
