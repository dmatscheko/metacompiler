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
var objHasOwn = Object.prototype.hasOwnProperty // See hasOwn below.
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
    // A grammar may map imports to project files (searched under the program's
    // directory and the -i roots): core.importFile parses and walks the file
    // with this grammar and returns true when it took the import.
    if (core.importFile != null && core.importFile(path, pos)) { return }
    var where = c.curFile() + ":" + c.lineOf(pos)
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

// ----- -main snippet: run a fragment of the language as the entry point -----
// A -main value is a snippet (a statement to parse and run) rather than a bare function
// name exactly when it contains a '(' - a call, which no bare identifier has.
function mainIsSnippet(s) {
    for (var i = 0; i < s.length; i++) { if (s.charCodeAt(i) == 40) { return true } }
    return false
}
// Parse the -main snippet from the grammar's stmtRule production, compile it, and run the
// resulting statement thunk in the current (already-populated) scope; returns its value.
function runMainSnippet(snippet, stmtRule) {
    var frag = c.parseFrom(c.agrammar, snippet, stmtRule)
    return c.compile(frag).stack[0]()
}

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

// setVar and getVar walk the scope chain on every write and read of the
// interpreted program, so both keep `scopes` and the hasOwn function in locals
// (under the frozen engine every mention of a global is a scope-chain lookup of
// its own) and call the latter directly rather than through hasOwn. The test
// itself is unchanged: a variable may legitimately hold undefined, so membership
// is what decides, innermost scope first.
function setVar(name, value) {
    if (name == core.blank) { return }
    var sc = scopes
    var ho = objHasOwn
    for (var i = sc.length - 1; i >= 0; i--) {
        if (ho.call(sc[i], name)) { scopePut(sc[i], name, value); return }
    }
    if (core.setMiss != null && core.setMiss(name, value)) { return }
    fail("assignment to unknown variable: " + name)
}

