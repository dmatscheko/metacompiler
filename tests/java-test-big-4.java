/* Java subset self test (big 4): string and number processing.
 *
 * Text and number crunching built from the String methods the subset provides
 * (length/charAt/substring/indexOf/equals/isEmpty) and int arithmetic only - there
 * is no char type, so character codes are obtained by indexOf into an alphabet:
 *   - parseInt / intToStr (signed), cross-checked against the built-in "" + n, plus
 *     round-tripping parseInt(intToStr(n)) == n
 *   - string reversal, palindrome testing, character counting
 *   - base conversion to binary and hex
 *   - a Caesar cipher (encrypt/decrypt round-trip) via alphabet indexing
 *   - run-length encode/decode round-tripping
 *   - number theory: gcd/lcm, primality by trial division, prime-factor strings,
 *     digit sums and digit counts
 *   - a recursive-descent arithmetic evaluator (a Parser object holding the source
 *     and a cursor) honouring +-*_/ precedence, parentheses, unary minus and spaces
 * Each result is checked against an independently known value; Main.main exits with
 * the failure count, so the run exits 0 exactly when every result is correct. **/

// A recursive-descent expression evaluator. The parser state (source + cursor)
// lives on the object; the grammar is expr -> term (('+'|'-') term)*, term ->
// factor (('*'|'/') factor)*, factor -> '(' expr ')' | '-' factor | number.
class Parser {
    String src;
    int pos;

    Parser(String s) {
        this.src = s;
        this.pos = 0;
    }

    String cur() {
        if (this.pos >= this.src.length()) {
            return "";
        }
        return this.src.substring(this.pos, this.pos + 1);
    }

    void skip() {
        while (this.cur().equals(" ")) {
            this.pos++;
        }
    }

    int parseExpr() {
        int v = this.parseTerm();
        this.skip();
        while (this.cur().equals("+") || this.cur().equals("-")) {
            String op = this.cur();
            this.pos++;
            int r = this.parseTerm();
            if (op.equals("+")) {
                v = v + r;
            } else {
                v = v - r;
            }
            this.skip();
        }
        return v;
    }

    int parseTerm() {
        int v = this.parseFactor();
        this.skip();
        while (this.cur().equals("*") || this.cur().equals("/")) {
            String op = this.cur();
            this.pos++;
            int r = this.parseFactor();
            if (op.equals("*")) {
                v = v * r;
            } else {
                v = v / r;
            }
            this.skip();
        }
        return v;
    }

    int parseFactor() {
        this.skip();
        if (this.cur().equals("(")) {
            this.pos++;
            int v = this.parseExpr();
            this.skip();
            this.pos++;                                 // consume ')'
            return v;
        }
        if (this.cur().equals("-")) {
            this.pos++;
            return -this.parseFactor();
        }
        return this.parseNumber();
    }

    int parseNumber() {
        this.skip();
        int n = 0;
        while (true) {
            String c = this.cur();
            if (c.isEmpty()) {                          // end of input
                break;
            }
            int d = "0123456789".indexOf(c);
            if (d < 0) {                                // non-digit stops the number
                break;
            }
            n = n * 10 + d;
            this.pos++;
        }
        return n;
    }
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

    static final String DIGITS = "0123456789";
    static final String HEXDIGITS = "0123456789abcdef";
    static final String ALPHA = "abcdefghijklmnopqrstuvwxyz";

    // ----- string <-> int -----

    static int parseInt(String s) {
        int i = 0;
        boolean neg = false;
        if (s.substring(0, 1).equals("-")) {
            neg = true;
            i = 1;
        }
        int n = 0;
        while (i < s.length()) {
            int d = Main.DIGITS.indexOf(s.substring(i, i + 1));
            n = n * 10 + d;
            i++;
        }
        return neg ? -n : n;
    }

    static String intToStr(int n) {
        if (n == 0) {
            return "0";
        }
        boolean neg = n < 0;
        if (neg) {
            n = -n;
        }
        String r = "";
        while (n > 0) {
            int d = n % 10;
            r = Main.DIGITS.substring(d, d + 1) + r;
            n = n / 10;
        }
        return neg ? "-" + r : r;
    }

    // ----- string utilities -----

    static String reverse(String s) {
        String r = "";
        for (int i = s.length() - 1; i >= 0; i--) {
            r = r + s.substring(i, i + 1);
        }
        return r;
    }

    static boolean isPalindrome(String s) {
        int i = 0;
        int j = s.length() - 1;
        while (i < j) {
            if (!s.substring(i, i + 1).equals(s.substring(j, j + 1))) {
                return false;
            }
            i++;
            j--;
        }
        return true;
    }

    static int countChar(String s, String ch) {
        int c = 0;
        for (int i = 0; i < s.length(); i++) {
            if (s.substring(i, i + 1).equals(ch)) {
                c++;
            }
        }
        return c;
    }

