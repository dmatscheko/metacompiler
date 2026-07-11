/* Java widened-surface test.
 * Exercises constructs added on top of the base subset: generics (parsed and
 * ignored), annotations, casts (identity), interfaces, enums, annotation types,
 * nested types, lambdas, method references and anonymous classes.
 *
 * Under a DEFAULT run this ABORTS at the first not-implemented construct (the
 * top-level interface below) with a clean file:line message. Under
 * -warn-unsupported every not-implemented construct warns and the genuinely
 * lowered parts (casts, generics) run; Main.main self checks them and exits with
 * the failure count (0 when all pass). (try/catch/finally + throw are now
 * implemented - see java-test-try.java.) **/

import java.util.List;

// Accepted structurally, reported as not implemented (no interface model).
interface Greeter {
    String greet(String who);
}

// Accepted structurally, reported as not implemented (no enum model).
enum Color { RED, GREEN, BLUE }

// Accepted structurally, reported as not implemented (annotation type).
@interface Marker { }

// A generic class header with an implements clause: the type parameter and the
// implements clause are parsed and ignored; the class itself is built normally.
class Box<T> implements Greeter {
    int value;

    Box(int v) {
        this.value = v;
    }

    @Override
    public String greet(String who) {          // annotation + implements: ignored
        return "hi " + who;
    }

    // A nested type inside a class body: accepted, not implemented.
    static class Inner { }

    int get() throws Exception {                // throws clause: parsed and ignored
        return this.value;
    }
}

// A subclass with an explicit super(args) call: the call is accepted and reported as
// not implemented (the implicit no-argument super already ran); the subclass's own
// field initializer still runs.
class Big extends Box {
    int extra = 100;

    Big() {
        super(7);
    }
}

class MyErr {                                   // a plain class used as a throwable
    int code = 1;
}

public class Main {
    static int fails = 0;

    static void check(String name, int got, int want) {
        if (got != want) {
            System.out.println("FAIL " + name + ": got " + got + " want " + want);
            Main.fails++;
        }
    }
    static void checkS(String name, String got, String want) {
        if (!got.equals(want)) {
            System.out.println("FAIL " + name + ": got " + got + " want " + want);
            Main.fails++;
        }
    }

    // A varargs parameter with an annotation: the annotation and the `...` are
    // parsed (varargs packing itself is not modelled, so this is not called for
    // its value; the method reference below just needs it to exist).
    static int firstOr0(@Deprecated int... xs) {
        return 0;
    }

    @SuppressWarnings("unchecked")              // annotation with an argument
    public static void main(String[] args) {
        // casts are identity
        int ci = (int) 7;
        Main.check("prim cast identity", ci, 7);
        Object obj = "hi";                      // Object type parsed, dynamic value
        String cs = (String) obj;
        Main.checkS("ref cast identity", cs, "hi");
        Main.check("cast inside expr", 10 - (int) 3, 7);

        // (a) - b stays subtraction, not a cast of -b
        int a5 = 5;
        int b2 = 2;
        Main.check("paren minus is subtraction", (a5) - b2, 3);

        // generic type annotations are parsed and ignored (value stays dynamic)
        List<Integer> nums = null;
        Main.check("generic typed null", nums == null ? 1 : 0, 1);
        Box<Integer> box = new Box<Integer>(42); // diamond-free generic new
        Main.check("generic new + method", box.get(), 42);
        Main.checkS("implements method", box.greet("bob"), "hi bob");
        Big big = new Big();                    // super(7) accepted; own field init runs
        Main.check("subclass own field init", big.extra, 100);

        // (try/catch/finally and throw are now implemented - see java-test-try.java.)

        // lambdas, method references and anonymous classes: parsed, not implemented
        var f = (int x, int y) -> x + y;
        var g = Main::firstOr0;
        var anon = new MyErr() { };

        if (Main.fails == 0) {
            System.out.println("Java widened-surface test passed");
        }
        System.exit(Main.fails);
    }
}
