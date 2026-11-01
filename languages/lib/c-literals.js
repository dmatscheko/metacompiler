// c-literals.js -- what c-interpreter.abnf and c-to-llvm-ir.abnf share (include):
// the values of C string literals, character constants and the standard headers'
// constants, brace-initializer items, the block-scope name table and the integer
// constant expression fold. digVal, fail, hasOwn, makeNum, ctI, ctInt, ctChar,
// fieldOf, globalScope and cLitCt (an integer literal's type, C11 6.4.4.1p5) come
// from the including grammar.
//
// The execution character set is UTF-8, as cc's: a narrow literal holds the
// UTF-8 bytes of every character outside ASCII and of every \u / \U universal
// character name (C11 6.4.3); a u literal holds UTF-16 code units (a code point
// above U+FFFF is a surrogate pair, C11 6.4.5p6 with __STDC_UTF_16__), and an L or
// U literal one element per code point.

// The code point that starts at text[i]: a UTF-16 surrogate pair joins into one.
function cCodePointAt(text, i) {
    var hi = text.charCodeAt(i)
    if (hi >= 55296 && hi <= 56319 && i + 1 < text.length) {
        var lo = text.charCodeAt(i + 1)
        if (lo >= 56320 && lo <= 57343) return (hi - 55296) * 1024 + (lo - 56320) + 65536
    }
    return hi
}
// Append one code point in the encoding of the literal's prefix k ("" or "u8"
// narrow, "u" UTF-16, "L" / "U" the code point itself).
function cPushCodePoint(bytes, cp, k) {
    if (k == "u" && cp > 65535) {
        bytes.push(55296 + ((cp - 65536) >> 10))
        bytes.push(56320 + ((cp - 65536) & 1023))
        return
    }
    if (k == "u" || k == "L" || k == "U" || cp < 128) { bytes.push(cp); return }
    if (cp < 2048) {
        bytes.push(192 + (cp >> 6))
    } else if (cp < 65536) {
        bytes.push(224 + (cp >> 12))
        bytes.push(128 + ((cp >> 6) & 63))
    } else {
        bytes.push(240 + (cp >> 18))
        bytes.push(128 + ((cp >> 12) & 63))
        bytes.push(128 + ((cp >> 6) & 63))
    }
    bytes.push(128 + (cp & 63))
}
// The bytes of a string-literal RUN (adjacent literals concatenate, C11 6.4.5p5),
// with the C escape sequences decoded and the terminating null appended. kind is
// the run's prefix ("" for none); an L / u / U prefix makes the elements wide.
function cStrBytes(text) {
    var bytes = []
    var kind = ""
    var i = 0
    while (i < text.length) {
        var cc = text.charCodeAt(i)
        if (cc == 117 && i + 1 < text.length && text.charCodeAt(i + 1) == 56) { i = i + 2; continue }
        if (cc == 76 || cc == 117 || cc == 85) { kind = text.charAt(i); i = i + 1; continue }
        if (cc != 34) { i = i + 1; continue }
        i = i + 1
        while (i < text.length && text.charCodeAt(i) != 34) {
            if (text.charCodeAt(i) != 92) {
                var cp = cCodePointAt(text, i)
                cPushCodePoint(bytes, cp, kind)
                i = i + (cp > 65535 ? 2 : 1)
                continue
            }
            var j = i + 1
            var e = text.charCodeAt(j)
            if (e == 120 || e == 88) {
                j = j + 1
                while (j < text.length && digVal(text.charCodeAt(j)) >= 0) j = j + 1
            } else if (e >= 48 && e <= 55) {
                j = j + 1
                var n = 1
                while (j < text.length && n < 3 && text.charCodeAt(j) >= 48 && text.charCodeAt(j) <= 55) { j = j + 1; n = n + 1 }
            } else if (e == 117) {
                j = j + 5
            } else if (e == 85) {
                j = j + 9
            } else {
                j = j + 1
            }
            if (e == 117 || e == 85) { cPushCodePoint(bytes, cCharCode(text.slice(i, j)), kind) }
            else { bytes.push(cCharCode(text.slice(i, j))) }
            i = j
        }
        i = i + 1
    }
    bytes.push(0)
    return {str: bytes, kind: kind}
}
// The element type of a string literal (C11 6.4.5p6): char, char16_t, char32_t or
// wchar_t, the last an int on this target. Also the type of a character constant
// with the same prefix (6.4.4.4p11); an unprefixed one is an int instead.
function cWideCt(kind) {
    if (kind == "u") return ctI(2, true)
    if (kind == "U") return ctI(4, true)
    if (kind == "L") return ctInt
    return ctChar
}
// The value of one character-constant body, including the C escape sequences:
// backslash x with hex digits, backslash with up to three octal digits, the
// universal character names \u and \U (a code point), and the simple escapes. A
// byte is answered unsigned; cCharConst gives it the type of a constant.
function cCharCode(s) {
    if (s.charCodeAt(0) != 92) return cCodePointAt(s, 0)
    var e = s.charCodeAt(1)
    if (e == 120 || e == 88 || e == 117 || e == 85) {
        var hv = 0
        for (var i = 2; i < s.length; i++) hv = hv * 16 + digVal(s.charCodeAt(i))
        return (e == 117 || e == 85) ? hv : (hv & 255)
    }
    if (e >= 48 && e <= 55) {
        var ov = 0
        for (var j = 1; j < s.length; j++) ov = ov * 8 + digVal(s.charCodeAt(j))
        return ov & 255
    }
    if (e == 110) return 10
    if (e == 116) return 9
    if (e == 114) return 13
    if (e == 118) return 11
    if (e == 102) return 12
    if (e == 97) return 7
    if (e == 98) return 8
    if (e == 101) return 27
    return e   // backslash, single quote, double quote, question mark: itself.
}
// A prefixed character constant from its body and its prefix k, which WCharLit pops
// in that order. A u8 one (a C23 form) stays an int.
function cWideChar(body, k) { return makeNum(cCharCode(body), k == "u8" ? ctInt : cWideCt(k)) }
// An integer character constant is an int holding the value of a char (C11
// 6.4.4.4p10), and char is signed on this target: '\377' is -1.
function cCharConst(s) {
    var v = cCharCode(s)
    return v > 127 ? v - 256 : v
}

