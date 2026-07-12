// Full-syntax test: C (ISO C11/C17 core language).
//
// This file belongs to the SECOND test group (./test.sh --full): it is NOT part
// of the default matrix. The goal of the metacompiler is to support the full
// languages; this file is the ratchet that measures how far the c grammars
// are. It walks the whole practical C11/C17 syntax, one self-contained
// SECTION per language area. The --full runner runs the file, and whenever a
// grammar aborts it removes the section around the error and retries - so the
// report lists every unsupported section, not just the first.
//
// Conventions (shared by every *-test-full.* file):
//   - prologue (before the first SECTION marker): the check/print helpers only
//     (plus the putchar prototype, exactly like the feature-matrix file)
//   - each section: '// ===== SECTION <nn>: <name> =====', top-level,
//     self-contained, no references to other sections
//   - main() calls each section via a line tagged 'SECTION-CALL <nn>'
//     and prints the summary line 'full: <checks> checks, <failures> failures'
//   - main() returns the failure count (exit 0 == full support, verified)
// The C subset has no strings, so check ids are numeric like in the
// feature-matrix file: id = section * 100 + ordinal, printed as 'FAIL <id>'.
//
// Deliberately out of scope (not core syntax, or unrunnable in this harness):
// the preprocessor (a separate c-preprocessor.abnf grammar exists; this file
// uses zero directives), the standard library (only the putchar prototype, as
// in the feature-matrix file), variadic functions (need the stdarg.h macros),
// VLAs (optional in C11), flexible array members, _Atomic and threads, setjmp,
// wide-character semantics (wide literals appear via sizeof only), K&R-style
// declarations, and anything undefined or implementation-defined - except the
// two pins the feature-matrix file already relies on: the 32-bit-int /
// 64-bit-pointer data model (sizeof checks) and the ASCII execution character
// set ('A' + 1 == 'B'). Binary 0b literals (a C23-ism) are included because
// the reference toolchain accepts them warning-free under -std=c11.
//
// Hand-written for the metacompiler project (Apache-2.0, no copied test-suite
// code), organized after the ISO C11/C17 standard with the ANTLR grammars-v4
// C grammar as a coverage checklist.

int putchar(int c);             /* Prototypes are parsed and ignored. */

int nfail = 0;
int nchecks = 0;

int print_num(int n) {          /* recursive decimal printer (n >= 0) */
    if (n >= 10) { print_num(n / 10); }
    putchar('0' + n % 10);
    return 0;
}

int check(int id, int cond) {   /* a failed check prints FAIL <id> */
    nchecks++;
    if (!cond) {
        nfail++;
        putchar('F'); putchar('A'); putchar('I'); putchar('L'); putchar(' ');
        print_num(id);
        putchar('\n');
    }
    return cond;
}

// ===== SECTION 01: baseline =====
// Condensed re-assertion of the feature-matrix basics this file builds on.
struct S01P { int x; int y; };
int s01_add(int a, int b) { return a + b; }
int s01(void) {
    int n = 0, i;
    for (i = 0; i < 3; i++) { n = n + i; }
    check(101, n == 3);
    struct S01P pt;
    pt.x = 3; pt.y = 4;
    check(102, pt.x + pt.y == 7);
    int arr[3];
    arr[0] = 1; arr[1] = 2; arr[2] = arr[0] + arr[1];
    check(103, arr[2] == 3);
    check(104, s01_add(2, 3) == 5);
    check(105, 7 / 2 == 3 && 7 % 2 == 1 && (3 > 2 ? 10 : 20) == 10);
    return 0;
}

// ===== SECTION 02: integer and character constants =====
int s02(void) {
    check(201, 0xFF == 255 && 0Xff == 255 && 0xCAFE == 51966);
    check(202, 017 == 15 && 0101 == 65 && 0 == 0x0);
    check(203, 0b1010 == 10);
    check(204, 10U == 10 && 20L == 20 && 30UL == 30 && 40LL == 40 && 50ULL == 50 && 60lu == 60);
    check(205, 'A' == 65 && '\n' == 10 && '\t' == 9 && '\0' == 0);
    check(206, '\x41' == 65 && '\101' == 65);
    check(207, '\'' == 39 && '\"' == 34 && '\\' == 92);
    check(208, 'A' + 1 == 'B' && '0' + 5 == '5' && '9' - '0' == 9);
    return 0;
}

