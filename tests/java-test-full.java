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
// Oracle: `java -Duser.language=en -Duser.country=US tests/java-test-full.java`
// (java 24.0.2). String.format follows the default locale, so fmt1-fmt3 fail
// under another one (a German locale writes 1.234.567 and 1,01).
//
// Deliberately out of scope (not core syntax, or unrunnable in this harness):
// packages, and the standard library beyond the java.lang implicits (String,
// Math, StrictMath, System and friends) and the java.util types imported
// below; the functional interfaces here are declared locally instead of
// importing java.util.function. Also out: modules - a module-info.java IS parsed
// (see ModuleDecl in the grammar) but is a file of its own, so it cannot be a
// section here - threads (and synchronized), reflection, javadoc.
//
// Hand-written for the metacompiler project (Apache-2.0, no copied test-suite
// code), organized after the Java Language Specification (SE 21) with the
// ANTLR grammars-v4 java grammar as a coverage checklist.

import java.util.AbstractList;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Comparator;
import java.util.ConcurrentModificationException;
import java.util.HashMap;
import java.util.HashSet;
import java.util.Iterator;
import java.util.List;
import java.util.Map;
import java.util.NoSuchElementException;
import java.util.Optional;
import java.util.OptionalInt;
import java.util.Set;
import java.util.stream.Collectors;
import java.util.stream.DoubleStream;
import java.util.stream.IntStream;
import java.util.stream.LongStream;
import java.util.stream.Stream;

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
        S35.run(); // SECTION-CALL 35
        S36.run(); // SECTION-CALL 36
        S37.run(); // SECTION-CALL 37
        S38.run(); // SECTION-CALL 38
        S39.run(); // SECTION-CALL 39
        S40.run(); // SECTION-CALL 40
        S41.run(); // SECTION-CALL 41
        S42.run(); // SECTION-CALL 42
        S43.run(); // SECTION-CALL 43
        S44.run(); // SECTION-CALL 44
        S45.run(); // SECTION-CALL 45
        S46.run(); // SECTION-CALL 46
        S47.run(); // SECTION-CALL 47
        S48.run(); // SECTION-CALL 48
        S49.run(); // SECTION-CALL 49
        S50.run(); // SECTION-CALL 50
        S51.run(); // SECTION-CALL 51
        S52.run(); // SECTION-CALL 52
        S53.run(); // SECTION-CALL 53
        S54.run(); // SECTION-CALL 54
        S55.run(); // SECTION-CALL 55
        S56.run(); // SECTION-CALL 56
        S57.run(); // SECTION-CALL 57
        S58.run(); // SECTION-CALL 58
        S59.run(); // SECTION-CALL 59
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
        // char is an INTEGRAL type that PRINTS as its glyph: "x" + c concatenates
        // the glyph, and charAt(1) + 0 is 101, not "e0" (JLS 4.2.1, 15.18.1).
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
            Main.check("iof5a", false);
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
// The component types the generated equals cannot compare with ===: a
// floating-point component (Double.compare, not ==), a boxed integral one (a long
// is a box at every magnitude) and a REFERENCE one (Objects.equals dispatches the
// component's own equals).
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
        // A CHAR record component. A char is the one shape === answers differently
        // in the engines: the Go twin's char is a comparable struct, layer 2's is a
        // {__char} box that === settles by identity. Every operand is read out of
        // an array so the grammar's constant folder cannot answer it.
        char[] gs = {'q', 'q', 'r'};
        int[] gn = {5, 5, 6};
        Main.check("rec7", new Glyph(gs[0], gn[0]).equals(new Glyph(gs[1], gn[1])));
        Main.check("rec8", !new Glyph(gs[0], gn[0]).equals(new Glyph(gs[2], gn[1]))
                        && !new Glyph(gs[0], gn[0]).equals(new Glyph(gs[1], gn[2])));
        // The rest of JLS 8.10.3, checked against java 24.0.2. Every operand is
        // read out of an array.
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
        // long / byte / short are BOXES here, so identity would call two equal
        // values unequal.
        long[] lz = {5000000000L, 5000000000L, 5000000001L};
        byte[] bz = {7, 7, 8};
        short[] sz = {300, 300, 301};
        Main.check("rec11", new RWide(lz[0], bz[0], sz[0]).equals(new RWide(lz[1], bz[1], sz[1])));
        Main.check("rec12", !new RWide(lz[2], bz[0], sz[0]).equals(new RWide(lz[0], bz[0], sz[0]))
                        && !new RWide(lz[0], bz[2], sz[0]).equals(new RWide(lz[0], bz[0], sz[0]))
                        && !new RWide(lz[0], bz[0], sz[2]).equals(new RWide(lz[0], bz[0], sz[0])));
        // A reference component is Objects.equals: a nested record compares
        // component-wise and a user-declared equals is CALLED.
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
        // false (it must not read __class off the null).
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
        // The built-in hierarchy is the JDK's: UnsupportedOperationException
        // extends RuntimeException, StackOverflowError VirtualMachineError.
        String arm = "";
        try { throw new UnsupportedOperationException("u"); }
        catch (IllegalArgumentException e) { arm = "IAE"; }
        catch (RuntimeException e) { arm = "RTE"; }
        Main.check("exc6", arm.equals("RTE"));
        arm = "";
        try { throw new StackOverflowError("s"); }
        catch (VirtualMachineError e) { arm = "VME"; }
        Main.check("exc7", arm.equals("VME"));
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
// against `java` (JDK 24) before it was written down: 1.0 / 3.0 is not 0,
// println(1.0) prints "1.0", and Infinity / NaN / -0.0 exist.
class S23 {
    static String s(double d) { return "" + d; }
    static String f(float v) { return "" + v; }
    static float third() { return 1.0f / 3.0f; }
    static double half() { return 0.5; }
    static float fWide = 16777217;
    static double dWide = 7f / 3;
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
        // false. (jsCompare's NaN sentinel 2 must not be read as an ordering.)
        Main.check("flt10a", !(nan < 1.0) && !(nan > 1.0) && !(nan <= 1.0) && !(nan >= 1.0));
        Main.check("flt10b", !(1.0 < nan) && !(1.0 > nan) && !(1.0 <= nan) && !(1.0 >= nan));
        Main.check("flt10c", !(nan <= nan) && !(nan >= nan) && !(nan < nan) && !(nan > nan));
        float fnan = 0.0f / 0.0f;
        Main.check("flt10d", !(fnan >= 1.0f) && !(fnan <= 1.0f) && !(fnan > 1.0f) && !(fnan < 1.0f));
        // JLS 15.21.1: == between two numeric operands applies binary numeric
        // promotion (5.6.2) first, so a long is compared against a double AS a
        // double (java 24.0.2 agrees). The operands come out of arrays so the
        // constant folder cannot answer at compile time.
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
        // ----- the float WIDTH -----
        // `float` is a binary32, not a double spelled differently. Every value
        // below was read off `java` 24.0.2 before it was written down; each line
        // discriminates the float result from the double one.
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
        // A declaration converts its initializer (JLS 5.2): a float widened to
        // double renders at double precision, an int stored in a float rounds.
        float f73 = 7f / 3;
        double d73 = f73;
        Main.check("flt45", ("" + d73).equals("2.3333332538604736") && ("" + f73).equals("2.3333333"));
        Main.check("flt46", ("" + S23.fWide).equals("1.6777216E7") && ("" + S23.dWide).equals("2.3333332538604736"));
    }
}

// ===== SECTION 24: value rendering (toString / println) =====
// What a value RENDERS as: the string `+` builds and the text println writes.
// Checked against `java` (JDK 24) before it was written down. The identity hash
// after the '@' is deliberately NOT asserted - real java prints
// System.identityHashCode, which differs between two runs of the SAME program,
// so only the class-name prefix is reproducible.
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
        try { int z = 0; int q = 1 / z; Main.check("int27a", q == 99); }
        catch (ArithmeticException e) { threw = true; msg = e.getMessage(); }
        Main.check("int27", threw && msg.equals("/ by zero"));
        boolean lthrew = false;
        try { long lz = 0; long lq = 1L % lz; Main.check("int28a", lq == 99); }
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
        // (A long must not take the int overload's 32 bit wrap.)
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

        // ----- a type annotation on the type of a method REFERENCE (JLS 15.13) -----
        Mk26 m = @TU Box26::new;
        Main.check("ta15", m.make().v == 0);
    }
}
interface Mk26 { Box26 make(); }

// ===== SECTION 29: unqualified instance and static field access =====
// A member body may name a field WITHOUT `this.` / `Cls.`, and a local or a
// parameter of the same name shadows it (JLS 6.5.6.1, 15.11). Every class in
// this file's other sections spells `this.w`, so this section is the one that
// reaches the bare form.

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
        // doubleToLongBits canonicalises every NaN, Double.NaN included.
        Main.check("hc46", new Di30(Double.NaN).hashCode() == 2146959360 && new Fl30(0f / 0f).hashCode() == 2143289344);

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

    // `case null, default` (JLS 14.11.1) is the one label list that may name
    // `default`; its arm takes null AND every value no other arm took, in the
    // arrow and the colon form, as a statement and as an expression.
    static String nd(Object o) {
        return switch (o) {
            case Integer i when i > 3 -> "big";
            case null, default -> "nd";
        };
    }
    static int ndColon(Integer o) {
        int r = 0;
        switch (o) {
            case 1: r = 10; break;
            case null, default: r = 99;
        }
        return r;
    }
    static int ndYield(Object o) {
        return switch (o) {
            case Integer i: yield i;
            case null, default: yield -1;
        };
    }
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

        // ----- `case null, default`: the null label and the default arm at once -----
        Main.check("pp16", S27.nd(null).equals("nd") && S27.nd(7).equals("big") && S27.nd("s").equals("nd"));
        Main.check("pp17", S27.ndColon(null) == 99 && S27.ndColon(1) == 10 && S27.ndColon(5) == 99);
        Main.check("pp18", S27.ndYield(null) == -1 && S27.ndYield(4) == 4 && S27.ndYield("s") == -1);
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
// the shared unescapeJs does not know - both halves decode them with jUnescape.
// `{,}` is javac's single-comma array initializer, and `case 3, 4:` is a
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
// WITHOUT declaring. A declared or
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
        // say true.
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

        // ----- the equals / hashCode contract -----
        Main.check("eq16", recs[0].equals(recs[1]) && recs[0].hashCode() == recs[1].hashCode());
        Main.check("eq17", plains[0].equals(plains[0]) && plains[0].hashCode() == plains[0].hashCode());
    }
}

