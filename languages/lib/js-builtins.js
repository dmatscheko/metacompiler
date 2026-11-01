// js-builtins.js -- JAVASCRIPT, not MetaJS: the Set and Map classes, Array.from /
// Array.of, isNaN, isFinite, structuredClone, Reflect, WeakMap / WeakSet,
// queueMicrotask, the URI functions, ArrayBuffer, Uint8Array, Date and the timers, written in the language itself and compiled
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
    // A call without `new`. The class-call check of every engine runs a class's OWN
    // static __call instead of raising "Class constructor ... cannot be invoked"
    // (jsClassCall in js-interpreter.abnf).
    static __call() { throw new TypeError("Constructor Set requires 'new'") }
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
    get ["__#@toStringTag"]() { return "Set" }
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
    static __call() { throw new TypeError("Constructor Map requires 'new'") }
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
    get ["__#@toStringTag"]() { return "Map" }
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
// Removes a dense array's element i: shift the tail down one, drop the last slot.
// No splice: tools/metajs-lint holds every lib/*.js to the MetaJS subset.
function __arrDrop(a, i) {
    for (var j = i + 1; j < a.length; j++) { a[j - 1] = a[j] }
    a.pop()
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
        __arrDrop(this.__k, i)
        __arrDrop(this.__v, i)
        return true
    }
}
class WeakMap extends __Weak {
    static __call() { throw new TypeError("Constructor WeakMap requires 'new'") }
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
    get ["__#@toStringTag"]() { return "WeakMap" }
}
class WeakSet extends __Weak {
    static __call() { throw new TypeError("Constructor WeakSet requires 'new'") }
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
    get ["__#@toStringTag"]() { return "WeakSet" }
}

// node's TypeError for a callback argument that is not a function.
function __cbCheck(f) {
    if (typeof f !== "function") {
        var t = typeof f
        var got = (f === undefined || f === null) ? String(f)
            : (t === "object" ? "an instance of Object" : "type " + t + " (" + (t === "string" ? "'" + f + "'" : String(f)) + ")")
        throw new TypeError("The \"callback\" argument must be of type function. Received " + got)
    }
}

// queueMicrotask(f): f runs as a microtask, after the running code and in order with
// the promise reactions queued before it.
function queueMicrotask(f) {
    __cbCheck(f)
    Promise.resolve().then(function () { f() })
}

