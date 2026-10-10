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

    // STATIC MEMBERS ARE INHERITED: `class D extends C {}` then `D.s()` walks the
    // class object's own prototype chain in node. Every value below is
    // `new vm.Script(src)`'s. The operands come out of an ARRAY so the constant
    // folder cannot answer any of it at compile time.
    const sA: number[] = [1, 2, 3, 4, 5];
    class SC7 {
        static sm(x: number): string { return "SC.sm:" + (x + sA[0]); }
        static sf: number = sA[1] * 10;
        static get sg(): string { return "SC.sg:" + this.sf; }
        static set sg(v: any) { this.sf = v + sA[3]; }
        static who(): string { return "SC"; }
        static callsWho(): string { return "via " + this.who(); }
    }
    class SD7 extends SC7 {
        static own(x: number): string { return "SD.own:" + (x + sA[4]); }
        static viaSuper(x: number): string { return "SD>" + super.sm(x); }
        static who(): string { return "SD"; }
    }
    class SE7 extends SD7 {}
    class SF7 extends SC7 { static sm(x: number): string { return "SF.sm:" + (x + sA[2]); } }
    // A method, a field and a getter, one and two links up the chain.
    check("cls10", SD7.sm(sA[0]) === "SC.sm:2" && SE7.sm(sA[1]) === "SC.sm:3");
    check("cls11", SD7.sf === 20 && SE7.sf === 20);
    check("cls12", SD7.sg === "SC.sg:20" && SE7.sg === "SC.sg:20");
    check("cls13", typeof SD7.sm === "function" && (SD7 as any).nope === undefined);
    // An OWN static wins over the inherited one, and the parent keeps its own.
    check("cls14", SF7.sm(sA[0]) === "SF.sm:4" && SC7.sm(sA[0]) === "SC.sm:2");
    // `this` inside an inherited static is the class the call named, so an override
    // one link down still wins - the chain is walked from the RECEIVER.
    check("cls15", SD7.callsWho() === "via SD" && SE7.callsWho() === "via SD"
                   && SC7.callsWho() === "via SC");
    // super.staticM() inside a static method: the parent's static is a plain
    // closure, not a receiver-prepended instance method, so the argument must not
    // be shifted by one.
    check("cls16", SD7.viaSuper(sA[3]) === "SD>SC.sm:5");
    // An inherited static SETTER runs with `this` bound to the SUBCLASS, so the
    // write lands there and the parent's own value is untouched.
    SD7.sg = sA[0];
    check("cls17", SD7.sf === 5 && SC7.sf === 20);
    // The engine's own descriptor slots are never inherited.
    check("cls18", SE7.own(sA[0]) === "SD.own:6");
    // A class's own NAME is bound inside its body, and bound BEFORE the static
    // initializers run.
    class SG7 { static f: number = sA[2]; static blk: number; static { SG7.blk = SG7.f + sA[0]; } }
    check("cls19", SG7.blk === 4);
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
    // One shape is deliberately avoided: an async body must have NO side effect
    // before its last await (this half drives one by REPLAY, so a print before an
    // await would repeat). `for await` and `async function*` are SECTION 35's.
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
// the type annotations stripped).
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

    // The Annex B legacy OCTAL string escape. The one-digit form is octal on its own -
    // '"\1"' is U+0001, not the character '1' - and only '\0' with nothing after it is the
    // NUL escape.
    check("oct1", "\1".charCodeAt(0) === 1 && "\1".length === 1);
    check("oct2", "\101" === "A" && "\0".charCodeAt(0) === 0 && "\0".length === 1);
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
// LhsAssign can express. The forward scan that decides whether an assignment operator
// follows at all must take the '=' of '<<=' / '>>=' / '>>>=' as an assignment and not
// as the tail of a '<=' / '>=', and the three LOGICAL assignments SHORT CIRCUIT rather
// than going through the compound-operator path. Ground truth from node 24 via
// new vm.Script (not `node --check`).
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
// TypeScript shares js's layer 2 and carries its own makeTemplate, so it gets its own
// copy of the ratchet.
//
// DELETE removes the key and its key-order entry (the floor's js_del), so key ORDER
// after a delete-then-reinsert, for-in and object spread are node's answers.
//
// TEMPLATE LITERAL. `${v}` is ToString(v) - ToPrimitive with hint STRING - so an
// object's own `toString` runs, an object with only a `valueOf` still spells
// "[object Object]", and a BigInt spells its digits. Both halves route every
// interpolated PART through it instead of concatenating the raw values.
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
    // Many deletions, and one that checks the surviving ORDER.
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

    // The implicit global. A plain `=` to a name that is on NO scope of the chain
    // creates the binding in the ROOT scope (js_scope_set_or_create), so it outlives
    // the function that wrote it. js_scope_typeof cannot be used to build this in an
    // emitter - it answers "undefined" for an absent name and for a slot HOLDING
    // undefined alike - which is why the C floor carries the walk itself.
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
// for-of over a generator is lazy, so a loop that leaves early has to CLOSE the
// generator: a second loop over it is empty, as in node, and a `finally` around the
// yield runs. runtime.c's gen_close throws a GEN_EXIT sentinel INTO the suspended
// body (so its finally clauses unwind normally), abnf/jsrt.go's closeBody does the
// same, and makeForOf calls the iterator's `return()` on break. The moment the
// finally runs is SECTION 36's subject; this one asserts the closed STATE and counts
// taken after exactly ONE step.
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
    // A RETURN out of the loop body and a LABELED BREAK to an OUTER statement close
    // it, exactly as node does. Neither reaches a block makeForOf owns - one rets the
    // frame, the other branches to the outer label's exit - so the compiler halves
    // emit js_jsiterclose on the loop's own iterable handle, which the loop's entry
    // block dominates, while the interpreter half reads the body's completion record.
    var rg = g34()
    check("itc18", takeOne34(rg) === 1)
    check("itc19", rg.next().done === true)
    var tg = g34()
    try { for (var tv of tg) { throw "x" } } catch (te) { }
    // A THROW through the loop body does NOT close it, in both halves: the only route
    // an emitter has is a runtime open-iterator stack unwound at each try, and
    // `for await` suspends INSIDE its own for-of, so such a stack closes the wrong
    // loop (a caught rejection in one async function would close another's suspended
    // `for await`). Pinned here.
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
// An async generator body carries its yields AND its awaits on
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
    // The async generator OBJECT the method call answers is what this asserts.
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
    // ----- yield* inside an async generator, and ag.throw() -----
    // Every value below is node v24's. Each trace is built in a LOCAL created per
    // call, so replay repeats no observable write. `yield* someAsyncGen()` has to
    // AWAIT each step (an async generator's next() answers a promise, whose `done` is
    // undefined), and ag.throw(e) raises at the yield rather than closing the body.
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
    // A throw that arrives while the body is parked at a `yield*` is FORWARDED to the
    // delegate's throw(). The delegate catches, the throw() request RESOLVES with the
    // value the delegate's catch yields, and the delegate is left where it can be
    // resumed rather than suspended forever. All four fields below are node v24's.
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
    // specification's `received.value`), so `a` below reads the sent value.
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
    // reaches the delegate here as well, through the identical loop. The sent value is
    // YIELDED BACK rather than pushed to ysync: this half replays a generator body
    // once per next(), so a write from inside the DELEGATE would repeat. ysync is
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
// When a finally runs, and closing a yield* delegate - neither visible to SECTION
// 34, whose assertions are about the closed STATE or a count.
//
// A generator suspends in the interpreter half by throwing a host-level signal, so
// its jsExcTry lets that signal through untouched; otherwise `try { yield } finally
// { f() }` would print f at every suspension instead of when node runs it.
//
// g.return() is gen_close, which throws the GEN_EXIT sentinel at the suspended
// yield; the emitted yield* loop yields inside a js_try whose finally runs
// IteratorClose on the DELEGATE (typescript-to-llvm-ir.abnf's emitYieldStarClose),
// so the delegate's finally runs first, as in node. The interpreter half raises the
// same close at the same yield (jpYieldDeleg).
//
// Every log below is written only in a `finally` or from the DRIVER, never before the
// yield the body parks at - a write before that yield repeats once per replay in the
// interpreter half. Every log is node v24's.
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
    // g.throw(v) on a PLAIN generator. Here as well as in the
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
    // finally belongs to the close, not to it. Reachable with no throw() at all.
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
// A METHOD IS NOT AN OWN PROPERTY: an instance keeps its methods on its __class
// descriptor, yet `typeof p.m` is "function" and `"m" in p` is true in node.
// getMember / js_jsvget / js_jshas walk the descriptor chain for a VALUE read (the
// call site still reads raw, so an ordinary p.m() allocates nothing).
//
// typeof [].push must be "function" in every engine: llvm.Run reads a plain member
// through the shared js_get, a native build through js_jsmget's method-name hiding,
// and one extern for the value read covers both.
//
// Object.prototype.toString.call(v) is the classic type probe; every engine builds
// its own `Object` prototype with toString / valueOf / hasOwnProperty.
//
// Function.prototype.bind, and f.call(o) / f.apply(o) with the receiver arriving.
//
// A BIGINT TypeError IS CATCHABLE. `1n + 1` is a TypeError in node and a program may
// catch it. Every BigInt-family raise - the mixed-operand TypeError, ToBigInt's
// SyntaxError, the RangeErrors - is a real throw in all four engines. The operands
// come out of an ARRAY so the constant folder cannot evaluate any of it at compile
// time.
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
    // NOT bound: a bare call has `this` undefined (class bodies are strict), so `am`
    // throws a TypeError, and the receiver of a call through the value is the call's.
    var amThrew = false
    try { am() } catch (e) { amThrew = e instanceof TypeError }
    check("mv4", bm() === "b" && amThrew)
    check("mv7", p.am === p.am && p.am === am && am.call(p) === "a1")
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

