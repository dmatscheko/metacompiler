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
// map/filter/forEach) plus SECTION 81's free functions - so no CaseIterable/Comparable/Codable conformances, and
// Equatable only where a section tests the synthesized == (own protocols carry
// the generic constraints), and no key paths
// (\.p is the stdlib KeyPath type). Concurrency at RUNTIME:
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
    // INITIALIZER INHERITANCE. A class does NOT get a memberwise
    // initializer - that is structs - and a subclass declaring no initializer of its
    // own inherits its superclass's. Every value here is swiftc 6.1.2's own.
    check("inh6", Cub13(legs: 3).legs == 3 && Cub13(legs: 3).intro() == "beast/mew/3")
    // The subclass's OWN defaulted property is still laid down, and the two-level
    // chain inherits through the middle class.
    check("inh7", Kit13(legs: 2).spots == 5 && Kit13(legs: 2).legs == 2)
    // The superclass's OVERLOADS all arrive, under their own argument labels - and
    // an inherited init is picked BY DECLARED TYPE (a Double is a box).
    check("inh8", Pup13().tag == "z" && Pup13(pair: 1.5, plus: 3.5).tag == "p")
    // An extension's convenience initializer MERGES with the inherited set instead
    // of shadowing it - self.init(legs:) here must reach the inherited init.
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
// A SECOND type nesting the SAME leaf names: each Inner keeps its own record and
// its own memberwise initializer.
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
    // print's separator: and terminator: labels. Asserted through stdout, which
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
    // A shift past 32 bits.
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
// wall as Optional(3).
struct Flo25 { var x: Double; var y: Int }

// The Float renderer and the Double one, reached through a function boundary so
// the grammar's constant folder cannot answer the rows below at compile time.
func s25f(_ v: Float) -> String { return "\(v)" }
func s25d(_ v: Double) -> String { return "\(v)" }
// Read out of arrays for the same reason. Written with Float(...) rather than a
// `[Float]` annotation; SECTION 29 asserts the annotated spelling separately.
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
    // THE UPPER BOUNDARY IS 2^53, NOT 1e16: SwiftDtoa switches to scientific above
    // 2^53, not at a decimal exponent of 16, and the two rules part between
    // 9007199254740992 and 1e16. Every literal below is swift 6.1.2's own answer.
    // flo5 above cannot see the difference (both rules agree at 1e15 and 1e16);
    // this line can.
    check("flo5b", "\(9007199254740992.0)" == "9007199254740992.0" &&
                   "\(9007199254740994.0)" == "9.007199254740994e+15")
    check("flo5c", "\(9999999999999000.0)" == "9.999999999999e+15" &&
                   "\(1234567890123456.0)" == "1234567890123456.0" &&
                   "\(-9007199254740994.0)" == "-9.007199254740994e+15")
    check("flo6", "\(1e-4)" == "0.0001" && "\(1e-5)" == "1e-05")
    check("flo7", "\(0.1 + 0.2)" == "0.30000000000000004" && "\(1.0 / 3.0)" == "0.3333333333333333")
    // The infinities and the NaN.
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
    // Double(String) reads as strtod does: inf, infinity and nan in any case, a
    // hexadecimal numeral, and a decimal one past the range as inf or zero.
    let fs = ["Inf", "-infinity", "NaN", "0x1p4", "0X1.8p1", "1e400", "1e-400", "infin", "0x1p", "+-1"]
    var fd: [String] = []
    for t in fs {
        if let v = Double(t) {
            fd.append("\(v)")
        } else {
            fd.append("nil")
        }
    }
    check("dps1", fd.joined(separator: ",") == "inf,-inf,nan,16.0,3.0,inf,0.0,nil,nil,nil")
    check("dps2", Float(fs[0])! == Float.infinity && Float(fs[3])! == 16)
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
    // Array + Array is CONCATENATION, not the text "1,23".
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
    // objects in this value model, so a key has to be normalised to be found again.
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
    // row below.
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
    // An integral-valued operand times a zero keeps the XOR sign (-0.0, as swiftc
    // and IEEE-754 say) - the interpreter's tag engine multiplies such operands as
    // integers, which would lose it.
    check("f32l", s25d(z25[1] * z25[0]) == "-0.0" && s25d(z25[1] / z25[0]) == "-0.0")
    // A Float method answers a Float; .rounded() and .magnitude keep a signed
    // zero.
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

// ===== SECTION 29: a literal ADOPTS its declared type =====
// An untyped literal takes the FLOATNESS and the WIDTH of the type it is written
// at - not only at a `let`/`var` annotation but at a PARAMETER, a RETURN type, a
// STORED PROPERTY, and the ELEMENT type of an array annotation.
//
// An Int literal that stays an Int INTEGER-DIVIDES, so every `29d*` row below
// would answer 1 where swiftc 6.1.2 says 1.5. Every value here is swift 6.1.2's
// own output.
//
// THE WRITE SITES are the `29s*` block at the end: `var d: Double = 0; d = 3`,
// `t.a = 3` on a stored property, and `ar[0] = 3` on an element. This value model
// carries floatness and integer width ON THE VALUE, and Swift is statically typed,
// so a write adopts the type of what the slot already holds. A slot still holding
// nil - a `var d: Double` with no initializer - falls back to the annotation (29s7).
//
// The dictionary annotation's value type and the tuple type's element types are
// covered by the structural annotation, and the function-typed slot
// (`let f: (Double) -> Double`) is SECTION 30.
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
    // A value that is ALREADY a Double travels unchanged.
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
    // must not parse as the identifier `return` applied to a tuple (FnBody's RetBody).
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

    // ----- the WRITE sites -----
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
//  * MEMBER OVERLOADS are picked by argument TYPE, not by LABELS and ARITY only:
//    `MB(3.0)` runs `init(_ x: Double)`, and `m(1.5)`, `m("a")` and `m(1, 2)`
//    each run their own METHOD overload; SUBSCRIPTS likewise.
//  * A FUNCTION-TYPED SLOT puts the type on the slot and the closure carries
//    nothing, yet `let f: (Double) -> Double = { x in x/2 }; f(3)` is 1.5. A stored
//    property holding a closure is CALLABLE.
//  * A NEW DICTIONARY KEY has no old value to adopt from, yet `var d: [String:
//    Double] = [:]; d["b"] = 5` makes `d["b"]! / 2` 2.5.
//  * `case is T:` is a type pattern.
//  * `3 as Int8` is a coercion: `(3 as Double) / 2` is 1.5,
//    `(250 as UInt8) &+ 10` is 4, and `(3 as Int8) is Int8` is true.
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
// A PARAMETER whose declared type is structural adopts its argument too.
// ap30's is a function type (the wrapper is built by the
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
    // value and a tuple element. Layer 2 builds no closures, so the emitter builds
    // one MAKER closure per distinct function-type leaf and passes js_swadoptdeep
    // a map from the leaf's canonical text to its maker. swiftc 6.1.2 says 1.5 for
    // all three.
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
//  * A STRUCTURALLY TYPED PARAMETER adopts: `func f(_ xs: [Double])` called
//    `f([3])` binds [3.0], so `xs[0] / 2` is 1.5 (swAdoptNeeded; `[String]` still
//    emits no call at all).
//  * ELEMENT ADOPTION THROUGH A METHOD CALL: `arr.append(3)` into a
//    `var arr: [Double]` stores 3.0, like the index write `arr[1] = 3`. The value
//    model cannot answer it (manual 7.15): appending to an EMPTY [Double] has no
//    sibling to take a type from.
//  * AN INOUT OVERLOAD is selectable by type: the argument arrives as the
//    one-slot {__ref, v} write-back box, which is opened first.
//  * A SUBCLASS OVERRIDING ONE OF SEVERAL inherited overloads does not shadow the
//    un-overridden siblings.
//  * STATIC METHOD OVERLOADS are overloads, not a last-one-wins bare name.
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
    // insert(_:at:) is the same element-adoption site as append.
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
// An extension's overloads JOIN the ones the type declared (static funcs, instance
// methods and subscripts as well as initializers) instead of overwriting them.
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
// A SECOND `extension Int` shares the first block's member table.
extension Int { func aa32() -> String { return "aa\(self)" } }
extension Int { func bb32() -> String { return "bb\(self)" } }
// A static OPERATOR and a STATIC SUBSCRIPT are the two remaining shapes of the
// static overload surface. The static subscript is the one static reached WITH a
// receiver - sw_subget pushes the type descriptor at args[0] - so its dispatcher
// counts arguments from 1.
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

    // The FloatingPoint range statics. Float's are FLOATS, so they print at float
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

// ===== SECTION 33: protocol conformance, and dispatch inside an extension =====
// Three rules: `x is P` is TRUE for a conforming type; a call from inside a protocol
// extension binds STATICALLY unless the protocol declares the name as a
// REQUIREMENT; and an `extension Int { ... }` method must not hijack a user type's
// member of the same name. Every operand is read out of an array so the constant
// folder cannot decide any of it.
protocol P33A {
    func tag33() -> String
}
extension P33A {
    func hi33() -> String { return "hi:" + tag33() }
    func tag33() -> String { return "?" }
    func plain33() -> String { return "p" }
    func viaPlain33() -> String { return "v:" + plain33() }
    func viaSelf33() -> String { return "s:" + self.plain33() }
    var loud33: String { return tag33() + "!" }
    func viaLoud33() -> String { return "l:" + loud33 }
}
protocol P33B: P33A { }
protocol P33C { }
struct S33A: P33A { func tag33() -> String { return "A" } }
struct S33B: P33A { }
class C33A: P33A { func tag33() -> String { return "C" } }
class C33B: C33A { }
enum E33: String, P33B { case one = "1"
    func tag33() -> String { return "E" }
}
struct S33C { }
extension S33C: P33A { func tag33() -> String { return "X" }
    func plain33() -> String { return "own" }
    var loud33: String { return "OWN" }
}
extension Int: P33A {
    func tag33() -> String { return "I" }
    func mine33() -> String { return "int" }
}
struct S33D { func mine33() -> String { return "struct" } }
struct Boom33: Error, P33C { }
struct Bang33: Error { }

func thrower33(_ n: Int) throws -> Int {
    if n == 0 { throw Boom33() }
    throw Bang33()
}

func s33() {
    let vals: [Any] = [S33A(), S33B(), C33A(), C33B(), E33.one, S33C(), 7, "z"]
    var isP = ""
    for v in vals { isP += (v is P33A) ? "1" : "0" }
    check("33a", isP == "11111110")
    var isB = ""
    for v in vals { isB += (v is P33B) ? "1" : "0" }
    check("33b", isB == "00001000")
    var isC = ""
    for v in vals { isC += (v is P33C) ? "1" : "0" }
    check("33c", isC == "00000000")
    // as? keeps the value, and its member call still dispatches on the real type.
    var got = ""
    for v in vals { if let p = v as? P33A { got += p.tag33() } else { got += "-" } }
    check("33d", got == "A?CCEXI-")
    // case is P, the same test through the pattern matcher.
    var cs = ""
    for v in vals {
        switch v {
        case is P33B: cs += "B"
        case is P33A: cs += "A"
        default: cs += "-"
        }
    }
    check("33e", cs == "AAAABAA-")
    // A REQUIREMENT dispatches to the conforming type; a member the extension alone
    // introduces does not, even when the type declares its own.
    let ps: [P33A] = [S33A(), S33B(), C33A(), E33.one, S33C()]
    var his = ""
    for p in ps { his += p.hi33() + "|" }
    check("33f", his == "hi:A|hi:?|hi:C|hi:E|hi:X|")
    let sc = [S33C()]
    check("33g", sc[0].viaPlain33() == "v:p")
    check("33h", sc[0].viaSelf33() == "s:p")
    check("33i", sc[0].plain33() == "own")
    // The same rule for a COMPUTED property: viaLoud33 reads the extension's loud33,
    // whose tag33() IS a requirement and so reaches S33C's own.
    check("33q", sc[0].viaLoud33() == "l:X!" && sc[0].loud33 == "OWN")
    // A subclass conforms through its superclass, and inherits the default.
    let cb: [Any] = [C33B()]
    check("33j", (cb[0] as! C33B).hi33() == "hi:C" && cb[0] is P33A)
    // extension Int: P33A - a builtin conforms, and its own extension method must
    // not hijack a same-named member of a user type.
    let ns = [7]
    check("33k", ns[0].tag33() == "I" && ns[0].hi33() == "hi:I")
    let sd = [S33D()]
    check("33l", sd[0].mine33() == "struct" && ns[0].mine33() == "int")
    // catch filtering on a user protocol: it FILTERS, in both directions.
    var caught = ""
    for n in [0, 1] {
        do { _ = try thrower33(n) }
        catch is P33C { caught += "C" }
        catch { caught += "g" }
    }
    check("33m", caught == "Cg")
    // A protocol INHERITS another, and the parent's default arrives from an
    // extension written after both declarations: the adoption has to reach the
    // parent's own table, not a key list copied before the extension ran.
    let ts: [Any] = [T33A(), T33B()]
    var inh = ""
    for t in ts {
        if let a = t as? A33 { inh += a.a33() }
        inh += (t is A33) ? "1" : "0"
        inh += (t is B33) ? "1" : "0"
    }
    check("33n", inh == "a?11a?10")
    check("33o", g33(T33A()) == "g:a?" && T33S.make33() == "m")
    // A protocol COMPOSITION tests every component, not the first one.
    var comp = ""
    for t in ts { comp += (t is A33 & B33) ? "1" : "0" }
    check("33p", comp == "10")
}
protocol A33 { }
protocol B33 { }
protocol C33: A33, B33 { }
extension A33 { func a33() -> String { return "a?" } }
struct T33A: C33 { }
struct T33B: A33 { }
protocol S33 { static func make33() -> String }
struct T33S: S33 { static func make33() -> String { return "m" } }
func g33<T: A33>(_ x: T) -> String { return "g:" + x.a33() }

// ===== SECTION 34: uppercased()/lowercased() are the FULL Unicode mapping =====
// Read off swiftc 6.1.2: "\u{00df}".uppercased() is "SS", "\u{fb01}" is "FI",
// "\u{0130}".lowercased() is "i" + U+0307 - and there is no Final_Sigma, so a
// word-final capital sigma lowercases to U+03C3, not U+03C2.
//
// Every character is an ESCAPE, never a raw literal: manual chapter 4, test
// specific characters by CODE.
func s34() {
    // The characters that GROW under the full mapping. `.count` is the cluster
    // count.
    check("34a", "\u{00df}".uppercased() == "SS" && "\u{00df}".uppercased().count == 2)
    check("34b", "\u{fb01}".uppercased() == "FI" && "\u{fb01}".uppercased().count == 2)
    check("34c", "\u{fb03}".uppercased() == "FFI" && "\u{fb03}".uppercased().count == 3)
    check("34d", "\u{0149}".uppercased() == "\u{02bc}N" && "\u{0149}".uppercased().count == 2)
    check("34e", "\u{1f50}".uppercased() == "\u{03a5}\u{0313}" && "\u{1f50}".uppercased().count == 1)
    // U+0130 grows on the way DOWN: "i" and a combining dot, one cluster.
    check("34f", "\u{0130}".lowercased() == "i\u{0307}" && "\u{0130}".lowercased().count == 1)
    check("34j", "A\u{03a3}".lowercased() == "a\u{03c3}"
                 && "\u{03a3}\u{0391}\u{03a3}".lowercased() == "\u{03c3}\u{03b1}\u{03c3}")
    // Simple == full here, so these pin the mapping itself rather than the choice:
    // dotless i, final sigma, capital sharp s.
    check("34g", "\u{0131}".uppercased() == "I" && "\u{03c2}".uppercased() == "\u{03a3}"
                 && "\u{1e9e}".lowercased() == "\u{00df}")
    // ASCII, the mixed case, and the identity on a string with no cased character.
    check("34h", "Hello".uppercased() == "HELLO" && "Hello".lowercased() == "hello"
                 && "MIXED case 123".lowercased() == "mixed case 123"
                 && "".uppercased() == "" && "123-".uppercased() == "123-")
    // Non-ASCII that IS one-to-one both ways.
    check("34i", "\u{00e4}\u{00f6}\u{00fc}".uppercased() == "\u{00c4}\u{00d6}\u{00dc}"
                 && "\u{00c4}\u{00d6}\u{00dc}".lowercased() == "\u{00e4}\u{00f6}\u{00fc}")
}

