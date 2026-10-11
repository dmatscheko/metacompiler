package abnf

import (
	"math"
	"strconv"
	"strings"
)

// System.String's and System.Char's members: the Go twin's port of
// languages/lib/csharp-strings.js, whose header has the rules. The strings are
// worked on as UTF-16 units, as the MetaJS hosts see them; the classification
// table is jsrtjavachars.go's and the case mapping and search the
// lib/utf16-strings.js ports in jsrtjavastr.go, as there.

func csxIsWs(c int) bool {
	return (c >= 9 && c <= 13) || c == 32 || c == 133 || c == 160 || c == 5760 || (c >= 8192 && c <= 8202) ||
		c == 8232 || c == 8233 || c == 8239 || c == 8287 || c == 12288
}

func csxIsUpper(c int) bool {
	if c < 128 {
		return c >= 65 && c <= 90
	}
	return jcClass(c)%8 == 1
}

func csxIsLower(c int) bool {
	if c < 128 {
		return c >= 97 && c <= 122
	}
	return jcClass(c)%8 == 2
}

func (rt *jsrt) csxNum(v interface{}) int { return jsToInt(rt.toNumber(v)) }

func (rt *jsrt) csxCode(v interface{}) int {
	if c, ok := v.(jsChar); ok {
		return int(c.code)
	}
	return rt.csxNum(v)
}

// csxArg is a char or string argument as UTF-16 units.
func (rt *jsrt) csxArg(v interface{}) []uint16 {
	if c, ok := v.(jsChar); ok {
		return []uint16{uint16(c.code)}
	}
	s, _ := v.(string)
	return strUnits(s)
}

func (rt *jsrt) csxCharSet(args []interface{}, k int) []int {
	src := args[k:]
	if len(args) == k+1 {
		if a, ok := args[k].(*jsArray); ok {
			src = a.elems
		}
	}
	out := make([]int, 0, len(src))
	for _, v := range src {
		out = append(out, rt.csxCode(v))
	}
	return out
}

func csxInSet(set []int, c int) bool {
	for _, x := range set {
		if x == c {
			return true
		}
	}
	return false
}

func csxTrim(u []uint16, set []int, lead, trail bool) []uint16 {
	drop := func(c uint16) bool {
		if len(set) == 0 {
			return csxIsWs(int(c))
		}
		return csxInSet(set, int(c))
	}
	b, e := 0, len(u)
	if lead {
		for b < e && drop(u[b]) {
			b++
		}
	}
	if trail {
		for e > b && drop(u[e-1]) {
			e--
		}
	}
	return u[b:e]
}

func csxSplit(s []uint16, seps [][]uint16, max int) *jsArray {
	out := &jsArray{}
	if max == 0 {
		return out
	}
	any := false
	for _, sp := range seps {
		if len(sp) > 0 {
			any = true
		}
	}
	last := 0
	for i := 0; i < len(s) && len(out.elems)+1 != max; {
		hit := 0
		if !any {
			if csxIsWs(int(s[i])) {
				hit = 1
			}
		} else {
			for _, sp := range seps {
				if len(sp) > 0 && jxAt(s, sp, i) {
					hit = len(sp)
					break
				}
			}
		}
		if hit > 0 {
			out.elems = append(out.elems, strFromUnits(s[last:i]))
			i += hit
			last = i
		} else {
			i++
		}
	}
	out.elems = append(out.elems, strFromUnits(s[last:]))
	return out
}

// csxSplitCall: Split(separator(s) [, count]), lib/csharp-strings.js's.
func (rt *jsrt) csxSplitCall(u []uint16, args []interface{}) *jsArray {
	opts := 0
	if len(args) > 0 {
		if b, ok := csxIsEnumBox(args[len(args)-1]); ok {
			opts = csxEnumV(b)
			args = args[:len(args)-1]
		}
	}
	if len(args) == 2 && !isUndefOrNull(args[1]) {
		switch args[1].(type) {
		case jsChar, string, *jsArray:
		default:
			max := rt.csxNum(args[1])
			if max < 0 {
				rt.csThrow("ArgumentOutOfRangeException", "Count cannot be less than zero. (Parameter 'count')")
			}
			return csxSplitOpts(u, rt.csxSeps(args[:1]), max, opts)
		}
	}
	return csxSplitOpts(u, rt.csxSeps(args), -1, opts)
}

// csxIndexOfAny / csxLastIndexOfAny: lib/csharp-strings.js's.
func (rt *jsrt) csxIndexOfAny(u []uint16, args []interface{}) int {
	set := rt.csxCharSet(args[:1], 0)
	from := 0
	if len(args) > 1 {
		from = rt.csxNum(args[1])
	}
	if from < 0 || from > len(u) {
		rt.csThrow("ArgumentOutOfRangeException", "Index was out of range. Must be non-negative and less than or equal to the size of the collection. (Parameter 'startIndex')")
	}
	end := len(u)
	if len(args) > 2 {
		cnt := rt.csxNum(args[2])
		if cnt < 0 || from+cnt > len(u) {
			rt.csThrow("ArgumentOutOfRangeException", "Count must be positive and count must refer to a location within the string/array/collection. (Parameter 'count')")
		}
		end = from + cnt
	}
	for i := from; i < end; i++ {
		if csxInSet(set, int(u[i])) {
			return i
		}
	}
	return -1
}

