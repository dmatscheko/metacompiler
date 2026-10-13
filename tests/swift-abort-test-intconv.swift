// ABORT PROBE. This program is SUPPOSED to die, and its diagnostic text is the
// thing under test. See the "SHOULD ABORT" rows in .vscode/launch.json and
// abort_norm() in test.sh.
//
// Int(x) of a Double whose truncation is outside Int's range traps rather than
// wrapping. swiftc 6.1.2 and all five legs answer "Fatal error: Double value
// cannot be converted to Int because the result would be greater than Int.max".
let ds = [-0.5, 127.9, 1e300]
print("before the abort", UInt(ds[0]), Int8(ds[1]))
print(Int(ds[2]))
print("never reached")
