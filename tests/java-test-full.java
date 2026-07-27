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
        S23.run(); // SECTION-CALL 23
        S24.run(); // SECTION-CALL 24
        S25.run(); // SECTION-CALL 25
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
        // char is an INTEGRAL type that PRINTS as its glyph, and the two halves used
        // to disagree about which of the two it is: the interpreter concatenated
        // 'a' as 97, and the compiler answered a one-character String from charAt,
        // so charAt(1) + 0 was "e0" instead of 101 (JLS 4.2.1, 15.18.1).
        Main.check("chr8", ("x" + c).equals("xb") && ("" + 'q').equals("q"));
        String hi = "hello";
        Main.check("chr9", hi.charAt(1) + 0 == 101 && hi.charAt(1) == 'e');
        Main.check("chr10", ("" + hi.charAt(1)).equals("e") && (int) hi.charAt(0) == 104);
        char[] cs = new char[]{'x', 'y'};
        Main.check("chr11", ("" + cs[0]).equals("x") && cs[1] + 1 == 122);
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

// ===== SECTION 23: floating point arithmetic =====
// double/float are a DIFFERENT type from int, and every value below was checked
// against `java` (JDK 24) before it was written down. The whole section exists
// because both grammars used to evaluate every arithmetic operator in 32 bit
// integers: 1.0 / 3.0 was 0, 2.5 * 1.5 was 3, println(1.0) printed "1" and
// Infinity / NaN / -0.0 did not exist at all.
class S23 {
    static String s(double d) { return "" + d; }
    static double half() { return 0.5; }
    static void run() {
        // Division is REAL division as soon as one side is a double, and integer
        // division (truncating towards zero) when both sides are integral.
        Main.check("flt1", 1.0 / 3.0 == 0.3333333333333333);
        Main.check("flt2", 7 / 2 == 3 && -7 / 2 == -3 && 1 / 3 == 0);
        Main.check("flt3", (double) 7 / 2 == 3.5 && 7 / 2.0 == 3.5);
        Main.check("flt4", 2.5 * 1.5 == 3.75 && 7.0 - 0.5 == 6.5 && 3.0 * 2.0 == 6.0);
        Main.check("flt5", 2.0 % 0.75 == 0.5 && 1.5 + 1 == 2.5);
        // The mixed-type rule (JLS 5.6.2): a double operand promotes the whole
        // operation, so an integral result still prints with its decimal point.
        Main.check("flt6", S23.s(1.0).equals("1.0") && S23.s(6.0).equals("6.0"));
        Main.check("flt7", S23.s(0.1 + 0.2).equals("0.30000000000000004"));
        Main.check("flt8", S23.s(1.0 / 3.0).equals("0.3333333333333333"));
        // Infinity, NaN and the two zeroes.
        double inf = 1.0 / 0.0;
        double nan = 0.0 / 0.0;
        Main.check("flt9", S23.s(inf).equals("Infinity") && S23.s(-1.0 / 0.0).equals("-Infinity"));
        Main.check("flt10", S23.s(nan).equals("NaN") && nan != nan);
        Main.check("flt11", S23.s(-0.0).equals("-0.0") && 0.0 == -0.0);
        Main.check("flt12", 1e300 * 1e300 == inf);
        // Double.toString switches to scientific notation outside [1e-3, 1e7).
        Main.check("flt13", S23.s(1e20).equals("1.0E20") && S23.s(1.5e-8).equals("1.5E-8"));
        Main.check("flt14", S23.s(1e-3).equals("0.001") && S23.s(2.5).equals("2.5"));
        // A double-declared variable converts its initializer; ++ and the compound
        // operators keep it a double.
        double d = 1;
        Main.check("flt15", S23.s(d).equals("1.0"));
        d += 0.25;
        Main.check("flt16", d == 1.25);
        d++;
        Main.check("flt17", d == 2.25 && S23.s(d).equals("2.25"));
        d *= 2;
        Main.check("flt18", d == 4.5);
        d /= 3;
        Main.check("flt19", d == 1.5);
        d--;
        Main.check("flt20", S23.s(d).equals("0.5"));
        // A cast in each direction, and unary minus on a double.
        Main.check("flt21", (int) 2.9 == 2 && (int) -2.9 == -2 && (double) 3 == 3.0);
        Main.check("flt22", S23.s(-2.5).equals("-2.5") && -S23.half() == -0.5);
        // Math.abs/max/min take the double overload as soon as one side is one.
        Main.check("flt23", Math.abs(-2.5) == 2.5 && S23.s(Math.abs(-2.0)).equals("2.0"));
        Main.check("flt24", S23.s(Math.max(1.5, 2)).equals("2.0") && Math.min(1.5, 2) == 1.5);
        // A double survives a field, an array element and a method boundary.
        double[] xs = new double[]{0.5, 1.5};
        Main.check("flt25", xs[0] + xs[1] == 2.0 && S23.s(xs[0] * 2).equals("1.0"));
        Main.check("flt26", S23.half() / 2 == 0.25);
        // Comparisons mix the two kinds freely.
        Main.check("flt27", 2.5 > 2 && 2 < 2.5 && 1.0 == 1 && 3.0 <= 3 && 3.0 >= 3);
        // A float literal and an f/d suffix are doubles here too.
        float f = 3;
        Main.check("flt28", f / 2 == 1.5 && 1.5f + 1.5F == 3.0f && 2.5d * 2 == 5.0D);
    }
}

