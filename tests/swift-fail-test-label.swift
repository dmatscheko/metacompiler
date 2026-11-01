// Swift break naming a label that does not enclose it. swiftc rejects it at
// compile time:
//   cannot find label 'outer' in scope
// The label is on the loop that CALLS f, which is not in f's scope, so this file
// SHOULD fail in both halves before it prints anything.

func f() {
    break outer
}

print("compiled")
outer: for _ in 0..<2 { f() }
