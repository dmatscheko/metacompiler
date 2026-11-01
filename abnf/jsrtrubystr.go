package abnf

import (
	"bytes"
	"encoding/base64"
	"fmt"
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

// rubyIsLazyEnum: v is an Enumerator::Lazy (and not the endless Enumerator of
// Array#cycle, which shares the representation).
func rubyIsLazyEnum(v interface{}) bool {
	o, isObj := v.(*jsObject)
	if !isObj || !rubyIsEnum(o) {
		return false
	}
	lz, _ := o.props["lazy"].(bool)
	return lz
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
	o.set("src", jsNull)
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
	c.set("src", t.props["src"])
	return c
}

func (rt *jsrt) rubyLazyFirst(t *jsObject, n int) []interface{} { return rt.rubyLazyPull(t, n, nil) }

// rubyLazyPull is rbaLazyFirst with its emit argument: non-nil, each result goes
// to emit(v, multi) instead of the answer.
func (rt *jsrt) rubyLazyPull(t *jsObject, n int, emit func(interface{}, bool)) []interface{} {
	ops := t.props["ops"].(*jsArray).elems
	items := t.props["items"].(*jsArray).elems
	kind := t.props["kind"].(string)
	lo := t.props["lo"].(float64)
	hi, bounded := t.props["hi"].(float64)
	excl := t.props["excl"] == true
	st := &rubyLazyState{out: []interface{}{}, n: n, cnt: make([]int, len(ops)), on: make([]bool, len(ops)),
		buf: make([][]interface{}, len(ops)), done: n == 0, stopAt: -1, emit: emit}
	if kind == "each" && !st.done {
		rt.rubyLazyDrive(t.props["src"], func(v interface{}, m bool) bool {
			rt.rubyLazyRun(st, ops, v, m, 0)
			return st.done
		})
	}
	for i := 0; kind != "each" && !st.done; i++ {
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
		rt.rubyLazyRun(st, ops, v, false, 0)
	}
	for j := 0; j < len(ops) && !st.full; j++ {
		if ops[j].(*jsObject).props["k"] == "slice" && len(st.buf[j]) > 0 && j > st.stopAt {
			rest := &jsArray{elems: st.buf[j]}
			st.buf[j] = nil
			rt.rubyLazyRun(st, ops, rest, false, j+1)
		}
	}
	return st.out
}

// rubyLazyDrive runs the #each of a Lazy's user source, handing step each value
// (a multi-value yield packed as an Array, m true), and leaves it through
// catch/throw once step answers true - so `lazy.map { }.first(1)' runs #each only
// up to its first value. Twin of the drive argument of rbaLazyFirst in
// lib/ruby-arr.js.
func (rt *jsrt) rubyLazyDrive(src interface{}, step func(interface{}, bool) bool) {
	tag := newJSObject()
	each := &hostFunc{name: "", fn: func(rt *jsrt, _ uint64, a []interface{}) interface{} {
		var it interface{} = jsNull
		switch len(a) {
		case 0:
		case 1:
			it = a[0]
		default:
			it = &jsArray{elems: append([]interface{}{}, a...)}
		}
		if step(it, len(a) > 1) {
			rt.rubyThrow(tag, jsNull)
		}
		return jsNull
	}}
	rt.rubyCatch(tag, &hostFunc{name: "", fn: func(rt *jsrt, _ uint64, _ []interface{}) interface{} {
		return rt.rubyMethod(src, "each", []interface{}{rubyBlkWrap(each)})
	}})
}

// rubyLazyState is one rubyLazyFirst run. Twin of the st object of rbaLazyFirst.
type rubyLazyState struct {
	out        []interface{}
	n          int
	cnt        []int
	on         []bool
	buf        [][]interface{}
	done, full bool
	stopAt     int
	emit       func(interface{}, bool)
}

func (st *rubyLazyState) stop(j int) {
	st.done = true
	if st.stopAt < 0 {
		st.stopAt = j
	}
}

// rubyLazyCall is an op's block on one value; m: v is a multi-value yield,
// spread over the block. Twin of lazyCall in ruby-interpreter.abnf.
func (rt *jsrt) rubyLazyCall(f, v interface{}, m bool) interface{} {
	if m {
		return rt.call(f, jsUndef, append([]interface{}{}, v.(*jsArray).elems...))
	}
	return rt.call(f, jsUndef, []interface{}{v})
}

// rubyLazyRun is one value (m: a packed multi-value yield) through ops[from:].
// Twin of rbaLazyRun in lib/ruby-arr.js.
func (rt *jsrt) rubyLazyRun(st *rubyLazyState, ops []interface{}, v interface{}, m bool, from int) {
	for j := from; j < len(ops); j++ {
		op := ops[j].(*jsObject)
		f := op.props["f"]
		switch op.props["k"] {
		case "map":
			v = rt.rubyLazyCall(f, v, m)
			m = false
		case "select":
			if !rubyTruthy(rt.rubyLazyCall(f, v, false)) {
				return
			}
		case "reject":
			if rubyTruthy(rt.rubyLazyCall(f, v, false)) {
				return
			}
		case "take_while":
			if !rubyTruthy(rt.rubyLazyCall(f, v, m)) {
				st.stop(j)
				return
			}
		case "drop_while":
			if !st.on[j] {
				if rubyTruthy(rt.rubyLazyCall(f, v, m)) {
					return
				}
				st.on[j] = true
			}
		case "take":
			lim := int(rubyToF(op.props["n"]))
			if lim <= 0 {
				st.stop(j)
				return
			}
			st.cnt[j]++
			if st.cnt[j] >= lim {
				st.stop(j)
			}
		case "slice":
			st.buf[j] = append(st.buf[j], v)
			if len(st.buf[j]) < int(rubyToF(op.props["n"])) {
				return
			}
			v = &jsArray{elems: st.buf[j]}
			st.buf[j] = nil
			m = false
		case "cons":
			lim := int(rubyToF(op.props["n"]))
			st.buf[j] = append(st.buf[j], v)
			if len(st.buf[j]) > lim {
				st.buf[j] = st.buf[j][1:]
			}
			if len(st.buf[j]) < lim {
				return
			}
			v = &jsArray{elems: append([]interface{}{}, st.buf[j]...)}
			m = false
		}
	}
	if st.emit != nil {
		st.emit(v, m)
		return
	}
	st.out = append(st.out, v)
	if st.n >= 0 && len(st.out) >= st.n {
		st.done, st.full = true, true
	}
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
			rt.rubyLazyPull(t, -1, func(v interface{}, m bool) { rt.rubyLazyCall(blk, v, m) })
			return t
		}
	case "lazy":
		if lazy {
			return t
		}
		return rubyLazyCopy(t, nil, true, "#<Enumerator::Lazy: "+desc+">")
	case "each_slice", "each_cons":
		if sn := int(math.Trunc(rubyToF(argAt(pos, 0)))); blk == nil && sn > 0 {
			sd := name + "(" + strconv.Itoa(sn) + ")>"
			if lazy {
				return rubySliceEnum(t, name, sn, true, "#<Enumerator::Lazy: "+desc+":"+sd)
			}
			return rubySliceEnum(t, name, sn, false, "#<Enumerator: "+desc+":"+sd)
		}
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

// rubySliceOp is the op a blockless each_slice / each_cons is on a lazy
// Enumerator, or "". Twin of rbaSliceOp in lib/ruby-arr.js.
func rubySliceOp(name string) string {
	switch name {
	case "each_slice":
		return "slice"
	case "each_cons":
		return "cons"
	}
	return ""
}

// rubySliceEnum is the Enumerator of a blockless each_slice(n) / each_cons(n) on
// t, desc its #inspect. Twin of the arm in rbLazyMethod / rbRangeMethod.
func rubySliceEnum(t *jsObject, name string, n int, lazy bool, desc string) *jsObject {
	op := newJSObject()
	op.set("k", rubySliceOp(name))
	op.set("f", jsNull)
	op.set("n", float64(n))
	return rubyLazyCopy(t, op, lazy, desc)
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
// not nil, wins over sep. It also answers the final pattern search (nil when it
// failed), which MRI leaves in $~. Twin of rbsSplit in lib/ruby-str.js.
func (rt *jsrt) rubySplit(str string, sep *string, re *rxRe, lim int) (*jsArray, *rxMatch) {
	out := &jsArray{}
	var last *rxMatch
	s := []rune(str)
	push := func(a, b int) { out.elems = append(out.elems, string(s[a:b])) }
	if lim == 1 {
		if len(s) > 0 {
			out.elems = append(out.elems, str)
		}
		return out, nil
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
			last = m
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
	return out, last
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
	out, last := rt.rubySplit(s, sep, re, lim)
	if re != nil && lim != 1 {
		if last == nil {
			rt.rxSetGlobals(nil)
		} else {
			rt.rxSetGlobals(rxNewMatchObj(re.src, re.flags, s, last))
		}
	}
	return out
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

// rubySuccWrap maps the largest code point of each UTF-8 length to the smallest
// of that length, where String#succ wraps and carries. Twin of rbsSuccWrap in
// lib/ruby-str.js.
var rubySuccWrap = map[rune]rune{0x7F: 0, 0x7FF: 0x80, 0xFFFF: 0x800, 0x10FFFF: 0x10000}

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
	for j := len(s) - 1; j >= 0; j-- {
		if w, at := rubySuccWrap[s[j]]; at {
			s[j] = w
			continue
		}
		if s[j] == 0xD7FF {
			s[j] = 0xE000
		} else {
			s[j]++
		}
		return string(s)
	}
	return "\x01" + string(s)
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

// rubyHashIterates: Hash#name called with a block runs inside an iteration over
// the Hash. Twin of hashIterates in ruby-interpreter.abnf, which states the rule.
func rubyHashIterates(name string, args []interface{}) bool {
	switch name {
	case "each", "each_pair", "each_key", "each_value", "transform_values", "transform_keys",
		"transform_values!", "to_h":
	case "sort":
		return false
	default:
		if !rubyHashEnumName(name) {
			return false
		}
	}
	pos, blk := rubyBlockArgs(args)
	return blk != nil && !((name == "min" || name == "max") && len(pos) > 0)
}

// rubyHashIter runs Hash d's method name, or Enumerator recv's over d, with d's
// iteration count raised for the duration. Twin of hashIter in
// ruby-interpreter.abnf.
func (rt *jsrt) rubyHashIter(d, recv *jsObject, name string, args []interface{}) interface{} {
	lev, _ := d.props["__iter"].(float64)
	d.set("__iter", lev+1)
	defer d.set("__iter", lev)
	if recv != d {
		return rt.rubyEnumMethod(recv, name, args)
	}
	keys, vals, _ := dictParts(d)
	if name == "each" {
		_, blk := rubyBlockArgs(args)
		rt.rubyHashEach(keys, vals, blk)
		return d
	}
	v, _ := rt.rubyHashMethod(d, keys, vals, name, args)
	return v
}

// rubyHashAddCheck refuses a key Hash d does not hold yet while an iteration
// over it runs. Twin of hashPut in ruby-interpreter.abnf.
func (rt *jsrt) rubyHashAddCheck(d *jsObject, frame string) {
	if lev, _ := d.props["__iter"].(float64); lev > 0 {
		e := rt.rubyExc(rt.rubyBuiltinClass("RuntimeError"), []interface{}{"can't add a new key into hash during iteration"})
		e.set("__rframe", frame)
		panic(&jsThrown{value: e})
	}
}

// rubyHashEach is Hash#each / #each_pair. An entry deleted before it is reached is
// not yielded. Twin of hashEach in ruby-interpreter.abnf.
func (rt *jsrt) rubyHashEach(keys, vals *jsArray, blk interface{}) {
	for i := 0; i < len(keys.elems); {
		k := keys.elems[i]
		rt.call(blk, jsUndef, []interface{}{rt.rubyKeyOut(k), vals.elems[i]})
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

// ----- Strings in another encoding -----

// jsRBin is a Ruby String whose encoding is not UTF-8, or whose UTF-8 bytes are
// invalid, holding its bytes as they are: enc is "" for ASCII-8BIT (binary:
// Integer#chr past 127, String#b, Array#pack), else "US-ASCII" (Integer#chr
// below 128, unpack("B") / ("H")) or "UTF-8" (a "\xFF" literal). A pointer, so
// << and force_encoding change it in place. Twin of the {__rbin} box of
// ruby-interpreter.abnf and lib/ruby-rt.metajs, which hold one char per byte.
//
// force_encoding to bytes that are valid UTF-8 makes it the ordinary String
// str: the other engines retag the box, and a Go value cannot change its type,
// so every alias reaches str through this one (rubyFwd).
type jsRBin struct {
	b   string
	enc string
	id  jsHandleID
	str *jsRStr
	fz  bool
}

func (t *jsRBin) encName() string {
	if t.enc == "" {
		return "ASCII-8BIT"
	}
	return t.enc
}

// rubyEncStr is the String of the bytes b in encoding enc: UTF-8 bytes that
// are valid are an ordinary String. Twin of encStr in ruby-interpreter.abnf.
func rubyEncStr(b, enc string) interface{} {
	switch enc {
	case "ASCII-8BIT":
		return &jsRBin{b: b}
	case "UTF-8":
		if utf8.ValidString(b) {
			return &jsRStr{s: b}
		}
	}
	return &jsRBin{b: b, enc: enc}
}

// rubyStrBE is a String value's bytes and encoding, for the rules that mix
// encodings; anything else converts through to_str (or raises TypeError).
func (rt *jsrt) rubyStrBE(v interface{}) (string, string) {
	switch t := v.(type) {
	case *jsRBin:
		if t.str != nil {
			return rt.rubyStrBE(t.str)
		}
		return t.b, t.encName()
	case *jsRStr:
		if t.bin != nil {
			return t.bin.b, t.bin.encName()
		}
		return t.s, "UTF-8"
	case string:
		return t, "UTF-8"
	}
	return rt.rubyStrOperand(v), "UTF-8"
}

// rubyBinText is the text the other engines hold for the bytes b: one char per
// byte, which is what the String methods a box delegates run on.
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

// rubyAsciiCase maps the ASCII letters of the bytes b only. Twin of
// rbsAsciiCase in ruby-str.js.
func rubyAsciiCase(b, name string) string {
	out := []byte(b)
	for i, c := range out {
		up := name == "upcase" || (name == "capitalize" && i == 0)
		down := name == "downcase" || (name == "capitalize" && i > 0)
		if (up || name == "swapcase") && c >= 'a' && c <= 'z' {
			out[i] = c - 32
		} else if (down || name == "swapcase") && c >= 'A' && c <= 'Z' {
			out[i] = c + 32
		}
	}
	return string(out)
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

// rubyBoxInspect is a box's #inspect: an invalid UTF-8 String shows its valid
// characters as text and every other byte as \xHH. Twin of binInspect in
// ruby-interpreter.abnf.
func rubyBoxInspect(t *jsRBin) string {
	if t.enc != "UTF-8" {
		return rubyBinInspect(t.b)
	}
	var sb strings.Builder
	sb.WriteByte('"')
	for i := 0; i < len(t.b); {
		j := i
		for j < len(t.b) {
			r, n := utf8.DecodeRuneInString(t.b[j:])
			if r == utf8.RuneError && n <= 1 {
				break
			}
			j += n
		}
		var q string
		if j > i {
			q = rubyStrInspect(t.b[i:j])
			i = j
		} else {
			q = rubyBinInspect(t.b[i : i+1])
			i++
		}
		sb.WriteString(q[1 : len(q)-1])
	}
	sb.WriteByte('"')
	return sb.String()
}

// rubyU8Len counts the characters of UTF-8 bytes b, a byte that starts no valid
// character as one. Twin of rbsU8Len in ruby-str.js.
func rubyU8Len(b string) int {
	n := 0
	for i := 0; i < len(b); n++ {
		r, k := utf8.DecodeRuneInString(b[i:])
		if r == utf8.RuneError && k <= 1 {
			k = 1
		}
		i += k
	}
	return n
}

// rubyPackDir is one Array#pack / String#unpack directive: its letter, count
// (-1 for `*'), byte width (0 for a String or position directive), byte order
// and sign. Twin of rbsPackDirs in ruby-str.js, which states the set.
type rubyPackDir struct {
	d      byte
	n      int
	size   int
	be     bool
	signed bool
}

func rubyPackSize(d byte) int {
	switch d {
	case 'C', 'c':
		return 1
	case 'S', 's', 'n', 'v':
		return 2
	case 'L', 'l', 'N', 'V', 'I', 'i':
		return 4
	case 'Q', 'q', 'J', 'j', 'E', 'd', 'D', 'G':
		return 8
	case 'e', 'f', 'F', 'g':
		return 4
	case 'U', 'a', 'A', 'Z', 'B', 'b', 'H', 'h', 'm', 'x', 'X', '@':
		return 0
	}
	return -1
}

func (rt *jsrt) rubyPackDirs(name, f string) []rubyPackDir {
	var out []rubyPackDir
	for i := 0; i < len(f); {
		d := f[i]
		i++
		if d == ' ' || d == '\t' || d == '\n' {
			continue
		}
		size := rubyPackSize(d)
		if size < 0 {
			rt.rubyRaiseErr(name, "NotImplementedError", name+" directive '"+string(rune(d))+"' is not supported")
		}
		be := d == 'n' || d == 'N' || d == 'g' || d == 'G'
		signed := strings.IndexByte("cslqji", d) >= 0
		for i < len(f) && strings.IndexByte("_!<>", f[i]) >= 0 {
			if strings.IndexByte("sSiIlLqQjJ", d) < 0 {
				rt.rubyRaiseErr(name, "ArgumentError", "'"+string(rune(f[i]))+"' allowed only after types sSiIlLqQjJ")
			}
			switch f[i] {
			case '<':
				be = false
			case '>':
				be = true
			default:
				if d == 'l' || d == 'L' {
					size = 8
				}
			}
			i++
		}
		n := 1
		if d == 'm' {
			n = 45
		}
		if i < len(f) && f[i] == '*' {
			n = -1
			i++
		} else if i < len(f) && f[i] >= '0' && f[i] <= '9' {
			n = 0
			for i < len(f) && f[i] >= '0' && f[i] <= '9' {
				n = n*10 + int(f[i]-'0')
				i++
			}
		}
		out = append(out, rubyPackDir{d, n, size, be, signed})
	}
	return out
}

// rubyPackInt is a pack value as an Integer: an Integer, or a Float truncated.
func rubyPackInt(v interface{}) (*big.Int, bool) {
	switch t := v.(type) {
	case float64:
		return big.NewInt(int64(t)), true
	case *jsBigInt:
		return t.v, true
	case jsFlo:
		i, _ := big.NewFloat(math.Trunc(rubyToF(t))).Int(nil)
		return i, true
	}
	return nil, false
}

func rubyFloDir(d byte) bool { return strings.IndexByte("eEgGdDfF", d) >= 0 }

// rubyFloBytes is a float directive's bytes for the pack value v: IEEE 754
// binary32 or binary64. A value past the largest binary32 is an infinity there
// (MRI's VALUE_to_float) and a NaN is the quiet one with the sign clear. Twin
// of rbsFloBytes in ruby-str.js.
func (rt *jsrt) rubyFloBytes(v interface{}, d rubyPackDir) []byte {
	var x float64
	switch t := v.(type) {
	case float64:
		x = t
	case jsFlo:
		x = t.f
	case *jsBigInt:
		x, _ = new(big.Float).SetInt(t.v).Float64()
	default:
		rt.rubyRaiseErr("pack", "TypeError", "can't convert "+rubyFmtConvName(v)+" into Float")
	}
	out := make([]byte, d.size)
	var bits uint64
	if d.size == 4 {
		f := float32(x)
		switch {
		case math.IsNaN(x):
			bits = 0x7FC00000
		case x > math.MaxFloat32:
			bits = uint64(math.Float32bits(float32(math.Inf(1))))
		case x < -math.MaxFloat32:
			bits = uint64(math.Float32bits(float32(math.Inf(-1))))
		default:
			bits = uint64(math.Float32bits(f))
		}
	} else if math.IsNaN(x) {
		bits = 0x7FF8000000000000
	} else {
		bits = math.Float64bits(x)
	}
	for q := 0; q < d.size; q++ {
		c := byte(bits >> (8 * q))
		if d.be {
			out[d.size-1-q] = c
		} else {
			out[q] = c
		}
	}
	return out
}

// rubyFloOf is the Float the bytes b of a float directive spell. Twin of
// rbsFloOf in ruby-str.js.
func rubyFloOf(b string, be bool) float64 {
	var bits uint64
	for q := 0; q < len(b); q++ {
		c := b[q]
		if be {
			c = b[len(b)-1-q]
		}
		bits |= uint64(c) << (8 * q)
	}
	if len(b) == 4 {
		return float64(math.Float32frombits(uint32(bits)))
	}
	return math.Float64frombits(bits)
}

// rubyIntBytes is the k little-endian bytes of x modulo 2^(8k), two's
// complement for a negative one. Twin of rbsIntBytes in ruby-str.js.
func rubyIntBytes(x *big.Int, k int) []byte {
	m := new(big.Int).Lsh(big.NewInt(1), uint(8*k))
	be := new(big.Int).Mod(x, m).Bytes()
	out := make([]byte, k)
	for i := 0; i < len(be); i++ {
		out[i] = be[len(be)-1-i]
	}
	return out
}

// rubyPack is Array#pack: a binary String, or a UTF-8 one when the first
// directive is U. Twin of rbsPack in ruby-str.js.
func (rt *jsrt) rubyPack(vals []interface{}, f string) interface{} {
	dirs := rt.rubyPackDirs("pack", f)
	var out []byte
	k := 0
	next := func() interface{} {
		if k >= len(vals) {
			rt.rubyRaiseErr("pack", "ArgumentError", "too few arguments")
		}
		k++
		return vals[k-1]
	}
	for _, d := range dirs {
		n := d.n
		if n < 0 && (d.d == 'x' || d.d == 'X' || d.d == '@') {
			n = 0
		}
		switch {
		case d.d == 'x':
			out = append(out, make([]byte, n)...)
		case d.d == 'X':
			if n > len(out) {
				rt.rubyRaiseErr("pack", "ArgumentError", "X outside of string")
			}
			out = out[:len(out)-n]
		case d.d == '@':
			if n <= len(out) {
				out = out[:n]
			} else {
				out = append(out, make([]byte, n-len(out))...)
			}
		case d.size == 0 && d.d != 'U':
			v := next()
			var s string
			switch t := v.(type) {
			case string, *jsRStr, *jsRBin:
				s, _ = rt.rubyStrBE(t)
			default:
				rt.rubyRaiseErr("pack", "TypeError", "no implicit conversion of "+rubyFmtConvName(v)+" into String")
			}
			out = append(out, rubyPackStr(d.d, d.n, s)...)
		default:
			if n < 0 {
				n = len(vals) - k
			}
			for j := 0; j < n; j++ {
				v := next()
				if rubyFloDir(d.d) {
					out = append(out, rt.rubyFloBytes(v, d)...)
					continue
				}
				x, ok := rubyPackInt(v)
				if !ok {
					rt.rubyRaiseErr("pack", "TypeError", "no implicit conversion of "+rubyFmtConvName(v)+" into Integer")
				}
				if d.d == 'U' {
					c := x.Int64()
					if !x.IsInt64() || c < 0 || c > 0x7FFFFFFF {
						rt.rubyRaiseErr("pack", "RangeError", "pack(U): value out of range")
					}
					out = append(out, rubyU8Enc(c)...)
					continue
				}
				bs := rubyIntBytes(x, d.size)
				for q := 0; q < d.size; q++ {
					if d.be {
						out = append(out, bs[d.size-1-q])
					} else {
						out = append(out, bs[q])
					}
				}
			}
		}
	}
	enc := "US-ASCII"
	for _, d := range dirs {
		if d.d == 'U' {
			if enc == "US-ASCII" {
				enc = "UTF-8"
			}
		} else if d.d != 'm' {
			enc = "ASCII-8BIT"
		}
	}
	return rubyEncStr(string(out), enc)
}

// rubyU8Enc is the bytes pack(U) writes for cp: UTF-8's scheme extended to six
// bytes. Twin of rbsU8Enc in ruby-str.js.
func rubyU8Enc(cp int64) []byte {
	if cp < 128 {
		return []byte{byte(cp)}
	}
	n, lead := 6, int64(252)
	switch {
	case cp < 2048:
		n, lead = 2, 192
	case cp < 65536:
		n, lead = 3, 224
	case cp < 2097152:
		n, lead = 4, 240
	case cp < 67108864:
		n, lead = 5, 248
	}
	out := make([]byte, n)
	for k := n - 1; k > 0; k-- {
		out[k] = byte(128 + cp%64)
		cp /= 64
	}
	out[0] = byte(lead + cp)
	return out
}

// rubyPackStr is one String directive of pack over the bytes s. Twin of
// rbsPackStr in ruby-str.js.
func rubyPackStr(d byte, n int, s string) []byte {
	switch d {
	case 'a', 'A', 'Z':
		if n < 0 {
			if d == 'Z' {
				return append([]byte(s), 0)
			}
			return []byte(s)
		}
		if len(s) >= n {
			return []byte(s[:n])
		}
		pad := byte(0)
		if d == 'A' {
			pad = ' '
		}
		return append([]byte(s), bytes.Repeat([]byte{pad}, n-len(s))...)
	case 'm':
		per := 45
		if n == 0 {
			per = 0
		} else if n >= 3 {
			per = n - n%3
		}
		enc := base64.StdEncoding
		var sb strings.Builder
		for i := 0; i < len(s); {
			end := len(s)
			if per > 0 && i+per < end {
				end = i + per
			}
			sb.WriteString(enc.EncodeToString([]byte(s[i:end])))
			if per > 0 {
				sb.WriteByte('\n')
			}
			i = end
		}
		return []byte(sb.String())
	}
	cnt := n
	if n < 0 {
		cnt = len(s)
	}
	var out []byte
	if d == 'B' || d == 'b' {
		acc, bits := 0, 0
		for i := 0; i < cnt; i++ {
			bit := 0
			if i < len(s) {
				bit = int(s[i]) % 2
			}
			if d == 'B' {
				acc = acc*2 + bit
			} else {
				acc += bit << bits
			}
			bits++
			if bits == 8 {
				out = append(out, byte(acc))
				acc, bits = 0, 0
			}
		}
		if bits > 0 {
			if d == 'B' {
				acc <<= 8 - bits
			}
			out = append(out, byte(acc))
		}
		return out
	}
	acc, half := 0, false
	for h := 0; h < cnt; h++ {
		nib := 0
		if h < len(s) {
			c := int(s[h])
			if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
				nib = (c%8 + 9) % 16
			} else {
				nib = c % 16
			}
		}
		switch {
		case !half && d == 'H':
			acc = nib * 16
		case !half:
			acc = nib
		case d == 'H':
			acc += nib
			out = append(out, byte(acc))
		default:
			acc += nib * 16
			out = append(out, byte(acc))
		}
		half = !half
	}
	if half {
		out = append(out, byte(acc))
	}
	return out
}

// rubyUnpack is String#unpack / #unpack1 of the bytes b. Twin of rbsUnpack in
// ruby-str.js.
func (rt *jsrt) rubyUnpack(b, name string, args []interface{}) interface{} {
	dirs := rt.rubyPackDirs(name, rt.rubyStr(argAt(args, 0)))
	out := &jsArray{}
	p := 0
	for _, d := range dirs {
		n := d.n
		switch {
		case d.d == 'x' || d.d == 'X' || d.d == '@':
			c := n
			if c < 0 {
				c = 0
				if d.d == 'x' {
					c = len(b) - p
				}
			}
			to := c
			if d.d == 'x' {
				to = p + c
			} else if d.d == 'X' {
				to = p - c
			}
			if to < 0 || to > len(b) {
				rt.rubyRaiseErr(name, "ArgumentError", string(rune(d.d))+" outside of string")
			}
			p = to
		case d.size == 0 && d.d != 'U':
			var v interface{}
			v, p = rubyUnpackStr(d.d, n, b, p)
			out.elems = append(out.elems, v)
		default:
			w := d.size
			if d.d == 'U' {
				w = 1
			}
			for j := 0; (n < 0 && p+w <= len(b)) || (n >= 0 && j < n); j++ {
				switch {
				case d.d == 'U':
					if p >= len(b) {
						j = n
						continue
					}
					r, size := utf8.DecodeRuneInString(b[p:])
					if r == utf8.RuneError && size <= 1 {
						rt.rubyRaiseErr(name, "ArgumentError", "malformed UTF-8 character")
					}
					out.elems = append(out.elems, float64(r))
					p += size
				case p+d.size > len(b):
					out.elems = append(out.elems, jsNull)
					p = len(b)
				case rubyFloDir(d.d):
					out.elems = append(out.elems, jsFlo{f: rubyFloOf(b[p:p+d.size], d.be)})
					p += d.size
				default:
					out.elems = append(out.elems, rt.rubyIntOf(b[p:p+d.size], d.be, d.signed))
					p += d.size
				}
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

// rubyIntOf is the Integer the bytes b spell. Twin of rbsIntOf in ruby-str.js.
func (rt *jsrt) rubyIntOf(b string, be, signed bool) interface{} {
	x := new(big.Int)
	for q := 0; q < len(b); q++ {
		c := b[len(b)-1-q]
		if be {
			c = b[q]
		}
		x.Lsh(x, 8)
		x.Or(x, big.NewInt(int64(c)))
	}
	if signed && x.Bit(8*len(b)-1) == 1 {
		x.Sub(x, new(big.Int).Lsh(big.NewInt(1), uint(8*len(b))))
	}
	return rt.rubyBigNarrow(x)
}

// rubyUnpackStr is one String directive of unpack at byte p: the String and the
// next position. Twin of rbsUnpackStr in ruby-str.js.
func rubyUnpackStr(d byte, n int, b string, p int) (interface{}, int) {
	rest := len(b) - p
	switch d {
	case 'a', 'A', 'Z':
		cnt := n
		if n < 0 || n > rest {
			cnt = rest
		}
		s := b[p : p+cnt]
		next := p + cnt
		if d == 'Z' {
			if z := strings.IndexByte(s, 0); z >= 0 {
				s = s[:z]
				if n < 0 {
					next = p + z + 1
				}
			}
		} else if d == 'A' {
			s = strings.TrimRight(s, " \x00")
		}
		return &jsRBin{b: s}, next
	case 'm':
		var clean []byte
		for i := p; i < len(b); i++ {
			c := b[i]
			if c == '=' {
				break
			}
			if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '+' || c == '/' {
				clean = append(clean, c)
			}
		}
		dec, _ := base64.RawStdEncoding.DecodeString(string(clean[:len(clean)-len(clean)%4+rubyB64Tail(len(clean)%4)]))
		return &jsRBin{b: string(dec)}, len(b)
	}
	var sb strings.Builder
	if d == 'B' || d == 'b' {
		bits := n
		if n < 0 || n > rest*8 {
			bits = rest * 8
		}
		for i := 0; i < bits; i++ {
			by := b[p+i/8]
			sh := uint(i % 8)
			if d == 'B' {
				sh = 7 - sh
			}
			sb.WriteByte('0' + (by>>sh)&1)
		}
		return &jsRBin{b: sb.String(), enc: "US-ASCII"}, p + (bits+7)/8
	}
	const hx = "0123456789abcdef"
	nibs := n
	if n < 0 || n > rest*2 {
		nibs = rest * 2
	}
	for h := 0; h < nibs; h++ {
		hb := b[p+h/2]
		if (h%2 == 0) == (d == 'H') {
			sb.WriteByte(hx[hb>>4])
		} else {
			sb.WriteByte(hx[hb&15])
		}
	}
	return &jsRBin{b: sb.String(), enc: "US-ASCII"}, p + (nibs+1)/2
}

// rubyB64Tail keeps the sextets of a final partial base64 group that still make
// a whole byte: one leftover sextet makes none.
func rubyB64Tail(r int) int {
	if r == 1 {
		return 0
	}
	return r
}

// rubyEncArg is the encoding a force_encoding argument (a name or an Encoding)
// selects: ASCII-8BIT, US-ASCII or UTF-8. Twin of encArgName in
// ruby-interpreter.abnf.
func (rt *jsrt) rubyEncArg(e interface{}) string {
	if o, isObj := e.(*jsObject); isObj {
		if n, isEnc := o.props["__renc"].(string); isEnc {
			return n
		}
	}
	switch strings.ToUpper(rt.rubyStr(e)) {
	case "BINARY", "ASCII-8BIT":
		return "ASCII-8BIT"
	case "US-ASCII", "ASCII":
		return "US-ASCII"
	case "UTF-8":
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

// rubyBinCat is String#+ / #<< / interpolation where a side is a box: the bytes
// join in the left side's encoding when the right is ASCII-only, in the right
// side's when the left is, in their common one otherwise; two non-ASCII sides
// of two encodings are MRI's Encoding::CompatibilityError. Twin of binCat in
// ruby-interpreter.abnf.
func (rt *jsrt) rubyBinCat(l, r interface{}) interface{} {
	l, r = rubyFwd(l), rubyFwd(r)
	// An ASCII-only box on the right leaves a UTF-8 left side as it is.
	if b, isBox := r.(*jsRBin); isBox && rubyAsciiOnly(b.b) {
		if _, lBox := l.(*jsRBin); !lBox {
			return &jsRStr{s: rt.rubyStr(l) + b.b}
		}
	}
	lb, le := rt.rubyStrBE(l)
	rb, re := rt.rubyStrBE(r)
	enc := le
	if !rubyAsciiOnly(rb) {
		if rubyAsciiOnly(lb) {
			enc = re
		} else if le != re {
			rt.rubyRaiseErr("+", "Encoding::CompatibilityError", "incompatible character encodings: "+le+" and "+re)
		}
	}
	return rubyEncStr(lb+rb, enc)
}

// rubyBinEq is == / eql? with the box b on one side: the same bytes, in the same
// encoding unless they are ASCII-only. Twin of binEq in ruby-interpreter.abnf.
func rubyBinEq(b *jsRBin, o interface{}) bool {
	var ob, oe string
	switch t := rubyFwd(o).(type) {
	case *jsRBin:
		ob, oe = t.b, t.encName()
	case *jsRStr:
		ob, oe = t.s, "UTF-8"
		if t.bin != nil {
			ob, oe = t.bin.b, t.bin.encName()
		}
	case string:
		ob, oe = t, "UTF-8"
	default:
		return false
	}
	return ob == b.b && (oe == b.encName() || rubyAsciiOnly(ob))
}

// rubyU8Bad is the invalid UTF-8 sequence at the start of b as MRI's transcoder
// delimits it: its length, and 0 for a byte that starts no character, 1 for a
// sequence b ends inside, 2 for one the next byte breaks. Twin of rbsU8Bad in
// ruby-str.js.
func rubyU8Bad(b string) (int, int) {
	c := b[0]
	n := 0
	switch {
	case c >= 0xC2 && c <= 0xDF:
		n = 2
	case c >= 0xE0 && c <= 0xEF:
		n = 3
	case c >= 0xF0 && c <= 0xF4:
		n = 4
	default:
		return 1, 0
	}
	for k := 1; k < n; k++ {
		if k >= len(b) {
			return k, 1
		}
		lo, hi := byte(0x80), byte(0xBF)
		if k == 1 {
			switch c {
			case 0xE0:
				lo = 0xA0
			case 0xED:
				hi = 0x9F
			case 0xF0:
				lo = 0x90
			case 0xF4:
				hi = 0x8F
			}
		}
		if b[k] < lo || b[k] > hi {
			return k, 2
		}
	}
	return n, 2
}

// rubyEncodeBytes is String#encode of the bytes b of encoding src into dst over
// UTF-8, US-ASCII and ASCII-8BIT; inv / undef replace an invalid sequence / a
// character dst has no code for, and nil raises MRI's error. Twin of rbsEncode
// in ruby-str.js.
func (rt *jsrt) rubyEncodeBytes(name, b, src, dst string, inv, undef *string) string {
	if src == dst && inv == nil {
		return b
	}
	var sb strings.Builder
	for i := 0; i < len(b); {
		c := b[i]
		switch {
		case c < 128:
			sb.WriteByte(c)
			i++
		case src == "ASCII-8BIT":
			if dst == "ASCII-8BIT" {
				sb.WriteByte(c)
			} else if undef != nil {
				sb.WriteString(*undef)
			} else {
				rt.rubyRaiseErr(name, "Encoding::UndefinedConversionError", rubyBinInspect(b[i:i+1])+" from ASCII-8BIT to "+dst)
			}
			i++
		case src == "US-ASCII":
			if inv == nil {
				rt.rubyRaiseErr(name, "Encoding::InvalidByteSequenceError", rubyBinInspect(b[i:i+1])+" on US-ASCII")
			}
			sb.WriteString(*inv)
			i++
		default:
			r, n := utf8.DecodeRuneInString(b[i:])
			if r != utf8.RuneError || n > 1 {
				switch {
				case dst == "UTF-8":
					sb.WriteString(b[i : i+n])
				case undef != nil:
					sb.WriteString(*undef)
				default:
					rt.rubyRaiseErr(name, "Encoding::UndefinedConversionError", fmt.Sprintf("U+%04X from UTF-8 to %s", r, dst))
				}
				i += n
				continue
			}
			n, kind := rubyU8Bad(b[i:])
			if inv == nil {
				q := rubyBinInspect(b[i : i+n])
				switch kind {
				case 1:
					q = "incomplete " + q
				case 2:
					q += " followed by " + rubyBinInspect(b[i+n:i+n+1])
				}
				rt.rubyRaiseErr(name, "Encoding::InvalidByteSequenceError", q+" on UTF-8")
			}
			sb.WriteString(*inv)
			i += n
		}
	}
	return sb.String()
}

// rubyStrEncode is String#encode(dst = enc, src = enc, invalid: :replace,
// undef: :replace, replace: str), and #scrub(rep) with no block, of the bytes b
// in encoding enc. Twin of strEncode in ruby-interpreter.abnf.
func (rt *jsrt) rubyStrEncode(b, enc, name string, pos []interface{}) interface{} {
	var keys, vals *jsArray
	if len(pos) > 0 {
		if k, v, isDict := dictParts(pos[len(pos)-1]); isDict {
			keys, vals = k, v
			pos = pos[:len(pos)-1]
		}
	}
	u8rep := "\uFFFD"
	if name == "scrub" {
		rep := "?"
		if enc == "UTF-8" {
			rep = u8rep
		}
		if len(pos) > 0 && pos[0] != jsNull {
			rep, _ = rt.rubyStrBE(pos[0])
		}
		return rubyEncStr(rt.rubyEncodeBytes(name, b, enc, enc, &rep, nil), enc)
	}
	dst, src := enc, enc
	if len(pos) > 0 {
		dst = rt.rubyEncodeName(pos[0])
	}
	if len(pos) > 1 {
		src = rt.rubyEncodeName(pos[1])
	}
	invR, undefR := false, false
	rep := "?"
	if dst == "UTF-8" {
		rep = u8rep
	}
	if keys != nil {
		for k, kv := range keys.elems {
			on := ""
			if sym, isSym := kv.(jsSym); isSym {
				on = sym.s
			}
			sv, isSym := vals.elems[k].(jsSym)
			isRep := isSym && sv.s == "replace"
			switch on {
			case "invalid":
				invR = isRep
			case "undef":
				undefR = isRep
			case "replace":
				rep, _ = rt.rubyStrBE(vals.elems[k])
			default:
				rt.rubyRaiseErr(name, "NotImplementedError", "String#encode option "+rt.rubyInspect(kv)+" is not supported")
			}
		}
	}
	if (invR || undefR) && dst != "UTF-8" && !rubyAsciiOnly(rep) {
		rt.rubyRaiseErr(name, "Encoding::ConverterNotFoundError", "code converter not found (UTF-8 to "+dst+")")
	}
	var inv, undef *string
	if invR {
		inv = &rep
	}
	if undefR {
		undef = &rep
	}
	return rubyEncStr(rt.rubyEncodeBytes(name, b, src, dst, inv, undef), dst)
}

// rubyEncodeName is an encoding String#encode names; the ones no engine models
// are NotImplementedError. Twin of encodeName in ruby-interpreter.abnf.
func (rt *jsrt) rubyEncodeName(e interface{}) string {
	if o, isObj := e.(*jsObject); isObj {
		if n, has := o.props["__renc"].(string); has {
			return n
		}
	}
	switch strings.ToUpper(rt.rubyStr(e)) {
	case "BINARY", "ASCII-8BIT":
		return "ASCII-8BIT"
	case "US-ASCII", "ASCII":
		return "US-ASCII"
	case "UTF-8":
		return "UTF-8"
	}
	rt.rubyRaiseErr("encode", "NotImplementedError", "encoding "+rt.rubyStr(e)+" is not supported")
	return ""
}

// rubyBinMethod is a method of a box: the ones that read its bytes here, every
// other String method on its text, a String it answers in its encoding. Twin
// of binMethod in ruby-interpreter.abnf.
func (rt *jsrt) rubyBinMethod(t *jsRBin, name string, args []interface{}) interface{} {
	pos, blk := rubyBlockArgs(args)
	if t.str != nil {
		if name == "equal?" {
			return argAt(pos, 0) == interface{}(t)
		}
		var r interface{}
		if len(pos) == 1 && (name == "+" || name == "*" || rubyBinOpName(name)) {
			r = rt.rubyBin(name, t.str, pos[0])
		} else {
			r = rt.rubyMethod(t.str, name, args)
		}
		if r == interface{}(t.str) {
			return t
		}
		return r
	}
	enc := t.encName()
	// Twin of the frozen arms of binMethod in ruby-interpreter.abnf.
	if name == "freeze" {
		t.fz = true
		return t
	}
	if name == "frozen?" {
		return t.fz
	}
	if t.fz && (rubyStrMutator(name) || name == "force_encoding") {
		rt.rubyRaiseErr(name, "FrozenError", "can't modify frozen String")
	}
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
		return rubyBoxInspect(t)
	case "encoding":
		return rt.rubyEncoding(enc)
	case "force_encoding":
		// In place; bytes that become valid UTF-8 are an ordinary String, a new one.
		v := rubyEncStr(t.b, rt.rubyEncArg(argAt(pos, 0)))
		if nb, isBox := v.(*jsRBin); isBox {
			t.enc = nb.enc
		} else {
			t.str = v.(*jsRStr)
		}
		return t
	case "b":
		return &jsRBin{b: t.b}
	case "dup", "clone", "+@", "-@":
		if (name == "+@" && !t.fz) || (name == "-@" && t.fz) {
			return t
		}
		cp := rubyEncStr(t.b, enc)
		rubyStrSetFz(cp, name == "-@" || (name == "clone" && t.fz))
		return cp
	case "valid_encoding?":
		return enc == "ASCII-8BIT" || (enc == "US-ASCII" && rubyAsciiOnly(t.b))
	case "ascii_only?":
		return rubyAsciiOnly(t.b)
	case "size", "length":
		if enc == "UTF-8" {
			return float64(rubyU8Len(t.b))
		}
		return float64(len(t.b))
	case "to_s", "to_str", "itself":
		return t
	case "equal?":
		return argAt(pos, 0) == interface{}(t)
	case "<<", "concat":
		for _, a := range pos {
			if f, isNum := a.(float64); isNum {
				a = &jsRBin{b: string([]byte{byte(int64(f) & 255)})}
			}
			t.b, t.enc = rt.rubyStrBE(rt.rubyBinCat(t, a))
			if t.enc == "ASCII-8BIT" {
				t.enc = ""
			}
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
		return rubyEncStr(strings.Repeat(t.b, n), enc)
	case "unpack", "unpack1":
		return rt.rubyUnpack(t.b, name, pos)
	case "encode":
		return rt.rubyStrEncode(t.b, enc, name, pos)
	case "scrub":
		if blk == nil {
			return rt.rubyStrEncode(t.b, enc, name, pos)
		}
	}
	// Twin of the case-mapping arm of binMethod in ruby-interpreter.abnf.
	switch cn := strings.TrimSuffix(name, "!"); cn {
	case "upcase", "downcase", "swapcase", "capitalize":
		if enc == "UTF-8" {
			rt.rubyRaiseErr(name, "ArgumentError", "input string invalid")
		}
		cb := rubyAsciiCase(t.b, cn)
		if cn == name {
			return rubyEncStr(cb, enc)
		}
		if cb == t.b {
			return jsNull
		}
		t.b = cb
		return t
	default:
		if cn != name && name != "slice!" {
			bt := rubyBinText(t.b)
			if (name == "sub!" || name == "gsub!") && rt.rubySubMisses(bt, args) {
				return jsNull
			}
			r := rt.rubyStrCall(bt, name, args)
			if isUndefOrNull(r) {
				return jsNull
			}
			t.b = rubyBinOfText(rt.rubyStr(r))
			return t
		}
	}
	var r interface{}
	if rubyBinOpName(name) {
		r = rt.rubyBin(name, rubyBinText(t.b), rubySV(argAt(pos, 0)))
	} else {
		r = rt.rubyMethod(rubyBinText(t.b), name, args)
	}
	switch v := r.(type) {
	case string:
		return rubyEncStr(rubyBinOfText(v), enc)
	case *jsRStr:
		return rubyEncStr(rubyBinOfText(v.s), enc)
	case *jsArray:
		for i, e := range v.elems {
			switch s := e.(type) {
			case string:
				v.elems[i] = rubyEncStr(rubyBinOfText(s), enc)
			case *jsRStr:
				v.elems[i] = rubyEncStr(rubyBinOfText(s.s), enc)
			}
		}
	}
	return r
}

// rubyStrSetFz sets the frozen flag of a String rubyEncStr built.
func rubyStrSetFz(v interface{}, fz bool) {
	switch t := v.(type) {
	case *jsRBin:
		t.fz = fz
	case *jsRStr:
		t.fz = fz
	}
}

// rubyBinOpName: the operators a binary String reaches through rubyBin.
func rubyBinOpName(n string) bool {
	switch n {
	case "%", "<", ">", "<=", ">=", "<=>", "=~":
		return true
	}
	return false
}

// rubyEnumSpreads names the Enumerable methods whose block MRI 2.6 hands a
// multi-value `yield a, b` of #each as the values themselves rather than packed
// into one Array. Twin of rbaEnumSpreads in languages/lib/ruby-arr.js.
func rubyEnumSpreads(n string) bool {
	switch n {
	case "map", "collect", "flat_map", "collect_concat", "find_index", "all?", "any?",
		"none?", "one?", "count", "take_while", "uniq", "to_h":
		return true
	}
	return false
}

// rubyEnumStop is how an Enumerable method ends #each early: 1 at the first
// element whose test holds, 2 at the first whose test fails, 3 at the second
// that holds, 4 after a count, 0 never. Twin of rbaEnumStop in
// languages/lib/ruby-arr.js, which names the methods.
func rubyEnumStop(n string, hasBlk bool, nargs int) int {
	switch n {
	case "first", "take":
		return 4
	case "any?", "none?", "include?", "member?":
		return 1
	case "all?":
		return 2
	case "one?":
		return 3
	}
	if !hasBlk && (n != "find_index" || nargs == 0) {
		return 0
	}
	switch n {
	case "find", "detect", "find_index":
		return 1
	case "take_while":
		return 2
	}
	return 0
}

// rubyEnumProbe is the Array method that tests one element for such a method
// without a block. Twin of rbaEnumProbe in languages/lib/ruby-arr.js.
func rubyEnumProbe(n string) string {
	switch n {
	case "any?", "all?", "none?", "one?":
		return "any?"
	}
	return "include?"
}

// rubyGrepHit is Enumerable#grep's pattern === element. Twin of caseEq in
// ruby-interpreter.abnf and js_rxwhen in lib/ruby-rt.metajs.
func (rt *jsrt) rubyGrepHit(pat, v interface{}) bool {
	if isCallable(pat) {
		return rubyTruthy(rt.call(pat, jsUndef, []interface{}{v}))
	}
	if rubyUserObj(pat) {
		if m, ok := rubyFindMethod(pat, "==="); ok {
			return rubyTruthy(rt.call(m, jsUndef, []interface{}{pat, v}))
		}
	}
	return rt.rubyCaseEqv(pat, v)
}

// rubyEnumArgArr is a zip / chain argument as an Array: itself, or what #to_a
// answers for a value that responds to each; nil for anything else. Twin of
// enumArgArr in ruby-interpreter.abnf.
func (rt *jsrt) rubyEnumArgArr(v interface{}) *jsArray {
	if a, isArr := v.(*jsArray); isArr {
		return a
	}
	if isUndefOrNull(v) || rt.rubyRespond(v, []interface{}{jsSym{s: "each"}}) != true {
		return nil
	}
	a, _ := rt.rubyMethod(v, "to_a", nil).(*jsArray)
	return a
}

// rubyArrEnumMore is Array's grep / grep_v / minmax_by / slice_before /
// slice_after / chunk / chain; the last four answer an Enumerator over the
// groups. Twin of the arms of arrayMethod in ruby-interpreter.abnf and of
// rbArrayMethod in lib/ruby-rt.metajs (rbaSliceAt / rbaChunk / rbaMinMaxBy in
// lib/ruby-arr.js).
func (rt *jsrt) rubyArrEnumMore(t *jsArray, name string, pos []interface{}, blk interface{}) interface{} {
	switch name {
	case "grep", "grep_v":
		out := &jsArray{}
		for _, e := range t.elems {
			if rt.rubyGrepHit(argAt(pos, 0), e) != (name == "grep_v") {
				if blk != nil {
					e = rt.call(blk, jsUndef, []interface{}{e})
				}
				out.elems = append(out.elems, e)
			}
		}
		return out
	case "minmax_by":
		kp := rt.rubyArrKeyed(t, blk)
		if len(kp) == 0 {
			return &jsArray{elems: []interface{}{jsNull, jsNull}}
		}
		key := func(i int) interface{} { return kp[i].(*jsArray).elems[0] }
		lo, hi := 0, 0
		for i := 1; i < len(kp); i++ {
			if rt.rubyCmp(key(i), key(lo)) < 0 {
				lo = i
			}
			if rt.rubyCmp(key(i), key(hi)) > 0 {
				hi = i
			}
		}
		return &jsArray{elems: []interface{}{kp[lo].(*jsArray).elems[1], kp[hi].(*jsArray).elems[1]}}
	case "slice_before", "slice_after":
		out := &jsArray{}
		cur := &jsArray{}
		for _, e := range t.elems {
			var hit bool
			if blk != nil {
				hit = rubyTruthy(rt.call(blk, jsUndef, []interface{}{e}))
			} else {
				hit = rt.rubyGrepHit(argAt(pos, 0), e)
			}
			if name == "slice_before" && hit && len(cur.elems) > 0 {
				out.elems = append(out.elems, cur)
				cur = &jsArray{}
			}
			cur.elems = append(cur.elems, e)
			if name == "slice_after" && hit {
				out.elems = append(out.elems, cur)
				cur = &jsArray{}
			}
		}
		if len(cur.elems) > 0 {
			out.elems = append(out.elems, cur)
		}
		return rubyMkEnum(out)
	case "chunk":
		// A nil or :_separator key drops the element, :_alone makes it a chunk of
		// its own, and any other Symbol starting with `_' is reserved.
		out := &jsArray{}
		var cur *jsArray
		for _, e := range t.elems {
			k := rt.call(blk, jsUndef, []interface{}{e})
			drop, alone := isUndefOrNull(k), false
			if sy, isSym := k.(jsSym); isSym && strings.HasPrefix(sy.s, "_") {
				switch sy.s {
				case "_separator":
					drop = true
				case "_alone":
					alone = true
				default:
					rt.rubyRaiseErr(name, "RuntimeError", "symbols beginning with an underscore are reserved")
				}
			}
			switch {
			case drop:
				cur = nil
			case alone:
				out.elems = append(out.elems, &jsArray{elems: []interface{}{k, &jsArray{elems: []interface{}{e}}}})
				cur = nil
			case cur != nil && rt.rubyEq(cur.elems[0], k):
				g := cur.elems[1].(*jsArray)
				g.elems = append(g.elems, e)
			default:
				cur = &jsArray{elems: []interface{}{k, &jsArray{elems: []interface{}{e}}}}
				out.elems = append(out.elems, cur)
			}
		}
		return rubyMkEnum(out)
	}
	// chain
	out := &jsArray{elems: append([]interface{}{}, t.elems...)}
	for _, a := range pos {
		arr := rt.rubyEnumArgArr(a)
		if arr == nil {
			rt.rubyNoMethod(a, "each")
		}
		out.elems = append(out.elems, arr.elems...)
	}
	return rubyMkEnum(out)
}
