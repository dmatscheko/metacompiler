package abnf

// The Go twin of languages/lib/regex.js: the shared regular-expression engine, for
// the COMPILER grammars.
//
// An interpreter grammar includes lib/regex.js into its tag script and matches from
// there. A compiler grammar cannot: its output is an LLVM IR module whose only way
// out is a js_* extern, and a tag script is long gone by the time that module runs.
// So the same engine exists once more here, as a line-by-line port. The two files
// are meant to stay in step - same instruction opcodes, same parser, same
// backtracking matcher, same answers - and the matrix compares an interpreter and a
// compiler on the same input, which is what keeps them honest.
//
// A backtracking matcher rather than Go's own regexp package, for two reasons: RE2
// has no backreferences (Ruby, JavaScript, Python and Kotlin all do), and its
// leftmost-longest / syntax details would diverge from lib/regex.js in ways the
// byte-identical output check would eventually find. Porting the JS is the cheaper
// guarantee.
//
// This file is append-only with respect to the rest of the runtime: nothing here
// runs unless a program calls one of the js_rx* externs, which only the Ruby
// compiler grammar emits. The one hook in jsrt.go is addRegexExterns below, which
// ADDS names to the extern map and rebinds none.
//
// Positions are RUNE indices, matching the JS side's UTF-16 indices for everything
// in the Basic Multilingual Plane (and therefore for every ASCII pattern).

import "strings"

// ----- Instruction opcodes (see lib/regex.js) -----
const (
	rxOpClass   = 1  // match one character against clss[pc]
	rxOpSplit   = 2  // try a, then b
	rxOpJmp     = 3  // continue at a
	rxOpSave    = 4  // caps[a] = sp
	rxOpMark    = 5  // marks[a] = sp (loop-progress register)
	rxOpProg    = 6  // if marks[a] == sp, leave the loop at b (empty-body guard)
	rxOpMatch   = 7  //
	rxOpAssert  = 8  // zero-width assertion a
	rxOpLook    = 9  // lookahead: a = address after the body, b = 1 when negative
	rxOpLookEnd = 10 //
	rxOpBref    = 11 // backreference to group a
)

// ----- Assertion kinds -----
const (
	rxBOL    = 1 // ^
	rxEOL    = 2 // $
	rxWordB  = 3 // \b
	rxNWordB = 4 // \B
	rxBOS    = 5 // \A
	rxEOS    = 6 // \z
	rxEOSNL  = 7 // \Z
)

// ----- Character-class shorthands -----
const (
	rxSD  = 1 // \d
	rxSND = 2 // \D
	rxSW  = 3 // \w
	rxSNW = 4 // \W
	rxSS  = 5 // \s
	rxSNS = 6 // \S
	rxSH  = 7 // \h
	rxSNH = 8 // \H
)

// rxSteps caps one rxSearch attempt, so a pathological pattern reports an error
// instead of hanging the run. Same number as RX_STEPS in lib/regex.js.
const rxSteps = 400000

// ===== Character classes =====

type rxCls struct {
	neg bool
	lo  []rune
	hi  []rune
	sp  []int
}

func rxNewCls() *rxCls { return &rxCls{} }

func (c *rxCls) add(from, to rune) {
	c.lo = append(c.lo, from)
	c.hi = append(c.hi, to)
}

func (c *rxCls) addSp(code int) { c.sp = append(c.sp, code) }

func rxIsWord(c rune) bool {
	if c >= '0' && c <= '9' {
		return true
	}
	if c >= 'A' && c <= 'Z' {
		return true
	}
	if c >= 'a' && c <= 'z' {
		return true
	}
	return c == '_'
}

func rxIsSpace(c rune) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' || c == 11
}

func rxIsDigit(c rune) bool { return c >= '0' && c <= '9' }

func rxIsHex(c rune) bool {
	return (c >= '0' && c <= '9') || (c >= 'A' && c <= 'F') || (c >= 'a' && c <= 'f')
}

func rxSpHas(code int, c rune) bool {
	switch code {
	case rxSD:
		return rxIsDigit(c)
	case rxSND:
		return !rxIsDigit(c)
	case rxSW:
		return rxIsWord(c)
	case rxSNW:
		return !rxIsWord(c)
	case rxSS:
		return rxIsSpace(c)
	case rxSNS:
		return !rxIsSpace(c)
	case rxSH:
		return rxIsHex(c)
	case rxSNH:
		return !rxIsHex(c)
	}
	return false
}

// rxSwapCase answers the other-case code point of an ASCII letter, or -1.
func rxSwapCase(c rune) rune {
	if c >= 'A' && c <= 'Z' {
		return c + 32
	}
	if c >= 'a' && c <= 'z' {
		return c - 32
	}
	return -1
}

func (c *rxCls) hit(ch rune) bool {
	for i := range c.lo {
		if ch >= c.lo[i] && ch <= c.hi[i] {
			return true
		}
	}
	for _, s := range c.sp {
		if rxSpHas(s, ch) {
			return true
		}
	}
	return false
}

// match applies case folding first and negation LAST - a negated class under /i/
// must reject both cases of a listed letter, which is only true in that order.
func (c *rxCls) match(ch rune, icase bool) bool {
	hit := c.hit(ch)
	if !hit && icase {
		if alt := rxSwapCase(ch); alt >= 0 {
			hit = c.hit(alt)
		}
	}
	if c.neg {
		return !hit
	}
	return hit
}

// ===== The pattern parser =====

type rxNode struct {
	t      string
	items  []*rxNode
	cls    *rxCls
	min    int
	max    int
	greedy bool
	gidx   int
}

func rxNewNode(t string) *rxNode {
	return &rxNode{t: t, cls: rxNewCls(), greedy: true, gidx: -1}
}

type rxSt struct {
	p          []rune
	i          int
	n          int
	err        string
	ngroups    int
	names      []string
	nameGroups []int
	ext        bool
	dotall     bool
}

func (st *rxSt) fail(msg string) {
	if st.err == "" {
		st.err = msg
	}
}

