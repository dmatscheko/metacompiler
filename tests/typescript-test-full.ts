// Full-syntax test: TypeScript (5.x core grammar on ES2022).
//
// This file belongs to the SECOND test group (./test.sh --full): it is NOT part
// of the default matrix. The goal of the metacompiler is to support the full
// languages; this file is the ratchet that measures how far the typescript
// grammars are. It walks the whole practical TypeScript 5 syntax, one
// self-contained SECTION per language area. The --full runner runs the file,
// and whenever a grammar aborts it removes the section around the error and
// retries - so the report lists every unsupported section, not just the first.
//
// Conventions (shared by every *-test-full.* file):
//   - prologue (before the first SECTION marker): the check helper only
//   - each section: '// ===== SECTION <nn>: <name> =====', top-level,
//     self-contained, no references to other sections
//   - main() calls each section via a line tagged 'SECTION-CALL <nn>'
//     and prints the summary line 'full: <checks> checks, <failures> failures'
//   - main() returns the failure count (exit 0 == full support, verified)
//
// TypeScript is a typed superset of JavaScript: this file covers the TYPE
// syntax, and the ES syntax only where the type system touches it (classes,
// parameters, catch clauses, async/generator signatures); the plain runtime
// grammar lives in js-test-full.js. Types are erased at run time, so every
// type-level construct is asserted through a value-level consequence - the
// point is that the grammar must parse it and the compiler must accept it.
//
// Deliberately out of scope (not syntax, or unrunnable in this harness):
// import/export modules, triple-slash directives, JSX, ambient declarations
// ('declare', except the single println shim in SECTION 20 that makes this
// file typecheck standalone under tsc), and the standard library (only the
// lib types that async/generator signatures force: Promise, Generator).
//
// Hand-written for the metacompiler project (Apache-2.0, no copied test-suite
// code), organized after the TypeScript 5 handbook and reference with the
// ANTLR grammars-v4 TypeScript grammar as a coverage checklist.

let failures: number = 0;
let checks: number = 0;

function check(id: string, cond: boolean): void {
    checks = checks + 1;
    if (!cond) { println("FAIL " + id); failures = failures + 1; }
}

// ===== SECTION 01: baseline =====
// Condensed re-assertion of the feature-matrix basics this file builds on.
function s01() {
    let n: number = 0;
    for (let i: number = 0; i < 3; i++) { n = n + i; }
    check("bas1", n === 3);
    const o: { a: number; b?: number } = { a: 1 };
    o.b = o.a + 1;
    check("bas2", o.b === 2);
    const arr: number[] = [1, 2, 3];
    check("bas3", arr.length === 3 && arr[2] === 3);
    function add(x: number, y: number): number { return x + y; }
    check("bas4", add(2, 3) === 5);
    let t: number = 0;
    try { throw "boom"; } catch (e) { t = e === "boom" ? 1 : 2; } finally { t = t + 10; }
    check("bas5", t === 11);
}

// ===== SECTION 02: type annotations on primitives =====
function s02() {
    let n: number = 1.5;
    let s: string = "x";
    let b: boolean = true;
    let nu: null = null;
    let ud: undefined = undefined;
    check("ann1", n === 1.5 && s === "x" && b === true && nu === null && ud === undefined);
    let big: bigint | number = 10;
    check("ann2", big === 10);
    const list: number[] = [1, 2];
    const boxed: Array<string> = ["a", "b"];
    const grid: number[][] = [[1], [2, 3]];
    check("ann3", list[1] === 2 && boxed[0] === "a" && grid[1][1] === 3);
    const f: (a: number, b: number) => number = (a, b) => a + b;
    check("ann4", f(2, 3) === 5);
    const mix: (string | number)[] = [1, "two"];
    check("ann5", mix.length === 2 && mix[0] === 1);
    const rec: { tag: string; n: number } = { tag: "t", n: 4 };
    check("ann6", rec.tag === "t" && rec.n === 4);
}