// ===== SECTION 24: value rendering (toString / println) =====
// What a value RENDERS as: the string `+` builds and the text println writes.
// Checked against `java` (JDK 24) before it was written down. The identity hash
// after the '@' is deliberately NOT asserted - real java prints
// System.identityHashCode, which differs between two runs of the SAME program,
// so only the class-name prefix is reproducible. The section exists because the
// compiler half used to hand the raw runtime value to Go's formatter (<nil> for
// null, [1 2 3] for an array, a pointer-bearing map dump for an instance) while
// the interpreter half fell through to JavaScript's ToString ("1,2,3",
// "Box@obj"), and neither printed what java prints.
class S24 {
    static class Box { int x; }
    static class Shown { public String toString() { return "NAMED"; } }
    record Pt(int x, int y) {}
    enum Col { RED, BLUE }

    static String s(Object o) { return "" + o; }
    static void run() {
        // A null reference renders as "null", alone and inside a concatenation.
        String ns = null;
        Object no = null;
        Main.check("ren1", S24.s(no).equals("null") && ("s=" + ns).equals("s=null"));
        // An array renders as its class name, '@' and an identity hash. The
        // ELEMENT TYPE is part of that class name.
        int[] ia = {1, 2, 3};
        double[] da = {1.5};
        boolean[] za = {true};
        String[] sa = {"p", "q"};
        String ir = S24.s(ia);
        Main.check("ren2", ir.substring(0, 3).equals("[I@") && ir.length() > 3);
        Main.check("ren3", S24.s(da).substring(0, 3).equals("[D@"));
        Main.check("ren4", S24.s(za).substring(0, 3).equals("[Z@"));
        Main.check("ren5", S24.s(sa).substring(0, 20).equals("[Ljava.lang.String;@"));
        // The same array renders identically twice; two arrays do not collide.
        int[] ib = {1, 2, 3};
        Main.check("ren6", S24.s(ia).equals(ir) && !S24.s(ib).equals(ir));
        // An instance renders as its BINARY class name - a nested type is spelled
        // Outer$Inner, the way Class#getName does - then '@' and the hash.
        Box b = new Box();
        String br = S24.s(b);
        Main.check("ren7", br.substring(0, 8).equals("S24$Box@") && br.indexOf("@") == 7);
        Main.check("ren8", S24.s(b).equals(br) && !S24.s(new Box()).equals(br));
        // A user toString() wins over all of it, wherever the value is rendered.
        Shown n = new Shown();
        Main.check("ren9", S24.s(n).equals("NAMED") && ("<" + n + ">").equals("<NAMED>"));
        Main.check("ren10", n.toString().equals("NAMED"));
        // The two GENERATED ones: a record's canonical form (which uses the simple
        // name, not the binary one) and an enum constant's name.
        Main.check("ren11", S24.s(new Pt(1, 2)).equals("Pt[x=1, y=2]"));
        Main.check("ren12", S24.s(Col.RED).equals("RED") && Col.BLUE.toString().equals("BLUE"));
        // Everything that already rendered correctly still does.
        Main.check("ren13", S24.s(true).equals("true") && ("" + 'x').equals("x") && S24.s(1.5).equals("1.5"));
    }
}

