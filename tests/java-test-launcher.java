// A program with no class Main: java's source launcher (`java File.java`, JEP 330)
// runs the FIRST top-level class of the file, so Launcher.main is the entry and
// Second.main, which would exit 1, never runs. Run by the matrix rows "Java
// launcher class (interpreter)" and "(compiler)"; java 24's own launcher gives
// exit status 0 and the same output.
public class Launcher {
    static int fails = 0;

    static void check(String id, boolean cond) {
        if (!cond) {
            System.out.println("FAIL " + id);
            fails++;
        }
    }

    public static void main(String[] args) {
        check("helper", Second.twice(21) == 42);
        System.out.print("java launcher test");
        System.out.flush();
        System.out.println(" passed");
        System.exit(fails);
    }
}

class Second {
    static int twice(int v) { return 2 * v; }

    public static void main(String[] args) {
        System.out.println("FAIL not the first class");
        System.exit(1);
    }
}
