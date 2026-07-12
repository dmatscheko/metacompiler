# Fast feature-matrix test for the Ruby interpreter (ruby-interpreter.abnf) and the
# LLVM-IR compiler (ruby-to-llvm-ir.abnf). It replaces the four algorithm-themed
# ruby-test-big-* stress tests: instead of large loops (five sorting routines,
# Ackermann, sieves, Roman numerals) every implemented construct is exercised with
# the SMALLEST program that can prove it works - loops run 0, 1, 3 or 4 times,
# recursion stays below depth 6. Floats, class inheritance, modules-as-namespaces,
# def self.x, ||=, %w literals, for-in and yield are recognized but not implemented
# (see ruby-test-recognize.rb) and stay out. A failed check prints its id (so a
# diff pinpoints it) and the file ends with exit(fails); exit 0 and byte-identical
# output on all four legs (interpreter/compiler x goja/-frozen) mean everything
# passed.

fails = 0
checks = 0

def check(name, got, want)
  checks = checks + 1
  if got != want
    puts "FAIL #{name}: got #{got} want #{want}"
    fails = fails + 1
  end
end

# ----- numbers, arithmetic, precedence (integers only; / truncates toward zero) -----
check("arith-precedence", 1 + 2 * 3, 7)
check("arith-paren", (1 + 2) * 3, 9)
check("arith-unary-minus", -5 + 2, -3)
check("arith-int-div", 7 / 2, 3)
check("arith-int-div-neg", -7 / 2, -3)
check("arith-mod", 7 % 3, 1)
check("arith-mod-neg", -7 % 3, -1)
check("arith-chain", 20 - 5 - 3, 12)
ca = 10
ca += 5
ca -= 3
ca *= 2
ca /= 4
ca %= 4
check("arith-compound", ca, 2)

# ----- comparison -----
check("cmp-lt", 2 < 3, true)
check("cmp-le", 3 <= 3, true)
check("cmp-gt", 3 > 2, true)
check("cmp-ge", 3 >= 4, false)
check("cmp-eq", 3 == 3, true)
check("cmp-ne", 3 != 4, true)

# ----- strings, interpolation, symbols -----
sname = "world"
check("str-interp", "hello #{sname}!", "hello world!")
check("str-interp-expr", "sum=#{2 + 3}", "sum=5")
check("str-concat", "a" + "b", "ab")
check("str-plus-num", "n=" + 5, "n=5")
check("str-to-s", "n=" + 5.to_s, "n=5")
check("str-length", "hello".length, 5)
check("str-size", "hello".size, 5)
check("str-index", "abc"[1], "b")
check("str-single-quote", 'raw text', "raw text")
check("str-escape-tab", "a\tb".length, 3)
check("str-upcase", "aBc".upcase, "ABC")
check("str-downcase", "aBc".downcase, "abc")
check("str-include", "hello".include?("ell"), true)
check("str-include-miss", "hello".include?("z"), false)
check("str-structural-eq", "ab" == "a" + "b", true)
check("str-compare-lt", "apple" < "banana", true)
check("str-unicode-len", "héllo".length, 5)
check("symbol-is-string", :hello, "hello")

# ----- booleans, truthiness (only nil and false are falsy), short-circuit -----
check("and-value", 3 && 4, 4)
check("or-value", false || 7, 7)
check("or-nil", nil || 7, 7)
check("and-nil", nil && 5, nil)
check("or-zero-truthy", 0 || 9, 0)
check("not", !false, true)
check("word-and", (1 < 2 and 2 < 3), true)
check("word-or", (false or 5), 5)
check("word-not", (not false), true)

hits = 0
def bump
  hits = hits + 1
  true
end
sc1 = false && bump()
sc2 = true && bump()
sc3 = true || bump()
check("logic-short-circuit", hits, 1)
check("logic-and-false", sc1, false)
check("logic-and-true", sc2, true)
check("logic-or-skip", sc3, true)

