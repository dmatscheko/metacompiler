// C# try/catch/finally/throw, genuinely executed (interpreter and compiler).
//
// throw raises a value that unwinds through calls to the nearest catch; catch binds it
// (the first catch clause wins - the exception type and any 'when' filter are parsed but
// not discriminated / evaluated), finally always runs. A return/break/continue that leaves
// a try or catch body works in both engines. An uncaught throw is a clean runtime error.
//
// Program.Main counts failed checks and returns that count, so the run exits 0 exactly when
// everything works; the interpreter and the compiler must agree.

using System;

namespace Demo
{
    class BoomException
    {
        public int Code;
        public BoomException(int c) { this.Code = c; }
    }

    class Program
    {
        static int Fails = 0;

        static int Risky(int n)
        {
            if (n > 3) { throw new BoomException(n); }
            return n * 2;
        }

        // return out of a try, and out of a catch.
        static int Classify(int n)
        {
            try
            {
                if (n > 0) { return n * 10; }
                throw new BoomException(0);
            }
            catch (Exception e)
            {
                return -1;
            }
            finally { }
        }

        // A return out of an INNER try propagates through the OUTER try.
        static int NestedReturn()
        {
            try
            {
                try { return 9; } finally { }
            }
            finally { }
            return 0;
        }

        // break / continue leaving a try body inside a loop.
        static int LoopBreak()
        {
            int sum = 0;
            for (int i = 0; i < 10; i++)
            {
                try { if (i == 3) { break; } sum = sum + i; } finally { }
            }
            return sum;             // 0+1+2 = 3
        }

        static int LoopContinue()
        {
            int sum = 0;
            for (int i = 0; i < 5; i++)
            {
                try { if (i == 2) { continue; } sum = sum + i; } catch (Exception e) { }
            }
            return sum;             // 0+1+3+4 = 8
        }

        static void Check(int got, int want)
        {
            if (got != want) { Program.Fails = Program.Fails + 1; }
        }

        static int Main()
        {
            // basic try/catch/finally with a catch binding; control flow stays out of finally.
            string log = "";
            try
            {
                log = log + "a";
                throw new BoomException(1);
            }
            catch (Exception e)
            {
                log = log + "b";
            }
            finally
            {
                log = log + "c";
            }
            if (log != "abc") { Program.Fails = Program.Fails + 1; }

            // throw from a nested call, caught with a field read; the 'when' filter is parsed
            // (and ignored: the first catch wins unconditionally).
            int caught = -1;
            try
            {
                Program.Risky(5);
                Program.Fails = Program.Fails + 1;   // not reached
            }
            catch (BoomException e) when (e.Code > 0)
            {
                caught = e.Code;
            }
            Program.Check(caught, 5);

            // the no-throw path returns normally.
            Program.Check(Program.Risky(2), 4);

            // a typed catch with no binding.
            int flag = 0;
            try { throw new BoomException(7); }
            catch (BoomException) { flag = 1; }
            Program.Check(flag, 1);

            // a parenless catch (no type, no binding).
            int flag2 = 0;
            try { throw new BoomException(8); }
            catch { flag2 = 2; }
            Program.Check(flag2, 2);

            Program.Check(Program.Classify(4), 40);
            Program.Check(Program.Classify(-1), -1);
            Program.Check(Program.NestedReturn(), 9);
            Program.Check(Program.LoopBreak(), 3);
            Program.Check(Program.LoopContinue(), 8);

            if (Program.Fails == 0) { Console.WriteLine("C# try/catch OK"); }
            return Program.Fails;
        }
    }
}