// ===== SECTION 35: Array(...), popLast()/removeLast(), and String as a SEQUENCE =====
// swiftc 6.1.2 answers all of these.
//
// A String is a sequence of extended grapheme CLUSTERS, and `Array(s)` and
// `for c in s` must both yield what `.count` counts: `for c in "cafe\u{301}"`
// is c, a, f, é.
func s35() {
    var a35 = [1, 2, 3]
    check("35a", a35.popLast() == 3 && a35 == [1, 2])
    check("35b", a35.removeLast() == 2 && a35 == [1])
    var e35: [Int] = []
    check("35c", e35.popLast() == nil && e35.count == 0)
    // Array(seq): a copy, a Range's elements, a String's Characters.
    let src35 = [4, 5, 6]
    var cp35 = Array(src35)
    cp35.append(9)
    check("35d", cp35.count == 4 && src35.count == 3)
    check("35e", Array(0..<3) == [0, 1, 2] && Array(1...3) == [1, 2, 3])
    check("35f", Array("hi") == ["h", "i"] && Array("").count == 0)
    check("35g", Array(repeating: 7, count: 3) == [7, 7, 7]
                 && Array(repeating: 0, count: 0).count == 0)
    // A String walked as a sequence, and Array(s), agree with .count - on a
    // combining mark and on an astral pair.
    let uni35 = "cafe\u{301}"
    var seen35 = ""
    var n35 = 0
    for c in uni35 { seen35 += c; n35 += 1 }
    check("35h", n35 == 4 && n35 == uni35.count && seen35 == uni35)
    check("35i", Array(uni35).count == 4 && Array(uni35)[3] == "e\u{301}")
    let ast35 = "a\u{1f600}b"
    var m35 = 0
    for _ in ast35 { m35 += 1 }
    check("35j", m35 == 3 && ast35.count == 3 && Array(ast35).count == 3)
    // THE SAME STRING WRITTEN AS ITSELF. A raw astral character in the source is
    // two code units, and copying the literal one unit at a time would hand a LONE
    // SURROGATE to the host (U+FFFD). swiftc 6.1.2 says 3, and says the two
    // spellings are equal.
    let raw35 = "a😀b"
    check("35k", raw35 == ast35 && raw35.count == 3 && Array(raw35).count == 3)
    check("35l", Array(raw35)[1] == Array(ast35)[1] && String(Array(raw35)[1]).count == 1)
    var r35 = ""
    for c in raw35 { r35 += String(c) }
    check("35m", r35 == raw35 && r35.count == 3)
    // isEmpty / first / last are PROPERTIES on every Collection; swiftc says
    // false / a value. A user type
    // that declares a member of one of those names must still win.
    check("35n", !a35.isEmpty && e35.isEmpty && !raw35.isEmpty && "".isEmpty)
    check("35o", a35.first == 1 && a35.last == 1 && e35.first == nil && e35.last == nil)
    check("35p", raw35.first == "a" && raw35.last == "b" && "".first == nil)
    check("35q", uni35.last == "e\u{301}" && uni35.first == "c")
    check("35r", ["k": 1].isEmpty == false && ([:] as [String: Int]).isEmpty)
    check("35s", Own35().isEmpty == 99 && Own35().first == 7 && Own35().last == 8)
}
struct Own35 { var isEmpty = 99; var first = 7; var last = 8 }

// ===== SECTION 36: the Sequence/Collection members, and Range as a VALUE =====
// A name no engine knows says "unknown Array method: X" in all three.
//
// Every value below was read off swiftc 6.1.2. FOUR THINGS ARE DELIBERATELY NOT
// ASSERTED because they cannot be matched and each is recorded at its site:
//   * min()/max()/firstIndex(of:) answer an OPTIONAL - swiftc prints Optional(1)
//     where this value model has no Optional at all (manual 7.9). The VALUE is
//     asserted; the printed form is not.
//   * reversed() / enumerated() / [[T]].joined() answer ReversedCollection /
//     EnumeratedSequence / FlattenSequence, whose printed forms name the STATIC
//     element type (`ReversedCollection<Array<Int>>(_base: [3, 1, 2])`) and need a
//     type checker. Array(...) of each IS asserted and is byte-exact.
//   * a[1...] is an ArraySlice, which keeps the ORIGINAL indices; a plain array
//     cannot, so only the elements and the count are asserted.
//   * shuffled() has no random source any three engines can agree on, so it draws
//     from a fixed-seed xorshift32 - the assertion is on the SORTED result and the
//     count, never on the order.
func s36() {
    var a36 = [3, 1, 2]
    let e36: [Int] = []
    check("36a", a36.sorted() == [1, 2, 3] && a36 == [3, 1, 2])
    check("36b", a36.sorted(by: { $0 > $1 }) == [3, 2, 1])
    check("36c", a36.reversed() == [2, 1, 3] && Array(a36.reversed()) == [2, 1, 3])
    check("36d", a36.min() == 1 && a36.max() == 3)
    check("36e", e36.min() == nil && e36.max() == nil)
    check("36f", a36.prefix(2) == [3, 1] && a36.suffix(2) == [1, 2])
    check("36g", Array(a36.prefix(9)) == a36 && Array(a36.suffix(9)) == a36)
    check("36h", a36.dropFirst() == [1, 2] && a36.dropLast() == [3, 1])
    check("36i", a36.dropFirst(2) == [2] && Array(a36.dropLast(5)) == e36)
    check("36j", a36.firstIndex(of: 1) == 1 && a36.firstIndex(of: 9) == nil)
    check("36k", a36.allSatisfy { $0 > 0 } && !a36.allSatisfy { $0 > 1 })
    check("36l", a36.compactMap { $0 > 1 ? $0 : nil } == [3, 2])
    check("36m", [[1, 2], [3]].flatMap { $0 } == [1, 2, 3])
    check("36n", a36.indices == 0..<3 && a36.indices.count == 3 && e36.indices == 0..<0)
    check("36o", a36.shuffled().sorted() == [1, 2, 3] && a36.shuffled().count == 3)
    check("36p", a36.contains(where: { $0 > 2 }) && !a36.contains(where: { $0 > 3 }))
    // A user type's own `static func <` WINS: Comparable is how Swift sorts anything
    // that is not a built-in, and the built-in comparison sees two opaque objects.
    // swiftc 6.1.2 says [1, 3].
    let cs36 = [C36(n: 3), C36(n: 1)]
    check("36pa", cs36.sorted().map { $0.n } == [1, 3] && cs36.min()!.n == 1 && cs36.max()!.n == 3)
    // The MUTATING members, each on a FRESH LITERAL.
    var b36 = [3, 1, 2]
    b36.removeFirst()
    check("36q", b36 == [1, 2])
    var c36 = [3, 1, 2]
    check("36r", c36.remove(at: 1) == 1 && c36 == [3, 2])
    var d36 = [3, 1, 2]
    d36.removeAll()
    check("36s", d36 == e36 && d36.isEmpty)
    var f36 = [3, 1, 2]
    f36.removeAll(where: { $0 == 1 })
    check("36t", f36 == [3, 2])
    var g36 = [3, 1, 2]
    g36.swapAt(0, 2)
    check("36u", g36 == [2, 1, 3])
    var h36 = [3, 1, 2]
    h36.sort()
    check("36v", h36 == [1, 2, 3])
    var i36 = [1, 2, 3]
    i36.removeFirst(2)
    check("36w", i36 == [3])
    // enumerated(), joined(), reduce - and an OPERATOR USED AS A VALUE.
    var eo36 = ""
    for (i, v) in a36.enumerated() { eo36 += "\(i):\(v);" }
    check("36x", eo36 == "0:3;1:1;2:2;")
    check("36y", "\(Array(a36.enumerated()))" == "[(offset: 0, element: 3), (offset: 1, element: 1), (offset: 2, element: 2)]")
    let j36 = ["a", "b"]
    check("36z", j36.joined(separator: "-") == "a-b" && j36.joined() == "ab")
    check("36aa", Array([[1, 2], [3]].joined()) == [1, 2, 3])
    check("36ab", a36.reduce(0, +) == 6 && a36.reduce(1, *) == 6 && a36.reduce(0, -) == -6)
    check("36ac", a36.sorted(by: >) == [3, 2, 1])
    check("36ad", a36.reduce(into: [Int]()) { $0.append($1) } == a36)
    // A STRING is a Collection of Characters, so every one of these applies to it;
    // the five whose Swift result is a String/Substring come back as text.
    let s36 = "hello"
    check("36ae", s36.map { String($0) } == ["h", "e", "l", "l", "o"])
    check("36af", s36.filter { $0 != "l" } == "heo")
    check("36ag", s36.prefix(2) == "he" && s36.suffix(2) == "lo")
    check("36ah", s36.dropFirst() == "ello" && s36.dropLast() == "hell")
    check("36ai", s36.allSatisfy { $0 != "z" } && s36.reduce(0, { n, _ in n + 1 }) == 5)
    check("36aj", s36.sorted() == ["e", "h", "l", "l", "o"])
    // `String(s.sorted())` is not String(describing:); SECTION 37 asserts both.
    // A RANGE printed as itself, its `count`, and a for-in over a range VALUE.
    let r36 = 0..<3
    let q36 = 1...3
    check("36ak", "\(r36)" == "0..<3" && "\(q36)" == "1...3")
    check("36al", r36.count == 3 && q36.count == 3)
    var w36: [Int] = []
    for i in r36 { w36.append(i) }
    check("36am", w36 == [0, 1, 2])
    var x36: [Int] = []
    for i in a36.indices { x36.append(i) }
    check("36an", x36 == [0, 1, 2])
    // A ONE-SIDED range subscript.
    check("36ao", Array(a36[1...]) == [1, 2] && a36[1...].count == 2)
    check("36ap", Array(a36[...1]) == [3, 1] && Array(a36[..<2]) == [3, 1])
    check("36aq", Array(a36[0...1]) == [3, 1] && Array(a36[0..<3]) == a36)
    // NOT ASSERTED: `s[1...]` is REJECTED by swiftc - "cannot subscript String with
    // an integer range, use a String.Index range instead" - while both halves here
    // accept it and answer "ello". A String.Index is not representable in this value
    // model, so the superset is kept and recorded rather than removed.
    // (Copy-on-assign for `var b = a` is SECTION 37.)
}

struct C36: Comparable {
    var n: Int
    static func < (l: C36, r: C36) -> Bool { return l.n < r.n }
    static func == (l: C36, r: C36) -> Bool { return l.n == r.n }
}

// ===== SECTION 37: VALUE SEMANTICS - every copy site, and String(_:) =====
// `var b = a` COPIES every Array, Dictionary, tuple and struct (all the way down).
//
// EVERY VALUE BELOW WAS READ OFF swiftc 6.1.2, and the design the rows pin down is
// COPY AT STORE, which is complete: two locations can only alias if one stored a
// value the other still reaches, so if every store deposits a graph nobody else
// holds, no read can produce one. The store sites are a var/let binding, an
// assignment (name, field, subscript), a non-inout argument, a tuple
// destructuring, an if/guard-let binding and a case-pattern binding. There is
// deliberately NO copy on READ, which is what keeps 37i's `a[0].append` in place.
//
// The two shapes that must NOT copy are asserted beside the ones that must: a
// CLASS instance is a reference (37p, 37q - and 37q is the one that says the array
// copy is shallow at a reference element) and an `inout` argument is an alias
// (37d, 37n).
//
// An enum CASE value is deliberately NOT copied: its
// payload is reachable only through a pattern binding, which is itself a copy
// site, so no program can observe two case values sharing a payload. swiftc
// agrees on the probe.
struct P37 { var x: Int; var ys: [Int]
    mutating func bump() { x += 1; ys.append(x) }
    func copyYs() -> [Int] { return ys }
}
struct Q37 { var p: P37 }
struct R37 { var m: [String: Int] }
class C37 { var s: P37; var arr: [Int]; init(_ p: P37) { s = p; arr = [7] } }
enum E37 { case some([Int]); case pair([Int], Int) }

func mutArr37(_ v: [Int]) -> [Int] { var w = v; w.append(99); return w }
func mutArrIn37(_ v: inout [Int]) { v.append(77) }
func mutStructIn37(_ p: inout P37) { p.x = 555 }
func guard37(_ v: [Int]?) -> [Int] { guard var w = v else { return [] }; w.append(3); return w }
func dflt37(_ v: [Int] = [1]) -> Int { var w = v; w.append(2); return w.count }

