/* See mod.kt. The self-contained-IR control: no js_* extern is involved, so a
   layer-2 or floor change must NOT move this row. `s` is volatile-by-use (it is
   printed via the exit path being 0 regardless) only in the sense that the loop
   must not be folded; keep the accumulator live. */
int main(void) {
    long s = 0;
    long i = 0;
    while (i < 40000) {
        s = s + i % 7;
        i = i + 1;
    }
    return s == 0 ? 1 : 0;
}