func (rt *jsrt) csxLastIndexOfAny(u []uint16, args []interface{}) int {
	set := rt.csxCharSet(args[:1], 0)
	if len(u) == 0 {
		return -1
	}
	from := len(u) - 1
	if len(args) > 1 {
		from = rt.csxNum(args[1])
	}
	if from < 0 || from >= len(u) {
		rt.csThrow("ArgumentOutOfRangeException", "Index was out of range. Must be non-negative and less than the size of the collection. (Parameter 'startIndex')")
	}
	cnt := from + 1
	if len(args) > 2 {
		cnt = rt.csxNum(args[2])
		if cnt < 0 || from-cnt+1 < 0 {
			rt.csThrow("ArgumentOutOfRangeException", "Count must be positive and count must refer to a location within the string/array/collection. (Parameter 'count')")
		}
	}
	for i := from; i > from-cnt; i-- {
		if csxInSet(set, int(u[i])) {
			return i
		}
	}
	return -1
}

func (rt *jsrt) csxSeps(args []interface{}) [][]uint16 {
	src := args
	if len(args) == 1 {
		if a, ok := args[0].(*jsArray); ok {
			src = a.elems
		}
	}
	out := make([][]uint16, 0, len(src))
	for _, v := range src {
		if !isUndefOrNull(v) {
			out = append(out, rt.csxArg(v))
		}
	}
	return out
}

func csxRepeat(c uint16, n int) []uint16 {
	out := make([]uint16, n)
	for i := range out {
		out[i] = c
	}
	return out
}

func (rt *jsrt) csxPad(u []uint16, args []interface{}, left bool) string {
	n := rt.csxNum(argAt(args, 0))
	if n < 0 {
		rt.csThrow("ArgumentOutOfRangeException", "Non-negative number required. (Parameter 'totalWidth')")
	}
	fill := uint16(' ')
	if len(args) > 1 {
		fill = uint16(rt.csxCode(args[1]))
	}
	if n <= len(u) {
		return strFromUnits(u)
	}
	pad := csxRepeat(fill, n-len(u))
	if left {
		return strFromUnits(append(pad, u...))
	}
	return strFromUnits(append(append([]uint16{}, u...), pad...))
}

func (rt *jsrt) csxIndexOf(u []uint16, args []interface{}) int {
	t := rt.csxArg(argAt(args, 0))
	from := 0
	if len(args) > 1 {
		from = rt.csxNum(args[1])
	}
	if from < 0 || from > len(u) {
		rt.csThrow("ArgumentOutOfRangeException", "Index was out of range. Must be non-negative and less than or equal to the size of the collection. (Parameter 'startIndex')")
	}
	limit := len(u)
	if len(args) > 2 {
		cnt := rt.csxNum(args[2])
		if cnt < 0 || from+cnt > len(u) {
			rt.csThrow("ArgumentOutOfRangeException", "Count must be positive and count must refer to a location within the string/array/collection. (Parameter 'count')")
		}
		limit = from + cnt
	}
	return jxIndexOf(u[:limit], t, from)
}

func (rt *jsrt) csxLastIndexOf(u []uint16, args []interface{}) int {
	t := rt.csxArg(argAt(args, 0))
	if len(args) < 2 {
		return jxLastIndexOf(u, t, len(u))
	}
	from := rt.csxNum(args[1])
	if (from < 0 || from >= len(u)) && !(len(u) == 0 && (from == 0 || from == -1)) {
		rt.csThrow("ArgumentOutOfRangeException", "Index was out of range. Must be non-negative and less than the size of the collection. (Parameter 'startIndex')")
	}
	end := from + 1
	if end > len(u) {
		end = len(u)
	}
	if end < 0 {
		end = 0
	}
	return jxLastIndexOf(u[:end], t, from+1)
}

func (rt *jsrt) csxSubstring(u []uint16, args []interface{}) string {
	b := 0
	if len(args) > 0 {
		b = rt.csxNum(args[0])
	}
	if b < 0 {
		rt.csThrow("ArgumentOutOfRangeException", "StartIndex cannot be less than zero. (Parameter 'startIndex')")
	}
	if b > len(u) {
		rt.csThrow("ArgumentOutOfRangeException", "startIndex cannot be larger than length of string. (Parameter 'startIndex')")
	}
	if len(args) < 2 {
		return strFromUnits(u[b:])
	}
	n := rt.csxNum(args[1])
	if n < 0 {
		rt.csThrow("ArgumentOutOfRangeException", "Length cannot be less than zero. (Parameter 'length')")
	}
	if b+n > len(u) {
		rt.csThrow("ArgumentOutOfRangeException", "Index and length must refer to a location within the string. (Parameter 'length')")
	}
	return strFromUnits(u[b : b+n])
}

func (rt *jsrt) csxRemove(u []uint16, args []interface{}) string {
	b := rt.csxNum(argAt(args, 0))
	if b < 0 {
		rt.csThrow("ArgumentOutOfRangeException", "StartIndex cannot be less than zero. (Parameter 'startIndex')")
	}
	if len(args) < 2 {
		if b >= len(u) {
			rt.csThrow("ArgumentOutOfRangeException", "startIndex must be less than length of string. (Parameter 'startIndex')")
		}
		return strFromUnits(u[:b])
	}
	n := rt.csxNum(args[1])
	if n < 0 {
		rt.csThrow("ArgumentOutOfRangeException", "Count cannot be less than zero. (Parameter 'count')")
	}
	if b+n > len(u) {
		rt.csThrow("ArgumentOutOfRangeException", "Index and count must refer to a location within the string. (Parameter 'count')")
	}
	return strFromUnits(append(append([]uint16{}, u[:b]...), u[b+n:]...))
}

