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

import java.io.Serializable;
import java.lang.reflect.Array;
import java.util.AbstractList;
import java.util.ArrayDeque;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.BitSet;
import java.util.DoubleSummaryStatistics;
import java.util.LongSummaryStatistics;
import java.util.Collections;
import java.util.Comparator;
import java.util.ConcurrentModificationException;
import java.util.EmptyStackException;
import java.util.EnumMap;
import java.util.EnumSet;
import java.util.Deque;
import java.util.HashMap;
import java.util.HashSet;
import java.util.Iterator;
import java.util.LinkedHashMap;
import java.util.LinkedHashSet;
import java.util.LinkedList;
import java.util.List;
import java.util.ListIterator;
import java.util.Map;
import java.util.NavigableMap;
import java.util.NavigableSet;
import java.util.NoSuchElementException;
import java.util.Objects;
import java.util.Optional;
import java.util.OptionalInt;
import java.util.PriorityQueue;
import java.util.Random;
import java.util.RandomAccess;
import java.util.Set;
import java.util.Stack;
import java.util.StringJoiner;
import java.util.TreeMap;
import java.util.TreeSet;
import java.util.stream.Collectors;
import java.util.stream.DoubleStream;
import java.util.stream.IntStream;
import java.util.stream.LongStream;
import java.util.stream.Stream;
import java.util.regex.MatchResult;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

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
        S60.run(); // SECTION-CALL 60
        S61.run(); // SECTION-CALL 61
        S62.run(); // SECTION-CALL 62
        S63.run(); // SECTION-CALL 63
        S64.run(); // SECTION-CALL 64
        S65.run(); // SECTION-CALL 65
        S66.run(); // SECTION-CALL 66
        S67.run(); // SECTION-CALL 67
        S68.run(); // SECTION-CALL 68
        S69.run(); // SECTION-CALL 69
        S70.run(); // SECTION-CALL 70
        S71.run(); // SECTION-CALL 71
        S72.run(); // SECTION-CALL 72
        S73.run(); // SECTION-CALL 73
        S74.run(); // SECTION-CALL 74
        S75.run(); // SECTION-CALL 75
        S76.run(); // SECTION-CALL 76
        S77.run(); // SECTION-CALL 77
        S78.run(); // SECTION-CALL 78
        S79.run(); // SECTION-CALL 79
        S80.run(); // SECTION-CALL 80
        S81.run(); // SECTION-CALL 81
        S82.run(); // SECTION-CALL 82
        S83.run(); // SECTION-CALL 83
        S84.run(); // SECTION-CALL 84
        S85.run(); // SECTION-CALL 85
        S86.run(); // SECTION-CALL 86
        S87.run(); // SECTION-CALL 87
        S88.run(); // SECTION-CALL 88
        S89.run(); // SECTION-CALL 89
        S90.run(); // SECTION-CALL 90
        S91.run(); // SECTION-CALL 91
        S92.run(); // SECTION-CALL 92
        S93.run(); // SECTION-CALL 93
        S94.run(); // SECTION-CALL 94
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

// ===== SECTION 60: overloaded static and default methods of an interface =====
//
// An interface's overloads are chosen by their arguments as a class's are
// (JLS 15.12.2), whether static or default, and a class's override of one of
// them wins over the default. Comparator.comparing and thenComparing take a
// key comparator as a second argument. Every expected value is java 24.0.2's.
interface S60Shape {
    double area();
    default String describe(String unit) { return "area " + area() + " " + unit; }
    default String describe(int digits) { return "area~" + Math.round(area() * Math.pow(10, digits)); }
    static String by(S60Shape s) { return "one"; }
    static String by(S60Shape s, boolean loud) { return loud ? "TWO" : "two"; }
}
class S60Sq implements S60Shape {
    double s;
    S60Sq(double s) { this.s = s; }
    public double area() { return s * s; }
    public String describe(String unit) { return "square " + area() + unit; }
}
record S60P(String name, int age) {}
class S60 {
    static void run() {
        S60Shape q = new S60Sq(1.5);
        S60Shape lam = () -> 3.0;
        Main.check("io60", q.describe("m2").equals("square 2.25m2") && q.describe(1).equals("area~23")
            && lam.describe("u").equals("area 3.0 u") && lam.describe(2).equals("area~300")
            && S60Shape.by(q).equals("one") && S60Shape.by(q, true).equals("TWO") && S60Shape.by(lam, false).equals("two"));
        List<S60P> ps = new ArrayList<>(List.of(new S60P("bo", 30), new S60P("al", 25), new S60P("cy", 30)));
        ps.sort(Comparator.comparing(S60P::age).thenComparing(S60P::name, Comparator.reverseOrder()));
        List<String> ws = new ArrayList<>(List.of("pear", "fig", "apple"));
        ws.sort(Comparator.comparing(String::length, Comparator.reverseOrder()));
        Main.check("cm60", ps.toString().equals("[S60P[name=al, age=25], S60P[name=cy, age=30], S60P[name=bo, age=30]]")
            && ws.toString().equals("[apple, pear, fig]"));
    }
}

// ===== SECTION 61: static library fields, append(char[]), (char) of a double, ?: promotion, suppressed =====
//
// A static field may be initialised with a java.lang library type. append of a
// char[] appends its characters. (char) of a double narrows to int first,
// saturating (JLS 5.1.3). A numeric conditional has the promoted type of its
// arms, except that a char, byte or short arm keeps its type against an int
// constant that fits it (JLS 15.25.2). try-with-resources keeps the body's
// exception and adds close()'s to its suppressed list (JLS 14.20.3.1). Every
// expected value is java 24.0.2's.
class S61Res implements AutoCloseable {
    String n;
    static String log = "";
    S61Res(String n) { this.n = n; }
    public void close() {
        log = log + n + ";";
        if (n.startsWith("bad")) { throw new IllegalStateException("close-" + n); }
    }
}
class S61 {
    static StringBuilder sb = new StringBuilder("o");
    static void run() {
        sb.append(new char[]{'x', 'y'});
        char[] cs = {'p', 'q', 'r'};
        StringBuilder b2 = new StringBuilder("a").append(cs).append(cs, 1, 2).insert(1, cs);
        Main.check("sb61", sb.toString().equals("oxy") && b2.toString().equals("apqrpqrqr"));
        double[] ds = {1e10, -1e10, 3e9, -3e9, 65.7, Double.NaN};
        String chs = "";
        for (double d : ds) { chs = chs + (int) (char) d + " "; }
        float[] fs = {3e9f, -3e9f};
        for (float f : fs) { chs = chs + (int) (char) f + " "; }
        long[] ls = {10000000000L, -1L};
        for (long l : ls) { chs = chs + (int) (char) l + " "; }
        Main.check("ch61", chs.equals("65535 0 65535 0 65 0 65535 0 58368 65535 "));
        boolean[] tt = {true, false};
        boolean t = tt[0];
        char cv = 'b';
        int iv = 7;
        Integer boxed = 4;
        byte by = 2;
        Object o = t ? iv : 2.0;
        String cond = "" + (t ? cv : iv) + "," + (t ? iv : 2.5) + "," + (t ? boxed : 1.5) + "," + o + "," + (t ? 1f : iv)
            + "," + (!t ? 1.0 : iv) + "," + (int) (!t ? 'x' : 0) + "," + (t ? 66 : 'a') + "," + (t ? by : 100) + "," + (t ? by : 200);
        Main.check("cd61", cond.equals("98,7.0,4.0,7.0,1.0,7.0,0,B,2,2") && (true ? 1 : 'a') == '\u0001');
        String caught = "";
        try (S61Res r1 = new S61Res("bad")) {
            throw new RuntimeException("body-ex");
        } catch (RuntimeException e) {
            caught = e + "|" + e.getSuppressed().length + "|" + e.getSuppressed()[0];
        }
        String two = "";
        try (S61Res r1 = new S61Res("bad1"); S61Res r2 = new S61Res("bad2")) {
            S61Res.log = S61Res.log + "body;";
        } catch (IllegalStateException e) {
            two = e.getMessage() + "|" + e.getSuppressed()[0].getMessage();
        }
        String nul = "";
        try (S61Res r0 = null) { nul = "ran"; }
        Main.check("tw61", caught.equals("java.lang.RuntimeException: body-ex|1|java.lang.IllegalStateException: close-bad")
            && two.equals("close-bad2|close-bad1") && nul.equals("ran") && S61Res.log.equals("bad;body;bad2;bad1;"));
    }
}

// ===== SECTION 62: a null dereferenced or unboxed is a NullPointerException =====
//
// A member read or write, a method call, an array length or element, a
// for-each and a String method's argument on null throw NullPointerException
// (JLS 15.11.1, 15.12.4.4, 15.10.4, 14.14.2), and so does unboxing a null in
// arithmetic, a comparison with a primitive, a condition, a switch selector
// or an assignment to a primitive (JLS 5.1.8, 14.11.3); `throw null` throws
// one (JLS 14.18). A null compared by reference or concatenated is no
// unboxing. Every expected value is java 24.0.2's.
class S62B {
    int v = 1;
    int get() { return v; }
}
class S62 {
    static Object[] nulls = {null};
    static String log = "";
    static void k(String id, Runnable r) {
        try { r.run(); log = log + id + ":none "; }
        catch (NullPointerException e) { log = log + id + ":npe "; }
        catch (RuntimeException e) { log = log + id + ":other "; }
    }
    static void run() {
        k("a", () -> { S62B b = (S62B) nulls[0]; log = log + b.v; });
        k("b", () -> { S62B b = (S62B) nulls[0]; log = log + b.get(); });
        k("c", () -> { int[] a = (int[]) nulls[0]; log = log + a.length; });
        k("d", () -> { String s = (String) nulls[0]; log = log + s.length(); });
        k("e", () -> { S62B b = (S62B) nulls[0]; b.v = 5; });
        k("f", () -> { List<Integer> l = (List<Integer>) nulls[0]; for (int x : l) { log = log + x; } });
        k("g", () -> { Integer i = (Integer) nulls[0]; log = log + (i + 1); });
        k("h", () -> { String s = (String) nulls[0]; switch (s) { case "a": log = log + "a"; break; default: log = log + "d"; } });
        k("i", () -> { Integer i = (Integer) nulls[0]; log = log + (i == 3); });
        k("j", () -> { Boolean bb = (Boolean) nulls[0]; if (bb) { log = log + "t"; } });
        k("l", () -> { Object o = nulls[0]; log = log + o.hashCode(); });
        k("m", () -> { RuntimeException re = (RuntimeException) nulls[0]; throw re; });
        k("n", () -> { String s = (String) nulls[0]; log = log + s.equals("x"); });
        k("o", () -> { Object o = nulls[0]; log = log + o.toString(); });
        k("p", () -> { int[] a = (int[]) nulls[0]; a[0] = 1; });
        k("q", () -> { Double d = (Double) nulls[0]; double e = d; log = log + e; });
        k("r", () -> { int v = new HashMap<String, Integer>().get("q"); log = log + v; });
        k("s", () -> { log = log + "abc".compareTo((String) nulls[0]); });
        k("t", () -> { Integer i = (Integer) nulls[0]; i++; });
        k("u", () -> { Integer i = (Integer) nulls[0]; log = log + (-i); });
        k("w", () -> { Boolean bb = (Boolean) nulls[0]; log = log + !bb; });
        k("x", () -> { Integer i = (Integer) nulls[0]; log = log + (i < 3); });
        k("y", () -> { String s = (String) nulls[0]; log = log + "x" + s; });
        k("z", () -> { Integer i = (Integer) nulls[0]; Object o = nulls[0]; log = log + (i == null) + (o == i); });
        Main.check("np62", log.equals("a:npe b:npe c:npe d:npe e:npe f:npe g:npe h:npe i:npe j:npe l:npe m:npe n:npe o:npe "
            + "p:npe q:npe r:npe s:npe t:npe u:npe w:npe x:npe xnully:none truetruez:none "));
        log = "";
        k("a", () -> { Boolean bb = (Boolean) nulls[0]; log = log + (bb && true); });
        k("b", () -> { Boolean bb = (Boolean) nulls[0]; log = log + (false || bb); });
        k("c", () -> { Boolean bb = (Boolean) nulls[0]; log = log + (false && bb); });
        Main.check("np62b", log.equals("a:npe b:npe falsec:none "));
        String msg = "";
        try { Object o = nulls[0]; o.toString(); } catch (NullPointerException e) { msg = e.getClass().getName(); }
        Main.check("nm62", msg.equals("java.lang.NullPointerException"));
    }
}

// ===== SECTION 63: a reference cast checks its operand (ClassCastException) =====
//
// (T) e throws ClassCastException when e's class is not T or a subtype (JLS
// 5.5.3, 15.16), and its message names both classes and their modules; null
// passes any cast. A user class' module line names the loader, which differs
// between the source launcher and javac + java, so only its type is checked
// here. Every expected value is java 24.0.2's.
interface S63Shape { double area(); }
class S63Sq implements S63Shape { public double area() { return 1; } }
class S63A {}
class S63B extends S63A {}
class S63 {
    static Object[] vals = {"x", 3, 2.5, new S63B(), new S63Sq(), null, 'c', 7L};
    static String cast(int i, int to) {
        Object o = vals[i];
        try {
            Object r = to == 0 ? (Integer) o : to == 1 ? (String) o : to == 2 ? (S63A) o : to == 3 ? (S63B) o
                : to == 4 ? (S63Shape) o : to == 5 ? (Number) o : (Long) o;
            return "ok";
        } catch (ClassCastException e) { return "cce"; }
    }
    static void run() {
        String grid = "";
        for (int i = 0; i < vals.length; i++) {
            for (int to = 0; to < 7; to++) { grid = grid + (cast(i, to).equals("ok") ? "o" : "x"); }
            grid = grid + " ";
        }
        Main.check("cc63", grid.equals("xoxxxxx oxxxxox xxxxxox xxooxxx xxxxoxx ooooooo xxxxxxx xxxxxoo "));
        String msg = "";
        try { Object o = vals[0]; Integer n = (Integer) o; msg = "" + n; } catch (ClassCastException e) { msg = e.getMessage(); }
        String msg2 = "";
        try { Object o = vals[1]; String t = (String) o; msg2 = t; } catch (ClassCastException e) { msg2 = e.toString(); }
        Main.check("cm63", msg.equals("class java.lang.String cannot be cast to class java.lang.Integer (java.lang.String and java.lang.Integer are in module java.base of loader 'bootstrap')")
            && msg2.equals("java.lang.ClassCastException: class java.lang.Integer cannot be cast to class java.lang.String (java.lang.Integer and java.lang.String are in module java.base of loader 'bootstrap')"));
    }
}

