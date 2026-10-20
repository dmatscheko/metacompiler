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
// runs unless a program calls one of the js_rx* externs, which only the compiler
// grammars that have regexps emit. The one hook in jsrt.go is addRegexExterns
// below, which ADDS names to the extern map and rebinds none.
//
// Positions are RUNE indices, matching the JS side's UTF-16 indices for everything
// in the Basic Multilingual Plane (and therefore for every ASCII pattern).

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

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
	rxOpClear   = 12 // caps[2a .. 2b+1] = -1 (per-iteration capture reset; j only)
	rxOpClassX  = 13 // match one character against the class TREE clss[pc] (J only)
	rxOpCInit   = 14 // counted loop a starts: its count is 0
	rxOpCLoop   = 15 // counted loop a: another iteration, or leave at b
	rxOpCInc    = 16 // counted loop a: one iteration done, back to b
	rxOpLookB   = 17 // lookbehind: a = address after the body, b = 2 * id + negative
	rxOpLBEnd   = 18 // the lookbehind body must end where the lookbehind stands
	rxOpAtom    = 19 // atomic group: a = address after the body (J only)
	rxOpAtomEnd = 20 //
	rxOpGPos    = 21 // \G: where the previous match ended (J only)
	rxOpXGraph  = 22 // \X: one extended grapheme cluster (J only)
	rxOpGBound  = 23 // \b{g}: a grapheme cluster boundary, as Java finds one (J only)
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
	rxMBOL   = 8 // ^ under P with m in force
	rxMEOL   = 9 // $ under P with m in force
	// java.util.regex's anchors (J); see RX_JBOL .. RX_JUNWORDB in lib/regex.js.
	rxJBOL     = 10
	rxJUBOL    = 11
	rxJEOL     = 12
	rxJUEOL    = 13
	rxJMEOL    = 14
	rxJUMEOL   = 15
	rxJWordB   = 16
	rxJNWordB  = 17
	rxJUWordB  = 18
	rxJUNWordB = 19
	// \b and \B under P without a: Python's Unicode \w (rxPyWord).
	rxPUWordB  = 20
	rxPUNWordB = 21
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
	// Java's \h, \H, \v and \V: horizontal and vertical whitespace.
	rxSJH  = 9
	rxSJNH = 10
	rxSJV  = 11
	rxSJNV = 12
	// Python's Unicode \w and \W (rxPyWord), \d and \D (pyCpDecimal), \s and \S
	// (pysIsSpaceCode).
	rxSPW  = 13
	rxSPNW = 14
	rxSPD  = 15
	rxSPND = 16
	rxSPS  = 17
	rxSPNS = 18
)

// rxSteps caps one rxSearch attempt, so a pathological pattern reports an error
// instead of hanging the run. Same number as RX_STEPS in lib/regex.js.
const rxSteps = 400000

// ===== Character classes =====

// Under J a class is a TREE (see lib/regex.js): subs are nested classes in the
// union, ands the operands of &&, props the \p{..} properties, and x marks a class
// using any of them, which compiles to rxOpClassX.
type rxCls struct {
	neg   bool
	lo    []rune
	hi    []rune
	sp    []int
	subs  []*rxCls
	ands  []*rxCls
	props []int
	x     bool
	// u marks ranges read under Java's (?iu) (see lib/regex.js); pu additionally
	// under P, whose ASCII letters are folded already (fold).
	u, pu bool
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

// rxPosixAdd adds the POSIX bracket class `name` ([:alpha:] and friends) to cls and
// answers whether the name is known. These exist only in the ERE dialect (flag e):
// JavaScript, Python and Java all read [[:alpha:]] as an ordinary nested-looking class,
// and Ruby is the only one of the five that has them - so the engine keeps reporting
// them as unsupported unless the caller asked for ERE.
//
// The definitions are the C locale's, and deliberately the SAME ones the emitted ERE
// engine in bash-to-llvm-ir.abnf carries in its re_ctype, so the two halves cannot
// drift. Every one is expressible as plain ranges, so no new shorthand opcode is needed.
func rxPosixAdd(cls *rxCls, name string) bool {
	switch name {
	case "alpha":
		cls.add('A', 'Z')
		cls.add('a', 'z')
	case "digit":
		cls.add('0', '9')
	case "alnum":
		cls.add('0', '9')
		cls.add('A', 'Z')
		cls.add('a', 'z')
	case "upper":
		cls.add('A', 'Z')
	case "lower":
		cls.add('a', 'z')
	case "space":
		cls.add(9, 13)
		cls.add(' ', ' ')
	case "blank":
		cls.add(9, 9)
		cls.add(' ', ' ')
	case "punct":
		cls.add(33, 47)
		cls.add(58, 64)
		cls.add(91, 96)
		cls.add(123, 126)
	case "print":
		cls.add(32, 126)
	case "graph":
		cls.add(33, 126)
	case "cntrl":
		cls.add(0, 31)
		cls.add(127, 127)
	case "xdigit":
		cls.add('0', '9')
		cls.add('A', 'F')
		cls.add('a', 'f')
	case "word":
		cls.add('0', '9')
		cls.add('A', 'Z')
		cls.add('_', '_')
		cls.add('a', 'z')
	case "ascii":
		cls.add(0, 127)
	default:
		return false
	}
	return true
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
	case rxSJH:
		return rxIsHSpaceJ(c)
	case rxSJNH:
		return !rxIsHSpaceJ(c)
	case rxSJV:
		return rxIsVSpaceJ(c)
	case rxSJNV:
		return !rxIsVSpaceJ(c)
	case rxSPW:
		return rxPyWord(c)
	case rxSPNW:
		return !rxPyWord(c)
	case rxSPD:
		return pyCpDecimal(int(c))
	case rxSPND:
		return !pyCpDecimal(int(c))
	case rxSPS:
		return pysIsSpaceCode(int(c))
	case rxSPNS:
		return !pysIsSpaceCode(int(c))
	}
	return false
}

// rxPyWord answers whether c is a word character of a Python str pattern: "_"
// or what str.isalnum() accepts. Twin of the rxPyWordHit knob of lib/regex.js.
func rxPyWord(c rune) bool {
	return rxIsWord(c) || (c >= 128 && (pyCpAlpha(int(c)) || pyCpNumeric(int(c))))
}

// java.util.regex's \h and \v. Twins of rxIsHSpaceJ / rxIsVSpaceJ in lib/regex.js.
func rxIsHSpaceJ(c rune) bool {
	switch c {
	case ' ', '\t', 0xA0, 0x1680, 0x180E, 0x202F, 0x205F, 0x3000:
		return true
	}
	return c >= 0x2000 && c <= 0x200A
}

func rxIsVSpaceJ(c rune) bool {
	return (c >= '\n' && c <= '\r') || c == 0x85 || c == 0x2028 || c == 0x2029
}

// hitX is hit for a class TREE: the union of the class's own ranges, its
// properties and its nested classes, intersected with every && operand.
func (c *rxCls) hitX(ch rune) bool {
	hit := c.hit(ch)
	for i := 0; !hit && i < len(c.props); i++ {
		hit = rxPropHit(c.props[i], ch)
	}
	for i := 0; !hit && i < len(c.subs); i++ {
		hit = c.subs[i].hitX(ch) != c.subs[i].neg
	}
	for i := 0; hit && i < len(c.ands); i++ {
		hit = c.ands[i].hitX(ch) != c.ands[i].neg
	}
	return hit
}

func (c *rxCls) matchX(ch rune, icase bool) bool {
	hit := c.hitX(ch)
	if !hit && icase {
		if alt := rxSwapCase(ch); alt >= 0 {
			hit = c.hitX(alt)
		}
	}
	return hit != c.neg
}

// fold adds the other case of every ASCII letter in the class's ranges. Twin of
// rxClsFold in lib/regex.js.
func (c *rxCls) fold() {
	n := len(c.lo)
	for i := 0; i < n; i++ {
		for _, r := range [2][2]rune{{'A', 'Z'}, {'a', 'z'}} {
			a, b := c.lo[i], c.hi[i]
			if a < r[0] {
				a = r[0]
			}
			if b > r[1] {
				b = r[1]
			}
			if a <= b {
				d := rune(32)
				if r[0] == 'a' {
					d = -32
				}
				c.add(a+d, b+d)
			}
		}
	}
}

// foldP is Python's Unicode IGNORECASE of a class. Twin of rxClsFoldP in
// lib/regex.js.
func (c *rxCls) foldP() {
	n := len(c.lo)
	for i := 0; i < n; i++ {
		a := c.lo[i]
		if a < 128 {
			a = 128
		}
		if c.hi[i]-a >= 256 {
			continue
		}
		for r := a; r <= c.hi[i]; r++ {
			l, u := unicode.ToLower(r), unicode.ToUpper(r)
			for _, v := range [4]rune{l, u, unicode.ToUpper(l), unicode.ToLower(u)} {
				c.add(v, v)
			}
		}
	}
	c.u, c.pu = true, true
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
	if c.u && !(c.pu && ch < 128) {
		return c.hitU(ch)
	}
	return false
}