check("ternary-true", (3 > 2 ? "yes" : "no"), "yes")
check("ternary-false", (1 > 2 ? "yes" : "no"), "no")
check("ternary-nested", (5 < 0 ? -1 : (5 == 0 ? 0 : 1)), 1)

# ----- if / elsif / else / unless, if as an expression -----
def classify(n)
  if n < 0
    "negative"
  elsif n == 0
    "zero"
  else
    "positive"
  end
end
check("if-neg", classify(-4), "negative")
check("if-zero", classify(0), "zero")
check("if-pos", classify(9), "positive")

g5 = 5
rif = if g5 > 3 then "big" else "small" end
check("if-expression", rif, "big")

ub = 0
unless g5 > 100
  ub = 1
end
check("unless-block", ub, 1)

# ----- while / until, zero and three iterations, post-condition loops -----
w0 = 0
while w0 > 0
  w0 -= 1
end
check("while-zero-iterations", w0, 0)

w3 = 0
while w3 < 3
  w3 += 1
end
check("while-three", w3, 3)

u0 = 5
until u0 >= 5
  u0 += 1
end
check("until-zero-iterations", u0, 5)

u3 = 0
until u3 >= 3
  u3 += 1
end
check("until-three", u3, 3)

runs = 0
begin
  runs += 1
end while runs < 3
check("do-while-repeats", runs, 3)

once = 0
begin
  once += 1
end while once > 9
check("do-while-once", once, 1)

duntil = 0
begin
  duntil += 1
end until duntil >= 3
check("do-until", duntil, 3)

# ----- break / next, nested loops, statement modifiers -----
bstr = ""
bi = 0
while bi < 9
  if bi == 2
    break
  end
  bstr = bstr + bi
  bi += 1
end
check("while-break", bstr, "01")

nstr = ""
ni = 0
while ni < 4
  ni += 1
  if ni % 2 == 0
    next
  end
  nstr = nstr + ni
end
check("while-next", nstr, "13")

nres = ""
oi = 0
while oi < 2
  ii = 0
  while ii < 3
    if ii == 1
      break
    end
    nres = nres + oi + ii
    ii += 1
  end
  oi += 1
end
check("nested-inner-break", nres, "0010")

m1 = 0
m1 = 5 if true
check("if-modifier-taken", m1, 5)
m2 = 0
m2 = 5 if false
check("if-modifier-skipped", m2, 0)
m3 = 0
m3 = 7 unless false
check("unless-modifier", m3, 7)
wcount = 0
wcount += 1 while wcount < 3
check("while-modifier", wcount, 3)
wzero = 0
wzero += 1 while wzero > 9
check("while-modifier-zero", wzero, 0)
dcount = 0
dcount += 1 until dcount >= 3
check("until-modifier", dcount, 3)

# ----- case / when: scalars, several values, ranges, expression, no subject -----
def bucket(n)
  case n
  when 0 then "zero"
  when 1, 2, 3 then "small"
  when 4..6 then "mid"
  else "big"
  end
end
check("case-scalar", bucket(0), "zero")
check("case-multi-value", bucket(2), "small")
check("case-range", bucket(5), "mid")
check("case-range-inclusive-end", bucket(6), "mid")
check("case-else", bucket(9), "big")

def band(v)
  case v
  when 0...4 then "lo"
  else "hi"
  end
end
check("case-range-exclusive-in", band(3), "lo")
check("case-range-exclusive-out", band(4), "hi")

cg = "B"
clabel = case cg
         when "A" then "first"
         when "B" then "second"
         else "other"
         end
check("case-expression", clabel, "second")

c9 = 7
ckind = case
        when c9 < 0 then "neg"
        when c9 == 0 then "zero"
        else "pos"
        end
check("case-no-subject", ckind, "pos")

# ----- methods: implicit/explicit return, recursion, first-class functions -----
def add(ax, bx)
  ax + bx
end
check("fn-implicit-return", add(20, 22), 42)

def sign(n)
  if n < 0
    return -1
  end
  1
