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
// importing java.util.function. Also out: modules - a module-info.java IS parsed
// (see ModuleDecl in the grammar) but is a file of its own, so it cannot be a
// section here - threads (and synchronized), reflection, javadoc.
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
        S26.run(); // SECTION-CALL 26
        S27.run(); // SECTION-CALL 27
        S28.run(); // SECTION-CALL 28
        S29.run(); // SECTION-CALL 29
        S30.run(); // SECTION-CALL 30
        S31.run(); // SECTION-CALL 31
        S32.run(); // SECTION-CALL 32
        S33.run(); // SECTION-CALL 33
        S34.run(); // SECTION-CALL 34
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
record Glyph(char c, int n) {}
// The component types the generated equals used to compare with ===, and the
// three shapes that answers wrongly: a floating-point component (Double.compare,
// not ==), a boxed integral one (a long is a box at every magnitude, so identity
// answered false for two equal values in the interpreter half) and a REFERENCE
// one (Objects.equals dispatches the component's own equals).
record RNum(double d, float f) {}
record RWide(long l, byte b, short s) {}
record Inner12(int a) {}
record Outer12(Inner12 in) {}
record REq(Eqv e) {}
record RPlain(Plain12 p) {}
record RArr(int[] a) {}
class Eqv {                       // a user-declared equals, which Objects.equals uses
    final int v;
    Eqv(int v) { this.v = v; }
    public boolean equals(Object o) {
        if (!(o instanceof Eqv)) { return false; }
        return ((Eqv) o).v == this.v;
    }
    public int hashCode() { return this.v; }
}
class Plain12 {                   // no equals, so Object.equals - i.e. identity - stands
    int v;
    Plain12(int v) { this.v = v; }
}
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
        // A CHAR record component. The generated equals compared components with
        // js_seq, and a char is the one shape === answers differently in the two
        // engines: the Go twin's char is a comparable struct, layer 2's is a
        // {__char} box that === settles by identity. So this held in llvm.Run and
        // in the interpreter half and FAILED in the native binary. Every operand is
        // read out of an array so the grammar's constant folder cannot answer it.
        char[] gs = {'q', 'q', 'r'};
        int[] gn = {5, 5, 6};
        Main.check("rec7", new Glyph(gs[0], gn[0]).equals(new Glyph(gs[1], gn[1])));
        Main.check("rec8", !new Glyph(gs[0], gn[0]).equals(new Glyph(gs[2], gn[1]))
                        && !new Glyph(gs[0], gn[0]).equals(new Glyph(gs[1], gn[2])));
        // The rest of JLS 8.10.3, all of it measured against java 24.0.2 and all
        // of it wrong before js_jvaleq. Every operand is read out of an array.
        //
        // A double / float component is Double.compare(a,b)==0, which === is NOT
        // in either direction: NaN equals NaN, and +0.0 does not equal -0.0.
        double[] dz = {0.0, -0.0, 1.5, 1.5};
        float[] fz = {0.0f, -0.0f, 2.5f, 2.5f};
        double[] nz = {dz[0] / dz[0], dz[1] / dz[1]};
        float[] fn = {fz[0] / fz[0], fz[1] / fz[1]};
        Main.check("rec9", new RNum(nz[0], fn[0]).equals(new RNum(nz[1], fn[1])));
        Main.check("rec10", !new RNum(dz[0], fz[2]).equals(new RNum(dz[1], fz[3]))
                        && !new RNum(dz[2], fz[0]).equals(new RNum(dz[3], fz[1]))
                        && new RNum(dz[2], fz[2]).equals(new RNum(dz[3], fz[3])));
        // long / byte / short are BOXES here, so identity said two equal values
        // were unequal - in the interpreter half only, which is why --cross saw it
        // and the two compiler engines did not.
        long[] lz = {5000000000L, 5000000000L, 5000000001L};
        byte[] bz = {7, 7, 8};
        short[] sz = {300, 300, 301};
        Main.check("rec11", new RWide(lz[0], bz[0], sz[0]).equals(new RWide(lz[1], bz[1], sz[1])));
        Main.check("rec12", !new RWide(lz[2], bz[0], sz[0]).equals(new RWide(lz[0], bz[0], sz[0]))
                        && !new RWide(lz[0], bz[2], sz[0]).equals(new RWide(lz[0], bz[0], sz[0]))
                        && !new RWide(lz[0], bz[0], sz[2]).equals(new RWide(lz[0], bz[0], sz[0])));
        // A reference component is Objects.equals: a nested record compares
        // component-wise and a user-declared equals is CALLED, where identity had
        // both answering false in the two compiler engines.
        int[] iz = {3, 3, 4};
        Inner12[] inz = {new Inner12(iz[0]), new Inner12(iz[1]), new Inner12(iz[2])};
        Main.check("rec13", new Outer12(inz[0]).equals(new Outer12(inz[1]))
                        && !new Outer12(inz[0]).equals(new Outer12(inz[2])));
        Eqv[] ez = {new Eqv(iz[0]), new Eqv(iz[1]), new Eqv(iz[2])};
        Main.check("rec14", new REq(ez[0]).equals(new REq(ez[1]))
                        && !new REq(ez[0]).equals(new REq(ez[2])));
        // ... and a class that declares no equals, and an array (which does not
        // override it either), KEEP identity: Object.equals is identity.
        Plain12[] pz = {new Plain12(iz[0]), new Plain12(iz[1])};
        Main.check("rec15", new RPlain(pz[0]).equals(new RPlain(pz[0]))
                        && !new RPlain(pz[0]).equals(new RPlain(pz[1])));
        int[][] az = new int[2][];
        az[0] = new int[] {iz[0]};
        az[1] = new int[] {iz[0]};
        Main.check("rec16", new RArr(az[0]).equals(new RArr(az[0]))
                        && !new RArr(az[0]).equals(new RArr(az[1])));
        // equals takes an Object, so equals(null) is a legal call that answers
        // false. Reading __class off it ABORTED the whole program in both
        // compiler engines.
        Main.check("rec17", !new Outer12(inz[0]).equals(null)
                        && !new Outer12(inz[0]).equals("x")
                        && new Outer12(null).equals(new Outer12(null)));
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

        // ----- the receiver of a BOUND reference may be any expression -----
        // JLS 15.13: `expr::m` evaluates expr once, where the reference is
        // written, and the closure then forwards only its own arguments.
        Fetch<Integer> viaCall = S15.pick("abcde")::len;   // a call result
        Main.check("lam11", viaCall.get() == 5);
        Fetch<Integer> viaNew = new Word("hi")::len;       // a `new` expression
        Main.check("lam12", viaNew.get() == 2);
        Fetch<Integer> viaCast = ((Word) S15.pick("wxyz"))::len; // a parenthesized cast
        Main.check("lam13", viaCast.get() == 4);
        S15.evals = 0;
        Fetch<Integer> once = S15.counted()::len;
        once.get();
        once.get();
        Main.check("lam14", S15.evals == 1); // receiver evaluated ONCE, not per call

        // ----- type arguments on either side of :: are parsed and erased -----
        LenOf targs = Word::<Integer>len;
        Main.check("lam15", targs.of(new Word("abc")) == 3);

        // ----- an ARRAY constructor reference: String[]::new -----
        ArrMaker am = String[]::new;
        String[] fresh = am.make(3);
        Main.check("lam16", fresh.length == 3 && fresh[0] == null);
    }
    static int evals = 0;
    static Word pick(String s) { return new Word(s); }
    static Word counted() { S15.evals++; return new Word("z"); }
}
interface ArrMaker { String[] make(int n); }

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
    static String f(float v) { return "" + v; }
    static float third() { return 1.0f / 3.0f; }
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
        // JLS 15.20.1: if either operand is NaN then <, <=, > and >= are ALL
        // false. Both halves got this wrong, in DIFFERENT directions, and until
        // these four lines nothing in the suite asked. The Go twin took the
        // sentinel 2 that jsCompare answers for a NaN and read it as an ordering,
        // so `>` and `>=` were true; layer 2's own copy of jsCompare dropped the
        // sentinel and answered 0, so `<=` and `>=` were true. The same defect
        // was found and fixed for C# in Part A of docs/runtime-merge-plan.md;
        // java is its twin and was not looked at.
        Main.check("flt10a", !(nan < 1.0) && !(nan > 1.0) && !(nan <= 1.0) && !(nan >= 1.0));
        Main.check("flt10b", !(1.0 < nan) && !(1.0 > nan) && !(1.0 <= nan) && !(1.0 >= nan));
        Main.check("flt10c", !(nan <= nan) && !(nan >= nan) && !(nan < nan) && !(nan > nan));
        float fnan = 0.0f / 0.0f;
        Main.check("flt10d", !(fnan >= 1.0f) && !(fnan <= 1.0f) && !(fnan > 1.0f) && !(fnan < 1.0f));
        // JLS 15.21.1: == between two numeric operands applies binary numeric
        // promotion (5.6.2) first, so a long is compared against a double AS a
        // double. Both COMPILER halves answered false - floEq (the floor's
        // jf_num_eq, the twin's jvmNumEq) had no sized-integer arm and a comment
        // at three sites called that deliberate - while both INTERPRETER halves
        // answered true. Nothing in the suite had ever compared a long to a
        // double, so --cross could not see it either. Found by the shape merge of
        // java's and csharp's strict equality into lib/runtime-jvm.metajs, and
        // confirmed against real java 24.0.2. The operands come out of arrays so
        // the constant folder cannot answer at compile time.
        long[] lz = {0L, 3L, 9007199254740993L};
        double[] dz = {0.0, 3.0, 9007199254740992.0};
        Main.check("flt10e", lz[0] == dz[0] && lz[1] == dz[1] && !(lz[0] != dz[0]));
        Main.check("flt10f", dz[0] == lz[0] && lz[0] != dz[1] && lz[1] != dz[0]);
        // The long converts FIRST, so a value past 2^53 compares as its rounded
        // double: 9007199254740993L == 9007199254740992.0 is true.
        Main.check("flt10g", lz[2] == dz[2] && dz[2] == lz[2]);
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
        // A d/D suffix is a double; an f/F suffix is a FLOAT (see below).
        float f = 3;
        Main.check("flt28", f / 2 == 1.5 && 1.5f + 1.5F == 3.0f && 2.5d * 2 == 5.0D);
        // ----- the float WIDTH (todo.md 1.2) -----
        // `float` is a binary32, not a double spelled differently. Every value
        // below was read off `java` 24.0.2 before it was written down; the whole
        // block used to answer the DOUBLE result, so each line discriminates.
        Main.check("flt29", 1.0f / 3.0f == 0.33333334f && 1.0f / 3.0f != 1.0 / 3.0);
        Main.check("flt30", S23.f(1.0f / 3.0f).equals("0.33333334"));
        Main.check("flt31", S23.f(0.1f + 0.2f).equals("0.3") && S23.s(0.1f + 0.2).equals("0.30000000149011613"));
        // The narrowing is REAL: (double)0.1f is not 0.1, and 16777217 has no
        // float, so it converts to 1.6777216E7 - including on the way INTO a
        // comparison (JLS 5.6.2 promotes the int operand to float first).
        Main.check("flt32", S23.s((double) 0.1f).equals("0.10000000149011612"));
        Main.check("flt33", S23.f(16777217f).equals("1.6777216E7") && 16777216f == 16777217);
        Main.check("flt34", S23.f((float) (1.0 / 3.0)).equals("0.33333334"));
        // The print window and the forced point are Double.toString's, read at
        // 24 bits: plain for 1e-3 <= |v| < 1e7 and d.dddEnn outside it.
        Main.check("flt35", S23.f(100f).equals("100.0") && S23.f(0.001f).equals("0.001"));
        Main.check("flt36", S23.f(9999999f).equals("9999999.0") && S23.f(1e7f).equals("1.0E7"));
        Main.check("flt37", S23.f(1e20f).equals("1.0E20") && S23.f(1e-20f).equals("1.0E-20"));
        // The two-significant-digit minimum, against the ACTUAL value: the
        // smallest float is 1.401298464324817E-45 and java writes "1.4E-45".
        Main.check("flt38", S23.f(1e-45f).equals("1.4E-45") && S23.f(3.4028235e38f).equals("3.4028235E38"));
        // A float overflows and underflows at the FLOAT boundaries.
        Main.check("flt39", S23.f(1e38f * 10f).equals("Infinity") && S23.f(1e-38f / 1e10f).equals("0.0"));
        Main.check("flt40", S23.f(1f / 0f).equals("Infinity") && S23.f(0f / 0f).equals("NaN") && S23.f(-0.0f).equals("-0.0"));
        // A compound assignment casts back to the LEFT operand's type, so a
        // double on the right does not widen the variable (JLS 15.26.2).
        float g = 1.1f; g += 0.1;
        float h = 0.1f; h += 16777217;
        Main.check("flt41", S23.f(g).equals("1.2") && S23.f(h).equals("1.6777216E7"));
        // Unary minus, Math, an array element and an array's own rendering.
        float[] fs = new float[]{0.5f, 1.5f};
        Main.check("flt42", S23.f(-S23.third()).equals("-0.33333334") && S23.f(fs[0] * 3).equals("1.5"));
        Main.check("flt43", S23.f(Math.abs(-2.5f)).equals("2.5") && S23.f(Math.max(1.5f, 2)).equals("2.0")
                && S23.s(Math.max(1.5f, 2.0)).equals("2.0"));
        Main.check("flt44", ("" + fs).indexOf("[F@") == 0 && (int) 2.9f == 2 && (long) -2.9f == -2L);
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

        // ----- Math.abs is OVERLOADED, and abs(long) answers a LONG -----
        // Every engine took the int overload's 32 bit wrap for a long argument, so
        // Math.abs(3000000000L) was -1294967296 and Math.abs(Long.MAX_VALUE) was 0
        // (the interpreter answered 0 for EVERY long, its `v < 0` reading a box).
        // All three agreed with each other, which is why the ratchet had no
        // assertion here and neither --cross nor the byte-identity matrix could
        // see it. 20 of the 26 lines on which our java differed from java 24 over
        // the whole long surface were this one cause.
        Main.check("int59", Math.abs(3000000000L) == 3000000000L && Math.abs(-3000000000L) == 3000000000L);
        Main.check("int60", Math.abs(Long.MAX_VALUE) == Long.MAX_VALUE && Math.abs(-1L) == 1L);
        // JLS 15.15.4: there is no positive long of MIN_VALUE's magnitude, so
        // abs(MIN_VALUE) is MIN_VALUE itself - a value, not an error.
        Main.check("int61", Math.abs(Long.MIN_VALUE) == Long.MIN_VALUE);
        Main.check("int62", Math.abs(Long.MIN_VALUE + 1) == Long.MAX_VALUE);
        // The guard rails: the int overload still wraps at 32 bits, and the double
        // overload is untouched.
        Main.check("int63", Math.abs(Integer.MIN_VALUE) == Integer.MIN_VALUE && Math.abs(-5) == 5);
        Main.check("int64", ("" + Math.abs(-1234567890123L)).equals("1234567890123"));
    }
}