// ===== SECTION 03: interfaces =====
// Optional and readonly members, index / call / construct signatures, extends.
function s03() {
    interface Named3 { name: string; nick?: string; readonly id: number; }
    const a: Named3 = { name: "Ada", id: 1 };
    check("ifc1", a.name === "Ada" && a.id === 1 && a.nick === undefined);
    interface Dict3 { [key: string]: number; }
    const d: Dict3 = { one: 1 };
    d["two"] = 2;
    check("ifc2", d["one"] + d["two"] === 3);
    interface Call3 { (a: number, b: number): number; }
    const add3: Call3 = function (a: number, b: number): number { return a + b; };
    check("ifc3", add3(1, 2) === 3);
    interface Maker3 { new (v: number): { v: number }; }
    class Boxy3 { v: number; constructor(v: number) { this.v = v; } }
    function build(m: Maker3, n: number): number { return new m(n).v; }
    check("ifc4", build(Boxy3, 6) === 6);
    interface Sized3 { size(): number; }
    interface Big3 extends Sized3 { big: boolean; }
    const bs: Big3 = { big: true, size: function (): number { return 9; } };
    check("ifc5", bs.size() === 9 && bs.big === true);
}

// ===== SECTION 04: type aliases, unions, intersections, literal types =====
function s04() {
    type Id4 = number | string;
    function fmt(x: Id4): string { return typeof x === "number" ? "#" + x : x; }
    check("ali1", fmt(3) === "#3" && fmt("ab") === "ab");
    type Dir4 = "up" | "down";
    function step(d: Dir4): number { return d === "up" ? 1 : -1; }
    check("ali2", step("up") - step("down") === 2);
    type One4 = 1;
    const one: One4 = 1;
    check("ali3", one === 1);
    type Yes4 = true;
    const y: Yes4 = true;
    check("ali4", y === true);
    type A4 = { a: number };
    type B4 = { b: string };
    const ab: A4 & B4 = { a: 1, b: "x" };
    check("ali5", ab.a === 1 && ab.b === "x");
    type Fn4 = (n: number) => number;
    const dbl: Fn4 = n => n * 2;
    check("ali6", dbl(4) === 8);
}

// ===== SECTION 05: generics =====
// Generic functions, arrows, interfaces and classes; constraints, defaults,
// and explicit type arguments at the call site.
function s05() {
    function ident<T>(x: T): T { return x; }
    check("gen1", ident(4) === 4 && ident("g") === "g");
    check("gen2", ident<string>("e") === "e");
    const first = <T>(xs: T[]): T => xs[0];
    check("gen3", first([7, 8]) === 7);
    function longest<T extends { length: number }>(a: T, b: T): T { return a.length >= b.length ? a : b; }
    check("gen4", longest("abc", "de") === "abc");
    function pair<T, U = string>(a: T, b: U): U { return b; }
    check("gen5", pair<number>(1, "s") === "s");
    interface Holder5<T> { v: T; }
    const h: Holder5<boolean> = { v: true };
    check("gen6", h.v === true);
    class Stack5<T> {
        items: T[] = [];
        push(x: T): void { this.items.push(x); }
        top(): T { return this.items[this.items.length - 1]; }
    }
    const st = new Stack5<string>();
    st.push("a");
    st.push("b");
    check("gen7", st.top() === "b");
}

// ===== SECTION 06: enums =====
// Auto-numbered, explicitly started, computed, string and const enums,
// plus the reverse mapping of a numeric enum.
function s06() {
    enum Color6 { Red, Green, Blue }
    check("enu1", Color6.Red === 0 && Color6.Blue === 2);
    check("enu2", Color6[1] === "Green");
    enum Start6 { A = 5, B }
    check("enu3", Start6.A === 5 && Start6.B === 6);
    enum Bits6 { None = 0, Two = 1 << 1, Both = Two | 1 }
    check("enu4", Bits6.Two === 2 && Bits6.Both === 3);
    enum Mode6 { On = "on", Off = "off" }
    const m: string = Mode6.On;
    check("enu5", m === "on");
    const enum Dir6 { Up, Down }
    check("enu6", Dir6.Down === 1);
}

// ===== SECTION 07: class modifiers, accessors, statics =====
// public/private/protected/readonly members, getters/setters, static fields,
// implements, and constructor parameter properties.
function s07() {
    interface Describable7 { describe(): string; }
    class Acct7 implements Describable7 {
        readonly id: number;
        private balance: number;
        protected kind: string = "acct";
        static count: number = 0;
        constructor(id: number, start: number) { this.id = id; this.balance = start; Acct7.count = Acct7.count + 1; }
        get total(): number { return this.balance; }
        set total(v: number) { this.balance = v; }
        deposit(n: number): void { this.balance = this.balance + n; }
        describe(): string { return this.kind + "#" + this.id; }
    }
    const a = new Acct7(1, 10);
    a.deposit(5);
    check("cls1", a.total === 15 && a.id === 1);
    a.total = 3;
    check("cls2", a.total === 3 && a.describe() === "acct#1");
    new Acct7(2, 0);
    check("cls3", Acct7.count === 2);
    class Pt7 {
        constructor(public x: number, private y: number, readonly z: number) {}
        sum(): number { return this.x + this.y + this.z; }
    }
    const p = new Pt7(1, 2, 3);
    check("cls4", p.sum() === 6 && p.x === 1 && p.z === 3);
}

