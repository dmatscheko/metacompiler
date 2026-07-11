/* Java subset self test (big 3): object orientation and functional programming.
 *
 * Leans on the signature OO/functional features of the subset:
 *   - an expression AST (Num/Add/Sub/Mul/Neg extending Expr) evaluated and
 *     pretty-printed purely by dynamic dispatch over a polymorphic object tree
 *   - a single-inheritance animal hierarchy (Animal -> Dog -> Puppy, Animal -> Bird)
 *     with method overriding, dynamic dispatch through `this`, a super.sound() call
 *     and dispatch through a supertyped array
 *   - records (Vec2, Pair) with auto-generated accessors
 *   - real closures and higher-order functions from java.util.function: fold/reduce
 *     with a BinaryOperator, in-place map and predicate counting with a Function,
 *     function composition, currying, a capturing factory (makeAdder) and a lambda
 *     produced by an instance method that captures and mutates `this`
 *   - method references: static (Main::inc), and a host one (Math::abs)
 * Every result is compared against an independently computed value; Main.main exits
 * with the failure count, so the run exits 0 exactly when everything agrees. **/

import java.util.function.Function;
import java.util.function.BinaryOperator;

// ----- expression AST: dynamic dispatch over a heterogeneous tree -----

class Expr {
    int eval() { return 0; }
    String show() { return "?"; }
}

class Num extends Expr {
    int v;
    Num(int v) { this.v = v; }
    int eval() { return this.v; }
    String show() { return "" + this.v; }
}

class Add extends Expr {
    Expr a;
    Expr b;
    Add(Expr a, Expr b) { this.a = a; this.b = b; }
    int eval() { return this.a.eval() + this.b.eval(); }
    String show() { return "(" + this.a.show() + "+" + this.b.show() + ")"; }
}

class Sub extends Expr {
    Expr a;
    Expr b;
    Sub(Expr a, Expr b) { this.a = a; this.b = b; }
    int eval() { return this.a.eval() - this.b.eval(); }
    String show() { return "(" + this.a.show() + "-" + this.b.show() + ")"; }
}

class Mul extends Expr {
    Expr a;
    Expr b;
    Mul(Expr a, Expr b) { this.a = a; this.b = b; }
    int eval() { return this.a.eval() * this.b.eval(); }
    String show() { return "(" + this.a.show() + "*" + this.b.show() + ")"; }
}

class Neg extends Expr {
    Expr e;
    Neg(Expr e) { this.e = e; }
    int eval() { return -this.e.eval(); }
    String show() { return "(-" + this.e.show() + ")"; }
}

// ----- animal hierarchy: overriding, super, dynamic dispatch -----

class Animal {
    String sound() { return "..."; }
    String speak() { return this.sound() + "!"; }             // calls the override
    int legs() { return 4; }
    String describe() { return this.speak() + " on " + this.legs() + " legs"; }
}

class Dog extends Animal {
    String sound() { return "woof"; }
}

class Puppy extends Dog {
    String sound() { return "yip-" + super.sound(); }         // super starts at Dog
}

class Bird extends Animal {
    String sound() { return "tweet"; }
    int legs() { return 2; }
}

// ----- records with auto-generated accessors -----

record Vec2(int x, int y) { }
record Pair(int lo, int hi) { }

// ----- a class whose instance method returns a closure capturing `this` -----

class Accum {
    int total;
    Accum() { this.total = 0; }
    Function<Integer, Integer> adder() {
        return x -> { this.total += x; return this.total; };
    }
    int get() { return this.total; }
}

public class Main {
    static int fails = 0;

    static void check(String name, int got, int want) {
        if (got != want) {
            System.out.println("FAIL " + name + ": got " + got + " want " + want);
            Main.fails++;
        }
    }

    static void checkB(String name, boolean got, boolean want) {
        Main.check(name, got ? 1 : 0, want ? 1 : 0);
    }

    static void checkS(String name, String got, String want) {
        if (!got.equals(want)) {
            System.out.println("FAIL " + name + ": got " + got + " want " + want);
            Main.fails++;
        }
    }

    // ----- higher-order helpers -----

    static int foldl(int[] a, int init, BinaryOperator<Integer> f) {
        int acc = init;
        for (var x : a) {
            acc = f.apply(acc, x);
        }
        return acc;
    }

    static void mapInPlace(int[] a, Function<Integer, Integer> f) {
        for (int i = 0; i < a.length; i++) {
            a[i] = f.apply(a[i]);
        }
    }

    static int countMatching(int[] a, Function<Integer, Integer> pred) {
        int c = 0;
        for (var x : a) {
            if (pred.apply(x) == 1) {
                c++;
            }
        }
        return c;
    }

    static Function<Integer, Integer> compose(Function<Integer, Integer> f, Function<Integer, Integer> g) {
        return x -> f.apply(g.apply(x));
    }

