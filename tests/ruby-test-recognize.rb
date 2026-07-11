# A real-world-looking Ruby file that exercises constructs the metacompiler now
# RECOGNIZES (parses) but does not fully implement. Each such construct emits a
# "not implemented (ignored)" warning under -warn-unsupported and is skipped or
# approximated. A normal run aborts at the first one - so this file is a
# SHOULD-FAIL by default and only exits 0 under -warn-unsupported. The output
# is byte-identical under goja and -frozen for both the interpreter and the
# LLVM-IR compiler. The constructs covered: module, class superclass, singleton
# methods (def self.x), default / splat / double-splat / keyword / block
# parameters, keyword / splat / block call arguments, ||= conditional assignment,
# float literals, scoped constants (::), %w / %i percent literals, for-in loops,
# and yield. Symbol-key hashes { name: v } are genuinely lowered (no warning).

# ----- module with a constant and a singleton method -----
module Config
  DEFAULTS = { retries: 3, verbose: false }

  def self.fetch(key)
    key
  end
end

# ----- class inheritance and a singleton (class) method -----
class Animal
  def initialize(name)
    @name = name
  end

  def speak
    "..."
  end
end

class Dog < Animal
  def speak
    "woof"
  end

  def self.species
    "canis familiaris"
  end
end

# ----- def with default, splat, double-splat, keyword and block parameters -----
def build_url(host, port = 80, *segments, scheme: "http", **query, &callback)
  host
end

# ----- keyword, splat and block arguments at the call site -----
def render(name)
  name
end

parts = [1, 2, 3]
render(name: "widget")
render(*parts)
render(&:upcase)

# ----- symbol-key hash (genuinely lowered) and ||= conditional assignment -----
options = { color: "red", size: 4, tag: :main }
puts options[:color]

cache = nil
cache ||= "warm"
puts cache

# ----- float literal and scoped constant (recognized, not implemented) -----
ratio = 3.14
scope = Config::DEFAULTS
puts "computed"

# ----- percent-literal word and symbol arrays -----
names = %w[alice bob carol]
flags = %i[read write execute]
puts "listed"

# ----- for-in loop (recognized, not implemented) -----
for n in [10, 20, 30]
  puts n
end

# ----- yield inside a method (recognized, not implemented); defined only -----
def each_pair
  yield 1, 2
end

puts "done"
