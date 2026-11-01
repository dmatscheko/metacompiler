/* Cyclic inheritance fail-test (JLS 8.1.4, 9.1.3). The qualifier A in A.B makes
 * A depend on itself, so javac 24.0.2 refuses the file with "cyclic inheritance
 * involving A" and nothing runs. Both halves must refuse it the same way before
 * main starts. This is a SHOULD-FAIL entry. **/

class Ok extends Base {}
class Base {}
interface A extends A.B {
    interface B {}
}

public class Main {
    public static void main(String[] args) {
        System.out.println("must not run");
    }
}
