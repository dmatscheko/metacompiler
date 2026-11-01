/* Constructor access fail-test (JLS 6.6.1, 15.9.3). util.Toolbox's Toolbox(int)
 * is declared without a modifier, so it is accessible only inside package
 * util, and an inaccessible constructor is no candidate: javac 24.0.2 refuses
 * `new Toolbox(3)` with "constructor Toolbox in class Toolbox cannot be
 * applied to given types;" for the reason "Toolbox(int) is not public in
 * Toolbox; cannot be accessed from outside package", and nothing runs. The
 * public Toolbox() would be accepted. Run with -i tests/imports. This is a
 * SHOULD-FAIL entry. **/
import util.Toolbox;

public class Main {
    public static void main(String[] args) {
        System.out.println("must not run " + new Toolbox().toString().length());
        Toolbox t = new Toolbox(3);
        System.out.println(Toolbox.clamp(15, 0, 10));
    }
}
