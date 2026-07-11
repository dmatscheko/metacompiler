<?php
// PHP try/catch/finally/throw, genuinely executed (interpreter and compiler).
//
// throw raises a value that unwinds through calls to the nearest catch; catch binds it
// (the first catch clause wins - exception types, including multi-catch A | B, are parsed
// but not discriminated, and the catch variable is optional); finally always runs. A
// return/break/continue that leaves a try or catch body works in both engines. An
// uncaught throw is a clean runtime error. Control flow inside a finally is not modelled,
// so this file keeps its finally blocks side-effect only.
//
// The program counts failed checks in $fails and ends with exit($fails), so the run exits
// 0 exactly when every check passes; the interpreter (php-interpreter.abnf) and the
// LLVM-IR compiler (php-to-llvm-ir.abnf) run the same file and must agree.

$fails = 0;

function check($name, $got, $want) {
    global $fails;
    if ($got !== $want) {
        echo "FAIL " . $name . "\n";
        $fails = $fails + 1;
    }
}

class Boom {
    public $code;
    public function __construct($c) {
        $this->code = $c;
    }
}

// A throw from a nested function, caught by the caller.
function risky($n) {
    if ($n > 3) {
        throw new Boom($n);
    }
    return $n * 2;
}

// return out of a try, and out of a catch.
function classify($n) {
    try {
        if ($n > 0) {
            return $n * 10;
        }
        throw new Boom(0);
    } catch (Exception $e) {
        return -1;
    } finally {
    }
}

// A return out of an INNER try propagates through the OUTER try.
function nestedReturn() {
    try {
        try {
            return 9;
        } finally {
        }
    } finally {
    }
    return 0;
}

// break / continue leaving a try body inside a loop.
function loopBreak() {
    $sum = 0;
    for ($i = 0; $i < 10; $i++) {
        try {
            if ($i == 3) {
                break;
            }
            $sum = $sum + $i;
        } finally {
        }
    }
    return $sum;          // 0+1+2 = 3
}

function loopContinue() {
    $sum = 0;
    for ($i = 0; $i < 5; $i++) {
        try {
            if ($i == 2) {
                continue;
            }
            $sum = $sum + $i;
        } catch (Exception $e) {
        }
    }
    return $sum;          // 0+1+3+4 = 8
}

// ----- basic try / catch / finally: order of effects -----
$log = "";
try {
    $log = $log . "a";
    throw new Boom(1);
} catch (Exception $e) {
    $log = $log . "b";
} finally {
    $log = $log . "c";
}
check("try order", $log, "abc");

// ----- no-throw path: the try runs, catch is skipped, finally still runs -----
$noThrow = "";
try {
    $noThrow = $noThrow . "x";
} catch (Exception $e) {
    $noThrow = $noThrow . "!";
} finally {
    $noThrow = $noThrow . "y";
}
check("no-throw path", $noThrow, "xy");

// ----- throw + catch binding (a field read on the caught object) -----
$caught = -1;
try {
    risky(5);
    check("unreachable after throw", true, false);
} catch (Exception $e) {
    $caught = $e->code;
}
check("catch binding", $caught, 5);

// ----- a nested throw that is NOT taken: the returned value flows out -----
check("nested no-throw value", risky(2), 4);

// ----- return out of try / out of catch -----
check("return from try", classify(4), 40);
check("return from catch", classify(-1), -1);

// ----- return through nested tries -----
check("nested return", nestedReturn(), 9);

// ----- break / continue out of a try inside a loop -----
check("break out of try", loopBreak(), 3);
check("continue out of try", loopContinue(), 8);

// ----- multi-catch in a single clause is parsed; the clause still binds and runs -----
$multi = 0;
try {
    throw new Boom(7);
} catch (Boom | Exception $e) {
    $multi = $e->code;
}
check("multi-catch clause", $multi, 7);

// ----- several catch clauses: the FIRST one wins (types are not discriminated) -----
$first = 0;
try {
    throw new Boom(11);
} catch (Boom $e) {
    $first = $e->code;
} catch (Exception $e) {
    $first = -99;
}
check("first catch wins", $first, 11);

// ----- catch with no bound variable (PHP 8): the clause still runs -----
$noVar = 0;
try {
    throw new Boom(3);
} catch (Exception) {
    $noVar = 42;
}
check("catch without variable", $noVar, 42);

// ----- done -----
if ($fails === 0) {
    echo "PHP try/catch OK\n";
}
exit($fails);
