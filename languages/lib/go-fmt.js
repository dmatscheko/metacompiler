// go-fmt.js -- Go's fmt.Sprintf: the format-string parser, the flags, width and
// precision, and every verb of fmt's documentation.
//
// Written once for the interpreter half (go-interpreter.abnf includes it) and for
// layer 2 (languages/lib/go-rt.metajs imports it), in the restricted subset both
// accept - see the head of regex.js. abnf/jsrtgolang.go is the Go twin the
// compiler half runs under llvm.Run; it hands each scalar to Go's own fmt, so the
// two answer alike only while this file stays a faithful port of fmt/print.go,
// fmt/format.go and strconv's ftoa.go. The names follow those files.
//
// The value model differs between the engines, so everything that touches a value
// goes through a HOOK the includer defines:
//   gfHKind(v)          "nil" "bool" "string" "int" "float" "complex" "list" "map"
//                       "struct" or "other" ("int" is every integer type)
//   gfHTy(v)            the type name a bad-verb note shows for a scalar, else ""
//   gfHIntMag(v)        an integer's magnitude as decimal digits, exactly
//   gfHIntNeg(v)        whether an integer is negative
//   gfHNum(v)           the double a float (or a small integer) holds
//   gfHF32(v)           whether a float is a float32
//   gfHCx(v)            a complex value's parts, [re, im]
//   gfHElems(v)         a slice's or an array's elements
//   gfHIsBytes(v)       whether a slice or an array is a []byte or [n]byte (a
//                       list without an element type: gfAllBytes)
//   gfHMapKV(v)         a map as [keys, values], in fmt's (sorted) print order
//   gfHFields(v)        a struct as [field names, field values]
//   gfHMeth(v)          the text of v's Error() or else String() method, or
//                       undefined when it has neither
//   gfHStr(v)           fmt's %v text of v
//   gfHShort(a, f32)    the shortest decimal of a > 0 as DIGITS "e" EXP, EXP the
//                       exponent of the first digit
//   gfHExact(a)         the exact decimal of a > 0 as {i: digits, f: digits}
//   gfHRound(ds, n)     ds rounded to n digits half to even (one longer on carry)
//   gfHBytes(s)         a string's UTF-8 bytes
//   gfHBytesStr(bs)     the string whose UTF-8 bytes are bs
//   gfHQuote(s, delim)  the body of strconv.Quote / QuoteRune (delim '"' or "'")
//   gfHRune(cp)         the string of a Unicode scalar value
//   gfHStructName(v)    a named struct's type as reflect spells it ("main.T"), or
//                       "" for an anonymous struct
//   gfHFTypes(v)        a struct's declared field types, as written ("" unknown)
//   gfHElemTy(v)        a slice's element type as written, or "" when the value
//                       does not record it
//   gfHIsArray(v)       whether a list is a fixed-length array
//   gfHMapTy(v)         a map's type as written ("map[K]V"), or "" when unrecorded
//   gfHChanTy(v)        a channel's element type as written, "" for a non-channel
//   gfHRef(v)           what %p takes an address from: [] for a value that is
//                       not a reference, else [object, key, offset] (see gfAddr)
//
// What the value model cannot show, shared with the twin: a pointer to a struct is
// the struct (%#v writes main.T{...} where go writes &main.T{...}), a defined
// non-struct type is its underlying type, an int64 or uint64 reads int or uint,
// and a function has no signature for %T to print.

// The flags, width and precision of one directive. f.plusV is fmt's own split: on
// %v, '+' asks for field names and is not a numeric flag any more ('#' asks for Go
// syntax, which is not rendered: it is dropped). u8 marks the elements of a byte
// slice, which a bad-verb note names uint8 even where the engine holds them as
// plain numbers.
function gfState() {
    return {sharp: false, zero: false, plus: false, minus: false, space: false,
            wid: 0, widPresent: false, prec: 0, precPresent: false, plusV: false, sharpV: false, u8: false}
}

function gfCopy(f) {
    return {sharp: f.sharp, zero: f.zero, plus: f.plus, minus: f.minus, space: f.space,
            wid: f.wid, widPresent: f.widPresent, prec: f.prec, precPresent: f.precPresent,
            plusV: f.plusV, sharpV: f.sharpV, u8: f.u8}
}

// The integer a '*' takes from its operand, or undefined when the operand is not
// an integer or is out of fmt's range (1e6).
function gfIntArg(v) {
    if (gfHKind(v) != "int") { return undefined }
    var mag = gfHIntMag(v)
    if (mag.length > 7) { return undefined }
    var n = parseInt(mag, 10)
    if (n > 1000000) { return undefined }
    return gfHIntNeg(v) ? 0 - n : n
}

function gfDigitAt(s, i) {
    if (i >= s.length) { return 0 - 1 }
    var c = s.charCodeAt(i)
    if (c < 48 || c > 57) { return 0 - 1 }
    return c - 48
}

// fmt.Sprintf. A verb is one UTF-16 unit, or two for an astral character.
function gfFormat(format, args) {
    var out = ""
    var argi = 0
    var n = format.length
    var i = 0
    while (i < n) {
        var c = format.charAt(i)
        if (c != "%") { out = out + c; i = i + 1; continue }
        i = i + 1
        var f = gfState()
        while (i < n) {
            var fc = format.charAt(i)
            if (fc == "#") { f.sharp = true }
            else if (fc == "0") { f.zero = !f.minus }
            else if (fc == "+") { f.plus = true }
            else if (fc == "-") { f.minus = true; f.zero = false }
            else if (fc == " ") { f.space = true }
            else { break }
            i = i + 1
        }
        if (i < n && format.charAt(i) == "*") {
            i = i + 1
            var wa = anytype
            wa = undefined
            if (argi < args.length) { wa = gfIntArg(args[argi]); argi = argi + 1 }
            if (wa === undefined) { out = out + "%!(BADWIDTH)" }
            else {
                f.widPresent = true
                f.wid = wa
                if (f.wid < 0) { f.minus = true; f.zero = false; f.wid = 0 - f.wid }
            }
        } else {
            while (gfDigitAt(format, i) >= 0) {
                f.wid = f.wid * 10 + gfDigitAt(format, i)
                f.widPresent = true
                i = i + 1
            }
        }
        if (i < n && format.charAt(i) == ".") {
            i = i + 1
            f.precPresent = true
            if (i < n && format.charAt(i) == "*") {
                i = i + 1
                var pa = anytype
                pa = undefined
                if (argi < args.length) { pa = gfIntArg(args[argi]); argi = argi + 1 }
                if (pa === undefined) { f.precPresent = false; out = out + "%!(BADPREC)" }
                else if (pa < 0) { f.precPresent = false }
                else { f.prec = pa }
            } else {
                while (gfDigitAt(format, i) >= 0) {
                    f.prec = f.prec * 10 + gfDigitAt(format, i)
                    i = i + 1
                }
            }
        }
        if (i >= n) { out = out + "%!(NOVERB)"; break }
        var verb = format.charAt(i)
        var vc = format.charCodeAt(i)
        if (vc >= 55296 && vc <= 56319 && i + 1 < n) { verb = format.substring(i, i + 2) }
        i = i + verb.length
        if (verb == "%") { out = out + "%"; continue }
        if (argi >= args.length) { out = out + "%!" + verb + "(MISSING)"; continue }
        if (verb == "T") { out = out + gfTypeVerb(f, args[argi]) }
        else if (verb == "p") { out = out + gfPointer(f, args[argi]) }
        else {
            if (verb == "v") {
                f.plusV = f.plus
                f.plus = false
                f.sharpV = f.sharp
                f.sharp = false
            }
            if (f.sharpV) { out = out + gfGoSyn(f, args[argi], "", 0) }
            else { out = out + gfArg(f, args[argi], verb, 0, false) }
        }
        argi = argi + 1
    }
    // Operands no directive used: %!(EXTRA type=value, ...).
    if (argi < args.length) {
        out = out + "%!(EXTRA "
        var first = argi
        while (argi < args.length) {
            if (argi > first) { out = out + ", " }
            var a = anytype
            a = args[argi]
            if (gfHKind(a) == "nil") { out = out + "<nil>" }
            else { out = out + gfTyPrefix(a) + gfArg(gfState(), a, "v", 0, false) }
            argi = argi + 1
        }
        out = out + ")"
    }
    return out
}

