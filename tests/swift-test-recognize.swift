// Swift recognition test: real-world syntax that the widened grammar PARSES but
// does not compile. Every construct below is accepted and routed to
// notImplemented, so a normal run aborts at the first one (nonzero exit) while a
// run with -warn-unsupported warns on each and still succeeds (exit 0). The
// compiler's -q output must be byte-identical between the goja and -frozen
// engines. This file is a SHOULD-FAIL by default.

import Foundation
import class UIKit.UIView

typealias Meters = Double

// A protocol and an extension are accepted structurally (not implemented).
protocol Shape {
    var area: Double { get }
    func describe() -> String
}

extension Shape {
    func describe() -> String {
        return "a shape"
    }
}

// An enum with raw / associated values (not implemented).
enum Token {
    case number(Int)
    case word(String)
    case eof
}

// Attributes on declarations and parameters are accepted and ignored, so the
// underlying declarations still parse.
@objc
@available(iOS 13.0, *)
class Cache {
    var count: Int = 0

    deinit {
        count = 0
    }

    subscript(index: Int) -> Int {
        get { return index }
        set { count = newValue }
    }

    @discardableResult
    func run(_ work: @escaping () -> Void) -> Int {
        work()
        return count
    }
}

// A generic function with a where clause genuinely parses (constraints ignored).
func maxOf<T>(_ a: T, _ b: T) -> T where T: Comparable {
    return a > b ? a : b
}

// try / try? / await as expression prefixes (outside any do/catch block).
func fetch(_ name: String) async throws -> String {
    let raw = try await read(name)
    let maybe = try? clean(raw)
    return maybe ?? name
}

// Optional binding, pattern matching and compound conditions in if / guard / while.
func classify(_ token: Token?, _ limit: Int) -> String {
    guard let t = token else {
        return "none"
    }
    if let n = t as? Int, n > limit {
        return "big"
    }
    if case .eof = t {
        return "end"
    }
    while let next = pending.first, next < limit {
        pending.removeFirst()
    }
    return "other"
}

// do / catch / defer / throw / try are recognized.
enum LoadError: Error {
    case missing
}

func load(_ name: String) -> String {
    defer {
        print("done loading")
    }
    do {
        let raw = try read(name)
        let trimmed = try? clean(raw)
        return trimmed ?? "empty"
    } catch LoadError.missing {
        return "missing"
    } catch {
        throw LoadError.missing
    }
}

// Casts (as / as? / as!), the is check, tuples, floating-point literals and
// closure shorthand arguments ($0).
func measure(_ values: [Int]) -> Double {
    let pi = 3.14159
    let origin = (x: 0, y: 0)
    let anything = values as [Any]
    let doubled = values.map { $0 * 2 }
    if anything is [Int] {
        let forced = anything as! [Int]
        return pi * Double(forced.count + doubled.count + origin.x)
    }
    return pi
}

// A switch with a fallthrough and a nested helper function.
func rank(_ n: Int) -> Int {
    func bonus(_ x: Int) -> Int {
        return x + 1
    }
    var score = 0
    switch n {
    case 0:
        score = 1
        fallthrough
    case 1:
        score += bonus(n)
    default:
        score = 0
    }
    return score
}

// Top level stays benign so a -warn-unsupported run exits 0.
let cache = Cache()
print("swift recognition test compiled")
