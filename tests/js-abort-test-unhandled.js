// ABORT PROBE. This program is SUPPOSED to die, and its diagnostic text is the
// thing under test: the interpreter half, llvm.Run and the NATIVE binary must all
// produce the same abort message. See the "SHOULD ABORT" rows in
// .vscode/launch.json and abort_norm() in test.sh.
//
// A rejection nothing handles by the end of the microtask drain is node's
// unhandled rejection: it is reported after main() and every job has run, and the
// process exits 1. A rejection handled later in the same drain is not one.
async function lost() { await null; throw new Error("unhandled rejection probe") }
async function handledLater() { throw new Error("handled") }
function main() {
    lost()
    var p = handledLater()
    p.catch(function (e) { println("caught " + e.message) })
    Promise.reject(1).then(function () { return 2 }).catch(function (e) { println("chain caught " + e) })
    println("main end")
}