// The "type=" a bad-verb or EXTRA note puts before a value, "" when the type is
// not known.
function gfTyPrefix(v) {
    var t = gfTypeName(v)
    if (t == "") { return "" }
    return t + "="
}

// %T: the operand's type, under the width and precision of a string.
function gfTypeVerb(f, v) {
    if (gfHKind(v) == "nil") { return gfPad(f, "<nil>") }
    var t = gfTypeName(v)
    if (t == "") { return gfBad(f, "T", v) }
    return gfPad(f, gfTrunc(f, t))
}

// The addresses %p prints. This value model has none, so each distinct referent
// is given one, in the order %p first meets it: a fixed base, a page apart, plus
// the referent's own offset (a slice's element offset). gfHRef(v) answers [] for
// an operand that is not a pointer, a map, a slice, a function or a channel, and
// otherwise [object, key, offset] with object null for a nil one; key tells two
// variables of one scope apart.
var GF_AOBJ = []
var GF_AKEY = []
function gfAddr(o, k) {
    var i = 0
    while (i < GF_AOBJ.length) {
        if (GF_AOBJ[i] === o && GF_AKEY[i] == k) { return 824633786368 + i * 4096 }
        i = i + 1
    }
    GF_AOBJ.push(o)
    GF_AKEY.push(k)
    return 824633786368 + i * 4096
}

// What %v prints for a function or a channel: its address, as fmtPointer does.
function gfAddrText(o) { return "0x" + gfToBase(gfIntText(gfAddr(o, "")), 16, false) }

// fmtPointer: the address in hex, with 0x unless # asks to drop it.
function gfPointer(f, v) {
    if (gfHKind(v) == "nil") { return gfBad(f, "p", v) }
    var r = gfHRef(v)
    if (r.length == 0) { return gfBad(f, "p", v) }
    var a = 0
    if (r[0] !== null) { a = gfAddr(r[0], r[1]) + r[2] }
    var g = gfCopy(f)
    g.sharp = !f.sharp
    return gfInteger(g, a, "x")
}

// ----- padding ------------------------------------------------------------------

function gfRepeat(c, n) {
    var out = ""
    var k = 0
    while (k < n) { out = out + c; k = k + 1 }
    return out
}

// Width counts RUNES: a surrogate pair is one.
function gfRuneCount(s) {
    var n = 0
    var i = 0
    while (i < s.length) {
        var c = s.charCodeAt(i)
        if (c >= 55296 && c <= 56319 && i + 1 < s.length) { i = i + 1 }
        n = n + 1
        i = i + 1
    }
    return n
}

// f.pad: to the left with zeros or spaces, to the right (f.minus) with spaces.
function gfPad(f, s) {
    if (!f.widPresent || f.wid == 0) { return s }
    var w = f.wid - gfRuneCount(s)
    if (w <= 0) { return s }
    if (f.minus) { return s + gfRepeat(" ", w) }
    return gfRepeat(f.zero ? "0" : " ", w) + s
}

// f.pad with the zero flag ignored, which is what the numeric formatters do once
// they have placed their own zeros.
function gfPadSp(f, s) {
    if (!f.widPresent || f.wid == 0) { return s }
    var w = f.wid - gfRuneCount(s)
    if (w <= 0) { return s }
    if (f.minus) { return s + gfRepeat(" ", w) }
    return gfRepeat(" ", w) + s
}

// ----- one operand ----------------------------------------------------------------

// printArg and printValue in one: v formatted under verb. depth > 0 is an element
// of a composite, and noMeth says it was reached through an UNEXPORTED field, where
// fmt cannot call a method (reflect's CanInterface is false there and below).
function gfArg(f, v, verb, depth, noMeth) {
    var k = gfHKind(v)
    // %v with nothing to apply is the engine's own rendering.
    if (verb == "v" && !f.widPresent && !f.precPresent && !f.space && !f.plus && !f.sharp && !f.plusV) { return gfHStr(v) }
    if (k == "nil") {
        if (verb == "v" || depth > 0) { return depth > 0 ? "<nil>" : gfPad(f, "<nil>") }
        return gfBad(f, verb, v)
    }
    if (!noMeth && (verb == "v" || verb == "s" || verb == "x" || verb == "X" || verb == "q")) {
        var ms = anytype
        ms = gfHMeth(v)
        if (ms !== undefined) { return gfString(f, ms, verb) }
    }
    if (k == "bool") {
        if (verb == "t" || verb == "v") { return gfPad(f, v === true ? "true" : "false") }
        return gfBad(f, verb, v)
    }
    if (k == "int") { return gfInteger(f, v, verb) }
    if (k == "float") { return gfFloatVerb(f, gfHNum(v), gfHF32(v), verb, v) }
    if (k == "complex") { return gfComplex(f, v, verb) }
    if (k == "string") { return gfString(f, v, verb) }
    if (k == "list") { return gfList(f, v, verb, depth, noMeth) }
    if (k == "map") {
        var kv = gfHMapKV(v)
        var out = "map["
        var i = 0
        while (i < kv[0].length) {
            if (i > 0) { out = out + " " }
            out = out + gfArg(f, kv[0][i], verb, depth + 1, noMeth) + ":" + gfArg(f, kv[1][i], verb, depth + 1, noMeth)
            i = i + 1
        }
        return out + "]"
    }
    if (k == "struct") {
        var fs = gfHFields(v)
        var outs = "{"
        var j = 0
        while (j < fs[0].length) {
            if (j > 0) { outs = outs + " " }
            var name = fs[0][j]
            if (f.plusV) { outs = outs + name + ":" }
            outs = outs + gfArg(f, fs[1][j], verb, depth + 1, noMeth || !gfExported(name))
            j = j + 1
        }
        return outs + "}"
    }
    if (verb == "v") { return gfPad(f, gfHStr(v)) }
    return gfBad(f, verb, v)
}

