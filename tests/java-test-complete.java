/* Java subset self test: lambdas, method references and closures.
 * Exercises the features completed on top of the base Java subset:
 *   - lambdas as real closures (js_closure capture, js_call to invoke)
 *   - every lambda shape: x -> e, (x) -> e, (int x) -> e, (a, b) -> e,
 *     () -> {...}, x -> { return e; }
 *   - capture of locals, of `this`, and lambdas that return lambdas
 *   - a functional-interface call site just invokes the value, whatever the
 *     single abstract method is named (apply / run / get / test / call ...)
 *   - method references: static (Class::m), bound (obj::m), unbound (Type::m)
 *     and a host one (Math::abs)
 *   - Java 10 `var` locals and the enhanced-for over an array, driving a lambda
 * Main.main counts failed checks and exits with that count, so the metacompiler
 * run exits with 0 exactly when everything works. **/

import java.util.function.Function;
import java.util.function.BinaryOperator;

class Box {
    int val;

    Box(int v) {
        this.val = v;
    }

    int get() {
        return this.val;
    }

    int addTo(int x) {
        return this.val + x;
    }

    static int triple(int x) {
        return x * 3;
    }

    // A lambda defined in an instance method captures `this`.
    Function<Integer, Integer> scaler() {
        return factor -> this.val * factor;
    }
}

public class Main {
    static int fails = 0;

    static void check(String name, int got, int want) {
        if (got != want) {
            System.out.println("FAIL " + name + ": got " + got + " want " + want);
            Main.fails++;
        }
    }

    // Functional-interface parameters: the value is just called.
    static int applyOnce(Function<Integer, Integer> f, int x) {
        return f.apply(x);
    }

    static int sumOver(int[] arr, Function<Integer, Integer> f) {
        int s = 0;
        for (var v : arr) {                 // var local + enhanced-for + lambda
            s += f.apply(v);
        }
        return s;
    }

    public static void main(String[] args) {
        // ----- lambda shapes -----
        Function<Integer, Integer> dbl = x -> x * 2;                 // single id, expr body
        Main.check("single-id expr lambda", dbl.apply(21), 42);

        Function<Integer, Integer> inc = (x) -> x + 1;               // parenthesised untyped
        Main.check("paren untyped lambda", inc.apply(41), 42);

        Function<Integer, Integer> half = (int x) -> x / 2;          // typed param
        Main.check("typed param lambda", half.apply(84), 42);

        Function<Integer, Integer> sq = x -> { return x * x; };      // block body + return
        Main.check("block-body lambda", sq.apply(9), 81);

        BinaryOperator<Integer> add = (a, b) -> a + b;               // two params
        Main.check("two-arg lambda", add.apply(19, 23), 42);

        // ----- closures -----
        int base = 100;                                              // captured local
        Function<Integer, Integer> addBase = n -> n + base;
        Main.check("capture local", addBase.apply(23), 123);

        Main.check("lambda as argument", Main.applyOnce(y -> y + 5, 37), 42);

        int[] nums = new int[]{1, 2, 3, 4};
        Main.check("lambda over enhanced-for", Main.sumOver(nums, n -> n * n), 30);

        // A lambda returning a lambda: nested capture (currying).
        Function<Integer, Function<Integer, Integer>> adder = a -> (b -> a + b);
        Main.check("curried lambda", adder.apply(30).apply(12), 42);

        // A void block lambda mutating a captured (effectively final) array.
        int[] acc = new int[]{0};
        Runnable bump = () -> { acc[0] = acc[0] + 7; };
        bump.run();
        bump.run();
        Main.check("void lambda side effect", acc[0], 14);

        // The single abstract method name is irrelevant: the value is just called.
        Supplier<Integer> supply = () -> 42;
        Main.check("SAM name get()", supply.get(), 42);
        Predicate<Integer> isBig = v -> v > 10;
        Main.check("SAM name test() true", isBig.test(50) ? 1 : 0, 1);
        Main.check("SAM name test() false", isBig.test(3) ? 1 : 0, 0);

        // A lambda that captures `this`, produced by an instance method.
        Box six = new Box(6);
        Function<Integer, Integer> times = six.scaler();
        Main.check("capture this", times.apply(7), 42);

        // ----- method references -----
        Function<Integer, Integer> tri = Box::triple;                // static
        Main.check("static method ref", tri.apply(14), 42);

        Box forty = new Box(40);
        Function<Integer, Integer> getForty = forty::get;            // bound, ignores arg
        Main.check("bound method ref", getForty.apply(0), 40);

        Function<Integer, Integer> addForty = forty::addTo;          // bound, uses arg
        Main.check("bound method ref with arg", addForty.apply(2), 42);

        Function<Box, Integer> getter = Box::get;                    // unbound: receiver is arg 0
        Main.check("unbound method ref", getter.apply(new Box(42)), 42);

        Function<Integer, Integer> absRef = Math::abs;               // host static
        Main.check("host method ref", absRef.apply(-7), 7);

        // Ordinary method dispatch still works next to functional-interface calls.
        Main.check("ordinary dispatch still works", forty.addTo(2), 42);

        if (Main.fails == 0) {
            System.out.println("Java lambda/closure self test passed");
        }
        System.exit(Main.fails);
    }
}
