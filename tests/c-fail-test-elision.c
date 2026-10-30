/* After a nested designator, an undesignated initializer continues INSIDE the
 * subobject the designator walked into (C11 6.7.9p17-20): cc stores 6 in t.b[2].
 * That is brace elision, which neither half implements, so both must refuse it
 * rather than store 6 in the next member, t.c. */

struct T { int a; int b[3]; int c; };

int main(void) {
    struct T t = { .b[1] = 5, 6 };
    return t.b[2] * 10 + t.c;
}