// ===== SECTION 08: abstract classes and inheritance =====
function s08() {
    abstract class Shape8 {
        protected label: string;
        constructor(label: string) { this.label = label; }
        abstract area(): number;
        describe(): string { return this.label + ":" + this.area(); }
    }
    class Sq8 extends Shape8 {
        constructor(private side: number) { super("sq"); }
        override area(): number { return this.side * this.side; }
    }
    class Tri8 extends Shape8 {
        constructor(private b: number, private h: number) { super("tri"); }
        override area(): number { return (this.b * this.h) / 2; }
        override describe(): string { return "T" + super.describe(); }
    }
    const s = new Sq8(3);
    check("abs1", s.area() === 9 && s.describe() === "sq:9");
    check("abs2", new Tri8(4, 3).describe() === "Ttri:6");
    const shapes: Shape8[] = [new Sq8(2), new Tri8(2, 2)];
    check("abs3", shapes[0].area() + shapes[1].area() === 6);
    const sh: Shape8 = new Sq8(4);
    check("abs4", sh.describe() === "sq:16");
}

// ===== SECTION 09: assertions and casts =====
// as, as const, the non-null postfix !, definite assignment !, satisfies.
function s09() {
    const wide: unknown = "text";
    const s = wide as string;
    check("ast1", s.length === 4);
    const lit = { kind: "a", n: 1 } as const;
    check("ast2", lit.kind === "a" && lit.n === 1);
    const pairc = [1, 2] as const;
    check("ast3", pairc[0] + pairc[1] === 3);
    let later!: number;
    later = 41;
    check("ast4", later + 1 === 42);
    function firstChar(x: string | null): string { return x!.charAt(0); }
    check("ast5", firstChar("zap") === "z");
    const sat = { a: 1, b: "s" } satisfies { a: number; b: string };
    check("ast6", sat.a === 1 && sat.b === "s");
}

// ===== SECTION 10: keyof, typeof types, indexed access =====
function s10() {
    const conf = { host: "h", port: 80 };
    type Conf10 = typeof conf;
    type Key10 = keyof Conf10;
    const k: Key10 = "port";
    check("key1", k === "port");
    function getProp<T, K extends keyof T>(o: T, key: K): T[K] { return o[key]; }
    check("key2", getProp(conf, "port") === 80 && getProp(conf, "host") === "h");
    type Port10 = Conf10["port"];
    const p: Port10 = 8080;
    check("key3", p === 8080);
    const c2: Conf10 = { host: "x", port: 1 };
    check("key4", c2.port === 1);
}

// ===== SECTION 11: mapped, conditional and template literal types =====
function s11() {
    type Flags11<T> = { [K in keyof T]: boolean };
    const fl: Flags11<{ a: number; b: string }> = { a: true, b: false };
    check("map1", fl.a === true && fl.b === false);
    type Part11<T> = { [K in keyof T]?: T[K] };
    const half: Part11<{ a: number; b: string }> = { a: 1 };
    check("map2", half.a === 1 && half.b === undefined);
    type Ro11<T> = { readonly [K in keyof T]: T[K] };
    const ro: Ro11<{ n: number }> = { n: 7 };
    check("map3", ro.n === 7);
    type Getters11<T> = { [K in keyof T as `get${string & K}`]: () => T[K] };
    const g: Getters11<{ n: number }> = { getn: function (): number { return 5; } };
    check("map4", g.getn() === 5);
    type Elem11<A> = A extends (infer E)[] ? E : never;
    const e: Elem11<number[]> = 6;
    check("map5", e === 6);
    type Is11<T> = T extends string ? "yes" : "no";
    const yn: Is11<"x"> = "yes";
    check("map6", yn === "yes");
    type Route11 = `/api/${string}`;
    const r: Route11 = "/api/users";
    check("map7", r === "/api/users");
}

