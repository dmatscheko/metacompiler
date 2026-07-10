// MetaJS anytype misuse demo: THIS PROGRAM SHOULD FAIL in the interpreter and the
// compiler. anytype may only initialize a declaration (var v = anytype); assigning
// it to an already existing variable must abort the run with a nonzero exit code.

function main() {
    var x = 5;
    x = anytype;
    return 0;
}
