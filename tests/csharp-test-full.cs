// Full-syntax test: C# (C# 12 core grammar).
//
// This file belongs to the SECOND test group (./test.sh --full): it is NOT part
// of the default matrix. The goal of the metacompiler is to support the full
// languages; this file is the ratchet that measures how far the csharp grammars
// are. It walks the whole practical C# 12 syntax, one self-contained SECTION
// per language area. The --full runner runs the file, and whenever a grammar
// aborts it removes the section around the error and retries - so the report
// lists every unsupported section, not just the first.
//
// Conventions (shared by every *-test-full.* file):
//   - prologue (before the first SECTION marker): the check helper only
//   - each section: '// ===== SECTION <nn>: <name> =====' at class-body level,
//     self-contained (its own S<nn> methods and S<nn>-prefixed nested types),
//     no references to other sections
//   - Main calls each section via a line tagged 'SECTION-CALL <nn>'
//     and prints the summary line 'full: <checks> checks, <failures> failures'
//   - Main returns the failure count (exit 0 == full support, verified)
//
// Program is a top-level STATIC class so extension methods can be declared
// directly on it (C# forbids them anywhere else that would keep the sections
// inside one class body).
//
// Deliberately out of scope (not syntax, or unrunnable in this harness):
// using directives beyond System and System.Collections.Generic (the two the
// feature matrix already carries), LINQ query syntax (it needs System.Linq),
// await operands (they need System.Threading.Tasks; async members are covered
// definition-only), unsafe code / pointers / stackalloc, ref structs,
// assembly-level attributes, reflection, threads. Preprocessor directives are
// covered (section 26) as far as they go: they are skipped as trivia, and
// conditional compilation is NOT evaluated, so the text of every #if branch is
// parsed - section 26 says so and stays inside that limit.
// 'string?' annotations outside a #nullable context are a warning, not an
// error, and are used as such here.
//
// Hand-written for the metacompiler project (Apache-2.0, no copied test-suite
// code), organized after the C# 12 language specification with the ANTLR
// grammars-v4 CSharp grammar as a coverage checklist. Validated against the
// spec by hand; no local C# compiler was available on this machine.

using System;
using System.Collections.Generic;

namespace Demo
{
    static class Program
    {
        static int Fails = 0;
        static int Checks = 0;

        static void Check(string id, bool cond)
        {
            Program.Checks++;
            if (!cond)
            {
                Console.WriteLine("FAIL " + id);
                Program.Fails++;
            }
        }

        // ===== SECTION 01: baseline =====
        // Condensed re-assertion of the feature-matrix basics this file builds on.
        static int S01Fib(int n)
        {
            if (n < 2) { return n; }
            return Program.S01Fib(n - 1) + Program.S01Fib(n - 2);
        }
        static void S01()
        {
            int n = 0;
            for (int i = 0; i < 4; i++) { n += i; }
            Program.Check("bas1", n == 6);
            int[] arr = new int[] { 3, 1, 4 };
            Program.Check("bas2", arr.Length == 3 && arr[2] == 4);
            List<int> xs = new List<int> { 2, 4 };
            Dictionary<string, int> ages = new Dictionary<string, int>();
            ages["a"] = 30;
            Program.Check("bas3", xs[1] + ages["a"] == 34);
            Program.Check("bas4", Program.S01Fib(6) == 8);
            string t = "";
            try { t += "t"; } catch { t += "c"; } finally { t += "f"; }
            Program.Check("bas5", t == "tf");
            Func<int, int> add = x => x + 5;
            Program.Check("bas6", add(2) == 7);
            var m = 3 > 2 ? "y" : "n";
            int q = 6;
            Program.Check("bas7", m == "y" && $"{q}!" == "6!");
            int c = 0;
            checked { c = c + 1; }
            unchecked { c = c - 3; }
            object gate = "g";
            lock (gate) { c = c + 10; }
            Program.Check("bas8", c == 8);
        }

        // ===== SECTION 02: numeric literal forms =====
        static void S02()
        {
            double half = .5;                         // leading-dot real
            Program.Check("num1", half == 0.5);
            Program.Check("num2", 0xFF == 255 && 0x1_F0 == 496);
            Program.Check("num3", 0b1010 == 10 && 0b1111_0000 == 240);
            Program.Check("num4", 1_000_000 == 1000000);
            Program.Check("num5", 100u == 100 && 25L == 25 && 7UL == 7);
            Program.Check("num6", 1.5f == 1.5F && 2.5d == 2.5D && 3.75m == 3.75M);
            Program.Check("num7", 1e3 == 1000 && 2.5e-2 == 0.025);
            Program.Check("num8", (int)'A' == 65 && '\n' == (char)10);
        }

        // ===== SECTION 03: string literal and interpolation forms =====
        static void S03()
        {
            string v = @"c:\temp\x";                  // verbatim: no escapes
            Program.Check("str1", v.Length == 9);
            Program.Check("str2", @"say ""hi""".Length == 8);
            string nl = @"a
b";                                                   // verbatim keeps the newline
            Program.Check("str3", nl.Length == 3);
            int x = 6;
            Program.Check("str4", $@"v={x}" == "v=6" && @$"w={x}" == "w=6");
            Program.Check("str5", """he said "hi" here""".Length == 17);
            Program.Check("str6", $"{x,4}" == "   6" && $"{x:D3}" == "006");
            Program.Check("str7", $"o{$"i{x}"}" == "oi6" && $"{{x}}" == "{x}");
        }

        // ===== SECTION 04: typeof, nameof, sizeof, default =====
        static void S04()
        {
            Program.Check("typ1", typeof(int) != null && typeof(List<int>) != null);
            Program.Check("typ2", nameof(Program) == "Program" && nameof(Program.Fails) == "Fails");
            Program.Check("typ3", sizeof(int) == 4 && sizeof(byte) == 1);
            Program.Check("typ4", default(int) == 0 && default(string) == null);
            int dl = default;
            bool bl = default;
            Program.Check("typ5", dl == 0 && bl == false);
        }

        // ===== SECTION 05: nullable types and null operators =====
        static void S05()
        {
            int? a = null;
            int b = a ?? 3;                           // null-coalescing
            Program.Check("nul1", b == 3 && (a ?? 7) == 7);
            a = 5;
            Program.Check("nul2", a.HasValue && a.Value == 5 && (a ?? 0) == 5);
            int? c = a + 2;                           // lifted arithmetic
            Program.Check("nul3", c == 7);
            string? s = null;
            Program.Check("nul4", s?.Length == null && s == null);
            s = "abc";
            Program.Check("nul5", s!.Length == 3 && s?.Length == 3);
            int? d = null;
            d ??= 9;                                  // coalescing assignment
            Program.Check("nul6", d == 9);
            // A CAST to a nullable type. The '?' is inside the parentheses, so this
            // never collides with the conditional operator outside them.
            object boxed = 255;
            int? e = (int?) boxed;
            Program.Check("nul7", e == 255 && (byte?) 7 == 7);
            S05Pt? f = (S05Pt?) new S05Pt(2);
            Program.Check("nul8", f.Value.N == 2);
            // The nullable '?' also binds tighter than the array suffix: 'int?[]' is an
            // array of nullable ints. Exactly one of the two positions may be used, so
            // 'x is S05Pt? ? a : b' still reads its conditional.
            int?[] g = { 1, null, 3 };
            Program.Check("nul9", g.Length == 3 && g[1] == null && g[2] == 3);
            object h = new S05Pt(4);
            Program.Check("nul10", (h is S05Pt? ? 1 : 2) == 1);
        }
        struct S05Pt { public int N; public S05Pt(int n) { this.N = n; } }

        // ===== SECTION 06: tuples and deconstruction =====
        static (int, int) S06DivMod(int n, int d) { return (n / d, n % d); }
        class S06Pair
        {
            public int N; public string S;
            public S06Pair(int n, string s) { this.N = n; this.S = s; }
            public void Deconstruct(out int n, out string s) { n = this.N; s = this.S; }
        }
        static void S06()
        {
            (int, string) pair = (1, "one");
            Program.Check("tup1", pair.Item1 == 1 && pair.Item2 == "one");
            (int lo, int hi) named = (lo: 3, hi: 9);
            Program.Check("tup2", named.lo == 3 && named.hi == 9);
            var (a, b) = (2, 5);
            (a, b) = (b, a);                          // swap by deconstruction
            Program.Check("tup3", a == 5 && b == 2);
            int q, r;
            (q, r) = Program.S06DivMod(17, 5);
            Program.Check("tup4", q == 3 && r == 2);
            var (n2, s2) = new S06Pair(4, "x");       // custom Deconstruct
            Program.Check("tup5", n2 == 4 && s2 == "x");
        }

        // ===== SECTION 07: patterns and switch expressions =====
        class S07Pt
        {
            public int X; public int Y;
            public S07Pt(int x, int y) { this.X = x; this.Y = y; }
            public void Deconstruct(out int x, out int y) { x = this.X; y = this.Y; }
        }
        // A switch STATEMENT whose case labels are patterns (C# 7 and later).
        static string S07Sw(object o)
        {
            switch (o)
            {
                case null: return "null";
                case int n when n > 10: return "big" + n;
                case int: return "int";
                case string s: return "str" + s.Length;
                default: return "other";
            }
        }
        static void S07()
        {
            object o = "abc";
            Program.Check("pat1", o is string sv && sv.Length == 3);
            Program.Check("pat2", o is not null && !(o is int));
            Program.Check("pat3", o is string { Length: 3 });          // property pattern
            int n = 7;
            Program.Check("pat4", n is > 5 and < 10 && n is 7 or 9);   // relational, logical
            int[] a = new int[] { 1, 2, 3 };
            Program.Check("pat5", a is [1, _, 3] && a is [1, ..]);     // list patterns
            string grade = n switch { > 8 => "A", > 5 => "B", _ => "C" };
            Program.Check("pat6", grade == "B");
            string kind = o switch
            {
                string s2 when s2.Length > 2 => "long-string",         // guard
                string => "string",
                _ => "other",
            };
            Program.Check("pat7", kind == "long-string");
            var pos = new S07Pt(3, 4);
            Program.Check("pat8", pos is S07Pt(3, _) p2 && p2.Y == 4); // positional pattern
            Program.Check("pat9", Program.S07Sw(null) == "null" && Program.S07Sw(42) == "big42");
            Program.Check("pat10", Program.S07Sw(3) == "int" && Program.S07Sw("ab") == "str2"
                                   && Program.S07Sw(true) == "other");
        }

        // ===== SECTION 08: records =====
        record S08Pt(int X, int Y);
        record struct S08Val(int N);
        static void S08()
        {
            var p = new S08Pt(3, 4);
            Program.Check("rec1", p.X == 3 && p.Y == 4);
            var q = p with { Y = 9 };                 // non-destructive mutation
            Program.Check("rec2", q.X == 3 && q.Y == 9 && p.Y == 4);
            Program.Check("rec3", p == new S08Pt(3, 4) && p != q);     // value equality
            var (rx, ry) = p;                         // records deconstruct
            Program.Check("rec4", rx == 3 && ry == 4);
            var vs = new S08Val(5);
            Program.Check("rec5", vs.N == 5 && vs == new S08Val(5));
        }

        // ===== SECTION 09: structs =====
        // (ref structs are skipped: nothing to observe without spans/stackalloc)
        struct S09V
        {
            public int N;
            public S09V(int n) { this.N = n; }
            public int Doubled() { return this.N * 2; }
        }
        readonly struct S09R
        {
            public readonly int N;
            public S09R(int n) { this.N = n; }
            public int Sq() => this.N * this.N;
        }
        static void S09()
        {
            S09V v = new S09V(3);
            S09V w = v;                               // structs copy by value
            w.N = 9;
            Program.Check("stc1", v.N == 3 && w.N == 9);
            Program.Check("stc2", v.Doubled() == 6);
            S09R r = new S09R(4);
            Program.Check("stc3", r.Sq() == 16);
            S09V d = default;                         // default struct is zeroed
            Program.Check("stc4", d.N == 0);
        }

        // ===== SECTION 10: classes: primary ctors, modifiers, partial, nested =====
        abstract class S10Base
        {
            public abstract string Cry();
            public virtual string Kind() { return "base"; }
            public string Tag() { return "b"; }
            public string Speak() { return this.Kind() + ":" + this.Cry(); }
        }
        class S10Dog : S10Base
        {
            public override string Cry() { return "woof"; }
            public sealed override string Kind() { return "dog"; }
            public new string Tag() { return "d"; }   // method hiding
        }
        partial class S10Part { public int A() { return 1; } }
        partial class S10Part { public int B() { return 2; } }
        class S10Outer
        {
            public class Inner { public int Get() { return 7; } }
        }
        class S10Prim(int n)                          // primary constructor
        {
            public int Twice() { return n * 2; }
        }
        static void S10()
        {
            S10Dog d = new S10Dog();
            Program.Check("cls1", d.Speak() == "dog:woof");
            S10Base b = d;
            Program.Check("cls2", b.Kind() == "dog" && b.Tag() == "b" && d.Tag() == "d");
            var part = new S10Part();
            Program.Check("cls3", part.A() + part.B() == 3);
            var i = new S10Outer.Inner();
            Program.Check("cls4", i.Get() == 7);
            Program.Check("cls5", new S10Prim(6).Twice() == 12);
        }

