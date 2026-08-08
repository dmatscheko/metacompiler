package abnf

// jsrtint.go -- SIZED INTEGERS for the statically typed languages' compiler
// grammars (languages/go-to-llvm-ir.abnf first; java/kotlin/csharp are the same
// shape when they follow).
//
// The handle runtime models every number as one JS double. That is enough for a
// dynamically typed language and wrong for a statically typed one in two ways
// that a dynamic rule cannot repair, because in an untyped value model the
// operands carry no type:
//
//	var i8 int8 = 127; i8++      real go: -128     here: 128
//	var u8 uint8 = 255; u8++     real go: 0        here: 256
//	1 << 62                      real go: 4611686018427387904   here: 0
//	int64max                     real go: 9223372036854775807   here: 9.223372036854776e+18
//	int64max + 1                 real go: -9223372036854775808  here: 9.223372036854776e+18
//
// The first two need the DECLARED WIDTH of the variable to survive into the
// increment; the last three need 64 bits of precision, which a double has not
// got. So the type goes ON the value, exactly as commit 636e326 put floatness on
// the value with the jsJFlo box.
//
// jsGInt is that box. The invariant, in both halves of every language that opts
// in:
//
//	plain number   ==  a signed 64 bit integer whose value is exact in a double
//	                   (|v| <= 2^53), i.e. Go's int / int64 / rune in the range
//	                   every ordinary program stays inside
//	jsGInt         ==  any other integer: a sized type (int8..uint32), any
//	                   unsigned type, or a 64 bit value that has left the
//	                   exactly representable range
//	jsJFlo         ==  a float64 / float32                    (jsrtjvm.go)
//
// Keeping the common case a PLAIN NUMBER is deliberate and is what makes the
// change affordable: a program that never writes a sized type and never leaves
// 2^53 allocates no box at all and runs exactly the code it ran before. The box
// is rare, so the handful of places that consume a number raw (an array index, a
// slice bound, a map key) meet one only in the programs that asked for it.
//
// The interpreter halves implement the identical rule on top of the i64* pair
// arithmetic in languages/lib/interp-core.js; ./test.sh --cross diffs the two, so
// the two files must stay in step.
//
// Registration is additive: an init() appends to rxExtraExterns, so nothing here
// runs unless a program calls one of the js_gi* externs.

import (
	"math"
	"strconv"
	"strings"
)

// giExact is 2^53: the largest magnitude an integer keeps exactly in a float64.
const giExact = 9007199254740992

// jsGInt is one sized integer. `v` holds the value already truncated to `w`
// bits: sign extended for a signed width, so a int8 -1 is int64(-1); zero
// extended into the low `w` bits for an unsigned one, except that a uint64 keeps
// the raw bit pattern and is READ as uint64(v).
type jsGInt struct {
	v int64
	w uint8 // 8, 16, 32 or 64
	u bool  // unsigned
}

func giIsInt(v interface{}) bool { _, ok := v.(jsGInt); return ok }

// giTrunc wraps a value into `w` bits with `u`'s signedness, which is what every
// arithmetic operator does on overflow in Go, Java, Kotlin and C#.
func giTrunc(v int64, w uint8, u bool) int64 {
	switch w {
	case 8:
		if u {
			return int64(uint8(v))
		}
		return int64(int8(v))
	case 16:
		if u {
			return int64(uint16(v))
		}
		return int64(int16(v))
	case 32:
		if u {
			return int64(uint32(v))
		}
		return int64(int32(v))
	}
	return v
}

// giNorm is the ONE place the invariant above is applied: it answers a plain
// number when the value is a signed 64 bit one that a double holds exactly, and
// a box otherwise. Everything that produces an integer goes through it.
func giNorm(v int64, w uint8, u bool) interface{} {
	v = giTrunc(v, w, u)
	if w == 64 && !u && v <= giExact && v >= -giExact {
		return float64(v)
	}
	return jsGInt{v: v, w: w, u: u}
}

// giVal is the 64 bit value of any integral operand: a box's payload, a plain
// number truncated toward zero, a char's code point. It is the operand reader
// for every operator below.
func giVal(rt *jsrt, x interface{}) int64 {
	switch t := x.(type) {
	case jsGInt:
		return t.v
	case float64:
		return giFromFloat(t)
	case jsChar:
		return int64(t.code)
	case jsJFlo:
		return giFromFloat(t.f)
	}
	return giFromFloat(rt.toNumber(x))
}

