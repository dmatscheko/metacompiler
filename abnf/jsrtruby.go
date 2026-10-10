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
)

func init() {
	// FloatDomainError < RangeError, so `rescue FloatDomainError' and
	// `rescue RangeError' both catch what rubyFloDomain raises. rubyExcParent is a
	// plain package var and rubyBuiltinClass makes a class object for any name, so
	// this is the whole registration.
	rubyExcParent["FloatDomainError"] = "RangeError"
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
		panic(&jsThrown{value: rt.rubyExc(rt.rubyBuiltinClass("NameError"),
			[]interface{}{"undefined local variable or method '" + name + "'"})})
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

func rubyFloPredicate(target interface{}, name string) (interface{}, bool) {
	if name != "nan?" && name != "infinite?" && name != "finite?" {
		return nil, false
	}
	var x float64
	switch t := target.(type) {
	case jsFlo:
		x = t.f
	case float64:
		x = t
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
		rt.rubyRaiseErr("NoMethodError", "undefined method `"+name+"' for "+rt.rubyInspect(t)+":Complex")
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