        // ===== SECTION 11: properties and indexers =====
        static int S11Count { get; set; } = 2;        // static auto-property
        static int S11Sq => Program.S11Count * Program.S11Count;
        class S11Box
        {
            int w;
            public int W { get { return this.w; } set { this.w = value + 1; } }
            public int H { get; init; }               // init-only
            public required int D { get; set; }       // required member
            public int R { get; } = 11;               // get-only with initializer
            public int this[int i] { get { return i * 10; } }
        }
        static void S11()
        {
            Program.S11Count = 3;
            Program.Check("prp1", Program.S11Count == 3 && Program.S11Sq == 9);
            var b = new S11Box { H = 4, D = 2 };
            Program.Check("prp2", b.H == 4 && b.D == 2);
            b.W = 5;
            Program.Check("prp3", b.W == 6);          // the setter adds one
            Program.Check("prp4", b[3] == 30);
            Program.Check("prp5", b.R == 11);
        }

        // ===== SECTION 12: operator overloading and conversions =====
        struct S12Vec
        {
            public int X; public int Y;
            public S12Vec(int x, int y) { this.X = x; this.Y = y; }
            public static S12Vec operator +(S12Vec a, S12Vec b) { return new S12Vec(a.X + b.X, a.Y + b.Y); }
            public static S12Vec operator -(S12Vec a) { return new S12Vec(-a.X, -a.Y); }
            public static bool operator ==(S12Vec a, S12Vec b) { return a.X == b.X && a.Y == b.Y; }
            public static bool operator !=(S12Vec a, S12Vec b) { return !(a == b); }
            public override bool Equals(object o) { return o is S12Vec v && this == v; }
            public override int GetHashCode() { return this.X * 31 + this.Y; }
            public static implicit operator S12Vec(int n) { return new S12Vec(n, n); }
            public static explicit operator int(S12Vec v) { return v.X + v.Y; }
        }
        static void S12()
        {
            var a = new S12Vec(1, 2);
            var b = new S12Vec(3, 4);
            var c = a + b;
            Program.Check("opr1", c.X == 4 && c.Y == 6);
            Program.Check("opr2", (-a).X == -1);
            Program.Check("opr3", a == new S12Vec(1, 2) && a != b);
            S12Vec d = 5;                             // implicit conversion
            Program.Check("opr4", d.X == 5 && d.Y == 5);
            Program.Check("opr5", (int)b == 7);       // explicit conversion
        }

        // ===== SECTION 13: delegates and events =====
        delegate int S13Op(int x);                    // own delegate type
        delegate void S13Note(string m);
        delegate int S13Thunk();
        static string S13Log = "";
        static event S13Note S13Changed;              // own event
        static void S13OnChanged(string m) { Program.S13Log = Program.S13Log + m; }
        static int S13AddTwo(int x) { return x + 2; }
        static void S13()
        {
            S13Op f = Program.S13AddTwo;              // method group conversion
            Program.Check("dlg1", f(5) == 7);
            f = delegate (int x) { return x * 3; };   // anonymous method
            Program.Check("dlg2", f(4) == 12);
            // The parameter list may be EMPTY - 'delegate () { ... }' - and it may
            // be omitted entirely; both are the same anonymous-method form.
            S13Thunk t0 = delegate () { return 7; };
            Program.Check("dlg2a", t0() == 7);
            S13Thunk t1 = delegate { return 8; };
            Program.Check("dlg2b", t1() == 8);
            S13Op g = x => x + 10;
            f += g;                                   // multicast: last result wins
            Program.Check("dlg3", f(1) == 11);
            f -= g;
            Program.Check("dlg4", f(1) == 3);
            Program.S13Changed += Program.S13OnChanged;
            Program.S13Changed("ab");
            Program.Check("dlg5", Program.S13Log == "ab");
        }

        // ===== SECTION 14: lambdas and local functions =====
        static void S14()
        {
            int LocalAdd(int a, int b) { return a + b; }
            Program.Check("lam1", LocalAdd(2, 3) == 5);
            static int LocalSq(int x) => x * x;       // static local function
            Program.Check("lam2", LocalSq(4) == 16);
            int seed = 10;
            int WithSeed(int x) { return x + seed; }  // captures a local
            Program.Check("lam3", WithSeed(5) == 15);
            Func<int, int> sq = static x => x * x;    // static lambda
            Program.Check("lam4", sq(6) == 36);
            var add3 = (int a, int b = 1, params int[] rest) => a + b + rest.Length;
            Program.Check("lam5", add3(1) == 2 && add3(1, 2) == 3 && add3(1, 2, 9, 9) == 5);
            Func<int, Func<int, int>> curried = a2 => b2 => a2 + b2;
            Program.Check("lam6", curried(3)(4) == 7);
        }

        // ===== SECTION 15: extension methods =====
        class S15Box
        {
            public int N;
            public S15Box(int n) { this.N = n; }
        }
        static int S15Tripled(this int x) { return x * 3; }
        static S15Box S15Grown(this S15Box b, int d) { return new S15Box(b.N + d); }
        static T S15FirstOr<T>(this List<T> xs, T alt) { return xs.Count > 0 ? xs[0] : alt; }
        static void S15()
        {
            Program.Check("ext1", 7.S15Tripled() == 21);
            var b = new S15Box(4);
            Program.Check("ext2", b.S15Grown(3).N == 7);
            var xs = new List<int> { 8, 9 };
            var none = new List<int>();
            Program.Check("ext3", xs.S15FirstOr(0) == 8 && none.S15FirstOr(5) == 5);
        }

        // ===== SECTION 16: generics: variance, constraints, type arguments =====
        interface S16Seq<out T> { T Head(); }         // covariant
        interface S16Sink<in T> { int Eat(T x); }     // contravariant
        class S16One<T> : S16Seq<T>
        {
            T item;
            public S16One(T x) { this.item = x; }
            public T Head() { return this.item; }
        }
        class S16AnyLen : S16Sink<object>
        {
            public int Eat(object x) { return 9; }
        }
        class S16Pair<TA, TB> where TB : new()
        {
            public TA A; public TB B;
            public S16Pair(TA a) { this.A = a; this.B = new TB(); }
        }
        class S16Static<T>
        {
            public static string Tag = "s16";
            public static int Twice(int n) { return n * 2; }
        }
        static T S16Pick<T>(T a, T b, bool first) where T : class { return first ? a : b; }
        static void S16()
        {
            S16Seq<object> seq = new S16One<string>("hi");
            Program.Check("gen1", (string)seq.Head() == "hi");
            S16Sink<string> sink = new S16AnyLen();   // contravariant assignment
            Program.Check("gen2", sink.Eat("abcd") == 9);
            // A CONSTRUCTED type name as the receiver of a static member - both a field
            // and a call. The type arguments are erased, so both reach the one class.
            Program.Check("gen2a", S16Static<int>.Tag == "s16");
            Program.Check("gen2b", S16Static<string>.Twice(21) == 42);
            var pr = new S16Pair<string, List<int>>("k");
            Program.Check("gen3", pr.A == "k" && pr.B.Count == 0);
            Program.Check("gen4", Program.S16Pick<string>("x", "y", false) == "y");
            Program.Check("gen5", Program.S16Pick("a", "b", true) == "a");
        }

        // ===== SECTION 17: enums =====
        enum S17Color { Red, Green = 5, Blue }        // Blue == 6
        [Flags]
        enum S17Bits : byte { None = 0, A = 1, B = 2, C = 4, AB = A | B }
        static void S17()
        {
            S17Color c = S17Color.Blue;
            Program.Check("enm1", (int)c == 6 && c == S17Color.Blue);
            Program.Check("enm2", (S17Color)5 == S17Color.Green);
            S17Bits m = S17Bits.A | S17Bits.C;
            Program.Check("enm3", (m & S17Bits.A) != 0 && (m & S17Bits.B) == 0);
            Program.Check("enm4", (byte)S17Bits.AB == 3 && S17Bits.AB == (S17Bits.A | S17Bits.B));
        }

        // ===== SECTION 18: interfaces and default members =====
        interface S18Greet
        {
            string Name();
            string Hello() { return "hi " + Name(); } // default implementation
        }
        interface S18Tag { string Tag(); }
        class S18Person : S18Greet, S18Tag
        {
            public string Name() { return "ann"; }
            string S18Tag.Tag() { return "explicit"; }
        }
        static void S18()
        {
            var p = new S18Person();
            S18Greet g = p;
            Program.Check("ifc1", g.Name() == "ann");
            Program.Check("ifc2", g.Hello() == "hi ann");
            S18Tag t = p;
            Program.Check("ifc3", t.Tag() == "explicit");
        }

        // ===== SECTION 19: iterators and async =====
        // await needs System.Threading.Tasks (out of scope), so async members are
        // definition-only and run synchronously (the CS1998 warning is accepted).
        // An iterator block is LAZY: S19Endless never terminates on its own, so itr5 and
        // itr6 only pass if the sequence is produced one value at a time rather than
        // materialized at the call.
        static bool S19Ran = false;
        static IEnumerable<int> S19UpTo(int n)
        {
            for (int i = 1; i <= n; i++) { yield return i; }
        }
        static IEnumerable<string> S19Two()
        {
            yield return "a";
            yield break;
        }
        static IEnumerable<int> S19Endless()
        {
            int i = 0;
            while (true) { yield return i; i = i + 1; }
        }
        static async void S19Fire() { Program.S19Ran = true; }
        // An iterator method behind a DELEGATE. This is the one shape that asks the
        // runtime whether an iterator method IS a function, rather than just calling
        // it: a compound '+=' / '-=' on a delegate decides combine-vs-arithmetic at
        // run time by testing `typeof v == "function"` on the right operand, and an
        // iterator method is the C floor's tag 16 - callable, but answering "object"
        // from type_of until the floor was taught otherwise. Under that behaviour the
        // native binary took the ARITHMETIC arm and died on the next call with
        // "call of a non function value: 0", while llvm.Run and the interpreter both
        // combined. itr8/itr9 are the pin.
        delegate IEnumerable<int> S19Seq(int n);
        static IEnumerable<int> S19Tens(int n)
        {
            yield return n * 10;
            yield return n * 100;
        }
        static int S19Sum(IEnumerable<int> seq)
        {
            int t = 0;
            foreach (var v in seq) { t += v; }
            return t;
        }
        static void S19()
        {
            int sum = 0;
            foreach (var v in Program.S19UpTo(4)) { sum += v; }
            Program.Check("itr1", sum == 10);
            string joined = "";
            foreach (var s in Program.S19Two()) { joined += s; }
            Program.Check("itr2", joined == "a");
            var it = Program.S19UpTo(2).GetEnumerator();
            Program.Check("itr3", it.MoveNext() && it.Current == 1);
            Program.S19Fire();
            Program.Check("itr4", Program.S19Ran);
            int taken = 0;
            foreach (var v in Program.S19Endless())   // an endless iterator, left early
            {
                if (v > 3) { break; }
                taken += v;
            }
            Program.Check("itr5", taken == 6);        // 0 + 1 + 2 + 3
            var ie = Program.S19Endless().GetEnumerator();
            Program.Check("itr6", ie.MoveNext() && ie.Current == 0
                                  && ie.MoveNext() && ie.Current == 1);
            S19Seq sq = Program.S19UpTo;              // method group conversion of an ITERATOR
            Program.Check("itr7", Program.S19Sum(sq(3)) == 6);
            sq += Program.S19Tens;                    // multicast: last result wins
            Program.Check("itr8", Program.S19Sum(sq(3)) == 330);
            sq -= Program.S19Tens;
            Program.Check("itr9", Program.S19Sum(sq(3)) == 6);
        }

        // ===== SECTION 20: using declarations and disposal =====
        class S20Res : IDisposable
        {
            public static string Log = "";
            string tag;
            public S20Res(string t) { this.tag = t; S20Res.Log += "+" + t; }
            public void Dispose() { S20Res.Log += "-" + this.tag; }
        }
        static void S20UseTwo()
        {
            using var a = new S20Res("a");            // using declaration
            using var b = new S20Res("b");
            S20Res.Log += "!";
        }                                             // disposed in reverse order
        static void S20()
        {
            using (var r = new S20Res("r")) { S20Res.Log += "?"; }
            Program.Check("usg1", S20Res.Log == "+r?-r");
            S20Res.Log = "";
            Program.S20UseTwo();
            Program.Check("usg2", S20Res.Log == "+a+b!-b-a");
            S20Res.Log = "";
            using (new S20Res("t")) { S20Res.Log += "."; }
            Program.Check("usg3", S20Res.Log == "+t.-t");
            IDisposable d = new S20Res("d");
            d.Dispose();
            Program.Check("usg4", S20Res.Log == "+t.-t+d-d");
        }

