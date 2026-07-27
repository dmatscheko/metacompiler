/* C subset widening: constructs that used to be accepted but not implemented.
 * Every construct here is genuinely implemented now, in both grammars: the
 * typedef has a real symbol table, floating point has a real numeric tower
 * rather than an integer machine, and goto, union and sizeof-over-an-expression
 * all lower. A flagless run reaches exit 0, and a -warn-unsupported run reports
 * no unsupported construct at all.
 *
 * History: this was a by-design SHOULD-FAIL guard, because a flagless run
 * aborted at the first not-implemented construct (the typedef). The full-syntax
 * work implemented all of them, which removed the guard's premise, so the matrix
 * entry became an ordinary one. Do not re-arm it. **/

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
