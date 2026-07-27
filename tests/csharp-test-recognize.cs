// C# recognition test: a real-world-looking file that exercises the widened surface.
//
// GENUINELY compiled here: attributes ([...]), a global using and a file scoped
// namespace, bitwise/shift operators (| ^ & << >> ~ and their |= &= <<= ... forms), a
// char literal, the extended numeric literals (0x.., 0b.., 1_000, 100L, 1.5), a where
// constraint, an expression bodied constructor, and the checked / lock / using statements.
//
// ALSO GENUINELY COMPILED NOW: an interface, an enum, a { get; set; } property, verbatim
// strings, ?? and ??=, is / as tests, null-conditional ?. access, and typeof / nameof.
// The file runs and prints 123 with or without -warn-unsupported, and a run with the flag
// reports no unsupported construct at all.
//
// History: those constructs used to be recognized-but-not-implemented, so a flagless run
// aborted at the first of them and this file was a by-design SHOULD-FAIL guard. The
// full-syntax work implemented every one (properties and is/as were the last two), which
// removed the guard's premise, so the matrix entry became an ordinary one. Do not re-arm
// it: the C# here is valid and is meant to keep working.
global using System;
using System.Collections.Generic;
using System.Text;

namespace Demo.Widget;

[Serializable]
public interface IShape
{
    double Area();
}

[Flags]
public enum Access
{
    None = 0,
    Read = 0b0001,
    Write = 0b0010,
    Execute = 0b0100,
}

[Serializable]
public class Program
{
    // Extended numeric literals (genuine): hex, binary, underscore groups and suffixes.
    private const int Mask = 0xFF;
    private static readonly long Big = 1_000_000L;
    private static double Ratio = 1.5;

    // An attributed { get; set; } property (recognized, not implemented).
    [Obsolete]
    public int Count { get; set; }

    // An expression bodied constructor (genuine).
    private int seed;
    public Program(int s) => seed = s;

    // A generic method with a where constraint (the constraint is accepted and ignored).
    static T Echo<T>(T value) where T : class
    {
        return value;
    }

    static int Main()
    {
        // Genuine bitwise / shift arithmetic and compound assignment.
        int flags = 0x0F;
        flags |= 0x10;
        flags &= ~0x01;
        int packed = (flags << 4) | 0x0F;
        int shifted = packed >> 2;

        // A char literal (genuine); checked / lock / using run their bodies (genuine).
        char grade = 'A';
        object gate = grade;
        checked
        {
            int total = shifted + 1;
            shifted = total - 1;
        }
        lock (gate)
        {
            shifted = shifted ^ 0;
        }
        using (var buffer = new List<int>())
        {
            buffer.Add(shifted);
        }

        // Recognized, not implemented: verbatim string, ??, ??=, is / as, ?. and typeof.
        string path = @"C:\logs\widget.txt";
        string name = null;
        string label = name ?? "default";
        name ??= "fallback";
        object boxed = shifted;
        bool isInt = boxed is int;
        string text = boxed as string;
        int? maybe = null;
        var len = path?.Length;
        var shape = typeof(IShape);
        var ident = nameof(Program);

        Console.WriteLine(shifted);
        return 0;
    }
}
