// Package strings: Go's strings package WRITTEN IN GO, compiled with a program that
// imports "strings" (core.importFile in both go grammars), so one implementation
// serves the interpreter, llvm.Run and a native build alike.
//
// s[i], len(s) and s[i:j] are exact byte operations in every engine, a slice that
// cuts a rune included, so this is written the way Go's own is.
//
// Case mapping covers U+0000..U+052F, Armenian, Latin Extended Additional and the
// fullwidth Latin letters, except the irregular letters of Latin Extended-B, the IPA
// block and the Greek symbols (U+0180..U+01CC, U+0220..U+0245, U+0250..U+029E,
// U+0345, U+037B..U+037F, U+03CF..U+03FF in part): those, and every letter outside
// these blocks, map to themselves here where Go's unicode tables would map them.
package strings

// ----- searching -----

func Index(s, substr string) int {
	n := len(substr)
	if n == 0 {
		return 0
	}
	c := substr[0]
	for i := 0; i+n <= len(s); i++ {
		if s[i] == c && s[i:i+n] == substr {
			return i
		}
	}
	return -1
}

func LastIndex(s, substr string) int {
	n := len(substr)
	if n == 0 {
		return len(s)
	}
	c := substr[0]
	for i := len(s) - n; i >= 0; i-- {
		if s[i] == c && s[i:i+n] == substr {
			return i
		}
	}
	return -1
}

func IndexByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

func LastIndexByte(s string, c byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == c {
			return i
		}
	}
	return -1
}

func IndexRune(s string, r rune) int {
	for i, c := range s {
		if c == r {
			return i
		}
	}
	return -1
}

func IndexFunc(s string, f func(rune) bool) int {
	for i, c := range s {
		if f(c) {
			return i
		}
	}
	return -1
}

func LastIndexFunc(s string, f func(rune) bool) int {
	last := -1
	for i, c := range s {
		if f(c) {
			last = i
		}
	}
	return last
}

func IndexAny(s, chars string) int {
	for i, c := range s {
		if IndexRune(chars, c) >= 0 {
			return i
		}
	}
	return -1
}

func LastIndexAny(s, chars string) int {
	last := -1
	for i, c := range s {
		if IndexRune(chars, c) >= 0 {
			last = i
		}
	}
	return last
}

func Contains(s, substr string) bool                { return Index(s, substr) >= 0 }
func ContainsRune(s string, r rune) bool            { return IndexRune(s, r) >= 0 }
func ContainsAny(s, chars string) bool              { return IndexAny(s, chars) >= 0 }
func ContainsFunc(s string, f func(rune) bool) bool { return IndexFunc(s, f) >= 0 }

func HasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func HasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

// Count counts non-overlapping instances; an empty substr counts the runes plus one.
func Count(s, substr string) int {
	if len(substr) == 0 {
		n := 1
		for range s {
			n++
		}
		return n
	}
	n := 0
	for {
		i := Index(s, substr)
		if i < 0 {
			return n
		}
		n++
		s = s[i+len(substr):]
	}
}

func Compare(a, b string) int {
	if a == b {
		return 0
	}
	if a < b {
		return -1
	}
	return 1
}

func Cut(s, sep string) (before, after string, found bool) {
	if i := Index(s, sep); i >= 0 {
		return s[:i], s[i+len(sep):], true
	}
	return s, "", false
}

func CutPrefix(s, prefix string) (after string, found bool) {
	if !HasPrefix(s, prefix) {
		return s, false
	}
	return s[len(prefix):], true
}

func CutSuffix(s, suffix string) (before string, found bool) {
	if !HasSuffix(s, suffix) {
		return s, false
	}
	return s[:len(s)-len(suffix)], true
}

// ----- splitting and joining -----

// explode splits s into its runes: at most n parts (all of them when n < 0), the
// last holding the rest.
func explode(s string, n int) []string {
	var a []string
	for i, c := range s {
		if n > 0 && len(a) == n-1 {
			a = append(a, s[i:])
			return a
		}
		a = append(a, string(c))
	}
	return a
}

func genSplit(s, sep string, sepSave, n int) []string {
	if n == 0 {
		return nil
	}
	if sep == "" {
		return explode(s, n)
	}
	var a []string
	for n < 0 || len(a) < n-1 {
		m := Index(s, sep)
		if m < 0 {
			break
		}
		a = append(a, s[:m+sepSave])
		s = s[m+len(sep):]
	}
	return append(a, s)
}

func Split(s, sep string) []string              { return genSplit(s, sep, 0, -1) }
func SplitN(s, sep string, n int) []string      { return genSplit(s, sep, 0, n) }
func SplitAfter(s, sep string) []string         { return genSplit(s, sep, len(sep), -1) }
func SplitAfterN(s, sep string, n int) []string { return genSplit(s, sep, len(sep), n) }