// An integer constant expression (CExpr) is folded as the tags walk, on TYPED
// values {v, t}: v is the value, held exactly as a double, t its integer C type.
// The type decides the result as C11 6.3.1.8's usual arithmetic conversions do:
// '-1 < 0u' compares in unsigned int, where -1 is UINT_MAX.
function cK(v, t) { return {v: v, t: t} }
// The integer promotions (C11 6.3.1.1p2): a type narrower than int becomes int.
// Anything else that reaches the fold (a pointer cast) counts as an unsigned long.
function cPromote(t) {
    if (t.k != "int") return ctI(8, true)
    return t.w < 4 ? ctInt : t
}
// The common type of two operands (C11 6.3.1.8p1) on this LP64 target, where
// ranks of equal width have equal width: the wider type wins with its own
// signedness, and equal widths are unsigned if either operand is.
function cCommonCt(a, b) {
    var pa = cPromote(a)
    var pb = cPromote(b)
    if (pa.w != pb.w) return pa.w > pb.w ? pa : pb
    return ctI(pa.w, pa.u == true || pb.u == true)
}
// items = [x0, {op}, x1, {op}, x2, ...] - ONE precedence level, folded left to
// right. A division by zero folds to 0.
function foldDimBin(items) {
    var x = items[0]
    var i = 1
    while (i + 1 < items.length) {
        x = cFoldBin(items[i].op, x, items[i + 1])
        i = i + 2
    }
    return x
}
// One binary operator: a comparison or a logical operator answers an int 0 or 1, a
// shift has its promoted left operand's type (C11 6.5.7p3), everything else the
// common type, the result wrapped into it.
function cFoldBin(op, a, b) {
    if (op == "&&") return cK((a.v != 0 && b.v != 0) ? 1 : 0, ctInt)
    if (op == "||") return cK((a.v != 0 || b.v != 0) ? 1 : 0, ctInt)
    if (op == "<<" || op == ">>") {
        var st = cPromote(a.t)
        var sv = op == "<<" ? a.v * Math.pow(2, b.v) : Math.floor(a.v / Math.pow(2, b.v))
        return cK(cFoldCast(sv, st), st)
    }
    var t = cCommonCt(a.t, b.t)
    var v = cFoldCast(a.v, t)
    var r = cFoldCast(b.v, t)
    if (op == "<") return cK(v < r ? 1 : 0, ctInt)
    if (op == ">") return cK(v > r ? 1 : 0, ctInt)
    if (op == "<=") return cK(v <= r ? 1 : 0, ctInt)
    if (op == ">=") return cK(v >= r ? 1 : 0, ctInt)
    if (op == "==") return cK(v == r ? 1 : 0, ctInt)
    if (op == "!=") return cK(v != r ? 1 : 0, ctInt)
    if (op == "+") { v = v + r }
    else if (op == "-") { v = v - r }
    else if (op == "*") { v = v * r }
    else if (op == "/") { v = r == 0 ? 0 : truncDiv(v, r) }
    else if (op == "%") { v = r == 0 ? 0 : v - truncDiv(v, r) * r }
    else { v = cFoldBits(op, v, r) }
    return cK(cFoldCast(v, t), t)
}
// Unary '+', '-' and '~' on the promoted operand (C11 6.5.3.3).
function cFoldUn(op, a) {
    var t = cPromote(a.t)
    var v = a.v
    if (op == "-") { v = -v }
    else if (op == "~") { v = -v - 1 }
    return cK(cFoldCast(v, t), t)
}
// '&', '|' or '^' on two integers held exactly as doubles: the high and the low 32
// bits are combined separately, since a script's bitwise operators see only 32.
function cFoldBits(op, a, b) {
    var ah = Math.floor(a / 4294967296)
    var bh = Math.floor(b / 4294967296)
    var al = a - ah * 4294967296
    var bl = b - bh * 4294967296
    var h = 0
    var l = 0
    if (op == "&") { h = ah & bh; l = al & bl }
    else if (op == "|") { h = ah | bh; l = al | bl }
    else { h = ah ^ bh; l = al ^ bl }
    if (l < 0) l = l + 4294967296
    return h * 4294967296 + l
}
// The integer conversion of C11 6.3.1.3 to the width and signedness of ct (a
// non-integer type leaves the value alone).
function cFoldCast(v, ct) {
    if (ct.bool == true) return v != 0 ? 1 : 0
    if (ct.k != "int") return v
    if (ct.w >= 8) return (ct.u == true && v < 0) ? v + Math.pow(2, 64) : v
    if (ct.w == 1) return ct.u ? (v & 255) : ((v << 24) >> 24)
    if (ct.w == 2) return ct.u ? (v & 65535) : ((v << 16) >> 16)
    return ct.u ? (v >>> 0) : (v | 0)
}
// A cast in a constant expression: the value converted to ct, which a non-integer
// type leaves as it was.
function cFoldCastK(x, ct) {
    if (ct.k != "int") return x
    return cK(cFoldCast(x.v, ct), ct.bool == true ? ctInt : ct)
}
// cond ? a : b: items = [cond] or [cond, a, b]; the result has the common type of
// a and b (C11 6.5.15p5).
function foldCond(items) {
    if (items.length == 1) return items[0]
    var t = cCommonCt(items[1].t, items[2].t)
    return cK(cFoldCast(items[0].v != 0 ? items[1].v : items[2].v, t), t)
}
// C integer division truncates toward zero, which Math.floor does not do for a
// negative quotient.
function truncDiv(a, b) {
    var q = a / b
    return q < 0 ? Math.ceil(q) : Math.floor(q)
}

