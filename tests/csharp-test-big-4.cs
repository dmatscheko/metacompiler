// C# subset self test -- BIG 4: string and number processing.
//
// Theme: parsing and formatting over strings and integers. A hand-written integer parser
// and formatter (cross-checked against $"{n}" interpolation), string reversal and
// palindrome tests, character counting and word counting, a Caesar cipher with wrap-around,
// run-length encoding with a decoder, Roman-numeral conversion both ways, and a
// space-separated RPN calculator driven by a List-backed stack. Program.Main returns the
// number of failed checks, so a clean run exits 0.
//
// Note: this subset implements the two-argument Substring(a, b) as slice(a, b), i.e. the
// second argument is an exclusive END index, not a length -- the checks below expect that.

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

        static void CheckS(string name, string got, string want)
        {
            if (got != want)
            {
                Console.WriteLine($"FAIL {name}: got {got} want {want}");
                Program.Fails++;
            }
        }

        // ----- digit helpers -----

        // The numeric value of a single-character digit string, or -1 if not a digit.
        static int Digit(string c)
        {
            return "0123456789".IndexOf(c);
        }

        static bool IsDigit(string c)
        {
            return Program.Digit(c) >= 0;
        }

        // ----- integer <-> string -----

        static int ParseInt(string s)
        {
            int i = 0;
            int sign = 1;
            if (s.Length > 0 && s[0] == "-")
            {
                sign = -1;
                i = 1;
            }
            int v = 0;
            while (i < s.Length)
            {
                v = v * 10 + Program.Digit(s[i]);
                i = i + 1;
            }
            return sign * v;
        }

        static string IntToStr(int n)
        {
            if (n == 0)
            {
                return "0";
            }
            bool neg = n < 0;
            if (neg)
            {
                n = -n;
            }
            string digits = "0123456789";
            string s = "";
            while (n > 0)
            {
                s = digits[n % 10] + s;
                n = n / 10;
            }
            if (neg)
            {
                s = "-" + s;
            }
            return s;
        }

        // ----- basic string ops -----

        static string Reverse(string s)
        {
            string r = "";
            for (int i = s.Length - 1; i >= 0; i--)
            {
                r = r + s[i];
            }
            return r;
        }

        static bool IsPalindrome(string s)
        {
            int i = 0;
            int j = s.Length - 1;
            while (i < j)
            {
                if (s[i] != s[j])
                {
                    return false;
                }
                i = i + 1;
                j = j - 1;
            }
            return true;
        }

        static int CountChar(string s, string c)
        {
            int n = 0;
            for (int i = 0; i < s.Length; i++)
            {
                if (s[i] == c)
                {
                    n = n + 1;
                }
            }
            return n;
        }

        // Counts space-separated words, tolerating leading, trailing and repeated spaces.
        static int CountWords(string s)
        {
            int count = 0;
            int i = 0;
            while (i < s.Length)
            {
                while (i < s.Length && s[i] == " ")
                {
                    i = i + 1;
                }
                if (i >= s.Length)
                {
                    break;
                }
                count = count + 1;
                while (i < s.Length && s[i] != " ")
                {
                    i = i + 1;
                }
            }
            return count;
        }

        // ----- Caesar cipher -----

        static string Caesar(string s, int k)
        {
            string alpha = "abcdefghijklmnopqrstuvwxyz";
            string res = "";
            for (int i = 0; i < s.Length; i++)
            {
                string c = s[i];
                int idx = alpha.IndexOf(c);
                if (idx < 0)
                {
                    res = res + c;
                }
                else
                {
                    int ni = (idx + k) % 26;
                    if (ni < 0)
                    {
                        ni = ni + 26;
                    }
                    res = res + alpha[ni];
                }
            }
            return res;
        }

        // ----- run-length encoding -----

        static string RleEncode(string s)
        {
            string res = "";
            int i = 0;
            while (i < s.Length)
            {
                string c = s[i];
                int count = 1;
                while (i + count < s.Length && s[i + count] == c)
                {
                    count = count + 1;
                }
                res = res + c + Program.IntToStr(count);
                i = i + count;
            }
            return res;
        }

        static string RleDecode(string s)
        {
            string res = "";
            int i = 0;
            while (i < s.Length)
            {
                string c = s[i];
                i = i + 1;
                int count = 0;
                while (i < s.Length && Program.IsDigit(s[i]))
                {
                    count = count * 10 + Program.Digit(s[i]);
                    i = i + 1;
                }
                for (int k = 0; k < count; k++)
                {
                    res = res + c;
                }
            }
            return res;
        }

        // ----- Roman numerals -----

        static string ToRoman(int n)
        {
            int[] vals = new int[] { 1000, 900, 500, 400, 100, 90, 50, 40, 10, 9, 5, 4, 1 };
            string[] syms = new string[] { "M", "CM", "D", "CD", "C", "XC", "L", "XL", "X", "IX", "V", "IV", "I" };
            string res = "";
            for (int i = 0; i < vals.Length; i++)
            {
                while (n >= vals[i])
                {
                    res = res + syms[i];
                    n = n - vals[i];
                }
            }
            return res;
        }

        static int RomanValue(string c)
        {
            switch (c)
            {
                case "I":
                    return 1;
                case "V":
                    return 5;
                case "X":
                    return 10;
                case "L":
                    return 50;
                case "C":
                    return 100;
                case "D":
                    return 500;
                case "M":
                    return 1000;
                default:
                    return 0;
            }
        }

        static int FromRoman(string s)
        {
            int total = 0;
            for (int i = 0; i < s.Length; i++)
            {
                int cur = Program.RomanValue(s[i]);
                int nxt = 0;
                if (i + 1 < s.Length)
                {
                    nxt = Program.RomanValue(s[i + 1]);
                }
                if (cur < nxt)
                {
                    total = total - cur;
                }
                else
                {
                    total = total + cur;
                }
            }
            return total;
        }

        // ----- RPN calculator -----

        // Evaluates a space-separated postfix expression using a List as an operand stack.
        static int EvalRpn(string s)
        {
            List<int> stack = new List<int>();
            int top = 0;
            int i = 0;
            while (i < s.Length)
            {
                string c = s[i];
                if (c == " ")
                {
                    i = i + 1;
                    continue;
                }
                if (c == "+" || c == "-" || c == "*" || c == "/")
                {
                    top = top - 1;
                    int b = stack[top];
                    top = top - 1;
                    int a = stack[top];
                    int r = 0;
                    switch (c)
                    {
                        case "+":
                            r = a + b;
                            break;
                        case "-":
                            r = a - b;
                            break;
                        case "*":
                            r = a * b;
                            break;
                        case "/":
                            r = a / b;
                            break;
                    }
                    if (top < stack.Count)
                    {
                        stack[top] = r;
                    }
                    else
                    {
                        stack.Add(r);
                    }
                    top = top + 1;
                    i = i + 1;
                }
                else
                {
                    int num = 0;
                    while (i < s.Length && Program.IsDigit(s[i]))
                    {
                        num = num * 10 + Program.Digit(s[i]);
                        i = i + 1;
                    }
                    if (top < stack.Count)
                    {
                        stack[top] = num;
                    }
                    else
                    {
                        stack.Add(num);
                    }
                    top = top + 1;
                }
            }
            return stack[top - 1];
        }

        static int Main()
        {
            // digit helpers
            Program.Check("digit 7", Program.Digit("7"), 7);
            Program.Check("digit 0", Program.Digit("0"), 0);
            Program.Check("digit non", Program.Digit("x"), -1);
            Program.CheckB("isdigit yes", Program.IsDigit("5"), true);
            Program.CheckB("isdigit no", Program.IsDigit("-"), false);

            // integer parsing
            Program.Check("parse 42", Program.ParseInt("42"), 42);
            Program.Check("parse 0", Program.ParseInt("0"), 0);
            Program.Check("parse neg", Program.ParseInt("-1234"), -1234);
            Program.Check("parse leading zeros", Program.ParseInt("007"), 7);
            Program.Check("parse big", Program.ParseInt("1000000"), 1000000);

            // integer formatting, cross-checked with interpolation
            Program.CheckS("fmt 0", Program.IntToStr(0), "0");
            Program.CheckS("fmt 42", Program.IntToStr(42), "42");
            Program.CheckS("fmt neg", Program.IntToStr(-7), "-7");
            Program.CheckS("fmt 1000", Program.IntToStr(1000), "1000");
            for (int n = -50; n <= 50; n++)
            {
                Program.CheckS($"fmt agrees {n}", Program.IntToStr(n), $"{n}");
            }
            // parse and format are inverse
            Program.Check("roundtrip 98765", Program.ParseInt(Program.IntToStr(98765)), 98765);
            Program.Check("roundtrip -321", Program.ParseInt(Program.IntToStr(-321)), -321);

            // reversal and palindrome
            Program.CheckS("reverse hello", Program.Reverse("hello"), "olleh");
            Program.CheckS("reverse empty", Program.Reverse(""), "");
            Program.CheckB("palindrome racecar", Program.IsPalindrome("racecar"), true);
            Program.CheckB("palindrome abba", Program.IsPalindrome("abba"), true);
            Program.CheckB("palindrome hello", Program.IsPalindrome("hello"), false);
            Program.CheckB("palindrome single", Program.IsPalindrome("q"), true);
            Program.CheckB("palindrome empty", Program.IsPalindrome(""), true);

            // counting
            Program.Check("count a in banana", Program.CountChar("banana", "a"), 3);
            Program.Check("count n in banana", Program.CountChar("banana", "n"), 2);
            Program.Check("count z in banana", Program.CountChar("banana", "z"), 0);
            Program.Check("words normal", Program.CountWords("the quick brown fox"), 4);
            Program.Check("words padded", Program.CountWords("  hello   world  "), 2);
            Program.Check("words single", Program.CountWords("single"), 1);
            Program.Check("words empty", Program.CountWords(""), 0);
            Program.Check("words spaces", Program.CountWords("     "), 0);

            // Substring (slice semantics: second arg is an exclusive end index) and IndexOf
            Program.CheckS("substring rest", "metacompiler".Substring(4), "compiler");
            Program.CheckS("substring slice", "metacompiler".Substring(0, 4), "meta");
            Program.Check("indexof found", "metacompiler".IndexOf("compiler"), 4);
            Program.Check("indexof missing", "metacompiler".IndexOf("xyz"), -1);
            Program.Check("length", "metacompiler".Length, 12);

            // Caesar cipher: shift, wrap-around, non-letters, and round trip
            Program.CheckS("caesar abc", Program.Caesar("abc", 3), "def");
            Program.CheckS("caesar wrap", Program.Caesar("xyz", 3), "abc");
            Program.CheckS("caesar spaces", Program.Caesar("a b c", 1), "b c d");
            Program.CheckS("caesar decrypt", Program.Caesar("def", -3), "abc");
            string secret = Program.Caesar("hello world", 13);
            Program.CheckS("caesar round trip", Program.Caesar(secret, -13), "hello world");
            Program.CheckS("caesar rot13 twice", Program.Caesar(Program.Caesar("rot", 13), 13), "rot");

            // run-length encoding and decoding
            Program.CheckS("rle aaabbc", Program.RleEncode("aaabbc"), "a3b2c1");
            Program.CheckS("rle run", Program.RleEncode("wwwwww"), "w6");
            Program.CheckS("rle distinct", Program.RleEncode("abc"), "a1b1c1");
            Program.CheckS("rle decode", Program.RleDecode("a3b2c1"), "aaabbc");
            Program.CheckS("rle round trip", Program.RleDecode(Program.RleEncode("aaaaabbbbccd")), "aaaaabbbbccd");

            // Roman numerals both directions
            Program.CheckS("roman 4", Program.ToRoman(4), "IV");
            Program.CheckS("roman 9", Program.ToRoman(9), "IX");
            Program.CheckS("roman 40", Program.ToRoman(40), "XL");
            Program.CheckS("roman 1994", Program.ToRoman(1994), "MCMXCIV");
            Program.CheckS("roman 2023", Program.ToRoman(2023), "MMXXIII");
            Program.Check("from roman MCMXCIV", Program.FromRoman("MCMXCIV"), 1994);
            Program.Check("from roman XLII", Program.FromRoman("XLII"), 42);
            for (int n = 1; n <= 100; n++)
            {
                Program.Check($"roman round trip {n}", Program.FromRoman(Program.ToRoman(n)), n);
            }
            Program.Check("roman round trip 3888", Program.FromRoman(Program.ToRoman(3888)), 3888);

            // RPN calculator
            Program.Check("rpn add", Program.EvalRpn("3 4 +"), 7);
            Program.Check("rpn mixed", Program.EvalRpn("3 4 + 5 *"), 35);
            Program.Check("rpn div sub", Program.EvalRpn("10 2 / 3 -"), 2);
            Program.Check("rpn precedence", Program.EvalRpn("2 3 4 * +"), 14);
            Program.Check("rpn nested", Program.EvalRpn("100 10 5 - /"), 20);
            Program.Check("rpn multidigit", Program.EvalRpn("12 34 +"), 46);

            if (Program.Fails == 0)
            {
                Console.WriteLine("C# big test 4 (string processing) passed");
            }
            return Program.Fails;
        }
    }
}
