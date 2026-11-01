/* Protected access fail-test (JLS 6.6.2). util.Toolbox's scale is protected,
 * so outside package util only a subclass's body may use it; Main is no
 * subclass, and javac 24.0.2 refuses the call with "scale() has protected
 * access in Toolbox" and nothing runs. tests/java-test-multifile.java calls it
 * from a subclass, which is accepted. Run with -i tests/imports. This is a
 * SHOULD-FAIL entry. **/
import util.Toolbox;

public class Main {
    public static void main(String[] args) {
        System.out.println("must not run " + Toolbox.clamp(15, 0, 10));
        System.out.println(new Toolbox().scale());
    }
}