// ===== SECTION 03: floating constants =====
// Every comparison uses exactly representable values only.
int s03(void) {
    double a = 1.5;
    float b = 0.25f;
    check(301, a + a == 3.0 && b * 4.0f == 1.0f);
    check(302, .5 == 0.5 && 5. == 5.0);
    check(303, 1e3 == 1000.0 && 2.5e-2 == 0.025 && 1E2 == 100.0);
    check(304, 0x1p3 == 8.0 && 0x1.8p1 == 3.0 && 0xAp-1 == 5.0);
    long double ld = 1.0L;
    check(305, ld / 4.0L == 0.25L);
    double c = 3.0;
    check(306, c / 2.0 == 1.5 && (c - 1.0) * 0.5 == 1.0);
    return 0;
}

// ===== SECTION 04: string literals =====
int s04(void) {
    char a[] = "ab";
    check(401, a[0] == 'a' && a[1] == 'b' && a[2] == '\0' && sizeof a == 3);
    char b[] = "ab" "cd";       /* adjacent literals concatenate */
    check(402, sizeof b == 5 && b[2] == 'c');
    check(403, "a\tb"[1] == '\t' && "x"[0] == 'x');
    check(404, sizeof "abc" == 4 && sizeof "a\0b" == 4);
    check(405, sizeof(L"ab") == 3 * sizeof(L'a'));   /* wide: syntax and sizes only */
    char c8[4] = "hi";          /* array longer than the literal: zero padded */
    check(406, c8[2] == 0 && c8[3] == 0);
    return 0;
}

// ===== SECTION 05: operators =====
int s05_hits = 0;
int s05_bump(void) { s05_hits += 1; return s05_hits; }
int s05(void) {
    check(501, 1 + 2 * 3 == 7 && (1 + 2) * 3 == 9 && 10 - 3 - 2 == 5);
    check(502, -7 / 2 == -3 && -7 % 2 == -1 && 7 / -2 == -3 && 7 % -2 == 1);
    check(503, (3 < 4) == 1 && (4 <= 4) == 1 && (5 > 6) == 0 && (5 >= 6) == 0 && (1 != 2) == 1);
    int zero = 0, seven = 7, three = 3, five = 5;
    check(504, (!zero) == 1 && (!seven) == 0 && (seven && three) == 1 && (zero || five) == 1);
    int r1 = 0 && s05_bump();
    int r2 = 1 || s05_bump();
    check(505, s05_hits == 0);  /* both right sides skipped */
    int r3 = 1 && s05_bump();
    int r4 = 0 || s05_bump();
    check(506, s05_hits == 2 && r1 + r2 + r3 + r4 == 3);
    check(507, (5 | 2) == 7 && (5 & 3) == 1 && (5 ^ 1) == 4 && (2 & 3 + 1) == 0);
    check(508, (~5u & 15u) == 10u && (~0u & 1u) == 1u);
    check(509, (1 << 4) == 16 && (32 >> 2) == 8 && (1 << 2 << 1) == 8);
    int x = 1;
    int y = (x = 5) + 1;        /* assignment is an expression */
    check(510, y == 6 && x == 5);
    x += 4; x -= 2; x *= 3; x /= 7; x %= 2;
    check(511, x == 1);
    x <<= 3; x >>= 1; x |= 16; x ^= 5; x &= 29;
    check(512, x == 17);
    check(513, x++ == 17 && x == 18 && ++x == 19 && x-- == 19 && --x == 17);
    int t;
    int cm = (t = 1, t + 2);    /* comma evaluates left to right */
    check(514, cm == 3 && t == 1);
    check(515, (0 ? 1 : 0 ? 2 : 3) == 3 && (87 >= 90 ? 1 : 87 >= 80 ? 2 : 3) == 2);
    check(516, sizeof(int) == 4 && sizeof(int *) == 8);   /* the pinned data model */
    return 0;
}

// ===== SECTION 06: integer conversions =====
int s06(void) {
    unsigned int m = (unsigned)-1;   /* conversion to unsigned is modulo 2^N */
    check(601, m == 0u - 1u);
    check(602, 0u - 1u > 0u);        /* unsigned wraparound is well defined */
    check(603, (0u - 1u) % 2u == 1u);
    check(604, m > 100u && m / 2u < m);
    check(605, 6u / 4u == 1u && 6u % 4u == 2u);
    check(606, (int)7u == 7 && (unsigned)3 + 4u == 7u);
    return 0;
}

