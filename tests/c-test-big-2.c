/* C subset big test 2 -- recursion and control flow.
 * Theme: deep and mutual recursion (factorial, Fibonacci naive and memoized,
 * Ackermann, Euclid and extended Euclid, fast exponentiation, Towers of Hanoi,
 * McCarthy 91, is_even/is_odd), number-theory routines, and intricate control
 * flow (nested loops with break/continue, a switch-driven mini calculator, a
 * tiny state machine). Every result is checked; main() returns the failure
 * count, so the run exits 0 on success. Identical under both C engines. **/

int putchar(int c);

int nfail = 0;

int check(int got, int want) {
    if (got != want) {
        nfail++;
        putchar('F');
        putchar('0' + (nfail % 10));
        putchar('\n');
    }
    return got == want;
}

/* ---- plain recursion ---- */

int factorial(int n) {
    if (n <= 1) { return 1; }
    return n * factorial(n - 1);
}

int fib(int n) {                    /* exponential naive recursion */
    if (n < 2) { return n; }
    return fib(n - 1) + fib(n - 2);
}

int memo[64];
int memo_set[64];
int fib_memo(int n) {               /* memoized through globals */
    if (n < 2) { return n; }
    if (memo_set[n]) { return memo[n]; }
    int v = fib_memo(n - 1) + fib_memo(n - 2);
    memo[n] = v;
    memo_set[n] = 1;
    return v;
}
int fib_iter(int n) {               /* independent cross-check */
    int a = 0;
    int b = 1;
    int i;
    for (i = 0; i < n; i++) {
        int t = a + b;
        a = b;
        b = t;
    }
    return a;
}

int ackermann(int m, int n) {       /* nested recursion */
    if (m == 0) { return n + 1; }
    if (n == 0) { return ackermann(m - 1, 1); }
    return ackermann(m - 1, ackermann(m, n - 1));
}

int mccarthy91(int n) {             /* M(n) = 91 for n <= 100 */
    if (n > 100) { return n - 10; }
    return mccarthy91(mccarthy91(n + 11));
}

/* ---- mutual recursion ---- */

int is_odd(int n);
int is_even(int n) {
    if (n == 0) { return 1; }
    return is_odd(n - 1);
}
int is_odd(int n) {
    if (n == 0) { return 0; }
    return is_even(n - 1);
}

/* ---- number theory ---- */

int gcd_rec(int a, int b) {
    if (b == 0) { return a; }
    return gcd_rec(b, a % b);
}
int gcd_iter(int a, int b) {
    while (b != 0) {
        int t = a % b;
        a = b;
        b = t;
    }
    return a;
}
int lcm(int a, int b) { return a / gcd_rec(a, b) * b; }

int ext_gcd(int a, int b, int *x, int *y) {     /* pointer out-params */
    if (b == 0) { *x = 1; *y = 0; return a; }
    int x1;
    int y1;
    int g = ext_gcd(b, a % b, &x1, &y1);
    *x = y1;
    *y = x1 - (a / b) * y1;
    return g;
}

int pow_fast(int base, int exp) {   /* exponentiation by squaring */
    if (exp == 0) { return 1; }
    int half = pow_fast(base, exp / 2);
    int sq = half * half;
    if (exp % 2 == 1) { return sq * base; }
    return sq;
}

int hanoi(int n) {                  /* number of moves = 2^n - 1 */
    if (n == 0) { return 0; }
    return hanoi(n - 1) + 1 + hanoi(n - 1);
}

int choose(int n, int k) {          /* Pascal's recurrence */
    if (k == 0 || k == n) { return 1; }
    if (k < 0 || k > n) { return 0; }
    return choose(n - 1, k - 1) + choose(n - 1, k);
}

int sum_digits(int n) {
    if (n < 0) { n = -n; }
    int s = 0;
    while (n > 0) {
        s += n % 10;
        n /= 10;
    }
    return s;
}
int reverse_number(int n) {
    int r = 0;
    while (n > 0) {
        r = r * 10 + n % 10;
        n /= 10;
    }
    return r;
}
int is_palindrome_number(int n) { return reverse_number(n) == n; }

int count_bits(int n) {             /* popcount by recursion, n treated as unsigned-ish */
    if (n == 0) { return 0; }
    return (n & 1) + count_bits((n >> 1) & 2147483647);
}

int collatz_steps(int n) {          /* steps to reach 1 */
    int steps = 0;
    while (n != 1) {
        if (n % 2 == 0) { n = n / 2; }
        else { n = 3 * n + 1; }
        steps++;
    }
    return steps;
}

/* ---- switch-driven mini calculator ---- */

int apply_op(int a, int b, int op) {
    switch (op) {
    case '+': return a + b;
    case '-': return a - b;
    case '*': return a * b;
    case '/': return a / b;
    case '%': return a % b;
    default:  return 0;
    }
}

