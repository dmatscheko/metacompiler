// Full-syntax test: Swift (5.10 core grammar).
//
// This file belongs to the SECOND test group (./test.sh --full): it is NOT part
// of the default matrix. The goal of the metacompiler is to support the full
// languages; this file is the ratchet that measures how far the swift grammars
// are. It walks the whole practical Swift 5.9/5.10 syntax, one self-contained
// SECTION per language area. The --full runner runs the file, and whenever a
// grammar aborts it removes the section around the error and retries - so the
// report lists every unsupported section, not just the first.
//
// Conventions (shared by every *-test-full.* file):
//   - prologue (before the first SECTION marker): the check helper only
//   - each section: '// ===== SECTION <nn>: <name> =====', top-level,
//     self-contained, no references to other sections
//   - main() calls each section via a line tagged 'SECTION-CALL <nn>' and
//     prints the summary line 'full: <checks> checks, <failures> failures';
//     the file ends features-style with main() and exit(fails)
//
// Deliberately out of scope (not syntax, or unrunnable in this harness):
// imports and Foundation - stdlib usage strictly mirrors the features file
// (Int/String/Bool/Array/Dictionary basics, print/exit/abs/min/max and
// map/filter/forEach) - so no CaseIterable/Equatable/Comparable/Codable
// conformances (own protocols carry the generic constraints), no key paths
// (\.p is the stdlib KeyPath type) and no stride. Concurrency at RUNTIME:
// async functions and one actor are defined and type-checked, never awaited.
// Also out: macros (#... / @attached), result builders (@resultBuilder),
// ARC observation (deinit is declared, its timing never asserted), unsafe
// pointers, ObjC interop (@objc/dynamic), @available/#available.
//
// Hand-written for the metacompiler project (Apache-2.0, no copied test-suite
// code), organized after The Swift Programming Language (5.10) grammar
// summary with the ANTLR grammars-v4 Swift grammar as a coverage checklist.

var fails = 0
var checks = 0

func check(_ id: String, _ cond: Bool) {
    checks += 1
    if !cond {
        print("FAIL \(id)")
        fails += 1
    }
}

// ===== SECTION 01: baseline =====
// Condensed re-assertion of the feature-matrix basics this file builds on.
func mulBas(_ a: Int, by b: Int) -> Int { return a * b }
func s01() {
    var total = 0
    for i in 1...4 { total += i }
    check("bas1", total == 10 && 2 + 3 * 4 == 14 && 7 % 3 == 1)
    let words = ["a", "bb"]
    var ages = ["ann": 3]
    ages["bo"] = 5
    check("bas2", words[1].count == 2 && ages["ann"] == 3 && ages["bo"] == 5)
    check("bas3", mulBas(3, by: 7) == 21 && "v=\(2 + 5)" == "v=7")
    let inc = { (x: Int) -> Int in x + 1 }
    check("bas4", inc(41) == 42 && (3 > 2 ? "y" : "n") == "y")
    var g = ""
    switch words.count {
    case 1, 2:
        g = "lo"
    default:
        g = "hi"
    }
    check("bas5", g == "lo")
}

// ===== SECTION 02: numeric literal forms =====
func s02() {
    check("num1", 0xFF == 255 && 0o17 == 15 && 0b1010 == 10)
    check("num2", 1_000_000 == 1000000 && 0xFF_EC == 65516 && 0b1_0000 == 16)
    check("num3", 1.25e2 == 125.0 && 25e-2 == 0.25 && 10_000.000_1 == 10000.0001)
    check("num4", 0xFp2 == 60.0 && 0xC.8p0 == 12.5) // hex floats: 15*4, 12.5*1
}

