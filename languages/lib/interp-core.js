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
    // Both default readers are own-property reads: a bare o[name] resolved
    // inherited Object.prototype members under a host JS engine, so a field
    // named toString that was never written read back as a host function.
    getField: function(o, name) { if (hasOwn(o, name)) { return o[name] }; return undefined },  // .name reads (Java: array .length).
    getIndex: function(o, i) { if (hasOwn(o, i)) { return o[i] }; return undefined },           // [i] reads (Go: map-aware).
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

// ----- Imports & not-implemented syntax (shared policy; see language-widening) -----
// A grammar wires these to its Package/Import/Type-op/... productions and sets the
// resolvable prefixes in core.stdlibImports. Source positions come from c.file /
// c.lineOf(up.pos); the -warn-imports / -warn-unsupported flags arrive as
// c.warnImports / c.warnUnsupported.

// stripWs removes all whitespace from a dotted import path.
function stripWs(s) {
    var out = ""
    for (var i = 0; i < s.length; i++) {
        var cc = s.charCodeAt(i)
        if (cc != 32 && cc != 9 && cc != 10 && cc != 13) { out = out + s.charAt(i) }
    }
    return out
}
// True when path (or its package prefix, ignoring a trailing .*) is a builtin the
// runtime already provides (core.stdlibImports is the per-language resolvable set).
function importResolvable(path) {
    var p = path
    if (p.length >= 2 && p.slice(p.length - 2) == ".*") { p = p.slice(0, p.length - 2) }
    for (var i = 0; i < core.stdlibImports.length; i++) {
        var pref = core.stdlibImports[i]
        if (p == pref) { return true }
        if (p.slice(0, pref.length + 1) == pref + ".") { return true }
    }
    return false
}
// Resolvable imports are ignored (already provided); an unresolvable one aborts,
// or warns and continues under -warn-imports.
function resolveImport(path, pos) {
    path = stripWs(path)
    if (importResolvable(path)) { return }
    var where = c.file + ":" + c.lineOf(pos)
    if (c.warnImports) { println("warning: " + where + ": unresolved import '" + path + "' (ignored)"); return }
    fail("unresolved import '" + path + "' (" + where + "); use -warn-imports to ignore")
}
// A construct that parsed but cannot be lowered. Default: abort with a clean
// file:line message; under -warn-unsupported warn and let the caller place a
// placeholder so the rest still runs (enough for call graphs / CFGs / traces).
function notImpl(construct, pos) {
    var where = c.file + ":" + c.lineOf(pos)
    if (c.warnUnsupported) { println("warning: " + where + ": " + construct + " not implemented (ignored)"); return }
    fail(construct + " not implemented (" + where + "); use -warn-unsupported to ignore")
}
// Placeholders - interpreter thunks are function() -> value / signal.
function notImplStmt(construct, pos) { notImpl(construct, pos); return function() { return undefined } }
function notImplExpr(construct, pos) { notImpl(construct, pos); return function() { return undefined } }