// ===== SECTION 38: the Error hierarchy =====
// The classes are ordinary {__isclass, __name, __super, __ctor} descriptors, so
// instanceof, `new` and `extends` are the ordinary class machinery.
//
// Every row below is node v24's answer, except s38keys - see the comment there.
function s38(): void {
    var xs = ["boom", 1n, 2, 255, "zz"]

    var te = new TypeError(xs[0])
    check("err1", te.name === "TypeError" && te.message === "boom")
    check("err2", te.toString() === "TypeError: boom" && ("" + te) === "TypeError: boom")
    // stack has no frames here (no engine in this project has a frame walker),
    // but it starts with exactly the line node's does, which is what the
    // `e.stack.split("\n")[0]` idiom reads.
    check("err3", typeof te.stack === "string" && te.stack.split("\n")[0] === "TypeError: boom")
    check("err4", te instanceof TypeError && te instanceof Error)
    check("err5", !(te instanceof RangeError) && !(te instanceof ReferenceError))

    // The five. EvalError and URIError are deliberately absent (see the
    // jsErrClassNames table in the emitters).
    var mk = [new Error(xs[0]), new TypeError(xs[0]), new RangeError(xs[0]),
              new ReferenceError(xs[0]), new SyntaxError(xs[0])]
    var nm = ["Error", "TypeError", "RangeError", "ReferenceError", "SyntaxError"]
    var okn = 0
    var oki = 0
    for (var i = 0; i < mk.length; i++) {
        if (mk[i].name === nm[i] && ("" + mk[i]) === nm[i] + ": boom") { okn = okn + 1 }
        if (mk[i] instanceof Error) { oki = oki + 1 }
    }
    check("err6", okn === 5 && oki === 5)

    // No message: `new Error()` and `new Error(undefined)` both leave "", and
    // toString then answers the bare name.
    var e0 = new Error()
    check("err7", e0.message === "" && e0.toString() === "Error" && e0.name === "Error")

    // A user subclass. name comes from the NEAREST BUILTIN ancestor, because in a
    // real engine it lives on the prototype and E.prototype does not shadow it.
    class S41MyErr extends Error {}
    var m1 = new S41MyErr(xs[0])
    check("err8", m1.message === "boom" && m1.name === "Error" && ("" + m1) === "Error: boom")
    check("err9", m1 instanceof S41MyErr && m1 instanceof Error && !(m1 instanceof TypeError))
    class S41TErr extends TypeError {
        constructor(msg: string) { super(msg); this.code = 7 }
    }
    var m2 = new S41TErr(xs[0])
    check("err10", m2.name === "TypeError" && m2.message === "boom" && m2.code === 7)
    check("err11", m2 instanceof S41TErr && m2 instanceof TypeError && m2 instanceof Error)
    // Two levels, and the message still arrives through BOTH default constructors.
    class S41Deep extends S41TErr {}
    var m3 = new S41Deep(xs[0])
    check("err12", m3.message === "boom" && m3.name === "TypeError" && m3.code === 7)

    // A default DERIVED constructor forwards its arguments: node says 5.
    class S41Base { constructor(a, b) { this.a = a; this.b = b } }
    class S41Sub extends S41Base {}
    var sb = new S41Sub(xs[2], xs[3])
    check("err13", sb.a === 2 && sb.b === 255)

    // Throwing and catching one, and re-throwing through a finally.
    var seen = ""
    try {
        throw new RangeError(xs[0])
    } catch (e) {
        seen = e.name + "/" + e.message + "/" + (e instanceof RangeError) + "/" + (e instanceof Error)
    }
    check("err14", seen === "RangeError/boom/true/true")

    // ----- the ENGINE's own errors are instances of these classes -----
    // The BigInt mix error, the iterator-throw TypeError, and the number
    // toString() radix RangeError.
    var got = []
    try { got.push("v" + (xs[1] + xs[2])) } catch (e) {
        got.push("mix:" + (e instanceof TypeError) + (e.message.indexOf("Cannot mix BigInt") === 0))
    }
    try { got.push("v" + BigInt(xs[4])) } catch (e) {
        got.push("syn:" + (e instanceof SyntaxError) + (e instanceof Error))
    }
    try { got.push("v" + (xs[3]).toString(1)) } catch (e) {
        got.push("rdx:" + (e instanceof RangeError) + (e.message.indexOf("radix") >= 0))
    }
    check("err15", got.join(",") === "mix:truetrue,syn:truetrue,rdx:truetrue")
    // A yield* delegate with no throw() method: the iterator-throw TypeError.
    var gd = s38deleg()
    gd.next()
    var itm = ""
    try { gd.throw(new Error(xs[0])) } catch (e) {
        itm = "" + (e instanceof TypeError) + (e.message.indexOf("does not provide") >= 0)
    }
    check("err16", itm === "truetrue")
    // And the program keeps running after every one of them.
    check("err17", xs[1] + xs[1] === 2n)

    // The message text survives the round trip unchanged, which is what keeps
    // every `("" + e).indexOf(...)` assertion in this file honest.
    check("err18", ("" + new TypeError(xs[0])).indexOf("TypeError: ") === 0)

    // name / message / stack are NOT enumerable: Object.keys(new Error("x")) is [],
    // for-in and spread see only the program's own fields, and a write keeps message
    // hidden. `name` lives on the prototype, so writing it creates an ordinary
    // enumerable own property at that point of the key order.
    check("err19", Object.keys(new Error(xs[0])).join(",") === "" && Object.keys(new Error()).length === 0)
    var ek = []
    for (var ekk in m2) { ek.push(ekk) }
    check("err20", ek.join(",") === "code" && Object.keys({...m2}).join(",") === "code")
    var ew = new RangeError(xs[0])
    ew.message = "b"
    ew.q = 1
    ew.name = "Q"
    check("err21", Object.keys(ew).join(",") === "q,name" && ("" + ew) === "Q: b" && ("message" in ew))
}
function* s38deleg(): Generator<number> { yield* [3, 4] }

