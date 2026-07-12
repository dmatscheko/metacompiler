// Full-syntax test: Java (Java SE 21 core grammar).
//
// This file belongs to the SECOND test group (./test.sh --full): it is NOT part
// of the default matrix. The goal of the metacompiler is to support the full
// languages; this file is the ratchet that measures how far the java grammars
// are. It walks the whole practical Java 17-21 syntax, one self-contained
// SECTION per language area. The --full runner runs the file, and whenever a
// grammar aborts it removes the section around the error and retries - so the
// report lists every unsupported section, not just the first.
//
// Conventions (shared by every *-test-full.* file):
//   - prologue (before the first SECTION marker): class Main with the check
//     helper and main - Main comes first so `java tests/java-test-full.java`
//     (the single-file source launcher) picks it as the class to run
//   - each section: '// ===== SECTION <nn>: <name> =====', a self-contained
//     group of top-level types, no references to other sections
//   - main calls each section via a line tagged 'SECTION-CALL <nn>'; the calls
//     are S<nn>.run()-qualified because the subset resolves statics by class
//   - main prints the summary line 'full: <checks> checks, <failures> failures'
//     and exits with the failure count (exit 0 == full support, verified)
//
// Deliberately out of scope (not core syntax, or unrunnable in this harness):
// packages and imports - and with them the whole standard library beyond the
// java.lang implicits the feature test already uses (String, Math, System and
// friends); the functional interfaces here are declared locally instead of
// importing java.util.function. Also out: modules (module-info), threads (and
// synchronized), reflection, javadoc.
//
// Hand-written for the metacompiler project (Apache-2.0, no copied test-suite
// code), organized after the Java Language Specification (SE 21) with the
// ANTLR grammars-v4 java grammar as a coverage checklist.

class Main {
    static int failures = 0;
    static int checks = 0;

    static void check(String id, boolean cond) {
        Main.checks++;
        if (!cond) {
            System.out.println("FAIL " + id);
            Main.failures++;
        }
    }

    public static void main(String[] args) {
        S01.run(); // SECTION-CALL 01
        S02.run(); // SECTION-CALL 02
        S03.run(); // SECTION-CALL 03
        S04.run(); // SECTION-CALL 04
        S05.run(); // SECTION-CALL 05
        S06.run(); // SECTION-CALL 06
        S07.run(); // SECTION-CALL 07
        S08.run(); // SECTION-CALL 08
        S09.run(); // SECTION-CALL 09
        S10.run(); // SECTION-CALL 10
        S11.run(); // SECTION-CALL 11
        S12.run(); // SECTION-CALL 12
        S13.run(); // SECTION-CALL 13
        S14.run(); // SECTION-CALL 14
        S15.run(); // SECTION-CALL 15
        S16.run(); // SECTION-CALL 16
        S17.run(); // SECTION-CALL 17
        S18.run(); // SECTION-CALL 18
        S19.run(); // SECTION-CALL 19
        S20.run(); // SECTION-CALL 20
        S21.run(); // SECTION-CALL 21
        S22.run(); // SECTION-CALL 22
        System.out.println("full: " + Main.checks + " checks, " + Main.failures + " failures");
        System.exit(Main.failures);
    }
}

// ===== SECTION 01: baseline =====
// Condensed re-assertion of the feature-matrix basics this file builds on (the
// feature file's exception style is not real Java, so throw lives in SECTION 19).
class S01 {
    static int add(int x, int y) { return x + y; }
    static void run() {
        int n = 0;
        for (int i = 0; i < 3; i++) { n = n + i; }
        Main.check("bas1", n == 3);
        int[] arr = new int[]{1, 2, 3};
        arr[1] = arr[1] + 10;
        Main.check("bas2", arr.length == 3 && arr[1] == 12);
        Main.check("bas3", ("a" + 1).equals("a1") && "ab".length() == 2);
        Main.check("bas4", S01.add(2, 3) == 5 && (3 > 2 ? "y" : "n").equals("y"));
        int w = 0;
        while (w < 3) { w = w + 1; }
        Main.check("bas5", w == 3);
    }
}

