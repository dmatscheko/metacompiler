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

// rubyMkLazy is an Enumerator::Lazy or the endless Enumerator of Array#cycle.
// Twins: mkLazy in ruby-interpreter.abnf (which states the rules), rbMkLazy in
// lib/ruby-rt.metajs; the run itself is rubyLazyFirst, twin of rbaLazyFirst in
// lib/ruby-arr.js.
func rubyMkLazy(kind string, items *jsArray, lo float64, hi interface{}, excl, lazy bool, desc string) *jsObject {
	o := newJSObject()
	o.set("__renum", true)
	o.set("__rlazy", true)
	o.set("items", items)
	o.set("pos", float64(0))
	o.set("op", "each")
	o.set("kind", kind)
	o.set("lo", lo)
	o.set("hi", hi)
	o.set("excl", excl)
	o.set("ops", &jsArray{})
	o.set("lazy", lazy)
	o.set("desc", desc)
	return o
}

func rubyLazyCopy(t *jsObject, op *jsObject, lazy bool, desc string) *jsObject {
	c := rubyMkLazy(t.props["kind"].(string), t.props["items"].(*jsArray), t.props["lo"].(float64), t.props["hi"],
		t.props["excl"] == true, lazy, desc)
	ops := &jsArray{elems: append([]interface{}{}, t.props["ops"].(*jsArray).elems...)}
	if op != nil {
		ops.elems = append(ops.elems, op)
	}
	c.set("ops", ops)
	return c
}

func (rt *jsrt) rubyLazyFirst(t *jsObject, n int) []interface{} {
	out := []interface{}{}
	ops := t.props["ops"].(*jsArray).elems
	items := t.props["items"].(*jsArray).elems
	kind := t.props["kind"].(string)
	lo := t.props["lo"].(float64)
	hi, bounded := t.props["hi"].(float64)
	excl := t.props["excl"] == true
	cnt := make([]int, len(ops))
	on := make([]bool, len(ops))
	call := func(f, v interface{}) interface{} { return rt.call(f, jsUndef, []interface{}{v}) }
	done := n == 0
	for i := 0; !done; i++ {
		var v interface{}
		if kind == "range" {
			x := lo + float64(i)
			if bounded && ((excl && x >= hi) || (!excl && x > hi)) {
				break
			}
			v = x
		} else {
			if len(items) == 0 || (kind == "arr" && i >= len(items)) {
				break
			}
			v = items[i%len(items)]
		}
		keep := true
		for j := 0; keep && j < len(ops); j++ {
			op := ops[j].(*jsObject)
			f := op.props["f"]
			switch op.props["k"] {
			case "map":
				v = call(f, v)
			case "select":
				keep = rubyTruthy(call(f, v))
			case "reject":
				keep = !rubyTruthy(call(f, v))
			case "take_while":
				if !rubyTruthy(call(f, v)) {
					keep, done = false, true
				}
			case "drop_while":
				if !on[j] {
					if rubyTruthy(call(f, v)) {
						keep = false
					} else {
						on[j] = true
					}
				}
			case "take":
				if lim := int(rubyToF(op.props["n"])); lim <= 0 {
					keep, done = false, true
				} else {
					cnt[j]++
					if cnt[j] >= lim {
						done = true
					}
				}
			}
		}
		if keep {
			out = append(out, v)
			if n >= 0 && len(out) >= n {
				done = true
			}
		}
	}
	return out
}

func (rt *jsrt) rubyRangeLazy(r *jsObject) interface{} {
	d := "#<Enumerator::Lazy: " + rt.rubyInspect(r) + ">"
	hi := r.props["end"]
	if isUndefOrNull(hi) {
		hi = jsNull
	} else if f, isFlo := hi.(jsFlo); isFlo && math.IsInf(f.f, 1) {
		hi = jsNull
	}
	lo, loNum := r.props["begin"].(float64)
	_, hiNum := hi.(float64)
	if loNum && (hi == jsNull || hiNum) {
		return rubyMkLazy("range", &jsArray{}, lo, hi, r.props["excl"] == true, true, d)
	}
	return rubyMkLazy("arr", rt.rubyRangeToArr(r), 0, jsNull, false, true, d)
}