// ===== SECTION 39: the Math object =====
// Math's MEMBERS are the same set in every engine that implements a host global
// (manual 7.2): the floor carries Go's own math package, ported (runtime.c, "the
// transcendentals"), and abnf/hostglobals_test.go's TestMathMembersAgree pins the
// member set.
//
// Every row is node v24's answer except the two marked GO-vs-V8: Go's math
// package and V8 differ in the last ulp there, and both engines here are Go's.
//
// Every operand is read out of an array so the constant folders cannot answer
// at compile time (manual chapter 3).
function s39(): void {
    const a: number[] = [1, 2, 0.5, 8, 100, 3, 4, 0, 27, 0.001, 1e300, 1e-300, -1, 16, 0.1, 4294967295];

    check("math01", Math.sin(a[0]) === 0.8414709848078965);
    check("math02", Math.cos(a[0]) === 0.5403023058681398);
    check("math03", Math.tan(a[0]) === 1.557407724654902);          // GO-vs-V8: node 1.5574077246549023
    check("math04", Math.asin(a[2]) === 0.5235987755982989);
    check("math05", Math.acos(a[2]) === 1.0471975511965976);        // GO-vs-V8: node 1.0471975511965979
    check("math06", Math.atan(a[0]) === 0.7853981633974483);
    check("math07", Math.sinh(a[0]) === 1.1752011936438014);
    check("math08", Math.cosh(a[0]) === 1.5430806348152437);
    check("math09", Math.tanh(a[0]) === 0.7615941559557649);
    check("math10", Math.asinh(a[0]) === 0.881373587019543);
    check("math11", Math.acosh(a[0]) === 0 && Math.atanh(a[2]) === 0.5493061443340548);
    check("math12", Math.exp(a[0]) === 2.718281828459045);
    check("math13", Math.expm1(a[0]) === 1.718281828459045);
    check("math14", Math.log(a[0]) === 0 && Math.log1p(a[0]) === 0.6931471805599453);
    // log2 of a power of two is EXACT, which Log(x)/Ln2 is not - Go's Log2 has a
    // frac == 0.5 arm for it and the port carries it.
    check("math15", Math.log2(a[3]) === 3 && Math.log2(a[13]) === 4);
    check("math16", Math.log10(a[4]) === 2 && Math.log10(a[9]) === -3);
    check("math17", Math.cbrt(a[8]) === 3 && Math.cbrt(a[12]) === -1);
    check("math18", Math.atan2(a[0], a[1]) === 0.4636476090008061);

    // hypot: variadic, and it must overflow only when the RESULT does (a naive sum
    // of squares gives Infinity for the first row and 0 for the second).
    check("math19", Math.hypot(a[3], a[5]) === 8.54400374531753);
    check("math20", Math.hypot(a[10], a[10]) === 1.4142135623730952e+300);
    check("math21", Math.hypot(a[11], a[11]) === 1.4142135623730952e-300);
    check("math22", Math.hypot(a[5], a[6]) === 5 && Math.hypot(a[5], a[6], a[13]) === 16.76305461424021);

    // clz32 and fround.
    check("math23", Math.clz32(a[0]) === 31 && Math.clz32(a[7]) === 32);
    check("math24", Math.clz32(a[15]) === 0 && Math.clz32(a[1]) === 30);
    check("math25", Math.fround(a[14]) === 0.10000000149011612);
    check("math26", Math.fround(a[0]) === 1 && Math.fround(a[10]) === Infinity);

    check("math27", Math.imul(a[5], a[6]) === 12 && Math.imul(a[12], a[15]) === 1);
    check("math28", Math.LN2 === 0.6931471805599453 && Math.LN10 === 2.302585092994046);
    check("math29", Math.LOG2E === 1.4426950408889634 && Math.LOG10E === 0.4342944819032518);
    check("math30", Math.SQRT2 === 1.4142135623730951 && Math.SQRT1_2 === 0.7071067811865476);
    check("math31", Math.PI === 3.141592653589793 && Math.E === 2.718281828459045);

    check("math32", Math.log2(a[10]) === 996.5784284662087);
    check("math33", Math.exp(a[12]) === 0.36787944117144233 && Math.sinh(a[12]) === -1.1752011936438014);
    check("math34", Math.atanh(a[12]) === -Infinity);

    // Operands outside the int32 range. ToInt32 is a modulo 2^32, so 4294967297 is 1
    // and 1e300 is 0. Values read off node 24.
    var b = [4294967296, 4294967297, 1e300, -1e300, 1e20, 16777217, 0.49999999999999994, -0.4, 2.5, -2.5, 0, 3.4028236e38];
    var nz = -b[10];
    check("math35", Math.imul(b[2], b[1]) === 0 && Math.imul(b[1], b[1]) === 1 && Math.imul(b[0], b[5]) === 0);
    check("math36", Math.imul(b[5], b[4]) === 1661992960 && Math.imul(b[3], b[5]) === 0);
    check("math37", Math.clz32(b[2]) === 32 && Math.clz32(b[4]) === 1 && Math.clz32(b[1]) === 31 && Math.clz32(b[0]) === 32);
    check("math38", Math.fround(b[11]) === Infinity && Math.fround(b[5]) === 16777216 && Math.hypot(b[2], b[3]) === 1.4142135623730952e+300);
    // round is not floor(x + 0.5): that addition rounds 0.49999999999999994 up, and a
    // negative operand rounding to zero gives -0. max/min order the two zeros.
    check("math39", Math.round(b[6]) === 0 && 1 / Math.round(b[7]) === -Infinity && Math.round(b[8]) === 3 && Math.round(b[9]) === -2);
    check("math40", 1 / Math.max(nz, b[10]) === Infinity && 1 / Math.min(b[10], nz) === -Infinity && 1 / Math.round(nz) === -Infinity);

    // Math.random: a number in [0, 1) on every call. The values differ per run, so
    // only that property is asserted.
    var rok = 0;
    for (var ri = 0; ri < 50; ri++) {
        var rv = Math.random();
        if (typeof rv === "number" && rv >= 0 && rv < 1) { rok = rok + 1 }
    }
    check("math41", typeof Math.random === "function" && rok === 50);
}

// ===== SECTION 40: three uncatchable aborts become catchable throws =====
// A name with no binding, a member read on null/undefined and `new Promise(5)` throw
// a ReferenceError or a TypeError in node that a plain try/catch recovers. The first
// two live in the FLOOR's scope_get / get_member (and abnf/jsrt.go's scopeGet /
// getMember), shared by all sixteen languages, so they are reached through a
// per-language hook the emitter registers - js_rt_missvar / js_rt_missmem - and a
// language that registers none is unchanged, which is what keeps the three
// SHOULD-ABORT matrix rows where they are.
//
// Also here: `typeof K` for a class is "function", and the BigInt mix message has
// no operator suffix. Every row is node v24's answer.
function s40() {
    var xs = [null, undefined, {a: 1}, 5, "k", 1n]

    var r1 = "none"
    try { r1 = "" + nosuchname_zz }
    catch (e) { r1 = (e instanceof ReferenceError) + "|" + (e instanceof Error) + "|" + e.name + "|" + e.message }
    check("ref1", r1 === "true|true|ReferenceError|nosuchname_zz is not defined")
    // `typeof <bare name>` is still the one expression that may name an
    // undeclared variable without raising.
    check("ref2", typeof nosuchname_zz === "undefined")

    var r2 = "none"
    try { r2 = "" + xs[0].k }
    catch (e) { r2 = (e instanceof TypeError) + "|" + e.message }
    check("ref3", r2 === "true|Cannot read properties of null (reading 'k')")
    var r3 = "none"
    try { r3 = "" + xs[1][xs[4]] }
    catch (e) { r3 = (e instanceof TypeError) + "|" + e.message }
    check("ref4", r3 === "true|Cannot read properties of undefined (reading 'k')")

    var r4 = "none"
    try { new Promise(xs[3]) }
    catch (e) { r4 = (e instanceof TypeError) + "|" + e.message }
    check("ref5", r4 === "true|Promise resolver 5 is not a function")

    // The program CONTINUES afterwards.
    var n = 0
    for (var i = 0; i < 3; i++) {
        try { n = n + xs[0].len }
        catch (e) { n = n + 1 }
    }
    check("ref6", n === 3)

    // ...and the throw travels the ordinary try/finally machinery.
    var log = []
    try {
        try { log.push("t"); var q = xs[1].z }
        finally { log.push("f") }
    } catch (e) { log.push("c:" + e.name) }
    check("ref7", log.join(",") === "t,f,c:TypeError")

    class K { m() { return 1 } }
    check("ref8", typeof K === "function")
    var arr = [K]
    check("ref9", typeof arr[0] === "function" && typeof new K() === "object")
    check("ref10", typeof Math === "object" && typeof Math.max === "function")

    // The WRITE sibling of ref3/ref4, with node's other wording.
    var r6 = "none"
    try { xs[0].k = 1 }
    catch (e) { r6 = (e instanceof TypeError) + "|" + e.message }
    check("ref12", r6 === "true|Cannot set properties of null (setting 'k')")

    var r5 = "none"
    try { r5 = "" + (xs[5] + xs[3]) }
    catch (e) { r5 = e.message }
    check("ref11", r5 === "Cannot mix BigInt and other types, use explicit conversions")
}