// encodeURIComponent / encodeURI / decodeURIComponent / decodeURI (ECMA-262 19.2.6):
// UTF-8 percent-encoding. A lone surrogate, or an escape that is not well-formed
// UTF-8, is a URIError. encodeURI keeps, and decodeURI leaves encoded, the reserved
// characters and '#'.
class URIError extends Error {
    static __call(m) { return new URIError(m) }
    constructor(m) {
        super(m)
        Object.defineProperty(this, "name", {value: "URIError", writable: true, configurable: true})
    }
}
// EvalError (ECMA-262 20.5.5.1): no engine operation raises one, so it is URIError's
// shape and loaded only when a program names it.
class EvalError extends Error {
    static __call(m) { return new EvalError(m) }
    constructor(m) {
        super(m)
        Object.defineProperty(this, "name", {value: "EvalError", writable: true, configurable: true})
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

// ===== ArrayBuffer and Uint8Array =====
// An ArrayBuffer holds its bytes in the plain array __b. A Uint8Array is a view of
// __n bytes at offset __o of such an array, which subarray and a view made over an
// ArrayBuffer share; its `buffer` is made on first use when the view owns its bytes.
// Each index is an enumerable accessor of the view's own; its write stores
// ToUint8 of the value, which `& 255` is (ToInt32's modulus is a multiple of 256).
class ArrayBuffer {
    constructor(n) {
        this.__b = []
        var len = __u8Len(n, "Invalid array buffer length", false)
        for (var i = 0; i < len; i++) { this.__b.push(0) }
    }
    get byteLength() { return this.__b.length }
    slice(s, e) {
        var n = this.__b.length
        var lo = __u8Rel(s, n, 0)
        var r = new ArrayBuffer(0)
        r.__b = this.__b.slice(lo, Math.max(__u8Rel(e, n, n), lo))
        return r
    }
    static isView(v) { return v instanceof Uint8Array }
    get ["__#@toStringTag"]() { return "ArrayBuffer" }
}
var __u8Share = null
function __u8Def(t, i) {
    Object.defineProperty(t, i, {
        get: function () { return t.__b[t.__o + i] },
        set: function (v) { t.__b[t.__o + i] = v & 255 },
        enumerable: true
    })
}
// ToIndex (ECMA-262 7.1.22), with V8's message: `what`, and the value when shown.
function __u8Len(v, what, shown) {
    if (v === undefined) { return 0 }
    var n = Math.trunc(+v)
    if (n !== n) { return 0 }
    if (n < 0 || n > 9007199254740991) { throw new RangeError(shown ? what + ": " + v : what) }
    return n
}
// A relative index of a slice-like method (ECMA-262 23.2.3): negative counts from
// the end, both clamped to [0, n].
function __u8Rel(v, n, dflt) {
    if (v === undefined) { return dflt }
    var k = Math.trunc(+v)
    if (k !== k) { return 0 }
    if (k < 0) { return Math.max(n + k, 0) }
    return Math.min(k, n)
}
// The values of a constructor or from() argument: a view's, an iterable's, or an
// array-like's by its length.
function __u8Vals(a) {
    if (a instanceof Uint8Array) { return a.__vals() }
    var out = []
    if (Array.isArray(a) || typeof a["__#@iterator"] === "function") {
        for (var v of a) { out.push(v) }
        return out
    }
    var n = __u8Len(a.length, "Invalid typed array length", true)
    for (var i = 0; i < n; i++) { out.push(a[i]) }
    return out
}
function __u8Check(f) { if (typeof f !== "function") { throw __notFn(f) } }
class Uint8Array {
    constructor(a, b, c) {
        this.__ab = undefined
        this.__tarr = true // The engines drop a write past the end (jsTypedDrop).
        if (__u8Share !== null) {
            var sh = __u8Share
            __u8Share = null
            this.__ab = sh.ab
            this.__b = sh.ab.__b
            this.__o = sh.o
            this.__n = sh.n
        } else if (a instanceof ArrayBuffer) {
            // new Uint8Array(buffer, byteOffset, length): a view of the buffer's bytes.
            var bl = a.__b.length
            var off = b === undefined ? 0 : Math.trunc(+b)
            if (off !== off) { off = 0 }
            if (off < 0 || off > bl) { throw new RangeError("Start offset " + off + " is outside the bounds of the buffer") }
            var len = c === undefined ? bl - off : __u8Len(c, "Invalid typed array length", true)
            if (off + len > bl) { throw new RangeError("Invalid typed array length: " + len) }
            this.__ab = a
            this.__b = a.__b
            this.__o = off
            this.__n = len
        } else {
            var src = (typeof a === "object" && a !== null) ? __u8Vals(a) : null
            this.__b = []
            this.__o = 0
            this.__n = src === null ? __u8Len(a, "Invalid typed array length", true) : src.length
            for (var i = 0; i < this.__n; i++) { this.__b.push(src === null ? 0 : src[i] & 255) }
        }
        for (var j = 0; j < this.__n; j++) { __u8Def(this, j) }
    }
    __vals() { return this.__b.slice(this.__o, this.__o + this.__n) }
    get length() { return this.__n }
    get byteLength() { return this.__n }
    get byteOffset() { return this.__o }
    get BYTES_PER_ELEMENT() { return 1 }
    static get BYTES_PER_ELEMENT() { return 1 }
    get buffer() {
        if (this.__ab === undefined) {
            this.__ab = new ArrayBuffer(0)
            this.__ab.__b = this.__b
        }
        return this.__ab
    }
    get ["__#@toStringTag"]() { return "Uint8Array" }
    static from(items, f, self) {
        if (f !== undefined) { __u8Check(f) }
        var vs = __u8Vals(items)
        if (f !== undefined) { for (var i = 0; i < vs.length; i++) { vs[i] = f.call(self, vs[i], i) } }
        return new Uint8Array(vs)
    }
    static of(...items) { return new Uint8Array(items) }
    at(i) {
        var k = Math.trunc(+i)
        if (k !== k) { k = 0 }
        if (k < 0) { k = k + this.__n }
        return (k < 0 || k >= this.__n) ? undefined : this.__b[this.__o + k]
    }
    fill(v, s, e) {
        var x = v & 255
        var hi = __u8Rel(e, this.__n, this.__n)
        for (var i = __u8Rel(s, this.__n, 0); i < hi; i++) { this.__b[this.__o + i] = x }
        return this
    }
    set(src, off) {
        var vs = __u8Vals(src)
        var o = off === undefined ? 0 : Math.trunc(+off)
        if (o !== o) { o = 0 }
        if (o < 0) { throw new RangeError("offset is out of bounds") }
        if (o + vs.length > this.__n) { throw new RangeError("offset is out of bounds") }
        for (var i = 0; i < vs.length; i++) { this.__b[this.__o + o + i] = vs[i] & 255 }
    }
    subarray(s, e) {
        var lo = __u8Rel(s, this.__n, 0)
        var hi = __u8Rel(e, this.__n, this.__n)
        __u8Share = {ab: this.buffer, o: this.__o + lo, n: Math.max(hi - lo, 0)}
        return new Uint8Array()
    }
    slice(s, e) {
        var lo = __u8Rel(s, this.__n, 0)
        var hi = __u8Rel(e, this.__n, this.__n)
        return new Uint8Array(this.__b.slice(this.__o + lo, this.__o + Math.max(hi, lo)))
    }
    indexOf(v, from) {
        for (var i = __u8Rel(from, this.__n, 0); i < this.__n; i++) { if (this.__b[this.__o + i] === v) { return i } }
        return -1
    }
    lastIndexOf(v, from) {
        var i = this.__n - 1
        if (from !== undefined) {
            var k = Math.trunc(+from)
            if (k !== k) { k = 0 }
            i = k < 0 ? this.__n + k : Math.min(k, this.__n - 1)
        }
        for (; i >= 0; i--) { if (this.__b[this.__o + i] === v) { return i } }
        return -1
    }
    includes(v, from) { return this.indexOf(v, from) >= 0 }
    join(sep) {
        var s = sep === undefined ? "," : String(sep)
        var out = ""
        for (var i = 0; i < this.__n; i++) { out = out + (i > 0 ? s : "") + this.__b[this.__o + i] }
        return out
    }
    toString() { return this.join(",") }
    forEach(f, self) {
        __u8Check(f)
        for (var i = 0; i < this.__n; i++) { f.call(self, this.__b[this.__o + i], i, this) }
    }
    map(f, self) {
        __u8Check(f)
        var out = []
        for (var i = 0; i < this.__n; i++) { out.push(f.call(self, this.__b[this.__o + i], i, this)) }
        return new Uint8Array(out)
    }
    filter(f, self) {
        __u8Check(f)
        var out = []
        for (var i = 0; i < this.__n; i++) {
            var v = this.__b[this.__o + i]
            if (f.call(self, v, i, this)) { out.push(v) }
        }
        return new Uint8Array(out)
    }
    reduce(f, ...init) {
        __u8Check(f)
        var i = 0
        var acc = init[0]
        if (init.length === 0) {
            if (this.__n === 0) { throw new TypeError("Reduce of empty array with no initial value") }
            acc = this.__b[this.__o]
            i = 1
        }
        for (; i < this.__n; i++) { acc = f(acc, this.__b[this.__o + i], i, this) }
        return acc
    }
    reduceRight(f, ...init) {
        __u8Check(f)
        var i = this.__n - 1
        var acc = init[0]
        if (init.length === 0) {
            if (this.__n === 0) { throw new TypeError("Reduce of empty array with no initial value") }
            acc = this.__b[this.__o + i]
            i = i - 1
        }
        for (; i >= 0; i--) { acc = f(acc, this.__b[this.__o + i], i, this) }
        return acc
    }
    every(f, self) {
        __u8Check(f)
        for (var i = 0; i < this.__n; i++) { if (!f.call(self, this.__b[this.__o + i], i, this)) { return false } }
        return true
    }
    some(f, self) {
        __u8Check(f)
        for (var i = 0; i < this.__n; i++) { if (f.call(self, this.__b[this.__o + i], i, this)) { return true } }
        return false
    }
    find(f, self) {
        var i = this.findIndex(f, self)
        return i < 0 ? undefined : this.__b[this.__o + i]
    }
    findIndex(f, self) {
        __u8Check(f)
        for (var i = 0; i < this.__n; i++) { if (f.call(self, this.__b[this.__o + i], i, this)) { return i } }
        return -1
    }
    reverse() {
        for (var i = 0, j = this.__n - 1; i < j; i++, j--) {
            var t = this.__b[this.__o + i]
            this.__b[this.__o + i] = this.__b[this.__o + j]
            this.__b[this.__o + j] = t
        }
        return this
    }
    // Numeric order by default, and a comparator's NaN reads as 0.
    sort(f) {
        if (f !== undefined) { __u8Check(f) }
        var vs = this.__vals()
        vs.sort(f === undefined ? function (x, y) { return x - y } : function (x, y) { var r = +f(x, y); return r !== r ? 0 : r })
        for (var i = 0; i < vs.length; i++) { this.__b[this.__o + i] = vs[i] }
        return this
    }
    *values() { for (var i = 0; i < this.__n; i++) { yield this.__b[this.__o + i] } }
    *keys() { for (var i = 0; i < this.__n; i++) { yield i } }
    *entries() { for (var i = 0; i < this.__n; i++) { yield [i, this.__b[this.__o + i]] } }
    ["__#@iterator"]() { return this.values() }
}

// ===== Date and the timers =====
// Compiled into a program only when it names Date or a timer function: the loaders
// cut the file here otherwise.

// Date (ECMA-262 21.4): a time value, milliseconds since the epoch in UTC, or NaN
// for an invalid date. Local time comes from the host's zone rules through
// __jsZone(kind, seconds) - kind 2 the offset east of UTC in seconds, 3 the zone's
// abbreviation - and the clock from __jsClockMs(); the engine provides both. Strings
// and toLocale* are node 24's in the en-US locale.
var __msDay = 86400000
// TimeClip's bound, 100,000,000 days either side of the epoch.
var __dMaxMs = 8640000000000000
var __dayNames = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"]
var __monNames = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"]
// The long name V8 shows for a zone, by the abbreviation the host reports; another
// zone shows its abbreviation.
var __zoneNames = {UTC: "Coordinated Universal Time", GMT: "Greenwich Mean Time", BST: "British Summer Time",
    CET: "Central European Standard Time", CEST: "Central European Summer Time",
    EET: "Eastern European Standard Time", EEST: "Eastern European Summer Time",
    WET: "Western European Standard Time", WEST: "Western European Summer Time",
    EST: "Eastern Standard Time", EDT: "Eastern Daylight Time", CST: "Central Standard Time",
    CDT: "Central Daylight Time", MST: "Mountain Standard Time", MDT: "Mountain Daylight Time",
    PST: "Pacific Standard Time", PDT: "Pacific Daylight Time", AKST: "Alaska Standard Time",
    AKDT: "Alaska Daylight Time", HST: "Hawaii-Aleutian Standard Time", JST: "Japan Standard Time",
    KST: "Korean Standard Time", IST: "India Standard Time", MSK: "Moscow Standard Time",
    AEST: "Australian Eastern Standard Time", AEDT: "Australian Eastern Daylight Time",
    NZST: "New Zealand Standard Time", NZDT: "New Zealand Daylight Time"}
// Reading Symbol.toPrimitive is what makes the engine consult a Date's
// [Symbol.toPrimitive] in `+`, `==` and the other ToPrimitive sites.
var __dateTP = Symbol.toPrimitive
// Elapsed time the timers skipped instead of waiting (__timerTick).
var __clockSkew = 0

function __dInt(v) {
    var n = Number(v)
    if (n !== n) { return 0 }
    return Math.trunc(n)
}
function __dMod(a, b) {
    var r = a % b
    return r < 0 ? r + b : r
}
// Days from 1970-01-01 to y-m-d of the proleptic Gregorian calendar, m in 1..12.
function __daysFromCivil(y, m, d) {
    if (m <= 2) { y = y - 1 }
    var era = Math.floor(y / 400)
    var yoe = y - era * 400
    var mp = (m + 9) % 12
    var doy = Math.floor((153 * mp + 2) / 5) + d - 1
    var doe = yoe * 365 + Math.floor(yoe / 4) - Math.floor(yoe / 100) + doy
    return era * 146097 + doe - 719468
}
// The calendar fields of a time value (UTC or already shifted to local time).
function __dFields(t) {
    var days = Math.floor(t / __msDay)
    var ms = t - days * __msDay
    var z = days + 719468
    var era = Math.floor(z / 146097)
    var doe = z - era * 146097
    var yoe = Math.floor((doe - Math.floor(doe / 1460) + Math.floor(doe / 36524) - Math.floor(doe / 146096)) / 365)
    var doy = doe - (365 * yoe + Math.floor(yoe / 4) - Math.floor(yoe / 100))
    var mp = Math.floor((5 * doy + 2) / 153)
    var d = doy - Math.floor((153 * mp + 2) / 5) + 1
    var m = mp < 10 ? mp + 3 : mp - 9
    var y = yoe + era * 400 + (m <= 2 ? 1 : 0)
    return {y: y, mo: m - 1, d: d, wd: __dMod(days + 4, 7), h: Math.floor(ms / 3600000),
        mi: Math.floor(ms / 60000) % 60, s: Math.floor(ms / 1000) % 60, ms: ms % 1000}
}
// MakeDay and MakeTime combined (21.4.1.28-29), every argument already a number.
function __dMake(y, mo, d, h, mi, s, ms) {
    var a = [y, mo, d, h, mi, s, ms]
    for (var i = 0; i < 7; i++) { if (a[i] !== a[i] || a[i] === Infinity || a[i] === -Infinity) { return NaN } }
    y = Math.trunc(y)
    mo = Math.trunc(mo)
    var ym = y + Math.floor(mo / 12)
    var day = __daysFromCivil(ym, __dMod(mo, 12) + 1, 1) + Math.trunc(d) - 1
    return day * __msDay + Math.trunc(h) * 3600000 + Math.trunc(mi) * 60000 + Math.trunc(s) * 1000 + Math.trunc(ms)
}
function __dClip(t) {
    if (t !== t || Math.abs(t) > __dMaxMs) { return NaN }
    return Math.trunc(t) + 0
}
// The local zone's offset in milliseconds at UTC time t.
function __dOff(t) { return __jsZone(2, Math.floor(t / 1000)) * 1000 }
function __dLocal(t) { return t + __dOff(t) }
// UTC(t) for a local time lt. Near a transition the offsets 12 hours either side
// differ: a local time both of them give (a repeated hour) is the earlier instant,
// one neither gives (a skipped hour) is read with the offset from before.
function __dUTC(lt) {
    if (lt !== lt) { return NaN }
    var oa = __dOff(lt - 43200000)
    var ob = __dOff(lt + 43200000)
    if (oa === ob) { return lt - __dOff(lt - oa) }
    var va = __dOff(lt - oa) === oa
    var vb = __dOff(lt - ob) === ob
    if (va && vb) { return lt - Math.max(oa, ob) }
    if (vb && !va) { return lt - ob }
    return lt - oa
}
// V8 asks whether summer time is in force at an EQUIVALENT time for a t before 1970
// or past 2^31 - 1 seconds: the same month, day and time of day in the year of
// 2008..2035 that is a leap year when t's is and starts on the same weekday
// (DateCache::EquivalentTime).
function __dNameTime(t) {
    if (t >= 0 && t <= 2147483647000) { return t }
    var f = __dFields(t)
    var leap = f.y % 4 === 0 && (f.y % 100 !== 0 || f.y % 400 === 0)
    var recent = (leap ? 1956 : 1967) + (__dMod(__daysFromCivil(f.y, 1, 1) + 4, 7) * 12) % 28
    var ey = 2008 + (recent + 3 * 28 - 2008) % 28
    var days = Math.floor(t / __msDay)
    return (__daysFromCivil(ey, f.mo + 1, f.d) - days) * __msDay + t
}
function __pad(n, w) {
    var s = String(Math.abs(n))
    while (s.length < w) { s = "0" + s }
    return n < 0 ? "-" + s : s
}
function __dYear(y) { return y < 0 ? "-" + __pad(-y, 4) : __pad(y, 4) }
// "GMT+0100 (Central European Standard Time)" for UTC time t. The offset is t's own;
// the name is the zone's standard or summer time as at __dNameTime(t).
function __dZone(t) {
    var off = __jsZone(2, Math.floor(t / 1000))
    var a = Math.abs(off)
    var zs = (off < 0 ? "-" : "+") + __pad(Math.floor(a / 3600), 2) + __pad(Math.floor(a / 60) % 60, 2)
    var ab = __jsZone(3, Math.floor(__dNameTime(t) / 1000))
    // Before standard time the host says "LMT"; V8 names the zone's standard time.
    if (ab === "LMT") { ab = __jsZone(3, __jsZone(2, 946684800) <= __jsZone(2, 962409600) ? 946684800 : 962409600) }
    var nm = __zoneNames[ab]
    return "GMT" + zs + " (" + (nm === undefined ? ab : nm) + ")"
}
// Date.parse is V8's DateParser: the date-time string format (21.4.1.32) first, and
// whatever that leaves unread through V8's legacy rules - numbers into a day (up to
// three, month 1 / day 1 / year 1 filling the missing ones, so "5" is May 1 2001) or
// a time (after a colon), month and zone words, AM/PM and a UTC offset after a time or
// "GMT". A date-only string in the format is UTC, anything else without an offset
// local time.
//
// The tokens, as V8's DateStringTokenizer cuts them: k is "num" (v its value from the
// first nine digits, n its digit count), "sym" (: - + . or ), v the character), "kw" (a
// word: t its keyword type, v its value, n its length), "ws" (one whitespace
// character), "unk" (anything else, a parenthesized comment included) and "end".
var __dKeys = ["jan", "feb", "mar", "apr", "may", "jun", "jul", "aug", "sep", "oct", "nov", "dec",
    "am", "pm", "ut", "utc", "z", "gmt", "cdt", "cst", "edt", "est", "mdt", "mst", "pdt", "pst", "t"]
var __dKeyType = ["mon", "mon", "mon", "mon", "mon", "mon", "mon", "mon", "mon", "mon", "mon", "mon",
    "ampm", "ampm", "tz", "tz", "tz", "tz", "tz", "tz", "tz", "tz", "tz", "tz", "tz", "tz", "tsep"]
var __dKeyVal = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 0, 12, 0, 0, 0, 0, -5, -6, -4, -5, -6, -7, -7, -8, 0]
function __dWs(c) {
    return c === 9 || c === 10 || c === 11 || c === 12 || c === 13 || c === 32 || c === 160 || c === 5760 ||
        (c >= 8192 && c <= 8202) || c === 8232 || c === 8233 || c === 8239 || c === 8287 || c === 12288 || c === 65279
}
function __dTokens(s) {
    var out = []
    var i = 0
    while (i < s.length) {
        var c = s.charCodeAt(i)
        if (c >= 48 && c <= 57) {
            var v = 0
            var n = 0
            while (i < s.length && s.charCodeAt(i) >= 48 && s.charCodeAt(i) <= 57) {
                if (n < 9) { v = v * 10 + s.charCodeAt(i) - 48 }
                n = n + 1
                i = i + 1
            }
            out.push({k: "num", v: v, n: n})
        } else if (c === 58 || c === 45 || c === 43 || c === 46 || c === 41) {
            out.push({k: "sym", v: s.charAt(i)})
            i = i + 1
        } else if (c >= 65 && !__dWs(c)) {
            // A word is matched on its first three characters, lowercased by setting bit
            // 0x20 as V8 does; only a month name may be longer than its keyword.
            var pre = ""
            var len = 0
            while (i < s.length && s.charCodeAt(i) >= 65 && !__dWs(s.charCodeAt(i))) {
                if (len < 3) { pre = pre + String.fromCharCode(s.charCodeAt(i) | 32) }
                len = len + 1
                i = i + 1
            }
            var kt = "bad"
            var kv = 0
            for (var j = 0; j < __dKeys.length; j++) {
                if (__dKeys[j] === pre && (len <= 3 || __dKeyType[j] === "mon")) {
                    kt = __dKeyType[j]
                    kv = __dKeyVal[j]
                    break
                }
            }
            out.push({k: "kw", t: kt, v: kv, n: len})
        } else if (__dWs(c)) {
            out.push({k: "ws"})
            i = i + 1
        } else if (c === 40) {
            var bal = 0
            do {
                if (s.charCodeAt(i) === 41) { bal = bal - 1 } else if (s.charCodeAt(i) === 40) { bal = bal + 1 }
                i = i + 1
            } while (bal > 0 && i < s.length)
            out.push({k: "unk"})
        } else {
            out.push({k: "unk"})
            i = i + 1
        }
    }
    out.push({k: "end"})
    return out
}
// The reading position over the tokens, with V8's scanner operations.
function __dScan(toks) {
    var at = 0
    var sc = {}
    sc.peek = function () { return toks[at] }
    sc.next = function () {
        var t = toks[at]
        if (at < toks.length - 1) { at = at + 1 }
        return t
    }
    sc.skip = function (ch) {
        if (toks[at].k === "sym" && toks[at].v === ch) {
            at = at + 1
            return true
        }
        return false
    }
    sc.fixed = function (n) { return toks[at].k === "num" && toks[at].n === n }
    return sc
}
function __dIn(x, lo, hi) { return x >= lo && x <= hi }
// A fraction's digits as milliseconds: its first three, as written.
function __dMillis(tok) {
    var v = tok.v
    var n = tok.n
    if (n === 1) { return v * 100 }
    if (n === 2) { return v * 10 }
    if (n > 9) { n = 9 }
    while (n > 3) {
        v = Math.floor(v / 10)
        n = n - 1
    }
    return v
}
// The date-time string format, filling day, time and zone (the composers below):
// "end" when the whole string was read, "bad" when it cannot be a date at all, and
// otherwise the first token it did not take, which the legacy rules go on from.
function __dParseEs5(sc, day, time, tz) {
    var t = sc.peek()
    if (t.k === "sym" && (t.v === "+" || t.v === "-")) {
        var sign = sc.next()
        if (!sc.fixed(6)) { return sign }
        var y = sc.next().v
        if (sign.v === "-" && y === 0) { return sign }
        day.c.push(sign.v === "-" ? -y : y)
    } else if (sc.fixed(4)) {
        day.c.push(sc.next().v)
    } else {
        return sc.next()
    }
    if (sc.skip("-")) {
        if (!sc.fixed(2) || !__dIn(sc.peek().v, 1, 12)) { return sc.next() }
        day.c.push(sc.next().v)
        if (sc.skip("-")) {
            if (!sc.fixed(2) || !__dIn(sc.peek().v, 1, 31)) { return sc.next() }
            day.c.push(sc.next().v)
        }
    }
    var p = sc.peek()
    if (p.k !== "kw" || p.t !== "tsep") {
        if (p.k !== "end") { return sc.next() }
    } else {
        sc.next()
        if (!sc.fixed(2) || !__dIn(sc.peek().v, 0, 24)) { return "bad" }
        var h24 = sc.peek().v === 24
        time.c.push(sc.next().v)
        if (!sc.skip(":")) { return "bad" }
        if (!sc.fixed(2) || !__dIn(sc.peek().v, 0, 59) || (h24 && sc.peek().v > 0)) { return "bad" }
        time.c.push(sc.next().v)
        if (sc.skip(":")) {
            if (!sc.fixed(2) || !__dIn(sc.peek().v, 0, 59) || (h24 && sc.peek().v > 0)) { return "bad" }
            time.c.push(sc.next().v)
            if (sc.skip(".")) {
                if (sc.peek().k !== "num" || (h24 && sc.peek().v > 0)) { return "bad" }
                time.c.push(__dMillis(sc.next()))
            }
        }
        var z = sc.peek()
        if (z.k === "kw" && z.t === "tz" && z.n === 1) {
            sc.next()
            __dTzSet(tz, 0)
        } else if (z.k === "sym" && (z.v === "+" || z.v === "-")) {
            tz.sign = sc.next().v === "+" ? 1 : -1
            if (sc.fixed(4)) {
                var hm = sc.next().v
                if (!__dIn(Math.floor(hm / 100), 0, 23) || !__dIn(hm % 100, 0, 59)) { return "bad" }
                tz.h = Math.floor(hm / 100)
                tz.m = hm % 100
            } else {
                if (!sc.fixed(2) || !__dIn(sc.peek().v, 0, 23)) { return "bad" }
                tz.h = sc.next().v
                if (!sc.skip(":")) { return "bad" }
                if (!sc.fixed(2) || !__dIn(sc.peek().v, 0, 59)) { return "bad" }
                tz.m = sc.next().v
            }
        }
        if (sc.peek().k !== "end") { return "bad" }
    }
    if (tz.sign === null && time.c.length === 0) { __dTzSet(tz, 0) }
    day.iso = true
    return "end"
}
// A zone of a whole number of hours, the way a zone word or "Z" sets it.
function __dTzSet(tz, hours) {
    tz.sign = hours < 0 ? -1 : 1
    tz.h = Math.abs(hours)
    tz.m = 0
}
// [year, month 0-11, day] from the day composer, or null.
function __dDayOf(day) {
    var c = day.c
    if (c.length < 1) { return null }
    while (c.length < 3) { c.push(1) }
    var y = 0
    var mo = 0
    var d = 0
    if (day.mon === 0) {
        if (day.iso || !__dIn(c[0], 1, 31)) {
            y = c[0]
            mo = c[1]
            d = c[2]
        } else {
            mo = c[0]
            d = c[1]
            y = c[2]
        }
    } else {
        mo = day.mon
        if (!__dIn(c[0], 1, 31)) {
            y = c[0]
            d = c[1]
        } else {
            d = c[0]
            y = c[1]
        }
    }
    if (!day.iso) {
        if (__dIn(y, 0, 49)) { y = y + 2000 } else if (__dIn(y, 50, 99)) { y = y + 1900 }
    }
    if (!__dIn(mo, 1, 12) || !__dIn(d, 1, 31)) { return null }
    return [y, mo - 1, d]
}
// [hour, minute, second, millisecond] from the time composer, or null.
function __dTimeOf(time) {
    var c = time.c
    while (c.length < 4) { c.push(0) }
    if (time.ampm !== null) {
        if (!__dIn(c[0], 0, 12)) { return null }
        c[0] = c[0] % 12 + time.ampm
    }
    if (!__dIn(c[0], 0, 23) || !__dIn(c[1], 0, 59) || !__dIn(c[2], 0, 59) || !__dIn(c[3], 0, 999)) {
        if (c[0] !== 24 || c[1] !== 0 || c[2] !== 0 || c[3] !== 0) { return null }
    }
    return c
}
// Whether the time composer takes n as its next field after an hour.
function __dTimeWants(time, n) {
    var i = time.c.length
    return (i === 1 && __dIn(n, 0, 59)) || (i === 2 && __dIn(n, 0, 59)) || (i === 3 && __dIn(n, 0, 999))
}
function __dTimeFinal(time, n) {
    if (time.c.length >= 4) { return false }
    time.c.push(n)
    while (time.c.length < 4) { time.c.push(0) }
    return true
}
function __dParse(str) {
    var sc = __dScan(__dTokens(String(str)))
    var day = {c: [], mon: 0, iso: false}
    var time = {c: [], ampm: null}
    var tz = {sign: null, h: null, m: null}
    var tok = __dParseEs5(sc, day, time, tz)
    if (tok === "bad") { return NaN }
    var hasNum = day.c.length > 0
    if (tok !== "end") {
        for (; tok.k !== "end"; tok = sc.next()) {
            if (tok.k === "num") {
                hasNum = true
                var n = tok.v
                if (sc.skip(":")) {
                    if (sc.skip(":")) {
                        if (time.c.length > 0) { return NaN }
                        time.c.push(n)
                        time.c.push(0)
                    } else {
                        if (time.c.length >= 4) { return NaN }
                        time.c.push(n)
                        if (sc.peek().k === "sym" && sc.peek().v === ".") { sc.next() }
                    }
                } else if (sc.skip(".") && __dTimeWants(time, n)) {
                    time.c.push(n)
                    if (sc.peek().k !== "num") { return NaN }
                    __dTimeFinal(time, __dMillis(sc.next()))
                } else if (tz.h !== null && tz.m === null && __dIn(n, 0, 59)) {
                    tz.m = n
                } else if (__dTimeWants(time, n)) {
                    __dTimeFinal(time, n)
                    var pk = sc.peek()
                    if (pk.k !== "end" && pk.k !== "ws" && !(pk.k === "kw" && pk.t === "tz" && pk.n === 1) &&
                        !(pk.k === "sym" && (pk.v === "+" || pk.v === "-"))) { return NaN }
                } else {
                    if (day.c.length >= 3) { return NaN }
                    day.c.push(n)
                    sc.skip("-")
                }
            } else if (tok.k === "kw") {
                if (tok.t === "ampm" && time.c.length > 0) {
                    time.ampm = tok.v
                } else if (tok.t === "mon") {
                    day.mon = tok.v
                    sc.skip("-")
                } else if (tok.t === "tz" && hasNum) {
                    __dTzSet(tz, tok.v)
                } else {
                    // A word that means nothing is allowed only before the first number,
                    // and not touching it.
                    if (hasNum || sc.peek().k === "num") { return NaN }
                }
            } else if (tok.k === "sym" && (tok.v === "+" || tok.v === "-") && ((tz.h === 0 && tz.m === 0) || time.c.length > 0)) {
                tz.sign = tok.v === "+" ? 1 : -1
                var on = 0
                var ol = 0
                if (sc.peek().k === "num") {
                    var nt = sc.next()
                    ol = nt.n
                    on = nt.v
                }
                hasNum = true
                if (sc.peek().k === "sym" && sc.peek().v === ":") {
                    tz.h = on
                    tz.m = null
                } else if (ol === 1 || ol === 2) {
                    tz.h = on
                    tz.m = 0
                } else if (ol === 3 || ol === 4) {
                    tz.h = Math.floor(on / 100)
                    tz.m = on % 100
                } else { return NaN }
            } else if (tok.k === "sym" && (tok.v === "+" || tok.v === "-" || tok.v === ")") && hasNum) {
                return NaN
            }
        }
    }
    var dv = __dDayOf(day)
    var tv = __dTimeOf(time)
    if (dv === null || tv === null) { return NaN }
    var t = __dMake(dv[0], dv[1], dv[2], tv[0], tv[1], tv[2], tv[3])
    if (tz.sign === null) {
        if (Math.abs(t) > __dMaxMs + 864000000) { return NaN }
        return __dClip(__dUTC(t))
    }
    var secs = (tz.h === null ? 0 : tz.h) * 3600 + (tz.m === null ? 0 : tz.m) * 60
    // V8 holds the offset in seconds as a 32-bit small integer.
    if (secs > 2147483647) { return NaN }
    t = t - secs * tz.sign * 1000
    if (Math.abs(t) > __dMaxMs) { return NaN }
    return __dClip(t)
}
function __dArgs(args, from, n, dflt) {
    var out = []
    for (var i = 0; i < n; i++) { out.push(from + i < args.length ? Number(args[from + i]) : dflt[i]) }
    return out
}

class Date {
    // Date() without `new` is the current time as a string (ECMA-262 21.4.2.1 step 1).
    static __call() { return new Date().toString() }
    constructor(...a) {
        var t = NaN
        if (a.length === 0) { t = Date.now() }
        else if (a.length === 1) {
            var v = a[0]
            if (v instanceof Date) { t = v.__t }
            else {
                if (typeof v === "object" && v !== null) { v = v.valueOf() }
                if (typeof v === "object" && v !== null) { v = String(v) }
                t = typeof v === "string" ? __dParse(v) : __dClip(Number(v))
            }
        } else {
            var f = __dArgs(a, 0, 7, [NaN, 0, 1, 0, 0, 0, 0])
            var y = f[0]
            if (y === y) {
                var yi = Math.trunc(y)
                if (yi >= 0 && yi <= 99) { y = 1900 + yi }
            }
            t = __dClip(__dUTC(__dMake(y, f[1], f[2], f[3], f[4], f[5], f[6])))
        }
        this.__t = t
        this.__strtag = "Date" // Object.prototype.toString's tag, for the [[DateValue]] slot.
    }
    static now() { return Math.floor(__jsClockMs() + __clockSkew) }
    static parse(s) { return __dParse(s) }
    static UTC(...a) {
        var f = __dArgs(a, 0, 7, [NaN, 0, 1, 0, 0, 0, 0])
        var y = f[0]
        if (y === y) {
            var yi = Math.trunc(y)
            if (yi >= 0 && yi <= 99) { y = 1900 + yi }
        }
        return __dClip(__dMake(y, f[1], f[2], f[3], f[4], f[5], f[6]))
    }
    getTime() { return this.__t }
    valueOf() { return this.__t }
    __f(utc) { return __dFields(utc ? this.__t : __dLocal(this.__t)) }
    __get(utc, k) {
        if (this.__t !== this.__t) { return NaN }
        return this.__f(utc)[k]
    }
    getFullYear() { return this.__get(false, "y") }
    getMonth() { return this.__get(false, "mo") }
    getDate() { return this.__get(false, "d") }
    getDay() { return this.__get(false, "wd") }
    getHours() { return this.__get(false, "h") }
    getMinutes() { return this.__get(false, "mi") }
    getSeconds() { return this.__get(false, "s") }
    getMilliseconds() { return this.__get(false, "ms") }
    getUTCFullYear() { return this.__get(true, "y") }
    getUTCMonth() { return this.__get(true, "mo") }
    getUTCDate() { return this.__get(true, "d") }
    getUTCDay() { return this.__get(true, "wd") }
    getUTCHours() { return this.__get(true, "h") }
    getUTCMinutes() { return this.__get(true, "mi") }
    getUTCSeconds() { return this.__get(true, "s") }
    getUTCMilliseconds() { return this.__get(true, "ms") }
    getTimezoneOffset() {
        if (this.__t !== this.__t) { return NaN }
        return -Math.trunc(__dOff(this.__t) / 60000)
    }
    setTime(t) {
        this.__t = __dClip(Number(t))
        return this.__t
    }
    // The setters: fields from `at` on (0 year .. 6 ms) replaced by args, the rest
    // kept, in local time or UTC. setFullYear starts an invalid date from +0.
    __set(utc, at, args) {
        var n = args.length
        if (n === 0) { n = 1 }
        var vals = []
        for (var i = 0; i < n; i++) { vals.push(Number(args[i])) }
        var t = this.__t
        var lt = utc ? t : __dLocal(t)
        if (t !== t) {
            if (at !== 0) { return NaN }
            lt = 0
        }
        var f = __dFields(lt)
        var cur = [f.y, f.mo, f.d, f.h, f.mi, f.s, f.ms]
        for (var j = 0; j < n && at + j < 7; j++) { cur[at + j] = vals[j] }
        var nt = __dMake(cur[0], cur[1], cur[2], cur[3], cur[4], cur[5], cur[6])
        this.__t = __dClip(utc ? nt : __dUTC(nt))
        return this.__t
    }
    setFullYear(...a) { return this.__set(false, 0, a.slice(0, 3)) }
    setMonth(...a) { return this.__set(false, 1, a.slice(0, 2)) }
    setDate(...a) { return this.__set(false, 2, a.slice(0, 1)) }
    setHours(...a) { return this.__set(false, 3, a.slice(0, 4)) }
    setMinutes(...a) { return this.__set(false, 4, a.slice(0, 3)) }
    setSeconds(...a) { return this.__set(false, 5, a.slice(0, 2)) }
    setMilliseconds(...a) { return this.__set(false, 6, a.slice(0, 1)) }
    setUTCFullYear(...a) { return this.__set(true, 0, a.slice(0, 3)) }
    setUTCMonth(...a) { return this.__set(true, 1, a.slice(0, 2)) }
    setUTCDate(...a) { return this.__set(true, 2, a.slice(0, 1)) }
    setUTCHours(...a) { return this.__set(true, 3, a.slice(0, 4)) }
    setUTCMinutes(...a) { return this.__set(true, 4, a.slice(0, 3)) }
    setUTCSeconds(...a) { return this.__set(true, 5, a.slice(0, 2)) }
    setUTCMilliseconds(...a) { return this.__set(true, 6, a.slice(0, 1)) }
    toISOString() {
        if (this.__t !== this.__t) { throw new RangeError("Invalid time value") }
        var f = this.__f(true)
        var ys = (f.y >= 0 && f.y <= 9999) ? __pad(f.y, 4) : (f.y < 0 ? "-" : "+") + __pad(Math.abs(f.y), 6)
        return ys + "-" + __pad(f.mo + 1, 2) + "-" + __pad(f.d, 2) + "T" + __pad(f.h, 2) + ":" + __pad(f.mi, 2) + ":" +
            __pad(f.s, 2) + "." + __pad(f.ms, 3) + "Z"
    }
    toJSON(k) { return this.__t !== this.__t ? null : this.toISOString() }
    toDateString() {
        if (this.__t !== this.__t) { return "Invalid Date" }
        var f = this.__f(false)
        return __dayNames[f.wd] + " " + __monNames[f.mo] + " " + __pad(f.d, 2) + " " + __dYear(f.y)
    }
    toTimeString() {
        if (this.__t !== this.__t) { return "Invalid Date" }
        var f = this.__f(false)
        return __pad(f.h, 2) + ":" + __pad(f.mi, 2) + ":" + __pad(f.s, 2) + " " + __dZone(this.__t)
    }
    toString() {
        if (this.__t !== this.__t) { return "Invalid Date" }
        return this.toDateString() + " " + this.toTimeString()
    }
    toUTCString() {
        if (this.__t !== this.__t) { return "Invalid Date" }
        var f = this.__f(true)
        return __dayNames[f.wd] + ", " + __pad(f.d, 2) + " " + __monNames[f.mo] + " " + __dYear(f.y) + " " +
            __pad(f.h, 2) + ":" + __pad(f.mi, 2) + ":" + __pad(f.s, 2) + " GMT"
    }
    toGMTString() { return this.toUTCString() }
    toLocaleDateString() {
        if (this.__t !== this.__t) { return "Invalid Date" }
        var f = this.__f(false)
        // A year before 1 shows as the era's: 2 BC for year -1.
        return (f.mo + 1) + "/" + f.d + "/" + (f.y <= 0 ? 1 - f.y : f.y)
    }
    toLocaleTimeString() {
        if (this.__t !== this.__t) { return "Invalid Date" }
        var f = this.__f(false)
        var h12 = f.h % 12 === 0 ? 12 : f.h % 12
        return h12 + ":" + __pad(f.mi, 2) + ":" + __pad(f.s, 2) + (f.h < 12 ? " AM" : " PM")
    }
    toLocaleString() {
        if (this.__t !== this.__t) { return "Invalid Date" }
        return this.toLocaleDateString() + ", " + this.toLocaleTimeString()
    }
    ["__#@toPrimitive"](hint) {
        if (hint !== "string" && hint !== "number" && hint !== "default") { throw new TypeError("Invalid hint: " + hint) }
        if (hint === "number") { return this.__t }
        return this.toString()
    }
}

// setTimeout / setInterval / clearTimeout / clearInterval. A timer is due `delay`
// milliseconds (at least 1) after the LOOP TIME: the clock when the first timer is set,
// then the due time of the timer running - so timers are ordered by their delays
// alone, however long the code between them takes. The engine calls __timerTick once the microtask
// queue is empty: it runs the earliest due timer (the earlier set first among
// equals) and answers whether one ran. Nothing waits: the clock is moved on to the
// timer's due time instead (__clockSkew), so the program sees the time pass.
var __timers = []
var __timerSeq = 0
var __loopNow = -1
function __timerAdd(f, delay, args, repeat) {
    __cbCheck(f)
    var ms = Number(delay)
    if (!(ms >= 1 && ms <= 2147483647)) { ms = 1 }
    ms = Math.trunc(ms)
    if (__loopNow < 0) { __loopNow = Date.now() }
    __timerSeq = __timerSeq + 1
    var tm = new __Timeout(__timerSeq, f, ms, args, repeat)
    __timers.push(tm)
    return tm
}
class __Timeout {
    constructor(id, f, ms, args, repeat) {
        this.__id = id
        this.__f = f
        this.__ms = ms
        this.__args = args
        this.__repeat = repeat
        this.__due = __loopNow + ms
        this.__seq = id
    }
    ref() { return this }
    unref() { return this }
    hasRef() { return true }
    refresh() {
        this.__due = __loopNow + this.__ms
        return this
    }
    ["__#@toPrimitive"](hint) { return this.__id }
}
function setTimeout(f, delay, ...args) { return __timerAdd(f, delay, args, false) }
function setInterval(f, delay, ...args) { return __timerAdd(f, delay, args, true) }
function __timerClear(h) {
    for (var i = 0; i < __timers.length; i++) {
        var tm = __timers[i]
        if (tm === h || (typeof h === "number" && tm.__id === h) || (typeof h === "string" && String(tm.__id) === h)) {
            __arrDrop(__timers, i)
            return
        }
    }
}
function clearTimeout(h) { __timerClear(h) }
function clearInterval(h) { __timerClear(h) }
function __timerTick() {
    if (__timers.length === 0) { return false }
    var at = 0
    for (var i = 1; i < __timers.length; i++) {
        var a = __timers[i]
        var b = __timers[at]
        if (a.__due < b.__due || (a.__due === b.__due && a.__seq < b.__seq)) { at = i }
    }
    var tm = __timers[at]
    var now = Date.now()
    if (tm.__due > now) { __clockSkew = __clockSkew + (tm.__due - now) }
    __loopNow = tm.__due
    if (tm.__repeat) {
        __timerSeq = __timerSeq + 1
        tm.__seq = __timerSeq
        tm.__due = tm.__due + tm.__ms
    } else {
        __arrDrop(__timers, at)
    }
    tm.__f(...tm.__args)
    return true
}