func (rt *jsrt) rubyLazyMethod(t *jsObject, name string, args []interface{}) interface{} {
	pos, blk := rubyBlockArgs(args)
	desc := t.props["desc"].(string)
	lazy := t.props["lazy"] == true
	if k := rubyLazyOp(name); k != "" && blk != nil {
		op := newJSObject()
		op.set("k", k)
		op.set("f", blk)
		op.set("n", float64(0))
		return rubyLazyCopy(t, op, true, "#<Enumerator::Lazy: "+desc+":"+name+">")
	}
	switch name {
	case "take":
		tn := int(math.Trunc(rubyToF(argAt(pos, 0))))
		if lazy {
			op := newJSObject()
			op.set("k", "take")
			op.set("f", jsNull)
			op.set("n", float64(tn))
			return rubyLazyCopy(t, op, true, "#<Enumerator::Lazy: "+desc+":take("+strconv.Itoa(tn)+")>")
		}
		return &jsArray{elems: rt.rubyLazyFirst(t, tn)}
	case "first":
		if len(pos) > 0 {
			return &jsArray{elems: rt.rubyLazyFirst(t, int(math.Trunc(rubyToF(pos[0]))))}
		}
		if f1 := rt.rubyLazyFirst(t, 1); len(f1) > 0 {
			return f1[0]
		}
		return jsNull
	case "to_a", "force", "entries":
		return &jsArray{elems: rt.rubyLazyFirst(t, -1)}
	case "each":
		if blk != nil {
			for _, v := range rt.rubyLazyFirst(t, -1) {
				rt.call(blk, jsUndef, []interface{}{v})
			}
			return t
		}
	case "lazy":
		if lazy {
			return t
		}
		return rubyLazyCopy(t, nil, true, "#<Enumerator::Lazy: "+desc+">")
	case "next", "peek":
		cur := int(rubyToF(t.props["pos"]))
		got := rt.rubyLazyFirst(t, cur+1)
		if len(got) <= cur {
			e := rt.rubyExc(rt.rubyBuiltinClass("StopIteration"), []interface{}{"iteration reached an end"})
			e.set("__rframe", name)
			panic(&jsThrown{value: e})
		}
		if name == "next" {
			t.set("pos", float64(cur+1))
		}
		return got[len(got)-1]
	case "rewind":
		t.set("pos", float64(0))
		return t
	case "size":
		if t.props["kind"] == "arr" && len(t.props["ops"].(*jsArray).elems) == 0 {
			return float64(len(t.props["items"].(*jsArray).elems))
		}
		return jsNull
	case "inspect", "to_s":
		return desc
	case "class":
		return rt.rubyBuiltinClass("Enumerator")
	}
	rt.rubyNoMethod(t, name)
	return nil
}

// rubyLazyOp is the lazy op a method name is, or "". Twin of rbaLazyOp in
// lib/ruby-arr.js.
func rubyLazyOp(name string) string {
	switch name {
	case "map", "collect":
		return "map"
	case "select", "filter":
		return "select"
	case "reject", "take_while", "drop_while":
		return name
	}
	return ""
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

// ----- Binary (ASCII-8BIT) Strings -----

// jsRBin is a binary Ruby String, holding its bytes as they are. Integer#chr past
// 127, String#b, force_encoding("BINARY") and Array#pack make one; a pointer, so
// << changes it in place. Twin of the {__rbin} box of ruby-interpreter.abnf and
// lib/ruby-rt.metajs, which hold one char per byte.
type jsRBin struct {
	b  string
	id jsHandleID
}

// rubyBinText is the text the other engines hold for the bytes b: one char per
// byte, which is what the String methods a binary String delegates run on.
func rubyBinText(b string) string {
	rs := make([]rune, len(b))
	for i := 0; i < len(b); i++ {
		rs[i] = rune(b[i])
	}
	return string(rs)
}

// rubyBinOfText is rubyBinText's inverse, for a String a delegated method answers.
func rubyBinOfText(s string) string {
	out := make([]byte, 0, len(s))
	for _, r := range s {
		out = append(out, byte(r))
	}
	return string(out)
}

func rubyAsciiOnly(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > 127 {
			return false
		}
	}
	return true
}

