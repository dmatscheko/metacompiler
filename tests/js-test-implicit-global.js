// Self-checking test for the implicit global of non-strict JavaScript: a plain
// `=` assignment to a name that was never declared creates a global variable,
// visible everywhere. The interpreter (setVar) and the LLVM-IR compiler
// (js_scope_set_or_create) must agree, and main() returns 0 when they do.

var failures = 0;

function check(cond) {
    if (!cond) { failures = failures + 1; }
}

// Create a global from inside a function (no var/let/const anywhere).
function makeGlobal() {
    counter = 100;        // implicit global
    return counter;
}

// A second function sees and mutates the same global.
function bumpGlobal() {
    counter = counter + 1;
    return counter;
}

// Assigning to an OUTER declared variable must find it in the chain and NOT
// create a global that shadows it.
function testChainWins() {
    var local = 1;
    function inner() { local = 42; } // updates the outer local, not a global
    inner();
    check(local === 42);
    check(typeof local === "number");
}

// A plain `=` at top level to an undeclared name is also an implicit global.
topLevelGlobal = "hello";

function main() {
    check(makeGlobal() === 100);
    check(counter === 100);       // visible in main
    check(bumpGlobal() === 101);
    check(counter === 101);       // the mutation is shared
    check(topLevelGlobal === "hello");

    // A global can change type freely (this is dynamic JS, not MetaJS).
    counter = "now a string";
    check(counter === "now a string");

    testChainWins();
    return failures;
}
