/* Kotlin try/catch/finally/throw, genuinely executed (interpreter and compiler).
 *
 * throw raises a value (a String or an object) that unwinds through any depth of
 * calls to the nearest catch; catch binds the value (the first catch clause wins -
 * exception types are parsed but not discriminated without runtime types); finally
 * always runs. An uncaught throw is a clean runtime error.
 *
 * A return/break/continue that LEAVES a try or catch body works in both engines: the
 * interpreter propagates the statement signal, and the compiler turns it into a control
 * signal that is re-issued in the enclosing function/loop (see relabelReturn / loopBreak
 * / loopContinue below). (A jump out of a finally block is still not supported.)
 *
 * main() counts failed checks and returns the count, so the run exits 0 exactly when
 * every check passes; the interpreter and compiler must agree byte-for-byte. **/

class BoomException(val code: Int)

// Throws for some inputs; the throw unwinds out of the call to the caller's catch.
fun risky(n: Int): Int {
    if (n > 3) { throw BoomException(n) }
    return n * 2
}

// Nested try + re-throw: the inner catch throws a new value the outer catch handles.
// The result is captured and returned AFTER the try (no return-across-try).
fun relabel(): String {
    var result = ""
    try {
        try { throw "inner" } catch (e: Exception) { throw "wrapped" }
    } catch (e: Exception) {
        result = "handled"
    }
    return result
}

// return that LEAVES a try body (and one that leaves a catch body): the enclosing
// function returns the value, finally still runs on the way out.
fun relabelReturn(n: Int): Int {
    try {
        if (n > 0) { return n * 10 }     // return out of the try
        throw "neg"
    } catch (e: Exception) {
        return -1                        // return out of the catch
    } finally {
        // runs on both paths
    }
}

// A return out of an INNER try propagates through the OUTER try (nested), re-signalled
// each level until it reaches the enclosing function.
fun nestedReturn(): Int {
    try {
        try { return 9 } finally { }
    } finally { }
    return 0
}

// break / continue that LEAVE a try body while inside a loop.
fun loopBreak(): Int {
    var sum = 0
    for (i in 0..9) {
        try {
            if (i == 3) { break }
            sum = sum + i
        } finally { }
    }
    return sum                           // 0+1+2 = 3
}
fun loopContinue(): Int {
    var sum = 0
    for (i in 0..4) {
        try {
            if (i == 2) { continue }
            sum = sum + i
        } catch (e: Exception) { }
    }
    return sum                           // 0+1+3+4 = 8
}

fun main() {
    var fails = 0

    // catch binds the value; finally runs; the statement after throw is skipped.
    var log = ""
    try {
        log = log + "a"
        throw "boom"
        log = log + "X"
    } catch (e: Exception) {
        log = log + "b"
    } finally {
        log = log + "c"
    }
    if (log != "abc") { fails = fails + 1 }

    // A throw from a nested call unwinds, carrying an object read in the catch.
    var caught = -1
    try {
        val r = risky(5)
        fails = fails + 1          // not reached
    } catch (e: BoomException) {
        caught = e.code
    }
    if (caught != 5) { fails = fails + 1 }

    // No throw: the try runs, the catch is skipped, finally still runs.
    if (risky(2) != 4) { fails = fails + 1 }
    var order = ""
    try { order = order + "t" } catch (e: Exception) { order = order + "c" } finally { order = order + "f" }
    if (order != "tf") { fails = fails + 1 }

    // Nested try + re-throw.
    if (relabel() != "handled") { fails = fails + 1 }

    // return / break / continue that leave a try or catch body.
    if (relabelReturn(4) != 40) { fails = fails + 1 }    // return out of try
    if (relabelReturn(-1) != -1) { fails = fails + 1 }   // return out of catch
    if (nestedReturn() != 9) { fails = fails + 1 }       // return through nested tries
    if (loopBreak() != 3) { fails = fails + 1 }
    if (loopContinue() != 8) { fails = fails + 1 }

    // try/finally with no catch: finally runs on the normal path.
    var fin = 0
    try { fin = fin + 1 } finally { fin = fin + 10 }
    if (fin != 11) { fails = fails + 1 }

    if (fails == 0) { println("Kotlin try/catch OK") }
    return fails
}
