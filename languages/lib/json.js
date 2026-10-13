// json.js -- JSON.stringify and JSON.parse for JavaScript and TypeScript.
//
// Written once for the two interpreter halves (include("lib/json.js")) and for
// layer 2 (languages/lib/js-rt.metajs imports it), in the restricted subset both
// accept - see the head of regex.js. abnf/jsrtjson.go is the Go twin the compiler
// halves run under llvm.Run; keep the two in step.
//
// The value model differs between the engines, so everything that touches a value
// goes through a HOOK the includer defines:
//   jsonHKeys(o)            the enumerable own string keys of an object, in order
//   jsonHGet(o, k)          a property read, getters included
//   jsonHPut(o, k, v)       create or overwrite an enumerable own data property
//   jsonHDel(o, k)          delete an own property
//   jsonHNew()              a fresh plain object
//   jsonHCall(f, self, a)   call a function value with a receiver and arguments
//   jsonHToJSON(v, k)       v.toJSON(k), or jsonMiss when v has no callable toJSON
//   jsonHIsFn(v)            typeof v is "function" (a class included)
//   jsonHIsBig(v)           v is a BigInt
//   jsonHIsObj(v)           v is an object that serializes by its keys (no Array,
//                           BigInt, function or symbol; an arguments object is one)
//   jsonHStr(v)             ToString(v); a finite number gives Number::toString
//   jsonHParseNum(s)        the double a JSON number lexeme denotes
//   jsonHRaise(text)        throw the Error the text names ("TypeError: ...")
//
// The error MESSAGES of JSON.parse are node 24's (V8's json-parser): a rule the
// parser names (jsonBadAt) carries the position, line and column, and any other
// unexpected character quotes the text around it (jsonBad).

var jsonMiss = {"__jsonmiss": true}

// ----- JSON.stringify -----

function jsonStringify(value, replacer, space) {
    var st = {}
    st["gap"] = jsonGap(space)
    st["indent"] = ""
    st["stack"] = []
    st["fn"] = undefined
    st["allow"] = undefined
    if (jsonHIsFn(replacer)) { st["fn"] = replacer }
    else if (Array.isArray(replacer)) { st["allow"] = jsonAllowList(replacer) }
    var wrapper = jsonHNew()
    jsonHPut(wrapper, "", value)
    return jsonSer(st, wrapper, "", value)
}

// The indent unit: a number is that many spaces (at most 10), a string its first ten
// characters, anything else none.
function jsonGap(space) {
    var g = ""
    if (typeof space == "number") {
        var i = 0
        while (i < 10 && i + 1 <= space) {
            g = g + " "
            i = i + 1
        }
        return g
    }
    if (typeof space == "string") {
        if (space.length > 10) { return space.substring(0, 10) }
        return space
    }
    return g
}

// An array replacer: the property names to keep, from its strings and numbers, once
// each, in order.
function jsonAllowList(r) {
    var out = []
    var i = 0
    while (i < r.length) {
        var e = anytype // An element of any type; only strings and numbers count.
        e = r[i]
        if (typeof e == "string" || typeof e == "number") {
            var k = jsonHStr(e)
            if (out.indexOf(k) < 0) { out.push(k) }
        }
        i = i + 1
    }
    return out
}

// SerializeJSONProperty: the text of holder[key], whose value is v0, or undefined
// when the property is skipped (undefined, a function).
function jsonSer(st, holder, key, v0) {
    var v = anytype // The value after toJSON and the replacer, of any type.
    v = v0
    if (Array.isArray(v) || jsonHIsObj(v)) {
        var tj = anytype
        tj = jsonHToJSON(v, key)
        if (tj !== jsonMiss) { v = tj }
    }
    if (st["fn"] !== undefined) { v = jsonHCall(st["fn"], holder, [key, v]) }
    if (v === null) { return "null" }
    if (v === true) { return "true" }
    if (v === false) { return "false" }
    if (typeof v == "string") { return jsonQuote(v) }
    if (typeof v == "number") {
        if (v !== v || v - v !== 0) { return "null" }
        return jsonHStr(v)
    }
    if (jsonHIsBig(v)) { jsonHRaise("TypeError: Do not know how to serialize a BigInt") }
    if (jsonHIsObj(v)) { return jsonObj(st, v) }
    if (Array.isArray(v)) { return jsonArr(st, v) }
    return undefined
}

