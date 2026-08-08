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
    // INITIALIZER INHERITANCE (docs/todo.md 1.3). A class does NOT get a memberwise
    // initializer - that is structs - and a subclass declaring no initializer of its
    // own inherits its superclass's. `Cub13(legs: 3)` used to run an EMPTY
    // synthesised init and answer nil for every inherited property; every value here
    // is swiftc 6.1.2's own.
    check("inh6", Cub13(legs: 3).legs == 3 && Cub13(legs: 3).intro() == "beast/mew/3")
    // The subclass's OWN defaulted property is still laid down, and the two-level
    // chain inherits through the middle class.
    check("inh7", Kit13(legs: 2).spots == 5 && Kit13(legs: 2).legs == 2)
    // The superclass's OVERLOADS all arrive, under their own argument labels - and
    // an inherited init is picked BY DECLARED TYPE, which the shared rtIsType could
    // not do once a Double became a box, so every Double overload was rejected and
    // the dispatcher fell back to the first entry.
    check("inh8", Pup13().tag == "z" && Pup13(pair: 1.5, plus: 3.5).tag == "p")
    // An extension's convenience initializer MERGES with the inherited set instead
    // of shadowing it - self.init(legs:) here used to reach itself and recurse.
    check("inh9", Cub13(triple: 4).legs == 12 && Cub13(legs: 7).legs == 7)
    // The CONTROLS, which pass either way and are not counted as coverage: a
    // subclass that declares its own designated init keeps it and super.init runs;
    // a struct's real memberwise initializer still exists.
    check("inh10", Dog13().legs == 4 && Snake13().legs == 0)
    check("inh11", Den13(w: 5, h: 6).w == 5 && Den13().h == 3)
}
struct Den13 { var w = 2, h = 3 }
class Cub13: Beast13 { override func noise() -> String { "mew" } }
class Kit13: Cub13 { var spots: Int = 5 }
extension Cub13 { convenience init(triple t: Int) { self.init(legs: t * 3) } }
class Paw13 {
    var tag: String
    init() { self.tag = "z" }
    init(pair a: Double, plus b: Double) { self.tag = a + b == 5.0 ? "p" : "?" }
}
class Pup13: Paw13 {}

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
// A SECOND type nesting the SAME leaf names. The compiler half kept typeInfo under
// the BARE name, so both Inners shared one record and the second one's memberwise
// initializer landed on a key nothing looked up - `call of a non function value`
// where the interpreter and swiftc 6.1.2 both answer (docs/todo.md 1.7).
struct Peer22 {
    struct Inner { let tag = "peer" }
    enum Mode { case fast, slow }
}
struct Deep22 { struct Mid { struct Leaf { var v: Int } }
    struct Inner { var v: Int
        struct Inner { var v: Int } } }
struct Other22 { struct Mid { struct Leaf { var v: Int } } }
// Two same-named nested types with COLLIDING memberwise labels and, separately,
// with overloaded initializers: both pick their slot from the type's own record.
struct Lhs22 { struct Cell { var n: Int
        init(n: Int) { self.n = n }
        init(text t: String) { self.n = t.count } } }
struct Rhs22 { struct Cell { var n: Int
        init(dbl d: Int) { self.n = d * 2 } } }
// extension of a NESTED type, and a static that reads its own static unqualified.
extension Deep22.Mid.Leaf {
    func twice() -> Int { v * 2 }
    static var origin = 9
    static func base() -> Int { origin + 1 }
    subscript(k: Int) -> Int { v + k }
}
enum Wrap22 { enum Mode: Int { case fast = 7, slow = 8 } }
extension Wrap22.Mode { func bumped() -> Int { self.rawValue + 1 } }
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
    // nested types: two outers with the same leaf names, three levels, and the same
    // leaf name at two depths
    check("mis7", Outer22.Inner().tag == "in" && Peer22.Inner().tag == "peer")
    var pm = Peer22.Mode.fast
    pm = .slow
    check("mis8", pm == .slow && Outer22.Mode.fast != Outer22.Mode.slow)
    check("mis9", Deep22.Mid.Leaf(v: 1).v == 1 && Other22.Mid.Leaf(v: 2).v == 2)
    check("mis10", Deep22.Inner(v: 3).v == 3 && Deep22.Inner.Inner(v: 4).v == 4)
    check("mis11", Lhs22.Cell(n: 5).n == 5 && Lhs22.Cell(text: "abc").n == 3
                    && Rhs22.Cell(dbl: 6).n == 12)
    check("mis12", Deep22.Mid.Leaf(v: 5).twice() == 10 && Deep22.Mid.Leaf.base() == 10
                    && Deep22.Mid.Leaf(v: 5)[3] == 8)
    check("mis13", Wrap22.Mode.slow.rawValue == 8 && Wrap22.Mode.fast.bumped() == 8)
}

// ===== SECTION 23: value description and value-type equality =====
// The ratchet for the print/String(describing:) rendering and for == over the
// three VALUE types. Every expected string here was verified against swift
// 6.1.2 on this machine (the section runs green under `swift <file>`), except
// where a comment says otherwise.
struct Point23 { var x: Int; var y: String }
struct Money23: CustomStringConvertible {
    var cents: Int
    var description: String { "$\(cents)" }
}
enum Shape23 { case dot; case line(Int, String); case at(n: Int) }
enum Tag23: String { case red = "RED" }
func s23() {
    let nothing: Int? = nil
    check("dsc1", "\(nothing)" == "nil")
    check("dsc2", "\([1, 2, 3])" == "[1, 2, 3]")
    check("dsc3", "\([[1, 2], [3]])" == "[[1, 2], [3]]")
    let empty: [String] = []
    check("dsc4", "\(empty)" == "[]")
    check("dsc5", "\(["k": 1])" == "[\"k\": 1]")
    let emptyD: [String: Int] = [:]
    check("dsc6", "\(emptyD)" == "[:]")
    // A String nested in a collection is QUOTED and escaped; a top-level one is not.
    check("dsc7", "\(["s"])" == "[\"s\"]")
    check("dsc8", "\(["a\nb"])" == "[\"a\\nb\"]")
    check("dsc9", "\((1, "two"))" == "(1, \"two\")")
    let lt = (x: 1, y: "s")
    check("dsc10", "\(lt)" == "(x: 1, y: \"s\")")
    check("dsc11", "\(Point23(x: 1, y: "hi"))" == "Point23(x: 1, y: \"hi\")")
    // CustomStringConvertible wins over the synthesized rendering, nested too.
    check("dsc12", "\(Money23(cents: 7))" == "$7")
    check("dsc13", "\([Money23(cents: 7)])" == "[$7]")
    check("dsc14", "\(Shape23.dot)" == "dot")
    check("dsc15", "\(Shape23.line(2, "x"))" == "line(2, \"x\")")
    check("dsc16", "\(Shape23.at(n: 5))" == "at(n: 5)")
    check("dsc17", "\(Tag23.red)" == "red") // a raw-value case prints its NAME
    check("dsc18", "\(true) \(3)" == "true 3")
    check("dsc19", "\("plain")" == "plain")
    // Array, Dictionary and tuple are VALUE types: == compares element by element.
    check("eqv1", [1, 2] == [1, 2] && [1, 2] != [1, 3])
    check("eqv2", [[1], [2]] == [[1], [2]])
    check("eqv3", ["a": 1] == ["a": 1] && ["a": 1] != ["a": 2])
    check("eqv4", (1, "a") == (1, "a") && (1, "a") != (2, "a"))
    check("eqv5", empty == [])
    // print's separator: and terminator: labels (they used to be dropped and
    // printed as two extra positional arguments). Asserted through stdout, which
    // the matrix compares byte for byte between the two engines and --cross
    // between the two halves. Real Swift writes "1-2-3\nt!\n" here.
    print(1, 2, 3, separator: "-")
    print("t", terminator: "!\n")
    // NOT asserted, and deliberately: a struct nested in a collection is
    // MODULE-QUALIFIED in real Swift ("[m.Point23(x: 1, ...)]") because the
    // nested form is debugDescription; this runtime has no module concept and
    // prints the bare name in both halves. Same reason a class instance prints
    // its bare type name, and a non-nil Optional prints the wrapped value where
    // Swift prints Optional(3) - there is no Optional box in this value model.
}