// ===== SECTION 26: type annotations, receiver parameters and assignable places =====
// A TYPE_USE annotation may sit in front of any type, including the one in a
// `new`, a cast, an `extends`/`implements` clause and every array bracket pair
// (JLS 9.7.4); a method may declare the type of `this` as a RECEIVER parameter
// (JLS 8.4.1); a cast may name an INTERSECTION of types (JLS 15.16); and the
// left-hand side of an assignment is a field or array access over ANY primary,
// optionally wrapped in parentheses (JLS 15.26). The annotation type is
// declared with a fully qualified @Target so the section needs no import.
@java.lang.annotation.Target({java.lang.annotation.ElementType.TYPE_USE,
                              java.lang.annotation.ElementType.METHOD,
                              java.lang.annotation.ElementType.FIELD,
                              java.lang.annotation.ElementType.LOCAL_VARIABLE})
@interface TU {}

interface Marker26 {}
class Base26 { int tag() { return 1; } }
class Sub26 extends @TU Base26 implements @TU Marker26 {
    int tag() { return 2; }
}
class Box26 {
    int v = 0;
    int get(@TU Box26 this) { return this.v; }
}
class Outer26 {
    class Inner26 { int val() { return 10; } }
}
record Pair26(String head, String... rest) {}

class S26 {
    static Box26 mk() { return new Box26(); }
    static int sum(@TU int @TU ... xs) {
        int t = 0;
        for (int x : xs) { t += x; }
        return t;
    }

