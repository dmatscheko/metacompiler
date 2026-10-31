// ABORT PROBE. This program is SUPPOSED to die, and its diagnostic text is the
// thing under test: a backtracking search that hits the regexp engine's step
// limit reports it in the same words in the interpreter half, llvm.Run (the Go
// twin) and the NATIVE binary (layer 2). See the "SHOULD ABORT" rows in
// .vscode/launch.json and abort_norm() in test.sh.
const s: string = "a".repeat(40) + "b"
println("before the abort")
println(/^(a+)+$/.test(s))
println("never reached")
