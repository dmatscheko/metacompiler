// Self-checking test for the TypeScript interpreter and LLVM-IR compiler.
// It counts failed checks and returns that count; exit 0 means every check passed.
//
// This is idiomatic TypeScript: it carries a full static type layer - annotations on
// variables, parameters and return values, an interface, generic functions, an enum,
// 'as' casts, non-null assertions and optional markers - all of which are ERASED at run
// time. What executes is plain JavaScript, so the two engines (goja and -frozen) and the
// interpreter and compiler all agree on every result.

let failures: number = 0;

function check(cond: boolean): void {
    if (!cond) { failures = failures + 1; }
}

// --- Type-only declarations: an interface and two type aliases. They compile to
// nothing but document the shapes used below. ---
interface Point {
    x: number;
    y: number;
    label?: string;          // optional member
}

interface Shape {
    area(): number;          // method signature
}

type Direction = "north" | "south" | "east" | "west";   // string-literal union
type NumberMap = { [key: string]: number };             // index signature

// An enum of bare names becomes integer constants; auto-increment from 0.
enum Color {
    Red,
    Green,
    Blue,
}

// An enum with explicit values and a continued auto-increment.
enum Status {
    Ok = 200,
    NotFound = 404,
    Teapot = 418,
}

enum Mix {
    A = 10,
    B,          // 11
    C,          // 12
}

// Arithmetic and precedence, with typed locals.
function testArithmetic(): void {
    const a: number = 1 + 2 * 3;
    check(a === 7);
    check((1 + 2) * 3 === 9);
    check(10 - 4 - 3 === 3);
    check(7 % 3 === 1);
    check(7 / 2 === 3.5);
    check(-5 + 8 === 3);
    check((6 & 3) === 2);
    check((6 | 1) === 7);
    check((1 << 4) === 16);
    check((~0) === -1);
    check(0x1A === 26);
}

// Comparisons and boolean logic.
function testLogic(): void {
    check(2 < 3);
    check(3 <= 3);
    check(5 > 4);
    check("abc" < "abd");
    check(1 != 2);
    check(1 !== (1 as number) - 0 + 1);   // 'as' cast is identity
    check((0 || 5) === 5);
    check((7 || 9) === 7);
    check((3 && 4) === 4);
    check((0 && 4) === 0);
    check(!!"x" === true);
    check(!0 === true);
    check((5 > 3 ? "a" : "b") === "a");
}

// The enum values read back through member access.
function testEnum(): void {
    check(Color.Red === 0);
    check(Color.Green === 1);
    check(Color.Blue === 2);
    check(Status.Ok === 200);
    check(Status.NotFound === 404);
    check(Status.Teapot === 418);
    check(Mix.A === 10);
    check(Mix.B === 11);
    check(Mix.C === 12);
    // An enum value flows through a typed variable like any number.
    const c: Color = Color.Blue;
    check(c + 1 === 3);
}

// A typed function signature over an interface parameter.
function distanceSquared(p: Point): number {
    return p.x * p.x + p.y * p.y;
}

// A class-free "Shape": a factory returning an object that satisfies the interface.
function makeSquare(side: number): Shape {
    return {
        area: function (): number { return side * side; },
    };
}

function testInterfaces(): void {
    const p: Point = { x: 3, y: 4 };
    check(distanceSquared(p) === 25);
    const q: Point = { x: 3, y: 4, label: "origin-ish" };
    check(q.label === "origin-ish");
    const sq: Shape = makeSquare(5);
    check(sq.area() === 25);
}

// A generic function; the <T> and the annotations are erased.
function identity<T>(x: T): T {
    return x;
}
function firstOf<T>(xs: T[]): T {
    return xs[0];
}

function testGenerics(): void {
    check(identity(42) === 42);
    check(identity("hi") === "hi");
    const xs: number[] = [7, 8, 9];
    check(firstOf(xs) === 7);
    // Non-null assertion is erased (identity here).
    const head: number = xs[0]!;
    check(head === 7);
}

// Strings.
function testStrings(): void {
    const s: string = "Hello";
    check(s.length === 5);
    check(s.charAt(1) === "e");
    check(s.charCodeAt(0) === 72);
    check(s.indexOf("l") === 2);
    check(s.slice(1, 3) === "el");
    check(s.toUpperCase() === "HELLO");
    check(("a,b,c".split(",")).length === 3);
    check("x" + 1 + "y" === "x1y");
    check(("  hi  ".trim()) === "hi");
    // Template literal.
    const name: string = "TS";
    check(`hi ${name}!` === "hi TS!");
}