// A parenthesized callee: its text (blanks already removed) and its expression
// thunk t. When the text is one name under '(' ')' '*' '&' only, the seed is
// {fname, t}: a FUNCTION's name stays a direct call, since '*' and '&' on a function
// designator give back the same function (C11 6.5.3.2p3-4), and a variable's is
// evaluated by t. Anything else is t itself.
function makeParenCallee(text, t) {
    var i = 0
    var j = text.length
    while (i < j && (text.charAt(i) == "(" || text.charAt(i) == "*" || text.charAt(i) == "&")) i = i + 1
    while (j > i && text.charAt(j - 1) == ")") j = j - 1
    if (i == j || (text.charCodeAt(i) >= 48 && text.charCodeAt(i) <= 57)) return t
    for (var k = i; k < j; k++) {
        var cc = text.charCodeAt(k)
        if (!(cc == 95 || (cc >= 48 && cc <= 57) || (cc >= 65 && cc <= 90) || (cc >= 97 && cc <= 122))) return t
    }
    return {fname: text.slice(i, j), t: t}
}

// The constants of the standard headers this front end treats as built in
// (core.stdlibImports), spelled as the reference toolchain's headers spell them on
// this target (LP64, signed char) so each one gets the C type that spelling gives
// it: INT_MIN is an int, UINT_MAX an unsigned int, ULONG_MAX an unsigned long. A
// leading '-' negates and a trailing '-1' subtracts one; a leading '(' is NULL's
// ((void*)0), the one pointer constant.
var cStdConsts = {
    "limits.h": {CHAR_BIT: "8", SCHAR_MIN: "-128", SCHAR_MAX: "127", UCHAR_MAX: "255",
        CHAR_MIN: "-128", CHAR_MAX: "127", SHRT_MIN: "-32768", SHRT_MAX: "32767",
        USHRT_MAX: "65535", INT_MIN: "-2147483647-1", INT_MAX: "2147483647",
        UINT_MAX: "0xffffffff", LONG_MIN: "-0x7fffffffffffffffL-1",
        LONG_MAX: "0x7fffffffffffffffL", ULONG_MAX: "0xffffffffffffffffUL",
        LLONG_MIN: "-0x7fffffffffffffffLL-1", LLONG_MAX: "0x7fffffffffffffffLL",
        ULLONG_MAX: "0xffffffffffffffffULL"},
    "stdint.h": {INT8_MIN: "-128", INT8_MAX: "127", UINT8_MAX: "255",
        INT16_MIN: "-32768", INT16_MAX: "32767", UINT16_MAX: "65535",
        INT32_MIN: "-2147483647-1", INT32_MAX: "2147483647", UINT32_MAX: "4294967295U",
        INT64_MIN: "-9223372036854775807LL-1", INT64_MAX: "9223372036854775807LL",
        UINT64_MAX: "18446744073709551615ULL", SIZE_MAX: "18446744073709551615UL",
        INTPTR_MIN: "-9223372036854775807L-1", INTPTR_MAX: "9223372036854775807L",
        UINTPTR_MAX: "18446744073709551615UL", INTMAX_MIN: "-9223372036854775807L-1",
        INTMAX_MAX: "9223372036854775807L", UINTMAX_MAX: "18446744073709551615UL",
        PTRDIFF_MIN: "-9223372036854775807L-1", PTRDIFF_MAX: "9223372036854775807L"},
    "stdbool.h": {"true": "1", "false": "0", __bool_true_false_are_defined: "1"},
    "stddef.h": {NULL: "((void*)0)"},
    "stdio.h": {NULL: "((void*)0)"},
    "stdlib.h": {NULL: "((void*)0)"},
    "string.h": {NULL: "((void*)0)"}
}
// The constant tables of the headers included so far.
var cStdIncluded = []
// Whether <stddef.h>, the one header defining offsetof (C11 7.19p3), was included.
var cHasOffsetof = false
// '#include <h>': its constants become visible, and <stdbool.h> defines bool as
// _Bool (C11 7.18), from here on like any typedef.
function cNoteSysHeader(h) {
    if (hasOwn(cStdConsts, h)) cStdIncluded.push(cStdConsts[h])
    if (h == "stdbool.h") typedefs["bool"] = ctBool
    if (h == "stddef.h") cHasOffsetof = true
}
// The constant thunk for a name an included standard header defines, or undefined.
// NULL is the null pointer constant cc's headers spell, ((void*)0) (C11 6.3.2.3p3).
// makeNumLit, makeNeg, foldBinary, makeCast and ctVoid come from the including grammar.
function cStdConst(name) {
    for (var i = 0; i < cStdIncluded.length; i++) {
        if (!hasOwn(cStdIncluded[i], name)) continue
        var s = cStdIncluded[i][name]
        if (s.charCodeAt(0) == 40) return makeCast([{k: "ptr", p: ctVoid}, makeNumLit(0, "0")])
        if (s.charCodeAt(0) != 45) return makeNumLit(parseCInt(s), s)
        var parts = s.slice(1).split("-")
        var t = makeNeg(makeNumLit(parseCInt(parts[0]), parts[0]))
        if (parts.length == 1) return t
        return foldBinary([t, "-", makeNumLit(parseCInt(parts[1]), parts[1])])
    }
    return undefined
}

