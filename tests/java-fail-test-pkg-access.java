/* Package access fail-test (JLS 6.6.1). util.Hidden in tests/imports/util is
 * declared without `public`, so it is accessible only inside package util;
 * javac 24.0.2 refuses this import with "Hidden is not public in util; cannot
 * be accessed from outside package" and nothing runs. util.Pair is public and
 * would be accepted. Both halves must refuse the file the same way before main
 * starts. Run with -i tests/imports. This is a SHOULD-FAIL entry. **/
import util.Pair;
import util.Hidden;

public class Main {
    public static void main(String[] args) {
        System.out.println("must not run " + Hidden.seven() + new Pair(1, 2).sum());
    }
}