    static void run() {
        // ----- a type annotation on a `new`, a cast, an array and a supertype -----
        Object o1 = new @TU Base26();
        Main.check("ta1", o1 != null && new Sub26().tag() == 2);
        @TU String s1 = (@TU String) (Object) "x";
        Main.check("ta2", s1.equals("x"));
        @TU int @TU [] arr = new @TU int @TU [3];
        arr[1] = 7;
        Main.check("ta3", arr.length == 3 && arr[1] == 7);

        // ----- a receiver parameter, and an annotated varargs parameter -----
        Main.check("ta4", new Box26().get() == 0 && sum(1, 2, 3) == 6);

        // ----- an intersection cast, and a cast of a cast -----
        Runnable r = (Runnable & Marker26) () -> { };
        Object o2 = (String) (Object) "y";
        Main.check("ta5", r != null && "y".equals(o2));

        // ----- a parenthesized assignment target -----
        int a = 0;
        int b = 0;
        (a) = (b) = 1;
        Main.check("ta6", a == 1 && b == 1);
        int i = 5;
        int j = (i)++;
        (i) += 2;
        Main.check("ta7", j == 5 && i == 8);

        // ----- an assignable place rooted at any primary -----
        Box26 box = new Box26();
        Object any = box;
        ((Box26) any).v = 11;
        Main.check("ta8", box.v == 11);
        mk().v = 9;
        new Box26().v = 12;
        Main.check("ta9", mk().get() == 0);

        // ----- a qualified `new` that also declares an anonymous subclass -----
        Outer26 out = new Outer26();
        Outer26.Inner26 in = out.new Inner26() { };
        Main.check("ta10", in.val() == 10);

        // ----- a VARARGS record component (the array form is passed straight through) -----
        Pair26 p = new Pair26("h", new String[] {"a", "b"});
        Main.check("ta11", p.head().equals("h") && p.rest().length == 2);

        // ----- hexadecimal doubles, and an underscore inside an exponent -----
        Main.check("ta12", 0x1.8p1 == 3.0 && 0X.003p12 == 3.0 && 0x1p4 == 16.0);
        Main.check("ta13", 1e1_0 == 1e10);

        // ----- a `when` guard whose expression ends in `null` (which is not a
        // lambda parameter, however much `null -> "y"` looks like one) -----
        Object k = "kk";
        String w = switch (k) {
            case String str when str != null -> "s" + str.length();
            default -> "d";
        };
        Main.check("ta14", w.equals("s2") && k instanceof final String fs && fs.length() == 2);
    }
}

// ===== SECTION 29: unqualified instance and static field access =====
// A member body may name a field WITHOUT `this.` / `Cls.`, and a local or a
// parameter of the same name shadows it (JLS 6.5.6.1, 15.11). Every class in
// this file's other sections spells `this.w`, which is exactly why nothing here
// reached the bare form - both java halves aborted on it, with `unknown name: w`
// and `variable not defined: w`, while javac printed the field.

interface K29 { int LIM = 42; default int lim() { return LIM; } }

class A29 {
    int a = 1;
    static int sa = 11;
    int ga() { return a; }
    static int gsa() { return sa; }
}

class B29 extends A29 {
    int b = 2;
    static int sb = 22;
    String s = "hi";
    long lf = 5L;
    double df = 1.5;
    char cf = 'a';
    boolean flag = false;
    int[] arr = {1, 2, 3};

    int own() { return b; }
    int inherited() { return a; }
    int sum() { return a + b; }
    int statFromInst() { return sa + sb; }
    static int statFromStat() { return sa + sb; }
    void setB(int v) { b = v; }
    void addB(int v) { b += v; }
    int postB() { return b++; }
    int preB() { return ++b; }
    int decB() { return --b; }
    String cat(String x) { return s + x; }
    long addL(long v) { lf += v; return lf; }
    double dbl() { df *= 2; return df; }
    char nextC() { cf++; return cf; }
    boolean flip() { flag = !flag; return flag; }
    int arrSum() { int t = 0; for (int i = 0; i < arr.length; i++) { t += arr[i]; } return t; }
    void bumpArr() { arr[1] = arr[1] + 10; }
    int later() { return post; }
    int post = 77;                                   // named above its declaration
    int paramWins(int b) { return b; }
    int paramAndField(int b) { return b * 100 + this.b; }
    int localWins() { int b = 9; return b; }
    int blockScoped() { int r = b; { int b = 100; r += b; } return r + b; }
    int loopVar() { int t = 0; for (int b = 0; b < 3; b++) { t += b; } return t * 10 + this.b; }
}

