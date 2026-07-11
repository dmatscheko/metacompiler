// C# completion self test.
// Exercises the newly implemented surface: lambdas ('x => e', '(a, b) => { ... }') and
// Func / Action delegate VALUES as real closures, delegate call sites, closures that
// capture (and observe mutations of) outer locals, methods that take and return
// delegates, and object / collection initializers. Program.Main counts failed checks and
// returns that count, so the metacompiler run exits with 0 exactly when everything works.

using System;
using System.Collections.Generic;

namespace Demo
{
    // A class with the default (parameterless) constructor, for object initializers.
    class Box
    {
        public int W;
        public int H;

        public int Area()
        {
            return this.W * this.H;
        }
    }

    // A class with an explicit constructor plus a settable field, for the
    // 'new T(args){ Field = ... }' form (constructor arguments then member initializer).
    class Labeled
    {
        public int Id;
        public int Extra = 1;

        public Labeled(int id)
        {
            this.Id = id;
        }
    }

    class Program
    {
        static int Fails = 0;

        static void Check(string name, int got, int want)
        {
            if (got != want)
            {
                Console.WriteLine($"FAIL {name}: got {got} want {want}");
                Program.Fails++;
            }
        }

        // Takes a delegate and invokes it (a delegate call inside a method).
        static int Apply(Func<int, int> f, int v)
        {
            return f(v);
        }

        // Returns a closure that captures the parameter n and outlives the call.
        static Func<int, int> Adder(int n)
        {
            return x => x + n;
        }

        static int Main()
        {
            // an expression-bodied lambda as a Func value, invoked at a call site
            Func<int, int> sq = x => x * x;
            Program.Check("lambda expr", sq(6), 36);

            // a multi-parameter, block-bodied lambda with an explicit return
            Func<int, int, int> add = (a, b) => { return a + b; };
            Program.Check("lambda block", add(3, 4), 7);
            Program.Check("lambda nested call", add(add(1, 2), add(3, 4)), 10);

            // a closure capturing an outer local; the capture is by reference, so a later
            // mutation of the local is observed by the delegate
            int k = 10;
            Func<int, int> addk = x => x + k;
            Program.Check("capture read", addk(5), 15);
            k = 100;
            Program.Check("capture mutation", addk(5), 105);

            // an Action (a void lambda) mutating captured state through a call
            int[] acc = new int[1];
            Action<int> bump = x => { acc[0] += x; };
            bump(5);
            bump(7);
            Program.Check("action side effect", acc[0], 12);

            // passing a lambda to a method that invokes it
            Program.Check("higher order arg", Program.Apply(x => x + 1, 41), 42);
            Program.Check("higher order inline", Program.Apply(y => y * 3, 7), 21);

            // a method returning a closure; two closures capture independent state
            Func<int, int> add10 = Program.Adder(10);
            Func<int, int> add100 = Program.Adder(100);
            Program.Check("returned closure a", add10(5), 15);
            Program.Check("returned closure b", add100(5), 105);
            Program.Check("closures independent", add10(1) + add100(1), 112);

            // delegates stored in a List and dispatched in a loop
            List<Func<int, int>> ops = new List<Func<int, int>>();
            ops.Add(x => x + 1);
            ops.Add(x => x * 2);
            ops.Add(x => x - 3);
            int piped = 10;
            foreach (var op in ops)
            {
                piped = op(piped);
            }
            Program.Check("delegate list", piped, 19);   // ((10 + 1) * 2) - 3

            // object initializer on a parameterless constructor
            Box b1 = new Box { W = 3, H = 4 };
            Program.Check("object init area", b1.Area(), 12);
            Box b2 = new Box { W = 5, H = 6 };
            Program.Check("object init field", b2.W, 5);
            Box b3 = new Box();
            Program.Check("object init default", b3.Area(), 0);

            // object initializer combined with constructor arguments
            Labeled lab = new Labeled(7) { Extra = 9 };
            Program.Check("ctor plus init id", lab.Id, 7);
            Program.Check("ctor plus init extra", lab.Extra, 9);
            Labeled plain = new Labeled(3);
            Program.Check("ctor no init", plain.Extra, 1);

            // collection initializer for List<int>
            List<int> xs = new List<int> { 2, 4, 6, 8 };
            Program.Check("collection count", xs.Count, 4);
            Program.Check("collection index", xs[2], 6);
            int csum = 0;
            foreach (var v in xs)
            {
                csum += v;
            }
            Program.Check("collection sum", csum, 20);

            // a lambda invoked immediately (a closure call directly on the value)
            Program.Check("immediate lambda", Program.Apply(z => z, 55), 55);

            if (Program.Fails == 0)
            {
                Console.WriteLine("C# completion self test passed");
            }
            return Program.Fails;
        }
    }
}
