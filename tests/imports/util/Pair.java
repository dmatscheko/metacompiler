// A second project module: one class per file, like idiomatic Java.
package util;

class Pair {
    int a;
    int b;

    Pair(int a, int b) {
        this.a = a;
        this.b = b;
    }

    int sum() {
        return this.a + this.b;
    }
}
