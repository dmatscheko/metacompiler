// Swift subset self test: object-oriented and functional features.
//
// Theme: the language's signature features - structs and classes with labelled
// initializers, methods and `self`, enums driving `switch` dispatch, and a full
// functional toolkit: map/filter/forEach, a generic fold taking a closure,
// function composition, closures that capture and mutate state, arrays of
// closures, and a small enum-plus-switch state machine. Top level code runs in
// source order, counts failed checks and ends with exit(failures), so the run
// exits 0 exactly when every check passes. The interpreter and the LLVM-IR
// compiler must agree on every result.
//
// Subset note: struct instances are reference types here (no value copy), and a
// closure's value is its last expression - an explicit `return` inside a closure
// is avoided because the two engines treat it differently.

var fails = 0

func check(_ name: String, _ got: Int, _ want: Int) {
    if got != want {
        print("FAIL \(name): got \(got) want \(want)")
        fails += 1
    }
}
func checkB(_ name: String, _ got: Bool, _ want: Bool) {
    check(name, got ? 1 : 0, want ? 1 : 0)
}

func gcdOf(_ a: Int, _ b: Int) -> Int {
    var x = a < 0 ? -a : a
    var y = b < 0 ? -b : b
    while y != 0 {
        let t = x % y
        x = y
        y = t
    }
    return x
}

// ---- a Point value type with methods and self ----

struct Point {
    var x: Int
    var y: Int
    init(_ x: Int, _ y: Int) {
        self.x = x
        self.y = y
    }
    func translated(dx: Int, dy: Int) -> Point {
        return Point(x + dx, y + dy)
    }
    func manhattan() -> Int {
        let ax = x < 0 ? -x : x
        let ay = y < 0 ? -y : y
        return ax + ay
    }
    func equalsTo(_ p: Point) -> Bool {
        return x == p.x && y == p.y
    }
}

// ---- a reduced rational number ----

struct Rational {
    var num: Int
    var den: Int
    init(_ n: Int, _ d: Int) {
        var nn = n
        var dd = d
        let g = gcdOf(n, d)
        if g != 0 {
            nn = n / g
            dd = d / g
        }
        if dd < 0 {
            nn = -nn
            dd = -dd
        }
        self.num = nn
        self.den = dd
    }
    func add(_ o: Rational) -> Rational {
        return Rational(num * o.den + o.num * den, den * o.den)
    }
    func mul(_ o: Rational) -> Rational {
        return Rational(num * o.num, den * o.den)
    }
    func equalsTo(_ o: Rational) -> Bool {
        return num == o.num && den == o.den
    }
}

// ---- shapes dispatched by an enum tag ----

enum ShapeKind {
    case square
    case rectangle
    case rightTriangle
}

class Shape {
    var kind: String
    var a: Int
    var b: Int
    init(kind: String, a: Int, b: Int) {
        self.kind = kind
        self.a = a
        self.b = b
    }
    func area() -> Int {
        switch kind {
        case ShapeKind.square:
            return a * a
        case ShapeKind.rectangle:
            return a * b
        case ShapeKind.rightTriangle:
            return a * b / 2
        default:
            return 0
        }
    }
    func perimeter() -> Int {
        switch kind {
        case ShapeKind.square:
            return 4 * a
        case ShapeKind.rectangle:
            return 2 * (a + b)
        case ShapeKind.rightTriangle:
            // 3-4-5 style: caller passes the two legs, hypotenuse computed
            return a + b + hypotOf(a, b)
        default:
            return 0
        }
    }
}

// integer hypotenuse (assumes a perfect Pythagorean pair)
func hypotOf(_ a: Int, _ b: Int) -> Int {
    let target = a * a + b * b
    var h = 1
    while h * h < target {
        h += 1
    }
    return h
}

// ---- a stateful bank account with an overdraft guard ----

class Account {
    var balance: Int
    var ops: Int
    init(_ start: Int) {
        self.balance = start
        self.ops = 0
    }
    func deposit(_ amount: Int) {
        balance += amount
        ops += 1
    }
    func withdraw(_ amount: Int) -> Bool {
        guard amount <= balance else {
            return false
        }
        balance -= amount
        ops += 1
        return true
    }
}

// ---- higher-order functions ----

func reduceInts(_ a: [Int], _ initial: Int, _ f: (Int, Int) -> Int) -> Int {
    var acc = initial
    for x in a {
        acc = f(acc, x)
    }
    return acc
}

func compose(_ f: (Int) -> Int, _ g: (Int) -> Int) -> (Int) -> Int {
    return { x in f(g(x)) }
}

func applyN(_ f: (Int) -> Int, _ n: Int, _ x: Int) -> Int {
    var r = x
    var i = 0
    while i < n {
        r = f(r)
        i += 1
    }
    return r
}

func makeAccumulator() -> (Int) -> Int {
    var total = 0
    return { delta in
        total += delta
        total
    }
}

// ---- an enum + switch state machine (traffic light) ----

enum Light {
    case red
    case green
    case yellow
}

