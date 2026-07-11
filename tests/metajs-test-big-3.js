/* MetaJS big test 3 - Closures, higher-order and object-oriented features.
 *
 * The signature style of the language: real lexical closures with private
 * state, a full higher-order toolkit built from scratch (map, filter, reduce,
 * every, some, find, count, zipWith), function composition and pipelines,
 * currying and partial application, memoization, lazy iterators, and the
 * closure-as-object pattern (counters, bank accounts, immutable points, a
 * stack) that stands in for classes in a subset without `this` or `new`.
 *
 * Self checking: main() returns the number of failed checks and the run exits 0
 * exactly when all pass, identically under both engines and both back ends. **/

var failures = 0;
var checks = 0;

function check(name, got, want) {
    checks = checks + 1;
    if (got !== want) {
        println("FAIL " + name + ": got " + got + " want " + want);
        failures = failures + 1;
    }
}

function checkArr(name, gotArr, wantStr) {
    check(name, gotArr.join(","), wantStr);
}

// ----- higher-order toolkit -----

function map(a, fn) {
    var out = [];
    for (var i = 0; i < a.length; i++) { out.push(fn(a[i])); }
    return out;
}

function filter(a, pred) {
    var out = [];
    for (var i = 0; i < a.length; i++) {
        if (pred(a[i])) { out.push(a[i]); }
    }
    return out;
}

function reduce(a, fn, init) {
    var acc = anytype;             // the accumulator may hold any type
    acc = init;
    for (var i = 0; i < a.length; i++) { acc = fn(acc, a[i]); }
    return acc;
}

function every(a, pred) {
    for (var i = 0; i < a.length; i++) {
        if (!pred(a[i])) { return false; }
    }
    return true;
}

function some(a, pred) {
    for (var i = 0; i < a.length; i++) {
        if (pred(a[i])) { return true; }
    }
    return false;
}

function count(a, pred) {
    var c = 0;
    for (var i = 0; i < a.length; i++) {
        if (pred(a[i])) { c++; }
    }
    return c;
}

function find(a, pred) {
    for (var i = 0; i < a.length; i++) {
        if (pred(a[i])) { return a[i]; }
    }
    return undefined;
}

function zipWith(a, b, fn) {
    var out = [];
    var n = a.length < b.length ? a.length : b.length;
    for (var i = 0; i < n; i++) { out.push(fn(a[i], b[i])); }
    return out;
}

function range(start, end) {
    var out = [];
    for (var i = start; i < end; i++) { out.push(i); }
    return out;
}

// ----- plain functions used as first-class values -----

function isEven(n) { return n % 2 == 0; }
function isPositive(n) { return n > 0; }
function double(n) { return n * 2; }
function square(n) { return n * n; }
function add(a, b) { return a + b; }
function mul(a, b) { return a * b; }
function max2(a, b) { return a > b ? a : b; }

// ----- composition and pipelines -----

function compose(f, g) {
    return function(x) { return f(g(x)); };
}

function pipeAll(fns, x) {
    var v = anytype;               // the value threads through, changing type
    v = x;
    for (var i = 0; i < fns.length; i++) { v = fns[i](v); }
    return v;
}

// ----- currying, partial application, factories -----

function curryAdd3(a) {
    return function(b) {
        return function(c) { return a + b + c; };
    };
}

function makeAdder(n) { return function(x) { return x + n; }; }
function makeMultiplier(n) { return function(x) { return x * n; }; }
function makeGreaterThan(n) { return function(x) { return x > n; }; }

// ----- memoization -----

function memoize(fn) {
    var cache = {};
    var calls = 0;
    return {
        call: function(n) {
            var key = "" + n;
            var cached = cache[key];
            if (cached !== undefined) { return cached; }
            calls++;
            var result = fn(n);
            cache[key] = result;
            return result;
        },
        misses: function() { return calls; }
    };
}

// ----- lazy iterator -----

function makeRangeIter(start, end) {
    var cur = start;
    return {
        hasNext: function() { return cur < end; },
        next: function() {
            var v = cur;
            cur++;
            return v;
        }
    };
}

function drainIter(it) {
    var out = [];
    while (it.hasNext()) { out.push(it.next()); }
    return out;
}