// ===== SECTION 32: an inner or anonymous class naming an OUTER field =====
// javac resolves an unqualified name in an inner or anonymous class body against
// the innermost ENCLOSING class that declares it (JLS 6.5.6.1), which needs the
// enclosing instance at run time. Every creation site of a nested or anonymous
// class records it, and `Outer.this` walks the same chain - correctly at more
// than one level.

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

        // TWO levels out - one unconditional __outer hop would get this wrong.
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
// JLS 6.4.2: a variable name OBSCURES a type name, even though the class
// descriptor is declared in the same scope chain the bare name reads. The
// controls below are what a wrong fix breaks: a bare type name used where NO
// field of that name is in scope must still be the type.

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
// Every value in the first three groups was EXECUTED against `java` 24.0.2, so
// they are oracle values and not spec citations. java.lang.Integer#toHexString
// is the UNSIGNED reading at the value's own width, which is why -1 is
// "ffffffff" and not "-1".
//
// The last group: a read or write past the end of an array THROWS, with java's
// own message.
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

        // ----- OUT OF BOUNDS THROWS, in all four engines. Every message below is
        // java 24.0.2's own text: `java tests/java-test-full.java` runs it green.
        int[] a = new int[2];
        Main.check("un0", a[0] + 1 == 1);          // the control: in range, still 0

        // A READ past the end, a NEGATIVE index and a far index all throw, and the
        // message names the index and the length.
        Main.check("un1", S34.aioobe(a, 5).equals("Index 5 out of bounds for length 2"));
        Main.check("un2", S34.aioobe(a, -1).equals("Index -1 out of bounds for length 2"));
        Main.check("un3", S34.aioobe(a, 1000).equals("Index 1000 out of bounds for length 2"));
        // A WRITE past the end throws too - it does not GROW the array.
        Main.check("un4", S34.aioobeW(a, 5).equals("Index 5 out of bounds for length 2"));
        Main.check("un5", S34.aioobeW(a, -1).equals("Index -1 out of bounds for length 2"));
        Main.check("un6", a.length == 2);
        // The compound and step forms go through the same place.
        Main.check("un7", S34.aioobeS(a, 5).equals("Index 5 out of bounds for length 2"));
        // A multi-dimensional array checks the OUTER index first.
        int[][] m = new int[2][3];
        Main.check("un8", S34.aioobe2(m, 5, 0).equals("Index 5 out of bounds for length 2"));
        Main.check("un9", S34.aioobe2(m, 0, 5).equals("Index 5 out of bounds for length 3"));

        // ----- String: charAt is an INDEX, substring is a RANGE, and the two
        // messages are different because of it.
        String s = "abc";
        Main.check("un10", S34.sioobe(s, 3).equals("Index 3 out of bounds for length 3"));
        Main.check("un11", S34.sioobe(s, -1).equals("Index -1 out of bounds for length 3"));
        Main.check("un12", S34.srange(s, 0, 100).equals("Range [0, 100) out of bounds for length 3"));
        Main.check("un13", S34.srange(s, -1, 3).equals("Range [-1, 3) out of bounds for length 3"));
        Main.check("un14", S34.srange1(s, -1).equals("Range [-1, 3) out of bounds for length 3"));
        // begin > end is out of bounds even though both are inside the string.
        Main.check("un15", S34.srange(s, 2, 0).equals("Range [2, 0) out of bounds for length 3"));
        Main.check("un16", s.substring(1, 3).equals("bc"));   // the control

        // ----- new T[n] with a negative length: NegativeArraySizeException, whose
        // message is the offending size and nothing else. EVERY dimension is
        // checked before anything is allocated, so [0][-1] throws as well.
        Main.check("un17", S34.nase(-1).equals("-1"));
        Main.check("un18", S34.nase2(0, -3).equals("-3"));

        // ----- the TYPE of the throw is observable, and the clauses are tried in
        // source order.
        Main.check("un19", S34.which(1).equals("AIOOBE"));
        Main.check("un20", S34.which(2).equals("SIOOBE"));
        Main.check("un21", S34.which(3).equals("NASE"));
        Main.check("un22", S34.which(4).equals("ARITH"));
        // Every one of them is an IndexOutOfBoundsException / RuntimeException /
        // Exception / Throwable as well, and a multi-catch clause takes either.
        Main.check("un23", S34.wide(1).equals("IOOBE"));
        Main.check("un24", S34.wide(2).equals("IOOBE"));
        Main.check("un25", S34.wide(3).equals("RTE"));
        Main.check("un26", S34.multi(1).equals("multi"));
        Main.check("un27", S34.multi(3).equals("multi"));
        // An unmatched clause RE-THROWS, so an enclosing try still sees it.
        Main.check("un28", S34.nested().equals("outer:Index 5 out of bounds for length 2"));
        // finally runs whether the clause matched or not.
        Main.check("un29", S34.fin().equals("f1f2"));
    }

    // Each helper returns the MESSAGE of the exception it provoked, so the
    // assertion above is one string comparison against java's own text.
    static String aioobe(int[] a, int i) {
        try { return "" + a[i]; } catch (ArrayIndexOutOfBoundsException e) { return e.getMessage(); }
    }
    static String aioobeW(int[] a, int i) {
        try { a[i] = 9; return "no throw"; } catch (ArrayIndexOutOfBoundsException e) { return e.getMessage(); }
    }
    static String aioobeS(int[] a, int i) {
        try { a[i]++; return "no throw"; } catch (ArrayIndexOutOfBoundsException e) { return e.getMessage(); }
    }
    static String aioobe2(int[][] m, int i, int j) {
        try { return "" + m[i][j]; } catch (ArrayIndexOutOfBoundsException e) { return e.getMessage(); }
    }
    static String sioobe(String s, int i) {
        try { return "" + s.charAt(i); } catch (StringIndexOutOfBoundsException e) { return e.getMessage(); }
    }
    static String srange(String s, int b, int e2) {
        try { return s.substring(b, e2); } catch (StringIndexOutOfBoundsException e) { return e.getMessage(); }
    }
    static String srange1(String s, int b) {
        try { return s.substring(b); } catch (StringIndexOutOfBoundsException e) { return e.getMessage(); }
    }
    static String nase(int n) {
        try { int[] d = new int[n]; return "len " + d.length; }
        catch (NegativeArraySizeException e) { return e.getMessage(); }
    }
    static String nase2(int n, int k) {
        try { int[][] d = new int[n][k]; return "len " + d.length; }
        catch (NegativeArraySizeException e) { return e.getMessage(); }
    }
    // The clause the throw lands in, with three clauses that do NOT match it in
    // front of the one that does.
    static String which(int k) {
        int[] a = new int[2];
        String s = "abc";
        try {
            if (k == 1) { int x = a[5]; }
            if (k == 2) { char c = s.charAt(9); }
            if (k == 3) { int[] d = new int[-2]; }
            if (k == 4) { int z = a.length - 2; int q = 1 / z; }
            return "none";
        } catch (NumberFormatException e) { return "NFE";
        } catch (ArrayIndexOutOfBoundsException e) { return "AIOOBE";
        } catch (StringIndexOutOfBoundsException e) { return "SIOOBE";
        } catch (NegativeArraySizeException e) { return "NASE";
        } catch (ArithmeticException e) { return "ARITH"; }
    }
    static String wide(int k) {
        int[] a = new int[2];
        String s = "abc";
        try {
            if (k == 1) { int x = a[5]; }
            if (k == 2) { char c = s.charAt(9); }
            if (k == 3) { int[] d = new int[-2]; }
            return "none";
        } catch (IndexOutOfBoundsException e) { return "IOOBE";
        } catch (RuntimeException e) { return "RTE"; }
    }
    static String multi(int k) {
        int[] a = new int[2];
        try {
            if (k == 1) { int x = a[5]; }
            if (k == 3) { int[] d = new int[-2]; }
            return "none";
        } catch (NegativeArraySizeException | ArrayIndexOutOfBoundsException e) { return "multi"; }
    }
    static String nested() {
        int[] a = new int[2];
        try {
            try { int x = a[5]; return "none"; }
            catch (NumberFormatException e) { return "inner"; }
        } catch (ArrayIndexOutOfBoundsException e) { return "outer:" + e.getMessage(); }
    }
    static String fin() {
        int[] a = new int[2];
        String r = "";
        try {
            try { int x = a[5]; }
            catch (NumberFormatException e) { r = r + "c"; }
            finally { r = r + "f1"; }
        } catch (ArrayIndexOutOfBoundsException e) { }
        try { int y = a[5]; }
        catch (ArrayIndexOutOfBoundsException e) { }
        finally { r = r + "f2"; }
        return r;
    }
}

// ===== SECTION 35: Integer.parseInt / Long.parseLong and NumberFormatException =====
//
// java.lang.Integer#parseInt accepts an optional sign and a digit run in the
// radix and NOTHING else: no surrounding whitespace, no underscore, no 0x, and no
// magnitude past the width's range. Every other input throws a catchable
// NumberFormatException, which is an IllegalArgumentException. Every message is
// java 24.0.2's own text.
class S35 {
    static String pi(String s) {
        try { return "" + Integer.parseInt(s); } catch (NumberFormatException e) { return e.getMessage(); }
    }
    static String pl(String s) {
        try { return "" + Long.parseLong(s); } catch (NumberFormatException e) { return e.getMessage(); }
    }
    static String pr(String s, int r) {
        try { return "" + Integer.parseInt(s, r); } catch (NumberFormatException e) { return e.getMessage(); }
    }
    static String kind(String s) {
        try { Integer.parseInt(s); return "none"; }
        catch (IllegalArgumentException e) { return (e instanceof NumberFormatException) ? "NFE" : "IAE"; }
    }
    static void run() {
        String[] s = { "12", "x", " 5", "", "-", "2147483647", "2147483648", "-2147483648",
                       "-2147483649", "9223372036854775807", "9223372036854775808",
                       "-9223372036854775808", "18446744073709551616", "+7", "1_0" };
        Main.check("pa1", S35.pi(s[0]).equals("12"));
        Main.check("pa2", S35.pi(s[1]).equals("For input string: \"x\""));
        Main.check("pa3", S35.pi(s[2]).equals("For input string: \" 5\""));
        Main.check("pa4", S35.pi(s[3]).equals("For input string: \"\""));
        Main.check("pa5", S35.pi(s[4]).equals("For input string: \"-\""));
        Main.check("pa6", S35.pi(s[5]).equals("2147483647"));
        Main.check("pa7", S35.pi(s[6]).equals("For input string: \"2147483648\""));
        Main.check("pa8", S35.pi(s[7]).equals("-2147483648"));
        Main.check("pa9", S35.pi(s[8]).equals("For input string: \"-2147483649\""));
        Main.check("pa10", S35.pl(s[6]).equals("2147483648"));
        Main.check("pa11", S35.pl(s[9]).equals("9223372036854775807"));
        Main.check("pa12", S35.pl(s[10]).equals("For input string: \"9223372036854775808\""));
        Main.check("pa13", S35.pl(s[11]).equals("-9223372036854775808"));
        Main.check("pa14", S35.pl(s[12]).equals("For input string: \"18446744073709551616\""));
        Main.check("pa15", S35.pi(s[13]).equals("7") && S35.pi(s[14]).equals("For input string: \"1_0\""));
        Main.check("pa16", S35.pr("ff", 16).equals("255") && S35.pr("-z", 36).equals("-35"));
        Main.check("pa17", S35.pr("g", 16).equals("For input string: \"g\" under radix 16"));
        Main.check("pa18", S35.pr("80000000", 16).equals("For input string: \"80000000\" under radix 16"));
        Main.check("pa19", S35.kind(s[1]).equals("NFE") && S35.kind(s[0]).equals("none"));
        Main.check("pa20", Integer.valueOf(s[0]) + Long.valueOf(s[9]) == -9223372036854775797L);
    }
}

// ===== SECTION 36: field and array defaults, Double / Float / Math members =====
//
// A field or array element nobody assigned holds its type's zero (JLS 4.12.5):
// 0.0, 0L, (byte) 0, '\u0000' - of the type's own kind and width, so a long
// default multiplies at 64 bits and a byte default's compound assignment wraps.
// The Double / Float constants and the java.lang.Math methods are checked
// against java 24.0.2's own output; every operand is read out of an array.
class S36 {
    static double fd; static long fl; static float ff; static byte fb; static char fc;
    double id; long il; short is;
    static String nfe(String s) {
        try { return "" + Double.parseDouble(s); } catch (NumberFormatException e) { return e.getMessage(); }
    }
    static void run() {
        Main.check("df1", ("" + S36.fd).equals("0.0") && ("" + S36.ff).equals("0.0"));
        Main.check("df2", ("" + S36.fl).equals("0") && ("" + S36.fb).equals("0") && S36.fc == 0);
        Main.check("df3", S36.fl * 1000000 * 1000000 * 1000 == 0 && (S36.fl + 1) * 1000000 * 1000000 * 1000 == 1000000000000000L);
        S36 o = new S36();
        Main.check("df4", ("" + o.id).equals("0.0") && ("" + o.il).equals("0") && ("" + o.is).equals("0"));
        o.il += 2;
        Main.check("df5", o.il * 1000000 * 1000000 * 1000 == 2000000000000000L);
        double[] da = new double[2];
        long[] la = new long[1];
        byte[] ba = new byte[1];
        char[] ca = new char[1];
        float[][] fm = new float[1][1];
        Main.check("df6", ("" + da[1]).equals("0.0") && ("" + fm[0][0]).equals("0.0") && ca[0] == 0);
        la[0] += 3;
        ba[0] += 200;
        Main.check("df7", la[0] * 1000000 * 1000000 * 1000 == 3000000000000000L && ba[0] == -56);

        Main.check("dc1", ("" + Double.POSITIVE_INFINITY).equals("Infinity") && ("" + Double.NEGATIVE_INFINITY).equals("-Infinity"));
        Main.check("dc2", ("" + Double.MAX_VALUE).equals("1.7976931348623157E308") && ("" + Double.MIN_VALUE).equals("4.9E-324"));
        Main.check("dc3", ("" + Double.MIN_NORMAL).equals("2.2250738585072014E-308") && Double.MAX_VALUE * 2 == Double.POSITIVE_INFINITY);
        Main.check("dc4", ("" + Float.MAX_VALUE).equals("3.4028235E38") && ("" + Float.MIN_VALUE).equals("1.4E-45"));
        Main.check("dc5", ("" + Float.NaN).equals("NaN") && ("" + Float.NEGATIVE_INFINITY).equals("-Infinity") && Float.SIZE == 32);
        double[] d = { 0.0, -0.0, 2.5, -2.5, 0.49999999999999994, 0.0 / 0, 1.0 / 0, 16.0, -1.5 };
        Main.check("dc6", Double.isNaN(d[5]) && !Double.isNaN(d[2]) && Double.isInfinite(d[6]) && Double.isFinite(d[2]) && !Double.isFinite(d[6]));
        Main.check("dc7", Double.compare(d[1], d[0]) == -1 && Double.compare(d[5], d[6]) == 1 && Double.compare(d[5], d[5]) == 0);
        Main.check("dc8", Double.parseDouble(" 1e3 ") == 1000.0 && ("" + Double.parseDouble("-0")).equals("-0.0") && Float.parseFloat("0.1") == 0.1f);
        Main.check("dc9", S36.nfe("").equals("empty String") && S36.nfe("1.5.5").equals("multiple points") && S36.nfe("1x").equals("For input string: \"1x\""));
        Main.check("dc10", ("" + Double.valueOf("2")).equals("2.0") && ("" + Float.valueOf(3)).equals("3.0"));

        Main.check("ma1", ("" + Math.sqrt(d[7])).equals("4.0") && ("" + Math.PI).equals("3.141592653589793"));
        Main.check("ma2", ("" + Math.rint(d[2])).equals("2.0") && ("" + Math.rint(d[3])).equals("-2.0") && ("" + Math.rint(-0.4)).equals("-0.0"));
        Main.check("ma3", Math.round(d[2]) == 3L && Math.round(d[3]) == -2L && Math.round(d[4]) == 0L && Math.round(d[5]) == 0L);
        Main.check("ma4", Math.round(1e20) == Long.MAX_VALUE && Math.round(2.5f) == 3 && Math.round(1e10f) == Integer.MAX_VALUE);
        Main.check("ma5", ("" + Math.floor(d[8])).equals("-2.0") && ("" + Math.ceil(d[8])).equals("-1.0") && ("" + Math.signum(d[3])).equals("-1.0"));
        Main.check("ma6", ("" + Math.pow(2, 10)).equals("1024.0") && ("" + Math.pow(1, d[5])).equals("NaN") && ("" + Math.pow(-1, d[6])).equals("NaN"));
        Main.check("ma7", ("" + Math.toRadians(180)).equals("3.141592653589793") && ("" + Math.toDegrees(1)).equals("57.29577951308232"));
        Main.check("ma8", Math.floorDiv(-7, 2) == -4 && Math.floorMod(-7, 2) == 1 && Math.floorMod(7, -2) == -1 && Math.floorDiv(-7L, 2L) == -4L);
        Main.check("ma9", ("" + Math.cbrt(27)).equals("3.0") && ("" + Math.hypot(3, 4)).equals("5.0") && ("" + Math.log10(1000)).equals("3.0"));

        int[] n = { 255, -255, Integer.MIN_VALUE };
        Main.check("rx1", Integer.toString(n[0], 16).equals("ff") && Integer.toString(n[1], 16).equals("-ff") && Integer.toString(n[0], 99).equals("255"));
        Main.check("rx2", Integer.toBinaryString(n[1]).equals("11111111111111111111111100000001") && Integer.toOctalString(n[2]).equals("20000000000"));
        Main.check("rx3", Long.toString(Long.MIN_VALUE, 36).equals("-1y2p0ij32e8e8") && Long.toBinaryString(5L).equals("101"));

        long[] q = { -255L, 4294967296L, 6L };
        Main.check("bo1", Integer.bitCount(n[1]) == 25 && Integer.numberOfLeadingZeros(n[0]) == 24 && Integer.numberOfTrailingZeros(n[2]) == 31);
        Main.check("bo2", Integer.highestOneBit(n[1]) == Integer.MIN_VALUE && Integer.lowestOneBit(n[0]) == 1 && Integer.signum(n[1]) == -1);
        Main.check("bo3", Long.bitCount(q[0]) == 57 && Long.numberOfLeadingZeros(q[1]) == 31 && Long.numberOfTrailingZeros(q[1]) == 32);
        Main.check("bo4", Long.highestOneBit(q[2]) == 4L && Long.lowestOneBit(q[0]) == 1L && Integer.sum(Integer.MAX_VALUE, n[0]) == -2147483394);
    }
}

