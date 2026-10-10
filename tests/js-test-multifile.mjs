// Multi-file JavaScript test: the Vec class and the hypotSq helper live in
// tests/imports/geomlib.js and are pulled in with an ES-style import resolved against the
// -i include root (mec -i tests/imports ...). The imported file is parsed with the same
// grammar; its top-level declarations register in the shared global scope, so the main
// file uses Vec / hypotSq directly (the { Vec, hypotSq } binding list is cosmetic -
// loading the file is what registers the names). Both engines - the js-interpreter and
// the js-to-llvm-ir compiler - run this identically.
//
// It counts failed checks and returns that count from main(); exit 0 means every check
// passed, and it prints "js multifile test passed" on success.

import { Vec, hypotSq } from './geomlib.js';

var failures = 0;

function check(name, cond) {
    if (!cond) {
        println("FAIL " + name);
        failures = failures + 1;
    }
}

function main() {
    var a = new Vec(3, 4);
    var b = new Vec(2, -1);

    check("imported method dot", a.dot(b) === 2);                  // 3*2 + 4*-1 = 2
    check("imported static scale", Vec.scale(a, 2).dot(b) === 4);  // (6,8).(2,-1) = 4
    check("imported method add", a.add(b).dot(a.add(b)) === 34);   // (5,3).(5,3) = 34
    check("imported helper hypotSq", hypotSq(a) === 25);           // 3*3 + 4*4 = 25

    if (failures === 0) {
        println("js multifile test passed");
    }
    return failures;
}
