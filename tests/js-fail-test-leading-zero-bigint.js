// A BigInt literal may not continue a leading '0' with more digits, even in sloppy code,
// where '01' alone is a legacy octal. node 24 rejects '01n' with
//   SyntaxError: Invalid or unexpected token
// so this file SHOULD fail in both halves before it prints anything.
function main() {
    println("compiled")
    println(01n)
}