func Fields(s string) []string { return FieldsFunc(s, isSpace) }

func FieldsFunc(s string, f func(rune) bool) []string {
	var a []string
	start := -1
	for i, c := range s {
		if f(c) {
			if start >= 0 {
				a = append(a, s[start:i])
				start = -1
			}
		} else if start < 0 {
			start = i
		}
	}
	if start >= 0 {
		a = append(a, s[start:])
	}
	return a
}

func Join(elems []string, sep string) string {
	out := ""
	for i, e := range elems {
		if i > 0 {
			out += sep
		}
		out += e
	}
	return out
}

func Repeat(s string, count int) string {
	if count < 0 {
		panic("strings: negative Repeat count")
	}
	out := ""
	for count > 0 {
		if count%2 == 1 {
			out += s
		}
		s += s
		count /= 2
	}
	return out
}

// ----- replacing -----

// Replace replaces the first n non-overlapping instances of old (all of them when
// n < 0). An empty old matches at the start and after every rune.
func Replace(s, old, new string, n int) string {
	if old == new || n == 0 {
		return s
	}
	out := ""
	if old == "" {
		k := 0
		for _, c := range s {
			if n >= 0 && k >= n {
				out += string(c)
				continue
			}
			out += new + string(c)
			k++
		}
		if n < 0 || k < n {
			out += new
		}
		return out
	}
	for n != 0 {
		i := Index(s, old)
		if i < 0 {
			break
		}
		out += s[:i] + new
		s = s[i+len(old):]
		n--
	}
	return out + s
}

func ReplaceAll(s, old, new string) string { return Replace(s, old, new, -1) }

// A Replacer applies its (old, new) pairs in one left-to-right pass; at each position
// the earliest pair whose old string matches there wins. An empty old string matches
// between RUNES here, where Go's replacer also matches between the bytes of one.
type Replacer struct {
	olds []string
	news []string
}

func NewReplacer(oldnew ...string) *Replacer {
	if len(oldnew)%2 == 1 {
		panic("strings.NewReplacer: odd argument count")
	}
	r := &Replacer{}
	for i := 0; i < len(oldnew); i += 2 {
		r.olds = append(r.olds, oldnew[i])
		r.news = append(r.news, oldnew[i+1])
	}
	return r
}

func (r *Replacer) Replace(s string) string {
	out := ""
	last := 0
	prevEmpty := false
	for i := 0; i <= len(s); {
		hit := -1
		for k, o := range r.olds {
			if o == "" && prevEmpty {
				continue
			}
			if HasPrefix(s[i:], o) {
				hit = k
				break
			}
		}
		prevEmpty = hit >= 0 && r.olds[hit] == ""
		if hit >= 0 {
			out += s[last:i] + r.news[hit]
			i += len(r.olds[hit])
			last = i
			continue
		}
		i += runeLen(s, i)
	}
	return out + s[last:]
}

// runeLen is the byte length of the rune utf8.DecodeRuneInString reads at s[i]: 1
// for a byte that does not begin a well-formed sequence, and 1 at the end of s.
func runeLen(s string, i int) int {
	if i >= len(s) || s[i] < 0x80 {
		return 1
	}
	n := 0
	switch {
	case s[i] >= 0xC2 && s[i] <= 0xDF:
		n = 2
	case s[i] >= 0xE0 && s[i] <= 0xEF:
		n = 3
	case s[i] >= 0xF0 && s[i] <= 0xF4:
		n = 4
	default:
		return 1
	}
	if i+n > len(s) {
		return 1
	}
	lo, hi := byte(0x80), byte(0xBF)
	switch s[i] {
	case 0xE0:
		lo = 0xA0
	case 0xED:
		hi = 0x9F
	case 0xF0:
		lo = 0x90
	case 0xF4:
		hi = 0x8F
	}
	if s[i+1] < lo || s[i+1] > hi {
		return 1
	}
	for k := 2; k < n; k++ {
		if s[i+k] < 0x80 || s[i+k] > 0xBF {
			return 1
		}
	}
	return n
}

// ----- trimming -----

func TrimLeftFunc(s string, f func(rune) bool) string {
	for i, c := range s {
		if !f(c) {
			return s[i:]
		}
	}
	return ""
}

// The kept prefix ends where the rune after the last one f refuses begins.
func TrimRightFunc(s string, f func(rune) bool) string {
	end, kept := 0, false
	for i, c := range s {
		if kept {
			end = i
		}
		kept = !f(c)
	}
	if kept {
		end = len(s)
	}
	return s[:end]
}

