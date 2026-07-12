/* Fast feature-matrix test for the C interpreter (c-interpreter.abnf) and the
 * LLVM-IR compiler (c-to-llvm-ir.abnf). It replaces the four algorithm-themed
 * c-test-big-* stress tests: instead of large loops (sorts, heaps, sieves) every
 * implemented construct is exercised with the SMALLEST program that can prove it
 * works - loops run 0, 1, 3 or 4 times, recursion stays below depth 6, arrays
 * hold 3-5 cells. The subset has no strings, so checks are numbered in source
 * order: a failed check prints "FAIL <n>" where n is the ordinal of the check()
 * call (count calls to find it). main() returns the failure count; exit 0 and
 * byte-identical output on all four legs (interpreter/compiler x goja/-frozen)
 * mean everything passed. **/

#ifndef C_TEST_FEATURES
#define C_TEST_FEATURES
#pragma once
#define UNUSED_MACRO 999        /* object-like macro, accepted and ignored (not expanded) */
#endif

int putchar(int c);             /* Prototypes are parsed and ignored. */

int nfail = 0;
int nchecks = 0;

int print_num(int n) {          /* recursive decimal printer (n >= 0) */
    if (n >= 10) { print_num(n / 10); }
    putchar('0' + n % 10);
    return 0;
}

int check(int cond) {           /* auto-numbered: prints FAIL <ordinal> */
    nchecks++;
    if (!cond) {
        nfail++;
        putchar('F'); putchar('A'); putchar('I'); putchar('L'); putchar(' ');
        print_num(nchecks);
        putchar('\n');
    }
    return cond;
}

/* ----- globals: scalars, arrays, struct types ----- */

int global_counter = 0;
int gtable[4];                              /* zero initialized */

struct Point { int x; int y; };
struct Rect { struct Point min; struct Point max; };   /* nested struct values */
struct Node { int value; struct Node *next; };         /* self referencing pointer */

struct Point corners[3];                    /* global array of structs */
struct Node *chain;                         /* global struct pointer */

enum Color { RED, GREEN, BLUE };
enum Status { OK = 0, WARN = 5, ERR };      /* ERR continues at 6 */

/* ----- functions ----- */

int add(int a, int b) { return a + b; }

int bump(void) { global_counter += 1; return global_counter; }

int early(int n) {                          /* early return */
    if (n < 0) { return -1; }
    return 1;
}

int fib(int n) {                            /* recursion, depth <= 6 */
    if (n < 2) { return n; }
    return fib(n - 1) + fib(n - 2);
}

int is_odd(int n);                          /* forward prototype for mutual recursion */
int is_even(int n) { return n == 0 ? 1 : is_odd(n - 1); }
int is_odd(int n)  { return n == 0 ? 0 : is_even(n - 1); }

int swap(int *x, int *y) {                  /* classic pointer swap */
    int t = *x;
    *x = *y;
    *y = t;
    return 0;
}

void noop(void) { }                         /* void return type */
int nonnull(void *p) { return p != 0; }     /* void* parameter */

int classify(int x) {                       /* switch: stacked labels, fallthrough, default */
    int r = 0;
    switch (x) {
    case 0:
        r = 100;
        break;
    case 1:                                 /* stacked labels */
    case 2:
        r = 12;
        break;
    case 3:
        r = 3;                              /* falls through into case 4 */
    case 4:
        r += 40;
        break;
    default:
        r = -1;
    }
    return r;
}

int rect_area(struct Rect *r) {             /* struct pointer param, nested access */
    return (r->max.x - r->min.x) * (r->max.y - r->min.y);
}

int widths(void) {                          /* extra integer type keywords, 32-bit model */
    unsigned int a = 5;
    long b = 10;
    short c = 3;
    signed s = -2;
    const int k = 7;
    unsigned long ul = 100;
    char ch = 'A';
    return a + b + c + s + k + (ul - 100) + (ch - 64);   /* 24 */
}

int labelled(void) {                        /* pass-through label (no goto) */
    int n = 0;
here:
    n = 42;
    return n;
}

/* one small combined pipeline: linked list + switch + ternary + pointers */
int pipeline(void) {
    struct Node pool[3];
    int digits = 0;
    struct Node *it;
    pool[0].value = 1;  pool[0].next = &pool[1];
    pool[1].value = 2;  pool[1].next = &pool[2];
    pool[2].value = -3; pool[2].next = 0;
    it = &pool[0];
    while (it != 0) {
        int code = 0;
        /* 0 = even, 1 = odd, 9 = negative */
        switch (it->value < 0 ? 9 : it->value % 2) {
        case 0:  code = 2; break;
        case 1:  code = 5; break;
        default: code = 9;
        }
        digits = digits * 10 + code;
        it = it->next;
    }
    return digits;                          /* odd, even, negative -> 529 */
}