// The value of a name in an integer constant expression folded while the tags walk
// (an array dimension, an enumerator), typed as cK: an enumeration constant
// declared earlier, an int, or a constant of an included standard header, typed as
// its spelling types it. Anything else - a variable, which would make a
// variable-length array - is refused.
function cConstOf(name) {
    var be = cBlockEnum(name)
    if (be != undefined) return cK(be, ctInt)
    if (hasOwn(enumConsts, name)) return cK(enumConsts[name] | 0, ctInt)
    for (var i = 0; i < cStdIncluded.length; i++) {
        if (!hasOwn(cStdIncluded[i], name)) continue
        var s = cStdIncluded[i][name]
        if (s.charCodeAt(0) == 40) fail("not an integer constant: " + name)
        var neg = s.charCodeAt(0) == 45
        var parts = (neg ? s.slice(1) : s).split("-")
        var x = cConstPart(name, parts[0])
        if (neg) x = cFoldUn("-", x)
        if (parts.length > 1) x = cFoldBin("-", x, cConstPart(name, parts[1]))
        return x
    }
    fail("not an integer constant: " + name)
    return cK(0, ctInt)
}
function cConstPart(name, digits) {
    var v = parseCInt(digits)
    if (typeof v != "number" || v > 9007199254740991) fail("constant too large to fold here: " + name)
    return cK(v, cLitCt(digits))
}