// ===== SECTION 64: java.util.Formatter's exceptions, %a, %h and %#.0f =====
//
// String.format throws a catchable IllegalFormatException subclass - each an
// IllegalArgumentException - for a mismatched argument (its message names the
// argument's class), a missing argument, an unknown conversion, a precision,
// flag or width the conversion does not take, and an invalid code point.
// %a is Double.toHexString's form (a precision rounds the hex significand
// half-even), %h the hash code in hex, and '#' keeps the point of %.0f. Every
// expected value is java 24.0.2's.
class S64 {
    static Object[] vals = {"str", 3.0, 3, 'c', 7L, 0x110000};
    static String fmt(String f, Object v) {
        try { return String.format(f, v); }
        catch (IllegalArgumentException e) { return e.getClass().getName().substring(10) + ":" + e.getMessage(); }
    }
    static void run() {
        Main.check("fx64", fmt("%d", vals[0]).equals("IllegalFormatConversionException:d != java.lang.String")
            && fmt("%d", vals[1]).equals("IllegalFormatConversionException:d != java.lang.Double")
            && fmt("%c", vals[4]).equals("IllegalFormatConversionException:c != java.lang.Long")
            && fmt("%f", vals[2]).equals("IllegalFormatConversionException:f != java.lang.Integer")
            && fmt("%s %s", vals[2]).equals("MissingFormatArgumentException:Format specifier '%s'")
            && fmt("%q", vals[2]).equals("UnknownFormatConversionException:Conversion = 'q'")
            && fmt("%.2d", vals[2]).equals("IllegalFormatPrecisionException:2")
            && fmt("%-d", vals[2]).equals("MissingFormatWidthException:%-d")
            && fmt("%c", vals[5]).equals("IllegalFormatCodePointException:Code point = 0x110000")
            && fmt("%+s", vals[0]).equals("FormatFlagsConversionMismatchException:Conversion = s, Flags = +")
            && fmt("%-05d", vals[2]).equals("IllegalFormatFlagsException:Flags = '-0'")
            && fmt("%.f", vals[1]).equals("UnknownFormatConversionException:Conversion = '.'"));
        double[] ds = {2.5, 0.1, 1e-320, -1.0 / 3, 1.0 + 1.0 / 32, 1.0 + 3.0 / 32};
        String hx = "";
        for (double d : ds) { hx = hx + String.format("%a %.1a %A|", d, d, d); }
        Main.check("fa64", hx.equals("0x1.4p1 0x1.4p1 0X1.4P1|0x1.999999999999ap-4 0x1.ap-4 0X1.999999999999AP-4|"
            + "0x0.00000000007e8p-1022 0x1.0p-1063 0X0.00000000007E8P-1022|-0x1.5555555555555p-2 -0x1.5p-2 -0X1.5555555555555P-2|"
            + "0x1.08p0 0x1.0p0 0X1.08P0|0x1.18p0 0x1.2p0 0X1.18P0|"));
        Main.check("fh64", String.format("%h %H %#.0f %#.0e %.2s", "hi", 255, 2.5, 2.5, null).equals("d01 FF 3. 3.e+00 nu"));
    }
}

// ===== SECTION 65: an Object[] is an Object[], varargs forwarding, Array.newInstance =====
//
// An array created as Object[] (a literal, `new Object[n]`, a varargs
// parameter's collected arguments) is an Object[] whatever it holds, so
// String.format spreads it as its arguments; a primitive array given to an
// Object... parameter is one argument. Arrays.copyOf and toArray(T[]) keep
// the array's own type, through Class.getComponentType and
// java.lang.reflect.Array.newInstance. Every expected value is java 24.0.2's.
class S65 {
    static String g(String fmt, Object... xs) { return String.format(fmt, xs); }
    static String cls(Object... xs) { return xs.getClass().getName() + xs.length; }
    static void run() {
        Object[] one = {3};
        Main.check("va65", String.format("%s", one).equals("3") && g("%d", 8).equals("8") && g("%s|%s", 8, 9).equals("8|9")
            && (cls() + cls(1, 2) + cls(new int[]{4, 5}) + cls(new String[]{"a"})).equals("[Ljava.lang.Object;0[Ljava.lang.Object;2[Ljava.lang.Object;1[Ljava.lang.String;1")
            && new Object[]{"a"}.getClass().getName().equals("[Ljava.lang.Object;") && one.getClass().getName().equals("[Ljava.lang.Object;"));
        String[] s = {"x", "y"};
        int[] ia = {1, 2};
        List<String> l = new ArrayList<>(List.of("p", "q"));
        Object made = Array.newInstance(s.getClass().getComponentType(), 2);
        Main.check("ct65", Arrays.copyOf(s, 3).getClass().getName().equals("[Ljava.lang.String;")
            && Arrays.toString(Arrays.copyOfRange(s, 1, 3)).equals("[y, null]")
            && l.toArray().getClass().getName().equals("[Ljava.lang.Object;")
            && l.toArray(new String[0]).getClass().getName().equals("[Ljava.lang.String;")
            && Arrays.toString(l.toArray(new String[3])).equals("[p, q, null]")
            && (s.getClass().getComponentType().getName() + " " + ia.getClass().getComponentType() + " " + "x".getClass().getComponentType()).equals("java.lang.String int null")
            && made.getClass().getName().equals("[Ljava.lang.String;") && Arrays.toString((Object[]) made).equals("[null, null]"));
    }
}

// ===== SECTION 66: inherited default overloads, functional-interface overloads, thenComparing =====
//
// A class overriding one overload of an interface's default method inherits
// the others (JLS 8.4.8), whatever the receiver's static type. An overload
// whose parameter is a functional interface takes a lambda or an instance of
// that interface, not any reference. A lambda or method reference matches only
// a functional interface of its own arity (JLS 15.12.2.1), which tells
// thenComparing's key-extractor form from its Comparator form. Every expected
// value is java 24.0.2's.
interface S66Shape {
    double area();
    default String describe() { return "area " + area(); }
    default String describe(String unit) { return "area " + area() + " " + unit; }
}
class S66Sq implements S66Shape {
    double s;
    S66Sq(double s) { this.s = s; }
    public double area() { return s * s; }
    public String describe(String unit) { return "square " + area() + unit; }
}
record S66P(String name, int age) {}
class S66 {
    static String kind(S66Shape s) { return "shape"; }
    static String kind(Object o) { return "object"; }
    static void run() {
        S66Shape q = new S66Sq(1.5);
        List<S66Shape> l = List.of(q);
        String d = "";
        for (S66Shape x : l) { d = d + x.describe() + "|" + x.describe("cm"); }
        Object o = q;
        Main.check("df66", (q.describe() + "|" + q.describe("m")).equals("area 2.25|square 2.25m") && d.equals("area 2.25|square 2.25cm")
            && ((S66Shape) o).describe().equals("area 2.25"));
        Main.check("fk66", (kind("s") + kind(q) + kind(() -> 2.0) + kind(3)).equals("objectshapeshapeobject"));
        List<S66P> ps = new ArrayList<>(List.of(new S66P("bo", 30), new S66P("al", 25), new S66P("cy", 30)));
        ps.sort(Comparator.comparing(S66P::age).thenComparing(S66P::name));
        String a = ps.toString();
        ps.sort(Comparator.comparing(S66P::age).reversed().thenComparing(p -> p.name()));
        String b = ps.toString();
        ps.sort(Comparator.comparing(S66P::name).thenComparingInt(S66P::age));
        Main.check("tc66", a.equals("[S66P[name=al, age=25], S66P[name=bo, age=30], S66P[name=cy, age=30]]")
            && b.equals("[S66P[name=bo, age=30], S66P[name=cy, age=30], S66P[name=al, age=25]]")
            && ps.toString().equals("[S66P[name=al, age=25], S66P[name=bo, age=30], S66P[name=cy, age=30]]"));
    }
}

// ===== SECTION 67: Throwable#toString through getMessage, parse of null, List.of's get =====
//
// Throwable#toString prints getLocalizedMessage(), which is getMessage(), so an
// override of either shows. Integer.parseInt(null) throws "Cannot parse null
// string" and Double.parseDouble(null) a NullPointerException. List.of's get
// out of range throws as java's ImmutableCollections do: List12 (one or two
// elements) "Index: i Size: n", ListN ArrayIndexOutOfBoundsException.
// compareToIgnoreCase compares code points where a surrogate pair differs.
// Every expected value is java 24.0.2's.
class S67MyEx extends RuntimeException {
    S67MyEx(String m) { super(m); }
}
class S67Sub extends S67MyEx {
    S67Sub() { super("orig"); }
    public String getMessage() { return "sub-msg"; }
}
class S67Loc extends Exception {
    public String getLocalizedMessage() { return "loc"; }
}
class S67 {
    static String[] strs = {null, "a😀", "a￿"};
    static String oob(List<Integer> l, int i) {
        try { return "" + l.get(i); } catch (IndexOutOfBoundsException e) { return e.getClass().getName().substring(10) + ":" + e.getMessage(); }
    }
    static void run() {
        Main.check("ts67", ("" + new S67Sub()).equals("S67Sub: sub-msg") && new S67Sub().getLocalizedMessage().equals("sub-msg")
            && ("" + new S67Loc()).equals("S67Loc: loc") && ("" + new RuntimeException("r")).equals("java.lang.RuntimeException: r"));
        String p = "";
        try { Integer.parseInt(strs[0]); } catch (NumberFormatException e) { p = p + e.getMessage() + "|"; }
        try { Long.parseLong(strs[0]); } catch (NumberFormatException e) { p = p + e.getMessage() + "|"; }
        try { Double.parseDouble(strs[0]); } catch (NullPointerException e) { p = p + "npe"; }
        Main.check("pn67", p.equals("Cannot parse null string|Cannot parse null string|npe"));
        Main.check("lo67", (oob(List.of(1, 2), 5) + " " + oob(List.of(1), -1) + " " + oob(List.of(1, 2, 3), 3) + " " + oob(List.of(), 0)
            + " " + oob(new ArrayList<>(List.of(1, 2)), 5)).equals("IndexOutOfBoundsException:Index: 5 Size: 2 IndexOutOfBoundsException:Index: -1 Size: 1 "
            + "ArrayIndexOutOfBoundsException:Index 3 out of bounds for length 3 ArrayIndexOutOfBoundsException:Index 0 out of bounds for length 0 "
            + "IndexOutOfBoundsException:Index 5 out of bounds for length 2"));
        Main.check("ci67", strs[1].compareToIgnoreCase(strs[2]) == 62977 && strs[2].compareToIgnoreCase(strs[1]) == -62977
            && strs[1].compareTo(strs[2]) == -10178 && "Ab".compareToIgnoreCase("aC") == -1);
    }
}

// ===== SECTION 68: Objects, Collections, TreeMap, ArrayDeque and the commonly hit members =====
//
// java.util's Objects, Collections, TreeMap (key order, navigation) and
// ArrayDeque; a list iterator's remove, replaceAll and subList; HashMap's
// computeIfPresent; Throwable's cause; String.formatted, regionMatches and
// codePointCount; Character.toChars; Integer's unsigned and bit members;
// Double and Float bits; Math.ulp and IEEEremainder. Every expected value is
// java 24.0.2's.
class S68Ex extends Exception {
    S68Ex(String m, Throwable c) { super(m, c); }
}
class S68 {
    static int[] vs = {1, -1, 0x12345678};
    static void run() {
        Main.check("ob68", Objects.equals(null, null) && !Objects.equals("a", null) && Objects.hash(1, 2) == 994 && Objects.hashCode(null) == 0
            && Objects.toString(null, "d").equals("d") && Objects.requireNonNull("a").equals("a") && Objects.isNull(null));
        List<Integer> l = new ArrayList<>(List.of(3, 1, 2));
        Collections.sort(l);
        String a = l.toString();
        Collections.reverse(l);
        Collections.swap(l, 0, 2);
        Main.check("co68", a.equals("[1, 2, 3]") && l.toString().equals("[1, 2, 3]") && Collections.max(l) == 3 && Collections.min(l) == 1
            && Collections.nCopies(2, "z").toString().equals("[z, z]") && Collections.frequency(List.of(1, 2, 1), 1) == 2
            && Collections.max(List.of("bb", "a", "ccc"), Comparator.comparing(String::length)).equals("ccc")
            && Collections.unmodifiableList(l).toString().equals("[1, 2, 3]"));
        TreeMap<String, Integer> m = new TreeMap<>();
        m.put("b", 2);
        m.put("a", 1);
        m.put("c", 3);
        m.put("b", 20);
        String it = "";
        for (Map.Entry<String, Integer> e : m.entrySet()) { it = it + e.getKey() + e.getValue(); }
        TreeMap<Integer, String> rm = new TreeMap<>(Comparator.reverseOrder());
        rm.put(1, "x");
        rm.put(3, "y");
        rm.put(2, "z");
        Main.check("tm68", m.toString().equals("{a=1, b=20, c=3}") && m.firstKey().equals("a") && m.lastKey().equals("c")
            && m.floorKey("bb").equals("b") && m.ceilingKey("bb").equals("c") && it.equals("a1b20c3") && m.keySet().toString().equals("[a, b, c]")
            && rm.toString().equals("{3=y, 2=z, 1=x}") && rm.higherKey(2) == 1);
        ArrayDeque<Integer> d = new ArrayDeque<>();
        d.push(1);
        d.addLast(2);
        d.offerFirst(0);
        for (int i = 3; i < 20; i++) { d.addLast(i); }
        Deque<String> q = new ArrayDeque<>(List.of("x", "y"));
        q.offer("z");
        Main.check("dq68", d.size() == 20 && d.pop() == 0 && d.pollLast() == 19 && d.peek() == 1 && q.toString().equals("[x, y, z]") && q.poll().equals("x"));
        List<Integer> r = new ArrayList<>(List.of(1, 2, 3, 4, 5));
        Iterator<Integer> ri = r.iterator();
        while (ri.hasNext()) { if (ri.next() % 2 == 0) { ri.remove(); } }
        r.replaceAll(x -> x * 10);
        List<Integer> sub = r.subList(1, 3);
        sub.set(0, 99);
        Map<String, Integer> hm = new HashMap<>();
        hm.put("a", 1);
        hm.computeIfPresent("a", (k, v) -> v + 1);
        hm.computeIfPresent("b", (k, v) -> 9);
        Main.check("li68", r.toString().equals("[10, 99, 50]") && sub.toString().equals("[99, 50]") && hm.toString().equals("{a=2}"));
        RuntimeException c = new RuntimeException("outer", new IllegalStateException("inner"));
        Exception c1 = new Exception(new IllegalStateException("in"));
        S68Ex me = new S68Ex("m", c1);
        Main.check("ca68", c.getCause().getMessage().equals("inner") && c1.getMessage().equals("java.lang.IllegalStateException: in")
            && me.getCause().getCause().getMessage().equals("in") && new Exception("x").getCause() == null);
        Main.check("st68", "%d!".formatted(5).equals("5!") && "Hello".regionMatches(true, 1, "ELL", 0, 3) && !"hello".regionMatches(1, "he", 0, 2)
            && "a😀b".codePointCount(0, 4) == 3 && new String(Character.toChars(0x1F600)).equals("😀")
            && Character.toChars(65)[0] == 'A');
        String u = "";
        for (int v : vs) {
            u = u + Integer.reverse(v) + " " + Integer.reverseBytes(v) + " " + Integer.toUnsignedString(v) + " " + Integer.rotateLeft(v, 33)
                + " " + Integer.rotateRight(v, -7) + " " + Integer.divideUnsigned(v, 3) + " " + Integer.remainderUnsigned(v, 7)
                + " " + Integer.compareUnsigned(v, 5) + " " + Integer.toUnsignedLong(v) + "|";
        }
        Main.check("un68", u.equals("-2147483648 16777216 1 2 128 0 1 -1 1|-1 -1 4294967295 -1 -1 1431655765 3 1 4294967295|"
            + "510274632 2018915346 305419896 610839792 439041033 101806632 5 1 305419896|")
            && Integer.parseUnsignedInt("4294967295") == -1);
        Main.check("bt68", Double.doubleToLongBits(1.5) == 4609434218613702656L && Double.hashCode(1.5) == 1073217536
            && Float.floatToIntBits(1.5f) == 1069547520 && Double.doubleToLongBits(-0.0) == Long.MIN_VALUE
            && Math.ulp(1.0) == 2.220446049250313E-16 && Math.ulp(1e-310) == 4.9E-324 && Math.ulp(1f) == 1.1920929E-7f
            && Math.IEEEremainder(5.0, 3.0) == -1.0 && Math.IEEEremainder(7.0, 2.0) == -1.0 && Math.IEEEremainder(-5.5, 3.0) == 0.5);
        long[] ls = {-1L, Long.MAX_VALUE, -987654321098765L};
        String mh = "";
        for (long x : ls) { for (long y : ls) { mh = mh + Math.multiplyHigh(x, y) + " "; } }
        String two = "a😀b";
        Main.check("sm68", mh.equals("0 -1 0 -1 4611686018427387903 -493827160549383 0 -493827160549383 52879849912 ")
            && two.chars().count() == 4 && two.codePoints().count() == 3 && "abc".chars().sum() == 294
            && "x\ny\r\nz\r".lines().collect(Collectors.toList()).toString().equals("[x, y, z]"));
    }
}

