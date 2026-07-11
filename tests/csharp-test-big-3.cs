// C# subset self test -- BIG 3: object orientation and functional programming.
//
// Theme: the signature C# features. A polymorphic Shape hierarchy with virtual methods
// and dynamic dispatch, a three-level inheritance chain exercising base.M() calls, a
// class-based expression AST (Num / Neg / BinOp) evaluated by virtual Eval(), and a small
// functional toolkit (map / filter / reduce / compose) built on Func / Action closures,
// including closures that capture and mutate outer state. Program.Main returns the number
// of failed checks, so a clean run exits 0.

using System;
using System.Collections.Generic;

namespace Demo
{
    // ----- a polymorphic shape hierarchy -----

    class Shape
    {
        public virtual int Area()
        {
            return 0;
        }

        public virtual string Kind()
        {
            return "shape";
        }

        // Non-virtual, but Kind() and Area() dispatch dynamically to the runtime type.
        public string Tag()
        {
            return this.Kind() + ":" + this.Area();
        }
    }

    class Rectangle : Shape
    {
        public int W;
        public int H;

        public Rectangle(int w, int h)
        {
            this.W = w;
            this.H = h;
        }

        public override int Area()
        {
            return this.W * this.H;
        }

        public override string Kind()
        {
            return "rect";
        }
    }

    class Square : Shape
    {
        public int S;

        public Square(int s)
        {
            this.S = s;
        }

        public override int Area()
        {
            return this.S * this.S;
        }

        public override string Kind()
        {
            return "square";
        }
    }

    class Triangle : Shape
    {
        public int Base;
        public int Height;

        public Triangle(int b, int h)
        {
            this.Base = b;
            this.Height = h;
        }

        public override int Area()
        {
            return this.Base * this.Height / 2;
        }

        public override string Kind()
        {
            return "tri";
        }
    }

    // ----- a three-level inheritance chain using base.M() -----

    class BaseCell
    {
        public int Seed = 10;

        public virtual int Value()
        {
            return this.Seed;
        }
    }

    class MidCell : BaseCell
    {
        public override int Value()
        {
            return base.Value() * 2 + 1;
        }
    }

    class LeafCell : MidCell
    {
        public override int Value()
        {
            return base.Value() + 100;
        }
    }

    // ----- a class-based expression AST -----

    class Expr
    {
        public virtual int Eval()
        {
            return 0;
        }
    }

    class Num : Expr
    {
        public int V;

        public Num(int v)
        {
            this.V = v;
        }

        public override int Eval()
        {
            return this.V;
        }
    }

    class Neg : Expr
    {
        public Expr Inner;

        public Neg(Expr e)
        {
            this.Inner = e;
        }

        public override int Eval()
        {
            return -this.Inner.Eval();
        }
    }

    class BinOp : Expr
    {
        public string Op;
        public Expr L;
        public Expr R;

        public BinOp(string op, Expr l, Expr r)
        {
            this.Op = op;
            this.L = l;
            this.R = r;
        }

