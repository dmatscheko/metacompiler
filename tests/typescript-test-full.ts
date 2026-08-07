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
// cross-module import linkage, triple-slash directives, JSX, ambient declarations
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
    // Async functions RUN here: an async body is compiled as a generator body whose
    // awaits are its yields, and a microtask queue drained after main() drives them
    // (see makeAsyncFn / js_jsasyncfn). The ORDERING below is byte-identical to node,
    // which is the only part a wrong implementation cannot get right by accident.
    // Two shapes are deliberately avoided: an async body must have NO side effect
    // before its last await (this half drives one by REPLAY, so a print before an
    // await would repeat), and `for await` / `async function*` are not implemented
    // and ABORT, which is why an async generator is no longer declared here.
    check("asy5", typeof Promise === "function");
    const alog: string[] = [];
    async function step(tag: string): Promise<void> { await 0; alog.push(tag); }
    async function twice(tag: string): Promise<void> { await 0; await 0; alog.push(tag); }
    async function bad(): Promise<void> { await 0; throw "boom"; }
    async function guard(tag: string): Promise<void> { try { await Promise.reject("r"); } catch (e) { alog.push(tag + e); } }
    Promise.resolve().then(function () { alog.push("t1"); }).then(function () { alog.push("t2"); });
    step("a");
    Promise.resolve(5).then(function (v: any) { alog.push("v" + v); });
    twice("w");
    guard("g");
    bad().catch(function (e: any) { alog.push("c" + e); });
    new Promise(function (res: any) { res("n"); }).then(function (v: any) { alog.push(v); });
    Promise.all([1, Promise.resolve(2)]).then(function (a: any) { alog.push("all" + a.join("")); });
    bump(1).then(function (v: any) { alog.push("ar" + v); });
    chain(Promise.resolve("z")).then(function (v: any) { alog.push("ch" + v); });
    Promise.resolve(7).finally(function () { alog.push("f"); }).then(function (v: any) { alog.push("fv" + v); });
    // The trace is complete only once the queue has been drained, which happens after
    // main() has returned - so the assertion runs in a job of its own, six ticks deep,
    // which is later than anything above can schedule.
    Promise.resolve().then(nop).then(nop).then(nop).then(nop).then(nop).then(function () {
        const ok: boolean = alog.join(",") === "t1,a,v5,gr,n,ar2,f,t2,w,cboom,all12,chz!,fv7";
        check("asy14", ok);
        // main() has already returned by the time a job runs, so a failure here cannot
        // reach the count it reported - exit() is the only way it reaches the EXIT CODE,
        // which is what clang-check and native-full read.
        if (!ok) { exit(1); }
    });
    // for-of drives an iterator ONE STEP AT A TIME, so the producer's side effects
    // INTERLEAVE with the consumer's instead of all happening first. The check is not
    // the exact log, because the interpreter half's generators REPLAY (every next()
    // re-runs the body, so a producer's effects repeat) - it is the one thing the two
    // halves share and a drain-first loop cannot have: the first value is CONSUMED
    // before the second is ever PRODUCED.
    let log: string = "";
    function* logged(): Generator<number, void, void> { log = log + "p1 "; yield 1; log = log + "p2 "; yield 2; }
    for (const lv of logged()) { log = log + "c" + lv + " "; }
    check("asy6", log.indexOf("c1") < log.lastIndexOf("p2"));
    // A loop that BREAKS never asks for the values it did not reach. 'endless' is
    // written as an endless generator and then bounded at 1000, so that a regression
    // to a drain-first for-of FAILS this check instead of hanging the suite.
    let reached: number = -1;
    function* endless(): Generator<number, void, void> { let i: number = 0; while (i < 1000) { reached = i; yield i; i = i + 1; } }
    const taken: number[] = [];
    for (const ev of endless()) { if (taken.length === 3) { break; } taken.push(ev); }
    check("asy7", taken.join(",") === "0,1,2" && reached === 3);
    // A HAND-WRITTEN iterator: an object whose next() answers {value, done}. There are
    // no symbols in this subset, so a callable `next` is the whole protocol - node
    // needs a [Symbol.iterator] beside it and throws "cursor is not iterable" without
    // one, and this line is the deliberate divergence that buys hand-written iterators
    // a spelling here at all.
    const cursor = {
        i: 0,
        next: function (): any {
            this.i = this.i + 1;
            if (this.i > 3) { return {value: undefined, done: true}; }
            return {value: this.i * 10, done: false};
        }
    };
    const cs: number[] = [];
    for (const cv of cursor) { cs.push(cv); }
    check("asy8", cs.join(",") === "10,20,30");
    // A destructuring loop head over a generator.
    function* pairs(): Generator<any, void, void> { yield [1, "a"]; yield [2, "b"]; }
    let flat: string = "";
    for (const [pn, ps] of pairs()) { flat = flat + pn + ps; }
    check("asy9", flat === "1a2b");
    // 'yield*' delegates to a GENERATOR, not only to an array, and lazily: the second
    // loop stops the delegate after two values, 998 short of its bound.
    function* inner19(): Generator<number, void, void> { yield 7; yield 8; }
    function* deleg(): Generator<number, void, void> { yield 0; yield* inner19(); yield 9; }
    const ds: number[] = [];
    for (const dv of deleg()) { ds.push(dv); }
    check("asy10", ds.join(",") === "0,7,8,9");
    reached = -1;
    function* delegEndless(): Generator<number, void, void> { yield* endless(); }
    const dn: number[] = [];
    for (const xv of delegEndless()) { if (dn.length === 2) { break; } dn.push(xv); }
    check("asy11", dn.join(",") === "0,1" && reached === 2);
}

function nop(): void {}

// ===== SECTION 20: ambient declaration (the harness print shim) =====
// println is the metacompiler's built-in output primitive; this single
// ambient declaration makes the file typecheck standalone under plain tsc.
declare function println(msg: string): void;
function s20() {
    check("amb1", typeof println === "function");
}

