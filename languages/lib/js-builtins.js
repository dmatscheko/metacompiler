// js-builtins.js -- JAVASCRIPT, not MetaJS: the Set and Map classes, Array.from /
// Array.of, isNaN, isFinite, structuredClone, Reflect, WeakMap / WeakSet,
// queueMicrotask and the URI functions, written in the language itself and compiled
// INTO a program by the js grammars, both halves (jsbLoad in js-interpreter.abnf and js-to-llvm-ir.abnf).
// A program that names Set, Map, Array, isNaN, isFinite, structuredClone or Reflect gets this file in
// front of it; one that does not pays nothing. Being one JavaScript source compiled by the same grammars is what
// makes the interpreter, llvm.Run and a native binary agree on every answer here.
//
// It is a Script ('.js'), runs before the program's own top level, and a program that
// declares a Set, a Map, an Array.from or an isNaN of its own replaces these.
//
// ITERATION. for-of, a spread, array destructuring and yield* drive any value with a
// callable `next` member lazily, and an array or a string by index. A Set or a Map is
// neither, so the grammars' iteration sites first call a value's [Symbol.iterator]
// method when it has one (jsIterOf / emitIterOf). This file spells that key as the
// string Symbol.iterator stands for, "__#@iterator", so that a program which never
// names a symbol itself has none, and the member externs skip their symbol test (see
// jvSymN in lib/js-rt.metajs). The grammars take every key of this file raw (jsKeyRaw,
// keyEscT), where a program's string of that shape is escaped and stays a string. The
// fields are '__'-prefixed so they stay out of Object.keys and JSON.
//
// Every message and value below is node 24's.

// A deleted slot. Iterators skip it, so a deletion during a for-of is not visited and
// an addition is.
var __collHole = {}

// SameValueZero as a string per primitive: two keys are the same entry exactly when
// their strings are equal. -0 reads "n0" like 0, and every NaN "nNaN". An object, a
// function or a symbol-less anything else answers null and is found by identity.
function __collKey(v) {
    var t = typeof v
    if (t === "number") { return v !== v ? "nNaN" : "n" + v }
    if (t === "string") { return "s" + v }
    if (t === "boolean") { return v ? "bt" : "bf" }
    if (t === "undefined") { return "u" }
    if (v === null) { return "l" }
    if (t === "bigint") { return "g" + v }
    return null
}

// node's "<what> is not iterable" for a constructor or Array.from argument.
function __iterCheck(v) {
    if (v === null || v === undefined) {
        throw new TypeError((v === null ? "object null" : "undefined") + " is not iterable (cannot read property Symbol(Symbol.iterator))")
    }
    var t = typeof v
    if (t === "string" || Array.isArray(v)) { return v }
    if (t === "object" || t === "function") {
        if (typeof v.next === "function" || typeof v["__#@iterator"] === "function") { return v }
        throw new TypeError(t + " is not iterable (cannot read property Symbol(Symbol.iterator))")
    }
    throw new TypeError(t + " " + v + " is not iterable (cannot read property Symbol(Symbol.iterator))")
}

// node's TypeError for a callback that is not callable: "number 5 is not a function",
// 'string "x" ...', "object null ...", "object ...".
function __notFn(f) {
    var t = typeof f
    var what = t
    if (f === null) { what = "object null" }
    else if (t === "string") { what = "string \"" + f + "\"" }
    else if (t === "number" || t === "boolean") { what = t + " " + f }
    return new TypeError(what + " is not a function")
}

