/* C recognition self test: a realistic file mixing constructs the widened C grammar
 * now ACCEPTS. The genuinely lowered ones compute real values:
 *   - storage-class / function specifiers (static, extern, inline),
 *   - hexadecimal and u/U/l/L-suffixed integer literals,
 *   - compound bitwise and shift assignment (&= |= ^= <<= >>=),
 *   - variadic prototypes ( ..., int printf(const char *, ...) ),
 *   - __attribute__((...)) specifiers.
 * The ones the flat integer machine cannot model are RECOGNISED and routed to
 * notImplemented:
 *   - string literals ("..."),
 *   - cast expressions ((int)x, (void)x),
 *   - aggregate / brace / multi-dimensional array initializers ( = { ... } ),
 *   - multi-level pointers (char **argv).
 * A normal run therefore ABORTS cleanly at the first not-implemented construct; a
 * -warn-unsupported run warns for each, places harmless placeholders, and still
 * reaches a clean exit 0. Runs identically on goja and -frozen, and on the interpreter
 * and the compiler. main() returns the number of failed genuine checks (0 = success). */

#include <stdio.h>
#include <stdlib.h>

/* variadic prototype and an __attribute__ specifier: both accepted, both ignored */
int printf(const char *fmt, ...);
static void die(const char *msg) __attribute__((noreturn));

/* a global lookup table with an aggregate initializer: recognised, not implemented */
static const int masks[] = {0x1, 0x2, 0x4, 0x8, 0x10};

static int nfail = 0;
static void ck(int got, int want) { if (got != want) nfail++; }

/* genuinely lowered: hex / suffixed literals and compound bitwise+shift assignment */
static inline int scramble(unsigned int seed) {
    int acc = seed;
    acc &= 0xFFFF;          /* keep low 16 bits */
    acc |= 0x1000;          /* set a bit       */
    acc ^= 0x00FF;          /* flip low byte   */
    acc <<= 1;              /* double          */
    acc >>= 2;              /* quarter of that */
    return acc;
}

int main(int argc, char **argv) {                /* char **argv: multi-level pointer */
    /* --- genuine checks: these must all pass so main() returns 0 under -warn --- */
    ck(0xFF, 255);
    ck(0xCAFE, 51966);
    ck(0Xa + 0xA, 20);
    ck(10U + 20L, 30);
    ck(1L << 10, 1024);

    int b = 0xF0; b &= 0x0F; ck(b, 0x00);
    int c = 1;    c <<= 8;   ck(c, 0x100);
    int d = 0xFF; d >>= 4;   ck(d, 0x0F);
    int e = 5;    e |= 0x10; ck(e, 0x15);
    int f = 0xAA; f ^= 0xFF; ck(f, 0x55);

    ck(scramble(0xABCD), 0x5D99);

    /* --- recognised-but-not-implemented constructs (harmless under -warn) --- */
    const char *greeting = "hello, recognizer";   /* string literal        */
    int table[4] = {1, 2, 3, 4};                   /* aggregate initializer */
    long big = (long)0x7FFFFFFFL;                  /* cast expression       */
    (void)argc;                                    /* cast to void          */

    return nfail;
}
