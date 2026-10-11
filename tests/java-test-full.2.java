// Full-syntax test: Java (Java SE 21 core grammar) - part 2 of 2, section 49.
//
// THE JAVA RATCHET IS TWO FILES, so that neither -frozen interpreter leg comes
// near the IR interpreter's step brake (-max-steps, 100M per call). Section 49
// (a 3000-iteration StringBuilder loop) needs more than half of that budget alone,
// so it has a file of its own:
//
//   tests/java-test-full.java     every other section
//   tests/java-test-full.2.java   section 49
//
// The section NUMBERS are global (grammar comments cite them), the two files
// together run the assertions, and ./test.sh --full reports them as the units
// java and java2. Add a new section to tests/java-test-full.java, and keep the
// numbers globally unique across the two.
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
//     helper and main - Main comes first so `java tests/java-test-full.2.java`
//     (the single-file source launcher) picks it as the class to run
//   - each section: '// ===== SECTION <nn>: <name> =====', a self-contained
//     group of top-level types, no references to other sections
//   - main calls each section via a line tagged 'SECTION-CALL <nn>'; the calls
//     are S<nn>.run()-qualified because the subset resolves statics by class
//   - main prints the summary line 'full: <checks> checks, <failures> failures'
//     and exits with the failure count (exit 0 == full support, verified)
//
// Deliberately out of scope (not core syntax, or unrunnable in this harness):
// packages, and the standard library beyond the java.lang implicits (String,
// Math, StrictMath, System and friends) and the java.util list types imported
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
import java.util.Iterator;
import java.util.List;

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
        S49.run(); // SECTION-CALL 49
        System.out.println("full: " + Main.checks + " checks, " + Main.failures + " failures");
        System.exit(Main.failures);
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
    }
}

