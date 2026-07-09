/* Java subset self test.
 * Exercises classes, records, statics, arrays, strings and control flow.
 * Main.main counts failed checks and exits with that count, so the
 * metacompiler run exits with 0 exactly when everything works. **/

class Counter {
    int value;
    int step = 1;

    Counter(int start) {
        this.value = start;
    }

    int next() {
        this.value += this.step;
        return this.value;
    }

    void setStep(int s) {
        this.step = s;
    }

    static int twice(int x) {
        return x * 2;
    }
}

class Point {
    int x;
    int y;

    Point(int x, int y) {
        this.x = x;
        this.y = y;
    }

    int manhattan() {
        return Math.abs(this.x) + Math.abs(this.y);
    }

    Point plus(Point other) {
        return new Point(this.x + other.x, this.y + other.y);
    }

    boolean equalsPoint(Point other) {
        return this.x == other.x && this.y == other.y;
    }
}

class Animal {
    int legs = 4;

    String name() {
        return "animal";
    }

    String describe() {                 // this.name() dispatches dynamically
        return this.name() + ":" + this.legs;
    }

    int base() {
        return 10;
    }
}

class Bird extends Animal {
    Bird() {
        this.legs = 2;                  // the implicit super() ran the field inits first
    }

    String name() {                     // overrides Animal.name
        return "bird";
    }

    int base() {                        // super.m() starts above the defining class
        return super.base() + 5;
    }
}

record Pair(int first, int second) { }

public class Main {
    static int fails = 0;
    static int calls = 0;

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

    static boolean bump() {
        Main.calls++;
        return true;
    }

    static int fib(int n) {
        if (n < 2) { return n; }
        return Main.fib(n - 1) + Main.fib(n - 2);
    }

    public static void main(String[] args) {
        // arithmetic (int is 32 bit, / truncates)
        Main.check("precedence", 1 + 2 * 3, 7);
        Main.check("division", 7 / 2, 3);
        Main.check("negative division", -7 / 2, -3);
        Main.check("modulo", 7 % 3, 1);
        Main.check("unary", -(-5), 5);

        // var and ternary
        var t = 3 > 2 ? 10 : 20;
        Main.check("var and ternary", t, 10);

        // strings
        String s = "Hello";
        s += ", world";
        Main.checkS("concat assign", s, "Hello, world");
        Main.check("length", s.length(), 12);
        Main.checkS("one char substring", s.substring(4, 5), "o");
        Main.check("indexOf", s.indexOf("world"), 7);
        Main.checkS("substring", s.substring(0, 5), "Hello");
        Main.checkB("isEmpty", "".isEmpty(), true);
        Main.checkS("int in concat", "n=" + 42, "n=42");
        Main.checkB("equals", "abc".equals("abc"), true);

        // short circuit
        Main.calls = 0;
        boolean b1 = false && Main.bump();
        Main.check("and skipped", Main.calls, 0);
        boolean b2 = true || Main.bump();
        Main.check("or skipped", Main.calls, 0);
        boolean b3 = Main.bump() && true;
        Main.check("ran once", Main.calls, 1);
        Main.checkB("bool results", b1 || b2 && b3, true);

        // control flow
        int sum = 0;
        for (int i = 1; i <= 10; i++) { sum += i; }
        Main.check("for", sum, 55);

        int w = 0;
        while (w < 5) { w++; }
        Main.check("while", w, 5);

        int dc = 0;
        do { dc++; } while (false);
        Main.check("do while", dc, 1);

        int odd = 0;
        for (int j = 0; j < 100; j++) {
            if (j % 2 == 0) { continue; }
            if (j > 10) { break; }
            odd += j;
        }
        Main.check("break continue", odd, 25);

        // arrays
        int[] arr = new int[5];
        Main.check("array default", arr[0], 0);
        for (int i = 0; i < arr.length; i++) { arr[i] = i * i; }
        Main.check("array store", arr[4], 16);
        arr[2] += 10;
        Main.check("compound elem", arr[2], 14);

        int[] lit = new int[]{3, 1, 4, 1, 5};
        Main.check("array literal", lit.length, 5);
        int esum = 0;
        for (var v : lit) { esum += v; }
        Main.check("enhanced for", esum, 14);

        // classes and objects
        Counter c = new Counter(10);
        Main.check("field init", c.step, 1);
        Main.check("method", c.next(), 11);
        c.setStep(5);
        Main.check("setter", c.next(), 16);
        Main.check("field read", c.value, 16);
        Main.check("static method", Counter.twice(21), 42);

        Point p = new Point(3, -4);
        Main.check("manhattan", p.manhattan(), 7);
        Point q = p.plus(new Point(1, 1));
        Main.check("returned object", q.x * 100 + q.y, 397);
        Main.checkB("value equality", q.equalsPoint(new Point(4, -3)), true);
        Main.checkB("identity ==", p == p, true);
        Main.checkB("identity != for twins", p == new Point(3, -4), false);

        // null
        Point none = null;
        Main.checkB("null check", none == null, true);

        // records
        Pair pr = new Pair(6, 7);
        Main.check("record accessor", pr.first(), 6);
        Main.check("record product", pr.first() * pr.second(), 42);

        // inheritance
        Animal an = new Animal();
        Main.checkS("describe", an.describe(), "animal:4");
        Bird bird = new Bird();
        Main.checkS("override and dispatch", bird.describe(), "bird:2");
        Main.check("super call", bird.base(), 15);
        Main.check("inherited field", bird.legs, 2);
        Animal upcast = bird;
        Main.checkS("dispatch via supertype", upcast.name(), "bird");

        // statics and recursion
        Main.check("fib", Main.fib(10), 55);

        if (Main.fails == 0) {
            System.out.println("Java subset self test passed");
        }
        System.exit(Main.fails);
    }
}