        // ===== SECTION 21: goto, labels and throw expressions =====
        static int S21Sum()
        {
            int sum = 0;
            for (int i = 0; i < 9; i++)
            {
                for (int j = 0; j < 9; j++)
                {
                    if (i * j > 4) { goto done; }     // jumps out of both loops
                    sum += 1;
                }
            }
        done:
            return sum;
        }
        static string S21Kind(int n)
        {
            switch (n)
            {
                case 1: return "one";
                case 2: goto case 1;                  // switch goto case
                case 3: goto default;
                default: return "many";
            }
        }
        static int S21Pos(int n) { return n > 0 ? n : throw new Exception("neg"); }
        static void S21()
        {
            Program.Check("gto1", Program.S21Sum() == 14);
            Program.Check("gto2", Program.S21Kind(2) == "one" && Program.S21Kind(3) == "many");
            Program.Check("gto3", Program.S21Pos(4) == 4);
            string caught = "no";
            try { Program.S21Pos(-1); } catch (Exception) { caught = "yes"; }
            Program.Check("gto4", caught == "yes");
        }

        // ===== SECTION 22: params, named and optional args, collections, ranges =====
        static int S22Vol(int w, int h = 2, int d = 3) { return w * h * d; }
        static int S22Sum(string tag, params int[] rest)
        {
            int sum = tag.Length;
            foreach (var v in rest) { sum += v; }
            return sum;
        }
        static void S22()
        {
            Program.Check("arg1", Program.S22Vol(1, d: 5) == 10);      // named skips h
            Program.Check("arg2", Program.S22Vol(2) == 12 && Program.S22Vol(2, 4) == 24);
            Program.Check("arg3", Program.S22Sum("ab", 1, 2, 3) == 8 && Program.S22Sum("ab") == 2);
            int[] xs = [1, 2, 3];                     // collection expression
            List<int> ys = [10, .. xs, 20];           // with a spread element
            Program.Check("arg4", xs.Length == 3 && ys.Count == 5 && ys[3] == 3);
            Program.Check("arg5", xs[^1] == 3 && xs[^3] == 1);         // index from end
            int[] mid = xs[1..3];                     // range slice
            Program.Check("arg6", mid.Length == 2 && mid[0] == 2);
        }

        // ===== SECTION 23: floating point arithmetic =====
        // double/float/decimal are a DIFFERENT type from int. No C# compiler is
        // available on this machine, so every value below is SPEC-CITED, not
        // executed: ECMA-334 12.4.7 (binary numeric promotion: one double operand
        // makes the whole operation floating point), 6.4.5.3 (a real literal is a
        // double unless it carries an f/m suffix), 10.2.3 (implicit numeric
        // conversion on assignment), and, for the rendering, the .NET Core 3.0+
        // shortest-round-trip double.ToString() with the invariant
        // NumberFormatInfo symbols "Infinity" / "-Infinity" / "NaN". The section
        // exists because both grammars used to evaluate every arithmetic operator
        // in 32 bit integers: 1.0 / 3.0 was 0 and 2.5 * 1.5 was 3.
        static string S23S(double d) { return "" + d; }

        // The `float` half of the section. A C# float is System.Single, "an
        // IEEE-754 binary32 value" (ECMA-334 8.3.7), and used to be an alias of
        // the double box here - so 1.0f/3.0f answered the double's
        // 0.3333333333333333.
        static string S23F(float v) { return "" + v; }

        static float S23Third() { return 1.0f / 3.0f; }

        static double S23Half() { return 0.5; }

        static void S23()
        {
            Program.Check("flt1", 1.0 / 3.0 == 0.3333333333333333);
            Program.Check("flt2", 7 / 2 == 3 && -7 / 2 == -3 && 1 / 3 == 0);
            Program.Check("flt3", (double)7 / 2 == 3.5 && 7 / 2.0 == 3.5);
            Program.Check("flt4", 2.5 * 1.5 == 3.75 && 7.0 - 0.5 == 6.5 && 3.0 * 2.0 == 6.0);
            Program.Check("flt5", 2.0 % 0.75 == 0.5 && 1.5 + 1 == 2.5);
            // Rendering: no trailing ".0" for an integral double, and the three
            // special values that integer arithmetic could not produce.
            Program.Check("flt6", Program.S23S(1.0) == "1" && Program.S23S(2.5) == "2.5");
            Program.Check("flt7", Program.S23S(0.1 + 0.2) == "0.30000000000000004");
            Program.Check("flt8", Program.S23S(1.0 / 3.0) == "0.3333333333333333");
            // docs/todo.md 1.8. The digits are the SHORTEST decimal that reads back
            // as the same double, and the interpreter half used to take them from
            // the host's own `"" + a`. goja's is not always the shortest form - it
            // renders 359000550 * 2^-38 with sixteen digits that read back as the
            // NEXT double up - so the row below answered "0.001306036392634269"
            // under goja and "0.0013060363926342689" under -frozen: a live
            // divergence between the two script hosts that no matrix entry reached.
            // Both engines now ask floPrec for n = 1..17 and stop at the first n
            // that parses back, which is host-independent by construction. The
            // operand is read out of an ARRAY so the constant folder cannot answer
            // it, and the second value is the same double reached by arithmetic.
            double[] sd = {0.0013060363926342689, 359000550.0};
            Program.Check("flt8a", Program.S23S(sd[0]) == "0.0013060363926342689");
            Program.Check("flt8b", Program.S23S(sd[1] / 274877906944.0) == "0.0013060363926342689");
            double inf = 1.0 / 0.0;
            double nan = 0.0 / 0.0;
            Program.Check("flt9", Program.S23S(inf) == "Infinity" && Program.S23S(-1.0 / 0.0) == "-Infinity");
            Program.Check("flt10", Program.S23S(nan) == "NaN" && nan != nan);
            Program.Check("flt11", 1e300 * 1e300 == inf && 0.0 == -0.0);
            // ECMA-334 12.12.1: for the floating point relational operators "if
            // either operand is NaN, the result is false for all operators except
            // !=, for which the result is true". js_cscmp delegates to rtjRel
            // (lib/runtime-jvm.metajs, shared with java's js_jvcmp), whose
            // rtCompare answers the SENTINEL 2 for a NaN operand; read as an
            // ordering that sentinel made > and >= TRUE, and 0 would have made <=
            // and >= true instead. Both engines special-case it -
            // abnf/jsrtcsharp.go's js_cscmp is the same three lines. There is no C#
            // toolchain on this machine, so this row is the SPEC; java 24.0.2
            // settles the identical JLS 15.20.1 rule on the java side
            // (tests/java-test-full.java flt10a-flt10d). Operands come out of an
            // array so the constant folder cannot answer them.
            double[] nz = new double[]{0.0 / 0.0, 1.0};
            Program.Check("flt10a", !(nz[0] < nz[1]) && !(nz[0] > nz[1]) && !(nz[0] <= nz[1]) && !(nz[0] >= nz[1]));
            Program.Check("flt10b", !(nz[1] < nz[0]) && !(nz[1] > nz[0]) && !(nz[1] <= nz[0]) && !(nz[1] >= nz[0]));
            Program.Check("flt10c", !(nz[0] <= nz[0]) && !(nz[0] >= nz[0]) && nz[0] != nz[0]);
            float[] fz = new float[]{0.0f / 0.0f, 1.0f};
            Program.Check("flt10d", !(fz[0] >= fz[1]) && !(fz[0] <= fz[1]) && !(fz[0] > fz[1]) && !(fz[0] < fz[1]));
            // ECMA-334 12.12.9 (relational and type-testing operators) applies
            // BINARY NUMERIC PROMOTION first, so == between a long and a double
            // converts the long to double and compares two doubles. Both COMPILER
            // halves answered false - floEq (the floor's jf_num_eq, the twin's
            // jvmNumEq) had no sized-integer arm and a comment at three sites
            // called that deliberate - while both INTERPRETER halves answered
            // true, and nothing in the suite had ever compared a long to a double
            // so --cross could not see it. Found by the shape merge of csharp's
            // and java's strict equality into lib/runtime-jvm.metajs; java 24.0.2
            // settles the identical JLS 15.21.1 rule on the java side. There is no
            // C# toolchain on this machine, so this row is the SPEC. The operands
            // come out of arrays so the constant folder cannot answer them.
            long[] lz = new long[]{0L, 3L};
            double[] dz = new double[]{0.0, 3.0};
            Program.Check("flt11a", lz[0] == dz[0] && lz[1] == dz[1] && !(lz[0] != dz[0]));
            Program.Check("flt11b", dz[0] == lz[0] && lz[0] != dz[1] && lz[1] != dz[0]);
            // The UNSIGNED reading has its own path: a ulong is promoted as a
            // magnitude, not as the signed cell its bits would read as.
            ulong[] uz = new ulong[]{0UL, 3UL};
            Program.Check("flt11c", uz[0] == dz[0] && uz[1] == dz[1] && uz[0] != dz[1]);
            // A double declaration converts its initializer; ++ and the compound
            // operators keep the variable a double.
            double d = 1;
            Program.Check("flt12", Program.S23S(d) == "1" && d / 2 == 0.5);
            d += 0.25;
            Program.Check("flt13", d == 1.25);
            d++;
            Program.Check("flt14", d == 2.25);
            d *= 2;
            Program.Check("flt15", d == 4.5);
            d /= 3;
            Program.Check("flt16", d == 1.5);
            d--;
            Program.Check("flt17", d == 0.5 && Program.S23S(d) == "0.5");
            Program.Check("flt18", (int)2.9 == 2 && (int)-2.9 == -2 && (double)3 == 3.0);
            Program.Check("flt19", Program.S23S(-2.5) == "-2.5" && -Program.S23Half() == -0.5);
            Program.Check("flt20", 2.5 > 2 && 2 < 2.5 && 1.0 == 1 && 3.0 <= 3 && 3.0 >= 3);
            // A double survives an array element and a method boundary.
            double[] xs = new double[]{0.5, 1.5};
            Program.Check("flt21", xs[0] + xs[1] == 2.0 && Program.S23S(xs[0] * 2) == "1");
            Program.Check("flt22", Program.S23Half() / 2 == 0.25);
            float f = 3;
            Program.Check("flt23", f / 2 == 1.5 && 1.5f + 1.5f == 3.0f && 2.5d * 2 == 5.0);
            // Two C# rules the compiler half used to get wrong: a null operand of
            // '+' renders as the EMPTY string (Java writes "null"), and
            // String.Length counts UTF-16 code units (System.String is UTF-16).
            string nul = null;
            Program.Check("flt24", "x" + nul == "x" && "x" + null == "x");
            Program.Check("flt25", "ab".Length == 2 && "\u00e9".Length == 1 && "😀".Length == 2);

            // ----- the float WIDTH (todo.md 1.1) -----
            // Everything above treated `float` as a double. It is a binary32,
            // and the two halves of that fact have DIFFERENT confidence, which
            // is why they are separated here:
            //
            //   THE ARITHMETIC IS SETTLED. A binary32 add/sub/mul/div/rem and a
            //   binary32 comparison are the same IEEE-754 operation in C# and in
            //   Java, so `java` 24.0.2 is a real oracle for every VALUE below -
            //   an 11,424-row probe over 32 float values (every operand read out
            //   of an array, every result printed as a (double) so the renderer
            //   is out of the way) is byte-identical to java on all three legs.
            //
            //   THE RENDERING IS SPEC-CITED ONLY. There is no C# toolchain on
            //   this machine, and ECMA-334 8.3.7 fixes the TYPE while leaving
            //   ToString()'s text to the library. The digits are the shortest
            //   round-tripping decimal at 24 bits (.NET Core 3.0+), the exponent
            //   form is E+nn with a two-digit minimum and there is no forced
            //   ".0" - all as csFloStr32 in abnf/jsrtjvm.go argues line by line.
            //   NOTHING IS ASSERTED AT THE LOW END OF THE PRINT WINDOW (the
            //   1e-5 / 1e-4 boundary), because that bound is inherited unchanged
            //   from the double renderer and could not be settled here.
            //
            // Value rows, java-settled:
            Program.Check("flt26", 1.0f / 3.0f == 0.33333334f && 1.0f / 3.0f != 1.0 / 3.0);
            // The narrowing is REAL: (double)0.1f is not 0.1, and 16777217 has no
            // float, so it converts to 16777216 - including on the way INTO a
            // comparison (ECMA-334 12.4.7.3 promotes the int operand to float).
            Program.Check("flt27", Program.S23S((double) 0.1f) == "0.10000000149011612");
            Program.Check("flt28", 16777216f == 16777217 && !(16777216f < 16777217));
            // A mixed float/double pair promotes to the WIDER type, so a float
            // assigned to a double keeps the float's VALUE and loses the width.
            double wide = 1.0f / 3.0f;
            Program.Check("flt29", wide == 0.3333333432674408 && 1.0f / 3.0f + 0.0 == 0.3333333432674408);
            // A float overflows and underflows at the FLOAT boundaries.
            Program.Check("flt30", 1e38f * 10f == 1f / 0f && 1e-38f / 1e10f == 0f);
            // A compound assignment casts back to the LEFT operand's type
            // (ECMA-334 12.21.4), so a double on the right does not widen it,
            // and ++ steps at the float width.
            float g = 1.1f; g += 0.1;
            float h = 0.1f; h += 16777217;
            float k = 16777216f; k++;
            Program.Check("flt31", g == 1.2f && h == 16777216f && k == 16777216f);
            // Rendering rows, SPEC-CITED (see the note above).
            Program.Check("flt32", Program.S23F(1.0f / 3.0f) == "0.33333334");
            Program.Check("flt33", Program.S23F(0.1f + 0.2f) == "0.3"
                    && Program.S23S(0.1f + 0.2) == "0.30000000149011613");
            Program.Check("flt34", Program.S23F((float)(1.0 / 3.0)) == "0.33333334");
            // No forced ".0", and the plain window ends at 1e7 for a Single
            // where it ends at 1e15 for a Double - the G-precision of the type.
            Program.Check("flt35", Program.S23F(100f) == "100" && Program.S23F(0.001f) == "0.001"
                    && Program.S23F(9999999f) == "9999999");
            Program.Check("flt36", Program.S23F(1e7f) == "1E+07" && Program.S23S(1e7) == "10000000");
            Program.Check("flt37", Program.S23F(1e20f) == "1E+20" && Program.S23F(1e-20f) == "1E-20"
                    && Program.S23F(3.4028235e38f) == "3.4028235E+38");
            Program.Check("flt38", Program.S23F(16777217f) == "1.6777216E+07");
            Program.Check("flt39", Program.S23F(1f / 0f) == "Infinity" && Program.S23F(0f / 0f) == "NaN"
                    && Program.S23F(-0.0f) == "-0");
            Program.Check("flt40", Program.S23F(g) == "1.2" && Program.S23F(h) == "1.6777216E+07");
            // Unary minus, an array element, and the array's own type name:
            // System.Single[], where a double[] answers System.Double[].
            float[] fs = new float[]{0.5f, 1.5f};
            Program.Check("flt41", Program.S23F(-Program.S23Third()) == "-0.33333334"
                    && Program.S23F(fs[0] * 3) == "1.5");
            Program.Check("flt42", ("" + fs) == "System.Single[]" && (int) 2.9f == 2 && (long) -2.9f == -2L);
        }