// ===== SECTION 03: string literals =====
func s03() {
    check("str1", "\u{48}\u{69}" == "Hi" && "a\tb\nc\0d\\e\"f\'g".count == 13)
    check("str2", "\u{1F600}".count == 1) // one extended grapheme cluster
    check("str3", "e\u{301}" == "\u{E9}" && "e\u{301}".count == 1) // canonical equivalence
    let multi = """
    line1
      line2
    """
    check("str4", multi == "line1\n  line2") // closing-delimiter indentation stripped
    check("str5", #"raw \n \#(1 + 1)"# == "raw \\n 2")
    check("str6", ##"quote "#" here"##.count == 14)
    let ch: Character = "ü"
    check("str7", "\(ch)" == "ü" && "\("a" + "b")c" == "abc")
}

// ===== SECTION 04: tuples =====
func minMax04(_ a: Int, _ b: Int) -> (lo: Int, hi: Int) { (min(a, b), max(a, b)) }
func s04() {
    let t = (404, "found")
    let named = (code: 7, ok: true)
    check("tup1", t.0 == 404 && t.1 == "found" && named.code == 7 && named.ok)
    let (a, _, c) = (1, 2, 3)
    check("tup2", a == 1 && c == 3)
    let mm = minMax04(4, 1)
    check("tup3", mm.lo == 1 && mm.hi == 4)
    var x = 1
    var y = 2
    (x, y) = (y, x)
    check("tup4", x == 2 && y == 1)
    let pair: (n: String, v: Int) = ("k", 9)
    check("tup5", pair.n == "k" && pair.1 == 9)
}

// ===== SECTION 05: optionals =====
struct Engine05 {
    var power = 90
    func label() -> String { "e" }
}
struct Car05 { var engine: Engine05? }
func halveEven05(_ n: Int?) -> Int {
    guard let v = n, v % 2 == 0 else { return -1 }
    return v / 2
}
func s05() {
    var maybe: Int? = nil
    check("opt1", maybe == nil && (maybe ?? 42) == 42)
    maybe = 7
    check("opt2", maybe! == 7 && (maybe ?? 42) == 7)
    var shorthand = 0
    if let maybe { shorthand = maybe } // 5.7 shorthand shadow binding
    check("opt3", shorthand == 7)
    check("opt4", halveEven05(8) == 4 && halveEven05(7) == -1 && halveEven05(nil) == -1)
    var countdown: Int? = 3
    var walked = 0
    while let n = countdown {
        walked += n
        countdown = n > 1 ? n - 1 : nil
    }
    check("opt5", walked == 6)
    let c1 = Car05(engine: Engine05())
    check("opt6", c1.engine?.power == 90 && Car05(engine: nil).engine?.power == nil && c1.engine?.label() == "e")
    let grid: [[Int]]? = [[5]]
    let sure: Int! = 12 // implicitly unwrapped
    check("opt7", grid?[0][0] == 5 && sure + 1 == 13)
    var picked = 0
    for case let n? in [3, nil, 4] { picked += n } // optional pattern
    let chain: Int? = nil
    check("opt8", picked == 7 && (chain ?? maybe ?? 0) == 7)
}

// ===== SECTION 06: enumerations =====
enum Planet06: Int { case mercury = 1, venus, earth }
enum Roman06: String { case two = "II", three } // raw value defaults to the case name
enum Coin06 {
    case heads, tails
    func flip() -> Coin06 { self == .heads ? .tails : .heads }
    var label: String { self == .heads ? "H" : "T" }
}
enum Shape06 {
    case circle(r: Int)
    case rect(Int, Int)
}
func area06(_ s: Shape06) -> Int {
    switch s {
    case .circle(let r): return r * r * 3
    case .rect(let w, let h): return w * h
    }
}
enum Tree06 {
    case leaf(Int)
    indirect case node(Tree06, Tree06)
}
func total06(_ t: Tree06) -> Int {
    switch t {
    case .leaf(let n): return n
    case let .node(l, r): return total06(l) + total06(r)
    }
}
func s06() {
    check("enu1", Planet06.venus.rawValue == 2 && Roman06.two.rawValue == "II" && Roman06.three.rawValue == "three")
    check("enu2", Planet06(rawValue: 3) == .earth && Planet06(rawValue: 9) == nil)
    check("enu3", Coin06.heads.flip() == .tails && Coin06.tails.label == "T")
    check("enu4", area06(.circle(r: 5)) == 75 && area06(.rect(4, 2)) == 8)
    check("enu5", total06(.node(.leaf(1), .node(.leaf(2), .leaf(3)))) == 6)
    // CaseIterable/allCases: stdlib-synthesized, out of the mirrored stdlib surface
}

// ===== SECTION 07: structs vs classes =====
struct PointV07 {
    var x: Int
    var y: Int
    mutating func reset() { self = PointV07(x: 0, y: 0) } // self reassignment
}
final class Cell07 {
    var v: Int
    init(_ v: Int) { self.v = v }
    deinit { } // declared for syntax; ARC timing never asserted
}
func s07() {
    let p1 = PointV07(x: 1, y: 2) // memberwise init
    var p2 = p1 // value copy
    p2.x = 99
    check("svc1", p1.x == 1 && p2.x == 99)
    p2.reset()
    check("svc2", p2.x == 0 && p2.y == 0)
    let c1 = Cell07(5)
    let c2 = c1 // reference alias
    c2.v = 42
    check("svc3", c1.v == 42)
    check("svc4", c1 === c2 && c1 !== Cell07(5))
    let konst = PointV07(x: 3, y: 4) // let struct: whole value is constant
    check("svc5", konst.x + konst.y == 7)
}

// ===== SECTION 08: properties =====
struct Gauge08 {
    var log = ""
    var level: Int = 0 { willSet { log += "w\(newValue)" } didSet(old) { log += "d\(old)" } }
    var scaled: Int { get { level * 10 } set(raw) { level = raw / 10 } }
    static let unit = "u"
}
struct LazyBox08 {
    static var builds = 0
    lazy var payload: String = {
        LazyBox08.builds += 1
        return "p\(LazyBox08.builds)"
    }()
}
class Meter08 { class var kind: String { "meter" } }
@propertyWrapper struct Capped08 {
    private var v: Int
    var wrappedValue: Int { get { v } set { v = min(newValue, 9) } }
    var projectedValue: Bool { v == 9 }
    init(wrappedValue: Int) { v = min(wrappedValue, 9) }
}
struct Player08 { @Capped08 var score = 3 }
func s08() {
    var g = Gauge08()
    g.level = 3
    g.level = 5
    check("prp1", g.log == "w3d0w5d3")
    g.scaled = 70
    check("prp2", g.level == 7 && g.scaled == 70)
    check("prp3", Gauge08.unit == "u" && Meter08.kind == "meter")
    var lb = LazyBox08()
    check("prp4", LazyBox08.builds == 0) // not built yet
    let first = lb.payload
    let again = lb.payload
    check("prp5", first == "p1" && again == "p1" && LazyBox08.builds == 1)
    var p = Player08()
    check("prp6", p.score == 3 && p.$score == false)
    p.score = 50
    check("prp7", p.score == 9 && p.$score == true)
}

// ===== SECTION 09: initializers =====
struct Ratio09 {
    var num: Int
    var den: Int
    init?(num: Int, den: Int) { // failable
        if den == 0 { return nil }
        self.num = num
        self.den = den
    }
    init(whole: Int) { self.init(num: whole, den: 1)! } // delegation, force-unwrapped
}
class Vehicle09 {
    var wheels: Int
    required init(wheels: Int) { self.wheels = wheels }
    convenience init() { self.init(wheels: 4) }
}
class Trike09: Vehicle09 {
    var bell: Bool
    init(bell: Bool) {
        self.bell = bell
        super.init(wheels: 3) // designated delegates up
    }
    required init(wheels: Int) { // required re-stated alongside new designated inits
        self.bell = false
        super.init(wheels: wheels)
    }
}
struct Size09 { var w = 2, h = 3 }
func s09() {
    check("ini1", Size09().w == 2 && Size09(h: 9).h == 9 && Size09(w: 1, h: 1).w == 1)
    check("ini2", Ratio09(num: 1, den: 0) == nil && Ratio09(num: 3, den: 4)?.num == 3)
    check("ini3", Ratio09(whole: 5).den == 1 && Vehicle09().wheels == 4)
    check("ini4", Trike09(bell: true).wheels == 3 && Trike09(bell: true).bell)
    check("ini5", Trike09(wheels: 6).wheels == 6) // via the required initializer
}

// ===== SECTION 10: functions =====
func greet10(_ name: String, from city: String = "Bern", loud: Bool = false) -> String {
    return loud ? name + "<" + city + "!" : name + "<" + city
}
func sum10(_ xs: Int...) -> Int { // variadic
    var t = 0
    for x in xs { t += x }
    return t
}
func bump10(_ n: inout Int, by k: Int) { n += k }
func twice10(_ n: Int) -> Int { n * 2 } // implicit single-expression return
func twice10(_ s: String) -> String { s + s } // overload by parameter type
func compose10() -> (Int) -> Int {
    func inner(_ x: Int) -> Int { x + 1 } // nested function
    return inner
}
@discardableResult func tick10(_ n: Int) -> Int { n + 1 }
func s10() {
    check("fun1", greet10("a") == "a<Bern" && greet10("a", from: "x", loud: true) == "a<x!")
    var n = 10
    bump10(&n, by: 5)
    check("fun2", sum10() == 0 && sum10(1, 2, 3) == 6 && n == 15)
    check("fun3", twice10(3) == 6 && twice10("ab") == "abab")
    let f: (Int) -> Int = compose10()
    check("fun4", f(4) == 5)
    tick10(1) // result discarded without warning
    check("fun5", tick10(2) == 3)
}

// ===== SECTION 11: closures =====
func runBoth11(work: () -> Int, cleanup: () -> Int) -> Int { work() + cleanup() }
func s11() {
    let full = { (x: Int, y: Int) -> Int in return x + y }
    let inferred: (Int, Int) -> Int = { x, y in x * y }
    let short: (Int, Int) -> Int = { $0 - $1 }
    check("clo1", full(1, 2) == 3 && inferred(3, 4) == 12 && short(9, 4) == 5)
    let mapped = [1, 2, 3].map { $0 * 3 } // trailing closure
    check("clo2", mapped.count == 3 && mapped[2] == 9)
    check("clo3", (runBoth11 { 30 } cleanup: { 12 }) == 42) // multiple trailing closures
    var base = 1
    let byRef = { base * 10 }
    let byVal = { [base] in base * 10 } // capture list snapshots
    base = 5
    check("clo4", byRef() == 50 && byVal() == 10)
    var queued: [() -> Int] = []
    func keep11(_ f: @escaping () -> Int) { queued.append(f) }
    keep11 { 20 }
    keep11 { 22 }
    check("clo5", queued[0]() + queued[1]() == 42)
    var probes = 0
    func probe11() -> Bool {
        probes += 1
        return true
    }
    func orElse11(_ a: Bool, _ b: @autoclosure () -> Bool) -> Bool { a ? true : b() }
    let r1 = orElse11(true, probe11()) // argument not evaluated
    let r2 = orElse11(false, probe11())
    check("clo6", r1 && r2 && probes == 1)
}

// ===== SECTION 12: subscripts =====
struct Grid12 {
    var cells = [0, 0, 0, 0]
    subscript(i: Int) -> Int { get { cells[i] } set { cells[i] = newValue } }
    subscript(r: Int, c: Int) -> Int { get { cells[r * 2 + c] } set { cells[r * 2 + c] = newValue } }
    static subscript(n: Int) -> Int { n * 3 }
}
func s12() {
    var g = Grid12()
    g[0] = 5
    g[1, 1] = 9 // multi-parameter subscript
    check("sub1", g[0] == 5 && g.cells[0] == 5 && g[1, 1] == 9 && g[3] == 9)
    g[0] += 1 // compound assignment through a subscript
    check("sub2", g[0] == 6 && Grid12[4] == 12)
}

// ===== SECTION 13: inheritance =====
class Beast13 {
    var legs: Int
    init(legs: Int) { self.legs = legs }
    func noise() -> String { "..." }
    var family: String { "beast" }
    func intro() -> String { family + "/" + noise() + "/\(legs)" }
}
class Dog13: Beast13 {
    init() { super.init(legs: 4) }
    override func noise() -> String { "woof" }
    override var family: String { "canine+" + super.family }
    final func fetch() -> String { "ball" }
}
final class Snake13: Beast13 {
    init() { super.init(legs: 0) }
    override func noise() -> String { "sss" }
}
func s13() {
    let pets: [Beast13] = [Dog13(), Snake13(), Beast13(legs: 2)]
    check("inh1", pets[0].intro() == "canine+beast/woof/4") // dynamic dispatch
    check("inh2", pets[1].noise() == "sss" && pets[2].noise() == "...")
    check("inh3", pets[0] is Dog13 && !(pets[1] is Dog13))
    check("inh4", (pets[0] as? Dog13)?.fetch() == "ball" && (pets[1] as? Dog13) == nil)
    let up = Dog13() as Beast13 // guaranteed upcast
    check("inh5", (pets[0] as! Dog13).legs == 4 && up.family == "canine+beast")
}

// ===== SECTION 14: protocols =====
protocol Named14 {
    var name: String { get }
    func describe() -> String
    static func kind() -> String
}
extension Named14 {
    func describe() -> String { "n:" + name } // default implementation
}
protocol Aged14: Named14 { var age: Int { get } } // protocol inheritance
struct Person14: Aged14 {
    let name: String
    let age: Int
    static func kind() -> String { "person" }
}
struct Robot14 { let serial: Int }
extension Robot14: Named14 { // conformance added via extension
    var name: String { "r\(serial)" }
    func describe() -> String { "bot:" + name }
    static func kind() -> String { "robot" }
}
protocol Sound14 { var noise: String { get } }
struct Dog14: Named14, Sound14 {
    let name: String
    let noise = "woof"
    static func kind() -> String { "dog" }
}
func hear14(_ x: any Named14 & Sound14) -> String { x.name + ":" + x.noise } // composition
func s14() {
    let p = Person14(name: "ada", age: 36)
    let r = Robot14(serial: 5)
    check("pro1", p.describe() == "n:ada" && Person14.kind() == "person")
    check("pro2", r.describe() == "bot:r5" && Robot14.kind() == "robot")
    check("pro3", hear14(Dog14(name: "rex")) == "rex:woof")
    let crowd: [any Named14] = [p, r, Dog14(name: "rex")] // existentials
    var digest = ""
    for member in crowd { digest += member.describe() + ";" }
    let elder: any Aged14 = p
    check("pro4", digest == "n:ada;bot:r5;n:rex;" && elder.age == 36)
}

// ===== SECTION 15: extensions =====
extension Int {
    var squared15: Int { self * self }
    func clamped15(max m: Int) -> Int { self > m ? m : self }
}
struct Meter15 { var v: Int }
extension Meter15 {
    init(feet: Int) { self.init(v: feet * 3) } // extension init keeps memberwise
    var doubled15: Int { v * 2 }
}
protocol Tagged15 { var tag: String { get } }
extension Meter15: Tagged15 { var tag: String { "m\(v)" } }
struct Crate15<T> { let load: T }
extension Crate15 where T == Int { // constrained extension
    func heavier(than n: Int) -> Bool { load > n }
}
func s15() {
    check("ext1", 4.squared15 == 16 && 9.clamped15(max: 5) == 5 && 3.clamped15(max: 5) == 3)
    check("ext2", Meter15(feet: 2).v == 6 && Meter15(v: 1).doubled15 == 2 && Meter15(v: 3).tag == "m3")
    check("ext3", Crate15(load: 7).heavier(than: 5) && !Crate15(load: 3).heavier(than: 5))
}

// ===== SECTION 16: generics and opaque types =====
protocol Sized16 { var size: Int { get } }
struct Chip16: Sized16 { var size: Int }
struct Duo16<T> { // generic type
    var a: T
    var b: T
    func swapped() -> Duo16<T> { Duo16(a: b, b: a) }
}
protocol Bin16 {
    associatedtype Load
    var load: Load { get }
}
struct IntBin16: Bin16 { var load: Int }
func sameLoad16<A: Bin16, B: Bin16>(_ x: A, _ y: B) -> Bool where A.Load == B.Load, A.Load == Int {
    return x.load == y.load
}
func measure16(_ x: some Sized16) -> Int { x.size } // 5.7 some-parameter
func makeSized16() -> some Sized16 { Chip16(size: 3) } // opaque result type
protocol Clone16 { func twin() -> Self } // Self requirement
struct Pin16: Clone16 {
    var n: Int
    func twin() -> Pin16 { Pin16(n: n) }
}
func doubleTwin16<T: Clone16>(_ x: T) -> T { x.twin().twin() }
func s16() {
    let d = Duo16(a: 1, b: 2).swapped()
    let s = Duo16(a: "x", b: "y").swapped()
    check("gen1", d.a == 2 && d.b == 1 && s.b == "x")
    check("gen2", sameLoad16(IntBin16(load: 4), IntBin16(load: 4)))
    check("gen3", measure16(Chip16(size: 6)) == 6 && makeSized16().size == 3)
    check("gen4", doubleTwin16(Pin16(n: 5)).n == 5)
}

// ===== SECTION 17: error handling =====
enum VendErr17: Error {
    case empty
    case limit(Int)
}
struct Snack17 {
    let bars: Int
    init(bars: Int) throws { // throwing initializer
        if bars < 0 { throw VendErr17.empty }
        self.bars = bars
    }
}
func vend17(_ n: Int) throws -> Int {
    if n == 0 { throw VendErr17.empty }
    if n > 5 { throw VendErr17.limit(n) }
    return n * 2
}
func firstVend17(_ xs: [Int], _ pick: (Int) throws -> Int) rethrows -> Int {
    for x in xs { return try pick(x) }
    return 0
}
func s17() {
    var trace = ""
    do {
        _ = try vend17(0)
        trace = "no"
    } catch VendErr17.empty { // pattern-matched catch
        trace = "empty"
    } catch let VendErr17.limit(k) {
        trace = "limit\(k)"
    } catch {
        trace = "other"
    }
    check("err1", trace == "empty")
    do { _ = try vend17(9) } catch let VendErr17.limit(k) { trace = "limit\(k)" } catch { trace = "?" }
    check("err2", trace == "limit9")
    var kind = ""
    do { throw VendErr17.empty } catch { kind = error is VendErr17 ? "v" : "?" } // implicit binding
    check("err3", kind == "v")
    let none = try? vend17(0)
    let some = try? vend17(2)
    check("err4", none == nil && some == 4 && (try! vend17(1)) == 2) // try? and try!
    check("err5", (firstVend17([4]) { $0 + 1 }) == 5) // rethrows: no try with a non-throwing closure
    var re = ""
    do { _ = try firstVend17([0], vend17) } catch { re = "thrown" }
    check("err6", re == "thrown")
    let bad = try? Snack17(bars: -1)
    let good = try? Snack17(bars: 2)
    check("err7", bad == nil && good?.bars == 2)
}

// ===== SECTION 18: defer =====
enum Halt18: Error { case now }
func layered18() -> String {
    var log = ""
    func work() {
        defer { log += "1" } // registered first, runs last
        defer { log += "2" }
        log += "b"
    }
    work()
    return log // LIFO: b21
}
func earlyOut18(_ n: Int) -> String {
    var log = ""
    func run() {
        defer { log += "d" }
        if n > 0 {
            log += "e"
            return // defer still runs
        }
        log += "f"
    }
    run()
    return log
}
func thrownPath18() -> String {
    var log = ""
    func risky() throws {
        defer { log += "d" }
        log += "t"
        throw Halt18.now
    }
    do { try risky() } catch { log += "c" }
    return log // defer runs on the thrown path, before the catch
}
func s18() {
    check("dfr1", layered18() == "b21")
    check("dfr2", earlyOut18(1) == "ed" && earlyOut18(0) == "fd")
    check("dfr3", thrownPath18() == "tdc")
}

// ===== SECTION 19: pattern matching =====
enum Load19 {
    case idle
    case busy(Int)
}
func classify19(_ p: (Int, Int)) -> String {
    switch p {
    case (0, 0): return "o"
    case (let x, 0), (0, let x): return "a\(x)" // compound case with bindings
    case (1...3, 1...3): return "b" // ranges inside a tuple pattern
    case let (x, y) where x == y: return "d\(x)" // where guard
    case (_, let y): return "r\(y)"
    }
}
func bucket19(_ n: Int) -> String {
    switch n {
    case ..<0: return "neg" // one-sided ranges as patterns
    case 0: return "zero"
    case 1...9: return "small"
    case 10...: return "big"
    default: return "?"
    }
}
func level19(_ l: Load19) -> Int {
    guard case let .busy(n) = l else { return 0 }
    return n
}
func s19() {
    check("pat1", classify19((0, 0)) == "o" && classify19((7, 0)) == "a7" && classify19((0, 9)) == "a9")
    check("pat2", classify19((2, 3)) == "b" && classify19((5, 5)) == "d5" && classify19((4, 9)) == "r9")
    check("pat3", bucket19(-2) == "neg" && bucket19(0) == "zero" && bucket19(4) == "small" && bucket19(99) == "big")
    var order = ""
    switch order.count + 1 {
    case 1:
        order += "a"
        fallthrough
    case 2:
        order += "b"
    default:
        order += "c"
    }
    check("pat4", order == "ab")
    var got = 0
    if case .busy(let n) = Load19.busy(3) { got = n }
    check("pat5", got == 3 && level19(.busy(7)) == 7 && level19(.idle) == 0)
    var busySum = 0
    for case let .busy(n) in [Load19.idle, .busy(2), .busy(4)] { busySum += n }
    check("pat6", busySum == 6)
}

// ===== SECTION 20: control flow =====
func s20() {
    var hits = 0
    outer: for i in 0..<3 {
        for j in 0..<3 {
            if j == 1 { continue outer } // labeled continue
            if i == 2 { break outer } // labeled break
            hits += 1
        }
    }
    check("flo1", hits == 2)
    var evens = ""
    for i in 0..<7 where i % 2 == 0 { evens += "\(i)" } // for-in where clause
    check("flo2", evens == "0246")
    var r = 0
    repeat { r += 1 } while r < 3
    check("flo3", r == 3)
    var vSum = 0
    var kLen = 0
    for (k, v) in ["a": 1, "bb": 2] { // dictionary order is unspecified: order-independent sums
        vSum += v
        kLen += k.count
    }
    check("flo4", vSum == 3 && kLen == 3)
    var wl = 0
    spin: while true {
        wl += 1
        if wl == 2 { break spin } // labeled while
    }
    check("flo5", wl == 2)
}

// ===== SECTION 21: operators =====
precedencegroup PowerPrecedence21 {
    higherThan: MultiplicationPrecedence
    associativity: right
}
infix operator ***: PowerPrecedence21
func *** (b: Int, e: Int) -> Int {
    var r = 1
    for _ in 0..<e { r *= b }
    return r
}
prefix operator ~~~
prefix func ~~~ (n: Int) -> Int { 0 - n }
struct Vec21 {
    var x: Int
    var y: Int
    static func + (l: Vec21, r: Vec21) -> Vec21 { Vec21(x: l.x + r.x, y: l.y + r.y) }
    static func == (l: Vec21, r: Vec21) -> Bool { l.x == r.x && l.y == r.y } // no Equatable needed
    static func += (l: inout Vec21, r: Vec21) { l = l + r }
    static prefix func - (v: Vec21) -> Vec21 { Vec21(x: 0 - v.x, y: 0 - v.y) }
}
func s21() {
    check("ops1", 2 *** 3 *** 2 == 512 && 2 * 2 *** 3 == 16) // right-assoc, binds tighter than *
    var v = Vec21(x: 1, y: 2)
    v += Vec21(x: 3, y: 4)
    check("ops2", ~~~5 == -5 && v == Vec21(x: 4, y: 6) && (-v).x == -4)
    check("ops3", (5 & 3) == 1 && (5 | 2) == 7 && (5 ^ 1) == 4 && ~0 == -1 && (1 << 4) == 16 && (32 >> 2) == 8)
    check("ops4", Int.max &+ 1 == Int.min && 3 &* 3 == 9 && 0 &- 1 == -1) // overflow operators
    check("ops5", 1...5 ~= 3 && !(1...5 ~= 9)) // pattern-match operator as an expression
}

// ===== SECTION 22: declarations misc =====
public struct Badge22 {
    public let id: Int
    public init(id: Int) { self.id = id }
}
internal func plain22() -> Int { 2 }
fileprivate func local22() -> Int { 4 }
private func hidden22() -> Int { 3 } // same-file: all callable here
struct Outer22 {
    private var secret = 5
    fileprivate var shared = 6
    func reveal() -> Int { secret + shared }
    struct Inner { let tag = "in" } // nested types
    enum Mode { case fast, slow }
}
typealias Words22 = [String]
typealias Duo22<T> = (T, T) // generic typealias
func fetch22() async -> Int { 41 } // async syntax: defined, never awaited (no executor here)
func fetchList22() async throws -> [Int] { [1] }
actor Till22 {
    var total = 0
    func add(_ n: Int) -> Int {
        total += n
        return total
    }
}
func s22() {
    check("mis1", Badge22(id: 9).id == 9)
    check("mis2", plain22() + local22() + hidden22() == 9 && Outer22().reveal() == 11)
    var m = Outer22.Mode.fast
    m = .slow
    check("mis3", Outer22.Inner().tag == "in" && m == .slow)
    let ws: Words22 = ["a", "b"]
    let d: Duo22<Int> = (1, 2)
    check("mis4", ws.count == 2 && d.0 + d.1 == 3)
    var semi = 1; semi += 2 // statements joined by a semicolon
    check("mis5", semi == 3)
    let handle: () async -> Int = fetch22 // the bindings type-check the async/actor decls
    _ = handle
    _ = Till22()
    check("mis6", true) // async funcs + actor are define-only in this harness
}

// ===== END SECTIONS =====

func main() {
    s01() // SECTION-CALL 01
    s02() // SECTION-CALL 02
    s03() // SECTION-CALL 03
    s04() // SECTION-CALL 04
    s05() // SECTION-CALL 05
    s06() // SECTION-CALL 06
    s07() // SECTION-CALL 07
    s08() // SECTION-CALL 08
    s09() // SECTION-CALL 09
    s10() // SECTION-CALL 10
    s11() // SECTION-CALL 11
    s12() // SECTION-CALL 12
    s13() // SECTION-CALL 13
    s14() // SECTION-CALL 14
    s15() // SECTION-CALL 15
    s16() // SECTION-CALL 16
    s17() // SECTION-CALL 17
    s18() // SECTION-CALL 18
    s19() // SECTION-CALL 19
    s20() // SECTION-CALL 20
    s21() // SECTION-CALL 21
    s22() // SECTION-CALL 22
    print("full: \(checks) checks, \(fails) failures")
}

main()
exit(fails)