// skipX steps over the whitespace and # comments the x flag makes insignificant.
func (st *rxSt) skipX() {
	if !st.ext {
		return
	}
	for st.i < st.n {
		c := st.p[st.i]
		if rxIsSpace(c) {
			st.i++
		} else if c == '#' {
			for st.i < st.n && st.p[st.i] != '\n' {
				st.i++
			}
		} else {
			return
		}
	}
}

func rxHexVal(c rune) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	}
	return -1
}

type rxEscRes struct {
	kind int // 0 literal character, 1 shorthand class, 2 backreference
	ch   rune
	sp   int
}

// rxEsc reads the escape whose backslash has already been consumed.
func (st *rxSt) esc(inClass bool) rxEscRes {
	var res rxEscRes
	if st.i >= st.n {
		st.fail("trailing backslash in pattern")
		return res
	}
	c := st.p[st.i]
	st.i++
	switch c {
	case 'd':
		return rxEscRes{kind: 1, sp: rxSD}
	case 'D':
		return rxEscRes{kind: 1, sp: rxSND}
	case 'w':
		return rxEscRes{kind: 1, sp: rxSW}
	case 'W':
		return rxEscRes{kind: 1, sp: rxSNW}
	case 's':
		return rxEscRes{kind: 1, sp: rxSS}
	case 'S':
		return rxEscRes{kind: 1, sp: rxSNS}
	case 'h':
		return rxEscRes{kind: 1, sp: rxSH}
	case 'H':
		return rxEscRes{kind: 1, sp: rxSNH}
	case 'n':
		return rxEscRes{ch: '\n'}
	case 't':
		return rxEscRes{ch: '\t'}
	case 'r':
		return rxEscRes{ch: '\r'}
	case 'f':
		return rxEscRes{ch: '\f'}
	case 'v':
		return rxEscRes{ch: 11}
	case 'a':
		return rxEscRes{ch: 7}
	case 'e':
		return rxEscRes{ch: 27}
	case '0':
		return rxEscRes{ch: 0}
	case 'b':
		if inClass {
			return rxEscRes{ch: 8}
		}
	case 'x':
		v, got := 0, 0
		for got < 2 && st.i < st.n {
			h := rxHexVal(st.p[st.i])
			if h < 0 {
				break
			}
			v = v*16 + h
			st.i++
			got++
		}
		if got == 0 {
			st.fail("invalid hex escape")
		}
		return rxEscRes{ch: rune(v)}
	case 'u':
		v, got := 0, 0
		for got < 4 && st.i < st.n {
			h := rxHexVal(st.p[st.i])
			if h < 0 {
				break
			}
			v = v*16 + h
			st.i++
			got++
		}
		if got == 0 {
			st.fail("invalid unicode escape")
		}
		return rxEscRes{ch: rune(v)}
	}
	if !inClass && c >= '1' && c <= '9' {
		return rxEscRes{kind: 2, ch: c - '0'}
	}
	return rxEscRes{ch: c}
}

func (st *rxSt) parseClass() *rxNode {
	node := rxNewNode("cls")
	cls := node.cls
	if st.i < st.n && st.p[st.i] == '^' {
		cls.neg = true
		st.i++
	}
	first := true
	for st.i < st.n {
		c := st.p[st.i]
		if c == ']' && !first {
			st.i++
			return node
		}
		first = false
		// A POSIX bracket [:alpha:] is not supported; report it rather than
		// silently reading it as a set of punctuation characters.
		if c == '[' && st.i+1 < st.n && st.p[st.i+1] == ':' {
			st.fail("POSIX bracket expressions are not supported")
			return node
		}
		var loCode rune
		if c == '\\' {
			st.i++
			e := st.esc(true)
			if e.kind == 1 {
				cls.addSp(e.sp)
				continue
			}
			loCode = e.ch
		} else {
			st.i++
			loCode = c
		}
		if st.i+1 < st.n && st.p[st.i] == '-' && st.p[st.i+1] != ']' {
			st.i++
			var hiCode rune
			c2 := st.p[st.i]
			if c2 == '\\' {
				st.i++
				e2 := st.esc(true)
				if e2.kind == 1 {
					// "a-\d" is not a range; take the '-' literally and the class on.
					cls.add(loCode, loCode)
					cls.add('-', '-')
					cls.addSp(e2.sp)
					continue
				}
				hiCode = e2.ch
			} else {
				st.i++
				hiCode = c2
			}
			if hiCode < loCode {
				st.fail("character class range is out of order")
				return node
			}
			cls.add(loCode, hiCode)
		} else {
			cls.add(loCode, loCode)
		}
	}
	st.fail("unterminated character class")
	return node
}

func (st *rxSt) readGroupName(closer rune) string {
	start := st.i
	for st.i < st.n && st.p[st.i] != closer {
		st.i++
	}
	if st.i >= st.n {
		st.fail("unterminated group name")
		return ""
	}
	name := string(st.p[start:st.i])
	st.i++
	return name
}

func (st *rxSt) expect(c rune, msg string) {
	st.skipX()
	if st.i < st.n && st.p[st.i] == c {
		st.i++
		return
	}
	st.fail(msg)
}