class C29 implements K29 { int useConst() { return LIM + lim(); } }

class D29 {
    static int counter = 0;
    static int step() { counter++; return counter; }
    static void reset() { counter = 0; }
}

class S29Init {
    static int one = 3;
    static int two = one + 1;                        // a static naming a static
    static int three;
    static { three = two * 2; }                      // and so does the block
}

record R29(int x, int y) {
    R29 { x = x + 1; }                               // the compact body names components
    int total() { return x + y; }
}

class S29 {
    static void run() {
        int[] ops = {5, 2};
        B29 o = new B29();
        Main.check("uf1", o.own() == 2 && o.inherited() == 1 && o.sum() == 3);
        Main.check("uf2", o.ga() == 1 && A29.gsa() == 11);
        Main.check("uf3", o.statFromInst() == 33 && B29.statFromStat() == 33);
        Main.check("uf4", o.later() == 77);
        o.setB(ops[0]);
        Main.check("uf5", o.b == 5);
        o.addB(ops[1]);
        Main.check("uf6", o.b == 7);
        Main.check("uf7", o.postB() == 7 && o.b == 8);
        Main.check("uf8", o.preB() == 9 && o.b == 9);
        Main.check("uf9", o.decB() == 8 && o.b == 8);
        Main.check("uf10", o.cat("!").equals("hi!"));
        Main.check("uf11", o.addL(ops[1]) == 7L && o.lf == 7L);
        Main.check("uf12", o.dbl() == 3.0 && o.df == 3.0);
        Main.check("uf13", o.nextC() == 'b');
        Main.check("uf14", o.flip() && !o.flip());
        Main.check("uf15", o.arrSum() == 6);
        o.bumpArr();
        Main.check("uf16", o.arrSum() == 16);
        Main.check("uf17", o.paramWins(ops[0]) == 5);
        Main.check("uf18", o.paramAndField(ops[0]) == 508);
        Main.check("uf19", o.localWins() == 9);
        Main.check("uf20", o.blockScoped() == 116);
        Main.check("uf21", o.loopVar() == 38);
        Main.check("uf22", new C29().useConst() == 84);
        D29.reset();
        Main.check("uf23", D29.step() == 1 && D29.step() == 2 && D29.counter == 2);
        Main.check("uf24", S29Init.one == 3 && S29Init.two == 4 && S29Init.three == 8);
        R29 r = new R29(ops[0], ops[1]);
        Main.check("uf25", r.x() == 6 && r.total() == 8);
    }
}

// ===== SECTION 30: hashCode =====
// Object#hashCode, the overrides of it the JLS pins to an exact value, and the
// generated one a record gets. THE RECORD COMBINATION IS NOT SPECIFIED: JLS
// 8.10.3 requires only that it derive from the components' hashCodes and that
// equal records hash equally, so this project PINS OpenJDK's - h = 0, then
// h = h*31 + hash(component) per component at int width - and the exact values
// below are what java 24.0.2 answers. The digits ARE assertable because every
// component type's own hashCode is exact: Integer.hashCode is the value,
// Long's is v ^ (v >>> 32), Double's is doubleToLongBits ^ its own high half,
// Float's is floatToIntBits, String's is s[0]*31^(n-1)+..., Boolean's is
// 1231/1237, and Objects.hashCode(null) is 0. A one-component record's hash IS
// its component's, because 0*31 + h == h.

record Di30(double v) { }
record Fl30(float v) { }
record Lo30(long v) { }
record St30(String v) { }
record In30(int v) { }
record Ch30(char v) { }
record Bl30(boolean v) { }
record Ob30(Object v) { }
record Trip30(int a, String b, double c) { }
record Nest30(Trip30 t, int k) { }

class Plain30 { int x = 1; }
class Over30 { public int hashCode() { return 4242; } }
class Sub30 extends Over30 { }

class S30 {
    static String hex(int n) {
        String d = "0123456789abcdef";
        String out = "";
        int x = n;
        if (x == 0) { return "0"; }
        while (x != 0) { out = d.charAt(x & 15) + out; x = x >>> 4; }
        return out;
    }