// ===== SECTION 07: declarations and declarators =====
extern int s07_seven(void);          /* extern forward declaration, defined below */
int s07(void) {
    int a, *p, **pp, arr[3];         /* one line, mixed declarators */
    a = 4; p = &a; pp = &p; arr[0] = **pp;
    check(701, arr[0] == 4 && *p == 4);
    int x = 1, y = 2, z;
    z = x + y;
    check(702, z == 3);
    const int k = 7;
    volatile int vol = 2;
    register int reg = 3;
    check(703, k + vol + reg == 12);
    check(704, s07_seven() == 7);
    arr[1] = 5; arr[2] = arr[1] - 5;
    check(705, arr[1] == 5 && arr[2] == 0);
    return 0;
}
int s07_seven(void) { return 7; }

// ===== SECTION 08: typedef =====
typedef int S08Int;
typedef S08Int S08Num;               /* typedef chain */
typedef int S08Tri[3];               /* array typedef */
typedef struct { int v; } S08Box;    /* struct typedef */
typedef int (*S08Fn)(int);           /* function-pointer typedef */
int s08_neg(int x) { return -x; }
S08Fn s08_pick(void) { return s08_neg; }
int s08(void) {
    S08Num n = 41;
    check(801, n + 1 == 42);
    S08Tri t = { 1, 2, 3 };
    check(802, t[0] + t[2] == 4);
    S08Box b;
    b.v = 9;
    check(803, b.v == 9);
    S08Fn f = s08_neg;
    check(804, f(5) == -5);
    check(805, s08_pick()(6) == -6);
    return 0;
}

// ===== SECTION 09: pointers =====
int s09(void) {
    int x = 1;
    int *p = &x, **pp = &p, ***ppp = &pp;   /* pointer levels */
    check(901, **pp == 1 && ***ppp == 1);
    check(902, *&x == 1);
    int a[5];
    int i;
    for (i = 0; i < 5; i++) { a[i] = i * 10; }
    int *b = &a[1], *e = &a[4];
    check(903, e - b == 3 && b < e && e >= b + 3);
    check(904, *(b + 2) == 30 && b[2] == 30 && *(e - 1) == 30);
    int *z = 0;                              /* null as the literal 0 */
    check(905, z == 0 && !z);
    int v = 7;
    void *vp = &v;
    int *ip = vp;                            /* void* converts back without a cast */
    check(906, *ip == 7 && vp != 0);
    return 0;
}

// ===== SECTION 10: arrays =====
int s10(void) {
    int m[2][3] = { { 1, 2, 3 }, { 4, 5, 6 } };
    check(1001, m[0][0] == 1 && m[1][2] == 6);
    int z[4] = { 7, 8 };                     /* partial init zeroes the rest */
    check(1002, z[0] == 7 && z[1] == 8 && z[2] == 0 && z[3] == 0);
    int d[5] = { [2] = 5, [4] = 9 };         /* designated initializers */
    check(1003, d[0] == 0 && d[2] == 5 && d[3] == 0 && d[4] == 9);
    int x = 1, y = 2;
    int *ap[2] = { &x, &y };                 /* array of pointers */
    check(1004, *ap[0] + *ap[1] == 3);
    int (*pa)[3] = m;                        /* pointer to array */
    check(1005, pa[1][0] == 4 && (*pa)[1] == 2);
    char cs[] = { 'h', 'i', 0 };
    check(1006, cs[1] == 'i' && cs[2] == 0);
    return 0;
}

// ===== SECTION 11: structs as values =====
struct S11P { int x; int y; };
struct S11B { int d[2]; int n; };            /* struct containing an array */
int s11_sum(struct S11P p) { return p.x + p.y; }
struct S11P s11_mk(int x, int y) { struct S11P r; r.x = x; r.y = y; return r; }
int s11(void) {
    struct S11P a = { .y = 2, .x = 1 };      /* designated initializer */
    check(1101, a.x == 1 && a.y == 2);
    struct S11P b = a;                       /* init copies the value */
    a.x = 99;
    check(1102, b.x == 1 && a.x == 99);
    check(1103, s11_sum(b) == 3);            /* passed by value */
    struct S11P c = s11_mk(4, 6);            /* returned by value */
    check(1104, c.x + c.y == 10);
    b = c;                                   /* assignment copies too */
    check(1105, b.x == 4 && c.y == 6);
    struct S11B box = { { 8, 9 }, 2 };
    check(1106, box.d[1] == 9 && box.n == 2);
    return 0;
}

