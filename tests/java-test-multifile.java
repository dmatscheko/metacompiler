// Multi-file Java test: util.Toolbox lives in tests/imports/util/Toolbox.java and
// is found via the -i include root (mec -i tests/imports ...). The imported file
// is parsed with the same grammar; its classes register like the main file's.
// java.util.List is a builtin no-op import, mixed in on purpose; util.Pair
// imports a second module file (one class per file, like idiomatic Java).
import java.util.List;
import util.Toolbox;
import util.Pair;

class Main {
    static int fails = 0;

    static void check(String name, boolean ok) {
        if (!ok) {
            System.out.println("FAIL " + name);
            Main.fails = Main.fails + 1;
        }
    }

    public static void main(String[] args) {
        Main.check("imported static method", Toolbox.clamp(15, 0, 10) == 10);
        Pair p = new Pair(19, 23);
        Main.check("imported class", p.sum() == 42);

        if (Main.fails == 0) {
            System.out.println("java multifile test passed");
        }
        System.exit(Main.fails);
    }
}
