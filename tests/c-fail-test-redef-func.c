/* Two definitions of one function (C11 6.9p3; cc: "redefinition of 'f'"). The
 * prototypes around them may repeat; only the second body is an error. */

typedef int I;

int f(int x);
int f(int x) { return x + 1; }
int f(int x);
I f(int x) { return x + 2; }

int main(void) {
    return f(1);
}
