package abnf

import (
	"math"
	"strconv"
	"strings"
	"unicode/utf8"
)

// PHP's byte-string primitives and the natives lib/php-stdlib.php is written on -
// the GO TWIN of languages/lib/php-strings.js (shared by the interpreter half and
// layer 2) plus the js_ph* externs that wrap it. A PHP string's text is held in
// WTF-8 like every string here, with a byte that is not part of a character held
// as the lone surrogate U+DC00 + byte (php-strings.js); phpRaw turns that text into
// the bytes PHP's offsets count, and phpBytesText turns bytes back into it.

// phpRaw is phsBytes: the bytes of s, a held byte (WTF-8 ED B2 80 .. ED B3 BF)
// being that one byte.
func phpRaw(s string) string {
	if strings.IndexByte(s, 0xED) < 0 {
		return s
	}
	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); {
		if s[i] == 0xED && i+2 < len(s) && (s[i+1] == 0xB2 || s[i+1] == 0xB3) {
			b = append(b, byte(s[i+1]&1)<<6|s[i+2]&0x3F|0x80)
			i += 3
			continue
		}
		b = append(b, s[i])
		i++
	}
	return string(b)
}

// phpBytesText is phsText: a well-formed UTF-8 sequence is its character and any
// other byte is held as U+DC00 + byte.
func phpBytesText(b string) string {
	if utf8.ValidString(b) {
		return b
	}
	out := make([]byte, 0, len(b)+8)
	for i := 0; i < len(b); {
		r, size := utf8.DecodeRuneInString(b[i:])
		if r == utf8.RuneError && size <= 1 {
			out = appendWTF8(out, 0xDC00+rune(b[i]))
			i++
			continue
		}
		out = append(out, b[i:i+size]...)
		i += size
	}
	return string(out)
}

// phpJoin is phsJoin: a . b, re-read when a ends in a held byte and b starts with
// a held continuation byte (WTF-8 ED B2 ..), since the two may spell a character.
func phpJoin(a, b string) string {
	n := len(a)
	if n < 3 || len(b) < 3 || b[0] != 0xED || b[1] != 0xB2 || a[n-3] != 0xED || (a[n-2] != 0xB2 && a[n-2] != 0xB3) {
		return a + b
	}
	return phpBytesText(phpRaw(a) + phpRaw(b))
}

// phpOffset is phsOffset: $s[$i], the one-byte string at byte offset i (from the
// end when negative), "" outside the string.
func phpOffset(s string, i int) string {
	r := phpRaw(s)
	if i < 0 {
		i += len(r)
	}
	if i < 0 || i >= len(r) {
		return ""
	}
	return phpBytesText(r[i : i+1])
}

// phpSetOffset is phsSetOffset: the byte at offset i replaced by c's first byte,
// a write past the end padding with spaces; ok is false before the start.
func phpSetOffset(s string, i int, c string) (string, bool) {
	r := []byte(phpRaw(s))
	if i < 0 {
		i += len(r)
	}
	if i < 0 {
		return "", false
	}
	for len(r) < i {
		r = append(r, ' ')
	}
	cb := phpRaw(c)[0]
	if i == len(r) {
		r = append(r, cb)
	} else {
		r[i] = cb
	}
	return phpBytesText(string(r)), true
}

// phpSubstr is phsSubstr: PHP 8's substr rules over bytes.
func phpSubstr(s string, start int, length int, hasLen bool) string {
	s = phpRaw(s)
	n := len(s)
	f := start
	if f > n {
		return ""
	}
	if f < 0 {
		if -f > n {
			f = 0
		} else {
			f = n + f
		}
	}
	l := n - f
	if hasLen {
		if length < 0 {
			if -length > n-f {
				l = 0
			} else {
				l = n - f + length
			}
		} else if length < n-f {
			l = length
		}
	}
	if l <= 0 {
		return ""
	}
	return phpBytesText(s[f : f+l])
}

// phpStrpos is phsStrpos: the first match at or after off, -1 for none, -2 for
// an offset outside the haystack.
func phpStrpos(h, needle string, off int) int {
	h, needle = phpRaw(h), phpRaw(needle)
	n := len(h)
	if off < 0 {
		off += n
	}
	if off < 0 || off > n {
		return -2
	}
	at := strings.Index(h[off:], needle)
	if at < 0 {
		return -1
	}
	return at + off
}

