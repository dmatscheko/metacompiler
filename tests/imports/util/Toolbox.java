// A project module imported by tests/java-test-multifile.java (via -i tests/imports).
package util;

public class Toolbox {
    public static int clamp(int v, int lo, int hi) {
        if (v < lo) { return lo; }
        if (v > hi) { return hi; }
        return v;
    }
}
