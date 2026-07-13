/* Exercises the -main flag for the C interpreter and LLVM-IR compiler: the real entry
 * here is checkAll(), not main(). Run as:
 *   mec languages/c-interpreter.abnf tests/c-test-main.c -q -main checkAll
 * main() is present but must NOT run (it returns 1); checkAll() self-checks and returns 0
 * on success, so a passing run exits 0 with byte-identical output on all four legs. */

int checkAll() {
    int fails = 0;
    if (2 + 3 != 5) { fails = fails + 1; }
    int sum = 0;
    for (int i = 1; i <= 4; i = i + 1) { sum = sum + i; }
    if (sum != 10) { fails = fails + 1; }
    return fails;
}

int main() {
    return 1;
}
