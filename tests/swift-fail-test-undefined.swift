// Swift undefined name. swiftc rejects a reference no declaration binds, even one
// that is never reached:
//   swift-fail-test-undefined.swift:32:11: error: cannot find 'zz' in scope
// so this file SHOULD fail in both halves before it prints anything. Everything
// before that line is a name swiftc resolves: a function declared further down,
// parameters, closure parameters, pattern bindings, accessor names, a member
// reached through implicit self and the implicit error of a catch.

struct Counter {
    var count = 0 {
        didSet { last = oldValue }
    }
    var last = 0
    mutating func add(_ step: Int) { count += step }
}

print("compiled")
var c = Counter()
c.add(later(2))
let total = [1, 2, 3].reduce(0) { acc, v in acc + v }
if let first = [total].first, case let n? = Optional(first) {
    print(n, c.count, c.last)
}
enum Oops: Error { case bad }
do {
    throw Oops.bad
} catch {
    print(error)
}
let skip = false
if skip {
    print(zz)
}

func later(_ n: Int) -> Int { n * 2 }
