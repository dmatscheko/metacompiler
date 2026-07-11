// Exercises the JavaScript constructs that are ACCEPTED structurally but NOT
// lowered (no runtime model in the subset): try/catch/finally, class
// declarations, destructuring bindings, and for-in.
//
// A default run of either grammar aborts cleanly at the FIRST such construct
// (a "not implemented (file:line); use -warn-unsupported to ignore" message).
// With -warn-unsupported every construct is warned and skipped, the genuinely
// supported code around them runs, and main() returns 0 (a clean exit).

var failures = 0;

function check(cond) {
    if (!cond) { failures = failures + 1; }
}

// try/catch/finally: under -warn the try block runs; catch/finally are dropped.
// (The try body must not throw - throw still terminates in this subset.)
function testTry() {
    var reached = 0;
    try {
        reached = 1;
    } catch (e) {
        reached = 99;
    } finally {
        reached = reached;
    }
    check(reached === 1);
}

// A class declaration is accepted, its body skipped, the name bound to undefined.
class Shape {
    constructor(kind) { this.kind = kind; }
    area() { return 0; }
}
class Circle extends Shape {
    area() { return 3; }
}

// Destructuring bindings: the initializer runs (so its calls are visible), but no
// names are bound - do not read them afterwards.
function testDestructure() {
    var sideEffect = 0;
    var bump = () => { sideEffect = sideEffect + 1; return [1, 2]; };
    var [a, b] = bump();          // array pattern; bump() runs
    var {x, y} = {x: 10, y: 20};  // object pattern
    check(sideEffect === 1);
}

// for-in is accepted; under -warn the iterable is evaluated and the body skipped.
function testForIn() {
    var obj = {p: 1, q: 2};
    var touched = 0;
    for (var k in obj) { touched = touched + 1; }
    check(touched === 0); // body skipped under -warn
}

function main() {
    testTry();
    testDestructure();
    testForIn();
    check(typeof Shape === "undefined"); // class name lowered to undefined
    return failures;
}
