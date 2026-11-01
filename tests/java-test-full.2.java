// Full-syntax test: Java (Java SE 21 core grammar) - part 2.
//
// The java ratchet is two files so that neither -frozen interpreter leg comes
// near the IR interpreter's step brake (-max-steps, 100M per call): the leg runs
// a whole file in one call, and the imported library sources (lib/java-*.java)
// are compiled into that same call. See tests/java-test-full.java for the
// conventions and the oracle, which are the same.

import java.io.Serializable;
import java.lang.reflect.Array;
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
        S95.run(); // SECTION-CALL 95
        S96.run(); // SECTION-CALL 96
        S97.run(); // SECTION-CALL 97
        System.out.println("full: " + Main.checks + " checks, " + Main.failures + " failures");
        System.exit(Main.failures);
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
// unboxing. A null Integer as an array index or dimension or a String
// position unboxes where it is evaluated - before an assignment's right-hand
// side and the next dimension - and a null stored into an int[] before the
// bounds check (JLS 15.10.1, 15.13.1, 15.26.1). Every expected value is java
// 24.0.2's.
class S62B {
    int v = 1;
    int get() { return v; }
}
class S62 {
    static Object[] nulls = {null};
    static Integer nn() { return (Integer) nulls[0]; }
    static int side(int v) { log = log + "S"; return v; }
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
        log = "";
        int[] base = {10, 20};
        int[] nul = null;
        k("a", () -> { int[] z = new int[nn()]; log = log + z.length; });
        k("b", () -> { log = log + base[nn()]; });
        k("c", () -> { base[nn()] = side(5); });
        k("d", () -> { int[][] z = new int[2][nn()]; log = log + z.length; });
        k("e", () -> { int[][] z = new int[nn()][side(-1)]; });
        k("f", () -> { log = log + "abc".charAt(nn()); });
        k("g", () -> { log = log + "abc".substring(0, nn()); });
        k("h", () -> { log = log + "abc".codePointAt(nn()); });
        k("i", () -> { log = log + "abc".repeat(nn()); });
        k("j", () -> { base[5] = nn(); });
        k("l", () -> { nul[side(0)] = side(1); });
        k("m", () -> { base[nn()] += 1; });
        k("n", () -> { S62B b = (S62B) nulls[0]; b.v = side(2); });
        Main.check("np62c", log.equals("a:npe b:npe c:npe d:npe e:npe f:npe g:npe h:npe i:npe j:npe SSl:npe m:npe Sn:npe "));
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
        // `q.new Inner()` names the Inner member of q's declared type.
        S88A a = new S88A();
        S88A.Inner ai = a.new Inner();
        S88A.Mid md = a.new Mid();
        S88A.Mid.Inner mi = md.new Inner();
        String s = ai.get() + " " + S88B.run() + " " + new S88B().new Inner().get() + " " + a.viaThis() + " "
            + new S88B().viaField() + " " + mi.get() + " " + md.viaOuterThis() + " " + S88B.deep();
        S88G<Integer> g = new S88G<>(7);
        s = s + " " + g.new Inner().get();
        Main.check("nt88b", s.equals("1 2 2 1 1 6 1 hi g7"));
    }
}
class S88A {
    int a = 1;
    class Inner { int get() { return a; } }
    int viaThis() { return this.new Inner().get(); }
    class Mid { int m = 5; class Inner { int get() { return m + a; } } int viaOuterThis() { return S88A.this.new Inner().get(); } }
}
class S88B {
    int b = 2;
    class Inner { int get() { return b; } }
    static int run() { S88B o = new S88B(); Inner i = o.new Inner(); return i.get(); }
    int viaField() { return f.new Inner().get(); }
    S88A f = new S88A();
    static String deep() { return new S88.Holder.Inner().w(); }
}
class S88G<T> { T v; S88G(T v) { this.v = v; } class Inner { String get() { return "g" + v; } } }