// ----- closure-as-object: counter, account, point, stack -----

function makeCounter(start, step) {
    var value = start;
    return {
        inc: function() { value += step; return value; },
        dec: function() { value -= step; return value; },
        get: function() { return value; },
        reset: function() { value = start; return value; }
    };
}

function makeAccount(initial) {
    var balance = initial;
    var moves = 0;
    return {
        deposit: function(amount) { balance += amount; moves++; return balance; },
        withdraw: function(amount) {
            if (amount > balance) { return -1; }
            balance -= amount;
            moves++;
            return balance;
        },
        balance: function() { return balance; },
        transactions: function() { return moves; }
    };
}

function makePoint(x, y) {
    return {
        getX: function() { return x; },
        getY: function() { return y; },
        translate: function(dx, dy) { return makePoint(x + dx, y + dy); },
        scale: function(k) { return makePoint(x * k, y * k); },
        distSq: function(other) {
            var ddx = x - other.getX();
            var ddy = y - other.getY();
            return ddx * ddx + ddy * ddy;
        },
        show: function() { return "(" + x + "," + y + ")"; }
    };
}

function makeStack() {
    var items = [];
    return {
        push: function(v) { items.push(v); return items.length; },
        pop: function() { return items.pop(); },
        size: function() { return items.length; },
        toArray: function() { return items.slice(0); }
    };
}

function once(fn) {
    var called = false;
    var result = anytype;
    return function(x) {
        if (!called) {
            result = fn(x);
            called = true;
        }
        return result;
    };
}

// ----- variadic via the arguments array -----

function sumAll() {
    var s = 0;
    for (var i = 0; i < arguments.length; i++) { s += arguments[i]; }
    return s;
}

function maxAll() {
    var m = arguments[0];
    for (var i = 1; i < arguments.length; i++) {
        if (arguments[i] > m) { m = arguments[i]; }
    }
    return m;
}

