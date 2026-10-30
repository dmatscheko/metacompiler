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

    putchar('0' + nchecks / 10); putchar('0' + nchecks % 10);
    putchar(' '); putchar('c'); putchar('h'); putchar('e'); putchar('c'); putchar('k');
    putchar('s'); putchar('\n');
    return nfail;
}