// ===== SECTION 24: sized integers =====
// Swift's Int is 64 bit and Int8 ... UInt64 are exactly what they say. Every
// assertion below was run through the real `swift` 6.1.2 first and matches its
// output; the file as a whole is a valid Swift program, so it can be re-checked
// with `swift tests/swift-test-full.swift` at any time.
//
// NOT asserted, and deliberately: real Swift TRAPS on signed overflow, so
// `Int.max + 1` is a runtime crash there and wraps here (see the divergence note
// in abnf/jsrtswift.go). Only the &+ / &- / &* forms, which wrap in real Swift
// too, are asserted.
func s24() {
    // Int is 64 bit on every platform this targets.
    check("int1", Int.max == 9223372036854775807 && Int.min == -9223372036854775808)
    check("int2", Int64.max == 9223372036854775807 && Int64.min == -9223372036854775808)
    check("int3", UInt64.max == 18446744073709551615 && UInt64.min == 0)
    check("int4", Int32.max == 2147483647 && Int32.min == -2147483648 && UInt32.max == 4294967295)
    check("int5", Int16.max == 32767 && Int16.min == -32768 && UInt16.max == 65535)
    check("int6", Int8.max == 127 && Int8.min == -128 && UInt8.max == 255)
    // A shift that used to fall off a 32 bit int.
    check("int7", 1 << 62 == 4611686018427387904 && 1 << 63 == Int.min)
    // The wrapping operators, at 64 bits and at 8.
    check("int8", Int.max &+ 1 == Int.min && Int.min &- 1 == Int.max && Int.max &* 2 == -2)
    check("int9", UInt8(200) &+ UInt8(100) == 44 && Int8(100) &* 2 == -56)
    // Truncating division; the remainder takes the DIVIDEND's sign.
    check("int10", -7 / 2 == -3 && 7 / -2 == -3 && -7 % 2 == -1 && 7 % -2 == 1)
    // Swift's smart shift: over-shift is 0 (sign fill for a signed >>), and a
    // NEGATIVE count shifts the other way.
    let neg = -2
    check("int11", 1 << 64 == 0 && Int.max >> 63 == 0 && (-1) >> 1 == -1)
    check("int12", 1 << neg == 0 && 256 >> neg == 1024)
    // Unsigned comparison and unsigned division above 2^63.
    check("int13", UInt64.max > 0 && UInt64.max / 3 == 6148914691236517205 && UInt64.max % 7 == 1)
    // ~ complements at the operand's own width.
    check("int14", ~0 == -1 && ~Int8(0) == -1 && ~UInt8(0) == 255)
    check("int15", Int8(-1) << 1 == -2 && Int8(-128) >> 1 == -64)
    // A declared width survives into the next operation.
    var u: UInt8 = 250
    u = u &+ 10
    var w: Int32 = 2147483647
    w = w &+ 1
    check("int16", u == 4 && w == -2147483648)
    // The literal forms all read exactly.
    check("int17", 0x7FFF_FFFF_FFFF_FFFF == Int.max && 0b1010 == 10 && 0o777 == 511)
    check("int18", 9223372036854775807 == Int.max && -9223372036854775808 == Int.min)
    // Conversions: truncating from a Double, wrapping between widths, failable
    // from a String.
    check("int19", Int(3.9) == 3 && Int(-3.9) == -3)
    check("int20", Int(truncatingIfNeeded: UInt64.max) == -1 && UInt(bitPattern: -1) == UInt64.max)
    check("int21", Int("17") == 17 && Int("abc") == nil && Int8("300") == nil && Int("-9223372036854775808") == Int.min)
    check("int22", Int.max.description == "9223372036854775807" && "\(UInt64.max)" == "18446744073709551615")
    check("int23", Int32(300) == 300 && Int8(truncatingIfNeeded: 300) == 44)
    // A 64 bit product that a double cannot hold exactly.
    check("int24", 1000000007 * 1000000007 == 1000000014000000049)
    // The integer members, on a box and on a plain number alike.
    check("int25", Int.max.bitWidth == 64 && Int8.max.bitWidth == 8 && UInt8.max.bitWidth == 8)
    check("int26", (-5).magnitude == 5 && Int8(-100).magnitude == 100 && 7.description == "7")
}

// ===== SECTION 25: floating point =====
// Swift's Double is a type of its own, not "an Int that happens to have a
// fraction": 7.0 / 2.0 is 3.5 where 7 / 2 is 3, and a whole Double still prints
// with its ".0". Every assertion below was run through the real `swift` 6.1.2
// first and matches its output; the file as a whole is a valid Swift program, so
// it can be re-checked with `swift tests/swift-test-full.swift` at any time.
//
// Float is a REAL binary32 as of the float-width work, so its own rows are
// asserted below (f32a...f32o) against swiftc 6.1.2. Real Swift REJECTS a mixed
// Int/Double or Float/Double expression unless a literal is involved, so only
// the literal forms appear - the note is spelled out in abnf/jsrtswift.go.
//
// STILL NOT ASSERTED, and the one Float row swiftc and this front end disagree
// on: `n / m == 1.0 / 3.0` with n, m declared Float is TRUE in swiftc, because
// the untyped literals take the Float type from the comparison itself. Recovering
// that needs a type checker, which this front end does not have; it is the same
// wall as Optional(3). One line of an 8,042-line probe.
struct Flo25 { var x: Double; var y: Int }