int main() {
    /* ----- arithmetic, precedence, literals ----- */
    check(1 + 2 * 3 == 7);
    check((1 + 2) * 3 == 9);
    check(-3 + 5 == 2);
    check(-(-5) == 5);
    check(7 / 2 == 3);                      /* int division truncates */
    check(-7 / 2 == -3);                    /* towards zero */
    check(7 % 3 == 1);
    check(-7 % 3 == -1);                    /* sign of the dividend */
    check(10 - 3 - 2 == 5);
    check('A' == 65);
    check('\n' == 10);
    check('0' + 5 == 53);
    check(0xFF == 255);
    check(0xCAFE == 51966);
    check(10U + 20L == 30);                 /* u/l suffixes collapse to int */
    check(1L << 10 == 1024);

    /* assignment as an expression, compound assigns, inc/dec */
    {
        int x = 1;
        int y = (x = 5) + 1;
        check(y == 6);
        x += 4;  check(x == 9);
        x -= 2;  check(x == 7);
        x *= 3;  check(x == 21);
        x /= 2;  check(x == 10);
        x %= 3;  check(x == 1);
        check(x++ == 1);
        check(x == 2);
        check(++x == 3);
        check(x-- == 3);
        check(--x == 1);
    }

    /* ----- bitwise, shifts, their precedence ----- */
    check((5 | 2) == 7);
    check((5 & 3) == 1);
    check((5 ^ 1) == 4);
    check((1 << 4) == 16);
    check((-8 >> 1) == -4);                 /* arithmetic shift */
    check(~0 == -1);
    check(~5 == -6);
    check((1 | 2 & 3) == 3);                /* & binds tighter than | */
    {
        int b = 0xF0; b &= 0x0F; check(b == 0x00);
        b = 1;    b <<= 3;  check(b == 8);
        b = 0xFF; b >>= 4;  check(b == 0x0F);
        b = 5;    b |= 0x10; check(b == 0x15);
        b = 0xAA; b ^= 0xFF; check(b == 0x55);
    }

    /* ----- comparisons, logic, ternary ----- */
    check(3 < 4);
    check((3 < 4) == 1);                    /* comparison yields int 1 */
    check(4 <= 4);
    check(5 > 4 && 4 >= 4 && 1 != 2 && 2 == 2);
    check(!0 == 1);
    check(!7 == 0);
    check((3 > 2 ? 10 : 20) == 10);
    check((0 ? 10 : 20) == 20);
    check((0 ? 1 : 0 ? 2 : 3) == 3);        /* ternary chains */

    /* short circuit evaluation */
    global_counter = 0;
    {
        int r1 = 0 && bump();
        check(global_counter == 0);         /* right side skipped */
        int r2 = 1 || bump();
        check(global_counter == 0);         /* right side skipped */
        int r3 = 1 && bump();
        check(global_counter == 1);         /* right side ran */
        int r4 = 0 || bump();
        check(global_counter == 2);         /* right side ran */
        check(r1 + r2 + r3 + r4 == 3);      /* && and || yield 0/1 */
    }

    /* comma operator */
    {
        int x;
        int i;
        int j;
        int total = 0;
        x = (1, 2, 3);
        check(x == 3);
        for (i = 0, j = 10; i < 3; i++, j--) { total += i + j; }
        check(total == 30);
    }

    /* ----- control flow ----- */
    {
        int g = 0;
        if (11 > 10)     { g = 1; }
        else if (11 > 5) { g = 2; }
        else             { g = 3; }
        check(g == 1);
        if (7 > 10)     { g = 1; }
        else if (7 > 5) { g = 2; }
        else            { g = 3; }
        check(g == 2);
        if (1 > 10)     { g = 1; }
        else if (1 > 5) { g = 2; }
        else            { g = 3; }
        check(g == 3);
    }
    {
        int w = 0;
        while (w > 0) { w--; }              /* runs zero times */
        check(w == 0);
        while (w < 3) { w++; }              /* runs three times */
        check(w == 3);
        int dc = 0;
        do { dc++; } while (0);             /* body runs exactly once */
        check(dc == 1);
    }
    {
        int fsum = 0;
        int fi;
        for (fi = 1; fi <= 3; fi++) { fsum += fi; }
        check(fsum == 6);
        int brk = 0;
        int bi;
        for (bi = 0; bi < 9; bi++) {
            if (bi == 2) { break; }
            brk = brk * 10 + bi + 1;
        }
        check(brk == 12);
        int cont = 0;
        int ci;
        for (ci = 0; ci < 4; ci++) {
            if (ci % 2 == 1) { continue; }
            cont += ci;
        }
        check(cont == 2);
        int nested = 0;
        int oi;
        int ii;
        for (oi = 0; oi < 2; oi++) {
            for (ii = 0; ii < 3; ii++) {
                if (ii == 1) { break; }     /* must not end the outer loop */
                nested++;
            }
        }
        check(nested == 2);
    }

    /* ----- switch: match, stacked labels, fallthrough, default ----- */
    check(classify(0) == 100);
    check(classify(1) == 12);               /* stacked label */
    check(classify(2) == 12);
    check(classify(3) == 43);               /* fell through into case 4 */
    check(classify(4) == 40);
    check(classify(9) == -1);               /* default */

    /* ----- functions, recursion ----- */
    check(add(2, 3) == 5);
    check(early(-4) == -1 && early(4) == 1);
    check(fib(6) == 8);
    check(is_even(4) && is_odd(5));         /* mutual recursion, depth <= 5 */
    noop();
    check(widths() == 24);
    check(labelled() == 42);

    /* ----- arrays ----- */
    {
        int arr[4];
        int i;
        for (i = 0; i < 4; i++) { arr[i] = i * i; }
        check(arr[0] == 0 && arr[3] == 9);
        arr[2] += 10;
        check(arr[2] == 14);
        gtable[1] = 5;
        check(gtable[0] == 0 && gtable[1] == 5);   /* globals zero initialized */

        /* ----- pointers ----- */
        int a = 1, b = 2;
        int *p = &a;
        check(*p == 1);
        *p = 42;
        check(a == 42);
        swap(&a, &b);
        check(a == 2 && b == 42);
        check(nonnull(&a));
        p = &arr[1];
        check(*(p + 1) == 14);              /* pointer arithmetic steps ints */
        check(p[2] == 9);                   /* indexing a pointer */
        *(p + 2) = 7;
        check(arr[3] == 7);
        *p += 1;                            /* compound assign through a pointer */
        check(arr[1] == 2);
    }

    /* ----- structs ----- */
    {
        struct Point pt;
        pt.x = 3;
        pt.y = 4;
        check(pt.x + pt.y == 7);
        pt.x += 10;
        check(pt.x == 13);

        struct Point *pp = &pt;             /* struct pointer with initializer */
        pp->y = 40;
        check(pt.y == 40);
        check(pp->x == 13);
        pp->x++;
        check(pt.x == 14);

        struct Rect rc;                     /* nested struct values */
        rc.min.x = 1; rc.min.y = 2;
        rc.max.x = 4; rc.max.y = 8;
        check(rect_area(&rc) == 18);
        check(rc.max.y - rc.min.y == 6);

        struct Node nodes[3];               /* linked list without malloc */
        nodes[0].value = 5;  nodes[0].next = &nodes[1];
        nodes[1].value = 30; nodes[1].next = &nodes[2];
        nodes[2].value = 7;  nodes[2].next = 0;
        check(nodes[0].next->value == 30);
        check(nodes[0].next->next->value == 7);
        nodes[0].next->value += 1;          /* write along a mixed path */
        check(nodes[1].value == 31);
        chain = &nodes[2];                  /* global struct pointer */
        check(chain->value == 7);
        check(chain->next == 0);

        corners[1].x = 9;                   /* global struct array */
        corners[2].y = corners[1].x + 1;
        check(corners[0].x + corners[1].x + corners[2].y == 19);
    }

    /* ----- enums ----- */
    check(RED == 0 && GREEN == 1 && BLUE == 2);
    check(OK == 0 && WARN == 5 && ERR == 6);
    {
        enum Color fav;
        int hit = 0;
        fav = BLUE;
        switch (fav) {
        case RED:   hit = 1; break;
        case GREEN: hit = 2; break;
        case BLUE:  hit = 3; break;
        default:    hit = 9;
        }
        check(hit == 3);
    }

    /* ----- sizeof of types (compile-time constants) ----- */
    check(sizeof(int) == 4);
    check(sizeof(unsigned long) == 4);
    check(sizeof(int*) == 8);
    check(sizeof(enum Color) == 4);

    /* ----- combined pipeline ----- */
    check(pipeline() == 529);

    /* summary: "features: <checks> checks, <fails> failures" */
    putchar('f'); putchar('e'); putchar('a'); putchar('t'); putchar('u');
    putchar('r'); putchar('e'); putchar('s'); putchar(':'); putchar(' ');
    print_num(nchecks);
    putchar(' '); putchar('c'); putchar('h'); putchar('e'); putchar('c');
    putchar('k'); putchar('s'); putchar(','); putchar(' ');
    print_num(nfail);
    putchar(' '); putchar('f'); putchar('a'); putchar('i'); putchar('l');
    putchar('u'); putchar('r'); putchar('e'); putchar('s'); putchar('\n');
    return nfail;
}