func (st *rxSt) parseGroup() *rxNode {
	if st.i < st.n && st.p[st.i] == '?' {
		st.i++
		if st.i >= st.n {
			st.fail("unterminated group")
			return rxNewNode("empty")
		}
		k := st.p[st.i]
		switch {
		case k == ':':
			st.i++
			g := rxNewNode("group")
			g.items = append(g.items, st.parseAlt())
			st.expect(')', "unterminated group")
			return g
		case k == '#':
			for st.i < st.n && st.p[st.i] != ')' {
				st.i++
			}
			st.expect(')', "unterminated comment group")
			return rxNewNode("empty")
		case k == '=' || k == '!':
			st.i++
			lk := rxNewNode("look")
			if k == '!' {
				lk.gidx = 1
			}
			lk.items = append(lk.items, st.parseAlt())
			st.expect(')', "unterminated lookahead")
			return lk
		case k == '<' || k == '\'':
			if k == '<' && st.i+1 < st.n {
				k2 := st.p[st.i+1]
				if k2 == '=' || k2 == '!' {
					st.fail("lookbehind is not supported")
					return rxNewNode("empty")
				}
			}
			st.i++
			closer := '>'
			if k == '\'' {
				closer = '\''
			}
			nm := st.readGroupName(closer)
			ng := rxNewNode("group")
			st.ngroups++
			ng.gidx = st.ngroups
			st.names = append(st.names, nm)
			st.nameGroups = append(st.nameGroups, ng.gidx)
			ng.items = append(ng.items, st.parseAlt())
			st.expect(')', "unterminated group")
			return ng
		}
		st.fail("unsupported group form (?" + string(st.p[st.i]) + ")")
		return rxNewNode("empty")
	}
	cg := rxNewNode("group")
	st.ngroups++
	cg.gidx = st.ngroups
	cg.items = append(cg.items, st.parseAlt())
	st.expect(')', "unterminated group")
	return cg
}

func (st *rxSt) parseAtom() *rxNode {
	c := st.p[st.i]
	switch c {
	case '(':
		st.i++
		return st.parseGroup()
	case '[':
		st.i++
		return st.parseClass()
	case '.':
		st.i++
		dot := rxNewNode("cls")
		dot.cls.neg = true
		if !st.dotall {
			dot.cls.add('\n', '\n')
		}
		return dot
	case '^':
		st.i++
		b := rxNewNode("assert")
		b.gidx = rxBOL
		return b
	case '$':
		st.i++
		d := rxNewNode("assert")
		d.gidx = rxEOL
		return d
	case '\\':
		var nxt rune
		if st.i+1 < st.n {
			nxt = st.p[st.i+1]
		}
		if nxt == 'b' || nxt == 'B' || nxt == 'A' || nxt == 'z' || nxt == 'Z' {
			st.i += 2
			a := rxNewNode("assert")
			switch nxt {
			case 'b':
				a.gidx = rxWordB
			case 'B':
				a.gidx = rxNWordB
			case 'A':
				a.gidx = rxBOS
			case 'z':
				a.gidx = rxEOS
			case 'Z':
				a.gidx = rxEOSNL
			}
			return a
		}
		st.i++
		e := st.esc(false)
		if e.kind == 1 {
			sn := rxNewNode("cls")
			sn.cls.addSp(e.sp)
			return sn
		}
		if e.kind == 2 {
			br := rxNewNode("bref")
			br.gidx = int(e.ch)
			return br
		}
		ln := rxNewNode("cls")
		ln.cls.add(e.ch, e.ch)
		return ln
	case ')', '|':
		return rxNewNode("empty")
	case '*', '+', '?':
		st.fail("quantifier with nothing to repeat")
		st.i++
		return rxNewNode("empty")
	}
	st.i++
	lit := rxNewNode("cls")
	lit.cls.add(c, c)
	return lit
}

// parseBound reads {n} / {n,} / {n,m}. It answers ok=false when what follows the
// brace is not a bound, so a literal { stays a literal {.
func (st *rxSt) parseBound() (int, int, bool) {
	j := st.i + 1
	minV, digits := 0, 0
	for j < st.n && rxIsDigit(st.p[j]) {
		minV = minV*10 + int(st.p[j]-'0')
		j++
		digits++
	}
	if digits == 0 {
		return 0, 0, false
	}
	maxV := minV
	if j < st.n && st.p[j] == ',' {
		j++
		mv, d2 := 0, 0
		for j < st.n && rxIsDigit(st.p[j]) {
			mv = mv*10 + int(st.p[j]-'0')
			j++
			d2++
		}
		if d2 == 0 {
			maxV = -1
		} else {
			maxV = mv
		}
	}
	if j >= st.n || st.p[j] != '}' {
		return 0, 0, false
	}
	st.i = j + 1
	return minV, maxV, true
}

func (st *rxSt) parsePiece() *rxNode {
	atom := st.parseAtom()
	st.skipX()
	for st.i < st.n {
		c := st.p[st.i]
		var minV, maxV int
		switch {
		case c == '*':
			st.i++
			minV, maxV = 0, -1
		case c == '+':
			st.i++
			minV, maxV = 1, -1
		case c == '?':
			st.i++
			minV, maxV = 0, 1
		case c == '{':
			lo, hi, ok := st.parseBound()
			if !ok {
				return atom
			}
			minV, maxV = lo, hi
			if maxV >= 0 && maxV < minV {
				st.fail("quantifier bounds are out of order")
				return atom
			}
		default:
			return atom
		}
		rep := rxNewNode("rep")
		rep.min = minV
		rep.max = maxV
		rep.greedy = true
		if st.i < st.n {
			q := st.p[st.i]
			if q == '?' {
				rep.greedy = false
				st.i++
			} else if q == '+' {
				st.i++ // possessive: treated as greedy
			}
		}
		rep.items = append(rep.items, atom)
		atom = rep
		st.skipX()
	}
	return atom
}

func (st *rxSt) parseCat() *rxNode {
	cat := rxNewNode("cat")
	st.skipX()
	for st.i < st.n {
		c := st.p[st.i]
		if c == '|' || c == ')' {
			break
		}
		before := st.i
		cat.items = append(cat.items, st.parsePiece())
		if st.err != "" {
			break
		}
		if st.i == before {
			break // no progress: a malformed pattern
		}
		st.skipX()
	}
	return cat
}

func (st *rxSt) parseAlt() *rxNode {
	alt := rxNewNode("alt")
	alt.items = append(alt.items, st.parseCat())
	for st.i < st.n && st.p[st.i] == '|' {
		st.i++
		alt.items = append(alt.items, st.parseCat())
		if st.err != "" {
			break
		}
	}
	if len(alt.items) == 1 {
		return alt.items[0]
	}
	return alt
}

// ===== The emitter =====

type rxProg struct {
	ops    []int
	as     []int
	bs     []int
	clss   []*rxCls
	nmarks int
	dummy  *rxCls
}

