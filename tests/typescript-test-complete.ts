// Self-checking test for the newly implemented TypeScript features:
//   - classes: fields (with initializers), a constructor, instance & static methods,
//     'this', object creation with 'new'; access modifiers (public/private/readonly/
//     static) and every type annotation are erased; method dispatch goes through the
//     shared __class machinery (js_mcall on the compiler, mcall in the interpreter);
//   - single inheritance: 'extends', an explicit super(...) constructor call, inherited
//     methods, method overriding, and super.method() calls;
//   - the for-in loop over an object's own keys (via js_keys), in insertion order.
//
// It counts failed checks and returns that count; exit 0 means every check passed. Both
// engines (goja and -frozen) and both the interpreter and the compiler must agree, so
// the byte-identical compiler outputs and the interpreter runs all end at 0.

let failures: number = 0;

function check(cond: boolean): void {
    if (!cond) { failures = failures + 1; }
}

// --- A basic class: fields with initializers, a constructor, instance methods, this. ---
class Counter {
    private count: number = 0;     // field with initializer; the modifier is erased
    readonly step: number;         // field assigned by the constructor

    constructor(step: number) {
        this.step = step;
    }

    tick(): void {
        this.count = this.count + this.step;
    }

    get(): number {
        return this.count;
    }

    // A method that calls other methods through 'this'.
    tickTwice(): number {
        this.tick();
        this.tick();
        return this.get();
    }
}

function testBasicClass(): void {
    const c: Counter = new Counter(3);
    check(c.get() === 0);
    c.tick();
    check(c.get() === 3);
    check(c.tickTwice() === 9);
    check(c.step === 3);
    // A second instance is independent of the first.
    const d: Counter = new Counter(10);
    check(d.get() === 0);
    d.tick();
    check(d.get() === 10);
    check(c.get() === 9);       // c is unaffected by d
}

// --- Static fields and a static factory method, plus instance methods and fields. ---
class Point {
    x: number;
    y: number;
    static count: number = 0;      // static field with an initializer

    constructor(x: number, y: number) {
        this.x = x;
        this.y = y;
    }

    normSquared(): number {
        return this.x * this.x + this.y * this.y;
    }

    static make(x: number, y: number): Point {
        return new Point(x, y);
    }
}

function testStatics(): void {
    const p: Point = Point.make(3, 4);           // static method returning an instance
    check(p.normSquared() === 25);
    check(Point.count === 0);                     // static field read
    Point.count = 5;                              // static field write
    check(Point.count === 5);
    const q: Point = new Point(1, 2);
    check(q.normSquared() === 5);
    check(p.normSquared() === 25);                // q did not disturb p
}

// --- Single inheritance: extends, super(...) constructor, method inheritance,
//     overriding, and super.method() calls. ---
class Animal {
    name: string;

    constructor(name: string) {
        this.name = name;
    }

    speak(): string {
        return this.name + " makes a sound";
    }

    describe(): string {
        return "Animal: " + this.name;
    }
}

class Dog extends Animal {
    readonly breed: string;

    constructor(name: string, breed: string) {
        super(name);                 // delegate to the parent constructor
        this.breed = breed;
    }

    // Override speak, reusing the parent version through super.
    speak(): string {
        return super.speak() + " (woof)";
    }

    // A new method that reaches an inherited method through this.
    fullDescribe(): string {
        return this.describe() + ", breed " + this.breed;
    }
}

function testInheritance(): void {
    const a: Animal = new Animal("generic");
    check(a.speak() === "generic makes a sound");

    const d: Dog = new Dog("Rex", "Lab");
    check(d.name === "Rex");                                 // inherited field via super()
    check(d.breed === "Lab");
    check(d.speak() === "Rex makes a sound (woof)");         // override + super.speak()
    check(d.describe() === "Animal: Rex");                   // inherited method
    check(d.fullDescribe() === "Animal: Rex, breed Lab");    // this.describe() dispatch
}

// --- for-in over an object's own keys (insertion order, via js_keys). ---
function testForIn(): void {
    const scores: { [k: string]: number } = { alice: 10, bob: 20, carol: 30 };
    let keyConcat: string = "";
    let total: number = 0;
    for (const k in scores) {
        keyConcat = keyConcat + k + ",";
        total = total + scores[k];
    }
    check(keyConcat === "alice,bob,carol,");
    check(total === 60);

    // A dynamically built object: keys are created by assignment in first-seen order.
    const counts: { [k: string]: number } = {};
    const words: string[] = ["a", "b", "a", "c", "b", "a"];
    for (const w of words) {
        if (counts[w] === undefined) { counts[w] = 0; }
        counts[w] = counts[w] + 1;
    }
    let keys: string = "";
    let sum: number = 0;
    for (const key in counts) {
        keys = keys + key;
        sum = sum + counts[key];
    }
    check(keys === "abc");          // first-seen order: a, then b, then c
    check(sum === 6);
    check(counts["a"] === 3);

    // for-in over an instance enumerates its own fields (never methods or __class).
    const p: Point = new Point(7, 8);
    let fieldNames: string = "";
    for (const f in p) { fieldNames = fieldNames + f + " "; }
    check(fieldNames === "x y ");
}

function main(): number {
    testBasicClass();
    testStatics();
    testInheritance();
    testForIn();
    return failures;
}
