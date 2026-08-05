// See mod.kt. csharp is here because it shares lib/runtime-jvm.metajs with java -
// including rtjIsIntegral, whose fifteen call sites made a one-line delegation
// cost java +5.7% (see that body's comment). A shared hot predicate needs BOTH
// its callers measured, and until this row existed only java's was.
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
