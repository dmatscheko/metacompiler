// js-builtins.js -- JAVASCRIPT, not MetaJS: the Set and Map classes, Array.from /
// Array.of, Boolean, isNaN, isFinite and structuredClone, written in the language itself and compiled
// INTO a program by the js grammars, both halves (jsbLoad in js-interpreter.abnf and js-to-llvm-ir.abnf).
// A program that names Set, Map, Array, Boolean, isNaN, isFinite or structuredClone gets this file in
// front of it; one that does not pays nothing. Being one JavaScript source compiled by the same grammars is what
// makes the interpreter, llvm.Run and a native binary agree on every answer here.
//
// It is a Script ('.js'), runs before the program's own top level, and a program that
// declares a Set, a Map, an Array.from, a Boolean or an isNaN of its own replaces these.
//
// ITERATION. for-of, a spread, array destructuring and yield* drive any value with a
// callable `next` member lazily, and an array or a string by index. A Set or a Map is
// neither, so the grammars' iteration sites first call a value's [Symbol.iterator]
// method when it has one (jsIterOf / emitIterOf). This file spells that key as the
// string Symbol.iterator stands for, "__#@iterator", so that a program which never
// names a symbol itself has none, and the member externs skip their symbol test (see
// jvSymN in lib/js-rt.metajs). The fields are '__'-prefixed so they stay out of
// Object.keys and JSON.
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

class Set extends __Coll {
    constructor(items) {
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
}

class Map extends __Coll {
    constructor(entries) {
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
        return out
    }
    // Any other iterable through its own iterator, which for an arguments object stops
    // at its length.
    if ((t === "object" || t === "function") && (typeof items.next === "function" || typeof items["__#@iterator"] === "function")) {
        var it = typeof items["__#@iterator"] === "function" ? items["__#@iterator"]() : items
        for (var s = it.next(); !s.done; s = it.next()) {
            out.push(mapFn === undefined ? s.value : mapFn.call(self, s.value, i))
            i = i + 1
        }
        return out
    }
    var n = Number(items.length)
    if (!(n > 0)) { n = 0 }
    n = Math.floor(n)
    for (var j = 0; j < n; j++) {
        out.push(mapFn === undefined ? items[j] : mapFn.call(self, items[j], j))
    }
    return out
}

Array.of = function of(...items) { return items }

// Boolean(v): ToBoolean, so `[0, 1, ""].filter(Boolean)` keeps the truthy ones. There
// are no wrapper objects here: `new Boolean(v)` builds a plain object.
function Boolean(v) { return !!v }

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
