# Ruby subset self test 2: constructs added on top of ruby-test-1.
# Exercises case/when/else, the ternary cond ? a : b, the << append operator,
# multiple assignment (including the swap), and attr_accessor / attr_reader /
# attr_writer. Genuinely implemented, so this file passes on a default run and is
# byte-identical under goja and -frozen for both the interpreter and the compiler.

fails = 0

def check(name, got, want)
  if got != want
    puts "FAIL #{name}: got #{got} want #{want}"
    fails = fails + 1
  end
end

# ----- case / when / else (subject form, chained ==) -----
def describe(n)
  case n
  when 0
    "zero"
  when 1, 2, 3
    "small"
  else
    "big"
  end
end
check("case zero", describe(0), "zero")
check("case small 1", describe(1), "small")
check("case small 3", describe(3), "small")
check("case big", describe(9), "big")

# case as an expression, with a string subject and 'then'
grade = "B"
label = case grade
        when "A" then "excellent"
        when "B" then "good"
        else "other"
        end
check("case expr", label, "good")

# subject-less case (each when is a truthiness test)
x = 7
kind = case
       when x < 0 then "neg"
       when x == 0 then "zero"
       else "pos"
       end
check("case no subject", kind, "pos")

# ----- ternary cond ? a : b -----
check("ternary true", (3 > 2 ? "yes" : "no"), "yes")
check("ternary false", (1 > 2 ? "yes" : "no"), "no")
y = 5
m = y % 2 == 0 ? "even" : "odd"
check("ternary assign", m, "odd")
# nested ternary
sgn = y < 0 ? -1 : (y == 0 ? 0 : 1)
check("ternary nested", sgn, 1)

# ----- << append -----
buf = []
buf << 1
buf << 2 << 3
check("append size", buf.size, 3)
check("append first", buf[0], 1)
check("append last", buf[2], 3)
words = ["a"]
words << "b"
check("append str elem", words[1], "b")

# ----- multiple assignment -----
a, b = 1, 2
check("multi a", a, 1)
check("multi b", b, 2)
a, b = b, a
check("swap a", a, 2)
check("swap b", b, 1)
p, q, r = 10, 20, 30
check("multi triple", p + q + r, 60)
# unpack a single array positionally
first, second = [100, 200]
check("unpack first", first, 100)
check("unpack second", second, 200)

# ----- attr_accessor / attr_reader / attr_writer -----
class Box
  attr_accessor :width, :height
  attr_reader :label
  def initialize(w, h)
    @width = w
    @height = h
    @label = "box"
  end
  def area
    @width * @height
  end
end
box = Box.new(3, 4)
check("attr read width", box.width, 3)
check("attr read height", box.height, 4)
check("attr reader label", box.label, "box")
box.width = 10
check("attr write width", box.width, 10)
check("attr area after set", box.area, 40)

class Temp
  attr_writer :celsius
  def initialize
    @celsius = 0
  end
  def fahrenheit
    @celsius * 9 / 5 + 32
  end
end
t = Temp.new
t.celsius = 100
check("attr_writer effect", t.fahrenheit, 212)

# ----- done -----
if fails == 0
  puts "Ruby subset self test 2 passed"
end
exit(fails)
