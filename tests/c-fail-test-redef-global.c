/* Two definitions of one file-scope object (C11 6.9p3; cc: "redefinition of 'x'").
 * The declarations before them - an extern one and a tentative one - name the same
 * object and are legal; only the second initializer is an error. */

extern int x;
int x;
int x = 1;
int x = 2;

int main(void) {
    return x;
}