func (pg *rxProg) add(op, a, b int, cls *rxCls) int {
	at := len(pg.ops)
	pg.ops = append(pg.ops, op)
	pg.as = append(pg.as, a)
	pg.bs = append(pg.bs, b)
	pg.clss = append(pg.clss, cls)
	return at
}

func (pg *rxProg) here() int { return len(pg.ops) }

func (pg *rxProg) emit(node *rxNode) {
	switch node.t {
	case "empty":
		return
	case "cls":
		pg.add(rxOpClass, 0, 0, node.cls)
	case "assert":
		pg.add(rxOpAssert, node.gidx, 0, pg.dummy)
	case "bref":
		pg.add(rxOpBref, node.gidx, 0, pg.dummy)
	case "cat":
		for _, it := range node.items {
			pg.emit(it)
		}
	case "alt":
		pg.emitAlt(node)
	case "group":
		gi := node.gidx
		if gi >= 0 {
			pg.add(rxOpSave, 2*gi, 0, pg.dummy)
		}
		if len(node.items) > 0 {
			pg.emit(node.items[0])
		}
		if gi >= 0 {
			pg.add(rxOpSave, 2*gi+1, 0, pg.dummy)
		}
	case "look":
		neg := 0
		if node.gidx == 1 {
			neg = 1
		}
		lk := pg.add(rxOpLook, 0, neg, pg.dummy)
		if len(node.items) > 0 {
			pg.emit(node.items[0])
		}
		pg.add(rxOpLookEnd, 0, 0, pg.dummy)
		pg.as[lk] = pg.here()
	case "rep":
		pg.emitRep(node)
	}
}

func (pg *rxProg) emitAlt(node *rxNode) {
	var jumps []int
	for i, it := range node.items {
		if i < len(node.items)-1 {
			s := pg.add(rxOpSplit, 0, 0, pg.dummy)
			pg.as[s] = pg.here()
			pg.emit(it)
			jumps = append(jumps, pg.add(rxOpJmp, 0, 0, pg.dummy))
			pg.bs[s] = pg.here()
		} else {
			pg.emit(it)
		}
	}
	end := pg.here()
	for _, j := range jumps {
		pg.as[j] = end
	}
}

func (pg *rxProg) emitRep(node *rxNode) {
	body := node.items[0]
	for i := 0; i < node.min; i++ {
		pg.emit(body)
	}
	if node.max < 0 {
		// Unbounded tail. The MARK/PROG pair is the empty-body guard: a body that
		// consumed nothing leaves the loop instead of spinning ((a*)* is a real
		// pattern, and without this it never terminates).
		mk := pg.nmarks
		pg.nmarks++
		l1 := pg.here()
		sp := pg.add(rxOpSplit, 0, 0, pg.dummy)
		pg.add(rxOpMark, mk, 0, pg.dummy)
		pg.emit(body)
		pgi := pg.add(rxOpProg, mk, 0, pg.dummy)
		pg.add(rxOpJmp, l1, 0, pg.dummy)
		out := pg.here()
		if node.greedy {
			pg.as[sp] = l1 + 1
			pg.bs[sp] = out
		} else {
			pg.as[sp] = out
			pg.bs[sp] = l1 + 1
		}
		pg.bs[pgi] = out
		return
	}
	var splits []int
	for k := 0; k < node.max-node.min; k++ {
		splits = append(splits, pg.add(rxOpSplit, 0, 0, pg.dummy))
		pg.emit(body)
	}
	endAt := pg.here()
	for _, at := range splits {
		if node.greedy {
			pg.as[at] = at + 1
			pg.bs[at] = endAt
		} else {
			pg.as[at] = endAt
			pg.bs[at] = at + 1
		}
	}
}

// ===== Compilation =====

type rxRe struct {
	ok         bool
	err        string
	ops        []int
	as         []int
	bs         []int
	clss       []*rxCls
	ngroups    int
	nmarks     int
	names      []string
	nameGroups []int
	icase      bool
	dotall     bool
	multi      bool
	src        string
	flags      string
}

func rxCompile(pattern, flags string) *rxRe {
	icase := strings.ContainsRune(flags, 'i')
	dotall := strings.ContainsRune(flags, 's')
	multi := strings.ContainsRune(flags, 'm')
	ext := strings.ContainsRune(flags, 'x')
	st := &rxSt{p: []rune(pattern), ext: ext, dotall: dotall}
	st.n = len(st.p)
	ast := st.parseAlt()
	if st.err == "" && st.i < st.n {
		if st.p[st.i] == ')' {
			st.fail("unmatched ) in pattern")
		} else {
			st.fail("unexpected character in pattern")
		}
	}
	if st.err != "" {
		return &rxRe{ok: false, err: st.err, icase: icase, dotall: dotall, multi: multi,
			src: pattern, flags: flags}
	}
	pg := &rxProg{dummy: rxNewCls()}
	pg.add(rxOpSave, 0, 0, pg.dummy)
	pg.emit(ast)
	pg.add(rxOpSave, 1, 0, pg.dummy)
	pg.add(rxOpMatch, 0, 0, pg.dummy)
	return &rxRe{
		ok: true, ops: pg.ops, as: pg.as, bs: pg.bs, clss: pg.clss,
		ngroups: st.ngroups, nmarks: pg.nmarks, names: st.names, nameGroups: st.nameGroups,
		icase: icase, dotall: dotall, multi: multi, src: pattern, flags: flags,
	}
}

// ===== The matcher =====

type rxRunSt struct {
	steps int
	over  bool
}

func (re *rxRe) assertOK(text []rune, kind, sp int) bool {
	n := len(text)
	switch kind {
	case rxBOS:
		return sp == 0
	case rxEOS:
		return sp == n
	case rxEOSNL:
		if sp == n {
			return true
		}
		return sp == n-1 && text[sp] == '\n'
	case rxBOL:
		if sp == 0 {
			return true
		}
		if !re.multi {
			return false
		}
		return text[sp-1] == '\n'
	case rxEOL:
		if sp == n {
			return true
		}
		if !re.multi {
			return false
		}
		return text[sp] == '\n'
	}
	before, after := false, false
	if sp > 0 {
		before = rxIsWord(text[sp-1])
	}
	if sp < n {
		after = rxIsWord(text[sp])
	}
	if kind == rxWordB {
		return before != after
	}
	return before == after
}