// ===== SECTION 69: regex exceptions, package-qualified names, class literals, varargs by static type =====
//
// A bad pattern throws the catchable java.util.regex.PatternSyntaxException (an
// IllegalArgumentException), a bad replacement IllegalArgumentException or
// IndexOutOfBoundsException; a JDK class named through its package
// (java.lang.Math.max, java.lang.reflect.Array.newInstance) is the class, a
// catch may name its type that way too; a class literal is the Class object
// getClass() answers; a variable arity call wraps its last argument by its
// STATIC type, so (Object) arr is one element; initCause sets the cause.
// Every expected value is java 24.0.2's.
class S69 {
    interface Sh { }
    static class Bx { }
    static String cls(Object... a) { return a.getClass().getName() + a.length; }
    static String over(Object o) { return "o"; }
    static String over(Object... a) { return "v" + a.length; }
    static <T> String gen(T x) { return cls(x); }
    String inst(Object... a) { return "i" + a.length; }
    static void run() {
        String[] pats = {"(", "[a", "a", "*"};
        String rx = "";
        for (String p : pats) {
            try { rx = rx + "a".matches(p); } catch (java.util.regex.PatternSyntaxException e) { rx = rx + "P:" + e.getClass().getName(); }
        }
        String[] reps = {"$9", "\\", "$", "${nm}", "$x", "<$0>"};
        for (String r : reps) {
            try { rx = rx + "|" + "a".replaceAll("a", r); }
            catch (IndexOutOfBoundsException e) { rx = rx + "|O:" + e.getMessage(); }
            catch (IllegalArgumentException e) { rx = rx + "|I:" + e.getMessage(); }
        }
        boolean isPse = false;
        try { "x(y".split("("); } catch (IllegalArgumentException e) { isPse = e instanceof java.util.regex.PatternSyntaxException; }
        String ae = "";
        try { int z = 0; ae = "" + (1 / z); } catch (java.lang.ArithmeticException e) { ae = e.getMessage(); }
        Main.check("rx69", rx.equals("P:java.util.regex.PatternSyntaxExceptionP:java.util.regex.PatternSyntaxExceptiontrueP:java.util.regex.PatternSyntaxException"
            + "|O:No group 9|I:character to be escaped is missing|I:Illegal group reference: group index is missing"
            + "|I:No group with name {nm}|I:Illegal group reference|<a>") && isPse && ae.equals("/ by zero"));
        Main.check("pq69", java.lang.Math.max(3, 4) == 4 && java.lang.Integer.parseInt("42") == 42 && java.lang.String.valueOf(7).equals("7")
            && java.lang.Integer.MAX_VALUE == 2147483647 && java.util.Objects.equals("a", "a") && java.lang.Character.isDigit('7'));
        Object made = java.lang.reflect.Array.newInstance(String.class, 2);
        String cl = int.class + " " + void.class + " " + String.class + " " + int[][].class + " " + Bx.class + " " + Bx[].class
            + " " + Sh.class + " " + RuntimeException.class.getSimpleName() + " " + Bx.class.getSimpleName();
        Main.check("cl69", cl.equals("int void class java.lang.String class [[I class S69$Bx class [LS69$Bx; interface S69$Sh RuntimeException Bx")
            && "x".getClass() == String.class && new Bx().getClass() == Bx.class && int.class != Integer.class
            && Integer.valueOf(3).getClass() == Integer.class && String[].class.getComponentType() == String.class
            && made.getClass().getName().equals("[Ljava.lang.String;") && java.lang.reflect.Array.getLength(made) == 2);
        String[] arr = {"x", "y"};
        Object o = arr;
        String va = cls((Object) arr) + cls((Object[]) arr) + cls(o) + S69.cls(o) + new S69().inst(o) + gen(arr) + over(o) + cls(null, null);
        Main.check("va69", va.equals("[Ljava.lang.Object;1[Ljava.lang.String;2[Ljava.lang.Object;1[Ljava.lang.Object;1i1[Ljava.lang.Object;1o[Ljava.lang.Object;2"));
        RuntimeException a = new RuntimeException("a");
        IllegalStateException b = new IllegalStateException("b");
        Main.check("ic69", a.initCause(b) == a && a.getCause() == b);
    }
}

// ===== SECTION 70: PatternSyntaxException's message =====
//
// getMessage is the description, " near index N" when the error has a position,
// the pattern, and a caret line under N when N is inside the pattern. Every
// expected value is java 24.0.2's.
class S70 {
    static void run() {
        String[] pats = {"(", "a(b", "a)", ")", "[a", "[b-a]", "a\\", "*", "a**", "a{3,2}", "(?<n", "\\k<zz>", "(?z)", "\\xZZ", "\\Q(\\E("};
        String got = "";
        for (String p : pats) {
            try { "a".matches(p); got = got + "ok;"; }
            catch (java.util.regex.PatternSyntaxException e) { got = got + e.getMessage().replace("\n", "/") + ";"; }
        }
        Main.check("pm70", got.equals("Unclosed group near index 1/(;Unclosed group near index 3/a(b;"
            + "Unmatched closing ')' near index 0/a)/^;Unmatched closing ')'/);Unclosed character class near index 1/[a/ ^;"
            + "Illegal character range near index 3/[b-a]/   ^;Unescaped trailing backslash near index 2/a\\;"
            + "Dangling meta character '*' near index 0/*/^;Dangling meta character '*' near index 2/a**/  ^;"
            + "Illegal repetition range near index 5/a{3,2}/     ^;named capturing group is missing trailing '>' near index 4/(?<n;"
            + "named capturing group <zz> does not exist near index 5/\\k<zz>/     ^;Unknown inline modifier near index 2/(?z)/  ^;"
            + "Illegal hexadecimal escape sequence near index 2/\\xZZ/  ^;Unclosed group near index 3/\\Q(\\E(/   ^;"));
    }
}

// ===== SECTION 71: LinkedHashMap, LinkedHashSet, TreeSet, Map.of, the further Collectors and stream sources =====
//
// A LinkedHashMap / LinkedHashSet iterates in insertion order, a TreeSet in
// its order; Map.of is unmodifiable; the Collectors beyond toList / joining /
// groupingBy; Stream.iterate, concat, flatMap, takeWhile / dropWhile; a
// method reference over an interface's method is unbound; the static
// hashCode of the wrappers, and a byte or short box hashes to its value.
// Every expected value is java 24.0.2's.
class S71 {
    static void run() {
        Map<String, Integer> lm = new LinkedHashMap<>();
        String[] ks = {"zeta", "alpha", "mid", "beta", "alpha"};
        for (int i = 0; i < ks.length; i++) { lm.put(ks[i], i); }
        lm.remove("mid");
        lm.put("mid", 9);
        lm.merge("zeta", 100, Integer::sum);
        StringBuilder fe = new StringBuilder();
        lm.forEach((k, v) -> fe.append(k.charAt(0)));
        Set<Integer> ls = new LinkedHashSet<>(List.of(50, 3, 17));
        ls.add(3);
        ls.add(-1);
        TreeSet<String> ts = new TreeSet<>(List.of("pear", "apple", "fig"));
        ts.add("banana");
        String col = lm + " " + lm.keySet() + " " + fe + " " + ls + " " + ts + " " + ts.first() + ts.ceiling("c") + ts.lower("apple")
            + " " + ts.pollLast() + " " + new TreeMap<>(Map.of(3, "c", 1, "a")) + " " + Map.entry("a", 2);
        Main.check("lh71", col.equals("{zeta=100, alpha=4, beta=3, mid=9} [zeta, alpha, beta, mid] zabm [50, 3, 17, -1]"
            + " [apple, banana, fig, pear] applefignull pear {1=a, 3=c} a=2"));
        String ex = "";
        try { Map.of("k", 1).put("z", 2); } catch (UnsupportedOperationException e) { ex = ex + "U"; }
        try { Map.of("a", 1, "a", 2); } catch (IllegalArgumentException e) { ex = ex + e.getMessage(); }
        try { Stream.of("a", "b", "a").collect(Collectors.toMap(x -> x, x -> 1)); } catch (IllegalStateException e) { ex = ex + "|" + e.getMessage(); }
        Main.check("mo71", ex.equals("Uduplicate key: a|Duplicate key a (attempted merging values 1 and 1)"));
        List<String> w = List.of("apple", "bob", "cat", "avocado");
        String cs = w.stream().collect(Collectors.partitioningBy(x -> x.length() > 3)) + " "
            + w.stream().collect(Collectors.groupingBy(String::length, TreeMap::new, Collectors.counting())) + " "
            + w.stream().collect(Collectors.toMap(x -> x.charAt(0), x -> 1, Integer::sum)) + " "
            + w.stream().collect(Collectors.groupingBy(x -> x.charAt(0), Collectors.mapping(x -> x.length(), Collectors.toList()))) + " "
            + w.stream().collect(Collectors.averagingInt(String::length)) + " " + w.stream().collect(Collectors.summingInt(String::length)) + " "
            + w.stream().collect(Collectors.maxBy(Comparator.naturalOrder())) + " " + w.stream().collect(Collectors.reducing("", (a, b) -> a + b.charAt(0))) + " "
            + w.stream().collect(Collectors.collectingAndThen(Collectors.toList(), List::size)) + " "
            + w.stream().mapToInt(String::length).summaryStatistics() + " " + Stream.of(3, 1, 2).collect(Collectors.toCollection(TreeSet::new));
        Main.check("co71", cs.equals("{false=[bob, cat], true=[apple, avocado]} {3=2, 5=1, 7=1} {a=2, b=1, c=1} {a=[5, 7], b=[3], c=[3]}"
            + " 4.5 18 Optional[cat] abca 4 IntSummaryStatistics{count=4, sum=18, min=3, average=4.500000, max=7} [1, 2, 3]"));
        String st = Stream.iterate(1, x -> x * 2).limit(5).toList() + " " + Stream.iterate(1, x -> x <= 20, x -> x * 3).toList() + " "
            + Stream.of(List.of(1, 2), List.of(3)).flatMap(List::stream).toList() + " " + Stream.concat(Stream.of("a"), Stream.of("b")).toList() + " "
            + Stream.of(1, 2, 3, 1).takeWhile(x -> x < 3).toList() + Stream.of(1, 2, 3, 1).dropWhile(x -> x < 3).toList() + " "
            + Arrays.toString(Stream.of("q", "r").toArray(String[]::new)) + " " + IntStream.iterate(5, x -> x - 2).limit(3).sum();
        Main.check("st71", st.equals("[1, 2, 4, 8, 16] [1, 3, 9] [1, 2, 3] [a, b] [1, 2][3, 1] [q, r] 9"));
        List<Integer> nums = new ArrayList<>(List.of(1, 2, 3));
        nums.addAll(1, List.of(7, 8));
        ListIterator<Integer> li = nums.listIterator(nums.size());
        String lr = "";
        while (li.hasPrevious()) {
            int v = li.previous();
            lr = lr + v;
            if (v == 8) { li.set(80); }
        }
        li.add(0);
        List<String> fl = new ArrayList<>(List.of("x", "y"));
        Collections.fill(fl, "z");
        Integer[] desc = {2, 9, 4};
        Arrays.sort(desc, Collections.reverseOrder());
        String ut = lr + nums + li.nextIndex() + " " + fl + " " + Arrays.toString(desc) + " " + Collections.binarySearch(List.of(1, 3, 5), 4)
            + " " + Arrays.deepToString(new Object[]{1, new int[][]{{2}, {3, 4}}, null}) + " " + Optional.of(3).flatMap(x -> Optional.of(x + 1))
            + " " + Optional.empty().or(() -> Optional.of("o")) + " " + Optional.of("v").orElseThrow(IllegalStateException::new);
        Main.check("ut71", ut.equals("32871[0, 1, 7, 80, 2, 3]1 [z, z] [9, 4, 2] -3 [1, [[2], [3, 4]], null] Optional[4] Optional[o] v"));
        Byte nb = (byte) -1;
        Short ns = (short) -300;
        String hc = nb.hashCode() + " " + ns.hashCode() + " " + Integer.hashCode(-7) + " " + Long.hashCode(1L << 40) + " " + Short.hashCode((short) -2)
            + " " + Byte.hashCode((byte) -3) + " " + Character.hashCode('x') + " " + Boolean.hashCode(false) + " " + Character.forDigit(11, 16)
            + (int) Character.forDigit(7, 2) + " " + Character.toString(97) + Character.toString('b');
        Main.check("hc71", hc.equals("-1 -300 -7 256 -2 -3 120 1237 b0 ab"));
    }
}