// ===== SECTION 41: calling a non-function is a catchable TypeError, worded as node does =====
// `xs[1]()` with undefined in the slot is a catchable TypeError in node, worded
// `xs[1] is not a function` - the callee's SOURCE TEXT, which V8 recovers by
// re-printing the AST and which no runtime value carries, since `xs[1]` and
// `xs[i]` reach the runtime as the same key.
//
// So the text is built at PARSE time (siteTexts, one copy per grammar) and the
// runtime's job is only to say "nothing claimed this call": the floor's js_call
// and the shared memberCall are used by every language's layer-2 internals and
// still abort, and js/ts opt out through js_rt_misscall - the fourth of the
// js_rt_miss* family - plus a raw callability test at the call site itself.
//
// Every message below is node v24's, byte for byte.
function s41() {
    var xs: any[] = [1, undefined, null, "s", {}, [1, 2], 7]
    var o: any = {a: 1, g: 5, sub: {}}
    var i = 1
    var n = xs[6]
    var s = xs[3]
    var u = xs[1]
    var nl = xs[2]
    var arr = xs[5]

    function msg(f: any): string {
        try { f(); return "none" }
        catch (e) { return (e instanceof TypeError) + "|" + (e instanceof Error) + "|" + e.message }
    }

    // A member call whose member is missing or is not callable.
    check("nfn1", msg(function () { return xs[1]() }) === "true|true|xs[1] is not a function")
    check("nfn2", msg(function () { return xs[i]() }) === "true|true|xs[i] is not a function")
    check("nfn3", msg(function () { return o.nope() }) === "true|true|o.nope is not a function")
    // node prints a string-literal index as a dot access.
    check("nfn4", msg(function () { return o["nope"]() }) === "true|true|o.nope is not a function")
    check("nfn5", msg(function () { return o.g() }) === "true|true|o.g is not a function")
    check("nfn6", msg(function () { return o.sub.q() }) === "true|true|o.sub.q is not a function")
    check("nfn7", msg(function () { return o.g?.() }) === "true|true|o.g is not a function")
    check("nfn8", msg(function () { return s.nope() }) === "true|true|s.nope is not a function")
    check("nfn9", msg(function () { return arr.nope() }) === "true|true|arr.nope is not a function")
    check("nfn10", msg(function () { return n.nope() }) === "true|true|n.nope is not a function")
    check("nfn11", msg(function () { return Math.nope() }) === "true|true|Math.nope is not a function")

    // A PLAIN call of a value that is not callable, which is the floor's js_call.
    check("nfn12", msg(function () { return u() }) === "true|true|u is not a function")
    check("nfn13", msg(function () { return nl() }) === "true|true|nl is not a function")
    check("nfn14", msg(function () { return n() }) === "true|true|n is not a function")
    check("nfn15", msg(function () { return s() }) === "true|true|s is not a function")
    check("nfn16", msg(function () { return o() }) === "true|true|o is not a function")
    check("nfn17", msg(function () { return arr() }) === "true|true|arr is not a function")

    // The parentheses of a parenthesised callee are not part of node's text, and
    // anything a CALL produced is "(intermediate value)".
    check("nfn18", msg(function () { return (o.g)() }) === "true|true|o.g is not a function")
    var f5 = function () { return 5 }
    check("nfn19", msg(function () { return f5()() }) === "true|true|f5(...) is not a function")
    // A literal receiver is printed as written.
    check("nfn20", msg(function () { return "abc".nope() }) === "true|true|\"abc\".nope is not a function")
    check("nfn21", msg(function () { return [].nope() }) === "true|true|[].nope is not a function")

    // A class instance whose method does not exist.
    class C { m() { return 1 } }
    var c = new C()
    check("nfn22", msg(function () { return c.zz() }) === "true|true|c.zz is not a function")

    // A generator's member table, and a tagged template.
    check("nfn23", msg(function () { var g = xs[3]; return g`x` }) === "true|true|g is not a function")

    // `new` on something that is not a constructor - node's other wording.
    check("nfn24", msg(function () { return new n() }) === "true|true|n is not a constructor")
    check("nfn25", msg(function () { return new o.nope() }) === "true|true|o.nope is not a constructor")
    check("nfn26", msg(function () { return new o() }) === "true|true|o is not a constructor")

    // ?.() short circuits on a nullish callee instead of raising.
    check("nfn27", u?.() === undefined && o.nope?.() === undefined && xs[1]?.() === undefined)

    // The program CONTINUES, the throw travels the ordinary try/finally machinery,
    // and it can be caught, wrapped and re-thrown.
    var k = 0
    for (var j = 0; j < 3; j++) {
        try { xs[1]() } catch (e) { k = k + 1 }
    }
    check("nfn28", k === 3)
    var log: string[] = []
    try {
        try { log.push("t"); o.nope() }
        finally { log.push("f") }
    } catch (e) { log.push("c:" + e.name) }
    check("nfn29", log.join(",") === "t,f,c:TypeError")
    var r = "none"
    try {
        try { xs[1]() } catch (e) { throw new TypeError("re:" + e.message) }
    } catch (e2) { r = e2.message }
    check("nfn30", r === "re:xs[1] is not a function")

    // A call in the MIDDLE of an assignment target, which is its own walk in every
    // engine.
    check("nfn31", msg(function () { n().x = 1 }) === "true|true|n is not a function")
    check("nfn32", msg(function () { o.sub.f().y = 1 }) === "true|true|o.sub.f is not a function")
    // super.m() with nothing on the __super chain: node has no source text for a
    // super receiver either and prints "(intermediate value)".
    class SA { }
    class SB extends SA { go() { return super.zz() } }
    check("nfn33", msg(function () { return new SB().go() }) === "true|true|(intermediate value).zz is not a function")
}

// ===== SECTION 42: the top-level 'this' is the global object (the Script goal) =====
// These grammars parse the SCRIPT goal symbol - the Annex B HTML-like comments are
// accepted (B.1.3 admits them in a Script only), the code is sloppy unless a
// "use strict" directive says otherwise, and top-level `return` is a syntax error,
// which rules out the CommonJS wrapper node's `--check` puts around a file. So the
// top-level `this` is the GLOBAL OBJECT, and every value below is
// `node -e 'vm.runInThisContext(src)'`'s, except where a comment says otherwise.
var s42this = this
var s42arrow = () => this
var s42nested = () => (() => this)()
function s42plain() { return this }
function s42strict() { "use strict"; return this }
function s42outer() { "use strict"; function inner() { return this } return inner() }
function s42sloppyOuter() { function inner2() { return this } return inner2() }
function s42arrowIn() { var a = () => this; return a() }
function s42arrowInStrict() { "use strict"; var a = () => this; return a() }
class s42K { mk() { function inner3() { return this } return inner3() } }
var s42litm = {q: function () { return this }}
class S42C {
    static tag = "S42"
    static who() { return this }
    static viaThis() { return this.tag }
}

function s42() {
    // The top level.
    check("gthis1", typeof s42this === "object")
    check("gthis2", (s42this === undefined) === false)
    check("gthis3", (s42this === null) === false)
    check("gthis4", typeof globalThis === "object")
    check("gthis5", s42this === globalThis)
    check("gthis6", !!s42this)
    // An arrow at the top level binds no `this` of its own, so it walks out to the
    // global object - at any nesting depth.
    check("gthis7", s42arrow() === s42this)
    check("gthis8", s42nested() === s42this)
    // The global object is a PLAIN object here, not a live view of the scope chain:
    // a global variable is not a property of it. node's is (`globalThis.s42this` is
    // the same value), and closing that gap needs an object identity for the root
    // scope in all four engines.
    check("gthis9", globalThis.s42this === undefined)
    // `"" + this` is "[object global]" in node and "[object Object]" here, for the
    // same reason: nothing in the value model marks this object as the global one.
    check("gthis10", ("" + s42this) === "[object Object]")

    // A STATIC method's `this` is the class.
    check("gthis11", S42C.who() === S42C)
    check("gthis12", S42C.viaThis() === "S42")
    check("gthis13", typeof S42C.who() === "function")
    // An instance method's and a constructor's are unchanged by any of that.
    class D { constructor() { this.v = 1 } m() { return this } }
    var d = new D()
    check("gthis14", d.m() === d && d.v === 1)
    var lit = {t: "L", m: function () { return this.t }}
    check("gthis15", lit.m() === "L")

    // OrdinaryCallBindThis (ES2023 10.2.1.2), the per-function STRICT-MODE flag.
    // Every value below is `new vm.Script(src)`'s.
    check("gthis16", s42plain() === globalThis)
    check("gthis18", s42plain.call(null) === globalThis)
    check("gthis19", s42plain.call(undefined) === globalThis)
    check("gthis20", s42plain.apply(null, []) === globalThis)
    // A STRICT function keeps whatever the call site handed over, undefined and
    // null included - which is what a flag that is per FUNCTION and not per file
    // is for: this one sits in the same sloppy Script as the row above it.
    check("gthis21", s42strict() === undefined)
    check("gthis22", s42strict.call(null) === null)
    check("gthis23", s42strict.call(7) === 7)
    // Strictness is INHERITED lexically, in both directions: a plain function
    // declared inside a strict one is strict, and one declared inside a CLASS
    // BODY is strict too (class code always is).
    check("gthis24", s42outer() === undefined)
    check("gthis25", (new s42K()).mk() === undefined)
    // ...while a function nested in an ordinary sloppy one stays sloppy.
    check("gthis26", s42sloppyOuter() === globalThis)
    // An ARROW binds no `this` of its own, so it reports the enclosing FUNCTION's -
    // which is the global object when that function is sloppy and undefined when
    // it is strict.
    check("gthis27", s42arrowIn() === globalThis)
    check("gthis28", s42arrowInStrict() === undefined)
    // An object-literal method, and a getter, are functions of the enclosing code's
    // strictness like any other.
    check("gthis29", s42litm.q.call(null) === globalThis)
    // A sloppy GENERATOR binds `this` the same way in the interpreter half and
    // under llvm.Run - and NOT in the native build, which is why there is no
    // assertion here. `js_this` is one global slot in layer 2, and the emitter arms
    // it with the RECEIVER of the `g.next()` call, so a generator body reads the
    // generator object instead of the receiver its function was called with.
    // Closing it needs a per-coroutine `this` in the C floor's control block.
    // Recorded rather than asserted, since clang-check and native-full run this file
    // natively.
    // Not asserted: a sloppy body boxes a primitive receiver (s42plain.call(7) is a
    // Number object in node), and this value model has no wrapper objects.
    var detached = lit.m
    check("gthis17", typeof detached === "function")
}

// ===== SECTION 43: parentheses do not drop the receiver =====
// `(o.m)()` calls m with `this` === o. A parenthesized expression evaluates to whatever
// the inner expression evaluates to, and a MemberExpression evaluates to a REFERENCE,
// which carries its base - so the parentheses are transparent to the receiver exactly
// as they are to an assignment target ((a.b).c = 1 has always assigned where a.b.c = 1
// does). The matrix and --cross cannot see a both-halves-agree defect here, so every
// row is node's answer. The TypeScript-only postfix forms are part of the same rule:
// `(x as T).m()` and `(x!).m()` are type-layer only, so they must keep the receiver
// too.
//
// The negative rows are the point of the section as much as the positive ones. Only a
// parenthesized MemberExpression is a Reference: `(0, o.m)()` is the IDIOM for calling a
// method WITHOUT its receiver, and a fix that made it pass `o` would be a regression
// dressed as a feature. Every value here is `new vm.Script(src)`'s.
//
// Every operand is read out of s43a so the constant folder cannot answer any of it.
var s43a: any[] = ["m", "n", 1, 2, 3];
function s43tag(v: any): string {
    if (v === undefined) { return "undefined" }
    if (v === globalThis) { return "GLOBAL" }
    if (v !== null && v.nm !== undefined) { return v.nm }
    return typeof v;
}
class S43B { who(): string { return "B/" + s43tag(this) } static swho(): string { return "SB/" + s43tag(this) } }
// `super.m` is a Reference too, and its thisValue is the CURRENT `this` - which is what
// makes the parenthesized and the computed spelling agree with the plain call.
class S43D extends S43B {
    who(): string { return "D" }
    static sPlain(): string { return super.swho() }
    static sParen(): string { return (super.swho)() }
}
class S43K { nm: string; constructor() { this.nm = "k" } m(): string { return s43tag(this) } static sm(): any { return this } }