// rubyBinInspect is a binary String's #inspect. Twin of rbsBinInspect in
// ruby-str.js.
func rubyBinInspect(b string) string {
	const hx = "0123456789ABCDEF"
	var sb strings.Builder
	sb.WriteByte('"')
	for i := 0; i < len(b); i++ {
		c := b[i]
		switch {
		case c == '"':
			sb.WriteString("\\\"")
		case c == '\\':
			sb.WriteString("\\\\")
		case c == '\n':
			sb.WriteString("\\n")
		case c == '\t':
			sb.WriteString("\\t")
		case c == '\r':
			sb.WriteString("\\r")
		case c == '\f':
			sb.WriteString("\\f")
		case c == '\v':
			sb.WriteString("\\v")
		case c == '\b':
			sb.WriteString("\\b")
		case c == 7:
			sb.WriteString("\\a")
		case c == 27:
			sb.WriteString("\\e")
		case c == '#' && i+1 < len(b) && (b[i+1] == '{' || b[i+1] == '$' || b[i+1] == '@'):
			sb.WriteString("\\#")
		case c < 32 || c >= 127:
			sb.WriteString("\\x")
			sb.WriteByte(hx[c/16])
			sb.WriteByte(hx[c%16])
		default:
			sb.WriteByte(c)
		}
	}
	sb.WriteByte('"')
	return sb.String()
}

// rubyPackDir is one Array#pack / String#unpack directive: C, c or U, and its
// count (-1 for `*'). Twin of rbsPackDirs in ruby-str.js.
type rubyPackDir struct {
	d byte
	n int
}

func (rt *jsrt) rubyPackDirs(name, fmtS string) []rubyPackDir {
	var out []rubyPackDir
	for i := 0; i < len(fmtS); {
		d := fmtS[i]
		i++
		if d == ' ' {
			continue
		}
		if d != 'C' && d != 'c' && d != 'U' {
			rt.rubyRaiseErr(name, "NotImplementedError", name+" directive '"+string(rune(d))+"' is not supported")
		}
		n := 1
		if i < len(fmtS) && fmtS[i] == '*' {
			n = -1
			i++
		} else if i < len(fmtS) && fmtS[i] >= '0' && fmtS[i] <= '9' {
			n = 0
			for i < len(fmtS) && fmtS[i] >= '0' && fmtS[i] <= '9' {
				n = n*10 + int(fmtS[i]-'0')
				i++
			}
		}
		out = append(out, rubyPackDir{d, n})
	}
	return out
}

// rubyPack is Array#pack over C, c and U: a binary String unless the first
// directive is U. Twin of rbsPack in ruby-str.js.
func (rt *jsrt) rubyPack(vals []interface{}, fmtS string) interface{} {
	dirs := rt.rubyPackDirs("pack", fmtS)
	var sb strings.Builder
	k := 0
	for _, d := range dirs {
		n := d.n
		if n < 0 {
			n = len(vals) - k
		}
		for j := 0; j < n; j++ {
			if k >= len(vals) {
				rt.rubyRaiseErr("pack", "ArgumentError", "too few arguments")
			}
			v := int64(rubyToF(vals[k]))
			k++
			if d.d != 'U' {
				sb.WriteByte(byte(((v % 256) + 256) % 256))
				continue
			}
			if v < 0 || v > 0x10FFFF || (v >= 0xD800 && v <= 0xDFFF) {
				rt.rubyRaiseErr("pack", "ArgumentError", "pack(U): value out of range")
			}
			sb.WriteString(string(rune(v)))
		}
	}
	if len(dirs) > 0 && dirs[0].d == 'U' {
		return &jsRStr{s: sb.String()}
	}
	return &jsRBin{b: sb.String()}
}

