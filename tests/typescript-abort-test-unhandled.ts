// ABORT PROBE. This program is SUPPOSED to die, and its diagnostic text is the
// thing under test: the interpreter half, llvm.Run and the NATIVE binary must all
// produce the same abort message. See the "SHOULD ABORT" rows in
// .vscode/launch.json and abort_norm() in test.sh.
//
// A rejection nothing handles by the end of the microtask drain is node's
// unhandled rejection: it is reported after main() and every job has run, and the
// process exits 1. A rejection handled later in the same drain is not one.
async function lost(): Promise<void> { await null; throw new Error("unhandled rejection probe"); }
async function handledLater(): Promise<void> { throw new Error("handled"); }
function main(): void {
    lost();
    const p = handledLater();
    p.catch((e: any) => { println("caught " + e.message); });
    Promise.reject(1).then(() => 2).catch((e: any) => { println("chain caught " + e); });
    println("main end");
}