// run is the backtracking VM: the end position of a match from pc/sp, or -1.
// Recursion happens only where the machine has a choice (SPLIT) or has to be able
// to undo a write (SAVE / MARK / LOOK).
func (re *rxRe) run(text []rune, pcIn, spIn int, caps, marks []int, st *rxRunSt) int {
	n := len(text)
	pc, sp := pcIn, spIn
	for {
		st.steps++
		if st.steps > rxSteps {
			st.over = true
			return -1
		}
		switch re.ops[pc] {
		case rxOpClass:
			if sp >= n || !re.clss[pc].match(text[sp], re.icase) {
				return -1
			}
			pc++
			sp++
		case rxOpJmp:
			pc = re.as[pc]
		case rxOpSplit:
			if r := re.run(text, re.as[pc], sp, caps, marks, st); r >= 0 {
				return r
			}
			if st.over {
				return -1
			}
			pc = re.bs[pc]
		case rxOpSave:
			slot := re.as[pc]
			old := caps[slot]
			caps[slot] = sp
			if r := re.run(text, pc+1, sp, caps, marks, st); r >= 0 {
				return r
			}
			caps[slot] = old
			return -1
		case rxOpMark:
			slot := re.as[pc]
			old := marks[slot]
			marks[slot] = sp
			if r := re.run(text, pc+1, sp, caps, marks, st); r >= 0 {
				return r
			}
			marks[slot] = old
			return -1
		case rxOpProg:
			if marks[re.as[pc]] == sp {
				pc = re.bs[pc]
			} else {
				pc++
			}
		case rxOpAssert:
			if !re.assertOK(text, re.as[pc], sp) {
				return -1
			}
			pc++
		case rxOpLook:
			r := re.run(text, pc+1, sp, caps, marks, st)
			if st.over {
				return -1
			}
			if (r >= 0) != (re.bs[pc] == 0) {
				return -1
			}
			pc = re.as[pc]
		case rxOpLookEnd:
			return sp
		case rxOpBref:
			g := re.as[pc]
			// A reference to a group NUMBER the pattern never defined (\1 in a
			// pattern with no groups) is out of range in caps. Treated exactly like
			// a group that did not take part - it matches the empty string. Without
			// the length test this panics with an index-out-of-range, and the JS
			// twin reads undefined and lets sp go NaN.
			from, to := -1, -1
			if 2*g+1 < len(caps) {
				from, to = caps[2*g], caps[2*g+1]
			}
			if from < 0 || to < 0 {
				// A group that did not take part matches the empty string.
				pc++
				break
			}
			length := to - from
			if sp+length > n {
				return -1
			}
			for bi := 0; bi < length; bi++ {
				lc, rc := text[from+bi], text[sp+bi]
				if lc != rc {
					if !re.icase || rxSwapCase(rc) != lc {
						return -1
					}
				}
			}
			pc++
			sp += length
		case rxOpMatch:
			return sp
		default:
			return -1
		}
	}
}

type rxMatch struct {
	begin int
	end   int
	caps  []int
}

// search finds the leftmost match at or after start. ok=false means the step cap
// was hit, which the caller turns into a clean runtime error.
func (re *rxRe) search(text []rune, start int) (*rxMatch, bool) {
	from := start
	if from < 0 {
		from = 0
	}
	nslots := 2 * (re.ngroups + 1)
	for at := from; at <= len(text); at++ {
		caps := make([]int, nslots)
		for i := range caps {
			caps[i] = -1
		}
		marks := make([]int, re.nmarks)
		for i := range marks {
			marks[i] = -1
		}
		st := &rxRunSt{}
		if end := re.run(text, 0, at, caps, marks, st); end >= 0 {
			return &rxMatch{begin: caps[0], end: caps[1], caps: caps}, true
		}
		if st.over {
			return nil, false
		}
	}
	return nil, true
}

func (re *rxRe) group(text []rune, m *rxMatch, i int) (string, bool) {
	if i < 0 || i > re.ngroups {
		return "", false
	}
	from, to := m.caps[2*i], m.caps[2*i+1]
	if from < 0 || to < 0 {
		return "", false
	}
	return string(text[from:to]), true
}

func (re *rxRe) indexOfName(name string) int {
	for i, n := range re.names {
		if n == name {
			return re.nameGroups[i]
		}
	}
	return -1
}

// expand substitutes \0 / \1 .. \9 and \k<name> in a replacement template.
func (re *rxRe) expand(text []rune, m *rxMatch, repl string) string {
	r := []rune(repl)
	var out strings.Builder
	for i := 0; i < len(r); {
		if r[i] != '\\' || i+1 >= len(r) {
			out.WriteRune(r[i])
			i++
			continue
		}
		d := r[i+1]
		if d >= '0' && d <= '9' {
			if g, ok := re.group(text, m, int(d-'0')); ok {
				out.WriteString(g)
			}
			i += 2
			continue
		}
		if d == 'k' && i+2 < len(r) && r[i+2] == '<' {
			j := i + 3
			for j < len(r) && r[j] != '>' {
				j++
			}
			if gi := re.indexOfName(string(r[i+3 : j])); gi >= 0 {
				if g, ok := re.group(text, m, gi); ok {
					out.WriteString(g)
				}
			}
			i = j + 1
			continue
		}
		out.WriteRune(r[i+1])
		i += 2
	}
	return out.String()
}

// ===== The compiled-pattern cache =====

// rxCache remembers one compiled program per (flags, source). A literal inside a
// loop is compiled once even though the emitted IR hands the pattern over as a
// plain string on every call.
var rxCache = map[string]*rxRe{}

