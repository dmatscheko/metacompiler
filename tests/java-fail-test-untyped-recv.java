/* Member access through a call's result (JLS 6.6.1). util.Toolbox's hits is
 * declared without a modifier, so only package util may use it. The receiver
 * here is the result of Main.mk(), a static method of the class whose body
 * contains the access, and javac 24.0.2 refuses the assignment with "hits is
 * not public in Toolbox; cannot be accessed from outside package" and nothing
 * runs. Run with -i tests/imports. This is a SHOULD-FAIL entry. **/
import util.Toolbox;

public class Main {
    static Toolbox mk() { return new Toolbox(); }

    public static void main(String[] args) {
        System.out.println("must not run " + Toolbox.clamp(15, 0, 10));
        Main.mk().hits += 2;
    }
}