// ===== SECTION 02: numeric literal forms =====
class S02 {
    static void run() {
        Main.check("num1", 0xFF == 255 && 0xcafe == 51966);
        Main.check("num2", 0b1010 == 10 && 0B11 == 3);
        Main.check("num3", 017 == 15);
        Main.check("num4", 1_000_000 == 1000000 && 0xFF_FF == 65535);
        long big = 4_000_000_000L;
        Main.check("num5", big / 2L == 2000000000L && 7l == 7L);
        Main.check("num6", 1e3 == 1000.0 && 2.5e-2 == 0.025);
        Main.check("num7", 1.5f + 1.5F == 3.0f && 2.5d * 2 == 5.0D);
        Main.check("num8", .5 == 0.5 && 5. == 5.0);
        Main.check("num9", 0x1p4 == 16.0); // hexadecimal floating-point
        Main.check("num10", 2147483647 + 1 == -2147483648); // int wraps
    }
}

// ===== SECTION 03: char literals and escapes =====
class S03 {
    static void run() {
        char a = 'A';
        Main.check("chr1", a == 65 && 'B' == 'B');
        Main.check("chr2", '\n' == 10 && '\t' == 9 && '\'' == 39 && '\\' == 92);
        Main.check("chr3", '\101' == 'A' && '\0' == 0); // octal escapes
        char c = 'a';
        c++;
        Main.check("chr4", c == 'b' && (char) (c + 1) == 'c');
        Main.check("chr5", 'b' - 'a' == 1 && 'a' + 1 == 98); // promotes to int
        int hit = 0;
        switch (c) { case 'b': hit = 1; break; default: hit = 2; }
        Main.check("chr6", hit == 1 && "abc".charAt(1) == 'b');
        Main.check("chr7", "A\102".equals("AB") && "a\tb".length() == 3);
    }
}

// ===== SECTION 04: text blocks =====
class S04 {
    static void run() {
        String tb = """
                alpha
                beta""";
        Main.check("txt1", tb.equals("alpha\nbeta"));
        String tb2 = """
                a "quoted" line
                """;
        Main.check("txt2", tb2.equals("a \"quoted\" line\n"));
        String tb3 = """
                one \
                line""";
        Main.check("txt3", tb3.equals("one line")); // \<newline> joins
        String tb4 = """
                x\sy
                  indented""";
        Main.check("txt4", tb4.equals("x y\n  indented")); // \s keeps the space
    }
}

// ===== SECTION 05: var and the enhanced for =====
class S05 {
    static void run() {
        var i = 42;
        var s = "hi";
        var arr = new int[]{1, 2, 3};
        Main.check("var1", i == 42 && s.length() == 2 && arr.length == 3);
        var sum = 0;
        for (var v : arr) { sum += v; }
        Main.check("var2", sum == 6);
        for (var k = 0; k < 2; k++) { sum += 10; }
        Main.check("var3", sum == 26);
        String acc = "";
        String[] words = new String[]{"a", "b"};
        for (String w : words) { acc += w; }
        Main.check("var4", acc.equals("ab"));
        var flag = arr[0] < arr[1];
        Main.check("var5", flag);
    }
}

// ===== SECTION 06: operators and bit manipulation =====
class S06 {
    static void run() {
        Main.check("opr1", (5 & 3) == 1 && (5 | 3) == 7 && (5 ^ 3) == 6 && ~5 == -6);
        Main.check("opr2", (1 << 4) == 16 && (-16 >> 2) == -4 && (-8 >>> 28) == 15);
        int m = 12;
        m &= 10;
        m |= 1;
        m ^= 3;
        m <<= 2;
        m >>= 1;
        Main.check("opr3", m == 20);
        int u = -1;
        u >>>= 28;
        Main.check("opr4", u == 15);
        String grade = 87 >= 90 ? "A" : 87 >= 80 ? "B" : "C"; // ternary chain
        Main.check("opr5", grade.equals("B"));
        Main.check("opr6", (true & true) && (true ^ true) == false && (false | true));
        Main.check("opr7", (7 & 3 | 4 ^ 1) == (3 | 5)); // & then ^ then |
    }
}

// ===== SECTION 07: labeled statements =====
class S07 {
    static void run() {
        int hits = 0;
        outer:
        for (int i = 0; i < 3; i++) {
            for (int j = 0; j < 3; j++) {
                if (j == 1) { continue outer; }
                if (i == 2) { break outer; }
                hits++;
            }
        }
        Main.check("lab1", hits == 2);
        int reached = 0;
        block: { reached = 1; if (reached == 1) { break block; } reached = 2; }
        Main.check("lab2", reached == 1);
        int w = 0;
        wloop: while (true) { w = 5; break wloop; }
        Main.check("lab3", w == 5);
    }
}