// phpStrrpos is phsStrrpos: the last match, searched from a non-negative off, or
// ending by n+off+len(needle) for a negative one.
func phpStrrpos(h, needle string, off int) int {
	h, needle = phpRaw(h), phpRaw(needle)
	n := len(h)
	lo, hi := 0, n
	if off >= 0 {
		if off > n {
			return -2
		}
		lo = off
	} else {
		if -off > n {
			return -2
		}
		if -off >= len(needle) {
			hi = n + off + len(needle)
		}
	}
	for p := hi - len(needle); p >= lo; p-- {
		if strings.HasPrefix(h[p:], needle) {
			return p
		}
	}
	return -1
}

// phpCase is phsCase: ASCII-only case mapping.
func phpCase(s string, up bool) string {
	b := []byte(s)
	for i, c := range b {
		if up && c >= 'a' && c <= 'z' {
			b[i] = c - 32
		} else if !up && c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}

// phpCoreIface is phsCoreIface: one of the interfaces the engine itself declares.
func phpCoreIface(n string) bool {
	switch phpCase(strings.TrimPrefix(n, "\\"), false) {
	case "traversable", "iterator", "iteratoraggregate", "arrayaccess", "countable", "serializable",
		"stringable", "throwable", "unitenum", "backedenum", "jsonserializable":
		return true
	}
	return false
}

// phpCoreIfaceExtends is phsCoreIfaceExtends: does the engine's own interface
// iface extend the interface name?
func phpCoreIfaceExtends(iface, name string) bool {
	p := phpCase(strings.TrimPrefix(iface, "\\"), false)
	switch phpCase(strings.TrimPrefix(name, "\\"), false) {
	case "traversable":
		return p == "iterator" || p == "iteratoraggregate"
	case "unitenum":
		return p == "backedenum"
	case "stringable":
		return p == "throwable"
	}
	return false
}

// phpCoreClassIs is phsCoreClassIs: is the engine's own class cls ("closure" or
// "generator") the class name, or an implementor of it?
func phpCoreClassIs(cls, name string) bool {
	q := phpCase(strings.TrimPrefix(name, "\\"), false)
	return q == cls || (cls == "generator" && (q == "iterator" || q == "traversable"))
}

// phpCloSplit is phsCloSplit: s cut at every sep, "" being no fields at all.
func phpCloSplit(s, sep string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, sep)
}

// phpCloStatics is phsCloStatics: the static variable names of closure record d.
func phpCloStatics(d string) []string { return phpCloSplit(strings.Split(d, "\n")[3], ",") }

// phpCloDump is phsCloDump: var_dump of a closure literal from its record d, its
// handle, the dumps of its static variables' values and of its bound $this.
func phpCloDump(d string, id int, ind int, statDumps []string, thisDump string) string {
	f := strings.Split(d, "\n")
	st, ps := phpCloSplit(f[3], ","), phpCloSplit(f[4], ",")
	nreq, _ := strconv.Atoi(f[5])
	pad := strings.Repeat(" ", ind)
	p2, p4 := pad+"  ", pad+"    "
	n := 3
	if f[1] == "" {
		n = 1
	}
	if len(st) > 0 {
		n++
	}
	if thisDump != "" {
		n++
	}
	if len(ps) > 0 {
		n++
	}
	out := pad + "object(Closure)#" + strconv.Itoa(id) + " (" + strconv.Itoa(n) + ") {\n"
	if f[1] == "" {
		out += p2 + "[\"function\"]=>\n" + p2 + "string(" + strconv.Itoa(len(phpRaw(f[0]))) + ") \"" + f[0] + "\"\n"
	} else {
		out += p2 + "[\"name\"]=>\n" + p2 + "string(" + strconv.Itoa(len(phpRaw(f[0]))) + ") \"" + f[0] + "\"\n"
		out += p2 + "[\"file\"]=>\n" + p2 + "string(" + strconv.Itoa(len(phpRaw(f[1]))) + ") \"" + f[1] + "\"\n"
		out += p2 + "[\"line\"]=>\n" + p2 + "int(" + f[2] + ")\n"
	}
	if len(st) > 0 {
		out += p2 + "[\"static\"]=>\n" + p2 + "array(" + strconv.Itoa(len(st)) + ") {\n"
		for i, s := range st {
			out += p4 + "[\"" + s[1:] + "\"]=>\n" + statDumps[i]
		}
		out += p2 + "}\n"
	}
	if thisDump != "" {
		out += p2 + "[\"this\"]=>\n" + thisDump
	}
	if len(ps) > 0 {
		out += p2 + "[\"parameter\"]=>\n" + p2 + "array(" + strconv.Itoa(len(ps)) + ") {\n"
		for j, p := range ps {
			req := "<optional>"
			if j < nreq {
				req = "<required>"
			}
			out += p4 + "[\"" + p + "\"]=>\n" + p4 + "string(10) \"" + req + "\"\n"
		}
		out += p2 + "}\n"
	}
	return out + pad + "}\n"
}

