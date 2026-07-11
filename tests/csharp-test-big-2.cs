// C# subset self test -- BIG 2: recursion, control flow and number theory.
//
// Theme: heavy recursion and branching over integers. Euclid gcd/lcm, factorial,
// recursive and iterative Fibonacci cross-checked, Ackermann, Collatz, digit tricks,
// perfect numbers, fast integer power, modular exponentiation, primality, the sieve of
// Eratosthenes, plus two little automata: a switch-driven DFA and a stack-driven bracket
// matcher. Program.Main returns the number of failed checks, so a clean run exits 0.

using System;
using System.Collections.Generic;

namespace Demo
{
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

        // ----- recursion -----

        static int Gcd(int a, int b)
        {
            if (b == 0)
            {
                return a;
            }
            return Program.Gcd(b, a % b);
        }

        static int Lcm(int a, int b)
        {
            return a / Program.Gcd(a, b) * b;
        }

        static int Fact(int n)
        {
            int f = 1;
            for (int i = 2; i <= n; i++)
            {
                f = f * i;
            }
            return f;
        }

        static int FibRec(int n)
        {
            if (n < 2)
            {
                return n;
            }
            return Program.FibRec(n - 1) + Program.FibRec(n - 2);
        }

        static int FibIter(int n)
        {
            int a = 0;
            int b = 1;
            for (int i = 0; i < n; i++)
            {
                int t = a + b;
                a = b;
                b = t;
            }
            return a;
        }

        // The Ackermann function: doubly recursive, a good recursion stress test.
        static int Ack(int m, int n)
        {
            if (m == 0)
            {
                return n + 1;
            }
            if (n == 0)
            {
                return Program.Ack(m - 1, 1);
            }
            return Program.Ack(m - 1, Program.Ack(m, n - 1));
        }

        // ----- number theory -----

        static int Collatz(int n)
        {
            int steps = 0;
            while (n != 1)
            {
                if (n % 2 == 0)
                {
                    n = n / 2;
                }
                else
                {
                    n = 3 * n + 1;
                }
                steps = steps + 1;
            }
            return steps;
        }

        static int DigitSum(int n)
        {
            if (n < 0)
            {
                n = -n;
            }
            int s = 0;
            while (n > 0)
            {
                s = s + n % 10;
                n = n / 10;
            }
            return s;
        }

        static int ReverseDigits(int n)
        {
            int r = 0;
            while (n > 0)
            {
                r = r * 10 + n % 10;
                n = n / 10;
            }
            return r;
        }

        static bool IsNumberPalindrome(int n)
        {
            return n == Program.ReverseDigits(n);
        }

        static int SumProperDivisors(int n)
        {
            int s = 0;
            for (int d = 1; d < n; d++)
            {
                if (n % d == 0)
                {
                    s = s + d;
                }
            }
            return s;
        }

        static bool IsPerfect(int n)
        {
            return Program.SumProperDivisors(n) == n;
        }

        static int IntPow(int b, int e)
        {
            int r = 1;
            while (e > 0)
            {
                if (e % 2 == 1)
                {
                    r = r * b;
                }
                b = b * b;
                e = e / 2;
            }
            return r;
        }

        // Modular exponentiation; b stays reduced mod m so products fit in 32 bits.
        static int ModPow(int b, int e, int m)
        {
            int r = 1;
            b = b % m;
            while (e > 0)
            {
                if (e % 2 == 1)
                {
                    r = (r * b) % m;
                }
                e = e / 2;
                b = (b * b) % m;
            }
            return r;
        }

        static bool IsPrime(int n)
        {
            if (n < 2)
            {
                return false;
            }
            if (n == 2)
            {
                return true;
            }
            if (n % 2 == 0)
            {
                return false;
            }
            for (int d = 3; d * d <= n; d = d + 2)
            {
                if (n % d == 0)
                {
                    return false;
                }
            }
            return true;
        }

        // Sieve of Eratosthenes over a List of flags; returns how many primes are <= limit.
        static int CountPrimes(int limit)
        {
            List<int> composite = new List<int>();
            for (int i = 0; i <= limit; i++)
            {
                composite.Add(0);
            }
            int count = 0;
            for (int i = 2; i <= limit; i++)
            {
                if (composite[i] == 0)
                {
                    count = count + 1;
                    for (int j = i + i; j <= limit; j = j + i)
                    {
                        composite[j] = 1;
                    }
                }
            }
            return count;
        }