// ===== SECTION 08: switch statements =====
class S08 {
    static int arrow(int x) {
        int r = 0;
        switch (x) {
            case 1, 2 -> r = 10;
            case 3 -> { r = 20; r += 1; }
            default -> r = -1;
        }
        return r;
    }
    static int classic(int x) { // fallthrough re-asserted, condensed
        int r = 0;
        switch (x) {
            case 0: r = 1;
            case 1: r += 2; break;
            default: r = 9;
        }
        return r;
    }
    static void run() {
        Main.check("sws1", S08.arrow(1) == 10 && S08.arrow(2) == 10);
        Main.check("sws2", S08.arrow(3) == 21 && S08.arrow(9) == -1);
        Main.check("sws3", S08.classic(0) == 3 && S08.classic(1) == 2 && S08.classic(7) == 9);
        String kind;
        switch ("sat") {
            case "sat", "sun" -> kind = "weekend";
            default -> kind = "workday";
        }
        Main.check("sws4", kind.equals("weekend"));
    }
}

// ===== SECTION 09: switch expressions =====
class S09 {
    static void run() {
        int n = 2;
        int a = switch (n) { case 1 -> 10; case 2, 3 -> 20; default -> 0; };
        Main.check("swe1", a == 20);
        int b = switch (n) {
            case 2 -> { int t = n * 3; yield t; }
            default -> -1;
        };
        Main.check("swe2", b == 6);
        int c = switch (n) { // the colon form yields too
            case 1: yield 100;
            default: yield n + 40;
        };
        Main.check("swe3", c == 42);
        String word = switch (n) { case 2 -> "two"; default -> "many"; };
        Main.check("swe4", word.equals("two"));
        Main.check("swe5", 5 + switch (n) { case 2 -> 1; default -> 0; } == 6);
    }
}

// ===== SECTION 10: instanceof pattern matching =====
class S10 {
    static String kindOf(Object o) {
        if (o instanceof String s) { return "s" + s.length(); }
        if (o instanceof Integer i && i > 5) { return "big"; }
        if (!(o instanceof Boolean b)) { return "other"; } // flow scoping
        return b ? "T" : "F";
    }
    static void run() {
        Main.check("iof1", S10.kindOf("ab").equals("s2"));
        Main.check("iof2", S10.kindOf(9).equals("big"));
        Main.check("iof3", S10.kindOf(3).equals("other"));
        Main.check("iof4", S10.kindOf(true).equals("T") && S10.kindOf(false).equals("F"));
        Object o = "text";
        if (o instanceof String) { // the pre-pattern form and a cast
            String plain = (String) o;
            Main.check("iof5", plain.length() == 4);
        } else {
            Main.check("iof5", false);
        }
    }
}

// ===== SECTION 11: sealed types and pattern switch =====
sealed interface Shape permits Dot, Line {}
record Dot(int x) implements Shape {}
record Line(Dot a, Dot b) implements Shape {}
class S11 {
    static int weight(Shape s) {
        return switch (s) { // exhaustive over the sealed hierarchy, no default
            case Dot d -> d.x();
            case Line l -> l.a().x() + l.b().x();
        };
    }
    static String label(Object o) {
        return switch (o) {
            case null -> "nil";
            case Integer i when i > 10 -> "big"; // guarded pattern
            case Integer i -> "int" + i;
            case String s -> "str" + s.length();
            default -> "other";
        };
    }
    static int ends(Shape s) {
        return switch (s) { // record deconstruction patterns
            case Dot(int x) -> x;
            case Line(Dot(var x1), Dot d2) -> x1 * 100 + d2.x();
        };
    }
    static void run() {
        Main.check("sel1", S11.weight(new Dot(5)) == 5);
        Main.check("sel2", S11.weight(new Line(new Dot(2), new Dot(3))) == 5);
        Main.check("sel3", S11.label(null).equals("nil"));
        Main.check("sel4", S11.label(11).equals("big") && S11.label(4).equals("int4"));
        Main.check("sel5", S11.label("abc").equals("str3") && S11.label(1.5).equals("other"));
        Main.check("sel6", S11.ends(new Line(new Dot(7), new Dot(8))) == 708);
        Shape sh = new Dot(6);
        Main.check("sel7", sh instanceof Dot(int v) && v == 6);
    }
}