// giFromFloat truncates toward zero and wraps modulo 2^64, which is what a
// conversion from a floating point value to a 64 bit integer type does.
func giFromFloat(f float64) int64 {
	if math.IsNaN(f) {
		return 0
	}
	if f >= 9223372036854775808 || f < -9223372036854775808 {
		// Out of int64 range: reduce modulo 2^64 the way the hardware wrap does.
		m := math.Mod(math.Trunc(f), 18446744073709551616)
		if m >= 9223372036854775808 {
			m -= 18446744073709551616
		} else if m < -9223372036854775808 {
			m += 18446744073709551616
		}
		return int64(m)
	}
	return int64(math.Trunc(f))
}

// giWidthOf is the result type of a binary operation. Go (and Java, Kotlin, C#)
// require both operands of an arithmetic operator to have the same type, so at
// most one of them is a box and it decides; two boxes agree in practice, and
// where a program manages to disagree the LEFT one wins, which is the same rule
// jvmStyleOf uses for the float box.
func giWidthOf(l, r interface{}) (uint8, bool) {
	if b, ok := l.(jsGInt); ok {
		return b.w, b.u
	}
	if b, ok := r.(jsGInt); ok {
		return b.w, b.u
	}
	return 64, false
}

// giArith is one binary arithmetic or bitwise operator, evaluated at the result
// type's full width and wrapped to it.
func (rt *jsrt) giArith(op string, l, r interface{}) interface{} {
	w, u := giWidthOf(l, r)
	// A SHIFT is the one operator whose result type is the LEFT operand's ALONE:
	// the count is a separate operand with a type of its own. Go says so outright
	// ("the result type of a shift is the type of the left operand", Arithmetic
	// operators), and so do JLS 15.19, ECMA-334 12.11, Kotlin's shl/shr/ushr
	// signatures (the count is always an Int) and Swift's
	// `>> <RHS: BinaryInteger>` (the count is any integer type, the result is
	// Self). giWidthOf reads the type off WHICHEVER operand is boxed, so a plain
	// `int` shifted by a `uint8` count would come out 8 bits UNSIGNED. Every
	// language's layer 2 normalises the count before it gets here - measured, see
	// the note in si_apply - so this is a guard on the primitive, not a repair of
	// an observable bug.
	if op == "<<" || op == ">>" {
		w, u = giWidthOf(l, l)
	}
	a, b := giVal(rt, l), giVal(rt, r)
	var x int64
	switch op {
	case "+":
		x = a + b
	case "-":
		x = a - b
	case "*":
		x = a * b
	case "/":
		if b == 0 {
			rt.fail("integer divide by zero")
		}
		if u {
			x = int64(giU(a, w) / giU(b, w))
		} else if a == math.MinInt64 && b == -1 {
			x = a // the hardware wrap; Go panics, but only for the 64 bit case
		} else {
			x = a / b
		}
	case "%":
		if b == 0 {
			rt.fail("integer divide by zero")
		}
		if u {
			x = int64(giU(a, w) % giU(b, w))
		} else if a == math.MinInt64 && b == -1 {
			x = 0
		} else {
			x = a % b
		}
	case "&":
		x = a & b
	case "|":
		x = a | b
	case "^":
		x = a ^ b
	case "&^":
		x = a &^ b
	case "<<":
		// The shift COUNT is a separate operand and is never negative in a
		// well-typed program; a count at or above the width shifts everything
		// out, which is Go's rule (and unlike C, not undefined).
		s := giVal(rt, r)
		if s < 0 || s >= int64(w) {
			x = 0
		} else {
			x = a << uint(s)
		}
	case ">>":
		s := giVal(rt, r)
		if s < 0 {
			x = 0
		} else if u {
			if s >= int64(w) {
				x = 0
			} else {
				x = int64(giU(a, w) >> uint(s))
			}
		} else {
			if s >= int64(w) {
				s = int64(w) - 1
			}
			x = a >> uint(s)
		}
	default:
		rt.fail("js_giarith: unknown operator %q", op)
	}
	return giNorm(x, w, u)
}

// giU reads a value as unsigned at width w.
func giU(v int64, w uint8) uint64 {
	switch w {
	case 8:
		return uint64(uint8(v))
	case 16:
		return uint64(uint16(v))
	case 32:
		return uint64(uint32(v))
	}
	return uint64(v)
}