        public override int Eval()
        {
            int a = this.L.Eval();
            int b = this.R.Eval();
            switch (this.Op)
            {
                case "+":
                    return a + b;
                case "-":
                    return a - b;
                case "*":
                    return a * b;
                case "/":
                    return a / b;
                default:
                    return 0;
            }
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

        // ----- functional toolkit over List<int> and Func / Action delegates -----

        static List<int> Map(List<int> xs, Func<int, int> f)
        {
            List<int> r = new List<int>();
            foreach (var x in xs)
            {
                r.Add(f(x));
            }
            return r;
        }

        static List<int> Filter(List<int> xs, Func<int, int> pred)
        {
            List<int> r = new List<int>();
            foreach (var x in xs)
            {
                if (pred(x) != 0)
                {
                    r.Add(x);
                }
            }
            return r;
        }

        static int Reduce(List<int> xs, Func<int, int, int> f, int init)
        {
            int acc = init;
            foreach (var x in xs)
            {
                acc = f(acc, x);
            }
            return acc;
        }

        static Func<int, int> Compose(Func<int, int> f, Func<int, int> g)
        {
            return x => f(g(x));
        }

        static Func<int, int> Adder(int n)
        {
            return x => x + n;
        }

        static int Main()
        {
            // polymorphism: a mixed list dispatched through virtual Area()
            List<Shape> shapes = new List<Shape>();
            shapes.Add(new Rectangle(3, 4));
            shapes.Add(new Square(5));
            shapes.Add(new Triangle(6, 4));
            int totalArea = 0;
            foreach (var sh in shapes)
            {
                totalArea = totalArea + sh.Area();
            }
            Program.Check("polymorphic area sum", totalArea, 49);   // 12 + 25 + 12
            Program.Check("rect area", shapes[0].Area(), 12);
            Program.Check("square area", shapes[1].Area(), 25);
            Program.Check("tri area", shapes[2].Area(), 12);
            Program.CheckS("square kind", shapes[1].Kind(), "square");
            Program.CheckS("rect tag", shapes[0].Tag(), "rect:12");
            Program.CheckS("tri tag", shapes[2].Tag(), "tri:12");

            // Shape reference to a subclass still dispatches dynamically
            Shape s = new Square(7);
            Program.Check("upcast area", s.Area(), 49);
            Program.CheckS("upcast kind", s.Kind(), "square");

            // three-level inheritance with base.M()
            BaseCell bc = new BaseCell();
            MidCell mc = new MidCell();
            LeafCell lc = new LeafCell();
            Program.Check("base value", bc.Value(), 10);
            Program.Check("mid value", mc.Value(), 21);      // 10*2+1
            Program.Check("leaf value", lc.Value(), 121);    // (10*2+1) + 100
            Program.Check("inherited seed", lc.Seed, 10);

            // expression AST: ((2 + 3) * (7 - 4)) - (-5) = 15 + 5 = 20
            Expr e = new BinOp("-",
                new BinOp("*",
                    new BinOp("+", new Num(2), new Num(3)),
                    new BinOp("-", new Num(7), new Num(4))),
                new Neg(new Num(5)));
            Program.Check("ast eval", e.Eval(), 20);

            // a deeper AST: 2 * (3 + 4 * (5 - 1)) = 2 * (3 + 16) = 38
            Expr e2 = new BinOp("*",
                new Num(2),
                new BinOp("+",
                    new Num(3),
                    new BinOp("*", new Num(4), new BinOp("-", new Num(5), new Num(1)))));
            Program.Check("ast deep", e2.Eval(), 38);

            // integer division and nested negation: -(20 / 3) = -6
            Expr e3 = new Neg(new BinOp("/", new Num(20), new Num(3)));
            Program.Check("ast div neg", e3.Eval(), -6);

            // functional core over 1..10
            List<int> nums = new List<int> { 1, 2, 3, 4, 5, 6, 7, 8, 9, 10 };
            Func<int, int> sq = x => x * x;
            Func<int, int> isEven = x => x % 2 == 0 ? 1 : 0;
            Func<int, int, int> add = (a, b) => a + b;
            Func<int, int, int> mul = (a, b) => a * b;

            List<int> squares = Program.Map(nums, sq);
            Program.Check("map first", squares[0], 1);
            Program.Check("map last", squares[9], 100);
            Program.Check("map count", squares.Count, 10);

            List<int> evens = Program.Filter(nums, isEven);
            Program.Check("filter count", evens.Count, 5);
            Program.Check("filter first", evens[0], 2);
            Program.Check("reduce sum", Program.Reduce(nums, add, 0), 55);
            Program.Check("reduce even sum", Program.Reduce(evens, add, 0), 30);

            // filter -> map -> reduce: sum of squares of evens = 4+16+36+64+100
            int sse = Program.Reduce(Program.Map(evens, sq), add, 0);
            Program.Check("sum sq evens", sse, 220);

            // reduce as product = 5!
            List<int> small = new List<int> { 1, 2, 3, 4, 5 };
            Program.Check("reduce product", Program.Reduce(small, mul, 1), 120);

            // function composition, both orders
            Func<int, int> inc = x => x + 1;
            Func<int, int> sqThenInc = Program.Compose(inc, sq);   // inc(sq(x))
            Func<int, int> incThenSq = Program.Compose(sq, inc);   // sq(inc(x))
            Program.Check("compose sq-inc", sqThenInc(4), 17);      // 16 + 1
            Program.Check("compose inc-sq", incThenSq(4), 25);      // (4+1)^2

            // closures capturing independent state
            Func<int, int> add10 = Program.Adder(10);
            Func<int, int> add100 = Program.Adder(100);
            Program.Check("closure a", add10(5), 15);
            Program.Check("closure b", add100(5), 105);
            Program.Check("closures independent", add10(1) + add100(1), 112);

            // a closure that captures and mutates List state across calls
            List<int> state = new List<int>();
            state.Add(0);
            Func<int, int> accumulate = x => { state[0] = state[0] + x; return state[0]; };
            Program.Check("accumulate 1", accumulate(5), 5);
            Program.Check("accumulate 2", accumulate(10), 15);
            Program.Check("accumulate 3", accumulate(100), 115);
            Program.Check("captured state", state[0], 115);

            // an Action (void delegate) with a side effect
            List<int> logged = new List<int>();
            Action<int> record = x => { logged.Add(x * x); };
            record(2);
            record(3);
            record(4);
            Program.Check("action count", logged.Count, 3);
            Program.Check("action last", logged[2], 16);

            // delegates stored in a List and applied in a pipeline
            List<Func<int, int>> pipeline = new List<Func<int, int>>();
            pipeline.Add(x => x + 3);
            pipeline.Add(x => x * 2);
            pipeline.Add(x => x - 1);
            int piped = 5;
            foreach (var stage in pipeline)
            {
                piped = stage(piped);
            }
            Program.Check("pipeline", piped, 15);   // ((5 + 3) * 2) - 1

            if (Program.Fails == 0)
            {
                Console.WriteLine("C# big test 3 (OO and functional) passed");
            }
            return Program.Fails;
        }
    }
}
