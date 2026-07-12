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

            Console.WriteLine("features: " + Program.Checks + " checks, " + Program.Fails + " failures");
            return Program.Fails;
        }
    }
}