    static void run() {
        int[] ints = {0, 1, -1, 2147483647, -2147483648, 65536};
        long[] longs = {0L, 1L, -1L, 9223372036854775807L, -9223372036854775808L, 4294967296L};
        String[] strs = {"", "a", "hello", "Hello, World!"};
        boolean[] bools = {true, false};
        char[] chars = {'a', 'Z'};

        // Integer.hashCode is the value itself.
        Main.check("hc1", new In30(ints[0]).hashCode() == 0 && new In30(ints[1]).hashCode() == 1);
        Main.check("hc2", new In30(ints[2]).hashCode() == -1 && new In30(ints[5]).hashCode() == 65536);
        Main.check("hc3", new In30(ints[3]).hashCode() == 2147483647 && new In30(ints[4]).hashCode() == -2147483648);

        // Long.hashCode is (int)(v ^ (v >>> 32)) - a 64-bit value the low half
        // alone cannot answer.
        Main.check("hc4", new Lo30(longs[0]).hashCode() == 0 && new Lo30(longs[1]).hashCode() == 1);
        Main.check("hc5", new Lo30(longs[2]).hashCode() == 0);
        Main.check("hc6", new Lo30(longs[3]).hashCode() == -2147483648);
        Main.check("hc7", new Lo30(longs[4]).hashCode() == -2147483648);
        Main.check("hc8", new Lo30(longs[5]).hashCode() == 1);

        // Double.hashCode reads doubleToLongBits, so +0.0 and -0.0 differ, NaN
        // has one pattern, and a subnormal is exact.
        double dz = 0.0;
        double dnan = dz / dz;
        double dinf = 1.0 / dz;
        Main.check("hc9", new Di30(0.0).hashCode() == 0);
        Main.check("hc10", new Di30(-0.0).hashCode() == -2147483648);
        Main.check("hc11", new Di30(1.0).hashCode() == 1072693248);
        Main.check("hc12", new Di30(-1.0).hashCode() == -1074790400);
        Main.check("hc13", new Di30(2.5).hashCode() == 1074003968);
        Main.check("hc14", new Di30(0.1).hashCode() == -1507852285);
        Main.check("hc15", new Di30(dnan).hashCode() == 2146959360);
        Main.check("hc16", new Di30(dinf).hashCode() == 2146435072 && new Di30(-dinf).hashCode() == -1048576);
        Main.check("hc17", new Di30(4.9E-324).hashCode() == 1);
        Main.check("hc18", new Di30(1.0E-320).hashCode() == 2024);
        Main.check("hc19", new Di30(1.7976931348623157E308).hashCode() == -2146435072);

        // Float.hashCode is floatToIntBits - a DIFFERENT function on the same
        // number, which is what the binary32 width is for.
        Main.check("hc20", new Fl30(0.0f).hashCode() == 0 && new Fl30(-0.0f).hashCode() == -2147483648);
        Main.check("hc21", new Fl30(1.0f).hashCode() == 1065353216);
        Main.check("hc22", new Fl30(-1.5f).hashCode() == -1077936128);
        Main.check("hc23", new Fl30(0.1f).hashCode() == 1036831949);
        Main.check("hc24", new Fl30(1.4E-45f).hashCode() == 1);
        Main.check("hc25", new Fl30(3.4028235E38f).hashCode() == 2139095039);
        Main.check("hc26", new Di30(1.0).hashCode() != new Fl30(1.0f).hashCode());

        // String.hashCode over UTF-16 code units.
        Main.check("hc27", new St30(strs[0]).hashCode() == 0 && strs[0].hashCode() == 0);
        Main.check("hc28", new St30(strs[1]).hashCode() == 97 && strs[1].hashCode() == 97);
        Main.check("hc29", strs[2].hashCode() == 99162322);
        Main.check("hc30", strs[3].hashCode() == 1498789909);

        Main.check("hc31", new Bl30(bools[0]).hashCode() == 1231 && new Bl30(bools[1]).hashCode() == 1237);
        Main.check("hc32", new Ch30(chars[0]).hashCode() == 97 && new Ch30(chars[1]).hashCode() == 90);
        Main.check("hc33", new Ob30(null).hashCode() == 0);

        // The multi-component combination, and a nested record - which recurses
        // through the component's OWN hashCode, not through this file.
        Main.check("hc34", new Trip30(ints[1], strs[1], 2.5).hashCode() == 1074007936);
        Main.check("hc35", new Nest30(new Trip30(ints[1], strs[1], 2.5), ints[1]).hashCode() == -1065492351);

        // A declared hashCode wins, and an inherited one is found.
        Main.check("hc36", new Over30().hashCode() == 4242 && new Sub30().hashCode() == 4242);

        // THE ONE INVARIANT JLS 8.10.3 ACTUALLY REQUIRES: equal records hash
        // equally. Asserted over every component type above, including the two
        // where equals is NOT === (NaN equals NaN, +0.0 does not equal -0.0).
        Main.check("hc37", eqPair(new Di30(dnan), new Di30(dnan)));
        Main.check("hc38", eqPair(new Di30(-0.0), new Di30(-0.0)));
        Main.check("hc39", !new Di30(0.0).equals(new Di30(-0.0))
                           && new Di30(0.0).hashCode() != new Di30(-0.0).hashCode());
        Main.check("hc40", eqPair(new Lo30(longs[3]), new Lo30(longs[3])));
        Main.check("hc41", eqPair(new St30(strs[3]), new St30(strs[3])));
        Main.check("hc42", eqPair(new Trip30(ints[2], strs[2], 0.1), new Trip30(ints[2], strs[2], 0.1)));
        Main.check("hc43", eqPair(new Nest30(new Trip30(ints[2], strs[2], 0.1), ints[3]),
                                  new Nest30(new Trip30(ints[2], strs[2], 0.1), ints[3])));

        // Object#hashCode: stable within a run, and the digits after the `@` in
        // toString ARE it (Object.toString is getName()+"@"+toHexString(hashCode())).
        Plain30 p = new Plain30();
        int ph = p.hashCode();
        Main.check("hc44", ph == p.hashCode());
        String rendered = "" + p;
        Main.check("hc45", rendered.substring(rendered.indexOf("@") + 1).equals(S30.hex(ph)));
    }

    static boolean eqPair(Object a, Object b) {
        return a.equals(b) && a.hashCode() == b.hashCode();
    }
}

// ===== END SECTIONS =====

// ===== SECTION 27: the near side of javac's post-shape parser checks =====
// Every rule the grammar carries for javac's post-shape diagnostics (`var` as a
// restricted type name, the `_` rules, varargs placement, repeated modifiers,
// not.stmt, and the scanner's literal limits) REFUSES something, and a refusal
// cannot be asserted from inside a program that runs. What this section pins is
// the other side of each of those rules: the legal shape that sits one step away
// from the refusal and has to keep parsing. The corpus reject row in
// tests/reference/sweep.sh measures the refusals themselves.

interface Fn27 { int apply(int a, int b); }
interface Run27 { void go(); }

class Box27 {
    int v = 5;
    static Box27 mk(Box27 b) { return b; }
}

record Rec27(int a) {
    static final int SHARED = 3;      // a record may declare a STATIC field
    int twice() { return this.a * 2; }
}

class S27 {
    @Deprecated public static final int K27 = 7;

    static int sum(int seed, int... xs) {
        int t = seed;
        for (int x : xs) { t = t + x; }
        return t;
    }

    // `case null, default ->` (JEP 441) is NOT here: both halves already refuse
    // it - the interpreter parses the label and then cannot resolve the name
    // `default`, and the compiler grammar does not parse it at all. The guard
    // that refuses `case default:` deliberately sits on the FIRST label only, so
    // it neither creates nor closes that gap.
    static String kind(Object o) {
        return switch (o) {
            case String s -> "s" + s.length();
            default -> "nd";
        };
    }

    static void run() {
        // ----- a PREFIX ++/-- whose target is a member CHAIN, not a bare name -----
        Box27 b = new Box27();
        --Box27.mk(b).v;
        Main.check("pp1", b.v == 4);
        ++b.v;
        Main.check("pp2", b.v == 5);

        // ----- a class literal over a PRIMITIVE type (`int.class`), which is no
        // member chain: `int` stopped being readable as a variable name -----
        Main.check("pp3", int.class == int.class && char.class == char.class);
        Main.check("pp4", int[].class == int[].class);

        // ----- literals one step inside the scanner's limits -----
        Main.check("pp5", 1e1__0 == 1e10 && 1_0_0 == 100 && 0x1__0 == 16);
        Main.check("pp6", 017 == 15 && 0 == 0 && 2147483647 > 0);
        Main.check("pp7", 12345678901234L > 0L && 1e308 > 0.0 && 1e-308 > 0.0);
        Main.check("pp8", "a\sb".equals("a b") && "a\102c".equals("aBc"));

        // ----- a pattern switch with a `default ->` arm: `default` is refused only
        // as a case LABEL (`case default:`), never as the default arm itself -----
        Main.check("pp9", S27.kind("ab").equals("s2") && S27.kind(3).equals("nd"));

        // ----- lambda parameters: all implicit, all `var`, all explicit -----
        Fn27 f1 = (x, y) -> x + y;
        Fn27 f2 = (var x, var y) -> x * y;
        Fn27 f3 = (int x, int y) -> x - y;
        Main.check("pp10", f1.apply(2, 3) == 5 && f2.apply(2, 3) == 6 && f3.apply(5, 3) == 2);

        // ----- a varargs parameter, in last position -----
        Main.check("pp11", S27.sum(0, 1, 2, 3) == 6 && S27.sum(4) == 4);

        // ----- statement expressions the not.stmt guard must still admit -----
        int sh = 1;
        sh <<= 4;
        Main.check("pp12", sh == 16);
        sh >>= 2;
        Main.check("pp13", sh == 4);
        Run27 r = new Run27() { public void go() { } };
        r.go();
        Main.check("pp14", S27.K27 == 7 && Rec27.SHARED == 3 && new Rec27(4).twice() == 8);

        // ----- a `var` local, which the compound-declaration rule still allows
        // one of, and a try that has a finally but no catch -----
        var one = 1;
        int seen = 0;
        try {
            seen = one;
        } finally {
            seen = seen + 1;
        }
        Main.check("pp15", seen == 2);
    }
}