// ===== SECTION 12: bitfields and anonymous members =====
struct S12F { unsigned a : 3; unsigned b : 5; int c : 4; };
struct S12A { int tag; union { int i; unsigned u; }; };   /* C11 anonymous union */
int s12(void) {
    struct S12F f;
    int nine = 9;
    f.a = nine;                  /* 9 masked to 3 bits -> 1 (unsigned fields wrap) */
    f.b = 17; f.c = 5;
    check(1201, f.a == 1);
    check(1202, f.b == 17 && f.c == 5);
    f.a = f.a + 7;               /* 1 + 7 = 8 -> masked to 0 */
    check(1203, f.a == 0);
    struct S12A x;
    x.tag = 3; x.i = 42;
    check(1204, x.tag == 3 && x.i == 42);
    x.u = 5u;
    check(1205, x.u == 5u);
    return 0;
}

// ===== SECTION 13: unions =====
union S13U { int i; unsigned u; };
struct S13W { int kind; union S13U v; };
int s13(void) {
    union S13U a;
    a.i = 5;                     /* write then read the SAME member only */
    check(1301, a.i == 5);
    a.u = 9u;
    check(1302, a.u == 9u);
    check(1303, sizeof(union S13U) >= sizeof(int) && sizeof(union S13U) >= sizeof(unsigned));
    union S13U b = { .u = 3u };
    check(1304, b.u == 3u);
    struct S13W w;
    w.kind = 1; w.v.i = 6;
    check(1305, w.kind == 1 && w.v.i == 6);
    return 0;
}

// ===== SECTION 14: enums =====
enum S14Color { S14_RED, S14_GREEN, S14_BLUE };
enum S14Status { S14_OK = 0, S14_WARN = 5, S14_ERR };     /* ERR continues at 6 */
enum S14Level { S14_LOW = -1, S14_MID, S14_TOP = 7 };
int s14(void) {
    check(1401, S14_RED == 0 && S14_GREEN == 1 && S14_BLUE == 2);
    check(1402, S14_OK == 0 && S14_WARN == 5 && S14_ERR == 6);
    check(1403, S14_LOW == -1 && S14_MID == 0 && S14_TOP == 7);
    enum S14Color fav = S14_BLUE;
    int hit = 0;
    switch (fav) {
    case S14_RED:   hit = 1; break;
    case S14_GREEN: hit = 2; break;
    case S14_BLUE:  hit = 3; break;
    default:        hit = 9;
    }
    check(1404, hit == 3);
    check(1405, S14_RED + 2 == S14_BLUE && S14_WARN * 2 == 10);
    return 0;
}

// ===== SECTION 15: scope and storage duration =====
static int s15_calls = 0;        /* static globals */
static int s15_gval = 4;
int s15_next(void) {
    static int hits = 0;         /* a static local persists across calls */
    hits = hits + 1;
    return hits;
}
int s15(void) {
    int x = 1;
    {
        int x = 2;               /* block scope shadows the outer x */
        check(1501, x == 2);
    }
    check(1502, x == 1);
    check(1503, s15_next() == 1 && s15_next() == 2);
    struct S15L { int v; };      /* block-scope type definition */
    struct S15L loc;
    loc.v = 8;
    check(1504, loc.v == 8);
    s15_calls = s15_calls + 1;
    check(1505, s15_gval + s15_calls == 5);
    return 0;
}

// ===== SECTION 16: iteration statements =====
int s16(void) {
    int w = 5;
    while (w > 5) { w = w - 1; } /* runs zero times */
    check(1601, w == 5);
    while (w < 8) { w++; }
    check(1602, w == 8);
    int dc = 0;
    do { dc++; } while (dc < 3);
    check(1603, dc == 3);
    do { dc++; } while (0);      /* body runs exactly once */
    check(1604, dc == 4);
    int sum = 0, i, j;
    for (i = 1; i <= 3; i++) { sum += i; }
    check(1605, sum == 6);
    int n = 0;
    for (;;) { n++; break; }     /* all three clauses empty */
    check(1606, n == 1);
    for (i = 0, j = 10; i < j; i++, j -= 2) { n += 1; }   /* comma expressions */
    check(1607, n == 5);
    for (i = 9; ; i--) { if (i < 8) { break; } }          /* empty condition */
    check(1608, i == 7);
    int hits = 0, oi, ii;
    for (oi = 0; oi < 2; oi++) {
        for (ii = 0; ii < 3; ii++) {
            if (ii == 1) { break; }          /* ends the inner loop only */
            hits++;
        }
    }
    check(1609, hits == 2);
    int cont = 0;
    for (i = 0; i < 5; i++) { if (i % 2 == 1) { continue; } cont += i; }
    check(1610, cont == 6);
    return 0;
}

