// Recognition test for the widened JavaScript grammars (js-interpreter.abnf and
// js-to-llvm-ir.abnf). It is a real-world-looking module that USES the modern syntax
// these grammars now RECOGNIZE but do NOT lower: spread/rest, the modern object-literal
// property forms, default parameters, optional chaining, nullish coalescing, **,
// instanceof/in, delete/void/await/yield, async/generator functions, class expressions,
// new.target, ES export, with/debugger, regex literals and the widened number
// literals (binary/octal/BigInt/numeric-separator). (ES import is now a real feature -
// project-file loading - so it is exercised by tests/js-test-multifile.js instead.)
//
// Each of those constructs is ACCEPTED (it parses) and routed to notImplemented, so:
//   * a DEFAULT run of either grammar aborts cleanly at the first not-lowered construct
//     (a "... not implemented (file:line); use -warn-unsupported to ignore" message);
//   * a run WITH -warn-unsupported warns on each not-lowered construct, skips it, runs
//     the genuinely supported code around it, and main() returns 0.
//
// Only genuinely supported behavior is asserted; the not-lowered constructs are merely
// exercised (so they warn) and their placeholder results are never checked.

export const VERSION = 2;                       // export: declaration
export { VERSION as v };                        // export: named with 'as'

// Default and rest parameters: the bare names still bind by position; the default value
// and the rest gathering are dropped (a + b is genuinely computed from the passed args).
function sum(a, b = 0, ...rest) {
    return a + b;
}

// Spread in array literals and in call arguments.
function spreads(f) {
    var base = [1, 2, 3];
    var more = [0, ...base, 4];                 // spread element
    var r = f(...base);                         // spread argument
    return more.length + r;
}

// The modern object-literal property forms.
function makeConfig(name) {
    var extra = { debug: true };
    return {
        name: name,                             // genuine key: value
        [name + "_id"]: 1,                       // computed key
        ...extra,                                // object spread
        get label() { return name; },            // getter
        set label(v) { name = v; },              // setter
        toString() { return name; }              // shorthand method
    };
}

// Modern operators: optional chaining, nullish coalescing, exponentiation,
// instanceof / in, delete / void, and the extended compound assignments.
function operators(o) {
    var deep = o?.a?.b;                         // optional chaining
    var fallback = o.missing ?? "none";         // nullish coalescing
    var power = 2 ** 10;                        // exponentiation
    var isObj = o instanceof Object;            // instanceof
    var hasKey = "a" in o;                      // in operator
    delete o.a;                                 // delete
    void 0;                                     // void
    var n = 4;
    n <<= 3;                                    // extended compound assignments
    n **= 2;
    n ??= 5;
    return deep;
}

// An async function declaration and a generator declaration (recognized; not bound),
// exercising await and yield inside their bodies.
async function fetchAll(urls) {
    var first = await urls[0];                  // await
    return first;
}
function* counter() {
    yield 1;                                    // yield
    yield 2;
}

// Regex and the widened number literals.
function lexers() {
    var re = /[a-z]+\d*/gi;                     // regex literal
    var bin = 0b1010;                          // binary literal
    var oct = 0o755;                           // octal literal
    var big = 1000n;                           // BigInt literal
    var sep = 1_000_000;                       // numeric separator
    return re;
}

// A class mixing the modern member forms with genuinely supported fields and methods.
class Counter {
    static kind = "counter";
    count = 0;
    static make() { return new Counter(); }     // genuine static method
    inc() { this.count = this.count + 1; return this.count; } // genuine method
    get value() { return this.count; }          // accessor (skipped)
    async load() { return this.count; }         // async method (skipped)
    *steps() { yield this.count; }              // generator method (skipped)
    #secret = 42;                               // private field (skipped)
}

// new.target, a class expression and an async arrow.
function misc() {
    var here = new.target;                      // new.target
    var Anon = class extends Counter { m() { return 1; } }; // class expression
    var af = async (x) => x + 1;                // async arrow
    return 0;
}

// Legacy with / debugger statements.
function legacy(o) {
    with (o) { debugger; }                      // with + debugger
    return 0;
}

var failures = 0;
function check(cond) { if (!cond) { failures = failures + 1; } }

function main() {
    // Genuinely supported behavior around the not-lowered constructs.
    check(sum(3, 4, 5, 6) === 7);               // default/rest dropped; a + b === 7
    var c = Counter.make();                     // genuine class + static + fields
    check(c.inc() === 1);                       // genuine instance method
    check(c.inc() === 2);

    // Exercise the not-lowered constructs (they warn); results are not checked.
    spreads(function(x) { return x; });
    makeConfig("widget");
    operators({ a: { b: 1 }, missing: 0 });
    lexers();
    misc();
    legacy({});

    return failures;
}
