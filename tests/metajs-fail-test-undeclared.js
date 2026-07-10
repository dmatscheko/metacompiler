// MetaJS declaration rule demo: THIS PROGRAM SHOULD FAIL in the interpreter and the
// compiler. Assigning to a name that was never declared with var/let/const must
// abort the run with a nonzero exit code.

function main() {
    zzz = 5;
    return zzz - 5;
}