        static int NthPrime(int k)
        {
            int count = 0;
            int n = 1;
            while (count < k)
            {
                n = n + 1;
                if (Program.IsPrime(n))
                {
                    count = count + 1;
                }
            }
            return n;
        }

        // ----- base conversion -----

        static string ToBinary(int n)
        {
            if (n == 0)
            {
                return "0";
            }
            string s = "";
            while (n > 0)
            {
                s = (n % 2 == 1 ? "1" : "0") + s;
                n = n / 2;
            }
            return s;
        }

        static int FromBinary(string s)
        {
            int v = 0;
            for (int i = 0; i < s.Length; i++)
            {
                v = v * 2 + (s[i] == "1" ? 1 : 0);
            }
            return v;
        }

        // ----- automata -----

        // A DFA recognizing a+b+ (one or more a's, then one or more b's), driven by a
        // switch on the current state (0 start, 1 in-a's, 2 in-b's, 3 dead).
        static bool MatchAB(string s)
        {
            int state = 0;
            for (int i = 0; i < s.Length; i++)
            {
                string c = s[i];
                switch (state)
                {
                    case 0:
                        state = c == "a" ? 1 : 3;
                        break;
                    case 1:
                        if (c == "a")
                        {
                            state = 1;
                        }
                        else if (c == "b")
                        {
                            state = 2;
                        }
                        else
                        {
                            state = 3;
                        }
                        break;
                    case 2:
                        state = c == "b" ? 2 : 3;
                        break;
                    default:
                        state = 3;
                        break;
                }
            }
            return state == 2;
        }

        static int OpenCode(string c)
        {
            switch (c)
            {
                case "(":
                    return 1;
                case "[":
                    return 2;
                case "{":
                    return 3;
                default:
                    return 0;
            }
        }

        static int CloseCode(string c)
        {
            switch (c)
            {
                case ")":
                    return 1;
                case "]":
                    return 2;
                case "}":
                    return 3;
                default:
                    return 0;
            }
        }

        // A pushdown bracket matcher over a List used as a stack.
        static bool IsBalanced(string s)
        {
            List<int> stack = new List<int>();
            int top = 0;
            for (int i = 0; i < s.Length; i++)
            {
                string c = s[i];
                int oc = Program.OpenCode(c);
                int cc = Program.CloseCode(c);
                if (oc != 0)
                {
                    if (top < stack.Count)
                    {
                        stack[top] = oc;
                    }
                    else
                    {
                        stack.Add(oc);
                    }
                    top = top + 1;
                }
                else if (cc != 0)
                {
                    if (top == 0)
                    {
                        return false;
                    }
                    top = top - 1;
                    if (stack[top] != cc)
                    {
                        return false;
                    }
                }
            }
            return top == 0;
        }

