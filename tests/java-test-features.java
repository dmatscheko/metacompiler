/* Fast feature-matrix test for the Java interpreter (java-interpreter.abnf) and the
 * LLVM-IR compiler (java-to-llvm-ir.abnf). It replaces the four algorithm-themed
 * java-test-big-* stress tests: instead of large loops (Ackermann, sieves, ciphers)
 * every implemented construct is exercised with the SMALLEST program that can prove
 * it works - loops run 0, 1, 3 or 4 times, recursion stays below depth 6. A failed
 * check prints its id (so a diff pinpoints it) and Main.main exits with the failure
 * count; exit 0 and byte-identical output on all four legs (interpreter/compiler x
 * goja/-frozen) mean everything passed. **/

import java.util.function.Function;
import java.util.function.BinaryOperator;

class Counter {
    int value;
    int step = 1;                          // field initializer

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

    Point plus(Point other) {              // returns a fresh instance
        return new Point(this.x + other.x, this.y + other.y);
    }

    boolean samePoint(Point other) {
        return this.x == other.x && this.y == other.y;
    }
}

class Animal {
    int legs = 4;

    String name() {
        return "animal";
    }

    String describe() {                    // this.name() dispatches dynamically
        return this.name() + ":" + this.legs;
    }

    int base() {
        return 10;
    }
}

class Bird extends Animal {
    Bird() {
        this.legs = 2;                     // the implicit super() ran the field inits first
    }

    String name() {                        // overrides Animal.name
        return "bird";
    }

    int base() {                           // super.m() starts above the defining class
        return super.base() + 5;
    }
}

record Pair(int first, int second) { }

class Boom {
    int code;

    Boom(int c) {
        this.code = c;
    }
}

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

    Function<Integer, Integer> scaler() {  // a lambda that captures `this`
        return factor -> this.val * factor;
    }
}

public class Main {
    static int fails = 0;
    static int checks = 0;
    static int calls = 0;
    static int finRuns = 0;

    static void check(String id, boolean cond) {
        Main.checks++;
        if (!cond) {
            System.out.println("FAIL " + id);
            Main.fails++;
        }
    }

    static boolean bump() {
        Main.calls++;
        return true;
    }

    // ----- functions: early return, recursion, mutual recursion -----
    static String grade(int n) {
        if (n > 10) { return "big"; }
        else if (n > 5) { return "mid"; }
        else { return "small"; }
    }
    static int sign(int n) {
        if (n < 0) { return -1; }          // early return
        return 1;
    }
    static int fib(int n) {
        if (n < 2) { return n; }
        return Main.fib(n - 1) + Main.fib(n - 2);
    }
    static boolean isEven(int n) {
        return n == 0 ? true : Main.isOdd(n - 1);
    }
    static boolean isOdd(int n) {
        return n == 0 ? false : Main.isEven(n - 1);
    }

    // ----- switch helpers -----
    static int classify(int x) {           // fallthrough, stacked labels, default
        int r = 0;
        switch (x) {
        case 0:
            r = 100;
            break;
        case 1:                            // stacked labels
        case 2:
            r = 12;
            break;
        case 3:
            r = 3;                         // falls through into case 4
        case 4:
            r += 40;
            break;
        default:
            r = -1;
        }
        return r;
    }
    static String dayKind(String d) {      // switch on String, return from a case
        switch (d) {
        case "sat":
        case "sun":
            return "weekend";
        default:
            return "workday";
        }
    }

    // ----- higher-order helpers -----
    static int applyOnce(Function<Integer, Integer> f, int x) {
        return f.apply(x);
    }
    static int sumOver(int[] arr, Function<Integer, Integer> f) {
        int s = 0;
        for (var v : arr) {
            s += f.apply(v);
        }
        return s;
    }
    static Function<Integer, Integer> makeAdder(int n) {   // capturing factory
        return x -> x + n;
    }