// ===== SECTION 12: records =====
record Range(int lo, int hi) {
    Range { // compact constructor normalizes
        if (lo > hi) { int t = lo; lo = hi; hi = t; }
    }
    Range(int single) { this(single, single); }
    int width() { return this.hi() - this.lo(); }
    static Range unit() { return new Range(0, 1); }
}
record Named(String name) {
    public String name() { return "Mr. " + this.name; } // custom accessor
}
record Pair2<T>(T first, T second) {}
class S12 {
    static void run() {
        Range r = new Range(9, 2);
        Main.check("rec1", r.lo() == 2 && r.hi() == 9);
        Main.check("rec2", r.equals(new Range(2, 9)) && !r.equals(new Range(2, 8)));
        Range one = new Range(4);
        Main.check("rec3", one.width() == 0);
        Range u = Range.unit();
        Main.check("rec4", u.hi() == 1);
        Named n = new Named("x");
        Main.check("rec5", n.name().equals("Mr. x"));
        Pair2<String> p = new Pair2<>("a", "b");
        Main.check("rec6", (p.first() + p.second()).equals("ab"));
    }
}

// ===== SECTION 13: enums =====
enum Size {
    S(1), M(2), L(3) {
        int rank() { return 30; } // constant-specific body
    };
    final int units;
    Size(int u) { this.units = u; }
    int rank() { return this.units; }
}
class S13 {
    static int price(Size z) {
        return switch (z) { case S -> 1; case M -> 5; case L -> 9; };
    }
    static void run() {
        Main.check("enu1", Size.S.units == 1 && Size.M.units == 2);
        Main.check("enu2", Size.M.rank() == 2 && Size.L.rank() == 30);
        Main.check("enu3", Size.values().length == 3);
        Main.check("enu4", Size.M.ordinal() == 1 && Size.L.name().equals("L"));
        Main.check("enu5", S13.price(Size.S) == 1 && S13.price(Size.L) == 9);
        Size z = Size.valueOf("M");
        Main.check("enu6", z == Size.M);
    }
}

// ===== SECTION 14: generics =====
class GBox<T> {
    private T item;
    GBox(T item) { this.item = item; }
    T get() { return this.item; }
    void set(T v) { this.item = v; }
}
class NumVal {
    final int n;
    NumVal(int n) { this.n = n; }
}
class IntVal extends NumVal {
    IntVal(int n) { super(n); }
}
class S14 {
    static <T> T firstNonNull(T a, T b) { return a != null ? a : b; }
    static <T extends NumVal> int rawOf(T t) { return t.n; }
    static int readAny(GBox<? extends NumVal> b) { return b.get().n; }
    static void putSeven(GBox<? super IntVal> b) { b.set(new IntVal(7)); }
    static void run() {
        GBox<String> gs = new GBox<>("hi"); // diamond
        Main.check("gen1", gs.get().length() == 2);
        GBox<Integer> gi = new GBox<>(41); // int boxes into the type argument
        gi.set(gi.get() + 1);
        Main.check("gen2", gi.get() == 42);
        Main.check("gen3", S14.<String>firstNonNull(null, "x").equals("x"));
        Main.check("gen4", S14.firstNonNull("a", "b").equals("a"));
        IntVal nine = new IntVal(9);
        Main.check("gen5", S14.rawOf(nine) == 9);
        GBox<IntVal> bi = new GBox<>(new IntVal(3));
        Main.check("gen6", S14.readAny(bi) == 3);
        GBox<NumVal> bn = new GBox<>(new NumVal(1));
        S14.putSeven(bn);
        Main.check("gen7", bn.get().n == 7);
    }
}