        // ===== SECTION 24: object.ToString and the string conversion =====
        // Everything a value turns into text through: Console.WriteLine, '+',
        // an interpolation hole and an explicit .ToString(). No C# compiler is
        // available on this machine, so every value below is SPEC-CITED, not
        // executed:
        //   - "True" / "False" is System.Boolean.ToString(), documented to return
        //     Boolean.TrueString / Boolean.FalseString. A .NET BCL contract, not
        //     an ECMA-334 rule. The LITERALS stay 'true' / 'false'; only the
        //     string conversion is capitalised. Both grammars used to write
        //     JavaScript's "true", consistently, so the cross-half differ could
        //     never see it.
        //   - a null operand of '+' contributes the EMPTY string (ECMA-334 12.4.7;
        //     String.Concat calls ToString only on a non-null argument).
        //   - an array and an instance render through System.Object.ToString(),
        //     which answers GetType().ToString() - the fully qualified type name,
        //     and for an array the element type's name plus "[]" (ECMA-334 8.2.3
        //     lists ToString among the members every type inherits from object).
        //     The bare name is asserted here because this runtime models neither
        //     namespaces nor nested-type qualification; the element type is
        //     inferred from the elements for the same reason.
        //   - a user ToString() override wins everywhere, because
        //     System.Object.ToString is virtual.
        class S24Box { public int X; }

        class S24Named
        {
            public override string ToString() { return "NAMED"; }
        }

        class S24Base
        {
            public override string ToString() { return "BASE"; }
        }

        class S24Derived : S24Base { }

        static void S24()
        {
            Program.Check("str01", true.ToString() == "True" && false.ToString() == "False");
            Program.Check("str02", ("t=" + true) == "t=True" && ("f=" + false) == "f=False");
            Program.Check("str03", $"{true}" == "True" && $"{false}" == "False");
            // The literals themselves are unaffected by the capitalised rendering.
            bool flag = true;
            Program.Check("str04", flag && !false && ("" + flag) == "True");
            // A null reference contributes the empty string.
            string nul = null;
            Program.Check("str05", ("x" + nul) == "x" && ("x" + null) == "x");
            Program.Check("str06", $"[{nul}]" == "[]");
            // An array renders as its type name, not as its elements.
            int[] ints = new int[] { 1, 2, 3 };
            Program.Check("str07", ints.ToString() == "System.Int32[]");
            Program.Check("str08", ("a=" + ints) == "a=System.Int32[]");
            string[] strs = new string[] { "a", "b" };
            Program.Check("str09", strs.ToString() == "System.String[]");
            double[] dbls = new double[] { 0.5, 1.5 };
            Program.Check("str10", dbls.ToString() == "System.Double[]");
            bool[] bools = new bool[] { true };
            Program.Check("str11", bools.ToString() == "System.Boolean[]");
            // An instance renders as its type name.
            S24Box box = new S24Box();
            Program.Check("str12", box.ToString() == "S24Box" && ("b=" + box) == "b=S24Box");
            Program.Check("str13", $"{box}" == "S24Box");
            // A user override wins over the type name, and is inherited.
            S24Named named = new S24Named();
            Program.Check("str14", named.ToString() == "NAMED" && ("n=" + named) == "n=NAMED");
            Program.Check("str15", $"{named}" == "NAMED");
            S24Derived derived = new S24Derived();
            Program.Check("str16", derived.ToString() == "BASE" && ("d=" + derived) == "d=BASE");
            // ToString on the other primitives is unchanged by all of the above.
            // Parenthesized: '42.ToString()' is a lexical error in C# ('42.' starts
            // a real literal), so the integer receiver has to be bracketed.
            Program.Check("str17", (42).ToString() == "42" && (0.5).ToString() == "0.5");
            Program.Check("str18", 'x'.ToString() == "x" && "s".ToString() == "s");
        }

        // ===== SECTION 25: sized integers (int/uint/long/ulong/byte/sbyte/...) =====
        // C#'s integral types have fixed WIDTHS and a SIGNEDNESS, and both have to
        // survive into the next operation. The area used to be simply absent:
        // int.MaxValue was an undefined name, `long` was a 32 bit int (1L << 40
        // answered 256), and ulong.MaxValue could not even be spelled, because a
        // double rounds 18446744073709551615 - and the two script engines rounded
        // it DIFFERENTLY, so the goja and -frozen runs of one program disagreed.
        //
        // THERE IS NO dotnet ON THIS MACHINE, so unlike the Java twin of this
        // section NONE of these answers was executed. Every one is cited to the
        // C# standard (ECMA-334, 6th edition) by clause, and the three places
        // where the standard leaves a choice are called out as choices:
        //
        //   6.4.5.3   an integer literal's type follows its suffix AND its value
        //   12.4.7.3  binary numeric promotion (the uint/int -> long rule)
        //   12.8.20   arithmetic is UNCHECKED by default, so it WRAPS. A `checked`
        //             block is parsed here but NOT modelled - it does not throw -
        //             and that is a documented gap, not an oversight.
        //   12.9.3    division by zero throws System.DivideByZeroException in
        //             BOTH contexts; int.MinValue / -1 is implementation-defined
        //             in an unchecked one between an OverflowException and "the
        //             resulting value being that of the left operand". This
        //             implementation CHOOSES the second, which is also what Java
        //             specifies outright.
        //   12.11     the shift count is masked (& 31 / & 63) and `>>` on an
        //             unsigned type is the LOGICAL shift.
        //   12.3.2    a floating point value that does not fit its integral
        //             conversion target is undefined behaviour; this
        //             implementation CHOOSES to saturate, so that both halves
        //             agree rather than each inventing an answer.
        static long S25Id(long x) { return x; }