// The type a declared object has, for 'sizeof x' in a constant expression, which
// is folded before any declaration has run: the innermost block declaring the name
// decides (an enumerator there is an int), then the file scope.
function cDeclCtOf(name) {
    for (var i = blockNames.length - 1; i >= 0; i--) {
        if (!hasOwn(blockNames[i], name)) continue
        var e = blockNames[i][name]
        if (typeof e == "number") return ctInt
        if (e.ct == undefined) fail("not an integer constant: sizeof " + name)
        return e.ct
    }
    if (hasOwn(globalScope, name)) return globalScope[name].ct
    if (hasOwn(enumConsts, name)) return ctInt
    fail("not an integer constant: sizeof " + name)
    return ctInt
}
// items = [ct, step...]: the type an object expression's '[i]', '.f' and '->f'
// steps reach from ct; the index itself is never evaluated (C11 6.5.3.4p2).
function cStepCt(items) {
    var ct = items[0]
    for (var i = 1; i < items.length; i++) {
        var s = items[i]
        if (s.fld != undefined) {
            if (ct.k != "struct") fail("'.' needs a struct value: ." + s.fld)
            ct = fieldOf(ct.s, s.fld).ct
        } else if (s.arw != undefined) {
            if (ct.k != "ptr" || ct.p.k != "struct") fail("'->' needs a struct pointer: ->" + s.arw)
            ct = fieldOf(ct.p.s, s.arw).ct
        } else {
            ct = cDerefCt(ct)
        }
    }
    return ct
}
// The type '*' or '[i]' reaches: an array's element, a pointer's target, and a
// function pointer itself (C11 6.5.3.2p4).
function cDerefCt(ct) {
    if (ct.k == "arr") return ct.e
    if (ct.k == "ptr") return ct.p
    if (ct.k == "fnp") return ct
    fail("indexing needs an array or a pointer")
    return ct
}

