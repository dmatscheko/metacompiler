// interp-core.js -- the shared tag-script core for the tree-walking interpreter grammars.
//
// A language grammar pulls this file in at the top of its startScript with
//     include("lib/interp-core.js")
// and gets the common interpreter machinery: scopes, the BRK/CONT loop protocol,
// constant/variable/operator thunk builders, string unescaping, and the function-call
// frame. Everything lands in the same global scope as the grammar's own script, so
// tags can call these builders directly and the grammar can override any of them by
// assignment (name = function(...) {...}) after the include.
//
// Where languages genuinely differ, the behavior is a knob on the `core` object below;
// the including grammar sets its knobs right before c.compile(c.asg), when all of its
// own functions are defined. The library late-binds one name the language file must
// provide: mcall(target, name, args) for method calls.

// ----- Configuration knobs -----
var core = {
    lang: "?",          // Language name used in error messages.
    blank: null,        // An assignment target of this name is silently discarded (Go: "_").
    nullWord: "null",   // How the language spells its null value in error messages (Go: "nil").
    add: function(l, r) {   // The + operator and += (default: string concat or 32-bit int add).
        if (typeof l == "string" || typeof r == "string") return l + r
        return (l + r) | 0
    },
    test: function(v) { return v },                  // Condition truthiness in if/while (Python: pytruthy).
    getField: function(o, name) { return o[name] },  // .name reads (Java: array .length).
    getIndex: function(o, i) { return o[i] },        // [i] reads (Go: map-aware).
    framePush: null,    // Called on function entry (Go: open a defer frame).
    framePop: null,     // Called after the body, before locals are dropped (Go: run defers).
    varMiss: null,      // Name not in any scope: return {v: value} or null (Kotlin: this properties).
    setMiss: null       // Assignment target not in any scope: handle it and return true (Kotlin).
}

// ----- Shared interpreter state -----
// Call frames are scope chains ending in globalScope; hostGlobals holds the built-ins
// the language file populates (they resolve after all scopes, see getVar).
var globalScope = {}
var scopes = [globalScope]
var hostGlobals = {}
// Loop control: a break/continue statement returns one of these markers through the
// statement thunks; the enclosing loop consumes it (any other value is a return).
var BRK = {sig: "break"}
var CONT = {sig: "continue"}

function fail(msg) {
    println(core.lang + " interpreter error: " + msg)
    exit(1)
}

function declVar(name, value) { if (name != core.blank) scopes[scopes.length - 1][name] = value }

function setVar(name, value) {
    if (name == core.blank) { return }
    for (var i = scopes.length - 1; i >= 0; i--) {
        if (hasOwn(scopes[i], name)) { scopes[i][name] = value; return }
    }
    if (core.setMiss != null && core.setMiss(name, value)) { return }
    fail("assignment to unknown variable: " + name)
}

function takeAll() {
    var items = []
    var v = anytype // The tags push values of every type.
    while ((v = pop()) != null) items.unshift(v)
    return items
}

function popName() {
    var items = takeAll()
    var name = ""
    for (var i = 0; i < items.length; i++) {
        if (items[i] != undefined && items[i].mname != undefined) { name = items[i].mname }
        else { push(items[i]) }
    }
    return name
}

function hexAt(s, pos, len) {
    var v = 0
    for (var i = 0; i < len; i++) {
        var c = s.charCodeAt(pos + i)
        var d = (c >= 48 && c <= 57) ? c - 48 : (c >= 97 ? c - 87 : c - 55)
        v = v * 16 + d
    }
    return v
}

function unescapeJs(s) {
    var out = ""
    var i = 0
    while (i < s.length) {
        var c = s.charCodeAt(i)
        if (c != 92) { out += String.fromCharCode(c); i++; continue }
        var e = s.charCodeAt(i + 1)
        if (e == 110) { out += "\n"; i += 2; continue }
        if (e == 116) { out += "\t"; i += 2; continue }
        if (e == 114) { out += "\r"; i += 2; continue }
        if (e == 48)  { out += String.fromCharCode(0); i += 2; continue }
        if (e == 120) { out += String.fromCharCode(hexAt(s, i + 2, 2)); i += 4; continue }
        if (e == 117) { out += String.fromCharCode(hexAt(s, i + 2, 4)); i += 6; continue }
        out += String.fromCharCode(e)
        i += 2
    }
    return out
}

function hasOwn(o, name) { return Object.prototype.hasOwnProperty.call(o, name) }

function getVar(name) {
    for (var i = scopes.length - 1; i >= 0; i--) {
        if (hasOwn(scopes[i], name)) return scopes[i][name]
    }
    if (core.varMiss != null) { var h = core.varMiss(name); if (h != null) return h.v }
    if (hasOwn(hostGlobals, name)) return hostGlobals[name]
    fail("unknown name: " + name)
}

function makeConst(v) { return function() { return v } }

function makeVarRef(name) { return function() { return getVar(name) } }

function makeNeg(t) { return function() { return (0 - t()) | 0 } }

