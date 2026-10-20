// A numeric separator may not follow a leading '0', even in sloppy code. node 24 rejects
// '0_1' with
//   SyntaxError: Numeric separator can not be used after leading 0.
// so this file SHOULD fail in both halves before it prints anything.
function main() {
    println("compiled")
    println(0_1)
}
