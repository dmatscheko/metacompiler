/* Multi-file MetaJS test: makeVec / vecDot / vecScaleX live in tests/imports/mjslib.js and
   are found via the -i include root (mec -i tests/imports ...). The imported file is parsed
   with the same grammar; its functions register in the shared global scope before main()
   runs. main() returns the failed-check count, so the run exits 0 exactly when the imported
   functions resolve and compute correctly. */

import "./mjslib.js";

var fails = 0;

function check(cond) { if (!cond) { fails = fails + 1; } }

function main() {
    var a = makeVec(3, 4);
    var b = makeVec(2, -1);
    check(vecDot(a, b) == 2);
    check(vecScaleX(a, 2) == 6);
    if (fails == 0) { println("metajs multifile test passed"); }
    return fails;
}