/* ---- a tiny state machine: count balanced-ish transitions over a code array ---- */
/* states: 0 idle, 1 running, 2 paused. events encoded as ints. Returns final state. */
int step_state(int state, int event) {
    switch (state) {
    case 0:                 /* idle */
        switch (event) {
        case 1: return 1;   /* start -> running */
        default: return 0;
        }
    case 1:                 /* running */
        switch (event) {
        case 2: return 2;   /* pause -> paused */
        case 3: return 0;   /* stop  -> idle */
        default: return 1;
        }
    case 2:                 /* paused */
        switch (event) {
        case 1: return 1;   /* resume -> running */
        case 3: return 0;   /* stop   -> idle */
        default: return 2;
        }
    default:
        return state;
    }
}

int main(void) {
    int i;
    int j;

    /* --- plain recursion --- */
    check(factorial(0), 1);
    check(factorial(5), 120);
    check(factorial(7), 5040);

    check(fib(0), 0);
    check(fib(1), 1);
    check(fib(10), 55);
    check(fib(15), 610);
    check(fib_memo(20), 6765);
    check(fib_memo(30), 832040);
    check(fib_memo(25), fib_iter(25));      /* two independent methods agree */
    check(fib(20), fib_iter(20));

    check(ackermann(0, 0), 1);
    check(ackermann(2, 3), 9);
    check(ackermann(3, 3), 61);
    check(mccarthy91(50), 91);
    check(mccarthy91(99), 91);
    check(mccarthy91(101), 91);
    check(mccarthy91(200), 190);

    /* --- mutual recursion --- */
    check(is_even(10), 1);
    check(is_even(7), 0);
    check(is_odd(7), 1);
    check(is_odd(10), 0);
    check(is_even(0), 1);

    /* --- number theory --- */
    check(gcd_rec(48, 36), 12);
    check(gcd_iter(48, 36), 12);
    check(gcd_rec(17, 5), 1);
    check(lcm(4, 6), 12);
    check(lcm(21, 6), 42);

    int gx;
    int gy;
    int g = ext_gcd(240, 46, &gx, &gy);
    check(g, 2);
    check(240 * gx + 46 * gy, 2);           /* Bezout identity holds */
    int g2 = ext_gcd(30, 12, &gx, &gy);
    check(g2, 6);
    check(30 * gx + 12 * gy, 6);

    check(pow_fast(2, 0), 1);
    check(pow_fast(2, 10), 1024);
    check(pow_fast(3, 5), 243);
    check(pow_fast(2, 20), 1048576);
    check(pow_fast(7, 3), 343);

    check(hanoi(0), 0);
    check(hanoi(1), 1);
    check(hanoi(5), 31);
    check(hanoi(10), 1023);

    check(choose(0, 0), 1);
    check(choose(5, 2), 10);
    check(choose(10, 5), 252);
    check(choose(10, 3), 120);
    check(choose(6, 7), 0);

    check(sum_digits(0), 0);
    check(sum_digits(12345), 15);
    check(sum_digits(-987), 24);
    check(reverse_number(12345), 54321);
    check(reverse_number(1200), 21);
    check(is_palindrome_number(12321), 1);
    check(is_palindrome_number(12345), 0);
    check(is_palindrome_number(7), 1);

    check(count_bits(0), 0);
    check(count_bits(7), 3);
    check(count_bits(255), 8);
    check(count_bits(1024), 1);

    check(collatz_steps(1), 0);
    check(collatz_steps(6), 8);
    check(collatz_steps(27), 111);

    /* --- switch calculator --- */
    check(apply_op(6, 3, '+'), 9);
    check(apply_op(6, 3, '-'), 3);
    check(apply_op(6, 3, '*'), 18);
    check(apply_op(7, 3, '/'), 2);
    check(apply_op(7, 3, '%'), 1);
    check(apply_op(1, 1, '?'), 0);          /* default */

    /* --- nested state machine over an event stream --- */
    int events[8];
    events[0] = 1; events[1] = 2; events[2] = 1; events[3] = 3;
    events[4] = 1; events[5] = 5; events[6] = 2; events[7] = 3;
    int state = 0;
    int running_seen = 0;
    for (i = 0; i < 8; i++) {
        state = step_state(state, events[i]);
        if (state == 1) { running_seen++; }
    }
    check(state, 0);                        /* ends idle after final stop */
    check(running_seen, 4);                 /* steps ending in the running state */

    /* --- nested loops with break/continue: count primes with a sieve-ish trial --- */
    int prime_count = 0;
    int last_prime = 0;
    for (i = 2; i <= 50; i++) {
        int is_prime = 1;
        for (j = 2; j * j <= i; j++) {
            if (i % j == 0) { is_prime = 0; break; }
        }
        if (!is_prime) { continue; }
        prime_count++;
        last_prime = i;
    }
    check(prime_count, 15);                 /* primes below 50 */
    check(last_prime, 47);

    /* --- a pass-through label plus a do/while accumulator --- */
    int acc = 0;
    int n = 1;
loop_body:                                  /* label, never jumped to */
    do {
        acc += n;
        n++;
    } while (n <= 10);
    check(acc, 55);

    /* --- deeply nested ternary chain (grade buckets) --- */
    int score = 83;
    int grade = score >= 90 ? 4 : (score >= 80 ? 3 : (score >= 70 ? 2 : (score >= 60 ? 1 : 0)));
    check(grade, 3);

    if (nfail == 0) {
        putchar('O'); putchar('K'); putchar('\n');
    }
    return nfail;
}
