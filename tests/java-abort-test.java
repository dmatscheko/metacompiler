// ABORT PROBE. This program is SUPPOSED to die, and its diagnostic text is the
// thing under test: the interpreter half, llvm.Run and the NATIVE binary must all
// produce the same abort message. See the "SHOULD ABORT" rows in
// .vscode/launch.json and abort_norm() in test.sh. The cause chain makes each
// leg print java's "Caused by: " lines after the first.
public class Main {
    public static void main(String[] args) {
        System.out.println("before the abort");
        throw new RuntimeException("abort probe", new RuntimeException(new IllegalStateException("root cause")));
    }
}
