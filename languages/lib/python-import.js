// python-import.js -- the module tables the import statements of both python
// grammars read, and the main module's implicit-entry test, for
// python-interpreter.abnf and python-to-llvm-ir.abnf (include).

// The modules (and submodules) written in Python: lib/python-<top>.py.
var pyLibMods = ["copy", "dataclasses", "collections", "json", "itertools",
                 "functools", "string", "os", "os.path", "random", "importlib", "operator", "enum",
                 "heapq", "bisect", "contextlib", "io", "types", "textwrap", "abc", "typing", "sys", "fractions", "statistics",
                 "datetime", "pprint", "struct", "array", "decimal"]

// The functions the runtime modules math, re and sys offer. Keep in step with
// pyMHas in languages/lib/python-rt.metajs and pyModHas in abnf/jsrtpymath.go.
var pyMathNames = ["sqrt", "cbrt", "exp", "expm1", "log", "log1p", "log2", "log10", "pow",
                   "fabs", "fmod", "copysign", "atan2", "hypot", "sin", "cos", "tan",
                   "asin", "acos", "atan", "sinh", "cosh", "tanh", "degrees", "radians",
                   "floor", "ceil", "trunc", "gcd", "lcm", "factorial",
                   "isnan", "isinf", "isfinite", "isclose",
                   "isqrt", "comb", "perm", "prod", "fsum",
                   "frexp", "ldexp", "modf", "ulp", "nextafter", "dist", "remainder", "sumprod"]
var pyReFnNames = ["compile", "search", "match", "fullmatch", "findall", "finditer",
                   "sub", "subn", "split", "escape", "purge"]
var pySysFnNames = ["exit", "get_int_max_str_digits", "set_int_max_str_digits"]
// The constants beside them.
var pyMathConsts = ["pi", "e", "tau", "inf", "nan"]
var pyReFlagNames = ["NOFLAG", "A", "ASCII", "I", "IGNORECASE", "L", "LOCALE", "M", "MULTILINE",
                     "S", "DOTALL", "U", "UNICODE", "X", "VERBOSE"]

// The names `from M import *` binds: a library module's __all__, read from
// libSrc, its file's text, and for a runtime module (libSrc "") every public
// name it offers - math and sys have no __all__, and re's lists only what it
// offers here. A bare runtime module offers none.
function pyStarNames(mod, libSrc) {
    if (libSrc != "") { return pyLibAll(libSrc, mod) }
    var out = []
    if (mod == "math") { pyAppendAll(out, pyMathNames); pyAppendAll(out, pyMathConsts) }
    if (mod == "re") { pyAppendAll(out, pyReFnNames); pyAppendAll(out, pyReFlagNames) }
    if (mod == "sys") { pyAppendAll(out, pySysFnNames) }
    return out
}
function pyAppendAll(out, names) {
    for (var i = 0; i < names.length; i++) { out.push(names[i]) }
}

// A library module's __all__, read from its source: the class its module
// object instantiates (`_mec_mod_<mod> = type("module", (<class>,), {})()`)
// lists the names first thing in its body as `__all__ = [...]`.
function pyLibAll(src, mod) {
    var out = []
    var key = "_mec_mod_" + mod + " = type(\"module\", ("
    var at = src.indexOf(key)
    if (at < 0) { return out }
    var from = at + key.length
    var cls = src.substring(from, src.indexOf(",", from))
    var ai = src.indexOf("__all__ = [", src.indexOf("class " + cls + ":"))
    var end = src.indexOf("]", ai)
    var q = src.indexOf("\"", ai)
    while (q >= 0 && q < end) {
        var close = src.indexOf("\"", q + 1)
        out.push(src.substring(q + 1, close))
        q = src.indexOf("\"", close + 1)
    }
    return out
}

// The implicit entry: a main module that defines the entry function (main, or
// -main's name) is entered through it after its top level ran - unless that top
// level runs the program itself, the way a real script does. pyTopRunsEntry
// answers that for the source text of one top-level statement: it does when it
// calls the entry by name, or when it is an `if` that tests __name__ (the
// `if __name__ == "__main__":` guard). A def, class or decorated definition never
// does, and strings and comments are skipped, so a quoted name or a remark does
// not count.
var pyTopEntry = false
function pyEntryName() {
    var e = c.mainName
    return (e == undefined || e == "") ? "main" : e
}
function pyTopRunsEntry(src, entry) {
    var n = src.length
    var i = 0
    var first = ""
    var prev = ""
    while (i < n) {
        var ch = src.charAt(i)
        if (ch == "#") {
            while (i < n && src.charAt(i) != "\n") { i++ }
        } else if (ch == "'" || ch == "\"") {
            var q = src.substring(i, i + 3) == ch + ch + ch ? ch + ch + ch : ch
            i += q.length
            while (i < n && src.substring(i, i + q.length) != q) {
                if (src.charAt(i) == "\\") { i++ }
                i++
            }
            i += q.length
            prev = ch
        } else if (pyEntryIdChar(ch)) {
            var j = i
            while (j < n && pyEntryIdChar(src.charAt(j))) { j++ }
            var w = src.substring(i, j)
            if (first == "") {
                first = w
                if (w == "def" || w == "class" || w == "async") { return false }
            }
            if (first == "if" && w == "__name__") { return true }
            if (w == entry && prev != ".") {
                var k = j
                while (k < n && (src.charAt(k) == " " || src.charAt(k) == "\t")) { k++ }
                if (src.charAt(k) == "(") { return true }
            }
            prev = "a"
            i = j
        } else {
            if (ch == "@" && first == "") { return false }
            if (ch != " " && ch != "\t" && ch != "\n" && ch != "\r") { prev = ch }
            i++
        }
    }
    return false
}
function pyEntryIdChar(ch) {
    return (ch >= "a" && ch <= "z") || (ch >= "A" && ch <= "Z") || (ch >= "0" && ch <= "9") || ch == "_" || ch > "\x7f"
}
