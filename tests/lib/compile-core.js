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
    scopeGetExt: "js_scope_get" // The external for reading a name (Kotlin: "js_kget", this-aware).
}

function takeAll() {
    var items = []
    var v
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
    var g = strGlobals[s]
    if (g == undefined) {
        strCount++
        g = {}
        g.glob = m.NewGlobalDef("str." + strCount, llvm.constant.NewCharArrayFromString(s))
        g.len = handle(s.length)
        g.ptr = llvm.constant.NewGetElementPtr(g.glob.ContentType, g.glob, [handle(0), handle(0)])
        strGlobals[s] = g
    }
    return callExt(b, "js_str_mem", [g.ptr, g.len])
}
function emitNum(b, n) {
    return callExt(b, "js_num_i", [handle(n)])
}

var curF = null
var curScopeV = null
var loopStack = []
var funcCount = 0
var deadCount = 0

function deadBlock() {
    deadCount++
    return curF.NewBlock("dead" + deadCount)
}
function truthy(b, v) {
    return b.NewICmp(llvm.enum.IPredNE, callExt(b, "js_truthy", [v]), handle(0))
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

function makeBreak() {
    return function(b) {
        b.NewBr(loopStack[loopStack.length - 1].brk)
        return deadBlock()
    }
}
function makeContinue() {
    return function(b) {
        b.NewBr(loopStack[loopStack.length - 1].cont)
        return deadBlock()
    }
}

