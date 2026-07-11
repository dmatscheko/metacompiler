/* Java annotations in every position are parsed and ignored, so the annotated
 * code still runs. Main.main counts failed checks and exits with that count, so
 * the run exits 0 exactly when everything works (interpreter and compiler agree). */

class Widget {
    int size;
    Widget(int s) { this.size = s; }

    @SuppressWarnings("unchecked")
    int doubled() { return this.size * 2; }
}

public class Main {
    static int fails = 0;

    @Deprecated
    static int inc(int x) { return x + 1; }

    // Annotation with an array-valued argument.
    @SuppressWarnings({"unchecked", "rawtypes"})
    static int sq(int x) { return x * x; }

    static void check(int got, int want) {
        if (got != want) { Main.fails += 1; }
    }

    public static void main(String[] args) {
        Widget w = new Widget(5);
        Main.check(w.doubled(), 10);
        Main.check(Main.inc(4), 5);
        Main.check(Main.sq(3), 9);
        // An annotation on a local variable declaration.
        @SuppressWarnings("unused") int local = Main.sq(2);
        Main.check(local, 4);
        if (Main.fails == 0) { System.out.println("Java annotations OK"); }
        System.exit(Main.fails);
    }
}
