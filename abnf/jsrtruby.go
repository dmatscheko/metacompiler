package abnf

// jsrtruby.go -- the Ruby half of Kernel#format that abnf/jsrt.go's rubyFormat
// does not reach: the ARGUMENT SELECTORS, the %a/%A hex-float directive, and
// FloatDomainError on every Float -> Integer conversion.
//
// WHY IT IS A SEPARATE FILE. rubyFormat in abnf/jsrt.go renders one directive.
// What this file adds is not inside any directive - it is the scanner AROUND
// them, which decides WHICH argument a directive gets. So this file is that
// scanner, and it delegates each directive it has resolved back to
// rubyFormat as a single "%<flags><width>.<prec><conv>" with a single argument.
// No directive body is duplicated: %a is the only one rendered here, because it
// is the only one rubyFormat has no arm for.
//
// Registration is additive in the way abnf/jsrtdart.go and abnf/jsrtcsharp.go
// are: an init() appends a registrar to rxExtraExterns, which runs AFTER the base
// table is filled, and it rebinds only the four RUBY-ONLY externs below
// (js_rformat, js_rmod, js_rmcall, js_rxmcall), each of which keeps and calls the
// base binding for everything it does not itself answer. Manual chapter 4's
// warning is about wrapping a SHARED extern, and none of these four is shared.
//
// The twins are rformat / fmtOne / hexBody / floDomainGuard in
// languages/ruby-interpreter.abnf and rbFormat / rbHexBody / rbFloDomain in
// languages/lib/ruby-rt.metajs.

import (
	"math"
	"math/big"
	"strings"
	"unicode/utf16"
)

func init() {
	// FloatDomainError < RangeError, so `rescue FloatDomainError' and
	// `rescue RangeError' both catch what rubyFloDomain raises. rubyExcParent is a
	// plain package var and rubyBuiltinClass makes a class object for any name, so
	// this is the whole registration.
	rubyExcParent["FloatDomainError"] = "RangeError"
	rubyExcParent["Math::DomainError"] = "StandardError"
	rxExtraExterns = append(rxExtraExterns, func(rt *jsrt, m map[string]func(args []uint64) uint64) {
		rt.addRubyFmtExterns(m)
	})
}

func (rt *jsrt) addRubyFmtExterns(m map[string]func(args []uint64) uint64) {
	u := rt.unwrap
	w := rt.wrap

	// Kernel#format / #sprintf.
	m["js_rformat"] = func(a []uint64) uint64 {
		arr, ok := u(a[0]).(*jsArray)
		if !ok || len(arr.elems) == 0 {
			return rt.wrapStr("")
		}
		return rt.wrapStr(rt.rubyFormat2(rt.rubyStr(arr.elems[0]), &jsArray{elems: arr.elems[1:]}))
	}

	m["js_rrespond"] = func(a []uint64) uint64 {
		arr, _ := u(a[1]).(*jsArray)
		if arr == nil {
			arr = &jsArray{}
		}
		return w(rt.rubyRespond(u(a[0]), arr.elems))
	}

	// A bare `.length` / `.size`: a String, Array, Hash, Enumerator or Range counts
	// through js_pylen, anything else (a Symbol, MatchData, a user object) is the
	// method call. Twin of rlen in ruby-interpreter.abnf.
	pylen := m["js_pylen"]
	m["js_rlen"] = func(a []uint64) uint64 {
		v := u(a[0])
		counted := false
		switch v.(type) {
		case string, *jsArray:
			counted = true
		default:
			_, _, isDict := dictParts(v)
			_, isRng := rubyIsRange(v)
			counted = isDict || isRng || rubyIsEnum(v)
		}
		if counted {
			return pylen(a[:1])
		}
		return m["js_rxmcall"]([]uint64{a[0], a[1], w(&jsArray{})})
	}

	// String#% - the same formatter reached through the % operator. Every other
	// left operand is the base binding's modulo.
	baseMod := m["js_rmod"]
	m["js_rmod"] = func(a []uint64) uint64 {
		if s, ok := u(a[0]).(string); ok {
			return rt.wrapStr(rt.rubyFormat2(s, u(a[1])))
		}
		return baseMod(a)
	}

	// Float#to_i and its five relatives. The guard has to sit in front of the
	// dispatcher rather than inside it, because rubyMethod's numeric arm is in
	// abnf/jsrt.go; it fires only for a non-finite Float receiver, so every other
	// call reaches the base binding after one type switch.
	for _, name := range []string{"js_rmcall", "js_rxmcall"} {
		base := m[name]
		m[name] = func(a []uint64) uint64 {
			nm, args := rt.rubyRealCall(rt.toString(u(a[1])), u(a[2]))
			// `Kernel.format(...)' with the module named explicitly does NOT go
			// through js_rformat - rubyMethod answers it off the Kernel builtin -
			// so it needs the same interception, or llvm.Run and the native binary
			// disagree on exactly the selectors this file adds.
			if rubyIsKernelObj(u(a[0])) && (nm == "format" || nm == "sprintf") {
				if arr, ok := args.(*jsArray); ok && len(arr.elems) > 0 {
					return rt.wrapStr(rt.rubyFormat2(rt.rubyStr(arr.elems[0]),
						&jsArray{elems: arr.elems[1:]}))
				}
			}
			if v, done := rubyFloPredicate(u(a[0]), nm); done {
				return w(v)
			}
			rt.rubyFloDomainGuard(u(a[0]), nm, args)
			return base(a)
		}
	}

	// A local read / write the emitter could not prove bound earlier in its frame.
	// A def does not see the locals around it, so the walk stops at the frame, the
	// scope holding "$blk" (the emitter's blkSlot). Past the frame only a callable -
	// a top-level def - still answers a read; anything else is MRI's NameError. A
	// write that finds no binding binds a[3] scopes up from where it stands, past
	// the begin/rescue/ensure closures Ruby has no scope for. Twins: js_rlvar /
	// js_rlset in languages/lib/ruby-rt.metajs.
	m["js_rlvar"] = func(a []uint64) uint64 {
		name := rt.toString(u(a[1]))
		inFrame := true
		for sc := rt.scopeOf(a[0]); sc != nil; sc = sc.parent {
			if v, ok := sc.get(name); ok {
				if inFrame || isCallable(v) {
					return w(v)
				}
				break
			}
			if sc.has("$blk") {
				inFrame = false
			}
		}
		self := rubyScopeSelf(rt, a[0])
		rt.rubyNameErr("NameError", "undefined local variable or method `"+name+"' for "+rt.rubyNoMethodDesc(self), name, self)
		return 0
	}
	// The callee of a bare `name(args)` that names no def: the chain walk of
	// js_scope_get, with a miss raising MRI's NoMethodError on the implicit self.
	// Twin of js_rfn in languages/lib/ruby-rt.metajs.
	m["js_rfn"] = func(a []uint64) uint64 {
		name := rt.toString(u(a[1]))
		for sc := rt.scopeOf(a[0]); sc != nil; sc = sc.parent {
			if v, ok := sc.get(name); ok {
				return w(v)
			}
		}
		rt.rubyNoMethod(rubyScopeSelf(rt, a[0]), name)
		return 0
	}
	// An @ivar write where self is a class object. Twin of js_rcivset in
	// languages/lib/ruby-rt.metajs.
	m["js_rcivset"] = func(a []uint64) uint64 {
		if o, ok := u(a[0]).(*jsObject); ok {
			rubyCivSet(o, rt.toString(u(a[1])), u(a[2]))
		}
		return 0
	}
	m["js_rlset"] = func(a []uint64) uint64 {
		name := rt.toString(u(a[1]))
		for sc := rt.scopeOf(a[0]); sc != nil; sc = sc.parent {
			if sc.has(name) {
				sc.put(name, u(a[2]))
				return 0
			}
			if sc.has("$blk") {
				break
			}
		}
		home := rt.scopeOf(a[0])
		for d := int(rt.toNumber(u(a[3]))); d > 0 && home.parent != nil; d-- {
			home = home.parent
		}
		home.put(name, u(a[2]))
		return 0
	}
}

func rubyIsKernelObj(v interface{}) bool {
	o, ok := v.(*jsObject)
	if !ok {
		return false
	}
	bn, isBuiltin := o.props["__rbuiltin"].(string)
	return isBuiltin && bn == "Kernel"
}

// rubyRealCall unwraps #send / #__send__ / #public_send, which re-enter the
// dispatcher with the real method name one argument along. The two hooks below
// sit OUTSIDE the dispatcher, so they have to do that themselves.
func (rt *jsrt) rubyRealCall(name string, args interface{}) (string, interface{}) {
	arr, _ := args.(*jsArray)
	for (name == "send" || name == "__send__" || name == "public_send") && arr != nil && len(arr.elems) > 0 {
		name = rt.rubyStr(arr.elems[0])
		arr = &jsArray{elems: arr.elems[1:]}
		args = arr
	}
	return name, args
}

// Float#nan? / #infinite? / #finite?. #infinite? answers 1, -1 or NIL and not a
// boolean. The twins are numberMethod in languages/ruby-interpreter.abnf and
// rbNumMethod in languages/lib/ruby-rt.metajs.
// Integer#fdiv, exact past the double range: the quotient is formed as an integer
// of about 70 bits, rounded once and scaled by a power of two. Twin of intFdiv in
// languages/ruby-interpreter.abnf (the reasoning is there) and rbIntFdiv in
// languages/lib/ruby-rt.metajs; the digit counts choose the same scale in all three.
func rubyIntFdiv(a, b interface{}) float64 {
	x, y := rubyToF(a), rubyToF(b)
	bi, isInt := rubyBigOf(b)
	if (!math.IsInf(x, 0) && !math.IsInf(y, 0)) || !isInt || y == 0 {
		return x / y
	}
	ai, _ := rubyBigOf(a)
	pa, pb := new(big.Int).Abs(ai), new(big.Int).Abs(bi)
	s := int(math.Ceil(float64(len(pb.String())-len(pa.String()))*3.33)) + 70
	q := new(big.Int)
	if s >= 0 {
		q.Quo(new(big.Int).Lsh(pa, uint(s)), pb)
	} else {
		q.Quo(pa, new(big.Int).Lsh(pb, uint(-s)))
	}
	r, _ := new(big.Float).SetInt(q).Float64()
	// Halving is exact while the result stays normal; the rest is one multiply by
	// an exact power of two, so a subnormal answer rounds once.
	k := s
	for ; k > 0 && r/2 >= 0x1p-1022; k-- {
		r = r / 2
	}
	if k > 0 {
		pk := 1.0
		for ; k > 0; k-- {
			pk = pk / 2
		}
		r = r * pk
	}
	for ; k < 0; k++ {
		r = r * 2
	}
	if (ai.Sign() < 0) != (bi.Sign() < 0) {
		return -r
	}
	return r
}