func s37() {
    // --- the binding sites ---
    var a = [1, 2, 3]
    var b = a
    b.append(9)
    check("37a", a == [1, 2, 3] && b == [1, 2, 3, 9])
    let c = [1, 2, 3]
    var d = c
    d.append(9)
    check("37b", c == [1, 2, 3] && d == [1, 2, 3, 9])
    var e = [1, 2, 3]
    check("37c", mutArr37(e) == [1, 2, 3, 99] && e == [1, 2, 3])
    mutArrIn37(&e)
    check("37d", e == [1, 2, 3, 77])

    // --- reading out of a container gives an independent value ---
    var f = [[1, 2], [3, 4]]
    var g = f[0]
    g.append(5)
    check("37e", f == [[1, 2], [3, 4]] && g == [1, 2, 5])
    var h = ["a": [1]]
    var i = h["a"]!
    i.append(2)
    check("37f", h["a"]! == [1] && i == [1, 2])
    var j = P37(x: 0, ys: [1, 2])
    var k = j.ys
    k.append(3)
    check("37g", j.ys == [1, 2] && k == [1, 2, 3])
    var l = P37(x: 1, ys: [1])
    check("37h", l.copyYs() == [1] && { var w = l.copyYs(); w.append(2); return l.ys == [1] && w == [1, 2] }())

    // --- ...but a subscript WRITE is still in place ---
    var m = [[1, 2], [3, 4]]
    m[0].append(5)
    check("37i", m == [[1, 2, 5], [3, 4]])
    var n = [P37(x: 1, ys: [1])]
    n[0].x = 5
    n[0].ys.append(2)
    check("37j", n[0].x == 5 && n[0].ys == [1, 2])

    // --- struct, nested struct, dictionary, tuple ---
    var o = P37(x: 1, ys: [1])
    var p = o
    p.x = 2
    p.ys.append(2)
    check("37k", o.x == 1 && o.ys == [1] && p.x == 2 && p.ys == [1, 2])
    var q = Q37(p: P37(x: 1, ys: [1]))
    var r = q
    r.p.x = 7
    r.p.ys.append(2)
    check("37l", q.p.x == 1 && q.p.ys == [1] && r.p.x == 7 && r.p.ys == [1, 2])
    var t = R37(m: ["a": 1])
    var u = t
    u.m["b"] = 2
    check("37m", t.m.count == 1 && u.m.count == 2)
    var v = P37(x: 1, ys: [])
    mutStructIn37(&v)
    check("37n", v.x == 555)
    var w = (1, [1, 2])
    var x = w
    x.0 = 9
    x.1.append(3)
    check("37o", w.0 == 1 && w.1 == [1, 2] && x.0 == 9 && x.1 == [1, 2, 3])

    // --- a CLASS is a reference, and the array copy is shallow at one ---
    let y = C37(P37(x: 1, ys: [1]))
    let z = y
    z.arr.append(8)
    check("37p", y.arr == [7, 8] && z.arr == [7, 8])
    var aa = [C37(P37(x: 0, ys: []))]
    var ab = aa
    ab[0].arr.append(9)
    check("37q", aa[0].arr == [7, 9])
    // ...but a STRUCT field of a class instance is read out as a value
    var ac = y.s
    ac.x = 9
    ac.ys.append(2)
    check("37r", y.s.x == 1 && y.s.ys == [1] && ac.x == 9 && ac.ys == [1, 2])

    // --- storing INTO a container copies too ---
    var ad = [1, 2]
    var ae = [[Int]]()
    ae.append(ad)
    ae.insert(ad, at: 0)
    var af = [ad, ad]
    var ag = ["k": ad]
    var ah = (ad, 1)
    var ai = [String: [Int]]()
    ai["k"] = ad
    var aj = P37(x: 0, ys: ad)
    ad.append(3)
    check("37s", ae[0] == [1, 2] && ae[1] == [1, 2] && af[0] == [1, 2] && af[1] == [1, 2])
    check("37t", ag["k"]! == [1, 2] && ah.0 == [1, 2] && ai["k"]! == [1, 2] && aj.ys == [1, 2])
    check("37u", ad == [1, 2, 3])

    // --- tuple destructuring, if/guard-let, case patterns ---
    let ak = ([1, 2], 9)
    var (al, _) = ak
    al.append(3)
    check("37v", ak.0 == [1, 2] && al == [1, 2, 3])
    var am = [1, 2]
    var an = [0]
    var ao = 0
    (an, ao) = (am, 1)
    an.append(3)
    check("37w", am == [1, 2] && an == [1, 2, 3] && ao == 1)
    let ap: [[Int]?] = [[1, 2]]
    if var aq = ap[0] { aq.append(3); check("37x", ap[0]! == [1, 2] && aq == [1, 2, 3]) }
    check("37y", guard37([1, 2]) == [1, 2, 3])
    let ar = E37.some([1, 2])
    if case .some(var asx) = ar { asx.append(3) }
    if case .some(let at) = ar { check("37z", at == [1, 2]) }
    let au = E37.pair([1, 2], 5)
    if case .pair(var av, _) = au { av.append(3) }
    if case .pair(let aw, _) = au { check("37aa", aw == [1, 2]) }
    var ax = P37(x: 1, ys: [1])
    switch ax { case let ay: var az = ay; az.ys.append(2); check("37ab", ax.ys == [1] && az.ys == [1, 2]) }

    // --- a mutating method mutates only its own receiver ---
    var ba = P37(x: 1, ys: [1])
    var bb = ba
    bb.bump()
    check("37ac", ba.x == 1 && ba.ys == [1] && bb.x == 2 && bb.ys == [1, 2])
    // ...and a default argument is fresh at every call
    check("37ad", dflt37() == 2 && dflt37() == 2)
    // ...and the same array passed twice is two arguments
    var bc = [1, 2]
    check("37ae", { (l: [Int], r: [Int]) -> Bool in var m2 = l; m2.append(9); return m2.count == 3 && r.count == 2 }(bc, bc) && bc == [1, 2])

    // --- String(_:) is NOT String(describing:) ---
    // Swift's idiom for rebuilding a String out of Characters.
    let bd = "hello"
    check("37af", String(bd.sorted()) == "ehllo")
    check("37ag", String(describing: bd.sorted()) == "[\"e\", \"h\", \"l\", \"l\", \"o\"]")
    let be: [Character] = ["a", "b"]
    check("37ah", String(be) == "ab")
    check("37ai", String(bd.reversed()) == "olleh")
    // ...and every other argument shape still agrees, which is why the LABEL is
    // the whole of the difference.
    check("37aj", String(3) == "3" && String(describing: 3) == "3"
               && String(3.5) == "3.5" && String(describing: 3.5) == "3.5"
               && String(true) == "true" && String(describing: true) == "true"
               && String("x") == "x" && String(describing: "x") == "x")
}

// ===== SECTION 38: a loop variable is a FRESH BINDING per iteration =====
// Swift's for-in binds its element with a `let` INSIDE the body,
// so a closure made in iteration k captures iteration k's binding and nothing
// else. `for i in 0..<3 { fs.append({ i }) }` then fs[0]() / fs[2]() is 0 / 2 in
// swiftc 6.1.2 (one shared cell would give 3 / 3).
//
// Every loop form that BINDS a name is asserted here, because slot reuse is a
// property of the emitter and not of the sequence: the range loop (makeForRange),
// the list loop over arrays / strings / dictionaries / enumerated() (makeForList),
// `for case` (makeForCase) and `while let` (makeCondWhile) each open their own
// per-iteration scope. `while` and `repeat` bodies are ordinary Blocks, whose
// js_scope_new is emitted INTO the loop body block, so it runs once per
// iteration - 38k/38l keep it that way.
//
// The other half of the property, and the one a careless fix breaks: a `var`
// declared OUTSIDE the loop is still ONE cell, shared by every closure and by the
// loop body (38i), and a `var` declared inside the body is still one cell for the
// whole of that iteration, so a mutating closure and its neighbours see each other
// (38m). Both were checked against swiftc 6.1.2.
//
// A for-in over `stride(from:to:by:)` is makeForList over its materialized
// elements, which is exactly what 38c/38d cover.
func s38() {
    // --- the range loop, half-open and closed ---
    var a: [() -> Int] = []
    for i in 0..<3 { a.append({ i }) }
    check("38a", a[0]() == 0 && a[1]() == 1 && a[2]() == 2)
    var b: [() -> Int] = []
    for i in 0...2 { b.append({ i }) }
    check("38b", b[0]() == 0 && b[1]() == 1 && b[2]() == 2)
    // --- the list loop: an array, and a materialized step sequence ---
    var c: [() -> Int] = []
    for x in [10, 20, 30] { c.append({ x }) }
    check("38c", c[0]() == 10 && c[1]() == 20 && c[2]() == 30)
    var d: [() -> Int] = []
    for x in [0, 2, 4] { d.append({ x }) }
    check("38d", d[0]() == 0 && d[1]() == 2 && d[2]() == 4)
    // --- enumerated(): BOTH names of the tuple pattern are per-iteration ---
    var e: [() -> Int] = []
    for (n, v) in ["p", "qq", "rrr"].enumerated() { e.append({ n * 10 + v.count }) }
    check("38e", e[0]() == 1 && e[1]() == 12 && e[2]() == 23)
    // ...and with a where clause, which reads the binding before the body does
    var f: [() -> Int] = []
    for (n, v) in [5, 6, 7, 8].enumerated() where v % 2 == 0 { f.append({ n * 100 + v }) }
    check("38f", f.count == 2 && f[0]() == 106 && f[1]() == 308)
    // --- a nested for-in captures both levels ---
    var g: [() -> Int] = []
    for i in 0..<2 { for j in 0..<2 { g.append({ i * 10 + j }) } }
    check("38g", g[0]() == 0 && g[1]() == 1 && g[2]() == 10 && g[3]() == 11)
    // --- a string is a sequence of clusters, and each cluster is a binding ---
    var h: [() -> String] = []
    for ch in "xyz" { h.append({ String(ch) }) }
    check("38h", h[0]() == "x" && h[1]() == "y" && h[2]() == "z")
    // --- THE OPPOSITE PROPERTY: a var declared outside the loop is ONE cell ---
    var i2: [() -> Int] = []
    var shared = 0
    for _ in 0..<3 {
        i2.append({ shared })
        shared += 1
    }
    check("38i", shared == 3 && i2[0]() == 3 && i2[1]() == 3 && i2[2]() == 3)
    // ...and a plain `for _ in` still runs the right number of times
    var counted = 0
    for _ in 0..<4 { counted += 1 }
    check("38j", counted == 4)
    // --- while and repeat: the body block was already a per-iteration scope ---
    var k2: [() -> Int] = []
    var kn = 0
    while kn < 3 {
        let m = kn
        k2.append({ m })
        kn += 1
    }
    check("38k", k2[0]() == 0 && k2[1]() == 1 && k2[2]() == 2)
    var l2: [() -> Int] = []
    var ln = 0
    repeat {
        let w = ln * 5
        l2.append({ w })
        ln += 1
    } while ln < 3
    check("38l", l2[0]() == 0 && l2[1]() == 5 && l2[2]() == 10)
    // --- within ONE iteration a body var is still one cell, mutated in place ---
    var mlog = ""
    for i in 0..<2 {
        var n = 0
        let inc = { n += 1 }
        inc()
        inc()
        mlog += "\(i)\(n)"
    }
    check("38m", mlog == "0212")
    // --- for case, one binding and two ---
    var n2: [() -> Int] = []
    let opts: [Int?] = [1, nil, 3]
    for case let v? in opts { n2.append({ v }) }
    check("38n", n2.count == 2 && n2[0]() == 1 && n2[1]() == 3)
    var o2: [() -> String] = []
    let pairs: [(Int, String)?] = [(1, "a"), nil, (2, "b")]
    for case let (num, txt)? in pairs { o2.append({ "\(num)\(txt)" }) }
    check("38o", o2.count == 2 && o2[0]() == "1a" && o2[1]() == "2b")
    // --- while let rebinds every iteration ---
    var p2: [() -> Int] = []
    var q2 = [1, 2, 3]
    while let t = q2.popLast() { p2.append({ t }) }
    check("38p", p2[0]() == 3 && p2[1]() == 2 && p2[2]() == 1)
    // ...and so does a second name declared beside it in the body
    var r2: [() -> String] = []
    var stack = ["p", "q"]
    while let t = stack.popLast() {
        let u = t + "!"
        r2.append({ t + u })
    }
    check("38q", r2[0]() == "qq!" && r2[1]() == "pp!")
    // --- the binding survives the loop leaving early, and a labelled continue ---
    var s2: [() -> Int] = []
    for i in 0..<5 {
        if i == 2 { break }
        s2.append({ i })
    }
    check("38r", s2.count == 2 && s2[0]() == 0 && s2[1]() == 1)
    var t2: [() -> Int] = []
    outer38: for i in 0..<3 {
        for w in 0..<3 {
            if w == 1 { continue outer38 }
            t2.append({ i * 10 + w })
        }
    }
    check("38s", t2.count == 3 && t2[0]() == 0 && t2[1]() == 10 && t2[2]() == 20)
    // --- a closure nested two deep, and one built inside a function and returned ---
    var u2: [() -> Int] = []
    for i in 0..<3 {
        let mk = { { i * 3 } }
        u2.append(mk())
    }
    check("38t", u2[0]() == 0 && u2[1]() == 3 && u2[2]() == 6)
    // --- a tuple pattern over a dictionary, and an index loop ---
    // Two entries, and the answers SORTED: Dictionary order is unspecified in Swift
    // itself, so one entry is the only orderable form and one entry cannot show slot
    // reuse at all (it has no second iteration to overwrite the first).
    var v3: [() -> String] = []
    for (kk, vv) in ["a": 1, "b": 2] { v3.append({ "\(kk)\(vv)" }) }
    var v3s: [String] = []
    for cl in v3 { v3s.append(cl()) }
    check("38u", v3s.count == 2 && v3s.sorted() == ["a1", "b2"])
    var w2: [() -> Int] = []
    let src = [1, 2, 3]
    for idx in src.indices { w2.append({ src[idx] }) }
    check("38v", w2[0]() == 1 && w2[2]() == 3)
    // --- a defer in the body sees its own iteration's binding ---
    var dlog = ""
    for i in 0..<2 {
        defer { dlog += "d\(i)" }
        dlog += "b\(i)"
    }
    check("38w", dlog == "b0d0b1d1")
    // --- the capture SITE varies independently of the loop, so three more of them:
    // a nested func (emitFunc, not a closure literal), a closure built inside a
    // do/catch (makeTry wraps the body in closures of its own over the same scope),
    // and one built inside an if. All three read the per-iteration scope through a
    // different emitter path than a bare closure in the body does.
    var x2: [() -> Int] = []
    for i in 0..<3 {
        func g38() -> Int { return i }
        x2.append(g38)
    }
    check("38x", x2[0]() == 0 && x2[2]() == 2)
    var y2: [() -> Int] = []
    for i in 0..<3 {
        do { y2.append({ i }) } catch { }
    }
    check("38y", y2[0]() == 0 && y2[2]() == 2)
    var z2: [() -> Int] = []
    for i in 0..<3 { if i != 1 { z2.append({ i }) } }
    check("38z", z2.count == 2 && z2[0]() == 0 && z2[1]() == 2)
    // --- a COMMA condition clause: `while a, b` is the same emitter as `while let`
    // (CplxWhile), and it reaches it without binding anything, so it must keep
    // working AND keep paying nothing. Then the two-binding form, where both names
    // are per-iteration.
    var aa = 0
    var ab = 5
    while aa < 3, ab > 0 {
        aa += 1
        ab -= 1
    }
    check("38aa", aa == 3 && ab == 2)
    var ac: [() -> Int] = []
    var ad: [Int?] = [7, 8, 9]
    while let e1 = ad.popLast(), let e2 = e1 { ac.append({ e2 }) }
    check("38ab", ac.count == 3 && ac[0]() == 9 && ac[2]() == 7)
}

