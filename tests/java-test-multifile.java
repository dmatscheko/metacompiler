// Multi-file Java test: util.Toolbox lives in tests/imports/util/Toolbox.java and
// is found via the -i include root (mec -i tests/imports ...). The imported file
// is parsed with the same grammar; its classes register like the main file's.
// java.util.List is a builtin no-op import, mixed in on purpose; util.Pair
// imports a second module file (one class per file, like idiomatic Java).
// PeerShape is named by no import: like javac's source path, the run finds it
// as PeerShape.java under the include root, since Square extends it.
import java.util.List;
import util.Toolbox;
import util.Pair;
import util.Gauge;

class Square extends PeerShape {
    int sides() { return 4; }
}

// A subclass in another package may use Toolbox's protected member.
class Tripler extends Toolbox {
    int triple(int v) { return v * this.scale(); }
}

// super(4) from another package: Gauge(long) is protected, Gauge(int, ...)
// forms are no candidates.
class Dial extends Gauge {
    Dial() { super(4); }
}

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
        Square sq = new Square();
        Main.check("superclass from a peer file", sq.sides() == 4 && sq.name().equals("shape"));
        Main.check("protected member from a subclass", new Tripler().triple(5) == 15);
        Gauge g = new Gauge(1, 2);
        Main.check("inaccessible ctor no candidate", g.v == 2);
        Main.check("protected ctor no candidate to new", new Gauge(4).v == 5);
        Main.check("protected ctor a candidate to super", new Dial().v == 4);
        Main.check("private method no candidate", g.pick(5) == 2);
        Main.check("package-private method no candidate", g.near(5) == 4);
        Main.check("private static method no candidate", Gauge.scale(5) == 6);
        Main.check("every overload a candidate inside", g.reading() == 13);

        if (Main.fails == 0) {
            System.out.println("java multifile test passed");
        }
        System.exit(Main.fails);
    }
}
