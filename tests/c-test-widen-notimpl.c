/* C subset widening: accepted-but-not-implemented surface.
 * These constructs now PARSE (so a real .c file gets far enough for call graphs),
 * but cannot be lowered to the integer machine, so each is reported as not
 * implemented. A default run ABORTS cleanly at the first such construct (typedef);
 * a -warn-unsupported run warns for every one, places harmless placeholders, and
 * reaches a normal exit 0. Runs identically on the interpreter and the compiler. **/

#ifndef C_TEST_NOTIMPL
#define C_TEST_NOTIMPL
#endif

typedef struct { int x; int y; } Pair;      /* typedef: no symbol table -> not implemented */

union Value { int i; int j; };               /* union: no overlapping layout -> not implemented */

float ratio = 2;                             /* floating-point global -> not implemented */
double scale = 100;                          /* floating-point global -> not implemented */

int main(void) {
    int n = 3;
    float local = 1;                         /* local floating-point decl -> not implemented */
    goto done;                               /* goto: no arbitrary jumps -> not implemented */
done:
    n = sizeof n;                            /* sizeof over an expression -> not implemented */
    return 0;
}