// hitU is the rest of a u class's range test: the simple uppercase of ch, then
// that uppercase's lowercase. Twin of rxClsHitU in lib/regex.js.
func (c *rxCls) hitU(ch rune) bool {
	v := unicode.ToUpper(ch)
	for k := 0; k < 2; k++ {
		for i := range c.lo {
			if v >= c.lo[i] && v <= c.hi[i] {
				return true
			}
		}
		v = unicode.ToLower(v)
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
	// bname carries the NAME of a \k<name> backreference until rxCompile resolves
	// it to a group number; it stays "" for every other node.
	bname string
	// cnt marks a J {n,m} repeat, which is emitted as a counted loop.
	cnt bool
	// closed marks a capturing group whose ")" has been read (P's open-group
	// reference check).
	closed bool
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
	// pending holds the \k<name> nodes still waiting for their group number.
	pending []*rxNode
	ext     bool
	dotall  bool
	// octal is the o flag (JavaScript Annex B legacy octal escapes), and ntotal the
	// capturing-group count from the rxCountGroups pre-pass.
	octal  bool
	ntotal int
	// ere is the POSIX ERE dialect (flag e) and depth the open-group nesting it needs
	// to tell a closing ) from a literal one.
	ere bool
	// posix is the p flag: POSIX bracket expressions are recognised. e implies it.
	posix bool
	depth int
	// java is the J flag; errAt the error's index (-1: none), and pcls the class
	// of the \p{..} escape just read.
	java  bool
	errAt int
	pcls  *rxCls
	// py is the P flag; icase and multi the i and m in force where the parser
	// stands, gend the end of the leading global flag groups and gflags their
	// letters.
	py     bool
	icase  bool
	multi  bool
	gend   int
	gflags string
	// ucase, uclass and unix are J's UNICODE_CASE, UNICODE_CHARACTER_CLASS and
	// UNIX_LINES where the parser stands; inl marks an inline (?flags) group,
	// whose flags outlive it.
	ucase  bool
	uclass bool
	unix   bool
	inl    bool
	// supp collects java.util.regex's hasSupplementary while parsing (see
	// rxSuppJ), and canon is J's CANON_EQ, which is only reported (Pattern.flags).
	supp  bool
	canon bool
	// pyuni: under P, \w \W \b \B \d \D \s \S are Python's Unicode ones (no a
	// flag in force).
	pyuni bool
	// gnodes[g] is capturing group g's node (rxWidthP), from index 1.
	gnodes []*rxNode
	// re2 is the R flag: the J pattern is rxRe2's rewrite of a Go one.
	re2 bool
}

// rxIsMetaCh answers whether a character has to be backslashed to stay a literal.
// Whitespace and # are included because the x (extended) flag would otherwise drop
// them, and - because it is a range operator inside a character class.
func rxIsMetaCh(ch rune) bool {
	return strings.ContainsRune("\\^$.|?*+()[]{}-#/ \t\n\r\f\v", ch)
}

// rxQuoteText is the pattern that matches the text s literally: every
// metacharacter backslashed. Twins: rxQuoteLit in ruby-interpreter.abnf and
// rbRxQuoteLit in lib/ruby-rt.metajs.
func rxQuoteText(s string) string {
	var b strings.Builder
	for _, ch := range s {
		if rxIsMetaCh(ch) {
			b.WriteByte('\\')
		}
		b.WriteRune(ch)
	}
	return b.String()
}

// rxExpandQuotes rewrites every \Q...\E region into the equivalent escaped text, as a
// PRE-PASS over the pattern rather than a parser mode. An unterminated \Q runs to the
// end of the pattern (Java's rule). Doing it here means a quantifier after \E binds to
// the LAST quoted character - \Qab\E+ is ab+ - which is also Java's rule, and it keeps
// the parser itself free of a quoting state that every rule would have to respect.
func rxExpandQuotes(p []rune) []rune {
	if !strings.Contains(string(p), `\Q`) {
		return p
	}
	var out []rune
	for i := 0; i < len(p); {
		if p[i] != '\\' || i+1 >= len(p) {
			out = append(out, p[i])
			i++
			continue
		}
		if p[i+1] != 'Q' {
			out = append(out, p[i], p[i+1])
			i += 2
			continue
		}
		i += 2
		for i < len(p) {
			if p[i] == '\\' && i+1 < len(p) && p[i+1] == 'E' {
				i += 2
				break
			}
			if rxIsMetaCh(p[i]) {
				out = append(out, '\\')
			}
			out = append(out, p[i])
			i++
		}
	}
	return out
}

// rxCountGroups is a cheap PRE-PASS over the pattern, counting the capturing groups
// before the real parse starts. A backslash-digit escape can only be read correctly
// once the total is known: \1 in a pattern with one group is a backreference, and in a
// pattern with none it is either a legacy octal escape (JavaScript, flag o) or an
// error (everywhere else) - and the octal reading has to CONSUME the following digits,
// which the parser cannot undo after the fact.
func rxCountGroups(p []rune) int {
	n := 0
	inCls := false
	for i := 0; i < len(p); {
		c := p[i]
		if c == '\\' {
			i += 2
			continue
		}
		if inCls {
			if c == ']' {
				inCls = false
			}
			i++
			continue
		}
		if c == '[' {
			inCls = true
			i++
			continue
		}
		if c != '(' {
			i++
			continue
		}
		if i+1 >= len(p) || p[i+1] != '?' {
			n++
			i++
			continue
		}
		// (?<name> and (?'name' capture; (?<= and (?<! are lookbehind, and every
		// other (? form is non-capturing.
		var c2, c3 rune
		if i+2 < len(p) {
			c2 = p[i+2]
		}
		if i+3 < len(p) {
			c3 = p[i+3]
		}
		if c2 == '\'' || (c2 == '<' && c3 != '=' && c3 != '!') {
			n++
		}
		i += 2
	}
	return n
}

// nameIdx is the group NUMBER a name was declared for, or -1.
func (st *rxSt) nameIdx(name string) int {
	for i, n := range st.names {
		if n == name {
			return st.nameGroups[i]
		}
	}
	return -1
}

func (st *rxSt) fail(msg string) {
	if st.err == "" {
		st.err = msg
	}
}

// failAt is fail with the error's index into the pattern (re.errAt).
func (st *rxSt) failAt(msg string, at int) {
	if st.err == "" {
		st.err = msg
		st.errAt = at
	}
}

// octAt is the octal digit at index i of the pattern, or -1.
func (st *rxSt) octAt(i int) int {
	if i >= st.n || st.p[i] < '0' || st.p[i] > '7' {
		return -1
	}
	return int(st.p[i] - '0')
}

// The \p{..} property ids rxPropHit answers for. Twins of RX_PL .. RX_PBLK in
// lib/regex.js, which says what each is.
const (
	rxPL        = 1
	rxPLu       = 2
	rxPLl       = 3
	rxPLt       = 4
	rxPND       = 5
	rxPNl       = 6
	rxPAlpha    = 7
	rxPUpper    = 8
	rxPLower    = 9
	rxPLD       = 10
	rxPWord     = 11
	rxPWhite    = 12
	rxPHex      = 13
	rxPJoin     = 14
	rxPNonCh    = 15
	rxPAssigned = 16
	rxPGraph    = 17
	rxPPrint    = 18
	rxPBlank    = 19
	rxPCased    = 20
	rxPAlnum    = 21
	rxPJStart   = 22
	rxPJPart    = 23
	rxPUStart   = 24
	rxPUPart    = 25
	rxPIgnore   = 26
	rxPJWhite   = 27
	rxPBin      = 32
	rxPGc       = 64
	rxPMn       = 70
	rxPSc       = 128
	rxPBlk      = 512
)

// escJ reads the escapes java.util.regex spells differently; twin of rxEscJ in
// lib/regex.js. c (already consumed) is the character after the backslash, and
// handled is false for the escapes every dialect shares.
func (st *rxSt) escJ(c rune, inClass bool) (res rxEscRes, handled bool) {
	p := st.p
	// UNICODE_CHARACTER_CLASS: \d \w \s and their complements are Unicode's.
	if st.uclass && strings.ContainsRune("dDwWsS", c) {
		id := rxPWhite
		if c == 'd' || c == 'D' {
			id = rxPND
		} else if c == 'w' || c == 'W' {
			id = rxPWord
		}
		st.pcls = &rxCls{props: []int{id}, x: true, neg: c == 'D' || c == 'W' || c == 'S'}
		return rxEscRes{kind: 3}, true
	}
	switch c {
	case 'h':
		return rxEscRes{kind: 1, sp: rxSJH}, true
	case 'H':
		return rxEscRes{kind: 1, sp: rxSJNH}, true
	case 'v':
		return rxEscRes{kind: 1, sp: rxSJV}, true
	case 'V':
		return rxEscRes{kind: 1, sp: rxSJNV}, true
	case 'c': // \cX is X with bit 6 flipped
		if st.i >= st.n {
			st.failAt("Illegal control escape sequence", st.i-1)
			return res, true
		}
		res.ch = p[st.i] ^ 64
		st.i++
		return res, true
	case '0': // \0n, \0nn, \0mnn (m <= 3)
		on := st.octAt(st.i)
		if on < 0 {
			st.failAt("Illegal octal escape sequence", st.i)
			return res, true
		}
		st.i++
		res.ch = rune(on)
		om := st.octAt(st.i)
		if om < 0 {
			return res, true
		}
		st.i++
		res.ch = rune(on*8 + om)
		oo := st.octAt(st.i)
		if oo < 0 || on > 3 {
			return res, true
		}
		st.i++
		res.ch = rune(on*64 + om*8 + oo)
		return res, true
	case 'x': // \xhh and \x{h...}
		x0 := st.i
		xa := -1
		if x0 < st.n {
			xa = rxHexVal(p[x0])
		}
		if xa >= 0 {
			xb := -1
			if x0+1 < st.n {
				xb = rxHexVal(p[x0+1])
			}
			if xb < 0 {
				st.failAt("Illegal hexadecimal escape sequence", x0+1)
				return res, true
			}
			st.i = x0 + 2
			res.ch = rune(xa*16 + xb)
			return res, true
		}
		if x0+1 < st.n && p[x0] == '{' && rxHexVal(p[x0+1]) >= 0 {
			xv := 0
			st.i = x0 + 1
			for st.i < st.n && rxHexVal(p[st.i]) >= 0 {
				xv = xv*16 + rxHexVal(p[st.i])
				if xv > 0x10FFFF {
					st.failAt("Hexadecimal codepoint is too big", st.i)
					return res, true
				}
				st.i++
			}
			if st.i >= st.n || p[st.i] != '}' {
				st.failAt("Unclosed hexadecimal escape sequence", st.i)
				return res, true
			}
			st.i++
			res.ch = rune(xv)
			return res, true
		}
		st.failAt("Illegal hexadecimal escape sequence", x0)
		return res, true
	case 'u': // \uhhhh: exactly four
		uv := 0
		for uk := 0; uk < 4; uk++ {
			uh := -1
			if st.i < st.n {
				uh = rxHexVal(p[st.i])
			}
			if uh < 0 {
				st.failAt("Illegal Unicode escape sequence", st.i)
				return res, true
			}
			uv = uv*16 + uh
			st.i++
		}
		res.ch = rune(uv)
		// A high surrogate escape and a low surrogate escape after it are one
		// code point, as Pattern.u() reads them.
		if isHighSurr(res.ch) && st.i+5 < st.n && p[st.i] == '\\' && p[st.i+1] == 'u' {
			lv, lk := 0, 2
			for lk < 6 && rxHexVal(p[st.i+lk]) >= 0 {
				lv = lv*16 + rxHexVal(p[st.i+lk])
				lk++
			}
			if lk == 6 && isLowSurr(rune(lv)) {
				res.ch = surrPair(res.ch, rune(lv))
				st.i += 6
			}
		}
		return res, true
	case 'p', 'P': // \pX, \p{Name}, and \P for the complement
		res.kind = 3
		pname := ""
		pat := st.i
		if st.i < st.n && p[st.i] == '{' {
			pe := st.i + 1
			for pe < st.n && p[pe] != '}' {
				pe++
			}
			if pe >= st.n {
				st.failAt("Unclosed character family", st.n)
				return res, true
			}
			if pe == st.i+1 {
				st.failAt("Empty character family", pe)
				return res, true
			}
			pname = string(p[st.i+1 : pe])
			pat = pe
			st.i = pe + 1
		} else if st.i < st.n {
			pname = string(p[st.i])
			st.i++
		} else {
			pname = "\x00"
		}
		st.pcls = st.propCls(pname, pat)
		st.supp = true
		st.pcls.neg = c == 'P'
		return res, true
	}
	if c >= '1' && c <= '9' { // \n..: as many digits as name a group opened so far
		if inClass {
			st.failAt("Illegal/unsupported escape sequence", st.i-1)
			return res, true
		}
		g := int(c - '0')
		for st.i < st.n && rxIsDigit(p[st.i]) && g*10+int(p[st.i]-'0') <= st.ngroups {
			g = g*10 + int(p[st.i]-'0')
			st.i++
		}
		return rxEscRes{kind: 2, ch: rune(g)}, true
	}
	// The letters java.util.regex gives no meaning (in a class, also the
	// assertions and \k, \R, \X, which only stand outside one) are errors;
	// \N{name} is Java's but not this engine's: it has no character name table.
	if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
		if strings.ContainsRune("dDwWsSntrfae", c) {
			return res, false
		}
		if !inClass && c == 'N' {
			st.fail("unsupported escape \\" + string(c))
			return res, true
		}
		st.failAt("Illegal/unsupported escape sequence", st.i-1)
		return res, true
	}
	return res, false
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
		} else if c == '#' { // (under J up to any line terminator)
			for st.i < st.n && st.p[st.i] != '\n' && !(st.java && rxIsLineSepJ(st.p[st.i], st.unix)) {
				st.i++
			}
		} else {
			return
		}
	}
}

// skipXAt is skipX from index j; twin of rxSkipXAt in lib/regex.js.
func (st *rxSt) skipXAt(j int) int {
	st.i = j
	st.skipX()
	return st.i
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
	kind int // 0 literal character, 1 shorthand class, 2 backreference, 3 \p{..} (st.pcls)
	ch   rune
	sp   int
}