function s43(): void {
    var o: any = {nm: "o", m: function (): string { return s43tag(this) }, n: function (): string { return s43tag(this) }};
    var p: any = {nm: "p"};
    var k0: any = s43a[0];
    check("paren01", o.m() === "o");
    check("paren02", (o.m)() === "o");
    check("paren03", ((o.m))() === "o");
    check("paren04", (o?.m)() === "o");
    check("paren05", (o[k0])() === "o");
    check("paren06", ((o).m)() === "o");
    check("paren07", (o.m).call(p) === "p");
    check("paren08", (o.m)?.() === "o");
    check("paren09", (o?.m)?.() === "o");
    // A TAGGED template behind a parenthesized member is a call like any other.
    check("paren10", (o.m)`x` === "o");
    var w: any = {nm: "w", g: function (): any { return {nm: "inner", h: function (): string { return s43tag(this) }} }};
    check("paren11", (w.g().h)() === "inner");
    check("paren12", (w.g)().nm === "inner");
    // As a STATEMENT, not only in an expression.
    var arr: any[] = [s43a[2]];
    (arr.push)(s43a[3]);
    check("paren13", arr.length === 2 && arr[1] === 2);
    check("paren14", ("abc".toUpperCase)() === "ABC");
    var kk: any = new S43K();
    check("paren15", (kk.m)() === "k");
    check("paren16", (S43K.sm)() === S43K);
    check("paren17", ((S43K.sm))() === S43K);
    check("paren18", S43D.sPlain() === "SB/" + s43tag(S43D));
    check("paren19", S43D.sParen() === S43D.sPlain());
    // The type-layer suffixes are erased, so they change nothing about the receiver.
    check("paren21", (o.m as any)() === "o");
    check("paren22", (o.m!)() === "o");
    // ...and the NEGATIVE side: a value, not a Reference, so `this` is the global object
    // in this sloppy Script.
    var f: any = o.m;
    check("paren23", f() === "GLOBAL");
    check("paren24", (f)() === "GLOBAL");
    check("paren25", (0, o.m)() === "GLOBAL");
    check("paren26", (s43a[0] === "m" ? o.m : o.n)() === "GLOBAL");
    check("paren27", (o.m, o.n)() === "GLOBAL");
    check("paren28", (o.m || o.n)() === "GLOBAL");
    check("paren29", (o.m = o.n, o.n)() === "GLOBAL");
    // V8 prints the callee's source text without the parentheses; so does siteTexts.
    check("paren30", (function (): string {
        try { (o.nope)(); return "no" } catch (e: any) { return e.message }
    })() === "o.nope is not a function");
}

// ===== SECTION 44: a class's STATIC and INSTANCE namespaces =====
// `class C { static m(){} m(){} }` is TWO members in JavaScript - one on the
// constructor and one on the prototype - so `C.m()` must answer the STATIC method,
// `class F extends E {}` must resolve an inherited static to E's static, and an
// instance must not see a static (`typeof c.sonly` is "undefined" in node). Every
// static member lives in a separate `__statics` table on the descriptor; instance
// members stay in the descriptor, which is what keeps the shared js_get / js_set /
// js_mcall - and therefore the other fifteen languages - untouched.
//
// Every value is `new vm.Script(src)`'s, and every operand comes out of s44a so the
// constant folder cannot answer any of it.
var s44a: any[] = [1, 2, 3, "m", "who"];
class S44C {
    static m(): string { return "S-m" }
    m(): string { return "I-m" }
    static get g(): string { return "S-g" }
    get g(): string { return "I-g" }
    static sf: number = s44a[0];
    static sonly(): string { return "S-only" }
    ionly(): string { return "I-only" }
    f: number;
    constructor() { this.f = s44a[1] }
    static set sv(v: any) { (S44C as any).svSeen = v }
}
class S44E { static x(): string { return "E-static-x" } static only(): string { return "E-only" } }
class S44F extends S44E { x(): string { return "F-inst-x" } }
class S44H { static n(): string { return "H-S-n" } n(): string { return "H-I-n" } static viaThis(): string { return this.n() } }
// The four spellings of a super member. `super.m` is a Reference whose thisValue is the
// CURRENT `this`, so which SIDE it resolves on is decided by the receiver: a static
// method's is the class, an instance method's is the instance. The parenthesized and
// computed forms have to agree with the plain one.
class S44G extends S44C {
    static viaSuper(): string { return super.m() }
    // `super[k]()`, without the `as any` a checker would want: `(super as any)[k]`
    // does not parse in either typescript grammar (a bare `super` is not a primary),
    // which is a separate parser gap.
    static viaSuperIdx(): string { return super[s44a[3]]() }
    viaSuperI(): string { return super.m() }
    viaSuperParen(): string { return (super.m)() }
}
class S44B { nm: string; who(): string { return "B/" + (this === undefined ? "undef" : this.nm) } }
class S44D extends S44B { who(): string { return "D" } paren(): string { return (super.who)() } }

function s44(): void {
    var c: any = new S44C();
    check("stat01", S44C.m() === "S-m");
    check("stat02", c.m() === "I-m");
    check("stat03", S44C.g === "S-g");
    check("stat04", c.g === "I-g");
    check("stat05", S44C.sf === 1 && c.f === 2);
    // An instance cannot see a static, and the constructor cannot see an instance member.
    check("stat06", typeof S44C.sonly === "function" && typeof c.sonly === "undefined");
    check("stat07", typeof (S44C as any).ionly === "undefined" && typeof c.ionly === "function");
    // An INHERITED static beats the subclass's own INSTANCE member of the same name.
    check("stat08", S44F.x() === "E-static-x" && (new S44F()).x() === "F-inst-x");
    check("stat09", S44F.only() === "E-only");
    // `this.n()` inside a static resolves on the static side.
    check("stat10", S44H.viaThis() === "H-S-n" && (new S44H()).n() === "H-I-n");
    check("stat11", (s44a[3] in S44C) === true && ("ionly" in S44C) === false);
    // Every class object has a `name`, and it is the class's own.
    check("stat12", S44C.name === "S44C" && S44F.name === "S44F");
    (S44C as any).zz = s44a[2];
    check("stat13", (S44C as any).zz === 3 && ((new S44C()) as any).zz === undefined);
    (S44G as any).sv = s44a[0];
    check("stat14", (S44G as any).svSeen === 1);
    check("stat15", S44G.viaSuper() === "S-m" && S44G.viaSuperIdx() === "S-m");
    check("stat16", (new S44G()).viaSuperI() === "I-m" && (new S44G()).viaSuperParen() === "I-m");
    // A parent INSTANCE method reached through a super REFERENCE runs on the current
    // receiver, not on undefined.
    var d: any = new S44D(); d.nm = "d";
    check("stat17", d.paren() === "B/d");
    check("stat18", (function (): string {
        try { (S44C as any).ionly(); return "no" } catch (e: any) { return e.message }
    })() === "S44C.ionly is not a function");
}

// ===== SECTION 45: a class object's OWN KEYS, e.constructor, and a method value's
//       OVERRIDABLE receiver =====
//
//   Object.keys(<class>)  node lists the static FIELDS. A static method, a static
//                         accessor and a private static are not enumerable; a
//                         runtime `C.zz = 1` is.
//   e.constructor         node hands back the class, and the FUNCTION itself for
//                         `new F()`.
//   f.call(x)             a method read AS A VALUE - `const f = o.m`, `super.m` -
//                         runs on x, not on the receiver the READ was made with.
//
// Every operand comes out of s45a so the constant folder cannot answer any of it.
var s45a: any[] = [1, "two", 3, "m", "who", "constructor"];
class S45C {
    static sf: number = s45a[0];
    static sf2: string = s45a[1];
    static sm(): string { return "sm" }
    static get sg(): string { return "sg" }
    m(): string { return "m" }
    get g(): string { return "g" }
    iv: number = s45a[0];
}
class S45Sub extends S45C { static own: number = s45a[2]; }
class S45B { nm: string = ""; who(): string { return "B/" + this.nm } }
class S45D extends S45B {
    constructor() { super(); this.nm = "d" }
    paren(): string { return (super.who)() }
    callx(o: any): string { return super.who.call(o) }
    applyx(o: any): string { return super.who.apply(o, []) }
    bindx(o: any): string { return super.who.bind(o)() }
}
function S45F(n: any) { (this as any).nm = n }

