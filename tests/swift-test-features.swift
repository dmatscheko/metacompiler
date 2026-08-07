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

// Declared-type adoption (docs/todo.md 1.1): the four sites, smallest form each.
func halfOf(_ x: Double) -> Double { return x / 2 }
func threeD() -> Double { return 3 }
struct Boxed { var v: Double; var w: UInt8 }
let dbls: [Double] = [3, 4]
let ints: [Int] = [3, 4]
let nestedD: [[Double]] = [[3, 4]]
// The STRUCTURAL declared type (docs/todo.md 1.3): a dictionary's VALUE type, a
// tuple's element types and an Optional annotation each carry the numeric leaf
// down to the untyped literal, exactly as `[Double]` does. All three answered 1
// in every engine where swiftc 6.1.2 says 1.5.
let dmap: [String: Double] = ["a": 3]
let dtup: (Double, Int) = (3, 4)
let dltup: (x: Double, y: Int) = (3, 4)
var dopt: Double? = 3
// An OBSERVED stored property keeps its value under a hidden name, so neither its
// default nor a later write had a declared type to adopt from.
struct Obs {
    var p: Double = 3 { didSet { } }
    mutating func bump() { p = 3 }
}
class ObsC {
    var p: Double = 3 { didSet { } }
    func bump() { p = 3 }
}
// A LINKED write through a force-unwrap, and a write to a tuple ELEMENT: neither
// parsed at all before - the whole statement was rejected in both halves.
class Node { var v: Double = 0; var next: Node? = nil }
// OVERLOAD SELECTION by declared parameter type. The compiler half tested the
// argument with the shared js_is_type, which says "an integral number" for
// `Double` and has no arm for `Bool`, so ovl(3.0) ran the Int body and ovl(true)
// fell back to the first candidate - wrong in that half only.
func ovl(_ x: Int) -> String { return "int" }
func ovl(_ x: Double) -> String { return "dbl" }
func ovl(_ x: String) -> String { return "str" }
func ovl(_ x: Bool) -> String { return "bool" }

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

// A subclass that declares no initializer of its own (docs/todo.md 1.3).
class Square: Rect { var sides = 4 }

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

// ----- value description (print / String(describing:)) and value-type == -----
// print used to hand the RAW runtime value to the shared println host function in
// the compiler (`<nil>`, `[1 2 3]`, a Go map holding a POINTER address for an
// instance) and to fall through to JavaScript's ToString in the interpreter
// ("1,2,3", "[object Object]"). Every string below is what swift 6.1.2 prints.