end
check("fn-early-return", sign(-8), -1)
check("fn-fallthrough", sign(3), 1)

def fib(n)
  if n < 2
    return n
  end
  fib(n - 1) + fib(n - 2)
end
check("fn-recursion", fib(6), 8)

def even_p(n)
  if n == 0
    return true
  end
  odd_p(n - 1)
end
def odd_p(n)
  if n == 0
    return false
  end
  even_p(n - 1)
end
check("fn-mutual-recursion", even_p(4) && odd_p(5), true)

def pick(digs, cx)
  di = 0
  while di < digs.size
    return di if digs[di] == cx
    di += 1
  end
  return -1
end
check("return-if-modifier", pick(["a", "b", "c"], "b"), 1)
check("return-if-modifier-miss", pick(["a"], "z"), -1)

def double(nx)
  nx * 2
end
def negate(nx)
  0 - nx
end
def apply_fn(fx, nx)
  fx(nx)
end
check("fn-first-class", apply_fn(double, 21), 42)
funcs = [double, negate]
fsel = funcs[1]
check("fn-in-array", fsel(3), -3)

# ----- ranges and blocks (do..end and { }, real closures) -----
rsum = 0
(1..3).each do |k|
  rsum += k
end
check("range-each-inclusive", rsum, 6)

xsum = 0
(1...3).each { |k| xsum += k }
check("range-each-exclusive", xsum, 3)

esum = 0
[10, 20, 30].each do |v|
  esum += v
end
check("block-each-do", esum, 60)

bacc = 100
[1, 2, 3].each { |v| bacc += v }
check("block-brace-closure", bacc, 106)

mapped = [1, 2, 3].map { |v| v * 3 }
check("block-map", mapped[1], 6)
check("block-select", [1, 2, 3, 4, 5].select { |v| v % 2 == 1 }.size, 3)
check("block-reject-sum", [1, 2, 3, 4].reject { |v| v % 2 == 0 }.sum, 4)
check("block-chain", [1, 2, 3, 4].map { |v| v * 2 }.select { |v| v > 4 }.sum, 14)

ewi = 0
[10, 20, 30].each_with_index { |v, ix| ewi += v * ix }
check("block-each-with-index", ewi, 80)

check("truthy-zero-in-select", [0, 1, 2].select { |v| v }.size, 3)
check("falsy-nil-false-dropped", [0, nil, 5, false, 7].select { |v| v }.sum, 12)

bnest = 0
[1, 2].each do |av|
  [10, 20].each { |bv| bnest += av * bv }
end
check("block-nested", bnest, 90)

# ----- arrays -----
arr = [3, 1, 4]
check("arr-index", arr[2], 4)
check("arr-size", arr.size, 3)
check("arr-length", arr.length, 3)
arr[0] = 9
check("arr-set", arr[0], 9)
arr.push(2)
check("arr-push", arr.size, 4)
check("arr-pop", arr.pop, 2)
check("arr-include", arr.include?(4), true)
check("arr-include-miss", arr.include?(7), false)
check("arr-neg-index", [10, 20, 30][-1], 30)

buf = []
buf << 1
buf << 2 << 3
check("arr-append-chain", buf.size, 3)
check("arr-append-last", buf[2], 3)
sel = []
sel << 5 if true
sel << 6 if false
check("arr-append-if-modifier", sel.size, 1)
check("arr-first-last", [10, 20, 30].first + [10, 20, 30].last, 40)
check("arr-empty-first", [].first, nil)
check("arr-empty-pop", [].pop, nil)
check("arr-to-a", [1, 2, 3].to_a.size, 3)
check("arr-nested", [[1, 2], [3]][0][1], 2)

# ----- hashes (string keys, symbol keys, iteration order) -----
h = {"a" => 1, "b" => 2}
check("hash-get", h["a"], 1)
h["c"] = 3
check("hash-set", h["c"], 3)
check("hash-size", h.size, 3)
check("hash-length", h.length, 3)
check("hash-keys-size", h.keys.size, 3)
check("hash-values-sum", h.values.sum, 6)
check("hash-include", h.include?("a"), true)
check("hash-has-key", h.has_key?("b"), true)
check("hash-key-missing", h.key?("z"), false)