function gfExported(name) {
    if (name.length == 0) { return false }
    var c = name.charCodeAt(0)
    return c >= 65 && c <= 90
}

// p.badVerb: %!verb(type=value), the value printed with %v under the same flags.
function gfBad(f, verb, v) {
    var ty = gfHTy(v)
    if (f.u8 && ty == "int") { ty = "uint8" }
    if (ty == "") { ty = gfTypeName(v) }
    if (gfHKind(v) == "nil") { return "%!" + verb + "(<nil>)" }
    var g = gfCopy(f)
    g.plusV = false
    var shown = gfArg(g, v, "v", 0, true)
    if (ty == "") { return "%!" + verb + "(" + shown + ")" }
    return "%!" + verb + "(" + ty + "=" + shown + ")"
}

// A slice or an array: every element under the verb, except that a byte slice is
// a STRING to %s %q %x %X (fmtBytes).
function gfList(f, v, verb, depth, noMeth) {
    var es = gfHElems(v)
    var g = f
    if (gfHIsBytes(v)) {
        if (verb == "s" || verb == "q" || verb == "x" || verb == "X") {
            var bs = []
            var b = 0
            while (b < es.length) { bs.push(gfHNum(es[b])); b = b + 1 }
            if (verb == "s") { return gfPad(f, gfTrunc(f, gfHBytesStr(bs))) }
            if (verb == "q") { return gfQuote(f, gfHBytesStr(bs)) }
            return gfHex(f, bs, verb == "X")
        }
        g = gfCopy(f)
        g.u8 = true
    }
    var out = "["
    var i = 0
    while (i < es.length) {
        if (i > 0) { out = out + " " }
        out = out + gfArg(g, es[i], verb, depth + 1, noMeth)
        i = i + 1
    }
    return out + "]"
}

// Every element a byte: how a list that records no element type is recognised.
function gfAllBytes(es) {
    if (es.length == 0) { return false }
    var i = 0
    while (i < es.length) {
        if (gfHTy(es[i]) != "uint8") { return false }
        i = i + 1
    }
    return true
}

// ----- types --------------------------------------------------------------------

// The names a type text leaves unqualified: the predeclared types and the type
// keywords. Space-delimited so a lookup is one indexOf.
var GF_PREDECL = " bool string int int8 int16 int32 int64 uint uint8 uint16 uint32 uint64 uintptr float32 float64 complex64 complex128 error map chan func struct interface "

function gfIdChar(c) { return (c >= 97 && c <= 122) || (c >= 65 && c <= 90) || c == 95 || (c >= 48 && c <= 57) || c > 127 }

// A type as written in package main, respelled the way reflect prints it: a
// declared name gets its package ("T" -> "main.T", "[]*T" -> "[]*main.T"), the
// aliases byte, rune and any are their targets, and interface{} gains its space,
// as does a channel of a predeclared type (a type text arrives with its blanks
// stripped, "chanint").
// A name already qualified ("pkga.T") is kept.
function gfQualTy(t) {
    var out = ""
    var i = 0
    while (i < t.length) {
        var c = t.charCodeAt(i)
        if (!gfIdChar(c) || (c >= 48 && c <= 57)) { out = out + t.charAt(i); i = i + 1; continue }
        var j = i
        while (j < t.length && gfIdChar(t.charCodeAt(j))) { j = j + 1 }
        var id = t.substring(i, j)
        var dotted = (i > 0 && t.charCodeAt(i - 1) == 46) || (j < t.length && t.charCodeAt(j) == 46)
        if (id == "byte") { id = "uint8" }
        else if (id == "rune") { id = "int32" }
        else if (id == "any" || (id == "interface" && t.substring(j, j + 2) == "{}")) {
            id = "interface {}"
            if (t.substring(j, j + 2) == "{}") { j = j + 2 }
        }
        else if (id.length > 4 && id.substring(0, 4) == "chan" && GF_PREDECL.indexOf(" " + id.substring(4) + " ") >= 0) {
            id = "chan " + id.substring(4)
        }
        else if (!dotted && GF_PREDECL.indexOf(" " + id + " ") < 0) { id = "main." + id }
        out = out + id
        i = j
    }
    return out
}

// reflect.Type.String of v's dynamic type, which is fmt's %T. A 64 bit integer
// reads as int or uint (the value model keeps int64 and uint64 in the same box),
// and a composite whose element types are not recorded takes the one type all its
// elements share, else interface {}. A channel loses its direction. "" when the
// value model cannot tell (a function, a pointer cell).
function gfTypeName(v) {
    var k = gfHKind(v)
    if (k == "nil") { return "<nil>" }
    if (k == "bool" || k == "string") { return k }
    if (k == "int" || k == "float" || k == "complex") { return gfHTy(v) }
    if (k == "struct") {
        var nm = gfHStructName(v)
        if (nm != "") { return nm }
        return gfAnonTy(v)
    }
    if (k == "list") {
        var es = gfHElems(v)
        var et = gfHElemTy(v)
        var ets = et != "" ? gfQualTy(et) : gfCommonTy(es)
        if (gfHIsArray(v)) { return "[" + gfIntText(es.length) + "]" + ets }
        return "[]" + ets
    }
    if (k == "map") {
        var mt = gfHMapTy(v)
        if (mt != "") { return gfQualTy(mt) }
        var kv = gfHMapKV(v)
        return "map[" + gfCommonTy(kv[0]) + "]" + gfCommonTy(kv[1])
    }
    var ct = gfHChanTy(v)
    if (ct != "") { return "chan " + gfQualTy(ct) }
    return ""
}

function gfCommonTy(es) {
    if (es.length == 0) { return "interface {}" }
    var t = gfTypeName(es[0])
    if (t == "" || t == "<nil>") { return "interface {}" }
    var i = 1
    while (i < es.length) {
        if (gfTypeName(es[i]) != t) { return "interface {}" }
        i = i + 1
    }
    return t
}

// struct { A int; B string }: a field whose declared type is not recorded takes
// its value's.
function gfAnonTy(v) {
    var fs = gfHFields(v)
    var tys = gfHFTypes(v)
    if (fs[0].length == 0) { return "struct {}" }
    var out = "struct { "
    var i = 0
    while (i < fs[0].length) {
        if (i > 0) { out = out + "; " }
        var ty = i < tys.length ? tys[i] : ""
        out = out + fs[0][i] + " " + (ty != "" ? gfQualTy(ty) : gfTypeName(fs[1][i]))
        i = i + 1
    }
    return out + " }"
}

// ----- %#v: Go syntax -------------------------------------------------------------