// ===== SECTION 21: regular expressions =====
// Every expected value below was cross-checked against node v24.
function s21() {
    const re: RegExp = /a(b+)c/;
    check("re01", re.source === "a(b+)c" && re.flags === "" && re.global === false);
    check("re02", re.test("xxabbbc") && !re.test("ac"));
    const m = re.exec("xxabbbc");
    check("re03", m[0] === "abbbc" && m[1] === "bbb" && m.index === 2 && m.length === 2);
    check("re04", m.input === "xxabbbc" && m.groups === undefined);
    check("re05", re.exec("nope") === null);
    // Flags: i folds case, s makes '.' match a newline, m makes ^ and $ line anchors -
    // and WITHOUT m they are string anchors, which is the one place JavaScript differs
    // from Ruby.
    check("re06", /AB/i.test("xxab"));
    check("re07", /a.b/s.test("a\nb") && !/a.b/.test("a\nb"));
    check("re08", /^b$/m.test("a\nb\nc") && !/^b$/.test("a\nb\nc"));
    // g is not a match mode, it is lastIndex iteration state.
    const g: RegExp = /a/g;
    let hits: string = "";
    let one = g.exec("aXaXa");
    while (one !== null) {
        hits = hits + one.index;
        one = g.exec("aXaXa");
    }
    check("re09", hits === "024" && g.lastIndex === 0);
    // y (sticky) anchors the search at lastIndex.
    const st: RegExp = /a/y;
    check("re10", !st.test("ba"));
    st.lastIndex = 1;
    check("re11", st.test("ba") && st.lastIndex === 2);
    // Named groups reach the match through .groups.
    const nm = /(?<y>[0-9]{4})-(?<m>[0-9]{2})/.exec("on 2024-05-06");
    check("re12", nm.groups.y === "2024" && nm.groups.m === "05" && nm.index === 3);
    // String.prototype.match / matchAll / search.
    check("re13", "a1b22".match(/\d+/)[0] === "1");
    check("re14", "a1b22".match(/\d+/g).length === 2);
    check("re15", "abc".match(/z/) === null);
    check("re16", "abc".search(/c/) === 2 && "abc".search(/z/) === -1);
    const all = [... "a1b2".matchAll(/(\w)(\d)/g)];
    let joined: string = "";
    for (let i: number = 0; i < all.length; i++) { joined = joined + all[i][1] + all[i][2]; }
    check("re17", joined === "a1b2");
    // replace / replaceAll, with the $-template and with a function.
    check("re18", "a1b22".replace(/\d+/, "#") === "a#b22");
    check("re19", "a1b22".replace(/\d+/g, "#") === "a#b#");
    check("re20", "one two".replace(/(\w+) (\w+)/, "$2 $1") === "two one");
    check("re21", "abc".replace(/b/, "[$&]") === "a[b]c");
    check("re22", "ab".replace(/a/, "$$") === "$b");
    check("re23", "2024-05".replace(/(?<y>\d{4})-(?<m>\d{2})/, "$<m>/$<y>") === "05/2024");
    check("re24", "aXbXc".replace(/X/g, function (w: string, off: number): string { return "(" + off + ")"; }) === "a(1)b(3)c");
    check("re25", "aaa".replaceAll(/a/g, "b") === "bbb");
    // split with a pattern keeps the separator's capture groups.
    check("re26", "a,b,,c".split(/,/).length === 4);
    check("re27", "a-b_c".split(/([-_])/).length === 5);
    check("re28", "a1b".split(/\d/)[1] === "b");
    // A string argument is not a pattern: these keep the plain string behaviour.
    check("re29", "a.c".replace(".", "-") === "a-c" && "a.c".split(".").length === 2);
    // new RegExp / RegExp with the flags arriving at run time.
    const dyn: RegExp = new RegExp("b+", "i");
    check("re30", dyn.test("aBBc") && dyn.source === "b+" && dyn.flags === "i");
    const dyn2: RegExp = RegExp("\\d");
    check("re31", dyn2.test("x9") && !dyn2.test("xy"));
    check("re32", typeof RegExp === "function" && typeof dyn === "object");
    // Backreferences, alternation, lazy quantifiers and a group that did not take part.
    check("re33", /(a)\1/.test("aa") && !/(a)\1/.test("ab"));
    check("re34", /<(.+?)>/.exec("<a><b>")[1] === "a");
    check("re35", /(a)(b)?/.exec("a")[2] === undefined);
    check("re36", /^(cat|dog)s?$/.test("dogs"));
    // Lookahead, and the escape forms the engine reads out of the RAW body.
    check("re37", /foo(?!bar)/.test("foobaz") && !/foo(?!bar)/.test("foobar"));
    check("re38", /a\/b/.test("a/b") && /\[\]/.test("[]"));
    check("re39", /^\d{2,3}$/.test("123") && !/^\d{2,3}$/.test("1234"));
    check("re40", /[a-z0-9_$]+/.exec("__x9!")[0] === "__x9");
    // A '/' after a value is still division, which is what keeps the literal out of the
    // way of ordinary arithmetic.
    const division: number = 8 / 2 / 2;
    check("re41", division === 2);
    check("re42", /x/.toString() === "/x/" && /x/gi.toString() === "/x/gi");
    // Each evaluation of a literal yields a NEW object, so the lastIndex of a global
    // pattern written inside a loop starts over every time round.
    let counted: number = 0;
    for (let k: number = 0; k < 3; k++) { if (/z/g.test("z")) { counted = counted + 1; } }
    check("re43", counted === 3);
    // Named backreferences, including a FORWARD one (the group is declared later).
    check("re44", /(?<n>a)\k<n>/.test("aa") && !/(?<n>a)\k<n>/.test("ab"));
    check("re45", /\k<n>(?<n>a)/.test("a"));
    check("re46", /(?<w>ab)-\k<w>/.exec("zab-ab")![0] === "ab-ab");
    check("re47", "xayb".replace(/(?<c>[ab])/g, "[$<c>]") === "x[a]y[b]");
    // Annex B legacy octal: a backslash-digit escape with no such group is a
    // CHARACTER, not a backreference. \1 is U+0001, so it cannot match the empty
    // string; 8 and 9 are not octal digits and stay identity escapes; and 4-7 admit
    // only two more digits, which is why \400 reads as \40 followed by "0".
    check("re48", !/\1/.test("") && /(a)\1/.test("aa") && !/(a)\2/.test("a"));
    check("re49", /\12/.test("\n") && /\101/.test("A") && /\400/.test(" 0"));
    check("re50", /\8/.test("8") && /\9/.test("9"));
    // Sticky: the match must BEGIN at lastIndex, not merely after it.
    const sticky: RegExp = /b/y;
    sticky.lastIndex = 1;
    check("re51", sticky.test("ab") && sticky.lastIndex === 2);
    const sticky2: RegExp = /b/y;
    sticky2.lastIndex = 0;
    check("re52", !sticky2.test("ab"));
    // \Q ... \E is a Java/Kotlin quote region and NOT a JavaScript one: here \Q and
    // \E are identity escapes, so this pattern is the literal text "Qa.bE". The
    // shared engine offers quote regions behind a dialect flag that JavaScript does
    // not pass, and this check is what keeps it from creeping in.
    check("re53", !/\Qa.b\E/.test("a.b") && /\Qa.b\E/.test("Qa.bE"));
    // ===== JavaScript repeat semantics =====
    // Two rules the shared engine keeps behind a dialect flag, both node v24's.
    // (a) An iteration of an unbounded loop that consumed NOTHING is discarded, so
    //     /^(a*)*$/ on "aaa" reports "aaa" for group 1 and not "" - Perl, Python,
    //     Ruby and Java print "" here, node and POSIX print "aaa".
    check("re54", /^(a*)*$/.exec("aaa")[1] === "aaa" && /^(a*)*$/.exec("")[1] === undefined);
    check("re55", /(a*)*/.exec("b")[1] === undefined &&
                  "aaa".replace(/(a*)*/g, "[$1]") === "[aaa][]");
    check("re56", /^(a|b|)*$/.exec("ab")[1] === "b" && /(a?)*/.exec("aa")[1] === "a" &&
                  /(|a)*/.exec("aa")[0] === "aa");
    check("re57", /^(a*)+$/.exec("aaa")[1] === "aaa" && /^(a*)*?$/.exec("aaa")[1] === "aaa");
    // (b) The captures INSIDE a repeated atom are cleared at the start of every
    //     iteration, so a group that took part in an earlier one but not the last
    //     reads as "did not take part". This holds for {n,m} as well as for + and *.
    check("re58", /(?:(a)|b)+/.exec("ab")[1] === undefined &&
                  /(?:(a)|b)+/.exec("ba")[1] === "a");
    check("re59", /^((a)|b*)*$/.exec("ab")[1] === "b" &&
                  /^((a)|b*)*$/.exec("ab")[2] === undefined &&
                  /(?:(a)|b){2}/.exec("ab")[1] === undefined);
}

// ===== SECTION 22: JavaScript value semantics (rendering, ==, instanceof, methods) =====
// The ratchet for the print/String rendering, ToPrimitive in == and the relational
// operators, instanceof over a plain constructor, delete on an array element, the
// Object statics and the String/Array/Number method surface. Every expected value
// below was verified against node v24 (the same section runs green under node with
// the type annotations stripped). Before this section the two halves and the two
// ENGINES disagreed on all of it: the compiler printed Go's %v ("<nil>", "[1 2 3]",
// "map[a:1]"), the interpreter leaked its hidden __keys slot out of Object.keys, and
// under -frozen the interpreter had neither ToPrimitive nor a method surface.
function Plain22(this: any): void { this.k = 1; }
class Kls22 { }
function s22(): void {
    check("val1", String(null) === "null" && String(undefined) === "undefined");
    check("val2", String([1, 2, 3]) === "1,2,3" && String([]) === "");
    check("val3", String({ a: 1 }) === "[object Object]");
    check("val4", String([1, null, 2]) === "1,,2"); // a null element joins as ""
    check("val5", ([1] as any) == 1 && ([] as any) == false);
    check("val6", ({} as any) == "[object Object]" && (null as any) == undefined);
    check("val7", !((null as any) == 0) && !(NaN == NaN));
    check("val8", ([2] as any) > 1 && ([2] as any) < 3 && 2 < ("10" as any));
    const p: any = new Plain22();
    check("val9", p instanceof Plain22 && p.k === 1);
    check("val10", new Kls22() instanceof Kls22);
    const arr: any[] = [1, 2, 3];
    delete arr[1];
    check("val11", arr.length === 3 && arr[1] === undefined);
    check("val12", Object.keys({ a: 1, b: 2 }).length === 2 && Object.keys({ a: 1, b: 2 })[1] === "b");
    check("val13", String(Object.entries({ a: 1 })) === "a,1");
    check("val14", String(Object.values({ a: 1, b: 2 })) === "1,2");
    check("val15", String([1, [2, [3]]].flat(2)) === "1,2,3" && "abc".padStart(5, "-") === "--abc");
    check("val16", "abcabc".indexOf("b", 2) === 4 && (5).toFixed(2) === "5.00");
    check("val17", String([1, 2].map(function (x: number, i: number) { return x + i; })) === "1,3");
    check("val18", String([1, 2, 3].slice(-2)) === "2,3" && String("a,b".split(",")) === "a,b");
}