func csxToChars(u []uint16) *jsArray {
	out := &jsArray{elems: make([]interface{}, len(u))}
	for i, c := range u {
		out.elems[i] = jsChar{code: int32(c)}
	}
	return out
}

// csxStrMethod is a string receiver's method; ok is false for a name it does not
// know, which the shared js_mcall then answers or rejects.
func (rt *jsrt) csxStrMethod(s, name string, args []interface{}) (interface{}, bool) {
	u := strUnits(s)
	n := len(args)
	if csxHasCmpArg(name, args) {
		return rt.csxStrCmpMethod(s, name, args), true
	}
	switch {
	case name == "Normalize":
		rt.csxNormalized(u, args)
		return s, true
	case name == "IsNormalized":
		return rt.csxNormalized(u, args), true
	case name == "Substring":
		return rt.csxSubstring(u, args), true
	case name == "IndexOf":
		return float64(rt.csxIndexOf(u, args)), true
	case name == "LastIndexOf":
		return float64(rt.csxLastIndexOf(u, args)), true
	case name == "Contains" && n == 1:
		return jxIndexOf(u, rt.csxArg(args[0]), 0) >= 0, true
	case name == "StartsWith" && n == 1:
		return jxAt(u, rt.csxArg(args[0]), 0), true
	case name == "EndsWith" && n == 1:
		t := rt.csxArg(args[0])
		return jxAt(u, t, len(u)-len(t)), true
	case (name == "ToUpper" || name == "ToUpperInvariant") && n == 0:
		return jxUpper(s), true
	case (name == "ToLower" || name == "ToLowerInvariant") && n == 0:
		return jxLower(s), true
	case name == "Trim":
		return strFromUnits(csxTrim(u, rt.csxCharSet(args, 0), true, true)), true
	case name == "TrimStart":
		return strFromUnits(csxTrim(u, rt.csxCharSet(args, 0), true, false)), true
	case name == "TrimEnd":
		return strFromUnits(csxTrim(u, rt.csxCharSet(args, 0), false, true)), true
	case name == "Replace" && n == 2:
		o := rt.csxArg(args[0])
		if len(o) == 0 {
			rt.csThrow("ArgumentException", "String cannot be of zero length. (Parameter 'oldValue')")
		}
		var r []uint16
		if !isUndefOrNull(args[1]) {
			r = rt.csxArg(args[1])
		}
		return jxReplace(s, strFromUnits(o), strFromUnits(r)), true
	case name == "Split":
		return rt.csxSplitCall(u, args), true
	case name == "IndexOfAny":
		return float64(rt.csxIndexOfAny(u, args)), true
	case name == "LastIndexOfAny":
		return float64(rt.csxLastIndexOfAny(u, args)), true
	case name == "ToCharArray":
		if n == 0 {
			return csxToChars(u), true
		}
		b, c := rt.csxNum(args[0]), rt.csxNum(argAt(args, 1))
		if b < 0 || c < 0 || b+c > len(u) {
			rt.csThrow("ArgumentOutOfRangeException", "Index was out of range. Must be non-negative and less than the size of the collection. (Parameter 'startIndex')")
		}
		return csxToChars(u[b : b+c]), true
	case name == "PadLeft":
		return rt.csxPad(u, args, true), true
	case name == "PadRight":
		return rt.csxPad(u, args, false), true
	case name == "Insert" && n == 2:
		at := rt.csxNum(args[0])
		if at < 0 || at > len(u) {
			rt.csThrow("ArgumentOutOfRangeException", "Index was out of range. Must be non-negative and less than or equal to the size of the collection. (Parameter 'startIndex')")
		}
		ins, _ := args[1].(string)
		out := append(append(append([]uint16{}, u[:at]...), strUnits(ins)...), u[at:]...)
		return strFromUnits(out), true
	case name == "Remove":
		return rt.csxRemove(u, args), true
	}
	return nil, false
}

func (rt *jsrt) csxCharMethod(c jsChar, name string, args []interface{}) interface{} {
	if name == "CompareTo" && len(args) == 1 {
		return float64(int(c.code) - rt.csxCode(args[0]))
	}
	rt.fail("unknown char method: " + name)
	return nil
}

func (rt *jsrt) csxCharArg(args []interface{}) int {
	if len(args) == 2 {
		s, _ := args[0].(string)
		u := strUnits(s)
		i := rt.csxNum(args[1])
		if i < 0 || i >= len(u) {
			rt.csThrow("ArgumentOutOfRangeException", "Index was out of range. Must be non-negative and less than the size of the collection. (Parameter 'index')")
		}
		return int(u[i])
	}
	return rt.csxCode(argAt(args, 0))
}

func (rt *jsrt) csxCharStatic(name string, args []interface{}) interface{} {
	switch name {
	case "ToUpper":
		return jsChar{code: int32(jvChUp(rt.csxCode(argAt(args, 0))))}
	case "ToLower":
		return jsChar{code: int32(jvChLo(rt.csxCode(argAt(args, 0))))}
	}
	c := rt.csxCharArg(args)
	switch name {
	case "IsDigit":
		return jcIsDigit(c)
	case "IsLetter":
		return jcIsLetter(c)
	case "IsLetterOrDigit":
		return jcIsLetter(c) || jcIsDigit(c)
	case "IsUpper":
		return csxIsUpper(c)
	case "IsLower":
		return csxIsLower(c)
	case "IsWhiteSpace":
		return csxIsWs(c)
	case "IsControl":
		return c < 32 || (c >= 127 && c < 160)
	case "IsSurrogate":
		return c >= 55296 && c <= 57343
	}
	rt.fail("unknown char method: " + name)
	return nil
}