// Arrays (typed as number[] and via the generic Array<T>).
function testArrays(): void {
    const a: Array<number> = [1, 2, 3];
    check(a.length === 3);
    check(a[0] === 1);
    a.push(4);
    check(a.length === 4);
    check(a.pop() === 4);
    check(a.indexOf(2) === 1);
    check((a.join("-")) === "1-2-3");
    a[1] = 20;
    check(a[1] === 20);
    const b: number[] = a.concat([9]);
    check(b.length === 4);
    // for-of over an array.
    let total: number = 0;
    for (const n of a) { total = total + n; }
    check(total === 1 + 20 + 3);
}

// Maps/dicts via an index-signature object type.
function testMaps(): void {
    const counts: NumberMap = {};
    const words: string[] = ["a", "b", "a", "c", "a", "b"];
    for (const w of words) {
        if (counts[w] === undefined) { counts[w] = 0; }
        counts[w] = counts[w] + 1;
    }
    check(counts["a"] === 3);
    check(counts["b"] === 2);
    check(counts["c"] === 1);
    // Object literal used as a record.
    const rec: { name: string; value: number } = { name: "x", value: 5 };
    check(rec.name === "x");
    check(rec["value"] === 5);
    rec.value = 6;
    check(rec.value === 6);
}

// Control flow: if/else chains, while, do-while, for, break, continue.
function testControlFlow(): void {
    let sum: number = 0;
    for (let i: number = 1; i <= 5; i++) { sum = sum + i; }
    check(sum === 15);

    let w: number = 0;
    let k: number = 0;
    while (k < 4) { w = w + k; k++; }
    check(w === 6);

    let d: number = 0;
    let j: number = 0;
    do { d = d + 1; j++; } while (j < 3);
    check(d === 3);

    let col: number = 0;
    for (let i: number = 0; i < 10; i++) {
        if (i === 5) { break; }
        if (i % 2 === 0) { continue; }
        col = col + i;
    }
    check(col === 4);   // 1 + 3

    let grade: string = "?";
    const score: number = 75;
    if (score >= 90) { grade = "A"; }
    else if (score >= 70) { grade = "B"; }
    else { grade = "C"; }
    check(grade === "B");

    // switch with fallthrough and default.
    function classify(n: number): string {
        let out: string = "";
        switch (n) {
            case 1:
                out = "one";
                break;
            case 2:
            case 3:
                out = "two-or-three";
                break;
            default:
                out = "many";
        }
        return out;
    }
    check(classify(1) === "one");
    check(classify(3) === "two-or-three");
    check(classify(9) === "many");
}

// Recursion and closures with typed signatures.
function fact(n: number): number {
    if (n <= 1) { return 1; }
    return n * fact(n - 1);
}
function makeAdder(x: number): (y: number) => number {
    return (y: number): number => x + y;
}
function makeCounter(): () => number {
    let n: number = 0;
    return function (): number { n = n + 1; return n; };
}

function testFunctions(): void {
    check(fact(5) === 120);
    const add5: (y: number) => number = makeAdder(5);
    check(add5(3) === 8);
    check(add5(10) === 15);
    const c: () => number = makeCounter();
    check(c() === 1);
    check(c() === 2);
    check(c() === 3);
    // Arrow with an inferred (erased) type and a typed reducer.
    const nums: number[] = [1, 2, 3, 4];
    const doubler = (z: number): number => z * 2;
    let acc: number = 0;
    for (const z of nums) { acc = acc + doubler(z); }
    check(acc === 20);
}

// A function taking a string-literal union type.
function step(dir: Direction): number {
    if (dir === "north") { return 1; }
    if (dir === "south") { return -1; }
    return 0;
}
function testUnions(): void {
    check(step("north") === 1);
    check(step("south") === -1);
    check(step("east") === 0);
}

function main(): number {
    testArithmetic();
    testLogic();
    testEnum();
    testInterfaces();
    testGenerics();
    testStrings();
    testArrays();
    testMaps();
    testControlFlow();
    testFunctions();
    testUnions();
    return failures;
}
