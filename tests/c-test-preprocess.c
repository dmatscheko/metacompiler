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

/* Function-like macros: the '(' must follow the name with no space between. */
#define ADD(a, b) ((a) + (b))
#define SUM3(a, b, c) ADD(ADD(a, b), c)   /* a macro call inside a macro body */
#define NOARG() 7                         /* empty parameter list */
#define FIRST(a, b) a                     /* an empty argument expands to nothing */
#define NEG(x) -x                         /* the seam with the source must not glue */
#define FN(x) ((x) * 2)                   /* also used WITHOUT '(' - then left alone */

/* # stringification and ## token pasting. */
#define STR(x) #x
#define XSTR(x) STR(x)                    /* argument macro-expanded first */
#define CAT(x, y) x ## y                  /* '##' operands are NOT pre-expanded */
#define XCAT(x, y) CAT(x, y)              /* ... but these are */
#define PFX cat
#define SFX var

/* Variadic macros. */
#define PICK1(a, ...) a
#define REST(a, ...) __VA_ARGS__

/* A macro is not re-expanded inside its own expansion, so this resolves to the
 * real function dbl() rather than looping. */
#define dbl(x) dbl(x)

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

int dbl(int v) { return v + v; }

int PFXSFX = 9;                 /* the name CAT(PFX, SFX) pastes to, unexpanded */
int catvar = 3;                 /* the name XCAT(PFX, SFX) pastes to, expanded */
int FN = 6;                     /* FN without '(' is not a macro call */

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

    /* ---- function-like macros ---- */
    check(ADD(2, 3), 5);
    check(ADD(ANSWER, SIZE), 47);       /* arguments are macro-expanded */
    check(SUM3(1, 2, 3), 6);
    check(NOARG(), 7);
    check(FIRST(, 1) 4, 4);             /* F(, 1) -> nothing, then the source's 4 */
    check(3 -NEG(2), 5);                /* '3 -' then '-2' must not become '3 --2' */
    check(FN, 6);                       /* no '(' follows: not a call, so not expanded */
    check(FN(4), 8);
    check(ADD(1,
             2), 3);                    /* a call may span lines */

    /* ---- # and ## ---- */
    check(STR(hello)[0], 'h');
    check(STR(hello)[4], 'o');
    check(STR(hello)[5], 0);
    check(XSTR(SIZE)[0], '5');          /* the argument is expanded before # sees it */
    check(STR(SIZE)[0], 'S');           /* ... but not under a plain # */
    check(CAT(cat, var), 3);
    check(CAT(PFX, SFX), 9);            /* '##' operands stay unexpanded: PFXSFX */
    check(XCAT(PFX, SFX), 3);           /* one level of indirection expands them: catvar */
    check(CAT(1, 0), 10);               /* pasting digits makes one number, not two */
    check(CAT(SIZE, ), 5);              /* an empty operand leaves a placemarker */

    /* ---- variadic ---- */
    check(PICK1(4, 5, 6), 4);
    check(REST(1, 2), 2);

    /* ---- the recursion rule ---- */
    check(dbl(21), 42);

#undef ADD
    e = 1;
    check(e, 1);                        /* #undef of a function-like macro */

    return nfail;
}
