// A package-private type of package util: tests/java-fail-test-pkg-access.java
// imports it from outside the package, which javac refuses.
package util;

class Hidden {
    public static int seven() { return 7; }
}