    static Function<Integer, Integer> makeAdder(int k) {
        return x -> x + k;                                    // captures k
    }

    static int dot(Vec2 a, Vec2 b) {
        return a.x() * b.x() + a.y() * b.y();
    }

    static int inc(int x) { return x + 1; }
    static int dbl(int x) { return x * 2; }

    public static void main(String[] args) {
        // ----- expression AST -----
        // ((2+3) * (4-1)) + (-5) = 5*3 - 5 = 10
        Expr tree = new Add(
            new Mul(new Add(new Num(2), new Num(3)), new Sub(new Num(4), new Num(1))),
            new Neg(new Num(5)));
        Main.check("ast eval", tree.eval(), 10);
        Main.checkS("ast show", tree.show(), "(((2+3)*(4-1))+(-5))");

        Expr deep = new Mul(new Num(2), new Mul(new Num(3), new Mul(new Num(4), new Num(5))));
        Main.check("ast deep eval", deep.eval(), 120);
        Main.checkS("ast deep show", deep.show(), "(2*(3*(4*5)))");

        // ----- animal hierarchy -----
        Animal[] zoo = new Animal[]{ new Dog(), new Puppy(), new Bird(), new Animal() };
        Main.checkS("dog describe", zoo[0].describe(), "woof! on 4 legs");
        Main.checkS("puppy super", zoo[1].describe(), "yip-woof! on 4 legs");
        Main.checkS("bird legs override", zoo[2].describe(), "tweet! on 2 legs");
        Main.checkS("base animal", zoo[3].describe(), "...! on 4 legs");
        String sounds = "";
        for (var an : zoo) {
            sounds = sounds + an.sound() + "|";
        }
        Main.checkS("polymorphic sounds", sounds, "woof|yip-woof|tweet|...|");

        // dispatch through the exact supertype, plus a direct Puppy super chain
        Animal upcast = new Puppy();
        Main.checkS("dispatch via supertype", upcast.sound(), "yip-woof");

        // ----- records -----
        Vec2 u = new Vec2(3, 4);
        Vec2 w = new Vec2(2, -1);
        Main.check("record accessor x", u.x(), 3);
        Main.check("record accessor y", u.y(), 4);
        Main.check("record dot product", Main.dot(u, w), 2);
        Pair span = new Pair(3, 9);
        Main.check("record pair width", span.hi() - span.lo(), 6);

        // ----- higher-order functions and closures -----
        int[] data = new int[]{1, 2, 3, 4, 5};
        Main.check("fold sum", Main.foldl(data, 0, (acc, x) -> acc + x), 15);
        Main.check("fold product", Main.foldl(data, 1, (acc, x) -> acc * x), 120);
        Main.check("fold max", Main.foldl(data, 0, (acc, x) -> acc > x ? acc : x), 5);
        Main.check("count evens", Main.countMatching(data, x -> x % 2 == 0 ? 1 : 0), 2);

        // map in place (mutating the literal), then re-fold
        Main.mapInPlace(data, x -> x * 2);               // {2,4,6,8,10}
        Main.check("mapped sum", Main.foldl(data, 0, (acc, x) -> acc + x), 30);
        Main.check("mapped first", data[0], 2);
        Main.check("mapped last", data[4], 10);

        // composition: inc(dbl(x))
        Function<Integer, Integer> incDbl = Main.compose(Main::inc, Main::dbl);
        Main.check("compose method refs", incDbl.apply(10), 21);
        Function<Integer, Integer> lamComp = Main.compose(x -> x + 1, y -> y * y);
        Main.check("compose lambdas", lamComp.apply(6), 37);

        // currying
        Function<Integer, Function<Integer, Integer>> adder = a -> (b -> a + b);
        Main.check("curried add", adder.apply(30).apply(12), 42);

        // capturing factory
        Function<Integer, Integer> add100 = Main.makeAdder(100);
        Main.check("closure add100", add100.apply(23), 123);
        Function<Integer, Integer> add7 = Main.makeAdder(7);
        Main.check("independent closure", add7.apply(35), 42);
        Main.check("first closure intact", add100.apply(0), 100);

        // method references
        Function<Integer, Integer> incRef = Main::inc;
        Main.check("static method ref", incRef.apply(41), 42);
        Function<Integer, Integer> absRef = Math::abs;
        Main.check("host method ref", absRef.apply(-7), 7);

        // an instance method returning a closure that captures and mutates `this`
        Accum acc = new Accum();
        Function<Integer, Integer> feed = acc.adder();
        Main.check("closure over this 1", feed.apply(10), 10);
        Main.check("closure over this 2", feed.apply(5), 15);
        Main.check("this mutated by closure", acc.get(), 15);

        if (Main.fails == 0) {
            System.out.println("Java big-3 (OO + functional) passed");
        }
        System.exit(Main.fails);
    }
}
