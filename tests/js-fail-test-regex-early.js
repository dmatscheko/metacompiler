// A regular expression literal the pattern grammar rejects is an early error. node 24
// rejects '/\p{scx=Foo}/u' with
//   SyntaxError: Invalid regular expression: /\p{scx=Foo}/u: Invalid property name
// so this file SHOULD fail in both halves before it prints anything.
function main() {
    println("compiled")
    println(/\p{scx=Foo}/u.test("a"))
}