// printValue under sharpV: v as a Go literal of its type. slot is the type the
// enclosing composite declares for this position (qualified, "" when unknown),
// which is what a nil there is spelled with and what makes a plain number held in
// a byte slot print as a uint8. No method is called: only a GoString() would be,
// and it is not modelled.
function gfGoSyn(f, v, slot, depth) {
    var k = gfHKind(v)
    if (k == "nil") {
        if (depth == 0) { return gfPad(f, "<nil>") }
        return gfNilOf(slot)
    }
    if (k == "bool") { return gfPad(f, v === true ? "true" : "false") }
    if (k == "int") {
        if (gfHTy(v).charAt(0) == "u" || (slot.charAt(0) == "u" && GF_PREDECL.indexOf(" " + slot + " ") >= 0)) {
            var g = gfCopy(f)
            g.sharp = true
            return gfInteger(g, v, "x")
        }
        return gfInteger(f, v, "d")
    }
    if (k == "float") { return gfFloat(f, gfHNum(v), gfHF32(v), "g", 0 - 1) }
    if (k == "complex") { return gfComplex(gfState(), v, "v") }
    if (k == "string") { return gfPad(f, "\"" + gfHQuote(v, "\"") + "\"") }
    var ty = gfTypeName(v)
    if (k == "struct") {
        var fs = gfHFields(v)
        var tys = gfHFTypes(v)
        var out = ty + "{"
        var i = 0
        while (i < fs[0].length) {
            if (i > 0) { out = out + ", " }
            var ft = i < tys.length ? tys[i] : ""
            out = out + fs[0][i] + ":" + gfGoSyn(f, fs[1][i], ft != "" ? gfQualTy(ft) : "", depth + 1)
            i = i + 1
        }
        return out + "}"
    }
    if (k == "list") {
        var arr = gfHIsArray(v)
        if (!arr && gfHRef(v)[0] === null) { return (ty == "[]uint8" && depth == 0 ? "[]byte" : ty) + "(nil)" }
        var et = ty.substring(ty.indexOf("]") + 1)
        var outl = (ty == "[]uint8" && depth == 0 ? "[]byte" : ty) + "{"
        var es = gfHElems(v)
        var j = 0
        while (j < es.length) {
            if (j > 0) { outl = outl + ", " }
            outl = outl + gfGoSyn(f, es[j], et, depth + 1)
            j = j + 1
        }
        return outl + "}"
    }
    if (k == "map") {
        if (gfHRef(v)[0] === null) { return ty + "(nil)" }
        var kt = gfMapKeyTy(ty)
        var vt = ty.substring(kt.length + 5)
        var kv = gfHMapKV(v)
        var outm = ty + "{"
        var m = 0
        while (m < kv[0].length) {
            if (m > 0) { outm = outm + ", " }
            outm = outm + gfGoSyn(f, kv[0][m], kt, depth + 1) + ":" + gfGoSyn(f, kv[1][m], vt, depth + 1)
            m = m + 1
        }
        return outm + "}"
    }
    if (gfHChanTy(v) != "") { return "(" + ty + ")(" + gfHStr(v) + ")" }
    return gfHStr(v)
}

// A nil in a slot of the given type: interface {}(nil), (*main.T)(nil),
// []int(nil), map[string]int(nil), error(nil).
function gfNilOf(slot) {
    if (slot == "" || slot == "interface {}") { return "interface {}(nil)" }
    var c = slot.charAt(0)
    if (c == "*" || slot.substring(0, 4) == "func" || slot.substring(0, 4) == "chan") { return "(" + slot + ")(nil)" }
    return slot + "(nil)"
}

// K of a "map[K]V" type text.
function gfMapKeyTy(t) {
    var d = 0
    var i = 3
    while (i < t.length) {
        var c = t.charCodeAt(i)
        if (c == 91) { d = d + 1 }
        else if (c == 93) {
            d = d - 1
            if (d == 0) { return t.substring(4, i) }
        }
        i = i + 1
    }
    return ""
}

// ----- strings ------------------------------------------------------------------

// strconv.IsPrint: a letter, mark, number, punctuation or symbol, or U+0020. The
// table is its runs over U+0000..U+10FFFF as go 1.27 (Unicode 17.0.0) answers
// them: alternately the gap before a run and the run's length, in base 36. It is
// decoded on first use into GF_PRT, run starts and (exclusive) ends.
var GF_PRT = []
function gfIsPrint(cp) {
    if (cp >= 32 && cp < 127) { return true }
    if (cp < 161) { return false }
    if (GF_PRT.length == 0) { gfPrintRuns() }
    var lo = 0
    var hi = GF_PRT.length / 2
    while (lo < hi) {
        var mid = Math.floor((lo + hi) / 2)
        if (cp < GF_PRT[2 * mid]) { hi = mid }
        else if (cp >= GF_PRT[2 * mid + 1]) { lo = mid + 1 }
        else { return true }
    }
    return false
}

// The escape strconv.Quote writes for a code point that is not printable and has
// no letter escape: \uHHHH, or \UHHHHHHHH above the BMP.
function gfUEsc(cp) {
    var hx = gfToBase(gfIntText(cp), 16, false)
    var w = cp < 65536 ? 4 : 8
    while (hx.length < w) { hx = "0" + hx }
    return (cp < 65536 ? "\\u" : "\\U") + hx
}