// giCmp is <, <=, > and >= for two integral operands: unsigned when the result
// type is unsigned, so uint64max > 0 rather than -1 < 0.
func (rt *jsrt) giCmp(l, r interface{}) int {
	w, u := giWidthOf(l, r)
	a, b := giVal(rt, l), giVal(rt, r)
	if u {
		ua, ub := giU(a, w), giU(b, w)
		if ua < ub {
			return -1
		}
		if ua > ub {
			return 1
		}
		return 0
	}
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

// giEq is == between an integer box and any other value. A box never equals a
// non-numeric value, and two integers compare by VALUE whatever their widths -
// which is what a well-typed program can observe, since it could not have
// written the comparison at all if the types disagreed.
func (rt *jsrt) giEq(l, r interface{}) bool {
	if giIsNumeric(l) && giIsNumeric(r) {
		if f, ok := l.(jsJFlo); ok {
			return f.f == giFloat(rt, r)
		}
		if f, ok := r.(jsJFlo); ok {
			return f.f == giFloat(rt, l)
		}
		return giVal(rt, l) == giVal(rt, r)
	}
	return rt.strictEq(l, r)
}

// giIsIntegral is the narrower test the ordered comparisons need: a float
// operand must NOT be truncated into an integer comparison.
func giIsIntegral(v interface{}) bool {
	switch v.(type) {
	case jsGInt, float64, jsChar:
		return true
	}
	return false
}

func giIsNumeric(v interface{}) bool {
	switch v.(type) {
	case jsGInt, float64, jsJFlo, jsChar:
		return true
	}
	return false
}

// giFloat is the floating point reading of an integral operand, which an
// unsigned 64 bit box needs its own path for (int64(-1) as a uint64 is 1.8e19).
func giFloat(rt *jsrt, v interface{}) float64 {
	if b, ok := v.(jsGInt); ok {
		if b.u && b.w == 64 {
			return float64(uint64(b.v))
		}
		return float64(b.v)
	}
	if f, ok := v.(jsJFlo); ok {
		return f.f
	}
	return rt.toNumber(v)
}

// giStr is the decimal text of a box - the one place the unsigned reading
// matters for output.
func giStr(b jsGInt) string {
	if b.u {
		return strconv.FormatUint(giU(b.v, b.w), 10)
	}
	return strconv.FormatInt(b.v, 10)
}

// giConv is an explicit conversion to a sized integer type: int8(x), uint32(x),
// int64(x), uint(x). A floating point source truncates toward zero first.
func (rt *jsrt) giConv(x interface{}, w uint8, u bool) interface{} {
	return giNorm(giVal(rt, x), w, u)
}

// giOpNames is sintOp's operator table: the same 0..10 indexing
// languages/lib/runtime.c's si_apply uses, so the two halves take one number
// rather than an operator string across the boundary.
var giOpNames = []string{"+", "-", "*", "/", "%", "&", "|", "^", "&^", "<<", ">>"}

// giBindings adds the TWELVE sint* host globals that languages/lib/runtime.c
// seeds (host ids 40..50, `seed_root("sint", mk_host(40))` and the ten after
// it, plus sintRaw at id 62) to a program's global set.
//
// WHY IT IS HERE. A natively linked program reaches a sized integer through
// those names; a program run by llvm.Run reaches the very same runtime through
// abnf/jsrt*.go, and until this existed it reached NOTHING - `sint` was
// "variable not defined" in the Go half while it worked in the C one. A layer-2
// file written against the C floor (languages/lib/lua-rt.metajs is the first)
// therefore could not be run by llvm.Run at all, and no MetaJS source program
// could pin the tag's semantics in a test that both halves run. Every function
// below is the same giXxx the C floor's si_xxx transliterates, so the three
// engines (goja/frozen interpreter, llvm.Run, native) implement one spec.
func giBindings(b map[string]interface{}) {
	// sint(hi, lo, bits, unsigned): two UNSIGNED 32 bit halves, because a
	// MetaJS number is a double and cannot carry 64 bits exactly. bits
	// defaults to 64 and unsigned to false, matching the floor's `n > 2` /
	// `n > 3`. The result goes through giNorm, so it is a PLAIN NUMBER
	// whenever the invariant says so.
	b["sint"] = jsHostFunc("sint", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		hi := uint64(giFromFloat(rt.toNumber(argAt(args, 0)))) & 0xffffffff
		lo := uint64(giFromFloat(rt.toNumber(argAt(args, 1)))) & 0xffffffff
		w := uint8(64)
		if len(args) > 2 {
			w = uint8(jsToInt(rt.toNumber(args[2])))
		}
		u := false
		if len(args) > 3 {
			u = rt.truthy(args[3])
		}
		return giNorm(int64(hi<<32|lo), w, u)
	})
	// sintRaw(hi, lo, bits, unsigned): sint() WITHOUT giNorm - ALWAYS a jsGInt,
	// never a plain float64. languages/lib/runtime.c seeds it as host id 62 and
	// the comment there carries the argument in full; the short version is that
	// giNorm unboxes a signed 64 bit value a double holds exactly, which is
	// exactly what a statically typed language's `long` must not do, and every
	// other sint* builtin goes through giNorm. giTrunc is still applied, so the
	// box keeps jsGInt's own invariant ("v holds the value already truncated to
	// w bits"); only the unboxing arm is dropped.
	b["sintRaw"] = jsHostFunc("sintRaw", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		hi := uint64(giFromFloat(rt.toNumber(argAt(args, 0)))) & 0xffffffff
		lo := uint64(giFromFloat(rt.toNumber(argAt(args, 1)))) & 0xffffffff
		w := uint8(64)
		if len(args) > 2 {
			w = uint8(jsToInt(rt.toNumber(args[2])))
		}
		u := false
		if len(args) > 3 {
			u = rt.truthy(args[3])
		}
		return jsGInt{v: giTrunc(int64(hi<<32|lo), w, u), w: w, u: u}
	})
	b["sintIs"] = jsHostFunc("sintIs", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return giIsInt(argAt(args, 0))
	})
	// The two halves back out, both read UNSIGNED - si_u(si_val(v), 64) >> 32
	// and si_val(v) & 0xffffffff.
	b["sintHi"] = jsHostFunc("sintHi", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return float64(uint64(giVal(rt, argAt(args, 0))) >> 32)
	})
	b["sintLo"] = jsHostFunc("sintLo", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return float64(uint64(giVal(rt, argAt(args, 0))) & 0xffffffff)
	})
	b["sintWidth"] = jsHostFunc("sintWidth", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		v := argAt(args, 0)
		w, _ := giWidthOf(v, v)
		return float64(w)
	})
	b["sintUns"] = jsHostFunc("sintUns", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		v := argAt(args, 0)
		_, u := giWidthOf(v, v)
		return u
	})
	// One binary operator by INDEX (see giOpNames). An index the table does not
	// have answers giNorm(0), which is what si_apply's `long x = 0` with no arm
	// taken produces - stated rather than approximated, so the halves agree even
	// on a caller's mistake.
	b["sintOp"] = jsHostFunc("sintOp", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		c := jsToInt(rt.toNumber(argAt(args, 0)))
		l, r := argAt(args, 1), argAt(args, 2)
		if c < 0 || c >= len(giOpNames) {
			w, u := giWidthOf(l, r)
			return giNorm(0, w, u)
		}
		return rt.giArith(giOpNames[c], l, r)
	})
	b["sintCmp"] = jsHostFunc("sintCmp", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return float64(rt.giCmp(argAt(args, 0), argAt(args, 1)))
	})
	b["sintConv"] = jsHostFunc("sintConv", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return giNorm(giVal(rt, argAt(args, 0)),
			uint8(jsToInt(rt.toNumber(argAt(args, 1)))), rt.truthy(argAt(args, 2)))
	})
	// The decimal text and the double reading. A non-box passes through the
	// ordinary toString / toNumber, exactly as si_str's `if (tag_of(v) == 13)`
	// and si_float's do.
	b["sintStr"] = jsHostFunc("sintStr", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		if g, ok := argAt(args, 0).(jsGInt); ok {
			return giStr(g)
		}
		return rt.toString(argAt(args, 0))
	})
	b["sintNum"] = jsHostFunc("sintNum", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return giFloat(rt, argAt(args, 0))
	})
}