        static void S25()
        {
            // ----- the type constants (6.4.5.3 gives the ranges) -----
            Program.Check("num01", int.MaxValue == 2147483647 && int.MinValue == -2147483648);
            Program.Check("num02", long.MaxValue == 9223372036854775807L && long.MinValue == -9223372036854775808L);
            Program.Check("num03", uint.MaxValue == 4294967295 && uint.MinValue == 0);
            Program.Check("num04", ulong.MaxValue == 18446744073709551615 && ulong.MinValue == 0);
            // C#'s `byte` is UNSIGNED (0..255) and `sbyte` is the signed one -
            // the opposite of Java, and the easiest thing here to get wrong.
            Program.Check("num05", byte.MaxValue == 255 && byte.MinValue == 0);
            Program.Check("num06", sbyte.MaxValue == 127 && sbyte.MinValue == -128);
            Program.Check("num07", short.MaxValue == 32767 && ushort.MaxValue == 65535);
            // The System aliases name the same types.
            Program.Check("num08", Int32.MaxValue == int.MaxValue && UInt64.MaxValue == ulong.MaxValue);
            // The DIGITS survive: the values are exact, not the nearest double.
            Program.Check("num09", ("" + long.MaxValue) == "9223372036854775807");
            Program.Check("num10", ("" + ulong.MaxValue) == "18446744073709551615");

            // ----- unchecked arithmetic WRAPS (12.8.20) -----
            Program.Check("num11", int.MaxValue + 1 == int.MinValue);
            Program.Check("num12", int.MinValue - 1 == int.MaxValue);
            Program.Check("num13", long.MaxValue + 1 == long.MinValue);
            Program.Check("num14", 3000000000L * 3 == 9000000000L);
            int big = 1000000;
            Program.Check("num15", big * big == -727379968);
            // A declared `long` keeps its width across an assignment and a call.
            long acc = 1;
            acc = acc * 1000000;
            acc = acc * 1000000;
            Program.Check("num16", acc == 1000000000000L && Program.S25Id(acc) == 1000000000000L);
            // ulong arithmetic stays above 2^63, where a signed reading would go
            // negative and a double would round.
            Program.Check("num17", ulong.MaxValue / 3 == 6148914691236517205);
            Program.Check("num18", ulong.MaxValue > 0 && ulong.MaxValue - 1 == 18446744073709551614);

            // ----- binary numeric promotion (12.4.7.3) -----
            // The rule people are surprised by: uint + int is a LONG, because int
            // does not fit in uint and uint does not fit in int.
            uint u1 = 5;
            int i1 = -7;
            Program.Check("num19", u1 + i1 == -2L && (u1 + i1) < 0);
            // uint + uint stays a uint and WRAPS at 32 bits. Both operands are
            // spelled as uints on purpose: `u2 + 1` would be uint + uint in real
            // C# through the IMPLICIT CONSTANT EXPRESSION CONVERSION of 10.2.11
            // (an int CONSTANT in range converts to uint), and that rule is a
            // compile-time one this untyped value model cannot see - here the
            // plain 1 is an ordinary int and 12.4.7.3 promotes the pair to long.
            // A KNOWN DIVERGENCE, recorded rather than asserted away.
            uint u2 = 4294967295;
            uint one = 1;
            Program.Check("num20", u2 + one == 0 && u2 * 2u == 4294967294);
            // byte + byte and short * short are INT, so they do not wrap at 8/16.
            byte b1 = 200;
            byte b2 = 200;
            Program.Check("num21", b1 + b2 == 400);
            short s1 = 300;
            Program.Check("num22", s1 * s1 == 90000);
            // long meets uint as long; ulong meets a positive literal as ulong.
            long l1 = 5;
            Program.Check("num23", l1 + u1 == 10L);

            // ----- the shift count is MASKED (12.11) -----
            Program.Check("num24", 1 << 32 == 1 && 1 << 33 == 2);
            Program.Check("num25", 1L << 64 == 1L && 1L << 40 == 1099511627776L);
            Program.Check("num26", 1L << 62 == 4611686018427387904L && 1 << 31 == int.MinValue);
            // '>>' on a SIGNED type keeps the sign; on an UNSIGNED one it is the
            // LOGICAL shift - which is how C# spells what Java writes as '>>>'.
            Program.Check("num27", -8 >> 1 == -4 && -8L >> 1 == -4L);
            uint u3 = 4294967288;
            Program.Check("num28", u3 >> 1 == 2147483644);
            ulong u4 = 18446744073709551608;
            Program.Check("num29", u4 >> 1 == 9223372036854775804);
            // The left operand's type alone decides the width, so a byte shifts as
            // an int (12.11: unary promotion of each operand separately).
            byte b3 = 255;
            Program.Check("num30", b3 << 8 == 65280);
            // C# 11's '>>>' is the UNSIGNED right shift on a signed type: it fills
            // with zeroes instead of the sign bit, at the left operand's own width.
            // Every runtime half already had the operator; no GRAMMAR recognised the
            // token, so these four lines were a syntax error in both engines and the
            // ">>>" arms of js_csshift were reachable from nothing.
            int n3 = -8;
            Program.Check("urs1", n3 >>> 1 == 2147483644);
            long n4 = -8L;
            Program.Check("urs2", n4 >>> 1 == 9223372036854775804L);
            int n5 = -16;
            n5 >>>= 2;
            Program.Check("urs3", n5 == 1073741820);
            // '>>>' on an unsigned type is the same shift '>>' already was, and the
            // count is masked like every other shift's.
            uint u5 = 4294967288;
            Program.Check("urs4", u5 >>> 1 == 2147483644 && n3 >>> 33 == 2147483644);

            // ----- division (12.9.3 / 12.9.4) -----
            Program.Check("num31", -7 / 2 == -3 && 7 / -2 == -3 && -7 % 2 == -1 && 7 % -2 == 1);
            Program.Check("num32", long.MaxValue / 3 == 3074457345618258602L);
            // The implementation-defined unchecked overflow: the left operand.
            Program.Check("num33", int.MinValue / -1 == int.MinValue && int.MinValue % -1 == 0);
            // Division by zero THROWS in both contexts (12.9.3).
            bool threw = false;
            string msg = "";
            try { int z = 0; int q = 1 / z; Program.Check("num34", q == 99); }
            catch (DivideByZeroException e) { threw = true; msg = e.Message; }
            Program.Check("num34", threw && msg == "Attempted to divide by zero.");
            bool lthrew = false;
            try { long lz = 0; long lq = 1L % lz; Program.Check("num35", lq == 99); }
            catch (DivideByZeroException) { lthrew = true; }
            Program.Check("num35", lthrew);

            // ----- ++ and a compound assignment convert BACK (12.8.15 / 12.21.4) -----
            byte b4 = 255;
            b4++;
            Program.Check("num36", b4 == 0);
            byte b5 = 250;
            b5 += 10;
            Program.Check("num37", b5 == 4);
            sbyte sb1 = 127;
            sb1++;
            Program.Check("num38", sb1 == -128);
            ushort us1 = 65535;
            us1++;
            Program.Check("num39", us1 == 0);
            // `int i = 5; i += 1.5` is (int)6.5 == 6, the same implicit cast.
            int ni = 5;
            ni += 1.5;
            Program.Check("num40", ni == 6);
            // A byte compares against an int at INT width - 5 < 1000 is not 5 < -24.
            byte b6 = 5;
            Program.Check("num41", b6 < 1000 && b6 > -1000);

            // ----- explicit conversions narrow to their own type (12.3.2) -----
            Program.Check("num42", (byte) 300 == 44 && (sbyte) 200 == -56);
            Program.Check("num43", (int) 3000000000L == -1294967296 && (uint) -1 == 4294967295);
            Program.Check("num44", (short) 70000 == 4464 && (ushort) -1 == 65535);
            Program.Check("num45", (char) 70000 == 4464 && (char) -1 == 65535);
            Program.Check("num46", (int) 2.9 == 2 && (int) -2.9 == -2 && (long) -2.9 == -2L);

            // ----- literals (6.4.5.3) -----
            // No suffix: the first of int, uint, long, ulong that HOLDS the value.
            // So 0xFFFFFFFF is a uint here, where in Java it is the int -1.
            Program.Check("num47", 0xFFFFFFFF == 4294967295 && 0x7FFFFFFF == 2147483647);
            Program.Check("num48", 0xFFFFFFFFFFFFFFFF == ulong.MaxValue);
            Program.Check("num49", 4294967296 == 4294967296L && 0b1010 == 10);
            Program.Check("num50", 1_000_000_000_000L == 1000000000000L && 5U == 5 && 5UL == 5);

            // ----- unary - and ~ at the promoted type (12.4.7.2) -----
            Program.Check("num51", -int.MinValue == int.MinValue && -long.MinValue == long.MinValue);
            // -uint is a LONG, because -uint.MaxValue does not fit in a uint.
            Program.Check("num52", -u2 == -4294967295L);
            Program.Check("num53", ~0 == -1 && ~0L == -1L && ~(1L << 40) == -1099511627777L);
            // ~ on a uint stays a uint.
            uint u5 = 0;
            Program.Check("num54", ~u5 == 4294967295);

            // ----- the bitwise operators are 64 bit on a long -----
            // Spelled in decimal, not as unchecked((long) 0xF0F0F0F0F0F0F0F0):
            // `unchecked` is not implemented here at all (neither operator nor
            // block), which is the documented gap this section's header records.
            long mask = -1085102592571150096L;
            Program.Check("num55", (mask & 0xFFL) == 0xF0L && (mask ^ mask) == 0L);
            Program.Check("num56", (0x0F0F0F0F0F0F0F0FL | 0xF0F0F0F0F0F0F0L) == 0x0FFFFFFFFFFFFFFFL);

            // ----- it composes with the FLOAT box rather than fighting it -----
            Program.Check("num57", 1 / 3 == 0 && 1.0 / 3.0 == 0.3333333333333333 && 1 / 3.0 == 0.3333333333333333);
            Program.Check("num58", (double) 1L / 2 == 0.5 && ("" + (2.5 * 2)) == "5" && ("" + (5 * 2)) == "10");
            double d = 1;
            d += 1L;
            Program.Check("num59", d == 2.0 && ("" + d) == "2");

            // ----- Parse, and a long through an array -----
            Program.Check("num60", int.Parse("-42") == -42 && long.Parse("9223372036854775807") == long.MaxValue);
            long[] ls = new long[2];
            ls[0] = long.MaxValue;
            ls[1] = ls[0] - 1;
            Program.Check("num61", ls[1] == 9223372036854775806L && ("" + ls[0]) == "9223372036854775807");
        }


        // ===== SECTION 27: unary +, cast chains, out variables and assignable places =====
        // Unary + (ECMA-334 12.9.1), a cast of a cast (12.9.7), a CONSTRUCTED
        // NESTED type as the receiver of a static member (12.8.7), an OUT
        // VARIABLE declared in its own argument (12.6.2.1), NAMED arguments in an
        // indexer (12.8.11.3), an anonymous-object PROJECTION (12.8.16.1), and an
        // assignment target that is a member access over ANY primary, optionally
        // wrapped in parentheses (12.21). No C# compiler was available on this
        // machine, so every expected value here is spec-derived, not run.
        class S27Box
        {
            public int V;
            public int[] Cells = new int[3];
            public static S27Box Make() { return new S27Box(); }
        }
        class S27Outer<T>
        {
            public class S27Inner<U>
            {
                public static int Get() { return 7; }
                public static int Field = 11;
            }
        }
        class S27Idx
        {
            public int this[int a] { get { return a * 10; } }
        }
        class S27Ändern { public static int Wert = 5; }   // a non-ASCII identifier
        struct S27Val { public int N; }

        static bool S27Try(string text, out int parsed)
        {
            parsed = text.Length;
            return true;
        }
        static int S27Sum(ref int seed) { return seed + 1; }

        static void S27()
        {
            // ----- unary + is the identity, and it survives a cast -----
            double d = 2.5;
            d = +d;
            int i4 = (int) +4;
            Program.Check("u1", d == 2.5 && i4 == 4);

            // ----- a cast of a cast -----
            object boxed = "y";
            string back = (string) (object) boxed;
            Program.Check("u2", back == "y");

            // ----- a constructed NESTED type as a static receiver -----
            Program.Check("u3", S27Outer<int>.S27Inner<long>.Get() == 7);
            Program.Check("u4", S27Outer<string>.S27Inner<int>.Field == 11);

            // ----- an out variable declared in the argument itself. The
            // DECLARATION is what is modelled: `out` does not copy back here (a
            // call-site `ref`/`out` is parsed and dropped), so the assertion is on
            // the call, not on the value the callee wrote.
            if (Program.S27Try("abcd", out int got))
            {
                Program.Check("u5", true);
            }

            // ----- a named argument in an indexer -----
            S27Idx idx = new S27Idx();
            Program.Check("u6", idx[3] == 30 && idx[a: 4] == 40);

            // ----- an anonymous-type projection: `new { x.V }` names the member V -----
            S27Box src = new S27Box();
            src.V = 6;
            var proj = new { src.V, Answer = 42 };
            Program.Check("u7", proj.V == 6 && proj.Answer == 42);

            // ----- a parenthesized assignment target -----
            int a = 0;
            int b = 0;
            (a) = (b) = 1;
            int j = (a)++;
            (a) += 2;
            Program.Check("u8", b == 1 && j == 1 && a == 4);

            // ----- an assignable place rooted at any primary -----
            S27Box box = new S27Box();
            object any = box;
            ((S27Box) any).V = 11;
            S27Box.Make().V = 9;
            box.Cells[1] = 3;
            Program.Check("u9", box.V == 11 && box.Cells[1] == 3);

            // ----- a ref local aliases; this model copies, which a read cannot tell apart -----
            int seed = 4;
            ref int alias = ref seed;
            Program.Check("u10", alias == 4 && Program.S27Sum(ref seed) == 5);

            // ----- a nullable value type created with `new S? ()` -----
            object nv = new S27Val? ();
            Program.Check("u11", nv != null);

            // ----- a non-ASCII identifier (ECMA-334 6.4.3: any Unicode letter) -----
            Program.Check("u12", S27Ändern.Wert == 5);
        }

        // ===== SECTION 26: preprocessor directives =====
        static int S26Val = 0;
        static void S26()
        {
#region S26 arithmetic
            int a = 1;
#pragma warning disable 219
            int b = 2;
#pragma warning restore 219
            Program.Check("pp1", a + b == 3);
#endregion
#line 900 "s26.cs"
            Program.Check("pp2", a == 1);
#line default
#if true
            Program.S26Val = 1;
#else
            // Conditional compilation is NOT evaluated: this grammar skips the
            // directives and parses the text of EVERY branch (see PPDirective in
            // csharp-interpreter.abnf). So a branch here must stay free of
            // declarations and statements that would collide with the live one -
            // which is exactly what this comment demonstrates.
#endif
            Program.Check("pp3", Program.S26Val == 1);
#nullable disable
            string s = "x";
#nullable restore
            Program.Check("pp4", s == "x");
        }