// One brace-initializer item: an optional designator chain ('[i]', '.field',
// '.b[1]') plus its value. The first designator picks the member of the list's
// own object (at / fld); the rest (sub) walk into that member.
function makeDItem(items) {
    var it = {v: items[items.length - 1]}
    if (items.length > 1) {
        if (items[0].fld != undefined) { it.fld = items[0].fld } else { it.at = items[0].at }
        if (items.length > 2) it.sub = items.slice(1, items.length - 1)
    }
    return it
}
// Lay the items of one brace list over an array or struct/union of type ct (C11
// 6.7.9p17-21) and call put(steps, v) for each item in order, where steps is the
// designator chain ([{at} | {fld}]) from the object to the subobject the item
// initializes. The current object is a stack of frames {ct, i}, i being the next
// member: a designation rebuilds the stack from the outermost object, and an item
// that lands on an aggregate without being a brace list or a whole value of it
// (whole(v, ct) for an expression, a string literal on a character array) descends
// into its first member - brace elision. A filled frame pops back to its parent.
// Returns how many elements of the outermost object were reached, which sizes an
// array declared with '[]' (ct.n < 0 is unbounded).
function cInitWalk(ct, list, whole, put) {
    var stack = [{ct: ct, i: 0}]
    var reach = 0
    for (var k = 0; k < list.length; k++) {
        var it = list[k]
        var v = it.v
        if (it.at != undefined || it.fld != undefined) {
            while (stack.length > 1) stack.pop()
            cInitSeek(stack, it)
            var sub = it.sub != undefined ? it.sub : []
            for (var d = 0; d < sub.length; d++) {
                stack.push({ct: cInitChild(stack[stack.length - 1]), i: 0})
                cInitSeek(stack, sub[d])
            }
        } else {
            while (stack[stack.length - 1].i >= cInitSize(stack[stack.length - 1].ct)) {
                if (stack.length == 1) {
                    fail(ct.k == "arr" ? "too many initializers for an array" : "too many initializers for " + tagName(ct.s))
                }
                stack.pop()
                cInitAdvance(stack[stack.length - 1])
            }
        }
        while (true) {
            var cct = cInitChild(stack[stack.length - 1])
            if (!isAggCt(cct)) break
            if (typeof v == "function") {
                if (whole(v, cct)) break
            } else if (v.list != undefined || (cct.k == "arr" && cct.e.k != "ptr" && !isAggCt(cct.e))) {
                break
            }
            if (cInitSize(cct) == 0) fail("an initializer for an empty aggregate")
            stack.push({ct: cct, i: 0})
        }
        if (stack[0].i + 1 > reach) reach = stack[0].i + 1
        var steps = []
        for (var s = 0; s < stack.length; s++) {
            var f = stack[s]
            steps.push(f.ct.k == "arr" ? {at: f.i} : {fld: structOf(f.ct.s).order[f.i]})
        }
        put(steps, v)
        cInitAdvance(stack[stack.length - 1])
    }
    return reach
}
// How many elements an initializer gives an array declared with '[]' first: a
// string literal's characters, or the outermost elements a brace list reaches.
// No code runs this early, so an expression on an aggregate element is counted as
// eliding into it, and the list is marked for cInitWholeOf to hold that guess to.
function initCount(init, dims, base) {
    if ((dims[0] | 0) >= 0 || init == undefined || typeof init == "function") return 0
    if (init.str != undefined) return init.str.length
    return cInitWalk(ctArr(-1, arrCt(dims.slice(1), base, 0)), init.list,
                     function(v, ct) { init.elides = true; return false }, function(steps, v) {})
}
// An automatic array sized by that guess, once its declaration runs and whole -
// the engine's test for an expression that is a whole aggregate element - can
// look at the expressions: its real type, and the list to initialize it with.
// Answers {ct, init}.
function cInitResize(ct, init, whole) {
    if (init == undefined || typeof init == "function" || init.elides != true) return {ct: ct, init: init}
    var n = cInitWalk(ctArr(-1, ct.e), init.list, whole, function(steps, v) {})
    return {ct: ctArr(n, ct.e), init: {list: init.list}}
}
// The whole-value test for placing a brace list: the engine's own, unless the
// list sized its array assuming an expression elides and was not resized
// (a static object), where a struct value that would take a whole element
// instead is refused.
function cInitWholeOf(init, whole) {
    if (init.elides != true) return whole
    return function(v, ct) {
        if (whole(v, ct)) fail("a struct value in an array sized by its initializer is not supported")
        return false
    }
}
// How many members a frame walks: an array's elements (unbounded while its size is
// still being inferred), a struct's members in declaration order.
function cInitSize(ct) {
    if (ct.k == "arr") return ct.n < 0 ? 4294967296 : ct.n
    return structOf(ct.s).order.length
}
// The type of the member a frame is at.
function cInitChild(f) {
    if (f.ct.k == "arr") return f.ct.e
    return fieldOf(f.ct.s, structOf(f.ct.s).order[f.i]).ct
}
// Past the member just initialized; a union holds only one, so it is then full.
function cInitAdvance(f) {
    if (f.ct.k != "arr" && structOf(f.ct.s).isUnion) { f.i = cInitSize(f.ct); return }
    f.i = f.i + 1
}
// The member index one designator ('[i]' or '.name') picks in an object of type ct.
// Point the top frame at what one designator names. A member of an ANONYMOUS
// struct or union is reached through it (C11 6.7.2.1p13): the frame is pointed at
// the anonymous member and a frame for that member is pushed, so the items after
// '.u' carry on inside it.
function cInitSeek(stack, d) {
    var f = stack[stack.length - 1]
    if (d.fld != undefined && f.ct.k == "struct") {
        var st = structOf(f.ct.s)
        for (var k = 0; k < st.order.length; k++) {
            var m = st.fields[st.order[k]]
            if (m.anon == true && hasOwn(structOf(m.ct.s).fields, d.fld)) {
                f.i = k
                stack.push({ct: m.ct, i: 0})
                cInitSeek(stack, d)
                return
            }
        }
    }
    f.i = cInitIndex(f.ct, d)
}
function cInitIndex(ct, d) {
    if (d.fld != undefined) {
        if (ct.k != "struct") fail("a '." + d.fld + "' designator needs a struct or union")
        var order = structOf(ct.s).order
        for (var k = 0; k < order.length; k++) { if (order[k] == d.fld) return k }
        fail(tagName(ct.s) + " has no field " + d.fld)
    }
    if (ct.k != "arr") fail("a '[" + d.at + "]' designator needs an array")
    if ((d.at | 0) >= cInitSize(ct)) fail("too many initializers for an array")
    return d.at | 0
}

