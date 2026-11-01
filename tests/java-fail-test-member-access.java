/* Member access fail-test (JLS 6.6.1). util.Toolbox is public, but its raw
 * method is declared without a modifier, so it is accessible only inside
 * package util; javac 24.0.2 refuses the call with "raw(int) is not public in
 * Toolbox; cannot be accessed from outside package" and nothing runs.
 * Toolbox.clamp is public and would be accepted. Run with -i tests/imports.
 * This is a SHOULD-FAIL entry. **/
import util.Toolbox;

public class Main {
    public static void main(String[] args) {
        System.out.println("must not run " + Toolbox.clamp(15, 0, 10));
        System.out.println(Toolbox.raw(3));
    }
}
