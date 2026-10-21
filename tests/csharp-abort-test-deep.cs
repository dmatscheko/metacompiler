// ABORT PROBE. This program is SUPPOSED to die. See the "SHOULD ABORT" rows in
// .vscode/launch.json and abort_norm() in test.sh. A recursion with no base case
// is a StackOverflowException, which since .NET 2.0 no catch clause can take:
// the runtime writes "Stack overflow." on stderr and aborts the process (its
// repeated frame list is not reproduced). Documentation-cited: no dotnet on
// this machine.
using System;

class Program {
    static int Rec(int n) { return Rec(n + 1) + 1; }
    static void Main() {
        Console.WriteLine("before the abort");
        try { Rec(0); } catch (Exception e) { Console.WriteLine("caught " + e); }
    }
}