// phpArityMsg is phsArityMsg: zend_wrong_parameters_count_error's text.
func phpArityMsg(name string, min, max, given int) string {
	want, how := max, "at most"
	if given < min {
		want, how = min, "at least"
	}
	if min == max {
		how = "exactly"
	}
	s := "s"
	if want == 1 {
		s = ""
	}
	return name + "() expects " + how + " " + strconv.Itoa(want) + " argument" + s + ", " + strconv.Itoa(given) + " given"
}

// phpMissArgMsg is phsMissArgMsg: zend_missing_arg_error's text.
func phpMissArgMsg(name string, passed, req int, exact bool, where string) string {
	s := "Too few arguments to function " + name + "(), " + strconv.Itoa(passed) + " passed" + where
	how := "at least"
	if exact {
		how = "exactly"
	}
	return s + " and " + how + " " + strconv.Itoa(req) + " expected"
}

// phpNotPassedMsg is phsNotPassedMsg: zend_handle_undef_args' text.
func phpNotPassedMsg(name string, n int, pname string) string {
	return name + "(): Argument #" + strconv.Itoa(n) + " (" + pname + ") not passed"
}

// phpCoreClass is phsCoreClass: the lower-case name of the engine's own class n
// denotes, or "".
func phpCoreClass(n string) string {
	if k := phpCase(strings.TrimPrefix(n, "\\"), false); k == "closure" || k == "generator" {
		return k
	}
	return ""
}

// phpBitOp is phsBitOp: &, |, ^ of two strings byte by byte ('|' to the longer,
// the others to the shorter), and "~" every byte of a inverted.
func phpBitOp(op string, a, b string) string {
	x, y := phpRaw(a), phpRaw(b)
	n := len(x)
	if (op == "|") == (len(y) > n) {
		n = len(y)
	}
	out := make([]byte, n)
	for i := range out {
		var p, q byte
		if i < len(x) {
			p = x[i]
		}
		if i < len(y) {
			q = y[i]
		}
		switch op {
		case "&":
			out[i] = p & q
		case "|":
			out[i] = p | q
		case "^":
			out[i] = p ^ q
		default:
			out[i] = ^p
		}
	}
	return phpBytesText(string(out))
}

// phpIncr is phsIncr: Perl's increment of a non-numeric string's trailing
// alphanumeric run.
func phpIncr(s string) string {
	if s == "" {
		return "1"
	}
	b := []byte(phpRaw(s))
	carry, last := false, byte(0)
	for pos := len(b) - 1; pos >= 0; pos-- {
		c := b[pos]
		var lo, hi byte
		switch {
		case c >= 'a' && c <= 'z':
			lo, hi, last = 'a', 'z', 'a'
		case c >= 'A' && c <= 'Z':
			lo, hi, last = 'A', 'Z', 'A'
		case c >= '0' && c <= '9':
			lo, hi, last = '0', '9', '1'
		default:
			carry = false
		}
		if hi == 0 {
			break
		}
		carry = c == hi
		if carry {
			b[pos] = lo
		} else {
			b[pos] = c + 1
			break
		}
	}
	t := phpBytesText(string(b))
	if carry {
		return string(last) + t
	}
	return t
}

