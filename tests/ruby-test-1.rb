# Ruby subset self test.
# The program counts failed checks in `fails` and ends with exit(fails), so the
# metacompiler run exits 0 exactly when every check passes. The interpreter
# (ruby-interpreter.abnf) and the LLVM-IR compiler (ruby-to-llvm-ir.abnf) run the
# same file and must agree.

fails = 0

def check(name, got, want)
  if got != want
    puts "FAIL #{name}: got #{got} want #{want}"
    fails = fails + 1
  end
end

# ----- arithmetic and precedence -----
check("precedence", 1 + 2 * 3, 7)
check("parens", (1 + 2) * 3, 9)
check("division", 7 / 2, 3)
check("modulo", 7 % 3, 1)
check("unary minus", -5 + 2, -3)
check("compare lt", 2 < 3, true)
check("compare eq", 3 == 3, true)
check("compare ne", 3 != 4, true)

# ----- boolean operators (symbol and word forms) -----
check("and true", true && true, true)
check("or short", false || 7, 7)
check("and value", 3 && 4, 4)
check("not", !false, true)
check("word and", (1 < 2 and 2 < 3), true)
check("word or", (false or 5), 5)
check("word not", (not false), true)

# ----- variables and compound assignment -----
x = 10
x += 5
check("plus assign", x, 15)
x -= 3
check("minus assign", x, 12)
x *= 2
check("times assign", x, 24)

# ----- strings, interpolation, concatenation -----
name = "world"
check("interpolation", "hello #{name}!", "hello world!")
check("interp expr", "sum=#{2 + 3}", "sum=5")
check("concat", "a" + "b", "ab")
check("mixed concat", "n=" + 5.to_s, "n=5")
check("length", "hello".length, 5)
check("single quote", 'raw text', "raw text")

# ----- if / elsif / else, unless -----
def classify(n)
  if n < 0
    "negative"
  elsif n == 0
    "zero"
  else
    "positive"
  end
end
check("if neg", classify(-4), "negative")
check("if zero", classify(0), "zero")
check("if pos", classify(9), "positive")

g = 5
r = if g > 3 then "big" else "small" end
check("if expression", r, "big")

u = 0
unless g > 100
  u = 1
end
check("unless", u, 1)

# ----- while / until with break and next -----
sum = 0
i = 1
while i <= 10
  sum += i
  i += 1
end
check("while sum", sum, 55)

evens = 0
j = 0
while j < 20
  j += 1
  if j % 2 == 1
    next
  end
  if j > 10
    break
  end
  evens += j
end
check("break next", evens, 30)

c = 0
until c >= 5
  c += 1
end
check("until", c, 5)

# ----- def: implicit return, explicit return, recursion -----
def add(a, b)
  a + b
end
check("implicit return", add(20, 22), 42)

def sign(n)
  if n < 0
    return -1
  end
  1
end
check("explicit return", sign(-8), -1)
check("fallthrough return", sign(3), 1)

def fib(n)
  if n < 2
    return n
  end
  fib(n - 1) + fib(n - 2)
end
check("recursion fib", fib(10), 55)

# ----- ranges and .each -----
rsum = 0
(1..5).each do |k|
  rsum += k
end
check("range each", rsum, 15)

esum = 0
(1...5).each do |k|
  esum += k
end
check("range exclusive", esum, 10)

# ----- arrays -----
arr = [3, 1, 4, 1, 5]
check("array index", arr[2], 4)
check("array size", arr.size, 5)
check("array length", arr.length, 5)
arr[0] = 9
check("array set", arr[0], 9)
arr.push(2)
check("array push", arr.size, 6)
check("array pop", arr.pop, 2)
check("array include", arr.include?(4), true)
check("array include miss", arr.include?(7), false)

asum = 0
[10, 20, 30].each do |v|
  asum += v
end
check("array each", asum, 60)

doubled = [1, 2, 3].map do |v|
  v * 2
end
check("array map", doubled[0] + doubled[2], 8)

odds = [1, 2, 3, 4, 5].select do |v|
  v % 2 == 1
end
check("array select", odds.size, 3)

# ----- blocks with brace syntax and closures -----
acc = 100
[1, 2, 3].each { |v| acc += v }
check("brace block closure", acc, 106)

tripled = [1, 2, 3].map { |v| v * 3 }
check("brace map", tripled[1], 6)

# ----- hashes -----
h = {"a" => 1, "b" => 2}
check("hash get", h["a"], 1)
h["c"] = 3
check("hash set", h["c"], 3)
check("hash size", h.size, 3)
check("hash keys", h.keys.size, 3)
check("hash values include", h.values.include?(2), true)

ksum = 0
{"x" => 10, "y" => 20}.keys.each do |k|
  ksum += 1
end
check("hash keys each", ksum, 2)

# ----- symbols (approximated as strings) -----
check("symbol", :hello, "hello")

# ----- classes: @ivars, methods, self, C.new -----
class Counter
  def initialize(start, step)
    @value = start
    @step = step
  end
  def increment
    @value += @step
    @value
  end
  def value
    @value
  end
  def reset
    @value = 0
    self
  end
end

ctr = Counter.new(10, 5)
check("class init", ctr.value(), 10)
check("class method", ctr.increment(), 15)
check("class method again", ctr.increment(), 20)
ctr.reset()
check("class reset", ctr.value(), 0)

class Point
  def initialize(x, y)
    @x = x
    @y = y
  end
  def manhattan
    ax = @x
    if ax < 0
      ax = -ax
    end
    ay = @y
    if ay < 0
      ay = -ay
    end
    ax + ay
  end
end
p = Point.new(3, -4)
check("class fields", p.manhattan(), 7)

# ----- done -----
if fails == 0
  puts "Ruby subset self test passed"
end
exit(fails)
