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

// ----- MEMBER overload selection, run-time, by argument TYPE (docs/todo.md 1.2) -----
// The compiler half picked an initializer at WALK time by labels and arity alone,
// so MB(3.0) ran `init(_ x: Int)` where the interpreter and swiftc both say
// Double; ordinary METHODS and SUBSCRIPTS had no dispatcher at all in that half,
// so s.m(1.5), s.m("a") and even s.m(1, 2) all ran `m(_ x: Int)`. A live halves
// divergence --cross never reached, because both files agreed on the wrong answer
// only in the compiler.
struct MB {
    var s: String
    init(_ x: Int) { s = "I" }
    init(_ x: Double) { s = "D" }
    init(_ x: String) { s = "S" }
    init(_ x: Bool) { s = "B" }
    init(a: Int) { s = "lI" }
    init(a: Double) { s = "lD" }
}
struct OvM {
    func m(_ x: Int) -> String { return "I" }
    func m(_ x: Double) -> String { return "D" }
    func m(_ x: String) -> String { return "S" }
    func m(_ a: Int, _ b: Int) -> String { return "II" }
    subscript(i: Int) -> String { return "sI" }
    subscript(t: String) -> String { return "sS" }
}
// A function-typed SLOT (docs/todo.md 1.4): the type is on the slot and the
// closure carries nothing, so f(3) integer-divided and answered 1.
let fslot: (Double) -> Double = { x in x / 2 }
let fslot2: (Double, Double) -> Double = { a, b in a / b }
// A stored property holding a closure is CALLABLE - it is not in the member
// table, and calling it aborted with "unknown method 'fn'" in all three engines.
struct FnField { var fn: (Double) -> Double }
// `case is T:` did not parse in either half and took the whole switch down.
enum IsE { case a, b }
func isKind(_ v: Any) -> String {
    switch v {
    case is Double: return "D"
    case is String: return "S"
    case is Bool: return "B"
    case is IsE: return "E"
    case is Int: return "I"
    default: return "?"
    }
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

// ----- docs/todo.md 1.2: a STRUCTURALLY typed parameter, collection method
// ----- adoption, inout overloads, inherited overload groups, static overloads.
// Every value here is swiftc 6.1.2's own; all five were wrong in all three
// engines, so --cross could not see any of them.
func adoptArrParam(_ xs: [Double]) -> Double { return xs[0] / 2 }
func adoptDictParam(_ m: [String: Double]) -> Double { return m["k"]! / 2 }
func adoptFnParam(_ f: (Double) -> Double) -> Double { return f(3) }
func adoptNoLeafParam(_ xs: [String]) -> String { return xs[0] }
func inoutOv(_ x: inout Int) { x = x + 100 }
func inoutOv(_ x: inout String) { x = x + "!" }
class OvBase {
    func pick(_ a: Int) -> String { return "bI" }
    func pick(_ a: String) -> String { return "bS" }
}
class OvSub: OvBase {
    override func pick(_ a: Int) -> String { return "sI" }
}
class StatOv {
    static func g(_ a: Int) -> String { return "I" }
    static func g(_ a: String) -> String { return "S" }
}

// ----- NESTED TYPES (docs/todo.md 1.7) -----
// typeInfo used to be keyed by the BARE type name, so two same-named nested types
// shared one record. The memberwise initializer's overload slot is taken from that
// record at walk time while the descriptor installs it at a per-declaration index,
// so the SECOND Inner's `init` landed on a key nothing looked up and the compiler
// half aborted with `call of a non function value` where the interpreter and
// swiftc 6.1.2 both print the answer. It needs the LABELS to collide to show, which
// is why the item's own `{var x}` beside `{var y}` example survived.
struct NA { struct Inner { var x: Int; func t() -> Int { return x } }
    enum E: Int { case a = 1, b = 2 }
    static func mk() -> Int { return Inner(x: 4).x } }
struct NB { struct Inner { var x: Int; func t() -> Int { return x * 10 } }
    enum E: Int { case a = 10, b = 20 } }
// Three levels, and the same leaf name at two depths.
struct NC { struct B { struct C { var v: Int } } }
struct ND { struct B { struct C { var v: Int } } }
struct NE { struct Inner { var x: Int
        struct Inner { var x: Int; func k() -> Int { return x + 100 } } } }
// Overloaded initializers on two same-named nested types: the sig LISTS must not
// merge either, or the second type's call site picks the first type's overload.
struct NF { struct Inner { var v = 0
        init(x: Int) { v = x }
        init(y: String) { v = y.count } } }
struct NG { struct Inner { var v = 0
        init(z: Int) { v = z + 100 } } }
// extension of a NESTED type: baseTypeName keeps the dot, so `A.Inner` matched no
// registered type and the block installed NOTHING - `unknown method` in the
// interpreter, `extension of an unknown type` in the compiler.
struct NH { struct Inner { var x: Int } }
extension NH.Inner { func dbl() -> Int { return x * 2 }
    static func s() -> Int { return 77 }
    init(twice t: Int) { self.x = t * 2 }
    subscript(i: Int) -> Int { return x + i } }
// A type nested in an ENUM lives on the enum's MEMBER TABLE, not on the descriptor,
// so a field read of it answered nil in the interpreter alone.
enum NI { enum Mode: Int { case fast = 3, slow = 4 } }
extension NI.Mode { func tag() -> Int { return self.rawValue + 10 } }
// A static method reading its own type's static member UNQUALIFIED: sw_kget
// answers an unqualified name from `this`, and a static was emitted with no `this`,
// so the compiler half aborted with `variable not defined` for every type.
struct NJ { static var sv = 5
    static func f() -> Int { return sv + 1 }
    struct Inner { static var sv = 50
        static func f() -> Int { return sv + 1 } } }
// A nested type as a protocol witness, as a declared type and as an element type,
// plus a STATIC SUBSCRIPT on two same-named nested types.
protocol NTag { func tag() -> String }
struct NK { struct Inner: NTag { var x: Int
        func tag() -> String { return "K\(x)" }
        static subscript(i: Int) -> Int { return i * 2 } } }
struct NL { struct Inner: NTag { var x: Int
        func tag() -> String { return "L\(x)" }
        static subscript(i: Int) -> Int { return i * 3 } } }
func ntake(_ p: NTag) -> String { return p.tag() }
// A nested type must NOT shadow a same-named TOP-LEVEL one (docs/todo.md 1.2). It
// used to be declared into the SURROUNDING scope under its bare name as well as
// being installed on its enclosing descriptor, so the nested descriptor overwrote
// the top-level binding and `NSInner()` at file scope answered the nested type in
// all three engines where swiftc 6.1.2 answers the top-level one. Two nested types
// of the same name at DIFFERENT depths collided in that one binding too, and there
// the two halves DISAGREED: the compiler resolved both correctly through typeKeyOf
// while the interpreter read whichever ran last.
struct NSInner { func who() -> Int { return 1 } }
enum NSMode: Int { case on = 1 }
struct NSOuter {
    struct NSInner { func who() -> Int { return 2 } }
    enum NSMode: Int { case on = 2 }
    class NSBase { func f() -> Int { return 20 } }
    class NSCub: NSBase { override func f() -> Int { return 200 + super.f() } }
    struct Deeper { struct NSInner { func who() -> Int { return 3 } }
                    func mk() -> Int { return NSInner().who() } }
    static func mk() -> Int { return NSInner().who() * 10 + NSMode.on.rawValue }
    static func cub() -> Int { return NSCub().f() }
}
// `catch let e as T` used to parse the `as T` and DISCARD it, so the clause bound e
// and matched EVERY thrown value - the swift arm of docs/todo.md 1.1's sweep.
// `as Error` still matches everything, which is what swiftc does. A user PROTOCOL
// used to match everything too, because conformance was on no value in either half;
// it is recorded now (docs/todo.md 1.1) and the clause filters - see PCat below.
// ----- protocol conformance and dispatch inside an extension (docs/todo.md 1.1) --
// `x is P` was FALSE for every conforming type in all three engines, so --cross was
// blind to it and the catch clause above had to exempt user protocols. And the
// dispatch rule inside a protocol extension is not "always dynamic": a REQUIREMENT
// dispatches to the conforming type's override, a member only the extension
// introduces does not. swiftc 6.1.2 settles every row here.
//
// ONE ROW IS DELIBERATELY NOT ASSERTED, and it is the value model rather than any
// of this: reading an extension-only member off a variable whose DECLARED type is
// the protocol (`let d: PGreet = PS3(); d.pplain()`) binds to the extension in
// swiftc and to PS3's own here, because that decision needs the expression's static
// type and this front end has none - the same wall that keeps `Optional(3)` out
// (docs/working-on-this-project.md 7.9). Reached through the concrete type, which is
// what pss[0] below is, every engine agrees with swiftc.
protocol PGreet {
    func ptag() -> String
}
extension PGreet {
    func phi() -> String { return "hi:" + ptag() }
    func ptag() -> String { return "?" }
    func pplain() -> String { return "p" }
    func pvia() -> String { return "v:" + pplain() + self.pplain() }
}
protocol PSub: PGreet { }
struct PS1: PGreet { func ptag() -> String { return "A" } }
struct PS2: PGreet { }
class PC1: PGreet { func ptag() -> String { return "C" } }
class PC2: PC1 { }
enum PE: String, PSub { case one = "1"
    func ptag() -> String { return "E" }
}
struct PS3 { }
extension PS3: PGreet { func ptag() -> String { return "X" }
                        func pplain() -> String { return "own" } }
extension Int: PGreet { func ptag() -> String { return "I" }
                        func pmine() -> String { return "int" } }
struct PS4 { func pmine() -> String { return "struct" } }
struct PCat: Error, PSub { }
struct PDog: Error { }
func pthrow(_ n: Int) throws -> Int {
    if n == 0 { throw PCat() }
    throw PDog()
}
func pcatch(_ n: Int) -> String {
    do { _ = try pthrow(n) } catch is PSub { return "sub" } catch { return "any" }
    return "none"
}
// Two nested types of the same name at different depths AND of different KINDS.
// classNames/enumNames are keyed by the bare name, so `Mode()` inside Sub took the
// enum rawValue arm and aborted with `list index out of range` in both compiled
// engines while the interpreter aborted on the clobbered bare binding.
struct NSKind {
    enum Mode: Int { case on = 1 }
    struct Sub { struct Mode { func t() -> Int { return 9 } }
                 func mk() -> Int { return Mode().t() } }
    static func mk() -> Int { return Mode.on.rawValue }
}
// A nested type constructed by a stored property's DEFAULT, by a memberwise default
// and by a static initialiser: none of those runs in the enclosing type's own frame -
// planArgs materialises a callee's defaults as argument records at the CALL SITE, and
// a static initialiser used to run before the nested declaration had been installed.
struct NSDef { var q = 7 }
struct NSHost {
    struct NSDef { var q = 1 }
    var x = NSDef()
    static let s = NSDef()
    var arr: [NSDef] = [NSDef()]
    func get() -> Int { return x.q + arr[0].q }
}
// A default ARGUMENT naming a nested type, and a LOCAL that shadows one. Neither
// runs in the enclosing type's frame: planArgs materialises defaults at the call site,
// and `let NSDef = 42` is legal Swift that must beat the nested type.
struct NSArg {
    struct NSDef { var q = 1 }
    func f(_ v: NSDef = NSDef()) -> Int { return v.q }
    static func g(_ v: NSDef = NSDef()) -> Int { return v.q + 100 }
    func shadow() -> Int { let NSDef = 42; return NSDef }
}
// The same shadowing with the TOP-LEVEL type declared AFTER the nested one.
struct NSOuter2 { struct NSLater { func who() -> Int { return 5 } } }
struct NSLater { func who() -> Int { return 6 } }
enum CNet: Error { case timeout, refused(code: Int) }
enum CDisk: Error { case full }
struct CBoom: Error { let n: Int }
func cthrow(_ k: Int) throws -> String {
    if k == 0 { throw CNet.timeout }
    if k == 1 { throw CNet.refused(code: 42) }
    if k == 2 { throw CDisk.full }
    if k == 3 { throw CBoom(n: 7) }
    return "ok"
}
func ccatch(_ k: Int) -> String {
    do { return try cthrow(k) }
    catch CNet.refused(let c) { return "refused\(c)" }
    catch let e as CNet { return "net" }
    catch is CDisk { return "disk" }
    catch let e as CBoom { return "boom\(e.n)" }
    catch { return "other" }
}
func ccatchAny(_ k: Int) -> String {
    do { return try cthrow(k) } catch let e as Error { return "any" }
}
func ccatchOuter(_ k: Int) -> String {
    do {
        do { return try cthrow(k) } catch let e as CDisk { return "inner-disk" }
    } catch { return "outer" }
    return "?"
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
    // A LOOP VARIABLE IS A FRESH BINDING PER ITERATION (docs/todo.md 1.1). Swift
    // binds the element with a `let` inside the body, so the closure made in
    // iteration k captures iteration k's cell: swiftc 6.1.2 answers 0 / 2 where both
    // compiled halves answered 3 / 3 from one shared slot. The interpreter half was
    // right, so --cross was blind to it - nothing here had ever captured a loop
    // variable. The full-syntax file's SECTION 38 walks every loop form; these four
    // are the ones cheap enough for the matrix.
    var caps: [() -> Int] = []
    for i in 0..<3 { caps.append({ i }) }
    check("closure-captures-for-in-per-iteration", caps[0]() == 0 && caps[2]() == 2)
    var lcaps: [() -> Int] = []
    for v in [10, 20, 30] { lcaps.append({ v }) }
    check("closure-captures-for-in-list-per-iteration", lcaps[0]() == 10 && lcaps[2]() == 30)
    // ...and the opposite property, which a careless fix breaks: a var declared
    // OUTSIDE the loop is still one cell, shared by the body and by every closure.
    var outerCaps: [() -> Int] = []
    var outerVar = 0
    for _ in 0..<3 {
        outerCaps.append({ outerVar })
        outerVar += 1
    }
    check("closure-captures-outer-var-shared", outerVar == 3
          && outerCaps[0]() == 3 && outerCaps[2]() == 3)
    // ...and a var declared INSIDE the body is one cell for that iteration, so a
    // mutating closure and the code beside it see each other.
    var inLog = ""
    for i in 0..<2 {
        var n = 0
        let inc = { n += 1 }
        inc()
        inc()
        inLog += "\(i)\(n)"
    }
    check("closure-captures-body-var-shared-within-iteration", inLog == "0212")

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

    // ----- member overloads picked by argument TYPE (docs/todo.md 1.2) -----
    let ovInts: [Int] = [1, 2]
    let ovDbls: [Double] = [1.5]
    let ovStrs: [String] = ["a"]
    let ovBools: [Bool] = [true]
    check("init-overload-type", MB(ovInts[0]).s == "I" && MB(ovDbls[0]).s == "D"
                                && MB(ovStrs[0]).s == "S" && MB(ovBools[0]).s == "B")
    check("init-overload-label", MB(a: ovInts[0]).s == "lI" && MB(a: ovDbls[0]).s == "lD")
    let ovm = OvM()
    check("method-overload-type", ovm.m(ovInts[0]) == "I" && ovm.m(ovDbls[0]) == "D"
                                  && ovm.m(ovStrs[0]) == "S")
    check("method-overload-arity", ovm.m(ovInts[0], ovInts[1]) == "II")
    check("subscript-overload-type", ovm[ovInts[0]] == "sI" && ovm[ovStrs[0]] == "sS")

    // ----- a function-typed SLOT adopts its arguments (docs/todo.md 1.4) -----
    check("fn-slot-arg", fslot(3) == 1.5 && fslot2(3, 2) == 1.5)
    check("fn-field-call", FnField(fn: { x in x / 2 }).fn(3) == 1.5)

    // ----- a NEW dictionary key adopts the declared element type (todo 1.4) -----
    var newk: [String: Double] = [:]
    newk["b"] = 5
    var newk2: [String: Double] = ["a": 1]
    newk2["b"] = 5
    var newki: [Int: Double] = [:]
    newki[1] = 5
    var newkw: [String: Int8] = [:]
    newkw["q"] = 100
    check("dict-newkey-double", newk["b"]! / 2 == 2.5 && newk2["b"]! / 2 == 2.5
                                && newki[1]! / 2 == 2.5)
    check("dict-newkey-width", newkw["q"]! &+ 100 == -56)

    // ----- `case is T:` - a TYPE pattern (docs/todo.md 1.4) -----
    let isAnys: [Any] = [3.0, "s", true, IsE.a, 3]
    check("case-is", isKind(isAnys[0]) == "D" && isKind(isAnys[1]) == "S"
                     && isKind(isAnys[2]) == "B" && isKind(isAnys[3]) == "E"
                     && isKind(isAnys[4]) == "I")

    // ----- `3 as Int8` is a COERCION, not a no-op -----
    let asAny: [Any] = [3 as Int8, 3 as Double]
    check("as-coerces", (3 as Double) / 2 == 1.5 && (250 as UInt8) &+ 10 == 4
                        && (asAny[0] is Int8) && !(asAny[0] is Int) && (asAny[1] is Double))

    // ----- a subclass with no init of its own INHERITS one (docs/todo.md 1.3) -----
    // A class never gets a memberwise initializer - that is structs - and the empty
    // one that used to be synthesised here shadowed the inherited init, so every
    // inherited property came back nil.
    check("inherit-init", Square(w: 2, h: 3).area() == 6 && Square(w: 2, h: 3).sides == 4)

    // ----- combined pipeline -----
    check("combined-pipeline", transform([1, 2, -3]) == "o1e2x")

    // ----- docs/todo.md 1.2 -----
    let ovInts: [Int] = [3]
    let ovStrs: [String] = ["x"]
    check("param-struct-adopts", adoptArrParam([3]) == 1.5 && adoptDictParam(["k": 3]) == 1.5)
    check("param-fn-adopts", adoptFnParam({ x in x / 2 }) == 1.5
                             && adoptNoLeafParam(["x"]) == "x")
    var appD: [Double] = []
    appD.append(3)
    var appD2: [Double] = [1.0]
    appD2.append(5)
    check("append-adopts", appD[0] / 2 == 1.5 && appD2[1] / 2 == 2.5)
    let nestedFns: [(Double) -> Double] = [{ x in x / 2 }]
    check("nested-fn-type-adopts", nestedFns[0](3) == 1.5)
    var iv = ovInts[0]
    var sv = ovStrs[0]
    inoutOv(&iv)
    inoutOv(&sv)
    check("inout-overload-by-type", iv == 103 && sv == "x!")
    let ovs = OvSub()
    check("inherited-overload-group", ovs.pick(ovInts[0]) == "sI" && ovs.pick(ovStrs[0]) == "bS")
    check("static-overloads", StatOv.g(ovInts[0]) == "I" && StatOv.g(ovStrs[0]) == "S")

    // ----- nested types (docs/todo.md 1.7) -----
    check("nested-same-name-memberwise", NA.Inner(x: 3).t() == 3 && NB.Inner(x: 3).t() == 30)
    check("nested-bare-inside-outer", NA.mk() == 4)
    check("nested-enum-same-name", NA.E.a.rawValue == 1 && NB.E.b.rawValue == 20)
    check("nested-three-levels", NC.B.C(v: 1).v == 1 && ND.B.C(v: 2).v == 2)
    check("nested-same-leaf-two-depths", NE.Inner(x: 1).x == 1 && NE.Inner.Inner(x: 2).k() == 102)
    check("nested-init-overloads", NF.Inner(x: 1).v == 1 && NF.Inner(y: "abcd").v == 4
                                    && NG.Inner(z: 2).v == 102)
    check("extension-of-nested", NH.Inner(x: 5).dbl() == 10 && NH.Inner.s() == 77
                                  && NH.Inner(twice: 3).x == 6 && NH.Inner(x: 5)[2] == 7)
    check("nested-enum-in-enum", NI.Mode.slow.rawValue == 4 && NI.Mode.fast.tag() == 13)
    check("static-reads-own-static", NJ.f() == 6 && NJ.Inner.f() == 51)
    let nkv: NK.Inner = NK.Inner(x: 1)
    let nlv: [NL.Inner] = [NL.Inner(x: 2)]
    check("nested-as-type-and-witness", ntake(nkv) == "K1" && ntake(nlv[0]) == "L2")
    check("nested-static-subscript", NK.Inner[4] == 8 && NL.Inner[4] == 12)
    check("nested-does-not-shadow-toplevel", NSInner().who() == 1 && NSMode.on.rawValue == 1)
    check("nested-qualified-still-reaches-nested",
          NSOuter.NSInner().who() == 2 && NSOuter.NSMode.on.rawValue == 2)
    check("nested-wins-unqualified-inside-outer", NSOuter.mk() == 22)
    check("nested-wins-at-the-deeper-level", NSOuter.Deeper().mk() == 3)
    check("nested-superclass-is-the-nested-one", NSOuter.cub() == 220)
    check("nested-same-name-different-kinds", NSKind.mk() == 1 && NSKind.Sub().mk() == 9)
    check("nested-in-property-and-static-defaults",
          NSHost().get() == 2 && NSHost.s.q == 1 && NSDef().q == 7)
    check("nested-in-default-argument-and-local-shadow",
          NSArg().f() == 1 && NSArg.g() == 101 && NSArg().shadow() == 42)
    check("toplevel-declared-after-nested-still-wins",
          NSLater().who() == 6 && NSOuter2.NSLater().who() == 5)

    // ----- typed catch (docs/todo.md 1.1, swift's arm) -----
    check("catch-as-type-discriminates",
          ccatch(0) == "net" && ccatch(1) == "refused42" && ccatch(2) == "disk"
            && ccatch(3) == "boom7" && ccatch(4) == "ok")
    check("catch-as-error-catches-everything",
          ccatchAny(0) == "any" && ccatchAny(3) == "any" && ccatchAny(4) == "ok")
    check("catch-as-type-rethrows-to-outer",
          ccatchOuter(2) == "inner-disk" && ccatchOuter(0) == "outer" && ccatchOuter(4) == "ok")

    // ----- protocol conformance / extension dispatch (docs/todo.md 1.1) -----
    let pvals: [Any] = [PS1(), PS2(), PC1(), PC2(), PE.one, PS3(), 7, "z"]
    var pIs = ""
    var pSub = ""
    for v in pvals {
        pIs += (v is PGreet) ? "1" : "0"
        pSub += (v is PSub) ? "1" : "0"
    }
    check("protocol-is-conforming-type", pIs == "11111110" && pSub == "00001000")
    var pAs = ""
    for v in pvals { if let p = v as? PGreet { pAs += p.ptag() } else { pAs += "-" } }
    check("protocol-as-optional-cast", pAs == "A?CCEXI-")
    let pps: [PGreet] = [PS1(), PS2(), PC1(), PE.one, PS3()]
    var pHi = ""
    for p in pps { pHi += p.phi() + "|" }
    check("protocol-requirement-dispatches-dynamically", pHi == "hi:A|hi:?|hi:C|hi:E|hi:X|")
    let pss = [PS3()]
    check("protocol-extension-only-member-is-static",
          pss[0].pvia() == "v:pp" && pss[0].pplain() == "own")
    let pns = [7]
    check("protocol-conformance-of-a-builtin",
          pns[0].ptag() == "I" && pns[0].phi() == "hi:I")
    check("builtin-extension-does-not-hijack-a-type-member",
          [PS4()][0].pmine() == "struct" && pns[0].pmine() == "int")
    check("catch-filters-on-a-user-protocol",
          pcatch(0) == "sub" && pcatch(1) == "any")
    // A COMPOSITION tests every component: PS1 conforms to PGreet alone.
    check("protocol-composition-tests-every-component",
          (pvals[4] is PGreet & PSub) && !(pvals[0] is PGreet & PSub))

    // ----- the Sequence/Collection members (docs/todo.md 2.10) -----
    // Nineteen members aborted in every engine and with TWO DIFFERENT WORDINGS -
    // "unknown Array method: sorted" here against "unknown list method 'sorted'"
    // under llvm.Run, which is the shared table in abnf/jsrt.go. SECTION 36 of
    // tests/swift-test-full.swift carries the whole set and swiftc 6.1.2 passes it
    // as written; these are the ones with a rendering the matrix should watch,
    // because a Range and a labelled tuple are new PRINTED shapes and the
    // goja/-frozen pair is the only gate that can see a divergence in one.
    let sq = [3, 1, 2]
    check("seq-sorted-min-max", sq.sorted() == [1, 2, 3] && sq.min() == 1 && sq.max() == 3)
    check("seq-slices", sq.prefix(2) == [3, 1] && sq.dropFirst() == [1, 2])
    check("seq-reduce-operator-as-value", sq.reduce(0, +) == 6 && sq.sorted(by: >) == [3, 2, 1])
    check("seq-range-prints-as-itself", "\(sq.indices)" == "0..<3" && "\(1...3)" == "1...3")
    check("seq-range-count-and-equality", sq.indices.count == 3 && sq.indices == 0..<3)
    check("seq-enumerated-is-a-labelled-tuple",
          "\(Array(sq.enumerated()))" == "[(offset: 0, element: 3), (offset: 1, element: 1), (offset: 2, element: 2)]")
    check("seq-string-is-a-collection",
          "hello".filter { $0 != "l" } == "heo" && "hello".map { String($0) }.count == 5)
    check("seq-one-sided-range-subscript", Array(sq[1...]) == [1, 2] && Array(sq[..<2]) == [3, 1])
    check("seq-shuffled-is-a-permutation", sq.shuffled().sorted() == [1, 2, 3])
    check("seq-contains-where", sq.contains(where: { $0 > 2 }) && !sq.contains(where: { $0 > 3 }))
}

// ----- The TOP LEVEL, deliberately outside main() -------------------------
// Every other check in this file runs inside main(). These run where the program
// itself does, and that is the environment docs/todo.md 1.12 changed: the
// compiler half now emits the WHOLE program inside one js_try, whose catch calls
// js_swtop, so that an error nobody caught is reported by the program instead of
// escaping the module and being rendered "[object Object]" by the floor's generic
// uncaught-throw handler. That wrapping puts the top level inside an IR CLOSURE,
// so the four things it could plausibly have broken are pinned here: a top-level
// binding stays visible to a top-level function, a top-level type is still
// usable, a top-level do/catch still wins over the new outer try, and try? still
// answers nil rather than being swallowed by it.
//
// DISCRIMINATING POWER AGAINST ec39aef: ZERO, and that is the honest number.
// These four pass at base too - base simply had no wrapper to break them. They
// are here because nothing else in the suite executes a do/catch at the top
// level, so without them the wrapper's blast radius is untested. What DOES
// discriminate is the abort text itself, and the only vehicle for that is a
// SHOULD-ABORT row in .vscode/launch.json (see tests/swift-abort-test*.swift).
enum TopErr: Error { case boom(Int) }
let topLet = 7
struct TopS { var v: Int }
func topReads() -> Int { return topLet + 1 }
func topThrows() throws -> Int { throw TopErr.boom(1) }
var topLog = ""
do {
    throw TopErr.boom(3)
} catch TopErr.boom(let n) {
    topLog += "boom\(n);"
} catch {
    topLog += "other;"
}
for i in 0..<3 { topLog += "\(i)" }
let topOpt = try? topThrows()
check("top-level-binding-visible-to-a-top-level-func", topReads() == 8)
check("top-level-struct-usable-at-top-level", TopS(v: 2).v == 2)
check("top-level-do-catch-wins-over-the-program-wide-try", topLog == "boom3;012")
check("top-level-try-optional-still-answers-nil", topOpt == nil)

// ----- An ASTRAL literal, and the Collection PROPERTIES --------------------
// This file is what ./test.sh --cross diffs the two halves with, and it had no
// astral character anywhere - which is why a literal that meant two different
// things in the two halves survived. A raw astral character is TWO code units,
// and copying a literal one unit at a time hands the host a LONE SURROGATE that
// cannot be encoded in UTF-8: `"a<emoji>b".count` was 4 in the interpreter and 3
// under llvm.Run, where the same string spelled \u{1F600} was 3 in both. swiftc
// 6.1.2 says 3 for both spellings and says they are equal.
//
// isEmpty / first / last rode along: all three are properties on every
// Collection and all three answered nil in BOTH halves, where swiftc says false
// and a value - agree-and-wrong, which byte-identity cannot see by construction.
let astRaw = "a😀b"
let astEsc = "a\u{1F600}b"
let astComb = "cafe\u{301}"
struct AstOwn { var isEmpty = 99; var first = 7; var last = 8 }
check("a-raw-astral-literal-is-the-escape-spelling", astRaw == astEsc && astRaw.count == 3)
check("an-astral-pair-is-one-cluster-in-every-leg",
      astEsc.count == 3 && Array(astRaw).count == 3
      && Array(astRaw)[1] == Array(astEsc)[1] && String(Array(astRaw)[1]).count == 1)
check("a-combining-mark-joins-the-cluster-before-it",
      astComb.count == 4 && Array(astComb)[3] == "e\u{301}")
var astIter = ""
for c in astRaw { astIter += String(c) }
check("for-in-over-a-string-walks-clusters", astIter == astRaw && astIter.count == 3)
check("isEmpty-is-a-Bool-not-nil",
      !astRaw.isEmpty && "".isEmpty && ![1].isEmpty && ([] as [Int]).isEmpty)
check("first-and-last-are-properties-too",
      astRaw.first == "a" && astRaw.last == "b" && [1, 2].first == 1 && [1, 2].last == 2)
check("first-and-last-of-an-empty-collection-are-nil",
      "".first == nil && ([] as [Int]).last == nil)
check("a-user-member-of-those-names-is-not-hijacked",
      AstOwn().isEmpty == 99 && AstOwn().first == 7 && AstOwn().last == 8)

// ----- A TOP-LEVEL defer (docs/todo.md 2.11) ------------------------------
// A `defer` among the top-level statements used to kill BOTH halves with a HOST
// error rather than a diagnostic - the marker record reached a call site that
// expected a thunk. The interpreter said "TypeError: Not a function:
// [object Object]" under goja and "call of a non function value" under -frozen
// (so the crash was itself a live goja-vs--frozen divergence) and llvm.Run said
// "TypeError: Not a function ... compile-core.js:617". A block defer worked in
// every engine; only the top level did not.
//
// swiftc 6.1.2 settles the semantics, and they are NOT the block's:
//   * top-level defers run at the END of the top level, LIFO;
//   * an uncaught error at top level TRAPS, so they do NOT run on that path
//     (the block form fires its defer on a throw, because a function unwinds);
//   * exit() does not run them either.
// The summary print and exit(fails) therefore live in the FIRST-registered
// defer, which is the LAST to run: nothing after exit() can observe anything.
// A FIFO order would spell "12" instead of "21", and a defer that ran where it
// was written would spell it before the loop's letters.
var tldLog = ""
defer {
    check("top-level-defers-run-at-the-end-LIFO", tldLog == "wxb0l0b1l121")
    print("features: \(checks) checks, \(fails) failures")
    exit(fails)
}
defer { tldLog += "1" }
if topLet == 7 {
    defer { tldLog += "x" }
    tldLog += "w"
}
for i in 0..<2 {
    defer { tldLog += "l\(i)" }
    tldLog += "b\(i)"
}
defer { tldLog += "2" }

main()
