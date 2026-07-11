/* Java subset self test (big 2): recursion, number theory and control flow.
 *
 * A breadth-first tour of the control-flow and recursion features of the subset,
 * all as static methods returning ints/booleans (no allocation beyond a couple of
 * fixed array literals that drive loops):
 *   - classic recursion: factorial, Fibonacci (recursive vs iterative agreement),
 *     Euclid's gcd/lcm, Ackermann, Towers-of-Hanoi move counts, fast exponentiation,
 *     recursive digit sum, C(n,k) via Pascal's identity
 *   - mutual recursion: isEven / isOdd
 *   - iterative control flow: Collatz step counts, integer reversal, popcount by
 *     repeated division, primality by trial division, prime counting (nested loops),
 *     a Pythagorean-triple search (triple-nested loops)
 *   - a small turnstile finite-state machine driven by an event array, implemented
 *     with nested switch/fallthrough
 *   - a switch with fallthrough and stacked labels
 * Each result is compared against an independently known value; Main.main exits with
 * the failure count, so the run exits 0 exactly when every computation is correct. **/

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

    // ----- classic recursion -----

    static int fact(int n) {
        if (n <= 1) {
            return 1;
        }
        return n * Main.fact(n - 1);
    }

    static int fib(int n) {
        if (n < 2) {
            return n;
        }
        return Main.fib(n - 1) + Main.fib(n - 2);
    }

    static int fibIter(int n) {
        if (n < 2) {
            return n;
        }
        int a = 0, b = 1;
        for (int i = 2; i <= n; i++) {
            int c = a + b;
            a = b;
            b = c;
        }
        return b;
    }

    static int gcd(int a, int b) {
        if (b == 0) {
            return a;
        }
        return Main.gcd(b, a % b);
    }

    static int lcm(int a, int b) {
        return a / Main.gcd(a, b) * b;
    }

    static int ack(int m, int n) {
        if (m == 0) {
            return n + 1;
        }
        if (n == 0) {
            return Main.ack(m - 1, 1);
        }
        return Main.ack(m - 1, Main.ack(m, n - 1));
    }

    static int hanoi(int n) {
        if (n == 0) {
            return 0;
        }
        return 2 * Main.hanoi(n - 1) + 1;
    }

    static int powi(int base, int exp) {
        if (exp == 0) {
            return 1;
        }
        int half = Main.powi(base, exp / 2);
        if (exp % 2 == 0) {
            return half * half;
        }
        return half * half * base;
    }

    static int sumDigits(int n) {
        if (n < 10) {
            return n;
        }
        return n % 10 + Main.sumDigits(n / 10);
    }

    static int choose(int n, int k) {
        if (k == 0 || k == n) {
            return 1;
        }
        return Main.choose(n - 1, k - 1) + Main.choose(n - 1, k);
    }

    // ----- mutual recursion -----

    static boolean isEven(int n) {
        if (n == 0) {
            return true;
        }
        return Main.isOdd(n - 1);
    }

    static boolean isOdd(int n) {
        if (n == 0) {
            return false;
        }
        return Main.isEven(n - 1);
    }

    // ----- iterative control flow -----

    static int collatz(int n) {
        int steps = 0;
        while (n != 1) {
            if (n % 2 == 0) {
                n = n / 2;
            } else {
                n = 3 * n + 1;
            }
            steps++;
        }
        return steps;
    }

    static int reverseInt(int n) {
        int r = 0;
        while (n > 0) {
            r = r * 10 + n % 10;
            n = n / 10;
        }
        return r;
    }

    static int popcount(int n) {
        int c = 0;
        while (n > 0) {
            c += n % 2;
            n = n / 2;
        }
        return c;
    }

    static boolean isPrime(int x) {
        if (x < 2) {
            return false;
        }
        for (int i = 2; i * i <= x; i++) {
            if (x % i == 0) {
                return false;
            }
        }
        return true;
    }

    static int countPrimes(int n) {                     // primes strictly below n
        int c = 0;
        for (int i = 2; i < n; i++) {
            if (Main.isPrime(i)) {
                c++;
            }
        }
        return c;
    }

    static int pythagCount(int limit) {                 // triples a<b<=c<=limit
        int count = 0;
        for (int a = 1; a <= limit; a++) {
            for (int b = a + 1; b <= limit; b++) {
                for (int c = b; c <= limit; c++) {
                    if (a * a + b * b == c * c) {
                        count++;
                    }
                }
            }
        }
        return count;
    }

    // ----- a turnstile finite-state machine (nested switch) -----
    // states: 0 locked, 1 unlocked; events: 0 push, 1 coin. Returns how many times
    // a push on the unlocked state opened the gate.
    static int turnstile(int[] events) {
        int state = 0;
        int opens = 0;
        for (var e : events) {
            switch (state) {
            case 0:                                     // locked
                switch (e) {
                case 1:
                    state = 1;                          // coin unlocks
                    break;
                default:
                    break;                              // push: stays locked
                }
                break;
            default:                                    // unlocked
                switch (e) {
                case 0:
                    opens++;                            // push opens then relocks
                    state = 0;
                    break;
                default:
                    break;                              // coin: stays unlocked
                }
            }
        }
        return opens;
    }

    // ----- a switch with fallthrough and stacked labels -----
    static int band(int score) {
        int r = 0;
        switch (score / 10) {
        case 10:
        case 9:
            r = 4;                                      // A
            break;
        case 8:
            r = 3;                                      // B
            break;
        case 7:
            r = 2;                                      // C
            break;
        case 6:
            r = 1;                                      // D
            break;
        default:
            r = 0;                                      // F
        }
        return r;
    }

    public static void main(String[] args) {
        // factorial and Fibonacci
        Main.check("fact 0", Main.fact(0), 1);
        Main.check("fact 6", Main.fact(6), 720);
        Main.check("fact 10", Main.fact(10), 3628800);
        Main.check("fib 10", Main.fib(10), 55);
        Main.check("fib 15", Main.fib(15), 610);
        Main.check("fib iter matches", Main.fibIter(15), Main.fib(15));
        boolean fibAgree = true;
        for (int i = 0; i <= 18; i++) {
            if (Main.fib(i) != Main.fibIter(i)) {
                fibAgree = false;
            }
        }
        Main.checkB("fib recursive == iterative", fibAgree, true);

        // gcd / lcm
        Main.check("gcd 48,36", Main.gcd(48, 36), 12);
        Main.check("gcd 17,5", Main.gcd(17, 5), 1);
        Main.check("gcd coprime pow", Main.gcd(1071, 462), 21);
        Main.check("lcm 4,6", Main.lcm(4, 6), 12);
        Main.check("lcm 21,6", Main.lcm(21, 6), 42);

        // Ackermann (kept small)
        Main.check("ack 0,0", Main.ack(0, 0), 1);
        Main.check("ack 2,3", Main.ack(2, 3), 9);
        Main.check("ack 3,3", Main.ack(3, 3), 61);

        // Towers of Hanoi move counts (2^n - 1)
        Main.check("hanoi 1", Main.hanoi(1), 1);
        Main.check("hanoi 4", Main.hanoi(4), 15);
        Main.check("hanoi 10", Main.hanoi(10), 1023);

        // fast exponentiation
        Main.check("pow 2^0", Main.powi(2, 0), 1);
        Main.check("pow 2^10", Main.powi(2, 10), 1024);
        Main.check("pow 3^7", Main.powi(3, 7), 2187);
        Main.check("pow 7^3", Main.powi(7, 3), 343);

        // recursive digit sum
        Main.check("digitsum 9", Main.sumDigits(9), 9);
        Main.check("digitsum 9875", Main.sumDigits(9875), 29);
        Main.check("digitsum 100000", Main.sumDigits(100000), 1);

        // binomial via Pascal's identity
        Main.check("C(6,2)", Main.choose(6, 2), 15);
        Main.check("C(10,5)", Main.choose(10, 5), 252);
        Main.check("C(n,0)", Main.choose(7, 0), 1);
        Main.check("C(n,n)", Main.choose(7, 7), 1);

        // mutual recursion parity
        Main.checkB("isEven 0", Main.isEven(0), true);
        Main.checkB("isEven 10", Main.isEven(10), true);
        Main.checkB("isOdd 7", Main.isOdd(7), true);
        Main.checkB("isEven 7", Main.isEven(7), false);
        int parityHits = 0;
        for (int i = 0; i < 12; i++) {
            if (Main.isEven(i) == (i % 2 == 0)) {
                parityHits++;
            }
        }
        Main.check("parity agrees 12x", parityHits, 12);

        // iterative control flow
        Main.check("collatz 1", Main.collatz(1), 0);
        Main.check("collatz 6", Main.collatz(6), 8);
        Main.check("collatz 27", Main.collatz(27), 111);
        Main.check("reverse 12345", Main.reverseInt(12345), 54321);
        Main.check("reverse 1000", Main.reverseInt(1000), 1);
        Main.check("popcount 255", Main.popcount(255), 8);
        Main.check("popcount 1024", Main.popcount(1024), 1);
        Main.check("popcount 0", Main.popcount(0), 0);

        // primality and counting (nested loops)
        Main.checkB("isPrime 2", Main.isPrime(2), true);
        Main.checkB("isPrime 97", Main.isPrime(97), true);
        Main.checkB("isPrime 91", Main.isPrime(91), false);
        Main.check("primes below 30", Main.countPrimes(30), 10);
        Main.check("primes below 100", Main.countPrimes(100), 25);

        // triple-nested loop: Pythagorean triples up to 20
        Main.check("pythag <=20", Main.pythagCount(20), 6);

        // finite-state machine
        int[] events = new int[]{0, 1, 0, 0, 1, 1, 0};
        Main.check("turnstile opens", Main.turnstile(events), 2);
        int[] noCoin = new int[]{0, 0, 0};
        Main.check("turnstile no coin", Main.turnstile(noCoin), 0);

        // switch with fallthrough and stacked labels
        Main.check("band 100", Main.band(100), 4);
        Main.check("band 95", Main.band(95), 4);
        Main.check("band 83", Main.band(83), 3);
        Main.check("band 71", Main.band(71), 2);
        Main.check("band 60", Main.band(60), 1);
        Main.check("band 42", Main.band(42), 0);

        if (Main.fails == 0) {
            System.out.println("Java big-2 (recursion + control flow) passed");
        }
        System.exit(Main.fails);
    }
}