// ===== SECTION 25: sized integers (int / long / byte / short / char) =====
// Java's integral types have FIXED WIDTHS and wrap silently, and the width has
// to survive into the next operation. Every answer below was run as a real
// .java file under `java` (JDK 24) BEFORE it was written down, and the whole
// ratchet file is itself a valid Java program - so `java tests/java-test-full.java`
// prints the same summary line both grammar halves print.
//
// The area used to be simply absent: Integer.MAX_VALUE was an undefined
// variable, `long` was a 32 bit int (1L << 40 answered 256), Long.MAX_VALUE
// could not even be spelled because a double rounds 9223372036854775807 to
// 9223372036854776000 - and the two script engines rounded it DIFFERENTLY, so
// the goja and -frozen runs of the same program disagreed.
class S25 {
    static int overflow(int x) { return x + 1; }
    static long lid(long x) { return x; }
    static void run() {
        // ----- the type constants -----
        Main.check("int1", Integer.MAX_VALUE == 2147483647 && Integer.MIN_VALUE == -2147483648);
        Main.check("int2", Long.MAX_VALUE == 9223372036854775807L && Long.MIN_VALUE == -9223372036854775808L);
        Main.check("int3", Byte.MAX_VALUE == 127 && Byte.MIN_VALUE == -128);
        Main.check("int4", Short.MAX_VALUE == 32767 && Short.MIN_VALUE == -32768);
        Main.check("int5", Character.MAX_VALUE == 65535 && Character.MIN_VALUE == 0);
        // A long's DIGITS survive: the value is exact, not the nearest double.
        Main.check("int6", ("" + Long.MAX_VALUE).equals("9223372036854775807"));
        Main.check("int7", ("" + Long.MIN_VALUE).equals("-9223372036854775808"));

        // ----- int wraps at 32 bits, long at 64 -----
        Main.check("int8", Integer.MAX_VALUE + 1 == Integer.MIN_VALUE);
        Main.check("int9", Integer.MIN_VALUE - 1 == Integer.MAX_VALUE);
        Main.check("int10", S25.overflow(2147483647) == -2147483648);
        Main.check("int11", Long.MAX_VALUE + 1 == Long.MIN_VALUE);
        // 3000000000 does not fit in an int, so the multiplication must be 64 bit.
        Main.check("int12", 3000000000L * 3 == 9000000000L);
        // ... and the same operands as ints wrap instead.
        int big = 1000000;
        Main.check("int13", big * big == -727379968);
        Main.check("int14", 1000000L * 1000000L == 1000000000000L);
        // A declared `long` keeps its width across an assignment and a call.
        long acc = 1;
        acc = acc * 1000000;
        acc = acc * 1000000;
        Main.check("int15", acc == 1000000000000L && S25.lid(acc) == 1000000000000L);

        // ----- the shift count is MASKED: & 31 for an int, & 63 for a long -----
        // This is the classic: `1 << 32` is 1, not 0. (Go shifts everything out.)
        Main.check("int16", 1 << 32 == 1 && 1 << 33 == 2);
        Main.check("int17", 1L << 64 == 1L && 1L << 65 == 2L);
        Main.check("int18", 1L << 40 == 1099511627776L && 1L << 62 == 4611686018427387904L);
        Main.check("int19", 1 << 31 == Integer.MIN_VALUE && 1L << 63 == Long.MIN_VALUE);
        // >> keeps the sign, >>> does not - and >>> is a 32 bit operation on an
        // int and a 64 bit one on a long, which is what a `| 0` could never model.
        Main.check("int20", -8 >> 1 == -4 && -8 >>> 1 == 2147483644);
        Main.check("int21", -8L >> 1 == -4L && -8L >>> 1 == 9223372036854775804L);
        Main.check("int22", -1 >>> 28 == 15 && -1L >>> 60 == 15L);

        // ----- division truncates toward zero; % takes the DIVIDEND's sign -----
        Main.check("int23", -7 / 2 == -3 && 7 / -2 == -3 && -7 % 2 == -1 && 7 % -2 == 1);
        Main.check("int24", Long.MAX_VALUE / 3 == 3074457345618258602L);
        // The one signed-division overflow: MIN_VALUE / -1 wraps back to itself.
        Main.check("int25", Integer.MIN_VALUE / -1 == Integer.MIN_VALUE);
        Main.check("int26", Long.MIN_VALUE / -1 == Long.MIN_VALUE && Integer.MIN_VALUE % -1 == 0);
        // Integer division by zero THROWS (JLS 15.17.2), it does not give Infinity.
        boolean threw = false;
        String msg = "";
        try { int z = 0; int q = 1 / z; Main.check("int27", q == 99); }
        catch (ArithmeticException e) { threw = true; msg = e.getMessage(); }
        Main.check("int27", threw && msg.equals("/ by zero"));
        boolean lthrew = false;
        try { long lz = 0; long lq = 1L % lz; Main.check("int28", lq == 99); }
        catch (ArithmeticException e) { lthrew = true; }
        Main.check("int28", lthrew);

        // ----- byte / short: their own widths, and PROMOTION to int -----
        byte b = (byte) 200;
        short sh = (short) 70000;
        Main.check("int29", b == -56 && sh == 4464);
        // byte + byte is an INT (JLS 5.6.2), so it does not wrap at 8 bits.
        byte b1 = 100;
        byte b2 = 100;
        Main.check("int30", b1 + b2 == 200);
        // ++ and a compound assignment DO narrow back (JLS 15.14.2 / 15.26.2).
        byte bm = 127;
        bm++;
        Main.check("int31", bm == -128);
        byte bc = 10;
        bc += 300;
        Main.check("int32", bc == 54);
        short sm = 32767;
        sm++;
        Main.check("int33", sm == -32768);
        // A byte compares against an int at INT width - 5 < 1000 is not 5 < -24.
        byte bs = 5;
        Main.check("int34", bs < 1000 && bs > -1000);

        // ----- casts narrow to the target width -----
        Main.check("int35", (int) 3000000000L == -1294967296 && (long) (int) 3000000000L == -1294967296L);
        Main.check("int36", (byte) 300 == 44 && (short) -70000 == -4464);
        Main.check("int37", (char) 70000 == 4464 && (char) -1 == 65535);
        // A double narrows to an integral type by TRUNCATING toward zero, and an
        // out-of-range value SATURATES (JLS 5.1.3) - it does not wrap.
        Main.check("int38", (int) 2.9 == 2 && (int) -2.9 == -2 && (long) -2.9 == -2L);
        Main.check("int39", (long) 1e20 == Long.MAX_VALUE && (int) 1e20 == Integer.MAX_VALUE);

        // ----- literals -----
        // Hex/binary/octal without a suffix are INTs, so the digits are read at 32
        // bits and 0xFFFFFFFF is -1; with L they are read at 64.
        Main.check("int40", 0xFFFFFFFF == -1 && 0xFFFFFFFFL == 4294967295L);
        Main.check("int41", 0xFFFFFFFFFFFFFFFFL == -1L && 0x7FFFFFFFFFFFFFFFL == Long.MAX_VALUE);
        Main.check("int42", 0b1010 == 10 && 017 == 15 && 1_000_000 == 1000000);
        Main.check("int43", 1_000_000_000_000L == 1000000000000L);

        // ----- unary -, ~ and char arithmetic -----
        Main.check("int44", -Integer.MIN_VALUE == Integer.MIN_VALUE && -Long.MIN_VALUE == Long.MIN_VALUE);
        Main.check("int45", ~0 == -1 && ~0L == -1L && ~(1L << 40) == -1099511627777L);
        char c = 'a';
        Main.check("int46", c + 1 == 98 && (char) (c + 1) == 'b');

        // ----- the bitwise operators are 64 bit on a long -----
        long mask = 0xF0F0F0F0F0F0F0F0L;
        Main.check("int47", (mask & 0xFFL) == 0xF0L && (mask | 1L) == 0xF0F0F0F0F0F0F0F1L);
        Main.check("int48", (mask ^ mask) == 0L && (mask >>> 4) == 0x0F0F0F0F0F0F0F0FL);

        // ----- it composes with the FLOAT box rather than fighting it -----
        // Mixed int/double arithmetic still promotes to double (JLS 5.6.2), and a
        // long promotes the same way.
        Main.check("int49", 1 / 3 == 0 && 1.0 / 3.0 == 0.3333333333333333 && 1 / 3.0 == 0.3333333333333333);
        Main.check("int50", Long.MAX_VALUE + 0.0 == 9.223372036854776E18 && (double) 1L / 2 == 0.5);
        Main.check("int51", ("" + (2.5 * 2)).equals("5.0") && ("" + (5 * 2)).equals("10"));
        double d = 1;
        d += 1L;
        Main.check("int52", d == 2.0 && ("" + d).equals("2.0"));
        // A compound assignment to an INT narrows a double result back to an int.
        int ni = 5;
        ni += 1.5;
        Main.check("int53", ni == 6);

        // ----- Integer.parseInt / Long.parseLong / compare -----
        Main.check("int54", Integer.parseInt("-42") == -42 && Long.parseLong("9223372036854775807") == Long.MAX_VALUE);
        Main.check("int55", Integer.compare(1, 2) == -1 && Long.compare(Long.MAX_VALUE, 0L) == 1);
        Main.check("int56", Integer.max(3, 7) == 7 && Long.min(Long.MIN_VALUE, 0L) == Long.MIN_VALUE);

        // ----- a long survives an array, a field and String.valueOf -----
        long[] ls = new long[2];
        ls[0] = Long.MAX_VALUE;
        ls[1] = ls[0] - 1;
        Main.check("int57", ls[1] == 9223372036854775806L && ("" + ls[0]).equals("9223372036854775807"));
        Main.check("int58", ("" + (Long.MAX_VALUE - 1)).equals("9223372036854775806"));
    }
}

// ===== END SECTIONS =====
