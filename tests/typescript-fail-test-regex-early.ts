// A regular expression literal the pattern grammar rejects is an error. tsc 5.8.3
// rejects '/\p{scx=Foo}/u' with
//   TS1526 Unknown Unicode property value.
// and node with "Invalid property name", so this file SHOULD fail in both halves
// before it prints anything.
function main(): void {
    println("compiled")
    println(/\p{scx=Foo}/u.test("a"))
}