func rxGet(pattern, flags string) *rxRe {
	key := flags + "\x00" + pattern
	if re, ok := rxCache[key]; ok {
		return re
	}
	re := rxCompile(pattern, flags)
	rxCache[key] = re
	return re
}

// ===== The Ruby value model: Regexp and MatchData =====
//
// Both are ordinary *jsObject values carrying __rx-prefixed slots, so nothing in
// the rest of the runtime has to learn a new type. Their METHODS are reached
// through js_rxmcall below, which is a wrapper around the existing js_rmcall: the
// Ruby compiler grammar emits the wrapper everywhere it used to emit js_rmcall,
// and every receiver that is not a Regexp or a MatchData is handed straight on to
// the original. Same shape for js_rxcase (case/when) and js_rxget (m[1]).

func rxIsRegexObj(v interface{}) bool {
	o, ok := v.(*jsObject)
	if !ok {
		return false
	}
	_, has := o.props["__rxsrc"]
	return has && o.props["__rxmd"] == nil
}

func rxIsMatchObj(v interface{}) bool {
	o, ok := v.(*jsObject)
	if !ok {
		return false
	}
	return o.props["__rxmd"] != nil
}

func rxObjRe(v interface{}) *rxRe {
	o := v.(*jsObject)
	src, _ := o.props["__rxsrc"].(string)
	flags, _ := o.props["__rxflags"].(string)
	return rxGet(src, flags)
}

func rxNewRegexObj(src, flags string) *jsObject {
	o := newJSObject()
	o.set("__rxsrc", src)
	o.set("__rxflags", flags)
	return o
}

func rxNewMatchObj(src, flags, text string, m *rxMatch) *jsObject {
	o := newJSObject()
	o.set("__rxmd", true)
	o.set("__rxsrc", src)
	o.set("__rxflags", flags)
	o.set("__rxtext", text)
	caps := &jsArray{}
	for _, c := range m.caps {
		caps.elems = append(caps.elems, float64(c))
	}
	o.set("__rxcaps", caps)
	return o
}

func rxMatchOf(v interface{}) (*rxRe, []rune, *rxMatch) {
	o := v.(*jsObject)
	re := rxObjRe(v)
	text, _ := o.props["__rxtext"].(string)
	arr, _ := o.props["__rxcaps"].(*jsArray)
	caps := make([]int, len(arr.elems))
	for i, e := range arr.elems {
		f, _ := e.(float64)
		caps[i] = int(f)
	}
	m := &rxMatch{caps: caps}
	if len(caps) >= 2 {
		m.begin, m.end = caps[0], caps[1]
	}
	return re, []rune(text), m
}

// rxSetGlobals publishes $~, $& and $1..$9 after a match attempt, exactly as
// ruby-interpreter.abnf's setLastMatch does. A FAILED match clears them.
func (rt *jsrt) rxSetGlobals(md *jsObject) {
	if rt.rubyGlobals == nil {
		rt.rubyGlobals = map[string]interface{}{}
	}
	if md == nil {
		rt.rubyGlobals["~"] = jsNull
		rt.rubyGlobals["&"] = jsNull
		for i := 1; i <= 9; i++ {
			rt.rubyGlobals[string(rune('0'+i))] = jsNull
		}
		return
	}
	re, text, m := rxMatchOf(md)
	rt.rubyGlobals["~"] = md
	if g, ok := re.group(text, m, 0); ok {
		rt.rubyGlobals["&"] = g
	} else {
		rt.rubyGlobals["&"] = jsNull
	}
	for i := 1; i <= 9; i++ {
		if g, ok := re.group(text, m, i); ok {
			rt.rubyGlobals[string(rune('0'+i))] = g
		} else {
			rt.rubyGlobals[string(rune('0'+i))] = jsNull
		}
	}
}

// rxRunMatch searches and publishes the globals; it answers nil for no match.
func (rt *jsrt) rxRunMatch(reObj interface{}, s string, from int) *jsObject {
	re := rxObjRe(reObj)
	if !re.ok {
		rt.fail("invalid regexp /%s/: %s", re.src, re.err)
	}
	m, ok := re.search([]rune(s), from)
	if !ok {
		rt.fail("regexp: step limit exceeded")
	}
	if m == nil {
		rt.rxSetGlobals(nil)
		return nil
	}
	o := rxNewMatchObj(re.src, re.flags, s, m)
	rt.rxSetGlobals(o)
	return o
}

// rxMatchOperand sorts the two sides of =~ / === into (pattern, subject).
func rxMatchOperand(l, r interface{}) (interface{}, interface{}, bool) {
	if rxIsRegexObj(l) {
		return l, r, true
	}
	if rxIsRegexObj(r) {
		return r, l, true
	}
	return nil, nil, false
}

func rxKeyName(v interface{}) (string, bool) {
	switch t := v.(type) {
	case jsSym:
		return t.s, true
	case string:
		return t, true
	}
	return "", false
}

// rxMdIndex is MatchData#[]: an Integer reads a group by number, a Symbol or a
// String by name.
func (rt *jsrt) rxMdIndex(md interface{}, key interface{}) interface{} {
	re, text, m := rxMatchOf(md)
	if f, isNum := key.(float64); isNum {
		if g, ok := re.group(text, m, int(f)); ok {
			return g
		}
		return jsNull
	}
	if name, ok := rxKeyName(key); ok {
		if gi := re.indexOfName(name); gi >= 0 {
			if g, ok2 := re.group(text, m, gi); ok2 {
				return g
			}
		}
	}
	return jsNull
}

func rxCaptureArray(re *rxRe, text []rune, m *rxMatch, withWhole bool) *jsArray {
	out := &jsArray{}
	first := 1
	if withWhole {
		first = 0
	}
	for i := first; i <= re.ngroups; i++ {
		if g, ok := re.group(text, m, i); ok {
			out.elems = append(out.elems, g)
		} else {
			out.elems = append(out.elems, jsNull)
		}
	}
	return out
}

