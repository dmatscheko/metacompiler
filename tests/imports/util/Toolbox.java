// A project module imported by tests/java-test-multifile.java (via -i tests/imports).
package util;

class Toolbox {
    static int clamp(int v, int lo, int hi) {
        if (v < lo) { return lo; }
        if (v > hi) { return hi; }
        return v;
    }
}
