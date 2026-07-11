// Swift subset self test: the completion features.
//
// Exercises switch (value cases, comma value lists, default, enum-value cases,
// multi-statement bodies, a break that ends a case, and switch nested in a loop)
// and guard ... else { ... } (early return, early continue, compound conditions,
// return of a value). Top level code runs in source order, counts failed checks
// and ends with exit(failures), so the run exits 0 exactly when every check
// passes. The interpreter and the LLVM-IR compiler must agree on every result.

var fails = 0

func check(_ name: String, _ got: Int, _ want: Int) {
    if got != want {
        print("FAIL \(name): got \(got) want \(want)")
        fails += 1
    }
}
func checkS(_ name: String, _ got: String, _ want: String) {
    if got != want {
        print("FAIL \(name): got \(got) want \(want)")
        fails += 1
    }
}

enum Direction {
    case north, south
    case east
    case west
}

// switch over an Int: a single value case, a comma value list, and default.
func size(_ n: Int) -> String {
    switch n {
    case 0:
        return "zero"
    case 1, 2, 3:
        return "small"
    case 4, 5, 6, 7, 8, 9:
        return "medium"
    default:
        return "big"
    }
}

// switch over a String, with a multi-statement case body.
func vowelScore(_ s: String) -> Int {
    var score = 0
    switch s {
    case "a", "e", "i", "o", "u":
        score = 10
        score += 1
    case "y":
        score = 5
    default:
        score = 0
    }
    return score
}

// switch over enum constants (compared by value).
func degrees(_ d: String) -> Int {
    switch d {
    case Direction.north:
        return 0
    case Direction.east:
        return 90
    case Direction.south:
        return 180
    case Direction.west:
        return 270
    default:
        return -1
    }
}

// A bare break ends the matched case without falling through.
func breakInCase(_ n: Int) -> Int {
    var out = -1
    switch n {
    case 1:
        out = 100
        break
    case 2:
        out = 200
    default:
        out = 999
    }
    return out
}

// switch nested inside a for-in over an array: classify and tally.
func tally(_ xs: [Int]) -> Int {
    var acc = 0
    for x in xs {
        switch x % 3 {
        case 0:
            acc += 100
        case 1:
            acc += 10
        default:
            acc += 1
        }
    }
    return acc
}

// guard with an early return on the failing path.
func reciprocalTimes(_ a: Int, _ b: Int) -> Int {
    guard b != 0 else {
        return -1
    }
    return a / b
}

// guard with a compound condition.
func inRange(_ n: Int, _ lo: Int, _ hi: Int) -> Int {
    guard n >= lo && n <= hi else {
        return 0
    }
    return 1
}

// guard whose else path continues the enclosing loop (skip odd values).
func sumEvens(_ xs: [Int]) -> Int {
    var total = 0
    for x in xs {
        guard x % 2 == 0 else {
            continue
        }
        total += x
    }
    return total
}

// guard used to validate, then a switch to dispatch: the two features combined.
func grade(_ score: Int) -> String {
    guard score >= 0 else {
        return "invalid"
    }
    switch score / 10 {
    case 10, 9:
        return "A"
    case 8:
        return "B"
    case 7:
        return "C"
    default:
        return "F"
    }
}

func main() {
    // switch over Int: value case, comma list, default
    checkS("switch zero", size(0), "zero")
    checkS("switch small lo", size(1), "small")
    checkS("switch small hi", size(3), "small")
    checkS("switch medium", size(6), "medium")
    checkS("switch big", size(42), "big")

    // switch over String with multi-statement body
    check("switch vowel", vowelScore("e"), 11)
    check("switch y", vowelScore("y"), 5)
    check("switch consonant", vowelScore("z"), 0)

    // switch over enum constants
    check("switch enum north", degrees(Direction.north), 0)
    check("switch enum east", degrees(Direction.east), 90)
    check("switch enum south", degrees(Direction.south), 180)
    check("switch enum west", degrees(Direction.west), 270)

    // break ends a case
    check("switch break case 1", breakInCase(1), 100)
    check("switch break case 2", breakInCase(2), 200)
    check("switch break default", breakInCase(5), 999)

    // switch nested in a for-in over an array
    // [3,1,4,2] -> mod3 = [0,1,1,2] -> 100 + 10 + 10 + 1 = 121
    check("switch in loop", tally([3, 1, 4, 2]), 121)

    // guard early return
    check("guard div ok", reciprocalTimes(20, 4), 5)
    check("guard div by zero", reciprocalTimes(20, 0), -1)

    // guard compound condition
    check("guard in range", inRange(5, 1, 10), 1)
    check("guard below range", inRange(0, 1, 10), 0)
    check("guard above range", inRange(11, 1, 10), 0)

    // guard else continue inside a loop
    check("guard continue evens", sumEvens([1, 2, 3, 4, 5, 6]), 12)

    // guard then switch combined
    checkS("grade A", grade(95), "A")
    checkS("grade B", grade(83), "B")
    checkS("grade C", grade(71), "C")
    checkS("grade F", grade(40), "F")
    checkS("grade invalid", grade(-5), "invalid")

    if fails == 0 {
        print("Swift completion self test passed")
    }
}

main()
exit(fails)
