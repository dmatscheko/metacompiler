# Ruby subset - big self test 4: STRING AND NUMBER PROCESSING.
# A recursive-descent arithmetic evaluator (a Calc class over a digit string with
# + - * / and parentheses), Roman-numeral conversion both ways, base conversion
# (binary / hex and back), palindrome tests for strings and numbers, a Caesar cipher
# with round-trip decode, run-length encoding, FizzBuzz, integer square root and
# assorted digit arithmetic. Everything is built from the implemented subset: string
# indexing s[i], structural string equality, interpolation, and integer math. Counts
# failures and ends with exit(fails); the interpreter and the LLVM-IR compiler must
# agree.

fails = 0

def check(name, got, want)
  if got != want
    puts "FAIL #{name}: got #{got} want #{want}"
    fails = fails + 1
  end
end

# ----- character helpers (scan a literal alphabet, no global tables) -----

def digit_val(c)
  digs = "0123456789"
  i = 0
  while i < 10
    return i if digs[i] == c
    i += 1
  end
  return -1
end

def is_digit(c)
  digit_val(c) >= 0
end

def char_index(c)
  alpha = "abcdefghijklmnopqrstuvwxyz"
  i = 0
  while i < 26
    return i if alpha[i] == c
    i += 1
  end
  return -1
end

def char_at_index(idx)
  alpha = "abcdefghijklmnopqrstuvwxyz"
  alpha[idx]
end

check("digit_val 7", digit_val("7"), 7)
check("digit_val non", digit_val("x"), -1)
check("is_digit yes", is_digit("3"), true)
check("is_digit no", is_digit("+"), false)
check("char_index a", char_index("a"), 0)
check("char_index z", char_index("z"), 25)
check("char_at 4", char_at_index(4), "e")

# ----- recursive-descent arithmetic evaluator -----

class Calc
  def initialize(src)
    @src = src
    @pos = 0
  end
  def peek
    return "" if @pos >= @src.length
    @src[@pos]
  end
  def advance
    @pos += 1
  end
  def parse_number
    n = 0
    while is_digit(self.peek)
      n = n * 10 + digit_val(self.peek)
      self.advance
    end
    n
  end
  def parse_factor
    if self.peek == "("
      self.advance
      v = self.parse_expr
      self.advance
      return v
    end
    self.parse_number
  end
  def parse_term
    v = self.parse_factor
    while self.peek == "*" || self.peek == "/"
      op = self.peek
      self.advance
      rhs = self.parse_factor
      if op == "*"
        v = v * rhs
      else
        v = v / rhs
      end
    end
    v
  end
  def parse_expr
    v = self.parse_term
    while self.peek == "+" || self.peek == "-"
      op = self.peek
      self.advance
      rhs = self.parse_term
      if op == "+"
        v = v + rhs
      else
        v = v - rhs
      end
    end
    v
  end
end

def evaluate(expr)
  Calc.new(expr).parse_expr
end

check("eval number", evaluate("42"), 42)
check("eval add", evaluate("2+3"), 5)
check("eval precedence", evaluate("2+3*4"), 14)
check("eval left assoc", evaluate("10-2-3"), 5)
check("eval parens", evaluate("(2+3)*4"), 20)
check("eval mixed", evaluate("2*3+4*5"), 26)
check("eval div", evaluate("100/5/2"), 10)
check("eval nested parens", evaluate("((1+2)*(3+4))"), 21)
check("eval deep", evaluate("2*(3+(4*5))"), 46)

# ----- Roman numerals both directions -----

def to_roman(n)
  values = [1000, 900, 500, 400, 100, 90, 50, 40, 10, 9, 5, 4, 1]
  symbols = ["M", "CM", "D", "CD", "C", "XC", "L", "XL", "X", "IX", "V", "IV", "I"]
  out = ""
  i = 0
  while i < values.size
    while n >= values[i]
      out += symbols[i]
      n -= values[i]
    end
    i += 1
  end
  out
end

def from_roman(s)
  vals = {"I" => 1, "V" => 5, "X" => 10, "L" => 50, "C" => 100, "D" => 500, "M" => 1000}
  total = 0
  i = 0
  n = s.length
  while i < n
    cur = vals[s[i]]
    if i + 1 < n && vals[s[i + 1]] > cur
      total -= cur
    else
      total += cur
    end
    i += 1
  end
  total
end

check("roman 4", to_roman(4), "IV")
check("roman 9", to_roman(9), "IX")
check("roman 58", to_roman(58), "LVIII")
check("roman 1994", to_roman(1994), "MCMXCIV")
check("roman 2023", to_roman(2023), "MMXXIII")
check("from_roman IV", from_roman("IV"), 4)
check("from_roman LVIII", from_roman("LVIII"), 58)
check("from_roman MCMXCIV", from_roman("MCMXCIV"), 1994)
# round-trip a whole span of numbers
roman_ok = true
k = 1
while k <= 100
  if from_roman(to_roman(k)) != k
    roman_ok = false
  end
  k += 1
end
check("roman round-trip 1..100", roman_ok, true)

# ----- base conversion -----

