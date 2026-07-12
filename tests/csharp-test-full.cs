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
// preprocessor directives, assembly-level attributes, reflection, threads.
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
        }

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
        static T S16Pick<T>(T a, T b, bool first) where T : class { return first ? a : b; }
        static void S16()
        {
            S16Seq<object> seq = new S16One<string>("hi");
            Program.Check("gen1", (string)seq.Head() == "hi");
            S16Sink<string> sink = new S16AnyLen();   // contravariant assignment
            Program.Check("gen2", sink.Eat("abcd") == 9);
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
        static async void S19Fire() { Program.S19Ran = true; }
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

        // ===== END SECTIONS =====

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
            Console.WriteLine("full: " + Program.Checks + " checks, " + Program.Fails + " failures");
            return Program.Fails;
        }
    }
}