// ===== SECTION 28: phase-one unicode escapes, Java's escape set, `{,}` and
// stacked colon labels =====
// JLS 3.3 says a unicode escape is translated BEFORE the scanner runs, so the
// six characters \u0078 here really are the identifier x and \u0041 inside a string
// literal really is an A - the grammars run that translation as a source
// pre-pass (UniPre). The near side of it is pinned too: a backslash preceded by
// an ODD number of backslashes is not eligible, so "\\u0041" stays six
// characters. The escape may also produce a raw CONTROL character, which is a
// legal string / char character (JStrPlain, CharPlain).
// Java's own escape set (JLS 3.10.7) carries the octal escapes and \s, which
// the shared unescapeJs does not know - both halves decode them with jUnescape
// now. `{,}` is javac's single-comma array initializer, and `case 3, 4:` is a
// colon case with stacked labels, which falls through into the next one.
// Verified against OpenJDK 24.0.2: javac -XDshould-stop.ifNoError=PARSE exits 0
// and `java` prints 15 of the checks below with 0 failures.

class S28 {
    static int classify(int n) {
        int r = 0;
        switch (n) {
            case 1, 2: r = 12; break;
            case 3, 4: r = 34;
            case 5: r = r + 100; break;
            default: r = -1;
        }
        return r;
    }
    static String pick(int n) {
        return switch (n) {
            case 1, 2: yield "lo";
            case 3, 4: yield "hi";
            default: yield "?";
        };
    }

    static void run() {
        // ----- phase one: the escape IS the character -----
        int \u0078 = 41;
        Main.check("uu1", x + 1 == 42);
        Main.check("uu2", "\u0041\u0042".equals("AB"));
        Main.check("uu3", '\u0041' == 'A');
        String bs = "\\u0041";
        Main.check("uu4", bs.length() == 6 && bs.charAt(0) == '\\' && bs.charAt(1) == 'u');
        String nul = "a\u0000b";
        Main.check("uu5", nul.length() == 3 && nul.charAt(1) == 0);
        Main.check("uu6", '\u0010' == 16);

        // ----- Java's escape set: the octal forms and \s -----
        Main.check("uu7", "A\102C".equals("ABC"));
        Main.check("uu8", "x\sy".equals("x y"));
        Main.check("uu9", '\s' == ' ');

        // ----- array initializers: the single comma and the trailing one -----
        int[] e = {,};
        Main.check("uu10", e.length == 0);
        int[] t = {1, 2, 3,};
        Main.check("uu11", t.length == 3 && t[2] == 3);

        // ----- stacked colon labels, in a statement switch and an expression one -----
        Main.check("uu12", S28.classify(1) == 12 && S28.classify(2) == 12);
        Main.check("uu13", S28.classify(3) == 134 && S28.classify(4) == 134);
        Main.check("uu14", S28.classify(5) == 100 && S28.classify(9) == -1);
        Main.check("uu15", S28.pick(2).equals("lo") && S28.pick(4).equals("hi") && S28.pick(8).equals("?"));
    }
}

// ===== SECTION 31: Object#toString and Object#equals =====
// The other two members of the java.lang.Object contract a class inherits
// WITHOUT declaring - the same gap hashCode had until 77eb804. A declared or
// inherited method wins; then a record's canonical form (JLS 8.10.3, which DOES
// specify both of these exactly, unlike the hashCode combination); then Object's
// own: identity for equals, and getClass().getName() + "@" +
// Integer.toHexString(hashCode()) for toString.
//
// THE DIGITS AFTER THE @ ARE NOT ASSERTABLE. Real java prints
// System.identityHashCode, an address-derived value that differs between two runs
// of the same program, so what is asserted is the CONTRACT that ties the two
// together - o.toString() is the class' binary name, an '@', and the same digits
// o.hashCode() renders in hex - plus self-consistency across two calls. That
// holds in real java and here, and it is exactly what a wrong implementation
// breaks.

class Pl31 { int x = 1; }
class Ts31 { public String toString() { return "TS!"; } }
class SubTs31 extends Ts31 { }
class Eq31 {
    int w;
    Eq31(int w) { this.w = w; }
    public boolean equals(Object o) { return o instanceof Eq31 && ((Eq31) o).w == this.w; }
}
class SubEq31 extends Eq31 { SubEq31(int w) { super(w); } }
record R31(int a, String b) { }
record Rd31(double d) { }
record Rn31(R31 r) { }
enum E31 { RED, BLUE }

class S31 {
    static String hex(int n) {
        String d = "0123456789abcdef";
        String out = "";
        int x = n;
        if (x == 0) { return "0"; }
        while (x != 0) { out = d.charAt(x & 15) + out; x = x >>> 4; }
        return out;
    }