// The byte size and alignment cc gives a type on this target (LP64 arm64, where
// long double is a double): what sizeof and _Alignof answer. Neither engine's
// storage has to be this shape - see ctSize in c-interpreter.abnf and machBytes in
// c-to-llvm-ir.abnf.
function ctBytes(ct) {
    if (ct.k == "ptr" || ct.k == "fnp") return 8
    if (ct.k == "struct") return structOf(ct.s).bytes
    if (ct.k == "arr") return ct.n * ctBytes(ct.e)
    if (ct.k == "void") return 1    // GCC's sizeof(void), which cc also answers
    return ct.w
}
function ctAlign(ct) {
    if (ct.k == "struct") return structOf(ct.s).align
    if (ct.k == "arr") return ctAlign(ct.e)
    return ctBytes(ct)
}
function cRoundUp(n, a) { return n % a == 0 ? n : n + a - n % a }
// cc's layout of a struct or union (C11 6.7.2.1 as the arm64 ABI fixes it): each
// member at the next multiple of its alignment; a bit-field packed into the bits
// that follow unless it would straddle a unit of its declared type, when it starts
// the next unit; a zero-width one only rounding up to such a unit; every member of
// a union at 0. The alignment is the largest member's - an UNNAMED bit-field counts
// for nothing - and the size is rounded up to it. ms: [{size, align, bits?, named,
// name?, offs?}]. Answers {bytes, align, offs}: offs lists each named member's byte
// offset as {n, o} (o is -1 for a bit-field), the members of an anonymous struct or
// union member (its own offs) shifted to the enclosing tag's start.
function cAbiLayout(ms, isUnion) {
    var bit = 0
    var end = 0
    var align = 1
    var offs = []
    for (var i = 0; i < ms.length; i++) {
        var m = ms[i]
        var at = isUnion ? 0 : bit
        var w = m.size * 8
        if (m.bits == undefined) { at = cRoundUp(at, m.align * 8) }
        else {
            w = m.bits
            var unit = m.size * 8
            if (w == 0 || (at - at % unit) != (at + w - 1) - (at + w - 1) % unit) at = cRoundUp(at, m.align * 8)
        }
        if (m.named && m.align > align) align = m.align
        if (at + w > end) end = at + w
        bit = at + w
        if (m.offs != undefined) {
            for (var k = 0; k < m.offs.length; k++) {
                var so = m.offs[k].o
                offs.push({n: m.offs[k].n, o: so < 0 ? so : at / 8 + so})
            }
        } else if (m.named) {
            offs.push({n: m.name, o: m.bits == undefined ? at / 8 : -1})
        }
    }
    return {bytes: cRoundUp(cRoundUp(end, 8) / 8, align), align: align, offs: offs}
}
// One member of a cAbiLayout list: a declared field of type ct, or (sub) the
// {bytes, align, offs} of an anonymous struct/union member.
function cAbiMember(ct, f) { return {size: ctBytes(ct), align: ctAlign(ct), bits: f.bits, named: f.pad != true, name: f.name} }
function cAbiSub(sub) { return {size: sub.bytes, align: sub.align, named: true, offs: sub.offs} }

// offsetof(type, member-designator): the byte offset cc gives the designated member
// (C11 7.19p3), read from the tag's cAbiLayout offsets, so it answers cc's bytes
// whatever either engine's own storage looks like. steps: [{fld} | {at}], a member
// followed by '.member' and '[index]' steps.
function cOffsetOf(ct, steps) {
    if (!cHasOffsetof) fail("offsetof needs #include <stddef.h>")
    var off = 0
    for (var i = 0; i < steps.length; i++) {
        var d = steps[i]
        if (d.fld == undefined) {
            if (ct.k != "arr") fail("offsetof: subscript of a non-array")
            ct = ct.e
            off = off + d.at * ctBytes(ct)
            continue
        }
        if (ct.k != "struct") fail("offsetof: '" + d.fld + "' needs a struct or union")
        var st = structOf(ct.s)
        var o = -2
        for (var k = 0; k < st.offs.length; k++) { if (st.offs[k].n == d.fld) o = st.offs[k].o }
        if (o == -2) fail(tagName(ct.s) + " has no field " + d.fld)
        if (o == -1) fail("cannot compute offset of bit-field '" + d.fld + "'")
        off = off + o
        ct = st.fields[d.fld].ct
    }
    return off
}