// The Float renderer and the Double one, reached through a function boundary so
// the grammar's constant folder cannot answer the rows below at compile time.
func s25f(_ v: Float) -> String { return "\(v)" }
func s25d(_ v: Double) -> String { return "\(v)" }
// Read out of arrays for the same reason. Written with Float(...) rather than a
// `[Float]` annotation because that is what it had to be before SECTION 29 gave
// an array annotation's ELEMENT type its adoption; it is left as it was, and
// SECTION 29 asserts the annotated spelling separately.
let f25 = [Float(1.0), Float(3.0), Float(0.1), Float(16777216.0), Float(16777217.0), Float(2.0)]
let z25 = [-14447655.0, 0.0]
func s25() {
    // A float literal is a Double, so / is real division.
    check("flo1", 7.0 / 2.0 == 3.5 && 1.0 / 4.0 == 0.25)
    check("flo2", 2.5 * 1.5 == 3.75 && 1.5 + 1.0 == 2.5 && 1.5 - 2.0 == -0.5)
    check("flo3", 7.0.truncatingRemainder(dividingBy: 2.0) == 1.0)
    // description / interpolation: a whole Double keeps its ".0".
    check("flo4", "\(1.0)" == "1.0" && "\(3.5)" == "3.5" && (1.0).description == "1.0")
    check("flo5", "\(1e15)" == "1000000000000000.0" && "\(1e16)" == "1e+16")
    // THE UPPER BOUNDARY IS 2^53, NOT 1e16. Both halves used to switch to
    // scientific at a decimal exponent of 16; SwiftDtoa switches above 2^53, and
    // the two rules part between 9007199254740992 and 1e16. Every literal below is
    // swift 6.1.2's own answer. flo5 above cannot see the difference (both rules
    // agree at 1e15 and 1e16), so this is the line with the discriminating power:
    // under the old rule flo5b's first term rendered 9007199254740994.0.
    check("flo5b", "\(9007199254740992.0)" == "9007199254740992.0" &&
                   "\(9007199254740994.0)" == "9.007199254740994e+15")
    check("flo5c", "\(9999999999999000.0)" == "9.999999999999e+15" &&
                   "\(1234567890123456.0)" == "1234567890123456.0" &&
                   "\(-9007199254740994.0)" == "-9.007199254740994e+15")
    check("flo6", "\(1e-4)" == "0.0001" && "\(1e-5)" == "1e-05")
    check("flo7", "\(0.1 + 0.2)" == "0.30000000000000004" && "\(1.0 / 3.0)" == "0.3333333333333333")
    // The infinities and the NaN - none of which existed before the box.
    let inf = 1.0 / 0.0
    let nan = 0.0 / 0.0
    check("flo8", inf == Double.infinity && "\(inf)" == "inf" && "\(-inf)" == "-inf")
    check("flo9", nan.isNaN && !(nan == nan) && "\(nan)" == "nan")
    check("flo10", inf.isInfinite && !inf.isFinite && (1.0).isFinite && (0.0).isZero)
    // -0.0 keeps its sign but equals +0.0.
    check("flo11", "\(-0.0)" == "-0.0" && -0.0 == 0.0)
    // Conversions both ways; Double(3) is 3.0, which is the whole point of a box.
    check("flo12", Double(3) == 3.0 && "\(Double(3))" == "3.0" && Double(3) / 2 == 1.5)
    check("flo13", Int(3.7) == 3 && Int(-3.7) == -3)
    check("flo14", Double("3.5") == 3.5 && Double("x") == nil && Double("3.5x") == nil)
    check("flo15", String(3.5) == "3.5" && String(3) == "3" && String(describing: 1.0) == "1.0")
    // An untyped integer literal takes the Double type of the other operand.
    check("flo16", 1 + 2.0 == 3.0 && 2.0 * 3 == 6.0 && 1.0 == 1)
    // An annotated declaration is a Double even from an integer literal.
    let d: Double = 3
    check("flo17", d / 2 == 1.5 && "\(d)" == "3.0")
    // Ordering, and a NaN which compares false every way.
    check("flo18", 1.5 < 2.0 && 2.0 <= 2.0 && 3.0 > 2.5 && !(nan < 1.0) && !(nan >= 1.0))
    // The stdlib helpers this subset carries.
    check("flo19", abs(-1.5) == 1.5 && max(1.5, 2.0) == 2.0 && min(1.5, 2.0) == 1.5)
    check("flo20", (2.5).squareRoot() == 1.5811388300841898 && (2.0).rounded() == 2.0)
    check("flo21", (-2.5).rounded() == -3.0 && (2.5).rounded() == 3.0 && (1.5).magnitude == 1.5)
    check("flo22", Double.pi == 3.141592653589793 && Double.zero == 0.0)
    // A Double inside a collection prints and compares as a Double.
    check("flo23", "\([1.5, 2.0])" == "[1.5, 2.0]" && [1.0, 2.0] == [1.0, 2.0])
    let q = "\""
    check("flo24", "\(["a": 1.5])" == "[" + q + "a" + q + ": 1.5]")
    // Array + Array is CONCATENATION, where the untyped + gave the text "1,23".
    check("flo25", [1, 2] + [3] == [1, 2, 3] && "\([1, 2] + [3])" == "[1, 2, 3]")
    // A Double accumulator stays a Double across a loop.
    var acc = 0.0
    for _ in 1...3 { acc += 1.5 }
    check("flo26", acc == 4.5 && "\(acc)" == "4.5")
    // A struct field declared Double.
    check("flo27", "\(Flo25(x: 1.5, y: 2))" == "Flo25(x: 1.5, y: 2)")
    // Values both widths hold exactly agree whichever type they have.
    check("flo28", Float(1.5) + 1.5 == 3.0 && "\(Float(2.5))" == "2.5")
    let f: Float = 3
    check("flo29", f / 2 == 1.5 && "\(f)" == "3.0")
    // A BOXED value as a Dictionary key: a Double and a sized integer are both
    // objects in this value model, so a key had to be normalised to be found again.
    var dm = [1.5: "h", 2.5: "j"]
    dm[3.5] = "k"
    check("flo30", dm[1.5] == "h" && dm[2.5] == "j" && dm[3.5] == "k" && dm.count == 3)
    let im: [Int8: String] = [Int8(1): "x", Int8(2): "y"]
    check("flo31", im[Int8(1)] == "x" && im[Int8(2)] == "y" && im.count == 2)
    // 1.0 and 1 are the SAME key: equal values hash equal.
    let one = [1.0: "a"]
    check("flo32", one[1] == "a" && one[1.0] == "a")

    // ----- Float is a BINARY32, not a Double spelled differently -----
    // A Swift Float is an IEEE-754 single, so Float(1) / Float(3) is 0.33333334
    // and NOT the Double's 0.3333333333333333, and Float.description is
    // SwiftDtoa's rule read at 24 significant bits. swiftc 6.1.2 settles every
    // row below, and an 8,042-line probe over 40 float values - every operand
    // read out of an array - is byte-identical to it in all three engines.
    //
    // Swift FORBIDS mixed Float/Double and Float/Int arithmetic outright, so
    // "if either operand is a Float, both are" is the whole promotion rule and
    // none of java's JLS-5.6.2 wider-type machinery is needed.

    // The headline: eight significant digits, not sixteen.
    check("f32a", s25f(f25[0] / f25[1]) == "0.33333334")
    // ...and it is a DIFFERENT NUMBER. Double(f) widens a Float back out.
    check("f32b", Double(f25[0] / f25[1]) != 1.0 / 3.0)
    // 16777217 is the first integer a binary32 cannot hold; the value itself
    // rounds down to 16777216.
    check("f32c", s25f(f25[4]) == "16777216.0" && f25[4] == f25[3])
    // 0.1 as a Float is a different number from 0.1, and prints as the shortest
    // decimal that round-trips at 24 bits rather than at 53.
    check("f32d", s25f(f25[2]) == "0.1" && Double(f25[2]) == 0.10000000149011612)
    // An untyped literal on the other side becomes a Float too, so adding 1 at
    // the precision boundary does not move the value.
    check("f32e", s25f(f25[3] + 1) == "16777216.0")
    // ...and the same promotion decides the RELATIONS. (swiftc warns that the
    // literal is not exactly representable, which is exactly the point.)
    check("f32f", !(16777217 < f25[3]) && 16777217 <= f25[3])
    // THE PLAIN/SCIENTIFIC BOUNDARY MOVES WITH THE WIDTH: SwiftDtoa goes
    // scientific above 2^24 for a Float where it goes above 2^53 for a Double,
    // so 16777216.0 is the last plain Float and 33554432.0 is not.
    check("f32g", s25f(f25[3]) == "16777216.0" && s25f(f25[3] * 2) == "3.3554432e+07")
    // Float.pi is pi rounded TOWARD ZERO (FloatingPoint.pi), so it is neither
    // Double.pi nor Float(Double.pi), which would be 3.1415927.
    check("f32h", s25f(Float.pi) == "3.1415925" && Double(Float.pi) != Double.pi)
    // A declared Float type retypes an integer literal, at the 32 bit width.
    let g25: Float = 1
    check("f32i", s25f(g25 / 3) == "0.33333334")
    // The extremes. No two-significant-digit minimum: 1e-45, not java's 1.4E-45.
    check("f32j", s25f(Float(1e20)) == "1e+20" && s25f(Float(1e-45)) == "1e-45"
                  && s25f(Float(3.4028235e38)) == "3.4028235e+38")
    // Unary minus and both signed zeros survive the width.
    check("f32k", s25f(-f25[2]) == "-0.1" && s25f(f25[0] * Float(0.0)) == "0.0"
                  && s25f(Float(0.0) / Float(-1.0)) == "-0.0")
    // THE PRE-EXISTING SIGNED-ZERO DEFECT this work turned up 336 times, in the
    // INTERPRETER half only and for a plain Double: an integral-valued operand
    // times a zero lost its sign (0.0 where swiftc, IEEE-754 and both compiled
    // halves say -0.0). Reproduced at 4f2e6e4 in two lines of swift.
    check("f32l", s25d(z25[1] * z25[0]) == "-0.0" && s25d(z25[1] / z25[0]) == "-0.0")
    // A Float method answers a Float; .rounded() and .magnitude keep a signed
    // zero, which the interpreter half also lost for a plain Double.
    check("f32m", s25f(f25[5].squareRoot()) == "1.4142135"
                  && s25f(Float(-0.1).rounded()) == "-0.0"
                  && s25f(Float(-0.0).magnitude) == "0.0")
    // Float's own statics are Floats, and the failable initializer is Double's.
    check("f32n", s25f(Float.infinity) == "inf" && Float.nan.isNaN && s25f(Float.zero) == "0.0")
    check("f32o", Float("0.25") == Float(0.25) && Float("x") == nil)
}

