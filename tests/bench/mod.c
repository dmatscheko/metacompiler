/* See mod.kt. The self-contained-IR control: no js_* extern is involved. */
int main(void) {
    long s = 0;
    long i = 0;
    while (i < 40000) {
        s = s + i % 7;
        i = i + 1;
    }
    return s == 0 ? 1 : 0;
}