func csxParts(args []interface{}, k int) []interface{} {
	if len(args) == k+1 {
		if a, ok := args[k].(*jsArray); ok {
			return a.elems
		}
	}
	if k > len(args) {
		return nil
	}
	return args[k:]
}

// csxSeqParts: one LINQ query or iterator argument is its elements.
func (rt *jsrt) csxSeqParts(args []interface{}, k int) []interface{} {
	if len(args) == k+1 && csqIsLazy(args[k]) {
		return rt.csqToArray(args[k])
	}
	return csxParts(args, k)
}

func (rt *jsrt) csxJoin(sep string, parts []interface{}) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += sep
		}
		out += rt.cspStr(p)
	}
	return out
}

func (rt *jsrt) csxStringStatic(name string, args []interface{}) interface{} {
	n := len(args)
	switch {
	case name == "Concat":
		return rt.csxJoin("", rt.csxSeqParts(args, 0))
	case name == "Join" && n >= 1:
		return rt.csxJoin(strFromUnits(rt.csxArg(args[0])), rt.csxSeqParts(args, 1))
	case name == "IsNullOrEmpty" && n == 1:
		s, ok := args[0].(string)
		return !ok || s == ""
	case name == "IsNullOrWhiteSpace" && n == 1:
		s, ok := args[0].(string)
		return !ok || len(csxTrim(strUnits(s), nil, true, false)) == 0
	case name == "Equals" && n == 2:
		a, aok := args[0].(string)
		b, bok := args[1].(string)
		if aok && bok {
			return a == b
		}
		return isUndefOrNull(args[0]) && isUndefOrNull(args[1])
	case name == "Equals" && n == 3:
		return csxEqStr(rt.csxCmpKind(args[2]), args[0], args[1])
	case name == "CompareOrdinal" && n == 2:
		return csxCompareOrdinal(args[0], args[1])
	case name == "Compare":
		if _, ok := csxIsEnumBox(argAt(args, 2)); ok && n == 3 {
			return csxCmpStr(rt.csxCmpKind(args[2]), args[0], args[1])
		}
		rt.fail("String.Compare without an ordinal StringComparison is the culture's collation, which is not modelled")
	case name == "Format" && n >= 1:
		return rt.csxComposite(args[0], csxParts(args, 1))
	}
	rt.fail("unknown string method: " + name)
	return nil
}

// csxCompareOrdinal is lib/csharp-strings.js's: UTF-16 unit by unit, the
// difference of the first pair that differs, else of the lengths; null first.
func csxCompareOrdinal(a, b interface{}) float64 {
	as, aok := a.(string)
	bs, bok := b.(string)
	if !aok || !bok {
		switch {
		case !aok && !bok:
			return 0
		case !aok:
			return -1
		}
		return 1
	}
	u, v := strUnits(as), strUnits(bs)
	for i := 0; i < len(u) && i < len(v); i++ {
		if u[i] != v[i] {
			return float64(int(u[i]) - int(v[i]))
		}
	}
	return float64(len(u) - len(v))
}

func (rt *jsrt) csxNewString(args []interface{}) string {
	if len(args) == 2 {
		if c, ok := args[0].(jsChar); ok {
			n := rt.csxNum(args[1])
			if n < 0 {
				rt.csThrow("ArgumentOutOfRangeException", "Non-negative number required. (Parameter 'count')")
			}
			return strFromUnits(csxRepeat(uint16(c.code), n))
		}
	}
	a, ok := argAt(args, 0).(*jsArray)
	if !ok {
		return ""
	}
	b, e := 0, len(a.elems)
	if len(args) == 3 {
		b = rt.csxNum(args[1])
		e = b + rt.csxNum(args[2])
		if b < 0 || e < b || e > len(a.elems) {
			rt.csThrow("ArgumentOutOfRangeException", "Index was out of range. Must be non-negative and less than the size of the collection. (Parameter 'startIndex')")
		}
	}
	u := make([]uint16, 0, e-b)
	for _, v := range a.elems[b:e] {
		u = append(u, uint16(rt.csxCode(v)))
	}
	return strFromUnits(u)
}

// csxParseDouble is double.Parse's reading of s, false where it throws
// FormatException. Twin of csxParseDouble in languages/lib/csharp-strings.js,
// which states the rules.
func csxParseDouble(s string) (float64, bool) {
	numWs := func(c byte) bool { return c == ' ' || (c >= '\t' && c <= '\r') }
	i, j := 0, len(s)
	for i < j && numWs(s[i]) {
		i++
	}
	for j > i && numWs(s[j-1]) {
		j--
	}
	p := i
	neg := false
	if p < j && (s[p] == '+' || s[p] == '-') {
		neg = s[p] == '-'
		p++
	}
	num := []byte{}
	nd := 0
	for ; p < j; p++ {
		c := s[p]
		if c >= '0' && c <= '9' {
			num = append(num, c)
			nd++
		} else if c != ',' || nd == 0 {
			break
		}
	}
	if p < j && s[p] == '.' {
		num = append(num, '.')
		for p++; p < j && s[p] >= '0' && s[p] <= '9'; p++ {
			num = append(num, s[p])
			nd++
		}
	}
	if nd > 0 && p < j && (s[p] == 'e' || s[p] == 'E') {
		k := p + 1
		if k < j && (s[k] == '+' || s[k] == '-') {
			k++
		}
		k0 := k
		for k < j && s[k] >= '0' && s[k] <= '9' {
			k++
		}
		if k > k0 {
			num = append(num, s[p:k]...)
			p = k
		}
	}
	if nd > 0 && p == j {
		v, _ := strconv.ParseFloat(string(num), 64) // out of range is +-Inf with ErrRange
		if neg {
			v = -v
		}
		return v, true
	}
	t := strings.TrimFunc(s, func(r rune) bool { return r < 0x10000 && csxIsWs(int(r)) })
	switch strings.ToLower(t) {
	case "infinity", "+infinity":
		return math.Inf(1), true
	case "-infinity":
		return math.Inf(-1), true
	case "nan", "+nan", "-nan":
		return math.NaN(), true
	}
	return 0, false
}

