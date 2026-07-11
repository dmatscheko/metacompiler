# Ruby subset - big self test 3: OBJECTS, POLYMORPHISM AND FUNCTIONAL STYLE.
# Classes with @ivars / attr_accessor / attr_reader, fluent self-returning methods,
# value objects (Vector2D, Fraction), a Matrix with multiply / transpose, duck-typed
# polymorphism over an array of shapes, a BankAccount with an overdraft guard and a
# transaction log, higher-order methods that take blocks (my_map / my_select /
# my_reduce), and first-class functions passed by name and stored in an array.
# Counts failures and ends with exit(fails); the interpreter and the LLVM-IR
# compiler run the same program and must agree.

fails = 0

def check(name, got, want)
  if got != want
    puts "FAIL #{name}: got #{got} want #{want}"
    fails = fails + 1
  end
end

def gcd(a, b)
  while b != 0
    t = b
    b = a % b
    a = t
  end
  a
end

def arr_eq(a, b)
  na = a.size
  return false if na != b.size
  i = 0
  while i < na
    return false if a[i] != b[i]
    i += 1
  end
  true
end

# ----- Vector2D: an immutable value object with arithmetic -----

class Vector2D
  attr_reader :x, :y
  def initialize(x, y)
    @x = x
    @y = y
  end
  def add(other)
    Vector2D.new(@x + other.x, @y + other.y)
  end
  def subtract(other)
    Vector2D.new(@x - other.x, @y - other.y)
  end
  def scale(k)
    Vector2D.new(@x * k, @y * k)
  end
  def dot(other)
    @x * other.x + @y * other.y
  end
  def mag_squared
    @x * @x + @y * @y
  end
  def manhattan
    ax = @x
    ax = 0 - ax if ax < 0
    ay = @y
    ay = 0 - ay if ay < 0
    ax + ay
  end
  def equals(other)
    @x == other.x && @y == other.y
  end
end

a = Vector2D.new(3, 4)
b = Vector2D.new(1, 2)
check("vec add x", a.add(b).x, 4)
check("vec add y", a.add(b).y, 6)
check("vec sub x", a.subtract(b).x, 2)
check("vec scale", a.scale(3).x, 9)
check("vec dot", a.dot(b), 11)
check("vec mag_squared", a.mag_squared, 25)
check("vec manhattan", Vector2D.new(-3, 4).manhattan, 7)
check("vec equals yes", a.equals(Vector2D.new(3, 4)), true)
check("vec equals no", a.equals(b), false)
# chaining several transformations
c = a.add(b).scale(2).subtract(Vector2D.new(1, 1))
check("vec chain x", c.x, 7)
check("vec chain y", c.y, 11)

# ----- Fraction: reduced on construction via gcd -----

class Fraction
  attr_reader :num, :den
  def initialize(n, d)
    g = gcd(n, d)
    g = 1 if g == 0
    @num = n / g
    @den = d / g
  end
  def add(other)
    Fraction.new(@num * other.den + other.num * @den, @den * other.den)
  end
  def multiply(other)
    Fraction.new(@num * other.num, @den * other.den)
  end
  def equals(other)
    @num == other.num && @den == other.den
  end
  def render
    "#{@num}/#{@den}"
  end
end

half = Fraction.new(1, 2)
third = Fraction.new(1, 3)
check("frac reduce num", Fraction.new(4, 8).num, 1)
check("frac reduce den", Fraction.new(4, 8).den, 2)
check("frac add render", half.add(third).render, "5/6")
check("frac mul render", half.multiply(third).render, "1/6")
check("frac add reduces", Fraction.new(1, 6).add(Fraction.new(1, 6)).render, "1/3")
check("frac equals", half.equals(Fraction.new(2, 4)), true)

# ----- Matrix: multiply and transpose -----

class Matrix
  def initialize(rows)
    @rows = rows
  end
  def get(r, c)
    @rows[r][c]
  end
  def row_count
    @rows.size
  end
  def col_count
    @rows[0].size
  end
  def multiply(other)
    result = []
    i = 0
    while i < @rows.size
      line = []
      j = 0
      while j < other.col_count
        sum = 0
        k = 0
        while k < @rows[0].size
          sum += @rows[i][k] * other.get(k, j)
          k += 1
        end
        line << sum
        j += 1
      end
      result << line
      i += 1
    end
    Matrix.new(result)
  end
  def transpose
    result = []
    j = 0
    while j < @rows[0].size
      line = []
      i = 0
      while i < @rows.size
        line << @rows[i][j]
        i += 1
      end
      result << line
      j += 1
    end
    Matrix.new(result)
  end
end

m1 = Matrix.new([[1, 2], [3, 4]])
m2 = Matrix.new([[5, 6], [7, 8]])
prod = m1.multiply(m2)
check("matrix mul 00", prod.get(0, 0), 19)
check("matrix mul 01", prod.get(0, 1), 22)
check("matrix mul 10", prod.get(1, 0), 43)
check("matrix mul 11", prod.get(1, 1), 50)
tr = m1.transpose
check("matrix transpose 01", tr.get(0, 1), 3)
check("matrix transpose 10", tr.get(1, 0), 2)
check("matrix dims", m1.row_count, 2)
# a 2x3 times 3x2 gives a 2x2
r1 = Matrix.new([[1, 2, 3], [4, 5, 6]])
r2 = Matrix.new([[7, 8], [9, 10], [11, 12]])
rp = r1.multiply(r2)
check("matrix rect 00", rp.get(0, 0), 58)
check("matrix rect 11", rp.get(1, 1), 154)

