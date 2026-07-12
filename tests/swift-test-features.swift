// Fast feature-matrix test for the Swift interpreter (swift-interpreter.abnf) and
// the LLVM-IR compiler (swift-to-llvm-ir.abnf). It replaces the four algorithm-
// themed swift-test-big-* stress tests: instead of large loops (sieves, sorts,
// Roman numerals) every implemented construct is exercised with the SMALLEST
// program that can prove it works - loops run 0, 1, 3 or 4 times, recursion stays
// below depth 6, arrays and dictionaries hold 3-5 elements. A failed check prints
// "FAIL <id>" (so a diff pinpoints it); top level code runs in source order and
// ends with exit(fails), so exit 0 and byte-identical output on all four legs
// (interpreter/compiler x goja/-frozen) mean everything passed.
//
// Subset notes honoured here: strings index as one-character strings (s[i]),
// dictionary reads must hit an existing key, structs/classes are reference types,
// there is no inheritance dispatch, no bitwise operators, no float arithmetic,
// no fallthrough, and Swift has NO finally (so none is tested).

var fails = 0
var checks = 0

func check(_ id: String, _ cond: Bool) {
    checks += 1
    if !cond {
        print("FAIL \(id)")
        fails += 1
    }
}

// ----- functions: labelled parameters, early return, recursion -----

func power(base b: Int, times n: Int) -> Int {
    var r = 1
    var i = 0
    while i < n {
        r = r * b
        i += 1
    }
    return r
}

func grade(_ n: Int) -> String {
    if n > 10 {
        return "big"
    } else if n > 5 {
        return "mid"
    } else {
        return "small"
    }
}

func fib(_ n: Int) -> Int {
    if n < 2 {
        return n
    }
    return fib(n - 1) + fib(n - 2)
}

func isEven(_ n: Int) -> Bool {
    if n == 0 {
        return true
    }
    return isOdd(n - 1)
}
func isOdd(_ n: Int) -> Bool {
    if n == 0 {
        return false
    }
    return isEven(n - 1)
}

func applyTwice(_ x: Int, _ f: (Int) -> Int) -> Int {
    return f(f(x))
}

func makeAdder(_ k: Int) -> (Int) -> Int {
    return { (x: Int) -> Int in x + k }
}

// side-effect counter for short-circuit checks
var sideEffects = 0
func bumpTrue() -> Bool {
    sideEffects += 1
    return true
}
func bumpFalse() -> Bool {
    sideEffects += 1
    return false
}

// ----- guard -----

func safeDiv(_ a: Int, _ b: Int) -> Int {
    guard b != 0 else {
        return -1
    }
    return a / b
}

func inRange(_ n: Int, _ lo: Int, _ hi: Int) -> Bool {
    guard n >= lo && n <= hi else {
        return false
    }
    return true
}

func sumEvens(_ xs: [Int]) -> Int {
    var total = 0
    for x in xs {
        guard x % 2 == 0 else {
            continue
        }
        total += x
    }
    return total
}

// ----- switch -----

func size(_ n: Int) -> String {
    switch n {
    case 0:
        return "zero"
    case 1, 2, 3:
        return "small"
    default:
        return "big"
    }
}

func vowelScore(_ s: String) -> Int {
    var score = 0
    switch s {
    case "a", "e", "i", "o", "u":
        score = 10
        score += 1
    default:
        score = 0
    }
    return score
}

func breakInCase(_ n: Int) -> Int {
    var out = -1
    switch n {
    case 1:
        out = 100
        break
    case 2:
        out = 200
    default:
        out = 999
    }
    return out
}

// ----- structs and classes -----

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

struct Pair {
    var x = 0
    var y = 0
}

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

// ----- exceptions: do / catch / throw (Swift has NO finally) -----

class BoomError {
    var code: Int
    init(_ c: Int) {
        self.code = c
    }
}

func risky(_ n: Int) throws -> Int {
    if n > 3 {
        throw BoomError(n)
    }
    return n * 2
}

