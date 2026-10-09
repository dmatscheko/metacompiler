// See mod.kt.
using System;

namespace Bench
{
    class Program
    {
        static int Main(string[] args)
        {
            long s = 0;
            long i = 0;
            while (i < 40000)
            {
                s = s + i % 7;
                i = i + 1;
            }
            Console.WriteLine(s);
            return 0;
        }
    }
}
