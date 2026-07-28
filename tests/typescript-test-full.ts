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
    println("full: " + checks + " checks, " + failures + " failures");
    return failures;
}
