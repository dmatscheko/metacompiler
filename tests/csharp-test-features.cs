// Fast feature-matrix test for the C# interpreter (csharp-interpreter.abnf) and the
// LLVM-IR compiler (csharp-to-llvm-ir.abnf). It replaces the four algorithm-themed
// csharp-test-big-* stress tests: instead of large loops (sorts, Ackermann, ciphers)
// every implemented construct is exercised with the SMALLEST program that can prove
// it works - loops run 0, 1, 3 or 4 times, recursion stays below depth 6. A failed
// check prints its id (so a diff pinpoints it) and Program.Main returns the failure
// count; exit 0 and byte-identical output on all four legs (interpreter/compiler x
// goja/-frozen) mean everything passed.
// Note: the two-argument Substring(a, b) is slice(a, b) here - b is an exclusive END
// index, not a length - so the checks below stick to those semantics.

using System;
using System.Collections.Generic;

namespace Demo
{
    class Counter
    {
        public int Value;
        public int Step = 1;                 // field initializer

        public Counter(int start)
        {
            this.Value = start;
        }

        public int Next()
        {
            this.Value += this.Step;
            return this.Value;
        }

        public void SetStep(int s)
        {
            this.Step = s;
        }

        public int Doubled() => this.Value * 2;      // expression-bodied instance method

        public static int Twice(int x) => x * 2;     // expression-bodied static method
    }

    class Point
    {
        public int X;
        public int Y;

        public Point(int x, int y)
        {
            this.X = x;
            this.Y = y;
        }

        public Point Plus(Point other)       // returns a fresh instance
        {
            return new Point(this.X + other.X, this.Y + other.Y);
        }

        public bool SamePoint(Point other)
        {
            return this.X == other.X && this.Y == other.Y;
        }
    }

    class Animal
    {
        public int Legs = 4;

        public virtual string Name()
        {
            return "animal";
        }

        public string Describe()             // this.Name() dispatches dynamically
        {
            return this.Name() + ":" + this.Legs;
        }

        public virtual int Base()
        {
            return 10;
        }
    }

    class Bird : Animal
    {
        public Bird()
        {
            this.Legs = 2;                   // the implicit base() ran the field inits first
        }

        public override string Name()
        {
            return "bird";
        }

        public override int Base()           // base.M() starts above the defining class
        {
            return base.Base() + 5;
        }
    }

    class Tag
    {
        public int Id;
        public Tag(int n) => this.Id = n * 2;        // expression-bodied constructor
    }

    class Box                                // for object initializers
    {
        public int W;
        public int H;

        public int Area()
        {
            return this.W * this.H;
        }
    }

    class Labeled                            // ctor arguments plus member initializer
    {
        public int Id;
        public int Extra = 1;

        public Labeled(int id)
        {
            this.Id = id;
        }
    }

    class Boom
    {
        public int Code;

        public Boom(int c)
        {
            this.Code = c;
        }
    }

    class Program
    {
        static int Fails = 0;
        static int Checks = 0;
        static int Calls = 0;
        static int FinRuns = 0;

        static void Check(string id, bool cond)
        {
            Program.Checks++;
            if (!cond)
            {
                Console.WriteLine("FAIL " + id);
                Program.Fails++;
            }
        }

        static bool Bump()
        {
            Program.Calls++;
            return true;
        }

        // ----- functions: early return, recursion, mutual recursion -----
        static string Grade(int n)
        {
            if (n > 10) { return "big"; }
            else if (n > 5) { return "mid"; }
            else { return "small"; }
        }
        static int Sign(int n)
        {
            if (n < 0) { return -1; }        // early return
            return 1;
        }
        static int Fib(int n)
        {
            if (n < 2) { return n; }
            return Program.Fib(n - 1) + Program.Fib(n - 2);
        }
        static bool IsEven(int n)
        {
            return n == 0 ? true : Program.IsOdd(n - 1);
        }
        static bool IsOdd(int n)
        {
            return n == 0 ? false : Program.IsEven(n - 1);
        }

        static T Echo<T>(T value) where T : class    // generic method, constraint ignored
        {
            return value;
        }

        // ----- switch helpers -----
        static int Classify(int x)           // stacked labels and default
        {
            int r = 0;
            switch (x)
            {
                case 0:
                    r = 100;
                    break;
                case 1:                      // stacked labels
                case 2:
                    r = 12;
                    break;
                case 3:
                    r = 3;
                    break;
                default:
                    r = -1;
                    break;
            }
            return r;
        }
        static string DayKind(string day)    // switch on string, return from a case
        {
            switch (day)
            {
                case "sat":
                case "sun":
                    return "weekend";
                default:
                    return "workday";
            }
        }

        // ----- delegates and closures -----
        static int Apply(Func<int, int> f, int v)
        {
            return f(v);
        }
        static Func<int, int> Adder(int n)   // returns a closure capturing n
        {
            return x => x + n;
        }