// ===== SECTION 72: PriorityQueue, LinkedList, Stack, StringJoiner, Random, decode and the comparator helpers =====
//
// A PriorityQueue is java's binary heap, so its array order (toString) is
// java's; Random(seed) is java's generator; Map.Entry's comparators chain;
// a lambda in a local of a package-qualified interface type is an instance of
// it; String.CASE_INSENSITIVE_ORDER is a Comparator; Integer.decode reads the
// 0x / # / 0 prefixes; parseInt takes any script's decimal digits.
// Every expected value is java 24.0.2's.
class S72 {
    static void run() {
        PriorityQueue<Integer> pq = new PriorityQueue<>(List.of(5, 1, 4, 2, 3));
        pq.add(0);
        String heap = pq.toString();
        String drained = "";
        while (!pq.isEmpty()) { drained = drained + pq.poll(); }
        PriorityQueue<String> pq2 = new PriorityQueue<>(Comparator.comparing(String::length).reversed());
        pq2.addAll(List.of("aa", "b", "cccc", "ddd"));
        LinkedList<Integer> ll = new LinkedList<>(List.of(1, 2, 3));
        ll.addFirst(0);
        ll.removeFirst();
        ll.add(1, 9);
        String lle = "";
        try { ll.get(7); } catch (IndexOutOfBoundsException e) { lle = e.getMessage(); }
        Stack<String> stk = new Stack<>();
        stk.push("a");
        stk.push("b");
        String se = "";
        try { new Stack<Integer>().pop(); } catch (EmptyStackException e) { se = "E" + e.getMessage(); }
        String q = heap + drained + " " + pq2.poll() + pq2.peek() + " " + ll + ll.peekLast() + ll.pop() + " " + lle + " " + stk.peek() + stk.search("a")
            + stk.pop() + stk + se + " " + new StringJoiner(", ", "{", "}").add("a").add("b") + new StringJoiner("-").setEmptyValue("E");
        Main.check("pq72", q.equals("[0, 2, 1, 5, 3, 4]012345 ccccddd [1, 9, 2, 3]31 Index: 7, Size: 4 b2b[a]Enull {a, b}E"));
        Random r = new Random(42);
        String rs = r.nextInt(100) + " " + r.nextInt() + " " + r.nextLong() + " " + r.nextDouble() + " " + r.nextBoolean() + " " + r.nextInt(5, 10);
        Main.check("rn72", rs.equals("30 234785527 -5843495416241995736 0.30871945533265976 false 8"));
        Map<String, Integer> m = new HashMap<>(Map.of("a", 2, "b", 1, "c", 2));
        java.util.Comparator<Integer> qc = (x, y) -> x - y;
        String cmp = m.entrySet().stream().sorted(Map.Entry.<String, Integer>comparingByValue().reversed().thenComparing(Map.Entry.comparingByKey()))
                .map(Map.Entry::getKey).toList() + " " + m.entrySet().stream().sorted(Map.Entry.comparingByValue(Comparator.reverseOrder())).findFirst().get().getValue()
            + " " + qc.reversed().compare(1, 2);
        String[] strs = {"b", "C", "a", "B"};
        Arrays.sort(strs, String.CASE_INSENSITIVE_ORDER);
        cmp = cmp + " " + Arrays.toString(strs) + String.CASE_INSENSITIVE_ORDER.reversed().compare("a", "B");
        Main.check("cm72", cmp.equals("[a, c, b] 2 1 [a, b, B, C]1"));
        String dec = Integer.decode("0x1F") + " " + Integer.decode("-010") + " " + Long.decode("#ff") + " " + Integer.decode("+7") + " " + Integer.parseInt("٣٤");
        try { Integer.decode("-+1"); } catch (NumberFormatException e) { dec = dec + " " + e.getMessage(); }
        try { Integer.decode(""); } catch (NumberFormatException e) { dec = dec + " " + e.getMessage(); }
        Main.check("dc72", dec.equals("31 -8 255 7 34 Sign character in wrong position Zero length string"));
        int[] a = {5, 2, 8, 1};
        List<Integer> al = new ArrayList<>(List.of(4, 5));
        String misc = Arrays.stream(a, 1, 3).sum() + " " + Arrays.compare(new int[]{1, 2}, new int[]{1, 3}) + Arrays.compare(new int[]{1}, new int[]{1, 0})
            + " " + Arrays.mismatch(new int[]{1, 2}, new int[]{1, 3}) + Arrays.mismatch(new int[]{1}, new int[]{1}) + " " + Objects.requireNonNullElseGet(null, () -> "g")
            + " " + new StringBuilder().appendCodePoint(128512).length() + " " + "abc".contentEquals(new StringBuilder("abc")) + " " + al.getFirst() + al.getLast()
            + al.removeFirst() + al;
        Main.check("mi72", misc.equals("10 -1-1 1-1 g 2 true 454[5]"));
    }
}

// ===== SECTION 73: Collections' shuffle, rotate and views, Set.of, Character.equals =====
//
// shuffle(list, rnd) is java's swap walk over java's Random, rotate java's
// cycle walk; the unmodifiable and singleton views refuse changes; Set.of
// rejects a repeat; Character.equals is true only for a Character.
// Every expected value is java 24.0.2's.
class S73 {
    static void run() {
        List<Integer> l = new ArrayList<>(List.of(1, 2, 3, 4, 5, 6, 7, 8));
        Collections.shuffle(l, new Random(7));
        String sh = l.toString();
        Collections.rotate(l, 2);
        String ex = "";
        try { Collections.unmodifiableMap(new HashMap<String, Integer>()).put("a", 1); } catch (UnsupportedOperationException e) { ex = ex + "U"; }
        try { Collections.singleton(1).add(2); } catch (UnsupportedOperationException e) { ex = ex + "S"; }
        try { Set.of(1, 1); } catch (IllegalArgumentException e) { ex = ex + e.getMessage(); }
        String c = sh + l + " " + Collections.unmodifiableMap(new TreeMap<>(Map.of("b", 2, "a", 1))) + Collections.emptyMap() + Collections.emptySet()
            + Collections.singletonMap("k", "v") + Collections.unmodifiableSet(new TreeSet<>(Set.of(2, 1))) + " " + ex + " " + List.copyOf(l).size()
            + Collections.disjoint(List.of(1, 2), List.of(3)) + Collections.indexOfSubList(List.of(1, 2, 3, 2, 3), List.of(2, 3));
        Main.check("cl73", c.equals("[1, 3, 7, 2, 5, 4, 8, 6][8, 6, 1, 3, 7, 2, 5, 4] {a=1, b=2}{}[]{k=v}[1, 2] USduplicate element: 1 8true1"));
        Object ch = 'a';
        Object str = "a";
        Main.check("ce73", !ch.equals(str) && !ch.equals(97) && ch.equals('a') && new HashSet<>(Arrays.asList("1", '1')).size() == 2);
    }
}

// ===== SECTION 74: the java.util.regex dialect =====
//
// A { is always a bound: one that does not read as {n}, {n,} or {n,m} with
// ints is an error, while a bound where an atom belongs quantifies nothing;
// a u escape wants four hex digits, \p{..} a known property, a group name a Latin
// letter first, a range a character at both ends; [..] classes nest and
// intersect with &&; \nn takes as many digits as name a group, and a
// reference to a group that did not take part FAILS; lookbehind works.
// Every expected value is java 24.0.2's.
class S74 {
    static String m(String p, String s) {
        try { return "" + s.matches(p); }
        catch (java.util.regex.PatternSyntaxException e) { String g = e.getMessage(); return g.substring(0, g.indexOf("\n")); }
    }
    static String all(String[] ps, String s) {
        String out = "";
        for (String p : ps) { out = out + m(p, s) + ";"; }
        return out;
    }
    static void run() {
        String[] bad = {"{", "a{", "a{x}", "a{2", "x{1,2}{", "a{,2}", "a{99999999999}", "a{1,2147483648}", "a{3,2}"};
        Main.check("bd74", all(bad, "a").equals("Illegal repetition near index 1;Illegal repetition near index 2;Illegal repetition near index 2;"
            + "Unclosed counted closure near index 3;Illegal repetition near index 7;Illegal repetition near index 2;"
            + "Illegal repetition range near index 11;Illegal repetition range near index 13;Illegal repetition range near index 5;"));
        String bounds = m("a{1}{2}", "a") + m("a*{2}", "aa") + m("{1}", "") + m("a{2147483647}", "a") + m("(a){1,2147483647}", "a") + m("(ab){2,3}?c", "ababc");
        Main.check("bn74", bounds.equals("truetruetruefalsetruetrue"));
        String[] esc = {"\\u12", "\\p{Foo}", "\\c", "\\08", "\\x4", "\\y", "[\\b]", "\\k", "\\p{", "\\x{110000}"};
        Main.check("es74", all(esc, "a").equals("Illegal Unicode escape sequence near index 4;Unknown character property name {Foo} near index 6;"
            + "Illegal control escape sequence near index 1;Illegal octal escape sequence near index 2;Illegal hexadecimal escape sequence near index 3;"
            + "Illegal/unsupported escape sequence near index 1;Illegal/unsupported escape sequence near index 2;"
            + "\\k is not followed by '<' for named capturing group near index 2;Unclosed character family near index 3;"
            + "Hexadecimal codepoint is too big near index 8;"));
        String good = m("\\cA", "\u0001") + m("\\0101", "A") + m("\\x{41}", "A") + m("\\h", " ") + m("\\v", " ") + m("\\H", " ")
            + m("\\p{Lower}+", "abc") + m("\\P{Alpha}", "A") + m("\\pL", "é") + m("\\p{Lu}", "a") + m("\\p{IsAlphabetic}", "x") + "a\r\nb\nc".split("\\R").length;
        Main.check("eg74", good.equals("truetruetruetruetruefalsetruefalsetruefalsetrue3"));
        String[] grp = {"(?<1a>x)", "(?<a>x)(?<a>y)", "(?<a_b>x)", "[a-", "[a-\\d]", "[b-a]", "[&&]"};
        Main.check("gr74", all(grp, "x").equals("capturing group name does not start with a Latin letter near index 3;"
            + "Named capturing group <a> is already defined near index 11;named capturing group is missing trailing '>' near index 4;"
            + "Illegal character range near index 3;Illegal character range near index 4;Illegal character range near index 3;Bad class syntax near index 2;"));
        String cls = m("[[:alpha:]]", ":") + m("[[:alpha:]]", "x") + m("[a-z&&[^aeiou]]", "e") + m("[a-z&&[^aeiou]]", "f") + m("[^a-c&&b]", "a")
            + m("[^a-c&&b]", "b") + m("[a&&&b]", "&") + m("[a-[b]]", "-") + m("[&&a]", "a") + m("[\\w&&[^\\d]]", "5");
        Main.check("cc74", cls.equals("truefalsefalsetruetruefalsetruetruetruefalse"));
        String refs = m("\\2", "") + m("(a)\\11", "aa1") + m("(b)?a\\1", "a") + m("(a)(b)(c)(d)(e)(f)(g)(h)(i)(j)\\10", "abcdefghijj") + m("\\1(a)", "a");
        Main.check("br74", refs.equals("falsetruefalsetruefalse"));
        String behind = "x1y2".replaceAll("(?<=x)\\d", "#") + "aXbX".replaceAll("(?<!a)X", "_") + m("aa(?<=a+)b", "aab") + m("ab(?<=(?<=a)b)", "ab")
            + m("(?<=a{1,3})b", "b") + "a1b22c333d".split("\\d{1,2}").length;
        Main.check("lb74", behind.equals("x#y2aXb_truetruefalse5"));
    }
}