// ===== SECTION 37: assignment conversion at every site (JLS 5.2) =====
//
// A value converts to the DECLARED type of the slot it lands in - not only at a
// declaration but when it is passed as an argument, returned, reassigned,
// stored into a field or an array element, or written in an array
// initializer. An Object slot keeps the value's own type. Every expected
// string is java 24.0.2's output.
interface S37Long { long apply(long x); }
class S37 {
    static double fd;
    static long fl;
    static Object fo;
    static double[] fda;
    long il;
    float iflt;
    static String d(double x) { return "" + x; }
    static String f(float x) { return "" + x; }
    static long mul(long x) { return x * 1000000 * 1000000; }
    static String cc(int c) { return "" + c; }
    static String vsum(double... xs) { String s = ""; for (double x : xs) s = s + x + ";"; return s; }
    static double r3() { return 3; }
    static long r2() { return 2; }
    static float rbig() { return 16777217; }
    static char rch() { return 66; }
    static int rint() { return 'z'; }
    static Object robj() { return 3; }
    static long twice(long x) { return x * 2; }
    static long viaRef(S37Long g) { return g.apply(3) * 1000000 * 1000000; }
    static void run() {
        int[] k = { 2, 3, 16777217, 97 };
        Main.check("ac1", d(k[0]).equals("2.0") && f(k[2]).equals("1.6777216E7") && mul(k[0]) == 2000000000000L);
        Main.check("ac2", cc('a').equals("97") && vsum(k[0], k[1]).equals("2.0;3.0;"));
        Main.check("ac3", ("" + r3()).equals("3.0") && r2() * 1000000 * 1000000 * 1000 == 2000000000000000L);
        Main.check("ac4", ("" + rbig()).equals("1.6777216E7") && ("" + rch()).equals("B") && ("" + rint()).equals("122"));
        Main.check("ac5", ("" + robj()).equals("3") && viaRef(S37::twice) == 6000000000000L);
        long t;
        t = k[0];
        Main.check("ac6", t * 1000000 * 1000000 * 1000 == 2000000000000000L);
        double dd;
        dd = k[1];
        char ch;
        ch = 65;
        int ic;
        ic = 'c';
        Main.check("ac7", ("" + dd).equals("3.0") && ("" + ch).equals("A") && ("" + ic).equals("99"));
        Object o = k[0];
        o = 2.5;
        Main.check("ac8", ("" + o).equals("2.5"));
        o = k[1];
        Main.check("ac9", ("" + o).equals("3"));
        fd = k[0];
        fl = k[0];
        fo = k[0];
        Main.check("ac10", ("" + S37.fd).equals("2.0") && S37.fl * 1000000 * 1000000 * 1000 == 2000000000000000L && ("" + S37.fo).equals("2"));
        S37 s = new S37();
        s.il = k[1];
        s.iflt = k[2];
        Main.check("ac11", s.il * 1000000 * 1000000 * 1000 == 3000000000000000L && ("" + s.iflt).equals("1.6777216E7"));
        double[] da = new double[2];
        da[0] = k[0];
        fda = new double[1];
        fda[0] = k[1];
        long[][] lm = new long[1][1];
        lm[0][0] = k[0];
        Main.check("ac12", ("" + da[0]).equals("2.0") && ("" + fda[0]).equals("3.0") && lm[0][0] * 1000000 * 1000000 * 1000 == 2000000000000000L);
        float[] fa = { 16777217 };
        double[][] dm = { { 1, 2 }, { 3 } };
        long[] la = new long[] { 3 };
        int ca[] = { 'a', 1 };
        Main.check("ac13", ("" + fa[0]).equals("1.6777216E7") && ("" + dm[1][0]).equals("3.0") && ("" + ca[0]).equals("97"));
        Main.check("ac14", la[0] * 1000000 * 1000000 * 1000 == 3000000000000000L);
        long acc = 0;
        for (int i = 0; i < 4; i++) { acc = acc * 1000000 + i; }
        Main.check("ac15", acc == 1000002000003L);
        String seen = "";
        for (double x : k) { seen = seen + x + ";"; }
        long lsum = 0;
        for (long x : k) { lsum = lsum + x * 1000000 * 1000; }
        Main.check("ac16", seen.equals("2.0;3.0;1.6777217E7;97.0;") && lsum == 16777319000000000L);
    }
}

// ===== SECTION 38: the shortest digits at a power of two =====
//
// Double.toString and Float.toString print the SHORTEST decimal that reads back
// as the same value. Just above a power of two the rounding interval is twice as
// wide as just below it, so the nearest n-digit decimal can miss while its
// neighbour on the far side hits: 2^-24 is exactly 5.9604644775390625E-8 and
// prints with 16 digits. Every expected string is java 24.0.2's output.
class S38 {
    static void run() {
        double[] d = { 16777216.0, 2.0, 1.0 };
        float[] f = { 2.0f, 1.0f };
        double x = d[2];
        for (int i = 0; i < 24; i++) x = x / d[1];
        Main.check("sd1", ("" + d[2] / d[0]).equals("5.960464477539063E-8") && ("" + x).equals("5.960464477539063E-8"));
        for (int i = 24; i < 44; i++) x = x / d[1];
        Main.check("sd2", ("" + x).equals("5.684341886080802E-14"));
        for (int i = 44; i < 140; i++) x = x / d[1];
        Main.check("sd3", ("" + x).equals("7.174648137343064E-43"));
        float g = f[1];
        for (int i = 0; i < 96; i++) g = g / f[0];
        Main.check("sd4", ("" + g).equals("1.2621775E-29"));
        g = f[1];
        for (int i = 0; i < 87; i++) g = g * f[0];
        Main.check("sd5", ("" + g).equals("1.5474251E26") && ("" + g * f[0] * f[0] * f[0]).equals("1.2379401E27"));
    }
}

// ===== SECTION 39: overload resolution by argument type (JLS 15.12.2) =====
//
// Among overloads of one arity the applicable ones are found in three phases
// (no boxing, boxing, varargs) and the most specific of the first phase that
// has any is called. An inherited overload stays a candidate next to the
// subclass' own, and super.m(...) stays non-virtual. Every expected string is
// java 24.0.2's output.
class S39Animal { String who() { return "animal"; } }
class S39Dog extends S39Animal { String who() { return "dog"; } }
class S39Base {
    String f(int x) { return "B.int " + x; }
    String f(String s) { return "B.str " + s; }
}
class S39Derived extends S39Base {
    String f(int x) { return "D.int " + x + "/" + super.f(x); }
}
class S39P {
    String tag;
    S39P(int a, int b) { this("ii " + (a + b)); }
    S39P(double a, double b) { this("dd " + (a + b)); }
    S39P(String s) { tag = s; }
}
class S39 {
    static String pick(int a, int b) { return "int " + (a + b); }
    static String pick(double a, double b) { return "double " + (a + b); }
    static String show(long x) { return "long " + x; }
    static String show(String s) { return "str " + s; }
    static String box(Integer i) { return "Integer " + i; }
    static String box(Object o) { return "Object " + o; }
    static String ch(char c) { return "char " + c; }
    static String ch(int i) { return "int " + i; }
    static String fl(float f) { return "float " + f; }
    static String fl(double d) { return "double " + d; }
    static String an(S39Animal a) { return "Animal " + a.who(); }
    static String an(S39Dog d) { return "Dog " + d.who(); }
    static String va(int... xs) { return "varargs " + xs.length; }
    static String va(int a) { return "single " + a; }
    static String arr(int[] xs) { return "int[] " + xs.length; }
    static String arr(Object o) { return "Object"; }
    static String nul(String s) { return "String " + s; }
    static String nul(Object o) { return "Object " + o; }
    static void run() {
        int[] k = { 1, 2, 3 };
        long[] l = { 5L };
        double[] d = { 1.5 };
        char[] c = { 'a' };
        float[] f = { 2.5f };
        String[] s = { "x" };
        Main.check("ov1", pick(k[0], k[1]).equals("int 3") && pick(d[0], k[1]).equals("double 3.5"));
        Main.check("ov2", show(k[0]).equals("long 1") && show(l[0]).equals("long 5") && show(s[0]).equals("str x"));
        Main.check("ov3", box(k[2]).equals("Integer 3") && box(s[0]).equals("Object x") && box(d[0]).equals("Object 1.5"));
        Main.check("ov4", ch(c[0]).equals("char a") && ch(k[0]).equals("int 1"));
        Main.check("ov5", fl(f[0]).equals("float 2.5") && fl(l[0]).equals("float 5.0") && fl(d[0]).equals("double 1.5"));
        Main.check("ov6", an(new S39Dog()).equals("Dog dog") && an(new S39Animal()).equals("Animal animal"));
        Main.check("ov7", va().equals("varargs 0") && va(k[0]).equals("single 1") && va(k[0], k[1]).equals("varargs 2") && va(k).equals("varargs 3"));
        Main.check("ov8", arr(k).equals("int[] 3") && arr(s).equals("Object") && nul(null).equals("String null") && nul(k[0]).equals("Object 1"));
        S39Base b = new S39Derived();
        Main.check("ov9", b.f(k[0]).equals("D.int 1/B.int 1") && b.f(s[0]).equals("B.str x"));
        Main.check("ov10", new S39P(k[0], k[1]).tag.equals("ii 3") && new S39P(d[0], k[1]).tag.equals("dd 3.5"));
    }
}