// The names declared so far in each enclosing block, innermost last. The tags
// fill it while they walk the tree, so a second declaration of one name in one
// block (C11 6.7p3) is a compile error even in code that never runs; a nested
// block opens a table of its own, which is what keeps shadowing legal. Only two
// extern declarations may repeat a name, since both denote the one object with
// linkage (6.7p3).
var blockNames = []
function openBlockScope() { blockNames.push({}) }
function closeBlockScope() { blockNames.pop() }
// Two declarations of one object with linkage must have compatible types (C11
// 6.7p4): the same type, or arrays of one element type where one has no size
// ('extern int a[];' then 'int a[3];'), which together give the sized type
// (6.2.7p3). Answers the type the object has from now on; fails on a conflict.
function cRedeclCt(name, prev, ct) {
    if (ctEq(prev, ct)) return prev
    if (prev.k == "arr" && ct.k == "arr" && (prev.n == 0 || ct.n == 0) && ctEq(prev.e, ct.e)) {
        return prev.n == 0 ? ct : prev
    }
    fail("conflicting types for '" + name + "'")
    return prev
}
// The parameters of the parameter list walked last, which a function body that
// follows it declares in its own outermost block.
var fnParams = []
function cParamList(items) { fnParams = items; return items }
function openFnScope() {
    openBlockScope()
    for (var i = 0; i < fnParams.length; i++) noteBlockName(fnParams[i].name, false, fnParams[i].ct)
}
// A declared object's entry is {ext, ct}: whether it was an extern declaration,
// and its type (cDeclCtOf).
function noteBlockName(name, ext, ct) {
    if (blockNames.length == 0) { return }
    var top = blockNames[blockNames.length - 1]
    if (hasOwn(top, name) && !(ext && typeof top[name] != "number" && top[name].ext == true)) {
        fail("redefinition of '" + name + "'")
    }
    top[name] = {ext: ext, ct: ct}
}
// An enumeration constant is an ordinary identifier scoped like any declaration
// (C11 6.2.1p4, 6.7.2.2p3): a file-scope one lives in enumConsts, a block-scope one
// in its block's name table, as its value. The values run from 0, or on from an
// explicit '= n'; the counter is updated as each enumerator reduces so that the
// next initializer can name it ('A = 4, B = A + 1').
var enumConsts = {}
var enumNext = 0
function enumStart() { enumNext = 0 }
// items = [name, {val}?]
function makeEnumerator(items) {
    var v = enumNext
    if (items.length > 1) { v = items[1].val | 0 }
    enumNext = v + 1
    if (blockNames.length == 0) { enumConsts[items[0]] = v; return }
    noteBlockName(items[0], false, ctInt)
    blockNames[blockNames.length - 1][items[0]] = v
}
// The value of the block-scope enumeration constant a name denotes here, or
// undefined: the innermost block declaring the name decides, and an object
// declared there hides any enumerator further out.
function cBlockEnum(name) {
    for (var i = blockNames.length - 1; i >= 0; i--) {
        if (hasOwn(blockNames[i], name)) {
            var e = blockNames[i][name]
            return typeof e == "number" ? e : undefined
        }
    }
    return undefined
}

// Whether a declaration's keyword run ('extern int', 'const extern long') holds
// the extern storage class. up.in arrives with the separators removed, so the
// run is read keyword by keyword and the scan stops at anything else.
var declKws = ["extern", "const", "volatile", "unsigned", "signed", "short", "long",
               "int", "char", "double", "float", "_Bool", "void"]
function isExternTy(text) {
    var i = 0
    while (i < text.length) {
        var hit = ""
        for (var k = 0; k < declKws.length; k++) {
            if (hit == "" && text.slice(i, i + declKws[k].length) == declKws[k]) { hit = declKws[k] }
        }
        if (hit == "") { return false }
        if (hit == "extern") { return true }
        i = i + hit.length
    }
    return false
}
