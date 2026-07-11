// Self-checking test for the newly supported JavaScript constructs that are
// GENUINELY implemented by both js-interpreter.abnf and js-to-llvm-ir.abnf:
// arrow functions, template literals, and for-of. It counts failed checks and
// returns that count; exit 0 means every check passed. Uses only supported
// features, so a default (non -warn-unsupported) run of either grammar passes.

var failures = 0;

function check(cond) {
    if (!cond) { failures = failures + 1; }
}

// Arrow functions: expression body, block body, no params, several params, and
// the same closure machinery as function expressions (capture + higher order).
function testArrows() {
    var inc = x => x + 1;
    check(inc(4) === 5);

    var add = (a, b) => a + b;
    check(add(3, 5) === 8);

    var answer = () => 42;
    check(answer() === 42);

    var fact = n => {
        var acc = 1;
        var i;
        for (i = 2; i <= n; i++) { acc = acc * i; }
        return acc;
    };
    check(fact(5) === 120);

    // Closures over the defining scope.
    var makeAdder = x => y => x + y;
    var add10 = makeAdder(10);
    check(add10(7) === 17);

    // Passed as a higher-order callback.
    function applyTwice(f, v) { return f(f(v)); }
    check(applyTwice(x => x * 2, 3) === 12);

    // typeof an arrow is "function".
    check(typeof inc === "function");
}

// Template literals: interpolation, coercion to string, nesting, a literal '$',
// escapes, and the empty template.
function testTemplates() {
    var name = "world";
    check(`hello ${name}` === "hello world");

    var a = 2;
    var b = 3;
    check(`${a}+${b}=${a + b}` === "2+3=5");

    // Numbers and booleans coerce like JS +.
    check(`n=${10}` === "n=10");
    check(`b=${true}` === "b=true");

    // A leading empty chunk and back-to-back interpolations.
    check(`${a}${b}` === "23");

    // A literal dollar that does not start an interpolation.
    check(`cost: $${a}` === "cost: $2");

    // Nested template inside an interpolation.
    check(`[${`x${a}`}]` === "[x2]");

    // Escapes and the empty template.
    check(`tab\tend`.length === 7);
    check(`` === "");

    // Calls inside interpolation.
    var twice = x => x + x;
    check(`v=${twice(4)}` === "v=8");
}

// for-of over arrays and strings, with break / continue and accumulation.
function testForOf() {
    var nums = [10, 20, 30, 40];
    var sum = 0;
    for (var n of nums) { sum = sum + n; }
    check(sum === 100);

    // Bare (already declared) loop variable.
    var product = 1;
    var m;
    for (m of [1, 2, 3, 4]) { product = product * m; }
    check(product === 24);

    // break and continue.
    var collected = 0;
    for (var v of [1, 2, 3, 4, 5, 6]) {
        if (v === 5) { break; }
        if (v % 2 === 0) { continue; }
        collected = collected + v;   // 1 + 3
    }
    check(collected === 4);

    // Iterating the characters of a string.
    var chars = "";
    for (var ch of "abc") { chars = chars + ch + "-"; }
    check(chars === "a-b-c-");

    // Nested for-of.
    var grid = 0;
    for (var r of [1, 2]) {
        for (var c of [10, 20]) { grid = grid + r * c; }
    }
    check(grid === 90); // (10+20)*1 + (10+20)*2

    // for-of over an array of arrows, combining with template literals.
    var ops = [x => x + 1, x => x * 10];
    var trace = "";
    for (var op of ops) { trace = trace + `${op(3)},`; }
    check(trace === "4,30,");
}

function main() {
    testArrows();
    testTemplates();
    testForOf();
    return failures;
}