// ===== SECTION 40: String.toCharArray, Runnable, Character, Math.*Exact, Double.sum =====
//
// A handful of java.lang members, each checked against java 24.0.2: an
// anonymous class over Runnable (a library interface the program does not
// declare) reaching a static field of the enclosing class; Character's
// classification and case mapping for Latin letters, digits and blanks; the
// overflow-checked Math.*Exact family (ArithmeticException "integer overflow"
// / "long overflow"); Double.sum / max / min, whose max and min order -0.0
// below 0.0.
class S40 {
    static int runs = 0;
    static String exact(int which, int[] k, long[] l) {
        try {
            if (which == 0) { return "" + Math.addExact(k[0], k[1]); }
            if (which == 1) { return "" + Math.subtractExact(k[2], k[1]); }
            if (which == 2) { return "" + Math.multiplyExact(k[3], k[3]); }
            if (which == 3) { return "" + Math.negateExact(k[2]); }
            if (which == 4) { return "" + Math.addExact(l[0], l[1]); }
            if (which == 5) { return "" + Math.multiplyExact(l[0], 2L); }
            return "" + Math.toIntExact(l[0]);
        } catch (ArithmeticException e) {
            return e.getMessage();
        }
    }
    static void run() {
        String[] s = { "hello", "" };
        char[] cs = s[0].toCharArray();
        Main.check("tc1", cs.length == 5 && cs[1] == 'e' && cs[0] + 1 == 105 && s[1].toCharArray().length == 0);
        Runnable r = new Runnable() { public void run() { runs++; } };
        r.run();
        r.run();
        Main.check("rn1", S40.runs == 2);
        char[] c = { '7', 'a', 'Z', ' ', '_', 'é', 'ß', '²' };
        Main.check("ch1", Character.isDigit(c[0]) && !Character.isDigit(c[1]) && Character.isLetter(c[1]) && !Character.isLetter(c[0]));
        Main.check("ch2", Character.isLetter(c[5]) && !Character.isLetter(c[4]) && Character.isLetterOrDigit(c[0]) && !Character.isLetterOrDigit(c[3]));
        Main.check("ch3", Character.toUpperCase(c[1]) == 'A' && Character.toUpperCase(c[5]) == 'É' && Character.toUpperCase(c[6]) == 'ß' && Character.toLowerCase(c[2]) == 'z');
        Main.check("ch4", Character.getNumericValue(c[0]) == 7 && Character.getNumericValue(c[1]) == 10 && Character.getNumericValue(c[2]) == 35
            && Character.getNumericValue(c[4]) == -1 && Character.getNumericValue(c[7]) == 2);
        Main.check("ch5", Character.isWhitespace(c[3]) && !Character.isWhitespace(c[4]) && Character.isUpperCase(c[2]) && Character.isLowerCase(c[6]));
        Main.check("ch6", ("" + Character.toUpperCase(c[1])).equals("A") && Character.toUpperCase(97) == 65);
        // Pairs added in Unicode 16: the case table must not be older than that.
        int[] u16 = { 0x019B, 0x0264, 0x1C89, 0x1C8A };
        Main.check("ch7", Character.toUpperCase((char) u16[0]) == 0xA7DC && Character.toUpperCase((char) u16[1]) == 0xA7CB
            && Character.toLowerCase((char) u16[2]) == 0x1C8A && Character.toUpperCase((char) u16[3]) == 0x1C89);
        int[] k = { Integer.MAX_VALUE, 1, Integer.MIN_VALUE, 46341 };
        long[] l = { Long.MAX_VALUE, 1L };
        Main.check("ex1", Math.addExact(k[1], k[1]) == 2 && Math.multiplyExact(k[3], k[1]) == 46341 && Math.toIntExact(l[1]) == 1);
        Main.check("ex2", exact(0, k, l).equals("integer overflow") && exact(1, k, l).equals("integer overflow") && exact(2, k, l).equals("integer overflow"));
        Main.check("ex3", exact(3, k, l).equals("integer overflow") && exact(4, k, l).equals("long overflow") && exact(5, k, l).equals("long overflow"));
        Main.check("ex4", exact(6, k, l).equals("integer overflow") && Math.incrementExact(l[1]) * 1000000 * 1000000 * 1000 == 2000000000000000L);
        double[] d = { 1.5, 2.25, -0.0, 0.0 };
        Main.check("ds1", ("" + Double.sum(d[0], d[1])).equals("3.75") && ("" + Double.max(d[0], d[1])).equals("2.25") && ("" + Double.min(d[0], d[1])).equals("1.5"));
        Main.check("ds2", ("" + Double.min(d[2], d[3])).equals("-0.0") && ("" + Double.max(d[2], d[3])).equals("0.0"));
    }
}

// ===== SECTION 41: a lambda takes its functional interface's primitive types =====
//
// A lambda stored as a functional interface sees its parameters as that
// interface's parameter types and answers its result type (JLS 15.27.3):
// DoubleUnaryOperator's x -> x + 1 applied to 2 is 3.0, IntToDoubleFunction's
// i -> i / 2 applied to 3 is 1.0. A declared one-method interface works the
// same, wherever it is declared, in a local, a field, a parameter or a return.
// The java.util.function types are spelled qualified: this file imports
// nothing. Every expected string is java 24.0.2's output.
class S41 {
    static S41Fn field;
    static double use(S41Fn f, int v) { return f.go(v); }
    static java.util.function.IntToDoubleFunction maker() { return i -> i * 3; }
    static void run() {
        int[] k = { 2, 3, 1000000 };
        java.util.function.DoubleUnaryOperator op = x -> x + 1;
        java.util.function.DoubleUnaryOperator one = x -> 1;
        Main.check("sm1", ("" + op.applyAsDouble(k[0])).equals("3.0") && ("" + one.applyAsDouble(k[0])).equals("1.0"));
        java.util.function.DoubleBinaryOperator div = (a, b) -> a / b;
        java.util.function.IntToDoubleFunction half = i -> i / 2;
        Main.check("sm2", ("" + div.applyAsDouble(k[0], k[1])).equals("0.6666666666666666") && ("" + half.applyAsDouble(k[1])).equals("1.0"));
        java.util.function.LongUnaryOperator sq = x -> x * x;
        java.util.function.IntUnaryOperator inc = x -> x + 1;
        Main.check("sm3", sq.applyAsLong(k[2]) == 1000000000000L && ("" + inc.applyAsInt(k[0])).equals("3"));
        S41Fn f = x -> x / 4;
        field = x -> x + 0.5;
        Main.check("sm4", ("" + f.go(k[1])).equals("0.75") && ("" + use(x -> x * 2, k[1])).equals("6.0") && ("" + field.go(k[0])).equals("2.5"));
        Main.check("sm5", ("" + maker().applyAsDouble(k[0])).equals("6.0") && ("" + f.twice(k[1])).equals("1.5"));
    }
}
interface S41Fn {
    double go(double x);
    default double twice(double x) { return go(x) * 2; }
}

// ===== SECTION 42: Math.exp is StrictMath.exp (fdlibm) =====
//
// java.lang.Math#exp answers what fdlibm's e_exp.c answers, which is one ulp
// away from a libm exp at e = exp(1), and reaches into the subnormals down to
// exp(-745). Every expected string is java 24.0.2's output.
class S42 {
    static void run() {
        double[] x = { 1.0, -745.0, -33.58169999999959, 0.5, 709.0, -0.0 };
        Main.check("xp1", ("" + Math.exp(x[0])).equals("2.7182818284590455") && ("" + Math.exp(x[1])).equals("4.9E-324"));
        Main.check("xp2", ("" + Math.exp(x[2])).equals("2.6040720551475586E-15") && ("" + Math.exp(x[3])).equals("1.6487212707001282"));
        Main.check("xp3", ("" + Math.exp(x[4])).equals("8.218407461554972E307") && ("" + Math.exp(x[5])).equals("1.0"));
    }
}

// ===== SECTION 43: Double.parseDouble / Float.parseFloat of a hex significand =====
//
// The text may be a HexSignificand with a mandatory binary exponent (JLS
// 3.10.2): rounded ONCE, half to even, to the target width, into the
// subnormals and up to Infinity. Every expected string is java 24.0.2's output.
class S43 {
    static String pd(String s) {
        try { return "" + Double.parseDouble(s); } catch (NumberFormatException e) { return e.getMessage(); }
    }
    static void run() {
        String[] t = { "0x1p3", "-0x1.8p1", " 0x.8p1d ", "0x1.fffffffffffff8p1023", "0x1.0000000000001p-1075",
                       "0x1.0000000000000801p0", "0x1.0000010000000001p0", "0x1p", "-0x0p0" };
        Main.check("hp1", pd(t[0]).equals("8.0") && pd(t[1]).equals("-3.0") && pd(t[2]).equals("1.0") && pd(t[8]).equals("-0.0"));
        Main.check("hp2", pd(t[3]).equals("Infinity") && pd(t[4]).equals("4.9E-324") && pd(t[5]).equals("1.0000000000000002"));
        Main.check("hp3", ("" + Float.parseFloat(t[6])).equals("1.0000001") && ("" + Double.valueOf(t[6])).equals("1.0000000596046448"));
        Main.check("hp4", pd(t[7]).equals("For input string: \"0x1p\""));
    }
}

// ===== SECTION 44: overloads resolve by STATIC type (JLS 15.12.2) =====
//
// javac chooses the overload from the arguments' static types: an Object
// variable holding a String still calls f(Object), an Integer takes the
// boxed overload ahead of a widening, and a cast or a conditional decides
// the type. An enum's and an anonymous class' own overloads resolve the same
// way, and so does a call through an interface. Every expected string is java
// 24.0.2's output.
class S44A { String who() { return "A"; } }
class S44B extends S44A { String who() { return "B"; } }
interface S44I { String q(Object o); String q(String s); }
enum S44E {
    X;
    String p(Object o) { return "pO"; }
    String p(String s) { return "pS"; }
    String p(int i) { return "pI"; }
}
class S44 {
    static String f(Object o) { return "O"; }
    static String f(String s) { return "S"; }
    static String g(long x) { return "long"; }
    static String g(Integer x) { return "Integer"; }
    static String h(Object o) { return "hO"; }
    static String h(int i) { return "hI"; }
    static String k(S44A a) { return "kA" + a.who(); }
    static String k(S44B b) { return "kB" + b.who(); }
    static String t(Runnable r) { return "tR"; }
    static String t(Object o) { return "tO"; }
    String m(Object o) { return "mO"; }
    String m(String s) { return "mS"; }
    static Object mk() { return "made"; }
    static void run() {
        Object o = "s";
        String s = "s";
        Integer boxed = 3;
        int prim = 3;
        S44A ab = new S44B();
        S44B bb = new S44B();
        Main.check("sov1", (f(o) + f(s) + f("l") + f((Object) "c") + f(null) + f(mk())).equals("OSSOSO"));
        Main.check("sov2", (g(prim) + g(boxed) + h(boxed) + h(prim) + h(Integer.valueOf(5))).equals("longIntegerhOhIhO"));
        Main.check("sov3", (k(ab) + k(bb) + k(new S44B()) + k((S44A) bb)).equals("kABkBBkBBkAB"));
        Main.check("sov4", (new S44().m(o) + new S44().m(s) + f(o == null ? s : o) + f(s + 1)).equals("mOmSOS"));
        Main.check("sov5", (S44E.X.p(o) + S44E.X.p(s) + S44E.X.p(7)).equals("pOpSpI"));
        S44I anon = new S44I() {
            public String q(Object x) { return "qO"; }
            public String q(String x) { return "qS"; }
        };
        Runnable rn = () -> { };
        Main.check("sov6", (anon.q(o) + anon.q(s) + t(rn) + t(() -> { }) + t("z")).equals("qOqStRtRtO"));
        Main.check("sov7", (new S44P(o).tag + new S44P(s).tag + new S44P(boxed).tag + new S44P(prim).tag
            + new S44Q(s).tag + new S44P((Object) "x").tag).equals("OSIntegerlongOO"));
    }
}
class S44P {
    String tag;
    S44P(Object o) { tag = "O"; }
    S44P(String s) { tag = "S"; }
    S44P(long x) { tag = "long"; }
    S44P(Integer x) { tag = "Integer"; }
}
class S44Q extends S44P {
    S44Q(Object o) { super(o); }
}

// ===== SECTION 45: the transcendentals are StrictMath's, fdlibm =====
//
// java.lang.StrictMath is fdlibm 5.3, and java.lang.Math answers what it
// answers for every function here except sin and cos, which a JIT may
// replace with an intrinsic (so those two are checked through StrictMath).
// Each value is one where a libm differs from fdlibm in the last place.
// Every expected string is java 24.0.2's output.
class S45 {
    static String s(double d) { return "" + d; }
    static void run() {
        double[] x = { 0.5, 2.6902782457437238, 0.1, 0.9999999999999999, 2.0, 0.7, 0.9, 0.6, 0.3, 10.0,
                       1.0, 1.0E-8, 4.9E-324, 2.8783165136312148, -7.217476819570581, 2.7873122074460528E302,
                       0.36327291121493643, 0.9886221146689158, 1.5725219022010501, 1.0E300 };
        Main.check("fd1", s(Math.cbrt(x[0])).equals("0.7937005259840998") && s(Math.log(x[1])).equals("0.989644625343953")
            && s(Math.log10(x[2])).equals("-1.0") && s(Math.log10(x[12])).equals("-323.3062153431158"));
        Main.check("fd2", s(Math.expm1(x[3])).equals("1.7182818284590449") && s(Math.log1p(x[3])).equals("0.6931471805599453")
            && s(Math.sinh(x[11])).equals("1.0000000000000002E-8") && s(Math.cosh(x[10])).equals("1.543080634815244")
            && s(Math.tanh(x[5])).equals("0.6043677771171636"));
        Main.check("fd3", s(StrictMath.sin(x[4])).equals("0.9092974268256817") && s(StrictMath.cos(x[5])).equals("0.7648421872844885")
            && s(Math.tan(x[6])).equals("1.2601582175503392") && s(StrictMath.sin(x[19])).equals("-0.8178819121159085"));
        Main.check("fd4", s(Math.asin(x[7])).equals("0.6435011087932844") && s(Math.acos(x[8])).equals("1.2661036727794992")
            && s(Math.atan(x[9])).equals("1.4711276743037347") && s(Math.atan2(x[13], x[14])).equals("2.7621227700656026"));
        Main.check("fd5", s(Math.pow(x[15], x[16])).equals("7.415544782946163E109") && s(Math.hypot(x[17], x[18])).equals("1.8574710814746076")
            && s(StrictMath.pow(x[14], x[4])).equals("52.09197164103867") && s(StrictMath.exp(x[10])).equals("2.7182818284590455"));
    }
}

