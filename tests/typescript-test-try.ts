// TypeScript try/catch/finally/throw, genuinely executed (interpreter and compiler).
//
// Same runtime model as JavaScript (types are erased): throw raises a value that
// unwinds to the nearest catch; the catch binding is optional and may carry a type
// (catch (e: unknown)); finally always runs; a return/break/continue leaving a try or
// catch body works in both engines. An uncaught throw is a clean runtime error.
//
// main() counts failed checks and returns the count, so the run exits 0 exactly when
// every check passes; the interpreter and compiler must agree.

let failures: number = 0;
function check(cond: boolean): void { if (!cond) { failures = failures + 1; } }

function risky(n: number): number {
    if (n > 3) { throw { code: n }; }
    return n * 2;
}

// return out of a try, and out of a catch (with a typed binding).
function classify(n: number): number {
    try {
        if (n > 0) { return n * 10; }
        throw "neg";
    } catch (e: unknown) {
        return -1;
    } finally { }
}

// A return out of an INNER try propagates through the OUTER try.
function nestedReturn(): number {
    try {
        try { return 9; } finally { }
    } finally { }
    return 0;
}

// break / continue leaving a try body inside a loop.
function loopBreak(): number {
    let sum: number = 0;
    for (let i: number = 0; i < 10; i = i + 1) {
        try { if (i === 3) { break; } sum = sum + i; } finally { }
    }
    return sum;              // 0+1+2 = 3
}
function loopContinue(): number {
    let sum: number = 0;
    for (let i: number = 0; i < 5; i = i + 1) {
        try { if (i === 2) { continue; } sum = sum + i; } catch (e) { }
    }
    return sum;              // 0+1+3+4 = 8
}

function main(): number {
    let log: string = "";
    try {
        log = log + "a";
        throw "boom";
        log = log + "X";
    } catch (e) {
        log = log + "b";
    } finally {
        log = log + "c";
    }
    check(log === "abc");

    let caught: number = -1;
    try { risky(5); check(false); } catch (e) { caught = e.code; }
    check(caught === 5);
    check(risky(2) === 4);

    // ES2019 optional catch binding.
    let ok: number = 0;
    try { throw "z"; } catch { ok = 1; }
    check(ok === 1);

    check(classify(4) === 40);
    check(classify(-1) === -1);
    check(nestedReturn() === 9);
    check(loopBreak() === 3);
    check(loopContinue() === 8);

    if (failures === 0) { println("TypeScript try/catch OK"); }
    return failures;
}