// The storage both collections share: keys and values in insertion order, holes where
// an entry was deleted, an index from __collKey to the slot for the primitive keys and
// the list of slots that hold an object key.
class __Coll {
    constructor() {
        this.__k = []
        this.__v = []
        this.__idx = {}
        this.__objs = []
        this.__n = 0
    }
    __find(k) {
        var p = __collKey(k)
        if (p !== null) {
            var i = this.__idx[p]
            return i === undefined ? -1 : i
        }
        for (var j = 0; j < this.__objs.length; j++) {
            if (this.__k[this.__objs[j]] === k) { return this.__objs[j] }
        }
        return -1
    }
    __put(k, v) {
        var p = __collKey(k)
        var i = this.__k.length
        this.__k.push(k)
        this.__v.push(v)
        if (p !== null) { this.__idx[p] = i } else { this.__objs.push(i) }
        this.__n = this.__n + 1
    }
    get size() { return this.__n }
    has(k) { return this.__find(k) >= 0 }
    delete(k) {
        var i = this.__find(k)
        if (i < 0) { return false }
        var p = __collKey(k)
        if (p !== null) { delete this.__idx[p] }
        else {
            var keep = []
            for (var j = 0; j < this.__objs.length; j++) { if (this.__objs[j] !== i) { keep.push(this.__objs[j]) } }
            this.__objs = keep
        }
        this.__k[i] = __collHole
        this.__v[i] = undefined
        this.__n = this.__n - 1
        return true
    }
    clear() {
        for (var i = 0; i < this.__k.length; i++) { this.__k[i] = __collHole; this.__v[i] = undefined }
        this.__idx = {}
        this.__objs = []
        this.__n = 0
    }
    forEach(f, self) {
        if (typeof f !== "function") { throw __notFn(f) }
        for (var i = 0; i < this.__k.length; i++) {
            if (this.__k[i] !== __collHole) { f.call(self, this.__v[i], this.__k[i], this) }
        }
    }
}

// An iterator over a collection: kind 0 the keys, 1 the values, 2 [key, value] pairs.
class __CollIter {
    constructor(c, kind) {
        this.__c = c
        this.__i = 0
        this.__kind = kind
    }
    next() {
        var c = this.__c
        while (c !== undefined && this.__i < c.__k.length) {
            var i = this.__i
            this.__i = i + 1
            if (c.__k[i] !== __collHole) {
                if (this.__kind === 0) { return {value: c.__k[i], done: false} }
                if (this.__kind === 1) { return {value: c.__v[i], done: false} }
                return {value: [c.__k[i], c.__v[i]], done: false}
            }
        }
        this.__c = undefined
        return {value: undefined, done: true}
    }
    ["__#@iterator"]() { return this }
}

// The iterable argument of Set and Map is optional, so it has a default: that is
// what makes the constructor's length, and so Set.length / Map.length, 0.
class Set extends __Coll {
    constructor(items = undefined) {
        super()
        if (items !== undefined && items !== null) {
            for (var v of __iterCheck(items)) { this.add(v) }
        }
    }
    add(v) {
        if (this.__find(v) < 0) {
            if (v === 0) { v = 0 }
            this.__put(v, v)
        }
        return this
    }
    values() { return new __CollIter(this, 1) }
    keys() { return new __CollIter(this, 1) }
    entries() { return new __CollIter(this, 2) }
    ["__#@iterator"]() { return new __CollIter(this, 1) }
    toString() { return "[object Set]" }
    // The set methods (ECMA-262 24.2.4.*): the argument is SET-LIKE, a size, a has and
    // a keys method (__setRec), and the answer is a new Set. Which side is iterated
    // depends on the two sizes, which decides the order of an intersection.
    union(o) {
        var r = new Set(this)
        for (var v of __setRec(o, "union").keys()) { r.add(v) }
        return r
    }
    intersection(o) {
        var n = __setRec(o, "intersection").size
        var r = new Set()
        if (this.size <= n) {
            for (var v of this) { if (o.has(v)) { r.add(v) } }
        } else {
            for (var w of o.keys()) { if (this.has(w)) { r.add(w) } }
        }
        return r
    }
    difference(o) {
        var n = __setRec(o, "difference").size
        var r = new Set(this)
        if (this.size <= n) {
            for (var v of this) { if (o.has(v)) { r.delete(v) } }
        } else {
            for (var w of o.keys()) { r.delete(w) }
        }
        return r
    }
    symmetricDifference(o) {
        var r = new Set(this)
        for (var v of __setRec(o, "symmetricDifference").keys()) {
            if (this.has(v)) { r.delete(v) } else { r.add(v) }
        }
        return r
    }
    isSubsetOf(o) {
        if (this.size > __setRec(o, "isSubsetOf").size) { return false }
        for (var v of this) { if (!o.has(v)) { return false } }
        return true
    }
    isSupersetOf(o) {
        if (this.size < __setRec(o, "isSupersetOf").size) { return false }
        for (var v of o.keys()) { if (!this.has(v)) { return false } }
        return true
    }
    isDisjointFrom(o) {
        var n = __setRec(o, "isDisjointFrom").size
        if (this.size <= n) {
            for (var v of this) { if (o.has(v)) { return false } }
        } else {
            for (var w of o.keys()) { if (this.has(w)) { return false } }
        }
        return true
    }
    // [Symbol.species] (ECMA-262 24.2.3.2): the class it is read on, a subclass too.
    static get ["__#@species"]() { return this }
}