struct PtDesc { var x: Int; var y: String }
struct MoneyDesc: CustomStringConvertible {
    var cents: Int
    var description: String { "$\(cents)" }
}
enum ShapeDesc { case dot; case line(Int, String); case at(n: Int) }

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
    // The two cases a bare `<` / `>` cannot see, and which -0.0 == 0.0 hides
    // from an == assertion: compared as TEXT against what swift 6.1.2 prints.
    // Swift's min/max are `y < x ? y : x` and `y >= x ? y : x`, so a tie keeps
    // the LEFT operand for min and the right for max, and a NaN operand - every
    // comparison against which is false - keeps the left one.
    let zNeg = -0.0, zPos = 0.0
    let dNaN = zPos / zPos
    check("builtin-min-negzero", String(min(zNeg, zPos)) == "-0.0" && String(min(zPos, zNeg)) == "0.0")
    check("builtin-max-negzero", String(max(zNeg, zPos)) == "0.0" && String(max(zPos, zNeg)) == "-0.0")
    check("builtin-abs-negzero", String(abs(zNeg)) == "0.0")
    check("builtin-minmax-nan",
          String(min(dNaN, 1.0)) == "nan" && String(min(1.0, dNaN)) == "1.0"
       && String(max(dNaN, 1.0)) == "nan" && String(max(1.0, dNaN)) == "1.0")

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

    // ----- value description and value-type equality -----
    let noneD: Int? = nil
    check("desc-nil", "\(noneD)" == "nil")
    check("desc-array", "\([1, 2, 3])" == "[1, 2, 3]")
    check("desc-nested", "\([[1, 2], [3]])" == "[[1, 2], [3]]")
    let emptyD: [String] = []
    check("desc-empty-array", "\(emptyD)" == "[]")
    check("desc-dict", "\(["k": 1])" == "[\"k\": 1]")
    let emptyM: [String: Int] = [:]
    check("desc-empty-dict", "\(emptyM)" == "[:]")
    check("desc-str-nested", "\(["s"])" == "[\"s\"]" && "\("plain")" == "plain")
    check("desc-tuple", "\((1, "two"))" == "(1, \"two\")")
    let labelled = (x: 1, y: "s")
    check("desc-tuple-labels", "\(labelled)" == "(x: 1, y: \"s\")")
    check("desc-struct", "\(PtDesc(x: 1, y: "hi"))" == "PtDesc(x: 1, y: \"hi\")")
    check("desc-convertible", "\(MoneyDesc(cents: 7))" == "$7" && "\([MoneyDesc(cents: 7)])" == "[$7]")
    check("desc-enum", "\(ShapeDesc.dot)" == "dot" && "\(ShapeDesc.line(2, "x"))" == "line(2, \"x\")")
    check("desc-enum-label", "\(ShapeDesc.at(n: 5))" == "at(n: 5)")
    check("eq-array", [1, 2] == [1, 2] && [1, 2] != [1, 3] && [[1]] == [[1]])
    check("eq-dict", ["a": 1] == ["a": 1] && ["a": 1] != ["a": 2])
    check("eq-tuple", (1, "a") == (1, "a") && (1, "a") != (2, "a"))
    // print's separator:/terminator: labels used to be dropped and printed as two
    // extra positional arguments. Real Swift writes "1-2-3\nt!\n" here.
    print(1, 2, 3, separator: "-")
    print("t", terminator: "!\n")

    // ----- a literal ADOPTS its declared type (docs/todo.md 1.1) -----
    // A parameter, a return type, a stored property and an array annotation's
    // element type all retype an untyped literal, exactly as a `let`/`var`
    // annotation does. Not a Float story: an Int literal that does not adopt
    // stays an Int and INTEGER-DIVIDES, so every row here answered 1 in both
    // halves where swiftc 6.1.2 says 1.5.
    check("adopt-param", halfOf(3) == 1.5)
    check("adopt-return", threeD() / 2 == 1.5)
    check("adopt-field", Boxed(v: 3, w: 250).v / 2 == 1.5 && Boxed(v: 3, w: 250).w &+ 10 == 4)
    check("adopt-array", dbls[0] / 2 == 1.5 && ints[0] / 2 == 1)
    // The WRITE half of the same rule: only the INITIAL value used to adopt, so
    // `var d: Double = 0; d = 3` and `b.v = 3` both answered 1.
    var wv: Double = 0
    wv = 3
    var wb = Boxed(v: 0, w: 0)
    wb.v = 3
    wb.w = 250
    var wr: [Double] = [0]
    wr[0] = 3
    check("adopt-write", wv / 2 == 1.5 && wb.v / 2 == 1.5 && wb.w &+ 10 == 4 && wr[0] / 2 == 1.5)
    check("adopt-nested", nestedD[0][0] / 2 == 1.5)
    // The STRUCTURAL declared type and the two write sites it did not reach.
    check("adopt-dict", dmap["a"]! / 2 == 1.5)
    check("adopt-tuple", dtup.0 / 2 == 1.5 && dtup.1 / 2 == 2)
    // A LABELLED tuple type declares element types too, and both copies of a
    // labelled element (the index and the label) have to hold the adopted value.
    check("adopt-tuple-labelled", dltup.x / 2 == 1.5 && dltup.0 / 2 == 1.5 && dltup.y / 2 == 2)
    check("adopt-optional", dopt! / 2 == 1.5)
    check("adopt-closure-param", { (x: Double) -> Double in x / 2 }(3) == 1.5)
    check("adopt-closure-ret", { (x: Int) -> Double in 3 }(1) / 2 == 1.5)
    var ob = Obs()
    let obInit = ob.p / 2
    ob.p = 3
    let obOut = ob.p / 2
    ob.bump()
    let obIn = ob.p / 2
    check("adopt-observed", obInit == 1.5 && obOut == 1.5 && obIn == 1.5)
    let obc = ObsC()
    obc.p = 3
    let obcOut = obc.p / 2
    obc.bump()
    check("adopt-observed-class", obcOut == 1.5 && obc.p / 2 == 1.5)
    // `n.next!.v = 4` and `tp.0 = 3` did not PARSE in either half.
    let nd = Node()
    nd.next = Node()
    nd.next!.v = 4
    var wtup: (Double, Int) = (1, 2)
    wtup.0 = 3
    check("write-chain-and-tuple", nd.next!.v / 2 == 2.0 && wtup.0 / 2 == 1.5)
    // The `!` in a target must not turn `a != b` into an assignment: the force-unwrap
    // is guarded by a "no = next" test, exactly as the expression-level one is.
    let ne1 = 1
    check("target-bang-not-noteq", ne1 != 2)

    // ----- `is` / `as?` answer by the VALUE, not by "is it a number"
    // (docs/todo.md 1.2) -----
    // The shared probe predates the float box: it said "an integral number" for Int
    // AND for Double, so `3 is Double` was TRUE and `3.0 is Double` FALSE - the
    // reverse of swiftc 6.1.2 - and had no arm for the name `Bool` at all. The `?`
    // rows go through the same test, so each was a wrong VALUE and not just a flag.
    let f32v: Float = 2.5
    let u8v: UInt8 = 7
    let anys: [Any] = [3.0, 3, "s", true, f32v, u8v]
    check("is-double", (anys[0] is Double) && !(anys[0] is Int) && !(anys[0] is Float))
    check("is-int", (anys[1] is Int) && !(anys[1] is Double) && !(anys[1] is Float))
    check("is-string", (anys[2] is String) && !(anys[2] is Int))
    check("is-bool", (anys[3] is Bool) && !(anys[3] is Int) && !(anys[3] is Double))
    check("is-float", (anys[4] is Float) && !(anys[4] is Double) && !(anys[4] is Int))
    check("is-sized", (anys[5] is UInt8) && !(anys[5] is Int) && !(anys[5] is Int8))
    check("as-double", (anys[0] as? Double) != nil && (anys[1] as? Double) == nil)
    check("as-int", (anys[1] as? Int) != nil && (anys[0] as? Int) == nil)
    check("overload-by-type", ovl(3) == "int" && ovl(3.0) == "dbl"
                              && ovl("s") == "str" && ovl(true) == "bool")

    // ----- a subclass with no init of its own INHERITS one (docs/todo.md 1.3) -----
    // A class never gets a memberwise initializer - that is structs - and the empty
    // one that used to be synthesised here shadowed the inherited init, so every
    // inherited property came back nil.
    check("inherit-init", Square(w: 2, h: 3).area() == 6 && Square(w: 2, h: 3).sides == 4)

    // ----- combined pipeline -----
    check("combined-pipeline", transform([1, 2, -3]) == "o1e2x")

    print("features: \(checks) checks, \(fails) failures")
}

main()
exit(fails)
