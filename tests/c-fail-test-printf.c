/* Undefined-function fail-test for the native build.
 * The varargs printf family is not part of the modelled C library (neither half
 * implements it), and a call to a function that nothing defines is a link error in
 * C. The -exe build must therefore REFUSE with "function not defined: printf"
 * instead of linking a stub that silently prints nothing. */

int puts(const char *s);
int printf(const char *fmt, ...);

int main(void) {
    printf("P4\n");
    puts("done");
    return 0;
}