// ===== SECTION 39: a write to an EXISTENTIAL slot keeps the value's own type =====
// A write to a numeric slot adopts the slot's type, which the value it already
// holds carries: `var d = 1.5; d = 2` holds 2.0. An existential slot - Any, a
// protocol, `any P`, an Optional of one - holds whatever it was given last, so
// `var a: Any = 1.5; a = 2` holds the Int 2 in swiftc 6.1.2, and the same is true
// of a stored property, an array element, a dictionary value, an implicit
// `self.q` and an inout parameter (adopting would wrongly give 2.0).
// 39k-39n are the controls: inferred and numeric slots still adopt, and a
// shadowing declaration decides for its own scope only. 39p-39r: one property
// NAME declared Any in one type and Double in another is decided by the type of
// the object written, an inherited property included. 39s-39u: the same for an
// element of such a property (`[Any]` in one type, `[Double]` in another).
protocol P39 {}
extension Int: P39 {}
extension Double: P39 {}
struct S39 { var v: Any; var d: Double; var xs: [Any] }
class C39 {
    var q: Any = 1.5
    var r: Double = 1.5
    func setBoth() { q = 8; r = 8 }
}
func io39(_ x: inout Any) { x = 5 }
struct MA39 { var m39: Any; mutating func set() { m39 = 4 } }
struct MD39 { var m39: Double; mutating func set() { m39 = 4 } }
class BA39 { var m39: Any = 1.5; func set() { m39 = 5 } }
class SA39: BA39 { func set2() { m39 = 6 } }
class N39 { var e39 = [1.5, 2.5]; var dv39 = ["a": 1.5] }
struct X39 { var e39: [Any]; var dv39: [String: Any] }
class XB39 { var e39: [Any] = [1.5] }
class XS39: XB39 { func put() { e39[0] = 4 } }
func s39() {
    var a: Any = 1.5
    a = 2
    check("39a", "\(a)" == "2")
    var arr: [Any] = [1.5, "x"]
    arr[0] = 3
    check("39b", "\(arr[0])" == "3" && "\(arr[1])" == "x")
    var s = S39(v: 1.5, d: 1.5, xs: [1.5])
    s.v = 7
    s.d = 7
    s.xs[0] = 3
    check("39c", "\(s.v)" == "7" && "\(s.d)" == "7.0" && "\(s.xs[0])" == "3")
    var dd: [String: Any] = ["k": 1.5]
    dd["k"] = 4
    let dk = dd["k"]!
    check("39d", "\(dk)" == "4")
    var oa: Any? = 1.5
    oa = 3
    check("39e", "\(oa!)" == "3")
    var p: P39 = 1.5
    p = 5
    var ep: any P39 = 1.5
    ep = 6
    check("39f", "\(p)" == "5" && "\(ep)" == "6")
    var nest: [[Any]] = [[1.5]]
    nest[0][0] = 3
    check("39g", "\(nest[0][0])" == "3")
    let c = C39()
    c.setBoth()
    check("39h", "\(c.q)" == "8" && "\(c.r)" == "8.0")
    c.q = 6
    c.r = 6
    check("39i", "\(c.q)" == "6" && "\(c.r)" == "6.0")
    var ia: Any = 1.5
    io39(&ia)
    check("39j", "\(ia)" == "5")
    var d = 1.5
    d = 2
    check("39k", "\(d)" == "2.0")
    var dn: [String: Double] = ["k": 1.5]
    dn["k"] = 4
    let dnk = dn["k"]!
    check("39l", "\(dnk)" == "4.0")
    var sh = 1.5
    do {
        var sh: Any = 1.5
        sh = 2
        check("39m", "\(sh)" == "2")
    }
    sh = 3
    var sh2: Any = 1.5
    do {
        var sh2 = 1.5
        sh2 = 2
        check("39n", "\(sh2)" == "2.0")
    }
    sh2 = 2
    check("39o", "\(sh)" == "3.0" && "\(sh2)" == "2")
    var ma = MA39(m39: 1.5)
    ma.m39 = 2
    let ma1 = "\(ma.m39)"
    ma.set()
    var md = MD39(m39: 1.5)
    md.m39 = 2
    let md1 = "\(md.m39)"
    md.set()
    check("39p", ma1 == "2" && "\(ma.m39)" == "4" && md1 == "2.0" && "\(md.m39)" == "4.0")
    let sa = SA39()
    sa.m39 = 3
    let sa1 = "\(sa.m39)"
    sa.set()
    let sa2 = "\(sa.m39)"
    sa.set2()
    check("39q", sa1 == "3" && sa2 == "5" && "\(sa.m39)" == "6")
    var ams = [MA39(m39: 1.5)]
    ams[0].m39 = 9
    var mds = [MD39(m39: 1.5)]
    mds[0].m39 = 9
    check("39r", "\(ams[0].m39)" == "9" && "\(mds[0].m39)" == "9.0")
    var x = X39(e39: [1.5], dv39: ["a": 1.5])
    x.e39[0] = 3
    x.dv39["a"] = 2
    let n = N39()
    n.e39[0] = 3
    n.dv39["a"] = 2
    check("39s", "\(x.e39[0]) \(x.dv39["a"]!)" == "3 2" && "\(n.e39[0]) \(n.dv39["a"]!)" == "3.0 2.0")
    let xs = XS39()
    xs.put()
    check("39t", "\(xs.e39[0])" == "4")
    var xa = [X39(e39: [0.5], dv39: [:])]
    xa[0].e39[0] = 8
    check("39u", "\(xa[0].e39[0])" == "8")
}

// ===== SECTION 40: optional subscripts, `as?` before `??`, builtin extensions =====
// A missing dictionary key is nil (only `!` traps); `x as? T ?? y` is
// `(x as? T) ?? y` with y typed T; a Dictionary's element is a (key:, value:)
// pair; inside an extension of a builtin, `count` and `uppercased()` are members of
// self. Every value below is swiftc 6.1.2's.
extension String {
    var lenPlus40: Int { return count + 1 }
    func shout40() -> String { return uppercased() + "!" }
}
extension Double {
    var bad40: Bool { return isNaN || isInfinite }
}
extension Array {
    var half40: Int { return count / 2 }
}
func s40() {
    let d: [String: String] = ["a": "1"]
    let ks = ["a", "b"]
    check("40a", (d[ks[0]] ?? "x") == "1" && (d[ks[1]] ?? "x") == "x")
    var hit = ""
    for k in ks {
        if let v = d[k] { hit += v } else { hit += "-" + k }
    }
    check("40b", hit == "1-b")
    var m = [String: Int]()
    m["q"] = 5
    check("40c", m["z"] == nil && m["q"] != nil && (m["z"] ?? -1) == -1 && m["q"]! + 1 == 6)
    let xs: [Any] = [1.5, "s", 2]
    var cs = ""
    for x in xs { cs += "\(x as? Double ?? -1) \(x as? Int ?? 0 + 1);" }
    check("40d", cs == "1.5 1;-1.0 1;-1.0 2;")
    let one = ["a": 1]
    var fs = "none"
    if let f = one.first { fs = "\(f.key) \(f.value) \(f)" }
    check("40e", fs == "a 1 (key: \"a\", value: 1)" && [String: Int]().first == nil)
    check("40f", "abc".lenPlus40 == 4 && "hi".shout40() == "HI!" && [1, 2, 3, 4].half40 == 2)
    let ds = [1.5, Double.nan]
    check("40g", !ds[0].bad40 && ds[1].bad40)
}

// ===== SECTION 41: write adoption - Optional, alias, observer, inferred locals =====
// A write to a numeric slot adopts its type (`Double?` included); a write to an
// existential one keeps what it is given - also when the existential is spelled
// through a typealias, is an observed or computed property, or is inferred from a
// local, an `if var` / `for var` source or a closure parameter. The top-level write
// below runs before the protocol it names is declared. Every value is swiftc 6.1.2's.
var top41: Num41 = 1.5
top41 = 2
let top41Txt = "\(top41)"
protocol Num41 {}
extension Double: Num41 {}
extension Int: Num41 {}
typealias JSON41 = [String: Any]
typealias Any41 = Any
class Obs41 {
    var p: Any = 1.5 { didSet { } }
    var q: Any { get { return store } set { store = newValue } }
    var store: Any = 1.5
}
class Sub41: Obs41 {}
class Num41B { var p: Double = 1.5; var q: Double = 0.5 }
func pass41(_ v: Any) -> String { var u = v; u = 4; return "\(u)" }
func s41() {
    var x: Double? = nil
    x = 3
    check("41a", "\(x!)" == "3.0")
    var j: JSON41 = ["a": 1.5]
    j["a"] = 2
    var at: Any41 = 1.5
    at = 2
    check("41b", "\(j["a"]!)" == "2" && "\(at)" == "2")
    let o = Obs41()
    o.p = 2
    o.q = 3
    check("41c", "\(o.p) \(o.q)" == "2 3")
    let anys: [Any] = [1.5]
    var fv = ""
    for var e in anys { e = 7; fv += "\(e)" }
    for e in anys { var z = e; z = 6; fv += " \(z)" }
    let oa: Any? = 1.5
    if var w = oa { w = 8; fv += " \(w)" }
    let od: Double? = 2.5
    if var w = od { w = 5; fv += " \(w)" }
    check("41d", fv == "7 6 8 5.0")
    let cl = { (v: Any) -> String in var u = v; u = 9; return "\(u)" }
    check("41e", cl(1.5) == "9" && pass41(1.5) == "4")
    check("41f", top41Txt == "2")
    let nb = Num41B()
    nb.p = 2
    nb.q = 3
    let sb = Sub41()
    sb.p = 4
    check("41g", "\(nb.p) \(nb.q) \(sb.p)" == "2.0 3.0 4")
}

// ===== SECTION 42: an extension-only member through a protocol-typed receiver =====
// A member a protocol extension adds and the protocol does not require is bound
// STATICALLY: through a value whose declared type is the protocol it is the
// extension's, through the concrete type it is the type's own. A requirement
// (here an inherited one) still dispatches dynamically. Every value is swiftc 6.1.2's.
protocol Base42 { func req() -> String }
protocol Named42: Base42 { var name: String { get } }
extension Named42 {
    var shout: String { return name + "!" }
    func greet(to who: String, times n: Int) -> String { return "hi \(who) x\(n) " + name }
    func req() -> String { return "default req" }
}
class K42: Named42 {
    var name: String { return "k" }
    var shout: String { return "own" }
    func greet(to who: String, times n: Int) -> String { return "own greet" }
    func req() -> String { return "K req" }
}
struct S42: Named42 { var name = "s" }
func use42(_ a: Named42, _ b: K42) -> String { return a.shout + "|" + b.shout + "|" + a.req() }
func s42() {
    let d: Named42 = K42()
    let c = K42()
    check("42a", d.shout == "k!" && d.greet(to: "b", times: 2) == "hi b x2 k" && d.req() == "K req"
                 && c.shout == "own" && c.greet(to: "b", times: 2) == "own greet")
    let ps: [Named42] = [K42(), S42()]
    check("42b", ps[0].shout + ps[1].shout == "k!s!")
    let o: Named42? = K42()
    check("42c", o!.shout == "k!" && use42(K42(), K42()) == "k!|own|K req")
}

// ===== SECTION 43: a call written with only a trailing closure, as an expression =====
// `name { ... }` is a call in every expression position: a return value, an
// initializer, an argument, and the implicit-self member call inside an extension.
// `if`/`while`/`guard`/`switch`/`for`/`catch` braces and `didSet` observer braces
// are never a trailing closure. Every value is swiftc 6.1.2's.
extension Array where Element == Int {
    func big43() -> [Int] { return filter { $0 > 1 } }
    func dbl43() -> [Int] { map { $0 * 2 } }
}
func apply43(_ f: () -> Int) -> Int { return f() }
func ret43() -> Int { return apply43 { 8 } }
func s43() {
    let v = apply43 { 7 }
    check("43a", v == 7 && ret43() == 8 && apply43 { 2 } + 1 == 3)
    check("43b", [1, 2, 3].big43() == [2, 3] && [1, 2].dbl43() == [2, 4])
    var hits = 0
    let ok = v == 7
    if ok { hits += 1 }
    while hits < 3 { hits += 1 }
    check("43c", hits == 3 && [apply43 { 4 }, apply43 { 5 }] == [4, 5])
}

// ===== SECTION 44: a Dictionary as a Sequence of (key:, value:) pairs =====
// One binder in a for-in takes the whole pair; the Sequence members run over the
// pairs, and `filter` alone answers a Dictionary. Swift's iteration order is
// unspecified, so every check is order-free. Every value is swiftc 6.1.2's.
func s44() {
    let one = ["a": 1]
    var seen = ""
    for e in one { seen += e.key + String(e.value) + "\(e)" }
    check("44a", seen == "a1(key: \"a\", value: 1)")
    let d = ["a": 1, "b": 2, "c": 3]
    var keys = [String]()
    for (k, v) in d where v > 1 { keys.append(k) }
    check("44b", keys.sorted() == ["b", "c"] && d.map { $0.value * 2 }.sorted() == [2, 4, 6])
    let f = d.filter { $0.value > 1 }
    check("44c", f.count == 2 && f["b"] == 2 && f["a"] == nil && d.filter { $0.value > 5 }.isEmpty)
    check("44d", d.first(where: { $0.value == 2 })!.key == "b" && d.min(by: { $0.value < $1.value })!.key == "a"
                 && d.max(by: { $0.value < $1.value })!.value == 3)
    let s = d.sorted(by: { $0.key < $1.key })
    check("44e", s.map { $0.key } == ["a", "b", "c"] && s[2].value == 3 && d.sorted { $0.value > $1.value }[0].key == "c")
    check("44f", d.contains(where: { $0.key == "b" }) && !d.contains(where: { $0.value > 3 })
                 && d.reduce(0) { $0 + $1.value } == 6 && d.count(where: { $0.value > 1 }) == 2
                 && d.allSatisfy { $0.value > 0 } && d.values.sorted() == [1, 2, 3])
}

// ===== SECTION 45: the String members that take a String =====
// hasPrefix / hasSuffix / starts(with:) / contains(_:) compare text; the closure
// forms contains(where:), first(where:) and count(where:) walk the Characters, as
// first(where:) / last(where:) / count(where:) walk an Array.
// Every value is swiftc 6.1.2's.
func s45() {
    let ws = ["abc", "xbz", ""]
    var row = ""
    for w in ws {
        row += "\(w.hasPrefix("ab")),\(w.hasSuffix("bc")),\(w.contains("b")),\(w.starts(with: "x"));"
    }
    check("45a", row == "true,true,true,false;false,false,true,true;false,false,false,false;")
    let w = ws[0]
    check("45b", w.hasPrefix("") && w.hasSuffix("") && !w.hasSuffix("zabc") && w.contains("bc") && !w.contains("ac"))
    check("45c", w.contains(where: { $0 == "c" }) && !w.contains(where: { $0 == "q" })
                 && w.first(where: { $0 > "a" })! == "b" && w.count(where: { $0 != "b" }) == 2)
    check("45d", [3, 1, 2].first(where: { $0 < 3 })! == 1 && [3, 1, 2].last(where: { $0 > 1 })! == 2
                 && [3, 1, 2].count(where: { $0 > 1 }) == 2 && [1].first(where: { $0 > 1 }) == nil)
}

// ===== SECTION 46: Optional(v), Optional.map and Optional.flatMap =====
// map/flatMap on an Optional hand the wrapped value to the closure and keep nil
// nil - also when the wrapped value is itself a String or an Array, which the
// receiver's static type (an annotation, an inferred `.first`, a dictionary
// subscript, a function's return type) tells apart from the collection's own map.
// Every value is swiftc 6.1.2's.
func ret46() -> String? { return "gg" }
struct N46 { var name: String? = "nm" }
func s46() {
    let o = Optional(2.5)
    let oi = Optional(4)
    check("46a", o! == 2.5 && oi.map { $0 + 1 }! == 5)
    let xs = ["h", "e"]
    let first: String? = xs.first
    let none: [String] = []
    let f2 = xs.first
    check("46b", (first.map { String($0) } ?? "-") == "h" && (none.first.map { String($0) } ?? "-") == "-"
                 && (f2.map { $0 + "!" } ?? "-") == "h!" && xs.min().map { $0 + "?" }! == "e?")
    let n: Int? = 3
    let z: Int? = nil
    check("46c", n.map { $0 * 2 }! == 6 && n.flatMap { $0 > 2 ? $0 + 1 : nil }! == 4
                 && (z.map { $0 * 2 } ?? -1) == -1 && (n.flatMap { _ -> Int? in nil } ?? -1) == -1)
    let words: [String]? = ["a", "b"]
    let d = ["k": "v"]
    check("46d", words.map { $0.count }! == 2 && ret46().map { $0.count }! == 2
                 && d["k"].map { $0 + "w" }! == "vw" && (d["q"].map { $0 + "w" } ?? "none") == "none"
                 && (N46().name.map { $0.count } ?? -1) == 2)
}

// ===== SECTION 47: an extension-only member through a call result or a property =====
// SECTION 42's static binding, for a receiver whose static type is a function's
// declared return type or a stored property's declared type, an implicit `self`
// property's included. A concrete return type keeps the type's own member.
// Every value is swiftc 6.1.2's.
protocol P47 { func req() -> String }
extension P47 {
    var shout: String { return "P" }
    func greet(_ n: Int) -> String { return "Pg\(n)" }
    func req() -> String { return "Preq" }
}
struct S47: P47 {
    var shout: String { return "S" }
    func greet(_ n: Int) -> String { return "Sg\(n)" }
    func req() -> String { return "Sreq" }
}
func f47() -> P47 { return S47() }
func fs47() -> S47 { return S47() }
func fo47() -> P47? { return S47() }
struct A47 {
    var b: P47 = S47()
    var c: S47 = S47()
    var ps: [P47] = [S47()]
    func inner() -> String { return b.shout + c.shout + self.b.shout }
}
func s47() {
    check("47a", f47().shout + f47().greet(1) + f47().req() == "PPg1Sreq" && fs47().shout + fs47().greet(2) == "SSg2"
                 && fo47()!.shout + (fo47()?.shout ?? "-") == "PP")
    let a = A47()
    check("47b", a.b.shout + a.c.shout + a.ps[0].shout + a.b.greet(3) + a.b.req() == "PSPPg3Sreq" && a.inner() == "PSP")
}

