// A second project module: one class per file, like idiomatic Java.
package util;

public class Pair {
    int a;
    int b;

    public Pair(int a, int b) {
        this.a = a;
        this.b = b;
    }

    public int sum() {
        return this.a + this.b;
    }
}
