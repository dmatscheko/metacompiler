// C# subset self test.
// Exercises classes, inheritance, statics, arrays, List, Dictionary, strings,
// interpolation and control flow. Program.Main counts failed checks and returns
// that count, so the metacompiler run exits with 0 exactly when everything works.

using System;
using System.Collections.Generic;

namespace Demo
{
    class Counter
    {
        public int Value;
        public int Step = 1;

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

        public static int Twice(int x) => x * 2;
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

        public int Manhattan()
        {
            int ax = this.X < 0 ? -this.X : this.X;
            int ay = this.Y < 0 ? -this.Y : this.Y;
            return ax + ay;
        }

        public Point Plus(Point other)
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

        public string Describe()          // this.Name() dispatches dynamically
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
            this.Legs = 2;                 // the implicit base() ran the field inits first
        }

        public override string Name()
        {
            return "bird";
        }

        public override int Base()        // base.M() starts above the defining class
        {
            return base.Base() + 5;
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

        static void CheckB(string name, bool got, bool want)
        {
            Program.Check(name, got ? 1 : 0, want ? 1 : 0);
        }

        static void CheckS(string name, string got, string want)
        {
            if (got != want)
            {
                Console.WriteLine($"FAIL {name}: got {got} want {want}");
                Program.Fails++;
            }
        }

        static int Fib(int n)
        {
            if (n < 2)
            {
                return n;
            }
            return Program.Fib(n - 1) + Program.Fib(n - 2);
        }

        static int Classify(int x)        // switch with stacked labels and default
        {
            int r = 0;
            switch (x)
            {
                case 0:
                    r = 100;
                    break;
                case 1:
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

        static string DayKind(string day)
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

        static int Main()
        {
            // arithmetic and precedence
            Program.Check("precedence", 2 + 3 * 4, 14);
            Program.Check("parens", (2 + 3) * 4, 20);
            Program.Check("int div", 17 / 5, 3);
            Program.Check("modulo", 17 % 5, 2);
            Program.Check("left assoc", 10 - 2 - 3, 5);
            Program.Check("unary minus", -7 + 2, -5);
            Program.Check("neg div trunc", -7 / 2, -3);

            // comparisons and boolean logic
            Program.CheckB("greater", 5 > 3, true);
            Program.CheckB("ge equal", 3 >= 3, true);
            Program.CheckB("and", 1 < 2 && 3 < 4, true);
            Program.CheckB("or", 1 > 2 || 2 < 3, true);
            Program.CheckB("not", !(2 == 3), true);
            Program.Check("ternary", 5 > 3 ? 10 : 20, 10);

            // strings
            Program.CheckS("concat", "ab" + "cd", "abcd");
            Program.Check("str length", "hello".Length, 5);
            Program.Check("index of", "hello".IndexOf("ll"), 2);
            Program.CheckS("substring", "hello".Substring(3), "lo");
            Program.CheckB("str equals", "abc" == "abc", true);
            Program.CheckB("str neq", "abc" == "abd", false);
            int a = 3;
            int b = 4;
            int c = a + b;
            Program.CheckS("interpolation", $"{a}+{b}={c}", "3+4=7");

            // arrays
            int[] arr = new int[5];
            Program.Check("array default", arr[0], 0);
            for (int i = 0; i < arr.Length; i++)
            {
                arr[i] = i * i;
            }
            Program.Check("array store", arr[4], 16);
            arr[2] += 10;
            Program.Check("compound elem", arr[2], 14);

            int[] lit = new int[] { 3, 1, 4, 1, 5 };
            Program.Check("array literal", lit.Length, 5);
            int esum = 0;
            foreach (var v in lit)
            {
                esum += v;
            }
            Program.Check("foreach array", esum, 14);

            // List<int>
            List<int> nums = new List<int>();
            nums.Add(10);
            nums.Add(20);
            nums.Add(30);
            Program.Check("list count", nums.Count, 3);
            Program.Check("list index", nums[1], 20);
            Program.CheckB("list contains", nums.Contains(20), true);
            Program.CheckB("list missing", nums.Contains(99), false);
            int lsum = 0;
            foreach (var n in nums)
            {
                lsum += n;
            }
            Program.Check("foreach list", lsum, 60);

            // Dictionary<string, int>
            Dictionary<string, int> ages = new Dictionary<string, int>();
            ages["alice"] = 30;
            ages["bob"] = 25;
            Program.Check("dict get", ages["alice"], 30);
            Program.Check("dict count", ages.Count, 2);
            Program.CheckB("dict haskey", ages.ContainsKey("bob"), true);
            Program.CheckB("dict nokey", ages.ContainsKey("carol"), false);
            ages["alice"] += 1;
            Program.Check("dict compound", ages["alice"], 31);
            int vsum = 0;
            foreach (var val in ages.Values)
            {
                vsum += val;
            }
            Program.Check("dict values", vsum, 56);
            int klen = 0;
            foreach (var key in ages.Keys)
            {
                klen += key.Length;
            }
            Program.Check("dict keys", klen, 8);

            // control flow
            int wsum = 0;
            int w = 1;
            while (w <= 5)
            {
                wsum += w;
                w++;
            }
            Program.Check("while", wsum, 15);

            int dc = 0;
            do
            {
                dc++;
            } while (false);
            Program.Check("do while", dc, 1);

            int odd = 0;
            for (int j = 0; j < 100; j++)
            {
                if (j % 2 == 0)
                {
                    continue;
                }
                if (j > 10)
                {
                    break;
                }
                odd += j;
            }
            Program.Check("break continue", odd, 25);

            // switch
            Program.Check("switch first", Program.Classify(0), 100);
            Program.Check("switch stacked", Program.Classify(2), 12);
            Program.Check("switch case", Program.Classify(3), 3);
            Program.Check("switch default", Program.Classify(9), -1);
            Program.CheckS("string switch", Program.DayKind("sun"), "weekend");
            Program.CheckS("string switch default", Program.DayKind("tue"), "workday");

            // objects
            Counter counter = new Counter(10);
            Program.Check("field init", counter.Step, 1);
            Program.Check("method", counter.Next(), 11);
            counter.SetStep(5);
            Program.Check("setter", counter.Next(), 16);
            Program.Check("field read", counter.Value, 16);
            Program.Check("static method", Counter.Twice(21), 42);

            Point p = new Point(3, -4);
            Program.Check("manhattan", p.Manhattan(), 7);
            Point q = p.Plus(new Point(1, 1));
            Program.Check("returned object", q.X * 100 + q.Y, 397);
            Program.CheckB("value equality", q.SamePoint(new Point(4, -3)), true);
            Program.CheckB("identity same", p == p, true);
            Program.CheckB("identity twins", p == new Point(3, -4), false);

            Point none = null;
            Program.CheckB("null check", none == null, true);

            // inheritance
            Animal an = new Animal();
            Program.CheckS("describe", an.Describe(), "animal:4");
            Bird bird = new Bird();
            Program.CheckS("override dispatch", bird.Describe(), "bird:2");
            Program.Check("base call", bird.Base(), 15);
            Program.Check("inherited field", bird.Legs, 2);
            Animal upcast = bird;
            Program.CheckS("dispatch via base type", upcast.Name(), "bird");

            // recursion
            Program.Check("fib", Program.Fib(10), 55);

            if (Program.Fails == 0)
            {
                Console.WriteLine("C# subset self test passed");
            }
            return Program.Fails;
        }
    }
}
