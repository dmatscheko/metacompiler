package abnf

import (
	"math"
	"math/big"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Ruby String algorithms of MRI's string.c. Twins of languages/lib/ruby-str.js,
// which the interpreter includes and layer 2 imports; positions here are code
// points.

// rubyCpLen is the character count of s; an ASCII string answers its length
// without a scan. Twin of rbsLen in ruby-str.js.
func (rt *jsrt) rubyCpLen(s string) int {
	if rt.strMeta(s).shape == 1 {
		return len(s)
	}
	return utf8.RuneCountInString(s)
}

// rubyCpRange is characters [b, e) of s (0 <= b <= e <= rubyCpLen(s)). Twin of
// rbsSub in ruby-str.js.
func (rt *jsrt) rubyCpRange(s string, b, e int) string {
	if rt.strMeta(s).shape == 1 {
		return s[b:e]
	}
	return string([]rune(s)[b:e])
}

// rubyIntSqrt is Integer.sqrt: the floor of the square root, exact past 2**53.
// Twins: intSqrt in ruby-interpreter.abnf, rbIntSqrt in lib/ruby-rt.metajs.
func (rt *jsrt) rubyIntSqrt(v interface{}) interface{} {
	if f, isFlo := v.(jsFlo); isFlo {
		v = math.Trunc(f.f)
	}
	n, ok := rubyBigOf(v)
	if !ok {
		rt.rubyRaiseErr("sqrt", "TypeError", "can't convert "+rt.rubyClassNameRT(v)+" into Integer")
	}
	if n.Sign() < 0 {
		rt.rubyRaiseErr("", "Math::DomainError", "Numerical argument is out of domain - \"isqrt\"")
	}
	return rt.rubyBigNarrow(new(big.Int).Sqrt(n))
}

// rubyChrUtf8 is Integer#chr("UTF-8"). Twin of rbsChrUtf8 in ruby-str.js.
func (rt *jsrt) rubyChrUtf8(x float64) string {
	cp := int64(x)
	if cp < 0 || cp > 0x10FFFF {
		rt.rubyRaiseErr("chr", "RangeError", rubyIntStr(x)+" out of char range")
	}
	if cp >= 0xD800 && cp <= 0xDFFF {
		rt.rubyRaiseErr("chr", "RangeError", "invalid codepoint 0x"+strings.ToUpper(strconv.FormatInt(cp, 16))+" in UTF-8")
	}
	return string(rune(cp))
}

// rubyFindRunes is the first index >= from where t occurs in s, or -1.
func rubyFindRunes(s, t []rune, from int) int {
	for i := from; i+len(t) <= len(s); i++ {
		j := 0
		for j < len(t) && s[i+j] == t[j] {
			j++
		}
		if j == len(t) {
			return i
		}
	}
	return -1
}

// rubySplit is String#split (rb_str_split_m); sep nil is awk mode and re, when
// not nil, wins over sep. Twin of rbsSplit in lib/ruby-str.js.
func (rt *jsrt) rubySplit(str string, sep *string, re *rxRe, lim int) *jsArray {
	out := &jsArray{}
	s := []rune(str)
	push := func(a, b int) { out.elems = append(out.elems, string(s[a:b])) }
	if lim == 1 {
		if len(s) > 0 {
			out.elems = append(out.elems, str)
		}
		return out
	}
	i, beg, p := 1, 0, 0
	switch {
	case re != nil:
		start, lastNull := 0, false
		for {
			m, ok := re.search(s, start)
			if !ok {
				rt.fail("regexp: step limit exceeded")
			}
			if m == nil {
				break
			}
			if start == m.begin && m.begin == m.end {
				if len(s) == 0 {
					out.elems = append(out.elems, "")
					break
				} else if lastNull {
					push(beg, beg+1)
					beg = start
				} else {
					start++
					lastNull = true
					continue
				}
			} else {
				push(beg, m.begin)
				beg, start = m.end, m.end
			}
			lastNull = false
			for g := 1; g <= re.ngroups; g++ {
				if m.caps[2*g] >= 0 && m.caps[2*g+1] >= 0 {
					push(m.caps[2*g], m.caps[2*g+1])
				}
			}
			i++
			if lim > 0 && lim <= i {
				break
			}
		}
	case sep == nil:
		skip, end := true, 0
		for p < len(s) {
			c := s[p]
			p++
			if skip {
				if c < 128 && rubyIsSpace(byte(c)) {
					beg = p
				} else {
					end, skip = p, false
					if lim > 0 && lim <= i {
						break
					}
				}
			} else if c < 128 && rubyIsSpace(byte(c)) {
				push(beg, end)
				skip, beg = true, p
				if lim > 0 {
					i++
				}
			} else {
				end = p
			}
		}
	case *sep == "":
		for p < len(s) {
			push(p, p+1)
			p++
			i++
			if lim > 0 && lim <= i {
				break
			}
		}
		beg = p
	default:
		t := []rune(*sep)
		for p < len(s) {
			at := rubyFindRunes(s, t, p)
			if at < 0 {
				break
			}
			push(p, at)
			p = at + len(t)
			i++
			if lim > 0 && lim <= i {
				break
			}
		}
		beg = p
	}
	if len(s) > 0 && (lim > 0 || len(s) > beg || lim < 0) {
		push(beg, len(s))
	}
	if lim == 0 {
		n := len(out.elems)
		for n > 0 && out.elems[n-1] == "" {
			n--
		}
		out.elems = out.elems[:n]
	}
	return out
}

// rubySplitCall reads String#split's arguments: a Regexp, nil / " " (awk mode)
// or a separator String, then the limit.
func (rt *jsrt) rubySplitCall(s string, args []interface{}) *jsArray {
	pos, _ := rubyBlockArgs(args)
	var sep *string
	var re *rxRe
	if len(pos) > 0 {
		if rxIsRegexObj(pos[0]) {
			re = rxObjRe(pos[0])
		} else if isUndefOrNull(pos[0]) != true {
			if t := rt.rubyStr(pos[0]); t != " " {
				sep = &t
			}
		}
	}
	lim := 0
	if len(pos) > 1 {
		lim = int(rubyToF(pos[1]))
	}
	return rt.rubySplit(s, sep, re, lim)
}

// rubySuccKind is MRI's ISALNUM class for String#succ: 1 a digit, 2 a letter.
func rubySuccKind(c rune) int {
	switch {
	case c >= '0' && c <= '9':
		return 1
	case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z':
		return 2
	}
	return 0
}

// rubyStrSucc is String#succ (str_succ). Twin of rbsSucc in lib/ruby-str.js.
func rubyStrSucc(str string) string {
	s := []rune(str)
	if len(s) == 0 {
		return str
	}
	skipped, lastKind, carryAt := false, 0, -1
	var carry rune
	for i := len(s) - 1; i >= 0; i-- {
		k := rubySuccKind(s[i])
		if skipped && lastKind != 0 && k != 0 && k != lastKind {
			break
		}
		if k == 0 {
			skipped = true
			continue
		}
		skipped = false
		switch s[i] {
		case '9':
			s[i], carry = '0', '1'
		case 'z':
			s[i], carry = 'a', 'a'
		case 'Z':
			s[i], carry = 'A', 'A'
		default:
			s[i]++
			return string(s)
		}
		lastKind, carryAt = k, i
	}
	if carryAt >= 0 {
		return string(s[:carryAt]) + string(carry) + string(s[carryAt:])
	}
	j := len(s) - 1
	if s[j] == 0xD7FF {
		s[j] = 0xE000
	} else {
		s[j]++
	}
	return string(s)
}

// rubyRfindRunes is the last index <= from where t occurs in s, or -1.
func rubyRfindRunes(s, t []rune, from int) int {
	if from+len(t) > len(s) {
		from = len(s) - len(t)
	}
	for i := from; i >= 0; i-- {
		if string(s[i:i+len(t)]) == string(t) {
			return i
		}
	}
	return -1
}

// rubyStrIndex is String#index / #rindex with a String or a Regexp and a
// character offset. Twin of rbStrSearch in lib/ruby-rt.metajs.
func (rt *jsrt) rubyStrIndex(str string, fwd bool, args []interface{}) interface{} {
	pos, _ := rubyBlockArgs(args)
	pat := rubySV(argAt(pos, 0))
	s := []rune(str)
	n := len(s)
	p := 0
	if len(pos) > 1 {
		p = int(rubyToF(pos[1]))
	} else if !fwd {
		p = n
	}
	if p < 0 {
		p += n
	}
	if p < 0 || (fwd && p > n) {
		p = -1
	} else if p > n {
		p = n
	}
	if rxIsRegexObj(pat) {
		if p >= 0 && !fwd {
			re := rxObjRe(pat)
			for ; p >= 0; p-- {
				m, ok := re.matchAt(s, p)
				if !ok {
					rt.fail("regexp: step limit exceeded")
				}
				if m != nil && m.begin >= 0 {
					break
				}
			}
		}
		if p < 0 {
			rt.rxSetGlobals(nil)
			return jsNull
		}
		md := rt.rxRunMatch(pat, str, p)
		if md == nil {
			return jsNull
		}
		_, _, m := rxMatchOf(md)
		return float64(m.begin)
	}
	if p < 0 {
		return jsNull
	}
	t := []rune(rt.rubyStr(pat))
	at := rubyRfindRunes(s, t, p)
	if fwd {
		at = rubyFindRunes(s, t, p)
	}
	if at < 0 {
		return jsNull
	}
	return float64(at)
}

// rubyStrPatAt: s[str] is str when s contains it; s[re] and s[re, group] the
// matched text; nil otherwise. Twin of rbStrPatAt in lib/ruby-rt.metajs.
func (rt *jsrt) rubyStrPatAt(s string, k, g interface{}) interface{} {
	k = rubySV(k)
	if !rxIsRegexObj(k) {
		t := rt.rubyStr(k)
		if strings.Contains(s, t) {
			return t
		}
		return jsNull
	}
	md := rt.rxRunMatch(k, s, 0)
	if md == nil {
		return jsNull
	}
	if g == nil {
		g = float64(0)
	}
	return rt.rxMdIndex(md, rubySV(g))
}

// rubyStrPatSpan is the rune range s[k1] = v / s[k1, k2] = v replaces when k1 is
// a String or a Regexp (k2 its group); IndexError when there is nothing to
// replace. Twin of rbStrPatSpan in lib/ruby-rt.metajs.
func (rt *jsrt) rubyStrPatSpan(s string, k1, k2 interface{}) (int, int) {
	k1 = rubySV(k1)
	rs := []rune(s)
	if !rxIsRegexObj(k1) {
		t := []rune(rt.rubyStr(k1))
		a := rubyFindRunes(rs, t, 0)
		if a < 0 {
			rt.rubyRaiseErr("[]=", "IndexError", "string not matched")
		}
		return a, a + len(t)
	}
	re := rxObjRe(k1)
	md := rt.rxRunMatch(k1, s, 0)
	if md == nil {
		rt.rubyRaiseErr("[]=", "IndexError", "regexp not matched")
	}
	_, _, m := rxMatchOf(md)
	g := 0
	switch k := rubySV(k2).(type) {
	case float64:
		g = int(k)
	case jsSym:
		g = re.indexOfName(k.s)
	case string:
		g = re.indexOfName(k)
	}
	if g < 0 || g > re.ngroups {
		rt.rubyRaiseErr("[]=", "IndexError", "index "+jsNumString(float64(g))+" out of regexp")
	}
	if m.caps[2*g] < 0 {
		rt.rubyRaiseErr("[]=", "IndexError", "regexp group "+jsNumString(float64(g))+" not matched")
	}
	return m.caps[2*g], m.caps[2*g+1]
}

// rubyExpand is a sub / gsub replacement template (rb_reg_regsub). Twin of
// rbsExpand in lib/ruby-str.js.
func rubyExpand(re *rxRe, text []rune, m *rxMatch, repl string) string {
	var out strings.Builder
	r := []rune(repl)
	for i := 0; i < len(r); i += 2 {
		if r[i] != '\\' || i+1 >= len(r) {
			out.WriteRune(r[i])
			i--
			continue
		}
		g := -1
		switch d := r[i+1]; {
		case d >= '0' && d <= '9':
			g = int(d - '0')
		case d == '&':
			g = 0
		case d == '+':
			for g = re.ngroups; g > 0 && m.caps[2*g] < 0; g-- {
			}
		case d == '`':
			out.WriteString(string(text[:m.begin]))
		case d == '\'':
			out.WriteString(string(text[m.end:]))
		case d == '\\':
			out.WriteRune('\\')
		case d == 'k' && i+2 < len(r) && r[i+2] == '<' && strings.ContainsRune(string(r[i+3:]), '>'):
			j := i + 3
			for r[j] != '>' {
				j++
			}
			g = re.indexOfName(string(r[i+3 : j]))
			i = j - 1
		default:
			out.WriteString(string(r[i : i+2]))
		}
		if g >= 0 {
			if t, ok := re.group(text, m, g); ok {
				out.WriteString(t)
			}
		}
	}
	return out.String()
}

// rubySubWith is sub / gsub's replacement for each match without a block: the
// Hash's value for the matched text, or the expanded template.
func (rt *jsrt) rubySubWith(repl interface{}) func(*rxRe, []rune, *rxMatch) string {
	if _, _, isDict := dictParts(repl); isDict {
		return func(re *rxRe, text []rune, m *rxMatch) string {
			g, _ := re.group(text, m, 0)
			v := rt.rubyIndexGet(repl, g)
			if isUndefOrNull(v) == true {
				return ""
			}
			return rt.rubyStr(v)
		}
	}
	tmpl := rt.rubyStr(repl)
	return func(re *rxRe, text []rune, m *rxMatch) string { return rubyExpand(re, text, m, tmpl) }
}

// rubyRoundArgs reads #floor / #ceil / #round / #truncate's arguments: ndigits,
// and for #round the half: keyword, as the digit rounders' mode byte ('r' half up,
// 'e' half even, 'd' half down). Twin of rbRoundArgs in lib/ruby-rt.metajs.
func (rt *jsrt) rubyRoundArgs(name string, args []interface{}) (int, byte) {
	mode := map[string]byte{"floor": 'f', "ceil": 'c', "truncate": 't'}[name]
	if mode == 0 {
		mode = 'r'
	}
	n := len(args)
	var last interface{}
	if n > 0 {
		last = args[n-1]
	}
	if keys, vals, isDict := dictParts(last); isDict {
		if i := rt.rubyDictFind(keys, jsSym{s: "half"}); i >= 0 && name == "round" {
			hn := ""
			if sy, isSym := vals.elems[i].(jsSym); isSym {
				hn = sy.s
			} else if isUndefOrNull(vals.elems[i]) != true {
				hn = rt.rubyStr(vals.elems[i])
			}
			switch hn {
			case "", "up":
			case "even":
				mode = 'e'
			case "down":
				mode = 'd'
			default:
				rt.rubyRaiseErr("round", "ArgumentError", "invalid rounding mode: "+hn)
			}
		}
		n--
	}
	nd := 0
	if n > 0 {
		nd = int(math.Trunc(rubyToF(args[0])))
	}
	return nd, mode
}

// rubyRoundHalf is round_half_even ('e') / round_half_down ('d') of numeric.c.
// Twin of rbnRoundHalf in lib/ruby-num.js.
func rubyRoundHalf(x, s float64, mode byte) float64 {
	// float64() keeps Go from fusing x*s into the subtractions below (FMA).
	xs := float64(x * s)
	if mode == 'd' {
		f := math.Round(xs)
		if x > 0 && (f-0.5)/s >= x {
			f--
		}
		if x < 0 && (f+0.5)/s <= x {
			f++
		}
		return f
	}
	if x > 0 {
		f := math.Floor(xs)
		d := xs - f
		if d > 0.5 {
			d = 1
		} else if d == 0.5 || (f+0.5)/s <= x {
			d = math.Mod(f, 2)
		} else {
			d = 0
		}
		return f + d
	}
	if x < 0 {
		f := math.Ceil(xs)
		d := f - xs
		if d > 0.5 {
			d = 1
		} else if d == 0.5 || (f-0.5)/s >= x {
			d = math.Mod(-f, 2)
		} else {
			d = 0
		}
		return f - d
	}
	return x
}

// rubyHalfAway: whether an Integer rounded at a negative ndigits moves away from
// zero, the tail of dropped digits not being all zero. Twin of rbnHalfAway in
// lib/ruby-num.js.
func rubyHalfAway(mode byte, head, tail string) bool {
	if tail[0] != '5' || mode == 'r' {
		return tail[0] >= '5'
	}
	if strings.TrimRight(tail[1:], "0") != "" {
		return true
	}
	if mode == 'd' || head == "" {
		return false
	}
	return (head[len(head)-1]-'0')%2 == 1
}

// rubyIsReal: an Integer, a Float or a Rational - what Array#sum adds as a double
// once a Float joined the sum.
func rubyIsReal(v interface{}) bool {
	switch v.(type) {
	case float64, jsFlo, *jsBigInt, jsRat:
		return true
	}
	return false
}

// rubyKahan is one step of Array#sum's Kahan-Babuska summation (ary_add_one in
// array.c) on the running sum f and compensation c. Twin of rbnKahan in
// lib/ruby-num.js.
func rubyKahan(f, c, x float64) (float64, float64) {
	switch {
	case math.IsNaN(f):
		return f, c
	case math.IsNaN(x), math.IsInf(x, 0) && !math.IsInf(f, 0):
		return x, c
	case math.IsInf(x, 0):
		if (x > 0) != (f > 0) {
			return math.NaN(), c
		}
		return x, c
	case math.IsInf(f, 0):
		return f, c
	}
	t := f + x
	if math.Abs(f) >= math.Abs(x) {
		c += (f - t) + x
	} else {
		c += (x - t) + f
	}
	return t, c
}

// rubyHashEach is Hash#each / #each_pair. An entry deleted before it is reached is
// not yielded; a key ADDED by the block is MRI's RuntimeError, and is taken out
// again. Twin of rbHashEach in lib/ruby-rt.metajs.
func (rt *jsrt) rubyHashEach(keys, vals *jsArray, blk interface{}) {
	added := func() {
		rt.rubyRaiseErr("each", "RuntimeError", "can't add a new key into hash during iteration")
	}
	left := len(keys.elems)
	for i := 0; i < len(keys.elems); {
		k := keys.elems[i]
		n0 := len(keys.elems)
		if left == 0 {
			added()
		}
		left--
		rt.call(blk, jsUndef, []interface{}{rt.rubyKeyOut(k), vals.elems[i]})
		if len(keys.elems) > n0 {
			keys.elems, vals.elems = keys.elems[:n0], vals.elems[:n0]
			keys.dropIdx()
			added()
		}
		if i < len(keys.elems) && keys.elems[i] == k {
			i++
		}
	}
}

// rubyIsIntV: an Integer, small or big.
func rubyIsIntV(v interface{}) bool {
	switch v.(type) {
	case float64, *jsBigInt:
		return true
	}
	return false
}

// rubyFloatStep is the doubles a Range#step over Floats visits
// (ruby_float_step). Twin of rbnFloatStep in lib/ruby-num.js.
func rubyFloatStep(beg, end, unit float64, excl bool) []float64 {
	out := []float64{}
	if math.IsInf(unit, 0) {
		if (unit > 0 && beg <= end) || (unit < 0 && beg >= end) {
			out = append(out, beg)
		}
		return out
	}
	n := (end - beg) / unit
	err := (math.Abs(beg) + math.Abs(end) + math.Abs(end-beg)) / math.Abs(unit) * 0x1p-52
	if err > 0.5 {
		err = 0.5
	}
	if excl {
		if n <= 0 {
			return out
		}
		if n < 1 {
			n = 0
		} else {
			n = math.Floor(n - err)
		}
	} else {
		if n < 0 {
			return out
		}
		n = math.Floor(n + err)
	}
	for i := 0.0; i <= n; i++ {
		d := float64(i*unit) + beg
		if (unit >= 0 && end < d) || (unit < 0 && d < end) {
			d = end
		}
		if !excl || (unit >= 0 && d < end) || (unit < 0 && d > end) {
			out = append(out, d)
		}
	}
	return out
}
