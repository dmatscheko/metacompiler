/* Java try/catch/finally/throw, genuinely executed (interpreter and compiler).
 *
 * throw raises a value that unwinds through calls to the nearest catch; catch binds
 * it (the first catch clause wins - exception types, including multi-catch, are parsed
 * but not discriminated); finally always runs. A return/break/continue that leaves a
 * try or catch body works in both engines. An uncaught throw is a clean runtime error.
 *
 * Main.main counts failed checks and exits with that count, so the run exits 0 exactly
 * when everything works; the interpreter and compiler must agree. **/

class BoomException {
    int code;
    BoomException(int c) { this.code = c; }
}

public class Main {
    static int fails = 0;

    static int risky(int n) {
        if (n > 3) { throw new BoomException(n); }
        return n * 2;
    }

    // return out of a try, and out of a catch.
    static int classify(int n) {
        try {
            if (n > 0) { return n * 10; }
            throw new BoomException(0);
        } catch (Exception e) {
            return -1;
        } finally { }
    }

    // A return out of an INNER try propagates through the OUTER try.
    static int nestedReturn() {
        try {
            try { return 9; } finally { }
        } finally { }
        return 0;
    }

    // break / continue leaving a try body inside a loop.
    static int loopBreak() {
        int sum = 0;
        for (int i = 0; i < 10; i = i + 1) {
            try { if (i == 3) { break; } sum = sum + i; } finally { }
        }
        return sum;             // 0+1+2 = 3
    }
    static int loopContinue() {
        int sum = 0;
        for (int i = 0; i < 5; i = i + 1) {
            try { if (i == 2) { continue; } sum = sum + i; } catch (Exception e) { }
        }
        return sum;             // 0+1+3+4 = 8
    }

    static void check(int got, int want) {
        if (got != want) { Main.fails = Main.fails + 1; }
    }

    public static void main(String[] args) {
        String log = "";
        try {
            log = log + "a";
            throw new BoomException(1);
        } catch (Exception e) {
            log = log + "b";
        } finally {
            log = log + "c";
        }
        if (!log.equals("abc")) { Main.fails = Main.fails + 1; }

        int caught = -1;
        try {
            Main.risky(5);
            Main.fails = Main.fails + 1;     // not reached
        } catch (Exception e) {
            caught = e.code;
        }
        Main.check(caught, 5);
        Main.check(Main.risky(2), 4);

        Main.check(Main.classify(4), 40);
        Main.check(Main.classify(-1), -1);
        Main.check(Main.nestedReturn(), 9);
        Main.check(Main.loopBreak(), 3);
        Main.check(Main.loopContinue(), 8);

        if (Main.fails == 0) { System.out.println("Java try/catch OK"); }
        System.exit(Main.fails);
    }
}