        static int Main()
        {
            // gcd / lcm
            Program.Check("gcd 48 36", Program.Gcd(48, 36), 12);
            Program.Check("gcd 17 5", Program.Gcd(17, 5), 1);
            Program.Check("gcd 100 0", Program.Gcd(100, 0), 100);
            Program.Check("lcm 4 6", Program.Lcm(4, 6), 12);
            Program.Check("lcm 21 6", Program.Lcm(21, 6), 42);

            // factorial
            Program.Check("fact 0", Program.Fact(0), 1);
            Program.Check("fact 5", Program.Fact(5), 120);
            Program.Check("fact 10", Program.Fact(10), 3628800);

            // fibonacci: recursive and iterative must agree
            Program.Check("fib rec 10", Program.FibRec(10), 55);
            Program.Check("fib iter 10", Program.FibIter(10), 55);
            for (int n = 0; n <= 20; n++)
            {
                Program.Check($"fib agree {n}", Program.FibRec(n), Program.FibIter(n));
            }
            Program.Check("fib 20", Program.FibIter(20), 6765);

            // Ackermann
            Program.Check("ack 0 0", Program.Ack(0, 0), 1);
            Program.Check("ack 2 3", Program.Ack(2, 3), 9);
            Program.Check("ack 3 3", Program.Ack(3, 3), 61);

            // Collatz step counts
            Program.Check("collatz 1", Program.Collatz(1), 0);
            Program.Check("collatz 6", Program.Collatz(6), 8);
            Program.Check("collatz 27", Program.Collatz(27), 111);

            // digit tricks
            Program.Check("digit sum 12345", Program.DigitSum(12345), 15);
            Program.Check("digit sum neg", Program.DigitSum(-908), 17);
            Program.Check("reverse 1234", Program.ReverseDigits(1234), 4321);
            Program.CheckB("palindrome 12321", Program.IsNumberPalindrome(12321), true);
            Program.CheckB("palindrome 12345", Program.IsNumberPalindrome(12345), false);

            // perfect numbers
            Program.Check("divisors 28", Program.SumProperDivisors(28), 28);
            Program.CheckB("perfect 6", Program.IsPerfect(6), true);
            Program.CheckB("perfect 28", Program.IsPerfect(28), true);
            Program.CheckB("perfect 496", Program.IsPerfect(496), true);
            Program.CheckB("perfect 12", Program.IsPerfect(12), false);

            // integer and modular power
            Program.Check("pow 2^10", Program.IntPow(2, 10), 1024);
            Program.Check("pow 3^5", Program.IntPow(3, 5), 243);
            Program.Check("pow 5^0", Program.IntPow(5, 0), 1);
            Program.Check("pow 7^4", Program.IntPow(7, 4), 2401);
            Program.Check("modpow 2^10 %1000", Program.ModPow(2, 10, 1000), 24);
            Program.Check("modpow 3^13 %7", Program.ModPow(3, 13, 7), 3);
            Program.Check("modpow 3^100 %7", Program.ModPow(3, 100, 7), 4);

            // primality and the sieve
            Program.CheckB("prime 2", Program.IsPrime(2), true);
            Program.CheckB("prime 97", Program.IsPrime(97), true);
            Program.CheckB("prime 1", Program.IsPrime(1), false);
            Program.CheckB("prime 91", Program.IsPrime(91), false);
            Program.Check("count primes 30", Program.CountPrimes(30), 10);
            Program.Check("count primes 100", Program.CountPrimes(100), 25);
            Program.Check("nth prime 1", Program.NthPrime(1), 2);
            Program.Check("nth prime 10", Program.NthPrime(10), 29);
            Program.Check("nth prime 25", Program.NthPrime(25), 97);

            // base conversion round trips
            Program.Check("to binary 0", Program.FromBinary(Program.ToBinary(0)), 0);
            Program.Check("to binary 13", Program.FromBinary(Program.ToBinary(13)), 13);
            Program.Check("to binary 255", Program.FromBinary(Program.ToBinary(255)), 255);
            Program.Check("from binary 1010", Program.FromBinary("1010"), 10);

            // DFA a+b+
            Program.CheckB("dfa aaabbb", Program.MatchAB("aaabbb"), true);
            Program.CheckB("dfa ab", Program.MatchAB("ab"), true);
            Program.CheckB("dfa ba", Program.MatchAB("ba"), false);
            Program.CheckB("dfa aabbaa", Program.MatchAB("aabbaa"), false);
            Program.CheckB("dfa empty", Program.MatchAB(""), false);
            Program.CheckB("dfa only a", Program.MatchAB("aaa"), false);

            // bracket matcher
            Program.CheckB("balanced nested", Program.IsBalanced("([{}])"), true);
            Program.CheckB("balanced pairs", Program.IsBalanced("()[]{}"), true);
            Program.CheckB("balanced empty", Program.IsBalanced(""), true);
            Program.CheckB("unbalanced cross", Program.IsBalanced("([)]"), false);
            Program.CheckB("unbalanced open", Program.IsBalanced("(()"), false);
            Program.CheckB("unbalanced close", Program.IsBalanced(")("), false);

            if (Program.Fails == 0)
            {
                Console.WriteLine("C# big test 2 (number theory) passed");
            }
            return Program.Fails;
        }
    }
}