function s45(): void {
    // A static FIELD is enumerable; a static method, a static accessor and the
    // instance members are not, and an inherited static is not an OWN key.
    check("keys01", Object.keys(S45C).join("|") === "sf|sf2")
    check("keys02", Object.keys(S45Sub).join("|") === "own")
    check("keys03", Object.values(S45C).join("|") === "1|two")
    var en: any[] = Object.entries(S45C)
    check("keys04", en.length === 2 && en[0][0] === "sf" && en[0][1] === 1 && en[1][1] === "two")
    ;(S45C as any).zz = s45a[2]
    check("keys05", Object.keys(S45C).join("|") === "sf|sf2|zz" && (S45C as any).zz === 3)
    var fi: string[] = []
    for (var k in S45C) { fi.push(k) }
    check("keys06", fi.join("|") === "sf|sf2|zz")
    check("keys07", Object.keys(Object.assign({}, S45C)).join("|") === "sf|sf2|zz")
    check("keys08", Object.keys(new S45C()).join("|") === "iv")
    // The class object still answers every static it did before.
    check("keys09", S45C.sm() === "sm" && S45C.sg === "sg" && S45Sub.sm() === "sm")
    var c: any = new S45C()
    check("ctor01", c.constructor === S45C && c.constructor.name === "S45C")
    check("ctor02", (new S45Sub()).constructor === S45Sub)
    check("ctor03", (s45a[5] in c) === true)
    check("ctor04", (new S45F("f")).constructor === S45F)
    check("ctor05", Object.keys(c).join("|") === "iv")
    // A method read as a VALUE takes the receiver of the call that eventually runs
    // it, and keeps the read-time one only when the call supplies none.
    var d: S45D = new S45D()
    var o: any = {nm: "x"}
    check("recv01", d.callx(o) === "B/x")
    check("recv02", d.applyx(o) === "B/x")
    check("recv03", d.bindx(o) === "B/x")
    check("recv04", d.paren() === "B/d")
    var bm: any = d.who
    check("recv05", bm.call(o) === "B/x" && bm.apply(o, []) === "B/x" && bm.bind(o)() === "B/x")
}