// ===== SECTION 26: parameter packs =====
// SE-0393 variadic generics: `each T` in a generic parameter list, `repeat each T`
// as a parameter and a return type, the `repeat` expansion EXPRESSION (whose value
// is a tuple), `each p` inside it, and pack iteration with `for x in repeat each p`.
// Verified against swift 6.1.2: this section prints "full: 5 checks, 0 failures".
func packScale<each T: Numeric>(args arg: repeat each T) -> (repeat each T) {
    (repeat 2 * each arg + 3)
}
func packCount<each T>(_ v: repeat each T) -> Int {
    var n = 0
    for _ in repeat each v { n += 1 }
    return n
}
func packJoin<each T>(_ v: repeat each T) -> String {
    var s = ""
    for x in repeat each v { s += "\(x)|" }
    return s
}
func packDesc<each T>(_ v: repeat each T) -> String {
    return "\((repeat each v))"
}
func s26() {
    let r = packScale(args: 12, 12.3)
    check("pk1", r.0 == 27)
    check("pk2", r.1 == 27.6)
    check("pk3", packCount(1, 2, 3) == 3 && packCount("a") == 1)
    check("pk4", packJoin(1, "b", true) == "1|b|true|")
    check("pk5", packDesc(9, 8) == "(9, 8)")
}

// ===== SECTION 27: collection type constructors =====
// `[T]()` / `[K: V]()` - a collection TYPE written as a literal and constructed
// empty, including the element type no expression can read: a function type.
// Verified against swift 6.1.2: this section prints "full: 4 checks, 0 failures".
func s27() {
    var xs = [Int]()
    check("ctl1", xs.count == 0)
    xs.append(4)
    xs.append(5)
    check("ctl2", xs.count == 2 && xs[0] == 4 && xs[1] == 5)
    var ds = [String: Int]()
    check("ctl3", ds.count == 0)
    ds["k"] = 7
    let fns = [() -> Int]()
    let afs = [() async throws -> ()]()
    check("ctl4", ds["k"] == 7 && fns.count == 0 && afs.count == 0)
}

// ===== SECTION 28: deep raw strings, line continuations, #if in a protocol =====
// Raw delimiters past three pounds, a multi-line line-continuation with TRAILING
// whitespace after the backslash, a multi-line literal nested in an interpolation
// of a multi-line literal, a #if between protocol requirements, and a closure body
// holding real statements (a nested func, guard, switch, do, for).
// Verified against swift 6.1.2: this section prints "full: 6 checks, 0 failures".
protocol Shape28 {
    func area() -> Int
#if DEBUG
    func debugName() -> String
#else
    func name() -> String
#endif
}
struct Sq28: Shape28 {
    let side: Int
    func area() -> Int { return side * side }
    func name() -> String { return "sq" }
}
func s28() {
    let deep = #####"raw \#(no) "quotes" #### here"#####
    check("str1", deep == "raw \\#(no) \"quotes\" #### here" && deep.count == 29)
    let joined = """
      one \   
      two
      """
    check("str2", joined == "one two")
    let nested = """
      outer \
      \("""
        inner \
        tail
        """) end
      """
    check("str3", nested == "outer inner tail end")
    let s: Shape28 = Sq28(side: 3)
    check("pif1", s.area() == 9 && s.name() == "sq")
    let classify = { (n: Int) -> String in
        func twice(_ x: Int) -> Int { return x * 2 }
        guard n > 0 else { return "neg" }
        switch twice(n) {
        case 2: return "two"
        default: return "many"
        }
    }
    check("lam1", classify(1) == "two" && classify(5) == "many" && classify(-1) == "neg")
    let counted = { () -> Int in
        var t = 0
        do { t += 3 }
        for i in 1...3 { t += i }
        return t
    }
    check("lam2", counted() == 9)
}