    // ----- exceptions -----
    static int risky(int n) {
        if (n > 3) { throw new Boom(n); }  // unwinds out of the call
        return n * 2;
    }
    static int rethrowNested() {
        try {
            try { throw new Boom(1); } catch (Exception e) { throw new Boom(e.code + 1); }
        } catch (Exception e2) {
            return e2.code;
        }
    }
    static String retAcrossTry() {
        try { return "from-try"; } finally { Main.finRuns += 1; }
    }
    static int retOutOfCatch(int n) {
        try {
            if (n > 0) { return n * 10; }  // return out of the try
            throw new Boom(0);
        } catch (Exception e) {
            return -1;                     // return out of the catch
        } finally {
            Main.finRuns += 1;             // runs on both paths
        }
    }
    static int nestedReturn() {
        try {
            try { return 9; } finally { }
        } finally { }
        return 0;
    }
    static int retInFinally() {
        try { return 1; } finally { return 2; }   // the finally's return overrides
    }
    static String finCancelsThrow() {
        try { throw new Boom(9); } finally { return "fin"; }   // cancels the pending throw
    }
    static int breakInFinally() {
        int i = 0;
        while (true) {
            i = i + 1;
            try { i = i + 10; } finally { break; }
        }
        return i;
    }
    static int continueInFinally() {
        int sum = 0;
        for (int i = 0; i < 3; i++) {
            try { if (i == 1) { throw new Boom(i); } } finally { continue; }
        }
        return sum;                        // the += after try is never written, sum stays 0
    }
    static int loopBreakOutOfTry() {
        int sum = 0;
        for (int i = 0; i < 6; i++) {
            try {
                if (i == 3) { break; }
                sum = sum + i;
            } finally { }
        }
        return sum;                        // 0+1+2 = 3
    }
    static int loopContinueOutOfTry() {
        int sum = 0;
        for (int i = 0; i < 4; i++) {
            try {
                if (i == 2) { continue; }
                sum = sum + i;
            } catch (Exception e) { }
        }
        return sum;                        // 0+1+3 = 4
    }

    // ----- everything combined in one small pipeline (3-element data flow) -----
    static String transform(int[] list) {
        String out = "";
        for (var n : list) {
            try {
                if (n < 0) { throw new Boom(n); }
                out = out + (n % 2 == 0 ? "e" : "o") + n;
            } catch (Exception e) {
                out = out + "x";
            }
        }
        return out;
    }

