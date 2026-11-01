// Swift break outside a loop or switch. swiftc rejects it at compile time:
//   'break' is only allowed inside a loop, if, do, or switch
// A function body is not inside the loop that calls it, so this file SHOULD fail
// in both halves before it prints anything.

func f() {
    break
}

print("compiled")
for _ in 0..<2 { f() }