// ===== SECTION 29: a literal ADOPTS its declared type (docs/todo.md 1.1) =====
// An untyped literal takes the FLOATNESS and the WIDTH of the type it is written
// at. Only ONE site did that before this section existed - a `let`/`var`
// annotation - and the four here did not: a PARAMETER, a RETURN type, a STORED
// PROPERTY, and the ELEMENT type of an array annotation.
//
// It reads like a Float story and it is not. The same missing adoption gave
// DOUBLE the wrong answer, because an Int literal that stays an Int
// INTEGER-DIVIDES: every `29d*` row below answered 1 where swiftc 6.1.2 says 1.5,
// in BOTH halves, which is why --cross was blind to it for three float rounds.
// Every value here is swift 6.1.2's own output.
//
// THE WRITE SITES ARE CLOSED TOO, and their rows are the `29s*` block at the end:
// `var d: Double = 0; d = 3`, `t.a = 3` on a stored property, and `ar[0] = 3` on an
// element all used to answer 1, because only the INITIAL value ever adopted. No
// var-type table was needed: this value model carries floatness and integer width
// ON THE VALUE, and Swift is statically typed, so a write adopts the type of what
// the slot already holds. The one shape that cannot see is a slot still holding
// nil - a `var d: Double` with no initializer - and 29s7 pins that as the gap.
//
// The three sites this paragraph used to list as STILL NOT ADOPTED are closed:
// the dictionary annotation's value type and the tuple type's element types went
// in with the structural annotation, and the function-typed slot
// (`let f: (Double) -> Double`) is SECTION 30. What is left is one nested case,
// asserted there with its own reason.
func a29d(_ x: Double) -> Double { return x / 2 }
func a29f(_ x: Float) -> Float { return x / 2 }
func a29u(_ x: UInt8) -> UInt8 { return x &+ 10 }
func a29i(_ x: Int32) -> Int32 { return x &* x }
func a29lab(a x: Double, b y: Double = 3) -> Double { return x / 2 + y / 2 }
func a29io(_ x: inout Double) { x = x + 1 }
func r29d() -> Double { return 3 }
func r29f() -> Float { return 16777217 }
func r29u() -> UInt8 { return 250 }
func e29d() -> Double { 3 }
func t29() -> (n: Int, v: Int) { return (1, 2) }
// A local named like an annotated global, unannotated: it is an Int and stays one.
var sh29g: Double = 1
func sh29() -> Int { var sh29g = 3; sh29g = 4; return sh29g / 2 }
struct A29 { var a: Double; var b: Float; var c: UInt8; var s: String }
struct B29 { var a: Double = 3; var n: Int = 7 }
class C29 { var a: Double = 3
            func m(_ x: Double) -> Double { return x / 2 + a / 2 } }
// Read out of arrays so the grammar's constant folder cannot answer these rows.
let n29 = [3, 250, 65536, 16777217]
func s29() {
    // ----- the parameter -----
    check("29d1", a29d(3) == 1.5 && s25d(a29d(3)) == "1.5")
    check("29d2", a29lab(a: 3) == 3.0 && a29lab(a: 3, b: 5) == 4.0)
    // A value that is ALREADY a Double travels as it always did.
    check("29d3", a29d(Double(n29[0])) == 1.5)
    // The width half of the same rule, at three widths.
    check("29w1", a29u(250) == 4 && a29u(UInt8(n29[1])) == 4)
    check("29w2", a29i(65536) == 0 && a29i(Int32(n29[2])) == 0)
    check("29w3", s25f(a29f(3)) == "1.5" && s25f(a29f(16777217)) == "8388608.0")
    // inout keeps its write-back, and the declared type still adopts.
    var v29: Double = 3
    a29io(&v29)
    check("29d4", v29 / 2 == 2.0)

    // ----- the return type -----
    check("29r1", r29d() / 2 == 1.5 && s25d(r29d()) == "3.0")
    check("29r2", s25f(r29f()) == "16777216.0" && r29u() &+ 10 == 4)
    // A single-expression body returns through the same site.
    check("29r3", e29d() / 2 == 1.5)
    // A LABELLED tuple return type still names its elements - and `return (1, 2)`
    // used to parse as the identifier `return` applied to a tuple, so this whole
    // shape died with *unknown name: return* before FnBody grew RetBody.
    check("29r4", t29().n == 1 && t29().v == 2)

    // ----- the stored property -----
    let a = A29(a: 3, b: 16777217, c: 250, s: "x")
    check("29p1", a.a / 2 == 1.5 && s25f(a.b) == "16777216.0" && a.c &+ 10 == 4 && a.s == "x")
    // A default value adopts too, and an un-annotated one is untouched.
    check("29p2", B29().a / 2 == 1.5 && B29().n / 2 == 3 && B29(a: 5).a / 2 == 2.5)
    check("29p3", C29().a / 2 == 1.5 && C29().m(3) == 3.0)

    // ----- the array annotation's element type -----
    let ad: [Double] = [3, 4]
    let af: [Float] = [16777217, 1]
    let au: [UInt8] = [250, 1]
    check("29a1", ad[0] / 2 == 1.5 && ad[1] / 2 == 2.0 && s25d(ad[0]) == "3.0")
    check("29a2", s25f(af[0]) == "16777216.0" && au[0] &+ 10 == 4)
    // An UNANNOTATED array literal is still [Int], and an [Int] annotation is
    // still an integer division - the adoption must not float everything.
    let plain = [3, 4]
    let ai: [Int] = [7, 4]
    check("29a3", plain[0] / 2 == 1 && ai[0] / 2 == 3)
    // A NESTED array annotation walks one level per bracket pair.
    let nd: [[Double]] = [[3, 4]]
    check("29a4", nd[0][0] / 2 == 1.5 && nd[0][1] / 2 == 2.0)

    // ----- the WRITE sites (docs/todo.md 1.1) -----
    // Only the INITIAL value used to adopt: every row below answered 1.
    var wd: Double = 0
    wd = 3
    check("29s1", wd / 2 == 1.5 && s25d(wd) == "3.0")
    var wu: UInt8 = 0
    wu = 250
    var wf: Float = 0
    wf = 16777217
    check("29s2", wu &+ 10 == 4 && s25f(wf) == "16777216.0")
    // A stored property of a struct value and of a class instance.
    var wa = A29(a: 0, b: 0, c: 0, s: "")
    wa.a = 3
    wa.c = 250
    check("29s3", wa.a / 2 == 1.5 && wa.c &+ 10 == 4)
    let wc = C29()
    wc.a = 5
    check("29s4", wc.a / 2 == 2.5 && wc.m(3) == 4.0)
    // An array ELEMENT, written plainly and compounded.
    var war: [Double] = [0, 0]
    war[0] = 3
    war[1] += 1
    check("29s5", war[0] / 2 == 1.5 && war[1] / 2 == 0.5)
    // A CONTROL: an unannotated Int must not float, or the adoption is too wide.
    var wi = n29[0]
    wi = 7
    check("29s6", wi / 2 == 3)
    // A slot that still holds NIL carries no type of its own, so the annotation of
    // a `var d: Double` with no initializer is the last resort - and it is read only
    // when the old value is nil, which is what keeps the inner unannotated `sh`
    // below an Int even though an outer `sh` is a Double.
    var wn: Double
    wn = 3
    check("29s7", wn / 2 == 1.5 && s25d(wn) == "3.0")
    check("29s8", sh29() == 2 && sh29g / 2 == 0.5)
}

