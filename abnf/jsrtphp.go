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

func phpIsNum(v interface{}) bool {
	switch v.(type) {
	case float64, jsPhpFlo:
		return true
	}
	return false
}

func phpNumOf(v interface{}) float64 {
	switch t := v.(type) {
	case float64:
		return t
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

// phpFloStr renders a float the way PHP does: one without a fractional part
// prints as an integer, so (string)3.0 is "3".
func phpFloStr(n float64) string {
	if n == phpTrunc(n) && n < 1e15 && n > -1e15 {
		return strconv.FormatFloat(phpTrunc(n), 'f', -1, 64)
	}
	return jsNumString(n)
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

func phpIsObj(v interface{}) bool {
	o, ok := v.(*jsObject)
	if !ok {
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
	f, err := strconv.ParseFloat(s[st:en], 64)
	if err != nil {
		return nil
	}
	if isF {
		return phpMkFlo(f)
	}
	return f
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
	f, err := strconv.ParseFloat(s[:i], 64)
	if err != nil {
		return float64(0)
	}
	if isF {
		return phpMkFlo(f)
	}
	return f
}

// phpToNum is the arithmetic coercion: everything becomes an int or a float box.
func (rt *jsrt) phpToNum(v interface{}) interface{} {
	switch t := v.(type) {
	case float64:
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

// phpArith: an int stays an int, and any float operand makes the result a float.
func (rt *jsrt) phpArith(op string, l, r interface{}) interface{} {
	a := rt.phpToNum(l)
	b := rt.phpToNum(r)
	f := phpIsFlo(a) || phpIsFlo(b)
	x, y := phpNumOf(a), phpNumOf(b)
	box := func(v float64) interface{} {
		if f {
			return phpMkFlo(v)
		}
		return v
	}
	switch op {
	case "+":
		return box(x + y)
	case "-":
		return box(x - y)
	case "*":
		return box(x * y)
	case "/":
		if y == 0 {
			rt.fail("division by zero")
		}
		q := x / y
		if !f && q == phpTrunc(q) {
			return q
		}
		return phpMkFlo(q)
	case "%":
		iy := phpTrunc(y)
		if iy == 0 {
			rt.fail("modulo by zero")
		}
		return math.Mod(phpTrunc(x), iy)
	case "**":
		p := math.Pow(x, y)
		if !f && y >= 0 && p == phpTrunc(p) {
			return p
		}
		return phpMkFlo(p)
	}
	rt.fail("unknown arithmetic operator %s", op)
	return nil
}

func (rt *jsrt) phpBit(op string, l, r interface{}) interface{} {
	x := int32(int64(phpTrunc(phpNumOf(rt.phpToNum(l)))))
	y := int32(int64(phpTrunc(phpNumOf(rt.phpToNum(r)))))
	switch op {
	case "&":
		return float64(x & y)
	case "|":
		return float64(x | y)
	case "^":
		return float64(x ^ y)
	case "<<":
		return float64(x << uint(uint32(y)&31))
	case ">>":
		return float64(x >> uint(uint32(y)&31))
	}
	rt.fail("unknown bit operator %s", op)
	return nil
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
			if !rt.phpIdentical(lv.elems[i], rv.elems[i]) {
				return false
			}
		}
		return true
	}
	if isNullish(l) || isNullish(r) {
		return isNullish(l) && isNullish(r)
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
	af, aIsN := a.(float64)
	bf, bIsN := b.(float64)
	if aIsN || bIsN {
		return aIsN && bIsN && af == bf
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
			if !rt.phpLoose(lv.elems[i], rv.elems[j]) {
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
			return phpNumOf(nl) == phpNumOf(nr)
		}
		return ls == rs
	}
	if lIsS && phpIsNum(r) {
		return rt.phpStrNumLoose(ls, r)
	}
	if rIsS && phpIsNum(l) {
		return rt.phpStrNumLoose(rs, l)
	}
	return phpNumOf(rt.phpToNum(l)) == phpNumOf(rt.phpToNum(r))
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
		return phpNumOf(ns) == phpNumOf(n)
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
			if cv := rt.phpCmp(lv.elems[i], rv.elems[j]); cv != 0 {
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
			return phpCmpNum(phpNumOf(nl), phpNumOf(nr))
		}
		return phpCmpStr(ls, rs)
	}
	if lIsS && phpIsNum(r) {
		x := phpNumStrVal(ls)
		if x == nil {
			return phpCmpStr(ls, rt.phpStr(r))
		}
		return phpCmpNum(phpNumOf(x), phpNumOf(r))
	}
	if rIsS && phpIsNum(l) {
		y := phpNumStrVal(rs)
		if y == nil {
			return phpCmpStr(rt.phpStr(l), rs)
		}
		return phpCmpNum(phpNumOf(l), phpNumOf(y))
	}
	return phpCmpNum(phpNumOf(rt.phpToNum(l)), phpNumOf(rt.phpToNum(r)))
}

// ----- arrays: ONE ordered map, with PHP's key normalization -----------------

// phpNormKey turns a subscript into PHP's stored key: an integer stays an
// integer, a canonical decimal string folds to one ("01" does NOT), a bool is
// 0/1, null is "" and a float truncates.
func phpNormKey(v interface{}) interface{} {
	switch t := v.(type) {
	case float64:
		return phpTrunc(t)
	case jsPhpFlo:
		return phpTrunc(t.f)
	case bool:
		if t {
			return float64(1)
		}
		return float64(0)
	case jsUndefT, jsNullT:
		return ""
	case string:
		if phpIsCanonInt(t) {
			f, err := strconv.ParseFloat(t, 64)
			if err == nil {
				return f
			}
		}
		return t
	}
	return v
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
	return len(s)-i <= 18
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
			i := int(phpTrunc(phpNumOf(rt.phpToNum(k))))
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
		vals.elems[i] = v
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
			return vals.elems[i]
		}
		return jsNull
	}
	if s, ok := av.(string); ok {
		i := int(phpTrunc(phpNumOf(rt.phpToNum(k))))
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
		i := int(phpTrunc(phpNumOf(rt.phpToNum(k))))
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
		ov.elems = append(ov.elems, rt.phpArrCopy(e))
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
		return phpTrunc(phpNumOf(rt.phpToNum(v)))
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
			rt.phpArrPush(out, e)
		}
	}
	return out
}

func (rt *jsrt) phpInArray(needle, hay interface{}) bool {
	if _, vals, ok := phpArrParts(hay); ok {
		for _, e := range vals.elems {
			if rt.phpLoose(e, needle) {
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
