// Recognition test for the widened TypeScript grammar: a real-world-looking module that
// USES the constructs the grammar now PARSES but does not lower. Each such construct is
// reported as not-implemented, so:
//   - a plain run ABORTS at the first one (this file SHOULD fail by default), and
//   - a -warn-unsupported run warns on each, substitutes a placeholder, and still runs,
//     so main() self-checks and returns 0.
// The genuinely-compiled additions (binary/octal literals, compound bitwise/shift
// assignments, object shorthand and computed keys, decorated/generic classes, and
// `export` in front of a declaration) are asserted for correctness; the not-implemented
// ones are merely exercised (their placeholder results are not asserted).

// --- Module system: every import form is recognized and not implemented. ---
import "./polyfills";
import defaultDep, { helper as h, type Config } from "./util";
import * as path from "path";

// `export` in front of a real declaration is a transparent prefix (compiled genuinely).
export const VERSION: string = "1.0";
export type ID = string | number;
export interface Named { readonly name: string; }

// A namespace body is recognized and skipped.
namespace Geometry {
    export const PI = 3;
}

// A decorator is recognized and warned; the decorated generic class still compiles.
@sealed
class Box<T> {
    private items: T[] = [];
    readonly id: number;

    constructor(id: number) {
        this.id = id;
    }

    add(x: T): void {
        this.items.push(x);
    }

    size(): number {
        return this.items.length;
    }
}

// An async function and a generator function are recognized and their bodies skipped
// (they are never called, only their declarations trigger a warning).
async function loadAll(): Promise<number> {
    const n = await fetchCount();
    return n;
}

function* counter(): Iterator<number> {
    yield 1;
    yield 2;
}

// Genuinely compiled helpers.
function check(cond: boolean, weight: number): number {
    return cond ? 0 : weight;
}

function passthrough(v: number): number {
    return v;
}

function collect(first: number, ...rest: number[]): number {
    return first;                       // rest parameter recognized (bound to first rest arg)
}

function greet(name: string = "world"): string {
    return name;                        // default value recognized (erased)
}

function main(): number {
    let failures: number = 0;

    // --- Genuinely compiled additions: asserted for correctness. ---
    failures += check(0b1010 === 10, 1);         // binary literal
    failures += check(0o17 === 15, 1);           // octal literal

    let bits: number = 0b0011;                   // compound bitwise / shift assignments
    bits |= 0b0100;                              // 0b0111 = 7
    bits <<= 1;                                  // 14
    bits &= 0b1110;                              // 14
    failures += check(bits === 14, 1);

    const x: number = 5;
    const key: string = "dyn";
    const obj = { x, [key]: 99 };                // object shorthand + computed key
    failures += check(obj.x === 5, 1);
    failures += check(obj.dyn === 99, 1);

    const b: Box<number> = new Box<number>(7);   // decorated generic class
    b.add(10);
    b.add(20);
    failures += check(b.size() === 2, 1);
    failures += check(b.id === 7, 1);

    failures += check(VERSION === "1.0", 1);     // `export const` is in scope
    failures += check(collect(1, 2, 3) === 1, 1);
    failures += check(greet("hi") === "hi", 1);
    failures += check(passthrough(4) === 4, 1);

    // --- Recognized-but-not-implemented constructs: exercised, results not asserted. ---
    const cfg: { level: number } = { level: 3 };
    const reach = cfg?.level;                    // optional chaining (?.)
    const chosen = cfg ?? { level: 0 };          // nullish coalescing (??)
    const power = 2 ** 8;                        // exponentiation (**)
    const isArr = cfg instanceof Object;         // instanceof
    const hasLevel = "level" in cfg;             // in
    const nums: number[] = [1, 2, 3];
    const grown = [...nums, 4];                  // array spread
    const folded = collect(...nums);             // call spread
    const pattern = /ab+c/gi;                    // regular-expression literal
    const later = await passthrough(9);          // await (evaluated as identity)
    const twice = async (n: number) => n * 2;    // async arrow (built as a sync closure)
    void reach;                                  // void
    const removable = { tmp: 1 };
    delete removable.tmp;                        // delete

    return failures;
}