// ===== SECTION 30: member overloads, function-typed slots, `case is` =====
// Four defects that share one root - the front end knowing a type it then threw
// away - and one that is the opposite, a type it never recorded.
//
//  * MEMBER OVERLOADS were picked at WALK time in the compiler half, by argument
//    LABELS and ARITY only. `MB(3.0)` ran `init(_ x: Int)` where the interpreter
//    and swiftc say Double; ordinary METHODS and SUBSCRIPTS had no dispatcher in
//    that half at all, so `m(1.5)`, `m("a")` and even `m(1, 2)` all ran
//    `m(_ x: Int)`. A live halves divergence --cross never reached.
//  * A FUNCTION-TYPED SLOT puts the type on the slot and the closure carries
//    nothing, so `let f: (Double) -> Double = { x in x/2 }; f(3)` integer-divided
//    and answered 1. A stored property holding a closure could not even be CALLED
//    ("unknown method 'fn'", all three engines).
//  * A NEW DICTIONARY KEY has no old value to adopt from, so `var d: [String:
//    Double] = [:]; d["b"] = 5` left an Int and `d["b"]! / 2` was 2.
//  * `case is T:` did not parse in EITHER half and took the whole switch down.
//  * `3 as Int8` was a no-op rather than a coercion: `(3 as Double) / 2` was 1,
//    `(250 as UInt8) &+ 10` was 260, and `(3 as Int8) is Int8` was false while
//    `is Int` was true - the reverse of swiftc 6.1.2 on every row.
//
// Every value below is swiftc 6.1.2's own output.
struct M30 {
    var s: String
    init(_ x: Int) { s = "I" }
    init(_ x: Double) { s = "D" }
    init(_ x: String) { s = "S" }
    init(_ x: Bool) { s = "B" }
    init(a: Int) { s = "lI" }
    init(a: Double) { s = "lD" }
}
struct O30 {
    func m(_ x: Int) -> String { return "I" }
    func m(_ x: Double) -> String { return "D" }
    func m(_ x: String) -> String { return "S" }
    func m(_ a: Int, _ b: Int) -> String { return "II" }
    subscript(i: Int) -> String { return "sI" }
    subscript(t: String) -> String { return "sS" }
}
struct F30 { var fn: (Double) -> Double }
enum E30 { case a, b }
func k30(_ v: Any) -> String {
    switch v {
    case is Double: return "D"
    case is String: return "S"
    case is Bool: return "B"
    case is E30: return "E"
    case is Int: return "I"
    default: return "?"
    }
}
let g30: (Double) -> Double = { x in x / 2 }
let g30b: (Double, Double) -> Double = { a, b in a / b }
func mk30() -> (Double) -> Double { return { x in x / 2 } }
// A PARAMETER whose declared type is structural adopts its argument too - the head
// of docs/todo.md 1.2. ap30's is a function type (the wrapper is built by the
// emitter), aq30's has no adoptable leaf anywhere in it, so it must still emit no
// adoption call at all and answer its argument untouched.
func ap30(_ f: (Double) -> Double) -> Double { return f(3) }
func aq30(_ xs: [String]) -> String { return xs[0] }

func s30() {
    // Every operand is read out of an array, so nothing here is constant-folded.
    let ii: [Int] = [1, 2]
    let dd: [Double] = [1.5]
    let ss: [String] = ["a"]
    let bb: [Bool] = [true]
    check("30a", M30(ii[0]).s == "I" && M30(dd[0]).s == "D")
    check("30b", M30(ss[0]).s == "S" && M30(bb[0]).s == "B")
    check("30c", M30(a: ii[0]).s == "lI" && M30(a: dd[0]).s == "lD")
    let o = O30()
    check("30d", o.m(ii[0]) == "I" && o.m(dd[0]) == "D" && o.m(ss[0]) == "S")
    check("30e", o.m(ii[0], ii[1]) == "II")
    check("30f", o[ii[0]] == "sI" && o[ss[0]] == "sS")

    // A function-typed slot: a let, a stored property and a RETURN type.
    check("30g", g30(3) == 1.5 && g30b(3, 2) == 1.5)
    check("30h", F30(fn: { x in x / 2 }).fn(3) == 1.5)
    check("30i", mk30()(3) == 1.5)
    // A function type NESTED in a container annotation - an array, a dictionary
    // value and a tuple element (docs/todo.md 1.2). It used to answer 1 and was
    // asserted at that value, because the compiler half hands a structural
    // annotation to js_swadoptdeep in LAYER 2 and layer 2 builds no closures. It
    // does not have to: the emitter knows every function-type leaf of the
    // annotation text at compile time, so it builds one MAKER closure per distinct
    // leaf and passes js_swadoptdeep a map from the leaf's canonical text to its
    // maker. swiftc 6.1.2 says 1.5 for all three.
    let arr30: [(Double) -> Double] = [{ x in x / 2 }]
    check("30j", arr30[0](3) == 1.5)
    let dic30: [String: (Double) -> Double] = ["h": { x in x / 2 }]
    check("30j2", dic30["h"]!(3) == 1.5)
    let tup30: ((Double) -> Double, Int) = ({ x in x / 2 }, 1)
    check("30j3", tup30.0(3) == 1.5 && tup30.1 == 1)
    // A parameter whose annotation is itself a function type adopts the same way,
    // and one with no adoptable leaf in it is left alone.
    check("30j4", ap30({ x in x / 2 }) == 1.5 && aq30(["x"]) == "x")

    // A NEW dictionary key adopts the declared element type.
    var d1: [String: Double] = [:]
    d1["b"] = 5
    var d2: [String: Double] = ["a": 1]
    d2["b"] = 5
    var d3: [Int: Double] = [:]
    d3[1] = 5
    var d4: [String: Int8] = [:]
    d4["q"] = 100
    check("30k", d1["b"]! / 2 == 2.5 && d2["b"]! / 2 == 2.5 && d2["a"]! / 2 == 0.5)
    check("30l", d3[1]! / 2 == 2.5 && d4["q"]! &+ 100 == -56)

    // `case is T:` - a type pattern, including an enum case value.
    let anys: [Any] = [3.0, "s", true, E30.a, 3]
    check("30m", k30(anys[0]) == "D" && k30(anys[1]) == "S" && k30(anys[2]) == "B")
    check("30n", k30(anys[3]) == "E" && k30(anys[4]) == "I")

    // `as` is a coercion.
    check("30o", (3 as Double) / 2 == 1.5 && (250 as UInt8) &+ 10 == 4)
    // Through Any, so the test is the RUNTIME one: swiftc folds a directly
    // spelled `(3 as Int8) is Int` and only warns.
    let a8: [Any] = [3 as Int8, 3 as Double]
    check("30p", (a8[0] is Int8) && !(a8[0] is Int) && (a8[1] is Double))
}