// rxScan is String#scan: with no group it collects the whole matches, with groups
// the capture list of each match (Ruby).
func (rt *jsrt) rxScan(reObj interface{}, s string) *jsArray {
	re := rxObjRe(reObj)
	text := []rune(s)
	out := &jsArray{}
	for at := 0; at <= len(text); {
		m, ok := re.search(text, at)
		if !ok {
			rt.fail("regexp: step limit exceeded")
		}
		if m == nil {
			break
		}
		if re.ngroups == 0 {
			g, _ := re.group(text, m, 0)
			out.elems = append(out.elems, g)
		} else {
			out.elems = append(out.elems, rxCaptureArray(re, text, m, false))
		}
		if m.end == m.begin {
			at = m.end + 1
		} else {
			at = m.end
		}
	}
	return out
}

// rxReplace is String#sub / #gsub with a STRING replacement (the block form is
// handled on the emitter side, which is where a closure can be called).
func (rt *jsrt) rxReplace(reObj interface{}, s, repl string, all bool) string {
	re := rxObjRe(reObj)
	text := []rune(s)
	var out strings.Builder
	at, last := 0, 0
	for at <= len(text) {
		m, ok := re.search(text, at)
		if !ok {
			rt.fail("regexp: step limit exceeded")
		}
		if m == nil {
			break
		}
		out.WriteString(string(text[last:m.begin]))
		out.WriteString(re.expand(text, m, repl))
		if m.end == m.begin {
			// An empty match must still make progress or this loop never ends.
			if m.end < len(text) {
				out.WriteRune(text[m.end])
			}
			last = m.end + 1
			at = m.end + 1
		} else {
			last = m.end
			at = m.end
		}
		if !all {
			break
		}
	}
	if last <= len(text) {
		out.WriteString(string(text[last:]))
	}
	return out.String()
}

func (rt *jsrt) rxSplit(reObj interface{}, s string) *jsArray {
	re := rxObjRe(reObj)
	text := []rune(s)
	parts := &jsArray{}
	at, last := 0, 0
	for at <= len(text) {
		m, ok := re.search(text, at)
		if !ok {
			rt.fail("regexp: step limit exceeded")
		}
		if m == nil {
			break
		}
		if m.end == m.begin {
			if m.begin >= len(text) {
				break
			}
			parts.elems = append(parts.elems, string(text[last:m.begin+1]))
			last, at = m.begin+1, m.begin+1
		} else {
			parts.elems = append(parts.elems, string(text[last:m.begin]))
			last, at = m.end, m.end
		}
	}
	parts.elems = append(parts.elems, string(text[last:]))
	return parts
}

// rxRegexMethod answers a method call whose RECEIVER is a Regexp or a MatchData,
// or ok=false when the name is none of theirs.
func (rt *jsrt) rxRegexMethod(target interface{}, name string, args []interface{}) (interface{}, bool) {
	if rxIsRegexObj(target) {
		re := rxObjRe(target)
		switch name {
		case "match?":
			m, ok := re.search([]rune(rt.rubyStr(argAt(args, 0))), 0)
			if !ok {
				rt.fail("regexp: step limit exceeded")
			}
			return m != nil, true
		case "match":
			from := 0
			if len(args) > 1 {
				if f, isNum := args[1].(float64); isNum {
					from = int(f)
				}
			}
			md := rt.rxRunMatch(target, rt.rubyStr(argAt(args, 0)), from)
			if md == nil {
				return jsNull, true
			}
			return md, true
		case "=~":
			return rt.rxMatchOp(target, argAt(args, 0)), true
		case "===":
			return rt.rxMatchOp(target, argAt(args, 0)) != jsNull, true
		case "source":
			return re.src, true
		case "options":
			return float64(0), true
		case "names":
			out := &jsArray{}
			for _, n := range re.names {
				out.elems = append(out.elems, n)
			}
			return out, true
		case "to_s":
			return "(?-mix:" + re.src + ")", true
		case "inspect":
			return "/" + re.src + "/" + re.flags, true
		case "class":
			return "Regexp", true
		}
		rt.fail("unknown Regexp method: %s", name)
	}
	if rxIsMatchObj(target) {
		re, text, m := rxMatchOf(target)
		switch name {
		case "[]":
			return rt.rxMdIndex(target, argAt(args, 0)), true
		case "to_a":
			return rxCaptureArray(re, text, m, true), true
		case "captures":
			return rxCaptureArray(re, text, m, false), true
		case "named_captures":
			keys, vals := &jsArray{}, &jsArray{}
			for _, n := range re.names {
				keys.elems = append(keys.elems, n)
				if g, ok := re.group(text, m, re.indexOfName(n)); ok {
					vals.elems = append(vals.elems, g)
				} else {
					vals.elems = append(vals.elems, jsNull)
				}
			}
			return &jsObject{props: map[string]interface{}{
				"__dict": true, "keys": keys, "vals": vals,
			}}, true
		case "names":
			out := &jsArray{}
			for _, n := range re.names {
				out.elems = append(out.elems, n)
			}
			return out, true
		case "pre_match":
			return string(text[:m.begin]), true
		case "post_match":
			return string(text[m.end:]), true
		case "begin", "end":
			g := 0
			if len(args) > 0 {
				if f, isNum := args[0].(float64); isNum {
					g = int(f)
				}
			}
			slot := 2 * g
			if name == "end" {
				slot++
			}
			if slot < len(m.caps) {
				return float64(m.caps[slot]), true
			}
			return jsNull, true
		case "size", "length":
			return float64(re.ngroups + 1), true
		case "to_s":
			g, _ := re.group(text, m, 0)
			return g, true
		case "inspect":
			g, _ := re.group(text, m, 0)
			return "#<MatchData \"" + g + "\">", true
		case "class":
			return "MatchData", true
		}
		rt.fail("unknown MatchData method: %s", name)
	}
	// A String receiver whose argument is a pattern: the pattern-taking half of
	// Ruby's String methods, which the base js_rmcall knows nothing about.
	if s, isStr := target.(string); isStr && len(args) > 0 {
		switch name {
		case "match", "match?", "=~", "scan", "sub", "gsub", "sub!", "gsub!", "split", "index":
			reObj := args[0]
			if !rxIsRegexObj(reObj) {
				if name == "split" || name == "index" {
					return nil, false // a plain-string argument stays the base behaviour
				}
				reObj = rxNewRegexObj(rt.rubyStr(args[0]), "m")
			}
			switch name {
			case "match":
				md := rt.rxRunMatch(reObj, s, 0)
				if md == nil {
					return jsNull, true
				}
				return md, true
			case "match?":
				m, ok := rxObjRe(reObj).search([]rune(s), 0)
				if !ok {
					rt.fail("regexp: step limit exceeded")
				}
				return m != nil, true
			case "=~":
				return rt.rxMatchOp(reObj, s), true
			case "scan":
				return rt.rxScan(reObj, s), true
			case "sub", "sub!":
				return rt.rxReplace(reObj, s, rt.rubyStr(argAt(args, 1)), false), true
			case "gsub", "gsub!":
				return rt.rxReplace(reObj, s, rt.rubyStr(argAt(args, 1)), true), true
			case "split":
				return rt.rxSplit(reObj, s), true
			case "index":
				return rt.rxMatchOp(reObj, s), true
			}
		}
	}
	return nil, false
}