func TrimFunc(s string, f func(rune) bool) string {
	return TrimRightFunc(TrimLeftFunc(s, f), f)
}

func TrimSpace(s string) string { return TrimFunc(s, isSpace) }

func Trim(s, cutset string) string {
	return TrimFunc(s, func(r rune) bool { return IndexRune(cutset, r) >= 0 })
}

func TrimLeft(s, cutset string) string {
	return TrimLeftFunc(s, func(r rune) bool { return IndexRune(cutset, r) >= 0 })
}

func TrimRight(s, cutset string) string {
	return TrimRightFunc(s, func(r rune) bool { return IndexRune(cutset, r) >= 0 })
}

func TrimPrefix(s, prefix string) string {
	if HasPrefix(s, prefix) {
		return s[len(prefix):]
	}
	return s
}

func TrimSuffix(s, suffix string) string {
	if HasSuffix(s, suffix) {
		return s[:len(s)-len(suffix)]
	}
	return s
}

// unicode.IsSpace.
func isSpace(r rune) bool {
	switch r {
	case '\t', '\n', '\v', '\f', '\r', ' ', 0x85, 0xA0, 0x1680, 0x2028, 0x2029, 0x202F, 0x205F, 0x3000:
		return true
	}
	return r >= 0x2000 && r <= 0x200A
}

// ----- case mapping -----

func Map(mapping func(rune) rune, s string) string {
	out := ""
	for _, c := range s {
		m := mapping(c)
		if m >= 0 {
			out += string(m)
		}
	}
	return out
}

func ToUpper(s string) string { return Map(toUpper, s) }
func ToLower(s string) string { return Map(toLower, s) }
func ToTitle(s string) string { return Map(toTitle, s) }

// Title maps the first letter of every word to its title case; a word starts
// after a separator - ASCII other than a letter, digit or '_', or a space.
func Title(s string) string {
	prev := ' '
	return Map(func(r rune) rune {
		sep := isSpace(prev)
		if prev < 0x80 {
			sep = !(prev >= 'a' && prev <= 'z' || prev >= 'A' && prev <= 'Z' || prev >= '0' && prev <= '9' || prev == '_')
		}
		prev = r
		if sep {
			return toTitle(r)
		}
		return r
	}, s)
}

// toTitle is toUpper except for the digraphs, whose title case is the middle one
// of the three (ǅ, ǈ, ǋ, ǲ).
func toTitle(r rune) rune {
	switch {
	case r >= 0x1C4 && r <= 0x1CC:
		return r - (r-0x1C4)%3 + 1
	case r >= 0x1F1 && r <= 0x1F3:
		return 0x1F2
	}
	return toUpper(r)
}

// EqualFold compares under simple case folding, rune by rune.
func EqualFold(s, t string) bool {
	a := []rune(s)
	b := []rune(t)
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if foldRune(a[i]) != foldRune(b[i]) {
			return false
		}
	}
	return true
}

// foldRune names r's case-fold orbit by one member: the lower-case letter, except
// where Unicode's orbit is not just an (upper, lower) pair. U+0130 and U+0131 fold to
// nothing else, so 'I' is not equal to either under EqualFold.
func foldRune(r rune) rune {
	switch r {
	case 0x130:
		return r
	case 0x17F:
		return 's'
	case 0xB5:
		return 0x3BC
	case 0x3C2:
		return 0x3C3
	}
	return toLower(r)
}

// pairCase maps r inside a block of (upper, lower) pairs, each lower letter one past
// its upper; upperOdd says whether the upper letters sit at the odd code points.
func pairCase(r rune, upperOdd bool, toUp bool) rune {
	isUp := (r%2 == 1) == upperOdd
	if isUp == toUp {
		return r
	}
	if toUp {
		return r - 1
	}
	return r + 1
}

func toUpper(r rune) rune {
	switch {
	case r >= 'a' && r <= 'z':
		return r - 32
	case r < 0x80:
		return r
	case r == 0xB5:
		return 0x39C
	case r == 0xFF:
		return 0x178
	case r >= 0xE0 && r <= 0xFE && r != 0xF7:
		return r - 32
	case r == 0x131:
		return 'I'
	case r == 0x17F:
		return 'S'
	// The Latin digraphs come in threes - upper, title, lower - and upper-case to
	// the first: Ǆ ǅ ǆ, Ǉ ǈ ǉ, Ǌ ǋ ǌ, Ǳ ǲ ǳ.
	case r == 0x1C5 || r == 0x1C6 || r == 0x1C8 || r == 0x1C9 || r == 0x1CB || r == 0x1CC:
		return r - (r-0x1C4)%3
	case r == 0x1F2 || r == 0x1F3:
		return 0x1F1
	case r == 0x3C2:
		return 0x3A3
	case r >= 0x3B1 && r <= 0x3CB:
		return r - 32
	case r == 0x3AC:
		return 0x386
	case r >= 0x3AD && r <= 0x3AF:
		return r - 37
	case r == 0x3CC:
		return 0x38C
	case r == 0x3CD || r == 0x3CE:
		return r - 63
	case r >= 0x430 && r <= 0x44F:
		return r - 32
	case r >= 0x450 && r <= 0x45F:
		return r - 80
	case r == 0x4CF:
		return 0x4C0
	case r >= 0x561 && r <= 0x586:
		return r - 0x30
	case r >= 0xFF41 && r <= 0xFF5A:
		return r - 0x20
	}
	return pairBlocks(r, true)
}