// ===== SECTION 12: tuples =====
// Fixed, optional, rest, labeled and readonly tuple elements.
function s12() {
    const t: [number, string] = [1, "a"];
    check("tup1", t[0] === 1 && t[1] === "a" && t.length === 2);
    const opt: [number, boolean?] = [1];
    check("tup2", opt.length === 1 && opt[1] === undefined);
    const rest: [string, ...number[]] = ["x", 1, 2];
    check("tup3", rest.length === 3 && rest[2] === 2);
    type Vec12 = [x: number, y: number];
    const v: Vec12 = [3, 4];
    check("tup4", v[0] + v[1] === 7);
    const ro: readonly [number, number] = [1, 2];
    check("tup5", ro[0] + ro[1] === 3);
    const nums: readonly number[] = [1, 2, 3];
    check("tup6", nums[2] === 3 && nums.length === 3);
}

// ===== SECTION 13: function types and overloads =====
// Overload signatures, optional / default / rest parameters, this typing.
function s13() {
    function wrap(x: number): number;
    function wrap(x: string): string;
    function wrap(x: number | string): number | string { return typeof x === "number" ? x + 1 : x + "!"; }
    check("fun1", wrap(1) === 2 && wrap("a") === "a!");
    function greet(name: string, punct?: string): string { return punct === undefined ? name + "." : name + punct; }
    check("fun2", greet("A") === "A." && greet("B", "!") === "B!");
    function scale(n: number, by: number = 3): number { return n * by; }
    check("fun3", scale(2) === 6 && scale(2, 5) === 10);
    function total(...nums: number[]): number {
        let t: number = 0;
        for (const n of nums) { t = t + n; }
        return t;
    }
    check("fun4", total(1, 2, 3) === 6 && total() === 0);
    function getV(this: { v: number }): number { return this.v; }
    const holder = { v: 8, getV: getV };
    check("fun5", holder.getV() === 8);
    const typedFn: (n: number) => string = function (n: number): string { return "n" + n; };
    check("fun6", typedFn(2) === "n2");
}

// ===== SECTION 14: narrowing and type guards =====
// typeof / instanceof / in narrowing, user-defined 'x is T' predicates,
// and a discriminated union.
function s14() {
    function describe(x: number | string): string {
        if (typeof x === "string") { return "s" + x.length; }
        return "n" + x;
    }
    check("nrw1", describe("ab") === "s2" && describe(4) === "n4");
    class Cat14 { meow(): string { return "meow"; } }
    class Dog14 { bark(): string { return "woof"; } }
    function talk(a: Cat14 | Dog14): string { return a instanceof Cat14 ? a.meow() : a.bark(); }
    check("nrw2", talk(new Cat14()) === "meow" && talk(new Dog14()) === "woof");
    function pick(o: { a: number } | { b: number }): number { return "a" in o ? o.a : o.b; }
    check("nrw3", pick({ a: 4 }) === 4 && pick({ b: 5 }) === 5);
    type Fish14 = { swim: () => string };
    type Bird14 = { fly: () => string };
    function isFish(p: Fish14 | Bird14): p is Fish14 { return (p as Fish14).swim !== undefined; }
    function move(p: Fish14 | Bird14): string { return isFish(p) ? p.swim() : p.fly(); }
    check("nrw4", move({ swim: function (): string { return "s"; } }) === "s");
    type Shape14 = { kind: "c"; r: number } | { kind: "q"; side: number };
    function area(sh: Shape14): number {
        switch (sh.kind) {
            case "c": return sh.r * 3;
            case "q": return sh.side * sh.side;
        }
    }
    check("nrw5", area({ kind: "c", r: 2 }) === 6 && area({ kind: "q", side: 3 }) === 9);
}

// ===== SECTION 15: namespaces =====
// Exported members, nesting, dotted names, and namespace merging.
namespace Pack15 {
    export const base: number = 40;
    export function bump(n: number): number { return n + base; }
    export namespace Inner {
        export function twice(n: number): number { return n * 2; }
    }
}
namespace Pack15 {
    export function more(): number { return Pack15.base + 2; }
}
namespace Deep15.Sub {
    export const k: number = 7;
}
function s15() {
    check("nsp1", Pack15.bump(2) === 42);
    check("nsp2", Pack15.Inner.twice(21) === 42);
    check("nsp3", Pack15.more() === 42);
    check("nsp4", Deep15.Sub.k === 7);
}