    // ----- base conversion -----

    static String toBinary(int n) {
        if (n == 0) {
            return "0";
        }
        String r = "";
        while (n > 0) {
            int d = n % 2;
            r = Main.DIGITS.substring(d, d + 1) + r;
            n = n / 2;
        }
        return r;
    }

    static String toHex(int n) {
        if (n == 0) {
            return "0";
        }
        String r = "";
        while (n > 0) {
            int d = n % 16;
            r = Main.HEXDIGITS.substring(d, d + 1) + r;
            n = n / 16;
        }
        return r;
    }

    // ----- Caesar cipher (encrypt with +shift, decrypt with -shift) -----

    static String caesar(String s, int shift) {
        String r = "";
        for (int i = 0; i < s.length(); i++) {
            String ch = s.substring(i, i + 1);
            int idx = Main.ALPHA.indexOf(ch);
            if (idx < 0) {
                r = r + ch;                             // punctuation/space passes through
            } else {
                int j = ((idx + shift) % 26 + 26) % 26;
                r = r + Main.ALPHA.substring(j, j + 1);
            }
        }
        return r;
    }

    // ----- run-length encoding -----

    static String rleEncode(String s) {
        String r = "";
        int i = 0;
        while (i < s.length()) {
            String ch = s.substring(i, i + 1);
            int count = 1;
            while (i + count < s.length() && s.substring(i + count, i + count + 1).equals(ch)) {
                count++;
            }
            r = r + count + ch;
            i = i + count;
        }
        return r;
    }

    static String rleDecode(String s) {
        String r = "";
        int i = 0;
        while (i < s.length()) {
            int count = 0;
            while (i < s.length() && Main.DIGITS.indexOf(s.substring(i, i + 1)) >= 0) {
                count = count * 10 + Main.DIGITS.indexOf(s.substring(i, i + 1));
                i++;
            }
            String ch = s.substring(i, i + 1);
            i++;
            for (int k = 0; k < count; k++) {
                r = r + ch;
            }
        }
        return r;
    }

    // ----- number theory -----

    static int gcd(int a, int b) {
        while (b != 0) {
            int t = a % b;
            a = b;
            b = t;
        }
        return a;
    }

    static int lcm(int a, int b) {
        return a / Main.gcd(a, b) * b;
    }

    static boolean isPrime(int n) {
        if (n < 2) {
            return false;
        }
        for (int i = 2; i * i <= n; i++) {
            if (n % i == 0) {
                return false;
            }
        }
        return true;
    }

    static String primeFactors(int n) {
        String r = "";
        int d = 2;
        while (d * d <= n) {
            while (n % d == 0) {
                if (!r.isEmpty()) {
                    r = r + "*";
                }
                r = r + d;
                n = n / d;
            }
            d++;
        }
        if (n > 1) {
            if (!r.isEmpty()) {
                r = r + "*";
            }
            r = r + n;
        }
        return r;
    }

    static int digitSum(int n) {
        if (n < 0) {
            n = -n;
        }
        int s = 0;
        while (n > 0) {
            s += n % 10;
            n = n / 10;
        }
        return s;
    }

    static int numDigits(int n) {
        if (n < 0) {
            n = -n;
        }
        if (n == 0) {
            return 1;
        }
        int c = 0;
        while (n > 0) {
            c++;
            n = n / 10;
        }
        return c;
    }

    static int evalExpr(String s) {
        Parser p = new Parser(s);
        return p.parseExpr();
    }