// ===== SECTION 46: new String(...), ArrayList / List, and for over an Iterable =====
//
// The String constructors copy (and range-check) a char[] or code points; the
// java.util.ArrayList / List of an import are java's own, error messages
// included; remove(Object) and remove(int) are told apart by the argument's
// static type; an enhanced for over anything but an array runs iterator(),
// hasNext() and next() in step with the body. Every expected string is java
// 24.0.2's output.
class S46Range implements Iterable<Integer> {
    int lo;
    int hi;
    String log = "";
    S46Range(int lo, int hi) { this.lo = lo; this.hi = hi; }
    public Iterator<Integer> iterator() {
        return new Iterator<Integer>() {
            int cur = lo;
            public boolean hasNext() { return cur < hi; }
            public Integer next() { log = log + "n" + cur; cur++; return cur - 1; }
        };
    }
}
class S46 {
    static void run() {
        char[] cs = { 'h', 'i', '!' };
        int[] cps = { 72, 0x1F600, 105 };
        String sio = "";
        try { sio = new String(cs, 1, 5); } catch (StringIndexOutOfBoundsException e) { sio = e.getMessage(); }
        Main.check("ns1", new String("ab").equals("ab") && new String().isEmpty() && new String(cs).equals("hi!")
            && new String(cs, 1, 2).equals("i!") && new String(cps, 0, 3).length() == 4);
        Main.check("ns2", sio.equals("Range [1, 1 + 5) out of bounds for length 3"));
        List<Integer> xs = new ArrayList<>();
        for (int i = 0; i < 6; i++) { xs.add(i * i); }
        xs.remove(Integer.valueOf(9));
        xs.remove(0);
        Integer four = 4;
        xs.remove(four);
        Main.check("al1", ("" + xs).equals("[1, 16, 25]") && xs.size() == 3 && xs.get(1) == 16 && xs.indexOf(25) == 2
            && xs.contains(1) && !xs.contains(4));
        int sum = 0;
        for (int x : xs) { sum += x; }
        ArrayList<String> ss = new ArrayList<>(List.of("b", "a"));
        ss.add(1, "z");
        String old = ss.set(0, "q");
        Main.check("al2", sum == 42 && ("" + ss).equals("[q, z, a]") && old.equals("b") && ss.equals(List.of("q", "z", "a"))
            && List.of(1, 2, 3).hashCode() == 30817);
        String err = "";
        try { xs.get(5); } catch (IndexOutOfBoundsException e) { err = e.getMessage(); }
        String err2 = "";
        try { List.of(1).add(2); } catch (UnsupportedOperationException e) { err2 = "" + e; }
        Main.check("al3", err.equals("Index 5 out of bounds for length 3") && err2.equals("java.lang.UnsupportedOperationException"));
        S46Range r = new S46Range(1, 4);
        String seen = "";
        for (int v : r) {
            seen = seen + v;
            if (v == 2) { break; }
        }
        Main.check("it1", seen.equals("12") && r.log.equals("n1n2"));
    }
}

// ===== SECTION 47: Character classification past Latin (Unicode 16.0) =====
//
// A letter of an uncased script is a letter, any script's decimal digits are
// digits with their values, a letter number is alphabetic and may be upper
// case without being a letter, a combining mark can be alphabetic and lower
// case, and getNumericValue reads Unicode's numeric values. Every expected
// value is java 24.0.2's.
class S47 {
    static void run() {
        int[] cp = { 0x0905, 0x0966, 0x0669, 0x2160, 0x216B, 0x0345, 0x02B0, 0xFF21, 0x1D7CE, 0x4E00, 0x00BD, 0x2468, 0x1C89 };
        char[] ch = { 'अ', '०', 'Ⅰ', 'ͅ' };
        Main.check("uc1", Character.isLetter(cp[0]) && Character.isLetter(ch[0]) && !Character.isUpperCase(cp[0])
            && Character.isLetterOrDigit(cp[0]) && Character.isLetter(cp[9]) && Character.isAlphabetic(cp[9]));
        Main.check("uc2", Character.isDigit(cp[1]) && Character.isDigit(ch[1]) && Character.digit(cp[2], 10) == 9
            && Character.getNumericValue(cp[1]) == 0 && Character.isDigit(cp[8]) && Character.digit(cp[8], 10) == 0);
        Main.check("uc3", !Character.isLetter(cp[3]) && Character.isAlphabetic(ch[2]) && Character.isUpperCase(cp[3])
            && Character.getNumericValue(cp[3]) == 1 && Character.getNumericValue(cp[4]) == 12);
        Main.check("uc4", Character.isAlphabetic(cp[5]) && Character.isLowerCase(ch[3]) && !Character.isLetter(cp[5])
            && Character.isLowerCase(cp[6]) && Character.isLetter(cp[6]));
        Main.check("uc5", Character.digit(cp[7], 36) == 10 && Character.getNumericValue(cp[9]) == -1
            && Character.getNumericValue(cp[10]) == -2 && Character.getNumericValue(cp[11]) == 9
            && Character.isUpperCase(cp[12]) && Character.isLetter(cp[12]));
    }
}

// ===== SECTION 48: instanceof an interface, and a user List =====
//
// `instanceof` holds for every interface a class implements, for each one those
// extend, and through a superclass; a class extending java.util.AbstractList is
// a List an ArrayList compares equal to. Every expected value is java 24.0.2's.
interface S48Col { int n(); }
interface S48Lst extends S48Col { }
interface S48Gen<T> { T first(); }
interface S48A { default String a() { return "a"; } }
interface S48B { default String b() { return "b"; } }
interface S48AB extends S48A, S48B { }
class S48Impl implements S48Lst, S48Gen<String> {
    public int n() { return 1; }
    public String first() { return "x"; }
}
class S48Base { }
class S48K extends S48Base implements S48AB { }
class S48Range extends AbstractList<Integer> {
    int size;
    S48Range(int size) { this.size = size; }
    public Integer get(int i) {
        if (i < 0 || i >= size) { throw new IndexOutOfBoundsException("at " + i); }
        return i * 10;
    }
    public int size() { return size; }
}
class S48 {
    static void run() {
        Object o = new S48Impl();
        Main.check("if1", o instanceof S48Lst && o instanceof S48Col && o instanceof S48Gen && o instanceof S48Impl);
        Object k = new S48K();
        S48K kk = new S48K();
        Main.check("if2", k instanceof S48A && k instanceof S48B && k instanceof S48AB && k instanceof S48Base
            && (kk.a() + kk.b()).equals("ab") && !(o instanceof S48A));
        S48Range r = new S48Range(3);
        Object ro = r;
        ArrayList<Integer> a = new ArrayList<>();
        a.add(0); a.add(10); a.add(20);
        Main.check("if3", ro instanceof List && a.equals(r) && r.equals(a) && r.equals(List.of(0, 10, 20))
            && a.hashCode() == r.hashCode() && ("" + r).equals("[0, 10, 20]") && r.indexOf(20) == 2 && !r.equals("x"));
        a.add(30);
        String nse = "";
        Iterator<Integer> it = r.iterator();
        it.next(); it.next(); it.next();
        try { it.next(); } catch (RuntimeException e) { nse = e.getMessage(); }
        String uoe = "";
        try { r.add(5); } catch (UnsupportedOperationException e) { uoe = "" + e; }
        Main.check("if4", !a.equals(r) && !r.equals(a) && nse.equals("java.lang.IndexOutOfBoundsException: at 3")
            && uoe.equals("java.lang.UnsupportedOperationException"));
    }
}

// ===== SECTION 49: java.lang.String's API, String.format, getClass, StringBuilder =====
//
// The String methods past charAt/substring (the regex ones are
// java.util.regex's: split's trailing empties and limit, $n and ${name}
// in a replacement), String.valueOf / join / format, Object and getClass, and
// StringBuilder. Every expected value is java 24.0.2's, run with
// -Duser.language=en -Duser.country=US: String.format rounds a double's
// Double.toString digits HALF_UP and groups with ',' in that locale.
class S49 {
    static String cat(String[] a) {
        String s = a.length + ":";
        for (String x : a) { s = s + "[" + x + "]"; }
        return s;
    }
    static void run() {
        String[] ss = { "  Hi There \t", "a,b,,c,,", "ÄbC" };
        String t = ss[0];
        Main.check("str1", t.trim().equals("Hi There") && t.strip().equals("Hi There") && t.stripLeading().equals("Hi There \t")
            && !t.isBlank() && "\t  ".isBlank() && t.toUpperCase().equals("  HI THERE \t") && ss[2].toLowerCase().equals("äbc")
            && ss[2].toUpperCase().equals("ÄBC"));
        // String.toUpperCase / toLowerCase are the FULL mapping with the final sigma;
        // Character.toUpperCase stays one unit (java 24).
        Main.check("str1b", "\u00dfx".toUpperCase().equals("SSX") && "\ufb01".toUpperCase().equals("FI")
            && "\u0130".toLowerCase().equals("i\u0307") && Character.toUpperCase('\u00df') == '\u00df'
            && "A\u03a3".toLowerCase().equals("a\u03c2") && "\u03a3A".toLowerCase().equals("\u03c3a")
            && "\u0390".toUpperCase().length() == 3);
        Main.check("str2", t.indexOf('e') == 7 && t.indexOf("e", 8) == 9 && t.lastIndexOf("e") == 9 && t.lastIndexOf('e', 8) == 7
            && t.indexOf("", 99) == 12 && t.contains("Th") && t.startsWith("Hi", 2) && t.endsWith("\t") && t.indexOf("zz") == -1);
        Main.check("str3", "abc".compareTo("abd") == -1 && "b".compareTo("abc") == 1 && "abc".compareTo("ab") == 1
            && "ABC".compareToIgnoreCase("abd") == -1 && "äB".equalsIgnoreCase("ÄB") && !"ab".equalsIgnoreCase("abc")
            && "ab".concat("!").equals("ab!") && "x".repeat(3).equals("xxx") && "".repeat(5).isEmpty());
        Main.check("str4", "a-b".replace('-', '+').equals("a+b") && "aaa".replace("aa", "b").equals("ba")
            && "ab".replace("", "-").equals("-a-b-") && "Hello".codePointAt(1) == 101 && "Hello".subSequence(1, 3).equals("el"));
        Main.check("str5", cat(ss[1].split(",")).equals("4:[a][b][][c]") && cat(ss[1].split(",", -1)).equals("6:[a][b][][c][][]")
            && cat(ss[1].split(",", 2)).equals("2:[a][b,,c,,]") && cat("a1b22c".split("[0-9]+")).equals("3:[a][b][c]")
            && cat("abc".split("")).equals("3:[a][b][c]") && cat("".split(",")).equals("1:[]") && cat(" x  y ".split("\\s+")).equals("3:[][x][y]"));
        Main.check("str6", "a-b-c".replaceAll("(\\w)-(\\w)", "$2+$1").equals("b+a-c") && "x1y2".replaceAll("(?<d>[0-9])", "${d}${d}").equals("x11y22")
            && "aaa".replaceAll("a*", "X").equals("XX") && "abc".replaceFirst("[bc]", "\\$").equals("a$c") && "a1".matches("[a-z][0-9]")
            && !"a1x".matches("[a-z][0-9]"));
        String rep = "";
        try { "x".repeat(-2); } catch (IllegalArgumentException e) { rep = e.getMessage(); }
        Main.check("str7", rep.equals("count is negative: -2"));
        java.util.List<String> parts = new java.util.ArrayList<>();
        parts.add("p");
        parts.add("q");
        char[] cs = { 'h', 'e', 'y' };
        Main.check("sst1", String.valueOf(42).equals("42") && String.valueOf(true).equals("true") && String.valueOf('c').equals("c")
            && String.valueOf(1.5).equals("1.5") && String.valueOf(cs).equals("hey") && String.valueOf(cs, 1, 2).equals("ey")
            && String.valueOf((Object) null).equals("null") && String.join(",", "a", "b").equals("a,b")
            && String.join("/", parts).equals("p/q") && String.join("-", new String[] {"m", "n"}).equals("m-n"));
        Main.check("fmt1", String.format("%d|%5d|%-5d|%05d|%,d|%+d|%(d|%x|%X|%o|%#x", 42, 42, 42, -42, 1234567, 42, -42, -1, 255, 8, 255)
            .equals("42|   42|42   |-0042|1,234,567|+42|(42)|ffffffff|FF|10|0xff"));
        Main.check("fmt2", String.format("%.2f|%.1f|%.2f|%.0f|%f|%10.3f|%e|%.3e", 1.005, 0.15, 9.995, 0.5, 1.0 / 3, 3.14159, 123.0, 0.000123)
            .equals("1.01|0.2|10.00|1|0.333333|     3.142|1.230000e+02|1.230e-04"));
        Main.check("fmt3", String.format("%s|%5s|%-3s|%.2s|%S|%c|%b|%%|%2$s %1$s", "ab", "r", "l", "xyz", "up", 'z', null)
            .equals("ab|    r|l  |xy|UP|z|false|%|r ab") && String.format("%x %d %s", (byte) -1, Long.MAX_VALUE, 1.5f).equals("ff 9223372036854775807 1.5")
            && String.format("%.3g|%g|%e", 1234.5678, 0.0001, Double.MIN_VALUE).equals("1.23e+03|0.000100000|4.900000e-324"));
        Object o = new Object();
        Main.check("cls1", o.getClass().getName().equals("java.lang.Object") && "s".getClass().getSimpleName().equals("String")
            && ("" + o.getClass()).equals("class java.lang.Object") && Integer.valueOf(1).getClass().getName().equals("java.lang.Integer")
            && new S49().getClass().getName().equals("S49") && "a".getClass() == "b".getClass() && !o.equals(new Object())
            && o.toString().startsWith("java.lang.Object@") && new RuntimeException().getClass().getSimpleName().equals("RuntimeException"));
        StringBuilder sb = new StringBuilder();
        sb.append("x").append(1).append('c').append(2.5).append(true).append((Object) null);
        sb.insert(0, "<").insert(sb.length(), '>');
        StringBuilder r = new StringBuilder("hello").reverse();
        Main.check("sb1", sb.toString().equals("<x1c2.5truenull>") && sb.length() == 16 && r.toString().equals("olleh")
            && r.charAt(1) == 'l' && r.indexOf("l") == 1 && r.lastIndexOf("l") == 2 && new StringBuilder(10).length() == 0
            && new StringBuilder('a').length() == 0);
        r.setCharAt(0, 'O');
        r.deleteCharAt(1);
        r.delete(1, 2);
        r.replace(0, 1, "AB");
        String sio = "";
        try { r.charAt(9); } catch (StringIndexOutOfBoundsException e) { sio = e.getMessage(); }
        StringBuilder big = new StringBuilder();
        for (int i = 0; i < 3000; i++) { big.append(i % 10); }
        Main.check("sb2", r.toString().equals("ABeh") && r.substring(1, 3).equals("Be") && sio.equals("Index 9 out of bounds for length 4")
            && big.length() == 3000 && big.charAt(1234) == '4' && big.toString().endsWith("789"));
        // Short and empty pieces, a few longer than lib/java-lang.java's 64-char
        // tail, and reads between the appends. 3000 appends also bound the cost of
        // an append: the -frozen interpreter must finish this file under -max-steps.
        String[] ws = { "ab", "", "cde", "f", "0123456789012345678901234567890123456789012345678901234567890123456789x" };
        StringBuilder w = new StringBuilder("<");
        int probe = 0;
        for (int i = 0; i < 3000; i++) {
            w.append(ws[i % 4]);
            if (i % 997 == 0) {
                w.append(ws[4]);
                probe = probe * 31 + w.length() + w.charAt(w.length() - 1);
            }
        }
        Main.check("sb3", w.length() == 4785 && probe == 7577883 && w.charAt(4000) == 'a' && w.substring(70, 84).equals("789xcdefabcdef")
            && w.indexOf("xab") == 4772 && w.lastIndexOf("cde") == 4781 && w.toString().hashCode() == -1379603752);
    }
}