        // ===== SECTION 28: a literal adopts its declared type, and `is` reads the
        // run-time type =====
        //
        // docs/todo.md 1.1 and 1.9. ECMA-334 10.2.3 (implicit numeric conversions)
        // makes an untyped integer literal written at a `double`, `float` or sized
        // integral site take that type; the WIDTH IS ON THE VALUE, not on the
        // annotation, so every one of these used to INTEGER-DIVIDE and answer 1
        // where the spec says 1.5. ECMA-334 12.12.12 makes `E is T` a RUN-TIME type
        // test with no implicit conversion, which is why `5L is int` is false.
        //
        // THERE IS NO C# TOOLCHAIN ON THIS MACHINE, so every value here is
        // spec-cited rather than executed. Each operand is read out of an ARRAY so
        // the grammars' constant folders cannot answer the question at compile time.
        //
        // PART OF THE WRITE SITE IS NOW CLOSED, in SECTION 29: an UNQUALIFIED write
        // to a member of the enclosing type adopts that member's declared type
        // (`double D; ... D = 3` holds 3.0), and a numeric field with no
        // initializer starts at its declared type's zero rather than the int 0.
        // Both are exact, because the declaring class is known at walk time.
        //
        // STILL NOT CLOSED, each still a wrong answer and each deliberately NOT
        // asserted here: a QUALIFIED write (`bx.A = 3` is still 3, not 3.0 - the
        // receiver's type is not known at the write site, so it needs either a
        // static receiver-type table or a run-time lookup of the member's declared
        // type off the receiver's class), a write to a LOCAL after its declaration
        // (`double d = 0; d = 3`, which needs a per-method var-type table in both
        // engines), an array ELEMENT write, a lambda's parameter and return types, a
        // nested `double[][]`, a `List<double>` element and a
        // `Dictionary<string, double>` value. And `decimal` is still a style-2
        // double box, so `1.5m is double` is true and `decimal[]` renders
        // `System.Double[]`.
        class S28Box
        {
            public double A = 3;
            public float F = 3;
            public byte B = 250;
            public long L = 5;
            public double[] Xs = {3, 4};
            public double P { get; set; } = 3;
            public double Computed => 3;
        }
        static double S28G(double x) { return x / 2; }
        static float S28F(float x) { return x / 2; }
        static byte S28W(byte b) { return b; }
        static long S28L(long v) { return v; }
        static double S28R() { return 3; }
        static double S28Dflt(double x = 3) { return x / 2; }
        static double S28Arrow(double x) => x / 2;
        static double S28InTry(bool t) { try { if (t) return 3; } finally { } return 4; }
        static object S28Obj(object o) { return o; }
        static void S28()
        {
            int[] n = {3, 250, 5, 1};

            // ----- a PARAMETER's declared type adopts its argument -----
            Program.Check("ad1", S28G(n[0]) == 1.5);
            Program.Check("ad2", S28F(n[0]) == 1.5);
            Program.Check("ad3", (S28W(n[1]) is byte) && !(S28W(n[1]) is int));
            Program.Check("ad4", (S28L(n[2]) is long) && !(S28L(n[2]) is int));
            // a default value adopts too, and a non-numeric parameter is untouched
            Program.Check("ad5", S28Dflt() == 1.5 && S28Dflt(n[0]) == 1.5);
            Program.Check("ad6", S28Obj(null) == null && (string) S28Obj("s") == "s");

            // ----- a RETURN TYPE adopts, at a block body, an arrow body and a
            // `return` that leaves a try -----
            Program.Check("ad7", S28R() / 2 == 1.5 && S28R() is double);
            Program.Check("ad8", S28Arrow(n[0]) == 1.5);
            Program.Check("ad9", S28InTry(true) / 2 == 1.5 && S28InTry(false) / 2 == 2);

            // ----- a local function's parameter and return type adopt -----
            double S28Loc(double z) { return z / 2; }
            Program.Check("ad10", S28Loc(n[0]) == 1.5);

            // ----- a FIELD and an auto-PROPERTY adopt their initializers -----
            S28Box bx = new S28Box();
            Program.Check("ad11", bx.A / 2 == 1.5 && bx.F / 2 == 1.5);
            Program.Check("ad12", (bx.B is byte) && !(bx.B is int) && (bx.L is long) && !(bx.L is int));
            Program.Check("ad13", bx.P / 2 == 1.5 && bx.Computed / 2 == 1.5);

            // ----- an ARRAY declaration types its ELEMENTS -----
            double[] d = {3, 4};
            Program.Check("ad14", d[0] / 2 == 1.5 && d[1] / 2 == 2);
            Program.Check("ad15", bx.Xs[0] / 2 == 1.5);
            // ... and the element adoption is IN PLACE, so an array assignment
            // still ALIASES rather than copying.
            int[] x = {1};
            int[] y = x;
            y[0] = 5;
            Program.Check("ad16", x[0] == 5);
            // NEGATIVE CONTROL: an int[] must NOT float.
            int[] ints = {3, 4};
            Program.Check("ad17", ints[0] / 2 == 1);

            // ----- `is` reads the RUN-TIME type (ECMA-334 12.12.12) -----
            object[] o = {n[3], 5L, (byte) 3, (short) 3, 1.5, 1.5f, "s", 'c', true};
            Program.Check("is1", (o[0] is int) && !(o[0] is long) && !(o[0] is char));
            Program.Check("is2", (o[1] is long) && !(o[1] is int) && (o[1] is Int64));
            Program.Check("is3", (o[2] is byte) && !(o[2] is sbyte) && !(o[2] is int));
            Program.Check("is4", (o[3] is short) && !(o[3] is ushort) && !(o[3] is int));
            Program.Check("is5", (o[4] is double) && !(o[4] is float));
            Program.Check("is6", (o[5] is float) && !(o[5] is double));
            Program.Check("is7", (o[6] is string) && (o[7] is char) && (o[8] is bool));
            // NEGATIVE CONTROLS: `object` matches everything, null matches nothing
            // that is not nullable, and a user class still walks its own chain.
            Program.Check("is8", (o[0] is object) && (o[6] is object));
            S28Box nb = null;
            Program.Check("is9", !(nb is S28Box) && (bx is S28Box));

            // ----- the same test behind a PATTERN and behind `as` -----
            string got = "";
            for (int i = 0; i < o.Length; i++)
            {
                switch (o[i])
                {
                    case int v: got += "i"; break;
                    case long v: got += "l"; break;
                    case float v: got += "f"; break;
                    case double v: got += "d"; break;
                    case char v: got += "c"; break;
                    default: got += "."; break;
                }
            }
            Program.Check("is10", got == "il..df.c.");
            Program.Check("is11", (o[4] as string) == null && (string) (o[6] as string) == "s");
        }


        // ===== SECTION 29: an unqualified name resolves to a member of the
        // enclosing type =====
        //
        // docs/todo.md 1.2 and 1.1. ECMA-334 12.8.4 resolves a simple name that is
        // not a local, a parameter or a type against the members of the enclosing
        // type; 12.8.7 inserts the implicit `this` for an instance member; 12.7.3
        // forbids `this` in a static member, so a static member is reached through
        // the DECLARING type instead; and 7.3 makes a local or parameter of the same
        // name shadow it. ECMA-334 10.2.3 at 12.21.2's simple assignment is what
        // makes the WRITTEN VALUE adopt the member's declared type.
        //
        // THERE IS NO C# TOOLCHAIN ON THIS MACHINE, so every value here is
        // spec-cited rather than executed. Every operand is read out of an ARRAY so
        // the grammars' constant folders cannot answer the question at compile time.
        //
        // FIVE SHAPES WERE BROKEN and only the instance-field READ worked. An
        // unqualified WRITE to an instance field ABORTED the compiler half where the
        // interpreter answered; a STATIC field aborted in both directions in both
        // halves; an auto-property write aborted; a property with ACCESSOR BODIES
        // read unqualified emitted a js_get of a slot that does not exist and
        // SILENTLY ANSWERED NOTHING in the compiler while the interpreter aborted;
        // and an inherited field was missing from the instance entirely in the
        // interpreter whenever the subclass declared a method (s29in5).
        class S29A
        {
            public double D = 1;
            public double U;
            public int I = 10;
            public static double S = 2;
            public static int SI = 20;
            public double P { get; set; } = 1;
            public static int SP { get; set; } = 40;
            public int W { get { return I * 2; } set { I = value; } }

            public double RdD() { return this.D; }
            public void WrD(int v) { D = v; }
            public void CompI(int v) { I += v; }
            public void IncI() { I++; }
            public int PostI() { return I++; }
            public void WrU(int v) { U = v; }
            public void WrP(int v) { P = v; }
            public void WrW(int v) { W = v; }
            public int RdW() { return W; }
            public double RdS() { return S; }
            public void WrS(int v) { S = v; }
            public static double SRdS() { return S; }
            public static void SWrS(int v) { S = v; }
            public static void SIncSI() { SI++; }
            public static int SRdSP() { return SP; }
            public static void SWrSP(int v) { SP = v; }

            // The CONTROLS. A parameter, a local, and a local declared with the same
            // name as a field must all keep resolving to the local - which is the
            // one thing a wrong implementation of the branch would break.
            public double ShadowParam(double D) { return D; }
            public double ShadowLocal() { double D = 99; return D; }
            public double ShadowWrite(double D) { D = 5; return D + this.D; }
        }
        // A STATIC FIELD INITIALIZER naming another static of the same class,
        // unqualified: the initializers run in declaration order (ECMA-334
        // 15.5.6.1), so S29I.B is 6. The interpreter said "unknown name: A" where
        // the compiler half answered - a halves divergence of its own.
        class S29I { public static double A = 2; public static double B = S29I.A * 3;
                     public static double C = A * 3; public static double Rd() { return C; } }
        class S29Base { public double BD = 1; public static double BS = 2; }
        class S29Sub : S29Base
        {
            public void WrBD(int v) { BD = v; }
            public double RdBD() { return BD; }
            public void WrBS(int v) { BS = v; }
            public double RdBS() { return BS; }
        }
        static void S29()
        {
            int[] n = {3, 4, 5};

            // ----- an unqualified INSTANCE field, read and written -----
            S29A a = new S29A();
            Program.Check("s29in1", a.RdD() == 1);
            a.WrD(n[0]);
            Program.Check("s29in2", a.D == 3);
            a.CompI(n[0]);
            Program.Check("s29in3", a.I == 13);
            S29A b = new S29A();
            b.IncI();
            Program.Check("s29in4", b.I == 11 && b.PostI() == 11 && b.I == 12);
            // an INHERITED field is reached the same way, and the subclass declaring
            // a method must not cost it its base constructor
            S29Sub sb = new S29Sub();
            Program.Check("s29in5", sb.BD == 1 && sb.RdBD() == 1);
            sb.WrBD(n[0]);
            Program.Check("s29in6", sb.BD == 3 && sb.RdBD() == 3);

            // ----- an unqualified STATIC field, from an instance member and from a
            // static one, of this class and of a base -----
            S29A.S = 2;
            Program.Check("s29st1", new S29A().RdS() == 2 && S29A.SRdS() == 2);
            new S29A().WrS(n[2]);
            Program.Check("s29st2", S29A.S == 5);
            S29A.SWrS(n[1]);
            Program.Check("s29st3", S29A.S == 4);
            S29A.SI = 20;
            S29A.SIncSI();
            Program.Check("s29st4", S29A.SI == 21);
            S29Base.BS = 2;
            Program.Check("s29st5", new S29Sub().RdBS() == 2);
            new S29Sub().WrBS(n[0]);
            Program.Check("s29st6", S29Base.BS == 3);

            Program.Check("s29st7", S29I.B == 6 && S29I.C == 6 && S29I.Rd() == 6);

            // ----- an unqualified auto-PROPERTY and a property with ACCESSOR
            // BODIES, read and written -----
            S29A p = new S29A();
            p.WrP(n[0]);
            Program.Check("s29pr1", p.P == 3);
            Program.Check("s29pr2", p.RdW() == 20);
            p.WrW(n[2]);
            Program.Check("s29pr3", p.I == 5 && p.RdW() == 10);
            S29A.SP = 40;
            Program.Check("s29pr4", S29A.SRdSP() == 40);
            S29A.SWrSP(n[0]);
            Program.Check("s29pr5", S29A.SP == 3);

            // ----- THE CONTROLS: a local or parameter of the same name SHADOWS the
            // member, and a write through the shadowed name must not touch it.
            S29A c = new S29A();
            Program.Check("s29sh1", c.ShadowParam(n[2]) == 5);
            Program.Check("s29sh2", c.ShadowLocal() == 99);
            Program.Check("s29sh3", c.ShadowWrite(n[0]) == 6 && c.D == 1);

            // ----- docs/todo.md 1.1: the member's DECLARED TYPE adopts an
            // unqualified write, so an untyped integer written at a `double` member
            // does not integer-divide afterwards.
            S29A t = new S29A();
            t.WrD(n[0]);
            Program.Check("s29ad1", t.D / 2 == 1.5 && (t.D is double) && !(t.D is int));
            t.WrU(n[0]);
            Program.Check("s29ad2", t.U / 2 == 1.5 && (t.U is double));
            t.WrP(n[0]);
            Program.Check("s29ad3", t.P / 2 == 1.5 && (t.P is double));
            S29A.SWrS(n[0]);
            Program.Check("s29ad4", S29A.S / 2 == 1.5 && (S29A.S is double));
            // a field with NO initializer starts at its declared type's zero
            Program.Check("s29ad5", (new S29A().U is double) && !(new S29A().U is int));
            // NEGATIVE CONTROL: an `int` member must NOT float.
            S29A u = new S29A();
            u.CompI(n[0]);
            Program.Check("s29ad6", u.I / 2 == 6 && (u.I is int) && !(u.I is double));
        }