// pop() yields a tag's pushed values last-first, so the list has to be turned
// around. unshift() is O(n) per element and the ASG has nodes with over a
// thousand children, so it collects with push and reverses once - O(n) instead
// of O(n^2). reverse() is a host builtin on both engines, so the turnaround
// stays one Go-speed pass rather than a per-element loop of externs.
function takeAll() {
    var items = []
    var v = anytype // The tags push values of every type.
    while ((v = pop()) != null) items.push(v)
    return items.reverse()
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

// objHasOwn is Object.prototype.hasOwnProperty, resolved once at include time:
// hasOwn runs per scope level of every variable access, and the three member
// reads that spell the name out (Object -> prototype -> hasOwnProperty) cost more
// than the test itself. The name hasOwn stays a plain function so grammars can
// call (or override) it as before.
function hasOwn(o, name) { return objHasOwn.call(o, name) }

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
    if (t == "Int" || t == "Integer" || t == "Long" || t == "Short" || t == "Byte" || t == "Char" || t == "Character") { return typeof v == "number" && Math.floor(v) == v }
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

// See setVar above for the shape of the walk.
function getVar(name) {
    var sc = scopes
    var ho = objHasOwn
    for (var i = sc.length - 1; i >= 0; i--) {
        if (ho.call(sc[i], name)) return sc[i][name]
    }
    if (core.varMiss != null) { var h = core.varMiss(name); if (h != null) return h.v }
    if (hasOwn(hostGlobals, name)) return hostGlobals[name]
    fail("unknown name: " + name)
}

function makeConst(v) { return function() { return v } }

function makeVarRef(name) { return function() { return getVar(name) } }

function makeNeg(t) { return function() { return (0 - t()) | 0 } }

function makeNot(t) { return function() { return !t() } }

// The operator of a fold is known when the thunk is BUILT, so the two-operand
// case - by far the most common shape - resolves it there and evaluates with a
// direct call and no operator test. Longer folds keep the accumulator loop
// (nesting them into pairs costs more calls than the dispatch saves) but bind
// binOp once. Whatever binOp is current at build time is what runs: every grammar
// installs its override in the startScript, long before c.compile(c.asg) builds
// any thunk.
function foldBinary(items) {
    if (items.length == 1) return items[0]
    if (items.length == 3) { return foldBinary2(items[0], items[1], items[2]) }
    var f = binOp
    return function() {
        var v = anytype // A comparison fold turns the running number into a boolean.
        v = items[0]()
        for (var i = 1; i < items.length; i += 2) v = f(items[i], v, items[i + 1]())
        return v
    }
}
// One binary step: the concrete function(l, r) when the base binOp is still in
// place, and the operator bound into the closure otherwise.
function foldBinary2(lt, op, rt) {
    var g = opResolve(op)
    if (g != null) { return function() { return g(lt(), rt()) } }
    var f = binOp
    return function() { return f(op, lt(), rt()) }
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

// binOps is the base operator table: one function(l, r) per operator, built once
// instead of an if-chain walked per evaluation. binOp stays the entry point every
// grammar knows - 11 of them replace it, two more wrap it and delegate the rest
// here - and a grammar that only ADDS operators can equally well put them in
// binOps, which keeps them on the resolved path below.
var binOps = {}
binOps["+"]  = function(l, r) { return core.add(l, r) }
binOps["-"]  = function(l, r) { return (l - r) | 0 }
binOps["*"]  = function(l, r) { return (l * r) | 0 }
binOps["/"]  = function(l, r) { return (l / r) | 0 }
binOps["%"]  = function(l, r) { return (l % r) | 0 }
binOps["=="] = function(l, r) { return l === r }
binOps["!="] = function(l, r) { return l !== r }
binOps["<"]  = function(l, r) { return l < r }
binOps[">"]  = function(l, r) { return l > r }
binOps["<="] = function(l, r) { return l <= r }
binOps[">="] = function(l, r) { return l >= r }

// A bare binOps[op] would resolve Object.prototype's members under a host JS
// engine (binOps["toString"] is a function there, binOps["__proto__"] an object),
// so the function-valued ones are shadowed with undefined once and the typeof
// test rejects everything else. That keeps the lookup a single property read -
// hasOwn per call would cost more than the if-chain it replaces.
var opShadow = ["constructor", "hasOwnProperty", "isPrototypeOf", "propertyIsEnumerable",
                "toLocaleString", "toString", "valueOf",
                "__defineGetter__", "__defineSetter__", "__lookupGetter__", "__lookupSetter__"]
for (var opsi = 0; opsi < opShadow.length; opsi++) { binOps[opShadow[opsi]] = undefined }

function binOp(op, l, r) {
    var f = binOps[op]
    if (typeof f == "function") { return f(l, r) }
    return undefined
}
// The base binOp, kept for the identity test in opResolve. A grammar's override
// is installed on binOp, so binOp !== coreBinOp says "someone took over".
var coreBinOp = binOp

// opResolve returns op's concrete function(l, r), or null when the caller has to
// keep going through binOp(op, l, r) - either because a grammar overrode it or
// because the operator has no table entry. Fail-safe: a null just keeps the old
// path, so an engine that cannot compare the two function values loses the
// optimization rather than the behavior.
function opResolve(op) {
    if (binOp !== coreBinOp) { return null }
    var f = binOps[op]
    if (typeof f != "function") { return null }
    return f
}

// The compound-assignment twin of binOps. Its operator arrives at run time (the
// assignment thunks live in the grammars), so only the dispatch is table-driven.
var applyOps = {}
applyOps["+="] = function(a, b) { return core.add(a, b) }
applyOps["-="] = function(a, b) { return (a - b) | 0 }
applyOps["*="] = function(a, b) { return (a * b) | 0 }
applyOps["/="] = function(a, b) { return (a / b) | 0 }
applyOps["%="] = function(a, b) { return (a % b) | 0 }
for (var opsj = 0; opsj < opShadow.length; opsj++) { applyOps[opShadow[opsj]] = undefined }

function applyOp(op, a, b) {
    var f = applyOps[op]
    if (typeof f == "function") { return f(a, b) }
    return undefined
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

// ----- Dictionaries: an insertion-ordered {__dict, keys, vals} box with two parallel
// arrays (mirrors the compiled runtime's js_dict handle). Any ===-comparable key works
// and keys keep their insertion order. Shared by the languages whose value model has a
// distinct dict/map type (python, ruby, php, swift, dart, c#); each language file adds
// its own literal/constructor builders (newDict, makeMap, makeDict, ...) on top.
function isDict(v) { return v !== null && v !== undefined && typeof v == "object" && v.__dict === true }

// The keys array carries the insertion order, so a lookup cannot reorder it -
// but scanning it makes every dict-heavy program quadratic. dictIndex keeps a
// position index in a hidden __idx property next to the arrays: {m: positions,
// n: the keys.length it describes, all: whether every key made it in}. It is a
// pure accelerator - a length change rebuilds it, a hit is confirmed against
// the array before it is used, and keys it cannot represent leave `all` false
// so a miss still falls back to the scan.
// dictKeyId is that representation: the type prefix keeps 1 and "1" apart, and
// keys whose === is not string equality (objects, NaN) stay out.
function dictKeyId(k) {
    var t = typeof k
    if (t == "string") { return "s" + k }
    if (t == "number") { return (k !== k) ? null : "n" + k }
    if (t == "boolean") { return "b" + k }
    return null
}
function dictIndex(d) {
    var ix = hasOwn(d, "__idx") ? d.__idx : null
    if (ix != null && ix.n == d.keys.length) { return ix }
    ix = {m: {}, n: d.keys.length, all: true}
    for (var i = 0; i < d.keys.length; i++) {
        var id = dictKeyId(d.keys[i])
        if (id == null) { ix.all = false }
        else if (!hasOwn(ix.m, id)) { ix.m[id] = i }   // Only the first entry of a key.
    }
    d.__idx = ix
    return ix
}
function dictFind(d, k) {
    var id = dictKeyId(k)
    var ix = dictIndex(d)
    if (id != null && hasOwn(ix.m, id)) {
        var i = ix.m[id]
        if (d.keys[i] === k) { return i }
        d.__idx = null                                  // An entry changed in place.
        ix = dictIndex(d)
        if (hasOwn(ix.m, id) && d.keys[ix.m[id]] === k) { return ix.m[id] }
        return -1
    }
    if (id != null && ix.all) { return -1 }
    for (var j = 0; j < d.keys.length; j++) { if (d.keys[j] === k) { return j } }
    return -1
}
function dictSet(d, k, v) {
    var i = dictFind(d, k)
    if (i >= 0) { d.vals[i] = v; return }
    d.keys.push(k)
    d.vals.push(v)
    var ix = hasOwn(d, "__idx") ? d.__idx : null        // dictFind just left it current.
    if (ix != null && ix.n == d.keys.length - 1) {
        var id = dictKeyId(k)
        if (id == null) { ix.all = false }
        else if (!hasOwn(ix.m, id)) { ix.m[id] = d.keys.length - 1 }
        ix.n = d.keys.length
    }
}