// ===== SECTION 48: observed variables =====
// willSet/didSet on a top-level and on a local variable, not only on a property
// (swiftc 6.1.2): the observers see the incoming and the previous value, a write
// inside the variable's own didSet does not fire it again, and a local of the same
// name in another function is not observed.
var log48 = ""
var w48 = 0 { didSet { log48 += "d\(oldValue)>\(w48)," } }
var lim48: Double = 1 {
    willSet(nv) { log48 += "w\(nv)," }
    didSet { if lim48 > 10 { lim48 = 10 } }
}
func bump48() { w48 += 2 }
func shadow48() -> Int { var w48 = 5; w48 = 6; return w48 }
func s48() {
    w48 = 3
    bump48()
    lim48 = 50
    var loc = 1 { didSet(o) { log48 += "l\(o)>\(loc)" } }
    loc = 9
    check("48a", log48 == "d0>3,d3>5,w50.0,l1>9" && lim48 == 10 && shadow48() == 6 && loc == 9)
}

// ===== SECTION 49: explicit generic arguments at a call =====
// `Optional<Int>(5)` is a call, not the comparison chain `Optional < Int > (5)`
// (swiftc 6.1.2: 5, 0 and [1.0, 2.0]); the arguments of Array/Dictionary/Optional
// also type the result, and those of a user generic type are only its spelling.
struct Box49<T> { var v: T }
func s49() {
    let o = Optional<Int>(5)
    var a = Array<Int>()
    a.append(3)
    var d = Dictionary<String, Int>()
    d["k"] = 2
    let ds = Array<Double>([1, 2])
    let od = Optional<Double>(3)
    let none = Optional<Int>.none
    let x = 3, y = 4
    let empty: [Int] = Array()
    let ed: [String: Int] = Dictionary()
    check("49a", o! == 5 && a == [3] && d.count == 1 && Array<Array<Int>>().count == 0)
    check("49b", "\(ds) \(od!)" == "[1.0, 2.0] 3.0" && none == nil && Box49<Int>(v: 9).v == 9)
    check("49c", (x<y) && !(x>y) && empty.isEmpty && ed.isEmpty && Character("q") == "q")
    check("49d", Array<Int>(repeating: 7, count: 2) == [7, 7] && Set<Int>().isEmpty)
}

// ===== SECTION 50: Set =====
// A Set built by Set(), Set(seq) and a `Set` annotation on a literal; its own
// members, the Sequence ones, ==, value semantics, and the printed form (swiftc
// 6.1.2; a Set's iteration order is per-process random there, so only one-member
// sets are printed and everything else is sorted first).
struct Box50 { var seen: Set<String> = [] }
func s50() {
    var s: Set<Int> = [7]
    let ins = s.insert(7)
    check("50a", "\(s) \([s])" == "[7] [Set([7])]" && !ins.inserted && ins.memberAfterInsert == 7
                 && s.insert(8).inserted && s.remove(8)! == 7 + 1 && s.remove(9) == nil)
    check("50b", Set([1, 2]) == Set([2, 1]) && Set([1]) != Set([1, 3]) && Set("aab").count == 2
                 && Set(1...4).count == 4 && Set<String>().isEmpty && !s.isEmpty && s.first! == 7)
    var u = Set<Int>()
    for i in [5, 3, 5, 1, 3] { u.insert(i) }
    var walked = 0
    for x in u { walked += x }
    check("50c", u.count == 3 && u.sorted() == [1, 3, 5] && walked == 9 && Array(u).count == 3
                 && u.map { $0 * 2 }.sorted() == [2, 6, 10] && u.filter { $0 > 2 }.sorted() == [3, 5]
                 && u.contains(3) && !u.contains(4) && u.contains { $0 > 4 } && u.reduce(0, +) == 9)
    check("50d", u.union([9, 1]).count == 4 && u.intersection([1, 9]).sorted() == [1]
                 && u.subtracting([1]).sorted() == [3, 5] && u.symmetricDifference([1, 9]).sorted() == [3, 5, 9]
                 && u.isSubset(of: [1, 3, 5, 7]) && u.isSuperset(of: [1]) && u.isDisjoint(with: [2])
                 && u.isStrictSubset(of: [1, 3, 5, 7]) && !u.isStrictSuperset(of: [1, 3, 5]))
    var t = u
    t.insert(100)
    u.formUnion([9])
    u.subtract([3])
    u.formIntersection([1, 9, 4])
    var bx = Box50()
    bx.seen.insert("a"); bx.seen.insert("a"); bx.seen.insert("b")
    var r = Set(1...4)
    r = [2, 2, 3]
    check("50e", t.count == 4 && u.sorted() == [1, 9] && bx.seen.count == 2 && r.count == 2
                 && Set([[1], [1], [2]]).count == 2 && Set(["x", "y"]).sorted(by: >) == ["y", "x"])
    u.removeAll()
    check("50f", u.isEmpty && "\(u)" == "[]")
}

// ===== SECTION 51: String, Array and Dictionary members =====
// split (all three forms and both options, Swift's own algorithm), the String
// views, firstIndex/lastIndex on a String (a Character offset here; swiftc's
// String.Index prints as a UTF-8 offset, so only its use is asserted),
// starts/elementsEqual, prefix(while:)/drop(while:), mapValues, Character(_:),
// Set(String) and .description (swiftc 6.1.2). An astral character inside a
// collection prints whole.
func s51() {
    let s = "a,b,,c"
    check("51a", s.split(separator: ",") == ["a", "b", "c"]
                 && s.split(separator: ",", maxSplits: 1) == ["a", "b,,c"]
                 && s.split(separator: ",", omittingEmptySubsequences: false) == ["a", "b", "", "c"]
                 && s.split(separator: ",", maxSplits: 2, omittingEmptySubsequences: false) == ["a", "b", ",c"]
                 && ",,".split(separator: ",").isEmpty && s.split { $0 == "," }.count == 3
                 && "e\u{301}xe".split(separator: "e").count == 1)
    let nums = [1, 0, 2, 0, 0, 3, 0]
    check("51b", nums.split(separator: 0).map { Array($0) } == [[1], [2], [3]]
                 && nums.split(separator: 0, omittingEmptySubsequences: false).count == 5
                 && nums.split(separator: 0, maxSplits: 2).map { $0.count } == [1, 1, 3])
    let ib = s.firstIndex(of: "b")!
    check("51c", String(s[s.startIndex..<ib]) == "a," && s.firstIndex(where: { $0 == "," }) != nil
                 && s.lastIndex(of: ",") != nil && s.firstIndex(of: "z") == nil
                 && "héllo😀".utf8.count == 10 && "héllo😀".unicodeScalars.count == 6 && s.lazy.count == 6
                 && Array("é".utf8) == [195, 169])
    check("51d", [1, 2, 3].starts(with: [1, 2]) && ![1, 2, 3].starts(with: [2]) && ![1].starts(with: [1, 2])
                 && [1, 2].elementsEqual([1, 2]) && ![1, 2].elementsEqual([1, 2, 3]) && "abc".elementsEqual("abc")
                 && [3, 1, 2].drop(while: { $0 > 2 }) == [1, 2] && [3, 4, 1].prefix(while: { $0 > 2 }) == [3, 4]
                 && "abcd".drop(while: { $0 < "c" }) == "cd")
    let mv = ["b": 1, "a": 2].mapValues { $0 * 10 }
    check("51e", mv["a"]! == 20 && mv["b"]! == 10 && "\(["k": 1].mapValues { String($0) })" == "[\"k\": \"1\"]"
                 && ["k": 3, "j": 1].compactMapValues { $0 > 2 ? $0 : nil } == ["k": 3])
    check("51f", Character("b") == "b" && Set("hello").count == 4 && s.description == s
                 && [1, 2].description == "[1, 2]" && true.description == "true" && "\(["😀"])" == "[\"😀\"]")
}

// ===== SECTION 52: an untyped closure parameter typed by the return type =====
// `{ x -> Double in x / 2 }(3)` divides a Double (swiftc 6.1.2: 1.5), because x's
// value flows into the Double result; inside a conversion it keeps the literal's
// Int (`Double(x / 2)` is 1.0), and an index stays an Int.
func s52() {
    let arr = [1.5, 2.5, 4.0]
    let f = { x -> Double in x / 2 }
    check("52a", { x -> Double in x / 2 }(3) == 1.5 && f(5) == 2.5 && { x, y -> Double in x / y }(3, 2) == 1.5
                 && { x -> Double in let h = x / 2; return h }(3) == 1.5 && { x -> Float in x / 3 }(1) == Float(1) / 3)
    check("52b", { x -> Double in Double(x / 2) }(3) == 1.0 && { x -> Int in x / 2 }(3) == 1
                 && { i -> Double in arr[i] / 2 }(2) == 2.0 && { x -> Double in abs(x) / 2 }(-3) == 1.5
                 && "\([3, 4].map { v -> Double in v / 8 })" == "[0.375, 0.5]" && { x -> Double? in x / 2 }(3)! == 1.5)
}

// ===== SECTION 53: an implicit `.none` with an Optional context =====
// No enum of this file declares a case `none`, so a leading-dot `.none` can only
// be Optional's: nil, as a value and as a pattern (swiftc 6.1.2).
func opt53(_ x: Int?) -> String { return x == nil ? "nil" : "val" }
func none53() -> Int? { return .none }
func s53() {
    let n: Int? = .none
    var s: String? = "x"
    s = .none
    let arr: [Int?] = [1, .none, 3]
    check("53a", n == nil && s == nil && opt53(.none) == "nil" && opt53(3) == "val" && none53() == nil
                 && arr[1] == nil && (n ?? 9) == 9 && n == .none)
    var hit = ""
    let z = Int("x")
    switch z { case .none: hit = "none"; default: hit = "other" }
    let o: Int? = arr[2]
    switch o { case .none: hit += "-none"; case let v?: hit += "-\(v)" }
    if case .none = z { hit += "-if" }
    check("53b", hit == "none-3-if")
}

// ===== SECTION 54: an in-place mutation of an observed variable =====
// `a.append(2)`, `a[0] = 5`, `p.x = 3` and a mutating method each write the WHOLE
// value of an observed variable of value type: willSet sees the mutated value and
// didSet the previous one. A class instance's members are not the variable's
// value, so mutating one fires nothing. Every expected value is swiftc 6.1.2's.
struct P54 { var x = 0; var y = 0; mutating func bump() { x += 1 }; func peek() -> Int { return x } }
final class C54 { var n = 0 }
// The same through an observed stored PROPERTY: `b.items.append(9)`, `b.items[0]
// = 8`, a mutating method on `b.pt`, and the implicit-self forms in a method.
final class Box54 {
    var log = ""
    var items: [Int] = [] { willSet { log += "w\(newValue.count)," } didSet { log += "d\(oldValue.count)>\(items.count);" } }
    var pt = P54() { didSet { log += "p\(oldValue.x)>\(pt.x);" } }
    func add(_ v: Int) { items.append(v) }
    func setFirst(_ v: Int) { items[0] = v }
    func bumpIt() { pt.bump() }
}
func s54() {
    var log = ""
    var a = [1] { willSet { log += "w\(newValue.count)," } didSet { log += "d\(a.count)/\(oldValue.count);" } }
    a.append(2)
    a[0] = 5
    a[1] += 1
    a.remove(at: 0)
    a.sort()
    _ = a.count
    check("54a", log == "w2,d2/1;w2,d2/2;w2,d2/2;w1,d1/2;w1,d1/1;" && a == [3])
    var plog = ""
    var p = P54() { didSet { plog += "\(oldValue.x)>\(p.x)," } }
    p.x = 3
    p.bump()
    p.y += 2
    _ = p.peek()
    check("54b", plog == "0>3,3>4,4>4," && p.y == 2)
    var clog = 0
    var c = C54() { didSet { clog += 1 } }
    c.n = 4
    check("54c", clog == 0 && c.n == 4)
    let b = Box54()
    b.items.append(9)
    b.items[0] = 8
    b.pt.x = 4
    b.pt.bump()
    check("54d", b.log == "w1,d0>1;w1,d1>1;p0>4;p4>5;" && b.items == [8])
    b.log = ""
    b.add(5)
    b.setFirst(7)
    b.bumpIt()
    let bs = [Box54()]
    bs[0].items.append(1)
    check("54e", b.log == "w2,d1>2;w2,d2>2;p5>6;" && b.items == [7, 5] && bs[0].log == "w1,d0>1;")
}

// ===== SECTION 55: an Optional kept inside an Any; a Unicode.Scalar in a collection =====
// `x as Any` of an Optional keeps the Optional (`Optional(7)`, nil stays nil) and a
// cast back out unwraps it. A Unicode.Scalar prints as its glyph, but inside a
// collection as its debugDescription, which escapes everything outside printable
// ASCII. Every expected value is swiftc 6.1.2's.
func half55(_ x: Int) -> Int? { return x % 2 == 0 ? x / 2 : nil }
func s55() {
    let ns = [7, 4]
    let o: Int? = ns[0]
    let a = o as Any
    let s: String? = "q"
    check("55a", "\(a)" == "Optional(7)" && "\(s as Any)" == "Optional(\"q\")" && "\(half55(ns[1]) as Any)" == "Optional(2)"
                 && "\(half55(ns[0]) as Any)" == "nil" && "\([o as Any, 1])" == "[Optional(7), 1]")
    check("55b", (a as? Int)! + 1 == 8 && a is Int && (a as! Int) * 2 == 14)
    let u: Unicode.Scalar = "\u{E9}"
    let sc = Array("a\u{E9}\u{1F600}".unicodeScalars)
    check("55c", "\(u)" == "\u{E9}" && "\([u])" == "[\"\\u{00E9}\"]" && "\(sc)" == "[\"a\", \"\\u{00E9}\", \"\\u{0001F600}\"]"
                 && "\((u, 1))" == "(\"\\u{00E9}\", 1)")
    check("55d", u.value == 233 && sc[2].value == 128512 && sc[0] == "a" && sc[1] == u && String(sc[1]) == "\u{E9}"
                 && Character(sc[0]) == "a")
}

// ===== SECTION 56: removing a key with nil; the mutating String methods =====
// `d[k] = nil` removes the key, for an Optional value type too. A String is a value,
// so `s.append("c")` and the other mutating methods rewrite the variable, field or
// element they are called on. Every expected value is swiftc 6.1.2's.
struct Tag56 { var name = "n"; mutating func grow() { name.append("!") } }
final class Note56 { var t = "c"; func add() { self.t.append("2"); t.append("3") } }
func s56() {
    let ks = ["k", "z"]
    var d: [String: Int] = ["k": 1, "j": 2]
    d[ks[0]] = nil
    d[ks[1]] = nil
    var od: [String: Int?] = ["a": 1]
    od["a"] = nil
    check("56a", d.count == 1 && d["k"] == nil && d["j"] == 2 && od.isEmpty)
    var s = "ab"
    s.append("c")
    s.append(contentsOf: "de")
    s.insert("X", at: s.firstIndex(of: "c")!)
    let rm = s.remove(at: s.firstIndex(of: "a")!)
    check("56b", s == "bXcde" && rm == "a")
    let f = s.removeFirst()
    let l = s.removeLast()
    s.removeAll(where: { $0 == "X" })
    check("56c", f == "b" && l == "e" && s == "cd" && s.popLast() == "d" && s == "c")
    s.removeAll()
    var caf = "cafe"
    caf.append("\u{301}")
    var w = "hello"
    w.removeFirst(2)
    w.removeLast(1)
    check("56d", s.isEmpty && s.popLast() == nil && caf.count == 4 && w == "ll")
    var tg = Tag56()
    tg.name.append("1")
    tg.grow()
    let nt = Note56()
    nt.add()
    nt.t.append("4")
    var arr = ["p", "q"]
    arr[1].append("r")
    arr.append("s")
    var dd = ["a": "x"]
    dd["a"]!.append("y")
    check("56e", tg.name == "n1!" && nt.t == "c234" && arr == ["p", "qr", "s"] && dd["a"] == "xy")
}