horder = ""
h.each { |hk, hv| horder = horder + hk }
check("hash-each-order", horder, "abc")
hvals = 0
h.each { |hk, hv| hvals += hv }
check("hash-each-values", hvals, 6)

opts = { color: "red", size: 4 }
check("hash-symbol-key", opts[:color], "red")
check("hash-symbol-as-string", opts["size"], 4)

# ----- multiple assignment -----
ma, mb = 1, 2
check("multi-a", ma, 1)
check("multi-b", mb, 2)
ma, mb = mb, ma
check("multi-swap-a", ma, 2)
check("multi-swap-b", mb, 1)
mp, mq, mr = 10, 20, 30
check("multi-triple", mp + mq + mr, 60)
mu1, mu2 = [100, 200]
check("multi-unpack-a", mu1, 100)
check("multi-unpack-b", mu2, 200)

# ----- classes: @ivars, methods, self, C.new, attr_*, blocks as parameters -----
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

c1 = Counter.new(10, 5)
c2 = Counter.new(0, 1)
check("class-init", c1.value(), 10)
check("class-method", c1.increment(), 15)
c2.increment()
c2.increment()
check("class-instances-independent", c1.value() + c2.value(), 17)
rself = c1.reset()
check("class-self-return", rself.value(), 0)

class Box
  attr_accessor :width
  attr_reader :label
  attr_writer :secret
  def initialize(wv)
    @width = wv
    @label = "box"
    @secret = 0
  end
  def area
    @width * @width
  end
  def reveal
    @secret
  end
end
box = Box.new(3)
check("attr-read", box.width, 3)
box.width = 10
check("attr-write", box.width, 10)
check("attr-affects-method", box.area(), 100)
check("attr-reader", box.label, "box")
box.secret = 42
check("attr-writer", box.reveal(), 42)

class Coll
  def initialize(items)
    @items = items
  end
  def my_map(fx)
    mout = []
    @items.each { |x| mout << fx(x) }
    mout
  end
  def my_reduce(init, fx)
    macc = init
    @items.each { |x| macc = fx(macc, x) }
    macc
  end
end
coll = Coll.new([1, 2, 3])
doubled = coll.my_map { |x| x * 2 }
check("class-block-param", doubled[2], 6)
reduced = coll.my_reduce(10) { |acc, x| acc + x }
check("class-block-two-params", reduced, 16)

# ----- safe navigation &. -----
gone = nil
here = "yellow"
check("safe-nav-present", "hello"&.upcase, "HELLO")
check("safe-nav-nil", gone&.upcase, nil)
check("safe-nav-chain-nil", gone&.upcase&.length, nil)
check("safe-nav-chain-present", here&.upcase&.length, 6)

pokes = 0
def poke
  pokes = pokes + 1
  "ell"
end
sn1 = gone&.include?(poke())
check("safe-nav-args-skipped", pokes, 0)
check("safe-nav-nil-result", sn1, nil)
sn2 = here&.include?(poke())
check("safe-nav-args-evaluated", pokes, 1)
check("safe-nav-present-result", sn2, true)

# ----- exceptions: begin / rescue / else / ensure, raise, control flow -----
def risky(n)
  if n > 3
    raise n
  end
  n * 2
end

def raise_info
  raise({"code" => 42})
end

exlog = ""
begin
  exlog = exlog + "a"
  raise "boom"
  exlog = exlog + "X"
rescue => e
  exlog = exlog + "b" + e
ensure
  exlog = exlog + "c"
end
check("begin-rescue-ensure-order", exlog, "abboomc")

caught = -1
begin
  risky(5)
  check("unreachable-after-raise", true, false)
rescue => e
  caught = e
end
check("raise-unwinds-calls", caught, 5)
check("raise-untaken-path", risky(2), 4)

info = 0
begin
  raise_info()