// ===== SECTION 89: collections' marker interfaces, descending views, Matcher regions =====
//
// The collections are Cloneable and Serializable; descendingMap,
// descendingKeySet and descendingSet are live views in reverse order (their
// first is the last, their floor the ceiling, their head the keys above);
// a Matcher's region bounds what it sees (opaque, anchoring), usePattern
// keeps its position and results() streams the further matches; hitEnd and
// requireEnd tell whether the last match operation read the end of the region
// and whether more text could break its match (neither cleared by reset);
// a region end that splits a surrogate pair is read across by a character
// class (which then fails, having hit the end), a greedy class loop and \X;
// \G stands where the previous operation's match ended, matches() and
// lookingAt() included, and find() goes on from the last match after a
// failed one. Every expected value is java 24.0.2's.
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
        String[] hp = {"abc", "abc", "a*", "^abc", "(?:^abc)", "abc$", "\\bab\\b", "a(?!b)", "(a)\\1", "ab|a", "\\ba", "(?m)$"};
        String[] ht = {"ab", "abcd", "aaa", "xyz", "xyz", "abc", "ab c", "a", "a", "a", "ba", "a"};
        String he = "";
        for (int i = 0; i < hp.length; i++) {
            Matcher h = Pattern.compile(hp[i]).matcher(ht[i]);
            while (h.find()) { he = he + h.start(); }
            he = he + (h.hitEnd() ? "H" : "h") + (h.requireEnd() ? "R" : "r") + (h.matches() ? "M" : "m") + (h.hitEnd() ? "H" : "h")
                + (h.requireEnd() ? "R" : "r") + " ";
        }
        Matcher hr = Pattern.compile("abc").matcher("xxabcxx");
        hr.region(2, 4);
        he = he + hr.hitEnd() + hr.find() + hr.hitEnd();
        hr.reset();
        he = he + hr.hitEnd() + hr.find() + hr.hitEnd() + hr.lookingAt() + hr.hitEnd();
        Main.check("he89", he.equals("HrmHr 0Hrmhr 03HrMHr hrmhr Hrmhr 0HrMHR 0Hrmhr 0HrMHR HrmHr 0HrMHr Hrmhr 1HRmhr "
            + "falsefalsetruetruetruefalsefalsefalse"));
        String e = "\uD83D\uDE00";
        String[] cp = {".", "\\p{L}+", "(?s).+", "\\X", "[a&&[^b]]+", "a" + e, "a.", "\\b"};
        String[] ct = {e, "a" + e, e, e, e + e, "a" + e, "a" + e, "a" + e};
        int[] ce = {1, 2, 1, 1, 3, 2, 2, 2};
        String cut = "";
        for (int i = 0; i < cp.length; i++) {
            for (int op = 0; op < 3; op++) {
                Matcher c = Pattern.compile(cp[i]).matcher(ct[i]);
                c.region(0, ce[i]);
                cut = cut + op89(c, op);
            }
            cut = cut + " ";
        }
        Main.check("cut89", cut.equals("FHFHFH T01HFHT01H T01HT01HT01H T02hFhT02h FhFhFH FHFHFH FHFHFH T00hFhT00h "));
        // Linear work on a long subject is never cut off by the matcher's step
        // limit (a pattern java.util.regex runs without recursing per iteration).
        String lng = "ab".repeat(35000);
        Matcher lm = Pattern.compile("(?>a|b)+").matcher(lng);
        Main.check("lin89", lm.find() && lm.end() == 70000);
        Matcher g = Pattern.compile("\\Ga").matcher("a");
        String gs = op89(g, 1) + op89(g, 0);
        Matcher g2 = Pattern.compile("\\G").matcher("a");
        gs = gs + op89(g2, 2) + op89(g2, 2) + op89(g2, 2);
        Matcher g3 = Pattern.compile("\\Ga*").matcher("aab");
        gs = gs + op89(g3, 2) + op89(g3, 2) + op89(g3, 2) + op89(g3, 0);
        Main.check("gp89", gs.equals("T01hFhT00hFHT00hT02hT22hFHFh"));
    }
    // lookingAt (0), matches (1) or find (2): the match's bounds and hitEnd.
    static String op89(Matcher m, int op) {
        boolean b = op == 0 ? m.lookingAt() : op == 1 ? m.matches() : m.find();
        return (b ? "T" + m.start() + m.end() : "F") + (m.hitEnd() ? "H" : "h");
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

// ===== SECTION 95: brackets after a parameter's name =====
//
// `double u[]` declares a parameter of type double[], as `double[] u` does, and
// `double[] g[]` one of type double[][] (JLS 8.4.1, 10.2); stores into it convert
// to the element type. Every expected value is java 24.0.2's.
class S95 {
    double dv95[];
    S95(double v[]) { dv95 = v; dv95[1] = 4; }
    static void add(double u[], double au[]) { au[0] = 0; au[0] += u[0]; au[1] = 7; }
    static int sum(int a[][], int n) { int s = 0; for (int i = 0; i < n; i++) s += a[i][0]; return s; }
    static String mix(int[] a, final double b[], int c) { b[0] = a[0] + c; return b[0] + " " + c; }
    static String two(double[] g[]) { g[0][0] = 1; return "" + g[0][0]; }
    static void run() {
        double[] x = {2.5, 0.0};
        double[] y = {0, 0};
        add(x, y);
        int[][] m = {{1}, {2}};
        String st = x[0] + " " + y[0] + " " + y[1] + " " + sum(m, 2) + " " + mix(new int[]{3}, new double[1], 4)
            + " " + two(new double[1][1]) + " " + new S95(new double[2]).dv95[1];
        Main.check("pd95", st.equals("2.5 2.5 7.0 3 7.0 4 1.0 4.0"));
    }
}

// ===== SECTION 96: fields of one name and different types in different classes =====
//
// A store converts to the type its own class (or a superclass, or an enclosing
// class) declares the field with, not to another class' field of that name
// (JLS 5.2, 6.5.6.1); String.codePointBefore reads the code point ending
// before an index. Every expected value is java 24.0.2's.
class S96 {
    double sf96;
    static class A96 { double d96 = 2.5; double x96; long[] arr96 = new long[2]; }
    static class B96 {
        double[] d96 = new double[3];
        int x96;
        double[] arr96 = new double[2];
        String set() { d96[1] = 4; this.d96[2] = 5; return d96[1] + " " + d96[2]; }
    }
    static class C96 extends B96 { long x96; String set2() { d96[0] = 7; x96 = 5; return d96[0] + " " + x96; } }
    static class St96 { static int sf96; static double ssf96; }
    static double ssf96;
    static class H96 { A96 a = new A96(); B96 b = new B96(); A96 self() { return a; } }
    class In96 { String s() { sf96 = 9; return "" + sf96; } }
    static void run() {
        A96 a = new A96();
        a.x96 = 1;
        a.arr96[1] = 2;
        B96 b = new B96();
        b.x96 = 3;
        b.arr96[1] = 4;
        H96 h = new H96();
        h.b.arr96[0] = 11;
        h.a.x96 = 13;
        h.self().x96 = 14;
        St96.ssf96 = 6;
        ssf96 = 8;
        double t = 0;
        for (double v : b.arr96) { t += v; }
        String st = b.set() + " " + new C96().set2() + " " + a.d96 + " " + a.x96 + " " + a.arr96[1] + " " + b.x96
            + " " + b.arr96[1] + " " + h.b.arr96[0] + " " + h.a.x96 + " " + St96.ssf96 + " " + ssf96 + " " + t
            + " " + new S96().new In96().s();
        Main.check("fc96", st.equals("4.0 5.0 7.0 5 2.5 1.0 2 3 4.0 11.0 14.0 6.0 8.0 4.0 9.0"));
        String s = "a😀b";
        String r = s.codePointBefore(1) + " " + s.codePointBefore(3) + " " + s.codePointBefore(2) + " " + s.codePointBefore(4)
            + " " + "\uDE00\uD83D".codePointBefore(2);
        try { s.codePointBefore(0); } catch (StringIndexOutOfBoundsException e) { r = r + " " + e.getMessage(); }
        try { s.codePointBefore(5); } catch (StringIndexOutOfBoundsException e) { r = r + " " + e.getMessage(); }
        Main.check("cpb96", r.equals("97 128512 55357 98 55357 Index -1 out of bounds for length 4 Index 4 out of bounds for length 4"));
    }
}

// ===== SECTION 97: library interfaces as supertypes =====
//
// An interface may extend a library interface (Comparable<T>, Runnable) and
// give it a default body; a class reaching one through an interface, directly,
// or as an anonymous class is an instance of it (JLS 9.1.3, 15.20.2), also
// before an interface it names is declared. Every expected value is java
// 24.0.2's.
interface Shape97 extends Comparable<Shape97> {
    double area();
    default int compareTo(Shape97 o) { return Double.compare(area(), o.area()); }
}
interface Task97 extends Runnable, Comparable<Task97> { }
class Sq97 implements Shape97 {
    double s;
    Sq97(double s) { this.s = s; }
    public double area() { return s * s; }
    public String toString() { return "Sq" + (int) s; }
}
class Early97 implements Later97 {
    static boolean seen = new Early97() instanceof Iterable;
}
interface Later97 extends Iterable<Integer> {
    default Iterator<Integer> iterator() { return List.of(1, 2).iterator(); }
}
class T97 implements Task97 {
    public void run() { }
    public int compareTo(Task97 o) { return 0; }
}
class S97 {
    static void run() {
        List<Shape97> l = new ArrayList<>(List.of(new Sq97(3), new Sq97(1), new Sq97(2)));
        Collections.sort(l);
        Shape97 a = new Sq97(2);
        Comparable<Shape97> c = a;
        Main.check("lib97", (l + " " + a.compareTo(new Sq97(5)) + " " + c.compareTo(new Sq97(1))).equals("[Sq1, Sq2, Sq3] -1 1"));
        Object o = a;
        Object t = new T97();
        Object r = new Runnable() { public void run() { } };
        Object e = new Early97();
        int sum = 0;
        for (int x : (Later97) e) { sum += x; }
        Main.check("libis97", (o instanceof Comparable) && (t instanceof Comparable) && (t instanceof Runnable)
            && (r instanceof Runnable) && !(r instanceof Comparable) && Early97.seen && (e instanceof Iterable) && sum == 3);
    }
}