// ----- the BCL enums a String member takes: lib/csharp-strings.js's -----

func csxEnumNames(ty string) []string {
	switch ty {
	case "StringComparison":
		return []string{"CurrentCulture", "CurrentCultureIgnoreCase", "InvariantCulture",
			"InvariantCultureIgnoreCase", "Ordinal", "OrdinalIgnoreCase"}
	case "StringSplitOptions":
		return []string{"None", "RemoveEmptyEntries", "TrimEntries"}
	}
	return []string{"FormC", "FormD", "FormKC", "FormKD"}
}

func csxEnumVals(ty string) []int {
	switch ty {
	case "StringComparison":
		return []int{0, 1, 2, 3, 4, 5}
	case "StringSplitOptions":
		return []int{0, 1, 2}
	}
	return []int{1, 2, 5, 6}
}

func csxEnumMake(t *jsObject, v int) *jsObject {
	b := newJSObject()
	b.set("__csbcl", t.props["__csbclty"])
	b.set("v", float64(v))
	b.set("__t", t)
	t.set("__v"+strconv.Itoa(v), b)
	return b
}

func csxEnumType(ty string) *jsObject {
	t := newJSObject()
	t.set("__csbclty", ty)
	vs := csxEnumVals(ty)
	for i, n := range csxEnumNames(ty) {
		t.set(n, csxEnumMake(t, vs[i]))
	}
	return t
}

func csxIsEnumBox(v interface{}) (*jsObject, bool) {
	o, ok := v.(*jsObject)
	if !ok {
		return nil, false
	}
	_, ok = o.props["__csbcl"].(string)
	return o, ok
}

func csxEnumV(b *jsObject) int { return jsToInt(b.props["v"].(float64)) }

func csxEnumOf(b *jsObject, v int) *jsObject {
	t := b.props["__t"].(*jsObject)
	if m, ok := t.props["__v"+strconv.Itoa(v)].(*jsObject); ok {
		return m
	}
	return csxEnumMake(t, v)
}

func csxEnumOr(a, b *jsObject) *jsObject { return csxEnumOf(a, csxEnumV(a)|csxEnumV(b)) }

func csxEnumStr(b *jsObject) string {
	ty := b.props["__csbcl"].(string)
	v := csxEnumV(b)
	ns, vs := csxEnumNames(ty), csxEnumVals(ty)
	for i, x := range vs {
		if x == v {
			return ns[i]
		}
	}
	if ty == "StringSplitOptions" && v > 0 && v < 4 {
		out := ""
		for i := 1; i < len(vs); i++ {
			if v&vs[i] != 0 {
				if out != "" {
					out += ", "
				}
				out += ns[i]
			}
		}
		return out
	}
	return strconv.Itoa(v)
}

func (rt *jsrt) csxEnumVal(v interface{}) int {
	if b, ok := csxIsEnumBox(v); ok {
		return csxEnumV(b)
	}
	return rt.csxNum(v)
}

// ----- ordinal comparison under a StringComparison -----

func (rt *jsrt) csxCmpKind(v interface{}) int {
	k := rt.csxEnumVal(v)
	if k == 4 || k == 5 {
		return k
	}
	if k >= 0 && k <= 3 {
		rt.fail("culture-sensitive string comparison (StringComparison." + csxEnumNames("StringComparison")[k] + ") is not modelled")
	}
	rt.csThrow("ArgumentException", "The string comparison type passed in is currently not supported. (Parameter 'comparisonType')")
	return 0
}

func csxCmpForm(k int, u []uint16) []uint16 {
	if k != 5 {
		return u
	}
	out := make([]uint16, len(u))
	for i, c := range u {
		out[i] = uint16(jvChUp(int(c)))
	}
	return out
}

func csxCmpStr(k int, a, b interface{}) float64 {
	as, aok := a.(string)
	bs, bok := b.(string)
	if !aok || !bok {
		return csxCompareOrdinal(a, b)
	}
	return csxCompareOrdinal(strFromUnits(csxCmpForm(k, strUnits(as))), strFromUnits(csxCmpForm(k, strUnits(bs))))
}

func csxEqStr(k int, a, b interface{}) bool {
	as, aok := a.(string)
	bs, bok := b.(string)
	if !aok || !bok {
		return isUndefOrNull(a) && isUndefOrNull(b)
	}
	return strFromUnits(csxCmpForm(k, strUnits(as))) == strFromUnits(csxCmpForm(k, strUnits(bs)))
}

