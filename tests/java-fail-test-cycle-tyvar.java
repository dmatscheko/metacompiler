/* Cyclic type-variable bounds fail-test (JLS 4.4). A type variable depends on
 * the type variable that is its bound; B -> C -> C1 -> B depends on itself
 * (A only reaches the cycle), so javac 24.0.2 refuses the file with "cyclic
 * inheritance involving B" and nothing runs. Both halves must refuse it the
 * same way before main starts. This is a SHOULD-FAIL entry. **/

class Ok<T extends Comparable<T>, U extends T, V extends java.util.List<V>> {}

class Neg<A extends B, B extends C, C extends C1, C1 extends B> {}

public class Main {
    static <S extends Comparable<S>, R extends S> R pick(R r) { return r; }

    public static void main(String[] args) {
        System.out.println("must not run");
    }
}