function gfPrintRuns() {
    var t = "w,2n,y,c,1,ju,2,6,4,7,1,1,1,k,1,b1,1,12,2,1e,2,3,1,1j,8,r,4,6,h,m,1,5c,1,1c," +
        "2,1n,2,2t,e,1n,2,1d,2,f,1,s,2,1,1,b,5,w,7,23,1,4h,1,8,2,2,2,m,1,7,1,1,3,4,2," +
        "9,2,2,2,4,8,1,4,2,1,5,2,p,2,3,1,6,4,2,2,m,1,7,1,2,1,2,1,2,2,1,1,5,4,2,2,3,3," +
        "1,7,4,1,1,7,h,a,3,1,9,1,3,1,m,1,7,1,2,1,5,2,a,1,3,1,3,2,1,f,4,2,c,7,7,1,3,1," +
        "8,2,2,2,m,1,7,1,2,1,5,2,9,2,2,2,3,7,3,4,2,1,5,2,i,a,2,1,6,3,3,1,4,3,2,1,1,1," +
        "2,3,2,3,3,3,c,4,5,3,3,1,4,2,1,6,1,e,l,5,d,1,3,1,n,1,g,2,9,1,3,1,4,7,2,1,3,1," +
        "2,2,4,2,a,7,m,1,3,1,n,1,a,1,5,2,9,1,3,1,4,7,2,5,3,1,4,2,a,1,3,c,d,1,3,1,1f," +
        "1,3,1,6,4,g,2,q,1,3,1,i,3,o,1,9,1,1,2,7,3,1,4,6,1,1,1,8,6,a,2,3,c,1m,4,t,11," +
        "2,1,1,1,5,1,o,1,1,1,n,2,5,1,1,1,7,1,a,2,4,w,20,1,10,4,13,1,10,1,f,1,d,11,5i," +
        "1,1,5,1,2,ah,1,4,2,7,1,1,1,4,2,15,1,4,2,x,1,4,2,7,1,1,1,4,2,f,1,1l,1,4,2,1v," +
        "2,w,3,q,6,2e,2,6,2,hs,1,s,3,2h,7,m,9,o,9,k,c,d,1,3,1,2,c,2m,2,a,6,a,6,e,1,b," +
        "6,2h,7,17,5,1y,a,v,1,c,4,c,4,1,3,16,2,5,b,18,4,q,6,b,3,1q,2,1t,1,t,2,b,6,a," +
        "6,e,2,1a,2,c,k,25,1,4m,8,1o,3,f,3,1q,5,17,2,b,8,17,5,eu,2,6,2,12,2,6,2,8,1," +
        "1,1,1,1,1,1,v,2,1h,1,f,1,e,2,6,1,j,2,3,1,9,h,o,8,1b,h,2,2,r,1,d,3,y,e,x,f," +
        "3w,4,ii,m,b,l,1ec,2,am,5,19,1,1,5,1,2,1k,7,2,e,o,9,7,1,7,1,7,1,7,1,7,1,7,1," +
        "7,1,7,1,3i,y,q,1,2h,c,5y,q,g,1,1r,1,2e,2,2v,5,17,1,2m,1,2e,9,1c,1,mlp,3,1j," +
        "9,9o,k,54,8,65,k,1o,3,a,6,1k,8,1y,8,c,6,38,b,u,3,26,1,b,4,x,1,1j,9,e,2,a,2," +
        "2v,o,s,a,6,2,6,2,6,9,7,1,7,1,1o,4,3i,2,a,6,8mc,c,n,4,1d,6is,a6,2,2y,12,7,c," +
        "5,5,q,1,5,1,1,1,2,1,2,1,i2,w,16,6,1f,1,j,1,4,4,5,1,3r,4,5a,3,6,2,6,2,6,2,3," +
        "3,7,1,7,d,2,2,c,1,q,1,j,1,2,1,f,2,e,y,3f,5,3,4,19,3,2g,1,d,3,1,1b,1a,3m,t,3," +
        "1d,f,s,4,10,9,u,5,17,5,u,1,11,4,e,16,4e,2,a,6,10,4,10,4,14,8,1g,b,c,1,f,1,7," +
        "1,2,1,b,1,f,1,7,1,2,3,1g,c,8n,9,m,a,8,o,6,1,16,1,9,1x,6,2,1,1,18,1,2,3,1,2," +
        "n,1,20,8,9,1c,j,1,2,5,x,3,r,5,r,12,1k,4,k,2,1e,1,2,5,8,1,3,1,t,2,3,4,a,7,9," +
        "7,1s,w,13,4,c,9,1i,3,t,2,r,5,q,7,4,c,7,28,21,1j,1f,d,1f,7,1a,8,a,6,12,3,t,8," +
        "2,5s,v,1,16,1,3,2,2,g,6,8,9,x,1a,8,16,m,q,12,s,k,n,9,26,4,10,9,1q,1,5,d,p,7," +
        "a,6,1h,1,i,8,13,9,2o,1,k,b,i,1,1b,1q,7,1,1,1,4,1,f,1,b,6,1n,5,a,6,4,1,8,2,2," +
        "2,m,1,7,1,2,1,5,1,a,2,2,2,3,2,1,6,1,5,7,2,7,3,5,b,a,1,1,2,1,1,12,1,a,1,1,2," +
        "1,1,4,1,a,1,2,8,2,t,2k,1,5,u,20,8,a,4m,1i,2,12,y,1x,b,a,6,d,j,1m,6,a,6,k,s," +
        "r,2,f,4,n,55,1o,2s,2b,c,8,2,1,2,8,1,2,1,u,1,2,2,c,9,a,1y,8,2,1a,2,b,r,20,8," +
        "2b,d,21,7,a,2e,8,2g,y,e,a,6,9,1,19,1,e,a,t,3,w,2,m,1,e,21,7,1,2,1,18,3,1,1," +
        "2,1,9,8,a,6,6,1,2,1,11,1,2,1,6,7,a,6,18,4,a,6u,p,7,h,1,15,3,t,2d,1,f,1e,d," +
        "pn,2u,33,1,5,b,5g,218,2r,d,ts,g,m,a,32z,5,g7,5a1,1m,1c6,ft,7,v,1,a,4,29,1,a," +
        "6,u,2,6,a,1y,a,a,1,7,1,l,5,j,c0,1m,5i,2j,5,p,2,p,18,23,4,1l,7,h,1s,5,b,7,9," +
        "5p2,15,w,2p,37,6pp,4,1,7,1,2,1,83,f,1,t,3,2,1,e,4,8,b0,1s4,2z,5,d,3,9,7,a,2," +
        "4,31c,71,3,c4,6,n,f,h,f,1a,2,n,9,38,1o,6u,a,13,2,22,8,34,l,1y,3e,k,c,k,c,2f," +
        "9,p,3r,2d,1,1z,1,2,2,1,2,2,2,4,1,c,1,1,1,7,1,1t,1,4,2,8,1,7,1,s,1,4,1,5,1,1," +
        "3,7,1,9g,2,84,2,ji,f,5,1,f,uo,v,6,6,5x,7,1,h,2,7,1,2,1,5,5,1q,x,1,34,19,3,e," +
        "2,a,4,2,8w,v,h,1m,5,1,cw,16,5y,17,4,1,5c,v,1,m,8,2,68,7,1,4,1,2,1,f,1,5h,2," +
        "g,15,24,4,a,4,2,lt,1w,24,1p,5e,4,1,r,1,2,1,1,2,1,1,a,1,4,1,1,1,1,6,1,4,1,1," +
        "1,1,1,1,3,1,2,1,1,2,1,1,1,1,1,1,1,1,1,1,2,1,1,2,4,1,7,1,4,1,4,1,1,1,a,1,h,5," +
        "3,1,5,1,h,1g,2,7i,18,4,2s,c,f,2,f,1,f,1,11,a,4u,1k,t,d,18,4,9,7,2,e,6,4a,rd," +
        "3,h,3,d,3,62,6,c,4,1,f,c,4,1k,8,a,6,14,8,u,2,c,4,2,e,9,13,9k,8,e,2,d,3,b,3," +
        "1l,1,1,4,g,2,c,4,a,7,43,1,2v,sl,wyo,w,3dq,2,4ge,2,5rl,f,ha,1wi,f2,15u,3t7,5," +
        "6ju,f62u,6o"
    var at = 0
    var n = 0
    var i = 0
    while (i <= t.length) {
        var c = i < t.length ? t.charCodeAt(i) : 44
        if (c == 44) {
            at = at + n
            GF_PRT.push(at)
            n = 0
        } else { n = n * 36 + (c <= 57 ? c - 48 : c - 87) }
        i = i + 1
    }
}