// ===== SECTION 75: Math.clamp / nextUp, Long's unsigned family, bit patterns, indent, TreeMap ranges, read-only views =====
//
// Math.clamp picks its overload from the argument types and rejects min > max
// (in Double.compare's order) or a NaN bound; nextUp / nextDown step one ulp,
// a negative power of two by half of one. Long reads its bits unsigned for
// toUnsignedString, parseUnsignedLong, divideUnsigned and friends. The bit
// pattern readers are doubleToLongBits' and floatToIntBits' inverses (a NaN
// float's bits are 0x7fc00000). indent and stripIndent work on lines().
// headMap / tailMap / subMap are live views of a key range, and an
// unmodifiable map's keySet, values and entries refuse every change.
// Every expected value is java 24.0.2's.
class S75 {
    static String iae(double v, double lo, double hi) {
        try { double r = Math.clamp(v, lo, hi); return "" + r; } catch (IllegalArgumentException e) { return e.getMessage(); }
    }
    static void run() {
        long[] lv = {5L, -5L, 3000000000L, Long.MIN_VALUE};
        String cl = "";
        for (long v : lv) {
            int c = Math.clamp(v, 0, 10);
            long l = Math.clamp(v, -7L, 2999999999L);
            double d = Math.clamp((double) v, -1.5, 1.5);
            float f = Math.clamp((float) v, -1.5f, 1.5f);
            cl = cl + c + "/" + l + "/" + d + "/" + f + " ";
        }
        String bad = "";
        try { int b = Math.clamp(5L, 3, 2); bad = bad + b; } catch (IllegalArgumentException e) { bad = bad + e.getMessage(); }
        Main.check("cl75", (cl + bad + "|" + iae(5.0, 3.0, 2.0) + "|" + iae(5.0, Double.NaN, 2.0) + "|" + iae(5.0, 0.0, -0.0) + "|" + iae(Double.NaN, 0.0, 1.0))
            .equals("5/5/1.5/1.5 0/-5/-1.5/-1.5 10/2999999999/1.5/1.5 0/-7/-1.5/-1.5 3 > 2|3.0 > 2.0|min is NaN|0.0 > -0.0|NaN"));
        double[] dv = {1.5, -0.0, Double.MAX_VALUE, Double.MIN_VALUE, -Double.MIN_VALUE, -1.0, Double.NEGATIVE_INFINITY, 1e-310};
        String nx = "";
        for (double d : dv) { nx = nx + Math.nextUp(d) + "," + Math.nextDown(d) + " "; }
        float fu = Math.nextUp(1.0f);
        float fd = Math.nextDown(-Float.MIN_VALUE);
        float fm = Math.nextUp(Float.MAX_VALUE);
        Main.check("nx75", (nx + fu + " " + fd + " " + fm).equals("1.5000000000000002,1.4999999999999998 4.9E-324,-4.9E-324 Infinity,1.7976931348623155E308 "
            + "9.9E-324,0.0 -0.0,-9.9E-324 -0.9999999999999999,-1.0000000000000002 -1.7976931348623157E308,-Infinity "
            + "1.00000000000005E-310,9.9999999999995E-311 1.0000001 -2.8E-45 Infinity"));
        long[] uv = {-1L, Long.MIN_VALUE, 1234567890123L, -1234567890123L};
        String un = "";
        for (long a : uv) {
            un = un + Long.toUnsignedString(a) + " " + Long.toUnsignedString(a, 36) + " " + Long.divideUnsigned(a, 1000000007L) + " "
                + Long.remainderUnsigned(a, -3L) + " " + Long.compareUnsigned(a, 5L) + " " + Long.reverse(a) + " " + Long.reverseBytes(a) + " "
                + Long.rotateLeft(a, 37) + " " + Long.rotateRight(a, -3) + ";";
        }
        Main.check("un75", un.equals("18446744073709551615 3w5e11264sgsf 18446743944 2 1 -1 -1 -1 -1;"
            + "9223372036854775808 1y2p0ij32e8e8 9223371972 -9223372036854775808 1 1 128 68719476736 4;"
            + "1234567890123 fr5hugnf 1234 1234567890123 1 -3233338727304527872 -3817650120833302528 4566818659749602286 9876543120984;"
            + "18446742839141661493 3w5e0lb0my051 18446742710 -1234567890123 1 -5990033309550247937 3889707714871230463 -4566818522310648815 -9876543120977;"));
        String[] us = {"18446744073709551615", "18446744073709551616", "+12", "-1", "+", "12a"};
        String pu = "";
        for (String s : us) {
            try { pu = pu + Long.parseUnsignedLong(s) + ";"; } catch (NumberFormatException e) { pu = pu + e.getMessage() + ";"; }
        }
        long hx = Long.parseUnsignedLong("ffffffffffffffff", 16);
        Main.check("pu75", (pu + hx).equals("-1;String value 18446744073709551616 exceeds range of unsigned long.;12;"
            + "Illegal leading minus sign on unsigned string -1.;For input string: \"+\";For input string: \"12a\";-1"));
        long[] bl = {Long.MIN_VALUE, 0x7ff0000000000000L, 0x7ff8000000000000L, 1L, 0xbff8000000000000L, 0x000fffffffffffffL};
        String bits = "";
        for (long b : bl) { bits = bits + Double.longBitsToDouble(b) + " "; }
        int[] fb = {0x80000000, 0x7f800000, 0x7fc00000, 1, 0x7f7fffff, 0x00800000};
        for (int b : fb) { bits = bits + Float.intBitsToFloat(b) + " "; }
        bits = bits + Float.floatToIntBits(Float.NaN) + " " + Double.toHexString(1.0) + " " + Double.toHexString(-Double.MIN_VALUE) + " "
            + Double.toHexString(0.1) + " " + Double.toHexString(Double.NaN);
        Main.check("bt75", bits.equals("-0.0 Infinity NaN 4.9E-324 -1.5 2.225073858507201E-308 -0.0 Infinity NaN 1.4E-45 3.4028235E38 1.1754944E-38 "
            + "2143289344 0x1.0p0 -0x0.0000000000001p-1022 0x1.999999999999ap-4 NaN"));
        String in = "a\n  b".indent(2) + "|" + "  a\n    b\r\nc".indent(-1) + "|" + "".indent(3) + "|" + "   a\n     b\n   c   ".stripIndent()
            + "|" + "  x\n  y\n".stripIndent() + "|" + "\ta\n b".stripIndent();
        Main.check("in75", in.equals("  a\n    b\n| a\n   b\nc\n||a\n  b\nc|  x\n  y\n|a\nb"));
        TreeMap<String, Integer> t = new TreeMap<>();
        String[] ks = {"d", "b", "f", "a", "c", "e"};
        for (int i = 0; i < ks.length; i++) { t.put(ks[i], i); }
        Map<String, Integer> h = t.headMap("c");
        Map<String, Integer> s = t.subMap("b", "e");
        Map<String, Integer> ti = t.tailMap("c", false);
        String tv = h + " " + s + " " + ti + " " + t.headMap("c", true) + " " + t.subMap("b", false, "e", true);
        t.put("bb", 9);
        t.remove("d");
        s.put("cc", 7);
        String ex = "";
        try { s.put("z", 1); } catch (IllegalArgumentException e) { ex = ex + e.getMessage(); }
        try { t.subMap("e", "b"); } catch (IllegalArgumentException e) { ex = ex + "|" + e.getMessage(); }
        s.keySet().remove("c");
        h.clear();
        Main.check("tm75", (tv + " " + t + " " + s + " " + s.size() + " " + h.isEmpty() + " " + ex + " " + t.tailMap("bb").keySet() + t.tailMap("cc").values())
            .equals("{a=3, b=1} {b=1, c=4, d=0} {d=0, e=5, f=2} {a=3, b=1, c=4} {c=4, d=0, e=5} {cc=7, e=5, f=2} {cc=7} 1 true "
                + "key out of range|fromKey > toKey [cc, e, f][7, 5, 2]"));
        Map<String, Integer> um = Collections.unmodifiableMap(new HashMap<>(Map.of("a", 1, "b", 2)));
        String ro = "";
        try { um.keySet().remove("a"); } catch (UnsupportedOperationException e) { ro = ro + "K"; }
        try { um.values().clear(); } catch (UnsupportedOperationException e) { ro = ro + "V"; }
        try { Iterator<String> it = um.keySet().iterator(); it.next(); it.remove(); } catch (UnsupportedOperationException e) { ro = ro + "I"; }
        try { for (Map.Entry<String, Integer> e : um.entrySet()) { e.setValue(5); } } catch (UnsupportedOperationException e) { ro = ro + "S"; }
        try { Map.of("x", 1).keySet().remove("x"); } catch (UnsupportedOperationException e) { ro = ro + "M"; }
        Object one = 1;
        Object oneL = 1L;
        Map<Object, String> hm = new HashMap<>();
        hm.put(1, "i");
        hm.put(1L, "l");
        Main.check("ro75", (ro + " " + um + " " + one.equals(oneL) + hm.size() + hm.get(1) + hm.get(1L) + " "
            + String.CASE_INSENSITIVE_ORDER.reversed().compare("a", "B")).equals("KVISM {a=1, b=2} false2il 1"));
    }
}

// ===== SECTION 76: inline flags, atomic groups, possessive quantifiers, line terminators =====
//
// (?idmsuxU-idmsuxU) switches flags up to the end of the enclosing group, a
// scoped (?..:X) only for X; (?>X) and X*+ never give back what X matched; .
// ^ $ \Z know \r, \r\n, U+0085, U+2028 and U+2029 unless (?d); (?iu) folds
// case the Unicode way, (?U) makes \w \d \s \b and the POSIX names Unicode's,
// and \b counts a non-spacing mark after a letter as part of the word.
// Every expected value is java 24.0.2's.
class S76 {
    static String all(String[] ps, String[] ss) {
        String out = "";
        for (int i = 0; i < ps.length; i++) { out = out + S74.m(ps[i], ss[i]) + ";"; }
        return out;
    }
    static void run() {
        String[] fp = {"(?i)abc", "a(?i)bc", "a(?i:b)c", "(?i:a(?-i:b)c)", "(?i)[^a]", "(?s).", "(?x) a b # c", "(?x)[a b]", "(a(?i)b)C", "a(?i)b|c",
            "(?)a", "(?-)a", "(?i-i)a", "(?q)a", "(?--i)a", "(?", "(?$)", "(?i", "(?i:a"};
        String[] fs = {"ABC", "aBC", "aBc", "aBc", "A", "\n", "ab", " ", "aBc", "C", "a", "a", "A", "a", "a", "a", "a", "a", "a"};
        Main.check("if76", all(fp, fs).equals("true;true;true;false;false;true;true;false;false;true;true;true;false;Unknown inline modifier near index 2;"
            + "Unknown inline modifier near index 3;Unknown inline modifier near index 2;Unknown group type near index 2;Unknown inline modifier near index 3;"
            + "Unclosed group near index 5;"));
        String[] ap = {"(?>a+)b", "(?>a+)a", "(?>a|ab)c", "a*+a", "a++b", "a?+a", "a{1,3}+a", "(a|ab)++c", "(?>(a|ab){2})", "(a|ab){2}+", "a+++", "a*+?", "[a-z]*+[0-9]"};
        String[] as = {"aab", "aaa", "abc", "aaa", "aab", "a", "aaa", "ababc", "aba", "abab", "a", "a", "abc1"};
        Main.check("at76", (all(ap, as) + " " + "aab".replaceAll("(?>(a))x|ab", "[$1]")).equals("true;false;false;false;true;false;false;false;true;false;"
            + "Dangling meta character '+' near index 3;Dangling meta character '?' near index 3;true; a[a]"));
        String lt = "a\rb\u0085c".replaceAll(".", "#") + "|" + "a\rb\u0085c".replaceAll("(?d).", "#") + "|" + "ab\r\n".replaceAll("$", "!")
            + "|" + "a\rb\r\nc".replaceAll("(?m)$", "!") + "|" + "a\r\nb\n".replaceAll("(?m)^", "!") + "|" + "a\nb\n".replaceAll("(?md)$", "!")
            + "|" + "ab\u2028".replaceAll("\\Z", "!") + "|" + "ab\u2028".replaceAll("(?d)\\Z", "!") + "|" + "a\u2028b".replaceAll("(?ms)^.", "#");
        Main.check("lt76", lt.equals("#\r#\u0085#|#####|ab!\r\n!|a!\rb!\r\nc!|!a\r\n!b\n|a!\nb!\n!|ab!\u2028!|ab\u2028!|#\u2028#"));
        String[] up = {"(?i)\u00e9", "(?iu)\u00e9", "(?u)(?i)\u00e9", "(?iu)k", "(?i)k", "(?iu)[k]", "(?iu)[\u00e0-\u00fe]", "(?i)[\u00e0-\u00fe]", "(?iu)(\u00e9)\\1",
            "(?i)(\u00e9)\\1", "(?iU)\u00e9", "(?i)\\p{Lu}", "(?i)\\p{Lower}", "(?i)[A-Z&&[^B]]", "(?i)\\P{Ll}", "(?iu)\u0131", "(?iu)[\u00b5]"};
        String[] us = {"\u00c9", "\u00c9", "\u00c9", "\u212a", "\u212a", "\u212a", "\u00c9", "\u00c9", "\u00e9\u00c9", "\u00e9\u00c9", "\u00c9", "a", "B", "b", "A", "I", "\u039c"};
        Main.check("uc76", all(up, us).equals("false;true;true;true;false;true;true;false;true;false;true;true;true;false;false;true;true;"));
        String uw = "x\u00e9y\u0301z".replaceAll("\\b", "|") + " " + "x\u00e9y\u0301z".replaceAll("(?U)\\b", "|") + " " + "a\u00e9\u0663_\u203f-".replaceAll("(?U)\\w", "#")
            + " " + "1\u0663\u00a0 \u2003".replaceAll("(?U)[\\d\\s]", "#") + " " + "aB\u00e9".replaceAll("(?U)\\p{Lower}", "#") + " " + "aB\u00e9".replaceAll("\\p{Lower}", "#")
            + " " + "a\u00e9\u0301".replaceAll("(?U)\\W", "#");
        Main.check("cx76", (S74.m("a(?#x)b", "ab") + ";" + S74.m("(?x)a{1, 2}", "a") + ";" + S74.m("(?x)a{1 2}", "a")).equals(
            "Unknown inline modifier near index 3;true;false"));
        Main.check("uw76", uw.equals("|x|\u00e9|y\u0301z| |x\u00e9y\u0301z| #####- ##### #B# #B\u00e9 a\u00e9\u0301"));
    }
}

// ===== SECTION 77: \k<name>, \G, \X, \b{g}, the \p families, PatternSyntaxException's getters =====
//
// \k<name> wants a group already open. \G is where the previous find ended.
// \X is one extended grapheme cluster and \b{g} Java's grapheme boundary,
// which checks against the first boundary after the previous match. \p reads
// general categories, scripts (Is.., sc=), blocks (In.., blk=), the binary
// properties and the java* methods. Every expected value is java 24.0.2's.
class S77 {
    static void run() {
        String kn = S74.m("\\k<x>(?<x>a)", "aa") + ";" + S74.m("(?<x>a)\\k<x>", "aa") + ";" + S74.m("(?i)(?<x>a)\\k<x>", "aA") + ";" + S74.m("\\k<y>", "");
        Main.check("kn77", kn.equals("named capturing group <x> does not exist near index 4;true;true;named capturing group <y> does not exist near index 4"));
        String ge = "aaa".replaceAll("\\G", "-") + "|" + "aab".replaceAll("\\Ga", "-") + "|" + "a,b,,c".replaceAll("\\G[^,]*,?", "<$0>") + "|"
            + String.join("/", "aXbX".split("\\GX|a")) + "|" + S74.m("a\\G", "a") + "|" + "xx".replaceFirst("\\Gx", "y");
        Main.check("ge77", ge.equals("-aaa|--b|<a,><b,><,><c><>|//bX|false|yx"));
        String xg = "e\u0301x\r\n\u1100\u1161\u11a8".replaceAll("\\X", "<$0>") + "|" + "\u0915\u094d\u0937a\u200db\u0600a".replaceAll("\\X", "<$0>") + "|"
            + "ae\u0301x".replaceAll("\\b{g}", "|") + "|" + String.join("/", "ae\u0301x".split("\\b{g}")) + "|" + S74.m("e\\X", "e\u0301") + "|" + S74.m("\\X\\X", "e\u0301")
            + "|" + S74.m("\\b{gx}", "") + "|" + S74.m("[\\X]", "") + "|" + S74.m("\\b{", "");
        // At the end of the input, after a previous match that ended there.
        String eg = "e\u0301".replaceAll("\\X|\\b{g}", "#") + "|" + "ab".replaceAll("b|\\b{g}", "#") + "|" + String.join("/", "ab".split("b\\b{g}"))
            + "|" + "aa".replaceAll("a\\G|\\G", "#") + "|" + "x".replaceAll("x\\X?", "#") + "|" + "ab".replaceAll("\\Gb|a", "#");
        Main.check("eg77", eg.equals("##|#a##|a|#aa|#|##"));
        Main.check("xg77", xg.equals("<e\u0301><x><\r\n><\u1100\u1161\u11a8>|<\u0915\u094d\u0937><a\u200d><b><\u0600a>||a|e\u0301|x||a/e\u0301/x|true|false"
            + "|Illegal/unsupported escape sequence near index 4|Illegal/unsupported escape sequence near index 2|Illegal repetition near index 3"));
        String pp = "abc\u00e9\u03b1\u0400".replaceAll("\\p{IsLatin}", "L") + "|" + "a\u03b1\u0400".replaceAll("\\p{InGreek}", "G") + "|" + "a\u03b1".replaceAll("\\p{sc=Grek}", "G")
            + "|" + "1\u0663\u00bd\u2167x".replaceAll("\\p{N}", "#") + "|" + "$\u20ac(".replaceAll("\\p{Sc}", "S") + "|" + "(a)".replaceAll("\\p{javaMirrored}", "M")
            + "|" + "a\u2764".replaceAll("\\p{IsEmoji}", "E") + "|" + "\u00e9a".replaceAll("\\p{InLatin-1 Supplement}", "S") + "|" + "\u00e9a".replaceAll("\\p{blk=latin_1_supplement}", "S")
            + "|" + "aA".replaceAll("\\p{IsLATIN}", "L") + "|" + "a\u4e00".replaceAll("\\p{IsIdeographic}", "I") + "|" + "\u01c5a".replaceAll("\\p{javaTitleCase}", "T")
            + "|" + "a b\u00a0".replaceAll("\\p{javaSpaceChar}", "_") + "|" + "a b\u00a0".replaceAll("\\p{javaWhitespace}", "_");
        Main.check("pp77", pp.equals("LLLL\u03b1\u0400|aG\u0400|aG|####x|SS(|MaM|aE|Sa|Sa|LL|aI|Ta|a_b_|a_b\u00a0"));
        String[] ep = {"\\p{Foo=bar}", "\\p{sc=Foo}", "\\p{IsFoo}", "\\p{isLatin}", "\\p{IsGreek_And_Coptic}", "\\p{InNoSuchBlock}"};
        String pe = "";
        for (String p : ep) { pe = pe + S74.m(p, "a") + ";"; }
        Main.check("pe77", pe.equals("Unknown Unicode property {name=<foo>, value=<bar>} near index 10;Unknown Unicode property {name=<sc>, value=<Foo>} near index 9;"
            + "Unknown character property name {IsFoo} near index 8;Unknown character property name {isLatin} near index 10;"
            + "Unknown character property name {IsGreek_And_Coptic} near index 21;Unknown character property name {InNoSuchBlock} near index 16;"));
        String ps = "";
        try { "x".matches("a\t\t{"); } catch (java.util.regex.PatternSyntaxException e) {
            ps = e.getDescription() + "|" + e.getIndex() + "|" + e.getPattern() + "|" + e.getMessage();
        }
        try { "x".split("(ab"); } catch (java.util.regex.PatternSyntaxException e) { ps = ps + "|" + e.getDescription() + "|" + e.getIndex() + "|" + e.getPattern(); }
        try { "x".replaceAll("\t**", "b"); } catch (java.util.regex.PatternSyntaxException e) { ps = ps + "|" + e.getDescription() + "|" + e.getIndex() + "|" + e.getMessage(); }
        Main.check("ps77", ps.equals("Illegal repetition|4|a\t\t{|Illegal repetition near index 4\na\t\t{|Unclosed group|3|(ab"
            + "|Dangling meta character '*'|2|Dangling meta character '*' near index 2\n\t**\n\t ^"));
    }
}