// ===== END SECTIONS =====

// ===== SECTION 57: String.Index moves, removeSubrange/replaceSubrange, append(contentsOf:) =====
// An index is the Character offset here, so only its printed form differs from
// swiftc's (`13[utf8]`); every comparison, move and subscript below agrees. A
// user type's own index(after:) / startIndex keep their meaning. Every expected
// value is swiftc 6.1.2's.
struct Ring57 {
    var n: Int
    func index(after i: Int) -> Int { return (i + 1) % n }
    var startIndex: Int { return 7 }
}
func s57() {
    let ks = [1, 2, 4]
    var s = "hello"
    s.insert("x", at: s.startIndex)
    s.insert("y", at: s.endIndex)
    s.insert("Z", at: s.index(s.startIndex, offsetBy: ks[1]))
    check("57a", s == "xhZelloy" && s[s.index(after: s.startIndex)] == "h" && s[s.index(before: s.endIndex)] == "y")
    var t = "héllo wörld"
    let mid = t.index(t.startIndex, offsetBy: ks[2] + 2)
    check("57b", t.distance(from: mid, to: t.endIndex) == 5 && t[mid...] == "wörld")
    t.removeSubrange(t.startIndex..<mid)
    t.replaceSubrange(t.index(before: t.endIndex)..<t.endIndex, with: "D!")
    var u = "abcdef"
    u.replaceSubrange(u.index(u.startIndex, offsetBy: ks[0])..<u.index(u.startIndex, offsetBy: 3), with: "XYZ")
    check("57c", t == "wörlD!" && u == "aXYZdef")
    var a = ["a", "b", "c"]
    a.append(contentsOf: ["d", "e"])
    var nested = [[1], [2]]
    nested.append([3, 4])
    nested.append(contentsOf: [[5]])
    var ns = [0]
    ns.append(contentsOf: 1...3)
    var cs: [Character] = ["x"]
    cs.append(contentsOf: "yz")
    check("57d", a == ["a", "b", "c", "d", "e"] && a.count == 5 && nested.count == 4 && ns == [0, 1, 2, 3] && cs.count == 3)
    var v = [1, 2, 3, 4, 5]
    v.removeSubrange(ks[0]..<3)
    v.replaceSubrange(0..<1, with: [9, 8])
    check("57e", v == [9, 8, 4, 5] && v.startIndex == 0 && v.endIndex == 4 && v.index(after: 2) == 3)
    let r = Ring57(n: 3)
    check("57f", r.index(after: 2) == 0 && r.startIndex == 7)
}

// ===== SECTION 58: Int64 and UInt64 are types of their own; type(of:) =====
// Int64 is not Int and UInt64 is not UInt, though each pair has one width. The
// bounds, bitWidth, the over-shift, magnitude (the UNSIGNED type of the same
// width) and every type(of:) name are swiftc 6.1.2's.
struct P58 { var x = 1 }
func s58() {
    let vs: [Any] = [Int64(5), 5, UInt(3), UInt64(4)]
    check("58a", (vs[0] is Int64) && !(vs[0] is Int) && (vs[1] is Int) && !(vs[1] is Int64))
    check("58b", (vs[2] is UInt) && !(vs[2] is UInt64) && (vs[3] is UInt64) && !(vs[3] is UInt))
    check("58c", (vs[0] as? Int) == nil && (vs[0] as? Int64) == 5 && (vs[3] as? UInt) == nil)
    let ns: [Int64] = [Int64.min, -7]
    let sh = [64, 63]
    check("58d", Int64.bitWidth == 64 && ns[1].bitWidth == 64 && ns[1] >> sh[0] == -1 && ns[1] << sh[0] == 0 && ns[0] >> sh[1] == -1)
    check("58e", "\(ns[0].magnitude)" == "9223372036854775808" && "\(type(of: ns[1].magnitude))" == "UInt64" && Int8(-128).magnitude == 128)
    let us: [UInt64] = [UInt64.max]
    check("58f", Double(us[0]) == 18446744073709551615.0 && us[0] >> UInt64(sh[0]) == 0 && "\(us[0])" == "18446744073709551615")
    var a: Int64 = 10
    a += 5
    check("58g", "\(type(of: a))" == "Int64" && "\(type(of: sh[0]))" == "Int" && "\(type(of: UInt(1)))" == "UInt" && "\(type(of: Int8(1)))" == "Int8")
    check("58h", "\(type(of: 1.5))" == "Double" && "\(type(of: Float(1)))" == "Float" && "\(type(of: "s"))" == "String" && "\(type(of: true))" == "Bool")
    check("58i", "\(type(of: sh))" == "Array<Int>" && "\(type(of: ["k": 1.5]))" == "Dictionary<String, Double>" && "\(type(of: Set([a])))" == "Set<Int64>" && "\(type(of: P58()))" == "P58")
}

