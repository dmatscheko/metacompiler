// Multi-file TypeScript test: the Vec class, the scale() function and the ORIGIN_TAG const
// live in tests/imports/geomlib.ts and are found via the -i include root
// (mec -i tests/imports ...). The imported file is parsed with the same grammar; its
// top-level declarations register like the main file's, so 'import { Vec, scale } from
// "./geomlib"' makes them usable here. The named-binding list is cosmetic - registration
// is flat and global, not real ES-module scoping. It self-checks: main() returns the fail
// count (0 on success) and prints a line on success. The same file is run by both the
// interpreter (typescript-interpreter.abnf) and the compiler (typescript-to-llvm-ir.abnf).
import { Vec, scale, ORIGIN_TAG } from "./geomlib";

let failures: number = 0;

function check(cond: boolean): void {
    if (!cond) { failures = failures + 1; }
}

function main(): number {
    const a: Vec = new Vec(3, 4);
    const b: Vec = new Vec(2, -1);

    // An imported instance method.
    check(a.dot(b) === 2);          // 3*2 + 4*(-1) = 2
    check(a.dot(a) === 25);         // 9 + 16

    // An imported method that constructs a new Vec inside the imported file.
    const c: Vec = a.add(b);        // (5, 3)
    check(c.x === 5);
    check(c.y === 3);

    // An imported free function.
    const d: Vec = scale(a, 2);     // (6, 8)
    check(d.dot(b) === 4);          // 6*2 + 8*(-1) = 4
    check(d.x === 6);
    check(d.y === 8);

    // An imported const binding.
    check(ORIGIN_TAG === "origin");

    if (failures === 0) {
        println("typescript multifile test passed");
    }
    return failures;
}
