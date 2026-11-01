// Swift continue naming the label of an if. swiftc rejects it at compile time:
//   'continue' cannot be used with if statements
// The loop inside the if does not make the label a loop's, so this file SHOULD
// fail in both halves before it prints anything.

print("compiled")
let go = true
check: if go {
    for _ in 0..<2 { continue check }
}
