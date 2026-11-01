// A project module whose overloads differ in access: from outside package util
// the private and package-private declarations are no candidates (JLS
// 15.12.2.1), so tests/java-test-multifile.java's calls reach the public ones
// although the others fit the arguments better. Inside the class every
// declaration is a candidate (reading).
package util;

public class Gauge {
    public int v;

    private Gauge(int a, int b) { v = 1; }
    public Gauge(double a, int b) { v = 2; }
    Gauge(int a, long b) { v = 3; }
    protected Gauge(long a) { v = 4; }
    public Gauge(double a) { v = 5; }

    private int pick(int a) { return 1; }
    public int pick(double a) { return 2; }
    int near(int a) { return 3; }
    public int near(long a) { return 4; }
    private static int scale(int a) { return 5; }
    public static int scale(double a) { return 6; }

    public int reading() { return pick(5) * 10 + near(5); }
}
