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

func csxSplit(s []uint16, seps [][]uint16) *jsArray {
	out := &jsArray{}
	any := false
	for _, sp := range seps {
		if len(sp) > 0 {
			any = true
		}
	}
	last := 0
	for i := 0; i < len(s); {
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

func (rt *jsrt) csxSeps(args []interface{}) [][]uint16 {
	src := args
	if len(args) == 1 {
		if a, ok := args[0].(*jsArray); ok {
			src = a.elems
		}
	}
	out := make([][]uint16, 0, len(src))
	for _, v := range src {
		out = append(out, rt.csxArg(v))
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
	switch name {
	case "indexOf":
		name = "IndexOf"
	case "contains":
		name = "Contains"
	}
	u := strUnits(s)
	n := len(args)
	switch {
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
		return csxSplit(u, rt.csxSeps(args)), true
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
		return rt.csxJoin("", csxParts(args, 0))
	case name == "Join" && n >= 1:
		return rt.csxJoin(strFromUnits(rt.csxArg(args[0])), csxParts(args, 1))
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
	}
	rt.fail("unknown string method: " + name)
	return nil
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