// rubyUnpack is String#unpack / #unpack1 of the bytes b. Twin of rbsUnpack in
// ruby-str.js.
func (rt *jsrt) rubyUnpack(b, name string, args []interface{}) interface{} {
	dirs := rt.rubyPackDirs(name, rt.rubyStr(argAt(args, 0)))
	out := &jsArray{}
	p := 0
	for _, d := range dirs {
		for j := 0; (d.n < 0 && p < len(b)) || (d.n >= 0 && j < d.n); j++ {
			switch {
			case p >= len(b):
				if d.d != 'U' {
					out.elems = append(out.elems, jsNull)
				}
			case d.d == 'U':
				r, size := utf8.DecodeRuneInString(b[p:])
				if r == utf8.RuneError && size <= 1 {
					rt.rubyRaiseErr(name, "ArgumentError", "malformed UTF-8 character")
				}
				out.elems = append(out.elems, float64(r))
				p += size
			default:
				v := float64(b[p])
				if d.d == 'c' && v > 127 {
					v -= 256
				}
				out.elems = append(out.elems, v)
				p++
			}
		}
	}
	if name == "unpack1" {
		if len(out.elems) == 0 {
			return jsNull
		}
		return out.elems[0]
	}
	return out
}

// rubyEncArg is the encoding a force_encoding argument (a name or an Encoding)
// selects: "BINARY" or "UTF-8", US-ASCII counting as UTF-8. Twin of encArgName in
// ruby-interpreter.abnf.
func (rt *jsrt) rubyEncArg(e interface{}) string {
	if o, isObj := e.(*jsObject); isObj {
		if n, isEnc := o.props["__renc"].(string); isEnc {
			if n == "ASCII-8BIT" {
				return "BINARY"
			}
			return "UTF-8"
		}
	}
	switch strings.ToUpper(rt.rubyStr(e)) {
	case "BINARY", "ASCII-8BIT":
		return "BINARY"
	case "UTF-8", "US-ASCII", "ASCII":
		return "UTF-8"
	}
	rt.rubyRaiseErr("force_encoding", "ArgumentError", "unknown encoding name - "+rt.rubyStr(e))
	return ""
}

// rubyEncoding is the Encoding object of a name: UTF-8, ASCII-8BIT or US-ASCII.
func (rt *jsrt) rubyEncoding(n string) interface{} {
	c := rt.rubyBuiltinClass("Encoding")
	switch n {
	case "ASCII-8BIT":
		return c.props["$c$BINARY"]
	case "US-ASCII":
		return c.props["$c$US_ASCII"]
	}
	return c.props["$c$UTF_8"]
}

// rubyBinCat is String#+ / #<< with a binary side: two binary Strings join, an
// ASCII-only side takes the other side's encoding, and two non-ASCII sides are
// MRI's Encoding::CompatibilityError. Twin of binCat in ruby-interpreter.abnf.
func (rt *jsrt) rubyBinCat(l, r interface{}) interface{} {
	lb, lBin := l.(*jsRBin)
	rb, rBin := r.(*jsRBin)
	if lBin && rBin {
		return &jsRBin{b: lb.b + rb.b}
	}
	if lBin {
		o := rt.rubyStrOperand(r)
		if rubyAsciiOnly(o) {
			return &jsRBin{b: lb.b + o}
		}
		if rubyAsciiOnly(lb.b) {
			return &jsRStr{s: lb.b + o}
		}
		rt.rubyRaiseErr("+", "Encoding::CompatibilityError", "incompatible character encodings: ASCII-8BIT and UTF-8")
	}
	o := rt.rubyStr(l)
	if rubyAsciiOnly(o) {
		return &jsRBin{b: o + rb.b}
	}
	if rubyAsciiOnly(rb.b) {
		return &jsRStr{s: o + rb.b}
	}
	rt.rubyRaiseErr("+", "Encoding::CompatibilityError", "incompatible character encodings: UTF-8 and ASCII-8BIT")
	return nil
}