// ===== SECTION 78: supplementary characters in java.util.regex, a constructed PatternSyntaxException =====
//
// A pattern and its input are read by code point: a surrogate pair is one
// character to . [^a] \p{..} and a class range, 😀 is one escape, and
// \b reads the code point before and at the position. A search steps over a
// pair whole only when the pattern can match a supplementary character (Java's
// StartS); an empty match still advances one unit, so a* lands between the
// halves. A program's own new PatternSyntaxException(desc, regex, index)
// formats its message as Pattern's do. Every expected value is java 24.0.2's.
class S78 {
    static void run() {
        String[] ps = {".", "[^a]", "\\p{So}", "\\uD83D\\uDE00", "[😀-😂]", "\\x{1F600}", "..", "[\\x{1F600}]+", "\\W", "\\P{L}"};
        String ms = "";
        for (String p : ps) { ms = ms + "😀".matches(p) + ";"; }
        ms = ms + "𐐨".matches("(?i)𐐀") + ";" + "𐐨".matches("(?iu)𐐀") + ";" + "😀😀".matches("(.)\\1")
            + ";" + "\uD83D".matches(".");
        Main.check("sp78", ms.equals("true;true;true;true;true;true;false;true;true;true;false;true;true;true"));
        String rp = "a😀b".replaceAll(".", "#") + "|" + "x😀😁y".replaceAll("[^a-z]", "#") + "|" + "😀".replaceAll("a*", "#").length()
            + "|" + "a𝐀b".replaceAll("\\B", "#").length() + "|" + "😀😀".replaceAll("\\p{Cs}", "#") + "|" + "😀b".replaceAll("(?<=😀)b", "#")
            + "|" + "a😀b".split("").length + "|" + String.join("/", "a😀b😁c".split("[^a-z]")) + "|" + "😀x".replaceFirst("\\X", "<$0>")
            + "|" + "𝐀́x".replaceAll("\\b", "#") + "|" + "x😀".replaceAll("(?U)\\b", "#") + "|" + "\uDE00z".replaceAll(".", "#");
        String pc = "";
        for (int k = 0; k < 3; k++) {
            try {
                if (k == 0) { throw new java.util.regex.PatternSyntaxException("Bad", "a(\tb", 3); }
                if (k == 1) { throw new java.util.regex.PatternSyntaxException("None", "xyz", -1); }
                throw new java.util.regex.PatternSyntaxException(null, null, 5);
            } catch (IllegalArgumentException e) {
                java.util.regex.PatternSyntaxException pe = (java.util.regex.PatternSyntaxException) e;
                pc = pc + pe.getDescription() + "|" + pe.getIndex() + "|" + pe.getPattern() + "|" + e.getMessage() + ";";
            }
        }
        Main.check("pc78", pc.equals("Bad|3|a(\tb|Bad near index 3\na(\tb\n  \t^;None|-1|xyz|None\nxyz;null|5|null|null near index 5\nnull;"));
        Main.check("sr78", rp.equals("###|x##y|5|5|😀😀|😀#|4|a/b/c|<😀>x|𝐀́#x#|#x#😀|##"));
    }
}

// ===== SECTION 79: local classes, array type patterns, enum constants, EnumMap and EnumSet, Character's statics =====
//
// A local class body reads the enclosing method's effectively final locals. A
// type pattern or instanceof may name an array type: int[] is only int[],
// Object[] takes every reference array, an array is no List. A pattern variable
// is a local of the pattern's type, and a bare getClass() is this' class. An
// enum body names its constants bare, and compareTo / getDeclaringClass are
// java.lang.Enum's. EnumMap and EnumSet keep declaration order. Every expected
// value is java 24.0.2's.
class S79 {
    enum Tone { LOW, MID, HIGH, TOP;
        boolean edge() { return this == LOW || this == TOP; }
    }
    enum Calc {
        PLUS { int on(int a, int b) { return a + b; } },
        TIMES { int on(int a, int b) { return a * b; } };
        abstract int on(int a, int b);
    }
    record Cell(int v) {
        String who() { return getClass().getSimpleName() + v; }
    }
    static String kind(Object v) {
        return switch (v) {
            case int[] a -> "ints" + a.length;
            case String[] a -> "strs" + a.length;
            case Object[] a -> "objs" + a.length;
            default -> "other";
        };
    }
    static void run() {
        int base = 7;
        String tag = "t";
        class Scale {
            int by(int k) { return k * base; }
            String name() { return tag + base; }
        }
        Main.check("lc79", new Scale().by(3) == 21 && new Scale().name().equals("t7"));
        Main.check("gc79", new Cell(5).who().equals("Cell5"));
        Object[] vs = {new int[]{1, 2}, new String[]{"a"}, new String[][]{{"b"}}, new int[][]{{1}}, new Cell[]{new Cell(1)}, "s", 4};
        String ap = "";
        for (Object v : vs) {
            ap = ap + (v instanceof int[] ? 1 : 0) + (v instanceof String[] ? 1 : 0) + (v instanceof Object[] ? 1 : 0)
                + (v instanceof Object[][] ? 1 : 0) + (v instanceof Cell[] ? 1 : 0) + (v instanceof List ? 1 : 0) + kind(v) + ";";
        }
        Main.check("ap79", ap.equals("100000ints2;011000strs1;001100objs1;001000objs1;001010objs1;000000other;000000other;"));
        String pv = "";
        for (Cell s : List.of(new Cell(2))) { pv = pv + s.v(); }
        Object o = "pattern";
        if (o instanceof String s && s.length() > 3) { pv = pv + s.toUpperCase(); }
        Object w = List.of(1, 2);
        String sw = switch (w) { case List<?> s -> "n" + s.size(); default -> "?"; };
        Main.check("pv79", (pv + sw).equals("2PATTERNn2"));
        String ec = Tone.LOW.edge() + " " + Tone.MID.edge() + " " + Tone.TOP.compareTo(Tone.MID) + " " + Calc.TIMES.getDeclaringClass().getSimpleName()
            + " " + (Calc.PLUS.getDeclaringClass() == Calc.class) + " " + Calc.class.isEnum() + " " + Tone.class.getEnumConstants().length + " " + String.class.getEnumConstants();
        Main.check("ec79", ec.equals("true false 2 Calc true true 4 null"));
        EnumMap<Tone, Integer> em = new EnumMap<>(Tone.class);
        em.put(Tone.TOP, 4); em.put(Tone.LOW, 1); em.merge(Tone.LOW, 10, Integer::sum);
        EnumSet<Tone> two = EnumSet.of(Tone.HIGH, Tone.LOW);
        String es = em + " " + two + " " + EnumSet.complementOf(two) + " " + EnumSet.range(Tone.MID, Tone.TOP) + " " + EnumSet.allOf(Calc.class)
            + " " + EnumSet.noneOf(Tone.class) + " " + EnumSet.copyOf(List.of(Tone.TOP, Tone.LOW));
        try { EnumSet.range(Tone.TOP, Tone.LOW); } catch (IllegalArgumentException e) { es = es + " " + e.getMessage(); }
        Main.check("em79", es.equals("{LOW=11, TOP=4} [LOW, HIGH] [MID, TOP] [MID, HIGH, TOP] [PLUS, TIMES] [] [LOW, TOP] TOP > LOW"));
        Set<Integer> st = new TreeSet<>(List.of(1, 2, 3, 4, 5));
        Set<Integer> hs = new HashSet<>(List.of(1, 2, 3, 4, 5));
        boolean r1 = st.removeAll(List.of(2, 4, 9));
        boolean r2 = hs.retainAll(Set.of(1, 5, 7));
        boolean r3 = hs.retainAll(Set.of(1, 5));
        Main.check("sr79", (st + " " + hs + " " + r1 + r2 + r3).equals("[1, 3, 5] [1, 5] truetruefalse"));
        int[] cs = {32, 160, 9, 127, 453, 888, 95, 36, 173, 65279, 55357, 56832, 8364, 19968, 10084};
        String ch = "";
        for (int c : cs) {
            char x = (char) c;
            ch = ch + (Character.isSpaceChar(x) ? 1 : 0) + (Character.isISOControl(x) ? 1 : 0) + (Character.isTitleCase(x) ? 1 : 0)
                + (Character.isDefined(x) ? 1 : 0) + (Character.isJavaIdentifierStart(x) ? 1 : 0) + (Character.isJavaIdentifierPart(x) ? 1 : 0)
                + (Character.isIdentifierIgnorable(x) ? 1 : 0) + (Character.isHighSurrogate(x) ? 1 : 0) + (Character.isLowSurrogate(x) ? 1 : 0)
                + (Character.isMirrored(x) ? 1 : 0) + (Character.isIdeographic(c) ? 1 : 0) + (Character.isEmoji(c) ? 1 : 0) + Character.getType(x) + ";";
        }
        ch = ch + Character.charCount(0x1F600) + Character.toCodePoint('\uD83D', '\uDE00') + (int) Character.highSurrogate(0x1F600)
            + (int) Character.lowSurrogate(0x1F600) + Character.isSurrogatePair('\uD83D', '\uDE00') + Character.isValidCodePoint(0x110000)
            + Character.isSupplementaryCodePoint(0x10000) + Character.isBmpCodePoint(0xFFFF) + (int) Character.reverseBytes('a');
        Main.check("ch79", ch.equals("10010000000012;10010000000012;01010000000015;01010110000015;0011110000003;0000000000000;00011100000023;00011100000026;00010110000016;00010110000016;00010001000019;00010000100019;00011100000026;0001110000105;00010000000128;21285125535756832truefalsetruetrue24832"));
    }
}

// ===== SECTION 80: Collection's bulk operations, descending TreeMap and TreeSet =====
//
// removeIf, removeAll, retainAll and containsAll work on every collection,
// a map's key, value and entry views and a deque included. descendingMap,
// descendingKeySet, descendingSet and descendingIterator read in reverse
// order. Every expected value is java 24.0.2's.
class S80 {
    static void run() {
        List<Integer> l = new ArrayList<>(List.of(1, 2, 3, 2, 4, 5));
        boolean b1 = l.removeAll(List.of(2, 9));
        boolean b2 = l.retainAll(Set.of(1, 3, 5, 7));
        boolean b3 = l.containsAll(List.of(1, 5));
        Map<String, Integer> m = new HashMap<>(Map.of("a", 1, "b", 2, "c", 3, "d", 4));
        m.entrySet().removeIf(e -> e.getValue() % 2 == 0);
        Map<String, Integer> tm = new TreeMap<>(Map.of("w", 1, "x", 2, "y", 3, "z", 4));
        tm.values().removeIf(v -> v > 3);
        tm.keySet().retainAll(Set.of("w", "y", "z"));
        Deque<Integer> dq = new ArrayDeque<>(List.of(5, 6, 7, 8));
        dq.removeIf(v -> v % 2 == 1);
        String bulk = l + " " + b1 + b2 + b3 + " " + new TreeMap<>(m) + " " + tm + " " + dq;
        Main.check("bk80", bulk.equals("[1, 3, 5] truetruetrue {a=1, c=3} {w=1, y=3} [6, 8]"));
        TreeMap<Integer, String> t = new TreeMap<>(Map.of(3, "c", 1, "a", 2, "b"));
        TreeSet<String> ts = new TreeSet<>(List.of("q", "p", "r"));
        String rev = t.descendingMap() + " " + t.descendingKeySet() + " " + ts.descendingSet() + " " + ts.descendingIterator().next() + " " + t.descendingMap().firstKey();
        Main.check("dm80", rev.equals("{3=c, 2=b, 1=a} [3, 2, 1] [r, q, p] r 3"));
    }
}