// rubyFloPredicate is #nan? / #finite? / #infinite?. Only a Float has #nan?; an
// Integer and a Rational are always finite, and a Complex is finite when both of
// its parts are. #infinite? answers 1, -1 or nil. Twins: numberMethod in
// ruby-interpreter.abnf and rbNumMethod in lib/ruby-rt.metajs.
func rubyFloPredicate(target interface{}, name string) (interface{}, bool) {
	if name != "nan?" && name != "infinite?" && name != "finite?" {
		return nil, false
	}
	var x float64
	switch t := target.(type) {
	case jsFlo:
		x = t.f
	case float64, *jsBigInt, jsRat:
		if name == "nan?" {
			return nil, false
		}
	case jsCpx:
		if name == "nan?" {
			return nil, false
		}
		re, okRe := rubyFloPredicate(rubyReOf(t), "infinite?")
		im, okIm := rubyFloPredicate(rubyImOf(t), "infinite?")
		inf := (okRe && !isUndefOrNull(re)) || (okIm && !isUndefOrNull(im))
		if name == "finite?" {
			return !inf, true
		}
		if inf {
			return float64(1), true
		}
		return jsNull, true
	default:
		return nil, false
	}
	switch name {
	case "nan?":
		return math.IsNaN(x), true
	case "finite?":
		return !math.IsNaN(x) && !math.IsInf(x, 0), true
	}
	if math.IsInf(x, 1) {
		return float64(1), true
	}
	if math.IsInf(x, -1) {
		return float64(-1), true
	}
	return jsNull, true
}

// rubyFloDomainGuard raises FloatDomainError when a non-finite Float is asked for
// an Integer. MRI's #round / #floor / #ceil with a POSITIVE ndigits answer a Float
// and do not raise, which is why the no-argument form is the one that is guarded.
func (rt *jsrt) rubyFloDomainGuard(target interface{}, name string, args interface{}) {
	f, ok := target.(jsFlo)
	if !ok {
		return
	}
	if !math.IsNaN(f.f) && !math.IsInf(f.f, 0) {
		return
	}
	switch name {
	case "to_i", "to_int", "truncate":
	// #floor and #ceil take an ndigits too, and a POSITIVE one answers the
	// receiver rather than raising - exactly as #round does
	// (Float::INFINITY.floor(2) is Infinity). The ndigits <= 0 case still
	// raises: rubyNumDigitOp does it, on the value it is about to convert.
	case "floor", "ceil", "round":
		if arr, isArr := args.(*jsArray); isArr && len(arr.elems) > 0 {
			return
		}
	default:
		return
	}
	rubyFloDomain(rt, f.f)
}

// rubyFloDomain is the raise itself, with the float's own #to_s as the message.
func rubyFloDomain(rt *jsrt, x float64) {
	if math.IsNaN(x) {
		rt.rubyRaiseErr("FloatDomainError", "NaN")
	}
	if math.IsInf(x, 1) {
		rt.rubyRaiseErr("FloatDomainError", "Infinity")
	}
	if math.IsInf(x, -1) {
		rt.rubyRaiseErr("FloatDomainError", "-Infinity")
	}
}

// ===== the argument selectors =====
//
// A directive does not have to take the next argument, and MRI offers four ways
// of saying which one it takes (`"%1$s-%2$s" % ["a","b"]' is "a-b"):
//
//	%N$s          the Nth argument, 1-based, in any order and reusable
//	%<name>s      the Hash entry :name, formatted by the directive
//	%{name}       the Hash entry :name, to_s, with NO conversion letter
//	%*d / %.*f    the width / precision comes from an argument; a NEGATIVE *
//	              width left-justifies, a negative * precision means none
//
// The `*` argument is read BEFORE the value and width before precision, and `*`
// takes an explicit selector too ("%1$*2$d" % [7, 6] is "     7").
//
// "TOO FEW ARGUMENTS" IS RAISED ("%s %s" % ["a"]); the full row table, and the
// shapes that do NOT raise, are at fmtTooFew in languages/ruby-interpreter.abnf.
// rbFmtTooFew in languages/lib/ruby-rt.metajs is the other twin. Not
// implemented: MRI's four ArgumentErrors for MIXING the numbered, unnumbered and
// named styles.
// rubyFmtCheckConv is MRI's malformed-format-string rule (ruby 2.6.10; sprintf
// is unchanged in 3.x):
//
//	"%z" % 5     MRI: malformed format string - %z
//	"%5" % 5     MRI: malformed format string - %*[0-9]
//	"100%" % []  MRI: incomplete format specifier…
//
// The messages are MRI's own, chosen by what the head consumed last - see the
// table in languages/ruby-interpreter.abnf.
//
// The one row a selector still saves in MRI and not here is "%1$%" % 5, which is
// "%" there and *invalid format character - %* here; it never reaches the
// argument list, so it is a malformed-head row.
func rubyIsFmtConv(c byte) bool {
	switch c {
	case 'd', 'i', 'u', 'f', 's', 'p', 'c', 'x', 'X', 'o', 'b', 'B', 'e', 'E', 'g', 'G', 'a', 'A':
		return true
	}
	return false
}

func (rt *jsrt) rubyFmtTooFew() {
	rt.rubyRaiseErr("ArgumentError", "too few arguments")
}

func (rt *jsrt) rubyFmtCheckConv(c byte) {
	if rubyIsFmtConv(c) {
		return
	}
	// % as the conversion of a non-empty head ("%5%"), and a newline or a NUL that
	// is NOT directly after the % ("%5\n"), are MRI's one message.
	if c == '%' || c == '\n' || c == 0 {
		rt.rubyRaiseErr("ArgumentError", "invalid format character - %")
	}
	if c >= 32 && c < 127 {
		rt.rubyRaiseErr("ArgumentError", "malformed format string - %"+string(c))
	}
	rt.rubyRaiseErr("ArgumentError", "malformed format string")
}

func (rt *jsrt) rubyFormat2(spec string, arg interface{}) string {
	args := []interface{}{arg}
	if a, ok := arg.(*jsArray); ok {
		args = a.elems
	}
	ai := 0
	// The next positional argument. Running off the end is MRI's ArgumentError,
	// not a nil - see rubyFmtTooFew's note above.
	take := func() interface{} {
		if ai >= len(args) {
			rt.rubyFmtTooFew()
		}
		v := args[ai]
		ai++
		return v
	}
	// The value a `*` reads: the next positional argument, or the explicit one of
	// a "*2$" form. Answers the integer and the position just past what it ate.
	star := func(j int) (int, int) {
		ds, dn := j, 0
		for j < len(spec) && spec[j] >= '0' && spec[j] <= '9' {
			dn = dn*10 + int(spec[j]-'0')
			j++
		}
		var v interface{} = jsNull
		if j > ds && j < len(spec) && spec[j] == '$' {
			j++
			if dn == 0 || dn-1 >= len(args) {
				rt.rubyFmtTooFew()
			}
			v = args[dn-1]
		} else {
			j = ds
			v = take()
		}
		return int(math.Trunc(rubyToF(v))), j
	}
	var out strings.Builder
	for i := 0; i < len(spec); {
		if spec[i] != '%' {
			out.WriteByte(spec[i])
			i++
			continue
		}
		// A MALFORMED directive is an ArgumentError in MRI. rubyFmtCheckConv
		// carries the rule.
		if i+1 >= len(spec) {
			rt.rubyRaiseErr("ArgumentError", "incomplete format specifier; use %% (double %) instead")
		}
		if spec[i+1] == '%' {
			out.WriteString("%")
			i += 2
			continue
		}
		// A newline or a NUL DIRECTLY after the % is the one shape MRI writes out
		// instead of raising - "%\n" is "%\n" - and only there: "%5\n" raises.
		if spec[i+1] == '\n' || spec[i+1] == 0 {
			out.WriteString("%")
			i++
			continue
		}
		j := i + 1
		flags := ""
		width, prec, sel := 0, -1, -1
		name := ""
		haveName, brace, bad := false, false, false
		// What the directive head consumed last, which is what MRI's message at
		// the end of the string is chosen by: 1 flags, 2 a number (a width, a
		// precision or a *), 3 an argument SELECTOR or a <name>.
		tail := 0
		// ONE loop over the whole directive head, because the pieces may come in
		// any order: "% 1$d" and "%1$ d" are both " 5" in MRI.
		for j < len(spec) {
			c := spec[j]
			if c == '-' || c == '0' || c == '+' || c == ' ' || c == '#' {
				flags += string(c)
				j++
				tail = 1
				continue
			}
			if c == '<' || c == '{' {
				close := byte('>')
				if c == '{' {
					close = '}'
				}
				e := j + 1
				for e < len(spec) && spec[e] != close {
					e++
				}
				if e >= len(spec) {
					bad = true
					break
				}
				name, haveName = spec[j+1:e], true
				j = e + 1
				tail = 3
				if c == '{' {
					brace = true
					break
				}
				continue
			}
			if c == '*' {
				n, nj := star(j + 1)
				j = nj
				tail = 2
				if n < 0 {
					flags += "-"
					width = -n
				} else {
					width = n
				}
				continue
			}
			if c == '.' {
				if prec >= 0 {
					rt.rubyRaiseErr("ArgumentError", "precision given twice")
				}
				j++
				tail = 2
				if j < len(spec) && spec[j] == '*' {
					n, nj := star(j + 1)
					j = nj
					prec = n
					if n < 0 {
						prec = -1
					}
					continue
				}
				prec = 0
				for j < len(spec) && spec[j] >= '0' && spec[j] <= '9' {
					prec = prec*10 + int(spec[j]-'0')
					j++
				}
				continue
			}
			if c >= '1' && c <= '9' {
				// A leading 0 was taken as the zero FLAG above, so a run of digits
				// here is a width - unless a "$" makes it an argument selector.
				dn := 0
				for j < len(spec) && spec[j] >= '0' && spec[j] <= '9' {
					dn = dn*10 + int(spec[j]-'0')
					j++
				}
				if j < len(spec) && spec[j] == '$' {
					sel = dn - 1
					j++
					tail = 3
					continue
				}
				width = dn
				tail = 2
				continue
			}
			break
		}
		conv := byte('s')
		if !brace {
			if bad {
				rt.rubyRaiseErr("ArgumentError", "malformed name - unmatched parenthesis")
			}
			if j >= len(spec) {
				if tail == 2 {
					rt.rubyRaiseErr("ArgumentError", "malformed format string - %*[0-9]")
				}
				if tail != 3 {
					rt.rubyRaiseErr("ArgumentError", "invalid format character - %")
				}
				// A selector or a <name> that ran off the end writes a bare "%"
				// in MRI ("%1$" % 5 is "%"), but the name is still LOOKED UP -
				// "%<x>" % 5 is *one hash required*.
				if haveName {
					rt.rubyFmtNamed(args, name, "<", ">")
				}
				out.WriteString("%")
				break
			}
			conv = spec[j]
			j++
		}
		var av interface{} = jsNull
		switch {
		case haveName:
			open, close := "<", ">"
			if brace {
				open, close = "{", "}"
			}
			av = rt.rubyFmtNamed(args, name, open, close)
		case sel >= 0:
			if sel >= len(args) {
				rt.rubyFmtTooFew()
			}
			av = args[sel]
		default:
			av = take()
		}
		// AFTER the argument, because MRI resolves the name first: "%<x>z" % 5 is
		// *one hash required* and not *malformed format string - %z*.
		if !brace {
			rt.rubyFmtCheckConv(conv)
		}
		// An infinity or a NaN reaching an INTEGER directive raises, because MRI's
		// %d/%x/%o/%b convert the operand with to_i (and Go's int64 saturation would
		// be ARCHITECTURE-DEPENDENT - manual 7.9's giFromFloat note).
		switch conv {
		case 'd', 'i', 'u', 'x', 'X', 'o', 'b', 'B':
			// The OPERAND CHECK comes first: MRI raises TypeError on nil before it
			// can ever reach the to_i that raises FloatDomainError, and this
			// pre-pass runs ahead of rubyFormat's own arm. rubyFmtIntOperand is in
			// abnf/jsrt.go beside rubyFormat.
			av = rt.rubyFmtIntOperand(av)
			if _, isBig := av.(*jsBigInt); !isBig {
				rubyFloDomain(rt, math.Trunc(rubyToF(av)))
			}
		}
		if conv == 'a' || conv == 'A' {
			// %a is the one directive rendered in this file, so it needs the operand
			// check here rather than in rubyFormat: "%a" % [nil] is MRI's
			// *can't convert nil into Float*.
			out.WriteString(rubyHexDirective(rt.rubyFmtFloOperand(av), flags, width, prec, conv == 'A'))
			i = j
			continue
		}
		head := "%" + flags
		if width > 0 {
			head += itoaPos(width)
		}
		if prec >= 0 {
			head += "." + itoaPos(prec)
		}
		// One directive, one argument: the argument is WRAPPED in a one-element
		// array, because rubyFormat reads a bare *jsArray as the argument LIST and
		// `"%s" % [[1, 2]]' would otherwise print the inner array's first element.
		out.WriteString(rt.rubyFormat(head+string(conv), &jsArray{elems: []interface{}{av}}))
		i = j
	}
	return out.String()
}