// ===== SECTION 50: static overload choice through interfaces, type variables, var =====
//
// javac chooses an overload from the arguments' STATIC types (JLS 15.12.2): an
// interface parameter is more specific than Object and a sub-interface than
// its super-interface; a type variable's static type is its erasure (JLS 4.6),
// a generic call's result is the receiver's type argument, `var` takes its
// initializer's type (JLS 14.4.1), and an anonymous class passes its
// arguments to the super constructor javac picks for them. Every expected
// value is java 24.0.2's.
interface S50Shape { }
interface S50Round extends S50Shape { }
class S50Circle implements S50Round { }
class S50Base {
    String tag;
    S50Base(Object o) { tag = "O"; }
    S50Base(String s) { tag = "S"; }
}
class S50Box<T> {
    T v;
    S50Box(T v) { this.v = v; }
    T get() { return v; }
    String show() { return S50.f(v); }
}
class S50 {
    static String f(Object o) { return "O"; }
    static String f(String s) { return "S"; }
    static String g(S50Shape s) { return "Shape"; }
    static String g(S50Round r) { return "Round"; }
    static String g(Object o) { return "Object"; }
    static <T> String gen(T t) { return f(t); }
    static void run() {
        S50Circle c = new S50Circle();
        S50Shape sh = c;
        Main.check("ov50a", g(c).equals("Round") && g(sh).equals("Shape") && g((Object) c).equals("Object"));
        S50Box<String> b = new S50Box<>("z");
        java.util.List<Object> lo = new java.util.ArrayList<>();
        lo.add("q");
        Main.check("ov50b", gen("x").equals("O") && b.show().equals("O") && f(b.get()).equals("S") && f(lo.get(0)).equals("O"));
        Object o = "s";
        var v = o;
        var w = "t";
        Main.check("ov50c", f(v).equals("O") && f(w).equals("S") && new S50Base(o) { }.tag.equals("O")
            && new S50Base("s") { }.tag.equals("S"));
    }
}

// ===== SECTION 51: astral characters in literals, super past the last user class =====
//
// A String is UTF-16 (JLS 3.1): an astral character, written as itself or as
// two unicode escapes (JLS 3.3), is a surrogate PAIR, and a lone surrogate escape is
// one char. `super.m()` past the last user class reaches java.lang.Object (or
// the toString Throwable and Enum put in its place). Every expected value is
// java 24.0.2's.
class S51Plain {
    String objStr() { return super.toString(); }
    boolean same(Object o) { return super.equals(o); }
    int idHash() { return super.hashCode(); }
}
class S51Named extends S51Plain {
    public String toString() { return "N:" + super.toString().startsWith("S51Named@"); }
}
class S51Exc extends RuntimeException {
    S51Exc(String m) { super(m); }
    public String toString() { return "<" + super.toString() + ">"; }
}
enum S51Color {
    RED;
    public String toString() { return super.toString().toLowerCase(); }
}
class S51 {
    static void run() {
        String[] lit = {"😀", "\uD83D\uDE00", "a😀b"};
        Main.check("astral51a", (int) lit[0].charAt(0) == 55357 && (int) lit[0].charAt(1) == 56832
            && lit[0].length() == 2 && lit[0].equals(lit[1]) && lit[0].codePointAt(0) == 128512);
        Main.check("astral51b", lit[2].indexOf("b") == 3 && lit[2].substring(1, 3).equals(lit[0]));
        char[] lone = {'\uD83D', '\uDE00'};
        Main.check("astral51c", (int) lone[0] == 55357 && (int) lone[1] == 56832);
        StringBuilder sb51 = new StringBuilder();
        sb51.append(lone[0]).append(lone[1]);
        Main.check("astral51d", (int) ("" + lone[0]).charAt(0) == 55357 && (int) String.valueOf(lone[1]).charAt(0) == 56832
            && ("" + lone[0] + lone[1]).equals(lit[0]) && new String(lone).equals(lit[0]) && sb51.toString().equals(lit[0]));
        S51Plain p = new S51Plain();
        Main.check("super51a", p.objStr().equals("S51Plain@" + Integer.toHexString(p.idHash()))
            && p.objStr().equals(p.toString()) && p.same(p) && !p.same(new S51Plain()) && !p.same(null));
        Main.check("super51b", ("" + new S51Named()).equals("N:true") && ("" + new S51Exc("x")).equals("<S51Exc: x>")
            && ("" + S51Color.RED).equals("red"));
    }
}

// ===== SECTION 52: for-update lists, a guard naming a variable, Unicode names =====
//
// A for statement's init and update parts are comma lists (JLS 14.14.1); a `when`
// guard may be a bare name, whose `->` ends the label rather than starting a
// lambda; and an identifier is any run of Character.isJavaIdentifierStart / Part
// characters (JLS 3.8), a unicode escape spelling one included. Every expected
// value is java 24.0.2's.
class S52 {
    static int guard(Object o, boolean g) {
        return switch (o) {
            case Integer j when g -> 0;
            case Integer j when (j > 3) -> 1;
            default -> 2;
        };
    }
    static int ignorable(char zero\u0000zero) { return zero\u0000zero; }
    static void run() {
        String trail = "";
        int i, j, n = 0;
        for (i = 0, j = 5; i < j; i++, j--, trail += "z") { n = n * 10 + i; }
        Main.check("for52a", n == 12 && i == 3 && j == 2 && trail.equals("zzz"));
        Main.check("when52a", guard(5, true) == 0 && guard(5, false) == 1 && guard(2, false) == 2 && guard("x", true) == 2);
        int café = 2;
        double π = 3.5;
        int £x = 4;
        int 𐐀abc = 1;
        Main.check("id52a", café + £x == 6 && π == 3.5 && 𐐀abc == 1 && café == 2);
        Main.check("id52b", ignorable((char) 65) == 65);
    }
}

// ===== SECTION 53: System.arraycopy, and the ArrayList moves built on it =====
//
// An overlapping copy within one array reads as if through a temporary; the
// bounds are checked source index, destination index, length, last source
// index, last destination index, in that order; ArrayList's add(int, E),
// remove(int), addAll, toArray and growth move its elements with it. Every
// expected string is java 24.0.2's.
class S53 {
    static String copy(Object src, int sp, Object dst, int dp, int n) {
        try { System.arraycopy(src, sp, dst, dp, n); return "ok"; }
        catch (ArrayIndexOutOfBoundsException e) { return e.getMessage(); }
        catch (NullPointerException e) { return "npe"; }
    }
    static String join(int[] a) {
        String s = "";
        for (int x : a) { s = s + x; }
        return s;
    }
    static void run() {
        int[] a = { 1, 2, 3, 4, 5 };
        System.arraycopy(a, 0, a, 1, 4);
        String right = join(a);
        System.arraycopy(a, 1, a, 0, 4);
        String left = join(a);
        int[] b = new int[3];
        System.arraycopy(a, 2, b, 1, 2);
        Main.check("ac53a", right.equals("11234") && left.equals("12344") && join(b).equals("034"));
        Object[] o = new Object[2];
        Main.check("ac53b", copy(a, -1, a, 0, 1).equals("arraycopy: source index -1 out of bounds for int[5]")
            && copy(a, 0, a, -2, 1).equals("arraycopy: destination index -2 out of bounds for int[5]")
            && copy(a, 0, a, 0, -1).equals("arraycopy: length -1 is negative")
            && copy(a, 3, a, 0, 3).equals("arraycopy: last source index 6 out of bounds for int[5]")
            && copy(o, 0, o, 1, 2).equals("arraycopy: last destination index 3 out of bounds for object array[2]")
            && copy(null, 0, a, 0, 1).equals("npe") && copy(a, 5, a, 0, 0).equals("ok"));
        ArrayList<Integer> xs = new ArrayList<>(0);
        for (int i = 0; i < 40; i++) { xs.add(0, i); }
        int front = xs.remove(0);
        int mid = xs.remove(19);
        xs.add(20, 99);
        int last = xs.remove(xs.size() - 1);
        ArrayList<Integer> ys = new ArrayList<>(xs);
        ys.addAll(List.of(7, 8));
        Object[] arr = ys.toArray();
        Main.check("al53a", front == 39 && mid == 19 && last == 0 && xs.size() == 38 && xs.get(18) == 20
            && xs.get(19) == 18 && xs.get(20) == 99 && xs.get(37) == 1 && ys.size() == 40
            && ys.get(39) == 8 && arr.length == 40 && arr[38].equals(7));
        ys.clear();
        boolean empty = ys.isEmpty() && ("" + ys).equals("[]");
        ys.add(5);
        Main.check("al53b", empty && ys.size() == 1 && ys.get(0) == 5 && ("" + ys).equals("[5]") && ys.indexOf(null) == -1);
    }
}

