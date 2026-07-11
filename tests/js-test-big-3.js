// Self-checking test for the JavaScript interpreter (js-interpreter.abnf) and the
// LLVM-IR compiler (js-to-llvm-ir.abnf). THEME: object-oriented + functional, the
// language's signature features.
//
// Part one is a class-based geometry kit: an abstract Shape base with fields, a
// constructor, instance and static methods, three subclasses using extends / super /
// super.method / overrides, polymorphic dispatch through an array of shapes, method
// chaining that returns 'this', and a static instance counter. Part two is a Fraction
// (rational number) class doing exact integer arithmetic with reduction by GCD. Part
// three is a functional toolkit (map / filter / reduce / compose / pipe / curry /
// partial / memoize / once) built from arrow functions, closures and template
// literals. main() returns the number of failed checks; exit 0 means all passed. Only
// genuinely implemented constructs are used, so both grammars pass and the compiler IR
// is byte-identical.

var failures = 0;
function check(cond) { if (!cond) { failures = failures + 1; } }

// ===== Part one: an OO geometry kit =====

var shapeCount = 0;   // module-level instance counter, bumped by the base constructor

class Shape {
    constructor(name) {
        this.name = name;
        shapeCount = shapeCount + 1;
    }
    area() { return 0; }            // overridden by every concrete subclass
    perimeter() { return 0; }
    describe() { return `${this.name}[area=${this.area()},perim=${this.perimeter()}]`; }
    isLargerThan(other) { return this.area() > other.area(); }
    static kinds() { return 3; }    // a static method on the base
}

class Rectangle extends Shape {
    constructor(w, h) {
        super("rectangle");
        this.w = w;
        this.h = h;
    }
    area() { return this.w * this.h; }
    perimeter() { return 2 * (this.w + this.h); }
}

// A Square specializes Rectangle and reuses the parent's area/perimeter through super().
class Square extends Rectangle {
    constructor(side) {
        super(side, side);
        this.name = "square";
    }
    // A method that reaches the grandparent-level implementation via the parent.
    describeSquare() { return super.describe() + "*"; }
}

class RightTriangle extends Shape {
    constructor(a, b, c) {
        super("triangle");
        this.a = a;
        this.b = b;
        this.c = c;   // hypotenuse, supplied so the perimeter stays integer
    }
    area() { return Math.floor(this.a * this.b / 2); }
    perimeter() { return this.a + this.b + this.c; }
}

function testShapes() {
    shapeCount = 0;
    var r = new Rectangle(3, 4);
    check(r.name === "rectangle");
    check(r.area() === 12);
    check(r.perimeter() === 14);
    check(r.describe() === "rectangle[area=12,perim=14]");

    var sq = new Square(5);
    check(sq.name === "square");
    check(sq.area() === 25);              // inherited Rectangle.area
    check(sq.perimeter() === 20);          // inherited Rectangle.perimeter
    check(sq.describe() === "square[area=25,perim=20]");
    check(sq.describeSquare() === "square[area=25,perim=20]*");   // super.describe()

    var t = new RightTriangle(3, 4, 5);
    check(t.area() === 6);
    check(t.perimeter() === 12);
    check(t.describe() === "triangle[area=6,perim=12]");

    check(shapeCount === 3);               // base constructor ran three times
    check(Shape.kinds() === 3);            // static method

    // Inherited method using overridden area() through 'this'.
    check(sq.isLargerThan(r) === true);    // 25 > 12
    check(r.isLargerThan(sq) === false);
    check(t.isLargerThan(r) === false);    // 6 < 12
}

// Polymorphic dispatch: one loop over a heterogeneous array of shapes.
function testPolymorphism() {
    var shapes = [new Rectangle(3, 4), new Square(5), new RightTriangle(6, 8, 10)];
    var totalArea = 0;
    var totalPerim = 0;
    var names = [];
    for (var i = 0; i < shapes.length; i++) {
        totalArea = totalArea + shapes[i].area();
        totalPerim = totalPerim + shapes[i].perimeter();
        names.push(shapes[i].name);
    }
    check(totalArea === 12 + 25 + 24);     // 61
    check(totalPerim === 14 + 20 + 24);    // 58
    check(names.join(",") === "rectangle,square,triangle");

    // Find the largest shape by polymorphic area().
    var largest = shapes[0];
    for (var j = 1; j < shapes.length; j++) {
        if (shapes[j].isLargerThan(largest)) { largest = shapes[j]; }
    }
    check(largest.name === "square");      // 25 is the max here
    check(largest.area() === 25);
}

// ===== Part two: a Fraction class doing exact rational arithmetic =====

function gcd(a, b) {
    a = a < 0 ? -a : a;
    b = b < 0 ? -b : b;
    while (b !== 0) { var t = b; b = a % b; a = t; }
    return a;
}

class Fraction {
    constructor(num, den) {
        if (den < 0) { num = -num; den = -den; }   // keep the denominator positive
        var g = gcd(num, den);
        if (g === 0) { g = 1; }
        this.num = Math.floor(num / g);
        this.den = Math.floor(den / g);
    }
    add(o) { return new Fraction(this.num * o.den + o.num * this.den, this.den * o.den); }
    sub(o) { return new Fraction(this.num * o.den - o.num * this.den, this.den * o.den); }
    mul(o) { return new Fraction(this.num * o.num, this.den * o.den); }
    equals(o) { return this.num === o.num && this.den === o.den; }
    compareTo(o) {
        var left = this.num * o.den;
        var right = o.num * this.den;
        if (left < right) { return -1; }
        if (left > right) { return 1; }
        return 0;
    }
    text() { return `${this.num}/${this.den}`; }
    static whole(n) { return new Fraction(n, 1); }
}