// ===== SECTION 59: raw multi-line strings, escaped names, typed throws, ownership =====
// A raw literal honours only the escapes written with its own pound count, and its
// multi-line form lays out like a plain one. Backtick-quoted names are the bare
// names; `throws(E)`, `sending`, `@isolated(any)`, `~Copyable`, `copy` / `consume`
// change nothing a run can see; a for-in annotation types the element literals.
// Every expected value is swiftc 6.1.2's.
enum E59: Error { case bad }
struct NC59: ~Copyable { var v = 4 }
enum Maybe59<T: ~Copyable>: ~Copyable { case none59 }
infix operator %%%% : AdditionPrecedence
func %%%%<T, U>(x: T, y: U) -> Int { return 7 }
func thr59(_ n: Int) throws(E59) -> Int { if n < 0 { throw .bad }; return n * 2 }
func send59(_ s: sending String) -> sending String { return s + "!" }
func iso59(_ f: @isolated(any) () -> Int) -> Int { return 1 }
func conv59(_ f: @convention(c) (Int) -> Int) -> Int { return f(3) }
func `protocol`(`in` x: Int) -> Int { return x + 1 }
func s59() {
    let r1 = #"""
        a "q" \n
          b\#tc
        """#
    check("lex1", r1 == "a \"q\" \\n\n  b\tc")
    let r4 = ####"""
      x"""###y \####n z
      """####
    check("lex2", r4 == "x\"\"\"###y \n z")
    check("lex3", #"p\#nq\r"# == "p\nq\\r" && ###"""##"### == "\"\"##")
    let xs = [1, 2, 3,]
    let d = ["k": 1,]
    check("lex4", xs.count == 3 && d.count == 1)
    var sum = 0.0
    for x: Double in [1, 2] { sum += x / 4 }
    check("lex5", sum == 0.75)
    check("lex6", `protocol`(in: 4) == 5 && 1 %%%% "a" == 7)
    let `default` = 3
    check("lex7", `default` + 1 == 4)
    var caught = false
    do { _ = try thr59(-1) } catch { caught = true }
    check("lex8", caught && (try? thr59(2)) == 4)
    check("lex9", send59("hi") == "hi!" && iso59({ 2 }) == 1)
    let a = 5
    let b = copy a
    let c = consume b
    check("lex10", c == 5)
    let nc = NC59()
    check("lex11", nc.v == 4 && conv59({ $0 + 1 }) == 4)
    var hit = 0
    switch hit {
    case 0:
        ()
    default:
        hit = 2
    }
    check("lex12", hit == 0)
}

// ===== SECTION 60: #if version checks =====
// compiler(...) and swift(...) are decided against swiftc 6.1.2 in its default
// language mode 5.10 (the quoted legacy _compiler_version is always true there),
// and an inactive clause of a version check is not parsed at all - it may hold
// code for another compiler.
func s60() {
    var got = ""
#if compiler(>=6.1)
    got += "a"
#else
    got += "b" this is not Swift at all
#endif
#if !compiler(>=5)
    unparsed junk here
#elseif swift(>=6)
    got += "c"
#else
    got += "d"
#endif
#if swift(<5.10) || false
    got += "e"
#endif
#if !_compiler_version("777.*.7")
    more junk
#elseif _compiler_version(>=6.1)
    got += "f"
#endif
    check("ver1", got == "adf")
}

// ===== SECTION 61: optional-chain assignment =====
// `a?.b = c` writes only when a is not nil, and a nil skips the right-hand side
// too. The assignment's value is `()?`: nil when it was skipped.
class OcNode { var ocv = 0; var ocxs = [1, 2]; var ocd: Double = 0; var next: OcNode? = nil }
struct OcPt { var ocx = 1; var ocy = 2 }
var ocCalls = 0
func ocVal(_ x: Int) -> Int { ocCalls += 1; return x }
func s61() {
    let n: OcNode? = OcNode()
    let none: OcNode? = nil
    n?.ocv = ocVal(5)
    none?.ocv = ocVal(6)
    check("oca1", n!.ocv == 5 && ocCalls == 1)
    n?.ocv += 3
    none?.ocv += ocVal(1)
    check("oca2", n!.ocv == 8 && ocCalls == 1)
    n?.ocxs[1] = 20
    n?.ocd = 2
    check("oca3", n!.ocxs == [1, 20] && "\(n!.ocd)" == "2.0")
    n?.next?.ocv = ocVal(7)
    n!.next = OcNode()
    n?.next?.ocv = 9
    check("oca4", n!.next!.ocv == 9 && ocCalls == 1)
    var arr: [Int]? = [1, 2]
    arr?[0] = 10
    var dict: [String: Int]? = ["a": 1]
    dict?["b"] = 2
    var p: OcPt? = OcPt()
    p?.ocx = 7
    check("oca5", arr! == [10, 2] && dict!["b"] == 2 && p!.ocx == 7 && p!.ocy == 2)
    check("oca6", (none?.ocv = 3) == nil && (n?.ocv = 3) != nil && n!.ocv == 3)
}

// ===== SECTION 62: inout closure parameters =====
// A closure's inout parameter writes back through the caller's `&x`, annotated
// or not; `reduce(into:)` hands its closure the accumulator itself.
func icApply(_ f: (inout Int) -> ()) -> Int { var v = 4; f(&v); return v }
func icApplyA(_ f: (inout [Int]) -> Void) -> [Int] { var a = [7]; f(&a); return a }
func s62() {
    check("icl1", icApply { (x: inout Int) in x += 2 } == 6)
    check("icl2", icApply { (x: inout Int) -> () in if x > 0 { x = 100; return }; x = 1 } == 100)
    check("icl3", icApply { x in x += 3 } == 7 && icApply { $0 *= 5 } == 20)
    check("icl4", icApplyA { xs in xs.append(8) } == [7, 8] && icApplyA { $0.append(9) } == [7, 9])
    var cnt = 0
    let bump = { (c: inout Int, by: Int) in c += by }
    bump(&cnt, 3)
    bump(&cnt, 4)
    var d = 1.5
    let scale = { (a: inout Double, n: Int) in a *= Double(n) }
    scale(&d, 3)
    check("icl5", cnt == 7 && "\(d)" == "4.5")
    check("icl6", [1, 2, 3].reduce(into: 0) { (acc: inout Int, x: Int) in acc += x } == 6
        && [1, 2].reduce(into: [Int]()) { (acc: inout [Int], x: Int) in acc.append(x * 10) } == [10, 20])
}

// ===== SECTION 63: a literal typed by `as` or by a tuple pattern's annotation =====
func s63() {
    check("ctx1", "\([1, 2] as [Double])" == "[1.0, 2.0]" && "\(["a": 1] as [String: Double])" == "[\"a\": 1.0]")
    check("ctx2", "\((1, 2) as (Double, Int))" == "(1.0, 2)" && "\([[1], [2]] as [[Double]])" == "[[1.0], [2.0]]")
    let (a, b): (Double, Int) = (1, 2)
    var (c, d): (Int, Double) = (3, 4)
    d += 1
    check("ctx3", "\(a) \(b)" == "1.0 2" && "\(c) \(d)" == "3 5.0" && d / 2 == 2.5)
    let (x, y): (x: Float, y: String) = (x: 5, y: "s")
    check("ctx4", "\(x)\(y)" == "5.0s")
}

// ===== SECTION 64: key paths, postfix and inout operators, a projected `$x` =====
// A key path literal reads its member chain, as a function (`.map(\.n)`) or
// through `[keyPath:]`. A user operator's inout parameter takes its operand by
// reference, prefix, infix or postfix alike. `$x` in a member body is self.$x.
struct KpIn { var z = 9 }
struct KpP { var name = "a"; var n = 3; var inner = KpIn() }
postfix operator +++
postfix func +++ (x: inout Int) -> Int { let o = x; x += 1; return o }
prefix operator ---
prefix func --- (x: inout Int) -> Int { x -= 1; return x }
postfix operator %%%
postfix func %%% (x: Double) -> Double { return x / 100 }
infix operator <~~ : AssignmentPrecedence
func <~~ (l: inout Int, r: Int) { l = r * 2 }
@propertyWrapper struct Twice64 {
    var v: Int
    var wrappedValue: Int { get { v } set { v = newValue } }
    var projectedValue: Int { v * 2 }
    init(wrappedValue: Int) { v = wrappedValue }
}
struct Proj64 { @Twice64 var w: Int = 4; func both() -> Int { return w + $w } }
func s64() {
    let kp = \KpP.n
    let p = KpP()
    let ps = [KpP(), KpP(name: "b", n: 5)]
    check("kp1", p[keyPath: kp] == 3 && p[keyPath: \.inner.z] == 9 && p[keyPath: \KpP.inner.z] == 9)
    check("kp2", ps.map(\.n) == [3, 5] && ps.map(\KpP.name) == ["a", "b"] && ps.map(\.name.count) == [1, 1])
    check("kp3", [4, 2].map(\.self) == [4, 2] && [[1], [2, 3]].map(\.count) == [1, 2])
    var i = 5
    let j = i+++
    let k = ---i
    check("pop1", i == 5 && j == 5 && k == 5)
    var m = 1
    m <~~ 4
    let pct = 25.0%%%
    check("pop2", m == 8 && "\(pct)" == "0.25" && 50.0%%% + 1 == 1.5)
    check("prj1", Proj64().both() == 12)
}

// ===== SECTION 65: an implicit member that names a static member =====
// `.name` resolves against the expected type, and a static func or property is a
// member as much as an enum case is: `let s: S65 = .make { 41 }` calls S65.make.
struct S65 {
    var n: Int
    static func make(_ f: () -> Int) -> S65 { return S65(n: f()) }
    static func dbl(_ k: Int) -> S65 { return S65(n: k * 2) }
    static var zero: S65 { return S65(n: 0) }
}
extension S65 {
    static func twice(_ k: Int, by f: (Int) -> Int) -> S65 { return S65(n: f(f(k))) }
}
enum D65 {
    case up, down
    static func pick(_ b: Bool) -> D65 { return b ? .up : .down }
}
class K65 {
    var tag: String
    init(_ t: String) { tag = t }
    static func named(_ s: String) -> K65 { return K65(s) }
}
func take65(_ s: S65) -> Int { return s.n }
func s65() {
    let a: S65 = .make { 41 }
    let b: S65 = .dbl(5)
    let z: S65 = .zero
    check("imp1", a.n == 41 && b.n == 10 && z.n == 0)
    let u: S65 = .twice(3) { $0 + 10 }
    check("imp2", u.n == 23 && take65(.dbl(21)) == 42)
    let d: D65 = .pick(false)
    let k: K65 = .named("x")
    check("imp3", d == .down && k.tag == "x")
    let ss: [S65] = [.dbl(1), .zero, .make { 7 }]
    check("imp4", ss.map { $0.n } == [2, 0, 7])
}

// ===== SECTION 66: an implicit member resolved by the declared type =====
// `.init(...)` is the declared type's initializer, and a member name that two types
// (and an enum case) share is the declared type's: Swift resolves `.name` against
// the expected type, which an annotation, a return type or a computed property's
// type gives.
struct P66 {
    var n: Int
    init(n: Int) { self.n = n * 2 }
    static var blank: P66 { .init(n: 0) }
    static func forge(_ k: Int) -> P66 { return P66(n: k + 1) }
    func bumped() -> P66 { return .init(n: n + 1) }
}
struct Q66 {
    var t: String
    static var blank: Q66 { Q66(t: "qz") }
    static func forge(_ k: Int) -> Q66 { Q66(t: "q\(k)") }
}
enum E66 { case blank, forge(Int) }
class R66 {
    var v: Int
    init(v: Int) { self.v = v }
    init() { v = -1 }
    var p: P66 = .init(n: 5)
    var q: Q66? = .forge(3)
}
func q66(_ b: Bool) -> Q66 {
    if b { return .blank }
    return .forge(2)
}
func s66() {
    let p: P66 = .init(n: 3)
    let p2: P66? = .forge(4)
    let q: Q66 = .blank
    check("dty1", p.n == 6 && p2!.n == 10 && q.t == "qz")
    var r: R66 = .init(v: 7)
    let v1 = r.v
    r = .init()
    check("dty2", v1 == 7 && r.v == -1 && r.p.n == 10 && r.q!.t == "q3")
    let e: E66 = .forge(1)
    var eOk = false
    if case .forge(let k) = e { eOk = k == 1 }
    check("dty3", eOk && P66.blank.n == 0 && P66(n: 1).bumped().n == 6)
    let g = { () -> P66 in return .init(n: 4) }
    check("dty4", q66(true).t == "qz" && q66(false).t == "q2" && g().n == 8)
}

// ===== SECTION 67: lazy static stored properties =====
// A static initialiser runs once, at the first access - a write included - so it
// may construct the type it is declared in and read its later members. A subclass
// reaches the one storage of an inherited static, and a static method or computed
// static reads and updates it unqualified.
var log67 = [String]()
func note67(_ s: String, _ v: Int) -> Int { log67.append(s); return v }
struct V67 {
    var n: Int
    static let zero = V67(n: note67("zero", 0))
    static let one = V67(n: zero.n + 1)
    static var hits = note67("hits", 10)
    static var unused = note67("unused", 0)
    static var half: Int { return hits / 2 }
    static func bump() { hits += 4 }
}
enum C67 {
    case r, g
    static let all: [C67] = [.r, .g]
    static var pick = C67.all[1]
}
class B67 { static var tag = note67("tag", 3) }
class D67: B67 {}
func s67() {
    check("lzs1", log67.isEmpty)
    check("lzs2", V67.one.n == 1 && log67 == ["zero"])
    V67.hits = 4
    V67.bump()
    check("lzs3", V67.hits == 8 && V67.half == 4 && log67 == ["zero", "hits"])
    check("lzs4", C67.all.count == 2 && C67.pick == .g)
    check("lzs5", D67.tag == 3 && B67.tag == 3 && log67 == ["zero", "hits", "tag"])
    D67.tag = 8
    check("lzs6", B67.tag == 8)
}

// ===== SECTION 68: implicit members inside a typed value =====
// The expected type reaches a leading-dot member through an array or dictionary
// literal, a tuple, a ternary or ?? branch, a closure's result, a parameter's
// default and a call argument (trailing closures included). Both types declare
// every member, so no single-owner guess can pick one.
struct T68 {
    var n: Int
    init(n: Int) { self.n = n * 2 }
    static let one = T68(n: 1)
    static func make(_ k: Int) -> T68 { return T68(n: k) }
}
struct U68 {
    var s: String
    static let one = U68(s: "u1")
    static func make(_ k: Int) -> U68 { return U68(s: "u\(k)") }
}
struct W68 { var t: T68; var u: U68 }
func take68(_ t: T68, or d: T68 = .one) -> Int { return t.n * 100 + d.n }
func via68(_ f: () -> U68) -> String { return f().s }
func rows68() -> [T68] { return [.make(3), .init(n: 4)] }
func s68() {
    let ts: [T68] = [.one, .make(2), .init(n: 3)]
    let ud: [String: [U68]] = ["k": [.make(5), .one]]
    check("dsub1", ts.map { $0.n } == [2, 4, 6] && ud["k"]!.map { $0.s } == ["u5", "u1"])
    let flag = false
    let u: U68 = flag ? .one : .make(7)
    let none: T68? = nil
    let t: T68 = none ?? .make(8)
    let pr: (T68, U68) = (.one, .make(9))
    check("dsub2", u.s == "u7" && t.n == 16 && pr.0.n == 2 && pr.1.s == "u9")
    let f: () -> T68 = { .make(8) }
    let g: (Int) -> U68 = { k in .make(k + 1) }
    check("dsub3", f().n == 16 && g(1).s == "u2" && rows68().map { $0.n } == [6, 8])
    let w = W68(t: .make(1), u: .one)
    check("dsub4", take68(.make(3)) == 602 && take68(.one, or: .init(n: 5)) == 210
                   && via68 { .make(4) } == "u4" && w.t.n == 2 && w.u.s == "u1")
    let sw: U68 = {
        if flag { return .one }
        return .make(6)
    }()
    let pick: (Bool) -> T68 = { b in
        if b { return .one }
        return .make(5)
    }
    check("dsub5", sw.s == "u6" && pick(true).n == 2 && pick(false).n == 10)
}

// ===== SECTION 69: a program property named like a Collection member =====
// A type may declare its own count / first / last / isEmpty, stored, computed or
// static; a value of that type reads its own, and a collection still answers.
struct Bag69 {
    var count = 2
    var first = "f"
    var isEmpty: Bool { return count == 0 }
    mutating func add() { count += 1 }
}
class Box69 { static var count = 0; var last = 9; init() { Box69.count += 1 } }
enum E69 { case a, b; var count: Int { return self == .a ? 1 : 2 } }
func s69() {
    var b = Bag69()
    b.add()
    check("cnt1", b.count == 3 && b.first == "f" && !b.isEmpty)
    let x = Box69()
    _ = Box69()
    check("cnt2", Box69.count == 2 && x.last == 9 && E69.b.count == 2)
    let xs = [Bag69(), Bag69()]
    check("cnt3", xs.count == 2 && xs.map { $0.count } == [2, 2] && "héllo".count == 5 && [4, 5].last! == 5)
}

// ===== SECTION 70: Self as a value =====
// Inside a type, Self names the type of the running member's receiver: a static
// member, an instance member, a mutating write through it, a construction, a
// protocol extension (the conforming type) and an enum.
struct P70 {
    static var num = 5
    var v = 1
    static func b() -> Int { Self.num }
    func c() -> Int { Self.num + v }
    static func make() -> Self { Self(v: 3) }
    mutating func bump() { Self.num += 1 }
}
class Q70 {
    class var k: Int { return 7 }
    class func f() -> Int { return Self.k + 1 }
    func g() -> Int { return Self.k * 2 }
    required init() {}
}
class R70: Q70 {
    override class var k: Int { return 10 }
}
enum E70 {
    case a, b
    static let label = "E!"
    func lab() -> String { return Self.label }
    static func first() -> Self { return Self.a }
}
protocol Named70 { static var tag: String { get } }
extension Named70 { func tagged() -> String { return Self.tag } }
struct N70a: Named70 { static var tag = "n1" }
struct N70b: Named70 { static var tag = "n2" }
func s70() {
    check("slf1", P70.b() == 5 && P70().c() == 6 && P70.make().c() == 8)
    var p = P70()
    p.bump()
    check("slf2", P70.num == 6 && Q70.f() == 8 && Q70().g() == 14 && R70().g() == 20)
    check("slf3", E70.b.lab() == "E!" && E70.first() == .a && N70a().tagged() == "n1" && N70b().tagged() == "n2")
}

// ===== SECTION 71: the expected type through ==, a generic result, a receiver =====
// Both types declare every static member, so no single-owner guess can pick one:
// an == operand gives the other operand its type, a generic function returning
// its own parameter passes the slot's type to that parameter's arguments, and a
// collection method's element argument takes the receiver's element type.
struct T71: Equatable {
    var v: Int
    static let one = T71(v: 1)
    static let zero = T71(v: 0)
    static func make(_ n: Int) -> T71 { return T71(v: n) }
}
struct U71: Equatable {
    var w: Int
    static let one = U71(w: 11)
    static let zero = U71(w: 10)
    static func make(_ n: Int) -> U71 { return U71(w: n * 100) }
}
enum E71 { case one, two; static let best = E71.two }
enum F71 { case one, three; static let best = F71.three }
func gen71<X>(_ x: X) -> X { return x }
func pick71<X>(_ a: X, _ b: X, first: Bool) -> X { return first ? a : b }
func opt71<X: Equatable>(_ x: X) -> X? { return x }
func head71(_ xs: [U71]) -> Bool { return xs[0] == .one }
struct H71 { var items: [T71] = [] }
func s71() {
    let tj: [T71] = [T71.one, T71.zero]
    let ej: [E71] = [.one, .two]
    check("xeq1", tj[0] == .one && .one == tj[0] && tj[1] != .one && head71([.one]) && !head71([.zero]))
    check("xeq2", ej[1] == .best && .best == ej[1] && ej[0] != .best)
    let g: T71 = gen71(.make(5))
    let h: U71 = gen71(.make(5))
    let p: U71 = pick71(.make(2), .one, first: true)
    let o: T71? = opt71(.make(4))
    check("xeq3", g.v == 5 && h.w == 500 && p.w == 200 && o!.v == 4)
    var arr: [T71] = []
    arr.append(.zero)
    arr.insert(.make(7), at: 0)
    arr.append(contentsOf: [.one])
    var hd = H71()
    hd.items.append(.make(6))
    check("xeq4", arr.map { $0.v } == [7, 0, 1] && arr.contains(.one) && arr.firstIndex(of: .zero)! == 1 && hd.items[0].v == 6)
}

// ===== SECTION 72: a struct's synthesized == =====
// An Equatable struct with no == of its own compares its stored properties, nested
// structs and arrays included; contains(_:) uses that ==.
struct In72: Equatable { var a: Int; var s: String }
struct Out72: Equatable { var i: In72; var xs: [Int]; var d: Double }
struct Odd72: Equatable {
    var a: Int
    var b: Int
    static func == (l: Odd72, r: Odd72) -> Bool { return l.a == r.a }
}
func s72() {
    let o1 = Out72(i: In72(a: 1, s: "x"), xs: [1, 2], d: 1.5)
    let o2 = Out72(i: In72(a: 1, s: "x"), xs: [1, 2], d: 1.5)
    let o3 = Out72(i: In72(a: 1, s: "y"), xs: [1, 2], d: 1.5)
    check("seq1", o1 == o2 && !(o1 == o3) && o1 != o3 && Odd72(a: 1, b: 2) == Odd72(a: 1, b: 3))
    var m = In72(a: 5, s: "q")
    let m0 = m
    m.a = 6
    let ch1 = m == m0
    m.a = 5
    check("seq2", !ch1 && m == m0 && [In72(a: 1, s: "a")] == [In72(a: 1, s: "a")])
    let ins = [In72(a: 1, s: "a"), In72(a: 2, s: "b")]
    let xs = [1.5, 2.5]
    let y = 2.5
    check("seq3", ins.contains(In72(a: 2, s: "b")) && !ins.contains(In72(a: 2, s: "c"))
                  && xs.contains(y) && [[1], [2]].contains([2]) && [Int8(3)].contains(Int8(3)))
}

// ===== SECTION 73: static members reached through the type value =====
// A static or class method takes the type value it was reached through as self,
// so an inherited `class func` sees the subclass; an unqualified call and super.m()
// inside one reach the statics. type(of:), T.self, a metatype parameter and an
// unapplied Type.m are all that same type value, and a builtin's extension statics
// are reached from Int itself.
protocol Mk73 { static func mk() -> Int }
extension Mk73 { static func twice() -> Int { return mk() * 2 } }
class Q73: Mk73 {
    static var k = 10
    class var ck: Int { return 1 }
    static func s() -> Int { return 1 }
    class func f() -> Int { return self.k + ck }
    class func g() -> Int { return s() + k }
    class func who() -> String { return "Q" }
    class func mk() -> Int { return 7 }
    class func viaClosure() -> [Int] { return [1, 2].map { $0 + mk() } }
    static subscript(i: Int) -> Int { return i * 10 }
    static func ov(_ a: Int) -> String { return "int" }
    static func ov(_ a: String) -> String { return "str" }
    func viaSelf() -> Int { return Self.s() + type(of: self).ck }
}
class R73: Q73 {
    override class var ck: Int { return 2 }
    override class func who() -> String { return "R" }
    override class func mk() -> Int { return 70 + super.mk() }
    static func t() -> Int { return s() + 100 }
}
struct V73 { static func hi() -> String { return "static" }; func hi() -> String { return "inst" } }
enum E73: Mk73 { case a; static func mk() -> Int { return 5 } }
protocol Z73 { static var zz: Self { get } }
extension Int: Z73 { static var zz: Int { return 99 } }
extension Int { static func dbl73(_ x: Int) -> Int { return x * 2 } }
func s73() {
    check("stc1", R73.s() == 1 && R73.f() == 12 && Q73.f() == 11 && R73.g() == 11 && R73.t() == 101)
    check("stc2", R73.who() == "R" && R73.mk() == 77 && R73.ov("x") == "str" && R73.ov(1) == "int")
    let x: Q73 = R73()
    check("stc3", type(of: x).who() == "R" && type(of: x).ck == 2 && x.viaSelf() == 3 && Q73().viaSelf() == 2)
    let mt: Q73.Type = R73.self
    let ms: [Mk73.Type] = [Q73.self, R73.self, E73.self]
    var tot = 0
    for m in ms { tot += m.mk() }
    check("stc4", mt.who() == "R" && mt.f() == 12 && tot == 89 && R73.twice() == 154 && E73.twice() == 10)
    let fn = V73.hi
    check("stc5", fn() == "static" && V73().hi() == "inst" && Int.zz == 99 && Int.dbl73(4) == 8)
    let made = [1, 2].map(Int.dbl73)
    check("stc6", R73[4] == 40 && R73.viaClosure() == [78, 79] && made == [2, 4])
}

// ===== SECTION 74: a generic parameter as a type value =====
// `X.zz74` inside `func f<X: P>(_ t: X.Type)` reads the static of whatever type t
// holds, and inside `func f<X: P>(_ v: X)` that of v's type, a builtin's included.
protocol Z74 { static var zz74: Self { get } }
extension Int: Z74 { static var zz74: Int { return 99 } }
struct W74: Z74 { var v: Int; static var zz74: W74 { return W74(v: 98) } }
func zeroOf74<X: Z74>(_ t: X.Type) -> X { return X.zz74 }
func both74<X: Z74>(_ a: X.Type, _ b: X.Type) -> [X] { return [X.zz74, b.zz74] }
func ofValue74<X: Z74>(_ v: X) -> X { return X.zz74 }
func s74() {
    check("tpv1", zeroOf74(Int.self) == 99 && zeroOf74(W74.self).v == 98)
    check("tpv2", both74(Int.self, Int.self) == [99, 99] && both74(W74.self, W74.self)[1].v == 98)
    check("tpv3", ofValue74(5) == 99 && ofValue74(W74(v: 1)).v == 98)
}

// ===== SECTION 75: an array literal typed as a Set by its context =====
// The element type of `[ss]`, a Set<Int> parameter and an [Set<Int>] annotation
// make an array literal a Set, so it compares as one.
func count75(_ s: Set<Int>) -> Int { return s.count }
func s75() {
    let ss: Set<Int> = [1, 2]
    check("sbc1", [ss].contains([1]) == false && [ss].contains([2, 1]) && ss == [2, 1] && [2, 1, 1] == ss)
    check("sbc2", ([ss].firstIndex(of: [1, 2]) ?? -1) == 0 && count75([1, 1, 2]) == 2)
    let arr: [Set<Int>] = [[1], [2, 3]]
    check("sbc3", arr.contains([3, 2]) && !arr.contains([4]) && arr[1].count == 2)
}

// ===== SECTION 76: a field write on a type that has a subscript =====
// `m.x += 2` and `T.n += 1` write the PROPERTY, never the type's subscript.
struct M76 {
    var x = 1
    var store = [0, 0]
    subscript(i: Int) -> Int {
        get { return store[i] }
        set { store[i] = newValue }
    }
}
class C76 {
    static var n = 0
    static subscript(i: Int) -> Int { return i * 10 }
    static func bump() -> Int { Self.n += 1; n += 1; return n }
}
func s76() {
    var m = M76()
    m.x = 5
    m.x += 2
    m[1] = 9
    m[1] += 1
    check("fws1", m.x == 7 && m.store == [0, 10])
    C76.n += 3
    C76.n = C76.n + 1
    check("fws2", C76.n == 4 && C76.bump() == 6 && C76[2] == 20)
}

// ===== SECTION 77: a Dictionary keyed by a Set, an Array or a struct =====
// A Hashable key compares BY VALUE: Set([1, 2]) and Set([2, 1]) are one key, and so
// are two equal arrays or structs. The declared key type makes an array literal
// key a Set.
struct K77: Hashable { var x: Int; var y: Int }
enum E77: Hashable { case a(Int), b }
func s77() {
    var d: [Set<Int>: String] = [Set([1]): "one"]
    d[Set([2, 3])] = "two"
    d[Set([3, 2])] = "TWO"
    check("vkd1", d[Set([1])] == "one" && d.count == 2 && d[Set([3, 2])] == "TWO" && d[[2, 3]] == "TWO")
    d[Set([1])] = nil
    check("vkd2", d.count == 1 && d.keys.contains(Set([2, 3])))
    var p: [K77: Int] = [K77(x: 1, y: 2): 3]
    p[K77(x: 1, y: 2)] = 4
    var e: [E77: Int] = [.a(1): 1, .b: 2]
    e[.a(1)] = 5
    var a: [[Int]: Int] = [[1, 2]: 1]
    a[[1, 2]] = 2
    a[[2, 1]] = 3
    check("vkd3", p.count == 1 && p[K77(x: 1, y: 2)] == 4 && e.count == 2 && e[.a(1)] == 5
                  && e[.a(2)] == nil && a.count == 2 && a[[1, 2]] == 2)
    var sk: [Set<String>: Int] = [["a", "b"]: 1]
    sk[["b", "a"]] = 2
    sk[["c"]] = 3
    check("vkd4", sk.count == 2 && sk[["a", "b"]] == 2 && sk[Set(["c"])] == 3)
}

// ===== SECTION 78: a builtin type as a value =====
// `String.self`, `type(of: "x")` and a generic parameter bound from a String are
// one type value, which prints as its name and compares equal to itself; Bool(x)
// is the failable Bool(String).
func name78<T>(_ v: T) -> String { return "\(T.self)" }
func meta78<T>(_ t: T.Type) -> String { return String(describing: t) }
func s78() {
    check("btv1", "\(String.self)" == "String" && "\(Bool.self)" == "Bool" && "\(Int8.self)" == "Int8"
                  && "\(Double.self)" == "Double" && "\(Character.self)" == "Character")
    check("btv2", name78("s") == "String" && name78(1) == "Int" && name78(true) == "Bool"
                  && name78(2.5) == "Double" && name78([1]) == "Array<Int>" && meta78(UInt8.self) == "UInt8")
    let t: Any.Type = String.self
    check("btv3", type(of: "x") == String.self && type(of: 1) == Int.self && type(of: Int8(1)) != Int.self
                  && Int.self != Int64.self && t == String.self && "\(type(of: true))" == "Bool")
    let bt = "true"
    check("btv4", Bool(bt) == true && Bool("false") == false && Bool("TRUE") == nil && Bool(" true") == nil
                  && Bool(false) == false)
}

// ===== SECTION 79: the Dictionary API =====
// The defaulting subscript (a read leaves the dictionary alone; `+=` and a mutating
// method store the default, typed as the value type), removeValue/updateValue
// (answering the old value as an Optional), removeAll, merge/merging, the three
// initializers from a sequence, debugDescription, and `[K: V]` / `[T]` written as a
// type: called, and as a value. Every value is swiftc 6.1.2's.
func s79() {
    let ws = ["apple", "avocado", "banana", "blue", "cherry"]
    var c = [Character: Int]()
    for w in ws { c[w.first!, default: 0] += 1 }
    var g = [String: [Int]]()
    g["x", default: []].append(3)
    g["x", default: []].append(4)
    var ss: [Int: Set<Int>] = [:]
    ss[1, default: []].insert(5)
    ss[1, default: []].insert(5)
    var dd: [String: Double] = ["a": 1.5]
    dd["c", default: 0] += 2
    var nest = [String: [String: Int]]()
    nest["a", default: [:]]["y", default: 0] += 5
    check("79a", c["a"]! == 2 && c["c"]! == 1 && c["z", default: 7] == 7 && c.count == 3 && g["x"]! == [3, 4]
                 && ss[1]!.count == 1 && "\(dd["c"]!) \(dd["q", default: 7])" == "2.0 7.0" && nest["a"]!["y"]! == 5)
    var d = ["a": 1, "b": 2]
    let r = d.removeValue(forKey: "a")
    let u = d.updateValue(9, forKey: "b")
    let u2 = d.updateValue(7, forKey: "c")
    check("79b", "\(r as Any) \(u as Any) \(u2 as Any)" == "Optional(1) Optional(2) nil" && d.count == 2
                 && d.removeValue(forKey: "q") == nil && d["b"]! == 9 && d.removeValue(forKey: "c")! == 7)
    let o = dd.updateValue(9, forKey: "b")
    dd.merge(["e": 3]) { a, _ in a }
    check("79c", o == nil && "\(dd["b"]!) \(dd["e"]!)" == "9.0 3.0")
    var m = ["a": 1, "b": 2]
    m.merge(["b": 10, "c": 3]) { old, new in old + new }
    let mg = m.merging([("a", 100), ("z", 0)], uniquingKeysWith: { $1 })
    var e = m
    e.removeAll()
    e.reserveCapacity(4)
    check("79d", m["b"]! == 12 && m["c"]! == 3 && mg["a"]! == 100 && mg.count == 4 && m["a"]! == 1
                 && e.isEmpty && m.count == 3 && "\(e)" == "[:]")
    let gr = Dictionary(grouping: ws, by: { $0.first! })
    let g2 = Dictionary(grouping: 1...6) { $0 % 3 }
    let uq = Dictionary(uniqueKeysWithValues: [("a", 1), ("b", 2)])
    let uk = Dictionary([("a", 1), ("a", 5), ("b", 2)], uniquingKeysWith: +)
    let sg = [String: Int](uniqueKeysWithValues: [("z", 26)])
    check("79e", gr["a"]! == ["apple", "avocado"] && gr["c"]! == ["cherry"] && gr.count == 3 && g2[0]! == [3, 6]
                 && uq["b"]! == 2 && uq.count == 2 && uk["a"]! == 6 && uk["b"]! == 2 && "\(sg)" == "[\"z\": 26]"
                 && [Int](repeating: 0, count: 2) == [0, 0])
    check("79f", "\([Int].self) \([String: Int].self)" == "Array<Int> Dictionary<String, Int>"
                 && "\([[Int]].self) \([Int?].self)" == "Array<Array<Int>> Array<Optional<Int>>"
                 && type(of: [1]) == [Int].self && ["k": 1].debugDescription == "[\"k\": 1]"
                 && "s".debugDescription == "\"s\"" && Set([7]).debugDescription == "Set([7])")
}

// ===== SECTION 80: reduce(into:) =====
// The body mutates its inout accumulator and answers Void: an assignment as the
// closure's last statement is Void too, never the assigned value. swiftc 6.1.2.
func s80() {
    let xs = [3, 1, 2]
    let a = xs.reduce(into: 0) { $0 += $1 }
    let b = xs.reduce(into: 0) { acc, x in acc += x; _ = acc }
    let s = xs.reduce(into: "") { s, x in s += String(x) }
    let d = xs.reduce(into: [String: Int]()) { m, x in m["k\(x)"] = x }
    let h = xs.reduce(into: 10) { acc, x in if x > 1 { acc -= x } }
    let byLen = ["aa", "b", "cc"].reduce(into: [Int: [String]]()) { $0[$1.count, default: []].append($1) }
    check("80a", a == 6 && b == 6 && s == "312" && d.count == 3 && d["k3"]! == 3 && h == 5
                 && byLen[2]! == ["aa", "cc"] && xs.reduce(0, +) == 6)
}

// ===== SECTION 81: the standard library's free functions =====
// zip, stride, repeatElement, swap, numericCast, min/max of three or more, abs
// at a sized width, the trapping functions on their passing path, a closure taking an
// n-tuple apart, nested tuple patterns, type(of:) of an empty collection, and
// strtod's NaN spellings. Every value read off swiftc 6.1.2.
func s81() {
    let a = [3, 1, 4, 1, 5]
    let b = ["x", "y", "z"]
    // zip stops at the shorter side; a PartialRangeFrom is bounded by the other.
    let ab = Array(zip(a, b))
    check("81a", ab.count == 3 && ab[1].0 == 1 && ab[1].1 == "y" && "\(ab[2])" == "(4, \"z\")")
    let byKey = Dictionary(uniqueKeysWithValues: zip(b, 1...))
    var walked = ""
    for (i, c) in zip(0..., "hey") { walked += "\(i)\(c)" }
    check("81b", byKey["y"]! == 2 && byKey.count == 3 && walked == "0h1e2y"
                 && Array(zip([1], [Int]())).isEmpty && Array(zip(Set([7]), [8]))[0].1 == 8)
    // A closure of n parameters takes an n-tuple argument apart.
    let joined = zip(a, b).map { "\($0)\($1)" }.joined(separator: ",")
    let sums = [(1, 2), (3, 4)].map { $0 + $1 }
    let named = [(1, 2), (3, 4)].map { x, y in x * y }
    let kv = ["k": 1].map { k, v in k + String(v) }
    check("81c", joined == "3x,1y,4z" && sums == [3, 7] && named == [2, 12] && kv == ["k1"]
                 && zip([1, 2], [3, 4]).map(+) == [4, 6] && a.enumerated().map { $0 * $1 } == [0, 1, 8, 3, 20])
    // Nested tuple patterns, in a for-in, a declaration and an assignment.
    var nest = ""
    for (i, (x, y)) in zip(a, b).enumerated() { nest += "\(i)\(x)\(y)" }
    let (p, (q, r)) = (1, (2, "s"))
    var (m, (n, _)) = (1, (2, 3))
    (m, (n, _)) = (n, (m, 9))
    check("81d", nest == "03x11y24z" && p == 1 && q == 2 && r == "s" && m == 2 && n == 1)
    // stride: an Int one adds the step; a floating-point one computes each element
    // as start + i*step in one rounding, so 0.6000000000000001 where rounding the
    // product first gives 0.6, and a Float one rounds at Float width (1.4000001).
    check("81e", Array(stride(from: 0, to: 10, by: 3)) == [0, 3, 6, 9]
                 && Array(stride(from: 10, through: 0, by: -5)) == [10, 5, 0]
                 && Array(stride(from: 3, to: 3, by: 1)).isEmpty && Array(stride(from: 1, through: 1, by: -1)) == [1])
    check("81f", "\(Array(stride(from: 0.1, to: 0.75, by: 0.1)))" == "[0.1, 0.2, 0.30000000000000004, 0.4, 0.5, 0.6000000000000001, 0.7000000000000001]"
                 && "\(Array(stride(from: 0.5, through: -0.1, by: -0.2)))" == "[0.5, 0.3, 0.09999999999999998]"
                 && "\(Array(stride(from: Float(0.5), to: 2, by: 0.3)))" == "[0.5, 0.8, 1.1, 1.4000001, 1.7]")
    // min/max of three or more fold the two-operand rule; abs keeps the width.
    let i8: Int8 = -3
    check("81g", min(4, 2, 9) == 2 && max(4, 2, 9, 1) == 9 && min(1.5, 0.5, 2.5) == 0.5
                 && abs(i8) == 3 && "\(type(of: abs(i8)))" == "Int8" && abs(-3000000000) == 3000000000)
    var s1 = "a", s2 = "b"
    swap(&s1, &s2)
    precondition(s1 == "b", "swapped")
    assert(s2 == "a")
    check("81h", s1 + s2 == "ba" && Array(repeatElement("ab", count: 3)) == ["ab", "ab", "ab"])
    // type(of:) of an empty collection names its static type.
    let es: Set<Int> = []
    let ed: [Int: Bool] = [:]
    var ev = Set<String>()
    check("81i", "\(type(of: Set<Int>()))" == "Set<Int>" && "\(type(of: [Int]()))" == "Array<Int>"
                 && "\(type(of: [String: Int]()))" == "Dictionary<String, Int>" && "\(type(of: es))" == "Set<Int>"
                 && "\(type(of: ed))" == "Dictionary<Int, Bool>" && "\(type(of: ev))" == "Set<String>")
    ev = ["q", "q"]
    check("81j", ev.count == 1)
    // strtod's NaN spellings parse (their sign and payload are not kept here).
    check("81k", Double("nan(0x1)")!.isNaN && Double("-snan")!.isNaN && Double("nan(a b)")!.isNaN
                 && Double("snan()") == nil && Double("nan(())") == nil && Float("nan(1)")!.isNaN)
    // numericCast converts to the integer type its context asks for.
    let u8s: [UInt8] = [250]
    let widened: Int = numericCast(u8s[0])
    check("81l", widened + 10 == 260 && "\(type(of: widened))" == "Int")
}

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
    s33() // SECTION-CALL 33
    s34() // SECTION-CALL 34
    s35() // SECTION-CALL 35
    s36() // SECTION-CALL 36
    s37() // SECTION-CALL 37
    s38() // SECTION-CALL 38
    s39() // SECTION-CALL 39
    s40() // SECTION-CALL 40
    s41() // SECTION-CALL 41
    s42() // SECTION-CALL 42
    s43() // SECTION-CALL 43
    s44() // SECTION-CALL 44
    s45() // SECTION-CALL 45
    s46() // SECTION-CALL 46
    s47() // SECTION-CALL 47
    s48() // SECTION-CALL 48
    s49() // SECTION-CALL 49
    s50() // SECTION-CALL 50
    s51() // SECTION-CALL 51
    s52() // SECTION-CALL 52
    s53() // SECTION-CALL 53
    s54() // SECTION-CALL 54
    s55() // SECTION-CALL 55
    s56() // SECTION-CALL 56
    s57() // SECTION-CALL 57
    s58() // SECTION-CALL 58
    s59() // SECTION-CALL 59
    s60() // SECTION-CALL 60
    s61() // SECTION-CALL 61
    s62() // SECTION-CALL 62
    s63() // SECTION-CALL 63
    s64() // SECTION-CALL 64
    s65() // SECTION-CALL 65
    s66() // SECTION-CALL 66
    s67() // SECTION-CALL 67
    s68() // SECTION-CALL 68
    s69() // SECTION-CALL 69
    s70() // SECTION-CALL 70
    s71() // SECTION-CALL 71
    s72() // SECTION-CALL 72
    s73() // SECTION-CALL 73
    s74() // SECTION-CALL 74
    s75() // SECTION-CALL 75
    s76() // SECTION-CALL 76
    s77() // SECTION-CALL 77
    s78() // SECTION-CALL 78
    s79() // SECTION-CALL 79
    s80() // SECTION-CALL 80
    s81() // SECTION-CALL 81
    print("full: \(checks) checks, \(fails) failures")
}

main()
exit(fails)