func itoaPos(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// rubyFmtNamed resolves %<name> / %{name} against the one Hash the right operand
// has to be. A SCAN and not rubyDictFind: the key is the Symbol :name, and a
// String key does NOT match - MRI's `"%{x}" % {"x" => 1}' is a KeyError.
func (rt *jsrt) rubyFmtNamed(args []interface{}, name, open, close string) interface{} {
	var h *jsObject
	if len(args) == 1 {
		if o, ok := args[0].(*jsObject); ok {
			if d, isDict := o.props["__dict"].(bool); isDict && d {
				h = o
			}
		}
	}
	if h == nil {
		rt.rubyRaiseErr("ArgumentError", "one hash required")
	}
	keys, _ := h.props["keys"].(*jsArray)
	vals, _ := h.props["vals"].(*jsArray)
	if keys != nil && vals != nil {
		for i, k := range keys.elems {
			if s, ok := k.(jsSym); ok && s.s == name && i < len(vals.elems) {
				return vals.elems[i]
			}
		}
	}
	rt.rubyRaiseErr("KeyError", "key"+open+name+close+" not found")
	return jsNull
}

// ===== %a / %A =====
//
// MRI writes a finite double as [-]0x1.<hex fraction>p<+|-><decimal binary
// exponent>, normalised so the leading digit is 1 for every non-zero value - a
// SUBNORMAL is normalised too ("%a" % 5.0e-324 is "0x1p-1074") - and zero is
// "0x0p+0". Without a precision the fraction's trailing zeros go; with one it is
// rounded half to even, or zero-padded, to exactly that many digits.
//
// Written on the same pure ARITHMETIC as the other two engines rather than on
// math.Frexp or strconv's 'x' format, deliberately: this is the twin the two
// MetaJS halves have to agree with digit for digit, and the cheapest way to be
// sure of that is to run the same algorithm. Scaling a double by a power of two
// is exact as long as the result neither overflows nor falls into the subnormals,
// and both loops are driven so that it never does.
//
// ONE DELIBERATE DIVERGENCE FROM THE ORACLE, and MRI 2.6.10 is the one that is
// wrong: at a precision of EXACTLY 13 - the full digit count, where no rounding
// can be needed - it rounds the last digit to even anyway. "%.13a" % (1.0/3) is
// 0x1.5555555555554p-2 there against the value's own 0x1.5555555555555p-2. C's
// printf, python3's float.hex and MRI's own %a with no precision
// all print the exact digits, and MRI is right again at 12 and at 14.
func rubyHexNorm(x float64) (float64, int) {
	pv, pk := []float64{}, []int{}
	for v, k := 2.0, 1; k <= 512; v, k = v*v, k*2 {
		pv = append(pv, v)
		pk = append(pk, k)
	}
	e := 0
	for i := len(pv) - 1; i >= 0; i-- {
		for x >= pv[i] {
			x /= pv[i]
			e += pk[i]
		}
	}
	for i := len(pv) - 1; i >= 0; i-- {
		for x*pv[i] < 2 {
			x *= pv[i]
			e -= pk[i]
		}
	}
	return x, e
}

// rubyHexBody is the finite-value body WITHOUT the sign - "0x1.8p+0". prec < 0
// means "as many digits as the value needs".
func rubyHexBody(fv float64, prec int, alt, upper bool) string {
	lead, frac, ex := "1", "", 0
	if fv != 0 {
		if fv < 0 {
			fv = -fv
		}
		m, e := rubyHexNorm(fv)
		ex = e
		// The 13 hex digits of m - 1: 13 x 4 is exactly the 52 fraction bits of a
		// double, so nothing is lost (m - 1 is exact and every *16 is a scale by a
		// power of two).
		d := make([]int, 13)
		f := m - 1
		for i := 0; i < 13; i++ {
			f *= 16
			d[i] = int(math.Floor(f))
			f -= float64(d[i])
		}
		n := 13
		if prec >= 0 && prec < 13 {
			up := false
			if d[prec] > 8 {
				up = true
			} else if d[prec] == 8 {
				rest := false
				for q := prec + 1; q < 13; q++ {
					if d[q] != 0 {
						rest = true
					}
				}
				last := 1
				if prec > 0 {
					last = d[prec-1]
				}
				up = rest || last%2 == 1
			}
			n = prec
			if up {
				c := n - 1
				for c >= 0 && d[c] == 15 {
					d[c] = 0
					c--
				}
				if c < 0 {
					ex++
				} else {
					d[c]++
				}
			}
		}
		for k := 0; k < n; k++ {
			frac += string("0123456789abcdef"[d[k]])
		}
		if prec < 0 {
			frac = strings.TrimRight(frac, "0")
		} else {
			for len(frac) < prec {
				frac += "0"
			}
		}
	} else {
		lead = "0"
	}
	if prec > 0 && frac == "" {
		frac = strings.Repeat("0", prec)
	}
	out := "0x" + lead
	if frac != "" {
		out += "." + frac
	} else if alt {
		out += "."
	}
	sign, ae := "+", ex
	if ex < 0 {
		sign, ae = "-", -ex
	}
	out += "p" + sign + itoaPos(ae)
	if upper {
		return strings.ToUpper(out)
	}
	return out
}

// rubyHexDirective is one whole %a/%A field: the body, the sign, the + / space
// flags and the padding. %a is a NUMERIC directive - it takes a sign and honours
// the 0 flag - and the zeros go behind BOTH the sign and the "0x", which is what
// `keep' counts: "%020a" % -0.0 is "-0x00000000000000p+0".
func rubyHexDirective(f float64, flags string, width, prec int, upper bool) string {
	left := strings.Contains(flags, "-")
	zero := strings.Contains(flags, "0")
	plus := strings.Contains(flags, "+")
	space := strings.Contains(flags, " ")
	alt := strings.Contains(flags, "#")
	body, keep, nonFinite := "", 0, false
	if w, bad := rubyFloatWord(f); bad {
		body, nonFinite = w, true
	} else {
		body = rubyHexBody(f, prec, alt, upper)
		keep = 2
		if f < 0 || (f == 0 && math.Signbit(f)) {
			body, keep = "-"+body, 3
		}
	}
	if !strings.HasPrefix(body, "-") {
		if plus {
			body, keep = "+"+body, keep+1
		} else if space {
			body, keep = " "+body, keep+1
		}
	}
	pad := " "
	if zero && !left && !nonFinite {
		pad = "0"
	}
	if nonFinite {
		keep = 0
	}
	for rubyCharCount(body) < width {
		if left {
			body += " "
		} else if pad != " " {
			body = body[:keep] + pad + body[keep:]
		} else {
			body = pad + body
		}
	}
	return body
}

// ===== Numeric#div / #divmod / #modulo / #rationalize =====
//
// Twin of numDivOp in languages/ruby-interpreter.abnf, which states the rules,
// and rbNumDivOp in languages/lib/ruby-rt.metajs.

func (rt *jsrt) rubyNumDivOp(t interface{}, name string, args []interface{}) interface{} {
	y := argAt(args, 0)
	switch name {
	case "rationalize":
		return rt.rubyRationalize(t, args)
	case "modulo":
		return rt.rubyNumBin("%", t, y)
	}
	_, tf := t.(jsFlo)
	_, yf := y.(jsFlo)
	if name == "divmod" && (tf || yf) {
		return rt.rubyFloDivmod(rubyToF(t), rubyToF(y))
	}
	if rubyToF(y) == 0 {
		rt.rubyRaiseErr("ZeroDivisionError", "divided by 0")
	}
	q := rt.rubyNumBin("/", t, y)
	switch qv := q.(type) {
	case jsFlo:
		rubyFloDomain(rt, qv.f)
		q = rt.rubyIntFromF(math.Floor(qv.f))
	case jsRat:
		q = rt.rubyRatDigitOp(qv, 'f', 0)
	}
	if name == "div" {
		return q
	}
	return &jsArray{elems: []interface{}{q, rt.rubyNumBin("-", t, rt.rubyNumBin("*", y, q))}}
}

// rubyFloDivmod is MRI's flodivmod.
func (rt *jsrt) rubyFloDivmod(x, y float64) interface{} {
	if y != y {
		rubyFloDomain(rt, y)
	}
	if y == 0 {
		rt.rubyRaiseErr("ZeroDivisionError", "divided by 0")
	}
	xInf, yInf := math.IsInf(x, 0), math.IsInf(y, 0)
	mod := x
	if !(x == 0 || (yInf && !xInf)) {
		mod = math.Mod(x, y)
	}
	div := x
	if !(xInf && !yInf) {
		if dq := (x - mod) / y; dq < 0 {
			div = -math.Floor(0.5 - dq)
		} else {
			div = math.Floor(dq + 0.5)
		}
	}
	if y*mod < 0 {
		mod += y
		div--
	}
	rubyFloDomain(rt, div)
	return &jsArray{elems: []interface{}{rt.rubyIntFromF(div), jsFlo{f: mod}}}
}

func (rt *jsrt) rubyRationalize(t interface{}, args []interface{}) interface{} {
	switch v := t.(type) {
	case float64, *jsBigInt:
		return rt.rubyAsRat(t)
	case jsRat:
		if len(args) == 0 {
			return v
		}
		if _, ef := args[0].(jsFlo); !ef {
			er := rt.rubyAsRat(args[0])
			if er.n == 0 {
				return v
			}
			vn, vd := rubyRatParts(v)
			en, ed := rubyRatParts(er)
			d := new(big.Int).Mul(vd, ed)
			c := new(big.Int).Mul(new(big.Int).Abs(vn), ed)
			e := new(big.Int).Mul(new(big.Int).Abs(en), vd)
			r, _ := rubyRatSimplest(new(big.Rat).SetFrac(new(big.Int).Sub(c, e), d), new(big.Rat).SetFrac(new(big.Int).Add(c, e), d))
			if v.n < 0 {
				r = rubyRatNeg(r)
			}
			return r
		}
	}
	x := rubyToF(t)
	rubyFloDomain(rt, x)
	ax := math.Abs(x)
	var r jsRat
	ok := true
	switch {
	case len(args) > 0:
		ef := math.Abs(rubyToF(args[0]))
		lo, hi := ax-ef, ax+ef
		if lo == hi {
			return rt.rubyAsRat(t)
		}
		r, ok = rubyFloSimplest(lo, hi)
	case ax == 0:
		return rubyMkRat(0, 1)
	default:
		f, n := ax, 0
		for f < 4503599627370496 {
			f, n = f*2, n-1
		}
		for f >= 9007199254740992 {
			f, n = f/2, n+1
		}
		if n >= 0 {
			return rubyMkRat(x, 1)
		}
		den := new(big.Int).Lsh(big.NewInt(1), uint(1-n))
		twoF := big.NewInt(2 * int64(f))
		r, ok = rubyRatSimplest(new(big.Rat).SetFrac(new(big.Int).Sub(twoF, big.NewInt(1)), den),
			new(big.Rat).SetFrac(new(big.Int).Add(twoF, big.NewInt(1)), den))
	}
	if !ok {
		return rt.rubyAsRat(t)
	}
	if x < 0 {
		r = rubyRatNeg(r)
	}
	return r
}

// ----- Rational parts past 2^53 --------------------------------------------
//
// A Rational's parts are Integers of any size, as in MRI: 10**20 + Rational(1, 3)
// is (300000000000000000001/3). A pair that fits the doubles keeps the plain
// float64 form every other site reads; past 2^53 the exact parts ride along as
// bn / bd, and n / d hold only their nearest doubles. Twins: mkRat / mkRatBig in
// languages/ruby-interpreter.abnf and rbMkRat in languages/lib/ruby-rt.metajs.

var rubyRatLim = new(big.Int).Lsh(big.NewInt(1), 53)

// rubyRatParts is the exact pair, whichever form the Rational has.
func rubyRatParts(r jsRat) (*big.Int, *big.Int) {
	if r.bn != nil {
		return r.bn, r.bd
	}
	return rubyExactInt(r.n), rubyExactInt(r.d)
}

// rubyMkRatB reduces an exact pair, sign on the numerator.
func rubyMkRatB(n0, d0 *big.Int) jsRat {
	n, d := new(big.Int).Set(n0), new(big.Int).Set(d0)
	if d.Sign() < 0 {
		n.Neg(n)
		d.Neg(d)
	}
	if g := new(big.Int).GCD(nil, nil, new(big.Int).Abs(n), d); g.Cmp(big.NewInt(1)) > 0 {
		n.Quo(n, g)
		d.Quo(d, g)
	}
	if n.CmpAbs(rubyRatLim) <= 0 && d.CmpAbs(rubyRatLim) <= 0 {
		return jsRat{n: rubyBigIntToF(n), d: rubyBigIntToF(d)}
	}
	return jsRat{n: rubyBigIntToF(n), d: rubyBigIntToF(d), bn: n, bd: d}
}

func (rt *jsrt) rubyRatArith(op string, p, q jsRat) jsRat {
	pn, pd := rubyRatParts(p)
	qn, qd := rubyRatParts(q)
	mul := func(a, b *big.Int) *big.Int { return new(big.Int).Mul(a, b) }
	switch op {
	case "+":
		return rubyMkRatB(new(big.Int).Add(mul(pn, qd), mul(qn, pd)), mul(pd, qd))
	case "-":
		return rubyMkRatB(new(big.Int).Sub(mul(pn, qd), mul(qn, pd)), mul(pd, qd))
	case "*":
		return rubyMkRatB(mul(pn, qn), mul(pd, qd))
	}
	return rubyMkRatB(mul(pn, qd), mul(pd, qn))
}

func rubyRatNeg(r jsRat) jsRat {
	if r.bn != nil {
		return jsRat{n: -r.n, d: r.d, bn: new(big.Int).Neg(r.bn), bd: r.bd}
	}
	return jsRat{n: -r.n, d: r.d}
}

func rubyRatToF(r jsRat) float64 {
	if r.bn != nil {
		return rubyIntFdiv(&jsBigInt{v: r.bn}, &jsBigInt{v: r.bd})
	}
	return r.n / r.d
}

// rubyRatPart is a numerator or a denominator as a Ruby Integer.
func (rt *jsrt) rubyRatPart(r jsRat, num bool) interface{} {
	if r.bn == nil {
		if num {
			return r.n
		}
		return r.d
	}
	if num {
		return rt.rubyBigNarrow(r.bn)
	}
	return rt.rubyBigNarrow(r.bd)
}

// rubyRatExact compares a Rational with an Integer or a Rational EXACTLY, by cross
// multiplication; ok is false for any other pair.
func (rt *jsrt) rubyRatExact(l, r interface{}) (int, bool) {
	_, lr := l.(jsRat)
	_, rr := r.(jsRat)
	if !lr && !rr {
		return 0, false
	}
	for _, v := range []interface{}{l, r} {
		switch v.(type) {
		case jsRat, float64, *jsBigInt:
		default:
			return 0, false
		}
	}
	pn, pd := rubyRatParts(rt.rubyAsRat(l))
	qn, qd := rubyRatParts(rt.rubyAsRat(r))
	return new(big.Int).Mul(pn, qd).Cmp(new(big.Int).Mul(qn, pd)), true
}

// rubyIntExponent is an Integer exponent, a Rational one over 1 included.
func (rt *jsrt) rubyIntExponent(r interface{}) (*big.Int, bool) {
	if q, ok := r.(jsRat); ok {
		n, d := rubyRatParts(q)
		return n, d.Cmp(big.NewInt(1)) == 0
	}
	switch r.(type) {
	case float64, *jsBigInt:
		return rubyBigOf(r)
	}
	return nil, false
}

// rubyRatPow is a Rational to an Integer power, exactly. Twin of ratPow in
// languages/ruby-interpreter.abnf.
func (rt *jsrt) rubyRatPow(p jsRat, k *big.Int) interface{} {
	n, d := rubyRatParts(p)
	if k.Sign() < 0 {
		if n.Sign() == 0 {
			rt.rubyRaiseErr("ZeroDivisionError", "divided by 0")
		}
		n, d = d, n
		k = new(big.Int).Neg(k)
	}
	return rubyMkRatB(new(big.Int).Exp(n, k, nil), new(big.Int).Exp(d, k, nil))
}

// rubyRatIntOpB is rubyRatIntOp on exact parts (d > 0, so Div is the floor).
func rubyRatIntOpB(n, d *big.Int, mode byte) *big.Int {
	switch mode {
	case 'f':
		return new(big.Int).Div(n, d)
	case 'c':
		return new(big.Int).Neg(new(big.Int).Div(new(big.Int).Neg(n), d))
	case 't':
		if n.Sign() < 0 {
			return rubyRatIntOpB(n, d, 'c')
		}
		return new(big.Int).Div(n, d)
	}
	n2 := new(big.Int).Lsh(n, 1)
	d2 := new(big.Int).Lsh(d, 1)
	if n.Sign() >= 0 {
		return new(big.Int).Div(new(big.Int).Add(n2, d), d2)
	}
	return new(big.Int).Neg(new(big.Int).Div(new(big.Int).Sub(d, n2), d2))
}

// rubyMainObj is MRI's top-level `self`: an Object whose singleton #to_s and
// #inspect answer "main". Twin of mainObj in ruby-interpreter.abnf.
func (rt *jsrt) rubyMainObj() *jsObject {
	o := newJSObject()
	o.set("__class", rt.rubyBuiltinClass("Object"))
	name := &hostFunc{name: "main", fn: func(rt *jsrt, this uint64, a []interface{}) interface{} { return "main" }}
	rubySingPut(o, "to_s", name)
	rubySingPut(o, "inspect", name)
	return o
}

// rubyRespondNames is Object#respond_to? for a value that is not a user object:
// per builtin class, the names that dispatch in all three ruby engines, row "*"
// holding the ones every class answers. Twin of rbRespondRows in
// languages/lib/ruby-respond.js (whose comment says how the list is made);
// ruby_respond_test.go keeps the two equal.
var rubyRespondNames = map[string][]string{
	"*": {"!", "!=", "==", "===", "class", "hash", "inspect", "itself", "object_id",
		"respond_to?", "tap", "to_s"},
	"private": {"format", "lambda", "loop", "p", "print", "printf", "proc", "puts", "raise",
		"rand", "require", "sprintf"},
	"Integer": {"%", "&", "*", "**", "+", "+@", "-", "-@", "/", "<", "<=", "<=>", ">", ">=",
		"[]", "^", "abs", "abs2", "allbits?", "angle", "anybits?", "arg", "bit_length", "ceil",
		"chr", "clone", "coerce", "conj", "conjugate", "denominator", "digits", "div", "divmod",
		"dup", "eql?", "equal?", "even?", "fdiv", "finite?", "floor", "frozen?", "gcd", "gcdlcm",
		"i", "imag", "imaginary", "infinite?", "instance_of?", "instance_variables", "integer?",
		"is_a?", "kind_of?", "lcm", "magnitude", "modulo", "negative?", "next", "nil?", "nobits?",
		"nonzero?", "numerator", "odd?", "ord", "phase", "polar", "positive?", "pow", "pred",
		"public_send", "quo", "rationalize", "real", "real?", "rect", "rectangular", "remainder",
		"round", "send", "size", "succ", "to_c", "to_f", "to_i", "to_int", "to_r", "truncate",
		"zero?", "|"},
	"Float": {"%", "*", "**", "+", "+@", "-", "-@", "/", "<", "<=", "<=>", ">", ">=", "abs",
		"abs2", "angle", "arg", "ceil", "clone", "coerce", "conj", "conjugate", "denominator",
		"div", "divmod", "dup", "eql?", "equal?", "fdiv", "finite?", "floor", "frozen?", "i",
		"imag", "imaginary", "infinite?", "instance_of?", "instance_variables", "integer?",
		"is_a?", "kind_of?", "magnitude", "modulo", "nan?", "negative?", "next_float", "nil?",
		"nonzero?", "numerator", "phase", "polar", "positive?", "prev_float", "public_send",
		"quo", "rationalize", "real", "real?", "rect", "rectangular", "remainder", "round",
		"send", "to_c", "to_f", "to_i", "to_int", "to_r", "truncate", "zero?"},
	"Rational": {"%", "*", "**", "+", "+@", "-", "-@", "/", "<", "<=", "<=>", ">", ">=", "abs",
		"abs2", "angle", "arg", "ceil", "clone", "coerce", "conj", "conjugate", "denominator",
		"div", "divmod", "dup", "eql?", "equal?", "fdiv", "finite?", "floor", "frozen?", "i",
		"imag", "imaginary", "infinite?", "instance_of?", "instance_variables", "integer?",
		"is_a?", "kind_of?", "magnitude", "negative?", "nil?", "nonzero?", "numerator", "phase",
		"polar", "positive?", "public_send", "quo", "rationalize", "real", "real?", "rect",
		"rectangular", "remainder", "round", "send", "to_c", "to_f", "to_i", "to_int", "to_r",
		"truncate", "zero?"},
	"Complex": {"*", "+", "+@", "-", "-@", "/", "abs", "abs2", "angle", "arg", "clone",
		"coerce", "conj", "conjugate", "denominator", "dup", "eql?", "equal?", "fdiv", "finite?",
		"frozen?", "imag", "imaginary", "infinite?", "instance_of?", "instance_variables",
		"integer?", "is_a?", "kind_of?", "magnitude", "nil?", "nonzero?", "numerator", "phase",
		"polar", "public_send", "quo", "rationalize", "real", "real?", "rect", "rectangular",
		"send", "to_c", "to_f", "to_i", "to_int", "to_r", "zero?"},
	"String": {"%", "*", "+", "+@", "<", "<<", "<=", "<=>", ">", ">=", "[]", "ascii_only?",
		"b", "bytes", "bytesize", "capitalize", "casecmp", "casecmp?", "center", "chars", "chomp",
		"chomp!", "chop", "chop!", "chr", "clear", "clone", "codepoints", "concat", "count",
		"delete", "delete!", "delete_prefix", "delete_prefix!", "delete_suffix", "delete_suffix!",
		"downcase", "dump", "dup", "each_byte", "each_char", "each_codepoint", "each_line",
		"empty?", "encode", "end_with?", "equal?", "force_encoding", "freeze", "frozen?",
		"getbyte", "hex", "include?", "index", "insert", "instance_of?", "instance_variables",
		"intern", "is_a?", "kind_of?", "length", "lines", "ljust", "lstrip", "lstrip!", "next",
		"nil?", "oct", "ord", "partition", "prepend", "public_send", "replace", "reverse",
		"reverse!", "rindex", "rjust", "rpartition", "rstrip", "rstrip!", "scrub", "send", "size",
		"slice", "slice!", "split", "squeeze", "squeeze!", "start_with?", "strip", "strip!",
		"succ", "sum", "swapcase", "swapcase!", "to_f", "to_i", "to_str", "to_sym", "tr", "tr!",
		"tr_s", "tr_s!", "unicode_normalize", "unicode_normalized?", "upcase", "upto",
		"valid_encoding?"},
	"Symbol": {"<", "<=", "<=>", ">", ">=", "[]", "capitalize", "downcase", "empty?", "equal?",
		"frozen?", "id2name", "instance_of?", "instance_variables", "is_a?", "kind_of?", "length",
		"name", "next", "nil?", "public_send", "send", "size", "succ", "swapcase", "to_proc",
		"to_sym", "upcase"},
	"Array": {"&", "*", "+", "-", "<<", "<=>", "[]", "all?", "any?", "append", "assoc",
		"clear", "clone", "collect", "collect!", "collect_concat", "combination", "compact",
		"compact!", "concat", "count", "cycle", "delete", "delete_at", "delete_if", "detect",
		"drop", "drop_while", "dup", "each", "each_cons", "each_index", "each_slice",
		"each_with_index", "each_with_object", "empty?", "entries", "equal?", "fetch", "fill",
		"filter", "filter!", "find", "find_index", "first", "flat_map", "flatten", "flatten!",
		"freeze", "frozen?", "group_by", "include?", "index", "inject", "insert", "instance_of?",
		"instance_variables", "is_a?", "itself", "join", "keep_if", "kind_of?", "last", "length",
		"map", "map!", "max", "max_by", "min", "min_by", "minmax", "nil?", "none?", "one?",
		"partition", "permutation", "pop", "prepend", "product", "public_send", "push", "reduce",
		"reject", "reject!", "reverse", "reverse!", "reverse_each", "rindex", "rotate", "sample",
		"select", "select!", "send", "shift", "shuffle", "shuffle!", "size", "slice", "sort",
		"sort!", "sort_by", "sort_by!", "sum", "take", "take_while", "tally", "to_a", "uniq",
		"uniq!", "unshift", "values_at", "zip", "|"},
	"Hash": {"<", "<=", "<=>", ">", ">=", "[]", "all?", "any?", "assoc", "clear", "clone",
		"collect", "collect_concat", "count", "default", "delete", "delete_if", "detect", "drop",
		"drop_while", "dup", "each", "each_cons", "each_key", "each_pair", "each_slice",
		"each_value", "each_with_index", "each_with_object", "empty?", "entries", "equal?",
		"fetch", "filter", "filter!", "find", "find_index", "first", "flat_map", "freeze",
		"frozen?", "group_by", "has_key?", "include?", "inject", "instance_of?",
		"instance_variables", "invert", "is_a?", "itself", "keep_if", "key", "key?", "keys",
		"kind_of?", "length", "map", "max", "max_by", "merge", "merge!", "min", "min_by",
		"minmax", "nil?", "none?", "one?", "partition", "public_send", "reduce", "reject",
		"reject!", "select", "select!", "send", "size", "sort", "sort_by", "sum", "take",
		"take_while", "tally", "to_a", "to_h", "transform_keys", "transform_values", "uniq",
		"update", "values", "values_at", "zip"},
	"Range": {"<=>", "all?", "any?", "begin", "clone", "collect", "collect_concat", "count",
		"cover?", "cycle", "detect", "drop", "drop_while", "dup", "each", "each_cons",
		"each_slice", "each_with_index", "each_with_object", "end", "entries", "eql?", "equal?",
		"exclude_end?", "filter", "find", "find_index", "first", "flat_map", "freeze", "frozen?",
		"group_by", "hash", "include?", "inject", "instance_of?", "instance_variables", "is_a?",
		"itself", "kind_of?", "last", "map", "max", "max_by", "member?", "min", "min_by",
		"minmax", "nil?", "none?", "object_id", "one?", "partition", "public_send", "reduce",
		"reject", "reverse_each", "select", "send", "size", "sort", "sort_by", "step", "sum",
		"take", "take_while", "tally", "to_a", "uniq", "zip"},
	"NilClass": {"&", "<=>", "^", "equal?", "frozen?", "instance_of?", "instance_variables",
		"is_a?", "kind_of?", "nil?", "public_send", "send", "to_a", "to_f", "to_i", "|"},
	"TrueClass": {"&", "<=>", "^", "equal?", "frozen?", "instance_of?", "instance_variables",
		"is_a?", "kind_of?", "nil?", "public_send", "send", "|"},
	"FalseClass": {"&", "<=>", "^", "equal?", "frozen?", "instance_of?", "instance_variables",
		"is_a?", "kind_of?", "nil?", "public_send", "send", "|"},
	"Proc": {"<=>", "[]", "arity", "call", "equal?", "frozen?", "instance_of?",
		"instance_variables", "is_a?", "kind_of?", "lambda?", "nil?", "public_send", "send",
		"to_proc", "yield"},
	"Method": {"<=>", "[]", "arity", "call", "equal?", "frozen?", "instance_of?",
		"instance_variables", "is_a?", "kind_of?", "name", "nil?", "public_send", "send",
		"to_proc"},
	"Regexp": {"<=>", "=~", "equal?", "frozen?", "instance_of?", "instance_variables", "is_a?",
		"kind_of?", "match", "match?", "names", "nil?", "options", "public_send", "send",
		"source"},
	"MatchData": {"<=>", "[]", "begin", "captures", "end", "equal?", "frozen?", "instance_of?",
		"instance_variables", "is_a?", "kind_of?", "length", "named_captures", "names", "nil?",
		"post_match", "pre_match", "public_send", "send", "size", "to_a"},
	"Enumerator": {"<=>", "all?", "any?", "clone", "collect", "collect_concat", "count",
		"cycle", "detect", "drop", "drop_while", "dup", "each", "each_cons", "each_slice",
		"each_with_index", "each_with_object", "entries", "equal?", "filter", "find",
		"find_index", "first", "flat_map", "freeze", "frozen?", "group_by", "include?", "inject",
		"instance_of?", "instance_variables", "is_a?", "itself", "kind_of?", "map", "max",
		"max_by", "min", "min_by", "minmax", "next", "nil?", "none?", "one?", "partition", "peek",
		"public_send", "reduce", "reject", "reverse_each", "rewind", "select", "send", "size",
		"sort", "sort_by", "sum", "take", "take_while", "tally", "to_a", "uniq", "with_index",
		"with_object", "zip"},
}

var rubyRespondSet = func() map[string]bool {
	out := map[string]bool{}
	for cls, names := range rubyRespondNames {
		for _, n := range names {
			out[cls+"#"+n] = true
		}
	}
	return out
}()

// rubyResponds answers respond_to? for a value of builtin class cls.
func rubyResponds(cls, name string) bool {
	return rubyRespondSet["*#"+name] || rubyRespondSet[cls+"#"+name]
}

// rubyRespond is `x.respond_to?(n, include_all)` (js_rrespond): a builtin answers
// from the table, a user instance or class from its own tables. A private or
// protected method counts only with include_all, and so do the private methods
// every object has: the "private" row and the top-level defs, which are private
// methods of Object recorded as "$p$" slots on the Object class object. Twin of
// js_rrespond in lib/ruby-rt.metajs and respondTo in ruby-interpreter.abnf.
func (rt *jsrt) rubyRespond(target interface{}, args []interface{}) interface{} {
	pos, _ := rubyBlockArgs(args)
	n := rt.rubyStr(argAt(pos, 0))
	all := rubyTruthy(argAt(pos, 1))
	if cls := rt.rubyRespondClass(target); cls != "" {
		return rubyResponds(cls, n) || (all && rt.rubyRespondsPrivate(n))
	}
	o, isObj := target.(*jsObject)
	if !isObj {
		return rt.rubyMethod(target, "respond_to?", args)
	}
	if rubyResponds("", n) {
		return true
	}
	if _, ok := rubySingFind(o, n); ok {
		return true
	}
	if rubyRestricted(o, n) {
		return all
	}
	return rubySingRespond(o, n) || (all && rt.rubyRespondsPrivate(n))
}

// rubyRespondsPrivate: is n a private method every object has?
func (rt *jsrt) rubyRespondsPrivate(n string) bool {
	if rubyRespondSet["private#"+n] {
		return true
	}
	_, top := rt.rubyBuiltinClass("Object").props["$p$"+n]
	return top
}

// rubyRestricted: is n a private or protected method of an instance's class chain?
// The emitter keeps such a method under "$q$" + n (methodKey in
// ruby-to-llvm-ir.abnf), and a nearer class holding only the plain name overrides
// it as a public one.
func rubyRestricted(o *jsObject, n string) bool {
	for cls := o.props["__class"]; cls != nil; {
		clsObj, ok := cls.(*jsObject)
		if !ok {
			break
		}
		if _, has := clsObj.props["$q$"+n]; has {
			return true
		}
		if _, has := clsObj.props[n]; has {
			return false
		}
		cls = clsObj.props["__super"]
	}
	return false
}

// rubyRespondClass is the builtin class respond_to? looks up, "" for a user
// instance or class (or anything else that answers from its own tables).
func (rt *jsrt) rubyRespondClass(v interface{}) string {
	if rxIsRegexObj(v) {
		return "Regexp"
	}
	if rxIsMatchObj(v) {
		return "MatchData"
	}
	if o, isObj := v.(*jsObject); isObj {
		if _, isCls := o.props["__isclass"]; isCls || rubyUserObj(o) {
			return ""
		}
	}
	return rt.rubyClassNameRT(v)
}

// rubyIvarAccess is Object#instance_variable_get / _set on an instance: an @ivar
// is the slot of its name without the "@". Twin of ivarAccess in
// ruby-interpreter.abnf.
func (rt *jsrt) rubyIvarAccess(o *jsObject, name string, args []interface{}) interface{} {
	pos, _ := rubyBlockArgs(args)
	n := rt.rubyStr(argAt(pos, 0))
	if !rubyIvarNameOk(n) {
		rt.rubyRaiseErr("NameError", "`"+n+"' is not allowed as an instance variable name")
	}
	if name == "instance_variable_set" {
		v := argAt(pos, 1)
		if rubyClassObj(o) {
			rubyCivSet(o, n[1:], v)
		} else {
			o.set(n[1:], v)
		}
		return v
	}
	if v, has := o.props[n[1:]]; has {
		return v
	}
	return jsNull
}

// rubyIvarNames is Object#instance_variables on an instance: its @ivar slots in
// assignment order, the hidden __ slots left out. Twin of ivarNames in
// ruby-interpreter.abnf and rbIvarNames in lib/ruby-rt.metajs.
func rubyIvarNames(o *jsObject) *jsArray {
	out := &jsArray{}
	if rubyClassObj(o) {
		if l, ok := o.props["__civn"].(*jsArray); ok {
			for _, k := range l.elems {
				name, _ := k.(string)
				out.elems = append(out.elems, jsSym{s: "@" + name})
			}
		}
		return out
	}
	for _, k := range o.keys {
		if len(k) >= 2 && k[0] == '_' && k[1] == '_' {
			continue
		}
		out.elems = append(out.elems, jsSym{s: "@" + k})
	}
	return out
}

// rubyIvarNameOk: "@" and then an identifier (a letter, "_" or a non-ASCII
// character first, digits allowed after it).
func rubyIvarNameOk(n string) bool {
	if len(n) < 2 || n[0] != '@' {
		return false
	}
	for i := 1; i < len(n); i++ {
		c := n[i]
		ok := c == '_' || c >= 128 || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (i > 1 && c >= '0' && c <= '9')
		if !ok {
			return false
		}
	}
	return true
}

// rubyIntBit is Integer#[]: bit i of the infinite two's complement, 0 for a
// negative i. Twin of intBitAt in ruby-interpreter.abnf.
func (rt *jsrt) rubyIntBit(n, i interface{}) interface{} {
	if rubyToF(i) < 0 {
		return 0.0
	}
	return rt.rubyBin("&", rt.rubyBin(">>", n, i), 1.0)
}

func rubyExactInt(x float64) *big.Int {
	b, _ := big.NewFloat(x).Int(nil)
	return b
}

func rubyBigIntToF(x *big.Int) float64 {
	f, _ := new(big.Float).SetInt(x).Float64()
	return f
}

// rubyRatSimplest is MRI's nurat_rationalize_internal over exact bounds a < b;
// the answer is exact (ok is always true).
func rubyRatSimplest(a, b *big.Rat) (jsRat, bool) {
	p0, p1, q0, q1 := big.NewInt(0), big.NewInt(1), big.NewInt(1), big.NewInt(0)
	var c *big.Int
	for {
		c = new(big.Int).Neg(new(big.Int).Div(new(big.Int).Neg(a.Num()), a.Denom()))
		if new(big.Rat).SetInt(c).Cmp(b) < 0 {
			break
		}
		k := new(big.Int).Sub(c, big.NewInt(1))
		p2 := new(big.Int).Add(new(big.Int).Mul(k, p1), p0)
		q2 := new(big.Int).Add(new(big.Int).Mul(k, q1), q0)
		kr := new(big.Rat).SetInt(k)
		t := new(big.Rat).Inv(new(big.Rat).Sub(b, kr))
		b = new(big.Rat).Inv(new(big.Rat).Sub(a, kr))
		a = t
		p0, q0, p1, q1 = p1, q1, p2, q2
	}
	p := new(big.Int).Add(new(big.Int).Mul(c, p1), p0)
	q := new(big.Int).Add(new(big.Int).Mul(c, q1), q0)
	return rubyMkRatB(p, q), true
}

func rubyFiniteRat(p, q float64) (jsRat, bool) {
	if math.IsInf(p, 0) || math.IsInf(q, 0) {
		return jsRat{}, false
	}
	return rubyMkRat(p, q), true
}

// rubyFloSimplest is the same walk in doubles, as MRI does once eps made the
// bounds Floats. float64() on each product stops arm64 fusing it into an FMA
// (manual chapter 4), which the other two engines cannot do.
func rubyFloSimplest(a, b float64) (jsRat, bool) {
	p0, p1, q0, q1 := 0.0, 1.0, 1.0, 0.0
	c := 0.0
	for {
		c = math.Ceil(a)
		if c < b {
			break
		}
		k := c - 1
		p2 := float64(k*p1) + p0
		q2 := float64(k*q1) + q0
		t := 1 / (b - k)
		b = 1 / (a - k)
		a = t
		p0, q0, p1, q1 = p1, q1, p2, q2
	}
	return rubyFiniteRat(float64(c*p1)+p0, float64(c*q1)+q0)
}

// rubyCpxAbs is MRI's Complex#abs: with one part zero it is the other part's
// own abs, a Float only if either part is one. Twin of cpxAbs in
// languages/ruby-interpreter.abnf.
func (rt *jsrt) rubyCpxAbs(t jsCpx) interface{} {
	_, reF := t.re.(jsFlo)
	_, imF := t.im.(jsFlo)
	if rt.rubyEq(t.re, 0.0) {
		a := rt.rubyRealAbs(t.im)
		if reF && !imF {
			return jsFlo{f: rubyToF(a)}
		}
		return a
	}
	if rt.rubyEq(t.im, 0.0) {
		a := rt.rubyRealAbs(t.re)
		if !reF && imF {
			return jsFlo{f: rubyToF(a)}
		}
		return a
	}
	return jsFlo{f: rubyHypot(rubyToF(t.re), rubyToF(t.im))}
}

// rubyCpxMethod is Complex's own surface. Twin of cpxMethod in
// languages/ruby-interpreter.abnf, which states the rules.
func (rt *jsrt) rubyCpxMethod(t jsCpx, name string) (interface{}, bool) {
	switch name {
	case "conjugate", "conj":
		return rt.rubyCpxConj(t), true
	case "rect", "rectangular":
		return &jsArray{elems: []interface{}{t.re, t.im}}, true
	case "abs", "magnitude":
		return rt.rubyCpxAbs(t), true
	case "polar":
		return &jsArray{elems: []interface{}{rt.rubyCpxAbs(t), jsFlo{f: math.Atan2(rubyToF(t.im), rubyToF(t.re))}}}, true
	case "arg", "angle", "phase":
		return jsFlo{f: math.Atan2(rubyToF(t.im), rubyToF(t.re))}, true
	case "zero?":
		return rt.rubyEq(t.re, 0.0) && rt.rubyEq(t.im, 0.0), true
	case "real?":
		return false, true
	case "round", "floor", "ceil", "truncate", "positive?", "negative?", "div", "divmod", "modulo", "remainder":
		rt.rubyNoMethod(t, name)
	}
	into := map[string]string{"to_i": "Integer", "to_int": "Integer", "to_f": "Float", "to_r": "Rational", "rationalize": "Rational"}[name]
	if into != "" {
		if _, imF := t.im.(jsFlo); imF || !rt.rubyEq(t.im, 0.0) {
			rt.rubyRaiseErr("RangeError", "can't convert "+rt.rubyStr(t)+" into "+into)
		}
		return rt.rubyMethod(t.re, name, nil), true
	}
	return nil, false
}

// rubyCpxBin is Complex arithmetic on the TYPED parts. Twin of cpxBin in
// languages/ruby-interpreter.abnf, which states the rules.
func (rt *jsrt) rubyCpxBin(op string, l, r interface{}) interface{} {
	rc, rIsCpx := r.(jsCpx)
	if !rIsCpx {
		lc := l.(jsCpx)
		switch op {
		case "+":
			return jsCpx{re: rt.rubyNumBin("+", lc.re, r), im: lc.im}
		case "-":
			return jsCpx{re: rt.rubyNumBin("-", lc.re, r), im: lc.im}
		case "*":
			return jsCpx{re: rt.rubyNumBin("*", lc.re, r), im: rt.rubyNumBin("*", lc.im, r)}
		case "/":
			return jsCpx{re: rt.rubyRealQuo(lc.re, r), im: rt.rubyRealQuo(lc.im, r)}
		}
		rt.fail("Complex does not support '%s'", op)
	}
	a, aIsCpx := l.(jsCpx)
	if !aIsCpx {
		a = jsCpx{re: l, im: 0.0}
	}
	switch op {
	case "+":
		return jsCpx{re: rt.rubyNumBin("+", a.re, rc.re), im: rt.rubyNumBin("+", a.im, rc.im)}
	case "-":
		return jsCpx{re: rt.rubyNumBin("-", a.re, rc.re), im: rt.rubyNumBin("-", a.im, rc.im)}
	case "*":
		return rt.rubyCpxMul(a, rc)
	case "/":
		nb := rt.rubyNumBin
		q := rt.rubyRealQuo
		var x, y interface{}
		if c, _ := rt.rubySpaceship(rt.rubyRealAbs(rc.re), rt.rubyRealAbs(rc.im)); c > 0 {
			q1 := q(rc.im, rc.re)
			n1 := nb("*", rc.re, nb("+", 1.0, nb("*", q1, q1)))
			x = nb("+", q(a.re, n1), q(nb("*", a.im, q1), n1))
			y = nb("-", q(a.im, n1), q(nb("*", a.re, q1), n1))
		} else {
			q2 := q(rc.re, rc.im)
			n2 := nb("*", rc.im, nb("+", 1.0, nb("*", q2, q2)))
			x = nb("+", q(nb("*", a.re, q2), n2), q(a.im, n2))
			y = nb("-", q(nb("*", a.im, q2), n2), q(a.re, n2))
		}
		for _, p := range []interface{}{a.re, a.im, rc.re, rc.im} {
			if _, f := p.(jsFlo); f {
				return jsCpx{re: x, im: y}
			}
		}
		return jsCpx{re: rubyRatCanon(x), im: rubyRatCanon(y)}
	}
	rt.fail("Complex does not support '%s'", op)
	return nil
}

func rubyRatCanon(v interface{}) interface{} {
	if r, ok := v.(jsRat); ok && r.d == 1 {
		if r.bn != nil {
			return &jsBigInt{v: r.bn}
		}
		return r.n
	}
	return v
}

func (rt *jsrt) rubyCpxMul(a, b jsCpx) jsCpx {
	nb := rt.rubyNumBin
	return jsCpx{re: nb("-", nb("*", a.re, b.re), nb("*", a.im, b.im)),
		im: nb("+", nb("*", a.re, b.im), nb("*", a.im, b.re))}
}

func (rt *jsrt) rubyCpxConj(c jsCpx) jsCpx { return jsCpx{re: c.re, im: rt.rubyNeg(c.im)} }

// Numeric#quo on two reals. Twin of realQuo in languages/ruby-interpreter.abnf.
func (rt *jsrt) rubyRealQuo(a, b interface{}) interface{} {
	_, af := a.(jsFlo)
	_, bf := b.(jsFlo)
	if af || bf {
		return rt.rubyNumBin("/", a, b)
	}
	if rt.rubyEq(b, 0.0) {
		rt.rubyRaiseErr("ZeroDivisionError", "divided by 0")
	}
	return rt.rubyNumBin("/", rt.rubyAsRat(a), b)
}

func (rt *jsrt) rubyRealAbs(v interface{}) interface{} {
	if f, ok := v.(jsFlo); ok {
		return jsFlo{f: math.Abs(f.f)}
	}
	if c, _ := rt.rubySpaceship(v, 0.0); c < 0 {
		return rt.rubyNeg(v)
	}
	return v
}

// rubyCpxText is Complex#to_s (insp false) and the inside of #inspect: MRI's
// f_format. Twin of cpxText in languages/ruby-interpreter.abnf.
func (rt *jsrt) rubyCpxText(t jsCpx, insp bool) string {
	render := rt.rubyStr
	if insp {
		render = rt.rubyInspect
	}
	pos := true
	if f, ok := t.im.(jsFlo); ok {
		pos = !math.Signbit(f.f)
	} else if c, _ := rt.rubySpaceship(t.im, 0.0); c < 0 {
		pos = false
	}
	sign := "+"
	if !pos {
		sign = "-"
	}
	it := render(rt.rubyRealAbs(t.im))
	out := render(t.re) + sign + it
	if lc := it[len(it)-1]; lc < '0' || lc > '9' {
		out += "*"
	}
	return out + "i"
}

// rubyNumMore is Numeric's remaining public methods for an Integer, a Float and a
// Rational, reached once rubyNumMethod has no arm of its own. Twin of numMore in
// languages/ruby-interpreter.abnf, which states the rules.
func (rt *jsrt) rubyNumMore(t interface{}, name string, args []interface{}) (interface{}, bool) {
	pos, _ := rubyBlockArgs(args)
	a0 := argAt(pos, 0)
	_, isF := t.(jsFlo)
	ti, isI := rubyIntBig(t)
	sign := func(v interface{}) int { c, _ := rt.rubySpaceship(v, 0.0); return c }
	switch name {
	case "dup", "clone":
		return t, true
	case "nonzero?":
		if rt.rubyEq(t, 0.0) {
			return jsNull, true
		}
		return t, true
	}
	if c, isCpx := t.(jsCpx); isCpx {
		_, cd := a0.(jsCpx)
		switch name {
		case "abs2":
			return rt.rubyNumBin("+", rt.rubyNumBin("*", c.re, c.re), rt.rubyNumBin("*", c.im, c.im)), true
		case "coerce":
			if cd {
				return &jsArray{elems: []interface{}{a0, t}}, true
			}
			return &jsArray{elems: []interface{}{jsCpx{re: a0, im: 0.0}, t}}, true
		case "quo":
			if !cd {
				re, _ := rt.rubyNumMore(c.re, "quo", pos)
				im, _ := rt.rubyNumMore(c.im, "quo", pos)
				return jsCpx{re: re, im: im}, true
			}
		case "fdiv":
			if !cd {
				return jsCpx{re: jsFlo{f: rubyToF(c.re) / rubyToF(a0)}, im: jsFlo{f: rubyToF(c.im) / rubyToF(a0)}}, true
			}
		}
		return nil, false
	}
	switch name {
	case "abs2":
		return rt.rubyNumBin("*", t, t), true
	case "conj", "conjugate":
		return t, true
	case "real?":
		return true, true
	case "rect", "rectangular":
		return &jsArray{elems: []interface{}{t, 0.0}}, true
	case "i":
		return jsCpx{re: 0.0, im: t}, true
	case "arg", "angle", "phase", "polar":
		var ar interface{} = 0.0
		if f, ok := t.(jsFlo); ok {
			if f.f != f.f {
				ar = t
			} else if math.Signbit(f.f) {
				ar = jsFlo{f: math.Pi}
			}
		} else if sign(t) < 0 {
			ar = jsFlo{f: math.Pi}
		}
		if name == "polar" {
			return &jsArray{elems: []interface{}{rt.rubyMethod(t, "abs", nil), ar}}, true
		}
		return ar, true
	case "coerce":
		_, coI := rubyIntBig(a0)
		if isI {
			if coI {
				return &jsArray{elems: []interface{}{a0, t}}, true
			}
			return &jsArray{elems: []interface{}{jsFlo{f: rubyToF(a0)}, jsFlo{f: rubyToF(t)}}}, true
		}
		if _, isR := t.(jsRat); isR {
			if coI {
				return &jsArray{elems: []interface{}{rt.rubyAsRat(a0), t}}, true
			}
			if _, coF := a0.(jsFlo); coF {
				return &jsArray{elems: []interface{}{a0, jsFlo{f: rubyToF(t)}}}, true
			}
			return &jsArray{elems: []interface{}{a0, t}}, true
		}
		return &jsArray{elems: []interface{}{jsFlo{f: rubyToF(a0)}, t}}, true
	case "quo":
		if _, qF := a0.(jsFlo); isF || qF {
			return rt.rubyNumBin("/", jsFlo{f: rubyToF(t)}, a0), true
		}
		if rt.rubyEq(a0, 0.0) {
			rt.rubyRaiseErr("ZeroDivisionError", "divided by 0")
		}
		return rt.rubyNumBin("/", rt.rubyAsRat(t), rt.rubyAsRat(a0)), true
	case "remainder":
		m := rt.rubyNumBin("%", t, a0)
		if !rt.rubyEq(m, 0.0) && (sign(t) < 0) != (sign(a0) < 0) {
			return rt.rubyNumBin("-", m, a0), true
		}
		return m, true
	}
	if f, ok := t.(jsFlo); ok {
		switch name {
		case "next_float":
			return jsFlo{f: math.Nextafter(f.f, math.Inf(1))}, true
		case "prev_float":
			return jsFlo{f: math.Nextafter(f.f, math.Inf(-1))}, true
		}
		return nil, false
	}
	if !isI {
		return nil, false
	}
	narrow := rt.rubyBigNarrow
	switch name {
	case "ord":
		return t, true
	case "gcd", "lcm", "gcdlcm":
		y, yI := rubyIntBig(a0)
		if !yI {
			rt.rubyRaiseErr("TypeError", "not an integer")
		}
		g := new(big.Int).GCD(nil, nil, new(big.Int).Abs(ti), new(big.Int).Abs(y))
		if name == "gcd" {
			return narrow(g), true
		}
		l := new(big.Int)
		if g.Sign() != 0 {
			l.Abs(l.Mul(new(big.Int).Quo(ti, g), y))
		}
		if name == "lcm" {
			return narrow(l), true
		}
		return &jsArray{elems: []interface{}{narrow(g), narrow(l)}}, true
	case "allbits?", "anybits?", "nobits?":
		m, _ := rubyIntBig(a0)
		and := new(big.Int).And(ti, m)
		switch name {
		case "allbits?":
			return and.Cmp(m) == 0, true
		case "anybits?":
			return and.Sign() != 0, true
		}
		return and.Sign() == 0, true
	case "bit_length":
		return float64(rubyBitLen(ti)), true
	case "size":
		if n := (new(big.Int).Abs(ti).BitLen() + 7) / 8; n > 8 {
			return float64(n), true
		}
		return 8.0, true
	case "digits":
		base := big.NewInt(10)
		if len(pos) > 0 {
			base, _ = rubyIntBig(a0)
		}
		if base.Sign() < 0 {
			rt.rubyRaiseErr("ArgumentError", "negative radix")
		}
		if base.Cmp(big.NewInt(2)) < 0 {
			rt.rubyRaiseErr("ArgumentError", "invalid radix "+base.String())
		}
		if ti.Sign() < 0 {
			rt.rubyRaiseErr("Math::DomainError", "out of domain")
		}
		out := &jsArray{}
		v := new(big.Int).Set(ti)
		for {
			q, r := new(big.Int).QuoRem(v, base, new(big.Int))
			out.elems = append(out.elems, narrow(r))
			if q.Sign() == 0 {
				return out, true
			}
			v = q
		}
	case "pow":
		if len(pos) < 2 {
			return rt.rubyBin("**", t, a0), true
		}
		e, eI := rubyIntBig(a0)
		if !eI {
			rt.rubyRaiseErr("TypeError", "Integer#pow() 2nd argument not allowed unless a 1st argument is integer")
		}
		m, mI := rubyIntBig(argAt(pos, 1))
		if !mI {
			rt.rubyRaiseErr("TypeError", "Integer#pow() 2nd argument not allowed unless all arguments are integers")
		}
		if e.Sign() < 0 {
			rt.rubyRaiseErr("RangeError", "Integer#pow() 1st argument cannot be negative when 2nd argument specified")
		}
		if e.Sign() == 0 {
			if m.Sign() > 0 {
				return 1.0, true
			}
			return narrow(new(big.Int).Add(m, big.NewInt(1))), true
		}
		if m.Sign() == 0 {
			rt.rubyRaiseErr("ZeroDivisionError", "divided by 0")
		}
		am := new(big.Int).Abs(m)
		r := new(big.Int).Exp(new(big.Int).Mod(ti, am), e, am)
		r.Mod(r, am)
		if m.Sign() < 0 && r.Sign() != 0 {
			r.Add(r, m)
		}
		return narrow(r), true
	case "[]":
		return rt.rubyIntBit(t, a0), true
	}
	return nil, false
}

// rubyIntBig is an Integer value as a big.Int; ok=false for anything else.
func rubyIntBig(v interface{}) (*big.Int, bool) {
	switch v.(type) {
	case float64, *jsBigInt:
		return rubyBigOf(v)
	}
	return nil, false
}

// rubyBitLen is Integer#bit_length: the bits of x, or of ~x for a negative x.
func rubyBitLen(x *big.Int) int {
	if x.Sign() < 0 {
		return new(big.Int).Not(x).BitLen()
	}
	return x.BitLen()
}

// rubyObjectId is Object#object_id. Twin of objectId in
// languages/ruby-interpreter.abnf, which states the rules.
func (rt *jsrt) rubyObjectId(v interface{}) interface{} {
	if isUndefOrNull(v) {
		return 8.0
	}
	if b, ok := v.(bool); ok {
		if b {
			return 20.0
		}
		return 0.0
	}
	if _, isI := rubyIntBig(v); isI {
		return rt.rubyNumBin("+", rt.rubyNumBin("*", v, 2.0), 1.0)
	}
	valued := false
	switch v.(type) {
	case string, jsRat, jsCpx:
		valued = true
	}
	for i, o := range rt.rubyOids {
		if rubyIdenticalP(o, v) || (valued && rt.rubyEql(o, v)) {
			return float64(16 + 8*i)
		}
	}
	rt.rubyOids = append(rt.rubyOids, v)
	return float64(8 + 8*len(rt.rubyOids))
}

// rubyStrHash is the 31-bit polynomial hash over a string's UTF-16 units.
func rubyStrHash(s string) float64 {
	h := int64(0)
	for _, u := range utf16.Encode([]rune(s)) {
		h = (h*31 + int64(u)) % 2147483647
	}
	return float64(h)
}

// rubyHashOf is Object#hash, by value. Twin of valHash in
// languages/ruby-interpreter.abnf, which states the rules.
func (rt *jsrt) rubyHashOf(v interface{}) float64 {
	if a, ok := v.(*jsArray); ok {
		h := int64(7)
		for _, e := range a.elems {
			h = (h*31 + int64(rt.rubyHashOf(e))) % 2147483647
		}
		return float64(h)
	}
	if keys, vals, isDict := dictParts(v); isDict {
		h := int64(11)
		for i := range keys.elems {
			h = (h + (int64(rt.rubyHashOf(keys.elems[i]))*31+int64(rt.rubyHashOf(vals.elems[i])))%2147483647) % 2147483647
		}
		return float64(h)
	}
	if _, hasOwn := rubyFindMethod(v, "hash"); hasOwn && rubyUserObj(v) {
		return rubyToF(rt.rubyNumBin("%", rt.rubyMethod(v, "hash", nil), 2147483647.0))
	}
	if isUndefOrNull(v) {
		return rubyStrHash("n" + rt.rubyInspect(v))
	}
	switch t := v.(type) {
	case bool:
		return rubyStrHash("n" + rt.rubyInspect(v))
	case float64, *jsBigInt:
		return rubyStrHash("i" + rt.rubyInspect(v))
	case string:
		return rubyStrHash("s" + t)
	case jsSym:
		return rubyStrHash("y" + t.s)
	case jsFlo:
		return rubyStrHash("f" + rt.rubyInspect(v))
	case jsRat:
		return rubyStrHash("r" + rt.rubyInspect(v))
	case jsCpx:
		return rubyStrHash("c" + rt.rubyInspect(v))
	}
	if _, isRng := rubyIsRange(v); isRng {
		return rubyStrHash("g" + rt.rubyInspect(v))
	}
	if rxIsRegexObj(v) {
		return rubyStrHash("x" + rt.rubyInspect(v))
	}
	return rubyStrHash("o" + rt.rubyInspect(rt.rubyObjectId(v)))
}

// rubyEqlDeep is #eql? on a builtin value. Twin of valEql in
// languages/ruby-interpreter.abnf.
func (rt *jsrt) rubyEqlDeep(a, b interface{}) bool {
	la, aArr := a.(*jsArray)
	lb, bArr := b.(*jsArray)
	if !aArr || !bArr {
		return rt.rubyEql(a, b)
	}
	if len(la.elems) != len(lb.elems) {
		return false
	}
	for i := range la.elems {
		if !rt.rubyEqlDeep(la.elems[i], lb.elems[i]) {
			return false
		}
	}
	return true
}

// rubyClassObj: a class or module object.
func rubyClassObj(o *jsObject) bool {
	_, isCls := o.props["__isclass"]
	return isCls
}

// rubyCivSet writes a class object's @ivar. Its instance methods are plain keys
// of the same object, so the @ivar names are recorded in __civn, in assignment
// order, for #instance_variables. Twin of js_rcivset in lib/ruby-rt.metajs.
func rubyCivSet(o *jsObject, k string, v interface{}) {
	l, ok := o.props["__civn"].(*jsArray)
	if !ok {
		l = &jsArray{}
		o.set("__civn", l)
	}
	found := false
	for _, e := range l.elems {
		if e == interface{}(k) {
			found = true
			break
		}
	}
	if !found {
		l.elems = append(l.elems, k)
	}
	o.set(k, v)
}

// rubyScopeSelf is the implicit receiver at scope h: the nearest `self` binding,
// or the main object where that is nil (the top level and its defs, see
// emitIvarSelf in ruby-to-llvm-ir.abnf). Twin of rbScopeSelf in
// languages/lib/ruby-rt.metajs.
func rubyScopeSelf(rt *jsrt, h uint64) interface{} {
	var self, main interface{} = jsNull, jsNull
	found := false
	for sc := rt.scopeOf(h); sc != nil; sc = sc.parent {
		if v, ok := sc.get("self"); ok && !found {
			self, found = v, true
		}
		if v, ok := sc.get("$main"); ok {
			main = v
		}
	}
	if isNullish(self) {
		return main
	}
	return self
}

// rubyCasecmp is String#casecmp: ASCII folded to UPPER case, then MRI's byte
// order, which is code-point order and therefore Go's string order. Twins:
// rbCasecmp in ruby-interpreter.abnf and languages/lib/ruby-rt.metajs.
func rubyCasecmp(a, b string) int {
	up := func(s string) string {
		return strings.Map(func(r rune) rune {
			if r >= 'a' && r <= 'z' {
				return r - 32
			}
			return r
		}, s)
	}
	return strings.Compare(up(a), up(b))
}

// rubySymInspect is Symbol#inspect: the name bare after the colon when MRI's
// symname test passes (rubySymPlain), else quoted as :"a b". Twins: symInspect in
// ruby-interpreter.abnf, rbSymInspect in languages/lib/ruby-rt.metajs.
func rubySymInspect(s string) string {
	if rubySymPlain(s) {
		return ":" + s
	}
	return ":" + rubyStrInspect(s)
}

func rubySymIdCh(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c >= 0x80
}

// rubySymPlain: an operator method name, an identifier with one trailing ? ! or =,
// an @ivar, an @@cvar, or a global ($name, $1, $-w, $~ and the other one-character
// specials). Twins: rbSymPlain in ruby-interpreter.abnf and
// languages/lib/ruby-rt.metajs.
func rubySymPlain(s string) bool {
	n := len(s)
	if n == 0 {
		return false
	}
	switch s {
	case "+", "-", "*", "/", "%", "**", "==", "===", "!=", "=~", "!~", "!", "~", "+@", "-@", "[]", "[]=",
		"<", "<=", ">", ">=", "<=>", "<<", ">>", "&", "|", "^", "`":
		return true
	}
	i := 0
	switch s[0] {
	case '$':
		if n == 1 {
			return false
		}
		if n == 2 && strings.IndexByte("~*$?!@/\\;,.=:<>\"&`'+0", s[1]) >= 0 {
			return true
		}
		if s[1] == '-' {
			return n == 3 && rubySymIdCh(s[2])
		}
		if s[1] >= '0' && s[1] <= '9' {
			k := 2
			for k < n && s[k] >= '0' && s[k] <= '9' {
				k++
			}
			return k == n
		}
		i = 1
	case '@':
		i = 1
		if n > 1 && s[1] == '@' {
			i = 2
		}
	}
	if i >= n || !rubySymIdCh(s[i]) || s[i] >= '0' && s[i] <= '9' {
		return false
	}
	for i < n && rubySymIdCh(s[i]) {
		i++
	}
	if i == n {
		return true
	}
	t := s[i]
	return s[0] != '$' && s[0] != '@' && i == n-1 && (t == '?' || t == '!' || t == '=')
}