func relabel() -> String {
    var result = ""
    do {
        do {
            throw "inner"
        } catch {
            throw "wrapped"          // rethrow out of the inner catch
        }
    } catch {
        result = "handled"
    }
    return result
}

func returnFromDo(_ n: Int) -> Int {
    do {
        if n > 0 {
            return n * 10            // return out of the do body
        }
        throw "neg"
    } catch {
        return -1                    // return out of the catch body
    }
}

func loopBreakInDo() -> Int {
    var sum = 0
    for i in 0...5 {
        do {
            if i == 3 {
                break                // break out of a do body ends the loop
            }
            sum = sum + i
        } catch {
        }
    }
    return sum                       // 0+1+2 = 3
}

func loopContinueInDo() -> Int {
    var sum = 0
    for i in 0...4 {
        do {
            if i == 2 {
                continue             // continue out of a do body skips the rest
            }
            sum = sum + i
        } catch {
        }
    }
    return sum                       // 0+1+3+4 = 8
}

// ----- one small combined pipeline: guard + switch + interpolation -----

func transform(_ list: [Int]) -> String {
    var out = ""
    for n in list {
        guard n >= 0 else {
            out += "x"
            continue
        }
        switch n % 2 {
        case 0:
            out += "e\(n)"
        default:
            out += "o\(n)"
        }
    }
    return out
}