// ===== SECTION 81: enclosing classes' members named unqualified, fields before their initializers =====
//
// A nested class names its enclosing class's statics unqualified, its static
// initializers run after the enclosing class's, an inner or anonymous class
// calls the enclosing instance's methods unqualified, and a field a superclass
// constructor reads through an override still holds its type's zero. Every
// expected value is java 24.0.2's.
class S81 {
    static int base = 40;
    static String tag(int n) { return "t" + n; }
    private int hits = 0;
    private void hit() { hits++; }
    static class Nest {
        static int derived = base + 2;
        String show() { return tag(base) + derived; }
        static String twice() { return tag(base * 2); }
    }
    class In {
        String go() { hit(); hit(); return "in" + hits; }
    }
    Runnable anon() {
        return new Runnable() {
            public void run() { hit(); }
        };
    }
    static class Top {
        String seen = "";
        Top() { seen = probe(); }
        String probe() { return "top"; }
    }
    static class Sub extends Top {
        int n = 5;
        boolean on = true;
        double d = 1.5;
        String probe() { return n + " " + on + " " + d; }
    }
    static void run() {
        Main.check("ns81", (new Nest().show() + " " + Nest.twice()).equals("t4042 t80"));
        S81 o = new S81();
        String ia = o.new In().go();
        o.anon().run();
        Main.check("ia81", (ia + " " + o.hits).equals("in2 3"));
        Sub s = new Sub();
        Main.check("fz81", (s.seen + " | " + s.probe()).equals("0 false 0.0 | 5 true 1.5"));
    }
}

// ===== SECTION 82: record components' declared types and a varargs component =====
//
// The canonical constructor converts each argument to its component's type
// (an int passed for a double component is 1.0) and collects a varargs last
// component into an array, or takes one given whole; a declared canonical
// constructor replaces the generated one. Every expected value is java 24.0.2's.
class S82 {
    record Comp(double x, long y) {}
    record Half(double v) {
        Half { v = v / 2; }
    }
    record Bag(String label, int... items) {
        int total() { int t = 0; for (int i : items) { t += i; } return t; }
    }
    record Inc(long a) {
        Inc(long a) { this.a = a + 1; }
    }
    static void run() {
        Comp p = new Comp(1, 2);
        String rc = p + " " + (p.x() / 4) + " " + new Half(3) + " " + new Bag("b", 1, 2, 3).total() + " " + new Bag("e").items().length
            + " " + new Bag("w", new int[]{4, 5}).total() + " " + new Inc(5);
        Main.check("rc82", rc.equals("Comp[x=1.0, y=2] 0.25 Half[v=1.5] 6 0 9 Inc[a=6]"));
    }
}

// ===== SECTION 83: java.util.regex.Pattern and Matcher =====
//
// find / matches / lookingAt, groups by number and name with their bounds,
// the flags, split, replaceAll with a template or a function,
// appendReplacement / appendTail, find(int), \G across finds, and the
// exceptions a Matcher throws. Every expected value is java 24.0.2's.
class S83 {
    static void run() {
        Matcher m = Pattern.compile("(\\w+)@(\\w+)\\.com").matcher("ann@foo.com, bob@bar.com");
        String fd = "";
        while (m.find()) { fd = fd + m.group(1) + "/" + m.group(2) + "@" + m.start() + "-" + m.end(2) + ";"; }
        Matcher d = Pattern.compile("(?<y>\\d{4})-(?<m>\\d\\d)").matcher("on 2024-03!");
        fd = fd + d.find() + d.group("y") + d.group("m") + d.start("m") + d.groupCount();
        Main.check("pm83", fd.equals("ann/foo@0-7;bob/bar@13-20;true20240382"));
        Pattern ci = Pattern.compile("hello", Pattern.CASE_INSENSITIVE);
        String fl = ci.matcher("HeLLo you").lookingAt() + " " + ci.matcher("HeLLo you").matches() + " " + ci.flags() + " " + Pattern.compile("(?im)x").flags()
            + " " + Pattern.compile("a.c", Pattern.DOTALL).matcher("a\nc").matches() + " " + Pattern.compile("a.c", Pattern.LITERAL).matcher("abc a.c").find()
            + " " + Pattern.matches("a+b", "aab") + " " + Pattern.quote("x.y");
        Main.check("pf83", fl.equals("true false 2 10 true true true \\Qx.y\\E"));
        Matcher am = Pattern.compile("cat").matcher("one cat two cats");
        StringBuilder sb = new StringBuilder();
        while (am.find()) { am.appendReplacement(sb, "dog"); }
        am.appendTail(sb);
        String rp = sb + "|" + Pattern.compile("(\\d+)").matcher("x1y22").replaceAll("<$1>") + "|" + Pattern.compile("\\d+").matcher("a1b22").replaceAll(r -> "" + r.group().length())
            + "|" + Arrays.toString(Pattern.compile(",").split("a,b,,c,,")) + "|" + Arrays.toString(Pattern.compile(",").split("a,b,c", 2));
        Main.check("pr83", rp.equals("one dog two dogs|x<1>y<22>|a1b2|[a, b, , c]|[a, b,c]"));
        Matcher fm = Pattern.compile("\\d").matcher("a1b2c3");
        Matcher gm = Pattern.compile("\\Ga").matcher("aab");
        int ga = 0;
        while (gm.find()) { ga++; }
        Matcher g = Pattern.compile("(a)|(b)").matcher("b");
        String ex = fm.find(3) + fm.group() + ga + g.matches() + g.group(1) + g.group(2) + g.start(1);
        try { Pattern.compile("z").matcher("abc").group(); } catch (IllegalStateException e) { ex = ex + "|" + e.getMessage(); }
        try { g.group(3); } catch (IndexOutOfBoundsException e) { ex = ex + "|" + e.getMessage(); }
        try { g.group("q"); } catch (IllegalArgumentException e) { ex = ex + "|" + e.getMessage(); }
        Main.check("px83", ex.equals("true22truenullb-1|No match found|No group 3|No group with name <q>"));
    }
}

// ===== SECTION 84: summary statistics, compensated double collectors, BitSet =====
//
// Collectors.summingDouble / averagingDouble and DoubleSummaryStatistics sum
// with Kahan compensation, as DoubleStream.sum does; the summarizing
// collectors and Double/LongStream.summaryStatistics answer the statistics
// objects; BitSet's set algebra and rendering. Every expected value is java
// 24.0.2's.
class S84 {
    static void run() {
        List<Double> ds = new ArrayList<>();
        for (int i = 0; i < 10; i++) { ds.add(0.1); }
        ds.add(0.7);
        String sm = ds.stream().collect(Collectors.summingDouble(d -> d)) + " " + ds.stream().collect(Collectors.averagingDouble(d -> d));
        DoubleSummaryStatistics dst = ds.stream().collect(Collectors.summarizingDouble(d -> d));
        LongSummaryStatistics lst = LongStream.of(4, 9, 1).summaryStatistics();
        sm = sm + " " + dst.getSum() + " " + dst.getMax() + " " + lst + " " + DoubleStream.of().summaryStatistics();
        Main.check("st84", sm.equals("1.7 0.15454545454545454 1.7 0.7 LongSummaryStatistics{count=3, sum=14, min=1, average=4.666667, max=9} DoubleSummaryStatistics{count=0, sum=0.000000, min=Infinity, average=0.000000, max=-Infinity}"));
        BitSet a = new BitSet();
        a.set(0, 5);
        a.clear(2);
        BitSet b = new BitSet();
        b.set(3);
        b.set(64);
        BitSet c = (BitSet) a.clone();
        c.or(b);
        String bs = a + " " + c + " " + c.cardinality() + " " + c.length() + " " + a.nextClearBit(0) + " " + c.nextSetBit(5) + " " + a.intersects(b) + " " + c.hashCode();
        Main.check("bs84", bs.equals("{0, 1, 3, 4} {0, 1, 3, 4, 64} 5 65 2 64 true 1227"));
    }
}

// ===== SECTION 85: a static field and a method of one name =====
//
// Fields and methods are separate namespaces (JLS 8.2): `static int count`
// and `static int count()` coexist, read and written unqualified, qualified,
// compound and through an array element, and a subclass's static field may
// share an inherited method's name. Every expected value is java 24.0.2's.
class S85 {
    static int count;
    static int count() { return count + 100; }
    static int[] arr = {1, 2};
    static int arr() { return arr[0] + arr[1]; }
    static String tag = "t";
    String tag() { return "m" + tag; }
    static class NmBase { int val() { return 1; } }
    static class NmSub extends NmBase { static int val = 40; int both() { return val() + val; } }
    static class Peek { int peek() { return count + count() + arr(); } }
    static void run() {
        count = 5;
        String nm = count() + " " + count + " " + S85.count() + " " + S85.count;
        S85.count = 7;
        S85.count++;
        S85.count += 2;
        ++count;
        count *= 2;
        nm = nm + " " + count + " " + count();
        S85.arr[0] = 10;
        arr[1] += 5;
        nm = nm + " " + arr() + " " + S85.arr[0] + " " + new S85().tag() + tag + S85.tag + " " + new NmSub().both() + " " + NmSub.val
            + " " + new Peek().peek();
        Main.check("nm85", nm.equals("105 5 105 5 22 122 17 10 mttt 41 40 161"));
    }
}

// ===== SECTION 86: a field that hides a superclass's =====
//
// A subclass field of an inherited field's name hides it (JLS 8.3) and the
// object holds both: the class the code is written in picks one for an
// unqualified name, the static type for a qualified one, and super.f and a
// cast reach the hidden one, for reads, writes, compound and ++/--. Every
// expected value is java 24.0.2's.
class S86 {
    static class HA {
        String name = "A";
        int n = 1;
        HA() { n += 10; }
        String who() { return name + n; }
        void bumpA() { n++; this.n += 2; }
    }
    static class HB extends HA {
        String name = "B";
        int n = 2;
        HB() { super(); n += 20; this.name = "B2"; super.name = "A2"; }
        String both() { return name + super.name + ((HA) this).name + who() + n + super.n + this.n; }
        void bumpB() { n++; ++n; super.n++; n *= 3; super.n -= 1; }
    }
    static class HC extends HB {
        String all() { return name + n + super.name + super.n + ((HA) this).name + ((HA) this).n; }
    }
    static class HD extends HA {
        int n = 5;
        int get(int n) { return n + this.n + super.n; }
    }
    static void run() {
        HB b = new HB();
        HA a = b;
        String hd = b.both() + " " + a.name + b.name + a.n + b.n;
        b.bumpA();
        b.bumpB();
        hd = hd + " " + b.both();
        b.name = "BB";
        a.name = "AA";
        b.n = 100;
        a.n = 200;
        b.n++;
        a.n += 5;
        hd = hd + " " + b.both() + a.n + b.n + " " + new HC().all() + " " + new HD().get(1000);
        Main.check("hd86", hd.equals("B2A2A2A211221122 A2B21122 B2A2A2A214721472 BBAAAAAA205101205101205101 B222B222A211 1016"));
    }
}

// ===== SECTION 87: a class is initialized at its first use =====
//
// JLS 12.4.1: a static block and the static field initializers run when an
// instance is created, a static method is invoked (after its arguments) or
// a static field that is no constant variable is used (a plain assignment
// after its right-hand side) - a superclass first, a member class not with
// its enclosing one. Until then a field holds its default, which an
// initialization cycle reads. Every expected value is java 24.0.2's.
class S87 {
    static StringBuilder log = new StringBuilder();
    static int note(String s) { log.append(s); return s.length(); }
    static class LzO { static { log.append("O"); } static int v = 7; static final int K = 3; static final String S = "s"; static void f() { log.append("f"); } }
    static class LzP { static int w = mk(); static int mk() { log.append("P"); return 1; } }
    static class LzBase { static { log.append("[B]"); } static int bv = note("b"); }
    static class LzDerived extends LzBase { static { log.append("[D]"); } LzDerived() { log.append("d"); } static int dv = 4; }
    static class LzArg { static { log.append("A"); } static int g(int x) { return x; } }
    static class LzWr { static int x; static { log.append("W"); } }
    static class LzCyc1 { static int a = LzCyc2.b + 1; static { log.append("c1:" + a); } }
    static class LzCyc2 { static int b = LzCyc1.a + 10; static { log.append("c2:" + b); } }
    static class LzOuter {
        static List<String> items = new ArrayList<>();
        static { items.add("o"); log.append("Ou"); }
        static class LzIn { static { log.append("In"); } int size() { return items.size(); } }
    }
    static class LzAnon { static { log.append("An"); } int k() { return 1; } }
    static class LzSelf { static int n; static { log.append("Se"); } static int bump() { return ++n; } static int twice() { return bump() + bump(); } }
    static void run() {
        log.append("L:");
        int k = LzO.K;
        String s = LzO.S;
        log.append("|" + k + s + "|");
        LzO.f();
        log.append("|");
        int x = LzP.w;
        log.append("|");
        new LzDerived();
        log.append("|" + LzDerived.dv + "|");
        LzArg.g(note("x"));
        log.append("|");
        LzWr.x = note("y");
        log.append("|" + LzWr.x + "|");
        log.append(LzCyc1.a + "," + LzCyc2.b + "|");
        log.append(new LzOuter.LzIn().size() + "|");
        LzAnon an = new LzAnon() { int k() { return 2; } };
        log.append(an.k() + "|" + LzSelf.twice() + "|" + LzO.v + x);
        Main.check("ci87", log.toString().equals("L:|3s|Of|P|[B]b[D]d|4|xA|yW|1|c2:10c1:1111,10|InOu1|AnSe2|3|71"));
    }
}

// ===== SECTION 88: nested types of one simple name =====
//
// A member or local type is in scope only in the type or block declaring it
// (JLS 6.3), so other sections' Pt, Box, Top, Sub and Inner are different
// types: each name here means this section's own, an inherited member type
// included, and two local classes of one name stay apart. getSimpleName,
// getName and a record's toString show the names javac gives. Every
// expected value is java 24.0.2's.
class S88 {
    record Pt(String tag) {}
    static class Box { String in() { return "box88"; } }
    static class Top { String probe() { return "t88"; } }
    static class Sub extends Top { String probe() { return "s88" + new Pt("p"); } }
    static class Holder { static class Inner { String w() { return "hi"; } } }
    static class Kid extends Holder { String use() { return new Inner().w(); } }
    static String one() { class Tmp { String v() { return "a"; } } return new Tmp().v(); }
    static String two() { class Tmp { String v() { return "b"; } } return new Tmp().v(); }
    static void run() {
        Pt p = new Pt("x");
        Object o = p;
        Top t = new Sub();
        String r = p + " " + Pt.class.getSimpleName() + " " + p.getClass().getName() + " " + new Box().in() + " " + t.probe()
            + " " + new Kid().use() + " " + one() + two() + " " + (o instanceof Pt q ? q.tag() : "-") + " " + new S88.Box().in()
            + " " + new S88.Holder.Inner().getClass().getName();
        Main.check("nt88", r.equals("Pt[tag=x] Pt S88$Pt box88 s88Pt[tag=p] hi ab x box88 S88$Holder$Inner"));
    }
}