        // ----- declared-type adoption (docs/todo.md 1.1) and the run-time type
        // test behind `is` (docs/todo.md 1.9). ECMA-334 10.2.3 makes an untyped
        // integer literal at a `double`/`float`/sized-integral site take that
        // type - the WIDTH IS ON THE VALUE, not on the annotation - and 12.12.12
        // makes `E is T` a run-time type test with no implicit conversion. Both
        // halves used to agree on every wrong answer here, so --cross was blind.
        static double AdoptG(double x) { return x / 2; }
        static double AdoptR() { return 3; }
        static byte AdoptW(byte b) { return b; }
        class AdoptBox { public double A = 3; public double P { get; set; } = 3; }

        // ----- method groups, method overloads and the integral local write
        // (ECMA-334 10.8 / 20.1 / 20.5 / 20.2 / 12.8.10.2 for the groups, 12.6.4 for
        // the overloads, 10.2.3 at 12.21.2 for the write; docs/todo.md 1.1/1.2/1.3).
        // A method group did not exist anywhere: `this.M`, a bare `M` and a bare
        // static `H` all answered undefined or "unknown name" in BOTH halves, a
        // delegate FIELD called like a method aborted, and .Invoke() was
        // unsupported. Overloading was implemented for constructors only, so a class
        // kept the LAST declaration of each method name. And `long x; x = 3;` stored
        // the raw int, so `x * 1000000 * 1000000` was a 32-bit multiply.
        delegate int MgOp(int x);
        class MgK
        {
            public int Bias = 1;
            public MgOp E;
            public int F(int x) { return x + this.Bias; }
            public static int G(int x) { return x + 20; }
            public int ViaThis() { MgOp o = this.F; return o(10); }
            public int ViaBare() { MgOp o = F; return o(100); }
            public int ViaStatic() { MgOp o = G; return o(1000); }
            public int FireE(int x) { return E(x); }
        }
        class MgOv
        {
            public int M(int a) { return 1; }
            public int M(double a) { return 2; }
            public int M(int a, int b) { return 3; }
            public int M(string a) { return 4; }
            public static int S(int a) { return 11; }
            public static int S(string a) { return 12; }
            public int Only(int a) { return a + 1; }
        }
        static int MgFree(int x) { return x + 7; }
        // An OVERLOADED OPERATOR keeps every declaration too (ECMA-334 12.4.5 /
        // 15.10.1): the second used to overwrite the first in both halves.
        class MgV
        {
            public double X;
            public MgV(double x) { X = x; }
            public static MgV operator +(MgV a, MgV b) { return new MgV(a.X + b.X); }
            public static MgV operator +(MgV a, int b) { return new MgV(a.X + b); }
        }


        // ----- inherited statics, base overloads, operator applicability and
        // `ref` / `out` (docs/todo.md 1.1) -----
        // ECMA-334 15.3.1 (a base member IS a member of the derived class), 15.5.1
        // (one storage location per static field), 15.4 (a const member is
        // implicitly static), 12.6.4 (the candidate set moves to the base when the
        // derived type offers nothing applicable) with 15.3.5 for `new`, 12.4.5 with
        // 12.10.5 (a user operator that accepts neither operand is not applicable
        // and the predefined string concatenation stands) and 12.6.2.3.3 / .4 (a
        // `ref` / `out` parameter is an ALIAS for the argument's variable).
        // There is no C# toolchain here; every value is spec-cited.
        delegate int RfOp(int x);
        class RfBas
        {
            public static int F = 5;
            public static int P { get { return 9; } }
            public const int C = 11;
            public static int SB() { return 7; }
            public static RfOp BH = (int x) => x + 500;
            public int M(int x) { return 1; }
        }
        class RfDer : RfBas { public int M(string s) { return 2; } }
        class RfHide : RfBas { public new int M(string s) { return 3; } }
        class RfV
        {
            public int N;
            public RfV(int n) { N = n; }
            public override string ToString() { return "V" + N; }
            public static RfV operator +(RfV a, RfV b) { return new RfV(a.N + b.N); }
        }
        class RfBox
        {
            public int Fld;
            public int[] Arr = {1, 2, 3};
            public void Bump(ref int a) { a = a + 1; a++; a += 10; }
            public void ByVal(int a) { a = a + 1; a++; a += 10; }
            public static void Swap(ref int a, ref int b) { int t = a; a = b; b = t; }
            public static bool TryGet(int k, out int v) { v = k * 2; return k > 0; }
        }

        // ----- constructor initializers and constructor overloads (ECMA-334
        // 15.11.2 / 12.6.4, docs/todo.md 1.1 and 1.10). The ': base(d)' header used
        // to be parsed and DISCARDED - the base class' parameterless constructor ran
        // instead - and a class kept only its LAST constructor. Both halves agreed
        // on both wrong answers, so --cross was blind to them.
        class CtP { public double D; public double T = 1; public CtP(double d) { D = d; } public CtP() { D = -1; } }
        class CtC : CtP { public CtC(double d) : base(d) { } public CtC() : this(7) { T = T + 100; } }
        class CtOv
        {
            public double D;
            public CtOv() { D = 0; }
            public CtOv(int a) { D = 9; }
            public CtOv(double a) { D = a; }
            public CtOv(string a) { D = 5; }
        }
        // A property with accessor bodies and a plain field of the SAME NAME in
        // another class: neither may take the other's meaning.
        class CollAcc { private double v = 7; public double Coll { get { return v; } set { v = value; } } }
        class CollFld { public double Coll = 5; }
        // An event whose type is a BUILT-IN delegate (Action), with no `delegate`
        // declaration anywhere in this file: 'E += h' must combine invocation lists
        // (ECMA-334 15.8.1) and not do arithmetic (docs/todo.md 1.10).
        class EvBox
        {
            public event Action E;
            public void Fire() { Action h = E; if (h != null) { h(); } }
            public bool Empty() { return E == null; }
        }
        static string EvLog = "";