// ===== SECTION 31: structural PARAMETERS, collection method adoption, =====
// =====             inout overloads, inherited overload groups, statics =====
//
// docs/todo.md 1.2, all five reproduced against swiftc 6.1.2 and all five wrong
// in ALL THREE engines before this section, so --cross was blind to every one.
//
//  * A STRUCTURALLY TYPED PARAMETER did not adopt. `func f(_ xs: [Double])`
//    called `f([3])` bound a plain Int array and `xs[0] / 2` answered 1 where
//    swiftc says 1.5. makeParam's gate was swAdoptable, i.e. a plain type NAME,
//    and the comment above the ParamType rule claimed the opposite. The gate is
//    now swAdoptNeeded, which asks whether any identifier token in the annotation
//    text names an adopting type - so `[String]` still emits no call at all, and
//    js_swadoptdeep does not go on the per-call path of every annotated program
//    (the +10.3% charAt trap of a68e16d).
//  * ELEMENT ADOPTION THROUGH A METHOD CALL. `arr.append(3)` into a
//    `var arr: [Double]` stored an Int, while the index write `arr[1] = 3` had
//    adopted since the write-site work - makeAssign knows its target's NAME and a
//    method chain did not. The value model cannot answer it either (manual 7.15):
//    appending to an EMPTY [Double] has no sibling to take a type from.
//  * AN INOUT OVERLOAD was not selectable by type: the argument arrives as the
//    one-slot {__ref, v} write-back box, so js_swfits was asking whether a BOX is
//    an Int and both `ov(inout Int)` and `ov(inout String)` ran the first entry.
//  * A SUBCLASS OVERRIDING ONE OF SEVERAL inherited overloads shadowed the whole
//    group: the derived table's plain `m` slot is what the __class walk finds, so
//    the un-overridden siblings were unreachable.
//  * STATIC METHOD OVERLOADS were stored under a bare name and the last one won.
//
// Every value below is swiftc 6.1.2's own output.
func p31a(_ xs: [Double]) -> Double { return xs[0] / 2 }
func p31b(_ m: [String: Double]) -> Double { return m["k"]! / 2 }
func p31c(_ t: (Double, Int)) -> Double { return t.0 / 2 }
func p31d(_ x: Double?) -> Double { return x! / 2 }
func p31e(_ xs: [[Double]]) -> Double { return xs[0][0] / 2 }
func p31f(_ xs: [Int8]) -> Int8 { return xs[0] }

func ov31(_ x: inout Int) { x = x + 100 }
func ov31(_ x: inout String) { x = x + "!" }

class B31 {
    func m(_ a: Int) -> String { return "bI" }
    func m(_ a: String) -> String { return "bS" }
    func m(_ a: Double) -> String { return "bD" }
}
class D31: B31 {
    override func m(_ a: Int) -> String { return "dI" }
}
class E31: D31 {
    override func m(_ a: String) -> String { return "eS" }
}
class K31 {
    static func g(_ a: Int) -> String { return "kI" }
    static func g(_ a: String) -> String { return "kS" }
}
struct S31 {
    static func h(_ a: Int) -> String { return "sI" }
    static func h(_ a: String) -> String { return "sS" }
}

func s31() {
    // Every operand that is not the subject of a literal's adoption is read out of
    // an array, so the constant folder cannot answer these rows.
    let ii: [Int] = [3, 5]
    let ss: [String] = ["x"]
    check("31a", p31a([3]) == 1.5 && p31a([5]) == 2.5)
    check("31b", p31b(["k": 3]) == 1.5)
    check("31c", p31c((3, 1)) == 1.5)
    check("31d", p31d(3) == 1.5)
    check("31e", p31e([[3]]) == 1.5)
    check("31f", p31f([3]) == 3)

    var a31: [Double] = [1.0]
    a31.append(3)
    check("31g", a31[1] / 2 == 1.5)
    var b31: [Double] = []
    b31.append(5)
    check("31h", b31[0] / 2 == 2.5)
    var c31: [Int8] = []
    c31.append(100)
    check("31i", c31[0] == 100)
    // A collection with no adoptable element type is untouched.
    var d31: [String] = []
    d31.append(ss[0])
    check("31j", d31[0] == "x")
    // insert(_:at:) existed in NO engine and aborted with *unknown Array method*;
    // it is the same element-adoption site as append.
    var e31: [Double] = []
    e31.insert(7, at: 0)
    e31.insert(9, at: 0)
    e31.insert(11, at: 1)
    check("31q", e31[0] / 2 == 4.5 && e31[1] / 2 == 5.5 && e31[2] / 2 == 3.5 && e31.count == 3)
    var f31: [String] = ["x", "z"]
    f31.insert(ss[0], at: 1)
    check("31r", f31[0] == "x" && f31[1] == "x" && f31[2] == "z")

    var vi = ii[0]
    var vs = ss[0]
    ov31(&vi)
    ov31(&vs)
    check("31k", vi == 103 && vs == "x!")

    let d = D31()
    check("31l", d.m(ii[0]) == "dI")
    check("31m", d.m(ss[0]) == "bS" && d.m(1.5) == "bD")
    // Two levels: E31 overrides the String one, D31 the Int one, B31 keeps Double.
    let e = E31()
    check("31n", e.m(ii[0]) == "dI" && e.m(ss[0]) == "eS" && e.m(1.5) == "bD")

    check("31o", K31.g(ii[0]) == "kI" && K31.g(ss[0]) == "kS")
    check("31p", S31.h(ii[0]) == "sI" && S31.h(ss[0]) == "sS")
}