// QuoteJSONString: the escapes JSON requires, \u00XX for the other control
// characters, and a LONE surrogate escaped rather than copied (well-formed stringify).
function jsonQuote(s) {
    var q = "\""
    var bs = "\\"
    var out = q
    var i = 0
    while (i < s.length) {
        var c = s.charCodeAt(i)
        if (c == 34) { out = out + bs + q }
        else if (c == 92) { out = out + bs + bs }
        else if (c == 8) { out = out + bs + "b" }
        else if (c == 12) { out = out + bs + "f" }
        else if (c == 10) { out = out + bs + "n" }
        else if (c == 13) { out = out + bs + "r" }
        else if (c == 9) { out = out + bs + "t" }
        else if (c < 32) { out = out + jsonU(c) }
        else if (c >= 55296 && c <= 57343) {
            var d = 0
            if (i + 1 < s.length) { d = s.charCodeAt(i + 1) }
            if (c <= 56319 && d >= 56320 && d <= 57343) {
                out = out + s.substring(i, i + 2)
                i = i + 1
            } else {
                out = out + jsonU(c)
            }
        }
        else { out = out + s.charAt(i) }
        i = i + 1
    }
    return out + q
}
function jsonU(c) {
    var hex = "0123456789abcdef"
    return "\\" + "u" + hex.charAt((c >> 12) & 15) + hex.charAt((c >> 8) & 15) + hex.charAt((c >> 4) & 15) + hex.charAt(c & 15)
}

// A cycle is node's TypeError (its message continues with a description of the cycle,
// which no engine here builds).
function jsonEnter(st, v) {
    var stack = st["stack"]
    var i = 0
    while (i < stack.length) {
        if (stack[i] === v) { jsonHRaise("TypeError: Converting circular structure to JSON") }
        i = i + 1
    }
    stack.push(v)
    return undefined
}

function jsonArr(st, a) {
    jsonEnter(st, a)
    var stepback = st["indent"]
    st["indent"] = stepback + st["gap"]
    var parts = []
    var i = 0
    while (i < a.length) {
        var s = anytype // A text, or undefined for a skipped element.
        s = jsonSer(st, a, jsonHStr(i), a[i])
        if (s === undefined) { s = "null" }
        parts.push(s)
        i = i + 1
    }
    var out = jsonWrap(st, parts, "[", "]", stepback)
    st["indent"] = stepback
    st["stack"].pop()
    return out
}

function jsonObj(st, o) {
    jsonEnter(st, o)
    var stepback = st["indent"]
    st["indent"] = stepback + st["gap"]
    var keys = anytype // The allow list or the object's own keys.
    keys = st["allow"]
    if (keys === undefined) { keys = jsonHKeys(o) }
    var colon = ":"
    if (st["gap"] != "") { colon = ": " }
    var parts = []
    var i = 0
    while (i < keys.length) {
        var s = anytype // A text, or undefined for a skipped member.
        s = jsonSer(st, o, keys[i], jsonHGet(o, keys[i]))
        if (s !== undefined) { parts.push(jsonQuote(keys[i]) + colon + s) }
        i = i + 1
    }
    var out = jsonWrap(st, parts, "{", "}", stepback)
    st["indent"] = stepback
    st["stack"].pop()
    return out
}

function jsonWrap(st, parts, open, close, stepback) {
    if (parts.length == 0) { return open + close }
    var gap = st["gap"]
    var sep = ","
    if (gap != "") { sep = ",\n" + st["indent"] }
    var out = open
    if (gap != "") { out = out + "\n" + st["indent"] }
    var i = 0
    while (i < parts.length) {
        if (i > 0) { out = out + sep }
        out = out + parts[i]
        i = i + 1
    }
    if (gap != "") { out = out + "\n" + stepback }
    return out + close
}

// ----- JSON.parse -----

function jsonParse(text, reviver) {
    var p = {}
    p["s"] = jsonHStr(text)
    p["i"] = 0
    var v = anytype // The parsed value, of any type.
    v = jsonValue(p)
    jsonWs(p)
    if (p["i"] < p["s"].length) { jsonBadAt(p, "Unexpected non-whitespace character after JSON") }
    if (jsonHIsFn(reviver)) {
        var root = jsonHNew()
        jsonHPut(root, "", v)
        return jsonRevive(root, "", v, reviver)
    }
    return v
}

// An unexpected character at the cursor, or the end of the text. The text is quoted
// whole up to 20 characters, else the 10 on either side of the cursor; four words
// that ToString makes of a non-JSON value are named alone.
function jsonBad(p) {
    var s = p["s"]
    var i = p["i"]
    if (i >= s.length) { jsonHRaise("SyntaxError: Unexpected end of JSON input") }
    if (s == "undefined" || s == "NaN" || s == "Infinity" || s == "[object Object]") {
        jsonHRaise("SyntaxError: \"" + s + "\" is not valid JSON")
    }
    var ctx = "\"" + s + "\""
    if (s.length > 20) {
        if (i < 10) { ctx = "\"" + s.substring(0, i + 10) + "\"..." }
        else if (i < s.length - 10) { ctx = "...\"" + s.substring(i - 10, i + 10) + "\"..." }
        else { ctx = "...\"" + s.substring(i - 10, s.length) + "\"" }
    }
    jsonHRaise("SyntaxError: Unexpected token '" + s.charAt(i) + "', " + ctx + " is not valid JSON")
    return undefined
}