// rxBadEscP: is \c an escape Python's re refuses ("bad escape")? Twin of
// rxBadEscP in lib/regex.js, which explains.
func rxBadEscP(c rune, inClass bool) bool {
	if c >= '0' && c <= '9' {
		return inClass && c >= '8'
	}
	if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')) {
		return false
	}
	ok := "abdfknrstuvwxzABDNSUWZ"
	if inClass {
		ok = "abdfnrstuvwxDNSUW"
	}
	return !strings.ContainsRune(ok, c)
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
	if st.java {
		if r, ok := st.escJ(c, inClass); ok {
			return r
		}
	}
	if st.py && rxBadEscP(c, inClass) {
		st.fail("bad escape \\" + string(c))
		return res
	}
	if st.py && c == 'U' { // \UHHHHHHHH
		v := 0
		for got := 0; got < 8 && st.i < st.n && rxHexVal(st.p[st.i]) >= 0; got++ {
			v = v*16 + rxHexVal(st.p[st.i])
			st.i++
		}
		return rxEscRes{ch: rune(v)}
	}
	switch c {
	case 'd':
		if st.pyuni {
			return rxEscRes{kind: 1, sp: rxSPD}
		}
		return rxEscRes{kind: 1, sp: rxSD}
	case 'D':
		if st.pyuni {
			return rxEscRes{kind: 1, sp: rxSPND}
		}
		return rxEscRes{kind: 1, sp: rxSND}
	case 'w':
		if st.pyuni {
			return rxEscRes{kind: 1, sp: rxSPW}
		}
		return rxEscRes{kind: 1, sp: rxSW}
	case 'W':
		if st.pyuni {
			return rxEscRes{kind: 1, sp: rxSPNW}
		}
		return rxEscRes{kind: 1, sp: rxSNW}
	case 's':
		if st.pyuni {
			return rxEscRes{kind: 1, sp: rxSPS}
		}
		return rxEscRes{kind: 1, sp: rxSS}
	case 'S':
		if st.pyuni {
			return rxEscRes{kind: 1, sp: rxSPNS}
		}
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
		// A backreference only if the pattern HAS that group (rxCountGroups counted
		// them before the parse began).
		// Python's group must be opened before the reference and closed by it.
		if st.py && int(c-'0') <= st.ntotal {
			if int(c-'0') > st.ngroups {
				st.fail("invalid group reference " + string(c))
			} else if !st.gnodes[c-'0'].closed {
				st.fail("cannot refer to an open group")
			}
		}
		if int(c-'0') <= st.ntotal {
			return rxEscRes{kind: 2, ch: c - '0'}
		}
		if !st.octal {
			st.fail("invalid group reference " + string(c))
			return res
		}
		// JavaScript Annex B. 8 and 9 are not octal digits, so they are an identity
		// escape; 0-3 admit three octal digits and 4-7 only two, which is what makes
		// node read /\400/ as the two characters \40 and "0".
		if c > '7' {
			return rxEscRes{ch: c}
		}
		oval := c - '0'
		omax := 2
		if c <= '3' {
			omax = 3
		}
		for ogot := 1; ogot < omax && st.i < st.n; ogot++ {
			oc := st.p[st.i]
			if oc < '0' || oc > '7' {
				break
			}
			oval = oval*8 + (oc - '0')
			st.i++
		}
		return rxEscRes{ch: oval}
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
		// A POSIX bracket [:alpha:] is a real class under the "e" (POSIX ERE) and
		// "p" (POSIX classes only, which is Ruby) dialects, and is not supported
		// anywhere else; without one of those, report it rather than silently
		// reading it as a set of punctuation characters.
		if c == '[' && st.i+1 < st.n && st.p[st.i+1] == ':' {
			if !st.posix {
				st.fail("POSIX bracket expressions are not supported")
				return node
			}
			pj := st.i + 2
			for pj+1 < st.n && !(st.p[pj] == ':' && st.p[pj+1] == ']') {
				pj++
			}
			if pj+1 >= st.n {
				st.fail("Unmatched [, [^, [:, [., or [=")
				return node
			}
			if !rxPosixAdd(cls, string(st.p[st.i+2:pj])) {
				st.fail("Invalid character class name")
				return node
			}
			st.i = pj + 2
			// A POSIX class is a SET, so it cannot be the low end of a range: bash
			// says "invalid character range" for [[:alpha:]-z] and Ruby refuses it
			// with "unmatched range specifier in char-class". A trailing "-]" is
			// still a literal hyphen.
			if st.posix && st.i+1 < st.n && st.p[st.i] == '-' && st.p[st.i+1] != ']' {
				st.fail("Invalid range end")
				return node
			}
			continue
		}
		var loCode rune
		// In a POSIX ERE a backslash inside a bracket expression is an ORDINARY
		// CHARACTER: [a\-b] is {a} plus the range \ .. b, and [a\] closes on the ],
		// leaving a literal backslash in the set. Every other dialect here escapes.
		if c == '\\' && !st.ere {
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
			// ... nor the high end: [a-[:digit:]] is the same error.
			if st.posix && c2 == '[' && st.i+1 < st.n && st.p[st.i+1] == ':' {
				st.fail("Invalid range end")
				return node
			}
			if c2 == '\\' && !st.ere {
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

// parseClassJ reads a java.util.regex class; the [ has already been consumed.
// Twin of rxParseClassJ in lib/regex.js.
func (st *rxSt) parseClassJ() *rxNode {
	node := rxNewNode("cls")
	node.cls = st.classJ(true)
	node.cls.simplify()
	return node
}

// classJ is java.util.regex's clazz(), its precedence and its shared set of
// single characters below U+0100 included; twin of rxClassJ in lib/regex.js,
// which explains both.
func (st *rxSt) classJ(consume bool) *rxCls {
	p := st.p
	var prev, curr *rxCls
	bits := rxNewCls()
	hasBits := false
	neg := false
	if st.i < st.n && p[st.i] == '^' && p[st.i-1] == '[' {
		neg = true
		st.i++
	}
	for st.err == "" {
		st.skipX()
		if st.i >= st.n {
			st.fail("unterminated character class")
			return rxNewCls()
		}
		c := p[st.i]
		if c == '[' {
			st.i++
			curr = st.classJ(true)
			if prev != nil {
				prev = rxClsOr(prev, curr)
			} else {
				prev = curr
			}
			continue
		}
		if c == '&' && st.i+1 < st.n && p[st.i+1] == '&' {
			st.i += 2
			var right *rxCls
			st.skipX()
			for st.err == "" && st.i < st.n && p[st.i] != ']' && p[st.i] != '&' {
				var r *rxCls
				if p[st.i] == '[' {
					st.i++
					r = st.classJ(true)
				} else {
					r = st.classJ(false)
				}
				if right != nil {
					right = rxClsOr(right, r)
				} else {
					right = r
				}
				st.skipX()
			}
			if st.err != "" {
				return rxNewCls()
			}
			if hasBits {
				if prev != nil {
					prev = rxClsOr(prev, bits)
				} else {
					prev, curr = bits, bits
				}
				hasBits = false
			}
			if right != nil {
				curr = right
			}
			if prev == nil {
				if right == nil {
					st.failAt("Bad class syntax", st.i-1)
					return rxNewCls()
				}
				prev = right
			} else {
				if curr == nil {
					st.failAt("Bad intersection syntax", st.i-1)
					return rxNewCls()
				}
				prev = rxClsAnd(prev, curr)
			}
			continue
		}
		if c == ']' && (prev != nil || hasBits) {
			if consume {
				st.i++
			}
			if prev == nil {
				prev = bits
			} else if hasBits {
				prev = rxClsOr(prev, bits)
			}
			if neg {
				return rxClsNot(prev)
			}
			return prev
		}
		item := st.rangeJ(bits)
		if st.err != "" {
			return rxNewCls()
		}
		if item == nil {
			hasBits = true
		} else {
			curr = item
			if prev != nil {
				prev = rxClsOr(prev, curr)
			} else {
				prev = curr
			}
		}
	}
	return rxNewCls()
}

// rangeJ is java.util.regex's range(); twin of rxRangeJ in lib/regex.js. A single
// character below U+0100 goes into bits and the answer is nil.
func (st *rxSt) rangeJ(bits *rxCls) *rxCls {
	p := st.p
	ch := p[st.i]
	if ch == '\\' {
		st.i++
		// \v as either end of a range is the one character U+000B.
		vt := st.i+1 < st.n && p[st.i] == 'v' && p[st.i+1] == '-'
		e := st.esc(true)
		switch {
		case vt:
			ch = 11
		case e.kind == 1:
			sh := rxNewCls()
			sh.addSp(e.sp)
			return sh
		case e.kind == 3:
			return st.pcls
		default:
			ch = e.ch
		}
	} else {
		st.i++
	}
	st.skipX()
	if st.i < st.n && p[st.i] == '-' {
		var end rune
		if st.i+1 < st.n {
			end = p[st.i+1]
		}
		if end != '[' && end != ']' {
			st.i++
			st.skipX()
			m := rune(-1)
			if st.i >= st.n {
				st.i++
			} else if p[st.i] == '\\' {
				st.i++
				if st.i < st.n && p[st.i] == 'v' {
					st.i++
					m = 11
				} else if e2 := st.esc(true); e2.kind == 0 {
					m = e2.ch
				}
			} else {
				m = p[st.i]
				st.i++
			}
			if m < ch {
				st.failAt("Illegal character range", st.i-1)
			}
			rg := rxNewCls()
			rg.add(ch, m)
			if st.icase {
				st.foldJ(rg, false)
				st.supp = true
			}
			return rg
		}
	}
	// Pattern.bitsOrSingle; see rxRangeJ in lib/regex.js.
	excl := st.icase && st.ucase && strings.ContainsRune("\u00ff\u00b5IiSsKk\u00c5\u00e5", ch)
	if ch < 256 && !excl {
		bits.add(ch, ch)
		if st.icase && ch < 128 {
			if sw := rxSwapCase(ch); sw >= 0 {
				bits.add(sw, sw)
			}
		} else if st.icase && st.ucase {
			bits.add(unicode.ToLower(ch), unicode.ToLower(ch))
			bits.add(unicode.ToUpper(ch), unicode.ToUpper(ch))
		}
		return nil
	}
	one := rxNewCls()
	one.add(ch, ch)
	if st.icase {
		st.foldJ(one, true)
	}
	return one
}

// The union, intersection and complement of class trees, by reference.
func rxClsOr(a, b *rxCls) *rxCls  { return &rxCls{subs: []*rxCls{a, b}, x: true} }
func rxClsAnd(a, b *rxCls) *rxCls { return &rxCls{subs: []*rxCls{a}, ands: []*rxCls{b}, x: true} }
func rxClsNot(a *rxCls) *rxCls    { return &rxCls{subs: []*rxCls{a}, neg: true, x: true} }

// simplify folds every positive flat member of a union into its parent's own
// ranges; twin of rxClsSimplify in lib/regex.js.
func (c *rxCls) simplify() {
	if !c.x {
		return
	}
	var keep []*rxCls
	for _, s := range c.subs {
		s.simplify()
		if !s.x && !s.neg && !s.u {
			for k := range s.lo {
				c.add(s.lo[k], s.hi[k])
			}
			for _, sp := range s.sp {
				c.addSp(sp)
			}
		} else {
			keep = append(keep, s)
		}
	}
	c.subs = keep
	for _, a := range c.ands {
		a.simplify()
	}
	c.x = len(keep)+len(c.ands)+len(c.props) > 0
}

func (st *rxSt) readGroupName(closer rune) string {
	if st.java {
		return st.readGroupNameJ()
	}
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

// readGroupNameJ reads a java.util.regex group name, an ASCII letter and then
// letters and digits, and its closing >. Twin of rxReadGroupNameJ in lib/regex.js.
func (st *rxSt) readGroupNameJ() string {
	start := st.i
	if st.i >= st.n || !((st.p[st.i] >= 'A' && st.p[st.i] <= 'Z') || (st.p[st.i] >= 'a' && st.p[st.i] <= 'z')) {
		st.failAt("capturing group name does not start with a Latin letter", st.i)
		return ""
	}
	for st.i < st.n && rxIsWord(st.p[st.i]) && st.p[st.i] != '_' {
		st.i++
	}
	if st.i >= st.n || st.p[st.i] != '>' {
		st.failAt("named capturing group is missing trailing '>'", st.i)
		return ""
	}
	st.i++
	return string(st.p[start : st.i-1])
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
	// A POSIX ERE has no (? forms at all: the ? is simply a quantifier sitting where an
	// atom belongs, which is the error bash reports for (?:a).
	if st.i < st.n && st.p[st.i] == '?' && !st.ere {
		st.i++
		if st.i >= st.n {
			if st.java {
				st.failAt("Unknown inline modifier", st.i)
			} else {
				st.fail("unterminated group")
			}
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
		case k == '#' && !st.java: // java.util.regex has no (?# comment
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
		case k == '<' || (k == '\'' && !st.java):
			if k == '<' && st.i+1 < st.n {
				k2 := st.p[st.i+1]
				if (k2 == '=' || k2 == '!') && (st.java || st.py) {
					st.i += 2
					lb := rxNewNode("lookb")
					if k2 == '!' {
						lb.gidx = 1
					}
					lb.items = append(lb.items, st.parseAlt())
					st.expect(')', "unterminated group")
					return lb
				}
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
			if st.java && st.nameIdx(nm) >= 0 {
				st.failAt("Named capturing group <"+nm+"> is already defined", st.i-1)
			}
			if st.py && st.nameIdx(nm) >= 0 {
				st.fail("redefinition of group name '" + nm + "' as group " + strconv.Itoa(st.ngroups+1) +
					"; was group " + strconv.Itoa(st.nameIdx(nm)))
			}
			ng := rxNewNode("group")
			st.ngroups++
			ng.gidx = st.ngroups
			st.gnodes = append(st.gnodes, ng)
			st.names = append(st.names, nm)
			st.nameGroups = append(st.nameGroups, ng.gidx)
			ng.items = append(ng.items, st.parseAlt())
			st.expect(')', "unterminated group")
			ng.closed = true
			return ng
		}
		if st.py && strings.ContainsRune("aiLmsux-", k) {
			return st.parseFlagsP()
		}
		if st.java {
			if k == '>' { // (?>X): X matched once, never backtracked into
				st.i++
				at := rxNewNode("atom")
				at.items = append(at.items, st.parseAlt())
				st.expect(')', "unterminated group")
				return at
			}
			if k == '$' || k == '@' {
				st.failAt("Unknown group type", st.i)
				return rxNewNode("empty")
			}
			return st.parseFlagsJ()
		}
		st.fail("unsupported group form (?" + string(st.p[st.i]) + ")")
		return rxNewNode("empty")
	}
	cg := rxNewNode("group")
	st.ngroups++
	cg.gidx = st.ngroups
	st.gnodes = append(st.gnodes, cg)
	st.depth++
	cg.items = append(cg.items, st.parseAlt())
	st.depth--
	if st.ere {
		st.expect(')', "Unmatched ( or \\(")
	} else {
		st.expect(')', "unterminated group")
	}
	cg.closed = true
	return cg
}

// parseFlagsP reads Python's inline flags. Twin of rxParseFlagsP in
// lib/regex.js, which states the rules.
func (st *rxSt) parseFlagsP() *rxNode {
	open := st.i - 2
	on, off := "", ""
	neg := false
	for ; st.i < st.n; st.i++ {
		c := st.p[st.i]
		if c == '-' && !neg {
			neg = true
		} else if strings.ContainsRune("aiLmsux", c) {
			if neg {
				if !strings.ContainsRune("imsx", c) {
					st.fail("bad inline flag: cannot turn off flags 'a', 'u' and 'L'")
					return rxNewNode("empty")
				}
				off += string(c)
			} else {
				on += string(c)
			}
		} else {
			break
		}
	}
	if strings.Contains(on, "L") {
		st.fail("bad inline flag: cannot use 'L' flag with a str pattern")
		return rxNewNode("empty")
	}
	k := rune(0)
	if st.i < st.n {
		k = st.p[st.i]
	}
	if k == ')' && !neg {
		st.i++
		if open != st.gend {
			st.failAt("global flags not at the start of the expression", open)
			return rxNewNode("empty")
		}
		st.gend = st.i
		st.gflags += on
		st.setFlags(on, true)
		return rxNewNode("empty")
	}
	if k != ':' {
		if neg {
			st.fail("missing :")
		} else {
			st.fail("missing -, : or )")
		}
		return rxNewNode("empty")
	}
	st.i++
	icase, multi, dotall, ext, pyuni := st.icase, st.multi, st.dotall, st.ext, st.pyuni
	st.setFlags(on, true)
	st.setFlags(off, false)
	g := rxNewNode("group")
	g.items = append(g.items, st.parseAlt())
	st.expect(')', "missing ), unterminated subpattern")
	st.icase, st.multi, st.dotall, st.ext, st.pyuni = icase, multi, dotall, ext, pyuni
	return g
}

// parseFlagsJ reads java.util.regex's (?idmsuxU-idmsuxU) and (?..:X). Twin of
// rxParseFlagsJ in lib/regex.js, which states the rules.
func (st *rxSt) parseFlagsJ() *rxNode {
	on := true
	for ; st.i < st.n; st.i++ {
		c := st.p[st.i]
		if c == '-' && on {
			on = false
		} else if strings.ContainsRune("idmsuxcU", c) {
			st.setFlags(string(c), on)
		} else {
			break
		}
	}
	k := rune(0)
	if st.i < st.n {
		k = st.p[st.i]
	}
	if k == ')' {
		st.i++
		st.inl = true
		return rxNewNode("empty")
	}
	if k != ':' {
		st.failAt("Unknown inline modifier", st.i)
		return rxNewNode("empty")
	}
	st.i++
	g := rxNewNode("group")
	g.items = append(g.items, st.parseAlt())
	st.expect(')', "unterminated group")
	return g
}

// setFlags switches the parse-time flags named by letters; twin of rxFlagsSet.
func (st *rxSt) setFlags(letters string, v bool) {
	if strings.Contains(letters, "i") {
		st.icase = v
	}
	if strings.Contains(letters, "m") {
		st.multi = v
	}
	if strings.Contains(letters, "s") {
		st.dotall = v
	}
	if strings.Contains(letters, "x") {
		st.ext = v
	}
	if strings.Contains(letters, "d") {
		st.unix = v
	}
	if strings.Contains(letters, "u") {
		st.ucase = v
	}
	if strings.Contains(letters, "U") {
		st.uclass = v
		st.ucase = v
	}
	if strings.Contains(letters, "c") {
		st.canon = v
	}
	if strings.Contains(letters, "a") {
		st.pyuni = !v
	}
}

// getFlags and putFlags save and restore the flags in force; twins of
// rxFlagsGet and rxFlagsPut.
func (st *rxSt) getFlags() string {
	s := ""
	for _, f := range []struct {
		on bool
		c  string
	}{{st.icase, "i"}, {st.multi, "m"}, {st.dotall, "s"}, {st.ext, "x"}, {st.unix, "d"}, {st.ucase, "u"}, {st.uclass, "U"}, {st.canon, "c"}} {
		if f.on {
			s += f.c
		}
	}
	return s
}

func (st *rxSt) putFlags(letters string) {
	st.setFlags("imsxdUc", false)
	st.setFlags(letters, true)
}

// eolJ is the kind of a J $ (multi true) or \Z (multi false).
func (st *rxSt) eolJ(multi bool) int {
	switch {
	case st.unix && multi:
		return rxJUMEOL
	case st.unix:
		return rxJUEOL
	case multi:
		return rxJMEOL
	}
	return rxJEOL
}

// foldJ is rxFoldJ: CASE_INSENSITIVE for a J class's ranges.
func (st *rxSt) foldJ(cls *rxCls, single bool) {
	if !st.ucase {
		cls.fold()
		return
	}
	if single {
		up := unicode.ToUpper(cls.lo[0])
		low := unicode.ToLower(up)
		if up == low {
			return
		}
		cls.lo[0], cls.hi[0] = low, low
	}
	cls.u = true
}

func (st *rxSt) parseAtom() *rxNode {
	c := st.p[st.i]
	switch c {
	case '(':
		st.i++
		if !st.java {
			return st.parseGroup()
		}
		fl := st.getFlags()
		g := st.parseGroup()
		if !st.inl {
			st.putFlags(fl)
		}
		st.inl = false
		return g
	case '[':
		st.i++
		if st.java {
			return st.parseClassJ()
		}
		return st.parseClass()
	case '.':
		st.i++
		dot := rxNewNode("cls")
		dot.cls.neg = true
		if !st.dotall {
			dot.cls.add('\n', '\n')
		}
		if st.java && !st.dotall && !st.unix {
			dot.cls.add('\r', '\r')
			dot.cls.add(0x85, 0x85)
			dot.cls.add(0x2028, 0x2029)
		}
		return dot
	case '^':
		st.i++
		b := rxNewNode("assert")
		b.gidx = rxBOL
		if st.py {
			b.gidx = rxBOS
			if st.multi {
				b.gidx = rxMBOL
			}
		}
		if st.java {
			b.gidx = rxBOS
			if st.multi && st.unix {
				b.gidx = rxJUBOL
			} else if st.multi {
				b.gidx = rxJBOL
			}
		}
		return b
	case '$':
		st.i++
		d := rxNewNode("assert")
		d.gidx = rxEOL
		if st.py {
			d.gidx = rxEOSNL
			if st.multi {
				d.gidx = rxMEOL
			}
		}
		if st.java {
			d.gidx = st.eolJ(st.multi)
		}
		return d
	case '\\':
		if st.ere {
			// A POSIX ERE has no escape SEQUENCES: a backslash always quotes the next
			// character, so \d is the letter d and \1 is the digit 1.
			if st.i+1 >= st.n {
				st.fail("Trailing backslash")
				st.i++
				return rxNewNode("empty")
			}
			qc := st.p[st.i+1]
			st.i += 2
			qn := rxNewNode("cls")
			qn.cls.add(qc, qc)
			return qn
		}
		var nxt rune
		if st.i+1 < st.n {
			nxt = st.p[st.i+1]
		}
		// \k<name>: a NAMED backreference. The group it names may be declared LATER
		// in the pattern (a forward reference is legal), so the node is parked on
		// st.pending and resolved once the whole pattern has been read.
		if nxt == 'k' && st.i+2 < st.n && st.p[st.i+2] == '<' {
			st.i += 3
			kb := rxNewNode("bref")
			kb.bname = st.readGroupName('>')
			if st.java {
				// Java resolves the name as it reads it: the group must already be open.
				kb.gidx = st.nameIdx(kb.bname)
				if kb.gidx < 0 && st.err == "" {
					st.failAt("named capturing group <"+kb.bname+"> does not exist", st.i-1)
				}
				return kb
			}
			if st.py {
				// So does Python, and the group must already be closed as well.
				kb.gidx = st.nameIdx(kb.bname)
				if kb.gidx < 0 {
					st.fail("unknown group name '" + kb.bname + "'")
				} else if !st.gnodes[kb.gidx].closed {
					st.fail("cannot refer to an open group")
				}
				return kb
			}
			st.pending = append(st.pending, kb)
			return kb
		}
		if nxt == 'k' && st.java {
			st.failAt("\\k is not followed by '<' for named capturing group", st.i+2)
			return rxNewNode("empty")
		}
		if nxt == 'R' && st.java { // Java's \R: \r\n or one vertical-whitespace character
			st.i += 2
			return rxLinebreakJ()
		}
		if st.java && (nxt == 'G' || nxt == 'X') {
			st.i += 2
			op := rxNewNode("op")
			op.gidx = rxOpXGraph
			if nxt == 'G' {
				op.gidx = rxOpGPos
			}
			return op
		}
		if st.java && nxt == 'b' && st.i+3 < st.n && st.p[st.i+2] == '{' && st.p[st.i+3] == 'g' {
			// \b{g}; any other \b{ is \b and a bound.
			if st.i+4 >= st.n || st.p[st.i+4] != '}' {
				st.failAt("Illegal/unsupported escape sequence", st.i+4)
				return rxNewNode("empty")
			}
			st.i += 5
			op := rxNewNode("op")
			op.gidx = rxOpGBound
			return op
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
				if st.py {
					a.gidx = rxEOS
				}
			}
			if st.pyuni && nxt == 'b' {
				a.gidx = rxPUWordB
			}
			if st.pyuni && nxt == 'B' {
				a.gidx = rxPUNWordB
			}
			// Go's \b and \B are ASCII \w ones (R).
			if st.java && !st.re2 {
				switch nxt {
				case 'b':
					a.gidx = rxJWordB
					if st.uclass {
						a.gidx = rxJUWordB
					}
				case 'B':
					a.gidx = rxJNWordB
					if st.uclass {
						a.gidx = rxJUNWordB
					}
				case 'Z':
					a.gidx = st.eolJ(false)
				}
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
		if e.kind == 3 {
			pn := rxNewNode("cls")
			pn.cls = st.pcls
			return pn
		}
		ln := rxNewNode("cls")
		ln.cls.add(e.ch, e.ch)
		if st.java && st.icase {
			st.foldJ(ln.cls, true)
		}
		return ln
	case ')':
		// A ) only closes a group when one is OPEN. At the top level of a POSIX ERE it
		// is an ordinary character - [[ "a)b" =~ ")" ]] matches in bash - where every
		// other dialect here calls it "unmatched )".
		if !st.ere || st.depth > 0 {
			return rxNewNode("empty")
		}
	case '|':
		return rxNewNode("empty")
	case '*', '+', '?':
		if st.ere {
			st.fail("Invalid preceding regular expression")
		} else {
			st.fail("quantifier with nothing to repeat")
		}
		st.i++
		return rxNewNode("empty")
	case '{':
		// Java reads a { where an atom belongs as a quantifier of NOTHING: {2}a and
		// a{1}{2} are legal, and a bare { is "Illegal repetition".
		if st.java {
			return rxNewNode("empty")
		}
		// A { that opens an interval where no atom precedes it is the same error; a {
		// NOT followed by a digit is an ordinary character (bash matches "a{b").
		if st.ere && st.i+1 < st.n && rxIsDigit(st.p[st.i+1]) {
			st.fail("Invalid preceding regular expression")
			st.i++
			return rxNewNode("empty")
		}
	}
	st.i++
	lit := rxNewNode("cls")
	lit.cls.add(c, c)
	if st.java && st.icase {
		st.foldJ(lit.cls, true)
	}
	return lit
}

// rxLinebreakJ is the node of Java's \R: CR LF, or one character of \v.
func rxLinebreakJ() *rxNode {
	cr := rxNewNode("cls")
	cr.cls.add('\r', '\r')
	lf := rxNewNode("cls")
	lf.cls.add('\n', '\n')
	crlf := rxNewNode("cat")
	crlf.items = []*rxNode{cr, lf}
	one := rxNewNode("cls")
	one.cls.addSp(rxSJV)
	alt := rxNewNode("alt")
	alt.items = []*rxNode{crlf, one}
	g := rxNewNode("group")
	g.items = []*rxNode{alt}
	return g
}

// parseBoundJ reads a java.util.regex bound, where a { is always one: ok=false,
// with Java's error, for anything that does not read as {n}, {n,} or {n,m} with
// n and m ints. Twin of rxParseBoundJ in lib/regex.js.
func (st *rxSt) parseBoundJ() (int, int, bool) {
	p := st.p
	j := st.i + 1
	if j >= st.n || !rxIsDigit(p[j]) {
		st.failAt("Illegal repetition", st.i+1)
		return 0, 0, false
	}
	minV := 0
	for j < st.n && rxIsDigit(p[j]) {
		minV = minV*10 + int(p[j]-'0')
		if minV > 2147483647 {
			st.failAt("Illegal repetition range", j)
			return 0, 0, false
		}
		j = st.skipXAt(j + 1)
	}
	maxV := minV
	if j < st.n && p[j] == ',' {
		j = st.skipXAt(j + 1)
		maxV = -1
		if j < st.n && rxIsDigit(p[j]) {
			maxV = 0
		}
		for j < st.n && rxIsDigit(p[j]) {
			maxV = maxV*10 + int(p[j]-'0')
			if maxV > 2147483647 {
				st.failAt("Illegal repetition range", j)
				return 0, 0, false
			}
			j = st.skipXAt(j + 1)
		}
	}
	if j >= st.n || p[j] != '}' {
		st.failAt("Unclosed counted closure", j)
		return 0, 0, false
	}
	if maxV >= 0 && maxV < minV {
		st.failAt("Illegal repetition range", j)
		return 0, 0, false
	}
	st.i = j + 1
	return minV, maxV, true
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
	// Python's IGNORECASE of a str pattern folds case by Unicode (foldP), by
	// ASCII under re.ASCII / (?a). Twin of rxParsePiece in lib/regex.js.
	if st.py && st.icase {
		if atom.t == "cls" {
			atom.cls.fold()
			if st.pyuni {
				atom.cls.foldP()
			}
		}
		if atom.t == "bref" {
			atom.min = 1
			if st.pyuni {
				atom.min = 2
			}
		}
	}
	if st.java && st.icase && atom.t == "bref" {
		atom.min = 1
		if st.ucase {
			atom.min = 2
		}
	}
	st.skipX()
	seen := 0
	for st.i < st.n {
		c := st.p[st.i]
		var minV, maxV int
		counted := false
		switch {
		case c == '{' && st.java:
			// A second {..} starts a new piece over an empty atom (parseAtom).
			if seen > 0 {
				return atom
			}
			lo, hi, ok := st.parseBoundJ()
			if !ok {
				return atom
			}
			minV, maxV = lo, hi
			counted = true
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
			// In a POSIX ERE a { opens an interval EXACTLY WHEN a digit follows, and an
			// interval that then fails to close is an error: bash matches "a{b" and "a{"
			// literally but rejects "a{1" outright. Everywhere else an unreadable brace
			// stays a literal brace.
			ivl := st.ere && st.i+1 < st.n && rxIsDigit(st.p[st.i+1])
			lo, hi, ok := st.parseBound()
			if !ok {
				if ivl {
					st.fail("Invalid content of \\{\\}")
				}
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
		// A second quantifier on the same atom - a**, a?*, a{2}{3}, and in an ERE also
		// a*? - is a syntax error in bash, node and python3, and a** is one in java too
		// (java reads the {3} of a{2}{3} as a new piece, see above). Ruby 2.6 is the one
		// dissenter: it warns about the "redundant nested repeat operator" and folds it.
		seen++
		if seen > 1 {
			if st.ere {
				st.fail("Invalid preceding regular expression")
			} else {
				st.fail("multiple repeat")
			}
			return atom
		}
		rep := rxNewNode("rep")
		rep.min = minV
		rep.max = maxV
		rep.greedy = true
		rep.cnt = counted && !st.re2
		// A POSIX ERE has no lazy or possessive form, so the ? of "a*?" is a SECOND
		// quantifier there rather than a modifier of the first.
		if st.java {
			st.skipX()
		}
		poss := false
		if st.i < st.n && !st.ere {
			q := st.p[st.i]
			if q == '?' {
				rep.greedy = false
				st.i++
			} else if q == '+' {
				st.i++
				poss = st.java
			}
		}
		// Outside J a possessive quantifier is read as greedy; under J it is
		// (?>(?>X)*), as in lib/regex.js.
		if poss && atom.t != "cls" {
			inner := rxNewNode("atom")
			inner.items = append(inner.items, atom)
			atom = inner
		}
		rep.items = append(rep.items, atom)
		atom = rep
		if poss {
			outer := rxNewNode("atom")
			outer.items = append(outer.items, rep)
			atom = outer
		}
		st.skipX()
	}
	return atom
}

func (st *rxSt) parseCat() *rxNode {
	cat := rxNewNode("cat")
	st.skipX()
	for st.i < st.n {
		c := st.p[st.i]
		if c == '|' {
			break
		}
		if c == ')' && (!st.ere || st.depth > 0) {
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
	// A POSIX ERE branch may not be empty: bash rejects "", "a|", "|a", "(a|)" and
	// "(|a)" alike with "empty (sub)expression". The one exception is a group whose
	// WHOLE body is empty - "()" and "(())" match the empty string in bash 5.3 - so the
	// emptiness is only an error for a branch of a real alternation, or for the top
	// level of the pattern.
	if st.ere && st.err == "" {
		for _, br := range alt.items {
			if len(br.items) == 0 && (len(alt.items) > 1 || st.depth == 0) {
				st.fail("empty (sub)expression")
				break
			}
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
	jsrep  bool
	java   bool
	re2    bool
	dummy  *rxCls
	// cmin/cmax/cgreedy describe counted loop i (rxOpCInit .. rxOpCInc), lbmin/lbmax
	// the length range of lookbehind i (-1: unbounded).
	cmin, cmax, cgreedy []int
	lbmin, lbmax        []int
}

// loG / hiG are the smallest and largest group NUMBER declared inside a subtree, or
// -1 for none. Only a "group" node carries a group number in gidx: "assert" keeps the
// assertion kind there, "bref" the group it points at and "look" the negation flag.
func (n *rxNode) loG() int {
	best := -1
	if n.t == "group" && n.gidx >= 0 {
		best = n.gidx
	}
	for _, k := range n.items {
		if g := k.loG(); g >= 0 && (best < 0 || g < best) {
			best = g
		}
	}
	return best
}

func (n *rxNode) hiG() int {
	best := -1
	if n.t == "group" && n.gidx >= 0 {
		best = n.gidx
	}
	for _, k := range n.items {
		if g := k.hiG(); g > best {
			best = g
		}
	}
	return best
}

// emitClear is the j dialect's per-iteration capture reset: every iteration of a
// repeat starts by clearing the captures declared inside the repeated atom, so a
// group that took part in an earlier iteration but not the last one reads as "did
// not take part" - /(?:(a)|b)+/.exec("ab") leaves group 1 undefined in node, and
// bash reports the same group empty. Perl, Python, Ruby and Java carry the earlier
// value forward, so nothing is emitted for them. The group numbers inside a subtree
// are contiguous ('(' is counted left to right), so one lo..hi range covers them.
func (pg *rxProg) emitClear(body *rxNode) {
	if !pg.jsrep {
		return
	}
	lo := body.loG()
	if lo < 0 {
		return
	}
	pg.add(rxOpClear, lo, body.hiG(), pg.dummy)
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
		if node.cls.x {
			pg.add(rxOpClassX, 0, 0, node.cls)
		} else {
			pg.add(rxOpClass, 0, 0, node.cls)
		}
	case "assert":
		pg.add(rxOpAssert, node.gidx, 0, pg.dummy)
	case "bref":
		pg.add(rxOpBref, node.gidx, node.min, pg.dummy)
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
	case "op":
		pg.add(node.gidx, 0, 0, pg.dummy)
	case "atom":
		am := pg.add(rxOpAtom, 0, 0, pg.dummy)
		pg.emit(node.items[0])
		pg.add(rxOpAtomEnd, 0, 0, pg.dummy)
		pg.as[am] = pg.here()
	case "lookb":
		id := len(pg.lbmin)
		pg.lbmin = append(pg.lbmin, node.items[0].minLen())
		// Lengths count characters; a J character is up to two UTF-16 units.
		lbm := node.items[0].maxLen()
		if pg.java && lbm > 0 {
			lbm = rxLenCap(2 * lbm)
		}
		pg.lbmax = append(pg.lbmax, lbm)
		neg := 0
		if node.gidx == 1 {
			neg = 1
		}
		lb := pg.add(rxOpLookB, 0, 2*id+neg, pg.dummy)
		pg.emit(node.items[0])
		pg.add(rxOpLBEnd, 0, 0, pg.dummy)
		pg.as[lb] = pg.here()
	case "rep":
		if node.cnt {
			pg.emitCount(node)
		} else {
			pg.emitRep(node)
		}
	}
}

// minLen / maxLen are the fewest / most characters a node can match, -1 being
// unbounded; twins of rxMinLen / rxMaxLen in lib/regex.js.
func (n *rxNode) minLen() int {
	switch n.t {
	case "cls":
		return 1
	case "op":
		if n.gidx == rxOpXGraph {
			return 1
		}
	case "cat":
		sum := 0
		for _, it := range n.items {
			sum = rxLenCap(sum + it.minLen())
		}
		return sum
	case "alt":
		best := -1
		for _, it := range n.items {
			if m := it.minLen(); best < 0 || m < best {
				best = m
			}
		}
		if best < 0 {
			return 0
		}
		return best
	case "group", "atom":
		if len(n.items) > 0 {
			return n.items[0].minLen()
		}
	case "rep":
		return rxLenCap(n.min * n.items[0].minLen())
	}
	return 0
}

func (n *rxNode) maxLen() int {
	switch n.t {
	case "cls":
		return 1
	case "bref":
		return -1
	case "cat", "alt":
		acc := 0
		for _, it := range n.items {
			m := it.maxLen()
			if m < 0 {
				return -1
			}
			if n.t == "cat" {
				acc = rxLenCap(acc + m)
			} else if m > acc {
				acc = m
			}
		}
		return acc
	case "group", "atom":
		if len(n.items) > 0 {
			return n.items[0].maxLen()
		}
	case "rep":
		b := n.items[0].maxLen()
		if b == 0 {
			return 0
		}
		if b < 0 || n.max < 0 {
			return -1
		}
		return rxLenCap(n.max * b)
	}
	return 0
}

// lookbVarP: does a look-behind under n match a varying number of characters?
// Twin of rxLookbVarP in lib/regex.js.
func (st *rxSt) lookbVarP(n *rxNode) bool {
	if n.t == "lookb" && st.widthP(n.items[0], 0) < 0 {
		return true
	}
	for _, it := range n.items {
		if st.lookbVarP(it) {
			return true
		}
	}
	return false
}

// widthP is the one number of characters a node always matches under P, or -1
// when it can vary: Python's look-behind must be fixed-width. A backreference is
// as wide as the group it names (st.gnodes); depth stops a group that refers to
// itself. Twin of rxWidthP in lib/regex.js.
func (st *rxSt) widthP(n *rxNode, depth int) int {
	switch n.t {
	case "cls":
		return 1
	case "cat", "alt":
		acc := 0
		for i, it := range n.items {
			w := st.widthP(it, depth)
			if w < 0 {
				return -1
			}
			if n.t == "cat" {
				acc = rxLenCap(acc + w)
			} else if i > 0 && w != acc {
				return -1
			} else {
				acc = w
			}
		}
		return acc
	case "group":
		return st.widthP(n.items[0], depth)
	case "rep":
		b := st.widthP(n.items[0], depth)
		if b == 0 {
			return 0
		}
		if b < 0 || n.min != n.max {
			return -1
		}
		return rxLenCap(n.min * b)
	case "bref":
		g := n.gidx
		if n.bname != "" {
			g = st.nameIdx(n.bname)
		}
		if g < 1 || g >= len(st.gnodes) || depth > st.ngroups {
			return -1
		}
		return st.widthP(st.gnodes[g], depth+1)
	}
	return 0
}

// rxLenCap saturates a length at the int range, which no text reaches.
func rxLenCap(n int) int {
	if n > 2147483647 {
		return 2147483647
	}
	return n
}

// emitCount emits a J {n,m} repeat as one copy of the body inside a counted
// loop; twin of rxEmitCount in lib/regex.js.
func (pg *rxProg) emitCount(node *rxNode) {
	c := len(pg.cmin)
	pg.cmin = append(pg.cmin, node.min)
	pg.cmax = append(pg.cmax, node.max)
	g := 0
	if node.greedy {
		g = 1
	}
	pg.cgreedy = append(pg.cgreedy, g)
	pg.add(rxOpCInit, c, 0, pg.dummy)
	top := pg.add(rxOpCLoop, c, 0, pg.dummy)
	pg.emit(node.items[0])
	pg.add(rxOpCInc, c, top, pg.dummy)
	pg.bs[top] = pg.here()
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
	if pg.re2 && node.max < 0 {
		pg.emitRepRe2(node)
		return
	}
	body := node.items[0]
	for i := 0; i < node.min; i++ {
		pg.emitClear(body)
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
		pg.emitClear(body)
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
		pg.emitClear(body)
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

// ===== The RE2 dialect (flag R): Go's regexp/syntax =====
//
// rxRe2 reads a pattern the way Go's regexp.Compile does and answers its J
// spelling or Go's error text; twin of rxRe2 in lib/regex.js, which states the
// rewrite. The pattern here is code points (wtf8Runes), so a character is one
// index wide where the JS side steps over a surrogate pair.

// Go's unicode tables for \p{..}; twins of RX_RE2_CATS, RX_RE2_CALIAS,
// RX_RE2_SCRIPTS and RX_RE2_NEWSC in lib/regex.js.
const rxRe2Cats = " C Cc Cf Cn Co Cs L Ll Lm Lo Lt Lu M Mc Me Mn N Nd Nl No P Pc Pd Pe Pf Pi Po Ps S Sc Sk Sm So Z Zl Zp Zs "
const rxRe2CAlias = " Cased_Letter=LC Close_Punctuation=Pe Combining_Mark=M Connector_Punctuation=Pc Control=Cc" +
	" Currency_Symbol=Sc Dash_Punctuation=Pd Decimal_Number=Nd Enclosing_Mark=Me Final_Punctuation=Pf" +
	" Format=Cf Initial_Punctuation=Pi Letter=L Letter_Number=Nl Line_Separator=Zl Lowercase_Letter=Ll" +
	" Mark=M Math_Symbol=Sm Modifier_Letter=Lm Modifier_Symbol=Sk Nonspacing_Mark=Mn Number=N" +
	" Open_Punctuation=Ps Other=C Other_Letter=Lo Other_Number=No Other_Punctuation=Po Other_Symbol=So" +
	" Paragraph_Separator=Zp Private_Use=Co Punctuation=P Separator=Z Space_Separator=Zs Spacing_Mark=Mc" +
	" Surrogate=Cs Symbol=S Titlecase_Letter=Lt Unassigned=Cn Uppercase_Letter=Lu cntrl=Cc digit=Nd punct=P "
const rxRe2Scripts = " Adlam Ahom Anatolian_Hieroglyphs Arabic Armenian Avestan Balinese Bamum Bassa_Vah Batak" +
	" Bengali Beria_Erfe Bhaiksuki Bopomofo Brahmi Braille Buginese Buhid Canadian_Aboriginal Carian" +
	" Caucasian_Albanian Chakma Cham Cherokee Chorasmian Common Coptic Cuneiform Cypriot Cypro_Minoan" +
	" Cyrillic Deseret Devanagari Dives_Akuru Dogra Duployan Egyptian_Hieroglyphs Elbasan Elymaic" +
	" Ethiopic Garay Georgian Glagolitic Gothic Grantha Greek Gujarati Gunjala_Gondi Gurmukhi" +
	" Gurung_Khema Han Hangul Hanifi_Rohingya Hanunoo Hatran Hebrew Hiragana Imperial_Aramaic Inherited" +
	" Inscriptional_Pahlavi Inscriptional_Parthian Javanese Kaithi Kannada Katakana Kawi Kayah_Li" +
	" Kharoshthi Khitan_Small_Script Khmer Khojki Khudawadi Kirat_Rai Lao Latin Lepcha Limbu Linear_A" +
	" Linear_B Lisu Lycian Lydian Mahajani Makasar Malayalam Mandaic Manichaean Marchen Masaram_Gondi" +
	" Medefaidrin Meetei_Mayek Mende_Kikakui Meroitic_Cursive Meroitic_Hieroglyphs Miao Modi Mongolian" +
	" Mro Multani Myanmar Nabataean Nag_Mundari Nandinagari New_Tai_Lue Newa Nko Nushu" +
	" Nyiakeng_Puachue_Hmong Ogham Ol_Chiki Ol_Onal Old_Hungarian Old_Italic Old_North_Arabian" +
	" Old_Permic Old_Persian Old_Sogdian Old_South_Arabian Old_Turkic Old_Uyghur Oriya Osage Osmanya" +
	" Pahawh_Hmong Palmyrene Pau_Cin_Hau Phags_Pa Phoenician Psalter_Pahlavi Rejang Runic Samaritan" +
	" Saurashtra Sharada Shavian Siddham Sidetic SignWriting Sinhala Sogdian Sora_Sompeng Soyombo" +
	" Sundanese Sunuwar Syloti_Nagri Syriac Tagalog Tagbanwa Tai_Le Tai_Tham Tai_Viet Tai_Yo Takri" +
	" Tamil Tangsa Tangut Telugu Thaana Thai Tibetan Tifinagh Tirhuta Todhri Tolong_Siki Toto" +
	" Tulu_Tigalari Ugaritic Vai Vithkuqi Wancho Warang_Citi Yezidi Yi Zanabazar_Square "
const rxRe2NewSc = " Beria_Erfe Sidetic Tai_Yo Tolong_Siki "

// rxRe2Spec: twin of rxRe2Spec in lib/regex.js.
func rxRe2Spec(name string) string {
	switch name {
	case "d", "digit":
		return "09"
	case "s":
		return "\t\n\f\r  "
	case "w", "word":
		return "09AZ__az"
	case "alnum":
		return "09AZaz"
	case "alpha":
		return "AZaz"
	case "ascii":
		return "\x00\x7f"
	case "blank":
		return "\t\t  "
	case "cntrl":
		return "\x00\x1f\x7f\x7f"
	case "graph":
		return "!~"
	case "lower":
		return "az"
	case "print":
		return " ~"
	case "punct":
		return "!/:@[`{~"
	case "space":
		return "\t\r  "
	case "upper":
		return "AZ"
	case "xdigit":
		return "09AFaf"
	}
	return ""
}

type rxRe2Frame struct {
	alts                     string
	nalts                    int
	cur                      []string
	needs                    []int
	need                     int
	cap                      int
	fold, multi, dotnl, lazy bool
}

type rxRe2St struct {
	p                        []rune
	i, k, n                  int
	err                      string
	fold, multi, dotnl, lazy bool
	frames                   []*rxRe2Frame
	depth                    int
	ncap                     int
	names                    []string
	nameGroups               []int
	lastRep                  int
}

type rxRe2Res struct {
	out, err   string
	names      []string
	nameGroups []int
}

func (t *rxRe2St) str(from, to int) string { return rxUnitsStr(t.p[from:to]) }

func (t *rxRe2St) fail(code, expr string) {
	if t.err == "" {
		t.err = "error parsing regexp: " + code + ": `" + expr + "`"
	}
}

func (t *rxRe2St) open(cp int) {
	t.frames = append(t.frames[:t.depth+1], &rxRe2Frame{cap: cp, fold: t.fold, multi: t.multi, dotnl: t.dotnl, lazy: t.lazy})
	t.depth++
}

func (t *rxRe2St) push(text string, need int) {
	f := t.frames[t.depth]
	f.cur = append(f.cur, text)
	f.needs = append(f.needs, need)
}

func (f *rxRe2Frame) body() string {
	s := strings.Join(f.cur, "")
	if f.nalts == 0 {
		return s
	}
	return f.alts + "|" + s
}

func (f *rxRe2Frame) needOf() int {
	n := f.need
	for _, v := range f.needs {
		if v > n {
			n = v
		}
	}
	return n
}

func (t *rxRe2St) bar() {
	f := t.frames[t.depth]
	f.need = f.needOf()
	f.alts = f.body()
	f.nalts++
	f.cur, f.needs = nil, nil
}

func (t *rxRe2St) close() {
	if t.depth == 0 {
		t.fail("unexpected )", t.str(0, t.n))
		return
	}
	f := t.frames[t.depth]
	t.depth--
	t.fold, t.multi, t.dotnl, t.lazy = f.fold, f.multi, f.dotnl, f.lazy
	if f.cap > 0 {
		t.push("("+f.body()+")", f.needOf())
	} else {
		t.push("(?:"+f.body()+")", f.needOf())
	}
	t.i++
}

func rxRe2Hex(c rune) string { return "\\x{" + strings.ToUpper(strconv.FormatInt(int64(c), 16)) + "}" }

func (t *rxRe2St) lit(c rune) {
	s := rxRe2Hex(c)
	if rxIsWord(c) && c != '_' {
		s = string(c)
	}
	if t.fold {
		s = "(?iu:" + s + ")"
	}
	t.push(s, 0)
}

type rxRe2Cls struct {
	lo, hi []rune
	props  string
}

func (cls *rxRe2Cls) rng(a, b rune) {
	cls.lo = append(cls.lo, a)
	cls.hi = append(cls.hi, b)
}

func (cls *rxRe2Cls) group(spec string, neg bool) {
	from := rune(0)
	for k := 0; k < len(spec); k += 2 {
		a, b := rune(spec[k]), rune(spec[k+1])
		if !neg {
			cls.rng(a, b)
		} else {
			if a > from {
				cls.rng(from, a-1)
			}
			from = b + 1
		}
	}
	if neg {
		cls.rng(from, 0x10FFFF)
	}
}

func (cls *rxRe2Cls) text(neg bool) string {
	s := "["
	if neg {
		s = "[^"
	}
	for i := range cls.lo {
		s += rxRe2Hex(cls.lo[i])
		if cls.hi[i] != cls.lo[i] {
			s += "-" + rxRe2Hex(cls.hi[i])
		}
	}
	return s + cls.props + "]"
}

func (t *rxRe2St) pushCls(cls *rxRe2Cls, neg bool) {
	s := cls.text(neg)
	if t.fold {
		s = "(?iu:" + s + ")"
	}
	t.push(s, 0)
}

// rxRe2Canon is regexp/syntax's canonicalName.
func rxRe2Canon(name string) string {
	var out []byte
	first := true
	for i := 0; i < len(name); i++ {
		c := name[i]
		if c == '_' || c == '-' || c == ' ' {
			continue
		}
		if first {
			if 'a' <= c && c <= 'z' {
				c -= 32
			}
			first = false
		} else if 'A' <= c && c <= 'Z' {
			c += 32
		}
		out = append(out, c)
	}
	return string(out)
}

func rxRe2Word(table, canon, sep string) string {
	for _, w := range strings.Fields(table) {
		key := w
		if sep != "" {
			if k := strings.Index(w, sep); k >= 0 {
				key = w[:k]
			}
		}
		if rxRe2Canon(key) == canon {
			return w
		}
	}
	return ""
}

func rxRe2Prop(name string, sign int) string {
	p, np := "\\p{", "\\P{"
	if sign < 0 {
		p, np = np, p
	}
	c := rxRe2Canon(name)
	switch c {
	case "Any":
		return p + "all}"
	case "Assigned":
		return np + "Cn}"
	case "Ascii":
		return p + "ASCII}"
	case "Lc":
		return p + "LC}"
	}
	cat := rxRe2Word(rxRe2Cats, c, "")
	if cat == "" {
		if al := rxRe2Word(rxRe2CAlias, c, "="); al != "" {
			cat = al[strings.Index(al, "=")+1:]
		}
	}
	if cat != "" {
		return p + cat + "}"
	}
	sc := rxRe2Word(rxRe2Scripts, c, "")
	if sc == "" {
		return ""
	}
	if strings.Contains(rxRe2NewSc, " "+sc+" ") {
		return np + "all}"
	}
	return p + "Is" + strings.ToUpper(sc) + "}"
}

func (t *rxRe2St) uni(at int, cls *rxRe2Cls) int {
	p := t.p
	sign := 1
	if p[at+1] == 'P' {
		sign = -1
	}
	j := at + 2
	seq, name := "", ""
	if j >= t.n {
		seq = t.str(at, j)
	} else if p[j] != '{' {
		j++
		seq, name = t.str(at, j), t.str(at+2, j)
	} else {
		end := -1
		for e := at; e < t.n; e++ {
			if p[e] == '}' {
				end = e
				break
			}
		}
		if end < 0 {
			t.fail("invalid character class range", t.str(at, t.n))
			return j
		}
		seq, name = t.str(at, end+1), t.str(at+3, end)
		j = end + 1
	}
	if name != "" && name[0] == '^' {
		sign = -sign
		name = name[1:]
	}
	item := rxRe2Prop(name, sign)
	if item == "" {
		t.fail("invalid character class range", seq)
		return j
	}
	cls.props += item
	return j
}

func (cls *rxRe2Cls) perl(c rune) bool {
	lc := c
	if 'A' <= c && c <= 'Z' {
		lc = c + 32
	}
	if lc != 'd' && lc != 's' && lc != 'w' {
		return false
	}
	cls.group(rxRe2Spec(string(lc)), lc != c)
	return true
}

func (t *rxRe2St) esc(at int) rune {
	p := t.p
	j := at + 1
	if j >= t.n {
		t.fail("trailing backslash at end of expression", "")
		return -1
	}
	c := p[j]
	j++
	t.k = j
	if c < 128 && !(rxIsWord(c) && c != '_') {
		return c
	}
	octal := c == '0'
	if c >= '1' && c <= '7' {
		octal = j < t.n && p[j] >= '0' && p[j] <= '7'
	}
	if octal {
		r := c - '0'
		for n := 1; n < 3 && j < t.n && p[j] >= '0' && p[j] <= '7'; n++ {
			r = r*8 + p[j] - '0'
			j++
		}
		t.k = j
		return r
	}
	if c == 'x' && j < t.n {
		x := p[j]
		j++
		if x == '{' {
			nh, v := 0, rune(0)
			for j < t.n {
				h := p[j]
				j++
				if h == '}' {
					if nh == 0 {
						break
					}
					t.k = j
					return v
				}
				hv := rune(rxHexVal(h))
				if !rxIsHex(h) {
					break
				}
				v = v*16 + hv
				if v > 0x10FFFF {
					break
				}
				nh++
			}
		} else {
			y := rune(-1)
			if j < t.n {
				yc := p[j]
				j++
				if rxIsHex(yc) {
					y = rune(rxHexVal(yc))
				}
			}
			if rxIsHex(x) && y >= 0 {
				t.k = j
				return rune(rxHexVal(x))*16 + y
			}
		}
	}
	switch c {
	case 'a':
		t.k = j
		return '\a'
	case 'f':
		t.k = j
		return '\f'
	case 'n':
		t.k = j
		return '\n'
	case 'r':
		t.k = j
		return '\r'
	case 't':
		t.k = j
		return '\t'
	case 'v':
		t.k = j
		return '\v'
	}
	t.fail("invalid escape sequence", t.str(at, j))
	return -1
}

func (t *rxRe2St) classChar(at, whole int) rune {
	if at >= t.n {
		t.fail("missing closing ]", t.str(whole, t.n))
		return -1
	}
	if t.p[at] == '\\' {
		return t.esc(at)
	}
	t.k = at + 1
	return t.p[at]
}

func (t *rxRe2St) class() {
	p := t.p
	s := t.i
	j := s + 1
	neg := false
	if j < t.n && p[j] == '^' {
		neg = true
		j++
	}
	cls := &rxRe2Cls{}
	first := true
	for t.err == "" && (j >= t.n || p[j] != ']' || first) {
		first = false
		if t.n-j > 2 && p[j] == '[' && p[j+1] == ':' {
			e := -1
			for k := j + 2; k+1 < t.n; k++ {
				if p[k] == ':' && p[k+1] == ']' {
					e = k
					break
				}
			}
			if e >= 0 {
				name := t.str(j, e+2)
				pn := name[2 : len(name)-2]
				pneg := pn != "" && pn[0] == '^'
				if pneg {
					pn = pn[1:]
				}
				spec := ""
				if len(pn) > 1 {
					spec = rxRe2Spec(pn)
				}
				if spec == "" {
					t.fail("invalid character class range", name)
					return
				}
				cls.group(spec, pneg)
				j = e + 2
				continue
			}
		}
		if j+1 < t.n && p[j] == '\\' {
			d := p[j+1]
			if d == 'p' || d == 'P' {
				j = t.uni(j, cls)
				continue
			}
			if cls.perl(d) {
				j += 2
				continue
			}
		}
		rs := j
		lo := t.classChar(j, s)
		if lo < 0 {
			return
		}
		j = t.k
		hi := lo
		if t.n-j >= 2 && p[j] == '-' && p[j+1] != ']' {
			hi = t.classChar(j+1, s)
			if hi < 0 {
				return
			}
			j = t.k
			if hi < lo {
				t.fail("invalid character class range", t.str(rs, j))
				return
			}
		}
		cls.rng(lo, hi)
	}
	if t.err != "" {
		return
	}
	t.i = j + 1
	t.pushCls(cls, neg)
}

// bound is regexp/syntax's parseRepeat; ok is false when '{' is a literal.
func (t *rxRe2St) bound() (lo, hi, end int, ok bool) {
	p := t.p
	j := t.i + 1
	lo = t.num(j)
	if lo == -2 {
		return
	}
	j = t.k
	hi = lo
	if j >= t.n {
		return
	}
	if p[j] == ',' {
		j++
		if j >= t.n {
			return
		}
		if p[j] == '}' {
			hi = -1
		} else {
			hi = t.num(j)
			if hi == -2 {
				return
			}
			j = t.k
			if hi == -1 {
				lo = -1
			}
		}
	}
	if j >= t.n || p[j] != '}' {
		return
	}
	return lo, hi, j + 1, true
}

func (t *rxRe2St) num(at int) int {
	p := t.p
	if at >= t.n || !rxIsDigit(p[at]) {
		return -2
	}
	if at+1 < t.n && p[at] == '0' && rxIsDigit(p[at+1]) {
		return -2
	}
	j, v := at, 0
	for j < t.n && rxIsDigit(p[j]) {
		if v >= 0 {
			if v >= 100000000 {
				v = -1
			} else {
				v = v*10 + int(p[j]-'0')
			}
		}
		j++
	}
	t.k = j
	return v
}

func (t *rxRe2St) repeat(op, min, max int, counted bool) {
	p := t.p
	q := t.str(op, t.i)
	lazy := false
	if t.i < t.n && p[t.i] == '?' {
		lazy = true
		t.i++
	}
	if t.lastRep >= 0 {
		t.fail("invalid nested repetition operator", t.str(t.lastRep, t.i))
		return
	}
	f := t.frames[t.depth]
	if len(f.cur) == 0 {
		t.fail("missing argument to repetition operator", t.str(op, t.i))
		return
	}
	atom := f.cur[len(f.cur)-1]
	need := f.needs[len(f.needs)-1]
	f.cur, f.needs = f.cur[:len(f.cur)-1], f.needs[:len(f.needs)-1]
	if counted {
		m := max
		if m < 0 {
			m = min
		}
		if m == 0 {
			need = 0
		} else if need > 1 {
			need = m * need
		} else {
			need = m
		}
		if (min >= 2 || max >= 2) && need > 1000 {
			t.fail("invalid repeat count", t.str(op, t.i))
			return
		}
	}
	if lazy != t.lazy {
		q += "?"
	}
	t.push("(?:"+atom+")"+q, need)
}

func (t *rxRe2St) flags() {
	p := t.p
	i := t.i
	named := t.n-i > 4 && p[i+2] == 'P' && p[i+3] == '<'
	start := i + 4
	if !named && t.n-i > 3 && p[i+2] == '<' {
		named = true
		start = i + 3
	}
	if named {
		end := -1
		for e := i; e < t.n; e++ {
			if p[e] == '>' {
				end = e
				break
			}
		}
		if end < 0 {
			t.fail("invalid named capture", t.str(i, t.n))
			return
		}
		name := t.str(start, end)
		ok := name != ""
		for _, c := range []rune(name) {
			if !rxIsWord(c) {
				ok = false
			}
		}
		if !ok {
			t.fail("invalid named capture", t.str(i, end+1))
			return
		}
		t.ncap++
		t.names = append(t.names, name)
		t.nameGroups = append(t.nameGroups, t.ncap)
		t.open(t.ncap)
		t.i = end + 1
		return
	}
	j := i + 2
	fold, multi, dotnl, ung := t.fold, t.multi, t.dotnl, t.lazy
	neg, saw := false, false
loop:
	for j < t.n {
		c := p[j]
		j++
		switch c {
		case 'i':
			fold = !neg
		case 'm':
			multi = !neg
		case 's':
			dotnl = !neg
		case 'U':
			ung = !neg
		case '-':
			if neg {
				break loop
			}
			neg, saw = true, false
			continue
		case ':', ')':
			if neg && !saw {
				break loop
			}
			if c == ':' {
				t.open(0)
			}
			t.fold, t.multi, t.dotnl, t.lazy = fold, multi, dotnl, ung
			t.i = j
			return
		default:
			break loop
		}
		saw = true
	}
	t.fail("invalid or unsupported Perl syntax", t.str(i, j))
}

func (t *rxRe2St) escape() {
	p := t.p
	i := t.i
	if i+1 < t.n {
		d := p[i+1]
		as := ""
		switch d {
		case 'A':
			as = "\\A"
		case 'b':
			as = "\\b"
		case 'B':
			as = "\\B"
		case 'z':
			as = "\\z"
		}
		if as != "" {
			t.push(as, 0)
			t.i = i + 2
			return
		}
		if d == 'C' {
			t.fail("invalid escape sequence", "\\C")
			return
		}
		if d == 'Q' {
			stop, next := t.n, t.n
			for e := i + 2; e+1 < t.n; e++ {
				if p[e] == '\\' && p[e+1] == 'E' {
					stop, next = e, e+2
					break
				}
			}
			for j := i + 2; j < stop; j++ {
				t.lit(p[j])
			}
			t.i = next
			return
		}
		cls := &rxRe2Cls{}
		if d == 'p' || d == 'P' {
			t.i = t.uni(i, cls)
			if t.err == "" {
				t.pushCls(cls, false)
			}
			return
		}
		if cls.perl(d) {
			t.i = i + 2
			t.pushCls(cls, false)
			return
		}
	}
	r := t.esc(i)
	if r < 0 {
		return
	}
	t.i = t.k
	t.lit(r)
}

func rxRe2(p []rune) *rxRe2Res {
	t := &rxRe2St{p: p, n: len(p), depth: -1, lastRep: -1}
	t.open(0)
	for t.i < t.n && t.err == "" {
		rep := -1
		switch c := p[t.i]; c {
		case '(':
			if t.i+1 < t.n && p[t.i+1] == '?' {
				t.flags()
			} else {
				t.ncap++
				t.open(t.ncap)
				t.i++
			}
		case '|':
			t.bar()
			t.i++
		case ')':
			t.close()
		case '^', '$':
			anc := "\\A"
			if c == '$' {
				anc = "\\z"
			}
			if t.multi {
				anc = "(?:\\A|(?<=\\x{A}))"
				if c == '$' {
					anc = "(?:\\z|(?=\\x{A}))"
				}
			}
			t.push(anc, 0)
			t.i++
		case '.':
			if t.dotnl {
				t.push("[\\x{0}-\\x{10FFFF}]", 0)
			} else {
				t.push("[^\\x{A}]", 0)
			}
			t.i++
		case '[':
			t.class()
		case '*', '+', '?':
			rep = t.i
			t.i++
			lo, hi := 0, -1
			if c == '+' {
				lo = 1
			}
			if c == '?' {
				hi = 1
			}
			t.repeat(rep, lo, hi, false)
		case '{':
			lo, hi, end, ok := t.bound()
			if !ok {
				t.lit('{')
				t.i++
			} else if lo < 0 || lo > 1000 || hi > 1000 || (hi >= 0 && lo > hi) {
				t.fail("invalid repeat count", t.str(t.i, end))
			} else {
				rep = t.i
				t.i = end
				t.repeat(rep, lo, hi, true)
			}
		case '\\':
			t.escape()
		default:
			if isSurrogate(c) {
				t.fail("invalid UTF-8", t.str(t.i, t.n))
			}
			t.lit(c)
			t.i++
		}
		t.lastRep = rep
	}
	if t.err == "" && t.depth != 0 {
		t.fail("missing closing )", t.str(0, t.n))
	}
	res := &rxRe2Res{err: t.err, names: t.names, nameGroups: t.nameGroups}
	if t.err == "" {
		res.out = t.frames[0].body()
	}
	return res
}

// emitRepRe2 is an unbounded repeat under R, compiled as Go compiles it; twin of
// rxEmitRepRe2 in lib/regex.js, which states the rule.
func (pg *rxProg) emitRepRe2(node *rxNode) {
	body := node.items[0]
	for i := 1; i < node.min; i++ {
		pg.emit(body)
	}
	skip := -1
	if node.min == 0 {
		skip = pg.add(rxOpSplit, 0, 0, pg.dummy)
	}
	m0 := pg.nmarks
	pg.nmarks += 2
	pg.add(rxOpMark, m0, 0, pg.dummy)
	pg.emit(body)
	keep := pg.add(rxOpProg, m0, 0, pg.dummy)
	l1 := pg.here()
	sp := pg.add(rxOpSplit, 0, 0, pg.dummy)
	pg.add(rxOpMark, m0+1, 0, pg.dummy)
	pg.emit(body)
	pg.add(rxOpProg, m0+1, -1, pg.dummy)
	pg.add(rxOpJmp, l1, 0, pg.dummy)
	out := pg.here()
	pg.bs[keep] = out
	pg.splitTo(sp, l1+1, out, node.greedy)
	if skip >= 0 {
		pg.splitTo(skip, skip+1, out, node.greedy)
	}
}

// splitTo points the SPLIT at `at` into a loop body and out of it, the body
// first when greedy.
func (pg *rxProg) splitTo(at, body, out int, greedy bool) {
	if greedy {
		pg.as[at], pg.bs[at] = body, out
	} else {
		pg.as[at], pg.bs[at] = out, body
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
	// jsrep is the j flag: an unbounded-loop iteration that consumed nothing is
	// DISCARDED rather than kept. See the rxOpProg case in run().
	jsrep bool
	// java is the J flag, errAt the index of a J error (-1: none), and the rest
	// the counted-loop and lookbehind tables (rxProg).
	java  bool
	errAt int
	// supp is java.util.regex's hasSupplementary (see rxSuppJ).
	supp bool
	// bfail is the n flag (implied by J and P): a backreference to a group that
	// did not take part fails. gflags are P's leading inline flag letters, or
	// under J the letters in force at the end (Pattern.flags()).
	bfail               bool
	gflags              string
	cmin, cmax, cgreedy []int
	lbmin, lbmax        []int
	src                 string
	flags               string
}

func rxCompile(pattern, flags string) *rxRe {
	icase := strings.ContainsRune(flags, 'i')
	dotall := strings.ContainsRune(flags, 's')
	multi := strings.ContainsRune(flags, 'm')
	ext := strings.ContainsRune(flags, 'x')
	octal := strings.ContainsRune(flags, 'o')
	quoting := strings.ContainsRune(flags, 'q')
	ere := strings.ContainsRune(flags, 'e')
	jsrep := strings.ContainsRune(flags, 'j')
	posix := strings.ContainsRune(flags, 'p')
	java := strings.ContainsRune(flags, 'J')
	py := strings.ContainsRune(flags, 'P')
	bfail := java || py || strings.ContainsRune(flags, 'n')
	// R reads the pattern as Go does and compiles its J spelling (rxRe2).
	re2 := strings.ContainsRune(flags, 'R')
	tr := rxRe2(nil)
	if re2 {
		tr = rxRe2(wtf8Runes(pattern))
		java, bfail = true, true
	}
	ucase := java && strings.ContainsRune(flags, 'u')
	unix := java && strings.ContainsRune(flags, 'd')
	// The POSIX ERE dialect (bash's [[ =~ ]]). What it changes is spelled out at each
	// site; the two whole-engine consequences are here. A "." matches a newline in an
	// ERE, and an ERE has POSIX capture semantics, of which the visible one is that an
	// unbounded loop whose iteration consumed nothing DISCARDS that iteration - so
	// ^(a*)*$ on "aaa" reports "aaa" for group 1, as bash and node do and Perl, Python,
	// Ruby and Java do not.
	if ere {
		dotall = true
		jsrep = true
		posix = true
	}
	// "p" on its own is the POSIX BRACKET EXPRESSIONS and nothing else. Ruby takes
	// [[:alpha:]] from POSIX but keeps \d, \w, \s, lazy quantifiers, (?...) groups and
	// an escaping backslash inside a bracket - so it wants this letter and NOT the whole
	// "e" dialect, which would switch all of those off.
	// \Q...\E is expanded before anything else looks at the text, so neither the group
	// counter nor the parser needs a quoting mode. re.src keeps the ORIGINAL pattern,
	// which is what a language's .source / inspect reports.
	pat := []rune(pattern)
	if java {
		pat = wtf8Runes(pattern)
	}
	if re2 {
		pat = wtf8Runes(tr.out)
	}
	if quoting {
		pat = rxExpandQuotes(pat)
	}
	st := &rxSt{p: pat, ext: ext, dotall: dotall, octal: octal, ere: ere, posix: posix, java: java, errAt: -1,
		py: py, icase: icase, multi: multi, ucase: ucase, unix: unix, gnodes: []*rxNode{nil},
		pyuni: py && !strings.ContainsRune(flags, 'a'), re2: re2}
	// Under P and J case folding and line anchoring are decided per atom while
	// parsing.
	if py || java {
		icase, multi = false, false
	}
	st.n = len(st.p)
	if !ere {
		st.ntotal = rxCountGroups(st.p)
	}
	ast := st.parseAlt()
	// Resolve every \k<name> now that the whole name table is known.
	for _, nd := range st.pending {
		gi := st.nameIdx(nd.bname)
		if gi < 0 {
			st.fail("unknown group name '" + nd.bname + "'")
		}
		nd.gidx = gi
	}
	if st.err == "" && st.i < st.n {
		if st.p[st.i] == ')' {
			st.fail("unmatched ) in pattern")
		} else {
			st.fail("unexpected character in pattern")
		}
	}
	// CPython checks a look-behind's width when it compiles the parsed pattern,
	// so every parse error is reported first.
	if py && st.err == "" && st.lookbVarP(ast) {
		st.fail("look-behind requires fixed-width pattern")
	}
	if tr.err != "" {
		st.err = tr.err
	}
	if st.err != "" {
		return &rxRe{ok: false, err: st.err, errAt: st.errAt, icase: icase, dotall: dotall, multi: multi,
			jsrep: jsrep, java: java, bfail: bfail, src: pattern, flags: flags}
	}
	// Under J gflags is Pattern.flags(): the letters in force at the end.
	gflags := st.gflags
	if java {
		gflags = st.getFlags()
	}
	pg := &rxProg{jsrep: jsrep, java: java, re2: re2, dummy: rxNewCls()}
	pg.add(rxOpSave, 0, 0, pg.dummy)
	pg.emit(ast)
	pg.add(rxOpSave, 1, 0, pg.dummy)
	pg.add(rxOpMatch, 0, 0, pg.dummy)
	names, nameGroups := st.names, st.nameGroups
	if re2 {
		names, nameGroups = tr.names, tr.nameGroups
	}
	return &rxRe{
		ok: true, ops: pg.ops, as: pg.as, bs: pg.bs, clss: pg.clss,
		ngroups: st.ngroups, nmarks: pg.nmarks, names: names, nameGroups: nameGroups,
		icase: icase, dotall: dotall, multi: multi, jsrep: jsrep, java: java, errAt: -1,
		supp: java && (re2 || st.supp || rxSuppJ(ast) || rxHasSurr(pat)), bfail: bfail, gflags: gflags,
		cmin: pg.cmin, cmax: pg.cmax, cgreedy: pg.cgreedy, lbmin: pg.lbmin, lbmax: pg.lbmax,
		src: pattern, flags: flags,
	}
}

// rxSuppJ, rxClsBmpJ and rxHasSurr decide java.util.regex's hasSupplementary;
// twins of the same names in lib/regex.js, which state the rule.
func rxSuppJ(node *rxNode) bool {
	if node.t == "cls" {
		return !rxClsBmpJ(node.cls)
	}
	for _, it := range node.items {
		if rxSuppJ(it) {
			return true
		}
	}
	return false
}

func rxClsBmpJ(c *rxCls) bool {
	if c.neg || c.u || len(c.props) > 0 {
		return false
	}
	for i := range c.lo {
		if c.hi[i] >= surrHigh0 && !(c.lo[i] >= surrEnd && c.hi[i] < 0x10000) {
			return false
		}
	}
	for _, k := range c.sp {
		if k == rxSND || k == rxSNW || k == rxSNS || k == rxSJNH || k == rxSJNV {
			return false
		}
	}
	for _, sub := range c.subs {
		if !rxClsBmpJ(sub) {
			return false
		}
	}
	for _, a := range c.ands {
		if !rxClsBmpJ(a) {
			return false
		}
	}
	return true
}

// rxHasSurr: does the pattern hold a character that is a surrogate in UTF-16?
func rxHasSurr(p []rune) bool {
	for _, c := range p {
		if c >= surrHigh0 && (c < surrEnd || c > 0xFFFF) {
			return true
		}
	}
	return false
}

// rxCpAt is the character at index i of a J text, which holds UTF-16 units:
// a surrogate pair is read as one (w 2). Twin of rxCpAt in lib/regex.js.
func rxCpAt(text []rune, i int) (c rune, w int) {
	c = text[i]
	if isHighSurr(c) && i+1 < len(text) && isLowSurr(text[i+1]) {
		return surrPair(c, text[i+1]), 2
	}
	return c, 1
}

// ===== The matcher =====

type rxRunSt struct {
	steps int
	over  bool
	// mustEnd is -1 for an ordinary match and the required end position for a
	// whole-input match (see rxRe.matchWhole).
	mustEnd int
	// lbt is where the innermost running lookbehind stands; ctr / cbeg are each
	// counted loop's iteration count and the position its iteration began at.
	lbt       int
	ctr, cbeg []int
	// gpos and glast are java.util.regex's Matcher.oldLast (\G) and last
	// (\b{g}); see searchFrom.
	gpos, glast int
	// adv is the start of an attempt that may not match empty (searchAdv), or -1.
	adv int
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
	case rxBOL, rxMBOL:
		if sp == 0 {
			return true
		}
		if !re.multi && kind == rxBOL {
			return false
		}
		return text[sp-1] == '\n'
	case rxEOL, rxMEOL:
		if sp == n {
			return true
		}
		if !re.multi && kind == rxEOL {
			return false
		}
		return text[sp] == '\n'
	}
	if kind == rxPUWordB || kind == rxPUNWordB {
		pb := sp > 0 && rxPyWord(text[sp-1])
		pa := sp < n && rxPyWord(text[sp])
		if kind == rxPUWordB {
			return pb != pa
		}
		return pb == pa
	}
	if kind >= rxJBOL {
		return rxAssertJ(text, kind, sp)
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

// rxIsLineSepJ: is c one of Java's line terminators (unix: \n alone)?
func rxIsLineSepJ(c rune, unix bool) bool {
	if c == '\n' {
		return true
	}
	return !unix && (c == '\r' || c == 0x85 || c == 0x2028 || c == 0x2029)
}

// rxAssertJ decides the rxJ* anchors; twin of rxAssertJ in lib/regex.js.
func rxAssertJ(text []rune, kind, sp int) bool {
	n := len(text)
	switch kind {
	case rxJBOL, rxJUBOL:
		if sp == n {
			return false
		}
		if sp == 0 {
			return true
		}
		pc := text[sp-1]
		if !rxIsLineSepJ(pc, kind == rxJUBOL) {
			return false
		}
		return !(pc == '\r' && kind == rxJBOL && text[sp] == '\n')
	case rxJEOL, rxJUEOL, rxJMEOL, rxJUMEOL:
		unix := kind == rxJUEOL || kind == rxJUMEOL
		multi := kind == rxJMEOL || kind == rxJUMEOL
		if sp == n {
			return true
		}
		c := text[sp]
		if !rxIsLineSepJ(c, unix) {
			return false
		}
		crlf := c == '\n' && sp > 0 && text[sp-1] == '\r'
		switch {
		case multi:
			return !(crlf && !unix)
		case unix:
			return sp == n-1
		case sp == n-2:
			return c == '\r' && text[sp+1] == '\n'
		}
		return sp == n-1 && !crlf
	}
	uw := kind == rxJUWordB || kind == rxJUNWordB
	before := false
	if sp > 0 {
		// Character.codePointBefore: a pair only when the unit before sp is its
		// low half.
		bc := text[sp-1]
		if isLowSurr(bc) && sp > 1 {
			if c, w := rxCpAt(text, sp-2); w == 2 {
				bc = c
			}
		}
		before = rxIsWordJ(text, bc, sp-1, uw)
	}
	after := false
	if sp < n {
		c, _ := rxCpAt(text, sp)
		after = rxIsWordJ(text, c, sp, uw)
	}
	if kind == rxJWordB || kind == rxJUWordB {
		return before != after
	}
	return before == after
}

// rxIsWordJ is Pattern.Bound's side test for the character c at or ending at
// unit i; twin of rxIsWordJ in lib/regex.js.
func rxIsWordJ(text []rune, c rune, i int, uw bool) bool {
	if (uw && rxPropHit(rxPWord, c)) || (!uw && rxIsWord(c)) {
		return true
	}
	// No non-spacing mark lies below U+0300.
	if c < 0x300 || !rxPropHit(rxPMn, c) {
		return false
	}
	for k := i; k >= 0; k-- {
		b, _ := rxCpAt(text, k)
		if rxPropHit(rxPLD, b) {
			return true
		}
		if b < 0x300 || !rxPropHit(rxPMn, b) {
			return false
		}
	}
	return false
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
			if sp >= n {
				return -1
			}
			c, w := text[sp], 1
			if re.java && isHighSurr(c) {
				c, w = rxCpAt(text, sp)
			}
			if !re.clss[pc].match(c, re.icase) {
				return -1
			}
			pc++
			sp += w
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
		case rxOpClear:
			// Same shape as Save: write, recurse, and put back what was there when
			// the run fails, so a backtrack out of the iteration restores them.
			from, to := 2*re.as[pc], 2*re.bs[pc]+1
			saved := make([]int, to-from+1)
			copy(saved, caps[from:to+1])
			for i := from; i <= to; i++ {
				caps[i] = -1
			}
			if r := re.run(text, pc+1, sp, caps, marks, st); r >= 0 {
				return r
			}
			copy(caps[from:to+1], saved)
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
				// The iteration consumed nothing. Two readings, and they differ ONLY in
				// what the captures inside the body are left holding:
				//   jsrep off - leave the loop from here, KEEPING whatever that empty
				//     iteration saved. /^(a*)*$/ on "aaa" then reports "" for group 1,
				//     which is what Perl, Python, Ruby and Java print.
				//   jsrep on  - FAIL the iteration, so the run unwinds through the
				//     enclosing MARK/SAVE frames (which restore what they wrote) and the
				//     SPLIT leaves the loop as if the body had never been entered. Group
				//     1 keeps "aaa", which is what node and bash print.
				// Failing here always terminates: the SPLIT above has exactly one
				// alternative left, and it is the loop exit.
				// Under R a PROG whose b is -1 always fails (emitRepRe2).
				if re.jsrep || re.bs[pc] < 0 {
					return -1
				}
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
			// a group that did not take part. Without
			// the length test this panics with an index-out-of-range, and the JS
			// twin reads undefined and lets sp go NaN.
			from, to := -1, -1
			if 2*g+1 < len(caps) {
				from, to = caps[2*g], caps[2*g+1]
			}
			if from < 0 || to < 0 {
				// A group that did not take part matches the empty string - except
				// under n (Java, Python, Ruby), where the backreference fails.
				if re.bfail {
					return -1
				}
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
					if !re.icase && re.bs[pc] == 0 {
						return -1
					}
					if re.bs[pc] == 2 {
						// Java's CIBackRef under UNICODE_CASE.
						lu, ru := unicode.ToUpper(lc), unicode.ToUpper(rc)
						if lu != ru && unicode.ToLower(lu) != unicode.ToLower(ru) {
							return -1
						}
					} else if rxSwapCase(rc) != lc {
						return -1
					}
				}
			}
			pc++
			sp += length
		case rxOpMatch:
			// matchWhole CONSTRAINS THE RUN rather than rewriting the pattern: a
			// match that stops short of mustEnd is refused right here, so the VM
			// keeps backtracking for a longer one. Rewriting the pattern to
			// \A(?:...)\z would work too, but every caller that tried it had to
			// rediscover that under the x flag a trailing # comment swallows the
			// wrapper's own closing paren. Constraining the run has no such edge.
			if st.mustEnd >= 0 && sp != st.mustEnd {
				return -1
			}
			// adv is the start of an attempt that may not match empty (searchAdv).
			if sp == st.adv {
				return -1
			}
			return sp
		case rxOpClassX:
			if sp >= n {
				return -1
			}
			c, w := rxCpAt(text, sp)
			if !re.clss[pc].matchX(c, re.icase) {
				return -1
			}
			pc++
			sp += w
		case rxOpCInit:
			// Like Mark: write, run on, and put the old count back on failure, so a
			// loop re-entered by an enclosing one starts from zero.
			slot := re.as[pc]
			old := st.ctr[slot]
			st.ctr[slot] = 0
			if r := re.run(text, pc+1, sp, caps, marks, st); r >= 0 {
				return r
			}
			st.ctr[slot] = old
			return -1
		case rxOpCLoop:
			slot := re.as[pc]
			cnt := st.ctr[slot]
			switch {
			case re.cmax[slot] >= 0 && cnt >= re.cmax[slot]:
				pc = re.bs[pc]
			case cnt < re.cmin[slot]:
				return re.iterJ(text, pc, sp, caps, marks, st)
			case re.cgreedy[slot] == 1:
				if r := re.iterJ(text, pc, sp, caps, marks, st); r >= 0 {
					return r
				}
				if st.over {
					return -1
				}
				pc = re.bs[pc]
			default:
				if r := re.run(text, re.bs[pc], sp, caps, marks, st); r >= 0 {
					return r
				}
				if st.over {
					return -1
				}
				return re.iterJ(text, pc, sp, caps, marks, st)
			}
		case rxOpCInc:
			slot := re.as[pc]
			if st.cbeg[slot] == sp {
				pc++
				break
			}
			old := st.ctr[slot]
			st.ctr[slot] = old + 1
			if r := re.run(text, re.bs[pc], sp, caps, marks, st); r >= 0 {
				return r
			}
			st.ctr[slot] = old
			return -1
		case rxOpLookB:
			// Try the body from every start the length range allows, shortest first,
			// each run required to end exactly here (rxOpLBEnd).
			id, neg := re.bs[pc]/2, re.bs[pc]%2 == 1
			from := sp - re.lbmin[id]
			to := 0
			if re.lbmax[id] >= 0 {
				to = sp - re.lbmax[id]
			}
			if to < 0 {
				to = 0
			}
			saved := st.lbt
			st.lbt = sp
			hit := false
			for ; !hit && from >= to; from-- {
				hit = re.run(text, pc+1, from, caps, marks, st) >= 0
				if st.over {
					return -1
				}
			}
			st.lbt = saved
			if hit == neg {
				return -1
			}
			pc = re.as[pc]
		case rxOpLBEnd:
			if sp != st.lbt {
				return -1
			}
			return sp
		case rxOpAtom:
			// The body's first success, and no other (see RX_ATOM in lib/regex.js).
			ra := re.run(text, pc+1, sp, caps, marks, st)
			if ra < 0 {
				return -1
			}
			pc = re.as[pc]
			sp = ra
		case rxOpAtomEnd:
			return sp
		case rxOpGPos:
			if sp != st.gpos {
				return -1
			}
			pc++
		case rxOpXGraph:
			if sp >= n {
				return -1
			}
			sp = jxGraphNext(text, sp, n)
			pc++
		case rxOpGBound:
			// Pattern.GraphemeBound: not between a surrogate pair, and not before
			// the first boundary after the previous match's end.
			if sp > 0 && sp < n {
				if isHighSurr(text[sp-1]) && isLowSurr(text[sp]) {
					return -1
				}
				if jxGraphNext(text, st.glast, n) > sp {
					return -1
				}
			}
			pc++
		default:
			return -1
		}
	}
}

// iterJ enters one more iteration of the counted loop whose rxOpCLoop is at pc,
// recording where it began so rxOpCInc can tell an empty one. Twin of rxIterJ in
// lib/regex.js.
func (re *rxRe) iterJ(text []rune, pc, sp int, caps, marks []int, st *rxRunSt) int {
	slot := re.as[pc]
	old := st.cbeg[slot]
	st.cbeg[slot] = sp
	if r := re.run(text, pc+1, sp, caps, marks, st); r >= 0 {
		return r
	}
	st.cbeg[slot] = old
	return -1
}

type rxMatch struct {
	begin int
	end   int
	caps  []int
}

// search finds the leftmost match at or after start. ok=false means the step cap
// was hit, which the caller turns into a clean runtime error.
func (re *rxRe) search(text []rune, start int) (*rxMatch, bool) {
	if start < 0 {
		start = 0
	}
	return re.searchFrom(text, start, start, 0)
}

// searchAdv is search with CPython's must_advance: when adv, a match that is
// empty at start itself is refused, though a longer one there is not. Twin of
// rxSearchAdv in lib/regex.js.
func (re *rxRe) searchAdv(text []rune, start int, adv bool) (*rxMatch, bool) {
	if !adv || start < 0 || start > len(text) {
		return re.search(text, start)
	}
	m, ok := re.attempt(text, start, -1, start, 0, start)
	if !ok || m != nil {
		return m, ok
	}
	return re.search(text, re.nextStart(text, start))
}

// searchFrom is search for java.util.regex's Matcher.find, whose \G and \b{g}
// read where the previous match ended (gpos, glast); search is a fresh Matcher's
// find. Twin of rxSearchFrom in lib/regex.js.
func (re *rxRe) searchFrom(text []rune, from, gpos, glast int) (*rxMatch, bool) {
	for at := from; at <= len(text); at = re.nextStart(text, at) {
		m, ok := re.attempt(text, at, -1, gpos, glast, -1)
		if !ok {
			return nil, false
		}
		if m != nil {
			return m, true
		}
	}
	return nil, true
}

// nextStart is where a search tries next after a failed attempt at `at`; twin
// of the step at the end of rxSearch in lib/regex.js.
func (re *rxRe) nextStart(text []rune, at int) int {
	if re.supp && at < len(text) {
		_, w := rxCpAt(text, at)
		return at + w
	}
	return at + 1
}

// attempt runs the program ONCE, starting at `at`. mustEnd is -1 for an ordinary
// match and the required end position for a whole-input match; it is the one knob
// search, matchAt and matchWhole differ by. adv is -1, or `at` when an empty match
// there is refused (searchAdv).
func (re *rxRe) attempt(text []rune, at, mustEnd, gpos, glast, adv int) (*rxMatch, bool) {
	nslots := 2 * (re.ngroups + 1)
	caps := make([]int, nslots)
	for i := range caps {
		caps[i] = -1
	}
	marks := make([]int, re.nmarks)
	for i := range marks {
		marks[i] = -1
	}
	st := &rxRunSt{mustEnd: mustEnd, lbt: -1, gpos: gpos, glast: glast, adv: adv}
	if len(re.cmin) > 0 {
		st.ctr = make([]int, len(re.cmin))
		st.cbeg = make([]int, len(re.cmin))
		for i := range st.cbeg {
			st.cbeg[i] = -1
		}
	}
	if end := re.run(text, 0, at, caps, marks, st); end >= 0 {
		return &rxMatch{begin: caps[0], end: caps[1], caps: caps}, true
	}
	if st.over {
		return nil, false
	}
	return nil, true
}

// matchAt: the match must BEGIN at `at` - no scanning forward. This is what a
// sticky (JavaScript /y/) match and Python's re.match(pattern, s, pos) want, and it
// is strictly cheaper than searching and then rejecting begin != at.
func (re *rxRe) matchAt(text []rune, at int) (*rxMatch, bool) {
	if at < 0 || at > len(text) {
		return nil, true
	}
	return re.attempt(text, at, -1, at, 0, -1)
}

// matchWhole: the match must cover the WHOLE input. This cannot be a search plus a
// length check - `(a|ab)` against "ab" has to BACKTRACK into the alternation, and a
// leftmost search answers "a" and would wrongly report no match.
func (re *rxRe) matchWhole(text []rune, at int) (*rxMatch, bool) {
	if at < 0 || at > len(text) {
		return nil, true
	}
	return re.attempt(text, at, len(text), at, 0, -1)
}

func (re *rxRe) group(text []rune, m *rxMatch, i int) (string, bool) {
	if i < 0 || i > re.ngroups {
		return "", false
	}
	from, to := m.caps[2*i], m.caps[2*i+1]
	if from < 0 || to < 0 {
		return "", false
	}
	return rxUnitsStr(text[from:to]), true
}

// wtf8Runes is s as code points with each WTF-8 lone surrogate kept as the
// surrogate itself, where []rune(s) would give three U+FFFD.
func wtf8Runes(s string) []rune {
	if strings.IndexByte(s, 0xED) < 0 {
		return []rune(s)
	}
	out := make([]rune, 0, len(s))
	for i := 0; i < len(s); {
		if c := wtf8Surrogate(s[i:]); c != 0 {
			out = append(out, c)
			i += 3
			continue
		}
		r, n := utf8.DecodeRuneInString(s[i:])
		out = append(out, r)
		i += n
	}
	return out
}

// rxUnits is a J text: s as UTF-16 units, one rune each, the way
// java.util.regex indexes a CharSequence. rxUnitsStr is the inverse, and also
// accepts code points: a surrogate pair is joined, a lone surrogate kept as
// WTF-8.
func rxUnits(s string) []rune {
	u := strUnits(s)
	out := make([]rune, len(u))
	for i, c := range u {
		out[i] = rune(c)
	}
	return out
}

func rxUnitsStr(r []rune) string {
	b := make([]byte, 0, len(r))
	for i := 0; i < len(r); i++ {
		c := r[i]
		if isHighSurr(c) && i+1 < len(r) && isLowSurr(r[i+1]) {
			b = utf8.AppendRune(b, surrPair(c, r[i+1]))
			i++
		} else if isSurrogate(c) {
			b = appendWTF8(b, c)
		} else {
			b = utf8.AppendRune(b, c)
		}
	}
	return string(b)
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

// rxGFlagBits answers CPython's flag bits for P's inline flag letters. Twin of
// rxGFlagBits in lib/regex.js.
func rxGFlagBits(letters string) int {
	n := 0
	for _, b := range []struct {
		c string
		v int
	}{{"i", 2}, {"m", 8}, {"s", 16}, {"u", 32}, {"x", 64}, {"a", 256}} {
		if strings.Contains(letters, b.c) {
			n |= b.v
		}
	}
	return n
}

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
// Ruby compiler grammar emits the wrapper for every method call,
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
	if re.java {
		return re, rxUnits(text), m
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

// rxReplace is String#sub / #gsub with a STRING replacement.
func (rt *jsrt) rxReplace(reObj interface{}, s, repl string, all bool) string {
	return rt.rxReplaceWith(reObj, s, all, func(re *rxRe, text []rune, m *rxMatch) string {
		return re.expand(text, m, repl)
	})
}

// rxReplaceBlock is Ruby's String#sub / #gsub with a BLOCK: the block is handed
// each match's text, with $~ set to that match, and answers its replacement.
// Twins: reSub in ruby-interpreter.abnf, rbRxReplaceBlock in lib/ruby-rt.metajs.
func (rt *jsrt) rxReplaceBlock(reObj interface{}, s string, blk interface{}, all bool) string {
	return rt.rxReplaceWith(reObj, s, all, func(re *rxRe, text []rune, m *rxMatch) string {
		rt.rxSetGlobals(rxNewMatchObj(re.src, re.flags, s, m))
		g, _ := re.group(text, m, 0)
		return rt.rubyStr(rt.call(blk, jsUndef, []interface{}{g}))
	})
}

// rxReplaceWith walks the matches of a sub / gsub, each one's replacement text
// coming from with.
func (rt *jsrt) rxReplaceWith(reObj interface{}, s string, all bool, with func(*rxRe, []rune, *rxMatch) string) string {
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
		out.WriteString(with(re, text, m))
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

// rxWantsPattern names the String methods whose argument is a PATTERN, which Ruby
// accepts as a plain String and compiles to a Regexp.
func rxWantsPattern(name string) bool {
	switch name {
	case "match", "match?", "=~", "scan", "sub", "gsub", "sub!", "gsub!":
		return true
	}
	return false
}

// rxRegexMethod answers a method call whose RECEIVER is a Regexp or a MatchData,
// or ok=false when the name is none of theirs - rubyMethod then answers the
// methods every object has (class, is_a?, send, frozen?, ...).
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
			return float64(rubyReOptBits(re.flags)), true
		case "names":
			out := &jsArray{}
			for _, n := range re.names {
				out.elems = append(out.elems, n)
			}
			return out, true
		case "to_s":
			return "(?" + rubyReOpts(re.flags, true) + ":" + rubyReDescSrc(re.src) + ")", true
		case "inspect":
			return "/" + rubyReDescSrc(re.src) + "/" + rubyReOpts(re.flags, false), true
		}
		return nil, false
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
		}
		return nil, false
	}
	// A String receiver whose argument is a pattern: the pattern-taking half of
	// Ruby's String methods, which the base js_rmcall knows nothing about.
	if s, isStr := target.(string); isStr && len(args) > 0 {
		switch name {
		case "match", "match?", "=~", "scan", "sub", "gsub", "sub!", "gsub!", "split", "index":
			pos, blk := rubyBlockArgs(args)
			reObj := argAt(pos, 0)
			if !rxIsRegexObj(reObj) {
				if name == "split" || name == "index" {
					return nil, false // a plain-string argument stays the base behaviour
				}
				// #match / #match? compile a String pattern; #sub / #gsub / #scan
				// match it LITERALLY ("a.c".gsub(".", "-") is "a-c").
				src := rt.rubyStr(reObj)
				if name != "match" && name != "match?" && name != "=~" {
					src = rxQuoteText(src)
				}
				reObj = rxNewRegexObj(src, "mpn")
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
			case "sub", "sub!", "gsub", "gsub!":
				all := name == "gsub" || name == "gsub!"
				if blk != nil {
					return rt.rxReplaceBlock(reObj, s, blk, all), true
				}
				return rt.rxReplace(reObj, s, rt.rubyStr(argAt(pos, 1)), all), true
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
		// A STRING PATTERN reaches the regexp half too: sub, gsub, sub!, gsub!,
		// match, match?, =~ and scan take one, and rxRegexMethod itself turns it into
		// a Regexp (a literal one for sub / gsub / scan). split and index are NOT in
		// the list: with a plain-string argument they are ordinary string operations.
		_, targetIsStr := target.(string)
		if rxIsRegexObj(target) || rxIsMatchObj(target) || rxIsRegexObj(argAt(arr.elems, 0)) ||
			(targetIsStr && rxWantsPattern(name)) {
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

// rubyReDescSrc is a Regexp's source as #inspect and #to_s write it: every "/"
// not already escaped gets a backslash. Twin of reDescSrc in
// ruby-interpreter.abnf.
func rubyReDescSrc(src string) string {
	var b strings.Builder
	for i := 0; i < len(src); i++ {
		switch {
		case src[i] == '\\' && i+1 < len(src):
			b.WriteByte(src[i])
			b.WriteByte(src[i+1])
			i++
		case src[i] == '/':
			b.WriteString("\\/")
		default:
			b.WriteByte(src[i])
		}
	}
	return b.String()
}

// rubyReOptBits is Regexp#options from the engine flags: IGNORECASE 1, EXTENDED
// 2, MULTILINE 4. Twin of reOptBits in ruby-interpreter.abnf.
func rubyReOptBits(flags string) int {
	flags = strings.TrimPrefix(flags, "mpn")
	n := 0
	if strings.Contains(flags, "i") {
		n += 1
	}
	if strings.Contains(flags, "x") {
		n += 2
	}
	if strings.Contains(flags, "s") {
		n += 4
	}
	return n
}

// rubyRegexNew is Regexp.new(src[, opts]): a String source, or a Regexp kept as
// it is (opts ignored). An Integer opts is the option bit set; any other truthy
// one means IGNORECASE (MRI 2.6). Twin of regexNew in ruby-interpreter.abnf.
func (rt *jsrt) rubyRegexNew(args []interface{}) interface{} {
	if rxIsRegexObj(argAt(args, 0)) {
		return argAt(args, 0)
	}
	bits := 0
	if f, isNum := argAt(args, 1).(float64); isNum {
		bits = int(f)
	} else if rubyTruthy(argAt(args, 1)) {
		bits = 1
	}
	flags := "mpn"
	if bits&1 != 0 {
		flags += "i"
	}
	if bits&4 != 0 {
		flags += "s"
	}
	if bits&2 != 0 {
		flags += "x"
	}
	src := rt.rubyStr(argAt(args, 0))
	if re := rxGet(src, flags); !re.ok {
		rt.fail("invalid regexp /%s/: %s", src, re.err)
	}
	return rxNewRegexObj(src, flags)
}

// rubyReOpts is a Ruby Regexp's options as MRI writes them, from the engine flags
// the ruby grammars compile with ("mpn" then i, s for Ruby's m, x). Twin of reOpts
// in languages/ruby-interpreter.abnf, which states the rule.
func rubyReOpts(flags string, embed bool) string {
	flags = strings.TrimPrefix(flags, "mpn")
	m, i, x := strings.Contains(flags, "s"), strings.Contains(flags, "i"), strings.Contains(flags, "x")
	on, off := "", ""
	for _, o := range []struct {
		set bool
		c   string
	}{{m, "m"}, {i, "i"}, {x, "x"}} {
		if o.set {
			on += o.c
		} else {
			off += o.c
		}
	}
	if !embed || off == "" {
		return on
	}
	return on + "-" + off
}