// ===== SECTION 89: collections' marker interfaces, descending views, Matcher regions =====
//
// The collections are Cloneable and Serializable; descendingMap,
// descendingKeySet and descendingSet are live views in reverse order (their
// first is the last, their floor the ceiling, their head the keys above);
// a Matcher's region bounds what it sees (opaque, anchoring), usePattern
// keeps its position and results() streams the further matches. Every
// expected value is java 24.0.2's.
class S89 {
    static void run() {
        Object[] os = { new ArrayList<Integer>(), new LinkedList<Integer>(), new HashMap<String, Integer>(), new TreeSet<Integer>(),
                        new ArrayDeque<Integer>(), new PriorityQueue<Integer>(), "s", new Object() };
        String mk = "";
        for (Object o : os) { mk = mk + (o instanceof Cloneable ? "C" : "c") + (o instanceof Serializable ? "S" : "s"); }
        Main.check("mk89", mk.equals("CSCSCSCSCScScScs"));
        TreeMap<Integer, String> tm = new TreeMap<>();
        for (int i = 1; i <= 6; i++) { tm.put(i * 10, "v" + i); }
        NavigableMap<Integer, String> d = tm.descendingMap();
        String dm = d + " " + d.firstKey() + d.lastKey();
        d.put(35, "x");
        tm.remove(60);
        d.remove(10);
        dm = dm + " " + tm + d + d.keySet() + d.floorKey(36) + d.ceilingKey(36) + d.floorKey(5) + d.headMap(30) + d.tailMap(30, false)
            + d.subMap(50, 20) + d.pollFirstEntry() + tm.descendingKeySet() + d.descendingMap();
        TreeSet<String> ts = new TreeSet<>(List.of("b", "a", "d", "c"));
        NavigableSet<String> ds = ts.descendingSet();
        ds.add("e");
        ts.remove("a");
        dm = dm + " " + ts + ds + ds.first() + ds.floor("bb");
        Main.check("dv89", dm.equals("{60=v6, 50=v5, 40=v4, 30=v3, 20=v2, 10=v1} 6010 {20=v2, 30=v3, 35=x, 40=v4, 50=v5}"
            + "{50=v5, 40=v4, 35=x, 30=v3, 20=v2}[50, 40, 35, 30, 20]403520{50=v5, 40=v4, 35=x}{20=v2}{50=v5, 40=v4, 35=x, 30=v3}"
            + "50=v5[40, 35, 30, 20]{20=v2, 30=v3, 35=x, 40=v4} [b, c, d, e][e, d, c, b]ec"));
        Matcher m = Pattern.compile("\\d+").matcher("a12b345c6d78");
        m.region(2, 9);
        String rg = "";
        while (m.find()) { rg = rg + m.group() + "@" + m.start() + " "; }
        Matcher a = Pattern.compile("^b\\d+").matcher("a12b345c6d78");
        a.region(3, 7);
        rg = rg + a.lookingAt() + a.group() + a.regionStart() + a.regionEnd() + Pattern.compile("(?<=2)b").matcher("a12b").region(3, 4).find()
            + Pattern.compile("b$").matcher("abc").region(0, 2).find();
        Matcher u = Pattern.compile("[a-z]+").matcher("abc 123 def");
        u.find();
        u.usePattern(Pattern.compile("\\d+"));
        rg = rg + " " + u.find() + u.group() + u.start();
        List<String> r = Pattern.compile("(\\w)(\\d)").matcher("a1 b2 c3").results().map(x -> x.group(2) + x.start()).collect(Collectors.toList());
        MatchResult first = Pattern.compile("o+").matcher("foo").results().findFirst().get();
        rg = rg + " " + r + first.group() + " " + a;
        try { a.region(3, 2); } catch (IndexOutOfBoundsException e) { rg = rg + " " + e.getMessage(); }
        Main.check("rx89", rg.equals("2@2 345@4 6@8 trueb34537falsetrue true1234 [10, 23, 36]oo "
            + "java.util.regex.Matcher[pattern=^b\\d+ region=3,7 lastmatch=b345] start > end"));
    }
}

// ===== SECTION 90: static members through a subclass, RandomAccess =====
//
// A static field or method is inherited (JLS 8.3, 8.4.8): named through a
// subclass, or an instance of one, it is the declaring class's, for reads,
// writes, compound and ++, and it initializes only that class (JLS 12.4.1);
// a redeclaration hides it further down. ArrayList and Stack are
// RandomAccess, neither a LinkedList nor a Stack is an ArrayList, a
// LinkedList is no RandomAccess and a StringBuilder is Serializable; a
// library type is named by its package even when another library is
// compiled while its own is. Every expected value is java 24.0.2's.
class S90 {
    static String log = "";
    static class Base {
        static int k = 5;
        static { log = log + "B"; }
        static String name() { return "B" + k; }
        static String pick(int x) { return "i"; }
        static String pick(String x) { return "s"; }
    }
    static class Mid extends Base {
        static { log = log + "M"; }
        static String name() { return "M"; }
    }
    static class Leaf extends Mid {
        static { log = log + "L"; }
    }
    static class Sub extends Base {
        static { log = log + "S"; }
        int sum() { return k + Sub.k; }
    }
    static void run() {
        String st = Sub.k + " " + log;
        Sub.k = 7;
        st = st + " " + Base.k;
        Sub.k += 2;
        Sub.k++;
        ++Leaf.k;
        st = st + " " + Base.k + Sub.k + Leaf.k + " " + Sub.name() + Leaf.name() + Mid.name() + " " + Sub.pick(1) + Leaf.pick("x");
        Sub s = new Sub();
        st = st + " " + s.k + s.sum() + " " + log;
        Main.check("st90", st.equals("5 B 7 111111 B11MM is 1122 BMS"));
        Object[] os = { new ArrayList<Integer>(), new LinkedList<Integer>(), new Stack<Integer>(), new StringBuilder("x"), "y" };
        String mk = "";
        for (Object o : os) { mk = mk + (o instanceof RandomAccess ? "R" : "r") + (o instanceof Serializable ? "S" : "s") + (o instanceof ArrayList ? "A" : "a"); }
        Main.check("mk90", mk.equals("RSArSaRSarSarSa"));
        Main.check("nm90", (os[1].getClass().getName() + " " + os[2].getClass().getName()).equals("java.util.LinkedList java.util.Stack"));
    }
}

// ===== SECTION 91: a static field named through an instance =====
//
// A static field named through an expression is the field of the class the
// expression's static type finds it in (JLS 15.11.1): `this.j`, `s.j` and
// `b.k` with b typed as the superclass read and write the class's one copy,
// a hiding redeclaration is chosen by the static type, and the expression
// is evaluated and its value discarded, even when it is null. Every
// expected value is java 24.0.2's.
class S91 {
    static String log = "";
    static class Base { static int k = 1; int inst = 2; }
    static class Sub extends Base {
        static int j = 5;
        int m() { return this.j + 1; }
        int n() { this.j = 7; return j; }
    }
    static class Hide extends Base { static int k = 20; }
    static Sub mk() { log = log + "mk"; return new Sub(); }
    static void run() {
        Sub s = new Sub();
        Base b = s;
        String st = s.j + " " + b.k + " " + s.k + " " + s.m();
        s.j = 8;
        b.k = 3;
        st = st + " " + Sub.j + Base.k + (s.j + b.k);
        s.j += 2; s.k++; ++b.k;
        st = st + " " + Sub.j + Base.k + s.n() + Sub.j;
        Hide h = new Hide();
        Base hb = h;
        Sub nul = null;
        st = st + " " + h.k + hb.k + ((Base) h).k + " " + mk().j + nul.j + log;
        Main.check("sf91", st.equals("5 1 1 6 8311 10577 2055 77mk"));
    }
}

// ===== SECTION 92: StringBuffer, CharSequence, the list types' clone =====
//
// StringBuffer has StringBuilder's operations and is its own class: neither
// is an instance of the other, both are CharSequences, as a String and a
// class implementing the interface are. An ArrayList's clone is a shallow
// copy of the same class, a LinkedList's a LinkedList and a Stack's a Stack.
// Every expected value is java 24.0.2's.
class S92 {
    static class Name implements CharSequence {
        public int length() { return 2; }
        public char charAt(int i) { return "ab".charAt(i); }
        public CharSequence subSequence(int s, int e) { return "ab".subSequence(s, e); }
        public String toString() { return "ab"; }
    }
    static int len(CharSequence cs) { return cs.length(); }
    static String f(CharSequence c) { return "cs" + c.length(); }
    static String f(Object o) { return "obj"; }
    static String g(StringBuilder b) { return "sb"; }
    static String g(CharSequence c) { return "cs"; }
    static void run() {
        StringBuffer buf = new StringBuffer("xy");
        Object o = "q";
        String ov = f("abc") + f(new StringBuilder("ab")) + f(buf) + f(3) + f(o) + " " + g(new StringBuilder()) + g(buf) + g("s")
            + " " + String.join("-", new StringBuilder("a"), buf, "c") + "xy".contentEquals(buf);
        Main.check("ov92", ov.equals("cs3cs2cs2objobj sbcscs a-xy-ctrue"));
        StringBuffer sb = new StringBuffer("x");
        sb.append(1).append('c').append(2.5).insert(0, "<").reverse();
        StringBuffer e = new StringBuffer();
        e.append("hello world").setLength(5);
        e.deleteCharAt(0).replace(0, 1, "E");
        String st = sb + " " + sb.length() + sb.charAt(0) + " " + sb.getClass().getName() + " " + e + e.indexOf("l");
        Object o1 = new StringBuilder("q"), o2 = sb, o3 = "s", o4 = new Name();
        st = st + " " + (o1 instanceof CharSequence) + (o2 instanceof CharSequence) + (o3 instanceof CharSequence)
            + (o4 instanceof CharSequence) + (o1 instanceof StringBuffer) + (o2 instanceof StringBuilder);
        CharSequence cs = new StringBuilder("hello");
        st = st + " " + len(new StringBuilder("abc")) + len(sb) + len("hi") + len(new Name()) + " " + cs.subSequence(1, 3)
            + cs.charAt(4) + cs.isEmpty() + " " + new StringBuilder(new Name()) + new StringBuffer(cs)
            + " " + new StringBuffer("abc").compareTo(new StringBuffer("abd"));
        Main.check("sb92", st.equals("5.2c1x< 75 java.lang.StringBuffer Ello1 truetruetruetruefalsefalse 3722 elofalse abhello -1"));
        ArrayList<Integer> a = new ArrayList<>(List.of(4, 7));
        @SuppressWarnings("unchecked")
        ArrayList<Integer> c = (ArrayList<Integer>) a.clone();
        c.add(1);
        LinkedList<Integer> l = new LinkedList<>(List.of(1, 2));
        Object lc = l.clone();
        Stack<Integer> s = new Stack<>();
        s.push(5);
        Object sc = s.clone();
        s.push(6);
        String cl = a + " " + c + " " + lc + lc.getClass().getName() + (lc instanceof LinkedList) + " " + sc + sc.getClass().getName() + s;
        Main.check("cl92", cl.equals("[4, 7] [4, 7, 1] [1, 2]java.util.LinkedListtrue [5]java.util.Stack[5, 6]"));
    }
}

// ===== SECTION 93: a type declared after the types that extend it =====
//
// Type declarations have no order (JLS 7.6): a class may extend a class, and
// implement or extend an interface, declared further down - at the top
// level, as a member type, from a member type to a top-level one, and from
// an enum - and its static initialization still runs at its first use,
// after its superclass's (JLS 12.4.1). Every expected value is java 24.0.2's.
class S93 {
    static String log = "";
    static int note(String t) { log = log + t; return t.length(); }
    static class Sub extends Base implements Serializable {
        static int s = note("S");
        Sub() { super(10); }
        String who() { return "Sub" + v; }
    }
    static class Leaf extends Sub { static int l = note("L"); }
    static class Base {
        static int b = note("B");
        int v;
        Base(int v) { this.v = v; }
        String who() { return "B" + v; }
    }
    static class A {
        static class In extends B93 { static int s = 4; int z() { return s + w; } }
        static int a = 1;
    }
    enum E implements J { X; public String j() { return "E" + K; } }
    interface J extends L { String j(); }
    interface L { int K = 7; default String l() { return "L" + K; } }
    static void run() {
        Base x = new Sub();
        String st = x.who() + " " + (x instanceof Serializable) + " " + new Base(10).who() + " " + log;
        Object o = new Leaf();
        st = st + " " + (o instanceof Base) + log + " " + new A.In().z() + A.a + A.In.s
            + " " + E.X.j() + E.X.l() + (E.X instanceof L) + " " + new T93().t();
        Main.check("or93", st.equals("Sub10 true B10 BS trueBSL 714 E7L7true 6"));
    }
}
class T93 extends U93 { int t() { return u + 1; } }
class B93 { static int b = 2; int w = 3; }
class U93 { int u = 5; }

// ===== SECTION 94: Object.clone through super.clone() =====
//
// Object.clone copies a Cloneable instance field by field - the copy is of the
// same class, and shares every object its fields refer to - and throws
// CloneNotSupportedException, whose message is the binary class name, for an
// instance that is not Cloneable (JLS 4.3.2, java.lang.Object). Every expected
// value is java 24.0.2's.
class S94 {
    int base = 40;
    class Lamb implements Cloneable {
        int k = 2;
        int sum() { return base + k; }
        Lamb dup() throws CloneNotSupportedException { return (Lamb) super.clone(); }
    }
    interface Copyable extends Cloneable { }
    static class Base { int b = 1; }
    static class Mid extends Base implements Copyable {
        long l = 5L; double d = 2.5; char ch = 'z'; int[] arr = {1, 2};
        public Mid clone() {
            try { return (Mid) super.clone(); } catch (CloneNotSupportedException e) { throw new AssertionError(e); }
        }
    }
    static class NotC { Object c() throws CloneNotSupportedException { return super.clone(); } }
    static void run() {
        S94 o = new S94();
        Lamb in = o.new Lamb();
        String st = "";
        try {
            Lamb c = in.dup();
            o.base = 50;
            st = c.sum() + " " + (c == in) + " " + c.getClass().getName();
        } catch (CloneNotSupportedException e) { st = "no"; }
        Mid m = new Mid();
        m.b = 9;
        Mid mc = m.clone();
        mc.arr[0] = 7;
        mc.b = 3;
        st = st + " " + m.b + mc.b + " " + (mc.l + 1L) + mc.d + mc.ch + m.arr[0] + " " + (mc instanceof Cloneable);
        try { new NotC().c(); st = st + " cloned"; }
        catch (CloneNotSupportedException e) { st = st + " " + e; }
        Main.check("cl94", st.equals("52 false S94$Lamb 93 62.5z7 true java.lang.CloneNotSupportedException: S94$NotC"));
    }
}
