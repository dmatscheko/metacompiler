/* C subset widening, second file: typedef, floating point, goto, union and
 * sizeof over an expression, lowered in both grammars. main() returns the
 * number of failed checks, so the run exits 0 exactly when every value is the
 * one a C compiler gives; a -warn-unsupported run reports nothing. **/

#ifndef C_TEST_WIDEN2
#define C_TEST_WIDEN2
#endif

int putchar(int c);

int nfail = 0;
int nchecks = 0;

int check(int cond) {           /* auto-numbered: prints FAIL <ordinal> */
    nchecks++;
    if (!cond) {
        nfail++;
        putchar('F'); putchar('A'); putchar('I'); putchar('L'); putchar(' ');
        putchar('0' + nchecks / 10); putchar('0' + nchecks % 10);
        putchar('\n');
    }
    return cond;
}

typedef struct { int x; int y; } Pair;
typedef int Count;

union Value { int i; int j; };

float ratio = 2;
double scale = 100;

int main(void) {
    Pair p;
    p.x = 3; p.y = 4;
    check(p.x * p.y == 12);
    Count c = 5;
    check(c + 1 == 6);

    union Value v;
    v.i = 41;
    check(v.j == 41);                       /* the members share storage */
    v.j = 7;
    check(v.i == 7);

    check(ratio * 3 == 6);
    check(scale / 8 == 12.5);
    float local = 1;
    local = local / 4;
    check(local == 0.25);
    check((int)(scale * ratio) == 200);

    int n = 3;
    goto done;
    n = 99;                                 /* jumped over */
done:
    check(n == 3);

    check(sizeof n == 4);
    check(sizeof p == 8);
    check(sizeof(n + 1) == 4);

    putchar('o'); putchar('k'); putchar(' ');
    putchar('0' + nchecks / 10); putchar('0' + nchecks % 10);
    putchar('\n');
    return nfail;
}
