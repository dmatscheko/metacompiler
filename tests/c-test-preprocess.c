/* Preprocessor self test, run through the two-stage pipeline
 *   c-preprocessor.abnf  prog.c  -pipe  c-to-llvm-ir.abnf   (and c-interpreter.abnf)
 * so the C front end only ever sees macro-expanded source. main() returns the
 * number of failed checks, so the run exits 0 exactly when every macro expanded
 * as C requires (including at-use expansion of nested macros). */

#define SIZE 5
#define BASE 10
#define OFFSET (BASE + BASE)   /* nested macro, expanded at each use */
#define ANSWER 42
#define EMPTY                  /* object-like macro with an empty body */

int nfail = 0;

int check(int got, int want) {
    if (got != want) {
        nfail = nfail + 1;
    }
    return 0;
}

int fill(int *arr, int n) {
    int i;
    for (i = 0; i < n; i = i + 1) {
        arr[i] = i * i;
    }
    return 0;
}

int main() {
    int squares[SIZE];          /* a macro as an array size */
    int i;
    int total;
    int e;

    fill(squares, SIZE);
    total = 0;
    for (i = 0; i < SIZE; i = i + 1) {
        total = total + squares[i];
    }
    check(total, 30);           /* 0+1+4+9+16 */

    check(OFFSET, 20);          /* (BASE + BASE) -> (10 + 10) */
    check(ANSWER, 42);
    check(BASE * SIZE, 50);

    e = EMPTY 8;                /* EMPTY expands to nothing */
    check(e, 8);

#undef BASE
#define BASE 100
    check(BASE, 100);           /* redefinition after #undef */
    check(OFFSET, 200);         /* at-use expansion: OFFSET now (100 + 100) */

    return nfail;
}
