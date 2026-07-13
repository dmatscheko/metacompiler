// Multi-file C# test: geo.Vec lives in tests/imports/geo.cs and is found via the -i
// include root (mec -i tests/imports ...). The imported file is parsed with the same
// grammar; its namespace and class register like the main file's. System is a builtin
// no-op import, mixed in on purpose.
using System;
using geo;

namespace Demo
{
    class Program
    {
        static int Fails = 0;

        static void Check(string name, bool cond)
        {
            if (!cond)
            {
                Console.WriteLine("FAIL " + name);
                Program.Fails++;
            }
        }

        static int Main()
        {
            Vec a = new Vec(3, 4);
            Vec b = new Vec(2, -1);
            Program.Check("imported class", a.Dot(b) == 2);
            Program.Check("imported static", Vec.Scale(a, 2).Dot(b) == 4);

            if (Program.Fails == 0)
            {
                Console.WriteLine("csharp multifile test passed");
            }
            return Program.Fails;
        }
    }
}