# ----- polymorphism: an array of different shapes, one interface -----

class Rectangle
  def initialize(w, h)
    @w = w
    @h = h
  end
  def area
    @w * @h
  end
  def perimeter
    2 * (@w + @h)
  end
  def shape_name
    "rectangle"
  end
end

class Triangle
  def initialize(base, height)
    @base = base
    @height = height
  end
  def area
    @base * @height / 2
  end
  def perimeter
    3 * @base
  end
  def shape_name
    "triangle"
  end
end

class Circle
  def initialize(r)
    @r = r
  end
  def area
    3 * @r * @r
  end
  def perimeter
    6 * @r
  end
  def shape_name
    "circle"
  end
end

shapes = [Rectangle.new(3, 4), Triangle.new(6, 4), Circle.new(2)]
total_area = 0
shapes.each { |s| total_area += s.area }
check("poly total area", total_area, 36)   # 12 + 12 + 12
names = shapes.map { |s| s.shape_name }
check("poly names", names[0], "rectangle")
check("poly names last", names[2], "circle")
big = shapes.select { |s| s.area >= 12 }
check("poly select", big.size, 3)
# find the shape with the largest perimeter (rect 14, triangle 18, circle 12)
widest = shapes[0]
shapes.each do |s|
  widest = s if s.perimeter > widest.perimeter
end
check("poly widest", widest.shape_name, "triangle")

# ----- BankAccount: state, overdraft guard, transaction log -----

class BankAccount
  attr_reader :owner, :balance
  def initialize(owner, opening)
    @owner = owner
    @balance = opening
    @history = []
  end
  def deposit(amount)
    @balance += amount
    @history.push(amount)
    self
  end
  def withdraw(amount)
    if amount > @balance
      false
    else
      @balance -= amount
      @history.push(0 - amount)
      true
    end
  end
  def transaction_count
    @history.size
  end
  def net_change
    @history.sum
  end
end

acct = BankAccount.new("Ada", 100)
acct.deposit(50).deposit(25)
check("bank balance after deposits", acct.balance, 175)
check("bank withdraw ok", acct.withdraw(75), true)
check("bank balance after withdraw", acct.balance, 100)
check("bank overdraft blocked", acct.withdraw(1000), false)
check("bank balance unchanged", acct.balance, 100)
check("bank tx count", acct.transaction_count, 3)
check("bank net change", acct.net_change, 0)   # +50 +25 -75
check("bank owner", acct.owner, "Ada")

# ----- higher-order methods that take a block -----

class Collection
  def initialize(items)
    @items = items
  end
  def my_map(f)
    out = []
    @items.each { |x| out << f(x) }
    out
  end
  def my_select(pred)
    out = []
    @items.each { |x| out << x if pred(x) }
    out
  end
  def my_reduce(init, f)
    acc = init
    @items.each { |x| acc = f(acc, x) }
    acc
  end
  def count
    @items.size
  end
end

coll = Collection.new([1, 2, 3, 4, 5])
squares = coll.my_map { |x| x * x }
check("hof map", arr_eq(squares, [1, 4, 9, 16, 25]), true)
evens = coll.my_select { |x| x % 2 == 0 }
check("hof select", arr_eq(evens, [2, 4]), true)
total = coll.my_reduce(0) { |acc, x| acc + x }
check("hof reduce sum", total, 15)
product = coll.my_reduce(1) { |acc, x| acc * x }
check("hof reduce product", product, 120)
check("hof count", coll.count, 5)
# compose: map then reduce
sum_of_squares = Collection.new(coll.my_map { |x| x * x }).my_reduce(0) { |acc, x| acc + x }
check("hof compose", sum_of_squares, 55)

# ----- first-class functions: passed by name and stored in an array -----

def double(x)
  x * 2
end

def square(x)
  x * x
end

def negate(x)
  0 - x
end

def apply_fn(f, x)
  f(x)
end

check("fn double", apply_fn(double, 21), 42)
check("fn square", apply_fn(square, 9), 81)
check("fn negate", apply_fn(negate, 5), -5)

funcs = [double, square, negate]
results = []
i = 0
while i < funcs.size
  f = funcs[i]
  results << f(3)
  i += 1
end
check("fn array double", results[0], 6)
check("fn array square", results[1], 9)
check("fn array negate", results[2], -3)

# apply a pipeline of functions to a seed value
def pipe(fns, seed)
  value = seed
  fns.each do |fn|
    g = fn
    value = g(value)
  end
  value
end
check("fn pipeline", pipe([double, double, square], 2), 64)  # 2->4->8->64

# ----- done -----
if fails == 0
  puts "Ruby big self test 3 (objects and functional) passed"
end
exit(fails)