    public static void main(String[] args) {
        // ----- parseInt / intToStr, cross-checked against "" + n -----
        Main.check("parseInt 42", Main.parseInt("42"), 42);
        Main.check("parseInt -17", Main.parseInt("-17"), -17);
        Main.check("parseInt 1000", Main.parseInt("1000"), 1000);
        Main.check("parseInt 0", Main.parseInt("0"), 0);
        Main.checkS("intToStr 42", Main.intToStr(42), "42");
        Main.checkS("intToStr -17", Main.intToStr(-17), "-17");
        Main.checkS("intToStr 0", Main.intToStr(0), "0");
        int[] roundtrip = new int[]{0, 7, 42, -1, -256, 1000, 99999, -100000};
        boolean rtOk = true;
        boolean matchesBuiltin = true;
        for (var n : roundtrip) {
            if (Main.parseInt(Main.intToStr(n)) != n) {
                rtOk = false;
            }
            if (!Main.intToStr(n).equals("" + n)) {     // agree with built-in concat
                matchesBuiltin = false;
            }
        }
        Main.checkB("intToStr round-trips", rtOk, true);
        Main.checkB("intToStr == builtin concat", matchesBuiltin, true);

        // ----- string utilities -----
        Main.checkS("reverse hello", Main.reverse("hello"), "olleh");
        Main.checkS("reverse abc", Main.reverse("abc"), "cba");
        Main.checkS("reverse empty", Main.reverse(""), "");
        Main.checkB("palindrome racecar", Main.isPalindrome("racecar"), true);
        Main.checkB("palindrome abcba", Main.isPalindrome("abcba"), true);
        Main.checkB("palindrome abca", Main.isPalindrome("abca"), false);
        Main.checkB("palindrome empty", Main.isPalindrome(""), true);
        Main.check("count s in mississippi", Main.countChar("mississippi", "s"), 4);
        Main.check("count i in mississippi", Main.countChar("mississippi", "i"), 4);
        Main.check("count a in banana", Main.countChar("banana", "a"), 3);
        Main.check("count z in banana", Main.countChar("banana", "z"), 0);

        // ----- base conversion -----
        Main.checkS("bin 0", Main.toBinary(0), "0");
        Main.checkS("bin 5", Main.toBinary(5), "101");
        Main.checkS("bin 10", Main.toBinary(10), "1010");
        Main.checkS("bin 255", Main.toBinary(255), "11111111");
        Main.checkS("hex 0", Main.toHex(0), "0");
        Main.checkS("hex 16", Main.toHex(16), "10");
        Main.checkS("hex 255", Main.toHex(255), "ff");
        Main.checkS("hex 4096", Main.toHex(4096), "1000");

        // ----- Caesar cipher round-trip -----
        Main.checkS("caesar abc+3", Main.caesar("abc", 3), "def");
        Main.checkS("caesar xyz+3 wraps", Main.caesar("xyz", 3), "abc");
        String msg = "the quick brown fox";
        String enc = Main.caesar(msg, 13);
        Main.checkS("caesar decrypt restores", Main.caesar(enc, -13), msg);
        Main.checkS("rot13 twice is identity", Main.caesar(Main.caesar(msg, 13), 13), msg);
        Main.checkB("caesar actually changed it", enc.equals(msg), false);

        // ----- run-length encoding round-trip -----
        Main.checkS("rle aaabbc", Main.rleEncode("aaabbc"), "3a2b1c");
        Main.checkS("rle wwwwww", Main.rleEncode("wwwwww"), "6w");
        Main.checkS("rle abc", Main.rleEncode("abc"), "1a1b1c");
        String txt = "aaaaabbbbcccd";
        Main.checkS("rle round-trip", Main.rleDecode(Main.rleEncode(txt)), txt);

        // ----- number theory -----
        Main.check("gcd 48,36", Main.gcd(48, 36), 12);
        Main.check("gcd 1071,462", Main.gcd(1071, 462), 21);
        Main.check("lcm 4,6", Main.lcm(4, 6), 12);
        Main.check("lcm 21,6", Main.lcm(21, 6), 42);
        Main.checkB("prime 2", Main.isPrime(2), true);
        Main.checkB("prime 97", Main.isPrime(97), true);
        Main.checkB("prime 1", Main.isPrime(1), false);
        Main.checkB("prime 91", Main.isPrime(91), false);
        Main.checkS("factors 12", Main.primeFactors(12), "2*2*3");
        Main.checkS("factors 360", Main.primeFactors(360), "2*2*2*3*3*5");
        Main.checkS("factors 97 (prime)", Main.primeFactors(97), "97");
        Main.checkS("factors 17 (prime)", Main.primeFactors(17), "17");
        Main.check("digitSum 9875", Main.digitSum(9875), 29);
        Main.check("digitSum -12345", Main.digitSum(-12345), 15);
        Main.check("numDigits 0", Main.numDigits(0), 1);
        Main.check("numDigits 1000", Main.numDigits(1000), 4);
        Main.check("numDigits -70707", Main.numDigits(-70707), 5);

        // ----- recursive-descent expression evaluator -----
        Main.check("eval add-mul precedence", Main.evalExpr("2+3*4"), 14);
        Main.check("eval parens", Main.evalExpr("(2+3)*4"), 20);
        Main.check("eval left assoc sub", Main.evalExpr("10-2-3"), 5);
        Main.check("eval mixed", Main.evalExpr("2*(3+4)-5"), 9);
        Main.check("eval unary minus", Main.evalExpr("-(3+4)"), -7);
        Main.check("eval div left assoc", Main.evalExpr("100/5/2"), 10);
        Main.check("eval nested", Main.evalExpr("((1+2)*(3+4))-((5-1)*2)"), 13);
        Main.check("eval with spaces", Main.evalExpr("  2 + 3 * 4  "), 14);
        Main.check("eval multi-digit", Main.evalExpr("12*12+1"), 145);
        Main.check("eval unary in product", Main.evalExpr("-3*4"), -12);

        if (Main.fails == 0) {
            System.out.println("Java big-4 (string + number processing) passed");
        }
        System.exit(Main.fails);
    }
}