// GetSetRecord (ECMA-262 24.2.1.2): o, checked to be set-like, in V8's words; fn is
// the method's name.
function __setRec(o, fn) {
    if ((typeof o !== "object" || o === null) && typeof o !== "function") {
        throw new TypeError("Set.prototype." + fn + " argument must be an object")
    }
    var n = +o.size
    if (n !== n) { throw new TypeError("The .size property is NaN") }
    if (n < 0) { throw new RangeError("'" + n + "' is an invalid size") }
    if (typeof o.has !== "function") { throw new TypeError("string \"has\" is not a function") }
    if (typeof o.keys !== "function") { throw new TypeError("string \"keys\" is not a function") }
    return o
}

class Map extends __Coll {
    constructor(entries = undefined) {
        super()
        if (entries !== undefined && entries !== null) {
            for (var e of __iterCheck(entries)) {
                if (e === null || (typeof e !== "object" && typeof e !== "function")) {
                    throw new TypeError("Iterator value " + e + " is not an entry object")
                }
                this.set(e[0], e[1])
            }
        }
    }
    get(k) {
        var i = this.__find(k)
        return i < 0 ? undefined : this.__v[i]
    }
    set(k, v) {
        var i = this.__find(k)
        if (i >= 0) { this.__v[i] = v }
        else {
            if (k === 0) { k = 0 }
            this.__put(k, v)
        }
        return this
    }
    keys() { return new __CollIter(this, 0) }
    values() { return new __CollIter(this, 1) }
    entries() { return new __CollIter(this, 2) }
    ["__#@iterator"]() { return new __CollIter(this, 2) }
    toString() { return "[object Map]" }
    static get ["__#@species"]() { return this }
    // Map.groupBy(items, f): the items in a Map from f(item, index) to the items it
    // gave that key, the keys in first-seen order and compared as Map keys.
    static groupBy(items, f) {
        if (items === null || items === undefined) { throw new TypeError("Map.groupBy called on null or undefined") }
        if (typeof f !== "function") {
            var what = (typeof f === "object" && f !== null) ? (Array.isArray(f) ? "[object Array]" : "#<Object>") : String(f)
            throw new TypeError(what + " is not a function")
        }
        var m = new Map()
        var i = 0
        for (var v of __iterCheck(items)) {
            var k = f(v, i)
            if (m.has(k)) { m.get(k).push(v) } else { m.set(k, [v]) }
            i = i + 1
        }
        return m
    }
}

// Array.from(items, mapFn, thisArg): an iterable is iterated, anything else is read as
// an array-like through its length.
Array.from = function from(items, mapFn, self) {
    if (mapFn !== undefined && typeof mapFn !== "function") { throw __notFn(mapFn) }
    if (items === null || items === undefined) { __iterCheck(items) }
    var out = []
    var t = typeof items
    var i = 0
    if (t === "string" || Array.isArray(items)) {
        for (var v of items) {
            out.push(mapFn === undefined ? v : mapFn.call(self, v, i))
            i = i + 1
        }
        return __arrOf(this, out)
    }
    // Any other iterable through its own iterator, which for an arguments object stops
    // at its length.
    if ((t === "object" || t === "function") && (typeof items.next === "function" || typeof items["__#@iterator"] === "function")) {
        var it = typeof items["__#@iterator"] === "function" ? items["__#@iterator"]() : items
        for (var s = it.next(); !s.done; s = it.next()) {
            out.push(mapFn === undefined ? s.value : mapFn.call(self, s.value, i))
            i = i + 1
        }
        return __arrOf(this, out)
    }
    var n = Number(items.length)
    if (!(n > 0)) { n = 0 }
    n = Math.floor(n)
    for (var j = 0; j < n; j++) {
        out.push(mapFn === undefined ? items[j] : mapFn.call(self, items[j], j))
    }
    return __arrOf(this, out)
}

Array.of = function of(...items) { return __arrOf(this, items) }

// Array.from and Array.of called on a class extending Array (`V.from(xs)`, which
// reaches them as V's inherited statics) answer an instance of that class: a
// `new V()` holding the elements.
function __arrOf(C, out) {
    if (C === Array || typeof C !== "function") { return out }
    var r = new C()
    for (var i = 0; i < out.length; i++) { r[i] = out[i] }
    return r
}