// ===== SECTION 23: super as a property =====
class S23Base {
    tag: string;
    boxv: any;
    n: any;
    fresh: any;
    d: any;
    e: any;
    ctrv: number;
    constructor() { this.tag = "base"; this.ctrv = 4; }
    get ctr(): number { return this.ctrv; }
    set ctr(v: number) { this.ctrv = v; }
    get label(): string { return "B:" + this.tag; }
    set label(v: string) { this.tag = v; }
    get box(): any { return this.boxv; }
    kind(): string { return "base"; }
}
class S23Sub extends S23Base {
    own: number;
    constructor() { super(); this.own = 1; }
    readGet(): string { return super.label; }
    readComputed(k: string): string { return (super[k] as any).call(this); }
    writeSetter(): string { super.label = "set"; return this.tag; }
    writePlain(): any { super.fresh = 5; return this.fresh; }
    compound(): any { super.n = 3; return super.n; }
    viaPath(): any { this.boxv = { v: 0 }; super.box.v = 9; return this.boxv.v; }
    destruct(): any { [super.d] = [11]; return this.d; }
    destructObj(): any { ({ q: super.e } = { q: 12 }); return this.e; }
    kind(): string { return "sub<" + super.kind() + ">"; }
    // '++super.x' / 'super.x--': the read and the store both go through the class chain,
    // so the accessor pair above is what runs. The prefix form yields the NEW value and
    // the postfix one the old, exactly as on an ordinary member.
    preUpdate(): number { return ++super.ctr; }
    postUpdate(): number { return super.ctr--; }
    // With a suffix behind it only the BASE is a super read: 'super.box.v' reads box
    // through the chain and then updates an ordinary member of the object it answers.
    pathUpdate(): string {
        this.boxv = { v: 4 };
        return (++super.box.v) + "," + this.boxv.v + "," + (super.box.v--) + "," + this.boxv.v;
    }
}
function s23(): void {
    const s = new S23Sub();
    check("sup1", s.readGet() === "B:base");
    check("sup2", s.readComputed("kind") === "base");
    check("sup3", s.writeSetter() === "set");
    check("sup4", s.writePlain() === 5);
    // 'super.n = 3' stores on the RECEIVER; 'super.n' then reads the class chain, which
    // has no n at all - so the read is undefined, exactly as in node.
    check("sup5", s.compound() === undefined);
    check("sup6", s.viaPath() === 9);
    check("sup7", s.destruct() === 11);
    check("sup8", s.destructObj() === 12);
    check("sup9", s.kind() === "sub<base>");
    check("sup10", s.preUpdate() === 5 && s.ctrv === 5);
    check("sup11", s.postUpdate() === 5 && s.ctrv === 4);
    check("sup12", s.pathUpdate() === "5,5,5,4");
}

// ===== SECTION 24: the general new forms =====
class S24C { x: number; constructor(x?: number) { this.x = x === undefined ? -1 : x; } }
function S24F(this: any): void { this.p = 9; }
const s24ns = { inner: { C: S24C } };
const s24arr: any[] = [S24F];
function s24mk(): any { return S24F; }
function s24(): void {
    check("new1", new s24ns.inner.C(3).x === 3);
    check("new2", new S24C().x === -1);
    check("new3", new (S24F as any)().p === 9);
    check("new4", new (s24mk())().p === 9);
    check("new5", new s24arr[0]().p === 9);
    check("new6", (new (class { z: number; constructor() { this.z = 8; } })()).z === 8);
    check("new7", new S24C(2).x + new s24ns.inner.C(5).x === 7);
    check("new8", new s24ns.inner.C(4) instanceof S24C);
}

// ===== SECTION 25: a class heritage EXPRESSION =====
class S25Base { b: number; constructor() { this.b = 1; } hi(): string { return "base"; } }
function s25mixin(B: any): any { return class extends B { hi(): string { return "mix+" + super.hi(); } }; }
class S25M extends s25mixin(S25Base) { hi(): string { return "M+" + super.hi(); } }
const s25holder = { K: S25Base };
class S25N extends s25holder.K { }
function s25(): void {
    const m = new S25M();
    check("her1", m.hi() === "M+mix+base");
    check("her2", (m as any).b === 1);
    const n = new S25N();
    check("her3", n.b === 1);
    check("her4", n.hi() === "base");
    const Q = class extends (S25Base) { hi(): string { return "q"; } };
    check("her5", new Q().hi() === "q");
    check("her6", new Q().b === 1);
    check("her7", n instanceof S25Base);
}

// ===== SECTION 26: a member path as the for-of target =====
function s26(): void {
    const o: any = { a: 0, b: 0 };
    let out = "";
    for (o.a of [1, 2, 3]) { out = out + o.a; }
    check("fom1", out === "123");
    const arr: any[] = [0, 0];
    for (arr[1] of ["x", "y"]) { out = out + arr[1]; }
    check("fom2", out === "123xy");
    const p: any = { k: 0 };
    for ([p.k] of [[7], [8]]) { out = out + p.k; }
    check("fom3", out === "123xy78");
    function box(): any { return o; }
    for (box().b of [5, 6]) { out = out + o.b; }
    check("fom4", out === "123xy7856");
    const nest: any = { m: { n: 0 } };
    for (nest.m.n of [4]) { check("fom5", nest.m.n === 4); }
}

// ===== SECTION 27: a destructuring catch parameter =====
function s27(): void {
    let r: any = "";
    try { throw [1, 2, 3]; } catch ([a, ...b]) { r = a + ":" + b.length + ":" + b[1]; }
    check("cat1", r === "1:2:3");
    try { throw { m: "msg", c: 5 }; } catch ({ m, c: code }) { r = m + code; }
    check("cat2", r === "msg5");
    try { throw []; } catch ([x = 9]) { r = x; }
    check("cat3", r === 9);
    try { throw { a: { b: 7 } }; } catch ({ a: { b } }) { r = b; }
    check("cat4", r === 7);
    try { throw "plain"; } catch (e) { r = e; }
    check("cat5", r === "plain");
    try { throw "bare"; } catch { r = "nobind"; }
    check("cat6", r === "nobind");
}

// ===== SECTION 28: export, and the debugger no-op =====
export const S28K: number = 7;
export function s28f(x: number): number { return x * 2; }
export class S28C { v: number; constructor() { this.v = 3; } }
let s28side = 0;
export { s28side as s28sideOut };
export { S28K as s28alias };
export default class S28D { m(): string { return "d"; } }
export function s28mark(): number { s28side = 42; return s28side; }
function s28(): void {
    check("exp1", S28K === 7);
    check("exp2", s28f(4) === 8);
    check("exp3", new S28C().v === 3);
    check("exp4", s28mark() === 42);
    check("exp5", new S28D().m() === "d");
    let n = 0;
    debugger
    n = n + 1;
    debugger;
    check("dbg1", n === 1);
}

// ===== SECTION 29: shapes TypeScript parses and then refuses =====
// Several TypeScript diagnostics are GRAMMAR CHECKS on a tree tsc has already built, not
// parse errors, so the grammar has to reach the same tree. Each one is asserted through a
// value-level consequence, since the checker's complaint is not observable here.
class S29Base {
    ctrv: number = 4;
    get ctr(): number { return this.ctrv; }
    set ctr(v: number) { this.ctrv = v; }
}
class S29Acc {
    _v: number = 1;
    // 1054 / 1049: a getter with a parameter and a setter with none or two. The extra (or
    // missing) name simply has no effect - a getter is invoked with no argument, a setter
    // with exactly one, whatever the list says.
    get g(): number { return this._v; }
    set g(a: number, b: number) { this._v = a; }
}
// 2369: an accessibility modifier outside a constructor is only a modifier for the type
// checker; here it must parse and bind an ordinary parameter.
function s29mods(public a: number, private b: number): number { return a + b; }
// "A rest parameter must be last" - two of them parse, and the FIRST one takes the tail.
function s29rest(...x: number[], ...y: number[]): number { return x.length; }
function s29(): void {
    const a: S29Acc = new S29Acc();
    a.g = 9;
    check("gc1", a.g === 9);
    check("gc2", s29mods(2, 3) === 5);
    check("gc3", s29rest(1, 2, 3) === 3);
    // '++super.x' with a suffix behind it, and the plain postfix form.
    const b: S29Base = new S29Base();
    check("gc4", b.ctr === 4);
    // 1091: several declarators in a for-in / for-of head. The loop binds the FIRST.
    let seen: string = "";
    for (var k = 1, unused = 2 in { x: 0, y: 0 }) { seen = seen + k; }
    check("gc5", seen === "xy");
    let n: number = 0;
    for (var e, spare of [1, 2, 3]) { n = n + e; }
    check("gc6", n === 6);
    // The Annex B for-in initializer is parsed and dropped, and its 'in' belongs to the
    // LOOP - not to the initializer expression.
    let m: string = "";
    for (var q = 1 in { p: 0 }) { m = m + q; }
    check("gc7", m === "p");
    // A numeric separator in the EXPONENT, and a comma expression as a case label.
    check("gc8", 1e1_0 === 10000000000);
    let hit: number = 0;
    switch (2) { case (1, 2): hit = 1; break; default: hit = 2; }
    check("gc9", hit === 1);
    // 'typeof x--' is typeof over an update expression, not a bare name plus '--'.
    let t: number = 3;
    check("gc10", (typeof t--) === "number" && t === 2);
}

