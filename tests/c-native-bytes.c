/* The compiled half's storage is cc's layout: member offsets, padding, bit-fields
 * packed into their storage units, and arrays that stride by sizeof - so a byte
 * view of an object and a malloc of n * sizeof agree with cc. The interpreter
 * keeps one cell per scalar (no byte view, no malloc), so this file runs on the
 * compiler only, under llvm.Run and natively. main() returns the number of failed
 * checks; every expected value was printed by cc. **/

void *malloc(unsigned long n);
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

struct Pad { char c; short s; int i; long l; };
struct Bits { unsigned a : 3; unsigned b : 5; int c; };
struct Mixed { char c; int x : 4; char d; int y : 12; short s : 3; };
struct Wide { signed a : 3; unsigned b : 5; _Bool f : 1; _Bool g : 1; long l : 40; long m : 30; };
struct Zero { char a; char b : 4; char d : 6; unsigned : 0; char e; };
union UB { unsigned a : 3; signed b : 5; int i; };
/* A float is four bytes holding its binary32 pattern, a function pointer eight. */
struct FS { char c; float f; short s; float g; };
struct FN { char c; int (*fn)(int); float f; };
int twice(int x) { return 2 * x; }
/* An anonymous struct inside a union: its members at their own offsets, sharing the
   union's bytes with the other members. */
union AS { struct { char a; int b : 5; int c : 3; }; long long q;
           struct { char d[3]; union { short e; char f; }; }; };
struct AN { char k; union { char m; struct { char p; union { int r; struct { short s1, s2; }; }; }; }; char t; };
unsigned fbits(float f) {
    unsigned char *b = (unsigned char *)&f;
    return b[0] | b[1] << 8 | b[2] << 16 | (unsigned)b[3] << 24;
}
float frombits(unsigned u) {
    float f;
    unsigned char *b = (unsigned char *)&f;
    b[0] = u; b[1] = u >> 8; b[2] = u >> 16; b[3] = u >> 24;
    return f;
}
double fvals[] = { -1.5, 3.4028234663852886e38, 1.1754942106924411e-38, 1.401298464324817e-45, 1e39, 0.1 };

