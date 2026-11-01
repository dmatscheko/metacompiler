/* Cyclic type-variable bounds of a generic CONSTRUCTOR fail-test (JLS 4.4,
 * 8.8.4). T depends on U and U on T, so javac 24.0.2 refuses the file with
 * "cyclic inheritance involving T" and nothing runs; the class's other
 * generic constructor is well-formed. Both halves must refuse it the same way
 * before main starts. This is a SHOULD-FAIL entry. **/

class Box {
    <S extends Comparable<S>> Box(S s) { }
    <T extends U, U extends T> Box(T t, U u) { }
}

public class Main {
    public static void main(String[] args) {
        System.out.println("must not run");
    }
}
