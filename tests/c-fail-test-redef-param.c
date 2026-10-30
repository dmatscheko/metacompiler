/* A function's parameters are declared in the outermost block of its body (C11
 * 6.2.1p4), so a declaration there may not reuse a parameter's name (cc:
 * "redefinition of 'a'"), even in a function nothing calls. A nested block may
 * shadow it, so g passes. */

int g(int a) { { int a = 2; return a; } }

int f(int a) {
    int a = 1;
    return a;
}

int main(void) {
    return g(1);
}