int main(void) {
    struct Pad p;
    check((char *)&p.s - (char *)&p == 2 && (char *)&p.i - (char *)&p == 4
          && (char *)&p.l - (char *)&p == 8 && sizeof p == 16);

    /* A malloc'd array of a bit-field struct is exactly n * sizeof bytes. */
    struct Bits *v = malloc(4 * sizeof(struct Bits));
    for (int i = 0; i < 4; i++) { v[i].a = i; v[i].b = 2 * i; v[i].c = 100 + i; }
    int s = 0;
    for (int i = 0; i < 4; i++) s += v[i].a + v[i].b + v[i].c;
    check(sizeof(struct Bits) == 8 && s == 424 && (char *)&v[3] - (char *)&v[0] == 24);

    /* Bit-fields share their unit with their neighbours, byte for byte as cc. */
    struct Mixed m = { 1, -3, 2, 1000, -2 };
    check(sizeof m == 8 && m.c == 1 && m.x == -3 && m.d == 2 && m.y == 1000 && m.s == -2);
    m.x = 9; m.y += 5000; m.s--; m.d++;
    check(m.c == 1 && m.x == -7 && m.d == 3 && m.y == 1904 && m.s == -3);
    unsigned char *pb = (unsigned char *)&m;
    check(pb[0] == 1 && pb[1] == 9 && pb[2] == 3 && pb[3] == 0
          && pb[4] == 112 && pb[5] == 87 && pb[6] == 0 && pb[7] == 0);
    struct Mixed *ma = malloc(3 * sizeof(struct Mixed));
    for (int i = 0; i < 3; i++) { ma[i].c = i; ma[i].x = i - 2; ma[i].d = 10 * i; ma[i].y = -i * 100; ma[i].s = i; }
    check(ma[1].c == 1 && ma[1].x == -1 && ma[1].d == 10 && ma[1].y == -100 && ma[1].s == 1
          && ma[2].x == 0 && ma[2].y == -200 && ma[2].s == 2 && ma[0].x == -2);

    struct Wide w = { 0 };
    w.a = 5; w.b = 40; w.f = 7; w.g = 0; w.l = 1099511627775L; w.m = -5;
    check(sizeof w == 16 && w.a == -3 && w.b == 8 && w.f == 1 && w.g == 0 && w.l == -1 && w.m == -5);
    w.l++; w.b -= 9; w.a = w.a * 3;
    struct Wide w2 = w;
    check(w2.a == -1 && w2.b == 31 && w2.l == 0 && w2.m == -5 && w2.f == 1);

    struct Zero z = { 1, 7, 33, 9 };
    check(sizeof z == 5 && z.b == 7 && z.d == -31 && z.e == 9 && (char *)&z.e - (char *)&z == 4);

    union UB u;
    u.i = 0; u.a = 7;
    int i1 = u.i;
    u.b = -1;
    check(i1 == 7 && u.i == 31 && u.a == 7 && sizeof u == 4);

    float *fa = malloc(6 * sizeof(float));
    for (int i = 0; i < 6; i++) fa[i] = (float)fvals[i];
    check(sizeof(float) == 4 && (char *)&fa[5] - (char *)fa == 20 && fbits(fa[0]) == 0xbfc00000u
          && fbits(fa[1]) == 0x7f7fffffu && fbits(fa[2]) == 0x007fffffu && fbits(fa[3]) == 1u
          && fbits(fa[4]) == 0x7f800000u && fbits(fa[5]) == 0x3dcccccdu && fa[5] * 2 == 0.2f);
    float sub = frombits(0x00012345u), nan = frombits(0x7fc12345u), mz = frombits(0x80000000u);
    check(sub > 0 && sub * 0x1p149 == 0x12345 && fbits(-sub) == 0x80012345u && nan != nan
          && fbits(nan) == 0x7fc12345u && 1 / mz < 0 && fbits(mz) == 0x80000000u);
    struct FS *fs = malloc(3 * sizeof(struct FS));
    for (int i = 0; i < 3; i++) { fs[i].c = i; fs[i].f = i + 0.5f; fs[i].s = 10 * i; fs[i].g = -i * 0.25f; }
    check(sizeof(struct FS) == 16 && (char *)&fs[0].g - (char *)&fs[0] == 12 && fs[2].f == 2.5f
          && fs[2].s == 20 && fs[2].g == -0.5f && fs[1].c == 1 && fbits(fs[1].f) == 0x3fc00000u);
    struct FN *fn = malloc(2 * sizeof(struct FN));
    fn[1].c = 7; fn[1].fn = twice; fn[1].f = 1.25f;
    struct FN fc = fn[1];
    check(sizeof(struct FN) == 24 && (char *)&fn[1].f - (char *)&fn[0] == 40 && fn[1].fn(21) == 42
          && fc.fn(4) == 8 && fc.f == 1.25f && fc.c == 7);

    union AS as;
    as.q = 0; as.a = 9; as.b = -3; as.c = 2;
    int ab = as.d[1];
    as.e = 301;
    check(sizeof as == 8 && ab == 93 && as.a == 9 && as.d[0] == 9 && as.e == 301 && as.f == 45
          && (char *)&as.e - (char *)&as == 4 && as.b == -3 && as.c == 2);
    struct AN an = { 1, { 2 }, 3 };
    an.r = 0x00050004;
    check(sizeof an == 16 && an.s1 == 4 && an.s2 == 5 && an.m == 2 && an.p == 2 && an.t == 3
          && (char *)&an.s2 - (char *)&an == 10 && (char *)&an.t - (char *)&an == 12);
    /* A member's offset from a null pointer, as offsetof is commonly spelled. */
    long o1 = (char *)&((struct AN *)0)->s2 - (char *)0;
    long o2 = (long)&((struct FN *)0)->f;
    check(o1 == 10 && o2 == 16 && (struct FN *)0 == 0);

    putchar('0' + nchecks / 10); putchar('0' + nchecks % 10);
    putchar(' '); putchar('c'); putchar('h'); putchar('e'); putchar('c'); putchar('k');
    putchar('s'); putchar('\n');
    return nfail;
}