// The global isNaN and isFinite convert their argument with ToNumber first (so
// isNaN("abc") is true and isFinite("12") is true), unlike Number.isNaN/isFinite.
function isNaN(v) { var n = +v; return n !== n }
function isFinite(v) { var n = +v; return n === n && n !== Infinity && n !== -Infinity }

// structuredClone(v): a deep copy of the arrays, plain objects, Maps and Sets in v,
// a value reached twice (a cycle too) copied once. A class instance copies as a plain
// object of its own enumerable keys. A function or a symbol is node's DataCloneError,
// which here is an Error of that name; nothing in this engine is a DOMException.
function structuredClone(v) { return __clone(v, [], []) }
function __clone(v, seen, made) {
    var t = typeof v
    if (t === "function" || t === "symbol") {
        var e = new Error(String(v) + " could not be cloned.")
        e.name = "DataCloneError"
        throw e
    }
    if (v === null || t !== "object") { return v }
    for (var i = 0; i < seen.length; i++) { if (seen[i] === v) { return made[i] } }
    seen.push(v)
    if (Array.isArray(v)) {
        var a = []
        made.push(a)
        for (var j = 0; j < v.length; j++) { a.push(__clone(v[j], seen, made)) }
        return a
    }
    if (v instanceof Map) {
        var m = new Map()
        made.push(m)
        v.forEach(function (x, k) { m.set(__clone(k, seen, made), __clone(x, seen, made)) })
        return m
    }
    if (v instanceof Set) {
        var s = new Set()
        made.push(s)
        v.forEach(function (x) { s.add(__clone(x, seen, made)) })
        return s
    }
    var o = {}
    made.push(o)
    var ks = Object.keys(v)
    for (var n = 0; n < ks.length; n++) { o[ks[n]] = __clone(v[ks[n]], seen, made) }
    return o
}

// Reflect (ECMA-262 28.1): each function performs the internal method it is named
// after on its target, written with the operator or Object function that performs it.
// Its functions are non-enumerable, as a builtin's are.
function __reflObj(o, fn) {
    if ((typeof o !== "object" || o === null) && typeof o !== "function") {
        throw new TypeError("Reflect." + fn + " called on non-object")
    }
    return o
}
// [[Set]] the way strict code performs it: a refused write throws.
function __reflSet(o, k, v) { "use strict"; o[k] = v }
var Reflect = (function () {
    var fns = {
        apply: function apply(f, self, args) {
            if (typeof f !== "function") {
                var t = typeof f
                throw new TypeError("Function.prototype.apply was called on " + String(f) + ", which is " +
                    ("aeiou".indexOf(t.charAt(0)) >= 0 ? "an " : "a ") + t + " and not a function")
            }
            if ((typeof args !== "object" || args === null) && typeof args !== "function") {
                throw new TypeError("CreateListFromArrayLike called on non-object")
            }
            return f.apply(self, args)
        },
        construct: function construct(C, args) {
            if (typeof C !== "function") { throw new TypeError(String(C) + " is not a constructor") }
            if ((typeof args !== "object" || args === null) && typeof args !== "function") {
                throw new TypeError("CreateListFromArrayLike called on non-object")
            }
            return new C(...args)
        },
        defineProperty: function defineProperty(o, k, d) {
            __reflObj(o, "defineProperty")
            if ((typeof d !== "object" || d === null) && typeof d !== "function") { Object.defineProperty(o, k, d) }
            try {
                Object.defineProperty(o, k, d)
                return true
            } catch (e) {
                if (e instanceof TypeError) { return false }
                throw e
            }
        },
        // A refused delete answers false in sloppy code and throws in strict code,
        // which is how the typescript grammars compile this file.
        deleteProperty: function deleteProperty(o, k) {
            __reflObj(o, "deleteProperty")
            try {
                return delete o[k]
            } catch (e) {
                if (e instanceof TypeError) { return false }
                throw e
            }
        },
        get: function get(o, k) {
            __reflObj(o, "get")
            if (arguments.length < 3) { return o[k] }
            // An accessor runs with the receiver as `this`.
            for (var p = o; p !== null; p = Object.getPrototypeOf(p)) {
                var d = Object.getOwnPropertyDescriptor(p, k)
                if (d !== undefined) {
                    if (!("get" in d) && !("set" in d)) { return d.value }
                    return d.get === undefined ? undefined : d.get.call(arguments[2])
                }
            }
            return undefined
        },
        getOwnPropertyDescriptor: function getOwnPropertyDescriptor(o, k) {
            return Object.getOwnPropertyDescriptor(__reflObj(o, "getOwnPropertyDescriptor"), k)
        },
        getPrototypeOf: function getPrototypeOf(o) { return Object.getPrototypeOf(__reflObj(o, "getPrototypeOf")) },
        has: function has(o, k) { return k in __reflObj(o, "has") },
        isExtensible: function isExtensible(o) { return Object.isExtensible(__reflObj(o, "isExtensible")) },
        ownKeys: function ownKeys(o) {
            __reflObj(o, "ownKeys")
            return Object.getOwnPropertyNames(o).concat(Object.getOwnPropertySymbols(o))
        },
        preventExtensions: function preventExtensions(o) {
            Object.preventExtensions(__reflObj(o, "preventExtensions"))
            return true
        },
        set: function set(o, k, v) {
            __reflObj(o, "set")
            try {
                __reflSet(o, k, v)
                return true
            } catch (e) {
                if (e instanceof TypeError) { return false }
                throw e
            }
        },
        setPrototypeOf: function setPrototypeOf(o, p) {
            __reflObj(o, "setPrototypeOf")
            if (p === undefined || (typeof p !== "object" && typeof p !== "function")) {
                throw new TypeError("Object prototype may only be an Object or null: " + String(p))
            }
            if (!Object.isExtensible(o)) { return Object.getPrototypeOf(o) === p }
            Object.setPrototypeOf(o, p)
            return true
        }
    }
    var r = {}
    var ks = Object.keys(fns)
    for (var i = 0; i < ks.length; i++) {
        Object.defineProperty(r, ks[i], { value: fns[ks[i]], writable: true, configurable: true })
    }
    return r
})()