func csxStrComparer(k int) *jsObject {
	o := newJSObject()
	o.set("__csstrcmp", float64(k))
	o.set("Compare", jsHostFunc("Compare", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return csxCmpStr(k, argAt(args, 0), argAt(args, 1))
	}))
	o.set("Equals", jsHostFunc("Equals", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return csxEqStr(k, argAt(args, 0), argAt(args, 1))
	}))
	return o
}

func csxStrComparerType() *jsObject {
	t := newJSObject()
	t.set("Ordinal", csxStrComparer(4))
	t.set("OrdinalIgnoreCase", csxStrComparer(5))
	return t
}

// csxStrComparerKind is a StringComparer's comparison kind, or -1.
func csxStrComparerKind(v interface{}) int {
	if o, ok := v.(*jsObject); ok {
		if k, ok := o.props["__csstrcmp"].(float64); ok {
			return int(k)
		}
	}
	return -1
}

func csxHasCmpArg(name string, args []interface{}) bool {
	if len(args) < 2 {
		return false
	}
	b, ok := csxIsEnumBox(args[len(args)-1])
	if !ok || b.props["__csbcl"] != "StringComparison" {
		return false
	}
	switch name {
	case "Equals", "StartsWith", "EndsWith", "Contains", "IndexOf", "LastIndexOf", "Replace":
		return true
	}
	return false
}

func (rt *jsrt) csxStrCmpMethod(s, name string, args []interface{}) interface{} {
	k := rt.csxCmpKind(args[len(args)-1])
	rest := append([]interface{}{}, args[:len(args)-1]...)
	if name == "Equals" {
		return csxEqStr(k, s, rest[0])
	}
	if isUndefOrNull(rest[0]) {
		rt.csThrow("ArgumentNullException", "Value cannot be null. (Parameter 'value')")
	}
	fs := csxCmpForm(k, strUnits(s))
	ft := csxCmpForm(k, rt.csxArg(rest[0]))
	switch name {
	case "StartsWith":
		return jxAt(fs, ft, 0)
	case "EndsWith":
		return jxAt(fs, ft, len(fs)-len(ft))
	case "Contains":
		return jxIndexOf(fs, ft, 0) >= 0
	}
	rest[0] = strFromUnits(ft)
	switch name {
	case "IndexOf":
		return float64(rt.csxIndexOf(fs, rest))
	case "LastIndexOf":
		return float64(rt.csxLastIndexOf(fs, rest))
	}
	if len(ft) == 0 {
		rt.csThrow("ArgumentException", "String cannot be of zero length. (Parameter 'oldValue')")
	}
	var r []uint16
	if !isUndefOrNull(argAt(rest, 1)) {
		r = rt.csxArg(rest[1])
	}
	u := strUnits(s)
	var out []uint16
	last := 0
	for at := jxIndexOf(fs, ft, 0); at >= 0; at = jxIndexOf(fs, ft, last) {
		out = append(append(out, u[last:at]...), r...)
		last = at + len(ft)
	}
	return strFromUnits(append(out, u[last:]...))
}

// csxNormalized: lib/csharp-strings.js's (a string of units below U+0300 is in
// form C, an ASCII one in every form; anything else is not modelled).
func (rt *jsrt) csxNormalized(u []uint16, args []interface{}) bool {
	form := 1
	if len(args) > 0 {
		form = rt.csxEnumVal(args[0])
	}
	lim := uint16(128)
	if form == 1 {
		lim = 768
	}
	for _, c := range u {
		if c >= lim {
			rt.fail("Unicode normalization of a string holding U+" + csxHex4(int(c)) + " is not modelled")
		}
	}
	return true
}

func csxHex4(c int) string {
	s := strings.ToUpper(strconv.FormatInt(int64(c), 16))
	for len(s) < 4 {
		s = "0" + s
	}
	return s
}

// ----- Split's options: lib/csharp-strings.js's csxSplitOpts -----

func csxSplitOpts(s []uint16, seps [][]uint16, max, opts int) *jsArray {
	if opts == 0 {
		return csxSplit(s, seps, max)
	}
	out := &jsArray{}
	if max == 0 {
		return out
	}
	rm := opts&1 != 0
	trim := func(u []uint16) []uint16 {
		if opts&2 != 0 {
			return csxTrim(u, nil, true, true)
		}
		return u
	}
	all := csxSplit(s, seps, -1).elems
	cur := 0
	step := func(i int) {
		n := len(strUnits(all[i].(string)))
		cur += n + csxSepLenAt(s, seps, cur+n)
	}
	for i := 0; max != 1 && i < len(all)-1; {
		if part := trim(strUnits(all[i].(string))); len(part) > 0 || !rm {
			out.elems = append(out.elems, strFromUnits(part))
		}
		step(i)
		i++
		if len(out.elems) == max-1 {
			for rm && i < len(all)-1 && len(trim(strUnits(all[i].(string)))) == 0 {
				step(i)
				i++
			}
			break
		}
	}
	if last := trim(s[cur:]); len(last) > 0 || !rm {
		out.elems = append(out.elems, strFromUnits(last))
	}
	return out
}

func csxSepLenAt(s []uint16, seps [][]uint16, i int) int {
	if i >= len(s) {
		return 0
	}
	any := false
	for _, sp := range seps {
		if len(sp) > 0 {
			any = true
			if jxAt(s, sp, i) {
				return len(sp)
			}
		}
	}
	if any {
		return 0
	}
	return 1
}

// ----- composite formatting: lib/csharp-strings.js's, whose header has the rules -----

func (rt *jsrt) csxFmtBad() { rt.csThrow("FormatException", "Format specifier was invalid.") }