// A malformed text breaking the rule `what` at the cursor, the end of the text
// included. The line counts CR, LF and CRLF as one break each.
function jsonBadAt(p, what) {
    var s = p["s"]
    var i = p["i"]
    var line = 1
    var start = 0
    var k = 0
    while (k < i) {
        var c = s.charCodeAt(k)
        if (c == 13 && k + 1 < i && s.charCodeAt(k + 1) == 10) { k = k + 1 }
        if (c == 10 || c == 13) {
            line = line + 1
            start = k + 1
        }
        k = k + 1
    }
    jsonHRaise("SyntaxError: " + what + " at position " + jsonHStr(i) + " (line " + jsonHStr(line) +
               " column " + jsonHStr(i - start + 1) + ")")
    return undefined
}

function jsonWs(p) {
    var s = p["s"]
    var i = p["i"]
    while (i < s.length) {
        var c = s.charCodeAt(i)
        if (c != 32 && c != 9 && c != 10 && c != 13) { break }
        i = i + 1
    }
    p["i"] = i
    return undefined
}

// The literal word w must follow at the cursor.
function jsonWord(p, w) {
    var s = p["s"]
    var i = p["i"]
    var j = 0
    while (j < w.length) {
        if (i + j >= s.length || s.charCodeAt(i + j) != w.charCodeAt(j)) {
            p["i"] = i + j
            jsonBad(p)
        }
        j = j + 1
    }
    p["i"] = i + w.length
    return undefined
}

function jsonValue(p) {
    jsonWs(p)
    var s = p["s"]
    var i = p["i"]
    if (i >= s.length) { jsonBad(p) }
    var c = s.charCodeAt(i)
    if (c == 123) { return jsonParseObj(p) }
    if (c == 91) { return jsonParseArr(p) }
    if (c == 34) { return jsonParseStr(p) }
    if (c == 116) { jsonWord(p, "true"); return true }
    if (c == 102) { jsonWord(p, "false"); return false }
    if (c == 110) { jsonWord(p, "null"); return null }
    if (c == 45 || (c >= 48 && c <= 57)) { return jsonParseNum(p) }
    jsonBad(p)
    return undefined
}

function jsonIsDigit(p) {
    return p["i"] < p["s"].length && p["s"].charCodeAt(p["i"]) >= 48 && p["s"].charCodeAt(p["i"]) <= 57
}

// One or more digits, or the rule `what` is broken.
function jsonDigits(p, what) {
    if (!jsonIsDigit(p)) { jsonBadAt(p, what) }
    while (jsonIsDigit(p)) { p["i"] = p["i"] + 1 }
    return undefined
}

// '-'? ('0' | [1-9][0-9]*) ('.' [0-9]+)? ([eE] [+-]? [0-9]+)?
function jsonParseNum(p) {
    var s = p["s"]
    var start = p["i"]
    if (s.charCodeAt(p["i"]) == 45) { p["i"] = p["i"] + 1 }
    if (p["i"] < s.length && s.charCodeAt(p["i"]) == 48) {
        p["i"] = p["i"] + 1
        if (jsonIsDigit(p)) { jsonBadAt(p, "Unexpected number in JSON") }
    } else { jsonDigits(p, "No number after minus sign in JSON") }
    if (p["i"] < s.length && s.charCodeAt(p["i"]) == 46) {
        p["i"] = p["i"] + 1
        jsonDigits(p, "Unterminated fractional number in JSON")
    }
    if (p["i"] < s.length && (s.charCodeAt(p["i"]) == 101 || s.charCodeAt(p["i"]) == 69)) {
        p["i"] = p["i"] + 1
        if (p["i"] < s.length && (s.charCodeAt(p["i"]) == 43 || s.charCodeAt(p["i"]) == 45)) { p["i"] = p["i"] + 1 }
        jsonDigits(p, "Exponent part is missing a number in JSON")
    }
    return jsonHParseNum(s.substring(start, p["i"]))
}