// WeakMap and WeakSet: keyed by object identity, and neither iterable nor sized. This
// engine's collector cannot see a key die, so an entry lives as long as its table.
function __weakKey(k, what) {
    if ((typeof k !== "object" || k === null) && typeof k !== "function") {
        throw new TypeError("Invalid value used " + what)
    }
}
class __Weak {
    constructor() {
        this.__k = []
        this.__v = []
    }
    __at(k) {
        for (var i = 0; i < this.__k.length; i++) { if (this.__k[i] === k) { return i } }
        return -1
    }
    has(k) { return this.__at(k) >= 0 }
    delete(k) {
        var i = this.__at(k)
        if (i < 0) { return false }
        this.__k.splice(i, 1)
        this.__v.splice(i, 1)
        return true
    }
}
class WeakMap extends __Weak {
    constructor(entries = undefined) {
        super()
        if (entries !== undefined && entries !== null) {
            for (var e of __iterCheck(entries)) { this.set(e[0], e[1]) }
        }
    }
    get(k) {
        var i = this.__at(k)
        return i < 0 ? undefined : this.__v[i]
    }
    set(k, v) {
        __weakKey(k, "as weak map key")
        var i = this.__at(k)
        if (i >= 0) { this.__v[i] = v } else {
            this.__k.push(k)
            this.__v.push(v)
        }
        return this
    }
    toString() { return "[object WeakMap]" }
}
class WeakSet extends __Weak {
    constructor(items = undefined) {
        super()
        if (items !== undefined && items !== null) {
            for (var v of __iterCheck(items)) { this.add(v) }
        }
    }
    add(k) {
        __weakKey(k, "in weak set")
        if (this.__at(k) < 0) {
            this.__k.push(k)
            this.__v.push(true)
        }
        return this
    }
    toString() { return "[object WeakSet]" }
}

// queueMicrotask(f): f runs as a microtask, after the running code and in order with
// the promise reactions queued before it.
function queueMicrotask(f) {
    if (typeof f !== "function") {
        var t = typeof f
        var got = (f === undefined || f === null) ? String(f)
            : (t === "object" ? "an instance of Object" : "type " + t + " (" + (t === "string" ? "'" + f + "'" : String(f)) + ")")
        throw new TypeError("The \"callback\" argument must be of type function. Received " + got)
    }
    Promise.resolve().then(function () { f() })
}

