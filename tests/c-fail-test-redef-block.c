/* A second declaration of one name in ONE block is a constraint violation (C11
 * 6.7p3; cc: "redefinition of 'zero'"), and both halves must reject it while
 * compiling - here in a branch that never runs. The same name in a NESTED block
 * is legal shadowing, so the first two blocks pass. */

int putchar(int c);

int f(void) {
    int zero = 1;
    { int zero = 2; { int zero = 3; } }
    if (0) { int zero = 0; int zero = 0; }
    return zero;
}

int main(void) {
    putchar('A');
    return f();
}
