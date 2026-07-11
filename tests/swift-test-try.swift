// Swift do/catch/throw, genuinely executed (interpreter and compiler).
//
// throw raises a value (a String or an object) that unwinds through any depth of
// calls to the nearest catch; catch binds the value. `catch let e [as T]` binds e,
// while a bare `catch { ... }` binds Swift's implicit `error`. The FIRST catch clause
// wins (patterns are parsed but not discriminated without runtime types). Swift has
// NO finally. An uncaught throw is a clean runtime error.
//
// A return/break/continue that LEAVES a do or catch body works in both engines: the
// interpreter propagates the statement signal, and the compiler turns it into a control
// signal that is re-issued in the enclosing function/loop (see relabelReturn /
// nestedReturn / loopBreak / loopContinue below).
//
// try / try? / try! are expression prefixes: they are parsed and ignored (the throw
// happens inside the call and unwinds regardless), so `let x = try risky(5)` works.
//
// Top level code counts failed checks and ends with exit(fails), so the run exits 0
// exactly when every check passes; the interpreter and compiler must agree.

var fails = 0

// A thrown error object carrying a value read back in the catch.
class BoomError {
    var code: Int
    init(_ c: Int) {
        self.code = c
    }
}

// Throws for some inputs; the throw unwinds out of the call to the caller's catch.
func risky(_ n: Int) throws -> Int {
    if n > 3 {
        throw BoomError(n)
    }
    return n * 2
}

// Nested do + re-throw: the inner catch throws a new value the outer catch handles.
// The result is captured and returned AFTER the do (no return-across-try here).
func relabel() -> String {
    var result = ""
    do {
        do {
            throw "inner"
        } catch {
            throw "wrapped"
        }
    } catch {
        result = "handled"
    }
    return result
}

// return that LEAVES a do body (and one that leaves a catch body): the enclosing
// function returns the value (Swift has no finally).
func relabelReturn(_ n: Int) -> Int {
    do {
        if n > 0 {
            return n * 10                // return out of the do body
        }
        throw "neg"
    } catch {
        return -1                        // return out of the catch body
    }
}

// A return out of an INNER do propagates through the OUTER do (nested), re-signalled
// each level until it reaches the enclosing function.
func nestedReturn() -> Int {
    do {
        do {
            return 9
        } catch {
        }
    } catch {
    }
    return 0
}

// break / continue that LEAVE a do body while inside a loop.
func loopBreak() -> Int {
    var sum = 0
    for i in 0...9 {
        do {
            if i == 3 {
                break
            }
            sum = sum + i
        } catch {
        }
    }
    return sum                           // 0+1+2 = 3
}
func loopContinue() -> Int {
    var sum = 0
    for i in 0...4 {
        do {
            if i == 2 {
                continue
            }
            sum = sum + i
        } catch {
        }
    }
    return sum                           // 0+1+3+4 = 8
}

// ----- Top level checks -----

// A bare catch binds the value; the statement after throw is skipped; no finally.
var log = ""
do {
    log = log + "a"
    throw "boom"
    log = log + "X"
} catch {
    log = log + "b"
}
if log != "ab" {
    fails += 1
}

// The default (bare) catch binds the implicit `error`.
var defaultMsg = ""
do {
    throw "boom2"
} catch {
    defaultMsg = "caught"
    if error == "boom2" {
        defaultMsg = defaultMsg + "-ok"
    }
}
if defaultMsg != "caught-ok" {
    fails += 1
}

// A throw from a nested call unwinds, carrying an object read in a typed catch binding.
var caught = -1
do {
    let r = try risky(5)
    fails += 1                           // not reached
} catch let e as BoomError {
    caught = e.code
}
if caught != 5 {
    fails += 1
}

// The implicit `error` also carries the thrown object (subset: no type discrimination).
var caught2 = -1
do {
    let r2 = try risky(9)
    fails += 1                           // not reached
} catch {
    caught2 = error.code
}
if caught2 != 9 {
    fails += 1
}

// No throw: the do body runs, the catch is skipped (Swift has no finally).
if try risky(2) != 4 {
    fails += 1
}
var order = ""
do {
    order = order + "t"
} catch {
    order = order + "c"
}
if order != "t" {
    fails += 1
}

// Nested do + re-throw.
if relabel() != "handled" {
    fails += 1
}

// return / break / continue that leave a do or catch body.
if relabelReturn(4) != 40 {              // return out of do
    fails += 1
}
if relabelReturn(-1) != -1 {             // return out of catch
    fails += 1
}
if nestedReturn() != 9 {                 // return through nested dos
    fails += 1
}
if loopBreak() != 3 {
    fails += 1
}
if loopContinue() != 8 {
    fails += 1
}

if fails == 0 {
    print("Swift do/catch OK")
}
exit(fails)