// f.truncateString: at most f.prec runes.
function gfTrunc(f, s) {
    if (!f.precPresent) { return s }
    var n = 0
    var i = 0
    while (i < s.length) {
        if (n == f.prec) { return s.substring(0, i) }
        var c = s.charCodeAt(i)
        if (c >= 55296 && c <= 56319 && i + 1 < s.length) { i = i + 1 }
        n = n + 1
        i = i + 1
    }
    return s
}

// fmtString: %v %s %x %X %q of a string.
function gfString(f, s, verb) {
    if (verb == "v" || verb == "s") { return gfPad(f, gfTrunc(f, s)) }
    if (verb == "x" || verb == "X") { return gfHex(f, gfHBytes(s), verb == "X") }
    if (verb == "q") { return gfQuote(f, s) }
    return gfBad(f, verb, s)
}

// strconv.CanBackquote.
function gfCanBackquote(s) {
    var i = 0
    while (i < s.length) {
        var c = s.charCodeAt(i)
        if (c == 96 || c == 127 || c == 65279) { return false }
        if (c < 32 && c != 9) { return false }
        i = i + 1
    }
    return true
}

// fmtQ.
function gfQuote(f, s0) {
    var s = gfTrunc(f, s0)
    if (f.sharp && gfCanBackquote(s)) { return gfPad(f, "`" + s + "`") }
    return gfPad(f, "\"" + gfQBody(f, s, "\"") + "\"")
}

// The quoted body, and with + (QuoteToASCII) every character outside ASCII
// escaped (gfUEsc).
function gfQBody(f, s, delim) {
    var q = gfHQuote(s, delim)
    if (!f.plus) { return q }
    var out = ""
    var i = 0
    while (i < q.length) {
        var c = q.charCodeAt(i)
        if (c < 128) { out = out + q.charAt(i); i = i + 1; continue }
        var cp = c
        if (c >= 55296 && c <= 56319 && i + 1 < q.length) {
            cp = (c - 55296) * 1024 + (q.charCodeAt(i + 1) - 56320) + 65536
            i = i + 1
        }
        out = out + gfUEsc(cp)
        i = i + 1
    }
    return out
}

var GF_LHEX = "0123456789abcdef"
var GF_UHEX = "0123456789ABCDEF"

// A byte value as two lower-case hex digits, for the \x and \u00 escapes.
function gfHex2(c) {
    var hi = Math.floor(c / 16)
    return GF_LHEX.charAt(hi) + GF_LHEX.charAt(c - hi * 16)
}

// fmtSbx: the bytes in hex, with % x spacing the bytes and %#x adding 0x.
function gfHex(f, bs, upper) {
    var digits = upper ? GF_UHEX : GF_LHEX
    var length = bs.length
    if (f.precPresent && f.prec < length) { length = f.prec }
    var width = 2 * length
    if (width > 0) {
        if (f.space) {
            if (f.sharp) { width = width * 2 }
            width = width + length - 1
        } else if (f.sharp) { width = width + 2 }
    } else {
        if (f.widPresent) { return gfRepeat(f.zero ? "0" : " ", f.wid) }
        return ""
    }
    var out = ""
    if (f.widPresent && f.wid > width && !f.minus) { out = gfRepeat(f.zero ? "0" : " ", f.wid - width) }
    var px = upper ? "0X" : "0x"
    if (f.sharp) { out = out + px }
    var i = 0
    while (i < length) {
        if (f.space && i > 0) {
            out = out + " "
            if (f.sharp) { out = out + px }
        }
        var c = bs[i]
        var hi = Math.floor(c / 16)
        out = out + digits.charAt(hi) + digits.charAt(c - hi * 16)
        i = i + 1
    }
    if (f.widPresent && f.wid > width && f.minus) { out = out + gfRepeat(" ", f.wid - width) }
    return out
}

// ----- integers -------------------------------------------------------------------

// A decimal digit string in base 2, 8 or 16, by long division.
function gfToBase(dec, base, upper) {
    if (dec == "0") { return "0" }
    var alpha = upper ? GF_UHEX : GF_LHEX
    var ds = []
    var i = 0
    while (i < dec.length) { ds.push(dec.charCodeAt(i) - 48); i = i + 1 }
    var out = ""
    while (ds.length > 0) {
        var rem = 0
        var q = []
        var j = 0
        while (j < ds.length) {
            var cur = rem * 10 + ds[j]
            var qd = Math.floor(cur / base)
            rem = cur - qd * base
            if (q.length > 0 || qd > 0) { q.push(qd) }
            j = j + 1
        }
        out = alpha.charAt(rem) + out
        ds = q
    }
    return out
}

// The code point %c %q %U take from an integer: rune(c), with anything above
// U+10FFFF (a negative included) and a surrogate reading as U+FFFD.
function gfRuneOf(v) {
    if (gfHIntNeg(v)) { return 65533 }
    var mag = gfHIntMag(v)
    if (mag.length > 7) { return 65533 }
    var cp = parseInt(mag, 10)
    if (cp > 1114111 || (cp >= 55296 && cp <= 57343)) { return 65533 }
    return cp
}

// fmtInteger and the verbs printArg sends to it.
function gfInteger(f, v, verb) {
    var base = 0
    if (verb == "v" || verb == "d") { base = 10 }
    else if (verb == "b") { base = 2 }
    else if (verb == "o" || verb == "O") { base = 8 }
    else if (verb == "x" || verb == "X") { base = 16 }
    else if (verb == "c") { return gfPad(f, gfHRune(gfRuneOf(v))) }
    else if (verb == "q") { return gfPad(f, "'" + gfQBody(f, gfHRune(gfRuneOf(v)), "'") + "'") }
    else if (verb == "U") { return gfUnicode(f, v) }
    else { return gfBad(f, verb, v) }
    var neg = gfHIntNeg(v)
    var mag = gfHIntMag(v)
    var digs = base == 10 ? mag : gfToBase(mag, base, verb == "X")
    var prec = 0
    if (f.precPresent) {
        prec = f.prec
        // Precision 0 and value 0 print nothing but the padding.
        if (prec == 0 && mag == "0") { return gfPadSp(f, "") }
    } else if (f.zero && f.widPresent) {
        prec = f.wid
        if (neg || f.plus || f.space) { prec = prec - 1 }
    }
    while (digs.length < prec) { digs = "0" + digs }
    if (f.sharp) {
        if (base == 2) { digs = "0b" + digs }
        else if (base == 8) { if (digs.charAt(0) != "0") { digs = "0" + digs } }
        else if (base == 16) { digs = (verb == "X" ? "0X" : "0x") + digs }
    }
    if (verb == "O") { digs = "0o" + digs }
    if (neg) { digs = "-" + digs }
    else if (f.plus) { digs = "+" + digs }
    else if (f.space) { digs = " " + digs }
    return gfPadSp(f, digs)
}