    static void run() {
        Object[] plains = { new Pl31(), new Pl31() };
        Object[] tss = { new Ts31(), new SubTs31() };
        Object[] recs = { new R31(1, "x"), new R31(1, "x"), new R31(2, "x") };

        // ----- toString: a declared one, and an inherited one -----
        Main.check("ts1", tss[0].toString().equals("TS!"));
        Main.check("ts2", tss[1].toString().equals("TS!"));
        Main.check("ts3", ("" + tss[0]).equals("TS!"));

        // ----- toString: a record's canonical form (JLS 8.10.3 specifies it) -----
        Main.check("ts4", recs[0].toString().equals("R31[a=1, b=x]"));
        Main.check("ts5", new Rd31(2.5).toString().equals("Rd31[d=2.5]"));
        Main.check("ts6", new Rn31(new R31(3, "y")).toString().equals("Rn31[r=R31[a=3, b=y]]"));

        // ----- toString: an enum constant's is its name -----
        Main.check("ts7", E31.RED.toString().equals("RED"));

        // ----- toString: Object's own, on a class that declares none -----
        String s0 = plains[0].toString();
        Main.check("ts8", s0.equals("Pl31@" + hex(plains[0].hashCode())));
        Main.check("ts9", s0.equals(plains[0].toString()));
        Main.check("ts10", !s0.equals(plains[1].toString()));
        Main.check("ts11", s0.equals("" + plains[0]));
        Main.check("ts12", s0.indexOf("@") == 4);

        // ----- toString on the values that are not instances at all -----
        Main.check("ts13", "abc".toString().equals("abc"));
        Main.check("ts14", Integer.valueOf(7).toString().equals("7"));

        // ----- equals: Object's own is IDENTITY -----
        Main.check("eq1", plains[0].equals(plains[0]));
        Main.check("eq2", !plains[0].equals(plains[1]));
        Main.check("eq3", !plains[0].equals(null));
        Main.check("eq4", !plains[0].equals("Pl31"));

        // ----- equals: a declared one wins, and an inherited one is used -----
        Main.check("eq5", new Eq31(1).equals(new Eq31(1)));
        Main.check("eq6", !new Eq31(1).equals(new Eq31(2)));
        Main.check("eq7", new SubEq31(3).equals(new SubEq31(3)));

        // ----- equals: a record's is component-wise -----
        Main.check("eq8", recs[0].equals(recs[1]));
        Main.check("eq9", !recs[0].equals(recs[2]));
        Main.check("eq10", !recs[0].equals(null));
        Main.check("eq11", new Rn31(new R31(3, "y")).equals(new Rn31(new R31(3, "y"))));

        // A double component compares with Double.compare, so NaN EQUALS NaN and
        // +0.0 does NOT equal -0.0 - both the opposite of ==.
        double[] ds = { 0.0, -0.0, 0.0 / 0.0 };
        Main.check("eq12", !new Rd31(ds[0]).equals(new Rd31(ds[1])));
        Main.check("eq13", new Rd31(ds[2]).equals(new Rd31(ds[2])));

        // ----- equals: strings, boxes and enum constants -----
        String[] strs = { "ab", "ab", "ba" };
        Main.check("eq14", strs[0].equals(strs[1]) && !strs[0].equals(strs[2]));
        Main.check("eq15", E31.RED.equals(E31.RED) && !E31.RED.equals(E31.BLUE));

        // ----- two PRIMITIVES OF DIFFERENT KINDS are never equal -----
        // Integer#equals asks `instanceof Integer` before it compares, so
        // ((Object) 1).equals(1L) is false - where a by-value comparison would
        // say true. This was a live halves divergence the moment Object#equals
        // started routing through the record-component comparison.
        Object[] prims = { (Object) 1, (Object) 1L, (Object) 1.0, (Object) 1.0f,
                           (Object) 'a', (Object) true, "1" };
        Main.check("eq18", prims[0].equals(prims[0]) && !prims[0].equals(prims[1]));
        Main.check("eq19", !prims[0].equals(prims[2]) && !prims[1].equals(prims[2]));
        Main.check("eq20", !prims[2].equals(prims[3]) && !prims[3].equals(prims[2]));
        Main.check("eq21", !prims[4].equals(prims[6]) && !prims[6].equals(prims[4]));
        Main.check("eq22", !prims[5].equals(prims[0]) && !prims[0].equals(prims[6]));
        Main.check("eq23", prims[1].equals(prims[1]) && prims[2].equals(prims[2])
                           && prims[3].equals(prims[3]) && prims[4].equals(prims[4])
                           && prims[5].equals(prims[5]) && prims[6].equals(prims[6]));
        // A DECLARED equals(Object) still gets to see a String argument - the
        // guard screens primitives only, never an instance.
        Main.check("eq24", !new Eq31(1).equals(prims[6]));

        // ----- the equals / hashCode contract, over the pair 77eb804 opened -----
        Main.check("eq16", recs[0].equals(recs[1]) && recs[0].hashCode() == recs[1].hashCode());
        Main.check("eq17", plains[0].equals(plains[0]) && plains[0].hashCode() == plains[0].hashCode());
    }
}

// ===== SECTION 32: an inner or anonymous class naming an OUTER field =====
// javac resolves an unqualified name in an inner or anonymous class body against
// the innermost ENCLOSING class that declares it (JLS 6.5.6.1), which needs the
// enclosing instance at run time. Every creation site of a nested or anonymous
// class now records it, and `Outer.this` walks the same chain - correctly at more
// than one level, which one unconditional hop was not.

interface Sup32 { int get(); }

class Out32 {
    int w = 41;
    String nm = "out";
    static int sw = 7;

    class In32 {
        int q = 100;
        int viaOuter() { return w + 1; }
        String viaOuterStr() { return nm + "!"; }
        int viaOuterThis() { return Out32.this.w + 2; }
        int viaStatic() { return sw + 3; }
        int ownWins() { return q; }
        int localWins() { int w = 5; return w; }
        int paramWins(int w) { return w; }
        int writeOuter() { w = w + 10; return w; }
        class Deep32 { int twoLevels() { return w + 4; } int deepOuterThis() { return Out32.this.w + 5; } }
    }

    Sup32 anon() { return new Sup32() { public int get() { return w + 6; } }; }
    Sup32 anonStr() { return new Sup32() { public int get() { return nm.length() + 7; } }; }
    Sup32 anonOuterThis() { return new Sup32() { public int get() { return Out32.this.w + 8; } }; }
    Sup32 anonStatic() { return new Sup32() { public int get() { return sw + 9; } }; }
    Sup32 anonOwn() { return new Sup32() { int f = 60; public int get() { return f; } }; }
    Sup32 anonLocalWins() { return new Sup32() { public int get() { int w = 3; return w; } }; }
    Sup32 lam() { return () -> w + 11; }

    int viaBareNew() { In32 i = new In32(); return i.viaOuter(); }
}

class S32 {
    static void run() {
        Out32 o = new Out32();
        Out32.In32[] ins = { o.new In32() };

        // ----- a NAMED inner class -----
        Main.check("oc1", ins[0].viaOuter() == 42);
        Main.check("oc2", ins[0].viaOuterStr().equals("out!"));
        Main.check("oc3", ins[0].viaOuterThis() == 43);
        Main.check("oc4", ins[0].viaStatic() == 10);
        Main.check("oc5", ins[0].ownWins() == 100);
        Main.check("oc6", ins[0].localWins() == 5);
        Main.check("oc7", ins[0].paramWins(9) == 9);

        // `new In32()` inside an instance method captures the enclosing instance
        // implicitly, exactly as `o.new In32()` does.
        Main.check("oc8", o.viaBareNew() == 42);

        // TWO levels out - the case one unconditional __outer hop got wrong.
        Out32.In32.Deep32[] deeps = { ins[0].new Deep32() };
        Main.check("oc9", deeps[0].twoLevels() == 45);
        Main.check("oc10", deeps[0].deepOuterThis() == 46);

        // ----- an ANONYMOUS class -----
        Sup32[] anons = { o.anon(), o.anonStr(), o.anonOuterThis(), o.anonStatic(),
                          o.anonOwn(), o.anonLocalWins(), o.lam() };
        Main.check("oc11", anons[0].get() == 47);
        Main.check("oc12", anons[1].get() == 10);
        Main.check("oc13", anons[2].get() == 49);
        Main.check("oc14", anons[3].get() == 16);
        Main.check("oc15", anons[4].get() == 60);
        Main.check("oc16", anons[5].get() == 3);
        Main.check("oc17", anons[6].get() == 52);

        // ----- an unqualified WRITE through the outer hop -----
        Out32 o2 = new Out32();
        Out32.In32[] ins2 = { o2.new In32() };
        Main.check("oc18", ins2[0].writeOuter() == 51);
        Main.check("oc19", o2.w == 51);
    }
}

// ===== SECTION 33: a field whose name equals a type's =====
// JLS 6.4.2: a variable name OBSCURES a type name. Every engine here answered the
// CLASS DESCRIPTOR for `Ctr33` inside Ctr33's own methods, because the descriptor
// is declared in the same scope chain the bare name reads. The controls below are
// what a wrong fix breaks: a bare type name used where NO field of that name is in
// scope must still be the type.

