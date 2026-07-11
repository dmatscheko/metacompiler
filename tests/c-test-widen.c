/* C subset widening self test (genuinely lowered features only).
 * Exercises the newly supported surface that runs on the integer machine:
 * extra integer type keywords (unsigned/long/short/char/signed/const), void
 * functions and void* parameters, enum constants and enum-typed variables, the
 * comma operator, sizeof of scalar and pointer types, preprocessor directives
 * (#ifndef/#define/#endif/#pragma), and a pass-through label. main() returns the
 * number of failed checks, so the run exits 0 exactly when everything works.
 * The same program runs identically on c-interpreter.abnf and c-to-llvm-ir.abnf. **/

#ifndef C_TEST_WIDEN
#define C_TEST_WIDEN
#pragma once
#define UNUSED_MACRO 12345      /* object-like macro, accepted and ignored (not expanded) */
#endif

#include <stdio.h>

int nfail = 0;

int check(int got, int want) {
    if (got != want) {
        nfail++;
        putchar('F');
        putchar('0' + nfail);
        putchar('\n');
    }
    return got == want;
}

/* enum constants become plain int constants; the tag is ignored. */
enum Color { RED, GREEN, BLUE };
enum Status { OK = 0, WARN = 5, ERR };      /* ERR continues at 6 */

void noop(void) { }                          /* void return type */

int nonnull(void *p) { return p != 0; }      /* void* parameter */

/* extra integer type keywords, all collapsing to the 32-bit int model */
int widths(void) {
    unsigned int a = 5;
    long b = 10;
    short c = 3;
    signed s = -2;
    const int k = 7;
    unsigned long ul = 100;
    char ch = 'A';
    return a + b + c + s + k + (ul - 100) + (ch - 64);   /* 5+10+3-2+7+0+1 = 24 */
}

/* a plain label passes its statement straight through (no goto jumps to it) */
int labelled(void) {
    int n = 0;
here:
    n = 42;
    return n;
}

/* the comma operator in a for-header and as a parenthesized expression */
int comma_sum(void) {
    int i;
    int j;
    int total = 0;
    for (i = 0, j = 10; i < 3; i++, j--) {
        total += i + j;                       /* (0+10)+(1+9)+(2+8) = 30 */
    }
    return total;
}

int main(void) {
    noop();

    /* enum constant values */
    check(RED, 0);
    check(GREEN, 1);
    check(BLUE, 2);
    check(OK, 0);
    check(WARN, 5);
    check(ERR, 6);

    /* enum-typed variable is a plain int */
    enum Color fav;
    fav = GREEN;
    check(fav, 1);
    fav = BLUE;
    switch (fav) {
    case RED:   check(0, 1); break;
    case GREEN: check(0, 1); break;
    case BLUE:  check(1, 1); break;
    default:    check(0, 1);
    }

    /* integer type keywords */
    check(widths(), 24);

    /* void* parameter */
    int v = 3;
    check(nonnull(&v), 1);

    /* comma operator */
    int x;
    x = (1, 2, 3);
    check(x, 3);
    check(comma_sum(), 30);

    /* sizeof of scalar and pointer types (compile-time constants) */
    check(sizeof(int), 4);
    check(sizeof(unsigned long), 4);
    check(sizeof(int*), 8);
    check(sizeof(enum Color), 4);

    /* pass-through label */
    check(labelled(), 42);

    if (nfail == 0) {
        putchar('O'); putchar('K'); putchar('\n');
    }
    return nfail;
}