// ===== SECTION 15: lambdas and method references =====
interface IntFn { int apply(int x); }
interface IntBi { int apply(int a, int b); }
interface Fetch<T> { T get(); }
@FunctionalInterface
interface Maker { Word make(String s); }
interface LenOf { int of(Word w); }
class Word {
    final String v;
    Word(String v) { this.v = v; }
    int len() { return this.v.length(); }
    static int dub(int x) { return x * 2; }
}
class S15 {
    static void run() {
        IntFn inc = x -> x + 1;
        Main.check("lam1", inc.apply(41) == 42);
        IntFn sq = (int x) -> { return x * x; }; // typed param, block body
        Main.check("lam2", sq.apply(7) == 49);
        IntFn neg = (var x) -> -x; // var-typed param
        Main.check("lam3", neg.apply(5) == -5);
        IntBi mul = (a, b) -> a * b;
        Main.check("lam4", mul.apply(6, 7) == 42);
        int base = 30;
        IntFn addBase = x -> x + base; // captured local
        Main.check("lam5", addBase.apply(12) == 42);
        Fetch<String> fs = () -> "p";
        Main.check("lam6", fs.get().equals("p"));
        IntFn dubRef = Word::dub; // static method reference
        Main.check("lam7", dubRef.apply(21) == 42);
        Word w = new Word("abc");
        Fetch<Integer> lenRef = w::len; // bound instance reference
        Main.check("lam8", lenRef.get() == 3);
        LenOf unbound = Word::len; // unbound: the receiver is argument 0
        Main.check("lam9", unbound.of(new Word("abcd")) == 4);
        Maker mk = Word::new; // constructor reference
        Word made = mk.make("xy");
        Main.check("lam10", made.len() == 2);
    }
}

// ===== SECTION 16: nested and inner classes =====
class Outer {
    final int base;
    Outer(int b) { this.base = b; }
    class Inner { // inner class captures the enclosing instance
        final int plus;
        Inner(int p) { this.plus = p; }
        int total() { return Outer.this.base + this.plus; }
    }
    static class Nested { int nine() { return 9; } }
}
interface Talker { String talk(); }
class S16 {
    static void run() {
        Outer o = new Outer(40);
        Outer.Inner in = o.new Inner(2); // qualified new
        Main.check("nst1", in.total() == 42);
        Outer.Nested nested = new Outer.Nested();
        Main.check("nst2", nested.nine() == 9);
        class Local { int five() { return 5; } }
        Local loc = new Local();
        Main.check("nst3", loc.five() == 5);
        Talker t = new Talker() {
            public String talk() { return "anon"; }
        };
        Main.check("nst4", t.talk().equals("anon"));
        Outer ext = new Outer(1) { }; // anonymous subclass of a class
        Main.check("nst5", ext.base == 1);
    }
}

// ===== SECTION 17: interface methods =====
interface Greet {
    String NAME = "G"; // implicitly public static final
    String id();
    default String hello() { return this.prefix() + this.id(); }
    private String prefix() { return "h:"; }
    static String kind() { return Greet.help(); }
    private static String help() { return "iface"; }
}
interface Greet2 { default String hello() { return "g2"; } }
class Both implements Greet, Greet2 {
    public String id() { return "B"; }
    public String hello() { return Greet.super.hello() + "/" + Greet2.super.hello(); }
}
class S17 {
    static void run() {
        Both b = new Both();
        Main.check("ifc1", b.hello().equals("h:B/g2"));
        Main.check("ifc2", Greet.kind().equals("iface"));
        Main.check("ifc3", Greet.NAME.equals("G"));
        Greet g = () -> "L"; // a functional interface despite the defaults
        Main.check("ifc4", g.hello().equals("h:L"));
    }
}

// ===== SECTION 18: varargs and arrays =====
class S18 {
    static int sumV(int... xs) {
        int s = 0;
        for (int x : xs) { s += x; }
        return s;
    }
    static String joinV(String sep, String... parts) {
        String out = "";
        for (String p : parts) { out = out + sep + p; }
        return out;
    }
    static void run() {
        Main.check("arr1", S18.sumV() == 0 && S18.sumV(5) == 5 && S18.sumV(1, 2, 3) == 6);
        int[] given = new int[]{4, 5};
        Main.check("arr2", S18.sumV(given) == 9); // an array feeds varargs
        Main.check("arr3", S18.joinV("-", "a", "b").equals("-a-b"));
        int[][] grid = new int[2][3];
        grid[1][2] = 7;
        Main.check("arr4", grid[0][0] == 0 && grid[1][2] == 7 && grid[0].length == 3);
        int[][] jag = {{1}, {2, 3}}; // initializer shorthand, jagged
        int total = 0;
        for (int[] row : jag) { for (int cell : row) { total += cell; } }
        Main.check("arr5", jag[0].length == 1 && total == 6);
        int cstyle[] = {5, 6}; // C-style declarator
        Main.check("arr6", cstyle[1] == 6);
        int[] mixed[] = new int[1][1]; // mixed-notation declarator
        mixed[0][0] = 8;
        Main.check("arr7", mixed[0][0] == 8);
        int[][] partial = new int[2][]; // only the first dimension given
        partial[1] = new int[]{4};
        Main.check("arr8", partial[0] == null && partial[1][0] == 4);
    }
}

