// Self-checking TypeScript test #big-3: OBJECT-ORIENTED + FUNCTIONAL FEATURES.
//
// Themes: a polymorphic class hierarchy (a Shape base with Circle / Rectangle / Square /
// Triangle subclasses, method overriding, super(...) chaining, static instance counters,
// static factories), interfaces and enums describing the shapes, and a full functional
// toolkit written by hand (map / filter / reduce / forEach / compose / pipe / curry,
// closures, memoization, an Optional/Maybe, and a fluent method-chaining builder).
// main() returns the failure count; 0 means every engine agrees.

let failures: number = 0;

function check(cond: boolean, _label: string): void {
    if (!cond) { failures = failures + 1; }
}

function approxEqual(a: number, b: number): boolean {
    return Math.abs(a - b) < 0.0001;
}

const PI: number = 3.141592653589793;

// ---- interfaces (type-only, erased) ----
interface Shaped {
    area(): number;
    perimeter(): number;
    kind(): ShapeKind;
}

enum ShapeKind {
    Circle,
    Rectangle,
    Square,
    Triangle,
}

// ---- a class hierarchy with polymorphism ----
class Shape implements Shaped {
    static instances: number = 0;

    constructor() {
        Shape.instances = Shape.instances + 1;
    }

    // Overridden by every subclass.
    area(): number { return 0; }
    perimeter(): number { return 0; }
    kind(): ShapeKind { return ShapeKind.Circle; }

    // A shared, non-overridden method that dispatches through 'this' to the override.
    describe(): string {
        return `shape#${this.kind()} area=${this.area()}`;
    }

    biggerThan(other: Shape): boolean {
        return this.area() > other.area();
    }
}

class Circle extends Shape {
    private radius: number;

    constructor(radius: number) {
        super();
        this.radius = radius;
    }

    area(): number { return PI * this.radius * this.radius; }
    perimeter(): number { return 2 * PI * this.radius; }
    kind(): ShapeKind { return ShapeKind.Circle; }
}

class Rectangle extends Shape {
    protected width: number;
    protected height: number;

    constructor(width: number, height: number) {
        super();
        this.width = width;
        this.height = height;
    }

    area(): number { return this.width * this.height; }
    perimeter(): number { return 2 * (this.width + this.height); }
    kind(): ShapeKind { return ShapeKind.Rectangle; }
}

// Square extends Rectangle: a two-level inheritance chain reaching Shape's ctor.
class Square extends Rectangle {
    constructor(side: number) {
        super(side, side);
    }

    // Override kind but reuse Rectangle's area/perimeter through inheritance.
    kind(): ShapeKind { return ShapeKind.Square; }
}

class Triangle extends Shape {
    private base: number;
    private heightLen: number;
    private sideA: number;
    private sideB: number;

    constructor(base: number, heightLen: number, sideA: number, sideB: number) {
        super();
        this.base = base;
        this.heightLen = heightLen;
        this.sideA = sideA;
        this.sideB = sideB;
    }

    area(): number { return (this.base * this.heightLen) / 2; }
    perimeter(): number { return this.base + this.sideA + this.sideB; }
    kind(): ShapeKind { return ShapeKind.Triangle; }
}

function testPolymorphism(): void {
    Shape.instances = 0;
    const shapes: Shape[] = [
        new Circle(2),
        new Rectangle(3, 4),
        new Square(5),
        new Triangle(6, 4, 5, 5),
    ];
    check(Shape.instances === 4, "instance-count");

    // Each area, computed polymorphically.
    check(approxEqual(shapes[0].area(), PI * 4), "circle-area");
    check(shapes[1].area() === 12, "rect-area");
    check(shapes[2].area() === 25, "square-area");
    check(shapes[3].area() === 12, "triangle-area");

    // Perimeters.
    check(approxEqual(shapes[0].perimeter(), 4 * PI), "circle-perimeter");
    check(shapes[1].perimeter() === 14, "rect-perimeter");
    check(shapes[2].perimeter() === 20, "square-perimeter");
    check(shapes[3].perimeter() === 16, "triangle-perimeter");

    // Kinds (Square overrides Rectangle's kind).
    check(shapes[0].kind() === ShapeKind.Circle, "circle-kind");
    check(shapes[1].kind() === ShapeKind.Rectangle, "rect-kind");
    check(shapes[2].kind() === ShapeKind.Square, "square-kind");
    check(shapes[3].kind() === ShapeKind.Triangle, "triangle-kind");

    // A Square IS a Rectangle: its inherited area works.
    const sq: Square = new Square(4);
    check(sq.area() === 16, "square-is-rectangle-area");
    check(sq.perimeter() === 16, "square-is-rectangle-perimeter");

    // describe() reaches the overridden kind/area through 'this'.
    check(shapes[2].describe() === "shape#2 area=25", "square-describe");

    // Sum of integer areas via a manual fold over the polymorphic array.
    let totalInt: number = 0;
    for (const s of shapes) {
        if (s.kind() !== ShapeKind.Circle) { totalInt = totalInt + s.area(); }
    }
    check(totalInt === 12 + 25 + 12, "poly-area-sum");

    // biggerThan comparison.
    check(shapes[2].biggerThan(shapes[1]), "square-bigger-rect");
    check(!shapes[1].biggerThan(shapes[2]), "rect-not-bigger-square");
}

