/* Two file-scope declarations of one name with different types (C11 6.7p4; cc:
 * "redefinition of 'y' with a different type: 'int' vs 'long'"). */

long y;
int y;

int main(void) {
    return y;
}