class Ctr33 {
    int Ctr33 = 7;
    static int sf = 9;
    int read() { return Ctr33; }
    int write() { Ctr33 = 8; return Ctr33; }
    int step() { Ctr33 = 20; Ctr33++; ++Ctr33; return Ctr33; }
    int compound() { Ctr33 = 4; Ctr33 += 6; return Ctr33; }
    int shadowedByLocal() { int Ctr33 = 3; return Ctr33; }
    int shadowedByParam(int Ctr33) { return Ctr33; }
}

class Box33 { static int Box33 = 11; static int read() { return Box33; } }

class S33 {
    static void run() {
        Ctr33[] cs = { new Ctr33(), new Ctr33(), new Ctr33(), new Ctr33(), new Ctr33() };
        Main.check("ob1", cs[0].read() == 7);
        Main.check("ob2", cs[1].write() == 8);
        Main.check("ob3", cs[2].step() == 22);
        Main.check("ob4", cs[3].compound() == 10);
        Main.check("ob5", cs[4].shadowedByLocal() == 3);
        Main.check("ob6", cs[4].shadowedByParam(5) == 5);
        Main.check("ob7", cs[0].Ctr33 == 7);

        // A STATIC field with the class' own name, read from a static method.
        Main.check("ob8", Box33.read() == 11);

        // ----- the controls: a bare type name with no field of that name in
        // scope is still the type, in every spelling that reads one -----
        Main.check("ob9", Ctr33.sf == 9);
        Main.check("ob10", new Ctr33() != null);
        Main.check("ob11", new Ctr33().read() == 7);
        Object x = new Ctr33();
        Main.check("ob12", x instanceof Ctr33);
        Main.check("ob13", ((Ctr33) x).read() == 7);
    }
}

// ===== SECTION 34: Integer/Long.toHexString, Double.NaN, Boolean.valueOf, and
// the double reading of a value that is not one =====
//
// docs/todo.md 1.10 (the java bullets) and 1.2. Every value in the first three
// groups was EXECUTED against `java` 24.0.2 on this machine, so they are oracle
// values and not spec citations.
//
// WHAT WAS BROKEN. Integer.toHexString, Long.toHexString, Double.NaN,
// Boolean.valueOf and Boolean.parseBoolean did not exist in ANY of the four
// engines: the descriptors java.lang.Integer / Long carried no toHexString at all
// and java.lang.Double / Boolean were not descriptors, so the program aborted with
// "unknown method 'toHexString'" / "unknown class". java.lang.Integer#toHexString
// is the UNSIGNED reading at the value's own width, which is why -1 is "ffffffff"
// and not "-1".
//
// THE LAST GROUP IS docs/todo.md 1.2 AND IT IS NOT JAVA'S ANSWER. A read past the
// end of an array throws ArrayIndexOutOfBoundsException in real java; this engine
// answers `undefined`, and the three engines then DISAGREED about what that is
// worth in arithmetic - the interpreter said `a[5] + 1 == 1`, llvm.Run said NaN,
// and the native binary said 1. rtjNum (languages/lib/runtime-jvm.metajs) now
// reads it as NaN in layer 2, which is what abnf/jsrt.go's rt.toNumber always
// answered, and java-interpreter.abnf's binOp agrees. The value asserted here is
// therefore ENGINE BEHAVIOUR, deliberately, and the point of asserting it is that
// all four engines say the same thing. `-` `*` `/` `%` `<` `>` reach jvLow32
// instead of rtjNum and read the same operand as 0, in all four engines; those are
// asserted too, so a future widening of the NaN arm cannot pass silently.
//
// Every operand is read out of an ARRAY so no constant folder can answer early.
class S34 {
    static void run() {
        int[] n = { 255, -1, 0, 305419896 };
        long[] l = { -1L, 1234605616436508552L, 255L, 0L };
        String[] w = { "true", "TrUe", "no", "TRUE", "" };

        // ----- Integer.toHexString: unsigned, at 32 bits -----
        Main.check("hx1", Integer.toHexString(n[0]).equals("ff"));
        Main.check("hx2", Integer.toHexString(n[1]).equals("ffffffff"));
        Main.check("hx3", Integer.toHexString(n[2]).equals("0"));
        Main.check("hx4", Integer.toHexString(n[3]).equals("12345678"));

        // ----- Long.toHexString: unsigned, at 64 bits, so the low half is
        // zero-padded to eight digits when the high half is not zero -----
        Main.check("hx5", Long.toHexString(l[0]).equals("ffffffffffffffff"));
        Main.check("hx6", Long.toHexString(l[1]).equals("1122334455667788"));
        Main.check("hx7", Long.toHexString(l[2]).equals("ff"));
        Main.check("hx8", Long.toHexString(l[3]).equals("0"));

        // ----- Double.NaN: a double, unequal to itself, and it PRINTS "NaN" -----
        double d = Double.NaN;
        Main.check("nan1", d != d);
        Main.check("nan2", ("" + d).equals("NaN"));
        Main.check("nan3", !(d < 1.0) && !(d > 1.0) && !(d == 1.0));

        // ----- Boolean.valueOf / parseBoolean: equalsIgnoreCase("true") -----
        Main.check("bv1", Boolean.valueOf(w[0]));
        Main.check("bv2", Boolean.valueOf(w[1]));
        Main.check("bv3", !Boolean.valueOf(w[2]));
        Main.check("bv4", !Boolean.valueOf(w[4]));
        Main.check("bv5", Boolean.parseBoolean(w[3]));
        Main.check("bv6", !Boolean.parseBoolean(w[2]));
        Main.check("bv7", Boolean.TRUE && !Boolean.FALSE);

        // ----- docs/todo.md 1.2: the reading of a value that is not a number.
        // ENGINE BEHAVIOUR, not java's - see the note above the class.
        //
        // THE try/catch IS LOAD-BEARING AND IT IS FOR THE REAL TOOLCHAIN, not for
        // us: `java tests/java-test-full.java` runs this whole file green, and it
        // must keep doing so. Real java throws at the first `a[5]` and skips the
        // eight checks; no engine here throws, so all four run them and must agree.
        int[] a = new int[2];
        Main.check("un0", a[0] + 1 == 1);          // the control: in range, still 0
        try {
            Main.check("un1", ("" + (a[5] + 1)).equals("NaN"));
            Main.check("un2", ("" + (1 + a[5])).equals("NaN"));
            Main.check("un3", ("" + a[5]).equals("null"));
            // The five that do NOT reach rtjNum, in every engine. `-` `*` `/` `%`
            // and the relations route through the integral path, which answers 0
            // outright when an operand is not integral - so `1 - a[5]` is 0, not 1.
            Main.check("un4", 1 - a[5] == 0);
            Main.check("un5", a[5] * 2 == 0);
            Main.check("un6", !(a[5] < 1));
            Main.check("un7", !(a[5] > 1));
            Main.check("un8", a[5] % 2 == 0);
        } catch (ArrayIndexOutOfBoundsException e) {
            // Only real java gets here.
        }
    }
}