// ----- Exceptions (try/catch/finally/throw), shared across the interpreters -----
// A throw wraps its value in a marker object and raises it as a real host exception,
// so it unwinds through any depth of expression evaluation up to the nearest catch.
// The target program's return/break/continue inside a try are the ORDINARY statement
// signals ({isRet}/BRK/CONT) that the body thunk returns, so they keep propagating;
// only a genuine throw uses host unwinding. On the way out of a throw the scope chain
// is restored to its try-entry snapshot (a call replaces `scopes`, so an interrupted
// call would otherwise leave the callee's chain in place). Distinct names (exc*) keep
// these clear of a grammar's own make* helpers, avoiding the include override trap.
function excThrow(t) { return function() { throw {__exc: true, v: t()} } }
function excIsUser(e) { return e != undefined && e.__exc == true }
// items = [name, blockThunk] or [blockThunk] (no binding).
function excCatch(items) {
    if (items.length > 1) { return {catchbody: items[1], catchname: items[0]} }
    return {catchbody: items[0], catchname: undefined}
}
// items = [{trybody}, {catchbody,catchname}*, {finbody}?]. The first catch clause wins
// (exception types cannot be discriminated without runtime types), finally always runs
// and its own control-flow signal overrides.
function excTry(items) {
    var tryT = anytype, catchT = anytype, catchName = anytype, finallyT = anytype
    for (var i = 0; i < items.length; i++) {
        if (items[i].trybody != undefined) { tryT = items[i].trybody }
        else if (items[i].catchbody != undefined) {
            if (catchT == undefined) { catchT = items[i].catchbody; catchName = items[i].catchname }
        }
        else if (items[i].finbody != undefined) { finallyT = items[i].finbody }
    }
    return function() {
        var savedChain = scopes.slice()
        var box = {sig: undefined}
        try {
            box.sig = tryT()
        } catch (e) {
            scopes = savedChain.slice()
            if (excIsUser(e) && catchT != undefined) {
                scopes.push({})
                if (catchName != undefined) { declVar(catchName, e.v) }
                box.sig = catchT()
                scopes.pop()
            } else {
                throw e
            }
        } finally {
            scopes = savedChain.slice()
            if (finallyT != undefined) {
                var fr = finallyT()
                // Returning from the host finally overrides the try/catch
                // completion AND cancels a rethrown exception, like in JS
                // (box.sig assignment could not stop the host throw).
                if (fr != undefined) { return fr }
            }
        }
        return box.sig
    }
}

// scopePut writes one own property. The single dangerous name is "__proto__":
// a plain write invokes the Object.prototype accessor under a host JS engine
// (a TypeError for primitive values, a silent reparenting of the scope object
// otherwise), while the frozen engine's objects have no prototype chain.
// rawSet (a host global on both engines) defines the own property directly.
function scopePut(obj, name, value) {
    if (name == "__proto__") { rawSet(obj, name, value) } else { obj[name] = value }
}

function declVar(name, value) { if (name != core.blank) scopePut(scopes[scopes.length - 1], name, value) }

function setVar(name, value) {
    if (name == core.blank) { return }
    for (var i = scopes.length - 1; i >= 0; i--) {
        if (hasOwn(scopes[i], name)) { scopePut(scopes[i], name, value); return }
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

// The dynamic type test behind the typed languages' `is` / `instanceof` checks.
// Must match the Go twin exactly (isTypeName in abnf/jsrt.go, extern js_is_type)
// - the compilers call the extern, the interpreters call this. Generic arguments
// are ignored (List<Int> tests as List), a trailing ? lets null match, and user
// classes match on __class.__name (walking __super when present). Integral
// numbers count as Int AND as Double: the value model has one number type.
function rtIsType(v, tname) {
    var t = tname.split("<")[0]
    var opt = false
    if (t.charAt(t.length - 1) == "?") { t = t.substring(0, t.length - 1); opt = true }
    if (v === null || v === undefined) { return opt }
    if (t == "Any" || t == "Object") { return true }
    if (t == "Int" || t == "Long" || t == "Short" || t == "Byte" || t == "Char") { return typeof v == "number" && Math.floor(v) == v }
    if (t == "Double" || t == "Float" || t == "Number") { return typeof v == "number" }
    if (t == "String" || t == "CharSequence") { return typeof v == "string" }
    if (t == "Boolean") { return typeof v == "boolean" }
    if (t == "List" || t == "MutableList" || t == "Collection" || t == "Array") {
        return typeof v == "object" && v !== null && typeof v.length == "number"
    }
    if (typeof v == "object" && v !== null) {
        var cls = v.__class
        while (cls != undefined && cls != null) {
            if (cls.__name == t) { return true }
            cls = cls.__super
        }
    }
    return false
}

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
        if (s.key != undefined) { o = hasOwn(o, s.key) ? o[s.key] : undefined }
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