func toLower(r rune) rune {
	switch {
	case r >= 'A' && r <= 'Z':
		return r + 32
	case r < 0x80:
		return r
	case r >= 0xC0 && r <= 0xDE && r != 0xD7:
		return r + 32
	case r == 0x130:
		return 'i'
	case r == 0x178:
		return 0xFF
	case r >= 0x1C4 && r <= 0x1CC && (r-0x1C4)%3 != 2:
		return r + 2 - (r-0x1C4)%3
	case r == 0x1F1 || r == 0x1F2:
		return 0x1F3
	case r == 0x1E9E:
		return 0xDF
	case r == 0x2126:
		return 0x3C9
	case r == 0x212A:
		return 'k'
	case r == 0x212B:
		return 0xE5
	case r >= 0x391 && r <= 0x3AB && r != 0x3A2:
		return r + 32
	case r == 0x386:
		return 0x3AC
	case r >= 0x388 && r <= 0x38A:
		return r + 37
	case r == 0x38C:
		return 0x3CC
	case r == 0x38E || r == 0x38F:
		return r + 63
	case r >= 0x410 && r <= 0x42F:
		return r + 32
	case r >= 0x400 && r <= 0x40F:
		return r + 80
	case r == 0x4C0:
		return 0x4CF
	case r >= 0x531 && r <= 0x556:
		return r + 0x30
	case r >= 0xFF21 && r <= 0xFF3A:
		return r + 0x20
	}
	return pairBlocks(r, false)
}

// pairBlocks maps r inside the blocks of alternating (upper, lower) pairs: most of
// Latin Extended-A and -B, and parts of Greek and Cyrillic. U+0130, U+0131, U+0138
// and U+0149 have no partner here (toUpper and toLower handle the first two).
func pairBlocks(r rune, toUp bool) rune {
	switch {
	case r == 0x130 || r == 0x131 || r == 0x138 || r == 0x149:
		return r
	case (r >= 0x139 && r <= 0x148) || (r >= 0x179 && r <= 0x17E) || (r >= 0x1CD && r <= 0x1DC) || (r >= 0x4C1 && r <= 0x4CE):
		return pairCase(r, true, toUp)
	case (r >= 0x100 && r <= 0x177) || (r >= 0x1DE && r <= 0x1EF) || (r >= 0x1F8 && r <= 0x21F) ||
		(r >= 0x222 && r <= 0x233) || (r >= 0x246 && r <= 0x24F) || (r >= 0x370 && r <= 0x373) ||
		r == 0x376 || r == 0x377 || (r >= 0x3D8 && r <= 0x3EF) || (r >= 0x460 && r <= 0x481) ||
		(r >= 0x48A && r <= 0x4BF) || (r >= 0x4D0 && r <= 0x52F) || (r >= 0x1E00 && r <= 0x1E95) ||
		(r >= 0x1EA0 && r <= 0x1EFF):
		return pairCase(r, false, toUp)
	}
	return r
}

// ----- Builder -----

// A Builder accumulates a string; its zero value is ready to use.
type Builder struct {
	s string
}

func (b *Builder) WriteString(s string) (int, error) {
	b.s += s
	return len(s), nil
}

func (b *Builder) WriteByte(c byte) error {
	b.s += string([]byte{c})
	return nil
}

func (b *Builder) WriteRune(r rune) (int, error) {
	t := string(r)
	b.s += t
	return len(t), nil
}

func (b *Builder) Write(p []byte) (int, error) {
	b.s += string(p)
	return len(p), nil
}

func (b *Builder) String() string { return b.s }
func (b *Builder) Len() int       { return len(b.s) }
func (b *Builder) Cap() int       { return len(b.s) }
func (b *Builder) Reset()         { b.s = "" }
func (b *Builder) Grow(n int) {
	if n < 0 {
		panic("strings.Builder.Grow: negative count")
	}
}