function jsonParseStr(p) {
    var s = p["s"]
    var i = p["i"] + 1
    var out = ""
    var run = i
    while (true) {
        if (i >= s.length) {
            p["i"] = i
            jsonBadAt(p, "Unterminated string in JSON")
        }
        var c = s.charCodeAt(i)
        if (c == 34) { break }
        if (c < 32) {
            p["i"] = i
            jsonBadAt(p, "Bad control character in string literal in JSON")
        }
        if (c == 92) {
            out = out + s.substring(run, i)
            if (i + 1 >= s.length) {
                p["i"] = i + 1
                jsonBad(p)
            }
            var e = s.charCodeAt(i + 1)
            i = i + 2
            if (e == 34) { out = out + "\"" }
            else if (e == 92) { out = out + "\\" }
            else if (e == 47) { out = out + "/" }
            else if (e == 98) { out = out + String.fromCharCode(8) }
            else if (e == 102) { out = out + String.fromCharCode(12) }
            else if (e == 110) { out = out + String.fromCharCode(10) }
            else if (e == 114) { out = out + String.fromCharCode(13) }
            else if (e == 116) { out = out + String.fromCharCode(9) }
            else if (e == 117) {
                var cu = 0
                var j = 0
                while (j < 4) {
                    var h = -1
                    if (i < s.length) { h = jsonHexVal(s.charCodeAt(i)) }
                    if (h < 0) {
                        p["i"] = i
                        jsonBadAt(p, "Bad Unicode escape in JSON")
                    }
                    cu = cu * 16 + h
                    i = i + 1
                    j = j + 1
                }
                out = out + String.fromCharCode(cu)
            } else {
                p["i"] = i - 1
                jsonBadAt(p, "Bad escaped character in JSON")
            }
            run = i
            continue
        }
        i = i + 1
    }
    p["i"] = i + 1
    return out + s.substring(run, i)
}
function jsonHexVal(c) {
    if (c >= 48 && c <= 57) { return c - 48 }
    if (c >= 97 && c <= 102) { return c - 87 }
    if (c >= 65 && c <= 70) { return c - 55 }
    return -1
}

function jsonParseArr(p) {
    p["i"] = p["i"] + 1
    var out = []
    jsonWs(p)
    if (p["i"] < p["s"].length && p["s"].charCodeAt(p["i"]) == 93) {
        p["i"] = p["i"] + 1
        return out
    }
    while (true) {
        out.push(jsonValue(p))
        jsonWs(p)
        var c = -1
        if (p["i"] < p["s"].length) { c = p["s"].charCodeAt(p["i"]) }
        if (c != 93 && c != 44) { jsonBadAt(p, "Expected ',' or ']' after array element in JSON") }
        p["i"] = p["i"] + 1
        if (c == 93) { return out }
    }
    return out
}

function jsonParseObj(p) {
    p["i"] = p["i"] + 1
    var o = jsonHNew()
    jsonWs(p)
    if (p["i"] < p["s"].length && p["s"].charCodeAt(p["i"]) == 125) {
        p["i"] = p["i"] + 1
        return o
    }
    var keyRule = "Expected property name or '}' in JSON"
    while (true) {
        jsonWs(p)
        if (p["i"] >= p["s"].length || p["s"].charCodeAt(p["i"]) != 34) { jsonBadAt(p, keyRule) }
        keyRule = "Expected double-quoted property name in JSON"
        var k = jsonParseStr(p)
        jsonWs(p)
        if (p["i"] >= p["s"].length || p["s"].charCodeAt(p["i"]) != 58) { jsonBadAt(p, "Expected ':' after property name in JSON") }
        p["i"] = p["i"] + 1
        jsonHPut(o, k, jsonValue(p))
        jsonWs(p)
        var c = -1
        if (p["i"] < p["s"].length) { c = p["s"].charCodeAt(p["i"]) }
        if (c != 125 && c != 44) { jsonBadAt(p, "Expected ',' or '}' after property value in JSON") }
        p["i"] = p["i"] + 1
        if (c == 125) { return o }
    }
    return o
}

// InternalizeJSONProperty: the reviver sees every value bottom-up, with the holder as
// its receiver; answering undefined deletes the property.
function jsonRevive(holder, key, val, reviver) {
    if (Array.isArray(val)) {
        var i = 0
        while (i < val.length) {
            var nv = anytype // The reviver's answer for the element, of any type.
            nv = jsonRevive(val, jsonHStr(i), val[i], reviver)
            val[i] = nv
            i = i + 1
        }
    } else if (jsonHIsObj(val)) {
        var ks = jsonHKeys(val)
        var j = 0
        while (j < ks.length) {
            var mv = anytype // The reviver's answer for the member, of any type.
            mv = jsonRevive(val, ks[j], jsonHGet(val, ks[j]), reviver)
            if (mv === undefined) { jsonHDel(val, ks[j]) }
            else { jsonHPut(val, ks[j], mv) }
            j = j + 1
        }
    }
    return jsonHCall(reviver, holder, [key, val])
}