// fmtUnicode: U+0041, and with # the quoted character after it. The value is read
// as a uint64, so a negative one is its two's complement.
function gfUnicode(f, v) {
    var hex = gfToBase(gfHIntMag(v), 16, true)
    if (gfHIntNeg(v)) {
        while (hex.length < 16) { hex = "0" + hex }
        var tw = ""
        var carry = 1
        var k = 15
        while (k >= 0) {
            var d = 15 - GF_UHEX.indexOf(hex.charAt(k)) + carry
            carry = 0
            if (d == 16) { d = 0; carry = 1 }
            tw = GF_UHEX.charAt(d) + tw
            k = k - 1
        }
        hex = tw
    }
    var prec = 4
    if (f.precPresent && f.prec > 4) { prec = f.prec }
    while (hex.length < prec) { hex = "0" + hex }
    var out = "U+" + hex
    if (f.sharp && !gfHIntNeg(v)) {
        var mag = gfHIntMag(v)
        if (mag.length <= 7) {
            var cp = parseInt(mag, 10)
            if (cp <= 1114111 && gfIsPrint(cp)) { out = out + " '" + gfHRune(cp) + "'" }
        }
    }
    return gfPadSp(f, out)
}

// ----- floats ------------------------------------------------------------------------

// fmtFloat in print.go: which verbs a float takes and their default precision.
function gfFloatVerb(f, x, f32, verb, v) {
    if (verb == "v") { return gfFloat(f, x, f32, "g", 0 - 1) }
    if (verb == "b" || verb == "g" || verb == "G" || verb == "x" || verb == "X") { return gfFloat(f, x, f32, verb, 0 - 1) }
    if (verb == "f" || verb == "e" || verb == "E") { return gfFloat(f, x, f32, verb, 6) }
    if (verb == "F") { return gfFloat(f, x, f32, "f", 6) }
    return gfBad(f, verb, v)
}

// fmtComplex: (re+imi), both parts under the verb, the imaginary one always signed.
function gfComplex(f, v, verb) {
    if (verb != "v" && verb != "b" && verb != "g" && verb != "G" && verb != "x" && verb != "X"
        && verb != "f" && verb != "F" && verb != "e" && verb != "E") { return gfBad(f, verb, v) }
    var parts = gfHCx(v)
    var g = gfCopy(f)
    g.plus = true
    return "(" + gfFloatVerb(f, parts[0], false, verb, v) + gfFloatVerb(g, parts[1], false, verb, v) + "i)"
}

// fmtFloat in format.go: the sign, the padding, and the # flag's trailing zeros.
function gfFloat(f, x, f32, verb, prec0) {
    var prec = prec0
    if (f.precPresent) { prec = f.prec }
    var num = gfFormatFloat(x, verb, prec, f32)
    var sgn = "+"
    var c0 = num.charAt(0)
    if (c0 == "-" || c0 == "+") { sgn = c0; num = num.substring(1) }
    if (f.space && sgn == "+" && !f.plus) { sgn = " " }
    if (num == "Inf" || num == "NaN") {
        var t = sgn + num
        if (num == "NaN" && !f.space && !f.plus) { t = num }
        return gfPadSp(f, t)
    }
    if (f.sharp && verb != "b") {
        var digits = 0
        if (verb == "v" || verb == "g" || verb == "G" || verb == "x") {
            digits = prec
            if (digits == 0 - 1) { digits = 6 }
        }
        var tail = ""
        var body = num
        var hasDot = false
        var sawNonzero = false
        var i = 0
        while (i < body.length) {
            var ch = body.charAt(i)
            if (ch == ".") { hasDot = true }
            else if (ch == "p" || ch == "P" || ((ch == "e" || ch == "E") && verb != "x" && verb != "X")) {
                tail = body.substring(i)
                body = body.substring(0, i)
                break
            } else {
                if (ch != "0") { sawNonzero = true }
                if (sawNonzero) { digits = digits - 1 }
            }
            i = i + 1
        }
        if (!hasDot) {
            if (body == "0") { digits = digits - 1 }
            body = body + "."
        }
        while (digits > 0) { body = body + "0"; digits = digits - 1 }
        num = body + tail
    }
    if (f.plus || sgn != "+") {
        if (f.zero && f.widPresent && f.wid > num.length + 1) {
            return sgn + gfRepeat("0", f.wid - num.length - 1) + num
        }
        return gfPad(f, sgn + num)
    }
    return gfPad(f, num)
}

// strconv.FormatFloat(x, fmt, prec, 64 or 32), for fmt in b e E f g G x X.
function gfFormatFloat(x, fc, prec, f32) {
    if (x !== x) { return "NaN" }
    if (x - x !== 0) { return x > 0 ? "+Inf" : "-Inf" }
    var neg = x < 0 || (x == 0 && 1 / x < 0)
    var a = neg ? 0 - x : x
    var sign = neg ? "-" : ""
    if (fc == "b") { return sign + gfFmtB(a, f32) }
    if (fc == "x" || fc == "X") { return sign + gfFmtX(a, prec, fc == "X") }
    var shortest = prec < 0
    var d = {d: "", dp: 0}
    if (shortest) {
        if (a != 0) { d = gfSplit(gfHShort(a, f32)) }
        if (fc == "e" || fc == "E") { prec = d.d.length - 1 }
        else if (fc == "f") { prec = d.d.length - d.dp > 0 ? d.d.length - d.dp : 0 }
        else { prec = d.d.length }
    } else {
        var keep = 0
        if (fc == "e" || fc == "E") { keep = prec + 1 }
        else if (fc == "f") { keep = 0 - 1 }
        else {
            if (prec == 0) { prec = 1 }
            keep = prec
        }
        d = gfExactRound(a, fc == "f", fc == "f" ? prec : keep)
    }
    if (fc == "e" || fc == "E") { return sign + gfFmtE(d, prec, fc) }
    if (fc == "f") { return sign + gfFmtF(d, prec) }
    // %e is used if the exponent from the conversion is less than -4 or greater
    // than or equal to the precision; the shortest form decides at precision 6.
    var eprec = prec
    if (eprec > d.d.length && d.d.length >= d.dp) { eprec = d.d.length }
    if (shortest) { eprec = 6 }
    var exp = d.dp - 1
    if (exp < 0 - 4 || exp >= eprec) {
        var pe = prec
        if (pe > d.d.length) { pe = d.d.length }
        return sign + gfFmtE(d, pe - 1, fc == "G" ? "E" : "e")
    }
    var pf = prec
    if (pf > d.dp) { pf = d.d.length }
    return sign + gfFmtF(d, pf - d.dp > 0 ? pf - d.dp : 0)
}

