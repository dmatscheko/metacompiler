// TypeScript parameter decorators are parsed and ignored, so the decorated method
// still runs. (Class and method decorators are accepted but notImplemented - see
// typescript-test-recognize.ts; this file only uses the genuinely-lowered forms.)
// main() returns the failure count, so exit 0 means every check passed; the
// interpreter and compiler must agree.

let failures: number = 0;
function check(cond: boolean): void {
    if (!cond) { failures = failures + 1; }
}

// A decorator factory, referenced only as a parameter decorator below.
function Inject() {
    return function (target: any, key: any, index: number) {};
}

class Calc {
    base: number = 10;
    // A parameter decorator on the first parameter (ignored at runtime).
    add(@Inject() x: number, y: number): number {
        return this.base + x + y;
    }
    scale(@Inject() factor: number): number {
        return this.base * factor;
    }
}

function main(): number {
    const c = new Calc();
    check(c.add(2, 3) === 15);
    check(c.scale(4) === 40);
    if (failures === 0) { println("TypeScript annotations OK"); }
    return failures;
}