// ===== SECTION 17: jump statements and labels =====
int s17_grid(void) {             /* goto out of nested loops */
    int i, j, n = 0;
    for (i = 0; i < 3; i++) {
        for (j = 0; j < 3; j++) {
            n++;
            if (i == 1 && j == 1) { goto s17_out; }
        }
    }
s17_out:
    return n;                    /* 3 + 2 = 5 */
}
int s17_skip(void) {
    int r = 1;
    goto s17_end;
    r = 2;                       /* jumped over */
s17_end:
    return r;
}
int s17_back(void) {             /* backward goto */
    int n = 0;
s17_again:
    n = n + 40;
    if (n < 41) { goto s17_again; }
    return n;                    /* 80 */
}
int s17_def(int x) {             /* default listed first, cases still win */
    int r;
    switch (x) {
    default: r = 9; break;
    case 1:  r = 10; break;
    case 2:  r = 20; break;
    }
    return r;
}
int s17(void) {
    check(1701, s17_grid() == 5);
    check(1702, s17_skip() == 1);
    check(1703, s17_back() == 80);
    check(1704, s17_def(2) == 20 && s17_def(7) == 9);
    return 0;
}

// ===== SECTION 18: functions =====
int s18_odd(int n);              /* prototype for mutual recursion */
int s18_even(int n) { return n == 0 ? 1 : s18_odd(n - 1); }
int s18_odd(int n)  { return n == 0 ? 0 : s18_even(n - 1); }
int s18_fib(int n) { return n < 2 ? n : s18_fib(n - 1) + s18_fib(n - 2); }
int s18_clamp(int n) {
    if (n < 0) { return 0; }     /* early return */
    return n;
}
void s18_touch(int *p) { *p = *p + 1; }      /* void return type */
int s18(void) {
    check(1801, s18_even(4) == 1 && s18_odd(5) == 1);
    check(1802, s18_fib(5) == 5);
    check(1803, s18_clamp(-4) == 0 && s18_clamp(4) == 4);
    int t = 6;
    s18_touch(&t);
    check(1804, t == 7);
    check(1805, s18_fib(s18_clamp(3)) == 2);
    return 0;
}

// ===== SECTION 19: function pointers =====
int s19_inc(int x) { return x + 1; }
int s19_dbl(int x) { return x + x; }
int s19_apply(int (*f)(int), int v) { return f(v); }
int (*s19_pick(int which))(int) { return which == 0 ? s19_inc : s19_dbl; }
int s19(void) {
    int (*fp)(int) = s19_inc;
    check(1901, fp(4) == 5 && (*fp)(4) == 5);
    fp = s19_dbl;
    check(1902, fp(4) == 8);
    int (*tab[2])(int) = { s19_inc, s19_dbl };   /* array of function pointers */
    check(1903, tab[0](3) + tab[1](3) == 10);
    check(1904, s19_apply(s19_dbl, 6) == 12);
    check(1905, s19_pick(1)(5) == 10);
    return 0;
}

// ===== SECTION 20: _Bool, _Static_assert, sizeof, alignment =====
int s20(void) {
    _Static_assert(1 + 1 == 2, "constant arithmetic");
    _Static_assert(sizeof(char) == 1, "char is exactly one byte");
    _Bool t = 1, f = 0;
    check(2001, t == 1 && f == 0);
    _Bool two = 2;               /* any nonzero value converts to 1 */
    check(2002, two == 1);
    check(2003, sizeof(char) == 1 && sizeof 'a' == sizeof(int));
    check(2004, sizeof(short) >= 2 && sizeof(long) >= 4 && sizeof(long long) >= 8);
    int sv = 3;
    check(2005, sizeof sv == sizeof(int) && sizeof(sv + 1) == sizeof(int));
    check(2006, _Alignof(char) == 1 && sizeof(int) % _Alignof(int) == 0);
    _Alignas(8) int al = 6;
    check(2007, al == 6);
    return 0;
}

