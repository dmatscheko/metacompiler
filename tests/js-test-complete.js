// Self-checking test for the newly GENUINELY implemented JavaScript construct:
// destructuring declarations (var [a, b] = xs / var {x, y: alias} = o, including
// nested and mixed array/object patterns). Both js-interpreter.abnf and
// js-to-llvm-ir.abnf lower the same reified pattern tree, so the interpreter and
// the compiler (under both the goja and the -frozen engine) agree byte for byte.
//
// It counts failed checks and returns that count; exit 0 means every check passed.
// Only genuinely supported features are used, so a default (non -warn-unsupported)
// run of either grammar passes.

var failures = 0;

function check(cond) {
    if (!cond) { failures = failures + 1; }
}

// Plain array patterns: exact, short (extra elements dropped), and long (missing
// elements bind to undefined).
function testArrayPattern() {
    var [a, b, c] = [1, 2, 3];
    check(a === 1);
    check(b === 2);
    check(c === 3);

    var [x, y] = [10, 20, 30];   // extra element ignored
    check(x === 10);
    check(y === 20);

    var [p, q, r] = [7, 8];      // r has no element
    check(p === 7);
    check(q === 8);
    check(typeof r === "undefined");

    // The initializer can be any expression, e.g. a variable.
    var pair = [8, 9];
    var [m, n] = pair;
    check(m === 8);
    check(n === 9);
}

// Object patterns: shorthand and aliased fields.
function testObjectPattern() {
    var {name, age} = {name: "Bob", age: 30};
    check(name === "Bob");
    check(age === 30);

    var {name: who, age: years} = {name: "Ann", age: 25};
    check(who === "Ann");
    check(years === 25);

    // A missing property binds undefined.
    var {present, missing} = {present: 1};
    check(present === 1);
    check(typeof missing === "undefined");

    // Types are preserved through the binding.
    var {s, k, f} = {s: "str", k: 42, f: function() { return 5; }};
    check(typeof s === "string");
    check(typeof k === "number");
    check(typeof f === "function");
    check(f() === 5);
}

// Nested patterns: array-in-array, object-in-array, array-in-object,
// object-in-object, arbitrarily deep.
function testNested() {
    var [[a1, a2], [b1, b2]] = [[1, 2], [3, 4]];
    check(a1 === 1);
    check(a2 === 2);
    check(b1 === 3);
    check(b2 === 4);

    var [first, {val}] = [7, {val: 99}];
    check(first === 7);
    check(val === 99);

    var {coords: [cx, cy]} = {coords: [5, 6]};
    check(cx === 5);
    check(cy === 6);

    var {outer: {inner}} = {outer: {inner: 42}};
    check(inner === 42);

    var {items: [i0, i1], count} = {items: [100, 200], count: 2};
    check(i0 === 100);
    check(i1 === 200);
    check(count === 2);

    // Aliased inside a nested object.
    var {box: {value: v}} = {box: {value: "deep"}};
    check(v === "deep");
}

// The initializer is evaluated exactly once, even for a destructuring binding.
var callCount = 0;
function makePair() {
    callCount = callCount + 1;
    return [callCount, callCount * 10];
}
function testEvalOnce() {
    callCount = 0;
    var [lo, hi] = makePair();
    check(callCount === 1);   // makePair ran once
    check(lo === 1);
    check(hi === 10);

    var [lo2, hi2] = makePair();
    check(callCount === 2);
    check(lo2 === 2);
    check(hi2 === 20);
}

// Destructuring cooperates with the rest of the language: loops, closures,
// arrays of pairs.
function testInteraction() {
    var points = [[1, 2], [3, 4], [5, 6]];
    var sum = 0;
    for (var pt of points) {
        var [px, py] = pt;
        sum = sum + px + py;
    }
    check(sum === 21); // (1+2)+(3+4)+(5+6)

    // Destructure the result of a returning function that builds an object.
    function record(id) { return {id: id, sq: id * id}; }
    var {id, sq} = record(6);
    check(id === 6);
    check(sq === 36);

    // A closure capturing a destructured binding.
    var {base} = {base: 100};
    var add = function(d) { return base + d; };
    check(add(23) === 123);
}

function main() {
    testArrayPattern();
    testObjectPattern();
    testNested();
    testEvalOnce();
    testInteraction();
    return failures;
}
