// A numeric separator stands between two digits, so a trailing one is a syntax error.
// tsc 6.0.3 rejects '1_' with
//   TS6188 Numeric separators are not allowed here.
// and node rejects it as well, so this file SHOULD fail in both halves before it prints
// anything.
function main(): void {
    println("compiled");
    println(1_);
}
