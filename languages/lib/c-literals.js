// c-literals.js -- what c-interpreter.abnf and c-to-llvm-ir.abnf share (include):
// the values of C string literals, character constants and the standard headers'
// constants, brace-initializer items, and the block-scope name table. digVal,
// fail and hasOwn come from the including grammar.
//
// The execution character set is UTF-8, as cc's: a narrow literal holds the
// UTF-8 bytes of every character outside ASCII and of every \u / \U universal
// character name (C11 6.4.3), and a wide one (L, u, U) one element per code point.

// The code point that starts at text[i]: a UTF-16 surrogate pair joins into one.
function cCodePointAt(text, i) {
    var hi = text.charCodeAt(i)
    if (hi >= 55296 && hi <= 56319 && i + 1 < text.length) {
        var lo = text.charCodeAt(i + 1)
        if (lo >= 56320 && lo <= 57343) return (hi - 55296) * 1024 + (lo - 56320) + 65536
    }
    return hi
}
// Append one code point: its UTF-8 bytes to a narrow literal, itself to a wide one.
function cPushCodePoint(bytes, cp, wide) {
    if (wide || cp < 128) { bytes.push(cp); return }
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
// with the C escape sequences decoded and the terminating null appended. An
// L / u / U prefix makes the elements wide.
function cStrBytes(text) {
    var bytes = []
    var wide = false
    var i = 0
    while (i < text.length) {
        var cc = text.charCodeAt(i)
        if (cc == 117 && i + 1 < text.length && text.charCodeAt(i + 1) == 56) { i = i + 2; continue }
        if (cc == 76 || cc == 117 || cc == 85) { wide = true; i = i + 1; continue }
        if (cc != 34) { i = i + 1; continue }
        i = i + 1
        while (i < text.length && text.charCodeAt(i) != 34) {
            if (text.charCodeAt(i) != 92) {
                var cp = cCodePointAt(text, i)
                cPushCodePoint(bytes, cp, wide)
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
            if (e == 117 || e == 85) { cPushCodePoint(bytes, cCharCode(text.slice(i, j)), wide) }
            else { bytes.push(cCharCode(text.slice(i, j))) }
            i = j
        }
        i = i + 1
    }
    bytes.push(0)
    return {str: bytes, wide: wide}
}
// The value of one character-constant body, including the C escape sequences:
// backslash x with hex digits, backslash with up to three octal digits, the
// universal character names \u and \U (a code point), and the simple escapes. A
// byte is answered unsigned; cCharConst gives it the type of a constant.
function cCharCode(s) {
    if (s.charCodeAt(0) != 92) return s.charCodeAt(0)
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
// An integer character constant is an int holding the value of a char (C11
// 6.4.4.4p10), and char is signed on this target: '\377' is -1.
function cCharConst(s) {
    var v = cCharCode(s)
    return v > 127 ? v - 256 : v
}

// The constants of the standard headers this front end treats as built in
// (core.stdlibImports), spelled as the reference toolchain's headers spell them on
// this target (LP64, signed char) so each one gets the C type that spelling gives
// it: INT_MIN is an int, UINT_MAX an unsigned int, ULONG_MAX an unsigned long. A
// leading '-' negates and a trailing '-1' subtracts one, the only two shapes used.
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
    "stdbool.h": {"true": "1", "false": "0", __bool_true_false_are_defined: "1"}
}
// The constant tables of the headers included so far.
var cStdIncluded = []
// '#include <h>': its constants become visible, and <stdbool.h> defines bool as
// _Bool (C11 7.18), from here on like any typedef.
function cNoteSysHeader(h) {
    if (hasOwn(cStdConsts, h)) cStdIncluded.push(cStdConsts[h])
    if (h == "stdbool.h") typedefs["bool"] = ctBool
}
// The constant thunk for a name an included standard header defines, or undefined.
// makeNumLit, makeNeg and foldBinary come from the including grammar.
function cStdConst(name) {
    for (var i = 0; i < cStdIncluded.length; i++) {
        if (!hasOwn(cStdIncluded[i], name)) continue
        var s = cStdIncluded[i][name]
        if (s.charCodeAt(0) != 45) return makeNumLit(parseCInt(s), s)
        var parts = s.slice(1).split("-")
        var t = makeNeg(makeNumLit(parseCInt(parts[0]), parts[0]))
        if (parts.length == 1) return t
        return foldBinary([t, "-", makeNumLit(parseCInt(parts[1]), parts[1])])
    }
    return undefined
}

// The value of a name in an integer constant expression folded while the tags walk
// (an array dimension, an enumerator): an enumeration constant declared earlier, or
// a constant of an included standard header. Anything else - a variable, which
// would make a variable-length array - is refused.
function cConstOf(name) {
    if (hasOwn(enumConsts, name)) return enumConsts[name] | 0
    for (var i = 0; i < cStdIncluded.length; i++) {
        if (!hasOwn(cStdIncluded[i], name)) continue
        var s = cStdIncluded[i][name]
        var neg = s.charCodeAt(0) == 45
        var parts = (neg ? s.slice(1) : s).split("-")
        var v = cConstPart(name, parts[0])
        if (neg) v = 0 - v
        if (parts.length > 1) v = v - cConstPart(name, parts[1])
        return v
    }
    fail("not an integer constant: " + name)
    return 0
}
function cConstPart(name, digits) {
    var v = parseCInt(digits)
    if (typeof v != "number" || v > 9007199254740991) fail("constant too large to fold here: " + name)
    return v
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
            stack[0].i = cInitIndex(ct, it)
            var sub = it.sub != undefined ? it.sub : []
            for (var d = 0; d < sub.length; d++) {
                var dct = cInitChild(stack[stack.length - 1])
                stack.push({ct: dct, i: cInitIndex(dct, sub[d])})
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
// The whole-value test for placing a brace list: the engine's own, unless the
// list sized its array assuming an expression elides, where a struct value that
// would take a whole element instead is refused.
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
    for (var i = 0; i < fnParams.length; i++) noteBlockName(fnParams[i].name, false)
}
function noteBlockName(name, ext) {
    if (blockNames.length == 0) { return }
    var top = blockNames[blockNames.length - 1]
    if (hasOwn(top, name) && !(ext && top[name] == "ext")) fail("redefinition of '" + name + "'")
    top[name] = ext ? "ext" : "obj"
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
