// Exercises the one JavaScript construct that is still ACCEPTED structurally but NOT
// lowered (no runtime model in the subset): try/catch/finally. (class declarations and
// for-in are now genuinely implemented; destructuring bindings genuinely bind.)
//
// A default run of either grammar aborts cleanly at the FIRST not-lowered construct
// (a "not implemented (file:line); use -warn-unsupported to ignore" message) - here the
// try/catch in testTry(). With -warn-unsupported the try block runs, the genuinely
// supported code around it runs, and main() returns 0 (a clean exit).

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

// Destructuring bindings genuinely bind here (real JS), so the initializer runs and
// its side effect is visible. (This test keeps them to still exercise a construct
// around the notImplemented try/catch.)
function testDestructure() {
    var sideEffect = 0;
    var bump = () => { sideEffect = sideEffect + 1; return [1, 2]; };
    var [a, b] = bump();          // array pattern; bump() runs
    var {x, y} = {x: 10, y: 20};  // object pattern
    check(sideEffect === 1);
    check(a === 1 && b === 2);
    check(x === 10 && y === 20);
}

function main() {
    testTry();
    testDestructure();
    return failures;
}
