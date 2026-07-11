/* MetaJS try/catch/finally/throw self test.
 *
 * throw raises a value that unwinds through any depth of calls to the nearest
 * catch; finally always runs (and its own return/break/continue would override);
 * catch binds the thrown value. A non-user exception and an uncaught throw are
 * reported as runtime errors.
 *
 * Note: a return/break/continue that leaves a try/catch/finally body is not in the
 * subset for the COMPILER (each body is its own IR closure), so this test keeps such
 * jumps outside the try - the value is captured in a variable and returned after.
 * The interpreter propagates them via the statement-signal protocol either way.
 *
 * main() counts failed checks and returns the count, so the run exits 0 exactly when
 * every check passes; the interpreter and compiler must agree. **/

var fails = 0;
function check(cond) { if (!cond) { fails = fails + 1; } }

// A helper that throws for some inputs; the throw unwinds out of the call.
function risky(n) {
    if (n > 3) { throw { code: n }; }
    return n * 2;
}

// Re-throw: the inner catch throws a new value that the outer catch handles. The
// result is captured in a variable so the function returns it AFTER the try.
function relabel() {
    var result = "";
    try {
        try { throw "inner"; } catch (e) { throw "rethrown:" + e; }
    } catch (e2) {
        result = e2;
    }
    return result;
}

function main() {
    // catch binds the value; finally runs; the statement after throw is skipped.
    var log = "";
    try {
        log = log + "a";
        throw "boom";
        log = log + "X";
    } catch (e) {
        log = log + "b" + e;
    } finally {
        log = log + "c";
    }
    check(log == "abboomc");

    // A throw from a nested call unwinds to the enclosing catch, carrying an object.
    var caught = -1;
    try {
        var r = risky(5);
        check(false);
    } catch (e) {
        caught = e.code;
    }
    check(caught == 5);

    // No throw: the try value is used, the catch is skipped, finally still runs.
    check(risky(2) == 4);
    var order = "";
    try { order = order + "t"; } catch (e) { order = order + "c"; } finally { order = order + "f"; }
    check(order == "tf");

    // Nested try + re-throw.
    check(relabel() == "rethrown:inner");

    // try/finally with no catch: finally runs on the normal path.
    var fin = 0;
    try { fin = fin + 1; } finally { fin = fin + 10; }
    check(fin == 11);

    if (fails == 0) { println("MetaJS try/catch OK"); }
    return fails;
}