func main() {
    // ----- numbers, arithmetic, precedence -----
    check("arith-precedence", 2 + 3 * 4 == 14)
    check("arith-paren", (2 + 3) * 4 == 20)
    check("arith-unary-minus", -3 + 5 == 2)
    check("arith-neg-paren", -(3 + 4) == -7)
    check("arith-int-div", 7 / 2 == 3)
    check("arith-int-div-neg", -7 / 2 == -3)
    check("arith-mod", 7 % 3 == 1)
    check("arith-mod-neg", -7 % 3 == -1)
    check("arith-chain", 20 - 5 - 3 == 12)
    var x = 5
    x += 3
    x -= 2
    x *= 4
    x /= 3
    x %= 5
    check("arith-compound", x == 3)
    check("builtin-abs-max-min", abs(-9) == 9 && max(3, 8) == 8 && min(3, 8) == 3)

    // ----- let / var -----
    let a = 10
    var b = 20
    b = b + 1
    check("let-var", a == 10 && b == 21)
    let typed: Int = 7
    check("type-annotation", typed == 7)

    // ----- strings -----
    var s = "swift"
    s += "!"
    check("str-concat", "foo" + "bar" == "foobar" && s == "swift!")
    check("str-count", "hello".count == 5 && "".count == 0)
    check("str-compare", "apple" < "banana" && "a" != "b" && "same" == "same")
    let hello = "hello"
    check("str-index", hello[0] == "h" && hello[4] == "o")
    check("str-escapes", "a\tb".count == 3 && "\\".count == 1 && "\"".count == 1)
    let seven = 6
    check("str-interpolation", "v=\(seven + 1)!" == "v=7!")
    check("str-interp-call", "\(fib(4)) \(2 < 3)" == "3 true")
    check("str-unicode", "héllo".count == 5 && "héllo"[1] == "é")
    var rev = ""
    for ch in "abc" {
        rev = ch + rev
    }
    check("str-build-reverse", rev == "cba")
    var vowels = 0
    for ch in "banana" {
        if ch == "a" {
            vowels += 1
        }
    }
    check("str-iterate-compare", vowels == 3)

    // ----- booleans, short-circuit, ternary -----
    check("bool-ops", true && !false || false)
    sideEffects = 0
    let sc1 = bumpFalse() && bumpTrue()     // right side must not run
    let sc2 = bumpTrue() || bumpTrue()      // right side must not run
    check("logic-short-circuit", !sc1 && sc2 && sideEffects == 2)
    check("ternary", (5 > 3 ? "a" : "b") == "a" && (5 < 3 ? "a" : "b") == "b")
    check("cmp-chain", 1 < 2 && 2 <= 2 && 3 > 2 && 3 >= 3 && 1 == 1 && 1 != 2)

    // ----- control flow -----
    check("if-elseif-else", grade(11) == "big" && grade(7) == "mid" && grade(1) == "small")

    var w = 0
    while w > 0 {           // runs zero times
        w -= 1
    }
    check("while-zero", w == 0)
    while w < 3 {           // runs three times
        w += 1
    }
    check("while-three", w == 3)

    var rc = 0
    repeat {                // body runs exactly once
        rc += 1
    } while false
    check("repeat-once", rc == 1)

    var closedSum = 0
    for i in 1...3 {
        closedSum += i
    }
    check("range-closed", closedSum == 6)
    var halfSum = 0
    for i in 0..<3 {
        halfSum += i
    }
    check("range-half-open", halfSum == 3)
    let hi = 4
    var varSum = 0
    for i in 1...hi {
        varSum += i
    }
    check("range-var-bound", varSum == 10)

    var brk = ""
    for i in 0..<9 {
        if i == 2 {
            break
        }
        brk += "\(i)"
    }
    check("for-break", brk == "01")

    var cont = ""
    for i in 0..<4 {
        if i % 2 == 1 {
            continue
        }
        cont += "\(i)"
    }
    check("for-continue", cont == "02")

    var nested = ""
    for oi in 0..<2 {
        for ii in 0..<3 {
            if ii == 1 {
                break           // inner break must not end the outer loop
            }
            nested += "\(oi)\(ii)"
        }
    }
    check("nested-break", nested == "0010")

    var blanks = 0
    for _ in 0..<3 {
        blanks += 1
    }
    check("for-blank-var", blanks == 3)

    // ----- switch -----
    check("switch-zero", size(0) == "zero")
    check("switch-comma-list", size(1) == "small" && size(3) == "small")
    check("switch-default", size(42) == "big")
    check("switch-string-multi", vowelScore("e") == 11 && vowelScore("z") == 0)
    check("switch-break-no-fallthrough", breakInCase(1) == 100 && breakInCase(2) == 200 && breakInCase(5) == 999)
    var tally = 0
    for v in [3, 1, 4] {
        switch v % 3 {
        case 0:
            tally += 100
        case 1:
            tally += 10
        default:
            tally += 1
        }
    }
    check("switch-in-loop", tally == 120)

    // ----- guard -----
    check("guard-pass", safeDiv(20, 4) == 5)
    check("guard-early-return", safeDiv(20, 0) == -1)
    check("guard-compound", inRange(5, 1, 10) && !inRange(0, 1, 10) && !inRange(11, 1, 10))
    check("guard-continue", sumEvens([1, 2, 3, 4]) == 6)

    // ----- functions and closures -----
    check("fn-labelled-params", power(base: 2, times: 3) == 8)
    check("fn-recursion", fib(6) == 8)
    check("fn-mutual-recursion", isEven(4) && isOdd(5))
    let addTen = { (x: Int) -> Int in x + 10 }
    check("closure-value", addTen(5) == 15)
    check("closure-argument", applyTwice(3, addTen) == 23)
    let five = { () -> Int in 5 }
    check("closure-zero-arg", five() == 5)
    let mul = { (p: Int, q: Int) -> Int in p * q }
    check("closure-two-params", mul(3, 4) == 12)
    var counter = 0
    let bump = { counter += 1 }
    bump()
    bump()
    check("closure-captures-var", counter == 2)
    let add3 = makeAdder(3)
    let add9 = makeAdder(9)
    check("closure-factory-independent", add3(1) == 4 && add9(1) == 10)

    // ----- arrays -----
    var nums = [3, 1, 4]
    check("arr-literal-index", nums.count == 3 && nums[0] == 3 && nums[2] == 4)
    nums[1] = 10
    check("arr-write", nums[1] == 10)
    nums.append(7)
    check("arr-append", nums.count == 4 && nums[3] == 7)
    check("arr-last-computed", nums[nums.count - 1] == 7)
    check("arr-contains", nums.contains(4) && !nums.contains(99))
    var lsum = 0
    for v in nums {
        lsum += v
    }
    check("arr-for-in", lsum == 24)
    let nestedArr = [[1, 2], [3]]
    check("arr-nested", nestedArr[0][1] == 2 && nestedArr[1][0] == 3)
    let strs = ["foo", "bar"]
    check("arr-of-strings", strs.contains("bar") && strs[0] == "foo")

    // higher-order array methods with trailing closures
    let base = [1, 2, 3, 4]
    let doubled = base.map { x in x * 2 }
    check("arr-map", doubled[0] == 2 && doubled[3] == 8)
    let evensOnly = base.filter { n in n % 2 == 0 }
    check("arr-filter", evensOnly.count == 2 && evensOnly[0] == 2)
    var acc = 0
    base.forEach { v in acc += v }
    check("arr-foreach", acc == 10)

    // ----- dictionaries (reads must hit existing keys in this subset) -----
    var ages = ["alice": 30, "bob": 25]
    check("dict-read", ages["alice"] == 30 && ages.count == 2)
    ages["carol"] = 40
    check("dict-add", ages["carol"] == 40 && ages.count == 3)
    ages["bob"] = 26
    check("dict-update", ages["bob"] == 26)
    let empty: [String: Int] = [:]
    check("dict-empty", empty.count == 0)

    // ----- optionals: nil and nil-coalescing -----
    let maybe: Int? = nil
    check("nil-coalesce-nil", maybe ?? 42 == 42)
    let present: Int? = 7
    check("nil-coalesce-value", present ?? 42 == 7)

    // ----- structs and classes -----
    var c = Counter(start: 10, step: 5)
    check("struct-init", c.value == 10 && c.step == 5)
    check("struct-mutating-method", c.next() == 15 && c.next() == 20)
    var pr = Pair()
    pr.x = 5
    check("struct-default-props", pr.x == 5 && pr.y == 0)
    let r = Rect(w: 3, h: 4)
    check("class-method", r.area() == 12)
    let r2 = r.scaled(by: 2)
    check("class-returns-instance", r2.area() == 48 && r.area() == 12)
    r2.w = 10
    check("class-prop-write", r2.area() == 80)
    let rects = [Rect(w: 2, h: 2), Rect(w: 3, h: 3)]
    check("class-array", rects[1].area() == 9)

    // ----- enums of bare cases -----
    let d = Direction.north
    check("enum-eq", d == Direction.north && Direction.east != Direction.west)
    var deg = -1
    switch d {
    case Direction.south:
        deg = 180
    case Direction.north:
        deg = 0
    default:
        deg = 99
    }
    check("enum-switch", deg == 0)

    // ----- exceptions: do / catch / throw -----
    var log = ""
    do {
        log += "t"
        throw "boom"
    } catch {
        log = log + "c" + error      // bare catch binds the implicit `error`
    }
    check("throw-catch-implicit", log == "tcboom")

    var noThrow = ""
    do {
        noThrow += "t"
    } catch {
        noThrow += "c"
    }
    check("try-no-throw", noThrow == "t")

    var caught = -1
    do {
        let r3 = try risky(5)        // try is a parsed-and-ignored prefix
        caught = r3                  // not reached
    } catch let e as BoomError {
        caught = e.code              // catch binds the thrown object
    }
    check("throw-object-binding", caught == 5)
    var okPath = -1
    do {
        okPath = try risky(2)
    } catch {
        okPath = -2
    }
    check("try-call-no-throw", okPath == 4)

    check("nested-rethrow", relabel() == "handled")
    check("return-out-of-do", returnFromDo(4) == 40)
    check("return-out-of-catch", returnFromDo(-1) == -1)
    check("break-out-of-do", loopBreakInDo() == 3)
    check("continue-out-of-do", loopContinueInDo() == 8)

    // ----- combined pipeline -----
    check("combined-pipeline", transform([1, 2, -3]) == "o1e2x")

    print("features: \(checks) checks, \(fails) failures")
}

main()
exit(fails)