// programJSBindings is standardJSBindings plus the sint* and flo* families: the host set of
// a RUNNING program (runJSModule), which is the only place a sized integer can
// appear. The GRAMMAR script hosts deliberately keep the standard set, so that
// goja (commonscript.go) and the frozen engine still bind the same names and a
// tag script cannot come to depend on one of them.
func programJSBindings() map[string]interface{} {
	b := standardJSBindings()
	giBindings(b)
	jfBindings(b)    // the boxed double, jsrtjvm.go - the same argument, one type over.
	keysBindings(b)  // keysOf, below - the same argument again, for object enumeration.
	scopeBindings(b) // the scope API and isGenerator - the argument once more, for the
	// one piece of floor storage layer 2 is HANDED and could not open.
	// floPrec, host id 70 of languages/lib/runtime.c: a layer-2 file that renders
	// a float the way C's %g does needs the value rounded to a FIXED number of
	// digits, and the dialect has only the shortest form. The grammar hosts bind
	// it too (frozenBaseBindings in frozen.go, commonscript.go for goja), because
	// the interpreter half of the same language asks the same question; see
	// floPrecStr in commonscript.go.
	b["floPrec"] = jsHostFunc("floPrec", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return floPrecStr(rt.toNumber(argAt(args, 0)), int(jsToInt(rt.toNumber(argAt(args, 1)))))
	})
	// fail(msg): a RUNTIME ERROR, not an exception - the twin of host id 36 in
	// languages/lib/runtime.c, whose comment already claims "the Go twin's rt.fail
	// panics with exactly this text". It did, but the name was not BOUND here, so
	// `fail` was a floor-only global: every layer-2 file calls it (kotlin-rt 73
	// times, python-rt 65, ruby-rt 38) and a MetaJS program could call it in a
	// native binary and not under llvm.Run. docs/todo.md 4.1.
	b["fail"] = jsHostFunc("fail", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		rt.fail("%s", rt.toString(argAt(args, 0)))
		return jsUndef
	})
	// THE PROGRAM SET IS THE C FLOOR'S seed_root LIST, EXACTLY - see
	// metajsProgramGlobals in hostglobals_test.go, which parses all three engines
	// out of their own sources and fails if any of them gains or loses a name.
	//
	// standardJSBindings serves two masters: this one and the GRAMMAR script host
	// (frozenBaseBindings in frozen.go, mirrored for goja in commonscript.go). The
	// three names below belong to the grammar host alone and were reaching a MetaJS
	// program only because it inherits that map:
	//
	//   Object  - only ever wanted as Object.prototype.hasOwnProperty.call(o, k) by
	//             tag scripts (c-preprocessor.abnf, lisp, tinyc, bash, swift...).
	//             languages/lib/runtime.c seeds no such name, so it worked under
	//             llvm.Run and died in the native binary. js and typescript do NOT
	//             lose anything: their emitters declare their own Object into the
	//             program scope with js_jsobject (js-to-llvm-ir.abnf:5786).
	//   rawSet  - an own-property write that bypasses a host accessor. A MetaJS
	//             program has no accessors, so it is meaningless to one; the floor
	//             seeds no such name either. interp-core.js's use of it is a TAG
	//             SCRIPT use and keeps working.
	//
	// __jmath is the one deliberate exception and it STAYS: it is `__`-prefixed
	// (not a name a MetaJS program is meant to see), and java-to-llvm-ir.abnf reads
	// it through jvScopeProbe, which takes the hit arm under llvm.Run and falls
	// back to layer 2's own java.lang descriptor natively. Removing it would change
	// java's answers under llvm.Run.
	delete(b, "Object")
	delete(b, "rawSet")
	return b
}