// DIGITS "e" EXP -> {d: the digits without trailing zeros, dp: the decimal point
// position}, the value being 0.d * 10^dp.
function gfSplit(s) {
    var ei = s.indexOf("e")
    var ds = s.substring(0, ei)
    var end = ds.length
    while (end > 0 && ds.charAt(end - 1) == "0") { end = end - 1 }
    return {d: ds.substring(0, end), dp: parseInt(s.substring(ei + 1), 10) + 1}
}

// The exact decimal of a > 0 rounded half to even: to n digits after the point
// when fixed, else to n significant digits. Zero digits left means the value is 0.
function gfExactRound(a, fixed, n) {
    if (a == 0) { return {d: "", dp: 0} }
    var ex = gfHExact(a)
    var ip = ex["i"]
    var all = ip + ex["f"]
    var dp = ip.length
    var lead = 0
    while (lead < all.length && all.charAt(lead) == "0") { lead = lead + 1 }
    all = all.substring(lead)
    dp = dp - lead
    var keep = fixed ? dp + n : n
    if (keep < 0) { return {d: "", dp: 0} }
    var kept = ""
    if (keep == 0) {
        // Rounding away every digit: up only past the half, or on a half with
        // more behind it (the even neighbour of a bare half is zero).
        var first = all.charCodeAt(0) - 48
        var more = false
        var i = 1
        while (i < all.length) { if (all.charAt(i) != "0") { more = true } i = i + 1 }
        if (first > 5 || (first == 5 && more)) { return {d: "1", dp: dp + 1} }
        return {d: "", dp: 0}
    }
    kept = gfHRound(all, keep)
    if (kept.length > keep) { dp = dp + 1 }
    var end = kept.length
    while (end > 0 && kept.charAt(end - 1) == "0") { end = end - 1 }
    if (end == 0) { return {d: "", dp: 0} }
    return {d: kept.substring(0, end), dp: dp}
}

// fmtE: d.ddddde±dd.
function gfFmtE(d, prec, fc) {
    var nd = d.d.length
    var out = nd == 0 ? "0" : d.d.charAt(0)
    if (prec > 0) {
        out = out + "."
        var i = 1
        var m = nd < prec + 1 ? nd : prec + 1
        if (i < m) { out = out + d.d.substring(i, m); i = m }
        while (i <= prec) { out = out + "0"; i = i + 1 }
    }
    var exp = nd == 0 ? 0 : d.dp - 1
    var es = "+"
    if (exp < 0) { es = "-"; exp = 0 - exp }
    out = out + fc + es
    if (exp < 10) { return out + "0" + exp }
    return out + exp
}

// fmtF: ddd.ddd.
function gfFmtF(d, prec) {
    var nd = d.d.length
    var out = ""
    if (d.dp > 0) {
        var m = nd < d.dp ? nd : d.dp
        out = d.d.substring(0, m)
        while (m < d.dp) { out = out + "0"; m = m + 1 }
    } else { out = "0" }
    if (prec > 0) {
        out = out + "."
        var i = 1
        while (i <= prec) {
            var j = d.dp + i - 1
            out = out + (j >= 0 && j < nd ? d.d.charAt(j) : "0")
            i = i + 1
        }
    }
    return out
}

// a = m * 2^e with m the integer significand and e the exponent fmtB prints:
// m in [2^52, 2^53) for a normal float64 ([2^23, 2^24) for a float32), and the
// subnormal exponent -1074 (-149) below that. Every scaling step is exact.
function gfBits(a, f32) {
    var lo = f32 ? 8388608 : 4503599627370496
    if (a == 0) { return {m: 0, e: f32 ? 0 - 149 : 0 - 1074} }
    if (a < gfMinNormal(f32)) {
        var s = a
        var k = 0
        var lim = f32 ? 149 : 1074
        while (k < lim) { s = s * 2; k = k + 1 }
        return {m: s, e: 0 - lim}
    }
    var m = a
    var e = 0
    while (m >= lo * 2) { m = m / 2; e = e + 1 }
    while (m < lo) { m = m * 2; e = e - 1 }
    return {m: m, e: e}
}

function gfPow2(n) {
    var p = 1
    var k = 0
    while (k < n) { p = p * 2; k = k + 1 }
    return p
}

// The smallest normal float64 (float32): 2^-1022 (2^-126).
function gfMinNormal(f32) {
    var p = 1
    var k = 0
    var n = f32 ? 126 : 1022
    while (k < n) { p = p / 2; k = k + 1 }
    return p
}

// An integer below 2^53 in decimal.
function gfIntText(m) {
    if (m == 0) { return "0" }
    var out = ""
    var x = m
    while (x > 0) {
        var q = Math.floor(x / 10)
        out = String.fromCharCode(48 + (x - q * 10)) + out
        x = q
    }
    return out
}

// fmtB: the significand in decimal, then p and the binary exponent.
function gfFmtB(a, f32) {
    var b = gfBits(a, f32)
    return gfIntText(b.m) + "p" + (b.e >= 0 ? "+" : "-") + gfIntText(b.e >= 0 ? b.e : 0 - b.e)
}

// fmtX: 0x1.hhhp±dd, the significand normalised to a leading 1 (a subnormal
// too), rounded half to even when fewer than 15 hex digits are asked for.
function gfFmtX(a, prec, upper) {
    var hex = upper ? GF_UHEX : GF_LHEX
    var lead = 0
    var fr = 0
    var exp = 0
    if (a != 0) {
        lead = 1
        fr = a
        while (fr >= 2) { fr = fr / 2; exp = exp + 1 }
        while (fr < 1) { fr = fr * 2; exp = exp - 1 }
        fr = fr - 1
    }
    var digs = ""
    if (prec >= 0 && prec < 15) {
        var scaled = fr
        var k = 0
        while (k < prec) { scaled = scaled * 16; k = k + 1 }
        var q = Math.floor(scaled)
        var r = scaled - q
        if (r > 0.5 || (r == 0.5 && q - Math.floor(q / 2) * 2 == 1)) { q = q + 1 }
        if (q == gfPow2(4 * prec)) {
            q = 0
            lead = lead + 1
            if (lead == 2) { lead = 1; exp = exp + 1 }
        }
        var j = 0
        while (j < prec) {
            var p16 = gfPow2(4 * (prec - 1 - j))
            var dg = Math.floor(q / p16)
            digs = digs + hex.charAt(dg)
            q = q - dg * p16
            j = j + 1
        }
    } else {
        var g = fr
        var cnt = 0
        while ((prec < 0 && g != 0) || (prec >= 0 && cnt < prec)) {
            g = g * 16
            var dg2 = Math.floor(g)
            digs = digs + hex.charAt(dg2)
            g = g - dg2
            cnt = cnt + 1
        }
    }
    var out = "0" + (upper ? "X" : "x") + lead
    if (digs.length > 0) { out = out + "." + digs }
    var es = "+"
    if (exp < 0) { es = "-"; exp = 0 - exp }
    out = out + (upper ? "P" : "p") + es
    if (exp < 10) { return out + "0" + exp }
    return out + exp
}