        // ----- an unqualified name resolving to a member of the enclosing type
        // (ECMA-334 12.8.4 / 12.8.7 / 12.7.3, docs/todo.md 1.2 and 1.1) -----
        class UnqBox
        {
            public double D = 1;
            public double U;
            public static double S = 2;
            // csPropAcc / csEvtAcc are keyed by member name ALONE in the compiler
            // half, so a property with accessor bodies used to CAPTURE every plain
            // field of the same name in every other class. `CollFld.Coll` below is
            // the assertion that it no longer does (docs/todo.md 1.10).
            public int Acc { get { return 20; } set { this.D = value; } }
            public void WrD(int v) { D = v; }          // instance field, unqualified
            public void WrU(int v) { U = v; }          // ... with no initializer
            public double RdS() { return S; }          // static, from an instance member
            public static void SWrS(int v) { S = v; }  // static, from a static member
            public int RdAcc() { return Acc; }         // property with accessor bodies
            public void WrAcc(int v) { Acc = v; }
            public double Shadow(double D) { return D; }   // CONTROL: the parameter wins
        }
        class UnqBase { public double BD = 1; }
        class UnqSub : UnqBase { public double RdBD() { return BD; } }

        // ----- exceptions -----
        static int Risky(int n)
        {
            if (n > 3) { throw new Boom(n); }        // unwinds out of the call
            return n * 2;
        }
        static int RethrowNested()
        {
            try
            {
                try { throw new Boom(1); } catch (Exception e) { throw new Boom(e.Code + 1); }
            }
            catch (Exception e2)
            {
                return e2.Code;
            }
        }
        static string RetAcrossTry()
        {
            try { return "from-try"; } finally { Program.FinRuns += 1; }
        }
        static int RetOutOfCatch(int n)
        {
            try
            {
                if (n > 0) { return n * 10; }        // return out of the try
                throw new Boom(0);
            }
            catch (Exception e)
            {
                return -1;                           // return out of the catch
            }
            finally
            {
                Program.FinRuns += 1;                // runs on both paths
            }
        }
        static int NestedReturn()
        {
            try
            {
                try { return 9; } finally { }
            }
            finally { }
            return 0;
        }
        static int RetInFinally()
        {
            try { return 1; } finally { return 2; }  // the finally's return overrides
        }
        static string FinCancelsThrow()
        {
            try { throw new Boom(9); } finally { return "fin"; }     // cancels the pending throw
        }
        static int BreakInFinally()
        {
            int i = 0;
            while (true)
            {
                i = i + 1;
                try { i = i + 10; } finally { break; }
            }
            return i;
        }
        static int ContinueInFinally()
        {
            int sum = 0;
            for (int i = 0; i < 3; i++)
            {
                try { if (i == 1) { throw new Boom(i); } } finally { continue; }
            }
            return sum;                              // the tail after try never runs, sum stays 0
        }
        static int LoopBreakOutOfTry()
        {
            int sum = 0;
            for (int i = 0; i < 6; i++)
            {
                try
                {
                    if (i == 3) { break; }
                    sum = sum + i;
                }
                finally { }
            }
            return sum;                              // 0+1+2 = 3
        }
        static int LoopContinueOutOfTry()
        {
            int sum = 0;
            for (int i = 0; i < 4; i++)
            {
                try
                {
                    if (i == 2) { continue; }
                    sum = sum + i;
                }
                catch (Exception e) { }
            }
            return sum;                              // 0+1+3 = 4
        }

        // ----- everything combined in one small pipeline (3-element data flow) -----
        static string Transform(List<int> list)
        {
            string outp = "";
            foreach (var n in list)
            {
                try
                {
                    if (n < 0) { throw new Boom(n); }
                    outp = outp + (n % 2 == 0 ? "e" : "o") + n;
                }
                catch (Exception e)
                {
                    outp = outp + "x";
                }
            }
            return outp;
        }

