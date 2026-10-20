// TypeScript has no legacy octal literal. tsc 6.0.3 rejects '010' with
//   TS1121 Octal literals are not allowed. Use the syntax '0o10'.
// and node rejects it in strict code, which TypeScript code is ("Octal literals are not
// allowed in strict mode"), so this file SHOULD fail in both halves before it prints
// anything.
function main(): void {
    println("compiled");
    println(010);
}
