// A numeric separator stands between two digits, so two in a row are a syntax error.
// node 24 rejects '1__0' with
//   SyntaxError: Only one underscore is allowed as numeric separator
// so this file SHOULD fail in both halves before it prints anything.
function main() {
    println("compiled")
    println(1__0)
}
