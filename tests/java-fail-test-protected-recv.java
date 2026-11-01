/* Protected receiver fail-test (JLS 6.6.2.1). util.Toolbox's scale is a
 * protected instance method, so outside package util a subclass S may call it
 * only on a receiver whose type is S or a subclass of S. Peeker extends
 * Toolbox but calls it on a Toolbox, and javac 24.0.2 refuses the call with
 * "scale() has protected access in Toolbox" and nothing runs.
 * tests/java-test-multifile.java calls it on a receiver of the subclass's own
 * type, which is accepted. Run with -i tests/imports. This is a SHOULD-FAIL
 * entry. **/
import util.Toolbox;

class Peeker extends Toolbox {
    int peek(Toolbox t) { return t.scale(); }
}

public class Main {
    public static void main(String[] args) {
        System.out.println("must not run " + Toolbox.clamp(15, 0, 10));
        System.out.println(new Peeker().peek(new Toolbox()));
    }
}