// ===== SECTION 30: private names through an optional chain =====
class S30 {
    #foo: number = 5;
    #bar: number = 7;
    read(): number | undefined { return this?.#foo; }
    both(): number | undefined { return this?.#foo + this.#bar; }
}
function s30(): void {
    const s: S30 = new S30();
    check("opc1", s.read() === 5);
    check("opc2", s.both() === 12);
}

// ===== SECTION 31: BigInt =====
// BigInt is an arbitrary-precision integer TYPE of its own ('bigint'), not a wide
// number: 2n ** 100n is exact, and mixing it with a number in an arithmetic or
// bitwise operator is a TypeError. Assertions marked 'as any' are the ones tsc
// rejects statically for exactly that reason - the RUNTIME rule is what is under
// test here, so the operand is widened the way SECTION 22 already does.
function s31(): void {
    // The literal forms and the type.
    check("big1", typeof 10n === "bigint");
    check("big2", String(10n) === "10" && "" + 10n === "10" && `${10n}` === "10");
    check("big3", 0xffn === 255n && 0b1010n === 10n && 0o17n === 15n && 1_000n === 1000n);
    // Arbitrary precision: every one of these is past what a double holds exactly.
    check("big4", String(2n ** 64n) === "18446744073709551616");
    check("big5", String(2n ** 100n) === "1267650600228229401496703205376");
    check("big6", String(9007199254740993n) === "9007199254740993");
    check("big7", 9007199254740993n !== 9007199254740992n);
    check("big8", String(100n * 100n * 100n * 100n * 100n * 100n * 100n * 100n * 100n * 100n) === "100000000000000000000");
    check("big9", String((2n ** 100n) % 1000000007n) === "976371285");
    check("big10", String(-(2n ** 100n) / 7n) === "-181092942889747057356671886482");
    // Division truncates toward zero; the remainder keeps the dividend's sign.
    check("big11", 7n / 2n === 3n && -7n / 2n === -3n);
    check("big12", 7n % 3n === 1n && -7n % 3n === -1n);
    // Equality is strict about the type, loose equality crosses it mathematically.
    check("big13", 10n === 10n && !((10n as any) === (10 as any)));
    check("big14", (10n as any) == 10 && (10n as any) == "10" && !((10n as any) == "10.0"));
    check("big15", 10n < 11 && 10n <= 10 && !(10n > 11) && 10n >= 10n);
    check("big16", 2n ** 64n > 2n ** 63n);
    // Bitwise, on the infinite two's complement bit string.
    check("big17", (1n & 3n) === 1n && (1n | 2n) === 3n && (5n ^ 3n) === 6n);
    check("big18", (1n << 10n) === 1024n && (1024n >> 3n) === 128n && (-1n >> 1n) === -1n);
    check("big19", ~1n === -2n && -(-5n) === 5n && (5n - 8n) === -3n);
    // 0n is the one falsy BigInt.
    check("big20", !0n && !(!1n) && (0n ? false : true) && (1n ? true : false));
    check("big21", ((0n as any) || "zero") === "zero" && ((1n as any) && "one") === "one");
    // Compound assignment and the steppers stay in the type.
    check("big22", (function (): bigint { let x: bigint = 5n; x += 3n; x *= 2n; x -= 1n; x **= 2n; return x; })() === 225n);
    check("big23", (function (): bigint { let y: bigint = 1n; y++; ++y; y--; return y; })() === 2n);
    check("big24", (function (): bigint { let c: bigint = 0n; while (c < 3n) { c++; } return c; })() === 3n);
    // Conversions in both directions.
    check("big25", BigInt(42) === 42n && BigInt("0x1f") === 31n && BigInt(true) === 1n);
    check("big26", Number(3n) === 3 && typeof Number(3n) === "number");
    check("big27", (255n).toString(16) === "ff" && (255n).toString() === "255");
    check("big28", (2n ** 64n - 1n).toString(16) === "ffffffffffffffff");
    check("big29", (2n ** 64n).toString(2).length === 65);
    // A BigInt is a PRIMITIVE: it joins and concatenates as its digits.
    check("big30", [1n, 2n].join(",") === "1,2" && ([1n] as any) + "" === "1");
    // ToPrimitive still runs FIRST, so an object operand reaches its primitive form
    // before the BigInt rules (and before the mixed-operand TypeError) see it.
    check("big31", (1n as any) == [1n] && (1n as any) < [2n] && ((1n as any) + []) === "1");
    check("big32", (-255n).toString(16) === "-ff" && BigInt("") === 0n && BigInt(" 12 ") === 12n);
    check("big33", (0n ** 0n) === 1n && ((-2n) ** 3n) === -8n && ((-8n) % (-3n)) === -2n);
    // The relational operators are ToNumber on the other side, not ToBigInt, so null
    // is 0, an unparsable string is NaN (every relation false) and 1.5 really orders.
    check("big34", (1n as any) > null && (1n as any) <= true && !((2n as any) > "abc")
        && !((1n as any) < undefined) && !((1n as any) < "") && 2n > 1.5);
}

// ===== END SECTIONS =====

// ===== SECTION 32: shift and logical assignment through an expression member =====
// '[o][0].a >>= 1' assigns through a member of an arbitrary expression, which only
// LhsAssign can express. Two things were wrong there. The forward scan that decides
// whether an assignment operator follows at all excluded EVERY '=' preceded by '<' or
// '>', so it took the '=' of '<<=' / '>>=' / '>>>=' for the tail of a '<=' / '>=' and
// never entered the production. And the three LOGICAL assignments were folded through the
// compound-operator path, which evaluates the right side unconditionally - they SHORT
// CIRCUIT. Ground truth from node 24 via new vm.Script (not `node --check`).
function s32(): void {
    const o: any = { a: 8 };
    [o][0].a >>= 1;
    check("sa1", o.a === 4);
    [o][0].a <<= 3;
    check("sa2", o.a === 32);
    [o][0].a >>>= 2;
    check("sa3", o.a === 8);
    const p: any = { b: 1 };
    [p][0].b ||= 7;
    check("sa4", p.b === 1);
    const q: any = { c: 0 };
    [q][0].c ||= 5;
    check("sa5", q.c === 5);
    [q][0].c &&= 9;
    check("sa6", q.c === 9);
    [q][0].c ??= 11;
    check("sa7", q.c === 9);
    // The short circuit is observable: the right side must not run at all.
    let n: number = 0;
    const z: any = { d: 3 };
    [z][0].d ||= (n = 1, 99);
    check("sa8", z.d === 3 && n === 0);
    const w: any = { e: 0 };
    [w][0].e &&= (n = 2, 99);
    check("sa9", w.e === 0 && n === 0);
    const y: any = { f: 5 };
    [y][0].f ??= (n = 3, 99);
    check("sa10", y.f === 5 && n === 0);
}

// ===== SECTION 33: delete, key order, and template-literal ToString =====
// Two defects that were invisible to every suite in the repo until this section
// existed, both measured against node v24 and both fixed in all three engines.
// TypeScript shares js's layer 2 and carries its own makeTemplate, so it needed both
// fixes and gets its own copy of the ratchet.
//
// DELETE. The C floor had no js_del at all, so layer 2 blanked the slot and kept a
// side table of what it had blanked. `delete` + `in` + Object.keys were right; key
// ORDER after a delete-then-reinsert, for-in and object spread were not, and the
// table was quadratic and leaked. The floor exports js_del now.
//
// TEMPLATE LITERAL. `${v}` is ToString(v) - ToPrimitive with hint STRING - so an
// object's own `toString` runs, an object with only a `valueOf` still spells
// "[object Object]", and a BigInt spells its digits. Both halves route every
// interpolated PART through it now instead of concatenating the raw values.
function s33(): void {
    const o: any = { a: 1, b: 2, c: 3 };
    check("del1", (delete o.b) === true && o.b === undefined);
    check("del2", Object.keys(o).join(",") === "a,c");
    // A key that comes back is a NEW key: it lands at the end of the order.
    o.b = 9;
    check("del3", Object.keys(o).join(",") === "a,c,b");
    let fi: string = "";
    for (const k in o) { fi = fi + k; }
    check("del4", fi === "acb");
    check("del5", keyStr33({ ...o }) === "a:1;c:3;b:9;");
    check("del6", keyStr33(Object.assign({}, o)) === "a:1;c:3;b:9;");
    const d: any = { x: 1, y: 2 };
    delete d.x;
    check("del7", !("x" in d) && ("y" in d) && Object.keys(d).length === 1);
    check("del8", keyStr33({ ...d }) === "y:2;");
    // Deleting what is not there answers true, exactly like deleting what is.
    check("del9", (delete d.nope) === true && (delete d.y) === true && Object.keys(d).length === 0);
    // On an ARRAY the length is unchanged and the slot reads as undefined.
    const arr: any = [1, 2, 3];
    check("del10", (delete arr[1]) === true && arr.length === 3 && arr[1] === undefined);
    // The shape that was quadratic, and one that checks the surviving ORDER.
    const big: any = {};
    for (let i: number = 0; i < 300; i++) { big["k" + i] = i; delete big["k" + i]; }
    check("del11", Object.keys(big).length === 0);
    for (let j: number = 0; j < 300; j++) { big["m" + j] = j; }
    for (let m: number = 0; m < 300; m = m + 2) { delete big["m" + m]; }
    check("del12", Object.keys(big).length === 150 && Object.keys(big)[0] === "m1");

    check("tpl1", `${{ toString: function (): string { return "T"; } }}` === "T");
    check("tpl2", `${new (class { toString(): string { return "D!"; } })()}` === "D!");
    // valueOf alone is NOT consulted: a template is hint string, not hint default.
    check("tpl3", `${{ valueOf: function (): number { return 7; } }}` === "[object Object]");
    check("tpl4", `${{ valueOf: function (): number { return 7; }, toString: function (): string { return "B"; } }}` === "B");
    check("tpl5", `${10n}` === "10" && `${2n ** 100n}` === "1267650600228229401496703205376");
    check("tpl6", `${1} ${"s"} ${true} ${undefined} ${null} ${({})}` === "1 s true undefined null [object Object]");
    check("tpl7", `${[1, 2]}` === "1,2" && `${[]}` === "");
    const t: any = { toString: function (): string { return "T"; } };
    check("tpl8", `a${t}b${t}c` === "aTbTc");

    // The implicit global, which is the third floor addition of this change.
    // A plain `=` to a name that is on NO scope of the chain creates the binding in
    // the ROOT scope (js_scope_set_or_create), so it outlives the function that
    // wrote it. js_scope_typeof cannot be used to build this in an emitter - it
    // answers "undefined" for an absent name and for a slot HOLDING undefined
    // alike - which is why the C floor now carries the walk itself.
    check("ig1", mkGlobal33() === 100);
    check("ig2", implicitGlobal33 === 100);
    implicitGlobal33 = implicitGlobal33 + 1;
    check("ig3", bumpGlobal33() === 102 && implicitGlobal33 === 102);
    // An assignment that DOES find the name on the chain must not shadow it in the
    // root: this is the arm js_pyset_var and js_scope_set_or_create agree on.
    let outer: number = 1;
    function inner33(): void { outer = 42; }
    inner33();
    check("ig4", outer === 42 && typeof outer === "number");
}
function mkGlobal33(): number {
    implicitGlobal33 = 100;
    return implicitGlobal33;
}
function bumpGlobal33(): number {
    implicitGlobal33 = implicitGlobal33 + 1;
    return implicitGlobal33;
}
function keyStr33(o: any): string {
    const ks: any = Object.keys(o);
    let s: string = "";
    for (let i: number = 0; i < ks.length; i++) { s = s + ks[i] + ":" + o[ks[i]] + ";"; }
    return s;
}

// ===== SECTION 34: an iterator is CLOSED on an early exit =====
// Making for-of lazy (SECTION 33's python twin, docs/todo.md 1.6) left a
// generator SUSPENDED when the loop left early: a second loop over it RESUMED
// where node closes it, and a `finally` around the yield never ran. It could not
// be fixed in the emitter alone - the floor's generator cell answered `next`
// alone - so runtime.c grew gen_close (a GEN_EXIT sentinel thrown INTO the
// suspended body, so its finally clauses unwind normally), abnf/jsrt.go grew the
// matching closeBody, and makeForOf now calls the iterator's `return()` on break.
//
// WHAT THE TWO HALVES AGREE ON, and what they cannot: this half's generators
// REPLAY, so the body's finally has already run once per next() and .return()
// only sets the done flag. Every assertion below is therefore about the CLOSED
// STATE (a second loop yields nothing, next() answers done) and about a count
// taken after exactly ONE step, where replay's one finally and the compiler's
// one close-time finally give the same number by different routes. The ORDER of
// the print is a halves divergence, stated in typescript-interpreter.abnf's :description
// and deliberately not asserted.
function s34(): void {
    var fins = 0
    function* g34() {
        try { yield 1; yield 2; yield 3 } finally { fins = fins + 1 }
    }
    // break: the loop closes the generator, so a second loop over it is empty.
    var a = g34()
    var seen = []
    for (var x of a) { seen.push(x); break }
    var again = []
    for (var y of a) { again.push(y) }
    check("itc1", seen.length === 1 && seen[0] === 1)
    check("itc2", again.length === 0)
    check("itc3", a.next().done === true && a.next().value === undefined)
    check("itc4", fins === 1)
    // Exhausting the loop normally leaves it done as well. No finally count here:
    // replay runs one per next(), so the two halves reach four and one.
    var b = g34()
    var total = 0
    for (var z of b) { total = total + z }
    check("itc5", total === 6 && b.next().done === true)
    // continue does NOT close; the break after it does.
    var c = g34()
    var got = []
    for (var w of c) { if (w === 1) { continue } got.push(w); break }
    check("itc6", got.length === 1 && got[0] === 2)
    var c2 = []
    for (var w2 of c) { c2.push(w2) }
    check("itc7", c2.length === 0)
    // A break out of a TRY inside the loop body: the break becomes a control
    // signal that excDispatch re-issues, and it lands on the closing block too.
    var d = g34()
    var dfin = 0
    for (var v of d) { try { break } finally { dfin = dfin + 1 } }
    check("itc8", dfin === 1 && d.next().done === true)
    // A LABELED break to THIS loop closes it (bindLabelBrk); the label's exit is
    // reached through the loop's own closing block.
    var e = g34()
    lbl34: for (var u of e) { break lbl34 }
    check("itc9", e.next().done === true)
    // g.return(v) is the same close, spelled by hand, and answers {value, done}.
    var f = g34()
    f.next()
    var r = f.return(9)
    check("itc10", r.value === 9 && r.done === true)
    check("itc11", f.next().done === true)
    // ... on a generator that never started (nothing to unwind) ...
    var h = g34()
    var rh = h.return(5)
    check("itc12", rh.value === 5 && rh.done === true && h.next().done === true)
    // ... and on one that already finished.
    var i2 = g34()
    for (var q of i2) { }
    var ri = i2.return(7)
    check("itc13", ri.value === 7 && ri.done === true)
    // A hand-written iterator gets its own return() called, receiver and all.
    var closed = 0
    var iter = {
        n: 0,
        next: function () { this.n = this.n + 1; return { value: this.n, done: this.n > 5 } },
        return: function () { closed = closed + 1; return { value: undefined, done: true } }
    }
    for (var p of iter) { break }
    check("itc14", closed === 1)
    // An iterator WITHOUT a return member is left alone rather than aborting,
    // and an array (the index arm) never looks for one.
    var bare = { n: 0, next: function () { this.n = this.n + 1; return { value: this.n, done: this.n > 3 } } }
    var bn = 0
    for (var s of bare) { bn = s; break }
    check("itc15", bn === 1)
    var asum = 0
    for (var t of [1, 2, 3]) { asum = asum + t; break }
    check("itc16", asum === 1)
    // Destructuring from a generator, then breaking out of it.
    var g2 = g34()
    var pairs = []
    function* pg34() { yield [1, 2]; yield [3, 4] }
    for (var [pa, pb] of pg34()) { pairs.push(pa + pb); break }
    check("itc17", pairs.length === 1 && pairs[0] === 3)
    // A RETURN out of the loop body and a LABELED BREAK to an OUTER statement now
    // close it, exactly as node does. Neither reaches a block makeForOf owns - one
    // rets the frame, the other branches to the outer label's exit - so the compiler
    // halves emit js_jsiterclose on the loop's own iterable handle, which the loop's
    // entry block dominates, while the interpreter half reads the body's completion
    // record. docs/todo.md 1.8.
    var rg = g34()
    check("itc18", takeOne34(rg) === 1)
    check("itc19", rg.next().done === true)
    var tg = g34()
    try { for (var tv of tg) { throw "x" } } catch (te) { }
    // A THROW through the loop body still does NOT close it, in both halves. The
    // only route an emitter has is a runtime open-iterator stack unwound at each
    // try, and `for await` suspends INSIDE its own for-of - so a suspended body
    // leaves an entry on that stack while unrelated frames run, and the unwind
    // closes the wrong loop. That was BUILT AND MEASURED: it made SECTION 35's
    // `for await` see one element instead of three, because SECTION 16's
    // `try { await Promise.reject(r) } catch` unwound past it. Pinned here.
    check("itc20", tg.next().done === false)
    // A labeled break to an OUTER statement, which leaves the loop without ever
    // reaching its own closing block.
    var lg = g34()
    out37t: for (var oi = 0; oi < 1; oi++) {
        for (var lv of lg) { break out37t }
    }
    check("itc21", lg.next().done === true)
    // A return out of NESTED for-of loops closes both, innermost first.
    var n1 = g34()
    var n2 = g34()
    check("itc22", nested37t(n1, n2) === 2)
    check("itc23", n1.next().done === true && n2.next().done === true)
    // The same gap when the throw leaves the loop by leaving the FUNCTION - the
    // try that catches it has no for-of of its own. Pinned with itc20.
    var fg = g34()
    try { throwOut37t(fg) } catch (fe) { }
    check("itc24", fg.next().done === false)
    // A return out of a try INSIDE the loop body: the return becomes a control
    // signal, so the close happens where excDispatch re-issues it.
    var sg = g34()
    check("itc25", retThroughTry37t(sg) === 1)
    check("itc26", sg.next().done === true)
    // ... and a plain loop with a try in it that does NOT leave still runs to the
    // end, so the unwind bookkeeping cannot close an iterator that is still live.
    var qg = g34()
    var qsum = 0
    for (var qv of qg) { try { qsum = qsum + qv } finally { qsum = qsum + 0 } }
    check("itc27", qsum === 6 && qg.next().done === true)
}
function nested37t(p: any, q: any): number { for (var a of p) { for (var b of q) { return a + b } } return -1 }
function throwOut37t(g: any): number { for (var x of g) { throw "z" } return 0 }
function retThroughTry37t(g: any): number { for (var x of g) { try { return x } finally { } } return 0 }
function takeOne34(g: any): number { for (var x of g) { return x } return 0 }

// ===== SECTION 35: for await, and async generators =====
// docs/todo.md 1.7. An async generator body carries its yields AND its awaits on
// ONE suspension channel; they are told apart by a marker record the emitter puts
// on every awaited operand (js_jsawaitmark), which is why the generator model
// needed nothing added to it and the C floor is not touched. `for await` drives an
// async iterator by awaiting each next(), and an array or a string by awaiting each
// ELEMENT - which is what the specification's async-from-sync iterator does.
//
// Every trace below is built in LOCALS. This half drives an async body by REPLAY
// (see the generator note at the top of the file), so a body that writes to shared
// state before its last suspension repeats that write; state recreated per replay
// is safe, and only that is used. Every value asserted here is byte-identical to
// node v24, including the interleaving in "ord".
function s35(): number {
    async function* nums(): any { yield 1; yield 2; yield 3 }
    async function* mixed(): any {
        var a = await Promise.resolve(10)
        yield a
        yield a + 1
        var b = await Promise.resolve(20)
        yield b
        return "r"
    }
    var agExpr = async function* (): any { yield "e" }
    var agObj = { async *m(): any { yield "o" } }
    class AGC { async *m(): any { yield "c" } }
    check("fa1", typeof nums === "function")
    check("fa2", typeof agExpr === "function")
    check("fa3", typeof agObj.m === "function")
    // A class method is not an own property of the instance in this value model
    // (it lives on the __class descriptor), so `typeof inst.m` is undefined in both
    // halves - a documented limitation, unrelated to this section. The async
    // generator OBJECT the call answers is what this asserts.
    check("fa4", typeof new AGC().m().next === "function")
    check("fa5", typeof nums().next === "function")
    // for await over an async generator.
    async function collect(): any {
        var out = []
        for await (var n of nums()) { out.push(n) }
        return out.join("")
    }
    // ... one whose body AWAITS between its yields, which is the case the marker
    // record exists for.
    async function awaitInGen(): any {
        var out = []
        for await (var v of mixed()) { out.push(v) }
        return out.join("")
    }
    // next() by hand: each call answers a PROMISE for a {value, done} record.
    async function manual(): any {
        var g = nums()
        var r1 = await g.next()
        var r2 = await g.next()
        var r3 = await g.next()
        var r4 = await g.next()
        return "" + r1.value + r2.value + r3.value + r4.done + r4.value
    }
    // A break out of a for await CLOSES the async generator.
    async function closeEarly(): any {
        var g = nums()
        for await (var x of g) { break }
        var a = await g.next()
        return "" + a.done + a.value
    }
    // for await over an ARRAY awaits each element, so a promise element arrives
    // resolved; over a SYNC generator it drives next() as an ordinary for-of does.
    async function overArray(): any {
        var out = []
        for await (var p of [Promise.resolve("a"), "b"]) { out.push(p) }
        return out.join("")
    }
    async function overSyncGen(): any {
        var out = []
        for await (var s of sg35()) { out.push(s) }
        return out.join("")
    }
    // ag.return(v) closes it and answers {value: v, done: true}.
    async function agReturn(): any {
        var g = nums()
        await g.next()
        var r = await g.return(9)
        var after = await g.next()
        return "" + r.value + r.done + after.done
    }
    // The expression, object-literal-method and class-method spellings all run.
    async function methodGen(): any {
        var out = []
        for await (var m of agObj.m()) { out.push(m) }
        for await (var c of new AGC().m()) { out.push(c) }
        for await (var e of agExpr()) { out.push(e) }
        return out.join("")
    }
    async function all(): any {
        var parts = []
        parts.push(await collect())
        parts.push(await awaitInGen())
        parts.push(await manual())
        parts.push(await closeEarly())
        parts.push(await overArray())
        parts.push(await overSyncGen())
        parts.push(await agReturn())
        parts.push(await methodGen())
        return parts.join("|")
    }
    all().then(function (r: any) {
        var ok = r === "123|101120|123trueundefined|trueundefined|ab|12|9truetrue|oce"
        check("fa6", ok)
        // main() has already returned by the time a job runs, so a failure here
        // cannot reach the count it reported - exit() is the only way it reaches the
        // EXIT CODE, which is what clang-check and native-full read.
        if (!ok) { exit(1) }
    })
    // ORDERING is the evidence, and it is pushed from .then callbacks only - never
    // from inside a replayed body. An async generator's first next() costs exactly
    // one more tick than a bare Promise.resolve().then, so the two chains interleave
    // p1, n1, p2, n2, p3 - node's own answer.
    var olog = []
    function tick(tag: string): any { return function (v: any): any { olog.push(tag); return v } }
    nums().next().then(tick("n1")).then(tick("n2"))
    Promise.resolve().then(tick("p1")).then(tick("p2")).then(tick("p3")).then(function (): any {
        var ordOk = olog.join(",") === "p1,n1,p2,n2,p3"
        check("fa7", ordOk)
        if (!ordOk) { exit(1) }
    })
    // ----- docs/todo.md 1.8: yield* inside an async generator, and ag.throw() -----
    // Every value below is byte-identical to node v24, measured four ways (the
    // interpreter half, llvm.Run, a native -exe binary, and node). Each trace is built
    // in a LOCAL created per call, so replay repeats no observable write.
    //
    // At ae0c62b these were the two residues. `yield* someAsyncGen()` did not merely
    // delegate synchronously: an async generator's next() answers a PROMISE, whose
    // `done` is undefined, so the unawaited drive never terminated - the interpreter
    // half HUNG and llvm.Run died on the step limit. And ag.throw(e) closed the body
    // and rejected instead of raising at the yield.
    function rec38(r: any): any { return "" + r.value + "/" + r.done }
    var agRan38 = 0 // Counts entries into never38's body; a suspendedStart throw must not.
    // yield* over an ASYNC delegate: each step awaited, and the value of the
    // expression is the delegate's RETURN value.
    async function* inner38(): any { yield 1; await null; yield 2; return 9 }
    async function* outer38(): any {
        var t = []
        t.push("pre")
        var r = yield* inner38()
        t.push("ret=" + r)
        yield t.join(",")
    }
    async function ystarAsync(): any {
        var out = []
        for await (var v of outer38()) { out.push(v) }
        return out.join("|")
    }
    // yield* over a SYNC generator, and over an ARRAY of promises, from inside an
    // async generator: the elements arrive resolved and in order.
    async function* ystarSyncGen38(): any { yield* sg35(); yield 3 }
    async function* ystarArr38(): any { yield* [Promise.resolve(4), 5] }
    async function ystarOther(): any {
        var out = []
        for await (var a of ystarSyncGen38()) { out.push(a) }
        for await (var b of ystarArr38()) { out.push(b) }
        return out.join("")
    }
    // ag.throw(e) RAISES AT THE SUSPENDED YIELD, so the try around it catches, the
    // catch's own yield answers the throw() request, and the finally runs on the
    // next resume.
    async function* caught38(): any {
        var t = []
        try {
            t.push("y10")
            yield 10
            t.push("unreached")
        } catch (e) {
            t.push("caught:" + e)
            yield t.join(",")
        } finally {
            t.push("fin")
        }
    }
    async function agThrowCaught(): any {
        var g = caught38()
        var r1 = await g.next()
        var r2 = await g.throw("boom")
        var r3 = await g.next()
        return rec38(r1) + "|" + rec38(r2) + "|" + rec38(r3)
    }
    // Nothing catches: the request REJECTS with the thrown value and the generator is
    // done - the abrupt completion node ends up with too.
    async function* bare38(): any { yield 20 }
    async function agThrowUncaught(): any {
        var g = bare38()
        var r1 = await g.next()
        var s = "?"
        try { await g.throw("bang") } catch (e) { s = "rej:" + e }
        var r3 = await g.next()
        return rec38(r1) + "|" + s + "|" + rec38(r3)
    }
    // A throw at suspendedStart does NOT enter the body (agStarted below asserts it).
    async function* never38(): any { agRan38 = agRan38 + 1; yield 30 }
    async function agThrowBeforeStart(): any {
        var g = never38()
        var s = "?"
        try { await g.throw("early") } catch (e) { s = "rej:" + e }
        var r = await g.next()
        return s + "|" + rec38(r) + "|ran=" + agRan38
    }
    // docs/todo.md 1.6: a throw that arrives while the body is parked at a `yield*` is
    // FORWARDED to the delegate's throw(). The delegate catches, the throw() request
    // RESOLVES with the value the delegate's catch yields, and the delegate is left
    // where it can be resumed rather than suspended forever. Until this it raised AT
    // the yield*, so delegOuter38's own (absent) handler saw it, the request rejected,
    // and 40/false|rej:t was pinned here as the gap. All four fields below are node
    // v24's, measured by running this section under it.
    async function* deleg38(): any { try { yield 40 } catch (e) { yield "d:" + e } }
    async function* delegOuter38(): any { yield* deleg38(); yield 41 }
    async function agThrowIntoDelegate(): any {
        var g = delegOuter38()
        var r1 = await g.next()
        var s = "?"
        try { s = rec38(await g.throw("t")) } catch (e) { s = "rej:" + e }
        var r3 = await g.next()
        return rec38(r1) + "|" + s + "|" + rec38(r3)
    }
    // A delegate with NO throw method - an array here, and equally a string or a plain
    // iterator object - is CLOSED through its return() and the yield* raises node's
    // TypeError, which the outer body's own try then catches. The message is asserted
    // verbatim because it is the whole observable difference from a forwarded throw.
    async function* arrDeleg38(): any { try { yield* [50, 51] } catch (e) { yield "e:" + e } }
    async function agThrowIntoArray(): any {
        var g = arrDeleg38()
        var r1 = await g.next()
        var s = "?"
        try { s = rec38(await g.throw("t2")) } catch (e) { s = "rej:" + e }
        return rec38(r1) + "|" + s
    }
    // The other half of the same loop: g.next(v) forwards v to the DELEGATE (the
    // specification's `received.value`). Before 1.6 the delegate's next() was called
    // with no argument at all, so `a` below read undefined in every engine.
    async function* sendInner38(): any { var a = yield 60; yield "a=" + a; return "R" }
    async function* sendOuter38(): any { var r = yield* sendInner38(); yield "r=" + r }
    async function agSendThroughDelegate(): any {
        var g = sendOuter38()
        var r1 = await g.next()
        var r2 = await g.next("s")
        var r3 = await g.next()
        var r4 = await g.next()
        return rec38(r1) + "|" + rec38(r2) + "|" + rec38(r3) + "|" + rec38(r4)
    }
    async function all38b(): any {
        var parts = []
        parts.push(await ystarAsync())
        parts.push(await ystarOther())
        parts.push(await agThrowCaught())
        parts.push(await agThrowUncaught())
        parts.push(await agThrowBeforeStart())
        parts.push(await agThrowIntoDelegate())
        parts.push(await agThrowIntoArray())
        parts.push(await agSendThroughDelegate())
        return parts.join("~")
    }
    all38b().then(function (r: any): any {
        var ok = r === "1|2|pre,ret=9~12345~10/false|y10,caught:boom/false|undefined/true~" +
                       "20/false|rej:bang|undefined/true~rej:early|undefined/true|ran=0~" +
                       "40/false|d:t/false|41/false~" +
                       "50/false|e:TypeError: The iterator does not provide a 'throw' method./false~" +
                       "60/false|a=s/false|r=R/false|undefined/true"
        check("fa8", ok)
        if (!ok) { exit(1) }
    }, function (e: any): any {
        check("fa8", false)
        exit(1)
    })
    // A PLAIN generator's yield* answers the delegate's return value too - the same
    // one line in the emitter, and synchronous, so it is asserted directly. next(v)
    // reaches the delegate here as well: a plain generator has no throw-in channel and
    // no .throw() at all, but the sent value goes through the identical loop. The sent
    // value is YIELDED BACK rather than pushed to ysync: this half replays a generator
    // body once per next(), so a write from inside the DELEGATE would repeat. ysync is
    // written once, after the delegate is done - only one replay reaches it.
    var ysync = []
    function* ysInner38(): any { var a = yield 1; yield "a=" + a; return "r" }
    function* ysOuter38(): any { ysync.push("v=" + (yield* ysInner38())) }
    var yg38 = ysOuter38()
    var ys1 = yg38.next().value
    var ys2 = yg38.next("S").value
    yg38.next()
    check("fa9", ys1 === 1 && ys2 === "a=S" && ysync.join(",") === "v=r")
    return 0
}
function* sg35(): any { yield 1; yield 2 }

// ===== SECTION 36: WHEN a generator's finally runs, and closing a yield* delegate =====
// docs/todo.md 1.10 and the js half of 2.6. Two defects, both about the MOMENT a
// finally runs, and neither visible to SECTION 34 - every assertion there is about
// the closed STATE or about a count, which both halves reached by different routes.
//
// THE INTERPRETER RAN THE FINALLY AT THE FIRST next(). A generator suspends there by
// throwing a host-level signal, and interp-core's excTry ran the program's `finally`
// for any unwinding - so `try { yield } finally { f() }` printed f before the first
// next() had even returned, and again at every step after. That is not replay
// repetition, it is the wrong ORDER, and it was a live halves divergence: the two
// compiled halves have real coroutines and were right. typescript-interpreter.abnf now has
// its own jsExcTry that lets the suspension through untouched.
//
// THE COMPILER HALVES DID NOT CLOSE A yield* DELEGATE. g.return() is gen_close, which
// throws the GEN_EXIT sentinel at the suspended yield; the outer body's finally ran
// and the DELEGATE was left suspended, where node runs its finally first. The emitted
// yield* loop now yields inside a js_try whose finally runs IteratorClose on the
// delegate (typescript-to-llvm-ir.abnf's emitYieldStarClose), and the interpreter half raises
// the same close at the same yield (jpYieldDeleg).
//
// Every log below is written only in a `finally` or from the DRIVER, never before the
// yield the body parks at - a write before that yield repeats once per replay in the
// interpreter half and the two halves would disagree about the count, not the order.
// Measured byte-for-byte against node v24.
function s36(): number {
    var lg = []
    function* g36(): any { try { yield 1; yield 2 } finally { lg.push("fin") } }
    var a36 = g36()
    lg.push("n1=" + a36.next().value)
    lg.push("n2=" + a36.next().value)
    lg.push("ret")
    var r36 = a36.return(0)
    lg.push("end")
    check("gcl1", lg.join(",") === "n1=1,n2=2,ret,fin,end")
    check("gcl2", r36.value === 0 && r36.done === true && a36.next().done === true)
    // The delegate is closed BEFORE the delegating body's own finally.
    var l2 = []
    function* in36(): any { try { yield 1; yield 2 } finally { l2.push("inner") } }
    function* out36(): any { try { yield* in36() } finally { l2.push("outer") } }
    var b36 = out36()
    l2.push("n=" + b36.next().value)
    b36.return(0)
    check("gcl3", l2.join(",") === "n=1,inner,outer")
    // Nested yield*: innermost first, one close per level.
    var l3 = []
    function* i3_36(): any { try { yield 1 } finally { l3.push("i") } }
    function* m3_36(): any { try { yield* i3_36() } finally { l3.push("m") } }
    function* o3_36(): any { try { yield* m3_36() } finally { l3.push("o") } }
    var c36 = o3_36()
    c36.next()
    c36.return(0)
    check("gcl4", l3.join(",") === "i,m,o")
    // A for-of that BREAKS out of a delegating generator closes the delegate too -
    // the loop calls return(), which is the same close.
    var l4 = []
    function* i4_36(): any { try { yield 1; yield 2 } finally { l4.push("i4") } }
    function* o4_36(): any { yield* i4_36() }
    for (var v36 of o4_36()) { l4.push("v" + v36); break }
    check("gcl5", l4.join(",") === "v1,i4")
    // An ARRAY delegate has no return member: nothing is called and nothing aborts.
    var l5 = []
    function* o5_36(): any { try { yield* [7, 8] } finally { l5.push("o5") } }
    var e36 = o5_36()
    check("gcl6", e36.next().value === 7)
    e36.return(0)
    check("gcl7", l5.join(",") === "o5")
    // A close is a RETURN completion: `catch` is not offered it, in any of the three
    // engines (runtime.c's GEN_EXIT guard, abnf/jsrt.go's genExit, jsExcTry here).
    var l6 = []
    function* g6_36(): any { try { yield 1 } catch (e6) { l6.push("caught") } finally { l6.push("f6") } }
    var f36 = g6_36()
    f36.next()
    f36.return(0)
    check("gcl8", l6.join(",") === "f6")
    // A generator that never started has no suspension point, so nothing unwinds.
    var l7 = []
    function* g7_36(): any { try { yield 1 } finally { l7.push("f7") } }
    var h36 = g7_36()
    h36.return(3)
    check("gcl9", l7.length === 0 && h36.next().done === true)
    // A body that runs to its own end runs the finally once, at the end.
    var l8 = []
    function* g8_36(): any { try { yield 1 } finally { l8.push("f8") } }
    var i36 = g8_36()
    l8.push("a" + i36.next().value)
    l8.push("d" + i36.next().done)
    check("gcl10", l8.join(",") === "a1,f8,dtrue")
    // Nested try/finally around ONE yield unwinds inside out.
    var l9 = []
    function* g9_36(): any {
        try { try { yield 1 } finally { l9.push("in") } } finally { l9.push("out") }
    }
    var j36 = g9_36()
    j36.next()
    j36.return(0)
    check("gcl11", l9.join(",") === "in,out")
    // g.throw(v) on a PLAIN generator (docs/todo.md 1.4). Here as well as in the
    // features file because the features file is never BUILT NATIVELY, and the
    // yield*-forwarding arm is emitter code that clang-check and native-full are the
    // only gates that reach. Every push is in the driver or in a finally: a write
    // before the parked yield repeats once per replay in the interpreter half.
    var t1 = []
    function* gt1_39() {
        try { yield 1; yield 2 } catch (e) { yield "c:" + e } finally { t1.push("ft1") }
    }
    var a39 = gt1_39()
    t1.push("n" + a39.next().value)
    t1.push("t" + a39.throw("X").value)
    t1.push("d" + a39.next().done)
    check("gth1", firsts39(t1).join(",") === "n1,tc:X,ft1,dtrue")
    // Uncaught: the value propagates out of throw() and the generator is left done.
    var t2 = []
    function* gt2_39() { try { yield 1; yield 2 } finally { t2.push("ft2") } }
    var b39 = gt2_39()
    b39.next()
    var t2c = ""
    try { b39.throw("Y") } catch (e) { t2c = e }
    check("gth2", t2c === "Y" && b39.next().done === true && firsts39(t2).join(",") === "ft2")
    // Never started: no suspension point, so NO finally runs at all.
    var t3 = []
    function* gt3_39() { try { yield 1 } finally { t3.push("ft3") } }
    var c39 = gt3_39()
    var t3c = ""
    try { c39.throw("Z") } catch (e) { t3c = e }
    check("gth3", t3c === "Z" && t3.length === 0 && c39.next().done === true)
    // Caught, then RETURN: the record carries the return value and done.
    function* gt4_39() { try { yield 1 } catch (e) { return 7 } }
    var d39 = gt4_39()
    d39.next()
    var d39r = d39.throw("Q")
    check("gth4", d39r.value === 7 && d39r.done === true && d39.next().done === true)
    // At a yield* the throw is FORWARDED to the delegate, so a delegate that would
    // catch it does - and the outer body carries on with what it yields next.
    var t5 = []
    function* it5_39() { try { yield 1 } catch (e) { yield "in:" + e } finally { t5.push("i5") } }
    function* ot5_39() { try { yield* it5_39() } finally { t5.push("o5") } }
    var e5_39 = ot5_39()
    t5.push("n" + e5_39.next().value)
    t5.push("t" + e5_39.throw("W").value)
    t5.push("d" + e5_39.next().done)
    check("gth5", firsts39(t5).join(",") === "n1,tin:W,i5,o5,dtrue")
    // A delegate with NO throw method - an array - is closed through its return()
    // and the yield* raises node's TypeError, which the outer body may catch.
    function* ot6_39() { try { yield* [1, 2, 3] } catch (e) { yield "a:" + e } }
    var f6_39 = ot6_39()
    f6_39.next()
    check("gth6", f6_39.throw("V").value ===
          "a:TypeError: The iterator does not provide a 'throw' method.")
    // A `yield` inside a CATCH ARM is a suspension like any other: the enclosing
    // finally belongs to the close, not to it. Reachable with no throw() at all,
    // and the interpreter halves used to run the finally at the suspension.
    var t7 = []
    function* gt7_39() { try { throw 1 } catch (e) { yield 99 } finally { t7.push("f7c") } }
    var g7c39 = gt7_39()
    t7.push("n" + g7c39.next().value)
    t7.push("d" + g7c39.next().done)
    check("gth7", firsts39(t7).join(",") === "n99,f7c,dtrue")
    return 0
}

// The first occurrence of each entry, in order: the interpreter half REPLAYS a
// generator body once per next(), so a log collapses to node's sequence rather than
// matching it row for row (the price is repetition, not order).
function firsts39(xs) {
    var out = []
    for (var i = 0; i < xs.length; i++) { if (out.indexOf(xs[i]) < 0) { out.push(xs[i]) } }
    return out
}

// ===== SECTION 37: methods as values, Object.prototype, bind, and a catchable BigInt =====
// docs/todo.md 2.5 and 2.6. Everything here ABORTED or answered "undefined" before,
// and the split across the engines is the interesting part - three of the six
// entries below were LIVE HALVES DIVERGENCES that --cross could not see, because no
// test reached them.
//
// A METHOD IS NOT AN OWN PROPERTY. An instance keeps its methods on its __class
// descriptor, so `typeof p.m` read "undefined" in all four engines where node says
// "function", and `"m" in p` was false. getMember / js_jsvget / js_jshas now walk
// the descriptor chain for a VALUE read (the call site still reads raw, so an
// ordinary p.m() allocates nothing).
//
// typeof [].push WAS ENGINE-DEPENDENT: "function" under llvm.Run, which reads a
// plain member through the shared js_get, and "undefined" natively, where the read
// went through js_jsmget's method-name hiding. One extern for the value read fixes
// both directions at once.
//
// Object.prototype.toString.call(v) is the classic type probe and it aborted BOTH
// compiled halves ("member 'call' of undefined" under llvm.Run, "member 'toString'
// of undefined" natively): the `Object` global carried no usable prototype. All
// three engines now build their own, with toString / valueOf / hasOwnProperty.
//
// Function.prototype.bind existed in NO engine, and f.call(o) in the js interpreter
// half silently ignored its receiver while the typescript half had had the shim
// since it was written.
//
// A BIGINT TypeError WAS NOT CATCHABLE. `1n + 1` is a TypeError in node and a
// program may catch it; here it aborted the process. Every BigInt-family raise -
// the mixed-operand TypeError, ToBigInt's SyntaxError, the RangeErrors - is now a
// real throw in all four engines. The operands come out of an ARRAY so the
// constant folder cannot evaluate any of it at compile time.
function s37(): number {
    class S37A { am() { return "a" + this.v } }
    class S37B extends S37A { constructor() { super(); this.v = 1 } bm() { return "b" } }
    var p = new S37B()
    var arr = [1, 2]

    // A method read as a VALUE, and called through that value.
    check("mv1", typeof p.bm === "function")
    check("mv2", typeof p.am === "function")        // inherited, one __super hop
    check("mv3", typeof p.zz === "undefined")
    var bm = p.bm
    var am = p.am
    check("mv4", bm() === "b" && am() === "a1")     // the shim stays bound to p
    check("mv5", typeof arr.push === "function")    // was engine-dependent
    check("mv6", typeof "s".slice === "function")

    // `in` sees a method, and does NOT see the engine's own __ slots.
    check("in1", "bm" in p)
    check("in2", "am" in p)
    check("in3", ("zz" in p) === false)
    check("in4", "v" in p)
    check("in5", ("__class" in p) === false)
    check("in6", (0 in arr) && (("5" in arr) === false))

    // Object.prototype, reached the way real code reaches it.
    var vals = [[1], {}, 3, "s", true, null, undefined]
    var tags = []
    for (var i = 0; i < vals.length; i++) { tags.push(Object.prototype.toString.call(vals[i])) }
    check("op1", tags.join("|") === "[object Array]|[object Object]|[object Number]|" +
                                    "[object String]|[object Boolean]|[object Null]|[object Undefined]")
    check("op2", typeof Object.prototype === "object")
    var own = { k: 1 }
    check("op3", Object.prototype.hasOwnProperty.call(own, "k") === true)
    check("op4", Object.prototype.hasOwnProperty.call(own, "j") === false)
    check("op5", Object.prototype.valueOf.call(own) === own)

    // call / apply / bind, with the receiver actually arriving.
    var recv = { x: 10 }
    var f = function(a, b) { return this.x + a + b }
    check("fn1", f.call(recv, 2, 3) === 15)
    check("fn2", f.apply(recv, [2, 3]) === 15)
    check("fn3", f.bind(recv)(2, 3) === 15)
    check("fn4", f.bind(recv, 2)(3) === 15)
    check("fn5", f.bind(recv, 2, 3)() === 15)

    // new through a constructor that RETURNS a function, and `new` with no argument
    // list: [[Construct]] keeps an explicitly returned object and a function is one.
    // The interpreter half dropped it and then said "unknown class: expression".
    function S37Meta() { return function() { this.q = 8 } }
    check("nw1", (new (new S37Meta())()).q === 8)
    function S37Plain() { this.q = 9 }
    check("nw2", (new S37Plain).q === 9)

    // The BigInt raises are CATCHABLE.
    var ops = [1n, 1, 0n, -1n, "zz", 1.5]
    var log = []
    try { log.push("v" + (ops[0] + ops[1])) } catch (e) { log.push("mix:" + (("" + e).indexOf("Cannot mix BigInt") >= 0)) }
    try { log.push("v" + (ops[0] / ops[2])) } catch (e) { log.push("div:" + (("" + e).indexOf("Division by zero") >= 0)) }
    try { log.push("v" + (ops[0] ** ops[3])) } catch (e) { log.push("pow:" + (("" + e).indexOf("Exponent") >= 0)) }
    try { log.push("v" + BigInt(ops[4])) } catch (e) { log.push("str:" + (("" + e).indexOf("SyntaxError") >= 0)) }
    try { log.push("v" + BigInt(ops[5])) } catch (e) { log.push("flo:" + (("" + e).indexOf("not an integer") >= 0)) }
    try { log.push("v" + (ops[0].toString(99))) } catch (e) { log.push("rdx:" + (("" + e).indexOf("radix") >= 0)) }
    check("bi1", log.join(",") === "mix:true,div:true,pow:true,str:true,flo:true,rdx:true")
    // The program keeps running after a caught one, which is the whole point.
    check("bi2", ops[0] + ops[0] === 2n)
    return 0
}

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
    s21(); // SECTION-CALL 21
    s22(); // SECTION-CALL 22
    s23(); // SECTION-CALL 23
    s24(); // SECTION-CALL 24
    s25(); // SECTION-CALL 25
    s26(); // SECTION-CALL 26
    s27(); // SECTION-CALL 27
    s28(); // SECTION-CALL 28
    s29(); // SECTION-CALL 29
    s30(); // SECTION-CALL 30
    s31(); // SECTION-CALL 31
    s32(); // SECTION-CALL 32
    s33(); // SECTION-CALL 33
    s34(); // SECTION-CALL 34
    s35(); // SECTION-CALL 35
    s36(); // SECTION-CALL 36
    s37(); // SECTION-CALL 37
    println("full: " + checks + " checks, " + failures + " failures");
    return failures;
}