func csxFmtKind(v interface{}) string {
	switch v.(type) {
	case float64, jsGInt:
		return "int"
	case jsJFlo:
		return "dbl"
	}
	return ""
}

// csxHex is an integer's two's complement hex digits at its own width.
func csxHex(v interface{}) string {
	if g, ok := v.(jsGInt); ok {
		x := uint64(g.v)
		if g.w < 64 {
			x &= 1<<g.w - 1
		}
		return strconv.FormatUint(x, 16)
	}
	return strconv.FormatUint(uint64(uint32(int32(jsToInt(v.(float64))))), 16)
}

func csxZeros(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat("0", n)
}

func csxRoundEven(ds string, keep int) (string, bool) {
	if keep >= len(ds) {
		return ds, false
	}
	head := []byte(ds[:keep])
	c := ds[keep]
	up := c > '5'
	if c == '5' {
		if strings.TrimRight(ds[keep+1:], "0") != "" {
			up = true
		} else if keep > 0 {
			up = (head[keep-1]-'0')%2 == 1
		}
	}
	if !up {
		return string(head), false
	}
	for k := len(head) - 1; k >= 0; k-- {
		if head[k] == '9' {
			head[k] = '0'
			continue
		}
		head[k]++
		return string(head), false
	}
	return "1" + string(head), true
}

type csxNum struct {
	neg, isInt bool
	x          float64
	ds         string
	e          int
}

func csxDigitsExp(t string) (string, int) {
	i := strings.IndexByte(t, 'e')
	e, _ := strconv.Atoi(t[i+1:])
	return strings.TrimRight(t[:i], "0"), e
}

func (rt *jsrt) csxNumParts(v interface{}) csxNum {
	var p csxNum
	if csxFmtKind(v) == "int" {
		p.isInt = true
		s := rt.cspStr(v)
		p.neg = strings.HasPrefix(s, "-")
		s = strings.TrimPrefix(s, "-")
		p.ds = strings.TrimRight(s, "0")
		p.e = len(s) - 1
		return p
	}
	x := rt.toNumber(v)
	p.neg = x < 0 || (x == 0 && math.Signbit(x))
	p.x = math.Abs(x)
	if x != 0 {
		p.ds, p.e = csxDigitsExp(floPrecStr(p.x, 17))
	}
	return p
}

func (rt *jsrt) csxSig(p csxNum, n int) (string, int) {
	if p.ds == "" || n < 0 {
		return "", p.e
	}
	if n == 0 {
		if p.ds[0] > '5' || (p.ds[0] == '5' && len(p.ds) > 1) {
			return "1", p.e + 1
		}
		return "", p.e
	}
	if p.isInt || (n >= len(p.ds) && n <= 17) {
		d, carry := csxRoundEven(p.ds, n)
		e := p.e
		if carry {
			e++
		}
		return strings.TrimRight(d, "0"), e
	}
	if n > 17 {
		if p.x != math.Floor(p.x) || p.x >= 1e17 {
			rt.fail("a numeric format needing more than 17 significant digits of a double is not modelled")
		}
		return p.ds, p.e
	}
	return csxDigitsExp(floPrecStr(p.x, n))
}

func csxPlace(ds string, e, prec int) (string, string) {
	ip, fp := "0", ""
	if ds != "" {
		if e >= 0 {
			if e+1 <= len(ds) {
				ip, fp = ds[:e+1], ds[e+1:]
			} else {
				ip = ds + csxZeros(e+1-len(ds))
			}
		} else {
			fp = csxZeros(-e-1) + ds
		}
	}
	return ip, fp + csxZeros(prec-len(fp))
}

func csxGroup(ip string) string {
	out := ""
	for i := 0; i < len(ip); i++ {
		if i > 0 && (len(ip)-i)%3 == 0 {
			out += ","
		}
		out += ip[i : i+1]
	}
	return out
}

func csxSignOf(p csxNum, ds string) string {
	if p.neg && (!p.isInt || ds != "") {
		return "-"
	}
	return ""
}

func (rt *jsrt) csxFixedFmt(p csxNum, prec, shift int, group bool) string {
	ds, e := rt.csxSig(p, p.e+shift+1+prec)
	ip, fp := csxPlace(ds, e+shift, prec)
	if group {
		ip = csxGroup(ip)
	}
	if prec > 0 {
		ip += "." + fp
	}
	return csxSignOf(p, ds) + ip
}

func csxExpPart(e int, letter string, minDigits int) string {
	sign := "+"
	if e < 0 {
		sign, e = "-", -e
	}
	s := strconv.Itoa(e)
	for len(s) < minDigits {
		s = "0" + s
	}
	return letter + sign + s
}