// ===== SECTION 32: extension overloads, and the FloatingPoint range statics =====
// An extension used to restart the overload counter at 0 in the COMPILER half, so
// its member overwrote the one the type declared: static funcs, instance methods
// and subscripts all lost the type's own, where an initializer did not. The
// interpreter's addMethod merged onto the same table and was always right, so every
// row here was a live halves divergence --cross never reached (docs/todo.md 1.4).
struct S32 {
    var v: Int = 0
    static func f32(_ a: String) -> String { return "sS" }
    func m32(_ a: String) -> String { return "mS" }
    subscript(_ a: String) -> String { return "xS" }
    init(s: String) { self.v = s.count }
}
extension S32 {
    static func f32(_ a: Int) -> String { return "sI" }
    func m32(_ a: Int) -> String { return "mI" }
    subscript(_ a: Int) -> String { return "xI" }
    init(i: Int) { self.v = i }
}
// The type already carries a DISPATCHER (two overloads); the extension adds a third.
struct T32 {
    func p32(_ a: Int) -> String { return "pI" }
    func p32(_ a: String) -> String { return "pS" }
    static func q32(_ a: Int) -> String { return "qI" }
    static func q32(_ a: String) -> String { return "qS" }
}
extension T32 {
    func p32(_ a: Double) -> String { return "pD" }
    static func q32(_ a: Double) -> String { return "qD" }
}
// Two separate extensions, each adding one candidate.
struct U32 { func r32(_ a: Int) -> String { return "rI" } }
extension U32 { func r32(_ a: String) -> String { return "rS" } }
extension U32 { func r32(_ a: Double) -> String { return "rD" } }
// An extension on a SUBCLASS re-opens the base's group rather than hiding it.
class B32 { func g32(_ a: Int) -> String { return "gI" } }
class D32: B32 {}
extension D32 { func g32(_ a: String) -> String { return "gS" } }
enum E32 {
    case one
    func w32(_ a: Int) -> String { return "wI" }
    static func z32(_ a: Int) -> String { return "zI" }
}
extension E32 {
    func w32(_ a: String) -> String { return "wS" }
    static func z32(_ a: String) -> String { return "zS" }
}
protocol P32 {}
extension P32 {
    func t32(_ a: Int) -> String { return "tI" }
    func t32(_ a: String) -> String { return "tS" }
}
struct V32: P32 {}
// A SECOND `extension Int` used to declare a second, empty table under the same
// name, and every member the first block added became unreachable - an outright
// abort in the compiler half where the interpreter printed the right answer.
extension Int { func aa32() -> String { return "aa\(self)" } }
extension Int { func bb32() -> String { return "bb\(self)" } }
// A static OPERATOR and a STATIC SUBSCRIPT are the two remaining shapes of the
// static overload surface. The static subscript is the one static reached WITH a
// receiver - sw_subget pushes the type descriptor at args[0] - so its dispatcher
// counts arguments from 1; counting from 0 made the arity test reject every
// candidate and the fallback run the first, and that was already wrong at base
// with no extension involved.
struct M32 {
    var v: Int
    static func + (a: M32, b: M32) -> M32 { return M32(v: a.v + b.v) }
    static subscript(_ i: Int) -> String { return "yI" }
    static subscript(_ s: String) -> String { return "yS" }
}
extension M32 {
    static func + (a: M32, b: Int) -> M32 { return M32(v: a.v + b * 10) }
    static subscript(_ f: Double) -> String { return "yD" }
}

func s32() {
    let ii: [Int] = [3]
    let ss: [String] = ["ab"]
    let dd: [Double] = [1.5]
    let s = S32(s: ss[0])
    check("32a", S32.f32(ss[0]) == "sS" && S32.f32(ii[0]) == "sI")
    check("32b", s.m32(ss[0]) == "mS" && s.m32(ii[0]) == "mI")
    check("32c", s[ss[0]] == "xS" && s[ii[0]] == "xI")
    check("32d", S32(s: ss[0]).v == 2 && S32(i: ii[0]).v == 3)
    let t = T32()
    check("32e", t.p32(ii[0]) == "pI" && t.p32(ss[0]) == "pS" && t.p32(dd[0]) == "pD")
    check("32f", T32.q32(ii[0]) == "qI" && T32.q32(ss[0]) == "qS" && T32.q32(dd[0]) == "qD")
    let u = U32()
    check("32g", u.r32(ii[0]) == "rI" && u.r32(ss[0]) == "rS" && u.r32(dd[0]) == "rD")
    let d = D32()
    check("32h", d.g32(ii[0]) == "gI" && d.g32(ss[0]) == "gS")
    let e = E32.one
    check("32i", e.w32(ii[0]) == "wI" && e.w32(ss[0]) == "wS")
    check("32j", E32.z32(ii[0]) == "zI" && E32.z32(ss[0]) == "zS")
    let v = V32()
    check("32k", v.t32(ii[0]) == "tI" && v.t32(ss[0]) == "tS")
    check("32l", ii[0].aa32() == "aa3" && ii[0].bb32() == "bb3")
    let ms: [M32] = [M32(v: 1), M32(v: 2)]
    check("32w", (ms[0] + ms[1]).v == 3 && (ms[0] + ii[0]).v == 31)
    check("32x", M32[ii[0]] == "yI" && M32[ss[0]] == "yS" && M32[dd[0]] == "yD")

    // The FloatingPoint range statics: all eight answered nil in all three engines
    // (docs/todo.md 1.10 named two). Float's are FLOATS, so they print at float
    // precision; the values are read back as text, which is what pins them.
    check("32m", "\(Double.greatestFiniteMagnitude)" == "1.7976931348623157e+308")
    check("32n", "\(Double.leastNonzeroMagnitude)" == "5e-324")
    check("32o", "\(Double.leastNormalMagnitude)" == "2.2250738585072014e-308")
    check("32p", "\(Double.ulpOfOne)" == "2.220446049250313e-16")
    check("32q", "\(Float.greatestFiniteMagnitude)" == "3.4028235e+38")
    check("32r", "\(Float.leastNonzeroMagnitude)" == "1e-45")
    check("32s", "\(Float.leastNormalMagnitude)" == "1.1754944e-38")
    check("32t", "\(Float.ulpOfOne)" == "1.1920929e-07")
    let two: [Double] = [2.0]
    let ft: [Float] = [2.0]
    check("32u", Double.greatestFiniteMagnitude * two[0] == Double.infinity
              && Double.leastNonzeroMagnitude / two[0] == 0.0
              && 1.0 + Double.ulpOfOne > 1.0)
    check("32v", Float.greatestFiniteMagnitude * ft[0] == Float.infinity
              && Float.leastNonzeroMagnitude / ft[0] == Float(0)
              && Float(1) + Float.ulpOfOne > Float(1))
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
    s23() // SECTION-CALL 23
    s24() // SECTION-CALL 24
    s25() // SECTION-CALL 25
    s26() // SECTION-CALL 26
    s27() // SECTION-CALL 27
    s28() // SECTION-CALL 28
    s29() // SECTION-CALL 29
    s30() // SECTION-CALL 30
    s31() // SECTION-CALL 31
    s32() // SECTION-CALL 32
    print("full: \(checks) checks, \(fails) failures")
}

main()
exit(fails)