func nextLight(_ s: String) -> String {
    switch s {
    case Light.red:
        return Light.green
    case Light.green:
        return Light.yellow
    case Light.yellow:
        return Light.red
    default:
        return Light.red
    }
}

func main() {
    // Point
    let p = Point(3, 4)
    check("point manhattan", p.manhattan(), 7)
    let q = p.translated(dx: 1, dy: -2)
    check("point tx x", q.x, 4)
    check("point tx y", q.y, 2)
    check("point manhattan2", q.manhattan(), 6)
    checkB("point eq", p.equalsTo(Point(3, 4)), true)
    checkB("point ne", p.equalsTo(q), false)
    let origin = Point(-5, -12)
    check("point neg manhattan", origin.manhattan(), 17)

    // Rational
    let half = Rational(1, 2)
    let third = Rational(1, 3)
    let five6 = half.add(third)
    check("rat add num", five6.num, 5)
    check("rat add den", five6.den, 6)
    let one = half.add(half)
    checkB("rat one", one.equalsTo(Rational(1, 1)), true)
    let sixth = half.mul(third)
    check("rat mul num", sixth.num, 1)
    check("rat mul den", sixth.den, 6)
    // reduction and sign normalization
    let r = Rational(6, -8)
    check("rat reduce num", r.num, -3)
    check("rat reduce den", r.den, 4)

    // Shapes
    let sq = Shape(kind: ShapeKind.square, a: 5, b: 0)
    check("square area", sq.area(), 25)
    check("square perim", sq.perimeter(), 20)
    let rect = Shape(kind: ShapeKind.rectangle, a: 3, b: 7)
    check("rect area", rect.area(), 21)
    check("rect perim", rect.perimeter(), 20)
    let tri = Shape(kind: ShapeKind.rightTriangle, a: 3, b: 4)
    check("tri area", tri.area(), 6)
    check("tri perim", tri.perimeter(), 12)
    // sum of areas via a loop over shapes
    let shapes = [sq, rect, tri]
    var areaSum = 0
    for sh in shapes {
        areaSum += sh.area()
    }
    check("total area", areaSum, 52)

    // Account: stateful mutation and an overdraft guard
    let acct = Account(100)
    acct.deposit(50)
    checkB("wd ok", acct.withdraw(120), true)
    check("balance", acct.balance, 30)
    checkB("wd overdraft", acct.withdraw(999), false)
    check("balance after fail", acct.balance, 30)
    check("op count", acct.ops, 2)

    // reduce with closures
    let nums = [1, 2, 3, 4, 5, 6]
    check("reduce sum", reduceInts(nums, 0, { acc, x in acc + x }), 21)
    check("reduce product", reduceInts([1, 2, 3, 4], 1, { acc, x in acc * x }), 24)
    check("reduce max", reduceInts(nums, nums[0], { acc, x in acc > x ? acc : x }), 6)
    check("reduce count evens", reduceInts(nums, 0, { acc, x in x % 2 == 0 ? acc + 1 : acc }), 3)

    // map / filter / forEach pipeline
    let squares = nums.map { x in x * x }
    check("map first", squares[0], 1)
    check("map last", squares[5], 36)
    let bigSquares = squares.filter { x in x > 10 }
    check("filter count", bigSquares.count, 3)
    var pipeSum = 0
    nums.filter { x in x % 2 == 1 }.forEach { x in pipeSum += x * 10 }
    check("pipeline", pipeSum, 90)

    // composition and repeated application
    let inc = { x in x + 1 }
    let dbl = { x in x * 2 }
    let incThenDbl = compose(dbl, inc)
    check("compose", incThenDbl(5), 12)
    let dblThenInc = compose(inc, dbl)
    check("compose2", dblThenInc(5), 11)
    check("applyN double", applyN(dbl, 5, 1), 32)
    check("applyN inc", applyN(inc, 100, 0), 100)

    // stateful closures: two independent accumulators
    let accA = makeAccumulator()
    let accB = makeAccumulator()
    check("accA 10", accA(10), 10)
    check("accA 5", accA(5), 15)
    check("accB 3", accB(3), 3)
    check("accA 100", accA(100), 115)
    check("accB isolated", accB(7), 10)

    // array of closures applied in sequence
    let stages = [{ x in x + 3 }, { x in x * x }, { x in x - 1 }]
    var val = 2
    for stage in stages {
        val = stage(val)
    }
    check("closure stages", val, 24)

    // traffic light state machine: one full cycle and counting
    var light = Light.red
    var reds = 0
    var steps = 0
    while steps < 9 {
        if (light == Light.red) {
            reds += 1
        }
        light = nextLight(light)
        steps += 1
    }
    check("light reds in 9", reds, 3)
    // after 9 steps (a multiple of the period) we are back at red
    checkB("light back to red", light == Light.red, true)
    // one more step goes green
    let ng = nextLight(light)
    checkB("light then green", ng == Light.green, true)

    if fails == 0 {
        print("Swift OO and functional self test passed")
    }
}

main()
exit(fails)