rescue => e
  info = e["code"]
end
check("raise-hash-value", info, 42)

def safe_div(av, bv)
  begin
    av / bv
  rescue => e
    -1
  end
end
check("begin-value-flows", safe_div(20, 4), 5)

def with_else(n)
  tagv = 0
  begin
    if n < 0
      raise n
    end
    tagv = 1
  rescue => e
    tagv = 2
  else
    tagv = tagv + 10
  end
  tagv
end
check("rescue-else-runs", with_else(7), 11)
check("rescue-else-skipped", with_else(-1), 2)

def ret_from_begin(n)
  begin
    if n > 0
      return n * 10
    end
    raise 0
  rescue => e
    return -1
  ensure
    # ensure always runs; no control flow on this path
  end
end
check("return-from-begin", ret_from_begin(4), 40)
check("return-from-rescue", ret_from_begin(0), -1)

def nested_return
  begin
    begin
      return 9
    ensure
    end
  ensure
  end
  0
end
check("nested-return", nested_return(), 9)

marks = 0
def note_mark
  marks = marks + 1
end
def ret_across_ensure
  begin
    return "from-begin"
  ensure
    note_mark()
  end
end
check("return-across-ensure", ret_across_ensure(), "from-begin")
check("ensure-ran-on-return", marks, 1)

def ret_in_ensure_overrides
  begin
    return "from-begin"
  ensure
    return "from-ensure"
  end
end
check("return-in-ensure-overrides", ret_in_ensure_overrides(), "from-ensure")

def ensure_return_cancels_raise
  begin
    begin
      raise "boom"
    ensure
      return "cancelled"
    end
  rescue
    return "rescued"
  end
end
check("ensure-return-cancels-raise", ensure_return_cancels_raise(), "cancelled")

mi = 0
while true
  mi = mi + 1
  break if mi >= 3
end
check("break-if-modifier-in-while", mi, 3)

mj = 0
mk = 0
while mj < 5
  mj = mj + 1
  next if mj == 2
  mk = mk + 1
end
check("next-if-modifier-in-while", mk, 4)

def ret_if_in_while(n)
  while true
    n = n + 1
    return "hit" if n >= 3
  end
  "unreached"
end
check("return-if-modifier-in-while", ret_if_in_while(0), "hit")

def loop_break_in_begin
  btotal = 0
  bx = 0
  while bx < 9
    begin
      break if bx == 2
      btotal = btotal + bx
    ensure
    end
    bx += 1
  end
  btotal
end
check("break-leaves-begin", loop_break_in_begin(), 1)

def loop_next_in_begin
  ntotal = 0
  nx = -1
  while nx < 3
    nx += 1
    begin
      next if nx == 1
      ntotal = ntotal + nx
    rescue => e
    end
  end
  ntotal
end
check("next-leaves-begin", loop_next_in_begin(), 5)

# a return out of an ensure replaces the begin body's return value
def ret_in_ensure
  begin
    return 1
  ensure
    return 2
  end
end
check("return-in-ensure-overrides", ret_in_ensure(), 2)

def break_in_ensure
  ix = 0
  while true
    begin
      ix = ix + 1
    ensure
      break
    end
  end
  ix
end
check("break-in-ensure", break_in_ensure(), 1)

def rethrow
  begin
    begin
      raise "deep"
    rescue => e
      raise e + "er"
    end
  rescue => e2
    return e2
  end
end
check("rethrow", rethrow(), "deeper")

# ----- everything combined in one small pipeline (3-element data flow) -----
def transform(items)
  tout = []
  items.each do |n|
    begin
      if n < 0
        raise "neg"
      end
      tout << (n % 2 == 0 ? "e#{n}" : "o#{n}")
    rescue => e
      tout << "x"
    end
  end
  tout
end
tres = transform([1, 2, -3])
check("combined-pipeline", tres[0] + tres[1] + tres[2], "o1e2x")

puts "features: #{checks} checks, #{fails} failures"
exit(fails)
