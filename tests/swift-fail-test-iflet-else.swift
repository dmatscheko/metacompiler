// An if's optional binding used in its else branch. swiftc rejects it:
//   swift-fail-test-iflet-else.swift:17:11: error: cannot find 'v' in scope
// The binding is in scope in the condition and the then block only, so this file
// SHOULD fail in both halves before it prints anything. Everything before that
// line uses a binding inside its scope, or an outer name the else branch sees.

func pick(_ o: Int?, _ w: Int?) -> Int {
    let base = 1
    if let v = o, v > 0 { return v + base } else if let base = w { return base }
    return base
}

print("compiled")
print(pick(nil, 4))
let seed: Int? = nil
if let v = seed { print(v) } else {
    print(v)
}