// ===== SECTION 54: java.util.Arrays, Comparable wrappers, fail-fast iterators =====
//
// Arrays' sort is stable and orders a double[] as Double.compare does; a typed
// overload decides the zero a copy pads with and what a fill converts its value
// to; asList writes through to its array and cannot change size. An
// ArrayList's iterator and forEach throw once the list changed size behind
// them, and a for-each variable declared Integer makes remove(x) the
// remove(Object) overload. Every expected string is java 24.0.2's.
class S54 {
    static <T> String g(T[] a) { return "T[]"; }
    static String g(int[] a) { return "int[]"; }
    static double[] none() { return new double[0]; }
    static String cme(ArrayList<Integer> l, int victim) {
        try {
            for (Integer x : l) { if (x == victim) { l.remove(x); } }
            return "" + l;
        } catch (ConcurrentModificationException e) { return "cme"; }
    }
    static void run() {
        int[] a = { 3, 1, 2 };
        Arrays.sort(a);
        double[] d = { 2.5, -0.0, 0.0, Double.NaN, -1 };
        Arrays.sort(d);
        String[] s = { "pear", "apple", "fig", "kiwi" };
        Arrays.sort(s, (x, y) -> x.length() - y.length());
        long[] ls = { 5L, 1L << 40, -7L };
        Arrays.sort(ls);
        Main.check("ar54a", Arrays.toString(a).equals("[1, 2, 3]") && Arrays.toString(d).equals("[-1.0, -0.0, 0.0, 2.5, NaN]")
            && Arrays.toString(s).equals("[fig, pear, kiwi, apple]") && Arrays.toString(ls).equals("[-7, 5, 1099511627776]"));
        int[] part = { 5, -3, 9, 0, 2 };
        Arrays.sort(part, 1, 4);
        Integer[] boxed = { 4, 1, 3 };
        Arrays.sort(boxed);
        char[] cs = { 'z', 'a', 'm' };
        Arrays.sort(cs);
        Main.check("ar54b", Arrays.toString(part).equals("[5, -3, 0, 9, 2]") && Arrays.toString(boxed).equals("[1, 3, 4]")
            && Arrays.toString(cs).equals("[a, m, z]") && Arrays.binarySearch(a, 2) == 1 && Arrays.binarySearch(a, 0) == -1
            && Arrays.binarySearch(a, 9) == -4);
        double[] f = new double[2];
        Arrays.fill(f, 1);
        long[] lf = new long[2];
        Arrays.fill(lf, 7);
        int[] g = new int[5];
        Arrays.fill(g, 1, 3, 9);
        Main.check("ar54c", Arrays.toString(f).equals("[1.0, 1.0]") && lf[0] * 1000000000 * 10 == 70000000000L
            && Arrays.toString(g).equals("[0, 9, 9, 0, 0]"));
        Main.check("ar54d", Arrays.toString(Arrays.copyOf(a, 5)).equals("[1, 2, 3, 0, 0]") && Arrays.copyOf(a, 2).length == 2
            && Arrays.toString(Arrays.copyOf(s, 5)).equals("[fig, pear, kiwi, apple, null]")
            && Arrays.toString(Arrays.copyOf(none(), 2)).equals("[0.0, 0.0]")
            && Arrays.toString(Arrays.copyOf(new boolean[] { true }, 2)).equals("[true, false]")
            && Arrays.toString(Arrays.copyOfRange(part, 3, 7)).equals("[9, 2, 0, 0]") && g(new int[0]).equals("int[]"));
        Main.check("ar54e", Arrays.equals(a, new int[] { 1, 2, 3 }) && !Arrays.equals(a, new int[] { 1, 2 })
            && Arrays.equals(new double[] { Double.NaN }, new double[] { Double.NaN })
            && !Arrays.equals(new double[] { 0.0 }, new double[] { -0.0 }) && Arrays.hashCode(a) == 30817
            && Arrays.hashCode(s) == -2070968051 && Arrays.hashCode(new double[] { 1.5 }) == 1073217567);
        String[] backing = { "p", "q" };
        List<String> v = Arrays.asList(backing);
        v.set(0, "P");
        backing[1] = "Q";
        String uoe = "";
        try { v.add("r"); } catch (UnsupportedOperationException e) { uoe = "uoe"; }
        List<Integer> copy = new ArrayList<>(Arrays.asList(3, 4));
        copy.add(5);
        Main.check("ar54f", ("" + v).equals("[P, Q]") && uoe.equals("uoe") && Arrays.asList("x", "y").indexOf("y") == 1
            && copy.equals(Arrays.asList(3, 4, 5)) && Arrays.asList(3, 4, 5).equals(copy));
        String msgs = "";
        try { Arrays.copyOfRange(a, 2, 1); } catch (IllegalArgumentException e) { msgs = msgs + e.getMessage() + "|"; }
        try { Arrays.fill(g, 3, 1, 0); } catch (IllegalArgumentException e) { msgs = msgs + e.getMessage() + "|"; }
        try { Arrays.sort(g, 0, 9); } catch (ArrayIndexOutOfBoundsException e) { msgs = msgs + e.getMessage(); }
        int[] nul = null;
        Main.check("ar54g", msgs.equals("2 > 1|fromIndex(3) > toIndex(1)|Array index out of range: 9")
            && Arrays.toString(nul).equals("null"));
        Integer i3 = 3;
        Double d15 = 1.5;
        Boolean yes = true;
        Character c = 'c';
        Short sh = (short) 3;
        Float nan = Float.NaN;
        Main.check("cmp54", i3.compareTo(4) == -1 && d15.compareTo(-2.0) == 1 && yes.compareTo(false) == 1 && c.compareTo('a') == 2
            && sh.compareTo((short) 10) == -7 && nan.compareTo(1f) == 1 && yes.compareTo(true) == 0);
        ArrayList<Integer> l = new ArrayList<>();
        l.add(1);
        l.add(2);
        l.add(3);
        Iterator<Integer> it = l.iterator();
        l.clear();
        l.add(5);
        String first = "";
        try { first = "" + it.next(); } catch (ConcurrentModificationException e) { first = "cme"; }
        Iterator<Integer> it2 = l.iterator();
        l.set(0, 8);
        int second = it2.next();
        String each = "";
        try { l.forEach(x -> { if (x == 8) { l.add(1); } }); } catch (ConcurrentModificationException e) { each = "cme"; }
        ArrayList<Integer> m = new ArrayList<>(List.of(5, 6, 7));
        ArrayList<Integer> n = new ArrayList<>(List.of(5, 6, 7));
        Main.check("cme54", first.equals("cme") && second == 8 && each.equals("cme") && ("" + l).equals("[8, 1]")
            && cme(m, 6).equals("[5, 7]") && cme(n, 5).equals("cme"));
    }
}

// ===== SECTION 55: instanceof a boxed primitive, the wrappers' members, array clone =====
//
// A boxed primitive is an instance of its own wrapper class and that class's
// supertypes only (JLS 15.20.2): an Integer is a Number and a Comparable but not
// a Double, a char is a Character and not a Number. Number's xxxValue() is the
// primitive conversion to that type, and an array's clone() is a shallow copy
// (JLS 10.7). Every expected value is java 24.0.2's.
class S55 {
    static String kinds(Object o) {
        return (o instanceof Integer ? "I" : "") + (o instanceof Long ? "L" : "") + (o instanceof Short ? "S" : "")
            + (o instanceof Byte ? "Y" : "") + (o instanceof Double ? "D" : "") + (o instanceof Float ? "F" : "")
            + (o instanceof Character ? "C" : "") + (o instanceof Boolean ? "B" : "") + (o instanceof String ? "T" : "")
            + (o instanceof Number ? "N" : "") + (o instanceof Comparable ? "c" : "") + (o instanceof Object ? "O" : "");
    }
    static String sw(Object o) {
        return switch (o) {
            case Integer i -> "int " + (i + 1);
            case Double d -> "double " + d;
            case Character c -> "char " + c;
            default -> "other";
        };
    }
    static void run() {
        Object[] xs = { 3, "b", 2.5, 'c', 7L, true, 1.5f, (short) 2, (byte) 1 };
        String all = "";
        for (Object o : xs) { all = all + kinds(o) + " "; }
        Main.check("io55a", all.equals("INcO TcO DNcO CcO LNcO BcO FNcO SNcO YNcO "));
        Object q = 4.0;
        Object n = null;
        Main.check("io55b", !(q instanceof Integer) && q instanceof Double d && d == 4.0 && !(n instanceof Object)
            && sw(4).equals("int 5") && sw(4.0).equals("double 4.0") && sw('z').equals("char z") && sw(4L).equals("other"));
        Object[] ns = { 3, 2.75, 7L, 1.5f, (short) -2, (byte) -1, -0.0, 1e20, 1L << 40 };
        String vals = "";
        for (Object o : ns) {
            Number x = (Number) o;
            vals = vals + x.intValue() + "/" + x.longValue() + "/" + x.doubleValue() + "/" + x.floatValue() + "/"
                + x.shortValue() + "/" + x.byteValue() + " ";
        }
        Main.check("wr55a", vals.equals("3/3/3.0/3.0/3/3 2/2/2.75/2.75/2/2 7/7/7.0/7.0/7/7 1/1/1.5/1.5/1/1 -2/-2/-2.0/-2.0/-2/-2 "
            + "-1/-1/-1.0/-1.0/-1/-1 0/0/-0.0/-0.0/0/0 2147483647/9223372036854775807/1.0E20/1.0E20/-1/-1 "
            + "0/1099511627776/1.099511627776E12/1.0995116E12/0/0 "));
        Integer i5 = 5;
        Character cq = Character.valueOf('q');
        Boolean yes = Boolean.TRUE;
        Main.check("wr55b", i5.doubleValue() == 5.0 && ("" + i5.doubleValue()).equals("5.0") && cq.charValue() == 'q'
            && yes.booleanValue() && !Boolean.FALSE && Character.compare('a', 'z') == -25 && Character.compare('b', 'a') == 1
            && Boolean.compare(true, false) == 1 && Boolean.compare(false, true) == -1 && Boolean.compare(true, true) == 0
            && Boolean.logicalXor(true, false) && !Boolean.logicalAnd(true, false) && Boolean.logicalOr(false, true)
            && Boolean.toString(false).equals("false") && ("" + Character.valueOf('x')).equals("x"));
        int[] a = { 3, 1, 2 };
        int[] b = a.clone();
        b[0] = 9;
        int[][] m = { { 1, 2 }, { 3 } };
        int[][] mc = m.clone();
        mc[0][0] = 7;
        mc[1] = new int[] { 8 };
        long[] ls = { 1L << 40 };
        Main.check("cl55", Arrays.toString(a).equals("[3, 1, 2]") && Arrays.toString(b).equals("[9, 1, 2]") && a != b
            && m[0][0] == 7 && m[1][0] == 3 && m[0] == mc[0] && ls.clone()[0] * 2 == 2199023255552L
            && new char[0].clone().length == 0 && "hey".toCharArray().clone().getClass().getName().equals("[C"));
    }
}

// ===== SECTION 56: java.util.stream and Optional =====
//
// A sequential stream runs each element through the whole pipeline before it
// reads the next, so the lambdas' side effects interleave, a short-circuiting
// operation reads no further than it needs, and sorted() first drains what is
// upstream of it; a stream can be operated upon once. Collectors.toList,
// joining and counting; IntStream's sum wraps as int arithmetic does. Every
// expected value is java 24.0.2's.
class S56 {
    static String trace = "";
    static void run() {
        Main.check("st56a", Arrays.stream(new int[] { 1, 2, 3 }).sum() == 6 && IntStream.range(0, 4).sum() == 6
            && IntStream.rangeClosed(1, 5).map(x -> x * x).filter(x -> x % 2 == 1).boxed().collect(Collectors.toList()).toString().equals("[1, 9, 25]")
            && IntStream.of(Integer.MAX_VALUE, 1).sum() == Integer.MIN_VALUE
            && IntStream.rangeClosed(Integer.MAX_VALUE - 1, Integer.MAX_VALUE).count() == 2);
        Main.check("st56b", Stream.of("a", "bb", "ccc").map(s -> s.toUpperCase()).collect(Collectors.joining(", ", "[", "]")).equals("[A, BB, CCC]")
            && Stream.of(5, 3, 9, 1).sorted().map(x -> x * 10).reduce(0, (x, y) -> x + y) == 180 && Stream.of(1, 2, 3).count() == 3
            && Arrays.stream(new String[] { "q", "r" }).collect(Collectors.joining()).equals("qr")
            && IntStream.range(0, 3).mapToObj(i -> "i" + i).collect(Collectors.joining("-")).equals("i0-i1-i2"));
        List<String> w = List.of("pear", "fig", "apple");
        Object cnt = Stream.of(1, 2).collect(Collectors.counting());
        Main.check("st56c", w.stream().filter(s -> s.length() > 3).count() == 2 && w.stream().sorted().toList().toString().equals("[apple, fig, pear]")
            && cnt instanceof Long && Stream.of("bb", "a", "ccc").sorted((x, y) -> y.length() - x.length()).toList().toString().equals("[ccc, bb, a]")
            && Stream.of(1, 1, 2, 1, 3).distinct().toList().toString().equals("[1, 2, 3]") && Stream.of("x", "yy").mapToInt(s -> s.length()).sum() == 3
            && Arrays.toString(IntStream.range(0, 3).toArray()).equals("[0, 1, 2]") && IntStream.range(0, 5).skip(2).limit(2).boxed().toList().toString().equals("[2, 3]"));
        Main.check("st56d", ("" + IntStream.of(4, 9, 2).max()).equals("OptionalInt[9]") && ("" + IntStream.empty().max()).equals("OptionalInt.empty")
            && ("" + IntStream.of(1, 2).average()).equals("OptionalDouble[1.5]") && ("" + IntStream.of(5, 6).reduce((x, y) -> x * y)).equals("OptionalInt[30]")
            && Stream.of(3, 1, 2).max((x, y) -> x - y).get() == 3 && Stream.of(3, 1).min(Integer::compare).get() == 1
            && Stream.of(1, 2, 3).anyMatch(x -> x > 2) && !Stream.of(1, 2, 3).allMatch(x -> x > 2) && IntStream.of(1, 2).noneMatch(x -> x > 5)
            && Stream.of("x", "y").findFirst().get().equals("x") && ("" + IntStream.range(0, 0).findFirst()).equals("OptionalInt.empty"));
        Stream.of(1, 2, 3, 4).filter(x -> { trace = trace + "f" + x; return x % 2 == 0; })
            .map(x -> { trace = trace + "m" + x; return x; }).limit(1).forEach(x -> trace = trace + "e" + x);
        trace = trace + "|";
        Stream.of(3, 1, 2).peek(x -> trace = trace + "p" + x).sorted().forEach(x -> trace = trace + "s" + x);
        Stream<Integer> once = Stream.of(1);
        once.count();
        String ise = "";
        try { once.count(); } catch (IllegalStateException e) { ise = e.getMessage(); }
        ArrayList<Integer> al = new ArrayList<>(List.of(1, 2));
        Stream<Integer> late = al.stream();
        al.add(3);
        Main.check("st56e", trace.equals("f1f2m2e2|p3p1p2s1s2s3") && ise.equals("stream has already been operated upon or closed")
            && late.map(x -> x + 1).toList().toString().equals("[2, 3, 4]"));
        Optional<String> o = Optional.of("x");
        String nse = "";
        try { Optional.empty().get(); } catch (NoSuchElementException e) { nse = e.getMessage(); }
        Main.check("op56", o.get().equals("x") && ("" + o).equals("Optional[x]") && ("" + Optional.empty()).equals("Optional.empty")
            && o.map(s -> s + "!").orElse("no").equals("x!") && !Optional.ofNullable(null).isPresent() && nse.equals("No value present")
            && Optional.empty() == Optional.empty() && OptionalInt.of(3).getAsInt() == 3 && OptionalInt.empty().orElse(7) == 7);
    }
}