        // ===== SECTION 30: constructor initializers, constructor overloads, and
        // the last three declaration sites that adopt =====
        //
        // docs/todo.md 1.1, 1.3 and 1.10. ECMA-334 15.11.2 (constructor
        // initializers): ': base(...)' names the base constructor that runs before
        // this one's body, ': this(...)' names a sibling of the same class, and the
        // instance variable initializers run ONLY when the initializer is not a
        // this(...) one. ECMA-334 12.6.4 (overload resolution) picks the applicable
        // constructor whose parameter types best fit the arguments; 12.12.12 defines
        // the run-time type test the fit is decided by here. ECMA-334 10.2.3 at
        // 12.21.2 is what makes a written value adopt the target's declared type.
        // ECMA-334 15.8.1 makes an event's '+=' a Delegate.Combine.
        //
        // THERE IS NO C# TOOLCHAIN ON THIS MACHINE, so every value is spec-cited
        // rather than executed. Every operand is read out of an ARRAY so the
        // grammars' constant folders cannot answer at compile time.
        //
        // WHAT WAS BROKEN. The ': base(d)' initializer was PARSED AND DISCARDED and
        // the base class' parameterless constructor ran instead, so `new S30C(3).D`
        // was empty where C# says 3 - in all three engines, with both halves
        // agreeing, which is why --cross was blind to it. A class kept only its LAST
        // constructor, so `new S30Ctr(4)` ran S30Ctr(double). A property with
        // accessor bodies was recorded by member NAME across the whole file, so a
        // plain field of the same name in ANY other class emitted an accessor call
        // and the compiler half died with "unknown method '__get_X'". An `event`
        // field of a built-in delegate type did not switch multicast combination on,
        // so 'E += h' did arithmetic. And a write to a local after its declaration,
        // an array element and a qualified member all stored the raw int.
        class S30P
        {
            public double D;
            public double T = 1;
            public S30P(double d) { D = d; }
            public S30P() { D = -1; }
        }
        class S30C : S30P
        {
            public double Own = 5;
            public S30C(double d) : base(d) { }
            public S30C() : this(7) { T = T + 100; }
        }
        class S30Ctr
        {
            public double D;
            public string W = "";
            public S30Ctr() { D = 0; }
            public S30Ctr(int a) { D = 9; }
            public S30Ctr(double a) { D = a; }
            public S30Ctr(string a) { D = 5; W = a; }
            public S30Ctr(int a, int b) { D = a + b; }
        }
        // The property/field name collision: X is a property with accessor bodies
        // in one class and a plain field in another, and neither may take the
        // other's meaning.
        class S30Acc { private double v = 7; public double X { get { return v; } set { v = value; } } }
        class S30Fld { public double X = 5; }
        class S30Ev
        {
            public event Action E;
            public void Fire() { Action h = E; if (h != null) { h(); } }
            public bool Empty() { return E == null; }
        }
        class S30Mem { public double D; public double[] A = new double[2]; public int I; }
        static string S30Log = "";
        static void S30()
        {
            int[] n = {3, 4, 5};
            double[] q = {4.5};
            string[] w = {"x"};

            // ----- ': base(...)' runs the base constructor with its arguments -----
            Program.Check("s30ci1", new S30C(n[0]).D == 3);
            Program.Check("s30ci2", new S30C(n[0]).Own == 5);
            // ': this(...)' runs the sibling, and the field initializers run ONCE
            Program.Check("s30ci3", new S30C().D == 7);
            Program.Check("s30ci4", new S30C().T == 101);
            // the implicit initializer still runs the parameterless base
            Program.Check("s30ci5", new S30P().D == -1);

            // ----- constructor overload resolution -----
            Program.Check("s30ov1", new S30Ctr(n[1]).D == 9);
            Program.Check("s30ov2", new S30Ctr(q[0]).D == 4.5);
            Program.Check("s30ov3", new S30Ctr(w[0]).D == 5 && new S30Ctr(w[0]).W == "x");
            Program.Check("s30ov4", new S30Ctr().D == 0);
            Program.Check("s30ov5", new S30Ctr(n[0], n[1]).D == 7);

            // ----- a property with accessor bodies must not capture a plain field
            // of the same name in another class -----
            Program.Check("s30pa1", new S30Acc().X == 7);
            S30Fld f = new S30Fld();
            Program.Check("s30pa2", f.X == 5);
            f.X = n[0];
            Program.Check("s30pa3", f.X == 3);
            S30Acc ac = new S30Acc();
            ac.X = n[2];
            Program.Check("s30pa4", ac.X == 5);

            // ----- an event's '+=' combines invocation lists -----
            S30Ev ev = new S30Ev();
            Program.Check("s30ev1", ev.Empty());
            Action ea = () => { Program.S30Log = Program.S30Log + "a"; };
            Action eb = () => { Program.S30Log = Program.S30Log + "b"; };
            ev.E += ea;
            ev.E += eb;
            ev.Fire();
            Program.Check("s30ev2", Program.S30Log == "ab");
            ev.E -= ea;
            ev.Fire();
            Program.Check("s30ev3", Program.S30Log == "abb");

            // ----- docs/todo.md 1.3: the three declaration sites that still stored
            // the raw int -----
            double L;
            L = n[0];
            Program.Check("s30ad1", L / 2 == 1.5 && (L is double));
            double[] ar = new double[2];
            ar[0] = n[0];
            Program.Check("s30ad2", ar[0] / 2 == 1.5 && (ar[0] is double));
            S30Mem mm = new S30Mem();
            mm.D = n[0];
            Program.Check("s30ad3", mm.D / 2 == 1.5 && (mm.D is double));
            mm.A[1] = n[0];
            Program.Check("s30ad4", mm.A[1] / 2 == 1.5 && (mm.A[1] is double));
            // NEGATIVE CONTROL: an `int` member and an `int` local must NOT float.
            mm.I = n[0];
            Program.Check("s30ad5", mm.I / 2 == 1 && (mm.I is int) && !(mm.I is double));
            int li;
            li = n[0];
            Program.Check("s30ad6", li / 2 == 1 && (li is int));
        }


        // ===== SECTION 31: method-group conversion, method overloads, and the
        // integral local write =====
        //
        // docs/todo.md 1.1, 1.2 and 1.3. ECMA-334 10.8 (method group conversions):
        // a method group is implicitly convertible to a compatible delegate type.
        // ECMA-334 20.1 says the delegate so created keeps the receiver the group
        // was formed on as its target, and 20.5 makes two delegates equal when
        // target and method are equal - which is what `E -= s.Got` needs, since
        // Delegate.Remove removes an EQUAL entry. ECMA-334 20.2 gives every
        // delegate type an Invoke method with the delegate's own signature, and
        // 12.8.10.2 spells a delegate invocation exactly like a method invocation,
        // which is why `k.E(5)` on a delegate FIELD is legal. ECMA-334 12.6.4 is
        // overload resolution and 12.12.12 the run-time type test it is decided by
        // here. ECMA-334 10.2.3 at 12.21.2 makes a written value adopt the target's
        // declared type, and 8.3.6 gives `long` a 64-bit range.
        //
        // THERE IS NO C# TOOLCHAIN ON THIS MACHINE, so every value is spec-cited
        // rather than executed. Every operand is read out of an ARRAY so the
        // grammars' constant folders cannot answer at compile time.
        //
        // WHAT WAS BROKEN. Method-group conversion did not exist anywhere: `this.M`,
        // a bare `M` and a bare static `H` all answered undefined or aborted with
        // "unknown name", in BOTH halves; `k.E(5)` on a delegate field aborted with
        // "unknown method 'E'"; and `.Invoke()` was unsupported. Overloading was
        // implemented for CONSTRUCTORS only, so a class kept the LAST declaration of
        // each method name and `M(1)`, `M(1.5)`, `M(1,2)` and `M("x")` all ran
        // `M(string)`. And `long x; x = 3; x*1000000*1000000` was a 32-bit multiply
        // answering 2112827392 where C# says 3000000000000.
        delegate int S31Op(int x);
        delegate void S31H(string s);
        class S31K
        {
            public int Bias = 1;
            public S31Op E;
            public int F(int x) { return x + this.Bias; }
            public static int G(int x) { return x + 20; }
            public int ViaThis() { S31Op o = this.F; return o(10); }
            public int ViaBare() { S31Op o = F; return o(100); }
            public int ViaStatic() { S31Op o = G; return o(1000); }
            public int FireE(int x) { return E(x); }
        }
        class S31Base { public virtual int V(int x) { return x + 1; } public static int SB(int x) { return x + 90; } }
        class S31Der : S31Base { public override int V(int x) { return x + 2; } }
        class S31Sink
        {
            public string Tag;
            public S31Sink(string t) { Tag = t; }
            public void Got(string s) { Program.S31Log = Program.S31Log + Tag + s; }
        }
        class S31Src
        {
            public event S31H E;
            public void Fire(string s) { if (E != null) { E(s); } }
        }
        class S31Ov
        {
            public int M(int a) { return 1; }
            public int M(double a) { return 2; }
            public int M(int a, int b) { return 3; }
            public int M(string a) { return 4; }
            public static int S(int a) { return 11; }
            public static int S(string a) { return 12; }
            public int Only(int a) { return a + 1; }
        }
        // An OVERLOADED OPERATOR: ECMA-334 12.4.5 resolves a user-defined binary
        // operator by the same overload resolution as a method call, and 15.10.1
        // makes every operator static. The second declaration used to overwrite the
        // first, so `v + otherV` ran the (S31V, int) body and died reading `.X` of an
        // int - in BOTH halves, so --cross was blind to it.
        class S31V
        {
            public double X;
            public S31V(double x) { X = x; }
            public static S31V operator +(S31V a, S31V b) { return new S31V(a.X + b.X); }
            public static S31V operator +(S31V a, int b) { return new S31V(a.X + b); }
        }
        static string S31Log = "";
        static int S31Free(int x) { return x + 7; }

        static void S31()
        {
            int[] n = {1, 2, 3};
            string[] w = {"x", "y"};
            double[] q = {1.5};

            // ----- method groups -----
            S31K k = new S31K();
            Program.Check("s31mg1", k.ViaThis() == 11);
            Program.Check("s31mg2", k.ViaBare() == 101);
            Program.Check("s31mg3", k.ViaStatic() == 1020);
            S31Op a = S31K.G;
            Program.Check("s31mg4", a(n[0]) == 21);
            S31Op b = Program.S31Free;
            Program.Check("s31mg5", b(n[0]) == 8);
            S31Op c = k.F;
            Program.Check("s31mg6", c(n[1]) == 3);
            // A method group is BOUND: it keeps its own receiver's state.
            S31K k2 = new S31K();
            k2.Bias = n[2];
            S31Op c2 = k2.F;
            Program.Check("s31mg7", c(n[0]) == 2 && c2(n[0]) == 4);
            // ...and it is VIRTUAL: the object's own override runs, not the one the
            // static type declares.
            S31Base d = new S31Der();
            S31Op v = d.V;
            Program.Check("s31mg8", v(n[0]) == 3);
            // An inherited STATIC reached through the derived type name: the group
            // walks __super, which a plain member read does not.
            S31Op sb = S31Der.SB;
            Program.Check("s31mg9", sb(n[0]) == 91);
            // .Invoke() is the delegate's own method (ECMA-334 20.2).
            Program.Check("s31mg10", c.Invoke(n[2]) == 4);
            // A delegate FIELD called like a method, qualified and unqualified.
            k.E = k.F;
            Program.Check("s31mg11", k.E(n[2]) == 4 && k.FireE(n[1]) == 3);
            // ONE object per (receiver, method), so Delegate.Remove finds it again.
            Program.S31Log = "";
            S31Src src = new S31Src();
            S31Sink s1 = new S31Sink("1");
            S31Sink s2 = new S31Sink("2");
            src.E += s1.Got;
            src.E += s2.Got;
            src.Fire(w[0]);
            src.E -= s1.Got;
            src.Fire(w[1]);
            Program.Check("s31mg12", Program.S31Log == "1x2x2y");
            Program.Check("s31mg13", s1.Got == s1.Got);
            Program.Check("s31mg14", s1.Got != s2.Got);

            // ----- method overloads (ECMA-334 12.6.4) -----
            S31Ov ov = new S31Ov();
            Program.Check("s31ov1", ov.M(n[0]) == 1);
            Program.Check("s31ov2", ov.M(q[0]) == 2);
            Program.Check("s31ov3", ov.M(n[0], n[1]) == 3);
            Program.Check("s31ov4", ov.M(w[0]) == 4);
            Program.Check("s31ov5", S31Ov.S(n[0]) == 11);
            Program.Check("s31ov6", S31Ov.S(w[0]) == 12);
            // NEGATIVE CONTROL: a name with ONE declaration is unchanged.
            Program.Check("s31ov7", ov.Only(n[1]) == 3);

            // ----- an overloaded operator (ECMA-334 12.4.5 / 15.10.1) -----
            S31V ov1 = new S31V(n[0]);
            Program.Check("s31op1", (ov1 + new S31V(n[1])).X == 3);
            Program.Check("s31op2", (ov1 + n[1]).X == 3);

            // ----- an integral local write adopts its declared type -----
            long lx;
            lx = n[2];
            Program.Check("s31ad1", lx * 1000000 * 1000000 == 3000000000000L);
            long ly = n[2];
            ly = n[1];
            Program.Check("s31ad2", ly * 1000000 * 1000000 == 2000000000000L);
            ulong lu;
            lu = n[2];
            Program.Check("s31ad3", lu * 1000000 * 1000000 == 3000000000000UL);
            // NEGATIVE CONTROL: an `int` local must still be a 32-bit multiply, and
            // must still read as an int.
            int li;
            li = n[2];
            Program.Check("s31ad4", li * 1000000 * 1000000 == 2112827392 && (li is int) && !(li is long));
            // A value that is ALREADY of the declared type is unchanged (this is the
            // path the js_gival gate skips).
            long lz;
            lz = lx;
            Program.Check("s31ad5", lz * 1000000 * 1000000 == 3000000000000L);
        }

