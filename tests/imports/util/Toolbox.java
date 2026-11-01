// A project module imported by tests/java-test-multifile.java (via -i tests/imports).
// raw is package-private: tests/java-fail-test-member-access.java calls it from
// outside package util, which javac refuses. scale is protected: a subclass in
// another package may call it, and tests/java-fail-test-protected-access.java
// calls it from a class that is none, which javac refuses.
package util;

public class Toolbox {
    public static int clamp(int v, int lo, int hi) {
        if (v < lo) { return lo; }
        if (v > hi) { return hi; }
        return raw(v);
    }

    static int raw(int v) {
        return v;
    }

    protected int scale() {
        return 3;
    }
}