        static int Main()
        {
            // ----- numbers, arithmetic, precedence -----
            Program.Check("arith-precedence", 2 + 3 * 4 == 14);
            Program.Check("dbl-add", 1.5 + 1.5 == 3.0 && 2.5 + 1.25 == 3.75);
            Program.Check("arith-paren", (2 + 3) * 4 == 20);
            Program.Check("arith-unary-minus", -3 + 5 == 2);
            Program.Check("arith-div-trunc", 7 / 2 == 3);
            Program.Check("arith-div-neg", -7 / 2 == -3);
            Program.Check("arith-mod", 7 % 3 == 1);
            Program.Check("arith-mod-neg", -7 % 3 == -1);
            Program.Check("arith-chain", 20 - 5 - 3 == 12);
            int cx = 5;
            cx += 3;
            cx -= 2;
            cx *= 4;
            cx /= 6;
            cx %= 3;
            Program.Check("arith-compound", cx == 1);
            int pi = 5;
            int a1 = pi++;                   // postfix yields the old value
            pi++;
            Program.Check("arith-incdec", a1 == 5 && pi == 7);
            int pd = 5;
            pd--;
            Program.Check("arith-decrement", pd == 4);

            // ----- bitwise and shifts -----
            Program.Check("bit-and-or-xor", (6 & 3) == 2 && (6 | 3) == 7 && (6 ^ 3) == 5);
            Program.Check("bit-not", (~5) == -6);
            Program.Check("bit-shl", (1 << 4) == 16);
            Program.Check("bit-shr-neg", (-8 >> 1) == -4);
            int bf = 0x0F;
            bf |= 0x10;
            bf &= ~0x01;
            bf <<= 1;
            bf >>= 2;
            bf ^= 0x05;
            Program.Check("bit-compound", bf == 10);

            // ----- numeric and char literals -----
            Program.Check("num-hex", 0xFF == 255);
            Program.Check("num-binary", 0b1010 == 10);
            Program.Check("num-underscore", 1_000 == 1000);
            char gr = 'A';
            Program.Check("char-literal", gr == "A" && ("x" + 'y') == "xy");

            // ----- comparison, equality, logic -----
            Program.Check("cmp-ops", 5 > 3 && 3 >= 3 && 2 < 3 && 2 <= 2 && 1 != 2);
            Program.Calls = 0;
            bool noRun = false && Program.Bump();
            Program.Check("logic-and-skipped", Program.Calls == 0 && !noRun);
            bool skipRun = true || Program.Bump();
            Program.Check("logic-or-skipped", Program.Calls == 0 && skipRun);
            bool oneRun = Program.Bump() && true;
            Program.Check("logic-ran-once", Program.Calls == 1 && oneRun);
            Program.Check("logic-not", !(2 == 3) && !false);
            Program.Check("ternary", (5 > 3 ? "a" : "b") == "a" && (5 < 3 ? "a" : "b") == "b");
            Program.Check("ternary-nested", (5 > 3 ? (2 > 1 ? 1 : 2) : 3) == 1);
            var vt = 3 > 2 ? 10 : 20;        // var local
            Program.Check("var-local", vt == 10);

            // ----- strings -----
            Program.Check("str-concat", "foo" + "bar" == "foobar");
            string sa = "Hello";
            sa += ", world";
            Program.Check("str-concat-assign", sa == "Hello, world");
            Program.Check("str-int-concat", "n=" + 42 == "n=42" && 42 + "x" == "42x" && 1 + 2 + "x" == "3x");
            Program.Check("str-length", "hello".Length == 5 && "".Length == 0);
            Program.Check("str-unicode-len", "héllo".Length == 5);
            Program.Check("str-substring-tail", "hello".Substring(3) == "lo");
            Program.Check("str-substring-slice", "metacompiler".Substring(4, 8) == "comp");
            Program.Check("str-indexof", "hello".IndexOf("ll") == 2 && "hello".IndexOf("z") == -1);
            Program.Check("str-eq", "abc" == "abc" && "abc" != "abd");
            Program.Check("str-equals-method", "abc".Equals("abc") && !"abc".Equals("abd"));
            int ia = 3;
            int ib = 4;
            Program.Check("str-interpolation", $"{ia}+{ib}={ia + ib}" == "3+4=7" && $"sum={ia * 2 + 1}" == "sum=7");
            Program.Check("str-escapes", "a\tb".Length == 3 && "a\nb".Length == 3 && "\\".Length == 1 && "\"".Length == 1);
            // The range indexer on a String is Substring, so its bounds are
            // UTF-16 CODE UNITS (ECMA-334 12.8.12 / String.Substring) - not the
            // BYTES a Go slice would take. The ASCII row cannot tell the two
            // apart; the CJK one can, and answered a lone replacement character
            // while the shared js_goslice extern was carrying both languages.
            // No dotnet on this machine, so the BCL contract is the oracle.
            string rngAscii = "abcdef";
            Program.Check("str-range-ascii", rngAscii[1..3] == "bc" && rngAscii[..2] == "ab" && rngAscii[4..] == "ef");
            string rngCjk = "日本語";
            Program.Check("str-range-utf16", rngCjk.Length == 3 && rngCjk[0..1] == "日" && rngCjk[1..3] == "本語");
            int[] rngArr = { 1, 2, 3, 4 };
            int[] rngMid = rngArr[1..3];
            Program.Check("arr-range-copies", rngMid.Length == 2 && rngMid[0] == 2 && rngArr[1] == 2);

            // ----- control flow: if / while / do-while / for -----
            Program.Check("if-elseif-else", Program.Grade(11) == "big" && Program.Grade(7) == "mid" && Program.Grade(1) == "small");
            int w0 = 0;
            while (w0 > 0) { w0 = w0 - 1; }  // runs zero times
            Program.Check("while-zero", w0 == 0);
            int w3 = 0;
            while (w3 < 3) { w3 = w3 + 1; }  // runs three times
            Program.Check("while-three", w3 == 3);
            int dw = 0;
            do { dw = dw + 1; } while (false);       // body runs exactly once
            Program.Check("do-while-once", dw == 1);
            int forSum = 0;
            for (int i = 1; i <= 3; i++) { forSum += i; }
            Program.Check("for-basic", forSum == 6);
            string brk = "";
            for (int i = 0; i < 6; i++)
            {
                if (i == 2) { break; }
                brk = brk + i;
            }
            Program.Check("for-break", brk == "01");
            string cont = "";
            for (int i = 0; i < 4; i++)
            {
                if (i % 2 == 1) { continue; }
                cont = cont + i;
            }
            Program.Check("for-continue", cont == "02");
            string nested = "";
            for (int oi = 0; oi < 2; oi++)
            {
                for (int ii = 0; ii < 3; ii++)
                {
                    if (ii == 1) { break; }  // inner break must not end the outer loop
                    nested = nested + oi + ii;
                }
            }
            Program.Check("nested-break", nested == "0010");

            // ----- switch: match, stacked labels, default -----
            Program.Check("switch-match", Program.Classify(0) == 100);
            Program.Check("switch-stacked", Program.Classify(1) == 12 && Program.Classify(2) == 12);
            Program.Check("switch-case", Program.Classify(3) == 3);
            Program.Check("switch-default", Program.Classify(9) == -1);
            Program.Check("switch-string", Program.DayKind("sun") == "weekend" && Program.DayKind("tue") == "workday");

            // ----- arrays -----
            int[] arr = new int[3];
            Program.Check("arr-default", arr[0] == 0 && arr[2] == 0);
            Program.Check("arr-new-length", arr.Length == 3);
            arr[0] = 10;
            arr[1] = 20;
            arr[2] = 30;
            Program.Check("arr-store", arr[0] == 10 && arr[2] == 30);
            arr[1] += 5;
            Program.Check("arr-compound-elem", arr[1] == 25);
            int[] lit = new int[] { 3, 1, 4 };
            Program.Check("arr-literal", lit.Length == 3 && lit[2] == 4);
            int esum = 0;
            foreach (var v in lit) { esum += v; }
            Program.Check("arr-foreach", esum == 8);

            // ----- List<T> -----
            List<int> nums = new List<int>();
            nums.Add(10);
            nums.Add(20);
            nums.Add(30);
            Program.Check("list-add-count", nums.Count == 3 && nums[1] == 20);
            Program.Check("list-contains", nums.Contains(20) && !nums.Contains(99));
            nums[0] = 9;
            Program.Check("list-write", nums[0] == 9);
            List<int> xs = new List<int> { 2, 4, 6, 8 };     // collection initializer
            int csum = 0;
            foreach (var v in xs) { csum += v; }
            Program.Check("list-initializer", xs.Count == 4 && xs[2] == 6 && csum == 20);
            List<List<int>> grid = new List<List<int>>();    // nested lists
            grid.Add(new List<int> { 1, 2 });
            grid.Add(new List<int> { 3 });
            Program.Check("list-nested", grid[0][1] == 2 && grid[1][0] == 3);
            grid[0][0] = 7;
            Program.Check("list-nested-write", grid[0][0] == 7);

            // ----- Dictionary<K,V> -----
            Dictionary<string, int> ages = new Dictionary<string, int>();
            ages["alice"] = 30;
            ages["bob"] = 25;
            Program.Check("dict-get-count", ages["alice"] == 30 && ages.Count == 2);
            Program.Check("dict-haskey", ages.ContainsKey("bob") && !ages.ContainsKey("carol"));
            ages["alice"] += 1;
            Program.Check("dict-compound", ages["alice"] == 31);
            int vsum = 0;
            foreach (var val in ages.Values) { vsum += val; }
            Program.Check("dict-values", vsum == 56);
            int klen = 0;
            foreach (var key in ages.Keys) { klen += key.Length; }
            Program.Check("dict-keys", klen == 8);

            // ----- classes, inheritance -----
            Counter ctr = new Counter(10);
            Program.Check("class-field-init", ctr.Step == 1 && ctr.Value == 10);
            Program.Check("class-method", ctr.Next() == 11);
            ctr.SetStep(5);
            Program.Check("class-setter", ctr.Next() == 16 && ctr.Value == 16);
            Program.Check("class-expr-method", ctr.Doubled() == 32);
            Program.Check("class-static-expr-method", Counter.Twice(21) == 42);
            Tag tagObj = new Tag(21);
            Program.Check("class-expr-ctor", tagObj.Id == 42);
            Point p = new Point(3, -4);
            Point q = p.Plus(new Point(1, 1));
            Program.Check("class-returns-instance", q.X * 100 + q.Y == 397);
            Program.Check("obj-value-equality", q.SamePoint(new Point(4, -3)));
            Program.Check("obj-identity", p == p && !(p == new Point(3, -4)));
            Point none = null;
            Program.Check("obj-null", none == null);
            Animal an = new Animal();
            Program.Check("class-this-dispatch", an.Describe() == "animal:4");
            Bird bird = new Bird();
            Program.Check("class-override", bird.Describe() == "bird:2");
            Program.Check("class-base-call", bird.Base() == 15);
            Program.Check("class-inherited-field", bird.Legs == 2);
            Animal upcast = bird;
            Program.Check("class-upcast-dispatch", upcast.Name() == "bird");

            // ----- object and collection initializers -----
            Box b1 = new Box { W = 3, H = 4 };
            Program.Check("object-initializer", b1.Area() == 12);
            Box b3 = new Box();
            Program.Check("object-init-default", b3.Area() == 0);
            Labeled lab = new Labeled(7) { Extra = 9 };      // ctor args then member init
            Program.Check("ctor-plus-initializer", lab.Id == 7 && lab.Extra == 9);
            Labeled plainLab = new Labeled(3);
            Program.Check("ctor-no-initializer", plainLab.Extra == 1);

            // ----- statics, recursion, generics -----
            Program.Check("fn-early-return", Program.Sign(-9) == -1 && Program.Sign(9) == 1);
            Program.Check("fn-recursion", Program.Fib(6) == 8);
            Program.Check("fn-mutual-recursion", Program.IsEven(4) && Program.IsOdd(5));
            Program.Check("generic-method", Program.Echo("x") == "x");

            // ----- lambdas, delegates, closures -----
            Func<int, int> sq = x => x * x;                  // expression-bodied lambda
            Program.Check("lambda-expr", sq(6) == 36);
            Func<int, int, int> add2 = (a, b) => { return a + b; };  // block body, two params
            Program.Check("lambda-block", add2(19, 23) == 42 && add2(add2(1, 2), 3) == 6);
            int k = 10;
            Func<int, int> addk = x => x + k;                // capture by reference
            Program.Check("closure-capture", addk(5) == 15);
            k = 100;
            Program.Check("closure-sees-update", addk(5) == 105);
            int[] acc = new int[1];
            Action<int> bumpAcc = x => { acc[0] += x; };     // void lambda with a side effect
            bumpAcc(5);
            bumpAcc(7);
            Program.Check("action-side-effect", acc[0] == 12);
            Program.Check("lambda-as-arg", Program.Apply(x => x + 1, 41) == 42);
            Func<int, int> add10 = Program.Adder(10);        // independent returned closures
            Func<int, int> add100 = Program.Adder(100);
            Program.Check("closure-independent", add10(5) == 15 && add100(5) == 105 && add10(1) + add100(1) == 112);
            List<Func<int, int>> ops = new List<Func<int, int>>();
            ops.Add(x => x + 1);
            ops.Add(x => x * 2);
            ops.Add(x => x - 3);
            int piped = 10;
            foreach (var op in ops) { piped = op(piped); }
            Program.Check("delegate-list", piped == 19);     // ((10 + 1) * 2) - 3

            // ----- checked / lock / using run their bodies -----
            int clu = 0;
            checked { clu = clu + 1; }
            object gate = "gate";
            lock (gate) { clu = clu + 1; }
            using (var buf = new List<int>()) { buf.Add(clu); clu = clu + buf[0]; }
            Program.Check("checked-lock-using", clu == 4);

            // ----- exceptions: throw / catch / finally / control flow -----
            string exOrder = "";
            int exCode = 0;
            try
            {
                exOrder = exOrder + "t";
                throw new Boom(5);
            }
            catch (Exception e)
            {
                exOrder = exOrder + "c";
                exCode = e.Code;
            }
            finally
            {
                exOrder = exOrder + "f";
            }
            Program.Check("try-throw-catch-finally", exOrder == "tcf" && exCode == 5);
            string noThrow = "";
            try { noThrow = noThrow + "t"; } catch (Exception e) { noThrow = noThrow + "c"; } finally { noThrow = noThrow + "f"; }
            Program.Check("try-no-throw", noThrow == "tf");
            int caught = -1;
            try
            {
                Program.Risky(5);
                caught = -2;                                 // not reached
            }
            catch (Boom e) when (e.Code > 0)                 // filter parsed; first catch wins
            {
                caught = e.Code;
            }
            Program.Check("throw-unwinds-call", caught == 5);
            Program.Check("throw-no-throw-path", Program.Risky(2) == 4);
            int flagTyped = 0;
            try { throw new Boom(7); }
            catch (Boom) { flagTyped = 1; }                  // typed catch, no binding
            Program.Check("catch-typed-no-binding", flagTyped == 1);
            int flagBare = 0;
            try { throw new Boom(8); }
            catch { flagBare = 2; }                          // parenless catch
            Program.Check("catch-parenless", flagBare == 2);
            Program.Check("rethrow", Program.RethrowNested() == 2);
            Program.Check("return-across-try", Program.RetAcrossTry() == "from-try" && Program.FinRuns == 1);
            Program.Check("return-out-of-catch", Program.RetOutOfCatch(4) == 40 && Program.RetOutOfCatch(-1) == -1 && Program.FinRuns == 3);
            Program.Check("nested-return", Program.NestedReturn() == 9);
            Program.Check("return-in-finally", Program.RetInFinally() == 2);
            Program.Check("finally-cancels-throw", Program.FinCancelsThrow() == "fin");
            Program.Check("break-in-finally", Program.BreakInFinally() == 11);
            Program.Check("continue-in-finally", Program.ContinueInFinally() == 0);
            Program.Check("loop-break-out-of-try", Program.LoopBreakOutOfTry() == 3);
            Program.Check("loop-continue-out-of-try", Program.LoopContinueOutOfTry() == 4);

            // ----- everything combined -----
            Program.Check("combined-pipeline", Program.Transform(new List<int> { 1, 2, -3 }) == "o1e2x");

            // ----- a literal adopts its declared type; `is` reads the run-time
            // type. Every operand is read out of an ARRAY so the constant folders
            // cannot answer at compile time. No C# toolchain on this machine: the
            // values are ECMA-334 10.2.3 / 12.12.12 cited, not executed.
            int[] adoptN = {3, 250};
            Program.Check("adopt-parameter", Program.AdoptG(adoptN[0]) == 1.5);
            Program.Check("adopt-return", Program.AdoptR() / 2 == 1.5);
            Program.Check("adopt-param-width",
                (Program.AdoptW(adoptN[1]) is byte) && !(Program.AdoptW(adoptN[1]) is int));
            AdoptBox adoptBox = new AdoptBox();
            Program.Check("adopt-field", adoptBox.A / 2 == 1.5 && adoptBox.P / 2 == 1.5);
            double[] adoptArr = {3, 4};
            int[] adoptInts = {3, 4};
            Program.Check("adopt-array-elem", adoptArr[0] / 2 == 1.5 && adoptInts[0] / 2 == 1);
            object[] adoptObj = {adoptN[0], 5L, 1.5, 1.5f};
            Program.Check("is-runtime-type",
                (adoptObj[0] is int) && !(adoptObj[0] is long)
                && (adoptObj[1] is long) && !(adoptObj[1] is int)
                && (adoptObj[2] is double) && !(adoptObj[2] is float)
                && (adoptObj[3] is float) && !(adoptObj[3] is double));

            // ----- an unqualified member of the enclosing type: read and written,
            // instance and static, field and property; and the member's declared
            // type adopts the written value. The last row is the CONTROL - a
            // parameter of the same name must still win.
            int[] unqN = {3, 5};
            UnqBox unq = new UnqBox();
            unq.WrD(unqN[0]);
            Program.Check("unqualified-field", unq.D == 3 && unq.D / 2 == 1.5);
            unq.WrU(unqN[0]);
            Program.Check("unqualified-field-noinit", unq.U / 2 == 1.5 && (unq.U is double));
            Program.Check("unqualified-static", unq.RdS() == 2);
            UnqBox.SWrS(unqN[0]);
            Program.Check("unqualified-static-write", UnqBox.S == 3 && UnqBox.S / 2 == 1.5);
            Program.Check("unqualified-property", unq.RdAcc() == 20);
            UnqBox unq2 = new UnqBox();
            unq2.WrAcc(unqN[1]);
            Program.Check("unqualified-property-write", unq2.D == 5);
            Program.Check("unqualified-inherited", new UnqSub().RdBD() == 1 && new UnqSub().BD == 1);
            Program.Check("unqualified-shadowed", unq.Shadow(unqN[1]) == 5 && unq.D == 3);

            // ----- constructor initializers, constructor overloads, the
            // property/field name collision, and an event's '+=' -----
            int[] ctN = {3, 4, 5};
            double[] ctQ = {4.5};
            string[] ctW = {"x"};
            Program.Check("ctor-base-init", new CtC(ctN[0]).D == 3);
            Program.Check("ctor-this-init", new CtC().D == 7 && new CtC().T == 101);
            Program.Check("ctor-overload-int", new CtOv(ctN[1]).D == 9);
            Program.Check("ctor-overload-double", new CtOv(ctQ[0]).D == 4.5);
            Program.Check("ctor-overload-string", new CtOv(ctW[0]).D == 5);
            Program.Check("ctor-overload-none", new CtOv().D == 0);
            CollFld coll = new CollFld();
            coll.Coll = ctN[0];
            Program.Check("prop-field-collision", new CollAcc().Coll == 7 && coll.Coll == 3);
            EvBox ev = new EvBox();
            Action evA = () => { Program.EvLog = Program.EvLog + "a"; };
            Action evB = () => { Program.EvLog = Program.EvLog + "b"; };
            Program.Check("event-empty", ev.Empty());
            ev.E += evA;
            ev.E += evB;
            ev.Fire();
            ev.E -= evA;
            ev.Fire();
            Program.Check("event-combine", Program.EvLog == "abb");
            // The three declaration sites that still stored the raw int
            // (docs/todo.md 1.3): a local written after its declaration, an array
            // element, and a qualified member write.
            double adL;
            adL = ctN[0];
            Program.Check("adopt-local-write", adL / 2 == 1.5 && (adL is double));
            double[] adAr = new double[2];
            adAr[0] = ctN[0];
            Program.Check("adopt-array-elem", adAr[0] / 2 == 1.5 && (adAr[0] is double));
            AdoptBox adB = new AdoptBox();
            adB.A = ctN[0];
            Program.Check("adopt-qualified-write", adB.A / 2 == 1.5 && (adB.A is double));

            // ----- method groups, method overloads, integral local writes
            // (docs/todo.md 1.1, 1.2, 1.3) -----
            int[] mgN = {1, 2, 3};
            string[] mgW = {"x"};
            double[] mgQ = {1.5};
            MgK mgk = new MgK();
            Program.Check("mg-this", mgk.ViaThis() == 11);
            Program.Check("mg-bare", mgk.ViaBare() == 101);
            Program.Check("mg-bare-static", mgk.ViaStatic() == 1020);
            MgOp mgA = MgK.G;
            Program.Check("mg-qualified-static", mgA(mgN[0]) == 21);
            MgOp mgB = Program.MgFree;
            Program.Check("mg-enclosing-static", mgB(mgN[0]) == 8);
            MgOp mgC = mgk.F;
            MgK mgk2 = new MgK();
            mgk2.Bias = mgN[2];
            MgOp mgC2 = mgk2.F;
            Program.Check("mg-bound", mgC(mgN[0]) == 2 && mgC2(mgN[0]) == 4);
            Program.Check("mg-invoke", mgC.Invoke(mgN[2]) == 4);
            Program.Check("mg-memoised", mgk.F == mgk.F && mgk.F != mgk2.F);
            mgk.E = mgk.F;
            Program.Check("mg-delegate-field-call", mgk.E(mgN[2]) == 4 && mgk.FireE(mgN[1]) == 3);
            MgOv mgO = new MgOv();
            Program.Check("ovl-int", mgO.M(mgN[0]) == 1);
            Program.Check("ovl-double", mgO.M(mgQ[0]) == 2);
            Program.Check("ovl-arity", mgO.M(mgN[0], mgN[1]) == 3);
            Program.Check("ovl-string", mgO.M(mgW[0]) == 4);
            Program.Check("ovl-static-int", MgOv.S(mgN[0]) == 11);
            Program.Check("ovl-static-string", MgOv.S(mgW[0]) == 12);
            Program.Check("ovl-single", mgO.Only(mgN[1]) == 3);
            MgV mgv = new MgV(mgN[0]);
            Program.Check("ovl-operator-obj", (mgv + new MgV(mgN[1])).X == 3);
            Program.Check("ovl-operator-int", (mgv + mgN[1]).X == 3);
            long mgL;
            mgL = mgN[2];
            Program.Check("adopt-long-local", mgL * 1000000 * 1000000 == 3000000000000L);
            ulong mgU;
            mgU = mgN[2];
            Program.Check("adopt-ulong-local", mgU * 1000000 * 1000000 == 3000000000000UL);
            int mgI;
            mgI = mgN[2];
            Program.Check("adopt-int-local-control", mgI * 1000000 * 1000000 == 2112827392 && (mgI is int));


            // ----- inherited statics, base overloads, operator applicability,
            // `ref` / `out` (docs/todo.md 1.1) -----
            int[] rfN = {1, 2, 3};
            string[] rfW = {"s"};
            Program.Check("static-inherited-call", RfDer.SB() == 7);
            Program.Check("static-inherited-field", RfDer.F == 5);
            Program.Check("static-inherited-prop", RfDer.P == 9);
            Program.Check("static-const-member", RfBas.C == 11 && RfDer.C == 11);
            Program.Check("static-inherited-delegate", RfDer.BH(rfN[0]) == 501);
            RfDer.F = rfN[2] * 100;
            Program.Check("static-one-storage", RfBas.F == 300);
            Program.Check("base-overload-int", new RfDer().M(rfN[0]) == 1);
            Program.Check("base-overload-string", new RfDer().M(rfW[0]) == 2);
            Program.Check("base-overload-new-hides", new RfHide().M(rfW[0]) == 3);
            RfV rfv = new RfV(rfN[0]);
            Program.Check("operator-declines-concat", (rfW[0] + rfv) == "sV1");
            Program.Check("operator-applies", (rfv + new RfV(rfN[1])).ToString() == "V3");
            RfBox rfb = new RfBox();
            int rfa = rfN[0];
            rfb.Bump(ref rfa);
            Program.Check("ref-param-writes-back", rfa == 13);
            int rfk = rfN[0];
            rfb.ByVal(rfk);
            Program.Check("byval-param-control", rfk == 1);
            rfb.Bump(ref rfb.Arr[1]);
            Program.Check("ref-param-array-place", rfb.Arr[1] == 14);
            int rfp = rfN[0];
            int rfq = rfN[1];
            RfBox.Swap(ref rfp, ref rfq);
            Program.Check("ref-param-swap", rfp == 2 && rfq == 1);
            Program.Check("out-param-declaring", RfBox.TryGet(rfN[1], out int rfo) && rfo == 4);

            Console.WriteLine("features: " + Program.Checks + " checks, " + Program.Fails + " failures");
            return Program.Fails;
        }
    }
}
