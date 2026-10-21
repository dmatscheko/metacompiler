// Full-syntax test: JavaScript - part 4 of 4, section 57: the MODULE goal.
//
// THE JS RATCHET IS FOUR FILES. Three are split for time (see tests/js-test-full.js);
// this one is split for its GOAL SYMBOL. A '.mjs' file is a Module, and only a Module
// may contain 'import' and 'export' declarations - a '.js' file is a Script, which
// refuses them (IsModule in the js grammars). So the export walk lives here, and the
// Script half of the goal pair stays in SECTION 34 of tests/js-test-full.js.
//
//   tests/js-test-full.js     sections 01-35, the ECMAScript-2022 syntax walk
//   tests/js-test-full.2.js   sections 36-43 and 101-108, engine agreement, generators,
//                             Error, Math, builtin constructors, integrity levels
//   tests/js-test-full.3.js   sections 44-56, 58-71 and 73-100, the call/class/
//                             private-name model, iterators, the arguments object and
//                             the rest of the builtin surface (see its header)
//   tests/js-test-full.4.mjs  sections 57 and 72, the Module goal
//
// The section NUMBERS are global across the four files; ./test.sh --full reports this
// file as the unit js4.
//
// Conventions (shared by every *-test-full.* file):
//   - prologue (before the first SECTION marker): the check helper only
//   - each section: '// ===== SECTION <nn>: <name> =====', top-level,
//     self-contained, no references to other sections
//   - main() calls each section via a line tagged 'SECTION-CALL <nn>'
//     and prints the summary line 'full: <checks> checks, <failures> failures'
//   - main() returns the failure count (exit 0 == full support, verified)
//
// Hand-written for the metacompiler project (Apache-2.0, no copied test-suite code).

var failures = 0
var checks = 0
function check(id, cond) {
    checks = checks + 1
    if (!cond) {
        println("FAIL " + id)
        failures = failures + 1
    }
}

// ===== SECTION 57: export, in a Module =====
// An import here runs the referenced file into the shared global scope, so an export
// is a transparent prefix on its declaration and the specifier forms are no-ops.
export const S57K = 7
export function s57f(x) { return x * 2 }
export class S57C { constructor() { this.v = 3 } }
var s57side = 0
export { s57side as s57sideOut }
export { S57K as s57alias }
export default class S57D { m() { return "d" } }
export function s57mark() { s57side = 42; return s57side }

function s57() {
    check("exp1", S57K === 7)
    check("exp2", s57f(4) === 8)
    check("exp3", new S57C().v === 3)
    check("exp4", s57mark() === 42)
    check("exp5", new S57D().m() === "d")
}

// ===== SECTION 72: a builtin class named only by `new` =====
// lib/js-builtins.js is compiled into a program that names Set, Map, Array or Boolean;
// this file names Set nowhere but in a `new`. Values read off node 24.
function s72() {
    check("nw72a", new Set([1, 2, 2]).size === 2)
}

// ===== END SECTIONS =====

function main() {
    s57() // SECTION-CALL 57
    s72() // SECTION-CALL 72
    println("full: " + checks + " checks, " + failures + " failures")
    return failures
}
