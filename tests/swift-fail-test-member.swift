// Swift undefined name inside a type. swiftc rejects it even when never reached:
//   swift-fail-test-member.swift:18:30: error: cannot find 'speed' in scope
// Tally conforms to Equatable and extends a protocol of the program, neither of
// which gives it a member speed, so this file SHOULD fail in both halves before it
// prints anything. Everything before that line is a member of Tally or of Named,
// or a stdlib type.

protocol Named { var name: String { get } }
extension Named { func shout() -> String { name.uppercased() } }

struct Tally: Equatable, Named {
    var name: String
    var hits = 0
    func report() -> String { shout() + " " + String(hits) }
}
extension Tally {
    func fast() -> Int {
        if hits < 0 { return speed }
        return Int(hits)
    }
}

print("compiled")
print(Tally(name: "a").report(), Tally(name: "b").fast())