// rxMatchOp is the =~ operator: either operand may be the pattern, and the answer
// is the INDEX of the match (not a boolean) or nil.
func (rt *jsrt) rxMatchOp(l, r interface{}) interface{} {
	reObj, subj, ok := rxMatchOperand(l, r)
	if !ok {
		rt.fail("=~ needs a Regexp on one side")
	}
	if isUndefOrNull(subj) == true {
		rt.rxSetGlobals(nil)
		return jsNull
	}
	md := rt.rxRunMatch(reObj, rt.rubyStr(subj), 0)
	if md == nil {
		return jsNull
	}
	_, _, m := rxMatchOf(md)
	return float64(m.begin)
}

// rxExtraExterns is the extension point for the PER-LANGUAGE method dispatchers.
// The engine above is language-neutral, but a compiler grammar still needs a
// receiver-and-name dispatch table in Go (the Ruby one is rxRegexMethod), and one
// table per language would otherwise mean editing this file once per language.
// Instead each sibling file (jsrtregexpy.go, jsrtregexkt.go, jsrtregexjs.go)
// appends its own registrar in an init(), and addRegexExterns runs them all. Purely
// additive: a registrar only ADDS names, and a language whose file is absent is
// unaffected.
var rxExtraExterns []func(rt *jsrt, m map[string]func(args []uint64) uint64)

// addRegexExterns is the ONE hook into abnf/jsrt.go. It only ADDS js_rx* names;
// the three wrappers capture the extern they extend and delegate to it unchanged,
// so no existing name is rebound and no other language's compiler can notice this
// file exists.
func (rt *jsrt) addRegexExterns(m map[string]func(args []uint64) uint64) {
	u := rt.unwrap
	w := rt.wrap
	boolH := func(b bool) uint64 {
		if b {
			return jsHTrue
		}
		return jsHFalse
	}

	// A regexp literal: /src/flags, validated here so a bad pattern is a clean
	// runtime error rather than a wrong match.
	m["js_rxnew"] = func(a []uint64) uint64 {
		src, flags := rt.toString(u(a[0])), rt.toString(u(a[1]))
		re := rxGet(src, flags)
		if !re.ok {
			rt.fail("invalid regexp /%s/: %s", src, re.err)
		}
		return w(rxNewRegexObj(src, flags))
	}
	// Pattern validity without building anything: "" when the pattern compiles.
	m["js_rxcheck"] = func(a []uint64) uint64 {
		re := rxGet(rt.toString(u(a[0])), rt.toString(u(a[1])))
		if re.ok {
			return rt.wrapStr("")
		}
		return rt.wrapStr(re.err)
	}

	baseMcall := m["js_rmcall"]
	m["js_rxmcall"] = func(a []uint64) uint64 {
		arr, ok := u(a[2]).(*jsArray)
		if !ok {
			rt.fail("js_rxmcall args must be an array")
		}
		target := u(a[0])
		name := rt.toString(u(a[1]))
		if rxIsRegexObj(target) || rxIsMatchObj(target) || rxIsRegexObj(argAt(arr.elems, 0)) {
			if v, handled := rt.rxRegexMethod(target, name, arr.elems); handled {
				return w(v)
			}
		}
		return baseMcall(a)
	}

	baseCase := m["js_rcase"]
	m["js_rxcase"] = func(a []uint64) uint64 {
		if rxIsRegexObj(u(a[0])) {
			return boolH(rt.rxMatchOp(u(a[0]), u(a[1])) != jsNull)
		}
		return baseCase(a)
	}

	baseWhen := m["js_rwhen"]
	m["js_rxwhen"] = func(a []uint64) uint64 {
		if rxIsRegexObj(u(a[0])) {
			return boolH(rt.rxMatchOp(u(a[0]), u(a[1])) != jsNull)
		}
		return baseWhen(a)
	}

	baseGet := m["js_rget"]
	m["js_rxget"] = func(a []uint64) uint64 {
		if rxIsMatchObj(u(a[0])) {
			return w(rt.rxMdIndex(u(a[0]), u(a[1])))
		}
		return baseGet(a)
	}

	// The =~ / !~ operators.
	m["js_rxmatch"] = func(a []uint64) uint64 { return w(rt.rxMatchOp(u(a[0]), u(a[1]))) }
	m["js_rxnmatch"] = func(a []uint64) uint64 { return boolH(rt.rxMatchOp(u(a[0]), u(a[1])) == jsNull) }

	// The per-language dispatchers (see rxExtraExterns above).
	for _, add := range rxExtraExterns {
		add(rt, m)
	}
}

func init() {
	// js_rxmcall reads its argument array in position 2, exactly like the
	// js_rmcall it wraps, so the handle recycler may hand that array through.
	jsThroughArgs["js_rxmcall"] = 1 << 2
}