// ===== SECTION 21: _Generic =====
int s21(void) {
    check(2101, _Generic(1, int: 10, double: 20, default: 30) == 10);
    check(2102, _Generic(1.5, int: 10, double: 20, default: 30) == 20);
    check(2103, _Generic('a', char: 1, int: 2, default: 3) == 2);   /* 'a' is an int */
    check(2104, _Generic((char)0, char: 1, int: 2, default: 3) == 1);
    check(2105, _Generic(1ul, int: 4, default: 42) == 42);
    return 0;
}

// ===== SECTION 22: compound literals =====
struct S22P { int x; int y; };
int s22(void) {
    check(2201, (int[]){ 1, 2, 3 }[1] == 2);
    int *p = (int[]){ 5, 6 };
    check(2202, p[0] == 5 && p[1] == 6);
    check(2203, (struct S22P){ .x = 3, .y = 4 }.y == 4);
    struct S22P *sp = &(struct S22P){ .x = 8, .y = 9 };
    check(2204, sp->x == 8 && sp->y == 9);
    return 0;
}

// ===== SECTION 23: casts =====
int s23(void) {
    check(2301, (int)3.9 == 3 && (int)-3.9 == -3);   /* truncation toward zero */
    check(2302, (double)1 / 2 == 0.5 && 1 / 2 == 0);
    check(2303, (int)'A' == 65 && (char)66 == 'B');
    int v = 7;
    void *vp = (void *)&v;
    int *ip = (int *)vp;
    check(2304, *ip == 7);
    check(2305, (unsigned)7 == 7u && (int)7u == 7);
    check(2306, (1 ? 2 : 2.5) == 2.0);               /* ternary balances to double */
    return 0;
}

// ===== SECTION 24: const, restrict, inline =====
static inline int s24_twice(int x) { return x + x; }
int s24_dot(const int *restrict a, const int *restrict b) { return a[0] * b[0] + a[1] * b[1]; }
int s24(void) {
    int v = 5;
    const int *pc = &v;          /* pointer to const */
    check(2401, *pc == 5);
    v = 6;
    check(2402, *pc == 6);       /* the pointee may still change */
    int w = 1;
    int *const cp = &w;          /* const pointer */
    *cp = 9;
    check(2403, w == 9);
    check(2404, s24_twice(21) == 42);
    int xs[2] = { 1, 2 };
    int ys[2] = { 3, 4 };
    check(2405, s24_dot(xs, ys) == 11);
    const int k = 7;
    check(2406, k == 7);
    return 0;
}

// ===== END SECTIONS =====

int main() {
    s01(); // SECTION-CALL 01
    s02(); // SECTION-CALL 02
    s03(); // SECTION-CALL 03
    s04(); // SECTION-CALL 04
    s05(); // SECTION-CALL 05
    s06(); // SECTION-CALL 06
    s07(); // SECTION-CALL 07
    s08(); // SECTION-CALL 08
    s09(); // SECTION-CALL 09
    s10(); // SECTION-CALL 10
    s11(); // SECTION-CALL 11
    s12(); // SECTION-CALL 12
    s13(); // SECTION-CALL 13
    s14(); // SECTION-CALL 14
    s15(); // SECTION-CALL 15
    s16(); // SECTION-CALL 16
    s17(); // SECTION-CALL 17
    s18(); // SECTION-CALL 18
    s19(); // SECTION-CALL 19
    s20(); // SECTION-CALL 20
    s21(); // SECTION-CALL 21
    s22(); // SECTION-CALL 22
    s23(); // SECTION-CALL 23
    s24(); // SECTION-CALL 24
    /* summary: "full: <checks> checks, <failures> failures" */
    putchar('f'); putchar('u'); putchar('l'); putchar('l'); putchar(':'); putchar(' ');
    print_num(nchecks);
    putchar(' '); putchar('c'); putchar('h'); putchar('e'); putchar('c');
    putchar('k'); putchar('s'); putchar(','); putchar(' ');
    print_num(nfail);
    putchar(' '); putchar('f'); putchar('a'); putchar('i'); putchar('l');
    putchar('u'); putchar('r'); putchar('e'); putchar('s'); putchar('\n');
    return nfail;
}