// ===== SECTION 19: exceptions =====
class AErr extends Exception {
    AErr(String m) { super(m); }
}
class BErr extends RuntimeException {
    BErr(String m) { super(m); }
}
class Res implements AutoCloseable {
    static String log = "";
    final String tag;
    Res(String t) { this.tag = t; }
    public void close() { Res.log += "c" + this.tag; }
}
class S19 {
    static String pick(int n) {
        try {
            if (n == 1) { throw new AErr("a"); }
            if (n == 2) { throw new BErr("b"); }
            return "ok";
        } catch (AErr | BErr e) { // multi-catch
            return "caught:" + e.getMessage();
        }
    }
    static int declThrows() throws AErr {
        throw new AErr("d");
    }
    static void run() {
        Main.check("exc1", S19.pick(0).equals("ok"));
        Main.check("exc2", S19.pick(1).equals("caught:a") && S19.pick(2).equals("caught:b"));
        Res.log = "";
        try (Res r1 = new Res("1"); Res r2 = new Res("2")) {
            Res.log += "w";
        }
        Main.check("exc3", Res.log.equals("wc2c1")); // closed in reverse order
        Res r3 = new Res("3");
        Res.log = "";
        try (r3) { Res.log += "u"; } // an effectively final resource
        Main.check("exc4", Res.log.equals("uc3"));
        String m = "";
        try { S19.declThrows(); } catch (AErr e) { m = e.getMessage(); } finally { m += "!"; }
        Main.check("exc5", m.equals("d!"));
    }
}

// ===== SECTION 20: inheritance and constructor chaining =====
abstract class Vehicle {
    final String kindName;
    Vehicle(String k) { this.kindName = k; }
    Vehicle() { this("generic"); } // this() chains to the other constructor
    abstract int wheels();
    String label() { return this.kindName + ":" + this.wheels(); }
    final int axles() { return this.wheels() / 2; }
    Vehicle self() { return this; }
}
class Bike extends Vehicle {
    Bike() { super("bike"); }
    int wheels() { return 2; }
    @Override
    Bike self() { return this; } // covariant return type
    @Override
    String label() { return "a " + super.label(); }
}
class S20 {
    static void run() {
        Bike b = new Bike();
        Main.check("inh1", b.label().equals("a bike:2"));
        Main.check("inh2", b.axles() == 1);
        Vehicle anon = new Vehicle() { // an abstract class, completed anonymously
            int wheels() { return 4; }
        };
        Main.check("inh3", anon.label().equals("generic:4"));
        Main.check("inh4", anon.kindName.equals("generic"));
        Bike b2 = b.self(); // no cast needed thanks to covariance
        Main.check("inh5", b2 == b);
        final int fixed = 6;
        Vehicle up = b;
        Main.check("inh6", fixed == 6 && up.wheels() == 2);
    }
}

// ===== SECTION 21: initializer blocks =====
class InitOrder {
    static String slog = "";
    static { InitOrder.slog += "S1."; }
    static { InitOrder.slog += "S2."; }
    String ilog = "f";
    { this.ilog += "-b1"; } // instance initializers run in textual order
    InitOrder() { this.ilog += "-c"; }
    { this.ilog += "-b2"; }
}
class S21 {
    static void run() {
        InitOrder io = new InitOrder();
        Main.check("ini1", InitOrder.slog.equals("S1.S2."));
        Main.check("ini2", io.ilog.equals("f-b1-b2-c"));
        InitOrder io2 = new InitOrder();
        Main.check("ini3", io2.ilog.equals(io.ilog));
    }
}

// ===== SECTION 22: annotations =====
@interface Mark {}
@interface Meta {
    int id();
    String tag() default "t";
    int[] nums() default {1};
}
@interface Level { int value(); }
@Mark
@Meta(id = 3, nums = {1, 2})
class Conf {
    @Level(9) // single-element value() shorthand
    static int size() { return 3; }
}
class S22 {
    static void run() {
        Main.check("ann1", Conf.size() == 3);
        @Mark int local = 5; // annotation on a local declaration
        Main.check("ann2", local == 5);
        @Meta(id = 1, nums = 4) int single = 1; // one value fills the array
        Main.check("ann3", single == 1);
    }
}

// ===== END SECTIONS =====