// ---- functional toolkit (all generic signatures are erased) ----

function mapArr<T, U>(xs: T[], f: (x: T) => U): U[] {
    const out: U[] = [];
    for (const x of xs) { out.push(f(x)); }
    return out;
}

function filterArr<T>(xs: T[], pred: (x: T) => boolean): T[] {
    const out: T[] = [];
    for (const x of xs) { if (pred(x)) { out.push(x); } }
    return out;
}

function reduceArr<T, U>(xs: T[], f: (acc: U, x: T) => U, init: U): U {
    let acc: U = init;
    for (const x of xs) { acc = f(acc, x); }
    return acc;
}

function forEachArr<T>(xs: T[], f: (x: T) => void): void {
    for (const x of xs) { f(x); }
}

// compose(f, g)(x) = f(g(x)); pipe is the reverse order.
function compose<A, B, C>(f: (b: B) => C, g: (a: A) => B): (a: A) => C {
    return (a: A): C => f(g(a));
}
function pipe<A, B, C>(f: (a: A) => B, g: (b: B) => C): (a: A) => C {
    return (a: A): C => g(f(a));
}

// A curried three-argument adder (expression bodies; the return-type layer is erased).
function curryAdd3(a: number): (b: number) => (c: number) => number {
    return (b: number) => (c: number): number => a + b + c;
}

function arraysEqual(a: number[], b: number[]): boolean {
    if (a.length !== b.length) { return false; }
    for (let i: number = 0; i < a.length; i++) {
        if (a[i] !== b[i]) { return false; }
    }
    return true;
}

function testFunctional(): void {
    const nums: number[] = [1, 2, 3, 4, 5, 6];

    const doubled: number[] = mapArr(nums, (x: number): number => x * 2);
    check(arraysEqual(doubled, [2, 4, 6, 8, 10, 12]), "map-double");

    const evens: number[] = filterArr(nums, (x: number): boolean => x % 2 === 0);
    check(arraysEqual(evens, [2, 4, 6]), "filter-evens");

    const sum: number = reduceArr(nums, (a: number, x: number): number => a + x, 0);
    check(sum === 21, "reduce-sum");

    const product: number = reduceArr(nums, (a: number, x: number): number => a * x, 1);
    check(product === 720, "reduce-product");

    const maxv: number = reduceArr(nums, (a: number, x: number): number => Math.max(a, x), 0);
    check(maxv === 6, "reduce-max");

    // Chained transformations: keep evens, square them, sum.
    const evenSquareSum: number = reduceArr(
        mapArr(filterArr(nums, (x: number): boolean => x % 2 === 0),
               (x: number): number => x * x),
        (a: number, x: number): number => a + x, 0);
    check(evenSquareSum === 4 + 16 + 36, "map-filter-reduce-chain");

    // forEach with a mutating closure.
    let seen: number = 0;
    forEachArr(nums, (x: number): void => { seen = seen + x; });
    check(seen === 21, "foreach-sum");

    // compose / pipe.
    const inc: (n: number) => number = (n: number): number => n + 1;
    const dbl: (n: number) => number = (n: number): number => n * 2;
    check(compose(inc, dbl)(5) === 11, "compose-inc-dbl");   // inc(dbl(5)) = 11
    check(pipe(inc, dbl)(5) === 12, "pipe-inc-dbl");         // dbl(inc(5)) = 12

    // Curried application.
    check(curryAdd3(1)(2)(3) === 6, "curry-add3");
    const add10: (b: number) => (c: number) => number = curryAdd3(10);
    check(add10(20)(30) === 60, "curry-partial");
}

// ---- closures ----

function makeCounter(start: number): () => number {
    let n: number = start;
    return (): number => { n = n + 1; return n; };
}

