/* C subset self test.
 * Exercises the whole language: pointers, arrays, globals, all operators,
 * control flow and recursion. main() returns the number of failed checks,
 * so the metacompiler run exits with 0 exactly when everything works. **/

int putchar(int c);                     /* Prototypes are parsed and ignored. */

int nfail = 0;
int global_counter;
int primes[10];

int check(int got, int want) {
    if (got != want) {
        nfail++;
        putchar('F');
        putchar('0' + nfail);
        putchar('\n');
    }
    return got == want;
}

int add(int a, int b) { return a + b; }

int fib(int n) {                       // recursion
    if (n < 2) { return n; }
    return fib(n - 1) + fib(n - 2);
}

int sum_array(int *a, int n) {         // pointer parameter + pointer arithmetic
    int s = 0;
    int i;
    for (i = 0; i < n; i++) {
        s += *(a + i);
    }
    return s;
}

int swap(int *x, int *y) {             // classic pointer swap
    int t = *x;
    *x = *y;
    *y = t;
    return 0;
}

int bump(void) {                       // works on a global
    global_counter += 1;
    return global_counter;
}

int classify(int x) {                  /* switch with fallthrough and default */
    int r = 0;
    switch (x) {
    case 0:
        r = 100;
        break;
    case 1:                            /* stacked labels */
    case 2:
        r = 12;
        break;
    case 3:
        r = 3;                         /* falls through into case 4 */
    case 4:
        r += 40;
        break;
    default:
        r = -1;
    }
    return r;
}

int count_primes(int limit) {          // nested loops, break/continue, arrays
    int found = 0;
    int n;
    for (n = 2; n <= limit; n++) {
        int is_prime = 1;
        int d;
        for (d = 2; d * d <= n; d++) {
            if (n % d == 0) { is_prime = 0; break; }
        }
        if (!is_prime) { continue; }
        if (found < 10) { primes[found] = n; }
        found++;
    }
    return found;
}

int main() {
    // arithmetic and precedence
    check(1 + 2 * 3, 7);
    check((1 + 2) * 3, 9);
    check(7 / 2, 3);                    // int division truncates
    check(-7 / 2, -3);                  // towards zero
    check(7 % 3, 1);
    check(-(-5), 5);
    check(10 - 3 - 2, 5);

    // bitwise and shifts
    check(5 | 2, 7);
    check(5 & 3, 1);
    check(5 ^ 1, 4);
    check(1 << 4, 16);
    check(-8 >> 1, -4);                 // arithmetic shift
    check(~0, -1);
    check(1 | 2 & 3, 3);                // & binds tighter than |

    // comparisons, ! and ternary
    check(3 < 4, 1);
    check(4 <= 4, 1);
    check(!0, 1);
    check(!7, 0);
    check(3 > 2 ? 10 : 20, 10);
    check(0 ? 10 : 20, 20);

    // short circuit evaluation
    global_counter = 0;
    int r1 = 0 && bump();
    check(global_counter, 0);           // right side skipped
    int r2 = 1 || bump();
    check(global_counter, 0);           // right side skipped
    int r3 = 1 && bump();
    check(global_counter, 1);           // right side ran
    check(r1 + r2 + r3, 2);

    // assignment as expression, compound assigns, inc/dec
    int x = 1;
    int y = (x = 5) + 1;
    check(y, 6);
    x += 4;  check(x, 9);
    x -= 2;  check(x, 7);
    x *= 3;  check(x, 21);
    x /= 2;  check(x, 10);
    x %= 3;  check(x, 1);
    check(x++, 1);
    check(x, 2);
    check(++x, 3);
    check(x--, 3);
    check(--x, 1);

    // chars
    check('A', 65);
    check('\n', 10);
    check('0' + 5, 53);

    // control flow
    int w = 0;
    while (w < 5) { w++; }
    check(w, 5);
    int dc = 0;
    do { dc++; } while (0);
    check(dc, 1);

    // functions and recursion
    check(add(2, 3), 5);
    check(fib(10), 55);

    // arrays
    int arr[5];
    int i;
    for (i = 0; i < 5; i++) { arr[i] = i * i; }
    check(arr[3], 9);
    arr[2] += 10;
    check(arr[2], 14);
    check(sum_array(arr, 5), 0 + 1 + 14 + 9 + 16);

    // pointers
    int a = 1, b = 2;
    int *p = &a;
    check(*p, 1);
    *p = 42;
    check(a, 42);
    swap(&a, &b);
    check(a, 2);
    check(b, 42);
    p = &arr[1];
    check(*(p + 1), 14);                // pointer arithmetic steps ints
    check(p[2], 9);
    *(p + 3) = 7;
    check(arr[4], 7);

    // switch
    check(classify(0), 100);
    check(classify(1), 12);
    check(classify(2), 12);
    check(classify(3), 43);
    check(classify(4), 40);
    check(classify(9), -1);

    // globals and a bigger computation
    check(count_primes(30), 10);
    check(primes[0], 2);
    check(primes[9], 29);
    check(count_primes(100), 25);

    if (nfail == 0) {
        putchar('O'); putchar('K'); putchar('\n');
    }
    return nfail;
}