// keysBindings adds keysOf and delKey, which languages/lib/runtime.c seeds as
// host ids 61 and 63 - the pair a layer-2 file needs to WALK and to PRUNE an
// object, neither of which MetaJS itself can express.
//
// WHY A BUILTIN AND NOT THE js_keys EXTERN. An extern is only callable from an
// emitter, and MetaJS has neither for..in nor Object.keys - so a layer-2 file
// could not enumerate an object's own keys AT ALL. That blocks every runtime
// operation that has to walk an object: PHP's var_dump, `==` between two
// objects, the (array) cast and clone are the first four, and the remaining nine
// languages have the same four.
//
// The body is the js_keys extern's (jsrt.go, the big extern table) restated for
// the host-call path: a dict box yields its key array, a plain object yields its
// string keys in INSERTION order, skipping the internal __-prefixed slots. The
// two must stay in step; the ratchet in tests/metajs-test-full.js runs the same
// program through this, through the C floor's host id 61 and through the
// interpreter grammar's own answer.
func keysBindings(b map[string]interface{}) {
	b["keysOf"] = jsHostFunc("keysOf", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		o, ok := argAt(args, 0).(*jsObject)
		if !ok {
			rt.fail("js_keys: not an object (got %s)", rt.typeOf(argAt(args, 0)))
		}
		if keys, _, isDict := dictParts(o); isDict {
			return &jsArray{elems: append([]interface{}{}, keys.elems...)}
		}
		out := &jsArray{}
		for _, k := range o.keys {
			if len(k) >= 2 && k[0] == '_' && k[1] == '_' {
				continue
			}
			out.elems = append(out.elems, k)
		}
		return out
	})
	// delKey(o, key), which languages/lib/runtime.c seeds as host id 63.
	//
	// The same argument as keysOf, one operation over: MetaJS has no `delete`, so
	// without this a layer-2 file cannot REMOVE a key at all - it has to blank the
	// slot and keep a side table of what it blanked, which is quadratic in the
	// number of deletions, leaks, and still gets key ORDER after a
	// delete-then-reinsert, for-in and object spread wrong, because all three walk
	// the runtime's own key array.
	//
	// The body is the js_del extern's (jsrt.go, the big extern table): drop the
	// property and its key-order entry, answering true whether or not it was
	// there. tests/metajs-test-full.js runs the same program through this, through
	// the C floor's host id 63 and through the interpreter grammar's own answer.
	b["delKey"] = jsHostFunc("delKey", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		o, ok := argAt(args, 0).(*jsObject)
		if !ok {
			return true
		}
		key := rt.toString(argAt(args, 1))
		if _, had := o.props[key]; had {
			delete(o.props, key)
			for i, k := range o.keys {
				if k == key {
					o.keys = append(o.keys[:i], o.keys[i+1:]...)
					break
				}
			}
		}
		return true
	})
}