function main() {

    var nums = range(1, 11);   // [1..10]

    // ----- map / filter / reduce -----
    checkArr("map square", map(range(1, 6), square), "1,4,9,16,25");
    checkArr("map double", map([1, 2, 3], double), "2,4,6");
    checkArr("filter even", filter(nums, isEven), "2,4,6,8,10");
    checkArr("filter gt5", filter(nums, makeGreaterThan(5)), "6,7,8,9,10");
    check("reduce sum", reduce(nums, add, 0), 55);
    check("reduce product 1..5", reduce(range(1, 6), mul, 1), 120);
    check("reduce max", reduce([3, 9, 2, 7, 5], max2, 0), 9);
    check("reduce build string", reduce(["a", "b", "c"], add, ""), "abc");

    // ----- pipeline: square, keep even, sum -----
    check("map filter reduce", reduce(filter(map(nums, square), isEven), add, 0), 220);

    // ----- every / some / find / count -----
    check("every even yes", every([2, 4, 6], isEven), true);
    check("every even no", every([2, 3, 6], isEven), false);
    check("some even yes", some([1, 3, 4], isEven), true);
    check("some even no", some([1, 3, 5], isEven), false);
    check("find first even", find([1, 3, 4, 5, 6], isEven), 4);
    check("find none", find([1, 3, 5], isEven), undefined);
    check("count even", count(nums, isEven), 5);
    check("count positive", count([-2, -1, 0, 1, 2], isPositive), 2);

    // ----- zipWith -----
    checkArr("zip add", zipWith([1, 2, 3], [10, 20, 30], add), "11,22,33");
    checkArr("zip mul uneven", zipWith([2, 3, 4, 5], [10, 10], mul), "20,30");

    // ----- composition and pipelines -----
    var inc = makeAdder(1);
    var dbl = makeMultiplier(2);
    check("compose f g", compose(inc, dbl)(5), 11);
    check("compose g f", compose(dbl, inc)(5), 12);
    check("compose is not commutative", compose(inc, dbl)(5) != compose(dbl, inc)(5), true);
    check("pipe numbers", pipeAll([inc, dbl, square], 3), 64);
    check("pipe empty", pipeAll([], 42), 42);

    // ----- pipeline over an array, threading changing types -----
    check("array pipeline", pipeAll([
        function(a) { return map(a, double); },
        function(a) { return filter(a, isPositive); },
        function(a) { return reduce(a, add, 0); }
    ], [1, 2, 3, 4]), 20);

    // ----- currying and factories -----
    check("curry add3", curryAdd3(1)(2)(3), 6);
    check("curry partial", curryAdd3(10)(20)(30), 60);
    var add10 = curryAdd3(10);
    var add10and5 = add10(5);
    check("curry reuse a", add10and5(1), 16);
    check("curry reuse b", add10and5(100), 115);
    check("adder factory", makeAdder(7)(35), 42);
    check("multiplier factory", makeMultiplier(6)(7), 42);

    // ----- closures capturing distinct values in a loop -----
    var fns = [];
    for (var i = 0; i < 4; i++) { fns.push(makeAdder(i * 10)); }
    check("closures capture values", fns[0](1) + fns[1](1) + fns[2](1) + fns[3](1), 64);

    // ----- memoization: repeated inputs are computed once -----
    var mem = memoize(square);
    check("memo first", mem.call(6), 36);
    check("memo repeat value", mem.call(6), 36);
    check("memo new value", mem.call(9), 81);
    check("memo repeat again", mem.call(9), 81);
    check("memo third", mem.call(6), 36);
    check("memo misses", mem.misses(), 2);

    // ----- lazy iterator -----
    checkArr("iterator drain", drainIter(makeRangeIter(3, 8)), "3,4,5,6,7");
    var it = makeRangeIter(0, 3);
    check("iter has next", it.hasNext(), true);
    check("iter next 0", it.next(), 0);
    check("iter next 1", it.next(), 1);
    check("iter next 2", it.next(), 2);
    check("iter exhausted", it.hasNext(), false);

    // ----- counters keep independent private state -----
    var c1 = makeCounter(0, 1);
    var c2 = makeCounter(100, 5);
    c1.inc();
    c1.inc();
    check("counter one", c1.inc(), 3);
    check("counter two", c2.inc(), 105);
    check("counter one unaffected", c1.get(), 3);
    check("counter dec", c2.dec(), 100);
    check("counter reset", c1.reset(), 0);

    // ----- bank account -----
    var acc = makeAccount(100);
    check("account start", acc.balance(), 100);
    check("deposit", acc.deposit(50), 150);
    check("withdraw", acc.withdraw(30), 120);
    check("overdraw rejected", acc.withdraw(1000), -1);
    check("balance after reject", acc.balance(), 120);
    check("transaction count", acc.transactions(), 2);

    // ----- immutable point objects -----
    var p = makePoint(1, 2);
    check("point x", p.getX(), 1);
    check("point y", p.getY(), 2);
    var q = p.translate(3, 4);
    check("translated x", q.getX(), 4);
    check("translated y", q.getY(), 6);
    check("original unchanged", p.getX(), 1);
    check("dist squared", p.distSq(q), 25);
    check("scaled", p.scale(10).getX(), 10);
    check("point show", p.show(), "(1,2)");
    check("chained transform", p.translate(1, 1).scale(2).show(), "(4,6)");

    // ----- closure-backed stack -----
    var st = makeStack();
    st.push(1);
    st.push(2);
    check("stack push", st.push(3), 3);
    check("stack pop", st.pop(), 3);
    checkArr("stack contents", st.toArray(), "1,2");

    // ----- once: side effect happens a single time -----
    var counter = makeCounter(0, 1);
    var wrapped = once(function(x) { counter.inc(); return x * x; });
    check("once first", wrapped(5), 25);
    check("once second keeps result", wrapped(9), 25);
    check("once third keeps result", wrapped(0), 25);
    check("once ran once", counter.get(), 1);

    // ----- variadic helpers -----
    check("sumAll", sumAll(1, 2, 3, 4, 5), 15);
    check("sumAll none", sumAll(), 0);
    check("maxAll", maxAll(3, 9, 2, 7, 1), 9);
    check("maxAll single", maxAll(42), 42);

    printf("%c%c %d checks\n", 79, 75, checks);
    if (failures == 0) { println("MetaJS big test 3 (closures and OO) passed"); }
    return failures;
}
