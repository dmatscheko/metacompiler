# A project library required by tests/ruby-test-multifile.rb (via -i tests/imports).
# Ruby's require is an ordinary method call; the grammar intercepts require /
# require_relative and loads this file at parse time. Its top-level class and def
# register globally (flat), so the requiring file can use Vec and gcd directly. The
# same file is loaded by the interpreter and the LLVM-IR compiler, which must agree.

class Vec
  def initialize(x, y)
    @x = x
    @y = y
  end
  def x
    @x
  end
  def y
    @y
  end
  def dot(o)
    @x * o.x + @y * o.y
  end
  def scale(f)
    Vec.new(@x * f, @y * f)
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