// scopeBindings adds the FIVE scope* host globals and isGenerator, which
// languages/lib/runtime.c seeds as host ids 64..69.
//
// WHY THEY EXIST. A scope is the floor's private storage. An emitter hands a
// layer-2 function a scope handle - js_pyset_var's `s`, and every `defined?` /
// `isset` / `global` / `nonlocal` probe - and until now layer 2 could do
// NOTHING with it: MetaJS cannot open a cell, and no extern is callable from a
// MetaJS source file. Six languages answered that by lowering the probe into
// their EMITTER instead (docs/runtime-next-plan.md: swift, dart, go, ruby,
// kotlin's nine helpers, python's six), which is the same IR-building written
// out once per language for one language-neutral question. Three separate
// migration reports asked for this API by name.
//
//	scopeNew(parent)      a scope whose parent is `parent`; undefined / null /
//	                      a missing argument means NO parent
//	scopeParent(s)        the enclosing scope, undefined at the top of a chain
//	scopeGet(s, name)     the value, walking the CHAIN (js_scope_get); aborts
//	                      with "variable not defined" when nowhere in it
//	scopeHas(s, name)     is the name bound in THIS scope - NO chain walk
//	scopeDecl(s, name, v) bind here, overwriting a binding of this scope
//
// scopeHas is the own-scope test on purpose. It is the one question
// js_scope_typeof cannot answer (that answers "undefined" for an absent name
// and for a slot holding undefined alike) and it is the one python asked for by
// name; a chain walk is scopeParent in a loop, and there is no scopeSet because
// "walk with scopeHas/scopeParent, then scopeDecl" is it.
//
// TWO DELIBERATE ASYMMETRIES with the js_scope_* externs, and they are what
// makes the three engines agree. An absent parent is undefined and NOT the root
// scope: a builtin's arguments are values, so the handle 0 that means "global"
// to an emitter never arrives here, and a chain that ran into the host globals
// would have no twin in metajs-interpreter.abnf, whose host globals are a plain
// object and not a scope. And the scope argument is TYPE CHECKED rather than
// coerced, so a caller's mistake is the same abort in all three halves.
func scopeBindings(b map[string]interface{}) {
	// scopeOfArg is languages/lib/runtime.c's host_scope: a real scope or an
	// abort. rt.scopeOf cannot be used - it takes a HANDLE and maps 0 to the
	// root, and a host builtin receives values.
	scopeOfArg := func(rt *jsrt, v interface{}) *jsScope {
		if sc, ok := v.(*jsScope); ok {
			return sc
		}
		rt.fail("not a scope: %s", rt.typeOf(v))
		return nil
	}
	b["scopeNew"] = jsHostFunc("scopeNew", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		p := argAt(args, 0)
		if p == nil || p == jsUndef || p == jsNull {
			return &jsScope{}
		}
		return &jsScope{parent: scopeOfArg(rt, p)}
	})
	b["scopeParent"] = jsHostFunc("scopeParent", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		p := scopeOfArg(rt, argAt(args, 0)).parent
		if p == nil {
			return jsUndef
		}
		return p
	})
	// The CHAIN walk, which is rt.scopeGet and therefore js_scope_get: the same
	// walk, the same "variable not defined" abort at the end of it.
	b["scopeGet"] = jsHostFunc("scopeGet", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return rt.scopeGet(scopeOfArg(rt, argAt(args, 0)), rt.toString(argAt(args, 1)))
	})
	b["scopeHas"] = jsHostFunc("scopeHas", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		return scopeOfArg(rt, argAt(args, 0)).has(rt.toString(argAt(args, 1)))
	})
	b["scopeDecl"] = jsHostFunc("scopeDecl", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		scopeOfArg(rt, argAt(args, 0)).put(rt.toString(argAt(args, 1)), argAt(args, 2))
		return jsUndef
	})
	// isGenerator(v): is this the value js_gen_create makes and js_gen_next
	// drives (tag 15 in the C floor, *jsGenerator here).
	//
	// WHY A PREDICATE AND NOT A GENERAL js_tag(v). A numeric tag cannot be the
	// same in the three engines: js_genfn is a tag 16 CELL in the floor and a
	// *hostFunc here, so it would be 16 in one half and 8 in the other, and
	// metajs-interpreter.abnf has no tag numbering at all - its values are the
	// host engine's, where a closure, a host function and a bound method are
	// one JS function. Every OTHER distinction a tag would carry already has a
	// name layer 2 can use: typeof, sintIs, floIs, and `typeof v.length ==
	// "number"` for an array. A generator is the one shape with no such name,
	// and languages/lib/php-rt.metajs's phIsGen is the evidence - it excludes
	// __dict / __refcell / __isclass / __class / length one at a time and then
	// asks whether v["next"] is callable, which php's own migration report
	// calls "the one guess in this port".
	//
	// A generator is an OBJECT to typeof in all three halves, so this refines
	// typeof and never contradicts it.
	b["isGenerator"] = jsHostFunc("isGenerator", func(rt *jsrt, this uint64, args []interface{}) interface{} {
		_, ok := argAt(args, 0).(*jsGenerator)
		return ok
	})
}

