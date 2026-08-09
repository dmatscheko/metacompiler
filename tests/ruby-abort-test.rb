# ABORT PROBE. This program is SUPPOSED to die, and its diagnostic text is the
# thing under test: the interpreter half, llvm.Run and the NATIVE binary must all
# produce the same abort message. See the "SHOULD ABORT" rows in
# .vscode/launch.json and abort_norm() in test.sh.
puts "before the abort"
raise "abort probe"
puts "never reached"