// ===== SECTION 46: the PRIVATE-NAME model, and for-in over what is not a plain
//       object =====
//
//   THE PRIVATE NAME. `#x` is stored under the mangled key '__#x' in all four
//   engines, which the engine's __-slot rule already hides: `Object.keys(p)` is "y"
//   as in node, and `o["#x"] = 1` and a private `#x` are two different slots. Every
//   __-name is skipped by the static-side test, so the mangled private spelling is
//   the one exception there (`C.#s`, a static private, must read).
//
//   for (k in [7,8])  yields the INDICES as strings, as in node. A string enumerates
//   the same way; null, undefined and a primitive enumerate nothing rather than
//   raising. `{...[7,8]}` goes through the same call.
//
//   #x in o - ES2022's ergonomic brand check.
//
// Every operand comes out of s46a so the constant folder cannot answer any of it.
const s46a: any[] = [0, 1, 2, "#x", "ab", 4];
class S46P {
    #x: number = s46a[1];
    y: number = s46a[2];
    static #s: number = s46a[5];
    #bump(n: number): number { this.#x = this.#x + n; return this.#x }
    get val(): number { return this.#x }
    run(n: number): number { return this.#bump(n) }
    brand(o: any): boolean { return #x in o }
    brandM(o: any): boolean { return #bump in o }
    static gets(): number { return S46P.#s }
    static bumps(): number { S46P.#s = S46P.#s + s46a[1]; return S46P.#s }
}

function s46(): void {
    const p: any = new S46P();
    // A private field is not an own key, is not spread and is not enumerated.
    check("priv01", Object.keys(p).join("|") === "y");
    check("priv02", Object.keys({ ...p }).join("|") === "y");
    var fi = "";
    for (var k in p) { fi = fi + k + "," }
    check("priv03", fi === "y,");
    // It is still readable and writable from inside the class body.
    check("priv04", p.val === 1 && p.run(s46a[5]) === 5 && p.val === 5);
    // A STATIC private is reachable, and it is not an own key of the class object
    // either.
    check("priv05", S46P.gets() === 4 && S46P.bumps() === 5 && S46P.gets() === 5);
    check("priv06", Object.keys(S46P).join("|") === "");
    // '#x' AS AN ORDINARY STRING KEY is a DIFFERENT SLOT from the private #x: the
    // write below must not disturb the field, and it IS enumerable.
    p[s46a[3]] = s46a[5]
    check("priv07", p.val === 5 && p[s46a[3]] === 4);
    check("priv08", Object.keys(p).join("|") === "y|#x");
    const q: any = {}
    q[s46a[3]] = s46a[1]
    check("priv09", Object.keys(q).join("|") === "#x" && q[s46a[3]] === 1);
    // The ergonomic brand check, over a field and over a method.
    check("priv10", p.brand(p) === true && p.brand({}) === false);
    check("priv11", p.brandM(p) === true && p.brandM({}) === false);

    // ----- for-in over what is not a plain object -----
    const arr: any[] = [s46a[5], s46a[5], s46a[5]]
    var ai = "";
    for (var i in arr) { ai = ai + i + "," }
    check("fin01", ai === "0,1,2,");
    // The key is a STRING, as node says, not a number.
    var tk = "";
    for (var i2 in arr) { tk = tk + typeof i2 }
    check("fin02", tk === "stringstringstring");
    var si = "";
    for (var j in s46a[4]) { si = si + j + "," }
    check("fin03", si === "0,1,");
    // null, undefined and a primitive enumerate NOTHING and do not raise.
    const none: any[] = [null, undefined, s46a[5], true, ""]
    var nc = 0
    for (var n = 0; n < none.length; n++) {
        for (var k2 in none[n]) { nc = nc + 1 }
    }
    check("fin04", nc === 0);
    // break and continue inside a for-in over an array.
    var cc = "";
    for (var c in arr) { if (c === "1") { continue } cc = cc + c }
    check("fin05", cc === "02");
    var bb = "";
    for (var b in arr) { if (b === "1") { break } bb = bb + b }
    check("fin06", bb === "0");
    // The same call is what object spread makes.
    const sa: any = { ...arr }
    check("fin07", Object.keys(sa).join("|") === "0|1|2" && sa[s46a[1]] === 4);
    const ss: any = { ...s46a[4] }
    check("fin08", Object.keys(ss).join("|") === "0|1" && ss[s46a[0]] === "a");
    check("fin09", Object.keys({ ...null }).length === 0 && Object.keys({ ...s46a[5] }).length === 0);
}

// ===== SECTION 47: for-in is a SNAPSHOT, a private name is per CLASS BODY, and a
//       FUNCTION carries own properties =====
// Three things every engine here could agree on wrongly, so byte-identity could not
// see them; node is the only witness. Each row below was checked against node v24.
//
//   DELETING A KEY DURING for-in. js_jskeys / jsKeys answer a SNAPSHOT, so a key the
//   body deleted before its turn came must be skipped: `for (k in o) { s += k;
//   delete o.c }` gives "ab" in node. The compilers guard each iteration with
//   js_jshas and the interpreters with propIn.
//
//   THE SAME PRIVATE NAME IN A SUBCLASS. `class A { #v } class B extends A { #v }` is
//   TWO names in TypeScript, so the key carries the CLASS BODY ('__#<body>#v'),
//   resolved to the innermost enclosing body that declares the name - so an inner
//   class reading an outer class's private name still reaches it.
//
//   AN OWN PROPERTY ON A FUNCTION. `f.p = 1`: the map lives beside the function in a
//   side table in all four engines, and for-in, Object.keys, `in`, hasOwnProperty
//   and delete all read it.
//
// Every operand comes out of s47a so the constant folder cannot answer any of it.
const s47a: any[] = [0, 1, 2, 3, "a", "b", "c", "d"];

class S47A {
    #v: number = s47a[1];
    getA(): number { return this.#v }
    setA(x: number): void { this.#v = x }
    hasA(o: any): boolean { return #v in o }
}
class S47B extends S47A {
    #v: number = s47a[2];
    getB(): number { return this.#v }
    setB(x: number): void { this.#v = x }
    hasB(o: any): boolean { return #v in o }
}
class S47C { #m(): string { return s47a[4] } callC(): string { return this.#m() } }
class S47D extends S47C { #m(): string { return s47a[5] } callD(): string { return this.#m() } }
class S47E { static #s: number = s47a[1]; static getE(): number { return S47E.#s } }
class S47F extends S47E { static #s: number = s47a[2]; static getF(): number { return S47F.#s } }
// An INNER class body reading the OUTER body's private name: the reference resolves
// to the innermost enclosing body that DECLARES it, which is the outer one here.
class S47G {
    #g: number = s47a[3];
    read(): number { return this.#g }
    make(): any { return class S47H { peek(o: any): number { return o.#g } } }
}
// A method that reads a field declared LATER in the same body - which is why the
// reference key cannot be settled while the body is being walked.
class S47L { late(): number { return this.#l } #l: number = s47a[3] }

const s47fn: any = function (x: any): any { return x };

function s47(): void {
    // ----- for-in is a snapshot, but a deleted key is skipped -----
    const o1: any = { a: s47a[1], b: s47a[2], c: s47a[3] };
    var r1 = "";
    for (var k1 in o1) { r1 = r1 + k1; delete o1.c }
    check("fin10", r1 === "ab");
    const o2: any = { a: s47a[1], b: s47a[2], c: s47a[3] };
    var r2 = "";
    for (var k2 in o2) { r2 = r2 + k2; if (k2 === s47a[4]) { delete o2.b } }
    check("fin11", r2 === "ac");
    const o3: any = { a: s47a[1], b: s47a[2], c: s47a[3], d: s47a[3] };
    var r3 = "";
    for (var k3 in o3) { r3 = r3 + k3; delete o3.b; delete o3.d }
    check("fin12", r3 === "ac");
    // A key DELETED AND PUT BACK is still visited: it is present when its turn comes.
    const o4: any = { a: s47a[1], b: s47a[2], c: s47a[3] };
    var r4 = "";
    for (var k4 in o4) { r4 = r4 + k4; delete o4.a; o4.a = s47a[1] }
    check("fin13", r4 === "abc");
    // A key ADDED during the loop is NOT visited - the list is still a snapshot.
    const o5: any = { a: s47a[1], b: s47a[2] };
    var r5 = "";
    for (var k5 in o5) { r5 = r5 + k5; o5.z = s47a[3] }
    check("fin14", r5 === "ab");
    // The guard must not drop anything from an array, a string or a class object.
    const ar: any[] = [s47a[1], s47a[2], s47a[3]];
    var r6 = "";
    for (var k6 in ar) { r6 = r6 + k6 }
    check("fin15", r6 === "012");
    var r7 = "";
    for (var k7 in s47a[4] + s47a[5]) { r7 = r7 + k7 }
    check("fin16", r7 === "01");
    var r8 = "";
    for (var k8 in S47E) { r8 = r8 + k8 }
    check("fin17", r8 === "");

    // ----- a private name is scoped to its CLASS BODY -----
    const b: any = new S47B();
    check("priv12", b.getA() === 1 && b.getB() === 2);
    b.setA(s47a[3] * 10);
    check("priv13", b.getA() === 30 && b.getB() === 2);
    b.setB(s47a[2] * 10);
    check("priv14", b.getA() === 30 && b.getB() === 20);
    const d: any = new S47D();
    check("priv15", d.callC() === "a" && d.callD() === "b");
    check("priv16", S47E.getE() === 1 && S47F.getF() === 2);
    // The brand check is per class body too: an A has A's #v and not B's.
    check("priv17", b.hasA(b) === true && b.hasB(b) === true);
    check("priv18", b.hasA(new S47A()) === true && b.hasB(new S47A()) === false);
    const g: any = new S47G();
    const H: any = g.make();
    check("priv19", g.read() === 3 && (new H()).peek(g) === 3);
    check("priv20", (new S47L()).late() === 3);
    // Neither slot is enumerable, in either class.
    check("priv21", Object.keys(b).join("|") === "" && Object.keys({ ...b }).join("|") === "");

    // ----- a FUNCTION carries own properties -----
    s47fn.p = s47a[1];
    s47fn.q = s47a[2];
    check("fnp01", s47fn.p === 1 && s47fn.q === 2 && s47fn.zz === undefined);
    var fk = "";
    for (var k9 in s47fn) { fk = fk + k9 + "," }
    check("fnp02", fk === "p,q,");
    check("fnp03", Object.keys(s47fn).join("|") === "p|q");
    check("fnp04", (s47a[4] in s47fn) === false && ("p" in s47fn) === true);
    check("fnp05", Object.prototype.hasOwnProperty.call(s47fn, "p") === true);
    delete s47fn.p;
    check("fnp06", s47fn.p === undefined && Object.keys(s47fn).join("|") === "q");
    // A function that was never written to has no keys and is not disturbed.
    check("fnp07", Object.keys(s47).join("|") === "" && ("q" in s47) === false);
    // A property holding a function is callable, and the function is still a function.
    s47fn.m = function (): number { return s47a[3] * 14 };
    check("fnp08", s47fn.m() === 42 && s47fn(s47a[3]) === 3 && typeof s47fn === "function");
    // A property holding UNDEFINED is still an own key, as it is on any object.
    s47fn.u = undefined;
    check("fnp09", ("u" in s47fn) === true && Object.keys(s47fn).join("|") === "q|m|u");

    // ----- a FUNCTION's Object.prototype SURFACE, as methods -----
    // All four are answered by this engine, in all four legs (the host's own
    // prototype chain exists under goja only).
    check("fnp10", s47fn.hasOwnProperty("q") === true && s47fn.hasOwnProperty(s47a[4]) === false);
    check("fnp11", s47fn.propertyIsEnumerable("q") === true && s47fn.propertyIsEnumerable(s47a[4]) === false);
    // toString agrees with String(f), which is the one spelling every leg already had;
    // node agrees with itself here too, so the row does not pin our text.
    check("fnp12", typeof s47fn.toString() === "string" && s47fn.toString() === String(s47fn));
    check("fnp13", s47fn.valueOf() === s47fn);
    // An OWN property still wins over the built-in, as it does in node.
    const s47g: any = function (): any { return s47a[1] };
    s47g.valueOf = s47a[3];
    check("fnp14", s47g.valueOf === 3 && typeof s47g.toString === "function");
    // A function with NO own properties answers false rather than throwing.
    check("fnp15", s47.hasOwnProperty("q") === false && s47.propertyIsEnumerable("q") === false);
    // The VALUES, not just the keys: Object.values / entries / assign / spread read a
    // function's own properties through a different path from for-in.
    const s47v: any[] = Object.values(s47fn);
    const s47e: any[] = Object.entries(s47fn);
    check("fnp16", s47v.length === 3 && s47v[0] === 2 && typeof s47v[1] === "function" && s47v[2] === undefined);
    check("fnp16b", s47e.length === 3 && s47e[0][0] === "q" && s47e[0][1] === 2);
    check("fnp17", Object.assign({}, s47fn).q === 2 && ({ ...s47fn }).q === 2);
}

// ===== SECTION 48: a function's own `name`, `length` and SELF NAME =====
// Every function has two own properties nothing in the program writes; node says
// `f`/1 for `function f(a){}`. Every value below was read off node 24 before it was
// asserted.
//
// `length` is the parameters BEFORE the first default and before the rest parameter,
// and a destructuring pattern counts as ONE - which is why fn9 is 3 and not 4.
//
// What is deliberately NOT asserted, because no engine here can answer it: the other
// three own names node reports on a function declaration (`arguments`, `caller`,
// `prototype` - there is no prototype chain and no call-site frame in this value
// model), and `f.constructor`, which diverges goja vs -frozen and is left unasserted
// on purpose so that no assertion pins the divergence.
function s48() {
    function fnDecl(a) { return a }
    var fnNamedExpr = function fnInner(a, b) { return a + b }
    var fnArrow = (x) => x;
    var fnAnon = function () { return 1 }
    var fnDflt = function (a, b = 1, c) { return a }
    var fnRest = function (...r) { return r.length }
    var fnDestr = function ({ a }, [b], c) { return c }
    var fnDrest = function (a, ...r) { return a }
    var fnZero = () => 0;
    check("fnm01", fnDecl.name === "fnDecl" && fnDecl.length === 1);
    // A NAMED function expression keeps ITS OWN name, and does not take the binding's.
    check("fnm02", fnNamedExpr.name === "fnInner" && fnNamedExpr.length === 2);
    // NamedEvaluation: an ANONYMOUS form takes the name of the binding it is
    // DIRECTLY assigned to.
    check("fnm03", fnArrow.name === "fnArrow" && fnArrow.length === 1);
    check("fnm04", fnAnon.name === "fnAnon" && fnAnon.length === 0);
    check("fnm05", fnZero.name === "fnZero" && fnZero.length === 0);
    check("fnm06", fnDflt.length === 1);
    check("fnm07", fnRest.length === 0);
    check("fnm08", fnDrest.length === 1);
    check("fnm09", fnDestr.length === 3);
    // ...and only when it IS the initializer: node names nothing in `f(function(){})`.
    function s48id(x) { return x }
    var fnVia = s48id(function () { return 2 });
    check("fnm10", fnVia.name === "" && fnVia.length === 0);
    var fnAssigned;
    fnAssigned = function () { return 3 }
    check("fnm11", fnAssigned.name === "fnAssigned");
    var s48obj = {
        mExpr: function () { return 1 },
        mShort(a) { return a },
        mArrow: (a, b) => a,
        deep: { inner: function () { return 4 } }
    }
    check("fnm12", s48obj.mExpr.name === "mExpr" && s48obj.mExpr.length === 0);
    check("fnm13", s48obj.mShort.name === "mShort" && s48obj.mShort.length === 1);
    check("fnm14", s48obj.mArrow.name === "mArrow" && s48obj.mArrow.length === 2);
    check("fnm15", s48obj.deep.inner.name === "inner");
    async function s48async(a, b) { return a }
    function* s48gen(a) { yield a }
    check("fnm16", s48async.name === "s48async" && s48async.length === 2);
    check("fnm17", s48gen.name === "s48gen" && s48gen.length === 1);
    class S51C {
        constructor(a) { this.a = a }
        meth(x) { return x }
        static stat(a, b) { return a }
        async am(a) { return a }
        *gm() { yield 1 }
    }
    var s48i = new S51C(1);
    // A method read as a VALUE answers the method's own name and arity: the receiver
    // arrives as a hidden first argument here and is not a declared parameter.
    check("fnm18", s48i.meth.name === "meth" && s48i.meth.length === 1);
    check("fnm19", S51C.stat.name === "stat" && S51C.stat.length === 2);
    check("fnm20", s48i.am.name === "am" && s48i.am.length === 1);
    check("fnm21", s48i.gm.name === "gm" && s48i.gm.length === 0);
    // A BOUND function: "bound " + the target's name, and the target's length less the
    // arguments already supplied, floored at 0.
    var fnBound = fnDecl.bind(null);
    var fnBound2 = fnNamedExpr.bind(null, 1);
    check("fnm22", fnBound.name === "bound fnDecl" && fnBound.length === 1);
    check("fnm23", fnBound2.name === "bound fnInner" && fnBound2.length === 1);
    check("fnm24", fnRest.bind(null, 1, 2, 3).length === 0);
    // Both are OWN properties, and NON-ENUMERABLE: getOwnPropertyNames reports them,
    // Object.keys and for-in do not. The list is asserted on an ARROW, where node's own
    // answer is exactly ["length","name"]; a function DECLARATION has three more there
    // (`arguments`, `caller`, `prototype`) and no engine here can answer any of them -
    // there is no prototype chain and no call-site frame in this value model - so for
    // that shape only the two names' PRESENCE is asserted, which is true in node too.
    var g = Object.getOwnPropertyNames(fnZero);
    check("fnm25", g.length === 2 && g[0] === "length" && g[1] === "name");
    var gd = Object.getOwnPropertyNames(fnDecl);
    check("fnm26", gd.indexOf("length") >= 0 && gd.indexOf("name") >= 0);
    check("fnm27", Object.keys(fnDecl).length === 0);
    check("fnm28", fnDecl.hasOwnProperty("name") && fnDecl.hasOwnProperty("length"));
    check("fnm29", !fnDecl.propertyIsEnumerable("name"));
    var fnKeys = [];
    for (var fk in fnDecl) { fnKeys.push(fk) }
    check("fnm30", fnKeys.length === 0);
    // Both are NON-WRITABLE: the write is dropped, silently, as it is in sloppy node.
    fnDecl.name = "zzz";
    fnDecl.length = 99;
    check("fnm31", fnDecl.name === "fnDecl" && fnDecl.length === 1);
    // ...and a function can still carry own properties of its own beside them.
    fnZero.tag = 7;
    var g2 = Object.getOwnPropertyNames(fnZero);
    check("fnm32", fnZero.tag === 7 && g2.length === 3 && g2[2] === "tag");
    check("fnm33", fnZero.hasOwnProperty("tag") && fnZero.propertyIsEnumerable("tag"));
    check("fnm34", Object.keys(fnZero).length === 1 && Object.keys(fnZero)[0] === "tag");
    // Closures built at ONE definition site in a loop each carry their own pair, and
    // binding one of them names only the bound value.
    var fnLoop = []
    for (var fi = 0; fi < 300; fi++) { fnLoop.push(function looped(a, b) { return a + b }) }
    var fnLoopB = fnLoop[150].bind(null, 1)
    check("fnm35", fnLoop[0].name === "looped" && fnLoop[299].length === 2 && fnLoop[150] !== fnLoop[151]);
    check("fnm36", fnLoopB.name === "bound looped" && fnLoopB.length === 1 && fnLoopB(2) === 3 && fnLoop[150].name === "looped");
    // ----- a named function EXPRESSION sees its OWN NAME (ES2023 15.2.5) -----
    // The spec's funcEnv - one scope, one binding of the name to the function object,
    // between the definition site and the function's own scope - is what all nine rows
    // below come out of; every value was read off node 24 first.
    var s48fact = function fact(n) { return n < 2 ? 1 : n * fact(n - 1) }
    check("fns01", s48fact(5) === 120);
    // The name does NOT leak into the enclosing scope.
    var s48hid = function hidden48() { return 1 }
    check("fns02", typeof hidden48 === "undefined");
    // An outer variable of the same name is SHADOWED inside and untouched outside.
    var shad48 = 7;
    var s48shad = function shad48() { return typeof shad48 }
    check("fns03", s48shad() === "function" && shad48 === 7);
    // A DECLARATION is unchanged: it binds in the ENCLOSING scope, and only there.
    check("fns04", s48decl(5) === 120 && typeof s48decl === "function");
    // A `var` of the same name IN THE BODY shadows the self binding, so this is 2.
    var s48var = function v48() { var v48 = 1; v48 = 2; return v48 }
    check("fns05", s48var() === 2);
    // The name IS the function - one object, bound after it exists.
    var s48id = function self48() { return self48 === s48id }
    check("fns06", s48id() === true);
    // A PARAMETER of the same name wins over the self binding.
    var s48par = function p48(p48) { return p48 }
    check("fns07", s48par(42) === 42);
    // ...and a NESTED function still sees it.
    var s48nest = function o48() { return (function () { return typeof o48 })() }
    check("fns08", s48nest() === "function");
    // A generator function EXPRESSION binds its own name too, which is what lets it
    // delegate to itself. Driven by next() rather than by spread: spreading a generator
    // answers an empty array in both halves (a separate gap).
    var s48gen = function* gen48(n) { yield n; if (n > 0) { yield* gen48(n - 1) } }
    var s48out = "";
    var s48it = s48gen(2);
    var s48r = s48it.next();
    while (!s48r.done) { s48out = s48out + s48r.value; s48r = s48it.next() }
    check("fns09", s48out === "210");
    // NOT asserted, and deliberately: an ASSIGNMENT to the name inside the body. node
    // ignores it silently in sloppy code and raises a TypeError in strict code, because
    // the binding is IMMUTABLE; the write lands on the funcEnv slot in BOTH halves here,
    // so they agree and an assertion would pin the divergence. The mechanism it would
    // take is written up at fnSelf in both grammars.
}
// The declaration s48decl is used by fns04 from inside s48, and a declaration binds in
// the scope AROUND the function it is written in - so it is a top-level sibling here,
// which is also what makes it the control for the whole group: it must keep working
// exactly as it did.
function s48decl(n) { return n < 2 ? 1 : n * s48decl(n - 1) }

// ===== SECTION 49: an escaped identifier is the name it spells =====
// '\u0061bc' and 'abc' are two spellings of ONE name, so every scope, property, class and
// parameter table must see the decoded name. Values read off node 24.
function s49() {
    var \u0061bc = 21;
    check("esc01", abc === 21);
    var x\u{79}z = 22;
    check("esc02", \u0078yz === 22 && typeof xyz === "number");
    var uo = { \u006b: 23, k\u0032: 24 };
    check("esc03", uo.k === 23 && uo.\u006b\u0032 === 24);
    function \u0067(\u0070: number): number { return p + 1 }
    check("esc04", g(1) === 2);
    class \u0055C { m\u0065(): number { return 25 } }
    check("esc05", new UC().me() === 25);
}

// ===== SECTION 50: an ITERATOR spreads and destructures =====
// A spread and an array pattern read an iterator through next(); an array pattern takes
// only as many values as it has positions and then CLOSES the iterator, so destructuring
// an endless generator terminates and runs its finally. Values read off node 24.
function* s50g(): Generator<number, void, void> { yield 1; yield 2; }
let s50log: string[] = [];
function* s50nat(): Generator<number, void, void> { try { let i = 0; while (true) { yield i++; } } finally { s50log.push("closed"); } }
function s50add(a: number, b: number): number { return a + b; }
function s50() {
    check("its01", [...s50g()].length === 2);
    check("its02", [0, ...s50g(), 3].join(",") === "0,1,2,3");
    check("its03", (s50add as any)(...s50g()) === 3 && Math.max(...s50g()) === 2);
    const [x, y] = s50g();
    check("its04", x === 1 && y === 2);
    const [p, ...rest] = s50g();
    check("its05", p === 1 && rest.length === 1);
    const [a, b, c] = s50nat();
    check("its06", (a as number) + (b as number) + (c as number) === 3 && s50log.join(",") === "closed");
}

// ===== END SECTIONS =====

function main() {
    s44() // SECTION-CALL 44;
    s45() // SECTION-CALL 45;
    s46() // SECTION-CALL 46;
    s47() // SECTION-CALL 47;
    s48() // SECTION-CALL 48;
    s49() // SECTION-CALL 49;
    s50() // SECTION-CALL 50;
    s48() // SECTION-CALL 48;
    println("full: " + checks + " checks, " + failures + " failures");
    return failures;
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
    s38(); // SECTION-CALL 38
    s39(); // SECTION-CALL 39
    s40(); // SECTION-CALL 40
    s41(); // SECTION-CALL 41
    s42(); // SECTION-CALL 42
    s43(); // SECTION-CALL 43
    s44(); // SECTION-CALL 44
    s45(); // SECTION-CALL 45
    s46(); // SECTION-CALL 46
    s47(); // SECTION-CALL 47
    s48(); // SECTION-CALL 48
    s49(); // SECTION-CALL 49
    s50(); // SECTION-CALL 50
    println("full: " + checks + " checks, " + failures + " failures");
    return failures;
}