// phpFmtFloat is phsFmtFloat: sprintf's %f / %e of a non-negative finite double,
// exact and rounded half to even, with php_conv_fp's unpadded exponent.
func phpFmtFloat(a float64, conv string, prec int) string {
	if conv == "f" {
		return strconv.FormatFloat(a, 'f', prec, 64)
	}
	s := strconv.FormatFloat(a, 'e', prec, 64)
	at := strings.IndexByte(s, 'e')
	exp, _ := strconv.Atoi(s[at+1:])
	if exp < 0 {
		return s[:at] + "e-" + strconv.Itoa(-exp)
	}
	return s[:at] + "e+" + strconv.Itoa(exp)
}

// phpMath is phsMath: the C library function of one or two doubles op names.
func phpMath(op string, a, b float64) float64 {
	switch op {
	case "floor":
		return math.Floor(a)
	case "ceil":
		return math.Ceil(a)
	case "trunc":
		return math.Trunc(a)
	case "sqrt":
		return math.Sqrt(a)
	case "exp":
		return math.Exp(a)
	case "expm1":
		return math.Expm1(a)
	case "log":
		return math.Log(a)
	case "log1p":
		return math.Log1p(a)
	case "log2":
		return math.Log2(a)
	case "log10":
		return math.Log10(a)
	case "sin":
		return math.Sin(a)
	case "cos":
		return math.Cos(a)
	case "tan":
		return math.Tan(a)
	case "asin":
		return math.Asin(a)
	case "acos":
		return math.Acos(a)
	case "atan":
		return math.Atan(a)
	case "sinh":
		return math.Sinh(a)
	case "cosh":
		return math.Cosh(a)
	case "tanh":
		return math.Tanh(a)
	case "asinh":
		return math.Asinh(a)
	case "acosh":
		return math.Acosh(a)
	case "atanh":
		return math.Atanh(a)
	case "atan2":
		return math.Atan2(a, b)
	case "hypot":
		return math.Hypot(a, b)
	case "pow":
		return crPow(a, b)
	case "div":
		return a / b
	case "fmod":
		return math.Mod(a, b)
	}
	return math.Copysign(a, b)
}

