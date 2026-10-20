// python-import.js -- the module tables the import statements of both python
// grammars read, for python-interpreter.abnf and python-to-llvm-ir.abnf
// (include).

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