func (rt *jsrt) csxFmtValue(v interface{}, spec string) string {
	kind := csxFmtKind(v)
	if spec == "" || kind == "" {
		return rt.cspStr(v)
	}
	letter := spec[:1]
	lc := strings.ToUpper(letter)
	i := 1
	for i < len(spec) && spec[i] >= '0' && spec[i] <= '9' {
		i++
	}
	if i < len(spec) || len(spec) > 10 || !strings.Contains("DXFNEPGRC", lc) {
		rt.fail("the custom numeric format string \"" + spec + "\" is not modelled")
	}
	hasPrec := len(spec) > 1
	prec := 0
	if hasPrec {
		prec, _ = strconv.Atoi(spec[1:])
	}
	switch lc {
	case "C":
		rt.fail("the currency format is culture-specific and is not modelled")
	case "D", "X":
		if kind != "int" {
			rt.csxFmtBad()
		}
		var s string
		if lc == "D" {
			s = rt.cspStr(v)
		} else {
			s = csxHex(v)
		}
		neg := strings.HasPrefix(s, "-")
		s = strings.TrimPrefix(s, "-")
		if letter == "X" {
			s = strings.ToUpper(s)
		}
		s = csxZeros(prec-len(s)) + s
		if neg {
			return "-" + s
		}
		return s
	case "R":
		if kind != "dbl" {
			rt.csxFmtBad()
		}
		return rt.cspStr(v)
	}
	if x := rt.toNumber(v); kind == "dbl" && (math.IsNaN(x) || math.IsInf(x, 0)) {
		return rt.cspStr(v)
	}
	p := rt.csxNumParts(v)
	dflt := func(d int) int {
		if hasPrec {
			return prec
		}
		return d
	}
	switch lc {
	case "F":
		return rt.csxFixedFmt(p, dflt(2), 0, false)
	case "N":
		return rt.csxFixedFmt(p, dflt(2), 0, true)
	case "P":
		return rt.csxFixedFmt(p, dflt(2), 2, true) + " %"
	case "E":
		ep := dflt(6)
		ds, e := rt.csxSig(p, ep+1)
		md := ds
		if md == "" {
			md, e = "0", 0
		}
		md += csxZeros(ep + 1 - len(md))
		mant := md
		if ep > 0 {
			mant = md[:1] + "." + md[1:]
		}
		le := "e"
		if letter == "E" {
			le = "E"
		}
		return csxSignOf(p, ds) + mant + csxExpPart(e, le, 3)
	}
	if prec == 0 {
		return rt.cspStr(v)
	}
	gd, ge := rt.csxSig(p, prec)
	sg := csxSignOf(p, gd)
	if gd == "" {
		return sg + "0"
	}
	if ge > -5 && ge < prec {
		ip, fp := csxPlace(gd, ge, 0)
		if fp != "" {
			ip += "." + fp
		}
		return sg + ip
	}
	gm := gd
	if len(gd) > 1 {
		gm = gd[:1] + "." + gd[1:]
	}
	le := "e"
	if letter == "G" {
		le = "E"
	}
	return sg + gm + csxExpPart(ge, le, 2)
}

func csxAlign(s string, al int) string {
	w := al
	if w < 0 {
		w = -w
	}
	n := len(strUnits(s))
	if n >= w {
		return s
	}
	if al < 0 {
		return s + strings.Repeat(" ", w-n)
	}
	return strings.Repeat(" ", w-n) + s
}

func (rt *jsrt) csxBadFormat() {
	rt.csThrow("FormatException", "Input string was not in a correct format.")
}

// csxComposite is String.Format(format, args), args the params object[] array.
func (rt *jsrt) csxComposite(f interface{}, args []interface{}) string {
	fs, ok := f.(string)
	if !ok {
		rt.csThrow("ArgumentNullException", "Value cannot be null. (Parameter 'format')")
	}
	fmt := strUnits(fs)
	var out []uint16
	n := len(fmt)
	digit := func(i int) bool { return i < n && fmt[i] >= '0' && fmt[i] <= '9' }
	skipSp := func(i int) int {
		for i < n && fmt[i] == ' ' {
			i++
		}
		return i
	}
	for i := 0; i < n; {
		c := fmt[i]
		switch {
		case c == '}':
			if i+1 >= n || fmt[i+1] != '}' {
				rt.csxBadFormat()
			}
			out = append(out, '}')
			i += 2
		case c != '{':
			out = append(out, c)
			i++
		case i+1 < n && fmt[i+1] == '{':
			out = append(out, '{')
			i += 2
		default:
			i++
			idx, nd := 0, 0
			for ; digit(i); i++ {
				idx = idx*10 + int(fmt[i]-'0')
				nd++
			}
			if nd == 0 {
				rt.csxBadFormat()
			}
			i = skipSp(i)
			al := 0
			if i < n && fmt[i] == ',' {
				i = skipSp(i + 1)
				aneg := i < n && fmt[i] == '-'
				if aneg {
					i++
				}
				an := 0
				for ; digit(i); i++ {
					al = al*10 + int(fmt[i]-'0')
					an++
				}
				if an == 0 {
					rt.csxBadFormat()
				}
				if aneg {
					al = -al
				}
				i = skipSp(i)
			}
			spec := ""
			if i < n && fmt[i] == ':' {
				i++
				sb := i
				for i < n && fmt[i] != '}' && fmt[i] != '{' {
					i++
				}
				spec = strFromUnits(fmt[sb:i])
			}
			if i >= n || fmt[i] != '}' {
				rt.csxBadFormat()
			}
			i++
			if idx >= len(args) {
				rt.csThrow("FormatException", "Index (zero based) must be greater than or equal to zero and less than the size of the argument list.")
			}
			text := ""
			if !isUndefOrNull(args[idx]) {
				text = rt.csxFmtValue(args[idx], spec)
			}
			out = append(out, strUnits(csxAlign(text, al))...)
		}
	}
	return strFromUnits(out)
}

// csxConsoleText is Console.WriteLine's / Write's text: with more than one
// argument the first is a composite format string.
func (rt *jsrt) csxConsoleText(args []interface{}) string {
	if len(args) > 1 {
		return rt.csxComposite(args[0], csxParts(args[1:], 0))
	}
	return rt.cspStr(argAt(args, 0))
}