func init() {
	rxExtraExterns = append(rxExtraExterns, func(rt *jsrt, m map[string]func(args []uint64) uint64) {
		u := rt.unwrap
		w := rt.wrap
		num := func(v interface{}) int { return int(rt.phpToLong(v)) }
		m["js_phsubstr"] = func(a []uint64) uint64 {
			l := u(a[2])
			return w(phpSubstr(rt.phpStr(u(a[0])), num(u(a[1])), num(l), !isNullish(l)))
		}
		pos := func(name string, at int) uint64 {
			if at == -2 {
				rt.phpRaise("ValueError", name+"(): Argument #3 ($offset) must be contained in argument #1 ($haystack)")
			}
			if at < 0 {
				return w(false)
			}
			return w(phpMkInt(int64(at)))
		}
		m["js_phstrpos"] = func(a []uint64) uint64 {
			return pos("strpos", phpStrpos(rt.phpStr(u(a[0])), rt.phpStr(u(a[1])), num(u(a[2]))))
		}
		m["js_phstrrpos"] = func(a []uint64) uint64 {
			return pos("strrpos", phpStrrpos(rt.phpStr(u(a[0])), rt.phpStr(u(a[1])), num(u(a[2]))))
		}
		m["js_phstrcmp"] = func(a []uint64) uint64 {
			return w(phpMkInt(int64(strings.Compare(phpRaw(rt.phpStr(u(a[0]))), phpRaw(rt.phpStr(u(a[1])))))))
		}
		m["js_phord"] = func(a []uint64) uint64 {
			s := phpRaw(rt.phpStr(u(a[0])))
			if s == "" {
				return w(phpMkInt(0))
			}
			return w(phpMkInt(int64(s[0])))
		}
		m["js_phstrlen"] = func(a []uint64) uint64 {
			return w(phpMkInt(int64(len(phpRaw(rt.phpStr(u(a[0])))))))
		}
		// A string literal assembled from bytes, one per character (dqPiece in
		// php-to-llvm-ir.abnf).
		m["js_phfrombytes"] = func(a []uint64) uint64 {
			s := rt.phpStr(u(a[0]))
			b := make([]byte, 0, len(s))
			for _, r := range s {
				b = append(b, byte(r))
			}
			return w(phpBytesText(string(b)))
		}
		// $s[$k] = $v on a string: the string with the one byte replaced.
		m["js_phsetoffset"] = func(a []uint64) uint64 {
			c := rt.phpStr(u(a[2]))
			if c == "" {
				rt.fail("cannot assign an empty string to a string offset")
			}
			s, ok := phpSetOffset(rt.phpStr(u(a[0])), int(rt.phpToLong(u(a[1]))), c)
			if !ok {
				rt.fail("cannot write to a negative string offset")
			}
			return w(s)
		}
		m["js_phchr"] = func(a []uint64) uint64 {
			b := rt.phpToLong(u(a[0])) % 256
			if b < 0 {
				b += 256
			}
			return w(phpBytesText(string([]byte{byte(b)})))
		}
		m["js_phcase"] = func(a []uint64) uint64 {
			return w(phpCase(rt.phpStr(u(a[0])), rt.truthy(u(a[1]))))
		}
		m["js_phfmtfloat"] = func(a []uint64) uint64 {
			x := phpNumOf(rt.phpToNum(u(a[0])))
			if x < 0 {
				x = -x
			}
			conv := rt.phpStr(u(a[1]))
			prec := num(u(a[2]))
			if conv == "g" || conv == "r" {
				return w(strings.Replace(phpFloFormat(x, prec, conv == "g"), "E", "e", 1))
			}
			return w(phpFmtFloat(x, conv, prec))
		}
		// A new object (or closure) takes the next handle; see phpHandles.
		m["js_phoid"] = func(a []uint64) uint64 {
			o, _ := u(a[0]).(*jsObject)
			rt.phpStamp(o)
			return a[0]
		}
		// (desc, sig, env, this) -> the record of a closure literal being made,
		// which takes the next handle: its debug record (phsCloDesc), the signature
		// js_phplace reads, the scope its captures live in and its bound $this.
		// The emitter stores it in the closure's meta (js_jsfnmeta). Twin of
		// js_phclo in lib/php-rt.metajs and closureBody in php-interpreter.abnf.
		// (desc, sig) -> the record of a named function or a method: its debug
		// record (phsCloDesc, no file) and its signature. A Closure of a named
		// function (js_phfnnew) takes a handle when var_dump or spl_object_id
		// first sees it (phpFnID). Twin of js_phfnrec in
		// lib/php-rt.metajs and fnRecs in php-interpreter.abnf.
		m["js_phfnrec"] = func(a []uint64) uint64 {
			rec := newJSObject()
			rec.set("fd", u(a[0]))
			rec.set("s", u(a[1]))
			return w(rec)
		}
		// (f) -> the record of a new Closure of the named function f: a copy of
		// f's, with f's by-reference mask (mk) and f itself; undefined for
		// anything else. Twin of js_phfnnew in lib/php-rt.metajs and fnFresh in
		// php-interpreter.abnf.
		m["js_phfnnew"] = func(a []uint64) uint64 {
			fr := phpFnRec(u(a[0]))
			if fr == nil {
				return jsHUndefined
			}
			rec := newJSObject()
			rec.set("fd", fr.props["fd"])
			rec.set("s", fr.props["s"])
			rec.set("mk", float64(jsvFnMeta(u(a[0])).length))
			rec.set("f", u(a[0]))
			return w(rec)
		}
		// (v) -> the function a Closure made by js_phfnnew forwards to; v itself
		// for anything else. Twin of js_phfnof in lib/php-rt.metajs.
		m["js_phfnof"] = func(a []uint64) uint64 {
			if fr := phpFnRec(u(a[0])); fr != nil && fr.props["f"] != nil {
				return w(fr.props["f"])
			}
			return a[0]
		}
		m["js_phclo"] = func(a []uint64) uint64 {
			rt.phpStamp(nil)
			rec := newJSObject()
			rec.set("d", u(a[0]))
			rec.set("s", u(a[1]))
			rec.set("e", rt.scopeOf(a[2]))
			rec.set("t", u(a[3]))
			rec.set("i", float64(phpHandles[rt]))
			return w(rec)
		}
		// An object's handle, or a closure literal's; any other Closure keeps none
		// and answers 0. Twin of __mec_oid in php-interpreter.abnf.
		m["js_phoidof"] = func(a []uint64) uint64 {
			if o, ok := u(a[0]).(*jsObject); ok && phpIsObj(o) {
				return rt.wrapNum(float64(rt.phpDumpID(o)))
			}
			if rec := phpLitRec(u(a[0])); rec != nil {
				return rt.wrapNum(rec.props["i"].(float64))
			}
			if fr := phpFnRec(u(a[0])); fr != nil {
				return rt.wrapNum(float64(rt.phpFnID(fr)))
			}
			if fk := phpFakeRec(u(a[0])); fk != nil {
				return rt.wrapNum(float64(rt.phpFnID(fk)))
			}
			return rt.wrapNum(0)
		}
		// The precision setting. Twin of __mec_precision in php-interpreter.abnf.
		m["js_phprecision"] = func(a []uint64) uint64 {
			old := phpPrecision
			if !isNullish(u(a[0])) {
				phpPrecision = int(num(u(a[0])))
			}
			return rt.wrapNum(float64(old))
		}
		m["js_phmath"] = func(a []uint64) uint64 {
			x := phpNumOf(rt.phpToNum(u(a[1])))
			y := phpNumOf(rt.phpToNum(u(a[2])))
			return w(phpMkFlo(phpMath(rt.phpStr(u(a[0])), x, y)))
		}
		m["js_phkeyexists"] = func(a []uint64) uint64 {
			keys, _, ok := phpArrParts(u(a[1]))
			if !ok {
				rt.phpRaise("TypeError", "array_key_exists(): Argument #2 ($array) must be of type array, "+
					rt.phpValueName(u(a[1]))+" given")
			}
			return w(rt.phpKeyIndex(keys, u(a[0])) >= 0)
		}
		m["js_phkeylast"] = func(a []uint64) uint64 {
			keys, _, ok := phpArrParts(u(a[0]))
			if !ok || len(keys.elems) == 0 {
				return w(jsNull)
			}
			return w(keys.elems[len(keys.elems)-1])
		}
		// The internal array pointer. Twin of __mec_aptr in php-interpreter.abnf,
		// which explains the model.
		m["js_phaptr"] = func(a []uint64) uint64 {
			op := rt.phpStr(u(a[1]))
			keys, vals, ok := phpArrParts(u(a[0]))
			if !ok {
				if op == "key" {
					return w(jsNull)
				}
				return w(false)
			}
			n := float64(len(keys.elems))
			o, isDict := phpAsArr(u(a[0]))
			p := float64(0)
			if isDict {
				p = phpArrPos(o)
			}
			switch op {
			case "reset":
				p = 0
			case "end":
				p = math.Max(n-1, 0)
			case "next":
				if p < n {
					p++
				}
			case "prev":
				if p < n {
					if p > 0 {
						p--
					} else {
						p = n
					}
				}
			}
			if isDict {
				o.set("__pos", p)
			}
			if op == "key" {
				if p < n {
					return w(keys.elems[int(p)])
				}
				return w(jsNull)
			}
			if p < n {
				return w(phpDeref(vals.elems[int(p)]))
			}
			return w(false)
		}
		// array_pop: the last value, and PHP's next free key steps back when the
		// popped key was the last one handed out.
		m["js_phpop"] = func(a []uint64) uint64 {
			o, ok := phpAsArr(u(a[0]))
			if !ok {
				return w(jsNull)
			}
			keys, _ := o.props["keys"].(*jsArray)
			vals, _ := o.props["vals"].(*jsArray)
			n := len(keys.elems)
			if n == 0 {
				return w(jsNull)
			}
			k, v := keys.elems[n-1], phpDeref(vals.elems[n-1])
			keys.dropIdx()
			vals.dropIdx()
			keys.elems = keys.elems[:n-1]
			vals.elems = vals.elems[:n-1]
			if f, isN := k.(float64); isN && f == phpArrNext(o)-1 {
				o.set("__next", f)
			}
			if _, has := o.props["__pos"]; has {
				o.set("__pos", float64(0))
			}
			return w(v)
		}
	})
}
