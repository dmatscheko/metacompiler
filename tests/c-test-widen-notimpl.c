/* C subset widening: typedef, floating point, goto, union and
 * sizeof-over-an-expression, lowered in both grammars. A flagless run exits 0,
 * and a -warn-unsupported run reports no unsupported construct at all. **/

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