function makeNot(t) { return function() { return !t() } }

function foldBinary(items) {
    if (items.length == 1) return items[0]
    return function() {
        var v = anytype // A comparison fold turns the running number into a boolean.
        v = items[0]()
        for (var i = 1; i < items.length; i += 2) v = binOp(items[i], v, items[i + 1]())
        return v
    }
}

function makeOrAnd(items, isOr) {
    if (items.length == 1) return items[0]
    return function() {
        var v = anytype // The operands may be of mixed types.
        v = items[0]()
        for (var i = 1; i < items.length; i++) {
            if (isOr) { if (v) return true }
            else      { if (!v) return false }
            v = items[i]()
        }
        return v ? true : false
    }
}

function makeTargetRef(items) {
    return {name: items[0], path: items.slice(1)}
}

function exprToStmt(t) { return function() { t() } }

function makeSeq(items) {
    return function() {
        for (var i = 0; i < items.length; i++) {
            var r = items[i]()
            if (r != undefined) return r
        }
    }
}

function makeBlockStmt(items) {
    var seq = makeSeq(items)
    return function() {
        scopes.push({})
        var r = seq()
        scopes.pop()
        return r
    }
}

function makeIf(items) {
    var cond = items[0]
    var thenT = items[1]
    var elseT = items.length > 2 ? items[2] : null
    return function() {
        if (core.test(cond())) return thenT()
        if (elseT != null) return elseT()
    }
}

function makeWhile(bodyT, condT) {
    return function() {
        while (core.test(condT())) {
            var r = bodyT()
            if (r != undefined) {
                if (r === BRK) break
                if (r !== CONT) return r
            }
        }
    }
}

function makeBreak() { return function() { return BRK } }

function makeContinue() { return function() { return CONT } }

function concat2impl(a, b) {
    var out = []
    var i
    for (i = 0; i < a.length; i++) out.push(a[i])
    for (i = 0; i < b.length; i++) out.push(b[i])
    return out
}

function binOp(op, l, r) {
    if (op == "+") return core.add(l, r)
    if (op == "-") return (l - r) | 0
    if (op == "*") return (l * r) | 0
    if (op == "/") return (l / r) | 0
    if (op == "%") return (l % r) | 0
    if (op == "==") return l === r
    if (op == "!=") return l !== r
    if (op == "<")  return l < r
    if (op == ">")  return l > r
    if (op == "<=") return l <= r
    if (op == ">=") return l >= r
}

function applyOp(op, a, b) {
    if (op == "+=") return core.add(a, b)
    if (op == "-=") return (a - b) | 0
    if (op == "*=") return (a * b) | 0
    if (op == "/=") return (a / b) | 0
    if (op == "%=") return (a % b) | 0
}

// Folds a primary expression with .method(args) / .field / [index] suffixes.
// Method dispatch goes through the language file's mcall(target, name, args).
function foldCallMember(items) {
    if (items.length == 1) return items[0]
    var primary = items[0]
    var suffixes = items.slice(1)
    return function() {
        var cur = anytype // A member chain hops between types: obj.name.length is object, string, number.
        cur = primary()
        for (var i = 0; i < suffixes.length; i++) {
            var s = suffixes[i]
            if (s.kind == "mcall") {
                var argv = []
                for (var j = 0; j < s.args.length; j++) argv.push(s.args[j]())
                cur = mcall(cur, s.name, argv)
            } else if (s.kind == "field") {
                if (cur === undefined || cur === null) fail("field ." + s.name + " of " + core.nullWord)
                cur = core.getField(cur, s.name)
            } else {
                cur = core.getIndex(cur, s.idx())
            }
        }
        return cur
    }
}

// Walks an assignment target's path up to the last step and returns the
// container/key pair the assignment writes to.
function resolveRef(ref) {
    var o = anytype // The containers along the path may be of mixed types.
    o = getVar(ref.name)
    for (var i = 0; i < ref.path.length - 1; i++) {
        var s = ref.path[i]
        if (s.key != undefined) { o = o[s.key] }
        else { o = core.getIndex(o, s.idx()) }
        if (o === undefined || o === null) fail(core.nullWord + " in assignment path of " + ref.name)
    }
    var last = ref.path[ref.path.length - 1]
    return {obj: o, key: (last.key != undefined) ? last.key : last.idx()}
}

// Runs a function body in a fresh frame: the receiver (if any) is bound to recvName,
// missing arguments become undefined, and the frame hooks wrap the body.
function invokeBody(params, body, self, recvName, args) {
    var saved = scopes
    scopes = [globalScope, {}]
    if (self != undefined) declVar(recvName, self)
    for (var i = 0; i < params.length; i++) {
        declVar(params[i], i < args.length ? args[i] : undefined)
    }
    if (core.framePush != null) core.framePush()
    var r = body()
    if (core.framePop != null) core.framePop()
    scopes = saved
    return (r != undefined && r.isRet) ? r.v : undefined
}