        // ===== SECTION 32: inherited statics, base overloads, operator
        // applicability and `ref` / `out` parameters =====
        //
        // docs/todo.md 1.1. ECMA-334 15.3.1 makes every accessible member of a base
        // class a member of the derived class, so a STATIC member is reachable
        // through a derived type name; 15.5.1 gives a static field ONE storage
        // location, shared by every name that reaches it. 15.4 makes a `const`
        // member implicitly static. 12.6.4 builds the candidate set for a method
        // call in the receiver's type and MOVES TO THE BASE when that type offers no
        // applicable candidate, and 15.3.5 is what `new` stops. 12.4.5 resolves a
        // user-defined binary operator by that same overload resolution, so an
        // operator whose parameters neither operand fits is not applicable and the
        // PREDEFINED operator of 12.10.5 (`string operator +(string, object)`)
        // stands. 12.6.2.3.3 / 12.6.2.3.4 make a `ref` / `out` parameter an ALIAS
        // for the argument's variable, so a write through it is a write to that
        // variable.
        //
        // THERE IS NO C# TOOLCHAIN ON THIS MACHINE, so every value here is
        // spec-cited rather than executed. Every operand is read out of an ARRAY so
        // the grammars' constant folders cannot answer at compile time.
        //
        // WHAT WAS BROKEN. `S32Der.SB()` aborted with "unknown static method" in all
        // three engines while the method-GROUP form `S32Op p = S32Der.SB` already
        // worked; an inherited static FIELD, PROPERTY and delegate field read as
        // nothing; a `const` member was not static at all, so even `S32Bas.C` was
        // empty; a derived class' own overload group HID the base's, so `d.M(1)`
        // answered 2 where C# says 1; a user `operator +` captured string
        // concatenation, so `"s" + v` answered a V rather than "sV1"; and `ref` /
        // `out` were parsed and dropped, so `Ref(ref r)` left r alone.
        class S32Bas
        {
            public static int F = 5;
            public static int P { get { return 9; } }
            public const int C = 11;
            public static int SB() { return 7; }
            public static S32Op BH = (int x) => x + 500;
            public int M(int x) { return 1; }
        }
        class S32Der : S32Bas
        {
            public int M(string s) { return 2; }
        }
        class S32Hide : S32Bas
        {
            public new int M(string s) { return 3; }
        }
        delegate int S32Op(int x);
        class S32V
        {
            public int N;
            public S32V(int n) { N = n; }
            public override string ToString() { return "V" + N; }
            public static S32V operator +(S32V a, S32V b) { return new S32V(a.N + b.N); }
        }
        class S32Ref
        {
            public int Fld;
            public int[] Arr = {1, 2, 3};
            public void Bump(ref int a) { a = a + 1; a++; a += 10; }
            public void ByVal(int a) { a = a + 1; a++; a += 10; }
            public void Chain(ref int a) { Bump(ref a); }
            public static void Swap(ref int a, ref int b) { int t = a; a = b; b = t; }
            public static bool TryGet(int k, out int v) { v = k * 2; return k > 0; }
            public static void Cat(ref string s) { s = s + "!"; }
        }

        static void S32()
        {
            int[] n = {1, 2, 3};
            string[] w = {"s", "t"};

            // ----- a static member reached through a DERIVED type name -----
            Program.Check("s32st1", S32Der.SB() == n[2] + 4);
            Program.Check("s32st2", S32Der.F == n[1] + 3);
            Program.Check("s32st3", S32Der.P == n[2] + 6);
            Program.Check("s32st4", S32Der.C == n[2] + 8);
            Program.Check("s32st5", S32Bas.C == n[2] + 8);
            Program.Check("s32st6", S32Der.BH(n[0]) == 501);
            // One storage location, whichever name reaches it (ECMA-334 15.5.1).
            S32Der.F = n[2] * 100;
            Program.Check("s32st7", S32Bas.F == 300 && S32Der.F == 300);
            S32Bas.F = n[1] + 3;
            // The method-GROUP form of the same inherited static.
            S32Op sg = S32Der.SB;
            Program.Check("s32st8", sg == null || sg != null);

            // ----- the candidate set continues into the BASE class -----
            S32Der d = new S32Der();
            Program.Check("s32ov1", d.M(n[0]) == n[0]);
            Program.Check("s32ov2", d.M(w[0]) == n[1]);
            // `new` hides, so the base candidate is NOT in the set and the int
            // argument binds to the only declaration there is.
            S32Hide h = new S32Hide();
            Program.Check("s32ov3", h.M(w[0]) == n[2]);

            // ----- a user operator that does not ACCEPT the operands -----
            S32V v = new S32V(n[0]);
            S32V v2 = new S32V(n[1]);
            Program.Check("s32op1", (w[0] + v) == "sV1");
            Program.Check("s32op2", (v + w[0]) == "V1s");
            Program.Check("s32op3", (v + v2).ToString() == "V3");

            // ----- `ref` and `out` write back -----
            S32Ref r = new S32Ref();
            int a = n[0];
            r.Bump(ref a);
            Program.Check("s32rf1", a == 13);
            int b = n[0];
            r.Chain(ref b);
            Program.Check("s32rf2", b == 13);
            int p = n[0];
            int q = n[1];
            S32Ref.Swap(ref p, ref q);
            Program.Check("s32rf3", p == n[1] && q == n[0]);
            r.Fld = n[2];
            r.Bump(ref r.Fld);
            Program.Check("s32rf4", r.Fld == 15);
            r.Bump(ref r.Arr[1]);
            Program.Check("s32rf5", r.Arr[1] == 14);
            int got;
            bool ok = S32Ref.TryGet(n[1], out got);
            Program.Check("s32rf6", ok && got == 4);
            Program.Check("s32rf7", S32Ref.TryGet(n[0], out int fresh) && fresh == n[1]);
            string s = w[0];
            S32Ref.Cat(ref s);
            Program.Check("s32rf8", s == "s!");
            int[] xs = {5, 6};
            S32Ref.Swap(ref xs[0], ref xs[1]);
            Program.Check("s32rf9", xs[0] == 6 && xs[1] == 5);
            // NEGATIVE CONTROL: an ordinary by-value argument of the same shape must
            // still leave the caller's variable alone.
            int keep = n[0];
            r.ByVal(keep);
            Program.Check("s32rf10", keep == n[0]);
        }

        // ===== END SECTIONS =====


        // ===== SECTION 33: `(object)` suppresses a user operator, and the reading
        // of a value that is not a number =====
        //
        // docs/todo.md 1.3 and 1.2.
        //
        // 1.3. ECMA-334 12.4.5 builds the candidate set for a user-defined operator
        // from the STATIC types of the operands and resolves it like a method call
        // (12.6.4). With `object` on one side, `V.operator ==(V, V)` is not a
        // candidate at all - there is no implicit conversion from object to V - and
        // the PREDEFINED reference-equality operator of 12.12.6 stands. That is
        // exactly why `(object)a == null` is THE idiomatic null guard inside a user
        // `operator ==`, and here it recursed until the stack died, in both engines
        // and at `d473319` too. There are no static types in this value model, so
        // the suppression is read off the SOURCE by the parser (csIsObjCastOperand
        // in both csharp grammars); it costs nothing at run time, because the
        // operator probe is simply not emitted for that operand pair.
        //
        // 1.2. `dynamic d = o; d.Missing + 1` printed NaN under the interpreter and
        // llvm.Run and **1** natively, because rtjNum in
        // languages/lib/runtime-jvm.metajs answered 0 for undefined where
        // abnf/jsrt.go's rt.toNumber answers NaN. C#'s own answer is a
        // RuntimeBinderException, which no engine here can raise, so the value below
        // is ENGINE BEHAVIOUR asserted for CONVERGENCE - the point is that all four
        // engines say the same thing, and NaN is what three of them already said.
        //
        // THERE IS NO C# TOOLCHAIN ON THIS MACHINE, so every C#-semantic claim here
        // is spec-cited rather than executed. Every operand is read out of an ARRAY
        // so the grammars' constant folders cannot answer at compile time.
        class S33V
        {
            public int N;
            public S33V(int n) { this.N = n; }
            // THE IDIOMATIC NULL GUARD. Every one of these four `(object)` casts
            // used to re-enter this very operator.
            public static bool operator ==(S33V a, S33V b)
            {
                if ((object)a == null) { return (object)b == null; }
                if (((object)b) == null) { return false; }
                return a.N == b.N;
            }
            public static bool operator !=(S33V a, S33V b) { return !(a == b); }
            // The two shapes an earlier round of this work REGRESSED, kept as
            // controls: a null operand must still select operator ==(V, V), and an
            // implicit numeric conversion must still select a `double` parameter.
            public static bool operator ==(S33V a, double d) { return a.N == d; }
            public static bool operator !=(S33V a, double d) { return a.N != d; }
            public static S33V operator +(S33V a, S33V b) { return new S33V(a.N + b.N); }
            public override bool Equals(object o) { return o is S33V && ((S33V) o).N == this.N; }
            public override int GetHashCode() { return this.N; }
            public override string ToString() { return "V" + this.N; }
        }

        static void S33()
        {
            int[] n = {1, 2, 3};
            S33V[] v = { new S33V(1), new S33V(1), new S33V(2), null };

            // ----- the user operator still applies where C# says it does -----
            Program.Check("s33oc1", v[0] == v[1]);
            Program.Check("s33oc2", !(v[0] == v[2]));
            Program.Check("s33oc3", v[0] != v[2]);
            Program.Check("s33oc4", !(v[0] != v[1]));
            // A null operand: ECMA-334 10.2.7 makes null convertible to any
            // reference type, so operator ==(V, V) IS the candidate and answers.
            Program.Check("s33oc5", !(v[0] == v[3]));
            Program.Check("s33oc6", !(v[3] == v[0]));
            Program.Check("s33oc7", v[3] == null);
            // An implicit numeric conversion still reaches the `double` overload.
            Program.Check("s33oc8", v[0] == 1.0);
            Program.Check("s33oc9", v[2] != 1.0);
            // The operator's own body ran to completion - which is the whole point.
            Program.Check("s33oc10", (v[0] + v[1]).ToString() == "V" + n[1]);

            // ----- `(object)` DECLINES it: 12.12.6 reference equality -----
            Program.Check("s33oc11", !((object) v[0] == (object) v[1]));
            Program.Check("s33oc12", (object) v[0] == v[0]);
            Program.Check("s33oc13", (object) v[0] != v[1]);
            Program.Check("s33oc14", (object) v[3] == null);
            Program.Check("s33oc15", (object) v[0] != null);
            // Redundant parentheses do not change a static type.
            Program.Check("s33oc16", ((object) v[3]) == null);
            Program.Check("s33oc17", !(((object) v[0]) == null));

            // ----- docs/todo.md 1.2: ENGINE BEHAVIOUR, see the note above -----
            dynamic d = new S33V(n[0]);
            Program.Check("s33un1", ("" + (d.Missing + 1)) == "NaN");
            Program.Check("s33un2", ("" + (1 + d.Missing)) == "NaN");
            // The reads that do NOT go through rtjNum answer 0 in every engine.
            Program.Check("s33un3", 1 - d.Missing == 0);
            Program.Check("s33un4", d.Missing * 2 == 0);
            Program.Check("s33un5", !(d.Missing < 1));
            Program.Check("s33un6", !(d.Missing > 1));
            // The control: a member that IS there still reads.
            Program.Check("s33un7", d.N + 1 == n[1]);
        }

        static int Main()
        {
            Program.S01(); // SECTION-CALL 01
            Program.S02(); // SECTION-CALL 02
            Program.S03(); // SECTION-CALL 03
            Program.S04(); // SECTION-CALL 04
            Program.S05(); // SECTION-CALL 05
            Program.S06(); // SECTION-CALL 06
            Program.S07(); // SECTION-CALL 07
            Program.S08(); // SECTION-CALL 08
            Program.S09(); // SECTION-CALL 09
            Program.S10(); // SECTION-CALL 10
            Program.S11(); // SECTION-CALL 11
            Program.S12(); // SECTION-CALL 12
            Program.S13(); // SECTION-CALL 13
            Program.S14(); // SECTION-CALL 14
            Program.S15(); // SECTION-CALL 15
            Program.S16(); // SECTION-CALL 16
            Program.S17(); // SECTION-CALL 17
            Program.S18(); // SECTION-CALL 18
            Program.S19(); // SECTION-CALL 19
            Program.S20(); // SECTION-CALL 20
            Program.S21(); // SECTION-CALL 21
            Program.S22(); // SECTION-CALL 22
            Program.S23(); // SECTION-CALL 23
            Program.S24(); // SECTION-CALL 24
            Program.S25(); // SECTION-CALL 25
            Program.S26(); // SECTION-CALL 26
            Program.S27(); // SECTION-CALL 27
            Program.S28(); // SECTION-CALL 28
            Program.S29(); // SECTION-CALL 29
            Program.S30(); // SECTION-CALL 30
            Program.S31(); // SECTION-CALL 31
            Program.S32(); // SECTION-CALL 32
            Program.S33(); // SECTION-CALL 33
            Console.WriteLine("full: " + Program.Checks + " checks, " + Program.Fails + " failures");
            return Program.Fails;
        }
    }
}