// ===== SECTION 16: declaration merging =====
// Two interface declarations merge; an interface merges into a class.
interface Merged16 { a: number; }
interface Merged16 { b: string; }
interface Wide16 extends Merged16 { c: boolean; }
class Tag16 { n: number = 1; }
interface Tag16 { extra?: string; }
function s16() {
    const m: Merged16 = { a: 1, b: "x" };
    check("mrg1", m.a === 1 && m.b === "x");
    const w: Wide16 = { a: 2, b: "y", c: true };
    check("mrg2", w.c === true && w.a === 2);
    const t = new Tag16();
    t.extra = "e";
    check("mrg3", t.n === 1 && t.extra === "e");
}

// ===== SECTION 17: decorators =====
// TC39 standard decorators: method and class decorators, and a factory.
function s17() {
    function doubled(original: (this: unknown, n: number) => number, _ctx: unknown) {
        return function (this: unknown, n: number): number { return original.call(this, n) * 2; };
    }
    let log: string = "";
    function traced(_value: unknown, _ctx: unknown): void { log = log + "T"; }
    function addTag(base: any, _ctx: unknown): any { return class extends base { tag: string = "dec"; }; }
    class Calc17 {
        bonus: number = 1;
        @doubled
        val(n: number): number { return n + this.bonus; }
    }
    check("dec1", new Calc17().val(4) === 10);
    @traced
    class Marked17 { id: number = 3; }
    check("dec2", log === "T" && new Marked17().id === 3);
    @addTag
    class Widget17 { w: number = 7; }
    const wd = new Widget17();
    check("dec3", wd.w === 7 && (wd as any).tag === "dec");
    function addN(extra: number) {
        return function (original: (this: unknown) => number, _ctx: unknown) {
            return function (this: unknown): number { return original.call(this) + extra; };
        };
    }
    class Fact17 { @addN(5) ten(): number { return 10; } }
    check("dec4", new Fact17().ten() === 15);
}

// ===== SECTION 18: any, unknown, never, void =====
function s18() {
    let a: any = 1;
    a = "str";
    a = { p: 5 };
    check("unk1", a.p === 5);
    let u: unknown = "hello";
    let len: number = 0;
    if (typeof u === "string") { len = u.length; }
    check("unk2", len === 5);
    function boom(msg: string): never { throw msg; }
    let got: string = "";
    try { boom("nope"); } catch (e: unknown) { got = typeof e === "string" ? e : "?"; }
    check("unk3", got === "nope");
    function quiet(x: number): void { if (x < 0) { return; } }
    const r: void = quiet(1);
    check("unk4", typeof r === "undefined");
    const empty: never[] = [];
    check("unk5", empty.length === 0);
}

// ===== SECTION 19: async and generator typing =====
// Defined and type-checked only where running would need an event loop.
function s19() {
    async function total(): Promise<number> { return 41; }
    check("asy1", typeof total === "function");
    const bump = async (x: number): Promise<number> => x + 1;
    check("asy2", typeof bump === "function");
    async function chain(p: Promise<string>): Promise<string> { const s: string = await p; return s + "!"; }
    check("asy3", typeof chain === "function");
    function* seq(): Generator<number, string, void> { yield 1; yield 2; return "end"; }
    const it = seq();
    const head = it.next();
    check("asy4", head.value === 1 && head.done === false);
    async function* astream(): AsyncGenerator<number> { yield 1; }
    check("asy5", typeof astream === "function");
}

// ===== SECTION 20: ambient declaration (the harness print shim) =====
// println is the metacompiler's built-in output primitive; this single
// ambient declaration makes the file typecheck standalone under plain tsc.
declare function println(msg: string): void;
function s20() {
    check("amb1", typeof println === "function");
}

// ===== END SECTIONS =====

function main(): number {
    s01(); // SECTION-CALL 01
    s02(); // SECTION-CALL 02
    s03(); // SECTION-CALL 03
    s04(); // SECTION-CALL 04
    s05(); // SECTION-CALL 05
    s06(); // SECTION-CALL 06
    s07(); // SECTION-CALL 07
    s08(); // SECTION-CALL 08
    s09(); // SECTION-CALL 09
    s10(); // SECTION-CALL 10
    s11(); // SECTION-CALL 11
    s12(); // SECTION-CALL 12
    s13(); // SECTION-CALL 13
    s14(); // SECTION-CALL 14
    s15(); // SECTION-CALL 15
    s16(); // SECTION-CALL 16
    s17(); // SECTION-CALL 17
    s18(); // SECTION-CALL 18
    s19(); // SECTION-CALL 19
    s20(); // SECTION-CALL 20
    println("full: " + checks + " checks, " + failures + " failures");
    return failures;
}