function testFractions() {
    var half = new Fraction(1, 2);
    var third = new Fraction(1, 3);
    check(half.text() === "1/2");

    // Reduction: 2/4 -> 1/2, 6/8 -> 3/4, sign moved onto the numerator.
    check(new Fraction(2, 4).text() === "1/2");
    check(new Fraction(6, 8).text() === "3/4");
    check(new Fraction(3, -6).text() === "-1/2");
    check(new Fraction(0, 5).text() === "0/1");

    // 1/2 + 1/3 = 5/6, 1/2 - 1/3 = 1/6, 1/2 * 1/3 = 1/6.
    check(half.add(third).text() === "5/6");
    check(half.sub(third).text() === "1/6");
    check(half.mul(third).text() === "1/6");

    // 1/2 + 1/2 = 1/1.
    check(half.add(half).text() === "1/1");
    check(Fraction.whole(3).text() === "3/1");

    // Comparisons.
    check(half.compareTo(third) === 1);
    check(third.compareTo(half) === -1);
    check(half.compareTo(new Fraction(2, 4)) === 0);
    check(half.equals(new Fraction(50, 100)) === true);

    // Telescoping sum 1/1 - 1/(n+1) = sum_{k=1..n} 1/(k*(k+1)).
    var acc = new Fraction(0, 1);
    for (var k = 1; k <= 5; k++) {
        acc = acc.add(new Fraction(1, k * (k + 1)));
    }
    check(acc.text() === "5/6");     // 1 - 1/6
}

// ===== Part three: a functional toolkit =====

function map(arr, f) {
    var out = [];
    for (var i = 0; i < arr.length; i++) { out.push(f(arr[i])); }
    return out;
}
function filter(arr, pred) {
    var out = [];
    for (var i = 0; i < arr.length; i++) { if (pred(arr[i])) { out.push(arr[i]); } }
    return out;
}
function reduce(arr, f, init) {
    var acc = init;
    for (var i = 0; i < arr.length; i++) { acc = f(acc, arr[i]); }
    return acc;
}
function range(n) {
    var out = [];
    for (var i = 1; i <= n; i++) { out.push(i); }
    return out;
}

function testFunctional() {
    var nums = range(5);   // [1,2,3,4,5]

    var squares = map(nums, x => x * x);
    check(squares.join(",") === "1,4,9,16,25");

    var evens = filter(nums, x => x % 2 === 0);
    check(evens.join(",") === "2,4");

    var sum = reduce(nums, (a, b) => a + b, 0);
    check(sum === 15);
    var product = reduce(nums, (a, b) => a * b, 1);
    check(product === 120);

    // Chained transforms: sum of squares of the even numbers.
    var sumEvenSquares = reduce(map(filter(nums, x => x % 2 === 0), x => x * x), (a, b) => a + b, 0);
    check(sumEvenSquares === 4 + 16);   // 20

    // Function composition and a pipeline.
    var inc = x => x + 1;
    var dbl = x => x * 2;
    var compose = (f, g) => x => f(g(x));
    var incThenDbl = compose(dbl, inc);   // dbl(inc(x))
    check(incThenDbl(5) === 12);           // (5+1)*2
    var pipe3 = x => dbl(inc(dbl(x)));
    check(pipe3(3) === 14);                // ((3*2)+1)*2

    // Currying and partial application.
    var add3 = a => b => c => a + b + c;
    check(add3(1)(2)(3) === 6);
    var add10 = add3(10);
    check(add10(20)(30) === 60);

    function partial(f, a) { return function(b) { return f(a, b); }; }
    var addPair = (a, b) => a + b;
    var plus100 = partial(addPair, 100);
    check(plus100(23) === 123);
}

// Closures with private mutable state: memoization and a once-guard.
function testClosures() {
    // A memoized, call-counting slow function.
    var calls = 0;
    function slowSquare(n) { calls = calls + 1; return n * n; }
    function memoize(f) {
        var cache = {};
        return function(n) {
            if (cache[n] === undefined) { cache[n] = f(n); }
            return cache[n];
        };
    }
    var fast = memoize(slowSquare);
    check(fast(4) === 16);
    check(fast(4) === 16);      // cache hit
    check(fast(5) === 25);
    check(fast(4) === 16);      // still cached
    check(calls === 2);        // slowSquare ran only for 4 and 5

    // A 'once' wrapper: the underlying function runs at most one time.
    function once(f) {
        var done = false;
        var value;
        return function() {
            if (!done) { value = f(); done = true; }
            return value;
        };
    }
    var initCount = 0;
    var init = once(function() { initCount = initCount + 1; return 42; });
    check(init() === 42);
    check(init() === 42);
    check(init() === 42);
    check(initCount === 1);

    // A counter factory: independent private state per counter.
    function makeCounter(start) {
        var n = start;
        return { next: function() { n = n + 1; return n; },
                 value: function() { return n; } };
    }
    var c1 = makeCounter(0);
    var c2 = makeCounter(100);
    check(c1.next() === 1);
    check(c1.next() === 2);
    check(c2.next() === 101);
    check(c1.value() === 2);
    check(c2.value() === 101);   // c2 is independent of c1
}

function main() {
    testShapes();
    testPolymorphism();
    testFractions();
    testFunctional();
    testClosures();
    return failures;
}
