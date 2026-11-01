/* Private access fail-test (JLS 6.6.1). A private member is accessible only
 * inside the top-level type that encloses its declaration: Box.Inner's private
 * field is Box's to read, but Main is another top-level type, and javac 24.0.2
 * refuses the read with "secret has private access in Inner" and nothing runs.
 * This is a SHOULD-FAIL entry. **/
class Box {
    static class Inner { private int secret = 4; }
    static int peek() { return new Inner().secret; }
}

public class Main {
    public static void main(String[] args) {
        System.out.println("must not run " + Box.peek());
        System.out.println(new Box.Inner().secret);
    }
}
