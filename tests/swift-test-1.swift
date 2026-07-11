// Swift subset self test.
//
// Top level code runs in source order; it counts failed checks and ends with
// exit(failures), so the metacompiler run exits with 0 exactly when every check
// passes. The interpreter and the LLVM-IR compiler must agree on every result.

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
func checkS(_ name: String, _ got: String, _ want: String) {
    if got != want {
        print("FAIL \(name): got \(got) want \(want)")
        fails += 1
    }
}

// Labelled parameters: the external label differs from the bound name.
func power(base b: Int, times n: Int) -> Int {
    var r = 1
    var i = 0
    while i < n {
        r = r * b
        i += 1
    }
    return r
}

func fib(_ n: Int) -> Int {
    if n < 2 {
        return n
    }
    return fib(n - 1) + fib(n - 2)
}

// A closure passed to a function.
func applyTwice(_ x: Int, _ f: (Int) -> Int) -> Int {
    return f(f(x))
}

// A struct with stored properties, an init and a mutating method.
struct Counter {
    var value: Int
    let step: Int
    init(start: Int, step: Int) {
        self.value = start
        self.step = step
    }
    mutating func next() -> Int {
        value += step        // implicit self.value
        return value
    }
}

// A class with a computed-in-init property and a method using self.
class Rect {
    var w: Int
    var h: Int
    init(w: Int, h: Int) {
        self.w = w
        self.h = h
    }
    func area() -> Int {
        return w * h
    }
    func scaled(by k: Int) -> Rect {
        return Rect(w: w * k, h: h * k)
    }
}

enum Direction {
    case north, south
    case east
    case west
}

func main() {
    // arithmetic and precedence
    check("precedence", 1 + 2 * 3, 7)
    check("parens", (1 + 2) * 3, 9)
    check("division", 7 / 2, 3)
    check("negative division", -7 / 2, -3)
    check("modulo", 7 % 3, 1)
    check("unary minus", -(3 + 4), -7)

    // comparisons and boolean logic
    checkB("lt", 2 < 3, true)
    checkB("ge", 3 >= 3, true)
    checkB("and", true && (1 < 2), true)
    checkB("or short circuit", (1 > 2) || (2 > 1), true)
    checkB("not", !(1 == 2), true)
    checkB("eq int", 5 == 5, true)
    checkB("ne int", 5 != 6, true)

    // let / var and reassignment
    let a = 10
    var b = 20
    if a > 5 {
        b = b + 1
    } else {
        b = b - 1
    }
    check("if statement", b, 21)

    // labelled call and while loop
    check("power", power(base: 2, times: 10), 1024)

    // strings, interpolation and equality
    let name = "world"
    checkS("interpolation", "hello \(name)!", "hello world!")
    checkS("interp expr", "sum = \(a + b)", "sum = 31")
    checkS("concat", "foo" + "bar", "foobar")
    check("string count", name.count, 5)
    checkB("string equality", name == "world", true)

    // for-in over closed and half-open ranges
    var sum = 0
    for i in 1...10 {
        sum += i
    }
    check("closed range", sum, 55)
    var sum2 = 0
    for i in 0..<5 {
        sum2 += i
    }
    check("half open range", sum2, 10)

    // break and continue
    var evens = 0
    for i in 0...20 {
        if i % 2 == 1 {
            continue
        }
        if i > 10 {
            break
        }
        evens += i
    }
    check("break continue", evens, 30)

    // repeat / while
    var rc = 0
    repeat {
        rc += 1
    } while rc < 3
    check("repeat while", rc, 3)

    // arrays: literal, index, count, append, contains, for-in
    var nums = [3, 1, 4, 1, 5]
    check("array count", nums.count, 5)
    check("array index", nums[2], 4)
    nums.append(9)
    check("array append", nums.count, 6)
    checkB("array contains", nums.contains(4), true)
    checkB("array missing", nums.contains(7), false)
    var lsum = 0
    for x in nums {
        lsum += x
    }
    check("array for-in", lsum, 23)
    nums[0] = 30
    check("array element assign", nums[0], 30)

    // array higher order methods with closures
    let base = [1, 2, 3, 4]
    let doubled = base.map { x in x * 2 }
    check("map", doubled[0] + doubled[3], 10)
    let evensOnly = base.filter { n in n % 2 == 0 }
    check("filter", evensOnly.count, 2)
    var acc = 0
    base.forEach { v in acc += v }
    check("forEach closure capture", acc, 10)

    // dictionaries: literal, subscript, count, update, add
    var ages = ["alice": 30, "bob": 25]
    check("dict count", ages.count, 2)
    check("dict read", ages["alice"], 30)
    ages["carol"] = 40
    check("dict add", ages.count, 3)
    ages["bob"] = 26
    check("dict update", ages["bob"], 26)
    let empty: [String: Int] = [:]
    check("empty dict", empty.count, 0)

    // closures as values, captured over the defining scope
    let addTen = { (x: Int) -> Int in x + 10 }
    check("closure value", addTen(5), 15)
    check("closure argument", applyTwice(3, addTen), 23)
    var counter = 0
    let bump = { counter += 1 }
    bump()
    bump()
    check("closure captures var", counter, 2)

    // struct: construction, property read, mutating method
    var c = Counter(start: 10, step: 5)
    check("struct init", c.value, 10)
    check("struct method", c.next(), 15)
    check("struct method again", c.next(), 20)

    // class: methods, self, returning a new instance
    let r = Rect(w: 3, h: 4)
    check("class area", r.area(), 12)
    let r2 = r.scaled(by: 2)
    check("class chained method", r2.area(), 48)
    check("class original unchanged", r.area(), 12)

    // enum of bare cases as constants
    let d = Direction.north
    checkB("enum equality", d == Direction.north, true)
    checkB("enum inequality", Direction.east == Direction.west, false)

    // recursion
    check("fib", fib(10), 55)

    // nil-coalescing
    let maybe: Int? = nil
    check("coalesce nil", maybe ?? 42, 42)
    let present: Int? = 7
    check("coalesce value", present ?? 42, 7)

    // builtins
    check("abs", abs(-9), 9)
    check("max", max(3, 8), 8)
    check("min", min(3, 8), 3)

    if fails == 0 {
        print("Swift subset self test passed")
    }
}

main()
exit(fails)