// rubyBinEq is == with the binary String b on one side: the same bytes, and an
// ASCII-only text where the other side is UTF-8. Twin of binEq in
// ruby-interpreter.abnf.
func rubyBinEq(b *jsRBin, o interface{}) bool {
	switch t := o.(type) {
	case *jsRBin:
		return t.b == b.b
	case *jsRStr:
		return t.s == b.b && rubyAsciiOnly(t.s)
	case string:
		return t == b.b && rubyAsciiOnly(t)
	}
	return false
}

// rubyBinMethod is a method of a binary String: the ones that read its bytes
// here, every other String method on its text, a String it answers binary too.
// Twin of binMethod in ruby-interpreter.abnf.
func (rt *jsrt) rubyBinMethod(t *jsRBin, name string, args []interface{}) interface{} {
	pos, blk := rubyBlockArgs(args)
	bytesArr := func() *jsArray {
		a := &jsArray{elems: make([]interface{}, len(t.b))}
		for i := 0; i < len(t.b); i++ {
			a.elems[i] = float64(t.b[i])
		}
		return a
	}
	switch name {
	case "bytes":
		return bytesArr()
	case "bytesize":
		return float64(len(t.b))
	case "getbyte":
		i := int(rubyToF(argAt(pos, 0)))
		if i < 0 {
			i += len(t.b)
		}
		if i < 0 || i >= len(t.b) {
			return jsNull
		}
		return float64(t.b[i])
	case "each_byte":
		if blk == nil {
			return rubyEnumAt(rubyMkEnum(bytesArr()), t, name, pos)
		}
		for i := 0; i < len(t.b); i++ {
			rt.call(blk, jsUndef, []interface{}{float64(t.b[i])})
		}
		return t
	case "inspect", "dump":
		return rubyBinInspect(t.b)
	case "encoding":
		return rt.rubyEncoding("ASCII-8BIT")
	case "force_encoding":
		if rt.rubyEncArg(argAt(pos, 0)) == "BINARY" || !utf8.ValidString(t.b) {
			return t
		}
		return &jsRStr{s: t.b}
	case "b", "dup", "clone", "+@":
		return &jsRBin{b: t.b}
	case "valid_encoding?":
		return true
	case "ascii_only?":
		return rubyAsciiOnly(t.b)
	case "to_s", "to_str", "-@":
		return t
	case "<<", "concat":
		for _, a := range pos {
			if f, isNum := a.(float64); isNum {
				a = &jsRBin{b: string([]byte{byte(int64(f) & 255)})}
			}
			t.b = rt.rubyBinCat(t, a).(*jsRBin).b
		}
		return t
	case "+":
		return rt.rubyBinCat(t, argAt(pos, 0))
	case "==", "===", "eql?":
		return rubyBinEq(t, argAt(pos, 0))
	case "*":
		n := int(rubyToF(argAt(pos, 0)))
		if n < 0 {
			rt.rubyRaiseErr("*", "ArgumentError", "negative argument")
		}
		return &jsRBin{b: strings.Repeat(t.b, n)}
	case "unpack", "unpack1":
		return rt.rubyUnpack(t.b, name, pos)
	}
	var r interface{}
	if rubyBinOpName(name) {
		r = rt.rubyBin(name, rubyBinText(t.b), rubySV(argAt(pos, 0)))
	} else {
		r = rt.rubyMethod(rubyBinText(t.b), name, args)
	}
	switch v := r.(type) {
	case string:
		return &jsRBin{b: rubyBinOfText(v)}
	case *jsRStr:
		return &jsRBin{b: rubyBinOfText(v.s)}
	case *jsArray:
		for i, e := range v.elems {
			switch s := e.(type) {
			case string:
				v.elems[i] = &jsRBin{b: rubyBinOfText(s)}
			case *jsRStr:
				v.elems[i] = &jsRBin{b: rubyBinOfText(s.s)}
			}
		}
	}
	return r
}

// rubyBinOpName: the operators a binary String reaches through rubyBin.
func rubyBinOpName(n string) bool {
	switch n {
	case "%", "<", ">", "<=", ">=", "<=>", "=~":
		return true
	}
	return false
}