// ===== SECTION 57: HashMap, Map.Entry, Collectors.groupingBy, qualified type names =====
//
// A HashMap iterates in its table's order: the spread hash picks the bin, a
// put appends to its bin while computeIfAbsent / merge / compute insert at the
// head and grow the table before inserting, and a resize splits every bin in
// order - so the printed order below is java's own. groupingBy collects into a
// HashMap through computeIfAbsent. A qualified type name (Map.Entry, an
// Outer.Inner) works in instanceof, in implements and as a static qualifier.
// Every expected value is java 24.0.2's.
interface S57Outer {
    interface S57Lid { static int m() { return 42; } }
    class S57Cup implements S57Outer.S57Lid { }
}
class S57 {
    static void run() {
        HashMap<Integer, String> h = new HashMap<>();
        h.computeIfAbsent(1, k -> "a");
        h.computeIfAbsent(17, k -> "b");
        h.put(33, "c");
        h.merge(49, "d", (x, y) -> x + y);
        h.compute(65, (k, v) -> "e");
        HashMap<Integer, Integer> a = new HashMap<>();
        HashMap<Integer, Integer> b = new HashMap<>();
        for (int i = 0; i < 13; i++) {
            int k = (i % 2 == 0) ? i : i + 16;
            a.put(k, i);
            b.computeIfAbsent(k, x -> 0);
        }
        Main.check("hm57a", h.toString().equals("{65=e, 49=d, 17=b, 1=a, 33=c}")
            && a.keySet().toString().equals("[0, 2, 4, 6, 8, 10, 12, 17, 19, 21, 23, 25, 27]")
            && b.keySet().toString().equals("[0, 17, 2, 19, 4, 21, 6, 23, 8, 25, 10, 27, 12]"));
        HashMap<String, Integer> s = new HashMap<>();
        for (String w : "the quick brown fox jumps over the lazy dog again and again".split(" ")) { s.merge(w, 1, Integer::sum); }
        Map.Entry<String, Integer> first = s.entrySet().iterator().next();
        Main.check("hm57b", s.toString().equals("{over=1, the=2, quick=1, and=1, lazy=1, again=2, jumps=1, brown=1, dog=1, fox=1}")
            && s.get("again") == 2 && s.getOrDefault("cat", 0) == 0 && s.containsKey("fox") && !s.containsKey("cat")
            && first.toString().equals("over=1") && first instanceof Map.Entry && s.equals(new HashMap<>(s)));
        s.remove("the");
        s.put(null, 0);
        Main.check("hm57c", s.hashCode() == 403228852 && s.get(null) == 0 && s.size() == 10
            && s.values().toString().equals("[1, 0, 1, 1, 1, 2, 1, 1, 1, 1]")
            && s.entrySet().stream().filter(e -> e.getKey() != null).sorted(Map.Entry.comparingByKey()).limit(3).toList().toString()
                .equals("[again=2, and=1, brown=1]"));
        HashMap<Integer, Integer> big = new HashMap<>();
        for (int i = 0; i < 100; i++) { big.put(i * 37 % 101, i); }
        String cme = "";
        try { for (Integer k : big.keySet()) { big.put(k + 1000, 0); } } catch (ConcurrentModificationException e) { cme = "cme"; }
        Main.check("hm57d", big.keySet().stream().limit(3).toList().toString().equals("[0, 1, 2]") && cme.equals("cme"));
        List<String> words = List.of("apple", "avocado", "banana", "blueberry", "cherry", "kiwi", "fig", "date", "plum", "pear");
        String npe = "";
        try { Stream.of("a", null).collect(Collectors.groupingBy(x -> x)); } catch (NullPointerException e) { npe = e.getMessage(); }
        Main.check("gb57", words.stream().collect(Collectors.groupingBy(w -> w.length())).toString()
                .equals("{3=[fig], 4=[kiwi, date, plum, pear], 5=[apple], 6=[banana, cherry], 7=[avocado], 9=[blueberry]}")
            && words.stream().collect(Collectors.groupingBy(w -> w.charAt(0), Collectors.counting())).toString()
                .equals("{p=2, a=2, b=2, c=1, d=1, f=1, k=1}")
            && npe.equals("element cannot be mapped to a null key"));
        Object box = new S57Outer.S57Cup();
        Main.check("qn57", S57Outer.S57Lid.m() == 42 && box instanceof S57Outer.S57Lid && box instanceof S57Outer.S57Cup
            && "s" instanceof java.lang.String && !(box instanceof java.lang.Number)
            && new String[] { "a", "b" }.getClass().getSimpleName().equals("String[]"));
    }
}

// ===== SECTION 58: static self-typed fields, unbound method references, array types =====
//
// A class's static initializers run once its methods and constructors exist,
// so one may construct the class itself or call its static methods. A method
// reference Type::m naming an instance method of a java.lang type is unbound
// (JLS 15.13.1): its first argument is the receiver. An array is created with
// a type that getClass() names whatever it holds, and clone() keeps it.
// Short.compare and Byte.compare answer x - y. Every expected value is java
// 24.0.2's.
final class S58Opt {
    static final S58Opt EMPTY = new S58Opt(7);
    static final int TWICE = twice(21);
    static int twice(int x) { return x * 2; }
    final int v;
    S58Opt(int v) { this.v = v; }
}
class S58Cell { int x; }
class S58 {
    static String names(Object[] os) {
        String s = "";
        for (Object o : os) { s = s + o.getClass().getSimpleName() + " " + o.getClass().getName() + ";"; }
        return s;
    }
    static void run() {
        Main.check("si58", S58Opt.EMPTY.v == 7 && S58Opt.TWICE == 42 && S58Opt.EMPTY == S58Opt.EMPTY);
        List<String> ws = List.of("bb", "a", "ccc", "dd");
        ArrayList<String> sorted = new ArrayList<>(ws);
        sorted.sort(String::compareTo);
        java.util.function.Function<String, Integer> len = String::length;
        java.util.function.BiFunction<String, String, String> cat = String::concat;
        java.util.function.Function<Object, String> str = Object::toString;
        java.util.function.Function<Object, String> val = String::valueOf;
        java.util.function.Function<Integer, Integer> iv = Integer::intValue;
        Main.check("mr58", ws.stream().collect(Collectors.groupingBy(String::length)).toString().equals("{1=[a], 2=[bb, dd], 3=[ccc]}")
            && ws.stream().map(String::toUpperCase).collect(Collectors.joining(",")).equals("BB,A,CCC,DD")
            && sorted.toString().equals("[a, bb, ccc, dd]") && len.apply("four") == 4 && cat.apply("x", "y").equals("xy")
            && str.apply(12).equals("12") && val.apply(null).equals("null") && iv.apply(5) == 5
            && Stream.of(3, 1, 2).sorted(Integer::compareTo).toList().toString().equals("[1, 2, 3]"));
        String[] s = { "x", null };
        Object[] c = s.clone();
        java.util.function.IntFunction<int[][]> mk = int[][]::new;
        Main.check("at58", names(new Object[] { s, c, new String[0], new int[0], new S58Cell[1], new int[2][3], new String[1][1],
                new Integer[] { 1 }, mk.apply(2), new double[0][] })
            .equals("String[] [Ljava.lang.String;;String[] [Ljava.lang.String;;String[] [Ljava.lang.String;;int[] [I;"
                + "S58Cell[] [LS58Cell;;int[][] [[I;String[][] [[Ljava.lang.String;;Integer[] [Ljava.lang.Integer;;"
                + "int[][] [[I;double[][] [[D;")
            && mk.apply(2)[0] == null);
        String[][] sm = { { "a" }, { "b" } };
        int[][] im = new int[2][0];
        Main.check("fm58", String.format("%s|%s", (Object[]) sm).replaceAll("@[0-9a-f]+", "@").equals("[Ljava.lang.String;@|[Ljava.lang.String;@")
            && String.format("%s+%s", (Object[]) im).replaceAll("@[0-9a-f]+", "@").equals("[I@+[I@"));
        short[] hs = { (short) 3, (short) -5, (short) 32767, (short) -32768 };
        byte[] bs = { (byte) 3, (byte) -5, (byte) 127, (byte) -128 };
        String cmp = "";
        for (int i = 0; i < 4; i++) { cmp = cmp + Short.compare(hs[i], hs[3 - i]) + " " + Byte.compare(bs[i], bs[3 - i]) + " "; }
        Main.check("sc58", cmp.equals("32771 131 -32772 -132 32772 132 -32771 -131 "));
    }
}

// ===== SECTION 59: LongStream, DoubleStream, HashSet, Collectors.toSet, Comparator =====
//
// DoubleStream's sum and average are compensated (Kahan) sums, so ten 0.1s
// sum to 1.0 where a loop gets 0.9999999999999999; its min and max are
// Math.min / Math.max. A LongStream range may end at Long.MAX_VALUE. A HashSet
// is a HashMap's key set and iterates in its table's order. Comparator's
// statics order by compareTo, and a comparator they return answers reversed()
// and thenComparing(). Every expected value is java 24.0.2's.
class S59 {
    static void run() {
        double[] tenth = new double[10];
        Arrays.fill(tenth, 0.1);
        double loop = 0;
        for (double d : tenth) { loop += d; }
        Main.check("ds59", DoubleStream.of(tenth).sum() == 1.0 && loop != 1.0 && Arrays.stream(tenth).average().getAsDouble() == 0.1
            && ("" + DoubleStream.of(0.0, -0.0).min() + DoubleStream.of(3.0, Double.NaN).max()).equals("OptionalDouble[-0.0]OptionalDouble[NaN]")
            && Arrays.toString(DoubleStream.of(3.0, -0.0, Double.NaN, 0.0).sorted().toArray()).equals("[-0.0, 0.0, 3.0, NaN]")
            && Double.isInfinite(DoubleStream.of(1.0, Double.POSITIVE_INFINITY).sum()) && ("" + DoubleStream.empty().average()).equals("OptionalDouble.empty")
            && DoubleStream.of(1, 2, 3).map(x -> x / 2).mapToObj(x -> "" + x).collect(Collectors.joining(",")).equals("0.5,1.0,1.5"));
        Main.check("ls59", LongStream.rangeClosed(Long.MAX_VALUE - 2, Long.MAX_VALUE).count() == 3 && LongStream.range(0, 5).sum() == 10
            && LongStream.of(Long.MAX_VALUE, 1).sum() == Long.MIN_VALUE && ("" + LongStream.of(4, 9).max()).equals("OptionalLong[9]")
            && ("" + LongStream.of(1, 2).average()).equals("OptionalDouble[1.5]") && IntStream.of(3, 4).asLongStream().map(x -> x * 3000000000L).sum() == 21000000000L
            && Stream.of("a", "bb").mapToDouble(String::length).sum() == 3.0 && IntStream.rangeClosed(Integer.MAX_VALUE - 1, Integer.MAX_VALUE).count() == 2);
        Set<String> hs = new HashSet<>(List.of("banana", "apple", "cherry", "date", "elderberry", "fig", "grape"));
        HashSet<Integer> hi = new HashSet<>();
        for (int i = 0; i < 20; i++) { hi.add(i * 37 % 50 - 10); }
        hi.removeIf(x -> x % 3 == 0);
        Set<Integer> t = Stream.of(3, 2, 1, 3).collect(Collectors.toSet());
        Main.check("hs59", hs.toString().equals("[banana, date, apple, cherry, fig, grape, elderberry]") && hs.hashCode() == -1879343967
            && hi.toString().equals("[-1, 32, 1, 34, -5, 38, -7, 8, -10, 10, 14, 19, 23, 25]") && !hi.add(32) && hi.remove(32) && !hi.contains(32)
            && t.equals(new HashSet<>(List.of(1, 2, 3))) && t.size() == 3 && !t.equals(hi));
        List<String> ws = new ArrayList<>(List.of("pear", "fig", "apple", "kiwi", "date"));
        ws.sort(Comparator.naturalOrder());
        String nat = ws.toString();
        ws.sort(Comparator.comparingInt(String::length).reversed());
        String rev = ws.toString();
        ws.sort(Comparator.comparing(String::length).thenComparing(Comparator.reverseOrder()));
        Main.check("cm59", nat.equals("[apple, date, fig, kiwi, pear]") && rev.equals("[apple, date, kiwi, pear, fig]")
            && ws.toString().equals("[fig, pear, kiwi, date, apple]")
            && Stream.of(5, 3, 9).sorted(Comparator.reverseOrder()).toList().toString().equals("[9, 5, 3]"));
    }
}