    public static void main(String[] args) {
        // ----- numbers, arithmetic, precedence -----
        Main.check("arith-precedence", 2 + 3 * 4 == 14);
        Main.check("arith-paren", (2 + 3) * 4 == 20);
        Main.check("arith-unary-minus", -3 + 5 == 2);
        Main.check("arith-unary-plus", +5 == 5);
        Main.check("arith-div-trunc", 7 / 2 == 3);
        Main.check("arith-div-neg", -7 / 2 == -3);
        Main.check("arith-mod", 7 % 3 == 1);
        Main.check("arith-mod-neg", -7 % 3 == -1);
        Main.check("arith-chain", 20 - 5 - 3 == 12);
        int cx = 5;
        cx += 3;
        cx -= 2;
        cx *= 4;
        cx /= 6;
        cx %= 3;
        Main.check("arith-compound", cx == 1);
        int pi = 5;
        int a1 = pi++;                     // postfix yields the old value
        int a2 = ++pi;                     // prefix yields the new value
        Main.check("arith-incdec", a1 == 5 && a2 == 7 && pi == 7);
        int pd = 5;
        int d1 = pd--;
        --pd;
        Main.check("arith-decrement", d1 == 5 && pd == 3);

        // ----- comparison, equality, logic -----
        Main.check("cmp-ops", 5 > 3 && 3 >= 3 && 2 < 3 && 2 <= 2 && 1 != 2);
        Main.calls = 0;
        boolean noRun = false && Main.bump();
        Main.check("logic-and-skipped", Main.calls == 0 && !noRun);
        boolean skipRun = true || Main.bump();
        Main.check("logic-or-skipped", Main.calls == 0 && skipRun);
        boolean oneRun = Main.bump() && true;
        Main.check("logic-ran-once", Main.calls == 1 && oneRun);
        Main.check("logic-not", !(2 == 3) && !false);
        Main.check("ternary", (5 > 3 ? "a" : "b").equals("a") && (5 < 3 ? "a" : "b").equals("b"));
        Main.check("ternary-nested", (5 > 3 ? (2 > 1 ? 1 : 2) : 3) == 1);
        var vt = 3 > 2 ? 10 : 20;          // var local
        Main.check("var-local", vt == 10);

        // ----- strings -----
        Main.check("str-concat", ("foo" + "bar").equals("foobar"));
        String sa = "Hello";
        sa += ", world";
        Main.check("str-concat-assign", sa.equals("Hello, world"));
        Main.check("str-int-concat", ("n=" + 42).equals("n=42") && (42 + "x").equals("42x") && (1 + 2 + "x").equals("3x"));
        Main.check("str-length", "hello".length() == 5 && "".length() == 0);
        Main.check("str-unicode-len", "héllo".length() == 5);
        Main.check("str-charat", "abc".charAt(0).equals("a") && "abc".charAt(2).equals("c"));
        Main.check("str-substring", "hello".substring(1, 3).equals("el") && "hello".substring(3).equals("lo"));
        Main.check("str-indexof", "abcabc".indexOf("b") == 1 && "abc".indexOf("z") == -1);
        Main.check("str-equals", "abc".equals("abc") && !"abc".equals("abd"));
        Main.check("str-isempty", "".isEmpty() && !"x".isEmpty());
        Main.check("str-escapes", "a\tb".length() == 3 && "a\nb".length() == 3 && "\\".length() == 1 && "\"".length() == 1);

        // ----- control flow: if / while / do-while / for -----
        Main.check("if-elseif-else", Main.grade(11).equals("big") && Main.grade(7).equals("mid") && Main.grade(1).equals("small"));
        int w0 = 0;
        while (w0 > 0) { w0 = w0 - 1; }    // runs zero times
        Main.check("while-zero", w0 == 0);
        int w3 = 0;
        while (w3 < 3) { w3 = w3 + 1; }    // runs three times
        Main.check("while-three", w3 == 3);
        int dw = 0;
        do { dw = dw + 1; } while (false); // body runs exactly once
        Main.check("do-while-once", dw == 1);
        int forSum = 0;
        for (int i = 1; i <= 3; i++) { forSum += i; }
        Main.check("for-basic", forSum == 6);
        String brk = "";
        for (int i = 0; i < 6; i++) {
            if (i == 2) { break; }
            brk = brk + i;
        }
        Main.check("for-break", brk.equals("01"));
        String cont = "";
        for (int i = 0; i < 4; i++) {
            if (i % 2 == 1) { continue; }
            cont = cont + i;
        }
        Main.check("for-continue", cont.equals("02"));
        String nested = "";
        for (int oi = 0; oi < 2; oi++) {
            for (int ii = 0; ii < 3; ii++) {
                if (ii == 1) { break; }    // inner break must not end the outer loop
                nested = nested + oi + ii;
            }
        }
        Main.check("nested-break", nested.equals("0010"));

        // ----- switch: match, stacked labels, fallthrough, default -----
        Main.check("switch-match", Main.classify(0) == 100);
        Main.check("switch-stacked", Main.classify(2) == 12);
        Main.check("switch-fallthrough", Main.classify(3) == 43);
        Main.check("switch-late-entry", Main.classify(4) == 40);
        Main.check("switch-default", Main.classify(9) == -1);
        Main.check("switch-string", Main.dayKind("sun").equals("weekend") && Main.dayKind("tue").equals("workday"));
        int sc = 0;
        for (int n = 0; n < 4; n++) {      // continue skips the tail, break only the switch
            switch (n % 3) {
            case 0: continue;
            case 1: sc += 10; break;
            default: sc += 1;
            }
            sc += 100;
        }
        Main.check("switch-in-loop", sc == 211);

        // ----- arrays -----
        int[] arr = new int[3];
        Main.check("arr-default", arr[0] == 0 && arr[2] == 0);
        Main.check("arr-new-length", arr.length == 3);
        arr[0] = 10;
        arr[1] = 20;
        arr[2] = 30;
        Main.check("arr-store", arr[0] == 10 && arr[2] == 30);
        arr[1] += 5;
        Main.check("arr-compound-elem", arr[1] == 25);
        int[] lit = new int[]{3, 1, 4};
        Main.check("arr-literal", lit.length == 3 && lit[2] == 4);
        int esum = 0;
        for (var v : lit) { esum += v; }
        Main.check("arr-enhanced-for", esum == 8);
        String[] words = new String[]{"aa", "b", "ccc"};
        int wlen = 0;
        for (var w : words) { wlen += w.length(); }
        Main.check("arr-strings", words.length == 3 && wlen == 6);
        Animal[] zoo = new Animal[]{new Bird(), new Animal()};
        String zooNames = "";
        for (var an : zoo) { zooNames = zooNames + an.name() + ","; }
        Main.check("arr-dispatch", zooNames.equals("bird,animal,"));

        // ----- classes, inheritance, records -----
        Counter ctr = new Counter(10);
        Main.check("class-field-init", ctr.step == 1 && ctr.value == 10);
        Main.check("class-method", ctr.next() == 11);
        ctr.setStep(5);
        Main.check("class-setter", ctr.next() == 16 && ctr.value == 16);
        Main.check("class-static-method", Counter.twice(21) == 42);
        Point p = new Point(3, -4);
        Point q = p.plus(new Point(1, 1));
        Main.check("class-returns-instance", q.x * 100 + q.y == 397);
        Main.check("obj-value-equality", q.samePoint(new Point(4, -3)));
        Main.check("obj-identity", p == p && !(p == new Point(3, -4)));
        Point none = null;
        Main.check("obj-null", none == null);
        Animal an = new Animal();
        Main.check("class-this-dispatch", an.describe().equals("animal:4"));
        Bird bird = new Bird();
        Main.check("class-override", bird.describe().equals("bird:2"));
        Main.check("class-super-call", bird.base() == 15);
        Main.check("class-inherited-field", bird.legs == 2);
        Animal upcast = bird;
        Main.check("class-upcast-dispatch", upcast.name().equals("bird"));
        Pair pr = new Pair(6, 7);
        Main.check("record-accessors", pr.first() == 6 && pr.first() * pr.second() == 42);

        // ----- statics and recursion -----
        Main.check("fn-early-return", Main.sign(-9) == -1 && Main.sign(9) == 1);
        Main.check("fn-recursion", Main.fib(6) == 8);
        Main.check("fn-mutual-recursion", Main.isEven(4) && Main.isOdd(5));
        Main.check("builtin-math", Math.abs(-7) == 7 && Math.max(3, 9) == 9 && Math.min(3, 9) == 3);

        // ----- lambdas, closures, method references -----
        Function<Integer, Integer> dbl = x -> x * 2;                 // single id, expr body
        Main.check("lambda-expr", dbl.apply(21) == 42);
        Function<Integer, Integer> inc = (x) -> x + 1;               // parenthesised untyped
        Main.check("lambda-paren", inc.apply(41) == 42);
        Function<Integer, Integer> half = (int x) -> x / 2;          // typed param
        Main.check("lambda-typed", half.apply(84) == 42);
        Function<Integer, Integer> sq = x -> { return x * x; };      // block body + return
        Main.check("lambda-block", sq.apply(9) == 81);
        BinaryOperator<Integer> plus = (a, b) -> a + b;              // two params
        Main.check("lambda-two-args", plus.apply(19, 23) == 42);
        int base = 100;
        Function<Integer, Integer> addBase = n -> n + base;          // captured local
        Main.check("closure-capture", addBase.apply(23) == 123);
        Main.check("lambda-as-arg", Main.applyOnce(y -> y + 5, 37) == 42);
        int[] nums = new int[]{1, 2, 3};
        Main.check("lambda-over-array", Main.sumOver(nums, n -> n * n) == 14);
        Function<Integer, Function<Integer, Integer>> adder = a -> (b -> a + b);
        Main.check("lambda-curried", adder.apply(30).apply(12) == 42);
        Function<Integer, Integer> add10 = Main.makeAdder(10);
        Function<Integer, Integer> add100 = Main.makeAdder(100);
        Main.check("closure-independent", add10.apply(5) == 15 && add100.apply(5) == 105);
        int[] acc = new int[]{0};
        Runnable bump7 = () -> { acc[0] = acc[0] + 7; };             // void block lambda
        bump7.run();
        bump7.run();
        Main.check("lambda-void-side-effect", acc[0] == 14);
        Supplier<Integer> supply = () -> 42;                         // SAM name is irrelevant
        Main.check("sam-get", supply.get() == 42);
        Predicate<Integer> isBig = v -> v > 10;
        Main.check("sam-test", isBig.test(50) && !isBig.test(3));
        Box six = new Box(6);
        Function<Integer, Integer> times = six.scaler();             // lambda capturing this
        Main.check("closure-captures-this", times.apply(7) == 42);
        Function<Integer, Integer> tri = Box::triple;                // static method ref
        Main.check("methodref-static", tri.apply(14) == 42);
        Box forty = new Box(40);
        Function<Integer, Integer> addForty = forty::addTo;          // bound method ref
        Main.check("methodref-bound", addForty.apply(2) == 42);
        Function<Box, Integer> getter = Box::get;                    // unbound: receiver is arg 0
        Main.check("methodref-unbound", getter.apply(new Box(42)) == 42);
        Function<Integer, Integer> absRef = Math::abs;               // host static
        Main.check("methodref-host", absRef.apply(-7) == 7);
        Main.check("dispatch-next-to-sam", forty.addTo(2) == 42);

        // ----- exceptions: throw / catch / finally / control flow -----
        String exOrder = "";
        int exCode = 0;
        try {
            exOrder = exOrder + "t";
            throw new Boom(5);
        } catch (Exception e) {
            exOrder = exOrder + "c";
            exCode = e.code;
        } finally {
            exOrder = exOrder + "f";
        }
        Main.check("try-throw-catch-finally", exOrder.equals("tcf") && exCode == 5);
        String noThrow = "";
        try { noThrow = noThrow + "t"; } catch (Exception e) { noThrow = noThrow + "c"; } finally { noThrow = noThrow + "f"; }
        Main.check("try-no-throw", noThrow.equals("tf"));
        int caught = -1;
        try {
            Main.risky(5);
            caught = -2;                   // not reached
        } catch (Exception e) {
            caught = e.code;
        }
        Main.check("throw-unwinds-call", caught == 5);
        Main.check("throw-no-throw-path", Main.risky(2) == 4);
        Main.check("rethrow", Main.rethrowNested() == 2);
        Main.check("return-across-try", Main.retAcrossTry().equals("from-try") && Main.finRuns == 1);
        Main.check("return-out-of-catch", Main.retOutOfCatch(4) == 40 && Main.retOutOfCatch(-1) == -1 && Main.finRuns == 3);
        Main.check("nested-return", Main.nestedReturn() == 9);
        Main.check("return-in-finally", Main.retInFinally() == 2);
        Main.check("finally-cancels-throw", Main.finCancelsThrow().equals("fin"));
        Main.check("break-in-finally", Main.breakInFinally() == 11);
        Main.check("continue-in-finally", Main.continueInFinally() == 0);
        Main.check("loop-break-out-of-try", Main.loopBreakOutOfTry() == 3);
        Main.check("loop-continue-out-of-try", Main.loopContinueOutOfTry() == 4);

        // ----- bitwise and shift operators (Java int semantics) -----
        Main.check("bit-and-or-xor", (5 & 3) == 1 && (5 | 3) == 7 && (5 ^ 3) == 6);
        Main.check("bit-not", ~5 == -6);
        Main.check("shifts", (1 << 4) == 16 && (-16 >> 2) == -4 && (-8 >>> 28) == 15);
        Main.check("bit-precedence", (7 & 3 | 4 ^ 1) == (3 | 5));
        Main.check("bool-non-short", (true & true) && (true ^ true) == false && (false | true));
        int bm = 12;
        bm &= 10;
        bm |= 1;
        bm ^= 3;
        bm <<= 2;
        bm >>= 1;
        Main.check("bit-compound", bm == 20);
        int bu = -1;
        bu >>>= 28;
        Main.check("bit-compound-ushr", bu == 15);

        // ----- labeled statements -----
        int labHits = 0;
        outer:
        for (int li = 0; li < 3; li++) {
            for (int lj = 0; lj < 3; lj++) {
                if (lj == 1) { continue outer; }
                if (li == 2) { break outer; }
                labHits = labHits + 1;
            }
        }
        Main.check("labeled-loop", labHits == 2);
        int labReached = 0;
        lblock: { labReached = 1; if (labReached == 1) { break lblock; } labReached = 2; }
        Main.check("labeled-block", labReached == 1);

        // ----- arrow switch (multi-label, block arm, no fallthrough) -----
        int arrowR = 0;
        switch (2) {
            case 1, 2 -> arrowR = 10;
            case 3 -> { arrowR = 20; arrowR += 1; }
            default -> arrowR = -1;
        }
        Main.check("arrow-switch", arrowR == 10);
        String arrowKind;
        switch ("sun") {
            case "sat", "sun" -> arrowKind = "weekend";
            default -> arrowKind = "workday";
        }
        Main.check("arrow-switch-string", arrowKind.equals("weekend"));

        // ----- numeric literal forms -----
        Main.check("lit-hex-bin-oct", 0xFF == 255 && 0b1010 == 10 && 017 == 15);
        Main.check("lit-underscore", 1_000_000 == 1000000 && 0xFF_FF == 65535);
        long litBig = 4_000_000_000L;
        Main.check("lit-long", litBig / 2L == 2000000000L && 7l == 7L);
        Main.check("lit-float-forms", 1e3 == 1000.0 && 2.5e-2 == 0.025 && .5 == 0.5 && 5. == 5.0);
        Main.check("lit-suffix", 1.5f + 1.5F == 3.0f && 2.5d * 2 == 5.0D);
        Main.check("lit-hex-float", 0x1p4 == 16.0);

        // ----- instanceof with pattern binding -----
        Object iofObj = "abcd";
        Main.check("iof-type-test", iofObj instanceof String && !(iofObj instanceof Boolean));
        if (iofObj instanceof String iofS) {
            Main.check("iof-binding", iofS.length() == 4);
        } else {
            Main.check("iof-binding", false);
        }
        Main.check("iof-int", 7 instanceof Integer);

        // ----- everything combined -----
        Main.check("combined-pipeline", Main.transform(new int[]{1, 2, -3}).equals("o1e2x"));

        System.out.println("features: " + Main.checks + " checks, " + Main.fails + " failures");
        System.exit(Main.fails);
    }
}