func init() {
	rxExtraExterns = append(rxExtraExterns, func(rt *jsrt, m map[string]func(args []uint64) uint64) {
		u := rt.unwrap
		w := rt.wrap
		boolH := func(b bool) uint64 {
			if b {
				return jsHTrue
			}
			return jsHFalse
		}

		// One binary operator: (op name, left, right). A single extern keeps the
		// emitted IR one call per operator, the way js_jf* is one per operator -
		// the operator itself is a compile time constant string, so the switch
		// above costs one comparison chain and no dispatch table.
		m["js_giarith"] = func(a []uint64) uint64 {
			op, l, r := rt.toString(u(a[0])), u(a[1]), u(a[2])
			// A float64 operand keeps the FLOAT path (the jsJFlo box of
			// jsrtjvm.go): the emitters route every arithmetic operator here, so
			// this is where the two boxes meet. Only '- * / %' exist for floats;
			// a bitwise operator on one is not a program the front end accepts.
			if jvmIsFlo(l) || jvmIsFlo(r) {
				switch op {
				case "-", "*", "/", "%":
					return w(rt.jvmArith(op[0], l, r))
				}
			}
			return w(rt.giArith(op, l, r))
		}
		// Ordered comparison: answers -1, 0 or 1 so one extern serves < <= > >=.
		m["js_gicmp"] = func(a []uint64) uint64 {
			return rt.wrapNum(float64(rt.giCmp(u(a[0]), u(a[1]))))
		}
		m["js_gieq"] = func(a []uint64) uint64 { return boolH(rt.giEq(u(a[0]), u(a[1]))) }
		// A conversion / a declared width: (value, bits, unsigned).
		m["js_giconv"] = func(a []uint64) uint64 {
			return w(rt.giConv(u(a[0]), uint8(jsToInt(rt.toNumber(u(a[1])))), rt.truthy(u(a[2]))))
		}
		// Unary minus and ^x (bitwise complement), both at the operand's width.
		m["js_gineg"] = func(a []uint64) uint64 {
			v := u(a[0])
			// -x keeps the operand's type: a float64 negates as a float64, so
			// -0.0 and -1.0/0.0 stay real; an integer negates at its own width,
			// so -int8(-128) wraps back to -128.
			if f, ok := v.(jsJFlo); ok {
				return w(jsJFlo{f: -f.f, sty: f.sty})
			}
			wd, un := giWidthOf(v, v)
			return w(giNorm(-giVal(rt, v), wd, un))
		}
		m["js_ginot"] = func(a []uint64) uint64 {
			v := u(a[0])
			wd, un := giWidthOf(v, v)
			return w(giNorm(^giVal(rt, v), wd, un))
		}
		// The plain-number reading an index, a slice bound or a repeat count
		// needs: a box is not a float64 and every such consumer wants one.
		m["js_ginum"] = func(a []uint64) uint64 { return rt.wrapNum(giFloat(rt, u(a[0]))) }
		// The decimal text, for the print and string-conversion paths.
		m["js_gistr"] = func(a []uint64) uint64 {
			if b, ok := u(a[0]).(jsGInt); ok {
				return rt.wrapStr(giStr(b))
			}
			return rt.wrapStr(rt.toString(u(a[0])))
		}
		m["js_giis"] = func(a []uint64) uint64 { return boolH(giIsInt(u(a[0]))) }
		// The UNBOXER: a sized integer becomes its plain number, everything else
		// passes through untouched. It is what an existing consumer that only
		// knows plain numbers needs in front of it - js_goconv's string(rune),
		// an index, a slice bound - without that consumer having to learn the box.
		m["js_gival"] = func(a []uint64) uint64 {
			if b, ok := u(a[0]).(jsGInt); ok {
				if b.u && b.w == 64 {
					return rt.wrapNum(float64(uint64(b.v)))
				}
				return rt.wrapNum(float64(b.v))
			}
			return a[0]
		}
		// The four ordered comparisons. A box has to be compared through giCmp
		// rather than through the shared jsCompare, because a uint64 above 2^63
		// reads as a NEGATIVE int64 there, so uint64max < 0 would hold. Anything
		// that is not two integers (a string pair, a float operand) keeps the
		// runtime's own comparison, which already reads a box through toNumber.
		rel := func(name string, want func(int) bool) {
			m[name] = func(a []uint64) uint64 {
				l, r := u(a[0]), u(a[1])
				if giIsInt(l) || giIsInt(r) {
					if giIsIntegral(l) && giIsIntegral(r) {
						return boolH(want(rt.giCmp(l, r)))
					}
				}
				return boolH(want(rt.jsCompare(l, r)))
			}
		}
		rel("js_gilt", func(c int) bool { return c < 0 })
		rel("js_gile", func(c int) bool { return c <= 0 })
		rel("js_gigt", func(c int) bool { return c > 0 })
		rel("js_gige", func(c int) bool { return c >= 0 })
		// '+' is the one arithmetic operator that is not only arithmetic: in Go
		// it also concatenates strings, and a float operand keeps its box. Two
		// integers add at 64 bits and wrap to the result type's width.
		m["js_giadd"] = func(a []uint64) uint64 {
			l, r := u(a[0]), u(a[1])
			if ls, ok := l.(string); ok {
				if rs, ok2 := r.(string); ok2 {
					return rt.wrapStr(strConcat(ls, rs))
				}
			}
			if jvmIsFlo(l) || jvmIsFlo(r) {
				return w(rt.jvmArith('+', l, r))
			}
			if !giIsNumeric(l) || !giIsNumeric(r) {
				return w(rt.jsAdd(l, r))
			}
			return w(rt.giArith("+", l, r))
		}
		// A literal that a double cannot hold exactly: the emitter passes its
		// DIGITS, because emitNum would already have rounded 9223372036854775807
		// to 9223372036854776000 on the way into the module. (text, radix, bits,
		// unsigned); an out-of-range value wraps, which is what a Go constant
		// conversion does.
		m["js_gilit"] = func(a []uint64) uint64 {
			s := rt.toString(u(a[0]))
			radix := uint64(jsToInt(rt.toNumber(u(a[1]))))
			bits := uint8(jsToInt(rt.toNumber(u(a[2]))))
			uns := rt.truthy(u(a[3]))
			neg := false
			if strings.HasPrefix(s, "-") {
				neg, s = true, s[1:]
			}
			// Accumulated by hand rather than with ParseUint: the full 64 bit
			// range has to round-trip (ParseInt rejects 18446744073709551615)
			// and an overflowing literal has to WRAP rather than saturate.
			var acc uint64
			for i := 0; i < len(s); i++ {
				var d uint64
				switch c := s[i]; {
				case c >= '0' && c <= '9':
					d = uint64(c - '0')
				case c >= 'a' && c <= 'f':
					d = uint64(c-'a') + 10
				case c >= 'A' && c <= 'F':
					d = uint64(c-'A') + 10
				default:
					continue // a '_' digit separator
				}
				if d >= radix {
					continue
				}
				acc = acc*radix + d
			}
			v := int64(acc)
			if neg {
				v = -v
			}
			return w(giNorm(v, bits, uns))
		}
	})
}
