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
    setMiss: null,      // Assignment target not in any scope: handle it and return true (Kotlin).
    // Map-key identity, for a language whose numbers are BOXED. dictKeyId below can
    // only index a value whose === is string equality, and dictFind confirms a hit
    // with ===, so two boxes holding the same number are two different keys. A
    // language that boxes (Dart: {__flo} for a double, {__sz} for a 64-bit int, and
    // 1 == 1.0 so they must key alike) sets these two; every other language leaves
    // them null and gets exactly the code path it had before.
    keyId: null,        // (k) -> a string id, or null to fall through to dictKeyId.
    keyEq: null,        // (a, b) -> whether two keys are the same key, when a !== b.
    // A null/undefined CONTAINER met while walking an assignment path (resolveRef).
    // The default is null and resolveRef then fails exactly as it always did, so the
    // five languages that do not set it are untouched. Go sets it because a nil
    // pointer dereference is a RECOVERABLE runtime panic there and an abort here was
    // a three-way split against llvm.Run and the native binary - docs/todo.md 1.5.
    nilPath: null       // (name) -> raise the language's own runtime error, or null.
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
    if (c.warnImports) { eprintln("warning: " + where + ": unresolved import '" + path + "' (ignored)"); return }
    fail("unresolved import '" + path + "' (" + where + "); use -warn-imports to ignore")
}
// A construct that parsed but cannot be lowered. Default: abort with a clean
// file:line message; under -warn-unsupported warn and let the caller place a
// placeholder so the rest still runs (enough for call graphs / CFGs / traces).
function notImpl(construct, pos) {
    var where = c.file + ":" + c.lineOf(pos)
    if (c.warnUnsupported) { eprintln("warning: " + where + ": " + construct + " not implemented (ignored)"); return }
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
// The same clause with a BINDER instead of a name, for a language whose catch parameter
// may be a destructuring pattern ('catch ([a, ...b])' in JavaScript). The grammar passes
// a function of the thrown value rather than a pattern node, so interp-core needs no
// knowledge of any grammar's pattern machinery; excTry calls it in the freshly pushed
// catch scope, exactly where declVar would have run.
// items = [binderFunction, blockThunk].
function excCatchBind(items) {
    return {catchbody: items[1], catchname: undefined, catchbind: items[0]}
}
// items = [{trybody}, {catchbody,catchname}*, {finbody}?]. The first catch clause wins
// (exception types cannot be discriminated without runtime types), finally always runs
// and its own control-flow signal overrides.
function excTry(items) {
    var tryT = anytype, catchT = anytype, catchName = anytype, finallyT = anytype
    var catchBind = anytype // A destructuring catch parameter (see excCatchBind).
    for (var i = 0; i < items.length; i++) {
        if (items[i].trybody != undefined) { tryT = items[i].trybody }
        else if (items[i].catchbody != undefined) {
            if (catchT == undefined) {
                catchT = items[i].catchbody
                catchName = items[i].catchname
                if (hasOwn(items[i], "catchbind")) { catchBind = items[i].catchbind }
            }
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
                if (catchBind != undefined) { catchBind(e.v) }
                else if (catchName != undefined) { declVar(catchName, e.v) }
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

// The same walk as getVar, answering whether the name resolves instead of its value
// and never aborting. A grammar needs this when a construct is only PARTLY genuine -
// Kotlin's `recv::member` is a real bound property reference when `recv` is a value
// and a not-implemented type-level reference (`String::length`) when it is a type
// name. Without the test the not-implemented arm would evaluate its operand first
// and abort with "unknown name: String" before notImpl ever ran, which is exactly
// how the two halves of a language drift: the compiler warns and places a
// placeholder, the interpreter dies on the same line.
function hasVar(name) {
    var sc = scopes
    var ho = objHasOwn
    for (var i = sc.length - 1; i >= 0; i--) {
        if (ho.call(sc[i], name)) return true
    }
    if (core.varMiss != null && core.varMiss(name) != null) return true
    return hasOwn(hostGlobals, name)
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

// The receiver-prepend of every instance method call ([target] + args). concat is
// one host call for the whole job, where the push loop was one per element - and
// under -frozen every one of those is an extern into the handle runtime.
function concat2impl(a, b) {
    return a.concat(b)
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
        if (o === undefined || o === null) {
            if (core.nilPath != null) { core.nilPath(ref.name) }
            fail(core.nullWord + " in assignment path of " + ref.name)
        }
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
    if (core.keyId != null) {
        var ck = core.keyId(k)
        if (ck != null) { return ck }
    }
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
// dictKeyEq is the confirmation step: === unless the language declared a key
// equality of its own (core.keyEq), which only a boxed-number language needs.
function dictKeyEq(a, b) {
    if (a === b) { return true }
    if (core.keyEq != null) { return core.keyEq(a, b) }
    return false
}
function dictFind(d, k) {
    var id = dictKeyId(k)
    var ix = dictIndex(d)
    if (id != null && hasOwn(ix.m, id)) {
        var i = ix.m[id]
        if (dictKeyEq(d.keys[i], k)) { return i }
        d.__idx = null                                  // An entry changed in place.
        ix = dictIndex(d)
        if (hasOwn(ix.m, id) && dictKeyEq(d.keys[ix.m[id]], k)) { return ix.m[id] }
        return -1
    }
    if (id != null && ix.all) { return -1 }
    for (var j = 0; j < d.keys.length; j++) { if (dictKeyEq(d.keys[j], k)) { return j } }
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

// ----- Typed catch clauses (shared) -----
// excTry above always runs the FIRST catch clause: the original interpreters had no
// runtime type test, so a `catch (e: IOException)` and a `catch (e: Exception)` could
// not be told apart. excTryTyped is the same protocol with that one difference - a
// clause may carry the DECLARED type of its catch parameter as `catchtype`, and it
// only runs when core.excMatches(thrownValue, catchtype) accepts it. Every language
// here whose grammar spells that type out - Java, Kotlin, C#, Swift, Dart, PHP,
// Python - can move its try/catch production onto this function by recording the type
// name in the clause; the rest of the shape is unchanged, so excCatch's existing
// {catchbody, catchname} entries keep working.
//
// A clause with no catchtype catches everything (Kotlin's `catch (e)`, Python's bare
// `except:`), the clauses are tried in source order and the first match wins, an
// unmatched exception is rethrown so an outer try can still see it, and `finally`
// always runs and its own control-flow signal overrides - exactly as in excTry. A
// language that has not set core.excMatches gets first-clause-wins behavior back, so
// adopting this rule is safe before the predicate exists.
//
// items = [{trybody}, {catchbody, catchname, catchtype?}*, {finbody}?].
function excTryTyped(items) {
    var tryT = anytype, finallyT = anytype
    var clauses = []
    for (var i = 0; i < items.length; i++) {
        if (items[i].trybody != undefined) { tryT = items[i].trybody }
        else if (items[i].catchbody != undefined) { clauses.push(items[i]) }
        else if (items[i].finbody != undefined) { finallyT = items[i].finbody }
    }
    return function() {
        var savedChain = scopes.slice()
        var box = {sig: undefined}
        try {
            box.sig = tryT()
        } catch (e) {
            scopes = savedChain.slice()
            var hit = null
            if (excIsUser(e)) {
                for (var k = 0; k < clauses.length; k++) {
                    if (clauses[k].catchtype == undefined || core.excMatches == null
                        || core.excMatches(e.v, clauses[k].catchtype)) { hit = clauses[k]; break }
                }
            }
            if (hit == null) { throw e }
            scopes.push({})
            if (hit.catchname != undefined) { declVar(hit.catchname, e.v) }
            box.sig = hit.catchbody()
            scopes.pop()
        } finally {
            scopes = savedChain.slice()
            if (finallyT != undefined) {
                var fr = finallyT()
                // Returning from the host finally overrides the try/catch completion
                // AND cancels a rethrown exception, like in JS.
                if (fr != undefined) { return fr }
            }
        }
        return box.sig
    }
}

// ----- Exact 64-bit integers (shared) -----
// A JS number carries 53 bits exactly, so an int64 value above 2^53 cannot be held
// in one: 9223372036854775807 reads back as 9223372036854776000 and
// int64max + 1 answers -1 instead of -9223372036854775808. The statically typed
// languages need the real answer, so a value that does not fit is carried as a pair
// of UNSIGNED 32-bit halves {h, l} and every operator below works on that pair.
//
// Signedness and WIDTH are deliberately not part of the pair - they belong to the
// sized-integer box a language grammar puts around it, exactly as the {__flo} box
// carries floatness. The same pair therefore serves int8 through uint64.
//
// Validated against `go run` over 156490 random and edge-case vectors (every
// operator below, all four widths, both signednesses) and run byte-identically
// under goja, the frozen MetaJS engine and node - so no BigInt, no for..in, no
// splice, no string.split appears here.

function i64Make(h, l) { return {h: h >>> 0, l: l >>> 0} }

// i64FromNum truncates toward zero and wraps modulo 2^64, which is what every
// Go/Java/C# conversion to a 64-bit integer type does.
function i64FromNum(n) {
    if (n !== n) { return i64Make(0, 0) }
    var neg = n < 0
    var a = neg ? -n : n
    a = Math.floor(a)
    // Reduce modulo 2^64 before splitting: a bigger double would give a
    // meaningless high half.
    if (a >= 18446744073709551616) { a = a - Math.floor(a / 18446744073709551616) * 18446744073709551616 }
    var h = Math.floor(a / 4294967296)
    var l = a - h * 4294967296
    var v = i64Make(h, l)
    if (neg) { v = i64Neg(v) }
    return v
}

// i64ToNumS / i64ToNumU are the nearest double. Exact while |value| <= 2^53.
function i64ToNumS(x) {
    if ((x.h & 0x80000000) !== 0) {
        var n = i64Neg(x)
        return -(n.h * 4294967296 + n.l)
    }
    return x.h * 4294967296 + x.l
}
function i64ToNumU(x) { return x.h * 4294967296 + x.l }

// ----- add / sub / neg -----
function i64Add(a, b) {
    var l = a.l + b.l
    var carry = l >= 4294967296 ? 1 : 0
    var h = a.h + b.h + carry
    return i64Make(h >>> 0, (l >>> 0))
}
function i64Not(a) { return i64Make(~a.h, ~a.l) }
function i64Neg(a) { return i64Add(i64Not(a), i64Make(0, 1)) }
function i64Sub(a, b) { return i64Add(a, i64Neg(b)) }

// ----- multiply -----
// Four 16-bit limbs per operand; each partial product is below 2^32 and every
// intermediate stays below 2^53, so the double arithmetic is exact.
function i64Mul(a, b) {
    var a0 = a.l & 0xffff, a1 = a.l >>> 16, a2 = a.h & 0xffff, a3 = a.h >>> 16
    var b0 = b.l & 0xffff, b1 = b.l >>> 16, b2 = b.h & 0xffff, b3 = b.h >>> 16
    var c0 = a0 * b0
    var c1 = (c0 >>> 16) + a1 * b0
    var c2 = c1 >>> 16
    c1 = (c1 & 0xffff) + a0 * b1
    c2 = c2 + (c1 >>> 16) + a2 * b0
    var c3 = c2 >>> 16
    c2 = (c2 & 0xffff) + a1 * b1
    c3 = c3 + (c2 >>> 16)
    c2 = (c2 & 0xffff) + a0 * b2
    c3 = c3 + (c2 >>> 16)
    c3 = c3 + a3 * b0 + a2 * b1 + a1 * b2 + a0 * b3
    return i64Make(((c2 & 0xffff) | ((c3 & 0xffff) << 16)) >>> 0,
                   ((c0 & 0xffff) | ((c1 & 0xffff) << 16)) >>> 0)
}

// ----- shifts (count already masked to 0..63 by the caller) -----
function i64Shl(a, n) {
    n = n & 63
    if (n === 0) { return i64Make(a.h, a.l) }
    if (n < 32) { return i64Make((a.h << n) | (a.l >>> (32 - n)), a.l << n) }
    return i64Make(a.l << (n - 32), 0)
}
function i64ShrU(a, n) {
    n = n & 63
    if (n === 0) { return i64Make(a.h, a.l) }
    if (n < 32) { return i64Make(a.h >>> n, (a.l >>> n) | (a.h << (32 - n))) }
    return i64Make(0, a.h >>> (n - 32))
}
function i64ShrS(a, n) {
    n = n & 63
    if (n === 0) { return i64Make(a.h, a.l) }
    if (n < 32) { return i64Make(a.h >> n, (a.l >>> n) | (a.h << (32 - n))) }
    return i64Make(a.h >> 31, a.h >> (n - 32))
}

// ----- bitwise -----
function i64And(a, b) { return i64Make(a.h & b.h, a.l & b.l) }
function i64Or(a, b)  { return i64Make(a.h | b.h, a.l | b.l) }
function i64Xor(a, b) { return i64Make(a.h ^ b.h, a.l ^ b.l) }

// ----- comparison -----
function i64IsZero(a) { return a.h === 0 && a.l === 0 }
function i64Eq(a, b) { return a.h === b.h && a.l === b.l }
function i64CmpU(a, b) {
    if (a.h !== b.h) { return (a.h >>> 0) < (b.h >>> 0) ? -1 : 1 }
    if (a.l !== b.l) { return (a.l >>> 0) < (b.l >>> 0) ? -1 : 1 }
    return 0
}
function i64CmpS(a, b) {
    var an = (a.h & 0x80000000) !== 0
    var bn = (b.h & 0x80000000) !== 0
    if (an !== bn) { return an ? -1 : 1 }
    return i64CmpU(a, b)
}

// ----- division -----
// i64DivModSmall divides an UNSIGNED pair by a positive divisor below 2^31,
// walking the value in 16-bit chunks so every intermediate stays below 2^47.
// Returns {q: pair, r: number}.
function i64DivModSmall(a, d) {
    var parts = [a.h >>> 16, a.h & 0xffff, a.l >>> 16, a.l & 0xffff]
    var out = [0, 0, 0, 0]
    var rem = 0
    for (var i = 0; i < 4; i++) {
        var cur = rem * 65536 + parts[i]
        var q = Math.floor(cur / d)
        rem = cur - q * d
        out[i] = q
    }
    return {q: i64Make(((out[0] * 65536) + out[1]) >>> 0, ((out[2] * 65536) + out[3]) >>> 0), r: rem}
}

// i64DivModU is unsigned 64/64 by shift-and-subtract. Only reached when the
// divisor does not fit the small path, so the 64 iterations are rare.
function i64DivModU(a, b) {
    if (i64IsZero(b)) { return {q: i64Make(0, 0), r: i64Make(0, 0)} }
    if (b.h === 0 && (b.l >>> 0) < 2147483648 && b.l !== 0) {
        var s = i64DivModSmall(a, b.l >>> 0)
        return {q: s.q, r: i64Make(0, s.r)}
    }
    var q = i64Make(0, 0)
    var r = i64Make(0, 0)
    for (var i = 63; i >= 0; i--) {
        r = i64Shl(r, 1)
        var bit = (i >= 32) ? ((a.h >>> (i - 32)) & 1) : ((a.l >>> i) & 1)
        r = i64Or(r, i64Make(0, bit))
        if (i64CmpU(r, b) >= 0) {
            r = i64Sub(r, b)
            q = i64Or(q, i64Shl(i64Make(0, 1), i))
        }
    }
    return {q: q, r: r}
}

// i64DivModS is Go's / and % on signed 64-bit: truncation toward zero, so the
// remainder takes the DIVIDEND's sign. Division of the most negative value by
// -1 wraps back to itself, which is what the hardware does.
function i64DivModS(a, b) {
    var an = (a.h & 0x80000000) !== 0
    var bn = (b.h & 0x80000000) !== 0
    var ua = an ? i64Neg(a) : a
    var ub = bn ? i64Neg(b) : b
    var d = i64DivModU(ua, ub)
    var q = d.q
    var r = d.r
    if (an !== bn) { q = i64Neg(q) }
    if (an) { r = i64Neg(r) }
    return {q: q, r: r}
}

// ----- text -----
function i64StrU(a) {
    if (i64IsZero(a)) { return "0" }
    var s = ""
    var v = a
    while (!i64IsZero(v)) {
        var d = i64DivModSmall(v, 1000000000)
        v = d.q
        if (i64IsZero(v)) { s = "" + d.r + s }
        else {
            var chunk = "" + d.r
            while (chunk.length < 9) { chunk = "0" + chunk }
            s = chunk + s
        }
    }
    return s
}
function i64StrS(a) {
    if ((a.h & 0x80000000) !== 0) { return "-" + i64StrU(i64Neg(a)) }
    return i64StrU(a)
}

// i64FromStr reads a decimal digit run (no sign, no separators - the caller has
// already stripped them). Overflow wraps, like the arithmetic above.
function i64FromStr(s) {
    var v = i64Make(0, 0)
    var ten = i64Make(0, 10)
    for (var i = 0; i < s.length; i++) {
        var c = s.charCodeAt(i)
        if (c < 48 || c > 57) { continue }
        v = i64Add(i64Mul(v, ten), i64Make(0, c - 48))
    }
    return v
}
// i64FromRadix reads a digit run in base 2, 8 or 16.
function i64FromRadix(s, base) {
    var v = i64Make(0, 0)
    var bb = i64Make(0, base)
    for (var i = 0; i < s.length; i++) {
        var c = s.charCodeAt(i)
        var d = -1
        if (c >= 48 && c <= 57) { d = c - 48 }
        else if (c >= 97 && c <= 122) { d = c - 87 }
        else if (c >= 65 && c <= 90) { d = c - 55 }
        if (d < 0 || d >= base) { continue }
        v = i64Add(i64Mul(v, bb), i64Make(0, d))
    }
    return v
}

// ----- width -----
// i64Trunc reduces a pair to `w` bits and re-extends it: sign extension for a
// signed width, zero extension for an unsigned one. w is 8, 16, 32 or 64.
function i64Trunc(a, w, unsigned) {
    if (w >= 64) { return a }
    if (w === 32) {
        if (unsigned) { return i64Make(0, a.l) }
        return i64Make((a.l & 0x80000000) !== 0 ? 0xffffffff : 0, a.l)
    }
    var mask = w === 8 ? 0xff : 0xffff
    var lo = a.l & mask
    if (unsigned) { return i64Make(0, lo) }
    var sign = w === 8 ? 0x80 : 0x8000
    if ((lo & sign) !== 0) { return i64Make(0xffffffff, (lo | ~mask) >>> 0) }
    return i64Make(0, lo)
}

// i64Fits53 says whether a pair's SIGNED value lies in [-2^53, 2^53], the range a
// JS number holds exactly. It is the test the sized-integer box below uses to
// decide between a plain number and a box, and it has to read the pair rather
// than the double: converting first would round 2^53 + 1 down to 2^53 and answer
// yes for a value that does not fit.
function i64Fits53(p) {
    var q = ((p.h & 0x80000000) !== 0) ? i64Neg(p) : p
    if ((q.h & 0x80000000) !== 0) { return false }      // only -2^63 negates to itself
    if (q.h < 0x200000) { return true }
    return q.h === 0x200000 && q.l === 0
}

// ----- The sized-integer box (shared) -----
// Go, Java, Kotlin and C# all have integer types of a DECLARED WIDTH, and the
// width has to survive into the next operation: `var i8 int8 = 127; i8++` is -128
// and nothing about the value 127 says so. This is the same problem the {__flo}
// box solves for floating point, and it gets the same answer - the type goes ON
// the value.
//
// Invariant, in every grammar that opts in (and in the compiled half, see
// abnf/jsrtint.go, which implements exactly these rules):
//
//     a plain number   ==  a SIGNED 64 bit integer inside [-2^53, 2^53]
//     an {__sz} box    ==  every other integer: a sized type, any unsigned type,
//                          or a 64 bit value that has left that range
//     an {__flo} box   ==  a float
//
// Keeping the ordinary case a plain number is what makes this affordable: a
// program that never writes a sized type and never leaves 2^53 allocates no box
// and runs the arithmetic it ran before, one magnitude test heavier. The box is
// rare, so the places that consume a number raw - an array index, a slice bound,
// a map key - meet one only in the programs that asked for it.
var SZ_EXACT = 9007199254740992      // 2^53

function szMake(p, w, u) { return {__sz: true, p: p, w: w, u: u} }
function szIs(v) { return v != undefined && v != null && typeof v == "object" && v.__sz === true }

// szNorm applies the invariant: truncate to the width, then answer a plain number
// when the result is a signed 64 bit value a double holds exactly.
function szNorm(p, w, u) {
    var q = i64Trunc(p, w, u)
    if (w === 64 && !u && i64Fits53(q)) { return i64ToNumS(q) }
    return szMake(q, w, u)
}
// szPair is the 64 bit pair of any integral operand.
function szPair(v) {
    if (szIs(v)) { return v.p }
    return i64FromNum(typeof v == "number" ? v : 0)
}
// szWidth / szUns are the RESULT type of a binary operation. Go, Java, Kotlin and
// C# all require both operands of an arithmetic operator to have the same type,
// so at most one side is a box and it decides; the left one wins if both are,
// which is the rule jvmStyleOf uses for the float box.
function szWidth(l, r) {
    if (szIs(l)) { return l.w }
    if (szIs(r)) { return r.w }
    return 64
}
function szUns(l, r) {
    if (szIs(l)) { return l.u }
    if (szIs(r)) { return r.u }
    return false
}
// szNum is the JS number reading - exact below 2^53, the nearest double above it.
// Every consumer that needs a plain number (an index, a length, a bound, a
// float conversion) goes through it.
function szNum(v) {
    if (!szIs(v)) { return v }
    if (v.u) { return i64ToNumU(v.p) }
    return i64ToNumS(v.p)
}
// szStr is the decimal text. All four languages spell an integer the same way, so
// this is shared; only the unsigned reading differs from the signed one.
function szStr(v) {
    if (!szIs(v)) { return "" + v }
    return v.u ? i64StrU(v.p) : i64StrS(v.p)
}
// szConv is an explicit conversion to a sized type: int8(x), uint32(x), int64(x).
function szConv(v, w, u) { return szNorm(szPair(v), w, u) }

// szDivN / szModN are truncated division and remainder on two plain numbers, both
// known to be integers inside [-2^53, 2^53]. The obvious Math.trunc(a / b) is not
// safe there: near 2^53 one ulp is 2, so a correctly rounded quotient can land on
// the wrong side of an integer. Dividing MAGNITUDES and correcting once is exact,
// because the corrected product stays below the dividend.
function szDivN(a, b) {
    var sa = a < 0 ? 0 - 1 : 1
    var sb = b < 0 ? 0 - 1 : 1
    var ma = a * sa
    var mb = b * sb
    var mq = Math.floor(ma / mb)
    var mr = ma - mq * mb
    if (mr < 0) { mq = mq - 1 }
    else if (mr >= mb) { mq = mq + 1 }
    return mq * sa * sb
}
function szModN(a, b) {
    var q = szDivN(a, b)
    return a - q * b
}

// szArith is one binary arithmetic or bitwise operator. Two plain operands take
// the FAST PATH - ordinary double arithmetic, which is exact while the result
// stays inside 2^53 - and only a result that leaves the range, or an operand that
// is already a box, pays for the exact 64 bit pair arithmetic.
function szArith(op, l, r) {
    if (!szIs(l) && !szIs(r) && typeof l == "number" && typeof r == "number") {
        var v = 0
        if (op == "+") { v = l + r }
        else if (op == "-") { v = l - r }
        else if (op == "*") { v = l * r }
        else if (op == "/") { if (r === 0) { fail("integer divide by zero") }; return szDivN(l, r) }
        else if (op == "%") { if (r === 0) { fail("integer divide by zero") }; return szModN(l, r) }
        else { return szArithSlow(op, l, r) }
        // STRICTLY inside 2^53, not up to it: a true result of 2^53 + 1 rounds
        // DOWN to 2^53 here, so a `<=` test would accept the rounded value as
        // exact. Below the bound the double result is exact (both operands are
        // integers and the sum/difference/product is representable), and 2^53
        // itself is handed to the exact path, which answers the same number.
        if (v < SZ_EXACT && v > 0 - SZ_EXACT) { return v }
        return szArithSlow(op, l, r)
    }
    return szArithSlow(op, l, r)
}

// szArithSlow is the exact path: everything happens on 64 bit pairs at the result
// type's width and wraps to it.
function szArithSlow(op, l, r) {
    var w = szWidth(l, r)
    var u = szUns(l, r)
    // A SHIFT is the one operator whose result type is the LEFT operand's ALONE -
    // the count is a separate operand with a type of its own (Go, Arithmetic
    // operators; JLS 15.19; ECMA-334 12.11; Kotlin's shl/shr/ushr take an Int;
    // Swift's `>> <RHS: BinaryInteger>` answers Self). szWidth/szUns read the type
    // off whichever operand is a box, so a plain `int` shifted by a `uint8` would
    // come out 8 bits UNSIGNED. This is the same guard si_apply carries; see the
    // measurement written out there (it fired zero times across the whole corpus,
    // because every language normalises the count before it gets here).
    if (op == "<<" || op == ">>") { w = szWidth(l, l); u = szUns(l, l) }
    var a = szPair(l)
    var b = szPair(r)
    if (op == "+") { return szNorm(i64Add(a, b), w, u) }
    if (op == "-") { return szNorm(i64Sub(a, b), w, u) }
    if (op == "*") { return szNorm(i64Mul(a, b), w, u) }
    if (op == "&") { return szNorm(i64And(a, b), w, u) }
    if (op == "|") { return szNorm(i64Or(a, b), w, u) }
    if (op == "^") { return szNorm(i64Xor(a, b), w, u) }
    if (op == "&^") { return szNorm(i64And(a, i64Not(b)), w, u) }
    if (op == "/" || op == "%") {
        if (i64IsZero(b)) { fail("integer divide by zero") }
        var d = u ? i64DivModU(a, b) : i64DivModS(a, b)
        return szNorm(op == "/" ? d.q : d.r, w, u)
    }
    // The shift COUNT is a separate operand of its own type; a count at or above
    // the width shifts everything out, which is what Go, Java and C# all specify
    // (and unlike C, is not undefined).
    var n = szNum(r)
    if (op == "<<") {
        if (n < 0 || n >= w) { return szNorm(i64Make(0, 0), w, u) }
        return szNorm(i64Shl(a, n), w, u)
    }
    if (op == ">>") {
        if (n < 0) { return szNorm(i64Make(0, 0), w, u) }
        if (u) {
            if (n >= w) { return szNorm(i64Make(0, 0), w, u) }
            return szNorm(i64ShrU(i64Trunc(a, w, u), n), w, u)
        }
        if (n >= w) { n = w - 1 }
        return szNorm(i64ShrS(a, n), w, u)
    }
    fail("szArith: unknown operator " + op)
}

// szCmp is the ordered comparison: -1, 0 or 1, unsigned when the result type is,
// so uint64max > 0 rather than -1 < 0.
function szCmp(l, r) {
    if (!szIs(l) && !szIs(r)) { return l < r ? 0 - 1 : (l > r ? 1 : 0) }
    var u = szUns(l, r)
    var w = szWidth(l, r)
    var a = i64Trunc(szPair(l), w, u)
    var b = i64Trunc(szPair(r), w, u)
    return u ? i64CmpU(a, b) : i64CmpS(a, b)
}
// szEq compares two integers by VALUE whatever their widths - which is all a
// well-typed program can observe, since it could not have written the comparison
// if the types disagreed.
function szEq(l, r) {
    if (!szIs(l) && !szIs(r)) { return l === r }
    return i64Eq(i64Trunc(szPair(l), 64, false), i64Trunc(szPair(r), 64, false))
}
// szNeg and szNot are unary minus and the bitwise complement, both at the
// operand's own width.
function szNeg(v) { return szArithSlow("-", szNorm(i64Make(0, 0), szWidth(v, v), szUns(v, v)), v) }
function szNot(v) { return szNorm(i64Not(szPair(v)), szWidth(v, v), szUns(v, v)) }
