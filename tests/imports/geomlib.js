// An imported library for tests/js-test-multifile.js (found via -i tests/imports). It is
// loaded by `import { Vec, hypotSq } from './geomlib.js'`; its top-level class and
// function register in the shared global scope, so the main file can use them directly.
// The named-binding list in the import is cosmetic - loading the file is what registers
// the names. The same file is parsed with the same grammar as the main program.

class Vec {
    constructor(x, y) {
        this.x = x;
        this.y = y;
    }
    dot(o) {
        return this.x * o.x + this.y * o.y;
    }
    add(o) {
        return new Vec(this.x + o.x, this.y + o.y);
    }
    static scale(v, f) {
        return new Vec(v.x * f, v.y * f);
    }
}

// A top-level helper that uses the imported class' method; it too becomes a global.
function hypotSq(v) {
    return v.dot(v);
}