// encodeURIComponent / encodeURI / decodeURIComponent / decodeURI (ECMA-262 19.2.6):
// UTF-8 percent-encoding. A lone surrogate, or an escape that is not well-formed
// UTF-8, is a URIError. encodeURI keeps, and decodeURI leaves encoded, the reserved
// characters and '#'.
class URIError extends Error {
    constructor(m) {
        super(m)
        Object.defineProperty(this, "name", {value: "URIError", writable: true, configurable: true})
    }
}
var __uriKeep = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_.!~*'()"
var __uriReserved = ";/?:@&=+$,#"
function __uriEncode(v, keep) {
    var s = String(v)
    var out = ""
    for (var i = 0; i < s.length; i++) {
        var ch = s.charAt(i)
        if (__uriKeep.indexOf(ch) >= 0 || keep.indexOf(ch) >= 0) {
            out = out + ch
            continue
        }
        var cp = s.charCodeAt(i)
        if (cp >= 0xDC00 && cp <= 0xDFFF) { throw new URIError("URI malformed") }
        if (cp >= 0xD800 && cp <= 0xDBFF) {
            var lo = i + 1 < s.length ? s.charCodeAt(i + 1) : 0
            if (lo < 0xDC00 || lo > 0xDFFF) { throw new URIError("URI malformed") }
            cp = (cp - 0xD800) * 0x400 + (lo - 0xDC00) + 0x10000
            i = i + 1
        }
        if (cp < 0x80) { out = out + __uriHex(cp) }
        else if (cp < 0x800) { out = out + __uriHex(0xC0 | (cp >> 6)) + __uriHex(0x80 | (cp & 63)) }
        else if (cp < 0x10000) { out = out + __uriHex(0xE0 | (cp >> 12)) + __uriHex(0x80 | ((cp >> 6) & 63)) + __uriHex(0x80 | (cp & 63)) }
        else {
            out = out + __uriHex(0xF0 | (cp >> 18)) + __uriHex(0x80 | ((cp >> 12) & 63)) +
                __uriHex(0x80 | ((cp >> 6) & 63)) + __uriHex(0x80 | (cp & 63))
        }
    }
    return out
}
function __uriHex(b) { return "%" + "0123456789ABCDEF".charAt(b >> 4) + "0123456789ABCDEF".charAt(b & 15) }
// The byte the escape at s[i] spells, or -1 when s[i..i+2] is not '%' and two hex digits.
function __uriByte(s, i) {
    if (i + 2 >= s.length || s.charAt(i) !== "%") { return -1 }
    var h = "0123456789abcdef".indexOf(s.charAt(i + 1).toLowerCase())
    var l = "0123456789abcdef".indexOf(s.charAt(i + 2).toLowerCase())
    if (h < 0 || l < 0) { return -1 }
    return h * 16 + l
}
function __uriDecode(v, reserved) {
    var s = String(v)
    var out = ""
    var i = 0
    while (i < s.length) {
        var ch = s.charAt(i)
        if (ch !== "%") {
            out = out + ch
            i = i + 1
            continue
        }
        var start = i
        var b = __uriByte(s, i)
        if (b < 0) { throw new URIError("URI malformed") }
        i = i + 3
        if (b < 0x80) {
            var c0 = String.fromCharCode(b)
            out = out + (reserved.indexOf(c0) >= 0 ? s.substring(start, i) : c0)
            continue
        }
        var n = 0
        var cp = 0
        var min = 0
        if ((b & 0xE0) === 0xC0) { n = 1; cp = b & 31; min = 0x80 }
        else if ((b & 0xF0) === 0xE0) { n = 2; cp = b & 15; min = 0x800 }
        else if ((b & 0xF8) === 0xF0) { n = 3; cp = b & 7; min = 0x10000 }
        else { throw new URIError("URI malformed") }
        for (var j = 0; j < n; j++) {
            var cb = __uriByte(s, i)
            if (cb < 0 || (cb & 0xC0) !== 0x80) { throw new URIError("URI malformed") }
            cp = cp * 64 + (cb & 63)
            i = i + 3
        }
        if (cp < min || cp > 0x10FFFF || (cp >= 0xD800 && cp <= 0xDFFF)) { throw new URIError("URI malformed") }
        out = out + String.fromCodePoint(cp)
    }
    return out
}
function encodeURIComponent(v) { return __uriEncode(v, "") }
function encodeURI(v) { return __uriEncode(v, __uriReserved) }
function decodeURIComponent(v) { return __uriDecode(v, "") }
function decodeURI(v) { return __uriDecode(v, __uriReserved) }