def to_base(n, base)
  return "0" if n == 0
  digs = "0123456789abcdef"
  out = ""
  while n > 0
    out = digs[n % base] + out
    n = n / base
  end
  out
end

def parse_binary(s)
  n = 0
  i = 0
  len = s.length
  while i < len
    n = n * 2
    n = n + 1 if s[i] == "1"
    i += 1
  end
  n
end

check("binary 13", to_base(13, 2), "1101")
check("binary 0", to_base(0, 2), "0")
check("hex 255", to_base(255, 16), "ff")
check("hex 4096", to_base(4096, 16), "1000")
check("octal 8", to_base(8, 8), "10")
check("parse_binary 1101", parse_binary("1101"), 13)
check("parse_binary round-trip", parse_binary(to_base(42, 2)), 42)

# ----- digit arithmetic -----

def reverse_num(n)
  r = 0
  while n > 0
    r = r * 10 + n % 10
    n = n / 10
  end
  r
end

def count_digits(n)
  return 1 if n == 0
  c = 0
  while n > 0
    c += 1
    n = n / 10
  end
  c
end

def palindrome_num(n)
  reverse_num(n) == n
end

check("reverse 1234", reverse_num(1234), 4321)
check("reverse 1200", reverse_num(1200), 21)
check("count_digits 0", count_digits(0), 1)
check("count_digits 90210", count_digits(90210), 5)
check("palindrome 1221", palindrome_num(1221), true)
check("palindrome 1231", palindrome_num(1231), false)
check("palindrome 7", palindrome_num(7), true)

# ----- integer square root -----

def isqrt(n)
  r = 0
  while (r + 1) * (r + 1) <= n
    r += 1
  end
  r
end

check("isqrt 0", isqrt(0), 0)
check("isqrt 16", isqrt(16), 4)
check("isqrt 24", isqrt(24), 4)
check("isqrt 25", isqrt(25), 5)
check("isqrt 143", isqrt(143), 11)

# ----- string processing -----

def reverse_str(s)
  out = ""
  i = s.length - 1
  while i >= 0
    out += s[i]
    i -= 1
  end
  out
end

def palindrome_str(s)
  i = 0
  j = s.length - 1
  while i < j
    return false if s[i] != s[j]
    i += 1
    j -= 1
  end
  true
end

def count_char(s, ch)
  c = 0
  i = 0
  len = s.length
  while i < len
    c += 1 if s[i] == ch
    i += 1
  end
  c
end

check("reverse_str", reverse_str("hello"), "olleh")
check("reverse palindrome", reverse_str("racecar"), "racecar")
check("palindrome_str yes", palindrome_str("racecar"), true)
check("palindrome_str no", palindrome_str("ruby"), false)
check("palindrome_str even", palindrome_str("abba"), true)
check("count_char l", count_char("hello world", "l"), 3)
check("count_char o", count_char("hello world", "o"), 2)
check("count_char none", count_char("hello", "z"), 0)

# ----- Caesar cipher with round-trip -----

def caesar(s, shift)
  out = ""
  i = 0
  len = s.length
  while i < len
    c = s[i]
    idx = char_index(c)
    if idx < 0
      out += c
    else
      out += char_at_index((idx + shift) % 26)
    end
    i += 1
  end
  out
end

check("caesar shift 3", caesar("abc", 3), "def")
check("caesar wrap", caesar("xyz", 3), "abc")
check("caesar keeps non-letters", caesar("a b", 1), "b c")
# encrypt then decrypt (shift 5 then 21 = 26 = identity)
secret = caesar("thequickbrownfox", 5)
check("caesar round-trip", caesar(secret, 21), "thequickbrownfox")

# ----- FizzBuzz -----

def fizzbuzz(n)
  if n % 15 == 0
    "FizzBuzz"
  elsif n % 3 == 0
    "Fizz"
  elsif n % 5 == 0
    "Buzz"
  else
    "#{n}"
  end
end

check("fb 3", fizzbuzz(3), "Fizz")
check("fb 5", fizzbuzz(5), "Buzz")
check("fb 15", fizzbuzz(15), "FizzBuzz")
check("fb 7", fizzbuzz(7), "7")
fb = ""
(1..5).each do |k|
  fb += fizzbuzz(k)
  fb += " "
end
check("fb sequence", fb, "1 2 Fizz 4 Buzz ")

# ----- run-length encoding -----

def rle(s)
  n = s.length
  return "" if n == 0
  out = ""
  count = 1
  i = 1
  while i <= n
    if i < n && s[i] == s[i - 1]
      count += 1
    else
      out += "#{s[i - 1]}#{count}"
      count = 1
    end
    i += 1
  end
  out
end

check("rle empty", rle(""), "")
check("rle single", rle("a"), "a1")
check("rle repeats", rle("aaabbc"), "a3b2c1")
check("rle all same", rle("zzzz"), "z4")
check("rle no repeats", rle("abcd"), "a1b1c1d1")

# ----- done -----
if fails == 0
  puts "Ruby big self test 4 (strings and numbers) passed"
end
exit(fails)
