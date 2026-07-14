// compile-core.js -- the shared tag-script core for the LLVM-IR compiler grammars.
//
// A language grammar pulls this file in at the top of its startScript with
//     include("lib/compile-core.js")
// and gets the common emitter machinery: the module, the handle constants, extern
// declaration on demand, string/number emission, the loopStack break/continue
// protocol, and the expression/statement thunk builders. Expression thunks are
// function(block) -> {b, v}; statement thunks are function(block) -> nextBlock.
// Everything lands in the grammar's own global scope, so tags call these builders
// directly and the grammar can override any of them by assignment after the include.
//
// Language differences are knobs on `core`, set right before c.compile(c.asg).

// ----- Configuration knobs -----
var core = {
    indexExt: "js_get",       // The external for [i] reads (Go: "js_map_get", map-aware).
    scopeGetExt: "js_scope_get", // The external for reading a name (Kotlin: "js_kget", this-aware).
    truthyExt: "js_truthy"    // The external deciding branch conditions (Python: "js_pytruthy").
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
// ----- Imports & not-implemented syntax (shared policy; see language-widening) -----
// A grammar wires these to its Package/Import/Type-op/... productions and sets the
// resolvable prefixes in core.stdlibImports. Positions come from c.file /
// c.lineOf(up.pos); -warn-imports / -warn-unsupported arrive as c.warnImports /
// c.warnUnsupported. `fail` is provided by the grammar (each compiler defines it).

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
// placeholder so the rest still compiles (enough for call graphs / CFGs / traces).
function notImpl(construct, pos) {
    var where = c.file + ":" + c.lineOf(pos)
    if (c.warnUnsupported) { println("warning: " + where + ": " + construct + " not implemented (ignored)"); return }
    fail(construct + " not implemented (" + where + "); use -warn-unsupported to ignore")
}
// Placeholders - compiler thunks are function(block) -> {b, v} / -> nextBlock.
function notImplStmt(construct, pos) { notImpl(construct, pos); return function(b) { return b } }
function notImplExpr(construct, pos) { notImpl(construct, pos); return function(b) { return {b: b, v: hUndef} } }

// ----- -main snippet: emit a fragment of the language as the entry point -----
// A -main value is a snippet (a statement to parse and emit) rather than a bare function
// name exactly when it contains a '(' - a call, which no bare identifier has.
function mainIsSnippet(s) {
    for (var i = 0; i < s.length; i++) { if (s.charCodeAt(i) == 40) { return true } }
    return false
}
// Parse the -main snippet from the grammar's stmtRule production, compile it, and emit the
// resulting statement thunk into block b (in the current scope); returns the next block.
function emitMainSnippet(b, snippet, stmtRule) {
    var frag = c.parseFrom(c.agrammar, snippet, stmtRule)
    return c.compile(frag).stack[0](b)
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

// ----- The emitter (the same scaffold as metajs-to-llvm-ir.abnf) -----

var i64 = llvm.types.I64
var m = llvm.ir.NewModule()
function handle(n) { return llvm.constant.NewInt(i64, n) }
var hUndef = handle(0)
var hNull = handle(1)
var hFalse = handle(2)
var hTrue = handle(3)

var externs = {}
function getExtern(name, paramCount) {
    var f = externs[name]
    if (f != undefined) return f
    var params = []
    if (name == "js_str_mem") {
        params.push(llvm.ir.NewParam("", llvm.types.I8Ptr))
        params.push(llvm.ir.NewParam("", i64))
    } else {
        for (var i = 0; i < paramCount; i++) params.push(llvm.ir.NewParam("", i64))
    }
    f = m.NewFunc(name, i64, params)
    externs[name] = f
    return f
}
function callExt(b, name, args) {
    return b.NewCall(getExtern(name, args.length), args)
}
var strGlobals = {}
var strCount = 0
function emitStr(b, s) {
    // Prefix the cache key: a bare string like "toString" or "valueOf" would resolve to
    // an inherited Object.prototype member under a host JS engine (goja), so the
    // undefined check would miss and g.ptr/g.len read off a function - malformed IR. The
    // "$" prefix cannot name any Object.prototype property, so the lookup is honest in
    // both the host and the frozen engine.
    var key = "$" + s
    var g = strGlobals[key]
    if (g == undefined) {
        strCount++
        g = {}
        g.glob = m.NewGlobalDef("str." + strCount, llvm.constant.NewCharArrayFromString(s))
        // byteLen, not s.length: the char array holds UTF-8 bytes, but .length
        // counts UTF-16 code units, so a non-ASCII constant was truncated to
        // its first bytes ("é" arrived as the single byte 0xC3).
        g.len = handle(byteLen(s))
        g.ptr = llvm.constant.NewGetElementPtr(g.glob.ContentType, g.glob, [handle(0), handle(0)])
        strGlobals[key] = g
    }
    return callExt(b, "js_str_mem", [g.ptr, g.len])
}
function emitNum(b, n) {
    if (n % 1 == 0 && n < 9007199254740992 && n > -9007199254740992) {
        return callExt(b, "js_num_i", [handle(n)])
    }
    return callExt(b, "js_num_str", [emitStr(b, "" + n)])
}

var curF = null
var curScopeV = null
var loopStack = []
var funcCount = 0
var deadCount = 0

// ----- Non-local control flow out of a try body -----
// A try/catch/finally body compiles to its own IR closure (so a Go recover can wrap
// it), so a return/break/continue that LEAVES the body cannot ret/br the enclosing
// frame directly. Instead such a jump returns a control signal (js_ctl_*), which
// js_try passes through and excDispatch (emitted after the js_try call) turns back
// into a real ret/br in the enclosing function/loop.
// ctlStack holds the function tag of each active try-body closure; curFTag is the tag
// of the function currently being emitted. A grammar's function emitter maintains
// curFTag (save/restore around each function) and, for a try body, pushes its tag
// with ctlEnter/ctlLeave. inCtlBody() is then true exactly when the code being emitted
// sits DIRECTLY in a try-body closure (a nested function has its own curFTag; an inner
// loop grows loopStack past the boundary), so its jump is the non-local kind.
var ctlStack = []
var curFTag = 0
function ctlEnter(tag) { ctlStack.push(tag) }
function ctlLeave() { ctlStack.pop() }
function inCtlBody() { return ctlStack.length > 0 && curFTag == ctlStack[ctlStack.length - 1] }

// retStmt is the return emitter every grammar's makeReturn calls: a plain ret, unless
// it leaves a try body, in which case it returns a control signal.
function retStmt(b, v) {
    if (inCtlBody()) {
        b.NewRet(callExt(b, "js_ctl_return", [v]))
    } else {
        b.NewRet(v)
    }
    return deadBlock()
}

// emitBreak / emitContinue emit a break/continue at block b: a branch to the enclosing
// loop, or a control signal when the jump leaves a try body with no loop of its own.
// makeBreak/makeContinue and excDispatch (re-issuing a signal) all go through these.
function emitBreak(b) {
    if (loopStack.length > 0) { b.NewBr(loopStack[loopStack.length - 1].brk); return }
    if (inCtlBody()) { b.NewRet(callExt(b, "js_ctl_break", [])); return }
    // Outside every loop and every try/finally closure: a plain grammar bug in
    // the program; the bare loopStack read used to crash the compiler instead.
    println("compile error: break outside of a loop")
    exit(1)
}
function emitContinue(b) {
    if (loopStack.length > 0) { b.NewBr(loopStack[loopStack.length - 1].cont); return }
    if (inCtlBody()) { b.NewRet(callExt(b, "js_ctl_continue", [])); return }
    println("compile error: continue outside of a loop")
    exit(1)
}

// excDispatch is emitted right after a js_try call: ctl is its result. If ctl is a
// control signal, re-issue the return/break/continue AS IF a return/break/continue
// statement stood here - so if this try is itself inside another try body, the signal
// re-signals (via retStmt/emitBreak/emitContinue) and propagates outward. Otherwise
// fall through. Returns the block to continue in.
function excDispatch(b, ctl) {
    var kind = callExt(b, "js_ctl_kind", [ctl])
    var normB = curF.NewBlock("")
    var retB = curF.NewBlock("")
    var afterRet = curF.NewBlock("")
    b.NewCondBr(b.NewICmp(llvm.enum.IPredEQ, kind, handle(1)), retB, afterRet)
    // retStmt terminates retB and hands back a fresh unreachable block; nothing else
    // flows into it, so terminate it too, or the emitted module has a block with no
    // terminator (harmless at run time but it breaks IR printing / -cfg / -callgraph).
    retStmt(retB, callExt(retB, "js_ctl_value", [ctl])).NewRet(hUndef)
    // break/continue can only arise from valid code if there is somewhere to send them:
    // an enclosing loop, or an enclosing try body to re-signal through. Otherwise (a
    // stray break/continue at a try not in a loop) fall through rather than emit an
    // invalid branch.
    if (loopStack.length > 0 || inCtlBody()) {
        var brkB = curF.NewBlock("")
        var contB = curF.NewBlock("")
        var afterBrk = curF.NewBlock("")
        afterRet.NewCondBr(afterRet.NewICmp(llvm.enum.IPredEQ, kind, handle(2)), brkB, afterBrk)
        emitBreak(brkB)
        afterBrk.NewCondBr(afterBrk.NewICmp(llvm.enum.IPredEQ, kind, handle(3)), contB, normB)
        emitContinue(contB)
    } else {
        afterRet.NewBr(normB)
    }
    return normB
}

// stmtPos wraps a statement thunk with a js_srcpos marker carrying the source
// position of its node (up.pos), so traces, diagrams and steppers know which
// statement executes. Only when the host collects positions (c.tracing, i.e.
// -trace or -cfg): otherwise the emitted IR stays exactly as without markers.
// The grammars apply it with a production level tag on their Statement rule.
function stmtPos(t) {
    if (!c.tracing || up.pos == undefined) return t
    var pos = up.pos
    return function(b) {
        callExt(b, "js_srcpos", [handle(pos)])
        return t(b)
    }
}

function deadBlock() {
    deadCount++
    return curF.NewBlock("dead" + deadCount)
}
function truthy(b, v) {
    return b.NewICmp(llvm.enum.IPredNE, callExt(b, core.truthyExt, [v]), handle(0))
}
// Int results are truncated to 32 bit through the bitwise or external.
function intWrap(b, v) {
    return callExt(b, "js_bor", [v, emitNum(b, 0)])
}

// ----- Expression thunks: function(block) -> {b, v} -----

function makeConst(v) {
    return function(b) {
        if (v === true) return {b: b, v: hTrue}
        if (v === false) return {b: b, v: hFalse}
        if (v === null) return {b: b, v: hNull}
        if (v === undefined) return {b: b, v: hUndef}
        if (typeof v == "string") return {b: b, v: emitStr(b, v)}
        return {b: b, v: emitNum(b, v)}
    }
}
function makeVarRef(name) {
    return function(b) {
        return {b: b, v: callExt(b, core.scopeGetExt, [curScopeV, emitStr(b, name)])}
    }
}
function makeNeg(t) {
    return function(b) {
        var r = t(b)
        return {b: r.b, v: intWrap(r.b, callExt(r.b, "js_neg", [r.v]))}
    }
}
function makeNot(t) {
    return function(b) {
        var r = t(b)
        return {b: r.b, v: callExt(r.b, "js_not", [r.v])}
    }
}

var binExt = {"+": "js_jadd", "-": "js_sub", "*": "js_mul", "/": "js_div", "%": "js_mod",
              "==": "js_seq", "!=": "js_sne",
              "<": "js_lt", ">": "js_gt", "<=": "js_le", ">=": "js_ge",
              "+=": "js_jadd", "-=": "js_sub", "*=": "js_mul", "/=": "js_div", "%=": "js_mod"}
var wraps = {"-": 1, "*": 1, "/": 1, "%": 1, "-=": 1, "*=": 1, "/=": 1, "%=": 1}
function emitBin(b, op, l, r) {
    var v = callExt(b, binExt[op], [l, r])
    if (wraps[op] == 1) v = intWrap(b, v)
    return v
}
function foldBinary(items) {
    if (items.length == 1) return items[0]
    return function(b) {
        var r = items[0](b)
        b = r.b
        var v = r.v
        for (var i = 1; i < items.length; i += 2) {
            var rr = items[i + 1](b)
            b = rr.b
            v = emitBin(b, items[i], v, rr.v)
        }
        return {b: b, v: v}
    }
}
function makeOrAnd(items, isOr) {
    if (items.length == 1) return items[0]
    return function(b) {
        var r = items[0](b)
        b = r.b
        var v = r.v
        for (var i = 1; i < items.length; i++) {
            var rightB = curF.NewBlock("")
            var mergeB = curF.NewBlock("")
            var c = truthy(b, v)
            if (isOr) { b.NewCondBr(c, mergeB, rightB) }
            else      { b.NewCondBr(c, rightB, mergeB) }
            var r2 = items[i](rightB)
            r2.b.NewBr(mergeB)
            var phi = mergeB.NewPhi([llvm.ir.NewIncoming(v, b), llvm.ir.NewIncoming(r2.v, r2.b)])
            b = mergeB
            v = phi
        }
        return {b: b, v: v}
    }
}

function emitArgs(b, args) {
    var argsV = callExt(b, "js_arr_new", [])
    for (var j = 0; j < args.length; j++) {
        var ar = args[j](b)
        b = ar.b
        callExt(b, "js_arr_push", [argsV, ar.v])
    }
    return {b: b, v: argsV}
}

// items = [primaryThunk, {kind:"mcall"|"field"|"index", ...}...].
function foldCallMember(items) {
    if (items.length == 1) return items[0]
    var primary = items[0]
    var suffixes = items.slice(1)
    return function(b) {
        var r = primary(b)
        b = r.b
        var cur = r.v
        for (var i = 0; i < suffixes.length; i++) {
            var s = suffixes[i]
            if (s.kind == "mcall") {
                var ea = emitArgs(b, s.args)
                b = ea.b
                cur = callExt(b, "js_mcall", [cur, emitStr(b, s.name), ea.v])
            } else if (s.kind == "field") {
                cur = callExt(b, "js_get", [cur, emitStr(b, s.name)])
            } else {
                var ir2 = s.idx(b)
                b = ir2.b
                cur = callExt(b, core.indexExt, [cur, ir2.v])
            }
        }
        return {b: b, v: cur}
    }
}

// Target items = [name, {key}|{idx}...].
function makeTargetRef(items) {
    return {name: items[0], path: items.slice(1)}
}

// ----- Statement thunks: function(block) -> block -----

function exprToStmt(t) {
    return function(b) { return t(b).b }
}

function makeSeq(items) {
    return function(b) {
        for (var i = 0; i < items.length; i++) b = items[i](b)
        return b
    }
}
function makeBlockStmt(items) {
    var seq = makeSeq(items)
    return function(b) {
        var saved = curScopeV
        curScopeV = callExt(b, "js_scope_new", [curScopeV])
        b = seq(b)
        curScopeV = saved
        return b
    }
}
function makeIf(items) {
    var cond = items[0]
    var thenT = items[1]
    var elseT = items.length > 2 ? items[2] : null
    return function(b) {
        var r = cond(b)
        var thenB = curF.NewBlock("")
        var mergeB = curF.NewBlock("")
        var elseB = elseT != null ? curF.NewBlock("") : mergeB
        r.b.NewCondBr(truthy(r.b, r.v), thenB, elseB)
        thenT(thenB).NewBr(mergeB)
        if (elseT != null) elseT(elseB).NewBr(mergeB)
        return mergeB
    }
}
function makeWhile(bodyT, condT) {
    return function(b) {
        var headB = curF.NewBlock("")
        var bodyB = curF.NewBlock("")
        var exitB = curF.NewBlock("")
        b.NewBr(headB)
        var r = condT(headB)
        r.b.NewCondBr(truthy(r.b, r.v), bodyB, exitB)
        loopStack.push({cont: headB, brk: exitB})
        bodyT(bodyB).NewBr(headB)
        loopStack.pop()
        return exitB
    }
}

// break/continue: a normal branch to the enclosing loop, unless it leaves a try body
// with no loop of its own inside (loopStack was reset at the closure boundary), in
// which case it returns a control signal for excDispatch to re-issue (see emitBreak).
function makeBreak() {
    return function(b) { emitBreak(b); return deadBlock() }
}
function makeContinue() {
    return function(b) { emitContinue(b); return deadBlock() }
}

// items = [name, blockThunk] or [blockThunk] (no binding): package a catch clause into
// {catchbody, catchname} for the language's makeTry. The compiler twin of interp-core's
// excCatch; every compiler that lowers try/catch consumes it the same way.
function makeCatch(items) {
    if (items.length > 1) { return {catchbody: items[1], catchname: items[0]} }
    return {catchbody: items[0], catchname: undefined}
}

