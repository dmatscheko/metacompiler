// A project module imported by tests/java-test-multifile.java (via -i tests/imports).
// raw and hits are package-private: tests/java-fail-test-member-access.java and
// tests/java-fail-test-untyped-recv.java use them from outside package util,
// which javac refuses. scale is protected: a subclass in another package may
// call it on a receiver of its own type, and
// tests/java-fail-test-protected-access.java and
// tests/java-fail-test-protected-recv.java call it from a class that is none
// and on a Toolbox receiver, which javac refuses. Toolbox(int) is
// package-private and tests/java-fail-test-ctor-access.java calls it.
package util;

public class Toolbox {
    int hits;

    public Toolbox() {
        hits = 0;
    }

    Toolbox(int v) {
        hits = v;
    }

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