function makeAccumulator(): (x: number) => number {
    let total: number = 0;
    return (x: number): number => { total = total + x; return total; };
}

// Memoize a unary numeric function; a shared call counter proves the cache works.
let fibCalls: number = 0;
function memoize(f: (n: number) => number): (n: number) => number {
    const cache: { [k: string]: number } = {};
    return (n: number): number => {
        const key: string = "" + n;
        if (cache[key] === undefined) { cache[key] = f(n); }
        return cache[key];
    };
}

function testClosures(): void {
    const c1: () => number = makeCounter(0);
    const c2: () => number = makeCounter(100);
    check(c1() === 1, "counter1-a");
    check(c1() === 2, "counter1-b");
    check(c2() === 101, "counter2-a");     // independent state
    check(c1() === 3, "counter1-c");
    check(c2() === 102, "counter2-b");

    const acc: (x: number) => number = makeAccumulator();
    check(acc(10) === 10, "acc-10");
    check(acc(5) === 15, "acc-15");
    check(acc(-3) === 12, "acc-12");

    // Memoized squarer: the second call for the same key does not re-invoke f.
    let calls: number = 0;
    const slowSquare: (n: number) => number = (n: number): number => { calls = calls + 1; return n * n; };
    const fast: (n: number) => number = memoize(slowSquare);
    check(fast(4) === 16, "memo-a");
    check(fast(4) === 16, "memo-b");
    check(fast(5) === 25, "memo-c");
    check(calls === 2, "memo-call-count");   // 4 and 5 computed once each
}

// ---- an Optional/Maybe encoded as a plain object ----
// The type layer models it as a discriminated union; at run time it is {present, value}.
type Maybe = { present: boolean; value: number };

function some(v: number): Maybe {
    return { present: true, value: v };
}
function none(): Maybe {
    return { present: false, value: 0 };
}
function mapMaybe(m: Maybe, f: (x: number) => number): Maybe {
    if (!m.present) { return m; }
    return some(f(m.value));
}
function getOrElse(m: Maybe, fallback: number): number {
    return m.present ? m.value : fallback;
}

function testMaybe(): void {
    const a: Maybe = some(21);
    const b: Maybe = mapMaybe(a, (x: number): number => x * 2);
    check(b.present, "maybe-present");
    check(b.value === 42, "maybe-value");
    check(getOrElse(b, -1) === 42, "maybe-getorelse-present");

    const n: Maybe = none();
    const n2: Maybe = mapMaybe(n, (x: number): number => x * 2);
    check(!n2.present, "maybe-none-preserved");
    check(getOrElse(n2, 99) === 99, "maybe-getorelse-fallback");
}

// ---- a fluent method-chaining builder ----
class TextBuilder {
    private parts: string[] = [];

    append(s: string): TextBuilder {
        this.parts.push(s);
        return this;                 // return this for chaining
    }

    appendLine(s: string): TextBuilder {
        this.parts.push(s);
        this.parts.push("\n");
        return this;
    }

    repeat(s: string, n: number): TextBuilder {
        for (let i: number = 0; i < n; i++) { this.parts.push(s); }
        return this;
    }

    build(): string {
        return this.parts.join("");
    }

    length(): number {
        return this.build().length;
    }
}

function testBuilder(): void {
    const text: string = new TextBuilder()
        .append("Hello")
        .append(", ")
        .append("World")
        .append("!")
        .build();
    check(text === "Hello, World!", "builder-basic");

    const chant: string = new TextBuilder()
        .repeat("ab", 3)
        .append("-")
        .repeat("z", 2)
        .build();
    check(chant === "ababab-zz", "builder-repeat");

    const doc: TextBuilder = new TextBuilder().appendLine("line1").appendLine("line2");
    check(doc.build() === "line1\nline2\n", "builder-lines");
    check(doc.length() === 12, "builder-length");
}

// ---- generic pair helpers (erased generics) ----
function makePair<A, B>(a: A, b: B): { first: A; second: B } {
    return { first: a, second: b };
}
function swapPair<A, B>(p: { first: A; second: B }): { first: B; second: A } {
    return { first: p.second, second: p.first };
}

function testGenerics(): void {
    const p: { first: number; second: string } = makePair(1, "one");
    check(p.first === 1, "pair-first");
    check(p.second === "one", "pair-second");
    const s = swapPair(p);
    check(s.first === "one", "swap-first");
    check(s.second === 1, "swap-second");
}

function main(): number {
    testPolymorphism();
    testFunctional();
    testClosures();
    testMaybe();
    testBuilder();
    testGenerics();
    return failures;
}
