// Exercises the -main flag for the C# interpreter and LLVM-IR compiler: the real entry
// here is the static CheckAll of class Program, not Main. Run as:
//   mec languages/csharp-interpreter.abnf tests/csharp-test-main.cs -q -main CheckAll
// Main() is present but must NOT run (it returns 1); CheckAll() self-checks and returns 0
// on success, so a passing run exits 0 with byte-identical output on all four legs.

class Program {
    static int CheckAll() {
        int fails = 0;
        if (2 + 3 != 5) { fails = fails + 1; }
        int sum = 0;
        for (int i = 1; i <= 4; i = i + 1) { sum = sum + i; }
        if (sum != 10) { fails = fails + 1; }
        return fails;
    }

    static int Main() {
        return 1;
    }
}
