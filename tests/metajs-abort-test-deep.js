// ABORT PROBE. This program is SUPPOSED to die. See the "SHOULD ABORT" rows in
// .vscode/launch.json and abort_norm() in test.sh. A call nested past MetaJS's
// limit of 10,000 frames is a runtime error, like a type-pinning violation: no
// catch sees it and the run exits 1. A recursion below the limit (the direct
// calls of a top-level function included) runs to its end.
function rec(n) { return rec(n + 1) + 1; }
function sum(n) { if (n == 0) { return 0; } return n + sum(n - 1); }
function main() {
    println(sum(9000));
    var t = 0;
    try { t = sum(5000); } catch (e) { t = -1; }
    println(t);
    println("before the abort");
    try { rec(0); } catch (e) { println("caught " + e); }
    println("never reached");
    return 0;
}
