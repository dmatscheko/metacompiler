// Swift name used outside its scope. swiftc rejects it even when never reached:
//   swift-fail-test-scope.swift:22:11: error: cannot find 'doubled' in scope
// doubled is declared, but only in the block of the loop inside spread, so this
// file SHOULD fail in both halves before it prints anything. Everything before
// that line is a local name used inside its scope: parameters, a guard binding,
// loop variables and a closure parameter.

func spread(_ xs: [Int], _ o: Int?) -> [Int] {
    guard let base = o else { return [] }
    var out: [Int] = []
    for (i, x) in xs.enumerated() {
        let doubled = x * 2
        out.append({ (k: Int) in k + doubled + i + base }(1))
    }
    return out
}

print("compiled")
print(spread([1, 2], 10))
let skip = false
if skip {
    print(doubled)
}
